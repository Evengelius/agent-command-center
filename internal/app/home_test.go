package app

import (
	"fmt"
	"image"
	"slices"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/Evengelius/agent-command-center/internal/input"
	"github.com/Evengelius/agent-command-center/internal/render"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// homePanel is the inside of a grid mode frame's panel (plan 51), 0 NEW
// SESSION and 1 PROJECTS: its rows' text between its frame's sides.
func homePanel(frame string, panel int) []string {
	lines := strings.Split(plainText(frame), "\n")
	top := slices.IndexFunc(lines, func(l string) bool { return strings.Contains(l, "╭") })
	if top < 0 {
		return nil
	}
	var edges []int
	for i, r := range []rune(lines[top]) {
		if r == '╭' || r == '╮' {
			edges = append(edges, i)
		}
	}
	if len(edges) < 2*(panel+1) {
		return nil
	}
	a, b := edges[2*panel], edges[2*panel+1]
	var out []string
	for _, l := range lines[top+1:] {
		rs := []rune(l)
		if len(rs) <= b || rs[a] == '╰' {
			break
		}
		out = append(out, string(rs[a+1:b]))
	}
	return out
}

// homeSays says the panel's rows hold text, whole and in order, wherever
// they wrap it: compared without their spaces.
func homeSays(rows []string, text string) bool {
	strip := func(s string) string { return strings.Join(strings.Fields(s), "") }
	return strings.Contains(strip(strings.Join(rows, " ")), strip(text))
}

// homeImages are the places of a sequence's Sixel images: the cell each
// one is written at.
func homeImages(seq string) []image.Point {
	var out []image.Point
	rest := seq
	for {
		i := strings.Index(rest, "\x1bP")
		if i < 0 {
			return out
		}
		j := strings.LastIndex(rest[:i], "\x1b[")
		var y, x int
		if _, err := fmt.Sscanf(rest[j:i], "\x1b[%d;%dH", &y, &x); err == nil {
			out = append(out, image.Pt(x-1, y-1))
		}
		rest = rest[i+2:]
	}
}

// homeTiles are the images that write a rectangle of the picture's cells
// (plan 51, designs D12 and D13): split at the banner's first and last
// rows, the banner's part in its scene's palette, the parts above and
// below as HomeSixel encodes them.
func homeTiles(h *homePic, r image.Rectangle) []string {
	w, ht := h.look.Geom.W, h.look.Geom.H
	var z image.Rectangle
	if h.banner != nil {
		z = h.banner.Cells()
	}
	var out []string
	for i, band := range []image.Rectangle{image.Rect(0, 0, w, z.Min.Y), z, image.Rect(0, z.Max.Y, w, ht)} {
		c := r.Intersect(band)
		if c.Empty() {
			continue
		}
		enc := render.HomeSixel
		if i == 1 {
			enc = h.banner.Encode
		}
		out = append(out, fmt.Sprintf("\x1b[%d;%dH%s", c.Min.Y+1, c.Min.X+1, enc(h.pixels, c)))
	}
	return out
}

// TestHomeSequence is plan 51's design D13 through the application: with
// Sixel at 208x59 the startup screen is a picture, written whole by its
// first drawing, image by image over the screen's rectangles, the banner's
// rows in its scene's palette (design D12), inside synchronized output;
// whole again by the second drawing; then nothing while nothing changes. A
// selection moved writes the cells it changed alone; a row the frame wrote
// again is written again whole, the last rows' as whole bands; a new size
// writes the picture whole; in grid mode the banner's rows alone are an
// image, none without room for it.
func TestHomeSequence(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		if _, err := input.Glyph('H', 20, false); err != nil {
			t.Skipf("GDI: %v", err) // no font: the text screen, the golden frames'
		}
		h := projectsRig(demoProjects(5), 0)(t)
		h.send(tea.WindowSizeMsg{Width: 208, Height: 59})
		h.send(PrimaryDeviceAttributesEvent{61, 4, 6, 7, 14})
		sc := screenNow(h)
		if !sc.Home.Shown || !sc.Home.Look.Geom.Picture {
			t.Fatalf("208x59 with Sixel: picture %v", sc.Home.Look.Geom.Picture)
		}
		rects := render.HomeRects(sc.Home.Look)
		at := func(r image.Rectangle) string { return fmt.Sprintf("\x1b[%d;%dH", r.Min.Y+1, r.Min.X+1) }
		seq := h.m.thumbSequence()
		if !strings.HasPrefix(seq, "\x1b[?2026h\x1b7") || !strings.HasSuffix(seq, "\x1b8\x1b[?2026l") {
			t.Error("not inside synchronized output, the cursor saved and restored")
		}
		// The banner: the whole rows within the design's, y 45 to 375, 69.5
		// to 590.2 image pixels at this size.
		if bn := h.m.home.banner; bn == nil || bn.Cells() != image.Rect(0, 4, 208, 29) {
			t.Fatalf("208x59: no banner, or not on rows 4 to 28")
		}
		images := 0
		for _, r := range rects {
			for _, tile := range homeTiles(&h.m.home, r) {
				images++
				if !strings.Contains(seq, tile) {
					t.Errorf("rectangle %v: a tile not written whole", r)
				}
			}
		}
		if got := len(homeImages(seq)); got != images || images != len(rects)+2 || h.m.home.whole == "" {
			t.Errorf("first drawing: %d images, whole %v; %d tiles of %d rectangles", got, h.m.home.whole != "", images, len(rects))
		}
		whole := len(seq)
		h.m.homeSecond = true
		if seq = h.m.thumbSequence(); len(homeImages(seq)) != images {
			t.Errorf("second drawing: %d images", len(homeImages(seq)))
		}
		h.m.homeSecond = false
		if seq = h.m.thumbSequence(); seq != "" {
			t.Errorf("nothing changed: %d bytes written", len(seq))
		}
		// The selection one row down: the cells of the two rows alone.
		before := h.m.home.look.Rows
		h.send(key(tea.KeyDown, 0))
		sc = screenNow(h)
		seq = h.m.thumbSequence()
		g := sc.Home.Look.Geom
		var y0, y1 float64
		for i, r := range before {
			if r.Selected {
				y0, y1 = g.Y(r.Y-16), g.Y(before[i+1].Y+16)
			}
		}
		imgs := homeImages(seq)
		if len(imgs) == 0 || len(seq) > whole/3 {
			t.Errorf("the selection moved: %d images, %d bytes, the whole %d", len(imgs), len(seq), whole)
		}
		for _, p := range imgs {
			if float64(p.Y*render.SixelCellH) < y0-render.SixelCellH || float64(p.Y*render.SixelCellH) > y1 {
				t.Errorf("an image at row %d, the band's rows %v to %v", p.Y, y0/render.SixelCellH, y1/render.SixelCellH)
			}
		}
		// A row the frame wrote again lost its pixels: written again whole, a
		// row of the banner in its palette.
		h.m.home.rows[5] = "stale"
		row := image.Rect(0, 5, g.W, 6)
		if seq = h.m.thumbSequence(); !strings.Contains(seq, at(row)+h.m.home.banner.Encode(h.m.home.pixels, row)) {
			t.Error("a row written again: its image not written")
		}
		// The last row: the last three written, whole bands (no scroll).
		h.m.home.rows[g.H-1] = "stale"
		bottom := image.Rect(0, g.H-3, g.W, g.H)
		if seq = h.m.thumbSequence(); len(homeImages(seq)) != 1 || !strings.Contains(seq, at(bottom)+render.HomeSixel(h.m.home.pixels, bottom)) {
			t.Errorf("the last row written again: %d images, not the last three rows'", len(homeImages(seq)))
		}
		// A new size: whole again.
		h.send(tea.WindowSizeMsg{Width: 150, Height: 40})
		sc = screenNow(h)
		seq = h.m.thumbSequence()
		images = 0
		for _, r := range render.HomeRects(sc.Home.Look) {
			images += len(homeTiles(&h.m.home, r))
		}
		if n := len(homeImages(seq)); h.m.home.whole == "" || n != images {
			t.Errorf("150x40: whole %v, %d images, %d tiles", h.m.home.whole != "", n, images)
		}
		// Grid mode: the banner's rows alone, an image over them in its
		// scene's palette (design D2).
		h.send(tea.WindowSizeMsg{Width: 100, Height: 30})
		sc = screenNow(h)
		banner := image.Rect(0, 2, 100, 14)
		seq = h.m.thumbSequence()
		if bn := h.m.home.banner; !sc.Home.Shown || bn == nil || bn.Cells() != banner || len(homeImages(seq)) != 1 || !strings.Contains(seq, at(banner)+bn.Encode(h.m.home.pixels, banner)) {
			t.Errorf("100x30: shown %v, %d images, not the banner's rows", sc.Home.Shown, len(homeImages(seq)))
		}
		// Grid mode with no room for the banner: no image.
		h.send(tea.WindowSizeMsg{Width: 80, Height: 24})
		sc = screenNow(h)
		if seq = h.m.thumbSequence(); sc.Home.Shown || strings.Contains(seq, "\x1bP") || h.m.home.pixels != nil {
			t.Errorf("80x24: shown %v, an image written %v", sc.Home.Shown, strings.Contains(seq, "\x1bP"))
		}
	})
}

// homeCell is the first cell of the frame drawn now whose span carries the
// home screen's item (plan 51, design D11).
func homeCell(t *testing.T, h *harness, item int) (int, int) {
	t.Helper()
	sc := screenNow(h)
	for y := range h.m.ui.Height {
		for x := range h.m.ui.Width {
			if p, it := sc.ItemAt(x, y); p == render.HomePanel && it == item {
				return x, y
			}
		}
	}
	t.Fatalf("no cell for item %#x", item)
	return 0, 0
}

// TestHomeMouse is plan 51's design D11, in picture mode and in grid mode:
// the home screen takes the mouse, F12 gives it back; an option chooses its
// agent CLI, the field takes the focus, a project's row is selected, its
// star toggles the favourite, the folder button opens the folder dialog
// (plan 52, criterion 52.1), START SESSION starts as Enter does; a right
// click does nothing.
func TestHomeMouse(t *testing.T) {
	for _, c := range []struct {
		name  string
		w, h  int
		sixel bool
	}{{"picture", 208, 59, true}, {"grid", 80, 24, false}} {
		t.Run(c.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				h := newHarness(t, Options{Providers: bothCLIs, Cwd: `C:\work\demo-project`, Projects: storeWith(t, favouriteProjects())})
				h.settle()
				h.send(tea.WindowSizeMsg{Width: c.w, Height: c.h})
				if c.sixel {
					h.send(PrimaryDeviceAttributesEvent{61, 4, 6, 7, 14})
				}
				if v := h.m.View(); v.MouseMode != tea.MouseModeCellMotion {
					t.Fatalf("mouse mode %v on the home screen", v.MouseMode)
				}
				x, y := homeCell(t, h, render.HomeItemOption+1)
				h.send(tea.MouseClickMsg{X: x, Y: y, Button: tea.MouseRight})
				if h.m.ui.Choice != "claude-code" {
					t.Errorf("a right click chose %q", h.m.ui.Choice)
				}
				h.send(click(x, y))
				if h.m.ui.Choice != "codex" || h.m.ui.Field != 0 {
					t.Errorf("the second option: choice %q, field %d", h.m.ui.Choice, h.m.ui.Field)
				}
				x, y = homeCell(t, h, render.HomeItemField)
				h.send(click(x, y))
				if h.m.ui.Field != 1 {
					t.Errorf("the field: focus on %d", h.m.ui.Field)
				}
				x, y = homeCell(t, h, render.HomeItemRow+2)
				h.send(click(x, y))
				if h.m.ui.ProjectSel != 2 || h.m.ui.Field != 3 {
					t.Errorf("the third row: selected %d, focus %d", h.m.ui.ProjectSel, h.m.ui.Field)
				}
				path, fav := h.m.projectList[2].Path, h.m.projectList[2].Favourite
				x, y = homeCell(t, h, render.HomeItemStar+2)
				h.send(click(x, y))
				for _, p := range h.m.projectList {
					if p.Path == path && p.Favourite == fav {
						t.Errorf("the star: %s still favourite %v", path, fav)
					}
				}
				h.send(key(tea.KeyF12, 0))
				if v := h.m.View(); v.MouseMode != tea.MouseModeNone {
					t.Errorf("F12: mouse mode %v", v.MouseMode)
				}
				h.send(key(tea.KeyF12, 0))
				fp := &folderPicker{answers: []string{`C:\work\chosen`}}
				h.m.opt.PickFolder = fp.pick
				h.m.ui.Field = 0
				x, y = homeCell(t, h, render.HomeItemFolder)
				h.send(click(x, y))
				if len(fp.opened()) != 1 || h.m.path.Text() != `C:\work\chosen` || h.m.ui.Field != 1 {
					t.Fatalf("the folder button: dialogs %q, field %q, focus %d", fp.opened(), h.m.path.Text(), h.m.ui.Field)
				}
				x, y = homeCell(t, h, render.HomeItemField)
				h.send(click(x, y))
				x, y = homeCell(t, h, render.HomeItemStart)
				h.send(click(x, y))
				if s := h.m.State(); s.Startup.Field != "project" || s.Startup.Message == "" {
					t.Errorf("START SESSION: field %q, message %q (the path does not exist)", s.Startup.Field, s.Startup.Message)
				}
			})
		})
	}
}

// TestHomeCtrlN is plan 51's design D9: Ctrl+N puts the form back as the
// screen opened it, the default agent CLI, the starting folder, the focus
// on Agent CLI, the list's first row, no message; not while a start is
// validated.
func TestHomeCtrlN(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newHarness(t, Options{Providers: bothCLIs, Cwd: `C:\work\demo-project`, Projects: storeWith(t, demoProjects(4))})
		h.settle()
		h.send(tea.WindowSizeMsg{Width: 120, Height: 40})
		h.send(key(tea.KeyDown, 0)) // Codex
		h.send(key(tea.KeyTab, 0))  // the field
		for _, k := range typedText(`\nowhere`) {
			h.send(k)
		}
		h.send(key(tea.KeyEnter, 0))
		h.m.ui.ProjectSel = 3
		if s := h.m.State(); h.m.ui.Choice != "codex" || s.Startup.Message == "" || h.m.ui.Field != 1 {
			t.Fatalf("before: choice %q, message %q, focus %d", h.m.ui.Choice, s.Startup.Message, h.m.ui.Field)
		}
		h.send(key('n', tea.ModCtrl))
		s := h.m.State()
		if h.m.ui.Choice != "claude-code" || h.m.path.Text() != `C:\work\demo-project` || h.m.ui.Field != 0 || h.m.ui.ProjectSel != 0 || s.Startup.Message != "" || s.Startup.Phase != state.Asking {
			t.Errorf("after: choice %q, path %q, focus %d, row %d, message %q, phase %s", h.m.ui.Choice, h.m.path.Text(), h.m.ui.Field, h.m.ui.ProjectSel, s.Startup.Message, s.Startup.Phase)
		}
		// While a start is validated: nothing.
		h.m.ui.Field = 2
		h.m.s.Startup.Phase = state.Validating
		h.send(key('n', tea.ModCtrl))
		if h.m.ui.Field != 2 || h.m.State().Startup.Phase != state.Validating {
			t.Errorf("validating: focus %d, phase %s", h.m.ui.Field, h.m.State().Startup.Phase)
		}
	})
}

// TestHomeSettle is plan 51's design D13 after a resize: the picture waits
// for the size to settle before it is drawn whole, homeQuiet after the last
// report; the second drawing writes the first's sequence again.
// TestHomeBanner is plan 51's design D12 through the application: at
// 208x59 the banner's scene is drawn with the picture and a frame asked
// for every render.BannerTick; a frame writes the cells its scene changed,
// in its palette, inside synchronized output, and asks for the next; the
// second drawing writes the scene as it is then; while the window lacks
// the focus no frame is drawn or asked for, the focus back asks again; a
// look that changes leaves the scene as it is; and a hidden banner
// (120x26: a picture with no room for one; 80x24: no picture) keeps no
// scene, asks for no frame and writes none (criterion 51.3).
func TestHomeBanner(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		if _, err := input.Glyph('H', 20, false); err != nil {
			t.Skipf("GDI: %v", err)
		}
		h := projectsRig(demoProjects(5), 0)(t)
		asked := 0
		h.m.opt.After = func(d time.Duration, msg tea.Msg) tea.Cmd {
			if _, ok := msg.(bannerMsg); ok {
				if d != render.BannerTick {
					t.Errorf("a banner frame asked for in %v", d)
				}
				asked++
			}
			return func() tea.Msg { return nil } // asked for, never delivered: the test delivers it
		}
		h.send(tea.WindowSizeMsg{Width: 208, Height: 59})
		h.send(PrimaryDeviceAttributesEvent{61, 4, 6, 7, 14})
		screenNow(h)
		h.m.thumbSequence()
		bn := h.m.home.banner
		if bn == nil || asked != 0 {
			t.Fatalf("208x59: banner %v, %d frames asked for before it", bn != nil, asked)
		}
		if !h.m.View().ReportFocus {
			t.Error("the startup screen does not ask for the terminal's focus reports")
		}
		h.send(tea.FocusMsg{})
		if asked != 1 {
			t.Fatalf("the banner drawn: %d frames asked for, want 1", asked)
		}
		z := bn.Cells()
		at := func(r image.Rectangle) string { return fmt.Sprintf("\x1b[%d;%dH", r.Min.Y+1, r.Min.X+1) }
		images := 0
		for i := range 12 {
			seqs := drawnBy(h, bannerMsg{})
			if bn.B.Frame != i+1 || asked != i+2 {
				t.Fatalf("frame %d: the scene at frame %d, %d asked for", i+1, bn.B.Frame, asked)
			}
			for _, s := range seqs {
				if !strings.HasPrefix(s, "\x1b[?2026h\x1b7") || !strings.HasSuffix(s, "\x1b8\x1b[?2026l") {
					t.Errorf("frame %d: not inside synchronized output, the cursor saved and restored", i+1)
				}
				for _, p := range homeImages(s) {
					images++
					if !p.In(z) {
						t.Errorf("frame %d: an image at %v, outside the banner's cells %v", i+1, p, z)
					}
				}
			}
		}
		if images == 0 {
			t.Error("12 frames wrote no image")
		}
		// The second drawing: the rest as the first wrote it, the scene as
		// it is now.
		h.m.homeSecond = true
		seq := h.m.thumbSequence()
		h.m.homeSecond = false
		if !strings.Contains(seq, h.m.home.whole) || !strings.Contains(seq, at(z)+bn.Encode(h.m.home.pixels, z)) {
			t.Error("the second drawing: not the first's sequence and the scene now")
		}
		// No focus: no frame drawn or asked for, until it comes back.
		h.send(tea.BlurMsg{})
		frame, was := bn.B.Frame, asked
		if seqs := drawnBy(h, bannerMsg{}); len(seqs) != 0 || bn.B.Frame != frame || asked != was {
			t.Errorf("no focus: %d sequences, frame %d to %d, %d asked for", len(seqs), frame, bn.B.Frame, asked-was)
		}
		h.send(tea.FocusMsg{})
		if asked != was+1 {
			t.Errorf("the focus back: %d frames asked for, want 1", asked-was)
		}
		// The selection moved: the scene stays, none of its cells written.
		h.m.thumbSequence()
		h.send(key(tea.KeyDown, 0))
		screenNow(h)
		seq = h.m.thumbSequence()
		for _, p := range homeImages(seq) {
			if p.In(z) {
				t.Errorf("the selection moved: an image at %v, in the banner", p)
			}
		}
		if h.m.home.banner != bn || bn.B.Frame != frame {
			t.Error("the selection moved: the scene not kept as it was")
		}
		// A size report, the same size: the picture whole again, the scene
		// going on.
		scene := bn.B
		h.send(tea.WindowSizeMsg{Width: 208, Height: 59})
		screenNow(h)
		if h.m.thumbSequence(); h.m.home.banner == nil || h.m.home.banner.B != scene {
			t.Error("a size report: the scene not kept")
		}
		// The panels grown into the banner (a note's lines): a new view, the
		// scene kept, all its cells to be written in its new palette.
		grown := h.m.home.look
		grown.Grow = 48
		pic := image.NewNRGBA(grown.Geom.Pixels())
		render.HomePaint(pic, grown, h.m.homeGlyphs(), pic.Rect)
		before := h.m.home.banner
		if !h.m.homeBanner(grown, pic) || h.m.home.banner == before || h.m.home.banner.B != scene || h.m.home.banner.Cells() == before.Cells() {
			t.Error("the panels grown: not a new view of the same scene")
		}
		// Hidden: no scene kept, no frame asked for or written.
		for _, size := range []tea.WindowSizeMsg{{Width: 120, Height: 26}, {Width: 80, Height: 24}} {
			h.send(size)
			screenNow(h)
			h.m.thumbSequence()
			was = asked
			seqs := drawnBy(h, bannerMsg{})
			if h.m.home.banner != nil || len(seqs) != 0 || asked != was {
				t.Errorf("%dx%d: scene kept %v, %d sequences, %d frames asked for", size.Width, size.Height, h.m.home.banner != nil, len(seqs), asked-was)
			}
		}
	})
}

func TestHomeSettle(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		if _, err := input.Glyph('H', 20, false); err != nil {
			t.Skipf("GDI: %v", err)
		}
		h := projectsRig(demoProjects(5), -1)(t)
		h.send(tea.WindowSizeMsg{Width: 208, Height: 59})
		h.send(PrimaryDeviceAttributesEvent{61, 4, 6, 7, 14})
		screenNow(h)
		if h.m.resized != testNow || h.m.homeSettle() != homeQuiet {
			t.Fatalf("just resized: wait %v", h.m.homeSettle())
		}
		if cmds := h.m.drawThumbs(thumbDrawMsg{gen: h.m.thumbGen}); len(cmds) != 1 || h.m.home.pixels != nil {
			t.Errorf("just resized: %d commands, drawn %v", len(cmds), h.m.home.pixels != nil)
		}
		h.m.resized = testNow.Add(-homeQuiet)
		if w := h.m.homeSettle(); w != 0 {
			t.Fatalf("settled: wait %v", w)
		}
		h.m.drawThumbs(thumbDrawMsg{gen: h.m.thumbGen})
		if h.m.home.pixels == nil || h.m.home.whole == "" {
			t.Fatal("settled: the picture not drawn whole")
		}
		if h.m.homeSettle() != 0 {
			t.Error("drawn: still waiting")
		}
		h.m.homeSecond = true
		seq := h.m.thumbSequence()
		h.m.homeSecond = false
		if !strings.Contains(seq, h.m.home.whole) {
			t.Error("the second drawing: not the first's sequence")
		}
	})
}
