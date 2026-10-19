package state

import (
	"strings"
	"testing"

	"github.com/Evengelius/agent-command-center/internal/event"
)

// Plan 18, item 1: the agents in play, which size zone C (brief amendment
// 14); none leaves it a strip.
func TestAgentsInPlay(t *testing.T) {
	// 1.1 No agent: none. The owner's turn deploys two agents: both, also
	// once they are done.
	b := newBench(t)
	b.running()
	b.eq("1.1 no agent", b.s.AgentsInPlay(), 0)
	b.ev(event.AgentStarted, "", event.Agent{Key: "t1", Type: "subagent"})
	b.ev(event.AgentStarted, "", event.Agent{Key: "t1b", Type: "subagent"})
	b.eq("1.1 two agents running", b.s.AgentsInPlay(), 2)
	b.ev(event.AgentCompleted, "", event.Agent{Key: "t1"})
	b.ev(event.AgentCompleted, "", event.Agent{Key: "t1b"})
	b.turnEnd("", "completed")
	b.eq("1.1 both done", b.s.AgentsInPlay(), 2)

	// 1.2 The next prompt with every agent done: none; an agent of that turn
	// counts alone.
	b.do(Send{Text: "next"})
	b.eq("1.2 next prompt", b.s.AgentsInPlay(), 0)
	b.ev(event.UserMessage, "", event.User{Text: "next"})
	b.ev(event.TurnStarted, "", nil)
	b.eq("1.2 its turn running", b.s.AgentsInPlay(), 0)
	b.ev(event.AgentStarted, "", event.Agent{Key: "t2", Type: "subagent"})
	b.eq("1.2 an agent of that turn", b.s.AgentsInPlay(), 1)
	b.ev(event.AgentCompleted, "", event.Agent{Key: "t2"})
	b.turnEnd("", "completed")

	// 1.3 A turn the CLI opens by itself changes nothing, with agents...
	b.ev(event.TurnStarted, "", nil)
	b.eq("1.3 CLI turn opened", b.s.Turn.ByCLI, true)
	b.eq("1.3 CLI turn, one before", b.s.AgentsInPlay(), 1)
	b.turnEnd("", "completed")
	// ...or without.
	b.do(Send{Text: "no agents"})
	b.ev(event.UserMessage, "", event.User{Text: "no agents"})
	b.ev(event.TurnStarted, "", nil)
	b.turnEnd("", "completed")
	b.eq("1.3 an owner's turn without agents", b.s.AgentsInPlay(), 0)
	b.ev(event.TurnStarted, "", nil)
	b.eq("1.3 CLI turn, none before", b.s.AgentsInPlay(), 0)
	b.turnEnd("", "completed")

	// 1.4 An agent still running when the owner submits a prompt stays in
	// play, also once it ends in the new turn; the next prompt resets it.
	b.do(Send{Text: "background"})
	b.ev(event.UserMessage, "", event.User{Text: "background"})
	b.ev(event.TurnStarted, "", nil)
	b.ev(event.AgentStarted, "", event.Agent{Key: "bg", Type: "subagent", Backgrounded: true})
	b.turnEnd("", "completed")
	b.do(Send{Text: "while it runs"})
	b.eq("1.4 prompt while it runs", b.s.AgentsInPlay(), 1)
	b.ev(event.UserMessage, "", event.User{Text: "while it runs"})
	b.ev(event.TurnStarted, "", nil)
	b.ev(event.AgentStarted, "", event.Agent{Key: "new", Type: "subagent"})
	b.eq("1.4 with an agent of the new turn", b.s.AgentsInPlay(), 2)
	b.ev(event.AgentCompleted, "", event.Agent{Key: "bg"})
	b.ev(event.AgentCompleted, "", event.Agent{Key: "new"})
	b.eq("1.4 both end in the new turn", b.s.AgentsInPlay(), 2)
	b.eq("1.4 the agents in play, the earlier turns' cleared", strings.Join(b.s.InPlay(), ","), "bg,new")
	b.turnEnd("", "completed")
	b.do(Send{Text: "after"})
	b.eq("1.4 the next prompt", b.s.AgentsInPlay(), 0)

	// 1.5 Before any prompt, as a replay can start, every agent counts.
	r := newBench(t)
	r.ready()
	r.ev(event.AgentStarted, "", event.Agent{Key: "early", Type: "subagent"})
	r.ev(event.AgentCompleted, "", event.Agent{Key: "early"})
	r.eq("1.5 before any prompt", r.s.AgentsInPlay(), 1)
}
