package render

import (
	"fmt"
	"runtime"
	"slices"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"

	"github.com/Evengelius/agent-command-center/internal/event"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// Plan 71's Markdown cache (the owner's request of 2026-10-31): a text's
// lines, kinds and wraps kept by place from frame to frame, read again from
// where a grown text stops being sure (WrapCache.markdown, mdRead, wraps).
// Whatever the cache keeps, every result is the one a nil cache gives.

// checkDoc is text at place k through wc, checked against what a nil cache
// gives at width w in base: its lines, their kinds, wraps and rows' count.
func checkDoc(t *testing.T, wc *WrapCache, k mdKey, text string, split func(string) []string, w int, base *lipgloss.Style) *mdDoc {
	t.Helper()
	d := wc.markdown(k, text, split)
	lines := split(text)
	if !slices.Equal(d.lines, lines) {
		t.Fatalf("%q: lines %q, want %q", text, d.lines, lines)
	}
	want := mdLines(lines)
	for j := range want {
		if d.md[j] != want[j] {
			t.Fatalf("%q: line %d %q kind %+v, want %+v", text, j, lines[j], d.md[j], want[j])
		}
	}
	rows := wc.wraps(d, w, base)
	n := 0
	for j, l := range lines {
		if want := (*WrapCache)(nil).mdRanges(l, want[j], w, base); !slices.Equal(rows[j], want) {
			t.Fatalf("%q at %d: line %d %q wraps %v, want %v", text, w, j, l, rows[j], want)
		}
		n += len(rows[j])
	}
	if d.total != n {
		t.Fatalf("%q: %d rows counted, %d wrapped", text, d.total, n)
	}
	return d
}

// docPieces make random Markdown: every block kind, spans opening and
// closing over lines, fences with and without a language and a string over
// lines, tables with and without their rule, escapes, wide characters and
// tabs.
var docPieces = []string{
	"a", "word ", "**", "*", "_", "~~", "`", "``", "\\*", " x", "- ", "1. ", "> ", "> > ", "# ",
	"```", "```go\n", "~~~\n", "| a | b |\n", "|---|---|\n", "| c ", "---\n",
	"\n", "\n", "\n", "\n\n", "  ", "\t", "/* ", " */", "\"", "日本", "x := `raw", "end`",
}

// docText is random text of n pieces, from seed; every eleventh starts with
// a byte order mark.
func docText(seed, n int) string {
	var b strings.Builder
	if seed%11 == 0 {
		b.WriteString(bomMark)
	}
	x := uint32(seed)*2654435761 + 1
	for range n {
		x ^= x << 13
		x ^= x >> 17
		x ^= x << 5
		b.WriteString(docPieces[x%uint32(len(docPieces))])
	}
	return b.String()
}

// TestMarkdownCacheStream: random texts streamed in chunks of 1 to 40
// bytes, a frame a chunk, split as zone B's answers and as the inspection's
// messages, at widths that change: at each frame the lines, kinds and wraps
// a nil cache gives, the text read again from where it grew; a frame
// reading the same text twice reads it once; a text replaced, cut or
// changed inside read whole.
func TestMarkdownCacheStream(t *testing.T) {
	wc := NewWrapCache()
	sizes := []int{1, 2, 3, 5, 8, 13, 40}
	widths := []int{40, 17, 80}
	for n := range 400 {
		text := docText(n, 40+n%80)
		split := splitLines
		if n%3 == 0 {
			split = paragraphs
		}
		k := mdKey{panel: 'B', i: n % 7}
		for at, f := 0, 0; at < len(text); f++ {
			at = min(at+sizes[(n+f)%len(sizes)], len(text))
			w := widths[n%len(widths)]
			if f%17 == 16 {
				w = widths[(n+1)%len(widths)] // a frame at another width
			}
			d := checkDoc(t, wc, k, text[:at], split, w, stText)
			if again := wc.markdown(k, text[:at], split); again != d {
				t.Fatalf("%q read twice in a frame", text[:at])
			}
			wc.sweep()
		}
		for _, c := range []string{docText(n+1000, 30), text[:len(text)/2], strings.Replace(text, "a", "b", 1)} {
			checkDoc(t, wc, k, c, split, widths[n%len(widths)], stText)
			wc.sweep()
		}
	}
	if wc.reads[0] < 1000 || wc.reads[1] < 5000 {
		t.Errorf("read %d texts whole and %d from where they grew", wc.reads[0], wc.reads[1])
	}
}

// TestMarkdownCacheWrap: a wrap reading a line's cells alone, in the
// cache's scratch, breaks it where the cells mdShow builds for its drawing
// do, for random lines of every kind at widths from 1 to 80, the scratch
// reused from line to line; it allocates its rows alone, no style and no
// code colouring.
func TestMarkdownCacheWrap(t *testing.T) {
	var buf []cell
	for _, c := range []struct {
		src string
		k   mdLine
	}{
		{"a **strong** and *em* and `code` and a [link](u)", mdLine{kind: mdPlain}},
		{"\tx := `raw` + \"s\" // a comment", mdLine{kind: mdCode, lang: "go"}},
	} {
		mdWrap(buf, c.src, c.k, 80, stText)
		if n := testing.AllocsPerRun(20, func() { _, buf = mdWrap(buf, c.src, c.k, 80, stText) }); n > 1 {
			t.Errorf("%q: %v allocations a wrap", c.src, n)
		}
	}
	for n := range 300 {
		lines := strings.Split(docText(n, 30+n%60), "\n")
		ks := mdLines(lines)
		for j, l := range lines {
			for _, w := range []int{1, 7, 23, 80} {
				for _, base := range []*lipgloss.Style{stText, stMdEm} {
					sh := mdShow(l, ks[j], w, base)
					want := wrapCells(l, sh.cells, w, w-sh.hang.width())
					var got []rowRange
					got, buf = mdWrap(buf, l, ks[j], w, base)
					if !slices.Equal(got, want) {
						t.Fatalf("%q, %+v, at %d: wrapped %v, want %v", l, ks[j], w, got, want)
					}
				}
			}
		}
	}
}

// TestMarkdownCacheCases: a text at one place frame after frame, read from
// where it grew or whole, as each case says; then at another width and
// base. Each against what a nil cache gives.
func TestMarkdownCacheCases(t *testing.T) {
	for _, c := range []struct {
		name  string
		texts []string
		grown []bool // each text after the first: read from where the one before grew
	}{
		{"a span closing on the next line", []string{"**a", "**a\nb**"}, []bool{true}},
		{"a blank last line growing into the paragraph's", []string{"**a\n", "**a\nb**"}, []bool{true}},
		{"a table's rule arriving", []string{"| a | b |\n", "| a | b |\n|---|---|\n| c | d |"}, []bool{true}},
		{"a row widening its table", []string{"| a | b |\n|---|---|\n| c | d |\n", "| a | b |\n|---|---|\n| c | d |\n| a much longer cell | e |"}, []bool{true}},
		{"code streaming in a fence, a raw string open", []string{"```go\nx := 1\n", "```go\nx := 1\ny := `raw\n", "```go\nx := 1\ny := `raw\nstill`\n```\nafter **b**"}, []bool{true, true}},
		{"a fence closing", []string{"~~~\ncode **a**\n", "~~~\ncode **a**\n~~~\n**b"}, []bool{true}},
		{"an item going on", []string{"- **item\n", "- **item\n  goes on** here"}, []bool{true}},
		{"a quote going on", []string{"> a **b\n", "> a **b\n> c** d"}, []bool{true}},
		{"a byte order mark's heading", []string{bomMark + "# h\n", bomMark + "# h\ntext **b**"}, []bool{true}},
		{"the last line changed, not grown", []string{"**a\nb**", "**a\nc**"}, []bool{true}},
		{"a line inside changed", []string{"**a\nb\nc**", "**a\nX\nc**"}, []bool{false}},
		{"the line before the last changed", []string{"**a\n# h\nz", "**a\nb**\nz"}, []bool{false}},
		{"cut", []string{"a\nb\nc", "a\nb"}, []bool{false}},
		{"cut after a paragraph", []string{"x\n\ny\nz", "x\n"}, []bool{false}},
		{"replaced", []string{"**a\nb**", "# other"}, []bool{false}},
		{"back to the text before", []string{"**a", "**a\nb**", "**a"}, []bool{true, false}},
	} {
		t.Run(c.name, func(t *testing.T) {
			wc := NewWrapCache()
			k := mdKey{panel: 'B'}
			for j, text := range c.texts {
				before := wc.reads[1]
				checkDoc(t, wc, k, text, splitLines, 30, stText)
				if grew := wc.reads[1] > before; j > 0 && grew != c.grown[j-1] {
					t.Errorf("text %d %q read from where it grew %v, want %v", j, text, grew, c.grown[j-1])
				}
				wc.sweep()
			}
			last := c.texts[len(c.texts)-1]
			checkDoc(t, wc, k, last, splitLines, 12, stText)
			checkDoc(t, wc, k, last, splitLines, 12, stMdEm)
		})
	}
}

// TestMarkdownCacheFrames: every frame drawn through the cache is the frame
// drawn without it: zone B streaming a long answer 13 bytes a frame after a
// finished one, scrolled and at widths that change as it streams; the turn
// ended, the answer in the result's box; the inspection of an agent whose
// messages grow, followed and scrolled; a Markdown file's preview
// scrolled, its source and back.
func TestMarkdownCacheFrames(t *testing.T) {
	wc, memo := NewWrapCache(), NewMemo()
	check := func(step string, s *state.State, ui UI) {
		t.Helper()
		cached := ui
		cached.Wrap, cached.Memo = wc, memo
		if got, want := Draw(s, cached).String(), Draw(s, ui).String(); got != want {
			t.Fatalf("%s: the cached frame differs:\n%s\nwant\n%s", step, got, want)
		}
	}
	b := newConvBench("codex", "a-model")
	b.prompt("First.")
	b.msg("m0", strings.Join(benchLines(30, benchLine), "\n"))
	b.end("completed")
	b.prompt("Second.")
	text := strings.Join(benchLines(300, benchDocLine), "\n")
	widths := []int{120, 120, 80, 200}
	for n, at := 0, 0; at < len(text); n++ {
		k := min(13, len(text)-at)
		b.ev(event.MessageDelta, event.Delta{MessageID: "m1", Block: "text", Text: text[at : at+k]})
		at += k
		ui := UI{Width: widths[n/40%len(widths)], Height: 40, Focus: "B"}
		if n%9 == 4 {
			ui.ConvHold, ui.ConvTop = true, n%60
		}
		check(fmt.Sprintf("delta %d", n), b.s, ui)
	}
	b.ev(event.MessageCompleted, event.Completed{MessageID: "m1", Text: text})
	b.end("completed")
	for _, w := range []int{120, 80, 120} {
		check(fmt.Sprintf("the turn ended, at %d", w), b.s, UI{Width: w, Height: 40, Focus: "B"})
	}
	a := &state.Agent{Key: "a1", Label: "worker", Status: state.AgentRunning}
	b.s.Agents["a1"] = a
	for n := range 60 {
		if n%10 == 0 {
			a.Messages = append(a.Messages, state.Entry{Role: "assistant"})
		}
		a.Messages[len(a.Messages)-1].Text += benchDocLine(n) + "\n"
		check(fmt.Sprintf("inspection %d", n), b.s, UI{Width: 120, Height: 40, Inspect: "a1", InspectHold: n%7 == 3, InspectTop: n % 5})
	}
	f := mdViewer("doc.md", false)
	f.Lines = benchLines(500, benchDocLine)
	for n := range 12 {
		ui := UI{Width: 120, Height: 40, Focus: "B", File: f}
		ui.File.Top, ui.File.Cursor = n*37, n*37
		ui.File.Source = n%4 == 2
		check(fmt.Sprintf("preview %d", n), b.s, ui)
	}
}

// TestMarkdownCacheRetention: the cache keeps what the last two frames
// used: a long answer scrolled out of view is dropped, its wraps too; a
// long line's cells are not kept for the next; the file's kinds are kept
// while the viewer shows the file, as its preview or as its source, and
// dropped once it closes; the inspection's texts once it closes.
func TestMarkdownCacheRetention(t *testing.T) {
	wc, memo := NewWrapCache(), NewMemo()
	draw := func(s *state.State, ui UI) {
		ui.Wrap, ui.Memo = wc, memo
		Draw(s, ui)
	}
	kept := func(panel byte, i int) bool {
		for _, m := range []map[mdKey]*mdDoc{wc.texts, wc.textsOld} {
			for k := range m {
				if k.panel == panel && (i < 0 || k.i == i) {
					return true
				}
			}
		}
		return false
	}
	b := newConvBench("claude-code", "a-model")
	for k := range 40 {
		b.prompt(fmt.Sprintf("Question %d", k))
		b.msg(fmt.Sprintf("m%d", k), "a short **answer**")
		b.end("completed")
	}
	b.prompt("Long.")
	b.msg("long", strings.Join(benchLines(2000, benchLine), "\n"))
	b.end("completed")
	long := slices.IndexFunc(b.s.Conversation, func(e state.Entry) bool { return e.MessageID == "long" })
	ui := UI{Width: 200, Height: 60, Focus: "B"}
	draw(b.s, ui)
	if !kept('B', long) {
		t.Fatal("the long answer drawn is not kept")
	}
	ui.ConvHold, ui.ConvTop = true, 0
	for range 3 {
		draw(b.s, ui)
	}
	if kept('B', long) {
		t.Error("the long answer is kept two frames after it left the view")
	}
	if n := len(wc.cur) + len(wc.old); n > 500 {
		t.Errorf("%d wraps kept", n)
	}
	if n := len(wc.shown) + len(wc.shownOld); n > 200 {
		t.Errorf("%d lines kept as shown", n)
	}
	draw(benchAnswered(benchLongLine).s, UI{Width: 200, Height: 60, Focus: "B"})
	if c := cap(wc.scratch); c > maxScratch {
		t.Errorf("a long line's %d cells kept for the next", c)
	}
	f := mdViewer("doc.md", false)
	f.Lines = benchLines(500, benchDocLine)
	draw(b.s, UI{Width: 120, Height: 40, Focus: "B", File: f})
	if memo.file.md == nil {
		t.Fatal("the preview's kinds are not kept")
	}
	first := &memo.file.md[0]
	for range 3 {
		draw(b.s, UI{Width: 120, Height: 40, Focus: "B", File: f})
	}
	if memo.file.md == nil || &memo.file.md[0] != first {
		t.Error("the preview's kinds are read again while it is shown")
	}
	f.Source = true
	for range 3 {
		draw(b.s, UI{Width: 120, Height: 40, Focus: "B", File: f})
	}
	if memo.file.md == nil {
		t.Error("the file's kinds are dropped while its source is shown")
	}
	for range 2 {
		draw(b.s, UI{Width: 120, Height: 40, Focus: "B"})
	}
	if memo.file.md != nil {
		t.Error("the file's kinds are kept after the viewer closed")
	}
	b.s.Agents["a1"] = &state.Agent{Key: "a1", Label: "worker", Status: state.AgentCompleted,
		Messages: []state.Entry{{Role: "assistant", Text: "**done**"}}}
	draw(b.s, UI{Width: 120, Height: 40, Inspect: "a1"})
	if !kept('I', -1) {
		t.Fatal("the inspection's texts are not kept")
	}
	for range 2 {
		draw(b.s, UI{Width: 120, Height: 40, Focus: "B"})
	}
	if kept('I', -1) {
		t.Error("the inspection's texts are kept after it closed")
	}
}

// TestMarkdownCacheBytes guards what a frame allocates through the cache,
// which the machine's load does not change (as TestKeystrokeAllocs does for
// a key): an unchanged frame of a 2,000-line answer, one streaming into it,
// one resizing it, and one streaming a 100 KB line a kilobyte at a time;
// before the cache 3.5, 6.2, 44 and 28 MB (doc/plan71-evaluation.md, The
// Markdown cache). Under -race, which allocates otherwise, they are logged.
func TestMarkdownCacheBytes(t *testing.T) {
	answer := strings.Join(benchLines(2000, benchLine), "\n")
	perFrame := func(frames int, frame func(n int)) int {
		frame(0) // the frame before
		var a, z runtime.MemStats
		runtime.ReadMemStats(&a)
		for n := 1; n <= frames; n++ {
			frame(n)
		}
		runtime.ReadMemStats(&z)
		return int(z.TotalAlloc-a.TotalAlloc) / frames
	}
	ui := UI{Width: 200, Height: 60, Focus: "B", Wrap: NewWrapCache(), Memo: NewMemo()}
	still := benchAnswered(answer)
	stream := newConvBench("claude-code", "a-model")
	stream.prompt("Answer.")
	stream.ev(event.MessageDelta, event.Delta{MessageID: "m1", Block: "text", Text: answer + "\n"})
	long := newConvBench("claude-code", "a-model")
	long.prompt("Answer.")
	for _, c := range []struct {
		name   string
		frames int
		frame  func(n int)
		most   int
	}{
		{"unchanged", 20, func(int) { Draw(still.s, ui) }, 1_300_000},
		{"streaming", 20, func(int) {
			stream.ev(event.MessageDelta, event.Delta{MessageID: "m1", Block: "text", Text: "x"})
			Draw(stream.s, ui)
		}, 3_000_000},
		{"resized", 20, func(n int) {
			r := ui
			r.Width = 200 - 40*(n%2)
			Draw(still.s, r)
		}, 4_000_000},
		{"a long line streaming", 99, func(n int) {
			long.ev(event.MessageDelta, event.Delta{MessageID: "m1", Block: "text", Text: benchLongLine[n*1024 : (n+1)*1024]})
			Draw(long.s, ui)
		}, 10_000_000},
	} {
		ui.Wrap, ui.Memo = NewWrapCache(), NewMemo()
		switch got := perFrame(c.frames, c.frame); {
		case raceBuild:
			t.Logf("%s: %d bytes a frame under -race, not judged (at most %d without it)", c.name, got, c.most)
		case got > c.most:
			t.Errorf("%s: %d bytes a frame, over %d", c.name, got, c.most)
		default:
			t.Logf("%s: %d bytes a frame (at most %d)", c.name, got, c.most)
		}
	}
}
