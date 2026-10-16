package app

import (
	"path/filepath"
	"strings"
	"testing"
	"testing/synctest"

	tea "charm.land/bubbletea/v2"

	"github.com/Evengelius/agent-command-center/internal/render"
)

// editKeys is the keys the keystroke measures cycle through: typing,
// moving, deleting, a newline and a selection.
var editKeys = []tea.KeyPressMsg{
	typed('x'), key(tea.KeyRight, 0), key(tea.KeyDown, 0), key(tea.KeyBackspace, 0),
	key('j', tea.ModCtrl), key(tea.KeyUp, 0), key(tea.KeyRight, tea.ModCtrl), key(tea.KeyDown, tea.ModShift),
}

// keystrokeAllocs is the most allocations a keystroke may take, from Update
// to the frame's string, on average over editKeys: 822 when it was set (plan
// 4, step 5's keystroke fix), against 8,462 before it. Drawing the whole
// draft, or the whole screen, is thousands more.
const keystrokeAllocs = 1200

// TestKeystrokeAllocs is test 6's deterministic guard (evaluation.md,
// Keystroke timing): a keystroke in a 10,000-line draft costs a bounded
// number of allocations, whatever the machine's load, and the editor draws
// from a window of the draft, never all of it. The timing step measures the
// time.
func TestKeystrokeAllocs(t *testing.T) {
	r, _ := copyRig(t, nil)
	h := r.h
	h.send(tea.PasteMsg{Content: draftLines(10_000)})
	if n := h.m.edit.Lines(); n < 10_000 {
		t.Fatalf("%d lines in the draft", n)
	}
	for _, place := range []tea.KeyPressMsg{key(tea.KeyHome, tea.ModCtrl), key(tea.KeyEnd, tea.ModCtrl)} {
		h.send(place)
		_ = h.m.View()
		got := testing.AllocsPerRun(20, func() {
			for _, k := range editKeys {
				h.m.Update(k)
				_ = h.m.View()
			}
		}) / float64(len(editKeys))
		t.Logf("%s: %.0f allocations per keystroke (bound %d)", place, got, keystrokeAllocs)
		if got > keystrokeAllocs {
			t.Errorf("%s: %.0f allocations per keystroke, over the bound of %d", place, got, keystrokeAllocs)
		}
		// Wrapping a line to count its rows allocates nothing: the bound
		// above would miss a renderer given every line of the draft.
		if n, most := len(h.m.editorView().Lines), 4*max(h.m.ui.Height, render.MinHeight); n > most {
			t.Errorf("%s: the editor draws from %d lines of the draft, over %d (4 screen heights)", place, n, most)
		}
	}
}

// TestFrameReuse checks the frame View builds from the one before it
// (render.Screen.StringAfter) is the frame drawn afresh, escape codes
// included, through edits, wrapping, a selection, a mouse selection and a
// resize.
func TestFrameReuse(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newHarness(t, Options{Config: Config{Mock: filepath.Join("..", "..", "testdata", "mock", "claude-subagent-approval.jsonl")}})
		h.settle()
		steps := []tea.Msg{
			tea.WindowSizeMsg{Width: 120, Height: 40},
			tea.PasteMsg{Content: "plain line\n\tTabbed\x01ctl\x7f del \u0085 c1\n日本語 wide 👍 é\xff bad\n" + strings.Repeat("long words wrap here ", 20) + "\nlast"},
		}
		for _, k := range []tea.KeyPressMsg{
			typed('x'), key(tea.KeyUp, 0), key(tea.KeyLeft, 0), key(tea.KeyDown, tea.ModShift), key(tea.KeyRight, tea.ModShift),
			key(tea.KeyHome, tea.ModCtrl), key(tea.KeyRight, tea.ModCtrl), key(tea.KeyEnd, tea.ModShift), key(tea.KeyDown, 0),
			key(tea.KeyBackspace, 0), key('j', tea.ModCtrl), key(tea.KeyTab, 0), key(tea.KeyEnd, tea.ModCtrl),
		} {
			steps = append(steps, k)
		}
		steps = append(steps,
			click(20, 8), tea.MouseMotionMsg{X: 70, Y: 12, Button: tea.MouseLeft}, tea.MouseMotionMsg{X: 40, Y: 10, Button: tea.MouseLeft},
			tea.MouseReleaseMsg{X: 40, Y: 10, Button: tea.MouseLeft},
			tea.WindowSizeMsg{Width: 200, Height: 60}, typed('y'), tea.WindowSizeMsg{Width: 80, Height: 24}, typed('z'),
		)
		prev := ""
		changed := 0
		for i, msg := range steps {
			h.send(msg)
			got := h.m.View().Content
			if want := render.Draw(h.m.s, h.m.ui).String(); got != want {
				t.Fatalf("step %d (%T): the reused frame differs from the frame drawn afresh", i, msg)
			}
			if got != prev {
				changed++
			}
			prev = got
		}
		if changed < len(steps)*3/4 {
			t.Errorf("%d of %d steps changed the frame: the steps do not exercise the reuse", changed, len(steps))
		}
	})
}
