package render

import (
	"slices"
	"strings"
	"sync/atomic"
	"unicode"
	"unicode/utf8"

	"github.com/Evengelius/agent-command-center/internal/state"
)

// The session history view (plan 75, brief amendment 89): the sessions the
// dashboard ran in the project the home screen's field names, in a box in the
// home panels' style, full screen like the Worktrees view so that nothing is
// drawn over the Sixel picture (design D2).

// HistoryPanel is the history view's panel for the mouse; its items say what
// a click there does.
const HistoryPanel = 'Z'

// The history view's items: the delete question's buttons, and an entry's
// row plus its index.
const (
	HistoryItemDelete = 1
	HistoryItemCancel = 2
	HistoryItemRow    = 1 << 10
)

// Asks of the history view: 0 none, the delete question of one entry, of all.
const (
	HistoryAskOne = 'o'
	HistoryAskAll = 'a'
)

// TitleChars is the most grapheme clusters a session's title keeps (plan 75,
// design D1a).
const TitleChars = 60

// HistoryView is the history view (plan 75, design D2).
type HistoryView struct {
	Open    bool
	Project string // the project's path, as Start resolves the field
	Rows    []HistoryRow
	Sel     int
	// Top is the first entry shown: the list scrolls only when the selection
	// would leave the rows shown, so a click never moves it (HistoryTop).
	Top int
	// Ask is the delete question in the box, HistoryAskOne or HistoryAskAll,
	// 0 for none; AskSel its selected answer, 0 Cancel, 1 Delete (design D4);
	// AskCount the entries it counted and AskCLIs the CLIs' names of them,
	// which the list may no longer show all of.
	Ask      byte
	AskSel   int
	AskCount int
	AskCLIs  []string
	// Note is why Enter did nothing ("<CLI> is not installed") or why the
	// list is not trusted, "" for none.
	Note string
}

// HistoryRow is one entry of the history: its title ("" for none), the grey
// line under it (its date and time, age, CLI, model and ID), its CLI's name
// and whether that CLI is not installed, which dims it.
type HistoryRow struct {
	Title, Meta, CLI string
	Dim              bool
}

// historyAsk is the delete question's title and what it says of the CLIs'
// own records (design D4): ACC's list only.
func historyAsk(v HistoryView) (title, note string) {
	title = "DELETE THIS SESSION FROM THE HISTORY?"
	if v.Ask == HistoryAskAll {
		title = "DELETE ALL " + plural(v.AskCount, "SESSION", "SESSIONS") + " OF " + base(hideUser(v.Project)) + "?"
	}
	var names []string
	for _, c := range v.AskCLIs {
		if !slices.Contains(names, c) {
			names = append(names, c)
		}
	}
	name := ""
	switch len(names) {
	case 0:
	case 1:
		name = names[0] + " keeps its own record"
	default:
		name = strings.Join(names[:len(names)-1], ", ") + " and " + names[len(names)-1] + " keep their own records"
	}
	return title, "ACC's list only: " + name
}

// SessionTitle is the title of a session whose first message began with text
// (plan 75, design D1a). The text is first cleaned into one string: control
// characters (Cc) and Unicode format characters (Cf) dropped, so no escape
// sequence reaches the terminal, but the joiners U+200C and U+200D and the tag
// characters U+E0020 to U+E007F, which join clusters; every run of white space
// one space, none at the start. Its first TitleChars grapheme clusters are
// then kept, none cut, and the end trimmed. text may be only the start of the
// message (partial): a last cluster that reaches the end of the cleaned text
// may lack the characters that follow, and is left out. The whole of text is
// read once, and the title is a copy of its own: the cost is the text's, in
// two allocations whatever its length. "" when nothing is left.
func SessionTitle(text string, partial bool) string {
	var b strings.Builder
	b.Grow(len(text))
	space := false
	for _, r := range text {
		switch {
		case unicode.IsSpace(r):
			space = true
			continue
		case unicode.IsControl(r), unicode.Is(unicode.Cf, r) && r != 0x200C && r != 0x200D && (r < 0xE0020 || r > 0xE007F):
			continue
		}
		if space && b.Len() > 0 {
			b.WriteByte(' ')
		}
		space = false
		b.WriteRune(r)
	}
	s := b.String()
	end := 0
	for n := 0; n < TitleChars && end < len(s); n++ {
		e := clusterEndAt(s, end)
		if partial && e == len(s) {
			break
		}
		end = e
	}
	return strings.Clone(strings.TrimSpace(s[:end]))
}

// titleScanned counts the bytes clusterEndAt has stepped over, for the
// tests: SessionTitle's scan of its clusters reads each byte once, its cost
// linear in its input whatever the input's size (TestSessionTitleCost).
var titleScanned atomic.Int64

// clusterEndAt is the end of the grapheme cluster that starts at byte i of s,
// which holds no control character, without measuring it.
func clusterEndAt(s string, i int) int {
	if s[i] < 0x80 && (i+1 == len(s) || s[i+1] < 0x80) {
		titleScanned.Add(1)
		return i + 1
	}
	r, n := utf8.DecodeRuneInString(s[i:])
	end, _ := clusterEnd(s, i, r, n)
	titleScanned.Add(int64(end - i))
	return end
}

// ModelOf is the model as zone A names it for the session, "" before the
// session or its telemetry names one (plan 75, design D1).
func ModelOf(s *state.State) string {
	m := model(s)
	if m == "" {
		return ""
	}
	return modelTitle(m)
}

// elideEnd is s cut at its end with "…" to at most w cells, never inside a
// grapheme cluster.
func elideEnd(s string, w int) string {
	if textWidth(s) <= w {
		return s
	}
	if w <= 1 {
		return "…"
	}
	return cut(s, w-1) + "…"
}

// historyBox is the box's size on a w by h screen with n rows of entries in
// all: the width, and the height, which holds the entries up to what the
// screen has and leaves room for the delete question.
func historyBox(w, h int, v HistoryView, entryRows int) (bw, bh int) {
	bw = min(w-2, 100)
	const fixed = 3 + 3 + 2 // frame and title, the head's three rows, the tail's two
	if len(v.Rows) == 0 {
		return bw, min(fixed+3, h-2)
	}
	return bw, min(fixed+max(entryRows, 8)+3, h-2)
}

// historyGeom is the box on a w by h screen for v: the rows all the entries
// take (a title row when an entry has a title, and its grey line), the box's
// size, and the rows the entries are shown in.
func historyGeom(w, h int, v HistoryView) (total, bw, bh, room int) {
	for _, r := range v.Rows {
		total++
		if r.Title != "" {
			total++
		}
	}
	bw, bh = historyBox(w, h, v, total)
	tail := 2
	if v.Ask != 0 {
		tail = 5
	}
	return total, bw, bh, max(bh-3-3-tail, 0)
}

// historyTop is the first entry shown for v when the entries have room rows:
// v.Top, moved only as far as keeps the selection in view, and, when the end
// of the list is in view with rows to spare, up to fill them.
func historyTop(v HistoryView, room int) int {
	n := len(v.Rows)
	if n == 0 {
		return 0
	}
	rows := func(i int) int {
		if v.Rows[i].Title != "" {
			return 2
		}
		return 1
	}
	sel := min(max(v.Sel, 0), n-1)
	top := min(max(v.Top, 0), n-1)
	top = min(top, sel)
	used := 0 // the rows from top to the selection
	for k := top; k <= sel; k++ {
		used += rows(k)
	}
	for used > room && top < sel {
		used -= rows(top)
		top++
	}
	tail := 0 // the rows from top to the end
	for k := top; k < n; k++ {
		tail += rows(k)
	}
	for tail <= room && top > 0 && tail+rows(top-1) <= room {
		top--
		tail += rows(top)
	}
	return top
}

// HistoryTop is the first entry the view shows on a w by h screen: what the
// application keeps as the list's scroll position, so that it moves only when
// the selection would leave the rows shown (design D2).
func HistoryTop(w, h int, v HistoryView) int {
	_, _, _, room := historyGeom(w, h, v)
	return historyTop(v, room)
}

// historyEntryRows are an entry's rows: its title row when it has one, then
// its grey line, selected or not, within iw cells.
func historyEntryRows(r HistoryRow, i int, sel bool, iw int) []line {
	gr, nc, mc := pageGround, cHomeName, cHomeRowPath
	left, right := " ", " "
	if sel {
		gr, nc, mc = rowFocusGround, cHomeNameSel, cHomeRowPathSel
		left, right = "▏", "▕"
	}
	if r.Dim {
		nc, mc = cHomeRowPath, cHomeRowPath
	}
	row := func(text string, c colour, bold bool, indent string) line {
		l := line{sp(" ", pageGround.on(stNone)), homeOn(left, cHomeFocus, false, gr), homeOn(indent+elideEnd(text, iw-3-textWidth(indent)), c, bold, gr)}
		l = append(l, homeOn(strings.Repeat(" ", max(iw-2-l.width()+1, 0)), c, false, gr), homeOn(right, cHomeFocus, false, gr))
		for k := range l {
			l[k].ref = ref{panel: HistoryPanel, kind: refDeco, item: HistoryItemRow + i}
		}
		return l
	}
	var out []line
	if r.Title != "" {
		out = append(out, row(r.Title, nc, sel, " "))
	}
	return append(out, row(r.Meta, mc, false, " "))
}

// historyScreen draws the view on a w by h screen: the home screen's header,
// then the box, centred, and records the box for a click outside it.
func historyScreen(s *state.State, ui UI, sc *Screen) []line {
	w, h, v := ui.Width, ui.Height, ui.History
	iw := min(w-2, 100) - 2
	total, bw, bh, room := historyGeom(w, h, v)
	top := 1 + max((h-1-bh)/2, 0)
	left := (w - bw) / 2
	ew := iw // the entries' width, a column less for the scroll mark
	if total > room {
		ew = iw - 1
	}
	var entries [][]line
	for i, r := range v.Rows {
		entries = append(entries, historyEntryRows(r, i, i == v.Sel, ew))
	}
	grey := func(text string, c colour) line {
		return line{sp("   ", pageGround.on(stNone)), homeOn(elideEnd(text, iw-5), c, false, pageGround)}
	}
	blank := line{sp(" ", pageGround.on(stNone))}
	var inner []line
	inner = append(inner, blank, append(line{sp("   ", pageGround.on(stNone)), homeOn("Project  ", cHomeSection, false, pageGround)},
		homeOn(elide(v.Project, iw-5-9), cHomePath, false, pageGround)), blank)

	first := historyTop(v, room)
	var rows []line
	before := 0
	for k := 0; k < first; k++ {
		before += len(entries[k])
	}
	for k := first; k < len(entries) && len(rows)+len(entries[k]) <= room; k++ {
		rows = append(rows, entries[k]...)
	}
	if total > len(rows) && len(rows) > 0 {
		// The slim mark: its length and place those of the rows shown.
		size := max(len(rows)*len(rows)/total, 1)
		at := before * len(rows) / total
		for i := range rows {
			mark := " "
			if i >= at && i < at+size {
				mark = "▐"
			}
			rows[i] = append(homeRow(rows[i], iw-1), homeOn(mark, cHomeFrame, false, pageGround))
		}
	}
	if len(v.Rows) == 0 {
		rows = []line{blank, grey("No sessions recorded for this project yet", cHomeRowPath), blank}
	}
	for len(rows) < room && len(v.Rows) > 0 {
		rows = append(rows, blank)
	}
	inner = append(inner, rows...)

	// The tail: a note and the keys, or the delete question over the keys.
	hint := grey("↑↓ move · Enter resume · Del delete · Shift+Del delete all · Esc close", cHomeAction)
	if v.Ask != 0 {
		hint = grey("←→ choose · Enter answer · Esc cancel", cHomeAction)
	}
	if v.Ask != 0 {
		cell := func(label string, on bool, item int) []span {
			gr, c := capGround, cHomeCapText
			if on {
				gr, c = buttonGround, cHomeButtonText
			}
			out := []span{homeOn("  "+label+"  ", c, on, gr)}
			out[0].ref = ref{panel: HistoryPanel, kind: refDeco, item: item}
			return out
		}
		buttons := line{sp("   ", pageGround.on(stNone))}
		buttons = append(buttons, cell("Delete", v.AskSel == 1, HistoryItemDelete)...)
		buttons = append(buttons, sp("   ", pageGround.on(stNone)))
		buttons = append(buttons, cell("Cancel", v.AskSel == 0, HistoryItemCancel)...)
		title, note := historyAsk(v)
		q := line{sp("   ", pageGround.on(stNone)), sp(elideEnd(title, iw-5), pageGround.on(stProgressBold))}
		inner = append(inner, blank, q, grey(note, cHomeRowPath), buttons, hint)
	} else {
		note := blank
		if v.Note != "" {
			note = line{sp("   ", pageGround.on(stNone)), sp(elideEnd("✗ "+hideUser(v.Note), iw-5), pageGround.on(stFailed))}
		}
		inner = append(inner, note, hint)
	}

	box := homePanel("SESSION HISTORY", inner[:min(len(inner), bh-3)], bw)
	sc.History = Box{X: left, Y: top, W: bw, H: len(box)}
	ls := []line{homeGridHeader(s, ui, w)}
	for len(ls) < top {
		ls = append(ls, homeBlank(w))
	}
	pad := func(n int) span { return sp(strings.Repeat(" ", max(n, 0)), pageGround.on(stNone)) }
	for _, b := range box {
		ls = append(ls, append(append(line{pad(left)}, b...), pad(w-left-bw)))
	}
	for len(ls) < h {
		ls = append(ls, homeBlank(w))
	}
	return ls
}
