package app

import (
	"fmt"
	"math/rand/v2"
	"slices"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/Evengelius/agent-command-center/internal/event"
	"github.com/Evengelius/agent-command-center/internal/input"
	"github.com/Evengelius/agent-command-center/internal/input/imagetest"
	"github.com/Evengelius/agent-command-center/internal/render"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// Plan 73: a long message as one row, in zone B and in the composer. These tests
// drive the model with tea.PasteMsg and keys through Update, a real terminal's paste
// being the owner's step (plan.md, Plan 73, Windows Terminal scenes).

// chipRig is a scripted dashboard at 120x40, with a fake clipboard.
func chipRig(t *testing.T) (inputRig, *fakeClipboard) {
	t.Helper()
	r, clip := copyRig(t, nil)
	r.h.send(tea.WindowSizeMsg{Width: 120, Height: 40})
	return r, clip
}

// longLines is n lines, each naming its number after word.
func longLines(n int, word string) string {
	ls := make([]string, n)
	for i := range ls {
		ls[i] = fmt.Sprintf("%s %d", word, i+1)
	}
	return strings.Join(ls, "\n")
}

// typeText types s, Ctrl+J for its line breaks.
func typeText(h *harness, s string) {
	for _, r := range s {
		if r == '\n' {
			h.send(key('j', tea.ModCtrl))
			continue
		}
		h.send(typed(r))
	}
}

// noProjectBar is a frame without its second row, the project bar, which names a
// temporary folder that differs between any two rigs.
func noProjectBar(f string) string {
	rows := strings.Split(f, "\n")
	if len(rows) > 2 {
		rows = slices.Delete(rows, 1, 2)
	}
	return strings.Join(rows, "\n")
}

// notInside fails when the cursor or an end of the selection is inside a range: the
// composer's rule of design D9.
func notInside(t *testing.T, h *harness, what string) {
	t.Helper()
	a, b, sel := h.m.edit.Selection()
	c := h.m.edit.Cursor()
	for _, r := range h.m.edit.Ranges() {
		if c > r.From && c < r.To {
			t.Fatalf("%s: the cursor %d inside %+v", what, c, r)
		}
		if sel && (a > r.From && a < r.To || b > r.From && b < r.To) {
			t.Fatalf("%s: the selection %d..%d cuts %+v", what, a, b, r)
		}
	}
}

// TestLongPaste is criterion 73.12: a block over three rows, judged alone, becomes a
// chip at once and sends nothing; at three rows it stays text; typed text is never
// judged, nor several short pastes together; a raw Ctrl+V and the paths' fallback are
// the same paste; with the draft window open nothing converts.
func TestLongPaste(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r, _ := chipRig(t)
		h := r.h
		w := render.MessageWidth(h.m.State(), h.m.ui)
		x := func(n int) string { return strings.Repeat("x", n) }
		for _, c := range []struct{ name, in, stored, chip string }{
			{"three lines", "one\ntwo\nthree", "one\ntwo\nthree", ""},
			{"a line filling three rows", x(3 * w), x(3 * w), ""},
			{"four lines", "one\ntwo\nthree\nfour", "one\ntwo\nthree\nfour", "▣ Long message · 4 lines · 18 B"},
			{"three lines and a final break", "one\ntwo\nthree\n", "one\ntwo\nthree\n", "▣ Long message · 3 lines · 13 B"},
			{"leading breaks", "\n\n\nabc", "\n\n\nabc", "▣ Long message · 4 lines · 6 B"},
			{"one cell over three rows", x(3*w + 1), x(3*w + 1), fmt.Sprintf("▣ Long message · 1 line · %d B", 3*w+1)},
			{"CRLF stored as LF first", "one\r\ntwo\r\nthree\r\nfour", "one\ntwo\nthree\nfour", "▣ Long message · 4 lines · 18 B"},
			{"lone CRs stored as LF first", "one\rtwo\rthree", "one\ntwo\nthree", ""},
		} {
			h.m.edit.Clear()
			h.send(tea.PasteMsg{Content: c.in})
			f := h.frame()
			if got := h.m.edit.Text(); got != c.stored {
				t.Errorf("%s: stored %q, want %q", c.name, got, c.stored)
			}
			if len(r.sent()) != 0 {
				t.Fatalf("%s: a paste sent %q", c.name, r.sent())
			}
			if c.chip == "" {
				if len(h.m.edit.Ranges()) != 0 || strings.Contains(f, "Long message") {
					t.Errorf("%s: a chip: %+v\n%s", c.name, h.m.edit.Ranges(), f)
				}
				continue
			}
			if len(h.m.edit.Ranges()) != 1 || !strings.Contains(f, c.chip) {
				t.Errorf("%s: ranges %+v, want %q:\n%s", c.name, h.m.edit.Ranges(), c.chip, f)
			}
			if strings.Contains(f, "two") || strings.Contains(f, x(10)) {
				t.Errorf("%s: its text in the frame:\n%s", c.name, f)
			}
		}

		// Two typed lines, then a long paste: they stay visible and editable, the
		// chip after them.
		h.m.edit.Clear()
		typeText(h, "first typed\nsecond typed\n")
		h.send(tea.PasteMsg{Content: longLines(9, "block")})
		f := h.frame()
		for _, want := range []string{"> first typed", "  second typed", "  ▣ Long message · 9 lines · 71 B"} {
			if !strings.Contains(f, want) {
				t.Errorf("typed lines and a chip: no %q:\n%s", want, f)
			}
		}
		h.send(key(tea.KeyUp, 0))
		h.send(key(tea.KeyUp, 0))
		h.send(key(tea.KeyHome, 0))
		typeText(h, "edited ")
		if !strings.HasPrefix(h.m.edit.Text(), "edited first typed\nsecond typed\nblock 1\n") || len(h.m.edit.Ranges()) != 1 {
			t.Errorf("editing a typed line: %q, %+v", h.m.edit.Text()[:40], h.m.edit.Ranges())
		}

		// Ten typed lines, then a two-line paste: judged alone, no chip.
		h.m.edit.Clear()
		typeText(h, longLines(10, "typed"))
		if len(h.m.edit.Ranges()) != 0 {
			t.Errorf("typed lines made %+v", h.m.edit.Ranges())
		}
		h.send(tea.PasteMsg{Content: "\npasted one\npasted two"})
		if len(h.m.edit.Ranges()) != 0 || !strings.Contains(h.frame(), "pasted two") {
			t.Errorf("a short paste after typed lines: %+v\n%s", h.m.edit.Ranges(), h.frame())
		}

		// Several short pastes, many lines in all: none.
		h.m.edit.Clear()
		for i := range 8 {
			h.send(tea.PasteMsg{Content: fmt.Sprintf("short %d\nlines\n", i)})
		}
		if n := h.m.edit.Lines(); n != 17 || len(h.m.edit.Ranges()) != 0 {
			t.Errorf("short pastes: %d lines, %+v", n, h.m.edit.Ranges())
		}

		// The draft window open: nothing converts.
		h.m.edit.Clear()
		draftWin(t, h)
		h.send(tea.PasteMsg{Content: longLines(9, "block")})
		if len(h.m.edit.Ranges()) != 0 || strings.Contains(h.frame(), "Long message") {
			t.Errorf("a paste in the draft window: %+v", h.m.edit.Ranges())
		}
		h.send(key(tea.KeyEscape, 0))
		if f := h.frame(); len(h.m.edit.Ranges()) != 0 || strings.Contains(f, "Long message") || !strings.Contains(f, "block 9") {
			t.Errorf("after Esc: %+v\n%s", h.m.edit.Ranges(), f)
		}
	})
}

// TestLongPasteOtherWays is criterion 73.12's other ways in: the raw Ctrl+V a terminal
// that sends one gives, and the paths' fallback, reach the same paste.
func TestLongPasteOtherWays(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		files, err := imagetest.WriteAll(t.TempDir())
		if err != nil {
			t.Fatal(err)
		}
		r := scriptRig(t, "claude-code")
		h := r.h
		h.send(tea.WindowSizeMsg{Width: 120, Height: 40})
		c := &clip{}
		h.m.opt.Clipboard = c
		text := longLines(9, "raw")
		c.set(text, nil)
		h.send(key('v', tea.ModCtrl))
		h.send(ctrlVRelease())
		if h.m.edit.Text() != text || len(h.m.edit.Ranges()) != 1 || !strings.Contains(h.frame(), "▣ Long message · 9 lines") {
			t.Errorf("a raw Ctrl+V: %q, %+v\n%s", h.m.edit.Text(), h.m.edit.Ranges(), h.frame())
		}
		// One of the paths not an image: they go in as text, a long line here.
		h.m.edit.Clear()
		list := []string{files["gif"], files["text"], files["png"], files["jpg"]}
		c.set("", nil, list...)
		paths := strings.Join(list, " ")
		h.send(tea.PasteMsg{Content: paths})
		if len(h.m.attach) != 0 || h.m.edit.Text() != paths {
			t.Fatalf("a non-image among them: %d attachments, draft %q", len(h.m.attach), h.m.edit.Text())
		}
		if !render.LongMessage(paths, render.MessageWidth(h.m.State(), h.m.ui)) {
			t.Fatalf("%d cells of paths are not long here", len(paths))
		}
		if len(h.m.edit.Ranges()) != 1 || !strings.Contains(h.frame(), "▣ Long message · 1 line") {
			t.Errorf("the paths' fallback: %+v\n%s", h.m.edit.Ranges(), h.frame())
		}
	})
}

// TestChipEdges is criterion 73.13: one final line break stays below the chip with the
// cursor on it, typed text goes where the bytes say, a block between typed text leaves
// both sides intact, and the row's count is the message's, none for a chip alone.
func TestChipEdges(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r, _ := chipRig(t)
		h := r.h
		h.send(tea.PasteMsg{Content: "a\nb\nc\n"})
		f := h.frame()
		v := h.m.ui.Editor
		if !strings.Contains(f, "> ▣ Long message · 3 lines · 5 B") || v.Line != 1 || v.Col != 0 || v.Total != 2 {
			t.Errorf("a final break: line %d col %d of %d:\n%s", v.Line, v.Col, v.Total, f)
		}
		typeText(h, "x")
		if h.m.edit.Text() != "a\nb\nc\nx" || !rowWith(h.frame(), "  x") {
			t.Errorf("typed below it: %q\n%s", h.m.edit.Text(), h.frame())
		}
		h.send(key(tea.KeyBackspace, 0))
		h.send(key(tea.KeyEnter, 0))
		if s := r.sent(); len(s) != 1 || s[0] != "a\nb\nc\n" {
			t.Errorf("sent %q", s)
		}
		r.emit(event.UserMessage, event.User{Text: r.sent()[0]})
		r.end()
		if f := h.frame(); !strings.Contains(f, "▣ Long message · 4 lines · 6 B") {
			t.Errorf("once sent, want 4 lines · 6 B:\n%s", f)
		}

		// No final break: typed text joins the block's last line in the bytes, and shows
		// on the chip's row; no byte added; Ctrl+J starts a new line.
		block := longLines(4, "line")
		h.send(tea.PasteMsg{Content: block})
		typeText(h, " tail")
		if h.m.edit.Text() != block+" tail" || !strings.Contains(h.frame(), "> ▣ Long message · 4 lines · 27 B tail") {
			t.Errorf("typed on its row: %q\n%s", h.m.edit.Text(), h.frame())
		}
		typeText(h, "\nnext")
		if h.m.edit.Text() != block+" tail\nnext" || !rowWith(h.frame(), "  next") {
			t.Errorf("Ctrl+J: %q\n%s", h.m.edit.Text(), h.frame())
		}

		// Pasted between typed text: both sides intact, either edge outside the range.
		h.m.edit.Clear()
		typeText(h, "before  after")
		for range len(" after") {
			h.send(key(tea.KeyLeft, 0))
		}
		h.send(tea.PasteMsg{Content: block})
		if h.m.edit.Text() != "before "+block+" after" || !strings.Contains(h.frame(), "> before ▣ Long message · 4 lines · 27 B after") {
			t.Errorf("between typed text: %q\n%s", h.m.edit.Text(), h.frame())
		}
		if got, want := h.m.edit.Ranges(), []input.Range{{From: 7, To: 7 + len(block), Lines: 4}}; !slices.Equal(got, want) {
			t.Errorf("its range %+v, want %+v", got, want)
		}

		// The row's count.
		h.m.edit.Clear()
		h.send(tea.PasteMsg{Content: longLines(200, "x")})
		if f := h.frame(); strings.Count(f, "200 lines") != 1 || strings.Contains(f, "201 lines") {
			t.Errorf("a chip alone shows only its own count:\n%s", f)
		}
		typeText(h, "\ntyped")
		if f := h.frame(); !strings.Contains(f, "201 lines") || strings.Count(f, "200 lines") != 1 {
			t.Errorf("a typed line beside it: want 201 lines:\n%s", f)
		}
	})
}

// TestTwoChips is criterion 73.14, on both adapters: two long pastes are two chips,
// with typed text before, after and between them, visible and editable; each chip's
// words its own; deleting and undoing one leaves the other and the text byte-identical;
// sent, the message is those blocks and that text in that order.
func TestTwoChips(t *testing.T) {
	for name, rig := range map[string]func(*testing.T) inputRig{"claude-code": claudeRig, "codex": codexRig} {
		t.Run(name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				r := rig(t)
				h := r.h
				a, b := longLines(5, "aaa"), longLines(7, "bbb")
				typeText(h, "intro ")
				h.send(tea.PasteMsg{Content: a})
				typeText(h, " between ")
				h.send(tea.PasteMsg{Content: b})
				typeText(h, " outro")
				want := "intro " + a + " between " + b + " outro"
				if h.m.edit.Text() != want || len(h.m.edit.Ranges()) != 2 {
					t.Fatalf("%q, %+v", h.m.edit.Text(), h.m.edit.Ranges())
				}
				f := h.frame()
				for _, w := range []string{"> intro ▣ Long message · 5 lines · 29 B", "between", "▣ Long message · 7 lines · 41 B", "outro"} {
					if !strings.Contains(f, w) {
						t.Errorf("no %q:\n%s", w, f)
					}
				}
				for range len(" outro") {
					h.send(key(tea.KeyLeft, 0))
				}
				h.send(key(tea.KeyBackspace, 0))
				if got := h.m.edit.Text(); got != "intro "+a+" between  outro" || len(h.m.edit.Ranges()) != 1 {
					t.Errorf("the second deleted: %q, %+v", got, h.m.edit.Ranges())
				}
				if f := h.frame(); strings.Contains(f, "7 lines") || !strings.Contains(f, "5 lines · 29 B") {
					t.Errorf("the second deleted, the frame:\n%s", f)
				}
				h.send(key('z', tea.ModCtrl))
				if h.m.edit.Text() != want || len(h.m.edit.Ranges()) != 2 {
					t.Errorf("undone: %q, %+v", h.m.edit.Text(), h.m.edit.Ranges())
				}
				// The typed text between them: editable, both chips kept.
				h.send(key(tea.KeyLeft, 0)) // over the second chip
				h.send(key(tea.KeyLeft, 0)) // the space before it
				typeText(h, "X")
				if got, w2 := h.m.edit.Text(), "intro "+a+" betweenX "+b+" outro"; got != w2 || len(h.m.edit.Ranges()) != 2 {
					t.Errorf("edited between: %q, %+v", got, h.m.edit.Ranges())
				}
				h.send(key('z', tea.ModCtrl))
				h.send(key(tea.KeyEnter, 0))
				r.end()
				if s := r.sent(); len(s) != 1 || s[0] != want {
					t.Fatalf("sent %q, want %q", s, want)
				}
				if r.raw != nil {
					r.raw(t, 0, want)
				}
			})
		})
	}
}

// TestChipCursor is criterion 73.15 and V3: every motion, its Shift form and a click,
// at random: the cursor never inside a range, a selection holding a chip holds it whole.
func TestChipCursor(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r, _ := chipRig(t)
		h := r.h
		now := testNow
		h.m.opt.Now = func() time.Time { return now }
		typeText(h, "typed one\nab ")
		h.send(tea.PasteMsg{Content: longLines(6, "aaa")})
		h.send(tea.PasteMsg{Content: longLines(5, "bbb")}) // adjacent to it
		typeText(h, " cd\nlast ")
		h.send(tea.PasteMsg{Content: longLines(4, "ccc") + "\n"})
		typeText(h, "end")
		text := h.m.edit.Text()
		if len(h.m.edit.Ranges()) != 3 {
			t.Fatalf("%+v", h.m.edit.Ranges())
		}
		codes := []rune{tea.KeyLeft, tea.KeyRight, tea.KeyUp, tea.KeyDown, tea.KeyHome, tea.KeyEnd}
		mods := []tea.KeyMod{0, tea.ModShift, tea.ModCtrl, tea.ModCtrl | tea.ModShift, tea.ModAlt}
		rng := rand.New(rand.NewPCG(73, 15))
		x0, y0 := at(t, h, "typed one", 0)
		for i := range 1500 {
			switch rng.IntN(6) {
			case 0:
				// A click, with or without Shift, on the composer's rows.
				now = now.Add(time.Second)
				m := tea.KeyMod(0)
				if rng.IntN(3) == 0 {
					m = tea.ModShift
				}
				x, y := x0-2+rng.IntN(72), y0+rng.IntN(5)
				mouseAt(t, h, tea.MouseClickMsg{X: x, Y: y, Button: tea.MouseLeft, Mod: m})
				mouseAt(t, h, tea.MouseReleaseMsg{X: x, Y: y, Button: tea.MouseLeft})
				h.m.ui.Focus = "B"
			default:
				h.send(key(codes[rng.IntN(len(codes))], mods[rng.IntN(len(mods))]))
			}
			notInside(t, h, fmt.Sprintf("step %d", i))
			if h.m.edit.Text() != text {
				t.Fatalf("step %d changed the text to %q", i, h.m.edit.Text())
			}
		}
	})
}

// TestChipClicks is V3 on and beside a chip: a click puts the cursor at one of its
// edges, a double click selects it whole, a triple click its line, a drag copies the
// bytes it holds, a Shift+click extends past it; none inside it.
func TestChipClicks(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r, clip := chipRig(t)
		h := r.h
		now := testNow
		h.m.opt.Now = func() time.Time { return now }
		block := longLines(6, "aaa")
		typeText(h, "before ")
		h.send(tea.PasteMsg{Content: block})
		typeText(h, " after")
		text := h.m.edit.Text()
		from, to := len("before "), len("before ")+len(block)

		x, y := at(t, h, "▣ Long message", 0)
		for _, off := range []int{0, 5, 20} {
			now = now.Add(time.Second)
			mouseClick(t, h, x+off, y)
			if c := h.m.edit.Cursor(); c != from && c != to {
				t.Errorf("a click %d cells into the chip: the cursor %d, want %d or %d", off, c, from, to)
			}
		}
		// Beside it.
		now = now.Add(time.Second)
		bx, by := at(t, h, "before", 0)
		mouseClick(t, h, bx+1, by)
		if c := h.m.edit.Cursor(); c != 1 {
			t.Errorf("a click on before's second cell: the cursor %d", c)
		}
		now = now.Add(time.Second)
		ax, ay := at(t, h, "after", 0)
		mouseClick(t, h, ax+1, ay)
		if c := h.m.edit.Cursor(); c != to+2 {
			t.Errorf("a click on after's second cell: the cursor %d, want %d", c, to+2)
		}
		// A double click: the chip whole, its bytes copied.
		now = now.Add(time.Second)
		mouseClick(t, h, x+3, y)
		mouseClick(t, h, x+3, y)
		if got := h.m.edit.SelectedText(); got != block || clip.last() != block {
			t.Errorf("a double click selected %q, copied %q", got, clip.last())
		}
		// A triple click: the line the compact form holds, all its bytes.
		now = now.Add(time.Second)
		for range 3 {
			mouseClick(t, h, x+3, y)
		}
		if got := h.m.edit.SelectedText(); got != text || clip.last() != text {
			t.Errorf("a triple click selected %q, copied %q", got, clip.last())
		}
		// A drag from before to after: every byte between.
		now = now.Add(time.Second)
		dragCopy(t, h, bx, by, ax+5, ay)
		if clip.last() != text {
			t.Errorf("a drag over the chip copied %q", clip.last())
		}
		// A click, then Shift+click past the chip: it holds it whole.
		now = now.Add(time.Second)
		mouseClick(t, h, bx, by)
		now = now.Add(time.Second)
		mouseAt(t, h, tea.MouseClickMsg{X: ax + 2, Y: ay, Button: tea.MouseLeft, Mod: tea.ModShift})
		mouseAt(t, h, tea.MouseReleaseMsg{X: ax + 2, Y: ay, Button: tea.MouseLeft})
		notInside(t, h, "Shift+click")
		if h.m.edit.Text() != text {
			t.Errorf("the clicks changed the text: %q", h.m.edit.Text())
		}
	})
}

// TestChipDeleteKeys is criterion 73.16, through the keys: whole-block deletion from
// each edge by each of its four keys; a word deletion stopping at an edge; Ctrl+Z and
// Ctrl+Y restoring bytes, chip and cursor; a selection holding a chip deleted, typed
// over and cut; several and adjacent chips; unrelated text byte-identical in all of them.
func TestChipDeleteKeys(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r, clip := chipRig(t)
		h := r.h
		a, b := longLines(5, "aaa"), longLines(4, "bbb")
		start := func() string {
			h.m.edit.Clear()
			typeText(h, "typed ")
			h.send(tea.PasteMsg{Content: a})
			typeText(h, " mid ")
			h.send(tea.PasteMsg{Content: b})
			h.send(tea.PasteMsg{Content: a}) // adjacent to b
			typeText(h, " end")
			return h.m.edit.Text()
		}
		text := start()
		first := len("typed ")
		firstEnd := first + len(a)
		place := func(p int) { h.m.edit.Select(p, p) }
		for _, c := range []struct {
			name string
			at   int
			k    tea.KeyPressMsg
			want string
		}{
			{"Backspace right after", firstEnd, key(tea.KeyBackspace, 0), "typed " + " mid " + b + a + " end"},
			{"Ctrl+Backspace right after", firstEnd, key(tea.KeyBackspace, tea.ModCtrl), "typed " + " mid " + b + a + " end"},
			{"Delete right before", first, key(tea.KeyDelete, 0), "typed " + " mid " + b + a + " end"},
			{"Ctrl+Delete right before", first, key(tea.KeyDelete, tea.ModCtrl), "typed " + " mid " + b + a + " end"},
			{"the adjacent one's Backspace", firstEnd + len(" mid ") + len(b), key(tea.KeyBackspace, 0), "typed " + a + " mid " + a + " end"},
			{"the adjacent one's Delete", firstEnd + len(" mid ") + len(b), key(tea.KeyDelete, 0), "typed " + a + " mid " + b + " end"},
			{"a word deletion before the chip's word", firstEnd + len(" mid"), key(tea.KeyBackspace, tea.ModCtrl), "typed " + a + "  " + b + a + " end"},
			{"a word deletion stopping at its end", firstEnd + 1, key(tea.KeyBackspace, tea.ModCtrl), "typed " + a + "mid " + b + a + " end"},
			{"a word deletion stopping at its start", first - 1, key(tea.KeyDelete, tea.ModCtrl), "typed" + a + " mid " + b + a + " end"},
		} {
			place(c.at)
			ranges := len(h.m.edit.Ranges())
			h.send(c.k)
			if got := h.m.edit.Text(); got != c.want {
				t.Errorf("%s: %q, want %q", c.name, got, c.want)
			}
			if got := h.m.edit.Text(); got == text {
				t.Errorf("%s: nothing deleted", c.name)
			}
			after := h.m.edit.Text()
			h.send(key('z', tea.ModCtrl))
			if h.m.edit.Text() != text || len(h.m.edit.Ranges()) != ranges || h.m.edit.Cursor() != c.at {
				t.Errorf("%s, Ctrl+Z: %q, %+v, cursor %d", c.name, h.m.edit.Text(), h.m.edit.Ranges(), h.m.edit.Cursor())
			}
			h.send(key('y', tea.ModCtrl))
			if h.m.edit.Text() != after {
				t.Errorf("%s, Ctrl+Y: %q", c.name, h.m.edit.Text())
			}
			h.send(key('z', tea.ModCtrl))
		}

		// A selection holding a chip, with typed text on either side: copied and cut as
		// its original bytes, deleted, typed over.
		sel := func() string {
			h.m.edit.Select(first-3, firstEnd+3)
			return text[first-3 : firstEnd+3]
		}
		want := sel()
		if _, ok := press(t, h, key('c', tea.ModCtrl)); !ok || clip.last() != want {
			t.Errorf("Ctrl+C copied %q, want %q", clip.last(), want)
		}
		if h.m.edit.Text() != text || len(h.m.edit.Ranges()) != 3 {
			t.Errorf("Ctrl+C changed the draft: %q", h.m.edit.Text())
		}
		sel()
		if _, ok := press(t, h, key('x', tea.ModCtrl)); !ok || clip.last() != want {
			t.Errorf("Ctrl+X copied %q, want %q", clip.last(), want)
		}
		if got := h.m.edit.Text(); got != text[:first-3]+text[firstEnd+3:] || len(h.m.edit.Ranges()) != 2 {
			t.Errorf("Ctrl+X left %q, %+v", got, h.m.edit.Ranges())
		}
		h.send(key('z', tea.ModCtrl))
		if h.m.edit.Text() != text || len(h.m.edit.Ranges()) != 3 {
			t.Errorf("Ctrl+X undone: %q, %+v", h.m.edit.Text(), h.m.edit.Ranges())
		}
		sel()
		h.send(key(tea.KeyBackspace, 0))
		if got := h.m.edit.Text(); got != text[:first-3]+text[firstEnd+3:] {
			t.Errorf("a selection deleted: %q", got)
		}
		h.send(key('z', tea.ModCtrl))
		sel()
		h.send(typed('Q'))
		if got := h.m.edit.Text(); got != text[:first-3]+"Q"+text[firstEnd+3:] || len(h.m.edit.Ranges()) != 2 {
			t.Errorf("a selection typed over: %q, %+v", got, h.m.edit.Ranges())
		}
		h.send(key('z', tea.ModCtrl))
		if h.m.edit.Text() != text || len(h.m.edit.Ranges()) != 3 {
			t.Errorf("typed over, undone: %q, %+v", h.m.edit.Text(), h.m.edit.Ranges())
		}
		// A drag's copy on release, from the typed text before it to the typed text after.
		place(len(text))
		h.m.View()
		x0, y0 := at(t, h, "typed", 0)
		x1, y1 := at(t, h, "mid", 0)
		dragCopy(t, h, x0+3, y0, x1, y1)
		if want := text[3 : firstEnd+2]; clip.last() != want {
			t.Errorf("a drag's copy %q, want %q", clip.last(), want)
		}
	})
}

// TestChipEnter is criterion 73.17: a paste ending in \: Enter sends, bytes unchanged;
// a typed \ after a chip then Enter: a newline, as today; in the draft window as today;
// during a paste's open markers no key sends; a turn running: the chip and text kept,
// the notice as today.
func TestChipEnter(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r, _ := chipRig(t)
		h := r.h
		block := longLines(5, "line") + `\`
		h.send(tea.PasteMsg{Content: block})
		h.send(key(tea.KeyEnter, 0))
		if s := r.sent(); len(s) != 1 || s[0] != block {
			t.Fatalf("a paste ending in a backslash: sent %q", s)
		}
		r.end()

		// A typed \ after a chip: the rule applies to what the owner can see.
		plain := longLines(5, "line")
		h.send(tea.PasteMsg{Content: plain})
		typeText(h, `\`)
		h.send(key(tea.KeyEnter, 0))
		if h.m.edit.Text() != plain+"\n" || len(r.sent()) != 1 || len(h.m.edit.Ranges()) != 1 {
			t.Errorf("a typed backslash: %q, %d sent, %+v", h.m.edit.Text(), len(r.sent()), h.m.edit.Ranges())
		}

		// In the draft window, the same paste's \ is as today's: a newline.
		h.m.edit.Clear()
		h.send(tea.PasteMsg{Content: block})
		draftWin(t, h)
		h.send(key(tea.KeyEnter, 0))
		if h.m.edit.Text() != plain+"\n" || len(r.sent()) != 1 {
			t.Errorf("the draft window: %q, %d sent", h.m.edit.Text(), len(r.sent()))
		}
		h.send(key(tea.KeyEscape, 0))

		// During a paste's open markers no key sends.
		h.m.edit.Clear()
		h.send(tea.PasteStartMsg{})
		h.send(tea.PasteMsg{Content: plain})
		h.send(key(tea.KeyEnter, 0))
		if len(r.sent()) != 1 || !strings.Contains(h.frame(), "pasting…") || h.m.edit.Text() != plain+"\n" {
			t.Errorf("open markers: %d sent, %q:\n%s", len(r.sent()), h.m.edit.Text(), h.frame())
		}
		h.send(tea.PasteEndMsg{})
		h.m.edit.Clear()

		// A turn running: the draft kept, as today, its chip with it.
		r.emit(event.UserMessage, event.User{Text: "a running turn"})
		r.emit(event.TurnStarted, nil)
		h.settle()
		h.send(tea.PasteMsg{Content: plain})
		h.send(key(tea.KeyEnter, 0))
		f := h.frame()
		if h.m.edit.Text() != plain || len(h.m.edit.Ranges()) != 1 || !strings.Contains(f, "▣ Long message · 5 lines") || h.m.State().Turn.Notice != "turn running: wait, or interrupt" {
			t.Errorf("a turn running: %q, %+v, notice %q:\n%s", h.m.edit.Text(), h.m.edit.Ranges(), h.m.State().Turn.Notice, f)
		}
		if len(r.sent()) != 1 {
			t.Errorf("a turn running: %d sent", len(r.sent()))
		}
	})
}

// TestChipDraft is criterion 73.18: the draft window draws and takes keys the same with
// and without ranges; no change then Esc: chips back, the cursor where it was on
// reopening; an edit: ranges gone, text ordinary, nothing lost; undo to the original
// bytes: chips back; an invalid range: dropped, the text intact.
func TestChipDraft(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		text := longLines(30, "alpha beta")
		a, clip := chipRig(t) // the chip's rig
		b, _ := chipRig(t)    // the ordinary draft's
		a.h.send(tea.PasteMsg{Content: text})
		fillOrdinary(b.h, text)
		if len(a.h.m.edit.Ranges()) != 1 || len(b.h.m.edit.Ranges()) != 0 {
			t.Fatalf("ranges %+v and %+v", a.h.m.edit.Ranges(), b.h.m.edit.Ranges())
		}
		both := func(what string, msg tea.Msg) {
			t.Helper()
			a.h.send(msg)
			b.h.send(msg)
			if fa, fb := noProjectBar(a.h.frame()), noProjectBar(b.h.frame()); fa != fb {
				t.Errorf("%s: the frames differ:\n%s\n---\n%s", what, fa, fb)
			}
			if a.h.m.edit.Cursor() != b.h.m.edit.Cursor() || a.h.m.edit.Text() != b.h.m.edit.Text() {
				t.Errorf("%s: cursor %d and %d", what, a.h.m.edit.Cursor(), b.h.m.edit.Cursor())
			}
		}
		both("Ctrl+G", key('g', tea.ModCtrl))
		for _, k := range []tea.KeyPressMsg{
			key(tea.KeyHome, tea.ModCtrl), key(tea.KeyRight, 0), key(tea.KeyRight, tea.ModCtrl), key(tea.KeyDown, 0), key(tea.KeyLeft, 0),
			key(tea.KeyRight, tea.ModShift), key(tea.KeyDown, tea.ModShift), key('f', tea.ModCtrl), key(tea.KeyEnd, 0), key(tea.KeyEnd, tea.ModCtrl),
		} {
			both(k.String(), k)
		}
		h := a.h
		// Back to the start, the cursor inside the block: Esc, no change: the chip.
		h.send(key(tea.KeyHome, tea.ModCtrl))
		h.send(key(tea.KeyRight, 0))
		h.send(key(tea.KeyRight, 0))
		h.send(key(tea.KeyEscape, 0)) // the full screen's window closes with it
		if h.m.ui.DraftWin.Open {
			h.send(key(tea.KeyEscape, 0))
		}
		f := h.frame()
		if h.m.edit.Text() != text || len(h.m.edit.Ranges()) != 1 || !strings.Contains(f, "▣ Long message · 30 lines") || strings.Contains(f, "alpha beta") {
			t.Errorf("Esc with no change: %+v:\n%s", h.m.edit.Ranges(), f)
		}
		if c := h.m.edit.Cursor(); c != 2 {
			t.Errorf("Esc: the cursor moved to %d", c)
		}
		if h.m.ui.Editor.Col != 1 {
			t.Errorf("the cursor left inside is drawn at col %d, want the chip's end", h.m.ui.Editor.Col)
		}
		// Reopened: the cursor where it was.
		draftWin(t, h)
		if c := h.m.edit.Cursor(); c != 2 {
			t.Errorf("reopened: the cursor %d, want where it was", c)
		}
		h.send(key(tea.KeyEscape, 0))
		// The first composer key moves it to the chip's end, no text changed.
		h.send(key(tea.KeyLeft, 0))
		if c := h.m.edit.Cursor(); c != 0 || h.m.edit.Text() != text {
			t.Errorf("a key after it (from the chip's end, over it): the cursor %d", c)
		}
		h.send(key(tea.KeyEnd, tea.ModCtrl))

		// An edit in the window: every range gone, the text ordinary, nothing lost.
		draftWin(t, h)
		h.send(key(tea.KeyHome, tea.ModCtrl))
		h.send(typed('X'))
		if h.m.edit.Text() != "X"+text || len(h.m.edit.Ranges()) != 0 {
			t.Errorf("an edit in the window: %q, %+v", h.m.edit.Text(), h.m.edit.Ranges())
		}
		h.send(key(tea.KeyEscape, 0))
		f = h.frame()
		if strings.Contains(f, "Long message") || !strings.Contains(f, "Xalpha beta 1") {
			t.Errorf("after an edit: ordinary rows:\n%s", f)
		}
		// Undo back to the original bytes: the chip again; the cursor at its end if the
		// step left it inside.
		h.send(key('z', tea.ModCtrl))
		f = h.frame()
		if h.m.edit.Text() != text || len(h.m.edit.Ranges()) != 1 || !strings.Contains(f, "▣ Long message · 30 lines") {
			t.Errorf("undone: %+v:\n%s", h.m.edit.Ranges(), f)
		}
		notInside(t, h, "undo after the window's edit")
		// The window's edit made with its cursor inside the block: its undo puts the
		// cursor at the chip's end.
		draftWin(t, h)
		h.m.edit.Select(3, 3)
		h.send(typed('X'))
		h.send(key(tea.KeyEscape, 0))
		h.send(key('z', tea.ModCtrl))
		if c := h.m.edit.Cursor(); c != len(text) || h.m.edit.Text() != text || len(h.m.edit.Ranges()) != 1 {
			t.Errorf("undo of an edit inside the block: the cursor %d, %+v", c, h.m.edit.Ranges())
		}

		// A selection the window left with an end inside the block: Ctrl+C copies what it
		// holds and it ends; no text changes.
		draftWin(t, h)
		h.m.edit.Select(0, 5)
		h.send(key(tea.KeyEscape, 0))
		if h.m.edit.Text() != text || h.m.edit.HasSelection() != true {
			t.Fatalf("the selection kept till a key: %v", h.m.edit.HasSelection())
		}
		h.send(typed('Z'))
		if h.m.edit.Text() != text+"Z" {
			t.Errorf("the first composer key after it: %q", h.m.edit.Text())
		}
		h.send(key(tea.KeyBackspace, 0))
		// The same selection, then Ctrl+C: it copies what the selection holds and ends it.
		draftWin(t, h)
		h.m.edit.Select(0, 5)
		h.send(key(tea.KeyEscape, 0))
		if _, ok := press(t, h, key('c', tea.ModCtrl)); !ok || clip.last() != text[:5] {
			t.Errorf("Ctrl+C on it: copied %v, %q", ok, clip.last())
		}
		if h.m.edit.HasSelection() || h.m.edit.Text() != text {
			t.Errorf("after Ctrl+C: the selection kept %v, %d bytes", h.m.edit.HasSelection(), len(h.m.edit.Text()))
		}
		notInside(t, h, "Ctrl+C on the window's selection")
		// The cursor left inside it, then a paste: at the chip's end, the chip kept.
		draftWin(t, h)
		h.m.edit.Select(5, 5)
		h.send(key(tea.KeyEscape, 0))
		h.send(tea.PasteMsg{Content: "short"})
		if h.m.edit.Text() != text+"short" || len(h.m.edit.Ranges()) != 1 {
			t.Errorf("a paste with the cursor left inside: %q, %+v", h.m.edit.Text(), h.m.edit.Ranges())
		}
		h.send(key('z', tea.ModCtrl))

		// An invalid list, by a bug: dropped at its next use, the text intact.
		e := &h.m.edit
		e.SetText(text)
		e.Mark(0, 10)
		e.Mark(5, 15) // overlaps the first
		h.send(key(tea.KeyLeft, 0))
		f = h.frame()
		if e.Text() != text || len(e.Ranges()) != 0 || strings.Contains(f, "Long message") || !strings.Contains(f, "alpha beta") {
			t.Errorf("an invalid range: %+v:\n%s", e.Ranges(), f)
		}
		// Sent from the window: as any draft, and the same rows in zone B.
		a.h.m.edit.Clear()
		a.h.send(tea.PasteMsg{Content: text})
		draftWin(t, a.h)
		a.h.send(key(tea.KeyEnter, 0))
		if s := a.sent(); len(s) != 1 || s[0] != text || a.h.m.ui.DraftWin.Open {
			t.Errorf("sent from the window: %d messages, open %v", len(s), a.h.m.ui.DraftWin.Open)
		}
	})
}

// TestChipRestored is criterion 73.19: a failed send's text comes back as ordinary text,
// byte-identical to what was sent; with text typed since, joined as today; a restored
// stored draft is ordinary text, byte-identical.
func TestChipRestored(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		text := longLines(30, "restored")
		w := &wireClaude{}
		h := wireRig(t, w, Options{})
		w.last().refuse.Store(true)
		h.send(tea.PasteMsg{Content: strings.ReplaceAll(text, "\n", "\r\n")}) // the rule's only change
		if len(h.m.edit.Ranges()) != 1 {
			t.Fatalf("no chip: %+v", h.m.edit.Ranges())
		}
		h.send(key(tea.KeyEnter, 0))
		h.settle()
		if s := h.m.State(); s.Turn.Outcome != "failed" {
			t.Fatalf("the write did not fail: outcome %q", s.Turn.Outcome)
		}
		if f := h.frame(); h.m.edit.Text() != text || len(h.m.edit.Ranges()) != 0 || !strings.Contains(f, "restored 30") {
			t.Errorf("restored: %d bytes of %d, %+v:\n%s", len(h.m.edit.Text()), len(text), h.m.edit.Ranges(), f)
		}

		// Typed since the send: both kept, as today (app.go, CmdRestoreDraft).
		r, _ := chipRig(t)
		r.h.send(tea.PasteMsg{Content: text})
		since := "typed since"
		r.h.m.edit.Clear()
		typeText(r.h, since)
		r.h.m.command(state.Command{Kind: state.CmdRestoreDraft, Text: text})
		r.h.send(tea.WindowSizeMsg{Width: 120, Height: 40})
		if got := r.h.m.edit.Text(); got != joinDraft(text, since) || len(r.h.m.edit.Ranges()) != 0 {
			t.Errorf("joined: %q, %+v", got, r.h.m.edit.Ranges())
		}
	})
}

// TestChipStoredDraft is criterion 73.19's stored draft: a chip's text stored and
// restored in a new dashboard is ordinary text, byte-identical; the stored format is
// the text alone.
func TestChipStoredDraft(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		text := longLines(30, "stored")
		d := input.Drafts{Dir: t.TempDir()}
		cwd := t.TempDir()
		h := rcRig(t, &d, cwd, &clip{}, &wireClaude{})
		h.send(tea.PasteMsg{Content: text})
		if len(h.m.edit.Ranges()) != 1 {
			t.Fatalf("no chip: %+v", h.m.edit.Ranges())
		}
		project := h.m.State().Context.Project
		for _, c := range h.m.closeSession() { // written and let go, for the next start (plan 33)
			h.exec(c)
		}
		h.settle()
		if got := string(draftFile(t, d, project)); got != text {
			t.Fatalf("stored %d bytes, want the text's %d", len(got), len(text))
		}
		h2 := rcRig(t, &d, cwd, &clip{}, &wireClaude{})
		if got := h2.m.edit.Text(); got != text || len(h2.m.edit.Ranges()) != 0 {
			t.Errorf("restored %q, %+v", got, h2.m.edit.Ranges())
		}
		if f := h2.frame(); strings.Contains(f, "Long message") || !strings.Contains(f, "stored 30") {
			t.Errorf("a restored draft: ordinary rows:\n%s", f)
		}
	})
}

// deliveryInputs are criterion 73.10's kinds of text, each long enough for a chip.
func deliveryInputs() map[string]string {
	return map[string]string{
		"markdown":   "# Title\n\nSome **bold** and `code`.\n\n- one\n- two\n\n> quoted\n\n[a link](https://example.com)",
		"fenced":     "Look:\n\n```go\nfunc f() {\n\treturn 1\n}\n```\n\nThat is it.",
		"table":      "| a | b |\n|---|:-:|\n| 1 | 2 |\n| 3 | 4 |\n| 5 | 6 |",
		"json":       "{\n  \"a\": [1, 2, 3],\n  \"b\": {\"c\": \"d\\n\"},\n  \"e\": null\n}",
		"yaml":       "a: 1\nb:\n  - c\n  - d: 'e'\n\nf: |\n  text\n",
		"powershell": "$x = Get-ChildItem `\n  -Path \"C:\\temp\"\nforeach ($i in $x) {\n\tWrite-Host $i.Name\n}",
		"unicode":    "日本語\n한국어\n👍🏽 👨‍👩‍👧 🇧🇪\ne\u0301 n\u0303\n",
		"indent":     "    four\n\ttab\n  \t mixed  \ntrailing   \n \t ",
		"tabs":       "\ta\t\tb\t\n\t\n\t\n\t",
		"empty":      "\n\n\n\nmiddle\n\n\n\n",
		"backslash":  "first\nsecond\nthird\nfourth \\",
		"no break":   "first\nsecond\nthird\nfourth",
		"crlf":       "one\r\ntwo\rthree\n\r\nfour",
		"controls":   "a\x01b\nc\x7fd\ne\u0085f\ng\x1bh",
		"big":        draftLines(50_000),
	}
}

// TestJoinedCells is criteria 73.1 and 73.10 on what found R-H, a false chip, on both
// adapters (doc/plan73-evaluation.md): characters joined by U+200D are one cell wider
// than a row and take one row, so one or three of them are ordinary text in the
// composer and in zone B, and four a chip; what is sent is their bytes either way.
func TestJoinedCells(t *testing.T) {
	joined := strings.Repeat("日"+string(rune(0x200D)), 399) + "日"
	for name, rig := range map[string]func(*testing.T) inputRig{"claude-code": claudeRig, "codex": codexRig} {
		t.Run(name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				r := rig(t)
				h := r.h
				for i, c := range []struct {
					name, text string
					chip       bool
				}{
					{"one", joined, false},
					{"three", joined + " " + joined + joined, false},
					{"four", joined + joined + " " + joined + joined, true},
				} {
					h.send(tea.PasteMsg{Content: c.text})
					if got := len(h.m.edit.Ranges()) == 1; got != c.chip || h.m.edit.Text() != c.text {
						t.Errorf("%s pasted: a chip %v, want %v; %d bytes, want %d", c.name, got, c.chip, len(h.m.edit.Text()), len(c.text))
					}
					if got := strings.Contains(h.frame(), "Long message"); got != c.chip {
						t.Errorf("%s pasted: the composer shows a chip: %v, want %v:\n%s", c.name, got, c.chip, h.frame())
					}
					h.send(key(tea.KeyEnter, 0))
					r.end()
					sent := r.sent()
					if len(sent) != i+1 || sent[i] != c.text {
						t.Fatalf("%s: %d messages sent, the last not its bytes", c.name, len(sent))
					}
					if r.raw != nil {
						r.raw(t, i, c.text)
					}
					var entry string
					for _, e := range h.m.State().Conversation {
						if e.Role == "user" {
							entry = e.Text
						}
					}
					if entry != c.text {
						t.Errorf("%s: Entry.Text has %d bytes, want %d", c.name, len(entry), len(c.text))
					}
					if got := strings.Contains(h.frame(), "Long message"); got != c.chip {
						t.Errorf("%s sent: zone B shows a chip: %v, want %v:\n%s", c.name, got, c.chip, h.frame())
					}
				}
			})
		})
	}
}

// TestDelivery is criterion 73.10, on both adapters: for each kind of text, pasted in
// the composer, in the draft window, edited in the draft window and mixed with typed
// text, the line the adapter writes decodes to exactly Editor.Text(), the input under
// the paste's line-break rule, and to Entry.Text; no placeholder byte in any of them.
func TestDelivery(t *testing.T) {
	rule := func(s string) string {
		return strings.ReplaceAll(strings.ReplaceAll(s, "\r\n", "\n"), "\r", "\n")
	}
	for name, rig := range map[string]func(*testing.T) inputRig{"claude-code": claudeRig, "codex": codexRig} {
		t.Run(name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				r := rig(t)
				h := r.h
				n := 0
				deliver := func(what, want string) {
					t.Helper()
					got := h.m.edit.Text()
					if got != want {
						t.Errorf("%s: the editor holds %d bytes, want %d", what, len(got), len(want))
					}
					if strings.IndexByte(got, 0) >= 0 {
						t.Errorf("%s: a NUL byte in the editor", what)
					}
					h.send(key(tea.KeyEnter, 0))
					r.end()
					sent := r.sent()
					if len(sent) != n+1 {
						t.Fatalf("%s: %d messages sent, want %d", what, len(sent), n+1)
					}
					if sent[n] != want {
						t.Errorf("%s: sent %d bytes differ from the %d wanted", what, len(sent[n]), len(want))
					}
					if r.raw != nil {
						r.raw(t, n, want)
					}
					var entry string
					for _, e := range h.m.State().Conversation {
						if e.Role == "user" {
							entry = e.Text
						}
					}
					if entry != want {
						t.Errorf("%s: Entry.Text has %d bytes, want %d", what, len(entry), len(want))
					}
					n++
				}
				inputs := deliveryInputs()
				names := make([]string, 0, len(inputs))
				for k := range inputs {
					names = append(names, k)
				}
				slices.Sort(names)
				for _, k := range names {
					in := inputs[k]
					want := rule(in)
					// In the composer: a chip for each, a final \ sent as it is.
					h.send(tea.PasteMsg{Content: in})
					if !render.LongMessage(want, render.MessageWidth(h.m.State(), h.m.ui)) || len(h.m.edit.Ranges()) != 1 {
						t.Fatalf("%s: not a chip: %+v", k, h.m.edit.Ranges())
					}
					deliver(k+", the composer", want)
					// In the draft window: the \ before Enter is a newline there, as today.
					draftWin(t, h)
					h.send(tea.PasteMsg{Content: in})
					if strings.HasSuffix(want, `\`) {
						h.send(typed('!'))
						deliver(k+", the draft window", want+"!")
					} else {
						deliver(k+", the draft window", want)
					}
					// Edited in the draft window.
					h.send(tea.PasteMsg{Content: in})
					draftWin(t, h)
					h.send(key(tea.KeyHome, tea.ModCtrl))
					h.send(typed('X'))
					h.send(key(tea.KeyEnd, tea.ModCtrl))
					h.send(typed('Y'))
					deliver(k+", edited in the draft window", "X"+want+"Y")
					// Mixed with typed text.
					typeText(h, "go on:\n")
					h.send(tea.PasteMsg{Content: in})
					typeText(h, " end")
					deliver(k+", mixed", "go on:\n"+want+" end")
				}
			})
		})
	}
}

// TestLiteralNUL: a NUL the owner pasted is the owner's byte: kept and sent, drawn as
// always, never taken for a chip's place, which comes from the ranges alone.
func TestLiteralNUL(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r, _ := chipRig(t)
		h := r.h
		block := longLines(5, "with\x00nul")
		h.send(tea.PasteMsg{Content: "x\x00y"})
		h.send(tea.PasteMsg{Content: block})
		h.send(tea.PasteMsg{Content: "z\x00"})
		want := "x\x00y" + block + "z\x00"
		f := h.frame()
		if strings.Count(f, "▣ Long message") != 1 || !strings.Contains(f, "x^@y▣ Long message") || !strings.Contains(f, "z^@") {
			t.Errorf("the frame:\n%s", f)
		}
		h.send(key(tea.KeyEnter, 0))
		if s := r.sent(); len(s) != 1 || s[0] != want {
			t.Errorf("sent %q, want %q", s, want)
		}
	})
}

// TestTypingUnchanged is criterion 73.11: typing any number of lines makes no range, its
// rows drawn as ever, and only a sent message's form follows the rule.
func TestTypingUnchanged(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r, _ := chipRig(t)
		h := r.h
		lines := longLines(60, "typed")
		typeText(h, lines)
		if f := h.frame(); len(h.m.edit.Ranges()) != 0 || strings.Contains(f, "Long message") || !strings.Contains(f, "typed 60") || !strings.Contains(f, "60 lines") {
			t.Errorf("60 typed lines: %+v:\n%s", h.m.edit.Ranges(), f)
		}
		if v := h.m.ui.Editor; v.Total != 60 || len(v.Chips) != 0 || v.Hidden != 0 {
			t.Errorf("the view: %d lines, chips %+v, hidden %d", v.Total, v.Chips, v.Hidden)
		}
		h.send(key(tea.KeyEnter, 0))
		r.emit(event.UserMessage, event.User{Text: lines})
		r.end()
		if f := h.frame(); !strings.Contains(f, "▣ Long message · 60 lines") || strings.Contains(f, "typed 30") {
			t.Errorf("once sent, a chip:\n%s", f)
		}
	})
}

// TestZoneBSameFrame is criterion 73.4's last clause: a message from the composer, from
// the draft window and from a replay gives the same frame.
func TestZoneBSameFrame(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		text := longLines(30, "same")
		for _, size := range [][2]int{{80, 24}, {120, 40}, {200, 60}, {208, 59}} {
			frame := func(fill func(r inputRig)) string {
				r, _ := chipRig(t)
				r.h.send(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
				fill(r)
				r.emit(event.UserMessage, event.User{Text: text})
				r.h.settle()
				return noProjectBar(r.h.frame())
			}
			composer := frame(func(r inputRig) {
				r.h.send(tea.PasteMsg{Content: text})
				r.h.send(key(tea.KeyEnter, 0))
			})
			window := frame(func(r inputRig) {
				draftWin(t, r.h)
				r.h.send(tea.PasteMsg{Content: text})
				r.h.send(key(tea.KeyEnter, 0))
			})
			if composer != window {
				t.Errorf("%v: the composer's and the draft window's differ:\n%s\n---\n%s", size, composer, window)
			}
			if !strings.Contains(composer, "▣ Long message · 30 lines") {
				t.Errorf("%v: no chip:\n%s", size, composer)
			}
			// A replay: the same entry with no key pressed: zone B the same.
			replayed := func() string {
				r, _ := chipRig(t)
				r.h.send(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
				r.emit(event.UserMessage, event.User{Text: text})
				r.h.settle()
				return noProjectBar(r.h.frame())
			}()
			if zoneB(replayed) != zoneB(composer) {
				t.Errorf("%v: the replay's zone B differs:\n%s\n---\n%s", size, replayed, composer)
			}
		}
	})
}

// zoneB is a frame's rows that show a message's chip and its YOU row, the turn's state
// elsewhere on the screen left out.
func zoneB(f string) string {
	var out []string
	for _, r := range strings.Split(f, "\n") {
		if strings.Contains(r, "YOU") || strings.Contains(r, "Long message") {
			if _, b, ok := strings.Cut(r, "││"); ok {
				r = b
			}
			out = append(out, r)
		}
	}
	return strings.Join(out, "\n")
}

// TestChipResize is criterion 73.6: zone B decides at the current width: one message
// normal, then a chip, then normal; a panel docked the same; four logical lines a chip
// at every width; the state unchanged; an entry cursor on it leaves when it stops being
// a chip; a composer chip unchanged.
func TestChipResize(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r, _ := chipRig(t)
		h := r.h
		w120 := render.MessageWidth(h.m.State(), h.m.ui)
		one := strings.Repeat("word ", (3*w120-10)/5) + "end"
		r.emit(event.UserMessage, event.User{Text: one})
		r.end()
		if render.LongMessage(one, w120) {
			t.Fatalf("the message is long at %d cells", w120)
		}
		entries := func() int { return len(h.m.State().Conversation) }
		n := entries()
		chip := func() bool { return strings.Contains(h.frame(), "▣ Long message · 1 line") }
		if chip() {
			t.Fatalf("a chip at 120x40:\n%s", h.frame())
		}
		h.send(tea.WindowSizeMsg{Width: 100, Height: 30})
		if !chip() {
			t.Errorf("at 100x30, no chip:\n%s", h.frame())
		}
		h.send(tea.WindowSizeMsg{Width: 200, Height: 60})
		if chip() {
			t.Errorf("at 200x60, a chip:\n%s", h.frame())
		}
		h.send(tea.WindowSizeMsg{Width: 120, Height: 40})
		if chip() || entries() != n || h.m.State().Conversation[0].Text != one {
			t.Errorf("back at 120x40: chip %v, %d entries:\n%s", chip(), entries(), h.frame())
		}

		// The cursor on it at 100x30, then room for all of it: it leaves.
		h.send(tea.WindowSizeMsg{Width: 100, Height: 30})
		h.draw(key(tea.KeyUp, 0))
		if h.m.ui.Entry == 0 {
			t.Fatalf("↑ found no chip at 100x30:\n%s", h.frame())
		}
		h.draw(tea.WindowSizeMsg{Width: 200, Height: 60})
		if h.m.ui.Entry != 0 {
			t.Errorf("the cursor stayed on entry %d, no chip any more", h.m.ui.Entry)
		}
		h.send(key(tea.KeyEscape, 0))

		// The Explorer docked at 200x60: narrower, the same rule at that width.
		h.send(tea.WindowSizeMsg{Width: 200, Height: 60})
		before := render.MessageWidth(h.m.State(), h.m.ui)
		for _, c := range h.m.showExplorer() {
			h.exec(c)
		}
		h.settle()
		after := render.MessageWidth(h.m.State(), h.m.ui)
		if after >= before {
			t.Fatalf("docking: %d cells, %d before", after, before)
		}
		if got, want := chip(), render.LongMessage(one, after); got != want {
			t.Errorf("docked at %d cells: chip %v, want %v:\n%s", after, got, want, h.frame())
		}
		h.m.ui.Explorer = render.ExplorerView{}
		h.m.ui.Focus = "B"
		h.send(tea.WindowSizeMsg{Width: 120, Height: 40})

		// Four logical lines: a chip at every width.
		r.emit(event.UserMessage, event.User{Text: "a\nb\nc\nd"})
		r.end()
		for _, size := range [][2]int{{80, 24}, {100, 30}, {120, 40}, {200, 60}, {208, 59}} {
			h.send(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
			if f := h.frame(); !strings.Contains(f, "▣ Long message · 4 lines · 7 B") {
				t.Errorf("%v: four lines:\n%s", size, f)
			}
		}

		// A composer chip stays a chip, whatever the width says of its block now.
		h.send(tea.WindowSizeMsg{Width: 100, Height: 30})
		narrow := render.MessageWidth(h.m.State(), h.m.ui)
		block := strings.Repeat("x", 3*narrow+1)
		h.send(tea.PasteMsg{Content: block})
		if len(h.m.edit.Ranges()) != 1 {
			t.Fatalf("no chip at 100x30: %+v", h.m.edit.Ranges())
		}
		h.send(tea.WindowSizeMsg{Width: 200, Height: 60})
		if f := h.frame(); len(h.m.edit.Ranges()) != 1 || !strings.Contains(f, "▣ Long message · 1 line") || strings.Contains(f, "xxxxx") {
			t.Errorf("a composer chip at 200x60: %+v:\n%s", h.m.edit.Ranges(), f)
		}
	})
}

// TestChipFullView is criteria 73.7 and 73.8: ↑ from an empty input stops on a chip with
// ▸ and the hint, on no normal message; B's height unchanged; Enter opens the full view
// from its top with every line, Esc returns; a double click on the chip's row or the YOU
// row opens it; y there gives the clipboard exactly the entry's text.
func TestChipFullView(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r, clip := chipRig(t)
		h := r.h
		now := testNow
		h.m.opt.Now = func() time.Time { return now }
		r.emit(event.UserMessage, event.User{Text: "a short question"})
		r.end()
		wide := strings.Repeat("0123456789", 1500)                                                                             // 15,000 cells, wrapped, no break added
		text := "\tindented\ttabs\ntrailing spaces   \n\nblank above\n日本語 👍🏽 e\u0301\n" + wide + "\n" + longLines(398, "last") // 404 lines
		r.emit(event.UserMessage, event.User{Text: text})
		r.end()
		s := h.m.State()
		p := 0
		for i, e := range s.Conversation {
			if e.Text == text {
				p = i + 1
			}
		}
		f := h.frame()
		rows := h.m.screen.Conv.Rows
		if !strings.Contains(f, "▣ Long message · 404 lines") {
			t.Fatalf("no chip:\n%s", f)
		}
		h.draw(key(tea.KeyUp, 0))
		f = h.frame()
		if h.m.ui.Entry != p || !rowWith(f, "▎▸", "▣ Long message · 404 lines", "Enter full view") {
			t.Fatalf("↑: entry %d, want %d:\n%s", h.m.ui.Entry, p, f)
		}
		if h.m.screen.Conv.Rows != rows {
			t.Errorf("the cursor changed B's rows: %d, want %d", h.m.screen.Conv.Rows, rows)
		}
		h.draw(key(tea.KeyUp, 0)) // the short message above is no stop
		if h.m.ui.Entry != p {
			t.Errorf("↑ again left the only chip for entry %d", h.m.ui.Entry)
		}
		before := h.frame()
		h.draw(key(tea.KeyEnter, 0))
		f = h.frame()
		for _, want := range []string{"◀ back (Esc)", "YOU", "404 lines", "indented", "trailing spaces", "blank above", "日本語", "y copy the message", "↑↓ PgUp PgDn scroll"} {
			if !strings.Contains(f, want) {
				t.Errorf("the full view lacks %q:\n%s", want, f)
			}
		}
		if h.m.ui.ViewEntry != p {
			t.Errorf("view %d, want %d", h.m.ui.ViewEntry, p)
		}
		h.draw(key(tea.KeyEnd, 0))
		if f := h.frame(); !strings.Contains(f, "last 398") {
			t.Errorf("End, the last line:\n%s", f)
		}
		h.draw(key(tea.KeyHome, 0))

		// y: the entry's text byte for byte, by the usual copy and toast; nothing else moves.
		if _, ok := press(t, h, typed('y')); !ok {
			t.Fatal("y started no copy")
		}
		if clip.last() != text {
			t.Errorf("y copied %d bytes, want the message's %d", len(clip.last()), len(text))
		}
		if h.m.ui.Toast != "copied 404 lines · native" || h.m.ui.ViewEntry != p {
			t.Errorf("after y: toast %q, view %d", h.m.ui.Toast, h.m.ui.ViewEntry)
		}
		if s.Conversation[p-1].Text != text {
			t.Error("y changed the entry")
		}
		h.send(toastDoneMsg{h.m.toastGen})
		h.draw(key(tea.KeyEscape, 0))
		if f := h.frame(); h.m.ui.ViewEntry != 0 || h.m.ui.Entry != p || f != before {
			t.Errorf("Esc: view %d, entry %d:\n%s\nbefore:\n%s", h.m.ui.ViewEntry, h.m.ui.Entry, f, before)
		}
		h.draw(key(tea.KeyEscape, 0))

		// A double click, on the chip's row and on the YOU row above it.
		for _, target := range []string{"▣ Long message", "YOU"} {
			now = now.Add(time.Second)
			nth := 0
			if target == "YOU" {
				nth = 1 // the short message's is the first
			}
			x, y := at(t, h, target, nth)
			mouseClick(t, h, x+1, y)
			if h.m.ui.ViewEntry != 0 {
				t.Errorf("one click on %s opened the view", target)
			}
			mouseClick(t, h, x+1, y)
			if h.m.ui.ViewEntry != p {
				t.Errorf("a double click on %s: view %d, want %d", target, h.m.ui.ViewEntry, p)
			}
			h.draw(key(tea.KeyEscape, 0))
		}
	})
}

// TestChipSearchSelection is criterion 73.9: a selection across a chip copies the whole
// message byte for byte, as across a fold; the search's parts are TestSearchLit's and
// TestSearchWhole's, on the same chip's rows.
func TestChipSearchSelection(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r, clip := chipRig(t)
		h := r.h
		text := "\tindented\ntrailing  \n\nline four\nline five"
		r.emit(event.MessageCompleted, event.Completed{MessageID: "m0", Text: "the answer before"})
		r.emit(event.UserMessage, event.User{Text: text})
		r.emit(event.MessageCompleted, event.Completed{MessageID: "m1", Text: "the answer after"})
		h.settle()
		x0, y0 := at(t, h, "the answer before", 0)
		x1, y1 := at(t, h, "the answer after", 0)
		chipRow := func() string {
			rows := strings.Split(h.m.View().Content, "\n")
			for _, row := range rows {
				if strings.Contains(plainText(row), "Long message") {
					return row
				}
			}
			t.Fatalf("no chip's row:\n%s", h.frame())
			return ""
		}
		before := chipRow()
		dragCopy(t, h, x0, y0, x1+len("the answer after"), y1)
		if !strings.Contains(clip.last(), text) || !strings.HasPrefix(clip.last(), "the answer before") || !strings.HasSuffix(clip.last(), "the answer after") {
			t.Errorf("a selection across the chip copied %q", clip.last())
		}
		// The chip lit as the bytes are copied, its words the same (The correction of F1).
		lit := chipRow()
		if lit == before || plainText(lit) != plainText(before) {
			t.Errorf("the chip's row under the selection:\n%q\nbefore it:\n%q", lit, before)
		}
		h.send(key(tea.KeyEscape, 0))
		if got := chipRow(); got != before {
			t.Errorf("the chip's row once the selection is cleared:\n%q\nwant:\n%q", got, before)
		}
	})
}

// TestComposerHiddenText is criterion 73.26's values: with a range and the draft window
// closed, EditorView's lines hold no hidden byte, and its line, column, selection and
// total are the compact form's; none of it depends on the block's size.
func TestComposerHiddenText(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r, _ := chipRig(t)
		h := r.h
		typeText(h, "typed one\nab ")
		block := longLines(50_000, "HIDDEN")
		h.send(tea.PasteMsg{Content: block})
		typeText(h, " cd\nlast")
		v := h.m.ui.Editor
		joined := strings.Join(v.Lines, "\n")
		if strings.Contains(joined, "HIDDEN") {
			t.Errorf("a hidden byte in the view's lines: %q", joined[:min(len(joined), 80)])
		}
		if want := "typed one\nab \x00 cd\nlast"; joined != want {
			t.Errorf("the view's lines %q, want %q", joined, want)
		}
		if v.Total != 3 || v.Line != 2 || v.Col != 4 || v.Hidden != 49_999 {
			t.Errorf("total %d, line %d, col %d, hidden %d", v.Total, v.Line, v.Col, v.Hidden)
		}
		if got, want := v.Chips, []render.EditorChip{{Line: 1, Off: 3, Lines: 50_000, Bytes: len(block)}}; !slices.Equal(got, want) {
			t.Errorf("chips %+v, want %+v", got, want)
		}
		// A selection from "ab" over the chip to "cd", in the compact form's places.
		from := len("typed one\na")
		to := from + len("b ") + len(block) + len(" c")
		h.m.edit.Select(from, to)
		h.send(tea.WindowSizeMsg{Width: 120, Height: 40})
		v = h.m.ui.Editor
		if v.From != [2]int{1, 1} || v.To != [2]int{1, 6} || v.Line != 1 || v.Col != 6 {
			t.Errorf("the selection %v..%v, cursor %d:%d", v.From, v.To, v.Line, v.Col)
		}
		// The window open: the whole text, today's view, no chip named.
		draftWin(t, h)
		v = h.m.ui.Editor
		if v.Total != h.m.edit.Lines() || len(v.Chips) != 0 || v.Hidden != 0 {
			t.Errorf("the draft window's view: total %d of %d, chips %+v, hidden %d", v.Total, h.m.edit.Lines(), v.Chips, v.Hidden)
		}
	})
}

// TestGuardFailsOnAChip is criterion 73.23's damaged input: the workload's check run on
// the original composer-paste fill, under the feature, must fail: it names the chip.
func TestGuardFailsOnAChip(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r, _ := chipRig(t)
		h := r.h
		h.send(tea.PasteMsg{Content: draftLines(1_000)})
		if why := ordinaryProblem(h, 1_000); !strings.Contains(why, "chip") {
			t.Errorf("the check passed a composer-paste fill: %q", why)
		}
		h.m.edit.Clear()
		fillOrdinary(h, draftLines(1_000))
		if why := ordinaryProblem(h, 1_000); why != "" {
			t.Errorf("the check on the replacement fill: %s", why)
		}
		// Too small a text, and a view that lies: each its own failure.
		if why := ordinaryProblem(h, 2_000); !strings.Contains(why, "lines in the draft") {
			t.Errorf("too few lines: %q", why)
		}
		// Draft may keep ranges while showing ordinary text, and passes (A2).
		h.m.edit.Clear()
		h.send(tea.PasteMsg{Content: draftLines(1_000)})
		draftWin(t, h)
		if why := ordinaryProblem(h, 1_000); why != "" {
			t.Errorf("the draft window over a chip: %s", why)
		}
	})
}

// chipText is the 50,000-line block of these measures, without its final break.
func chipText() string { return strings.TrimSuffix(draftLines(50_000), "\n") }

// TestChipFrameAllocs is criterion 73.25's guard: a frame with a chip in view allocates
// no more than one with a three-row message, whatever the chip's size.
func TestChipFrameAllocs(t *testing.T) {
	allocs := func(text string) float64 {
		r, _ := copyRig(t, nil)
		h := r.h
		h.send(tea.WindowSizeMsg{Width: 120, Height: 40})
		r.emit(event.UserMessage, event.User{Text: text})
		h.settle()
		_ = h.m.View()
		s, ui := h.m.State(), h.m.ui
		return testing.AllocsPerRun(50, func() { render.Draw(s, ui) })
	}
	three := allocs("first line of the message\nthe second line of it\nthe third and last one")
	for _, c := range []struct {
		name, text string
	}{{"four lines", "a\nb\nc\nd"}, {"404 lines", longLines(404, "line")}, {"50,000 lines", chipText()}, {"one 3 MB line", strings.Repeat("0123456789", 300_000)}} {
		got := allocs(c.text)
		t.Logf("a frame with a chip of %s: %.0f allocations; with a three-row message: %.0f", c.name, got, three)
		if got > three {
			t.Errorf("a frame with a chip of %s: %.0f allocations, over the three-row message's %.0f", c.name, got, three)
		}
	}
}
