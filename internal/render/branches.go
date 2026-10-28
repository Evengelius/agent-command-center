package render

import (
	"strconv"
	"strings"

	"charm.land/lipgloss/v2"
)

// The branches popup (plan 59, brief amendment 61; ui-design.md, Branches):
// under the project bar's branch name, a box of the repository's branches
// and tags, searched and folded, drawn from what the application read. The
// branch name's cells are panel L, where a click opens it; the box's are
// panel J.

const (
	BranchPanel   = 'L'
	BranchesPanel = 'J'

	ItemBranchFrame = 1    // the box's other cells: a click there does nothing
	ItemBranchFetch = 2    // "fetch" on the search row
	ItemBranchRow   = 1000 // plus the row's index in BranchRows
)

// The groups, in their order (design D2).
const (
	GroupRecent = iota
	GroupLocal
	GroupRemote
	GroupTags
	BranchGroups
)

var (
	branchGroupNames = [BranchGroups]string{"Recent", "Local", "Remote", "Tags"}
	branchGroupEmpty = [BranchGroups]string{"No recent branches", "No local branches", "No remote branches", "No tags"}
)

// The popup's icons where plan 55's rule draws the Explorer's (UI.NerdIcons),
// with their name in Nerd Fonts 3.2.1's glyphnames.json; elsewhere the rows
// have none and the search row says "F5 fetch".
var (
	nfBranch  = newExpIcon(0xF418, cIconBlue)   // oct-git_branch
	nfTag     = newExpIcon(0xEA66, cIconYellow) // cod-tag
	nfRefresh = newExpIcon(0xEB37, cModelFrame) // cod-refresh
)

// BranchRow is a branch or a tag as the popup shows it.
type BranchRow struct {
	Name string
	// Upstream is a local branch's, short; Ahead and Behind its counts; Gone
	// says the upstream is gone.
	Upstream      string
	Ahead, Behind int
	Gone          bool
	// Current says HEAD is on it; Elsewhere names the other worktree that
	// has it checked out (design D5).
	Current   bool
	Elsewhere string
}

// BranchesView is the popup (designs D2 to D10).
type BranchesView struct {
	Open bool
	// Reading says git's read runs; Problem is what stopped it, shown in
	// place of the groups.
	Reading bool
	Problem string
	// The current state: Head is the branch, ID HEAD's commit while
	// Detached; NoCommits; the branch's Upstream and counts.
	Head, ID            string
	Detached, NoCommits bool
	Upstream            string
	Ahead, Behind       int
	Gone                bool
	// Groups are Recent's, Local's, Remote's and Tags' rows, Totals how many
	// each had before git's read was capped.
	Groups [BranchGroups][]BranchRow
	Totals [BranchGroups]int
	// Folded says a group is folded; Query is the search typed.
	Folded [BranchGroups]bool
	Query  string
	// Cursor is the key of the cursor's row (BranchItem.Key); Top the list's
	// first row shown.
	Cursor string
	Top    int
	// Confirm is the branch a switch waits to go to, Changes the
	// uncommitted changes the confirmation names (design D5).
	Confirm string
	Changes int
	// Busy is what runs ("switching…", "fetching… Esc stops"); Note the last
	// result, NoteBad when it is a failure.
	Busy, Note string
	NoteBad    bool
}

// BranchItem is one row of the popup's list: a group's heading ('g'), a
// branch ('b') or a tag ('t') of it, or a message ('m').
type BranchItem struct {
	Kind  byte
	Group int // -1 for the message that nothing matches
	Row   BranchRow
	Text  string // a message's
}

// Key names the row for the cursor: "g:1", "b:1:main", "t:3:v1"; a message
// has none. It is made when asked: a list holds thousands of rows.
func (it BranchItem) Key() string {
	switch it.Kind {
	case 'g':
		return "g:" + strconv.Itoa(it.Group)
	case 'b', 't':
		return string(it.Kind) + ":" + strconv.Itoa(it.Group) + ":" + it.Row.Name
	}
	return ""
}

// BranchesShown is what the frame showed of the popup, for the wheel and
// the clicks: the list's first row, its height and its length.
type BranchesShown struct {
	Shown             bool
	Top, Height, Rows int
}

// BranchShown says the frame drew the project bar's branch name, under
// which the popup opens (design D1).
func (sc *Screen) BranchShown() bool {
	_, _, ok := sc.find(BranchPanel)
	return ok
}

// BranchRows is the popup's list (designs D2, D3): each group's heading,
// then, open, its rows or its message and the cap's note. While a search is
// typed, case ignored, every group with a match shows open, one with none is
// hidden, and nothing matching is one message.
func BranchRows(v BranchesView) []BranchItem {
	q := strings.ToLower(v.Query)
	// Room for every row shown at once, a heading and a message each, and
	// nothing matching: the list is made each frame, a group holding 1,000.
	n := 2*BranchGroups + 1
	for g := range BranchGroups {
		if q != "" || !v.Folded[g] {
			n += len(v.Groups[g])
		}
	}
	out := make([]BranchItem, 0, n)
	for g := range BranchGroups {
		hits := v.Groups[g]
		if q != "" {
			hits = nil
			for _, r := range v.Groups[g] {
				if strings.Contains(strings.ToLower(r.Name), q) {
					hits = append(hits, r)
				}
			}
			if len(hits) == 0 {
				continue
			}
		}
		out = append(out, BranchItem{Kind: 'g', Group: g})
		if q == "" && v.Folded[g] {
			continue
		}
		if len(hits) == 0 {
			out = append(out, BranchItem{Kind: 'm', Group: g, Text: branchGroupEmpty[g]})
			continue
		}
		kind := byte('b')
		if g == GroupTags {
			kind = 't'
		}
		for _, r := range hits {
			out = append(out, BranchItem{Kind: kind, Group: g, Row: r})
		}
		if n := len(v.Groups[g]); v.Totals[g] > n {
			out = append(out, BranchItem{Kind: 'm', Group: g, Text: "first " + grouped(n) + " shown"})
		}
	}
	if q != "" && len(out) == 0 {
		out = append(out, BranchItem{Kind: 'm', Group: -1, Text: "No branch or tag matches “" + v.Query + "”"})
	}
	return out
}

// BranchCursor is the index in rows of the row key names (BranchItem.Key),
// -1 when none: the key read once, the rows compared by their fields.
func BranchCursor(rows []BranchItem, key string) int {
	if len(key) < 3 || key[1] != ':' {
		return -1
	}
	kind := key[0]
	gs, name, _ := strings.Cut(key[2:], ":")
	g, err := strconv.Atoi(gs)
	if err != nil || kind != 'g' && kind != 'b' && kind != 't' {
		return -1
	}
	for i, r := range rows {
		if r.Kind == kind && r.Group == g && (kind == 'g' || r.Row.Name == name) {
			return i
		}
	}
	return -1
}

// The box's limits: its width, wide enough at least for the second keys'
// hint, and how many rows its list shows at most.
const (
	branchesMinW = 64
	branchesMaxW = 84
	branchesMaxH = 20
)

// branchesKeys are the footer's key hints, the longest first.
var branchesKeys = []string{
	"↑↓ move · → open · ← fold · Enter switch · Ctrl+C copy · F5 fetch · Esc close",
	"↑↓ move · Enter switch · Ctrl+C copy · F5 fetch · Esc close",
	"Enter switch · F5 fetch · Esc close",
	"Esc close",
}

// branchesPopup draws the popup under the project bar's branch name (design
// D2): its left edge one cell left of the name, further left when the
// screen's right edge pushes it, its top on the row under the bar. It draws
// nothing when the name is not on screen or the rows under it leave none for
// the list.
func branchesPopup(sc *Screen, ui UI) {
	v := ui.Branches
	bx, by, ok := sc.find(BranchPanel)
	if !ok {
		return
	}
	nerd := ui.NerdIcons
	rows := BranchRows(v)
	w := branchesWidth(v, rows, nerd, ui.Width)
	inner := w - 2
	x := max(min(bx-1, ui.Width-w), 0)
	y := by + 1
	foot := branchesFoot(v, inner)
	room := min(ui.Height-y-4-len(foot), branchesMaxH)
	if room < 1 {
		return
	}
	// Only the rows in view are drawn: a group holds up to 1,000.
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
		cur := BranchCursor(rows, v.Cursor)
		top = ListTop(v.Top, max(cur, 0), h, n)
		for i := top; i < top+h; i++ {
			l := branchItemRow(v, rows[i], nerd, inner)
			if i == cur {
				l = banded(l, inner)
			}
			list = append(list, branchItems(l, ItemBranchRow+i))
		}
	}
	sc.Branches = BranchesShown{Shown: true, Top: top, Height: h, Rows: n}

	frame := func(l line, right string) line {
		out := line{deco("│", stModelFrame)}
		out = append(out, padRow(l, inner)...)
		return append(out, deco(right, stModelFrame))
	}
	title := " Branches "
	box := []line{
		{deco("╭─", stModelFrame), deco(title, stModelHead), deco(strings.Repeat("─", max(inner-1-textWidth(title), 0))+"╮", stModelFrame)},
		frame(branchesSearch(v, nerd, inner), "│"),
		frame(branchesState(v, inner), "│"),
	}
	from, to := scrollThumb(n, h, top)
	for i := 0; i < h; i++ {
		edge := "│"
		if i >= from && i < to {
			edge = "┃" // the list's scroll bar, as the model selector's
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
				box[i][j].ref = ref{panel: BranchesPanel, kind: refDeco, item: ItemBranchFrame}
			}
		}
	}
	sc.place(box, x, y, w)
}

// branchItems gives each span of l the popup's panel and the item.
func branchItems(l line, item int) line {
	for j := range l {
		l[j].ref = ref{panel: BranchesPanel, kind: refDeco, item: item}
	}
	return l
}

// branchesWidth is the box's width, frame included: its widest row, within
// branchesMinW and branchesMaxW, and the screen. It measures the rows
// without drawing them, and stops at the widest the box takes.
func branchesWidth(v BranchesView, rows []BranchItem, nerd bool, screen int) int {
	need := branchesStateLine(v).width()
	for _, it := range rows {
		if need >= branchesMaxW-2 {
			break
		}
		need = max(need, branchItemWidth(it, nerd))
	}
	return max(min(need+2, branchesMaxW, screen), min(branchesMinW, screen))
}

// branchItemWidth is the cells a row asks for, uncut.
func branchItemWidth(it BranchItem, nerd bool) int {
	switch it.Kind {
	case 'g':
		return 3 + textWidth(branchGroupNames[it.Group])
	case 'm':
		return 5 + textWidth(it.Text)
	}
	n := 5 + textWidth(it.Row.Name) + 2 + branchAsideWidth(it.Row)
	if nerd {
		n += 2
	}
	return n
}

// branchAsideWidth is branchAside(r)'s width, counted without building it.
func branchAsideWidth(r BranchRow) int {
	switch {
	case r.Elsewhere != "":
		return len("in another worktree ")
	case r.Upstream == "":
		return 0
	}
	n := textWidth(r.Upstream) + 1
	if r.Gone {
		n += len(" gone")
	}
	if r.Ahead > 0 {
		n += 2 + len(strconv.Itoa(r.Ahead))
	}
	if r.Behind > 0 {
		n += 2 + len(strconv.Itoa(r.Behind))
	}
	return n
}

// branchItemRow is a row of the list, inner cells wide: a group's heading,
// its chevron the Explorer's; a branch or tag, its mark, icon and name, and
// on a local branch's right its upstream and counts, or the other worktree
// that holds it; a message.
func branchItemRow(v BranchesView, it BranchItem, nerd bool, inner int) line {
	switch it.Kind {
	case 'g':
		open := v.Query != "" || !v.Folded[it.Group]
		chev := expClosed
		if open {
			chev = expOpen
		}
		if nerd {
			chev = nfClosed
			if open {
				chev = nfOpen
			}
		}
		return padRow(line{deco(" ", stNone), deco(chev, stGrey), deco(branchGroupNames[it.Group], stBright)}, inner)
	case 'm':
		return padRow(line{deco("     ", stNone), deco(fitText(it.Text, inner-6), stGrey)}, inner)
	}
	r := it.Row
	mark, markSt := " ", stNone
	if r.Current {
		mark, markSt = "✓", stDone
	}
	left := line{deco("   ", stNone), deco(mark, markSt), deco(" ", stNone)}
	if nerd {
		icon := nfBranch
		if it.Kind == 't' {
			icon = nfTag
		}
		left = append(left, deco(icon.glyph, icon.st), deco(" ", stNone))
	}
	aside := branchAside(r)
	room := inner - left.width() - 1
	if aside.width() > 0 && room-aside.width()-2 >= min(textWidth(r.Name), 12) {
		room -= aside.width() + 2
	} else {
		aside = nil
	}
	name := fitText(r.Name, max(room, 1))
	if i := strings.Index(name, "/"); i > 0 && i < len(name)-1 {
		left = append(left, deco(name[:i+1], stGrey), deco(name[i+1:], stBright))
	} else {
		left = append(left, deco(name, stBright))
	}
	if r.Elsewhere != "" {
		for j := range left {
			if left[j].st == stBright {
				left[j].st = stGrey // not offered (design D5)
			}
		}
	}
	return spread(padRow(left, inner-aside.width()), aside, inner)
}

// branchAside is a branch's right part: the other worktree that holds it,
// else its upstream, "gone", ↑ahead and ↓behind; nothing for the others.
func branchAside(r BranchRow) line {
	switch {
	case r.Elsewhere != "":
		return line{deco("in another worktree ", stGrey)}
	case r.Upstream == "":
		return nil
	}
	l := line{deco(r.Upstream, stGrey)}
	if r.Gone {
		l = append(l, deco(" gone", stFailed))
	}
	if r.Ahead > 0 {
		l = append(l, deco(" ↑"+strconv.Itoa(r.Ahead), stDone))
	}
	if r.Behind > 0 {
		l = append(l, deco(" ↓"+strconv.Itoa(r.Behind), stAttention))
	}
	return append(l, deco(" ", stNone))
}

// branchesSearch is the search row: "›", the query and its caret, or the
// placeholder; "fetch" on its right, "fetching…" while it runs.
func branchesSearch(v BranchesView, nerd bool, inner int) line {
	left := line{deco(" › ", stModelFrame)}
	if v.Query == "" {
		left = append(left, deco("▌", stModelFrame), deco("Search branches and tags", stGrey))
	} else {
		left = append(left, deco(v.Query, stBright), deco("▌", stModelFrame))
	}
	var fetch line
	switch {
	case strings.HasPrefix(v.Busy, "fetching"):
		fetch = line{deco("fetching… ", stProgress)}
	case nerd:
		fetch = line{deco(nfRefresh.glyph, nfRefresh.st), deco(" fetch ", stGrey)}
	default:
		fetch = line{deco("F5 fetch ", stGrey)}
	}
	return spread(left, branchItems(fetch, ItemBranchFetch), inner)
}

// branchesState is the current state's row, inner cells wide.
func branchesState(v BranchesView, inner int) line {
	return padRow(branchesStateLine(v).fit(inner), inner)
}

// branchesStateLine is the current state: the branch, an arrow and its
// upstream with "gone", ↑ahead and ↓behind; "detached at <id>"; "<branch>,
// no commits yet".
func branchesStateLine(v BranchesView) line {
	l := line{deco(" ", stNone)}
	switch {
	case v.Detached:
		l = append(l, deco("detached at ", stAttention), deco(v.ID, stBright))
	case v.Head == "":
		return l
	case v.NoCommits:
		l = append(l, deco(v.Head, stBright), deco(", no commits yet", stGrey))
	default:
		l = append(l, deco(v.Head, stBright))
		if v.Upstream != "" {
			l = append(l, deco(" → ", stGrey), deco(v.Upstream, stGrey))
			if v.Gone {
				l = append(l, deco(" gone", stFailed))
			}
			if v.Ahead > 0 {
				l = append(l, deco(" ↑"+strconv.Itoa(v.Ahead), stDone))
			}
			if v.Behind > 0 {
				l = append(l, deco(" ↓"+strconv.Itoa(v.Behind), stAttention))
			}
		}
	}
	return l
}

// branchesFootRows is the most rows a text of the footer takes, its last
// cut: git's longest refusals list files, and the list keeps its rows.
const branchesFootRows = 3

// branchesFoot is the box's last rows: the switch's confirmation (design
// D5), else what runs, else the last result, else the keys, cut to fit.
func branchesFoot(v BranchesView, inner int) []line {
	text := func(s string, st *lipgloss.Style) []line {
		rows := wrap(s, inner-2)
		if len(rows) > branchesFootRows {
			rows = rows[:branchesFootRows]
			last := rows[len(rows)-1]
			if textWidth(last) > inner-3 {
				last = cut(last, inner-3)
			}
			rows[len(rows)-1] = last + "…"
		}
		var out []line
		for _, w := range rows {
			out = append(out, padRow(line{deco(" ", stNone), deco(w, st)}, inner))
		}
		return out
	}
	if v.Confirm != "" {
		s := strconv.Itoa(v.Changes) + " uncommitted changes. Git keeps them on " + v.Confirm +
			" unless they touch files that differ there; then it refuses and nothing changes."
		if v.Changes == 1 {
			s = "1 uncommitted change. Git keeps it on " + v.Confirm +
				" unless it touches a file that differs there; then Git refuses and nothing changes."
		}
		out := text(s, stAttention)
		return append(out, padRow(line{deco(" Enter switch · Esc cancel", stBright)}, inner))
	}
	switch {
	case v.Busy != "":
		return []line{padRow(line{deco(" "+fitText(v.Busy, inner-2), stProgress)}, inner)}
	case v.Note != "":
		st := stDone
		if v.NoteBad {
			st = stFailed
		}
		return text(v.Note, st)
	}
	keys := branchesKeys[len(branchesKeys)-1]
	for _, k := range branchesKeys {
		if textWidth(k)+2 <= inner {
			keys = k
			break
		}
	}
	return []line{padRow(line{deco(" "+keys, stGrey)}, inner)}
}
