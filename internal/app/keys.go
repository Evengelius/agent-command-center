package app

import (
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/Evengelius/agent-command-center/internal/input"
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
	switch k.String() {
	case "ctrl+c":
		return m.do(state.Interrupt{})
	case "ctrl+r":
		return m.do(state.Resume{})
	case "esc":
		if m.s.Startup.Phase == state.StartFailed {
			cmds := m.do(state.Back{})
			m.closeSession()
			m.focusMessage()
			return cmds
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
		case "enter":
			if strings.TrimSpace(m.edit.Text()) == "" {
				return nil
			}
			return m.do(state.Send{Text: m.edit.Text()})
		case "ctrl+j":
			m.edit.Newline()
			return nil
		}
		return m.editKey(&m.edit, k, false)
	case "C":
		switch k.String() {
		case "up", "down":
			m.ui.Selected = step(m.s.Order, m.ui.Selected, k.String() == "down")
		}
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
				return nil
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
