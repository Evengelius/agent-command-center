package app

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"testing/synctest"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/Evengelius/agent-command-center/internal/provider"
)

// checkFrame fails unless the frame is exactly h lines, none wider than w,
// with no control character once the style escapes are removed: a control
// written to the terminal breaks the frame there, whatever the test sees.
func checkFrame(t *testing.T, frame string, w, h int) {
	t.Helper()
	lines := strings.Split(frame, "\n")
	if len(lines) != h {
		t.Fatalf("%d lines, want %d:\n%s", len(lines), h, plainText(frame))
	}
	for i, l := range lines {
		if n := lipgloss.Width(l); n > w {
			t.Fatalf("line %d is %d columns wide, terminal %d: %q", i+1, n, w, plainText(l))
		}
		for _, r := range plainText(l) {
			if r < 0x20 || r == 0x7f || r >= 0x80 && r <= 0x9f {
				t.Fatalf("line %d has control %U: %q", i+1, r, plainText(l))
			}
		}
	}
}

// exactWidth fails unless every row of the frame is exactly w columns wide
// (plan 43, criterion 43.2).
func exactWidth(t *testing.T, frame string, w int) {
	t.Helper()
	for i, l := range strings.Split(frame, "\n") {
		if n := lipgloss.Width(l); n != w {
			t.Fatalf("line %d is %d columns wide, terminal %d: %q", i+1, n, w, plainText(l))
		}
	}
}

// TestFrameInvariants replays every mock file at widths between and beyond
// the golden frames' and checks every frame drawn on the way, then the last
// state at three heights, the Explorer's panel hidden and shown (plan 43,
// criterion 43.2: docked, in zone B's place, or under 100 columns), each
// row exactly the screen's width. Windows Terminal, maximised at 156
// columns, showed an agent's multi-line summary breaking zone C, a width no
// golden frame used (plan 4, step 3, criterion 9).
func TestFrameInvariants(t *testing.T) {
	files, err := filepath.Glob(filepath.Join("..", "..", "testdata", "mock", "*.jsonl"))
	if err != nil || len(files) == 0 {
		t.Fatalf("no mock file: %v", err)
	}
	for _, file := range files {
		for _, w := range []int{80, 96, 100, 120, 140, 156, 160, 180, 200, 240} {
			t.Run(fmt.Sprintf("%s-%d", filepath.Base(file), w), func(t *testing.T) {
				synctest.Test(t, func(t *testing.T) {
					h := newHarness(t, Options{
						Config:    Config{Mock: file},
						Providers: []provider.Provider{spawnTrap{t}},
						Cwd:       t.TempDir(),
						Git:       &fakeGit{},
					})
					hgt, frames := 40, 0
					h.deliver(tea.WindowSizeMsg{Width: w, Height: hgt})
					h.check = func(frame string) {
						frames++
						checkFrame(t, frame, w, hgt)
					}
					h.settle()
					if frames == 0 {
						t.Fatal("no frame drawn during the replay")
					}
					for _, shown := range []bool{false, true} {
						if shown {
							h.send(key(tea.KeyF3, 0))
							if !h.m.ui.Explorer.Open {
								t.Fatal("F3 showed no Explorer")
							}
						}
						for _, hgt := range []int{24, 40, 60} {
							h.check = func(frame string) { checkFrame(t, frame, w, hgt); exactWidth(t, frame, w) }
							h.send(tea.WindowSizeMsg{Width: w, Height: hgt})
						}
					}
				})
			})
		}
	}
}
