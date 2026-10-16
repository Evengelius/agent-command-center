package app

import (
	"fmt"
	"sort"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/Evengelius/agent-command-center/internal/event"
	"github.com/Evengelius/agent-command-center/internal/provider/claude"
)

// Test 6's targets (proposal step 3, item 3): over target is a FAIL.
// overTarget fails the test for a missed timing target, except under -race:
// the work still runs there, for the race detector, and the time is logged.
func overTarget(t *testing.T, format string, args ...any) {
	t.Helper()
	if raceBuild {
		t.Logf("not judged under -race: "+format, args...)
		return
	}
	t.Errorf(format, args...)
}

const (
	keystrokeTarget = 16 * time.Millisecond  // 95th percentile, 10,000-line draft
	pasteTarget     = 100 * time.Millisecond // 50,000-line paste, to the frame
)

// draftLines is n lines of source-like text, with tabs and wide characters.
func draftLines(n int) string {
	var b strings.Builder
	for i := 0; i < n; i++ {
		fmt.Fprintf(&b, "\tline %05d: the quick brown fox jumps over the lazy dog · 日本\n", i)
	}
	return b.String()
}

// frameTime is the time one message takes from Update to a drawn frame.
func frameTime(h *harness, msg tea.Msg) time.Duration {
	start := time.Now()
	h.m.Update(msg)
	_ = h.m.View()
	return time.Since(start)
}

func percentile(ds []time.Duration, p float64) time.Duration {
	s := append([]time.Duration(nil), ds...)
	sort.Slice(s, func(i, j int) bool { return s[i] < s[j] })
	return s[min(int(p*float64(len(s))), len(s)-1)]
}

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
	keys := []tea.KeyPressMsg{
		key('x', 0), key(tea.KeyRight, 0), key(tea.KeyDown, 0), key(tea.KeyBackspace, 0),
		key('j', tea.ModCtrl), key(tea.KeyUp, 0), key(tea.KeyRight, tea.ModCtrl), key(tea.KeyDown, tea.ModShift),
	}
	var ds []time.Duration
	for i := 0; i < 400; i++ {
		ds = append(ds, frameTime(h, keys[i%len(keys)]))
	}
	h.send(key(tea.KeyEnd, tea.ModCtrl)) // and at the bottom
	for i := 0; i < 400; i++ {
		ds = append(ds, frameTime(h, keys[i%len(keys)]))
	}
	p50, p95, worst := percentile(ds, 0.50), percentile(ds, 0.95), percentile(ds, 1)
	t.Logf("keystroke-to-frame, 10,000-line draft, %d keys: p50 %v, p95 %v, max %v (target p95 ≤ %v)", len(ds), p50, p95, worst, keystrokeTarget)
	if p95 > keystrokeTarget {
		overTarget(t, "p95 %v over the %v target", p95, keystrokeTarget)
	}
}

// TestPasteToFrame is test 6, second measure: a 50,000-line paste into an
// empty draft, timed to its frame, ten times.
func TestPasteToFrame(t *testing.T) {
	r, _ := copyRig(t, nil)
	h := r.h
	text := draftLines(50_000)
	var ds []time.Duration
	for i := 0; i < 10; i++ {
		ds = append(ds, frameTime(h, tea.PasteMsg{Content: text}))
		if h.m.edit.Text() != text {
			t.Fatal("the paste was not kept byte for byte")
		}
		h.m.edit.SelectAll()
		h.m.edit.Backspace()
	}
	p50, worst := percentile(ds, 0.50), percentile(ds, 1)
	t.Logf("paste-to-frame, 50,000 lines (%d bytes), %d runs: p50 %v, max %v (target ≤ %v)", len(text), len(ds), p50, worst, pasteTarget)
	if worst > pasteTarget {
		overTarget(t, "max %v over the %v target", worst, pasteTarget)
	}
}

// streamTarget is criterion 7's target (proposal step 3; ASM-17): each delta
// of a streamed answer drawn within a frame.
const streamTarget = 16 * time.Millisecond

// answerLine is line i of a 2,000-line answer in the CLIs' Markdown subset:
// headings, lists, prose that wraps, wide characters, and fenced code with
// tabs.
func answerLine(i int) string {
	switch {
	case i%100 == 0:
		return fmt.Sprintf("## Section %d", i/100)
	case i%100 >= 40 && i%100 < 60:
		switch i % 100 {
		case 40:
			return "```go"
		case 59:
			return "```"
		}
		return fmt.Sprintf("\tif err := step(%d); err != nil {\treturn fmt.Errorf(\"step %%d: %%w\", %d, err) }", i, i)
	case i%7 == 0:
		return fmt.Sprintf("- item %d: **bold** and `code` and a link [doc](https://example.com/%d)", i, i)
	}
	return fmt.Sprintf("Line %d of the answer, long enough to wrap in zone B: the quick brown fox jumps over the lazy dog, 日本語のテキスト, and then some more words to fill the row.", i)
}

// TestStreamToFrame is criterion 7: a 2,000-line answer streamed one line per
// delta, each delta timed from Update to its frame, with our own renderer
// (the owner's decision of 2026-10-16: no Glamour).
func TestStreamToFrame(t *testing.T) {
	h, cli := startScript(t, claude.ProviderID)
	seq := uint64(1_000_000) // after anything the start applied
	ev := func(k event.Kind, p any) eventsMsg {
		seq++
		return eventsMsg{sess: h.m.sess, evs: []event.Event{{Seq: seq, Provider: cli.id, Kind: k, Origin: event.AI, Payload: p}}}
	}
	h.m.Update(ev(event.SessionSpawned, event.Spawned{}))
	h.m.Update(ev(event.SessionReady, event.Ready{Detail: "scripted"}))
	h.m.Update(ev(event.TurnStarted, nil))
	const lines = 2_000
	var ds []time.Duration
	var want strings.Builder
	for i := 0; i < lines; i++ {
		text := answerLine(i) + "\n"
		want.WriteString(text)
		ds = append(ds, frameTime(h, ev(event.MessageDelta, event.Delta{MessageID: "m1", Block: "text", Text: text})))
	}
	s := h.m.State()
	if n := len(s.Conversation); n == 0 || s.Conversation[n-1].Text != want.String() {
		t.Fatal("the streamed answer is not the conversation's last entry")
	}
	if f := plainText(h.m.View().Content); !strings.Contains(f, fmt.Sprintf("Line %d", lines-1)) {
		t.Errorf("the answer's end is not drawn:\n%s", f)
	}
	p50, p95, worst := percentile(ds, 0.50), percentile(ds, 0.95), percentile(ds, 1)
	t.Logf("delta-to-frame, %d-line answer, one line per delta: p50 %v, p95 %v, max %v (target p95 ≤ %v)", lines, p50, p95, worst, streamTarget)
	if p95 > streamTarget {
		overTarget(t, "p95 %v over the %v target", p95, streamTarget)
	}
}
