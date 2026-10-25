//go:build timing

package app

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/Evengelius/agent-command-center/internal/input"
)

// bigSides is a file of n source-like lines and the same with one line in
// ten changed.
func bigSides(n int) (old, cur []byte) {
	var a, b strings.Builder
	for i := 1; i <= n; i++ {
		line := fmt.Sprintf("\tline %05d: the quick brown fox jumps over the lazy dog, return a + b\n", i)
		a.WriteString(line)
		if i%10 == 0 {
			line = fmt.Sprintf("\tline %05d: the quick brown fox jumps over the lazy cat, return a + b + 10\n", i)
		}
		b.WriteString(line)
	}
	return []byte(a.String()), []byte(b.String())
}

// TestDiffToFrame is plan 44's criterion 44.13: a large diff's scrolling,
// each key timed to its frame, at 120x40 and the acc tab's 208x59, unified
// and widened side by side. The file has 20,000 lines, one in ten changed:
// 2,000 hunks, compared once when it opens; the keys are ↓ ↑ PgDn PgUp, n
// and N, and the wheel.
func TestDiffToFrame(t *testing.T) {
	old, cur := bigSides(20_000)
	g := &fakeGit{snap: input.GitSnapshot{Top: `C:\work\big`, Branch: input.GitBranch{Name: "main"},
		Entries: []input.GitEntry{{X: ' ', Y: 'M', Path: "big.go", RepoPath: "big.go"}}},
		sides: map[string]input.DiffPair{"big.go false": {Old: input.DiffSide{Name: "index", ID: "b1", Data: old},
			New: input.DiffSide{Name: "working tree", Data: cur}}}}
	h := newHarness(t, Options{Config: Config{Mock: filepath.Join("..", "..", "testdata", "mock", "claude-subagent-approval.jsonl")}, Git: g})
	h.settle()
	keys := []tea.Msg{key(tea.KeyDown, 0), key(tea.KeyDown, 0), key(tea.KeyPgDown, 0), typed('n'), typed('n'), key(tea.KeyUp, 0),
		key(tea.KeyPgUp, 0), typed('N'), tea.MouseWheelMsg{X: 60, Y: 20, Button: tea.MouseWheelDown}, typed('n')}
	for _, size := range []tea.WindowSizeMsg{{Width: 120, Height: 40}, {Width: 208, Height: 59}} {
		h.send(size)
		for _, c := range h.m.showSource() {
			h.exec(c)
		}
		h.settle()
		h.send(key(tea.KeyHome, 0))
		h.send(key(tea.KeyDown, 0))
		start := time.Now()
		h.send(key(tea.KeyEnter, 0))
		opened := time.Since(start)
		if v := h.m.ui.Diff; !v.Open || v.Layout == nil || len(v.Hunks) != 2000 {
			t.Fatalf("%dx%d: the diff open %v, %d hunks, note %q", size.Width, size.Height, v.Open, len(v.Hunks), v.Note)
		}
		t.Logf("%dx%d: big.go opened, compared and laid out in %v (in a command, the frames going on)", size.Width, size.Height, opened)
		for _, wide := range []bool{false, true} {
			if h.m.ui.Diff.Wide != wide {
				h.send(typed('f'))
			}
			var ds []time.Duration
			for i := 0; i < warmUp+400; i++ {
				if d := frameTime(h, keys[i%len(keys)]); i >= warmUp {
					ds = append(ds, d)
				}
			}
			if h.m.ui.Diff.Top == 0 {
				t.Fatalf("%dx%d: the keys did not scroll", size.Width, size.Height)
			}
			p50, p95, worst := percentile(ds, 0.50), percentile(ds, 0.95), percentile(ds, 1)
			t.Logf("diff view, wide %v (side by side %v), at %dx%d, %d keys after %d warm-up: p50 %v, p95 %v, max %v (target p95 ≤ %v)",
				wide, h.m.diffInner() >= 150, size.Width, size.Height, len(ds), warmUp, p50, p95, worst, keystrokeTarget)
			if p95 > keystrokeTarget {
				overTarget(t, "diff view at %dx%d, wide %v: p95 %v over the %v target", size.Width, size.Height, wide, p95, keystrokeTarget)
			}
		}
		h.send(key(tea.KeyEscape, 0))
	}
}
