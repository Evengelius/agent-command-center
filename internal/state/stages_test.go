package state_test

import (
	"strings"
	"testing"

	"github.com/Evengelius/agent-command-center/internal/event"
	"github.com/Evengelius/agent-command-center/internal/provider/claude"
	"github.com/Evengelius/agent-command-center/internal/provider/codex"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// Plan 4, step 6, checks 1 and 4: the stages and agents the recordings give.

func outcomes(s *state.State) string {
	var out []string
	for _, st := range s.Stages {
		o := st.Outcome
		if st.Open {
			o = "open"
		}
		out = append(out, o)
	}
	return strings.Join(out, ",")
}

// P2: three turns, the last interrupted; its Write under the turn it ran in.
func TestStagesP2(t *testing.T) {
	s, _ := replayState(t, "p2-claude-conversation.jsonl", claude.ProviderID, claude.NewMapper())
	if got := outcomes(s); got != "completed,completed,interrupted" {
		t.Fatalf("stages %q, want completed,completed,interrupted", got)
	}
	var writes []*state.ToolCall
	for _, tc := range s.Tools {
		if tc.Name == "Write" {
			writes = append(writes, tc)
		}
	}
	if len(writes) != 1 || writes[0].Agent != "" || writes[0].Stage < 1 || writes[0].Stage > 3 {
		t.Fatalf("Write calls %+v, want one, of the orchestrator, in a stage", writes)
	}
	for _, e := range s.Conversation {
		if e.Role == "tool" && e.MessageID == writes[0].ID && writes[0].Seq != e.Seq {
			t.Errorf("Write seq %d, its conversation entry %d", writes[0].Seq, e.Seq)
		}
	}
	for i, st := range s.Stages {
		if st.N != i+1 || st.StartedAt.IsZero() || st.EndedAt.IsZero() || st.ByCLI || st.Prompt == "" {
			t.Errorf("stage %d: %+v", i, st)
		}
	}
}

// P3b: the first turn holds its Agent tool and the background subagent; the
// subagent runs, waits twice for an approval, and completes; Claude Code then
// opens a second turn itself for the report.
func TestStagesP3b(t *testing.T) {
	var statuses []state.AgentStatus
	s := state.New()
	const key = "a9dcbbc8d63adf34c"
	for _, ev := range replayEvents(t, "p3b-claude-subagent-approval.jsonl", claude.ProviderID, claude.NewMapper()) {
		s.Apply(ev)
		if a := s.Agents[key]; a != nil && (len(statuses) == 0 || statuses[len(statuses)-1] != a.Status) {
			statuses = append(statuses, a.Status)
		}
	}
	// The second turn is the CLI's own, for the agent's report.
	if got := outcomes(s); got != "completed,completed" || s.Stages[0].ByCLI || !s.Stages[1].ByCLI ||
		!strings.HasPrefix(s.Stages[0].Prompt, "Use the Agent tool") || s.Stages[1].Prompt != "" {
		t.Fatalf("stages %+v, want two completed, the second opened by the CLI", s.Stages)
	}
	a := s.Agents[key]
	if a == nil || a.Stage != 1 || a.Parent != "" || !a.Backgrounded {
		t.Fatalf("agent %+v, want a background agent of stage 1 at the top", a)
	}
	want := []state.AgentStatus{state.AgentRunning, state.AgentWaiting, state.AgentRunning, state.AgentWaiting, state.AgentRunning, state.AgentCompleted}
	if len(statuses) != len(want) {
		t.Fatalf("statuses %v, want %v", statuses, want)
	}
	for i := range want {
		if statuses[i] != want[i] {
			t.Fatalf("statuses %v, want %v", statuses, want)
		}
	}
	if a.StatusOrigin != event.Observed {
		t.Errorf("final status origin %q, want observed", a.StatusOrigin)
	}
	n := 0
	for _, tc := range s.Tools {
		if tc.Agent == "" && tc.Name == "Agent" {
			n++
			if tc.Stage != 1 || tc.Seq >= a.Seq {
				t.Errorf("Agent tool stage %d seq %d, agent seq %d", tc.Stage, tc.Seq, a.Seq)
			}
		}
	}
	if n != 1 {
		t.Errorf("%d Agent tools, want 1", n)
	}
}

// P7: one turn, both children directly under the orchestrator, each ending
// completed.
func TestStagesP7(t *testing.T) {
	s, _ := replayState(t, "p7-codex-subagents.jsonl", codex.ProviderID, codex.NewMapper())
	if got := outcomes(s); got != "completed" {
		t.Fatalf("stages %q, want completed", got)
	}
	if !strings.HasPrefix(s.Stages[0].Prompt, "Spawn two subagents") {
		t.Errorf("stage prompt %q", s.Stages[0].Prompt)
	}
	labels := map[string]bool{}
	for _, k := range s.Order {
		a := s.Agents[k]
		labels[a.Label] = true
		if a.Parent != "" || a.Stage != 1 || a.Status != state.AgentCompleted {
			t.Errorf("child %s %q: parent %q, stage %d, status %q", k, a.Label, a.Parent, a.Stage, a.Status)
		}
	}
	if len(s.Order) != 2 || !labels["/root/list_files"] || !labels["/root/create_found"] {
		t.Errorf("children %v, want /root/list_files and /root/create_found", labels)
	}
}
