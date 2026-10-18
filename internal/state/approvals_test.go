package state_test

import (
	"strings"
	"testing"
	"time"

	"github.com/Evengelius/agent-command-center/internal/event"
	"github.com/Evengelius/agent-command-center/internal/fixture"
	"github.com/Evengelius/agent-command-center/internal/provider/claude"
	"github.com/Evengelius/agent-command-center/internal/provider/codex"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// Plan 12, item 2, criteria 2, 3, 6 and 7: the four probes replayed through
// their mappers and the reducer, and synthetic cases where no probe has the
// event (an escalation in Auto).

func claudeMapper() fixture.Mapper { return claude.NewMapper() }
func codexMapper() fixture.Mapper  { return codex.NewMapper() }

func kinds(evs []event.Event, k event.Kind) (out []event.Event) {
	for _, e := range evs {
		if e.Kind == k {
			out = append(out, e)
		}
	}
	return
}

func TestProbeClaudeAuto(t *testing.T) {
	evs := replayEvents(t, "p12c-claude-auto.jsonl", claude.ProviderID, claudeMapper())
	s := state.New()
	for _, e := range evs {
		s.Apply(e)
	}
	if s.Mode() != state.ModeAuto || s.Session.ApprovalRequested != "" {
		t.Errorf("mode %q requested %q, want auto", s.Mode(), s.Session.ApprovalRequested)
	}
	if d := kinds(evs, event.ToolDecision); len(d) != 1 {
		t.Fatalf("%d tool decisions, want 1", len(d))
	} else if v := d[0].Payload.(event.Verdict); v.ToolID != "toolu_01Dx5mj2rxVr2zh2W7db9Jq4" || v.Decision != "accept" || v.ReasonType != "classifier" {
		t.Errorf("verdict %+v", v)
	}
	if got := s.Tools["toolu_01Dx5mj2rxVr2zh2W7db9Jq4"]; got == nil || got.Reviewed != "approved by auto-review" {
		t.Errorf("classifier tool %+v", got)
	}
	if got := s.Tools["toolu_01VV6kiM6rckE6omxnGS22kM"]; got == nil || got.Reviewed != "" {
		t.Errorf("a decision with reason_type mode marks %+v", got)
	}
	if len(s.Approvals) != 0 {
		t.Errorf("%d approvals in Auto without escalation", len(s.Approvals))
	}
}

func TestProbeClaudeSwitch(t *testing.T) {
	evs := replayEvents(t, "p12c-switch-claude-default-to-auto.jsonl", claude.ProviderID, claudeMapper())
	s := state.New()
	var seen []string // the mode in force after each mode event
	for _, e := range evs {
		s.Apply(e)
		if e.Kind == event.ApprovalMode {
			seen = append(seen, s.Mode())
		}
	}
	if len(seen) < 2 {
		t.Errorf("modes after each report %v", seen)
	}
	var answered bool
	for _, e := range kinds(evs, event.ApprovalMode) {
		if p := e.Payload.(event.Mode); p.Answers && p.Mode == event.ModeAuto {
			answered = true
		}
	}
	if !answered {
		t.Error("no set_permission_mode response in force")
	}
	if len(kinds(evs, event.ApprovalRequest)) != 2 {
		t.Errorf("%d approval requests in default mode, want 2", len(kinds(evs, event.ApprovalRequest)))
	}
	if s.Mode() != state.ModeAuto {
		t.Errorf("final mode %q", s.Mode())
	}
}

func TestProbeCodexReviews(t *testing.T) {
	for _, name := range []string{"p12x-codex-auto-review.jsonl", "p12x-switch-codex-user-to-auto-review.jsonl"} {
		evs := replayEvents(t, name, codex.ProviderID, codexMapper())
		s := state.New()
		for _, e := range evs {
			s.Apply(e)
		}
		if s.Unrecognised != 0 {
			t.Errorf("%s: %d unrecognised: %v", name, s.Unrecognised, s.UnrecognisedLast)
		}
		if s.Mode() != state.ModeAuto {
			t.Errorf("%s: mode %q", name, s.Mode())
		}
		var reviews []state.Entry
		for _, e := range s.Conversation {
			if e.Role == "review" {
				reviews = append(reviews, e)
			}
		}
		ids := map[string]bool{}
		for _, e := range kinds(evs, event.ReviewUpdated) {
			ids[e.Payload.(event.Review).ReviewID] = true
		}
		if len(reviews) == 0 || len(reviews) != len(ids) {
			t.Errorf("%s: %d review entries for %d review ids", name, len(reviews), len(ids))
		}
		for _, r := range reviews {
			if r.Status != "approved" || r.Origin != event.Observed || !strings.HasPrefix(r.Text, "auto-review approved · risk low · authorization high: ") {
				t.Errorf("%s: review entry %+v", name, r)
			}
		}
		if w := kinds(evs, event.ReviewWarning); len(w) == 0 && name == "p12x-codex-auto-review.jsonl" {
			t.Errorf("%s: no guardianWarning mapped", name)
		}
		for _, e := range s.Conversation {
			if strings.Contains(strings.ToLower(e.Text), "guardian") {
				t.Errorf("%s: a warning in zone B: %q", name, e.Text)
			}
		}
	}
}

func TestProbeCodexSwitchModes(t *testing.T) {
	evs := replayEvents(t, "p12x-switch-codex-user-to-auto-review.jsonl", codex.ProviderID, codexMapper())
	s := state.New()
	var modes []string
	cardsInManual := 0
	for _, e := range evs {
		s.Apply(e)
		switch e.Kind {
		case event.ApprovalMode:
			modes = append(modes, s.Mode())
		case event.ApprovalRequest:
			if s.Mode() == state.ModeManual {
				cardsInManual++
			}
		}
	}
	if len(modes) < 2 || modes[0] != state.ModeManual || modes[len(modes)-1] != state.ModeAuto {
		t.Errorf("modes %v", modes)
	}
	if cardsInManual == 0 {
		t.Error("no approval card in Manual before the switch")
	}
}

// start is a ready session.
func start(t *testing.T, started event.Started) *state.State {
	t.Helper()
	s := state.New()
	s.Do(state.Launch{Mock: "x"})
	at := time.Date(2026, 10, 18, 9, 0, 0, 0, time.UTC)
	for i, e := range []event.Event{
		{Kind: event.SessionSpawned, Payload: event.Spawned{}},
		{Kind: event.SessionReady, Payload: event.Ready{}},
		{Kind: event.SessionStarted, Payload: started},
	} {
		e.Seq, e.At, e.Origin, e.Provider = uint64(i+1), at, event.Observed, "claude-code"
		s.Apply(e)
	}
	if s.Session.Phase != state.Ready {
		t.Fatalf("session %q, want ready", s.Session.Phase)
	}
	return s
}

func apply(s *state.State, seq uint64, k event.Kind, p any) {
	s.Apply(event.Event{Seq: seq, At: time.Date(2026, 10, 18, 9, 0, int(seq), 0, time.UTC), Provider: "claude-code", Kind: k, Origin: event.Observed, Payload: p})
}

func TestModeRequested(t *testing.T) {
	s := start(t, event.Started{SessionID: "s", ApprovalMode: event.ModeManual})
	if s.Mode() != state.ModeManual {
		t.Fatalf("mode %q", s.Mode())
	}
	cmds := s.Do(state.ToggleApprovals{Capable: true})
	if len(cmds) != 1 || cmds[0].Kind != state.CmdSetApprovals || cmds[0].Decision != state.ModeAuto || s.Session.ApprovalRequested != state.ModeAuto {
		t.Fatalf("cmds %+v requested %q", cmds, s.Session.ApprovalRequested)
	}
	// A second press asks for the other mode, relative to the request.
	if cmds = s.Do(state.ToggleApprovals{Capable: true}); cmds[0].Decision != state.ModeManual || s.Session.ApprovalRequested != "" {
		t.Fatalf("second press: %+v requested %q", cmds, s.Session.ApprovalRequested)
	}
	s.Do(state.ToggleApprovals{Capable: true})
	apply(s, 10, event.ApprovalMode, event.Mode{Mode: event.ModeAuto, Answers: true})
	if s.Mode() != state.ModeAuto || s.Session.ApprovalRequested != "" {
		t.Fatalf("answered: mode %q requested %q", s.Mode(), s.Session.ApprovalRequested)
	}
	// Refusal: the mode stays, the request clears, zone B says why.
	s.Do(state.ToggleApprovals{Capable: true})
	apply(s, 11, event.ApprovalMode, event.Mode{Refused: true, Text: "set_permission_mode refused by Claude Code: nope"})
	if s.Mode() != state.ModeAuto || s.Session.ApprovalRequested != "" {
		t.Fatalf("refused: mode %q requested %q", s.Mode(), s.Session.ApprovalRequested)
	}
	if last := s.Conversation[len(s.Conversation)-1]; last.Text != "set_permission_mode refused by Claude Code: nope" {
		t.Errorf("zone B shows %q", last.Text)
	}
	// Another word of the CLI's is shown as given.
	apply(s, 12, event.ApprovalMode, event.Mode{Mode: "acceptEdits"})
	if s.Mode() != "acceptEdits" {
		t.Errorf("mode %q", s.Mode())
	}
	// Before the CLI reports, Manual.
	if state.New().Mode() != state.ModeManual {
		t.Error("a new state is not Manual")
	}
}

func TestToggleRefused(t *testing.T) {
	s := start(t, event.Started{SessionID: "s"})
	if cmds := s.Do(state.ToggleApprovals{Capable: false}); cmds != nil || s.Turn.Notice == "" || s.Session.ApprovalRequested != "" {
		t.Errorf("no capability: %+v %q", cmds, s.Turn.Notice)
	}
	n := state.New()
	if cmds := n.Do(state.ToggleApprovals{Capable: true}); cmds != nil || n.Turn.Notice == "" {
		t.Errorf("not ready: %+v %q", cmds, n.Turn.Notice)
	}
	s.Do(state.ToggleApprovals{Capable: true})
	s.Do(state.ApprovalFailed{Message: "broken pipe"})
	if s.Session.ApprovalRequested != "" || !strings.Contains(s.Turn.Notice, "broken pipe") {
		t.Errorf("failed write: requested %q notice %q", s.Session.ApprovalRequested, s.Turn.Notice)
	}
}

func TestVerdicts(t *testing.T) {
	s := start(t, event.Started{SessionID: "s", ApprovalMode: event.ModeAuto})
	for i, id := range []string{"a", "b", "c", "d"} {
		apply(s, uint64(10+i), event.ToolStarted, event.Tool{ToolID: id, Name: "Bash"})
	}
	apply(s, 20, event.ToolDecision, event.Verdict{ToolID: "a", Decision: "accept", ReasonType: "classifier"})
	apply(s, 21, event.ToolDecision, event.Verdict{ToolID: "b", Decision: "reject", ReasonType: "classifier"})
	apply(s, 22, event.ToolDecision, event.Verdict{ToolID: "c", Decision: "accept", ReasonType: "mode"})
	apply(s, 23, event.ToolDecision, event.Verdict{ToolID: "d", Decision: "reject", ReasonType: "user_reject"})
	want := map[string]string{"a": "approved by auto-review", "b": "denied by auto-review", "c": "", "d": ""}
	for id, w := range want {
		if got := s.Tools[id].Reviewed; got != w {
			t.Errorf("tool %s: %q, want %q", id, got, w)
		}
	}
}

func TestReviewEntries(t *testing.T) {
	for _, c := range []struct{ status, want string }{
		{"approved", "auto-review approved · risk low · authorization high: fine"},
		{"denied", "auto-review denied · risk low · authorization high: fine"},
		{"timedOut", "auto-review timed out · risk low · authorization high: fine"},
		{"aborted", "auto-review aborted · risk low · authorization high: fine"},
	} {
		s := start(t, event.Started{SessionID: "s", ApprovalMode: event.ModeAuto})
		apply(s, 10, event.ReviewUpdated, event.Review{ReviewID: "r1", Command: "ls", Status: "inProgress"})
		if e := s.Conversation[len(s.Conversation)-1]; e.Role != "review" || e.Text != "auto-review in progress: ls" {
			t.Fatalf("%s: started entry %+v", c.status, e)
		}
		n := len(s.Conversation)
		apply(s, 11, event.ReviewUpdated, event.Review{ReviewID: "r1", Command: "ls", Status: c.status, Risk: "low", Authorization: "high", Rationale: "fine"})
		if len(s.Conversation) != n {
			t.Fatalf("%s: a second entry", c.status)
		}
		if e := s.Conversation[n-1]; e.Text != c.want || e.Status != c.status {
			t.Errorf("%s: entry %+v", c.status, e)
		}
	}
}

// Criterion 7: an escalation in Auto is a card exactly as in Manual. No
// probe holds one (P12c's boundary probes were blocked by the sandbox or the
// reviewer), so it is synthetic.
func TestEscalationStaysCardInAuto(t *testing.T) {
	for _, mode := range []string{state.ModeManual, state.ModeAuto} {
		s := start(t, event.Started{SessionID: "s", ApprovalMode: mode})
		apply(s, 10, event.TurnStarted, nil)
		apply(s, 11, event.ApprovalRequest, event.Approval{ID: "r1", Tool: "Bash", ToolID: "t1", Summary: "rm -rf x"})
		a := s.Approvals["r1"]
		if a == nil || a.Status != state.Pending {
			t.Fatalf("%s: approval %+v", mode, a)
		}
		if s.Turn.Phase != state.WaitingApproval {
			t.Errorf("%s: turn %q", mode, s.Turn.Phase)
		}
	}
}
