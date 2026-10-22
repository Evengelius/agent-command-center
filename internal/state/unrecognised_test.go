package state

import (
	"fmt"
	"testing"
	"time"

	"github.com/Evengelius/agent-command-center/internal/event"
)

// TestUnrecognisedKept (plan 25, D1; criterion 25.1): every unrecognised
// line is counted, and the last MaxUnrecognised are kept oldest first with
// their time, the mapper's type and reason, and the line itself.
func TestUnrecognisedKept(t *testing.T) {
	s := New()
	at := time.Date(2026, 10, 22, 15, 36, 0, 0, time.UTC)
	for i := 1; i <= MaxUnrecognised+2; i++ {
		s.Apply(event.Event{Seq: uint64(i), At: at.Add(time.Duration(i) * time.Second), Kind: event.Unrecognised, Origin: event.Observed,
			Raw: fmt.Sprintf(`{"type":"system","subtype":"s%d"}`, i), Payload: event.Unknown{Type: fmt.Sprintf("system/s%d", i), Reason: "r"}})
	}
	if s.Unrecognised != MaxUnrecognised+2 || len(s.UnrecognisedLast) != MaxUnrecognised {
		t.Fatalf("counted %d, kept %d; want %d and %d", s.Unrecognised, len(s.UnrecognisedLast), MaxUnrecognised+2, MaxUnrecognised)
	}
	first, last := s.UnrecognisedLast[0], s.UnrecognisedLast[MaxUnrecognised-1]
	if first.Type != "system/s3" || first.Reason != "r" || first.Line != `{"type":"system","subtype":"s3"}` || !first.At.Equal(at.Add(3*time.Second)) {
		t.Errorf("the oldest kept: %+v", first)
	}
	if last.Type != fmt.Sprintf("system/s%d", MaxUnrecognised+2) || !last.At.Equal(at.Add(time.Duration(MaxUnrecognised+2)*time.Second)) {
		t.Errorf("the newest kept: %+v", last)
	}
}
