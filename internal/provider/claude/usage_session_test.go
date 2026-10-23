package claude

import (
	"testing"
	"time"

	"github.com/Evengelius/agent-command-center/internal/event"
	"github.com/Evengelius/agent-command-center/internal/provider"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// TestUsageReadAroundTurns (plan 24, amendment 23): the session reads the
// usage report as soon as the handshake is answered and after each turn,
// in order with the owner's messages; neither read is a turn or an entry.
func TestUsageReadAroundTurns(t *testing.T) {
	r, _ := start(t, "usage", nil)
	r.wait(event.SessionReady)
	read := func(ev event.Event) bool {
		st, ok := ev.Payload.(event.Status)
		return ok && st.Detail == "usage report: read"
	}
	if !r.until(read, 10*time.Second) {
		t.Fatalf("the first read never ended (%d events)", len(r.evs))
	}
	if v := r.st.Metrics["quota.5h"]; !v.Present || v.V != 6 {
		t.Errorf("quota.5h after the first read: %+v", v)
	}
	if v := r.st.Metrics["quota.7d.fable"]; !v.Present || v.V != 11 || !v.Documented {
		t.Errorf("quota.7d.fable after the first read: %+v", v)
	}
	if len(r.st.Conversation) != 0 || len(r.st.Stages) != 0 || r.st.Turn.Phase != state.Idle {
		t.Errorf("the read left %d entries, %d stages, turn %s", len(r.st.Conversation), len(r.st.Stages), r.st.Turn.Phase)
	}
	// The owner's turn.
	if err := r.sess.Send(provider.Message{Text: "hello"}); err != nil {
		t.Fatal(err)
	}
	r.st.Do(state.Send{Text: "hello"}) // as the application does, before the write
	r.wait(event.TurnCompleted)
	if len(r.st.Stages) != 1 || r.st.Stages[0].Open || r.st.Turn.Phase != state.Idle {
		t.Fatalf("after the turn: %d stages, turn %s", len(r.st.Stages), r.st.Turn.Phase)
	}
	entries := len(r.st.Conversation)
	// The refresh: the context, then the report again.
	if err := r.sess.Refresh(); err != nil {
		t.Fatal(err)
	}
	if !r.until(read, 10*time.Second) {
		t.Fatalf("the refresh's read never ended (%d events)", len(r.evs))
	}
	if v := r.st.Metrics["quota.5h"]; v.V != 7 {
		t.Errorf("quota.5h after the refresh: %+v", v)
	}
	if v := r.st.Metrics["context.used"]; !v.Present || v.V != 15 {
		t.Errorf("context.used after the refresh: %+v", v)
	}
	if len(r.st.Stages) != 1 || len(r.st.Conversation) != entries || r.st.Turn.Phase != state.Idle || len(r.st.Errors) != 0 {
		t.Errorf("the refresh's read left %d stages, %d entries (had %d), turn %s, %d errors", len(r.st.Stages), len(r.st.Conversation), entries, r.st.Turn.Phase, len(r.st.Errors))
	}
}
