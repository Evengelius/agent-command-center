//go:build live

package app

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/Evengelius/agent-command-center/internal/input"
	"github.com/Evengelius/agent-command-center/internal/provider"
	"github.com/Evengelius/agent-command-center/internal/provider/codex"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// L4 is L2's scenario with Codex (proposal step 5, item 4). It uses the
// owner's quota: it runs only with the live tag, in the project named by
// ACC_LIVE_PROJECT, an empty folder outside the repository, with the
// configuration check before and after, under the scope the owner approves;
// any failure stops the run, and an approval outside the folder is declined
// and stops it:
//
//	go test -tags live -count=1 -v -run TestLiveDashboardCodex ./internal/app
//
// The model is Codex's default; the adapter's fallback applies if it is
// refused. ACC_LIVE_RECORD is as for L2.

// detectedCodex is the real Codex adapter with its detection made once.
type detectedCodex struct {
	*codex.Provider
	d provider.Detection
}

func (p detectedCodex) Detect(context.Context) provider.Detection { return p.d }

// TestLiveDashboardCodex is L4: one turn with a file-change approval
// accepted through the dashboard, one turn interrupted mid-stream, codex
// app-server killed from outside, Ctrl+R, and one turn showing the resumed
// thread remembers.
func TestLiveDashboardCodex(t *testing.T) {
	dir := os.Getenv("ACC_LIVE_PROJECT")
	if dir == "" {
		t.Fatal("ACC_LIVE_PROJECT must name an empty project folder outside the repository")
	}
	if ents, err := os.ReadDir(dir); err != nil || len(ents) != 0 {
		t.Fatalf("project %q not an empty folder: %v", dir, err)
	}
	rec := os.Getenv("ACC_LIVE_RECORD")
	if rel, err := filepath.Rel(dir, rec); rec != "" && (err != nil || !strings.HasPrefix(rel, "..")) {
		t.Fatalf("ACC_LIVE_RECORD %q must be outside the project %q", rec, dir)
	}
	host := provider.SystemHost()
	spawns := &spawnLog{rec: rec, t0: time.Now()}
	defer spawns.close()
	cp := &codex.Provider{Host: host, Spawner: spawns}
	d := cp.Detect(context.Background())
	if !d.Installed {
		t.Fatalf("Codex: %s", d.Problem)
	}
	t.Logf("Codex %s at %s", d.Version, d.Path)
	drafts := input.Drafts{Dir: t.TempDir()}
	h := newHarness(t, Options{Providers: []provider.Provider{detectedCodex{cp, d}}, Cwd: dir, Drafts: &drafts})
	h.settle()
	h.m.ui.Choice = codex.ProviderID
	h.send(key(tea.KeyEnter, 0))
	h.send(tea.WindowSizeMsg{Width: 160, Height: 60})
	// Ready comes from the thread/start response, the thread id from the
	// session.started event that follows it (codex/mapper.go, threadStart).
	started := func(s *state.State) bool {
		return s.Session.Phase == state.Ready && s.Session.ID != "" || s.Session.Phase == state.Crashed
	}
	if !h.until(started, HandshakeTimeout) ||
		h.m.State().Session.Phase != state.Ready {
		s := h.m.State()
		t.Fatalf("not Ready: startup %q, session %q (%s)\n%s", s.Startup.Phase, s.Session.Phase, s.Session.Reason, frameText(h))
	}
	// Codex gives the thread id with thread/start, at Ready.
	s := h.m.State()
	id := s.Session.ID
	if id == "" {
		t.Fatal("no thread id at Ready")
	}
	t.Logf("Ready, cwd %s, thread %s, model %s", s.Context.Project, id, s.Session.Model)

	// 1. A turn writes a file; its approval is accepted with F2 then a. When
	// Codex refuses the session's model, the adapter starts a new thread with
	// model/list's default and says so in zone B (codex/session.go, react):
	// the message is sent again there, once.
	answered, declined := 0, 0
	for resent := false; ; resent = true {
		say(h, "The code word is "+liveWord+"; remember it. Create the file allow.txt in the current directory containing exactly: ok (a trailing newline is fine). Then reply with the single word done.")
		stop := time.Now().Add(liveTurn)
		for {
			if !h.until(func(s *state.State) bool { return pending(s) != nil || idle(s) }, time.Until(stop)) {
				t.Fatalf("turn 1 did not end within %v\n%s", liveTurn, frameText(h))
			}
			ap := pending(h.m.State())
			if ap == nil {
				break
			}
			// Only the first request inside the folder is accepted; any other is
			// declined, as L3 does, and the turn goes on.
			if !insideCodex(dir, ap) || answered > 0 {
				t.Logf("turn 1: approval %s %q declined with F2, d (outside the folder, unknown paths or a second one)", ap.Tool, ap.Summary)
				h.send(key(tea.KeyF2, 0))
				h.send(typed('d'))
				declined++
				continue
			}
			t.Logf("turn 1: approval %s %q, accepted with F2, a", ap.Tool, ap.Summary)
			h.send(key(tea.KeyF2, 0))
			h.send(typed('a'))
			answered++
		}
		if s = h.m.State(); s.Turn.Outcome != "failed" || answered > 0 || resent {
			break
		}
		for _, e := range s.Errors {
			t.Logf("turn 1 failed: error %q", e.Message)
		}
		refusal := func(s *state.State) bool {
			for _, e := range s.Conversation {
				if e.Role == "notice" && strings.HasPrefix(e.Text, "Codex refused the model") {
					return s.Session.Phase == state.Ready && s.Session.ID != id
				}
			}
			return s.Session.Phase == state.Crashed
		}
		refused := s.Session.Model
		if !h.until(refusal, HandshakeTimeout) || h.m.State().Session.Phase != state.Ready {
			t.Fatalf("turn 1 failed and no model fallback followed\n%s", frameText(h))
		}
		s = h.m.State()
		t.Logf("turn 1: model %s refused; fallback thread %s, model %s; message sent again", refused, s.Session.ID, s.Session.Model)
		id = s.Session.ID
	}
	s = h.m.State()
	for _, e := range s.Errors {
		t.Logf("error after turn 1: %q", e.Message)
	}
	b, err := os.ReadFile(filepath.Join(dir, "allow.txt"))
	if s.Turn.Outcome != "completed" || answered != 1 || err != nil || strings.TrimSpace(string(b)) != "ok" {
		t.Fatalf("turn 1: outcome %q, %d accepted, %d declined, allow.txt %q (%v)\n%s", s.Turn.Outcome, answered, declined, b, err, frameText(h))
	}
	if f := frameText(h); !strings.Contains(f, "allowed by you") {
		t.Errorf("turn 1: no resolved approval line in the frame:\n%s", f)
	}
	t.Logf("turn 1: completed, %d declined, allow.txt %q, answer %q", declined, strings.TrimSpace(string(b)), lastAssistant(s))

	// 2. A longer turn interrupted with Ctrl+C at its first streamed text,
	// its answer one entry.
	c, err := interruptMidStream(h, "Write the numbers from one to three hundred in words, one per line, and nothing else.", liveTurn)
	if err != nil {
		t.Fatalf("turn 2: %v; %s\n%s", err, c.stats, frameText(h))
	}
	s = h.m.State()
	t.Logf("turn 2: %q %v after Ctrl+C, session %q, %d bytes streamed as one answer; %s", s.Turn.Outcome, c.took.Round(time.Millisecond), s.Session.Phase, c.stats.textBytes, c.stats)

	// 3. codex app-server killed from outside: the crash card offers Ctrl+R.
	p, argv1, n := spawns.last()
	out, err := exec.Command("taskkill", "/F", "/PID", strconv.Itoa(p.Pid())).CombinedOutput()
	t.Logf("taskkill %d: %s", p.Pid(), strings.TrimSpace(string(out)))
	if err != nil {
		t.Fatalf("taskkill: %v", err)
	}
	if !h.until(func(s *state.State) bool { return s.Session.Phase == state.Crashed }, 15*time.Second) {
		t.Fatalf("no crash after the kill: session %q\n%s", h.m.State().Session.Phase, frameText(h))
	}
	f := frameText(h)
	for _, want := range []string{"The session ended unexpectedly", "Ctrl+R resume"} {
		if !strings.Contains(f, want) {
			t.Errorf("crash card lacks %q:\n%s", want, f)
		}
	}
	s = h.m.State()
	t.Logf("crash: exit code %d (%v), %s", s.Session.ExitCode, s.Session.HasExitCode, s.Session.Reason)

	// 4. Ctrl+R spawns app-server again, with the same argv, and resumes the
	// thread with thread/resume (TestCodexCrashAndResume).
	h.send(key('r', tea.ModCtrl))
	if !h.until(started, HandshakeTimeout) ||
		h.m.State().Session.Phase != state.Ready {
		t.Fatalf("not Ready after Ctrl+R: session %q (%s)\n%s", h.m.State().Session.Phase, h.m.State().Session.Reason, frameText(h))
	}
	_, argv2, n2 := spawns.last()
	if n2 != n+1 || !slices.Equal(argv1, argv2) || h.m.State().Session.ID != id {
		t.Errorf("resume: %d spawns, argv %q, thread %q; want %d, %q, %s", n2, argv2, h.m.State().Session.ID, n+1, argv1, id)
	}
	t.Logf("resumed thread %s", h.m.State().Session.ID)

	// 5. The resumed thread remembers turn 1.
	say(h, "What is the code word I gave you earlier? Reply with the code word only.")
	if !h.until(func(s *state.State) bool { return pending(s) != nil || idle(s) }, liveTurn) {
		t.Fatalf("turn 3 did not end within %v\n%s", liveTurn, frameText(h))
	}
	if p := pending(h.m.State()); p != nil {
		h.send(key(tea.KeyF2, 0))
		h.send(typed('d'))
		t.Fatalf("turn 3: unexpected approval %s %q, declined; run stopped", p.Tool, p.Summary)
	}
	s = h.m.State()
	if a := lastAssistant(s); s.Turn.Outcome != "completed" || !strings.Contains(a, liveWord) {
		t.Errorf("turn 3: outcome %q, answer %q; want %s", s.Turn.Outcome, a, liveWord)
	} else {
		t.Logf("turn 3: answered %q", a)
	}

	// Ctrl+Q while idle closes app-server's input and quits once it has
	// ended, within the close step before any kill.
	q := time.Now()
	h.deliver(key('q', tea.ModCtrl))
	if !h.quitWithin(CloseTimeout) {
		t.Errorf("Ctrl+Q did not quit within %v: session %q\n%s", CloseTimeout, h.m.State().Session.Phase, frameText(h))
	} else {
		t.Logf("quit %v after Ctrl+Q, session %q", time.Since(q).Round(time.Millisecond), h.m.State().Session.Phase)
	}
	// A clean end, checked before the shutdown could kill anything: the
	// session Closed on app-server's own exit, code 0, and its process gone.
	last, _, _ := spawns.last()
	if s := h.m.State(); s.Session.Phase != state.Closed || !s.Session.HasExitCode || s.Session.ExitCode != 0 {
		t.Errorf("after Ctrl+Q: session %q, exit code %d (%v); want Closed with 0", s.Session.Phase, s.Session.ExitCode, s.Session.HasExitCode)
	}
	ended := make(chan int, 1)
	go func() { c, _ := last.Wait(); ended <- c }()
	select {
	case c := <-ended:
		if c != 0 {
			t.Errorf("codex app-server (pid %d) exited with %d after Ctrl+Q; want 0", last.Pid(), c)
		} else {
			t.Logf("codex app-server (pid %d) exited with 0 before the shutdown", last.Pid())
		}
	case <-time.After(2 * time.Second):
		t.Errorf("codex app-server (pid %d) still running 2 s after the quit", last.Pid())
	}
	t0 := time.Now()
	h.m.Shutdown()
	done := make(chan int, 1)
	go func() { c, _ := last.Wait(); done <- c }()
	select {
	case c := <-done:
		t.Logf("codex app-server exited with %d, %v after the shutdown", c, time.Since(t0).Round(time.Millisecond))
	case <-time.After(30 * time.Second):
		t.Errorf("codex app-server still running 30 s after the shutdown")
	}
	t.Logf("frame at the end:\n%s", frameText(h))
}
