package render

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/Evengelius/agent-command-center/internal/state"
)

// The diff view (plan 44, brief amendment 45, designs D8 and D9): a file
// opened from Source Control, its two sides compared, drawn in zone B's
// place; zone A, the docked panel, the sidebar and zone C stay as they
// were, and zone B's conversation, draft and scroll come back as they were
// when it closes. The application computes the diff in a command
// (internal/diff) and holds it here.

// DiffView is the diff as the application holds it.
type DiffView struct {
	Open bool
	// Path is the file, relative to the project; Orig a rename's source.
	Path, Orig string
	// Staged is the comparison: HEAD against the index, else the index
	// against the working tree; Both says the file has the other one too.
	Staged, Both bool
	// OldName and NewName are the sides compared: "HEAD", "index", "ours",
	// "working tree", "" for none (an added or a deleted file).
	OldName, NewName string
	// Conflict names an unmerged file's state, never said resolved.
	Conflict string
	// Note is said in the lines' place: "comparing…", or why there are no
	// lines (binary, an LFS pointer, line endings only, no textual change,
	// too large, git's error: Failed).
	Note   string
	Failed bool
	Hunks  []DiffHunk
	// Added and Removed count the lines; Whole says the bound applied (the
	// middle shown removed then added); NoNewline* that a side's last line
	// has no line end.
	Added, Removed             int
	Whole                      bool
	NoNewlineOld, NoNewlineNew bool
	// Layout is the rows, made once a diff (LayoutDiff).
	Layout *DiffLayout
	// Top is the first row shown, Left the columns the lines are shifted
	// by, Wide the view over zone A too ("f").
	Top, Left int
	Wide      bool
}

// DiffHunk is a hunk, as "@@ -OldStart,OldLines +NewStart,NewLines @@".
type DiffHunk struct {
	OldStart, OldLines, NewStart, NewLines int
	Lines                                  []DiffLine
}

// DiffLine is a line: Kind ' ' context, '-' removed, '+' added; its numbers
// on each side, 0 where it has none; its text without its line end.
type DiffLine struct {
	Kind     byte
	Old, New int
	Text     string
}

// DiffRow is a row of the view: a hunk's header (L and R -1) or its lines,
// L and R their indexes in the hunk; unified rows use L alone, side by
// side rows put a removed line (L) facing its replacement (R), -1 where a
// side has none.
type DiffRow struct {
	Hunk int
	L, R int32
}

// DiffLayout is a diff's rows, unified and side by side, the rows where a
// change starts in each (n and N), the line numbers' width and the longest
// line's cells, its tabs expanded.
type DiffLayout struct {
	Unified, Side           []DiffRow
	UniChanges, SideChanges []int
	NumWidth, TextWidth     int
}

// LayoutDiff lays the diff's rows out, once a diff: every row is an index,
// so a frame draws only the rows in view, whatever the diff's size.
func LayoutDiff(v DiffView) *DiffLayout {
	lay := &DiffLayout{NumWidth: 3}
	for h, hk := range v.Hunks {
		lay.NumWidth = max(lay.NumWidth, len(fmt.Sprint(hk.OldStart+hk.OldLines)), len(fmt.Sprint(hk.NewStart+hk.NewLines)))
		lay.Unified = append(lay.Unified, DiffRow{Hunk: h, L: -1, R: -1})
		lay.Side = append(lay.Side, DiffRow{Hunk: h, L: -1, R: -1})
		prev := byte(' ')
		for i, ln := range hk.Lines {
			if ln.Kind != ' ' && prev == ' ' {
				lay.UniChanges = append(lay.UniChanges, len(lay.Unified))
			}
			lay.Unified = append(lay.Unified, DiffRow{Hunk: h, L: int32(i), R: -1})
			lay.TextWidth = max(lay.TextWidth, textCells(ln.Text))
			prev = ln.Kind
		}
		// Side by side: a run of removed lines facing the run of added lines
		// after it, row by row.
		for i := 0; i < len(hk.Lines); {
			if hk.Lines[i].Kind == ' ' {
				lay.Side = append(lay.Side, DiffRow{Hunk: h, L: int32(i), R: int32(i)})
				i++
				continue
			}
			lay.SideChanges = append(lay.SideChanges, len(lay.Side))
			var del, add []int32
			for ; i < len(hk.Lines) && hk.Lines[i].Kind == '-'; i++ {
				del = append(del, int32(i))
			}
			for ; i < len(hk.Lines) && hk.Lines[i].Kind == '+'; i++ {
				add = append(add, int32(i))
			}
			for k := range max(len(del), len(add)) {
				r := DiffRow{Hunk: h, L: -1, R: -1}
				if k < len(del) {
					r.L = del[k]
				}
				if k < len(add) {
					r.R = add[k]
				}
				lay.Side = append(lay.Side, r)
			}
		}
	}
	return lay
}

// textCells is a line's cells as diffText draws them, its tabs expanded.
func textCells(s string) int {
	col := 0
	for i := 0; i < len(s); {
		n, w := cellSpan(s, i)
		if s[i] == '\t' {
			w = TabWidth - col%TabWidth
		}
		col, i = col+w, i+n
	}
	return col
}

// DiffMaxLeft is the most the lines shift at width w: the longest line's
// end at the right of its room.
func DiffMaxLeft(v DiffView, w int) int {
	if v.Layout == nil {
		return 0
	}
	room := w - v.Layout.NumWidth*2 - 2 - 2 // unified: the two numbers, their spaces, the marker
	if DiffSideBySide(w) {
		room = (w-1)/2 - v.Layout.NumWidth - 1 - 2
	}
	return max(v.Layout.TextWidth-max(room, 1)+1, 0)
}

// diffSideMin is the box's least width for side by side (design D8): under
// it, each side would cut most code lines.
const diffSideMin = 150

// DiffSideBySide says a diff box w wide draws its sides apart.
func DiffSideBySide(w int) bool { return w >= diffSideMin }

// DiffRows are the rows the view shows at width w, and the rows where a
// change starts.
func DiffRows(v DiffView, w int) ([]DiffRow, []int) {
	if v.Layout == nil {
		return nil, nil
	}
	if DiffSideBySide(w) {
		return v.Layout.Side, v.Layout.SideChanges
	}
	return v.Layout.Unified, v.Layout.UniChanges
}

// diffFixed are the box's rows outside its lines: its two edges, the path's
// row and the rule.
const diffFixed = 4

// diffHead are the rows over the lines: the sides' names when they are
// apart, the bound's note when it applied.
func diffHead(v DiffView, w int) int {
	n := 0
	if DiffSideBySide(w) && v.Note == "" {
		n++
	}
	if v.Whole && v.Note == "" {
		n++
	}
	if !DiffSideBySide(w) && noNewline(v) != "" && v.Note == "" {
		n++
	}
	return n
}

// noNewline is the unified view's row for a side whose last line, shown,
// has no line end, as git's "\ No newline at end of file"; side by side the
// sides' heads say it.
func noNewline(v DiffView) string {
	switch {
	case v.NoNewlineOld && v.NoNewlineNew:
		return "no newline at the end of either side"
	case v.NoNewlineOld:
		return "no newline at the end of the original"
	case v.NoNewlineNew:
		return "no newline at the end of the modified"
	}
	return ""
}

// diffMinW is the least width of a diff box with the panel docked beside it
// where the panel would take zone B's place: a unified row keeps 34
// columns of text (design D8; the owner: "The Source Control sidebar stays
// visible").
const diffMinW = 48

// DiffBox is the diff box's columns on this screen (design D8): zone B's,
// or zone A's and B's when wide. Where the panel would take zone B's place,
// the diff takes it with the panel docked beside it at its own width, or
// the whole place when the diff would keep fewer than diffMinW columns, as
// under 100 columns.
func DiffBox(s *state.State, ui UI) Cols {
	g := layout(s, ui)
	switch {
	case g.narrow:
		return Cols{0, ui.Width}
	case g.bw == 0: // the panel in zone B's place
		w := g.pw
		if pw := explorerWidth(ui.Width); w-pw >= diffMinW {
			w -= pw // the panel beside the diff
		}
		if ui.Diff.Wide {
			return Cols{0, g.aw + w}
		}
		return Cols{g.aw, w}
	case ui.Diff.Wide:
		return Cols{0, g.aw + g.bw}
	}
	return Cols{g.aw, g.bw}
}

// PanelUnderDiff says the docked panel is shown but the diff takes its
// place: under 100 columns, or where the diff would keep fewer than
// diffMinW columns beside it (design D8).
func PanelUnderDiff(s *state.State, ui UI) bool {
	if !ui.Diff.Open || !PanelShown(ui) {
		return false
	}
	g, d := layout(s, ui), DiffBox(s, ui)
	return g.aw+g.bw+g.pw-d.X-d.W <= 0
}

// DiffRoom is how many rows of lines the view shows on this screen, what
// PgUp and PgDn move by.
func DiffRoom(s *state.State, ui UI) int {
	b := DiffBox(s, ui)
	return max(layout(s, ui).upper-diffFixed-diffHead(ui.Diff, b.W-4), 0)
}

// DiffPanel is the diff's panel letter: its line rows' cells carry their
// row plus one, the rest 0.
const DiffPanel = 'D'

// diffView is the diff box w by h (design D8): the file's name in the top
// edge, "+N −M" at its right; the path and the comparison; a rule; then
// the sides' names when apart, the bound's note, and the lines in view, or
// the note in their place. Its frame in the title colour while it has the
// focus.
func diffView(ui UI, w, h int) []line {
	v := ui.Diff
	inner := w - 4
	focused := ui.Focus == "D"
	_, name := splitPath(v.Path)
	var right line
	if v.Layout != nil {
		right = line{deco(fmt.Sprintf("+%d", v.Added), stDone), deco(" ", stNone), deco(fmt.Sprintf("−%d", v.Removed), stFailed)}
	}
	body := []line{diffPathRow(v, inner), teleRule(inner)}
	room := max(h-diffFixed, 0)
	switch {
	case v.Note != "":
		body = append(body, wrapDeco(v.Note, inner, "", either(v.Failed, stAttention, stGrey))...) // whole, on as many rows as it takes
	case v.Layout == nil:
	default:
		side := DiffSideBySide(inner)
		if side {
			body = append(body, diffSideHeads(v, inner))
			room--
		}
		if v.Whole {
			body = append(body, decoLine("too large to align: the changed middle shown removed, then added", stAttention))
			room--
		}
		if s := noNewline(v); s != "" && !side {
			body = append(body, decoLine(s, stGrey))
			room--
		}
		rows, _ := DiffRows(v, inner)
		top := min(max(v.Top, 0), max(len(rows)-room, 0))
		for i := top; i < top+room && i < len(rows); i++ {
			var l line
			if side {
				l = diffSideRow(v, rows[i], inner)
			} else {
				l = diffUniRow(v, rows[i], inner)
			}
			for j := range l {
				l[j].ref = ref{panel: DiffPanel, kind: refDeco, item: i + 1}
			}
			body = append(body, l)
		}
	}
	return box(DiffPanel, zoneTitle("DIFF · "+name, focused), right, body, w, h, borderFor(focused))
}

// diffPathRow is the path's row: its directory muted, its name, a rename's
// "old → new"; then the comparison, a conflict named and said unresolved,
// and a side ending with no newline.
func diffPathRow(v DiffView, w int) line {
	var l line
	if v.Orig != "" {
		l = append(l, deco(v.Orig, stGrey), deco(" → ", stGrey))
	}
	dir, name := splitPath(v.Path)
	l = append(l, deco(dir, stGrey), deco(name, stExpFileName))
	what := "Working tree"
	if v.Staged {
		what = "Staged"
	}
	if v.Conflict != "" {
		what = "Conflict, " + v.Conflict + ", unresolved"
	}
	l = append(l, deco(" · ", stGrey), deco(what, stTitle), deco(": "+sideName(v.OldName)+" → "+sideName(v.NewName), stMuted))
	if v.Both {
		other := "staged"
		if v.Staged {
			other = "working tree"
		}
		l = append(l, deco(" · Tab "+other, stGrey))
	}
	return l.fit(w)
}

// sideName is a side's name, "nothing" for none.
func sideName(n string) string {
	if n == "" {
		return "nothing"
	}
	return n
}

// diffSideHeads is the row over the sides: ORIGINAL and MODIFIED with what
// each is, a side ending with no newline said.
func diffSideHeads(v DiffView, w int) line {
	half := (w - 1) / 2
	head := func(t, n string, nonl bool, width int) line {
		l := line{deco(t, stTitleBold), deco("  "+sideName(n), stMuted)}
		if nonl {
			l = append(l, deco(" · no newline at the end", stGrey))
		}
		return l.fit(width)
	}
	l := head("ORIGINAL", v.OldName, v.NoNewlineOld, half)
	l = append(l, deco("│", stFrame), deco(" ", stNone))
	return append(l, head("MODIFIED", v.NewName, v.NoNewlineNew, w-half-2)...)
}

// diffUniRow is a unified row w cells: a hunk's header in lavender, or a
// line's old and new numbers in grey, its "−"/"+" and its text, removed on
// the red ground, added on the green one, context in the muted colour.
func diffUniRow(v DiffView, r DiffRow, w int) line {
	hk := v.Hunks[r.Hunk]
	nw := v.Layout.NumWidth
	if r.L < 0 {
		return line{deco(fmt.Sprintf("@@ -%d,%d +%d,%d @@", hk.OldStart, hk.OldLines, hk.NewStart, hk.NewLines), stTitle)}.fit(w)
	}
	ln := hk.Lines[r.L]
	num := func(n int) string {
		if n == 0 {
			return strings.Repeat(" ", nw)
		}
		return fmt.Sprintf("%*d", nw, n)
	}
	l := line{deco(num(ln.Old)+" "+num(ln.New)+" ", stGrey)}
	return append(l, diffText(ln, v.Left, w-l.width())...)
}

// diffSideRow is a side by side row w cells: a hunk's header across both
// sides, or the left side's line (a removed or a context one) facing the
// right side's (an added or the same context one), each with its number,
// blank where a side has none.
func diffSideRow(v DiffView, r DiffRow, w int) line {
	hk := v.Hunks[r.Hunk]
	if r.L < 0 && r.R < 0 {
		return line{deco(fmt.Sprintf("@@ -%d,%d +%d,%d @@", hk.OldStart, hk.OldLines, hk.NewStart, hk.NewLines), stTitle)}.fit(w)
	}
	nw := v.Layout.NumWidth
	half := (w - 1) / 2
	side := func(i int32, old bool, width int) line {
		if i < 0 {
			return blankRow(width)
		}
		ln := hk.Lines[i]
		n := ln.New
		if old {
			n = ln.Old
		}
		l := line{deco(fmt.Sprintf("%*d ", nw, n), stGrey)}
		return append(l, diffText(ln, v.Left, width-l.width())...)
	}
	l := side(r.L, true, half)
	l = append(l, deco("│", stFrame))
	return append(l, side(r.R, false, w-half-1)...)
}

// diffText is a line's marker and its text from column left, w cells: a
// removed line's on the red ground, an added one's on the green, cut with
// "…" when it goes on; tabs to the next multiple of 4 and control
// characters as the editor shows them.
func diffText(ln DiffLine, left, w int) line {
	if w <= 0 {
		return nil
	}
	mark, st, mst, g := "  ", stMuted, stNone, (*ground)(nil)
	switch ln.Kind {
	case '-':
		mark, st, mst, g = "− ", stDiffDel, stFailed, diffDelGround
	case '+':
		mark, st, mst, g = "+ ", stDiffAdd, stDone, diffAddGround
	}
	on := func(s *lipgloss.Style) *lipgloss.Style {
		if g == nil {
			return s
		}
		return g.on(s)
	}
	var b strings.Builder
	col, stop := 0, left+w-2+1 // one cell past the room, for fit's "…"
	for i, after := 0, 0; i < len(ln.Text); i = after {
		if col >= stop {
			break
		}
		c := cellAt(ln.Text, i)
		after = i + c.n
		if ln.Text[i] == '\t' {
			c.text, c.w = strings.Repeat(" ", TabWidth-col%TabWidth), TabWidth-col%TabWidth
		}
		switch {
		case col+c.w <= left:
		case col < left:
			b.WriteString(strings.Repeat(" ", col+c.w-left))
		default:
			b.WriteString(c.text)
		}
		col += c.w
	}
	l := line{deco(mark, on(mst)), deco(b.String(), on(st))}.fit(w)
	if n := w - l.width(); n > 0 {
		l = append(l, deco(strings.Repeat(" ", n), on(stNone)))
	}
	for i := range l {
		if g != nil && l[i].st == stNone {
			l[i].st = on(stNone)
		}
	}
	return l
}

// diffKeys are the view's key hints in the footer while it has the focus,
// the longest first: the first that fits shows (design D9).
var diffKeys = []string{
	"↑↓ PgUp PgDn Home End scroll · ←→ shift · n N change · Tab other comparison · f wide · r compare again · Esc back · F1 keys",
	"↑↓ PgUp PgDn scroll · ←→ shift · n N change · Tab other · f wide · Esc back · F1 keys",
	"↑↓ scroll · n N change · Tab other · f wide · Esc back",
	"↑↓ · n N · Esc back",
}

// diffHints is the footer's text while the diff v has the focus, on a screen
// w columns wide: under 100 columns, with no zone A to widen over, no "f";
// for a file with one comparison, no Tab.
func diffHints(v DiffView, w int) string {
	trim := func(k string) string {
		if w < standardWidth {
			k = strings.Replace(k, " · f wide", "", 1)
		}
		if !v.Both {
			k = strings.Replace(strings.Replace(k, " · Tab other comparison", "", 1), " · Tab other", "", 1)
		}
		return k
	}
	for _, k := range diffKeys {
		if k = trim(k); textWidth(k)+1 <= w {
			return k
		}
	}
	return trim(diffKeys[len(diffKeys)-1])
}
