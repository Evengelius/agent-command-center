package app

import (
	"math"
	"slices"

	"github.com/Evengelius/agent-command-center/internal/render"
)

// Scrolling zone A's tree, zone C's list and C's preview (ui-design.md,
// Scrolling, collapsing, full screen; plan 5, item 5). A pane rests where it
// always did; scrolled, it is held at its top. The tree and the preview rest
// at their tail and follow again there; the list rests at its top.

// tail is a held top past every bottom: the pane shows its last rows, and
// keeps showing them as rows come (render.paneTop clamps it).
const tail = math.MaxInt

// pane is the owner's scroll of pane p ('A' the tree, 'C' the list, 'P' the
// preview) and what the last frame showed of it; nil before a frame.
func (m *Model) pane(p byte) (*render.Pane, *render.PaneView) {
	if m.screen == nil {
		return nil, nil
	}
	switch p {
	case 'A':
		return &m.ui.Tree, &m.screen.Tree
	case 'C':
		return &m.ui.List, &m.screen.List
	case 'P':
		return &m.ui.Preview, &m.screen.Preview
	}
	return nil, nil
}

// paneScroll moves pane p d rows; a pane that fits does not move.
func (m *Model) paneScroll(p byte, d int) {
	ui, v := m.pane(p)
	if ui == nil || v.Rows <= v.Room {
		return
	}
	bottom := v.Rows - v.Room
	top := v.Top
	if ui.Hold {
		top = min(max(ui.Top, 0), bottom)
	}
	m.paneAt(p, min(max(top+d, 0), bottom))
}

// panePage moves pane p a page, its rows less one.
func (m *Model) panePage(p byte, down bool) {
	_, v := m.pane(p)
	if v == nil {
		return
	}
	d := max(v.Room-1, 1)
	if !down {
		d = -d
	}
	m.paneScroll(p, d)
}

// paneAt holds pane p at top. At its bottom the tree and the preview follow
// again: they rest there unless the tree's cursor is above it, when the tree
// stays held at its tail.
func (m *Model) paneAt(p byte, top int) {
	ui, v := m.pane(p)
	if ui == nil {
		return
	}
	bottom := max(v.Rows-v.Room, 0)
	if top >= bottom && p != 'C' {
		*ui = render.Pane{}
		if p == 'A' {
			if i := slices.Index(v.IDs, m.ui.ACursor); m.ui.ACursor != "" && i >= 0 && i < bottom {
				*ui = render.Pane{Top: tail, Hold: true}
			}
		}
		v.Top = bottom // a second step before the next frame starts here
		return
	}
	*ui = render.Pane{Top: top, Hold: true}
	v.Top = min(top, bottom)
}

// paneKey scrolls pane p for PgUp, PgDn, Home and End; false for any other
// key.
func (m *Model) paneKey(p byte, key string) bool {
	switch key {
	case "pgup", "pgdown":
		m.panePage(p, key == "pgdown")
	case "home":
		m.paneAt(p, 0)
	case "end":
		m.paneAt(p, tail)
	default:
		return false
	}
	return true
}

// paneReveal keeps id in held pane p's view, moving it no more than needed; a
// pane at rest keeps its cursor in view itself.
func (m *Model) paneReveal(p byte, id string) {
	ui, v := m.pane(p)
	if ui == nil || !ui.Hold || v.Room <= 0 {
		return
	}
	i := slices.Index(v.IDs, id)
	if i < 0 {
		return
	}
	top := min(max(ui.Top, 0), max(v.Rows-v.Room, 0))
	switch {
	case i < top:
		m.paneAt(p, i)
	case i >= top+v.Room:
		m.paneAt(p, i-v.Room+1)
	}
}

// selectAgent selects agent key in zone C; another agent's preview starts at
// its tail.
func (m *Model) selectAgent(key string) {
	if key != m.ui.Selected {
		m.ui.Preview = render.Pane{}
	}
	m.ui.Selected = key
}
