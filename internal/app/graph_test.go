package app

import (
	"errors"
	"fmt"
	"runtime"
	"slices"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"
	"unsafe"

	tea "charm.land/bubbletea/v2"

	"github.com/Evengelius/agent-command-center/internal/input"
	"github.com/Evengelius/agent-command-center/internal/render"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// Plan 62 (brief amendment 64): the Git Graph through the application, git
// answered by a test double; the reads on real repositories are the input
// package's (graph_test.go there).

// fakeGraph is the Git Graph's git over fakeGit: the references, a history
// whose commits are listed newest first in a valid topological order, each
// commit's files and each file's sides. It records every read.
type fakeGraph struct {
	*fakeGit
	refs     input.GraphRefs
	refsErr  error
	history  []input.GraphCommit
	files    map[string]input.CommitFiles
	sides    map[string]input.DiffPair
	messages map[string]string // a commit's whole message, its subject when none
	logs     [][]string        // the tips of each GraphLog
	readers  []*fakeReader
	// gate, when set, holds the next GraphLog until it is closed; dup makes
	// each page after the first start with the page before's last commit.
	gate chan struct{}
	dup  bool
}

// fakeReader reads a fakeGraph's history reachable from its tips.
type fakeReader struct {
	mu      sync.Mutex
	commits []input.GraphCommit
	at      int
	closed  bool
}

func (r *fakeReader) Next(n int) ([]input.GraphCommit, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return nil, false, errors.New("the history's read was closed")
	}
	end := min(r.at+n, len(r.commits))
	out := slices.Clone(r.commits[r.at:end])
	for i := range out {
		out[i].Subject = strings.Clone(out[i].Subject) // fresh, as git's pipe gives them
	}
	r.at = end
	return out, r.at == len(r.commits), nil
}

func (r *fakeReader) Close() {
	r.mu.Lock()
	r.closed = true
	r.mu.Unlock()
}

func (r *fakeReader) isClosed() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.closed
}

func (f *fakeGraph) GraphRefs(dir string) (input.GraphRefs, error) {
	f.record("refs")
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.refs, f.refsErr
}

func (f *fakeGraph) GraphLog(dir string, tips []string) (input.GraphReader, error) {
	f.record("log " + strings.Join(short(tips), " "))
	f.mu.Lock()
	gate := f.gate
	f.gate = nil
	f.mu.Unlock()
	if gate != nil {
		<-gate
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.logs = append(f.logs, slices.Clone(tips))
	reach := map[string]bool{}
	for _, t := range tips {
		reach[t] = true
	}
	var commits []input.GraphCommit
	for _, c := range f.history { // children before parents: the list's order
		if reach[c.Hash] {
			commits = append(commits, c)
			for _, p := range c.Parents {
				reach[p] = true
			}
		}
	}
	if f.dup {
		for i := graphPage; i < len(commits); i += graphPage + 1 {
			commits = slices.Insert(commits, i, commits[i-1])
		}
	}
	r := &fakeReader{commits: commits}
	f.readers = append(f.readers, r)
	return r, nil
}

func (f *fakeGraph) CommitFiles(dir, commit, parent string) (input.CommitFiles, error) {
	f.record("files " + commit[:7] + " " + shortOf(parent))
	f.mu.Lock()
	defer f.mu.Unlock()
	fs := f.files[commit]
	if fs.Total == 0 {
		fs.Total = len(fs.Files)
	}
	return fs, nil
}

func (f *fakeGraph) CommitMessage(dir, commit string) (string, error) {
	f.record("message " + commit[:7])
	f.mu.Lock()
	defer f.mu.Unlock()
	if msg, ok := f.messages[commit]; ok {
		return msg, nil
	}
	for _, c := range f.history {
		if c.Hash == commit {
			return c.Subject, nil
		}
	}
	return "", errors.New("no such commit")
}

func (f *fakeGraph) CommitSides(dir, commit, parent string, file input.CommitFile) (input.DiffPair, error) {
	f.record("sides " + commit[:7] + " " + shortOf(parent) + " " + file.Path)
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.sides[file.Path], nil
}

// short is each hash's first seven digits.
func short(hs []string) []string {
	out := make([]string, len(hs))
	for i, h := range hs {
		out[i] = shortOf(h)
	}
	return out
}

func shortOf(h string) string {
	if h == "" {
		return "-"
	}
	return h[:7]
}

// hashOf is commit n's made-up hash: forty hex digits.
func hashOf(n int) string { return fmt.Sprintf("%040x", n) }

// linearHistory is n commits on main, the newest first: commit n-1 down to
// commit 0, the root, each by "Ana Hubaut", "commit <i>".
func linearHistory(n int) []input.GraphCommit {
	out := make([]input.GraphCommit, 0, n)
	for i := n - 1; i >= 0; i-- {
		c := input.GraphCommit{Hash: hashOf(i), Author: "Ana Hubaut", Time: int64(1_700_000_000 + i), Subject: fmt.Sprintf("commit %d", i)}
		if i > 0 {
			c.Parents = []string{hashOf(i - 1)}
		}
		out = append(out, c)
	}
	return out
}

// newFakeGraph is the Git Graph's fake: main at the newest of n commits,
// origin/main two behind it, a tag v1 on commit 1.
func newFakeGraph(n int) *fakeGraph {
	f := &fakeGraph{fakeGit: &fakeGit{}, history: linearHistory(n), files: map[string]input.CommitFiles{}, sides: map[string]input.DiffPair{}}
	f.refs = input.GraphRefs{Head: hashOf(n - 1), Branch: "refs/heads/main", Upstream: "refs/remotes/origin/main",
		Refs: []input.GraphRef{{Name: "refs/heads/main", Commit: hashOf(n - 1)}, {Name: "refs/remotes/origin/main", Commit: hashOf(max(n-3, 0))},
			{Name: "refs/tags/v1", Commit: hashOf(1)}}}
	return f
}

// sampleGraph is the golden frames' history (golden_test.go): twelve
// commits on main by two authors, commit 9 merging commit 4, the newest
// changing five files, graph.go's sides.
func sampleGraph() *fakeGraph {
	f := newFakeGraph(12)
	subjects := []string{"Name the footer's parts", "Fold the rows on ←", "Merge branch 'rows'", "Rework the rows: no CR, no last empty line",
		"Read the history in pages of 200", "Keep the cursor on its commit", "Watch the followed references", "Add the scope popup",
		"Count each file's lines", "Show a rename's source", "Add the panel", "Initial commit"}
	for i := range f.history {
		f.history[i].Subject = subjects[i]
		if i%3 == 1 {
			f.history[i].Author = "Bo Li"
		}
	}
	f.history[2].Parents = append(f.history[2].Parents, hashOf(4))
	f.files[hashOf(11)] = input.CommitFiles{Files: []input.CommitFile{{Status: 'M', Path: "internal/render/graph.go", Added: 2, Removed: 1},
		{Status: 'A', Path: "internal/render/graph_test.go", Added: 120}, {Status: 'R', Path: "doc/graph.md", Orig: "doc/history.md", Added: 3, Removed: 1},
		{Status: 'D', Path: "old.txt", Removed: 3}, {Status: 'A', Path: "logo.png", Binary: true}}}
	f.sides["internal/render/graph.go"] = input.DiffPair{
		Old: input.DiffSide{Name: hashOf(10)[:7], Path: "internal/render/graph.go", Data: []byte(`package render

const graphFixed = 4
`)},
		New: input.DiffSide{Name: hashOf(11)[:7], Path: "internal/render/graph.go", Data: []byte(`package render

// graphFixed are the rows outside the history.
const graphFixed = 5
`)}}
	return f
}

// lanesGraph is the golden frames' history of many lanes (golden_test.go,
// design D10): main's tip merging ten parents, each a commit on a base,
// the base on the root; eleven lanes, more than the docked panel shows.
func lanesGraph() *fakeGraph {
	f := newFakeGraph(13)
	for i := range f.history {
		f.history[i].Parents = nil
	}
	set := func(i int, ps ...int) {
		c := &f.history[i] // row i, newest first, is commit 12-i
		for _, p := range ps {
			c.Parents = append(c.Parents, hashOf(12-p))
		}
	}
	set(0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10)
	for i := 1; i <= 10; i++ {
		set(i, 11)
	}
	set(11, 12)
	return f
}

// graphRig is the file rig's project with newFakeGraph's history of n
// commits.
func graphRig(t *testing.T, n int) (*harness, *fakeGraph) {
	t.Helper()
	r, _, _ := fileRig(t)
	h := r.h
	f := newFakeGraph(n)
	h.m.opt.Git = f
	h.send(tea.WindowSizeMsg{Width: 120, Height: 40}) // the panel in zone B's place, 76 columns
	return h, f
}

// altG is Alt+G.
func altG() tea.KeyPressMsg { return key('g', tea.ModAlt) }

// TestGraphShow is criterion 62.1 through the application (design D1):
// Alt+G and a click on the tile show and hide the panel with the focus,
// the three panels sharing the docked place, each kept as it was; under
// 100 columns Alt+G shows it in zone B's place.
func TestGraphShow(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, _ := graphRig(t, 10)
		h.send(altG())
		if v := h.m.ui; !v.Graph.Open || v.Explorer.Open || v.Source.Open || v.Focus != "R" || !strings.Contains(h.frame(), "GIT GRAPH") {
			t.Fatalf("Alt+G: graph %v, Explorer %v, Source %v, focus %q", v.Graph.Open, v.Explorer.Open, v.Source.Open, v.Focus)
		}
		h.send(PrimaryDeviceAttributesEvent{61, 4, 6, 7, 14}) // Sixel: the tiles' looks on the screen
		if lk := sideLookOf(h, render.SideGraph); lk != render.SideSelected {
			t.Errorf("the tile's look %v, want selected", lk)
		}
		h.send(keyOf(tea.KeyDown)) // the cursor on the second commit, kept
		h.send(keyOf(tea.KeyF3))   // the Explorer in its place
		if v := h.m.ui; v.Graph.Open || !v.Explorer.Open {
			t.Fatalf("F3: graph %v, Explorer %v", v.Graph.Open, v.Explorer.Open)
		}
		x, y := itemCell(t, h, render.SidePanel, render.SideGraph)
		mouseClick(t, h, x, y)
		if v := h.m.ui; !v.Graph.Open || v.Explorer.Open || v.Graph.Cursor != 1 || v.Focus != "R" {
			t.Fatalf("a click on the tile: graph %v, Explorer %v, cursor %d, focus %q", v.Graph.Open, v.Explorer.Open, v.Graph.Cursor, v.Focus)
		}
		mouseClick(t, h, x, y)
		if h.m.ui.Graph.Open {
			t.Error("a second click on the tile left the panel shown")
		}
		h.send(altG())
		h.send(altG())
		if h.m.ui.Graph.Open || h.m.ui.Focus != "B" {
			t.Errorf("Alt+G twice: graph %v, focus %q", h.m.ui.Graph.Open, h.m.ui.Focus)
		}
		h.send(tea.WindowSizeMsg{Width: 80, Height: 24})
		h.send(altG())
		if sc := screenNow(h); !h.m.ui.Graph.Open || sc.Panel.W != 80 || !strings.Contains(h.frame(), "GIT GRAPH") {
			t.Errorf("Alt+G at 80x24: graph %v, the panel %+v", h.m.ui.Graph.Open, sc.Panel)
		}
	})
}

// TestGraphHistory is criterion 62.2 with the test double (design D4): the
// history in git's order, each row's subject and labels; pages of 200
// joining with none missing or twice as the cursor nears the end; the
// reader closed at the history's end, when the panel hides and when the
// dashboard leaves, a history cut short read again when shown; no commit,
// "no commits yet" and no read; the scope row's Auto and what it follows.
func TestGraphHistory(t *testing.T) {
	t.Run("to the end", func(t *testing.T) { synctest.Test(t, testGraphToTheEnd) })
	t.Run("hidden during a read", func(t *testing.T) { synctest.Test(t, testGraphHiddenDuringRead) })
	t.Run("no commit yet", func(t *testing.T) { synctest.Test(t, testGraphNoCommit) })
}

func testGraphToTheEnd(t *testing.T) {
	{
		h, f := graphRig(t, 450)
		h.send(altG())
		v := h.m.ui.Graph
		if len(v.Commits) != graphPage || len(f.logs) != 1 || !slices.Equal(f.logs[0], []string{hashOf(449), hashOf(447)}) {
			t.Fatalf("the first page: %d commits, logs %v", len(v.Commits), f.logs)
		}
		frame := h.frame()
		for _, want := range []string{"◎ Auto ⌄", "main → origin/main", "◎ main", "commit 449", "origin/main", "commit 447"} {
			if !strings.Contains(frame, want) {
				t.Errorf("no %q in:\n%s", want, frame)
			}
		}
		h.send(keyOf(tea.KeyEnd))
		h.send(keyOf(tea.KeyEnd))
		h.send(keyOf(tea.KeyEnd))
		v = h.m.ui.Graph
		seen := map[[20]byte]bool{}
		for _, e := range v.Commits {
			if seen[e.Hash] {
				t.Fatalf("commit %s twice", e.Short())
			}
			seen[e.Hash] = true
		}
		if len(v.Commits) != 450 || v.Commits[449].Subject != "commit 0" || !strings.Contains(h.frame(), "commit 0") {
			t.Fatalf("to the end: %d commits, the last %q", len(v.Commits), v.Commits[len(v.Commits)-1].Subject)
		}
		if r := f.readers[0]; !r.isClosed() || h.m.graph.reader != nil {
			t.Errorf("the history ended: reader closed %v, held %v", r.isClosed(), h.m.graph.reader != nil)
		}
		if !strings.Contains(h.frame(), "v1") {
			t.Error("the tag's label not on its commit")
		}
		// Hidden and shown again, the history whole and its tips the same:
		// nothing read again.
		h.send(altG())
		h.send(altG())
		if len(f.readers) != 1 || len(h.m.ui.Graph.Commits) != 450 {
			t.Errorf("shown again: %d reads, %d commits", len(f.readers), len(h.m.ui.Graph.Commits))
		}
	}
}

func testGraphHiddenDuringRead(t *testing.T) {
	h, f := graphRig(t, 450)
	h.send(altG())
	h.send(altG())
	if !f.readers[0].isClosed() || h.m.graph.reader != nil {
		t.Fatalf("hidden: the read under way closed %v", f.readers[0].isClosed())
	}
	// Shown again: read again from the start, to go on past the page.
	h.send(altG())
	if len(f.readers) != 2 || len(h.m.ui.Graph.Commits) != graphPage {
		t.Fatalf("shown again: %d reads, %d commits", len(f.readers), len(h.m.ui.Graph.Commits))
	}
	h.send(keyOf(tea.KeyEnd))
	if n := len(h.m.ui.Graph.Commits); n != 2*graphPage {
		t.Errorf("End after showing again: %d commits", n)
	}
	// The dashboard leaving: the read ended.
	h.m.Update(leaveMsg{})
	if !f.readers[1].isClosed() {
		t.Error("leaving: the read under way not closed")
	}
}

func testGraphNoCommit(t *testing.T) {
	h, f := graphRig(t, 3)
	f.refs = input.GraphRefs{}
	h.send(altG())
	if v := h.m.ui.Graph; v.State != "no commits yet" || len(v.Rows) != 0 || len(f.readers) != 0 || v.Scope != "Auto" {
		t.Errorf("no commit: state %q, %d rows, %d reads, scope %q", v.State, len(v.Rows), len(f.readers), v.Scope)
	}
}

// TestGraphTipsMoved is criterion 62.2's consistency (design D4b): commits
// made on the followed branch between two pages read the history again from
// the new tips, the cursor kept on its commit.
func TestGraphTipsMoved(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, f := graphRig(t, 450)
		h.send(altG())
		h.send(keyOf(tea.KeyDown))
		h.send(keyOf(tea.KeyDown)) // the cursor on commit 447
		f.mu.Lock()
		f.history = append([]input.GraphCommit{{Hash: hashOf(451), Parents: []string{hashOf(450)}, Subject: "commit 451"},
			{Hash: hashOf(450), Parents: []string{hashOf(449)}, Subject: "commit 450"}}, f.history...)
		f.refs.Head, f.refs.Refs[0].Commit = hashOf(451), hashOf(451)
		f.mu.Unlock()
		h.send(keyOf(tea.KeyEnd)) // the first page's end: the next asked, the tips checked first
		v := h.m.ui.Graph
		if len(f.logs) != 2 || f.logs[1][0] != hashOf(451) {
			t.Fatalf("the tips moved: logs %v", f.logs)
		}
		if v.Commits[0].Subject != "commit 451" {
			t.Errorf("the history read again: first %q", v.Commits[0].Subject)
		}
		if cur := v.Rows[v.Cursor]; v.Commits[cur.Commit].Subject != "commit 250" {
			t.Errorf("the cursor on %q, want the commit it was on, commit 250", v.Commits[cur.Commit].Subject)
		}
		seen := map[[20]byte]bool{}
		for _, e := range v.Commits {
			if seen[e.Hash] {
				t.Fatalf("commit %s twice", e.Short())
			}
			seen[e.Hash] = true
		}
		if !f.readers[0].isClosed() {
			t.Error("the first read not closed")
		}
		// The new read's own lanes: its commits laid out once each, from the
		// new tips (design D9).
		if v.Lanes == nil || v.Lanes != h.m.graph.lanes || v.Lanes.Len() != len(v.Commits) {
			t.Errorf("the lanes: %d laid out for %d commits", v.Lanes.Len(), len(v.Commits))
		}
	})
}

// TestGraphWide is design D10's wide view through the application: f gives
// the panel zone A's columns while it has the focus, Esc gives them back
// with the focus to the input, the panel shown again wide, f once more
// docked.
func TestGraphWide(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, _ := graphRig(t, 5)
		h.send(tea.WindowSizeMsg{Width: 208, Height: 59})
		h.send(altG())
		if frame := h.frame(); !strings.Contains(frame, "A · ORCHESTRATOR") || !render.PanelShown(h.m.ui) {
			t.Fatalf("docked:\n%s", frame)
		}
		h.send(typed('f'))
		rows := strings.Split(h.frame(), "\n")
		if !h.m.ui.Graph.Wide || !render.GraphWide(h.m.ui) || strings.Contains(h.frame(), "A · ORCHESTRATOR") || !strings.HasPrefix(rows[3], "│ GIT GRAPH") {
			t.Fatalf("f: wide %v, rows %q", h.m.ui.Graph.Wide, rows[2:5])
		}
		h.send(keyOf(tea.KeyEscape))
		if render.GraphWide(h.m.ui) || h.m.ui.Focus != "B" || !strings.Contains(h.frame(), "A · ORCHESTRATOR") {
			t.Errorf("Esc: focus %q, wide drawn %v", h.m.ui.Focus, render.GraphWide(h.m.ui))
		}
		h.send(altG()) // hidden, then shown again with the focus: wide again
		h.send(altG())
		if !render.GraphWide(h.m.ui) {
			t.Errorf("shown again: wide drawn %v, focus %q", render.GraphWide(h.m.ui), h.m.ui.Focus)
		}
		h.send(typed('f'))
		if h.m.ui.Graph.Wide || !strings.Contains(h.frame(), "A · ORCHESTRATOR") {
			t.Errorf("f again: wide %v", h.m.ui.Graph.Wide)
		}
	})
}

// TestGraphScope is criterion 62.3 (design D3): Auto at first; the popup by
// s and by a click on the scope row; All; one reference; two; All and Auto
// exclusive, the last box cleared giving Auto; the search; Enter applying,
// Esc and a click outside keeping the scope; the history read from each
// scope's tips; a reference gone; never a switch or a fetch.
func TestGraphScope(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, f := graphRig(t, 20)
		h.send(altG())
		h.send(typed('s'))
		v := h.m.ui.Scope
		if !v.Open || v.All || len(v.Checked) != 0 || !strings.Contains(h.frame(), "◉ Auto") || !strings.Contains(h.frame(), "History scope") {
			t.Fatalf("s: open %v, all %v, checked %v:\n%s", v.Open, v.All, v.Checked, h.frame())
		}
		// All, applied.
		h.m.ui.Scope.Cursor = "a"
		h.send(key(' ', 0))
		h.send(keyOf(tea.KeyEnter))
		if s := h.m.graph.scope; !s.All || h.m.ui.Graph.Scope != "All" || h.m.ui.Scope.Open {
			t.Fatalf("All applied: scope %+v, row %q", s, h.m.ui.Graph.Scope)
		}
		if last := f.logs[len(f.logs)-1]; !slices.Equal(last, []string{hashOf(19), hashOf(17), hashOf(1)}) {
			t.Errorf("All's tips %v", short(last))
		}
		// One reference, by its row: the popup opened by a click on the scope
		// row, the tag found by the search.
		x, y := itemCell(t, h, render.GraphPanel, render.ItemGraphScope)
		mouseClick(t, h, x, y)
		for _, k := range typedText("v1") {
			h.send(k)
		}
		if h.m.ui.Scope.Cursor != "r:refs/tags/v1" || !strings.Contains(h.frame(), "☐ v1") {
			t.Fatalf("searched v1: cursor %q:\n%s", h.m.ui.Scope.Cursor, h.frame())
		}
		h.send(key(' ', 0))
		if v := h.m.ui.Scope; v.All || !v.Checked["refs/tags/v1"] || !strings.Contains(h.frame(), "☑ v1") || !strings.Contains(h.frame(), "○ All") {
			t.Fatalf("v1 ticked: all %v, checked %v", v.All, v.Checked)
		}
		h.send(keyOf(tea.KeyEnter))
		if s := h.m.graph.scope; !slices.Equal(s.Refs, []string{"refs/tags/v1"}) || h.m.ui.Graph.Scope != "v1" {
			t.Fatalf("v1 applied: scope %+v, row %q", s, h.m.ui.Graph.Scope)
		}
		if last := f.logs[len(f.logs)-1]; !slices.Equal(last, []string{hashOf(1)}) || len(h.m.ui.Graph.Commits) != 2 {
			t.Errorf("v1's history: tips %v, %d commits", short(last), len(h.m.ui.Graph.Commits))
		}
		// Two references; then Esc and a click outside change nothing.
		h.send(typed('s'))
		h.m.ui.Scope.Folded = [render.ScopeGroups]bool{}
		h.m.ui.Scope.Cursor = "r:refs/remotes/origin/main"
		h.send(key(' ', 0))
		h.send(keyOf(tea.KeyEnter))
		if s := h.m.graph.scope; len(s.Refs) != 2 || h.m.ui.Graph.Scope != "2 references" {
			t.Fatalf("two applied: scope %+v, row %q", s, h.m.ui.Graph.Scope)
		}
		before := len(f.logs)
		h.send(typed('s'))
		h.m.ui.Scope.Cursor = "a"
		h.send(key(' ', 0))
		h.send(keyOf(tea.KeyEscape))
		if s := h.m.graph.scope; s.All || len(s.Refs) != 2 || h.m.ui.Scope.Open || len(f.logs) != before {
			t.Errorf("Esc: scope %+v, open %v, %d new reads", s, h.m.ui.Scope.Open, len(f.logs)-before)
		}
		h.send(typed('s'))
		h.m.ui.Scope.Cursor = "u"
		h.send(key(' ', 0))
		mouseClick(t, h, 1, 0)
		if s := h.m.graph.scope; len(s.Refs) != 2 || h.m.ui.Scope.Open {
			t.Errorf("a click outside: scope %+v, open %v", s, h.m.ui.Scope.Open)
		}
		// The last box cleared: Auto.
		h.send(typed('s'))
		h.m.ui.Scope.Folded = [render.ScopeGroups]bool{}
		v = h.m.ui.Scope
		for name := range v.Checked {
			h.m.ui.Scope.Cursor = "r:" + name
			h.send(key(' ', 0))
		}
		if v := h.m.ui.Scope; len(v.Checked) != 0 || v.All || !strings.Contains(h.frame(), "◉ Auto") {
			t.Errorf("every box cleared: checked %v, all %v", v.Checked, v.All)
		}
		h.send(keyOf(tea.KeyEnter))
		if !h.m.graph.scope.Auto() || h.m.ui.Graph.Scope != "Auto" {
			t.Errorf("Auto again: %+v", h.m.graph.scope)
		}
		// A reference of the selection gone: left out, said.
		h.m.graph.scope = input.GraphScope{Refs: []string{"refs/tags/v1", "refs/heads/gone"}}
		h.send(graphRefsMsg{gen: h.m.graph.refsGen, refs: f.refs})
		if v := h.m.ui.Graph; v.ScopeNote != "1 reference gone" {
			t.Errorf("a reference gone: note %q", v.ScopeNote)
		}
		if got := f.called("switch"); len(got) != 0 || len(f.called("fetch")) != 0 {
			t.Errorf("the scope switched or fetched: %v", f.calls)
		}
	})
}

// TestGraphFiles is criterion 62.4 through the application (design D5): →
// unfolds a commit into its files, read once, against its first parent; ←
// folds it, on a file back to its commit; a root's files against nothing.
func TestGraphFiles(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, f := graphRig(t, 5)
		f.files[hashOf(4)] = input.CommitFiles{Files: []input.CommitFile{{Status: 'M', Path: "main.go", Added: 3, Removed: 1},
			{Status: 'R', Path: "web/app.ts", Orig: "web/old.ts"}, {Status: 'A', Path: "logo.png", Binary: true}}}
		f.files[hashOf(0)] = input.CommitFiles{Files: []input.CommitFile{{Status: 'A', Path: "README.md", Added: 1}}}
		h.send(altG())
		h.send(keyOf(tea.KeyRight))
		v := h.m.ui.Graph
		if len(v.Rows) != 5+3 || v.Rows[1].File != 0 || f.called("files")[0] != "files "+hashOf(4)[:7]+" "+hashOf(3)[:7] {
			t.Fatalf("→: %d rows, calls %v", len(v.Rows), f.called("files"))
		}
		frame := h.frame()
		for _, want := range []string{"M  main.go", "+3 −1", "R  app.ts ← web/old.ts", "A  logo.png", "binary"} {
			if !strings.Contains(frame, want) {
				t.Errorf("no %q:\n%s", want, frame)
			}
		}
		h.send(keyOf(tea.KeyDown))
		h.send(keyOf(tea.KeyLeft))
		if v := h.m.ui.Graph; v.Cursor != 0 || len(v.Rows) != 8 {
			t.Errorf("← on a file: cursor %d, %d rows", v.Cursor, len(v.Rows))
		}
		h.send(keyOf(tea.KeyLeft))
		h.send(keyOf(tea.KeyRight))
		if n := len(f.called("files")); n != 1 || len(h.m.ui.Graph.Rows) != 8 {
			t.Errorf("folded and unfolded again: %d reads, %d rows", n, len(h.m.ui.Graph.Rows))
		}
		h.send(keyOf(tea.KeyEnd))
		h.send(keyOf(tea.KeyRight))
		if got := f.called("files"); len(got) != 2 || got[1] != "files "+hashOf(0)[:7]+" -" {
			t.Errorf("the root unfolded: %v", got)
		}
	})
}

// TestGraphNoFiles is design D5's empty case: a commit that changed no
// file unfolds into one row saying so, a merge's against its first parent;
// Enter on that row opens nothing.
func TestGraphNoFiles(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, f := graphRig(t, 3)
		f.history[0].Parents = append(f.history[0].Parents, hashOf(0)) // commit 2 merges the root
		h.send(altG())
		h.send(keyOf(tea.KeyRight))
		if v := h.m.ui.Graph; len(v.Rows) != 4 || v.Rows[1] != (render.GraphRowRef{Commit: 0, File: 0}) {
			t.Fatalf("the merge unfolded: rows %v", v.Rows)
		}
		if frame := h.frame(); !strings.Contains(frame, "  no change against its first parent") {
			t.Errorf("no note under the merge:\n%s", frame)
		}
		h.send(keyOf(tea.KeyDown))
		h.send(keyOf(tea.KeyEnter))
		if h.m.ui.Diff.Open || len(f.called("sides")) != 0 {
			t.Errorf("Enter on the note: %+v, sides read %v", h.m.ui.Diff, f.called("sides"))
		}
		h.send(keyOf(tea.KeyDown))
		h.send(keyOf(tea.KeyRight))
		if frame := h.frame(); !strings.Contains(frame, "  no file changed") {
			t.Errorf("no note under commit 1:\n%s", frame)
		}
	})
}

// TestGraphDiff is criterion 62.5 through the application (design D6): Enter
// on a file opens its diff in the viewer, read only, the sides the commit's
// and its first parent's; Source Control's own diff kept apart and shown
// again with Source Control; nothing added to zone B, nothing sent.
func TestGraphDiff(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, f := graphRig(t, 3)
		f.files[hashOf(2)] = input.CommitFiles{Files: []input.CommitFile{{Status: 'M', Path: "main.go", Added: 1, Removed: 1}}}
		f.sides["main.go"] = input.DiffPair{Old: input.DiffSide{Name: hashOf(1)[:7], Path: "main.go", Data: []byte("a\nb\n")},
			New: input.DiffSide{Name: hashOf(2)[:7], Path: "main.go", Data: []byte("a\nc\n")}}
		h.m.ui.Diff = render.DiffView{Open: true, Path: "notes.txt", Staged: true, Note: "Source Control's"}
		entries := len(h.m.s.Conversation)
		h.send(altG())
		if h.m.ui.Diff.Open || !h.m.graph.keptDiff.Open {
			t.Fatalf("shown: diff %+v, kept %v", h.m.ui.Diff, h.m.graph.keptDiff.Open)
		}
		h.send(keyOf(tea.KeyRight))
		h.send(keyOf(tea.KeyDown))
		h.send(keyOf(tea.KeyEnter))
		d := h.m.ui.Diff
		if !d.Open || d.Commit != "Commit "+hashOf(2)[:7] || d.OldName != hashOf(1)[:7] || d.NewName != hashOf(2)[:7] || d.Added != 1 || d.Removed != 1 || h.m.ui.Focus != "D" {
			t.Fatalf("Enter on main.go: %+v", d)
		}
		if got := f.called("sides"); len(got) != 1 || got[0] != "sides "+hashOf(2)[:7]+" "+hashOf(1)[:7]+" main.go" {
			t.Errorf("the sides read: %v", got)
		}
		if !strings.Contains(h.frame(), "Commit "+hashOf(2)[:7]) {
			t.Errorf("the path row:\n%s", h.frame())
		}
		h.send(keyOf(tea.KeyEscape))
		if h.m.ui.Diff.Open || h.m.ui.Focus != "R" {
			t.Errorf("Esc: diff %v, focus %q", h.m.ui.Diff.Open, h.m.ui.Focus)
		}
		h.send(altG()) // hidden; then Source Control shown: its own diff back
		h.m.showSource()
		if d := h.m.ui.Diff; !d.Open || d.Note != "Source Control's" || d.Path != "notes.txt" {
			t.Errorf("Source Control's diff back: %+v", d)
		}
		if len(h.m.s.Conversation) != entries || h.m.State().Turn.Phase != state.Idle {
			t.Errorf("zone B got %d entries, the turn %q: something sent", len(h.m.s.Conversation)-entries, h.m.State().Turn.Phase)
		}
	})
}

// TestGraphPlaceKept is criterion 62.9 (design D7): hidden and shown again,
// three commits made meanwhile, the cursor, the first row and the unfolded
// commit stay on their commits.
func TestGraphPlaceKept(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, f := graphRig(t, 100)
		f.files[hashOf(97)] = input.CommitFiles{Files: []input.CommitFile{{Status: 'M', Path: "main.go"}}}
		h.send(altG())
		for range 2 {
			h.send(keyOf(tea.KeyDown))
		}
		h.send(keyOf(tea.KeyRight)) // commit 97 unfolded, the cursor on it
		h.send(keyOf(tea.KeyF3))
		f.mu.Lock()
		f.history = append([]input.GraphCommit{{Hash: hashOf(102), Parents: []string{hashOf(101)}, Subject: "commit 102"},
			{Hash: hashOf(101), Parents: []string{hashOf(100)}, Subject: "commit 101"},
			{Hash: hashOf(100), Parents: []string{hashOf(99)}, Subject: "commit 100"}}, f.history...)
		f.refs.Head, f.refs.Refs[0].Commit = hashOf(102), hashOf(102)
		f.mu.Unlock()
		h.send(altG())
		v := h.m.ui.Graph
		cur := v.Rows[v.Cursor]
		if got := v.Commits[cur.Commit].Subject; got != "commit 97" || cur.File != -1 {
			t.Errorf("the cursor on %q (file %d), want commit 97's row", got, cur.File)
		}
		if next := v.Rows[v.Cursor+1]; next.Commit != cur.Commit || next.File != 0 {
			t.Errorf("commit 97 no longer unfolded: next row %+v", next)
		}
		if v.Commits[0].Subject != "commit 102" {
			t.Errorf("the new commits not read: first %q", v.Commits[0].Subject)
		}
	})
}

// TestGraphFrameCost is criterion 62.10 through the application (design
// D15): a history of 50,000 commits, a merge every tenth, each subject 48
// bytes, read to its first commit by End, a page each, none missing or
// twice; the heap the history keeps measured, its parts named; a key to its
// frame with all of them read, judged in the timing step.
func TestGraphFrameCost(t *testing.T) {
	const n = 50_000
	h, f := graphRig(t, n)
	for i := 10; i < n; i += 10 {
		c := &f.history[n-1-i]
		c.Parents = append(c.Parents, hashOf(i-5))
	}
	for i := range f.history {
		f.history[i].Subject = fmt.Sprintf("Rework the rows of the panel: step %05d of 50000", n-1-i)
	}
	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	h.send(altG())
	// Each page waited for by until, not by send's quiet of 300 ms.
	for i := 0; !h.m.graph.done && i < n/graphPage+5; i++ {
		read := len(h.m.ui.Graph.Commits)
		h.deliver(keyOf(tea.KeyEnd))
		if !h.until(func(*state.State) bool { return h.m.graph.done || len(h.m.ui.Graph.Commits) > read }, 10*time.Second) {
			t.Fatalf("End: no page after commit %d", read)
		}
	}
	h.settle()
	v := h.m.ui.Graph
	seen := make(map[[20]byte]bool, n)
	for _, e := range v.Commits {
		seen[e.Hash] = true
	}
	if len(v.Commits) != n || len(seen) != n || !strings.HasSuffix(v.Commits[n-1].Subject, "step 00000 of 50000") || len(f.readers) != 1 {
		t.Fatalf("read %d commits, %d distinct, %d reads, want %d once each", len(v.Commits), len(seen), len(f.readers), n)
	}
	f.readers[0].commits = nil // the fake's copy, not the application's
	seen = nil
	runtime.GC()
	runtime.ReadMemStats(&after)
	total := (int64(after.HeapAlloc) - int64(before.HeapAlloc)) / n
	entries, rows := int64(cap(v.Commits))*int64(unsafe.Sizeof(render.GraphEntry{}))/n, int64(cap(v.Rows))*int64(unsafe.Sizeof(render.GraphRowRef{}))/n
	t.Logf("the history kept: %d bytes a commit: %d its entry (%d bytes, the slice's room included), %d its row, %d its subject's text, %d the index by hash and the lanes",
		total, entries, unsafe.Sizeof(render.GraphEntry{}), rows, len(v.Commits[0].Subject), total-entries-rows-int64(len(v.Commits[0].Subject)))
	h.send(keyOf(tea.KeyHome))
	var ds []time.Duration
	for i := range 400 {
		k := keyOf(tea.KeyDown)
		if i%100 >= 50 {
			k = keyOf(tea.KeyUp)
		}
		ds = append(ds, frameTime(h, k))
	}
	p50, p95 := percentile(ds, 0.50), percentile(ds, 0.95)
	t.Logf("a key to its frame, 50,000 commits: p50 %v, p95 %v (target p95 ≤ %v)", p50, p95, keystrokeTarget)
	if p95 > keystrokeTarget {
		overTarget(t, "p95 %v over the %v target", p95, keystrokeTarget)
	}
}

// TestGraphCard is criterion 62.8 through the application (design D13): the
// pointer on a commit's row shows its card, its message and its files read
// once each; a motion within the row goes no further than the filter, one
// off the rows takes the card away; i shows the cursor's card and i again
// takes it away; read once, a message is not read again.
func TestGraphCard(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, f := graphRig(t, 5)
		f.messages = map[string]string{hashOf(3): "commit 3\n\nIts body."}
		f.files[hashOf(3)] = input.CommitFiles{Files: []input.CommitFile{{Status: 'M', Path: "main.go", Added: 2, Removed: 1}}}
		h.send(altG())
		x, y := at(t, h, "commit 3", 0)
		h.send(tea.MouseMotionMsg{X: x, Y: y})
		c := h.m.ui.Graph.Card
		if c == nil || c.Row != 1 || c.Message != "commit 3\n\nIts body." || c.Files != "1 file changed, +2 −1" || c.State != "" {
			t.Fatalf("the pointer on commit 3: card %+v", c)
		}
		if m, fs := f.called("message"), f.called("files"); len(m) != 1 || len(fs) != 1 {
			t.Errorf("reads: messages %v, files %v", m, fs)
		}
		if frame := h.frame(); !strings.Contains(frame, "Its body.") || !strings.Contains(frame, "1 file changed, +2 −1") {
			t.Errorf("the card not drawn:\n%s", frame)
		}
		if motionFilter(h.m, tea.MouseMotionMsg{X: x + 1, Y: y}) != nil {
			t.Error("a motion within the row went through the filter")
		}
		h.send(tea.MouseMotionMsg{X: 0, Y: 0})
		if h.m.ui.Graph.Card != nil {
			t.Errorf("off the rows: card %+v", h.m.ui.Graph.Card)
		}
		h.send(typed('i'))
		if c := h.m.ui.Graph.Card; c == nil || c.Row != 0 {
			t.Errorf("i: card %+v, want the cursor's, row 0", c)
		}
		h.send(typed('i'))
		if h.m.ui.Graph.Card != nil {
			t.Errorf("i again: card %+v", h.m.ui.Graph.Card)
		}
		h.send(tea.MouseMotionMsg{X: x, Y: y})
		if c := h.m.ui.Graph.Card; c == nil || c.Message == "" || len(f.called("message")) != 2 {
			t.Errorf("commit 3 again: card %+v, reads %v (commit 4's by i, commit 3's once)", c, f.called("message"))
		}
	})
}

// TestGraphMouseMode is design D13's pointer under 100 columns, where no
// sidebar asks for it: every motion reported while the Git Graph is drawn,
// for its card; button motions only once it is hidden.
func TestGraphMouseMode(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, _ := graphRig(t, 3)
		h.send(tea.WindowSizeMsg{Width: 80, Height: 24})
		h.send(altG())
		h.frame()
		if mode := h.m.View().MouseMode; mode != tea.MouseModeAllMotion {
			t.Errorf("shown at 80x24: mouse mode %v, want all motions", mode)
		}
		h.send(altG())
		h.frame()
		if mode := h.m.View().MouseMode; mode != tea.MouseModeCellMotion {
			t.Errorf("hidden at 80x24: mouse mode %v, want button motions", mode)
		}
	})
}

// TestGraphWatch is criterion 62.2's watch (design D4b): while the panel
// shows, the references it follows are read again every GraphWatch, a
// commit made outside on one of them seen at the next tick and the history
// read again from it; a fetch the branches popup ran reads them at once;
// hidden, a tick reads nothing and asks for no other.
func TestGraphWatch(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, f := graphRig(t, 5)
		var due []tea.Msg
		h.m.opt.GraphWatch = 2 * time.Second
		h.m.opt.After = func(d time.Duration, msg tea.Msg) tea.Cmd {
			if _, ok := msg.(graphTickMsg); ok && d == 2*time.Second {
				due = append(due, msg)
			}
			return nil
		}
		next := func() tea.Msg {
			t.Helper()
			if len(due) != 1 {
				t.Fatalf("%d ticks asked, want 1", len(due))
			}
			tick := due[0]
			due = nil
			return tick
		}
		commit := func(n int) {
			f.mu.Lock()
			f.history = append([]input.GraphCommit{{Hash: hashOf(n), Parents: []string{hashOf(n - 1)}, Subject: fmt.Sprintf("commit %d", n)}}, f.history...)
			f.refs.Head, f.refs.Refs[0].Commit = hashOf(n), hashOf(n)
			f.mu.Unlock()
		}
		h.send(altG())
		commit(5) // outside, on main
		h.send(next())
		if v := h.m.ui.Graph; v.Commits[0].Subject != "commit 5" || len(f.logs) != 2 {
			t.Errorf("the tick after a commit outside: first %q, %d reads", v.Commits[0].Subject, len(f.logs))
		}
		tick := next() // the watch goes on
		commit(6)
		stop := &input.GitStop{}
		h.m.fetch = stop
		h.send(fetchedMsg{stop: stop})
		if v := h.m.ui.Graph; v.Commits[0].Subject != "commit 6" || len(f.logs) != 3 {
			t.Errorf("after a fetch: first %q, %d reads", v.Commits[0].Subject, len(f.logs))
		}
		h.send(altG())
		refs := len(f.called("refs"))
		h.send(tick)
		if len(f.called("refs")) != refs || len(due) != 0 {
			t.Errorf("hidden: the references read %d times more, %d ticks asked", len(f.called("refs"))-refs, len(due))
		}
	})
}

// TestGraphStalePage is design D4's reads replaced: a page of a read the
// scope's change replaced, coming after the new read's, is dropped and its
// reader closed, the history the new read's.
func TestGraphStalePage(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, f := graphRig(t, 5)
		gate := make(chan struct{})
		f.gate = gate
		h.send(altG()) // Auto's read held before its page
		h.send(typed('s'))
		h.frame()                // the popup drawn, as the program draws it before a key
		h.send(keyOf(tea.KeyUp)) // All
		h.send(keyOf(tea.KeySpace))
		h.send(keyOf(tea.KeyEnter))
		if v := h.m.ui.Graph; len(v.Commits) != 5 || v.Scope != "All" {
			t.Fatalf("All read: %d commits, scope %q", len(v.Commits), v.Scope)
		}
		f.mu.Lock()
		f.history[0].Subject = "stale"
		f.mu.Unlock()
		close(gate) // Auto's page comes now, its read long replaced
		h.settle()
		if v := h.m.ui.Graph; v.Commits[0].Subject != "commit 4" || len(f.readers) != 2 || !f.readers[1].isClosed() {
			t.Errorf("the stale page: first %q, %d reads, the stale one closed %v", v.Commits[0].Subject, len(f.readers), len(f.readers) == 2 && f.readers[1].isClosed())
		}
	})
}

// TestGraphDuplicate is design D4's check by hash: a commit git would give
// twice, across two pages, is kept once, its lanes laid out once.
func TestGraphDuplicate(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, f := graphRig(t, 450)
		f.dup = true
		h.send(altG())
		for range 4 {
			h.send(keyOf(tea.KeyEnd))
		}
		v := h.m.ui.Graph
		seen := map[[20]byte]bool{}
		for _, e := range v.Commits {
			if seen[e.Hash] {
				t.Fatalf("commit %s twice", e.Short())
			}
			seen[e.Hash] = true
		}
		if len(v.Commits) != 450 || v.Lanes.Len() != 450 {
			t.Errorf("%d commits, %d laid out, want 450", len(v.Commits), v.Lanes.Len())
		}
	})
}

// TestGraphCopyAndScroll is criteria 62.3, 62.4 and 62.8 through the
// application (designs D3, D5 and D8): Ctrl+C copies the cursor's commit's
// full hash, or its file's path, never an interrupt; the wheel over the
// panel moves the cursor three rows; a commit of 1,500 files, each listed
// and scrolled to; the scope popup's long list scrolled to its last
// reference.
func TestGraphCopyAndScroll(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r, clip, _ := fileRig(t)
		h := r.h
		f := newFakeGraph(5)
		var files []input.CommitFile
		for i := 1; i <= 1500; i++ {
			files = append(files, input.CommitFile{Status: 'A', Path: fmt.Sprintf("gen/f%04d.txt", i), Added: 1})
		}
		f.files[hashOf(3)] = input.CommitFiles{Files: files}
		for i := range 60 {
			f.refs.Refs = append(f.refs.Refs, input.GraphRef{Name: fmt.Sprintf("refs/heads/topic-%02d", i), Commit: hashOf(2)})
		}
		h.m.opt.Git = f
		h.send(tea.WindowSizeMsg{Width: 120, Height: 40})
		h.send(altG())
		h.send(key('c', tea.ModCtrl))
		if got := clip.last(); got != hashOf(4) || h.m.State().Turn.Phase != state.Idle {
			t.Errorf("Ctrl+C on commit 4: copied %q, turn %q", got, h.m.State().Turn.Phase)
		}
		x, y := at(t, h, "commit 4", 0)
		h.send(tea.MouseWheelMsg{X: x, Y: y, Button: tea.MouseWheelDown})
		if v := h.m.ui.Graph; v.Cursor != 3 {
			t.Errorf("the wheel down: cursor %d, want 3", v.Cursor)
		}
		h.send(keyOf(tea.KeyHome))
		h.send(keyOf(tea.KeyDown))
		h.send(keyOf(tea.KeyRight)) // commit 3's 1,500 files
		h.send(keyOf(tea.KeyEnd))
		h.send(keyOf(tea.KeyUp))
		h.send(keyOf(tea.KeyUp))
		h.send(keyOf(tea.KeyUp)) // three rows above the root, under commits 2 and 1: the last file
		v := h.m.ui.Graph
		if row := v.Rows[v.Cursor]; len(v.Rows) != 5+1500 || row.File != 1499 || !strings.Contains(h.frame(), "f1500.txt") {
			t.Fatalf("1,500 files: %d rows, the cursor on %+v", len(v.Rows), row)
		}
		h.send(key('c', tea.ModCtrl))
		if got := clip.last(); got != "gen/f1500.txt" {
			t.Errorf("Ctrl+C on the last file: copied %q", got)
		}
		h.send(typed('s')) // Local open, its 61 branches
		h.frame()
		h.send(keyOf(tea.KeyEnd))
		frame := h.frame()
		if sv := h.m.screen.Scope; !strings.Contains(frame, "topic-59") || sv.Top == 0 || sv.Rows < 60 {
			t.Errorf("the popup scrolled to its end: %+v\n%s", sv, frame)
		}
	})
}

// TestGraphScopeKeepsPlace is criterion 62.9's scope changed (design D7):
// the cursor's commit, unfolded, still in the new scope's history, the
// cursor stays on it and it stays unfolded; one no longer in it leaves the
// cursor on the history's first commit.
func TestGraphScopeKeepsPlace(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, _ := graphRig(t, 10) // main on commit 9, origin/main on commit 7
		h.send(altG())
		for range 4 {
			h.send(keyOf(tea.KeyDown)) // commit 5
		}
		h.send(keyOf(tea.KeyRight))
		only := func(refs ...string) { // each ticked or cleared in turn
			h.send(typed('s'))
			h.m.ui.Scope.Folded = [render.ScopeGroups]bool{}
			for _, ref := range refs {
				h.m.ui.Scope.Cursor = "r:" + ref
				h.send(key(' ', 0))
			}
			h.send(keyOf(tea.KeyEnter))
		}
		only("refs/remotes/origin/main")
		v := h.m.ui.Graph
		if len(v.Commits) != 8 || v.Commits[v.Rows[v.Cursor].Commit].Subject != "commit 5" || !h.m.graph.unfolded[v.Commits[v.Rows[v.Cursor].Commit].Hash] || v.Rows[v.Cursor+1].File != 0 {
			t.Fatalf("origin/main's history: %d commits, the cursor on %q", len(v.Commits), v.Commits[v.Rows[v.Cursor].Commit].Subject)
		}
		h.send(keyOf(tea.KeyHome))                       // commit 7
		only("refs/remotes/origin/main", "refs/tags/v1") // origin/main cleared, v1 ticked: commits 1 and 0
		if v := h.m.ui.Graph; len(v.Commits) != 2 || v.Cursor != 0 {
			t.Errorf("v1's history: %d commits, cursor %d", len(v.Commits), v.Cursor)
		}
	})
}

// TestGraphFollows is criterion 62.2's Auto without an upstream and on a
// detached HEAD (design D2): the history read from HEAD alone, the scope
// row saying what Auto follows.
func TestGraphFollows(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, f := graphRig(t, 4)
		f.refs.Upstream = ""
		h.send(altG())
		if v := h.m.ui.Graph; v.Follows != "main, no upstream" || len(f.logs) != 1 || !slices.Equal(f.logs[0], []string{hashOf(3)}) || len(v.Commits) != 4 {
			t.Errorf("no upstream: follows %q, logs %v", v.Follows, f.logs)
		}
		h.send(altG())
		f.refs.Branch, f.refs.Head = "", hashOf(2)
		h.send(altG())
		if v := h.m.ui.Graph; v.Follows != "detached at "+hashOf(2)[:7] || len(f.logs) != 2 || !slices.Equal(f.logs[1], []string{hashOf(2)}) || len(v.Commits) != 3 {
			t.Errorf("detached: follows %q, logs %v, %d commits", v.Follows, f.logs, len(v.Commits))
		}
	})
}
