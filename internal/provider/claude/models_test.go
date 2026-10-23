package claude

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/Evengelius/agent-command-center/internal/event"
	"github.com/Evengelius/agent-command-center/internal/provider"
)

// Plan 34 (brief amendment 34): the session's models from the initialize
// answer, the settings in force from get_settings, the answers to
// set_model and apply_flag_settings, replayed from P34c, which sent no
// message.

// p34c is P34c's events, one slice per record.
func p34c(t *testing.T) [][]event.Event {
	t.Helper()
	_, per := replay(t, "p34c-claude-model-settings.jsonl")
	return per
}

// settingsOf is every ModelSettings payload of the replay, in order.
func settingsOf(per [][]event.Event) []event.Settings {
	var out []event.Settings
	for _, evs := range per {
		for _, e := range evs {
			if p, ok := e.Payload.(event.Settings); ok && e.Kind == event.ModelSettings {
				out = append(out, p)
			}
		}
	}
	return out
}

// TestModelsOfP34c is criterion 1: the initialize answer's 13 models, in
// the CLI's order, with its names, descriptions and levels; Haiku 4.5 takes
// no level, Opus 4.6 and Sonnet 4.6 no xhigh.
func TestModelsOfP34c(t *testing.T) {
	var listed []event.Models
	for _, evs := range p34c(t) {
		for _, e := range evs {
			if p, ok := e.Payload.(event.Models); ok && e.Kind == event.ModelsListed {
				listed = append(listed, p)
			}
		}
	}
	if len(listed) != 1 || listed[0].Source != "initialize" {
		t.Fatalf("models listed %d times (%+v), want once from initialize", len(listed), listed)
	}
	es := listed[0].Entries
	var ids []string
	for _, e := range es {
		ids = append(ids, e.ID)
	}
	want := []string{"default", "opus", "fable", "sonnet", "haiku", "claude-haiku-4-5-20251001", "claude-sonnet-5", "claude-opus-5",
		"claude-fable-5", "claude-opus-4-8", "claude-opus-4-7", "claude-opus-4-6", "claude-sonnet-4-6"}
	if !reflect.DeepEqual(ids, want) {
		t.Fatalf("models %q, want %q", ids, want)
	}
	all := []string{"low", "medium", "high", "xhigh", "max"}
	for _, c := range []struct {
		i                 int
		model, name, desc string
		efforts           []string
	}{
		{0, "claude-opus-5-5", "Default (recommended)", "Opus 5.5 · Best for everyday, complex tasks", all},
		// The run with the driver's own session variables, which P34c does
		// not keep, read "Most capable for your hardest and longest-running
		// tasks" here: the CLI's words, never the dashboard's.
		{2, "claude-fable-5-1", "Fable 5.1", "For your toughest challenges", all},
		{4, "claude-haiku-5-5", "Haiku 5.5", "Fastest for quick answers", all},
		{5, "claude-haiku-4-5-20251001", "Haiku 4.5", "Fastest for quick answers", nil},
		{11, "claude-opus-4-6", "Opus 4.6", "Best for everyday, complex tasks", []string{"low", "medium", "high", "max"}},
	} {
		e := es[c.i]
		if e.Model != c.model || e.Name != c.name || e.Description != c.desc || !reflect.DeepEqual(e.Efforts, c.efforts) || e.Default || e.DefaultEffort != "" {
			t.Errorf("model %d: %+v", c.i, e)
		}
	}
}

// TestSettingsOfP34c is criteria 3, 7 and 12 on the wire: the first read
// gives the owner's settings in force (Fable 5.1, Max, Ultracode on and
// available), each set_model answered is followed by the read that says
// what is in force, with the entry that selects it, and the unknown model
// is refused with the CLI's words. Every read is undocumented.
func TestSettingsOfP34c(t *testing.T) {
	ss := settingsOf(p34c(t))
	first := ss[0]
	want := event.Settings{Model: "claude-fable-5-1", Choice: "fable", Effort: "max", EffortKnown: true, Ultracode: true,
		UltracodeAvailable: true, UltracodeKnown: true, Source: "get_settings", Undocumented: true}
	if first != want {
		t.Errorf("first read %+v\n want %+v", first, want)
	}
	var choices []string
	refused := 0
	for _, s := range ss {
		switch {
		case s.Refused:
			refused++
			if !s.Answers || s.Text != "set_model refused by Claude Code: Model 'claude-not-a-model-34' not found" {
				t.Errorf("refusal %+v", s)
			}
		case s.Answers:
			choices = append(choices, s.Choice)
		}
		if s.Source == "get_settings" && !s.Undocumented {
			t.Errorf("a read not marked undocumented: %+v", s)
		}
	}
	if refused != 1 {
		t.Errorf("%d refusals, want 1", refused)
	}
	// The reads after set_model sonnet, the flag settings (low, ultracode,
	// max, ultracode off), claude-opus-4-6 with xhigh, haiku, fable, the
	// extreme level, default with the levels cleared.
	wantChoices := []string{"sonnet", "sonnet", "sonnet", "sonnet", "sonnet", "claude-opus-4-6", "haiku", "fable", "fable", "default"}
	if !reflect.DeepEqual(choices, wantChoices) {
		t.Errorf("answered choices %q, want %q", choices, wantChoices)
	}
}

// TestSettingsLineKept is design D1: a get_settings answer becomes an event
// whose line keeps the values in force and the settings' model, nothing
// else of the owner's settings.
func TestSettingsLineKept(t *testing.T) {
	m := NewMapper()
	m.In([]byte(`{"type":"control_request","request_id":"read_1","request":{"subtype":"get_settings"}}`))
	evs := m.Out([]byte(`{"type":"control_response","response":{"subtype":"success","request_id":"read_1","response":{` +
		`"effective":{"model":"fable","env":{"HTTPS_PROXY":"http://127.0.0.1:3128","SECRET_TOKEN":"abc123"},"hooks":{"x":1}},` +
		`"sources":[{"source":"userSettings","settings":{"env":{"SECRET_TOKEN":"abc123"}}}],` +
		`"applied":{"model":"claude-fable-5-1","effort":"xhigh","ultracode":false,"ultracodeRequested":false,"ultracodeAvailable":true}}}}`))
	if len(evs) != 1 || evs[0].Kind != event.ModelSettings {
		t.Fatalf("events %+v", evs)
	}
	for _, leak := range []string{"SECRET_TOKEN", "abc123", "HTTPS_PROXY", "sources", "hooks", "env"} {
		if strings.Contains(evs[0].Raw, leak) {
			t.Errorf("the event's line keeps %q: %s", leak, evs[0].Raw)
		}
	}
	var kept struct {
		Response struct {
			Response struct {
				Applied   map[string]any    `json:"applied"`
				Effective map[string]string `json:"effective"`
			} `json:"response"`
		} `json:"response"`
	}
	if err := json.Unmarshal([]byte(evs[0].Raw), &kept); err != nil || kept.Response.Response.Applied["model"] != "claude-fable-5-1" ||
		kept.Response.Response.Effective["model"] != "fable" {
		t.Errorf("kept line %s (%v)", evs[0].Raw, err)
	}
	if s := evs[0].Payload.(event.Settings); s.EffortLock != "" || s.Effort != "xhigh" || s.Ultracode || !s.UltracodeAvailable {
		t.Errorf("settings %+v", s)
	}
}

// TestEffortLock is design D5: CLAUDE_CODE_EFFORT_LEVEL, in a settings env
// block get_settings reports or in the CLI's environment, holds the effort.
func TestEffortLock(t *testing.T) {
	read := func(m *Mapper, env string) event.Settings {
		m.In([]byte(`{"type":"control_request","request_id":"read_1","request":{"subtype":"get_settings"}}`))
		evs := m.Out([]byte(`{"type":"control_response","response":{"subtype":"success","request_id":"read_1","response":{` +
			`"effective":{"env":` + env + `},"applied":{"model":"claude-fable-5-1","effort":"max"}}}}`))
		return evs[0].Payload.(event.Settings)
	}
	if s := read(NewMapper(), `{"CLAUDE_CODE_EFFORT_LEVEL":"max"}`); s.EffortLock != EffortVar || s.Effort != "max" {
		t.Errorf("in the settings' env: %+v", s)
	}
	if s := read(NewMapper(), `{"CLAUDE_CODE_EFFORT_LEVEL":""}`); s.EffortLock != "" {
		t.Errorf("empty in the settings' env: %+v", s)
	}
	m := NewMapper()
	m.SetEffortVar("high")
	if s := read(m, `{}`); s.EffortLock != EffortVar {
		t.Errorf("in the CLI's environment: %+v", s)
	}
}

// TestModelSettingsWire is criteria 7, 10, 12 and 17 through the real
// process plumbing and a CLI that answers as P34c and P34c3 did: the
// handshake's read gives what is in force; an effort goes as
// apply_flag_settings with the Ultracode in force, which a level alone
// would turn off; Ultracode alone keeps the effort; a model goes as
// set_model; an unknown one is refused and the state keeps what was in
// force; every answer is followed by a read; the process is never
// restarted.
func TestModelSettingsWire(t *testing.T) {
	log := filepath.Join(t.TempDir(), "in.log")
	t.Setenv("ACC_FAKE_LOG", log)
	r, _ := start(t, "models", nil)
	answered := func(what string) event.Settings {
		t.Helper()
		var got event.Settings
		if !r.until(func(ev event.Event) bool {
			s, ok := ev.Payload.(event.Settings)
			got = s
			return ok && (s.Answers || what == "first" && s.Source == "get_settings")
		}, 10*time.Second) {
			t.Fatalf("%s: no settings read", what)
		}
		return got
	}
	if s := answered("first"); s.Choice != "fable" || s.Effort != "max" || !s.Ultracode || !s.UltracodeAvailable {
		t.Fatalf("the handshake's read %+v", s)
	}
	in := func() event.Settings { return r.st.Models.InForce }
	if err := r.sess.SetModelSettings(provider.ModelRequest{Effort: "low"}); err != nil {
		t.Fatal(err)
	}
	if s := answered("effort"); s.Effort != "low" || !s.Ultracode || in().Effort != "low" || !in().Ultracode {
		t.Errorf("after the effort: %+v, in force %+v", s, in())
	}
	off := false
	if err := r.sess.SetModelSettings(provider.ModelRequest{Ultracode: &off}); err != nil {
		t.Fatal(err)
	}
	if s := answered("ultracode"); s.Ultracode || s.Effort != "low" {
		t.Errorf("after Ultracode off: %+v", s)
	}
	if err := r.sess.SetModelSettings(provider.ModelRequest{Model: "claude-opus-4-6"}); err != nil {
		t.Fatal(err)
	}
	if s := answered("model"); s.Model != "claude-opus-4-6" || s.Choice != "claude-opus-4-6" || s.UltracodeAvailable {
		t.Errorf("after the model: %+v", s)
	}
	if r.st.Session.Model != "claude-opus-4-6" {
		t.Errorf("zone A's and zone B's model %q", r.st.Session.Model)
	}
	if err := r.sess.SetModelSettings(provider.ModelRequest{Model: "claude-nope"}); err != nil {
		t.Fatal(err)
	}
	if s := answered("refused"); !s.Refused || s.Text != "set_model refused by Claude Code: Model 'claude-nope' not found" {
		t.Errorf("the refusal %+v", s)
	}
	if in().Model != "claude-opus-4-6" || r.st.Session.Model != "claude-opus-4-6" {
		t.Errorf("after the refusal: in force %+v, session's model %q", in(), r.st.Session.Model)
	}
	notice := false
	for _, e := range r.st.Conversation {
		notice = notice || e.Role == "notice" && strings.Contains(e.Text, "Model 'claude-nope' not found")
	}
	if !notice {
		t.Error("the refusal is not in zone B")
	}
	r.until(func(event.Event) bool { return false }, 200*time.Millisecond) // the read after the refusal
	b, err := os.ReadFile(log)
	if err != nil {
		t.Fatal(err)
	}
	var kinds []string
	for _, l := range strings.Split(strings.TrimSpace(string(b)), "\n") {
		var m struct {
			Type    string          `json:"type"`
			Request json.RawMessage `json:"request"`
		}
		_ = json.Unmarshal([]byte(l), &m)
		if m.Type == "control_request" {
			kinds = append(kinds, string(m.Request))
		}
	}
	want := []string{
		`{"subtype":"initialize","hooks":null}`,
		`{"subtype":"get_settings"}`,
		`{"subtype":"apply_flag_settings","settings":{"effortLevel":"low","ultracode":true}}`,
		`{"subtype":"get_settings"}`,
		`{"subtype":"apply_flag_settings","settings":{"ultracode":false}}`,
		`{"subtype":"get_settings"}`,
		`{"subtype":"set_model","model":"claude-opus-4-6"}`,
		`{"subtype":"get_settings"}`,
		`{"subtype":"set_model","model":"claude-nope"}`,
		`{"subtype":"get_settings"}`,
	}
	if !reflect.DeepEqual(kinds, want) {
		t.Errorf("control requests written:\n%s\nwant\n%s", strings.Join(kinds, "\n"), strings.Join(want, "\n"))
	}
	for _, ev := range r.evs {
		if ev.Kind == event.SessionSpawned && ev.Seq != 1 {
			t.Error("the CLI was started again")
		}
	}
}

// TestChangeAnswered is design D3's order: a change's acceptance asks for a
// read, and the read written for it answers the change; a read written
// before (after a turn, say) does not.
func TestChangeAnswered(t *testing.T) {
	m := NewMapper()
	m.In([]byte(`{"type":"control_request","request_id":"read_1","request":{"subtype":"get_settings"}}`))
	m.In([]byte(`{"type":"control_request","request_id":"req_2","request":{"subtype":"apply_flag_settings","settings":{"effortLevel":"low","ultracode":true}}}`))
	if m.ReadDue() {
		t.Fatal("a read due before the change's answer")
	}
	read := func(id string) event.Settings {
		evs := m.Out([]byte(`{"type":"control_response","response":{"subtype":"success","request_id":"` + id + `","response":{"applied":{"model":"claude-fable-5-1","effort":"low","ultracode":true,"ultracodeAvailable":true}}}}`))
		return evs[0].Payload.(event.Settings)
	}
	if s := read("read_1"); s.Answers {
		t.Errorf("the earlier read answers the change: %+v", s)
	}
	evs := m.Out([]byte(`{"type":"control_response","response":{"subtype":"success","request_id":"req_2","response":{}}}`))
	if len(evs) != 1 || evs[0].Kind != event.StatusChanged || !m.ReadDue() {
		t.Fatalf("acceptance: %+v, read due %v", evs, m.ReadDue())
	}
	m.In([]byte(`{"type":"control_request","request_id":"read_2","request":{"subtype":"get_settings"}}`))
	if m.ReadDue() {
		t.Error("the read written still due")
	}
	if s := read("read_2"); !s.Answers || s.Effort != "low" || !s.Ultracode {
		t.Errorf("the change's read %+v", s)
	}
	if on, known := m.UltracodeInForce(); !on || !known {
		t.Errorf("Ultracode in force %v, known %v", on, known)
	}
}
