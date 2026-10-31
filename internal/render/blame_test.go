package render

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"

	"github.com/Evengelius/agent-command-center/internal/state"
)

// blameSample is a four-line file whose blame names Ana Hubaut's commit,
// Bo Li's twice, and a line not committed; the cursor on the first.
func blameSample() FileView {
	return FileView{Open: true, Path: "f.txt", Rel: "f.txt", Opened: "Explorer", Match: -1,
		Lines: []string{"alpha", "beta", "gamma", "delta"},
		Blame: FileBlame{On: true, Lines: []int{0, 1, 2, 1}, Commits: []BlameCommit{
			{Short: "f52c343", Author: "Ana Hubaut", Initials: "AH", Summary: "Add f.txt with three lines",
				Date: "15/11/2022", DateShort: "15/11/22", When: "15/11/2022 10:00"},
			{Short: "9a1b2c3", Author: "Bo Li", Initials: "BL", Summary: "Rework f.txt",
				Date: "02/03/2023", DateShort: "02/03/23", When: "02/03/2023 08:30"},
			{Uncommitted: true},
		}}}
}

// blameFrame draws the dashboard at w x h with the viewer f over it, and
// gives the screen and its rows' text.
func blameFrame(f FileView, w, h int) (*Screen, []string) {
	s := state.New()
	s.Screen = state.ScreenDashboard
	sc := Draw(s, UI{Width: w, Height: h, Focus: "B", File: f})
	return sc, strings.Split(texts(sc.lines), "\n")
}

// rowWith is the first row holding text, -1 when none.
func rowWith(rows []string, text string) int {
	for i, r := range rows {
		if strings.Contains(r, text) {
			return i
		}
	}
	return -1
}

// blameStyle is the style of the first span holding text on row y.
func blameStyle(sc *Screen, y int, text string) *lipgloss.Style {
	for _, s := range sc.lines[y] {
		if strings.Contains(s.text, text) {
			return s.st
		}
	}
	return nil
}

// TestBlameColumn is plan 60, criteria 60.2 and 60.3 (designs D2 and D3):
// left of the line numbers, each line's date and author, aligned, "Not
// committed" in the attention colour, the cursor row on its ground; the full
// form where the code keeps 60 cells, the compact one below, full screen by
// the same rule; a long name cut to 12 cells.
func TestBlameColumn(t *testing.T) {
	f := blameSample()
	sc, rows := blameFrame(f, 120, 40)
	for _, want := range []string{
		"15/11/2022 Ana Hubaut    1  alpha",
		"02/03/2023 Bo Li         2  beta",
		"Not committed            3  gamma",
		"02/03/2023 Bo Li         4  delta",
	} {
		if rowWith(rows, want) < 0 {
			t.Errorf("120x40: no row %q:\n%s", want, strings.Join(rows, "\n"))
		}
	}
	if y := rowWith(rows, "3  gamma"); y < 0 || blameStyle(sc, y, "Not committed") != stAttention {
		t.Errorf("the line not committed is not in the attention colour")
	}
	if y := rowWith(rows, "2  beta"); y < 0 || blameStyle(sc, y, "02/03/2023 Bo Li") != stGrey {
		t.Errorf("a committed line's date and author are not grey")
	}
	if y := rowWith(rows, "1  alpha"); y < 0 || blameStyle(sc, y, "15/11/2022 Ana Hubaut") != stFileRowNum {
		t.Errorf("the cursor row's column is not on the row's ground")
	}
	f.Cursor = 2
	if sc, rows := blameFrame(f, 120, 40); blameStyle(sc, rowWith(rows, "3  gamma"), "Not committed") != stFileRowMark {
		t.Errorf("the cursor row's line not committed is not on the row's ground")
	}

	f = blameSample()
	sc, rows = blameFrame(f, 80, 24)
	for _, want := range []string{"15/11/22 AH   1  alpha", "02/03/23 BL   2  beta", "uncommitted   3  gamma"} {
		if rowWith(rows, want) < 0 {
			t.Errorf("80x24: no row %q:\n%s", want, strings.Join(rows, "\n"))
		}
	}
	if y := rowWith(rows, "3  gamma"); y < 0 || blameStyle(sc, y, "uncommitted") != stAttention {
		t.Errorf("80x24: the line not committed is not in the attention colour")
	}
	// The full form from exactly 60 cells of code, windowed and full screen;
	// one cell fewer, the compact one.
	for _, full := range []bool{false, true} {
		found := 0
		for w := 80; w <= 130; w++ {
			_, _, ww, _ := FileRect(w, 40, full)
			code := ww - 4 - 1 - 2 - blameFull
			if code != blameCode && code != blameCode-1 {
				continue
			}
			f := blameSample()
			f.Full = full
			want := "15/11/2022 Ana Hubaut    1  alpha"
			if code < blameCode {
				want = "15/11/22 AH   1  alpha"
			}
			if _, rows := blameFrame(f, w, 40); rowWith(rows, want) < 0 {
				t.Errorf("%dx40, full screen %v, %d cells of code beside the full form: no row %q", w, full, code, want)
			}
			found++
		}
		if found < 2 {
			t.Errorf("full screen %v: %d widths at the boundary", full, found)
		}
	}
	for _, c := range []struct {
		w, h int
		want string
	}{{80, 24, "15/11/22 AH   1  alpha"}, {120, 40, "15/11/2022 Ana Hubaut    1  alpha"}} {
		f := blameSample()
		f.Full = true
		if _, rows := blameFrame(f, c.w, c.h); rowWith(rows, c.want) < 0 {
			t.Errorf("full screen at %dx%d: no row %q", c.w, c.h, c.want)
		}
	}

	f = blameSample()
	f.Blame.Commits[0].Author = "Christophe-Alexandre Hubaut"
	if _, rows := blameFrame(f, 120, 40); rowWith(rows, "15/11/2022 Christophe-…  1  alpha") < 0 {
		t.Errorf("a long name not cut to 12 cells:\n%s", strings.Join(rows, "\n"))
	}
	// While git reads, the column is blank; off, it is not drawn at all.
	f = blameSample()
	f.Blame.Lines, f.Blame.Note = nil, "blame…"
	if _, rows := blameFrame(f, 120, 40); rowWith(rows, strings.Repeat(" ", 25)+"1  alpha") < 0 {
		t.Errorf("reading: the column not blank")
	}
	f.Blame.On = false
	if _, rows := blameFrame(f, 120, 40); rowWith(rows, "│ 1  alpha") < 0 {
		t.Errorf("off: the column drawn")
	}
}

// TestBlameWidth is criterion 60.6's shift limit: the code's cells are the
// row's less the column, the column's only while the blame shows.
func TestBlameWidth(t *testing.T) {
	f := blameSample()
	for _, c := range []struct {
		w, h, on, off int
	}{{120, 40, 96 - 4 - 1 - 2 - blameFull, 96 - 4 - 1 - 2}, {80, 24, 76 - 4 - 1 - 2 - blameCompact, 76 - 4 - 1 - 2}} {
		f.Blame.On = true
		if got := FileCodeWidth(UI{Width: c.w, Height: c.h, File: f}); got != c.on {
			t.Errorf("%dx%d, the blame on: %d cells of code, want %d", c.w, c.h, got, c.on)
		}
		f.Blame.On = false
		if got := FileCodeWidth(UI{Width: c.w, Height: c.h, File: f}); got != c.off {
			t.Errorf("%dx%d, the blame off: %d cells of code, want %d", c.w, c.h, got, c.off)
		}
	}
	// A row is as wide as asked, a long line's code cut to what the column
	// leaves it.
	f = blameSample()
	f.Lines[0] = strings.Repeat("x", 300)
	for _, bw := range []int{blameFull, blameCompact} {
		if got := codeRow(f, 0, 1, bw, 94, nil).width(); got != 94 {
			t.Errorf("a long line beside a %d-cell column: %d cells, want 94", bw, got)
		}
	}
}

// TestBlameFooter is criteria 60.4 and 60.5 as drawn (designs D4 and D6):
// the cursor line's commit in the footer's place of the keys, following the
// cursor; a line not committed; the notes, a failure in the attention
// colour; a toast before them; the search field's hints while it is typed;
// the keys again with the blame off.
func TestBlameFooter(t *testing.T) {
	footer := func(f FileView, ui UI) (string, *lipgloss.Style) {
		s := state.New()
		s.Screen = state.ScreenDashboard
		ui.Width, ui.Height, ui.Focus, ui.File = 120, 40, "B", f
		sc := Draw(s, ui)
		rows := strings.Split(texts(sc.lines), "\n")
		y := rowWith(rows, "line ")
		if y < 0 {
			y = rowWith(rows, "Enter find")
		}
		if y < 0 {
			return "", nil
		}
		var st *lipgloss.Style
		for _, s := range sc.lines[y] {
			if strings.TrimSpace(s.text) != "" && !strings.ContainsAny(s.text, "│╰") {
				st = s.st
				break
			}
		}
		return rows[y], st
	}
	f := blameSample()
	if row, _ := footer(f, UI{}); !strings.Contains(row, "f52c343 · Ana Hubaut · 15/11/2022 10:00 · Add f.txt with three lines") || strings.Contains(row, "b blame") {
		t.Errorf("the first line's commit: %q", row)
	}
	f.Cursor = 2
	if row, st := footer(f, UI{}); !strings.Contains(row, "Not committed yet: changed in the working tree") || st != stAttention {
		t.Errorf("a line not committed: %q", row)
	}
	f.Cursor = 3
	if row, _ := footer(f, UI{}); !strings.Contains(row, "9a1b2c3 · Bo Li · 02/03/2023 08:30 · Rework f.txt") {
		t.Errorf("the fourth line's commit: %q", row)
	}
	// A subject longer than the room: cut with "…", a space before the line.
	f = blameSample()
	f.Blame.Commits[0].Summary = strings.Repeat("a long subject ", 10)
	if row, _ := footer(f, UI{}); !strings.Contains(row, "f52c343 · Ana Hubaut · 15/11/2022 10:00 · a long subject") || !strings.Contains(row, "… line 1/4") {
		t.Errorf("a long subject: %q", row)
	}
	for _, c := range []struct {
		note string
		bad  bool
		st   *lipgloss.Style
	}{{"blame…", false, stGrey}, {"not in git yet: no history", true, stAttention}} {
		f := blameSample()
		f.Blame.Lines, f.Blame.Note, f.Blame.NoteBad = nil, c.note, c.bad
		if row, st := footer(f, UI{}); !strings.Contains(row, c.note) || st != c.st {
			t.Errorf("the note %q: %q", c.note, row)
		}
	}
	if row, _ := footer(blameSample(), UI{Toast: "copied 1 line"}); !strings.Contains(row, "copied 1 line") || strings.Contains(row, "f52c343") {
		t.Errorf("a toast first: %q", row)
	}
	f = blameSample()
	f.Field = '/'
	if row, _ := footer(f, UI{}); !strings.Contains(row, "Enter find · Esc cancel") || strings.Contains(row, "f52c343") {
		t.Errorf("the search field typed: %q", row)
	}
	f = blameSample()
	f.Blame.On = false
	if row, _ := footer(f, UI{}); !strings.Contains(row, "b blame") || strings.Contains(row, "f52c343") {
		t.Errorf("the blame off: %q", row)
	}
}
