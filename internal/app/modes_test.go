package app

import (
	"fmt"
	"slices"
	"strings"
	"testing"
	"testing/synctest"

	tea "charm.land/bubbletea/v2"

	"github.com/Evengelius/agent-command-center/internal/event"
	"github.com/Evengelius/agent-command-center/internal/render"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// Plan 39 (brief amendment 40): the modes selector worked with the keys and
// the mouse over a scripted session, which records each mode the
// application asks for.

// modesRig is a ready scripted session of cli at 120x40, mode in force, a
// draft typed.
func modesRig(t *testing.T, cli, mode string) (*harness, *scriptCLI) {
	t.Helper()
	h, c := readyScript(t, cli, true, event.Started{SessionID: "s1", Version: "1.0", ApprovalMode: mode})
	h.send(tea.WindowSizeMsg{Width: 120, Height: 40})
	h.send(tea.PasteMsg{Content: "a draft kept"})
	return h, c
}

// modesCell is the first cell on the screen that carries the modes
// selector's item, the frame drawn first.
func modesCell(t *testing.T, h *harness, item int) (x, y int) {
	t.Helper()
	h.m.View()
	for y := range 60 {
		for x := range 200 {
			if p, it := h.m.screen.ItemAt(x, y); p == render.ModesPanel && it == item {
				return x, y
			}
		}
	}
	t.Fatalf("no cell of item %d:\n%s", item, h.frame())
	return 0, 0
}

// pillCell is the mode pill's first cell.
func pillCell(t *testing.T, h *harness) (x, y int) {
	t.Helper()
	h.m.View()
	for y := range 60 {
		for x := range 200 {
			if p, _ := h.m.screen.ItemAt(x, y); p == render.ModePillPanel {
				return x, y
			}
		}
	}
	t.Fatalf("no mode pill:\n%s", h.frame())
	return 0, 0
}

// TestModesKeys is criterion 39.10's keys: F4 opens the selector on the
// mode in force; ↑ ↓ Home End move the cursor, never past the ends; Enter
// and Space choose and close it, the pill keeping the mode in force until
// the CLI reports the new one; Esc and F4 close it; any other key closes it
// and goes on to the editor. The draft is never sent, the focus stays.
func TestModesKeys(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, cli := modesRig(t, "claude-code", event.ModeAuto)
		h.send(key(tea.KeyF4, 0))
		if v := h.m.ui.Modes; !v.Open || v.Cursor != 3 {
			t.Fatalf("F4: %+v, want open on Auto", v)
		}
		if f := h.frame(); !strings.Contains(f, "MODES") || !strings.Contains(f, "Current session") || !strings.Contains(f, "Edit automatically") {
			t.Fatalf("the selector:\n%s", f)
		}
		for _, c := range []struct {
			k    rune
			want int
		}{{tea.KeyDown, 3}, {tea.KeyUp, 2}, {tea.KeyHome, 0}, {tea.KeyUp, 0}, {tea.KeyEnd, 3}, {tea.KeyUp, 2}, {tea.KeyUp, 1}} {
			h.send(key(c.k, 0))
			if got := h.m.ui.Modes.Cursor; got != c.want {
				t.Fatalf("key %v: cursor %d, want %d", c.k, got, c.want)
			}
		}
		h.send(key(tea.KeyEnter, 0))
		if got := modesOf(cli); !slices.Equal(got, []string{state.ModeAcceptEdits}) || h.m.ui.Modes.Open {
			t.Fatalf("Enter: asked %v, open %v", got, h.m.ui.Modes.Open)
		}
		if f := h.frame(); !strings.Contains(f, "↯ Auto") || strings.Contains(f, "</> Edit automatically") {
			t.Errorf("the pill changed before the CLI answered:\n%s", f)
		}
		pushEvents(h, "claude-code", evOf(event.ApprovalMode, event.Mode{Mode: event.ModeAcceptEdits, Answers: true}))
		if f := h.frame(); !strings.Contains(f, "</> Edit automatically") {
			t.Errorf("the pill after the answer:\n%s", f)
		}
		h.send(key(tea.KeyF4, 0))
		if v := h.m.ui.Modes; !v.Open || v.Cursor != 1 {
			t.Fatalf("F4 again: %+v, want the cursor on Edit automatically", v)
		}
		h.send(key(tea.KeyDown, 0))
		h.send(key(tea.KeySpace, 0))
		if got := modesOf(cli); !slices.Equal(got, []string{state.ModeAcceptEdits, state.ModePlan}) || h.m.ui.Modes.Open {
			t.Fatalf("Space: asked %v", got)
		}
		for _, k := range []rune{tea.KeyEsc, tea.KeyF4} {
			h.send(key(tea.KeyF4, 0))
			h.send(key(k, 0))
			if h.m.ui.Modes.Open {
				t.Errorf("%v did not close the selector", k)
			}
		}
		h.send(key(tea.KeyF4, 0))
		h.send(typed('x'))
		if h.m.ui.Modes.Open || h.m.edit.Text() != "a draft keptx" {
			t.Errorf("another key: open %v, draft %q", h.m.ui.Modes.Open, h.m.edit.Text())
		}
		h.send(key(tea.KeyF4, 0))
		h.send(tea.PasteMsg{Content: "y"})
		if h.m.ui.Modes.Open || h.m.edit.Text() != "a draft keptxy" {
			t.Errorf("a paste: open %v, draft %q", h.m.ui.Modes.Open, h.m.edit.Text())
		}
		if sent, _, _ := cli.sess.calls(); len(sent) != 0 || h.m.ui.Focus != "B" || len(modesOf(cli)) != 2 {
			t.Errorf("sent %q, focus %q, asked %v", sent, h.m.ui.Focus, modesOf(cli))
		}
	})
}

// TestModesMouse is criterion 39.10's mouse: a click on the pill opens the
// selector; the pointer's moving onto an option moves the cursor there, a
// half row leaves it, the heading, the frame and the outside put it back on
// the mode in force; a click on an option chooses it; one on the frame
// does nothing; the wheel over the box scrolls nothing; a click outside
// closes it and goes on.
func TestModesMouse(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, cli := modesRig(t, "claude-code", event.ModeManual)
		px, py := pillCell(t, h)
		clickAt(t, h, px+1, py)
		if v := h.m.ui.Modes; !v.Open || v.Cursor != 0 || !v.PointerKnown || v.Pointer != -1 {
			t.Fatalf("a click on the pill: %+v", v)
		}
		move := func(x, y int) { mouseAt(t, h, tea.MouseMotionMsg{X: x, Y: y}) }
		planX, planY := modesCell(t, h, render.ItemModeOpt+2)
		for _, c := range []struct {
			what string
			x, y int
			want int
		}{
			{"Plan's title", planX + 12, planY, 2},
			{"Plan's description", planX + 14, planY + 1, 2},
			{"the half row under Plan", planX + 12, planY + 3, 2},
			{"Auto", planX + 12, planY + 4, 3},
			{"the heading", planX + 4, planY - 10, 0},
			{"Edit automatically", planX + 12, planY - 4, 1},
			{"the box's edge", planX - 1 + 0, planY, 0},
			{"Plan again", planX + 12, planY, 2},
			{"zone A, outside", 3, 10, 0},
		} {
			move(c.x, c.y)
			if got := h.m.ui.Modes.Cursor; got != c.want {
				t.Fatalf("the pointer onto %s (%d,%d): cursor %d, want %d", c.what, c.x, c.y, got, c.want)
			}
		}
		// The frame: nothing; the wheel: nothing.
		fx, fy := modesCell(t, h, render.ItemModesFrame)
		clickAt(t, h, fx, fy)
		mouseAt(t, h, tea.MouseWheelMsg{X: planX + 12, Y: planY, Button: tea.MouseWheelUp})
		if !h.m.ui.Modes.Open || len(modesOf(cli)) != 0 || h.m.ui.ConvHold {
			t.Fatalf("frame click and wheel: open %v, asked %v, held %v", h.m.ui.Modes.Open, modesOf(cli), h.m.ui.ConvHold)
		}
		clickAt(t, h, planX+12, planY+5) // Auto's description
		if got := modesOf(cli); !slices.Equal(got, []string{state.ModeAuto}) || h.m.ui.Modes.Open {
			t.Fatalf("a click on Auto: asked %v, open %v", got, h.m.ui.Modes.Open)
		}
		// A click outside closes it and goes on: zone A takes the focus.
		clickAt(t, h, px+1, py)
		clickAt(t, h, 3, 10)
		if h.m.ui.Modes.Open || h.m.ui.Focus != "A" || len(modesOf(cli)) != 1 {
			t.Errorf("a click outside: open %v, focus %q, asked %v", h.m.ui.Modes.Open, h.m.ui.Focus, modesOf(cli))
		}
		// The pill's click closes it too.
		clickAt(t, h, px+1, py)
		clickAt(t, h, px+1, py)
		if h.m.ui.Modes.Open || h.m.edit.Text() != "a draft kept" {
			t.Errorf("the pill twice: open %v, draft %q", h.m.ui.Modes.Open, h.m.edit.Text())
		}
	})
}

// TestModesPointerAtRest is plan 36's rule in the modes selector: opened by
// F4, the first report of the pointer is where it rests, and reports that
// stay on its item move nothing; only the pointer onto another item does.
func TestModesPointerAtRest(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, cli := modesRig(t, "claude-code", event.ModeManual)
		h.send(key(tea.KeyF4, 0))
		x, y := modesCell(t, h, render.ItemModeOpt+2) // Plan, under the pointer at rest
		for _, dx := range []int{12, 13, 12} {
			mouseAt(t, h, tea.MouseMotionMsg{X: x + dx, Y: y})
		}
		if c := h.m.ui.Modes.Cursor; c != 0 {
			t.Fatalf("after the reports of a pointer at rest the cursor is on %d, want 0 (Manual)", c)
		}
		h.send(key(tea.KeyEnter, 0))
		if got := modesOf(cli); len(got) != 0 {
			t.Fatalf("Enter on the mode in force asked %v", got)
		}
		h.send(key(tea.KeyF4, 0))
		mouseAt(t, h, tea.MouseMotionMsg{X: x + 12, Y: y})
		mouseAt(t, h, tea.MouseMotionMsg{X: x + 12, Y: y + 4}) // onto Auto
		if c := h.m.ui.Modes.Cursor; c != 3 {
			t.Errorf("the pointer onto Auto: cursor %d, want 3", c)
		}
	})
}

// TestModesFullAccess is criterion 39.5 through the application: Full
// access first opens the confirmation, the cursor on Cancel; Cancel and
// Esc go back to the modes, Esc again and a click outside close; nothing
// is asked for until Turn on Full access; the second click of a double
// click on Full access lands on a half row or on Cancel; a mode the
// requirements leave out is greyed, skipped by the keys, and asks for
// nothing, zone B saying why.
func TestModesFullAccess(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, cli := modesRig(t, "codex", event.ModeManual)
		h.send(key(tea.KeyF4, 0))
		h.send(key(tea.KeyEnd, 0))
		h.send(key(tea.KeyEnter, 0))
		if v := h.m.ui.Modes; !v.Open || !v.Confirm || v.Cursor != 1 || v.PointerKnown {
			t.Fatalf("Enter on Full access: %+v", v)
		}
		if f := h.frame(); !strings.Contains(f, "TURN ON FULL ACCESS?") || !strings.Contains(f, "Keep Ask for approval") || strings.Contains(f, "MODES") {
			t.Fatalf("the confirmation:\n%s", f)
		}
		h.send(key(tea.KeyEnter, 0)) // Cancel
		if v := h.m.ui.Modes; !v.Open || v.Confirm || v.Cursor != 2 {
			t.Fatalf("Cancel: %+v, want the modes, the cursor on Full access", v)
		}
		h.send(key(tea.KeyEnter, 0))
		h.send(key(tea.KeyEsc, 0))
		if v := h.m.ui.Modes; !v.Open || v.Confirm {
			t.Fatalf("Esc in the confirmation: %+v", v)
		}
		h.send(key(tea.KeyEsc, 0))
		if h.m.ui.Modes.Open || len(modesOf(cli)) != 0 {
			t.Fatalf("Esc: open %v, asked %v", h.m.ui.Modes.Open, modesOf(cli))
		}
		// A double click on Full access's title: the second lands on a half
		// row; on its description: on Cancel.
		h.send(key(tea.KeyF4, 0))
		fx, fy := modesCell(t, h, render.ItemModeOpt+2)
		clickAt(t, h, fx+12, fy)
		clickAt(t, h, fx+12, fy)
		if v := h.m.ui.Modes; !v.Confirm || len(modesOf(cli)) != 0 {
			t.Fatalf("a double click on the title: %+v, asked %v", v, modesOf(cli))
		}
		if p, it := h.m.screen.ItemAt(fx+12, fy); p != render.ModesPanel || it != render.ItemModesHalf {
			t.Errorf("under Full access's title in the confirmation: %c %d", p, it)
		}
		if _, it := h.m.screen.ItemAt(fx+12, fy+1); it != render.ItemModeOpt+1 {
			t.Errorf("under Full access's description in the confirmation: item %d, want Cancel", it)
		}
		clickAt(t, h, fx+12, fy+1)
		if v := h.m.ui.Modes; !v.Open || v.Confirm || len(modesOf(cli)) != 0 {
			t.Fatalf("a click where Cancel lies: %+v, asked %v", v, modesOf(cli))
		}
		// A click outside the confirmation closes it.
		h.send(key(tea.KeyEnter, 0))
		clickAt(t, h, 3, 10)
		if h.m.ui.Modes.Open || len(modesOf(cli)) != 0 {
			t.Fatalf("a click outside: open %v, asked %v", h.m.ui.Modes.Open, modesOf(cli))
		}
		// Turn on Full access, by the keys.
		h.send(key(tea.KeyF4, 0))
		h.send(key(tea.KeyEnd, 0))
		h.send(key(tea.KeyEnter, 0))
		h.send(key(tea.KeyUp, 0))
		h.send(key(tea.KeyEnter, 0))
		if got := modesOf(cli); !slices.Equal(got, []string{state.ModeFull}) || h.m.ui.Modes.Open {
			t.Fatalf("Turn on Full access: asked %v, open %v", got, h.m.ui.Modes.Open)
		}
		if b := barLine(t, h); !strings.HasSuffix(b, "approvals Ask for approval → Full access …") {
			t.Errorf("bar %q", b)
		}
		pushEvents(h, "codex", evOf(event.ApprovalMode, event.Mode{Mode: event.ModeFull, Answers: true}))
		if f := h.frame(); !strings.Contains(f, "⚠ Full access") {
			t.Errorf("the pill:\n%s", f)
		}
		// Back, then the requirements leave Full access out.
		pushEvents(h, "codex", evOf(event.ApprovalMode, event.Mode{Mode: event.ModeManual}),
			evOf(event.ModesAllowed, event.ModeLimits{Unavailable: map[string]string{event.ModeFull: "Codex's requirements do not allow the approval policy never"}}))
		h.send(key(tea.KeyF4, 0))
		h.send(key(tea.KeyEnd, 0))
		if c := h.m.ui.Modes.Cursor; c != 1 {
			t.Errorf("End with Full access left out: cursor %d, want 1", c)
		}
		fx, fy = modesCell(t, h, render.ItemModeOpt+2)
		mouseAt(t, h, tea.MouseMotionMsg{X: fx + 12, Y: fy})
		if c := h.m.ui.Modes.Cursor; c != 1 {
			t.Errorf("the pointer onto the greyed Full access: cursor %d", c)
		}
		// A click on it asks for nothing and says why; the reason, too long
		// to sit beside the pills, has the editor row alone, so the pill and
		// the selector go (plan 29's rule, design D9).
		clickAt(t, h, fx+12, fy)
		if v := h.m.ui.Modes; v.Confirm || len(modesOf(cli)) != 1 {
			t.Errorf("a click on the greyed Full access: %+v, asked %v", v, modesOf(cli))
		}
		if s := h.m.State(); s.Turn.Notice != "Full access is not available: Codex's requirements do not allow the approval policy never" ||
			!strings.Contains(h.frame(), "Full access is not available: Codex's requirements do not allow") {
			t.Errorf("no reason in zone B: %q\n%s", s.Turn.Notice, h.frame())
		}
	})
}

// TestModesConfirmPointer is design D4's pointer rule: the confirmation
// takes the rows of the modes' box, so the pointer resting over Approve for
// me lies over Turn on Full access once the confirmation opens; its next
// report is where it rests, the cursor staying on Cancel, and Enter asks
// for nothing.
func TestModesConfirmPointer(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, cli := modesRig(t, "codex", event.ModeManual)
		h.send(key(tea.KeyF4, 0))
		ax, ay := modesCell(t, h, render.ItemModeOpt+1) // Approve for me
		mouseAt(t, h, tea.MouseMotionMsg{X: 3, Y: 10})  // where the pointer rests, outside
		mouseAt(t, h, tea.MouseMotionMsg{X: ax + 12, Y: ay})
		if c := h.m.ui.Modes.Cursor; c != 1 {
			t.Fatalf("the pointer onto Approve for me: cursor %d", c)
		}
		h.send(key(tea.KeyEnd, 0))
		h.send(key(tea.KeyEnter, 0))
		if v := h.m.ui.Modes; !v.Confirm || v.Cursor != 1 {
			t.Fatalf("the confirmation %+v", v)
		}
		h.m.View()
		if p, it := h.m.screen.ItemAt(ax+12, ay); p != render.ModesPanel || it != render.ItemModeOpt {
			t.Fatalf("under the resting pointer: %c %d, want Turn on Full access", p, it)
		}
		mouseAt(t, h, tea.MouseMotionMsg{X: ax + 13, Y: ay}) // a report of the pointer at rest
		if c := h.m.ui.Modes.Cursor; c != 1 {
			t.Fatalf("a resting pointer moved the cursor to %d", c)
		}
		h.send(key(tea.KeyEnter, 0))
		if v := h.m.ui.Modes; v.Confirm || len(modesOf(cli)) != 0 {
			t.Errorf("Enter on Cancel: %+v, asked %v", v, modesOf(cli))
		}
	})
}

// TestModesRoomToast is criterion 39.11 through the application: no
// selector where the box does not fit above the editor, nor without the
// pill on screen; a toast says why, and nothing is drawn of it. A frame
// that loses the room closes it.
func TestModesRoomToast(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, cli := modesRig(t, "claude-code", event.ModeManual)
		h.send(tea.WindowSizeMsg{Width: 80, Height: 24})
		h.send(key(tea.KeyF4, 0))
		if h.m.ui.Modes.Open || h.m.ui.Toast != "no room for the modes above the editor" {
			t.Fatalf("80x24: open %v, toast %q", h.m.ui.Modes.Open, h.m.ui.Toast)
		}
		h.send(tea.WindowSizeMsg{Width: 120, Height: 40})
		h.send(key(tea.KeyF4, 0))
		if !h.m.ui.Modes.Open {
			t.Fatal("120x40: closed")
		}
		h.send(tea.WindowSizeMsg{Width: 80, Height: 24})
		h.m.View()
		if h.m.ui.Modes.Open || strings.Contains(h.frame(), "MODES") {
			t.Errorf("the frame lost the room: open %v", h.m.ui.Modes.Open)
		}
		h.send(tea.WindowSizeMsg{Width: 120, Height: 40})
		h.send(key(tea.KeyF6, 0))
		h.send(key(tea.KeyF6, 0)) // zone A
		h.send(key(tea.KeyEnter, 0))
		h.send(key('z', 0)) // zone A full screen: no editor row
		h.m.View()
		h.send(key(tea.KeyF4, 0))
		if h.m.ui.Modes.Open || !strings.Contains(h.m.ui.Toast, "mode pill") && !strings.Contains(h.m.ui.Toast, "no room") {
			t.Errorf("a full view: open %v, toast %q", h.m.ui.Modes.Open, h.m.ui.Toast)
		}
		if len(modesOf(cli)) != 0 {
			t.Errorf("asked %v", modesOf(cli))
		}
	})
}

// TestModesBesideOthers: one popup at a time (plan 39, design D8): F4 with
// the model selector open closes it and opens the modes; Alt+P and the
// "+" band close the modes.
func TestModesBesideOthers(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, s := selectorRig(t)
		s.auto = true
		pushEvents(h, "claude-code", evOf(event.SessionStarted, event.Started{SessionID: "s1", ApprovalMode: event.ModeAuto}))
		h.send(key('p', tea.ModAlt))
		h.send(key(tea.KeyF4, 0))
		if !h.m.ui.Modes.Open || h.m.ui.Models.Open {
			t.Fatalf("F4 over the model selector: modes %v, models %v", h.m.ui.Modes.Open, h.m.ui.Models.Open)
		}
		h.send(key('p', tea.ModAlt))
		if h.m.ui.Modes.Open || !h.m.ui.Models.Open {
			t.Fatalf("Alt+P over the modes: modes %v, models %v", h.m.ui.Modes.Open, h.m.ui.Models.Open)
		}
		h.send(key('p', tea.ModAlt))
		h.send(key(tea.KeyF4, 0))
		bx, by := at(t, h, " + ", 0)
		clickAt(t, h, bx+1, by)
		if h.m.ui.Modes.Open || !h.m.ui.Plus {
			t.Errorf("the band over the modes: modes %v, plus %v", h.m.ui.Modes.Open, h.m.ui.Plus)
		}
	})
}

// TestModesDraw is designs D6 and D7 through the application, with Sixel:
// the pill's image drawn over its cells after the frame; once the
// selector opens, its two strips at their places, their looks the frame's;
// a hover changes the strips' looks and draws them again; closed, their
// places written over with the frame's cells and no strip drawn.
func TestModesDraw(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, _ := modesRig(t, "claude-code", event.ModeAuto)
		h.send(PrimaryDeviceAttributesEvent{61, 4})
		h.frame()
		mp := h.m.screen.ModePill
		if !mp.Shown || mp.Look != (render.ModePillLook{Icon: state.IconBolt, Title: "Auto"}) {
			t.Fatalf("the pill %+v", mp)
		}
		raws := drawn(h, thumbDrawMsg{gen: h.m.kicks[len(h.m.kicks)-1]})
		if len(raws) != 1 || !strings.Contains(raws[0], pillDrawing(h)) {
			t.Fatalf("the pill's drawing %q", trunc(raws))
		}
		h.send(key(tea.KeyF4, 0))
		h.frame()
		md := h.m.screen.Modes
		if !md.Strips || md.Left.Rows != 18 || len(md.Left.Options) != 4 || !md.Left.Options[3].Selected {
			t.Fatalf("the strips %+v", md)
		}
		strips := func(md render.ModesShown) string {
			return fmt.Sprintf("\x1b[%d;%dH%s\x1b[%d;%dH%s", md.Y+1, md.LeftX+1, render.StripSixel(md.Left), md.Y+1, md.RightX+1, render.StripSixel(md.Right))
		}
		raws = drawn(h, thumbDrawMsg{gen: h.m.kicks[len(h.m.kicks)-1]})
		if len(raws) != 1 || !strings.Contains(raws[0], strips(md)) {
			t.Fatalf("the strips' drawing %q", trunc(raws))
		}
		n := len(h.m.kicks)
		h.send(key(tea.KeyUp, 0)) // Plan under the cursor
		h.frame()
		if md2 := h.m.screen.Modes; len(h.m.kicks) != n+1 || !md2.Left.Options[2].Cursor {
			t.Fatalf("the cursor on Plan: %d drawings asked, strips %+v", len(h.m.kicks)-n, md2.Left.Options)
		}
		raws = drawn(h, thumbDrawMsg{gen: h.m.kicks[len(h.m.kicks)-1]})
		if len(raws) != 1 || !strings.Contains(raws[0], strips(h.m.screen.Modes)) {
			t.Fatalf("drawn again %q", trunc(raws))
		}
		h.send(key(tea.KeyEsc, 0))
		h.frame()
		raws = drawn(h, thumbDrawMsg{gen: h.m.kicks[len(h.m.kicks)-1]})
		left := fmt.Sprintf("\x1b[%d;%dH", md.Y+1, md.LeftX+1)
		if len(raws) != 1 || !strings.Contains(raws[0], left) || strings.Contains(withoutPill(t, h, raws[0]), "\x1bP") || h.m.screen.Modes.Shown {
			t.Errorf("closed: %q", trunc(raws))
		}
		if len(h.m.drawn) != 1 || h.m.drawn[0].Item != -5 {
			t.Errorf("images left %+v, want the pill's alone", h.m.drawn)
		}
	})
}

// TestModesMock is criterion 39.12: on both CLIs' mock files the selector
// offers the file's CLI's modes, and a mode chosen comes into force at
// once, the mock answering as the CLI would.
func TestModesMock(t *testing.T) {
	for _, c := range []struct{ file, pick, pill string }{
		{"claude-models.jsonl", state.ModePlan, "▤ Plan"},
		{"codex-models.jsonl", state.ModeAuto, "⛨ Approve for me"},
	} {
		t.Run(c.file, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				h := mockDash(t, c.file)
				pickMode(t, h, c.pick)
				if got := h.m.State().Mode(); got != c.pick {
					t.Fatalf("mode %q, want %q", got, c.pick)
				}
				if f := h.frame(); !strings.Contains(f, c.pill) {
					t.Errorf("the pill:\n%s", f)
				}
			})
		})
	}
}
