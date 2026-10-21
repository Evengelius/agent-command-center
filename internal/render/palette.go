package render

import (
	"image/color"
	"strings"

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
	cBright    = colour{"#F9FAFB", "15"} // the owner's prompts in zone B (plan 19)
	cGround    = colour{"#1E2133", ""}   // zone B's tool rows; no ground on 16 colours (plan 19)
	cMuted     = colour{"#A1A9C0", "7"}  // a tool row's target, readable on its ground (plan 20)
	cPill      = colour{"#3B1531", ""}   // the "F2 to answer" pill (plan 20)

	// The violet of thinking (plan 22, design 2): the live card's border, its
	// bar's fill, the fill's lit leading edge, the mid shade (and the
	// expressions' lavender), the dim shade of the entrance and the exit, and
	// the bar's track.
	cThinkBorder = colour{"#7C6BB3", "5"}
	cThinkFill   = colour{"#A78BFA", "5"}
	cThinkEdge   = colour{"#EDE9FE", "15"}
	cThinkMid    = colour{"#C4B5FD", "13"}
	cThinkDim    = colour{"#4C4470", "8"}
	cThinkTrack  = colour{"#2E2A47", "8"}
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

	// stSynBuiltin and stSynTag colour the file viewer's built-in names and
	// markup tags (plan 17, ui-design.md, the file viewer).
	stSynBuiltin, stSynTag = new(lipgloss.Style), new(lipgloss.Style)

	// Zone B's labels and the owner's prompts (plan 19, ui-design.md, Zone
	// B): YOU violet, the CLI's speaker blue, RESULT green, each led by a
	// bar of its colour.
	stBright, stYou, stSpeaker, stResult = new(lipgloss.Style), new(lipgloss.Style), new(lipgloss.Style), new(lipgloss.Style)

	// A tool row's name and target, the fold row's triangle, and the pink
	// pill of a pending approval (plan 20, ui-design.md, Zone B).
	stToolName, stMuted, stFoldMark, stPill, stPillEnd = new(lipgloss.Style), new(lipgloss.Style), new(lipgloss.Style), new(lipgloss.Style), new(lipgloss.Style)

	// Zone B's thinking (plan 22): the card's border, the bar's fill, track,
	// lit edge and mid shade, the dim shade, the expression in lavender bold
	// (dim for a new one's first 200 ms), a count in bright bold or lit.
	stThinkBorder, stThinkFill, stThinkTrack, stThinkEdge, stThinkMid, stThinkDim = new(lipgloss.Style), new(lipgloss.Style), new(lipgloss.Style), new(lipgloss.Style), new(lipgloss.Style), new(lipgloss.Style)
	stThinkWord, stThinkWordDim, stThinkLit, stBrightBold                         = new(lipgloss.Style), new(lipgloss.Style), new(lipgloss.Style), new(lipgloss.Style)

	// The file viewer's frame, its cursor line (each class on zone A's
	// cursor row background, the line numbers likewise) and its search
	// matches on an amber background, the current one bold (plan 17, The
	// window).
	stFileFrame, stFileRowNum, stFileRowMark, stMatch, stMatchCur = new(lipgloss.Style), new(lipgloss.Style), new(lipgloss.Style), new(lipgloss.Style), new(lipgloss.Style)
	// The Explorer's cursor row: a folder's name, a dimmed name and git's
	// markers on zone A's cursor row background (plan 17, amendment 12).
	stExpRowDir, stExpRowGrey, stExpRowProgress, stExpRowDone, stExpRowFailed = new(lipgloss.Style), new(lipgloss.Style), new(lipgloss.Style), new(lipgloss.Style), new(lipgloss.Style)
	synRow                                                                    = func() (out []*lipgloss.Style) {
		for range SynHeading + 1 {
			out = append(out, new(lipgloss.Style))
		}
		return out
	}()

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

// paint builds every shared style from the palette in use, and its wrap.
func paint() {
	wraps = map[*lipgloss.Style][2]string{}
	paintStyle(stFrame, fg(cFrame))
	paintStyle(stTitle, fg(cTitle).Bold(true))
	paintStyle(stText, fg(cText))
	paintStyle(stDone, fg(cDone))
	paintStyle(stProgress, fg(cProgress))
	paintStyle(stAttention, fg(cAttention))
	paintStyle(stFailed, fg(cFailed))
	paintStyle(stGrey, fg(cGrey))
	paintStyle(stBold, fg(cText).Bold(true))
	paintStyle(stButton, fg(cInk).Background(cTitle.get()).Bold(true))
	paintStyle(stAttentionBold, stAttention.Bold(true))
	paintStyle(stFailedBold, stFailed.Bold(true))
	paintStyle(stNone, lipgloss.NewStyle())

	paintStyle(stCursor, fg(cInk).Background(cTitle.get()))
	paintStyle(stSelected, fg(cText).Background(cSelection.get()))
	paintStyle(stControl, fg(cAttention))
	paintStyle(stInferred, stText.Italic(true))
	paintStyle(stTreeCursor, fg(cTreeText).Background(cTreeRow.get()))

	paintStyle(stMdBullet, fg(cTitle))
	paintStyle(stMdEm, fg(cText).Italic(true))
	paintStyle(stMdCode, fg(cCode))
	paintStyle(stMdLink, fg(cLink).Underline(true))
	paintStyle(stMdLang, fg(cProgress))
	paintStyle(stMdKeyword, fg(cKeyword).Bold(true))
	paintStyle(stMdString, fg(cString))
	paintStyle(stMdNumber, fg(cNumber))
	paintStyle(stMdComment, fg(cGrey).Italic(true))
	paintStyle(stSynBuiltin, fg(cLink))
	paintStyle(stSynTag, fg(cKeyword))
	paintStyle(stFileFrame, fg(cTitle))
	paintStyle(stFileRowNum, fg(cGrey).Background(cTreeRow.get()))
	paintStyle(stFileRowMark, fg(cAttention).Background(cTreeRow.get()))
	paintStyle(stMatch, fg(cInk).Background(cProgress.get()))
	paintStyle(stMatchCur, stMatch.Bold(true).Underline(true))
	paintStyle(stExpRowDir, stTitle.Background(cTreeRow.get()))
	paintStyle(stExpRowGrey, stGrey.Background(cTreeRow.get()))
	paintStyle(stExpRowProgress, stProgress.Background(cTreeRow.get()))
	paintStyle(stExpRowDone, stDone.Background(cTreeRow.get()))
	paintStyle(stExpRowFailed, stFailed.Background(cTreeRow.get()))
	for c := range synRow {
		paintStyle(synRow[c], SynClass(c).style().Background(cTreeRow.get()))
	}

	paintStyle(stBright, fg(cBright))
	paintStyle(stYou, fg(cKeyword).Bold(true))
	paintStyle(stSpeaker, fg(cLink).Bold(true))
	paintStyle(stResult, fg(cDone).Bold(true))
	paintStyle(stToolName, fg(cCode).Bold(true))
	paintStyle(stMuted, fg(cMuted))
	paintStyle(stFoldMark, fg(cLink))
	paintStyle(stThinkBorder, fg(cThinkBorder))
	paintStyle(stThinkFill, fg(cThinkFill))
	paintStyle(stThinkTrack, fg(cThinkTrack))
	paintStyle(stThinkEdge, fg(cThinkEdge))
	paintStyle(stThinkMid, fg(cThinkMid))
	paintStyle(stThinkDim, fg(cThinkDim))
	paintStyle(stThinkWord, fg(cThinkMid).Bold(true))
	paintStyle(stThinkWordDim, fg(cThinkDim).Bold(true))
	paintStyle(stThinkLit, fg(cThinkEdge).Bold(true))
	paintStyle(stBrightBold, fg(cBright).Bold(true))
	if sixteen { // no ground to spare: the pill is its bold pink text
		paintStyle(stPill, stAttentionBold.Background(lipgloss.NoColor{}))
		paintStyle(stPillEnd, lipgloss.NewStyle())
	} else {
		paintStyle(stPill, stAttentionBold.Background(cPill.get()))
		paintStyle(stPillEnd, fg(cPill))
	}

	for i, m := range modelColours {
		paintStyle(modelStyles[i], fg(m.c))
	}
	for _, g := range grounds {
		for base, t := range g.tints {
			if !sixteen { // unused there: on gives the base style
				paintStyle(t, base.Background(g.c.get()))
			}
		}
	}
}

// A ground is a background zone B draws on, with the shared styles on it,
// by the style each one grounds; paint repaints them: the tool rows' (plan
// 19) and the cards' (plan 20).
type ground struct {
	c     colour
	tints map[*lipgloss.Style]*lipgloss.Style
}

var (
	toolGround = &ground{c: cGround, tints: map[*lipgloss.Style]*lipgloss.Style{}}
	grounds    = []*ground{toolGround}
)

// on is st on the ground, made on first use; st itself on a 16-colour
// terminal, which has no ground to spare.
func (g *ground) on(st *lipgloss.Style) *lipgloss.Style {
	if sixteen {
		return st
	}
	if t := g.tints[st]; t != nil {
		return t
	}
	t := new(lipgloss.Style)
	paintStyle(t, st.Background(g.c.get()))
	g.tints[st] = t
	return t
}

// tint is st on the tool rows' ground (plan 19).
func tint(st *lipgloss.Style) *lipgloss.Style { return toolGround.on(st) }

// wraps is each shared style's escape codes before and after a span's text,
// for line.String: what Lip Gloss's Render gives a single line without tabs,
// for a style that styles spaces as the rest (ui-design.md, Budget).
var wraps map[*lipgloss.Style][2]string

// paintStyle gives the shared style p the value v, and its wrap when Render is the
// text between two fixed strings: a probe with spaces, a wide and an accented
// character checks it, so a style Lip Gloss draws otherwise (underlined
// spaces) keeps going through Render.
func paintStyle(p *lipgloss.Style, v lipgloss.Style) {
	*p = v
	delete(wraps, p)
	x := v.Render("x")
	i := strings.IndexByte(x, 'x')
	if i < 0 {
		return
	}
	const probe = "a b · é日 z"
	if w := [2]string{x[:i], x[i+1:]}; v.Render(probe) == w[0]+probe+w[1] {
		wraps[p] = w
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

// style is the shared style of a syntax class, so that a palette change
// repaints it (plan 17, ui-design.md, the file viewer).
func (c SynClass) style() *lipgloss.Style {
	switch c {
	case SynKeyword:
		return stMdKeyword
	case SynString:
		return stMdString
	case SynNumber:
		return stMdNumber
	case SynComment:
		return stMdComment
	case SynFunc, SynAttr:
		return stMdCode
	case SynBuiltin:
		return stSynBuiltin
	case SynTag:
		return stSynTag
	case SynInserted:
		return stDone
	case SynDeleted:
		return stFailed
	case SynHeading:
		return stTitle
	}
	return stText
}
