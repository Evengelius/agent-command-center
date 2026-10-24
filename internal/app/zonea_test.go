package app

import (
	"strings"
	"testing"
	"testing/synctest"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/Evengelius/agent-command-center/internal/render"
)

// Plan 24 (brief amendment 22): the telemetry pane's mouse and keys, the
// overlay's and the footer's words, and the clock's redraws, through the
// application.

// aRow is the screen row of zone A's node id in the last frame.
func aRow(t *testing.T, h *harness, id string) int {
	t.Helper()
	for y, n := range h.m.screen.ARows {
		if n == id {
			return y
		}
	}
	t.Fatalf("no row of %s in zone A:\n%s", id, h.frame())
	return 0
}

// TestTelemetryMouse is criterion 24.23's mouse half: a click on TELEMETRY
// opens the details, on SESSION USAGE folds and unfolds the section; a
// click on a tree node moves the cursor.
func TestTelemetryMouse(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := mockDash(t, "claude-quota-windows.jsonl")
		h.m.View()
		y := aRow(t, h, render.NodeUsage)
		h.draw(click(3, y))
		h.draw(tea.MouseReleaseMsg{X: 3, Y: y, Button: tea.MouseLeft})
		if f := h.frame(); h.m.ui.Focus != "A" || h.m.ui.ACursor != render.NodeUsage || !h.m.ui.Fold[render.NodeUsage] || !strings.Contains(f, "▸ SESSION USAGE") || strings.Contains(f, "CONTEXT") {
			t.Errorf("after a click on SESSION USAGE: focus %q cursor %q fold %v:\n%s", h.m.ui.Focus, h.m.ui.ACursor, h.m.ui.Fold, f)
		}
		y = aRow(t, h, render.NodeUsage)
		h.draw(click(3, y))
		h.draw(tea.MouseReleaseMsg{X: 3, Y: y, Button: tea.MouseLeft})
		if f := h.frame(); h.m.ui.Fold[render.NodeUsage] || !strings.Contains(f, "▾ SESSION USAGE") || !strings.Contains(f, "Context ") {
			t.Errorf("after a second click: fold %v:\n%s", h.m.ui.Fold, f)
		}
		y = aRow(t, h, "s:1")
		h.draw(click(3, y))
		h.draw(tea.MouseReleaseMsg{X: 3, Y: y, Button: tea.MouseLeft})
		if h.m.ui.ACursor != "s:1" || h.m.ui.Details {
			t.Errorf("after a click on turn 1: cursor %q details %v", h.m.ui.ACursor, h.m.ui.Details)
		}
		y = aRow(t, h, render.NodeTelemetry)
		h.draw(click(3, y))
		h.draw(tea.MouseReleaseMsg{X: 3, Y: y, Button: tea.MouseLeft})
		if f := h.frame(); !h.m.ui.Details || !strings.Contains(f, "ERRORS AND TELEMETRY") || !strings.Contains(f, "quota.7d.fable") {
			t.Errorf("after a click on TELEMETRY: details %v:\n%s", h.m.ui.Details, f)
		}
		h.draw(key(tea.KeyEscape, 0))
		if h.m.ui.Details || h.m.ui.Focus != "A" || h.m.ui.ACursor != render.NodeTelemetry {
			t.Errorf("after Esc: details %v focus %q cursor %q", h.m.ui.Details, h.m.ui.Focus, h.m.ui.ACursor)
		}
	})
}

// TestZoneAKeyWords is criterion 24.24: the overlay's and the footer's
// words name the three zones and the details; since plan 43 (design D7) the
// Explorer's panel among them while shown.
func TestZoneAKeyWords(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := mockDash(t, "claude-subagent-approval.jsonl")
		h.draw(tea.WindowSizeMsg{Width: 200, Height: 60})
		h.draw(key(tea.KeyF1, 0))
		seen := h.frame()
		for range 10 {
			h.draw(key(tea.KeyPgDown, 0))
			seen += h.frame()
		}
		for _, want := range []string{"next / previous zone (B, the Explorer while shown, C, A)", "in A on TELEMETRY, the errors and every value", "A FOCUSED PANEL (A, C, INSPECTION)"} {
			if !strings.Contains(seen, want) {
				t.Errorf("the overlay lacks %q", want)
			}
		}
		if strings.Contains(seen, "B, C, A, D") || strings.Contains(seen, "in D,") {
			t.Errorf("the overlay still names zone D")
		}
		h.draw(key(tea.KeyEscape, 0))
		focusZone(t, h, "A")
		rows := strings.Split(h.frame(), "\n")
		if footer := rows[len(rows)-1]; !strings.Contains(footer, "↑↓ move · ←→ fold · Enter inspect or details · z zoom · PgUp PgDn Home End scroll · F6 zone · Esc back to the input · F1 keys") {
			t.Errorf("the footer with A focused: %q", footer)
		}
	})
}

// TestClockRedraw is criterion 24.29 through the application: the clock's
// tick a second later changes rows of zone A only; a minute later its
// countdown changed, and no row of B or C.
func TestClockRedraw(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := mockDash(t, "claude-quota-windows.jsonl")
		// At 160x60 zone A's pane keeps its full form, "Resets in" rows and
		// all (plan 27's zone C takes more rows at 120x40).
		h.send(tea.WindowSizeMsg{Width: 160, Height: 60})
		base := strings.Split(h.frame(), "\n")
		aw := 45 // zone A's width at 160 columns
		aBottom := 2
		for y := 3; y < len(base); y++ { // zone C's top edge ends zone A
			if strings.HasPrefix(base[y], "╭") {
				aBottom = y - 1
				break
			}
		}
		for _, c := range []struct {
			name string
			d    time.Duration
		}{{"a second later", time.Second}, {"a minute later", time.Minute}} {
			at := testNow.Add(c.d)
			h.m.opt.Now = func() time.Time { return at }
			h.draw(clockMsg{})
			later := strings.Split(h.frame(), "\n")
			changed := 0
			for y := range base {
				if base[y] == later[y] {
					continue
				}
				rb, rl := []rune(base[y]), []rune(later[y])
				if y > aBottom || len(rb) < aw || len(rl) < aw || string(rb[aw:]) != string(rl[aw:]) {
					t.Errorf("%s: row %d differs outside zone A:\n%s\n%s", c.name, y, base[y], later[y])
				}
				changed++
			}
			if changed == 0 {
				t.Errorf("%s: no row of zone A changed", c.name)
			}
			if c.d == time.Minute && !strings.Contains(strings.Join(later, "\n"), "Resets in 4h 11m") {
				t.Errorf("a minute later, the countdown did not change:\n%s", strings.Join(later, "\n"))
			}
		}
	})
}
