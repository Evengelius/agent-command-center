package render

import (
	"fmt"
	"math/rand/v2"
	"slices"
	"strings"
	"testing"
)

// dag is a history for the lanes' tests, newest first: commit i's parents
// are commits after it, the first parent first.
type dag [][]int

// randomDAG is a history of n commits in a topological order, children
// before parents, from seed: mostly one parent close below, merges of two
// and now and then three, roots now and then and at the end; every commit no
// earlier one names a branch's tip.
func randomDAG(seed uint64, n int) dag {
	r := rand.New(rand.NewPCG(seed, 7))
	d := make(dag, n)
	for i := range n {
		if i == n-1 || r.IntN(60) == 0 {
			continue // a root
		}
		want := 1
		switch p := r.IntN(100); {
		case p < 3:
			want = 3
		case p < 18:
			want = 2
		}
		for len(d[i]) < want {
			j := i + 1 + r.IntN(min(12, n-1-i))
			if r.IntN(10) == 0 {
				j = i + 1 + r.IntN(n-1-i)
			}
			if !slices.Contains(d[i], j) {
				d[i] = append(d[i], j)
			} else if n-1-i < want {
				break
			}
		}
	}
	return d
}

// laneHash is commit i's made-up hash.
func laneHash(i int) [20]byte {
	var h [20]byte
	h[0], h[1], h[2] = byte(i>>16), byte(i>>8), byte(i)
	h[19] = 1
	return h
}

// layOut lays d out.
func layOut(d dag) *GraphLanes {
	var g GraphLanes
	for i, ps := range d {
		var hs [][20]byte
		for _, p := range ps {
			hs = append(hs, laneHash(p))
		}
		g.Add(laneHash(i), hs)
	}
	return &g
}

// laneFig is a history's figure as laneFig, each row's cells, its commit's lane
// and its overflow column's glyph.
type laneFig struct {
	cells [][]laneCell
	dot   []int
	over  []rune
}

// drawAll draws every row of g, show lanes shown.
func drawAll(g *GraphLanes, show int) laneFig {
	w := 0
	g.walk(0, g.Len(), func(_ int, r *laneRow) { w = max(w, r.width()) })
	var out laneFig
	g.walk(0, g.Len(), func(_ int, r *laneRow) {
		cells := make([]laneCell, 2*min(w, show))
		out.over = append(out.over, r.figure(cells, show))
		out.cells = append(out.cells, cells)
		out.dot = append(out.dot, r.c)
	})
	return out
}

// text is row i's glyphs, the overflow column's after them, the spaces at
// the end cut.
func (d laneFig) text(i int) string {
	var b strings.Builder
	for _, c := range d.cells[i] {
		b.WriteRune(max(c.glyph, ' '))
	}
	if d.over[i] != 0 {
		b.WriteRune(d.over[i])
	}
	return strings.TrimRight(b.String(), " ")
}

// connects is which sides glyph g joins: up, down, left, right.
func connects(g rune) (up, down, left, right bool) {
	switch g {
	case '│':
		return true, true, false, false
	case '─':
		return false, false, true, true
	case '╯':
		return true, false, true, false
	case '╰':
		return true, false, false, true
	case '╮':
		return false, true, true, false
	case '╭':
		return false, true, false, true
	case '┴':
		return true, false, true, true
	case '┬':
		return false, true, true, true
	case '╵':
		return true, false, false, false
	case '╷':
		return false, true, false, false
	}
	return false, false, false, false
}

// trace follows the figure's lines from row i's dot as a reader does, never
// upwards, a line going on behind a lane it passes ("│" between two "─"),
// and gives the commits whose dots it reaches; a line that leads nowhere, or
// to a dot sideways off its curve, is an error.
func (d laneFig) trace(t *testing.T, i int) []int {
	t.Helper()
	type at struct{ row, x, dx int } // dx 0: moving down into (row, x)
	var reached []int
	seen := map[at]bool{}
	x0 := 2 * d.dot[i]
	todo := []at{{i + 1, x0, 0}, {i, x0 + 1, 1}, {i, x0 - 1, -1}}
	for len(todo) > 0 {
		p := todo[len(todo)-1]
		todo = todo[:len(todo)-1]
		if seen[p] || p.row >= len(d.cells) || p.x < 0 || p.x >= len(d.cells[p.row]) {
			continue
		}
		seen[p] = true
		g := d.cells[p.row][p.x].glyph
		if (g == '●' || g == '◉') && p.x == 2*d.dot[p.row] {
			if p.row == i {
				t.Errorf("row %d: a line comes back to its own dot", i)
			} else {
				reached = append(reached, p.row)
			}
			continue
		}
		up, down, left, right := connects(g)
		if p.dx == 0 {
			// Coming down: the glyph must take the line from above.
			if !up {
				if p.row == i+1 && p.x == x0 {
					continue // the dot's lane does not go on: no first parent
				}
				t.Errorf("row %d: the line from row %d down cell %d meets %q", i, p.row-1, p.x, g)
				continue
			}
			if down {
				todo = append(todo, at{p.row + 1, p.x, 0})
			}
			if left {
				todo = append(todo, at{p.row, p.x - 1, -1})
			}
			if right {
				todo = append(todo, at{p.row, p.x + 1, 1})
			}
			continue
		}
		// Moving sideways: a curve's cells, or a lane passed behind.
		if p.row == i && (p.x == x0+p.dx) && g == 0 {
			continue // no curve on that side of the dot
		}
		if g == '│' && p.x%2 == 0 {
			todo = append(todo, at{p.row, p.x + p.dx, p.dx})
			continue
		}
		from := left
		if p.dx < 0 {
			from = right
		}
		if !from {
			t.Errorf("row %d: the curve on row %d meets %q at cell %d", i, p.row, g, p.x)
			continue
		}
		if down {
			todo = append(todo, at{p.row + 1, p.x, 0})
		}
		if p.dx > 0 && right || p.dx < 0 && left {
			todo = append(todo, at{p.row, p.x + p.dx, p.dx})
		}
	}
	slices.Sort(reached)
	return slices.Compact(reached)
}

// TestLanesTopology is criterion 62.6's topology (design D9): on random
// histories of branches, merges of two and three parents and roots, each
// commit's lines, followed as laneFig, reach its parents and no other commit;
// with lanes hidden (design D10), never a commit that is not a parent, the
// rows of the commits in hidden lanes marked "◆".
func TestLanesTopology(t *testing.T) {
	for seed := range uint64(40) {
		d := randomDAG(seed, 300)
		g := layOut(d)
		for _, show := range []int{1000, 3, 2} {
			fig := drawAll(g, show)
			for i, ps := range d {
				want := slices.Sorted(slices.Values(ps))
				got := fig.trace(t, i)
				if show == 1000 && !slices.Equal(got, want) {
					t.Fatalf("seed %d, row %d %q: reaches %v, its parents %v", seed, i, fig.text(i), got, want)
				}
				for _, j := range got {
					if !slices.Contains(ps, j) {
						t.Fatalf("seed %d, %d lanes shown, row %d %q: a line to %d, not a parent of %v", seed, show, i, fig.text(i), j, ps)
					}
				}
				if hidden := fig.dot[i] >= show; hidden != (fig.over[i] == '◆') {
					t.Fatalf("seed %d, %d lanes shown, row %d %q: in lane %d, overflow %q", seed, show, i, fig.text(i), fig.dot[i], fig.over[i])
				}
			}
		}
	}
}

// TestLanesFigures is design D9 on small histories, each row's figure as
// laneFig: a branch merged back, the merge's ring and the branch's curve to
// its fork; two branches ending on one commit; three parents; a merge that
// is also two branches' parent, the ending lanes at its left and the new
// one at its right; a new branch kept from under a root's dot; a merge's
// new lane passing nothing.
func TestLanesFigures(t *testing.T) {
	for _, c := range []struct {
		name string
		d    dag
		want []string
	}{
		{"merged branch", dag{{3, 1}, {2}, {4}, {4}, {5}, {}}, []string{"◉─╮", "│ ●", "│ ●", "● │", "●─╯", "●"}},
		{"two branches end", dag{{3}, {3}, {3}, {}}, []string{"●", "│ ●", "│ │ ●", "●─┴─╯"}},
		{"three parents", dag{{1, 2, 3}, {4}, {4}, {4}, {}}, []string{"◉─┬─╮", "● │ │", "│ ● │", "│ │ ●", "●─┴─╯"}},
		{"merge and fork", dag{{2}, {2}, {3, 4}, {5}, {5}, {}}, []string{"●", "│ ●", "╰─◉─╮", "  ● │", "  │ ●", "  ●─╯"}},
		{"no dot under a root", dag{{2}, {2}, {3}, {}, {5}, {}}, []string{"●", "│ ●", "●─╯", "●", "  ●", "  ●"}},
		{"a lane passed", dag{{1, 3}, {2}, {4}, {4}, {}}, []string{"◉─╮", "● │", "● │", "│ ●", "●─╯"}},
	} {
		g := layOut(c.d)
		fig := drawAll(g, 1000)
		var got []string
		for i := range c.d {
			got = append(got, fig.text(i))
		}
		if !slices.Equal(got, c.want) {
			t.Errorf("%s:\n%s\nwant\n%s", c.name, strings.Join(got, "\n"), strings.Join(c.want, "\n"))
		}
	}
}

// TestLanesCrossing is a curve passing a lane (design D9): the lane keeps
// its "│", the curve its "─" on both sides, and the lane's line still goes
// straight down.
func TestLanesCrossing(t *testing.T) {
	// 0 and 1 are tips; 2, on 0's line, merges 4, its new lane right of
	// 1's, passing it.
	d := dag{{2}, {3}, {5, 4}, {5}, {5}, {}}
	fig := drawAll(layOut(d), 1000)
	if got := fig.text(2); got != "◉─│─╮" {
		t.Errorf("the merge's row %q, want %q", got, "◉─│─╮")
	}
	for i, ps := range d {
		if got := fig.trace(t, i); !slices.Equal(got, slices.Sorted(slices.Values(ps))) {
			t.Errorf("row %d reaches %v, its parents %v", i, got, ps)
		}
	}
}

// TestLanesColours is design D9's colours: a lane keeps its colour down its
// line, a new lane takes the next of the eight, and a curve takes its lane's.
func TestLanesColours(t *testing.T) {
	fig := drawAll(layOut(dag{{3, 1}, {2}, {4}, {4}, {5}, {}}), 1000)
	if c := fig.cells; c[0][0].colour != 0 || c[0][1].colour != 1 || c[0][2].colour != 1 || c[3][0].colour != 0 || c[1][2].colour != 1 || c[4][2].colour != 1 {
		t.Errorf("colours %v", c)
	}
	var many dag
	for i := range 20 {
		many = append(many, []int{20}) // twenty tips on one root
		_ = i
	}
	many = append(many, nil)
	g := layOut(many)
	seen := map[uint8]int{}
	g.walk(0, 20, func(_ int, r *laneRow) { seen[r.colour]++ })
	if len(seen) != laneColours {
		t.Errorf("twenty lanes took %d colours, want %d", len(seen), laneColours)
	}
}

// TestLanesWalk is design D15's bound: a row laneFig from the lanes kept
// whole before it is the row laneFig from the start, at every distance from
// them.
func TestLanesWalk(t *testing.T) {
	g := layOut(randomDAG(3, 1000))
	var all []string
	g.walk(0, g.Len(), func(_ int, r *laneRow) {
		all = append(all, fmt.Sprint(r.c, r.flags, r.colour, r.ending, r.starting, r.before, r.after))
	})
	for _, from := range []int{0, 1, 63, 64, 65, 500, 999} {
		g.walk(from, from+3, func(i int, r *laneRow) {
			if got := fmt.Sprint(r.c, r.flags, r.colour, r.ending, r.starting, r.before, r.after); got != all[i] {
				t.Errorf("row %d from %d: %s, want %s", i, from, got, all[i])
			}
		})
	}
}
