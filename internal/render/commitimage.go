package render

import (
	"fmt"
	"image"
	"image/color"
	"sync"
)

// The commit controls as Sixel images (plan 45, design D5): the button and
// the dropdown one image each, their labels from GDI's coverage masks as
// the mode pill's (plan 39, design D7); the message box's outline four
// images around its text cells. The geometry is the owner's image
// (references/15-commit-push.png) scaled to SixelCellW by SixelCellH a
// cell: each box two rows high to the eye, corners of radius 5.

// The button's geometry in pixels: its box from the middle of its first
// row to the middle of its last, its outline btnLine wide; the separator a
// pixel wide at the chevron part's left edge, btnSepInset short of the
// box's top and bottom; the chevron chevW by chevH, centred in its part; the check
// checkW by checkH, checkGap before the label. LabelEm is the labels' em in
// pixels, Cascadia Mono's at a cell's height, LabelTop the button's
// label's top, which centres its capitals on the box's middle.
const (
	btnTop, btnBottom = 10.0, 50.0
	boxRadius         = 5.0
	btnLine           = 1.2
	btnSepInset       = 7.0
	chevW, chevH      = 13.0, 7.0
	chevLine          = 1.4
	checkW, checkH    = 15.0, 12.5
	checkLine         = 1.6
	checkGap          = 10.0
	LabelEm           = 17
	LabelTop          = 20
)

// The dropdown's geometry: its box from its first row's top, menuBottom
// high, as close under the button as the rows allow; its band, while hot,
// bandInsetX inside its sides, bandInsetTop under its top and
// bandInsetBottom above its bottom, corners of radius bandRadius; its label
// MenuTextX in, its top MenuTextTop down, centred on the band.
const (
	menuTop, menuBottom = 0.6, 52.0
	menuLine            = 1.2
	bandInsetX          = 6.0
	bandInsetTop        = 8.0
	bandInsetBottom     = 7.0
	bandRadius          = 4.0
	MenuTextX           = 27
	MenuTextTop         = 17
)

// msgLine is the message box's outline's width; its box runs from the
// middle of its top row to the middle of its bottom row.
const msgLine = 1.2

// paintBox paints the rounded box from (x0, y0) to (x1, y1), fill inside an
// outline lw wide: each of a pixel's 4x4 samples takes the colour of what
// covers it; a pixel at least half covered takes its covered samples' mean,
// the others stay transparent, so the terminal's ground shows around the
// corners (as sideGround, plan 43).
func paintBox(img *image.NRGBA, x0, y0, x1, y1, r, lw float64, fill, outline color.NRGBA) {
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			var sum [3]int
			n := 0
			for sy := range 4 {
				for sx := range 4 {
					px, py := float64(x)+(float64(sx)+0.5)/4, float64(y)+(float64(sy)+0.5)/4
					var c color.NRGBA
					switch {
					case inRounded(px, py, x0, y0, x1, y1, r, lw):
						c = fill
					case inRounded(px, py, x0, y0, x1, y1, r, 0):
						c = outline
					default:
						continue
					}
					sum[0], sum[1], sum[2] = sum[0]+int(c.R), sum[1]+int(c.G), sum[2]+int(c.B)
					n++
				}
			}
			if n >= 8 {
				mean := func(v int) uint8 { return uint8((v + n/2) / n) }
				img.SetNRGBA(x, y, color.NRGBA{mean(sum[0]), mean(sum[1]), mean(sum[2]), 0xFF})
			}
		}
	}
}

// putLabel puts the coverage mask label in c at x over img's opaque pixels.
func putLabel(img *image.NRGBA, label *image.Alpha, x int, c color.NRGBA) {
	if label == nil {
		return
	}
	lb, b := label.Bounds(), img.Bounds()
	for y := lb.Min.Y; y < lb.Max.Y && y < b.Max.Y; y++ {
		for lx := lb.Min.X; lx < lb.Max.X; lx++ {
			px := x + lx
			if px < 0 || px >= b.Max.X {
				continue
			}
			if a := label.AlphaAt(lx, y).A; a > 0 && img.NRGBAAt(px, y).A != 0 {
				img.SetNRGBA(px, y, blend(img.NRGBAAt(px, y), c, float64(a)/255))
			}
		}
	}
}

// buttonColours are the button's fill, outline, label and separator for
// its look: halfway to the page with nothing to do, the outline in the
// title colour with the keyboard's focus.
func buttonColours(look CommitLook) (fill, outline, label, sep color.NRGBA) {
	fill, outline, label, sep = hexColour(cBtnFill.tc), hexColour(cBtnOutline.tc), hexColour(cBtnLabel.tc), hexColour(cBtnSep.tc)
	if look.Dim {
		fill, outline, label, sep = hexColour(cBtnFillDim.tc), hexColour(cBtnOutlineDim.tc), hexColour(cBtnLabelDim.tc), hexColour(cBtnSepDim.tc)
	}
	if look.Focus {
		outline = hexColour(cTitle.tc)
	}
	return fill, outline, label, sep
}

// CommitImage is the button (design D3, the owner's image), its cells wide
// and BtnRows high: the filled box and its outline, the separator, the
// chevron, and in the left part the check mark and the label from label
// (nil draws none), the two centred together.
func CommitImage(look CommitLook, label *image.Alpha) *image.NRGBA {
	w, h := look.Cells*SixelCellW, BtnRows*SixelCellH
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	fill, outline, text, sep := buttonColours(look)
	paintBox(img, 0.5, btnTop, float64(w)-0.5, btnBottom, boxRadius, btnLine, fill, outline)
	sx := float64(w - moreImageCells*SixelCellW)
	fillShape(img, func(px, py float64) bool {
		return px >= sx && px < sx+1 && py >= btnTop+btnSepInset && py <= btnBottom-btnSepInset
	}, sep)
	cy := (btnTop + btnBottom) / 2
	cx := (sx + float64(w)) / 2
	strokePaths(img, [][]pt{{{cx - chevW/2, cy - chevH/2}, {cx, cy + chevH/2}, {cx + chevW/2, cy - chevH/2}}}, nil, chevLine, text)
	group := 0.0
	if label != nil {
		group = float64(label.Bounds().Dx())
	}
	if look.Check {
		group += checkW + checkGap
	}
	gx := (sx - group) / 2
	if look.Check {
		// A short stroke down to two fifths of its width, a long one up.
		strokePaths(img, [][]pt{{{gx, cy}, {gx + 0.4*checkW, cy + checkH/2}, {gx + checkW, cy - checkH/2}}}, nil, checkLine, text)
		gx += checkW + checkGap
	}
	putLabel(img, label, int(gx+0.5), text)
	return img
}

// MenuImage is the dropdown (design D4, the owner's image), its cells wide
// and MenuRows high: its ground and outline, the band while hot, and its
// label from label.
func MenuImage(look MenuLook, label *image.Alpha) *image.NRGBA {
	w, h := look.Cells*SixelCellW, MenuRows*SixelCellH
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	paintBox(img, 0.5, menuTop, float64(w)-0.5, menuBottom, boxRadius, menuLine, hexColour(cMenuGround.tc), hexColour(cMenuOutline.tc))
	if look.Hot {
		x0, y0, x1, y1 := 0.5+bandInsetX, menuTop+bandInsetTop, float64(w)-0.5-bandInsetX, menuBottom-bandInsetBottom
		fillShape(img, func(px, py float64) bool { return inRounded(px, py, x0, y0, x1, y1, bandRadius, 0) }, hexColour(cMenuBand.tc))
	}
	putLabel(img, label, MenuTextX, hexColour(cMenuText.tc))
	return img
}

// MsgBoxImage is the message box's outline and inside (design D2), its
// cells wide and its rows and the two outline rows high, the outline in
// the focus colour with the focus; MsgStrips cuts it.
func MsgBoxImage(look MsgLook) *image.NRGBA {
	w, h := look.Cells*SixelCellW, (look.Rows+2)*SixelCellH
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	outline := hexColour(cMsgOutline.tc)
	if look.Focus {
		outline = hexColour(cMsgFocus.tc)
	}
	paintBox(img, 0.5, SixelCellH/2, float64(w)-0.5, float64(h)-SixelCellH/2, boxRadius, msgLine, hexColour(cMsgInside.tc), outline)
	return img
}

// MsgStrips are the message box's four images, over the cells its frame
// keeps blank: its top row, its bottom row, and its first and last cells
// beside the text rows.
func MsgStrips(look MsgLook) [4]*image.NRGBA {
	full := MsgBoxImage(look)
	w, h := full.Bounds().Dx(), full.Bounds().Dy()
	cut := func(r image.Rectangle) *image.NRGBA {
		out := image.NewNRGBA(image.Rect(0, 0, r.Dx(), r.Dy()))
		for y := range r.Dy() {
			for x := range r.Dx() {
				out.SetNRGBA(x, y, full.NRGBAAt(r.Min.X+x, r.Min.Y+y))
			}
		}
		return out
	}
	return [4]*image.NRGBA{
		cut(image.Rect(0, 0, w, SixelCellH)),
		cut(image.Rect(0, h-SixelCellH, w, h)),
		cut(image.Rect(0, SixelCellH, SixelCellW, h-SixelCellH)),
		cut(image.Rect(w-SixelCellW, SixelCellH, w, h-SixelCellH)),
	}
}

// The images encoded, one a look (and a label: the application keeps one
// mask a label, so a label is encoded once).
var commitSixels, menuSixels, msgSixels sync.Map

// CommitSixel is CommitImage encoded as Sixel, made once a look and label;
// a live look's, whose label changes every second, is not kept (design D15).
func CommitSixel(look CommitLook, label *image.Alpha) string {
	if look.Live {
		return Sixel(CommitImage(look, label))
	}
	k := fmt.Sprintf("%v %p", look, label)
	if s, ok := commitSixels.Load(k); ok {
		return s.(string)
	}
	s := Sixel(CommitImage(look, label))
	commitSixels.Store(k, s)
	return s
}

// MenuSixel is MenuImage encoded as Sixel, made once a look and label.
func MenuSixel(look MenuLook, label *image.Alpha) string {
	k := fmt.Sprintf("%v %p", look, label)
	if s, ok := menuSixels.Load(k); ok {
		return s.(string)
	}
	s := Sixel(MenuImage(look, label))
	menuSixels.Store(k, s)
	return s
}

// MsgSixels are MsgStrips encoded as Sixel, made once a look: the top, the
// bottom, the left, the right.
func MsgSixels(look MsgLook) [4]string {
	if s, ok := msgSixels.Load(look); ok {
		return s.([4]string)
	}
	var out [4]string
	for i, img := range MsgStrips(look) {
		out[i] = Sixel(img)
	}
	msgSixels.Store(look, out)
	return out
}
