package app

import (
	"strconv"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/Evengelius/agent-command-center/internal/event"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// TestSessionEventsInOrder is plan 49's criterion 49.1, the application's
// side (design D1): a session's events, more than one screen update drains
// at once (maxDrain), all reach the reducer through the listen, in their
// order, and the session's end after them. Every event goes through the
// session's channel, the handshake's too: an eventsMsg sent by the test
// (pushEvents) arms a listen of its own, and two listens would share the
// channel, which the program never does.
func TestSessionEventsInOrder(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, cli := startScript(t, "claude-code")
		seq, at := uint64(0), time.Date(2026, 10, 25, 9, 0, 0, 0, time.UTC)
		emit := func(k event.Kind, origin event.Origin, p any) {
			seq++
			at = at.Add(time.Millisecond)
			cli.sess.events <- event.Event{Seq: seq, At: at, Provider: cli.id, Kind: k, Origin: origin, Payload: p}
		}
		emit(event.SessionSpawned, event.Observed, event.Spawned{})
		emit(event.SessionReady, event.Observed, event.Ready{})
		emit(event.SessionStarted, event.Observed, event.Started{SessionID: "s1", Version: "2.1.300", PermissionMode: "default", ApprovalMode: event.ModeManual})
		emit(event.TurnStarted, event.Observed, nil)
		const n = 4 * maxDrain
		var want strings.Builder
		for i := range n {
			text := strconv.Itoa(i) + "\n"
			want.WriteString(text)
			emit(event.MessageDelta, event.AI, event.Delta{MessageID: "m1", Block: "text", Text: text})
		}
		emit(event.SessionEnded, event.Observed, event.Ended{Code: 0, HasCode: true, Reason: "Claude Code exited with code 0"})
		cli.sess.close.Do(func() { close(cli.sess.events) })
		h.settle()

		s := h.m.State()
		var got string
		for _, e := range s.Conversation {
			if e.Role == "assistant" && e.MessageID == "m1" {
				got = e.Text
			}
		}
		if w := want.String(); got != w {
			i := 0
			for i < min(len(got), len(w)) && got[i] == w[i] {
				i++
			}
			t.Fatalf("the answer's %d bytes differ from the %d streamed from byte %d: %q", len(got), len(w), i, got[i:min(i+40, len(got))])
		}
		if s.Session.Phase != state.Crashed || s.Session.Reason != "the CLI exited unexpectedly" {
			t.Fatalf("session %q (%q) after the end, want crashed by it", s.Session.Phase, s.Session.Reason)
		}
	})
}
