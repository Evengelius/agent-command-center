package app

import (
	"os"
	"runtime"
	"slices"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/Evengelius/agent-command-center/internal/input"
	"github.com/Evengelius/agent-command-center/internal/provider"
	"github.com/Evengelius/agent-command-center/internal/provider/claude"
	"github.com/Evengelius/agent-command-center/internal/provider/codex"
	"github.com/Evengelius/agent-command-center/internal/render"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// Plan 69: the last session of each agent CLI in a project, resumed from
// the home screen through the real adapters over the fake CLIs; criteria
// 69.2 to 69.4, 69.6 and 69.8 at the application's level.

// homeRig is the home screen over the real adapters, Claude Code's on wc and
// Codex's on wx, with the store ps and the project folder dir in the field.
func homeRig(t *testing.T, wc *wireClaude, wx *wireCodex, ps *input.Projects, dir string) *harness {
	t.Helper()
	h := newHarness(t, Options{Providers: []provider.Provider{
		installedClaude{&claude.Provider{Spawner: wc, Now: func() time.Time { return testNow }}},
		installedCodex{&codex.Provider{Spawner: wx, Now: func() time.Time { return testNow }}},
	}, Cwd: dir, Projects: ps})
	h.settle()
	h.send(tea.WindowSizeMsg{Width: 120, Height: 40})
	if s := h.m.State(); s.Screen != state.ScreenStartup || s.Startup.Phase != state.Asking {
		t.Fatalf("screen %q, phase %q", s.Screen, s.Startup.Phase)
	}
	return h
}

// claudeInit is Claude Code's system/init for session id in dir.
func claudeInit(dir, id string) string {
	return `{"type":"system","subtype":"init","cwd":` + quote(dir) + `,"session_id":"` + id + `","model":"claude-haiku-4-5","permissionMode":"default"}`
}

// claudeTurn is a turn's answer and its result in session id.
func claudeTurn(id, text string) []string {
	return []string{
		`{"type":"assistant","message":{"id":"msg_` + text + `","type":"message","role":"assistant","model":"claude-haiku-4-5","content":[{"type":"text","text":"` + text + `"}]},"parent_tool_use_id":null,"session_id":"` + id + `"}`,
		`{"type":"result","subtype":"success","is_error":false,"duration_ms":10,"num_turns":1,"result":"` + text + `","session_id":"` + id + `","total_cost_usd":0.01,"usage":{"input_tokens":10,"output_tokens":5},"modelUsage":{}}`,
	}
}

// withSession is a store holding project dir with its last session id of
// agent CLI cli, ended at ended.
func withSession(t *testing.T, dir, cli, id string, ended time.Time) *input.Projects {
	t.Helper()
	return storeWith(t, []input.Project{{Path: dir, Used: ended.Add(-time.Hour), Sessions: map[string]input.Session{cli: {ID: id, Ended: ended}}}})
}

// lastIn is cli's last session in the store for dir, as the file holds it.
func lastIn(t *testing.T, ps *input.Projects, dir, cli string) (input.Session, bool) {
	t.Helper()
	list, err := ps.Load()
	if err != nil {
		t.Fatal(err)
	}
	_, s, ok := ps.LastSession(list, dir, cli)
	return s, ok
}

// resumedArg is a Claude Code spawn's --resume argument, "" without one.
func resumedArg(argv []string) string {
	for i, a := range argv {
		if a == "--resume" && i+1 < len(argv) {
			return argv[i+1]
		}
	}
	return ""
}

// lastNote is zone B's last note of the dashboard's.
func lastNote(h *harness) string {
	conv := h.m.State().Conversation
	for i := len(conv) - 1; i >= 0; i-- {
		if conv[i].Role == "dashboard" {
			return conv[i].Text
		}
	}
	return ""
}

// resumeProjects are demoProjects(5) and the golden frames' project, the
// field's, with Claude Code's last session, its turn ended 3 minutes ago
// (plan 69, criterion 69.4).
func resumeProjects() []input.Project {
	return append(demoProjects(5), input.Project{Path: `C:\work\demo-project`, Used: testNow.Add(-8 * time.Minute),
		Sessions: map[string]input.Session{claude.ProviderID: {ID: "21f3b775-162f-47e1-bb16-aa91f8a808b9", Ended: testNow.Add(-3 * time.Minute)}}})
}

// says reports whether the frame holds text, wherever it wraps.
func says(frame, text string) bool {
	return strings.Contains(strings.Join(strings.Fields(frame), " "), text)
}

// TestLastSessionRestart is criteria 69.2 and 69.3: a session started from
// the home screen is recorded when the owner's turn ends, not before, a
// project another dashboard recorded meanwhile kept; a new dashboard on the
// same store, as after a full restart, offers it; Ctrl+R resumes it with
// Claude Code's --resume and zone B says so once Claude Code names it;
// START SESSION still starts a new session.
func TestLastSessionRestart(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		dir := t.TempDir()
		ps := &input.Projects{Dir: t.TempDir()}
		w := &wireClaude{}
		h := homeRig(t, w, &wireCodex{}, ps, dir)
		if h.m.ui.LastSession != "" {
			t.Fatalf("a note before any session: %q", h.m.ui.LastSession)
		}
		h.send(key(tea.KeyEnter, 0))
		if s := h.m.State(); s.Session.Phase != state.Ready || resumedArg(w.spawns()[0]) != "" || s.Session.Resumed != "" {
			t.Fatalf("START SESSION: session %q, argv %q", s.Session.Phase, w.spawns()[0])
		}
		w.last().emit(claudeInit(dir, "s-first"))
		h.settle()
		if _, ok := lastIn(t, ps, dir, claude.ProviderID); ok {
			t.Error("recorded before the owner's turn ended: the handshake's usage read is no turn")
		}
		// Another dashboard records a project meanwhile: the record, written
		// through Update, keeps it.
		other := t.TempDir()
		if _, err := ps.Update(func(l []input.Project) []input.Project { return ps.Record(l, other, testNow.Add(-time.Minute)) }); err != nil {
			t.Fatal(err)
		}
		h.send(tea.PasteMsg{Content: "remember the code word"})
		h.send(key(tea.KeyEnter, 0))
		w.last().emit(claudeTurn("s-first", "OK")...)
		h.settle()
		if s, ok := lastIn(t, ps, dir, claude.ProviderID); !ok || s.ID != "s-first" || !s.Ended.Equal(testNow) {
			t.Fatalf("after the turn: %+v, %t", s, ok)
		}
		if list, err := ps.Load(); err != nil || !slices.ContainsFunc(list, func(e input.Project) bool { return e.Path == other }) {
			t.Errorf("the record lost the project another dashboard recorded: %v, %v", list, err)
		}
		h.crash()

		// A full restart: a new dashboard on the same store.
		w2 := &wireClaude{}
		h2 := homeRig(t, w2, &wireCodex{}, ps, dir)
		if want := "Claude Code's last session here: now · Ctrl+R continues it"; h2.m.ui.LastSession != want || !says(h2.frame(), "Claude Code's last session") {
			t.Fatalf("the note %q, want %q:\n%s", h2.m.ui.LastSession, want, h2.frame())
		}
		h2.send(ctrl('r'))
		if sp := w2.spawns(); len(sp) != 1 || resumedArg(sp[0]) != "s-first" {
			t.Fatalf("Ctrl+R spawned %q", sp)
		}
		if s := h2.m.State(); s.Session.Resumed != "s-first" || s.Screen != state.ScreenDashboard {
			t.Fatalf("resumed %q on %q", s.Session.Resumed, s.Screen)
		}
		w2.last().emit(claudeInit(dir, "s-first"))
		h2.settle()
		if n := lastNote(h2); n != "Resumed Claude Code session s-first · Claude Code has its earlier context; its earlier messages are not shown here" {
			t.Errorf("zone B's note %q", n)
		}
		h2.crash()

		// START SESSION in a third dashboard: a new session.
		w3 := &wireClaude{}
		h3 := homeRig(t, w3, &wireCodex{}, ps, dir)
		h3.send(key(tea.KeyEnter, 0))
		if sp := w3.spawns(); len(sp) != 1 || resumedArg(sp[0]) != "" {
			t.Errorf("START SESSION with a session recorded spawned %q", sp)
		}
	})
}

// TestLastSessionOtherID is criterion 69.6's other ID: a CLI that names
// another session than the one asked for is said as such, and that session
// becomes the record.
func TestLastSessionOtherID(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		dir := t.TempDir()
		ps := withSession(t, dir, claude.ProviderID, "s-asked-0001", testNow.Add(-time.Hour))
		w := &wireClaude{}
		h := homeRig(t, w, &wireCodex{}, ps, dir)
		h.send(ctrl('r'))
		w.last().emit(claudeInit(dir, "s-other-0002"))
		h.settle()
		if n := lastNote(h); n != "Claude Code reports session s-other-, not s-asked-: its earlier context may not be there" {
			t.Errorf("zone B's note %q", n)
		}
		h.send(tea.PasteMsg{Content: "go on"})
		h.send(key(tea.KeyEnter, 0))
		w.last().emit(claudeTurn("s-other-0002", "OK")...)
		h.settle()
		if s, ok := lastIn(t, ps, dir, claude.ProviderID); !ok || s.ID != "s-other-0002" {
			t.Errorf("the record %+v, %t: want the session the CLI named", s, ok)
		}
	})
}

// TestLastSessionMatching is criterion 69.3's matching: only the chosen
// agent CLI's record in the chosen project is offered, a path differing in
// case or separators the same project on Windows; never the other CLI's,
// nor another project's or a worktree of it; Ctrl+R with none does nothing.
func TestLastSessionMatching(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		dir, other := t.TempDir(), t.TempDir()
		ps := storeWith(t, []input.Project{
			{Path: dir, Used: testNow.Add(-time.Hour), Sessions: map[string]input.Session{claude.ProviderID: {ID: "s-1", Ended: testNow.Add(-3 * time.Minute)}}},
			{Path: other, Used: testNow.Add(-2 * time.Hour)},
		})
		wc, wx := &wireClaude{}, &wireCodex{thread: "th-new"}
		h := homeRig(t, wc, wx, ps, dir)
		if want := "Claude Code's last session here: 3m ago · Ctrl+R continues it"; h.m.ui.LastSession != want {
			t.Fatalf("the note %q, want %q", h.m.ui.LastSession, want)
		}
		nothing := func(what string) {
			t.Helper()
			h.send(ctrl('r'))
			if len(wc.spawns()) != 0 || len(wx.spawns()) != 0 || h.m.State().Startup.Phase != state.Asking {
				t.Errorf("%s: Ctrl+R started something: %q %q, phase %q", what, wc.spawns(), wx.spawns(), h.m.State().Startup.Phase)
			}
		}
		h.send(key(tea.KeyDown, 0))
		if h.m.ui.Choice != codex.ProviderID || h.m.ui.LastSession != "" {
			t.Errorf("Codex chosen: %q, note %q", h.m.ui.Choice, h.m.ui.LastSession)
		}
		nothing("Codex, whose session is not recorded")
		h.send(key(tea.KeyUp, 0))
		h.m.path.SetText(other)
		h.send(tea.WindowSizeMsg{Width: 120, Height: 40})
		if h.m.ui.LastSession != "" {
			t.Errorf("another project: note %q", h.m.ui.LastSession)
		}
		nothing("another project")
		// A worktree of the project where the Worktrees view adds one,
		// <project>.worktrees\<name> (input.WorktreePath): another project,
		// though its path begins with the project's.
		wt := input.WorktreePath(dir, "feature")
		if err := os.MkdirAll(wt, 0o755); err != nil {
			t.Fatal(err)
		}
		h.m.path.SetText(wt)
		h.send(tea.WindowSizeMsg{Width: 120, Height: 40})
		if h.m.ui.LastSession != "" {
			t.Errorf("a worktree of the project: note %q", h.m.ui.LastSession)
		}
		nothing("a worktree of the project")
		if runtime.GOOS == "windows" {
			h.m.path.SetText(strings.ReplaceAll(strings.ToUpper(dir), `\`, "/"))
			h.send(tea.WindowSizeMsg{Width: 120, Height: 40})
			if h.m.ui.LastSession == "" {
				t.Error("the project written in capitals and with slashes is not the same project")
			}
		}
	})
}

// TestLastSessionCodex is criteria 69.4 and 69.6 for Codex, in picture mode
// and in grid mode: a click on the note resumes the thread, thread/resume
// asking for it without its turns, zone B saying so; after a crash, the
// crash resume's thread/resume is as before.
func TestLastSessionCodex(t *testing.T) {
	for _, c := range []struct {
		name  string
		w, h  int
		sixel bool
	}{{"picture", 208, 59, true}, {"grid", 80, 24, false}} {
		t.Run(c.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				dir := t.TempDir()
				ps := withSession(t, dir, codex.ProviderID, "th-0069", testNow.Add(-2*time.Hour))
				wx := &wireCodex{thread: "th-new"}
				h := homeRig(t, &wireClaude{}, wx, ps, dir)
				h.send(tea.WindowSizeMsg{Width: c.w, Height: c.h})
				if c.sixel {
					h.send(PrimaryDeviceAttributesEvent{61, 4, 6, 7, 14})
				}
				h.send(key(tea.KeyDown, 0)) // Codex
				if want := "OpenAI Codex's last session here: 2h ago · Ctrl+R continues it"; h.m.ui.LastSession != want {
					t.Fatalf("the note %q, want %q", h.m.ui.LastSession, want)
				}
				x, y := homeCell(t, h, render.HomeItemResume)
				h.send(click(x, y))
				ls := wx.method("thread/resume")
				if len(ls) != 1 || !strings.Contains(ls[0], `"threadId":"th-0069"`) || !strings.Contains(ls[0], `"excludeTurns":true`) {
					t.Fatalf("thread/resume %q", ls)
				}
				if n := wx.method("thread/start"); len(n) != 0 {
					t.Errorf("thread/start %q", n)
				}
				if n := lastNote(h); n != "Resumed OpenAI Codex thread th-0069 · OpenAI Codex has its earlier context; its earlier turns are not shown here" {
					t.Errorf("zone B's note %q", n)
				}
				wx.last().exit(1, "")
				h.settle()
				if s := h.m.State(); s.Session.Phase != state.Crashed {
					t.Fatalf("after the exit: %q", s.Session.Phase)
				}
				h.send(ctrl('r'))
				if ls = wx.method("thread/resume"); len(ls) != 2 || strings.Contains(ls[1], "excludeTurns") || !strings.Contains(ls[1], `"threadId":"th-0069"`) {
					t.Errorf("the crash resume's thread/resume %q", ls)
				}
			})
		})
	}
}

// TestLastSessionRefused is criterion 69.8: each CLI's refusal for want of
// the session clears the record, the card and the home note saying so;
// any other failure keeps it, the card and the note saying why, and Ctrl+R
// tries again; no new session after a refused resume, START SESSION still
// starting one.
func TestLastSessionRefused(t *testing.T) {
	type rig struct {
		h     *harness
		ps    *input.Projects
		dir   string
		spawn func() int // the CLI's spawns
		news  func() int // new sessions it was asked for: no --resume, or a thread/start
	}
	claudeRig := func(t *testing.T, id string, refuse func(p *wireProc)) rig {
		dir := t.TempDir()
		ps := withSession(t, dir, claude.ProviderID, id, testNow.Add(-time.Hour))
		w := &wireClaude{noInit: true}
		h := homeRig(t, w, &wireCodex{}, ps, dir)
		h.send(tea.WindowSizeMsg{Width: 160, Height: 60})
		h.send(ctrl('r'))
		refuse(w.last())
		h.settle()
		news := func() int {
			n := 0
			for _, a := range w.spawns() {
				if resumedArg(a) == "" {
					n++
				}
			}
			return n
		}
		return rig{h, ps, dir, func() int { return len(w.spawns()) }, news}
	}
	codexRig := func(t *testing.T, id, err string) rig {
		dir := t.TempDir()
		ps := withSession(t, dir, codex.ProviderID, id, testNow.Add(-time.Hour))
		w := &wireCodex{thread: "th-new", resumeErr: err}
		h := homeRig(t, &wireClaude{}, w, ps, dir)
		h.send(tea.WindowSizeMsg{Width: 160, Height: 60})
		h.send(key(tea.KeyDown, 0))
		h.send(ctrl('r'))
		return rig{h, ps, dir, func() int { return len(w.spawns()) }, func() int { return len(w.method("thread/start")) }}
	}
	for _, c := range []struct {
		name, cli, id string
		start         func(t *testing.T) rig
		gone          bool
		card, note    string
	}{
		{"Claude Code has no such session", claude.ProviderID, "s-gone", func(t *testing.T) rig {
			return claudeRig(t, "s-gone", func(p *wireProc) {
				p.emit(`{"type":"result","subtype":"error_during_execution","is_error":true,"num_turns":0,"session_id":"s-gone","errors":["No conversation found with session ID: s-gone"]}`)
				p.exit(1, "No conversation found with session ID: s-gone\n")
			})
		}, true, "Claude Code no longer has this session: it is no longer offered", "Not resumed: Claude Code no longer has its last session here"},
		{"Claude Code exits otherwise", claude.ProviderID, "s-kept", func(t *testing.T) rig {
			return claudeRig(t, "s-kept", func(p *wireProc) { p.exit(3, "Error: cannot start\n") })
		}, false, "The session stays offered: Ctrl+R on the startup screen tries again", "Not resumed: the CLI exited unexpectedly"},
		{"Codex has no such thread", codex.ProviderID, "th-gone", func(t *testing.T) rig {
			return codexRig(t, "th-gone", `{"code":-32600,"message":"no rollout found for thread id th-gone"}`)
		}, true, "OpenAI Codex no longer has this session: it is no longer offered", "Not resumed: OpenAI Codex no longer has its last session here"},
		{"Codex refuses otherwise", codex.ProviderID, "th-kept", func(t *testing.T) rig {
			return codexRig(t, "th-kept", `{"code":-32600,"message":"thread th-kept is locked"}`)
		}, false, "The session stays offered: Ctrl+R on the startup screen tries again", "Not resumed: handshake failed: thread/resume refused by Codex: thread th-kept is locked"},
	} {
		t.Run(c.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				r := c.start(t)
				h := r.h
				s := h.m.State()
				if s.Startup.Phase != state.StartFailed || s.Startup.ResumeGone != c.gone {
					t.Fatalf("startup %q, gone %t, reason %q", s.Startup.Phase, s.Startup.ResumeGone, s.Session.Reason)
				}
				if f := h.frame(); !says(f, "✗ The session could not be resumed") || !says(f, c.card) {
					t.Errorf("the card lacks its title or %q:\n%s", c.card, f)
				}
				if _, kept := lastIn(t, r.ps, r.dir, c.cli); kept == c.gone {
					t.Errorf("the record kept %t, want %t", kept, !c.gone)
				}
				if r.spawn() != 1 || r.news() != 0 {
					t.Errorf("%d spawns, %d new sessions asked for: no fallback", r.spawn(), r.news())
				}
				h.send(key(tea.KeyEscape, 0))
				if h.m.State().Screen != state.ScreenStartup || h.m.ui.NotResumed != c.note {
					t.Errorf("back home: screen %q, note %q, want %q", h.m.State().Screen, h.m.ui.NotResumed, c.note)
				}
				if c.gone != (h.m.ui.LastSession == "") {
					t.Errorf("back home: the offer %q, gone %t", h.m.ui.LastSession, c.gone)
				}
				if !says(h.frame(), "✗ Not resumed:") {
					t.Errorf("back home: the note not drawn:\n%s", h.frame())
				}
				// The note is the chosen CLI's in the chosen project: none for
				// the other CLI, nor in another project (the CLI field focused
				// for the arrows).
				toOther, toBack := key(tea.KeyDown, 0), key(tea.KeyUp, 0)
				if c.cli == codex.ProviderID {
					toOther, toBack = toBack, toOther
				}
				h.m.ui.Field = 0
				h.send(toOther)
				if h.m.ui.NotResumed != "" {
					t.Errorf("the other CLI chosen: note %q", h.m.ui.NotResumed)
				}
				h.send(toBack)
				h.m.path.SetText(t.TempDir())
				h.send(tea.WindowSizeMsg{Width: 160, Height: 60})
				if h.m.ui.NotResumed != "" {
					t.Errorf("another project: note %q", h.m.ui.NotResumed)
				}
				h.m.path.SetText(r.dir)
				h.send(tea.WindowSizeMsg{Width: 160, Height: 60})
				if h.m.ui.Choice != c.cli || h.m.ui.NotResumed != c.note {
					t.Errorf("the CLI and project again: %q, note %q, want %q", h.m.ui.Choice, h.m.ui.NotResumed, c.note)
				}
				if c.gone {
					h.send(key(tea.KeyEnter, 0)) // START SESSION still starts a new session
					if r.news() != 1 {
						t.Errorf("START SESSION after the refusal: %d new sessions", r.news())
					}
					return
				}
				h.send(ctrl('r')) // tries again
				if r.spawn() != 2 || r.news() != 0 {
					t.Errorf("Ctrl+R again: %d spawns, %d new", r.spawn(), r.news())
				}
			})
		})
	}
}

// TestLastSessionRefusedUnsaved is criterion 69.8 when the store cannot be
// written: the session its CLI refused for good is no longer offered,
// though its record could not be cleared, the store's problem said.
func TestLastSessionRefusedUnsaved(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		dir := t.TempDir()
		ps := withSession(t, dir, claude.ProviderID, "s-gone", testNow.Add(-time.Hour))
		w := &wireClaude{noInit: true}
		h := homeRig(t, w, &wireCodex{}, ps, dir)
		f, err := ps.File()
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(f, []byte("not a list"), 0o600); err != nil { // read past it, never overwritten
			t.Fatal(err)
		}
		h.send(ctrl('r'))
		p := w.last()
		p.emit(`{"type":"result","subtype":"error_during_execution","is_error":true,"num_turns":0,"session_id":"s-gone","errors":["No conversation found with session ID: s-gone"]}`)
		p.exit(1, "No conversation found with session ID: s-gone\n")
		h.settle()
		if s := h.m.State(); s.Startup.Phase != state.StartFailed || !s.Startup.ResumeGone {
			t.Fatalf("startup %q, gone %t", s.Startup.Phase, s.Startup.ResumeGone)
		}
		h.send(key(tea.KeyEscape, 0))
		if h.m.ui.LastSession != "" || h.m.ui.NotResumed != "Not resumed: Claude Code no longer has its last session here" || !strings.HasPrefix(h.m.ui.ProjectsProblem, "Recent projects not saved: ") {
			t.Errorf("back home: offer %q, note %q, problem %q", h.m.ui.LastSession, h.m.ui.NotResumed, h.m.ui.ProjectsProblem)
		}
	})
}
