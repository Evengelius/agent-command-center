package app

import (
	"strings"
	"testing"
	"testing/synctest"

	tea "charm.land/bubbletea/v2"

	"github.com/Evengelius/agent-command-center/internal/render"
)

// Plan 38 (brief amendment 39): a Codex change between turns taken at once,
// as Claude Code's, with no ring and no "(next turn)", through the real
// Codex adapter (wireCodex's models).

// toCodexEffort opens the selector by Alt+P and puts the cursor on Effort.
func toCodexEffort(t *testing.T, h *harness) {
	t.Helper()
	h.send(key('p', tea.ModAlt))
	for range 10 {
		items := render.SelectorItems(h.m.s, h.m.ui.Models)
		if c := h.m.ui.Models.Cursor; c < len(items) && items[c].Kind == 'e' {
			return
		}
		h.send(key(tea.KeyDown, 0))
	}
	t.Fatal("the cursor did not reach Effort")
}

// lineWith is the frame's first line holding text, "" when none does.
func lineWith(h *harness, text string) string {
	for _, l := range strings.Split(frameText(h), "\n") {
		if strings.Contains(l, text) {
			return l
		}
	}
	return ""
}

// TestCodexSliderAtOnce is criterion 38.8 on the real Codex adapter: → on
// Effort writes one thread/settings/update for XHigh, and Codex's
// notification moves the slider and the pill, with no ring and no "(next
// turn)"; with a Codex that refuses the update, → rings the level, the
// editor row reads "(next turn)" and zone B says why, as plan 34's Codex.
func TestCodexSliderAtOnce(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := &wireCodex{thread: "th-38", models: true}
		h := codexWire(t, w, Options{})
		toCodexEffort(t, h)
		if row := lineWith(h, "Effort ("); !strings.Contains(row, "Effort (High)") {
			t.Fatalf("the Effort row %q", row)
		}
		h.send(key(tea.KeyRight, 0))
		h.settle()
		if u := w.method("thread/settings/update"); len(u) != 1 || !strings.Contains(u[0], `"params":{"threadId":"th-38","effort":"xhigh"}`) {
			t.Errorf("thread/settings/update written %q", u)
		}
		row := lineWith(h, "Effort (")
		if !strings.Contains(row, "Effort (XHigh)") || strings.Contains(row, "○") || h.m.s.Models.Asked != nil ||
			strings.Contains(frameText(h), "(next turn)") || !strings.Contains(frameText(h), "GPT-6-Sol XHigh") {
			t.Errorf("after the answer, the row %q, asked %+v; frame:\n%s", row, h.m.s.Models.Asked, frameText(h))
		}
		if len(w.method("turn/start")) != 0 {
			t.Error("a turn/start written")
		}
	})
	synctest.Test(t, func(t *testing.T) {
		w := &wireCodex{thread: "th-38", models: true, refuse: true}
		h := codexWire(t, w, Options{})
		toCodexEffort(t, h)
		h.send(key(tea.KeyRight, 0))
		h.settle()
		row := lineWith(h, "Effort (")
		if a := h.m.s.Models.Asked; a == nil || !a.NextTurn || a.Effort != "xhigh" || !strings.Contains(row, "Effort (High)") || !strings.Contains(row, "○") {
			t.Errorf("after the refusal: asked %+v, the row %q", a, row)
		}
		f := frameText(h)
		if !strings.Contains(f, "→ GPT-6-Sol XHigh (next turn)") || !strings.Contains(f, "Codex takes model changes with the next message") {
			t.Errorf("no next-turn label or notice:\n%s", f)
		}
		// The next → adds to the waiting change, with nothing written.
		h.send(key(tea.KeyRight, 0))
		h.settle()
		if a := h.m.s.Models.Asked; a == nil || a.Effort != "max" || !a.NextTurn || len(w.method("thread/settings/update")) != 1 {
			t.Errorf("the second →: asked %+v, updates %q", a, w.method("thread/settings/update"))
		}
	})
}
