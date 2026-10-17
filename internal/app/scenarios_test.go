package app

import (
	"strings"
	"testing"
	"testing/synctest"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// The synthetic mock scenarios (files_test.go, in internal/provider/mock):
// cases no recording has, built from a recording with the documented reply
// put in. Each one goes through the whole pipeline, mapper to frame, and zone
// D shows the reason a value is missing, never a figure.
func TestSyntheticScenarios(t *testing.T) {
	for _, c := range []struct {
		file      string
		want, not []string
	}{
		{"claude-api-key-context-refused.jsonl",
			[]string{"5h N/A (API key: no subscription window)", "7d N/A (API key: no subscription window)", "context 3% of 1M (derived)", "errors 1"},
			[]string{"%  resets"}},
		{"codex-api-key.jsonl",
			[]string{"5h N/A (API key: ChatGPT sign-in needed)", "7d N/A (API key: ChatGPT sign-in needed)", "plan N/A (API key: ChatGPT sign-in needed)", "credits N/A (API key: ChatGPT sign-in needed)", "errors 0"},
			[]string{"%  resets", "plan plus"}},
		{"codex-no-5h-window.jsonl",
			[]string{"5h N/A (plan has no 5-hour limit)", "7d ", "1%  resets", "plan plus", "errors 0"},
			[]string{"5h ▱", "5h ▰"}},
		// Plan 6's simulated run: the failed command's last line with its
		// exit code (item 1); the requested model, and N/A without one (item
		// 4), marked "*" with its footnote (item 5), with its effort (item 6).
		{"codex-plan6-limits.jsonl",
			[]string{"✗ commandExecution · exit 1 · fatal: No configured push destination.", "/root/list_files  gpt-6-mini* low completed", "/root/create_fou… model N/A       completed", "* requested model · 0 ACTIVE · 2 DONE", "errors 0"},
			[]string{"commandExecution · On branch main", "commandExecution · exit 1 · On branch", "model N/A*"}},
	} {
		t.Run(c.file, func(t *testing.T) {
			h := mockDash(t, c.file)
			for _, size := range [][2]int{{120, 40}, {200, 60}} {
				h.send(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
				f := h.frame()
				for _, l := range strings.Split(f, "\n") {
					if w := lipgloss.Width(l); w > size[0] {
						t.Errorf("%dx%d: a row %d cells wide: %q", size[0], size[1], w, l)
					}
				}
				if size[0] < 200 {
					continue // zone D's rows are complete at 200 columns
				}
				for _, s := range c.want {
					if !strings.Contains(f, s) {
						t.Errorf("%dx%d: %q missing from:\n%s", size[0], size[1], s, f)
					}
				}
				for _, s := range c.not {
					if strings.Contains(f, s) {
						t.Errorf("%dx%d: %q shown:\n%s", size[0], size[1], s, f)
					}
				}
			}
		})
	}
}

// Plan 10, item 1, criterion 3: P9a replayed, the agent Codex names nowhere
// is shown with its spawn request's task, never as "unannounced agent", and
// its inspection gives the whole prompt.
func TestSpawnRequestLabel(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := mockDash(t, "codex-spawn-request.jsonl")
		for _, c := range []struct {
			w, h int
			row  string
		}{
			{120, 40, "✓ task: Count the files in the… gpt-5.6-luna* low completed"},
			{200, 60, "✓ task: Count the … gpt-5.6-luna* low completed"},
		} {
			h.send(tea.WindowSizeMsg{Width: c.w, Height: c.h})
			f := h.frame()
			if !strings.Contains(f, c.row) || strings.Contains(f, "unannounced agent") {
				t.Errorf("%dx%d: want %q and no \"unannounced agent\":\n%s", c.w, c.h, c.row, f)
			}
		}
		h.send(key(tea.KeyF6, 0))
		h.send(key(tea.KeyDown, 0))
		h.send(key(tea.KeyEnter, 0))
		f := h.frame()
		for _, s := range []string{"◀ back (Esc)   task: Count the files in the c… · gpt-5.6-luna (requested)", "started by orchestrator · Count the files in the current directory and reply with the number only."} {
			if !strings.Contains(f, s) {
				t.Errorf("inspection %q: %q missing from:\n%s", h.m.ui.Inspect, s, f)
			}
		}
	})
}
