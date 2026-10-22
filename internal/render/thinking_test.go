package render

import (
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"

	"charm.land/lipgloss/v2"

	"github.com/Evengelius/agent-command-center/internal/event"
	"github.com/Evengelius/agent-command-center/internal/fixture"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// Plan 22, criteria 2.1 to 2.15 at the render level: zone B's thinking card
// and its animations, its summary row and details, the activity line, from
// states built by hand at chosen frame times (the reducer that makes the
// thinking entries is tested in internal/state).

// thinkAt is the recordings' time ms milliseconds in.
func thinkAt(ms int) time.Time { return fixture.Epoch.Add(time.Duration(ms) * time.Millisecond) }

// thinkBench is a session whose turn runs, opened at 0 ms: the owner's
// prompt, then thinking entry 2 with phase th; Codex's when the phase is
// inferred, Claude Code's otherwise.
func thinkBench(th state.Thought) *state.State {
	if th.ID == 0 {
		th.ID = 2 // as the reducer numbers phases: entry 2's, its key in UI.Live
	}
	s := state.New()
	s.Screen = state.ScreenDashboard
	s.Context.Provider, s.Context.ProviderName, s.Session.Model = "claude-code", "Claude Code", "claude-haiku-5-5"
	origin := event.Observed
	if th.Inferred {
		s.Context.Provider, s.Context.ProviderName, s.Session.Model = "codex", "Codex", "gpt-6-sol"
		origin = event.Inferred
	}
	s.Session.Phase, s.Turn.Phase = state.Ready, state.Running
	s.Stages = []state.Stage{{N: 1, Open: true, StartedAt: thinkAt(0)}}
	s.Conversation = []state.Entry{
		{Seq: 1, Role: "user", Text: "Analyze the repository architecture.", At: thinkAt(0), Stage: 1},
		{Seq: 2, Role: "thinking", Origin: origin, Streaming: th.End.IsZero(), At: th.Start, Stage: 1, Thought: &th},
	}
	return s
}

// phase is entry 2's phase in s.
func phase(s *state.State) *state.Thought { return s.Conversation[1].Thought }

// zoneBText is zone B's inner rows on a frame of ui's size, as text: the
// conversation, then the input.
func zoneBText(s *state.State, ui UI) []string {
	x0 := 0
	if ui.Width >= standardWidth {
		x0 = zoneAWidth(ui.Width)
	}
	var out []string
	in := false
	for _, l := range Draw(s, ui).lines {
		rs := []rune(strings.TrimSuffix(texts([]line{l}), "\n"))
		if len(rs) <= x0+4 {
			continue
		}
		switch {
		case strings.HasPrefix(string(rs[x0:]), "╭─ B"):
			in = true
		case in && rs[x0] == '│':
			out = append(out, string(rs[x0+2:len(rs)-2]))
		case in:
			return out
		}
	}
	return out
}

// boxIn are the rows of the first bordered card in rows, its edges included.
func boxIn(rows []string) []string {
	for i, r := range rows {
		if !strings.HasPrefix(r, "▎ ╭") {
			continue
		}
		for j := i; j < len(rows); j++ {
			if strings.HasPrefix(rows[j], "▎ ╰") {
				return rows[i : j+1]
			}
		}
	}
	return nil
}

// card is entry 2's card or exit as drawn in zone B's inner width w, size
// rows when steady, without its block.
func card(s *state.State, ui UI, w, size int) []line {
	lk, _ := lookAll(s, ui)
	k := lk[1]
	k.size = size
	return newCard(s, ui, s.Conversation[1], cardContent(k.card, w), size).rows(s, ui, 1, k)
}

// bInner is zone B's inner width on a screen w columns wide.
func bInner(w int) int {
	if w < standardWidth {
		return w - 4
	}
	return w - zoneAWidth(w) - 4
}

// barRead reads a bar's row: its half cells filled, each filled cell's
// style, its cells in all, and its scale's label with its style.
func barRead(l line) (halves int, fills []*lipgloss.Style, cells int, label string, labelSt *lipgloss.Style) {
	for _, x := range l {
		if strings.HasSuffix(x.text, "k") && strings.HasPrefix(x.text, " ") {
			label, labelSt = x.text, x.st
			continue
		}
		for _, r := range x.text {
			switch {
			case r == '━' && x.st == stThinkTrack:
				cells++
			case r == '━':
				halves, cells = halves+2, cells+1
				fills = append(fills, x.st)
			case r == '╸':
				halves, cells = halves+1, cells+1
				fills = append(fills, x.st)
			}
		}
	}
	return
}

// bar is a card's bar row: the row before its bottom edge, or the 2-row
// card's second.
func bar(rows []line) line {
	if len(rows) == 2 {
		return rows[1]
	}
	return rows[len(rows)-2]
}

// plainRow is a row's text.
func plainRow(l line) string { return strings.TrimSuffix(texts([]line{l}), "\n") }

// styled is a row's text with each span's style, to compare two frames.
func styled(ls []line) string {
	var b strings.Builder
	for _, l := range ls {
		for _, x := range l {
			fmt.Fprintf(&b, "%p:%s|", x.st, x.text)
		}
		b.WriteString("\n")
	}
	return b.String()
}

var digitsRe = regexp.MustCompile(`[0-9]`)

// TestThinkingCardModeB is criterion 2.1: P12c-block's second phase (it
// began at 7,449 ms, its estimate 650 came at 10,610 after 350) drawn at
// 120x40: seven rows, "00:03" at the right of the expression's row, "≈ 650
// thinking tokens · live estimate", the bar's label "2k"; its fill 650/2,000
// of the bar once grown, 300 ms after the estimate.
func TestThinkingCardModeB(t *testing.T) {
	s := thinkBench(state.Thought{Start: thinkAt(7449), Estimate: 650, EstimateFrom: 350, EstimateAt: thinkAt(10610)})
	ui := UI{Width: 120, Height: 40, Now: thinkAt(10700), Focus: "B"}
	all := zoneBText(s, ui)
	rows := boxIn(all)
	if len(rows) != 7 {
		t.Fatalf("the card has %d rows, want 7:\n%s", len(rows), strings.Join(all, "\n"))
	}
	// Under the speaker's label, a blank row of the block, then the card (the
	// owner's drawing, amendment 18, section 2).
	for i, r := range all {
		if strings.HasPrefix(r, "▎ ╭") && (!strings.HasPrefix(all[i-2], "▎ CLAUDE  Haiku 5.5  ") || strings.TrimRight(all[i-1], " ") != "▎") {
			t.Errorf("above the card: %q, %q", all[i-2], all[i-1])
		}
	}
	word, _ := expression(2, 3251*time.Millisecond)
	for _, c := range []struct {
		row          int
		prefix, tail string
	}{
		{0, "▎ ╭───", "─╮"},
		{1, "▎ │ " + spinGlyph(3251*time.Millisecond) + " " + word, " 00:03 │"},
		{2, "▎ │", "│"},
		{3, "▎ │ ≈ 650 thinking tokens · live estimate ", "│"},
		{4, "▎ │", "│"},
		{5, "▎ │ ━", "━ 2k │"},
		{6, "▎ ╰───", "─╯"},
	} {
		if r := strings.TrimRight(rows[c.row], " "); !strings.HasPrefix(r, c.prefix) || !strings.HasSuffix(r, c.tail) {
			t.Errorf("row %d %q, want %q … %q", c.row, r, c.prefix, c.tail)
		}
	}
	// The bar: 70 cells and " 2k" in the card's 73 (zone B is 84 columns at
	// 120 since plan 24, A 36; it was 94, the bar 80 cells). At 10,700 ms
	// the fill still grows from 350 (the design's 300 ms): 350 + 300 ×
	// ease(0.3), 38 half cells; from 10,910 ms, 650/2,000 of 70 cells,
	// 22.75 cells, 45 half cells.
	w := bInner(120)
	for _, c := range []struct {
		ms, halves int
	}{{10700, 38}, {10910, 45}, {11500, 45}} {
		ui.Now = thinkAt(c.ms)
		halves, _, cells, label, _ := barRead(bar(card(s, ui, w, 7)))
		if halves != c.halves || cells != 70 || label != " 2k" {
			t.Errorf("%d ms: %d half cells of %d, label %q; want %d of 70, \" 2k\"", c.ms, halves, cells, label, c.halves)
		}
	}
}

// TestThinkingBarMath is criteria 2.2 and 2.3: the fill to half a cell, the
// scale doubled until the count fits, a phase's label never lower, a phase
// that ended never filled for it; mode A with a budget.
func TestThinkingBarMath(t *testing.T) {
	for _, c := range []struct {
		v, scale int64
		label    string
	}{{50, 2000, " 2k"}, {1950, 2000, " 2k"}, {2000, 2000, " 2k"}, {2200, 4000, " 4k"}, {4001, 8000, " 8k"}, {9000, 16000, " 16k"}} {
		if sc := barScale(c.v); sc != c.scale || scaleLabel(sc) != c.label {
			t.Errorf("%d tokens: scale %d %q, want %d %q", c.v, sc, scaleLabel(sc), c.scale, c.label)
		}
	}
	for _, c := range []struct {
		w      int
		v      float64
		scale  int64
		halves int
		text   string
	}{
		{80, 650, 2000, 52, ""}, {80, 1950, 2000, 156, ""}, {10, 125, 1000, 2, "━━━━━━━━━━"},
		{10, 150, 1000, 3, "━╸━━━━━━━━"}, {10, 49, 1000, 0, ""}, {10, 50, 1000, 1, "╸━━━━━━━━━"},
		{10, 1000, 1000, 20, ""}, {10, 2500, 1000, 20, ""}, {10, 0, 1000, 0, ""},
	} {
		h := fillHalves(c.w, c.v, c.scale)
		if h != c.halves {
			t.Errorf("%v of %d on %d cells: %d half cells, want %d", c.v, c.scale, c.w, h, c.halves)
		}
		l := barCells(c.w, h, stThinkFill, -1)
		if c.text != "" && plainRow(l) != c.text {
			t.Errorf("%v of %d on %d cells drawn %q, want %q", c.v, c.scale, c.w, plainRow(l), c.text)
		}
		if got, _, cells, _, _ := barRead(l); got != h || cells != c.w {
			t.Errorf("%v of %d: the bar reads %d half cells of %d", c.v, c.scale, got, cells)
		}
	}
	// P12c-block's second phase: its labels never go down.
	var labels []string
	from := int64(0)
	for _, v := range []int64{50, 350, 650, 850, 1100, 1450, 1700, 1950, 2200, 2450, 2700} {
		s := thinkBench(state.Thought{Start: thinkAt(7449), Estimate: v, EstimateFrom: from, EstimateAt: thinkAt(15000)})
		_, _, _, label, _ := barRead(bar(card(s, UI{Now: thinkAt(16000)}, 90, 7)))
		labels, from = append(labels, strings.TrimSpace(label)), v
	}
	if got := strings.Join(labels, " "); got != "2k 2k 2k 2k 2k 2k 2k 2k 4k 4k 4k" {
		t.Errorf("labels %s", got)
	}
	// The phase ended at 2,700 of 4k: its exit's bar is the last value's, 108
	// half cells of 80 cells, never full because it ended; then its summary.
	s := thinkBench(state.Thought{Start: thinkAt(7449), End: thinkAt(21351), Estimate: 2700, EstimateFrom: 2450, EstimateAt: thinkAt(20632)})
	ui := UI{Now: thinkAt(21351), Live: map[uint64]bool{2: true}}
	if halves, _, cells, label, _ := barRead(bar(card(s, ui, 90, 7))); halves != 108 || cells != 80 || label != " 4k" {
		t.Errorf("the exit's bar: %d half cells of %d, %q; want 108 of 80, \" 4k\"", halves, cells, label)
	}
	// 2.3, mode A: a budget of 4,096 and 1,284; the fill 1,284/4,096 of the
	// card's 83 cells, no label. No adapter sets a budget.
	s = thinkBench(state.Thought{Start: thinkAt(7449), Estimate: 1284, EstimateFrom: 900, EstimateAt: thinkAt(9000), Budget: 4096})
	rows := card(s, UI{Now: thinkAt(12000)}, 90, 7)
	if r := plainRow(rows[3]); !strings.HasPrefix(r, "│ Thinking · 1,284 / 4,096 tokens ") {
		t.Errorf("mode A's count row %q", r)
	}
	if halves, _, cells, label, _ := barRead(bar(rows)); halves != 52 || cells != 83 || label != "" {
		t.Errorf("mode A's bar: %d half cells of %d, label %q; want 52 of 83, none", halves, cells, label)
	}
}

// TestThinkingModeC is criterion 2.4: no estimate yet, Claude Code's
// "token usage pending"; a Codex step's "Model working"; the segment at its
// places for 0, 0.4, 0.8 and 1.2 s; no token count drawn.
func TestThinkingModeC(t *testing.T) {
	for _, c := range []struct {
		th    state.Thought
		count string
	}{
		{state.Thought{Start: thinkAt(7449)}, "│ Thinking · token usage pending "},
		{state.Thought{Start: thinkAt(7449), Inferred: true}, "│ Model working · reasoning tokens reported when the step ends "},
	} {
		s := thinkBench(c.th)
		for _, d := range []int{0, 400, 800, 1200} {
			ui := UI{Now: thinkAt(7449 + 1600 + d)}
			rows := card(s, ui, 90, 7)
			if r := plainRow(rows[3]); !strings.HasPrefix(r, c.count) || digitsRe.MatchString(r) {
				t.Errorf("%s: the count row %q", c.count, r)
			}
			b := bar(rows)
			if digitsRe.MatchString(plainRow(b)) {
				t.Errorf("a number on the bar's row %q", plainRow(b))
			}
			// The segment: 13 of 83 cells (83/6), its ends lit, from 83 × d /
			// 1.6 s.
			head := 83 * d / 1600
			var got []string
			cell := 0
			for _, x := range b[1 : len(b)-1] { // inside "│ " and " │"
				for range x.text {
					switch x.st {
					case stThinkEdge:
						got = append(got, fmt.Sprintf("e%d", cell))
					case stThinkFill:
						got = append(got, fmt.Sprintf("f%d", cell))
					}
					cell++
				}
			}
			want := []string{fmt.Sprintf("e%d", head)}
			for k := 1; k < 12; k++ {
				want = append(want, fmt.Sprintf("f%d", head+k))
			}
			want = append(want, fmt.Sprintf("e%d", head+12))
			if strings.Join(got, " ") != strings.Join(want, " ") || cell != 83 {
				t.Errorf("%s at %d ms: the segment %v in %d cells, want %v", c.count, d, got, cell, want)
			}
		}
	}
	// The sweep comes back in at the left as it leaves at the right.
	if got := plainRow(sweep(12, 1500*time.Millisecond)); len([]rune(got)) != 12 {
		t.Errorf("a sweep of 12 cells drawn %q", got)
	}
	l := sweep(12, 1500*time.Millisecond) // its head at 11: cells 11, 0 to 4
	if l[0].st != stThinkFill || l[len(l)-1].st != stThinkEdge {
		t.Errorf("the sweep does not wrap: %v", l)
	}
}

// TestThinkingGrowth is criteria 2.5 and 2.6: an estimate from 300 to 650
// at T; the fill f(300) at T, between at T+150, f(650) from T+300; the edge
// lit until T+300, in the mid shade until T+700, then normal; the fill
// constant between two estimates; the count only 300 or 650, lit for
// 400 ms after the change.
func TestThinkingGrowth(t *testing.T) {
	const T = 20000
	before := state.Thought{Start: thinkAt(15000), Estimate: 300, EstimateFrom: 50, EstimateAt: thinkAt(T - 1500)}
	after := before
	after.Estimate, after.EstimateFrom, after.EstimateAt = 650, 300, thinkAt(T)
	frame := func(ms int) (halves int, lead *lipgloss.Style, count string, countSt *lipgloss.Style) {
		th := after
		if ms < T {
			th = before
		}
		rows := card(thinkBench(th), UI{Now: thinkAt(ms)}, 90, 7)
		halves, fills, _, _, _ := barRead(bar(rows))
		if len(fills) > 0 {
			lead = fills[len(fills)-1]
		}
		for _, x := range rows[3] {
			if strings.HasPrefix(x.text, "≈ ") {
				count, countSt = x.text, x.st
			}
		}
		return halves, lead, count, countSt
	}
	for _, c := range []struct {
		ms, halves int
		lead       *lipgloss.Style
	}{
		{T, 24, stThinkEdge}, {T + 150, 48, stThinkEdge}, {T + 299, 51, stThinkEdge},
		{T + 300, 52, stThinkMid}, {T + 699, 52, stThinkMid}, {T + 700, 52, stThinkFill},
	} {
		if halves, lead, _, _ := frame(c.ms); halves != c.halves || lead != c.lead {
			t.Errorf("T+%d ms: %d half cells, lead %p; want %d, %p", c.ms-T, halves, lead, c.halves, c.lead)
		}
	}
	if h, _, _, _ := frame(T + 150); h <= 24 || h >= 52 {
		t.Errorf("T+150 ms: %d half cells, not between f(300) and f(650)", h)
	}
	for ms := T + 300; ms < T+1500; ms += 50 {
		if h, _, _, _ := frame(ms); h != 52 {
			t.Errorf("T+%d ms: the fill moved to %d half cells between two estimates", ms-T, h)
		}
	}
	for ms := T - 200; ms <= T+1000; ms += 10 {
		_, _, count, st := frame(ms)
		want, lit := "≈ 300", false
		if ms >= T {
			want, lit = "≈ 650", ms < T+400
		}
		if count != want || (st == stThinkLit) != lit {
			t.Errorf("T%+d ms: the count %q, lit %v; want %q, lit %v", ms-T, count, st == stThinkLit, want, lit)
		}
	}
	// A shimmer, the fill unchanged: once every 2 s from T+700, across the
	// 26 cells filled in 600 ms, in the mid shade.
	for _, c := range []struct {
		ms, cell int
	}{{T + 700, 0}, {T + 1000, 13}, {T + 1299, 25}, {T + 1300, -1}, {T + 2699, -1}, {T + 2700, 0}} {
		_, fills, _, _, _ := barRead(bar(card(thinkBench(after), UI{Now: thinkAt(c.ms)}, 90, 7)))
		got := -1
		for i, st := range fills {
			if st == stThinkMid {
				got = i
			}
		}
		if got != c.cell || len(fills) != 26 {
			t.Errorf("T+%d ms: the shimmer on cell %d of %d, want %d of 26", c.ms-T, got, len(fills), c.cell)
		}
	}
}

// TestThinkingSpinner is criterion 2.7: six frames, 100 ms each, cycling;
// still while a request waits for the owner.
func TestThinkingSpinner(t *testing.T) {
	var got []string
	for ms := 0; ms < 1200; ms += 100 {
		got = append(got, spinGlyph(time.Duration(ms)*time.Millisecond))
	}
	if strings.Join(got, "") != "◜◠◝◞◡◟◜◠◝◞◡◟" {
		t.Errorf("frames %q", got)
	}
	s := thinkBench(state.Thought{Start: thinkAt(7449)})
	for ms := 300; ms < 1500; ms += 100 {
		rows := card(s, UI{Now: thinkAt(7449 + ms)}, 90, 7)
		if r := plainRow(rows[1]); !strings.HasPrefix(r, "│ "+spinGlyph(time.Duration(ms)*time.Millisecond)+" ") {
			t.Errorf("%d ms: the spinner's row %q", ms, r)
		}
	}
	s.Turn.Phase = state.WaitingApproval
	if r := plainRow(card(s, UI{Now: thinkAt(9000)}, 90, 7)[1]); !strings.HasPrefix(r, "│ ◌ ") {
		t.Errorf("a request waiting: the spinner's row %q", r)
	}
}

// TestThinkingExpressions is criterion 2.8: the owner's ten word for word; a
// new one every 4 s, never twice in a row over 200 s, dim for its first
// 200 ms.
func TestThinkingExpressions(t *testing.T) {
	want := []string{"Crunching...", "Vibing...", "Connecting dots...", "Smoothing...", "Pondering...", "Untangling...", "Brewing ideas...", "Cooking...", "Piecing it together...", "Making magic..."}
	if fmt.Sprint(expressions[:]) != fmt.Sprint(want) {
		t.Fatalf("expressions %q", expressions)
	}
	// From its seed (Seq mod 10), each slot three on: entry 17's first ten.
	var order []string
	for n := range 10 {
		word, _ := expression(17, time.Duration(n)*4*time.Second)
		order = append(order, word)
	}
	if got := strings.Join(order, " "); got != "Cooking... Crunching... Smoothing... Brewing ideas... Making magic... Connecting dots... Untangling... Piecing it together... Vibing... Pondering..." {
		t.Errorf("entry 17's order: %s", got)
	}
	for seed := uint64(0); seed < 10; seed++ {
		seen := map[string]bool{}
		prev, _ := expression(seed, 0)
		if prev != want[seed] {
			t.Errorf("seed %d starts with %q", seed, prev)
		}
		for ms := 100; ms <= 200000; ms += 100 {
			d := time.Duration(ms) * time.Millisecond
			word, dim := expression(seed, d)
			seen[word] = true
			slot := d%(4*time.Second) == 0
			if (word != prev) != slot {
				t.Fatalf("seed %d at %v: %q after %q", seed, d, word, prev)
			}
			if wantDim := d >= 4*time.Second && d%(4*time.Second) < 200*time.Millisecond; dim != wantDim {
				t.Errorf("seed %d at %v: dim %v", seed, d, dim)
			}
			prev = word
		}
		if len(seen) != 10 {
			t.Errorf("seed %d: %d of the ten in 200 s", seed, len(seen))
		}
	}
	// On the card: the dim shade for a new one's first 200 ms.
	s := thinkBench(state.Thought{Start: thinkAt(7449)})
	for _, c := range []struct {
		ms int
		st *lipgloss.Style
	}{{3990, stThinkWord}, {4000, stThinkWordDim}, {4199, stThinkWordDim}, {4200, stThinkWord}} {
		row := card(s, UI{Now: thinkAt(7449 + c.ms)}, 90, 7)[1] // "│ ", the spinner, " ", the expression
		if word, _ := expression(2, time.Duration(c.ms)*time.Millisecond); row[3].text != word || row[3].st != c.st {
			t.Errorf("%d ms: the expression %q styled %p, want %q %p", c.ms, row[3].text, row[3].st, word, c.st)
		}
	}
}

// TestThinkingEntrance is criterion 2.9: a phase drawn live from its start
// reveals the border with a light on its top edge, then the spinner's row,
// the count's and the bar's track; the card's rows are final from the first
// frame.
func TestThinkingEntrance(t *testing.T) {
	s := thinkBench(state.Thought{Start: thinkAt(7449)})
	// spanSt is the style of the span of l holding text, nil for none.
	spanSt := func(l line, text string) *lipgloss.Style {
		x, _ := spanOf(l, text)
		return x.st
	}
	// shown is a row's text inside the border.
	shown := func(l line) string { return strings.TrimSpace(strings.Trim(plainRow(l), "│")) }
	word, _ := expression(2, 0)
	for _, size := range []int{7, 5, 2} {
		full := map[int]int{7: 83, 5: 83, 2: 87}[size] // the bar's cells
		for _, ms := range []int{0, 35, 70, 105, 140, 175, 210, 245, 279, 280} {
			rows := card(s, UI{Now: thinkAt(7449 + ms)}, 90, size)
			if len(rows) != size {
				t.Fatalf("size %d at %d ms: %d rows", size, ms, len(rows))
			}
			// The spinner's row, the count's (the same without a border), the bar's.
			spin, count, b := rows[1], rows[map[int]int{7: 3, 5: 2, 2: 0}[size]], bar(rows)
			if size == 2 {
				spin = rows[0]
			}
			switch stage := ms / 70; {
			case stage == 0 && size > 2:
				// The edge dim, a light of four cells crossing it.
				at, n, cells := -1, 0, 0
				for _, x := range rows[0] {
					switch x.st {
					case stThinkEdge:
						at, n = cells, n+textWidth(x.text)
					case stThinkDim:
					default:
						t.Errorf("size %d at %d ms: the edge %q styled %p", size, ms, x.text, x.st)
					}
					cells += textWidth(x.text)
				}
				if want := (87 - 4 + 1) * ms / 70; n != 4 || at != want {
					t.Errorf("size %d at %d ms: a light of %d cells at %d, want 4 at %d", size, ms, n, at, want)
				}
				for _, r := range rows[1 : size-1] {
					if shown(r) != "" {
						t.Errorf("size %d at %d ms: a row shows %q", size, ms, plainRow(r))
					}
				}
			case stage == 0:
				if shown(rows[0]) != "" || shown(rows[1]) != "" {
					t.Errorf("size 2 at %d ms: %q", ms, texts(rows))
				}
			case stage == 1:
				if spanSt(spin, word) != stThinkDim || spanSt(spin, "00:00") != stThinkDim {
					t.Errorf("size %d at %d ms: the spinner's row %q not dim", size, ms, plainRow(spin))
				}
				if strings.Contains(plainRow(spin)+plainRow(count), "Thinking") || shown(b) != "" {
					t.Errorf("size %d at %d ms: the count or the bar shows", size, ms)
				}
			case stage == 2:
				if spanSt(spin, word) != stThinkWord || spanSt(count, "Thinking") != stThinkDim || spanSt(count, " · token usage pending") != stThinkDim {
					t.Errorf("size %d at %d ms: the spinner's row %q not normal, or the count %q not dim", size, ms, plainRow(spin), plainRow(count))
				}
				if shown(b) != "" {
					t.Errorf("size %d at %d ms: the bar shows %q", size, ms, plainRow(b))
				}
			case stage == 3:
				if spanSt(count, "Thinking") != stBright {
					t.Errorf("size %d at %d ms: the count still dim", size, ms)
				}
				if _, _, cells, _, _ := barRead(b); cells != full*(ms-210)/70 {
					t.Errorf("size %d at %d ms: %d cells of the bar shown, want %d", size, ms, cells, full*(ms-210)/70)
				}
			default: // steady
				if _, _, cells, _, _ := barRead(b); cells != full || spanSt(rows[0], "─") == stThinkDim || spanSt(spin, word) != stThinkWord {
					t.Errorf("size %d at %d ms: not the steady card:\n%s", size, ms, texts(rows))
				}
			}
		}
	}
	// In zone B the rows are reserved from the first frame: the
	// conversation's row count does not change through the entrance.
	n := -1
	for ms := 0; ms <= 400; ms += 10 {
		_, rows, _, _ := conversation(s, UI{Now: thinkAt(7449 + ms)}, 90, 25)
		if n >= 0 && rows != n {
			t.Errorf("%d ms: %d rows, then %d", ms, n, rows)
		}
		n = rows
	}
}

// TestThinkingExit is criterion 2.10: at its end the spinner is ✓ and the
// expression the duration; fewer rows at End+100 and End+200; from End+300
// the summary row only; no exit for a phase no frame drew live.
func TestThinkingExit(t *testing.T) {
	th := state.Thought{Start: thinkAt(7449), End: thinkAt(10265), Estimate: 650, EstimateFrom: 350, EstimateAt: thinkAt(9800)}
	s := thinkBench(th)
	live := map[uint64]bool{2: true}
	stage := func(ms int, l map[uint64]bool) thinkStage {
		return thoughtStage(UI{Now: thinkAt(ms), Live: l}, s.Conversation[1])
	}
	for _, c := range []struct {
		ms   int
		rows int
		head string
	}{
		{10265, 7, "│ ✓ Thought for 2.8s"}, {10364, 7, "│ ✓ Thought for 2.8s"},
		{10365, 3, "│ ✓ Thought for 2.8s"}, {10464, 3, "│ ✓ Thought for 2.8s"},
		{10465, 1, " ▸ ✓ Thought for 2.8s · ≈ 650 thinking tokens (estimate)"},
	} {
		ui := UI{Now: thinkAt(c.ms), Live: live}
		var rows []line
		if c.rows == 1 {
			// From End+200 the summary row, lit, laid out as a summary row, so
			// nothing moves when the exit ends (plan 22, the review).
			if stage(c.ms, live) != thinkSettle {
				t.Fatalf("%d ms: not the exit's last stage", c.ms)
			}
			rows = []line{summaryRow(s, ui, 1, s.Conversation[1], 87, look{}, false, settleLit(ui, s.Conversation[1].Thought))}
		} else {
			if stage(c.ms, live) != thinkExit {
				t.Fatalf("%d ms: not the exit", c.ms)
			}
			rows = card(s, ui, 90, 7)
		}
		at := min(1, len(rows)-1)
		if len(rows) != c.rows || !strings.HasPrefix(plainRow(rows[at]), c.head) {
			t.Errorf("End+%d ms: %d rows, %q; want %d, %q", c.ms-10265, len(rows), plainRow(rows[at]), c.rows, c.head)
		}
		if c.rows > 1 {
			if r := plainRow(rows[1]); !strings.HasSuffix(strings.TrimRight(r, " │"), "00:02") {
				t.Errorf("End+%d ms: the time it took not at the right: %q", c.ms-10265, r)
			}
			if x := rows[1][1]; x.text != "✓" || x.st != stDone {
				t.Errorf("End+%d ms: the mark %q styled %p", c.ms-10265, x.text, x.st)
			}
		}
		if c.rows == 7 {
			// The bar as it ended: 650 of 2k, no edge, no shimmer.
			halves, fills, _, _, _ := barRead(bar(rows))
			for _, st := range fills {
				if st != stThinkFill {
					t.Errorf("End+%d ms: a cell of the frozen bar styled %p", c.ms-10265, st)
				}
			}
			if halves != 52 {
				t.Errorf("End+%d ms: the bar at %d half cells, want 52", c.ms-10265, halves)
			}
		}
	}
	// The summary row lit at End+200, settling at End+250, settled from 300.
	for _, c := range []struct {
		ms          int
		words, rest *lipgloss.Style
	}{{10465, stThinkLit, stBright}, {10515, stThinkWord, stText}, {10565, stThinkMid, stMuted}} {
		var l line
		if c.ms < 10565 {
			ui := UI{Now: thinkAt(c.ms), Live: live}
			l = summaryRow(s, ui, 1, s.Conversation[1], 87, look{}, false, settleLit(ui, s.Conversation[1].Thought))
		} else {
			if stage(c.ms, live) != thinkSummary {
				t.Fatalf("End+300 ms: not the summary row")
			}
			l = summaryRow(s, UI{Now: thinkAt(c.ms)}, 1, s.Conversation[1], 87, look{}, false, 0)
		}
		w, _ := spanOf(l, "Thought for 2.8s")
		r, _ := spanOf(l, " · ≈ 650 thinking tokens (estimate)")
		if w.st != tint(c.words) || r.st != tint(c.rest) {
			t.Errorf("End+%d ms: the words %p, the rest %p", c.ms-10265, w.st, r.st)
		}
	}
	// No frame drew it live: its summary at once (a replay, plan 22, design 7).
	for _, ms := range []int{10265, 10365, 10565} {
		if got := stage(ms, nil); got != thinkSummary {
			t.Errorf("%d ms, not drawn live: stage %d", ms, got)
		}
	}
	// The 2-row card: its ✓ row alone at End+100.
	if rows := card(s, UI{Now: thinkAt(10365), Live: live}, 90, 2); len(rows) != 1 || !strings.HasPrefix(plainRow(rows[0]), "✓ Thought for 2.8s") {
		t.Errorf("2 rows at End+100: %q", texts(rows))
	}
	// Interrupted: ✗ in red.
	phase(s).Outcome = "interrupted"
	if l := card(s, UI{Now: thinkAt(10265), Live: live}, 90, 7)[1]; l[1].text != "✗" || l[1].st != stFailed || !strings.Contains(plainRow(l), "Thinking interrupted after 2.8s") {
		t.Errorf("interrupted: %q", plainRow(l))
	}
	// A Codex step that did not reason: dimmed, then its edge and a row,
	// then a row, then nothing.
	s = thinkBench(state.Thought{Start: thinkAt(7000), End: thinkAt(9000), Inferred: true, Dropped: true})
	for _, c := range []struct{ ms, rows int }{{9000, 7}, {9100, 2}, {9200, 1}, {9300, 0}} {
		ui := UI{Now: thinkAt(c.ms), Live: live}
		var rows []line
		if thoughtStage(ui, s.Conversation[1]) == thinkExit {
			rows = card(s, ui, 90, 7)
		}
		if len(rows) != c.rows {
			t.Errorf("a dropped step at End+%d: %d rows, want %d", c.ms-9000, len(rows), c.rows)
		}
		for _, r := range rows {
			for _, x := range r {
				if strings.TrimSpace(x.text) != "" && x.st != stThinkDim {
					t.Errorf("a dropped step at End+%d: %q styled %p", c.ms-9000, x.text, x.st)
				}
			}
		}
	}
	s.Turn.Phase = state.Idle // no activity line: nothing of the CLI's left
	if got := strings.Join(rowTexts(bRows(s, UI{Now: thinkAt(9300), Live: live}, 90)), "\n"); strings.Contains(got, "CODEX") {
		t.Errorf("a dropped step leaves a row:\n%s", got)
	}
}

// TestThinkingSummary is criterion 2.11: the summary rows word for word,
// drawn as tool rows, their end time and origin at the right.
func TestThinkingSummary(t *testing.T) {
	ms := func(n int) time.Time { return lookAt.Add(time.Duration(n) * time.Millisecond) }
	for _, c := range []struct {
		th   state.Thought
		want string
	}{
		{state.Thought{Start: ms(940), End: ms(3756), Tokens: 719, HasTokens: true}, "✓ Thought for 2.8s · 719 thinking tokens"},
		{state.Thought{Start: ms(940), End: ms(3756), Estimate: 650}, "✓ Thought for 2.8s · ≈ 650 thinking tokens (estimate)"},
		{state.Thought{Start: ms(940), End: ms(996)}, "✓ Thought for 0.1s"},
		{state.Thought{Start: ms(1200), End: ms(4340), Inferred: true, Confirmed: true, Tokens: 83, HasTokens: true}, "✓ Thought for up to 3.1s · 83 reasoning tokens"},
		{state.Thought{End: ms(4340), Inferred: true, Confirmed: true, Tokens: 83, HasTokens: true}, "✓ Thought · 83 reasoning tokens"},
		// A message cut short (P303a): the CLI's 0 is no count, so the row says
		// nothing after the outcome, its estimate included (plan 22, correction C1).
		{state.Thought{Start: ms(940), End: ms(1140), Outcome: "interrupted", Cut: true}, "✗ Thinking interrupted after 0.2s"},
		{state.Thought{Start: ms(940), End: ms(1140), Outcome: "interrupted", Cut: true, CutTokens: 7, Estimate: 50}, "✗ Thinking interrupted after 0.2s"},
		{state.Thought{Start: ms(940), End: ms(1140), Outcome: "interrupted", Estimate: 50}, "✗ Thinking interrupted after 0.2s"},
		{state.Thought{Start: ms(940), End: ms(1140), Outcome: "failed", Tokens: 9, HasTokens: true}, "✗ Thinking stopped after 0.2s · the turn failed"},
		{state.Thought{Start: ms(0), End: ms(1200), Tokens: 1268, HasTokens: true, Shared: 2}, "✓ Thought for 1.2s · 1,268 thinking tokens over 2 phases"},
		{state.Thought{Start: ms(0), End: ms(3476), Tokens: 717, HasTokens: true}, "✓ Thought for 3.5s · 717 thinking tokens"},
		{state.Thought{Start: ms(0), End: ms(13902), Tokens: 2852, HasTokens: true}, "✓ Thought for 14s · 2,852 thinking tokens"},
		{state.Thought{Start: ms(0), End: ms(65000)}, "✓ Thought for 1m 05s"},
	} {
		s := thinkBench(c.th)
		s.Turn.Phase = state.Idle
		e := s.Conversation[1]
		if !Expandable(e) || thoughtStage(UI{}, e) != thinkSummary {
			t.Errorf("%q: not a summary row", c.want)
		}
		text, _, _, _ := thoughtSummary(e.Thought)
		if mark := map[bool]string{true: "✗ ", false: "✓ "}[strings.HasPrefix(c.want, "✗")]; mark+text != c.want {
			t.Errorf("summary %q, want %q", mark+text, c.want)
		}
		// Drawn as a tool row: ▸, the mark, the text; the end time and the
		// origin at the right; the text the entry's first line, copied.
		var row string
		var l line
		for _, r := range bRows(s, UI{}, 100) {
			if strings.Contains(plainRow(r), text) {
				row, l = strings.TrimRight(plainRow(r), " "), r
			}
		}
		origin := map[bool]string{true: "inferred", false: "observed"}[c.th.Inferred]
		if !strings.HasPrefix(row, "▎  ▸ "+c.want+" ") || !strings.HasSuffix(row, clockSec(c.th.End)+"  "+origin) {
			t.Errorf("row %q, want %q … %s  %s", row, "▎  ▸ "+c.want, clockSec(c.th.End), origin)
		}
		if got := entryLines(s, e); len(got) == 0 || got[0] != text {
			t.Errorf("entry lines %q", got)
		}
		if x, _ := spanOf(l, strings.Fields(text)[0]); x.ref.kind != refText || x.ref.item != 1 || x.ref.line != 0 || x.ref.off != 0 {
			t.Errorf("%q: the summary is not the entry's content: %+v", c.want, x.ref)
		}
		if x, _ := spanOf(l, "▸"); x.st != tint(stFoldMark) {
			t.Errorf("%q: ▸ styled %p", c.want, x.st)
		}
	}
	// The colours: the words lavender, the rest muted, on the tool rows'
	// ground; an exact count that came after the phase lit for 600 ms.
	th := state.Thought{Start: thinkAt(7449), End: thinkAt(10265), Estimate: 650, Tokens: 719, HasTokens: true, TokensAt: thinkAt(10900)}
	s := thinkBench(th)
	for _, c := range []struct {
		ms int
		st *lipgloss.Style
	}{{10900, stThinkLit}, {11499, stThinkLit}, {11500, stMuted}} {
		l := summaryRow(s, UI{Now: thinkAt(c.ms)}, 1, s.Conversation[1], 87, look{}, false, 0)
		w, _ := spanOf(l, "Thought for 2.8s")
		n, _ := spanOf(l, "719")
		if w.st != tint(stThinkMid) || n.st != tint(c.st) || n.text != "719" {
			t.Errorf("%d ms: the words %p, the count %q %p", c.ms, w.st, n.text, n.st)
		}
	}
	for _, d := range []struct {
		d    time.Duration
		want string
	}{{56 * time.Millisecond, "0.1s"}, {2816 * time.Millisecond, "2.8s"}, {2850 * time.Millisecond, "2.9s"}, {3476 * time.Millisecond, "3.5s"},
		{9950 * time.Millisecond, "10s"}, {13902 * time.Millisecond, "14s"}, {59600 * time.Millisecond, "1m 00s"}, {65 * time.Second, "1m 05s"}} {
		if got := thoughtFor(d.d); got != d.want {
			t.Errorf("thoughtFor(%v) = %q, want %q", d.d, got, d.want)
		}
	}
}

// TestThinkingDetails is criterion 2.12: the details open under the row
// (the times, each value's source and scope) and fold; they are the entry's
// lines, copied; the full view shows the phase whole; the entry cursor stops
// on a summary row, never on a live card; a run of calls folds only calls.
func TestThinkingDetails(t *testing.T) {
	ms := func(n int) time.Time { return lookAt.Add(time.Duration(n) * time.Millisecond) }
	claude := state.Thought{Start: ms(940), End: ms(3756), Estimate: 650, Tokens: 719, HasTokens: true, TokensAt: ms(4300)}
	codex := state.Thought{Start: ms(1200), End: ms(5300), Inferred: true, Confirmed: true, Tokens: 83, HasTokens: true, TokensAt: ms(5400)}
	for _, c := range []struct {
		th   state.Thought
		want []string
	}{
		{claude, []string{
			"from 11:30:00.940 to 11:30:03.756 · a thinking block, observed",
			"719 thinking tokens · exact, for this phase · message_delta usage.output_tokens_details.thinking_tokens",
			"live estimate 650 at its last update · system/thinking_tokens, estimated by the CLI",
			"no thinking text: the CLI sends none",
		}},
		{codex, []string{
			"the model step from 11:30:01.200 (inferred) to its reasoning's end at 11:30:05.300 · reasoning item",
			"83 reasoning tokens · exact, for this step · thread/tokenUsage/updated last.reasoningOutputTokens",
			"no live estimate: Codex reports reasoning tokens when the step ends", // plan 22, amendment 20
			"no reasoning summary: Codex sent none",
		}},
	} {
		s := thinkBench(c.th)
		e := s.Conversation[1]
		if got := entryLines(s, e)[1:]; strings.Join(got, "\n") != strings.Join(c.want, "\n") {
			t.Errorf("details\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(c.want, "\n"))
		}
		closed := rowTexts(bRows(s, UI{}, 200))
		open := rowTexts(bRows(s, UI{Unfolded: map[string]bool{ThoughtKey(e): true}}, 200))
		if len(open) != len(closed)+4 {
			t.Errorf("open: %d rows, closed %d", len(open), len(closed))
		}
		for i, r := range open {
			if !strings.HasPrefix(r, "▎  ▾ ✓ Thought") {
				continue
			}
			for j, want := range c.want {
				if open[i+1+j] != "▎     "+want {
					t.Errorf("detail %d %q, want %q", j, open[i+1+j], "▎     "+want)
				}
			}
		}
		// Copied whole, the summary and its details, from the content.
		ui := UI{Width: 200, Height: 60, Unfolded: map[string]bool{ThoughtKey(e): true}}
		sc := Draw(s, ui)
		ls := entryLines(s, e)
		got := sc.Text(s, Selection{From: Pos{Panel: 'B', Item: 1}, To: Pos{Panel: 'B', Item: 1, Line: len(ls) - 1, Off: len(ls[len(ls)-1])}})
		if got != strings.Join(ls, "\n") {
			t.Errorf("copied %q", got)
		}
		// The full view: the summary, then the details.
		view := texts(entryViewer(s, UI{Width: 120, Height: 40}, 1, &Screen{}))
		for _, want := range append([]string{"THINKING", ls[0]}, c.want[0][:30]) {
			if !strings.Contains(view, want) {
				t.Errorf("the full view lacks %q:\n%s", want, view)
			}
		}
	}
	// The entry cursor: on a summary row, never on a live card or its exit.
	s := thinkBench(state.Thought{Start: thinkAt(7449)})
	ui := UI{Now: thinkAt(8000), Live: map[uint64]bool{2: true}}
	if Targets(s, ui)[1] {
		t.Error("the cursor stops on a live card")
	}
	phase(s).End = thinkAt(9000)
	ui.Now = thinkAt(9100)
	if Targets(s, ui)[1] {
		t.Error("the cursor stops on an exit")
	}
	ui.Now = thinkAt(9300)
	if !Targets(s, ui)[1] {
		t.Error("the cursor does not stop on the summary row")
	}
	// Its hint, and the footer's: → opens, ← folds.
	ui.Entry, ui.Focus = 2, "B"
	if got := entryFooter(s, ui); got != "Enter full view · → details · ↑↓ entries · Esc back to the input · F1 keys" {
		t.Errorf("footer %q", got)
	}
	ui.Unfolded = map[string]bool{ThoughtKey(s.Conversation[1]): true}
	if got := entryFooter(s, ui); got != "Enter full view · ← fold · ↑↓ entries · Esc back to the input · F1 keys" {
		t.Errorf("footer %q", got)
	}
	if got := strings.Join(rowTexts(bRows(s, ui, 120)), "\n"); !strings.Contains(got, "Enter full view · ← fold · ↑↓ entries · Esc back") {
		t.Errorf("no hint on the row:\n%s", got)
	}
	// Plan 19's folds count calls only; a phase between two calls of a run
	// folds with them.
	b := newConvBench("claude-code", "claude-haiku-5-5")
	b.prompt("go")
	b.calls(0, 2)
	b.s.Conversation = append(b.s.Conversation, state.Entry{Seq: 99, Role: "thinking", Origin: event.Observed, At: lookAt, Stage: 1,
		Thought: &state.Thought{Start: lookAt, End: lookAt.Add(time.Second), Tokens: 5, HasTokens: true}})
	b.calls(2, 4)
	folded := strings.Join(rowTexts(bRows(b.s, UI{}, 100)), "\n")
	if !strings.Contains(folded, "▸ 3 earlier tool events · Expand") || strings.Contains(folded, "Thought for") {
		t.Errorf("folded:\n%s", folded)
	}
	open := strings.Join(rowTexts(bRows(b.s, UI{Unfolded: map[string]bool{"c0": true}}, 100)), "\n")
	if !strings.Contains(open, "▾ 3 earlier tool events · Fold") || !strings.Contains(open, "▎  ▸ ✓ Thought for 1.0s · 5 thinking tokens") ||
		strings.Count(open, "✓ Read") != 6 || strings.Contains(open, "▎\n▎  ▸ ✓ Thought") {
		t.Errorf("expanded, the phase among the calls with no blank row:\n%s", open)
	}
}

// TestThinkingActivity is criterion 2.13: while the turn works with no
// card, the spinner, an expression, the turn's time and the agents at work;
// none while a request waits, while text streams, once the turn ended.
func TestThinkingActivity(t *testing.T) {
	bench := func() *state.State {
		s := thinkBench(state.Thought{Start: thinkAt(2000), End: thinkAt(4000), Tokens: 120, HasTokens: true})
		s.Tools["t1"] = &state.ToolCall{ID: "t1", Name: "Bash", Status: state.ToolRunning, StartedAt: thinkAt(4100), Origin: event.Observed}
		s.Conversation = append(s.Conversation, state.Entry{Seq: 3, Role: "tool", MessageID: "t1", Text: "Bash", Origin: event.Observed, At: thinkAt(4100), Stage: 1})
		s.Agents["a1"] = &state.Agent{Key: "a1", Label: "general-purpose", Status: state.AgentRunning, StartedAt: thinkAt(4200), Stage: 1}
		s.Order = []string{"a1"}
		return s
	}
	// The line is drawn once its condition has held 200 ms (ActivitySince, which
	// the application notes; plan 22, correction C6): here since 23,000 ms.
	now, since := thinkAt(24000), thinkAt(23000)
	word, _ := expression(1, 24*time.Second)
	activity := []string{"▎", "▎ " + spinGlyph(24*time.Second) + " " + word, "▎ ◇ 1 subagent working", ""}
	tailOf := func(s *state.State) []string {
		rows := rowTexts(bRows(s, UI{Now: now, ActivitySince: since}, 100))
		return rows[max(len(rows)-4, 0):]
	}
	s := bench()
	got := tailOf(s)
	for i, want := range activity {
		if !strings.HasPrefix(got[i], want) {
			t.Errorf("row %d %q, want %q", i, got[i], want)
		}
	}
	if !strings.HasSuffix(got[1], "00:24") {
		t.Errorf("the turn's time not at the right: %q", got[1])
	}
	for _, row := range bRows(s, UI{Now: now, ActivitySince: since}, 100) {
		for _, x := range row {
			if strings.Contains(plainRow(row), word) && x.ref.kind != refDeco {
				t.Errorf("the activity line is content: %q", x.text)
			}
		}
	}
	// A subagent waiting for the owner, its approval not in the queue yet: it
	// is at work too; with its request queued the turn waits and the line is
	// off (below), so it never says who waits (plan 22, the review).
	s.Agents["a2"] = &state.Agent{Key: "a2", Status: state.AgentWaiting, Stage: 1}
	if got := tailOf(s); got[2] != "▎ ◇ 2 subagents working" {
		t.Errorf("two at work: %q", got[2])
	}
	none := func(name string, s *state.State) {
		t.Helper()
		if got := strings.Join(rowTexts(bRows(s, UI{Now: now, ActivitySince: since}, 100)), "\n"); strings.Contains(got, word) || strings.Contains(got, "subagent working") {
			t.Errorf("%s: the activity line in\n%s", name, got)
		}
	}
	s = bench()
	s.Turn.Phase = state.WaitingApproval
	s.Approvals["ap"] = &state.Approval{ID: "ap", Tool: "Bash", ToolID: "t1", Status: state.Pending, RequestedAt: thinkAt(5000)}
	s.Queue = []string{"ap"}
	none("a request waiting", s)
	s = bench()
	s.Conversation = append(s.Conversation, state.Entry{Seq: 4, Role: "assistant", Text: "Here is", Streaming: true, At: thinkAt(6000), Stage: 1})
	none("text streaming", s)
	s = bench()
	s.Turn.Phase, s.Stages[0].Open, s.Stages[0].Outcome = state.Idle, false, "completed"
	none("the turn ended", s)
	s = bench()
	s.Conversation = append(s.Conversation, state.Entry{Seq: 4, Role: "thinking", Streaming: true, At: thinkAt(20000), Stage: 1, Thought: &state.Thought{Start: thinkAt(20000)}})
	none("a card live", s)
	// A card in its exit, drawn live: the line waits for the exit's end, so
	// it never shows for a frame under a closing card.
	s = bench()
	s.Conversation = append(s.Conversation, state.Entry{Seq: 4, Role: "thinking", At: thinkAt(23800), Stage: 1, Thought: &state.Thought{ID: 4, Start: thinkAt(23800), End: thinkAt(23900)}})
	if got := strings.Join(rowTexts(bRows(s, UI{Now: now, ActivitySince: since, Live: map[uint64]bool{4: true}}, 100)), "\n"); strings.Contains(got, word) {
		t.Errorf("a card in its exit: the activity line in\n%s", got)
	}
	if got := strings.Join(rowTexts(bRows(s, UI{Now: thinkAt(24300), ActivitySince: thinkAt(24100), Live: map[uint64]bool{4: true}}, 100)), "\n"); !strings.Contains(got, "subagent working") {
		t.Errorf("after the exit: no activity line in\n%s", got)
	}
	// No entry of the CLI yet: after the rule, under the speaker's label.
	s = bench()
	s.Conversation = s.Conversation[:1]
	got = rowTexts(bRows(s, UI{Now: now, ActivitySince: since}, 100))
	want := []string{"", "▎ YOU  " + clock(thinkAt(0)), "▎ Analyze the repository architecture.", "", strings.Repeat("─", 100), "",
		"▎ CLAUDE  Haiku 5.5  " + clock(thinkAt(0)), "▎ " + spinGlyph(24*time.Second) + " " + word, "▎ ◇ 1 subagent working", ""}
	if len(got) != len(want) {
		t.Fatalf("rows\n%s", strings.Join(got, "\n"))
	}
	for i := range want {
		if !strings.HasPrefix(got[i], want[i]) || want[i] == "" && got[i] != "" {
			t.Errorf("row %d %q, want %q", i, got[i], want[i])
		}
	}
}

// TestThinkingFoldKeepsCard: a card in its exit inside a folded run of calls,
// before the run's cut, is never folded away; once its exit ended, its
// summary row folds with the calls (plan 22, design 1; criterion 2.12).
func TestThinkingFoldKeepsCard(t *testing.T) {
	s := thinkBench(state.Thought{Start: thinkAt(3000), End: thinkAt(23950), Tokens: 50, HasTokens: true})
	th := s.Conversation[1]
	th.Seq, th.Thought.ID = 3, 3
	call := func(seq int, id string) state.Entry {
		s.Tools[id] = &state.ToolCall{ID: id, Name: "Read", Status: state.ToolCompleted, StartedAt: thinkAt(seq * 100), EndedAt: thinkAt(seq*100 + 50), Origin: event.Observed, Stage: 1}
		return state.Entry{Seq: uint64(seq), Role: "tool", MessageID: id, Text: "Read", Origin: event.Observed, At: thinkAt(seq * 100), Stage: 1}
	}
	conv := []state.Entry{s.Conversation[0], call(2, "t1"), th}
	for i := 4; i <= 8; i++ {
		conv = append(conv, call(i, fmt.Sprintf("t%d", i-2)))
	}
	s.Conversation = conv
	live := map[uint64]bool{3: true}
	lk, _ := lookAll(s, UI{Now: thinkAt(24000), Live: live})
	if lk[1].fold == 0 {
		t.Fatalf("the run of six calls does not fold")
	}
	if lk[2].think != thinkExit || lk[2].hidden {
		t.Errorf("in its exit: stage %d, hidden %v; want its exit, shown", lk[2].think, lk[2].hidden)
	}
	lk, _ = lookAll(s, UI{Now: thinkAt(25000), Live: live})
	if lk[2].think != thinkSummary || !lk[2].hidden {
		t.Errorf("after its exit: stage %d, hidden %v; want its summary, folded", lk[2].think, lk[2].hidden)
	}
}

// TestThinkingSizes is criterion 2.14: seven, five and two rows by zone B's
// room, none wider than the zone; drawn at 120x40, then the same instant at
// 80x24, the frame is the 80x24 frame drawn at once. Since plan 24 zone B
// has zone D's rows: at 80x24 its room is 12, the five-row card; the
// two-row card comes with a room under 12, as cardSize is checked.
func TestThinkingSizes(t *testing.T) {
	for room, want := range map[int]int{43: 7, 25: 7, 20: 7, 19: 5, 15: 5, 12: 5, 11: 2, 9: 2} {
		if got := cardSize(room); got != want {
			t.Errorf("room %d: %d rows, want %d", room, got, want)
		}
	}
	for _, c := range []struct {
		th   state.Thought
		name string
	}{
		{state.Thought{Start: thinkAt(7449), Estimate: 650, EstimateFrom: 350, EstimateAt: thinkAt(10610)}, "mode B"},
		{state.Thought{Start: thinkAt(7449)}, "mode C"},
		{state.Thought{Start: thinkAt(7449), Inferred: true}, "a Codex step"},
	} {
		s := thinkBench(c.th)
		for _, size := range []struct{ w, h, rows int }{{120, 40, 7}, {200, 60, 7}, {100, 30, 5}, {80, 24, 5}} {
			ui := UI{Width: size.w, Height: size.h, Now: thinkAt(10700), Focus: "B"}
			rows := zoneBText(s, ui)
			got := len(boxIn(rows))
			if size.rows == 2 {
				got = 0
				for i, r := range rows {
					if strings.Contains(r, "Connecting dots...") && strings.Contains(rows[i+1], "━") {
						got = 2
					}
				}
			}
			if got != size.rows {
				t.Errorf("%s at %dx%d: %d rows, want %d:\n%s", c.name, size.w, size.h, got, size.rows, strings.Join(rows, "\n"))
			}
			// Every row of the card within its block: none cut ("…").
			for _, l := range card(s, ui, bInner(size.w), size.rows) {
				if l.width() > cardContent(cardAI, bInner(size.w)) || strings.Contains(plainRow(l), "…") && c.name != "a Codex step" {
					t.Errorf("%s at %dx%d: a row %d wide for %d: %q", c.name, size.w, size.h, l.width(), cardContent(cardAI, bInner(size.w)), plainRow(l))
				}
			}
		}
		// The resize: drawn at 120x40, then 80x24, with the cache.
		wc := NewWrapCache()
		big := UI{Width: 120, Height: 40, Now: thinkAt(10700), Focus: "B", Wrap: wc}
		Draw(s, big)
		small := big
		small.Width, small.Height = 80, 24
		got := Draw(s, small).String()
		small.Wrap = nil
		if want := Draw(s, small).String(); got != want {
			t.Errorf("%s: 80x24 after 120x40 differs from 80x24 drawn at once", c.name)
		}
	}
}

// TestThinkingScroll is criterion 2.15 at the render level: 50 frames of a
// live card with the view held up keep the view where it is.
func TestThinkingScroll(t *testing.T) {
	s := thinkBench(state.Thought{Start: thinkAt(60000)})
	var history []state.Entry
	for i := range 30 {
		history = append(history, state.Entry{Seq: uint64(100 + i), Role: "user", Text: fmt.Sprintf("question %d", i), At: thinkAt(0)},
			state.Entry{Seq: uint64(200 + i), Role: "assistant", Text: fmt.Sprintf("answer %d", i), At: thinkAt(0)})
	}
	s.Conversation = append(history, s.Conversation...)
	ui := UI{Width: 120, Height: 40, Focus: "B", ConvHold: true, ConvTop: 10, ConvSeen: len(s.Conversation), Wrap: NewWrapCache()}
	var first string
	for f := range 50 {
		ui.Now = thinkAt(60300 + 50*f)
		sc := Draw(s, ui)
		if sc.Conv.Top != 10 || !sc.Conv.Held {
			t.Fatalf("frame %d: the view at %d, held %v", f, sc.Conv.Top, sc.Conv.Held)
		}
		if top := strings.Join(zoneBText(s, ui)[:5], "\n"); f == 0 {
			first = top
		} else if top != first {
			t.Errorf("frame %d: the rows in view moved:\n%s", f, top)
		}
	}
}

// TestAnimating is plan 22, design 6's frame rate: true while a frame 50 ms
// later would differ, false otherwise.
func TestAnimating(t *testing.T) {
	live := map[uint64]bool{2: true}
	th := state.Thought{Start: thinkAt(7449), Estimate: 650, EstimateFrom: 350, EstimateAt: thinkAt(10610)}
	for _, c := range []struct {
		name  string
		th    state.Thought
		turn  state.TurnPhase
		ms    int
		live  map[uint64]bool
		since int // when the activity line's condition began, ms; 0 for none (plan 22, correction C6)
		want  bool
	}{
		{"a live card", th, state.Running, 15000, nil, 0, true},
		{"a live card, a request waiting", th, state.WaitingApproval, 15000, nil, 0, false},
		{"its entrance, a request waiting", state.Thought{Start: thinkAt(14900)}, state.WaitingApproval, 15000, nil, 0, true},
		{"a value growing, a request waiting", th, state.WaitingApproval, 10900, nil, 0, true},
		{"its edge settling, a request waiting", th, state.WaitingApproval, 11300, nil, 0, true},
		{"settled, a request waiting", th, state.WaitingApproval, 11310, nil, 0, false},
		{"a new scale lit, a request waiting", state.Thought{Start: thinkAt(7449), Estimate: 2200, EstimateFrom: 1950, EstimateAt: thinkAt(10000)}, state.WaitingApproval, 10900, nil, 0, true},
		{"an exit", state.Thought{Start: thinkAt(7449), End: thinkAt(14800)}, state.Running, 15000, live, 0, true},
		{"an exit's end", state.Thought{Start: thinkAt(7449), End: thinkAt(14700)}, state.Idle, 15000, live, 0, false},
		{"an exit not drawn live", state.Thought{Start: thinkAt(7449), End: thinkAt(14800)}, state.Idle, 15000, nil, 0, false},
		{"a count reconciled", state.Thought{Start: thinkAt(7449), End: thinkAt(14000), HasTokens: true, TokensAt: thinkAt(14600)}, state.Idle, 15000, nil, 0, true},
		{"a count reconciled, 600 ms on", state.Thought{Start: thinkAt(7449), End: thinkAt(14000), HasTokens: true, TokensAt: thinkAt(14400)}, state.Idle, 15000, nil, 0, false},
		{"the activity line", state.Thought{Start: thinkAt(1000), End: thinkAt(2000)}, state.Running, 9000, nil, 2000, false},
		{"the activity line's next expression", state.Thought{Start: thinkAt(1000), End: thinkAt(2000)}, state.Running, 7960, nil, 2000, true},
		{"the activity line's new expression", state.Thought{Start: thinkAt(1000), End: thinkAt(2000)}, state.Running, 8150, nil, 2000, true},
		// The line's frames wait for the line: not before its condition has held
		// 200 ms, and never without the application having noted it began.
		{"the next expression, the line not due", state.Thought{Start: thinkAt(1000), End: thinkAt(2000)}, state.Running, 7960, nil, 7811, false},
		{"the next expression, the line due", state.Thought{Start: thinkAt(1000), End: thinkAt(2000)}, state.Running, 7960, nil, 7760, true},
		{"the next expression, no beginning noted", state.Thought{Start: thinkAt(1000), End: thinkAt(2000)}, state.Running, 7960, nil, 0, false},
		{"idle", state.Thought{Start: thinkAt(1000), End: thinkAt(2000)}, state.Idle, 8150, nil, 2000, false},
	} {
		s := thinkBench(c.th)
		s.Turn.Phase = c.turn
		if c.turn == state.WaitingApproval {
			s.Approvals["ap"] = &state.Approval{ID: "ap", Status: state.Pending, RequestedAt: thinkAt(9000)}
			s.Queue = []string{"ap"}
		}
		ui := UI{Now: thinkAt(c.ms), Live: c.live}
		if c.since > 0 {
			ui.ActivitySince = thinkAt(c.since)
		}
		if got := Animating(s, ui); got != c.want {
			t.Errorf("%s: %v, want %v", c.name, got, c.want)
		}
	}
	// While a request waits, the card stands still: the same frame later.
	s := thinkBench(state.Thought{Start: thinkAt(7449)})
	s.Turn.Phase = state.WaitingApproval
	s.Approvals["ap"] = &state.Approval{ID: "ap", Status: state.Pending, RequestedAt: thinkAt(9000)}
	s.Queue = []string{"ap"}
	a, b := card(s, UI{Now: thinkAt(9500)}, 90, 7), card(s, UI{Now: thinkAt(9850)}, 90, 7)
	if styled(a) != styled(b) || !strings.Contains(plainRow(a[1]), "◌") {
		t.Errorf("the card moves while a request waits:\n%s\n%s", styled(a), styled(b))
	}
}

// TestThinkingCache is plan 22 with test 11's row cache: with a card, an exit
// and details, the frame drawn with the cache is the frame drawn without it.
func TestThinkingCache(t *testing.T) {
	s := thinkBench(state.Thought{Start: thinkAt(7449), Estimate: 650, EstimateFrom: 350, EstimateAt: thinkAt(10610)})
	for i := range 20 {
		s.Conversation = append([]state.Entry{{Seq: uint64(100 + i), Role: "assistant", Text: strings.Repeat("an answer ", 3+i), At: thinkAt(0)}}, s.Conversation...)
	}
	n := len(s.Conversation) - 1
	sizes := [][2]int{{120, 40}, {80, 24}, {200, 60}}
	caches := []*WrapCache{NewWrapCache(), NewWrapCache(), NewWrapCache()} // one a size: frame after frame
	check := func(name string, ui UI) {
		t.Helper()
		for i, size := range sizes {
			ui.Width, ui.Height, ui.Wrap = size[0], size[1], caches[i]
			got := Draw(s, ui)
			ui.Wrap = nil
			if want := Draw(s, ui); got.String() != want.String() || got.Conv.Rows != want.Conv.Rows {
				t.Errorf("%s at %dx%d: the cached frame differs (%d rows, want %d)", name, size[0], size[1], got.Conv.Rows, want.Conv.Rows)
			}
		}
	}
	live := map[uint64]bool{2: true}
	for ms := 7449; ms < 7800; ms += 35 {
		check(fmt.Sprintf("entrance %d", ms), UI{Now: thinkAt(ms), Live: live})
	}
	check("steady", UI{Now: thinkAt(10700), Live: live})
	s.Conversation[n].Thought.End, s.Conversation[n].Streaming = thinkAt(10800), false
	for ms := 10800; ms <= 11200; ms += 50 {
		check(fmt.Sprintf("exit %d", ms), UI{Now: thinkAt(ms), Live: live})
	}
	key := ThoughtKey(s.Conversation[n])
	check("details", UI{Now: thinkAt(12000), Unfolded: map[string]bool{key: true}})
	s.Conversation[n].Thought.HasTokens, s.Conversation[n].Thought.Tokens = true, 2852
	check("details, the count come", UI{Now: thinkAt(12100), Unfolded: map[string]bool{key: true}})
	check("folded", UI{Now: thinkAt(12200)})
}

// TestThinkingBadge is plan 22, design 10, and the badge: zone B's badge says
// Thinking while Claude Code reports a phase, the placeholder that the CLI
// works on the request.
func TestThinkingBadge(t *testing.T) {
	s := thinkBench(state.Thought{Start: thinkAt(7449)})
	ui := UI{Width: 120, Height: 40, Now: thinkAt(8000)}
	if got := plainRow(turnBadge(s, ui)); got != spinner[0]+" Thinking" {
		t.Errorf("badge %q", got)
	}
	if got := placeholder(s); got != "Claude Code is processing your request…" {
		t.Errorf("placeholder %q", got)
	}
	if frame := texts(Draw(s, ui).lines); !strings.Contains(frame, ">  Claude Code is processing your request…") {
		t.Errorf("the input does not say it:\n%s", frame)
	}
	s = thinkBench(state.Thought{Start: thinkAt(7449), Inferred: true})
	if got := plainRow(turnBadge(s, ui)); got != spinner[0]+" Running" {
		t.Errorf("a Codex step: badge %q", got)
	}
	if got := placeholder(s); got != "Codex is processing your request…" {
		t.Errorf("placeholder %q", got)
	}
	s.Turn.Phase = state.Idle
	if got := placeholder(s); got != "Ask Codex anything…" {
		t.Errorf("idle: placeholder %q", got)
	}
}

// TestThinkingPalette is plan 22's colours (design 2), each with its
// 16-colour fallback.
func TestThinkingPalette(t *testing.T) {
	defer UsePalette(pickTrue)
	for _, c := range []struct {
		c        colour
		tc, ansi string
	}{
		{cThinkBorder, "#7C6BB3", "5"}, {cThinkFill, "#A78BFA", "5"}, {cThinkEdge, "#EDE9FE", "15"},
		{cThinkMid, "#C4B5FD", "13"}, {cThinkDim, "#4C4470", "8"}, {cThinkTrack, "#2E2A47", "8"},
	} {
		if c.c.tc != c.tc || c.c.ansi != c.ansi {
			t.Errorf("%v, want %s and %s", c.c, c.tc, c.ansi)
		}
	}
	sts := map[*lipgloss.Style]string{stThinkBorder: "5", stThinkFill: "5", stThinkEdge: "15", stThinkMid: "13", stThinkDim: "8",
		stThinkTrack: "8", stThinkWord: "13", stThinkWordDim: "8", stThinkLit: "15", stBrightBold: "15"}
	UsePalette(pickANSI)
	for st, want := range sts {
		if st.GetForeground() != lipgloss.Color(want) {
			t.Errorf("16 colours: %q is %v, want %s", st.Render("x"), st.GetForeground(), want)
		}
	}
	UsePalette(pickTrue)
	if !stThinkWord.GetBold() || stThinkWord.GetForeground() != lipgloss.Color("#C4B5FD") || !stThinkLit.GetBold() {
		t.Error("the expression is not lavender bold, or a lit count not bold")
	}
}

// TestThinkingAllocs is amendment 18, 4.3: a frame's card costs a few small
// allocations, and a conversation without thinking what it did.
func TestThinkingAllocs(t *testing.T) {
	s := thinkBench(state.Thought{Start: thinkAt(7449), Estimate: 650, EstimateFrom: 350, EstimateAt: thinkAt(10610)})
	ui := UI{Now: thinkAt(12000)}
	if n := testing.AllocsPerRun(50, func() { card(s, ui, 90, 7) }); n > 120 {
		t.Errorf("a card: %v allocations", n)
	}
}

// The review of plan 22's change, each finding's fix: the summary row in its
// place for the exit's last 100 ms, so it never moves when the exit ends; a
// phase ended by its turn said so, a Codex one "up to"; the 2-row card's
// count in its short form when the long one would be cut; a Codex step that
// reasoned "Thinking" as it closes; the placeholder for any turn under way;
// the "new" badge without the steps that left no row; the phases' keys by
// their ID, two sessions' equal Seq apart.
func TestThinkingReviewFixes(t *testing.T) {
	// No jump: the summary row's place at End+250 is its place at End+350.
	s := thinkBench(state.Thought{Start: thinkAt(7449), End: thinkAt(10265), Tokens: 719, HasTokens: true, TokensAt: thinkAt(10300)})
	row := func(ms int) int {
		for i, r := range rowTexts(bRows(s, UI{Now: thinkAt(ms), Live: map[uint64]bool{2: true}}, 90)) {
			if strings.Contains(r, "Thought for 2.8s") {
				return i
			}
		}
		return -1
	}
	if a, b := row(10515), row(10615); a < 0 || a != b {
		t.Errorf("the summary row at End+250 on row %d, at End+350 on row %d: it moved", a, b)
	}
	// A Codex phase interrupted: its time an upper bound still.
	codex := &state.Thought{Start: thinkAt(1000), End: thinkAt(4000), Inferred: true, Confirmed: true, Outcome: "interrupted", EndInferred: true}
	if got, _, _, _ := thoughtSummary(codex); got != "Thinking interrupted after up to 3.0s" {
		t.Errorf("a Codex phase interrupted: %q", got)
	}
	if d := thoughtDetails(codex); !strings.Contains(d[0], "when its turn ended (inferred)") || d[1] != "no exact count reported" {
		t.Errorf("its details: %q", d)
	}
	claude := &state.Thought{Start: thinkAt(1000), End: thinkAt(4000), Confirmed: true, EndInferred: true}
	if d := thoughtDetails(claude); !strings.Contains(d[0], "whose stop was not reported") || d[1] != "no exact count reported" {
		t.Errorf("a Claude Code block with no stop: %q", d)
	}
	ok := &state.Thought{Start: thinkAt(1000), End: thinkAt(4000), Confirmed: true}
	if d := thoughtDetails(ok); !strings.Contains(d[0], "a thinking block, observed") || d[1] != "exact count not reported yet" {
		t.Errorf("a block that stopped, its count to come: %q", d)
	}
	// The 2-row card at 80 columns: no count cut.
	for _, th := range []state.Thought{{Start: thinkAt(1000), Inferred: true}, {Start: thinkAt(1000), Confirmed: true, Estimate: 2450, EstimateAt: thinkAt(1500)}} {
		s := thinkBench(th)
		rows := card(s, UI{Now: thinkAt(3000)}, bInner(80), 2)
		if r := plainRow(rows[0]); strings.Contains(r, "…") || !strings.Contains(r, "00:02") {
			t.Errorf("the 2-row card's first row cut: %q", r)
		}
	}
	// A Codex step that reasoned, closing: "Thinking", no longer "Model working".
	s = thinkBench(state.Thought{Start: thinkAt(1000), End: thinkAt(4000), Inferred: true, Confirmed: true})
	if rows := card(s, UI{Now: thinkAt(4050), Live: map[uint64]bool{2: true}}, 90, 5); !strings.Contains(plainRow(rows[2]), "Thinking · reasoning tokens reported when the step ends") {
		t.Errorf("a Codex step that reasoned, closing: %q", plainRow(rows[2]))
	}
	// The placeholder while a request waits, and while an interrupt is asked for.
	for _, p := range []state.TurnPhase{state.WaitingApproval, state.Interrupting} {
		s.Session.Phase, s.Turn.Phase = state.Ready, p
		if got := placeholder(s); got != s.Context.ProviderName+" is processing your request…" {
			t.Errorf("%s: placeholder %q", p, got)
		}
	}
	// "↓ N new" leaves out a step that did not reason.
	s = thinkBench(state.Thought{Start: thinkAt(1000), End: thinkAt(2000), Inferred: true, Dropped: true})
	s.Conversation = append(s.Conversation, state.Entry{Seq: 3, Role: "assistant", Text: "done", At: thinkAt(2100), Stage: 1})
	if n := NewShown(s, 1); n != 1 {
		t.Errorf("%d new entries shown, want 1 (the dropped step none)", n)
	}
	// Two phases with the same Seq, from two sessions, keep their own keys.
	a := state.Entry{Seq: 5, Role: "thinking", Thought: &state.Thought{ID: 1}}
	b := state.Entry{Seq: 5, Role: "thinking", Thought: &state.Thought{ID: 9}}
	if ThoughtKey(a) == ThoughtKey(b) {
		t.Errorf("two phases with one Seq share the key %q", ThoughtKey(a))
	}
}
