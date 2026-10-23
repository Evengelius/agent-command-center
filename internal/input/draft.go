package input

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
)

// Drafts keeps each dashboard's draft on disk, so a crash, a panic or a
// closed window loses no draft (proposal step 4, item 4; ASM-23). Each
// dashboard's draft has a folder of its own in the user's own directory
// (under %LOCALAPPDATA% on Windows), named by the SHA-256 of the project's
// path and a random key (plan 33, design D1): its text byte for byte, its
// images list, the pasted images' files, and a lock the dashboard holds
// while the draft is its own. A draft let go waits for its project's next
// start; it is deleted only once its message was accepted or its text and
// images removed.
type Drafts struct {
	Dir string // "" is DraftDir()
}

// The files of a draft's folder (plan 33, design D1).
const (
	lockName   = "lock"
	textName   = "draft.txt"
	listName   = "images.json"
	asideName  = listName + ".unreadable" // a damaged list, set aside (design D6)
	tempPrefix = ".draft-"
	pastedExt  = ".png"
	keyDigits  = 16 // the random key's hexadecimal digits
)

// newborn is how long the sweep leaves a folder without its lock, which is
// being made, and a temporary file beside the folders, which a version
// before plan 33 may still be writing (design D9).
const newborn = time.Minute

// pastedPrefix starts a pasted image's entry in the images list, where a
// file has its path: no file is named so (plan 32, design D3).
const pastedPrefix = "pasted:"

// DraftDir is where drafts live: agent-command-center\drafts in the user's
// cache directory, %LOCALAPPDATA% on Windows.
func DraftDir() (string, error) {
	d, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "agent-command-center", "drafts"), nil
}

func (d Drafts) dir() (string, error) {
	if d.Dir != "" {
		return d.Dir, nil
	}
	return DraftDir()
}

// DraftKey is what the names of a project's drafts start with: the SHA-256
// of its cleaned path, lower-cased on Windows where paths ignore case. The
// project's path itself is written nowhere.
func DraftKey(project string) string {
	p := filepath.Clean(project)
	if runtime.GOOS == "windows" {
		p = strings.ToLower(p)
	}
	sum := sha256.Sum256([]byte(p))
	return hex.EncodeToString(sum[:])
}

// PastedEntry is a pasted image kept on disk as the images list names it:
// "pasted:", its name, "#", the SHA-256 of its bytes in hexadecimal; its
// file is that SHA-256 with ".png" in the draft's folder (plan 33, design
// D8).
func PastedEntry(name string, sum [32]byte) string {
	return pastedPrefix + name + "#" + hex.EncodeToString(sum[:])
}

// ParsePasted reads an entry of the images list: a pasted image's name and
// SHA-256 in hexadecimal, the SHA-256 "" for an entry of plan 32, whose
// bytes were never kept; ok is false for a file's path.
func ParsePasted(entry string) (name, sum string, ok bool) {
	rest, found := strings.CutPrefix(entry, pastedPrefix)
	if !found {
		return "", "", false
	}
	if i := strings.LastIndexByte(rest, '#'); i >= 0 && isHex(rest[i+1:], 64) {
		return rest[:i], rest[i+1:], true
	}
	return rest, "", true
}

func isHex(s string, n int) bool {
	if len(s) != n {
		return false
	}
	for _, c := range s {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}

// isSlot says a name is a draft's folder: a project's key, "-", a random
// key.
func isSlot(name string) bool {
	return len(name) == 64+1+keyDigits && name[64] == '-' && isHex(name[:64], 64) && isHex(name[65:], keyDigits)
}

// StoredDraft is a stored draft as a dashboard takes it.
type StoredDraft struct {
	Text     string
	Images   []string // files' paths and pasted images' entries, in order
	Others   int      // the project's other stored drafts, left waiting
	SetAside bool     // the images list could not be read and was set aside (design D6)
}

// Slot is a dashboard's own draft: a folder whose lock it holds open with
// no sharing, so that no other dashboard takes, sweeps or changes it, and
// Windows lets it go when the process ends, killed or not (plan 33, design
// D1). It is not safe for concurrent use: the dashboard orders its calls.
type Slot struct {
	root string // the drafts folder
	key  string // the project's DraftKey
	cli  string // the CLI's id, written in the lock
	dir  string // the draft's folder; "" until its first save makes it
	lock *os.File
	gone bool // let go: nothing more is written
}

// candidate is a stored draft a dashboard may take: a folder nobody holds,
// or a draft from before plan 33 (dir "").
type candidate struct {
	dir string
	cli string
	at  time.Time // its last save
}

// Take sweeps the drafts folder (design D9), then takes a stored draft of
// project for cli: one of cli first, then the newest (design D1). With
// none, the slot is empty and makes its folder at its first save. A text
// that cannot be read is an error, the draft left where it is.
func (d Drafts) Take(project, cli string) (*Slot, StoredDraft, error) {
	root, err := d.dir()
	if err != nil {
		return nil, StoredDraft{}, err
	}
	s := &Slot{root: root, key: DraftKey(project), cli: cli}
	es, err := os.ReadDir(root)
	if errors.Is(err, fs.ErrNotExist) {
		return s, StoredDraft{}, nil
	}
	if err != nil {
		return nil, StoredDraft{}, err
	}
	sweep(root, es)
	cs, err := candidates(root, s.key, cli)
	if err != nil {
		return nil, StoredDraft{}, err
	}
	// One budget for the whole start: a lock another program has open a
	// moment is waited for, together, about a second at most.
	deadline := time.Now().Add(time.Second)
	busy := 0 // in use by another program even after the wait
	for i, c := range cs {
		switch s.adopt(c, deadline) {
		case adoptBusy:
			busy++
			continue
		case adoptGone:
			continue // another dashboard took it meanwhile, or it went
		}
		st, err := s.read()
		if err != nil {
			_ = s.Release()
			return nil, StoredDraft{}, err
		}
		st.Others = busy + len(cs) - i - 1
		return s, st, nil
	}
	return s, StoredDraft{Others: busy}, nil
}

// adoptResult is what taking a candidate came to.
type adoptResult int

const (
	adopted   adoptResult = iota
	adoptGone             // taken by another dashboard, emptied or deleted
	adoptBusy             // its lock open in another program past the wait
)

// candidates are the project's stored drafts, those of cli first, then the
// newest first.
func candidates(root, key, cli string) ([]candidate, error) {
	es, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	var out []candidate
	for _, e := range es {
		name := e.Name()
		if !e.IsDir() || !isSlot(name) || !strings.HasPrefix(name, key) {
			continue
		}
		dir := filepath.Join(root, name)
		b, err := os.ReadFile(filepath.Join(dir, lockName))
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			// Held: another dashboard's own draft; or, for a moment,
			// another start sweeping it: looked at once more.
			time.Sleep(20 * time.Millisecond)
			b, err = os.ReadFile(filepath.Join(dir, lockName))
		}
		switch {
		case errors.Is(err, fs.ErrNotExist):
			b = nil // made by a dashboard that ended before its lock: no CLI
		case err != nil:
			continue // held: another dashboard's own draft
		}
		at, ok := lastSave(dir)
		if !ok {
			continue // no draft in it: the sweep's
		}
		out = append(out, candidate{dir: dir, cli: string(b), at: at})
	}
	// A draft from before plan 33, beside the folders.
	if at, ok := lastSave(root, key+".txt", key+".images.json"); ok {
		out = append(out, candidate{at: at})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if mi, mj := out[i].cli == cli, out[j].cli == cli; mi != mj {
			return mi
		}
		return out[i].at.After(out[j].at)
	})
	return out, nil
}

// lastSave is the latest change of a draft's text and list in dir, false
// when it has neither.
func lastSave(dir string, names ...string) (time.Time, bool) {
	if len(names) == 0 {
		names = []string{textName, listName}
	}
	var at time.Time
	found := false
	for _, n := range names {
		if st, err := os.Stat(filepath.Join(dir, n)); err == nil {
			found = true
			if st.ModTime().After(at) {
				at = st.ModTime()
			}
		}
	}
	return at, found
}

// adopt takes a candidate: its lock opened with no sharing, the CLI's id
// written in it; a draft from before plan 33 is moved into a new folder.
// A lock that refuses the unshared open but can still be read is open in
// another program a moment (an antivirus, an indexer, another start
// looking at it): tried again until deadline. One that cannot be read is
// another dashboard's, which took it meanwhile: left at once.
func (s *Slot) adopt(c candidate, deadline time.Time) adoptResult {
	if c.dir == "" {
		if s.adoptOld() {
			return adopted
		}
		return adoptGone
	}
	p := filepath.Join(c.dir, lockName)
	var f *os.File
	for wait := time.Millisecond; ; wait = min(2*wait, 128*time.Millisecond) {
		var err error
		f, err = openExclusive(p, false)
		if errors.Is(err, fs.ErrNotExist) {
			f, err = openExclusive(p, true)
		}
		if err == nil {
			break
		}
		if !transient(err) {
			return adoptGone
		}
		if _, rerr := os.ReadFile(p); rerr != nil {
			return adoptGone // a dashboard holds it, unshared
		}
		if time.Now().After(deadline) {
			return adoptBusy
		}
		time.Sleep(wait)
	}
	if _, ok := lastSave(c.dir); !ok {
		f.Close() // emptied before it was taken
		return adoptGone
	}
	if err := writeLock(f, s.cli); err != nil {
		f.Close()
		return adoptGone
	}
	s.dir, s.lock = c.dir, f
	return adopted
}

// adoptOld moves a draft from before plan 33 into a new folder of the
// slot's: its text and its list, each by one rename.
func (s *Slot) adoptOld() bool {
	if err := s.make(); err != nil {
		return false
	}
	moved := false
	for from, to := range map[string]string{s.key + ".txt": textName, s.key + ".images.json": listName} {
		if os.Rename(filepath.Join(s.root, from), filepath.Join(s.dir, to)) == nil {
			moved = true
		}
	}
	if !moved {
		s.drop() // another dashboard took it meanwhile
		return false
	}
	return true
}

// read reads the slot's draft. A list that cannot be parsed is renamed
// aside and the text restored without it (design D6).
func (s *Slot) read() (StoredDraft, error) {
	var st StoredDraft
	b, err := os.ReadFile(filepath.Join(s.dir, textName))
	switch {
	case err == nil:
		st.Text = string(b)
	case !errors.Is(err, fs.ErrNotExist):
		return st, err
	}
	b, err = os.ReadFile(filepath.Join(s.dir, listName))
	switch {
	case errors.Is(err, fs.ErrNotExist):
	case err != nil:
		return st, errors.New("the draft's images list is unreadable: " + err.Error())
	case json.Unmarshal(b, &st.Images) != nil:
		st.Images = nil
		if err := os.Rename(filepath.Join(s.dir, listName), filepath.Join(s.dir, asideName)); err != nil {
			return st, errors.New("the draft's images list is unreadable and cannot be set aside: " + err.Error())
		}
		st.SetAside = true
	}
	return st, nil
}

// make makes the slot's folder and its lock, held from then on.
func (s *Slot) make() error {
	if err := os.MkdirAll(s.root, 0o700); err != nil {
		return err
	}
	var key [keyDigits / 2]byte
	if _, err := rand.Read(key[:]); err != nil {
		return err
	}
	dir := filepath.Join(s.root, s.key+"-"+hex.EncodeToString(key[:]))
	if err := os.Mkdir(dir, 0o700); err != nil {
		return err
	}
	f, err := openExclusive(filepath.Join(dir, lockName), true)
	if err == nil {
		err = writeLock(f, s.cli)
		if err != nil {
			f.Close()
		}
	}
	if err != nil {
		_ = os.RemoveAll(dir)
		return err
	}
	s.dir, s.lock = dir, f
	return nil
}

// drop closes the lock and deletes the slot's folder.
func (s *Slot) drop() {
	if s.lock != nil {
		s.lock.Close()
	}
	_ = os.RemoveAll(s.dir)
	s.dir, s.lock = "", nil
}

func writeLock(f *os.File, cli string) error {
	if err := f.Truncate(0); err != nil {
		return err
	}
	_, err := f.WriteAt([]byte(cli), 0)
	return err
}

// Save writes the draft (design D5): the pasted images' files first, then
// the images list, then the text, each through a temporary file renamed
// over the old one, so a crash mid-write leaves the previous one; an empty
// text removes the text's file only, an empty list the list's only; a
// pasted image's file the list no longer names goes last, when it can.
// pasted holds the bytes of the pasted images the list names and that are
// not kept yet, by their SHA-256 in hexadecimal; one named without its
// file or its bytes (a stored image still being read back whose copy went
// missing) stays listed. The folder is made at the first save with
// something in it.
func (s *Slot) Save(text string, images []string, pasted map[string][]byte) error {
	if s.gone {
		return nil
	}
	if s.dir == "" {
		if text == "" && len(images) == 0 {
			return nil
		}
		if err := s.make(); err != nil {
			return err
		}
	}
	keep := map[string]bool{}
	for _, e := range images {
		_, sum, ok := ParsePasted(e)
		if !ok || sum == "" {
			continue
		}
		keep[sum+pastedExt] = true
		p := filepath.Join(s.dir, sum+pastedExt)
		if _, err := os.Stat(p); err == nil {
			continue // its bytes are its name: already kept
		}
		b, ok := pasted[sum]
		if !ok {
			// A stored image still being read back whose copy went
			// missing: listed as it was, its reading says so.
			continue
		}
		if err := writeAtomic(s.dir, p, b); err != nil {
			return err
		}
	}
	if len(images) > 0 {
		b, err := json.Marshal(images)
		if err != nil {
			return err
		}
		if err := writeAtomic(s.dir, filepath.Join(s.dir, listName), b); err != nil {
			return err
		}
	} else if err := removeFile(filepath.Join(s.dir, listName)); err != nil {
		return err
	}
	if text != "" {
		if err := writeAtomic(s.dir, filepath.Join(s.dir, textName), []byte(text)); err != nil {
			return err
		}
	} else if err := removeFile(filepath.Join(s.dir, textName)); err != nil {
		return err
	}
	// The draft is written: a pasted image's file that cannot be deleted
	// now is deleted by the next save, or swept at the next start.
	_ = prunePasted(s.dir, keep)
	return nil
}

// Pasted is a pasted image's stored bytes, checked against the SHA-256
// its entry names (design D8).
func (s *Slot) Pasted(sum string) ([]byte, error) {
	if s.dir == "" || !isHex(sum, 64) {
		return nil, errors.New("its stored copy is missing")
	}
	var b []byte
	err := retried(func() (err error) {
		b, err = os.ReadFile(filepath.Join(s.dir, sum+pastedExt))
		return err
	})
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return nil, errors.New("its stored copy is missing")
	case err != nil:
		return nil, errors.New("its stored copy cannot be read: " + err.Error())
	}
	if got := sha256.Sum256(b); hex.EncodeToString(got[:]) != sum {
		return nil, errors.New("its stored copy is damaged")
	}
	return b, nil
}

// Release lets the draft go (design D1): its folder stays, a stored draft
// for the project's next start, unless it holds no draft, when it is
// deleted. Nothing is written after it.
func (s *Slot) Release() error {
	if s == nil || s.gone {
		return nil
	}
	s.gone = true
	if s.dir == "" {
		return nil
	}
	_, keep := lastSave(s.dir)
	err := s.lock.Close()
	if !keep {
		_ = os.RemoveAll(s.dir) // what it cannot delete, the next start's sweep does
	}
	return err
}

// Abandon lets the draft go as a killed process does: the lock closed,
// every file left as it is, nothing written after (the crashes of the
// tests).
func (s *Slot) Abandon() {
	if s == nil || s.gone {
		return
	}
	s.gone = true
	if s.lock != nil {
		s.lock.Close()
	}
}

// Dir is the slot's folder, "" before its first save.
func (s *Slot) Dir() string { return s.dir }

// sweep deletes what no draft needs (design D9): in each folder nobody
// holds, the temporary files and the pasted images' files its list does
// not name, and the folder when it holds no draft; beside the folders, a
// temporary file a minute old. A held folder is never touched. What it
// fails to delete is no error: the next start tries again.
func sweep(root string, es []fs.DirEntry) {
	now := time.Now()
	for _, e := range es {
		p := filepath.Join(root, e.Name())
		if !e.IsDir() {
			if strings.HasPrefix(e.Name(), tempPrefix) && older(e, now) {
				_ = os.Remove(p)
			}
			continue
		}
		if !isSlot(e.Name()) {
			continue
		}
		lock, err := openExclusive(filepath.Join(p, lockName), false)
		switch {
		case errors.Is(err, fs.ErrNotExist):
			if !older(e, now) {
				continue // being made
			}
		case err != nil:
			continue // held
		}
		sweepSlot(p)
		_, keep := lastSave(p)
		if lock != nil {
			lock.Close()
		}
		if !keep {
			_ = os.RemoveAll(p)
		}
	}
}

// sweepSlot deletes a folder's temporary files and the pasted images'
// files its list does not name; with a list that cannot be read, none of
// them.
func sweepSlot(dir string) {
	es, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	keep := map[string]bool{}
	listed := true
	if b, err := os.ReadFile(filepath.Join(dir, listName)); err == nil {
		var images []string
		if json.Unmarshal(b, &images) != nil {
			listed = false
		}
		for _, e := range images {
			if _, sum, ok := ParsePasted(e); ok && sum != "" {
				keep[sum+pastedExt] = true
			}
		}
	} else if !errors.Is(err, fs.ErrNotExist) {
		listed = false
	}
	for _, e := range es {
		switch name := e.Name(); {
		case e.IsDir():
		case strings.HasPrefix(name, tempPrefix):
			_ = os.Remove(filepath.Join(dir, name))
		case listed && isPastedFile(name) && !keep[name]:
			_ = os.Remove(filepath.Join(dir, name))
		}
	}
}

// prunePasted deletes the pasted images' files keep does not name, each
// tried once: one another program holds stays for the next save.
func prunePasted(dir string, keep map[string]bool) error {
	es, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	var errs []error
	for _, e := range es {
		if name := e.Name(); !e.IsDir() && isPastedFile(name) && !keep[name] {
			if err := os.Remove(filepath.Join(dir, name)); err != nil && !errors.Is(err, fs.ErrNotExist) {
				errs = append(errs, err)
			}
		}
	}
	return errors.Join(errs...)
}

func isPastedFile(name string) bool {
	return strings.HasSuffix(name, pastedExt) && isHex(strings.TrimSuffix(name, pastedExt), 64)
}

func older(e fs.DirEntry, now time.Time) bool {
	info, err := e.Info()
	return err == nil && now.Sub(info.ModTime()) > newborn
}

// writeAtomic writes b to path through a temporary file in dir, synced,
// then renamed over path.
func writeAtomic(dir, path string, b []byte) error {
	f, err := os.CreateTemp(dir, tempPrefix+"*")
	if err != nil {
		return err
	}
	tmp := f.Name()
	_, werr := f.Write(b)
	if err := errors.Join(werr, f.Sync(), f.Close()); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	if err := retried(func() error { return os.Rename(tmp, path) }); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

func removeFile(p string) error {
	if err := retried(func() error { return os.Remove(p) }); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return nil
}

// retried runs a rename or a delete again, for about a second, while
// Windows refuses it because another program has the file open; that
// program lets go soon (plan 33: a save refused once was a draft unsaved).
func retried(op func() error) error {
	for wait := time.Millisecond; ; wait *= 2 {
		err := op()
		if err == nil || !transient(err) || wait > 512*time.Millisecond {
			return err
		}
		time.Sleep(wait)
	}
}
