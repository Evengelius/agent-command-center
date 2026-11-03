package app

import (
	"bytes"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/Evengelius/agent-command-center/internal/input"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// Plan 75, designs D7 to D9: the Worktrees view's start waits for the last
// session's end; the staging question keeps Esc; the program's end runs a
// session's end not yet started; a forced kill with no end seen still leads
// home; an approval waiting in the queue refuses the return.

// TestReturnWorktreesStart: Enter on a worktree in the Worktrees view, on the
// home screen, waits for the last session's end, saying so; once it is done
// the start goes, its draft back.
func TestReturnWorktreesStart(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := offLoopRig(t)
		h := r.h
		if h.m.ending == nil || h.m.State().Screen != state.ScreenStartup {
			t.Fatalf("the end not held: ending %v, screen %q", h.m.ending != nil, h.m.State().Screen)
		}
		h.m.opt.Git = &fakeGit{rows: []input.Worktree{{Path: r.dir, Branch: "main", Main: true}}}
		h.send(ctrl('w'))
		if h.m.wt == nil || len(h.m.wt.rows) != 1 {
			t.Fatalf("the Worktrees view: %+v", h.m.wt)
		}
		h.send(key(tea.KeyEnter, 0))
		if sp := r.w.spawns(); len(sp) != 1 || h.m.wt == nil || !strings.Contains(h.frame(), stillClosingNote) {
			t.Fatalf("Enter while the end runs: %d spawns, view %v:\n%s", len(sp), h.m.wt != nil, h.frame())
		}
		r.freeNotify()
		time.Sleep(4 * time.Second)
		h.settle()
		if h.m.ending != nil {
			t.Fatal("the end not done")
		}
		h.send(key(tea.KeyEnter, 0))
		if sp := r.w.spawns(); len(sp) != 2 {
			t.Fatalf("Enter once the end is done: %d spawns", len(sp))
		}
		r.w.last().emit(claudeInit(r.dir, "s-2"))
		h.settle()
		if h.m.edit.Text() != "kept draft" {
			t.Errorf("the draft at the worktree's start: %q", h.m.edit.Text())
		}
		r.freeLate()
		h.settle()
	})
}

// TestReturnStagingQuestion: with the staging question open and the focus
// moved to zone B by F6, Esc opens no question to return home and leaves the
// staging question as it was.
func TestReturnStagingQuestion(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, _ := commitRigOn(t, "")
		stagingAsked(t, h)
		ask := h.m.ui.Source.Ask
		focusB(t, h)
		h.send(esc())
		noQuestion(t, h, "the staging question open")
		if h.m.ui.Source.Ask != ask || !strings.Contains(h.frame(), "Nothing is staged. Stage") {
			t.Errorf("the staging question: %d, was %d:\n%s", h.m.ui.Source.Ask, ask, h.frame())
		}
	})
}

// stagingAsked commits what is staged, then asks to commit with nothing
// staged: the staging question opens.
func stagingAsked(t *testing.T, h *harness) {
	t.Helper()
	typeMessage(t, h, "Staged only")
	h.send(ctrlEnter)
	h.settle()
	typeMessage(t, h, "All of it")
	h.send(ctrlEnter)
	if h.m.ui.Source.Ask == 0 {
		t.Fatal("no staging question")
	}
}

// TestReturnShutdownRunsEnd: the program ends right after the update that
// went home, before its end's command ran: Shutdown runs the end itself, its
// waits bounded, the store removed and the draft written.
func TestReturnShutdownRunsEnd(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		dir := t.TempDir()
		d := input.Drafts{Dir: t.TempDir()}
		w := &wireClaude{}
		h := homeRig(t, w, &wireCodex{}, &input.Projects{Dir: t.TempDir()}, dir)
		turns := &fakeTurns{trees: []string{"t1", "t2"}}
		h.m.opt.Drafts = &d
		h.m.opt.TurnStore = func(string) (TurnStore, error) { return turns, nil }
		h.send(key(tea.KeyEnter, 0))
		w.last().emit(claudeInit(dir, "s-1"))
		h.settle()
		h.send(tea.PasteMsg{Content: "at the end"})
		w.last().exit(1, "")
		h.settle()
		h.send(esc())
		h.send(key(tea.KeyLeft, 0))
		h.m.Update(key(tea.KeyEnter, 0)) // home at once; its commands, the end among them, never run
		if h.m.State().Screen != state.ScreenStartup || h.m.ending == nil {
			t.Fatalf("screen %q, ending %v", h.m.State().Screen, h.m.ending != nil)
		}
		h.m.Shutdown()
		if !turns.closed {
			t.Error("the store not removed at the program's end")
		}
		if got := draftFile(t, d, dir); !bytes.Equal(got, []byte("at the end")) {
			t.Errorf("the stored draft %q", got)
		}
	})
}

// TestReturnKillGrace: during a return, a forced kill that yields no session
// end still leads home after the quit's kill grace, the toast saying the
// CLI's end was not seen.
func TestReturnKillGrace(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		release := make(chan struct{})
		defer close(release)
		w := &wireClaude{}
		w.onEOF = func() { <-release }
		h := wireRig(t, w, Options{})
		p := w.last()
		p.deaf.Store(true)
		defer p.deaf.Store(false)
		var mu sync.Mutex
		var timers []sessionMsg
		var waits []time.Duration
		h.m.opt.After = func(d time.Duration, msg tea.Msg) tea.Cmd {
			if sm, ok := msg.(sessionMsg); ok {
				mu.Lock()
				timers, waits = append(timers, sm), append(waits, d)
				mu.Unlock()
			}
			return nil
		}
		taken := func() (sessionMsg, time.Duration) {
			mu.Lock()
			defer mu.Unlock()
			if len(timers) == 0 {
				t.Fatal("no timer asked")
			}
			sm, d := timers[len(timers)-1], waits[len(waits)-1]
			timers, waits = nil, nil
			return sm, d
		}
		yesReturn(h)
		sm, d := taken()
		if d != CloseTimeout || sm.msg != (actionMsg{state.CloseTimeout{}}) {
			t.Fatalf("the close's timer: %v %+v", d, sm.msg)
		}
		h.send(sm) // the forced kill: nothing ends
		if s := h.m.State(); s.Session.Phase != state.Closing || s.Screen != state.ScreenDashboard {
			t.Fatalf("after the kill: session %q, screen %q", s.Session.Phase, s.Screen)
		}
		sm, d = taken()
		if d != KillGrace {
			t.Fatalf("the grace's timer: %v %+v", d, sm.msg)
		}
		h.send(sm)
		if s := h.m.State(); s.Screen != state.ScreenStartup || s.Startup.Phase != state.Asking || !strings.Contains(h.m.ui.Toast, "end was not seen") {
			t.Errorf("after the grace: screen %q, phase %q, toast %q", s.Screen, s.Startup.Phase, h.m.ui.Toast)
		}
		h.send(sm) // late: nothing more
		if s := h.m.State(); s.Screen != state.ScreenStartup || len(s.Notes) != 0 {
			t.Errorf("a late grace: screen %q, notes %q", s.Screen, s.Notes)
		}
	})
}

// TestReturnKillGraceQuit: a Ctrl+Q pressed during the kill's grace stays
// the quit: the grace's timer, come after it, leads nowhere, and the program
// ends.
func TestReturnKillGraceQuit(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		release := make(chan struct{})
		defer close(release)
		w := &wireClaude{}
		w.onEOF = func() { <-release }
		h := wireRig(t, w, Options{})
		p := w.last()
		p.deaf.Store(true)
		defer p.deaf.Store(false)
		var mu sync.Mutex
		var asked []tea.Msg
		h.m.opt.After = func(d time.Duration, msg tea.Msg) tea.Cmd {
			mu.Lock()
			asked = append(asked, msg)
			mu.Unlock()
			return nil
		}
		find := func(want func(tea.Msg) bool) tea.Msg {
			mu.Lock()
			defer mu.Unlock()
			for i := len(asked) - 1; i >= 0; i-- {
				if want(asked[i]) {
					return asked[i]
				}
			}
			t.Fatal("no such timer asked")
			return nil
		}
		yesReturn(h)
		closeTimer := find(func(m tea.Msg) bool {
			sm, ok := m.(sessionMsg)
			return ok && sm.msg == (actionMsg{state.CloseTimeout{}})
		})
		h.send(closeTimer) // the forced kill: nothing ends
		grace := find(func(m tea.Msg) bool { sm, ok := m.(sessionMsg); _, g := sm.msg.(returnGraceMsg); return ok && g })
		h.send(ctrlQ)
		if s := h.m.State(); s.Startup.Phase != state.Exited {
			t.Fatalf("Ctrl+Q during the grace: startup %q", s.Startup.Phase)
		}
		leave := find(func(m tea.Msg) bool { _, ok := m.(leaveMsg); return ok })
		h.send(grace)
		if s := h.m.State(); s.Startup.Phase != state.Exited || s.Screen != state.ScreenDashboard || h.m.ending != nil {
			t.Errorf("the grace after the quit: startup %q, screen %q, an end run %v", s.Startup.Phase, s.Screen, h.m.ending != nil)
		}
		h.send(leave)
		if !h.quit {
			t.Error("the program did not quit")
		}
	})
}

// TestReturnQueuedApproval: an approval waiting in the queue, with no turn
// and no agent running, refuses the return: no question, nothing answered,
// the input row saying why.
func TestReturnQueuedApproval(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := liveDash(t, nil)
		// No CLI event leaves it so (an agent's end expires its requests): the
		// queue is given the request as the state keeps one.
		s := h.m.State()
		s.Approvals["q-1"] = &state.Approval{ID: "q-1", Tool: "Bash", Status: state.Pending}
		s.Queue = append(s.Queue, "q-1")
		if s.Running() != "" || s.Turn.Phase != state.Idle || s.Approvals["q-1"] == nil || s.Approvals["q-1"].Status != state.Pending {
			t.Fatalf("running %q, turn %q, approval %+v", s.Running(), s.Turn.Phase, s.Approvals["q-1"])
		}
		h.send(esc())
		noQuestion(t, h, "an approval waiting")
		const note = "Not now: an approval is waiting · F2 answers it"
		if s.Turn.Notice != note || !strings.Contains(h.frame(), note) || s.Approvals["q-1"].Status != state.Pending || s.Session.Phase != state.Ready {
			t.Errorf("notice %q, approval %q, session %q", s.Turn.Notice, s.Approvals["q-1"].Status, s.Session.Phase)
		}
	})
}

// TestReturnNoWorktreeOpWhileClosing: while the return closes the CLI, the
// Worktrees view's merge and removal start nothing, each saying so, so that
// none is dropped unseen.
func TestReturnNoWorktreeOpWhileClosing(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		release := make(chan struct{})
		defer close(release)
		w := &wireClaude{}
		w.onEOF = func() { <-release } // the CLI held open
		h := wireRig(t, w, Options{})
		proj := h.m.State().Context.Project
		fake := &fakeGit{rows: []input.Worktree{
			{Path: proj, Branch: "main", Main: true},
			{Path: proj + "-feature", Branch: "feature", Ahead: 2},
		}}
		h.m.opt.Git = fake
		yesReturn(h)
		if s := h.m.State(); s.Session.Phase != state.Closing || !s.Returning {
			t.Fatalf("Yes: session %q, returning %v", s.Session.Phase, s.Returning)
		}
		h.send(ctrl('w'))
		if h.m.wt == nil || len(h.m.wt.rows) != 2 {
			t.Fatalf("the Worktrees view: %+v", h.m.wt)
		}
		h.send(key(tea.KeyDown, 0))
		for _, op := range []rune{'m', 'x'} {
			h.send(typed(op))
			if h.m.wt.mode != byte(op) {
				t.Fatalf("%c: no question, note %q", op, h.m.wt.note)
			}
			h.send(typed('y'))
			if h.m.wt.note != returningNote || !h.m.wt.failed || h.m.wt.busy != "" || !strings.Contains(h.frame(), returningNote) {
				t.Errorf("%c then y during the close: note %q, busy %q", op, h.m.wt.note, h.m.wt.busy)
			}
		}
		if got := append(fake.called("merge"), fake.called("remove")...); len(got) != 0 {
			t.Errorf("git asked during the close: %q", got)
		}
	})
}
