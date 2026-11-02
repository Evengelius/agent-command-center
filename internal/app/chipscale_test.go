//go:build timing && windows

package app

import (
	"slices"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/Evengelius/agent-command-center/internal/input"
	"github.com/Evengelius/agent-command-center/internal/render"
)

// perCall is the median of 200 samples of one call of f, each sample 20 calls on the
// performance counter, Go's clock stepping by 0.5 ms here (doc/evaluation.md, plan 51).
func perCall(f func()) time.Duration {
	const calls = 20
	samples := make([]time.Duration, 200)
	for i := range samples {
		start := perfCount()
		for range calls {
			f()
		}
		samples[i] = perfSince(start) / calls
	}
	slices.Sort(samples)
	return samples[len(samples)/2]
}

// TestChipScale is criterion 73.26: the view and every motion around chips of 5 KB,
// 500 KB and 5 MB, of many lines and of one line, the same text around them, so that only
// the hidden size differs. It records how the cost changes; it judges nothing, a ratio of
// five from 5 KB to 5 MB being a provisional diagnostic (doc/plan73-evaluation.md).
func TestChipScale(t *testing.T) {
	r, _ := copyRig(t, nil)
	h := r.h
	h.send(tea.WindowSizeMsg{Width: 200, Height: 60})
	motions := []struct {
		name  string
		m     input.Motion
		place string // where it starts: before or after the chip, on the line above or below it
	}{
		{"Left", input.Left, "after"}, {"Right", input.Right, "before"}, {"Up", input.Up, "below"}, {"Down", input.Down, "above"},
		{"Home", input.LineStart, "after"}, {"End", input.LineEnd, "before"}, {"WordLeft", input.WordLeft, "after"}, {"WordRight", input.WordRight, "before"},
	}
	sizes := []int{5_000, 500_000, 5_000_000}
	costs := map[string][]time.Duration{}
	var names []string
	add := func(name string, d time.Duration) {
		if _, seen := costs[name]; !seen {
			names = append(names, name)
		}
		costs[name] = append(costs[name], d)
	}
	for _, shape := range []string{"many lines", "one line"} {
		for _, size := range sizes {
			block := strings.Repeat("0123456789", size/10)
			if shape == "many lines" {
				const l = "the quick brown fox jumps over the lazy dog, a block's line"
				block = strings.TrimSuffix(strings.Repeat(l+"\n", size/(len(l)+1)), "\n")
			}
			e := &h.m.edit
			e.Clear()
			e.Insert("typed before\nab ")
			h.m.Update(tea.PasteMsg{Content: block})
			e.Insert(" cd\nlast line")
			rs := e.Ranges()
			if len(rs) != 1 || rs[0].To-rs[0].From != len(block) {
				t.Fatalf("%s %d: ranges %+v", shape, size, rs)
			}
			places := map[string]int{"before": rs[0].From, "after": rs[0].To, "above": 3, "below": len(e.Text()) - 4}
			width := render.MessageWidth(h.m.State(), h.m.ui)
			add(shape+" LongMessage", perCall(func() { render.LongMessage(block, width) }))
			e.Select(rs[0].To, rs[0].To)
			add(shape+" view", perCall(func() { h.m.editorView() }))
			for _, mv := range motions {
				at := places[mv.place]
				add(shape+" "+mv.name, perCall(func() { e.Select(at, at); e.Move(mv.m, false) }))
				add(shape+" Shift+"+mv.name, perCall(func() { e.Select(at, at); e.Move(mv.m, true) }))
				if got := e.Cursor(); got > rs[0].From && got < rs[0].To {
					t.Fatalf("%s %d %s: the cursor %d inside %+v", shape, size, mv.name, got, rs[0])
				}
			}
		}
	}
	for _, name := range names {
		c := costs[name]
		t.Logf("%-28s 5 KB %10v  500 KB %10v  5 MB %10v  5 MB / 5 KB %.2f", name, c[0], c[1], c[2], float64(c[2])/float64(max(c[0], 1)))
	}
}

// TestChipCount records what the number of chips costs (plan 73, P2: ranges adjust in
// their number): the view, a motion over one and a key to its frame, around 1, 10 and 100
// chips of 50 KB each with typed text between them. It judges nothing.
func TestChipCount(t *testing.T) {
	r, _ := copyRig(t, nil)
	h := r.h
	h.send(tea.WindowSizeMsg{Width: 200, Height: 60})
	const l = "the quick brown fox jumps over the lazy dog, a block's line"
	block := strings.TrimSuffix(strings.Repeat(l+"\n", 50_000/(len(l)+1)), "\n")
	for _, n := range []int{1, 10, 100} {
		e := &h.m.edit
		e.Clear()
		for range n {
			h.m.Update(tea.PasteMsg{Content: block})
			e.Insert(" typed text ")
		}
		rs := e.Ranges()
		if len(rs) != n {
			t.Fatalf("%d chips: ranges %d", n, len(rs))
		}
		at := rs[n/2].To
		view := perCall(func() { h.m.editorView() })
		motion := perCall(func() { e.Select(at, at); e.Move(input.Left, false) })
		frames := make([]time.Duration, 100)
		for i := range frames {
			e.Select(at, at)
			start := perfCount()
			h.m.Update(key(tea.KeyLeft, 0))
			_ = h.m.View()
			frames[i] = perfSince(start)
		}
		slices.Sort(frames)
		t.Logf("%3d chips, %d bytes: the view %v, a motion over a chip %v, a key to its frame p50 %v, max %v", n, len(e.Text()), view, motion, frames[50], frames[99])
	}
}
