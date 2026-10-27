package app

import (
	"context"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/Evengelius/agent-command-center/internal/input"
	"github.com/Evengelius/agent-command-center/internal/provider"
	"github.com/Evengelius/agent-command-center/internal/state"
)

func write(t *testing.T, p, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestCheckProject is proposal step 1, item 1, on disk: steps 4 and 5 of INT
// paths and the branch, each with ui-design.md's message.
func TestCheckProject(t *testing.T) {
	root := t.TempDir()
	env := input.PathEnv{Windows: runtime.GOOS == "windows", Home: root, Cwd: root}
	empty := filepath.Join(root, "empty")
	if err := os.Mkdir(empty, 0o755); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(root, "notes.txt")
	write(t, file, "x")
	repo := filepath.Join(root, "repo")
	write(t, filepath.Join(repo, ".git", "HEAD"), "ref: refs/heads/feature/x\n")
	write(t, filepath.Join(repo, "src", "main.go"), "package main\n")
	wt := filepath.Join(root, "worktree")
	write(t, filepath.Join(wt, ".git"), "gitdir: ../repo/.git/worktrees/wt\n")
	write(t, filepath.Join(repo, ".git", "worktrees", "wt", "HEAD"), "0123456789abcdef0123456789abcdef01234567\n")
	missing := filepath.Join(root, "agent-comand-center")

	cases := []struct {
		name, in, path, branch, problem string
	}{
		{"empty directory", empty, empty, "", ""},
		{"relative, against the current directory", "empty", empty, "", ""},
		{"home", "~/empty", empty, "", ""},
		{"does not exist", missing, "", "", "Does not exist: " + missing},
		{"a file", file, "", "", "Is a file, not a directory: " + file},
		{"a form refused before any I/O", "~bob/app", "", "", "Another user's home is not expanded. Type the full path"},
		{"git work tree", repo, repo, "feature/x", ""},
		{"inside a git work tree", filepath.Join(repo, "src"), filepath.Join(repo, "src"), "feature/x", ""},
		{"linked worktree, detached head", wt, wt, "0123456", ""},
	}
	if env.Windows {
		// /mnt/<drive>/… of the missing folder: translated, then not found.
		mnt := "/mnt/" + strings.ToLower(missing[:1]) + filepath.ToSlash(missing[2:])
		cases = append(cases, struct{ name, in, path, branch, problem string }{
			"translated WSL path not found", mnt, "", "",
			"Does not exist: " + missing + " (the WSL mount root may not be /mnt/)"})
	}
	for _, c := range cases {
		p, b, problem := checkProject(c.in, env)
		if p != c.path || b != c.branch || problem != c.problem {
			t.Errorf("%s: checkProject(%q) = %q, %q, %q; want %q, %q, %q", c.name, c.in, p, b, problem, c.path, c.branch, c.problem)
		}
	}
}

// TestCheckProjectNotReadable: a folder the user may not list is refused
// with its message. On Windows the folder gets a deny ACE for the current
// user's "list folder" right; the test fails, not skips, if it cannot.
func TestCheckProjectNotReadable(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "locked")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS == "windows" {
		u, err := user.Current()
		if err != nil {
			t.Fatal(err)
		}
		if out, err := exec.Command("icacls", dir, "/deny", u.Username+":(RD)").CombinedOutput(); err != nil {
			t.Fatalf("icacls deny: %v: %s", err, out)
		}
		t.Cleanup(func() {
			if out, err := exec.Command("icacls", dir, "/remove:d", u.Username).CombinedOutput(); err != nil {
				t.Errorf("icacls remove: %v: %s", err, out)
			}
		})
	} else {
		if err := os.Chmod(dir, 0o000); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { os.Chmod(dir, 0o755) })
	}
	env := input.PathEnv{Windows: runtime.GOOS == "windows", Cwd: filepath.Dir(dir)}
	_, _, problem := checkProject(dir, env)
	if want := "Not readable (permission denied): " + dir; problem != want {
		t.Errorf("problem %q, want %q", problem, want)
	}
}

func TestParseHead(t *testing.T) {
	for in, want := range map[string]string{
		"ref: refs/heads/main\n":                                           "main",
		"ref: refs/heads/feature/a-b":                                      "feature/a-b",
		"ref: refs/remotes/origin/x\n":                                     "refs/remotes/origin/x",
		"0123456789abcdef0123456789abcdef01234567\n":                       "0123456",
		"0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef": "0123456",
		"":         "",
		"garbage":  "",
		"ZZZZZZZZ": "",
	} {
		if got := parseHead(in); got != want {
			t.Errorf("parseHead(%q) = %q, want %q", in, got, want)
		}
	}
}

// hangCLI is an adapter whose detection never returns.
type hangCLI struct {
	id, name string
	stop     chan struct{}
}

func (p hangCLI) ID() string          { return p.id }
func (p hangCLI) DisplayName() string { return p.name }
func (p hangCLI) Detect(context.Context) provider.Detection {
	<-p.stop
	return provider.Detection{}
}
func (p hangCLI) Start(context.Context, provider.SessionContext) (provider.Session, error) {
	return nil, nil
}

// TestStartupGate is proposal step 1, items 2 and 4, through the
// application: a CLI not found cannot be chosen, Start does nothing with
// neither CLI, a detection that hangs is reported after its bound, and a
// project is validated, with its branch, before anything starts.
func TestStartupGate(t *testing.T) {
	t.Run("neither CLI: Start disabled", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			h := newHarness(t, Options{Providers: []provider.Provider{
				fakeCLI{provider.Detection{ID: "claude-code", Name: "Claude Code", Problem: "claude not found on PATH"}},
				fakeCLI{provider.Detection{ID: "codex", Name: "OpenAI Codex", Problem: "codex not found on PATH"}},
			}, Cwd: t.TempDir()})
			h.settle()
			h.send(key(tea.KeyEnter, 0))
			s := h.m.State()
			if s.Startup.Phase != state.Asking || s.Screen != state.ScreenStartup {
				t.Fatalf("phase %q screen %q", s.Startup.Phase, s.Screen)
			}
			h.send(tea.WindowSizeMsg{Width: 120, Height: 40})
			// NEW SESSION's text: a message is found whole across its wrap.
			panel := homePanel(h.m.View().Content, 0)
			for _, want := range []string{
				"✗ No supported agent CLI found. Install Claude Code or Codex CLI, then restart.",
				"not installed: claude not found on PATH",
				"not installed: codex not found on PATH",
			} {
				if !homeSays(panel, want) {
					t.Errorf("NEW SESSION lacks %q", want)
				}
			}
			if homeSays(panel, "✗ not installed") {
				t.Error("the field message repeats the no-CLI message")
			}
		})
	})

	t.Run("detection timeout", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			stop := make(chan struct{})
			defer close(stop)
			h := newHarness(t, Options{
				Providers: []provider.Provider{
					hangCLI{"claude-code", "Claude Code", stop},
					fakeCLI{provider.Detection{ID: "codex", Name: "OpenAI Codex", Installed: true, Version: "0.145.0", Path: `C:\Tools\codex.exe`}},
				},
				Cwd:           t.TempDir(),
				DetectTimeout: 200 * time.Millisecond,
			})
			h.settle()
			s := h.m.State()
			if s.Startup.Phase != state.Asking {
				t.Fatalf("phase %q", s.Startup.Phase)
			}
			var got string
			for _, c := range s.Startup.CLIs {
				if c.ID == "claude-code" {
					got = c.Problem
				}
			}
			if got != "detection timed out after 0.2 s" {
				t.Errorf("problem %q", got)
			}
			if strings.Join(s.Startup.Installed, ",") != "codex" || h.m.ui.Choice != "codex" {
				t.Errorf("installed %v, choice %q", s.Startup.Installed, h.m.ui.Choice)
			}
		})
	})

	t.Run("--provider and --project only prefill", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			dir := t.TempDir()
			installed := func(id, name string) provider.Provider {
				return fakeCLI{provider.Detection{ID: id, Name: name, Installed: true, Version: "1.0.0", Path: `C:\Tools\x.exe`}}
			}
			h := newHarness(t, Options{Config: Config{Provider: "codex", Project: dir},
				Providers: []provider.Provider{installed("claude-code", "Claude Code"), installed("codex", "OpenAI Codex")}})
			h.settle()
			s := h.m.State()
			if s.Startup.Phase != state.Asking || s.Screen != state.ScreenStartup || h.m.ui.Choice != "codex" || h.m.path.Text() != dir {
				t.Errorf("phase %q screen %q choice %q path %q", s.Startup.Phase, s.Screen, h.m.ui.Choice, h.m.path.Text())
			}
			// A --provider not installed falls back to the first installed CLI.
			g := newHarness(t, Options{Config: Config{Provider: "codex"}, Cwd: dir,
				Providers: []provider.Provider{installed("claude-code", "Claude Code"),
					fakeCLI{provider.Detection{ID: "codex", Name: "OpenAI Codex", Problem: "codex not found on PATH"}}}})
			g.settle()
			if g.m.ui.Choice != "claude-code" || g.m.State().Startup.Phase != state.Asking {
				t.Errorf("choice %q phase %q", g.m.ui.Choice, g.m.State().Startup.Phase)
			}
		})
	})

	t.Run("project refused, then accepted with its branch", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			root := t.TempDir()
			repo := filepath.Join(root, "repo")
			write(t, filepath.Join(repo, ".git", "HEAD"), "ref: refs/heads/main\n")
			h := newHarness(t, Options{Providers: []provider.Provider{
				fakeCLI{provider.Detection{ID: "claude-code", Name: "Claude Code", Installed: true, Version: "2.1.300", Path: `C:\Tools\claude.exe`}},
			}, Cwd: root})
			h.settle()
			h.m.path.SetText("missing")
			h.send(key(tea.KeyEnter, 0))
			s := h.m.State()
			if want := "Does not exist: " + filepath.Join(root, "missing"); s.Startup.Phase != state.Asking || s.Startup.Field != "project" || s.Startup.Message != want {
				t.Fatalf("phase %q field %q message %q, want %q", s.Startup.Phase, s.Startup.Field, s.Startup.Message, want)
			}
			h.m.path.SetText("repo")
			h.send(key(tea.KeyEnter, 0))
			s = h.m.State()
			if s.Context.Project != repo || s.Context.Branch != "main" || s.Context.Provider != "claude-code" {
				t.Fatalf("context %+v", s.Context)
			}
			// The fake CLI refuses to start: the session never ran.
			if s.Startup.Phase != state.StartFailed {
				t.Errorf("phase %q, want %q", s.Startup.Phase, state.StartFailed)
			}
		})
	})
}
