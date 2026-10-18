package render

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"charm.land/lipgloss/v2"
)

// classOfText is the class of the first run holding needle, and whether one
// does.
func classOfText(h Highlighted, needle string) (SynClass, bool) {
	for _, l := range h.Lines {
		for _, r := range l {
			if strings.Contains(r.Text, needle) {
				return r.Class, true
			}
		}
	}
	return 0, false
}

// joined checks plan 17's criterion 2.3: each line's runs join to the line.
func joined(t *testing.T, name string, h Highlighted, lines []string) {
	t.Helper()
	if want := len(lines) - h.From; h.From+len(h.Lines) > len(lines) || (h.From == 0 && len(h.Lines) != want) {
		t.Errorf("%s: %d lines from %d, of %d", name, len(h.Lines), h.From, len(lines))
	}
	for i, l := range h.Lines {
		var b strings.Builder
		for _, r := range l {
			b.WriteString(r.Text)
		}
		if b.String() != lines[h.From+i] {
			t.Errorf("%s line %d: runs join to %q, want %q", name, h.From+i+1, b.String(), lines[h.From+i])
		}
	}
}

// samples are plan 17's criterion 2.1: a file per language, its expected
// language name, and a text of each class that must be found coloured so.
var samples = []struct {
	name, lang, text string
	classes          map[string]SynClass
}{
	{"main.go", "Go", "package main\n\nimport \"fmt\"\n\n// main greets.\nfunc main() {\n\tfmt.Println(\"hello\", 42)\n}\n",
		map[string]SynClass{"func": SynKeyword, `"hello"`: SynString, "42": SynNumber, "// main greets.": SynComment}},
	{"app.js", "JavaScript", "// greet\nconst greet = (name) => {\n  return `hi ${name}` + 'x';\n};\n",
		map[string]SynClass{"const": SynKeyword, "'x'": SynString, "// greet": SynComment}},
	{"app.ts", "TypeScript", "interface User { name: string }\nexport function hello(u: User): string {\n  return \"hello \" + u.name;\n}\n",
		map[string]SynClass{"interface": SynKeyword, `"hello "`: SynString}},
	{"Main.java", "Java", "public class Main {\n  public static void main(String[] args) {\n    System.out.println(\"hi\");\n  }\n}\n",
		map[string]SynClass{"public": SynKeyword, `"hi"`: SynString}},
	{"app.py", "Python", "def greet(name):\n    # say it\n    return f\"hi {name}\" if name else 'nobody'\n",
		map[string]SynClass{"def": SynKeyword, "'nobody'": SynString, "# say it": SynComment}},
	{"main.rs", "Rust", "fn main() {\n    let s = \"hi\";\n    println!(\"{}\", s);\n}\n",
		map[string]SynClass{"fn": SynKeyword, `"hi"`: SynString}},
	{"Program.cs", "C#", "using System;\nclass Program {\n  static void Main() { Console.WriteLine(\"hi\"); }\n}\n",
		map[string]SynClass{"using": SynKeyword, `"hi"`: SynString}},
	{"q.sql", "MySQL", "SELECT name FROM users WHERE id = 'abc';\n",
		map[string]SynClass{"SELECT": SynKeyword, "'abc'": SynString}},
	{"c.yaml", "YAML", "name: demo\nenabled: true\ntitle: \"quoted\"\n",
		map[string]SynClass{"true": SynKeyword, `"quoted"`: SynString}},
	{"d.json", "JSON", "{\n  \"name\": \"demo\",\n  \"ok\": true,\n  \"n\": 3\n}\n",
		map[string]SynClass{"true": SynKeyword, `"demo"`: SynString, "3": SynNumber}},
	{"README.md", "markdown", "# Title\n\nSome `code` here.\n",
		map[string]SynClass{"# Title": SynHeading, "`code`": SynString}},
	{"run.sh", "Bash", "#!/bin/sh\nif [ -n \"$1\" ]; then\n  echo 'hi'\nfi\n",
		map[string]SynClass{"if": SynKeyword, "'hi'": SynString}},
}

// TestHighlight covers plan 17's criteria 2.1 to 2.3 and 2.5's whole file.
func TestHighlight(t *testing.T) {
	for _, s := range samples {
		lines := strings.Split(strings.TrimSuffix(s.text, "\n"), "\n")
		h := Highlight(s.name, lines, 0, len(lines))
		if h.Lang != s.lang {
			t.Errorf("%s: language %q, want %q", s.name, h.Lang, s.lang)
		}
		for text, want := range s.classes {
			if got, ok := classOfText(h, text); !ok || got != want {
				t.Errorf("%s: %q is class %d (found %v), want %d", s.name, text, got, ok, want)
			}
		}
		joined(t, s.name, h, lines)
		// The same by path, Windows or slash form.
		if p := Highlight(`C:\src\`+s.name, lines, 0, len(lines)); p.Lang != s.lang {
			t.Errorf("%s by path: %q", s.name, p.Lang)
		}
	}

	// 2.2: an unknown name and prose.
	prose := []string{"Some notes about the week.", "", "Nothing here is code, just words."}
	h := Highlight("notes.zzz", prose, 0, len(prose))
	if h.Lang != "Plain text" {
		t.Errorf("notes.zzz: language %q", h.Lang)
	}
	for _, l := range h.Lines {
		for _, r := range l {
			if r.Class != SynText {
				t.Errorf("notes.zzz: %q is class %d", r.Text, r.Class)
			}
		}
	}
	joined(t, "notes.zzz", h, prose)

	// 2.3: edge cases and a window starting inside a block comment.
	edge := []string{"package p", "", "/* a comment", "\tthat goes on", "   ", "*/", "func f() {", "\t\treturn // é ✓ \uFFFD", "}", ""}
	joined(t, "edge.go", Highlight("edge.go", edge, 0, len(edge)), edge)
	w := Highlight("edge.go", edge, 3, 8)
	if w.From != 3 || len(w.Lines) != 5 || w.Lang != "Go" {
		t.Errorf("window: from %d, %d lines, %q", w.From, len(w.Lines), w.Lang)
	}
	joined(t, "edge.go window", w, edge)
	if c := Highlight("edge.go", edge, 7, 99); c.From != 7 || len(c.Lines) != 3 {
		t.Errorf("clamped: from %d, %d lines", c.From, len(c.Lines))
	}
	if e := Highlight("empty.go", nil, 0, 0); len(e.Lines) != 0 {
		t.Errorf("empty: %d lines", len(e.Lines))
	}

	// 2.5: a 1 MiB Go file is highlighted whole.
	var big []string
	size := 0
	for n := 0; size < 1<<20; n++ {
		big = append(big, fmt.Sprintf("// f%d adds.", n), fmt.Sprintf("func f%d(a, b int) int { return a + b + %d } // \"x\"", n, n), "")
		size += len(big[len(big)-3]) + len(big[len(big)-2]) + 3
	}
	start := time.Now()
	hb := Highlight("big.go", big, 0, len(big))
	t.Logf("%d lines, %d bytes: %v", len(big), len(strings.Join(big, "\n")), time.Since(start))
	if len(hb.Lines) != len(big) || hb.Lang != "Go" {
		t.Errorf("big.go: %d lines of %d, %q", len(hb.Lines), len(big), hb.Lang)
	}
	if c, _ := classOfText(Highlighted{Lines: hb.Lines[len(hb.Lines)-3:]}, "func"); c != SynKeyword {
		t.Errorf("big.go: the last function is not coloured: %d", c)
	}
	joined(t, "big.go", hb, big)
}

// synColours is each class's palette colour, nil for none.
var synColours = map[SynClass]colour{SynText: cText, SynKeyword: cKeyword, SynString: cString, SynNumber: cNumber,
	SynComment: cGrey, SynFunc: cCode, SynBuiltin: cLink, SynTag: cKeyword, SynAttr: cCode, SynInserted: cDone,
	SynDeleted: cFailed, SynHeading: cTitle}

// TestSyntaxPalette covers plan 17's criterion 2.4: each class takes its
// palette colour, true or ANSI.
func TestSyntaxPalette(t *testing.T) {
	defer UsePalette(pickTrue)
	for _, pick := range []lipgloss.CompleteFunc{pickTrue, pickANSI} {
		UsePalette(pick)
		for c := SynText; c <= SynHeading; c++ {
			want := synColours[c].get()
			if got := c.style().GetForeground(); got != want {
				t.Errorf("sixteen %v: class %d is %v, want %v", sixteen, c, got, want)
			}
		}
	}
	if !stMdKeyword.GetBold() || !stMdComment.GetItalic() || stSynBuiltin.GetUnderline() {
		t.Error("keywords bold, comments italic, built-ins not underlined")
	}
}
