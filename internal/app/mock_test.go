package app

import (
	"context"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/Evengelius/agent-command-center/internal/event"
	"github.com/Evengelius/agent-command-center/internal/provider"
	"github.com/Evengelius/agent-command-center/internal/render"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// testNow is the fixed clock of the application tests.
var testNow = time.Date(2026, 10, 15, 9, 30, 0, 0, time.UTC)

var ansi = regexp.MustCompile("\x1b(\\[[0-9;:?<>=]*[ -/]*[@-~]|\\][^\x07\x1b]*(\x07|\x1b\\\\))")

// plainText strips the terminal escapes from a frame.
func plainText(s string) string { return ansi.ReplaceAllString(s, "") }

// harness runs the model as Bubble Tea would, without a terminal: each
// command runs in its own goroutine and its message is delivered in turn.
// A command that waits (a listen on a quiet session) ends when the test's
// cleanup closes the session.
type harness struct {
	t     *testing.T
	m     *Model
	msgs  chan tea.Msg
	quit  bool
	check func(frame string) // called on every frame
}

func newHarness(t *testing.T, opt Options) *harness {
	opt.Now = func() time.Time { return testNow }
	opt.After = func(time.Duration, tea.Msg) tea.Cmd { return nil } // no timers
	h := &harness{t: t, m: New(opt), msgs: make(chan tea.Msg, 4096)}
	h.exec(h.m.Init())
	t.Cleanup(func() {
		// What Run does at the end, and the write queue's worker stopped:
		// a bubble (synctest) ends only once its goroutines have.
		h.m.Shutdown()
		if h.m.writes != nil {
			close(h.m.writes)
		}
	})
	return h
}

func (h *harness) exec(c tea.Cmd) {
	if c != nil {
		go func() { h.msgs <- c() }()
	}
}

// send delivers one message, then every message the commands produce until
// none arrives for a while.
func (h *harness) send(msg tea.Msg) {
	h.deliver(msg)
	h.settle()
}

func (h *harness) deliver(msg tea.Msg) {
	switch msg := msg.(type) {
	case nil:
		return
	case tea.BatchMsg:
		for _, c := range msg {
			h.exec(c)
		}
		return
	case tea.QuitMsg:
		h.quit = true
		return
	}
	_, cmd := h.m.Update(msg)
	if h.check != nil {
		h.check(h.m.View().Content)
	}
	h.exec(cmd)
}

// settle delivers messages until none arrives for 300 ms. In a bubble
// (synctest.Test) that time passes as soon as every goroutine of the test is
// blocked, so it returns at once when nothing more can come; the tests that
// time real work (perf_test.go, TestDraftKeystrokes) run outside a bubble and
// wait for real.
func (h *harness) settle() {
	for {
		select {
		case msg := <-h.msgs:
			h.deliver(msg)
		case <-time.After(300 * time.Millisecond):
			return
		}
	}
}

func key(code rune, mod tea.KeyMod) tea.KeyPressMsg { return tea.KeyPressMsg{Code: code, Mod: mod} }

// spawnTrap is an agent CLI adapter that fails the test if anything reaches
// it: a mock run must not detect or start a CLI.
type spawnTrap struct{ t *testing.T }

func (p spawnTrap) ID() string          { return "claude-code" }
func (p spawnTrap) DisplayName() string { return "Claude Code" }
func (p spawnTrap) Detect(context.Context) provider.Detection {
	p.t.Error("mock run called Detect on an agent CLI adapter")
	return provider.Detection{ID: "claude-code", Installed: true}
}
func (p spawnTrap) Start(context.Context, provider.SessionContext) (provider.Session, error) {
	p.t.Error("mock run started an agent CLI session")
	return nil, context.Canceled
}

// TestMockMode is test 8 (proposal step 0, item 8): no process, every event
// of origin mock, the badge on every frame.
func TestMockMode(t *testing.T) {
	for _, file := range []string{"claude-subagent-approval.jsonl", "codex-subagents.jsonl"} {
		t.Run(file, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				frames := 0
				h := newHarness(t, Options{
					Config:    Config{Mock: filepath.Join("..", "..", "testdata", "mock", file)},
					Providers: []provider.Provider{spawnTrap{t}},
					Cwd:       t.TempDir(),
				})
				h.check = func(frame string) {
					frames++
					if !strings.Contains(plainText(frame), render.MockBadge) {
						t.Fatalf("frame %d has no %q:\n%s", frames, render.MockBadge, plainText(frame))
					}
				}
				h.check(h.m.View().Content) // the first frame, drawn before any message
				h.send(tea.WindowSizeMsg{Width: 120, Height: 40})
				s := h.m.State()
				if s.Screen != state.ScreenDashboard || !s.Context.Mock {
					t.Fatalf("screen %q, mock %v: want the dashboard in mock mode", s.Screen, s.Context.Mock)
				}
				replayed := len(s.Log)
				if replayed == 0 {
					t.Fatal("no event replayed")
				}

				// A message goes to the mock session, never to a CLI.
				h.send(tea.PasteMsg{Content: "hello\nmock"})
				h.send(key(tea.KeyEnter, 0))
				if len(s.Log) <= replayed {
					t.Errorf("sending added no event (log %d)", len(s.Log))
				}
				for _, size := range [][2]int{{80, 24}, {200, 60}, {79, 24}, {120, 40}} {
					h.send(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
				}

				// Ctrl+Q closes the mock session and leaves.
				h.send(key('q', tea.ModCtrl))
				if !h.quit {
					t.Fatalf("Ctrl+Q did not quit; session %q, startup %q", s.Session.Phase, s.Startup.Phase)
				}
				if s.Session.Phase != state.Closed {
					t.Errorf("session %q after quitting, want closed", s.Session.Phase)
				}
				for _, ev := range s.Log {
					if ev.Origin != event.Mock {
						t.Fatalf("event %d (%s) has origin %q, want mock", ev.Seq, ev.Kind, ev.Origin)
					}
				}
				if last := s.Log[len(s.Log)-1]; last.Kind != event.SessionEnded {
					t.Errorf("last event %s, want session.ended", last.Kind)
				}
				t.Logf("%d replayed events, %d in all, %d frames", replayed, len(s.Log), frames)
			})
		})
	}
}

func TestParseArgs(t *testing.T) {
	good := map[string]Config{
		"":                                       {},
		"--mock x.jsonl":                         {Mock: "x.jsonl"},
		"--provider claude-code":                 {Provider: "claude-code"},
		"--provider codex --project C:\\work\\p": {Provider: "codex", Project: "C:\\work\\p"},
	}
	for args, want := range good {
		got, err := ParseArgs(strings.Fields(args))
		if err != nil || got != want {
			t.Errorf("%q: %+v, %v; want %+v", args, got, err, want)
		}
	}
	bad := map[string]string{
		"--mock x.jsonl --provider claude-code": "--mock runs alone",
		"--mock x.jsonl --project p":            "--mock runs alone",
		"--provider mock":                       "use claude-code or codex",
		"--provider gemini":                     "use claude-code or codex",
		"--mock x.jsonl extra":                  "unexpected argument",
		"--unknown":                             "usage:",
	}
	for args, want := range bad {
		if _, err := ParseArgs(strings.Fields(args)); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%q: error %v, want one containing %q", args, err, want)
		}
	}
}
