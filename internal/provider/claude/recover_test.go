package claude

import (
	"context"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Evengelius/agent-command-center/internal/event"
	"github.com/Evengelius/agent-command-center/internal/provider/pipe"
)

// readerProc is a process whose stdout is a reader the test holds, its
// stdin taken whole and its stderr empty; it exits when the test says so.
type readerProc struct {
	out  io.Reader
	done chan struct{}
	once sync.Once
}

type takeAll struct{ io.Writer }

func (takeAll) Close() error { return nil }

func (p *readerProc) Stdin() io.WriteCloser { return takeAll{io.Discard} }
func (p *readerProc) Stdout() io.Reader     { return p.out }
func (p *readerProc) Stderr() io.Reader     { return strings.NewReader("") }
func (p *readerProc) Pid() int              { return 0 }
func (p *readerProc) Wait() (int, error)    { <-p.done; return 0, nil }
func (p *readerProc) Terminate() error      { p.exit(); return nil }
func (p *readerProc) Kill() error           { p.exit(); return nil }
func (p *readerProc) Release()              {}
func (p *readerProc) exit()                 { p.once.Do(func() { close(p.done) }) }

// TestReaderPanic is plan 48's criterion 48.9 (design D5; audit finding 6),
// on the shared lifecycle since plan 49 (design D8): the reader's mapping
// panicking, here through a nil mapper, the reader returns; it has queued a
// fatal error naming the panic, read its input to the end, and left both
// locks free. A function run apart that panics gives
// the same error.
func TestReaderPanic(t *testing.T) {
	rest := strings.Repeat("a line Claude Code wrote\n", 10_000) // more than the reader's 64 KiB buffer
	in := strings.NewReader(`{"type":"system","subtype":"status"}` + "\n" + rest)
	p := &readerProc{out: in, done: make(chan struct{})}
	t.Cleanup(p.exit)
	s := &Session{} // no mapper: its mapping panics
	s.cli = pipe.New(pipe.Config{Name: "Claude Code", Provider: ProviderID, Mu: &s.mu, Line: s.line}, p)
	s.cli.Run(context.Background(), []string{"claude"}, "")
	next := func() event.Event {
		t.Helper()
		select {
		case ev := <-s.Events():
			return ev
		case <-time.After(5 * time.Second):
			t.Fatal("no event")
		}
		return event.Event{}
	}
	if ev := next(); ev.Kind != event.SessionSpawned {
		t.Fatalf("first %s, want session.spawned", ev.Kind)
	}
	fatal := func(want string) {
		t.Helper()
		ev := next()
		p, ok := ev.Payload.(event.Problem)
		if ev.Kind != event.Error || !ok || !p.Fatal || !strings.HasPrefix(p.Message, want) {
			t.Errorf("event %s %+v, want a fatal error starting %q", ev.Kind, ev.Payload, want)
		}
	}
	fatal("reading Claude Code's output failed: panic: runtime error: invalid memory address or nil pointer dereference (at claude.(*Mapper).")
	s.cli.Apart("testing", func() { panic("boom") })
	fatal("testing failed: panic: boom (at claude.TestReaderPanic.func")
	if !s.mu.TryLock() {
		t.Error("the mapping's lock left held")
	} else {
		s.mu.Unlock()
	}
	wrote := make(chan error, 1)
	go func() {
		wrote <- s.cli.Write(map[string]string{"probe": "x"}, func([]byte) []*pipe.Item { return nil })
	}()
	select {
	case err := <-wrote:
		if err != nil {
			t.Errorf("a write after the panic: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Error("the writes' lock left held")
	}
	// The process ends: session.ended comes once the reader is done, having
	// read its input to the end, so Claude Code would never block on a full pipe.
	p.exit()
	for ev := next(); ev.Kind != event.SessionEnded; ev = next() {
	}
	if in.Len() != 0 {
		t.Errorf("%d bytes left unread: Claude Code would block on a full pipe", in.Len())
	}
}
