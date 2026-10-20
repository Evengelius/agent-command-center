package render

import (
	"strings"
	"testing"
	"time"

	"github.com/Evengelius/agent-command-center/internal/event"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// Plan 21, item 1: zone D's session time counts from the first event the
// state applied, not from the oldest the log still keeps (state.MaxLog).

// lines applies n stderr lines one second apart from t0, a stand-in for the
// text deltas of a long session.
func lines(s *state.State, t0 time.Time, from, n int) {
	for i := from; i < from+n; i++ {
		s.Apply(event.Event{Seq: uint64(i + 1), At: t0.Add(time.Duration(i) * time.Second), Kind: event.Stderr,
			Origin: event.Observed, Payload: event.Line{Text: "x"}})
	}
}

func TestSessionTimeKeepsFirstEvent(t *testing.T) {
	t0 := time.Date(2026, 10, 20, 9, 0, 0, 0, time.UTC)

	// 1.1 Past the log's bound: 6,000 events, the frame at t0 + 6,000 s.
	s := state.New()
	lines(s, t0, 0, 6000)
	if len(s.Log) != state.MaxLog {
		t.Fatalf("log keeps %d events, want %d", len(s.Log), state.MaxLog)
	}
	if d := zoneDText(s, t0.Add(6000*time.Second)); !strings.Contains(d, "time 1h 40m session") {
		t.Errorf("6,000 events over 1h 40m: zone D has no \"time 1h 40m session\":\n%s", d)
	}

	// 1.2 Within the bound, the value counted from the log's first event; no
	// event, N/A.
	s = state.New()
	lines(s, t0, 0, state.MaxLog-1)
	now := t0.Add(time.Duration(state.MaxLog-1) * time.Second)
	if got, want := sessionTime(s, UI{Now: now}), elapsed(s.Log[0].At, time.Time{}, now)+" session"; got != want || want != "33m 19s session" {
		t.Errorf("1,999 events: %q, want %q from the log's first event (33m 19s session)", got, want)
	}
	if got := sessionTime(state.New(), UI{Now: now}); !strings.HasPrefix(got, "N/A (") {
		t.Errorf("no event: %q, want N/A with its reason", got)
	}

	// 1.3 A resume after a crash, past the bound: still from the first event.
	s = state.New()
	s.Context = state.Context{Provider: "claude-code", Project: `C:\work\demo`}
	s.Do(state.ValidationOK{Provider: "claude-code", ProviderName: "Claude Code", Project: `C:\work\demo`})
	at := func(i int) time.Time { return t0.Add(time.Duration(i) * time.Second) }
	s.Apply(event.Event{Seq: 1, At: at(0), Kind: event.SessionSpawned, Origin: event.Observed, Payload: event.Spawned{}})
	s.Apply(event.Event{Seq: 2, At: at(1), Kind: event.SessionReady, Origin: event.Observed, Payload: event.Ready{}})
	s.Apply(event.Event{Seq: 3, At: at(2), Kind: event.SessionStarted, Origin: event.Observed,
		Payload: event.Started{SessionID: "sess-1", Cwd: `C:\work\demo`}})
	lines(s, t0, 3, 2500)
	s.Apply(event.Event{Seq: 2504, At: at(2503), Kind: event.SessionEnded, Origin: event.Observed, Payload: event.Ended{Code: 1, HasCode: true}})
	if s.Session.Phase != state.Crashed {
		t.Fatalf("session %s after an unexpected exit, want Crashed", s.Session.Phase)
	}
	s.Do(state.Resume{})
	s.Apply(event.Event{Seq: 2505, At: at(2504), Kind: event.SessionSpawned, Origin: event.Observed, Payload: event.Spawned{}})
	lines(s, t0, 2505, 1000)
	if got := sessionTime(s, UI{Now: at(3600)}); got != "1h 00m session" {
		t.Errorf("after a resume, past the bound: %q, want \"1h 00m session\" from the first event", got)
	}
}
