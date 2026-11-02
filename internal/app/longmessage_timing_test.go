//go:build timing

package app

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/Evengelius/agent-command-center/internal/event"
	"github.com/Evengelius/agent-command-center/internal/input"
	"github.com/Evengelius/agent-command-center/internal/provider/claude"
)

// The long-message path's measures (plan 73, criteria 73.25 and 73.26), in the timing
// step, doc/evaluation.md: run there alone, judged only there.

// chipWarmUp is the keys or deltas drawn before a measure starts.
const chipWarmUp = 50

// TestLongMessageToFrame is criterion 73.25: a 50,000-line chip in zone B's view and
// 400 answer deltas arriving beside it, each delta timed to its frame.
func TestLongMessageToFrame(t *testing.T) {
	h, cli := startScript(t, claude.ProviderID)
	seq := uint64(1_000_000) // after anything the start applied
	ev := func(k event.Kind, p any) eventsMsg {
		seq++
		return eventsMsg{sess: h.m.sess, evs: []event.Event{{Seq: seq, Provider: cli.id, Kind: k, Origin: event.AI, Payload: p}}}
	}
	h.m.Update(ev(event.SessionSpawned, event.Spawned{}))
	h.m.Update(ev(event.SessionReady, event.Ready{Detail: "scripted"}))
	h.m.Update(tea.WindowSizeMsg{Width: 200, Height: 60})
	text := chipText()
	h.m.Update(ev(event.UserMessage, event.User{Text: text}))
	h.m.Update(ev(event.TurnStarted, nil))
	const deltas = 400
	var ds []time.Duration
	for i := range deltas + chipWarmUp {
		d := frameTime(h, ev(event.MessageDelta, event.Delta{MessageID: "m1", Block: "text", Text: fmt.Sprintf("word%d ", i%10)}))
		if i >= chipWarmUp {
			ds = append(ds, d)
		}
	}
	if f := h.frame(); !strings.Contains(f, "▣ Long message · 50,000 lines") || strings.Contains(f, "line 00001") {
		t.Fatalf("the chip is not what the frame shows:\n%s", f)
	}
	p50, p95, worst := percentile(ds, 0.50), percentile(ds, 0.95), percentile(ds, 1)
	t.Logf("a 50,000-line chip in view (%d bytes), %d deltas to their frames at 200x60: p50 %v, p95 %v, max %v (target p95 ≤ %v)", len(text), len(ds), p50, p95, worst, keystrokeTarget)
	if p95 > keystrokeTarget {
		overTarget(t, "a chip in view: p95 %v over the %v target", p95, keystrokeTarget)
	}
}

// TestLongPasteKeys is criterion 73.25: keys around a 50,000-line chip in the composer,
// each timed to its frame: the motions, which visit no hidden byte, the edits, which
// copy the text as every draft's do (design D11), and the chip's deletion and undo.
func TestLongPasteKeys(t *testing.T) {
	r, _ := copyRig(t, nil)
	h := r.h
	h.send(tea.WindowSizeMsg{Width: 200, Height: 60})
	block := chipText()
	h.m.edit.Insert("typed before\nab ")
	h.send(tea.PasteMsg{Content: block})
	h.m.edit.Insert(" cd\nlast line")
	text := h.m.edit.Text()
	rs := h.m.edit.Ranges()
	if len(rs) != 1 || !strings.Contains(h.frame(), "▣ Long message · 50,000 lines") {
		t.Fatalf("no chip: %+v", rs)
	}
	to := rs[0].To
	measure := func(name string, keys []tea.Msg, n int) {
		t.Helper()
		var ds []time.Duration
		for i := range chipWarmUp + n {
			if d := frameTime(h, keys[i%len(keys)]); i >= chipWarmUp {
				ds = append(ds, d)
			}
		}
		p50, p95, worst := percentile(ds, 0.50), percentile(ds, 0.95), percentile(ds, 1)
		t.Logf("around a 50,000-line chip, %s, %d keys at 200x60: p50 %v, p95 %v, max %v (target p95 ≤ %v)", name, len(ds), p50, p95, worst, keystrokeTarget)
		if p95 > keystrokeTarget {
			overTarget(t, "%s: p95 %v over the %v target", name, p95, keystrokeTarget)
		}
		if got := h.m.edit.Text(); got != text || len(h.m.edit.Ranges()) != 1 {
			t.Fatalf("%s: the text or its chip changed: %d bytes, %+v", name, len(got), h.m.edit.Ranges())
		}
	}
	h.m.edit.Select(to, to)
	measure("the motions", []tea.Msg{
		key(tea.KeyLeft, 0), key(tea.KeyRight, 0), key(tea.KeyUp, 0), key(tea.KeyDown, 0), key(tea.KeyHome, 0), key(tea.KeyEnd, 0),
		key(tea.KeyLeft, tea.ModCtrl), key(tea.KeyRight, tea.ModCtrl), key(tea.KeyLeft, tea.ModShift), key(tea.KeyRight, tea.ModShift),
		key(tea.KeyHome, tea.ModCtrl), key(tea.KeyEnd, tea.ModCtrl),
	}, 480)
	h.m.edit.Select(to, to)
	measure("typing beside it", []tea.Msg{typed('x'), key(tea.KeyBackspace, 0)}, 400)
	h.m.edit.Select(to, to)
	measure("its whole deletion and its undo", []tea.Msg{key(tea.KeyBackspace, 0), key('z', tea.ModCtrl)}, 200)
}

// TestLongMessageView is criterion 73.25 and P5: the full view of a 50,000-line message,
// its keys timed to their frames, and y's key in Update, its copy running off the event
// loop as every copy's does.
func TestLongMessageView(t *testing.T) {
	r, _ := copyRig(t, nil)
	h := r.h
	text := chipText()
	r.emit(event.UserMessage, event.User{Text: text})
	h.settle()
	for _, size := range []tea.WindowSizeMsg{{Width: 120, Height: 40}, {Width: 208, Height: 59}} {
		h.send(size)
		h.draw(key(tea.KeyUp, 0))
		opened := frameTime(h, key(tea.KeyEnter, 0))
		if f := h.frame(); h.m.ui.ViewEntry == 0 || !strings.Contains(f, "50,000 lines") || !strings.Contains(f, "line 00000") {
			t.Fatalf("the full view did not open:\n%s", f)
		}
		t.Logf("the full view of 50,000 lines at %dx%d, its one wrap, on opening: %v", size.Width, size.Height, opened)
		keys := []tea.Msg{key(tea.KeyPgDown, 0), key(tea.KeyDown, 0), key(tea.KeyPgDown, 0), key(tea.KeyUp, 0), key(tea.KeyPgUp, 0), key(tea.KeyEnd, 0), key(tea.KeyHome, 0)}
		var ds []time.Duration
		for i := range chipWarmUp + 200 {
			if d := frameTime(h, keys[i%len(keys)]); i >= chipWarmUp {
				ds = append(ds, d)
			}
		}
		p50, p95, worst := percentile(ds, 0.50), percentile(ds, 0.95), percentile(ds, 1)
		t.Logf("the full view of 50,000 lines at %dx%d, %d keys: p50 %v, p95 %v, max %v (target p95 ≤ %v)", size.Width, size.Height, len(ds), p50, p95, worst, keystrokeTarget)
		if p95 > keystrokeTarget {
			overTarget(t, "the full view at %dx%d: p95 %v over the %v target", size.Width, size.Height, p95, keystrokeTarget)
		}
		var ys []time.Duration
		for range 100 {
			start := time.Now()
			h.m.Update(typed('y'))
			ys = append(ys, time.Since(start))
		}
		p50, p95, worst = percentile(ys, 0.50), percentile(ys, 0.95), percentile(ys, 1)
		t.Logf("y's key in Update at %dx%d, %d presses: p50 %v, p95 %v, max %v (target ≤ %v)", size.Width, size.Height, len(ys), p50, p95, worst, keystrokeTarget)
		if worst > keystrokeTarget {
			overTarget(t, "y's key at %dx%d: max %v over the %v target", size.Width, size.Height, worst, keystrokeTarget)
		}
		// What the event loop then does with the copy's sequence: Bubble Tea's copy of it
		// into its output buffer (tea.go, RawMsg, p.execute); the write is off the loop.
		seq := input.OSC52(text)
		start := time.Now()
		var buf bytes.Buffer
		buf.WriteString(fmt.Sprint(any(seq)))
		t.Logf("y's sequence, %d bytes: its copy on the event loop %v", buf.Len(), time.Since(start))
		h.send(key(tea.KeyEscape, 0))
		h.send(key(tea.KeyEscape, 0))

		// The search's bar kept over the same view: Tab on a match behind the chip. Its
		// lighting reads the message each frame, as it reads any entry's (plan 66, design
		// D8): recorded, with no budget of this plan's.
		h.send(key('f', tea.ModCtrl))
		for _, k := range typedText("line 40000") {
			h.m.Update(k)
		}
		h.draw(key(tea.KeyTab, 0))
		if h.m.ui.ViewEntry == 0 || !h.m.ui.Search.Open {
			t.Fatalf("Tab on the match: view %d, bar %v", h.m.ui.ViewEntry, h.m.ui.Search.Open)
		}
		ds = ds[:0]
		for i := range 20 + 40 {
			if d := frameTime(h, key(tea.KeyUp, 0)); i >= 20 {
				ds = append(ds, d)
			}
		}
		t.Logf("that view with the search's bar open at %dx%d, %d keys: p50 %v, p95 %v, max %v", size.Width, size.Height, len(ds), percentile(ds, 0.50), percentile(ds, 0.95), percentile(ds, 1))
		h.send(key(tea.KeyEscape, 0))
		h.send(key(tea.KeyEscape, 0))
		h.send(key(tea.KeyEscape, 0))
	}
}
