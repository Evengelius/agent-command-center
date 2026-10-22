package app

import (
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/Evengelius/agent-command-center/internal/input"
	"github.com/Evengelius/agent-command-center/internal/render"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// zones is the F6 order (ui-design.md, Keys); zone D is gone since plan 24.
var zones = []string{"B", "C", "A"}

// key handles one key press (ui-design.md, Keys). Step 0 binds the
// essentials; step 3 adds selection, undo, the external editor and the rest.
func (m *Model) key(k tea.KeyPressMsg) []tea.Cmd {
	if k.String() == "ctrl+q" {
		m.wt = nil // a quit question shows on the screen under it
		return m.do(state.Quit{})
	}
	if m.s.QuitAsked && k.String() == "esc" {
		return m.do(state.Stay{})
	}
	if m.wt != nil {
		return m.worktreesKey(k)
	}
	// Ctrl+W, on both screens: the Worktrees view (plan 16); not while a
	// start is validated or a quit question waits.
	if k.String() == "ctrl+w" && m.browse == nil && !m.s.QuitAsked &&
		!(m.s.Screen == state.ScreenStartup && m.s.Startup.Phase == state.Validating) {
		return m.openWorktrees()
	}
	if m.s.Screen == state.ScreenStartup && m.browse != nil {
		return m.browseKey(k)
	}
	if m.s.Screen == state.ScreenStartup {
		return m.startupKey(k)
	}
	return m.dashboardKey(k)
}

func (m *Model) startupKey(k tea.KeyPressMsg) []tea.Cmd {
	switch k.String() {
	case "esc":
		if m.s.Startup.Phase == state.Asking {
			return m.do(state.Quit{})
		}
	case "up", "down":
		if m.ui.Field == 3 {
			// The project list (plan 15); elsewhere ↑↓ choose the agent.
			step := -1
			if k.String() == "down" {
				step = 1
			}
			m.ui.ProjectSel = max(min(m.ui.ProjectSel+step, len(m.projectList)-1), 0)
			return nil
		}
		m.moveChoice(k.String() == "down")
	case "ctrl+o":
		// The project browser (plan 15), not while a start is validated.
		if m.s.Startup.Phase != state.Validating {
			return []tea.Cmd{m.openBrowser()}
		}
	case "tab":
		m.nextField(1)
	case "shift+tab":
		m.nextField(-1)
	case "delete":
		switch m.ui.Field {
		case 1:
			return m.editKey(&m.path, k, true)
		case 3:
			return []tea.Cmd{m.forgetProject()}
		}
	case "f":
		switch m.ui.Field {
		case 1:
			return m.editKey(&m.path, k, true)
		case 3:
			return []tea.Cmd{m.starProject()}
		}
	case "enter":
		if m.ui.Field == 3 && m.ui.ProjectSel < len(m.projectList) {
			m.path.SetText(m.projectList[m.ui.ProjectSel].Path)
		}
		cmds := m.do(state.Confirm{Provider: m.ui.Choice, Project: m.path.Text()})
		m.focusMessage()
		return cmds
	default:
		if m.ui.Field == 1 {
			return m.editKey(&m.path, k, true)
		}
	}
	return nil
}

// moveChoice moves the radio selection among the installed CLIs.
func (m *Model) moveChoice(down bool) {
	inst := m.s.Startup.Installed
	if len(inst) == 0 {
		return
	}
	i := 0
	for j, id := range inst {
		if id == m.ui.Choice {
			i = j
		}
	}
	if down {
		i = (i + 1) % len(inst)
	} else {
		i = (i + len(inst) - 1) % len(inst)
	}
	m.ui.Choice = inst[i]
}

// focusMessage moves the focus to the field a validation message belongs to.
func (m *Model) focusMessage() {
	switch m.s.Startup.Field {
	case "provider":
		m.ui.Field = 0
	case "project":
		m.ui.Field = 1
	}
}

func (m *Model) dashboardKey(k tea.KeyPressMsg) []tea.Cmd {
	if m.ui.Keys {
		if cmds, done := m.overlayKey(k); done {
			return cmds
		}
	}
	// The window in front, the draft, the quick open, the file viewer or the
	// Explorer, takes every key but the global ones (plan 17, Keys;
	// amendments 12 and 13).
	switch {
	case m.ui.DraftWin.Open:
		if cmds, done := m.draftWinKey(k); done {
			return cmds
		}
	case m.ui.Finder.Open:
		if cmds, done := m.finderKey(k); done {
			return cmds
		}
	case m.ui.File.Open:
		if cmds, done := m.fileKey(k); done {
			return cmds
		}
	case m.ui.Explorer.Open:
		if cmds, done := m.explorerKey(k); done {
			return cmds
		}
	}
	// F1 anywhere, ? outside the editor: the key overlay (ui-design.md, Focus
	// and keys).
	if k.String() == "f1" || k.Text == "?" && (m.ui.Focus != "B" || render.WindowOpen(m.ui) || m.ui.Card != "" || m.ui.Inspect != "" || m.ui.Details || m.ui.View != "" || m.ui.ViewEntry > 0) {
		m.openOverlay()
		return nil
	}
	// F4 from any zone, with a card open too: Manual or Auto approvals (plan 12,
	// item 2).
	if k.String() == "f4" {
		capable := m.sess != nil && m.sess.Capabilities().AutoApprovals
		return m.do(state.ToggleApprovals{Capable: capable})
	}
	// F3: the file the entry or request on screen names, else the Explorer
	// (plan 17, Opening; amendment 12).
	if k.String() == "f3" {
		return m.f3()
	}
	if m.ui.View != "" || m.ui.ViewEntry > 0 {
		if cmds, done := m.viewKey(k); done {
			return cmds
		}
	}
	if m.ui.Card != "" {
		if cmds, done := m.cardKey(k); done {
			return cmds
		}
	}
	if m.ui.Inspect != "" && m.s.Agents[m.ui.Inspect] != nil {
		if cmds, done := m.inspectKey(k); done {
			return cmds
		}
	}
	if m.ui.Details {
		if cmds, done := m.detailsKey(k); done {
			return cmds
		}
	}
	if m.ui.Entry > 0 {
		if cmds, done := m.entryKey(k); done {
			return cmds
		}
	}
	switch k.String() {
	case "ctrl+c", "ctrl+shift+c":
		// Copy the selection if there is one, the mouse's first; else Ctrl+C
		// interrupts (ui-design.md, Selection and copy).
		if !m.ui.Sel.Empty() && m.screen != nil {
			text := m.screen.Text(m.s, m.ui.Sel)
			m.ui.Sel = render.Selection{} // as in WT: the next Ctrl+C interrupts
			return m.copy(text)
		}
		if m.ui.Focus == "B" && m.edit.HasSelection() {
			return m.copy(m.edit.SelectedText())
		}
		if k.String() == "ctrl+c" {
			return m.do(state.Interrupt{})
		}
		return nil
	case "f12":
		// Capture off: Windows Terminal selects on its own (ui-design.md,
		// Selection and copy).
		m.ui.MouseOff = !m.ui.MouseOff
		m.drag.on = false
		return nil
	case "shift+pgup", "shift+pgdown":
		m.scroll(m.pageStep(k.String() == "shift+pgdown"))
		return nil
	case "ctrl+o":
		return m.copy(lastAnswer(m.s))
	case "ctrl+r":
		return m.do(state.Resume{})
	case "f2":
		return m.focusCard("", true)
	case "esc":
		if m.s.Startup.Phase == state.StartFailed {
			cmds := m.do(state.Back{})
			m.closeSession()
			m.focusMessage()
			return cmds
		}
		if !m.ui.Sel.Empty() {
			m.ui.Sel = render.Selection{} // Esc clears a selection first
			return nil
		}
		if m.ui.Focus == "B" && m.edit.HasSelection() {
			m.edit.ClearSelection()
			return nil
		}
		if m.ui.Zoom {
			m.ui.Zoom = false // back to the exact dashboard, the focus kept
			return nil
		}
		m.ui.Focus = "B"
		return nil
	case "f6", "shift+f6":
		m.ui.Focus = nextZone(m.ui.Focus, k.String() == "f6")
		return nil
	case "tab", "shift+tab":
		if m.ui.Focus != "B" { // in the editor Tab inserts a tab
			m.ui.Focus = nextZone(m.ui.Focus, k.String() == "tab")
			return nil
		}
	case "z":
		if m.ui.Focus != "B" { // in the editor z is typed
			m.ui.Zoom = !m.ui.Zoom
			return nil
		}
	case "F":
		if m.s.Turn.ForceStopOffered {
			return m.do(state.ForceStop{})
		}
	}
	switch m.ui.Focus {
	case "B":
		switch k.String() {
		case "pgup", "pgdown":
			m.scroll(m.pageStep(k.String() == "pgdown"))
			return nil
		case "up":
			// ↑ from an empty input: the cursor on the newest entry B shows
			// in part (ui-design.md, Zone B, Expanding).
			if m.ui.Card == "" && m.edit.Text() == "" && m.moveEntry(len(m.s.Conversation), -1) {
				return nil
			}
		}
		return m.editorKey(k)
	case "A":
		if m.paneKey('A', k.String()) {
			return nil
		}
		return m.treeKey(k)
	case "C":
		if m.paneKey('C', k.String()) {
			return nil
		}
		return m.listKey(k)
	}
	if k.String() == "end" {
		m.follow() // End resumes follow mode; in B, End is the editor's, in A and C the pane's
	}
	return nil
}

// closeSession forgets a failed session after Back; its process was already
// closed by the crash.
func (m *Model) closeSession() {
	if m.cancel != nil {
		m.cancel()
	}
	m.sess, m.cancel = nil, nil
	m.closeWindows()
	m.ui.DraftWin = render.DraftWindow{}
	m.ui.Explorer = render.ExplorerView{} // the next session may be in another project
	m.draft = nil                         // the stored draft stays on disk for the next start
	m.edit.Clear()
}

// editKey applies an editing key; oneLine refuses newlines (the path field).
func (m *Model) editKey(e *input.Editor, k tea.KeyPressMsg, oneLine bool) []tea.Cmd {
	switch k.String() {
	case "backspace":
		e.Backspace()
	case "delete":
		e.Delete()
	case "left":
		e.Left()
	case "right":
		e.Right()
	case "home":
		e.Home()
	case "end":
		e.End()
	case "ctrl+v":
		// A terminal that sends a raw Ctrl+V instead of a bracketed paste:
		// read the system clipboard (ui-design.md, Input editor).
		if m.opt.Clipboard == nil {
			return nil
		}
		cb := m.opt.Clipboard
		return []tea.Cmd{func() tea.Msg {
			text, err := cb.Read()
			if err != nil {
				return toastMsg{"paste failed: " + err.Error()}
			}
			return pasteMsg{text}
		}}
	default:
		if k.Text != "" && k.Mod&(tea.ModCtrl|tea.ModAlt) == 0 {
			text := k.Text
			if oneLine {
				text = strings.NewReplacer("\r", "", "\n", "").Replace(text)
			}
			e.Insert(text)
		}
	}
	return nil
}

func nextZone(z string, forward bool) string {
	return step(zones, z, forward)
}

// step moves to the next or previous item of list, wrapping; from an
// unknown item it goes to the first.
func step(list []string, cur string, forward bool) string {
	if len(list) == 0 {
		return ""
	}
	for i, x := range list {
		if x == cur {
			if forward {
				return list[(i+1)%len(list)]
			}
			return list[(i+len(list)-1)%len(list)]
		}
	}
	return list[0]
}

// cardKey handles a key while an approval card has the focus (ui-design.md,
// Approvals). Only a focused card takes a, s, d and c, and the focus stays on
// the card until its answer is final, so a second press never answers
// another card. Letters typed meanwhile go nowhere; done is false for the
// keys that keep their usual meaning (Ctrl+Q, F6, ...).
func (m *Model) cardKey(k tea.KeyPressMsg) (cmds []tea.Cmd, done bool) {
	ap := m.s.Approvals[m.ui.Card]
	if ap.Question && ap.Status == state.Pending {
		if cmds, done := m.questionKey(ap, k.String()); done {
			return cmds, true
		}
	}
	switch k.String() {
	case "a":
		return m.do(state.Answer{ID: ap.ID, Decision: m.decision(true)}), true
	case "d":
		return m.do(state.Answer{ID: ap.ID, Decision: m.decision(false)}), true
	case "s":
		if !ap.ForSession || ap.Question { // only a Codex request takes acceptForSession
			return nil, true // not offered: nothing sent
		}
		return m.do(state.Answer{ID: ap.ID, Decision: "acceptForSession"}), true
	case "c":
		return m.do(state.CancelTurn{ID: ap.ID}), true
	case "esc":
		m.ui.Card = "" // later: back to the input, still pending
		m.ui.Focus = "B"
		return nil, true
	case "enter":
		m.openView(ap.ID, 0)
		return nil, true
	case "up", "down":
		return m.focusCard(ap.ID, k.String() == "down"), true
	}
	return nil, k.Text != "" && k.Mod&(tea.ModCtrl|tea.ModAlt) == 0
}

// decision is the CLI's word for allow once or deny (ui-design.md,
// Approvals). Allow for the session (s, in cardKey) is never offered for
// Claude Code (CC-4), and for Codex unless the request's availableDecisions
// omits acceptForSession (plan 5, item 8; plan 8); the amendments it may
// list are never offered.
func (m *Model) decision(allow bool) string {
	switch {
	case m.s.Context.Provider == "codex" && allow:
		return "accept"
	case m.s.Context.Provider == "codex":
		return "decline"
	case allow:
		return "allow"
	}
	return "deny"
}

// focusCard gives the focus to a pending approval card: the oldest with
// from "", else the next or previous pending one after from. With none
// pending, the focus stays where it is and a toast says so.
func (m *Model) focusCard(from string, forward bool) []tea.Cmd {
	var pending []string
	for _, id := range m.s.Queue {
		if m.s.Approvals[id].Status == state.Pending {
			pending = append(pending, id)
		}
	}
	if len(pending) == 0 {
		if from == "" {
			return m.toast("no approval waiting")
		}
		return nil
	}
	id := pending[0]
	if from != "" {
		id = step(pending, from, forward)
		if !contains(pending, from) {
			id = pending[0]
		}
	}
	m.ui.Card, m.ui.Focus = id, "B"
	m.follow() // the pending cards are the conversation's latest rows
	return nil
}

// settleCard returns the focus to the input once the focused card's answer
// is final, or its approval is gone (a resumed session).
func (m *Model) settleCard() {
	if m.ui.Card == "" {
		return
	}
	if ap := m.s.Approvals[m.ui.Card]; ap == nil || ap.Status.Final() {
		m.ui.Card = ""
	}
}

// questionKey handles the keys of a pending question card, on the card and in
// its full view alike (plan 14, item 1): 1-4 choose an option of the selected
// question, ← → select a question, a sends the answers once every question has
// a choice. done is false for the keys it does not take.
func (m *Model) questionKey(ap *state.Approval, key string) (cmds []tea.Cmd, done bool) {
	qs := render.QuestionsOf(ap)
	if len(qs) == 0 {
		return nil, false
	}
	if !m.ui.Pick.Valid(ap.ID, qs) { // another approval's pick, or none yet
		m.ui.Pick = render.NewPick(ap.ID, qs)
	}
	p := &m.ui.Pick
	switch key {
	case "1", "2", "3", "4":
		q, j := qs[p.At], int(key[0]-'1')
		if j >= len(q.Options) {
			return nil, true // no such option: nothing happens
		}
		if q.Multi {
			p.Chosen[p.At][j] = !p.Chosen[p.At][j]
			return nil, true
		}
		for i := range p.Chosen[p.At] {
			p.Chosen[p.At][i] = i == j
		}
		if p.At+1 < len(qs) {
			p.At++
		}
	case "left":
		p.At = max(p.At-1, 0)
	case "right":
		p.At = min(p.At+1, len(qs)-1)
	case "a":
		answers, ok := p.Answers(qs)
		if !ok {
			return m.toast("choose an option for each question first"), true
		}
		return m.do(state.Answer{ID: ap.ID, Decision: "allow", Answers: answers}), true
	default:
		return nil, false
	}
	return nil, true
}

// pickComplete says whether every question of ap has a choice.
func (m *Model) pickComplete(ap *state.Approval) bool {
	qs := render.QuestionsOf(ap)
	if !m.ui.Pick.Valid(ap.ID, qs) {
		return false
	}
	_, ok := m.ui.Pick.Answers(qs)
	return ok
}
