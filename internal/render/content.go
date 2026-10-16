package render

import (
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/Evengelius/agent-command-center/internal/state"
)

// entryLines is the content of a zone B entry: the logical lines both drawn
// and copied (ui-design.md, Selection and copy: the content, not the
// drawing). Labels, symbols and the streaming cursor are drawn around them
// and never part of them. Step 8's Markdown renderer draws these same lines,
// styled, so selection and copy do not change with it.
func entryLines(e state.Entry) []string {
	switch e.Role {
	case "tool", "approval":
		if i := strings.IndexByte(e.Text, '\n'); i >= 0 {
			return []string{e.Text[:i]}
		}
		return []string{e.Text}
	}
	return strings.Split(e.Text, "\n")
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
func wrapSource(src string, first, rest int, at ref, st lipgloss.Style) []line {
	rs := wrapRanges(src, first, rest)
	out := make([]line, 0, len(rs))
	for _, r := range rs {
		out = append(out, rowLine(src, r, at, st))
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

// rowLine draws one wrapped row of src.
func rowLine(src string, r rowRange, at ref, st lipgloss.Style) line {
	if r.from == r.to {
		a := at
		a.kind, a.off = refText, r.from
		return line{{st: st, ref: a}} // an empty line still takes a click
	}
	cs := cells(src[r.from:r.to])
	for i := range cs {
		cs[i].off += r.from
	}
	return cellSpans(src, cs, at, st)
}

// WrapCache keeps where source lines break, so a frame wraps only the lines
// that changed and draws only the rows it shows (proposal step 3, item 7: a
// 2,000-line answer within 16 ms a frame). It is a memo, never a source of
// what is drawn: a nil cache wraps every line, with the same frame. Lines
// not used in a frame are dropped after it.
type WrapCache struct {
	cur, old map[wrapKey][]rowRange
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
// one text span, and each tab or control character an atom.
func cellSpans(src string, cs []cell, at ref, st lipgloss.Style) line {
	var l line
	start := -1 // the first byte of the current run
	flush := func(end int) {
		if start >= 0 {
			r := at
			r.kind, r.off = refText, start
			l = append(l, span{text: src[start:end], st: st, ref: r})
			start = -1
		}
	}
	for _, c := range cs {
		if !c.ctrl && src[c.off] != '\t' {
			if start < 0 {
				start = c.off
			}
			continue
		}
		flush(c.off)
		r := at
		r.kind, r.off, r.n = refAtom, c.off, c.n
		cst := st
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
	l     line
	src   string
	r     rowRange
	line  int  // the source line, in the entry
	first bool // the source line's first row
	p     *rowStyle
}

// rowStyle is what the rows of one entry share.
type rowStyle struct {
	lead0, lead span
	at          ref
	st          lipgloss.Style
}

func (c crow) draw() line {
	if c.p == nil {
		return c.l
	}
	lead, at := c.p.lead, c.p.at
	if c.first && c.line == 0 {
		lead = c.p.lead0 // the entry's first row
	}
	at.line = c.line
	return append(line{lead}, rowLine(c.src, c.r, at, c.p.st)...)
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
// of two.
func contentRows(out []crow, wc *WrapCache, i int, lines []string, w int, st lipgloss.Style) []crow {
	lead := deco("  ", st)
	p := &rowStyle{lead0: lead, lead: lead, at: ref{panel: 'B', item: i}, st: st}
	for j, l := range lines {
		out = sourceRows(out, wc, p, j, l, w-2, w-2)
	}
	return out
}
