package render

import (
	"image/color"

	"charm.land/lipgloss/v2"
)

// The palette (ui-design.md, Colours and symbols). Each colour has its true
// colour, drawn as is in 24 bits and mapped by Bubble Tea's renderer to the
// nearest of 256, and the ANSI colour it takes on a 16-colour terminal, where
// the renderer's own mapping folds several roles into one (plan 5, item 13).
// Every shared style is built here and rebuilt in place when the palette
// changes: spans hold pointers to them.

// colour is a palette colour: true colour, then ANSI index 0 to 15.
type colour struct{ tc, ansi string }

var (
	cFrame     = colour{"#3B3F51", "8"}
	cTitle     = colour{"#C4B5FD", "15"}
	cText      = colour{"#E5E7EB", "7"}
	cDone      = colour{"#4ADE80", "10"}
	cProgress  = colour{"#FBBF24", "11"}
	cAttention = colour{"#F472B6", "13"}
	cFailed    = colour{"#F87171", "9"}
	cGrey      = colour{"#6B7280", "8"}
	cInk       = colour{"#111827", "0"}  // text on the title colour
	cSelection = colour{"#4C4F6B", "4"}  // the editor's selection
	cTreeRow   = colour{"#2E3247", "8"}  // zone A's cursor row
	cKeyword   = colour{"#A78BFA", "5"}  // as Opus
	cString    = colour{"#86EFAC", "2"}  //
	cNumber    = colour{"#FB923C", "3"}  // as Sonnet
	cCode      = colour{"#93C5FD", "6"}  //
	cLink      = colour{"#60A5FA", "12"} // as Haiku
	cTreeText  = colour{"#E5E7EB", "15"} // text on zone A's cursor row
)

// modelColours colours a model by its family; any other model is text.
var modelColours = []struct {
	family string
	c      colour
}{{"opus", colour{"#A78BFA", "5"}}, {"sonnet", colour{"#FB923C", "3"}}, {"haiku", colour{"#60A5FA", "12"}}, {"gpt", colour{"#2DD4BF", "14"}}}

// sixteen is the 16-colour palette in use.
var sixteen bool

// get is c in the palette in use.
func (c colour) get() color.Color {
	if sixteen {
		return lipgloss.Color(c.ansi)
	}
	return lipgloss.Color(c.tc)
}

func fg(c colour) lipgloss.Style { return lipgloss.NewStyle().Foreground(c.get()) }

var (
	stFrame, stTitle, stText, stDone, stProgress, stAttention, stFailed, stGrey, stBold, stButton      = new(lipgloss.Style), new(lipgloss.Style), new(lipgloss.Style), new(lipgloss.Style), new(lipgloss.Style), new(lipgloss.Style), new(lipgloss.Style), new(lipgloss.Style), new(lipgloss.Style), new(lipgloss.Style)
	stAttentionBold, stFailedBold, stNone                                                              = new(lipgloss.Style), new(lipgloss.Style), new(lipgloss.Style)
	stCursor, stSelected, stControl, stInferred, stTreeCursor                                          = new(lipgloss.Style), new(lipgloss.Style), new(lipgloss.Style), new(lipgloss.Style), new(lipgloss.Style)
	stMdBullet, stMdEm, stMdCode, stMdLink, stMdLang, stMdKeyword, stMdString, stMdNumber, stMdComment = new(lipgloss.Style), new(lipgloss.Style), new(lipgloss.Style), new(lipgloss.Style), new(lipgloss.Style), new(lipgloss.Style), new(lipgloss.Style), new(lipgloss.Style), new(lipgloss.Style)

	stMdHeading = stTitle
	stMdMark    = stGrey // #, >, list bullets' neighbours, |, fences, ** and `
	stMdStrong  = stBold

	modelStyles = func() (out []*lipgloss.Style) {
		for range modelColours {
			out = append(out, new(lipgloss.Style))
		}
		return out
	}()
)

func init() { paint() }

// paint builds every shared style from the palette in use.
func paint() {
	*stFrame = fg(cFrame)
	*stTitle = fg(cTitle).Bold(true)
	*stText = fg(cText)
	*stDone = fg(cDone)
	*stProgress = fg(cProgress)
	*stAttention = fg(cAttention)
	*stFailed = fg(cFailed)
	*stGrey = fg(cGrey)
	*stBold = fg(cText).Bold(true)
	*stButton = fg(cInk).Background(cTitle.get()).Bold(true)
	*stAttentionBold = stAttention.Bold(true)
	*stFailedBold = stFailed.Bold(true)
	*stNone = lipgloss.NewStyle()

	*stCursor = fg(cInk).Background(cTitle.get())
	*stSelected = fg(cText).Background(cSelection.get())
	*stControl = fg(cAttention)
	*stInferred = stText.Italic(true)
	*stTreeCursor = fg(cTreeText).Background(cTreeRow.get())

	*stMdBullet = fg(cTitle)
	*stMdEm = fg(cText).Italic(true)
	*stMdCode = fg(cCode)
	*stMdLink = fg(cLink).Underline(true)
	*stMdLang = fg(cProgress)
	*stMdKeyword = fg(cKeyword).Bold(true)
	*stMdString = fg(cString)
	*stMdNumber = fg(cNumber)
	*stMdComment = fg(cGrey).Italic(true)

	for i, m := range modelColours {
		*modelStyles[i] = fg(m.c)
	}
}

// UsePalette takes the palette for the terminal's colour profile, given as
// Lip Gloss's choice among an ANSI, a 256 and a true colour: the 16-colour
// palette when it picks the ANSI one, the true colours otherwise, which
// Bubble Tea maps to 256 or strips with no colour. It reports whether the
// palette changed, so that the next frame draws every row again.
func UsePalette(pick lipgloss.CompleteFunc) bool {
	ansi := lipgloss.Color("1")
	s := pick(ansi, lipgloss.Color("100"), lipgloss.Color("#010203")) == ansi
	if s == sixteen {
		return false
	}
	sixteen = s
	paint()
	return true
}
