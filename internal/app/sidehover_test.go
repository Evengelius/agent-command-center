package app

import (
	"fmt"
	"io"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/Evengelius/agent-command-center/internal/event"
	"github.com/Evengelius/agent-command-center/internal/render"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// Plan 43 (brief amendment 44), stage 2: the sidebar's hover and the
// motion filter (design D4), the tiles' images through the application
// (design D2), and the compact rule: where the Explorer's panel takes zone
// B's place, zone B taking the focus hides it.

// moveTo is a motion report with no button at x, y.
func moveTo(x, y int) tea.MouseMotionMsg { return tea.MouseMotionMsg{X: x, Y: y} }

// tooltipOnly says frame after is before with a tile's tooltip, label,
// left of the sidebar on row y (plan 44, design D1): every other row the
// same, that row ending as before from the sidebar's frame on.
func tooltipOnly(before, after string, y int, label string) bool {
	bl, al := strings.Split(before, "\n"), strings.Split(after, "\n")
	if len(bl) != len(al) || y >= len(al) {
		return false
	}
	for i := range bl {
		if i != y && bl[i] != al[i] {
			return false
		}
	}
	i := strings.LastIndex(al[y], label+"│")
	return i >= 0 && strings.HasSuffix(bl[y], al[y][i+len(label):])
}

// TestSideHover is criterion 43.5's hover (design D4), at 120x40 and
// 200x60: a motion with no button onto a tile hovers it, one off it
// unhovers it, a report wandering within a tile changes nothing and is
// dropped before Update; a drag over a tile hovers nothing; hovering
// Source Control while the Explorer is selected keeps the Explorer selected, the
// panel shown with its focus; hovered, each tile shows its look.
func TestSideHover(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, _ := explorerRig(t, &fakeGit{})
		for _, size := range []tea.WindowSizeMsg{{Width: 120, Height: 40}, {Width: 200, Height: 60}} {
			h.send(size)
			name := fmt.Sprintf("%dx%d", size.Width, size.Height)
			ex, ey := itemCell(t, h, render.SidePanel, render.SideExplorer)
			sx, sy := itemCell(t, h, render.SidePanel, render.SideSource)
			hover := func(want int, what string) {
				t.Helper()
				if h.m.ui.SideHover != want {
					t.Errorf("%s, %s: hover %d, want %d", name, what, h.m.ui.SideHover, want)
				}
			}
			before := h.m.View().Content
			mouseAt(t, h, moveTo(ex, ey))
			hover(render.SideExplorer, "onto the Explorer's tile")
			// Its look drawn, and its tooltip beside it on its middle row (plan
			// 44, design D1); no other row changed.
			if after := h.m.View().Content; after == before || !tooltipOnly(plainText(before), plainText(after), ey+1, " Explorer ") {
				t.Errorf("%s: the hovered tile's look not drawn, or more than its look and tooltip changed", name)
			}
			d := h.m.motionDropped
			for dy := range render.SideTileRows {
				for dx := range render.SideTileCells {
					mouseAt(t, h, moveTo(ex+dx, ey+dy))
				}
			}
			hover(render.SideExplorer, "wandering within its tile")
			if got := h.m.motionDropped - d; got != render.SideTileCells*render.SideTileRows {
				t.Errorf("%s: %d reports within the tile dropped, want all %d", name, got, render.SideTileCells*render.SideTileRows)
			}
			mouseAt(t, h, moveTo(ex+render.SideTileCells, ey)) // the sidebar's right edge
			hover(0, "onto the sidebar's frame")
			mouseAt(t, h, moveTo(sx+3, sy+2))
			hover(render.SideSource, "onto Source Control's tile")
			mouseAt(t, h, moveTo(10, 10))
			hover(0, "into zone A")
			mouseAt(t, h, click(60, 10)) // a selection dragged from zone B over a tile
			mouseAt(t, h, tea.MouseMotionMsg{X: ex, Y: ey, Button: tea.MouseLeft})
			hover(0, "a drag onto a tile")
			mouseAt(t, h, tea.MouseReleaseMsg{X: ex, Y: ey, Button: tea.MouseLeft})
			hover(render.SideExplorer, "the drag released on the tile, the pointer on it")
			mouseAt(t, h, moveTo(10, 10))
		}
		// Source Control hovered while the Explorer is selected: the Explorer stays
		// selected, the panel shown, its focus kept; with Sixel the frame
		// leaves each tile its own look.
		h.send(PrimaryDeviceAttributesEvent{61, 4})
		h.send(key(tea.KeyF3, 0))
		sx, sy := itemCell(t, h, render.SidePanel, render.SideSource)
		mouseAt(t, h, moveTo(sx, sy))
		h.m.View()
		tiles := h.m.screen.Tiles
		if h.m.ui.SideHover != render.SideSource || !h.m.ui.Explorer.Open || h.m.ui.Focus != "R" ||
			tiles[0].Look.Look != render.SideSelected || tiles[1].Look.Look != render.SideHover {
			t.Errorf("Source Control hovered with the Explorer selected: hover %d, shown %v, focus %q, looks %+v %+v",
				h.m.ui.SideHover, h.m.ui.Explorer.Open, h.m.ui.Focus, tiles[0].Look, tiles[1].Look)
		}
		ex, ey := itemCell(t, h, render.SidePanel, render.SideExplorer)
		mouseAt(t, h, moveTo(ex, ey))
		if h.m.View(); h.m.screen.Tiles[0].Look.Look != render.SideSelectedHover || h.m.screen.Tiles[1].Look.Look != render.SideInactive {
			t.Errorf("the Explorer hovered while selected: looks %+v %+v", h.m.screen.Tiles[0].Look, h.m.screen.Tiles[1].Look)
		}
	})
}

// TestMotionFilter is criterion 43.5's filter (design D4): through the
// filter the program runs before Update (run.go), a motion report with no
// button that changes no hover reaches neither Update nor View and is
// counted; one that changes the hover reaches both. Every other message
// passes: a key, a click, a drag's motion, and any motion while a popup
// follows the pointer (the "+" menu, the model selector, the modes
// selector) or a drag runs.
func TestMotionFilter(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, _ := selectorRig(t)
		h.send(tea.WindowSizeMsg{Width: 120, Height: 40})
		updates, views := 0, 0
		updateHook = func(_ *Model, msg tea.Msg) []tea.Cmd {
			if _, ok := msg.(tea.MouseMotionMsg); ok {
				updates++
			}
			return nil
		}
		t.Cleanup(func() { updateHook = nil })
		h.m.View()
		h.check = func(string) { views++ }
		ex, ey := itemCell(t, h, render.SidePanel, render.SideExplorer)
		for _, c := range []struct {
			x, y   int
			passes bool
			hover  int
		}{
			{60, 10, false, 0},                  // zone B, nothing hovered: no change
			{61, 12, false, 0},                  // and again
			{ex, ey, true, render.SideExplorer}, // onto the tile
			{ex + 2, ey + 1, false, render.SideExplorer},
			{ex + 5, ey + 2, false, render.SideExplorer},
			{60, 10, true, 0}, // off it
			{2, 30, false, 0},
		} {
			u, v, d := updates, views, h.m.motionDropped
			h.send(moveTo(c.x, c.y))
			reached := updates > u || views > v
			if reached != c.passes || (updates > u) != (views > v) || h.m.ui.SideHover != c.hover || (h.m.motionDropped > d) == c.passes {
				t.Errorf("a motion at %d,%d: Update %v, View %v, dropped %v, hover %d; want reached %v, hover %d",
					c.x, c.y, updates > u, views > v, h.m.motionDropped > d, h.m.ui.SideHover, c.passes, c.hover)
			}
		}
		h.check = nil
		// The filter's own conditions.
		idle := moveTo(60, 10)
		for _, c := range []struct {
			name string
			set  func()
			msg  tea.Msg
		}{
			{"a key", func() {}, key('a', 0)},
			{"a click", func() {}, click(60, 10)},
			{"a drag's motion", func() {}, tea.MouseMotionMsg{X: 60, Y: 10, Button: tea.MouseLeft}},
			{"the + menu open", func() { h.m.ui.Plus = true }, idle},
			{"the model selector open", func() { h.m.ui.Models.Open = true }, idle},
			{"the modes selector open", func() { h.m.ui.Modes.Open = true }, idle},
			{"a drag running", func() { h.m.drag.on = true }, idle},
		} {
			ui, dr := h.m.ui, h.m.drag
			c.set()
			if motionFilter(h.m, c.msg) == nil {
				t.Errorf("%s: dropped", c.name)
			}
			h.m.ui, h.m.drag = ui, dr
		}
		if motionFilter(h.m, idle) != nil {
			t.Error("an idle motion passed")
		}
	})
}

// TestMotionFilterProgram is criterion 43.5's filter in the real program
// (Run installs it): the reports of motion with no button Windows Terminal
// sends under mode 1003, written to the program's input, reach Update only
// when they change the hover; the others are dropped before it.
func TestMotionFilterProgram(t *testing.T) {
	// The sidebar at 120x40: the Explorer's tile at columns 113 to 118,
	// rows 3 to 5, Source Control's rows 7 to 9; zone B at column 50.
	report := func(x, y int) string { return fmt.Sprintf("%c[<35;%d;%dM", 0x1b, x+1, y+1) }
	moves := [][2]int{{50, 10}, {113, 3}, {115, 4}, {118, 5}, {114, 8}, {50, 10}, {51, 11}}
	in, w := io.Pipe()
	var seen []tea.MouseMotionMsg
	started := false
	updateHook = func(m *Model, msg tea.Msg) []tea.Cmd {
		if mo, ok := msg.(tea.MouseMotionMsg); ok {
			seen = append(seen, mo)
		}
		if !started && m.s.Screen == state.ScreenDashboard && m.s.Session.Phase == state.Ready {
			started = true
			go func() {
				time.Sleep(300 * time.Millisecond) // the dashboard drawn
				var b strings.Builder
				for _, p := range moves {
					b.WriteString(report(p[0], p[1]))
				}
				_, _ = io.WriteString(w, b.String())
				time.Sleep(300 * time.Millisecond)
				_, _ = io.WriteString(w, wtCtrlQ)
				time.Sleep(300 * time.Millisecond)
				_, _ = io.WriteString(w, wtCtrlQ)
			}()
		}
		return nil
	}
	resetTerminal = func(string) {}
	t.Cleanup(func() { resetTerminal, updateHook = consoleReset, nil; _ = in.Close() })
	m := New(Options{
		Config: Config{Mock: filepath.Join("..", "..", "testdata", "mock", "claude-subagent-approval.jsonl")},
		Cwd:    t.TempDir(),
		Now:    time.Now,
	})
	done := make(chan int, 1)
	go func() {
		code, _ := Run(m, tea.WithInput(in), tea.WithOutput(io.Discard), tea.WithWindowSize(120, 40), tea.WithoutSignals())
		done <- code
	}()
	select {
	case code := <-done:
		if code != ExitOK {
			t.Fatalf("exit code %d", code)
		}
	case <-time.After(20 * time.Second):
		t.Fatal("the program did not end on Ctrl+Q")
	}
	want := []tea.MouseMotionMsg{moveTo(113, 3), moveTo(114, 8), moveTo(50, 10)}
	if !slices.Equal(seen, want) || m.motionDropped != 4 {
		t.Errorf("Update saw %v, %d dropped; want %v, 4 dropped", seen, m.motionDropped, want)
	}
}

// TestSideMotionMode is criterion 43.5's mouse mode (design D4): every
// motion reported while the sidebar shows, from 100 columns on the
// dashboard; cell motion without it (under 100 columns, the viewer or the
// draft window open, zone C zoomed, the keys' overlay), where the "+" menu
// still asks for every motion (plan 30); none with F12.
func TestSideMotionMode(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r, _, _ := fileRig(t)
		h := r.h
		mode := func(what string, want tea.MouseMode) {
			t.Helper()
			if got := h.m.View().MouseMode; got != want {
				t.Errorf("%s: mouse mode %v, want %v", what, got, want)
			}
		}
		mode("120x40", tea.MouseModeAllMotion)
		h.send(tea.WindowSizeMsg{Width: 99, Height: 40})
		mode("99x40", tea.MouseModeCellMotion)
		h.send(tea.WindowSizeMsg{Width: 80, Height: 24})
		mode("80x24", tea.MouseModeCellMotion)
		if x, y, ok := plusCell(h); ok {
			mouseClick(t, h, x, y)
		}
		if !h.m.ui.Plus {
			t.Fatal("80x24: the + menu did not open")
		}
		mode("80x24, the + menu open", tea.MouseModeAllMotion)
		h.send(key(tea.KeyEscape, 0))
		h.send(tea.WindowSizeMsg{Width: 200, Height: 60})
		mode("200x60", tea.MouseModeAllMotion)
		h.send(key(tea.KeyF3, 0))
		mode("the panel shown", tea.MouseModeAllMotion)
		h.send(typed('/'))
		for _, k := range typedText("main.go") {
			h.send(k)
		}
		h.send(key(tea.KeyEnter, 0))
		if !h.m.ui.File.Open {
			t.Fatal("the viewer did not open")
		}
		mode("the viewer open", tea.MouseModeCellMotion)
		h.send(key(tea.KeyF3, 0)) // the viewer closed, the panel hidden
		h.send(key('g', tea.ModCtrl))
		mode("the draft window", tea.MouseModeCellMotion)
		h.send(key(tea.KeyEscape, 0))
		h.send(key(tea.KeyF6, 0)) // zone C
		h.send(typed('z'))
		if !h.m.ui.Zoom {
			t.Fatal("z did not zoom zone C")
		}
		mode("zone C zoomed", tea.MouseModeCellMotion)
		h.send(key(tea.KeyEscape, 0))
		h.send(key(tea.KeyF1, 0))
		mode("the keys' overlay", tea.MouseModeCellMotion)
		h.send(key(tea.KeyEscape, 0))
		mode("back on the dashboard", tea.MouseModeAllMotion)
		h.send(key(tea.KeyF12, 0))
		mode("F12", tea.MouseModeNone)
		h.send(key(tea.KeyF12, 0))
		mode("F12 again", tea.MouseModeAllMotion)
	})
}

// TestSideDraw is design D2 through the application, with Sixel: the
// tiles' images drawn after the frame that shows them, each over its cells
// written first; a change of hover asks for one drawing, the tile in its
// new look; a report within the tile asks for none, no frame drawn for it;
// a resize of the height alone draws them again, the renderer erasing the
// screen for it, and so does a report of the same size; under 100 columns,
// the sidebar gone, their places are written over with the frame's cells
// and no tile is drawn, the hover forgotten; back at 120 columns, drawn
// again, inactive. The mode pill is in text (plan 39's fallback when its
// font is missing), so the tiles are the frame's only images: the pill's
// image, which moves with the height, would draw them all again.
func TestSideDraw(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, _ := explorerRig(t, &fakeGit{})
		h.m.ui.PillNoFont = true
		h.send(PrimaryDeviceAttributesEvent{61, 4})
		h.frame()
		last := func() []string { return drawn(h, thumbDrawMsg{gen: h.m.kicks[len(h.m.kicks)-1]}) }
		tile := func(tl render.SideTileShown) string { // one tile's part of a drawing
			var b strings.Builder
			for r := range render.SideTileRows {
				fmt.Fprintf(&b, "%c[%d;%dH%s", 0x1b, tl.Y+1+r, tl.X+1, h.m.screen.Region(tl.X, tl.Y+r, render.SideTileCells))
			}
			fmt.Fprintf(&b, "%c[%d;%dH%s", 0x1b, tl.Y+1, tl.X+1, render.SideSixel(tl.Look))
			return b.String()
		}
		tiles := h.m.screen.Tiles
		inactive := func(tl int) render.SideTileLook { return render.SideTileLook{Tile: tl, Look: render.SideInactive} }
		if tiles[0] != (render.SideTileShown{Shown: true, X: 113, Y: 3, Look: inactive(render.SideExplorer)}) ||
			tiles[1] != (render.SideTileShown{Shown: true, X: 113, Y: 7, Look: inactive(render.SideSource)}) {
			t.Fatalf("the tiles %+v", tiles)
		}
		if raws := last(); len(raws) != 1 || !strings.Contains(raws[0], tile(tiles[0])+tile(tiles[1])) {
			t.Fatalf("the tiles' drawing %q", trunc(raws))
		}
		// A hover: one drawing, the Explorer's tile hovered, its cells as they
		// were; the frame changed by the tile's tooltip alone (plan 44, design
		// D1).
		n := len(h.m.kicks)
		rows := h.m.screen.String()
		mouseAt(t, h, moveTo(115, 4))
		h.frame()
		tg := h.m.screen.Tiles[0]
		if len(h.m.kicks) != n+1 || tg.Look.Look != render.SideHover || !tooltipOnly(plainText(rows), plainText(h.m.screen.String()), tg.Y+1, " Explorer ") {
			t.Fatalf("hovered: %d drawings asked, the tile %+v, more than the tooltip changed %v", len(h.m.kicks)-n, tg,
				!tooltipOnly(plainText(rows), plainText(h.m.screen.String()), tg.Y+1, " Explorer "))
		}
		if raws := last(); len(raws) != 1 || !strings.Contains(raws[0], tile(tg)) {
			t.Fatalf("the hovered tile's drawing %q", trunc(raws))
		}
		// Within the tile: dropped, nothing drawn.
		n, d := len(h.m.kicks), h.m.motionDropped
		mouseAt(t, h, moveTo(117, 5))
		h.frame()
		if len(h.m.kicks) != n || h.m.motionDropped != d+1 {
			t.Errorf("a report within the tile: %d drawings asked, %d dropped", len(h.m.kicks)-n, h.m.motionDropped-d)
		}
		// The height alone: drawn again at the same places.
		h.send(tea.WindowSizeMsg{Width: 120, Height: 41})
		h.frame()
		if raws := last(); len(h.m.kicks) != n+1 || len(raws) != 1 || !strings.Contains(raws[0], tile(h.m.screen.Tiles[0])+tile(h.m.screen.Tiles[1])) {
			t.Errorf("120x41: %d drawings asked, %q", len(h.m.kicks)-n, trunc(raws))
		}
		// The same size reported again: Bubble Tea's renderer erases the
		// screen for it all the same, so they are drawn again.
		n = len(h.m.kicks)
		h.send(tea.WindowSizeMsg{Width: 120, Height: 41})
		h.frame()
		if raws := last(); len(h.m.kicks) != n+1 || len(raws) != 1 || !strings.Contains(raws[0], tile(h.m.screen.Tiles[0])+tile(h.m.screen.Tiles[1])) {
			t.Errorf("120x41 again: %d drawings asked, %q", len(h.m.kicks)-n, trunc(raws))
		}
		n = len(h.m.kicks)
		// Under 100 columns: the places written over, no tile drawn.
		h.send(tea.WindowSizeMsg{Width: 99, Height: 41})
		h.frame()
		raws := last()
		if len(raws) != 1 || h.m.screen.Tiles[0].Shown || h.m.screen.Tiles[1].Shown || slices.Contains(drawnItems(h.m), -6) || slices.Contains(drawnItems(h.m), -7) {
			t.Fatalf("99 columns: %q, tiles %+v, images left %+v", trunc(raws), h.m.screen.Tiles, h.m.drawn)
		}
		for _, p := range [][2]int{{113, 3}, {113, 5}, {113, 7}, {113, 9}} {
			if at := fmt.Sprintf("%c[%d;%dH%s", 0x1b, p[1]+1, p[0]+1, h.m.screen.Region(p[0], p[1], render.SideTileCells)); !strings.Contains(raws[0], at) {
				t.Errorf("99 columns: the tile's row at %d,%d not written over", p[0], p[1])
			}
		}
		for _, tl := range []int{render.SideExplorer, render.SideSource} {
			for _, lk := range []render.SideLook{render.SideInactive, render.SideHover, render.SideSelected, render.SideSelectedHover} {
				if strings.Contains(raws[0], render.SideSixel(render.SideTileLook{Tile: tl, Look: lk})) {
					t.Errorf("99 columns: tile %d drawn in look %d", tl, lk)
				}
			}
		}
		// Back at 120 columns: drawn again, the hover forgotten.
		h.send(tea.WindowSizeMsg{Width: 120, Height: 41})
		h.frame()
		if tl := h.m.screen.Tiles; h.m.ui.SideHover != 0 || tl[0] != (render.SideTileShown{Shown: true, X: 113, Y: 3, Look: inactive(render.SideExplorer)}) {
			t.Errorf("back at 120 columns: hover %d, tiles %+v", h.m.ui.SideHover, tl)
		}
		if raws := last(); len(raws) != 1 || !strings.Contains(raws[0], tile(h.m.screen.Tiles[0])+tile(h.m.screen.Tiles[1])) {
			t.Errorf("back at 120 columns: %q", trunc(raws))
		}
	})
}

// TestCompactFocus is plan 43's compact rule: where the Explorer's panel
// takes zone B's place, at 120x40 and 80x24, zone B taking the focus hides
// it, so typed text never goes into a draft that is not drawn: Esc on a
// top-level row, F6 or Tab coming round to B, Esc from zone C, F2 to an
// approval; the draft kept, and a key typed after reaches it. Docked, at
// 200x60, zone B's focus keeps the panel; a resize leaving it no room, B
// focused, hides it, the panel focused keeps it.
func TestCompactFocus(t *testing.T) {
	for _, size := range []tea.WindowSizeMsg{{Width: 120, Height: 40}, {Width: 80, Height: 24}} {
		for _, c := range []struct {
			name string
			keys []tea.KeyPressMsg
		}{
			{"Esc on a top-level row", []tea.KeyPressMsg{key(tea.KeyEscape, 0)}},
			{"F6 round to B", []tea.KeyPressMsg{key(tea.KeyF6, 0), key(tea.KeyF6, 0), key(tea.KeyF6, 0)}},
			{"Tab round to B", []tea.KeyPressMsg{key(tea.KeyTab, 0), key(tea.KeyTab, 0), key(tea.KeyTab, 0)}},
			{"Shift+F6 back to B", []tea.KeyPressMsg{key(tea.KeyF6, tea.ModShift)}},
			{"Esc from zone C", []tea.KeyPressMsg{key(tea.KeyF6, 0), key(tea.KeyEscape, 0)}},
			{"F2 from zone C", []tea.KeyPressMsg{key(tea.KeyF6, 0), key(tea.KeyF2, 0)}},
		} {
			t.Run(fmt.Sprintf("%dx%d %s", size.Width, size.Height, c.name), func(t *testing.T) {
				synctest.Test(t, func(t *testing.T) {
					r, _, _ := fileRig(t)
					h := r.h
					h.m.opt.Git = &fakeGit{}
					h.send(size)
					for _, k := range typedText("draft") {
						h.send(k)
					}
					if c.name == "F2 from zone C" {
						r.emit(event.ApprovalRequest, event.Approval{ID: "ap1", Tool: "Bash", ToolID: "t1", Input: toolInput(t, map[string]any{"command": "ls"})})
						h.settle()
					}
					h.send(key(tea.KeyF3, 0))
					if f := h.frame(); !h.m.ui.Explorer.Open || h.m.ui.Focus != "R" || h.m.screen.B.W != 0 || strings.Contains(f, "B · CONVERSATION") || !strings.Contains(f, "EXPLORER") {
						t.Fatalf("F3: shown %v, focus %q, zone B %d columns\n%s", h.m.ui.Explorer.Open, h.m.ui.Focus, h.m.screen.B.W, f)
					}
					for i, k := range c.keys {
						h.send(k)
						if i < len(c.keys)-1 && !h.m.ui.Explorer.Open {
							t.Fatalf("%s: hidden before zone B had the focus (focus %q)", k.String(), h.m.ui.Focus)
						}
					}
					if f := h.frame(); h.m.ui.Explorer.Open || h.m.ui.Focus != "B" || !strings.Contains(f, "B · CONVERSATION") || strings.Contains(f, "EXPLORER") {
						t.Fatalf("zone B's focus: shown %v, focus %q\n%s", h.m.ui.Explorer.Open, h.m.ui.Focus, f)
					}
					if c.name == "F2 from zone C" {
						if h.m.ui.Card != "ap1" {
							t.Errorf("F2: the card %q", h.m.ui.Card)
						}
						return
					}
					h.send(typed('x'))
					if got := h.m.edit.Text(); got != "draftx" {
						t.Errorf("typed after: the draft %q", got)
					}
				})
			})
		}
	}
	t.Run("docked, then a resize", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			h, _ := explorerRig(t, &fakeGit{})
			h.send(tea.WindowSizeMsg{Width: 200, Height: 60})
			h.send(key(tea.KeyF3, 0))
			for _, k := range []tea.KeyPressMsg{key(tea.KeyEscape, 0), key(tea.KeyF6, 0), key(tea.KeyF6, tea.ModShift)} {
				h.send(k)
				if !h.m.ui.Explorer.Open {
					t.Fatalf("docked: %s hid the panel (focus %q)", k.String(), h.m.ui.Focus)
				}
			}
			if h.m.ui.Focus != "B" {
				t.Fatalf("docked: focus %q", h.m.ui.Focus)
			}
			h.send(tea.WindowSizeMsg{Width: 120, Height: 40}) // no room to dock, zone B focused
			if h.m.ui.Explorer.Open || h.m.ui.Focus != "B" {
				t.Errorf("120x40, zone B focused: shown %v, focus %q", h.m.ui.Explorer.Open, h.m.ui.Focus)
			}
			h.send(tea.WindowSizeMsg{Width: 200, Height: 60})
			h.send(key(tea.KeyF3, 0))
			h.send(tea.WindowSizeMsg{Width: 120, Height: 40}) // the panel focused: in zone B's place
			if !h.m.ui.Explorer.Open || h.m.ui.Focus != "R" {
				t.Errorf("120x40, the panel focused: shown %v, focus %q", h.m.ui.Explorer.Open, h.m.ui.Focus)
			}
		})
	})
}
