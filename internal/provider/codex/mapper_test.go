package codex

import (
	"path/filepath"
	"slices"
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

// countRaw counts the events of a kind whose line is the given notification.
func countRaw(evs []event.Event, k event.Kind, method string) int {
	n := 0
	for _, e := range evs {
		if e.Kind == k && strings.Contains(e.Raw, `"method":"`+method+`"`) {
			n++
		}
	}
	return n
}

var appServer = []string{"p6-codex-app-server.jsonl", "p6b-codex-app-server.jsonl", "p7-codex-subagents.jsonl"}

// Every line of every app-server recording maps to at least one event, and no
// line is unreadable or unrecognised.
func TestEveryLineMaps(t *testing.T) {
	for _, name := range appServer {
		recs, per := replay(t, name)
		if len(per) != len(recs) {
			t.Errorf("%s: %d results for %d lines", name, len(per), len(recs))
		}
		for i, evs := range per {
			if len(evs) == 0 {
				t.Errorf("%s line %d: no event", name, i+1)
			}
			for _, e := range evs {
				if p, ok := e.Payload.(event.Problem); ok && strings.HasPrefix(p.Message, "unreadable") {
					t.Errorf("%s line %d: %s", name, i+1, p.Message)
				}
				if e.Kind == event.Unrecognised {
					t.Errorf("%s line %d: unrecognised %+v", name, i+1, e.Payload)
				}
			}
		}
	}
}

func TestHandshake(t *testing.T) {
	_, per := replay(t, "p6-codex-app-server.jsonl")
	all := fixture.Flatten(per)
	if n := count(all, event.SessionReady); n != 1 {
		t.Fatalf("session.ready %d, want 1", n)
	}
	for _, e := range all {
		if e.Kind != event.SessionStarted {
			continue
		}
		s := e.Payload.(event.Started)
		if s.SessionID != "01a14116-b6ce-7441-8ac1-108f96c85169" || s.Model != "gpt-6-sol" || s.Sandbox != "readOnly" || s.ApprovalPolicy != "on-request" || s.Version != "0.145.0" || s.Cwd == "" {
			t.Errorf("started %+v", s)
		}
		return
	}
	t.Error("no session.started")
}

// P7: 11 rate-limit pushes and 10 thread status changes, counted from the file.
func TestP7Counts(t *testing.T) {
	_, per := replay(t, "p7-codex-subagents.jsonl")
	all := fixture.Flatten(per)
	if n := countRaw(all, event.Telemetry, "account/rateLimits/updated"); n != 11 {
		t.Errorf("rate-limit telemetry %d, want 11", n)
	}
	n := 0
	for _, e := range all {
		if m, ok := e.Payload.(event.Metrics); ok && len(m.Items) > 0 && m.Items[0].Source == "account/rateLimits/updated" {
			n++
			if m.Items[0].Name != "quota.5h" || m.Items[0].Unit != "%" || !m.Items[0].Documented || m.Items[0].ResetsAt.IsZero() {
				t.Errorf("rate-limit metric %+v", m.Items[0])
			}
		}
	}
	if n != 11 {
		t.Errorf("telemetry with source account/rateLimits/updated %d, want 11", n)
	}
	if n := countRaw(all, event.StatusChanged, "thread/status/changed"); n != 10 {
		t.Errorf("thread status changes %d, want 10", n)
	}
	if n := count(all, event.ApprovalRequest); n != 1 {
		t.Errorf("approvals %d, want 1", n)
	}
}

// P7: the child threads' events carry their thread id as Agent.
func TestP7ChildAgents(t *testing.T) {
	_, per := replay(t, "p7-codex-subagents.jsonl")
	all := fixture.Flatten(per)
	var main string
	for _, e := range all {
		if e.Kind == event.SessionStarted {
			main = e.Payload.(event.Started).SessionID
		}
	}
	if main == "" {
		t.Fatal("no main thread")
	}
	children := map[string]int{}
	for _, e := range all {
		if e.Agent == main {
			t.Errorf("%v of the main thread carries Agent %q", e.Kind, e.Agent)
		}
		if e.Agent != "" {
			children[e.Agent]++
		}
	}
	if len(children) < 2 {
		t.Errorf("child keys %v, want at least 2", children)
	}
	if n := count(all, event.AgentStarted); n < 2 {
		t.Errorf("agent.started %d, want at least 2", n)
	}
	for _, e := range all {
		if e.Kind == event.ApprovalRequest && (e.Agent == "" || e.Agent == main) {
			t.Errorf("the approval of a child thread has Agent %q", e.Agent)
		}
		if e.Kind == event.ApprovalResolved && e.Agent == "" {
			t.Errorf("approval answer %+v lost its agent", e.Payload)
		}
		// The orchestrator spawned both children: they hang at the top of the
		// tree, "" and never the main thread's id (plan 4, step 6, check 7).
		if a, ok := e.Payload.(event.Agent); ok && a.Parent != "" {
			t.Errorf("%v of %q has parent %q, want the orchestrator", e.Kind, a.Key, a.Parent)
		}
	}
}

// A turn/plan/updated maps to a stage change with its steps, observed;
// schema only, not recorded (v2 schema TurnPlanUpdatedNotification).
func TestPlanUpdated(t *testing.T) {
	m := NewMapper()
	evs := m.Out([]byte(`{"method":"turn/plan/updated","params":{"threadId":"t","turnId":"u","explanation":"two steps","plan":[{"step":"list","status":"completed"},{"step":"write","status":"inProgress"}]}}`))
	if len(evs) != 1 || evs[0].Kind != event.StageChanged || evs[0].Origin != event.Observed {
		t.Fatalf("events %+v, want one observed stage change", evs)
	}
	p, ok := evs[0].Payload.(event.Plan)
	want := []event.PlanStep{{Step: "list", Status: "completed"}, {Step: "write", Status: "inProgress"}}
	if !ok || p.Explanation != "two steps" || !slices.Equal(p.Steps, want) {
		t.Errorf("plan %+v, want %v", evs[0].Payload, want)
	}
	if evs := m.Out([]byte(`{"method":"turn/plan/updated","params":{"plan":"x"}}`)); len(evs) != 1 || evs[0].Kind == event.StageChanged {
		t.Errorf("an unreadable plan gives %+v, want one unrecognised event", evs)
	}
}

func TestP6bApproval(t *testing.T) {
	_, per := replay(t, "p6b-codex-app-server.jsonl")
	all := fixture.Flatten(per)
	var reqs, accepted, resolved int
	for _, e := range all {
		switch e.Kind {
		case event.ApprovalRequest:
			reqs++
			a := e.Payload.(event.Approval)
			if a.ID == "" || a.Tool != "fileChange" || a.Problem != "" || a.Summary == "" {
				t.Errorf("approval %+v", a)
			}
		case event.ApprovalResolved:
			resolved++
			if e.Payload.(event.Approval).Decision == "accept" {
				accepted++
			}
		}
	}
	if reqs != 1 || accepted != 1 || resolved != 2 {
		t.Errorf("requests %d, accepted %d, resolved %d, want 1, 1, 2", reqs, accepted, resolved)
	}
	var outcomes []string
	deltas := 0
	for _, e := range all {
		switch p := e.Payload.(type) {
		case event.TurnEnd:
			outcomes = append(outcomes, p.Outcome)
		case event.Delta:
			deltas++
			if e.Origin != event.AI || p.Block != "text" {
				t.Errorf("delta origin %q block %q", e.Origin, p.Block)
			}
		}
	}
	if len(outcomes) != 2 || outcomes[0] != "completed" || outcomes[1] != "interrupted" {
		t.Errorf("turn outcomes %v, want [completed interrupted]", outcomes)
	}
	if deltas != 20 {
		t.Errorf("deltas %d, want 20", deltas)
	}
	if n := count(all, event.UserMessage); n != 2 {
		t.Errorf("user messages %d, want 2", n)
	}
	if n := count(all, event.DiffUpdated); n != 4 {
		t.Errorf("diffs %d, want 4", n)
	}
	if n := count(all, event.ToolStarted); n != 2 {
		t.Errorf("tools started %d, want 2", n)
	}
}

func TestP6ModelError(t *testing.T) {
	_, per := replay(t, "p6-codex-app-server.jsonl")
	all := fixture.Flatten(per)
	if n := count(all, event.Error); n != 2 {
		t.Errorf("errors %d, want 2", n)
	}
	for _, e := range all {
		if e.Kind == event.Error && !strings.Contains(e.Payload.(event.Problem).Message, "not supported") {
			t.Errorf("error message %q", e.Payload.(event.Problem).Message)
		}
	}
	var outcomes []string
	for _, e := range all {
		if p, ok := e.Payload.(event.TurnEnd); ok {
			outcomes = append(outcomes, p.Outcome)
			if p.Error == "" {
				t.Error("failed turn without an error message")
			}
		}
	}
	if len(outcomes) != 2 || outcomes[0] != "failed" || outcomes[1] != "failed" {
		t.Errorf("turn outcomes %v, want [failed failed]", outcomes)
	}
}

// P5 and P5b are `codex exec --json` recordings, a different protocol: the
// app-server mapper does not read exec output, so each line is unrecognised.
func TestExecOutputIsUnrecognised(t *testing.T) {
	for _, name := range []string{"p5-codex-exec.jsonl", "p5b-codex-exec.jsonl"} {
		_, per := replay(t, name)
		for i, evs := range per {
			if len(evs) != 1 || evs[0].Kind != event.Unrecognised {
				t.Errorf("%s line %d: %d events, first %v; want one unrecognised", name, i+1, len(evs), evs)
			}
		}
	}
}

func TestMalformedAndUnknown(t *testing.T) {
	m := NewMapper()
	evs := m.Out([]byte(`{"id":9,"method":"item/commandExecution/requestApproval","params":"oops"}`))
	if len(evs) != 1 || evs[0].Kind != event.ApprovalRequest || evs[0].Payload.(event.Approval).Problem == "" {
		t.Errorf("malformed approval: %+v", evs)
	}
	evs = m.Out([]byte(`{"id":10,"method":"item/fileChange/requestApproval","params":{"threadId":"t"}}`))
	if len(evs) != 1 || evs[0].Payload.(event.Approval).Problem == "" {
		t.Errorf("approval without itemId: %+v", evs)
	}
	evs = m.Out([]byte(`{"id":11,"method":"item/tool/call","params":{}}`))
	if len(evs) != 1 || evs[0].Kind != event.Unrecognised || evs[0].Payload.(event.Unknown).Reason != "server request the dashboard does not handle" {
		t.Errorf("unknown server request: %+v", evs)
	}
	evs = m.Out([]byte(`not json`))
	if len(evs) != 1 || evs[0].Kind != event.Error || !strings.HasPrefix(evs[0].Payload.(event.Problem).Message, "unreadable line from Codex: ") {
		t.Errorf("not json: %+v", evs)
	}
	evs = m.In([]byte(`not json`))
	if len(evs) != 1 || evs[0].Kind != event.Error {
		t.Errorf("not json in: %+v", evs)
	}
}

func TestErrorResponseAndRetry(t *testing.T) {
	m := NewMapper()
	m.In([]byte(`{"id":3,"method":"thread/start","params":{}}`))
	evs := m.Out([]byte(`{"id":3,"error":{"code":-32600,"message":"bad cwd"}}`))
	if len(evs) != 1 || evs[0].Kind != event.Error || !strings.Contains(evs[0].Payload.(event.Problem).Message, "bad cwd") {
		t.Errorf("error response: %+v", evs)
	}
	evs = m.Out([]byte(`{"method":"error","params":{"error":{"message":"slow down"},"willRetry":true,"threadId":"t","turnId":"u"}}`))
	if len(evs) != 1 || evs[0].Kind != event.Retry || !evs[0].Payload.(event.Problem).WillRetry {
		t.Errorf("retry: %+v", evs)
	}
}

func TestRateLimitWindows(t *testing.T) {
	m := NewMapper()
	evs := m.Out([]byte(`{"method":"account/rateLimits/updated","params":{"rateLimits":{"primary":{"usedPercent":42,"windowDurationMins":60,"resetsAt":1792111369},"secondary":{"usedPercent":7,"windowDurationMins":10080,"resetsAt":1792657845},"planType":"plus"}}}`))
	if len(evs) != 1 {
		t.Fatalf("events %d", len(evs))
	}
	items := evs[0].Payload.(event.Metrics).Items
	if len(items) != 3 || items[0].Name != "quota.60m" || items[0].Value != 42 || items[1].Name != "quota.7d" || items[2].Name != "plan" || items[2].Text != "plus" {
		t.Errorf("metrics %+v", items)
	}
	if items[1].ResetsAt.Unix() != 1792657845 {
		t.Errorf("resets at %v", items[1].ResetsAt)
	}
}

func TestInterruptMarksOutcome(t *testing.T) {
	m := NewMapper()
	m.In([]byte(`{"id":6,"method":"turn/interrupt","params":{"threadId":"t","turnId":"u"}}`))
	evs := m.Out([]byte(`{"method":"turn/completed","params":{"threadId":"t","turn":{"id":"u","status":"cancelled","error":null}}}`))
	evs = append(evs, m.Out([]byte(`{"method":"turn/completed","params":{"threadId":"t","turn":{"id":"v","status":"failed","error":{"message":"boom"}}}}`))...)
	if evs[0].Payload.(event.TurnEnd).Outcome != "interrupted" || evs[1].Payload.(event.TurnEnd).Outcome != "failed" {
		t.Errorf("outcome %+v", evs[0].Payload)
	}
}

func TestDeterministic(t *testing.T) {
	_, a := replay(t, "p7-codex-subagents.jsonl")
	_, b := replay(t, "p7-codex-subagents.jsonl")
	x, y := fixture.Flatten(a), fixture.Flatten(b)
	if len(x) != len(y) {
		t.Fatalf("%d events then %d", len(x), len(y))
	}
	for i := range x {
		if x[i].Kind != y[i].Kind || x[i].Agent != y[i].Agent || x[i].Raw != y[i].Raw {
			t.Fatalf("event %d differs", i)
		}
	}
}
