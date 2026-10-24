package app

import (
	"testing"
	"testing/synctest"

	tea "charm.land/bubbletea/v2"
)

// Plan 36 (brief amendment 37): the selector's cursor follows the pointer
// onto another row, not the terminal's reports of a pointer at rest.

// TestSelectorPointerAtRest is criteria 1 to 3. Opened by Alt+P, the
// selector gets the probe's three reports of a pointer resting on the
// Ultracode toggle (its cell, a cell to the right, back): the cursor stays
// on the model in force, and Enter asks for that model, not Ultracode.
// Then a report over Sonnet 5.5 moves the cursor there and one over the
// toggle moves it back; after ↑ a report on the toggle again leaves it.
func TestSelectorPointerAtRest(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, s := selectorRig(t)
		h.send(key('p', tea.ModAlt))
		tx, ty := at(t, h, halfLeft, 0) // the text toggle's left end
		for _, dx := range []int{1, 2, 1} {
			mouseAt(t, h, tea.MouseMotionMsg{X: tx + dx, Y: ty})
		}
		if c := h.m.ui.Models.Cursor; c != 2 {
			t.Fatalf("after the reports of a pointer at rest the cursor is on row %d, want 2 (Fable 5.1)", c)
		}
		h.send(key(tea.KeyEnter, 0))
		if got := s.modelChanges(); len(got) != 1 || got[0].Model != "fable" || got[0].Ultracode != nil {
			t.Fatalf("Enter asked for %+v, want Fable 5.1 alone", got)
		}

		sx, sy := at(t, h, "Sonnet 5.5", 0)
		mouseAt(t, h, tea.MouseMotionMsg{X: sx + 2, Y: sy})
		if c := h.m.ui.Models.Cursor; c != 3 {
			t.Errorf("a report over Sonnet 5.5: cursor on row %d, want 3", c)
		}
		mouseAt(t, h, tea.MouseMotionMsg{X: tx + 1, Y: ty})
		if c := h.m.ui.Models.Cursor; c != 7 {
			t.Errorf("a report back over the toggle: cursor on row %d, want 7 (Ultracode)", c)
		}
		h.send(key(tea.KeyUp, 0))
		mouseAt(t, h, tea.MouseMotionMsg{X: tx + 2, Y: ty})
		if c := h.m.ui.Models.Cursor; c != 6 {
			t.Errorf("↑, then a report on the same row: cursor on row %d, want 6 (Effort)", c)
		}
	})
}

// TestSelectorPointerLevels is criterion 5: with the effort list open, a
// report over another level moves the list's cursor there; after ↓, a
// report again over the same level leaves it.
func TestSelectorPointerLevels(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, _ := selectorRig(t)
		h.send(key('p', tea.ModAlt))
		for range 4 {
			h.send(key(tea.KeyDown, 0)) // Sonnet 5.5, Haiku 5.5, More models, Effort
		}
		h.send(key(tea.KeyEnter, 0))
		if !h.m.ui.Models.Effort {
			t.Fatalf("Enter on Effort: %+v", h.m.ui.Models)
		}
		ax, ay := at(t, h, "ORCHESTRATOR", 0)
		mouseAt(t, h, tea.MouseMotionMsg{X: ax, Y: ay}) // zone A: the pointer outside the selector
		lx, ly := at(t, h, "Low", 0)
		mouseAt(t, h, tea.MouseMotionMsg{X: lx + 1, Y: ly})
		if l := h.m.ui.Models.Level; l != 0 {
			t.Fatalf("a report over Low: level %d, want 0", l)
		}
		h.send(key(tea.KeyDown, 0))
		mouseAt(t, h, tea.MouseMotionMsg{X: lx + 2, Y: ly})
		if l := h.m.ui.Models.Level; l != 1 {
			t.Errorf("↓, then a report on Low again: level %d, want 1 (Medium)", l)
		}
	})
}
