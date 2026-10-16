package app

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
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

// editorReturnedAt is the shortest run a blocking editor is expected to take;
// a quicker return means a GUI editor without its blocking flag (ui-design.md,
// $EDITOR, item 4).
const editorReturnedAt = time.Second

type (
	copiedMsg    struct{ seq, toast string }
	toastDoneMsg struct{ gen int }
	editedMsg    struct {
		path string
		took time.Duration
		note string
		err  error
	}
)

// editorKey handles a key in zone B's editor (ui-design.md, Input editor).
func (m *Model) editorKey(k tea.KeyPressMsg) []tea.Cmd {
	e := &m.edit
	switch k.String() {
	case "enter":
		switch {
		case m.pasting:
			// The paste's end marker never came: no key submits
			// (ui-design.md, Paste).
			e.Newline()
		case !e.HasSelection() && strings.HasSuffix(e.Text()[:e.Cursor()], `\`):
			// `\` then Enter: a newline, as in Claude Code (ui-design.md, Keys).
			e.Backspace()
			e.Newline()
		case strings.TrimSpace(e.Text()) != "":
			m.follow() // sending shows the end of the conversation
			return m.do(state.Send{Text: e.Text()})
		}
	case "ctrl+j", "shift+enter", "alt+enter":
		e.Newline()
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
		return m.externalEditor()
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

// externalEditor opens the draft in $VISUAL or $EDITOR (ui-design.md,
// $EDITOR). The draft stays in the editor whatever happens.
func (m *Model) externalEditor() []tea.Cmd {
	args, note, err := editorCommand(m.opt.Getenv, exec.LookPath)
	if err != nil {
		return m.toast("editor: " + err.Error())
	}
	f, err := os.CreateTemp("", "acc-draft-*.md")
	if err != nil {
		return m.toast("editor: " + err.Error())
	}
	path := f.Name()
	_, err = f.WriteString(m.edit.Text())
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		_ = os.Remove(path)
		return m.toast("editor: " + err.Error())
	}
	c := exec.Command(args[0], append(args[1:], path)...)
	start := m.opt.Now()
	return []tea.Cmd{tea.ExecProcess(c, func(err error) tea.Msg {
		return editedMsg{path: path, took: m.opt.Now().Sub(start), note: note, err: err}
	})}
}

// edited reads the file back byte for byte and deletes it; on failure the
// draft is kept as it was.
func (m *Model) edited(msg editedMsg) []tea.Cmd {
	defer os.Remove(msg.path)
	if msg.err != nil {
		return m.toast("editor failed: " + msg.err.Error() + "; the draft is kept")
	}
	b, err := os.ReadFile(msg.path)
	if err != nil {
		return m.toast("editor: " + err.Error() + "; the draft is kept")
	}
	if string(b) != m.edit.Text() {
		m.edit.SelectAll()
		m.edit.Insert(string(b)) // one undo step
	}
	switch {
	case msg.took < editorReturnedAt:
		return m.toast("the editor returned immediately: add --wait (or your editor's blocking flag)")
	case msg.note != "":
		return m.toast(msg.note)
	}
	return nil
}

// editorCommand resolves VISUAL, then EDITOR, split as the platform's shell
// would; with neither, a detected editor, named in the note.
func editorCommand(getenv func(string) string, lookPath func(string) (string, error)) (args []string, note string, err error) {
	for _, name := range []string{"VISUAL", "EDITOR"} {
		if v := strings.TrimSpace(getenv(name)); v != "" {
			args, err := splitCommand(v)
			if err != nil || len(args) == 0 {
				return nil, "", fmt.Errorf("$%s %q cannot be read: %v", name, v, err)
			}
			return args, "", nil
		}
	}
	candidates := [][]string{{"code", "--wait"}, {"nano"}, {"vi"}}
	if runtime.GOOS == "windows" {
		candidates = [][]string{{"code", "--wait"}, {"notepad"}}
	}
	for _, c := range candidates {
		if _, err := lookPath(c[0]); err == nil {
			return c, "no $VISUAL or $EDITOR: using " + strings.Join(c, " "), nil
		}
	}
	return nil, "", errors.New("no $VISUAL or $EDITOR, and no editor found")
}
