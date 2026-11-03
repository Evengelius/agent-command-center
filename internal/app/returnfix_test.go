package app

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/Evengelius/agent-command-center/internal/input"
	"github.com/Evengelius/agent-command-center/internal/render"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// Plan 75, designs D8 and D9: the "Not now" notice's life, the force
// stop's offer kept, the session's end off the event loop, the ended
// session's late results, git refused during the return's close, and the
// turn review after Back.

// released is the close of c, done once, and at the test's end at the
// latest, before the harness's own end: a failing check never leaves the
// bubble's goroutines blocked on c.
func released(t *testing.T, c chan struct{}) func() {
	var once sync.Once
	free := func() { once.Do(func() { close(c) }) }
	t.Cleanup(free)
	return free
}

// heldTurns is a turn review store whose take number holdAt waits for
// release, whatever its context says, then gives tree; the others give
// "tree-<n>".
type heldTurns struct {
	mu      sync.Mutex
	takes   int
	holdAt  int
	release chan struct{}
	tree    string
	closed  bool
}

func (f *heldTurns) Take(context.Context) (input.TurnTree, error) {
	f.mu.Lock()
	f.takes++
	n := f.takes
	f.mu.Unlock()
	if n == f.holdAt {
		<-f.release
		return input.TurnTree{Tree: f.tree}, nil
	}
	return input.TurnTree{Tree: fmt.Sprintf("tree-%d", n)}, nil
}
func (f *heldTurns) Changing(context.Context) ([]string, error) { return nil, nil }
func (f *heldTurns) Files(string, string) (input.CommitFiles, error) {
	return input.CommitFiles{}, nil
}
func (f *heldTurns) Sides(string, string, input.CommitFile) (input.DiffPair, error) {
	return input.DiffPair{}, nil
}
func (f *heldTurns) Close() error {
	f.mu.Lock()
	f.closed = true
	f.mu.Unlock()
	return nil
}
func (f *heldTurns) isClosed() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.closed
}

// TestReturnNotNowEnds: the "Not now" notice goes once what it names has
// ended, the turn, or the commit (design D8).
func TestReturnNotNowEnds(t *testing.T) {
	const turnNote = "Not now: the turn is still running · Ctrl+C interrupts it"
	synctest.Test(t, func(t *testing.T) {
		w := &wireClaude{}
		h := wireRig(t, w, Options{})
		w.last().emit(claudeInit(h.m.s.Context.Project, "s-1"))
		h.settle()
		h.send(tea.PasteMsg{Content: "go"})
		h.send(key(tea.KeyEnter, 0))
		h.send(esc())
		if h.m.s.Turn.Notice != turnNote || !strings.Contains(h.frame(), turnNote) {
			t.Fatalf("during the turn: notice %q", h.m.s.Turn.Notice)
		}
		w.last().emit(claudeTurn("s-1", "DONE")...)
		h.settle()
		if h.m.s.Turn.Phase != state.Idle || h.m.s.Turn.Notice != "" || strings.Contains(h.frame(), "Not now") {
			t.Errorf("after the turn: turn %q, notice %q:\n%s", h.m.s.Turn.Phase, h.m.s.Turn.Notice, h.frame())
		}
	})
	synctest.Test(t, func(t *testing.T) {
		h, w := commitRig(t)
		typeMessage(t, h, "Slow")
		free := released(t, w.holdOn())
		h.send(ctrlEnter) // the commit, held
		focusB(t, h)
		h.send(esc())
		if n := h.m.s.Turn.Notice; n != "Not now: the commit is still running" {
			t.Fatalf("during the commit: notice %q", n)
		}
		free()
		h.settle()
		if h.m.op != nil || h.m.s.Turn.Notice != "" || strings.Contains(h.frame(), "Not now") {
			t.Errorf("after the commit: running %v, notice %q:\n%s", h.m.op != nil, h.m.s.Turn.Notice, h.frame())
		}
	})
}

// TestReturnForceStopKept: while the force stop is offered, Esc leaves its
// notice, the only place the screen names F (design D8).
func TestReturnForceStopKept(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := &wireClaude{}
		h := wireRig(t, w, Options{})
		w.last().emit(claudeInit(h.m.s.Context.Project, "s-1"))
		h.settle()
		h.send(tea.PasteMsg{Content: "go"})
		h.send(key(tea.KeyEnter, 0))
		w.last().emit(`{"type":"stream_event","event":{"type":"message_start","message":{"id":"m1","role":"assistant","model":"claude-haiku-4-5","content":[]}},"parent_tool_use_id":null,"session_id":"s-1"}`)
		h.settle()
		h.send(ctrl('c'))
		h.send(sessionMsg{gen: h.m.sessGen, msg: actionMsg{state.InterruptTimeout{}}})
		s := h.m.State()
		if s.Turn.Phase != state.Interrupting || !s.Turn.ForceStopOffered {
			t.Fatalf("turn %q, force stop offered %v", s.Turn.Phase, s.Turn.ForceStopOffered)
		}
		want := s.Turn.Notice
		h.send(esc())
		noQuestion(t, h, "the force stop offered")
		if s.Turn.Notice != want || !strings.Contains(h.frame(), "force stop") {
			t.Errorf("Esc: notice %q, want %q:\n%s", s.Turn.Notice, want, h.frame())
		}
	})
}

// offLoop is a session returned home whose end is held: the notification's
// removal blocked until releaseNotify, the turn's end snapshot held by
// first until its release.
type offLoop struct {
	h                      *harness
	w                      *wireClaude
	first, second          *heldTurns
	entered, releaseNotify chan struct{}
	d                      input.Drafts
	dir                    string
	// freeNotify and freeLate release the notification's removal and the
	// held end snapshot, once, and at the test's end at the latest.
	freeNotify, freeLate func()
}

func offLoopRig(t *testing.T) *offLoop {
	t.Helper()
	r := &offLoop{dir: t.TempDir(), d: input.Drafts{Dir: t.TempDir()}, w: &wireClaude{}, entered: make(chan struct{}), releaseNotify: make(chan struct{})}
	h := homeRig(t, r.w, &wireCodex{}, &input.Projects{Dir: t.TempDir()}, r.dir)
	r.h = h
	r.first = &heldTurns{holdAt: 2, release: make(chan struct{}), tree: "LATE-END"}
	r.second = &heldTurns{}
	r.freeNotify, r.freeLate = released(t, r.releaseNotify), released(t, r.first.release)
	stores := []TurnStore{r.first, r.second}
	var mu sync.Mutex
	h.m.opt.TurnStore = func(string) (TurnStore, error) {
		mu.Lock()
		defer mu.Unlock()
		if len(stores) == 0 {
			return &heldTurns{}, nil
		}
		st := stores[0]
		stores = stores[1:]
		return st, nil
	}
	h.m.opt.Drafts = &r.d
	var once sync.Once
	h.m.opt.Notify = func(string, string) error { return nil }
	h.m.opt.Unnotify = func() error { once.Do(func() { close(r.entered) }); <-r.releaseNotify; return nil }

	h.send(key(tea.KeyEnter, 0))
	r.w.last().emit(claudeInit(r.dir, "s-1"))
	h.settle()
	h.send(tea.PasteMsg{Content: "first"})
	h.send(key(tea.KeyEnter, 0))
	h.send(tea.BlurMsg{}) // the turn's end rings: the notification goes up
	r.w.last().emit(claudeTurn("s-1", "ONE")...)
	h.settle()
	if !h.m.ping.notified || h.m.s.Stages[0].Review.From != "tree-1" || h.m.s.Stages[0].Review.To != "" {
		t.Fatalf("the first session: notified %v, review %+v", h.m.ping.notified, h.m.s.Stages[0].Review)
	}
	h.send(tea.PasteMsg{Content: "kept draft"})
	h.send(esc())
	h.send(key(tea.KeyLeft, 0))
	h.send(key(tea.KeyEnter, 0))
	return r
}

// TestReturnOffLoop is the session's end off the event loop: the home
// screen comes while the notification's removal and the end snapshot are
// still blocked; START SESSION, Ctrl+R and the history's resume wait for it
// and say so; once released the end is done (the notification out, the
// store removed, the draft written), and a start works. The snapshot held
// past the return, answered at home, leaves the state New's but for the
// answers.
func TestReturnOffLoop(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := offLoopRig(t)
		h := r.h
		select {
		case <-r.entered:
		default:
			t.Fatal("the notification's removal was not asked")
		}
		if s := h.m.State(); s.Screen != state.ScreenStartup || s.Startup.Phase != state.Asking || h.m.ending == nil {
			t.Fatalf("with the end blocked: screen %q, phase %q, ending %v", s.Screen, s.Startup.Phase, h.m.ending != nil)
		}
		h.send(key(tea.KeyEnter, 0))
		h.send(ctrl('r'))
		if sp := r.w.spawns(); len(sp) != 1 || !strings.Contains(h.frame(), stillClosingNote) {
			t.Fatalf("a start while the end runs: %d spawns\n%s", len(sp), h.frame())
		}
		openHist(t, h)
		h.send(key(tea.KeyEnter, 0))
		if h.m.hist == nil || h.m.hist.note != stillClosingNote || len(r.w.spawns()) != 1 {
			t.Fatalf("the history's resume while the end runs: view %v, spawns %d", h.m.hist != nil, len(r.w.spawns()))
		}
		h.send(esc())

		r.freeNotify()
		time.Sleep(4 * time.Second) // the snapshot worker's 3 s, the end snapshot still held
		h.settle()
		if h.m.ending != nil || !r.first.isClosed() {
			t.Fatalf("after the release: ending %v, store removed %v", h.m.ending != nil, r.first.isClosed())
		}
		if got := draftFile(t, r.d, r.dir); string(got) != "kept draft" {
			t.Errorf("the stored draft %q", got)
		}
		if strings.Contains(h.frame(), stillClosingNote) {
			t.Error("the note outlived the end")
		}

		// The held end snapshot answers at home: nothing of it reaches the state.
		r.freeLate()
		h.settle()
		want := *state.New()
		want.Startup = h.m.s.Startup
		if !reflect.DeepEqual(*h.m.s, want) {
			t.Errorf("after the late snapshot the state is not New's but for the answers:\n%+v", *h.m.s)
		}
		h.send(key(tea.KeyEnter, 0))
		if sp := r.w.spawns(); len(sp) != 2 {
			t.Errorf("a start once the end is done: %d spawns", len(sp))
		}
	})
}

// TestReturnLateSnapshot is the late result's other side: the first
// session's end snapshot answers while the second session's turn 1 runs,
// and its review does not take it.
func TestReturnLateSnapshot(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := offLoopRig(t)
		h := r.h
		r.freeNotify()
		time.Sleep(4 * time.Second)
		h.settle()
		if h.m.ending != nil {
			t.Fatal("the end not done")
		}
		h.send(key(tea.KeyEnter, 0))
		r.w.last().emit(claudeInit(r.dir, "s-2"))
		h.settle()
		h.m.edit.Clear()
		h.send(tea.PasteMsg{Content: "second"})
		h.send(key(tea.KeyEnter, 0))
		s := h.m.State()
		if len(s.Stages) != 1 || s.Stages[0].Review.From != "tree-1" || s.Stages[0].Review.To != "" {
			t.Fatalf("the second session's turn 1: %+v", s.Stages)
		}
		r.freeLate()
		h.settle()
		if rv := s.Stages[0].Review; rv.To != "" || rv.Done || len(s.Notes) != 0 {
			t.Errorf("the late snapshot reached the second session: review %+v, notes %q", rv, s.Notes)
		}
		r.w.last().emit(claudeTurn("s-2", "TWO")...)
		h.settle()
		if rv := s.Stages[0].Review; rv.To != "tree-2" {
			t.Errorf("the second session's own end: %+v", rv)
		}
	})
}

// TestReturnLateWrite: a write's failure of a session that ended changes
// nothing of the state, and the toast says it.
func TestReturnLateWrite(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := liveDash(t, nil)
		gen := h.m.sessGen
		h.m.sessGen++ // as a return does
		h.send(sessionMsg{gen: gen, msg: actionMsg{state.WriteFailed{Message: "broken pipe"}}})
		h.send(sessionMsg{gen: gen, msg: actionMsg{state.TurnSnapshot{Turn: 9, End: true, Tree: "late"}}})
		if s := h.m.State(); s.Turn.Notice != "" || len(s.Notes) != 0 || !strings.Contains(h.m.ui.Toast, "broken pipe") {
			t.Errorf("late messages: notice %q, notes %q, toast %q", s.Turn.Notice, s.Notes, h.m.ui.Toast)
		}
	})
}

// TestReturnNoGitWhileClosing: while the return closes the session, no git
// operation starts, each saying so; Ctrl+Q still quits.
func TestReturnNoGitWhileClosing(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, w := commitRig(t)
		typeMessage(t, h, "During the close")
		focusB(t, h)
		h.send(esc())
		h.send(key(tea.KeyLeft, 0))
		h.send(key(tea.KeyEnter, 0))
		if s := h.m.State(); s.Session.Phase != state.Closing || !s.Returning {
			t.Fatalf("Yes: session %q, returning %v", s.Session.Phase, s.Returning)
		}
		clickItem(t, h, render.ItemMsg, 0, 0)
		h.send(ctrlEnter)
		if h.m.op != nil || len(w.logged("commit")) != 0 || h.m.ui.Source.Notice != returningNote {
			t.Errorf("a commit during the close: running %v, writes %q, note %q", h.m.op != nil, w.log, h.m.ui.Source.Notice)
		}
		var leave []time.Duration
		var mu sync.Mutex
		h.m.opt.After = func(d time.Duration, msg tea.Msg) tea.Cmd {
			if _, ok := msg.(leaveMsg); ok {
				mu.Lock()
				leave = append(leave, d)
				mu.Unlock()
			}
			return nil
		}
		h.send(ctrlQ)
		mu.Lock()
		defer mu.Unlock()
		if h.m.State().Startup.Phase != state.Exited || len(leave) != 1 || leave[0] != KillGrace {
			t.Errorf("Ctrl+Q during the close: startup %q, leave timers %v", h.m.State().Startup.Phase, leave)
		}
	})
	synctest.Test(t, func(t *testing.T) {
		r, f, _ := branchesRig(t)
		h := r.h
		h.send(esc())
		h.send(key(tea.KeyLeft, 0))
		h.send(key(tea.KeyEnter, 0))
		if !h.m.State().Returning {
			t.Fatalf("Yes: session %q", h.m.State().Session.Phase)
		}
		openBranches(t, h)
		h.send(key(tea.KeyF5, 0))
		if h.m.fetch != nil || len(f.called("fetch")) != 0 || h.m.ui.Branches.Note != returningNote {
			t.Errorf("a fetch during the close: running %v, calls %v, note %q", h.m.fetch != nil, f.called("fetch"), h.m.ui.Branches.Note)
		}
		branchTo(t, h, "b:1:feat")
		h.send(key(tea.KeyEnter, 0))
		h.settle()
		if h.m.sw != nil || len(f.called("switch")) != 0 || h.m.ui.Branches.Note != returningNote {
			t.Errorf("a switch during the close: %+v, calls %v, note %q", h.m.sw, f.called("switch"), h.m.ui.Branches.Note)
		}
	})
}

// TestBackTurnStore: Back after a failed start removes its turn review
// store, and a start in another project then reviews its turns.
func TestBackTurnStore(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var mu sync.Mutex
		var dirs []string
		var stores []*fakeTurns
		open := func(dir string) (TurnStore, error) {
			mu.Lock()
			defer mu.Unlock()
			st := &fakeTurns{trees: []string{"t1", "t2", "t3"}}
			dirs, stores = append(dirs, dir), append(stores, st)
			return st, nil
		}
		w := &wireClaude{noInit: true}
		h := wireRig(t, w, Options{TurnStore: open})
		h.send(sessionMsg{gen: h.m.sessGen, msg: actionMsg{state.HandshakeTimeout{}}})
		if h.m.State().Startup.Phase != state.StartFailed {
			t.Fatalf("startup %q", h.m.State().Startup.Phase)
		}
		h.send(esc())
		h.settle()
		mu.Lock()
		if len(stores) != 1 || !stores[0].closed {
			t.Errorf("after Back: %d stores, the first removed %v", len(stores), len(stores) == 1 && stores[0].closed)
		}
		mu.Unlock()
		other := t.TempDir()
		w.mu.Lock()
		w.noInit = false
		w.mu.Unlock()
		h.m.path.SetText(other)
		h.send(key(tea.KeyEnter, 0))
		h.settle()
		w.last().emit(claudeInit(other, "s-2"))
		h.settle()
		h.send(tea.PasteMsg{Content: "review me"})
		h.send(key(tea.KeyEnter, 0))
		w.last().emit(claudeTurn("s-2", "OK")...)
		h.settle()
		mu.Lock()
		defer mu.Unlock()
		s := h.m.State()
		if len(dirs) != 2 || !state.SamePath(dirs[1], other) || !s.Reviewing.On || len(s.Stages) != 1 || s.Stages[0].Review.From == "" || s.Stages[0].Review.To == "" {
			t.Errorf("the start in %s: stores for %q, reviewing %v, stages %+v", other, dirs, s.Reviewing.On, s.Stages)
		}
	})
}
