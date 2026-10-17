package app

import (
	"strings"
	"testing"

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
