package render

import (
	"strconv"
	"strings"
)

// The Git Graph's scope popup (plan 62, design D3): under the panel's
// scope row, a box listing All and Auto, then the local branches, the
// remote branches and the tags with check boxes, searched and folded as
// the branches popup's; Enter applies the scope ticked, Esc and a click
// outside keep the one there was. Its cells are panel V.

const (
	ScopePanel = 'V'

	ItemScopeFrame = 1    // the box's other cells: a click there does nothing
	ItemScopeRow   = 1000 // plus the row's index in ScopeRows
)

// The scope popup's groups, in their order.
const (
	ScopeLocal = iota
	ScopeRemote
	ScopeTags
	ScopeGroups
)

var (
	scopeGroupNames = [ScopeGroups]string{"Local branches", "Remote branches", "Tags"}
	scopeGroupEmpty = [ScopeGroups]string{"No local branches", "No remote branches", "No tags"}
)

// ScopeRef is a reference as the popup lists it: its full name, the name
// shown, its commit's short hash.
type ScopeRef struct {
	Name, Short, Hash string
}

// ScopeView is the popup (design D3): the references read, what is ticked,
// the search, the folds and the cursor.
type ScopeView struct {
	Open bool
	// Reading says git's read runs; Problem what stopped it.
	Reading bool
	Problem string
	// Groups are the local branches, the remote branches and the tags.
	Groups [ScopeGroups][]ScopeRef
	// All and Checked are the scope being chosen: All, the references ticked
	// (full names), or neither, Auto.
	All     bool
	Checked map[string]bool
	// Folded says a group is folded; Query is the search typed.
	Folded [ScopeGroups]bool
	Query  string
	// Cursor is the key of the cursor's row (ScopeItem.Key); Top the list's
	// first row shown.
	Cursor string
	Top    int
}

// ScopeItem is one row of the popup's list: All ('a'), Auto ('u'), a
// group's heading ('g'), a reference ('r') of it, or a message ('m').
type ScopeItem struct {
	Kind  byte
	Group int // -1 for All, Auto and the message that nothing matches
	Ref   ScopeRef
	Text  string // a message's
}

// Key names the row for the cursor: "a", "u", "g:1", "r:refs/heads/main";
// a message has none.
func (it ScopeItem) Key() string {
	switch it.Kind {
	case 'a', 'u':
		return string(it.Kind)
	case 'g':
		return "g:" + strconv.Itoa(it.Group)
	case 'r':
		return "r:" + it.Ref.Name
	}
	return ""
}

// ScopeRows is the popup's list: All and Auto, then each group's heading
// and, open, its references or its message. While a search is typed, case
// ignored, All and Auto stay, every group with a match shows open, one with
// none is hidden, and nothing matching is one message.
func ScopeRows(v ScopeView) []ScopeItem {
	q := strings.ToLower(v.Query)
	out := []ScopeItem{{Kind: 'a', Group: -1}, {Kind: 'u', Group: -1}}
	any := false
	for g := range ScopeGroups {
		hits := v.Groups[g]
		if q != "" {
			hits = nil
			for _, r := range v.Groups[g] {
				if strings.Contains(strings.ToLower(r.Short), q) {
					hits = append(hits, r)
				}
			}
			if len(hits) == 0 {
				continue
			}
		}
		any = true
		out = append(out, ScopeItem{Kind: 'g', Group: g})
		if q == "" && v.Folded[g] {
			continue
		}
		if len(hits) == 0 {
			out = append(out, ScopeItem{Kind: 'm', Group: g, Text: scopeGroupEmpty[g]})
			continue
		}
		for _, r := range hits {
			out = append(out, ScopeItem{Kind: 'r', Group: g, Ref: r})
		}
	}
	if q != "" && !any {
		out = append(out, ScopeItem{Kind: 'm', Group: -1, Text: "No reference matches “" + v.Query + "”"})
	}
	return out
}

// ScopeCursor is the index in rows of the row key names, -1 when none.
func ScopeCursor(rows []ScopeItem, key string) int {
	for i, r := range rows {
		if r.Kind != 'm' && r.Key() == key {
			return i
		}
	}
	return -1
}

// ScopeShown is what the frame showed of the popup, for the wheel and the
// clicks: the list's first row, its height and its length.
type ScopeShown struct {
	Shown             bool
	Top, Height, Rows int
}

// The box's limits: its width and how many rows its list shows at most.
const (
	scopeMinW = 52
	scopeMaxW = 72
	scopeMaxH = 18
)

// scopeKeys are the footer's key hints, the longest first.
var scopeKeys = []string{
	"↑↓ move · Space tick · → open · ← fold · Enter apply · Esc cancel",
	"Space tick · Enter apply · Esc cancel",
	"Enter apply · Esc cancel",
}

// scopePopup draws the popup under the Git Graph's scope row (design D3):
// its right edge at the panel's, further left when the screen's left edge
// pushes it, its top on the row under the scope row. It draws nothing when
// the scope row is not on screen or the rows under it leave the list none.
func scopePopup(sc *Screen, ui UI) {
	v := ui.Scope
	sx, sy, ok := sc.findItem(GraphPanel, ItemGraphScope)
	if !ok {
		return
	}
	rows := ScopeRows(v)
	w := min(max(scopeMinW, scopeWidth(rows)), scopeMaxW, ui.Width)
	inner := w - 2
	right := sc.Panel.X + sc.Panel.W
	if right <= 0 {
		right = ui.Width
	}
	x := max(min(right-w, ui.Width-w), 0)
	if sx < x {
		x = max(sx-1, 0)
	}
	y := sy + 1
	foot := []line{padRow(decoLine(" "+fitHint(scopeKeys, inner-1), stGrey), inner)}
	room := min(ui.Height-y-4-len(foot), scopeMaxH)
	if room < 1 {
		return
	}
	n, top, h := 1, 0, 1
	var list []line
	switch {
	case v.Problem != "":
		list = []line{padRow(line{deco("  ", stNone), deco(fitText(v.Problem, inner-3), stAttention)}, inner)}
	case v.Reading:
		list = []line{padRow(line{deco("  reading…", stGrey)}, inner)}
	default:
		n = len(rows)
		h = min(n, room)
		cur := ScopeCursor(rows, v.Cursor)
		top = ListTop(v.Top, max(cur, 0), h, n)
		for i := top; i < top+h; i++ {
			l := scopeItemRow(v, rows[i], inner)
			if i == cur {
				l = banded(l, inner)
			}
			for j := range l {
				l[j].ref = ref{panel: ScopePanel, kind: refDeco, item: ItemScopeRow + i}
			}
			list = append(list, l)
		}
	}
	sc.Scope = ScopeShown{Shown: true, Top: top, Height: h, Rows: n}

	frame := func(l line, edge string) line {
		out := line{deco("│", stModelFrame)}
		out = append(out, padRow(l, inner)...)
		return append(out, deco(edge, stModelFrame))
	}
	title := " History scope "
	box := []line{
		{deco("╭─", stModelFrame), deco(title, stModelHead), deco(strings.Repeat("─", max(inner-1-textWidth(title), 0))+"╮", stModelFrame)},
		frame(scopeSearch(v, inner), "│"),
	}
	from, to := scrollThumb(n, h, top)
	for i := 0; i < h; i++ {
		edge := "│"
		if i >= from && i < to {
			edge = "┃"
		}
		box = append(box, frame(list[i], edge))
	}
	for _, l := range foot {
		box = append(box, frame(l, "│"))
	}
	box = append(box, line{deco("╰"+strings.Repeat("─", inner)+"╯", stModelFrame)})
	for i := range box {
		for j := range box[i] {
			if box[i][j].ref.panel == 0 {
				box[i][j].ref = ref{panel: ScopePanel, kind: refDeco, item: ItemScopeFrame}
			}
		}
	}
	sc.place(box, x, y, w)
}

// scopeWidth is the widest row's cells, the frame included, measured
// without drawing.
func scopeWidth(rows []ScopeItem) int {
	w := 0
	for _, r := range rows {
		n := 0
		switch r.Kind {
		case 'a', 'u':
			n = 4 + textWidth(scopeModeText(r.Kind))
		case 'r':
			n = 6 + textWidth(r.Ref.Short) + 2 + textWidth(scopeAside(r))
		default:
			n = 4 + textWidth(r.Text)
		}
		if w = max(w, n+2); w >= scopeMaxW {
			break
		}
	}
	return w
}

// scopeModeText is All's or Auto's row text.
func scopeModeText(kind byte) string {
	if kind == 'a' {
		return "All · every branch, remote branch and tag"
	}
	return "Auto · the current branch and its upstream"
}

// scopeAside is a reference's row's right part: its commit's short hash,
// "Remote branch at" before a remote one's.
func scopeAside(r ScopeItem) string {
	if r.Group == ScopeRemote {
		return "Remote branch at " + r.Ref.Hash
	}
	return r.Ref.Hash
}

// scopeItemRow is a row of the list, inner cells: All and Auto with their
// mark, "◉" the mode, "○" otherwise; a heading with its fold mark and its
// count; a reference with its check box, "☑" ticked, "☐" not, its name and
// its commit; a message in grey.
func scopeItemRow(v ScopeView, it ScopeItem, inner int) line {
	switch it.Kind {
	case 'a', 'u':
		mark := "○ "
		if it.Kind == 'a' && v.All || it.Kind == 'u' && !v.All && len(v.Checked) == 0 {
			mark = "◉ "
		}
		return padRow(line{deco(" "+mark, stTitle), deco(fitText(scopeModeText(it.Kind), inner-3), stText)}, inner)
	case 'g':
		fold := "▾ "
		if v.Folded[it.Group] && v.Query == "" {
			fold = "▸ "
		}
		count := " " + strconv.Itoa(len(v.Groups[it.Group]))
		return padRow(line{deco(" "+fold, stTitle), deco(scopeGroupNames[it.Group], stTitleBold), deco(count, stMuted)}, inner)
	case 'r':
		box := "☐ "
		if v.Checked[it.Ref.Name] {
			box = "☑ "
		}
		aside := "  " + scopeAside(it)
		name := fitText(it.Ref.Short, max(inner-5-textWidth(aside), 4))
		l := line{deco("   ", stNone), deco(box, stTitle), deco(name, stText)}
		if l.width()+textWidth(aside) <= inner {
			l = append(l, deco(aside, stMuted))
		}
		return padRow(l, inner)
	}
	return padRow(line{deco("   "+fitText(it.Text, inner-3), stGrey)}, inner)
}

// scopeSearch is the search row: the query and its cursor, or the hint.
func scopeSearch(v ScopeView, inner int) line {
	if v.Query == "" {
		return padRow(line{deco(" ⌕ ", stTitle), deco("type to search the references", stGrey)}, inner)
	}
	return padRow(line{deco(" ⌕ ", stTitle), deco(fitText(v.Query, inner-4), stText), deco("▌", stText)}, inner)
}
