package input

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"
)

// files lists a folder's file names, sorted.
func files(t *testing.T, dir string) []string {
	t.Helper()
	es, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, e := range es {
		out = append(out, e.Name())
	}
	slices.Sort(out)
	return out
}

func take(t *testing.T, d Drafts, project, cli string) (*Slot, StoredDraft) {
	t.Helper()
	s, st, err := d.Take(project, cli)
	if err != nil {
		t.Fatal(err)
	}
	return s, st
}

func save(t *testing.T, s *Slot, text string, images []string, pasted map[string][]byte) {
	t.Helper()
	if err := s.Save(text, images, pasted); err != nil {
		t.Fatal(err)
	}
}

func readFile(t *testing.T, p string) string {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// age sets a file's or a folder's times to d ago.
func age(t *testing.T, p string, d time.Duration) {
	t.Helper()
	at := time.Now().Add(-d)
	if err := os.Chtimes(p, at, at); err != nil {
		t.Fatal(err)
	}
}

// TestDraftSlots is plan 33's criterion 33.1 (design D1): one folder per
// dashboard's draft, made at its first save, held while its dashboard
// runs, taken by the project's next start once let go, one of the same
// CLI first, then the newest; a draft from before plan 33 moved into a
// folder; the text kept byte for byte.
func TestDraftSlots(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("a held file is Windows' (plan 33, design D1)")
	}
	root := filepath.Join(t.TempDir(), "drafts")
	d := Drafts{Dir: root}
	project := filepath.Join(t.TempDir(), "secret-project")
	text := "\tline 1  \r\nline 2\n日本語 👍🏽\x1b[0m\x00end"

	s, st := take(t, d, project, "claude-code")
	if st.Text != "" || st.Images != nil || st.Others != 0 || s.Dir() != "" {
		t.Fatalf("nothing stored: took %+v, folder %q", st, s.Dir())
	}
	if _, err := os.Stat(root); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("the drafts folder exists before the first save: %v", err)
	}
	save(t, s, "", nil, nil)
	if _, err := os.Stat(root); !errors.Is(err, fs.ErrNotExist) {
		t.Fatal("an empty save made the drafts folder")
	}
	save(t, s, text, []string{`C:\a.png`}, nil)
	es := files(t, root)
	if len(es) != 1 || !isSlot(es[0]) || !strings.HasPrefix(es[0], DraftKey(project)+"-") || strings.Contains(es[0], "secret") {
		t.Fatalf("drafts folder %v; want one folder named by the project's SHA-256 and a key", es)
	}
	dir := filepath.Join(root, es[0])
	if dir != s.Dir() {
		t.Fatalf("slot folder %q, want %q", s.Dir(), dir)
	}
	if got := files(t, dir); !slices.Equal(got, []string{"draft.txt", "images.json", "lock"}) {
		t.Fatalf("folder files %v; no temporary file may be left", got)
	}
	if got := readFile(t, filepath.Join(dir, "draft.txt")); got != text {
		t.Fatalf("byte for byte: got %q", got)
	}
	if got := readFile(t, filepath.Join(dir, "images.json")); got != `["C:\\a.png"]` {
		t.Fatalf("images list %s", got)
	}

	// Held: another dashboard of the project takes nothing and touches
	// nothing of it.
	other, st := take(t, d, project, "claude-code")
	if st.Text != "" || other.Dir() != "" {
		t.Fatalf("a held draft was taken: %+v", st)
	}
	if _, err := os.ReadFile(filepath.Join(dir, "lock")); err == nil {
		t.Fatal("the lock can be read while held")
	}
	if got := files(t, dir); !slices.Equal(got, []string{"draft.txt", "images.json", "lock"}) {
		t.Fatalf("held folder after another start: %v", got)
	}

	// Let go: the next start of the project takes it, unchanged.
	if err := s.Release(); err != nil {
		t.Fatal(err)
	}
	if err := s.Save("after the release", nil, nil); err != nil || readFile(t, filepath.Join(dir, "draft.txt")) != text {
		t.Fatalf("a save after the release changed the draft: %v", err)
	}
	if got := readFile(t, filepath.Join(dir, "lock")); got != "claude-code" {
		t.Fatalf("lock holds %q", got)
	}
	again, st := take(t, d, project, "codex")
	if st.Text != text || !slices.Equal(st.Images, []string{`C:\a.png`}) || st.Others != 0 || again.Dir() != dir {
		t.Fatalf("took %+v from %q; want the draft let go, in %q", st, again.Dir(), dir)
	}
	again.Release()
	if got := readFile(t, filepath.Join(dir, "lock")); got != "codex" {
		t.Fatalf("lock holds %q after Codex took the draft", got)
	}
	other.Release()

	// Two let go, one per CLI: each CLI takes its own.
	p2 := filepath.Join(t.TempDir(), "p2")
	a, _ := take(t, d, p2, "claude-code")
	b, _ := take(t, d, p2, "codex")
	save(t, a, "from claude", nil, nil)
	save(t, b, "from codex", nil, nil)
	age(t, filepath.Join(b.Dir(), "draft.txt"), time.Hour) // Claude Code's is the newer
	a.Release()
	b.Release()
	c, st := take(t, d, p2, "codex")
	if st.Text != "from codex" || st.Others != 1 {
		t.Fatalf("Codex took %+v; want its own, the other counted", st)
	}
	cc, st := take(t, d, p2, "claude-code")
	if st.Text != "from claude" || st.Others != 0 {
		t.Fatalf("Claude Code took %+v; want its own", st)
	}
	c.Release()
	cc.Release()

	// Two of one CLI: the newer, the other counted.
	p3 := filepath.Join(t.TempDir(), "p3")
	x, _ := take(t, d, p3, "codex")
	y, _ := take(t, d, p3, "codex")
	save(t, x, "older", nil, nil)
	save(t, y, "newer", nil, nil)
	age(t, filepath.Join(x.Dir(), "draft.txt"), time.Hour)
	x.Release()
	y.Release()
	z, st := take(t, d, p3, "codex")
	if st.Text != "newer" || st.Others != 1 {
		t.Fatalf("took %+v; want the newer, one more counted", st)
	}
	z.Release()

	// A draft from before plan 33, beside the folders.
	p4 := filepath.Join(t.TempDir(), "p4")
	key := DraftKey(p4)
	if err := os.WriteFile(filepath.Join(root, key+".txt"), []byte("old text"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, key+".images.json"), []byte(`["C:\\old.png","pasted:pasted-image-1.png"]`), 0o600); err != nil {
		t.Fatal(err)
	}
	old, st := take(t, d, p4, "claude-code")
	if st.Text != "old text" || !slices.Equal(st.Images, []string{`C:\old.png`, "pasted:pasted-image-1.png"}) {
		t.Fatalf("took %+v from before plan 33", st)
	}
	for _, n := range []string{key + ".txt", key + ".images.json"} {
		if _, err := os.Stat(filepath.Join(root, n)); !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("%s left beside the folders", n)
		}
	}
	if got := files(t, old.Dir()); !slices.Equal(got, []string{"draft.txt", "images.json", "lock"}) || !strings.HasPrefix(filepath.Base(old.Dir()), key+"-") {
		t.Fatalf("moved into %q: %v", old.Dir(), got)
	}
	old.Release()

	// Emptied, then let go: its folder deleted.
	p5 := filepath.Join(t.TempDir(), "p5")
	e, _ := take(t, d, p5, "codex")
	save(t, e, "soon sent", nil, nil)
	gone := e.Dir()
	save(t, e, "", nil, nil)
	if got := files(t, gone); !slices.Equal(got, []string{"lock"}) {
		t.Fatalf("emptied folder %v", got)
	}
	e.Release()
	if _, err := os.Stat(gone); !errors.Is(err, fs.ErrNotExist) {
		t.Error("an empty draft's folder was kept after its release")
	}
}

// TestDraftKey: the project's key is the SHA-256 of its cleaned path,
// whatever its case on Windows.
func TestDraftKey(t *testing.T) {
	project := `C:\Work\Some Project\`
	sum := sha256.Sum256([]byte(strings.ToLower(filepath.Clean(project))))
	if got := DraftKey(project); got != hex.EncodeToString(sum[:]) {
		t.Errorf("key %s", got)
	}
	if os.PathSeparator == '\\' && DraftKey(`c:\work\some project`) != DraftKey(project) {
		t.Error("the same project in another case has another key")
	}
}

// TestImageOnlySave is criterion 33.6 (design D5): a draft without text
// keeps its images list.
func TestImageOnlySave(t *testing.T) {
	d := Drafts{Dir: t.TempDir()}
	s, _ := take(t, d, t.TempDir(), "claude-code")
	images := []string{`C:\a.png`, `C:\b.png`}
	save(t, s, "", images, nil)
	if got := files(t, s.Dir()); !slices.Equal(got, []string{"images.json", "lock"}) {
		t.Fatalf("image-only draft: %v", got)
	}
	for _, text := range []string{"text", ""} {
		save(t, s, text, images, nil)
		if got := readFile(t, filepath.Join(s.Dir(), "images.json")); got != `["C:\\a.png","C:\\b.png"]` {
			t.Fatalf("after a save of %q: list %s", text, got)
		}
	}
	if _, err := os.Stat(filepath.Join(s.Dir(), "draft.txt")); !errors.Is(err, fs.ErrNotExist) {
		t.Error("an empty text left its file")
	}
	save(t, s, "", nil, nil)
	if got := files(t, s.Dir()); !slices.Equal(got, []string{"lock"}) {
		t.Fatalf("an empty draft left %v", got)
	}
	s.Release()
}

// TestDraftPasted is design D8's store: a pasted image kept as its
// SHA-256 with ".png", read back checked, deleted once no longer listed.
func TestDraftPasted(t *testing.T) {
	d := Drafts{Dir: t.TempDir()}
	s, _ := take(t, d, t.TempDir(), "claude-code")
	data := []byte("\x89PNG\r\n\x1a\nnot really an image, bytes all the same")
	sum := sha256.Sum256(data)
	hexSum := hex.EncodeToString(sum[:])
	entry := PastedEntry("pasted-image-1.png", sum)
	if entry != "pasted:pasted-image-1.png#"+hexSum {
		t.Fatalf("entry %q", entry)
	}
	for _, c := range []struct {
		in, name, sum string
		ok            bool
	}{
		{entry, "pasted-image-1.png", hexSum, true},
		{"pasted:pasted-image-1.png", "pasted-image-1.png", "", true},
		{"pasted:odd#name.png", "odd#name.png", "", true},
		{`C:\x#y.png`, "", "", false},
	} {
		if name, sum, ok := ParsePasted(c.in); name != c.name || sum != c.sum || ok != c.ok {
			t.Errorf("ParsePasted(%q) = %q, %q, %v", c.in, name, sum, ok)
		}
	}
	// Named without its file or its bytes (a stored image still read
	// back whose copy went missing): listed, nothing written for it.
	save(t, s, "t", []string{entry}, nil)
	if got := readFile(t, filepath.Join(s.Dir(), "images.json")); got != `["`+entry+`"]` {
		t.Fatalf("list %s", got)
	}
	if _, err := os.Stat(filepath.Join(s.Dir(), hexSum+".png")); !errors.Is(err, fs.ErrNotExist) {
		t.Fatal("a copy written without its bytes")
	}
	save(t, s, "t", []string{entry, `C:\a.png`}, map[string][]byte{hexSum: data})
	stored := filepath.Join(s.Dir(), hexSum+".png")
	if got := readFile(t, stored); got != string(data) {
		t.Fatal("the stored copy differs from the pasted bytes")
	}
	if got, err := s.Pasted(hexSum); err != nil || !bytes.Equal(got, data) {
		t.Fatalf("Pasted: %v", err)
	}
	save(t, s, "t2", []string{entry}, nil) // kept already: no bytes needed
	if err := os.WriteFile(stored, append(data, '!'), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Pasted(hexSum); err == nil || err.Error() != "its stored copy is damaged" {
		t.Errorf("an altered copy: %v", err)
	}
	save(t, s, "t2", []string{`C:\a.png`}, nil)
	if _, err := os.Stat(stored); !errors.Is(err, fs.ErrNotExist) {
		t.Error("a copy no longer listed was kept")
	}
	if _, err := s.Pasted(hexSum); err == nil || err.Error() != "its stored copy is missing" {
		t.Errorf("a missing copy: %v", err)
	}
	s.Release()
}

// TestDamagedList is design D6's store: a list that cannot be parsed is
// set aside, its bytes unchanged, and the text taken.
func TestDamagedList(t *testing.T) {
	d := Drafts{Dir: t.TempDir()}
	project := t.TempDir()
	s, _ := take(t, d, project, "claude-code")
	save(t, s, "precious text", []string{`C:\a.png`}, nil)
	dir := s.Dir()
	s.Release()
	damaged := `["C:\\half`
	if err := os.WriteFile(filepath.Join(dir, "images.json"), []byte(damaged), 0o600); err != nil {
		t.Fatal(err)
	}
	s2, st := take(t, d, project, "claude-code")
	if st.Text != "precious text" || st.Images != nil || !st.SetAside {
		t.Fatalf("took %+v", st)
	}
	if got := readFile(t, filepath.Join(dir, "images.json.unreadable")); got != damaged {
		t.Errorf("set aside %q", got)
	}
	if _, err := os.Stat(filepath.Join(dir, "images.json")); !errors.Is(err, fs.ErrNotExist) {
		t.Error("the damaged list stayed")
	}
	s2.Release()
}

// TestDraftSweep is criterion 33.10 (design D9).
func TestDraftSweep(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("a held file is Windows' (plan 33, design D1)")
	}
	root := t.TempDir()
	d := Drafts{Dir: root}
	project := t.TempDir()
	data := []byte("pasted bytes")
	sum := sha256.Sum256(data)
	hexSum := hex.EncodeToString(sum[:])
	stray := strings.Repeat("ab", 32) + ".png"

	// A draft let go, with a temporary file and a stray pasted file; a
	// held draft, with the same (both started before either was let go).
	a, _ := take(t, d, project, "claude-code")
	h, _ := take(t, d, project, "codex")
	save(t, a, "kept", []string{PastedEntry("pasted-image-1.png", sum)}, map[string][]byte{hexSum: data})
	save(t, h, "held", nil, nil)
	aDir := a.Dir()
	a.Release()
	for _, n := range []string{".draft-123", stray} {
		if err := os.WriteFile(filepath.Join(aDir, n), []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	for _, n := range []string{".draft-456", stray} {
		if err := os.WriteFile(filepath.Join(h.Dir(), n), []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	heldFiles := files(t, h.Dir())
	// A folder holding only its lock (a dashboard killed with nothing
	// stored), one without its lock a minute old, one just made.
	slot := func(k string) string {
		p := filepath.Join(root, DraftKey(project)+"-"+k)
		if err := os.Mkdir(p, 0o700); err != nil {
			t.Fatal(err)
		}
		return p
	}
	lockOnly := slot("0000000000000001")
	if err := os.WriteFile(filepath.Join(lockOnly, "lock"), []byte("codex"), 0o600); err != nil {
		t.Fatal(err)
	}
	lockless, young := slot("0000000000000002"), slot("0000000000000003")
	age(t, lockless, 2*time.Minute)
	// Beside the folders: temporary files of a version before plan 33.
	oldTemp, newTemp := filepath.Join(root, ".draft-old"), filepath.Join(root, ".draft-new")
	for _, p := range []string{oldTemp, newTemp} {
		if err := os.WriteFile(p, []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	age(t, oldTemp, 2*time.Minute)

	s, _ := take(t, d, t.TempDir(), "claude-code") // another project's start sweeps too
	s.Release()

	if got := files(t, aDir); !slices.Equal(got, []string{hexSum + ".png", "draft.txt", "images.json", "lock"}) {
		t.Errorf("draft let go after the sweep: %v", got)
	}
	if got := files(t, h.Dir()); !slices.Equal(got, heldFiles) {
		t.Errorf("held draft after the sweep: %v, want %v", got, heldFiles)
	}
	for _, p := range []string{lockOnly, lockless, oldTemp} {
		if _, err := os.Stat(p); !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("%s kept", filepath.Base(p))
		}
	}
	for _, p := range []string{young, newTemp} {
		if _, err := os.Stat(p); err != nil {
			t.Errorf("%s deleted", filepath.Base(p))
		}
	}
	h.Release()
}

// TestSaveWhileRead: a program reading the list without sharing its
// deletion (an antivirus, an indexer, Go's os.Open) makes Windows refuse
// the rename over it; the save waits for it to let go (plan 33).
func TestSaveWhileRead(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows refuses the rename")
	}
	d := Drafts{Dir: t.TempDir()}
	s, _ := take(t, d, t.TempDir(), "claude-code")
	save(t, s, "text", []string{`C:\a.png`}, nil)
	list := filepath.Join(s.Dir(), "images.json")
	f, err := os.Open(list)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(list, list); err == nil {
		t.Log("the rename over an open file succeeded at once")
	}
	done := make(chan error, 1)
	go func() { done <- s.Save("text", []string{`C:\b.png`}, nil) }()
	time.Sleep(100 * time.Millisecond)
	f.Close()
	if err := <-done; err != nil {
		t.Fatalf("the save while the list was read: %v", err)
	}
	if got := readFile(t, list); got != `["C:\\b.png"]` {
		t.Errorf("list %s", got)
	}
	// Held for longer than the retries: the error is told.
	f, err = os.Open(list)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := s.Save("text", []string{`C:\c.png`}, nil); err == nil {
		t.Error("a save refused for two seconds reported no error")
	}
	s.Release()
}

// TestTakeWhileOpen: a stored draft whose lock another program holds a
// moment is taken once it lets go; one held for longer is counted as in
// use, not lost in silence (plan 33).
func TestTakeWhileOpen(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("a held file is Windows'")
	}
	d := Drafts{Dir: t.TempDir()}
	project := t.TempDir()
	s, _ := take(t, d, project, "claude-code")
	save(t, s, "kept", nil, nil)
	lock := filepath.Join(s.Dir(), "lock")
	s.Release()

	f, err := os.Open(lock) // a reader sharing reads and writes: the unshared open waits
	if err != nil {
		t.Fatal(err)
	}
	go func() { time.Sleep(100 * time.Millisecond); f.Close() }()
	s2, st := take(t, d, project, "claude-code")
	if st.Text != "kept" || st.Others != 0 {
		t.Fatalf("taken %+v once the reader let go", st)
	}
	s2.Release()

	f, err = os.Open(lock)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	s3, st := take(t, d, project, "claude-code")
	if st.Text != "" || st.Others != 1 || s3.Dir() != "" {
		t.Errorf("held past the retries: taken %+v; want nothing, one in use", st)
	}
	s3.Release()
}

// TestTakeOwned: a draft another dashboard owns is left at once, neither
// waited for nor counted, whether it was held before the start looked or
// taken between its look and its open (plan 33, the review's FAIL 13).
func TestTakeOwned(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("a held file is Windows'")
	}
	d := Drafts{Dir: t.TempDir()}
	project := t.TempDir()
	owner, _ := take(t, d, project, "codex")
	save(t, owner, "owned", nil, nil)
	start := time.Now()
	s, st := take(t, d, project, "codex")
	if took := time.Since(start); st.Text != "" || st.Others != 0 || took > 300*time.Millisecond {
		t.Errorf("held before the look: %+v after %v; want nothing, none counted, no wait", st, took)
	}
	s.Release()
	// Taken between the look and the open: adopt alone, on that candidate.
	other := &Slot{root: d.Dir, key: DraftKey(project), cli: "claude-code"}
	start = time.Now()
	if got := other.adopt(candidate{dir: owner.Dir()}, time.Now().Add(time.Second)); got != adoptGone {
		t.Errorf("adopt of an owned draft: %v, want gone", got)
	}
	if took := time.Since(start); took > 100*time.Millisecond {
		t.Errorf("adopt waited %v for an owned draft", took)
	}
	owner.Release()
}

// TestPruneWhileOpen: a pasted image's file another program holds when
// its draft no longer lists it does not fail the save; the next save
// deletes it (plan 33).
func TestPruneWhileOpen(t *testing.T) {
	d := Drafts{Dir: t.TempDir()}
	s, _ := take(t, d, t.TempDir(), "claude-code")
	data := []byte("pasted bytes")
	sum := sha256.Sum256(data)
	hexSum := hex.EncodeToString(sum[:])
	save(t, s, "t", []string{PastedEntry("pasted-image-1.png", sum)}, map[string][]byte{hexSum: data})
	copy := filepath.Join(s.Dir(), hexSum+".png")
	f, err := os.Open(copy)
	if err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	if err := s.Save("t2", nil, nil); err != nil {
		t.Fatalf("the save failed on its prune: %v", err)
	}
	if took := time.Since(start); took > 500*time.Millisecond {
		t.Errorf("the save waited %v for its prune", took)
	}
	if got := readFile(t, filepath.Join(s.Dir(), "draft.txt")); got != "t2" {
		t.Errorf("text %q", got)
	}
	f.Close()
	save(t, s, "t3", nil, nil)
	if _, err := os.Stat(copy); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("the unlisted copy after the next save: %v", err)
	}
	s.Release()
}
