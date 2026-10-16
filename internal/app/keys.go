package app

import (
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/Evengelius/agent-command-center/internal/input"
	"github.com/Evengelius/agent-command-center/internal/render"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// zones is the F6 order (ui-design.md, Keys).
var zones = []string{"B", "C", "A", "D"}

// key handles one key press (ui-design.md, Keys). Step 0 binds the
// essentials; step 3 adds selection, undo, the external editor and the rest.
func (m *Model) key(k tea.KeyPressMsg) []tea.Cmd {
	if k.String() == "ctrl+q" {
		return m.do(state.Quit{})
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
		m.moveChoice(k.String() == "down")
	case "tab":
		m.ui.Field = (m.ui.Field + 1) % 3
	case "shift+tab":
		m.ui.Field = (m.ui.Field + 2) % 3
	case "enter":
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
	if m.ui.Card != "" {
		if cmds, done := m.cardKey(k); done {
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
		m.ui.Focus = "B"
		return nil
	case "f6", "shift+f6":
		m.ui.Focus = nextZone(m.ui.Focus, k.String() == "f6")
		return nil
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
		}
		return m.editorKey(k)
	case "C":
		switch k.String() {
		case "up", "down":
			m.ui.Selected = step(m.s.Order, m.ui.Selected, k.String() == "down")
		}
	}
	if k.String() == "end" {
		m.follow() // End resumes follow mode; in B, End is the editor's
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
	m.draft = nil // the stored draft stays on disk for the next start
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
// Approvals). Only a focused card takes a, d and c, and the focus stays on
// the card until its answer is final, so a second press never answers
// another card. Letters typed meanwhile go nowhere; done is false for the
// keys that keep their usual meaning (Ctrl+Q, F6, ...).
func (m *Model) cardKey(k tea.KeyPressMsg) (cmds []tea.Cmd, done bool) {
	ap := m.s.Approvals[m.ui.Card]
	switch k.String() {
	case "a":
		return m.do(state.Answer{ID: ap.ID, Decision: m.decision(true)}), true
	case "d":
		return m.do(state.Answer{ID: ap.ID, Decision: m.decision(false)}), true
	case "c":
		return m.do(state.CancelTurn{ID: ap.ID}), true
	case "esc":
		m.ui.Card = "" // later: back to the input, still pending
		m.ui.Focus = "B"
		return nil, true
	case "up", "down":
		return m.focusCard(ap.ID, k.String() == "down"), true
	}
	return nil, k.Text != "" && k.Mod&(tea.ModCtrl|tea.ModAlt) == 0
}

// decision is the CLI's word for allow once or deny (ui-design.md,
// Approvals). Allow for the session (s) is never offered for Claude Code
// (CC-4); Codex offers it from step 5, when the request says it may.
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
