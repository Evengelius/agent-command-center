// Package diff compares two texts line by line for the Source Control's diff
// view (plan 44, design D7): a minimal Myers difference in linear space, cut
// into hunks with their context, bounded so that a huge or wholly rewritten
// file is answered at once. It is pure: standard library only, no file, no
// process, no clock.
package diff

import (
	"bytes"
	"strings"
)

// Kind tells what a line of a hunk is, as the first column of a unified
// diff does.
type Kind byte

// The three kinds of line.
const (
	Context Kind = ' ' // in both sides
	Removed Kind = '-' // in the old side only
	Added   Kind = '+' // in the new side only
)

// Line is one line of a hunk.
type Line struct {
	Kind Kind
	// Old and New are the line's 1-based numbers in each side, 0 where the
	// side has none: a Removed line has no New, an Added line no Old.
	Old, New int
	// Text is the line without its end: a CR before the LF is dropped and
	// invalid UTF-8 is replaced by U+FFFD. It can still hold a CR that is
	// not before an LF, a tab or another control character: drawing it is
	// the caller's business.
	Text string
}

// Hunk is a run of changes with the context around it, numbered as the
// header "@@ -OldStart,OldLines +NewStart,NewLines @@" of a unified diff:
// a side with no line in the hunk has Start the number of the line before
// it (0 at the top of the file), as git writes it.
type Hunk struct {
	OldStart, OldLines int
	NewStart, NewLines int
	Lines              []Line
}

// Result is the comparison of two texts.
type Result struct {
	// Hunks are the changes with their context; none when the lines are the
	// same.
	Hunks []Hunk
	// Added and Removed count the lines of every hunk.
	Added, Removed int
	// Whole says the bound applied (MaxLines, MaxEdits): the common head and
	// tail are kept and the whole middle is one change, every old line
	// Removed and then every new line Added, whether or not some match.
	Whole bool
	// NoNewlineOld and NoNewlineNew say the side's last line has no LF: it
	// is the line numbered as many as the side has. A last line with an LF
	// and the same text without is not the same line: it shows as a change.
	NoNewlineOld, NoNewlineNew bool
	// EOLOnly says the bytes differ and nothing else does: the lines (a CR
	// before the LF dropped) and the final-newline flags are equal. There
	// is no hunk then.
	EOLOnly bool
}

// The bounds (plan 44, D7). A middle, between the common head and tail, of
// more than MaxLines lines a side, or a difference of more than MaxEdits
// edits (every Removed or Added line is one; a changed line is two), is not
// compared line by line.
const (
	MaxLines = 100_000
	MaxEdits = 20_000
)

// Compare compares the old text a with the new text b: the lines are split
// at LF with a CR before the LF dropped, a minimal set of Removed and Added
// lines is found, and each run of them is returned as a Hunk with context
// lines of context on each side (negative is 0); two runs whose contexts
// touch or overlap are one Hunk. In a replaced block the Removed lines come
// before the Added ones. Above the bounds Whole is set (see Result).
//
// It reads the bytes only; the caller keeps the refusals (binary, size).
func Compare(a, b []byte, context int) Result {
	if context < 0 {
		context = 0
	}
	res := Result{NoNewlineOld: noFinalLF(a), NoNewlineNew: noFinalLF(b)}
	if bytes.Equal(a, b) {
		return res
	}
	sa, sb := split(a), split(b)
	la, lb := sa.lines, sb.lines

	// The common head and tail are never compared again.
	p := 0
	for p < len(la) && p < len(lb) && la[p] == lb[p] {
		p++
	}
	s := 0
	for s < len(la)-p && s < len(lb)-p && la[len(la)-1-s] == lb[len(lb)-1-s] {
		s++
	}
	ma, mb := la[p:len(la)-s], lb[p:len(lb)-s]
	if len(ma) == 0 && len(mb) == 0 {
		res.EOLOnly = true
		return res
	}

	// The bound is the middle's: a long file whose changes lie close together
	// is still compared line by line (plan 44, design D7).
	var blocks []block
	whole := len(ma) > MaxLines || len(mb) > MaxLines
	if !whole {
		var ok bool
		blocks, ok = changes(ma, mb, p)
		whole = !ok
	}
	if whole {
		res.Whole = true
		blocks = []block{{p, len(la) - s, p, len(lb) - s}}
	}
	for _, k := range blocks {
		res.Removed += k.a1 - k.a0
		res.Added += k.b1 - k.b0
	}
	res.Hunks = hunks(sa, sb, blocks, context)
	return res
}

// noFinalLF reports whether data has a last line with no LF.
func noFinalLF(data []byte) bool {
	return len(data) > 0 && data[len(data)-1] != '\n'
}

// text is one side split into lines.
type text struct {
	// lines are the lines compared, each without its end; a last line with
	// no LF has "\n" appended, so that it is not the same line as the text
	// with its LF: no other line holds an LF.
	lines []string
	noLF  bool
}

// split cuts data at LF, dropping a CR before each LF; the empty line after
// a final LF is not a line.
func split(data []byte) text {
	if len(data) == 0 {
		return text{}
	}
	s := string(data)
	t := text{lines: make([]string, 0, strings.Count(s, "\n")+1)}
	for len(s) > 0 {
		i := strings.IndexByte(s, '\n')
		if i < 0 {
			t.lines = append(t.lines, s+"\n")
			t.noLF = true
			break
		}
		end := i
		if i > 0 && s[i-1] == '\r' {
			end--
		}
		t.lines = append(t.lines, s[:end])
		s = s[i+1:]
	}
	return t
}

// at is line i as a hunk shows it, a copy of its own: the Result then holds
// the lines it shows and not the whole texts they were cut from.
func (t text) at(i int) string {
	l := t.lines[i]
	if t.noLF && i == len(t.lines)-1 {
		l = l[:len(l)-1]
	}
	return strings.Clone(strings.ToValidUTF8(l, "�"))
}

// block is one change: the old lines [a0, a1) are removed and the new lines
// [b0, b1) added, indexes in the whole sides; between two blocks the lines
// match one for one.
type block struct{ a0, a1, b0, b1 int }

// changes compares the middle lines ma and mb, which begin at line p of each
// side, and returns the blocks of change; false when it takes more than
// MaxEdits edits.
func changes(ma, mb []string, p int) ([]block, bool) {
	ids := make(map[string]int32, max(len(ma), len(mb)))
	idOf := func(l string) int32 {
		id, ok := ids[l]
		if !ok {
			id = int32(len(ids))
			ids[l] = id
		}
		return id
	}
	ia, ib := make([]int32, len(ma)), make([]int32, len(mb))
	for i, l := range ma {
		ia[i] = idOf(l)
	}
	for j, l := range mb {
		ib[j] = idOf(l)
	}

	// A line the other side lacks is an edit whatever the rest does, and the
	// lines both sides hold bound the edits from below: no more than the
	// common ones can match.
	count := make([]int32, 2*len(ids))
	ca, cb := count[:len(ids)], count[len(ids):]
	for _, id := range ia {
		ca[id]++
	}
	for _, id := range ib {
		cb[id]++
	}
	common := 0
	for id := range ca {
		common += int(min(ca[id], cb[id]))
	}
	if len(ma)+len(mb)-2*common > MaxEdits {
		return nil, false
	}

	d := &differ{inA: make([]bool, len(ma)), inB: make([]bool, len(mb))}
	for i, id := range ia {
		if cb[id] > 0 {
			d.a, d.xa = append(d.a, id), append(d.xa, int32(i))
		}
	}
	for j, id := range ib {
		if ca[id] > 0 {
			d.b, d.xb = append(d.b, id), append(d.xb, int32(j))
		}
	}
	budget := MaxEdits - (len(ma) - len(d.a)) - (len(mb) - len(d.b))
	limit := (budget + 1) / 2
	d.off = min((len(d.a)+len(d.b)+1)/2, limit) + 2
	d.vf, d.vb = make([]int, 2*d.off+1), make([]int, 2*d.off+1)
	if !d.solve(0, len(d.a), 0, len(d.b), budget) {
		return nil, false
	}

	var blocks []block
	i, j := 0, 0
	for i < len(ma) || j < len(mb) {
		if i < len(ma) && j < len(mb) && d.inA[i] && d.inB[j] {
			i, j = i+1, j+1
			continue
		}
		bi, bj := i, j
		for i < len(ma) && !d.inA[i] {
			i++
		}
		for j < len(mb) && !d.inB[j] {
			j++
		}
		if i == bi && j == bj {
			panic("diff: the common lines are not in the same order in both sides")
		}
		blocks = append(blocks, block{p + bi, p + i, p + bj, p + j})
	}
	return blocks, true
}

// differ finds a longest common subsequence of two sequences of line ids
// with Myers' linear space refinement of his O(ND) algorithm (E. Myers, "An
// O(ND) Difference Algorithm and Its Variations", Algorithmica 1, 1986): the
// middle snake of a shortest edit script is found from both ends at once and
// each half is solved on its own.
type differ struct {
	a, b     []int32 // the sequences: only the lines both sides hold
	xa, xb   []int32 // the index of each element in the full middle lines
	inA, inB []bool  // the lines of the full middle that match
	vf, vb   []int   // furthest x on each diagonal, forward and backward
	off      int     // the index of diagonal 0 in vf and vb
}

// none marks a diagonal no path reaches inside the grid.
const none = -1 << 30

// pair marks a[i] and b[j] as a match.
func (d *differ) pair(i, j int) {
	d.inA[d.xa[i]], d.inB[d.xb[j]] = true, true
}

// solve matches a[a0:a1] with b[b0:b1]. With budget >= 0 (the whole problem)
// it gives up, returning false, when the shortest edit script is longer
// than budget.
func (d *differ) solve(a0, a1, b0, b1, budget int) bool {
	for a0 < a1 && b0 < b1 && d.a[a0] == d.b[b0] {
		d.pair(a0, b0)
		a0, b0 = a0+1, b0+1
	}
	for a0 < a1 && b0 < b1 && d.a[a1-1] == d.b[b1-1] {
		d.pair(a1-1, b1-1)
		a1, b1 = a1-1, b1-1
	}
	if a0 == a1 || b0 == b1 {
		return budget < 0 || (a1-a0)+(b1-b0) <= budget
	}
	limit := -1
	if budget >= 0 {
		limit = (budget + 1) / 2
	}
	x, y, u, v, dist, ok := d.middleSnake(d.a[a0:a1], d.b[b0:b1], limit)
	if !ok || budget >= 0 && dist > budget {
		return false
	}
	for i, j := x, y; i < u; i, j = i+1, j+1 {
		d.pair(a0+i, b0+j)
	}
	d.solve(a0, a0+x, b0, b0+y, -1)
	d.solve(a0+u, a1, b0+v, b1, -1)
	return true
}

// middleSnake finds the middle snake of the non-empty sequences a and b,
// which differ at both ends: the run of matches (x, y) to (u, v) that a
// shortest edit script, dist edits long, goes through, with as many edits
// before it as after it, give or take one. Both searches stop at limit
// rounds when limit >= 0, and ok is false if they have not met by then.
//
// The forward search is the paper's: round r extends, on every diagonal
// k = x - y of the right parity, the furthest point reachable with r edits.
// The backward one is the same on the sequences read from their ends, its
// diagonal c being the forward diagonal delta - c. Points outside the grid
// are never kept (none), so that they cannot hide a point inside it.
func (d *differ) middleSnake(a, b []int32, limit int) (x, y, u, v, dist int, ok bool) {
	n, m := len(a), len(b)
	delta := n - m
	odd := delta&1 != 0
	maxd := (n + m + 1) / 2
	if limit >= 0 && limit < maxd {
		maxd = limit
	}
	off, vf, vb := d.off, d.vf, d.vb
	// next is the furthest x on diagonal k after round r from the round
	// before's v; down is a step from diagonal k+1, right one from k-1.
	next := func(v []int, r, k int) int {
		if r == 0 {
			return 0
		}
		x := none
		if k != r { // diagonal k+1 was reached in the round before: insert b[y]
			if px := v[off+k+1]; px != none && px-k <= m {
				x = px
			}
		}
		if k != -r { // diagonal k-1: delete a[x]
			if px := v[off+k-1]; px != none && px+1 <= n && px+1 > x {
				x = px + 1
			}
		}
		return x
	}
	for r := 0; r <= maxd; r++ {
		for k := -r; k <= r; k += 2 {
			x := next(vf, r, k)
			if x == none {
				vf[off+k] = none
				continue
			}
			xs, y := x, x-k
			for x < n && y < m && a[x] == b[y] {
				x, y = x+1, y+1
			}
			vf[off+k] = x
			if odd {
				if c := delta - k; c >= -(r-1) && c <= r-1 {
					if xb := vb[off+c]; xb != none && x+xb >= n {
						return xs, xs - k, x, y, 2*r - 1, true
					}
				}
			}
		}
		for k := -r; k <= r; k += 2 {
			x := next(vb, r, k)
			if x == none {
				vb[off+k] = none
				continue
			}
			xs, y := x, x-k
			for x < n && y < m && a[n-1-x] == b[m-1-y] {
				x, y = x+1, y+1
			}
			vb[off+k] = x
			if !odd {
				if kf := delta - k; kf >= -r && kf <= r {
					if xf := vf[off+kf]; xf != none && xf+x >= n {
						return n - x, m - y, n - xs, m - (xs - k), 2 * r, true
					}
				}
			}
		}
	}
	return 0, 0, 0, 0, 0, false
}

// hunks cuts the blocks of change into hunks with context lines around
// them: blocks whose contexts touch (at most 2*context lines match between
// them) share a hunk.
func hunks(sa, sb text, blocks []block, context int) []Hunk {
	var out []Hunk
	for k := 0; k < len(blocks); {
		e := k
		for e+1 < len(blocks) && blocks[e+1].a0-blocks[e].a1 <= 2*context {
			e++
		}
		first, last := blocks[k], blocks[e]
		before := first.a0
		if k > 0 {
			before -= blocks[k-1].a1
		}
		lead := min(context, before)
		trail := min(context, len(sa.lines)-last.a1)

		ai, bi := first.a0-lead, first.b0-lead
		h := Hunk{OldStart: ai + 1, NewStart: bi + 1}
		h.Lines = make([]Line, 0, lead+trail+(last.a1-first.a0)+(last.b1-first.b0))
		same := func(end int) {
			for ; ai < end; ai, bi = ai+1, bi+1 {
				h.Lines = append(h.Lines, Line{Context, ai + 1, bi + 1, sa.at(ai)})
			}
		}
		for g := k; g <= e; g++ {
			blk := blocks[g]
			same(blk.a0)
			for ; ai < blk.a1; ai++ {
				h.Lines = append(h.Lines, Line{Removed, ai + 1, 0, sa.at(ai)})
			}
			for ; bi < blk.b1; bi++ {
				h.Lines = append(h.Lines, Line{Added, 0, bi + 1, sb.at(bi)})
			}
		}
		same(last.a1 + trail)

		for _, l := range h.Lines {
			switch l.Kind {
			case Context:
				h.OldLines, h.NewLines = h.OldLines+1, h.NewLines+1
			case Removed:
				h.OldLines++
			default:
				h.NewLines++
			}
		}
		if h.OldLines == 0 {
			h.OldStart--
		}
		if h.NewLines == 0 {
			h.NewStart--
		}
		out = append(out, h)
		k = e + 1
	}
	return out
}
