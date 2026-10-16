// Package input is the input editor and the clipboard (doc/ui-design.md, Input
// editor and Clipboard). It knows nothing of any provider: the same editor
// serves every agent CLI.
package input

import (
	"strings"
	"unicode/utf8"
)

// Editor is a multiline text buffer with a cursor. The text is kept byte for
// byte as typed or pasted; the cursor is a byte offset on a rune boundary.
// Step 0 holds the essentials; step 3 adds the rest of the editing keys.
type Editor struct {
	text   string
	cursor int
}

// Text is the whole text.
func (e *Editor) Text() string { return e.text }

// Cursor is the cursor's byte offset.
func (e *Editor) Cursor() int { return e.cursor }

// SetText replaces the text and puts the cursor at its end.
func (e *Editor) SetText(s string) { e.text, e.cursor = s, len(s) }

// Clear empties the editor.
func (e *Editor) Clear() { e.SetText("") }

// Insert inserts s at the cursor, unchanged: a paste keeps every byte,
// including its newlines, and never submits.
func (e *Editor) Insert(s string) {
	e.text = e.text[:e.cursor] + s + e.text[e.cursor:]
	e.cursor += len(s)
}

// Newline inserts a line break.
func (e *Editor) Newline() { e.Insert("\n") }

// Backspace deletes the rune before the cursor.
func (e *Editor) Backspace() {
	if e.cursor == 0 {
		return
	}
	_, n := utf8.DecodeLastRuneInString(e.text[:e.cursor])
	e.text = e.text[:e.cursor-n] + e.text[e.cursor:]
	e.cursor -= n
}

// Delete deletes the rune after the cursor.
func (e *Editor) Delete() {
	if e.cursor == len(e.text) {
		return
	}
	_, n := utf8.DecodeRuneInString(e.text[e.cursor:])
	e.text = e.text[:e.cursor] + e.text[e.cursor+n:]
}

// Left moves the cursor one rune left.
func (e *Editor) Left() {
	if e.cursor > 0 {
		_, n := utf8.DecodeLastRuneInString(e.text[:e.cursor])
		e.cursor -= n
	}
}

// Right moves the cursor one rune right.
func (e *Editor) Right() {
	if e.cursor < len(e.text) {
		_, n := utf8.DecodeRuneInString(e.text[e.cursor:])
		e.cursor += n
	}
}

// Home moves the cursor to the start of its line.
func (e *Editor) Home() { e.cursor = strings.LastIndexByte(e.text[:e.cursor], '\n') + 1 }

// End moves the cursor to the end of its line.
func (e *Editor) End() {
	if i := strings.IndexByte(e.text[e.cursor:], '\n'); i >= 0 {
		e.cursor += i
		return
	}
	e.cursor = len(e.text)
}

// Lines is the number of lines.
func (e *Editor) Lines() int { return strings.Count(e.text, "\n") + 1 }

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
