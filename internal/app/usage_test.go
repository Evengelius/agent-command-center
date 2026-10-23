package app

import (
	"encoding/json"
	"strings"
	"testing"
	"testing/synctest"

	tea "charm.land/bubbletea/v2"

	"github.com/Evengelius/agent-command-center/internal/state"
)

// stdinKinds names each line the dashboard wrote to the CLI: its type and
// subtype, or "usage" for the adapter's own usage read.
func stdinKinds(w *wireClaude) []string {
	w.mu.Lock()
	defer w.mu.Unlock()
	var out []string
	for _, l := range w.lines {
		if isUsageRead(l) {
			out = append(out, "usage")
			continue
		}
		var m struct {
			Type    string `json:"type"`
			Request struct {
				Subtype string `json:"subtype"`
			} `json:"request"`
		}
		_ = json.Unmarshal([]byte(l), &m)
		k := m.Type
		if m.Request.Subtype != "" {
			k += "/" + m.Request.Subtype
		}
		out = append(out, k)
	}
	return out
}

// TestUsageReads (plan 24, amendment 23; P24u): the Claude adapter reads the
// usage report right after the handshake and after each turn, in order with
// the owner's messages; the quota windows, the model-scoped one included,
// show in zone A before any turn, and the reads leave no entry, no stage and
// no error.
func TestUsageReads(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := &wireClaude{}
		h := wireRig(t, w, Options{})
		h.settle()
		s := h.m.State()
		if s.Session.Phase != state.Ready || s.Turn.Phase != state.Idle {
			t.Fatalf("session %q, turn %q", s.Session.Phase, s.Turn.Phase)
		}
		for name, want := range map[string]float64{"quota.5h": 6, "quota.7d": 24, "quota.7d.fable": 11} {
			if v := s.Metrics[name]; !v.Present || v.V != want || !v.Documented || v.ResetsAt.IsZero() {
				t.Errorf("%s before any turn: %+v, want %v%% documented with a reset time", name, v, want)
			}
		}
		if len(s.Conversation) != 0 || len(s.Stages) != 0 || len(s.Errors) != 0 || s.Unrecognised != 0 {
			t.Errorf("the read left %d entries, %d stages, %d errors, %d unrecognised", len(s.Conversation), len(s.Stages), len(s.Errors), s.Unrecognised)
		}
		f := frameText(h)
		for _, want := range []string{"Session (5hr)", "Weekly (7 day)", "Fable limit", "Resets in"} {
			if !strings.Contains(f, want) {
				t.Errorf("zone A lacks %q before any turn:\n%s", want, f)
			}
		}
		if got, want := stdinKinds(w), []string{"control_request/initialize", "control_request/get_settings", "usage"}; strings.Join(got, " ") != strings.Join(want, " ") {
			t.Errorf("written at startup: %q, want %q", got, want)
		}
		// A turn: its end asks for the context usage and the report again,
		// ahead of the next prompt.
		h.send(tea.PasteMsg{Content: "hello"})
		h.send(key(tea.KeyEnter, 0))
		p := w.last()
		p.emit(`{"type":"system","subtype":"init","cwd":`+quote(h.m.State().Context.Project)+`,"session_id":"s-usage","model":"claude-haiku-5-5","permissionMode":"default"}`,
			`{"type":"result","subtype":"success","is_error":false,"duration_ms":1,"num_turns":1,"result":"ok","session_id":"s-usage"}`)
		h.settle()
		h.send(tea.PasteMsg{Content: "again"})
		h.send(key(tea.KeyEnter, 0))
		h.settle()
		// The settings reads of plan 34: after the handshake and with the
		// context usage.
		want := []string{"control_request/initialize", "control_request/get_settings", "usage", "user", "control_request/get_context_usage", "control_request/get_settings", "usage", "user"}
		if got := stdinKinds(w); strings.Join(got, " ") != strings.Join(want, " ") {
			t.Errorf("written: %q, want %q", got, want)
		}
		s = h.m.State()
		if len(s.Stages) != 2 || len(s.Errors) != 0 || s.Unrecognised != 0 {
			t.Errorf("after two prompts: %d stages, %d errors, %d unrecognised", len(s.Stages), len(s.Errors), s.Unrecognised)
		}
		if us := w.written("user"); len(us) != 2 || !strings.Contains(us[0], `"hello"`) || !strings.Contains(us[1], `"again"`) {
			t.Errorf("the owner's messages as written: %q", us)
		}
	})
}
