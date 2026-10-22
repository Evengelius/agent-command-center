package claude

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Evengelius/agent-command-center/internal/event"
	"github.com/Evengelius/agent-command-center/internal/fixture"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// Plan 24, amendment 23 (probe P24u): the usage report. "/usage" written
// by the dashboard is answered by Claude Code itself, with no model call,
// as a turn of its own; the mapper takes the report's windows and keeps
// the exchange out of the conversation. The same answer to a "/usage" the
// owner typed is shown as any message.

// p24u plays the recording through the mapper: the dashboard's lines by In,
// except "/usage", which goes by in; Claude Code's by Out.
func p24u(t *testing.T, m *Mapper, in func([]byte) []event.Event) []event.Event {
	t.Helper()
	recs, err := fixture.Read(filepath.Join("..", "..", "..", "testdata", "probes", "p24u-claude-usage-report.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	var all []event.Event
	reads := 0
	for _, r := range recs {
		switch r.Dir {
		case "in":
			if strings.Contains(string(r.Msg), `"/usage"`) {
				reads++
				all = append(all, in(r.Msg)...)
			} else {
				all = append(all, m.In(r.Msg)...)
			}
		case "out":
			all = append(all, m.Out(r.Msg)...)
		}
	}
	if reads != 1 {
		t.Fatalf("%d usage reads in P24u, want 1", reads)
	}
	return all
}

const usageLine = `{"type":"user","message":{"role":"user","content":"/usage"},"parent_tool_use_id":null,"session_id":"default"}`
const runningLine = `{"type":"system","subtype":"session_state_changed","state":"running","session_id":"4d551235-2308-4a95-a44a-9cc515010729"}`

func kinds(evs []event.Event) string {
	var ks []string
	for _, ev := range evs {
		ks = append(ks, string(ev.Kind))
	}
	return strings.Join(ks, " ")
}

func has(evs []event.Event, k event.Kind) bool {
	for _, ev := range evs {
		if ev.Kind == k {
			return true
		}
	}
	return false
}

// TestUsageReadInternal: the dashboard's own read gives the windows and
// nothing else: no turn, no message, no result's figures; the reducer keeps
// the conversation and the stages as they were.
func TestUsageReadInternal(t *testing.T) {
	m := NewMapper()
	all := p24u(t, m, m.InInternal)
	sent := 0 // the usage read's own Sent event, after the handshake's
	for _, ev := range all {
		if w, ok := ev.Payload.(event.Write); ok && ev.Kind == event.Sent && w.What == "usage" {
			sent++
		}
	}
	if sent != 1 || !has(all, event.SessionReady) || !has(all, event.SessionStarted) {
		t.Fatalf("%d usage writes, ready %v, started %v, from: %s", sent, has(all, event.SessionReady), has(all, event.SessionStarted), kinds(all))
	}
	for _, k := range []event.Kind{event.TurnStarted, event.TurnCompleted, event.MessageCompleted, event.MessageDelta, event.UserMessage, event.Unrecognised, event.Error} {
		for _, ev := range all {
			if ev.Kind == k {
				t.Errorf("a %s event from the read: %.120s", k, ev.Raw)
			}
		}
	}
	var tel *event.Metrics
	for _, ev := range all {
		if p, ok := ev.Payload.(event.Metrics); ok && ev.Kind == event.Telemetry {
			tel = &p
		}
	}
	if tel == nil {
		t.Fatalf("no telemetry from: %s", kinds(all))
	}
	want := map[string]struct {
		pct   float64
		reset time.Time
		kind  string
	}{
		"quota.5h":       {6, time.Date(2026, 10, 22, 15, 20, 0, 473808000, time.UTC), "session"},
		"quota.7d":       {24, time.Date(2026, 10, 27, 9, 0, 0, 473852000, time.UTC), "weekly_all"},
		"quota.7d.fable": {11, time.Date(2026, 10, 27, 9, 0, 0, 474012000, time.UTC), "weekly_scoped"},
	}
	if len(tel.Items) != len(want) {
		t.Fatalf("%d metrics, want %d: %+v", len(tel.Items), len(want), tel.Items)
	}
	for _, it := range tel.Items {
		w, ok := want[it.Name]
		if !ok || it.Value != w.pct || it.Unit != "%" || !it.ResetsAt.Equal(w.reset) || !it.Documented || it.Source != "usage_report ("+w.kind+")" {
			t.Errorf("metric %+v, want %+v", it, w)
		}
	}
	// The reducer: the windows in the state, no entry, no stage, the turn idle.
	s := state.New()
	s.Context.Provider = ProviderID
	s.Do(state.ValidationOK{Provider: ProviderID, ProviderName: "Claude Code", Project: "<probe-work>"})
	s.Apply(event.Event{Seq: 1, Kind: event.SessionSpawned, Origin: event.Observed, Payload: event.Spawned{}})
	for i, ev := range all {
		ev.Seq = uint64(i + 2)
		s.Apply(ev)
	}
	if v := s.Metrics["quota.7d.fable"]; !v.Present || v.V != 11 || !v.Documented || v.ResetsAt != want["quota.7d.fable"].reset {
		t.Errorf("quota.7d.fable in the state: %+v", v)
	}
	if len(s.Conversation) != 0 || len(s.Stages) != 0 || s.Turn.Phase != state.Idle || s.Unrecognised != 0 || len(s.Errors) != 0 || s.Session.Phase != state.Ready {
		t.Errorf("after the read: %d entries, %d stages, turn %s, %d unrecognised, %d errors, session %s", len(s.Conversation), len(s.Stages), s.Turn.Phase, s.Unrecognised, len(s.Errors), s.Session.Phase)
	}
	if s.Session.Model == "" {
		t.Error("the read's system/init gave no model")
	}
	// The next turn is the owner's again: its start maps as before.
	if evs := m.Out([]byte(runningLine)); len(evs) != 1 || evs[0].Kind != event.TurnStarted {
		t.Errorf("after the read, a running state maps to %s", kinds(evs))
	}
}

// TestUsageReadRunningState: a running state and a streamed part during
// the dashboard's read are statuses, not the turn's events.
func TestUsageReadRunningState(t *testing.T) {
	m := NewMapper()
	m.InInternal([]byte(usageLine))
	if evs := m.Out([]byte(runningLine)); len(evs) != 1 || evs[0].Kind != event.StatusChanged || evs[0].Payload.(event.Status).Detail != "usage report: reading" {
		t.Errorf("a running state during the read maps to %+v", evs)
	}
	delta := `{"type":"stream_event","event":{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Current"}},"session_id":"s","parent_tool_use_id":null}`
	if evs := m.Out([]byte(delta)); len(evs) != 1 || evs[0].Kind != event.StatusChanged || evs[0].Payload.(event.Status).Detail != "usage report: streaming" {
		t.Errorf("a streamed part during the read maps to %+v", evs)
	}
	if evs := m.Out([]byte(`{"type":"assistant","message":{"id":"m","role":"assistant","model":"<synthetic>","content":[{"type":"text","text":"x"}]},"parent_tool_use_id":null,"session_id":"s"}`)); len(evs) != 1 || evs[0].Kind != event.StatusChanged || evs[0].Payload.(event.Status).Detail != "usage report: the answer holds none" {
		t.Errorf("an answer without a report maps to %+v", evs)
	}
	if evs := m.Out([]byte(`{"type":"result","subtype":"success","is_error":false,"num_turns":0,"session_id":"s"}`)); len(evs) != 1 || evs[0].Kind != event.StatusChanged || evs[0].Payload.(event.Status).Detail != "usage report: read" {
		t.Errorf("the read's result maps to %+v", evs)
	}
	if evs := m.Out([]byte(runningLine)); len(evs) != 1 || evs[0].Kind != event.TurnStarted {
		t.Errorf("after the read, a running state maps to %s", kinds(evs))
	}
}

// TestUsageTypedByOwner: the same answer to a "/usage" the owner typed is
// a message of the conversation, and its report still gives the windows.
func TestUsageTypedByOwner(t *testing.T) {
	m := NewMapper()
	all := p24u(t, m, m.In)
	for _, k := range []event.Kind{event.UserMessage, event.Telemetry, event.MessageCompleted, event.TurnCompleted} {
		if !has(all, k) {
			t.Errorf("no %s among: %s", k, kinds(all))
		}
	}
	for _, ev := range all {
		if c, ok := ev.Payload.(event.Completed); ok && !strings.Contains(c.Text, "Fable") {
			t.Errorf("the message's text lacks the page: %.80q", c.Text)
		}
	}
}

// TestUsageReadFailed: a result with no answer before it is the read's
// failure, reported once, and the next turn is the owner's.
func TestUsageReadFailed(t *testing.T) {
	m := NewMapper()
	m.InInternal([]byte(usageLine))
	evs := m.Out([]byte(`{"type":"result","subtype":"error_during_execution","is_error":true,"errors":["usage endpoint unavailable"],"num_turns":0,"session_id":"s"}`))
	if len(evs) != 1 || evs[0].Kind != event.Error || !strings.Contains(evs[0].Payload.(event.Problem).Message, "usage report not read: usage endpoint unavailable") {
		t.Fatalf("the failed read maps to %+v", evs)
	}
	if m.usageReads != 0 {
		t.Errorf("%d reads still pending", m.usageReads)
	}
	m.In([]byte(strings.Replace(usageLine, "/usage", "hello", 1)))
	if evs := m.Out([]byte(`{"type":"result","subtype":"success","is_error":false,"duration_ms":5,"num_turns":1,"session_id":"s"}`)); !has(evs, event.TurnCompleted) {
		t.Errorf("the owner's turn end maps to %s", kinds(evs))
	}
}

// TestUsageReportShapes: the rows' kinds and scopes, and a reset time that
// does not parse.
func TestUsageReportShapes(t *testing.T) {
	m := NewMapper()
	line := `{"type":"assistant","message":{"id":"m","role":"assistant","model":"<synthetic>","content":[{"type":"text","text":"x"}]},"usage_report":{"rate_limits":{"limits":[` +
		`{"kind":"session","percent":50,"resets_at":"soon"},{"kind":"weekly_scoped","percent":3,"resets_at":"2026-10-27T09:00:00+00:00","scope":{"model":{"display_name":"Opus 5.5"}}},` +
		`{"kind":"weekly_scoped","percent":4,"resets_at":"2026-10-27T09:00:00+00:00"},{"kind":"monthly_new","percent":9,"resets_at":"2026-11-01T00:00:00+00:00"}]}},"parent_tool_use_id":null,"session_id":"s"}`
	var tel event.Metrics
	for _, ev := range m.Out([]byte(line)) {
		if p, ok := ev.Payload.(event.Metrics); ok {
			tel = p
		}
	}
	names := map[string]event.Metric{}
	for _, it := range tel.Items {
		names[it.Name] = it
	}
	if len(names) != 3 || !names["quota.5h"].ResetsAt.IsZero() || names["quota.5h"].Value != 50 ||
		names["quota.7d.opus55"].Value != 3 || names["quota.7d.scoped"].Value != 4 {
		t.Errorf("metrics %+v", tel.Items)
	}
	if got := scopeSlug("Fable"); got != "fable" {
		t.Errorf("scopeSlug(Fable) = %q", got)
	}
}
