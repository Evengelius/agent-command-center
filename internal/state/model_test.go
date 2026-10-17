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

// TestRequestedModel is plan 6, item 4, criterion 2: a model a spawn call
// requested fills only an empty model, sourced "spawn request"; the agent's
// own report wins in either order.
func TestRequestedModel(t *testing.T) {
	req := func(k, m string) event.Agent { return event.Agent{Key: k, Model: m, ModelRequested: true} }
	b := newBench(t)
	b.ev(event.AgentStarted, "k1", event.Agent{Key: "k1", Type: "thread", Label: "/root/a"})
	b.ev(event.AgentProgress, "k1", req("k1", "gpt-6-mini"))
	if m := b.s.Agents["k1"].Model; !m.Present || m.Text != "gpt-6-mini" || m.Source != "spawn request" {
		t.Errorf("requested: %+v", m)
	}
	b.ev(event.AgentProgress, "k1", event.Agent{Key: "k1", Model: "gpt-6-sol"}) // its own report replaces it
	if m := b.s.Agents["k1"].Model; m.Text != "gpt-6-sol" || m.Source != "agent model" {
		t.Errorf("own after requested: %+v", m)
	}
	b.ev(event.AgentProgress, "k1", req("k1", "gpt-6-mini")) // a later request does not replace it
	if m := b.s.Agents["k1"].Model; m.Text != "gpt-6-sol" || m.Source != "agent model" {
		t.Errorf("requested after own: %+v", m)
	}
	b.ev(event.AgentStarted, "k2", event.Agent{Key: "k2", Type: "thread", Label: "/root/b", Model: "gpt-6-sol"})
	b.ev(event.AgentProgress, "k2", req("k2", "gpt-6-mini"))
	if m := b.s.Agents["k2"].Model; m.Text != "gpt-6-sol" {
		t.Errorf("k2: %+v", m)
	}
	b.ev(event.AgentCompleted, "k1", event.Agent{Key: "k1", Status: "completed"})
	late := req("k1", "gpt-6-mini")
	late.Parent = "k0"
	b.ev(event.AgentProgress, "k1", late) // a request after the end is not progress
	if h := b.s.Agents["k1"].History; len(h) != 0 {
		t.Errorf("k1 history: %q", h)
	}
	early := req("k3", "gpt-6-mini")
	early.Parent = "k2"
	b.ev(event.AgentProgress, "k3", early) // before any announcement of k3
	if a := b.s.Agents["k3"]; a.Model.Text != "gpt-6-mini" || a.Parent != "k2" {
		t.Errorf("k3: %+v, parent %q", a.Model, a.Parent)
	}
}

// TestRequestedEffort is plan 6, item 6, criteria 1 and 3: the agent's own
// effort always wins over a spawn request's, in either order; an ended agent
// still takes an effort-only event, and nothing else.
func TestRequestedEffort(t *testing.T) {
	req := func(k, e string) event.Agent { return event.Agent{Key: k, Effort: e, ModelRequested: true} }
	b := newBench(t)
	b.ev(event.AgentStarted, "k1", event.Agent{Key: "k1", Type: "thread", Label: "/root/a"})
	b.ev(event.AgentProgress, "k1", req("k1", "low"))
	if e := b.s.Agents["k1"].Effort; !e.Present || e.Text != "low" || e.Source != "spawn request" {
		t.Errorf("requested: %+v", e)
	}
	if m := b.s.Agents["k1"].Model; m.Present {
		t.Errorf("an effort-only request gave a model: %+v", m)
	}
	b.ev(event.AgentProgress, "k1", event.Agent{Key: "k1", Model: "gpt-6-sol"}) // a report without effort keeps it
	if e := b.s.Agents["k1"].Effort; e.Text != "low" || e.Source != "spawn request" {
		t.Errorf("model-only report: %+v", e)
	}
	b.ev(event.AgentProgress, "k1", event.Agent{Key: "k1", Effort: "high"}) // its own report replaces it
	if e := b.s.Agents["k1"].Effort; e.Text != "high" || e.Source != "agent effort" {
		t.Errorf("own after requested: %+v", e)
	}
	b.ev(event.AgentProgress, "k1", req("k1", "low")) // a later request does not replace it
	if e := b.s.Agents["k1"].Effort; e.Text != "high" || e.Source != "agent effort" {
		t.Errorf("requested after own: %+v", e)
	}
	b.ev(event.AgentCompleted, "k1", event.Agent{Key: "k1", Status: "completed"})
	late := req("k1", "low")
	late.Parent = "k0"
	b.ev(event.AgentProgress, "k1", late) // a request after the end is not progress
	b.ev(event.AgentProgress, "k1", event.Agent{Key: "k1", Effort: "medium", ToolUses: 3})
	if a := b.s.Agents["k1"]; len(a.History) != 1 || a.History[0] != "progress after end" || a.Effort.Text != "high" || a.ToolUses == 3 {
		t.Errorf("k1 after its end: history %q, effort %+v, tool uses %d", a.History, a.Effort, a.ToolUses)
	}
	b.ev(event.AgentProgress, "k1", event.Agent{Key: "k1", Effort: "xhigh"}) // its own, after its end
	if e := b.s.Agents["k1"].Effort; e.Text != "xhigh" {
		t.Errorf("own after the end: %+v", e)
	}
}
