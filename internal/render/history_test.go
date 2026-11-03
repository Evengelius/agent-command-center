package render

import (
	"fmt"
	"image"
	"image/color"
	"strings"
	"testing"
	"unicode"

	"charm.land/lipgloss/v2"

	"github.com/Evengelius/agent-command-center/internal/state"
)

// Plan 75, design D1a: the title, grapheme clusters counted, none cut.
func TestSessionTitle(t *testing.T) {
	repeat := func(s string, n int) string { return strings.Repeat(s, n) }
	family := "👨\u200d👩\u200d👧\u200d👦" // one cluster of seven code points
	for _, c := range []struct{ name, in, want string }{
		{"plain", "remember the code word", "remember the code word"},
		{"whitespace runs", "  fix \t the \n\n bug   now  ", "fix the bug now"},
		{"CRLF and tabs", "a\r\nb\tc", "a b c"},
		{"control characters", "a\x00b\x07c\x1bd\x7fe", "abcde"},
		{"an escape sequence", "\x1b[31mred\x1b[0m \u009b2Jtext", "[31mred[0m 2Jtext"},
		{"controls between spaces", "a\x01 \x02 b", "a b"},
		{"unicode spaces", "a\u00a0\u2003b\u2028c", "a b c"},
		{"empty", "", ""},
		{"only white space and controls", " \t\n\x1b ", ""},
		{"sixty", repeat("x", 60), repeat("x", 60)},
		{"sixty-one", repeat("x", 61), repeat("x", 60)},
		{"sixty clusters of combining marks", repeat("e\u0301", 61), repeat("e\u0301", 60)},
		{"emoji clusters", repeat(family, 61), repeat(family, 60)},
		{"flags", repeat("🇫🇷", 61), repeat("🇫🇷", 60)},
		{"ends trimmed after the cut", repeat("x", 59) + " yyy", repeat("x", 59)},
		{"wide characters count as one", repeat("日", 61), repeat("日", 60)},
		{"bidi controls", "a\u202ab\u202ec\u202bd\u202ce\u202d", "abcde"},
		{"bidi isolates", "\u2066x\u2067y\u2068z\u2069", "xyz"},
		{"a zero-width space", "a\u200bb", "ab"},
		{"a byte order mark", "\ufeffhello", "hello"},
		{"the joiners stay", "a\u200db\u200cc", "a\u200db\u200cc"},
	} {
		if got := SessionTitle(c.in, false); got != c.want {
			t.Errorf("%s: %q, want %q", c.name, got, c.want)
		}
	}
	// A message cut short: its last cluster may lack what follows it, and is
	// left out; a long one still gives its 60 clusters, none cut.
	for _, c := range []struct{ name, in, want string }{
		{"a cut inside a cluster", "ab" + "e", "ab"},
		{"a short start", "hello world", "hello worl"},
		{"a cut after white space: the cluster before it may still grow", "ab ", "a"},
		{"a long start", repeat("x", 100), repeat("x", 60)},
		{"a long start of marks", repeat("é", 100), repeat("é", 60)},
	} {
		if got := SessionTitle(c.in, true); got != c.want {
			t.Errorf("partial, %s: %q, want %q", c.name, got, c.want)
		}
	}
	// A very long first message gives its first words.
	if got := SessionTitle(repeat("word ", 200000), false); got != strings.TrimSpace(repeat("word ", 12)) {
		t.Errorf("a long message: %q", got)
	}
}

func historyUI(w, h, n int) (*state.State, UI) {
	s, ui := homeTestUI()
	s.Screen = state.ScreenStartup
	ui.Width, ui.Height, ui.Sixel = w, h, false
	ui.History = HistoryView{Open: true, Project: `C:\work\demo-project`}
	for i := range n {
		r := HistoryRow{CLI: "Claude Code", Meta: fmt.Sprintf("2026-11-03 09:%02d · %dm ago · Claude Code · Claude Haiku 5.5 · 21f3b77%d", 59-i, i, i%10)}
		if i%3 != 2 {
			r.Title = fmt.Sprintf("Title of session %d", i)
		}
		ui.History.Rows = append(ui.History.Rows, r)
	}
	return s, ui
}

// Plan 75, design D2: the box, its title, the rows, in every size, with
// entries and with none; nothing but the box and the header on the screen.
func TestHistoryScreen(t *testing.T) {
	for _, sz := range [][2]int{{80, 24}, {120, 40}, {200, 60}} {
		w, h := sz[0], sz[1]
		s, ui := historyUI(w, h, 6)
		sc := Draw(s, ui)
		text := strings.Split(strings.TrimSuffix(rowsText(sc.lines), "\n"), "\n")
		if len(text) != h {
			t.Fatalf("%dx%d: %d rows", w, h, len(text))
		}
		all := strings.Join(text, "\n")
		for _, want := range []string{"AGENT COMMAND CENTER", "› SESSION HISTORY", "Project  C:\\work\\demo-project", "Title of session 0", "2026-11-03 09:59 · 0m ago · Claude Code · Claude Haiku 5.5 · 21f3b770", "Esc close", "▏"} {
			if !strings.Contains(all, want) {
				t.Errorf("%dx%d lacks %q:\n%s", w, h, want, all)
			}
		}
		if strings.Contains(all, "NEW SESSION") || strings.Contains(all, "PROJECTS") || strings.Contains(all, "Shortcuts") {
			t.Errorf("%dx%d: the home screen shows through:\n%s", w, h, all)
		}
		b := sc.History
		if b.W != min(w-2, 100) || b.X != (w-b.W)/2 || b.Y < 1 || b.Y+b.H > h-1 {
			t.Errorf("%dx%d: box %+v", w, h, b)
		}
		// Entry 2 has no title: the grey line alone.
		if strings.Contains(all, "Title of session 2") {
			t.Errorf("%dx%d: a title for the untitled entry", w, h)
		}
		for y, row := range text {
			if textWidth(row) != w {
				t.Errorf("%dx%d: row %d is %d cells", w, h, y, textWidth(row))
			}
		}
		// The selected entry's rows carry the ends ▏ ▕; the others do not.
		sel := 0
		for _, row := range text {
			if strings.Contains(row, "▏") {
				sel++
			}
		}
		if sel != 2 {
			t.Errorf("%dx%d: %d rows selected, want the 2 of entry 0", w, h, sel)
		}
	}

	s, ui := historyUI(80, 24, 0)
	text := rowsText(Draw(s, ui).lines)
	if !strings.Contains(text, "No sessions recorded for this project yet") || strings.Contains(text, "▏") {
		t.Errorf("none:\n%s", text)
	}
}

// Plan 75, design D2: the list follows the selection, a slim mark showing
// where it is; every entry reachable, the selected one always drawn.
func TestHistoryScroll(t *testing.T) {
	s, ui := historyUI(80, 24, 20)
	for sel := range 20 {
		ui.History.Sel = sel
		sc := Draw(s, ui)
		text := rowsText(sc.lines)
		if !strings.Contains(text, fmt.Sprintf("2026-11-03 09:%02d", 59-sel)) {
			t.Fatalf("selected %d not drawn:\n%s", sel, text)
		}
		if sel > 0 && sel < 19 && !strings.Contains(text, "▐") {
			t.Errorf("selected %d: no scroll mark", sel)
		}
		if n := strings.Count(text, "▏"); n < 1 || n > 2 {
			t.Errorf("selected %d: %d marked rows", sel, n)
		}
	}
	ui.History.Sel = 19
	last := rowsText(Draw(s, ui).lines)
	ui.History.Sel = 0
	first := rowsText(Draw(s, ui).lines)
	if first == last || !strings.Contains(last, "09:40") || strings.Contains(last, "09:59") || !strings.Contains(first, "09:59") {
		t.Errorf("first:\n%s\nlast:\n%s", first, last)
	}
	// The mark is below the top at the start and at the bottom at the end.
	mark := func(text string) (int, int) {
		a, b := -1, -1
		for y, row := range strings.Split(text, "\n") {
			if strings.Contains(row, "▐") {
				if a < 0 {
					a = y
				}
				b = y
			}
		}
		return a, b
	}
	a0, _ := mark(first)
	_, b1 := mark(last)
	_, bf := mark(first)
	if a0 < 0 || b1 < 0 || bf >= b1 {
		t.Errorf("the mark does not move: first at %d..%d, last ends at %d", a0, bf, b1)
	}
}

// Plan 75, design D4: the question in the box, in amber, naming ACC's list
// and the CLIs' own records, with Cancel selected when it opens.
func TestHistoryQuestion(t *testing.T) {
	s, ui := historyUI(80, 24, 6)
	ui.History.Ask, ui.History.AskCount, ui.History.AskCLIs = HistoryAskOne, 1, []string{"Claude Code"}
	text := rowsText(Draw(s, ui).lines)
	for _, want := range []string{"DELETE THIS SESSION FROM THE HISTORY?", "ACC's list only: Claude Code keeps its own record", "Delete", "Cancel"} {
		if !strings.Contains(text, want) {
			t.Errorf("one: lacks %q:\n%s", want, text)
		}
	}
	// The keys row is the question's while it is open, the list's otherwise.
	if !strings.Contains(text, "←→ choose · Enter answer · Esc cancel") || strings.Contains(text, "Shift+Del") {
		t.Errorf("the keys row with the question open:\n%s", text)
	}
	ui.History.AskCLIs = []string{"OpenAI Codex"}
	if t2 := rowsText(Draw(s, ui).lines); !strings.Contains(t2, "ACC's list only: OpenAI Codex keeps its own record") {
		t.Errorf("a Codex entry:\n%s", t2)
	}
	ui.History.Ask, ui.History.AskCount, ui.History.AskCLIs = HistoryAskAll, 6, []string{"Claude Code", "OpenAI Codex", "Claude Code"}
	t3 := rowsText(Draw(s, ui).lines)
	for _, want := range []string{"DELETE ALL 6 SESSIONS OF demo-project?", "ACC's list only: Claude Code and OpenAI Codex keep their own records"} {
		if !strings.Contains(t3, want) {
			t.Errorf("all: lacks %q:\n%s", want, t3)
		}
	}
	// Both titles in amber, the palette's In progress (#FBBF24), bold, on the
	// page's ground.
	for _, c := range []struct {
		ask   byte
		title string
	}{{HistoryAskOne, "DELETE THIS SESSION FROM THE HISTORY?"}, {HistoryAskAll, "DELETE ALL 6 SESSIONS OF demo-project?"}} {
		ui.History.Ask = c.ask
		var got []*lipgloss.Style
		for _, l := range Draw(s, ui).lines {
			for _, sp := range l {
				if sp.text == c.title {
					got = append(got, sp.st)
				}
			}
		}
		if len(got) != 1 || got[0] != pageGround.on(stProgressBold) {
			t.Errorf("%q: drawn %d times, style %v; want the amber of In progress, bold", c.title, len(got), got)
		}
	}
	ui.History.Ask = HistoryAskAll
	// The selected answer is on the button's fill, Cancel at first.
	sc := Draw(s, ui)
	x, y, ok := sc.findItem(HistoryPanel, HistoryItemCancel)
	if !ok {
		t.Fatal("no Cancel button")
	}
	if x2, y2, ok := sc.findItem(HistoryPanel, HistoryItemDelete); !ok || y2 != y || x2 >= x {
		t.Errorf("Delete at %d,%d, Cancel at %d,%d", x2, y2, x, y)
	}
	if _, item := sc.ItemAt(x+3, y); item != HistoryItemCancel {
		t.Errorf("the cell on Cancel: item %d", item)
	}
	// Entries show their index as the item of their rows.
	if _, item := sc.ItemAt(sc.History.X+5, sc.History.Y+6); item < HistoryItemRow {
		t.Errorf("an entry's row: item %d", item)
	}
}

// Plan 75, design D6: grid mode's footer, the full row from 100 columns, the
// short row under it; the picture's fourth group after the third.
func TestHomeFooter(t *testing.T) {
	text := func(w int) string { return rowsText([]line{homeGridFooter(UI{}, w)}) }
	for _, c := range []struct {
		w    int
		want string
	}{
		{80, " ›  Ctrl+N  New │  Ctrl+O  Open │  Alt+H  History │  Ctrl+Q  Quit"},
		{99, " ›  Ctrl+N  New │  Ctrl+O  Open │  Alt+H  History │  Ctrl+Q  Quit"},
		{100, " › Shortcuts:   Ctrl+N  New Session  │   Ctrl+O  Open Project  │   Alt+H  History  │   Ctrl+Q  Quit"},
		{200, " › Shortcuts:   Ctrl+N  New Session  │   Ctrl+O  Open Project  │   Alt+H  History  │   Ctrl+Q  Quit"},
	} {
		got := strings.TrimRight(text(c.w), "\n")
		if strings.TrimRight(got, " ") != c.want || textWidth(got) != c.w {
			t.Errorf("%d columns: %q (%d cells), want %q", c.w, got, textWidth(got), c.want)
		}
	}
	if n := textWidth(strings.TrimRight(text(99), " \n")); n != 65 {
		t.Errorf("the short row is %d cells, want 65", n)
	}
	if n := textWidth(strings.TrimRight(text(100), " \n")); n != 99 {
		t.Errorf("the full row is %d cells, want 99 (in 100 columns)", n)
	}

	// Picture mode: the first three groups where they were, the fourth after
	// them at the design's spacing, none past the screen.
	s, ui := homeTestUI()
	g := HomeLayout(208, 59, true)
	l := homeLook(s, ui, g)
	if len(l.Keys) != 4 || l.Keys[2].Key != "Alt+H" || l.Keys[3].Key != "Ctrl+Q" {
		t.Fatalf("the picture's keys: %+v", l.Keys)
	}
	img := image.NewNRGBA(g.Pixels())
	HomePaint(img, l, &boxGlyphs{}, img.Rect)
	at := func(x, y float64) color.NRGBA { return img.NRGBAAt(int(g.X(x)), int(g.Y(y))) }
	for i, x := range footCaps {
		// Inside the cap, right of its key: the cap's fill.
		if c := at(x+62, 738); c != hexColour(cHomeCapFill.tc) {
			t.Errorf("group %d: %v at the cap's end, want its fill %v", i, c, hexColour(cHomeCapFill.tc))
		}
	}
	for i, x := range footBars {
		// The bar is a pixel or two wide, anti-aliased: one of the columns
		// about it is not the page's ground.
		found := false
		for d := -1.0; d <= 1; d++ {
			found = found || at(x+d/g.K, 739) != hexColour(cHomePage.tc)
		}
		if !found {
			t.Errorf("bar %d: none at %.1f", i, x)
		}
	}
	if footCaps[0] != 141 || footCaps[1] != 357.3 || footCaps[2] != 582.5 {
		t.Errorf("the first three groups moved: %v", footCaps)
	}
	if last := footActions[3] + 4*7.78; last >= designW {
		t.Errorf("the fourth group ends at %.1f, past the screen", last)
	}
}

// Plan 75, design D2: the list keeps its place and scrolls only when the
// selection would leave the rows shown, so that a click never moves it.
func TestHistoryTop(t *testing.T) {
	s, ui := historyUI(80, 24, 20)
	_ = s
	v := ui.History
	_, _, _, room := historyGeom(80, 24, v)
	top := 0
	moved := false
	for sel := range 20 {
		v.Sel, v.Top = sel, top
		next := HistoryTop(80, 24, v)
		if next != top {
			// Only when the selection was past the last row shown.
			used := 0
			for k := next; k <= sel; k++ {
				used += 1
				if v.Rows[k].Title != "" {
					used++
				}
			}
			if sel < top || used > room+2 || next < top {
				t.Fatalf("selection %d: top %d to %d", sel, top, next)
			}
			moved = true
		}
		top = next
		// The selection is among the rows shown.
		rows := 0
		for k := top; k <= sel; k++ {
			rows++
			if v.Rows[k].Title != "" {
				rows++
			}
		}
		if rows > room {
			t.Fatalf("selection %d with top %d: %d rows, room %d", sel, top, rows, room)
		}
	}
	if !moved {
		t.Fatal("the list never scrolled")
	}
	// Going back up leaves the top where it is until the selection passes it.
	atEnd := top
	for sel := 19; sel >= atEnd; sel-- {
		v.Sel, v.Top = sel, top
		if got := HistoryTop(80, 24, v); got != atEnd {
			t.Fatalf("selection %d moved the top from %d to %d", sel, atEnd, got)
		}
	}
	v.Sel, v.Top = atEnd-1, top
	if got := HistoryTop(80, 24, v); got != atEnd-1 {
		t.Errorf("the selection above the top: top %d, want %d", got, atEnd-1)
	}
	// A click on an entry in view (the selection moved to it) moves nothing.
	v.Sel, v.Top = 3, 0
	if got := HistoryTop(80, 24, v); got != 0 {
		t.Errorf("a selection in view moved the top to %d", got)
	}
}

// cleanedTitleText is the text SessionTitle works on, made the plain way as a
// reference: white space runs one space, no control or format characters but
// the joiners and the tags, the ends trimmed.
func cleanedTitleText(text string) string {
	m := strings.Map(func(r rune) rune {
		switch {
		case unicode.IsSpace(r):
			return ' '
		case unicode.IsControl(r), unicode.Is(unicode.Cf, r) && r != 0x200C && r != 0x200D && (r < 0xE0020 || r > 0xE007F):
			return -1
		}
		return r
	}, text)
	return strings.Join(strings.Fields(m), " ")
}

// clusterEnds are the ends of the grapheme clusters of s, by the cluster
// function the width of a line is judged by.
func clusterEnds(s string) []int {
	var out []int
	for i := 0; i < len(s); {
		j, _ := next(s, i)
		out = append(out, j)
		i = j
	}
	return out
}

// checkTitle judges SessionTitle's answer for text against design D1a.
func checkTitle(t *testing.T, text string, partial bool) {
	t.Helper()
	got := SessionTitle(text, partial)
	cleaned := cleanedTitleText(text)
	ends := clusterEnds(cleaned)
	if n := len(clusterEnds(got)); n > TitleChars {
		t.Fatalf("%q (partial %t): %d clusters", text, partial, n)
	}
	for _, r := range got {
		if unicode.IsControl(r) || unicode.Is(unicode.Cf, r) && r != 0x200C && r != 0x200D && (r < 0xE0020 || r > 0xE007F) {
			t.Fatalf("%q: %U kept in %q", text, r, got)
		}
	}
	if got != strings.TrimSpace(got) || strings.Contains(got, "  ") {
		t.Fatalf("%q: spaces in %q", text, got)
	}
	// The result is the cleaned text up to a cluster boundary, the end trimmed.
	if !strings.HasPrefix(cleaned, got) {
		t.Fatalf("%q: %q is no prefix of %q", text, got, cleaned)
	}
	ok := got == ""
	for _, e := range ends {
		ok = ok || strings.TrimSpace(cleaned[:e]) == got
	}
	if !ok {
		t.Fatalf("%q: %q ends inside a cluster of %q", text, got, cleaned)
	}
	// A whole message of 60 clusters or more gives exactly its first 60.
	if !partial && len(ends) >= TitleChars {
		if want := strings.TrimSpace(cleaned[:ends[TitleChars-1]]); got != want {
			t.Fatalf("%q: %q, want its first %d clusters %q", text, got, TitleChars, want)
		}
	}
	// A short one is kept whole.
	if !partial && len(ends) < TitleChars && got != cleaned {
		t.Fatalf("%q: %q, want %q", text, got, cleaned)
	}
}

// FuzzSessionTitle holds design D1a's rules on any text, whole or cut short.
func FuzzSessionTitle(f *testing.F) {
	family := "👨\u200d👩\u200d👧"
	flag := "🏴" + string(rune(0xE0067)) + string(rune(0xE0062)) + string(rune(0xE007F))
	for _, s := range []string{
		"", " ", "x", "remember the code word",
		"Add a history view to the home screen with resume and delete entries please",
		strings.Repeat("x", 60) + " yyy", strings.Repeat("x", 59) + " yyy", strings.Repeat("x", 61),
		strings.Repeat("é", 70), "e" + strings.Repeat("́", 200), strings.Repeat(family, 70), strings.Repeat("🇫🇷", 70),
		strings.Repeat(flag, 30), "a\x1b[31mb\u009bc\u202ed\u200be\ufeff", "a \t\n b  c", "\xff\xfe a", strings.Repeat("日 ", 70),
	} {
		f.Add(s, false)
		f.Add(s, true)
	}
	f.Fuzz(func(t *testing.T, text string, partial bool) { checkTitle(t, text, partial) })
}

// TestSessionTitleBoundary is design D1a's 60th cluster in every place the
// white space after it can fall: never more than 60 clusters.
func TestSessionTitleBoundary(t *testing.T) {
	for _, b := range []bool{false, true} {
		if got := SessionTitle(strings.Repeat("x", 60)+" yyy", b); got != strings.Repeat("x", 60) {
			t.Errorf("partial %t: %q", b, got)
		}
		if got := SessionTitle("Add a history view to the home screen with resume and delete entries please", b); got != "Add a history view to the home screen with resume and delete" {
			t.Errorf("partial %t: %q", b, got)
		}
		for n := 55; n <= 65; n++ {
			checkTitle(t, strings.Repeat("x", n)+" "+strings.Repeat("y", 10), b)
			checkTitle(t, strings.Repeat("x", n)+"   \t\n"+strings.Repeat("y", 10), b)
		}
	}
	if got := SessionTitle(strings.Repeat("x", 63)+" "+strings.Repeat("y", 10), false); got != strings.Repeat("x", 60) {
		t.Errorf("past 60: %q", got)
	}
	// The tag characters of a flag of a region and the joiners stay.
	flag := "🏴" + string(rune(0xE0067)) + string(rune(0xE0062)) + string(rune(0xE007F))
	if got := SessionTitle("go "+flag+" now", false); got != "go "+flag+" now" {
		t.Errorf("a regional flag: %q", got)
	}
}

// TestSessionTitleCost: the cost of a title does not grow with the message
// beyond the one pass over it: the same few allocations at 1 KiB and at 8
// KiB, for plain text and for a single cluster of combining marks, and a
// time bound (not under -race, which allocates and slows on its own).
func TestSessionTitleCost(t *testing.T) {
	if raceBuild {
		t.Skip("-race allocates on its own")
	}
	plain := func(n int) string { return strings.Repeat("word ", n/5+1)[:n] }
	marks := func(n int) string { return "e" + strings.Repeat("́", n/2) }
	for _, c := range []struct {
		name string
		text func(int) string
	}{{"plain text", plain}, {"one combining-mark cluster", marks}, {"one cluster of white space and marks", func(n int) string { return strings.Repeat(" ́", n/4) }}} {
		small, large := c.text(1<<10), c.text(8<<10)
		a1 := testing.AllocsPerRun(20, func() { _ = SessionTitle(small, true) })
		a8 := testing.AllocsPerRun(20, func() { _ = SessionTitle(large, true) })
		if a1 != a8 || a8 > 3 {
			t.Errorf("%s: %v allocations at 1 KiB, %v at 8 KiB", c.name, a1, a8)
		}
		// The work, counted, not timed (doc/evaluation.md: a time is judged in
		// the timing step only): the scan of the title's clusters steps over
		// each byte once, at most the clusters' a title keeps and the one
		// after, so that its cost is the same at 8 KiB as at 1 KiB, or linear
		// in a cluster as long as the text.
		scanned := func(text string) int64 {
			before := titleScanned.Load()
			_ = SessionTitle(text, true)
			return titleScanned.Load() - before
		}
		s1, s8 := scanned(small), scanned(large)
		if s8 > int64(len(large)) || s1 > int64(len(small)) {
			t.Errorf("%s: %d bytes scanned for %d, %d for %d: a byte read twice", c.name, s1, len(small), s8, len(large))
		}
		if c.name == "plain text" && (s1 != s8 || s8 > 4*TitleChars) {
			t.Errorf("%s: %d bytes scanned at 1 KiB, %d at 8 KiB; want the title's clusters alone", c.name, s1, s8)
		}
	}
}
