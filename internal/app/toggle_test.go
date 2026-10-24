package app

import (
	"fmt"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/Evengelius/agent-command-center/internal/event"
	"github.com/Evengelius/agent-command-center/internal/render"
)

// Plan 35 (brief amendment 36): the Ultracode toggle's image drawn over
// its blank cells as the thumbnails are (plan 32), and clicked.

// halfLeft is the text toggle's left end, Powerline's left half circle
// (render's toggleText).
const halfLeft = "\ue0b6"

// toggleAt is the text toggle's left end: the first half circle on the
// Ultracode row, the slider above it having its own (plan 37).
func toggleAt(t *testing.T, h *harness) (x, y int) {
	t.Helper()
	_, y = at(t, h, render.UltracodeDescription, 0)
	row := strings.Split(frameText(h), "\n")[y]
	i := strings.Index(row, halfLeft)
	if i < 0 {
		t.Fatalf("no text toggle on the Ultracode row %q", row)
	}
	return lipgloss.Width(row[:i]), y
}

// TestToggleDraw is criteria 3 and 5 (design D2): with Sixel, the toggle's
// image drawn on its cells after the frame that opens the selector, not
// after a frame that leaves its row as it was, again when the cursor comes
// onto its row and when Ultracode's state changes, and its place written
// over with the frame's cells once the selector closes; a click on its
// cells changes Ultracode, with Sixel and without.
func TestToggleDraw(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, s := selectorRig(t)
		h.m.opt.After = func(time.Duration, tea.Msg) tea.Cmd { return nil }
		h.send(PrimaryDeviceAttributesEvent{61, 4})
		h.frame()
		n := len(h.m.kicks)
		// drawsToggle draws the frame and checks it asked for one drawing:
		// the frame's own cells at the toggle's place, which erase the image
		// before, then the image in that state.
		drawsToggle := func(what string, on, band bool) render.ToggleShown {
			t.Helper()
			h.frame()
			tg := h.m.screen.Toggle
			if !tg.Shown || tg.On != on || !tg.Usable || tg.Band != band {
				t.Fatalf("%s: the toggle %+v", what, tg)
			}
			if len(h.m.kicks) != n+1 {
				t.Fatalf("%s: %d drawings asked, want 1", what, len(h.m.kicks)-n)
			}
			n++
			place := fmt.Sprintf("\x1b[%d;%dH", tg.Y+1, tg.X+1)
			want := place + h.m.screen.Region(tg.X, tg.Y, render.ToggleCells) + place + render.ToggleSixel(on, true, band)
			if raws := drawn(h, thumbDrawMsg{gen: h.m.kicks[n-1]}); len(raws) != 1 || !strings.Contains(raws[0], want) {
				t.Fatalf("%s: the drawing %q", what, trunc(raws))
			}
			return tg
		}
		h.send(key('p', tea.ModAlt))
		tg := drawsToggle("opened", true, false)

		h.send(key(tea.KeyDown, 0)) // Sonnet 5.5: the toggle's row as it was
		h.frame()
		if len(h.m.kicks) != n {
			t.Errorf("a frame leaving the toggle's row as it was asked for %d drawings", len(h.m.kicks)-n)
		}
		for range 4 { // Haiku 5.5, More models, Effort, Ultracode
			h.send(key(tea.KeyDown, 0))
		}
		drawsToggle("the cursor on its row", true, true)

		mouseAt(t, h, click(tg.X+1, tg.Y))
		mouseAt(t, h, tea.MouseReleaseMsg{X: tg.X + 1, Y: tg.Y, Button: tea.MouseLeft})
		if got := s.modelChanges(); len(got) != 1 || got[0].Ultracode == nil || *got[0].Ultracode {
			t.Fatalf("a click on the image: changes %+v", got)
		}
		s.events <- event.Event{Provider: "claude-code", Kind: event.ModelSettings, Origin: event.Observed, Payload: event.Settings{Model: "claude-fable-5-1",
			Choice: "fable", Effort: "high", EffortKnown: true, UltracodeAvailable: true, UltracodeKnown: true, Answers: true}}
		h.settle()
		n = len(h.m.kicks) // the frames around the click may have asked too
		drawsToggle("Ultracode off", false, true)

		h.send(key(tea.KeyEsc, 0))
		h.frame()
		if h.m.screen.Toggle.Shown || len(h.m.kicks) != n+1 {
			t.Fatalf("closed: the toggle %+v, %d drawings asked", h.m.screen.Toggle, len(h.m.kicks)-n)
		}
		n++
		place := fmt.Sprintf("\x1b[%d;%dH", tg.Y+1, tg.X+1)
		if raws := drawn(h, thumbDrawMsg{gen: h.m.kicks[n-1]}); len(raws) != 1 || !strings.Contains(raws[0], place) || strings.Contains(raws[0], "\x1bP") {
			t.Errorf("the vacated place: %q", trunc(raws))
		}
		h.send(typed('y'))
		h.frame()
		if len(h.m.kicks) != n || len(h.m.drawn) != 0 {
			t.Errorf("once the selector closed: %d drawings asked, %d images left", len(h.m.kicks)-n, len(h.m.drawn))
		}

		// Without Sixel: the text toggle, from one cell to the left, clicked.
		h.send(PrimaryDeviceAttributesEvent{61, 6})
		h.send(key('p', tea.ModAlt))
		h.frame()
		if h.m.screen.Toggle.Shown || len(h.m.kicks) != n {
			t.Errorf("without Sixel: the toggle %+v, %d drawings asked", h.m.screen.Toggle, len(h.m.kicks)-n)
		}
		if x, y := toggleAt(t, h); x != tg.X-1 || y != tg.Y {
			t.Errorf("the text toggle at %d,%d, want %d,%d", x, y, tg.X-1, tg.Y)
		}
		mouseAt(t, h, click(tg.X, tg.Y)) // its knob, off
		mouseAt(t, h, tea.MouseReleaseMsg{X: tg.X, Y: tg.Y, Button: tea.MouseLeft})
		if got := s.modelChanges(); len(got) != 2 || got[1].Ultracode == nil || !*got[1].Ultracode {
			t.Errorf("a click on the text toggle: changes %+v", got)
		}
	})
}
