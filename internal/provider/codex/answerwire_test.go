package codex

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

// wireProc is a CLI process on in-memory pipes: the test reads its stdin
// and writes its stdout.
type wireProc struct {
	inR, outR, errR *io.PipeReader
	inW, outW, errW *io.PipeWriter
	done            chan struct{}
	once            sync.Once
}

func newWireProc() *wireProc {
	p := &wireProc{done: make(chan struct{})}
	p.inR, p.inW = io.Pipe()
	p.outR, p.outW = io.Pipe()
	p.errR, p.errW = io.Pipe()
	return p
}

func (p *wireProc) Stdin() io.WriteCloser { return p.inW }
func (p *wireProc) Stdout() io.Reader     { return p.outR }
func (p *wireProc) Stderr() io.Reader     { return p.errR }
func (p *wireProc) Pid() int              { return 0 }
func (p *wireProc) Wait() (int, error)    { <-p.done; return 0, nil }
func (p *wireProc) Terminate() error      { p.exit(); return nil }
func (p *wireProc) Kill() error           { p.exit(); return nil }
func (p *wireProc) Release()              { p.outR.Close(); p.errR.Close() }

func (p *wireProc) exit() {
	p.once.Do(func() {
		p.outW.Close()
		p.errW.Close()
		p.inR.Close()
		close(p.done)
	})
}

type wireSpawner struct{ p *wireProc }

func (s wireSpawner) Spawn(proc.Spec) (proc.Process, error) { return s.p, nil }

// TestAnswerLines is plan 49's criterion 49.11 (design D10): each decision's line, byte for byte as before the decisions were the dashboard's own, as
// Codex takes it.
func TestAnswerLines(t *testing.T) {
	change := `{"method":"item/fileChange/requestApproval","id":0,"params":{"threadId":"th1","turnId":"tu1","itemId":"it1","startedAtMs":1792093439876,"reason":null,"grantRoot":null}}`
	for _, c := range []struct {
		name, request, id string
		d                 event.Decision
		want              string
	}{
		{"allow", change, "0", event.Allow, `{"id":0,"result":{"decision":"accept"}}`},
		{"allow for the session", change, "0", event.AllowSession, `{"id":0,"result":{"decision":"acceptForSession"}}`},
		{"deny", change, "0", event.Deny, `{"id":0,"result":{"decision":"decline"}}`},
		{"cancel", change, "0", event.Cancel, `{"id":0,"result":{"decision":"cancel"}}`},
	} {
		t.Run(c.name, func(t *testing.T) {
			p := newWireProc()
			t.Cleanup(p.exit)
			// The session writes initialize as it starts: its stdin is read
			// from the start.
			lines := make(chan string, 16)
			go func() {
				in := bufio.NewReader(p.inR)
				for {
					l, err := in.ReadString('\n')
					if err != nil {
						close(lines)
						return
					}
					lines <- strings.TrimSuffix(l, "\n")
				}
			}()
			line := func() string {
				select {
				case l, ok := <-lines:
					if !ok {
						t.Fatal("stdin closed")
					}
					return l
				case <-time.After(10 * time.Second):
					t.Fatal("no line written")
				}
				return ""
			}
			sess, err := (&Provider{Spawner: wireSpawner{p}, CloseStep: 100 * time.Millisecond}).Start(context.Background(),
				provider.SessionContext{Executable: "codex", ProjectPath: t.TempDir()})
			if err != nil {
				t.Fatal(err)
			}
			line() // initialize
			go func() { _, _ = io.WriteString(p.outW, c.request+"\n") }()
			for asked := false; !asked; {
				select {
				case ev := <-sess.Events():
					asked = ev.Kind == event.ApprovalRequest || ev.Kind == event.Question
				case <-time.After(10 * time.Second):
					t.Fatal("no request event")
				}
			}
			answered := make(chan error, 1)
			go func() { answered <- sess.Answer(c.id, c.d) }()
			got := line()
			if err := <-answered; err != nil {
				t.Fatal(err)
			}
			if got != c.want {
				t.Errorf("wrote %s\nwant  %s", got, c.want)
			}
		})
	}
}
