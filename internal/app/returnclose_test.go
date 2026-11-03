package app

import (
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/Evengelius/agent-command-center/internal/input"
	"github.com/Evengelius/agent-command-center/internal/provider/claude"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// Plan 75: the return's close (criterion 75.13), Esc on the home screen
// (criterion 75.15), and Alt+H under the folder dialog (design D2).

// yesReturn asks the question and answers Yes.
func yesReturn(h *harness) {
	h.send(esc())
	h.send(key(tea.KeyLeft, 0))
	h.send(key(tea.KeyEnter, 0))
}

// TestReturnEscalation is criterion 75.13's close: a CLI that does not exit
// when its stdin closes is terminated 5 s later by the close's escalation,
// as at quit, and the home screen comes once it has exited; before then the
// dashboard stays, its session closing. The close's own timer, 5 s, is the
// forced kill's, as at quit; one of an ended session does nothing.
func TestReturnEscalation(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		release := make(chan struct{})
		defer close(release)
		w := &wireClaude{}
		w.onEOF = func() { <-release } // stdin read to its end, and no exit
		h := wireRig(t, w, Options{})
		yesReturn(h)
		time.Sleep(4 * time.Second)
		h.settle()
		if s := h.m.State(); s.Screen != state.ScreenDashboard || s.Session.Phase != state.Closing {
			t.Fatalf("4 s after Yes: screen %q, session %q; want the dashboard, closing", s.Screen, s.Session.Phase)
		}
		if f := h.frame(); !strings.Contains(f, "the session is closing") {
			t.Errorf("the input row while it closes:\n%s", f)
		}
		time.Sleep(2 * time.Second)
		h.settle()
		if s := h.m.State(); s.Screen != state.ScreenStartup || s.Startup.Phase != state.Asking {
			t.Errorf("6 s after Yes: screen %q, phase %q; want the home screen, the CLI terminated", s.Screen, s.Startup.Phase)
		}
	})
	synctest.Test(t, func(t *testing.T) {
		release := make(chan struct{})
		defer close(release)
		w := &wireClaude{}
		w.onEOF = func() { <-release }
		h := wireRig(t, w, Options{})
		var timers []sessionMsg
		var mu sync.Mutex
		h.m.opt.After = func(d time.Duration, msg tea.Msg) tea.Cmd {
			if sm, ok := msg.(sessionMsg); ok && d == CloseTimeout {
				mu.Lock()
				timers = append(timers, sm)
				mu.Unlock()
			}
			return nil
		}
		yesReturn(h)
		mu.Lock()
		got := slices.Clone(timers)
		mu.Unlock()
		if len(got) != 1 || got[0].gen != h.m.sessGen || got[0].msg != (actionMsg{state.CloseTimeout{}}) {
			t.Fatalf("the close's timers %+v", got)
		}
		h.send(got[0])
		if s := h.m.State(); s.Screen != state.ScreenStartup || s.Startup.Phase != state.Asking {
			t.Errorf("after the forced kill: screen %q, phase %q; want the home screen", s.Screen, s.Startup.Phase)
		}
		// A timer of the ended session, come late, does nothing.
		h.send(got[0])
		if s := h.m.State(); s.Screen != state.ScreenStartup || len(s.Notes) != 0 {
			t.Errorf("a late timer: screen %q, notes %q", s.Screen, s.Notes)
		}
	})
}

// TestReturnCrashMockQuit is the rest of criterion 75.13: after a crash Yes
// returns at once; a mock run asks nothing; Ctrl+Q while the session closes
// quits.
func TestReturnCrashMockQuit(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := &wireClaude{}
		h := wireRig(t, w, Options{})
		w.last().exit(1, "boom\n")
		h.settle()
		if s := h.m.State(); s.Session.Phase != state.Crashed || s.Startup.Phase != state.StartReady {
			t.Fatalf("session %q, startup %q; want a crash after the start", s.Session.Phase, s.Startup.Phase)
		}
		h.send(esc())
		if !h.m.ui.Return.Open {
			t.Fatalf("Esc after the crash: no question\n%s", h.frame())
		}
		h.send(key(tea.KeyLeft, 0))
		_, cmd := h.m.Update(key(tea.KeyEnter, 0)) // one update, its commands not run yet: at once
		if s := h.m.State(); s.Screen != state.ScreenStartup || s.Startup.Phase != state.Asking || s.Startup.Provider != claude.ProviderID {
			t.Errorf("Yes after the crash: screen %q, phase %q; want the home screen at once", s.Screen, s.Startup.Phase)
		}
		h.exec(cmd)
		h.settle()
		if sp := w.spawns(); len(sp) != 1 {
			t.Errorf("%d spawns: the return started a CLI", len(sp))
		}
	})
	synctest.Test(t, func(t *testing.T) {
		h := newHarness(t, Options{Config: Config{Mock: filepath.Join("..", "..", "testdata", "mock", "claude-subagent-approval.jsonl")}})
		h.settle()
		h.m.s.Turn.Notice = ""
		h.send(esc())
		noQuestion(t, h, "a mock run")
		if s := h.m.State(); s.Screen != state.ScreenDashboard || s.Session.Phase != state.Ready || s.Turn.Notice != "" {
			t.Errorf("Esc in a mock run: screen %q, session %q, notice %q", s.Screen, s.Session.Phase, s.Turn.Notice)
		}
		if s := h.m.State(); !slices.ContainsFunc(s.Notes, func(n string) bool { return strings.Contains(n, "a mock run ends with Ctrl+Q") }) {
			t.Errorf("notes %q", s.Notes)
		}
	})
	synctest.Test(t, func(t *testing.T) {
		release := make(chan struct{})
		defer close(release)
		w := &wireClaude{}
		w.onEOF = func() { <-release }
		h := wireRig(t, w, Options{})
		yesReturn(h)
		if s := h.m.State(); s.Session.Phase != state.Closing || !s.Returning {
			t.Fatalf("Yes: session %q, returning %v", s.Session.Phase, s.Returning)
		}
		h.send(ctrlQ)
		if !h.quit {
			t.Errorf("Ctrl+Q while the session closes: no quit; session %q, startup %q", h.m.State().Session.Phase, h.m.State().Startup.Phase)
		}
	})
}

// TestHomeEsc is criterion 75.15: Esc on the home screen does nothing, the
// form as it was; Ctrl+Q quits.
func TestHomeEsc(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		dir := t.TempDir()
		h := homeRig(t, &wireClaude{}, &wireCodex{}, &input.Projects{Dir: t.TempDir()}, dir)
		for _, field := range []int{0, 1, 2} {
			h.m.ui.Field = field
			h.send(esc())
			if s := h.m.State(); h.quit || s.Startup.Phase != state.Asking || s.Screen != state.ScreenStartup || h.m.path.Text() != dir || h.m.ui.Field != field {
				t.Errorf("Esc on field %d: quit %v, phase %q, field %q, focus %d", field, h.quit, s.Startup.Phase, h.m.path.Text(), h.m.ui.Field)
			}
		}
		h.send(ctrlQ)
		if !h.quit || h.m.State().Startup.Phase != state.Exited {
			t.Errorf("Ctrl+Q on the home screen: quit %v, phase %q", h.quit, h.m.State().Startup.Phase)
		}
	})
}

// TestHistoryFolderDialog is design D2's Alt+H: it does nothing
// while the folder dialog is open, so that a folder picked meanwhile cannot
// change the field under an open history view; once it is closed, Alt+H
// opens the view for the folder picked.
func TestHistoryFolderDialog(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		dir, other := t.TempDir(), t.TempDir()
		h, _, _, _ := histRig(t, histProjects(dir, other), dir)
		fp := &folderPicker{answers: []string{other}, hold: make(chan struct{})}
		freeDialog := released(t, fp.hold)
		h.m.opt.PickFolder = fp.pick
		h.send(key('o', tea.ModCtrl))
		if !h.m.folderOpen {
			t.Fatal("Ctrl+O opened no dialog")
		}
		h.send(key('h', tea.ModAlt))
		if h.m.hist != nil || h.m.ui.History.Open {
			t.Error("Alt+H opened the view while the folder dialog is open")
		}
		freeDialog()
		h.settle()
		if h.m.folderOpen || h.m.path.Text() != other || h.m.hist != nil {
			t.Fatalf("after the dialog: open %v, field %q, view %v", h.m.folderOpen, h.m.path.Text(), h.m.hist)
		}
		openHist(t, h)
		if h.m.hist.project != other || !strings.Contains(h.frame(), "ANOTHER PROJECT") {
			t.Errorf("the view's project %q, want the folder picked %q:\n%s", h.m.hist.project, other, h.frame())
		}
	})
	synctest.Test(t, func(t *testing.T) {
		// The dialog opened between Alt+H and its check's answer: the view
		// does not open.
		dir, other := t.TempDir(), t.TempDir()
		h, _, _, _ := histRig(t, histProjects(dir, other), dir)
		hold := make(chan struct{})
		freeCheck := released(t, hold)
		check := h.m.check
		h.m.check = func(raw string, env input.PathEnv) (string, string, string) { <-hold; return check(raw, env) }
		fp := &folderPicker{hold: make(chan struct{})}
		freeDialog := released(t, fp.hold)
		h.m.opt.PickFolder = fp.pick
		h.send(key('h', tea.ModAlt))
		h.send(key('o', tea.ModCtrl))
		freeCheck()
		h.settle()
		if h.m.hist != nil {
			t.Error("the view opened under the folder dialog")
		}
		freeDialog()
		h.settle()
	})
}
