//go:build live

package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/Evengelius/agent-command-center/internal/input"
	"github.com/Evengelius/agent-command-center/internal/proc"
	"github.com/Evengelius/agent-command-center/internal/provider"
	"github.com/Evengelius/agent-command-center/internal/provider/claude"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// L2 uses the owner's quota: it runs only with the live tag, in the project
// named by ACC_LIVE_PROJECT, an empty folder outside the repository, with
// the configuration check before and after (proposal step 4, check 6, and the
// scope the owner approved: Haiku, three short turns, every turn capped at
// 2 minutes, any failure stops the run, an approval outside the folder is
// denied and stops it):
//
//	go test -tags live -count=1 -v -run TestLiveDashboard ./internal/app
//
// ACC_LIVE_RECORD, optional, names a folder outside the project for the
// timing of the CLI's stdout, without its content. What the turns check is
// in livecheck_test.go, checked there on fixtures.

const liveWord = "LANTERN-73"

// spawnLog is the real spawner, keeping each process and its argv. With rec
// set, it also writes for each process when each read of its stdout came
// and its size (reads-N.txt), and each stdout line's shape (lines-N.tsv):
// types, sizes and the order of message ids, never a text or an id, to show
// how the CLI paces its stream (the second L2 run, 2026-10-16).
type spawnLog struct {
	mu    sync.Mutex
	procs []proc.Process
	argv  [][]string
	rec   string
	t0    time.Time
	files []*os.File
}

func (s *spawnLog) Spawn(sp proc.Spec) (proc.Process, error) {
	p, err := proc.Exec{}.Spawn(sp)
	if err != nil {
		return p, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.procs = append(s.procs, p)
	s.argv = append(s.argv, append([]string(nil), sp.Args...))
	if s.rec == "" {
		return p, nil
	}
	n := len(s.procs)
	reads, err1 := os.Create(filepath.Join(s.rec, fmt.Sprintf("reads-%d.txt", n)))
	lines, err2 := os.Create(filepath.Join(s.rec, fmt.Sprintf("lines-%d.tsv", n)))
	if err := errors.Join(err1, err2); err != nil {
		return p, err
	}
	s.files = append(s.files, reads, lines)
	fmt.Fprintln(lines, "t_ms\tbytes\ttype\tsubtype\tevent\tindex\tdelta\ttext_bytes\tmessage")
	return recorded{p, &shapeReader{r: p.Stdout(), reads: reads, lines: lines, t0: s.t0, ids: map[string]int{}}}, nil
}

func (s *spawnLog) close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, f := range s.files {
		f.Close()
	}
}

type recorded struct {
	proc.Process
	out io.Reader
}

func (r recorded) Stdout() io.Reader { return r.out }

func (s *spawnLog) last() (proc.Process, []string, int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.procs[len(s.procs)-1], s.argv[len(s.argv)-1], len(s.procs)
}

// detected is the real adapter with its detection made once, before the
// dashboard opens.
type detected struct {
	*claude.Provider
	d provider.Detection
}

func (p detected) Detect(context.Context) provider.Detection { return p.d }

// inside reports whether an approval's file path is in the project.
func inside(dir string, ap *state.Approval) bool {
	var in struct {
		FilePath string `json:"file_path"`
	}
	if ap.Tool != "Write" || json.Unmarshal(ap.Input, &in) != nil || in.FilePath == "" {
		return false
	}
	rel, err := filepath.Rel(strings.ToLower(filepath.Clean(dir)), strings.ToLower(filepath.Clean(in.FilePath)))
	return err == nil && !filepath.IsAbs(rel) && !strings.HasPrefix(rel, "..")
}

// TestLiveDashboard is L2: one turn with an approval allowed through the
// dashboard, one turn interrupted mid-stream, the CLI killed from outside,
// Ctrl+R, and one turn showing the resumed session remembers.
func TestLiveDashboard(t *testing.T) {
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
	cp := &claude.Provider{Host: host, Spawner: spawns, Args: []string{"--model", "haiku"}, CloseStep: 11 * time.Second}
	d := cp.Detect(context.Background())
	if !d.Installed {
		t.Fatalf("Claude Code: %s", d.Problem)
	}
	t.Logf("Claude Code %s at %s", d.Version, d.Path)
	drafts := input.Drafts{Dir: t.TempDir()}
	h := newHarness(t, Options{Providers: []provider.Provider{detected{cp, d}}, Cwd: dir, Drafts: &drafts})
	h.settle()
	h.m.ui.Choice = claude.ProviderID
	h.send(key(tea.KeyEnter, 0))
	h.send(tea.WindowSizeMsg{Width: 160, Height: 60})
	if !h.until(func(s *state.State) bool { return s.Session.Phase == state.Ready || s.Session.Phase == state.Crashed }, HandshakeTimeout) ||
		h.m.State().Session.Phase != state.Ready {
		s := h.m.State()
		t.Fatalf("not Ready: startup %q, session %q (%s)\n%s", s.Startup.Phase, s.Session.Phase, s.Session.Reason, frameText(h))
	}
	t.Logf("Ready, cwd %s", h.m.State().Context.Project)

	// 1. A turn writes a file; its approval is allowed with F2 then a.
	say(h, "The code word is "+liveWord+"; remember it. Use the Write tool to create the file allow.txt in the current directory containing exactly: ok. Then reply with the single word done.")
	answered := 0
	stop := time.Now().Add(liveTurn)
	for {
		if !h.until(func(s *state.State) bool { return pending(s) != nil || idle(s) }, time.Until(stop)) {
			t.Fatalf("turn 1 did not end within %v\n%s", liveTurn, frameText(h))
		}
		ap := pending(h.m.State())
		if ap == nil {
			break
		}
		if !inside(dir, ap) || answered > 0 {
			h.send(key(tea.KeyF2, 0))
			h.send(typed('d'))
			t.Fatalf("turn 1: approval %s %s denied (outside the folder or a second one); run stopped", ap.Tool, ap.Input)
		}
		t.Logf("turn 1: approval %s %s, allowed with F2, a", ap.Tool, ap.Input)
		h.send(key(tea.KeyF2, 0))
		h.send(typed('a'))
		answered++
	}
	s := h.m.State()
	// Claude Code reports the session id with the first turn, not at Ready.
	id := s.Session.ID
	if id == "" {
		t.Fatalf("turn 1: no session id")
	}
	t.Logf("session %s, model %s", id, s.Session.Model)
	b, err := os.ReadFile(filepath.Join(dir, "allow.txt"))
	if s.Turn.Outcome != "completed" || answered != 1 || err != nil || strings.TrimSpace(string(b)) != "ok" {
		t.Fatalf("turn 1: outcome %q, %d approvals, allow.txt %q (%v)\n%s", s.Turn.Outcome, answered, b, err, frameText(h))
	}
	if f := frameText(h); !strings.Contains(f, "allowed by you") {
		t.Errorf("turn 1: no resolved approval line in the frame:\n%s", f)
	}
	t.Logf("turn 1: completed, allow.txt %q, answer %q", strings.TrimSpace(string(b)), lastAssistant(s))
	// What steps 6 to 8 draw from the turn (plan 4, Done): zone A's stage and
	// tool, zone D's values or N/A with a reason, B's badge; telemetry may
	// trail the turn's end by a few events.
	h.until(func(s *state.State) bool { return activityShown(s, frameText(h)) == nil }, 10*time.Second)
	if err := activityShown(h.m.State(), frameText(h)); err != nil {
		t.Errorf("turn 1: %v\n%s", err, frameText(h))
	}
	checkFrame(t, h.m.View().Content, 160, 60)
	t.Logf("turn 1: activity shown; frame:\n%s", frameText(h))

	// 2. A longer turn interrupted with Ctrl+C at its first streamed text,
	// its answer one entry (interruptMidStream, checked on P2's turn by
	// TestInterruptMidStream).
	c, err := interruptMidStream(h, "Write the numbers from one to three hundred in words, one per line, and nothing else.", liveTurn)
	if err != nil {
		t.Fatalf("turn 2: %v; %s\n%s", err, c.stats, frameText(h))
	}
	s = h.m.State()
	t.Logf("turn 2: %q %v after Ctrl+C, session %q, %d bytes streamed as one answer; %s", s.Turn.Outcome, c.took.Round(time.Millisecond), s.Session.Phase, c.stats.textBytes, c.stats)

	// 3. Claude Code killed from outside: the crash card offers Ctrl+R.
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

	// 4. Ctrl+R resumes the session in a new process.
	h.send(key('r', tea.ModCtrl))
	if !h.until(func(s *state.State) bool { return s.Session.Phase == state.Ready || s.Session.Phase == state.Crashed }, HandshakeTimeout) ||
		h.m.State().Session.Phase != state.Ready {
		t.Fatalf("not Ready after Ctrl+R: session %q (%s)\n%s", h.m.State().Session.Phase, h.m.State().Session.Reason, frameText(h))
	}
	_, argv2, n2 := spawns.last()
	want := resumeArgv(argv1, id, cp.Args) // checked in TestResumeAndQuitChecks
	if n2 != n+1 || want == nil || strings.Join(argv2, "\x00") != strings.Join(want, "\x00") {
		t.Errorf("resume: %d spawns, argv %q; want %d, %q", n2, argv2, n+1, want)
	}
	t.Logf("resumed %s as %s", id, h.m.State().Session.ID)

	// 5. The resumed session remembers turn 1.
	say(h, "What is the code word I gave you earlier? Reply with the code word only.")
	if !h.until(func(s *state.State) bool { return pending(s) != nil || idle(s) }, liveTurn) {
		t.Fatalf("turn 3 did not end within %v\n%s", liveTurn, frameText(h))
	}
	if p := pending(h.m.State()); p != nil {
		h.send(key(tea.KeyF2, 0))
		h.send(typed('d'))
		t.Fatalf("turn 3: unexpected approval %s %s, denied; run stopped", p.Tool, p.Input)
	}
	s = h.m.State()
	if a := lastAssistant(s); s.Turn.Outcome != "completed" || !strings.Contains(a, liveWord) {
		t.Errorf("turn 3: outcome %q, answer %q; want %s", s.Turn.Outcome, a, liveWord)
	} else {
		t.Logf("turn 3: answered %q", a)
	}

	// Ctrl+Q while idle closes Claude Code's input and quits once it has
	// ended, within the close step before any kill (TestResumeAndQuitChecks).
	q := time.Now()
	h.deliver(key('q', tea.ModCtrl))
	if !h.quitWithin(CloseTimeout) {
		t.Errorf("Ctrl+Q did not quit within %v: session %q\n%s", CloseTimeout, h.m.State().Session.Phase, frameText(h))
	} else {
		t.Logf("quit %v after Ctrl+Q, session %q", time.Since(q).Round(time.Millisecond), h.m.State().Session.Phase)
	}
	// A clean end, checked before the shutdown could kill anything: the
	// session Closed on Claude Code's own exit, code 0, and its process gone.
	last, _, _ := spawns.last()
	if s := h.m.State(); s.Session.Phase != state.Closed || !s.Session.HasExitCode || s.Session.ExitCode != 0 {
		t.Errorf("after Ctrl+Q: session %q, exit code %d (%v); want Closed with 0", s.Session.Phase, s.Session.ExitCode, s.Session.HasExitCode)
	}
	ended := make(chan int, 1)
	go func() { c, _ := last.Wait(); ended <- c }()
	select {
	case c := <-ended:
		if c != 0 {
			t.Errorf("Claude Code (pid %d) exited with %d after Ctrl+Q; want 0", last.Pid(), c)
		} else {
			t.Logf("Claude Code (pid %d) exited with 0 before the shutdown", last.Pid())
		}
	case <-time.After(2 * time.Second):
		t.Errorf("Claude Code (pid %d) still running 2 s after the quit", last.Pid())
	}
	t0 := time.Now()
	h.m.Shutdown()
	done := make(chan int, 1)
	go func() { c, _ := last.Wait(); done <- c }()
	select {
	case c := <-done:
		t.Logf("Claude Code exited with %d, %v after the shutdown", c, time.Since(t0).Round(time.Millisecond))
	case <-time.After(30 * time.Second):
		t.Errorf("Claude Code still running 30 s after the shutdown")
	}
	t.Logf("frame at the end:\n%s", frameText(h))
}
