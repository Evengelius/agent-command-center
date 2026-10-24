package codex

import (
	"encoding/json"
	"maps"
	"strings"
	"testing"
	"time"

	"github.com/Evengelius/agent-command-center/internal/event"
	"github.com/Evengelius/agent-command-center/internal/fixture"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// p39Updated is P39x's recorded thread/settings/updated with that reviewer
// and sandbox, its thread made thread, as a recorded output.
func p39Updated(t *testing.T, thread, reviewer, sandbox string) string {
	t.Helper()
	recs, err := fixture.Read(probe("p39x-codex-permission-modes.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range recs {
		var m struct {
			Method string
			Params struct {
				ThreadID string `json:"threadId"`
				Settings struct {
					Reviewer string                `json:"approvalsReviewer"`
					Sandbox  struct{ Type string } `json:"sandboxPolicy"`
				} `json:"threadSettings"`
			}
		}
		if r.Dir == "out" && json.Unmarshal(r.Msg, &m) == nil && m.Method == "thread/settings/updated" &&
			m.Params.Settings.Reviewer == reviewer && m.Params.Settings.Sandbox.Type == sandbox {
			return `{"dir":"out","msg":` + strings.ReplaceAll(string(r.Msg), m.Params.ThreadID, thread) + `}`
		}
	}
	t.Fatalf("P39x has no thread/settings/updated with %s and %s", reviewer, sandbox)
	return ""
}

// TestModeDuringTurn is criterion 39.4 during a turn: from the main
// turn/start to its turn/completed a mode asked for writes nothing; at the
// turn's end one thread/settings/update with the mode's three settings,
// answered by P39x's notification, the mode then in force.
func TestModeDuringTurn(t *testing.T) {
	recs := []string{
		`{"dir":"in","msg":{"id":8,"method":"turn/start","params":{"threadId":"` + t38 + `"}}}`,
		`{"dir":"out","msg":{"id":8,"result":{"turn":{"id":"tu-39","status":"inProgress","items":[]}}}}`,
		`{"dir":"out","msg":{"method":"turn/started","params":{"threadId":"` + t38 + `","turn":{"id":"tu-39","status":"inProgress","items":[]}}}}`,
		`{"dir":"in","msg":{"id":9,"method":"turn/interrupt","params":{"threadId":"` + t38 + `"}}}`,
		`{"dir":"out","msg":{"id":9,"result":{}}}`,
		`{"dir":"out","msg":{"method":"turn/completed","params":{"threadId":"` + t38 + `","turn":{"id":"tu-39","status":"interrupted","items":[]}}}}`,
	}
	recs = append(recs, update38(`{"dir":"out","msg":{"id":7,"result":{}}}`, p39Updated(t, t38, "auto_review", "workspaceWrite"))...)
	r := start38(t, nil, recs...)
	r.send("hello")
	r.wait(event.TurnStarted)
	if err := r.sess.SetApprovalMode(event.ModeAuto); err != nil {
		t.Fatal(err)
	}
	if u := updates(r); len(u) != 0 {
		t.Errorf("thread/settings/update written during the turn: %q", u)
	}
	if err := r.sess.Interrupt(); err != nil {
		t.Fatal(err)
	}
	r.wait(event.TurnCompleted)
	for _, e := range r.evs {
		if p, ok := e.Payload.(event.Mode); ok && p.Answers {
			t.Errorf("the mode answered before the turn's end: %+v", p)
		}
	}
	ev := r.waitFor("the mode's answer", func(ev event.Event) bool {
		p, ok := ev.Payload.(event.Mode)
		return ok && p.Answers
	})
	if p := ev.Payload.(event.Mode); p.Mode != event.ModeAuto || p.Sandbox != "workspaceWrite" {
		t.Errorf("the answer %+v", p)
	}
	// The fake logs a line once it reads it: both lines, then their order.
	at, interrupt := -1, -1
	for wait := time.Now().Add(5 * time.Second); (at < 0 || interrupt < 0) && time.Now().Before(wait); time.Sleep(20 * time.Millisecond) {
		at, interrupt = -1, -1
		for i, l := range r.written() {
			switch {
			case strings.Contains(l, `"method":"thread/settings/update"`):
				at = i
			case strings.Contains(l, `"method":"turn/interrupt"`):
				interrupt = i
			}
		}
	}
	want := `{"threadId":"` + t38 + `","sandboxPolicy":{"type":"workspaceWrite"},"approvalPolicy":"on-request","approvalsReviewer":"auto_review"}`
	if u := updates(r); len(u) != 1 || u[0] != want || at < interrupt {
		t.Errorf("thread/settings/update written %q, line %d, the interrupt's %d", u, at, interrupt)
	}
	if r.st.Mode() != state.ModeAuto {
		t.Errorf("mode %q in the state", r.st.Mode())
	}
}

// TestModeNextTurnAfterRefusal is criterion 39.4 with a Codex that does
// not take thread/settings/update (plan 38, design D4): the mode asked for
// goes with the next turn/start, the state told so, the turn then putting
// it in force.
func TestModeNextTurnAfterRefusal(t *testing.T) {
	recs := update38(`{"dir":"out","msg":{"id":7,"error":{"code":-32601,"message":"Method not found"}}}`)
	recs = append(recs,
		`{"dir":"in","msg":{"id":8,"method":"turn/start","params":{"threadId":"`+t38+`"}}}`,
		`{"dir":"out","msg":{"id":8,"result":{"turn":{"id":"tu-39","status":"inProgress","items":[]}}}}`,
	)
	r := start38(t, nil, recs...)
	r.st.Do(state.ChooseMode{Mode: event.ModeAuto, Capable: true})
	if err := r.sess.SetApprovalMode(event.ModeAuto); err != nil {
		t.Fatal(err)
	}
	ev := r.waitFor("the fallback", func(ev event.Event) bool {
		p, ok := ev.Payload.(event.Mode)
		return ok && p.NextTurn
	})
	if p := ev.Payload.(event.Mode); p.Text != "Codex takes mode changes with the next message: thread/settings/update refused by Codex: Method not found" {
		t.Errorf("the fallback %+v", p)
	}
	if !r.sess.Capabilities().ModelsNextTurn || r.st.Mode() != state.ModeManual || r.st.Session.ApprovalRequested != state.ModeAuto || !r.st.Session.ModeNextTurn {
		t.Errorf("next turn %v, mode %q, requested %q, marked %v", r.sess.Capabilities().ModelsNextTurn, r.st.Mode(), r.st.Session.ApprovalRequested, r.st.Session.ModeNextTurn)
	}
	r.send("hello")
	r.waitFor("Auto in force", func(ev event.Event) bool {
		p, ok := ev.Payload.(event.Mode)
		return ok && p.Mode == event.ModeAuto
	})
	// The fake logs a line once it reads it, which may come after the event.
	var turn string
	for wait := time.Now().Add(5 * time.Second); turn == "" && time.Now().Before(wait); time.Sleep(20 * time.Millisecond) {
		for _, l := range r.written() {
			if strings.Contains(l, `"method":"turn/start"`) {
				turn = l
			}
		}
	}
	if !strings.Contains(turn, `"approvalPolicy":"on-request","approvalsReviewer":"auto_review"`) || !strings.Contains(turn, `"sandboxPolicy":{"type":"workspaceWrite"}`) {
		t.Errorf("the turn/start does not carry Auto: %s", turn)
	}
	if r.st.Mode() != state.ModeAuto || r.st.Session.ApprovalRequested != "" {
		t.Errorf("after the turn: mode %q, requested %q", r.st.Mode(), r.st.Session.ApprovalRequested)
	}
}

// TestModeUpdateRefusals is criterion 39.4's refusals on the mapper: a
// refusal saying Codex does not take the method leaves the mode to the
// next message; any other refuses it, zone B saying why in Codex's words;
// an update with a model and a mode answers both.
func TestModeUpdateRefusals(t *testing.T) {
	full := `{"threadId":"th","sandboxPolicy":{"type":"dangerFullAccess"},"approvalPolicy":"never","approvalsReviewer":"user"}`
	refuse := func(params, code, msg string) []event.Event {
		m := NewMapper()
		m.In([]byte(`{"id":7,"method":"thread/settings/update","params":` + params + `}`))
		return m.Out([]byte(`{"id":7,"error":{"code":` + code + `,"message":` + quote(msg) + `}}`))
	}
	for _, c := range []struct{ code, msg string }{{"-32601", "Method not found"}, {"-32600", "thread/settings/update requires experimentalApi capability"}} {
		evs := refuse(full, c.code, c.msg)
		want := event.Mode{NextTurn: true, Text: "Codex takes mode changes with the next message: thread/settings/update refused by Codex: " + c.msg}
		if len(evs) != 1 || evs[0].Kind != event.ApprovalMode || evs[0].Payload != want {
			t.Errorf("%s %q: %+v", c.code, c.msg, evs)
		}
	}
	why := "thread/settings/update refused by Codex: invalid params"
	evs := refuse(full, "-32602", "invalid params")
	if len(evs) != 2 || evs[0].Payload != (event.Problem{Message: why}) || evs[1].Payload != (event.Mode{Refused: true, Text: "the mode was not changed: " + why}) {
		t.Errorf("-32602: %+v", evs)
	}
	both := `{"threadId":"th","effort":"low","sandboxPolicy":{"type":"readOnly"},"approvalPolicy":"on-request","approvalsReviewer":"user"}`
	evs = refuse(both, "-32602", "invalid params")
	if len(evs) != 3 || evs[1].Kind != event.ModelSettings || evs[2].Kind != event.ApprovalMode {
		t.Errorf("a model and a mode: %+v", evs)
	}
}

// TestModeLimits is plan 39's design D1 and criterion 39.5: the managed
// requirements' sandboxes and approval policies leave modes out, each with
// why; none set leaves none out; a refused read says so and leaves none
// out (an older Codex: every mode shown, Codex refusing what it does not
// allow).
func TestModeLimits(t *testing.T) {
	read := func(reply string) []event.Event {
		m := NewMapper()
		m.In([]byte(`{"id":3,"method":"configRequirements/read"}`))
		return m.Out([]byte(reply))
	}
	full := "Codex's requirements do not allow "
	for _, c := range []struct {
		reply string
		want  map[string]string
	}{
		{`{"id":3,"result":{"supportsIndependentSpeedModes":true,"requirements":null}}`, nil}, // P39x
		{`{"id":3,"result":{"requirements":{}}}`, nil},
		{`{"id":3,"result":{"requirements":{"allowedSandboxModes":["read-only","workspace-write"]}}}`,
			map[string]string{event.ModeFull: full + "the sandbox danger-full-access"}},
		{`{"id":3,"result":{"requirements":{"allowedApprovalPolicies":["on-request","untrusted"]}}}`,
			map[string]string{event.ModeFull: full + "the approval policy never"}},
		{`{"id":3,"result":{"requirements":{"allowedSandboxModes":["read-only"],"allowedApprovalPolicies":["on-request"]}}}`,
			map[string]string{event.ModeAuto: full + "the sandbox workspace-write", event.ModeFull: full + "the sandbox danger-full-access or the approval policy never"}},
		{`{"id":3,"result":{"requirements":{"allowedSandboxModes":["workspace-write","danger-full-access"],"allowedApprovalPolicies":["never",{"granular":{}}]}}}`,
			map[string]string{event.ModeManual: full + "the sandbox read-only or the approval policy on-request", event.ModeAuto: full + "the approval policy on-request"}},
	} {
		evs := read(c.reply)
		if len(evs) != 1 || evs[0].Kind != event.ModesAllowed {
			t.Fatalf("%s: events %+v", c.reply, evs)
		}
		if l := evs[0].Payload.(event.ModeLimits); !maps.Equal(l.Unavailable, c.want) || l.Source != "configRequirements/read" {
			t.Errorf("%s:\n got %+v\nwant %v", c.reply, l, c.want)
		}
	}
	evs := read(`{"id":3,"error":{"code":-32601,"message":"method not found"}}`)
	for _, e := range evs {
		if e.Kind == event.ModesAllowed {
			t.Errorf("a refused read left modes out: %+v", e)
		}
	}
	if len(evs) == 0 || !strings.Contains(evs[len(evs)-1].Raw+evs[len(evs)-1].Payload.(event.Status).Detail, "configRequirements/read refused: method not found") {
		t.Errorf("a refused read: %+v", evs)
	}
}

// TestCodexModeOf is plan 39's design D2: the three settings Codex reports
// name one of the presets, the CLI's spellings and its other auto-reviewer
// read as VS Code's extension reads them, any other combination Custom,
// and a turn/start naming no sandbox read by its reviewer, as plan 12 read
// it.
func TestCodexModeOf(t *testing.T) {
	for _, c := range []struct{ sandbox, policy, reviewer, want string }{
		{"readOnly", "on-request", "user", event.ModeManual},
		{"read-only", "", "", event.ModeManual}, // thread/start's form: the policy and reviewer left to Codex's defaults
		{"workspaceWrite", "on-request", "auto_review", event.ModeAuto},
		{"workspace-write", "on-request", "guardian_subagent", event.ModeAuto},
		{"dangerFullAccess", "never", "user", event.ModeFull},
		{"danger-full-access", "never", "", event.ModeFull},
		{"workspaceWrite", "on-request", "user", event.ModeCustom}, // VS Code's workspace-write "Ask for approval" (P39x)
		{"dangerFullAccess", "on-request", "user", event.ModeCustom},
		{"readOnly", "never", "user", event.ModeCustom},
		{"readOnly", "on-request", "auto_review", event.ModeCustom},
		{"readOnly", "granular", "user", event.ModeCustom},
		{"readOnly", "untrusted", "user", event.ModeCustom},
		{"externalSandbox", "on-request", "user", event.ModeCustom},
		{"", "", "user", event.ModeManual}, // P12x's turn/start
		{"", "", "auto_review", event.ModeAuto},
		{"", "", "someone", event.ModeCustom},
		{"", "", "", ""},
	} {
		if got := codexMode(c.sandbox, c.policy, c.reviewer); got != c.want {
			t.Errorf("codexMode(%q, %q, %q) = %q, want %q", c.sandbox, c.policy, c.reviewer, got, c.want)
		}
	}
}
