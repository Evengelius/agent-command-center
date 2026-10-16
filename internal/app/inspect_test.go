package app

import (
	"path/filepath"
	"strings"
	"testing"
	"testing/synctest"

	tea "charm.land/bubbletea/v2"

	"github.com/Evengelius/agent-command-center/internal/render"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// Plan 4, step 6, checks 2 and 5: zone A's cursor and folds, zone C's
// order, and the inspection's keys.

func mockDash(t *testing.T, file string) *harness {
	h := newHarness(t, Options{Config: Config{Mock: filepath.Join("..", "..", "testdata", "mock", file)}})
	h.settle()
	h.send(tea.WindowSizeMsg{Width: 120, Height: 40})
	if s := h.m.State(); s.Screen != state.ScreenDashboard {
		t.Fatalf("screen %q", s.Screen)
	}
	return h
}

func (h *harness) frame() string { return plainText(h.m.View().Content) }

// draw sends msg and draws the frame, as the program does between messages.
func (h *harness) draw(msg tea.Msg) {
	h.send(msg)
	h.m.View()
}

// From zone C: Enter inspects the selected agent, [ and ] go through zone
// C's order, the tabs change, and Esc returns to zone C.
func TestInspectKeysFromC(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := mockDash(t, "codex-subagents.jsonl")
		order := render.AgentOrder(h.m.State())
		if len(order) != 2 {
			t.Fatalf("order %v", order)
		}
		h.send(key(tea.KeyF6, 0))
		if h.m.ui.Focus != "C" {
			t.Fatalf("focus %q", h.m.ui.Focus)
		}
		h.send(key(tea.KeyDown, 0))
		h.send(key(tea.KeyEnter, 0))
		if h.m.ui.Inspect != order[0] || h.m.ui.Tab != 0 {
			t.Fatalf("inspect %q tab %d, want %q 0", h.m.ui.Inspect, h.m.ui.Tab, order[0])
		}
		if f := h.frame(); !strings.Contains(f, "◀ back (Esc)") || !strings.Contains(f, "Raw events") || strings.Contains(f, "B · CONVERSATION") {
			t.Fatalf("inspection frame:\n%s", f)
		}
		h.send(typed(']'))
		if h.m.ui.Inspect != order[1] || h.m.ui.Selected != order[1] {
			t.Errorf("] inspects %q selects %q, want %q", h.m.ui.Inspect, h.m.ui.Selected, order[1])
		}
		h.send(typed('['))
		if h.m.ui.Inspect != order[0] {
			t.Errorf("[ inspects %q, want %q", h.m.ui.Inspect, order[0])
		}
		for _, k := range []struct {
			k   tea.KeyPressMsg
			tab int
		}{{key(tea.KeyRight, 0), 1}, {key(tea.KeyRight, 0), 2}, {typed('4'), 3}, {key(tea.KeyRight, 0), 0}, {key(tea.KeyLeft, 0), 3}, {typed('1'), 0}} {
			h.send(k.k)
			if h.m.ui.Tab != k.tab {
				t.Errorf("after %q tab %d, want %d", k.k.String(), h.m.ui.Tab, k.tab)
			}
		}
		h.send(typed('x')) // dropped: it never reaches the editor behind
		if h.m.edit.Text() != "" {
			t.Errorf("editor %q", h.m.edit.Text())
		}
		h.send(key(tea.KeyEscape, 0))
		if h.m.ui.Inspect != "" || h.m.ui.Focus != "C" {
			t.Errorf("after Esc: inspect %q focus %q, want none and C", h.m.ui.Inspect, h.m.ui.Focus)
		}
		if f := h.frame(); !strings.Contains(f, "B · CONVERSATION") {
			t.Errorf("dashboard not back:\n%s", f)
		}
	})
}

// From zone A: ↑ ↓ move the cursor through the nodes, ← folds, → unfolds,
// Enter inspects an agent's node and leaves the orchestrator's to zone B.
func TestTreeKeys(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := mockDash(t, "codex-subagents.jsonl")
		h.send(key(tea.KeyF6, 0))
		h.send(key(tea.KeyF6, 0))
		if h.m.ui.Focus != "A" {
			t.Fatalf("focus %q", h.m.ui.Focus)
		}
		nodes := render.Tree(h.m.State(), h.m.ui)
		if len(nodes) != 5 || nodes[0].ID != "s:1" {
			t.Fatalf("nodes %+v", nodes)
		}
		h.send(key(tea.KeyUp, 0)) // no cursor yet: the last node
		if h.m.ui.ACursor != nodes[4].ID {
			t.Fatalf("cursor %q, want %q", h.m.ui.ACursor, nodes[4].ID)
		}
		h.send(key(tea.KeyEnter, 0)) // the orchestrator's tool
		if h.m.ui.Inspect != "" || !strings.Contains(h.m.ui.Toast, "zone B") {
			t.Errorf("Enter on a tool: inspect %q, toast %q", h.m.ui.Inspect, h.m.ui.Toast)
		}
		for range 3 {
			h.send(key(tea.KeyUp, 0))
		}
		if h.m.ui.ACursor != nodes[1].ID || nodes[1].Agent == "" {
			t.Fatalf("cursor %q, want the first child %q", h.m.ui.ACursor, nodes[1].ID)
		}
		h.send(key(tea.KeyEnter, 0))
		if h.m.ui.Inspect != nodes[1].Agent {
			t.Fatalf("inspect %q, want %q", h.m.ui.Inspect, nodes[1].Agent)
		}
		h.send(key(tea.KeyEscape, 0))
		if h.m.ui.Inspect != "" || h.m.ui.Focus != "A" || h.m.ui.ACursor != nodes[1].ID {
			t.Errorf("after Esc: inspect %q focus %q cursor %q", h.m.ui.Inspect, h.m.ui.Focus, h.m.ui.ACursor)
		}
		h.send(key(tea.KeyUp, 0))
		h.send(key(tea.KeyUp, 0)) // stays on the first node
		h.send(key(tea.KeyLeft, 0))
		if got := render.Tree(h.m.State(), h.m.ui); len(got) != 1 || !got[0].Folded {
			t.Errorf("stage folded: nodes %+v", got)
		}
		if f := h.frame(); strings.Contains(f, "/root/list_files ") && strings.Contains(f, "├─ ✓ child") {
			t.Errorf("folded stage still shows its children:\n%s", f)
		}
		h.send(key(tea.KeyRight, 0))
		if got := render.Tree(h.m.State(), h.m.ui); len(got) != 5 {
			t.Errorf("stage unfolded: %d nodes, want 5", len(got))
		}
		if h.m.ui.ACursor != "s:1" {
			t.Errorf("cursor %q, want s:1", h.m.ui.ACursor)
		}
	})
}

// The inspection scrolls on its own: ↑ holds it, End follows again.
func TestInspectScroll(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := mockDash(t, "codex-subagents.jsonl")
		h.draw(tea.WindowSizeMsg{Width: 80, Height: 24})
		h.draw(key(tea.KeyF6, 0))
		h.draw(key(tea.KeyEnter, 0))
		h.draw(typed('4')) // Raw events
		v := h.m.screen.Inspect
		if v.Rows <= v.Room {
			t.Fatalf("raw tab %d rows in %d: nothing to scroll", v.Rows, v.Room)
		}
		h.draw(key(tea.KeyUp, 0))
		if !h.m.ui.InspectHold || h.m.screen.Inspect.Top != v.Top-1 {
			t.Errorf("↑: hold %v top %d, want true %d", h.m.ui.InspectHold, h.m.screen.Inspect.Top, v.Top-1)
		}
		if f := h.frame(); !strings.Contains(f, "↓ 1 more") {
			t.Errorf("held frame lacks '↓ 1 more'")
		}
		h.draw(key(tea.KeyHome, 0))
		if h.m.screen.Inspect.Top != 0 {
			t.Errorf("Home: top %d", h.m.screen.Inspect.Top)
		}
		h.draw(tea.MouseWheelMsg{X: 10, Y: 8, Button: tea.MouseWheelDown})
		if h.m.screen.Inspect.Top != wheelRows {
			t.Errorf("wheel: top %d, want %d", h.m.screen.Inspect.Top, wheelRows)
		}
		h.draw(key(tea.KeyEnd, 0))
		if h.m.ui.InspectHold || h.m.screen.Inspect.Top != v.Top {
			t.Errorf("End: hold %v top %d, want false %d", h.m.ui.InspectHold, h.m.screen.Inspect.Top, v.Top)
		}
		if h.m.ui.ConvHold {
			t.Error("the inspection's scrolling held the conversation")
		}
	})
}
