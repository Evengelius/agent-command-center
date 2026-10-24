package render

import (
	"image"
	"image/color"
	"sync"
)

// The sidebar's tiles as images (plan 43, designs D2 and D3; the owner's
// images references/12-sidebar-collapsed.png and 13-sidebar-explorer.webp,
// the drawing tuned against them on an enlarged prototype): with Sixel
// each tile is a 60x60 image over its SideTileCells by SideTileRows blank
// cells, drawn after the frame as the mode pill is (plan 39), in one of
// its four looks.

// SideTileLook is a tile as its image draws it: the tile, SideExplorer or
// SideSkills, and its look.
type SideTileLook struct {
	Tile int
	Look SideLook
}

// The tile's geometry in pixels (design D3): the rounded tile, its corners'
// radius and its outline's width; the accent bar on its left edge, rounded
// at both ends; the icons on a grid of 24 units of sideUnit pixels centred
// at (sideIconX, sideIconY), their strokes sideStroke units wide.
const (
	sideX0, sideY0       = 5.0, 4.0
	sideX1, sideY1       = 57.0, 56.0
	sideRadius           = 9.0
	sideOutline          = 1.4
	sideBarX0, sideBarY0 = 2.0, 8.0
	sideBarX1, sideBarY1 = 6.2, 52.0
	sideBarRadius        = 2.0
	sideUnit             = 1.25
	sideIconX, sideIconY = 31.0, 30.0
	sideStroke           = 1.9
)

// The looks' colours (design D3, read from the references' pixels and the
// theme): the icon inactive; hovered, the selectors' hover ground (the
// model selector's band, plan 34) and the icon brighter; selected, the
// tile's fill, outline and icon; selected and hovered, the same a shade
// lighter; the accent bar.
var (
	sideIconOff   = hexColour(cSideIcon.tc)
	sideHoverFill = hexColour(cModelBand.tc)
	sideHoverIcon = hexColour("#B9C3D3")
	sideSelFill   = hexColour(cSideSel.tc)
	sideSelLine   = hexColour("#7A4EBA")
	sideSelIcon   = hexColour(cSideIconLit.tc)
	sideBothFill  = hexColour(cSideSelHover.tc)
	sideBothLine  = hexColour("#8C5DD0")
	sideBothIcon  = hexColour("#BAC6D8")
	sideBarColour = hexColour(cSideAccent.tc)
)

// SideImage is a tile in its look (design D3), SideTileCells by
// SideTileRows cells of SixelCellW by SixelCellH pixels: inactive, its icon
// alone in the muted grey; hovered, a rounded tile of the hover ground, the
// icon brighter; selected, the rounded tile's dark fill inside a thin
// purple outline, the lavender accent bar on its left edge, the icon light
// grey-blue; selected and hovered, the selected tile a shade lighter. The
// pixels around them are transparent, the terminal's ground showing there,
// an edge under half covered left out, as the mode pill's (plan 39, design
// D7).
func SideImage(t SideTileLook) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, SideTileCells*SixelCellW, SideTileRows*SixelCellH))
	icon := sideIconOff
	switch t.Look {
	case SideHover:
		sideGround(img, sideHoverFill, sideHoverFill, false)
		icon = sideHoverIcon
	case SideSelected:
		sideGround(img, sideSelFill, sideSelLine, true)
		icon = sideSelIcon
	case SideSelectedHover:
		sideGround(img, sideBothFill, sideBothLine, true)
		icon = sideBothIcon
	}
	sideIcon(img, t.Tile, icon)
	return img
}

// sideGround paints the rounded tile, fill inside an outline of outline's
// colour sideOutline wide, and with bar the accent bar over its left edge:
// each of a pixel's 4x4 samples takes the colour of what covers it, the bar
// over the fill over the outline; a pixel at least half covered takes its
// covered samples' mean, the others stay transparent, as at the mode pill's
// edge (plan 39, design D7).
func sideGround(img *image.NRGBA, fill, outline color.NRGBA, bar bool) {
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
					case bar && inRounded(px, py, sideBarX0, sideBarY0, sideBarX1, sideBarY1, sideBarRadius, 0):
						c = sideBarColour
					case inRounded(px, py, sideX0, sideY0, sideX1, sideY1, sideRadius, sideOutline):
						c = fill
					case inRounded(px, py, sideX0, sideY0, sideX1, sideY1, sideRadius, 0):
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

// sideIcon draws tile's icon in c over img (design D3): the Explorer's a
// tree, its trunk with a crossbar to the upper folder and a foot to the
// lower one, both folders filled, each with its tab; Skills' three filled
// nodes and their links, a branch from the left line up to the right node.
func sideIcon(img *image.NRGBA, tile int, c color.NRGBA) {
	const s = sideUnit
	ox, oy := sideIconX-12*s, sideIconY-12*s
	p := func(u ...float64) []pt {
		out := make([]pt, 0, len(u)/2)
		for i := 0; i+1 < len(u); i += 2 {
			out = append(out, pt{ox + u[i]*s, oy + u[i+1]*s})
		}
		return out
	}
	switch tile {
	case SideExplorer:
		strokePaths(img, [][]pt{p(5.5, 3, 5.5, 17.5, 10, 17.5), p(5.5, 7, 10, 7)}, nil, sideStroke*s, c)
		for _, f := range [][2]float64{{3.2, 10.8}, {13.7, 21.3}} {
			x0, x1, y0, y1 := ox+11.5*s, ox+21.5*s, oy+f[0]*s, oy+f[1]*s
			fillShape(img, func(px, py float64) bool {
				return inRounded(px, py, x0, y0+1.4*s, x1, y1, 1.6*s, 0) || inRounded(px, py, x0, y0, x0+4.6*s, y0+3*s, 1.2*s, 0)
			}, c)
		}
	case SideSkills:
		strokePaths(img, [][]pt{p(6, 5, 6, 19), p(6, 13.2, 14.5, 13.2, 17.5, 10.6, 17.5, 5)}, nil, sideStroke*s, c)
		r := 2.7 * s
		strokePaths(img, nil, [][3]float64{{ox + 6*s, oy + 5*s, r}, {ox + 17.5*s, oy + 5*s, r}, {ox + 6*s, oy + 19*s, r}}, 0, c)
	}
}

// fillShape paints c over img where in says a sample lies, 4x4 samples a
// pixel, through coverPixel: blended over what is drawn, drawn where at
// least half covered over what is not.
func fillShape(img *image.NRGBA, in func(px, py float64) bool, c color.NRGBA) {
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			n := 0
			for sy := range 4 {
				for sx := range 4 {
					if in(float64(x)+(float64(sx)+0.5)/4, float64(y)+(float64(sy)+0.5)/4) {
						n++
					}
				}
			}
			if n > 0 {
				coverPixel(img, x, y, n, c)
			}
		}
	}
}

// sideSixels are the tiles encoded, one a tile and look: eight at most.
var sideSixels sync.Map

// SideSixel is SideImage encoded as Sixel, made once a tile and look.
func SideSixel(t SideTileLook) string {
	if s, ok := sideSixels.Load(t); ok {
		return s.(string)
	}
	s := Sixel(SideImage(t))
	sideSixels.Store(t, s)
	return s
}
