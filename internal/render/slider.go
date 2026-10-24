package render

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"strings"
	"sync"

	"charm.land/lipgloss/v2"
)

// The effort slider (plan 37, brief amendment 38; the owner's five images
// of it, lowest to highest): one pill with a dot for each level the model
// lists, a fill from its left end to the level shown, and a white thumb
// there; the highest level fills the whole pill with a purple gradient and
// its specks. It is drawn as a Sixel image of SliderCells cells when the
// terminal draws Sixel, as the toggle is (plan 35), else in text.

// SliderPillCells is the pill's width: 160x20 pixels, the 8:1 of the
// owner's 289x36.
const SliderPillCells = 16

// SliderCells is the image's width: the pill and the cell after it, the
// band's last while the cursor is on the row, so that the band coming or
// going never writes the image's cells (plan 35, design D2).
const SliderCells = SliderPillCells + 1

// The slider's colours, from the owner's images' pixels (design D1).
var (
	sliderTrack    = color.NRGBA{0x27, 0x2B, 0x2D, 0xFF} // the pill's dark track
	sliderFill     = color.NRGBA{0x67, 0x84, 0xC1, 0xFF} // the fill below the highest level
	sliderDot      = color.NRGBA{0x69, 0x72, 0x76, 0xFF} // a level's dot on the track
	sliderDotFill  = color.NRGBA{0x95, 0xA9, 0xD4, 0xFF} // on the fill
	sliderDotTop   = color.NRGBA{0xC9, 0xB8, 0xEC, 0xFF} // on the highest level's gradient
	sliderThumb    = color.NRGBA{0xFF, 0xFF, 0xFF, 0xFF}
	sliderSpeck    = color.NRGBA{0xE6, 0xD8, 0xF5, 0xFF} // the highest level's specks
	sliderLockFill = color.NRGBA{0x4B, 0x50, 0x5A, 0xFF} // a held effort, greyed
	sliderLockDot  = color.NRGBA{0x5E, 0x61, 0x6D, 0xFF}
	sliderLockKnob = color.NRGBA{0x9A, 0xA0, 0xAA, 0xFF}
)

// sliderGradient is the highest level's fill along the pill, from its left
// end (0) to its right (1): the owner's fifth image's middle row, deep
// indigo, lavender at about half way, purple near the thumb.
var sliderGradient = []struct {
	at float64
	c  color.NRGBA
}{
	{0, color.NRGBA{0x4C, 0x3F, 0x99, 0xFF}},
	{0.083, color.NRGBA{0x60, 0x4D, 0xA3, 0xFF}},
	{0.166, color.NRGBA{0x6F, 0x56, 0xAE, 0xFF}},
	{0.249, color.NRGBA{0x8D, 0x70, 0xC9, 0xFF}},
	{0.332, color.NRGBA{0xA4, 0x81, 0xDA, 0xFF}},
	{0.415, color.NRGBA{0xB0, 0x87, 0xDE, 0xFF}},
	{0.5, color.NRGBA{0xB2, 0x86, 0xDD, 0xFF}},
	{0.581, color.NRGBA{0xAF, 0x7F, 0xD7, 0xFF}},
	{0.664, color.NRGBA{0x95, 0x69, 0xC1, 0xFF}},
	{0.747, color.NRGBA{0x87, 0x5D, 0xB5, 0xFF}},
	{0.83, color.NRGBA{0x81, 0x5A, 0xB3, 0xFF}},
	{1, color.NRGBA{0x80, 0x5C, 0xB4, 0xFF}},
}

// gradientAt is the highest level's colour at t along the pill.
func gradientAt(t float64) color.NRGBA {
	g := sliderGradient
	if t <= g[0].at {
		return g[0].c
	}
	for i := 1; i < len(g); i++ {
		if t <= g[i].at {
			return blend(g[i-1].c, g[i].c, (t-g[i-1].at)/(g[i].at-g[i-1].at))
		}
	}
	return g[len(g)-1].c
}

// sliderSpecks are the highest level's specks, where the owner's fifth
// image has them: along the pill (0 to 1), down it (0 to 1), and how
// bright against the gradient.
var sliderSpecks = []struct{ t, v, a float64 }{
	{0.074, 0.67, 0.55}, {0.081, 0.56, 0.6}, {0.116, 0.54, 0.65}, {0.168, 0.59, 0.6},
	{0.285, 0.64, 0.75}, {0.424, 0.56, 0.9}, {0.479, 0.33, 0.9}, {0.497, 0.39, 0.8},
	{0.559, 0.67, 0.8}, {0.607, 0.33, 0.85}, {0.718, 0.62, 0.7},
}

// SliderLook is what the slider shows: the model's levels, the position
// of its thumb (a level's index, or between two while it moves; negative
// for no level known), a level asked for and not yet in force (-1 for
// none), whether a variable holds the effort, and whether the row lies on
// the band.
type SliderLook struct {
	Levels int
	Pos    float64
	Asked  int
	Locked bool
	Band   bool
}

// sliderX is the x of the level i's centre in a pill w pixels long and h
// high: the levels evenly spaced from one end's centre to the other's, as
// the owner's thumbs are.
func sliderX(i, n int, w, h float64) float64 {
	r := h / 2
	if n <= 1 {
		return w - r
	}
	return r + float64(i)*(w-2*r)/float64(n-1)
}

// SliderImage is the slider as an image of SliderCells cells of SixelCellW
// by SixelCellH pixels (design D1): the pill in its first SliderPillCells,
// a track whose ends are half circles; the fill from its left end to the
// thumb, the owner's blue below the highest level and, as the thumb comes
// to the highest, the purple gradient over the whole pill with its specks;
// a dot at each level, the thumb's hidden under it; a ring round a level
// asked for; the thumb a white disc 0.944 of the pill's height, as the
// owner's 34 pixels in 36. Greyed while a variable holds the effort. On the
// band the pill's edge is blended into the band's colour, which fills the
// rest; elsewhere the pixels outside the pill are transparent, its edge
// drawn where half covered.
func SliderImage(look SliderLook) *image.NRGBA {
	w, h := SliderCells*SixelCellW, SixelCellH
	pw, ph := float64(SliderPillCells*SixelCellW), float64(h)
	r := ph / 2
	n := look.Levels
	ground := hexColour(cModelBand.tc)
	// top is how far the thumb has come into the highest level's interval:
	// 1 on it, 0 a level or more below, between while it moves.
	top := 0.0
	if n >= 1 && look.Pos >= 0 {
		top = math.Max(0, math.Min(1, look.Pos-float64(n-2)))
		if n == 1 {
			top = 1
		}
	}
	thumbX := -1.0
	if look.Pos >= 0 && n >= 1 {
		lo := math.Floor(look.Pos)
		thumbX = sliderX(int(lo), n, pw, ph)
		if f := look.Pos - lo; f > 0 && int(lo)+1 < n {
			thumbX += f * (sliderX(int(lo)+1, n, pw, ph) - thumbX)
		}
	}
	kr := 0.944 * r // the thumb's radius
	const ss = 4    // samples a side of a pixel
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			var sr, sg, sb, cov float64
			for sy := 0; sy < ss; sy++ {
				for sx := 0; sx < ss; sx++ {
					px, py := float64(x)+(float64(sx)+0.5)/ss, float64(y)+(float64(sy)+0.5)/ss
					c, in := sliderSample(look, px, py, pw, ph, r, kr, thumbX, top, n)
					if !in {
						continue
					}
					cov++
					sr += float64(c.R)
					sg += float64(c.G)
					sb += float64(c.B)
				}
			}
			if cov == 0 {
				if look.Band {
					img.SetNRGBA(x, y, ground)
				}
				continue
			}
			c := color.NRGBA{uint8(math.Round(sr / cov)), uint8(math.Round(sg / cov)), uint8(math.Round(sb / cov)), 0xFF}
			f := cov / (ss * ss)
			switch {
			case look.Band:
				c = blend(ground, c, f)
			case f < 0.5:
				c = color.NRGBA{}
			}
			img.SetNRGBA(x, y, c)
		}
	}
	return img
}

// sliderSample is the slider's colour at the point (px, py), and whether
// the point lies on the pill.
func sliderSample(look SliderLook, px, py, pw, ph, r, kr, thumbX, top float64, n int) (color.NRGBA, bool) {
	cx := math.Max(r, math.Min(pw-r, px))
	if math.Hypot(px-cx, py-r) > r {
		return color.NRGBA{}, false
	}
	t := px / pw
	// The pill: the track, the fill to the thumb's centre, the gradient as
	// the thumb comes to the highest level.
	c := sliderTrack
	// The fill reaches the thumb's centre; on the lowest level there is
	// none, not even the sliver the thumb's edge leaves (the owner's first
	// image).
	filled := thumbX > r && px <= thumbX
	if look.Locked {
		if filled {
			c = sliderLockFill
		}
	} else {
		fill := blend(sliderFill, gradientAt(t), top)
		switch {
		case top > 0:
			// The gradient lights the whole pill as the thumb reaches the
			// highest level (the owner's fifth image).
			if filled {
				c = fill
			} else {
				c = blend(sliderTrack, gradientAt(t), top)
			}
		case filled:
			c = fill
		}
		for _, s := range sliderSpecks {
			if top > 0 && math.Hypot(px-s.t*pw, py-s.v*ph) <= 0.85 {
				c = blend(c, sliderSpeck, s.a*top)
			}
		}
	}
	// A dot at each level, the thumb's hidden under the thumb.
	for i := 0; i < n; i++ {
		dx := sliderX(i, n, pw, ph)
		if math.Hypot(px-dx, py-r) <= 1.15 {
			d := sliderDot
			switch {
			case look.Locked:
				d = sliderLockDot
			case top > 0:
				d = blend(sliderDot, sliderDotTop, top)
			case thumbX >= 0 && dx <= thumbX:
				d = sliderDotFill
			}
			c = d
		}
		if i == look.Asked {
			if d := math.Hypot(px-dx, py-r); d >= 3.3 && d <= 4.6 {
				c = sliderThumb // the ring of a level asked for (design D4)
			}
		}
	}
	if thumbX >= 0 && math.Hypot(px-thumbX, py-r) <= kr {
		c = sliderThumb
		if look.Locked {
			c = sliderLockKnob
		}
	}
	return c, true
}

// SliderShown is the slider as the frame left it for its image: its first
// cell and what it shows; Shown is false when the frame has no image to
// draw (no selector open, no Sixel, a model with no level).
type SliderShown struct {
	Shown bool
	X, Y  int
	Look  SliderLook
}

// sliderCellLevel is the level of n nearest the pill's cell c, the one a
// click there asks for (design D4).
func sliderCellLevel(c, n int) int {
	pw, ph := float64(SliderPillCells*SixelCellW), float64(SixelCellH)
	x := (float64(c) + 0.5) * SixelCellW
	best := 0
	for i := 1; i < n; i++ {
		if math.Abs(x-sliderX(i, n, pw, ph)) < math.Abs(x-sliderX(best, n, pw, ph)) {
			best = i
		}
	}
	return best
}

// sliderTopGrounds are the highest level's gradient in text, a ground a
// cell of the pill (design D2).
var sliderTopGrounds [SliderPillCells]*ground

func init() {
	for c := range sliderTopGrounds {
		g := gradientAt((float64(c) + 0.5) / SliderPillCells)
		sliderTopGrounds[c] = &ground{c: colour{fmt.Sprintf("#%02X%02X%02X", g.R, g.G, g.B), "5"}, tints: map[*lipgloss.Style]*lipgloss.Style{}}
		grounds = append(grounds, sliderTopGrounds[c])
	}
}

// sliderDotCell is the cell of the level i of n in text: the levels
// evenly spread over the cells between the ends.
func sliderDotCell(i, n int) int {
	if n <= 1 {
		return SliderPillCells - 2
	}
	return int(math.Round(1 + float64(i)*float64(SliderPillCells-3)/float64(n-1)))
}

// sliderKnobCell is the first of the two cells the thumb takes on the
// level i of n in text: over the left end on the lowest level and the
// right on the highest; else its level's cell and the next in the pill's
// left half, the one before and its level's in the right.
func sliderKnobCell(i, n int) int {
	d := sliderDotCell(i, n)
	switch {
	case i == n-1:
		return SliderPillCells - 2
	case i == 0:
		return 0
	case d <= (SliderPillCells-1)/2:
		return d
	}
	return d - 1
}

// sliderText is the slider without Sixel (design D2): SliderPillCells
// cells, one span each; Powerline's half circles at the ends; the cells
// between on the fill's ground up to the thumb's left half, the track's
// after it, the gradient's at the highest level; a "·" at each level, a
// "○" at a level asked for; the thumb two white half circles, over the
// left end on the lowest level and the right on the highest. Greyed while
// a variable holds the effort; the ends on the band while the row is.
func sliderText(look SliderLook) line {
	n := look.Levels
	lv := -1
	if look.Pos >= 0 && n > 0 {
		lv = int(math.Round(look.Pos))
	}
	top := n > 0 && lv == n-1 && !look.Locked
	knob := -1
	if lv >= 0 {
		knob = sliderKnobCell(lv, n)
	}
	out := stNone
	if look.Band {
		out = bandGround.on(stNone)
	}
	filled := func(c int) bool { return lv >= 1 && c <= knob }
	groundOf := func(c int) *ground {
		switch {
		case top:
			return sliderTopGrounds[c]
		case look.Locked && filled(c):
			return sliderLockGround
		case filled(c):
			return sliderFillGround
		}
		return sliderTrackGround
	}
	knobSt := stModelKnob
	if look.Locked {
		knobSt = stSliderLockKnob
	}
	dots := map[int]int{}
	for i := 0; i < n; i++ {
		dots[sliderDotCell(i, n)] = i
	}
	l := make(line, 0, SliderPillCells)
	for c := 0; c < SliderPillCells; c++ {
		edge := c == 0 || c == SliderPillCells-1 // the ends lie on the row's ground
		switch {
		case knob >= 0 && c == knob:
			l = append(l, deco(halfLeft, sliderOn(edge, out, groundOf(c), knobSt)))
		case knob >= 0 && c == knob+1:
			l = append(l, deco(halfRight, sliderOn(edge, out, groundOf(c), knobSt)))
		case c == 0:
			end := stSliderTrackEnd
			switch {
			case top:
				end = stSliderTopLeft
			case look.Locked && lv >= 1:
				end = stSliderLockEnd
			case lv >= 1:
				end = stSliderFillEnd
			}
			l = append(l, deco(halfLeft, sliderOn(true, out, nil, end)))
		case c == SliderPillCells-1:
			end := stSliderTrackEnd
			if top {
				end = stSliderTopRight
			}
			l = append(l, deco(halfRight, sliderOn(true, out, nil, end)))
		default:
			i, isDot := dots[c]
			switch {
			case isDot && i == look.Asked:
				l = append(l, deco("○", groundOf(c).on(stModelKnob)))
			case isDot:
				dot := stSliderDot
				switch {
				case look.Locked:
					dot = stModelRule
				case top:
					dot = stSliderDotTop
				case filled(c):
					dot = stSliderDotFill
				}
				l = append(l, deco("·", groundOf(c).on(dot)))
			default:
				l = append(l, deco(" ", groundOf(c).on(stNone)))
			}
		}
	}
	return l
}

// sliderOn is st on the row's ground out (none, or the band's) for an end
// cell, on g between.
func sliderOn(edge bool, out *lipgloss.Style, g *ground, st *lipgloss.Style) *lipgloss.Style {
	switch {
	case !edge:
		return g.on(st)
	case out == stNone:
		return st
	}
	return bandGround.on(st)
}

// sliderCells are the slider's cells in its row, each carrying its
// nearest level for a click: with Sixel blank cells on the ground for the
// image, the last the band's (an image cell outside the pill); else the
// text slider (design D1, D2, D4).
func sliderCells(look SliderLook, sixel bool) line {
	var l line
	if sixel {
		for c := 0; c < SliderPillCells; c++ {
			l = append(l, items(line{deco(" ", stNone)}, ItemLevel+sliderCellLevel(c, look.Levels))...)
		}
		return append(l, items(line{deco(strings.Repeat(" ", SliderCells-SliderPillCells), stNone)}, ItemEffort)...)
	}
	for c, sp := range sliderText(look) {
		l = append(l, items(line{sp}, ItemLevel+sliderCellLevel(c, look.Levels))...)
	}
	return l
}

// sliderSixels are the images at levels, encoded, one a look.
var sliderSixels sync.Map

// SliderSixel is SliderImage encoded as Sixel; made once a look whose
// thumb is on a level, every time for one between two.
func SliderSixel(look SliderLook) string {
	if look.Pos != math.Trunc(look.Pos) {
		return Sixel(SliderImage(look))
	}
	k := fmt.Sprint(look)
	if s, ok := sliderSixels.Load(k); ok {
		return s.(string)
	}
	s := Sixel(SliderImage(look))
	sliderSixels.Store(k, s)
	return s
}
