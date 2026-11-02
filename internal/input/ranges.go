package input

import (
	"strings"
	"unicode/utf8"
)

// Paste ranges (plan 73, brief amendment 76; ui-design.md, Input editor, Long
// pastes): which bytes of the text the composer draws as one chip. A range holds
// offsets and a line count, never bytes, and the text stays the only source of
// truth: Text never reads a range, and dropping every range changes nothing sent.
// The one text change a range decides is the block's deletion the owner asks for
// at its edge, through apply, so that it can be undone.

// Range is a pasted block, text[From:To], and its logical lines.
type Range struct {
	From, To, Lines int
}

// Chip is a range as the compact form shows it: the placeholder byte's offset in
// that form, the lines and the bytes the chip stands for.
type Chip struct {
	Pos, Lines, Bytes int
}

// bound says ranges bind the editor: one exists and the draft window is closed
// (plan 73, design D10). Every use of a range asks it first, and a list failing its
// check is dropped here, its text then ordinary: a range wrong through a bug can
// neither lose nor corrupt text (the plan's invariant; design D8).
func (e *Editor) bound() bool {
	if e.Whole || len(e.ranges) == 0 {
		return false
	}
	if !e.rangesValid() {
		e.ranges = nil
		return false
	}
	return true
}

// Mark makes text[from:to], what a paste just inserted, a range, in the paste's
// undo step. One final line break stays outside it as ordinary text, so that the
// cursor's row says where typed text goes (design D7).
func (e *Editor) Mark(from, to int) {
	if from < 0 || to > len(e.text) {
		return
	}
	if to > from && e.text[to-1] == '\n' {
		to--
	}
	if to <= from {
		return
	}
	r := Range{from, to, strings.Count(e.text[from:to], "\n") + 1}
	out := make([]Range, 0, len(e.ranges)+1)
	placed := false
	for _, x := range e.ranges {
		if !placed && r.To <= x.From {
			out = append(out, r)
			placed = true
		}
		out = append(out, x)
	}
	if !placed {
		out = append(out, r)
	}
	e.ranges = out
	if n := len(e.undo); n > 0 {
		e.undo[n-1].rangesAfter = out
	}
}

// Ranges are the ranges in force, sorted; the caller keeps them unchanged.
func (e *Editor) Ranges() []Range { return e.ranges }

// edited is the ranges after text[start:end] became n bytes. While the draft
// window is open any text change ends them all (design D10). In the composer an
// edit before a range shifts it, one after it leaves it, one touching it drops it,
// its text then ordinary (design D8). A new slice each time: the undo entries hold
// the old ones.
func (e *Editor) edited(start, end, n int) []Range {
	if len(e.ranges) == 0 || e.Whole {
		return nil
	}
	d := n - (end - start)
	var out []Range
	for _, r := range e.ranges {
		switch {
		case end <= r.From:
			r.From, r.To = r.From+d, r.To+d
		case start >= r.To:
		default:
			continue
		}
		out = append(out, r)
	}
	return out
}

// rangesValid says the ranges are sorted, apart, within the text and on rune
// boundaries. Each edit keeps them so; a failure is a bug, and its answer is to show
// the text.
func (e *Editor) rangesValid() bool {
	at := 0
	for _, r := range e.ranges {
		if r.From < at || r.To <= r.From || r.To > len(e.text) {
			return false
		}
		if !utf8.RuneStart(e.text[r.From]) || r.To < len(e.text) && !utf8.RuneStart(e.text[r.To]) {
			return false
		}
		at = r.To
	}
	return true
}

// Compact is the editor over its compact form while ranges bind it (plan 73,
// design D11): the text with each range as one placeholder byte, for every
// calculation of visible lines, cursor and viewport, so that none visits a hidden
// byte. It is built from the visible segments for this call and dropped: never
// edited, stored or sent. Otherwise it is the editor itself.
func (e *Editor) Compact() *Editor {
	if !e.bound() {
		return e
	}
	n := len(e.text)
	for _, r := range e.ranges {
		n -= r.To - r.From - 1
	}
	var b strings.Builder
	b.Grow(n)
	at := 0
	for _, r := range e.ranges {
		b.WriteString(e.text[at:r.From])
		b.WriteByte(0)
		at = r.To
	}
	b.WriteString(e.text[at:])
	return &Editor{text: b.String(), cursor: e.toCompact(e.cursor), anchor: e.toCompact(e.anchor), goal: e.goal, goalOK: e.goalOK}
}

// toCompact is a text offset in the compact form: one inside a range, which only
// the draft window leaves there, is the chip's end.
func (e *Editor) toCompact(off int) int {
	hidden := 0
	for _, r := range e.ranges {
		if off <= r.From {
			break
		}
		if off < r.To {
			return r.From - hidden + 1
		}
		hidden += r.To - r.From - 1
	}
	return off - hidden
}

// FromCompact is an offset of the compact form in the text, never inside a range.
func (e *Editor) FromCompact(off int) int {
	if !e.bound() {
		return off
	}
	hidden := 0
	for _, r := range e.ranges {
		if off <= r.From-hidden {
			break
		}
		hidden += r.To - r.From - 1
	}
	return off + hidden
}

// Chips are the ranges as the compact form shows them; none unless they bind.
func (e *Editor) Chips() []Chip {
	if !e.bound() {
		return nil
	}
	out := make([]Chip, len(e.ranges))
	for i, r := range e.ranges {
		out[i] = Chip{e.toCompact(r.From), r.Lines, r.To - r.From}
	}
	return out
}

// HiddenLines is the logical lines the chips hide, kept with the ranges, so that
// the message's count needs no scan of them.
func (e *Editor) HiddenLines() int {
	if !e.bound() {
		return 0
	}
	n := 0
	for _, r := range e.ranges {
		n += r.Lines - 1
	}
	return n
}

// AfterRange says the cursor is right after a chip: what ends before it is hidden.
func (e *Editor) AfterRange() bool {
	_, ok := e.endingAt(e.cursor)
	return ok
}

// endingAt is the range ending at p, while ranges bind.
func (e *Editor) endingAt(p int) (Range, bool) {
	if e.bound() {
		for _, r := range e.ranges {
			if r.To == p {
				return r, true
			}
		}
	}
	return Range{}, false
}

// startingAt is the range starting at p, while ranges bind.
func (e *Editor) startingAt(p int) (Range, bool) {
	if e.bound() {
		for _, r := range e.ranges {
			if r.From == p {
				return r, true
			}
		}
	}
	return Range{}, false
}

// inside is the range p is strictly inside of.
func (e *Editor) inside(p int) (Range, bool) {
	for _, r := range e.ranges {
		if r.From < p && p < r.To {
			return r, true
		}
	}
	return Range{}, false
}

// Settle puts a cursor the draft window left inside a range at the chip's end, and
// lets go of a selection with an end inside one, before a composer key or paste
// acts on them; no text changes (design D10).
func (e *Editor) Settle() {
	if !e.bound() {
		return
	}
	if r, ok := e.inside(e.cursor); ok {
		e.cursor, e.anchor, e.goalOK = r.To, r.To, false
		return
	}
	if _, ok := e.inside(e.anchor); ok {
		e.anchor = e.cursor
	}
}

// wordLeftEdge is where a word deletion left of the cursor stops: at the word's
// start, or at the nearest chip's edge, searched on the compact form (design D9).
func (e *Editor) wordLeftEdge() int {
	if !e.bound() {
		return e.wordLeft(e.cursor)
	}
	c := e.Compact()
	t := c.wordLeft(c.cursor)
	for i := len(e.ranges) - 1; i >= 0; i-- {
		if p := e.toCompact(e.ranges[i].From); p < c.cursor {
			t = max(t, p+1)
			break
		}
	}
	return e.FromCompact(t)
}

// wordRightEdge is where a word deletion right of the cursor stops, likewise.
func (e *Editor) wordRightEdge() int {
	if !e.bound() {
		return e.wordRight(e.cursor)
	}
	c := e.Compact()
	t := c.wordRight(c.cursor)
	for _, r := range e.ranges {
		if p := e.toCompact(r.From); p >= c.cursor {
			t = min(t, p)
			break
		}
	}
	return e.FromCompact(t)
}
