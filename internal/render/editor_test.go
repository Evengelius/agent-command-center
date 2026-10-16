package render

import (
	"math/rand/v2"
	"reflect"
	"strings"
	"testing"
)

// The editor's drawing before plan 4, step 5's keystroke fix: every line of
// the window wrapped into rows, one span per cell. The tests below hold the
// new drawing to it, cell by cell.

func refRowsOf(n int, text string, w int) []editorRow {
	cs := cells(text)
	var out []editorRow
	cur, cw := []cell{}, 0
	for _, c := range cs {
		if cw+c.w > w && len(cur) > 0 {
			out = append(out, editorRow{line: n, cells: cur})
			cur, cw = []cell{}, 0
		}
		cur, cw = append(cur, c), cw+c.w
	}
	return append(out, editorRow{line: n, cells: cur, end: true})
}

func refShownRows(v EditorView, text, minRows, maxRows int) []editorRow {
	var rows []editorRow
	cursor := 0
	for i, l := range v.Lines {
		n := v.First + i
		wrapped := refRowsOf(n, l, text)
		if n == v.Line {
			cursor = len(rows)
			for k, r := range wrapped {
				if k > 0 && r.cells[0].off <= v.Col {
					cursor = len(rows) + k
				}
			}
		}
		rows = append(rows, wrapped...)
	}
	n := len(rows)
	if v.Total > len(v.Lines) {
		n = maxRows
	}
	show := min(max(n, minRows), maxRows)
	start := min(max(len(rows)-show, 0), cursor)
	return rows[start:min(start+show, len(rows))]
}

func refDrawRow(v EditorView, r editorRow, focused bool) line {
	l := line{{ref: ref{panel: 'E', kind: refText, line: r.line}}}
	if len(r.cells) > 0 {
		l = nil
	}
	cursorShown := false
	for _, c := range r.cells {
		st := stText
		switch {
		case focused && r.line == v.Line && c.off == v.Col:
			st, cursorShown = stCursor, true
		case v.selected(r.line, c.off):
			st = stSelected
		case c.ctrl:
			st = stControl
		}
		at := ref{panel: 'E', kind: refText, line: r.line, off: c.off}
		if c.ctrl || c.text == "    " && c.n == 1 {
			at.kind, at.n = refAtom, c.n
		}
		l = append(l, span{text: c.text, st: st, ref: at})
	}
	if focused && r.end && r.line == v.Line && !cursorShown {
		l = append(l, deco(" ", stCursor))
	}
	return l
}

// cellOut is one cell as the terminal gets it: its text, styled, and what it
// shows for selection and copy.
type cellOut struct {
	text, styled string
	ref          ref
}

// perCell splits a line's text spans into their cells; the other spans stay
// as they are.
func perCell(l line) []cellOut {
	var out []cellOut
	for _, x := range l {
		if x.ref.kind != refText || x.text == "" {
			out = append(out, cellOut{x.text, line{x}.String(), x.ref})
			continue
		}
		for _, c := range cells(x.text) {
			at := x.ref
			at.off += c.off
			out = append(out, cellOut{c.text, line{{text: c.text, st: x.st}}.String(), at})
		}
	}
	return out
}

// randomLine mixes what a draft holds: ASCII, spaces, tabs, C0 and C1
// controls, DEL, wide characters, an emoji, a combining mark, invalid UTF-8
// and the replacement character itself.
func randomLine(r *rand.Rand) string {
	pieces := []string{"a", "b", "z", " ", "\t", "\x01", "\x1b", "\x7f", "\u0085", "日", "本", "👍", "é", "\xff", "�", "·", "word "}
	var b strings.Builder
	for n := r.IntN(60); n > 0; n-- {
		b.WriteString(pieces[r.IntN(len(pieces))])
	}
	return b.String()
}

// TestEditorRowsMatchReference is plan 4, step 5's keystroke fix, check
// 2: the editor wraps every line of its window but builds only the rows it
// shows, and draws a run of plain cells as one span; what it shows, and how
// each cell is styled and copied, is unchanged.
func TestEditorRowsMatchReference(t *testing.T) {
	sigs := map[string]bool{}
	for _, st := range []string{stText.Render("x"), stCursor.Render("x"), stSelected.Render("x"), stControl.Render("x")} {
		sigs[st] = true
	}
	if len(sigs) != 4 {
		t.Fatal("the editor's styles render alike: the comparison could not tell them apart")
	}
	r := rand.New(rand.NewPCG(5, 16))
	for c := 0; c < 3000; c++ {
		var v EditorView
		for n := r.IntN(12); n > 0; n-- {
			v.Lines = append(v.Lines, randomLine(r))
		}
		v.First = r.IntN(5)
		v.Total = v.First + len(v.Lines) + r.IntN(2)*r.IntN(5)
		if len(v.Lines) > 0 {
			i := r.IntN(len(v.Lines))
			v.Line = v.First + i
			v.Col = r.IntN(len(v.Lines[i]) + 1)
		}
		if r.IntN(2) == 0 {
			a := [2]int{v.First + r.IntN(len(v.Lines)+1), r.IntN(30)}
			b := [2]int{a[0] + r.IntN(3), r.IntN(30)}
			if b[0] == a[0] && b[1] < a[1] {
				a[1], b[1] = b[1], a[1]
			}
			v.From, v.To = a, b
		}
		text := 1 + r.IntN(40)
		minRows, maxRows := 1+r.IntN(3), 1+r.IntN(20)
		if maxRows < minRows {
			maxRows = minRows
		}
		focused := r.IntN(4) != 0

		want := refShownRows(v, text, minRows, maxRows)
		got, show := shownRows(v, text, minRows, maxRows)
		if len(got) > show {
			got = got[:show]
		}
		if len(got) != len(want) {
			t.Fatalf("case %d: %d rows shown, want %d\n%+v", c, len(got), len(want), v)
		}
		for i := range want {
			g, w := got[i], want[i]
			if g.line != w.line || g.end != w.end || len(g.cells) != len(w.cells) || len(g.cells) > 0 && !reflect.DeepEqual(g.cells, w.cells) {
				t.Fatalf("case %d, row %d: got line %d end %v %+v, want line %d end %v %+v", c, i, g.line, g.end, g.cells, w.line, w.end, w.cells)
			}
			gd, wd := perCell(drawRow(v, g, focused)), perCell(refDrawRow(v, w, focused))
			if !reflect.DeepEqual(gd, wd) {
				t.Fatalf("case %d, row %d drawn differently:\n got %+v\nwant %+v", c, i, gd, wd)
			}
		}
	}
}
