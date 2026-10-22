package state_test

import (
	"path/filepath"
	"reflect"
	"testing"

	"github.com/Evengelius/agent-command-center/internal/event"
	"github.com/Evengelius/agent-command-center/internal/fixture"
	"github.com/Evengelius/agent-command-center/internal/provider/claude"
	"github.com/Evengelius/agent-command-center/internal/provider/codex"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// Test 4 (proposal step 0, item 7): replaying a recording through its mapper
// and the reducer twice gives the same state, field for field.

var recordings = []struct {
	name     string
	provider string
	mapper   func() fixture.Mapper
}{
	{"p1-claude-stream.jsonl", claude.ProviderID, func() fixture.Mapper { return claude.NewMapper() }},
	{"p2-claude-conversation.jsonl", claude.ProviderID, func() fixture.Mapper { return claude.NewMapper() }},
	{"p3-claude-subagent.jsonl", claude.ProviderID, func() fixture.Mapper { return claude.NewMapper() }},
	{"p3b-claude-subagent-approval.jsonl", claude.ProviderID, func() fixture.Mapper { return claude.NewMapper() }},
	{"p4-claude-resume.jsonl", claude.ProviderID, func() fixture.Mapper { return claude.NewMapper() }},
	{"p6-codex-app-server.jsonl", codex.ProviderID, func() fixture.Mapper { return codex.NewMapper() }},
	{"p6b-codex-app-server.jsonl", codex.ProviderID, func() fixture.Mapper { return codex.NewMapper() }},
	{"p7-codex-subagents.jsonl", codex.ProviderID, func() fixture.Mapper { return codex.NewMapper() }},
	{"p7b-codex-subagents.jsonl", codex.ProviderID, func() fixture.Mapper { return codex.NewMapper() }},
	{"p12c-claude-auto.jsonl", claude.ProviderID, func() fixture.Mapper { return claude.NewMapper() }},
	{"p12c-block-claude-auto-boundary-edit.jsonl", claude.ProviderID, func() fixture.Mapper { return claude.NewMapper() }}, // plan 22: four thinking blocks and the live estimates
	{"p12c-switch-claude-default-to-auto.jsonl", claude.ProviderID, func() fixture.Mapper { return claude.NewMapper() }},
	{"p12x-codex-auto-review.jsonl", codex.ProviderID, func() fixture.Mapper { return codex.NewMapper() }},
	{"p12x-switch-codex-user-to-auto-review.jsonl", codex.ProviderID, func() fixture.Mapper { return codex.NewMapper() }},
	{"p13c-claude-auto-classifier-deny.jsonl", claude.ProviderID, func() fixture.Mapper { return claude.NewMapper() }},
	{"p14a-claude-question-answer.jsonl", claude.ProviderID, func() fixture.Mapper { return claude.NewMapper() }},
	{"p22o-claude-opus-thinking.jsonl", claude.ProviderID, func() fixture.Mapper { return claude.NewMapper() }},  // plan 22, amendment 20: Opus thinks; the owner's SessionStart hook
	{"p22s-codex-reasoning-summary.jsonl", codex.ProviderID, func() fixture.Mapper { return codex.NewMapper() }}, // plan 22, amendment 20: summaries asked for
	{"p303a-claude-conversation.jsonl", claude.ProviderID, func() fixture.Mapper { return claude.NewMapper() }},
	{"p303b-claude-subagent-approval.jsonl", claude.ProviderID, func() fixture.Mapper { return claude.NewMapper() }},
	{"p303c-claude-default-to-auto.jsonl", claude.ProviderID, func() fixture.Mapper { return claude.NewMapper() }},
	{"p303d-claude-question-answer.jsonl", claude.ProviderID, func() fixture.Mapper { return claude.NewMapper() }},
	{"p303e-claude-interrupt-tool.jsonl", claude.ProviderID, func() fixture.Mapper { return claude.NewMapper() }},
}

func replayEvents(t *testing.T, name, provider string, m fixture.Mapper) []event.Event {
	t.Helper()
	recs, err := fixture.Read(filepath.Join("../../testdata/probes", name))
	if err != nil {
		t.Fatal(err)
	}
	return fixture.Flatten(fixture.Replay(provider, recs, m))
}

func replayState(t *testing.T, name, provider string, m fixture.Mapper) (*state.State, int) {
	t.Helper()
	s := state.New()
	n := 0
	for _, ev := range replayEvents(t, name, provider, m) {
		s.Apply(ev)
		n++
	}
	return s, n
}

func TestReplayDeterministic(t *testing.T) {
	for _, r := range recordings {
		a, n := replayState(t, r.name, r.provider, r.mapper())
		b, _ := replayState(t, r.name, r.provider, r.mapper())
		if !reflect.DeepEqual(a, b) {
			t.Errorf("%s: two replays differ", r.name)
		}
		t.Logf("%s: %d events; session %s; turns %d; entries %d; agents %d; approvals %d; tools %d; errors %d; notes %d; unrecognised %d",
			r.name, n, a.Session.Phase, a.Turn.Count, len(a.Conversation), len(a.Agents), len(a.Approvals), len(a.Tools), len(a.Errors), len(a.Notes), a.Unrecognised)
		for _, x := range a.Notes {
			t.Logf("  %s", x)
		}
		for _, u := range a.UnrecognisedLast {
			t.Logf("  %s %s: %.120s", u.Type, u.Reason, u.Line)
		}
	}
}
