//go:build live

package claude

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Evengelius/agent-command-center/internal/event"
	"github.com/Evengelius/agent-command-center/internal/provider"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// The live gate uses the owner's quota: it runs only with the live tag, in
// the project named by ACC_LIVE_PROJECT, an empty folder outside the
// repository, with the configuration check before and after (proposal,
// Approvals requested, item 4, run L1):
//
//	go test -tags live -count=1 -v -run TestLiveGate ./internal/provider/claude
//
// ACC_LIVE_LOG, when set, receives every event's kind and raw line.

const codeWord = "ORCHID-41"

// liveTurn is long enough for a short Haiku turn through Claude Code.
const liveTurn = 120 * time.Second

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
	exe string
	dir string
	log *os.File
}

// start launches one process; resume is "" for a new session.
func (l *live) start(ctx context.Context, resume string) *run {
	l.t.Helper()
	st := state.New()
	st.Context = state.Context{Provider: ProviderID, Project: l.dir}
	st.Startup.Phase = state.Starting
	sess, err := l.p.Start(ctx, provider.SessionContext{Executable: l.exe, ProjectPath: l.dir, ResumeID: resume})
	if err != nil {
		l.t.Fatal(err)
	}
	r := &run{t: l.t, sess: sess, st: st}
	if !r.until(func(ev event.Event) bool { return ev.Kind == event.SessionReady }, 60*time.Second) || r.done {
		l.t.Fatalf("no handshake (%d events)", len(r.evs))
	}
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

// turn sends text and reads until the turn completes; on is called for each
// event and may answer approvals or interrupt. It returns the turn's end and
// its events.
func (l *live) turn(r *run, text string, on func(event.Event)) (event.TurnEnd, []event.Event) {
	l.t.Helper()
	r.st.Do(state.Send{Text: text})
	if err := r.sess.Send(text); err != nil {
		l.t.Fatal(err)
	}
	from := len(r.evs)
	deadline := time.Now().Add(liveTurn)
	for time.Now().Before(deadline) {
		ev, ok := r.next(time.Second)
		if r.done {
			l.t.Fatalf("the session ended during the turn %q", text)
		}
		if !ok {
			if on != nil {
				on(event.Event{}) // a tick, for timed actions
			}
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

// answer answers every approval request of the turn with decision.
func (l *live) answer(r *run, decision string) func(event.Event) {
	return func(ev event.Event) {
		if ev.Kind == event.ApprovalRequest {
			a := ev.Payload.(event.Approval)
			l.t.Logf("approval %s %s: %s", a.Tool, decision, a.Input)
			if err := r.sess.Answer(a.ID, decision); err != nil {
				l.t.Error(err)
			}
		}
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

// interrupted checks an interrupt ended the turn within 10 s.
func (l *live) interrupted(what string, end event.TurnEnd, at time.Time) {
	l.t.Helper()
	d := time.Since(at)
	if at.IsZero() {
		l.t.Errorf("%s: no interrupt was sent; turn %+v", what, end)
		return
	}
	l.t.Logf("%s: turn ended %v after the interrupt: %+v", what, d.Round(time.Millisecond), end)
	if end.Outcome != "interrupted" || end.Subtype != "error_during_execution" || d > 10*time.Second {
		l.t.Errorf("%s: turn %+v, %v after the interrupt; want interrupted, error_during_execution, within 10 s", what, end, d)
	}
}

// bashInterrupt interrupts a running Bash command, 3 s after it was allowed
// or 8 s after the call when no approval was asked, and checks the turn
// ends. A standalone `sleep` is refused by Claude Code 2.1.300 before it
// runs (L1, first run), so the command is a ping.
func (l *live) bashInterrupt(r *run) {
	l.t.Helper()
	var at, allowed, called time.Time
	var tool string
	finished := false
	end, _ := l.turn(r, "Use the Bash tool to run exactly this command: ping -n 30 127.0.0.1. Then reply with the single word done.", func(ev event.Event) {
		switch tl, _ := ev.Payload.(event.Tool); {
		case ev.Kind == event.ApprovalRequest:
			l.answer(r, "allow")(ev)
			allowed = time.Now()
		case ev.Kind == event.ToolStarted && tl.Name == "Bash" && tool == "":
			tool, called = tl.ToolID, time.Now()
		case ev.Kind == event.ToolCompleted && tl.ToolID == tool && tool != "" && at.IsZero():
			finished = true
			l.t.Logf("(c) Bash finished before the interrupt: %s", tl.Output)
		}
		if at.IsZero() && !finished && (!allowed.IsZero() && time.Since(allowed) > 3*time.Second ||
			allowed.IsZero() && !called.IsZero() && time.Since(called) > 8*time.Second) {
			at = time.Now()
			if err := r.sess.Interrupt(); err != nil {
				l.t.Error(err)
			}
		}
	})
	l.t.Logf("(c) Bash approval asked: %v", !allowed.IsZero())
	l.interrupted("(c) running Bash", end, at)
}

// TestLiveBashInterrupt is (c) alone, for a re-run after a fix: a running
// Bash command interrupted, then the next turn runs.
func TestLiveBashInterrupt(t *testing.T) {
	l := newLive(t)
	r := l.start(context.Background(), "")
	defer l.dump(r, "bash")
	l.bashInterrupt(r)
	end, evs := l.turn(r, "Reply with the single word ready.", l.answer(r, "deny"))
	if _, text := textDeltas(evs); end.Outcome != "completed" || !strings.Contains(strings.ToLower(text), "ready") {
		t.Errorf("(c) next turn %+v answered %q", end, text)
	}
	r.st.Do(state.Quit{})
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := r.sess.Close(ctx); err != nil {
		t.Error(err)
	}
	t.Logf("%s", r.end().Reason)
}

// newLive checks the project and finds Claude Code.
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
		t.Fatalf("Claude Code: %s", d.Problem)
	}
	t.Logf("Claude Code %s at %s", d.Version, d.Path)
	// CloseStep above 10 s: an exit within 10 s of stdin closing is the CLI's own.
	l := &live{t: t, p: &Provider{Host: h, Args: []string{"--model", "haiku"}, CloseStep: 11 * time.Second}, exe: d.Path, dir: dir}
	if p := os.Getenv("ACC_LIVE_LOG"); p != "" {
		f, err := os.Create(p)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { f.Close() })
		l.log = f
	}
	return l
}

// TestLiveGate is proposal step 2, item 5, (a) to (g); (h) is the
// configuration check run around it.
func TestLiveGate(t *testing.T) {
	l := newLive(t)
	dir := l.dir

	// Process 1.
	r := l.start(context.Background(), "")
	defer l.dump(r, "p1")

	// (b) allowed: the file is written.
	end, _ := l.turn(r, "The code word is "+codeWord+"; remember it. Use the Write tool to create the file allow.txt in the current directory containing exactly: ok. Then reply with the single word done.", l.answer(r, "allow"))
	started := r.st.Session
	b, err := os.ReadFile(filepath.Join(dir, "allow.txt"))
	if end.Outcome != "completed" || err != nil || strings.TrimSpace(string(b)) != "ok" {
		t.Errorf("(b) allow: turn %+v, allow.txt %q, %v", end, b, err)
	}
	var id, cwd string
	for _, ev := range r.evs {
		if s, ok := ev.Payload.(event.Started); ok && ev.Kind == event.SessionStarted {
			id, cwd = s.SessionID, s.Cwd
			t.Logf("session %s, model %s, cwd %s, version %s", s.SessionID, s.Model, s.Cwd, s.Version)
		}
	}
	if id == "" || cwd != dir {
		t.Fatalf("session id %q, cwd %q, project %q (%+v)", id, cwd, dir, started)
	}

	// (b) denied: no file, and the turn ends normally.
	end, _ = l.turn(r, "Use the Write tool to create the file deny.txt in the current directory containing exactly: no. If the write is refused, do not try any other way; reply with the single word refused.", l.answer(r, "deny"))
	if end.Outcome != "completed" || l.exists("deny.txt") {
		t.Errorf("(b) deny: turn %+v, deny.txt exists %v", end, l.exists("deny.txt"))
	}

	// (d) an interrupt while an approval is pending ends the turn.
	var at time.Time
	end, _ = l.turn(r, "Use the Write tool to create the file pending.txt in the current directory containing exactly: wait. Then reply with the single word done.", func(ev event.Event) {
		if ev.Kind == event.ApprovalRequest && at.IsZero() {
			at = time.Now()
			if err := r.sess.Interrupt(); err != nil {
				t.Error(err)
			}
		}
	})
	l.interrupted("(d) pending approval", end, at)
	if l.exists("pending.txt") {
		t.Error("(d) pending.txt was written")
	}

	// (c) an interrupt during a running Bash command (CC-10 q4) ends the turn.
	l.bashInterrupt(r)

	// (a) and (c): the next turn runs and streams.
	end, evs := l.turn(r, "Write the numbers from one to fifteen in words, one per line, and nothing else.", l.answer(r, "deny"))
	if n, text := textDeltas(evs); end.Outcome != "completed" || n < 2 {
		t.Errorf("(a) turn %+v with %d text deltas: %q", end, n, text)
	} else {
		t.Logf("(a) %d text deltas before the result", n)
	}
	if r.st.Session.Phase != state.Ready {
		t.Errorf("phase %q (%s)", r.st.Session.Phase, r.st.Session.Reason)
	}

	// (e) after stdin closes the process exits within 10 s.
	r.st.Do(state.Quit{})
	t0 := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	if err := r.sess.Close(ctx); err != nil {
		t.Error(err)
	}
	cancel()
	ended := r.end()
	t.Logf("(e) %s, %v after stdin closed", ended.Reason, time.Since(t0).Round(time.Millisecond))
	if d := time.Since(t0); d > 10*time.Second || !strings.HasPrefix(ended.Reason, "Claude Code closed") {
		t.Errorf("(e) %+v after %v", ended, d)
	}

	// (f) a new process resumes the history.
	ctx2, kill := context.WithCancel(context.Background())
	defer kill()
	r2 := l.start(ctx2, id)
	defer l.dump(r2, "p2")
	ask := "What is the code word I gave you earlier? Reply with the code word only."
	end, evs = l.turn(r2, ask, l.answer(r2, "deny"))
	if _, text := textDeltas(evs); end.Outcome != "completed" || !strings.Contains(text, codeWord) {
		t.Errorf("(f) turn %+v answered %q", end, text)
	}
	id2 := id
	for _, ev := range r2.evs {
		if s, ok := ev.Payload.(event.Started); ok && ev.Kind == event.SessionStarted {
			id2 = s.SessionID
		}
	}
	t.Logf("(f) resumed %s as %s", id, id2)

	// (g) killed mid-turn by its job object, then resumed.
	text := "Write the numbers from one to three hundred in words, one per line, and nothing else."
	r2.st.Do(state.Send{Text: text})
	if err := r2.sess.Send(text); err != nil {
		t.Fatal(err)
	}
	if !r2.until(func(ev event.Event) bool {
		d, ok := ev.Payload.(event.Delta)
		return ok && ev.Kind == event.MessageDelta && d.Block == "text"
	}, liveTurn) || r2.done {
		t.Fatal("(g) no text before the kill")
	}
	t0 = time.Now()
	kill()
	ended = r2.end()
	t.Logf("(g) killed mid-turn: %s, %v after the kill", ended.Reason, time.Since(t0).Round(time.Millisecond))
	if d := time.Since(t0); d > 5*time.Second {
		t.Errorf("(g) ended %v after the kill", d)
	}

	r3 := l.start(context.Background(), id2)
	defer l.dump(r3, "p3")
	end, evs = l.turn(r3, ask, l.answer(r3, "deny"))
	if _, text := textDeltas(evs); end.Outcome != "completed" || !strings.Contains(text, codeWord) {
		t.Errorf("(g) turn %+v answered %q", end, text)
	}
	r3.st.Do(state.Quit{})
	ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := r3.sess.Close(ctx); err != nil {
		t.Error(err)
	}
	t.Logf("p3 %s", r3.end().Reason)
}
