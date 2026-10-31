package render

import (
	"fmt"
	"strings"
	"testing"
)

// Plan 71, design D5, criterion 71.4: a Markdown file in the file viewer is
// its preview, one row per source line, until m shows its source.

// mdFile is a Markdown file with every construct.
var mdFile = []string{
	"# Agent Command Center",
	"",
	"A **mission control** for `claude` and *codex*, see [the brief](doc/brief.md).",
	"",
	"- first",
	"  - nested",
	"1. one",
	"",
	"> a quote",
	"",
	"| Key | Action |",
	"|-----|--------|",
	"| m | source |",
	"",
	"```go",
	"func main() {} // **not bold**",
	"```",
	"---",
}

// mdViewer is the viewer on mdFile at path, its source when src.
func mdViewer(path string, src bool) FileView {
	return FileView{Open: true, Path: path, Rel: path, Opened: "Explorer", Match: -1, Lines: mdFile, Source: src}
}

func TestMarkdownPreview(t *testing.T) {
	bold := mdStyle(stText, true, false, false)
	for _, size := range [][2]int{{80, 24}, {120, 40}, {200, 60}} {
		w, h := size[0], size[1]
		sc, rows := blameFrame(mdViewer("README.md", false), w, h)
		frame := strings.Join(rows, "\n")
		// Each source line one row, numbered as before; the window's rows
		// as wide as before.
		for i := range min(len(mdFile), FileRows(UI{Width: w, Height: h, File: mdViewer("README.md", false)})) {
			if rowWith(rows, fmt.Sprintf("%2d  ", i+1)) < 0 {
				t.Errorf("%dx%d: no row numbered %d:\n%s", w, h, i+1, frame)
			}
		}
		for y, l := range sc.lines {
			if l.width() > w {
				t.Errorf("%dx%d: row %d %d cells wide", w, h, y, l.width())
			}
		}
		for _, want := range []string{"preview · ", "Agent Command Center", "A mission control for claude and codex, see"} {
			if rowWith(rows, want) < 0 {
				t.Errorf("%dx%d: no %q:\n%s", w, h, want, frame)
			}
		}
		if h >= 40 {
			for _, want := range []string{"• first", "◦ nested", "1. one", "│ a quote", "│ Key │ Action │", "├─────┼────────┤", "│ m   │ source │",
				"func main() {} // **not bold**", "────────", "m source"} {
				if rowWith(rows, want) < 0 {
					t.Errorf("%dx%d: no %q:\n%s", w, h, want, frame)
				}
			}
		}
		for _, gone := range []string{"# Agent", "**mission", "`claude`", "*codex*", "](doc", "- first", "> a quote", "|-----|", "```"} {
			if rowWith(rows, gone) >= 0 {
				t.Errorf("%dx%d: %q drawn as written:\n%s", w, h, gone, frame)
			}
		}
		if y := rowWith(rows, "A mission control"); y < 0 || blameStyle(sc, y, "mission control") != onCursorRow(bold) && blameStyle(sc, y, "mission control") != bold {
			t.Errorf("%dx%d: the strong text not bold", w, h)
		}
		if y := rowWith(rows, "Agent Command Center"); y < 0 || blameStyle(sc, y, "Agent Command Center") != onCursorRow(stMdHeading) {
			t.Errorf("%dx%d: the cursor's heading not on the cursor's ground: %v", w, h, blameStyle(sc, y, "Agent Command Center"))
		}
	}

	// m's other view: the source, as before plan 71, its title and footer
	// saying so.
	sc, rows := blameFrame(mdViewer("README.md", true), 120, 40)
	for _, want := range []string{"source · ", "# Agent Command Center", "A **mission control** for `claude`", "- first", "|-----|--------|", "```go", "m preview"} {
		if rowWith(rows, want) < 0 {
			t.Errorf("the source: no %q:\n%s", want, strings.Join(rows, "\n"))
		}
	}
	if rowWith(rows, "• first") >= 0 || sc.fileMd != nil {
		t.Error("the source drawn as a preview")
	}
	// Any other file: as before, its title and footer without m.
	_, rows = blameFrame(mdViewer("notes.txt", false), 120, 40)
	if rowWith(rows, "# Agent Command Center") < 0 || rowWith(rows, "preview") >= 0 || rowWith(rows, "source ·") >= 0 || rowWith(rows, "m source") >= 0 {
		t.Errorf("a text file:\n%s", strings.Join(rows, "\n"))
	}
	// A file that starts with a byte order mark, as Windows' tools write
	// them: its first line's heading rendered, the mark's bytes counted in
	// no column.
	f := mdViewer("README.md", false)
	f.Lines = []string{"\uFEFF# Title of the file", "text **after** it"}
	if _, rows := blameFrame(f, 120, 40); rowWith(rows, "1  Title of the file") < 0 || rowWith(rows, "#") >= 0 || rowWith(rows, "2  text after it") < 0 {
		t.Errorf("a file with a byte order mark:\n%s", strings.Join(rows, "\n"))
	}
	if c := FileColumnAt(UI{Width: 120, Height: 40, File: f}, 0, len("\uFEFF# ")); c != 0 {
		t.Errorf("its heading's text at column %d, want 0", c)
	}

	// The cursor on the strong line: its cells on the cursor's ground.
	f = mdViewer("README.md", false)
	f.Cursor = 2
	sc, rows = blameFrame(f, 120, 40)
	if y := rowWith(rows, "A mission control"); y < 0 || blameStyle(sc, y, "mission control") != onCursorRow(bold) {
		t.Errorf("the cursor line's strong text: %v", blameStyle(sc, y, "mission control"))
	}

	// Blame and the edit marks beside the preview, as beside the source: the
	// date and author left of the number, a marked line's number in the
	// attention colour and its "▌".
	f = mdViewer("README.md", false)
	f.Mark = [2]int{2, 3}
	f.Blame = FileBlame{On: true, Lines: make([]int, len(mdFile)), Commits: []BlameCommit{{Short: "f52c343", Author: "Ana Hubaut", Initials: "AH",
		Summary: "Add the readme", Date: "15/11/2022", DateShort: "15/11/22", When: "15/11/2022 10:00"}}}
	sc, rows = blameFrame(f, 120, 40)
	if y := rowWith(rows, "Agent Command Center"); y < 0 || !strings.Contains(rows[y], "15/11/2022 Ana Hubaut") || strings.Contains(rows[y], "# Agent") {
		t.Errorf("the blame beside the preview:\n%s", strings.Join(rows, "\n"))
	}
	if y := rowWith(rows, "A mission control"); y < 0 || !strings.Contains(rows[y], " 3▌ A mission control") || blameStyle(sc, y, " 3") != stAttention {
		t.Errorf("the marked line beside the preview: %q", rows[max(y, 0)])
	}

	// The search: a match lit where the preview draws it, its column the
	// cells shown before it.
	f = mdViewer("README.md", false)
	off := strings.Index(mdFile[2], "control")
	f.Query, f.Matches, f.Match = "control", []FileMatch{{Line: 2, Off: off, N: len("control")}}, 0
	sc, rows = blameFrame(f, 120, 40)
	if y := rowWith(rows, "A mission control"); y < 0 || blameStyle(sc, y, "control") != stMatchCur {
		t.Errorf("the match not lit: %v", blameStyle(sc, y, "control"))
	}
	ui := UI{Width: 120, Height: 40, File: f}
	if c := FileColumnAt(ui, 2, off); c != len("A mission ") {
		t.Errorf("the match's column in the preview: %d, want %d", c, len("A mission "))
	}
	ui.File.Source = true
	if c := FileColumnAt(ui, 2, off); c != off {
		t.Errorf("the match's column in the source: %d, want %d", c, off)
	}

	// The shift: from the cells shown.
	f = mdViewer("README.md", false)
	f.Left = 4
	if _, rows := blameFrame(f, 120, 40); rowWith(rows, "ssion control for claude") < 0 {
		t.Errorf("shifted by 4:\n%s", strings.Join(rows, "\n"))
	}

	// A long line: cut at the window's edge with "…", the row as wide.
	f = mdViewer("README.md", false)
	f.Lines = []string{"**" + strings.Repeat("long ", 80) + "**"}
	sc, rows = blameFrame(f, 120, 40)
	if y := rowWith(rows, "long long"); y < 0 || !strings.Contains(rows[y], "…") || sc.lines[y].width() != 120 {
		t.Errorf("a long line: %q", rows[max(y, 0)])
	}

	// The copy: the source of what is selected, a construct's markers with
	// it when it is selected whole.
	f = mdViewer("README.md", false)
	sc, _ = blameFrame(f, 120, 40)
	at := strings.Index(mdFile[2], "mission")
	sel := Selection{From: Pos{Panel: 'F', Item: 2, Off: at}, To: Pos{Panel: 'F', Item: 2, Off: at + len("mission control")}}
	if got := sc.Text(nil, sel); got != "**mission control**" {
		t.Errorf("the strong text copies as %q", got)
	}
	sel = Selection{From: Pos{Panel: 'F', Item: 0, Off: 2}, To: Pos{Panel: 'F', Item: 2, Off: len(mdFile[2])}}
	if got := sc.Text(nil, sel); got != strings.Join(mdFile[:3], "\n") {
		t.Errorf("the first lines copy as %q", got)
	}
}
