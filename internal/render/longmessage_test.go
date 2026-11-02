package render

import (
	"fmt"
	"math/rand/v2"
	"strings"
	"testing"

	"github.com/Evengelius/agent-command-center/internal/event"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// rowsDrawn is design D1's definition: the rows entryLines and cardText draw for
// text at width w, an empty line a row.
func rowsDrawn(text string, w int) int {
	n := 0
	for _, l := range strings.Split(text, "\n") {
		n += len(wrapRanges(l, w, w))
	}
	return n
}

// cardRows is the rows the renderer itself gives text at width w: cardText, which zone B
// draws an ordinary message with, over entryLines' lines. rowsDrawn is checked against it.
func cardRows(text string, w int) int {
	lines := entryLines(nil, state.Entry{Role: "user", Text: text})
	return len(cardText(nil, nil, &frame{kind: cardYou, w: w + 3}, 0, lines, w, nil, nil, stBright))
}

// joinedCell is n wide characters, each joined to the next by U+200D: one cell as wide
// as they all are, which a wrap puts on one row, a row too narrow for it showing "…".
func joinedCell(n int) string {
	return strings.Repeat("日"+string(rune(0x200D)), max(n-1, 0)) + "日"
}

// messageWidths are the widths of criterion 73.1: zone B's content at 100x30,
// 80x24, 120x40 and 200x60.
var messageWidths = []int{49, 69, 73, 129}

// boundWidths are those and the narrow ones, where a row is narrower than a tab and
// the bound takes a tab's 4 cells for the row's, and just over them.
var boundWidths = []int{1, 2, 3, 4, 5, 6, 7, 8, 49, 69, 73, 129}

// TestLongMessageDefinition is criterion 73.1: LongMessage is rows > 3 for the rows
// of design D1, on what its bound could get wrong.
func TestLongMessageDefinition(t *testing.T) {
	for _, w := range messageWidths {
		x := func(n int) string { return strings.Repeat("x", n) }
		for _, c := range []struct {
			name string
			text string
			want bool
		}{
			{"three lines", "a\nb\nc", false},
			{"a final line break draws a blank row", "a\nb\nc\n", true},
			{"leading breaks count", "\n\n\nabc", true},
			{"trailing breaks count", "a\n\n\n\n", true},
			{"only three breaks", "\n\n\n", true},
			{"only two breaks", "\n\n", false},
			{"empty", "", false},
			{"a line filling three rows", x(3 * w), false},
			{"one cell more", x(3*w + 1), true},
			{"a line filling two rows and a line", x(2*w) + "\nb", false},
			{"a line filling three rows and a line", x(3*w) + "\nb", true},
			{"tabs filling three rows", strings.Repeat("\t", 3*(w/4)), false},
			{"tabs over three rows", strings.Repeat("\t", 3*w), true},
			{"control characters filling three rows", strings.Repeat("\x01", 3*(w/2)), false},
			{"control characters over three rows", strings.Repeat("\x01", 2*w), true},
			{"a byte that is not UTF-8", strings.Repeat("\xff", 3*(w/4)), false},
			{"C1 controls", strings.Repeat("\u0085", 2*w), true},
			{"wide clusters at a row's edge", strings.Repeat("日", 3*(w/2)) + "x", w%2 == 0},
			{"wide clusters over three rows", strings.Repeat("日", 2*w), true},
			{"emoji clusters", strings.Repeat("👍🏽", 2*w), true},
			{"a long word after a short one", "a " + x(3*w-2), true},
			{"spaces dropped at three rows' edges", strings.Repeat(x(w)+" ", 3), false},
			{"the same and one cell", strings.Repeat(x(w)+" ", 3) + "y", true},
			{"only spaces", strings.Repeat(" ", 3*w), false},
			{"far over the bound", x(100 * w), true},
		} {
			got, sum := LongMessage(c.text, w), rowsDrawn(c.text, w)
			if got != (sum > 3) {
				t.Errorf("w=%d, %s: LongMessage %v, but %d rows are drawn", w, c.name, got, sum)
			}
			if got != c.want {
				t.Errorf("w=%d, %s: LongMessage %v, want %v (%d rows)", w, c.name, got, c.want, sum)
			}
			if long, _ := messageScan(NewWrapCache(), c.text, w); long != got {
				t.Errorf("w=%d, %s: through the cache %v, without it %v", w, c.name, long, got)
			}
		}
	}
}

// FuzzLongMessage is criterion 73.1: no text and no width where the bounded answer
// differs from the sum of design D1, itself the rows cardText draws; and 73.2: the scan
// stays within the bound's cells.
func FuzzLongMessage(f *testing.F) {
	for _, s := range []string{
		"", "a\nb\nc", "a\nb\nc\n", "\n\n\nabc", "a\n\n\n\n", "\t\t\t\t\t", "\x01\x02\x7f", "\xff\xfe", "\u0085\u0085",
		"日本語 日本語", "👍🏽👍🏽", "e\u0301\u0301", "\r\n\r\n", "word word word word word word", "    ", "a  b   c", "x \t y",
		"\u0600a", "\U0001F1E6\U0001F1E7\U0001F1E8", "क्ष", "\u200d", "a\u200d\nb",
	} {
		for _, w := range []uint8{0, 1, 2, 3, 4, 5, 8, 48, 72} {
			f.Add(s, w)
		}
	}
	// What found R-H, a false chip (doc/plan73-evaluation.md): characters joined by U+200D, one
	// cell wider than a row, kept for good: the fewest at zone B's widths and at the narrow ones.
	for _, s := range []struct {
		n int
		w uint8
	}{{106, 68}, {112, 72}, {76, 48}, {196, 128}, {8, 0}, {8, 1}, {8, 3}, {16, 6}, {20, 20}} {
		f.Add(joinedCell(s.n), s.w)
		f.Add(joinedCell(s.n)+" "+joinedCell(s.n)+"\n"+joinedCell(s.n), s.w)
		f.Add(string(rune(0x200B))+joinedCell(s.n)+string(rune(0x200B)), s.w)
	}
	f.Fuzz(func(t *testing.T, text string, wb uint8) {
		w := int(wb)%140 + 1
		want := rowsDrawn(text, w) > messageRows
		if got := LongMessage(text, w); got != want {
			t.Errorf("LongMessage(%q, %d) = %v, but %d rows are drawn", text, w, got, rowsDrawn(text, w))
		}
		if got, sum := cardRows(text, w), rowsDrawn(text, w); got != sum {
			t.Errorf("%q at %d: cardText draws %d rows, the sum says %d", text, w, got, sum)
		}
		if _, looked := messageScan(nil, text, w); looked > 4*messageRows*(w+1)+w {
			t.Errorf("messageScan(%q, %d) looked at %d cells, over its bound", text, w, looked)
		}
	})
}

// TestLongMessageBound is criterion 73.2: with most = 3 × (w + 1) and a cell wider than w
// counted as w, it reads at most four lines and, in each, cells summing to at most most
// and the one cell that passes that sum; with no allocation for a text its line breaks
// or a line's cells decide. It counts cells, not bytes. TestChipScale, in the timing
// step, records that its time does not follow the text's size.
func TestLongMessageBound(t *testing.T) {
	lineText := "some words on a line"
	lines := strings.Repeat(lineText+"\n", 50_000)
	shortLines := strings.Repeat("a\n", 50_000)
	emptyLines := strings.Repeat("\n", 50_000)
	one := strings.Repeat("0123456789", 300_000)
	fifth, fourth := "a\nb\nc\nd\n"+one, "a\nb\nc\n"+one
	tabs := strings.Repeat("\t", 3_000_000)
	kinds := []struct {
		name string
		text string
		wide int
	}{
		{"one cell each", one, 1},
		{"wide characters", strings.Repeat("日", 1_000_000), 2},
		{"emoji clusters", strings.Repeat("👍🏽", 400_000), 2},
		{"control characters", strings.Repeat("\x01", 3_000_000), 2},
		{"tabs", tabs, 4},
		{"C1 controls", strings.Repeat(string(rune(0x85)), 1_500_000), 4},
		{"bytes that are not UTF-8", strings.Repeat("\xff", 3_000_000), 4},
		{"joined clusters wider than any row", strings.Repeat(joinedCell(300), 1_700), 600},
	}
	for _, w := range boundWidths {
		most := 3 * (w + 1)
		for _, k := range kinds {
			// The first multiple of the cell, counted as at most w, past most: at most most + w.
			cell := min(k.wide, w)
			want := (most/cell + 1) * cell
			if long, looked := messageScan(nil, k.text, w); !long || looked != want || looked > most+w {
				t.Errorf("w=%d, one 3 MB line of %s: long %v, %d cells looked at, want %d", w, k.name, long, looked, want)
			}
		}
		// Each line's cells tell which lines were read.
		for _, c := range []struct {
			name   string
			text   string
			looked int
		}{
			{"50,000 lines of a cell", shortLines, 4},
			{"50,000 empty lines", emptyLines, 0},
			{"a fifth line, not read", fifth, 4},
			{"a fourth line, read to the bound", fourth, 3 + most + 1},
			{"one line of exactly the bound's cells, over 3 rows", strings.Repeat("x", most), most},
		} {
			if long, looked := messageScan(nil, c.text, w); !long || looked != c.looked {
				t.Errorf("w=%d, %s: long %v, %d cells looked at, want %d", w, c.name, long, looked, c.looked)
			}
		}
		// The bound can be no lower: exactly its cells on exactly 3 rows, each row full, or
		// one cell wider than itself, and its space dropped at the edge; and one cell more
		// is over them.
		shapes := []string{strings.Repeat("x", w) + " ", joinedCell(w/2+1) + " "}
		if w <= 4 {
			shapes = append(shapes, "\t ")
		}
		for _, shape := range shapes {
			three := strings.Repeat(shape, 3)
			if long, looked := messageScan(nil, three, w); long || looked != most || rowsDrawn(three, w) != 3 {
				t.Errorf("w=%d, %q: long %v, %d cells looked at, want %d, on %d rows", w, three, long, looked, most, rowsDrawn(three, w))
			}
			if long, looked := messageScan(nil, three+"y", w); !long || looked != most+1 || rowsDrawn(three+"y", w) != 4 {
				t.Errorf("w=%d, %q and a cell: long %v, %d cells looked at, want %d, on %d rows", w, three, long, looked, most+1, rowsDrawn(three+"y", w))
			}
		}
		for name, text := range map[string]string{"50,000 lines": shortLines, "one 3 MB line": one} {
			if n := testing.AllocsPerRun(20, func() { LongMessage(text, w) }); n != 0 {
				t.Errorf("w=%d, %s: %v allocations", w, name, n)
			}
		}
	}
	for _, w := range messageWidths {
		if long, looked := messageScan(nil, lines, w); !long || looked != 4*len(lineText) {
			t.Errorf("w=%d, 50,000 lines: long %v, %d cells looked at, want %d", w, long, looked, 4*len(lineText))
		}
		if n := testing.AllocsPerRun(20, func() { LongMessage(lines, w) }); n != 0 {
			t.Errorf("w=%d, 50,000 lines: %v allocations", w, n)
		}
	}
	// The passing cell at its widest: 24 cells, then a tab from 24 to 28.
	if _, looked := messageScan(nil, tabs, 7); looked != 24+4 {
		t.Errorf("tabs at 7 cells: %d cells looked at, want %d", looked, 24+4)
	}
	// A width of none reads as one cell does.
	if long, looked := messageScan(nil, one, 0); !long || looked != 3*(1+1)+1 {
		t.Errorf("no width, one 3 MB line: long %v, %d cells looked at, want %d", long, looked, 3*(1+1)+1)
	}
	// Cells, not bytes: what takes no cell is read to the line's end, whatever its size,
	// and a cluster of any length is its one cell (the evaluation measures them, R-I).
	zero := strings.Repeat(string(rune(0x200B)), 1_000_000)
	marks := "e" + strings.Repeat(string(rune(0x301)), 1_000_000)
	for _, w := range []int{1, 73} {
		most := 3 * (w + 1)
		for _, c := range []struct {
			name   string
			text   string
			long   bool
			looked int
		}{
			{"zero-width spaces", zero, false, 0},
			{"one cluster of a million marks", marks, false, 1},
			{"zero-width spaces, then cells", zero + one, true, most + 1},
		} {
			if long, looked := messageScan(nil, c.text, w); long != c.long || looked != c.looked {
				t.Errorf("w=%d, %s: long %v, %d cells looked at, want %v, %d", w, c.name, long, looked, c.long, c.looked)
			}
		}
	}
}

// TestLongMessageRenderer is criterion 73.1 against the renderer itself, on what its
// bound could get wrong, the false chip of R-H first (doc/plan73-evaluation.md): the
// rows cardText draws, the answer that follows from them, and what zone B then shows,
// the message's own rows for 3 or fewer, its chip for more. Rows -1 asks only that all
// of them agree.
func TestLongMessageRenderer(t *testing.T) {
	zero := string(rune(0x200B))
	zeros := strings.Repeat(zero, 1_000_000)
	marks := "e" + strings.Repeat(string(rune(0x301)), 1_000_000)
	mixed := "a 日本語\t\x01 " + string(rune(0x85)) + "\xff 👍🏽 e" + string(rune(0x301))
	huge := joinedCell(2_000) // wider than a row and than the bound, at every size
	for _, size := range goldenSizes {
		ui := UI{Width: size[0], Height: size[1], Focus: "B", Now: lookAt}
		w := MessageWidth(messageState("a"), ui)
		x := func(n int) string { return strings.Repeat("x", n) }
		fits := joinedCell(w / 2) // joined characters a row can hold
		for _, c := range []struct {
			name string
			text string
			rows int
		}{
			{"exactly 3 rows", x(3 * w), 3},
			{"exactly 4 rows", x(3*w + 1), 4},
			{"a long ASCII line", x(100 * w), 100},
			{"three lines", "a\nb\nc", 3},
			{"three lines and a final break", "a\nb\nc\n", 4},
			{"one extremely wide grapheme", huge, 1},
			{"the fewest joined characters past the bound", joinedCell(3*(w+1)/2 + 1), 1},
			{"a wide grapheme between two words", "a " + huge + " b", 3},
			{"three wide graphemes", huge + huge + huge, 3},
			{"three, spaces between them", huge + " " + huge + " " + huge, 3},
			{"three and a final space", huge + " " + huge + " " + huge + " ", 3},
			{"four wide graphemes", huge + huge + huge + huge, 4},
			{"four, spaces between them", huge + " " + huge + " " + huge + " " + huge, 4},
			{"three and one cell", huge + huge + huge + "x", 4},
			{"three on three lines", huge + "\n" + huge + "\n" + huge, 3},
			{"three on three lines and a final break", huge + "\n" + huge + "\n" + huge + "\n", 4},
			{"joined characters filling 3 rows", fits + fits + fits, 3},
			{"joined characters filling 4 rows", fits + fits + fits + fits, 4},
			{"a zero-width cell before a wide grapheme", zero + huge, 2},
			{"a zero-width cell after it", huge + zero, 2},
			{"zero-width cells around three", zero + huge + zero + huge, 4},
			{"a large zero-width line", zeros, 1},
			{"a large zero-width line and three more", zeros + "\na\nb\nc", 4},
			{"one cluster of a million marks", marks, 1},
			{"mixed Unicode, spaces, tabs and controls", mixed, 1},
			{"the same, wrapping", strings.Repeat(mixed, 5), -1},
			{"the same, further", strings.Repeat(mixed, 9), -1},
		} {
			rows := cardRows(c.text, w)
			if c.rows >= 0 && rows != c.rows {
				t.Errorf("%v, %s: cardText draws %d rows, want %d", size, c.name, rows, c.rows)
			}
			if sum := rowsDrawn(c.text, w); sum != rows {
				t.Errorf("%v, %s: cardText draws %d rows, the sum says %d", size, c.name, rows, sum)
			}
			if got := LongMessage(c.text, w); got != (rows > 3) {
				t.Errorf("%v, %s: LongMessage %v, but %d rows are drawn", size, c.name, got, rows)
			}
			if long, _ := messageScan(NewWrapCache(), c.text, w); long != (rows > 3) {
				t.Errorf("%v, %s: through the cache %v, but %d rows are drawn", size, c.name, long, rows)
			}
			if drawn := youRows(messageState(c.text), ui); rows > 3 {
				if len(drawn) != 2 || !strings.Contains(plainRow(drawn[1]), "▣ Long message") {
					t.Errorf("%v, %s: zone B has %d rows and no chip", size, c.name, len(drawn))
				}
			} else if len(drawn) != rows+1 || strings.Contains(plainRow(drawn[1]), "Long message") {
				t.Errorf("%v, %s: zone B has %d rows, want the YOU row and %d", size, c.name, len(drawn), rows)
			}
		}
	}
}

// TestLongMessageEquivalence is criterion 73.1 in every run, not only under -fuzz: random
// mixes of every kind of cell at any width, each against the rows cardText draws, and
// within the bound of 73.2.
func TestLongMessageEquivalence(t *testing.T) {
	u := func(r ...rune) string { return string(r) }
	kinds := []string{
		"x", "word ", " ", "  ", "\t", "\x01", "\x7f", "\r", "\xff", "\n", "\n", "日", "👍🏽", "e" + u(0x301),
		u(0x85), u(0x301), u(0x200B), u(0x200D), u(0xFEFF), u(0xAD), u(0x600), u(0x1100), u(0x1161), u(0x1F1E6),
		"a" + u(0x200D), u(0x915, 0x94D, 0x937), u(0x1F468, 0x200D, 0x1F469), joinedCell(2), joinedCell(3), joinedCell(40), joinedCell(200),
	}
	rng := rand.New(rand.NewPCG(73, 8))
	for range 20_000 {
		var b strings.Builder
		for range rng.IntN(30) {
			b.WriteString(kinds[rng.IntN(len(kinds))])
		}
		text, w := b.String(), rng.IntN(140)+1
		rows := cardRows(text, w)
		if sum := rowsDrawn(text, w); sum != rows {
			t.Fatalf("%q at %d: cardText draws %d rows, the sum says %d", text, w, rows, sum)
		}
		long, looked := messageScan(nil, text, w)
		if long != (rows > 3) {
			t.Fatalf("%q at %d: long %v, but %d rows are drawn", text, w, long, rows)
		}
		if cached, _ := messageScan(NewWrapCache(), text, w); cached != long {
			t.Fatalf("%q at %d: through the cache %v, without it %v", text, w, cached, long)
		}
		if most := 3 * (w + 1); looked > 4*most+w {
			t.Fatalf("%q at %d: %d cells looked at, over 4 x %d + %d", text, w, looked, most, w)
		}
		// A width of none wraps as one cell does (wrapCells), the narrowest MessageWidth gives.
		if got, want := LongMessage(text, 0), rowsDrawn(text, 0) > 3; got != want || want != (cardRows(text, 1) > 3) {
			t.Fatalf("%q at no width: long %v, want %v", text, got, want)
		}
	}
}

// linesOf is a text of n lines and exactly size bytes.
func linesOf(n, size int) string {
	line := strings.Repeat("x", (size-(n-1))/n)
	text := strings.TrimSuffix(strings.Repeat(line+"\n", n), "\n")
	return text + strings.Repeat("y", size-len(text))
}

// TestChipWords is criterion 73.3: the chip's exact words, at every width, and the
// full view's head giving the same count.
func TestChipWords(t *testing.T) {
	cjk := strings.TrimSuffix(strings.Repeat("日本語\n", 4), "\n")
	for _, c := range []struct {
		text, words, head string
	}{
		{linesOf(404, 18_000), "▣ Long message · 404 lines · 18 KB", "404 lines"},
		{strings.Repeat("x", 15_000), "▣ Long message · 1 line · 15 KB", "1 line"},
		{linesOf(1204, 40_000), "▣ Long message · 1,204 lines · 40 KB", "1,204 lines"},
		{"a\nb\nc\n", "▣ Long message · 4 lines · 6 B", "4 lines"},
		{cjk, "▣ Long message · 4 lines · 39 B", "4 lines"}, // UTF-8 bytes, not characters
		{linesOf(5, 1_200_000), "▣ Long message · 5 lines · 1.2 MB", "5 lines"},
	} {
		for _, w := range messageWidths {
			if got := strings.TrimRight(plainOf(chipRow(0, c.text, w, false)), " "); got != c.words {
				t.Errorf("w=%d: %q, want %q", w, got, c.words)
			}
		}
		s := state.New()
		s.Conversation = []state.Entry{{Role: "user", Text: c.text}}
		head := plainOf(entryViewer(s, UI{Width: 120, Height: 40, ViewEntry: 1, InspectHold: true}, 0, &Screen{})[1])
		if !strings.HasSuffix(strings.TrimRight(head, " "), " "+c.head) {
			t.Errorf("the full view's head %q, want it to end in %q", head, c.head)
		}
	}
}

// messageState is a conversation of one message, the turn it opened.
func messageState(text string, images ...event.Image) *state.State {
	b := newConvBench("claude-code", "a-model")
	b.ev(event.UserMessage, event.User{Text: text})
	b.s.Conversation[0].Images = images
	return b.s
}

// youRows is the message's rows in zone B: from YOU's down to the last non-empty one.
func youRows(s *state.State, ui UI) []line {
	rows := bRows(s, ui, layout(s, ui).bw-4)
	first := -1
	for i, r := range rows {
		if strings.Contains(plainRow(r), "YOU") {
			first = i
			break
		}
	}
	if first < 0 {
		return nil
	}
	last := first
	for i := first; i < len(rows); i++ {
		if plainRow(rows[i]) == "" || strings.TrimSpace(strings.TrimLeft(plainRow(rows[i]), "▎ ")) == "" {
			break
		}
		last = i
	}
	return rows[first : last+1]
}

// goldenSizes are the sizes of criterion 73.4, and zone B's narrowest form.
var goldenSizes = [][2]int{{80, 24}, {100, 30}, {120, 40}, {200, 60}, {208, 59}}

// TestZoneBForms is criterion 73.4: at each size 1, 2 and 3 rows are today's rows,
// each span the card's; 4 rows and 404 lines are the YOU row and one chip's row, no
// byte of the text in zone B, no box, no "more lines".
func TestZoneBForms(t *testing.T) {
	for _, size := range goldenSizes {
		ui := UI{Width: size[0], Height: size[1], Focus: "B", Now: lookAt}
		w := MessageWidth(messageState("a"), ui)
		x := func(n int) string { return strings.Repeat("x", n) }
		for _, c := range []struct {
			name string
			text string
			rows int
		}{
			{"one row", "one short message", 1},
			{"two lines", "first line\nsecond line", 2},
			{"one line on two rows", x(w + 1), 2},
			{"three lines", "first\nsecond\nthird", 3},
			{"one line filling three rows", x(3 * w), 3},
			{"two lines on three rows", x(w+5) + "\nend", 3},
			{"tabs and indentation kept", "\tfirst\n  second  \nthird", 3},
		} {
			s := messageState(c.text)
			got := youRows(s, ui)
			if len(got) != c.rows+1 {
				t.Errorf("%v, %s: %d rows, want the label and %d:\n%s", size, c.name, len(got), c.rows, texts(got))
				continue
			}
			// Today's rows: the lines' wraps, each in the message's style, as cardText
			// has always drawn them (cards.go), in the card's frame.
			fr := &frame{kind: cardYou, w: layout(s, ui).bw - 4}
			want := []line{fr.wrap(youLabel(s.Conversation[0]), false)}
			for j, l := range strings.Split(c.text, "\n") {
				for _, r := range wrapSource(l, w, w, ref{panel: 'B', item: 0, line: j}, stBright) {
					want = append(want, fr.wrap(r, false))
				}
			}
			if styled(got) != styled(want) {
				t.Errorf("%v, %s: rows differ from today's:\n%s\nwant:\n%s", size, c.name, styled(got), styled(want))
			}
			if frame := texts(Draw(s, ui).lines); strings.Contains(frame, "Long message") {
				t.Errorf("%v, %s: a chip for a short message:\n%s", size, c.name, frame)
			}
		}
		for _, c := range []struct {
			name, text, words string
		}{
			{"four lines", "qq-one\nqq-two\nqq-three\nqq-four", "▣ Long message · 4 lines · 30 B"},
			{"three lines and a break", "qq-one\nqq-two\nqq-three\n", "▣ Long message · 4 lines · 23 B"},
			{"one line one cell over", "qq" + x(3*w-1), fmt.Sprintf("▣ Long message · 1 line · %s", byteSize(int64(3*w+1)))},
			{"404 lines", "first\n" + strings.Repeat("qq-line\n", 402) + "last", "▣ Long message · 404 lines · 3 KB"},
		} {
			s := messageState(c.text)
			got := youRows(s, ui)
			if len(got) != 2 || !strings.Contains(plainRow(got[0]), "YOU") || !strings.Contains(plainRow(got[1]), c.words) {
				t.Errorf("%v, %s: rows %q, want YOU's and %q", size, c.name, rowTexts(got), c.words)
			}
			all := bRows(s, ui, layout(s, ui).bw-4)
			for _, r := range all {
				if p := plainRow(r); strings.Contains(p, "qq") || strings.Contains(p, "more lines") || strings.ContainsAny(p, "╭╰") {
					t.Errorf("%v, %s: row %q: text, a fold or a box in zone B", size, c.name, p)
				}
			}
			if sp, ok := spanOf(got[1], "▣ Long message"); !ok || sp.st != stMuted || sp.ref.kind != refFold || sp.ref.item != 0 {
				t.Errorf("%v, %s: the chip's span %+v", size, c.name, sp)
			}
		}
	}
}

// TestMessageWidthIsDrawn is design D1 and V6: the width from layout is what zone B
// wraps to, in each of its forms, a panel docked too: a message filling exactly three
// rows there is text, one cell more a chip.
func TestMessageWidthIsDrawn(t *testing.T) {
	for _, size := range goldenSizes {
		for _, docked := range []bool{false, true} {
			ui := UI{Width: size[0], Height: size[1], Focus: "B", Now: lookAt}
			if docked {
				ui.Explorer = ExplorerView{Open: true}
			}
			s := messageState("a")
			s.Screen = state.ScreenDashboard
			if layout(s, ui).bw <= 4 {
				continue // the panel in zone B's place: no zone B to wrap to
			}
			w := MessageWidth(s, ui)
			exact := messageState(strings.Repeat("x", 3*w))
			if f := texts(Draw(exact, ui).lines); strings.Contains(f, "Long message") {
				t.Errorf("%v docked %v, w=%d: three rows drawn as a chip:\n%s", size, docked, w, f)
			}
			over := messageState(strings.Repeat("x", 3*w+1))
			if f := texts(Draw(over, ui).lines); !strings.Contains(f, "Long message") {
				t.Errorf("%v docked %v, w=%d: one cell more is no chip:\n%s", size, docked, w, f)
			}
		}
	}
}

// TestChipWithImages is criterion 73.5: long text and images, the chip's row, then a
// row an image, as today; images and short text, today's rows.
func TestChipWithImages(t *testing.T) {
	imgs := []event.Image{{Name: "one.png", Size: 1500}, {Name: "two.png"}}
	ui := UI{Width: 120, Height: 40, Focus: "B", Now: lookAt}
	got := rowTexts(youRows(messageState("a\nb\nc\nd", imgs...), ui))
	if len(got) != 4 || !strings.Contains(got[1], "▣ Long message · 4 lines · 7 B") || !strings.Contains(got[2], "one.png · 2 KB") || !strings.Contains(got[3], "two.png") {
		t.Errorf("a chip and two images: %q", got)
	}
	got = rowTexts(youRows(messageState("short", imgs...), ui))
	if len(got) != 4 || !strings.HasSuffix(got[1], "short") || !strings.Contains(got[2], "one.png") || !strings.Contains(got[3], "two.png") {
		t.Errorf("short text and two images: %q", got)
	}
}

// TestChipRowFocused is criterion 73.21's narrow form and design D4: under the
// cursor the chip's row has ▸, its words in the attention colour and the hint whole
// on its right; its size goes first when the row has no room, then spread's cut; one
// row, never wider than the content, at any width.
func TestChipRowFocused(t *testing.T) {
	text := linesOf(404, 18_000)
	for _, w := range []int{129, 73, 69, 49, 30, 10} {
		l := chipRow(3, text, w, true)
		p := plainOf(l)
		if l.width() > w {
			t.Errorf("w=%d: %d cells: %q", w, l.width(), p)
		}
		if w >= 49 && !strings.HasSuffix(p, entryHint) {
			t.Errorf("w=%d: the hint not whole: %q", w, p)
		}
		if w >= 49 && !strings.Contains(p, "Long") {
			t.Errorf("w=%d: no chip's word: %q", w, p)
		}
		if w >= 49 && (l[0].ref.kind != refFold || l[0].ref.item != 3 || l[0].st != stAttentionBold) {
			t.Errorf("w=%d: first span %+v", w, l[0])
		}
	}
	if p := plainOf(chipRow(0, text, 129, true)); !strings.Contains(p, "· 18 KB") {
		t.Errorf("room for the size, and no size: %q", p)
	}
	if p := plainOf(chipRow(0, text, 69, true)); strings.Contains(p, "KB") || !strings.Contains(p, "404 lines") {
		t.Errorf("no room for the size: %q", p)
	}
}

// chipEditorView is a composer's view: typed text before and after a chip of 200
// lines and 5 KB, the cursor on col of its line.
func chipEditorView(col int) EditorView {
	return EditorView{Lines: []string{"before \x00 after"}, Total: 1, Hidden: 199, Line: 0, Col: col,
		Chips: []EditorChip{{Line: 0, Off: 7, Lines: 200, Bytes: 5000}}}
}

// TestEditorChipCell is design D11's drawing: a chip takes its one cell of the compact
// form, drawn as its words in the muted colour, one atom for the mouse; the cursor on
// it and a selection over it mark it whole; no other row changes.
func TestEditorChipCell(t *testing.T) {
	s := state.New()
	draw := func(v EditorView) (string, line) {
		ls := draftRows(s, UI{Editor: v, Focus: "B"}, 100, 1, 8)
		return strings.TrimSuffix(texts(ls), "\n"), ls[0]
	}
	text, row := draw(chipEditorView(0))
	if want := "> before ▣ Long message · 200 lines · 5 KB after"; text != want {
		t.Errorf("%q, want %q", text, want)
	}
	chip, ok := spanOf(row, "▣ Long message")
	if !ok || chip.st != stMuted || chip.ref.kind != refAtom || chip.ref.off != 7 || chip.ref.n != 1 {
		t.Errorf("the chip's span %+v", chip)
	}
	_, row = draw(chipEditorView(7))
	if chip, _ := spanOf(row, "▣ Long message"); chip.st != stCursor {
		t.Errorf("the cursor on the chip: %+v", chip)
	}
	v := chipEditorView(0)
	v.From, v.To = [2]int{0, 5}, [2]int{0, 9}
	_, row = draw(v)
	if chip, _ := spanOf(row, "▣ Long message"); chip.st != stSelected {
		t.Errorf("the selection over the chip: %+v", chip)
	}
	// A chip that does not fit the rest of its row takes the next.
	narrow := EditorView{Lines: []string{strings.Repeat("x", 90) + "\x00"}, Total: 1, Line: 0, Col: 91,
		Chips: []EditorChip{{Line: 0, Off: 90, Lines: 5, Bytes: 100}}}
	if got := texts(draftRows(s, UI{Editor: narrow, Focus: "B"}, 100, 1, 8)); !strings.Contains(got, "\n  ▣ Long message · 5 lines · 100 B") {
		t.Errorf("a chip at a row's end:\n%s", got)
	}
}

// TestEditorLineCount is design D11's count: the message's lines, those chips hide
// among them, shown while the composer draws more than one line; a chip alone shows
// only its own (criterion 73.13).
func TestEditorLineCount(t *testing.T) {
	if got := lineCount(EditorView{Total: 2, Hidden: 199}); got != "201 lines" {
		t.Errorf("a chip of 200 lines and a typed line: %q", got)
	}
	if got := lineCount(EditorView{Total: 12}); got != "12 lines" {
		t.Errorf("a typed draft: %q", got)
	}
	s := state.New()
	s.Screen = state.ScreenDashboard
	alone := EditorView{Lines: []string{"\x00"}, Total: 1, Hidden: 199, Chips: []EditorChip{{Off: 0, Lines: 200, Bytes: 5000}}}
	if got := texts(editorLines(s, UI{Editor: alone, Focus: "B"}, 100, 1, 8)); strings.Contains(got, "lines ") && strings.Contains(got, "200 lines ·") == false || strings.Contains(got, "200 lines  ") {
		t.Errorf("a chip alone:\n%s", got)
	}
	two := EditorView{Lines: []string{"\x00", "typed"}, Total: 2, Hidden: 199, Line: 1, Chips: []EditorChip{{Off: 0, Lines: 200, Bytes: 5000}}}
	if got := texts(editorLines(s, UI{Editor: two, Focus: "B"}, 100, 1, 8)); !strings.Contains(got, "201 lines") {
		t.Errorf("a chip and a typed line:\n%s", got)
	}
}

// TestMessageViewMemo is plan 73's P4: a memo, never a source of what is drawn: the full
// view with it is the frame without it, across widths, scrolling and another message;
// the rows are wrapped once for a width and dropped once the view closes.
func TestMessageViewMemo(t *testing.T) {
	// A first line that wraps, so that a width changes the rows.
	wide := strings.Repeat("word ", 90)
	s := messageState(wide + "\n" + longLines(60, "first message"))
	s.Conversation = append(s.Conversation, state.Entry{Role: "user", Text: wide + "\n" + longLines(50, "the other one"), Seq: 99})
	memo := NewMemo()
	for _, c := range []struct {
		w, h, n, top int
	}{
		{120, 40, 1, 0}, {120, 40, 1, 5}, {120, 40, 1, 30}, {80, 24, 1, 0}, {120, 40, 1, 0}, {120, 40, 2, 3}, {200, 60, 2, 0}, {120, 40, 1, 2},
	} {
		ui := UI{Width: c.w, Height: c.h, ViewEntry: c.n, InspectHold: true, InspectTop: c.top, Now: lookAt}
		with := ui
		with.Memo = memo
		if got, want := Draw(s, with).String(), Draw(s, ui).String(); got != want {
			t.Errorf("%+v: the frame with the memo differs:\n%s\nwant:\n%s", c, got, want)
		}
	}
	ui := UI{Width: 120, Height: 40, ViewEntry: 1, InspectHold: true, Now: lookAt, Memo: memo}
	Draw(s, ui)
	rows := memo.view.rows
	Draw(s, ui)
	if &memo.view.rows[0] != &rows[0] {
		t.Error("the rows were wrapped again for the same message and width")
	}
	// Another text of the same length at the same width: never the old rows.
	s.Conversation[0].Text = strings.Replace(s.Conversation[0].Text, "first", "FIRST", 1)
	if f := Draw(s, ui).String(); !strings.Contains(f, "FIRST") {
		t.Errorf("a changed text drew the old rows:\n%s", f)
	}
	// The view closed: dropped at the second frame.
	ui.ViewEntry = 0
	Draw(s, ui)
	Draw(s, ui)
	if memo.view.rows != nil || memo.view.text != "" {
		t.Errorf("the view closed, the memo holds %d rows", len(memo.view.rows))
	}
}

// longLines is n lines, each naming its number after word.
func longLines(n int, word string) string {
	ls := make([]string, n)
	for i := range ls {
		ls[i] = fmt.Sprintf("%s %d", word, i+1)
	}
	return strings.Join(ls, "\n")
}

// TestChipMatchRow is design D5's search: a match behind a chip is on the chip's own row,
// not its label's above it, whichever line of the message holds it, and the row's place
// stands for all the message's lines; the chip is lit in the match's style, the current
// match's in its own (criterion 73.9).
func TestChipMatchRow(t *testing.T) {
	c := newSearchConv()
	c.s.Session.Phase = state.Ready
	c.add(state.Entry{Role: "assistant", Text: "an answer before it"})
	item := c.add(state.Entry{Role: "user", Text: "first\nsecond\nthe needle\nfourth\nthe last needle"})
	c.add(state.Entry{Role: "assistant", Text: "an answer with a needle after it"})
	ui := UI{Width: 120, Height: 40, Focus: "B", Now: lookAt}
	w := layout(c.s, ui).bw - 4
	rows := bRows(c.s, ui, w)
	ms := c.found("needle")
	var behind []ConvMatch
	for _, m := range ms {
		if m.Item == item {
			behind = append(behind, m)
		}
	}
	if len(ms) != 3 || len(behind) != 2 {
		t.Fatalf("matches %+v", ms)
	}
	for _, m := range behind {
		row, total, ok := ConvMatchRow(c.s, ui, w, 30, m)
		if !ok || total != len(rows) || !strings.Contains(plainRow(rows[row]), "▣ Long message · 5 lines") {
			t.Errorf("match %+v: row %d of %d (%v): %q", m, row, total, ok, plainRow(rows[min(row, len(rows)-1)]))
		}
	}
	view, _, _, _ := conversation(c.s, ui, w, 1<<20)
	found := false
	for _, r := range view {
		if !r.more {
			continue
		}
		found = true
		if p, ok := rowEnd(r, "user"); !ok || p.Line != AllLines || p.Item != item {
			t.Errorf("the chip's row's place %+v (%v), want all of entry %d", p, ok, item)
		}
	}
	if !found {
		t.Fatal("no row stands for the message's lines")
	}
	// Lit: the current match's style on the chip while one behind it is current, the
	// match's style once another is.
	lit := func(cur int) (match, current []string) {
		sc := Draw(c.s, UI{Width: 120, Height: 40, Focus: "B", Now: lookAt, Search: SearchView{Open: true, Query: "needle", Matches: ms, Cur: cur}})
		return litSpans(sc, stMatch), litSpans(sc, stMatchCur)
	}
	onChip := func(spans []string) bool {
		for _, s := range spans {
			if strings.Contains(s, "▣ Long message") {
				return true
			}
		}
		return false
	}
	for i, m := range ms {
		match, current := lit(i)
		if m.Item == item && !onChip(current) {
			t.Errorf("current match %d, behind the chip: not lit as current: %q", i, current)
		}
		if m.Item != item && (onChip(current) || !onChip(match)) {
			t.Errorf("current match %d, elsewhere: the chip lit %q as a match, %q as current", i, match, current)
		}
	}
}

// TestChipSelectionLit is the reviewer's point on zone B's selection (doc/plan73-evaluation.md,
// The correction of F1): a chip's words are lit exactly when the selection copies some of
// the message, whatever its ends, the copy itself unchanged; never the hint beside them; the
// selection's style wins over the search's; another panel's selection leaves them.
func TestChipSelectionLit(t *testing.T) {
	const before, after = "the answer before", "the answer after"
	const message = "first line\n\tsecond  \n\nfourth\nfifth"
	b := newConvBench("claude-code", "a-model")
	b.msg("m0", before)
	b.ev(event.UserMessage, event.User{Text: message})
	b.msg("m1", after)
	s := b.s
	if len(s.Conversation) != 3 || s.Conversation[1].Role != "user" {
		t.Fatalf("the conversation: %+v", s.Conversation)
	}
	base := UI{Width: 120, Height: 40, Focus: "B", Now: lookAt}
	chip := func(sc *Screen) (line, span) {
		t.Helper()
		for _, l := range sc.lines {
			if x, ok := spanOf(l, "▣ Long message"); ok {
				return l, x
			}
		}
		t.Fatalf("no chip:\n%s", sc.String())
		return nil, span{}
	}
	plain, none := chip(Draw(s, base))
	if none.st != stMuted {
		t.Fatalf("no selection: the chip's style %p, want stMuted", none.st)
	}
	holds := func(got string) bool {
		for _, word := range []string{"first", "second", "fourth", "fifth"} {
			if strings.Contains(got, word) {
				return true
			}
		}
		return false
	}
	for _, c := range []struct {
		name     string
		from, to Pos
		lit      bool
		copied   string
	}{
		{"across it", Pos{'B', 0, 0, 0}, Pos{'B', 2, 0, len(after)}, true, before + "\n\n" + message + "\n\n" + after},
		{"from the end of one answer to the start of the other", Pos{'B', 0, 0, len(before)}, Pos{'B', 2, 0, 0}, true, "\n\n" + message + "\n\n"},
		{"inside the answer before", Pos{'B', 0, 0, 0}, Pos{'B', 0, 0, len(before)}, false, before},
		{"up to its start", Pos{'B', 0, 0, 0}, Pos{'B', 1, 0, 0}, false, before + "\n\n"},
		{"inside the answer after", Pos{'B', 2, 0, 0}, Pos{'B', 2, 0, 5}, false, after[:5]},
		{"its last lines and on", Pos{'B', 1, 3, 0}, Pos{'B', 2, 0, 3}, true, "fourth\nfifth\n\n" + after[:3]},
		{"its first line", Pos{'B', 1, 0, 0}, Pos{'B', 1, 0, 5}, true, "first"},
	} {
		sel := Selection{From: c.from, To: c.to}
		ui := base
		ui.Sel = sel
		sc := Draw(s, ui)
		l, x := chip(sc)
		if (x.st == stSelected) != c.lit || !c.lit && x.st != stMuted {
			t.Errorf("%s: the chip's style %p, lit %v", c.name, x.st, c.lit)
		}
		if plainOf(l) != plainOf(plain) {
			t.Errorf("%s: the row's words changed: %q, %q", c.name, plainOf(l), plainOf(plain))
		}
		got := sc.Text(s, sel)
		if got != c.copied {
			t.Errorf("%s: copied %q, want %q", c.name, got, c.copied)
		}
		if holds(got) != c.lit {
			t.Errorf("%s: lit %v, but %q copied", c.name, c.lit, got)
		}
	}
	all := Selection{From: Pos{'B', 0, 0, 0}, To: Pos{'B', 2, 0, len(after)}}

	// Under the entry cursor: its words lit, the hint as it is.
	ui := base
	ui.Entry = 2
	l, x := chip(Draw(s, ui))
	if x.st != stAttentionBold {
		t.Fatalf("under the cursor: the chip's style %p", x.st)
	}
	ui.Sel = all
	l, x = chip(Draw(s, ui))
	if hint, ok := spanOf(l, entryHint); x.st != stSelected || !ok || hint.st != stAttention {
		t.Errorf("under the cursor, selected: the chip %p, the hint %p (%v)", x.st, hint.st, ok)
	}

	// Over the search's light.
	ms, _ := ConvMatches(s, UI{}, "fourth", nil)
	ui = base
	ui.Search = SearchView{Open: true, Query: "fourth", Matches: ms, Cur: 0}
	if _, x = chip(Draw(s, ui)); x.st != stMatchCur {
		t.Fatalf("the search's current match: the chip's style %p", x.st)
	}
	ui.Sel = all
	if _, x = chip(Draw(s, ui)); x.st != stSelected {
		t.Errorf("selected over the search: the chip's style %p", x.st)
	}

	// Another panel's selection.
	ui = base
	ui.Sel = Selection{From: Pos{'I', 0, 0, 0}, To: Pos{'I', 5, 0, 0}}
	if _, x = chip(Draw(s, ui)); x.st != stMuted {
		t.Errorf("another panel's selection: the chip's style %p", x.st)
	}
}
