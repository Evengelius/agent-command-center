package claude

import (
	"encoding/json"
	"path/filepath"
	"slices"
	"testing"

	"github.com/Evengelius/agent-command-center/internal/event"
)

// streamMessageID is the message id of a mapped stream event.
func streamMessageID(e event.Event) (string, bool) {
	switch p := e.Payload.(type) {
	case event.Delta:
		return p.MessageID, true
	case event.Marker:
		return p.MessageID, true
	}
	return "", false
}

// Plan 21, item 3.1: in every Claude Code recording, each stream_event line
// maps to the id of its agent's last message_start, which equals the line's
// undocumented api_message_id.
func TestStreamEventsTakeTheMessageStartID(t *testing.T) {
	names, err := filepath.Glob("../../../testdata/probes/p*-claude-*.jsonl")
	if err != nil || len(names) == 0 {
		t.Fatalf("no recordings: %v", err)
	}
	slices.Sort(names)
	total, files := 0, 0
	for _, path := range names {
		name := filepath.Base(path)
		recs, per := replay(t, name)
		last := map[string]string{} // agent key -> last message_start id
		n := 0
		for i, r := range recs {
			if r.Dir != "out" {
				continue
			}
			var l struct {
				Type            string  `json:"type"`
				ParentToolUseID *string `json:"parent_tool_use_id"`
				AgentID         string  `json:"agent_id"`
				APIMessageID    string  `json:"api_message_id"`
				Event           struct {
					Type    string `json:"type"`
					Message struct {
						ID string `json:"id"`
					} `json:"message"`
				} `json:"event"`
			}
			if json.Unmarshal(r.Msg, &l) != nil || l.Type != "stream_event" {
				continue
			}
			agent := ""
			if l.ParentToolUseID != nil && *l.ParentToolUseID != "" {
				if agent = l.AgentID; agent == "" {
					agent = "tool:" + *l.ParentToolUseID
				}
			}
			if l.Event.Type == "message_start" {
				last[agent] = l.Event.Message.ID
			}
			want := last[agent]
			if want == "" {
				t.Errorf("%s line %d: no message_start before this stream event", name, i+1)
				continue
			}
			if l.APIMessageID != want {
				t.Errorf("%s line %d: api_message_id %q, last message_start %q", name, i+1, l.APIMessageID, want)
			}
			got := 0
			for _, e := range per[i] {
				id, ok := streamMessageID(e)
				if !ok {
					continue
				}
				got++
				if id != want {
					t.Errorf("%s line %d: mapped id %q, want %q", name, i+1, id, want)
				}
			}
			if got != 1 {
				t.Errorf("%s line %d: %d stream events mapped, want 1", name, i+1, got)
			}
			n++
		}
		total += n
		if n > 0 {
			files++
		}
		t.Logf("%s: %d stream_event lines", name, n)
	}
	t.Logf("%d stream_event lines in %d of %d recordings", total, files, len(names))
	// 1,237 in 13 of 14 recordings until plan 22 added P12c-block, whose 162
	// stream events all take their message_start's id too; then 1,399 in 14 of 15
	// until amendment 20 added P22o, whose 36 do too; then 1,435 in 15 of 16
	// until plan 24 added P24u, which has no stream event, nor has plan 34's
	// P34c, nor plan 39's P39c.
	if total != 1435 || files != 15 || len(names) != 19 {
		t.Errorf("%d lines in %d of %d recordings, want 1435 in 15 of 19", total, files, len(names))
	}
}

// Plan 21, item 3.3: an agent's stream event with no message_start before it
// takes the line's undocumented api_message_id; a message_start later takes
// over; another agent's message_start changes nothing for this one.
func TestStreamEventWithoutMessageStartFallsBack(t *testing.T) {
	m := NewMapper()
	delta := func(parent, apiID, text string) string {
		p := "null"
		if parent != "" {
			p = `"` + parent + `"`
		}
		return `{"type":"stream_event","parent_tool_use_id":` + p + `,"api_message_id":"` + apiID + `","event":{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"` + text + `"}}}`
	}
	idOf := func(raw string) string {
		t.Helper()
		evs := m.Out([]byte(raw))
		if len(evs) != 1 {
			t.Fatalf("%d events for %s", len(evs), raw)
		}
		id, ok := streamMessageID(evs[0])
		if !ok {
			t.Fatalf("not a stream event: %#v", evs[0])
		}
		return id
	}
	if got := idOf(delta("", "msg_api_1", "a")); got != "msg_api_1" {
		t.Errorf("delta with no message_start: id %q, want msg_api_1", got)
	}
	idOf(`{"type":"stream_event","parent_tool_use_id":"toolu_x","agent_id":"ag1","api_message_id":"msg_sub","event":{"type":"message_start","message":{"id":"msg_sub","model":"m"}}}`)
	if got := idOf(delta("", "msg_api_2", "b")); got != "msg_api_2" {
		t.Errorf("main delta after a subagent's message_start: id %q, want msg_api_2", got)
	}
	idOf(`{"type":"stream_event","parent_tool_use_id":null,"api_message_id":"msg_api_3","event":{"type":"message_start","message":{"id":"msg_start","model":"m"}}}`)
	if got := idOf(delta("", "msg_api_3", "c")); got != "msg_start" {
		t.Errorf("delta after message_start: id %q, want msg_start", got)
	}
}
