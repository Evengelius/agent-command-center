//go:build live && windows

package codex

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/Evengelius/agent-command-center/internal/event"
	"github.com/Evengelius/agent-command-center/internal/proc"
	"github.com/Evengelius/agent-command-center/internal/provider"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// The live gate uses the owner's quota: it runs only with the live tag, in
// the project named by ACC_LIVE_PROJECT, an empty folder outside the
// repository, with the configuration check before and after (proposal,
// step 5, item 4, run L3):
//
//	go test -tags live -count=1 -v -run TestLiveGate ./internal/provider/codex
//
// ACC_LIVE_MODEL, when set, is the model of every thread the test starts;
// otherwise Codex's default. ACC_LIVE_LOG, when set, receives every event's
// kind and raw line.

const codeWord = "ORCHID-41"

// liveTurn is long enough for a short turn through codex app-server.
const liveTurn = 120 * time.Second

// tee is the app-server's stdin, keeping every line the adapter writes so
// the gate can check none is forbidden (CX-10: no cwd, no configuration
// write).
type tee struct {
	io.WriteCloser
	mu    *sync.Mutex
	lines *[]string
	part  []byte
}

func (w *tee) Write(b []byte) (int, error) {
	w.mu.Lock()
	w.part = append(w.part, b...)
	for {
		i := strings.IndexByte(string(w.part), '\n')
		if i < 0 {
			break
		}
		*w.lines = append(*w.lines, string(w.part[:i]))
		w.part = w.part[i+1:]
	}
	w.mu.Unlock()
	return w.WriteCloser.Write(b)
}

type teed struct {
	proc.Process
	in *tee
}

func (p teed) Stdin() io.WriteCloser { return p.in }

// spawns is the real spawner, keeping each process and what was written to
// it.
type spawns struct {
	mu    sync.Mutex
	procs []proc.Process
	lines []string
}

func (s *spawns) Spawn(sp proc.Spec) (proc.Process, error) {
	p, err := proc.Exec{}.Spawn(sp)
	if err != nil {
		return p, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.procs = append(s.procs, p)
	return teed{p, &tee{WriteCloser: p.Stdin(), mu: &s.mu, lines: &s.lines}}, nil
}

func (s *spawns) last() proc.Process {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.procs[len(s.procs)-1]
}

// written checks every line written so far against the forbidden requests.
func (s *spawns) written(t *testing.T) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, l := range s.lines {
		if why := forbidden(l); why != "" {
			t.Errorf("forbidden write (%s): %s", why, l)
		}
	}
	return len(s.lines)
}

// descendants lists every process started, directly or not, by pid.
func descendants(t *testing.T, pid int) []int {
	t.Helper()
	h, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer windows.CloseHandle(h)
	children := map[uint32][]uint32{}
	var e windows.ProcessEntry32
	e.Size = uint32(unsafe.Sizeof(e))
	for err = windows.Process32First(h, &e); err == nil; err = windows.Process32Next(h, &e) {
		children[e.ParentProcessID] = append(children[e.ParentProcessID], e.ProcessID)
	}
	var out []int
	queue := []uint32{uint32(pid)}
	for len(queue) > 0 {
		p := queue[0]
		queue = queue[1:]
		for _, c := range children[p] {
			out = append(out, int(c))
			queue = append(queue, c)
		}
	}
	return out
}

// next reads and applies one event; false on timeout or when the channel
// closed.
func (r *run) next(d time.Duration) (event.Event, bool) {
	n := len(r.evs)
	r.until(func(event.Event) bool { return true }, d)
	if len(r.evs) > n {
		return r.evs[n], true
	}
	return event.Event{}, false
}

type live struct {
	t   *testing.T
	p   *Provider
	sp  *spawns
	exe string
	dir string
	log *os.File
}

// start launches one app-server; resume is "" for a new thread. The
// handshake must end within 60 s (ASM-21).
func (l *live) start(ctx context.Context, resume string) *run {
	l.t.Helper()
	st := state.New()
	st.Context = state.Context{Provider: ProviderID, Project: l.dir}
	st.Startup.Phase = state.Starting
	t0 := time.Now()
	sess, err := l.p.Start(ctx, provider.SessionContext{Executable: l.exe, ProjectPath: l.dir, ResumeID: resume})
	if err != nil {
		l.t.Fatal(err)
	}
	r := &run{t: l.t, sess: sess, st: st}
	if !r.until(func(ev event.Event) bool { return ev.Kind == event.SessionReady }, 60*time.Second) || r.done {
		l.t.Fatalf("no handshake within 60 s (%d events)", len(r.evs))
	}
	// The thread/start or thread/resume response gives session.ready, then
	// session.started from the same line (mapper.go, threadStart).
	if !r.until(func(ev event.Event) bool { return ev.Kind == event.SessionStarted }, 5*time.Second) || r.done {
		l.t.Fatalf("no session.started after the handshake (%d events)", len(r.evs))
	}
	l.t.Logf("handshake in %v (pid %d)", time.Since(t0).Round(time.Millisecond), l.sp.last().Pid())
	return r
}

// dump writes the run's events to ACC_LIVE_LOG and fails on error events.
func (l *live) dump(r *run, name string) {
	for _, ev := range r.evs {
		if l.log != nil {
			fmt.Fprintf(l.log, "%s\t%d\t%s\t%s\n", name, ev.Seq, ev.Kind, strings.TrimSpace(ev.Raw))
		}
		switch ev.Kind {
		case event.Error:
			l.t.Errorf("%s: error event: %s", name, ev.Payload.(event.Problem).Message)
		case event.Stderr:
			l.t.Logf("%s: stderr: %s", name, ev.Payload.(event.Line).Text)
		case event.Unrecognised:
			l.t.Logf("%s: unrecognised: %s", name, strings.TrimSpace(ev.Raw))
		}
	}
}

// turn sends text and reads until the main thread's turn completes; on is
// called for each event and may answer approvals or interrupt. It returns
// the turn's end and its events.
func (l *live) turn(r *run, text string, on func(event.Event)) (event.TurnEnd, []event.Event) {
	l.t.Helper()
	r.send(text)
	from := len(r.evs)
	deadline := time.Now().Add(liveTurn)
	for time.Now().Before(deadline) {
		ev, ok := r.next(time.Second)
		if r.done {
			l.t.Fatalf("the session ended during the turn %q", text)
		}
		if !ok {
			continue
		}
		if on != nil {
			on(ev)
		}
		if ev.Kind == event.TurnCompleted && ev.Agent == "" {
			return ev.Payload.(event.TurnEnd), r.evs[from:]
		}
	}
	l.t.Fatalf("the turn %q did not complete within %v", text, liveTurn)
	return event.TurnEnd{}, nil
}

// do writes the commands an action gives, as the application does.
func (l *live) do(r *run, cmds []state.Command) {
	l.t.Helper()
	for _, c := range cmds {
		var err error
		switch c.Kind {
		case state.CmdAnswer:
			err = r.sess.Answer(c.ID, c.Decision)
		case state.CmdInterrupt:
			err = r.sess.Interrupt()
		default:
			continue
		}
		if err != nil {
			l.t.Error(err)
		}
	}
}

// approve answers the first approval the orchestrator asks for in the turn
// with act, and records when; any other request is declined.
func (l *live) approve(r *run, act func(event.Approval) []state.Command, at *time.Time) func(event.Event) {
	return func(ev event.Event) {
		if ev.Kind != event.ApprovalRequest {
			return
		}
		a := ev.Payload.(event.Approval)
		l.t.Logf("approval %s (agent %q): %s", a.Tool, ev.Agent, a.Input)
		if !at.IsZero() || ev.Agent != "" {
			l.do(r, []state.Command{{Kind: state.CmdAnswer, ID: a.ID, Decision: event.Deny}})
			return
		}
		*at = time.Now()
		l.do(r, act(a))
	}
}

func textDeltas(evs []event.Event) (n int, text string) {
	for _, ev := range evs {
		if d, ok := ev.Payload.(event.Delta); ok && ev.Kind == event.MessageDelta && d.Block == "text" && ev.Agent == "" {
			n++
			text += d.Text
		}
	}
	return n, text
}

func (l *live) exists(name string) bool {
	_, err := os.Stat(filepath.Join(l.dir, name))
	return err == nil
}

// close closes stdin and checks the app-server and everything it started
// are gone.
func (l *live) close(r *run, name string) {
	l.t.Helper()
	pid := l.sp.last().Pid()
	kids := descendants(l.t, pid)
	r.st.Do(state.Quit{})
	t0 := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := r.sess.Close(ctx); err != nil {
		l.t.Error(err)
	}
	e := r.end()
	l.t.Logf("%s: %s, %v after stdin closed", name, e.Reason, time.Since(t0).Round(time.Millisecond))
	l.left(name, append(kids, pid))
}

// left fails for every process of pids still running 1 s on.
func (l *live) left(name string, pids []int) {
	l.t.Helper()
	for _, p := range pids {
		if !gone(p, time.Second) {
			l.t.Errorf("%s: process %d still running", name, p)
		}
	}
	l.t.Logf("%s: %d processes checked gone", name, len(pids))
}

// newLive checks the project and finds Codex.
func newLive(t *testing.T) *live {
	t.Helper()
	dir := os.Getenv("ACC_LIVE_PROJECT")
	if dir == "" {
		t.Fatal("ACC_LIVE_PROJECT must name an empty project folder outside the repository")
	}
	if ents, err := os.ReadDir(dir); err != nil || len(ents) != 0 {
		t.Fatalf("project %q not an empty folder: %v", dir, err)
	}
	h := provider.SystemHost()
	d := (&Provider{Host: h}).Detect(context.Background())
	if !d.Installed {
		t.Fatalf("Codex: %s", d.Problem)
	}
	t.Logf("Codex %s at %s", d.Version, d.Path)
	sp := &spawns{}
	l := &live{t: t, p: &Provider{Host: h, Spawner: sp}, sp: sp, exe: d.Path, dir: dir}
	if m := os.Getenv("ACC_LIVE_MODEL"); m != "" {
		l.p.setFallbackModel(m) // the model thread/start and thread/resume carry
	}
	if p := os.Getenv("ACC_LIVE_LOG"); p != "" {
		f, err := os.Create(p)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { f.Close() })
		l.log = f
	}
	t.Cleanup(func() { t.Logf("%d lines written, none forbidden unless reported", sp.written(t)) })
	return l
}

// TestLiveGate is proposal step 5, item 4 (L3); the configuration check is
// run around it.
func TestLiveGate(t *testing.T) {
	l := newLive(t)
	dir := l.dir

	// Process 1.
	r := l.start(context.Background(), "")
	defer l.dump(r, "p1")
	var id, cwd string
	for _, ev := range r.evs {
		if s, ok := ev.Payload.(event.Started); ok && ev.Kind == event.SessionStarted {
			id, cwd = s.SessionID, s.Cwd
			t.Logf("thread %s, model %s, cwd %s, version %s", s.SessionID, s.Model, s.Cwd, s.Version)
		}
	}
	if id == "" || !strings.EqualFold(filepath.Clean(cwd), filepath.Clean(dir)) {
		t.Fatalf("thread %q, cwd %q, project %q", id, cwd, dir)
	}

	// A file-change approval accepted writes the file.
	var asked time.Time
	accept := func(a event.Approval) []state.Command {
		return []state.Command{{Kind: state.CmdAnswer, ID: a.ID, Decision: event.Allow}}
	}
	end, evs := l.turn(r, "The code word is "+codeWord+"; remember it. Create the file accept.txt in the current directory containing exactly: ok. Then reply with the single word done.", l.approve(r, accept, &asked))
	b, err := os.ReadFile(filepath.Join(dir, "accept.txt"))
	if asked.IsZero() || end.Outcome != "completed" || err != nil || strings.TrimSpace(string(b)) != "ok" {
		t.Errorf("accept: approval asked %v, turn %+v, accept.txt %q, %v", !asked.IsZero(), end, b, err)
	}
	n, text := textDeltas(evs)
	t.Logf("accept: %d text deltas before turn/completed: %q", n, text)

	// decline: no file, the turn ends completed (CX-10 q1, recorded).
	asked = time.Time{}
	decline := func(a event.Approval) []state.Command {
		return []state.Command{{Kind: state.CmdAnswer, ID: a.ID, Decision: event.Deny}}
	}
	end, evs = l.turn(r, "Create the file decline.txt in the current directory containing exactly: no. If the change is refused, do not try any other way; reply with the single word refused.", l.approve(r, decline, &asked))
	_, text = textDeltas(evs)
	t.Logf("decline: turn %+v, %v after the answer, reply %q", end, time.Since(asked).Round(time.Millisecond), text)
	if asked.IsZero() || end.Outcome != "completed" || l.exists("decline.txt") {
		t.Errorf("decline: approval asked %v, turn %+v, decline.txt exists %v", !asked.IsZero(), end, l.exists("decline.txt"))
	}

	// cancel (c on the card): cancel, then turn/interrupt; no file, the turn
	// ends interrupted within 10 s (CX-10 q1, recorded).
	asked = time.Time{}
	cancelTurn := func(a event.Approval) []state.Command { return r.st.Do(state.CancelTurn{ID: a.ID}) }
	end, _ = l.turn(r, "Create the file cancel.txt in the current directory containing exactly: stop. Then reply with the single word done.", l.approve(r, cancelTurn, &asked))
	t.Logf("cancel: turn %+v, %v after the answer", end, time.Since(asked).Round(time.Millisecond))
	if asked.IsZero() || end.Outcome != "interrupted" || time.Since(asked) > 10*time.Second || l.exists("cancel.txt") {
		t.Errorf("cancel: approval asked %v, turn %+v, cancel.txt exists %v", !asked.IsZero(), end, l.exists("cancel.txt"))
	}

	// turn/interrupt ends a streaming turn interrupted within 10 s.
	var at time.Time
	deltas := 0
	end, _ = l.turn(r, "Write the numbers from one to two hundred in words, one per line, and nothing else.", func(ev event.Event) {
		l.approve(r, decline, new(time.Time))(ev)
		if d, ok := ev.Payload.(event.Delta); ok && ev.Kind == event.MessageDelta && d.Block == "text" && ev.Agent == "" {
			deltas++
		}
		if deltas >= 2 && at.IsZero() {
			at = time.Now()
			l.do(r, r.st.Do(state.Interrupt{}))
		}
	})
	t.Logf("interrupt: turn %+v, %d text deltas, %v after the interrupt", end, deltas, time.Since(at).Round(time.Millisecond))
	if at.IsZero() || end.Outcome != "interrupted" || time.Since(at) > 10*time.Second {
		t.Errorf("interrupt: sent %v, turn %+v", !at.IsZero(), end)
	}
	if r.st.Session.Phase != state.Ready {
		t.Errorf("phase %q (%s)", r.st.Session.Phase, r.st.Session.Reason)
	}
	l.close(r, "p1")

	// thread/resume from a new process continues the thread (CX-10 q4), its
	// answer streamed before turn/completed.
	ctx2, kill := context.WithCancel(context.Background())
	defer kill()
	r2 := l.start(ctx2, id)
	defer l.dump(r2, "p2")
	if r2.st.Session.ID != id {
		t.Errorf("resumed thread %q, want %q", r2.st.Session.ID, id)
	}
	end, evs = l.turn(r2, "What is the code word I gave you earlier? Reply with the code word only.", l.approve(r2, decline, new(time.Time)))
	n, text = textDeltas(evs)
	if end.Outcome != "completed" || n < 1 || !strings.Contains(text, codeWord) {
		t.Errorf("resume: turn %+v, %d text deltas: %q", end, n, text)
	}
	t.Logf("resume: %d text deltas before turn/completed: %q", n, text)

	// Killed mid-turn by its job object: nothing it started is left (CX-10 q8).
	r2.send("Write the numbers from one to three hundred in words, one per line, and nothing else.")
	if !r2.until(func(ev event.Event) bool {
		d, ok := ev.Payload.(event.Delta)
		return ok && ev.Kind == event.MessageDelta && d.Block == "text"
	}, liveTurn) || r2.done {
		t.Fatal("kill: no text before the kill")
	}
	pid := l.sp.last().Pid()
	pids := append(descendants(t, pid), pid)
	t0 := time.Now()
	kill()
	ended := r2.end()
	t.Logf("kill: %s, %v after the kill", ended.Reason, time.Since(t0).Round(time.Millisecond))
	if d := time.Since(t0); d > 5*time.Second {
		t.Errorf("kill: ended %v after the kill", d)
	}
	l.left("kill", pids)
}
