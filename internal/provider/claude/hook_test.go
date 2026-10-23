package claude

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Evengelius/agent-command-center/internal/event"
	"github.com/Evengelius/agent-command-center/internal/fixture"
	"github.com/Evengelius/agent-command-center/internal/provider"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// Plan 22, amendment 20: the events the owner's SessionStart hook makes Claude
// Code send, and the turn's summary line. P22o is Claude Code 2.1.303 on Opus
// 5.5 with that hook (doc/research/probes/p22o-claude-opus-thinking.jsonl):
// system/hook_started (line 3), system/hook_response (line 4, outcome
// "success", exit code 0) and system/post_turn_summary (line 55) were
// unrecognised; they are known now and give no event, and a hook that fails
// gives one observed notice, not an error: the CLI carries on after it, and an
// error before the initialize response would end the session as a failed
// handshake (criterion 20.7).

const p22o = "p22o-claude-opus-thinking.jsonl"

// quietLine says a recorded line is one the adapter knows and gives no event
// for: a hook's start, the turn's summary, a hook's successful end (outcome
// "success" and exit code 0). It reads the line apart from the mapper.
func quietLine(msg []byte) bool {
	var l struct {
		Type     string `json:"type"`
		Subtype  string `json:"subtype"`
		Outcome  string `json:"outcome"`
		ExitCode *int   `json:"exit_code"`
	}
	if json.Unmarshal(msg, &l) != nil || l.Type != "system" {
		return false
	}
	switch l.Subtype {
	case "hook_started", "post_turn_summary":
		return true
	case "hook_response":
		return l.Outcome == "success" && l.ExitCode != nil && *l.ExitCode == 0
	}
	return false
}

// 20.7: P22o through the adapter has no unrecognised event (it had 3); exactly
// its lines 3, 4 and 55 give no event, and every other line gives the events
// it gave before (54 records with events, 55 events).
func TestHookEventsOfP22o(t *testing.T) {
	recs, per := replay(t, p22o)
	all := fixture.Flatten(per)
	if n := count(all, event.Unrecognised); n != 0 {
		t.Errorf("%d unrecognised events, want none (was 3)", n)
	}
	var quiet []int
	for i, evs := range per {
		if len(evs) == 0 {
			quiet = append(quiet, i+1)
		}
		if quietLine(recs[i].Msg) != (len(evs) == 0) {
			t.Errorf("line %d gave %d events; quiet by its own fields: %v", i+1, len(evs), quietLine(recs[i].Msg))
		}
	}
	if len(quiet) != 3 || quiet[0] != 3 || quiet[1] != 4 || quiet[2] != 55 {
		t.Errorf("lines with no event: %v, want 3, 4 and 55", quiet)
	}
	// 56: the initialize answer's models are an event of their own since
	// plan 34.
	if len(per) != 57 || len(all) != 56 {
		t.Errorf("%d records and %d events, want 57 and 56", len(per), len(all))
	}
	for _, e := range all {
		if strings.Contains(e.Raw, `"hook_`) || strings.Contains(e.Raw, "post_turn_summary") {
			t.Errorf("%s comes from a hook line: %.80s", e.Kind, e.Raw)
		}
	}

	// Its conversation: the owner's prompt, one observed thinking phase with its
	// exact count, and the answer, as before; nothing unrecognised in the state.
	s := state.New()
	for _, e := range all {
		s.Apply(e)
	}
	if s.Unrecognised != 0 {
		t.Errorf("state counts %d unrecognised events", s.Unrecognised)
	}
	var roles []string
	for _, e := range s.Conversation {
		roles = append(roles, e.Role)
	}
	if strings.Join(roles, " ") != "user thinking assistant" {
		t.Fatalf("conversation %v, want user thinking assistant", roles)
	}
	th := s.Conversation[1].Thought
	if th.Inferred || !th.HasTokens || th.Tokens != 928 || th.Start.Sub(fixture.Epoch).Milliseconds() != 2796 || th.End.Sub(fixture.Epoch).Milliseconds() != 10807 {
		t.Errorf("thinking phase %+v, want observed, 928 tokens, 2,796 to 10,807 ms", th)
	}
	if text := s.Conversation[2].Text; !strings.HasPrefix(text, "**59**: each remainder is one less than its divisor") {
		t.Errorf("answer %q", text)
	}
}

// failing is P22o's hook_response (line 4) with the fields a failing hook
// changes: its outcome, exit code and stderr.
func failing(t *testing.T, outcome string, exit string, stderr string) []byte {
	t.Helper()
	recs, err := fixture.Read(filepath.Join("..", "..", "..", "testdata", "probes", p22o))
	if err != nil {
		t.Fatal(err)
	}
	line := string(recs[3].Msg)
	for _, from := range []string{`"stderr":""`, `"exit_code":0`, `"outcome":"success"`} {
		if !strings.Contains(line, from) {
			t.Fatalf("P22o line 4 lacks %s: %s", from, line)
		}
	}
	line = strings.Replace(line, `"stderr":""`, `"stderr":`+quote(stderr), 1)
	line = strings.Replace(line, `"exit_code":0`, exit, 1)
	line = strings.Replace(line, `"outcome":"success"`, `"outcome":`+quote(outcome), 1)
	return []byte(line)
}

// 20.7: a hook response that is a success draws nothing; any other end is one
// observed notice (a status.changed whose status is "notice"), never an error,
// naming the hook, its exit code and the first non-blank line of its stderr,
// that part left out when stderr is blank. The lines are P22o's line 4 with the
// failing fields put in: no recording holds a failing hook.
func TestHookResponse(t *testing.T) {
	const name = "SessionStart:startup"
	for _, c := range []struct {
		what     string
		line     []byte
		want     string // the notice's text; "" for no event
		wantNone bool
	}{
		{"a success with exit code 0", failing(t, "success", `"exit_code":0`, ""), "", true},
		{"a success whose stderr says something", failing(t, "success", `"exit_code":0`, "a warning\n"), "", true},
		{"outcome error, exit code 1, a stderr", failing(t, "error", `"exit_code":1`, "pxw: not found\nsecond line\n"),
			"hook " + name + " failed: exit 1: pxw: not found", false},
		{"a stderr of one line, no newline", failing(t, "error", `"exit_code":1`, "pxw: not found"),
			"hook " + name + " failed: exit 1: pxw: not found", false},
		{"a blank stderr", failing(t, "error", `"exit_code":1`, ""), "hook " + name + " failed: exit 1", false},
		{"a stderr of blank lines only", failing(t, "error", `"exit_code":1`, "\n  \t\n\r\n"), "hook " + name + " failed: exit 1", false},
		{"blank lines before the first", failing(t, "error", `"exit_code":1`, "\n   \n  first real line \t\nsecond"),
			"hook " + name + " failed: exit 1: first real line", false},
		{"windows line ends", failing(t, "error", `"exit_code":2`, "boom\r\nnext\r\n"), "hook " + name + " failed: exit 2: boom", false},
		{"a success outcome with a non-zero exit code", failing(t, "success", `"exit_code":3`, "late\n"),
			"hook " + name + " failed: exit 3: late", false},
		{"an error outcome whose exit code is 0", failing(t, "error", `"exit_code":0`, "odd\n"),
			"hook " + name + " failed: outcome error: odd", false},
		{"a cancelled hook with no exit code", failing(t, "cancelled", `"x":0`, ""), "hook " + name + " failed: outcome cancelled", false},
		{"a success with no exit code", failing(t, "success", `"x":0`, ""), "", true},
		{"no outcome", failing(t, "", `"exit_code":0`, ""), "hook " + name + " failed: no outcome reported", false},
	} {
		evs := NewMapper().Out(c.line)
		if c.wantNone {
			if len(evs) != 0 {
				t.Errorf("%s: %d events, want none: %+v", c.what, len(evs), evs)
			}
			continue
		}
		if len(evs) != 1 || evs[0].Kind != event.StatusChanged || evs[0].Origin != event.Observed || evs[0].Agent != "" {
			t.Errorf("%s: events %+v, want one observed notice of the orchestrator", c.what, evs)
			continue
		}
		if n := count(evs, event.Error); n != 0 {
			t.Errorf("%s: %d error events, want none", c.what, n)
		}
		st, ok := evs[0].Payload.(event.Status)
		if !ok || st.Status != "notice" || st.Detail != c.want {
			t.Errorf("%s: %+v, want a notice of %q", c.what, evs[0].Payload, c.want)
		}
		if evs[0].Raw != string(c.line) {
			t.Errorf("%s: the event keeps another line than the one read", c.what)
		}
	}

	// No name: the notice says "hook" alone.
	bare := NewMapper().Out([]byte(`{"type":"system","subtype":"hook_response","outcome":"error","exit_code":1,"stderr":"x"}`))
	if len(bare) != 1 {
		t.Fatalf("a hook with no name: %+v", bare)
	}
	if st, ok := bare[0].Payload.(event.Status); !ok || bare[0].Kind != event.StatusChanged || st.Status != "notice" || st.Detail != "hook failed: exit 1: x" {
		t.Errorf("a hook with no name: %+v", bare[0].Payload)
	}
}

// hook_started, post_turn_summary and task_summary give no event, whatever
// they hold and whichever agent they come from. The two task_summary lines
// are the ones the owner's scenario 1 run of 2026-10-22 counted as
// unrecognised, as the rerun through the logging wrapper recorded them
// (plan 25, criterion 25.3): the first right after task_started, the second
// right after the turn's result.
func TestHookStartAndSummaryGiveNoEvent(t *testing.T) {
	for _, l := range []string{
		`{"type":"system","subtype":"hook_started"}`,
		`{"type":"system","subtype":"hook_started","hook_id":"h","hook_name":"SessionStart:startup","hook_event":"SessionStart","session_id":"s"}`,
		`{"type":"system","subtype":"hook_started","hook_name":"PreToolUse:Bash","parent_tool_use_id":"toolu_1","agent_id":"a1"}`,
		`{"type":"system","subtype":"post_turn_summary","summarizes_uuid":"u","status_category":"completed","status_detail":"done","needs_action":""}`,
		`{"type":"system","subtype":"post_turn_summary","status_category":"blocked","needs_action":"approve a tool"}`,
		`{"type":"system","subtype":"task_summary","detail":"Px reminder: your local Px proxy on 127.0.0.1:3128 is set up for every tool. If anything shows \"Connection refused\", run ` + "`pxw`" + `. I'm launching one general-purpose subagent to list the files here, write their names to found.txt, and report them back.","uuid":"f61173de-eaba-4c2b-ac37-0f2f0f279c55","session_id":"d8cc1c6e-d2ad-45f0-b306-9f48638ed917"}`,
		`{"type":"system","subtype":"task_summary","detail":null,"uuid":"26b3dac7-0198-40fd-8524-4356128d6bbf","session_id":"d8cc1c6e-d2ad-45f0-b306-9f48638ed917"}`,
		`{"type":"system","subtype":"task_summary","detail":"a summary","agent_id":"a1"}`,
	} {
		if evs := NewMapper().Out([]byte(l)); len(evs) != 0 {
			t.Errorf("%s: %d events, want none: %+v", l, len(evs), evs)
		}
	}
}

// p22oInProject is P22o with its work folder made the fake CLI's placeholder,
// which it fills in with the test's project: the reducer closes a session whose
// system/init names another directory than the chosen one, and P22o ran in
// <probe-root>\probes\p22o\work.
func p22oInProject(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "..", "..", "testdata", "probes", p22o))
	if err != nil {
		t.Fatal(err)
	}
	s := strings.ReplaceAll(string(b), `<probe-root>\\probes\\p22o\\work`, `<probe-work>`)
	if strings.Contains(s, "<probe-root>") {
		t.Fatal("P22o names another directory than its work folder")
	}
	p := filepath.Join(t.TempDir(), p22o)
	if err := os.WriteFile(p, []byte(s), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

// P22o through the real session: its three lines that give no event leave the
// events in order (start's check of Seq), the handshake completes with the
// hook lines before its response, the turn ends, and nothing is unrecognised
// (plan 22, amendment 20: a mapper that gives no event for a line is handled
// by the session).
func TestSessionPlaysP22o(t *testing.T) {
	t.Setenv("ACC_FAKE_FILE", p22oInProject(t))
	r, _ := start(t, "replay", nil)
	r.wait(event.SessionReady)
	if r.st.Session.Phase != state.Ready || r.st.Startup.Phase != state.StartReady {
		t.Fatalf("session %q startup %q (%s): a hook line before the handshake's response broke it", r.st.Session.Phase, r.st.Startup.Phase, r.st.Session.Reason)
	}
	text := "Without using any tool, find the smallest positive integer that leaves the remainders 1, 2, 3, 4 and 5 when divided by 2, 3, 4, 5 and 6."
	r.st.Do(state.Send{Text: text})
	if err := r.sess.Send(provider.Message{Text: text}); err != nil {
		t.Fatal(err)
	}
	if end := r.wait(event.TurnCompleted).Payload.(event.TurnEnd); end.Outcome != "completed" {
		t.Errorf("turn %+v", end)
	}
	for _, e := range r.evs {
		if strings.Contains(e.Raw, `"hook_`) || strings.Contains(e.Raw, "post_turn_summary") {
			t.Errorf("%s comes from a hook line: %.80s", e.Kind, e.Raw)
		}
	}
	if r.st.Unrecognised != 0 || len(r.st.Errors) != 0 {
		t.Errorf("%d unrecognised, %d errors; want none", r.st.Unrecognised, len(r.st.Errors))
	}
	var phases []*state.Thought
	for _, e := range r.st.Conversation {
		if e.Role == "thinking" {
			phases = append(phases, e.Thought)
		}
	}
	if len(phases) != 1 || !phases[0].HasTokens || phases[0].Tokens != 928 {
		t.Errorf("thinking phases %+v, want one with 928 tokens", phases)
	}
}

// p22oFailingHookInProject is p22oInProject with the hook's end (line 4) made a
// failure: outcome "error", exit code 1, stderr "pxw: not found".
func p22oFailingHookInProject(t *testing.T) string {
	t.Helper()
	recs, err := fixture.Read(filepath.Join("..", "..", "..", "testdata", "probes", p22o))
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(p22oInProject(t))
	if err != nil {
		t.Fatal(err)
	}
	s := strings.Replace(string(b), string(recs[3].Msg), string(failing(t, "error", `"exit_code":1`, "pxw: not found\n")), 1)
	if s == string(b) {
		t.Fatal("P22o's line 4 was not found in the file")
	}
	p := filepath.Join(t.TempDir(), "failing-"+p22o)
	if err := os.WriteFile(p, []byte(s), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

// A failing SessionStart hook through the real session and the reducer, as the
// application runs them (the chosen project, the startup screen waiting): its
// end comes before the initialize response, and the session still reaches Ready
// and runs the turn. Claude Code carries on after a failing hook, so the
// dashboard must not end the session (plan 22, amendment 20); the hook's end is
// one notice, ahead of the owner's prompt, and no error.
func TestSessionSurvivesAFailingHook(t *testing.T) {
	const notice = "hook SessionStart:startup failed: exit 1: pxw: not found"
	t.Setenv("ACC_FAKE_FILE", p22oFailingHookInProject(t))
	r, _ := start(t, "replay", nil)
	r.wait(event.SessionReady)
	if r.st.Session.Phase != state.Ready || r.st.Startup.Phase != state.StartReady {
		t.Fatalf("session %q startup %q (%s): a failing hook before the handshake's response ended the session", r.st.Session.Phase, r.st.Startup.Phase, r.st.Session.Reason)
	}
	text := "Without using any tool, find the smallest positive integer that leaves the remainders 1, 2, 3, 4 and 5 when divided by 2, 3, 4, 5 and 6."
	r.st.Do(state.Send{Text: text})
	if err := r.sess.Send(provider.Message{Text: text}); err != nil {
		t.Fatal(err)
	}
	if end := r.wait(event.TurnCompleted).Payload.(event.TurnEnd); end.Outcome != "completed" {
		t.Errorf("turn %+v", end)
	}
	if r.st.Session.Phase != state.Ready || len(r.st.Errors) != 0 || r.st.Unrecognised != 0 {
		t.Errorf("session %q (%s), %d errors, %d unrecognised; want ready, none, none", r.st.Session.Phase, r.st.Session.Reason, len(r.st.Errors), r.st.Unrecognised)
	}
	var roles []string
	for _, e := range r.st.Conversation {
		roles = append(roles, e.Role)
		if e.Role == "notice" && (e.Text != notice || e.Origin != event.Observed) {
			t.Errorf("notice %q (%s), want %q observed", e.Text, e.Origin, notice)
		}
	}
	if strings.Join(roles, " ") != "notice user thinking assistant" {
		t.Errorf("conversation %v, want the notice, then the prompt, the thinking and the answer", roles)
	}
	for _, e := range r.evs {
		if e.Kind == event.Error {
			t.Errorf("an error event: %+v", e.Payload)
		}
	}
}
