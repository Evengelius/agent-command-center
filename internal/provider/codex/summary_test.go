package codex

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Evengelius/agent-command-center/internal/event"
	"github.com/Evengelius/agent-command-center/internal/fixture"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// Plan 22, amendment 20: Codex's reasoning summaries. Every turn/start asks
// for them (criterion 20.1); the adapter knows summaryPartAdded and carries the
// item's summary on its completion (20.2). P22s is Codex 0.163.0 asked for
// "auto", then "detailed" (doc/research/probes/p22s-codex-reasoning-summary.jsonl).

const p22s = "p22s-codex-reasoning-summary.jsonl"

// p22sInProject is P22s with its working directory made the fake app-server's
// own placeholder, which it fills in with the test's project: the reducer
// closes a session whose CLI reports another directory than the chosen one,
// and P22s ran in <probe-root>\probes\p22s\work, not the shape the other
// recordings' directories have.
func p22sInProject(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile(probe(p22s))
	if err != nil {
		t.Fatal(err)
	}
	s := strings.ReplaceAll(string(b), `<probe-root>\\probes\\p22s\\work`, `<probe-work>`)
	if strings.Contains(s, "<probe-root>") {
		t.Fatal("P22s names another directory than its work folder")
	}
	p := filepath.Join(t.TempDir(), p22s)
	if err := os.WriteFile(p, []byte(s), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

// reasoningMarkers are the markers of the recording's reasoning items, with
// the record (1-based line) each came from.
func reasoningMarkers(per [][]event.Event) (started, completed []event.Marker, completedAt []int) {
	for i, evs := range per {
		for _, e := range evs {
			p, ok := e.Payload.(event.Marker)
			if !ok || e.Kind != event.MessageMarker || p.Block != "thinking" {
				continue
			}
			switch p.Name {
			case "item_started":
				started = append(started, p)
			case "item_completed":
				completed = append(completed, p)
				completedAt = append(completedAt, i+1)
			}
		}
	}
	return
}

// 20.2: P22s through the adapter. summaryPartAdded (line 68, unrecognised
// before) gives no event; summaryTextDelta (line 69) stays a hidden thinking
// delta; the second reasoning item's completion (line 70) carries "**Computing
// f(10)**" and the first's (line 23) nothing; no event is unrecognised.
func TestSummaryP22s(t *testing.T) {
	recs, per := replay(t, p22s)
	all := fixture.Flatten(per)
	if n := count(all, event.Unrecognised); n != 0 {
		t.Errorf("%d unrecognised events, want none (was 1, summaryPartAdded)", n)
	}
	if !strings.Contains(string(recs[67].Msg), `"item/reasoning/summaryPartAdded"`) || len(per[67]) != 0 {
		t.Errorf("line 68 (%.60s...) gave %d events, want none", recs[67].Msg, len(per[67]))
	}
	if len(per[68]) != 1 || per[68][0].Kind != event.MessageDelta {
		t.Fatalf("line 69 gave %+v, want one message delta", per[68])
	}
	d, ok := per[68][0].Payload.(event.Delta)
	if !ok || d.Block != "thinking" || d.Text != "**Computing f(10)**" || per[68][0].Origin != event.AI {
		t.Errorf("summaryTextDelta mapped to %+v (origin %q), want a thinking delta of the heading, AI", per[68][0].Payload, per[68][0].Origin)
	}
	started, completed, at := reasoningMarkers(per)
	if len(started) != 2 || len(completed) != 2 || at[0] != 23 || at[1] != 70 {
		t.Fatalf("%d starts, %d completions at lines %v; want 2, 2 at 23 and 70", len(started), len(completed), at)
	}
	for _, p := range started {
		if p.Summary != "" {
			t.Errorf("an item's start carries a summary: %+v", p)
		}
	}
	if completed[0].Summary != "" || completed[1].Summary != "**Computing f(10)**" {
		t.Errorf("summaries %q and %q, want none and \"**Computing f(10)**\"", completed[0].Summary, completed[1].Summary)
	}
}

// A summary is carried as Codex sent it: its parts joined by a blank line, the
// parts with no text left out; none for a missing, null or empty list, or one
// with no text; a field that is not a list of strings keeps the marker and is
// kept visible as one unrecognised event. Only an item's completion carries it.
func TestReasoningSummaryShapes(t *testing.T) {
	line := func(method, summary string) []byte {
		field := ""
		if summary != "" {
			field = `,"summary":` + summary
		}
		return []byte(`{"method":"` + method + `","params":{"threadId":"t","turnId":"u","item":{"type":"reasoning","id":"rs1","content":[]` + field + `}}}`)
	}
	for _, c := range []struct {
		name, summary string
		want          string
		unrecognised  int
	}{
		{"one part", `["**Computing f(10)**"]`, "**Computing f(10)**", 0},
		{"two parts", `["**First**\n\nbody one","**Second**"]`, "**First**\n\nbody one\n\n**Second**", 0},
		{"parts as sent, markdown and all", `["  keeps its spaces  ","line\nbreak"]`, "  keeps its spaces  \n\nline\nbreak", 0},
		{"an empty list", `[]`, "", 0},
		{"null", `null`, "", 0},
		{"no field", ``, "", 0},
		{"parts with no text", `["", "  ", "\n"]`, "", 0},
		{"a blank part between two", `["a","","b"]`, "a\n\nb", 0},
		{"a string", `"**Computing f(10)**"`, "", 1},
		{"a list of objects", `[{"text":"x"}]`, "", 1},
		{"a number", `7`, "", 1},
	} {
		evs := NewMapper().Out(line("item/completed", c.summary))
		var markers []event.Marker
		for _, e := range evs {
			if p, ok := e.Payload.(event.Marker); ok {
				markers = append(markers, p)
			}
		}
		if len(markers) != 1 || markers[0].Name != "item_completed" || markers[0].Block != "thinking" || markers[0].MessageID != "rs1" {
			t.Errorf("%s: markers %+v, want the reasoning item's completion", c.name, markers)
			continue
		}
		if markers[0].Summary != c.want {
			t.Errorf("%s: summary %q, want %q", c.name, markers[0].Summary, c.want)
		}
		if n := count(evs, event.Unrecognised); n != c.unrecognised || len(evs) != 1+c.unrecognised {
			t.Errorf("%s: %d unrecognised among %d events, want %d among %d", c.name, n, len(evs), c.unrecognised, 1+c.unrecognised)
		}
		if c.unrecognised == 1 && len(evs) == 2 {
			if u, ok := evs[1].Payload.(event.Unknown); !ok || u.Type != "item/reasoning" || evs[1].Kind != event.Unrecognised {
				t.Errorf("%s: the kept failure is %+v", c.name, evs[1].Payload)
			}
		}
		// The start of the same item carries none, whatever the line holds.
		for _, e := range NewMapper().Out(line("item/started", `["x"]`)) {
			if p, ok := e.Payload.(event.Marker); ok && (p.Summary != "" || p.Name != "item_started") {
				t.Errorf("%s: the item's start is %+v", c.name, p)
			}
		}
	}
}

// summaryPartAdded is known: a quiet line for the main thread and for a child
// thread, whatever the part's index, and it changes nothing the next line sees.
func TestSummaryPartAddedGivesNoEvent(t *testing.T) {
	m := NewMapper()
	m.Out([]byte(`{"method":"thread/started","params":{"thread":{"id":"main"}}}`))
	for _, thread := range []string{"main", "child"} {
		for _, idx := range []string{"0", "1", "5"} {
			l := `{"method":"item/reasoning/summaryPartAdded","params":{"threadId":"` + thread + `","turnId":"u","itemId":"rs1","summaryIndex":` + idx + `}}`
			if evs := m.Out([]byte(l)); len(evs) != 0 {
				t.Errorf("thread %s, index %s: %d events, want none: %+v", thread, idx, len(evs), evs)
			}
		}
	}
	// A line whose params cannot be read is still unrecognised, not quiet.
	if evs := m.Out([]byte(`{"method":"item/reasoning/summaryPartAdded","params":"oops"}`)); len(evs) != 1 || evs[0].Kind != event.Unrecognised {
		t.Errorf("unreadable params gave %+v, want one unrecognised event", evs)
	}
	// The delta that follows stays a thinking delta, never conversation text.
	evs := m.Out([]byte(`{"method":"item/reasoning/summaryTextDelta","params":{"threadId":"main","turnId":"u","itemId":"rs1","delta":"**x**","summaryIndex":0}}`))
	if len(evs) != 1 || evs[0].Kind != event.MessageDelta || evs[0].Payload.(event.Delta).Block != "thinking" {
		t.Errorf("summaryTextDelta gave %+v, want one thinking delta", evs)
	}
}

// 20.1: every turn/start the session writes carries "summary":"auto" and its
// other parameters as before, in Manual and in Auto, whatever a recording
// asked for ("detailed" in P22s's second turn); thread/start and thread/resume
// carry none. P22s is also played through the real session: its summaryPartAdded
// line gives no event, the events stay in order, and the second item's summary
// reaches the state.
func TestTurnStartAsksForSummaries(t *testing.T) {
	r, _ := start(t, "replay", nil, "", p22sInProject(t))
	r.wait(event.SessionStarted)
	r.limits()
	r.send("how many trailing zeros does 100! have?")
	r.wait(event.TurnCompleted)
	r.send("in how many ways can 10 be written as an ordered sum of 1s and 2s?")
	r.wait(event.TurnCompleted)
	r.close()

	type params map[string]json.RawMessage
	turns, threads := 0, 0
	for _, l := range r.written() {
		var m struct {
			Method string `json:"method"`
			Params params `json:"params"`
		}
		if err := json.Unmarshal([]byte(l), &m); err != nil {
			t.Fatalf("written line %q: %v", l, err)
		}
		switch m.Method {
		case "turn/start":
			turns++
			if string(m.Params["summary"]) != `"auto"` {
				t.Errorf("turn/start %d: summary %s, want \"auto\": %s", turns, m.Params["summary"], l)
			}
			keys := map[string]bool{}
			for k := range m.Params {
				keys[k] = true
			}
			want := map[string]bool{"threadId": true, "input": true, "approvalPolicy": true, "approvalsReviewer": true, "sandboxPolicy": true, "summary": true} // the policy since plan 39
			if len(keys) != len(want) {
				t.Errorf("turn/start %d: parameters %v, want %v", turns, keys, want)
			}
			for k := range want {
				if !keys[k] {
					t.Errorf("turn/start %d: no %s in %s", turns, k, l)
				}
			}
		case "thread/start", "thread/resume":
			threads++
			if _, has := m.Params["summary"]; has || strings.Contains(l, "summary") {
				t.Errorf("%s asks for a summary: %s", m.Method, l)
			}
		}
	}
	if turns != 2 || threads != 1 {
		t.Errorf("%d turn/start and %d thread/start written, want 2 and 1", turns, threads)
	}

	var phases []*state.Thought
	for _, e := range r.st.Conversation {
		if e.Role == "thinking" {
			phases = append(phases, e.Thought)
		}
	}
	if len(phases) != 2 || phases[0].Summary != "" || phases[1].Summary != "**Computing f(10)**" {
		t.Fatalf("phases %+v, want two, the second with the summary", phases)
	}
	if r.st.Unrecognised != 0 || len(r.st.Errors) != 0 {
		t.Errorf("%d unrecognised, %d errors; want none", r.st.Unrecognised, len(r.st.Errors))
	}
}

// 20.1 in Auto: the turn/start of an Auto turn asks for a summary too, with
// its own reviewer and sandbox.
func TestAutoTurnStartAsksForSummaries(t *testing.T) {
	r, _ := start(t, "replay", nil, "", probe("p6b-codex-app-server.jsonl"))
	r.wait(event.SessionStarted)
	r.limits()
	if err := r.sess.SetApprovalMode(event.ModeAuto); err != nil {
		t.Fatal(err)
	}
	r.send("make hello.txt")
	req := r.wait(event.ApprovalRequest).Payload.(event.Approval)
	if err := r.sess.Answer(req.ID, "accept"); err != nil {
		t.Fatal(err)
	}
	r.wait(event.TurnCompleted)
	r.close()
	n := 0
	for _, l := range r.written() {
		if !strings.Contains(l, `"method":"turn/start"`) {
			continue
		}
		n++
		for _, want := range []string{`"approvalsReviewer":"auto_review"`, `"sandboxPolicy":{"type":"workspaceWrite"}`, `"summary":"auto"`} {
			if !strings.Contains(l, want) {
				t.Errorf("Auto turn/start lacks %s: %s", want, l)
			}
		}
	}
	if n != 1 {
		t.Errorf("%d turn/start written, want 1", n)
	}
}
