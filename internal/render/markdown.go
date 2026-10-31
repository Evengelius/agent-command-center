package render

import (
	"sort"
	"strconv"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/Evengelius/agent-command-center/internal/state"
)

// AI text as Markdown (ui-design.md, Markdown; plan 71): each source line is
// shown as Markdown renders it. Its markers are hidden; a list's bullet, a
// quote's bar, a table's borders and a rule are drawn as glyphs; its other
// bytes are drawn as they are, styled. Every cell shown keeps the source
// bytes it stands for, a glyph as an atom does, so the wrap, the entry
// cursor, the search and a selection's copy work on the source (plan 71,
// designs D1 to D3). A line's block kind comes from its own text and the
// lines around it (the fences above it, its table's rows), and so do the
// spans that run over a paragraph's lines (spans); its inline constructs
// are read only for the rows a frame shows.

// markdownOn draws AI text as Markdown; tests turn it off to compare.
var markdownOn = true

// mdKind is a source line's block kind.
type mdKind uint8

const (
	mdPlain mdKind = iota
	mdHeading
	mdQuote
	mdList
	mdTable
	mdRule
	mdFence // an opening or closing fence line
	mdCode  // inside a fence; an unclosed fence runs to the end
	mdNone  // not Markdown: shown as it is (a thinking phase's own details)
)

// mdLine is a line's kind and, for code and fences, its language; open is what
// a code line starts inside, the delimiter that closes it: "*/" for a block
// comment (plan 5, item 12), a backtick or three double or single quotes for
// a string over several lines (plan 6, item 3); "" when it starts in none. A
// table's row has its columns' widths, comma-separated, head on its header
// row (plan 71, design D4). bom marks a first line that starts with a byte
// order mark, a file's, not its text. A paragraph's line has carry, what it
// starts inside of the spans opened on a line above it, and cross, its
// markers that open or close such spans (encodeMarks).
type mdLine struct {
	kind  mdKind
	lang  string
	open  string
	cols  string
	head  bool
	bom   bool
	carry uint8
	cross string
}

// bomMark is UTF-8's byte order mark, which Windows' tools put at a file's
// start.
const bomMark = "\uFEFF"

// The constructs a span over a paragraph's lines sets, bit by bit, in
// mdLine.carry and a mark's.
const (
	mdInBold uint8 = 1 << iota
	mdInItalic
	mdInStrike
	mdInCode
)

// mdLines classifies an entry's lines; fences, and block comments and
// strings within a fence, carry over from line to line, a table's rows
// share their columns, and a paragraph's spans run over its lines.
func mdLines(lines []string) []mdLine {
	out := make([]mdLine, len(lines))
	mdRead(lines, out, 0, mdState{}, lineDelims)
	return out
}

// mdState is what a line's kind takes from the lines before it: the open
// fence's marker, "" outside one, its language, and what its code is
// inside (mdLine.open).
type mdState struct{ fence, lang, open string }

// mdRead classifies lines into out from line from on, out holding the
// kinds of the lines before and st the state they leave, a line's runs of
// markers from delims. It returns where a text that grows from lines is
// read again, and the state there: its last line, still growing, or
// before it the paragraph or the run of rows starting with "|" that the
// line before the last is in, which the last may go on (plan 71, the
// cache: WrapCache.markdown).
func mdRead(lines []string, out []mdLine, from int, st mdState, delims func(src string, from int) []mdDelim) (int, mdState) {
	var last mdState // the state before the last line
	for j := from; j < len(lines); j++ {
		if j == len(lines)-1 {
			last = st
		}
		l := lines[j]
		if j == 0 {
			l = strings.TrimPrefix(l, bomMark) // classified as its text
		}
		t := strings.TrimLeft(l, " ")
		indent := len(l) - len(t)
		if st.fence != "" {
			if indent < 4 && strings.HasPrefix(t, st.fence) && strings.Trim(strings.TrimRight(t, " \t"), st.fence[:1]) == "" {
				out[j], st = mdLine{kind: mdFence, lang: st.lang}, mdState{}
				continue
			}
			out[j] = mdLine{kind: mdCode, lang: st.lang, open: st.open}
			if st.lang != "" {
				st.open = codeRuns(nil, l, st.lang, st.open)
			}
			continue
		}
		if indent < 4 {
			if m := fenceMarker(t); m != "" {
				lang, _, _ := strings.Cut(strings.TrimSpace(t[len(m):]), " ")
				st = mdState{fence: m, lang: lang}
				out[j] = mdLine{kind: mdFence, lang: lang}
				continue
			}
		}
		out[j] = mdLine{kind: blockKind(t, indent)}
	}
	if from == 0 && len(lines) > 0 && strings.HasPrefix(lines[0], bomMark) {
		out[0].bom = true
	}
	at := len(lines) - 1
	for _, s := range []int{tables(lines[from:], out[from:]), spans(lines[from:], out[from:], delims)} {
		if s >= 0 {
			at = min(at, from+s)
		}
	}
	if at < 0 {
		return 0, mdState{}
	}
	if at < len(lines)-1 {
		return at, mdState{} // a paragraph's or a table's first line, outside any fence
	}
	return at, last
}

// The part a line takes in a paragraph (mdSegment).
const (
	mdParaNone      = iota // it ends one: a blank line, a heading, a table, a rule, code, a fence
	mdParaText             // text, which goes on the paragraph before it, or begins one
	mdParaItem             // a list item's text, which begins one
	mdParaQuote            // a quote's text, which goes on a quote's paragraph as deep
	mdParaQuoteItem        // a list item's text in a quote, which begins one
)

// mdSegment is where line src, of kind k, holds a paragraph's text, its
// bytes from from to its end, as show parses them; its part in a paragraph;
// and, for a quote's, how deep it is.
func mdSegment(src string, k mdLine) (from, part, depth int) {
	start := 0
	if k.bom && strings.HasPrefix(src, bomMark) {
		start = len(bomMark)
	}
	t := strings.TrimLeft(src[start:], " ")
	at := len(src) - len(t)
	switch k.kind {
	case mdPlain:
		if strings.TrimSpace(t) == "" {
			return 0, mdParaNone, 0
		}
		return at, mdParaText, 0
	case mdList:
		return at + listMarker(src[at:]), mdParaItem, 0
	case mdQuote:
		for at < len(src) && src[at] == '>' {
			depth++
			at++
			if at < len(src) && src[at] == ' ' {
				at++
			}
		}
		rest := strings.TrimLeft(src[at:], " ")
		in := at + len(src[at:]) - len(rest)
		switch {
		case blockKind(rest, 0) == mdHeading, strings.TrimSpace(rest) == "":
			return 0, mdParaNone, depth
		case blockKind(rest, 0) == mdList:
			return in + listMarker(rest), mdParaQuoteItem, depth
		}
		return in, mdParaQuote, depth
	}
	return 0, mdParaNone, 0
}

// mdDelim is a run of markers at its line's top level, outside any
// construct of the line's own parse: a run of backticks or of * _ or ~ the
// line shows as written or, used, a run of * _ or ~ that opens a span of
// the line's own.
type mdDelim struct {
	off, n int
	c      byte
	used   bool
}

// lineDelims are line src's runs at its top level, its text parsed from
// from to its end.
func lineDelims(src string, from int) []mdDelim {
	b := mdBuild{src: src, dry: true}
	b.inline(from, len(src), mdFlags{base: stText})
	return b.delims
}

// spans pairs the runs a paragraph's lines leave open into the spans that
// run over its lines, in out's carry and cross (plan 71, after the owner's
// acceptance run: a span over two lines was shown with its markers). A
// paragraph is its lines from one that begins it to the one before a line
// that ends it or begins another: a blank line, a heading, a table, a rule,
// code or a fence ends one; text goes on it, a list item begins one, a
// quote's text goes on a quote's paragraph as deep (pairPara pairs them).
// It returns the first line of the paragraph the line before the last is
// in, -1 for none (mdRead).
func spans(lines []string, out []mdLine, delims func(src string, from int) []mdDelim) int {
	var para []mdSeg
	began, deep := mdParaNone, 0 // the part that began the paragraph, its depth
	held, hold := -1, len(lines)-2
	end := func() {
		if len(para) > 0 && para[0].line <= hold && hold <= para[len(para)-1].line {
			held = para[0].line
		}
		if len(para) > 1 {
			pairPara(lines, out, para, delims)
		}
		para, began = para[:0], mdParaNone
	}
	for j, l := range lines {
		from, part, depth := mdSegment(l, out[j])
		switch part {
		case mdParaNone:
			end()
			continue
		case mdParaItem, mdParaQuoteItem:
			end()
		case mdParaQuote:
			if began != mdParaQuote && began != mdParaQuoteItem || deep != depth {
				end()
			}
		}
		if len(para) == 0 {
			began, deep = part, depth
		}
		para = append(para, mdSeg{j, from})
	}
	end()
	return held
}

// mdSeg is a paragraph's line and where its text begins.
type mdSeg struct{ line, from int }

// pairPara pairs paragraph para's runs over its lines, as CommonMark reads a
// paragraph's text whole, each run by the rules its line's own parse
// applies. The lines are read in order, each from where its text begins and
// again after each marker that closes a span, or opens a code span, over
// lines, as para parses the part after one:
//   - a run of backticks shown as written opens a code span when a later
//     line has a run as long, the first of which closes it, whatever its
//     line's own parse makes of it; the text between is the span's;
//   - a run of emphasis that may close (mayClose) closes the nearest opener
//     of its marker and length on an earlier line, the openers between
//     dropped, even a run its line's own parse opens a span with;
//   - one shown as written that may open (mayOpen) is an opener.
//
// A run with no partner is shown as written, as before.
func pairPara(lines []string, out []mdLine, para []mdSeg, delims func(src string, from int) []mdDelim) {
	type opener struct {
		seq, line, off, n int
		bits              uint8
	}
	marks := map[int][]mdMark{}
	span := func(o opener, line, c int) {
		marks[o.line] = append(marks[o.line], mdMark{o.off, o.n, o.bits, true})
		marks[line] = append(marks[line], mdMark{c, o.n, o.bits, false})
		for l := o.line + 1; l <= line; l++ {
			out[l].carry |= o.bits
		}
	}
	// ticks is, by length, the last of para's lines with a run of backticks
	// as long, made when a run asks.
	var ticks map[int]int
	later := func(p, n int) bool {
		if ticks == nil {
			ticks = map[int]int{}
			for q, s := range para {
				src := lines[s.line]
				for i := s.from; i < len(src); {
					if src[i] != '`' {
						i++
						continue
					}
					m := runOf(src, i, len(src), '`')
					ticks[m] = q
					i += m
				}
			}
		}
		q, ok := ticks[n]
		return ok && q > p
	}
	var stacks [7][]opener // emphasis's openers left open, by delimKind
	var code opener        // the open code span's opener, n 0 for none
	seq := 0
	for p, s := range para {
		src := lines[s.line]
		pos := s.from
	read:
		for {
			if code.n > 0 {
				c := codeEnd(src, pos, len(src), code.n)
				if c < 0 {
					break // the line's text is the code span's
				}
				span(code, s.line, c)
				pos, code = c+code.n, opener{}
			}
			for _, d := range delims(src, pos) {
				if d.c == '`' {
					if later(p, d.n) {
						code = opener{line: s.line, off: d.off, n: d.n, bits: mdInCode}
						pos = d.off + d.n
						continue read
					}
					continue
				}
				k := delimKind(d)
				if k < 0 {
					continue
				}
				if st := stacks[k]; len(st) > 0 && st[len(st)-1].line != s.line && mayClose(src, s.from, d) {
					o := st[len(st)-1]
					span(o, s.line, d.off)
					for x := range stacks {
						for len(stacks[x]) > 0 && stacks[x][len(stacks[x])-1].seq >= o.seq {
							stacks[x] = stacks[x][:len(stacks[x])-1]
						}
					}
					pos = d.off + d.n
					continue read
				}
				if !d.used && mayOpen(src, d) {
					stacks[k] = append(stacks[k], opener{seq, s.line, d.off, d.n, delimBits(d)})
					seq++
				}
			}
			break
		}
	}
	for l, ms := range marks {
		out[l].cross = encodeMarks(ms)
	}
}

// mayOpen reports whether run d of line src may open a span, as emphasis
// has it: before a non-space, an underscore run not after a letter or
// digit.
func mayOpen(src string, d mdDelim) bool {
	next := d.off + d.n
	return next < len(src) && src[next] != ' ' && (d.c != '_' || !wordByte(src, d.off-1))
}

// mayClose reports whether run d of line src, its text from from, may close
// a span, as closeRun has it: after a non-space of its text, an underscore
// run not before a letter or digit.
func mayClose(src string, from int, d mdDelim) bool {
	prev := d.off - 1
	return prev >= from && src[prev] != ' ' && (d.c != '_' || !wordByte(src, d.off+d.n))
}

// delimKind is a run of emphasis's marker and length as a stack's index:
// * and _ once, twice or three times, ~ twice; -1 for any other run.
func delimKind(d mdDelim) int {
	switch {
	case d.c == '*' && d.n <= 3:
		return d.n - 1
	case d.c == '_' && d.n <= 3:
		return 2 + d.n
	case d.c == '~' && d.n == 2:
		return 6
	}
	return -1
}

// delimBits are the constructs a run of emphasis sets.
func delimBits(d mdDelim) uint8 {
	switch {
	case d.c == '~':
		return mdInStrike
	case d.n == 1:
		return mdInItalic
	case d.n == 2:
		return mdInBold
	}
	return mdInBold | mdInItalic
}

// mdMark is a run of markers that opens or closes a span over a
// paragraph's lines: its bytes [off, off+n) and the constructs it sets.
type mdMark struct {
	off, n int
	bits   uint8
	open   bool
}

// encodeMarks are marks as mdLine.cross holds them, by offset, each
// "off,n,bits" and "+" for an opener or "-" for a closer, then ";".
func encodeMarks(ms []mdMark) string {
	sort.Slice(ms, func(a, b int) bool { return ms[a].off < ms[b].off })
	var b []byte
	for _, m := range ms {
		b = strconv.AppendInt(b, int64(m.off), 10)
		b = append(b, ',')
		b = strconv.AppendInt(b, int64(m.n), 10)
		b = append(b, ',')
		b = strconv.AppendInt(b, int64(m.bits), 10)
		if m.open {
			b = append(b, '+')
		} else {
			b = append(b, '-')
		}
		b = append(b, ';')
	}
	return string(b)
}

// decodeMarks are the marks of mdLine.cross s.
func decodeMarks(s string) []mdMark {
	var out []mdMark
	for s != "" {
		var e string
		e, s, _ = strings.Cut(s, ";")
		if len(e) < 2 {
			continue
		}
		f := strings.Split(e[:len(e)-1], ",")
		if len(f) != 3 {
			continue
		}
		off, _ := strconv.Atoi(f[0])
		n, _ := strconv.Atoi(f[1])
		bits, _ := strconv.Atoi(f[2])
		out = append(out, mdMark{off, n, uint8(bits), e[len(e)-1] == '+'})
	}
	return out
}

// tables confirms each run of rows starting with "|" as a table when its
// second row is the header's rule (GitHub's tables), its first then the
// header; the columns are as wide as their widest cell as shown. A run with
// no rule is text. It returns the first line of the run the line before
// the last is in, -1 for none (mdRead).
func tables(lines []string, out []mdLine) int {
	held, hold := -1, len(out)-2
	for j := 0; j < len(out); {
		if out[j].kind != mdTable {
			j++
			continue
		}
		e := j + 1
		for e < len(out) && out[e].kind == mdTable {
			e++
		}
		if j <= hold && hold < e {
			held = j
		}
		if e-j < 2 || !tableRule(lines[j+1]) {
			for x := j; x < e; x++ {
				out[x].kind = mdPlain
			}
			j = e
			continue
		}
		var widths []int
		for x := j; x < e; x++ {
			if x == j+1 {
				continue
			}
			for c, r := range tableCells(lines[x]) {
				var b mdBuild
				b.src = lines[x]
				b.inline(r[0], r[1], mdFlags{base: stText})
				w := b.width()
				if c < len(widths) {
					widths[c] = max(widths[c], w)
				} else {
					widths = append(widths, w)
				}
			}
		}
		parts := make([]string, len(widths))
		for c, w := range widths {
			parts[c] = strconv.Itoa(w)
		}
		cols := strings.Join(parts, ",")
		for x := j; x < e; x++ {
			out[x].cols = cols
		}
		out[j].head = true
		j = e
	}
	return held
}

// fenceMarker is the run of three or more backticks or tildes t starts
// with, or "".
func fenceMarker(t string) string {
	if len(t) < 3 || (t[0] != '`' && t[0] != '~') {
		return ""
	}
	n := 0
	for n < len(t) && t[n] == t[0] {
		n++
	}
	if n < 3 || t[0] == '`' && strings.IndexByte(t[n:], '`') >= 0 {
		return ""
	}
	return t[:n]
}

// blockKind is a line's kind from its text t after indent spaces: a list
// item at any indent, so that a nested one is one; the others under four.
func blockKind(t string, indent int) mdKind {
	if t == "" {
		return mdPlain
	}
	if indent < 4 {
		switch {
		case t[0] == '#':
			n := 0
			for n < len(t) && t[n] == '#' {
				n++
			}
			if n <= 6 && (n == len(t) || t[n] == ' ') {
				return mdHeading
			}
		case t[0] == '>':
			return mdQuote
		case t[0] == '|':
			return mdTable
		case isRule(t):
			return mdRule
		}
	}
	if listMarker(t) > 0 {
		return mdList
	}
	return mdPlain
}

// isRule reports a thematic break: three or more of -, * or _, spaces
// between.
func isRule(t string) bool {
	n := 0
	for i := 0; i < len(t); i++ {
		switch {
		case t[i] == t[0] && (t[0] == '-' || t[0] == '*' || t[0] == '_'):
			n++
		case t[i] != ' ':
			return false
		}
	}
	return n >= 3
}

// listMarker is the length of t's list marker with its space ("- ", "12. "),
// or 0.
func listMarker(t string) int {
	if len(t) >= 2 && (t[0] == '-' || t[0] == '*' || t[0] == '+') && t[1] == ' ' {
		return 2
	}
	n := 0
	for n < len(t) && n < 9 && t[n] >= '0' && t[n] <= '9' {
		n++
	}
	if n > 0 && n+1 < len(t) && (t[n] == '.' || t[n] == ')') && t[n+1] == ' ' {
		return n + 2
	}
	return 0
}

// tableRule reports a table's header rule: pipes, dashes, colons and
// spaces, a dash among them.
func tableRule(l string) bool {
	t := strings.TrimSpace(l)
	return strings.HasPrefix(t, "|") && strings.Contains(t, "-") && strings.Trim(t, "|-: ") == ""
}

// tableCells are a table row's cells, each the bytes [from, to) of its
// content without the spaces around it: between its pipes, a pipe escaped or
// in a code span not one; with no pipe at its end, its last runs to it.
func tableCells(src string) [][2]int {
	var pipes []int
	for i := 0; i < len(src); {
		switch src[i] {
		case '\\':
			i += 2
			continue
		case '`':
			n := runOf(src, i, len(src), '`')
			if j := codeEnd(src, i+n, len(src), n); j >= 0 {
				i = j + n
				continue
			}
			i += n
			continue
		case '|':
			pipes = append(pipes, i)
		}
		i++
	}
	var out [][2]int
	for c := range pipes {
		from, to := pipes[c]+1, len(src)
		if c+1 < len(pipes) {
			to = pipes[c+1]
		} else if strings.TrimSpace(src[from:]) == "" {
			break // the row's closing pipe
		}
		for from < to && src[from] == ' ' {
			from++
		}
		for to > from && src[to-1] == ' ' {
			to--
		}
		out = append(out, [2]int{from, to})
	}
	return out
}

// styleRun styles src's bytes up to to, from the previous run's end.
type styleRun struct {
	to int
	st *lipgloss.Style
}

// runs builds a line's style runs; adjacent runs of one style merge.
type runs []styleRun

func (r *runs) add(to int, st *lipgloss.Style) {
	if n := len(*r); n > 0 && ((*r)[n-1].st == st || (*r)[n-1].to >= to) {
		if (*r)[n-1].to < to {
			(*r)[n-1].to = to
		}
		return
	}
	*r = append(*r, styleRun{to, st})
}

// mdShown is a line as Markdown shows it (plan 71, design D1): its cells in
// source order, hidden bytes in none and a glyph in one, sub set, its source
// bytes kept; the styles of its bytes; and what its later rows start with,
// so that a list item's and a quote's wrapped rows hang under its text.
type mdShown struct {
	cells []cell
	runs  []styleRun
	hang  line
}

// mdBuild is a line's mdShown as it is built, byte by byte in order; with
// keep, its constructs' hidden markers too, for a copy (mdWiden). With dry
// it only parses, keeping the runs of markers at its top level
// (lineDelims); depth is how deep in a construct the parse is. carry and
// marks are the line's spans over its paragraph's lines (para). With
// measure it builds the cells alone, no style, for a wrap (mdWrap).
type mdBuild struct {
	src     string
	cells   []cell
	runs    runs
	keep    bool
	pairs   []mdPair
	dry     bool
	depth   int
	delims  []mdDelim
	carry   uint8
	marks   []mdMark
	paraAt  []int // where para began, when trace is set: the tests'
	trace   bool
	measure bool
}

// mdPair is a construct's hidden opener, the bytes [o0, o1), and its
// closer [c0, c1): an emphasis's runs, a code span's backticks; an empty
// closer after what an escape's backslash or a link's bracket opens, whose
// end is shown. An affix of the line's block, a heading's #s or a fence's
// marker, has no closer, c0 -1.
type mdPair struct{ o0, o1, c0, c1 int }

// pair keeps a construct's markers, when b keeps them.
func (b *mdBuild) pair(o0, o1, c0, c1 int) {
	if b.keep && o0 < o1 {
		b.pairs = append(b.pairs, mdPair{o0, o1, c0, c1})
	}
}

// text shows the bytes [from, to) as they are, in st.
func (b *mdBuild) text(from, to int, st *lipgloss.Style) {
	if from >= to || b.dry {
		return
	}
	b.cells = appendCells(b.cells, b.src[from:to], from)
	if !b.measure {
		b.runs.add(to, st)
	}
}

// appendCells adds s's cells, as cells splits it, to cs, s being bytes off
// on of the line.
func appendCells(cs []cell, s string, off int) []cell {
	for i := 0; i < len(s); {
		c := cellAt(s, i)
		c.off += off
		cs = append(cs, c)
		i += c.n
	}
	return cs
}

// hide shows nothing of the bytes [from, to).
func (b *mdBuild) hide(from, to int) {
	if from < to && !b.dry && !b.measure {
		b.runs.add(to, stNone)
	}
}

// glyph shows g in st in place of the bytes [from, to).
func (b *mdBuild) glyph(from, to int, g string, st *lipgloss.Style) {
	if b.dry {
		return
	}
	b.cells = append(b.cells, cell{text: g, w: textWidth(g), off: from, n: to - from, sub: true})
	if !b.measure {
		b.runs.add(to, st)
	}
}

// delim notes a run of markers shown as written or, used, opening a span:
// one at the top level is one of the line's runs another line may pair
// (lineDelims).
func (b *mdBuild) delim(i, n int, c byte, used bool) {
	if b.dry && b.depth == 0 {
		b.delims = append(b.delims, mdDelim{i, n, c, used})
	}
}

// width is the cells' width.
func (b *mdBuild) width() int {
	n := 0
	for _, c := range b.cells {
		n += c.w
	}
	return n
}

// mdShow is line src, of kind k, as Markdown shows it in rows w cells wide
// (a rule is as wide), its text in base (zone B's text colour, a phase's
// details' muted one).
func mdShow(src string, k mdLine, w int, base *lipgloss.Style) mdShown {
	b := mdBuild{src: src, cells: make([]cell, 0, len(src))} // a cell takes a byte or more, a glyph too
	hang := b.show(k, w, base)
	return mdShown{cells: b.cells, runs: b.runs, hang: hang}
}

// mdWrap is where line src, of kind k, breaks into rows w cells wide, the
// later ones less its hang, as mdShow shows it: its cells, which the rows
// keep none of, built in buf, and given back grown for the next line.
func mdWrap(buf []cell, src string, k mdLine, w int, base *lipgloss.Style) ([]rowRange, []cell) {
	if cap(buf) < len(src) {
		buf = make([]cell, 0, len(src))
	}
	b := mdBuild{src: src, cells: buf[:0], measure: true}
	hang := b.show(k, w, base)
	return wrapCells(src, b.cells, w, w-hang.width()), b.cells
}

// show builds b's line, of kind k, as mdShow says, and returns what its
// later rows start with.
func (b *mdBuild) show(k mdLine, w int, base *lipgloss.Style) line {
	src := b.src
	var hang line
	if k.carry != 0 || k.cross != "" {
		b.carry, b.marks = k.carry, decodeMarks(k.cross)
	}
	start := 0 // the line's text, after a file's byte order mark
	if k.bom && strings.HasPrefix(src, bomMark) {
		b.hide(0, len(bomMark))
		start = len(bomMark)
	}
	t := strings.TrimLeft(src[start:], " ")
	at := len(src) - len(t)
	switch k.kind {
	case mdNone:
		b.text(start, len(src), base)
	case mdCode:
		b.text(start, at, base)
		switch {
		case k.lang == "":
			b.text(at, len(src), stMdCode)
		case b.measure:
			b.cells = appendCells(b.cells, src[at:], at)
		default:
			var r runs
			r.add(at, base)
			codeRuns(&r, src, k.lang, k.open)
			b.cells = appendCells(b.cells, src[at:], at)
			b.runs = r
		}
	case mdFence:
		m := fenceMarker(t)
		if m == "" {
			m = strings.TrimRight(t, " \t")
		}
		b.hide(0, at+len(m))
		info := src[at+len(m):]
		lead := len(info) - len(strings.TrimLeft(info, " "))
		b.hide(at+len(m), at+len(m)+lead)
		from := at + len(m) + lead
		b.pair(0, from, -1, -1)
		if k.lang != "" && strings.HasPrefix(src[from:], k.lang) {
			b.text(from, from+len(k.lang), stMdLang)
			from += len(k.lang)
		}
		b.text(from, len(src), stGrey)
	case mdRule:
		b.glyph(start, len(src), strings.Repeat("─", max(w, 1)), stMdMark)
	case mdTable:
		b.table(k)
	case mdQuote:
		b.text(start, at, base)
		depth := 0
		for at < len(src) && src[at] == '>' {
			b.glyph(at, at+1, "│", stMdMark)
			depth++
			at++
			if at < len(src) && src[at] == ' ' {
				b.text(at, at+1, base)
				at++
			}
		}
		hang = line{deco(strings.Repeat("│ ", depth), stMdMark)}
		rest := strings.TrimLeft(src[at:], " ")
		in := at + len(src[at:]) - len(rest)
		b.text(at, in, base)
		switch blockKind(rest, 0) {
		case mdHeading:
			b.heading(in)
		case mdList:
			hang = append(hang, spaces(in-at+b.list(in, in-at, stMdEm))...)
		default:
			b.para(in, len(src), stMdEm)
		}
	case mdHeading:
		b.heading(at)
	case mdList:
		b.text(start, at, base)
		hang = spaces(textWidth(src[start:at]) + b.list(at, at-start, base))
	default:
		b.text(start, at, base)
		b.para(at, len(src), base)
	}
	return hang
}

// para shows a paragraph's line's text, the bytes [from, to), as inline
// Markdown on base: from inside the spans its carry says it starts in, its
// marks hidden, each opening or closing a span over the paragraph's lines
// (spans), each part between them parsed as the line's own. A code span's
// part is its text.
func (b *mdBuild) para(from, to int, base *lipgloss.Style) {
	if b.trace {
		b.paraAt = append(b.paraAt, from)
	}
	f := mdFlags{base: base, bold: b.carry&mdInBold != 0, italic: b.carry&mdInItalic != 0, strike: b.carry&mdInStrike != 0}
	code := b.carry&mdInCode != 0
	at := from
	part := func(end int) {
		if code {
			b.text(at, end, f.on(stMdCode))
		} else {
			b.inline(at, end, f)
		}
	}
	for _, m := range b.marks {
		if m.off < at || m.off+m.n > to {
			continue
		}
		part(m.off)
		b.hide(m.off, m.off+m.n)
		switch on := m.open; {
		case m.bits&mdInCode != 0:
			code = on
		default:
			if m.bits&mdInBold != 0 {
				f.bold = on
			}
			if m.bits&mdInItalic != 0 {
				f.italic = on
			}
			if m.bits&mdInStrike != 0 {
				f.strike = on
			}
		}
		at = m.off + m.n
	}
	part(to)
}

// heading shows a heading from at, its #s: them and the spaces after them
// hidden, as a closing run of #s, the text in the headings' style.
func (b *mdBuild) heading(at int) {
	src := b.src
	n := runOf(src, at, len(src), '#')
	from := at + n
	for from < len(src) && src[from] == ' ' {
		from++
	}
	b.hide(at, from)
	end := len(strings.TrimRight(src, " "))
	if j := strings.TrimRight(src[:end], "#"); len(j) < end && len(j) >= from && (len(j) == from || src[len(j)-1] == ' ') {
		end = len(strings.TrimRight(j, " "))
	}
	end = max(end, from)
	b.inline(from, end, mdFlags{base: stMdHeading})
	b.hide(end, len(src))
	b.pair(at, from, -1, -1)
	b.pair(end, len(src), -1, -1)
}

// list shows a list item from at, indent spaces in: its bullet a glyph, • and
// ◦ and ▪ by depth, or its number, in the bullets' colour, then its text in
// base; it is the cells before the text, for the item's later rows.
func (b *mdBuild) list(at, indent int, base *lipgloss.Style) int {
	src := b.src
	n := listMarker(src[at:])
	if c := src[at]; c == '-' || c == '*' || c == '+' {
		b.glyph(at, at+1, [3]string{"•", "◦", "▪"}[min(indent/2, 2)], stMdBullet)
	} else {
		b.text(at, at+n-1, stMdBullet)
	}
	b.text(at+n-1, at+n, base)
	b.para(at+n, len(src), base)
	return textWidth(src[at:at+n-1]) + 1
}

// table shows a table's row: the header's rule as one rule across its
// columns, ├─┼─┤; a row's pipes as bars │ and its cells padded to their
// column, the header's bold (plan 71, design D4).
func (b *mdBuild) table(k mdLine) {
	src := b.src
	var cols []int
	for _, c := range strings.Split(k.cols, ",") {
		n, _ := strconv.Atoi(c)
		cols = append(cols, n)
	}
	if tableRule(src) {
		var g strings.Builder
		g.WriteString("├")
		for c, w := range cols {
			if c > 0 {
				g.WriteString("┼")
			}
			g.WriteString(strings.Repeat("─", w+2))
		}
		g.WriteString("┤")
		b.glyph(0, len(src), g.String(), stMdMark)
		return
	}
	base := stText
	if k.head {
		base = stBold
	}
	cells := tableCells(src)
	at := 0
	for c, r := range cells {
		pipe := strings.LastIndexByte(src[:r[0]], '|')
		pad := ""
		if c > 0 {
			pad = strings.Repeat(" ", max(cols[min(c-1, len(cols)-1)]-b.cellWidth(cells[c-1]), 0)) + " "
		}
		b.hide(at, pipe)
		b.glyph(pipe, pipe+1, pad+"│ ", stMdMark)
		b.hide(pipe+1, r[0])
		b.inline(r[0], r[1], mdFlags{base: base})
		at = r[1]
	}
	if last := strings.LastIndexByte(src, '|'); len(cells) > 0 && last >= at {
		c := len(cells) - 1
		pad := strings.Repeat(" ", max(cols[min(c, len(cols)-1)]-b.cellWidth(cells[c]), 0))
		b.hide(at, last)
		b.glyph(last, last+1, pad+" │", stMdMark)
		at = last + 1
	}
	b.hide(at, len(src))
}

// cellWidth is a table cell's width as shown.
func (b *mdBuild) cellWidth(r [2]int) int {
	c := mdBuild{src: b.src}
	c.inline(r[0], r[1], mdFlags{base: stText})
	return c.width()
}

// mdFlags are the inline constructs a byte is inside, on its block's base
// style (plan 71, design D4).
type mdFlags struct {
	base                 *lipgloss.Style
	bold, italic, strike bool
}

func (f mdFlags) st() *lipgloss.Style { return mdStyle(f.base, f.bold, f.italic, f.strike) }

// on is f's constructs on another base: a code span's, a link's.
func (f mdFlags) on(base *lipgloss.Style) *lipgloss.Style {
	return mdStyle(base, f.bold, f.italic, f.strike)
}

// inline shows src[i:end] as inline Markdown in f: escapes, code spans,
// strong, emphasis, both, strikethrough, links and images; a marker with no
// closing one before end is shown as it is.
func (b *mdBuild) inline(i, end int, f mdFlags) {
	src := b.src
	for i < end {
		switch c := src[i]; c {
		case '\\':
			if i+1 < end && mdPunct(src[i+1]) {
				b.hide(i, i+1)
				b.text(i+1, i+2, f.st())
				b.pair(i, i+1, i+2, i+2)
				i += 2
				continue
			}
		case '`':
			n := runOf(src, i, end, '`')
			if j := codeEnd(src, i+n, end, n); j >= 0 {
				b.hide(i, i+n)
				b.text(i+n, j, f.on(stMdCode))
				b.hide(j, j+n)
				b.pair(i, i+n, j, j+n)
				i = j + n
				continue
			}
			b.delim(i, n, c, false)
			b.text(i, i+n, f.st())
			i += n
			continue
		case '*', '_', '~':
			if j := b.emphasis(i, end, f); j > i {
				i = j
				continue
			}
			n := runOf(src, i, end, c)
			b.delim(i, n, c, false)
			b.text(i, i+n, f.st())
			i += n
			continue
		case '[', '!':
			if j := b.link(i, end, f); j > i {
				i = j
				continue
			}
		}
		j := i + 1
		for j < end && !mdSpecial(src[j]) {
			j++
		}
		b.text(i, j, f.st())
		i = j
	}
}

// emphasis shows the strong, emphasis or strikethrough run opening at i and
// returns where it ends, or i when its marker opens none: a run of one, two
// or three * or _, or two ~, before a non-space, closed by a run as long
// after a non-space before end; an underscore run neither after nor before a
// letter or digit, so that a snake_case name stays as it is.
func (b *mdBuild) emphasis(i, end int, f mdFlags) int {
	src := b.src
	c := src[i]
	n := runOf(src, i, end, c)
	if n > 3 || c == '~' && n != 2 || i+n >= end || src[i+n] == ' ' || c == '_' && wordByte(src, i-1) {
		return i
	}
	j := closeRun(src, i+n, end, c, n)
	if j < 0 {
		return i
	}
	g := f
	switch {
	case c == '~':
		g.strike = true
	case n == 1:
		g.italic = true
	case n == 2:
		g.bold = true
	default:
		g.bold, g.italic = true, true
	}
	b.delim(i, n, c, true)
	b.hide(i, i+n)
	b.depth++
	b.inline(i+n, j, g)
	b.depth--
	b.hide(j, j+n)
	b.pair(i, i+n, j, j+n)
	return j + n
}

// closeRun is where the run of exactly n c's that closes one opened before
// from starts, before end, after a non-space, an underscore run not before a
// letter or digit; escapes and code spans are skipped. -1 for none.
func closeRun(src string, from, end int, c byte, n int) int {
	for j := from; j < end; {
		switch src[j] {
		case '\\':
			j += 2
			continue
		case '`':
			m := runOf(src, j, end, '`')
			if k := codeEnd(src, j+m, end, m); k >= 0 {
				j = k + m
				continue
			}
			j += m
			continue
		case c:
			m := runOf(src, j, end, c)
			if m == n && j > from && src[j-1] != ' ' && (c != '_' || !wordByte(src, j+m)) {
				return j
			}
			j += m
			continue
		}
		j++
	}
	return -1
}

// link shows the link or image opening at i, "[label](url)" or
// "![alt](url)", and returns where it ends, or i when none does: its label
// as inline Markdown in the links' style, then its URL grey in brackets,
// "label (url)" (plan 71, design D4).
func (b *mdBuild) link(i, end int, f mdFlags) int {
	src := b.src
	o := i
	if src[i] == '!' {
		if i+1 >= end || src[i+1] != '[' {
			return i
		}
		o = i + 1
	}
	rb := closing(src, o+1, end, '[', ']')
	if rb <= o+1 || rb+1 >= end || src[rb+1] != '(' {
		return i
	}
	rp := closing(src, rb+2, end, '(', ')')
	if rp < 0 {
		return i
	}
	b.hide(i, o+1)
	b.depth++
	b.inline(o+1, rb, mdFlags{base: stMdLink, bold: f.bold, italic: f.italic, strike: f.strike})
	b.depth--
	b.glyph(rb, rb+1, " ", stMdComment) // a space, where the wrap may break
	if d := src[rb+2 : rp]; len(d) >= 2 && d[0] == '<' && d[len(d)-1] == '>' {
		// A destination in angle brackets, which are not its own (Codex
		// links the files it writes so, P12x).
		b.text(rb+1, rb+2, stMdComment)
		b.hide(rb+2, rb+3)
		b.text(rb+3, rp-1, stMdComment)
		b.hide(rp-1, rp)
		b.text(rp, rp+1, stMdComment)
	} else {
		b.text(rb+1, rp+1, stMdComment)
	}
	b.pair(i, o+1, rp+1, rp+1)
	return rp + 1
}

// closing is the close that ends what an open before from opened, nested
// ones counted, escapes and code spans skipped, before end; -1 for none.
func closing(src string, from, end int, open, close byte) int {
	depth := 0
	for j := from; j < end; {
		switch src[j] {
		case '\\':
			j += 2
			continue
		case '`':
			m := runOf(src, j, end, '`')
			if k := codeEnd(src, j+m, end, m); k >= 0 {
				j = k + m
				continue
			}
			j += m
			continue
		case open:
			depth++
		case close:
			if depth == 0 {
				return j
			}
			depth--
		}
		j++
	}
	return -1
}

// codeEnd is where the run of exactly n backticks closing a code span
// starts, from from before end; -1 for none.
func codeEnd(src string, from, end, n int) int {
	for j := from; j < end; {
		if src[j] != '`' {
			j++
			continue
		}
		m := runOf(src, j, end, '`')
		if m == n {
			return j
		}
		j += m
	}
	return -1
}

// runOf is how many c's src has from i before end.
func runOf(src string, i, end int, c byte) int {
	n := 0
	for i+n < end && src[i+n] == c {
		n++
	}
	return n
}

// mdSpecial reports a byte inline Markdown reads.
func mdSpecial(c byte) bool {
	return c == '\\' || c == '`' || c == '*' || c == '_' || c == '~' || c == '[' || c == '!'
}

// mdPunct reports an ASCII punctuation character, which a backslash escapes.
func mdPunct(c byte) bool {
	return c >= '!' && c <= '/' || c >= ':' && c <= '@' || c >= '[' && c <= '`' || c >= '{' && c <= '~'
}

// mdStyleKey is a style of inline Markdown: its base, strong, emphasis and
// strikethrough.
type mdStyleKey struct {
	base                 *lipgloss.Style
	bold, italic, strike bool
}

// mdStyles are the styles inline Markdown made, painted again with the
// palette.
var mdStyles = map[mdStyleKey]*lipgloss.Style{}

// mdStyle is base with the constructs set, made on first use; base itself
// with none.
func mdStyle(base *lipgloss.Style, bold, italic, strike bool) *lipgloss.Style {
	if !bold && !italic && !strike {
		return base
	}
	k := mdStyleKey{base, bold, italic, strike}
	if st := mdStyles[k]; st != nil {
		return st
	}
	st := new(lipgloss.Style)
	paintStyle(st, mdStyleOf(k))
	mdStyles[k] = st
	return st
}

func mdStyleOf(k mdStyleKey) lipgloss.Style {
	v := *k.base
	if k.bold {
		v = v.Bold(true)
	}
	if k.italic {
		v = v.Italic(true)
	}
	if k.strike {
		v = v.Strikethrough(true)
	}
	return v
}

// entryMd is the Markdown kinds of conversation entry e's lines ls as zone B
// shows them: an answer's, and a thinking phase's Codex summary (plan 71,
// design D5); nil for an entry shown as it is.
func entryMd(e state.Entry, ls []string) []mdLine {
	if !markdownOn {
		return nil
	}
	switch e.Role {
	case "assistant":
		return mdLines(ls)
	case "thinking":
		if d := detailsMd(ls[min(1, len(ls)):]); d != nil && len(ls) > 0 {
			return append([]mdLine{{kind: mdNone}}, d...)
		}
	}
	return nil
}

// mdWiden widens a selection's ends, offset o0 of line l0 and o1 of line l1,
// over the hidden markers of the constructs whose content it covers whole,
// so that its copy is their source (plan 71, design D3): an emphasis's or a
// code span's runs, an escape's backslash, a link's bracket; a heading's #s
// and a fence's marker when it covers all their line shows; a span over a
// paragraph's lines when it covers its text from its first line to its
// last. A word of a construct copies as the word. md nil leaves them.
func mdWiden(lines []string, md []mdLine, l0, o0, l1, o1 int) (int, int) {
	in := func(l int) bool { return l >= 0 && l < len(lines) && l < len(md) }
	over := crossSpans(md)
	// A construct taken in widens the selection, which may take in the one
	// around it, on its line or over lines.
	for again := true; again; {
		n0, n1 := o0, o1
		switch {
		case l0 == l1:
			if in(l0) {
				n0, n1 = widenLine(lines[l0], md[l0], o0, o1)
			}
		default:
			if in(l0) {
				n0, _ = widenLine(lines[l0], md[l0], o0, len(lines[l0]))
			}
			if in(l1) {
				_, n1 = widenLine(lines[l1], md[l1], 0, o1)
			}
		}
		for _, s := range over {
			// Its text taken in whole: the selection from no later than its
			// opener's end to no earlier than its closer's start.
			if (l0 < s.l0 || l0 == s.l0 && n0 <= s.o+s.n) && (l1 > s.l1 || l1 == s.l1 && n1 >= s.c) {
				if l0 == s.l0 {
					n0 = min(n0, s.o)
				}
				if l1 == s.l1 {
					n1 = max(n1, s.c+s.n)
				}
			}
		}
		again = n0 != o0 || n1 != o1
		o0, o1 = n0, n1
	}
	return o0, o1
}

// mdOver is a span over a paragraph's lines: its opener's bytes [o, o+n) on
// line l0, its closer's [c, c+n) on line l1.
type mdOver struct{ l0, o, l1, c, n int }

// crossSpans are the spans over lines that md's marks open and close: a
// closer with the latest opener left of its constructs, as spans nest.
func crossSpans(md []mdLine) []mdOver {
	type opened struct {
		line int
		m    mdMark
	}
	var stack []opened
	var out []mdOver
	for j, k := range md {
		if k.cross == "" {
			continue
		}
		for _, m := range decodeMarks(k.cross) {
			if m.open {
				stack = append(stack, opened{j, m})
				continue
			}
			for x := len(stack) - 1; x >= 0; x-- {
				if o := stack[x]; o.m.bits == m.bits && o.m.n == m.n {
					out = append(out, mdOver{o.line, o.m.off, j, m.off, m.n})
					stack = stack[:x]
					break
				}
			}
		}
	}
	return out
}

// widenLine is mdWiden on the bytes [o0, o1) of line src, of kind k.
func widenLine(src string, k mdLine, o0, o1 int) (int, int) {
	o0 = min(max(o0, 0), len(src))
	o1 = min(max(o1, o0), len(src))
	b := mdBuild{src: src, keep: true}
	b.show(k, 1, stText)
	first, last := len(src), len(src) // the bytes the line shows
	if n := len(b.cells); n > 0 {
		first, last = b.cells[0].off, b.cells[n-1].off+b.cells[n-1].n
	}
	// A construct taken in widens the selection, which may take in the one
	// around it.
	for again := true; again; {
		again = false
		for _, p := range b.pairs {
			whole := o0 <= first && o1 >= last // an affix: all the line shows
			if p.c0 >= 0 {
				whole = o0 <= p.o1 && o1 >= p.c0 // what is between opener and closer
			}
			if end := max(p.o1, p.c1); whole && (p.o0 < o0 || end > o1) {
				o0, o1, again = min(o0, p.o0), max(o1, end), true
			}
		}
	}
	return o0, o1
}

// mdText is text's lines as Markdown shows them, without styles: each line's
// shown cells' text, glyphs included (a timeline's or a row's excerpt, plan
// 71, design D5). With room > 0 only its first lines are read, until they
// hold room bytes, and one more, for a table's rule.
func mdText(text string, room int) string {
	var lines []string
	for rest, n, more := text, 0, true; more; {
		var l string
		l, rest, more = strings.Cut(rest, "\n")
		lines = append(lines, l)
		if room > 0 && n >= room {
			break
		}
		n += len(l) + 1
	}
	ks := mdLines(lines)
	var b strings.Builder
	for j, l := range lines {
		if j > 0 {
			b.WriteByte('\n')
		}
		for _, c := range mdShow(l, ks[j], 3, stText).cells {
			b.WriteString(c.text)
		}
	}
	return b.String()
}

// wordByte reports whether src[i] is a letter or digit; out of range is not.
func wordByte(src string, i int) bool {
	if i < 0 || i >= len(src) {
		return false
	}
	c := src[i]
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c >= 0x80
}

// codeRuns styles a line of code in lang: keywords, strings, comments and
// numbers. A string ends with its line, except a backtick one in Go, JavaScript
// and TypeScript and a triple-quoted one in Python; a block comment runs on to
// its */ (plan 5, item 12). in is what the line starts inside (mdLine.open)
// and codeRuns returns what it ends inside, "" for none (plan 6, item 3). A
// nil r only scans for that.
func codeRuns(r *runs, src, lang, in string) string {
	hash, dash := hashComment[strings.ToLower(lang)], dashComment[strings.ToLower(lang)]
	add := func(to int, st *lipgloss.Style) {
		if r != nil {
			r.add(to, st)
		}
	}
	low := strings.ToLower(lang)
	i := 0
	if in != "" {
		st := stMdString
		if in == "*/" {
			st = stMdComment
		}
		j := closeAt(src, 0, in, in != "*/" && (in != "`" || low != "go"))
		if j < 0 {
			add(len(src), st)
			return in
		}
		i = j
		add(i, st)
	}
	for i < len(src) {
		c := src[i]
		switch {
		case hash && c == '#',
			dash && strings.HasPrefix(src[i:], "--"),
			!hash && !dash && strings.HasPrefix(src[i:], "//"):
			add(len(src), stMdComment)
			return ""
		case !hash && strings.HasPrefix(src[i:], "/*"):
			j := strings.Index(src[i+2:], "*/")
			if j < 0 {
				add(len(src), stMdComment)
				return "*/"
			}
			i += 2 + j + 2
			add(i, stMdComment)
		case multiTick[low] && c == '`', pyTriple[low] && (strings.HasPrefix(src[i:], `"""`) || strings.HasPrefix(src[i:], `'''`)):
			d := src[i : i+1]
			if c != '`' {
				d = src[i : i+3]
			}
			j := closeAt(src, i+len(d), d, d != "`" || low != "go")
			if j < 0 {
				add(len(src), stMdString)
				return d
			}
			add(j, stMdString)
			i = j
		case c == '"' || c == '\'' || c == '`':
			j := i + 1
			for j < len(src) && src[j] != c {
				if src[j] == '\\' {
					j++
				}
				j++
			}
			end := min(j+1, len(src))
			add(end, stMdString)
			i = end
		case c >= '0' && c <= '9' && !wordByte(src, i-1) && (i == 0 || src[i-1] != '_'):
			j := i
			for j < len(src) && (wordByte(src, j) || src[j] == '.' || src[j] == '_') {
				j++
			}
			add(j, stMdNumber)
			i = j
		case wordByte(src, i) || c == '_':
			j := i
			for j < len(src) && (wordByte(src, j) || src[j] == '_') {
				j++
			}
			st := stMdCode
			if r != nil && keywords[src[i:j]] {
				st = stMdKeyword
			}
			add(j, st)
			i = j
		default:
			add(i+1, stMdCode)
			i++
		}
	}
	return ""
}

// closeAt is the offset just after the first delim in src from from, a
// backslash escaping the byte after it when esc; -1 when there is none.
func closeAt(src string, from int, delim string, esc bool) int {
	for i := from; i < len(src); i++ {
		if esc && src[i] == '\\' {
			i++
			continue
		}
		if strings.HasPrefix(src[i:], delim) {
			return i + len(delim)
		}
	}
	return -1
}

// Languages whose strings run over several lines (plan 6, item 3): backtick
// ones, and Python's triple-quoted ones.
var (
	multiTick = set("go", "js", "javascript", "jsx", "mjs", "ts", "typescript", "tsx")
	pyTriple  = set("python", "py")
)

// Languages whose line comments start with # or --; the others use //.
var (
	hashComment = set("python", "py", "sh", "bash", "shell", "zsh", "console", "yaml", "yml", "toml", "ruby", "rb", "perl", "r", "powershell", "ps1", "pwsh", "dockerfile", "makefile", "make", "ini", "conf", "nix", "elixir")
	dashComment = set("sql", "lua", "haskell", "hs")
)

// keywords are the common ones of the languages an agent writes most; one
// set for all, since a line is short and the cost of a word wrongly
// coloured is a colour.
var keywords = set(
	// Go
	"break", "case", "chan", "const", "continue", "default", "defer", "else", "fallthrough", "for", "func", "go", "goto", "if", "import", "interface", "map", "package", "range", "return", "select", "struct", "switch", "type", "var", "nil", "true", "false",
	// JavaScript, TypeScript, Java, C, C#, Rust
	"async", "await", "class", "catch", "delete", "do", "enum", "export", "extends", "finally", "from", "function", "implements", "in", "instanceof", "let", "new", "null", "of", "private", "protected", "public", "static", "super", "this", "throw", "throws", "try", "typeof", "undefined", "void", "while", "yield", "abstract", "final", "int", "long", "double", "float", "boolean", "char", "string", "bool", "fn", "impl", "mut", "pub", "use", "mod", "match", "trait", "where", "self", "Self", "crate", "struct", "unsafe", "namespace", "using", "readonly", "override", "virtual",
	// Python, shell, SQL
	"def", "elif", "except", "lambda", "pass", "raise", "with", "as", "is", "not", "and", "or", "None", "True", "False", "global", "nonlocal", "assert", "then", "fi", "esac", "done", "echo", "local", "SELECT", "FROM", "WHERE", "INSERT", "UPDATE", "DELETE", "JOIN", "ORDER", "BY", "GROUP", "CREATE", "TABLE",
)

func set(words ...string) map[string]bool {
	m := make(map[string]bool, len(words))
	for _, w := range words {
		m[w] = true
	}
	return m
}
