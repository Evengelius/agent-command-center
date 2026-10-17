package state

import (
	"testing"

	"github.com/Evengelius/agent-command-center/internal/event"
)

// TestAgentModel is plan 5, item 7, criteria 1, 2 and 4: an agent's model is
// only what its own events report, observed; a model reported after its end
// is kept without a "progress after end" note.
func TestAgentModel(t *testing.T) {
	b := newBench(t)
	b.ev(event.SessionStarted, "", event.Started{SessionID: "s1", Model: "claude-opus-5-5"})
	b.ev(event.AgentStarted, "k1", event.Agent{Key: "k1", Type: "subagent", Label: "explore"})
	b.ev(event.AgentStarted, "k2", event.Agent{Key: "k2", Type: "thread", Label: "/root/scan", Model: "gpt-6-sol"})
	if m := b.s.Agents["k1"].Model; m.Present {
		t.Errorf("k1 before any model: %+v, want not present (never the session's)", m)
	}
	if m := b.s.Agents["k2"].Model; !m.Present || m.Text != "gpt-6-sol" || m.Origin != event.Observed || !m.Documented {
		t.Errorf("k2: %+v", m)
	}
	b.ev(event.AgentProgress, "k1", event.Agent{Key: "k1", Model: "claude-haiku-5-5"})
	if m := b.s.Agents["k1"].Model; !m.Present || m.Text != "claude-haiku-5-5" || m.Origin != event.Observed {
		t.Errorf("k1: %+v", m)
	}
	if b.s.Agents["k2"].Model.Text != "gpt-6-sol" {
		t.Errorf("k2 took another agent's model: %+v", b.s.Agents["k2"].Model)
	}
	b.ev(event.AgentProgress, "k1", event.Agent{Key: "k1", LastTool: "Read"}) // no model: kept
	b.ev(event.AgentCompleted, "k1", event.Agent{Key: "k1", Status: "completed"})
	b.ev(event.AgentProgress, "k1", event.Agent{Key: "k1", Model: "claude-sonnet-5-5"})
	a := b.s.Agents["k1"]
	if a.Model.Text != "claude-sonnet-5-5" || a.Status != AgentCompleted || len(a.History) != 0 {
		t.Errorf("late model: %q, status %s, history %q", a.Model.Text, a.Status, a.History)
	}
	b.ev(event.AgentProgress, "k1", event.Agent{Key: "k1", Model: "claude-opus-5-5", LastTool: "Edit"})
	if a.Model.Text != "claude-sonnet-5-5" || len(a.History) != 1 || a.History[0] != "progress after end" {
		t.Errorf("late progress: %q, history %q", a.Model.Text, a.History)
	}
}
