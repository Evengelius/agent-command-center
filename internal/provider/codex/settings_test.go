package codex

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/Evengelius/agent-command-center/internal/event"
	"github.com/Evengelius/agent-command-center/internal/fixture"
	"github.com/Evengelius/agent-command-center/internal/provider"
)

// Plan 38 (brief amendment 39): a change written at once with
// thread/settings/update, Codex's experimental API, answered by the
// thread/settings/updated that follows (P38x); one asked during a turn
// written at its end; with a Codex that refuses the update, every change
// with the next turn/start (models_test.go, TestModelChangeWire);
// currentTime/read answered with the time.

const t38 = "th-38"

// p38Updated is P38x's recorded thread/settings/updated with that model and
// effort, its thread made thread, as a recorded output.
func p38Updated(t *testing.T, thread, model, effort string) string {
	t.Helper()
	recs, err := fixture.Read(probe("p38x-codex-settings.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range recs {
		var m struct {
			Method string
			Params struct {
				ThreadID string                         `json:"threadId"`
				Settings struct{ Model, Effort string } `json:"threadSettings"`
			}
		}
		if r.Dir == "out" && json.Unmarshal(r.Msg, &m) == nil && m.Method == "thread/settings/updated" &&
			m.Params.Settings.Model == model && m.Params.Settings.Effort == effort {
			return `{"dir":"out","msg":` + strings.ReplaceAll(string(r.Msg), m.Params.ThreadID, thread) + `}`
		}
	}
	t.Fatalf("P38x has no thread/settings/updated with %s at %s", model, effort)
	return ""
}

// start38 starts a session whose thread runs gpt-6-sol at "xhigh", as
// P38x's, then plays more.
func start38(t *testing.T, p *Provider, more ...string) *run {
	t.Helper()
	recs := append([]string{handshake[0], handshake[1], handshake[2],
		`{"dir":"in","msg":{"id":2,"method":"thread/start"}}`,
		`{"dir":"out","msg":{"id":2,"result":{"thread":{"id":"` + t38 + `"},"model":"gpt-6-sol","reasoningEffort":"xhigh","cwd":"<probe-work>","sandbox":{"type":"readOnly"},"approvalPolicy":"on-request"}}}`,
		handshake[5], handshake[6]}, more...)
	r, _ := start(t, "replay", p, "", script(t, recs...))
	r.waitFor("the thread's settings", func(ev event.Event) bool {
		s, ok := ev.Payload.(event.Settings)
		return ok && s.Source == "thread/start"
	})
	r.limits()
	return r
}

// update38 is a recorded thread/settings/update on t38 and what follows it.
func update38(out ...string) []string {
	return append([]string{`{"dir":"in","msg":{"id":7,"method":"thread/settings/update","params":{"threadId":"` + t38 + `"}}}`}, out...)
}

// updates are the thread/settings/update params written, in order.
func updates(r *run) []string {
	var out []string
	for _, l := range r.written() {
		var m struct {
			Method string          `json:"method"`
			Params json.RawMessage `json:"params"`
		}
		if json.Unmarshal([]byte(l), &m) == nil && m.Method == "thread/settings/update" {
			out = append(out, string(m.Params))
		}
	}
	return out
}

// answer waits for the change's answer; a settings event before it fails.
func (r *run) answer() event.Settings {
	r.t.Helper()
	return r.waitFor("the change's answer", func(ev event.Event) bool {
		s, ok := ev.Payload.(event.Settings)
		if ok && !s.Answers {
			r.t.Errorf("a settings event before the answer: %+v", s)
		}
		return ok && s.Answers
	}).Payload.(event.Settings)
}

// TestSettingsUpdateWire is criterion 38.2: initialize asks for the
// experimental API; between turns a change writes thread/settings/update at
// once with what it asks for alone, and no turn/start; the {} that answers
// it gives no settings event; P38x's notification that follows is the
// answer and in force.
func TestSettingsUpdateWire(t *testing.T) {
	r := start38(t, nil, update38(`{"dir":"out","msg":{"id":7,"result":{}}}`, p38Updated(t, t38, "gpt-6-sol", "low"))...)
	if l := r.written()[0]; l != initLine {
		t.Errorf("initialize\n got %s\nwant %s", l, initLine)
	}
	if r.sess.Capabilities().ModelsNextTurn {
		t.Error("between turns a change waits for the next turn")
	}
	if err := r.sess.SetModelSettings(provider.ModelRequest{Effort: "low"}); err != nil {
		t.Fatal(err)
	}
	got := r.answer()
	want := event.Settings{Model: "gpt-6-sol", Choice: "gpt-6-sol", Effort: "low", EffortKnown: true, Source: "thread/settings/updated", Answers: true}
	if got != want {
		t.Errorf("the answer\n got %+v\nwant %+v", got, want)
	}
	acked := false
	for _, ev := range r.evs {
		if s, ok := ev.Payload.(event.Status); ok && s.Status == "settings update acknowledged" {
			acked = true
		}
	}
	if !acked {
		t.Error("the update's {} gave no status")
	}
	if u := updates(r); len(u) != 1 || u[0] != `{"threadId":"`+t38+`","effort":"low"}` {
		t.Errorf("thread/settings/update written %q", u)
	}
	for _, l := range r.written() {
		if strings.Contains(l, `"method":"turn/start"`) {
			t.Errorf("a turn/start written: %s", l)
		}
	}
	if in := r.st.Models.InForce; in.Effort != "low" || in.Source != "thread/settings/updated" {
		t.Errorf("in force %+v", in)
	}
	if r.sess.Capabilities().ModelsNextTurn {
		t.Error("after the answer a change waits for the next turn")
	}
}

// TestSettingsDuringTurn is criterion 38.3: from the main turn/start to its
// turn/completed a change waits for the next turn; a change asked then
// writes nothing until the turn ends, then one thread/settings/update with
// the whole change, answered by its notification.
func TestSettingsDuringTurn(t *testing.T) {
	recs := []string{
		`{"dir":"in","msg":{"id":8,"method":"turn/start","params":{"threadId":"` + t38 + `"}}}`,
		`{"dir":"out","msg":{"id":8,"result":{"turn":{"id":"tu-38","status":"inProgress","items":[]}}}}`,
		`{"dir":"out","msg":{"method":"turn/started","params":{"threadId":"` + t38 + `","turn":{"id":"tu-38","status":"inProgress","items":[]}}}}`,
		`{"dir":"in","msg":{"id":9,"method":"turn/interrupt","params":{"threadId":"` + t38 + `"}}}`,
		`{"dir":"out","msg":{"id":9,"result":{}}}`,
		`{"dir":"out","msg":{"method":"turn/completed","params":{"threadId":"` + t38 + `","turn":{"id":"tu-38","status":"interrupted","items":[]}}}}`,
	}
	recs = append(recs, update38(`{"dir":"out","msg":{"id":7,"result":{}}}`, p38Updated(t, t38, "gpt-6-luna", "low"))...)
	r := start38(t, nil, recs...)
	r.send("hello")
	if !r.sess.Capabilities().ModelsNextTurn {
		t.Error("with the turn/start written, a change does not wait for the turn")
	}
	r.wait(event.TurnStarted)
	if err := r.sess.SetModelSettings(provider.ModelRequest{Model: "gpt-6-luna", Effort: "low"}); err != nil {
		t.Fatal(err)
	}
	if !r.sess.Capabilities().ModelsNextTurn {
		t.Error("during the turn, a change does not wait for it")
	}
	if u := updates(r); len(u) != 0 {
		t.Errorf("thread/settings/update written during the turn: %q", u)
	}
	if err := r.sess.Interrupt(); err != nil {
		t.Fatal(err)
	}
	got := r.answer()
	if got.Model != "gpt-6-luna" || got.Effort != "low" || got.Source != "thread/settings/updated" {
		t.Errorf("the answer %+v", got)
	}
	lines := r.written()
	at, interrupt := -1, -1
	for i, l := range lines {
		switch {
		case strings.Contains(l, `"method":"thread/settings/update"`):
			at = i
		case strings.Contains(l, `"method":"turn/interrupt"`):
			interrupt = i
		}
	}
	if u := updates(r); len(u) != 1 || u[0] != `{"threadId":"`+t38+`","model":"gpt-6-luna","effort":"low"}` || at < interrupt {
		t.Errorf("thread/settings/update written %q, line %d, the interrupt's %d", u, at, interrupt)
	}
	if r.sess.Capabilities().ModelsNextTurn {
		t.Error("after the turn a change waits for the next turn")
	}
}

// TestSettingsBackstop is criterion 38.5: an update answered {} with no
// notification goes with the next turn/start, whose thread/read answers it.
func TestSettingsBackstop(t *testing.T) {
	recs := update38(`{"dir":"out","msg":{"id":7,"result":{}}}`)
	recs = append(recs,
		`{"dir":"in","msg":{"id":8,"method":"turn/start","params":{"threadId":"`+t38+`"}}}`,
		`{"dir":"out","msg":{"id":8,"result":{"turn":{"id":"tu-38","status":"inProgress","items":[]}}}}`,
		`{"dir":"in","msg":{"id":10,"method":"thread/read","params":{"threadId":"`+t38+`"}}}`,
		`{"dir":"out","msg":{"id":10,"result":{"thread":{"id":"`+t38+`","model":"gpt-6-sol","reasoningEffort":"low"}}}}`,
		`{"dir":"out","msg":{"method":"turn/completed","params":{"threadId":"`+t38+`","turn":{"id":"tu-38","status":"completed","items":[]}}}}`,
	)
	r := start38(t, nil, recs...)
	if err := r.sess.SetModelSettings(provider.ModelRequest{Effort: "low"}); err != nil {
		t.Fatal(err)
	}
	r.waitFor("the update's {}", func(ev event.Event) bool {
		s, ok := ev.Payload.(event.Status)
		return ok && s.Status == "settings update acknowledged"
	})
	r.send("hello")
	got := r.answer()
	if got.Effort != "low" || got.Source != "thread/read" {
		t.Errorf("the answer %+v", got)
	}
	var turn string
	for _, l := range r.written() {
		if strings.Contains(l, `"method":"turn/start"`) {
			turn = l
		}
	}
	if !strings.Contains(turn, `"effort":"low"`) {
		t.Errorf("the turn/start does not carry the change: %s", turn)
	}
}

// TestSettingsRefused is criterion 38.4's other refusal through the
// session: a -32602 refuses the change, which the next turn/start does not
// carry, and changes are still taken at once.
func TestSettingsRefused(t *testing.T) {
	recs := update38(`{"dir":"out","msg":{"id":7,"error":{"code":-32602,"message":"invalid params"}}}`)
	recs = append(recs,
		`{"dir":"in","msg":{"id":8,"method":"turn/start","params":{"threadId":"`+t38+`"}}}`,
		`{"dir":"out","msg":{"id":8,"result":{"turn":{"id":"tu-38","status":"inProgress","items":[]}}}}`,
	)
	r := start38(t, nil, recs...)
	if err := r.sess.SetModelSettings(provider.ModelRequest{Effort: "low"}); err != nil {
		t.Fatal(err)
	}
	got := r.answer()
	if !got.Refused || got.Text != "the model change was not taken: thread/settings/update refused by Codex: invalid params" {
		t.Errorf("the answer %+v", got)
	}
	if r.sess.Capabilities().ModelsNextTurn {
		t.Error("after a refused change, changes wait for the next turn")
	}
	r.send("hello")
	r.waitFor("the turn accepted", func(ev event.Event) bool {
		s, ok := ev.Payload.(event.Status)
		return ok && s.Status == "turn accepted"
	})
	for _, l := range r.written() {
		if strings.Contains(l, `"method":"turn/start"`) && (strings.Contains(l, `"effort"`) || strings.Contains(l, `"model"`)) {
			t.Errorf("the refused change carried: %s", l)
		}
	}
}

// TestCurrentTime is criterion 38.6: currentTime/read is answered with the
// session clock's whole Unix seconds and its id as received; the request is
// a status, the reply a write, neither unrecognised nor an approval.
func TestCurrentTime(t *testing.T) {
	at := time.Unix(1792829210, 600_000_000)
	p := &Provider{Now: func() time.Time { return at }}
	r := start38(t, p, `{"dir":"in","msg":{"id":20,"method":"turn/start","params":{"threadId":"`+t38+`"}}}`,
		`{"dir":"out","msg":{"id":0,"method":"currentTime/read","params":{"threadId":"`+t38+`"}}}`)
	r.send("hello")
	r.waitFor("the reply", func(ev event.Event) bool {
		w, ok := ev.Payload.(event.Write)
		return ok && w.What == "currentTime/read reply"
	})
	// The fake logs a line once it reads it, which may come after the event.
	var reply string
	for wait := time.Now().Add(5 * time.Second); reply == "" && time.Now().Before(wait); time.Sleep(20 * time.Millisecond) {
		for _, l := range r.written() {
			if strings.Contains(l, `"currentTimeAt"`) {
				reply = l
			}
		}
	}
	if reply != `{"id":0,"result":{"currentTimeAt":1792829210}}` {
		t.Errorf("the reply %s", reply)
	}
	asked := false
	for _, ev := range r.evs {
		if s, ok := ev.Payload.(event.Status); ok && s.Status == "current time asked" {
			asked = true
		}
		if ev.Kind == event.Unrecognised || ev.Kind == event.ApprovalResolved || ev.Kind == event.ApprovalRequest {
			t.Errorf("%s: %s", ev.Kind, ev.Raw)
		}
	}
	if !asked {
		t.Error("the request gave no status")
	}
}

// TestUpdateRefusals is criterion 38.4 on the mapper: a refusal that says
// Codex does not take the method falls back to the next turn, Codex's
// words in the text; any other refuses the change.
func TestUpdateRefusals(t *testing.T) {
	const fall = "Codex takes model changes with the next message: thread/settings/update refused by Codex: "
	for _, c := range []struct{ code, msg string }{
		{"-32600", "thread/settings/update requires experimentalApi capability"}, // P34x
		{"-32601", "Method not found"},
		{"-32600", "Invalid request: unknown variant `thread/settings/update`, expected one of `initialize`"},
	} {
		m := NewMapper()
		m.In([]byte(`{"id":7,"method":"thread/settings/update","params":{"threadId":"th","effort":"low"}}`))
		evs := m.Out([]byte(`{"id":7,"error":{"code":` + c.code + `,"message":` + quote(c.msg) + `}}`))
		if len(evs) != 1 || evs[0].Payload != (event.Settings{Source: "thread/settings/update", NextTurn: true, Text: fall + c.msg}) {
			t.Errorf("%s %q: %+v", c.code, c.msg, evs)
		}
	}
	m := NewMapper()
	m.In([]byte(`{"id":7,"method":"thread/settings/update","params":{"threadId":"th","effort":"low"}}`))
	evs := m.Out([]byte(`{"id":7,"error":{"code":-32602,"message":"invalid params"}}`))
	why := "thread/settings/update refused by Codex: invalid params"
	if len(evs) != 2 || evs[0].Payload != (event.Problem{Message: why}) ||
		evs[1].Payload != (event.Settings{Source: "thread/settings/update", Answers: true, Refused: true, Text: "the model change was not taken: " + why}) {
		t.Errorf("-32602: %+v", evs)
	}
	// Neither a fallback nor a refusal names another method's words.
	if unsupported(json.RawMessage(`{"code":-32600,"message":"unknown variant `+"`thread/fork`"+`"}`), "thread/settings/update") {
		t.Error("another method's unknown variant taken for this one")
	}
}

// TestSettingsUpdatedAnswers is criterion 38.2 on the mapper: a
// notification answers only a change waiting; one with none, from a
// turn/start that changed another setting (P38t), only reports; another
// thread's reports nothing; a read after the notification answered no
// longer answers.
func TestSettingsUpdatedAnswers(t *testing.T) {
	settings := func(evs []event.Event) []event.Settings {
		var out []event.Settings
		for _, e := range evs {
			if s, ok := e.Payload.(event.Settings); ok {
				out = append(out, s)
			}
		}
		return out
	}
	m := NewMapper()
	m.Out([]byte(`{"method":"thread/started","params":{"thread":{"id":"th"}}}`))
	updated := func(thread, effort string) []byte {
		return []byte(`{"method":"thread/settings/updated","params":{"threadId":"` + thread + `","threadSettings":{"model":"gpt-6-sol","effort":"` + effort + `"}}}`)
	}
	if s := settings(m.Out(updated("th", "xhigh"))); len(s) != 1 || s[0].Answers || s[0].Effort != "xhigh" {
		t.Errorf("with no change waiting: %+v", s)
	}
	m.In([]byte(`{"id":7,"method":"thread/settings/update","params":{"threadId":"th","effort":"low"}}`))
	if evs := m.Out([]byte(`{"id":7,"result":{}}`)); len(settings(evs)) != 0 || len(evs) != 1 {
		t.Errorf("the {}: %+v", evs)
	}
	if s := settings(m.Out(updated("th-child", "low"))); len(s) != 0 {
		t.Errorf("another thread's: %+v", s)
	}
	if s := settings(m.Out(updated("th", "low"))); len(s) != 1 || !s[0].Answers || s[0].Effort != "low" {
		t.Errorf("the answer: %+v", s)
	}
	if s := settings(m.Out(updated("th", "low"))); len(s) != 1 || s[0].Answers {
		t.Errorf("a second notification answers: %+v", s)
	}
	// A turn/start that carried a change, its notification first: the read
	// that follows only reports.
	m.In([]byte(`{"id":8,"method":"turn/start","params":{"threadId":"th","effort":"medium"}}`))
	m.Out([]byte(`{"id":8,"result":{"turn":{"id":"tu"}}}`))
	if s := settings(m.Out(updated("th", "medium"))); len(s) != 1 || !s[0].Answers {
		t.Errorf("the turn's notification: %+v", s)
	}
	if !m.TakeReadDue() {
		t.Fatal("no read due after the turn that carried a change")
	}
	m.In([]byte(`{"id":9,"method":"thread/read","params":{"threadId":"th"}}`))
	if s := settings(m.Out([]byte(`{"id":9,"result":{"thread":{"id":"th","model":"gpt-6-sol","reasoningEffort":"medium"}}}`))); len(s) != 1 || s[0].Answers {
		t.Errorf("the read after the notification: %+v", s)
	}
}

// TestP38Replay is criterion 38.9's recordings on the mapper: P38x's and
// P38t's settings in order, each update answered by its notification but
// the one to the effort in force, which Codex did not notify, and the
// turn/start's own notification only a report.
func TestP38Replay(t *testing.T) {
	type got struct {
		model, effort, source string
		answers               bool
	}
	want := []got{
		{"gpt-6-sol", "xhigh", "thread/start", false},
		{"gpt-6-sol", "low", "thread/settings/updated", true},
		{"gpt-6-sol", "low", "thread/read", false},
		{"gpt-6.1-sol", "low", "thread/settings/updated", true},
		{"gpt-6-luna", "low", "thread/settings/updated", true},
		{"gpt-6-luna", "low", "thread/read", false},
	}
	for _, name := range []string{"p38x-codex-settings.jsonl", "p38t-codex-turn-settings.jsonl"} {
		_, per := replay(t, name)
		var gots []got
		for _, e := range fixture.Flatten(per) {
			if s, ok := e.Payload.(event.Settings); ok {
				gots = append(gots, got{s.Model, s.Effort, s.Source, s.Answers})
			}
		}
		w := want
		if name == "p38t-codex-turn-settings.jsonl" {
			w = append(append([]got(nil), want...), got{"gpt-6-luna", "low", "thread/settings/updated", false}, got{"gpt-6-luna", "low", "thread/read", false})
		}
		if len(gots) != len(w) {
			t.Fatalf("%s: settings %+v\nwant %+v", name, gots, w)
		}
		for i := range w {
			if gots[i] != w[i] {
				t.Errorf("%s: settings %d %+v, want %+v", name, i, gots[i], w[i])
			}
		}
	}
}
