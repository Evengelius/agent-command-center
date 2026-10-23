package app

import (
	"strings"
	"sync"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/Evengelius/agent-command-center/internal/input"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// draftEvery is the most often a changing draft is written to disk.
const draftEvery = 500 * time.Millisecond

type (
	draftLoadedMsg struct {
		store *draftStore
		text  string
		paths []string // the draft's attached images (plan 31)
		err   error
	}
	draftSaveMsg   struct{}
	draftFailedMsg struct{ err error }
)

// draftStore is the project's stored draft (proposal step 4, item 4;
// ASM-23). Saves run outside Update, possibly two at once; gen keeps an
// older text from overwriting a newer one.
type draftStore struct {
	d       input.Drafts
	project string
	mu      sync.Mutex
	last    int // the generation written last
}

func (ds *draftStore) save(gen int, text string, paths []string) error {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	if gen <= ds.last {
		return nil
	}
	ds.last = gen
	if err := ds.d.Save(ds.project, text); err != nil {
		return err
	}
	return ds.d.SaveImages(ds.project, paths) // after Save, which clears it with an empty text
}

// pathsKey is the paths as one string, to tell a change.
func pathsKey(paths []string) string { return strings.Join(paths, "\x00") }

// loadDraft reads the project's stored draft when its session first starts.
// Nothing is saved before it is read, so the stored draft is never
// overwritten by what was typed while it loaded.
func (m *Model) loadDraft() tea.Cmd {
	if m.opt.Drafts == nil || m.s.Context.Mock || m.draft != nil {
		return nil
	}
	ds := &draftStore{d: *m.opt.Drafts, project: m.s.Context.Project}
	m.draft, m.draftReady = ds, false
	return func() tea.Msg {
		text, err := ds.d.Load(ds.project)
		if err != nil {
			return draftLoadedMsg{ds, text, nil, err}
		}
		paths, err := ds.d.LoadImages(ds.project)
		return draftLoadedMsg{ds, text, paths, err}
	}
}

func (m *Model) draftLoaded(msg draftLoadedMsg) []tea.Cmd {
	if msg.store != m.draft {
		return nil // a session given up since (Back)
	}
	m.draftReady = true
	if msg.err != nil {
		// Saving would replace a draft that could not be read: drafts stay
		// off for this run, and the screen says so.
		m.draft = nil
		return m.toast("stored draft not read, drafts off: " + msg.err.Error())
	}
	m.draftSaved, m.draftImgs = msg.text, pathsKey(msg.paths)
	if msg.text == "" && len(msg.paths) == 0 {
		return nil
	}
	if msg.text != "" {
		if typed := m.edit.Text(); typed != "" {
			m.edit.SetText(msg.text + "\n" + typed) // typed while it loaded: both kept
		} else {
			m.edit.SetText(msg.text)
		}
	}
	cmds := m.toast("draft restored")
	if len(msg.paths) > 0 {
		// Its images, read and checked again: a file deleted since is said
		// and left out (plan 31, design D6).
		cmds = append(cmds, inspectImages(msg.paths, inputLimits(m.limits()), true))
	}
	return cmds
}

// draftChanged saves the draft after it changed, at most every draftEvery.
// While a sent message waits to be accepted the stored draft is that
// message, and nothing replaces it (a failed send restores it to the
// editor; an accepted one deletes it).
func (m *Model) draftChanged() []tea.Cmd {
	if m.draft == nil || !m.draftReady || m.draftTimer || m.s.Turn.Phase == state.Submitted || m.edit.Text() == m.draftSaved && pathsKey(imagePaths(m.attach)) == m.draftImgs {
		return nil
	}
	if c := m.opt.After(draftEvery, draftSaveMsg{}); c != nil {
		m.draftTimer = true
		return []tea.Cmd{c}
	}
	return []tea.Cmd{m.saveDraft()}
}

// draftDue is the draft timer: it writes what the editor holds now.
func (m *Model) draftDue() []tea.Cmd {
	m.draftTimer = false
	if m.draft == nil || !m.draftReady || m.s.Turn.Phase == state.Submitted || m.edit.Text() == m.draftSaved && pathsKey(imagePaths(m.attach)) == m.draftImgs {
		return nil
	}
	return []tea.Cmd{m.saveDraft()}
}

// saveDraft writes what the editor holds, its text and its images' paths,
// outside Update; an empty draft deletes the files.
func (m *Model) saveDraft() tea.Cmd {
	m.draftGen++
	paths := imagePaths(m.attach)
	m.draftSaved, m.draftImgs = m.edit.Text(), pathsKey(paths)
	ds, gen, text := m.draft, m.draftGen, m.draftSaved
	return func() tea.Msg {
		if err := ds.save(gen, text, paths); err != nil {
			return draftFailedMsg{err}
		}
		return nil
	}
}

// draftSent stores the message just sent as the draft until the CLI accepts
// it, so a crash or a panic in between loses nothing (ASM-23).
func (m *Model) draftSent(text string, imgs []state.Attachment) tea.Cmd {
	if m.draft == nil || !m.draftReady {
		return nil
	}
	m.draftGen++
	m.draftSaved, m.draftImgs = "", ""
	ds, gen := m.draft, m.draftGen
	var paths []string
	for _, a := range imgs {
		paths = append(paths, a.Path)
	}
	return func() tea.Msg {
		if err := ds.save(gen, text, paths); err != nil {
			return draftFailedMsg{err}
		}
		return nil
	}
}

// draftAccepted is the CLI accepting the sent message: the stored draft
// becomes what the editor holds now, nothing if it is empty.
func (m *Model) draftAccepted() tea.Cmd {
	if m.draft == nil || !m.draftReady {
		return nil
	}
	return m.saveDraft()
}

// flushDraft writes the draft now, on the way out (quit, error, panic). A
// message sent and not accepted is kept, with what was typed since, as a
// failed send would restore them.
func (m *Model) flushDraft() error {
	if m.draft == nil || !m.draftReady {
		return nil
	}
	text := m.edit.Text()
	paths := imagePaths(m.attach)
	if d := m.s.Turn.Draft; m.s.Turn.Phase == state.Submitted && (d != "" || len(m.s.Turn.DraftImages) > 0) {
		if text != "" && d != "" {
			d += "\n" + text
		} else if d == "" {
			d = text
		}
		text = d
		// The sent images first, then the ones attached since (plan 31).
		var sent []string
		for _, a := range m.s.Turn.DraftImages {
			sent = append(sent, a.Path)
		}
		paths = append(sent, paths...)
	}
	m.draftGen++
	m.draftSaved, m.draftImgs = text, pathsKey(paths)
	return m.draft.save(m.draftGen, text, paths)
}
