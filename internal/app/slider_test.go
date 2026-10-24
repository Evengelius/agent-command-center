package app

import (
	"fmt"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/Evengelius/agent-command-center/internal/event"
	"github.com/Evengelius/agent-command-center/internal/render"
)

// Plan 37 (brief amendment 38): the effort slider worked with ← and →,
// drawn over its cells and gliding when the level in force changes.

// fableSettings is Fable 5.1 in force at the level, as the session reports
// it in answer to a change.
func fableSettings(level string) event.Event {
	return event.Event{Provider: "claude-code", Kind: event.ModelSettings, Origin: event.Observed, Payload: event.Settings{Model: "claude-fable-5-1",
		Choice: "fable", Effort: level, EffortKnown: true, Ultracode: true, UltracodeAvailable: true, UltracodeKnown: true, Answers: true}}
}

// toEffort opens the selector by Alt+P and puts the cursor on Effort.
func toEffort(t *testing.T, h *harness) {
	t.Helper()
	h.send(key('p', tea.ModAlt))
	for range 4 {
		h.send(key(tea.KeyDown, 0)) // Sonnet 5.5, Haiku 5.5, More models, Effort
	}
	if c := h.m.ui.Models.Cursor; c != 6 {
		t.Fatalf("the cursor on row %d, want 6 (Effort)", c)
	}
}

// TestSliderKeys is criterion 6: on Effort, → and ← ask for the level after
// and before the one in force, Claude Code's one at a time; nothing past the
// ends; a held effort asks nothing and zone B says why; a Codex level
// waiting for the next message counts as shown; on another row ← closes the
// selector and goes on.
func TestSliderKeys(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, s := selectorRig(t) // Fable 5.1 at High
		toEffort(t, h)
		h.send(key(tea.KeyRight, 0))
		if got := s.modelChanges(); len(got) != 1 || got[0].Effort != "xhigh" || got[0].Model != "" || got[0].Ultracode != nil {
			t.Fatalf("→ asked %+v, want XHigh alone (Claude Code's adapter adds the Ultracode in force, plan 34)", got)
		}
		h.send(key(tea.KeyLeft, 0)) // while XHigh waits for its answer
		if got := s.modelChanges(); len(got) != 1 {
			t.Errorf("a key while a change waits asked %+v", got[1:])
		}
		s.events <- fableSettings("xhigh")
		h.settle()
		h.send(key(tea.KeyLeft, 0))
		if got := s.modelChanges(); len(got) != 2 || got[1].Effort != "high" {
			t.Fatalf("← from XHigh asked %+v", got)
		}
		s.events <- fableSettings("max")
		h.settle()
		h.send(key(tea.KeyRight, 0))
		s.events <- fableSettings("low")
		h.settle()
		h.send(key(tea.KeyLeft, 0))
		if got := s.modelChanges(); len(got) != 2 || !h.m.ui.Models.Open {
			t.Errorf("past the ends asked %+v; open %v", got[2:], h.m.ui.Models.Open)
		}
		h.send(key(tea.KeyUp, 0)) // More models
		h.send(key(tea.KeyLeft, 0))
		if h.m.ui.Models.Open || len(s.modelChanges()) != 2 {
			t.Errorf("← on More models: open %v, changes %+v", h.m.ui.Models.Open, s.modelChanges())
		}
	})
	synctest.Test(t, func(t *testing.T) {
		h, s := selectorRig(t)
		s.events <- event.Event{Provider: "claude-code", Kind: event.ModelSettings, Origin: event.Observed, Payload: event.Settings{Model: "claude-fable-5-1",
			Choice: "fable", Effort: "high", EffortKnown: true, EffortLock: "CLAUDE_CODE_EFFORT_LEVEL", Ultracode: true, UltracodeAvailable: true, UltracodeKnown: true}}
		h.settle()
		toEffort(t, h)
		h.send(key(tea.KeyRight, 0))
		if got := s.modelChanges(); len(got) != 0 || !strings.Contains(frameText(h), "the effort is set by CLAUDE_CODE_EFFORT_LEVEL") {
			t.Errorf("a held effort: changes %+v\n%s", got, frameText(h))
		}
	})
	synctest.Test(t, func(t *testing.T) {
		h, s := selectorRig(t)
		s.nextTurn = true // as Codex: a change waits for the next message
		toEffort(t, h)
		h.send(key(tea.KeyRight, 0))
		h.send(key(tea.KeyRight, 0))
		if a := h.m.s.Models.Asked; a == nil || a.Effort != "max" || !a.NextTurn {
			t.Errorf("→ twice with a change waiting: asked %+v, want Max for the next turn", a)
		}
	})
}

// drawnBy delivers a drawing message of any kind and gives the raw
// sequences it wrote.
func drawnBy(h *harness, msg tea.Msg) []string {
	_, cmd := h.m.Update(msg)
	var out []string
	for _, c := range batch(cmd) {
		if raw, ok := c().(tea.RawMsg); ok {
			out = append(out, fmt.Sprint(raw.Msg))
		}
	}
	return out
}

// TestSliderDraw is criterion 7: with Sixel, the slider's image drawn after
// the frame that opens the selector, its cells written first; when the
// level in force changes, drawn again gliding, its clock from the frame
// that changed the label, so the first drawing, 50 ms after it, is already
// on its way: positions between the two levels, rising, the last on the
// new one, every 30 ms within 180 ms of the frame; its place written over
// once the selector closes, the glide forgotten.
func TestSliderDraw(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, s := selectorRig(t)
		h.m.opt.After = func(time.Duration, tea.Msg) tea.Cmd { return nil }
		h.m.opt.Now = time.Now // the bubble's clock, which time.Sleep moves, for the glide
		h.send(PrimaryDeviceAttributesEvent{61, 4})
		h.frame()
		h.send(key('p', tea.ModAlt))
		h.frame()
		sl := h.m.screen.Slider
		if !sl.Shown || sl.Look.Levels != 5 || sl.Look.Pos != 2 {
			t.Fatalf("the slider %+v", sl)
		}
		place := fmt.Sprintf("\x1b[%d;%dH", sl.Y+1, sl.X+1)
		want := place + h.m.screen.Region(sl.X, sl.Y, render.SliderCells) + place + render.SliderSixel(sl.Look)
		if raws := drawnBy(h, thumbDrawMsg{gen: h.m.kicks[len(h.m.kicks)-1]}); len(raws) != 1 || !strings.Contains(raws[0], want) || h.m.slide.moving {
			t.Fatalf("the drawing %q, gliding %v", trunc(raws), h.m.slide.moving)
		}

		s.events <- fableSettings("max")
		h.settle()
		h.frame()
		if sl = h.m.screen.Slider; sl.Look.Pos != 4 {
			t.Fatalf("after Max came in force: %+v", sl)
		}
		gen := h.m.kicks[len(h.m.kicks)-1]
		time.Sleep(thumbFirst) // the drawing comes once the frame is out
		if raws := drawnBy(h, thumbDrawMsg{gen: gen}); len(raws) != 1 || !strings.Contains(raws[0], place) || !h.m.slide.moving {
			t.Fatalf("the glide's first drawing %q, gliding %v", trunc(raws), h.m.slide.moving)
		}
		pos := []float64{h.m.slide.drawn.Look.Pos}
		if pos[0] <= 2.5 {
			t.Errorf("the first drawing, 50 ms after the frame, at %.2f: not on its way from 2", pos[0])
		}
		for i := 0; i < 10 && h.m.slide.moving; i++ {
			time.Sleep(slideStep)
			raws := drawnBy(h, slideMsg{gen: gen})
			if len(raws) != 1 || !strings.Contains(raws[0], place+h.m.screen.Region(sl.X, sl.Y, render.SliderCells)+place+"\x1bP") {
				t.Fatalf("step %d: %q", i+1, trunc(raws))
			}
			pos = append(pos, h.m.slide.drawn.Look.Pos)
		}
		for i := 1; i < len(pos); i++ {
			if pos[i] <= pos[i-1] || pos[i] > 4 {
				t.Errorf("positions %v: not rising from 2 to 4", pos)
				break
			}
		}
		if len(pos) < 4 || len(pos) > 6 || pos[len(pos)-1] != 4 || h.m.slide.moving {
			t.Errorf("positions %v, gliding %v; want 4 to 6 drawings ending on 4", pos, h.m.slide.moving)
		}

		h.send(key(tea.KeyEsc, 0))
		h.frame()
		raws := drawnBy(h, thumbDrawMsg{gen: h.m.kicks[len(h.m.kicks)-1]})
		if len(raws) != 1 || !strings.Contains(raws[0], place) || strings.Contains(withoutPill(t, h, raws[0]), "\x1bP") || h.m.slide != (slide{}) {
			t.Errorf("closed: %q, the glide %+v", trunc(raws), h.m.slide)
		}
	})
}
