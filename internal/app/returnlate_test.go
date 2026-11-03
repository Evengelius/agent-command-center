package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
	"testing/synctest"

	tea "charm.land/bubbletea/v2"

	"github.com/Evengelius/agent-command-center/internal/event"
	"github.com/Evengelius/agent-command-center/internal/input"
	"github.com/Evengelius/agent-command-center/internal/input/imagetest"
	"github.com/Evengelius/agent-command-center/internal/provider/claude"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// Plan 75, design D9: every read the session started that answers after the
// return changes nothing, neither the state nor the model, and never ends
// the program.

// heldGraph is sampleGraph's git whose commit messages and files wait for
// hold to close.
type heldGraph struct {
	*fakeGraph
	hold chan struct{}
}

func (g *heldGraph) CommitMessage(dir, commit string) (string, error) {
	<-g.hold
	return g.fakeGraph.CommitMessage(dir, commit)
}

func (g *heldGraph) CommitFiles(dir, commit, parent string) (input.CommitFiles, error) {
	<-g.hold
	return g.fakeGraph.CommitFiles(dir, commit, parent)
}

// lateRig is a live Claude Code session at 200x60 in a fresh project, its
// git git, from the home screen of a store of its own.
type lateRig struct {
	h   *harness
	w   *wireClaude
	ps  *input.Projects
	dir string
}

func newLateRig(t *testing.T, git input.Git) *lateRig {
	t.Helper()
	r := &lateRig{w: &wireClaude{}, ps: &input.Projects{Dir: t.TempDir()}, dir: t.TempDir()}
	r.h = homeRig(t, r.w, &wireCodex{}, r.ps, r.dir)
	r.h.m.opt.Git = git
	r.h.send(key(tea.KeyEnter, 0))
	r.w.last().emit(claudeInit(r.dir, "s-late"))
	r.h.settle()
	r.h.send(tea.WindowSizeMsg{Width: 200, Height: 60})
	if s := r.h.m.State(); s.Session.Phase != state.Ready {
		t.Fatalf("session %q", s.Session.Phase)
	}
	return r
}

// home returns to the home screen: after a crash, at once; else from idle,
// once the CLI has exited.
func (r *lateRig) home(t *testing.T, crash bool) {
	t.Helper()
	if crash {
		r.w.last().exit(1, "")
		r.h.settle()
	}
	// Esc as the owner presses it: what is open closes first, then the question.
	for i := 0; i < 6 && !r.h.m.ui.Return.Open; i++ {
		r.h.send(esc())
	}
	r.h.send(key(tea.KeyLeft, 0))
	r.h.send(key(tea.KeyEnter, 0))
	if s := r.h.m.State(); s.Screen != state.ScreenStartup {
		t.Fatalf("no home screen: session %q", s.Session.Phase)
	}
}

// fresh checks nothing of the session is left: the state New's but for the
// answers, every field of the model a fresh model's.
func (r *lateRig) fresh(t *testing.T, before map[string]int) {
	t.Helper()
	want := *state.New()
	want.Startup = r.h.m.s.Startup
	if !reflect.DeepEqual(*r.h.m.s, want) {
		t.Errorf("the state after the late answers:\n%+v", *r.h.m.s)
	}
	other := homeRig(t, &wireClaude{}, &wireCodex{}, r.ps, r.dir)
	sameAsFresh(t, r.h.m, other.m, before)
}

// lateAnswers takes in the late answers now released, as the program would;
// one that ends the program, as a panic in Update does, fails the test.
func lateAnswers(t *testing.T, h *harness) {
	t.Helper()
	defer func() {
		if v := recover(); v != nil {
			t.Errorf("a late answer of the ended session ended the program: %v", v)
		}
	}()
	h.settle()
}

// TestReturnLateGraphCard: the Git Graph's card reads, a commit's message and
// its files, and an unfolded commit's files, held past the return, after a
// crash and from idle, answer on the home screen: no panic, nothing kept.
func TestReturnLateGraphCard(t *testing.T) {
	for _, crash := range []bool{true, false} {
		synctest.Test(t, func(t *testing.T) {
			g := &heldGraph{fakeGraph: sampleGraph(), hold: make(chan struct{})}
			r := newLateRig(t, g)
			free := released(t, g.hold)
			h := r.h
			h.send(key('g', tea.ModAlt))
			if v := h.m.ui.Graph; !v.Open || len(v.Commits) == 0 {
				t.Fatalf("the Git Graph: open %v, %d commits", v.Open, len(v.Commits))
			}
			h.send(key(tea.KeyDown, 0))
			h.send(typed('i'))           // the card pinned: its message and files asked
			h.send(key(tea.KeyRight, 0)) // the commit unfolded: its files asked
			h.send(typed('i'))           // the card unpinned, its reads still under way: Esc may ask
			if len(h.m.graph.messageAsked) == 0 || len(h.m.graph.filesAsked) == 0 {
				t.Fatalf("no card read under way: messages %v, files %v", h.m.graph.messageAsked, h.m.graph.filesAsked)
			}
			before := gensOf(h.m)
			r.home(t, crash)
			free()
			lateAnswers(t, h)
			if h.m.graph.messages != nil || h.m.graph.files != nil || h.m.ui.Graph.Open {
				t.Errorf("crash %v: the Git Graph's reads kept: messages %v, files %v", crash, h.m.graph.messages, h.m.graph.files)
			}
			r.fresh(t, before)
		})
	}
}

// TestReturnLateReads: one read of each other kind the session starts, held
// past the return and answered on the home screen, changes nothing:
// Source Control's status, the diff view's sides, the Explorer's marks, the
// branches popup's lists, the file viewer's blame, the Open dialog's images.
func TestReturnLateReads(t *testing.T) {
	entries := []input.GitEntry{{X: ' ', Y: 'M', Path: "a.go", RepoPath: "a.go"}}
	sides := map[string]input.DiffPair{"a.go false": {Old: input.DiffSide{Name: "index", ID: "1a", Data: []byte("a\n")}, New: input.DiffSide{Name: "working tree", Data: []byte("b\n")}}}
	block := func(f *fakeGit) chan struct{} {
		c := make(chan struct{})
		f.set(func(f *fakeGit) { f.block = c })
		return c
	}
	type row struct {
		name  string
		git   func() (input.Git, *fakeGit)
		start func(t *testing.T, r *lateRig, f *fakeGit) (release func())
	}
	rows := []row{
		{"Source Control's status", func() (input.Git, *fakeGit) {
			f := &fakeGit{snap: input.GitSnapshot{Top: "x", Branch: input.GitBranch{Name: "main"}, Entries: entries}}
			return f, f
		}, func(t *testing.T, r *lateRig, f *fakeGit) func() {
			c := block(f)
			for _, cmd := range r.h.m.showSource() {
				r.h.exec(cmd)
			}
			r.h.settle()
			return func() { close(c) }
		}},
		{"the diff view's sides", func() (input.Git, *fakeGit) {
			f := &fakeGit{snap: input.GitSnapshot{Top: "x", Branch: input.GitBranch{Name: "main"}, Entries: entries}, sides: sides}
			return f, f
		}, func(t *testing.T, r *lateRig, f *fakeGit) func() {
			for _, cmd := range r.h.m.showSource() {
				r.h.exec(cmd)
			}
			r.h.settle()
			sourceTo(t, r.h, "f:changes:a.go")
			c := block(f)
			r.h.send(key(tea.KeyEnter, 0))
			if !r.h.m.ui.Diff.Open {
				t.Fatal("no diff open")
			}
			return func() { close(c) }
		}},
		{"the Explorer's marks", func() (input.Git, *fakeGit) { f := &fakeGit{}; return f, f }, func(t *testing.T, r *lateRig, f *fakeGit) func() {
			c := block(f)
			r.h.send(key(tea.KeyF3, 0))
			if !r.h.m.ui.Explorer.Open {
				t.Fatal("no Explorer")
			}
			return func() { close(c) }
		}},
		{"the branches popup's lists", func() (input.Git, *fakeGit) {
			b := &fakeBranches{fakeGit: &fakeGit{}, b: sampleBranches()}
			return b, b.fakeGit
		}, func(t *testing.T, r *lateRig, f *fakeGit) func() {
			r.h.send(actionMsg{state.BranchRead{Branch: "main"}})
			c := block(f)
			r.h.send(altB)
			if !r.h.m.ui.Branches.Open {
				t.Fatal("no branches popup")
			}
			r.h.send(key(tea.KeyF6, 0)) // the popup closes; its read goes on
			return func() { close(c) }
		}},
		{"the file viewer's blame", nil, nil},
		{"the Open dialog's images", func() (input.Git, *fakeGit) { f := &fakeGit{}; return f, f }, func(t *testing.T, r *lateRig, f *fakeGit) func() {
			hold := make(chan struct{})
			files, err := imagetest.WriteAll(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			file := files["png"]
			r.h.m.opt.PickImages = func() ([]string, error) { <-hold; return []string{file}, nil }
			for _, cmd := range r.h.m.pickImages() {
				r.h.exec(cmd)
			}
			return released(t, hold)
		}},
	}
	var blame *fakeBlame // each run's own, made in its bubble
	rows[4].git = func() (input.Git, *fakeGit) {
		blame = &fakeBlame{fakeGit: &fakeGit{}, hold: true, answer: sampleBlame}
		return blame, blame.fakeGit
	}
	rows[4].start = func(t *testing.T, r *lateRig, f *fakeGit) func() {
		if err := os.WriteFile(filepath.Join(r.dir, "late.go"), []byte("package late\n\nfunc A() {}\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		in, _ := json.Marshal(map[string]string{"file_path": "late.go", "old_string": "x", "new_string": "func A() {}"})
		pushEvents(r.h, claude.ProviderID, evOf(event.ToolStarted, event.Tool{ToolID: "late-edit", Name: "Edit", Input: in}))
		r.h.send(key(tea.KeyUp, 0))
		r.h.send(key(tea.KeyF3, 0))
		if !r.h.m.ui.File.Open {
			t.Fatal("no file viewer")
		}
		r.h.send(typed('b'))
		blame.mu.Lock()
		n := len(blame.gates)
		blame.mu.Unlock()
		if n == 0 {
			t.Fatal("no blame asked")
		}
		pushEvents(r.h, claude.ProviderID, evOf(event.ToolCompleted, event.Tool{ToolID: "late-edit", Name: "Edit"}))
		var once sync.Once
		free := func() { once.Do(func() { blame.release(0) }) }
		t.Cleanup(free)
		return free
	}
	for _, c := range rows {
		for _, crash := range []bool{true, false} {
			t.Run(c.name, func(t *testing.T) {
				synctest.Test(t, func(t *testing.T) {
					git, f := c.git()
					r := newLateRig(t, git)
					release := c.start(t, r, f)
					before := gensOf(r.h.m)
					r.home(t, crash)
					release()
					lateAnswers(t, r.h)
					r.fresh(t, before)
				})
			})
		}
	}
}
