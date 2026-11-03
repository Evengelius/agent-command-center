package render

import (
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/Evengelius/agent-command-center/internal/state"
)

// The question to return to the home screen (plan 75, brief amendments 89 to 91):
// Esc where it does nothing else on the dashboard asks it, in a card over
// the dashboard drawn with the summary card's frame and buttons (design D7).

const (
	// ReturnPanel is the question's buttons' panel for a click: item ReturnYes
	// or ReturnNo.
	ReturnPanel = 'W'
	ReturnYes   = 1
	ReturnNo    = 2

	returnTitle = "RETURN TO THE HOME SCREEN?"
	returnYes   = "[ Yes ]"
	returnNo    = "[ No ]"
	returnHint  = "←→ select · Enter answer · Esc no"
	// returnWidth is the card's width, narrower where the screen is.
	returnWidth = 60
)

// ReturnAsk is the question (design D7): Open while it shows, Yes when Yes is
// the button selected, else No, the one selected when it opens.
type ReturnAsk struct {
	Open, Yes bool
}

// ReturnText is what the question says the return does, name the session's
// CLI as the startup screen names it (design D7).
func ReturnText(name string) string {
	return "The session ends and " + name + " closes. Its changes stay in the project, and the session stays in the history."
}

// returnBox draws the question over the dashboard, centred on zone B's
// columns where they hold it, else on the screen's, and on the screen's
// rows: inside a rounded frame, two cells from each side edge, the title in
// amber, the text, the two buttons, the keys in grey, a blank row between
// them and at each end. It records where it lies, for a click outside it
// (No). Rows the screen has no room for are left out from the blank ones,
// then the keys'; none at all where the card has no cell inside its margins.
func returnBox(sc *Screen, s *state.State, ui UI) {
	x0, room := 0, ui.Width
	if b := sc.B; b.W >= min(returnWidth, ui.Width-2)+2 {
		x0, room = b.X, b.W
	}
	cw := min(room-2, returnWidth)
	inner := cw - 2 - 2*len(summaryPad)
	if inner < len(returnYes) || ui.Height < 7 {
		return
	}
	row := func(l line) line {
		r := append(line{deco("│"+summaryPad, stSummaryFrame)}, l.fit(inner)...)
		return append(r, deco(summaryPad+"│", stSummaryFrame))
	}
	text := func(t string, st *lipgloss.Style) (out []line) {
		for _, r := range wrap(t, inner) {
			out = append(out, row(decoLine(r, st)))
		}
		return out
	}
	name := s.Context.ProviderName
	if name == "" {
		name = "the agent CLI"
	}
	yes := cardButton(returnYes, ref{panel: ReturnPanel, kind: refDeco, item: ReturnYes}, true, ui.Return.Yes)
	no := cardButton(returnNo, ref{panel: ReturnPanel, kind: refDeco, item: ReturnNo}, true, !ui.Return.Yes)
	buttons := []line{row(append(append(yes, deco(summaryGap, stNone)), no...))}
	if textWidth(returnYes+summaryGap+returnNo) > inner {
		buttons = []line{row(yes), row(no)}
	}
	title, body := text(returnTitle, stProgressBold), text(ReturnText(name), stText)
	hint := []line{row(decoLine(elideEnd(returnHint, inner), stGrey))}
	blank := row(nil)
	// The rows: the edges, the title, the text, the buttons, the keys, and
	// the five blank rows; those the screen cannot hold go, then the keys.
	blanks := 5
	need := func() int { return 2 + len(title) + len(body) + len(buttons) + len(hint) + blanks }
	for blanks > 0 && need() > ui.Height {
		blanks--
	}
	if need() > ui.Height {
		hint = nil
	}
	if need() > ui.Height {
		return
	}
	gap := func(i int) []line {
		if i < blanks {
			return []line{blank}
		}
		return nil
	}
	box := []line{decoLine("╭"+strings.Repeat("─", cw-2)+"╮", stSummaryFrame)}
	box = append(box, gap(3)...)
	box = append(append(box, title...), gap(0)...)
	box = append(append(box, body...), gap(1)...)
	box = append(append(box, buttons...), gap(2)...)
	if hint != nil {
		box = append(append(box, hint...), gap(4)...)
	}
	box = append(box, decoLine("╰"+strings.Repeat("─", cw-2)+"╯", stSummaryFrame))
	x := x0 + (room-cw)/2
	y := (ui.Height - len(box)) / 2
	sc.place(box, x, y, cw)
	sc.Return = Box{X: x, Y: y, W: cw, H: len(box)}
}
