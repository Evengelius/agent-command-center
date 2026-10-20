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
		"\t/* a block comment",
		"\t   over lines */ m := 7",
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
	// at 200 columns zone B shares its height with C: 80 rows show it all;
	// at 120, 64 do
	for _, size := range [][2]int{{80, 60}, {120, 64}, {200, 80}} {
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
				{"over lines */", stMdComment}, // a block comment's second line (plan 5, item 12)
				{"7", stMdNumber},              // after its end
				{"sure", stMdCode},             // a code span's row after the wrap
				{"def", stMdKeyword},           // in the unclosed fence
				{"# note", stMdComment},
			} {
				// Each style on the card ground: the fixture's answers are in
				// the CLI's card (plan 20).
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

// TestBlockComment is plan 5, item 12, criteria 1 and 2: a block comment is
// a comment on every line to its */, and only there.
func TestBlockComment(t *testing.T) {
	lines := []string{
		"```c",
		"int a; /* opens here",
		"   still \"in\" the comment // too",
		"   ends */ return 42;",
		`x = "/* in a string" + 1;`,
		"y = 2; // line /* not opened",
		"z = 3; /* one line */ w = 4;",
		"int b; /* left open",
		"```",
		"```c",
		"int c = 5;",
		"```",
		"```python",
		"x = 1 /* not a comment",
		"y = 2",
		"```",
		"```",
		"a /* b",
		"c */ d",
		"```",
	}
	ks := mdLines(lines)
	// style is the style of the first occurrence of text on line j.
	style := func(j int, text string) *lipgloss.Style {
		at := strings.Index(lines[j], text)
		if at < 0 {
			t.Fatalf("no %q on line %d", text, j)
		}
		for _, x := range mdRuns(lines[j], ks[j]) {
			if x.to > at {
				return x.st
			}
		}
		return nil
	}
	for _, c := range []struct {
		line int
		text string
		st   *lipgloss.Style
	}{
		{1, "int", stMdKeyword},
		{1, "/* opens here", stMdComment},
		{2, "still", stMdComment}, // the comment's next lines,
		{2, `"in"`, stMdComment},  // a string in it,
		{2, "// too", stMdComment},
		{3, "ends */", stMdComment}, // its end,
		{3, "return", stMdKeyword},  // and the code after it
		{3, "42", stMdNumber},
		{4, `"/* in a string"`, stMdString},
		{5, "y", stMdCode},
		{6, "z", stMdCode}, // after a line comment holding /*
		{6, "/* one line */", stMdComment},
		{6, "w", stMdCode},
		{6, "4", stMdNumber},
		{7, "/* left open", stMdComment},
		{10, "int", stMdKeyword}, // a new fence starts as code
		{10, "5", stMdNumber},
		{13, "/", stMdCode}, // not a comment in Python
		{13, "not", stMdKeyword},
		{14, "2", stMdNumber},
		{17, "a /* b", stMdCode}, // a fence without a language
		{18, "c */ d", stMdCode},
	} {
		if got := style(c.line, c.text); got != c.st {
			t.Errorf("line %d %q styled %v, want %v", c.line, c.text, got, c.st)
		}
	}
	for j, want := range map[int]bool{1: false, 2: true, 3: true, 4: false, 5: false, 6: false, 7: false, 8: false, 10: false, 14: false, 18: false} {
		if (ks[j].open == "*/") != want {
			t.Errorf("line %d starts in a comment: %v, want %v", j, ks[j].open == "*/", want)
		}
	}
}

// TestMultiLineString is plan 6, item 3, criteria 1 to 3: a backtick string in
// Go, JavaScript and TypeScript, and a triple-quoted one in Python, run over
// the lines to their close; elsewhere a string ends with its line, one left
// open ends with its fence, a /* in a string is string and a backtick in a
// block comment is comment.
func TestMultiLineString(t *testing.T) {
	lines := []string{
		"```go",                        // 0
		"q := `select *",               // 1
		"  from \"t\" // not one",      // 2
		"  /* nor this` + x",           // 3
		"```",                          // 4
		"```tsx",                       // 5
		"const a = `x ${y}",            // 6
		"still \\` inside`; let n = 3", // 7
		"```",                          // 8
		"```python",                    // 9
		`s = """doc`,                   // 10
		"  more # text",                // 11
		`""" ; k = 4`,                  // 12
		`t = '''open`,                  // 13
		"```",                          // 14
		"```python",                    // 15
		"n = 5",                        // 16
		"```",                          // 17
		"```sh",                        // 18
		"echo `date",                   // 19
		"x=1",                          // 20
		"```",                          // 21
		"```c",                         // 22
		"/* a ` b",                     // 23
		"c` */ d = 6;",                 // 24
		`s = "/* in a string`,          // 25
		"y = 7;",                       // 26
		"```",                          // 27
	}
	ks := mdLines(lines)
	style := func(j int, text string) *lipgloss.Style {
		at := strings.Index(lines[j], text)
		if at < 0 {
			t.Fatalf("no %q on line %d", text, j)
		}
		for _, x := range mdRuns(lines[j], ks[j]) {
			if x.to > at {
				return x.st
			}
		}
		return nil
	}
	for _, c := range []struct {
		line int
		text string
		st   *lipgloss.Style
	}{
		{1, "select", stMdString}, {2, "from", stMdString}, {2, "// not one", stMdString}, {3, "/* nor this`", stMdString},
		{3, "x", stMdCode}, {4, "```", stMdMark},
		{6, "x ${y}", stMdString}, {7, "still", stMdString}, {7, "inside", stMdString}, {7, "let", stMdKeyword}, {7, "3", stMdNumber},
		{10, "doc", stMdString}, {11, "more # text", stMdString}, {12, `"""`, stMdString}, {12, "k", stMdCode}, {12, "4", stMdNumber},
		{13, "open", stMdString},
		{16, "5", stMdNumber},                         // the next fence starts as code
		{19, "date", stMdString}, {20, "x", stMdCode}, // a shell backtick ends with its line
		{23, "a", stMdComment}, {24, "c", stMdComment}, {24, "d", stMdCode}, {24, "6", stMdNumber},
		{25, "/* in", stMdString}, {26, "y", stMdCode},
	} {
		if got := style(c.line, c.text); got != c.st {
			t.Errorf("line %d %q styled %v, want %v", c.line, c.text, got, c.st)
		}
	}
}
