package render

import (
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/Evengelius/agent-command-center/internal/state"
)

// Pos is a place in a panel's content (ui-design.md, Selection and copy): in
// zone B an entry, a logical line of it and a byte offset; in the editor a
// logical line and a byte offset; in the other panels a body row (Item) and
// a byte offset in what the row shows.
type Pos struct {
	Panel           byte
	Item, Line, Off int
}

// Before orders two places of the same panel.
func (p Pos) Before(q Pos) bool {
	if p.Item != q.Item {
		return p.Item < q.Item
	}
	if p.Line != q.Line {
		return p.Line < q.Line
	}
	return p.Off < q.Off
}

// Selection is a mouse selection in one panel, From before To.
type Selection struct{ From, To Pos }

// Empty is true when the selection covers nothing.
func (s Selection) Empty() bool { return s.From.Panel == 0 || !s.From.Before(s.To) }

// ConvView is what zone B showed of the conversation, for scrolling.
type ConvView struct {
	Rows int  // the conversation's rows at this width
	Room int  // the rows it is shown in
	Top  int  // the first row shown
	Y    int  // the screen row the first shown row is on
	Held bool // scrolled up: new rows do not move the view
	// Starts are each entry's first row, then the end of the entries: the
	// entry cursor keeps its entry in view.
	Starts []int
}

// Screen is a drawn frame that still knows what each cell shows, so a mouse
// position maps to content and a selection to its text. Hit-testing uses
// the spans' references, not the drawing (ui-design.md, Selection and copy).
type Screen struct {
	lines   []line
	rows    []string          // the lines as the terminal shows them, once drawn
	src     map[srcKey]string // the text of the panels other than B and E, by row
	Conv    ConvView
	Inspect ConvView // what the inspection showed of its tab, for scrolling
	// What zone A's tree, C's list and C's preview showed, for scrolling;
	// zero when not drawn.
	Tree, List, Preview PaneView
	// fileLines are the file the viewer shows, for its selection (panel F).
	fileLines []string
}

// PaneView is what a pane of zone A or C showed (plan 5, item 5).
type PaneView struct {
	Rows, Room, Top int
	IDs             []string // each row's node (A) or agent (C); "" for neither
}

type srcKey struct {
	panel byte
	item  int
}

// Draw draws the whole screen: exactly ui.Height lines, none wider than
// ui.Width, with ui.Sel highlighted.
func Draw(s *state.State, ui UI) *Screen {
	sc := &Screen{}
	var ls []line
	switch {
	case ui.Width < MinWidth || ui.Height < MinHeight:
		ls = tooSmall(s, ui)
	case ui.Worktrees.Open:
		ls = worktreesScreen(s, ui)
	case s.Screen == state.ScreenStartup && ui.Browser.Open:
		ls = browserScreen(s, ui)
	case s.Screen == state.ScreenDashboard && ui.Keys:
		ls = keysOverlay(s, ui, sc)
	case s.Screen == state.ScreenDashboard && ui.View != "" && s.Approvals[ui.View] != nil:
		ls = viewer(s, ui, s.Approvals[ui.View], sc)
	case s.Screen == state.ScreenDashboard && ui.ViewEntry > 0 && ui.ViewEntry <= len(s.Conversation):
		ls = entryViewer(s, ui, ui.ViewEntry-1, sc)
	case s.Screen == state.ScreenDashboard && ui.Inspect != "" && s.Agents[ui.Inspect] != nil:
		ls = inspect(s, ui, s.Agents[ui.Inspect], sc)
	case s.Screen == state.ScreenDashboard && ui.Details:
		ls = details(s, ui, sc)
	case s.Screen == state.ScreenDashboard:
		ls = dashboard(s, ui, sc)
	default:
		ls = startup(s, ui)
	}
	sc.lines = make([]line, ui.Height)
	for i := range sc.lines {
		var l line
		if i < len(ls) {
			l = ls[i]
		}
		sc.lines[i] = l.fit(ui.Width)
	}
	if s.Screen == state.ScreenDashboard && WindowOpen(ui) && !ui.Keys && !ui.Worktrees.Open &&
		ui.Width >= MinWidth && ui.Height >= MinHeight {
		floatWindow(s, ui, sc)
	}
	if !ui.Sel.Empty() && ui.Sel.From.Panel != 'E' {
		for i, l := range sc.lines {
			sc.lines[i] = highlight(l, ui.Sel)
		}
	}
	ui.Wrap.sweep()
	return sc
}

// String is the frame as the terminal shows it.
func (sc *Screen) String() string { return sc.StringAfter(nil) }

// StringAfter is String, taking from prev, the frame drawn before, each line
// drawn the same: a keystroke restyles the editor's rows, not the whole
// screen (ui-design.md, Budget). Spans hold their style by pointer, so equal
// spans render to the same text and a reused line is never stale.
func (sc *Screen) StringAfter(prev *Screen) string {
	sc.rows = make([]string, len(sc.lines))
	for i, l := range sc.lines {
		if prev != nil && i < len(prev.rows) && slices.Equal(l, prev.lines[i]) {
			sc.rows[i] = prev.rows[i]
		} else {
			sc.rows[i] = l.String()
		}
	}
	return strings.Join(sc.rows, "\n")
}

// highlight restyles the part of l that sel covers, splitting text spans at
// its ends.
func highlight(l line, sel Selection) line {
	var out line
	for _, x := range l {
		r := x.ref
		if r.panel != sel.From.Panel || r.kind != refText && r.kind != refAtom {
			out = append(out, x)
			continue
		}
		at := Pos{r.panel, r.item, r.line, r.off}
		if r.kind == refAtom {
			if !at.Before(sel.From) && at.Before(sel.To) {
				x.st = stSelected
			}
			out = append(out, x)
			continue
		}
		lo, hi := bound(sel.From, at, len(x.text)), bound(sel.To, at, len(x.text))
		if lo >= hi {
			out = append(out, x)
			continue
		}
		for _, part := range [][2]int{{0, lo}, {lo, hi}, {hi, len(x.text)}} {
			if part[0] == part[1] {
				continue
			}
			y := x
			y.text, y.ref.off = x.text[part[0]:part[1]], r.off+part[0]
			if part[0] == lo {
				y.st = stSelected
			}
			out = append(out, y)
		}
	}
	return out
}

// bound is where p falls in a text span of n bytes starting at at: 0 before
// it, n after it.
func bound(p, at Pos, n int) int {
	switch {
	case p.Item != at.Item || p.Line != at.Line:
		if p.Before(at) {
			return 0
		}
		return n
	}
	return min(max(p.Off-at.Off, 0), n)
}

// unit is one drawn unit of content on a row: a character with its marks,
// a tab, a control character, or an empty line's anchor.
type unit struct {
	x0, x1     int
	start, end Pos
}

// units are the content units of panel p on row y, left to right.
func (sc *Screen) units(p byte, y int) []unit {
	var out []unit
	x := 0
	for _, s := range sc.lines[y] {
		w := textWidth(s.text)
		r := s.ref
		if r.panel == p && (r.kind == refText || r.kind == refAtom) {
			pos := func(off int) Pos { return Pos{p, r.item, r.line, off} }
			switch {
			case r.kind == refAtom:
				out = append(out, unit{x, x + w, pos(r.off), pos(r.off + r.n)})
			case s.text == "":
				out = append(out, unit{x, x, pos(r.off), pos(r.off)})
			default:
				cx := x
				for i := 0; i < len(s.text); {
					_, n := utf8.DecodeRuneInString(s.text[i:])
					j := i + n
					for j < len(s.text) {
						r2, n2 := utf8.DecodeRuneInString(s.text[j:])
						if runeWidth(r2) != 0 {
							break
						}
						j += n2
					}
					cw := textWidth(s.text[i:j])
					out = append(out, unit{cx, cx + cw, pos(r.off + i), pos(r.off + j)})
					cx, i = cx+cw, j
				}
			}
		}
		x += w
	}
	return out
}

// PanelAt is the panel under the cell (x, y), 0 outside every panel. The
// editor's row belongs to the editor, its frame included.
func (sc *Screen) PanelAt(x, y int) byte {
	if y < 0 || y >= len(sc.lines) {
		return 0
	}
	cx, p := 0, byte(0)
	for _, s := range sc.lines[y] {
		w := textWidth(s.text)
		if x >= cx && x < cx+w {
			p = s.ref.panel
			break
		}
		cx += w
	}
	if p == 'B' {
		for _, s := range sc.lines[y] {
			if s.ref.panel == 'E' {
				return 'E'
			}
		}
	}
	return p
}

// HitIn maps the cell (x, y) to panel p's content, clamped to the panel: the
// unit under it (start and end), or the nearest place on its row. On a row
// without content it looks up for the previous content's end when back,
// else down for the next start, then the other way.
func (sc *Screen) HitIn(p byte, x, y int, back bool) (start, end Pos, ok bool) {
	top, bottom := -1, -1
	for i, l := range sc.lines {
		for _, s := range l {
			if s.ref.panel == p {
				if top < 0 {
					top = i
				}
				bottom = i
				break
			}
		}
	}
	if top < 0 {
		return Pos{}, Pos{}, false
	}
	y = min(max(y, top), bottom)
	if us := sc.units(p, y); len(us) > 0 {
		return pick(us, x)
	}
	dirs := []int{1, -1}
	if back {
		dirs = []int{-1, 1}
	}
	for _, d := range dirs {
		for i := y + d; i >= top && i <= bottom; i += d {
			if us := sc.units(p, i); len(us) > 0 {
				if d < 0 {
					u := us[len(us)-1]
					return u.end, u.end, true
				}
				return us[0].start, us[0].start, true
			}
		}
	}
	return Pos{}, Pos{}, false
}

// pick is the place x is at on a row of units: the unit under it, the row's
// start left of it, its end right of it, and the end of the unit before a
// gap.
func pick(us []unit, x int) (Pos, Pos, bool) {
	if x < us[0].x0 {
		return us[0].start, us[0].start, true
	}
	var prev *unit
	for i := range us {
		u := &us[i]
		if x >= u.x0 && x < u.x1 {
			return u.start, u.end, true
		}
		if u.x1 <= x {
			prev = u
		}
	}
	return prev.end, prev.end, true
}

// Text is the content sel covers: zone B's from the conversation itself and
// the session state after it, whole logical lines joined by line breaks and
// items by a blank line;
// the other panels' from what their rows show.
func (sc *Screen) Text(s *state.State, sel Selection) string {
	if sel.Empty() {
		return ""
	}
	f, t := sel.From, sel.To
	if f.Panel == 'B' {
		var parts []string
		for i := f.Item; i <= t.Item; i++ {
			ls := bLines(s, i)
			if len(ls) == 0 {
				continue
			}
			l0, o0, l1, o1 := 0, 0, len(ls)-1, len(ls[len(ls)-1])
			if i == f.Item {
				l0, o0 = f.Line, f.Off
			}
			if i == t.Item {
				l1, o1 = t.Line, t.Off
			}
			parts = append(parts, between(ls, l0, o0, l1, o1))
		}
		return strings.Join(parts, "\n\n")
	}
	if f.Panel == 'F' {
		if len(sc.fileLines) == 0 {
			return ""
		}
		return between(sc.fileLines, f.Item, f.Off, t.Item, t.Off)
	}
	var rows []string
	for i := f.Item; i <= t.Item; i++ {
		r := sc.source(f.Panel, i)
		lo, hi := 0, len(r)
		if i == f.Item {
			lo = min(f.Off, hi)
		}
		if i == t.Item {
			hi = min(t.Off, hi)
		}
		rows = append(rows, r[lo:max(lo, hi)])
	}
	return strings.Join(rows, "\n")
}

// bLines is zone B's item i: entry i's lines, or after the last entry the
// session state's (plan 5, item 9); nil for any other.
func bLines(s *state.State, i int) []string {
	switch {
	case i >= 0 && i < len(s.Conversation):
		return entryLines(s, s.Conversation[i])
	case i == len(s.Conversation):
		return sessionLines(s)
	}
	return nil
}

// between is the text of lines from (l0, o0) to (l1, o1), clamped.
func between(ls []string, l0, o0, l1, o1 int) string {
	l0, l1 = min(max(l0, 0), len(ls)-1), min(max(l1, 0), len(ls)-1)
	if l1 < l0 {
		return ""
	}
	o0, o1 = min(max(o0, 0), len(ls[l0])), min(max(o1, 0), len(ls[l1]))
	if l0 == l1 {
		return ls[l0][o0:max(o0, o1)]
	}
	parts := append([]string{ls[l0][o0:]}, ls[l0+1:l1]...)
	return strings.Join(append(parts, ls[l1][:o1]), "\n")
}

// LineText is the logical line p is on, for word and line selection.
func (sc *Screen) LineText(s *state.State, p Pos) string {
	if p.Panel == 'B' {
		if ls := bLines(s, p.Item); p.Line >= 0 && p.Line < len(ls) {
			return ls[p.Line]
		}
		return ""
	}
	if p.Panel == 'F' {
		if p.Item >= 0 && p.Item < len(sc.fileLines) {
			return sc.fileLines[p.Item]
		}
		return ""
	}
	return sc.source(p.Panel, p.Item)
}

// source is the text panel p's body row item shows, built on first use.
func (sc *Screen) source(p byte, item int) string {
	if sc.src == nil {
		sc.src = map[srcKey]string{}
		for _, l := range sc.lines {
			for _, s := range l {
				if r := s.ref; r.kind == refText && r.panel != 'B' && r.panel != 'E' && r.panel != 'F' {
					sc.src[srcKey{r.panel, r.item}] += s.text
				}
			}
		}
	}
	return sc.src[srcKey{p, item}]
}
