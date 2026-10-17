package app

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/synctest"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/Evengelius/agent-command-center/internal/provider"
	"github.com/Evengelius/agent-command-center/internal/render"
	"github.com/Evengelius/agent-command-center/internal/state"
)

var update = flag.Bool("update", false, "rewrite the golden frames in testdata/frames")

// fakeCLI is a detected agent CLI that is never started.
type fakeCLI struct{ d provider.Detection }

func (p fakeCLI) ID() string                                { return p.d.ID }
func (p fakeCLI) DisplayName() string                       { return p.d.Name }
func (p fakeCLI) Detect(context.Context) provider.Detection { return p.d }
func (p fakeCLI) Start(context.Context, provider.SessionContext) (provider.Session, error) {
	return nil, fmt.Errorf("fake CLI")
}

var sizes = [][2]int{{80, 24}, {120, 40}, {200, 60}, {79, 24}}

// TestGoldenFrames is test 9 (proposal step 0, item 9): both screens at the
// three reference sizes and too small at 79x24, each frame exactly the
// terminal's height and no line wider than it.
func TestGoldenFrames(t *testing.T) {
	startup := func(t *testing.T) *harness {
		h := newHarness(t, Options{
			Providers: []provider.Provider{
				fakeCLI{provider.Detection{ID: "claude-code", Name: "Claude Code", Installed: true, Version: "2.1.300", Path: `C:\Tools\claude.exe`}},
				fakeCLI{provider.Detection{ID: "codex", Name: "OpenAI Codex", Problem: "not found on the PATH"}},
			},
			Cwd: `C:\work\demo-project`,
		})
		h.settle()
		if s := h.m.State(); s.Screen != state.ScreenStartup || s.Startup.Phase != state.Asking {
			t.Fatalf("startup: screen %q, phase %q", s.Screen, s.Startup.Phase)
		}
		return h
	}
	dash := func(t *testing.T) *harness {
		h := newHarness(t, Options{Config: Config{Mock: filepath.Join("..", "..", "testdata", "mock", "claude-subagent-approval.jsonl")}})
		h.settle()
		if s := h.m.State(); s.Screen != state.ScreenDashboard || len(s.Log) != 155 { // 154 and the subagent's model (plan 5, item 7)
			t.Fatalf("dashboard: screen %q, %d events, want 155", s.Screen, len(s.Log))
		}
		return h
	}
	codexDash := func(t *testing.T) *harness {
		h := newHarness(t, Options{Config: Config{Mock: filepath.Join("..", "..", "testdata", "mock", "codex-subagents.jsonl")}})
		h.settle()
		if s := h.m.State(); s.Screen != state.ScreenDashboard || len(s.Log) != 177 {
			t.Fatalf("codex dashboard: screen %q, %d events, want 177", s.Screen, len(s.Log))
		}
		return h
	}
	// The inspection: the Claude mock's subagent, on its Tools tab.
	inspected := func(t *testing.T) *harness {
		h := dash(t)
		h.send(key(tea.KeyF6, 0)) // zone C
		h.send(key(tea.KeyEnter, 0))
		h.send(typed('2'))
		if h.m.ui.Inspect == "" || h.m.ui.Tab != 1 {
			t.Fatalf("inspect %q, tab %d", h.m.ui.Inspect, h.m.ui.Tab)
		}
		return h
	}

	// Zone A focused, its cursor on the first child: at 80 columns the tree
	// takes the place of zones B and C.
	tree := func(t *testing.T) *harness {
		h := codexDash(t)
		for _, k := range []tea.KeyPressMsg{key(tea.KeyF6, 0), key(tea.KeyF6, 0), key(tea.KeyUp, 0), key(tea.KeyUp, 0), key(tea.KeyUp, 0), key(tea.KeyUp, 0)} {
			h.send(k)
		}
		if h.m.ui.Focus != "A" || !strings.HasPrefix(h.m.ui.ACursor, "a:") {
			t.Fatalf("focus %q cursor %q", h.m.ui.Focus, h.m.ui.ACursor)
		}
		return h
	}

	// The details of zone D, with errors of the session, of an agent and a
	// retry, one long and on two lines (plan 4, step 7).
	details := func(t *testing.T) *harness {
		h := dash(t)
		addErrors(h, 2)
		openDetailsView(t, h)
		return h
	}

	for _, sc := range []struct {
		name string
		rig  func(*testing.T) *harness
	}{{"startup", startup}, {"dashboard", dash}, {"dashboard-codex", codexDash}, {"inspect", inspected}, {"tree", tree}, {"details", details}} {
		for _, size := range sizes {
			w, hgt := size[0], size[1]
			name := fmt.Sprintf("%s-%dx%d", sc.name, w, hgt)
			t.Run(name, func(t *testing.T) {
				synctest.Test(t, func(t *testing.T) {
					h := sc.rig(t)
					h.send(tea.WindowSizeMsg{Width: w, Height: hgt})
					frame := h.m.View().Content
					lines := strings.Split(frame, "\n")
					if len(lines) != hgt {
						t.Errorf("%d lines, want %d", len(lines), hgt)
					}
					for i, l := range lines {
						if n := lipgloss.Width(l); n > w {
							t.Errorf("line %d is %d columns wide, terminal %d: %q", i+1, n, w, plainText(l))
						}
					}
					text := plainText(frame)
					small := strings.Contains(text, "Terminal too small")
					if small != (w < render.MinWidth || hgt < render.MinHeight) {
						t.Errorf("too-small message shown %v at %dx%d", small, w, hgt)
					}
					golden(t, name, text)
				})
			})
		}
	}
}

// golden compares a frame with testdata/frames/<name>.txt; -update rewrites it.
func golden(t *testing.T, name, text string) {
	t.Helper()
	path := filepath.Join("..", "..", "testdata", "frames", name+".txt")
	text = strings.Join(trimRight(strings.Split(text, "\n")), "\n") + "\n"
	if *update {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(want); got != text {
		gl, wl := strings.Split(text, "\n"), strings.Split(got, "\n")
		for i := 0; i < len(gl) && i < len(wl); i++ {
			if gl[i] != wl[i] {
				t.Fatalf("%s differs at line %d:\n got %q\nwant %q\n(rerun with -update and review the diff)", path, i+1, gl[i], wl[i])
			}
		}
		t.Fatalf("%s: %d lines, golden has %d", path, len(gl), len(wl))
	}
}

// trimRight drops trailing spaces, so the golden files diff cleanly; the
// width check runs on the frame itself.
func trimRight(ls []string) []string {
	for i, l := range ls {
		ls[i] = strings.TrimRight(l, " ")
	}
	return ls
}

// TestStartupMessageFrames is proposal step 1, item 5: the startup screen
// with each kind of message, at 80x24 where they are tightest. Each message
// is wrapped whole, never cut.
func TestStartupMessageFrames(t *testing.T) {
	claudeOK := fakeCLI{provider.Detection{ID: "claude-code", Name: "Claude Code", Installed: true, Version: "2.1.300", Path: `C:\Users\dev\.local\bin\claude.exe`}}
	codexOK := fakeCLI{provider.Detection{ID: "codex", Name: "OpenAI Codex", Installed: true, Version: "0.145.0", Path: `C:\Users\dev\AppData\Local\Programs\OpenAI\Codex\bin\codex.exe`}}
	codexMissing := fakeCLI{provider.Detection{ID: "codex", Name: "OpenAI Codex", Problem: "codex not found on PATH"}}
	cases := []struct {
		name  string
		provs []provider.Provider
		typed string // "" keeps the default
		enter bool
		want  string // in the frame, whitespace collapsed
	}{
		{"no-cli", []provider.Provider{
			fakeCLI{provider.Detection{ID: "claude-code", Name: "Claude Code", Path: `C:\Users\dev\.local\bin\claude.exe`, Problem: "version unreadable: exit 1: error: the installation is damaged, reinstall Claude Code"}},
			fakeCLI{provider.Detection{ID: "codex", Name: "OpenAI Codex", Problem: "npm install found (codex.cmd), native codex.exe needed"}},
		}, "", true, "✗ No supported agent CLI found. Install Claude Code or Codex CLI, then restart."},
		{"does-not-exist", []provider.Provider{claudeOK, codexMissing}, `C:\Users\dev\projets\agent-comand-center`, true,
			`✗ Does not exist: C:\Users\…\projets\agent-comand-center`},
		{"wsl-not-found", []provider.Provider{claudeOK, codexOK}, "/mnt/q/acc-test-missing/app", true,
			`✗ Does not exist: Q:\acc-test-missing\app (the WSL mount root may not be /mnt/)`},
		{"live-linux-path", []provider.Provider{claudeOK, codexOK}, "/home/u/app", false,
			`✗ A Linux path. On Windows use C:\…, or /mnt/c/… for a Windows drive`},
		{"live-drive-relative", []provider.Provider{claudeOK, codexOK}, `C:dev\app`, false,
			`✗ Drive-relative path. Type a full path, for example C:\dev\app`},
		{"warning-wsl", []provider.Provider{claudeOK, codexOK}, `\\wsl.localhost\Ubuntu\home\u\app`, false,
			"! files in Linux, used by a Windows agent CLI: slow, and unverified"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				h := newHarness(t, Options{Providers: c.provs, Cwd: `C:\work\demo-project`, Home: `C:\Users\dev`})
				h.settle()
				if c.typed != "" {
					h.m.path.SetText(c.typed)
				}
				h.send(tea.WindowSizeMsg{Width: 80, Height: 24})
				if c.enter {
					h.send(key(tea.KeyEnter, 0))
				}
				frame := h.m.View().Content
				lines := strings.Split(frame, "\n")
				if len(lines) != 24 {
					t.Errorf("%d lines, want 24", len(lines))
				}
				for i, l := range lines {
					if n := lipgloss.Width(l); n > 80 {
						t.Errorf("line %d is %d columns wide", i+1, n)
					}
				}
				text := plainText(frame)
				if flat := strings.Join(strings.Fields(text), " "); !strings.Contains(flat, c.want) {
					t.Errorf("frame lacks %q", c.want)
				}
				if strings.Contains(text, "…\n") {
					t.Error("a line is cut")
				}
				golden(t, "startup-msg-"+c.name, text)
			})
		})
	}
}
