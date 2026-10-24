package app

import (
	"bytes"
	"encoding/json"
	"slices"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/Evengelius/agent-command-center/internal/event"
	"github.com/Evengelius/agent-command-center/internal/provider/claude"
	"github.com/Evengelius/agent-command-center/internal/render"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// Plan 12, item 2: the Manual or Auto switch through the application, on a
// scripted session (no CLI is started); since plan 39 through the modes
// selector F4 opens.

// pushEvents delivers events as the session's channel would, numbered after
// the log's last one.
func pushEvents(h *harness, provider string, evs ...event.Event) {
	s := h.m.State()
	seq, at := uint64(0), time.Date(2026, 10, 18, 9, 0, 0, 0, time.UTC)
	if n := len(s.Log); n > 0 {
		seq, at = s.Log[n-1].Seq, s.Log[n-1].At
	}
	for i := range evs {
		seq++
		evs[i].Seq, evs[i].At, evs[i].Provider = seq, at.Add(time.Duration(i+1)*time.Second), provider
		if evs[i].Origin == "" {
			evs[i].Origin = event.Observed
		}
	}
	h.send(eventsMsg{sess: h.m.sess, evs: evs})
}

func evOf(k event.Kind, p any) event.Event { return event.Event{Kind: k, Payload: p} }

// readyScript is a dashboard on a scripted session whose handshake is done.
func readyScript(t *testing.T, id string, auto bool, started event.Started) (*harness, *scriptCLI) {
	t.Helper()
	h, cli := startScript(t, id)
	cli.sess.auto = auto
	pushEvents(h, id, evOf(event.SessionSpawned, event.Spawned{}), evOf(event.SessionReady, event.Ready{}), evOf(event.SessionStarted, started))
	if s := h.m.State(); s.Session.Phase != state.Ready {
		t.Fatalf("session %q, want ready", s.Session.Phase)
	}
	return h, cli
}

// barLine is the project bar of the frame, the line holding the approvals mode.
func barLine(t *testing.T, h *harness) string {
	t.Helper()
	// The project bar: the mode shows on it only while Auto or changing
	// (plan 28).
	for _, l := range strings.Split(h.frame(), "\n") {
		if strings.Contains(l, "▣") {
			return strings.TrimRight(l, " ")
		}
	}
	t.Fatalf("no project bar in\n%s", h.frame())
	return ""
}

func modesOf(c *scriptCLI) []string {
	c.sess.mu.Lock()
	defer c.sess.mu.Unlock()
	return append([]string(nil), c.sess.modes...)
}

// pickMode opens the modes selector with F4 and chooses mode with the keys:
// Home, ↓ until the cursor is on it, Enter (plan 39, design D8).
func pickMode(t *testing.T, h *harness, mode string) {
	t.Helper()
	h.send(key(tea.KeyF4, 0))
	if !h.m.ui.Modes.Open {
		t.Fatalf("F4 did not open the modes (toast %q):\n%s", h.m.ui.Toast, h.frame())
	}
	h.send(key(tea.KeyHome, 0))
	for i := 0; render.ModesMode(h.m.State(), h.m.ui.Modes, h.m.ui.Modes.Cursor) != mode; i++ {
		if i > 6 {
			t.Fatalf("no option %q in the modes:\n%s", mode, h.frame())
		}
		h.send(key(tea.KeyDown, 0))
	}
	h.send(key(tea.KeyEnter, 0))
}

func TestF4ClaudeSwitch(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, cli := readyScript(t, "claude-code", true, event.Started{SessionID: "s1", Version: "2.1.300", PermissionMode: "default", ApprovalMode: event.ModeManual})
		// Manual with no change asked: the version ends the bar, no mode
		// (plan 28, the owner's image).
		if b := barLine(t, h); !strings.HasSuffix(b, "2.1.300") || strings.Contains(b, "approvals") {
			t.Fatalf("bar %q", b)
		}
		pickMode(t, h, state.ModeAuto)
		if got := modesOf(cli); !slices.Equal(got, []string{"auto"}) {
			t.Fatalf("writes %v, want [auto]", got)
		}
		if h.m.ui.Modes.Open {
			t.Fatal("the modes stayed open after the choice")
		}
		if b := barLine(t, h); !strings.HasSuffix(b, "approvals Manual → Auto …") {
			t.Fatalf("requested: bar %q", b)
		}
		pushEvents(h, "claude-code", evOf(event.ApprovalMode, event.Mode{Mode: event.ModeAuto, Answers: true}))
		if b := barLine(t, h); !strings.HasSuffix(b, "approvals Auto") {
			t.Fatalf("in force: bar %q", b)
		}
		pickMode(t, h, state.ModeManual)
		if got := modesOf(cli); !slices.Equal(got, []string{"auto", "manual"}) {
			t.Fatalf("writes %v, want [auto manual]", got)
		}
		if b := barLine(t, h); !strings.HasSuffix(b, "approvals Auto → Manual …") {
			t.Fatalf("back to Manual requested: bar %q", b)
		}
		// A refusal leaves the mode in force and says so in zone B.
		pushEvents(h, "claude-code", evOf(event.ApprovalMode, event.Mode{Refused: true, Text: "set_permission_mode refused by Claude Code: not allowed"}))
		if b := barLine(t, h); !strings.HasSuffix(b, "approvals Auto") {
			t.Fatalf("refused: bar %q", b)
		}
		if f := h.frame(); !strings.Contains(f, "set_permission_mode refused by Claude Code: not allowed") {
			t.Fatalf("refusal not shown:\n%s", f)
		}
	})
}

func TestF4CodexNextTurn(t *testing.T) {
	for _, w := range []int{100, 120, 156} {
		synctest.Test(t, func(t *testing.T) {
			h, cli := readyScript(t, "codex", true, event.Started{SessionID: "th", Version: "0.145.0", Model: "gpt-5.5", Sandbox: "workspace-write", ApprovalPolicy: "on-request", ApprovalMode: event.ModeManual})
			cli.sess.nextTurn = true // a Codex without thread/settings/update, or during its turn (plan 38)
			h.send(tea.WindowSizeMsg{Width: w, Height: 40})
			b := barLine(t, h)
			if strings.Contains(b, "approvals") || (w >= 120 && !strings.HasSuffix(b, "policy on-request")) {
				t.Fatalf("%d: bar %q", w, b)
			}
			pickMode(t, h, state.ModeAuto)
			if b := barLine(t, h); !strings.HasSuffix(b, "approvals Ask for approval → Approve for me (next turn)") {
				t.Fatalf("%d: requested: bar %q", w, b)
			}
			if got := modesOf(cli); !slices.Equal(got, []string{"auto"}) {
				t.Fatalf("%d: writes %v", w, got)
			}
			pushEvents(h, "codex", evOf(event.ApprovalMode, event.Mode{Mode: event.ModeAuto}))
			if b := barLine(t, h); !strings.HasSuffix(b, "approvals Approve for me") {
				t.Fatalf("%d: in force: bar %q", w, b)
			}
		})
	}
	// Between turns Codex takes the change at once (plan 38's path): no
	// "(next turn)", until Codex says it takes it with the next message.
	synctest.Test(t, func(t *testing.T) {
		h, _ := readyScript(t, "codex", true, event.Started{SessionID: "th", Version: "0.163.0", Model: "gpt-5.5", ApprovalMode: event.ModeManual})
		h.send(tea.WindowSizeMsg{Width: 200, Height: 40})
		pickMode(t, h, state.ModeAuto)
		if b := barLine(t, h); !strings.HasSuffix(b, "approvals Ask for approval → Approve for me …") {
			t.Fatalf("requested at once: bar %q", b)
		}
		pushEvents(h, "codex", evOf(event.ApprovalMode, event.Mode{NextTurn: true, Text: "Codex takes mode changes with the next message: unsupported"}))
		if b := barLine(t, h); !strings.HasSuffix(b, "approvals Ask for approval → Approve for me (next turn)") {
			t.Fatalf("for the next turn: bar %q", b)
		}
	})
}

func TestF4CodexPolicyLabel(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, _ := readyScript(t, "codex", true, event.Started{SessionID: "th", Version: "0.145.0", Model: "gpt-5.5", Sandbox: "workspace-write", ApprovalPolicy: "on-request"})
		h.send(tea.WindowSizeMsg{Width: 156, Height: 40})
		if b := barLine(t, h); !strings.Contains(b, "policy on-request") {
			t.Fatalf("bar %q", b)
		}
	})
}

func TestF4Refused(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		// The capability is off: no selector, a toast, nothing sent.
		h, cli := readyScript(t, "claude-code", false, event.Started{SessionID: "s1", ApprovalMode: event.ModeManual})
		h.send(key(tea.KeyF4, 0))
		if h.m.ui.Modes.Open || len(modesOf(cli)) != 0 {
			t.Fatalf("open %v, written %v without the capability", h.m.ui.Modes.Open, modesOf(cli))
		}
		if h.m.ui.Toast != "no modes with this session" {
			t.Fatalf("toast %q", h.m.ui.Toast)
		}
	})
	synctest.Test(t, func(t *testing.T) {
		// The session is not ready: a notice, nothing sent.
		h, cli := startScript(t, "claude-code")
		cli.sess.auto = true
		pushEvents(h, "claude-code", evOf(event.SessionSpawned, event.Spawned{}))
		pickMode(t, h, state.ModeAuto)
		if len(modesOf(cli)) != 0 {
			t.Fatalf("written before the handshake: %v", modesOf(cli))
		}
		if f := h.frame(); !strings.Contains(f, "mode not changed: the session is") {
			t.Fatalf("no reason shown:\n%s", f)
		}
	})
}

func TestF4WithCardOpen(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, cli := readyScript(t, "claude-code", true, event.Started{SessionID: "s1", ApprovalMode: event.ModeManual})
		in, _ := json.Marshal(map[string]string{"command": "echo hi"})
		pushEvents(h, "claude-code",
			evOf(event.TurnStarted, nil),
			evOf(event.ApprovalRequest, event.Approval{ID: "r1", Tool: "Bash", ToolID: "t1", Input: in}))
		if len(h.m.State().Approvals) != 1 {
			t.Fatalf("%d approvals", len(h.m.State().Approvals))
		}
		for range 2 { // from two zones: F4 opens, Esc closes, the focus stays
			h.send(key(tea.KeyF6, 0))
			focus := h.m.ui.Focus
			h.send(key(tea.KeyF4, 0))
			if !h.m.ui.Modes.Open {
				t.Fatalf("F4 from %s did not open the modes", focus)
			}
			h.send(key(tea.KeyEsc, 0))
			if h.m.ui.Modes.Open || h.m.ui.Focus != focus {
				t.Fatalf("Esc: open %v, focus %q, want closed and %q", h.m.ui.Modes.Open, h.m.ui.Focus, focus)
			}
		}
		pickMode(t, h, state.ModeAuto)
		if got := modesOf(cli); !slices.Equal(got, []string{"auto"}) {
			t.Fatalf("writes %v, want [auto]", got)
		}
		if a := h.m.State().Approvals["r1"]; a == nil || a.Status != state.Pending {
			t.Fatalf("the card changed: %+v", a)
		}
	})
}

func TestApprovalHints(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, _ := readyScript(t, "claude-code", true, event.Started{SessionID: "s1"})
		h.send(tea.WindowSizeMsg{Width: 200, Height: 60})
		if f := h.frame(); !strings.Contains(f, "F4 modes") {
			t.Fatalf("no F4 in the hints:\n%s", f)
		}
		h.send(key(tea.KeyF1, 0))
		if f := h.frame(); !strings.Contains(f, "F4") || !strings.Contains(f, "the modes selector (a click on the mode pill too)") {
			t.Fatalf("no F4 in the F1 list:\n%s", f)
		}
	})
}

func TestReviewerInZoneB(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, _ := readyScript(t, "codex", true, event.Started{SessionID: "th", ApprovalMode: event.ModeAuto})
		h.send(tea.WindowSizeMsg{Width: 156, Height: 40})
		pushEvents(h, "codex",
			evOf(event.TurnStarted, nil),
			evOf(event.ReviewUpdated, event.Review{ReviewID: "rv1", Command: "touch f.txt", Status: "inProgress"}))
		if f := h.frame(); !strings.Contains(f, "auto-review in progress: touch f.txt") {
			t.Fatalf("no progress entry:\n%s", f)
		}
		pushEvents(h, "codex", evOf(event.ReviewUpdated, event.Review{ReviewID: "rv1", Command: "touch f.txt", Status: "approved", Risk: "low", Authorization: "high", Rationale: "a harmless write"}))
		f := h.frame()
		if strings.Contains(f, "in progress") || !strings.Contains(f, "auto-review approved · risk low: a harmless write") {
			t.Fatalf("entry not updated in place:\n%s", f)
		}
		if n := strings.Count(f, "auto-review"); n != 1 {
			t.Fatalf("%d review entries, want 1:\n%s", n, f)
		}
	})
}

func TestToolMarkerInZoneB(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, _ := readyScript(t, "claude-code", true, event.Started{SessionID: "s1", ApprovalMode: event.ModeAuto})
		h.send(tea.WindowSizeMsg{Width: 156, Height: 40})
		in, _ := json.Marshal(map[string]string{"command": "echo hi"})
		pushEvents(h, "claude-code",
			evOf(event.TurnStarted, nil),
			evOf(event.ToolStarted, event.Tool{ToolID: "t1", Name: "Bash", Input: in}),
			evOf(event.ToolStarted, event.Tool{ToolID: "t2", Name: "Read", Input: in}),
			evOf(event.ToolDecision, event.Verdict{ToolID: "t1", Decision: "accept", ReasonType: "classifier"}),
			evOf(event.ToolDecision, event.Verdict{ToolID: "t2", Decision: "accept", ReasonType: "mode"}))
		f := h.frame()
		if n := strings.Count(f, "approved by auto-review"); n != 1 {
			t.Fatalf("%d markers, want 1:\n%s", n, f)
		}
	})
}

// Plan 13, item 1: the classifier's refusal recorded live (P13c) reaches
// zone B as the reviewer's, on the refused call's own row.
func TestClassifierDenialInZoneB(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, _ := readyScript(t, "claude-code", true, event.Started{SessionID: "s1", ApprovalMode: event.ModeAuto})
		h.send(tea.WindowSizeMsg{Width: 156, Height: 40})
		m := claude.NewMapper()
		evs := []event.Event{evOf(event.TurnStarted, nil)}
		for _, r := range probe(t, "p13c-claude-auto-classifier-deny.jsonl") {
			if r.Dir != "out" {
				continue
			}
			for _, e := range m.Out(r.Msg) {
				switch e.Kind {
				case event.SessionSpawned, event.SessionReady, event.SessionStarted:
				default:
					evs = append(evs, e)
				}
			}
			if bytes.Contains(r.Msg, []byte(`"reason_type":"classifier"`)) {
				break // the refused call's result; the answer after it would scroll it away
			}
		}
		pushEvents(h, "claude-code", evs...)
		f := h.frame()
		var row string
		for _, l := range strings.Split(f, "\n") {
			if _, b, ok := strings.Cut(l, "││"); ok && strings.Contains(b, "✗ PowerShell") { // B's row, not zone A's tree
				row = b
			}
		}
		if !strings.Contains(row, "denied by auto-review") {
			t.Fatalf("refused call's row %q, want the reviewer's denial on it:\n%s", row, f)
		}
		if n := strings.Count(f, "by auto-review"); n != 1 {
			t.Fatalf("%d markers, want 1:\n%s", n, f)
		}
	})
}
