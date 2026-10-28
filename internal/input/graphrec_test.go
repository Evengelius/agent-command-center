package input

import (
	"bytes"
	"flag"
	"fmt"
	"math/rand/v2"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
)

// Plan 62, criterion 62.6's check against git (design D9): seven made
// histories, each recorded as `git log --graph --topo-order` prints it in
// testdata/graph, for the lane layout's test in internal/render to follow;
// the render package is pure and runs no git, so the recordings stand between
// them. -update writes them again from freshly made repositories.

var updateGraph = flag.Bool("update", false, "rewrite testdata/graph from made repositories")

// graphRecDir is where the recordings are, from this package's folder.
const graphRecDir = "../../testdata/graph"

// graphRecCommit is a commit line of a recording: its hash and its parents'.
type graphRecCommit struct {
	hash    string
	parents []string
}

// graphRecLine is a commit line of `git log --graph --format="%H %P"`: git's
// drawing, the "*" of the commit in it, then the hash and the parents'.
var graphRecLine = regexp.MustCompile(`^[ |\\/_.*-]*\*[ |\\/_.*-]*([0-9a-f]{40})((?: [0-9a-f]{40})*) *$`)

// graphRecParse gives the commits of a recording in its order, ignoring the
// rows git draws between them, which a newer git may draw differently.
func graphRecParse(text string) []graphRecCommit {
	var out []graphRecCommit
	for _, l := range lines(text) {
		if m := graphRecLine.FindStringSubmatch(l); m != nil {
			out = append(out, graphRecCommit{m[1], strings.Fields(m[2])})
		}
	}
	return out
}

// graphRecHistories are the made histories, by the recording's name.
var graphRecHistories = []struct {
	name  string
	build func(h *graphHist)
}{
	{"line", graphRecLineHist},
	{"merge", graphRecMergeHist},
	{"octopus", graphRecOctopusHist},
	{"crisscross", graphRecCrissCrossHist},
	{"roots", graphRecRootsHist},
	{"lanes", graphRecLanesHist},
	{"random", graphRecRandomHist},
}

// TestGraphRecordings is criterion 62.6's recordings (design D9): each made
// history's `git log --graph --topo-order --format="%H %P" --all` is what
// testdata/graph holds, as far as the commit lines go: their order, hashes
// and parents. With -update the whole output is written, git's drawing too,
// the spaces at its lines' ends cut.
func TestGraphRecordings(t *testing.T) {
	for _, hist := range graphRecHistories {
		t.Run(hist.name, func(t *testing.T) {
			dir := srcRepo(t)
			var h graphHist
			hist.build(&h)
			h.run(t, dir)
			out, err := exec.Command("git", "-C", dir, "log", "--graph", "--topo-order", "--format=%H %P", "--all").Output()
			if err != nil {
				t.Fatalf("git log: %v", err)
			}
			// Each line without the spaces git leaves at its end, which git's
			// own whitespace check would flag in the recordings.
			ls := strings.Split(strings.ReplaceAll(string(out), "\r\n", "\n"), "\n")
			for i, l := range ls {
				ls[i] = strings.TrimRight(l, " ")
			}
			got := strings.Join(ls, "\n")
			file := filepath.Join(graphRecDir, hist.name+".txt")
			if *updateGraph {
				if err := os.MkdirAll(graphRecDir, 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(file, []byte(got), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			recorded, err := os.ReadFile(file)
			if err != nil {
				t.Fatalf("%v: make the recordings with go test ./internal/input -run TestGraphRecordings -update", err)
			}
			if bytes.Contains(recorded, []byte("\r")) {
				t.Errorf("%s holds a CR: LF line endings only", file)
			}
			want, now := graphRecParse(string(recorded)), graphRecParse(got)
			if len(want) != h.marks {
				t.Errorf("%s holds %d commits, the history made %d: -update", hist.name, len(want), h.marks)
			}
			for i := range max(len(want), len(now)) {
				if i >= len(want) || i >= len(now) || want[i].hash != now[i].hash || !slices.Equal(want[i].parents, now[i].parents) {
					t.Fatalf("%s: git log's commit %d is %v, the recording's %v (%d and %d commits): -update after a change of the history made",
						hist.name, i, graphRecAt(now, i), graphRecAt(want, i), len(now), len(want))
				}
			}
		})
	}
}

// graphRecAt describes commit i of commits, or says there is none.
func graphRecAt(commits []graphRecCommit, i int) string {
	if i >= len(commits) {
		return "(none)"
	}
	return fmt.Sprintf("%s %v", commits[i].hash, commits[i].parents)
}

// graphRecLineHist: 5 commits on main.
func graphRecLineHist(h *graphHist) {
	prev := h.commit("refs/heads/main", "Ana", "line 1")
	for i := 2; i <= 5; i++ {
		prev = h.commit("refs/heads/main", "Ana", fmt.Sprintf("line %d", i), prev)
	}
}

// graphRecMergeHist: main with commits before and after, a branch of 2
// commits merged back into it.
func graphRecMergeHist(h *graphHist) {
	m1 := h.commit("refs/heads/main", "Ana", "main 1")
	m2 := h.commit("refs/heads/main", "Ana", "main 2", m1)
	f1 := h.commit("refs/heads/feature", "Bo", "feature 1", m2)
	f2 := h.commit("refs/heads/feature", "Bo", "feature 2", f1)
	m3 := h.commit("refs/heads/main", "Ana", "main 3", m2)
	m4 := h.commit("refs/heads/main", "Ana", "merge feature", m3, f2)
	h.commit("refs/heads/main", "Ana", "main 5", m4)
}

// graphRecOctopusHist: one merge of three parents, main and two branches.
func graphRecOctopusHist(h *graphHist) {
	base := h.commit("refs/heads/main", "Ana", "base")
	a1 := h.commit("refs/heads/alpha", "Bo", "alpha 1", base)
	b1 := h.commit("refs/heads/beta", "Cy", "beta 1", base)
	m1 := h.commit("refs/heads/main", "Ana", "main 1", base)
	m2 := h.commit("refs/heads/main", "Ana", "octopus", m1, a1, b1)
	h.commit("refs/heads/main", "Ana", "main 3", m2)
}

// graphRecCrissCrossHist: two branches each merging the other, two merges
// whose two parents are the same pair in swapped order, then both merged.
func graphRecCrissCrossHist(h *graphHist) {
	base := h.commit("refs/heads/main", "Ana", "base")
	x1 := h.commit("refs/heads/x", "Bo", "x 1", base)
	y1 := h.commit("refs/heads/y", "Cy", "y 1", base)
	x2 := h.commit("refs/heads/x", "Bo", "x merges y", x1, y1)
	y2 := h.commit("refs/heads/y", "Cy", "y merges x", y1, x1)
	m := h.commit("refs/heads/main", "Ana", "both", x2, y2)
	h.commit("refs/heads/main", "Ana", "main after", m)
}

// graphRecRootsHist: two roots, an unrelated history on another ref merged
// into main.
func graphRecRootsHist(h *graphHist) {
	a1 := h.commit("refs/heads/main", "Ana", "main 1")
	a2 := h.commit("refs/heads/main", "Ana", "main 2", a1)
	r1 := h.commit("refs/heads/other", "Di", "other root")
	r2 := h.commit("refs/heads/other", "Di", "other 2", r1)
	a3 := h.commit("refs/heads/main", "Ana", "main 3", a2)
	m := h.commit("refs/heads/main", "Ana", "merge other", a3, r2)
	h.commit("refs/heads/main", "Ana", "main 5", m)
}

// graphRecLanesHist: 12 branches off one base commit, 2 commits each, all
// unmerged but 4 merged back into main: more lanes than a 31-column panel
// shows.
func graphRecLanesHist(h *graphHist) {
	base := h.commit("refs/heads/main", "Ana", "base")
	main := base
	for b := range 12 {
		ref := fmt.Sprintf("refs/heads/b%02d", b)
		c1 := h.commit(ref, "Bo", fmt.Sprintf("b%02d 1", b), base)
		c2 := h.commit(ref, "Bo", fmt.Sprintf("b%02d 2", b), c1)
		if b%3 == 0 {
			main = h.commit("refs/heads/main", "Ana", fmt.Sprintf("merge b%02d", b), main, c2)
		}
	}
}

// graphRecRandomHist: 300 commits from a fixed seed, with branches, merges of
// two parents and some of three, two roots and several tips. Every commit is
// the tip of its branch when made, and later commits of the branch have it for
// first parent, so that all are reachable from the refs.
func graphRecRandomHist(h *graphHist) {
	r := rand.New(rand.NewPCG(62, 9))
	type branch struct {
		ref string
		tip int
	}
	var all []int
	branches := []*branch{{"refs/heads/main", h.commit("refs/heads/main", "Ana", "root")}}
	all = append(all, branches[0].tip)
	roots := 1
	authors := []string{"Ana", "Bo", "Cy", "Di"}
	for h.marks < 300 {
		subject := fmt.Sprintf("commit %d", h.marks+1)
		author := authors[r.IntN(len(authors))]
		var mark int
		switch p := r.IntN(100); {
		case p < 2 && roots < 3:
			roots++
			b := &branch{ref: fmt.Sprintf("refs/heads/r%d", len(branches))}
			b.tip = h.commit(b.ref, author, subject)
			branches = append(branches, b)
			mark = b.tip
		case p < 10 && len(branches) < 10:
			b := &branch{ref: fmt.Sprintf("refs/heads/b%d", len(branches))}
			b.tip = h.commit(b.ref, author, subject, all[r.IntN(len(all))])
			branches = append(branches, b)
			mark = b.tip
		case p < 35 && len(branches) > 1:
			b := branches[r.IntN(len(branches))]
			parents := []int{b.tip}
			want := 2
			if r.IntN(5) == 0 && len(all) > 2 {
				want = 3
			}
			for len(parents) < want {
				o := all[r.IntN(len(all))]
				if r.IntN(10) < 7 {
					o = branches[r.IntN(len(branches))].tip
				}
				if !slices.Contains(parents, o) {
					parents = append(parents, o)
				}
			}
			b.tip = h.commit(b.ref, author, subject, parents...)
			mark = b.tip
		default:
			b := branches[r.IntN(len(branches))]
			b.tip = h.commit(b.ref, author, subject, b.tip)
			mark = b.tip
		}
		all = append(all, mark)
	}
}
