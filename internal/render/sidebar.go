package render

import (
	"strings"
	"sync"

	"charm.land/lipgloss/v2"
)

// The activity sidebar (plan 43, brief amendment 44, the owner's images
// references/12-sidebar-collapsed.png and 13-sidebar-explorer.webp): from
// 100 columns a box of the zones' thin grey border right of zone B on the
// upper row, no title, with two square tiles under its top edge, the
// Explorer's on its rows 1 to 3 and Skills' on its rows 5 to 7 (design D2),
// with Sixel images over blank cells (sidebarimage.go), else in text. A
// click on the Explorer's shows or hides the Explorer's panel, as F3 does;
// Skills' is visual only (design D5). The tile under the pointer is hovered
// (UI.SideHover, design D4), which never changes what is selected.

// SidePanel is the sidebar's panel letter; its tiles' cells carry their
// tile, SideExplorer or SideSkills, its frame's and blank cells 0.
const (
	SidePanel    = 'K'
	SideExplorer = 1
	SideSkills   = 2
)

// The sidebar's width, its frame included, and a tile's cells and rows:
// with Sixel its image's 60x60 pixels (design D2).
const (
	sideWidth     = 8
	SideTileCells = 6
	SideTileRows  = 3
)

// sideTiles are the tiles in order with the box row each starts on.
var sideTiles = []struct{ tile, row int }{{SideExplorer, 1}, {SideSkills, 5}}

// SideLook is a tile's look (plan 43, design D3).
type SideLook int

// The looks: inactive, hovered, selected (the Explorer's while its panel is
// shown), selected and hovered.
const (
	SideInactive SideLook = iota
	SideHover
	SideSelected
	SideSelectedHover
)

// sideLook is tile's look on this frame: selected while it is the
// Explorer's and the panel is shown, hovered while the pointer is over it;
// a hover never takes the selection away (design D4).
func sideLook(ui UI, tile int) SideLook {
	sel, hover := tile == SideExplorer && ui.Explorer.Open, ui.SideHover == tile
	switch {
	case sel && hover:
		return SideSelectedHover
	case sel:
		return SideSelected
	case hover:
		return SideHover
	}
	return SideInactive
}

// sideGlyph is each tile's icon in text, one cell wide: "≡" from Cascadia
// Mono for the Explorer, "⑂" for Skills, which Cascadia Mono lacks and
// Windows Terminal takes from Segoe UI Symbol, narrower than a cell there
// (plan 43, design D2; measured from the fonts' tables).
var sideGlyph = map[int]string{SideExplorer: "≡", SideSkills: "⑂"}

// sideKey is what the sidebar's rows depend on: its size, its tiles' looks
// in text (with Sixel the images carry them, the cells the same in every
// look), Sixel, and the palette in use (a ground's tints are true colour
// only).
type sideKey struct {
	w, h           int
	looks          [2]SideLook
	sixel, sixteen bool
}

// sideRows keeps the sidebar's rows drawn, by their key: a frame draws them
// again only when a look, the height or the palette changes, so a
// keystroke costs the sidebar no allocation (ui-design.md, Budget; test 6's
// bound). The rows are only read: beside copies their spans.
var sideRows = struct {
	sync.Mutex
	m map[sideKey][]line
}{m: map[sideKey][]line{}}

// sidebar is the sidebar w by h: its frame, the tiles in their looks, blank
// rows elsewhere; its rows are shared, never to be changed.
func sidebar(ui UI, w, h int) []line {
	k := sideKey{w: w, h: h, sixel: ui.Sixel, sixteen: sixteen}
	if !ui.Sixel {
		k.looks = [2]SideLook{sideLook(ui, SideExplorer), sideLook(ui, SideSkills)}
	}
	sideRows.Lock()
	defer sideRows.Unlock()
	if ls, ok := sideRows.m[k]; ok {
		return ls
	}
	if len(sideRows.m) >= 64 { // a few heights and looks; resizes add more
		clear(sideRows.m)
	}
	ls := drawSidebar(ui, w, h)
	sideRows.m[k] = ls
	return ls
}

// drawSidebar draws the sidebar w by h.
func drawSidebar(ui UI, w, h int) []line {
	if w < 2 || h < 2 {
		return in(SidePanel, blank(w, h))
	}
	inner := w - 2
	edge := func(l, r string) line { return line{sideDeco(l+strings.Repeat("─", inner)+r, stFrame, 0)} }
	rows := make([]line, h-2)
	for _, t := range sideTiles {
		for i, l := range sideTile(t.tile, sideLook(ui, t.tile), ui.Sixel) {
			if r := t.row - 1 + i; r < len(rows) {
				rows[r] = l
			}
		}
	}
	out := []line{edge("╭", "╮")}
	for _, r := range rows {
		// The tile centred in the box's inner cells, the rest blank.
		left := max(inner-r.width(), 0) / 2
		row := line{sideDeco("│", stFrame, 0)}
		row = append(append(row, sideBlank(left)...), r...)
		row = append(row, sideBlank(inner-left-r.width())...)
		out = append(out, append(row, sideDeco("│", stFrame, 0)))
	}
	return append(out, edge("╰", "╯"))
}

// sideBlank is n blank cells of the sidebar outside its tiles; none for
// n under 1.
func sideBlank(n int) line {
	if n < 1 {
		return nil
	}
	return line{sideDeco(strings.Repeat(" ", n), stNone, 0)}
}

// sideTile is a tile's SideTileRows rows, SideTileCells cells each, in its
// look (design D2). With Sixel they are blank, the same in every look, for
// the image the application draws over them: a cell the renderer writes
// again erases the image over it (plan 35, design D2), so a hover or the
// selection changes the image alone. In text, its icon alone on the middle
// row in the muted grey while inactive; on the hover band, or on the
// selected ground with a lavender "▎" in its left cell, the icon lit.
// Every cell carries the tile, for the mouse.
func sideTile(tile int, look SideLook, sixel bool) []line {
	if sixel {
		out := make([]line, SideTileRows)
		for r := range out {
			out[r] = line{sideDeco(strings.Repeat(" ", SideTileCells), stNone, tile)}
		}
		return out
	}
	var g *ground
	switch look {
	case SideHover:
		g = bandGround
	case SideSelected:
		g = sideSelGround
	case SideSelectedHover:
		g = sideSelHoverGround
	}
	on := func(st *lipgloss.Style) *lipgloss.Style {
		if g == nil {
			return st
		}
		return g.on(st)
	}
	lead, icon := sideDeco(" ", on(stNone), tile), sideDeco(sideGlyph[tile], on(stSideIconLit), tile)
	switch look {
	case SideInactive:
		icon.st = stSideIcon
	case SideSelected, SideSelectedHover:
		lead = sideDeco("▎", on(stSideAccent), tile)
	}
	at := (SideTileCells - 1) / 2 // the icon's cell, the tile's middle on its left
	out := make([]line, SideTileRows)
	for r := range out {
		if r != SideTileRows/2 {
			out[r] = line{lead, sideDeco(strings.Repeat(" ", SideTileCells-1), on(stNone), tile)}
			continue
		}
		out[r] = line{lead, sideDeco(strings.Repeat(" ", at-1), on(stNone), tile), icon,
			sideDeco(strings.Repeat(" ", SideTileCells-at-1), on(stNone), tile)}
	}
	return out
}

// sideDeco is a span of the sidebar: decoration, panel K, item its tile.
func sideDeco(text string, st *lipgloss.Style, item int) span {
	return span{text: text, st: st, ref: ref{panel: SidePanel, kind: refDeco, item: item}}
}

// SideTile is the sidebar's tile at the frame's cell (x, y), SideExplorer
// or SideSkills; 0 off the tiles, on the sidebar's frame or its blank
// cells (plan 43, design D4): the tile a pointer there hovers.
func (sc *Screen) SideTile(x, y int) int {
	if sc.Side.W == 0 || x < sc.Side.X || x >= sc.Side.X+sc.Side.W {
		return 0
	}
	if p, item := sc.ItemAt(x, y); p == SidePanel {
		return item
	}
	return 0
}

// SideTileShown is where the frame left a tile's image to be drawn and the
// look it shows (design D2); Shown is false when the frame has none: no
// sidebar, no Sixel, or a window over the tile's cells.
type SideTileShown struct {
	Shown bool
	X, Y  int
	Look  SideTileLook
}

// sideTilesShown records in sc.Tiles each tile's image to be drawn, with
// Sixel: at its first cell, on the sidebar's rows from its top edge, the
// upper row's top, zone A's (sc.AY), when the whole of its cells are still
// the tile's once the popups and windows are drawn; a full-screen window
// covers them.
func (sc *Screen) sideTilesShown(ui UI) {
	if !ui.Sixel || sc.Side.W < SideTileCells+2 {
		return
	}
	x := sc.Side.X + 1 + (sc.Side.W-2-SideTileCells)/2 // centred, as drawSidebar draws it
	for i, t := range sideTiles {
		y, whole := sc.AY+t.row, true
		for r := range SideTileRows {
			whole = whole && sc.cellsOf(x, y+r, SideTileCells, SidePanel, t.tile)
		}
		if whole {
			sc.Tiles[i] = SideTileShown{Shown: true, X: x, Y: y, Look: SideTileLook{Tile: t.tile, Look: sideLook(ui, t.tile)}}
		}
	}
}

// cellsOf says the cells [x, x+w) of the frame's row y all belong to spans
// of panel p carrying item.
func (sc *Screen) cellsOf(x, y, w int, p byte, item int) bool {
	if y < 0 || y >= len(sc.lines) {
		return false
	}
	n, cx := 0, 0
	for _, s := range sc.lines[y] {
		sw := textWidth(s.text)
		if lo, hi := max(x, cx), min(x+w, cx+sw); lo < hi {
			if s.ref.panel != p || s.ref.item != item {
				return false
			}
			n += hi - lo
		}
		if cx += sw; cx >= x+w {
			break
		}
	}
	return n == w
}
