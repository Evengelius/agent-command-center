package app

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/Evengelius/agent-command-center/internal/event"
	"github.com/Evengelius/agent-command-center/internal/provider/claude"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// Plan 12, item 2: the Manual or Auto switch through the application, on a
// scripted session (no CLI is started).

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
	for _, l := range strings.Split(h.frame(), "\n") {
		if strings.Contains(l, "approvals ") && strings.Contains(l, "▣") {
			return strings.TrimRight(l, " ")
		}
	}
	t.Fatalf("no project bar with the approvals mode in\n%s", h.frame())
	return ""
}

func modesOf(c *scriptCLI) []bool {
	c.sess.mu.Lock()
	defer c.sess.mu.Unlock()
	return append([]bool(nil), c.sess.modes...)
}

func TestF4ClaudeSwitch(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, cli := readyScript(t, "claude-code", true, event.Started{SessionID: "s1", Version: "2.1.300", PermissionMode: "default", ApprovalMode: event.ModeManual})
		if b := barLine(t, h); !strings.HasSuffix(b, "approvals Manual") {
			t.Fatalf("bar %q", b)
		}
		h.send(key(tea.KeyF4, 0))
		if got := modesOf(cli); len(got) != 1 || !got[0] {
			t.Fatalf("writes %v, want [true]", got)
		}
		if b := barLine(t, h); !strings.HasSuffix(b, "approvals Manual → Auto …") {
			t.Fatalf("requested: bar %q", b)
		}
		pushEvents(h, "claude-code", evOf(event.ApprovalMode, event.Mode{Mode: event.ModeAuto, Answers: true}))
		if b := barLine(t, h); !strings.HasSuffix(b, "approvals Auto") {
			t.Fatalf("in force: bar %q", b)
		}
		h.send(key(tea.KeyF4, 0))
		if got := modesOf(cli); len(got) != 2 || got[1] {
			t.Fatalf("writes %v, want [true false]", got)
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
			h.send(tea.WindowSizeMsg{Width: w, Height: 40})
			b := barLine(t, h)
			if !strings.HasSuffix(b, "approvals Manual") || (w >= 120 && !strings.Contains(b, "policy on-request")) {
				t.Fatalf("%d: bar %q", w, b)
			}
			h.send(key(tea.KeyF4, 0))
			if b := barLine(t, h); !strings.HasSuffix(b, "approvals Manual → Auto (next turn)") {
				t.Fatalf("%d: requested: bar %q", w, b)
			}
			if len(modesOf(cli)) != 1 {
				t.Fatalf("%d: %d writes", w, len(modesOf(cli)))
			}
			pushEvents(h, "codex", evOf(event.ApprovalMode, event.Mode{Mode: event.ModeAuto}))
			if b := barLine(t, h); !strings.HasSuffix(b, "approvals Auto") {
				t.Fatalf("%d: in force: bar %q", w, b)
			}
		})
	}
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
		// The capability is off: a notice, nothing sent.
		h, cli := readyScript(t, "claude-code", false, event.Started{SessionID: "s1", ApprovalMode: event.ModeManual})
		h.send(key(tea.KeyF4, 0))
		if len(modesOf(cli)) != 0 {
			t.Fatalf("written without the capability: %v", modesOf(cli))
		}
		if f := h.frame(); !strings.Contains(f, "no Manual/Auto switch here") {
			t.Fatalf("no reason shown:\n%s", f)
		}
	})
	synctest.Test(t, func(t *testing.T) {
		// The session is not ready: a notice, nothing sent.
		h, cli := startScript(t, "claude-code")
		cli.sess.auto = true
		pushEvents(h, "claude-code", evOf(event.SessionSpawned, event.Spawned{}))
		h.send(key(tea.KeyF4, 0))
		if len(modesOf(cli)) != 0 {
			t.Fatalf("written before the handshake: %v", modesOf(cli))
		}
		if f := h.frame(); !strings.Contains(f, "approvals not changed: the session is") {
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
		for range 2 { // from two zones
			h.send(key(tea.KeyF6, 0))
			h.send(key(tea.KeyF4, 0))
			h.send(key(tea.KeyF4, 0))
		}
		if got := modesOf(cli); len(got) != 4 {
			t.Fatalf("writes %v, want four", got)
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
		if f := h.frame(); !strings.Contains(f, "F4 Manual/Auto") {
			t.Fatalf("no F4 in the hints:\n%s", f)
		}
		h.send(key(tea.KeyF1, 0))
		if f := h.frame(); !strings.Contains(f, "F4") || !strings.Contains(f, "approvals Manual / Auto") {
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
		if strings.Contains(f, "in progress") || !strings.Contains(f, "auto-review approved · risk low · authorization high: a harmless write") {
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
