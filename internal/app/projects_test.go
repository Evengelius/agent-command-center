package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/Evengelius/agent-command-center/internal/input"
	"github.com/Evengelius/agent-command-center/internal/provider"
	"github.com/Evengelius/agent-command-center/internal/state"
)

var bothCLIs = []provider.Provider{
	fakeCLI{provider.Detection{ID: "claude-code", Name: "Claude Code", Installed: true, Version: "2.1.300", Path: `C:\Tools\claude.exe`}},
	fakeCLI{provider.Detection{ID: "codex", Name: "OpenAI Codex", Installed: true, Version: "0.145.0", Path: `C:\Tools\codex.exe`}},
}

// storeWith is a project store in a temporary folder holding list.
func storeWith(t *testing.T, list []input.Project) *input.Projects {
	p := &input.Projects{Dir: t.TempDir()}
	if list != nil {
		if err := p.Save(list); err != nil {
			t.Fatal(err)
		}
	}
	return p
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

// demoProjects are n recent projects, the latest first, from minutes to days old.
func demoProjects(n int) []input.Project {
	names := []string{"agent-command-center", "billing-service", "a-project-with-a-very-long-directory-name", "docs", "infra", "mobile-app", "web", "scratch", "tools", "payments"}
	var out []input.Project
	for i := range n {
		out = append(out, input.Project{Path: `C:\javadev\projets\` + names[i%len(names)], Used: testNow.Add(-time.Duration(i*i*i*7+i) * time.Minute)})
	}
	return out
}

// favouriteProjects are demoProjects(10) with two of them starred and a
// starred directory no session started in.
func favouriteProjects() []input.Project {
	list := demoProjects(10)
	list[3].Favourite, list[7].Favourite = true, true
	return append(list, input.Project{Path: `D:\clients\acme-portal`, Favourite: true})
}

// projectsRig is the startup screen with list; sel >= 0 focuses the list on
// that row (golden frames, plan 15).
func projectsRig(list []input.Project, sel int) func(*testing.T) *harness {
	return func(t *testing.T) *harness {
		h := newHarness(t, Options{Providers: bothCLIs[:1], Cwd: `C:\work\demo-project`, Projects: storeWith(t, list)})
		h.settle()
		if len(h.m.projectList) != len(list) {
			t.Fatalf("%d projects listed, want %d (%s)", len(h.m.projectList), len(list), h.m.ui.ProjectsProblem)
		}
		if sel >= 0 {
			h.send(key(tea.KeyTab, 0))
			h.send(key(tea.KeyTab, 0))
			for range sel {
				h.send(key(tea.KeyDown, 0))
			}
			if h.m.ui.Field != 3 || h.m.ui.ProjectSel != sel {
				t.Fatalf("field %d row %d", h.m.ui.Field, h.m.ui.ProjectSel)
			}
		}
		return h
	}
}

// TestProjectRecording is criterion 1.3 of plan 15: a session started from
// the startup screen records its validated path; a refused validation, a
// mock run or no store records nothing.
func TestProjectRecording(t *testing.T) {
	t.Run("validated path recorded, refused one not", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			root := t.TempDir()
			repo := filepath.Join(root, "repo")
			write(t, filepath.Join(repo, ".git", "HEAD"), "ref: refs/heads/main\n")
			store := storeWith(t, nil)
			h := newHarness(t, Options{Providers: bothCLIs, Cwd: root, Projects: store})
			h.settle()
			h.m.path.SetText("missing")
			h.send(key(tea.KeyEnter, 0))
			if f, _ := store.File(); fileExists(f) {
				t.Fatal("a refused validation wrote the list")
			}
			h.m.path.SetText(`repo\`)
			h.send(key(tea.KeyEnter, 0))
			if h.m.State().Context.Project != repo {
				t.Fatalf("context %+v", h.m.State().Context)
			}
			got, err := store.Load()
			if err != nil || len(got) != 1 || got[0].Path != repo || !got[0].Used.Equal(testNow) {
				t.Fatalf("stored %v, %v; want %s at %v", got, err, repo, testNow)
			}
		})
	})
	t.Run("mock run", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			store := storeWith(t, nil)
			h := newHarness(t, Options{Config: Config{Mock: filepath.Join("..", "..", "testdata", "mock", "claude-subagent-approval.jsonl")}, Projects: store})
			h.settle()
			if h.m.State().Screen != state.ScreenDashboard || h.m.projects != nil || h.m.projectsGen != 0 {
				t.Fatalf("screen %q, store %v, %d reads or changes", h.m.State().Screen, h.m.projects, h.m.projectsGen)
			}
			if entries, _ := os.ReadDir(store.Dir); len(entries) != 0 {
				t.Errorf("a mock run wrote %v", entries)
			}
		})
	})
	t.Run("no store", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			root := t.TempDir()
			h := newHarness(t, Options{Providers: bothCLIs, Cwd: root})
			h.settle()
			h.send(key(tea.KeyEnter, 0))
			if h.m.State().Context.Project != root || h.m.projectsGen != 0 || h.m.ui.ProjectsProblem != "" {
				t.Fatalf("project %q, %d reads or changes, problem %q", h.m.State().Context.Project, h.m.projectsGen, h.m.ui.ProjectsProblem)
			}
		})
	})
}

// TestProjectListKeys is criterion 1.5 of plan 15: Tab goes agent, project,
// list, Start; ↑↓ move in the list and still choose the agent elsewhere;
// Enter starts with the entry's path; Del forgets it in the store too.
func TestProjectListKeys(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		dirs := []string{t.TempDir(), t.TempDir(), t.TempDir()}
		var list []input.Project
		for i, d := range dirs {
			list = append(list, input.Project{Path: d, Used: testNow.Add(-time.Duration(i) * time.Hour)})
		}
		store := storeWith(t, list)
		h := newHarness(t, Options{Providers: bothCLIs, Cwd: t.TempDir(), Projects: store})
		h.settle()

		var order []int
		for range 5 {
			order = append(order, h.m.ui.Field)
			h.send(key(tea.KeyTab, 0))
		}
		if fmt.Sprint(order) != "[0 1 3 2 0]" {
			t.Errorf("Tab order %v, want [0 1 3 2 0]", order)
		}
		order = nil
		for range 5 {
			order = append(order, h.m.ui.Field)
			h.send(key(tea.KeyTab, tea.ModShift))
		}
		if fmt.Sprint(order) != "[1 0 2 3 1]" {
			t.Errorf("Shift+Tab order %v, want [1 0 2 3 1]", order)
		}

		// Field 1: ↑↓ still choose the agent.
		h.m.ui.Field = 1
		choice := h.m.ui.Choice
		h.send(key(tea.KeyDown, 0))
		if h.m.ui.Choice == choice || h.m.ui.ProjectSel != 0 {
			t.Errorf("down in the project field: choice %q (was %q), row %d", h.m.ui.Choice, choice, h.m.ui.ProjectSel)
		}

		h.send(key(tea.KeyTab, 0)) // the list
		for _, k := range []rune{tea.KeyUp, tea.KeyDown, tea.KeyDown, tea.KeyDown, tea.KeyUp} {
			h.send(key(k, 0))
		}
		if h.m.ui.Field != 3 || h.m.ui.ProjectSel != 1 {
			t.Fatalf("field %d row %d, want 3 and 1", h.m.ui.Field, h.m.ui.ProjectSel)
		}
		if h.m.ui.Choice == choice {
			t.Error("↑↓ in the list changed the agent")
		}

		h.send(key(tea.KeyDelete, 0))
		got, err := store.Load()
		if err != nil || len(got) != 2 || got[0].Path != dirs[0] || got[1].Path != dirs[2] {
			t.Fatalf("after Del: stored %v, %v", got, err)
		}
		if len(h.m.projectList) != 2 || h.m.ui.ProjectSel != 1 {
			t.Fatalf("after Del: %d listed, row %d", len(h.m.projectList), h.m.ui.ProjectSel)
		}

		h.send(key(tea.KeyEnter, 0))
		if s := h.m.State(); s.Context.Project != dirs[2] || h.m.path.Text() != dirs[2] {
			t.Fatalf("Enter: project %q, field %q, want %s", s.Context.Project, h.m.path.Text(), dirs[2])
		}
		if got, _ := store.Load(); len(got) != 2 || got[0].Path != dirs[2] || !got[0].Used.Equal(testNow) {
			t.Errorf("after the start: stored %v", got)
		}
	})
}

// TestProjectStoreProblems is criterion 1.6 of plan 15: a list that cannot
// be read says so on the startup screen, never as an empty list; one that
// cannot be written says so too.
func TestProjectStoreProblems(t *testing.T) {
	t.Run("unreadable", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			store := storeWith(t, nil)
			f, _ := store.File()
			if err := os.WriteFile(f, []byte("{"), 0o600); err != nil {
				t.Fatal(err)
			}
			h := newHarness(t, Options{Providers: bothCLIs, Cwd: t.TempDir(), Projects: store})
			h.settle()
			h.send(tea.WindowSizeMsg{Width: 120, Height: 40})
			text := strings.Join(strings.Fields(plainText(h.m.View().Content)), " ")
			if !strings.Contains(text, "✗ Recent projects unavailable: ") || !strings.Contains(text, "projects.json") {
				t.Errorf("frame lacks the read failure:\n%s", text)
			}
			h.send(key(tea.KeyEnter, 0)) // the start tries to record: still not overwritten
			if b, _ := os.ReadFile(f); string(b) != "{" {
				t.Errorf("the unreadable list was overwritten: %q", b)
			}
		})
	})
	t.Run("unwritable", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			blocker := filepath.Join(t.TempDir(), "a-file")
			write(t, blocker, "")
			store := &input.Projects{Dir: filepath.Join(blocker, "acc")} // under a file
			h := newHarness(t, Options{Providers: bothCLIs, Cwd: t.TempDir(), Projects: store})
			h.settle()
			if h.m.ui.ProjectsProblem != "" {
				t.Fatalf("read: %q", h.m.ui.ProjectsProblem)
			}
			h.send(key(tea.KeyEnter, 0))
			if !strings.HasPrefix(h.m.ui.ProjectsProblem, "Recent projects not saved: ") {
				t.Fatalf("problem %q", h.m.ui.ProjectsProblem)
			}
			h.send(tea.WindowSizeMsg{Width: 120, Height: 40})
			text := strings.Join(strings.Fields(plainText(h.m.View().Content)), " ")
			if h.m.State().Screen == state.ScreenStartup && !strings.Contains(text, "✗ Recent projects not saved: ") {
				t.Errorf("startup frame lacks the write failure:\n%s", text)
			}
		})
	})
}

func TestAge(t *testing.T) {
	for _, c := range []struct {
		d    time.Duration
		want string
	}{
		{0, "now"}, {59 * time.Second, "now"}, {time.Minute, "1m ago"}, {59 * time.Minute, "59m ago"},
		{time.Hour, "1h ago"}, {23 * time.Hour, "23h ago"}, {24 * time.Hour, "1d ago"}, {29 * 24 * time.Hour, "29d ago"},
	} {
		if got := age(c.d, testNow.Add(-c.d)); got != c.want {
			t.Errorf("age(%v) = %q, want %q", c.d, got, c.want)
		}
	}
	if got, want := age(40*24*time.Hour, testNow), testNow.Local().Format("2006-01-02"); got != want {
		t.Errorf("40 days: %q, want %q", got, want)
	}
}

// TestProjectFavourites is criteria 2.1 to 2.3 of plan 15: f stars the
// selected project and the store keeps it, first in the list with the
// selection on it; f again unstars it; Del forgets a favourite.
func TestProjectFavourites(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		dirs := []string{t.TempDir(), t.TempDir(), t.TempDir()}
		var list []input.Project
		for i, d := range dirs {
			list = append(list, input.Project{Path: d, Used: testNow.Add(-time.Duration(i) * time.Hour)})
		}
		store := storeWith(t, list)
		h := newHarness(t, Options{Providers: bothCLIs, Cwd: t.TempDir(), Projects: store})
		h.settle()
		h.send(key(tea.KeyTab, 0))
		h.send(key(tea.KeyTab, 0))
		h.send(key(tea.KeyDown, 0))
		h.send(key(tea.KeyDown, 0))
		h.send(typed('f'))
		got, err := store.Load()
		if err != nil || len(got) != 3 || got[0].Path != dirs[2] || !got[0].Favourite || got[1].Favourite || got[2].Favourite {
			t.Fatalf("after f: stored %v, %v", got, err)
		}
		if !h.m.projectList[0].Favourite || h.m.ui.ProjectSel != 0 || !h.m.ui.Projects[0].Favourite {
			t.Fatalf("after f: row %d, list %v", h.m.ui.ProjectSel, h.m.projectList)
		}
		field := h.m.path.Text()

		h.send(typed('f'))
		if got, _ := store.Load(); len(got) != 3 || got[0].Path != dirs[0] || got[2].Path != dirs[2] || got[2].Favourite || h.m.ui.ProjectSel != 2 {
			t.Fatalf("after f again: stored %v, row %d", got, h.m.ui.ProjectSel)
		}

		h.send(typed('f'))
		h.send(key(tea.KeyDelete, 0))
		if got, _ := store.Load(); len(got) != 2 || got[0].Path != dirs[0] || got[1].Path != dirs[1] {
			t.Fatalf("Del on a favourite: stored %v", got)
		}

		// The project field is untouched; in it, f is typed.
		if h.m.path.Text() != field {
			t.Fatalf("f in the list changed the project field: %q", h.m.path.Text())
		}
		h.send(key(tea.KeyTab, tea.ModShift))
		h.send(typed('f'))
		if h.m.ui.Field != 1 || h.m.path.Text() != field+"f" {
			t.Errorf("f in the project field: field %d, text %q", h.m.ui.Field, h.m.path.Text())
		}
	})
}
