package render

import (
	"fmt"
	"math/rand/v2"
	"slices"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"charm.land/lipgloss/v2"

	"github.com/Evengelius/agent-command-center/internal/state"
)

// wrapBefore is wrap as it was before plan 5, item 11 split it into
// paragraphs and wrapPara, with the one change item 11 made: a rune wider
// than the row takes a row alone, where wrap looped for ever.
func wrapBefore(text string, w int) []string {
	if w < 1 {
		w = 1
	}
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\t", "    ")
	var out []string
	for _, para := range strings.Split(text, "\n") {
		para = strings.Map(func(r rune) rune {
			if r < 0x20 || r == 0x7f {
				return -1
			}
			return r
		}, para)
		if para == "" {
			out = append(out, "")
			continue
		}
		cur, cw := "", 0
		for _, word := range strings.SplitAfter(para, " ") {
			ww := textWidth(word)
			if cw+ww <= w {
				cur, cw = cur+word, cw+ww
				continue
			}
			if cur != "" {
				out = append(out, strings.TrimRight(cur, " "))
				cur, cw = "", 0
			}
			for textWidth(word) > w {
				head := cut(word, w)
				if head == "" {
					_, n := utf8.DecodeRuneInString(word)
					head = word[:n]
				}
				out = append(out, head)
				word = word[len(head):]
			}
			cur, cw = word, textWidth(word)
		}
		out = append(out, strings.TrimRight(cur, " "))
	}
	return out
}

// TestWrapUnchanged is plan 5, item 11, criterion 2's ground: wrap draws
// the rows it drew before, for texts of words, runs of spaces, tabs, line
// breaks, control and wide characters, at widths 1 to 30.
func TestWrapUnchanged(t *testing.T) {
	if got := wrap("a界b", 1); !slices.Equal(got, []string{"a", "界", "b"}) {
		t.Fatalf("wrap at 1 cell = %q", got) // a wide rune once hung wrap
	}
	parts := []string{"a", "bb", "word", "averyveryverylongword", " ", "  ", "   ", "\t", "\n", "\r\n", "\r", "\x01", "\x1f", "\x7f", "界", "é", "**x**", "`c`"}
	r := rand.New(rand.NewPCG(11, 2026))
	for n := range 3000 {
		var b strings.Builder
		for range r.IntN(24) {
			b.WriteString(parts[r.IntN(len(parts))])
		}
		text, w := b.String(), 1+n%30
		if got, want := wrap(text, w), wrapBefore(text, w); !slices.Equal(got, want) {
			t.Fatalf("wrap(%q, %d) = %q, want %q", text, w, got, want)
		}
	}
}

// inspectAnswers are an agent's two AI messages: every construct of
// ui-design.md's Markdown section, then a message ending inside an unclosed
// fence.
var inspectAnswers = []string{strings.Join([]string{
	"# Plan for the fix",
	"Some **bold** words, *emphasis*, `inline code` and a [link](https://example.com).",
	"> a quoted line",
	"- first item",
	"2. second item",
	"| Name | Value |",
	"|------|-------|",
	"| alpha | 42 |",
	"---",
	"```go",
	"func main() {",
	"\t// a comment here",
	"\ts := \"a string\"",
	"\tn := 1234",
	"\t/* a block comment",
	"\t   over lines */ m := 7",
	"}",
	"```",
	"A long line with `a long inline code span that is going to be wrapped across rows at narrow widths for sure` end.",
}, "\n"), "```python\ndef f(x):\n    return x  # note"}

// inspectMDState is an inspected agent with the two AI messages around a
// long notice, an end, an error and a history line.
func inspectMDState() *state.State {
	at := time.Date(2026, 10, 17, 9, 30, 5, 0, time.Local)
	s := state.New()
	s.Screen = state.ScreenDashboard
	s.Session.Phase = state.Ready
	s.Agents["a1"] = &state.Agent{Key: "a1", Label: "general-purpose", Description: "fix the **plan**", Status: state.AgentCompleted,
		StartedAt: at, EndedAt: at.Add(time.Minute), Summary: "**done**: all `green`", Error: "a late error", History: []string{"a history line"},
		Messages: []state.Entry{
			{Role: "assistant", Text: inspectAnswers[0], At: at},
			{Role: "notice", Text: "a notice with **stars** and `ticks` that is long enough to wrap onto a second row at eighty columns", At: at},
			{Role: "assistant", Text: inspectAnswers[1], At: at},
		}}
	return s
}

// TestInspectMarkdown is plan 5, item 11, as plan 71 has it (design D5,
// criterion 71.3): the Messages tab draws an agent's AI messages as zone B
// does, markers hidden, in zone B's styles; the start and end lines show the
// description's and the summary's text; a notice and every other row are
// drawn as before; scrolled from top to bottom the rows follow each other,
// and a selection copies what they show.
func TestInspectMarkdown(t *testing.T) {
	defer func() { markdownOn = true }()
	s := inspectMDState()
	draw := func(ui UI, md bool) *Screen {
		markdownOn = md
		defer func() { markdownOn = true }()
		return Draw(s, ui)
	}
	// Each construct in zone B's style; 80 columns wrap the long code span;
	// 200 rows show the whole tab.
	ui := UI{Width: 80, Height: 200, Inspect: "a1"}
	md, plain := draw(ui, true), draw(ui, false)
	for _, c := range []struct {
		text string
		st   *lipgloss.Style
	}{
		{"Plan for the fix", stMdHeading},
		{"bold", mdStyle(stText, true, false, false)},
		{"emphasis", mdStyle(stText, false, true, false)},
		{"inline code", stMdCode},
		{"link", stMdLink},
		{"a quoted line", stMdEm},
		{"•", stMdBullet},
		{"2.", stMdBullet},
		{"│ ", stMdMark}, // a table's border
		{"├───────┼───────┤", stMdMark},
		{"go", stMdLang},
		{"func", stMdKeyword},
		{"// a comment here", stMdComment},
		{`"a string"`, stMdString},
		{"1234", stMdNumber},
		{"over lines */", stMdComment}, // a block comment's second line (item 12)
		{"7", stMdNumber},              // after its end
		{"sure", stMdCode},             // a code span's row after the wrap
		{"def", stMdKeyword},           // in the unclosed fence
		{"# note", stMdComment},
	} {
		if got := panelStyle(md, 'I', c.text); got != c.st {
			t.Errorf("%q styled %v, want %v", c.text, got, c.st)
		}
	}
	text := texts(md.lines)
	for _, gone := range []string{"# Plan", "**bold**", "*emphasis*", "`inline code`", "[link]", "> a quoted", "- first item", "|------|", "```", "**plan**", "**done**", "`green`"} {
		if strings.Contains(text, gone) {
			t.Errorf("%q drawn as written:\n%s", gone, text)
		}
	}
	for _, want := range []string{"started by orchestrator · fix the plan", "• first item", "│ Name  │ Value │", "│ alpha │ 42    │", "(https://example.com).",
		"a notice with **stars** and `ticks`", "second row at eighty columns", "completed · done: all green", "a late error", "a history line"} {
		if !strings.Contains(text, want) {
			t.Errorf("no %q:\n%s", want, text)
		}
	}
	// The rows that are not an AI message's are drawn as with Markdown off,
	// the notice's among them.
	same := 0
	for _, want := range []string{"a notice with **stars**", "second row at eighty columns", "a late error", "a history line", "Messages"} {
		y, z := rowWith(strings.Split(text, "\n"), want), rowWith(strings.Split(texts(plain.lines), "\n"), want)
		if y < 0 || z < 0 {
			t.Errorf("no row %q", want)
			continue
		}
		if a, b := fmt.Sprint(spansOf(md.lines[y])), fmt.Sprint(spansOf(plain.lines[z])); a != b {
			t.Errorf("row %q differs:\n%s\nplain\n%s", want, a, b)
		}
		same++
	}
	if same < 5 {
		t.Errorf("only %d rows compared", same)
	}

	// Scrolled from top to bottom at three sizes: each top's rows are the
	// last top's moved by one, and a selection over them copies what they
	// show.
	for _, size := range [][2]int{{80, 24}, {120, 40}, {200, 60}} {
		ui := UI{Width: size[0], Height: size[1], Inspect: "a1", InspectHold: true}
		var last []string
		for top := 0; ; top++ {
			ui.InspectTop = top
			sc := draw(ui, true)
			v := sc.Inspect
			var rows []string
			for i := v.Top; i < v.Top+v.Room; i++ {
				rows = append(rows, sc.source('I', i))
			}
			if last != nil && !slices.Equal(rows[:len(rows)-1], last[1:]) {
				t.Fatalf("%v top %d: rows\n%s\nafter\n%s", size, top, strings.Join(rows, "\n"), strings.Join(last, "\n"))
			}
			sel := Selection{Pos{'I', v.Top, 0, 0}, Pos{'I', v.Top + v.Room - 1, 0, 1 << 20}}
			if got := sc.Text(s, sel); got != strings.Join(rows, "\n") || strings.Contains(got, "**bold**") {
				t.Fatalf("%v top %d: copied\n%q\nwant\n%q", size, top, got, strings.Join(rows, "\n"))
			}
			last = rows
			if top >= v.Rows-v.Room {
				break
			}
		}
	}
}

// panelStyle is the style of panel p's first span whose text is text, or
// else that contains it.
func panelStyle(sc *Screen, p byte, text string) *lipgloss.Style {
	var near *lipgloss.Style
	for _, l := range sc.lines {
		for _, x := range l {
			if x.ref.panel != p {
				continue
			}
			if x.text == text {
				return x.st
			}
			if near == nil && strings.Contains(x.text, text) {
				near = x.st
			}
		}
	}
	return near
}

// spansOf is a row's spans: text, style and kind, to compare two rows.
func spansOf(l line) []string {
	var out []string
	for _, x := range l {
		st := "-"
		if x.st != nil {
			st = x.st.Render("x")
		}
		out = append(out, fmt.Sprintf("%q %s %d", x.text, st, x.ref.kind))
	}
	return out
}
