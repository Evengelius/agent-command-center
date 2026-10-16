package app

import (
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

func (ds *draftStore) save(gen int, text string) error {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	if gen <= ds.last {
		return nil
	}
	ds.last = gen
	return ds.d.Save(ds.project, text)
}

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
		return draftLoadedMsg{ds, text, err}
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
	m.draftSaved = msg.text
	if msg.text == "" {
		return nil
	}
	if typed := m.edit.Text(); typed != "" {
		m.edit.SetText(msg.text + "\n" + typed) // typed while it loaded: both kept
	} else {
		m.edit.SetText(msg.text)
	}
	return m.toast("draft restored")
}

// draftChanged saves the draft after it changed, at most every draftEvery.
// While a sent message waits to be accepted the stored draft is that
// message, and nothing replaces it (a failed send restores it to the
// editor; an accepted one deletes it).
func (m *Model) draftChanged() []tea.Cmd {
	if m.draft == nil || !m.draftReady || m.draftTimer || m.s.Turn.Phase == state.Submitted || m.edit.Text() == m.draftSaved {
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
	if m.draft == nil || !m.draftReady || m.s.Turn.Phase == state.Submitted || m.edit.Text() == m.draftSaved {
		return nil
	}
	return []tea.Cmd{m.saveDraft()}
}

// saveDraft writes what the editor holds, outside Update; an empty draft
// deletes the file.
func (m *Model) saveDraft() tea.Cmd {
	m.draftGen++
	m.draftSaved = m.edit.Text()
	ds, gen, text := m.draft, m.draftGen, m.draftSaved
	return func() tea.Msg {
		if err := ds.save(gen, text); err != nil {
			return draftFailedMsg{err}
		}
		return nil
	}
}

// draftSent stores the message just sent as the draft until the CLI accepts
// it, so a crash or a panic in between loses nothing (ASM-23).
func (m *Model) draftSent(text string) tea.Cmd {
	if m.draft == nil || !m.draftReady {
		return nil
	}
	m.draftGen++
	m.draftSaved = ""
	ds, gen := m.draft, m.draftGen
	return func() tea.Msg {
		if err := ds.save(gen, text); err != nil {
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
	if d := m.s.Turn.Draft; m.s.Turn.Phase == state.Submitted && d != "" {
		if text != "" {
			d += "\n" + text
		}
		text = d
	}
	m.draftGen++
	m.draftSaved = text
	return m.draft.save(m.draftGen, text)
}
