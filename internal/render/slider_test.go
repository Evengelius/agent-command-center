package render

import (
	"image/color"
	"math"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"

	"github.com/Evengelius/agent-command-center/internal/event"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// Plan 37 (brief amendment 38, the owner's five images of the slider): the
// effort slider as an image with Sixel, in text without, and the settings
// rows on shared columns.

// nearer says p is nearer a than b, by the sum of the channels' distances.
func nearer(p, a, b color.NRGBA) bool {
	d := func(q color.NRGBA) float64 {
		return math.Abs(float64(p.R)-float64(q.R)) + math.Abs(float64(p.G)-float64(q.G)) + math.Abs(float64(p.B)-float64(q.B))
	}
	return d(a) < d(b)
}

// TestSliderImage is criterion 1: 170x20 pixels; on the ground transparent
// outside the pill and opaque inside; with 5 levels the thumb white at x
// 10, 45, 80, 115 and 150, the fill #6784C1 to it above the lowest level and
// none at it, the other levels' dots, the gradient and its specks at the
// highest; 6 and 4 levels lit at their last; a held effort greyed; a level
// asked for ringed; none known, no thumb; on the band, its colour around
// the pill and in the last 10 columns.
func TestSliderImage(t *testing.T) {
	look := func(n int, pos float64) SliderLook { return SliderLook{Levels: n, Pos: pos, Asked: -1} }
	img := SliderImage(look(5, 2))
	if b := img.Bounds(); b.Dx() != 170 || b.Dy() != 20 {
		t.Fatalf("%dx%d, want 170x20", b.Dx(), b.Dy())
	}
	for y := 0; y < 20; y++ {
		for x := 0; x < 170; x++ {
			px, py := float64(x)+0.5, float64(y)+0.5
			d := math.Hypot(px-math.Max(10, math.Min(150, px)), py-10)
			a := img.NRGBAAt(x, y).A
			if d <= 9.3 && a != 0xFF || (d >= 10.7 || x >= 160) && a != 0 {
				t.Errorf("pixel %d,%d at %.1f from the pill's middle line has alpha %d", x, y, d, a)
			}
		}
	}
	for i := 0; i < 5; i++ {
		im := SliderImage(look(5, float64(i)))
		if got := hexOf(im.NRGBAAt(10+35*i, 10)); got != "#FFFFFF" {
			t.Errorf("level %d: the thumb's centre %s", i, got)
		}
		fill := hexOf(im.NRGBAAt(25, 10)) // between the left end's dot and the second level's
		switch {
		case i == 0 && fill != "#272B2D":
			t.Errorf("the lowest level: %s left of the thumb, want the track's #272B2D", fill)
		case i >= 1 && i <= 3 && fill != "#6784C1":
			t.Errorf("level %d: the fill %s, want #6784C1", i, fill)
		}
	}
	if p := SliderImage(look(5, 0)).NRGBAAt(0, 10); int(p.B)-int(p.R) > 30 {
		t.Errorf("the lowest level: %s at the left edge, blue from a sliver of fill", hexOf(p))
	}
	mid := SliderImage(look(5, 2))
	if p := mid.NRGBAAt(10, 10); !nearer(p, sliderDotFill, sliderFill) {
		t.Errorf("the lowest level's dot on the fill %s", hexOf(p))
	}
	if p := mid.NRGBAAt(115, 10); !nearer(p, sliderDot, sliderTrack) {
		t.Errorf("the fourth level's dot on the track %s", hexOf(p))
	}
	top := SliderImage(look(5, 4))
	for _, x := range []int{5, 72, 128} {
		want := gradientAt((float64(x) + 0.5) / 160)
		if p := top.NRGBAAt(x, 10); !nearer(p, want, sliderFill) || math.Abs(float64(p.R)-float64(want.R)) > 14 {
			t.Errorf("the highest level at x %d: %s, want near %s", x, hexOf(p), hexOf(want))
		}
	}
	specks := 0
	for y := 0; y < 20; y++ {
		for x := 0; x < 138; x++ {
			if p := top.NRGBAAt(x, y); p.A == 0xFF && int(p.R)+int(p.G)+int(p.B) > 3*0xC8 {
				specks++
			}
		}
	}
	if specks == 0 {
		t.Error("no speck on the highest level's gradient")
	}
	for _, c := range []struct {
		n       int
		pos     float64
		lit     bool
		whatFor string
	}{{6, 5, true, "Ultra, 6 levels"}, {6, 4, false, "the fifth of 6"}, {4, 3, true, "the fourth of 4"}, {4, 2, false, "the third of 4"}} {
		p := SliderImage(look(c.n, c.pos)).NRGBAAt(25, 10)
		if lit := !nearer(p, sliderFill, gradientAt(25.5/160)); lit != c.lit {
			t.Errorf("%s: %s at x 25, lit %v, want %v", c.whatFor, hexOf(p), lit, c.lit)
		}
	}
	held := SliderImage(SliderLook{Levels: 5, Pos: 4, Asked: -1, Locked: true})
	if f, k := hexOf(held.NRGBAAt(25, 10)), hexOf(held.NRGBAAt(150, 10)); f != "#4B505A" || k != "#9AA0AA" {
		t.Errorf("a held effort: the fill %s, the thumb %s; want #4B505A, #9AA0AA", f, k)
	}
	if p := SliderImage(SliderLook{Levels: 5, Pos: 2, Asked: 4}).NRGBAAt(153, 10); p.R < 0xB0 || p.G < 0xB0 || p.B < 0xB0 {
		t.Errorf("the ring round the level asked for: %s, want light", hexOf(p)) // 1.3 pixels thick, edged
	}
	none := SliderImage(SliderLook{Levels: 5, Pos: -1, Asked: -1})
	for y := 0; y < 20; y++ {
		for x := 0; x < 170; x++ {
			if hexOf(none.NRGBAAt(x, y)) == "#FFFFFF" {
				t.Fatalf("none known: a white pixel at %d,%d", x, y)
			}
		}
	}
	band := SliderImage(SliderLook{Levels: 5, Pos: 2, Asked: -1, Band: true})
	for _, p := range [][2]int{{0, 0}, {159, 19}, {165, 5}, {169, 19}} {
		if c := band.NRGBAAt(p[0], p[1]); hexOf(c) != "#342B47" || c.A != 0xFF {
			t.Errorf("on the band: %v %s", p, hexOf(c))
		}
	}
	if SliderSixel(look(5, 2)) != Sixel(SliderImage(look(5, 2))) {
		t.Error("SliderSixel is not the image encoded")
	}
}

// effortRowOf draws the selector at w by h, with Sixel or not and the
// cursor on Effort when band, and gives the screen, the Effort row's text,
// its screen row and the box's left column and width.
func effortRowOf(t *testing.T, s *state.State, ultracode bool, w, h int, sixel, band bool) (sc *Screen, row string, y, bx, bw int) {
	t.Helper()
	ui := UI{Width: w, Height: h, Now: appClock, Focus: "B", Sixel: sixel}
	ui.Models = OpenSelector(s, ultracode, 17)
	if band {
		for i, it := range SelectorItems(s, ui.Models) {
			if it.Kind == 'e' {
				ui.Models.Cursor = i
			}
		}
	}
	sc = Draw(s, ui)
	rows, bx, by := boxOf(t, sc)
	at := len(rows) - 2
	if ultracode {
		at = len(rows) - 4
	}
	return sc, rows[at], by + at, bx, textWidth(rows[0])
}

// fableAt is Fable 5.1 in force at the level, Ultracode on.
func fableAt(t *testing.T, level string) *state.State {
	return selectorState(t, "claude-code", testModels, event.Settings{Model: "claude-fable-5-1", Choice: "fable", Effort: level, EffortKnown: true,
		Ultracode: true, UltracodeKnown: true, UltracodeAvailable: true})
}

// TestSliderCells is criterion 2: with Sixel, the Effort row's columns W-19
// to W-3 are blank cells on the ground, the same from W-19 to the box's
// edge with the cursor on the row or not; each pill cell carries its
// nearest level, the last Effort's; the frame reports the slider's place
// and look; without Sixel it reports none.
func TestSliderCells(t *testing.T) {
	defer UsePalette(pickTrue)
	s := fableAt(t, "high")
	plain, row, y, bx, w := effortRowOf(t, s, true, 120, 40, true, false)
	band, _, _, _, _ := effortRowOf(t, s, true, 120, 40, true, true)
	if !strings.HasSuffix(row, strings.Repeat(" ", 18)+"│") {
		t.Errorf("the row %q", row)
	}
	for x := bx + w - 19; x < bx+w; x++ {
		p, _ := spanAt(plain.lines[y], x)
		b, _ := spanAt(band.lines[y], x)
		if p.st != b.st || p.ref != b.ref {
			t.Errorf("cell %d differs with the cursor on the row: %v %+v, then %v %+v", x-bx, p.st, p.ref, b.st, b.ref)
		}
		c := x - (bx + w - 19)
		want := ItemEffort
		if c < SliderPillCells {
			want = ItemLevel + sliderCellLevel(c, 5)
		}
		if c <= SliderPillCells && (p.st != stNone || p.ref.panel != SelectorPanel || p.ref.item != want) {
			t.Errorf("cell %d: style %v, panel %c, item %d; want item %d", c, p.st, p.ref.panel, p.ref.item, want)
		}
	}
	if got := []int{sliderCellLevel(0, 5), sliderCellLevel(4, 5), sliderCellLevel(8, 5), sliderCellLevel(15, 5)}; got[0] != 0 || got[1] != 1 || got[2] != 2 || got[3] != 4 {
		t.Errorf("cells 0, 4, 8, 15 take levels %v, want 0 1 2 4", got)
	}
	wantShown := SliderShown{Shown: true, X: bx + w - 19, Y: y, Look: SliderLook{Levels: 5, Pos: 2, Asked: -1}}
	if plain.Slider != wantShown {
		t.Errorf("the slider %+v, want %+v", plain.Slider, wantShown)
	}
	if wantShown.Look.Band = true; band.Slider != wantShown {
		t.Errorf("on the band: %+v", band.Slider)
	}
	if sc, _, _, _, _ := effortRowOf(t, s, true, 120, 40, false, false); sc.Slider.Shown {
		t.Error("without Sixel the frame reports an image to draw")
	}
}

// TestSliderText is criterion 3: without Sixel, 16 cells: the ends'
// half circles, a "·" at each level but the thumb's, the thumb two half
// circles over the left end at the lowest level and the right at the
// highest; the fill's ground to the thumb, the track's after, the
// gradient's at the highest; a level asked for "○"; greyed while held; the
// ends on the band while the row is.
func TestSliderText(t *testing.T) {
	defer UsePalette(pickTrue)
	text := func(l line) string { return strings.TrimSuffix(texts([]line{l}), "\n") }
	for i, want := range []string{
		halfLeft + halfRight + "  ·   ·  ·  ·" + halfRight,
		halfLeft + "·  " + halfLeft + halfRight + "  ·  ·  ·" + halfRight,
		halfLeft + "·  ·  " + halfLeft + halfRight + "  ·  ·" + halfRight,
		halfLeft + "·  ·   · " + halfLeft + halfRight + "  ·" + halfRight,
		halfLeft + "·  ·   ·  ·  " + halfLeft + halfRight,
	} {
		l := sliderText(SliderLook{Levels: 5, Pos: float64(i), Asked: -1})
		if len(l) != SliderPillCells || text(l) != want {
			t.Errorf("level %d: %d cells %q, want %q", i, len(l), text(l), want)
		}
	}
	type cell struct {
		c  int
		st *lipgloss.Style
	}
	for _, c := range []struct {
		what  string
		look  SliderLook
		cells []cell
	}{
		{"the middle level", SliderLook{Levels: 5, Pos: 2, Asked: -1}, []cell{
			{0, stSliderFillEnd}, {1, sliderFillGround.on(stSliderDotFill)}, {2, sliderFillGround.on(stNone)},
			{7, sliderFillGround.on(stModelKnob)}, {8, sliderTrackGround.on(stModelKnob)}, {10, sliderTrackGround.on(stNone)},
			{11, sliderTrackGround.on(stSliderDot)}, {15, stSliderTrackEnd}}},
		{"the highest", SliderLook{Levels: 5, Pos: 4, Asked: -1}, []cell{
			{0, stSliderTopLeft}, {1, sliderTopGrounds[1].on(stSliderDotTop)}, {2, sliderTopGrounds[2].on(stNone)},
			{14, sliderTopGrounds[14].on(stModelKnob)}, {15, stModelKnob}}},
		{"the lowest", SliderLook{Levels: 5, Pos: 0, Asked: -1}, []cell{
			{0, stModelKnob}, {1, sliderTrackGround.on(stModelKnob)}, {2, sliderTrackGround.on(stNone)}}},
		{"held at the highest", SliderLook{Levels: 5, Pos: 4, Asked: -1, Locked: true}, []cell{
			{0, stSliderLockEnd}, {1, sliderLockGround.on(stModelRule)}, {2, sliderLockGround.on(stNone)},
			{14, sliderLockGround.on(stSliderLockKnob)}, {15, stSliderLockKnob}}},
		{"the highest asked for", SliderLook{Levels: 5, Pos: 2, Asked: 4}, []cell{{14, sliderTrackGround.on(stModelKnob)}}},
		{"on the band", SliderLook{Levels: 5, Pos: 2, Asked: -1, Band: true}, []cell{
			{0, bandGround.on(stSliderFillEnd)}, {15, bandGround.on(stSliderTrackEnd)}}},
	} {
		l := sliderText(c.look)
		for _, want := range c.cells {
			if l[want.c].st != want.st {
				t.Errorf("%s: cell %d %q styled %v", c.what, want.c, l[want.c].text, l[want.c].st)
			}
		}
		if c.look.Asked >= 0 && l[sliderDotCell(c.look.Asked, 5)].text != "○" {
			t.Errorf("%s: %q", c.what, text(l))
		}
	}
}

// TestSettingsColumns is criterion 4: at 80x24, 120x40 and 200x60, at every
// level of Fable 5.1, "Effort (" and "Ultracode" at column 3, both
// descriptions at column 20, the slider's pill and the toggle's ending at
// W-4, the box 66 wide, the model rows' text where it was; Codex's Effort
// row on the same columns.
func TestSettingsColumns(t *testing.T) {
	defer UsePalette(pickTrue)
	for _, size := range [][2]int{{80, 24}, {120, 40}, {200, 60}} {
		for i, lv := range []string{"low", "medium", "high", "xhigh", "max"} {
			s := fableAt(t, lv)
			ui := UI{Width: size[0], Height: size[1], Now: appClock, Focus: "B"}
			ui.Models = OpenSelector(s, true, 17)
			rows, _, _ := boxOf(t, Draw(s, ui))
			w := textWidth(rows[0])
			e, u := rows[len(rows)-4], rows[len(rows)-2]
			slider := strings.TrimSuffix(texts([]line{sliderText(SliderLook{Levels: 5, Pos: float64(i), Asked: -1})}), "\n")
			if w != 66 || col(e, "Effort ("+state.EffortLabel(lv)+")") != 3 || col(e, EffortDescription) != 20 || col(e, slider) != w-19 ||
				col(u, "Ultracode") != 3 || col(u, UltracodeDescription) != 20 || col(u, toggleOnText) != w-8 {
				t.Errorf("%dx%d at %s, a box %d wide:\n%s\n%s", size[0], size[1], lv, w, e, u)
			}
			for _, r := range rows {
				if c := col(r, "Fable 5.1"); c >= 0 && c != 5 {
					t.Errorf("%dx%d: %q", size[0], size[1], r)
				}
			}
		}
	}
	codex := []event.ModelEntry{{ID: "gpt-6-sol", Model: "gpt-6-sol", Name: "GPT-6-Sol", Description: "Previous generation workhorse model.",
		Efforts: []string{"low", "medium", "high", "xhigh", "max", "ultra"}, DefaultEffort: "medium"}}
	cs := selectorState(t, "codex", codex, event.Settings{Model: "gpt-6-sol", Choice: "gpt-6-sol", Effort: "ultra", EffortKnown: true})
	_, e, _, _, w := effortRowOf(t, cs, false, 120, 40, false, false)
	if col(e, "Effort (Ultra)") != 3 || col(e, EffortDescription) != 20 || !strings.HasSuffix(e, halfLeft+halfRight+"  │") {
		t.Errorf("Codex's Effort row, a box %d wide: %q", w, e)
	}
}

// TestSliderLevels is criterion 5: only the model's own levels, the
// highest the last it lists: Fable 5.1 five, Opus 4.6 four, Haiku 4.5 none
// (no slider, "Effort (N/A)"), GPT-6-Sol six, lit at Ultra and not at Max.
func TestSliderLevels(t *testing.T) {
	defer UsePalette(pickTrue)
	models := append(append([]event.ModelEntry{}, testModels...), event.ModelEntry{ID: "claude-opus-4-6", Model: "claude-opus-4-6", Name: "Opus 4.6",
		Description: "Best for everyday, complex tasks", Efforts: []string{"low", "medium", "high", "max"}})
	codex := []event.ModelEntry{{ID: "gpt-6-sol", Model: "gpt-6-sol", Name: "GPT-6-Sol", Description: "Previous generation workhorse model.",
		Efforts: []string{"low", "medium", "high", "xhigh", "max", "ultra"}, DefaultEffort: "medium"}}
	for _, c := range []struct {
		what   string
		s      *state.State
		claude bool
		levels int
		label  string
		lit    bool
	}{
		{"Fable 5.1 at Max", fableAt(t, "max"), true, 5, "Effort (Max)", true},
		{"Opus 4.6 at High", selectorState(t, "claude-code", models, event.Settings{Model: "claude-opus-4-6", Choice: "claude-opus-4-6", Effort: "high", EffortKnown: true, UltracodeKnown: true}),
			true, 4, "Effort (High)", false},
		{"Opus 4.6 at Max", selectorState(t, "claude-code", models, event.Settings{Model: "claude-opus-4-6", Choice: "claude-opus-4-6", Effort: "max", EffortKnown: true, UltracodeKnown: true}),
			true, 4, "Effort (Max)", true},
		{"Haiku 4.5", selectorState(t, "claude-code", models, event.Settings{Model: "claude-haiku-4-5-20251001", Choice: "claude-haiku-4-5-20251001", EffortKnown: true, UltracodeKnown: true}),
			true, 0, "Effort (N/A)", false},
		{"GPT-6-Sol at Ultra", selectorState(t, "codex", codex, event.Settings{Model: "gpt-6-sol", Choice: "gpt-6-sol", Effort: "ultra", EffortKnown: true}),
			false, 6, "Effort (Ultra)", true},
		{"GPT-6-Sol at Max", selectorState(t, "codex", codex, event.Settings{Model: "gpt-6-sol", Choice: "gpt-6-sol", Effort: "max", EffortKnown: true}),
			false, 6, "Effort (Max)", false},
	} {
		sc, row, y, bx, w := effortRowOf(t, c.s, c.claude, 120, 40, false, false)
		dots, halves := strings.Count(row, "·"), strings.Count(row, halfLeft)+strings.Count(row, halfRight)
		wantDots, wantHalves := max(c.levels-1, 0), 4 // the ends and the thumb's halves
		switch {
		case c.levels == 0:
			wantHalves = 0
		case c.lit:
			wantHalves = 3 // the thumb over the right end
		}
		if col(row, c.label) != 3 || dots != wantDots || halves != wantHalves {
			t.Errorf("%s: %q: %d dots, %d half circles; want %q, %d, %d", c.what, row, dots, halves, c.label, wantDots, wantHalves)
		}
		if c.levels == 0 {
			continue
		}
		mid, _ := spanAt(sc.lines[y], bx+w-19+2)
		if lit := mid.st == sliderTopGrounds[2].on(stNone); lit != c.lit {
			t.Errorf("%s: lit %v, want %v", c.what, lit, c.lit)
		}
	}
}
