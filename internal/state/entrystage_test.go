package state

import (
	"fmt"
	"strings"
	"testing"

	"github.com/Evengelius/agent-command-center/internal/event"
)

// TestEntryStage is plan 19's criterion 3.1: each entry of zone B records
// the turn it belongs to (Entry.Stage), which zone B's RESULT label reads.
func TestEntryStage(t *testing.T) {
	b := newBench(t).ready()
	b.ev(event.StatusChanged, "", event.Status{Status: "notice", Detail: "before any turn"})

	// The owner's prompt opens turn 1: the prompt, its reply and its call
	// belong to it, and so does an entry after it ends.
	b.do(Send{Text: "hi"})
	b.ev(event.UserMessage, "", event.User{Text: "hi"})
	b.ev(event.TurnStarted, "", nil)
	b.ev(event.MessageCompleted, "", event.Completed{MessageID: "m1", Text: "I'll look."})
	b.ev(event.ToolStarted, "", event.Tool{ToolID: "t1", Name: "Read"})
	b.ev(event.ToolCompleted, "", event.Tool{ToolID: "t1", Name: "Read"})
	b.turnEnd("", "completed")
	b.ev(event.StatusChanged, "", event.Status{Status: "notice", Detail: "after turn 1"})

	// A turn the CLI opens by itself is turn 2; its message belongs to it.
	b.ev(event.TurnStarted, "", nil)
	b.ev(event.MessageCompleted, "", event.Completed{MessageID: "m2", Text: "The agent is done."})
	b.turnEnd("", "completed")

	// In a replay the prompt comes with the turn idle: it opens turn 3.
	b.ev(event.UserMessage, "", event.User{Text: "again"})
	b.ev(event.MessageCompleted, "", event.Completed{MessageID: "m3", Text: "ok"})
	b.turnEnd("", "failed")

	var got []string
	for _, e := range b.s.Conversation {
		got = append(got, fmt.Sprintf("%s:%d", e.Role, e.Stage))
	}
	want := "notice:0 user:1 assistant:1 tool:1 notice:1 assistant:2 user:3 assistant:3"
	b.eq("3.1 the entries' turns", strings.Join(got, " "), want)
	b.eq("3.1 three turns", len(b.s.Stages), 3)
	b.eq("3.1 turn 2 opened by the CLI", b.s.Stages[1].ByCLI, true)
}
