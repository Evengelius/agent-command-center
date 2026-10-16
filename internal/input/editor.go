// Package input is the input editor and the clipboard (doc/ui-design.md, Input
// editor and Clipboard). It knows nothing of any provider: the same editor
// serves every agent CLI.
package input

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// Motion is a cursor movement (doc/ui-design.md, Input editor, Keys).
type Motion int

// The motions Move understands.
const (
	Left Motion = iota
	Right
	Up
	Down
	LineStart
	LineEnd
	WordLeft
	WordRight
	BufferStart
	BufferEnd
)

// edit is one undo operation: the bytes removed and inserted at pos, never a
// snapshot of the text (doc/ui-design.md, Budget).
type edit struct {
	pos      int
	removed  string
	inserted string
	before   int // cursor before
	anchorB  int // selection anchor before
	after    int // cursor after
	typed    bool
}

// Editor is a multiline text buffer with a cursor and an optional selection.
// The text is kept byte for byte as typed or pasted; the cursor is a byte
// offset on a rune boundary. Storage is a Go string: the benchmarks meet the
// budget of doc/ui-design.md.
type Editor struct {
	text   string
	cursor int
	anchor int // selection anchor; no selection when equal to cursor
	goal   int // goal column in runes for Up/Down; 0 with goalSet false = unset
	goalOK bool

	undo  []edit
	redo  []edit
	merge bool // the last undo entry may take the next typed rune
}

// Text is the whole text.
func (e *Editor) Text() string { return e.text }

// Cursor is the cursor's byte offset.
func (e *Editor) Cursor() int { return e.cursor }

// SetText replaces the text, puts the cursor at its end and resets the
// history and the selection.
func (e *Editor) SetText(s string) {
	e.text, e.cursor, e.anchor, e.goalOK = s, len(s), len(s), false
	e.undo, e.redo, e.merge = nil, nil, false
}

// Clear empties the editor and its history.
func (e *Editor) Clear() { e.SetText("") }

func breaksStep(s string) bool { return s == " " || s == "\n" || s == "\t" }

// apply replaces text[start:end] with ins, records the operation and puts the
// cursor after the insertion.
func (e *Editor) apply(start, end int, ins string, typed bool) {
	op := edit{pos: start, removed: e.text[start:end], inserted: ins,
		before: e.cursor, anchorB: e.anchor, typed: typed}
	canMerge := false
	if typed && e.merge && len(e.undo) > 0 {
		last := &e.undo[len(e.undo)-1]
		canMerge = last.typed && start == end && start == last.pos+len(last.inserted)
	}
	e.text = e.text[:start] + ins + e.text[end:]
	e.cursor = start + len(ins)
	e.anchor = e.cursor
	e.goalOK = false
	e.redo = nil
	op.after = e.cursor
	if canMerge {
		last := &e.undo[len(e.undo)-1]
		last.inserted += ins
		last.after = e.cursor
	} else {
		e.undo = append(e.undo, op)
	}
	e.merge = typed && !breaksStep(ins)
}

// selRange is the selection, or the empty range at the cursor.
func (e *Editor) selRange() (int, int) {
	if e.anchor < e.cursor {
		return e.anchor, e.cursor
	}
	return e.cursor, e.anchor
}

func (e *Editor) hasSel() bool { return e.anchor != e.cursor }

// HasSelection reports whether some text is selected.
func (e *Editor) HasSelection() bool { return e.hasSel() }

// ClearSelection drops the selection; the cursor stays.
func (e *Editor) ClearSelection() { e.anchor = e.cursor }

// Insert inserts s at the cursor, unchanged, replacing the selection. A single
// rune merges into the previous typing step.
func (e *Editor) Insert(s string) {
	if s == "" && !e.hasSel() {
		return
	}
	a, b := e.selRange()
	typed := !e.hasSel() && utf8.RuneCountInString(s) == 1
	e.apply(a, b, s, typed)
}

// Paste inserts s as one undo step, replacing the selection. Each CRLF and
// each lone CR becomes one LF (doc/ui-design.md, Paste); nothing else changes.
func (e *Editor) Paste(s string) {
	if strings.IndexByte(s, '\r') >= 0 {
		s = strings.ReplaceAll(s, "\r\n", "\n")
		s = strings.ReplaceAll(s, "\r", "\n")
	}
	if s == "" && !e.hasSel() {
		return
	}
	a, b := e.selRange()
	e.apply(a, b, s, false)
}

// Newline inserts a line break.
func (e *Editor) Newline() { e.Insert("\n") }

// Backspace deletes the selection, or the rune before the cursor.
func (e *Editor) Backspace() {
	if e.hasSel() {
		e.apply(e.selRangeArgs())
		return
	}
	if e.cursor == 0 {
		return
	}
	_, n := utf8.DecodeLastRuneInString(e.text[:e.cursor])
	e.apply(e.cursor-n, e.cursor, "", false)
}

// Delete deletes the selection, or the rune after the cursor.
func (e *Editor) Delete() {
	if e.hasSel() {
		e.apply(e.selRangeArgs())
		return
	}
	if e.cursor == len(e.text) {
		return
	}
	_, n := utf8.DecodeRuneInString(e.text[e.cursor:])
	e.apply(e.cursor, e.cursor+n, "", false)
}

func (e *Editor) selRangeArgs() (int, int, string, bool) {
	a, b := e.selRange()
	return a, b, "", false
}

// DeleteWordLeft deletes the selection, or back to the previous word start.
func (e *Editor) DeleteWordLeft() {
	if e.hasSel() {
		e.apply(e.selRangeArgs())
		return
	}
	if t := e.wordLeft(e.cursor); t < e.cursor {
		e.apply(t, e.cursor, "", false)
	}
}

// DeleteWordRight deletes the selection, or up to the next word end.
func (e *Editor) DeleteWordRight() {
	if e.hasSel() {
		e.apply(e.selRangeArgs())
		return
	}
	if t := e.wordRight(e.cursor); t > e.cursor {
		e.apply(e.cursor, t, "", false)
	}
}

// Cut returns the selected text and removes it, as one undo step.
func (e *Editor) Cut() string {
	if !e.hasSel() {
		return ""
	}
	s := e.SelectedText()
	e.apply(e.selRangeArgs())
	return s
}

// SelectAll selects the whole text, the cursor at its end.
func (e *Editor) SelectAll() {
	e.anchor, e.cursor, e.goalOK, e.merge = 0, len(e.text), false, false
}

// Selection is the selected byte range.
func (e *Editor) Selection() (start, end int, ok bool) {
	if !e.hasSel() {
		return 0, 0, false
	}
	a, b := e.selRange()
	return a, b, true
}

// SelectedText is the selected text, or "".
func (e *Editor) SelectedText() string {
	a, b := e.selRange()
	return e.text[a:b]
}

func isWord(r rune) bool { return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' }

func (e *Editor) wordLeft(p int) int {
	for p > 0 {
		r, n := utf8.DecodeLastRuneInString(e.text[:p])
		if isWord(r) {
			break
		}
		p -= n
	}
	for p > 0 {
		r, n := utf8.DecodeLastRuneInString(e.text[:p])
		if !isWord(r) {
			break
		}
		p -= n
	}
	return p
}

func (e *Editor) wordRight(p int) int {
	for p < len(e.text) {
		r, n := utf8.DecodeRuneInString(e.text[p:])
		if isWord(r) {
			break
		}
		p += n
	}
	for p < len(e.text) {
		r, n := utf8.DecodeRuneInString(e.text[p:])
		if !isWord(r) {
			break
		}
		p += n
	}
	return p
}

func (e *Editor) lineStart(p int) int { return strings.LastIndexByte(e.text[:p], '\n') + 1 }

func (e *Editor) lineEnd(p int) int {
	if i := strings.IndexByte(e.text[p:], '\n'); i >= 0 {
		return p + i
	}
	return len(e.text)
}

// Move moves the cursor. With extend the selection grows from its anchor;
// without, it is cleared (Left and Right collapse to the selection's edge).
// Up and Down move by logical lines and keep a goal column counted in runes.
func (e *Editor) Move(m Motion, extend bool) {
	e.merge = false
	a, b := e.selRange()
	sel := e.hasSel()
	p := e.cursor
	vertical := m == Up || m == Down
	switch m {
	case Left:
		if sel && !extend {
			p = a
		} else if p > 0 {
			_, n := utf8.DecodeLastRuneInString(e.text[:p])
			p -= n
		}
	case Right:
		if sel && !extend {
			p = b
		} else if p < len(e.text) {
			_, n := utf8.DecodeRuneInString(e.text[p:])
			p += n
		}
	case Up, Down:
		if !e.goalOK {
			e.goal = utf8.RuneCountInString(e.text[e.lineStart(p):p])
			e.goalOK = true
		}
		var t int
		if m == Up {
			ls := e.lineStart(p)
			if ls == 0 {
				p = 0
				break
			}
			t = e.lineStart(ls - 1)
		} else {
			le := e.lineEnd(p)
			if le == len(e.text) {
				p = len(e.text)
				break
			}
			t = le + 1
		}
		end := e.lineEnd(t)
		q := t
		for i := 0; i < e.goal && q < end; i++ {
			_, n := utf8.DecodeRuneInString(e.text[q:])
			q += n
		}
		p = q
	case LineStart:
		p = e.lineStart(p)
	case LineEnd:
		p = e.lineEnd(p)
	case WordLeft:
		p = e.wordLeft(p)
	case WordRight:
		p = e.wordRight(p)
	case BufferStart:
		p = 0
	case BufferEnd:
		p = len(e.text)
	}
	if !vertical {
		e.goalOK = false
	}
	e.cursor = p
	if !extend {
		e.anchor = p
	}
}

// Left moves the cursor one rune left, clearing the selection.
func (e *Editor) Left() { e.Move(Left, false) }

// Right moves the cursor one rune right, clearing the selection.
func (e *Editor) Right() { e.Move(Right, false) }

// Home moves the cursor to the start of its line.
func (e *Editor) Home() { e.Move(LineStart, false) }

// End moves the cursor to the end of its line.
func (e *Editor) End() { e.Move(LineEnd, false) }

// Undo reverts the last step; it reports whether there was one.
func (e *Editor) Undo() bool {
	if len(e.undo) == 0 {
		return false
	}
	op := e.undo[len(e.undo)-1]
	e.undo = e.undo[:len(e.undo)-1]
	e.text = e.text[:op.pos] + op.removed + e.text[op.pos+len(op.inserted):]
	e.cursor, e.anchor, e.goalOK, e.merge = op.before, op.anchorB, false, false
	e.redo = append(e.redo, op)
	return true
}

// Redo reapplies the last undone step; it reports whether there was one.
func (e *Editor) Redo() bool {
	if len(e.redo) == 0 {
		return false
	}
	op := e.redo[len(e.redo)-1]
	e.redo = e.redo[:len(e.redo)-1]
	e.text = e.text[:op.pos] + op.inserted + e.text[op.pos+len(op.removed):]
	e.cursor, e.anchor, e.goalOK, e.merge = op.after, op.after, false, false
	e.undo = append(e.undo, op)
	return true
}

// Offset is the byte offset of byte col of line, both clamped to the text,
// moved back to a rune boundary (a mouse click in the editor).
func (e *Editor) Offset(line, col int) int {
	p := 0
	for i := 0; i < line; i++ {
		j := strings.IndexByte(e.text[p:], '\n')
		if j < 0 {
			return len(e.text)
		}
		p += j + 1
	}
	q := min(p+max(col, 0), e.lineEnd(p))
	for q > p && q < len(e.text) && !utf8.RuneStart(e.text[q]) {
		q--
	}
	return q
}

// Select selects text[from:to], the cursor at to; from equal to to only
// moves the cursor. Offsets are clamped to the text.
func (e *Editor) Select(from, to int) {
	clamp := func(p int) int { return min(max(p, 0), len(e.text)) }
	e.merge, e.goalOK = false, false
	e.anchor, e.cursor = clamp(from), clamp(to)
}

// WordBounds is the word around byte off of s, for a double click: a run of
// letters, digits and _, or the single other rune at off.
func WordBounds(s string, off int) (int, int) {
	off = min(max(off, 0), len(s))
	for off > 0 && off < len(s) && !utf8.RuneStart(s[off]) {
		off--
	}
	if off == len(s) {
		return off, off
	}
	r, n := utf8.DecodeRuneInString(s[off:])
	if !isWord(r) {
		return off, off + n
	}
	a, b := off, off
	for a > 0 {
		r, n := utf8.DecodeLastRuneInString(s[:a])
		if !isWord(r) {
			break
		}
		a -= n
	}
	for b < len(s) {
		r, n := utf8.DecodeRuneInString(s[b:])
		if !isWord(r) {
			break
		}
		b += n
	}
	return a, b
}

// Lines is the number of lines.
func (e *Editor) Lines() int { return strings.Count(e.text, "\n") + 1 }

// CursorLine is the cursor's 0-based line.
func (e *Editor) CursorLine() int { return strings.Count(e.text[:e.cursor], "\n") }

// LineRange returns lines first..first+n-1 (fewer at the end), without
// splitting the whole buffer.
func (e *Editor) LineRange(first, n int) []string {
	if first < 0 {
		n += first
		first = 0
	}
	p := 0
	for i := 0; i < first; i++ {
		j := strings.IndexByte(e.text[p:], '\n')
		if j < 0 {
			return nil
		}
		p += j + 1
	}
	out := make([]string, 0, max(n, 0))
	for len(out) < n {
		j := strings.IndexByte(e.text[p:], '\n')
		if j < 0 {
			out = append(out, e.text[p:])
			break
		}
		out = append(out, e.text[p:p+j])
		p += j + 1
	}
	return out
}

// CleanPath turns a paste into the project field's one line: surrounding
// whitespace, the trailing newline and Explorer's quotes ("Copy as path")
// removed.
func CleanPath(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexAny(s, "\r\n"); i >= 0 {
		s = strings.TrimSpace(s[:i])
	}
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		s = s[1 : len(s)-1]
	}
	return s
}
