package app

import (
	"encoding/hex"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/Evengelius/agent-command-center/internal/input"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// draftEvery is the most often a changing text is written to disk; a change
// of the images is written at once (plan 33, design D3).
const draftEvery = 500 * time.Millisecond

// draftRetry is how long a failed save waits to be tried again (plan 33).
const draftRetry = time.Second

type (
	draftLoadedMsg struct {
		store  *draftStore
		slot   *input.Slot
		stored input.StoredDraft
		err    error
	}
	draftSaveMsg   struct{}
	draftFailedMsg struct{ err error }
	draftRetryMsg  struct{}
)

// draftStore is this dashboard's stored draft (proposal step 4, item 4;
// ASM-23): a folder of its own in the drafts folder, held until it is let
// go (plan 33, design D1). Saves run outside Update, possibly two at once;
// gen keeps an older draft from overwriting a newer one.
type draftStore struct {
	d       input.Drafts
	project string
	cli     string
	mu      sync.Mutex
	last    int         // the generation written last
	slot    *input.Slot // nil until the stored draft was taken
	gone    bool        // let go: nothing more is written
}

func (ds *draftStore) save(gen int, text string, images []string, pasted map[string][]byte) error {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	if gen <= ds.last || ds.slot == nil || ds.gone {
		return nil
	}
	ds.last = gen
	return ds.slot.Save(text, images, pasted)
}

// release lets the draft go: its folder waits for the project's next
// start, or is deleted when it holds no draft. A slot taken after it (the
// session given up while its draft loaded) is let go as it arrives.
func (ds *draftStore) release() error {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	ds.gone = true
	return ds.slot.Release()
}

// pastedCopy is a pasted image's stored bytes, checked (design D8).
func (ds *draftStore) pastedCopy(sum string) ([]byte, error) {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	return ds.slot.Pasted(sum)
}

// pathsKey is the paths as one string, to tell a change.
func pathsKey(paths []string) string { return strings.Join(paths, "\x00") }

// imagesKey is the stored images list as a save would write it now: the
// same whether a stored image is still being read back or attached
// (plan 33).
func (m *Model) imagesKey() string {
	entries, _ := m.draftEntries(m.storedPaths())
	return pathsKey(entries)
}

// loadDraft takes the project's stored draft when its session first
// starts: one of this CLI first, then the newest (plan 33, design D1).
// Nothing is saved before it is read, so the stored draft is never
// overwritten by what was typed while it loaded.
func (m *Model) loadDraft() tea.Cmd {
	if m.opt.Drafts == nil || m.s.Context.Mock || m.draft != nil {
		return nil
	}
	ds := &draftStore{d: *m.opt.Drafts, project: m.s.Context.Project, cli: m.s.Context.Provider}
	m.draft, m.draftReady = ds, false
	return func() tea.Msg {
		slot, st, err := ds.d.Take(ds.project, ds.cli)
		return draftLoadedMsg{ds, slot, st, err}
	}
}

func (m *Model) draftLoaded(msg draftLoadedMsg) []tea.Cmd {
	msg.store.mu.Lock()
	msg.store.slot = msg.slot
	msg.store.mu.Unlock()
	if msg.store != m.draft {
		_ = msg.store.release() // a session given up since (Back): the draft waits for the next
		return nil
	}
	m.draftReady = true
	if msg.err != nil {
		// Saving would replace a draft that could not be read: drafts stay
		// off for this run, and the screen says so.
		m.draft = nil
		_ = msg.store.release()
		return m.toast("stored draft not read, drafts off: " + msg.err.Error())
	}
	st := msg.stored
	var restore, gone []string
	for _, e := range st.Images {
		if name, sum, pasted := input.ParsePasted(e); pasted && sum == "" {
			gone = append(gone, name) // a plan 32 entry: its bytes were never kept
			continue
		}
		restore = append(restore, e)
	}
	// What is stored, as a save writes it: the list without plan 32's
	// entries differs, and the next change saves it without them.
	m.draftSaved, m.draftImgs = st.Text, pathsKey(st.Images)
	if st.Text == "" && len(st.Images) == 0 && !st.SetAside {
		switch {
		case st.Others == 1:
			// Its lock held by another program even after the retries: it
			// waits for the project's next start, said (plan 33).
			return m.toast("a stored draft of this project is in use: the next start takes it")
		case st.Others > 1:
			return m.toast(fmt.Sprintf("%d stored drafts of this project are in use: the next starts take them", st.Others))
		}
		return nil
	}
	if st.Text != "" {
		m.edit.SetText(joinDraft(st.Text, m.edit.Text())) // typed while it loaded: both kept
	}
	note := "draft restored"
	switch len(gone) {
	case 0:
	case 1:
		note += "; " + gone[0] + " was pasted, not kept"
	default:
		note += "; " + strings.Join(gone, ", ") + " were pasted, not kept"
	}
	if st.SetAside {
		note += "; its images list was unreadable and was set aside"
	}
	switch {
	case st.Others == 1:
		note += " · 1 more stored draft of this project"
	case st.Others > 1:
		note += fmt.Sprintf(" · %d more stored drafts of this project", st.Others)
	}
	cmds := m.toast(note)
	if len(restore) > 0 {
		// Its images, read and checked again: a file deleted since is said
		// and left out (plan 31, design D6), a pasted image comes from its
		// stored copy (plan 33, design D8). Until they are read, however
		// long that takes (a share gone offline), saves list them first,
		// as stored, so nothing typed or attached meanwhile waits on them.
		m.unread = restore
		cmds = append(cmds, readDraftImages(restore, msg.store, inputLimits(m.limits())))
	}
	return cmds
}

// inspectPreview reads and checks a stored draft's file, loadImage a file
// as it is sent, and unreachable says its drive or share cannot be
// reached; a test puts a share gone offline in their place (plan 33).
var (
	inspectPreview = input.InspectPreview
	loadImage      = input.LoadImage
	unreachable    = input.Unreachable
)

// noteUnreachable is a stored draft's file whose drive or share cannot be
// reached: kept as an attachment, unread, Sent 0 (plan 33).
const noteUnreachable = "its folder cannot be reached; kept in the draft"

// readDraftImages reads the stored draft's images again, in their order:
// a file as an upload is read, a pasted image from its stored copy checked
// against its SHA-256 (plan 33, design D8).
func readDraftImages(entries []string, ds *draftStore, lim input.ImageLimits) tea.Cmd {
	return func() tea.Msg {
		out := inspectedMsg{restore: true, store: ds, limited: lim.MaxCount > 0, thumbs: map[string]thumbSixel{}}
		for _, e := range entries {
			name, sum, pasted := input.ParsePasted(e)
			if !pasted {
				img, im, err := inspectPreview(e, lim)
				if err != nil && unreachable(e) {
					// A share gone offline, a drive unplugged: kept in the
					// draft, shown, sent once it answers (plan 33).
					out.imgs = append(out.imgs, input.Image{Path: e, Name: filepath.Base(e), Note: noteUnreachable})
					continue
				}
				if err != nil {
					out.refused = append(out.refused, filepath.Base(e)+": "+err.Error())
					continue
				}
				out.imgs = append(out.imgs, img)
				out.thumbs[img.Path] = thumbsOf(im)
				continue
			}
			b, err := ds.pastedCopy(sum)
			if err != nil {
				out.refused = append(out.refused, name+": "+err.Error())
				continue
			}
			img, _, im, err := input.InspectData(name, b, lim)
			if err != nil {
				out.refused = append(out.refused, name+": "+err.Error())
				continue
			}
			img.Path = pastedPrefix + name
			out.imgs = append(out.imgs, img)
			out.thumbs[img.Path] = thumbsOf(im)
			if out.pasted == nil {
				out.pasted = map[string][]byte{}
			}
			out.pasted[img.Path] = b
		}
		return out
	}
}

// waiting says a sent message waits for its acceptance.
func waiting(t state.TurnState) bool {
	return t.Phase == state.Submitted && (t.Draft != "" || len(t.DraftImages) > 0)
}

// storedText is the text the stored draft holds now: the editor's, after
// the message waiting for its acceptance when one does (plan 33, design
// D4).
func (m *Model) storedText() string {
	if t := m.s.Turn; waiting(t) {
		return joinDraft(t.Draft, m.edit.Text())
	}
	return m.edit.Text()
}

// storedPaths are the images the stored draft holds now: the editor's,
// after the waiting message's, none twice (design D4).
func (m *Model) storedPaths() []string {
	paths := imagePaths(m.attach)
	if len(m.unread) > 0 {
		// The stored draft's images still being read back keep their
		// place first among the editor's, as it lists them; once read
		// they take that place in the editor (plan 33).
		paths = append(append([]string(nil), m.unread...), paths...)
	}
	if t := m.s.Turn; waiting(t) {
		sent := make([]string, len(t.DraftImages))
		for i, a := range t.DraftImages {
			sent[i] = a.Path
		}
		paths = m.mergeImages(sent, paths)
	}
	return paths
}

// joinDraft joins a draft's two texts by a newline, either alone as it is.
func joinDraft(a, b string) string {
	switch {
	case a == "":
		return b
	case b == "":
		return a
	}
	return a + "\n" + b
}

// mergeImages is sent's images, then those of since that are not among
// them: a file by its path, a pasted image by its bytes.
func (m *Model) mergeImages(sent, since []string) []string {
	out := append([]string(nil), sent...)
	for _, p := range since {
		dup := false
		for _, q := range sent {
			if m.sameImage(p, q) {
				dup = true
				break
			}
		}
		if !dup {
			out = append(out, p)
		}
	}
	return out
}

// sameImage says two keys are one image: a file's path compared as
// attachedAt does, a pasted image by its SHA-256.
func (m *Model) sameImage(p, q string) bool {
	if isPasted(p) || isPasted(q) {
		sp, okp := m.pastedSum[p]
		sq, okq := m.pastedSum[q]
		return p == q || okp && okq && sp == sq
	}
	a, b := filepath.Clean(p), filepath.Clean(q)
	return a == b || runtime.GOOS == "windows" && strings.EqualFold(a, b)
}

// draftChanged saves the draft after it changed: a change of its images at
// once, of its text at most every draftEvery (plan 33, design D3), the
// message waiting for its acceptance included (design D4), and the stored
// draft's images still being read back.
func (m *Model) draftChanged() []tea.Cmd {
	if m.draft == nil || !m.draftReady {
		return nil
	}
	if m.imagesKey() != m.draftImgs {
		return []tea.Cmd{m.saveDraft()}
	}
	if m.draftTimer || m.storedText() == m.draftSaved {
		return nil
	}
	if c := m.opt.After(draftEvery, draftSaveMsg{}); c != nil {
		m.draftTimer = true
		return []tea.Cmd{c}
	}
	return []tea.Cmd{m.saveDraft()}
}

// draftDue is the draft timer: it writes what is to be stored now.
func (m *Model) draftDue() []tea.Cmd {
	m.draftTimer = false
	if m.draft == nil || !m.draftReady {
		return nil
	}
	if m.storedText() == m.draftSaved && m.imagesKey() == m.draftImgs {
		return nil
	}
	return []tea.Cmd{m.saveDraft()}
}

// draftWrite writes the draft now when drafts are on: the message just
// sent, which stays the stored draft until the CLI accepts it, so a crash
// or a panic in between loses nothing (ASM-23); then, at the acceptance,
// what the editor holds, nothing if it is empty.
func (m *Model) draftWrite() tea.Cmd {
	if m.draft == nil || !m.draftReady {
		return nil
	}
	return m.saveDraft()
}

// draftAccepted is the CLI accepting the sent message.
func (m *Model) draftAccepted() tea.Cmd { return m.draftWrite() }

// draftFailed says a save failed and tries it again after draftRetry, for
// as long as it fails: a save refused once would otherwise leave the
// stored draft behind until the next change (plan 33).
func (m *Model) draftFailed(msg draftFailedMsg) []tea.Cmd {
	cmds := m.toast("draft not saved: " + msg.err.Error())
	if m.retrying {
		return cmds
	}
	if c := m.opt.After(draftRetry, draftRetryMsg{}); c != nil {
		m.retrying = true
		cmds = append(cmds, c)
	}
	return cmds
}

// draftRetried is the retry's timer: what is to be stored now, written.
func (m *Model) draftRetried() []tea.Cmd {
	m.retrying = false
	if c := m.draftWrite(); c != nil {
		return []tea.Cmd{c}
	}
	return nil
}

// saveDraft writes what is to be stored now, its text and its images,
// outside Update; an empty draft removes its files.
func (m *Model) saveDraft() tea.Cmd {
	m.draftGen++
	text, paths := m.storedText(), m.storedPaths()
	images, pasted := m.draftEntries(paths)
	m.draftSaved, m.draftImgs = text, pathsKey(images)
	ds, gen := m.draft, m.draftGen
	return func() tea.Msg {
		if err := ds.save(gen, text, images, pasted); err != nil {
			return draftFailedMsg{err}
		}
		return nil
	}
}

// draftEntries are the stored list's entries for paths: a file's path as
// it is, a pasted image's entry, its bytes to keep by their SHA-256 in
// hexadecimal (plan 33, design D8).
func (m *Model) draftEntries(paths []string) ([]string, map[string][]byte) {
	var out []string
	var pasted map[string][]byte
	for _, p := range paths {
		if _, sum, ok := input.ParsePasted(p); !isPasted(p) || ok && sum != "" {
			out = append(out, p) // a file, or a stored image still being read back: as listed
			continue
		}
		b, ok := m.pasted[p]
		if !ok {
			continue // a pasted image is attached with its bytes: never
		}
		sum := m.pastedSum[p]
		out = append(out, input.PastedEntry(strings.TrimPrefix(p, pastedPrefix), sum))
		if pasted == nil {
			pasted = map[string][]byte{}
		}
		pasted[hex.EncodeToString(sum[:])] = b
	}
	return out, pasted
}

// flushDraft writes the draft now, on the way out (quit, error, panic,
// Back). A message sent and not accepted is kept with what was typed and
// attached since (design D4); the stored draft's images still being read
// back keep their place (storedPaths).
func (m *Model) flushDraft() error {
	if m.draft == nil || !m.draftReady {
		return nil
	}
	m.draftGen++
	text, paths := m.storedText(), m.storedPaths()
	images, pasted := m.draftEntries(paths)
	m.draftSaved, m.draftImgs = text, pathsKey(images)
	return m.draft.save(m.draftGen, text, images, pasted)
}

// letDraftGo writes the draft, then lets it go (quit, Back): the project's
// next start takes it (plan 33, designs D1 and D2).
func (m *Model) letDraftGo() error {
	if m.draft == nil {
		return nil
	}
	err := m.flushDraft()
	if rerr := m.draft.release(); err == nil {
		err = rerr
	}
	m.draft, m.draftReady, m.draftTimer, m.unread = nil, false, false, nil
	return err
}
