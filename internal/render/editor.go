package render

import (
	"fmt"
	"strings"
	"unicode/utf8"

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
	// Attach are the images attached to the draft, as their chips show
	// them (plan 31).
	Attach []Attachment
}

// Attachment is an image attached to the draft as its chip shows it: the
// file's name and its size as sent (plan 31, design D4); Pixels when the
// dashboard has its thumbnail to draw, else the preview shows Format's name
// in a box (plan 32, design D4).
type Attachment struct {
	Name   string
	Size   string
	Pixels bool
	Format string
}

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
		out = append(out, cellAt(s, i, r))
	}
	return out
}

// cellAt is the drawn unit of rune r at byte i of s.
func cellAt(s string, i int, r rune) cell {
	n := utf8.RuneLen(r)
	switch {
	case r == '\t':
		return cell{"    ", TabWidth, i, 1, false}
	case r < 0x20:
		return cell{"^" + string(rune(r+'@')), 2, i, 1, true}
	case r == 0x7f:
		return cell{"^?", 2, i, 1, true}
	case r >= 0x80 && r < 0xa0: // C1 controls
		return cell{fmt.Sprintf("<%02X>", r), 4, i, n, true}
	case r == utf8.RuneError:
		if _, n := utf8.DecodeRuneInString(s[i:]); n == 1 {
			return cell{fmt.Sprintf("<%02X>", s[i]), 4, i, 1, true}
		}
		return cell{string(r), 1, i, n, false}
	}
	return cell{s[i : i+n], runeWidth(r), i, n, false} // its bytes: no allocation
}

// cellWidth is the width of cellAt(s, i, r), without building its text.
func cellWidth(s string, i int, r rune) int {
	switch {
	case r == '\t':
		return TabWidth
	case r < 0x20 || r == 0x7f:
		return 2
	case r >= 0x80 && r < 0xa0:
		return 4
	case r == utf8.RuneError:
		if _, n := utf8.DecodeRuneInString(s[i:]); n == 1 {
			return 4
		}
		return 1
	}
	return runeWidth(r)
}

// editorRow is one screen row of the editor: a piece of a logical line.
type editorRow struct {
	line  int
	text  string // the logical line
	cells []cell
	end   bool // the logical line's last row
}

// rowsOf wraps one logical line into rows of at most w cells.
func rowsOf(n int, text string, w int) []editorRow {
	cs := cells(text)
	var out []editorRow
	from, cw := 0, 0
	for i, c := range cs {
		if cw+c.w > w && i > from {
			out = append(out, editorRow{line: n, text: text, cells: cs[from:i:i]})
			from, cw = i, 0
		}
		cw += c.w
	}
	out = append(out, editorRow{line: n, text: text, cells: cs[from:], end: true})
	return out
}

// rowCount is len(rowsOf(n, text, w)), without building the rows: the
// editor wraps every line of its window to place the cursor, but draws only
// the rows it shows.
func rowCount(text string, w int) int {
	rows, cw, k := 1, 0, 0
	for i, r := range text {
		cw2 := cellWidth(text, i, r)
		if cw+cw2 > w && k > 0 {
			rows, cw, k = rows+1, 0, 0
		}
		cw, k = cw+cw2, k+1
	}
	return rows
}

func (v EditorView) selected(line, off int) bool {
	at := func(p [2]int) bool { return line > p[0] || line == p[0] && off >= p[1] }
	return v.From != v.To && at(v.From) && !at(v.To)
}

// editorLines is the editor at B's bottom edge: a rule, the draft, a
// separator, then the row of the owner's image (plan 29, brief amendment
// 29; ui-design.md, Input editor): the "+" and the model pill on the tool
// rows' ground, the mode pill at the right end (plan 39), and in the free
// middle what the status line said when it had something to say. It grows with its content
// from minRows up to maxRows, then scrolls to keep the cursor in view
// (ui-design.md, Input editor, Size).
func editorLines(s *state.State, ui UI, w, minRows, maxRows int) []line {
	v := ui.Editor
	out := []line{rule(w)}
	if len(v.Attach) > 0 {
		// The attachments' row, under the rule, within the editor's half of B
		// (plan 31, design D4); above it, when the terminal draws Sixel
		// images, their thumbnails (plan 32, design D4).
		row, starts := attachLayout(v.Attach, w)
		if cols, rows := ThumbCells(ui.Height); ui.Sixel && rows > 0 {
			out = append(out, thumbStrip(v.Attach, starts, cols, rows, w)...)
			maxRows = max(maxRows-rows, minRows)
		}
		out = append(out, row)
		maxRows = max(maxRows-1, minRows)
	}
	out = append(out, draftRows(s, ui, w, minRows, maxRows)...)
	status := editorStatus(s, ui)
	if v.Pasting {
		status = "pasting…"
	}
	switch {
	case keysProblemShown(s, ui) && status == ui.KeysProblem:
		// The configuration file's problem is long: the row is its alone (plan 21, item 5.1).
		out = append(out, rule(w), plain("  "+status, stFailed))
	default:
		// The middle: the draft's line count once it passes one line, then
		// what the status line said; the idle hint goes, the keys being the
		// footer's and the key page's.
		var middle []string
		if p := s.PendingLabel(); p != "" {
			middle = append(middle, p) // Codex's change for the next turn (plan 34, design D4)
		}
		if v.Total > 1 {
			middle = append(middle, lineCount(v))
		}
		if status != idleHint(ui) {
			middle = append(middle, status)
		}
		out = append(out, rule(w), inputRow(s, strings.Join(middle, " · "), w, ui.Plus, ui.Sixel && !ui.PillNoFont))
	}
	for i := range out {
		out[i] = out[i].fit(w) // the padding is the editor's: a click on it goes to the editor
	}
	return in('E', out)
}

// inputRow is the row under the draft (plan 29, design D1): a margin
// cell, " + " on the ground, three cells, the model pill on the ground,
// the middle text in grey three cells after it, and the mode pill and a
// cell at the right end. The pill shows the model, the effort and Ultracode in force (plan
// 34, design D7) and opens the model selector, its cells panel M; the mode
// pill at the right end shows the mode in force and opens the modes
// selector, panel Q (plan 39, design D7). A middle text that
// does not fit beside the pills, three cells on either side, has the row
// alone, as the status line had. The band is panel '+', so a click on it
// opens the menu; while the menu is open (plan 30, design D2) it lies on
// the active ground with the "+" in the lavender-white.
func inputRow(s *state.State, middle string, w int, open, sixel bool) line {
	pill := modelPill(s)
	left := line{deco(" ", stNone)}
	band := span{text: " + ", st: stText, ref: ref{panel: '+', kind: refDeco}}
	if open {
		band.st = active(stThinkEdge)
		left = append(left, band)
	} else {
		left = append(left, grounded(line{band}, 3)...)
	}
	left = append(left, deco("   ", stNone))
	left = append(left, grounded(pill, pill.width())...)
	right := modePill(s, sixel, false) // the mode pill (plan 39, design D7)
	if middle != "" {
		// Three cells before the mode pill too, as after the model pill: a
		// middle that would touch it takes the pill's title away, and one that
		// would touch its icon alone has the row alone (plan 34's review).
		fits := func() bool { return left.width()+3+textWidth(middle)+3+right.width() <= w }
		if !fits() {
			right = modePill(s, sixel, true)
		}
		if !fits() {
			return plain("  "+middle, stGrey)
		}
		left = append(left, deco("   ", stNone), sp(middle, stGrey))
	}
	return spread(left, right, w)
}

// lineCount is the draft's logical lines, "1 line" or "n lines".
func lineCount(v EditorView) string {
	if v.Total > 1 {
		return fmt.Sprintf("%d lines", v.Total)
	}
	return "1 line"
}

// draftRows are the draft's rows, w cells wide, from minRows up to maxRows:
// "> " before the first, the placeholder while it is empty.
func draftRows(s *state.State, ui UI, w, minRows, maxRows int) []line {
	v := ui.Editor
	if maxRows < minRows {
		maxRows = minRows
	}
	text := max(w-2-1, 1) // the prefix, and room for the cursor at a row's end
	rows, show := shownRows(v, text, minRows, maxRows)
	focused := ui.Focus == "B" || ui.Focus == ""
	empty := v.Total <= 1 && (len(v.Lines) == 0 || v.Lines[0] == "")

	var out []line
	for i := 0; i < show; i++ {
		prefix := "  "
		if i == 0 {
			prefix = "> "
		}
		l := line{sp(prefix, stTitle)}
		switch {
		case i == 0 && empty:
			l = append(l, span{ref: ref{panel: 'E', kind: refText}})
			if focused {
				l = append(l, deco(" ", stCursor))
			}
			l = append(l, deco(placeholder(s), stGrey))
		case i < len(rows):
			l = append(l, drawRow(v, rows[i], focused)...)
		}
		out = append(out, l)
	}
	return out
}

// DraftWindow is the draft in a window over the dashboard, Ctrl+G from the
// input box: the same editor, more of it on screen (amendment 13).
type DraftWindow struct {
	Open bool
	Full bool // the window fills the screen (Ctrl+F)
}

// draftKeys are the draft window's key hints, the longest first.
func draftKeys(ui UI) []string {
	sn := sendNewline(ui)
	return []string{
		sn + " · Ctrl+F full screen · Ctrl+C copy · F1 keys · Esc back",
		sn + " · Ctrl+F full screen · Esc back",
		sn + " · Esc back",
	}
}

// DraftRect is the draft window's place on a w by h screen: the file
// viewer's (FileRect).
func DraftRect(ui UI) (x, y, ww, hh int) { return FileRect(ui.Width, ui.Height, ui.DraftWin.Full) }

// draftWindow is the draft window, ww by hh: the draft's rows, then a footer
// with the keys and, on the right, why Enter would not send, or the cursor's
// line.
func draftWindow(s *state.State, ui UI, ww, hh int) []line {
	v := ui.Editor
	inner := ww - 4
	rows := max(hh-3, 1)
	body := draftRows(s, ui, inner, rows, rows)
	right := decoLine(fmt.Sprintf("line %s/%s", grouped(v.Line+1), grouped(max(v.Total, 1))), stGrey)
	switch st := editorStatus(s, ui); {
	case v.Pasting:
		right = decoLine("pasting…", stProgress)
	case st != idleStatus(s, ui):
		right = decoLine(st, stProgress)
	}
	room := inner - right.width() - 1
	body = append(body, spread(windowLeft(s, ui, draftKeys(ui), room), right, inner))
	for i := range body {
		body[i] = body[i].fit(inner)
	}
	return in('E', frameBox(line{deco("Draft", stTitle)}, decoLine(lineCount(v), stGrey), body, ww, hh, stFileFrame))
}

// shownRows is the editor rows shown, each at most text cells wide: show
// of them, rows holding those of the window from the first shown on. Every
// line of the window is wrapped to place the cursor; only the rows shown are
// built (ui-design.md, Budget).
func shownRows(v EditorView, text, minRows, maxRows int) (rows []editorRow, show int) {
	counts := make([]int, len(v.Lines))
	total, cursor := 0, 0
	for i, l := range v.Lines {
		n := v.First + i
		if n == v.Line {
			// The cursor's row: the last one starting at or before it.
			wrapped := rowsOf(n, l, text)
			cursor = total
			for k, r := range wrapped {
				if k > 0 && r.cells[0].off <= v.Col {
					cursor = total + k
				}
			}
			counts[i] = len(wrapped)
		} else {
			counts[i] = rowCount(l, text)
		}
		total += counts[i]
	}
	n := total
	if v.Total > len(v.Lines) {
		n = maxRows // more lines than the window holds: the editor is at its largest
	}
	show = min(max(n, minRows), maxRows)
	// Keep the last rows in view, unless the cursor is above them.
	start := min(max(total-show, 0), cursor)
	for i, at := 0, 0; i < len(v.Lines) && at < start+show; i++ {
		if at+counts[i] > start {
			wrapped := rowsOf(v.First+i, v.Lines[i], text)
			rows = append(rows, wrapped[max(start-at, 0):]...)
		}
		at += counts[i]
	}
	return rows, show
}

// drawRow draws one row with the selection and the cursor.
func drawRow(v EditorView, r editorRow, focused bool) line {
	l := line{{ref: ref{panel: 'E', kind: refText, line: r.line}}} // an empty line still takes a click
	if len(r.cells) > 0 {
		l = nil
	}
	cursorShown := false
	// A run of plain cells in one style is one span of the line's bytes, as
	// the conversation's text is: hit-testing splits it again by rune
	// (Screen.units), and the frame has a span per run, not per character.
	const (
		plainText = iota
		cursor
		selected
		control
	)
	run := -1 // the style of the last span when it is a plain run, else -1
	for _, c := range r.cells {
		st, kind := stText, plainText
		switch {
		case focused && r.line == v.Line && c.off == v.Col:
			st, kind, cursorShown = stCursor, cursor, true
		case v.selected(r.line, c.off):
			st, kind = stSelected, selected
		case c.ctrl:
			st, kind = stControl, control
		}
		at := ref{panel: 'E', kind: refText, line: r.line, off: c.off}
		if c.ctrl || c.text == "    " && c.n == 1 {
			at.kind, at.n = refAtom, c.n
			l = append(l, span{text: c.text, st: st, ref: at})
			run = -1
			continue
		}
		if last := len(l) - 1; run == kind && l[last].ref.off+len(l[last].text) == c.off {
			l[last].text = r.text[l[last].ref.off : c.off+c.n]
			continue
		}
		l = append(l, span{text: c.text, st: st, ref: at})
		run = kind
	}
	if focused && r.end && r.line == v.Line && !cursorShown {
		l = append(l, deco(" ", stCursor))
	}
	return l
}
