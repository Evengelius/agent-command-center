package app

import (
	"strings"
	"testing"
	"testing/synctest"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/Evengelius/agent-command-center/internal/render"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// draftWin opens the draft window with Ctrl+G and checks it is open.
func draftWin(t *testing.T, h *harness) {
	t.Helper()
	h.send(key('g', tea.ModCtrl))
	if !h.m.ui.DraftWin.Open {
		t.Fatal("Ctrl+G did not open the draft window")
	}
}

// fromCol is the row s from the cell column x on.
func fromCol(s string, x int) string {
	for i := range s {
		if lipgloss.Width(s[:i]) >= x {
			return s[i:]
		}
	}
	return ""
}

// TestDraftWindow is amendment 13: Ctrl+G opens the draft in a window over
// the dashboard, placed as the file viewer; it is the same draft, with the
// input box's keys, Ctrl+F full screen, Esc back with the draft kept, Enter
// sending and closing it, and Ctrl+G doing nothing more.
func TestDraftWindow(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r, clip := copyRig(t, nil)
		h := r.h
		h.send(tea.PasteMsg{Content: "first\nsecond"})
		if f := h.frame(); !strings.Contains(f, "Ctrl+G draft") || strings.Contains(f, "Ctrl+G editor") {
			t.Errorf("the footer does not name the draft window:\n%s", f)
		}
		draftWin(t, h)
		x, y, ww, wh := render.FileRect(h.m.ui.Width, h.m.ui.Height, false)
		if ww == h.m.ui.Width || wh == h.m.ui.Height {
			t.Fatalf("the window is %dx%d on a %dx%d screen, want it smaller", ww, wh, h.m.ui.Width, h.m.ui.Height)
		}
		rows := strings.Split(h.frame(), "\n")
		if top := fromCol(rows[y], x); !strings.HasPrefix(top, "╭─ Draft ") || lipgloss.Width(top) < ww || !strings.HasPrefix(fromCol(top, ww-1), "╮") {
			t.Errorf("row %d does not open the window at column %d, %d wide: %q", y, x, ww, rows[y])
		}
		if bottom := fromCol(rows[y+wh-1], x); !strings.HasPrefix(bottom, "╰") || !strings.HasPrefix(fromCol(bottom, ww-1), "╯") {
			t.Errorf("row %d does not close the window: %q", y+wh-1, rows[y+wh-1])
		}
		f := strings.Join(rows, "\n")
		for _, want := range []string{"2 lines", "Enter send · Ctrl+J newline · Ctrl+F full screen", "line 2/2", "> first", "  second"} {
			if !strings.Contains(f, want) {
				t.Errorf("no %q in the window:\n%s", want, f)
			}
		}

		// Ctrl+G again does nothing more; f, ? and z are typed.
		cmds := h.m.key(key('g', tea.ModCtrl))
		if len(cmds) != 0 || !h.m.ui.DraftWin.Open || h.m.edit.Text() != "first\nsecond" {
			t.Errorf("Ctrl+G in the window: %d commands, open %v, draft %q", len(cmds), h.m.ui.DraftWin.Open, h.m.edit.Text())
		}
		for _, k := range typedText("f?z") {
			h.send(k)
		}
		if h.m.edit.Text() != "first\nsecondf?z" || h.m.ui.Keys || h.m.ui.DraftWin.Full {
			t.Errorf("f ? z: draft %q, keys %v, full %v", h.m.edit.Text(), h.m.ui.Keys, h.m.ui.DraftWin.Full)
		}

		// The input box's keys: Ctrl+J, Ctrl+Z, Shift+← selecting, Ctrl+C copying.
		h.send(key('j', tea.ModCtrl))
		h.send(typed('3'))
		if h.m.edit.Text() != "first\nsecondf?z\n3" {
			t.Errorf("Ctrl+J then 3: %q", h.m.edit.Text())
		}
		h.send(key('z', tea.ModCtrl))
		if h.m.edit.Text() == "first\nsecondf?z\n3" {
			t.Errorf("Ctrl+Z in the window undid nothing")
		}
		h.send(key('y', tea.ModCtrl))
		if h.m.edit.Text() != "first\nsecondf?z\n3" {
			t.Errorf("Ctrl+Y in the window: %q", h.m.edit.Text())
		}
		h.send(key(tea.KeyBackspace, 0))
		h.send(key(tea.KeyBackspace, 0))
		h.send(key(tea.KeyEnd, tea.ModCtrl))
		h.send(key(tea.KeyLeft, tea.ModShift))
		if _, ok := press(t, h, key('c', tea.ModCtrl)); !ok || clip.last() != "z" {
			t.Errorf("Ctrl+C on a selection: copied %v, clipboard %q", ok, clip.last())
		}
		if n := len(r.sent()); n != 0 || !h.m.ui.DraftWin.Open {
			t.Errorf("Ctrl+C with a selection: %d messages sent, open %v", n, h.m.ui.DraftWin.Open)
		}
		h.send(key(tea.KeyRight, 0))

		// Ctrl+F: full screen under the title bar (plan 21, item 5.7), and back.
		h.send(key('f', tea.ModCtrl))
		if rows := strings.Split(h.frame(), "\n"); !h.m.ui.DraftWin.Full || !strings.Contains(rows[0], "AGENT COMMAND CENTER") || !strings.HasPrefix(rows[1], "╭─ Draft ") {
			t.Errorf("Ctrl+F: full %v, first rows %q and %q", h.m.ui.DraftWin.Full, rows[0], rows[1])
		}
		h.send(key('f', tea.ModCtrl))
		if h.m.ui.DraftWin.Full {
			t.Error("a second Ctrl+F left the window full screen")
		}

		// F1: the key page over the window, Esc back to it.
		h.send(key(tea.KeyF1, 0))
		if !h.m.ui.Keys || !strings.Contains(h.frame(), "draft window: the draft larger") {
			t.Errorf("F1 in the window: keys %v:\n%s", h.m.ui.Keys, h.frame())
		}
		h.send(key(tea.KeyEscape, 0))
		if h.m.ui.Keys || !h.m.ui.DraftWin.Open || !strings.Contains(h.frame(), "╭─ Draft ") {
			t.Errorf("Esc on the key page: keys %v, window open %v", h.m.ui.Keys, h.m.ui.DraftWin.Open)
		}

		// Esc: back to the input box, the draft kept.
		h.send(key(tea.KeyEscape, 0))
		if h.m.ui.DraftWin.Open || h.m.edit.Text() != "first\nsecondf?z" || h.m.ui.Focus != "B" {
			t.Errorf("Esc: open %v, draft %q, focus %q", h.m.ui.DraftWin.Open, h.m.edit.Text(), h.m.ui.Focus)
		}
		if f := h.frame(); strings.Contains(f, "╭─ Draft") {
			t.Errorf("the window is still drawn after Esc:\n%s", f)
		}

		// F6 closes the window and moves to the next zone, as the footer says.
		draftWin(t, h)
		h.send(key(tea.KeyF6, 0))
		if h.m.ui.DraftWin.Open || h.m.ui.Focus == "B" || h.m.edit.Text() != "first\nsecondf?z" {
			t.Errorf("F6 in the window: open %v, focus %q, draft %q", h.m.ui.DraftWin.Open, h.m.ui.Focus, h.m.edit.Text())
		}

		h.send(key(tea.KeyF6, tea.ModShift))
		if h.m.ui.Focus != "B" {
			t.Fatalf("Shift+F6 back: focus %q", h.m.ui.Focus)
		}

		// Enter on an empty draft sends nothing and keeps the window.
		r2, _ := copyRig(t, nil)
		draftWin(t, r2.h)
		r2.h.send(key(tea.KeyEnter, 0))
		if n := len(r2.sent()); n != 0 || !r2.h.m.ui.DraftWin.Open {
			t.Errorf("Enter on an empty draft: %d messages, open %v", n, r2.h.m.ui.DraftWin.Open)
		}

		// Enter sends the draft and closes the window.
		draftWin(t, h)
		h.send(key(tea.KeyEnter, 0))
		if sent := r.sent(); len(sent) != 1 || sent[0] != "first\nsecondf?z" {
			t.Errorf("Enter in the window sent %q", sent)
		}
		if h.m.ui.DraftWin.Open || h.m.edit.Text() != "" || h.m.State().Turn.Phase == state.Idle {
			t.Errorf("after sending: open %v, draft %q, turn %q", h.m.ui.DraftWin.Open, h.m.edit.Text(), h.m.State().Turn.Phase)
		}
		r.end()
	})
}

// TestDraftWindowByteForByte is amendment 13 with the input box's guarantee
// (amendment 3): a 10,000-line draft pasted and typed in the window is the
// draft byte for byte, after Esc and as sent.
func TestDraftWindowByteForByte(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r, _ := copyRig(t, nil)
		h := r.h
		draftWin(t, h)
		want := ""
		for _, c := range pasteCases {
			h.send(tea.PasteMsg{Content: c.in})
			want += c.want
		}
		h.send(tea.PasteMsg{Content: draftLines(10_000)})
		want += draftLines(10_000)
		for _, k := range typedText("typed: · 日本 👍🏽 end") {
			h.send(k)
		}
		h.send(key(tea.KeyTab, 0))
		h.send(key('j', tea.ModCtrl))
		h.send(typed('x'))
		want += "typed: · 日本 👍🏽 end\t\nx"
		if n := h.m.edit.Lines(); n < 10_000 {
			t.Fatalf("%d lines in the draft", n)
		}
		if got := h.m.edit.Text(); got != want {
			t.Fatalf("draft in the window differs: %d bytes, want %d", len(got), len(want))
		}
		if f := h.frame(); !strings.Contains(f, "╭─ Draft ") || !strings.Contains(f, "x") {
			t.Errorf("the window does not show the draft's end:\n%s", f)
		}
		h.send(key(tea.KeyEscape, 0))
		if got := h.m.edit.Text(); got != want {
			t.Fatalf("draft after Esc differs: %d bytes, want %d", len(got), len(want))
		}
		draftWin(t, h)
		h.send(key(tea.KeyEnter, 0))
		if sent := r.sent(); len(sent) != 1 || sent[0] != want {
			t.Errorf("sent %d messages; the first is not the draft (%d bytes)", len(sent), len(want))
		}
		r.end()
	})
}

// TestDraftWindowMouse is amendment 13 with the mouse: a drag in the window
// selects in the draft and copies on release; a click outside it changes
// nothing; the wheel moves through the draft.
func TestDraftWindowMouse(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r, clip := copyRig(t, nil)
		h := r.h
		h.send(tea.PasteMsg{Content: "alpha beta gamma\n" + draftLines(40)})
		draftWin(t, h)
		h.send(key(tea.KeyHome, tea.ModCtrl))
		x0, y0 := at(t, h, "beta", 0)
		_, wy, _, _ := render.FileRect(h.m.ui.Width, h.m.ui.Height, false)
		if y0 <= wy {
			t.Fatalf("beta on row %d, not inside the window from row %d", y0, wy)
		}
		dragCopy(t, h, x0, y0, x0+3, y0)
		if clip.last() != "beta" || h.m.edit.SelectedText() != "beta" {
			t.Errorf("drag over beta: copied %q, selection %q", clip.last(), h.m.edit.SelectedText())
		}
		focus := h.m.ui.Focus
		mouseAt(t, h, click(0, 0)) // the title bar, outside the window
		if h.m.ui.Focus != focus || !h.m.ui.DraftWin.Open || h.m.edit.SelectedText() != "beta" {
			t.Errorf("a click outside the window: focus %q, open %v, selection %q", h.m.ui.Focus, h.m.ui.DraftWin.Open, h.m.edit.SelectedText())
		}
		h.send(key(tea.KeyHome, tea.ModCtrl))
		mouseAt(t, h, tea.MouseWheelMsg{X: x0, Y: y0, Button: tea.MouseWheelDown})
		if l := h.m.edit.CursorLine(); l != wheelRows {
			t.Errorf("the wheel moved the cursor to line %d, want %d", l, wheelRows)
		}
	})
}
