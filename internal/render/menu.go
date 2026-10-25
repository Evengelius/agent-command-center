package render

import (
	"strings"

	"charm.land/lipgloss/v2"
)

// The "+" menu (plan 30, brief amendment 30, the owner's image;
// reference-design.md, The "+" menu): a click on the editor row's band
// opens a three-row box above it, its bottom edge on the editor's rule,
// with one item, a folder icon and "Upload from computer". The item lies on
// the active ground while the pointer is over it. Nothing acts behind it
// yet (design D4).
const (
	// The folder icon, U+1F5C0, one cell by its width table, which Windows
	// Terminal draws over two: two spaces follow it, so one shows either way.
	menuItem  = "🗀  Upload from computer"
	menuInner = 26 // the box's inner cells: the item and its padding
)

// menuPanel is the item's panel, the band's being '+'.
const menuPanel = 'U'

// plusMenu draws the menu over the dashboard when it is open: its left edge
// on the band's cell, its bottom edge on the editor's rule. It draws
// nothing when the band or the rule is not on screen, the rows above the
// rule are too few, or it would pass zone B's right edge, over the
// Explorer's panel or the sidebar (plan 43, design D9).
func plusMenu(sc *Screen, ui UI) {
	px, py, ok := sc.find('+')
	if !ok {
		return
	}
	ry := sc.editorRule(py)
	if ry < 2 || px+menuInner+2 > sc.bRight(ui.Width) {
		return
	}
	border := func(l, r string) line { return line{menuDeco(l+strings.Repeat("─", menuInner)+r, stFrame)} }
	st := stThinkEdge
	if ui.PlusHover {
		st = active(stThinkEdge)
	}
	inner := padCell("  "+menuItem, menuInner) // two cells before, one after
	item := line{menuDeco("│", stFrame), menuDeco(inner, st), menuDeco("│", stFrame)}
	sc.place([]line{border("╭", "╮"), item, border("╰", "╯")}, px, ry-2, menuInner+2)
	sc.Plus = true
}

// menuDeco is a span of the menu: decoration, panel U.
func menuDeco(text string, st *lipgloss.Style) span {
	return span{text: text, st: st, ref: ref{panel: menuPanel, kind: refDeco}}
}

// find is the cell of the first span of panel p on the screen, from the
// bottom up.
func (sc *Screen) find(p byte) (x, y int, ok bool) {
	for y := len(sc.lines) - 1; y >= 0; y-- {
		x := 0
		for _, s := range sc.lines[y] {
			if s.ref.panel == p {
				return x, y, true
			}
			x += textWidth(s.text)
		}
	}
	return 0, 0, false
}

// editorRule is the row of the editor's rule above the band's row, as zone
// B recorded it when it drew the editor (plan 49, design D7); -1 when it
// drew none above that row.
func (sc *Screen) editorRule(bandY int) int {
	if sc.rule > 0 && sc.rule < bandY-1 {
		return sc.rule
	}
	return -1
}
