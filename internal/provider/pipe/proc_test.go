package pipe

import (
	"bufio"
	"context"
	"io"
	"slices"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/Evengelius/agent-command-center/internal/event"
)

// Plan 49's criterion 49.9 (design D8; audit finding 9): the lifecycle both
// adapters' sessions run on, tested on its own, in bubbles, where time is
// the bubble's and a deadlock fails the test.

// fakeProc is a CLI process on in-memory pipes: the test reads its stdin
// and writes its stdout and stderr. It exits when the test says so, and when
// terminated or killed if obey says it does.
type fakeProc struct {
	inR, outR, errR *io.PipeReader
	inW, outW, errW *io.PipeWriter
	done            chan struct{}
	once            sync.Once
	mu              sync.Mutex
	calls           []string
	obey            map[string]bool
	code            int
}

func newFakeProc() *fakeProc {
	p := &fakeProc{done: make(chan struct{}), obey: map[string]bool{"terminate": true, "kill": true}}
	p.inR, p.inW = io.Pipe()
	p.outR, p.outW = io.Pipe()
	p.errR, p.errW = io.Pipe()
	return p
}

func (p *fakeProc) Stdin() io.WriteCloser { return p.inW }
func (p *fakeProc) Stdout() io.Reader     { return p.outR }
func (p *fakeProc) Stderr() io.Reader     { return p.errR }
func (p *fakeProc) Pid() int              { return 0 }
func (p *fakeProc) Wait() (int, error)    { <-p.done; return p.code, nil }
func (p *fakeProc) Terminate() error      { return p.call("terminate") }
func (p *fakeProc) Kill() error           { return p.call("kill") }
func (p *fakeProc) Release()              { p.call("release") }

func (p *fakeProc) call(what string) error {
	p.mu.Lock()
	p.calls = append(p.calls, what)
	obey := p.obey[what]
	p.mu.Unlock()
	if obey {
		p.exit(true)
	}
	return nil
}

// exit ends the process; closed says its stdout and stderr close with it,
// else a process it started keeps them open.
func (p *fakeProc) exit(closed bool) {
	p.once.Do(func() {
		if closed {
			p.outW.Close()
			p.errW.Close()
		}
		p.inR.Close()
		close(p.done)
	})
}

func (p *fakeProc) called() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return slices.Clone(p.calls)
}

// started runs a Proc on p whose line mapping queues each line as a stderr
// event of its own, as a mapper queues its events, holding Mu.
func started(p *fakeProc) (*Proc, *sync.Mutex) {
	mu := &sync.Mutex{}
	var x *Proc
	x = New(Config{Name: "Fake CLI", Provider: "fake", Mu: mu, Line: func(l []byte) {
		mu.Lock()
		x.Queue().Push(event.Event{Provider: "fake", Kind: event.Stderr, Payload: event.Line{Text: "mapped " + string(l)}})
		mu.Unlock()
	}}, p)
	x.Run(context.Background(), []string{"fake", "--flag"}, `C:\proj`)
	return x, mu
}

// all reads every event until the channel closes.
func all(x *Proc) []event.Event {
	var out []event.Event
	for ev := range x.Events() {
		out = append(out, ev)
	}
	return out
}

func lineOf(ev event.Event) string {
	if l, ok := ev.Payload.(event.Line); ok {
		return l.Text
	}
	return ""
}

// TestProcEndsLast: session.spawned first, with the command and its folder;
// the lines mapped in their order, the stderr lines kept; session.ended last,
// once both readers are done, with the CLI's name and code in its reason.
func TestProcEndsLast(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		p := newFakeProc()
		p.code = 3
		x, _ := started(p)
		go func() {
			io.WriteString(p.outW, "one\n\n  \ntwo\n")
			io.WriteString(p.errW, "a warning\n")
			p.exit(true)
		}()
		evs := all(x)
		if sp, ok := evs[0].Payload.(event.Spawned); evs[0].Kind != event.SessionSpawned || !ok || strings.Join(sp.Argv, " ") != "fake --flag" || sp.Dir != `C:\proj` {
			t.Fatalf("first %+v, want session.spawned with the command", evs[0])
		}
		last := evs[len(evs)-1]
		if end, ok := last.Payload.(event.Ended); last.Kind != event.SessionEnded || !ok || end.Reason != "Fake CLI exited with code 3" || end.Code != 3 {
			t.Fatalf("last %+v, want session.ended with the code", last)
		}
		var lines []string
		for _, ev := range evs[1 : len(evs)-1] {
			lines = append(lines, lineOf(ev))
		}
		if i, j := slices.Index(lines, "mapped one"), slices.Index(lines, "mapped two"); i < 0 || j < i || !slices.Contains(lines, "a warning") || len(lines) != 3 {
			t.Errorf("events between %q, want the two lines mapped in order, the warning, nothing for the blank lines", lines)
		}
		if got := p.called(); !slices.Equal(got, []string{"release"}) {
			t.Errorf("calls %q, want the release alone", got)
		}
	})
}

// TestProcDrain: the CLI exits while a process it started keeps its stdout
// open; 2 s later the session ends without the readers, saying so.
func TestProcDrain(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		p := newFakeProc()
		x, _ := started(p)
		t0 := time.Now()
		p.exit(false)
		evs := all(x)
		if d := time.Since(t0); d != drainWait {
			t.Errorf("ended after %v, want %v", d, drainWait)
		}
		n := len(evs)
		problem, _ := evs[n-2].Payload.(event.Problem)
		if evs[n-1].Kind != event.SessionEnded || problem.Message != "Fake CLI's output stayed open after its exit: a process it started still holds it" {
			t.Fatalf("the last two events %+v %+v", evs[n-2], evs[n-1])
		}
		p.outW.Close()
		p.errW.Close()
	})
}

// TestProcCloseEscalates: Close closes stdin at once; a CLI that does not
// exit is terminated one step later, and killed one step after that.
func TestProcCloseEscalates(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		p := newFakeProc()
		p.obey["terminate"] = false
		x, _ := started(p)
		eof := make(chan time.Time, 1)
		go func() {
			_, _ = io.Copy(io.Discard, p.inR)
			eof <- time.Now()
		}()
		t0 := time.Now()
		if err := x.Close(context.Background()); err != nil {
			t.Fatal(err)
		}
		if at := <-eof; at != t0 {
			t.Errorf("stdin closed after %v, want at once", at.Sub(t0))
		}
		if d := time.Since(t0); d != 2*CloseStep {
			t.Errorf("exited after %v, want %v: stdin closed, then terminated, then killed", d, 2*CloseStep)
		}
		if got := p.called(); !slices.Equal(got[:2], []string{"terminate", "kill"}) {
			t.Errorf("calls %q, want terminate then kill", got)
		}
		evs := all(x)
		if end, _ := evs[len(evs)-1].Payload.(event.Ended); end.Reason != "Fake CLI closed (exit code 0)" {
			t.Errorf("the end %+v, want closed", end)
		}
		if err := x.Write(map[string]string{"a": "b"}, func([]byte) []*Item { return nil }); err == nil || err.Error() != "Fake CLI's input is closed" {
			t.Errorf("a write after the close: %v", err)
		}
	})
}

// TestProcWrite: a line goes out as JSON, HTML characters as they are, its
// held events leaving once it is written; one not written drops them and
// says so.
func TestProcWrite(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		p := newFakeProc()
		x, mu := started(p)
		in := bufio.NewReader(p.inR)
		hold := func(l []byte) []*Item {
			mu.Lock()
			defer mu.Unlock()
			return x.Queue().Hold([]event.Event{{Kind: event.Sent, Payload: event.Write{What: string(l)}}})
		}
		done := make(chan error, 1)
		go func() { done <- x.Write(map[string]string{"text": "<b>&</b>"}, hold) }()
		if got, _ := in.ReadString('\n'); got != `{"text":"<b>&</b>"}`+"\n" {
			t.Errorf("wrote %q", got)
		}
		if err := <-done; err != nil {
			t.Fatal(err)
		}
		if ev := <-x.Events(); ev.Kind != event.SessionSpawned {
			t.Fatalf("first %s", ev.Kind)
		}
		if ev := <-x.Events(); ev.Kind != event.Sent || ev.Payload.(event.Write).What != `{"text":"<b>&</b>"}` {
			t.Errorf("the held event %+v, want the line's", ev)
		}
		p.inR.Close() // the CLI reads no more
		err := x.Write(map[string]string{"text": "lost"}, hold)
		if err == nil || !strings.HasPrefix(err.Error(), "not delivered to Fake CLI: ") {
			t.Errorf("a write the CLI no longer reads: %v", err)
		}
		p.exit(true)
		for _, ev := range all(x) {
			if ev.Kind == event.Sent {
				t.Errorf("the lost line's event left: %+v", ev)
			}
		}
	})
}
