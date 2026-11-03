package app

import (
	"slices"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/Evengelius/agent-command-center/internal/input"
	"github.com/Evengelius/agent-command-center/internal/render"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// The session history view (plan 75, brief amendment 89): the sessions the
// dashboard ran in the project the home screen's field names, listed from the
// project store, resumed or deleted from the list. A view of its own, full
// screen, modelled on the Worktrees view's.

// history is the view's state: the project it is about and the selection.
// The entries are the store's, read from the project list each time, so a
// change another dashboard made shows.
type history struct {
	project string // the project's path in the list, as Start resolves the field
	sel     int
	ask     byte // render.HistoryAskOne or HistoryAskAll while a delete question waits, else 0
	askSel  int  // 0 Cancel, 1 Delete
	note    string
	top     int    // the first entry shown (render.HistoryTop)
	lastKey string // the entry the last press landed on, by its CLI and ID; "" for none
	// asked is what the delete question counted: the entry, or all the
	// entries, by their CLI and ID; the answer removes exactly these.
	asked []input.Entry
}

// historyCheckedMsg is the answer of Alt+H's check of the field's path: gen
// the press it was for, raw the field's text, path the project as Start
// validates it, problem why Start would refuse it.
type historyCheckedMsg struct {
	gen                int
	raw, path, problem string
}

// openHistory is Alt+H on the home screen (plan 75, design D2): the view for
// the project field's path as Start validates it. Nothing while a start is
// validated; a path whose resolution Start
// refuses opens nothing, the field's note already saying why; the path is then
// checked on disk in a command, as Start's validation is, and one Start would
// refuse opens nothing and has the field show Start's own refusal.
func (m *Model) openHistory() []tea.Cmd {
	if m.folderOpen {
		return nil // a folder picked meanwhile would change the field under the view
	}
	if m.s.Startup.Phase == state.Validating || m.wt != nil || m.hist != nil {
		return nil
	}
	raw, env, check := m.path.Text(), m.env, m.check
	if input.ResolvePath(raw, env).Problem != "" {
		return nil
	}
	m.histRefused = ""
	m.histGen++
	gen := m.histGen
	return []tea.Cmd{func() tea.Msg {
		path, _, problem := check(raw, env)
		return historyCheckedMsg{gen, raw, path, problem}
	}}
}

// historyChecked opens the view once the path checked is the field's still
// and the screen has stayed as Alt+H found it, or has the field show Start's
// refusal.
func (m *Model) historyChecked(msg historyCheckedMsg) {
	if msg.gen != m.histGen || m.hist != nil || m.wt != nil || m.folderOpen || m.s.Screen != state.ScreenStartup || m.s.Startup.Phase == state.Validating || m.path.Text() != msg.raw {
		return
	}
	if msg.problem != "" {
		m.histRefused, m.histRefusedFor = msg.problem, msg.raw
		return
	}
	m.hist = &history{project: msg.path}
}

// detecting says detection is still running: which agent CLIs are installed
// is not known yet.
func (m *Model) detecting() bool {
	return m.s.Startup.Phase == state.Detecting || m.s.Startup.Phase == ""
}

// historyEntries are the history of the view's project, newest first: none
// without a store or a project in the list (Load keeps no entry without an ID).
func (m *Model) historyEntries() []input.Entry {
	if m.projects == nil {
		return nil
	}
	e, _ := m.projects.p.Find(m.projectList, m.hist.project)
	return e.History
}

// historyKey handles a key while the view is open (design D3): ↑ ↓ PgUp PgDn
// Home End move, Enter resumes, Del deletes, Shift+Del deletes all, Esc closes,
// F12 gives the mouse to the terminal and back.
// Ctrl+Q is the dashboard's before it comes here; every other key, Ctrl+N,
// Ctrl+O, Ctrl+R and Ctrl+W among them, does nothing.
func (m *Model) historyKey(k tea.KeyPressMsg) []tea.Cmd {
	h := m.hist
	if k.String() == "f12" {
		m.ui.MouseOff = !m.ui.MouseOff // the mouse to the terminal and back, as on the home screen
		return nil
	}
	if h.ask != 0 {
		return m.historyAskKey(k)
	}
	es := m.historyEntries()
	page := max((m.ui.Height-12)/2, 1)
	move := func(to int) { h.sel, h.note = max(min(to, len(es)-1), 0), "" }
	switch k.String() {
	case "esc":
		m.hist = nil
	case "up":
		move(h.sel - 1)
	case "down":
		move(h.sel + 1)
	case "pgup":
		move(h.sel - page)
	case "pgdown":
		move(h.sel + page)
	case "home":
		move(0)
	case "end":
		move(len(es) - 1)
	case "enter":
		return m.historyResume()
	case "delete":
		m.historyAsk(render.HistoryAskOne)
	case "shift+delete":
		m.historyAsk(render.HistoryAskAll)
	}
	return nil
}

// historyAsk opens the delete question, Cancel selected (design D4); nothing
// when there is no entry to delete.
func (m *Model) historyAsk(kind byte) {
	h := m.hist
	es := m.historyEntries()
	if len(es) == 0 {
		return
	}
	h.ask, h.askSel, h.note = kind, 0, ""
	h.asked = slices.Clone(es)
	if kind == render.HistoryAskOne {
		h.asked = []input.Entry{es[max(min(h.sel, len(es)-1), 0)]}
	}
}

// historyAskKey answers the delete question: ← selects Delete and → Cancel,
// the way they point as the buttons are drawn, as the return question's do;
// Enter answers, Esc cancels; nothing else does anything.
func (m *Model) historyAskKey(k tea.KeyPressMsg) []tea.Cmd {
	h := m.hist
	switch k.String() {
	case "left":
		h.askSel = 1
	case "right":
		h.askSel = 0
	case "enter":
		return m.historyAnswer(h.askSel == 1)
	case "esc":
		h.ask = 0
	}
	return nil
}

// historyAnswer closes the question; with yes, it deletes what it asked
// about from the project's history, by the entry's CLI and ID and never by
// its place, and from the project store, no other project's (design D4). The
// CLIs' own records are not touched.
func (m *Model) historyAnswer(yes bool) []tea.Cmd {
	h := m.hist
	h.ask = 0
	asked := h.asked
	if !yes || m.projects == nil || len(asked) == 0 {
		return nil
	}
	path, ps := h.project, m.projects.p
	m.projectList = ps.ForgetSessions(m.projectList, path, asked) // shown at once
	h.sel = max(min(h.sel, len(m.historyEntries())-1), 0)
	return []tea.Cmd{m.projectsCmd(func(p input.Projects, l []input.Project) []input.Project {
		return p.ForgetSessions(l, path, asked)
	})}
}

// historyResume is Enter on an entry and a double click on it (design D5):
// the project validated as START SESSION validates it, the entry's CLI
// chosen in question 1 and started on that session; an entry whose CLI is not
// installed says so and starts nothing.
func (m *Model) historyResume() []tea.Cmd {
	h := m.hist
	es := m.historyEntries()
	if h.sel >= len(es) {
		return nil
	}
	e := es[h.sel]
	if m.stillClosing() {
		h.note = stillClosingNote // its draft and store still being settled (plan 75)
		return nil
	}
	if m.detecting() {
		h.note = "The agent CLIs are still being detected"
		return nil
	}
	if !contains(m.s.Startup.Installed, e.CLI) {
		h.note = cliName(e.CLI) + " is not installed"
		return nil
	}
	proj, _ := m.projects.p.Find(m.projectList, h.project)
	newest, _ := proj.Newest(e.CLI, "")
	m.olderResume = newest.ID != e.ID // a refusal then names this session, not "the last"
	m.hist = nil
	m.ui.Choice = e.CLI
	cmds := m.do(state.Confirm{Provider: e.CLI, Project: m.path.Text(), Resume: e.ID, ResumeProject: proj.Path})
	m.focusMessage()
	return cmds
}

// historyMouse handles a mouse message while the view is open (design D3): a
// click on an entry selects it, a double click resumes it, a click outside
// the box closes the view (plan 58's rule), the wheel moves the selection one
// entry a notch; with the delete question open, a click answers it, one
// elsewhere cancels it. A right click, and anything else, does nothing.
func (m *Model) historyMouse(msg tea.MouseMsg) []tea.Cmd {
	h := m.hist
	if wheel, ok := msg.(tea.MouseWheelMsg); ok {
		if h.ask != 0 || m.screen == nil || !m.screen.History.Holds(wheel.X, wheel.Y) {
			return nil
		}
		es := m.historyEntries()
		switch wheel.Button {
		case tea.MouseWheelUp:
			h.sel, h.note = max(h.sel-1, 0), ""
		case tea.MouseWheelDown:
			h.sel, h.note = max(min(h.sel+1, len(es)-1), 0), ""
		}
		return nil
	}
	click, ok := msg.(tea.MouseClickMsg)
	if !ok || click.Button != tea.MouseLeft || m.screen == nil {
		return nil
	}
	m.countClick(click.X, click.Y)
	prev := h.lastKey // the entry the press before landed on
	h.lastKey = ""
	p, item := m.screen.ItemAt(click.X, click.Y)
	if p != render.HistoryPanel {
		item = 0
	}
	if h.ask != 0 {
		switch item {
		case render.HistoryItemDelete:
			return m.historyAnswer(true)
		case render.HistoryItemCancel:
			return m.historyAnswer(false)
		}
		if !m.screen.History.Holds(click.X, click.Y) {
			h.ask = 0 // a click elsewhere: the question is cancelled
		}
		return nil
	}
	if !m.screen.History.Holds(click.X, click.Y) {
		m.hist = nil
		return nil
	}
	if item < render.HistoryItemRow {
		return nil
	}
	i := item - render.HistoryItemRow
	if i >= len(m.historyEntries()) {
		return nil
	}
	e := m.historyEntries()[i]
	h.sel, h.note, h.lastKey = i, "", e.CLI+"/"+e.ID
	if m.click.n == 2 && prev == h.lastKey { // a double click: both presses on the same entry
		m.click.n = 0 // a third press starts afresh
		return m.historyResume()
	}
	return nil
}

// historyView is the view as the renderer draws it.
func (m *Model) historyView() render.HistoryView {
	h := m.hist
	if h == nil {
		return render.HistoryView{}
	}
	es := m.historyEntries()
	h.sel = max(min(h.sel, len(es)-1), 0)

	v := render.HistoryView{Open: true, Project: h.project, Sel: h.sel, Top: h.top, Ask: h.ask, AskSel: h.askSel, AskCount: len(h.asked), Note: h.note}
	for _, a := range h.asked {
		v.AskCLIs = append(v.AskCLIs, cliName(a.CLI))
	}
	if v.Note == "" {
		v.Note = m.ui.ProjectsProblem // the list may be missing sessions: said, not hidden
	}
	loc := m.opt.Location
	if loc == nil {
		loc = time.Local
	}
	for _, e := range es {
		meta := []string{e.Ended.In(loc).Format("2006-01-02 15:04"), age(m.ui.Now.Sub(e.Ended), e.Ended), cliName(e.CLI), "N/A", state.ShortID(e.ID)}
		if e.Model != "" {
			meta[3] = e.Model
		}
		v.Rows = append(v.Rows, render.HistoryRow{Title: e.Title, Meta: strings.Join(meta, " · "), CLI: cliName(e.CLI), Dim: !m.detecting() && !contains(m.s.Startup.Installed, e.CLI)})
	}
	h.top = render.HistoryTop(m.ui.Width, m.ui.Height, v) // the list scrolls as the selection needs, never under a click
	v.Top = h.top
	return v
}
