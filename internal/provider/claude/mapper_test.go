package claude

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/Evengelius/agent-command-center/internal/event"
	"github.com/Evengelius/agent-command-center/internal/fixture"
)

func replay(t *testing.T, name string) ([]fixture.Record, [][]event.Event) {
	t.Helper()
	recs, err := fixture.Read(filepath.Join("../../../testdata/probes", name))
	if err != nil {
		t.Fatal(err)
	}
	return recs, fixture.Replay(ProviderID, recs, NewMapper())
}

func count(evs []event.Event, k event.Kind) int {
	n := 0
	for _, e := range evs {
		if e.Kind == k {
			n++
		}
	}
	return n
}

// Every line of every Claude recording maps to at least one event, except the
// quiet ones (a hook's start, the turn's summary, a hook's success: plan 22,
// amendment 20), and no line is unreadable (proposal step 0, item 6: 0 lines
// dropped); none of P22o's is unrecognised (it had 3).
func TestEveryLineMaps(t *testing.T) {
	for _, name := range []string{"p1-claude-stream.jsonl", "p2-claude-conversation.jsonl", "p3-claude-subagent.jsonl", "p3b-claude-subagent-approval.jsonl", "p4-claude-resume.jsonl", "p12c-claude-auto.jsonl", "p12c-switch-claude-default-to-auto.jsonl", "p13c-claude-auto-classifier-deny.jsonl", "p14a-claude-question-answer.jsonl", "p303a-claude-conversation.jsonl", "p303b-claude-subagent-approval.jsonl", "p303c-claude-default-to-auto.jsonl", "p303d-claude-question-answer.jsonl", "p303e-claude-interrupt-tool.jsonl", p22o} {
		recs, per := replay(t, name)
		for i, evs := range per {
			if len(evs) == 0 && !quietLine(recs[i].Msg) {
				t.Errorf("%s line %d: no event", name, i+1)
			}
			if len(evs) != 0 && quietLine(recs[i].Msg) {
				t.Errorf("%s line %d: %d events from a quiet line", name, i+1, len(evs))
			}
			for _, e := range evs {
				if e.Kind == event.Error {
					if p, ok := e.Payload.(event.Problem); ok && len(p.Message) > 10 && p.Message[:10] == "unreadable" {
						t.Errorf("%s line %d: %s", name, i+1, p.Message)
					}
				}
				if name == p22o && e.Kind == event.Unrecognised {
					t.Errorf("%s line %d: unrecognised %+v", name, i+1, e.Payload)
				}
			}
		}
		if len(per) != len(recs) {
			t.Errorf("%s: %d results for %d lines", name, len(per), len(recs))
		}
	}
}

func TestP2Conversation(t *testing.T) {
	_, per := replay(t, "p2-claude-conversation.jsonl")
	all := fixture.Flatten(per)
	deltas, text, thinking, input, sig := 0, 0, 0, 0, 0
	for _, e := range all {
		if e.Kind != event.MessageDelta {
			continue
		}
		deltas++
		if e.Origin != event.AI {
			t.Errorf("delta origin %q, want ai", e.Origin)
		}
		switch e.Payload.(event.Delta).Block {
		case "text":
			text++
		case "thinking":
			thinking++
		case "tool_input":
			input++
		case "signature":
			sig++
		}
	}
	if deltas != 25 || text != 3 || thinking != 1 || input != 20 || sig != 1 {
		t.Errorf("deltas %d (text %d, thinking %d, tool_input %d, signature %d), want 25 (3, 1, 20, 1)", deltas, text, thinking, input, sig)
	}
	if n := count(all, event.SessionReady); n != 1 {
		t.Errorf("session.ready %d, want 1", n)
	}
	if n := count(all, event.SessionSpawned); n != 1 {
		t.Errorf("session.spawned %d, want 1", n)
	}
	var outcomes []string
	var ctx, window float64
	for _, e := range all {
		switch p := e.Payload.(type) {
		case event.TurnEnd:
			outcomes = append(outcomes, p.Outcome)
		case event.Metrics:
			for _, m := range p.Items {
				if m.Source == "get_context_usage" && m.Name == "context.used" {
					ctx = m.Value
				}
				if m.Source == "get_context_usage" && m.Name == "context.window" {
					window = m.Value
				}
			}
		}
	}
	if len(outcomes) != 3 || outcomes[0] != "completed" || outcomes[1] != "completed" || outcomes[2] != "interrupted" {
		t.Errorf("turn outcomes %v, want [completed completed interrupted]", outcomes)
	}
	if ctx != 21275 || window != 1000000 {
		t.Errorf("context %v of %v, want 21275 of 1000000", ctx, window)
	}
	if n := count(all, event.ApprovalRequest); n != 1 {
		t.Errorf("approvals %d, want 1", n)
	}
	if n := count(all, event.ApprovalResolved); n != 1 {
		t.Errorf("approval answers %d, want 1", n)
	}
	if n := count(all, event.UserMessage); n != 3 {
		t.Errorf("user messages %d, want 3", n)
	}
	last := all[len(all)-1]
	if last.Kind != event.SessionEnded || !last.Payload.(event.Ended).HasCode || last.Payload.(event.Ended).Code != 1 {
		t.Errorf("last event %v %+v, want session.ended code 1", last.Kind, last.Payload)
	}
}

func TestP3bSubagentApproval(t *testing.T) {
	_, per := replay(t, "p3b-claude-subagent-approval.jsonl")
	all := fixture.Flatten(per)
	var fromAgent int
	for _, e := range all {
		if e.Kind == event.ApprovalRequest {
			a := e.Payload.(event.Approval)
			if a.ID == "" || a.Tool == "" || a.Problem != "" {
				t.Errorf("approval %+v incomplete", a)
			}
			if e.Agent != "" {
				fromAgent++
			}
		}
	}
	if n := count(all, event.ApprovalRequest); n != 2 || fromAgent != 2 {
		t.Errorf("approvals %d (%d from the subagent), want 2 (2)", n, fromAgent)
	}
	unknown := map[string]int{}
	var commands []event.Status
	for _, e := range all {
		if e.Kind == event.Unrecognised {
			unknown[e.Payload.(event.Unknown).Type]++
		}
		if e.Kind == event.StatusChanged && strings.Contains(e.Raw, `"subtype":"commands_changed"`) {
			commands = append(commands, e.Payload.(event.Status))
		}
	}
	if len(unknown) != 0 {
		t.Errorf("unrecognised %v, want none", unknown)
	}
	// commands_changed is known and changes no status.
	if len(commands) != 1 || commands[0].Status != "" || !strings.HasSuffix(commands[0].Detail, " commands available") || commands[0].Detail == "0 commands available" {
		t.Errorf("commands_changed mapped to %+v; want one status.changed with no status and its command count", commands)
	}
	if n := count(all, event.AgentSet); n != 2 {
		t.Errorf("agent.set %d, want 2", n)
	}
	if n := count(all, event.AgentStarted); n < 1 {
		t.Errorf("agent.started %d, want at least 1", n)
	}
}

func TestP1Cost(t *testing.T) {
	_, per := replay(t, "p1-claude-stream.jsonl")
	var cost float64
	found := false
	for _, e := range fixture.Flatten(per) {
		if p, ok := e.Payload.(event.Metrics); ok {
			for _, m := range p.Items {
				if m.Name == "cost.usd" {
					cost, found = m.Value, true
				}
			}
		}
	}
	if !found || cost != 0.00249374 {
		t.Errorf("cost %v (found %v), want 0.00249374", cost, found)
	}
}

func TestMalformedApprovalHasProblem(t *testing.T) {
	m := NewMapper()
	evs := m.Out([]byte(`{"type":"control_request","request_id":"r1","request":{"subtype":"can_use_tool"}}`))
	if len(evs) != 1 || evs[0].Kind != event.ApprovalRequest || evs[0].Payload.(event.Approval).Problem == "" {
		t.Errorf("got %+v, want an approval request with a problem", evs)
	}
	evs = m.Out([]byte(`not json`))
	if len(evs) != 1 || evs[0].Kind != event.Error {
		t.Errorf("got %+v, want one error", evs)
	}
}

// TestCompactBoundary: a compaction's system/compact_boundary (the Agent
// SDK's SDKCompactBoundaryMessage shape) is one observed status "compacted"
// (reliability.md, Not verified: compaction is Claude Code's, not depended on).
func TestCompactBoundary(t *testing.T) {
	m := NewMapper()
	evs := m.Out([]byte(`{"type":"system","subtype":"compact_boundary","session_id":"s1","compact_metadata":{"trigger":"auto","pre_tokens":150000}}`))
	if len(evs) != 1 || evs[0].Kind != event.StatusChanged || evs[0].Origin != event.Observed {
		t.Fatalf("got %+v, want one observed status change", evs)
	}
	if st, ok := evs[0].Payload.(event.Status); !ok || st.Status != "compacted" {
		t.Errorf("payload %+v, want status \"compacted\"", evs[0].Payload)
	}
}

// A tool_result with is_error completes its call as an error, its content as
// the output (plan 5, item 6: B marks the call ✗ with that output's first
// line). P3's only one, a subagent's.
func TestToolError(t *testing.T) {
	_, per := replay(t, "p3-claude-subagent.jsonl")
	var got []string
	for _, e := range fixture.Flatten(per) {
		if p, ok := e.Payload.(event.Tool); ok && e.Kind == event.ToolCompleted && p.IsError {
			got = append(got, p.ToolID+" "+p.Output)
		}
	}
	if want := "toolu_01GCkWKjmPUFvje2MRWNXr8N Tool permission request failed: AbortError: Stream closed"; len(got) != 1 || got[0] != want {
		t.Errorf("failed calls %q, want [%q]", got, want)
	}
}

// TestSubagentModel is plan 5, item 7, criteria 1 and 4: a subagent's own
// model, from its messages, sent once per change; never the main thread's.
func TestSubagentModel(t *testing.T) {
	for _, c := range []struct{ file, key string }{
		{"p3-claude-subagent.jsonl", "ad30fdcc2c6928f90"},
		{"p3b-claude-subagent-approval.jsonl", "a9dcbbc8d63adf34c"},
	} {
		_, per := replay(t, c.file)
		var got []string
		for _, e := range fixture.Flatten(per) {
			if p, ok := e.Payload.(event.Agent); ok && p.Model != "" {
				got = append(got, string(e.Kind)+" "+e.Agent+" "+p.Key+" "+p.Model)
			}
		}
		if want := "agent.progress " + c.key + " " + c.key + " claude-haiku-5-5"; len(got) != 1 || got[0] != want {
			t.Errorf("%s: model events %q, want [%q]", c.file, got, want)
		}
	}

	m := NewMapper()
	msg := func(parent, id, model, content string) []event.Event {
		p := "null"
		if parent != "" {
			p = `"` + parent + `"`
		}
		return m.Out([]byte(`{"type":"assistant","session_id":"s1","parent_tool_use_id":` + p + `,"agent_id":"ag1","message":{"id":"` + id + `","model":"` + model + `","content":[` + content + `]}}`))
	}
	tool := func(id string) string { return `{"type":"tool_use","id":"t` + id + `","name":"Bash","input":{}}` }
	models := func(evs []event.Event) (out []string, completed bool) {
		for _, e := range evs {
			if p, ok := e.Payload.(event.Agent); ok {
				out = append(out, e.Agent+" "+p.Model)
			}
			completed = completed || e.Kind == event.MessageCompleted
		}
		return out, completed
	}
	for _, c := range []struct {
		name, parent, id, model, content string
		want                             []string
		completed                        bool
	}{
		{"first", "toolu_1", "m1", "claude-haiku-5-5", tool("1"), []string{"ag1 claude-haiku-5-5"}, false},
		{"same model", "toolu_1", "m2", "claude-haiku-5-5", tool("2"), nil, false},
		{"changed", "toolu_1", "m3", "claude-sonnet-5-5", tool("3"), []string{"ag1 claude-sonnet-5-5"}, false},
		{"made by the CLI", "toolu_1", "m4", "<synthetic>", tool("4"), nil, false},
		{"main thread", "", "m5", "claude-opus-5-5", tool("5"), nil, false},
		{"no blocks", "toolu_1", "m6", "claude-opus-5-5", "", []string{"ag1 claude-opus-5-5"}, true},
	} {
		got, completed := models(msg(c.parent, c.id, c.model, c.content))
		if strings.Join(got, ",") != strings.Join(c.want, ",") || completed != c.completed {
			t.Errorf("%s: model events %q, completed %v; want %q, %v", c.name, got, completed, c.want, c.completed)
		}
	}
}

// Plan 12, item 2, criterion 3: set_permission_mode answers and refusals, and
// the modes system/status and system/init report.
func TestPermissionModeReports(t *testing.T) {
	in := `{"type":"control_request","request_id":"req_2","request":{"subtype":"set_permission_mode","mode":"auto"}}`
	m := NewMapper()
	m.In([]byte(in))
	evs := m.Out([]byte(`{"type":"control_response","response":{"subtype":"error","request_id":"req_2","error":"nope"}}`))
	if len(evs) != 1 || evs[0].Kind != event.ApprovalMode {
		t.Fatalf("refusal: %+v", evs)
	}
	if p := evs[0].Payload.(event.Mode); !p.Refused || p.Text != "set_permission_mode refused by Claude Code: nope" || p.Mode != "" {
		t.Errorf("refusal payload %+v", p)
	}
	m.In([]byte(in))
	evs = m.Out([]byte(`{"type":"control_response","response":{"subtype":"success","request_id":"req_2","response":{"mode":"auto"}}}`))
	if len(evs) != 1 || evs[0].Payload != (event.Mode{Mode: event.ModeAuto, Answers: true}) {
		t.Errorf("answer: %+v", evs)
	}
	for word, want := range map[string]string{"default": event.ModeManual, "auto": event.ModeAuto, "acceptEdits": "acceptEdits"} {
		if got := approvalMode(word); got != want {
			t.Errorf("approvalMode(%q) = %q, want %q", word, got, want)
		}
	}
}

// A decision whose reason_type is not classifier adds no event.
func TestVerdictsOnlyForTheClassifier(t *testing.T) {
	for _, c := range []struct {
		reason string
		want   int
	}{{"classifier", 1}, {"mode", 0}, {"user_reject", 0}} {
		m := NewMapper()
		evs := m.Out([]byte(`{"type":"user","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"toolu_1","content":"ok"}]},"tool_result_meta":[{"id":"toolu_1","permission_decision":{"decision":"accept","reason_type":"` + c.reason + `"}}]}`))
		if n := count(evs, event.ToolDecision); n != c.want {
			t.Errorf("reason_type %q: %d decisions, want %d", c.reason, n, c.want)
		}
	}
}

// Plan 14, item 6: a tool result without tool_result_meta's
// permission_decision is one unrecognised event naming the field, unless its
// entry marks a tool use an interrupt rejected (non_execution_kind
// "user-rejected", P303e); the classifier's verdicts still map.
func TestToolResultWithoutPermissionDecision(t *testing.T) {
	const result = `{"type":"user","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"toolu_1","content":"ok"}]}`
	for name, c := range map[string]struct {
		line               string
		missing, decisions int
	}{
		"no meta":                    {result + `}`, 1, 0},
		"meta, no decision":          {result + `,"tool_result_meta":[{"id":"toolu_1"}]}`, 1, 0},
		"another result's":           {result + `,"tool_result_meta":[{"id":"toolu_2","permission_decision":{"decision":"accept","reason_type":"classifier"}}]}`, 1, 1},
		"another shape":              {result + `,"tool_result_meta":{"id":"toolu_1"}}`, 1, 0},
		"decision":                   {result + `,"tool_result_meta":[{"id":"toolu_1","permission_decision":{"decision":"accept","reason_type":"mode"}}]}`, 0, 0},
		"classifier verdict":         {result + `,"tool_result_meta":[{"id":"toolu_1","permission_decision":{"decision":"reject","reason_type":"classifier"}}]}`, 0, 1},
		"interrupted":                {result + `,"tool_result_meta":[{"id":"toolu_1","non_execution_kind":"user-rejected"}]}`, 0, 0},
		"another result interrupted": {result + `,"tool_result_meta":[{"id":"toolu_2","non_execution_kind":"user-rejected"}]}`, 1, 0},
		"another kind, no decision":  {result + `,"tool_result_meta":[{"id":"toolu_1","non_execution_kind":"permission-rule"}]}`, 1, 0},
	} {
		evs := NewMapper().Out([]byte(c.line))
		n := 0
		for _, e := range evs {
			if u, ok := e.Payload.(event.Unknown); ok && e.Kind == event.Unrecognised && u.Type == "user/tool_result_meta" {
				n++
				if !strings.Contains(u.Reason, "permission_decision") {
					t.Errorf("%s: reason %q does not name the field", name, u.Reason)
				}
			}
		}
		if n != c.missing || count(evs, event.ToolDecision) != c.decisions {
			t.Errorf("%s: %d unrecognised, %d decisions; want %d, %d", name, n, count(evs, event.ToolDecision), c.missing, c.decisions)
		}
	}
}

// Every Claude recording carries permission_decision on each tool result, or
// marks it rejected by an interrupt (P303e): none replays with the shape
// check's unrecognised event.
func TestRecordingsHavePermissionDecision(t *testing.T) {
	names, err := filepath.Glob("../../../testdata/probes/p*-claude-*.jsonl")
	if err != nil || len(names) == 0 {
		t.Fatalf("no recordings: %v", err)
	}
	results := 0
	for _, path := range names {
		recs, per := replay(t, filepath.Base(path))
		for _, r := range recs {
			results += strings.Count(string(r.Msg), `"type":"tool_result"`)
		}
		for i, evs := range per {
			for _, e := range evs {
				if u, ok := e.Payload.(event.Unknown); ok && u.Type == "user/tool_result_meta" {
					t.Errorf("%s line %d: %s", filepath.Base(path), i+1, u.Reason)
				}
			}
		}
	}
	if results == 0 {
		t.Error("no tool result in the recordings")
	}
	t.Logf("%d recordings, %d tool results", len(names), results)
}
