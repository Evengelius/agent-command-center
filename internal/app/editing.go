package app

import (
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/Evengelius/agent-command-center/internal/input"
	"github.com/Evengelius/agent-command-center/internal/render"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// toastFor is how long a toast replaces the key hints (ui-design.md,
// Clipboard, Feedback).
const toastFor = 3 * time.Second

type (
	copiedMsg    struct{ seq, toast string }
	toastDoneMsg struct{ gen int }
)

// editorKey handles a key in zone B's editor (ui-design.md, Input editor).
func (m *Model) editorKey(k tea.KeyPressMsg) []tea.Cmd {
	e := &m.edit
	name := k.String()
	// The send and newline keys are the configured lists (plan 21, item 5.1);
	// a key in neither falls through to the fixed ones below.
	switch {
	case m.opt.Keys.IsSend(name):
		switch {
		case m.pasting:
			// The paste's end marker never came: no key submits
			// (ui-design.md, Paste).
			e.Newline()
		case !e.HasSelection() && strings.HasSuffix(e.Text()[:e.Cursor()], `\`):
			// `\` then a send key: a newline, as in Claude Code (ui-design.md, Keys).
			e.Backspace()
			e.Newline()
		case strings.TrimSpace(e.Text()) != "":
			m.follow()            // sending shows the end of the conversation
			m.ui.KeysProblem = "" // a mock run's status row shows the configuration's problem until now
			return m.do(state.Send{Text: e.Text()})
		}
		return nil
	case m.opt.Keys.IsNewline(name):
		e.Newline()
		return nil
	}
	switch name {
	case "tab":
		e.Insert("\t")
	case "ctrl+a":
		e.SelectAll()
	case "ctrl+z":
		e.Undo()
	case "ctrl+y":
		e.Redo()
	case "ctrl+x":
		if e.HasSelection() {
			return m.copy(e.Cut())
		}
	case "ctrl+backspace", "alt+backspace", "ctrl+h": // WT sends 0x08, read as ctrl+h
		e.DeleteWordLeft()
	case "ctrl+delete", "alt+delete":
		e.DeleteWordRight()
	case "ctrl+g":
		m.ui.DraftWin = render.DraftWindow{Open: true} // the same draft, in a window (amendment 13)
	default:
		if mo, extend, ok := motion(k); ok {
			e.Move(mo, extend)
			return nil
		}
		return m.editKey(e, k, false)
	}
	return nil
}

// motion maps a movement key, with Shift extending the selection.
func motion(k tea.KeyPressMsg) (mo input.Motion, extend, ok bool) {
	mod := k.Mod &^ (tea.ModCapsLock | tea.ModNumLock | tea.ModScrollLock)
	extend = mod&tea.ModShift != 0
	word := mod&(tea.ModCtrl|tea.ModAlt) != 0
	if mod&^(tea.ModShift|tea.ModCtrl|tea.ModAlt) != 0 {
		return 0, false, false
	}
	switch k.Code {
	case tea.KeyLeft:
		mo = input.Left
		if word {
			mo = input.WordLeft
		}
	case tea.KeyRight:
		mo = input.Right
		if word {
			mo = input.WordRight
		}
	case tea.KeyUp:
		mo = input.Up
	case tea.KeyDown:
		mo = input.Down
	case tea.KeyHome:
		mo = input.LineStart
		if word {
			mo = input.BufferStart
		}
	case tea.KeyEnd:
		mo = input.LineEnd
		if word {
			mo = input.BufferEnd
		}
	default:
		return 0, false, false
	}
	return mo, extend, true
}

// copy puts text on the clipboard, native and OSC 52, off the event loop
// (ui-design.md, Clipboard).
func (m *Model) copy(text string) []tea.Cmd {
	c := input.Copier{Native: m.opt.Clipboard, Getenv: m.opt.Getenv}
	return []tea.Cmd{func() tea.Msg {
		seq, toast, err := c.Copy(text)
		if err != nil {
			return copiedMsg{toast: "copy failed: " + err.Error()}
		}
		return copiedMsg{seq, toast}
	}}
}

// lastAnswer is the orchestrator's last message, for Ctrl+O.
func lastAnswer(s *state.State) string {
	for i := len(s.Conversation) - 1; i >= 0; i-- {
		if e := s.Conversation[i]; e.Role == "assistant" && e.Text != "" {
			return e.Text
		}
	}
	return ""
}

// toast replaces the key hints for a moment.
func (m *Model) toast(text string) []tea.Cmd {
	m.toastGen++
	m.ui.Toast = text
	if c := m.opt.After(toastFor, toastDoneMsg{m.toastGen}); c != nil {
		return []tea.Cmd{c}
	}
	return nil
}

// editorView is the part of the draft the renderer draws: the logical lines
// around the cursor, never the whole buffer (ui-design.md, Input editor,
// Budget).
func (m *Model) editorView() render.EditorView {
	e := &m.edit
	cur := e.CursorLine()
	span := max(m.ui.Height, render.MinHeight)
	first := max(cur-span, 0)
	v := render.EditorView{
		Lines:   e.LineRange(first, 2*span),
		First:   first,
		Total:   e.Lines(),
		Line:    cur,
		Pasting: m.pasting,
	}
	text := e.Text()
	v.Col = e.Cursor() - (strings.LastIndexByte(text[:e.Cursor()], '\n') + 1)
	if from, to, ok := e.Selection(); ok {
		v.From, v.To = linePos(text, from), linePos(text, to)
	}
	return v
}

// linePos turns a byte offset into (line, byte in line).
func linePos(text string, off int) [2]int {
	return [2]int{strings.Count(text[:off], "\n"), off - (strings.LastIndexByte(text[:off], '\n') + 1)}
}

// draftWinKey handles a key in the draft window: the input box's keys,
// Esc back to it with the draft kept, Ctrl+F full screen (amendment 13).
func (m *Model) draftWinKey(k tea.KeyPressMsg) (cmds []tea.Cmd, done bool) {
	switch k.String() {
	case "f1", "f4", "f12", "ctrl+c", "ctrl+shift+c", "ctrl+q", "ctrl+o":
		return nil, false
	case "f2", "f3", "f6", "shift+f6":
		m.ui.DraftWin = render.DraftWindow{} // the approvals, the Explorer or another zone instead
		return nil, false
	case "esc":
		if !m.ui.Sel.Empty() {
			m.ui.Sel = render.Selection{} // Esc clears a selection first
		} else {
			m.ui.DraftWin = render.DraftWindow{}
		}
		return nil, true
	case "ctrl+f":
		m.ui.DraftWin.Full = !m.ui.DraftWin.Full
		return nil, true
	case "ctrl+g":
		return nil, true // the window is open
	}
	m.ui.Focus = "B"
	draft := m.edit.Text() != ""
	cmds = m.editorKey(k)
	if m.opt.Keys.IsSend(k.String()) && draft && m.edit.Text() == "" {
		m.ui.DraftWin = render.DraftWindow{} // sent: back to the dashboard
	}
	return cmds, true
}
