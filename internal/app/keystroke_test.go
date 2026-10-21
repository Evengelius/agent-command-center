//go:build timing

package app

import (
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/Evengelius/agent-command-center/internal/input"
)

// The timing step (evaluation.md, Keystroke timing; seven tests since plan
// 22): test 6's measures, on the wall clock with the garbage collector on,
// run on their own, never in parallel with the other packages' tests:
//
//	go test -tags timing ./internal/app -run '^(TestKeystrokeToFrame|TestDraftKeystrokes|TestKeystrokeCost|TestLongSession|TestThinkingFrame|TestPasteToFrame|TestStreamToFrame)$' -count=1 -v
//
// Each starts with warmUp untimed keys, so the first frames' allocations and
// caches are not counted; the target stays 16 ms at the 95th percentile, and
// the maximum is reported. TestKeystrokeAllocs, in the ordinary suite, is the
// deterministic guard that a keystroke never redraws the whole draft.

// warmUp is the untimed keys before each measure.
const warmUp = 50

// TestKeystrokeToFrame is test 6, first measure: typing, moving and deleting
// in a 10,000-line draft, each key timed to its frame.
func TestKeystrokeToFrame(t *testing.T) {
	r, _ := copyRig(t, nil)
	h := r.h
	h.send(tea.PasteMsg{Content: draftLines(10_000)})
	if n := h.m.edit.Lines(); n < 10_000 {
		t.Fatalf("%d lines in the draft", n)
	}
	h.send(key(tea.KeyHome, tea.ModCtrl)) // the cursor at the top: the worst place to edit
	for i := 0; i < warmUp; i++ {
		frameTime(h, editKeys[i%len(editKeys)])
	}
	var ds []time.Duration
	for i := 0; i < 400; i++ {
		ds = append(ds, frameTime(h, editKeys[i%len(editKeys)]))
	}
	h.send(key(tea.KeyEnd, tea.ModCtrl)) // and at the bottom
	for i := 0; i < 400; i++ {
		ds = append(ds, frameTime(h, editKeys[i%len(editKeys)]))
	}
	p50, p95, worst := percentile(ds, 0.50), percentile(ds, 0.95), percentile(ds, 1)
	t.Logf("keystroke-to-frame, 10,000-line draft, %d keys after %d warm-up: p50 %v, p95 %v, max %v (target p95 ≤ %v)", len(ds), warmUp, p50, p95, worst, keystrokeTarget)
	if p95 > keystrokeTarget {
		overTarget(t, "p95 %v over the %v target", p95, keystrokeTarget)
	}

	// The same keys in the draft window, Ctrl+G (amendment 13).
	h.send(key('g', tea.ModCtrl))
	if !h.m.ui.DraftWin.Open {
		t.Fatal("Ctrl+G did not open the draft window")
	}
	ds = ds[:0]
	for _, place := range []tea.KeyPressMsg{key(tea.KeyHome, tea.ModCtrl), key(tea.KeyEnd, tea.ModCtrl)} {
		h.send(place)
		for i := 0; i < 400; i++ {
			ds = append(ds, frameTime(h, editKeys[i%len(editKeys)]))
		}
	}
	p50, p95, worst = percentile(ds, 0.50), percentile(ds, 0.95), percentile(ds, 1)
	t.Logf("keystroke-to-frame in the draft window, 10,000-line draft, %d keys: p50 %v, p95 %v, max %v (target p95 ≤ %v)", len(ds), p50, p95, worst, keystrokeTarget)
	if p95 > keystrokeTarget {
		overTarget(t, "draft window: p95 %v over the %v target", p95, keystrokeTarget)
	}
}

// TestDraftKeystrokes is step 4, check 4: typing in a 50,000-line draft
// while it is saved stays under test 6's keystroke target.
func TestDraftKeystrokes(t *testing.T) {
	d := input.Drafts{Dir: t.TempDir()}
	w := &wireClaude{}
	h := wireRig(t, w, Options{Drafts: &d})
	project := h.m.State().Context.Project
	// Saves at their real pace; the other timers stay off.
	h.m.opt.After = func(dur time.Duration, msg tea.Msg) tea.Cmd {
		if _, ok := msg.(draftSaveMsg); !ok {
			return nil
		}
		return func() tea.Msg { time.Sleep(dur); return msg }
	}
	h.send(tea.PasteMsg{Content: draftLines(50_000)})
	h.send(key(tea.KeyHome, tea.ModCtrl))
	gen := h.m.draftGen
	var ds []time.Duration
	for i := 0; i < warmUp+800; i++ {
		start := time.Now()
		_, cmd := h.m.Update(editKeys[i%len(editKeys)])
		_ = h.m.View()
		if i >= warmUp {
			ds = append(ds, time.Since(start))
		}
		h.exec(cmd)
		for drained := false; !drained; {
			select {
			case msg := <-h.msgs:
				h.deliver(msg)
			default:
				drained = true
			}
		}
		time.Sleep(3 * time.Millisecond) // a fast typist's pace, so saves run between keys
	}
	for deadline := time.Now().Add(5 * time.Second); h.m.draftTimer && time.Now().Before(deadline); {
		h.settle() // the last save is due
	}
	h.settle()
	saves := h.m.draftGen - gen
	p50, p95, worst := percentile(ds, 0.50), percentile(ds, 0.95), percentile(ds, 1)
	t.Logf("keystroke-to-frame, 50,000-line draft, %d keys after %d warm-up, %d saves: p50 %v, p95 %v, max %v (target p95 ≤ %v)", len(ds), warmUp, saves, p50, p95, worst, keystrokeTarget)
	if p95 > keystrokeTarget {
		overTarget(t, "p95 %v over the %v target", p95, keystrokeTarget)
	}
	if saves < 2 {
		t.Errorf("%d saves while typing; want the saves running", saves)
	}
	if got := draftFile(t, d, project); string(got) != h.m.edit.Text() {
		t.Errorf("stored draft (%d bytes) differs from the editor (%d bytes)", len(got), len(h.m.edit.Text()))
	}
}

// TestKeystrokeCost reports a keystroke's mean cost, from Update to the
// frame's string, in time and allocations, in 10,000- and 50,000-line
// drafts: the before and after of a change to the drawing. It judges
// nothing; the measures above and TestKeystrokeAllocs do.
func TestKeystrokeCost(t *testing.T) {
	for _, n := range []int{10_000, 50_000} {
		r, _ := copyRig(t, nil)
		h := r.h
		h.send(tea.PasteMsg{Content: draftLines(n)})
		h.send(key(tea.KeyHome, tea.ModCtrl))
		i := 0
		res := testing.Benchmark(func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				h.m.Update(editKeys[i%len(editKeys)])
				_ = h.m.View()
				i++
			}
		})
		t.Logf("%d-line draft: %d keys, %v per key, %d B and %d allocations per key", n, res.N, time.Duration(res.NsPerOp()), res.AllocedBytesPerOp(), res.AllocsPerOp())
	}
}
