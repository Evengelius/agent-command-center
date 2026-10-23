package app

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"testing/synctest"

	tea "charm.land/bubbletea/v2"

	"github.com/Evengelius/agent-command-center/internal/event"
	"github.com/Evengelius/agent-command-center/internal/fixture"
	"github.com/Evengelius/agent-command-center/internal/provider"
	"github.com/Evengelius/agent-command-center/internal/provider/claude"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// scriptCLI is an installed agent CLI whose session the test drives: it
// records what the application writes and emits the events the test pushes.
type scriptCLI struct {
	id   string
	sess *scriptSession
	mu   sync.Mutex
	runs int
}

func (p *scriptCLI) ID() string          { return p.id }
func (p *scriptCLI) DisplayName() string { return p.id }
func (p *scriptCLI) Detect(context.Context) provider.Detection {
	return provider.Detection{ID: p.id, Name: p.id, Installed: true, Version: "1.0.0", Path: `C:\Tools\cli.exe`}
}
func (p *scriptCLI) Start(context.Context, provider.SessionContext) (provider.Session, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.runs++
	return p.sess, nil
}

func (p *scriptCLI) starts() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.runs
}

type scriptSession struct {
	events     chan event.Event
	close      sync.Once
	mu         sync.Mutex
	sent       []string
	msgs       []provider.Message // the messages sent, their images with them (plan 31)
	answers    []string
	questions  []string            // AnswerQuestion calls: the approval ids
	qanswers   []map[string]string // and their answers
	interrupts int
	auto       bool   // Capabilities().AutoApprovals
	modes      []bool // the SetApprovalMode calls, in order
}

func newScriptSession() *scriptSession { return &scriptSession{events: make(chan event.Event, 4096)} }

func (s *scriptSession) Events() <-chan event.Event { return s.events }
func (s *scriptSession) Send(msg provider.Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sent = append(s.sent, msg.Text)
	s.msgs = append(s.msgs, msg)
	return nil
}
func (s *scriptSession) Answer(id, decision string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.answers = append(s.answers, id+" "+decision)
	return nil
}
func (s *scriptSession) AnswerQuestion(id string, answers map[string]string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.questions = append(s.questions, id)
	s.qanswers = append(s.qanswers, answers)
	return nil
}
func (s *scriptSession) Interrupt() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.interrupts++
	return nil
}
func (s *scriptSession) Refresh() error { return nil }
func (s *scriptSession) Capabilities() provider.Capabilities {
	// Claude Code's limits on images (plan 31), so attachments can be tested.
	return provider.Capabilities{AutoApprovals: s.auto, Images: provider.ImageLimits{MaxCount: 100, MaxBytes: 7_400_000, MaxTotal: 7_400_000, MaxSide: 8000}}
}
func (s *scriptSession) SetApprovalMode(auto bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.modes = append(s.modes, auto)
	return nil
}
func (s *scriptSession) Close(context.Context) error {
	s.close.Do(func() { close(s.events) }) // as a real session ends its events
	return nil
}

func (s *scriptSession) calls() (sent, answers []string, interrupts int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.sent...), append([]string(nil), s.answers...), s.interrupts
}

// startScript opens the dashboard on a scripted session through the startup
// screen, as the owner would.
func startScript(t *testing.T, id string) (*harness, *scriptCLI) {
	t.Helper()
	cli := &scriptCLI{id: id, sess: newScriptSession()}
	h := newHarness(t, Options{Providers: []provider.Provider{cli}, Cwd: t.TempDir()})
	h.settle()
	h.m.ui.Choice = id
	h.send(key(tea.KeyEnter, 0))
	if s := h.m.State(); s.Screen != state.ScreenDashboard || cli.starts() != 1 {
		t.Fatalf("screen %q, phase %q, %d starts", s.Screen, s.Startup.Phase, cli.starts())
	}
	h.send(tea.WindowSizeMsg{Width: 120, Height: 40})
	return h, cli
}

// TestStreamingP2 is proposal step 3, item 4: P2's three turns replayed
// through the application, the owner's writes made with the keys. The answer
// is drawn delta by delta (a golden frame at each text delta and each turn end), the
// follow-up goes to the same session, and the interrupt leaves the session
// ready, the turn idle and the draft intact.
func TestStreamingP2(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		recs, err := fixture.Read(filepath.Join("..", "..", "testdata", "probes", "p2-claude-conversation.jsonl"))
		if err != nil {
			t.Fatal(err)
		}
		per := fixture.Replay(claude.ProviderID, recs, claude.NewMapper())
		h, cli := startScript(t, claude.ProviderID)
		sess := cli.sess
		const sessionID = "00ef0150-56f0-4033-8467-bd6e93e296bb"
		const draft = "a draft\n\tkept as typed"

		frame, prompts, deltas := 0, 0, 0
		for i, r := range recs {
			var msg struct {
				Type    string `json:"type"`
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
				Request struct {
					Subtype string `json:"subtype"`
				} `json:"request"`
			}
			_ = json.Unmarshal(r.Msg, &msg)
			if r.Dir == "in" {
				h.settle() // the owner acts on what the screen shows
			}
			s := h.m.State()
			switch {
			case r.Dir == "exit":
				// Stop before the process exit: the session must still be usable.
			case r.Dir == "in" && msg.Type == "user":
				// The owner writes it; the session echoes what was written.
				h.send(tea.PasteMsg{Content: msg.Message.Content})
				h.send(key(tea.KeyEnter, 0))
				sent, _, _ := sess.calls()
				prompts++
				if len(sent) != prompts || sent[prompts-1] != msg.Message.Content {
					t.Fatalf("record %d: writes %q, want %d ending with %q", i, sent, prompts, msg.Message.Content)
				}
				if h.m.edit.Text() != "" {
					t.Errorf("record %d: editor %q after sending", i, h.m.edit.Text())
				}
				if prompts > 1 && (s.Session.ID != sessionID || cli.starts() != 1) {
					t.Errorf("follow-up %d: session %q, %d starts; want %s, 1", prompts, s.Session.ID, cli.starts(), sessionID)
				}
			case r.Dir == "in" && msg.Type == "control_response":
				if len(s.Queue) != 1 {
					t.Fatalf("record %d: %d approvals pending, want 1", i, len(s.Queue))
				}
				id := s.Queue[0]
				h.send(actionMsg{state.Answer{ID: id, Decision: "allow"}}) // the approval keys are step 4
				if _, answers, _ := sess.calls(); len(answers) != 1 || answers[0] != id+" allow" {
					t.Fatalf("record %d: answers %q; session %q, approval %+v, notice %q, notes %q, errors %+v", i, answers, s.Session.Phase, s.Approvals[id].Status, s.Turn.Notice, s.Notes, s.Errors)
				}
			case r.Dir == "in" && msg.Request.Subtype == "interrupt":
				h.send(tea.PasteMsg{Content: draft}) // typed while the turn runs
				if s.Turn.Phase != state.Running {
					t.Fatalf("record %d: turn %q before the interrupt", i, s.Turn.Phase)
				}
				h.send(key('c', tea.ModCtrl))
				if _, _, n := sess.calls(); n != 1 {
					t.Fatalf("record %d: %d interrupts written", i, n)
				}
			}
			if r.Dir == "exit" {
				break
			}
			snap := false
			for _, ev := range per[i] {
				if st, ok := ev.Payload.(event.Started); ok {
					// The probe ran in its own directory; a live run reports the
					// chosen one (a mismatch crashes the session, internal/state/reduce.go).
					st.Cwd = h.m.State().Context.Project
					ev.Payload = st
				}
				sess.events <- ev
				text := ev.Kind == event.MessageDelta && ev.Payload.(event.Delta).Block == "text"
				if text || ev.Kind == event.TurnCompleted {
					snap = true
				}
				if text {
					deltas++
				}
			}
			if snap {
				h.settle()
				frame++
				golden(t, fmt.Sprintf("stream-p2-%02d", frame), fixedProject(plainText(h.m.View().Content)))
			}
		}
		h.settle()
		s := h.m.State()
		if prompts != 3 || deltas == 0 {
			t.Errorf("%d prompts, %d deltas; want 3 prompts and some deltas", prompts, deltas)
		}
		if s.Session.Phase != state.Ready || s.Turn.Phase != state.Idle || s.Turn.Outcome != "interrupted" {
			t.Errorf("after the interrupt: session %q, turn %q (%q); want ready, idle, interrupted", s.Session.Phase, s.Turn.Phase, s.Turn.Outcome)
		}
		if s.Session.ID != sessionID || cli.starts() != 1 {
			t.Errorf("session %q after %d starts", s.Session.ID, cli.starts())
		}
		if h.m.edit.Text() != draft || h.m.ui.Draft != draft {
			t.Errorf("draft %q (shown %q), want %q", h.m.edit.Text(), h.m.ui.Draft, draft)
		}
		t.Logf("%d records, %d deltas, %d frames", len(recs), deltas, frame)
	})
}

// fixedProject replaces the project bar, which shows the test's random
// temporary directory, so the golden frames are stable.
func fixedProject(frame string) string {
	ls := strings.SplitN(frame, "\n", 3)
	if len(ls) == 3 {
		ls[1] = " (project bar: the test's temporary directory)"
	}
	return strings.Join(ls, "\n")
}
