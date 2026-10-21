package claude

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"slices"
	"testing"

	"github.com/Evengelius/agent-command-center/internal/event"
	"github.com/Evengelius/agent-command-center/internal/fixture"
)

// markersNamed are the stream markers of one name, in order, whatever their
// agent.
func markersNamed(evs []event.Event, name string) []event.Event {
	var out []event.Event
	for _, e := range evs {
		if p, ok := e.Payload.(event.Marker); ok && e.Kind == event.MessageMarker && p.Name == name {
			out = append(out, e)
		}
	}
	return out
}

// thinkingCounts are the message_delta markers' thinking counts.
func thinkingCounts(evs []event.Event) (counts []int64, allHave bool) {
	allHave = true
	for _, e := range markersNamed(evs, "message_delta") {
		p := e.Payload.(event.Marker)
		counts = append(counts, p.Thinking)
		allHave = allHave && p.HasThinking
	}
	return counts, allHave
}

// estimates are the live estimates of the thinking.estimate metrics, in order.
func estimates(t *testing.T, evs []event.Event) []float64 {
	t.Helper()
	var out []float64
	for _, e := range evs {
		m, ok := e.Payload.(event.Metrics)
		if !ok {
			continue
		}
		for _, it := range m.Items {
			if it.Name != event.ThinkingEstimate {
				continue
			}
			if e.Agent != "" || e.Origin != event.Observed || it.Unit != "tokens" || !it.Documented || it.Source != "system/thinking_tokens" || it.Reason != "" || it.Text != "" {
				t.Errorf("estimate %+v of agent %q, origin %q", it, e.Agent, e.Origin)
			}
			out = append(out, it.Value)
		}
	}
	return out
}

// Plan 22, criterion 1.1 (C3): each message_delta marker carries its message's
// exact thinking-token count, here P303b's three messages, and every one has
// it. The subagent's thinking comes in assistant lines only (C6): no marker.
func TestP303bThinkingCounts(t *testing.T) {
	_, per := replay(t, "p303b-claude-subagent-approval.jsonl")
	all := fixture.Flatten(per)
	got, allHave := thinkingCounts(all)
	if want := []int64{719, 281, 268}; !slices.Equal(got, want) || !allHave {
		t.Errorf("thinking counts %v (all reported %v), want %v (true)", got, allHave, want)
	}
	for _, e := range markersNamed(all, "message_delta") {
		if e.Agent != "" || e.Origin != event.Observed {
			t.Errorf("message_delta of agent %q, origin %q", e.Agent, e.Origin)
		}
	}
	// The three counts sum to the turn's result: 1,268 (C4), and the live
	// estimates of each block are another thing, an estimate (C5).
	var sum int64
	for _, n := range got {
		sum += n
	}
	if sum != 1268 {
		t.Errorf("the counts sum to %d, want 1268, the result's total", sum)
	}
	if want := []float64{50, 300, 650, 50, 250, 374, 50, 200, 360}; !slices.Equal(estimates(t, all), want) {
		t.Errorf("estimates %v, want %v", estimates(t, all), want)
	}
}

// P12c-block (C2, C3): four thinking blocks, their exact counts and, for the
// second, the live estimate from 50 to 2,700 every second or so; the third
// block, 56 ms long, has none.
func TestP12cBlockThinkingCounts(t *testing.T) {
	_, per := replay(t, "p12c-block-claude-auto-boundary-edit.jsonl")
	all := fixture.Flatten(per)
	got, allHave := thinkingCounts(all)
	if want := []int64{717, 2852, 17, 354}; !slices.Equal(got, want) || !allHave {
		t.Errorf("thinking counts %v (all reported %v), want %v (true)", got, allHave, want)
	}
	want := []float64{
		50, 350, 450, 734, // the first block: the last value is the uneven one at its stop
		50, 350, 650, 850, 1100, 1450, 1700, 1950, 2200, 2450, 2700, // the second
		50, 250, 447, // the fourth
	}
	if got := estimates(t, all); !slices.Equal(got, want) {
		t.Errorf("estimates %v, want %v", got, want)
	}
}

// C7: a message cut short, with its stop_reason null, still reports its count
// (0): the marker says what the line says, and the reducer reads the rest.
func TestInterruptedMessageDeltaReportsZero(t *testing.T) {
	_, per := replay(t, "p303a-claude-conversation.jsonl")
	n := 0
	for _, e := range markersNamed(fixture.Flatten(per), "message_delta") {
		p := e.Payload.(event.Marker)
		if p.StopReason != "" {
			continue
		}
		n++
		if !p.HasThinking || p.Thinking != 0 {
			t.Errorf("interrupted message_delta %+v, want a reported 0", p)
		}
	}
	if n != 1 {
		t.Errorf("%d message_delta lines with a null stop_reason, want 1", n)
	}
}

// Plan 22, criterion 1.1: a message_delta line without
// usage.output_tokens_details.thinking_tokens reports no count, never a zero;
// a reported 0 is a count. An odd shape cannot turn the line into an
// unrecognised one.
func TestMessageDeltaThinkingPresence(t *testing.T) {
	delta := func(usage string) string {
		return `{"type":"stream_event","parent_tool_use_id":null,"api_message_id":"msg_x","event":{"type":"message_delta","delta":{"stop_reason":"end_turn"}` + usage + `}}`
	}
	for _, c := range []struct {
		name, usage string
		n           int64
		has         bool
	}{
		{"count", `,"usage":{"output_tokens":9,"output_tokens_details":{"thinking_tokens":281}}`, 281, true},
		{"zero", `,"usage":{"output_tokens":9,"output_tokens_details":{"thinking_tokens":0}}`, 0, true},
		{"no usage", ``, 0, false},
		{"usage without details", `,"usage":{"output_tokens":9}`, 0, false},
		{"empty details", `,"usage":{"output_tokens":9,"output_tokens_details":{}}`, 0, false},
		{"null details", `,"usage":{"output_tokens":9,"output_tokens_details":null}`, 0, false},
		{"null count", `,"usage":{"output_tokens_details":{"thinking_tokens":null}}`, 0, false},
		{"text count", `,"usage":{"output_tokens_details":{"thinking_tokens":"12"}}`, 0, false},
		{"usage not an object", `,"usage":"n/a"`, 0, false},
		{"details not an object", `,"usage":{"output_tokens_details":[1]}`, 0, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			evs := NewMapper().Out([]byte(delta(c.usage)))
			if len(evs) != 1 {
				t.Fatalf("%d events, want one marker", len(evs))
			}
			if evs[0].Kind != event.MessageMarker {
				t.Fatalf("event %v, want a marker", evs[0].Kind)
			}
			p := evs[0].Payload.(event.Marker)
			if p.Name != "message_delta" || p.StopReason != "end_turn" || p.Thinking != c.n || p.HasThinking != c.has {
				t.Errorf("marker %+v, want message_delta end_turn, thinking %d, reported %v", p, c.n, c.has)
			}
		})
	}
}

// independent reads of the lines, apart from the mapper: the thinking count of
// a message_delta, and the index of a content block event.
type streamLine struct {
	Type            string  `json:"type"`
	ParentToolUseID *string `json:"parent_tool_use_id"`
	AgentID         string  `json:"agent_id"`
	Event           struct {
		Type  string `json:"type"`
		Index *int   `json:"index"`
		Usage struct {
			Details *struct {
				Thinking *float64 `json:"thinking_tokens"`
			} `json:"output_tokens_details"`
		} `json:"usage"`
	} `json:"event"`
}

func claudeRecordings(t *testing.T) []string {
	t.Helper()
	names, err := filepath.Glob("../../../testdata/probes/p*-claude-*.jsonl")
	if err != nil || len(names) == 0 {
		t.Fatalf("no recordings: %v", err)
	}
	slices.Sort(names)
	for i, n := range names {
		names[i] = filepath.Base(n)
	}
	return names
}

// In every recording, each message_delta marker takes its count from the line
// as the line gives it.
func TestMessageDeltaThinkingMatchesEveryLine(t *testing.T) {
	deltas, reported := 0, 0
	for _, name := range claudeRecordings(t) {
		recs, per := replay(t, name)
		for i, r := range recs {
			var l streamLine
			if r.Dir != "out" || json.Unmarshal(r.Msg, &l) != nil || l.Type != "stream_event" || l.Event.Type != "message_delta" {
				continue
			}
			deltas++
			var got []event.Marker
			for _, e := range markersNamed(per[i], "message_delta") {
				got = append(got, e.Payload.(event.Marker))
			}
			if len(got) != 1 {
				t.Errorf("%s line %d: %d message_delta markers, want 1", name, i+1, len(got))
				continue
			}
			has := l.Event.Usage.Details != nil && l.Event.Usage.Details.Thinking != nil
			if got[0].HasThinking != has {
				t.Errorf("%s line %d: reported %v, the line has the count %v", name, i+1, got[0].HasThinking, has)
				continue
			}
			if has {
				reported++
				if got[0].Thinking != int64(*l.Event.Usage.Details.Thinking) {
					t.Errorf("%s line %d: %d thinking tokens, the line says %v", name, i+1, got[0].Thinking, *l.Event.Usage.Details.Thinking)
				}
			}
		}
	}
	if deltas == 0 || reported == 0 {
		t.Errorf("%d message_delta lines, %d with a count", deltas, reported)
	}
	t.Logf("%d message_delta lines in the recordings, %d with a thinking count", deltas, reported)
}

// Plan 22, criterion 1.1 (C1): each content_block_start and content_block_stop
// marker carries its block's index, as the line gives it, so a block's stop
// finds its start: in every recording the stops close the open block, one at
// a time per message. The other markers carry none.
func TestBlockMarkersCarryTheirIndex(t *testing.T) {
	blocks, later := 0, 0
	for _, name := range claudeRecordings(t) {
		recs, per := replay(t, name)
		open := map[string]int{} // agent -> the open block's index; absent when none
		for i, r := range recs {
			var l streamLine
			if r.Dir != "out" || json.Unmarshal(r.Msg, &l) != nil || l.Type != "stream_event" || l.Event.Type == "content_block_delta" {
				continue // a delta is a text event, not a marker
			}
			agent := ""
			if l.ParentToolUseID != nil && *l.ParentToolUseID != "" {
				if agent = l.AgentID; agent == "" {
					agent = "tool:" + *l.ParentToolUseID
				}
			}
			var markers []event.Marker
			for _, e := range per[i] {
				if p, ok := e.Payload.(event.Marker); ok {
					markers = append(markers, p)
				}
			}
			if len(markers) != 1 {
				t.Errorf("%s line %d: %d markers for a %s, want 1", name, i+1, len(markers), l.Event.Type)
				continue
			}
			p := markers[0]
			if l.Event.Type == "message_start" {
				delete(open, agent) // a new message: its blocks are its own
			}
			switch l.Event.Type {
			case "content_block_start", "content_block_stop":
				if l.Event.Index == nil {
					t.Errorf("%s line %d: a %s without an index", name, i+1, l.Event.Type)
					continue
				}
				if p.Index != *l.Event.Index {
					t.Errorf("%s line %d: index %d, the line says %d", name, i+1, p.Index, *l.Event.Index)
				}
				cur, isOpen := open[agent]
				if l.Event.Type == "content_block_start" {
					blocks++
					if p.Index >= 1 {
						later++
					}
					if isOpen {
						t.Errorf("%s line %d: block %d starts with block %d open", name, i+1, p.Index, cur)
					}
					open[agent] = p.Index
				} else {
					if !isOpen || cur != p.Index {
						t.Errorf("%s line %d: block %d stops, the open one is %d (open %v)", name, i+1, p.Index, cur, isOpen)
					}
					delete(open, agent)
				}
			default:
				if p.Index != 0 {
					t.Errorf("%s line %d: a %s marker with index %d", name, i+1, l.Event.Type, p.Index)
				}
			}
		}
	}
	if blocks == 0 || later == 0 {
		t.Errorf("%d blocks, %d at index 1 or more", blocks, later)
	}
	t.Logf("%d content blocks in the recordings, %d at index 1 or more", blocks, later)
}

// P303b and P12c-block, by hand (plan 22, C1): a thinking block is index 0
// and first in its message; the block after it is 1; P12c-block's third
// block of a message, a tool use, is 2.
func TestBlockIndexesByHand(t *testing.T) {
	for _, c := range []struct {
		file string
		want []string
	}{
		{"p303b-claude-subagent-approval.jsonl", []string{"0:thinking", "1:tool_use", "0:thinking", "1:tool_use", "0:thinking", "1:text"}},
		{"p12c-block-claude-auto-boundary-edit.jsonl", []string{"0:thinking", "1:text", "0:thinking", "1:text", "2:tool_use", "0:thinking", "1:text", "2:tool_use", "0:thinking", "1:text"}},
	} {
		_, per := replay(t, c.file)
		var got, stops []string
		for _, e := range markersNamed(fixture.Flatten(per), "content_block_start") {
			p := e.Payload.(event.Marker)
			got = append(got, fmt.Sprintf("%d:%s", p.Index, p.Block))
		}
		for _, e := range markersNamed(fixture.Flatten(per), "content_block_stop") {
			stops = append(stops, fmt.Sprintf("%d", e.Payload.(event.Marker).Index))
		}
		if !slices.Equal(got, c.want) {
			t.Errorf("%s: block starts %v, want %v", c.file, got, c.want)
		}
		if len(stops) != len(c.want) {
			t.Errorf("%s: %d block stops, want %d", c.file, len(stops), len(c.want))
		}
		for i, s := range stops {
			if i < len(c.want) && s != c.want[i][:1] {
				t.Errorf("%s: stop %d has index %s, want %s", c.file, i+1, s, c.want[i][:1])
			}
		}
	}
}

// The metric's name is the constant's: the reducer and the mock files read it
// by that string.
func TestThinkingEstimateName(t *testing.T) {
	if event.ThinkingEstimate != "thinking.estimate" {
		t.Errorf("event.ThinkingEstimate = %q", event.ThinkingEstimate)
	}
	evs := NewMapper().Out([]byte(`{"type":"system","subtype":"thinking_tokens","estimated_tokens":650,"estimated_tokens_delta":350,"session_id":"s","uuid":"u"}`))
	if len(evs) != 1 || evs[0].Kind != event.Telemetry {
		t.Fatalf("%d events, want one telemetry", len(evs))
	}
	m := evs[0].Payload.(event.Metrics)
	if len(m.Items) != 1 || m.Items[0].Name != "thinking.estimate" || m.Items[0].Value != 650 || !m.Items[0].Documented {
		t.Errorf("metrics %+v, want thinking.estimate 650, documented", m.Items)
	}
}
