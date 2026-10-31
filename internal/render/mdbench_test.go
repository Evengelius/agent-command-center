package render

import (
	"fmt"
	"strings"
	"testing"

	"github.com/Evengelius/agent-command-center/internal/event"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// Plan 71's Markdown cache (the owner's request of 2026-10-31), measured a
// frame at a time through one WrapCache and one Memo, as the program draws:
//
//	go test ./internal/render -run '^$' -bench Markdown -benchmem
//
// Zone B unchanged, streaming, scrolled and resized, the inspection, and a
// Markdown file's preview, at the sizes a long session reaches. Go's clock
// steps by 0.5 ms on the owner's machine (hometiming_test.go), too coarse
// for one frame: a frame's tail is measured as its cost at the largest size
// (the "-at-2000" streams) and by what it allocates.

// benchLine is TestStreamToFrame's answer's line i (internal/app's
// answerLine): a heading, a Go block, list items with strong, code and a
// link, and long lines of mixed widths.
func benchLine(i int) string {
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

// benchLines are a generator's first n lines.
func benchLines(n int, line func(int) string) []string {
	ls := make([]string, n)
	for i := range ls {
		ls[i] = line(i)
	}
	return ls
}

// benchDocLine is line i of hard-wrapped Markdown, as AGENTS.md is written:
// blocks of 48 lines, each a paragraph of 30 lines whose strong, emphasis
// and strikethrough run over its lines, a list whose items go on over two
// lines, a quote over two lines, a table and a fenced block, a blank line
// between them.
func benchDocLine(i int) string {
	j := i % 48
	switch {
	case j < 30:
		w := fmt.Sprintf("paragraph %d, line %d of words that a hard wrap broke here", i/48, j)
		switch j % 6 {
		case 0:
			return "**" + w + ","
		case 1:
			return w + "** and *an emphasis"
		case 2:
			return "that goes on* with `code` and ~~struck"
		case 3:
			return "words~~ " + w
		}
		return w
	case j == 30, j == 35, j == 38, j == 43, j == 47:
		return ""
	case j < 35: // two items, each over two lines
		if j%2 == 1 {
			return fmt.Sprintf("- **item %d that", j)
		}
		return "  goes on** over two lines"
	case j < 38:
		return []string{"> a quote's **strong", "> over its lines** ends"}[j-36]
	case j < 43:
		return []string{"| a | **b** |", "|---|---|", "| `c` | d |", "| e | f |"}[j-39]
	}
	return []string{"```go", "func main() { fmt.Println(\"**not bold**\") }", "```"}[j-44]
}

// benchLongLine is a line of 100 KB, its constructs within it.
var benchLongLine = strings.Repeat("a **long** line with `code` and *em* ", 2800)

// benchConv is a conversation of n exchanges, each the owner's message and
// a finished 20-line answer.
func benchConv(n int) *convBench {
	b := newConvBench("claude-code", "a-model")
	for k := range n {
		b.prompt(fmt.Sprintf("Question %d", k))
		b.msg(fmt.Sprintf("m%d", k), strings.Join(benchLines(20, func(i int) string { return benchLine(k*20 + i) }), "\n"))
		b.end("completed")
	}
	return b
}

// benchAnswered is the owner's message and its finished answer.
func benchAnswered(text string) *convBench {
	b := newConvBench("claude-code", "a-model")
	b.prompt("Answer.")
	b.msg("m1", text)
	return b
}

// benchFrames draws b.N frames of cb, a frame drawn before them, step
// changing the state or the view before each.
func benchFrames(b *testing.B, cb *convBench, ui UI, step func(n int, ui *UI)) {
	ui.Wrap, ui.Memo = NewWrapCache(), NewMemo()
	Draw(cb.s, ui)
	b.ReportAllocs()
	b.ResetTimer()
	for n := range b.N {
		if step != nil {
			step(n, &ui)
		}
		Draw(cb.s, ui)
	}
}

func BenchmarkMarkdownUnchanged(b *testing.B) {
	ui := UI{Width: 200, Height: 60, Focus: "B"}
	b.Run("answer-2000", func(b *testing.B) {
		benchFrames(b, benchAnswered(strings.Join(benchLines(2000, benchLine), "\n")), ui, nil)
	})
	b.Run("doc-2000", func(b *testing.B) {
		benchFrames(b, benchAnswered(strings.Join(benchLines(2000, benchDocLine), "\n")), ui, nil)
	})
	b.Run("conversation-500", func(b *testing.B) { benchFrames(b, benchConv(500), ui, nil) })
	b.Run("scrolled-500", func(b *testing.B) {
		held := ui
		held.ConvHold, held.ConvTop = true, 6000
		benchFrames(b, benchConv(500), held, nil)
	})
	b.Run("long-line", func(b *testing.B) { benchFrames(b, benchAnswered(benchLongLine), ui, nil) })
}

// benchStream streams lines into an answer, chunk bytes a delta, one delta a
// frame, from the start again once they are all in.
func benchStream(b *testing.B, lines []string, chunk int) {
	text := strings.Join(lines, "\n")
	var cb *convBench
	rest := ""
	ui := UI{Width: 200, Height: 60, Focus: "B", Wrap: NewWrapCache(), Memo: NewMemo()}
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		if rest == "" {
			b.StopTimer()
			cb = newConvBench("claude-code", "a-model")
			cb.prompt("Answer.")
			rest = text
			ui.Wrap, ui.Memo = NewWrapCache(), NewMemo()
			b.StartTimer()
		}
		k := min(chunk, len(rest))
		if chunk == 0 { // a line a delta
			k = strings.IndexByte(rest, '\n') + 1
			if k == 0 {
				k = len(rest)
			}
		}
		cb.ev(event.MessageDelta, event.Delta{MessageID: "m1", Block: "text", Text: rest[:k]})
		rest = rest[k:]
		Draw(cb.s, ui)
	}
}

// benchStreamAt streams one byte a frame into an answer of lines, from the
// same size again after 500 frames: the cost of a frame at that size.
func benchStreamAt(b *testing.B, lines []string) {
	text := strings.Join(lines, "\n") + "\n"
	var cb *convBench
	ui := UI{Width: 200, Height: 60, Focus: "B"}
	b.ReportAllocs()
	b.ResetTimer()
	for n := range b.N {
		if n%500 == 0 {
			b.StopTimer()
			cb = newConvBench("claude-code", "a-model")
			cb.prompt("Answer.")
			cb.ev(event.MessageDelta, event.Delta{MessageID: "m1", Block: "text", Text: text})
			ui.Wrap, ui.Memo = NewWrapCache(), NewMemo()
			Draw(cb.s, ui)
			b.StartTimer()
		}
		cb.ev(event.MessageDelta, event.Delta{MessageID: "m1", Block: "text", Text: "x"})
		Draw(cb.s, ui)
	}
}

func BenchmarkMarkdownStream(b *testing.B) {
	b.Run("answer-lines", func(b *testing.B) { benchStream(b, benchLines(2000, benchLine), 0) })
	b.Run("answer-at-2000", func(b *testing.B) { benchStreamAt(b, benchLines(2000, benchLine)) })
	b.Run("doc-lines", func(b *testing.B) { benchStream(b, benchLines(2000, benchDocLine), 0) })
	b.Run("doc-at-2000", func(b *testing.B) { benchStreamAt(b, benchLines(2000, benchDocLine)) })
	b.Run("doc-7-bytes", func(b *testing.B) { benchStream(b, benchLines(400, benchDocLine), 7) })
	b.Run("paragraph-at-1000", func(b *testing.B) {
		// One paragraph of 1,000 lines, its spans over them: a frame pairs
		// the paragraph it streams into.
		benchStreamAt(b, benchLines(1000, func(i int) string { return benchDocLine(i % 30) }))
	})
	b.Run("code-at-1000", func(b *testing.B) {
		lines := append([]string{"```go"}, benchLines(1000, func(i int) string {
			return fmt.Sprintf("\tx%d := `a raw string` + \"**not bold**\" // a comment %d", i, i)
		})...)
		benchStreamAt(b, lines)
	})
	b.Run("long-line-1k", func(b *testing.B) { benchStream(b, []string{benchLongLine}, 1024) })
}

func BenchmarkMarkdownResize(b *testing.B) {
	cb := benchAnswered(strings.Join(benchLines(2000, benchLine), "\n"))
	benchFrames(b, cb, UI{Width: 200, Height: 60, Focus: "B"}, func(n int, ui *UI) { ui.Width = 200 - 40*(n%2) })
}

func BenchmarkMarkdownScroll(b *testing.B) {
	held := UI{Width: 200, Height: 60, Focus: "B", ConvHold: true}
	benchFrames(b, benchConv(500), held, func(n int, ui *UI) { ui.ConvTop = n % 12000 })
}

func BenchmarkMarkdownInspect(b *testing.B) {
	cb := newConvBench("claude-code", "a-model")
	a := &state.Agent{Key: "a1", Label: "worker", Status: state.AgentCompleted}
	for k := range 100 {
		a.Messages = append(a.Messages, state.Entry{Role: "assistant", Text: strings.Join(benchLines(20, func(i int) string { return benchDocLine(k*20 + i) }), "\n")})
	}
	cb.s.Agents["a1"] = a
	benchFrames(b, cb, UI{Width: 200, Height: 60, Inspect: "a1"}, nil)
}

func BenchmarkMarkdownPreview(b *testing.B) {
	f := mdViewer("big.md", false)
	f.Lines = benchLines(5000, benchDocLine)
	cb := newConvBench("claude-code", "a-model")
	b.Run("scroll", func(b *testing.B) {
		benchFrames(b, cb, UI{Width: 200, Height: 60, Focus: "B", File: f}, func(n int, ui *UI) {
			ui.File.Top, ui.File.Cursor = n%4900, n%4900
		})
	})
	b.Run("toggle", func(b *testing.B) {
		benchFrames(b, cb, UI{Width: 200, Height: 60, Focus: "B", File: f}, func(n int, ui *UI) { ui.File.Source = n%2 == 1 })
	})
}
