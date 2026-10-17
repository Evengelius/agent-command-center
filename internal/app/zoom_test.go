package app

import (
	"strings"
	"testing"
	"testing/synctest"

	tea "charm.land/bubbletea/v2"
)

// Plan 4, step 8, checks 6 and 7: zoom, the key overlay, and Tab between
// zones.

var zoneTitles = map[string]string{"A": "A · ORCHESTRATOR", "B": "B · CONVERSATION", "D": "D · QUOTA & TELEMETRY"}

// focusZone moves the focus with F6 until it is z.
func focusZone(t *testing.T, h *harness, z string) {
	t.Helper()
	for range zones {
		if h.m.ui.Focus == z {
			return
		}
		h.draw(key(tea.KeyF6, 0))
	}
	t.Fatalf("focus %q never reached", z)
}

// z on A, C or D draws that zone alone under the bars; z or Esc restores
// the exact frame, at the three sizes; F6 to B leaves the zoom.
func TestZoom(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := mockDash(t, "codex-subagents.jsonl")
		for _, size := range [][2]int{{80, 24}, {120, 40}, {200, 60}} {
			h.draw(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
			for _, z := range []string{"A", "C", "D"} {
				focusZone(t, h, z)
				before := h.frame()
				h.draw(typed('z'))
				zoomed := h.frame()
				title := zoneTitles[z]
				if z == "C" {
					title = "C · "
				}
				for other, ot := range zoneTitles {
					if other != z && strings.Contains(zoomed, ot) {
						t.Errorf("%dx%d %s zoomed shows %q:\n%s", size[0], size[1], z, ot, zoomed)
					}
				}
				rows := strings.Split(zoomed, "\n")
				if !strings.Contains(zoomed, title) || len(rows) != size[1] || !strings.Contains(rows[len(rows)-1], "z or Esc restore") {
					t.Errorf("%dx%d %s zoomed:\n%s", size[0], size[1], z, zoomed)
				}
				if !strings.Contains(rows[2], title) || !strings.Contains(rows[len(rows)-2], "╰") {
					t.Errorf("%dx%d %s: the zone does not span rows 2 to %d:\n%s", size[0], size[1], z, len(rows)-2, zoomed)
				}
				h.draw(typed('z'))
				if f := h.frame(); f != before {
					t.Errorf("%dx%d %s: z did not restore the frame:\n%s\nwant\n%s", size[0], size[1], z, f, before)
				}
				h.draw(typed('z'))
				h.draw(key(tea.KeyEscape, 0))
				if f := h.frame(); f != before || h.m.ui.Focus != z {
					t.Errorf("%dx%d %s: Esc gave focus %q and frame:\n%s", size[0], size[1], z, h.m.ui.Focus, f)
				}
			}
			h.draw(typed('z')) // D zoomed; F6 reaches B
			focusZone(t, h, "B")
			if h.m.ui.Zoom || !strings.Contains(h.frame(), "B · CONVERSATION") {
				t.Errorf("%dx%d: B reached, zoom %v", size[0], size[1], h.m.ui.Zoom)
			}
		}

		// In B, z is typed.
		h.draw(typed('z'))
		if h.m.ui.Zoom || h.m.edit.Text() != "z" {
			t.Errorf("z in B: zoom %v, editor %q", h.m.ui.Zoom, h.m.edit.Text())
		}
		h.draw(key(tea.KeyBackspace, 0))

		// Enter in zoomed C inspects; Esc returns to the zoom.
		h.draw(tea.WindowSizeMsg{Width: 120, Height: 40})
		focusZone(t, h, "C")
		h.draw(typed('z'))
		h.draw(key(tea.KeyDown, 0))
		zoomed := h.frame()
		h.draw(key(tea.KeyEnter, 0))
		if h.m.ui.Inspect == "" || !strings.Contains(h.frame(), "◀ back (Esc)") {
			t.Fatalf("Enter in zoomed C: inspect %q", h.m.ui.Inspect)
		}
		h.draw(key(tea.KeyEscape, 0))
		if f := h.frame(); f != zoomed {
			t.Errorf("Esc from the inspection:\n%s\nwant the zoom\n%s", f, zoomed)
		}
	})
}

// F1 anywhere and ? outside the editor show the overlay; it scrolls at
// 80x24 to every row; Esc gives back the exact view under it.
func TestKeyOverlay(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := mockDash(t, "codex-subagents.jsonl")
		h.draw(tea.WindowSizeMsg{Width: 80, Height: 24})
		before := h.frame()
		h.draw(key(tea.KeyF1, 0))
		f := h.frame()
		if !h.m.ui.Keys || !strings.Contains(f, "KEYS") || !strings.Contains(f, "ANYWHERE") || !strings.Contains(f, "↓ ") {
			t.Fatalf("F1 at 80x24:\n%s", f)
		}
		seen := f
		for range 20 {
			h.draw(key(tea.KeyPgDown, 0))
			seen += h.frame()
		}
		for _, want := range []string{"Ctrl+Q", "A FOCUSED PANEL", "zoom the panel", "THE APPROVAL CARD", "THE EDITOR", "insert a tab"} {
			if !strings.Contains(seen, want) {
				t.Errorf("scrolling the overlay never shows %q", want)
			}
		}
		h.draw(key(tea.KeyEscape, 0))
		if f := h.frame(); h.m.ui.Keys || f != before {
			t.Errorf("Esc: overlay %v, frame:\n%s", h.m.ui.Keys, f)
		}

		// ? in the editor is typed; in C it opens the overlay, F1 closes it.
		h.draw(typed('?'))
		if h.m.ui.Keys || h.m.edit.Text() != "?" {
			t.Errorf("? in B: overlay %v, editor %q", h.m.ui.Keys, h.m.edit.Text())
		}
		h.draw(key(tea.KeyBackspace, 0))
		focusZone(t, h, "C")
		h.draw(typed('?'))
		if !h.m.ui.Keys {
			t.Errorf("? in C: no overlay")
		}
		h.draw(key(tea.KeyF1, 0))
		if h.m.ui.Keys || h.m.ui.Focus != "C" {
			t.Errorf("F1 again: overlay %v focus %q", h.m.ui.Keys, h.m.ui.Focus)
		}

		// Over a held inspection: Esc gives the inspection back where it was.
		h.draw(key(tea.KeyEnter, 0))
		h.draw(typed('4'))
		h.draw(key(tea.KeyUp, 0))
		held := h.frame()
		top, hold := h.m.ui.InspectTop, h.m.ui.InspectHold
		h.draw(key(tea.KeyF1, 0))
		h.draw(key(tea.KeyPgDown, 0))
		h.draw(key(tea.KeyEscape, 0))
		if f := h.frame(); f != held || h.m.ui.InspectTop != top || h.m.ui.InspectHold != hold {
			t.Errorf("after the overlay: top %d hold %v, want %d %v; frame:\n%s", h.m.ui.InspectTop, h.m.ui.InspectHold, top, hold, f)
		}
	})
}

// Tab and Shift+Tab move between zones outside the editor; in it, Tab
// inserts a tab.
func TestTabZones(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := mockDash(t, "codex-subagents.jsonl")
		h.draw(key(tea.KeyTab, 0))
		if h.m.ui.Focus != "B" || h.m.edit.Text() != "\t" {
			t.Fatalf("Tab in B: focus %q editor %q", h.m.ui.Focus, h.m.edit.Text())
		}
		h.draw(key(tea.KeyBackspace, 0))
		focusZone(t, h, "C")
		for _, step := range []struct {
			k    tea.KeyPressMsg
			want string
		}{{key(tea.KeyTab, 0), "A"}, {key(tea.KeyTab, 0), "D"}, {key(tea.KeyTab, tea.ModShift), "A"}, {key(tea.KeyTab, tea.ModShift), "C"}, {key(tea.KeyTab, tea.ModShift), "B"}} {
			h.draw(step.k)
			if h.m.ui.Focus != step.want {
				t.Errorf("%q: focus %q, want %q", step.k.String(), h.m.ui.Focus, step.want)
			}
		}
		if h.m.edit.Text() != "" {
			t.Errorf("editor %q", h.m.edit.Text())
		}
	})
}
