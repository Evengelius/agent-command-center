//go:build timing

package app

import (
	"path/filepath"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/Evengelius/agent-command-center/internal/input"
	"github.com/Evengelius/agent-command-center/internal/render"
)

// TestMessageToFrame is plan 45's criterion 45.16: typing in Source
// Control's message field, each key timed to its frame, at 120x40 and the
// acc tab's 208x59, with Sixel on, so the controls' images are part of
// every frame's signature. The keys write a commit message as one does:
// characters, spaces, Backspace, a new line, the arrows; the field grows to
// its three rows, then scrolls.
func TestMessageToFrame(t *testing.T) {
	g := &fakeGit{snap: input.GitSnapshot{Top: `C:\work\demo`, Branch: input.GitBranch{Name: "main", Upstream: "origin/main"},
		Entries: []input.GitEntry{
			{X: 'M', Y: ' ', Path: "internal/app/commit.go", RepoPath: "internal/app/commit.go"},
			{X: ' ', Y: 'M', Path: "internal/render/commit.go", RepoPath: "internal/render/commit.go"},
			{X: '?', Y: '?', Path: "notes/", RepoPath: "notes/"},
		}}}
	h := newHarness(t, Options{Config: Config{Mock: filepath.Join("..", "..", "testdata", "mock", "claude-subagent-approval.jsonl")}, Git: &fakeWrite{fakeGit: g}})
	h.settle()
	h.send(PrimaryDeviceAttributesEvent{61, 4, 6, 7, 14})
	var keys []tea.Msg
	for _, r := range "Fix the commit button" {
		keys = append(keys, typed(r))
	}
	keys = append(keys, key(tea.KeyBackspace, 0), typed('n'), key(tea.KeyLeft, 0), key(tea.KeyRight, 0), key(tea.KeyEnter, 0))
	for _, size := range []tea.WindowSizeMsg{{Width: 120, Height: 40}, {Width: 208, Height: 59}} {
		h.send(size)
		for _, c := range h.m.showSource() {
			h.exec(c)
		}
		h.settle()
		h.send(key(tea.KeyHome, 0))
		h.send(key(tea.KeyUp, 0))
		h.send(key(tea.KeyUp, 0))
		if h.m.ui.Source.Place != render.PlaceMsg {
			t.Fatalf("%dx%d: the field has not the keys: %q", size.Width, size.Height, h.m.ui.Source.Place)
		}
		var ds []time.Duration
		for i := 0; i < warmUp+400; i++ {
			if d := frameTime(h, keys[i%len(keys)]); i >= warmUp {
				ds = append(ds, d)
			}
		}
		if h.m.msg.Lines() < 4 || !h.m.screen.MsgBox.Shown {
			t.Fatalf("%dx%d: %d lines typed, the box's images shown %v", size.Width, size.Height, h.m.msg.Lines(), h.m.screen.MsgBox.Shown)
		}
		p50, p95, worst := percentile(ds, 0.50), percentile(ds, 0.95), percentile(ds, 1)
		t.Logf("message field at %dx%d, %d keys after %d warm-up, %d lines typed: p50 %v, p95 %v, max %v (target p95 ≤ %v)",
			size.Width, size.Height, len(ds), warmUp, h.m.msg.Lines(), p50, p95, worst, keystrokeTarget)
		if p95 > keystrokeTarget {
			overTarget(t, "message field at %dx%d: p95 %v over the %v target", size.Width, size.Height, p95, keystrokeTarget)
		}
		h.send(key(tea.KeyEscape, 0))
		h.m.msg.Clear()
	}
}
