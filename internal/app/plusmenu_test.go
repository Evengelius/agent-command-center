package app

import (
	"strings"
	"testing"
	"testing/synctest"

	tea "charm.land/bubbletea/v2"
)

// TestPlusMenuMouse is plan 30's criterion 30.2 (brief amendment 30): a
// click on the editor row's band opens the "+" menu, the pointer's motion
// sets its hover, a click on its item closes it and does nothing else, a
// click on the band again opens it, Esc closes it and is consumed, a click
// elsewhere closes it and goes on, any other key closes it and acts; the
// view asks for every mouse motion while the menu is open, and since plan
// 43 (design D4) while the sidebar shows, as at 120x40 here (under 100
// columns, cell motion with the menu closed: TestSideMotionMode).
func TestPlusMenuMouse(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r, _, _ := selectRig(t)
		h := r.h
		h.send(tea.PasteMsg{Content: "draft"})
		frame := func() string { return plainText(h.m.View().Content) }
		if h.m.View().MouseMode != tea.MouseModeAllMotion {
			t.Fatalf("mouse mode %v before the menu, the sidebar shown, want all motion", h.m.View().MouseMode)
		}
		x, y := at(t, h, "+     ", 0) // the band, then the gap before the pill
		press := func(px, py int) {
			mouseAt(t, h, click(px, py))
			mouseAt(t, h, tea.MouseReleaseMsg{X: px, Y: py, Button: tea.MouseLeft})
		}
		press(x+1, y)
		if !h.m.ui.Plus || !strings.Contains(frame(), "Upload from computer") {
			t.Fatalf("a click on the band did not open the menu:\n%s", frame())
		}
		if h.m.View().MouseMode != tea.MouseModeAllMotion {
			t.Errorf("mouse mode %v with the menu open, want all motion", h.m.View().MouseMode)
		}
		ix, iy := at(t, h, "Upload from computer", 0)
		mouseAt(t, h, tea.MouseMotionMsg{X: ix + 3, Y: iy})
		if !h.m.ui.PlusHover {
			t.Error("the pointer over the item: no hover")
		}
		mouseAt(t, h, tea.MouseMotionMsg{X: ix + 3, Y: iy + 5})
		if h.m.ui.PlusHover {
			t.Error("the pointer off the item: the hover stays")
		}
		// A click on the item closes the menu and does nothing else.
		sent := len(r.sent())
		press(ix+3, iy)
		if h.m.ui.Plus || strings.Contains(frame(), "Upload from computer") || h.m.edit.Text() != "draft" || len(r.sent()) != sent || h.m.ui.Focus != "B" {
			t.Errorf("a click on the item: open %v, draft %q, %d sent, focus %q", h.m.ui.Plus, h.m.edit.Text(), len(r.sent()), h.m.ui.Focus)
		}
		if h.m.View().MouseMode != tea.MouseModeAllMotion {
			t.Errorf("mouse mode %v after the menu, the sidebar shown, want all motion", h.m.View().MouseMode)
		}
		// The band again, then Esc.
		press(x+1, y)
		if !h.m.ui.Plus {
			t.Fatal("a second click on the band did not open the menu")
		}
		h.send(key(tea.KeyEscape, 0))
		if h.m.ui.Plus || h.m.edit.Text() != "draft" || strings.Contains(frame(), "Upload from computer") {
			t.Errorf("Esc: open %v, draft %q", h.m.ui.Plus, h.m.edit.Text())
		}
		// Open, a click in zone C closes it and focuses C.
		press(x+1, y)
		cx, cy := at(t, h, "no agents yet", 0)
		press(cx, cy)
		if h.m.ui.Plus || h.m.ui.Focus != "C" {
			t.Errorf("a click in C with the menu open: open %v, focus %q", h.m.ui.Plus, h.m.ui.Focus)
		}
		// Open, typing closes it and the key acts.
		press(x+1, y)
		if !h.m.ui.Plus || h.m.ui.Focus != "B" {
			t.Fatalf("the band: open %v, focus %q", h.m.ui.Plus, h.m.ui.Focus)
		}
		h.send(typed('a'))
		if h.m.ui.Plus || h.m.edit.Text() != "drafta" {
			t.Errorf("typing with the menu open: open %v, draft %q", h.m.ui.Plus, h.m.edit.Text())
		}
	})
}
