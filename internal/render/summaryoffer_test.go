package render

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"

	"github.com/Evengelius/agent-command-center/internal/event"
	"github.com/Evengelius/agent-command-center/internal/fixture"
	"github.com/Evengelius/agent-command-center/internal/provider/claude"
	"github.com/Evengelius/agent-command-center/internal/provider/codex"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// Plan 70, criterion 70.2: the one-time summary suggestion after a resume
// from the home screen, drawn in zone B after the resumed note as the
// owner's card (brief amendment 73).

// ownerCard is the owner's design, row for row (amendment 73).
var ownerCard = []string{
	"╭─ SESSION RESUMED · SUGGESTION ──────────────────────╮",
	"│                                                     │",
	"│  Want a summary of your earlier work?               │",
	"│                                                     │",
	"│  [ Prepare summary ]     [ Not now ]                │",
	"│                                                     │",
	"│  -------------------------------------------------  │",
	"│                                                     │",
	"│  The request will be added to your draft.           │",
	"│  Press Enter to send it.                            │",
	"│                                                     │",
	"╰─────────────────────────────────────────────────────╯",
}

// offerState is a session resumed from the home screen once its CLI named
// the session asked for: the resumed note, and the offer standing (plan
// 70, design D1).
func offerState(t *testing.T, cli, name, id string) *state.State {
	t.Helper()
	s := state.New()
	s.Context.Provider, s.Context.ProviderName, s.Context.Project = cli, name, `C:\work\demo-project`
	s.Startup.Phase = state.Starting
	s.Session.Resumed = id
	for i, ev := range []struct {
		k event.Kind
		p any
	}{
		{event.SessionSpawned, event.Spawned{}},
		{event.SessionReady, event.Ready{}},
		{event.SessionStarted, event.Started{SessionID: id, Model: "a-model"}},
	} {
		s.Apply(event.Event{Seq: uint64(i + 1), At: fixture.Epoch, Provider: cli, Kind: ev.k, Origin: event.Observed, Payload: ev.p})
	}
	if !s.Session.SummaryOffered || s.Session.Phase != state.Ready || len(s.Conversation) != 1 {
		t.Fatalf("offered %v, session %q, %d entries", s.Session.SummaryOffered, s.Session.Phase, len(s.Conversation))
	}
	return s
}

var offerCLIs = []struct{ cli, name, id string }{
	{claude.ProviderID, "Claude Code", "21f3b775-162f-47e1-bb16-aa91f8a808b9"},
	{codex.ProviderID, "OpenAI Codex", "01a18936-726b-71c2-bda9-13b405475191"},
}

// frameRow is the first row of the frame holding text after row from, -1
// for none, and the column it starts at.
func frameRow(rows []string, from int, text string) (int, int) {
	for y := max(from, 0); y < len(rows); y++ {
		if i := strings.Index(rows[y], text); i >= 0 {
			return y, textWidth(rows[y][:i])
		}
	}
	return -1, -1
}

// cellText is the text of the frame's row y from cell x, n cells wide.
func cellText(sc *Screen, x, y, n int) string {
	var b strings.Builder
	cx := 0
	for _, sp := range sc.lines[y] {
		for _, r := range sp.text {
			rw := textWidth(string(r))
			if cx >= x && cx+rw <= x+n {
				b.WriteRune(r)
			}
			cx += rw
		}
	}
	return b.String()
}

// cardKept is the owner's card with the blank rows a short zone B keeps:
// all five; 3, without the two around the rule; 1, without the two inside
// the edges too; 0, without the one under the question too.
func cardKept(keep int) []string {
	drop := map[int]bool{5: keep < 5, 7: keep < 5, 1: keep < 3, 10: keep < 3, 3: keep < 1}
	var out []string
	for i, r := range ownerCard {
		if !drop[i] {
			out = append(out, r)
		}
	}
	return out
}

// TestSummaryOfferDrawn is the suggestion at 80x24, 90x30, 120x40 and
// 200x60 for both CLIs: the owner's card cell for cell at the left of zone
// B's inside, its five blank rows dropped at 80x24, where B's 11 rows hold
// it whole with the resumed note above only so; the note in view, a blank
// row between it and the card and one after the card, every row within
// the screen, the buttons' cells a click's targets and no other cell of
// the card.
func TestSummaryOfferDrawn(t *testing.T) {
	for _, r := range ownerCard {
		if textWidth(r) != summaryWidth {
			t.Fatalf("the design's row %q is %d cells", r, textWidth(r))
		}
	}
	for _, c := range offerCLIs {
		s := offerState(t, c.cli, c.name, c.id)
		for _, sz := range []struct{ w, h, room, keep int }{{80, 24, 11, 0}, {90, 30, 17, 5}, {120, 40, 27, 5}, {200, 60, 47, 5}} {
			w, h := sz.w, sz.h
			sc := Draw(s, UI{Width: w, Height: h, Focus: "B"})
			rows := strings.Split(strings.TrimSuffix(texts(sc.lines), "\n"), "\n")
			if len(rows) != h || sc.Conv.Room != sz.room {
				t.Fatalf("%s %dx%d: %d rows, B's room %d, want %d", c.name, w, h, len(rows), sc.Conv.Room, sz.room)
			}
			for y, r := range rows {
				if textWidth(r) != w {
					t.Errorf("%s %dx%d: row %d is %d cells: %q", c.name, w, h, y, textWidth(r), r)
				}
			}
			card := cardKept(sz.keep)
			_, bx := frameRow(rows, 0, "╭─ B · CONVERSATION")
			note, _ := frameRow(rows, 0, "Resumed "+c.name)
			top, x := frameRow(rows, note, card[0])
			if bx < 0 || note < sc.Conv.Y || top <= note || x != bx+2 || sc.Conv.W < summaryWidth {
				t.Fatalf("%s %dx%d: B at %d, inside %d wide from row %d, the note on row %d, the card at (%d, %d):\n%s", c.name, w, h, bx, sc.Conv.W, sc.Conv.Y, note, x, top, strings.Join(rows, "\n"))
			}
			for i, want := range card {
				if got := cellText(sc, x, top+i, summaryWidth); got != want {
					t.Errorf("%s %dx%d: the card's row %d is\n%q, want\n%q", c.name, w, h, i, got, want)
				}
			}
			for _, y := range []int{top - 1, top + len(card)} {
				if blank := cellText(sc, x, y, sc.Conv.W); strings.TrimSpace(blank) != "" || textWidth(blank) != sc.Conv.W {
					t.Errorf("%s %dx%d: row %d around the card is not blank: %q", c.name, w, h, y, blank)
				}
			}
			if !sc.Conv.Summary {
				t.Errorf("%s %dx%d: ConvView.Summary false with the buttons in view", c.name, w, h)
			}
			b, _ := frameRow(rows, top, summaryPrepared)
			bx0 := x + 3 // Prepare summary's first cell
			notNow := bx0 + textWidth(summaryPrepared+summaryGap)
			for cx := x; cx < x+summaryWidth; cx++ {
				for y := top; y < top+len(card); y++ {
					p, item := sc.ItemAt(cx, y)
					switch {
					case y == b && cx >= bx0 && cx < bx0+textWidth(summaryPrepared):
						if p != SummaryPanel || item != SummaryPrepare {
							t.Errorf("%s %dx%d: cell (%d, %d) is (%q, %d), want Prepare summary", c.name, w, h, cx, y, p, item)
						}
					case y == b && cx >= notNow && cx < notNow+textWidth(summaryNotNow):
						if p != SummaryPanel || item != SummaryNotNow {
							t.Errorf("%s %dx%d: cell (%d, %d) is (%q, %d), want Not now", c.name, w, h, cx, y, p, item)
						}
					case p == SummaryPanel:
						t.Errorf("%s %dx%d: cell (%d, %d) %q is a button's", c.name, w, h, cx, y, cellText(sc, cx, y, 1))
					}
				}
			}
		}
	}
}

// TestSummaryOfferRows is the card at every width from 1 to 80: none where
// it has no cell inside its margins; else min(w, 55) wide, its edges whole,
// its margins of two cells, the title whole or SUGGESTION alone or none,
// the buttons side by side where they fit and one above the other where
// they do not, the rule as wide as the texts, every span decoration (never
// copied), the texts wrapped and kept whole, and the blank row after it.
func TestSummaryOfferRows(t *testing.T) {
	s := offerState(t, offerCLIs[0].cli, offerCLIs[0].name, offerCLIs[0].id)
	side := textWidth(summaryPrepared + summaryGap + summaryNotNow)
	for w := 1; w <= 80; w++ {
		rows := summaryRows(s, UI{Summary: SummaryFocus{On: true}}, w, 40)
		cw := min(w, 55)
		inner := cw - 6
		if inner < 1 {
			if rows != nil {
				t.Errorf("width %d: %d rows, want none", w, len(rows))
			}
			continue
		}
		if len(rows) < 3 || rows[len(rows)-1].solid() {
			t.Fatalf("width %d: %d rows, the last not blank", w, len(rows))
		}
		card := rows[:len(rows)-1]
		var all []string
		buttons, rules := 0, 0
		for i, r := range card {
			l := r.draw()
			text := plainRow(l)
			if l.width() != cw {
				t.Errorf("width %d: row %d is %d cells, want %d: %q", w, i, l.width(), cw, text)
			}
			for _, sp := range l {
				if sp.text != "" && sp.ref.kind != refDeco {
					t.Errorf("width %d: span %q is not decoration", w, sp.text)
				}
			}
			cells := []rune(text)
			switch i {
			case 0:
				want := "╭" + strings.Repeat("─", cw-2) + "╮"
				switch {
				case cw >= 34:
					want = "╭─ " + summaryTitle + " " + strings.Repeat("─", cw-5-textWidth(summaryTitle)) + "╮"
				case cw >= 16:
					want = "╭─ SUGGESTION " + strings.Repeat("─", cw-15) + "╮"
				}
				if text != want {
					t.Errorf("width %d: the top edge %q, want %q", w, text, want)
				}
				continue
			case len(card) - 1:
				if text != "╰"+strings.Repeat("─", cw-2)+"╯" {
					t.Errorf("width %d: the bottom edge %q", w, text)
				}
				continue
			}
			if string(cells[:3]) != "│  " || string(cells[len(cells)-3:]) != "  │" {
				t.Errorf("width %d: row %d's edges and margins %q", w, i, text)
			}
			if r.summary {
				buttons++
			}
			inside := string(cells[3 : len(cells)-3])
			if inside == strings.Repeat("-", inner) {
				rules++
				continue
			}
			all = append(all, strings.TrimSpace(inside))
		}
		want := 1
		if inner < side {
			want = 2
		}
		if buttons != want || rules != 1 {
			t.Errorf("width %d: %d buttons rows, want %d; %d rules", w, buttons, want, rules)
		}
		if inner >= textWidth("summary") { // the longest word: wrapping keeps every word whole
			text := strings.Join(strings.Fields(strings.Join(all, " ")), " ")
			for _, part := range []string{summaryQuestion, summaryAdded, summaryEnter} {
				if !strings.Contains(text, part) {
					t.Errorf("width %d: %q not whole in %q", w, part, text)
				}
			}
		}
		if inner >= textWidth(summaryAdded) && len(rows) != len(ownerCard)+1 { // the widest text: the owner's card, then the blank row
			t.Errorf("width %d: %d rows, want %d", w, len(rows), len(ownerCard)+1)
		}
	}
	if sessionLines(s) != nil {
		t.Errorf("the suggestion among the session's lines: %q", sessionLines(s))
	}
}

// TestSummaryOfferHeights is the card in every room from 1 to 30 rows:
// the owner's design from 17, where B holds it, the blank row after it and
// the resumed note's four rows above it; from 15 without the blank rows
// around the rule; from 13 without those inside the edges too; below,
// without the one under the question too; its edges, title, texts and rule
// always whole, and the blank row after it.
func TestSummaryOfferHeights(t *testing.T) {
	s := offerState(t, offerCLIs[1].cli, offerCLIs[1].name, offerCLIs[1].id)
	for room := 1; room <= 30; room++ {
		keep := 0
		switch {
		case room >= 17:
			keep = 5
		case room >= 15:
			keep = 3
		case room >= 13:
			keep = 1
		}
		want := cardKept(keep)
		rows := summaryRows(s, UI{}, 76, room)
		if len(rows) != len(want)+1 || rows[len(rows)-1].solid() {
			t.Errorf("room %d: %d rows, want %d and a blank one", room, len(rows), len(want)+1)
			continue
		}
		for i, r := range want {
			if got := plainRow(rows[i].draw()); got != r {
				t.Errorf("room %d: row %d is\n%q, want\n%q", room, i, got, r)
			}
		}
	}
}

// TestSummaryOfferFollows is the suggestion at the end of a conversation
// taller than zone B: a turn the CLI opened itself before the owner's
// first message, its long answer above the suggestion; B follows its end,
// the card whole, its bottom edge above the blank row that ends the
// conversation, at every size.
func TestSummaryOfferFollows(t *testing.T) {
	s := offerState(t, offerCLIs[0].cli, offerCLIs[0].name, offerCLIs[0].id)
	var lines []string
	for i := 1; i <= 60; i++ {
		lines = append(lines, "line "+strings.Repeat("x", i%7+1))
	}
	for i, ev := range []struct {
		k event.Kind
		p any
	}{
		{event.MessageMarker, event.Marker{Name: "message_start"}},
		{event.MessageCompleted, event.Completed{MessageID: "m1", Text: strings.Join(lines, "\n")}},
		{event.TurnCompleted, event.TurnEnd{Outcome: "completed"}},
	} {
		s.Apply(event.Event{Seq: uint64(10 + i), At: fixture.Epoch, Provider: s.Context.Provider, Kind: ev.k, Origin: event.Observed, Payload: ev.p})
	}
	if !SummaryShown(s) || len(s.Conversation) != 2 {
		t.Fatalf("shown %v, %d entries", SummaryShown(s), len(s.Conversation))
	}
	for _, sz := range [][2]int{{80, 24}, {90, 30}, {120, 40}, {200, 60}} {
		sc := Draw(s, UI{Width: sz[0], Height: sz[1], Focus: "B"})
		rows := strings.Split(strings.TrimSuffix(texts(sc.lines), "\n"), "\n")
		first, _ := frameRow(rows, 0, ownerCard[0])
		last, _ := frameRow(rows, 0, ownerCard[len(ownerCard)-1])
		note, _ := frameRow(rows, 0, summaryEnter)
		edge := 2 // the last note, the blank row inside the bottom edge, the edge
		if sc.Conv.Room < 15 {
			edge = 1
		}
		if first < sc.Conv.Y || last < 0 || !sc.Conv.Summary || sc.Conv.Top+sc.Conv.Room != sc.Conv.Rows || last != sc.Conv.Y+sc.Conv.Room-2 || note != last-edge {
			t.Errorf("%dx%d: the card from row %d to %d, the last note on %d, B's rows %d to %d, top %d of %d, buttons in view %v", sz[0], sz[1], first, last, note, sc.Conv.Y, sc.Conv.Y+sc.Conv.Room-1, sc.Conv.Top, sc.Conv.Rows, sc.Conv.Summary)
		}
	}
}

// TestSummaryOfferFocus is the buttons' look, on both palettes: in the
// title colour without the focus; with it, the selected one bright and
// bold, its label underlined, the other subdued, not bold; never a ground
// or inverse anywhere in the card, the focused button included; the
// frame in the thinking card's violet, its title in the title colour.
func TestSummaryOfferFocus(t *testing.T) {
	defer UsePalette(pickTrue)
	s := offerState(t, offerCLIs[1].cli, offerCLIs[1].name, offerCLIs[1].id)
	for _, pick := range []lipgloss.CompleteFunc{pickTrue, pickANSI} {
		UsePalette(pick)
		for _, c := range []struct {
			name            string
			focus           SummaryFocus
			prepare, notNow [2]*lipgloss.Style // brackets, label
		}{
			{"no focus", SummaryFocus{}, [2]*lipgloss.Style{stTitle, stTitle}, [2]*lipgloss.Style{stTitle, stTitle}},
			{"focus", SummaryFocus{On: true}, [2]*lipgloss.Style{stBrightBold, stSummaryFocus}, [2]*lipgloss.Style{stMuted, stMuted}},
			{"Not now", SummaryFocus{On: true, NotNow: true}, [2]*lipgloss.Style{stMuted, stMuted}, [2]*lipgloss.Style{stBrightBold, stSummaryFocus}},
			{"Not now kept, no focus", SummaryFocus{NotNow: true}, [2]*lipgloss.Style{stTitle, stTitle}, [2]*lipgloss.Style{stTitle, stTitle}},
		} {
			got := map[int][]*lipgloss.Style{}
			for _, r := range summaryRows(s, UI{Summary: c.focus}, 80, 40) {
				l := r.draw()
				for _, sp := range l {
					if sp.st == nil {
						continue
					}
					if _, none := sp.st.GetBackground().(lipgloss.NoColor); !none || sp.st.GetReverse() {
						t.Errorf("sixteen %v, %s: %q has a ground %v or is inverse", sixteen, c.name, sp.text, sp.st.GetBackground())
					}
					if sp.ref.panel == SummaryPanel {
						got[sp.ref.item] = append(got[sp.ref.item], sp.st)
					}
				}
				if out := l.String(); strings.Contains(out, "48;") || strings.Contains(out, "\x1b[7m") {
					t.Errorf("sixteen %v, %s: a ground or inverse written: %q", sixteen, c.name, out)
				}
			}
			for item, want := range map[int][2]*lipgloss.Style{SummaryPrepare: c.prepare, SummaryNotNow: c.notNow} {
				if sts := got[item]; len(sts) != 3 || sts[0] != want[0] || sts[1] != want[1] || sts[2] != want[0] {
					t.Errorf("sixteen %v, %s: button %d's styles %p, want %p, %p, %p", sixteen, c.name, item, sts, want[0], want[1], want[0])
				}
			}
		}
		if !stSummaryFocus.GetBold() || !stSummaryFocus.GetUnderline() || !stBrightBold.GetBold() || stBrightBold.GetUnderline() || stMuted.GetBold() || stMuted.GetUnderline() || !stTitle.GetBold() || stTitle.GetUnderline() {
			t.Errorf("sixteen %v: the focused label not bold and underlined, or another button's text underlined, or the unfocused bold", sixteen)
		}
		want := map[bool][2]string{false: {"#7C6BB3", "#F9FAFB"}, true: {"5", "15"}}[sixteen]
		if stSummaryFrame != stThinkBorder || stSummaryFrame.GetForeground() != lipgloss.Color(want[0]) || stSummaryFocus.GetForeground() != lipgloss.Color(want[1]) {
			t.Errorf("sixteen %v: the frame %v, the focused label %v; want %s, %s", sixteen, stSummaryFrame.GetForeground(), stSummaryFocus.GetForeground(), want[0], want[1])
		}
	}
	rows := summaryRows(s, UI{}, 80, 40)
	top, bottom := rows[0].draw(), rows[len(rows)-2].draw()
	if len(top) != 3 || top[0].st != stSummaryFrame || top[1].text != summaryTitle || top[1].st != stTitle || top[2].st != stSummaryFrame || len(bottom) != 1 || bottom[0].st != stSummaryFrame {
		t.Errorf("the edges' spans %q, %q", plainRow(top), plainRow(bottom))
	}
	// The question bold in the text colour, the rule in the frames' grey,
	// the notes grey, the side edges and their margins in the frame's
	// violet.
	n := 0
	for _, r := range rows[1 : len(rows)-2] {
		for _, sp := range r.draw() {
			want := map[string]*lipgloss.Style{summaryQuestion: stBold, strings.Repeat("-", 49): stFrame, summaryAdded: stGrey, summaryEnter: stGrey,
				"│" + summaryPad: stSummaryFrame, summaryPad + "│": stSummaryFrame}[sp.text]
			if want != nil {
				n++
			}
			if want != nil && sp.st != want {
				t.Errorf("%q in the wrong style", sp.text)
			}
		}
	}
	if n != 4+2*10 { // the four texts and the ten inner rows' two edges
		t.Errorf("%d spans checked, want 24", n)
	}
}

// TestSummaryOfferHidden is the suggestion not drawn, the offer kept: the
// session not Ready, a turn under way (a message being sent among them),
// zone B not drawn (the narrow layout's tree, the diff in its place); and
// once the offer is gone.
func TestSummaryOfferHidden(t *testing.T) {
	for _, c := range []struct {
		name string
		set  func(s *state.State, ui *UI)
	}{
		{"crashed", func(s *state.State, ui *UI) { s.Session.Phase = state.Crashed }},
		{"closing", func(s *state.State, ui *UI) { s.Session.Phase = state.Closing }},
		{"closed", func(s *state.State, ui *UI) { s.Session.Phase = state.Closed }},
		{"handshaking", func(s *state.State, ui *UI) { s.Session.Phase = state.Handshaking }},
		{"submitted", func(s *state.State, ui *UI) { s.Turn.Phase = state.Submitted }},
		{"running", func(s *state.State, ui *UI) { s.Turn.Phase = state.Running }},
		{"zone A's tree in B's place", func(s *state.State, ui *UI) { ui.Width, ui.Height, ui.Focus = 80, 24, "A" }},
		{"the diff in B's place", func(s *state.State, ui *UI) { ui.Diff.Open = true }},
		{"the offer gone", func(s *state.State, ui *UI) { s.Session.SummaryOffered = false }},
	} {
		s := offerState(t, offerCLIs[0].cli, offerCLIs[0].name, offerCLIs[0].id)
		ui := UI{Width: 120, Height: 40, Focus: "B", Summary: SummaryFocus{On: true}}
		c.set(s, &ui)
		offered := s.Session.SummaryOffered
		sc := Draw(s, ui)
		if text := texts(sc.lines); strings.Contains(text, summaryQuestion) || strings.Contains(text, summaryPrepared) || strings.Contains(text, summaryTitle) {
			t.Errorf("%s: the suggestion drawn:\n%s", c.name, text)
		}
		if sc.Conv.Summary {
			t.Errorf("%s: ConvView.Summary true", c.name)
		}
		if s.Session.SummaryOffered != offered {
			t.Errorf("%s: drawing changed the offer", c.name)
		}
	}
}
