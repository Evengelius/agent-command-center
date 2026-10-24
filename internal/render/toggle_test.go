package render

import (
	"fmt"
	"image/color"
	"math"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"

	"github.com/Evengelius/agent-command-center/internal/event"
)

// Plan 35 (brief amendment 36, the owner's image of the toggle): the
// Ultracode toggle as a pill, an image with Sixel, text without.

// The text toggle on, and off or unavailable (design D3).
var (
	toggleOnText  = halfLeft + " " + halfLeft + halfRight + halfRight
	toggleOffText = halfLeft + halfLeft + halfRight + " " + halfRight
)

// hexOf is a pixel's colour as #RRGGBB.
func hexOf(c color.NRGBA) string { return fmt.Sprintf("#%02X%02X%02X", c.R, c.G, c.B) }

// TestToggleImage is criterion 1: 50x20 pixels, the pill in the first 40
// columns; on the ground, transparent outside the track and opaque inside
// it; on the band, no pixel transparent, the pill's corners and the last
// 10 columns the band's colour; the track's and the knob's colours on, off
// and unavailable, the knob at x 30 or 10, 16 pixels across along its
// middle row.
func TestToggleImage(t *testing.T) {
	for _, c := range []struct {
		name        string
		on, usable  bool
		track, knob string
		knobX       int
	}{
		{"on", true, true, "#CA84FB", "#FFFFFF", 30},
		{"off", false, true, "#3B3F51", "#FFFFFF", 10},
		{"unavailable", true, false, "#2A2D38", "#5E616D", 10},
	} {
		img := ToggleImage(c.on, c.usable, false)
		if b := img.Bounds(); b.Dx() != 50 || b.Dy() != 20 {
			t.Fatalf("%s: %dx%d, want 50x20", c.name, b.Dx(), b.Dy())
		}
		// Away from the track's edge: opaque inside it, transparent outside.
		for y := 0; y < 20; y++ {
			for x := 0; x < 50; x++ {
				px, py := float64(x)+0.5, float64(y)+0.5
				d := math.Hypot(px-math.Max(10, math.Min(30, px)), py-10)
				a := img.NRGBAAt(x, y).A
				if d <= 9.3 && a != 0xFF || (d >= 10.7 || x >= 40) && a != 0 {
					t.Errorf("%s: pixel %d,%d at %.1f from the track's middle line has alpha %d", c.name, x, y, d, a)
				}
			}
		}
		if got := hexOf(img.NRGBAAt(20, 10)); got != c.track {
			t.Errorf("%s: the track's middle %s, want %s", c.name, got, c.track)
		}
		if got := hexOf(img.NRGBAAt(c.knobX, 10)); got != c.knob {
			t.Errorf("%s: the knob's centre %s, want %s", c.name, got, c.knob)
		}
		// The knob's pixels along its middle row: those nearer its colour
		// than the track's.
		near := func(p, q color.NRGBA) float64 {
			return math.Abs(float64(p.R)-float64(q.R)) + math.Abs(float64(p.G)-float64(q.G)) + math.Abs(float64(p.B)-float64(q.B))
		}
		across, kn, tr := 0, hexColour(c.knob), hexColour(c.track)
		for x := 0; x < 40; x++ {
			if p := img.NRGBAAt(x, 10); p.A == 0xFF && near(p, kn) < near(p, tr) {
				across++
			}
		}
		if across != 16 {
			t.Errorf("%s: the knob %d pixels across, want 16", c.name, across)
		}
		band := ToggleImage(c.on, c.usable, true)
		for y := 0; y < 20; y++ {
			for x := 0; x < 50; x++ {
				if band.NRGBAAt(x, y).A != 0xFF {
					t.Fatalf("%s on the band: pixel %d,%d transparent", c.name, x, y)
				}
				if got := hexOf(band.NRGBAAt(x, y)); x >= 40 && got != "#342B47" {
					t.Fatalf("%s on the band: pixel %d,%d %s, want the band's #342B47", c.name, x, y, got)
				}
			}
		}
		for _, p := range [][2]int{{0, 0}, {39, 0}, {0, 19}, {39, 19}} {
			if got := hexOf(band.NRGBAAt(p[0], p[1])); got != "#342B47" {
				t.Errorf("%s on the band: corner %v %s, want the band's #342B47", c.name, p, got)
			}
		}
	}
	if ToggleSixel(true, true, false) != Sixel(ToggleImage(true, true, false)) {
		t.Error("ToggleSixel is not the image encoded")
	}
}

// ultracodeRowOf draws the selector at 120x40, with Sixel or not and the
// cursor on Ultracode when band, and gives the screen, the Ultracode row's
// text, its screen row, and the box's left column and width.
func ultracodeRowOf(t *testing.T, in event.Settings, sixel, band bool) (sc *Screen, row string, y, bx, w int) {
	t.Helper()
	s := selectorState(t, "claude-code", testModels, in)
	ui := UI{Width: 120, Height: 40, Now: appClock, Focus: "B", Sixel: sixel}
	ui.Models = OpenSelector(s, true, 17)
	if band {
		for i, it := range SelectorItems(s, ui.Models) {
			if it.Kind == 'u' {
				ui.Models.Cursor = i
			}
		}
	}
	sc = Draw(s, ui)
	rows, bx, by := boxOf(t, sc)
	return sc, rows[len(rows)-2], by + len(rows) - 2, bx, textWidth(rows[0])
}

// The Ultracode states: on, off, and unavailable.
var (
	toggleOn  = event.Settings{Model: "claude-fable-5-1", Choice: "fable", Effort: "max", EffortKnown: true, Ultracode: true, UltracodeKnown: true, UltracodeAvailable: true}
	toggleOff = event.Settings{Model: "claude-fable-5-1", Choice: "fable", UltracodeKnown: true, UltracodeAvailable: true}
	toggleDim = event.Settings{Model: "claude-fable-5-1", Choice: "fable", UltracodeKnown: true}
)

// TestToggleCells is criterion 2: with Sixel, the Ultracode row's columns
// W-7 to W-3 are blank cells of the row's item on the ground, and the
// frame reports where the image goes and its state; no character of the
// text toggle is drawn. From W-7 to the box's edge the cells are the same
// with the cursor on the row or not: the renderer writes again every cell
// between a row's first and last changed cells, so the band coming or
// going leaves the image alone.
func TestToggleCells(t *testing.T) {
	defer UsePalette(pickTrue)
	for _, c := range []struct {
		name       string
		in         event.Settings
		band       bool
		on, usable bool
	}{
		{"on", toggleOn, false, true, true},
		{"on, the cursor on the row", toggleOn, true, true, true},
		{"off", toggleOff, false, false, true},
		{"unavailable", toggleDim, false, false, false},
	} {
		sc, row, y, bx, w := ultracodeRowOf(t, c.in, true, c.band)
		if strings.ContainsAny(row, "●▐▌"+halfLeft+halfRight) || !strings.HasSuffix(row, strings.Repeat(" ", 6)+"│") {
			t.Errorf("%s: the row %q", c.name, row)
		}
		want := ToggleShown{Shown: true, X: bx + w - 7, Y: y, On: c.on, Usable: c.usable, Band: c.band}
		if sc.Toggle != want {
			t.Errorf("%s: the toggle %+v, want %+v", c.name, sc.Toggle, want)
		}
		for x := bx + w - 7; x <= bx+w-3; x++ {
			if sp, _ := spanAt(sc.lines[y], x); sp.ref.panel != SelectorPanel || sp.ref.item != ItemUltracode || sp.st != stNone {
				t.Errorf("%s: cell %d of the toggle: panel %c item %d, style %v", c.name, x-bx, sp.ref.panel, sp.ref.item, sp.st)
			}
		}
	}
	plain, _, y, bx, w := ultracodeRowOf(t, toggleOn, true, false)
	band, _, _, _, _ := ultracodeRowOf(t, toggleOn, true, true)
	for x := bx + w - 7; x < bx+w; x++ {
		p, _ := spanAt(plain.lines[y], x)
		b, _ := spanAt(band.lines[y], x)
		if p.st != b.st || p.ref != b.ref {
			t.Errorf("cell %d differs with the cursor on the row: %v %+v, then %v %+v", x-bx, p.st, p.ref, b.st, b.ref)
		}
	}
	if sc, _, _, _, _ := ultracodeRowOf(t, toggleOn, false, false); sc.Toggle.Shown {
		t.Error("without Sixel the frame reports an image to draw")
	}
}

// TestToggleText is criterion 4: without Sixel, columns W-8 to W-4 hold
// the text toggle: a half circle at each end in the track's colour, the
// knob's two half circles on the track, at the right on, at the left off
// and unavailable; on the band its ends lie on the band.
func TestToggleText(t *testing.T) {
	defer UsePalette(pickTrue)
	for _, c := range []struct {
		name           string
		in             event.Settings
		band           bool
		end, knob, gap *lipgloss.Style
	}{
		{"on", toggleOn, false, stModelTrackEnd, trackGround.on(stModelKnob), trackGround.on(stNone)},
		{"on, the cursor on the row", toggleOn, true, bandGround.on(stModelTrackEnd), trackGround.on(stModelKnob), trackGround.on(stNone)},
		{"off", toggleOff, false, stModelOffEnd, offGround.on(stModelKnob), offGround.on(stNone)},
		{"unavailable", toggleDim, false, stModelDimEnd, dimGround.on(stModelRule), dimGround.on(stNone)},
	} {
		sc, row, y, bx, w := ultracodeRowOf(t, c.in, false, c.band)
		text, knobAt, gapAt := toggleOffText, w-7, w-5
		if c.in.Ultracode && c.in.UltracodeAvailable {
			text, knobAt, gapAt = toggleOnText, w-6, w-7
		}
		if got := col(row, text); got != w-8 {
			t.Errorf("%s: %q at column %d, want %d: %q", c.name, text, got, w-8, row)
			continue
		}
		for _, p := range []struct {
			what string
			x    int
			want *lipgloss.Style
		}{{"the left end", w - 8, c.end}, {"the right end", w - 4, c.end}, {"the knob's left half", knobAt, c.knob}, {"the knob's right half", knobAt + 1, c.knob}, {"the track", gapAt, c.gap}} {
			if sp, _ := spanAt(sc.lines[y], bx+p.x); sp.st != p.want {
				t.Errorf("%s: %s at column %d styled %v", c.name, p.what, p.x, sp.st)
			}
		}
	}
}
