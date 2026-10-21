package render

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/Evengelius/agent-command-center/internal/state"
)

// entryLines is the content of a zone B entry: the logical lines both drawn
// and copied (ui-design.md, Selection and copy: the content, not the
// drawing). Labels, symbols and the streaming cursor are drawn around them
// and never part of them. Step 8's Markdown renderer draws these same lines,
// styled, so selection and copy do not change with it. A failed tool call
// adds its error as a second line, drawn on the same row (plan 5, item 6).
// A phase of thinking that ended is its summary, then its details, drawn
// when open; a live card and a step that left no row have none: the card is
// decoration (plan 22, design 8).
func entryLines(s *state.State, e state.Entry) []string {
	switch e.Role {
	case "thinking":
		if !thoughtShown(e.Thought) {
			return nil
		}
		text, _, _, _ := thoughtSummary(e.Thought)
		return append([]string{text}, thoughtDetails(e.Thought)...)
	case "tool":
		name, _, _ := strings.Cut(e.Text, "\n")
		if err := toolError(s, e); err != "" {
			return []string{name, err}
		}
		return []string{name}
	case "approval", "error":
		if i := strings.IndexByte(e.Text, '\n'); i >= 0 {
			return []string{e.Text[:i]}
		}
		return []string{e.Text}
	}
	return strings.Split(e.Text, "\n")
}

// toolError is what B shows of a failed call's error: the first non-blank
// line of its output, as received; "" when the call did not fail or wrote
// nothing. A Codex command that exited non-zero shows "exit N" and the last
// non-blank line of its output instead (plan 6, item 1): Codex sends one
// output, stdout and stderr aggregated, so the error line cannot be singled
// out and a failing command most often ends with it.
func toolError(s *state.State, e state.Entry) string {
	if s == nil {
		return ""
	}
	tc := s.Tools[e.MessageID]
	if tc == nil || tc.Status != state.ToolFailed {
		return ""
	}
	if tc.Name == "commandExecution" && tc.HasExitCode && tc.ExitCode != 0 {
		out := fmt.Sprintf("exit %d", tc.ExitCode)
		ls := strings.Split(tc.Output, "\n")
		for i := len(ls) - 1; i >= 0; i-- {
			if l := strings.TrimRight(ls[i], " \t\r"); strings.TrimSpace(l) != "" {
				return out + " · " + l
			}
		}
		return out
	}
	for l := range strings.SplitSeq(tc.Output, "\n") {
		if l = strings.TrimRight(l, " \t\r"); strings.TrimSpace(l) != "" {
			return l
		}
	}
	return ""
}

// Long sent messages fold to their first lines (ui-design.md, Zone B, Long
// pasted messages; ASM-22).
const (
	foldOver = 20
	foldShow = 5
)

// wrapSource wraps one source line into rows, the first at most first cells
// wide and the others rest, breaking after a space where it can. Each span
// keeps the bytes it shows (at, with the offset): tabs and control
// characters are drawn as in the editor and copied as they are, and a
// wrapped line copies as one line.
func wrapSource(src string, first, rest int, at ref, st *lipgloss.Style) []line {
	rs := wrapRanges(src, first, rest)
	out := make([]line, 0, len(rs))
	for _, r := range rs {
		out = append(out, rowLine(src, r, at, st, nil))
	}
	return out
}

// rowRange is the bytes [from, to) of a source line one wrapped row shows; a
// space the wrap drops at a row's edge is in neither row.
type rowRange struct{ from, to int }

// wrapRanges is where wrapSource breaks src: one range per row, at least one.
func wrapRanges(src string, first, rest int) []rowRange {
	if src == "" {
		return []rowRange{{0, 0}}
	}
	width := func(row int) int {
		if row == 0 {
			return max(first, 1)
		}
		return max(rest, 1)
	}
	cs := cells(src)
	var rows []rowRange
	cur := rowRange{-1, -1} // the current row's cells, as a range of cs
	n := func() int { return cur.to - cur.from }
	cw, space := 0, -1 // the row's width and its last space, in cs
	// end is cells a to b-1 as source bytes.
	end := func(a, b int) rowRange {
		return rowRange{cs[a].off, cs[b-1].off + cs[b-1].n}
	}
	for i, c := range cs {
		if cur.from < 0 {
			cur = rowRange{i, i}
		}
		skip := false
		for cw+c.w > width(len(rows)) && n() > 0 {
			switch {
			case c.text == " ": // a space at the edge ends the row, undrawn
				rows, cur = append(rows, end(cur.from, cur.to)), rowRange{i + 1, i + 1}
				skip = true
			case space >= 0: // the word moves to the next row
				rows = append(rows, end(cur.from, space+1))
				cur.from = space + 1
			default: // a word longer than the row is cut
				rows, cur = append(rows, end(cur.from, cur.to)), rowRange{i, i}
			}
			cw, space = 0, -1
			for _, d := range cs[cur.from:cur.to] {
				cw += d.w
			}
		}
		if skip {
			continue
		}
		if c.text == " " {
			space = i
		}
		cur.to, cw = i+1, cw+c.w
	}
	if n() > 0 || len(rows) == 0 {
		if n() > 0 {
			rows = append(rows, end(cur.from, cur.to))
		} else {
			rows = append(rows, rowRange{len(src), len(src)})
		}
	}
	return rows
}

// rowLine draws one wrapped row of src; runs, when not nil, style its bytes.
func rowLine(src string, r rowRange, at ref, st *lipgloss.Style, runs []styleRun) line {
	if r.from == r.to {
		a := at
		a.kind, a.off = refText, r.from
		return line{{st: st, ref: a}} // an empty line still takes a click
	}
	cs := cells(src[r.from:r.to])
	for i := range cs {
		cs[i].off += r.from
	}
	return cellSpans(src, cs, at, st, runs)
}

// styledRow is row r of src as spans of its runs' styles, st without runs;
// they are not keyed, so a panel keys them as its row's content (the
// inspection's Messages tab, plan 5, item 11).
func styledRow(src string, r rowRange, runs []styleRun, st *lipgloss.Style) line {
	if runs == nil || r.from == r.to {
		return plain(src[r.from:r.to], st)
	}
	var l line
	from := 0
	for _, x := range runs {
		a, b := max(from, r.from), min(x.to, r.to)
		if a < b {
			l = append(l, sp(src[a:b], x.st))
		}
		from = x.to
	}
	if from < r.to { // past the last run: the base style
		l = append(l, sp(src[max(from, r.from):r.to], st))
	}
	return l
}

// WrapCache keeps where source lines break, so a frame wraps only the lines
// that changed and draws only the rows it shows (proposal step 3, item 7: a
// 2,000-line answer within 16 ms a frame). It is a memo, never a source of
// what is drawn: a nil cache wraps every line, with the same frame. Lines
// not used in a frame are dropped after it.
type WrapCache struct {
	cur, old map[wrapKey][]rowRange
	counts   []entryCount // by entry: its rows, kept across frames
}

// entryCount is how many rows a finished entry takes at width w: a function of
// these fields alone, so an entry that moved still finds a right count. shape
// is what its look adds (plan 19): the labels above it, or folded away.
type entryCount struct {
	text, role string
	w          int
	thinking   bool
	shape      uint16
	n          int
}

// rowCount is entry i's row count, from count when not cached. Streaming
// entries and approval cards change between frames and are always counted;
// so are a thinking card, its exit and a phase's details open, which change
// with the frame's time or a count come later (plan 22, design 6).
func (c *WrapCache) rowCount(i int, e state.Entry, w int, shape uint16, count func() int) int {
	if c == nil || e.Streaming || e.Role == "approval" || shape&(shapeAnim|shapeDetails) != 0 {
		return count()
	}
	if i < len(c.counts) {
		if k := c.counts[i]; k.text == e.Text && k.role == e.Role && k.w == w && k.thinking == e.Thinking && k.shape == shape {
			return k.n
		}
	}
	n := count()
	for len(c.counts) <= i {
		c.counts = append(c.counts, entryCount{w: -1})
	}
	c.counts[i] = entryCount{e.Text, e.Role, w, e.Thinking, shape, n}
	return n
}

type wrapKey struct {
	src         string
	first, rest int
}

// NewWrapCache is an empty cache.
func NewWrapCache() *WrapCache { return &WrapCache{cur: map[wrapKey][]rowRange{}} }

func (c *WrapCache) ranges(src string, first, rest int) []rowRange {
	if c == nil {
		return wrapRanges(src, first, rest)
	}
	k := wrapKey{src, first, rest}
	if rs, ok := c.cur[k]; ok {
		return rs
	}
	rs, ok := c.old[k]
	if !ok {
		rs = wrapRanges(src, first, rest)
	}
	c.cur[k] = rs
	return rs
}

// sweep ends a frame: what it did not use is dropped at the next one.
func (c *WrapCache) sweep() {
	if c != nil {
		c.old, c.cur = c.cur, make(map[wrapKey][]rowRange, len(c.cur))
	}
}

// cellSpans turns drawn cells into spans: runs shown as their bytes become
// one text span, and each tab or control character an atom. With runs, a
// span also ends where the style changes.
func cellSpans(src string, cs []cell, at ref, st *lipgloss.Style, runs []styleRun) line {
	var l line
	start := -1 // the first byte of the current run
	cur, ri := st, 0
	flush := func(end int) {
		if start >= 0 {
			r := at
			r.kind, r.off = refText, start
			l = append(l, span{text: src[start:end], st: cur, ref: r})
			start = -1
		}
	}
	for _, c := range cs {
		cst := st
		if runs != nil {
			for ri < len(runs)-1 && runs[ri].to <= c.off {
				ri++
			}
			cst = runs[ri].st
		}
		if !c.ctrl && src[c.off] != '\t' {
			if start >= 0 && cst != cur {
				flush(c.off)
			}
			if start < 0 {
				start, cur = c.off, cst
			}
			continue
		}
		flush(c.off)
		r := at
		r.kind, r.off, r.n = refAtom, c.off, c.n
		if c.ctrl {
			cst = stControl
		}
		l = append(l, span{text: c.text, st: cst, ref: r})
	}
	if len(cs) > 0 {
		last := cs[len(cs)-1]
		flush(last.off + last.n)
	}
	return l
}

// crow is one row of the conversation: drawn, or a wrapped row of a source
// line, drawn only when it is shown. Rows share their styles through p: a
// lipgloss.Style is hundreds of bytes, and a long answer has thousands of
// rows each frame.
type crow struct {
	l      line
	src    string
	r      rowRange
	line   int  // the source line, in the entry
	first  bool // the source line's first row
	p      *rowStyle
	fn     func() line // a row drawn only when shown
	fold   string      // a fold row's run (plan 19)
	fr     *frame      // the card it is drawn in; nil for none (plan 20)
	cursor bool        // the entry cursor's row: ▸ in the card's margin
	sep    bool        // the rule between two messages (amendment 17)
}

// solid says the row shows something: text, a block's bar, a rule; not a
// blank row between messages.
func (c crow) solid() bool {
	return c.fn != nil || c.p != nil || c.fr != nil || c.l.width() > 0
}

// rowStyle is what the rows of one entry share. lead0 and lead have the
// same width: a label beside the text, then its column (plan 20).
type rowStyle struct {
	lead0, lead line
	at          ref
	st          *lipgloss.Style
	md          []mdLine // the lines' Markdown kinds; nil draws them plain
}

// draw is the row as shown: its content, in its card.
func (c crow) draw() line {
	l := c.content()
	if c.fr != nil {
		l = c.fr.wrap(l, c.cursor)
	}
	return l
}

// content is the row without its card.
func (c crow) content() line {
	if c.fn != nil {
		return c.fn()
	}
	if c.p == nil {
		return c.l
	}
	lead, at := c.p.lead, c.p.at
	if c.first && c.line == 0 {
		lead = c.p.lead0 // the entry's first row
	}
	at.line = c.line
	var rs []styleRun
	if c.line < len(c.p.md) {
		rs = mdRuns(c.src, c.p.md[c.line]) // the whole source line: a span may cross the wrap
	}
	return append(append(line{}, lead...), rowLine(c.src, c.r, at, c.p.st, rs)...)
}

// drawn adds rows already drawn.
func drawn(out []crow, ls ...line) []crow {
	for _, l := range ls {
		out = append(out, crow{l: l})
	}
	return out
}

// sourceRows adds the wrapped rows of source line j, the first first cells
// wide, the others rest; the entry's first row is led by p.lead0, the
// others by p.lead.
func sourceRows(out []crow, wc *WrapCache, p *rowStyle, j int, src string, first, rest int) []crow {
	for k, r := range wc.ranges(src, first, rest) {
		out = append(out, crow{src: src, r: r, line: j, first: k == 0, p: p})
	}
	return out
}

// contentRows adds entry i's lines, each wrapped to w cells after an indent
// of two; markdown styles them as Markdown (ui-design.md, Markdown).
func contentRows(out []crow, wc *WrapCache, i int, lines []string, w int, st *lipgloss.Style, markdown bool) []crow {
	lead := line{deco("  ", st)}
	p := &rowStyle{lead0: lead, lead: lead, at: ref{panel: 'B', item: i}, st: st}
	if markdown && markdownOn {
		p.md = mdLines(lines)
	}
	for j, l := range lines {
		out = sourceRows(out, wc, p, j, l, w-2, w-2)
	}
	return out
}
