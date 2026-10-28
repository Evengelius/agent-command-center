package render

import (
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
)

// Plan 62, criterion 62.6's check against git (design D9): the lane layout on
// the histories recorded in testdata/graph from `git log --graph --topo-order
// --format="%H %P" --all` (TestGraphRecordings in internal/input makes them).
// The render package is pure: it reads the recordings and runs no git.

// graphRecNames are the recordings, one made history each.
var graphRecNames = []string{"line", "merge", "octopus", "crisscross", "roots", "lanes", "random"}

// graphRecLine is a commit line of a recording: git's drawing with the "*" of
// the commit in it, then the hash and the parents'. The other lines are git's
// rows in between.
var graphRecLine = regexp.MustCompile(`^[ |\\/_.*-]*\*[ |\\/_.*-]*([0-9a-f]{40})((?: [0-9a-f]{40})*) *$`)

// graphRec is a recording's commits in git's order, as the lanes take them.
type graphRec struct {
	hashes  [][20]byte
	parents [][][20]byte
}

// graphHash is a 40-digit hex hash as bytes.
func graphHash(t *testing.T, s string) [20]byte {
	t.Helper()
	var h [20]byte
	if b, err := hex.DecodeString(s); err != nil || len(b) != len(h) {
		t.Fatalf("%q is no hash: %v", s, err)
	} else {
		copy(h[:], b)
	}
	return h
}

// readGraphRec reads the recording name.
func readGraphRec(t *testing.T, name string) graphRec {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "..", "testdata", "graph", name+".txt"))
	if err != nil {
		t.Fatalf("recording %s: %v (go test ./internal/input -run TestGraphRecordings -update makes it)", name, err)
	}
	var rec graphRec
	for _, l := range strings.Split(string(b), "\n") {
		m := graphRecLine.FindStringSubmatch(l)
		if m == nil {
			continue
		}
		rec.hashes = append(rec.hashes, graphHash(t, m[1]))
		var ps [][20]byte
		for _, p := range strings.Fields(m[2]) {
			ps = append(ps, graphHash(t, p))
		}
		rec.parents = append(rec.parents, ps)
	}
	if len(rec.hashes) == 0 {
		t.Fatalf("recording %s holds no commit", name)
	}
	return rec
}

// layOut lays the recording out in git's order.
func (r graphRec) layOut() *GraphLanes {
	var g GraphLanes
	for i, h := range r.hashes {
		g.Add(h, r.parents[i])
	}
	return &g
}

// rows gives the row of each parent of commit i, and fails the test for a
// parent the recording does not hold or one that is not below its child.
func (r graphRec) rows(t *testing.T, name string, row map[[20]byte]int, i int) []int {
	t.Helper()
	var rows []int
	for _, p := range r.parents[i] {
		j, ok := row[p]
		if !ok {
			t.Fatalf("recording %s, row %d: the parent %x is not in the history", name, i, p)
		}
		if j <= i {
			t.Fatalf("recording %s, row %d: the parent %x is at row %d, not below", name, i, p, j)
		}
		rows = append(rows, j)
	}
	slices.Sort(rows)
	return rows
}

// TestLanesAgainstGit is criterion 62.6 on git's own histories (design D9):
// every commit's lines, followed as a reader does, reach the rows of its
// parents as git gives them and no other; with lanes hidden (design D10), a
// subset of them, never a commit that is not a parent, and "◆" exactly on the
// commits of the hidden lanes.
func TestLanesAgainstGit(t *testing.T) {
	for _, name := range graphRecNames {
		t.Run(name, func(t *testing.T) {
			rec := readGraphRec(t, name)
			row := map[[20]byte]int{}
			for i, h := range rec.hashes {
				if _, twice := row[h]; twice {
					t.Fatalf("recording %s: %x is twice in the history", name, h)
				}
				row[h] = i
			}
			g := rec.layOut()
			width := 0
			g.walk(0, g.Len(), func(_ int, r *laneRow) { width = max(width, r.width()) })
			t.Logf("%s: %d commits, %d lanes at most", name, len(rec.hashes), width)
			for _, show := range []int{1000, 3, 2} {
				fig := drawAll(g, show)
				for i := range rec.hashes {
					want := rec.rows(t, name, row, i)
					got := fig.trace(t, i)
					if show == 1000 && !slices.Equal(got, want) {
						t.Fatalf("recording %s, row %d %q: reaches rows %v, its git parents are rows %v", name, i, fig.text(i), got, want)
					}
					for _, j := range got {
						if !slices.Contains(want, j) {
							t.Fatalf("recording %s, %d lanes shown, row %d %q: a line to row %d, not a parent (git's: rows %v)", name, show, i, fig.text(i), j, want)
						}
					}
					if hidden := fig.dot[i] >= show; hidden != (fig.over[i] == '◆') {
						t.Fatalf("recording %s, %d lanes shown, row %d %q: in lane %d, overflow %q", name, show, i, fig.text(i), fig.dot[i], fig.over[i])
					}
				}
			}
		})
	}
}

// key is row i's figure with its colours, for comparing rows laid out at
// different times.
func (d laneFig) key(i int) string {
	var b strings.Builder
	for _, c := range d.cells[i] {
		if c.glyph != 0 {
			fmt.Fprintf(&b, "%c%d", c.glyph, c.colour)
		} else {
			b.WriteByte(' ')
		}
	}
	return fmt.Sprintf("%d %s %c", d.dot[i], strings.TrimRight(b.String(), " "), max(d.over[i], ' '))
}

// TestLanesPaged is design D9's "the rows laid never move": a history laid
// out in its first k commits, and then in the rest as the pages come, draws
// the first k rows as it did before, with their colours.
func TestLanesPaged(t *testing.T) {
	type commit struct {
		hash    [20]byte
		parents [][20]byte
	}
	var histories = map[string][]commit{}
	rec := readGraphRec(t, "random")
	for i, h := range rec.hashes {
		histories["random"] = append(histories["random"], commit{h, rec.parents[i]})
	}
	for i, ps := range randomDAG(5, 1000) {
		c := commit{hash: laneHash(i)}
		for _, p := range ps {
			c.parents = append(c.parents, laneHash(p))
		}
		histories["randomDAG(5, 1000)"] = append(histories["randomDAG(5, 1000)"], c)
	}
	for name, commits := range histories {
		for _, k := range []int{1, 63, 64, 65, 200} {
			var g GraphLanes
			for _, c := range commits[:k] {
				g.Add(c.hash, c.parents)
			}
			before := drawAll(&g, 1000)
			var was []string
			for i := range k {
				was = append(was, before.key(i))
			}
			for _, c := range commits[k:] {
				g.Add(c.hash, c.parents)
			}
			after := drawAll(&g, 1000)
			for i := range k {
				if now := after.key(i); now != was[i] {
					t.Fatalf("%s, %d commits laid out first: row %d was %q, then %q", name, k, i, was[i], now)
				}
			}
		}
	}
}
