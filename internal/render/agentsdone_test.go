package render

import (
	"fmt"
	"testing"

	"github.com/Evengelius/agent-command-center/internal/provider/claude"
	"github.com/Evengelius/agent-command-center/internal/provider/codex"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// agentsDoneByWalk is plan 20's agentsDone, the rule itself: every agent
// whose stage is from to last completed. It walks every agent, as the code
// did before plan 21's fix.
func agentsDoneByWalk(s *state.State, from, last int) bool {
	for _, k := range s.Order {
		if a := s.Agents[k]; a != nil && a.Stage >= from && a.Stage <= last && a.Status != state.AgentCompleted {
			return false
		}
	}
	return true
}

// TestAgentsDoneCounts is plan 21's fix of the long session's frame time:
// agentsDone from undoneByStage, counted once a frame, gives the rule's
// answer for every pair of turns, in recorded sessions and in one with an
// agent of each status in each turn, before the first turn included.
func TestAgentsDoneCounts(t *testing.T) {
	synthetic := state.New()
	statuses := []state.AgentStatus{state.AgentRunning, state.AgentWaiting, state.AgentCompleted, state.AgentFailed,
		state.AgentStopped, state.AgentEnded, state.AgentAnnounced}
	for n := 1; n <= 6; n++ {
		synthetic.Stages = append(synthetic.Stages, state.Stage{N: n})
	}
	i := 0
	for stage := 0; stage <= 6; stage++ {
		for j, st := range statuses {
			switch {
			case stage == 3: // a turn whose agents all completed
				st = state.AgentCompleted
			case stage == 5 && j > 0: // a turn with one agent not completed
				st = state.AgentCompleted
			case (stage+j)%3 == 0:
				st = state.AgentCompleted
			}
			k := fmt.Sprintf("a%02d", i)
			synthetic.Agents[k] = &state.Agent{Key: k, Stage: stage, Status: st}
			synthetic.Order = append(synthetic.Order, k)
			i++
		}
	}
	for _, c := range []struct {
		name string
		s    *state.State
	}{
		{"P3b", probeState(t, "p3b-claude-subagent-approval.jsonl", claude.ProviderID, claude.NewMapper())},
		{"P7", probeState(t, "p7-codex-subagents.jsonl", codex.ProviderID, codex.NewMapper())},
		{"synthetic", synthetic},
	} {
		undone, n := undoneByStage(c.s), len(c.s.Stages)
		checked, done := 0, 0
		for from := 0; from <= n+1; from++ {
			for last := -1; last <= n+1; last++ {
				want := agentsDoneByWalk(c.s, from, last)
				if got := agentsDone(undone, from, last); got != want {
					t.Errorf("%s: turns %d to %d: %v, the rule says %v", c.name, from, last, got, want)
				}
				checked++
				if want {
					done++
				}
			}
		}
		// The recordings' agents all completed; the synthetic case holds
		// both answers.
		if c.name == "synthetic" && (done == 0 || done == checked) {
			t.Errorf("%s: %d of %d pairs done: the case tests one answer only", c.name, done, checked)
		}
		t.Logf("%s: %d pairs, %d done", c.name, checked, done)
	}
}
