package state_test

import (
	"path/filepath"
	"testing"

	"github.com/Evengelius/agent-command-center/internal/event"
	"github.com/Evengelius/agent-command-center/internal/provider/mock"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// Proposal step 3, item 5: the conversation is real (AGENTS.md). Every entry
// of every record carries the Seq of an applied CLI event of a kind that may
// create an entry of its role, with that event's agent and origin.

var entryKinds = map[string][]event.Kind{
	"user":      {event.UserMessage},
	"assistant": {event.MessageDelta, event.MessageCompleted},
	"tool":      {event.ToolStarted},
	"approval":  {event.ApprovalRequest, event.Question},
	"notice":    {event.StatusChanged},
}

// untraced lists the entries of s that no applied event accounts for.
func untraced(s *state.State, applied map[uint64]event.Event) []string {
	var bad []string
	check := func(where string, list []state.Entry) {
		for i, e := range list {
			ev, ok := applied[e.Seq]
			switch {
			case e.Seq == 0 || !ok:
				bad = append(bad, where+": entry "+itoa(i)+" ("+e.Role+") has no applied event")
			case !kindIn(ev.Kind, entryKinds[e.Role]):
				bad = append(bad, where+": entry "+itoa(i)+" ("+e.Role+") comes from "+string(ev.Kind))
			case ev.Agent != e.Agent || ev.Origin != e.Origin:
				bad = append(bad, where+": entry "+itoa(i)+" ("+e.Role+") differs from its event's agent or origin")
			}
		}
	}
	check("conversation", s.Conversation)
	for _, k := range s.Order {
		check("agent "+k, s.Agents[k].Messages)
	}
	return bad
}

func kindIn(k event.Kind, list []event.Kind) bool {
	for _, x := range list {
		if x == k {
			return true
		}
	}
	return false
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for ; n > 0; n /= 10 {
		b = append([]byte{byte('0' + n%10)}, b...)
	}
	return string(b)
}

func TestEveryEntryTracesToAnEvent(t *testing.T) {
	type source struct {
		name string
		evs  []event.Event
	}
	var sources []source
	for _, r := range recordings {
		evs := replayEvents(t, r.name, r.provider, r.mapper())
		sources = append(sources, source{r.name, evs})
	}
	for _, f := range []string{"claude-subagent-approval.jsonl", "codex-subagents.jsonl"} {
		timed, err := mock.Load(filepath.Join("../../testdata/mock", f))
		if err != nil {
			t.Fatal(err)
		}
		var evs []event.Event
		for i, x := range timed {
			ev := x.Event
			ev.Seq, ev.Origin = uint64(i+1), event.Mock // as the mock session emits them
			evs = append(evs, ev)
		}
		sources = append(sources, source{"mock/" + f, evs})
	}
	for _, src := range sources {
		s := state.New()
		applied := map[uint64]event.Event{}
		for _, ev := range src.evs {
			s.Apply(ev)
			applied[ev.Seq] = ev
		}
		entries := len(s.Conversation)
		for _, k := range s.Order {
			entries += len(s.Agents[k].Messages)
		}
		if entries == 0 {
			t.Errorf("%s: no entries; the check would prove nothing", src.name)
		}
		for _, b := range untraced(s, applied) {
			t.Errorf("%s: %s", src.name, b)
		}
		t.Logf("%s: %d entries traced", src.name, entries)
	}
}

// The check itself fails on planted entries.
func TestUntracedEntryFails(t *testing.T) {
	applied := map[uint64]event.Event{7: {Seq: 7, Kind: event.TurnCompleted}}
	for name, e := range map[string]state.Entry{
		"no seq":       {Role: "assistant", Text: "simulated"},
		"unknown seq":  {Seq: 99, Role: "assistant"},
		"wrong kind":   {Seq: 7, Role: "assistant"},
		"wrong origin": {Seq: 8, Role: "user", Origin: event.AI},
	} {
		s := state.New()
		s.Conversation = append(s.Conversation, e)
		evs := map[uint64]event.Event{8: {Seq: 8, Kind: event.UserMessage, Origin: event.Observed}}
		for k, v := range applied {
			evs[k] = v
		}
		if len(untraced(s, evs)) != 1 {
			t.Errorf("%s: planted entry not reported", name)
		}
	}
}
