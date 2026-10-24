package render

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"sync"

	"github.com/Evengelius/agent-command-center/internal/state"
)

// The modes selector's images (plan 39, designs D6 and D7; brief amendment
// 40's measures): the box's two strips, the band's rounded ends with the
// options' icons on the left and the check on the right, and the mode
// pill, drawn with Sixel over blank cells of the frame as the toggle (plan
// 35) and the slider (plan 37). The icons are the dashboard's own outline
// drawings on a 24-unit grid, after the owner's images.

// StripOption is one option as a strip draws it: its title's row and its
// rows (the title and the description) from the strip's first row, its
// icon, and how it is lit.
type StripOption struct {
	Row, Rows        int
	Icon             string
	Warning, Off     bool
	Selected, Cursor bool
}

// StripLook is a strip as the frame leaves it: its height in rows, the
// right strip or the left, the options.
type StripLook struct {
	Rows    int
	Right   bool
	Options []StripOption
}

// The strips' and the pill's colours (brief amendment 40).
var (
	stripGround  = hexColour(cModesGround.tc)
	stripBand    = hexColour(cModesBand.tc)
	stripIcon    = hexColour(cModesIcon.tc)
	stripIconSel = hexColour("#EEE0FA")
	stripBoltSel = hexColour("#E874F2")
	stripWarn    = hexColour("#F4C46E")
	stripOff     = hexColour(cModesOff.tc)
	stripCheck   = hexColour("#F6F2F8")
	pillFill     = hexColour("#21223A")
	pillOutline  = hexColour("#7E4AE1")
	pillBolt     = hexColour("#E982F6")
	pillLabel    = hexColour("#DBD9E6")
)

// The strips' geometry in pixels (design D5): the band 10 pixels inside
// the box's border line, its corners of radius 6; on the left strip each
// icon 36 pixels square, its centre 49 pixels into the strip, beside the
// title and the first description line; on the right strip the check's
// centre 15 pixels into it, on the title row.
const (
	stripBandL   = 5.0  // the band's left edge in the left strip
	stripBandR   = 45.0 // its right edge in the right strip
	stripRadius  = 6.0
	stripIconX   = 49.0
	stripIconBox = 36.0
	stripCheckX  = 15.0
	stripStroke  = 2.4
)

// StripImage is a strip as an opaque image, modesStripL or modesStripR
// cells wide and Rows rows high: the box's ground; each lit option's band
// from the middle of the half row above its title to the middle of the one
// below its last line, its outer corners rounded, meeting the text cells'
// band at the strip's inner edge; the left strip's icons, the right strip's
// check of the selected option.
func StripImage(look StripLook) *image.NRGBA {
	cells := modesStripL
	if look.Right {
		cells = modesStripR
	}
	w, h := cells*SixelCellW, look.Rows*SixelCellH
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			img.SetNRGBA(x, y, stripGround)
		}
	}
	for _, o := range look.Options {
		top := float64((o.Row-1)*SixelCellH + SixelCellH/2)
		bottom := float64((o.Row+o.Rows)*SixelCellH + SixelCellH/2)
		if o.Selected || o.Cursor {
			x0, x1, round := stripBandL, float64(w), [2]bool{true, false}
			if look.Right {
				x0, x1, round = 0, stripBandR, [2]bool{false, true}
			}
			fillRounded(img, x0, top, x1, bottom, stripRadius, round, stripBand)
		}
		cy := float64(o.Row*SixelCellH) + SixelCellH // the icon's middle: between the title and the first line
		if look.Right {
			if o.Selected {
				ty := float64(o.Row*SixelCellH) + SixelCellH/2
				strokePaths(img, [][]pt{{{stripCheckX - 11, ty + 0.5}, {stripCheckX - 3.5, ty + 7.5}, {stripCheckX + 12, ty - 7.5}}}, nil, stripStroke, stripCheck)
			}
			continue
		}
		col := stripIcon
		switch {
		case o.Warning:
			col = stripWarn
		case o.Off:
			col = stripOff
		case o.Selected && o.Icon == state.IconBolt:
			col = stripBoltSel
		case o.Selected:
			col = stripIconSel
		}
		paths, dots := iconShape(o.Icon, stripIconX-stripIconBox/2, cy-stripIconBox/2, stripIconBox/24)
		strokePaths(img, paths, dots, stripStroke, col)
	}
	return img
}

// pt is a point of an icon's path, in pixels.
type pt struct{ x, y float64 }

// iconShape is an icon's strokes and filled dots (centre, radius) in a box
// of 24 units at (x, y), each unit s pixels: the owner's images' outline
// icons, drawn anew.
func iconShape(icon string, x, y, s float64) (paths [][]pt, dots [][3]float64) {
	p := func(u ...float64) []pt {
		out := make([]pt, 0, len(u)/2)
		for i := 0; i+1 < len(u); i += 2 {
			out = append(out, pt{x + u[i]*s, y + u[i+1]*s})
		}
		return out
	}
	// arc is n points of the circle of centre (cx, cy) and radius r, from
	// angle a0 to a1 in degrees, y downwards.
	arc := func(cx, cy, r, a0, a1 float64, n int) []float64 {
		var out []float64
		for i := 0; i <= n; i++ {
			a := (a0 + (a1-a0)*float64(i)/float64(n)) * math.Pi / 180
			out = append(out, cx+r*math.Cos(a), cy+r*math.Sin(a))
		}
		return out
	}
	cat := func(parts ...[]float64) []float64 {
		var out []float64
		for _, q := range parts {
			out = append(out, q...)
		}
		return out
	}
	switch icon {
	case state.IconHand:
		// Four fingers, each an arch, the palm and the thumb on the left.
		for _, f := range [][3]float64{{7, 10, 6}, {10, 13, 3.8}, {13, 16, 4.8}, {16, 19, 7.2}} {
			r := (f[1] - f[0]) / 2
			paths = append(paths, p(cat([]float64{f[0], 12.5}, arc(f[0]+r, f[2]+r, r, 180, 360, 10), []float64{f[1], 12.5})...))
		}
		paths = append(paths, p(cat([]float64{7, 12.5, 7, 15.4, 5.2, 13.6}, arc(4.2, 14.6, 1.42, -45, 135, 8),
			[]float64{6.9, 19.6}, arc(10.5, 17.6, 4.2, 150, 90, 6), []float64{14.5, 21.8}, arc(14.5, 17.3, 4.5, 90, 0, 8), []float64{19, 12.5})...))
	case state.IconCode:
		paths = [][]pt{p(8.5, 6.5, 3, 12, 8.5, 17.5), p(15.5, 6.5, 21, 12, 15.5, 17.5), p(13.8, 4.5, 10.2, 19.5)}
	case state.IconPlan:
		// A scroll: its body, the roll at its top left, the curl at its
		// bottom right, a line of writing.
		paths = [][]pt{
			p(cat([]float64{7, 17.5, 7, 4.2}, arc(5.6, 4.2, 1.4, 0, -180, 6), []float64{4.2, 6.5, 7, 6.5})...),
			p(cat(arc(5.6, 4.2, 1.4, -90, -90+1, 1)[:2], []float64{18, 2.8}, arc(18, 4.6, 1.8, -90, 0, 5), []float64{19.8, 18.2})...),
			p(cat([]float64{7, 17.5, 15.6, 17.5}, arc(17.7, 19.6, 2.1, 180, 360, 6), arc(17.7, 19.6, 2.1, 0, 90, 4), []float64{7.6, 21.7}, arc(7.6, 19.6, 2.1, 90, 180, 4), []float64{5.5, 17.5, 7, 17.5})...),
			p(10, 10.6, 11.2, 9.4, 12.4, 10.6, 13.6, 9.4, 14.8, 10.6, 16, 9.4),
			p(10, 13.8, 15, 13.8),
		}
	case state.IconBolt:
		paths = [][]pt{p(13.5, 2.2, 4.6, 13.6, 11.6, 13.6, 10.5, 21.8, 19.4, 10.4, 12.4, 10.4, 13.5, 2.2)}
	case state.IconShield:
		var curve []float64
		for i := 0; i <= 8; i++ { // the right side's quadratic, then the left's
			t := float64(i) / 8
			curve = append(curve, (1-t)*(1-t)*19.4+2*t*(1-t)*19.4+t*t*12, (1-t)*(1-t)*11.6+2*t*(1-t)*17.8+t*t*21.6)
		}
		for i := 1; i <= 8; i++ {
			t := float64(i) / 8
			curve = append(curve, (1-t)*(1-t)*12+2*t*(1-t)*4.6+t*t*4.6, (1-t)*(1-t)*21.6+2*t*(1-t)*17.8+t*t*11.6)
		}
		paths = [][]pt{p(cat([]float64{12, 2.4, 19.4, 5.2, 19.4, 11.6}, curve, []float64{4.6, 5.2, 12, 2.4})...),
			p(8.6, 9.4, 11.2, 12, 8.6, 14.6), p(12.8, 14.6, 15.8, 14.6)}
	case state.IconHexagon:
		var hex []float64
		for i := 0; i <= 6; i++ {
			a := (-90 + 60*float64(i)) * math.Pi / 180
			hex = append(hex, 12+10*math.Cos(a), 12+10*math.Sin(a))
		}
		paths = [][]pt{p(hex...), p(12, 7.4, 12, 13.2)}
		dots = [][3]float64{{x + 12*s, y + 16.6*s, 1.25 * s}}
	}
	return paths, dots
}

// fillRounded fills the rectangle from (x0, y0) to (x1, y1) with c, the
// corners of the left (round[0]) or the right side (round[1]) rounded with
// radius r, its edges blended into what lies under them, 4x4 samples a
// pixel.
func fillRounded(img *image.NRGBA, x0, y0, x1, y1, r float64, round [2]bool, c color.NRGBA) {
	b := img.Bounds()
	in := func(px, py float64) bool {
		if px < x0 || px > x1 || py < y0 || py > y1 {
			return false
		}
		cx, cy := 0.0, 0.0
		switch {
		case round[0] && px < x0+r:
			cx = x0 + r
		case round[1] && px > x1-r:
			cx = x1 - r
		default:
			return true
		}
		switch {
		case py < y0+r:
			cy = y0 + r
		case py > y1-r:
			cy = y1 - r
		default:
			return true
		}
		return math.Hypot(px-cx, py-cy) <= r
	}
	for y := max(int(y0), b.Min.Y); y < min(int(math.Ceil(y1)), b.Max.Y); y++ {
		for x := max(int(x0), b.Min.X); x < min(int(math.Ceil(x1)), b.Max.X); x++ {
			n := 0
			for sy := range 4 {
				for sx := range 4 {
					if in(float64(x)+(float64(sx)+0.5)/4, float64(y)+(float64(sy)+0.5)/4) {
						n++
					}
				}
			}
			if n > 0 {
				img.SetNRGBA(x, y, blend(img.NRGBAAt(x, y), c, float64(n)/16))
			}
		}
	}
}

// strokePaths draws the paths w pixels wide, with round caps and joins,
// and the dots filled, in c over what img holds, 4x4 samples a pixel.
func strokePaths(img *image.NRGBA, paths [][]pt, dots [][3]float64, w float64, c color.NRGBA) {
	var segs [][2]pt
	minX, minY, maxX, maxY := math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)
	grow := func(q pt, r float64) {
		minX, minY, maxX, maxY = math.Min(minX, q.x-r), math.Min(minY, q.y-r), math.Max(maxX, q.x+r), math.Max(maxY, q.y+r)
	}
	for _, path := range paths {
		for i := range path {
			grow(path[i], w)
			if i > 0 {
				segs = append(segs, [2]pt{path[i-1], path[i]})
			}
		}
	}
	for _, d := range dots {
		grow(pt{d[0], d[1]}, d[2]+1)
	}
	b := img.Bounds()
	half := w / 2
	for y := max(int(minY), b.Min.Y); y < min(int(math.Ceil(maxY)), b.Max.Y); y++ {
		for x := max(int(minX), b.Min.X); x < min(int(math.Ceil(maxX)), b.Max.X); x++ {
			n := 0
			for sy := range 4 {
				for sx := range 4 {
					q := pt{float64(x) + (float64(sx)+0.5)/4, float64(y) + (float64(sy)+0.5)/4}
					hit := false
					for _, sg := range segs {
						if segDist(q, sg[0], sg[1]) <= half {
							hit = true
							break
						}
					}
					for _, d := range dots {
						hit = hit || math.Hypot(q.x-d[0], q.y-d[1]) <= d[2]
					}
					if hit {
						n++
					}
				}
			}
			if n > 0 {
				img.SetNRGBA(x, y, blend(img.NRGBAAt(x, y), c, float64(n)/16))
			}
		}
	}
}

// segDist is the distance from q to the segment a-b.
func segDist(q, a, b pt) float64 {
	dx, dy := b.x-a.x, b.y-a.y
	l := dx*dx + dy*dy
	t := 0.0
	if l > 0 {
		t = math.Max(0, math.Min(1, ((q.x-a.x)*dx+(q.y-a.y)*dy)/l))
	}
	return math.Hypot(q.x-(a.x+t*dx), q.y-(a.y+t*dy))
}

// stripSixels are the strips encoded, one a look.
var stripSixels sync.Map

// StripSixel is StripImage encoded as Sixel, made once a look.
func StripSixel(look StripLook) string {
	k := fmt.Sprint(look)
	if s, ok := stripSixels.Load(k); ok {
		return s.(string)
	}
	s := Sixel(StripImage(look))
	stripSixels.Store(k, s)
	return s
}

// ModePillLook is the mode pill as the frame leaves it: the mode in
// force's icon, its title, and the warning colour for Full access.
type ModePillLook struct {
	Icon, Title string
	Warning     bool
}

// The pill's geometry in pixels (design D7): its height inside the row, its
// corners' radius (0.36 of its height, as the owner's third image's), the
// icon's box and centre, where the title starts, the padding after it.
const (
	pillTop, pillBottom = 1.0, 19.0
	pillRadius          = 6.5
	pillCompact         = 3 // cells: the icon alone
	pillIconBox         = 12.0
	pillIconX           = 13.0
	pillTextX           = 24
	pillTextPad         = 10
	pillCharW           = 8 // the title's pixels a character (Cascadia Mono at 13, as GDI draws it)
	pillOutlineW        = 1.3
)

// ModePillCells is the pill's width in cells: its ends and padding, the
// icon, and the title at pillCharW pixels a character; the icon alone in
// pillCompact cells for no title (the compact pill, design D7).
func ModePillCells(look ModePillLook) int {
	if look.Title == "" {
		return pillCompact
	}
	return (pillTextX + pillCharW*textWidth(look.Title) + pillTextPad + SixelCellW - 1) / SixelCellW
}

// PillImage is the mode pill (design D7, the owner's third image), the
// width of its cells and a row high: a fill with a thin outline and rounded
// corners, transparent around it where less than half covered (the row's
// ground shows); the mode's icon at the left, the lightning in its purple,
// Full access's in amber, the others in the title's colour; the title from
// label, its coverage at pillTextX (nil draws none).
func PillImage(look ModePillLook, label *image.Alpha) *image.NRGBA {
	w := ModePillCells(look) * SixelCellW
	img := image.NewNRGBA(image.Rect(0, 0, w, SixelCellH))
	x0, x1, r := 0.6, float64(w)-0.6, pillRadius
	cover := func(px, py, inset float64) bool {
		// The rounded rectangle inset pixels in: inside when the point
		// nearest it within the corners' centres lies within r-inset.
		cx := math.Max(x0+r, math.Min(x1-r, px))
		cy := math.Max(pillTop+r, math.Min(pillBottom-r, py))
		return math.Hypot(px-cx, py-cy) <= r-inset
	}
	for y := range SixelCellH {
		for x := range w {
			outer, inner := 0, 0
			for sy := range 4 {
				for sx := range 4 {
					px, py := float64(x)+(float64(sx)+0.5)/4, float64(y)+(float64(sy)+0.5)/4
					if cover(px, py, 0) {
						outer++
						if cover(px, py, pillOutlineW) {
							inner++
						}
					}
				}
			}
			switch {
			case outer < 8:
				// transparent: the row's own ground shows
			case inner == 0:
				img.SetNRGBA(x, y, pillOutline)
			default:
				img.SetNRGBA(x, y, blend(pillOutline, pillFill, float64(inner)/float64(outer)))
			}
		}
	}
	col, text := pillLabel, pillLabel
	switch {
	case look.Warning:
		col, text = stripWarn, stripWarn
	case look.Icon == state.IconBolt:
		col = pillBolt
	}
	if look.Icon != "" {
		ix := pillIconX
		if look.Title == "" {
			ix = float64(w) / 2 // the compact pill: the icon in the middle
		}
		paths, dots := iconShape(look.Icon, ix-pillIconBox/2, SixelCellH/2-pillIconBox/2, pillIconBox/24)
		strokePaths(img, paths, dots, 1.5, col)
	}
	if label != nil {
		lb := label.Bounds()
		for y := lb.Min.Y; y < lb.Max.Y && y < SixelCellH; y++ {
			for x := lb.Min.X; x < lb.Max.X && pillTextX+x < w; x++ {
				if a := label.AlphaAt(x, y).A; a > 0 && img.NRGBAAt(pillTextX+x, y).A != 0 {
					img.SetNRGBA(pillTextX+x, y, blend(img.NRGBAAt(pillTextX+x, y), text, float64(a)/255))
				}
			}
		}
	}
	return img
}

// pillSixels are the pills encoded, one a look.
var pillSixels sync.Map

// PillSixel is PillImage encoded as Sixel, made once a look and a label:
// the application keeps one label a title, so a title is encoded once.
func PillSixel(look ModePillLook, label *image.Alpha) string {
	k := fmt.Sprintf("%v %p", look, label)
	if s, ok := pillSixels.Load(k); ok {
		return s.(string)
	}
	s := Sixel(PillImage(look, label))
	pillSixels.Store(k, s)
	return s
}
