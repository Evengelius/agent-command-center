package claude

import (
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Evengelius/agent-command-center/internal/event"
	"github.com/Evengelius/agent-command-center/internal/provider/pipe"
)

// TestReaderPanic is plan 48's criterion 48.9 (design D5; audit finding 6):
// the reader's mapping panicking, here through a nil mapper, the reader
// returns; it has queued a fatal error naming the panic, read its input to
// the end, and left both mutexes free. A function run apart that panics gives
// the same error.
func TestReaderPanic(t *testing.T) {
	s := &Session{q: pipe.NewQueue(time.Now), inputs: map[string]pendingReq{}}
	go s.q.Forward()
	t.Cleanup(s.q.Close)
	rest := strings.Repeat("a line Claude Code wrote\n", 10_000) // more than the reader's 64 KiB buffer
	in := strings.NewReader(`{"type":"system","subtype":"status"}` + "\n" + rest)
	s.read(in)
	if in.Len() != 0 {
		t.Errorf("%d bytes left unread: Claude Code would block on a full pipe", in.Len())
	}
	for name, mu := range map[string]*sync.Mutex{"the mapping's": &s.mu, "the writes'": &s.wmu} {
		if !mu.TryLock() {
			t.Errorf("%s lock left held", name)
		} else {
			mu.Unlock()
		}
	}
	fatal := func(want string) {
		t.Helper()
		select {
		case ev := <-s.q.Out():
			p, ok := ev.Payload.(event.Problem)
			if ev.Kind != event.Error || !ok || !p.Fatal || !strings.HasPrefix(p.Message, want) {
				t.Errorf("event %s %+v, want a fatal error starting %q", ev.Kind, ev.Payload, want)
			}
		case <-time.After(5 * time.Second):
			t.Errorf("no fatal error starting %q", want)
		}
	}
	fatal("reading Claude Code's output failed: panic: runtime error: invalid memory address or nil pointer dereference (at claude.(*Mapper).")
	s.apart("testing", func() { panic("boom") })
	fatal("testing failed: panic: boom (at claude.TestReaderPanic.func")
}
