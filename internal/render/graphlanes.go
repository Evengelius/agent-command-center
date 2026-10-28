package render

import (
	"encoding/binary"
	"slices"
)

// The Git Graph's lanes (plan 62, designs D9 and D10; ui-design.md, Git
// Graph): each commit in a lane, laid out from its parents in the order git
// gives them, children before parents, by a small running state as each
// page of the history comes; the rows laid never move. A commit's row draws
// its dot in its lane, the lanes passing it, the curves from the lanes of
// its children that end on it and those to the new lanes of a merge's other
// parents, in that one row: the figure of `git log --graph` without its
// rows in between.
//
// No line ever joins two commits git does not join: the curves of the
// lanes ending on a commit and those of its new lanes are on either side of
// its dot, so that no stretch of a row carries both; a curve passing a lane
// leaves that lane's line whole, "│", never a crossing's "┼"; a merge's other
// parents always get lanes of their own, joining no other until their
// commit; a new branch never takes the lane a root ended on the row above,
// whose dot over its own would read as its child.

// laneCheck is how many rows apart the lanes are kept whole: a row is drawn
// from the nearest such row before it.
const laneCheck = 64

// laneFree is a free lane's colour.
const laneFree = 0xFF

// laneColours is how many colours the lanes take in turn.
const laneColours = 8

// A row's flags.
const (
	laneTip   = 1 << iota // the commit starts its lane: no child of it read
	laneRoot              // it has no parent: its lane ends on it
	laneMerge             // it has two parents or more
)

// GraphLanes is the history's lanes as its commits come (design D9).
type GraphLanes struct {
	// The running state: the lanes waiting for each commit, each lane's
	// colour, laneFree when the lane is free, the next new lane's colour.
	wait   map[[20]byte][]int
	colour []uint8
	next   uint8
	// rooted is the lane of the row before, plus one, when its commit was a
	// root; 0 otherwise.
	rooted int
	// The rows: commit i's events from at[i] in events; snaps[k] the lanes'
	// colours before row k*laneCheck.
	at     []uint32
	events []byte
	snaps  [][]uint8
}

// Len is how many commits are laid out.
func (g *GraphLanes) Len() int {
	if g == nil {
		return 0
	}
	return len(g.at)
}

// Add lays out the next commit in git's order: its hash and its parents',
// the first parent first.
func (g *GraphLanes) Add(hash [20]byte, parents [][20]byte) {
	if g.wait == nil {
		g.wait = map[[20]byte][]int{}
	}
	if len(g.at)%laneCheck == 0 {
		g.snaps = append(g.snaps, slices.Clone(g.colour))
	}
	g.at = append(g.at, uint32(len(g.events)))
	waiting := g.wait[hash]
	delete(g.wait, hash)
	var flags byte
	if len(parents) > 1 {
		flags |= laneMerge
	}
	if len(parents) == 0 {
		flags |= laneRoot
	}
	var c int
	switch {
	case len(waiting) == 0:
		flags |= laneTip
		c = g.tipLane()
	case len(waiting) > 1 && len(parents) > 1:
		// The ending lanes at the left of the dot, the new ones at its right.
		c = slices.Max(waiting)
	default:
		c = slices.Min(waiting)
	}
	g.events = binary.AppendUvarint(g.events, uint64(c))
	g.events = append(g.events, flags)
	if flags&laneTip != 0 {
		g.colour[c] = g.newColour()
		g.events = append(g.events, g.colour[c])
	}
	g.events = binary.AppendUvarint(g.events, uint64(len(waiting)-min(len(waiting), 1)))
	for _, j := range waiting {
		if j != c {
			g.events = binary.AppendUvarint(g.events, uint64(j))
		}
	}
	// The other parents' lanes, before the ending ones are freed: a lane
	// ending on this row is no lane for another parent on it.
	g.events = binary.AppendUvarint(g.events, uint64(max(len(parents)-1, 0)))
	for _, p := range parents[min(len(parents), 1):] {
		k := g.nearLane(c, len(waiting) > 1)
		g.colour[k] = g.newColour()
		g.wait[p] = append(g.wait[p], k)
		g.events = binary.AppendUvarint(g.events, uint64(k))
		g.events = append(g.events, g.colour[k])
	}
	for _, j := range waiting {
		if j != c {
			g.colour[j] = laneFree
		}
	}
	g.rooted = 0
	if len(parents) == 0 {
		g.colour[c] = laneFree
		g.rooted = c + 1
	} else {
		g.wait[parents[0]] = append(g.wait[parents[0]], c)
	}
	for len(g.colour) > 0 && g.colour[len(g.colour)-1] == laneFree {
		g.colour = g.colour[:len(g.colour)-1]
	}
}

// tipLane is a new branch's lane: the leftmost free one, but the lane a
// root ended on the row above.
func (g *GraphLanes) tipLane() int {
	for i, c := range g.colour {
		if c == laneFree && i != g.rooted-1 {
			return i
		}
	}
	k := len(g.colour)
	if k == g.rooted-1 {
		k++
	}
	for len(g.colour) <= k {
		g.colour = append(g.colour, laneFree)
	}
	return k
}

// nearLane is a merge's other parent's lane: the free lane nearest to the
// dot's lane c, the right one on a tie, or a new one at the right; right
// of c only when lanes end on the row, at c's left.
func (g *GraphLanes) nearLane(c int, right bool) int {
	for d := 1; ; d++ {
		if k := c + d; k >= len(g.colour) {
			g.colour = append(g.colour, slices.Repeat([]uint8{laneFree}, k+1-len(g.colour))...)
			return k
		} else if g.colour[k] == laneFree {
			return k
		}
		if k := c - d; !right && k >= 0 && g.colour[k] == laneFree {
			return k
		}
	}
}

// newColour is the next new lane's colour.
func (g *GraphLanes) newColour() uint8 {
	c := g.next
	g.next = (g.next + 1) % laneColours
	return c
}

// laneEnd is a lane a row's curve joins to the dot, and its colour.
type laneEnd struct {
	lane   int
	colour uint8
}

// laneRow is one commit's row of lanes, as walk gives it.
type laneRow struct {
	// c is the commit's lane and colour its colour.
	c      int
	colour uint8
	flags  byte
	// ending are its children's lanes that end on it, others than its own;
	// starting the new lanes of its other parents.
	ending, starting []laneEnd
	// before are the lanes coming down to the row and after those going on
	// below it: each lane's colour, laneFree when free.
	before, after []uint8
	// curves is figure's room for the row's curves.
	curves []laneCurve
}

// laneCurve is a curve of a row: to a lane ending on the dot, or down to a
// new lane.
type laneCurve struct {
	laneEnd
	down bool
}

// width is how many lanes the row's figure takes.
func (r *laneRow) width() int {
	w := max(len(r.before), len(r.after), r.c+1)
	for _, e := range r.starting {
		w = max(w, e.lane+1)
	}
	return w
}

// walk calls fn with rows from to to-1 in turn, each drawn from the lanes
// kept whole before it; fn must not keep the row's slices.
func (g *GraphLanes) walk(from, to int, fn func(i int, r *laneRow)) {
	to = min(to, g.Len())
	if from < 0 || from >= to {
		return
	}
	k := from / laneCheck
	lanes := slices.Clone(g.snaps[k])
	var r laneRow
	for i := k * laneCheck; i < to; i++ {
		lanes = g.row(i, lanes, &r)
		if i >= from {
			fn(i, &r)
		}
	}
}

// row reads row i's events into r, lanes the lanes before it, and gives the
// lanes after it.
func (g *GraphLanes) row(i int, lanes []uint8, r *laneRow) []uint8 {
	b := g.events[g.at[i]:]
	next := func() int {
		v, n := binary.Uvarint(b)
		b = b[n:]
		return int(v)
	}
	r.before = append(r.before[:0], lanes...)
	r.c = next()
	r.flags, b = b[0], b[1:]
	if r.flags&laneTip != 0 {
		r.colour, b = b[0], b[1:]
	} else {
		r.colour = lanes[r.c]
	}
	r.ending = r.ending[:0]
	for n := next(); n > 0; n-- {
		j := next()
		r.ending = append(r.ending, laneEnd{j, lanes[j]})
		lanes[j] = laneFree
	}
	grow := func(k int) {
		for len(lanes) <= k {
			lanes = append(lanes, laneFree)
		}
	}
	grow(r.c)
	lanes[r.c] = r.colour
	if r.flags&laneRoot != 0 {
		lanes[r.c] = laneFree
	}
	r.starting = r.starting[:0]
	for n := next(); n > 0; n-- {
		k := next()
		c := b[0]
		b = b[1:]
		r.starting = append(r.starting, laneEnd{k, c})
		grow(k)
		lanes[k] = c
	}
	for len(lanes) > 0 && lanes[len(lanes)-1] == laneFree {
		lanes = lanes[:len(lanes)-1]
	}
	r.after = append(r.after[:0], lanes...)
	return lanes
}

// laneCell is one cell of a row's figure: its glyph, 0 for a space, and its
// lane's colour.
type laneCell struct {
	glyph  rune
	colour uint8
}

// figure draws r's figure into cells, two a lane, the lanes from show on
// hidden (design D10): no curve drawn to or from a hidden lane, a lane
// ending on a hidden commit "╵", one starting from it "╷". It gives what the
// overflow column shows: "◆" the commit hidden, "┆" hidden lanes passing,
// 0 nothing. The dot is "●", a merge's "◉".
func (r *laneRow) figure(cells []laneCell, show int) rune {
	clear(cells)
	n := len(cells) / 2
	put := func(lane int, glyph rune, colour uint8) {
		if lane < n {
			cells[2*lane] = laneCell{glyph, colour}
		}
	}
	ends := func(lane int) bool {
		return lane == r.c && r.flags&laneTip == 0 || slices.ContainsFunc(r.ending, func(e laneEnd) bool { return e.lane == lane })
	}
	var over rune
	for lane, c := range r.before {
		if c == laneFree || ends(lane) {
			continue
		}
		if lane >= show {
			over = '┆'
			continue
		}
		put(lane, '│', c)
	}
	if r.c >= show {
		for _, e := range r.ending {
			if e.lane < show {
				put(e.lane, '╵', e.colour)
			}
		}
		for _, e := range r.starting {
			if e.lane < show {
				put(e.lane, '╷', e.colour)
			}
		}
		return '◆'
	}
	dot := '●'
	if r.flags&laneMerge != 0 {
		dot = '◉'
	}
	put(r.c, dot, r.colour)
	// The nearest curves first: a stretch two of them pass takes the
	// nearer one's colour.
	r.curves = r.curves[:0]
	for _, e := range r.ending {
		r.curves = append(r.curves, laneCurve{e, false})
	}
	for _, e := range r.starting {
		r.curves = append(r.curves, laneCurve{e, true})
	}
	slices.SortFunc(r.curves, func(a, b laneCurve) int { return abs(a.lane-r.c) - abs(b.lane-r.c) })
	for i, e := range r.curves {
		if e.lane >= show {
			over = '┆'
			continue
		}
		side := sign(e.lane - r.c)
		beyond := slices.ContainsFunc(r.curves[i+1:], func(o laneCurve) bool { return o.lane < show && sign(o.lane-r.c) == side })
		r.curve(cells, e, beyond)
	}
	return over
}

// abs is |v|.
func abs(v int) int { return max(v, -v) }

// sign is v's sign, -1, 0 or 1.
func sign(v int) int { return min(max(v, -1), 1) }

// curve draws curve e between the dot and its lane, its glyph there: from
// above, "╯" right of the dot, "╰" left of it; downwards, "╮" and "╭"; "┴"
// and "┬" where another curve goes on beyond it, on the same side, of the
// same kind. The cells it passes take "─" in its colour, unless a nearer
// curve took them; a lane passing them keeps its "│".
func (r *laneRow) curve(cells []laneCell, e laneCurve, beyond bool) {
	n := len(cells) / 2
	step := sign(e.lane - r.c)
	for x := 2*r.c + step; x != 2*e.lane && x >= 0 && x < 2*n; x += step {
		if cells[x].glyph == 0 {
			cells[x] = laneCell{'─', e.colour}
		}
	}
	down := e.down
	var glyph rune
	switch {
	case beyond && down:
		glyph = '┬'
	case beyond:
		glyph = '┴'
	case down && step > 0:
		glyph = '╮'
	case down:
		glyph = '╭'
	case step > 0:
		glyph = '╯'
	default:
		glyph = '╰'
	}
	if e.lane < n {
		cells[2*e.lane] = laneCell{glyph, e.colour}
	}
}

// below draws the lanes going on under r into cells, as the rows of its
// files and the notes under it show them, and gives the overflow column's
// "┆" when hidden lanes go on.
func (r *laneRow) below(cells []laneCell, show int) rune {
	clear(cells)
	var over rune
	for lane, c := range r.after {
		switch {
		case c == laneFree:
		case lane >= show:
			over = '┆'
		case 2*lane < len(cells):
			cells[2*lane] = laneCell{'│', c}
		}
	}
	return over
}
