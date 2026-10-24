package codex

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/Evengelius/agent-command-center/internal/event"
	"github.com/Evengelius/agent-command-center/internal/fixture"
	"github.com/Evengelius/agent-command-center/internal/provider"
)

// Plan 34 (brief amendment 34): the session's models from every page of
// model/list, hidden ones left out; the model and effort in force from the
// thread's start and from thread/read; a change carried by the next
// turn/start, since plan 38 with a Codex that refuses thread/settings/update
// (settings_test.go). P34x started no turn.

const p34x = "p34x-codex-model-list-settings.jsonl"

// TestModelsOfP34x is criterion 2 on the mapper: P34x's seven pages of one
// model make one list of 7, in the CLI's order, the default marked, each
// with its levels and default level; the list with hidden models still
// has 7; the thread starts on gpt-6-sol, High.
func TestModelsOfP34x(t *testing.T) {
	recs, err := fixture.Read(probe(p34x))
	if err != nil {
		t.Fatal(err)
	}
	var lists []event.Models
	var settings []event.Settings
	for _, evs := range fixture.Replay(ProviderID, recs, NewMapper()) {
		for _, e := range evs {
			switch p := e.Payload.(type) {
			case event.Models:
				lists = append(lists, p)
			case event.Settings:
				settings = append(settings, p)
			}
		}
	}
	// Run 1: the paged list, the default page, the one with hidden models;
	// run 2: the default page.
	if len(lists) != 4 {
		t.Fatalf("%d lists, want 4", len(lists))
	}
	want := []string{"gpt-6.1-sol", "gpt-6-astra", "gpt-6-sol", "gpt-6-luna", "gpt-5.6-sol", "gpt-5.6-terra", "gpt-5.6-luna"}
	for i, l := range lists {
		var ids []string
		for _, e := range l.Entries {
			ids = append(ids, e.ID)
		}
		if !reflect.DeepEqual(ids, want) || l.Source != "model/list" {
			t.Errorf("list %d: %q from %q, want %q", i+1, ids, l.Source, want)
		}
	}
	es := lists[0].Entries
	five := []string{"low", "medium", "high", "xhigh", "max"}
	for _, c := range []struct {
		i              int
		name, desc, df string
		def            bool
		efforts        []string
	}{
		{0, "GPT-6.1-Sol", "Latest workhorse model for coding and everyday work.", "low", true, append(five, "ultra")},
		{2, "GPT-6-Sol", "Previous generation workhorse model.", "medium", false, append(five, "ultra")},
		{3, "GPT-6-Luna", "Fast and affordable model for easier tasks.", "medium", false, five},
	} {
		e := es[c.i]
		if e.Name != c.name || e.Description != c.desc || e.DefaultEffort != c.df || e.Default != c.def || !reflect.DeepEqual(e.Efforts, c.efforts) {
			t.Errorf("model %d: %+v", c.i, e)
		}
	}
	if len(settings) == 0 || settings[0] != (event.Settings{Model: "gpt-6-sol", Choice: "gpt-6-sol", Effort: "high", EffortKnown: true, Source: "thread/start"}) {
		t.Errorf("the thread's start: %+v", settings)
	}
}

// pages are P34x's seven model/list exchanges of one model a page, as a
// script: each request and its answer.
func pages(t *testing.T) []string {
	t.Helper()
	recs, err := fixture.Read(probe(p34x))
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for i, r := range recs {
		if r.Dir != "in" || !strings.Contains(string(r.Msg), `"limit":1`) {
			continue
		}
		var req struct {
			ID json.RawMessage `json:"id"`
		}
		_ = json.Unmarshal(r.Msg, &req)
		for _, a := range recs[i+1:] { // its answer: the next line with its id
			var ans struct {
				ID     json.RawMessage `json:"id"`
				Result json.RawMessage `json:"result"`
			}
			if a.Dir == "out" && json.Unmarshal(a.Msg, &ans) == nil && string(ans.ID) == string(req.ID) && ans.Result != nil {
				in, _ := json.Marshal(map[string]json.RawMessage{"dir": json.RawMessage(`"in"`), "msg": r.Msg})
				got, _ := json.Marshal(map[string]json.RawMessage{"dir": json.RawMessage(`"out"`), "msg": a.Msg})
				out = append(out, string(in), string(got))
				break
			}
		}
	}
	if len(out) != 14 {
		t.Fatalf("%d page records, want 14", len(out))
	}
	return out
}

// TestModelChangeWire is criteria 2, 9, 13 and 17 through the real process
// plumbing, with a Codex that refuses thread/settings/update as P34x's first
// run did (plan 38, criterion 38.4): the session follows model/list's
// cursors to the last page; a change written at once is refused, said in
// the next-turn event, and every change goes with the next message, whose
// turn/start carries the model and the effort; the answer to that turn
// calls for thread/read, whose values are then in force, on the same
// thread.
func TestModelChangeWire(t *testing.T) {
	const thread = "th-34"
	recs := append([]string{handshake[0], handshake[1], handshake[2]}, pages(t)...)
	recs = append(recs,
		`{"dir":"in","msg":{"id":2,"method":"thread/start"}}`,
		`{"dir":"out","msg":{"id":2,"result":{"thread":{"id":"`+thread+`"},"model":"gpt-6-sol","reasoningEffort":"high","cwd":"<probe-work>","sandbox":{"type":"readOnly"},"approvalPolicy":"on-request"}}}`,
		handshake[5], handshake[6],
		`{"dir":"in","msg":{"id":8,"method":"thread/settings/update","params":{"threadId":"`+thread+`"}}}`,
		`{"dir":"out","msg":{"id":8,"error":{"code":-32600,"message":"thread/settings/update requires experimentalApi capability"}}}`,
		`{"dir":"in","msg":{"id":9,"method":"turn/start","params":{"threadId":"`+thread+`"}}}`,
		`{"dir":"out","msg":{"id":9,"result":{"turn":{"id":"tu-1","status":"inProgress","items":[]}}}}`,
		`{"dir":"out","msg":{"method":"turn/started","params":{"threadId":"`+thread+`","turn":{"id":"tu-1","status":"inProgress","items":[]}}}}`,
		`{"dir":"in","msg":{"id":10,"method":"thread/read","params":{"threadId":"`+thread+`"}}}`,
		`{"dir":"out","msg":{"id":10,"result":{"thread":{"id":"`+thread+`","model":"gpt-6.1-sol","reasoningEffort":"high"}}}}`,
		`{"dir":"out","msg":{"method":"turn/completed","params":{"threadId":"`+thread+`","turn":{"id":"tu-1","status":"completed","items":[]}}}}`,
	)
	r, _ := start(t, "replay", nil, "", script(t, recs...))
	listed := r.waitFor("the models", func(ev event.Event) bool { return ev.Kind == event.ModelsListed }).Payload.(event.Models)
	if len(listed.Entries) != 7 {
		t.Errorf("%d models listed, want 7", len(listed.Entries))
	}
	if r.st.Models.InForce.Model == "" { // the thread may start before the last page
		r.waitFor("the thread's settings", func(ev event.Event) bool { return ev.Kind == event.ModelSettings })
	}
	if in := r.st.Models.InForce; in.Model != "gpt-6-sol" || in.Effort != "high" {
		t.Errorf("in force at the start %+v", in)
	}
	before := len(r.written())
	if err := r.sess.SetModelSettings(provider.ModelRequest{Model: "gpt-6.1-sol", Effort: "high"}); err != nil {
		t.Fatal(err)
	}
	fell := r.waitFor("the next-turn event", func(ev event.Event) bool {
		s, ok := ev.Payload.(event.Settings)
		return ok && s.NextTurn
	}).Payload.(event.Settings)
	if fell != (event.Settings{Source: "thread/settings/update", NextTurn: true,
		Text: "Codex takes model changes with the next message: thread/settings/update refused by Codex: thread/settings/update requires experimentalApi capability"}) {
		t.Errorf("the next-turn event %+v", fell)
	}
	if !r.sess.Capabilities().ModelsNextTurn {
		t.Error("after the refusal a change does not wait for the next turn")
	}
	if n := len(r.written()); n != before+1 {
		t.Errorf("%d lines written by the change, want the refused update alone before the next message", n-before)
	}
	r.send("hello")
	answer := r.waitFor("the change's answer", func(ev event.Event) bool {
		s, ok := ev.Payload.(event.Settings)
		return ok && s.Answers
	}).Payload.(event.Settings)
	if answer.Model != "gpt-6.1-sol" || answer.Effort != "high" || answer.Source != "thread/read" {
		t.Errorf("the answer %+v", answer)
	}
	if in := r.st.Models.InForce; in.Model != "gpt-6.1-sol" || r.st.Session.Model != "gpt-6.1-sol" || r.st.Session.ID != thread {
		t.Errorf("in force %+v, session %q on %q", in, r.st.Session.Model, r.st.Session.ID)
	}
	var lists, turns, reads []string
	for _, l := range r.written() {
		var m struct {
			Method string          `json:"method"`
			Params json.RawMessage `json:"params"`
		}
		_ = json.Unmarshal([]byte(l), &m)
		switch m.Method {
		case "model/list":
			lists = append(lists, string(m.Params))
		case "turn/start":
			turns = append(turns, string(m.Params))
		case "thread/read":
			reads = append(reads, string(m.Params))
		}
	}
	wantLists := []string{`{}`, `{"cursor":"1"}`, `{"cursor":"2"}`, `{"cursor":"3"}`, `{"cursor":"4"}`, `{"cursor":"5"}`, `{"cursor":"6"}`}
	if !reflect.DeepEqual(lists, wantLists) {
		t.Errorf("model/list written %q, want %q", lists, wantLists)
	}
	if len(turns) != 1 || !strings.Contains(turns[0], `"model":"gpt-6.1-sol"`) || !strings.Contains(turns[0], `"effort":"high"`) {
		t.Errorf("turn/start written %q", turns)
	}
	if len(reads) != 1 || reads[0] != `{"threadId":"`+thread+`"}` {
		t.Errorf("thread/read written %q", reads)
	}
}

// TestTurnStartRefused is plan 34's review: a refused turn/start started no
// turn, which its error says (the state gives the message back); one that
// carried a change answers it, refused, zone B's words with it.
func TestTurnStartRefused(t *testing.T) {
	const why = "turn/start refused by Codex: invalid thread settings override"
	m := NewMapper()
	m.In([]byte(`{"id":9,"method":"turn/start","params":{"threadId":"th","model":"gpt-6.1-sol","effort":"high"}}`))
	evs := m.Out([]byte(`{"id":9,"error":{"code":-32600,"message":"invalid thread settings override"}}`))
	if len(evs) != 2 || evs[0].Payload != (event.Problem{Message: why, TurnRefused: true}) ||
		evs[1].Payload != (event.Settings{Source: "turn/start", Answers: true, Refused: true, Text: "the model change was not taken: " + why}) {
		t.Errorf("a refused turn with a change: %+v", evs)
	}
	m.In([]byte(`{"id":10,"method":"turn/start","params":{"threadId":"th"}}`))
	evs = m.Out([]byte(`{"id":10,"error":{"code":-32001,"message":"busy"}}`))
	if len(evs) != 1 || evs[0].Payload != (event.Problem{Message: "turn/start refused by Codex: busy", TurnRefused: true}) {
		t.Errorf("a refused turn: %+v", evs)
	}
}

// TestModelListGuarded is plan 34's review: a cursor that comes back ends
// the listing with the models so far and an error, the session asking for
// no further page; so does a hundredth page; a refused page before any
// list gives the selector the reason.
func TestModelListGuarded(t *testing.T) {
	page := func(id int, model, next string) []byte {
		return []byte(fmt.Sprintf(`{"id":%d,"result":{"data":[{"id":%q,"model":%q,"displayName":%q}],"nextCursor":%q}}`, id, model, model, model, next))
	}
	ask := func(m *Mapper, id int, cursor string) {
		params := `{}`
		if cursor != "" {
			params = fmt.Sprintf(`{"cursor":%q}`, cursor)
		}
		m.In([]byte(fmt.Sprintf(`{"id":%d,"method":"model/list","params":%s}`, id, params)))
	}
	listed := func(evs []event.Event) (event.Models, bool) {
		for _, e := range evs {
			if p, ok := e.Payload.(event.Models); ok {
				return p, true
			}
		}
		return event.Models{}, false
	}
	m := NewMapper()
	ask(m, 1, "")
	if _, ok := listed(m.Out(page(1, "a", "c1"))); ok || m.NextPage() != "c1" {
		t.Fatal("the first page did not ask for the next")
	}
	ask(m, 2, "c1")
	evs := m.Out(page(2, "b", "c1"))
	l, ok := listed(evs)
	if !ok || len(l.Entries) != 2 || !strings.Contains(l.Problem, `"c1" came back`) || m.NextPage() != "" {
		t.Errorf("a cursor back: %+v, next %q", evs, m.NextPage())
	}
	if p, isErr := evs[0].Payload.(event.Problem); !isErr || evs[0].Kind != event.Error || p.Message != l.Problem {
		t.Errorf("the error %+v", evs[0])
	}

	m = NewMapper()
	ask(m, 1, "")
	pages := 0
	for id := 1; ; id++ {
		pages++
		evs = m.Out(page(id, fmt.Sprintf("m%d", id), fmt.Sprintf("c%d", id)))
		next := m.NextPage()
		if next == "" {
			break
		}
		ask(m, id+1, next)
	}
	if l, ok := listed(evs); pages != 100 || !ok || len(l.Entries) != 100 || !strings.Contains(l.Problem, "more than 100 pages") {
		t.Errorf("%d pages read, list %+v", pages, l)
	}

	m = NewMapper()
	ask(m, 1, "")
	evs = m.Out([]byte(`{"id":1,"error":{"code":-32603,"message":"boom"}}`))
	if l, ok := listed(evs); !ok || len(l.Entries) != 0 || l.Problem != "model/list refused by Codex: boom" {
		t.Errorf("a refused page: %+v", evs)
	}
}
