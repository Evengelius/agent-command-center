package codex

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Evengelius/agent-command-center/internal/event"
	"github.com/Evengelius/agent-command-center/internal/fixture"
	"github.com/Evengelius/agent-command-center/internal/state"
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

var appServer = []string{"p6-codex-app-server.jsonl", "p6b-codex-app-server.jsonl", "p7-codex-subagents.jsonl", "p7b-codex-subagents.jsonl", "p7c-codex-spawn-model.jsonl", "p9a-codex-spawn-request.jsonl", "p12x-codex-auto-review.jsonl", "p12x-switch-codex-user-to-auto-review.jsonl", "p12x-sandbox-codex-turn-workspace-write.jsonl"}

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

// P7b (Codex 0.163.0): 28 command output chunks become tool progress, and the
// children's "interacted" toward "/root" never names the orchestrator as an
// agent; the two "completed" activities end the two children.
func TestP7bCounts(t *testing.T) {
	_, per := replay(t, "p7b-codex-subagents.jsonl")
	all := fixture.Flatten(per)
	if n := countRaw(all, event.ToolProgress, "item/commandExecution/outputDelta"); n != 28 {
		t.Errorf("tool progress %d, want 28", n)
	}
	var main string
	for _, e := range all {
		if e.Kind == event.SessionStarted {
			main = e.Payload.(event.Started).SessionID
		}
	}
	for _, e := range all {
		if a, ok := e.Payload.(event.Agent); ok && (a.Key == main || a.Label == "/root") {
			t.Errorf("%v names the orchestrator as agent %q (%q)", e.Kind, a.Key, a.Label)
		}
	}
	n := 0
	for _, e := range all {
		if e.Kind == event.AgentCompleted && strings.Contains(e.Raw, `"type":"subAgentActivity"`) && strings.Contains(e.Raw, `"kind":"completed"`) {
			n++
		}
	}
	if n != 4 {
		t.Errorf("completions from subAgentActivity %d, want 4 (started and completed items of 2 children)", n)
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
	// No 300-minute window: the 5-hour quota is N/A for that reason, first.
	if len(items) != 4 || items[0].Name != "quota.5h" || items[0].Reason != "plan has no 5-hour limit" ||
		items[1].Name != "quota.60m" || items[1].Value != 42 || items[2].Name != "quota.7d" || items[3].Name != "plan" || items[3].Text != "plus" {
		t.Fatalf("metrics %+v", items)
	}
	if items[2].ResetsAt.Unix() != 1792657845 {
		t.Errorf("resets at %v", items[2].ResetsAt)
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

// A command that failed completes its call as an error, its aggregated output
// as the output (plan 5, item 6: B marks the call ✗ with that output's first
// line). P7 has one, p7b two, all children's.
func TestFailedCommand(t *testing.T) {
	for _, c := range []struct {
		file string
		want []string
	}{
		{"p7-codex-subagents.jsonl", []string{"exec-8519b056-1ecc-4773-8aa3-ad18ba31e7c8 Cannot invoke method."}},
		{"p7b-codex-subagents.jsonl", []string{"exec-f0113449-fe16-4fee-9b99-226b2ec86369 \r\nFullName", "exec-25fb5cc3-231b-4d2a-b6c3-e0096bba1852 Set-Content : Access to the path"}},
	} {
		_, per := replay(t, c.file)
		var got []string
		for _, e := range fixture.Flatten(per) {
			if p, ok := e.Payload.(event.Tool); ok && e.Kind == event.ToolCompleted && p.IsError {
				got = append(got, p.ToolID+" "+p.Output)
				// Plan 6, item 1: the exit code comes from the item, not the text.
				if strings.HasPrefix(p.Name, "commandExecution") && !p.HasExitCode {
					t.Errorf("%s: failed command %s carries no exit code", c.file, p.ToolID)
				}
			}
		}
		if len(got) != len(c.want) {
			t.Errorf("%s: failed calls %q, want %d", c.file, got, len(c.want))
			continue
		}
		for i, w := range c.want {
			if !strings.HasPrefix(got[i], w) {
				t.Errorf("%s: failed call %q, want it to start %q", c.file, got[i], w)
			}
		}
	}
}

// TestSubagentModel is plan 5, item 7, criterion 2: Codex 0.163.0's
// subAgentActivity gives each child its model (P7b); 0.145.0 reports none
// (P7).
func TestSubagentModel(t *testing.T) {
	for _, c := range []struct {
		file string
		want map[string]string
	}{
		{"p7-codex-subagents.jsonl", map[string]string{}},
		{"p7b-codex-subagents.jsonl", map[string]string{
			"01a14937-2992-7d51-8e12-7ff3613a8069": "gpt-6-sol", // /root/list_files
			"01a14937-2f86-7543-9d91-e0a0e52125cc": "gpt-6-sol", // /root/create_found
		}},
		// P7c (plan 7, item 4): asked for a smaller model, Codex 0.163.0
		// sends no spawnAgent item; subAgentActivity carries the model.
		{"p7c-codex-spawn-model.jsonl", map[string]string{
			"01a14b87-c26f-7a73-aebb-233a7dd48103": "gpt-6-luna", // /root/count_files
		}},
	} {
		_, per := replay(t, c.file)
		got := map[string]string{}
		for _, e := range fixture.Flatten(per) {
			if p, ok := e.Payload.(event.Agent); ok && p.Model != "" {
				if e.Kind != event.AgentStarted {
					t.Errorf("%s: model %q on %s", c.file, p.Model, e.Kind)
				}
				got[p.Key] = p.Model
			}
		}
		if len(got) != len(c.want) {
			t.Errorf("%s: models %v, want %v", c.file, got, c.want)
		}
		for k, w := range c.want {
			if got[k] != w {
				t.Errorf("%s: %s model %q, want %q", c.file, k, got[k], w)
			}
		}
	}
}

// TestOfferedDecisions is plan 5, item 8, criterion 1, and plan 8,
// criterion 1: a request's availableDecisions strings are kept, never its
// amendments (objects); a readable command or file-change request takes an
// allow for the session unless its list omits acceptForSession. No request
// of P6b, P7 or P7b lists any (Codex 0.163.0), so each takes it; P8's
// command request lists accept, the amendment and cancel, so it does not.
func TestOfferedDecisions(t *testing.T) {
	for _, name := range []string{"p6b-codex-app-server.jsonl", "p7-codex-subagents.jsonl", "p7b-codex-subagents.jsonl"} {
		_, per := replay(t, name)
		n := 0
		for _, e := range fixture.Flatten(per) {
			if a, ok := e.Payload.(event.Approval); ok && e.Kind == event.ApprovalRequest {
				n++
				if a.Offered != nil || !a.ForSession || a.Problem != "" {
					t.Errorf("%s: %s offered %q, for session %v, problem %q", name, a.Tool, a.Offered, a.ForSession, a.Problem)
				}
			}
		}
		if name == "p6b-codex-app-server.jsonl" && n == 0 {
			t.Errorf("%s: no approval request", name)
		}
	}
	const cmd = `{"id":3,"method":"item/commandExecution/requestApproval","params":{"threadId":"t","turnId":"u","itemId":"i","command":"go vet"`
	const file = `{"id":4,"method":"item/fileChange/requestApproval","params":{"threadId":"t","turnId":"u","itemId":"i"`
	// Probe P8 (Codex 0.163.0, 2026-10-17, the dashboard's handshake): an
	// escalated command request as recorded, its cwd replaced.
	const p8 = `{"method":"item/commandExecution/requestApproval","id":0,"params":{"kind":"command","threadId":"01a14bbf-391e-74d0-8c2c-647c3c7a1f0f","turnId":"01a14bbf-3f49-7673-b86f-52293c852cc1","itemId":"exec-a7a62e76-1f25-4707-b8a8-ae09ebb799c8","startedAtMs":1792272190903,"environmentId":"local","reason":"plan 8 test","command":"\"C:\\\\windows\\\\System32\\\\WindowsPowerShell\\\\v1.0\\\\powershell.exe\" -Command 'Add-Content -Path note.txt -Value one'","cwd":"<probe-work>","commandActions":[{"type":"unknown","command":"Add-Content -Path note.txt -Value one"}],"proposedExecpolicyAmendment":["Add-Content","-Path","note.txt","-Value","one"],"availableDecisions":["accept",{"acceptWithExecpolicyAmendment":{"execpolicy_amendment":["Add-Content","-Path","note.txt","-Value","one"]}},"cancel"]}}`
	for _, c := range []struct {
		line       string
		want       []string
		forSession bool
		problem    bool
	}{
		{p8, []string{"accept", "cancel"}, false, false},
		// L3's request, Codex 0.145.0 (plan.md, plan 4, step 5 rerun).
		{cmd + `,"availableDecisions":["accept",{"acceptWithExecpolicyAmendment":{"execpolicy_amendment":["go","vet"]}},"cancel"]}}`, []string{"accept", "cancel"}, false, false},
		{cmd + `,"availableDecisions":["accept","acceptForSession",{"applyNetworkPolicyAmendment":{}},"decline","cancel"]}}`, []string{"accept", "acceptForSession", "decline", "cancel"}, true, false},
		{cmd + `,"availableDecisions":[]}}`, nil, false, false},
		{cmd + `,"availableDecisions":null}}`, nil, true, false},
		{cmd + `}}`, nil, true, false},
		{file + `}}`, nil, true, false},
		{file + `,"availableDecisions":["accept","decline","cancel"]}}`, []string{"accept", "decline", "cancel"}, false, false},
		{`{"id":5,"method":"item/fileChange/requestApproval","params":{"threadId":"t","turnId":"u"}}`, nil, false, true},
		{cmd + `,"availableDecisions":"accept"}}`, nil, false, true},
	} {
		evs := NewMapper().Out([]byte(c.line))
		if len(evs) != 1 || evs[0].Kind != event.ApprovalRequest {
			t.Fatalf("%s: %+v", c.line, evs)
		}
		a := evs[0].Payload.(event.Approval)
		if !slices.Equal(a.Offered, c.want) || a.ForSession != c.forSession || (a.Problem != "") != c.problem {
			t.Errorf("%s: offered %q, for session %v, problem %q; want %q, %v, problem %v", c.line, a.Offered, a.ForSession, a.Problem, c.want, c.forSession, c.problem)
		}
	}
}

// configReadEvents maps a config/read request and its reply.
func configReadEvents(t *testing.T, reply string) []event.Event {
	t.Helper()
	m := NewMapper()
	m.In([]byte(`{"id":7,"method":"config/read","params":{"cwd":"C:/x","includeLayers":false}}`))
	return m.Out([]byte(reply))
}

func limitOf(t *testing.T, evs []event.Event) event.Metric {
	t.Helper()
	for _, e := range evs {
		if p, ok := e.Payload.(event.Metrics); ok && len(p.Items) == 1 && p.Items[0].Name == event.LimitMetric {
			return p.Items[0]
		}
	}
	t.Fatalf("no limit event in %+v", evs)
	return event.Metric{}
}

// TestConfigReadKeepsOnlyTheLimit: the reply can hold secrets; the limit is
// read and no event carries a byte of the result.
func TestConfigReadKeepsOnlyTheLimit(t *testing.T) {
	evs := configReadEvents(t, `{"id":7,"result":{"config":{"model":"x","mcp_servers":{"x":{"env":{"TOKEN":"SECRET-MARKER-123"}}},"agents":{"max_concurrent_threads_per_session":3,"job_max_runtime_seconds":null}},"origins":{}}}`)
	if m := limitOf(t, evs); m.Value != 3 || m.Text != "" || m.Source != "config/read" {
		t.Errorf("limit %+v", m)
	}
	for _, e := range evs {
		if strings.Contains(e.Raw, "SECRET-MARKER-123") || strings.Contains(fmt.Sprintf("%+v", e), "SECRET-MARKER-123") {
			t.Errorf("event keeps the secret: %+v", e)
		}
	}
	if len(evs) != 1 || !strings.HasPrefix(evs[0].Raw, `{"id":"7","result":"config/read reply not kept`) {
		t.Errorf("raw %q", evs[0].Raw)
	}
}

func TestConfigReadFallbackName(t *testing.T) {
	m := limitOf(t, configReadEvents(t, `{"id":7,"result":{"config":{"agents":{"max_threads":5}}}}`))
	if m.Value != 5 {
		t.Errorf("limit %+v", m)
	}
}

func TestConfigReadNotSet(t *testing.T) {
	m := limitOf(t, configReadEvents(t, `{"id":7,"result":{"config":{"model":"x","mcp_servers":{"x":{"env":{"TOKEN":"SECRET-MARKER-123"}}}},"origins":{}}}`))
	if m.Text != event.LimitNotSet || m.Value != 0 {
		t.Errorf("limit %+v", m)
	}
}

// TestConfigReadV2Limit: features.multi_agent_v2's limit counts the main thread
// and applies on V2 only; the backend is not reported, so a limit is shown
// only when both backends agree (true or false: no limit of its own).
func TestConfigReadV2Limit(t *testing.T) {
	for _, c := range []struct {
		name, config string
		want         float64
		text         string
	}{
		{"v2 only", `{"features":{"multi_agent_v2":{"max_concurrent_threads_per_session":5}}}`, 0, event.LimitPerBackend},
		{"v2 one, agents unset", `{"features":{"multi_agent_v2":{"max_concurrent_threads_per_session":1}}}`, 0, event.LimitPerBackend},
		{"both differ", `{"agents":{"max_threads":3},"features":{"multi_agent_v2":{"max_concurrent_threads_per_session":3}}}`, 0, event.LimitPerBackend},
		{"both agree", `{"agents":{"max_threads":3},"features":{"multi_agent_v2":{"max_concurrent_threads_per_session":4}}}`, 3, ""},
		{"v2 true", `{"agents":{"max_threads":3},"features":{"multi_agent_v2":true}}`, 3, ""},
		{"v2 true, agents unset", `{"features":{"multi_agent_v2":true}}`, 0, event.LimitNotSet},
		{"v2 table without limit", `{"features":{"multi_agent_v2":{"enabled":true}}}`, 0, event.LimitNotSet},
	} {
		m := limitOf(t, configReadEvents(t, `{"id":7,"result":{"config":`+c.config+`}}`))
		if m.Value != c.want || m.Text != c.text {
			t.Errorf("%s: limit %+v", c.name, m)
		}
	}
}

func TestConfigReadRefused(t *testing.T) {
	evs := configReadEvents(t, `{"id":7,"error":{"code":-32600,"message":"config unavailable"}}`)
	if len(evs) != 1 || evs[0].Kind != event.Error {
		t.Fatalf("events %+v", evs)
	}
	if p := evs[0].Payload.(event.Problem); p.Message != "config/read refused by Codex: config unavailable" {
		t.Errorf("message %q", p.Message)
	}
}

// TestSpawnRequestModel is plan 6, item 4, criteria 1 and 3. The fixture is
// built from Codex 0.163.0's generated schema (collabAgentToolCall: `model`,
// "Model requested for the spawned agent, when applicable", and
// `receiverThreadIds`), not from a recording: none carries a model there.
func TestSpawnRequestModel(t *testing.T) {
	line := func(tool, model string) []byte {
		m := ""
		if model != "" {
			m = `"model":"` + model + `",`
		}
		return []byte(`{"method":"item/completed","params":{"threadId":"th0","turnId":"u1","item":{"type":"collabAgentToolCall","id":"c1","tool":"` + tool + `",` + m +
			`"status":"completed","senderThreadId":"th0","receiverThreadIds":["th1","th2"],"prompt":"go","agentsStates":{"th1":{"status":"running","message":null}}}}}`)
	}
	requested := func(evs []event.Event) map[string]string {
		got := map[string]string{}
		for _, e := range evs {
			if p, ok := e.Payload.(event.Agent); ok && p.Model != "" {
				if !p.ModelRequested || e.Kind != event.AgentProgress {
					t.Errorf("model %q on %s, requested %v", p.Model, e.Kind, p.ModelRequested)
				}
				got[p.Key] = p.Model
			}
		}
		return got
	}
	mp := NewMapper()
	mp.main = "thm" // th0, the caller, is a subagent: its spawns nest under it
	evs := mp.Out(line("spawnAgent", "gpt-6-mini"))
	got := requested(evs)
	if len(got) != 2 || got["th1"] != "gpt-6-mini" || got["th2"] != "gpt-6-mini" {
		t.Errorf("spawnAgent with a model: %v", got)
	}
	// Each request names the caller as parent, as agentsStates' events do.
	parents := map[bool]map[string]bool{true: {}, false: {}}
	for _, e := range evs {
		if p, ok := e.Payload.(event.Agent); ok {
			parents[p.ModelRequested][p.Parent] = true
		}
	}
	if !parents[true]["th0"] || len(parents[true]) != 1 || fmt.Sprint(parents[true]) != fmt.Sprint(parents[false]) {
		t.Errorf("parents: requested %v, states %v", parents[true], parents[false])
	}
	if got := requested(NewMapper().Out(line("spawnAgent", ""))); len(got) != 0 {
		t.Errorf("spawnAgent without a model: %v", got)
	}
	if got := requested(NewMapper().Out(line("sendInput", "gpt-6-mini"))); len(got) != 0 {
		t.Errorf("sendInput: %v", got)
	}
}

// TestSpawnRequestRecorded is plan 9, item 1, criterion 2: probe P9a (Codex
// 0.163.0, the orchestrator on gpt-5.6-luna, whose catalogue entry declares
// multi-agent v1) recorded a spawnAgent item carrying the requested model
// and effort. The completed item names the spawned thread; the started one
// names none, so it gives nothing. No subAgentActivity reports the child's
// own values. The caller is the orchestrator, whose key is "". Plan 10,
// item 1, criterion 1: the same event gives the child its prompt's label
// and the whole prompt as its description, Codex naming it nowhere else.
func TestSpawnRequestRecorded(t *testing.T) {
	const main, child = "", "01a14bd7-1754-76f2-b210-de4cf274f58b"
	_, per := replay(t, "p9a-codex-spawn-request.jsonl")
	var got []string
	for _, e := range fixture.Flatten(per) {
		if p, ok := e.Payload.(event.Agent); ok && (p.Model != "" || p.Effort != "") {
			got = append(got, fmt.Sprintf("%s %s parent=%q %s %s requested=%v label=%q description=%q", e.Kind, p.Key, p.Parent, p.Model, p.Effort, p.ModelRequested, p.Label, p.Description))
		}
	}
	want := []string{fmt.Sprintf("%s %s parent=%q gpt-5.6-luna low requested=true label=%q description=%q", event.AgentProgress, child, main,
		"task: Count the files in the c…", "Count the files in the current directory and reply with the number only.")}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Errorf("model and effort events:\n got %q\nwant %q", got, want)
	}
}

// TestAgentEffort is plan 6, item 6, criterion 1: Codex 0.163.0's
// subAgentActivity gives each child its reasoning effort beside its model
// (P7b; 0.145.0 reports none, P7), and a spawnAgent call's reasoningEffort,
// "Reasoning effort requested for the spawned agent, when applicable" (the
// generated schema; no recording carries one), is sent as requested, with
// or without a model; null sends none, the prompt's label alone (plan 10,
// item 1).
func TestAgentEffort(t *testing.T) {
	for _, c := range []struct {
		file string
		want map[string]string
	}{
		{"p7-codex-subagents.jsonl", map[string]string{}},
		{"p7b-codex-subagents.jsonl", map[string]string{
			"01a14937-2992-7d51-8e12-7ff3613a8069": "high", // /root/list_files
			"01a14937-2f86-7543-9d91-e0a0e52125cc": "high", // /root/create_found
		}},
		{"p7c-codex-spawn-model.jsonl", map[string]string{
			"01a14b87-c26f-7a73-aebb-233a7dd48103": "low", // /root/count_files
		}},
	} {
		_, per := replay(t, c.file)
		got := map[string]string{}
		for _, e := range fixture.Flatten(per) {
			if p, ok := e.Payload.(event.Agent); ok && p.Effort != "" {
				if e.Kind != event.AgentStarted || p.ModelRequested {
					t.Errorf("%s: effort %q on %s, requested %v", c.file, p.Effort, e.Kind, p.ModelRequested)
				}
				got[p.Key] = p.Effort
			}
		}
		if fmt.Sprint(got) != fmt.Sprint(c.want) {
			t.Errorf("%s: efforts %v, want %v", c.file, got, c.want)
		}
	}
	spawn := func(fields string) []event.Event {
		return NewMapper().Out([]byte(`{"method":"item/completed","params":{"threadId":"th0","turnId":"u1","item":{"type":"collabAgentToolCall","id":"c1","tool":"spawnAgent",` + fields +
			`"status":"completed","senderThreadId":"th0","receiverThreadIds":["th1"],"prompt":"go","agentsStates":{}}}}`))
	}
	for _, c := range []struct{ fields, model, effort string }{
		{`"model":"gpt-6-mini","reasoningEffort":"low",`, "gpt-6-mini", "low"},
		{`"model":null,"reasoningEffort":"medium",`, "", "medium"},
		{`"model":null,"reasoningEffort":null,`, "", ""},
	} {
		var got []event.Agent
		for _, e := range spawn(c.fields) {
			if p, ok := e.Payload.(event.Agent); ok && e.Kind == event.AgentProgress && p.ModelRequested {
				got = append(got, p)
			}
		}
		switch {
		case c.effort == "" && (len(got) != 1 || got[0].Model != "" || got[0].Effort != "" || got[0].Label != "task: go"):
			t.Errorf("%s: %+v", c.fields, got)
		case c.effort != "" && (len(got) != 1 || got[0].Key != "th1" || got[0].Model != c.model || got[0].Effort != c.effort):
			t.Errorf("%s: %+v", c.fields, got)
		}
	}
}

// TestSpawnLabel is plan 10, item 1, criterion 1: a spawn request's label is
// "task: " and its prompt's first non-blank line, cut at 24 characters with
// "…"; a blank prompt gives none, and so does a label-only request without
// receivers.
func TestSpawnLabel(t *testing.T) {
	for _, c := range []struct{ prompt, want string }{
		{"go", "task: go"},
		{"Count the files in the current directory.", "task: Count the files in the c…"},
		{"Create found.txt cont", "task: Create found.txt cont"},
		{"Twenty-four characters!!", "task: Twenty-four characters!!"},
		{"Twenty-four characters!!!", "task: Twenty-four characters!!…"},
		{"Cut before a space here x", "task: Cut before a space here…"},
		{"\n  \n  Second line first\nthird", "task: Second line first"},
		{"   \n\t", ""},
		{"", ""},
	} {
		if got := spawnLabel(c.prompt); got != c.want {
			t.Errorf("spawnLabel(%q) = %q, want %q", c.prompt, got, c.want)
		}
	}
	for _, prompt := range []string{"", "  "} {
		evs := NewMapper().Out([]byte(`{"method":"item/completed","params":{"threadId":"th0","turnId":"u1","item":{"type":"collabAgentToolCall","id":"c1","tool":"spawnAgent","model":null,"reasoningEffort":null,"status":"completed","senderThreadId":"th0","receiverThreadIds":["th1"],"prompt":"` + prompt + `","agentsStates":{}}}}`))
		for _, e := range evs {
			if p, ok := e.Payload.(event.Agent); ok && p.ModelRequested {
				t.Errorf("prompt %q: %+v", prompt, p)
			}
		}
	}
}

// Plan 12, item 2, criteria 3 and 6: the reviewer echoed by thread/start, the
// auto-review notifications as one review each, and guardianWarning mapped.
func TestAutoReviewEvents(t *testing.T) {
	_, per := replay(t, "p12x-codex-auto-review.jsonl")
	all := fixture.Flatten(per)
	if n := count(all, event.ApprovalMode); n != 2 {
		t.Errorf("%d mode events, want 2 (the thread/start echo and the turn)", n)
	}
	var reviews []event.Review
	for _, e := range all {
		if e.Kind == event.ReviewUpdated {
			if e.Origin != event.Observed {
				t.Errorf("review origin %q", e.Origin)
			}
			reviews = append(reviews, e.Payload.(event.Review))
		}
	}
	if len(reviews) != 4 {
		t.Fatalf("%d review updates, want 4 (two reviews, started then completed)", len(reviews))
	}
	for _, r := range reviews {
		switch r.Status {
		case "inProgress":
			if r.Command == "" {
				t.Errorf("started review without its command: %+v", r)
			}
		case "approved":
			if r.Risk != "low" || r.Authorization != "high" || r.Rationale == "" {
				t.Errorf("completed review %+v", r)
			}
		default:
			t.Errorf("status %q", r.Status)
		}
	}
	if n := count(all, event.ReviewWarning); n != 2 {
		t.Errorf("%d guardian warnings, want 2", n)
	}
}

// The plan 12 addendum: each turn/start's sandbox comes with its mode, and
// the state shows the one the last turn named (P12x-sandbox: two Auto turns
// in workspace-write, then a Manual turn in read-only).
func TestTurnSandbox(t *testing.T) {
	_, per := replay(t, "p12x-sandbox-codex-turn-workspace-write.jsonl")
	var got []string
	st := state.New()
	for _, e := range fixture.Flatten(per) {
		st.Apply(e)
		if m, ok := e.Payload.(event.Mode); ok && e.Kind == event.ApprovalMode && m.Sandbox != "" {
			got = append(got, m.Mode+" "+m.Sandbox)
			if st.Session.Sandbox != m.Sandbox {
				t.Errorf("sandbox shown %q after %q", st.Session.Sandbox, m.Sandbox)
			}
		}
	}
	want := []string{"auto workspaceWrite", "auto workspaceWrite", "manual readOnly"}
	if !slices.Equal(got, want) {
		t.Errorf("turn modes %q, want %q", got, want)
	}
	if st.Session.Sandbox != "readOnly" || st.Mode() != state.ModeManual {
		t.Errorf("at the end: sandbox %q, mode %q", st.Session.Sandbox, st.Mode())
	}
}
