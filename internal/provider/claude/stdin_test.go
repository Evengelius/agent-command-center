package claude

import (
	"bufio"
	"context"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Evengelius/agent-command-center/internal/event"
	"github.com/Evengelius/agent-command-center/internal/proc"
	"github.com/Evengelius/agent-command-center/internal/provider"
)

// pipeProc is a Claude Code process on in-memory pipes (plan 48, design D3):
// a write to its stdin waits until the test reads it, as when the CLI has
// stopped reading its stdin, and its stdout is what the test writes.
type pipeProc struct {
	inR, outR, errR *io.PipeReader
	inW, outW, errW *io.PipeWriter
	done            chan struct{}
	once            sync.Once
}

func newPipeProc() *pipeProc {
	p := &pipeProc{done: make(chan struct{})}
	p.inR, p.inW = io.Pipe()
	p.outR, p.outW = io.Pipe()
	p.errR, p.errW = io.Pipe()
	return p
}

func (p *pipeProc) Stdin() io.WriteCloser { return p.inW }
func (p *pipeProc) Stdout() io.Reader     { return p.outR }
func (p *pipeProc) Stderr() io.Reader     { return p.errR }
func (p *pipeProc) Pid() int              { return 0 }
func (p *pipeProc) Wait() (int, error)    { <-p.done; return 0, nil }
func (p *pipeProc) Terminate() error      { p.exit(); return nil }
func (p *pipeProc) Kill() error           { p.exit(); return nil }
func (p *pipeProc) Release()              { p.outR.Close(); p.errR.Close() }

// exit ends the process: its stdout and stderr close, its stdin is read no
// more.
func (p *pipeProc) exit() {
	p.once.Do(func() {
		p.outW.Close()
		p.errW.Close()
		p.inR.Close()
		close(p.done)
	})
}

type pipeSpawner struct{ p *pipeProc }

func (s pipeSpawner) Spawn(proc.Spec) (proc.Process, error) { return s.p, nil }

// TestReaderNeverWaitsOnStdin is plan 48's criterion 48.6 (design D3; audit
// finding 9): Claude Code stops reading its stdin after initialize, and the
// reader goes on reading its stdout while the handshake's two reads wait
// apart; once stdin is read again they arrive, get_settings then "/usage",
// and session.ready comes before both.
func TestReaderNeverWaitsOnStdin(t *testing.T) {
	p := newPipeProc()
	t.Cleanup(p.exit)
	in := bufio.NewReader(p.inR)
	readIn := func(what string) string {
		t.Helper()
		l, err := in.ReadString('\n')
		if err != nil {
			t.Fatalf("%s: %v", what, err)
		}
		return l
	}
	started := make(chan provider.Session, 1)
	go func() {
		s, err := (&Provider{Spawner: pipeSpawner{p}, CloseStep: 100 * time.Millisecond}).Start(context.Background(),
			provider.SessionContext{Executable: "claude", ProjectPath: t.TempDir()})
		if err != nil {
			t.Error(err)
		}
		started <- s
	}()
	if l := readIn("initialize"); !strings.Contains(l, `"subtype":"initialize"`) {
		t.Fatalf("first line %q", l)
	}
	sess := <-started
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = sess.Close(ctx)
	})
	// Lines written to stdout must each be read while nothing reads stdin.
	writeOut := func(what string, lines ...string) {
		t.Helper()
		for i, l := range lines {
			done := make(chan error, 1)
			go func() { _, err := io.WriteString(p.outW, l+"\n"); done <- err }()
			select {
			case err := <-done:
				if err != nil {
					t.Fatalf("%s, stdout line %d: %v", what, i+1, err)
				}
			case <-time.After(2 * time.Second):
				t.Fatalf("%s, stdout line %d not read within 2 s: the reader waits on stdin", what, i+1)
			}
		}
	}
	writeOut("the handshake", `{"type":"control_response","response":{"subtype":"success","request_id":"req_1","response":{}}}`,
		`{"type":"keep_alive"}`, `{"type":"keep_alive"}`, `{"type":"keep_alive"}`)
	// Stdin read again: the settings read, then "/usage".
	if l := readIn("the settings read"); !strings.Contains(l, `"subtype":"get_settings"`) {
		t.Errorf("second line %q, want get_settings", l)
	}
	if l := readIn("the usage read"); !strings.Contains(l, `/usage`) {
		t.Errorf("third line %q, want /usage", l)
	}
	// The events: initialize's write, session.ready, then the two reads'
	// writes.
	var seen []string
	timeout := time.After(5 * time.Second)
	for len(seen) < 4 {
		select {
		case ev := <-sess.Events():
			if w, ok := ev.Payload.(event.Write); ok && ev.Kind == event.Sent {
				seen = append(seen, w.What)
			} else if ev.Kind == event.SessionReady {
				seen = append(seen, "ready")
			}
		case <-timeout:
			t.Fatalf("events %v: want initialize, ready, get_settings, usage", seen)
		}
	}
	if got := strings.Join(seen, " "); got != "initialize ready get_settings usage" {
		t.Errorf("events %q, want initialize, ready, get_settings, usage", got)
	}

	// A change answered: the settings read that follows waits apart too.
	changed := make(chan error, 1)
	go func() { changed <- sess.SetModelSettings(provider.ModelRequest{Model: "claude-opus-4-6"}) }()
	if l := readIn("the change"); !strings.Contains(l, `"subtype":"set_model"`) {
		t.Fatalf("fourth line %q, want set_model", l)
	}
	if err := <-changed; err != nil {
		t.Fatal(err)
	}
	writeOut("the change's answer", `{"type":"control_response","response":{"subtype":"success","request_id":"req_2","response":{}}}`,
		`{"type":"keep_alive"}`, `{"type":"keep_alive"}`)
	if l := readIn("the read after the change"); !strings.Contains(l, `"subtype":"get_settings"`) {
		t.Errorf("fifth line %q, want get_settings", l)
	}
}
