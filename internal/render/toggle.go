package render

import (
	"image"
	"image/color"
	"math"
	"strings"
	"sync"

	"github.com/Evengelius/agent-command-center/internal/state"
)

// The Ultracode toggle (plan 35, brief amendment 36; the owner's image of
// it): a pill with fully rounded ends and a round knob, drawn as a Sixel
// image of ToggleCells cells when the terminal draws Sixel, as the
// thumbnails are (plan 32), the frame keeping its cells blank; else in
// text with Powerline's rounded ends.

// ToggleCells is the toggle's width as an image, 50x20 pixels: the pill in
// its first pillCells cells, the last the band's last cell while the cursor
// is on the row. The renderer writes again every cell between a row's first
// and last changed cells, so a band cell of the frame after the image would
// have the band's coming and going erase it (design D2).
const ToggleCells = pillCells + 1

// pillCells is the pill's width, 40 pixels.
const pillCells = 4

// toggleTextCells is its width in text.
const toggleTextCells = 5

// ToggleShown is the toggle as the frame left it for its image: its first
// cell and the state the image shows; Shown is false when the frame has
// no image to draw (no selector open, no Sixel).
type ToggleShown struct {
	Shown            bool
	X, Y             int
	On, Usable, Band bool
}

// toggleColours are the track's and the knob's colours (design D1): on,
// off, and while Ultracode is unavailable.
func toggleColours(on, usable bool) (track, knob colour) {
	switch {
	case !usable:
		return cModelDim, cModelRule
	case on:
		return cModelToggle, cModelKnob
	}
	return cModelOff, cModelKnob
}

// ToggleImage is the toggle as an image of ToggleCells cells of
// SixelCellW by SixelCellH pixels (design D1): in its first pillCells, a
// track whose ends are half circles, a knob of 0.8 its height centred on
// the right end's centre (on) or the left's (off, and unavailable), its
// edge blended into the track. On the band the track's edge is blended
// into the band's colour, the dashboard's own, which fills the rest;
// elsewhere the pixels outside the track are transparent and its edge is
// drawn where half covered, so the terminal's own background shows around
// it, whatever it is.
func ToggleImage(on, usable, band bool) *image.NRGBA {
	w, h := ToggleCells*SixelCellW, SixelCellH
	pw := float64(pillCells * SixelCellW)
	tc, kc := toggleColours(on, usable)
	track, knob, ground := hexColour(tc.tc), hexColour(kc.tc), hexColour(cModelBand.tc)
	r := float64(h) / 2
	kx := r
	if on && usable {
		kx = pw - r
	}
	kr := 0.4 * float64(h)
	const n = 4 // samples a side of a pixel
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			var tcov, kcov float64
			for sy := 0; sy < n; sy++ {
				for sx := 0; sx < n; sx++ {
					px, py := float64(x)+(float64(sx)+0.5)/n, float64(y)+(float64(sy)+0.5)/n
					cx := math.Max(r, math.Min(pw-r, px))
					if math.Hypot(px-cx, py-r) <= r {
						tcov++
					}
					if math.Hypot(px-kx, py-r) <= kr {
						kcov++
					}
				}
			}
			tcov, kcov = tcov/(n*n), kcov/(n*n)
			c := blend(track, knob, kcov)
			switch {
			case band:
				c = blend(ground, c, tcov)
			case tcov < 0.5:
				c = color.NRGBA{}
			}
			img.SetNRGBA(x, y, c)
		}
	}
	return img
}

// blend is a with f of b over it.
func blend(a, b color.NRGBA, f float64) color.NRGBA {
	m := func(x, y uint8) uint8 { return uint8(math.Round(float64(x)*(1-f) + float64(y)*f)) }
	return color.NRGBA{m(a.R, b.R), m(a.G, b.G), m(a.B, b.B), 0xFF}
}

// toggleSixels are the images encoded, one a state.
var toggleSixels sync.Map

// ToggleSixel is ToggleImage encoded as Sixel, made once a state.
func ToggleSixel(on, usable, band bool) string {
	k := [3]bool{on, usable, band}
	if s, ok := toggleSixels.Load(k); ok {
		return s.(string)
	}
	s := Sixel(ToggleImage(on, usable, band))
	toggleSixels.Store(k, s)
	return s
}

// toggleState is the toggle's state: Ultracode on as the CLI reports it,
// and whether the CLI reports it available.
func toggleState(s *state.State) (on, usable bool) {
	usable = UltracodeUsable(s)
	return usable && s.Models.InForce.Ultracode, usable
}

// halfLeft and halfRight are Powerline's left and right half circles
// (U+E0B6, U+E0B4), which Windows Terminal draws the cell's height.
const halfLeft, halfRight = "\ue0b6", "\ue0b4"

// toggleText is the toggle without Sixel (design D3): a half circle at
// each end in the track's colour, and a knob of two half circles in the
// knob's colour on the track, at the right (on) or the left; the ends on
// the band while the cursor is on the row. On 16 colours, which have no
// ground to spare, the track is its ends alone.
func toggleText(on, usable, band bool) line {
	end, track, knob := stModelTrackEnd, trackGround, stModelKnob
	switch {
	case !usable:
		end, track, knob = stModelDimEnd, dimGround, stModelRule
	case !on:
		end, track = stModelOffEnd, offGround
	}
	if band {
		end = bandGround.on(end)
	}
	k := line{deco(halfLeft, track.on(knob)), deco(halfRight, track.on(knob))}
	gap := deco(" ", track.on(stNone))
	if on && usable {
		return append(append(line{deco(halfLeft, end), gap}, k...), deco(halfRight, end))
	}
	return append(append(line{deco(halfLeft, end)}, k...), gap, deco(halfRight, end))
}

// toggleCells is the toggle's cells in the row: the text toggle, or blank
// cells for the image to lie on. These stay on the ground while the band
// comes and goes, the image on the band carrying the band's colour: a cell
// the frame writes again erases the image over it, which would leave the
// toggle out until the next drawing (design D2).
func toggleCells(on, usable, band, sixel bool) line {
	if !sixel {
		return toggleText(on, usable, band)
	}
	return line{deco(strings.Repeat(" ", ToggleCells), stNone)}
}
