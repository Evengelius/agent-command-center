package render

import (
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/Evengelius/agent-command-center/internal/state"
)

// The one-time offer of a summary request after a resume from the home
// screen (plan 70, design D2): a card of the owner's texts and two buttons,
// drawn in zone B after the session's parts while the offer stands, as the
// owner's design draws it (brief amendment 73).
const (
	// SummaryPanel is the buttons' panel for a click: item SummaryPrepare
	// or SummaryNotNow.
	SummaryPanel   = 'N'
	SummaryPrepare = 1
	SummaryNotNow  = 2

	summaryTitle    = "SESSION RESUMED · SUGGESTION"
	summaryShort    = "SUGGESTION" // the title where the whole one does not fit
	summaryQuestion = "Want a summary of your earlier work?"
	summaryPrepared = "[ Prepare summary ]"
	summaryNotNow   = "[ Not now ]"
	summaryGap      = "     "
	summaryAdded    = "The request will be added to your draft."
	summaryEnter    = "Press Enter to send it."
	// summaryWidth is the card's width in the owner's design, narrower
	// where zone B is; summaryPad, its margin inside each side edge.
	summaryWidth = 55
	summaryPad   = "  "
	// summaryAbove is the rows the card leaves above it in zone B: the
	// resumed note, two rows where B is narrow, and a blank row on each
	// side.
	summaryAbove = 4

	summaryFooter = "←→ select · Enter choose · ↑ entries · Esc back to the input · F1 keys"
)

// SummaryFocus is the suggestion's keyboard focus (plan 70, design D3): On
// while it has it, NotNow when Not now is the button selected, else
// Prepare summary.
type SummaryFocus struct {
	On, NotNow bool
}

// SummaryShown says zone B shows the suggestion: the offer stands, the
// session is Ready and no turn is under way, a message being sent among
// them (plan 70, design D2).
func SummaryShown(s *state.State) bool {
	return s.Session.SummaryOffered && s.Session.Phase == state.Ready && s.Turn.Phase == state.Idle
}

// summaryRows are the suggestion's rows in a row w cells wide, in zone B's
// room rows: a card summaryWidth wide, or w where B is narrower, its title
// in the top edge, SUGGESTION alone or none where it does not fit; inside,
// two cells from each side edge and a blank row at each end, the question,
// the two buttons side by side or, too wide, one above the other, a rule
// of hyphens and the two notes, a blank row between the four (amendment
// 73); then a blank row above the input's rule, as after the last message
// (amendment 17). Where room does not hold it with that row and
// summaryAbove rows above it, the card drops blank rows, those around the
// rule, then those inside its edges, then the one under the question, as
// many as that takes, so that its edges stay whole with the note above
// them. None where the card has no cell inside its margins. Decoration,
// never copied; no ground anywhere, the focused button's label underlined.
func summaryRows(s *state.State, ui UI, w, room int) []crow {
	cw := min(w, summaryWidth)
	inner := cw - 2 - 2*len(summaryPad)
	if !SummaryShown(s) || inner < 1 {
		return nil
	}
	row := func(l line, buttons bool) crow {
		r := append(line{deco("│"+summaryPad, stSummaryFrame)}, l.fit(inner)...)
		return crow{l: append(r, deco(summaryPad+"│", stSummaryFrame)), summary: buttons}
	}
	text := func(t string, st *lipgloss.Style) (out []crow) {
		for _, r := range wrap(t, inner) {
			out = append(out, row(decoLine(r, st), false))
		}
		return out
	}
	prepare := summaryButton(summaryPrepared, SummaryPrepare, ui.Summary.On, !ui.Summary.NotNow)
	notNow := summaryButton(summaryNotNow, SummaryNotNow, ui.Summary.On, ui.Summary.NotNow)
	buttons := []crow{row(append(append(prepare, deco(summaryGap, stNone)), notNow...), true)}
	if textWidth(summaryPrepared+summaryGap+summaryNotNow) > inner {
		buttons = []crow{row(prepare, true), row(notNow, true)}
	}
	question, notes := text(summaryQuestion, stBold), append(text(summaryAdded, stGrey), text(summaryEnter, stGrey)...)
	// The blank rows kept: 5, the design's; 3 without the rule's, 1 without
	// the edges' too, 0 without the question's.
	keep := 5
	for _, k := range []int{5, 3, 1, 0} {
		if keep = k; 2+len(question)+len(buttons)+1+len(notes)+k+1+summaryAbove <= room {
			break
		}
	}
	blank := func(on bool) []crow {
		if on {
			return []crow{row(nil, false)}
		}
		return nil
	}
	top := decoLine("╭"+strings.Repeat("─", cw-2)+"╮", stSummaryFrame)
	for _, t := range []string{summaryTitle, summaryShort} {
		if fill := cw - 5 - textWidth(t); fill >= 1 {
			top = line{deco("╭─ ", stSummaryFrame), deco(t, stTitle), deco(" "+strings.Repeat("─", fill)+"╮", stSummaryFrame)}
			break
		}
	}
	out := append([]crow{{l: top}}, blank(keep >= 3)...)
	out = append(append(out, question...), blank(keep >= 1)...)
	out = append(append(out, buttons...), blank(keep == 5)...)
	out = append(out, row(decoLine(strings.Repeat("-", inner), stFrame), false))
	out = append(append(out, blank(keep == 5)...), notes...)
	out = append(append(out, blank(keep >= 3)...), crow{l: decoLine("╰"+strings.Repeat("─", cw-2)+"╯", stSummaryFrame)})
	return append(out, crow{})
}

// summaryButton is a button's brackets and label, each a click's target
// (item): with the focus on it, bright and bold, its label underlined; the
// other subdued while the focus is on one; both in the title colour
// without it. No ground and no inverse: only the text changes (amendment
// 73).
func summaryButton(text string, item int, focus, selected bool) line {
	r := ref{panel: SummaryPanel, kind: refDeco, item: item}
	edge, label := stTitle, stTitle
	switch {
	case focus && selected:
		edge, label = stBrightBold, stSummaryFocus
	case focus:
		edge, label = stMuted, stMuted
	}
	return line{{text: "[ ", st: edge, ref: r}, {text: text[2 : len(text)-2], st: label, ref: r}, {text: " ]", st: edge, ref: r}}
}
