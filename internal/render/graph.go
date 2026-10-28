package render

import (
	"encoding/hex"
	"slices"
	"strconv"
	"strings"
	"time"

	"charm.land/lipgloss/v2"

	"github.com/Evengelius/agent-command-center/internal/state"
)

// The Git Graph (plan 62, brief amendment 64; ui-design.md, Git Graph): a
// panel of the docked place, shown by the sidebar's third tile or Alt+G,
// the history of the scope's references newest first, a commit unfolded
// into its files, its scope chosen in a popup (graphscope.go). Its cells
// are panel Y: a row's carry its index in GraphView.Rows plus one, the
// scope row's ItemGraphScope, the header's "⌄" ItemPanelHide, the rest 0.

const (
	GraphPanel     = 'Y'
	ItemGraphScope = -2
)

// GraphView is the Git Graph as the application holds it (designs D2 to
// D8): the history read so far, the rows the panel shows of it, the cursor.
type GraphView struct {
	Open bool
	// Scope names the scope (design D2): "Auto", "All", a reference's short
	// name, "3 references"; Follows says what Auto follows ("main →
	// origin/main", "main, no upstream", "detached at 1a2b3c4"); ScopeNote a
	// note beside it ("1 reference gone").
	Scope, Follows, ScopeNote string
	// State is said in the history's place: "reading…", "no commits yet",
	// git's failure (Failed).
	State  string
	Failed bool
	// Commits are the history read, in git's topological order; Rows the
	// panel's rows: each a commit, or one of an unfolded commit's files.
	Commits []GraphEntry
	Rows    []GraphRowRef
	// Files are the unfolded commits' files and Labels the references on a
	// commit, both by the commit's index in Commits.
	Files  map[int32]*GraphFiles
	Labels map[int32][]GraphLabel
	// Cursor is the cursor's row in Rows, Top the first row shown.
	Cursor, Top int
	// More is said under the last commit: "reading…" while a page comes,
	// the guard's "first 1,000,000 commits".
	More string
	// Lanes are the history's lanes, laid out by the application as the
	// commits come (graphlanes.go, design D9).
	Lanes *GraphLanes
	// Wide is the wide view (design D10): zone A's columns too while the
	// panel has the focus, more lanes, the author and the date in columns of
	// their own, the dates in Loc's zone, nil the machine's.
	Wide bool
	Loc  *time.Location
	// Hover is the row under the pointer, plus one, 0 off the rows; Card the
	// card shown beside its row (design D13), nil none.
	Hover int
	Card  *GraphCard
	// hidden is the frame's own: lanes hidden in view, for the footer.
	hidden bool
}

// GraphEntry is a commit as the history keeps it: compact, about a hundred
// bytes (design D4).
type GraphEntry struct {
	// Hash is the commit's; Parent its first parent's, zero for a root;
	// Parents how many it has.
	Hash, Parent [20]byte
	Parents      uint8
	// Author is the author's name, one string shared by a name's commits;
	// Time the author's time, Unix seconds.
	Author  string
	Time    int64
	Subject string
}

// Short is the commit's short hash, seven digits.
func (e GraphEntry) Short() string { return hex.EncodeToString(e.Hash[:])[:7] }

// HashText is the commit's full hash.
func (e GraphEntry) HashText() string { return hex.EncodeToString(e.Hash[:]) }

// ParentText is the first parent's full hash, "" for a root.
func (e GraphEntry) ParentText() string {
	if e.Parents == 0 {
		return ""
	}
	return hex.EncodeToString(e.Parent[:])
}

// GraphRowRef is one of the panel's rows: commit Commit's own row, File -1,
// or its file File.
type GraphRowRef struct {
	Commit, File int32
}

// GraphFiles is an unfolded commit's files (design D5).
type GraphFiles struct {
	// State is said under the commit while git reads ("reading…") or when it
	// failed (Failed).
	State  string
	Failed bool
	Files  []GraphFile
	// Total is how many the commit changed: more than listed past the guard.
	Total int
}

// GraphFile is a file a commit changed.
type GraphFile struct {
	// Status is git's letter, A M D R or T; Path the file's path from the
	// repository's top level; Orig a rename's source.
	Status     byte
	Path, Orig string
	// Added and Removed count its lines; Binary says git counts none.
	Added, Removed int
	Binary         bool
}

// GraphLabel is a reference on a commit, as its row shows it (design D11).
type GraphLabel struct {
	// Name is short: "main", "origin/main", "v1.0".
	Name string
	// Kind is 'h' a local branch, 'r' a remote branch, 't' a tag; Head says
	// HEAD is on it.
	Kind byte
	Head bool
}

// graphFixed are the panel's rows outside its history: the frame's two
// edges, the header, the rule and the scope row.
const graphFixed = 5

// GraphRoom is how many rows of the history the panel shows on this screen,
// what PgUp and PgDn move by.
func GraphRoom(s *state.State, ui UI) int {
	return max(layout(s, ui).upper-graphFixed, 0)
}

// graphPanel is the Git Graph's panel w by h (design D2): the header "GIT
// GRAPH" with its "⌄", a rule, the scope row, then the history or the
// state in its place, the cursor kept in view; its frame in the title
// colour while it has the focus. It notes on sc lanes hidden in view, for
// the footer (design D10).
func graphPanel(ui UI, w, h int, sc *Screen) []line {
	v := ui.Graph
	inner := w - 4
	body := []line{
		spread(decoLine("GIT GRAPH", stTitle), line{graphDeco("⌄", stTitle, ItemPanelHide)}, inner),
		teleRule(inner),
		scopeRow(v, inner),
	}
	room := max(h-graphFixed, 0)
	switch {
	case v.State != "":
		body = append(body, wrapDeco(v.State, inner, "", either(v.Failed, stAttention, stGrey))...)
	default:
		top := ListTop(v.Top, max(v.Cursor, 0), room, len(v.Rows))
		end := min(top+room, len(v.Rows))
		lv := viewLanes(v, top, end, inner)
		sc.GraphHidden = lv.hidden
		for i := top; i < end; i++ {
			r := graphRow(v, v.Rows[i], i == v.Cursor, inner, lv, i-top)
			for j := range r {
				r[j].ref = ref{panel: GraphPanel, kind: refDeco, item: i + 1}
			}
			body = append(body, r)
		}
		if v.More != "" && top+room > len(v.Rows) {
			l := laneSpans(lv.end, lv.endOver, lv.hidden, plainOn)
			body = append(body, append(l, deco("  "+v.More, stGrey)).fit(inner))
		}
	}
	return box(GraphPanel, nil, nil, body, w, h, borderFor(ui.Focus == "R"))
}

// laneView is the lanes of the rows in view (designs D9 and D10): each
// row's cells, two a lane, and its overflow column's glyph; the lanes going
// on under the last commit in view; whether lanes are hidden.
type laneView struct {
	cells   [][]laneCell
	over    []rune
	end     []laneCell
	endOver rune
	hidden  bool
}

// viewLanes lays out the lanes of rows top to end-1, inner cells wide, from
// the lanes kept whole before them: as many lanes as leave the subject 12
// cells, two at least, the others hidden behind one overflow column (design
// D10). No lanes, before the commits in view are laid out.
func viewLanes(v GraphView, top, end, inner int) laneView {
	var lv laneView
	if v.Lanes == nil || top >= end || int(v.Rows[end-1].Commit) >= v.Lanes.Len() {
		return lv
	}
	c0, c1 := int(v.Rows[top].Commit), int(v.Rows[end-1].Commit)
	need := 0
	v.Lanes.walk(c0, c1+1, func(_ int, r *laneRow) { need = max(need, r.width()) })
	show := need
	if s := max(2, (inner-14)/2); 2*need+12 > inner && s < need {
		show, lv.hidden = s, true
	}
	lv.cells, lv.over = make([][]laneCell, end-top), make([]rune, end-top)
	j := top
	v.Lanes.walk(c0, c1+1, func(i int, r *laneRow) {
		for ; j < end && int(v.Rows[j].Commit) == i; j++ {
			cells := make([]laneCell, 2*show)
			if v.Rows[j].File < 0 {
				lv.over[j-top] = r.figure(cells, show)
			} else {
				lv.over[j-top] = r.below(cells, show)
			}
			lv.cells[j-top] = cells
		}
		if i == c1 {
			lv.end = make([]laneCell, 2*show)
			lv.endOver = r.below(lv.end, show)
		}
	})
	return lv
}

// plainOn is a row's style function off the band.
func plainOn(st *lipgloss.Style) *lipgloss.Style { return st }

// laneSpans is a row's lanes as spans through on: each glyph in its lane's
// colour; after them, where lanes are hidden, the overflow column's "┆" or
// "◆" in grey.
func laneSpans(cells []laneCell, over rune, hidden bool, on func(*lipgloss.Style) *lipgloss.Style) line {
	var l line
	add := func(text string, st *lipgloss.Style) {
		if n := len(l); n > 0 && l[n-1].st == st {
			l[n-1].text += text
			return
		}
		l = append(l, deco(text, st))
	}
	for _, c := range cells {
		if c.glyph == 0 {
			add(" ", on(stNone))
		} else {
			add(string(c.glyph), on(stLane[c.colour]))
		}
	}
	if hidden {
		add(string(max(over, ' ')), on(stGrey))
		add(" ", on(stNone))
	}
	return l
}

// scopeRow is the scope row (design D2): "◎ Auto ⌄" and what Auto follows,
// a note in the attention colour; a click on it opens the scope popup.
func scopeRow(v GraphView, w int) line {
	scope := v.Scope
	if scope == "" {
		scope = "Auto"
	}
	l := line{graphDeco("◎ ", stTitle, ItemGraphScope), graphDeco(scope, stTitleBold, ItemGraphScope), graphDeco(" ⌄", stTitle, ItemGraphScope)}
	if v.Follows != "" {
		l = append(l, graphDeco(" · "+v.Follows, stMuted, ItemGraphScope))
	}
	if v.ScopeNote != "" {
		l = append(l, graphDeco(" · "+v.ScopeNote, stAttention, ItemGraphScope))
	}
	l = l.fit(w)
	for i := range l {
		l[i].ref = ref{panel: GraphPanel, kind: refDeco, item: ItemGraphScope}
	}
	return l
}

// The wide view's columns (design D10): the author's and the date's.
const (
	graphAuthorW = 18
	graphDateW   = 16
)

// graphRow is row r, w cells, view row k of lv: a commit's lanes with its
// dot, its labels, its subject cut only where the row ends and its author
// muted where room remains (design D11), or in the wide view the author
// and the date in columns of their own (design D10); a file under the
// lanes going on, its status letter in Source Control's colours, its name,
// its folder muted and its counts; the state of a commit's files under it,
// or that it changed none. The cursor's row on the band. Without lanes, a
// commit's dot alone, a merge's ring.
func graphRow(v GraphView, r GraphRowRef, cur bool, w int, lv laneView, k int) line {
	var g *ground
	if cur {
		g = bandGround
	}
	on := func(st *lipgloss.Style) *lipgloss.Style {
		if g == nil {
			return st
		}
		return g.on(st)
	}
	var lead line
	if k < len(lv.cells) && lv.cells[k] != nil {
		lead = laneSpans(lv.cells[k], lv.over[k], lv.hidden, on)
	}
	var l line
	if r.File < 0 {
		e := v.Commits[r.Commit]
		if lead == nil {
			dot := "● "
			if e.Parents > 1 {
				dot = "◉ "
			}
			lead = line{deco(dot, on(stTitle))}
		}
		room := w
		var cols line
		if v.Wide && w-lead.width()-4-graphAuthorW-graphDateW >= 24 {
			loc := v.Loc
			if loc == nil {
				loc = time.Local
			}
			cols = line{deco("  "+padRight(fitText(e.Author, graphAuthorW), graphAuthorW), on(stMuted)),
				deco("  "+time.Unix(e.Time, 0).In(loc).Format("02/01/2006 15:04"), on(stGrey))}
			room -= cols.width()
		}
		// The labels give way, the last first, before the subject is cut
		// (design D11).
		labels, subject := v.Labels[r.Commit], e.Subject
		for n := len(labels); n >= 0; n-- {
			l = slices.Clone(lead)
			for _, lb := range labels[:n] {
				l = append(l, deco(labelText(lb), on(labelStyle(lb))), deco(" ", on(stNone)))
			}
			if textWidth(subject) <= room-l.width() {
				break
			}
		}
		l = append(l, deco(fitText(subject, max(room-l.width(), 1)), on(stText)))
		switch {
		case cols != nil:
			l = append(l.fit(room), cols...)
		case e.Author != "" && l.width()+2+textWidth(e.Author) <= w:
			l = append(l, deco("  "+e.Author, on(stMuted)))
		}
	} else {
		if lead == nil {
			lead = line{deco("  ", on(stNone))}
		}
		lead = append(lead, deco("  ", on(stNone)))
		fs := v.Files[r.Commit]
		switch {
		case fs == nil:
			l = lead
		case r.File == 0 && fs.State != "":
			l = append(lead, deco(fs.State, on(either(fs.Failed, stAttention, stGrey))))
		case fs.Total == 0:
			note := "no file changed"
			if v.Commits[r.Commit].Parents > 1 {
				note = "no change against its first parent"
			}
			l = append(lead, deco(note, on(stGrey)))
		case int(r.File) >= len(fs.Files):
			l = append(lead, deco("first "+grouped(len(fs.Files))+" of "+grouped(fs.Total)+" files", on(stGrey)))
		default:
			l = graphFileRow(fs.Files[r.File], lead, on, w)
		}
	}
	l = l.fit(w)
	for i := range l {
		if g != nil && l[i].st == stNone {
			l[i].st = on(stNone) // fit's padding on the row's ground too
		}
	}
	return l
}

// padRight is s with spaces after it to n cells.
func padRight(s string, n int) string {
	return s + strings.Repeat(" ", max(n-textWidth(s), 0))
}

// graphFileRow is a commit's file, w cells, after lead: its letter, its
// name and its folder muted, a rename's source after "←", its counts at the
// right.
func graphFileRow(f GraphFile, lead line, on func(*lipgloss.Style) *lipgloss.Style, w int) line {
	l := append(lead, deco(string(f.Status), on(letterStyle(f.Status))), deco("  ", on(stNone)))
	var counts line
	switch {
	case f.Binary:
		counts = line{deco("binary", on(stGrey))}
	case f.Added > 0 || f.Removed > 0:
		counts = line{deco("+"+strconv.Itoa(f.Added), on(stDiffAdd)), deco(" −"+strconv.Itoa(f.Removed), on(stDiffDel))}
	}
	dir, name := splitPath(f.Path)
	room := w - l.width() - textWidth(name)
	if counts != nil && room-counts.width()-1 >= 0 {
		room -= counts.width() + 1
	} else {
		counts = nil
	}
	l = append(l, deco(name, on(stExpFileName)))
	if f.Orig != "" {
		if src := " ← " + f.Orig; textWidth(src) <= room {
			l = append(l, deco(src, on(stGrey)))
			room -= textWidth(src)
		}
	}
	if dir != "" && room > 1 {
		l = append(l, deco(" "+fitText(dir, room-1), on(stMuted)))
	}
	if counts != nil {
		return spreadOn(l, counts, w, on(stNone))
	}
	return l
}

// labelText is a label as its pill reads: HEAD's branch marked "◎".
func labelText(lb GraphLabel) string {
	if lb.Head {
		return "◎ " + lb.Name
	}
	return lb.Name
}

// labelStyle is a label's colour: a local branch blue, a remote branch the
// title colour, a tag amber.
func labelStyle(lb GraphLabel) *lipgloss.Style {
	switch lb.Kind {
	case 'r':
		return stTitle
	case 't':
		return stProgress
	}
	return stInfo
}

// graphDeco is a span of the panel: decoration, panel Y, its item.
func graphDeco(text string, st *lipgloss.Style, item int) span {
	return span{text: text, st: st, ref: ref{panel: GraphPanel, kind: refDeco, item: item}}
}

// graphHints are the panel's key hints while it has the focus, the longest
// first (design D8).
var graphHints = []string{
	"↑↓ PgUp PgDn Home End move · → open · ← fold · Enter diff · s scope · f wide · Ctrl+C copy hash · Esc back to the input · F6 zone · F1 keys",
	"↑↓ move · → open · ← fold · Enter diff · s scope · f wide · Ctrl+C copy · Esc back to the input · F1 keys",
	"↑↓ move · ←→ fold · Enter diff · s scope · f wide · Esc back · F1 keys",
	"↑↓ · Enter diff · s scope · Esc back",
}

// graphKeys is the footer's text while the panel has the focus, on a screen
// w columns wide: the hints, after the note that lanes are hidden in view
// when they are (design D10).
func graphKeys(w int, hidden, wide bool) string {
	if !hidden {
		return fitHint(graphHints, w)
	}
	note := "lanes hidden: f widens the graph · "
	if wide {
		note = "lanes hidden · "
	}
	hints := make([]string, len(graphHints))
	for i, h := range graphHints {
		hints[i] = note + h
	}
	return fitHint(hints, w)
}

// GraphCommitAt is the commit of row i, -1 off the rows.
func (v GraphView) GraphCommitAt(i int) int {
	if i < 0 || i >= len(v.Rows) {
		return -1
	}
	return int(v.Rows[i].Commit)
}
