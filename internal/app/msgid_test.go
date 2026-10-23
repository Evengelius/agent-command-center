package app

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"slices"
	"sort"
	"testing"

	"github.com/Evengelius/agent-command-center/internal/fixture"
	"github.com/Evengelius/agent-command-center/internal/provider/claude"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// withoutAPIMessageID removes the undocumented api_message_id from the top
// level of every line the CLI wrote (plan 21, item 3).
func withoutAPIMessageID(t *testing.T, recs []fixture.Record) ([]fixture.Record, int) {
	t.Helper()
	out := make([]fixture.Record, len(recs))
	removed := 0
	for i, r := range recs {
		out[i] = r
		if r.Dir != "out" || !bytes.Contains(r.Msg, []byte(`"api_message_id"`)) {
			continue
		}
		var obj map[string]json.RawMessage
		if json.Unmarshal(r.Msg, &obj) != nil {
			continue
		}
		if _, ok := obj["api_message_id"]; !ok {
			continue
		}
		delete(obj, "api_message_id")
		b, err := json.Marshal(obj)
		if err != nil {
			t.Fatal(err)
		}
		out[i].Msg = b
		removed++
	}
	return out, removed
}

// replayState applies every event of a recording to a fresh state.
func replayState(recs []fixture.Record) *state.State {
	s := state.New()
	for _, e := range fixture.Flatten(fixture.Replay(claude.ProviderID, recs, claude.NewMapper())) {
		s.Apply(e)
	}
	return s
}

// conversation is what the message id decides: each entry's agent, role,
// message id and text.
func conversation(es []state.Entry) []entryKey {
	out := make([]entryKey, len(es))
	for i, e := range es {
		out[i] = entryKey{agent: e.Agent, role: e.Role, id: e.MessageID, text: e.Text}
	}
	return out
}

// entryKey is what the comparison reads of an entry: a comparable value,
// since an entry's images make it a slice holder (plan 31).
type entryKey struct{ agent, role, id, text string }

// Plan 21, item 3.2: every Claude Code recording replayed twice, as recorded
// and with api_message_id removed from every line, gives the same
// conversation, the main thread's and each agent's, with one entry per
// message. The mapper takes the id from message_start first, so the
// undocumented field is not needed.
func TestConversationDoesNotNeedAPIMessageID(t *testing.T) {
	names, err := filepath.Glob(filepath.Join("..", "..", "testdata", "probes", "p*-claude-*.jsonl"))
	if err != nil || len(names) == 0 {
		t.Fatalf("no recordings: %v", err)
	}
	slices.Sort(names)
	entries, removedTotal := 0, 0
	for _, path := range names {
		name := filepath.Base(path)
		recs := probe(t, name)
		bare, removed := withoutAPIMessageID(t, recs)
		removedTotal += removed
		with, without := replayState(recs), replayState(bare)

		for label, s := range map[string]*state.State{"as recorded": with, "without api_message_id": without} {
			if err := oneEntryPerMessage(s, 0); err != nil {
				t.Errorf("%s, %s: %v", name, label, err)
			}
		}
		a, b := conversation(with.Conversation), conversation(without.Conversation)
		if !slices.Equal(a, b) {
			t.Errorf("%s: the conversation differs without api_message_id: %d entries, then %d", name, len(a), len(b))
		}
		entries += len(a)

		keys := make([]string, 0, len(with.Agents))
		for k := range with.Agents {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		if len(without.Agents) != len(with.Agents) {
			t.Errorf("%s: %d agents, then %d without api_message_id", name, len(with.Agents), len(without.Agents))
		}
		for _, k := range keys {
			other := without.Agents[k]
			if other == nil {
				t.Errorf("%s: agent %s missing without api_message_id", name, k)
				continue
			}
			if !slices.Equal(conversation(with.Agents[k].Messages), conversation(other.Messages)) {
				t.Errorf("%s: agent %s: messages differ without api_message_id", name, k)
			}
		}
		t.Logf("%s: %d entries, %d lines stripped", name, len(a), removed)
	}
	if entries == 0 || removedTotal == 0 {
		t.Errorf("%d entries, %d lines stripped: nothing was compared", entries, removedTotal)
	}
}
