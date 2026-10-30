package state

import (
	"testing"

	"github.com/Evengelius/agent-command-center/internal/event"
)

// TestRecordSession is criterion 69.2 for the state (plan 69, design D2):
// a turn the owner sent that ends, completed, failed or interrupted, in a
// session whose ID the CLI reported, records it with the turn end's time; a
// turn the CLI opened, a mock run, or a session with no ID yet records
// nothing.
func TestRecordSession(t *testing.T) {
	recorded := func(b *bench) (Command, bool) {
		for _, c := range b.cmds {
			if c.Kind == CmdRecordSession {
				return c, true
			}
		}
		return Command{}, false
	}
	for _, outcome := range []string{"completed", "failed", "interrupted"} {
		b := newBench(t).running()
		b.ev(event.SessionStarted, "", event.Started{SessionID: "s-1", Cwd: project}) // Claude Code's comes with the turn
		if outcome == "interrupted" {
			b.do(Interrupt{})
		}
		b.turnEnd("", outcome)
		at := b.s.Log[len(b.s.Log)-1].At
		if c, ok := recorded(b); !ok || c.ID != "s-1" || !c.At.Equal(at) {
			t.Errorf("%s: %+v, %t", outcome, c, ok)
		}
	}

	b := newBench(t).ready()
	b.ev(event.SessionStarted, "", event.Started{SessionID: "s-1", Cwd: project})
	b.ev(event.MessageMarker, "", event.Marker{MessageID: "m1", Name: "message_start"}) // the CLI opens a turn
	b.turnEnd("", "completed")
	if c, ok := recorded(b); ok {
		t.Errorf("a turn the CLI opened: %+v", c)
	}

	n := newBench(t).running() // no session.started: no ID
	n.turnEnd("", "completed")
	if c, ok := recorded(n); ok {
		t.Errorf("no session ID: %+v", c)
	}

	m := newBench(t)
	m.do(Launch{Mock: "demo.jsonl"})
	m.ev(event.SessionSpawned, "", event.Spawned{})
	m.ev(event.SessionReady, "", event.Ready{})
	m.ev(event.SessionStarted, "", event.Started{SessionID: "s-mock"})
	m.do(Send{Text: "hi"})
	m.ev(event.UserMessage, "", event.User{Text: "hi"})
	m.ev(event.TurnStarted, "", nil)
	m.turnEnd("", "completed")
	if c, ok := recorded(m); ok {
		t.Errorf("a mock run: %+v", c)
	}
}
