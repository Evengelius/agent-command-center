package input

import (
	"fmt"
	"math/rand/v2"
	"reflect"
	"strings"
	"testing"
)

// marked is an editor holding text with the blocks text[from:to] as ranges, the
// cursor at the text's end.
func marked(text string, blocks ...[2]int) *Editor {
	e := &Editor{}
	e.SetText(text)
	for _, b := range blocks {
		e.Mark(b[0], b[1])
	}
	return e
}

// TestMark is design D7: the block as a range, one final line break outside it as
// ordinary text, its line count kept; sorted; part of the paste's undo step.
func TestMark(t *testing.T) {
	e := &Editor{}
	e.Paste("a\nb\nc\n")
	e.Mark(0, e.Cursor())
	if got, want := e.Ranges(), []Range{{0, 5, 3}}; !reflect.DeepEqual(got, want) {
		t.Fatalf("a final break: %+v, want %+v", got, want)
	}
	if e.Cursor() != 6 || e.Text() != "a\nb\nc\n" {
		t.Errorf("the cursor %d, the text %q", e.Cursor(), e.Text())
	}
	e = &Editor{}
	e.Insert("x ")
	e.Paste("p\nq")
	e.Mark(2, e.Cursor())
	e.Undo()
	if e.Text() != "x " || len(e.Ranges()) != 0 {
		t.Errorf("undo of the paste: %q, %+v", e.Text(), e.Ranges())
	}
	e.Redo()
	if got, want := e.Ranges(), []Range{{2, 5, 2}}; e.Text() != "x p\nq" || !reflect.DeepEqual(got, want) {
		t.Errorf("redo of the paste: %q, %+v", e.Text(), got)
	}
	// Out of order, then one between: sorted, and neither a bad call nor an empty
	// block adds any.
	e = marked("0123456789", [2]int{6, 8}, [2]int{0, 2}, [2]int{3, 5})
	if got, want := e.Ranges(), []Range{{0, 2, 1}, {3, 5, 1}, {6, 8, 1}}; !reflect.DeepEqual(got, want) {
		t.Errorf("sorted: %+v", got)
	}
	for _, b := range [][2]int{{-1, 2}, {3, 99}, {4, 4}} {
		e := marked("0123456789")
		e.Mark(b[0], b[1])
		if len(e.Ranges()) != 0 {
			t.Errorf("Mark%v: %+v", b, e.Ranges())
		}
	}
	if e := marked("\n", [2]int{0, 1}); len(e.Ranges()) != 0 {
		t.Errorf("a block of one break: %+v", e.Ranges())
	}
}

// TestRangesFollowEdits is design D8: an edit before a range shifts it, after it
// leaves it, touching it drops it; its edges take typed text outside it.
func TestRangesFollowEdits(t *testing.T) {
	block := [2]int{4, 9} // "BLOCK"
	const text = "abc BLOCK def"
	for _, c := range []struct {
		name string
		edit func(e *Editor)
		text string
		want []Range
	}{
		{"typed before", func(e *Editor) { e.Select(1, 1); e.Insert("X") }, "aXbc BLOCK def", []Range{{5, 10, 1}}},
		{"typed at its start", func(e *Editor) { e.Select(4, 4); e.Insert("X") }, "abc XBLOCK def", []Range{{5, 10, 1}}},
		{"typed at its end", func(e *Editor) { e.Select(9, 9); e.Insert("X") }, "abc BLOCKX def", []Range{{4, 9, 1}}},
		{"typed after", func(e *Editor) { e.Select(11, 11); e.Insert("X") }, "abc BLOCK dXef", []Range{{4, 9, 1}}},
		{"deleted before", func(e *Editor) { e.Select(0, 3); e.Delete() }, " BLOCK def", []Range{{1, 6, 1}}},
		{"deleted just before", func(e *Editor) { e.Select(3, 4); e.Delete() }, "abcBLOCK def", []Range{{3, 8, 1}}},
		{"deleted just after", func(e *Editor) { e.Select(9, 10); e.Delete() }, "abc BLOCKdef", []Range{{4, 9, 1}}},
		{"a selection holding it", func(e *Editor) { e.Select(2, 11); e.Delete() }, "abef", nil},
		{"exactly it", func(e *Editor) { e.Select(4, 9); e.Delete() }, "abc  def", nil},
		{"a selection into it", func(e *Editor) { e.text, e.anchor, e.cursor = text, 2, 6; e.Delete() }, "abLOCK def"[:2] + "OCK def", nil},
		{"typed over it", func(e *Editor) { e.Select(3, 10); e.Insert("X") }, "abcXdef", nil},
		{"pasted over it", func(e *Editor) { e.Select(4, 9); e.Paste("new") }, "abc new def", nil},
	} {
		e := marked(text, block)
		c.edit(e)
		if e.Text() != c.text || !reflect.DeepEqual(e.Ranges(), c.want) {
			t.Errorf("%s: %q %+v, want %q %+v", c.name, e.Text(), e.Ranges(), c.text, c.want)
		}
		// Undo gives the text and its range back, whatever the edit did to it.
		e.Undo()
		if e.Text() != text || !reflect.DeepEqual(e.Ranges(), []Range{{4, 9, 1}}) {
			t.Errorf("%s, undone: %q %+v", c.name, e.Text(), e.Ranges())
		}
		e.Redo()
		if e.Text() != c.text || !reflect.DeepEqual(e.Ranges(), c.want) {
			t.Errorf("%s, redone: %q %+v, want %q %+v", c.name, e.Text(), e.Ranges(), c.text, c.want)
		}
	}
	// SetText and Clear end them all: a send, a failed send's text, a session's close.
	e := marked(text, block)
	e.SetText(text)
	if len(e.Ranges()) != 0 {
		t.Errorf("SetText: %+v", e.Ranges())
	}
	e = marked(text, block)
	e.Clear()
	if len(e.Ranges()) != 0 || e.Undo() {
		t.Errorf("Clear: %+v", e.Ranges())
	}
	// A slice an undo entry holds is never written: the edits make new ones.
	e = marked(text, block)
	held := e.Ranges()
	e.Select(0, 0)
	e.Insert("X")
	if !reflect.DeepEqual(held, []Range{{4, 9, 1}}) {
		t.Errorf("the ranges held were written: %+v", held)
	}
}

// TestChipDeletion is design D9: Backspace or Ctrl+Backspace right after a chip,
// Delete or Ctrl+Delete right before it, take the whole block in one undo step;
// anywhere else, today's; a word deletion stops at a chip's edge; adjacent chips go
// one at a time.
func TestChipDeletion(t *testing.T) {
	const text = "ab BLOCK cd"
	for _, c := range []struct {
		name string
		at   int
		del  func(e *Editor)
		text string
	}{
		{"Backspace after", 8, (*Editor).Backspace, "ab  cd"},
		{"Ctrl+Backspace after", 8, (*Editor).DeleteWordLeft, "ab  cd"},
		{"Delete before", 3, (*Editor).Delete, "ab  cd"},
		{"Ctrl+Delete before", 3, (*Editor).DeleteWordRight, "ab  cd"},
		{"Backspace elsewhere", 2, (*Editor).Backspace, "a BLOCK cd"},
		{"Delete elsewhere", 9, (*Editor).Delete, "ab BLOCK d"},
		{"Backspace before", 3, (*Editor).Backspace, "abBLOCK cd"},
		{"Delete after", 8, (*Editor).Delete, "ab BLOCKcd"},
		{"Ctrl+Backspace one away stops at the edge", 9, (*Editor).DeleteWordLeft, "ab BLOCKcd"},
		{"Ctrl+Delete one away stops at the edge", 2, (*Editor).DeleteWordRight, "abBLOCK cd"},
		{"Ctrl+Backspace before the block", 2, (*Editor).DeleteWordLeft, " BLOCK cd"},
		{"Ctrl+Delete after the block", 9, (*Editor).DeleteWordRight, "ab BLOCK "},
	} {
		e := marked(text, [2]int{3, 8})
		e.Select(c.at, c.at)
		c.del(e)
		if e.Text() != c.text {
			t.Errorf("%s: %q, want %q", c.name, e.Text(), c.text)
		}
		if !e.Undo() || e.Text() != text || !reflect.DeepEqual(e.Ranges(), []Range{{3, 8, 1}}) || e.Cursor() != c.at {
			t.Errorf("%s, undone: %q %+v, cursor %d, want %d", c.name, e.Text(), e.Ranges(), e.Cursor(), c.at)
		}
		if e.Undo() {
			t.Errorf("%s: more than one undo step", c.name)
		}
	}
	// In the draft window the same keys are today's.
	e := marked(text, [2]int{3, 8})
	e.Whole = true
	e.Select(8, 8)
	e.Backspace()
	if e.Text() != "ab BLOC cd" || len(e.Ranges()) != 0 {
		t.Errorf("the draft window: %q %+v", e.Text(), e.Ranges())
	}
	// Adjacent chips, one at a time, the other byte for byte.
	e = marked("AAABBB", [2]int{0, 3}, [2]int{3, 6})
	e.Select(3, 3)
	e.Backspace()
	if e.Text() != "BBB" || !reflect.DeepEqual(e.Ranges(), []Range{{0, 3, 1}}) {
		t.Errorf("the first of two: %q %+v", e.Text(), e.Ranges())
	}
	e.Undo()
	e.Select(3, 3)
	e.Delete()
	if e.Text() != "AAA" || !reflect.DeepEqual(e.Ranges(), []Range{{0, 3, 1}}) {
		t.Errorf("the second of two: %q %+v", e.Text(), e.Ranges())
	}
}

// TestChipMotions is design D9's cursor: every motion passes a chip as one unit and
// stops on none of its bytes; a selection holding a chip holds it whole.
func TestChipMotions(t *testing.T) {
	const text = "one two\nab LINE1\nLINE2\nLINE3 cd\nlast"
	from, to := strings.Index(text, "LINE1"), strings.Index(text, " cd")
	e := marked(text, [2]int{from, to})
	if got, want := e.Compact().Text(), "one two\nab \x00 cd\nlast"; got != want {
		t.Fatalf("the compact form %q, want %q", got, want)
	}
	move := func(start int, m Motion, extend bool) *Editor {
		e.Select(start, start)
		e.goalOK = false
		e.Move(m, extend)
		return e
	}
	for _, c := range []struct {
		name  string
		start int
		m     Motion
		want  int
	}{
		{"Right onto it", from - 1, Right, from},
		{"Right over it", from, Right, to},
		{"Left over it", to, Left, from},
		{"Left to it", to + 1, Left, to},
		{"Up onto its line", strings.Index(text, "last"), Up, from - 3},
		{"Up from under its end", strings.Index(text, "last") + 4, Up, to},
		{"Down onto its line", 1, Down, from - 2},
		{"Down at its column ends after it", 3, Down, from},
		{"Down past its column", 5, Down, to + 1},
		{"Home from after it", to + 2, LineStart, from - 3},
		{"End from before it", from - 3, LineEnd, to + 3},
		{"Word left from after it", to + 3, WordLeft, to + 1},
		{"Word left from its end", to, WordLeft, from - 3},
		{"Word right from before it", from, WordRight, to + 3},
		{"Word right from its start", from - 1, WordRight, to + 3},
		{"Buffer start", to, BufferStart, 0},
		{"Buffer end", from, BufferEnd, len(text)},
	} {
		if got := move(c.start, c.m, false); got.Cursor() != c.want {
			t.Errorf("%s from %d: %d, want %d", c.name, c.start, got.Cursor(), c.want)
		}
		if p := e.Cursor(); p > from && p < to {
			t.Errorf("%s: the cursor at %d inside the chip", c.name, p)
		}
	}
	// The selection: its ends never inside, the chip's bytes all in it.
	e.Select(from-1, from-1)
	e.Move(Right, true)
	e.Move(Right, true)
	if a, b, ok := e.Selection(); !ok || a != from-1 || b != to {
		t.Errorf("Shift+→ twice: %d..%d", a, b)
	}
	if got, want := e.SelectedText(), " "+text[from:to]; got != want {
		t.Errorf("selected %q, want %q", got, want)
	}
	e.Select(to+1, to+1)
	e.Move(Left, true)
	e.Move(Left, true)
	if a, b, ok := e.Selection(); !ok || a != from || b != to+1 {
		t.Errorf("Shift+← twice: %d..%d", a, b)
	}
	e.Move(Left, false) // collapses to its start
	if e.Cursor() != from || e.HasSelection() {
		t.Errorf("← with the selection: %d", e.Cursor())
	}
	// A cut of a selection holding the chip, then the text it leaves.
	e.Select(from-1, to+1)
	if got, want := e.Cut(), text[from-1:to+1]; got != want || len(e.Ranges()) != 0 {
		t.Errorf("cut %q, want %q; ranges %+v", got, want, e.Ranges())
	}
	// With Whole, the same text and the same keys are today's: stops inside.
	e = marked(text, [2]int{from, to})
	e.Whole = true
	e.Select(from, from)
	e.Move(Right, false)
	if e.Cursor() != from+1 {
		t.Errorf("the draft window's →: %d", e.Cursor())
	}
}

// TestCompactMapping is design D11's mapping: the compact form holds no hidden byte,
// its offsets map back to the text and never inside a range, the chips' places and
// counts are the ranges', and with no range it is the editor itself, with nothing
// allocated.
func TestCompactMapping(t *testing.T) {
	const text = "x\x00y AAA\nAAA z BBB\x00"
	a, b := strings.Index(text, "AAA"), strings.Index(text, "BBB")
	e := marked(text, [2]int{a, a + 7}, [2]int{b, b + 3})
	c := e.Compact()
	if want := "x\x00y \x00 z \x00\x00"; c.Text() != want {
		t.Fatalf("compact %q, want %q", c.Text(), want)
	}
	if strings.Contains(c.Text(), "A") || strings.Contains(c.Text(), "B") {
		t.Errorf("a hidden byte in %q", c.Text())
	}
	for _, p := range []struct{ shown, off int }{{0, 0}, {4, a}, {5, a + 7}, {8, b}, {9, b + 3}, {10, b + 4}} {
		if got := e.FromCompact(p.shown); got != p.off {
			t.Errorf("FromCompact(%d) = %d, want %d", p.shown, got, p.off)
		}
		if got := e.toCompact(p.off); got != p.shown {
			t.Errorf("toCompact(%d) = %d, want %d", p.off, got, p.shown)
		}
	}
	// Every offset of the compact form maps outside the ranges; a text offset inside
	// one is the chip's end, so the cursor left there by the draft window draws there.
	for p := 0; p <= len(c.Text()); p++ {
		if _, in := e.inside(e.FromCompact(p)); in {
			t.Errorf("FromCompact(%d) = %d, inside a range", p, e.FromCompact(p))
		}
	}
	if got := e.toCompact(a + 2); got != 5 {
		t.Errorf("toCompact inside the first: %d, want its end 5", got)
	}
	if got, want := e.Chips(), []Chip{{4, 2, 7}, {8, 1, 3}}; !reflect.DeepEqual(got, want) {
		t.Errorf("chips %+v, want %+v", got, want)
	}
	if e.HiddenLines() != 1 || c.Lines()+e.HiddenLines() != e.Lines() {
		t.Errorf("hidden %d, compact %d, text %d lines", e.HiddenLines(), c.Lines(), e.Lines())
	}
	// With Whole or no range: the editor itself, identity mappings, no chips.
	e.Whole = true
	if e.Compact() != e || e.FromCompact(5) != 5 || e.Chips() != nil || e.HiddenLines() != 0 {
		t.Errorf("Whole: compact %p of %p, chips %+v", e.Compact(), e, e.Chips())
	}
	plain := marked(strings.Repeat("words\n", 100))
	if n := testing.AllocsPerRun(100, func() {
		if plain.Compact() != plain || plain.FromCompact(7) != 7 || plain.Chips() != nil || plain.HiddenLines() != 0 || plain.AfterRange() {
			t.Fatal("no range: not the editor")
		}
		plain.Settle()
	}); n != 0 {
		t.Errorf("no range: %v allocations", n)
	}
}

// TestSettle is design D10: a cursor the draft window left inside a chip goes to its
// end; a selection with an end inside one ends; no text changes.
func TestSettle(t *testing.T) {
	const text = "ab BLOCK cd"
	e := marked(text, [2]int{3, 8})
	e.Whole = true
	e.Select(5, 5)
	e.Whole = false
	e.Settle()
	if e.Cursor() != 8 || e.HasSelection() || e.Text() != text {
		t.Errorf("a cursor inside: %d, selection %v, %q", e.Cursor(), e.HasSelection(), e.Text())
	}
	e.Whole = true
	e.Select(1, 5)
	e.Whole = false
	e.Settle()
	if e.Cursor() != 8 || e.HasSelection() {
		t.Errorf("the free end inside: %d, selection %v", e.Cursor(), e.HasSelection())
	}
	e.Whole = true
	e.Select(5, 10)
	e.Whole = false
	e.Settle()
	if e.Cursor() != 10 || e.HasSelection() {
		t.Errorf("the anchor inside: %d, selection %v", e.Cursor(), e.HasSelection())
	}
	e.Select(1, 10)
	e.Settle()
	if a, b, ok := e.Selection(); !ok || a != 1 || b != 10 {
		t.Errorf("a selection holding the chip whole: %d..%d, %v", a, b, ok)
	}
	// With Whole on, it leaves everything where it is.
	e.Whole = true
	e.Select(5, 5)
	e.Settle()
	if e.Cursor() != 5 {
		t.Errorf("Whole: %d", e.Cursor())
	}
}

// TestDraftWindowRanges is design D10: with Whole nothing binds, a text change ends
// every range, undo back to the bytes brings them back, and with no change they are
// there when the draft window closes.
func TestDraftWindowRanges(t *testing.T) {
	const text = "ab BLOCK cd"
	e := marked(text, [2]int{3, 8})
	e.Whole = true
	e.Move(BufferStart, false)
	e.Move(Right, false)
	e.Whole = false
	if !reflect.DeepEqual(e.Ranges(), []Range{{3, 8, 1}}) || e.Cursor() != 1 {
		t.Errorf("no change: %+v, cursor %d", e.Ranges(), e.Cursor())
	}
	e.Whole = true
	e.Select(0, 0)
	e.Insert("X") // far from the range: it ends all the same
	if len(e.Ranges()) != 0 {
		t.Errorf("a text change in the window: %+v", e.Ranges())
	}
	e.Whole = false
	if e.Compact() != e {
		t.Errorf("after it, no compact form")
	}
	e.Undo()
	if e.Text() != text || !reflect.DeepEqual(e.Ranges(), []Range{{3, 8, 1}}) {
		t.Errorf("undone: %q %+v", e.Text(), e.Ranges())
	}
	e.Redo()
	if e.Text() != "X"+text || len(e.Ranges()) != 0 {
		t.Errorf("redone: %q %+v", e.Text(), e.Ranges())
	}
	// Typed runes merged in the window: their one step's undo brings them back too.
	e = marked(text, [2]int{3, 8})
	e.Whole = true
	e.Select(9, 9)
	e.Insert("x")
	e.Insert("y")
	e.Undo()
	if e.Text() != text || !reflect.DeepEqual(e.Ranges(), []Range{{3, 8, 1}}) {
		t.Errorf("merged, undone: %q %+v", e.Text(), e.Ranges())
	}
	// A paste in the window never marks: the application asks Mark only in the composer.
}

// TestInvalidRanges is the plan's invariant: invalid metadata never causes content
// loss or corruption: a list failing its check is dropped at its next use, the text
// byte for byte, every key then today's.
func TestInvalidRanges(t *testing.T) {
	const text = "ab BLOCK cd é"
	for _, c := range []struct {
		name string
		bad  []Range
	}{
		{"past the end", []Range{{3, 99, 1}}},
		{"backwards", []Range{{8, 3, 1}}},
		{"empty", []Range{{3, 3, 1}}},
		{"negative", []Range{{-1, 3, 1}}},
		{"overlapping", []Range{{3, 8, 1}, {5, 9, 1}}},
		{"out of order", []Range{{9, 10, 1}, {3, 8, 1}}},
		{"inside a rune", []Range{{3, len(text) - 1, 1}}},
		{"starting inside a rune", []Range{{len(text) - 1, len(text), 1}}},
	} {
		for name, use := range map[string]func(e *Editor){
			"Compact":   func(e *Editor) { e.Compact() },
			"Chips":     func(e *Editor) { e.Chips() },
			"Hidden":    func(e *Editor) { e.HiddenLines() },
			"Move":      func(e *Editor) { e.Move(Right, false) },
			"Backspace": func(e *Editor) { e.Backspace(); e.Undo() },
			"Delete":    func(e *Editor) { e.Delete(); e.Undo() },
			"WordLeft":  func(e *Editor) { e.DeleteWordLeft(); e.Undo() },
			"WordRight": func(e *Editor) { e.DeleteWordRight(); e.Undo() },
			"Settle":    func(e *Editor) { e.Settle() },
			"After":     func(e *Editor) { e.AfterRange() },
		} {
			e := marked(text)
			e.Select(8, 8)
			e.ranges = c.bad
			use(e)
			if e.Text() != text {
				t.Errorf("%s, %s: the text became %q", c.name, name, e.Text())
			}
			if len(e.Ranges()) != 0 {
				t.Errorf("%s, %s: not dropped: %+v", c.name, name, e.Ranges())
			}
		}
	}
	// Undo into a state whose list is bad: dropped when read, the text its own.
	e := marked(text, [2]int{3, 8})
	e.Select(0, 0)
	e.Insert("x")
	e.undo[len(e.undo)-1].rangesBefore = []Range{{3, 99, 1}}
	e.Undo()
	if e.Compact() != e || e.Text() != text || len(e.Ranges()) != 0 {
		t.Errorf("a bad list undone into: %q %+v", e.Text(), e.Ranges())
	}
}

// rangeRig is the model of criterion 73.20: e, under every operation, against p, a
// plain editor given the same edits, whole-block deletions as selections: the text
// and the cursor equal after every operation, and p's own methods wherever nothing
// binds e, so that with no range the editor is today's.
type rangeRig struct {
	t      testing.TB
	e, p   Editor
	blocks map[string]bool
	n      int
}

func newRangeRig(t testing.TB) *rangeRig { return &rangeRig{t: t, blocks: map[string]bool{}} }

// align gives p e's cursor and goal when they differ: motions over chips are e's.
func (r *rangeRig) align() {
	r.p.goal, r.p.goalOK = r.e.goal, r.e.goalOK
	if r.p.cursor != r.e.cursor || r.p.anchor != r.e.anchor {
		goal, ok := r.p.goal, r.p.goalOK
		r.p.Select(r.e.anchor, r.e.cursor)
		r.p.goal, r.p.goalOK = goal, ok
	}
}

// block is a new block: its first line says which, its shape chosen by arg.
func (r *rangeRig) block(arg byte) string {
	r.n++
	head := fmt.Sprintf("block%d", r.n)
	switch arg % 5 {
	case 0:
		return head + "\nsecond\nthird\nfourth"
	case 1:
		return head + "\nsecond\nthird\n"
	case 2:
		return head + " " + strings.Repeat("日本語 word\t", 30)
	case 3:
		return head + "\n\n\n\n\\"
	}
	return "\n\n" + head + "\n\n"
}

// step is one operation, chosen by op and arg, then the invariants.
func (r *rangeRig) step(op, arg byte) {
	e, p := &r.e, &r.p
	settled := arg&0x80 == 0
	if settled {
		e.Settle()
	}
	r.align()
	bound := e.bound()
	sel := e.hasSel()
	before := e.text
	wholeEdit := false
	switch op % 30 {
	case 0, 1, 2:
		s := string(rune('a' + arg%26))
		e.Insert(s)
		p.Insert(s)
		wholeEdit = true
	case 3:
		s := []string{" ", "\t", "日", "é", "\\", "_", "👍", "\x00"}[arg%8]
		e.Insert(s)
		p.Insert(s)
		wholeEdit = true
	case 4:
		e.Newline()
		p.Newline()
		wholeEdit = true
	case 5:
		s := fmt.Sprintf("p%d", arg)
		if arg&1 == 1 {
			s += "\r\nnext\rline"
		}
		e.Paste(s)
		p.Paste(s)
		wholeEdit = true
	case 6, 7:
		// A long paste, as the application's paste site does it: marked only with the
		// draft window closed.
		s := r.block(arg)
		from, _ := e.selRange()
		e.Paste(s)
		p.Paste(s)
		if !e.Whole {
			e.Mark(from, e.Cursor())
			r.blocks[strings.TrimSuffix(s, "\n")] = true
		}
		wholeEdit = true
	case 8:
		rg, chip := e.endingAt(e.cursor)
		e.Backspace()
		if !sel && chip {
			p.Select(rg.From, rg.To)
		}
		p.Backspace()
		wholeEdit = true
	case 9:
		rg, chip := e.startingAt(e.cursor)
		e.Delete()
		if !sel && chip {
			p.Select(rg.To, rg.From)
		}
		p.Delete()
		wholeEdit = true
	case 10:
		c := e.cursor
		rg, chip := e.endingAt(c)
		ranges := e.ranges
		e.DeleteWordLeft()
		switch {
		case !bound || sel:
			p.DeleteWordLeft()
		case chip:
			p.Select(rg.From, rg.To)
			p.Backspace()
		case e.cursor < c:
			for _, x := range ranges {
				// With the cursor left inside a block, which no composer key leaves it
				// (design D10), the deletion runs from there and ends the range.
				if settled && x.From < c && x.To > e.cursor {
					r.t.Fatalf("a word deletion %d..%d crossed into %+v", e.cursor, c, x)
				}
			}
			p.Select(e.cursor, c)
			p.Backspace()
		}
		wholeEdit = true
	case 11:
		c := e.cursor
		rg, chip := e.startingAt(c)
		ranges := e.ranges
		e.DeleteWordRight()
		switch {
		case !bound || sel:
			p.DeleteWordRight()
		case chip:
			p.Select(rg.To, rg.From)
			p.Delete()
		case len(e.text) < len(before):
			gone := len(before) - len(e.text)
			for _, x := range ranges {
				if settled && x.From < c+gone && x.To > c {
					r.t.Fatalf("a word deletion %d..%d crossed into %+v", c, c+gone, x)
				}
			}
			p.Select(c+gone, c)
			p.Delete()
		}
		wholeEdit = true
	case 12:
		if a, b := e.Cut(), p.Cut(); a != b {
			r.t.Fatalf("cut %q, the model %q", a, b)
		}
		wholeEdit = true
	case 13, 14, 15, 16, 17, 18, 19, 20, 21, 22:
		m, extend := Motion(op%30-13), arg&1 == 1
		e.Move(m, extend)
		if bound {
			goal, ok := e.goal, e.goalOK
			p.Select(e.anchor, e.cursor)
			p.goal, p.goalOK = goal, ok
		} else {
			p.Move(m, extend)
		}
	case 23:
		e.SelectAll()
		p.SelectAll()
	case 24:
		// A click or a drag, through the compact form as the mouse goes.
		c := e.Compact()
		line, col := int(arg)%(c.Lines()+1), int(arg>>3)
		to := e.FromCompact(c.Offset(line, col))
		from := to
		if arg&1 == 1 {
			from = e.FromCompact(c.Offset(int(arg>>1)%(c.Lines()+1), int(arg>>2)))
		}
		e.Select(from, to)
		p.Select(from, to)
	case 25, 26:
		if a, b := e.Undo(), p.Undo(); a != b {
			r.t.Fatalf("undo: %v, the model %v", a, b)
		}
		if x, in := e.inside(p.cursor); in && !e.Whole {
			// The step was the draft window's, its cursor inside the block it gives
			// back: the chip's end (design D10).
			if e.cursor != x.To {
				r.t.Fatalf("undo left the cursor %d, inside %+v", e.cursor, x)
			}
			r.align()
		}
	case 27:
		if a, b := e.Redo(), p.Redo(); a != b {
			r.t.Fatalf("redo: %v, the model %v", a, b)
		}
	case 28:
		if arg%16 == 0 { // rare: it ends the history
			s := fmt.Sprintf("restored %d\nline", arg)
			e.SetText(s)
			p.SetText(s)
		}
	case 29:
		e.Whole = !e.Whole // Ctrl+G, Esc: a cursor inside a block stays till the next key
		settled = false
	}
	if e.Whole && wholeEdit && e.text != before && len(e.ranges) != 0 {
		r.t.Fatalf("a text change with the draft window open left %+v", e.ranges)
	}
	r.check(settled)
}

// check is the invariants of criterion 73.20 and of design D11.
func (r *rangeRig) check(settled bool) {
	e, p := &r.e, &r.p
	if e.text != p.text {
		r.t.Fatalf("the text %q, the model's %q", e.text, p.text)
	}
	if e.cursor != p.cursor {
		r.t.Fatalf("the cursor %d, the model's %d, in %q", e.cursor, p.cursor, e.text)
	}
	at := 0
	for _, x := range e.ranges {
		if x.From < at || x.To <= x.From || x.To > len(e.text) {
			r.t.Fatalf("ranges %+v in %d bytes", e.ranges, len(e.text))
		}
		at = x.To
		b := e.text[x.From:x.To]
		if strings.Count(b, "\n")+1 != x.Lines {
			r.t.Fatalf("range %+v holds %q", x, b)
		}
		if !r.blocks[b] {
			r.t.Fatalf("range %+v holds %q, no block pasted", x, b)
		}
	}
	if e.Whole {
		if e.Compact() != e || e.Chips() != nil {
			r.t.Fatal("ranges bind the draft window")
		}
		return
	}
	if settled {
		if x, in := e.inside(e.cursor); in {
			r.t.Fatalf("the cursor %d inside %+v", e.cursor, x)
		}
		if x, in := e.inside(e.anchor); in {
			r.t.Fatalf("the anchor %d inside %+v", e.anchor, x)
		}
	}
	if len(e.ranges) == 0 {
		return
	}
	c := e.Compact()
	want := len(e.text)
	var b strings.Builder
	from := 0
	for _, x := range e.ranges {
		want -= x.To - x.From - 1
		b.WriteString(e.text[from:x.From])
		b.WriteByte(0)
		from = x.To
	}
	b.WriteString(e.text[from:])
	if c.text != b.String() || len(c.text) != want {
		r.t.Fatalf("the compact form %q, want %q", c.text, b.String())
	}
	if c.Lines()+e.HiddenLines() != e.Lines() {
		r.t.Fatalf("%d lines + %d hidden, the text has %d", c.Lines(), e.HiddenLines(), e.Lines())
	}
	for i, chip := range e.Chips() {
		x := e.ranges[i]
		if c.text[chip.Pos] != 0 || chip.Lines != x.Lines || chip.Bytes != x.To-x.From {
			r.t.Fatalf("chip %+v of %+v in %q", chip, x, c.text)
		}
		if e.FromCompact(chip.Pos) != x.From || e.FromCompact(chip.Pos+1) != x.To {
			r.t.Fatalf("chip %+v maps to %d..%d, range %+v", chip, e.FromCompact(chip.Pos), e.FromCompact(chip.Pos+1), x)
		}
	}
	if settled && e.FromCompact(c.cursor) != e.cursor {
		r.t.Fatalf("the cursor %d shown at %d maps to %d", e.cursor, c.cursor, e.FromCompact(c.cursor))
	}
}

// TestEditorRangesModel is criterion 73.20, on random sequences of every operation,
// in both modes.
func TestEditorRangesModel(t *testing.T) {
	for seed := range uint64(60) {
		rng := rand.New(rand.NewPCG(seed, 73))
		r := newRangeRig(t)
		for range 1500 {
			r.step(byte(rng.IntN(256)), byte(rng.IntN(256)))
		}
	}
}

// TestEditorPlainModel is plan 73's P3: an editor no range ever touched is today's,
// its anchor and goal included, against the model's own methods on every operation.
func TestEditorPlainModel(t *testing.T) {
	for seed := range uint64(30) {
		rng := rand.New(rand.NewPCG(seed, 3))
		r := newRangeRig(t)
		for range 1500 {
			op, arg := byte(rng.IntN(256)), byte(rng.IntN(256))
			if o := op % 30; o == 6 || o == 7 {
				continue // the one operation that makes a range
			}
			r.step(op, arg&0x7f)
			if r.e.anchor != r.p.anchor || len(r.e.ranges) != 0 {
				t.Fatalf("anchor %d, the model's %d; ranges %+v", r.e.anchor, r.p.anchor, r.e.ranges)
			}
		}
	}
}

// FuzzEditorRanges is criterion 73.20: no sequence of operations breaks the model.
func FuzzEditorRanges(f *testing.F) {
	f.Add([]byte{6, 1, 8, 0, 25, 0, 27, 0})
	f.Add([]byte{0, 1, 6, 0, 0, 2, 7, 3, 13, 1, 14, 1, 12, 0, 25, 0})
	f.Add([]byte{6, 4, 29, 0, 0, 1, 29, 0, 25, 0, 13, 0, 8, 0})
	f.Add([]byte{6, 2, 19, 0, 19, 1, 11, 0, 25, 0, 24, 0x41, 10, 0, 3, 7})
	f.Add([]byte{6, 3, 6, 0, 29, 0, 19, 0x80, 29, 0, 0, 0x80, 25, 0, 26, 0, 27, 0})
	f.Add([]byte("B1Y010Y0)\xff0000000000")) // a word deletion with the cursor left inside a block
	f.Fuzz(func(t *testing.T, ops []byte) {
		r := newRangeRig(t)
		for i := 0; i+1 < len(ops); i += 2 {
			r.step(ops[i], ops[i+1])
		}
	})
}
