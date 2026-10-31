package render

import (
	"fmt"
	"sort"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/Evengelius/agent-command-center/internal/state"
)

// entryLines is the content of a zone B entry: the logical lines both drawn
// and copied (ui-design.md, Selection and copy: the content, not the
// drawing). Labels, symbols and the streaming cursor are drawn around them
// and never part of them. The Markdown renderer draws these same lines as
// Markdown shows them, each cell keeping its bytes, so a selection copies
// them, its ends taking in the markers of what they mark whole (plan 71,
// design D3). A failed tool call
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
	return wrapCells(src, cells(src), first, rest)
}

// wrapCells is where src's cells cs break into rows, the first first cells
// wide and the others rest: as wrapRanges for the cells drawn, Markdown's
// (plan 71, design D2); with none, one empty row at src's end.
func wrapCells(src string, cs []cell, first, rest int) []rowRange {
	width := func(row int) int {
		if row == 0 {
			return max(first, 1)
		}
		return max(rest, 1)
	}
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

// rowCells are the cells of sh that row r of their line shows, found by
// their offsets: a long line's rows cost their own cells.
func rowCells(sh *mdShown, r rowRange) []cell {
	i := sort.Search(len(sh.cells), func(k int) bool { return sh.cells[k].off >= r.from })
	j := i + sort.Search(len(sh.cells)-i, func(k int) bool { return sh.cells[i+k].off >= r.to })
	return sh.cells[i:j]
}

// runAt is the index of the style run that byte off is in.
func runAt(runs []styleRun, off int) int {
	return min(sort.Search(len(runs), func(k int) bool { return runs[k].to > off }), max(len(runs)-1, 0))
}

// shownRow draws row r of a Markdown line as sh shows it (plan 71, design
// D2): its cells, each span keyed by its source bytes; with none, an anchor
// that still takes a click.
func shownRow(src string, r rowRange, at ref, st *lipgloss.Style, sh *mdShown) line {
	cs := rowCells(sh, r)
	if len(cs) == 0 {
		a := at
		a.kind, a.off = refText, r.from
		return line{{st: st, ref: a}}
	}
	return cellSpans(src, cs, at, st, sh.runs)
}

// shownPlain is row r of a Markdown line as sh shows it, its spans not
// keyed, so a panel keys them as its row's content (the inspection's
// Messages tab, plan 5, item 11; a full view).
func shownPlain(src string, r rowRange, sh *mdShown) line {
	var l line
	start, end := -1, 0
	var cur *lipgloss.Style
	flush := func() {
		if start >= 0 {
			l = append(l, sp(src[start:end], cur))
			start = -1
		}
	}
	cs := rowCells(sh, r)
	ri := 0
	if len(cs) > 0 {
		ri = runAt(sh.runs, cs[0].off)
	}
	for _, c := range cs {
		for ri < len(sh.runs)-1 && sh.runs[ri].to <= c.off {
			ri++
		}
		st := sh.runs[ri].st
		if c.ctrl || c.sub || src[c.off] == '\t' {
			flush()
			if c.ctrl {
				st = stControl
			}
			l = append(l, sp(c.text, st))
			continue
		}
		if start >= 0 && (st != cur || c.off != end) {
			flush()
		}
		if start < 0 {
			start, cur = c.off, st
		}
		end = c.off + c.n
	}
	flush()
	return l
}

// mdWrapped is a Markdown line as a full view draws it, w cells wide: its
// rows, the later ones behind lead and its hang (plan 71, design D5).
func mdWrapped(src string, k mdLine, w int, base *lipgloss.Style, lead string) []line {
	tw := max(w-len(lead), 1)
	v := mdShow(src, k, tw, base)
	sh := &v
	var out []line
	for n, r := range wrapCells(src, sh.cells, tw, tw-sh.hang.width()) {
		if n == 0 {
			out = append(out, shownPlain(src, r, sh))
			continue
		}
		row := append(line{deco(lead, stNone)}, sh.hang...)
		out = append(out, append(row, shownPlain(src, r, sh)...))
	}
	return out
}

// WrapCache keeps where source lines break, so a frame wraps only the lines
// that changed and draws only the rows it shows (proposal step 3, item 7: a
// 2,000-line answer within 16 ms a frame). It is a memo, never a source of
// what is drawn: a nil cache wraps every line, with the same frame. Lines
// not used in a frame are dropped after it.
type WrapCache struct {
	cur, old map[wrapKey][]rowRange
	counts   []entryCount // by entry: its rows, kept across frames
	// shown are the Markdown lines drawn, as shown, kept as the wraps are
	// (plan 71, design D2): a long line's rows are drawn from one.
	shown, shownOld map[shownKey]*mdShown
	// delims are a paragraph's lines' runs of markers (lineDelims), kept
	// as the wraps are, so that a frame pairs a paragraph's spans without
	// parsing its lines again (plan 71, spans).
	delims, delimsOld map[delimKey][]mdDelim
	// texts are the Markdown texts drawn, by place, their lines, kinds and
	// wraps, kept as the wraps are, so that a frame reads again only what
	// changed (plan 71, the cache: markdown).
	texts, textsOld map[mdKey]*mdDoc
	scratch         []cell // the cells a wrap builds, for the next (mdRanges)
	reads           [2]int // the texts read whole and from where they grew, for the tests
}

// mdKey is a Markdown text's place, the same from frame to frame: zone B's
// entry i, or message i of the agent the inspection shows.
type mdKey struct {
	panel byte
	agent string
	i     int
}

// mdDoc is a text as Markdown shows it: its lines, as its surface splits
// them, and their kinds; where a text grown from it is read again, and the
// state there (mdRead); its lines' wraps at one width and base (wraps),
// the first known of them carried over from the text it grew from, and
// their rows' count.
type mdDoc struct {
	text   string
	lines  []string
	md     []mdLine
	resume int
	st     mdState
	w      int
	base   *lipgloss.Style
	rows   [][]rowRange
	known  int
	total  int
}

// splitLines are an answer's lines, as entryLines splits them.
func splitLines(text string) []string { return strings.Split(text, "\n") }

// markdown is text at place k, its lines split by split and their kinds:
// from this frame or the one before when it is the same text; else read
// again from where the text before it stops being sure (mdRead) when its
// lines are that text's but the last, which may have grown, as a streamed
// answer's are; else read whole. A nil cache reads it whole.
func (c *WrapCache) markdown(k mdKey, text string, split func(string) []string) *mdDoc {
	if c == nil {
		d, _ := readDoc(text, split(text), nil, lineDelims)
		return d
	}
	d := c.texts[k]
	if d == nil || d.text != text {
		if old := c.textsOld[k]; old != nil && old.text == text {
			d = old
		} else {
			if old == nil {
				old = d
			}
			var grown bool
			d, grown = readDoc(text, split(text), old, c.lineDelims)
			if grown {
				c.reads[1]++
			} else {
				c.reads[0]++
			}
		}
		if c.texts == nil {
			c.texts = map[mdKey]*mdDoc{}
		}
		c.texts[k] = d
	}
	return d
}

// readDoc reads lines, text's, as Markdown: from where base, the text
// before, is read again when lines grew from its lines, base's kinds of the
// lines before taken, and its wraps there; whole otherwise. It reports
// which.
func readDoc(text string, lines []string, base *mdDoc, delims func(src string, from int) []mdDelim) (*mdDoc, bool) {
	d := &mdDoc{text: text, lines: lines, md: make([]mdLine, len(lines))}
	from, st := 0, mdState{}
	grown := base != nil && grownFrom(lines, base.lines)
	if grown {
		from, st = base.resume, base.st
		copy(d.md, base.md[:from])
		if base.rows != nil {
			d.rows, d.w, d.base, d.known = make([][]rowRange, len(lines)), base.w, base.base, min(from, base.known)
			copy(d.rows, base.rows[:d.known])
		}
	}
	d.resume, d.st = mdRead(lines, d.md, from, st, delims)
	return d, grown
}

// grownFrom reports that lines are the lines before, as many or more, all
// but its last the same.
func grownFrom(lines, before []string) bool {
	if len(before) == 0 || len(lines) < len(before) {
		return false
	}
	for j := range len(before) - 1 {
		if lines[j] != before[j] {
			return false
		}
	}
	return true
}

// wraps are d's lines' rows w cells wide in base, as mdRanges breaks them,
// kept with d: the rows it carried over found again only from its known
// lines on, all of them at another width or base.
func (c *WrapCache) wraps(d *mdDoc, w int, base *lipgloss.Style) [][]rowRange {
	if d.rows != nil && d.w == w && d.base == base && d.known == len(d.lines) {
		return d.rows
	}
	if d.rows == nil || d.w != w || d.base != base {
		d.rows, d.w, d.base, d.known = make([][]rowRange, len(d.lines)), w, base, 0
	}
	d.total = 0
	for j := range d.known {
		d.total += len(d.rows[j])
	}
	for j := d.known; j < len(d.lines); j++ {
		d.rows[j] = c.mdRanges(d.lines[j], d.md[j], w, base)
		d.total += len(d.rows[j])
	}
	d.known = len(d.lines)
	return d.rows
}

// delimKey is a line and where its parse for a paragraph's runs begins.
type delimKey struct {
	src  string
	from int
}

// lineDelims is lineDelims from this frame or the one before.
func (c *WrapCache) lineDelims(src string, from int) []mdDelim {
	key := delimKey{src, from}
	if d, ok := c.delims[key]; ok {
		return d
	}
	d, ok := c.delimsOld[key]
	if !ok {
		d = lineDelims(src, from)
	}
	if c.delims == nil {
		c.delims = map[delimKey][]mdDelim{}
	}
	c.delims[key] = d
	return d
}

// shownKey is a line as Markdown shows it: its text, kind, width and base.
type shownKey struct {
	src  string
	md   mdLine
	w    int
	base *lipgloss.Style
}

// show is Markdown line src of kind k as mdShow shows it, from this frame
// or the one before; a nil cache shows it anew.
func (c *WrapCache) show(src string, k mdLine, w int, base *lipgloss.Style) *mdShown {
	if c == nil {
		v := mdShow(src, k, w, base)
		return &v
	}
	key := shownKey{src, k, w, base}
	if sh := c.shown[key]; sh != nil {
		return sh
	}
	sh := c.shownOld[key]
	if sh == nil {
		v := mdShow(src, k, w, base)
		sh = &v
	}
	if c.shown == nil {
		c.shown = map[shownKey]*mdShown{}
	}
	c.shown[key] = sh
	return sh
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

// counted is entry i's row count when the cache holds it. Streaming
// entries and approval cards change between frames and are always counted;
// so are a thinking card, its exit and a phase's details open, which change
// with the frame's time or a count come later (plan 22, design 6).
func (c *WrapCache) counted(i int, e *state.Entry, w int, shape uint16) (int, bool) {
	if !c.keeps(e, shape) || i >= len(c.counts) {
		return 0, false
	}
	if k := c.counts[i]; k.text == e.Text && k.role == e.Role && k.w == w && k.thinking == e.Thinking && k.shape == shape {
		return k.n, true
	}
	return 0, false
}

// keepCount keeps entry i's row count n, for an entry the cache keeps.
func (c *WrapCache) keepCount(i int, e *state.Entry, w int, shape uint16, n int) {
	if !c.keeps(e, shape) {
		return
	}
	for len(c.counts) <= i {
		c.counts = append(c.counts, entryCount{w: -1})
	}
	c.counts[i] = entryCount{e.Text, e.Role, w, e.Thinking, shape, n}
}

// keeps says the cache keeps the entry's count: none changing between frames.
func (c *WrapCache) keeps(e *state.Entry, shape uint16) bool {
	return c != nil && !e.Streaming && e.Role != "approval" && shape&(shapeAnim|shapeDetails) == 0
}

// wrapKey is a line's wrap: its text and widths, and, shown as Markdown,
// its kind (plan 71, design D2).
type wrapKey struct {
	src         string
	first, rest int
	md          mdLine
	shown       bool
}

// NewWrapCache is an empty cache.
func NewWrapCache() *WrapCache { return &WrapCache{cur: map[wrapKey][]rowRange{}} }

func (c *WrapCache) ranges(src string, first, rest int) []rowRange {
	if c == nil {
		return wrapRanges(src, first, rest)
	}
	return c.cached(wrapKey{src: src, first: first, rest: rest}, func() []rowRange { return wrapRanges(src, first, rest) })
}

// mdRanges is where Markdown line src of kind k breaks into rows w cells
// wide, the later ones less its hang (plan 71, design D2): its cells built
// in the cache's scratch, kept for the next line when they are few.
func (c *WrapCache) mdRanges(src string, k mdLine, w int, base *lipgloss.Style) []rowRange {
	if c == nil {
		rs, _ := mdWrap(nil, src, k, w, base)
		return rs
	}
	return c.cached(wrapKey{src: src, first: w, rest: w, md: k, shown: true}, func() []rowRange {
		rs, buf := mdWrap(c.scratch, src, k, w, base)
		if cap(buf) <= maxScratch {
			c.scratch = buf
		}
		return rs
	})
}

// maxScratch is the most cells the cache keeps for wrapping the next line:
// a longer line's are dropped with it.
const maxScratch = 4096

// cached is k's wrap from this frame or the one before, else wrap's.
func (c *WrapCache) cached(k wrapKey, wrap func() []rowRange) []rowRange {
	if rs, ok := c.cur[k]; ok {
		return rs
	}
	rs, ok := c.old[k]
	if !ok {
		rs = wrap()
	}
	c.cur[k] = rs
	return rs
}

// sweep ends a frame: what it did not use is dropped at the next one. The
// next frame's maps are made as large as this one's, as a frame most often
// uses what the frame before did.
func (c *WrapCache) sweep() {
	if c != nil {
		c.old, c.cur = c.cur, make(map[wrapKey][]rowRange, len(c.cur))
		c.shownOld, c.shown = c.shown, sized(c.shown)
		c.delimsOld, c.delims = c.delims, sized(c.delims)
		c.textsOld, c.texts = c.texts, sized(c.texts)
	}
}

// sized is an empty map as large as m, nil for an empty m.
func sized[K comparable, V any](m map[K]V) map[K]V {
	if len(m) == 0 {
		return nil
	}
	return make(map[K]V, len(m))
}

// cellSpans turns drawn cells into spans: runs shown as their bytes become
// one text span, and each tab, control character or Markdown glyph an atom.
// With runs, a span also ends where the style changes; it ends too where
// hidden bytes leave a gap between two cells (plan 71, design D2).
func cellSpans(src string, cs []cell, at ref, st *lipgloss.Style, runs []styleRun) line {
	var l line
	start, end := -1, 0 // the current run's first byte, and the end of its last cell
	cur, ri := st, 0
	if runs != nil && len(cs) > 0 {
		ri = runAt(runs, cs[0].off)
	}
	flush := func() {
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
		if !c.ctrl && !c.sub && src[c.off] != '\t' {
			if start >= 0 && (cst != cur || c.off != end) {
				flush()
			}
			if start < 0 {
				start, cur = c.off, cst
			}
			end = c.off + c.n
			continue
		}
		flush()
		r := at
		r.kind, r.off, r.n = refAtom, c.off, c.n
		if c.ctrl {
			cst = stControl
		}
		l = append(l, span{text: c.text, st: cst, ref: r})
	}
	flush()
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
	fold   string      // a click's fold in ui.Unfolded: a run's row (plan 19), a thinking summary row (plan 42)
	entry  int         // the entry, 1-based, whose own content the row shows; 0 none (plan 42)
	fr     *frame      // the card it is drawn in; nil for none (plan 20)
	cursor bool        // the entry cursor's row: ▸ in the card's margin
	sep    bool        // the rule between two messages (amendment 17)
	// turn and file are a review's file row: a double click opens its diff
	// (plan 65, design D8); 0 for any other row.
	turn, file int
	// more is a long paste's fold row, "… N more lines": it stands for the
	// lines folded under it, which a search finds (plan 66, design D5).
	more bool
	// summary is the summary suggestion's buttons row (plan 70, design D2).
	summary bool
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
	md          []mdLine   // the lines' Markdown kinds; nil draws them as they are
	w           int        // the text's width, for Markdown's rules (plan 71)
	wc          *WrapCache // the lines shown, kept (plan 71)
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
	l := append(line{}, lead...)
	if c.line < len(c.p.md) {
		// The whole source line, shown as Markdown: a span may cross the wrap
		// (plan 71, design D2).
		sh := c.p.wc.show(c.src, c.p.md[c.line], c.p.w, c.p.st)
		if !c.first {
			l = append(l, sh.hang...)
		}
		return append(l, shownRow(c.src, c.r, at, c.p.st, sh)...)
	}
	return append(l, rowLine(c.src, c.r, at, c.p.st, nil)...)
}

// drawn adds rows already drawn.
func drawn(out []crow, ls ...line) []crow {
	for _, l := range ls {
		out = append(out, crow{l: l})
	}
	return out
}
