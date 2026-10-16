package app

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/Evengelius/agent-command-center/internal/event"
)

// The conversation the selection tests draw: a long line that wraps, a code
// block with a tab, and a tool row with its origin label.
var (
	longLine = strings.Repeat("alpha beta gamma delta ", 12) + "END"
	answer   = "intro line\n" + longLine + "\n```go\nfunc f() {\n\treturn 1\n}\n```\nlast line"
)

// selectRig is a dashboard showing that conversation, with a clock the test
// moves so separate clicks are not read as a double click.
func selectRig(t *testing.T) (inputRig, *fakeClipboard, *time.Time) {
	t.Helper()
	r, clip := copyRig(t, nil)
	now := testNow
	r.h.m.opt.Now = func() time.Time { return now }
	r.emit(event.MessageCompleted, event.Completed{MessageID: "m1", Text: answer})
	r.emit(event.ToolStarted, event.Tool{ToolID: "t1", Name: "Bash"})
	r.h.settle()
	return r, clip, &now
}

// at is the cell the nth occurrence (from 0) of s starts on in the frame.
func at(t *testing.T, h *harness, s string, nth int) (x, y int) {
	t.Helper()
	for y, l := range strings.Split(plainText(h.m.View().Content), "\n") {
		from := 0
		for {
			i := strings.Index(l[from:], s)
			if i < 0 {
				break
			}
			if nth == 0 {
				return lipgloss.Width(l[:from+i]), y
			}
			nth--
			from += i + len(s)
		}
	}
	t.Fatalf("%q not on the screen:\n%s", s, plainText(h.m.View().Content))
	return 0, 0
}

// mouseAt sends one mouse message and returns the copy it started, if any.
func mouseAt(t *testing.T, h *harness, msg tea.Msg) (copiedMsg, bool) {
	t.Helper()
	_ = h.m.View() // the frame the pointer is over
	_, cmd := h.m.Update(msg)
	var got copiedMsg
	found := false
	for _, m := range msgsOf(cmd) {
		if c, ok := m.(copiedMsg); ok {
			got, found = c, true
		}
		h.deliver(m)
	}
	h.settle()
	return got, found
}

func click(x, y int) tea.MouseClickMsg {
	return tea.MouseClickMsg{X: x, Y: y, Button: tea.MouseLeft}
}

// dragCopy drags from one cell to another; the release copies.
func dragCopy(t *testing.T, h *harness, x0, y0, x1, y1 int) {
	t.Helper()
	mouseAt(t, h, click(x0, y0))
	mouseAt(t, h, tea.MouseMotionMsg{X: x1, Y: y1, Button: tea.MouseLeft})
	if _, ok := mouseAt(t, h, tea.MouseReleaseMsg{X: x1, Y: y1, Button: tea.MouseLeft}); !ok {
		t.Fatal("the release copied nothing")
	}
}

// TestMouseSelection is proposal step 3, item 6 (test 12): a selection made
// with the mouse in zone B is copied on release from the conversation
// itself, never from the drawing.
func TestMouseSelection(t *testing.T) {
	t.Run("wrapped line copies as one line", func(t *testing.T) {
		r, clip, _ := selectRig(t)
		h := r.h
		x0, y0 := at(t, h, "alpha", 0)
		x1, y1 := at(t, h, "END", 0)
		if y1 <= y0 {
			t.Fatalf("the long line does not wrap: rows %d and %d", y0, y1)
		}
		dragCopy(t, h, x0, y0, x1+2, y1)
		if clip.last() != longLine {
			t.Errorf("copied %q, want %q", clip.last(), longLine)
		}
	})
	t.Run("triple click on a wrapped row selects the logical line", func(t *testing.T) {
		r, clip, _ := selectRig(t)
		h := r.h
		x, y := at(t, h, "END", 0)
		for i := 0; i < 3; i++ {
			mouseAt(t, h, click(x, y))
			mouseAt(t, h, tea.MouseReleaseMsg{X: x, Y: y, Button: tea.MouseLeft})
		}
		if clip.last() != longLine {
			t.Errorf("copied %q, want %q", clip.last(), longLine)
		}
	})
	t.Run("double click selects a word", func(t *testing.T) {
		r, clip, _ := selectRig(t)
		h := r.h
		x, y := at(t, h, "gamma", 1)
		for i := 0; i < 2; i++ {
			mouseAt(t, h, click(x+2, y))
			mouseAt(t, h, tea.MouseReleaseMsg{X: x + 2, Y: y, Button: tea.MouseLeft})
		}
		if clip.last() != "gamma" {
			t.Errorf("copied %q, want %q", clip.last(), "gamma")
		}
	})
	t.Run("code block exact, tab included", func(t *testing.T) {
		r, clip, now := selectRig(t)
		h := r.h
		x0, y0 := at(t, h, "func f()", 0)
		_, y1 := at(t, h, "return 1", 0)
		dragCopy(t, h, x0, y0, x0+1, y1+1) // past the "}"
		if want := "func f() {\n\treturn 1\n}"; clip.last() != want {
			t.Errorf("copied %q, want %q", clip.last(), want)
		}
		// A triple click on the indented line keeps its tab.
		x, y := at(t, h, "return 1", 0)
		*now = testNow.Add(time.Minute)
		for i := 0; i < 3; i++ {
			mouseAt(t, h, click(x, y))
			mouseAt(t, h, tea.MouseReleaseMsg{X: x, Y: y, Button: tea.MouseLeft})
		}
		if want := "\treturn 1"; clip.last() != want {
			t.Errorf("copied %q, want %q", clip.last(), want)
		}
	})
	t.Run("no borders, symbols or labels; clamped to the panel", func(t *testing.T) {
		r, clip, _ := selectRig(t)
		h := r.h
		x0, y0 := at(t, h, "intro", 0)
		// Start left of the text, on the border, and end far below, in zone C.
		_, yc := at(t, h, "C · AGENTS", 0)
		dragCopy(t, h, x0-3, y0, 200, yc+3)
		want := answer + "\n\nBash"
		if clip.last() != want {
			t.Errorf("copied\n%q\nwant\n%q", clip.last(), want)
		}
		for _, bad := range []string{"│", "▸", "╭", "─", "observed", "CODEX", "AGENTS"} {
			if strings.Contains(clip.last(), bad) {
				t.Errorf("copied text holds %q", bad)
			}
		}
	})
	t.Run("selection is highlighted and the frame keeps its text", func(t *testing.T) {
		r, _, _ := selectRig(t)
		h := r.h
		body := func(f string) string { return f[:strings.LastIndexByte(f, '\n')] } // the toast aside
		styled := h.m.View().Content
		before := body(plainText(styled))
		x0, y0 := at(t, h, "intro", 0)
		dragCopy(t, h, x0, y0, x0+4, y0)
		after := h.m.View().Content
		if body(plainText(after)) != before {
			t.Error("the selection changed the frame's text")
		}
		if body(after) == body(styled) {
			t.Error("the selection is not highlighted")
		}
		h.send(key(tea.KeyEscape, 0))
		if !h.m.ui.Sel.Empty() || body(h.m.View().Content) != body(styled) {
			t.Error("Esc did not clear the selection")
		}
	})
}

// TestCtrlCAfterMouseSelection: Ctrl+C with a mouse selection copies it and
// clears it, as Windows Terminal does; the next Ctrl+C interrupts. Ctrl+Shift+C
// never interrupts.
func TestCtrlCAfterMouseSelection(t *testing.T) {
	r, clip, _ := selectRig(t)
	h := r.h
	h.send(tea.PasteMsg{Content: "go on"})
	h.send(key(tea.KeyEnter, 0))
	r.emit(event.TurnStarted, nil) // a turn runs
	h.settle()
	x0, y0 := at(t, h, "intro", 0)
	dragCopy(t, h, x0, y0, x0+4, y0)
	clip.writes = nil
	if _, ok := press(t, h, key('c', tea.ModCtrl)); !ok || clip.last() != "intro" {
		t.Fatalf("Ctrl+C copied %q, want %q", clip.last(), "intro")
	}
	if _, _, n := r.script.calls(); n != 0 {
		t.Fatalf("Ctrl+C with a selection interrupted (%d)", n)
	}
	if !h.m.ui.Sel.Empty() {
		t.Fatal("the selection is still there after Ctrl+C")
	}
	press(t, h, key('c', tea.ModCtrl|tea.ModShift))
	if _, _, n := r.script.calls(); n != 0 {
		t.Fatalf("Ctrl+Shift+C interrupted (%d)", n)
	}
	press(t, h, key('c', tea.ModCtrl))
	if _, _, n := r.script.calls(); n != 1 {
		t.Fatalf("the second Ctrl+C sent %d interrupts, want 1", n)
	}
}

// TestEditorMouse: a double click in the editor selects a word of the draft
// with the editor's own selection, copied on release; the draft is unchanged.
func TestEditorMouse(t *testing.T) {
	r, clip, now := selectRig(t)
	h := r.h
	const draft = "hello world\n\tsecond line"
	h.send(tea.PasteMsg{Content: draft})
	x, y := at(t, h, "world", 0)
	for i := 0; i < 2; i++ {
		mouseAt(t, h, click(x+1, y))
		mouseAt(t, h, tea.MouseReleaseMsg{X: x + 1, Y: y, Button: tea.MouseLeft})
	}
	if got := h.m.edit.SelectedText(); got != "world" {
		t.Errorf("editor selection %q, want %q", got, "world")
	}
	if clip.last() != "world" {
		t.Errorf("copied %q, want %q", clip.last(), "world")
	}
	if !h.m.ui.Sel.Empty() {
		t.Error("a mouse selection was made outside the editor")
	}
	// A drag from "hello" to the end of the second line.
	*now = testNow.Add(time.Minute)
	x0, y0 := at(t, h, "hello", 0)
	x1, y1 := at(t, h, "second line", 0)
	dragCopy(t, h, x0, y0, x1+40, y1)
	if clip.last() != draft {
		t.Errorf("copied %q, want %q", clip.last(), draft)
	}
	if h.m.edit.Text() != draft {
		t.Errorf("the draft changed to %q", h.m.edit.Text())
	}
}

// TestMouseFocusAndCapture: a click focuses the panel under it; F12 gives
// the mouse back to the terminal and says so.
func TestMouseFocusAndCapture(t *testing.T) {
	r, clip, _ := selectRig(t)
	h := r.h
	x, y := at(t, h, "no agents yet", 0)
	mouseAt(t, h, click(x, y))
	mouseAt(t, h, tea.MouseReleaseMsg{X: x, Y: y, Button: tea.MouseLeft})
	if len(clip.writes) != 0 {
		t.Errorf("a plain click copied %q", clip.writes)
	}
	if h.m.ui.Focus != "C" {
		t.Errorf("focus %q after a click in C", h.m.ui.Focus)
	}
	if v := h.m.View(); v.MouseMode != tea.MouseModeCellMotion {
		t.Fatalf("mouse mode %v, want cell motion", v.MouseMode)
	}
	h.send(key(tea.KeyF12, 0))
	v := h.m.View()
	if v.MouseMode != tea.MouseModeNone {
		t.Errorf("mouse mode %v after F12, want none", v.MouseMode)
	}
	if f := plainText(v.Content); !strings.Contains(f, "mouse capture off") {
		t.Errorf("F12 not shown:\n%s", f)
	}
	h.send(key(tea.KeyF12, 0))
	if v := h.m.View(); v.MouseMode != tea.MouseModeCellMotion {
		t.Errorf("mouse mode %v after F12 twice", v.MouseMode)
	}
}

// TestConversationScroll: the wheel and PgUp hold the view; a new entry
// while held shows "↓ 1 new" and does not move it; PgDn to the end follows
// again. A long sent message folds.
func TestConversationScroll(t *testing.T) {
	r, _, _ := selectRig(t)
	h := r.h
	var b strings.Builder
	for i := 0; i < 60; i++ {
		b.WriteString("row ")
		b.WriteString(strings.Repeat("x", i%7))
		b.WriteString("\n")
	}
	b.WriteString("the very end")
	r.emit(event.MessageCompleted, event.Completed{MessageID: "m2", Text: b.String()})
	h.settle()
	if f := plainText(h.m.View().Content); !strings.Contains(f, "the very end") || strings.Contains(f, "intro line") {
		t.Fatalf("not following the end:\n%s", f)
	}
	x, y := at(t, h, "the very end", 0)
	for i := 0; i < 30; i++ {
		mouseAt(t, h, tea.MouseWheelMsg{X: x, Y: y - 3, Button: tea.MouseWheelUp})
	}
	f := plainText(h.m.View().Content)
	if !strings.Contains(f, "intro line") || strings.Contains(f, "the very end") {
		t.Fatalf("the wheel did not scroll to the top:\n%s", f)
	}
	r.emit(event.MessageCompleted, event.Completed{MessageID: "m3", Text: "a newer answer"})
	h.settle()
	f = plainText(h.m.View().Content)
	if !strings.Contains(f, "↓ 1 new") || !strings.Contains(f, "intro line") {
		t.Fatalf("a held view moved or did not count the new entry:\n%s", f)
	}
	for i := 0; i < 10 && h.m.ui.ConvHold; i++ {
		h.send(key(tea.KeyPgDown, 0))
	}
	f = plainText(h.m.View().Content)
	if h.m.ui.ConvHold || !strings.Contains(f, "a newer answer") || strings.Contains(f, "new\n") {
		t.Fatalf("PgDn to the end does not follow again:\n%s", f)
	}
	h.send(key(tea.KeyPgUp, tea.ModShift))
	if !h.m.ui.ConvHold {
		t.Error("Shift+PgUp did not hold the view")
	}

	// A long message the owner sends folds to its first lines.
	long := strings.TrimSuffix(strings.Repeat("pasted\n", 30), "\n")
	h.send(tea.PasteMsg{Content: long})
	h.send(key(tea.KeyEnter, 0))
	if h.m.ui.ConvHold {
		t.Error("sending did not follow the end")
	}
	r.emit(event.UserMessage, event.User{Text: long}) // as the CLI echoes it
	h.settle()
	if f := plainText(h.m.View().Content); !strings.Contains(f, "… 25 more lines") {
		t.Errorf("the sent message is not folded:\n%s", f)
	}
}
