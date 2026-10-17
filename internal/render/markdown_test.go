package render

import (
	"fmt"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"

	"github.com/Evengelius/agent-command-center/internal/state"
)

// mdFixture has every construct ui-design.md's Markdown section styles; the
// streaming answer ends inside an unclosed fence.
var mdFixture = []state.Entry{
	{Role: "user", Text: "hi"},
	{Role: "assistant", Text: strings.Join([]string{
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
		"}",
		"```",
		"A long line with `a long inline code span that is going to be wrapped across rows at narrow widths for sure` end.",
	}, "\n")},
	{Role: "user", Text: "more"},
	{Role: "assistant", Streaming: true, Text: "```python\ndef f(x):\n    return x  # note"},
}

// Plan 4, step 8, criterion 8: AI text styled as Markdown draws the same
// characters as the plain text, each at the same place for selection, at
// three widths; each construct takes its style.
func TestMarkdownSameText(t *testing.T) {
	defer func() { markdownOn = true }()
	s := state.New()
	s.Screen = state.ScreenDashboard
	s.Session.Phase = state.Ready
	s.Conversation = mdFixture
	// at 200 columns zone B shares its height with C: 80 rows show it all
	for _, size := range [][2]int{{80, 60}, {120, 60}, {200, 80}} {
		t.Run(fmt.Sprintf("%dx%d", size[0], size[1]), func(t *testing.T) {
			ui := UI{Width: size[0], Height: size[1], Focus: "B"}
			markdownOn = false
			plain := Draw(s, ui)
			markdownOn = true
			md := Draw(s, ui)
			if a, b := texts(md.lines), texts(plain.lines); a != b {
				t.Fatalf("styled text differs from plain:\n%s\n--- plain ---\n%s", a, b)
			}
			if md.String() == plain.String() {
				t.Error("the styled frame draws as the plain one")
			}
			for y := range size[1] {
				for x := range size[0] {
					p1, q1, ok1 := md.HitIn('B', x, y, false)
					p2, q2, ok2 := plain.HitIn('B', x, y, false)
					if p1 != p2 || q1 != q2 || ok1 != ok2 {
						t.Fatalf("cell %d,%d hits %v %v, plain %v %v", x, y, p1, q1, p2, q2)
					}
				}
			}
			for _, c := range []struct {
				text string
				st   *lipgloss.Style
			}{
				{"#", stMdMark},
				{"Plan for the fix", stMdHeading},
				{"bold", stMdStrong},
				{"emphasis", stMdEm},
				{"inline code", stMdCode},
				{"link", stMdLink},
				{"a quoted line", stMdEm},
				{"- ", stMdBullet},
				{"2. ", stMdBullet},
				{"|", stMdMark},
				{"|------|-------|", stMdMark},
				{"go", stMdLang},
				{"func", stMdKeyword},
				{"// a comment here", stMdComment},
				{`"a string"`, stMdString},
				{"1234", stMdNumber},
				{"sure", stMdCode},   // a code span's row after the wrap
				{"def", stMdKeyword}, // in the unclosed fence
				{"# note", stMdComment},
			} {
				if got := spanStyle(md, c.text); got != c.st {
					t.Errorf("%q styled %v, want %v", c.text, got, c.st)
				}
				if got := spanStyle(plain, c.text); got != stText {
					t.Errorf("plain %q styled %v, want stText", c.text, got)
				}
			}
		})
	}
}

// spanStyle is the style of zone B's first text span that is text, or
// contains it when no span is.
func spanStyle(sc *Screen, text string) *lipgloss.Style {
	var near *lipgloss.Style
	for _, l := range sc.lines {
		for _, sp := range l {
			if sp.ref.panel != 'B' || sp.ref.kind != refText {
				continue
			}
			if sp.text == text {
				return sp.st
			}
			if near == nil && strings.Contains(sp.text, text) {
				near = sp.st
			}
		}
	}
	return near
}

// The block kinds: fences carry over lines, close only on a fence as long,
// and an unclosed one runs to the end.
func TestMarkdownLines(t *testing.T) {
	for _, c := range []struct {
		lines []string
		want  string
	}{
		{[]string{"# h", "#nope", "> q", "- l", "1) l", "| t |", "***", "x"}, "HPQLLTRP"},
		{[]string{"````go", "a", "```", "b", "`````", "c"}, "FCCCFP"},
		{[]string{"~~~~", "```", "~~~", "~~~~", "d"}, "FCCFP"},
		{[]string{"text", "```", "open", "    # still code"}, "PFCC"},
		{[]string{"    # indented", "``` not ` a fence"}, "PP"},
	} {
		var got strings.Builder
		for _, k := range mdLines(c.lines) {
			got.WriteByte("PHQLTRFC"[k.kind])
		}
		if got.String() != c.want {
			t.Errorf("%q: %s, want %s", c.lines, got.String(), c.want)
		}
	}
}
