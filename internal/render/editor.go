package render

import (
	"fmt"
	"unicode/utf8"

	"charm.land/lipgloss/v2"

	"github.com/Evengelius/agent-command-center/internal/state"
)

// EditorView is the part of zone B's editor the frame draws: a window of
// logical lines around the cursor, never the whole buffer (ui-design.md,
// Budget: only the visible rows, wrapped lazily). The application fills it.
type EditorView struct {
	Lines   []string // logical lines First, First+1, ... of the buffer
	First   int
	Total   int    // logical lines in the buffer
	Line    int    // the cursor's logical line
	Col     int    // the cursor's byte offset in its line
	From    [2]int // the selection's start (line, byte); equal to To when none
	To      [2]int // the selection's end, exclusive
	Pasting bool   // between a paste's start and end markers
}

var (
	stCursor   = lipgloss.NewStyle().Foreground(lipgloss.Color("#111827")).Background(cTitle)
	stSelected = lipgloss.NewStyle().Foreground(cText).Background(lipgloss.Color("#4C4F6B"))
	stControl  = lipgloss.NewStyle().Foreground(cAttention)
)

// TabWidth is the cells a tab takes on screen; the text keeps the tab
// (ui-design.md, Input editor, Content; Markdown, Code blocks).
const TabWidth = 4

// cell is one drawn unit of a logical line: a rune, a tab's spaces, or a
// control character's ^X, with the byte offset it comes from.
type cell struct {
	text string
	w    int
	off  int
	n    int // the source bytes it shows
	ctrl bool
}

// cells splits a logical line into drawn units. Control characters are drawn
// visibly as ^X; nothing is dropped (ui-design.md, Input editor, Content).
func cells(s string) []cell {
	out := make([]cell, 0, len(s))
	for i, r := range s {
		n := utf8.RuneLen(r)
		switch {
		case r == '\t':
			out = append(out, cell{"    ", TabWidth, i, 1, false})
		case r < 0x20:
			out = append(out, cell{"^" + string(rune(r+'@')), 2, i, 1, true})
		case r == 0x7f:
			out = append(out, cell{"^?", 2, i, 1, true})
		case r >= 0x80 && r < 0xa0: // C1 controls
			out = append(out, cell{fmt.Sprintf("<%02X>", r), 4, i, n, true})
		case r == utf8.RuneError:
			_, n := utf8.DecodeRuneInString(s[i:])
			if n == 1 {
				out = append(out, cell{fmt.Sprintf("<%02X>", s[i]), 4, i, 1, true})
				continue
			}
			out = append(out, cell{string(r), 1, i, n, false})
		default:
			out = append(out, cell{s[i : i+n], runeWidth(r), i, n, false}) // its bytes: no allocation
		}
	}
	return out
}

// editorRow is one screen row of the editor: a piece of a logical line.
type editorRow struct {
	line  int
	cells []cell
	end   bool // the logical line's last row
}

// rowsOf wraps one logical line into rows of at most w cells.
func rowsOf(n int, text string, w int) []editorRow {
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
	out = append(out, editorRow{line: n, cells: cur, end: true})
	return out
}

func (v EditorView) selected(line, off int) bool {
	at := func(p [2]int) bool { return line > p[0] || line == p[0] && off >= p[1] }
	return v.From != v.To && at(v.From) && !at(v.To)
}

// editorLines is the editor at B's bottom edge: a rule, the draft, a status.
// It grows with its content from minRows up to maxRows, then scrolls to keep
// the cursor in view (ui-design.md, Input editor, Size).
func editorLines(s *state.State, ui UI, w, minRows, maxRows int) []line {
	v := ui.Editor
	if maxRows < minRows {
		maxRows = minRows
	}
	text := max(w-2-1, 1) // the prefix, and room for the cursor at a row's end
	var rows []editorRow
	cursor := 0
	for i, l := range v.Lines {
		n := v.First + i
		wrapped := rowsOf(n, l, text)
		if n == v.Line {
			// The cursor's row: the last one starting at or before it.
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
		n = maxRows // more lines than the window holds: the editor is at its largest
	}
	show := min(max(n, minRows), maxRows)
	// Keep the last rows in view, unless the cursor is above them.
	start := min(max(len(rows)-show, 0), cursor)
	focused := ui.Focus == "B" || ui.Focus == ""
	empty := v.Total <= 1 && (len(v.Lines) == 0 || v.Lines[0] == "")

	out := []line{rule(w)}
	for i := 0; i < show; i++ {
		prefix := "  "
		if i == 0 {
			prefix = "> "
		}
		l := line{sp(prefix, stTitle)}
		switch k := start + i; {
		case i == 0 && empty:
			l = append(l, span{ref: ref{panel: 'E', kind: refText}})
			if focused {
				l = append(l, deco(" ", stCursor))
			}
			l = append(l, deco(placeholder(s), stGrey))
		case k < len(rows):
			l = append(l, drawRow(v, rows[k], focused)...)
		}
		out = append(out, l)
	}
	count := fmt.Sprintf("%d line", max(v.Total, 1))
	if v.Total > 1 {
		count += "s"
	}
	status := editorStatus(s)
	if v.Pasting {
		status = "pasting…"
	}
	out = append(out, spread(plain("  "+status, stGrey), plain(count+" · Ctrl+G editor", stGrey), w))
	for i := range out {
		out[i] = out[i].fit(w) // the padding is the editor's: a click on it goes to the editor
	}
	return in('E', out)
}

// drawRow draws one row with the selection and the cursor.
func drawRow(v EditorView, r editorRow, focused bool) line {
	l := line{{ref: ref{panel: 'E', kind: refText, line: r.line}}} // an empty line still takes a click
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
