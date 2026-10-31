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
	cActive    = colour{"#272745", "4"}  // the "+" pressed and the menu item under the mouse (plan 30, the owner's image)

	// The model selector (plan 34, the owner's image; reference-design.md,
	// The model selector), each read from the image's pixels: the frame,
	// the header and the tail; "Current session"; the descriptions; the
	// cursor's band; its pink "▶"; the separators; the toggle's track on,
	// and off.
	cModelFrame  = colour{"#CA92F2", "13"}
	cModelAside  = colour{"#9299A9", "8"}
	cModelDesc   = colour{"#848FA2", "8"}
	cModelBand   = colour{"#342B47", "5"}
	cModelMark   = colour{"#E286D0", "13"}
	cModelRule   = colour{"#5E616D", "8"}
	cModelToggle = colour{"#CA84FB", "13"}
	cModelOff    = colour{"#3B3F51", "8"}
	// The toggle's knob, and its track while Ultracode is unavailable (plan
	// 35, design D1; the knob then cModelRule).
	cModelKnob = colour{"#FFFFFF", "15"}
	cModelDim  = colour{"#2A2D38", "8"}
	// The effort slider in text (plan 37, design D2; the image's colours in
	// slider.go): its track, its fill, its dots on each, a held effort's
	// fill and thumb, and the highest level's ends.
	cSliderTrack    = colour{"#272B2D", "8"}
	cSliderFill     = colour{"#6784C1", "4"}
	cSliderDot      = colour{"#697276", "8"}
	cSliderDotFill  = colour{"#95A9D4", "12"}
	cSliderDotTop   = colour{"#C9B8EC", "13"}
	cSliderLockFill = colour{"#4B505A", "8"}
	cSliderLockKnob = colour{"#9AA0AA", "7"}
	cSliderTopLeft  = colour{"#4C3F99", "5"}
	cSliderTopRight = colour{"#805CB4", "5"}

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

	// The modes selector (plan 39; brief amendment 40's measures): the
	// box's ground, its frame, "MODES", "Current session", a title, a
	// description, the band, the selected option's title and description,
	// an unavailable option's text, Full access's amber, an icon in text.
	cModesGround   = colour{"#111518", "0"}
	cModesFrame    = colour{"#B064F4", "13"}
	cModesHead     = colour{"#B86AF6", "13"}
	cModesAside    = colour{"#A0AECC", "7"}
	cModesTitle    = colour{"#F0F0F2", "15"}
	cModesDesc     = colour{"#95A3BF", "7"}
	cModesBand     = colour{"#2E2842", "8"}
	cModesSelTitle = colour{"#F2EFF6", "15"}
	cModesSelDesc  = colour{"#B09FD5", "13"}
	cModesOff      = colour{"#5A6170", "8"}
	cModesWarn     = colour{"#F5C673", "11"}
	cModesIcon     = colour{"#AEBCDC", "7"}

	// The activity sidebar's tiles in text (plan 43, designs D2 and D3, read
	// from the references' pixels): an icon inactive, an icon lit (selected
	// or hovered), the selected tile's ground, a shade lighter while
	// hovered, and its lavender accent bar; no ground on 16 colours.
	cSideIcon     = colour{"#6F7C88", "8"}
	cSideIconLit  = colour{"#A6B3C6", "7"}
	cSideSel      = colour{"#181C30", ""}
	cSideSelHover = colour{"#20253D", ""}
	cSideAccent   = colour{"#A272F1", "13"}
	// The Explorer panel's icons (plan 43, design D6): a folder's blue, a
	// file's grey.
	cExpFolder = colour{"#6CA1DB", "12"}
	cExpFile   = colour{"#A3B1C9", "7"}
	// Its names, after the reference (design D6): a folder's and the
	// project root's, a file's, light lavender greys, not bold.
	cExpDirName  = colour{"#A0ABD8", "7"}
	cExpFileName = colour{"#9EAAD5", "7"}
	// The Explorer's Nerd Font icons' colours (plan 55, design D4): each
	// kind's usual one, as VS Code's icon themes give it.
	cIconGo         = colour{"#00ADD8", "14"}
	cIconBlue       = colour{"#519ABA", "12"}
	cIconPowerShell = colour{"#5391FE", "12"}
	cIconGreen      = colour{"#89E051", "10"}
	cIconYellow     = colour{"#CBCB41", "11"}
	cIconPurple     = colour{"#A074C4", "13"}
	cIconConfig     = colour{"#6D8086", "8"}
	cIconOrange     = colour{"#E37933", "3"}
	cIconRed        = colour{"#E34C26", "9"}
	cIconMaven      = colour{"#C71A36", "1"}
	cIconTeal       = colour{"#4DB6AC", "6"}
	cIconPython     = colour{"#4B8BBE", "12"}
	cIconAmber      = colour{"#ECA517", "3"}
	cIconDocker     = colour{"#2496ED", "12"}
	cIconLock       = colour{"#BBA77A", "3"}
	cIconLicense    = colour{"#D0BF41", "11"}
	cIconGit        = colour{"#F14E32", "9"}
	cIconNpm        = colour{"#CB3837", "1"}
	cIconGitHub     = colour{"#C9D1D9", "7"}
	cIconVSCode     = colour{"#23A9F2", "12"}
	cIconClaude     = colour{"#D97757", "3"}
	cIconCode       = colour{"#B392F0", "13"}
	cIconTest       = colour{"#E5C07B", "11"}
	cIconBookmark   = colour{"#F78C6C", "9"}
	cIconPackage    = colour{"#D19A66", "3"}
	// Source Control's hovered row (plan 44, design D4), subtler than the
	// cursor's purple band.
	cSourceHover = colour{"#221D2E", ""}
	// The diff's lines (plan 44, design D8): an added line's text and ground,
	// green, a removed one's, red; on 16 colours the text alone.
	cDiffAdd       = colour{"#B4F1C6", "10"}
	cDiffAddGround = colour{"#12301F", ""}
	cDiffDel       = colour{"#FFC4C4", "9"}
	cDiffDelGround = colour{"#3A161A", ""}
	// Source Control's commit controls (plan 45, design D5; the owner's image
	// references/15-commit-push.png, read from its pixels): the message
	// box's inside, outline, outline with the focus and placeholder; the
	// button's fill, outline, label and separator, and each halfway to the
	// page (#0C0D0F) while it has nothing to do; the dropdown's ground,
	// outline, band and text. No ground on 16 colours.
	cMsgInside     = colour{"#151722", ""}
	cMsgOutline    = colour{"#2E304E", "8"}
	cMsgFocus      = colour{"#8062D9", "5"}
	cMsgHolder     = colour{"#71789E", "8"}
	cBtnFill       = colour{"#312C55", ""}
	cBtnOutline    = colour{"#8062D9", "5"}
	cBtnLabel      = colour{"#ECE5F9", "15"}
	cBtnSep        = colour{"#745EBA", "5"}
	cBtnFillDim    = colour{"#1E1C32", ""}
	cBtnOutlineDim = colour{"#463774", "8"}
	cBtnLabelDim   = colour{"#7C7984", "8"}
	cBtnSepDim     = colour{"#403564", "8"}
	cMenuGround    = colour{"#10101A", ""}
	cMenuOutline   = colour{"#935EF5", "13"}
	cMenuBand      = colour{"#23213C", ""}
	cMenuText      = colour{"#EDE7F8", "15"}
	// The home screen (plan 51; the owner's image,
	// references/16-home-screen.webp, read from its pixels in
	// reference-design.md, The home screen): a fill's commonest colour, a
	// mark's the mean of its brightest pixels, which a thin stroke reaches
	// at its middle only. No ground on 16 colours.
	cHomePage        = colour{"#090B0C", ""}
	cHomeBanner      = colour{"#050709", ""}
	cHomeDot         = colour{"#14161C", "8"}
	cHomeStar        = colour{"#D8C4FC", "15"}
	cHomeHalftone    = colour{"#ABA5CB", "8"}
	cHomeHalftoneRim = colour{"#2A2152", "5"}
	cHomeSparkle     = colour{"#F4E6FF", "15"}
	cHomeSparkleDim  = colour{"#7C70A0", "8"}
	cHomeSparkleGlow = colour{"#3A0A9C", "5"}
	cHomeLogo        = colour{"#988AC3", "5"}
	cHomeTitle       = colour{"#C7C4E2", "7"}
	cHomeVersion     = colour{"#9998AE", "8"}
	cHomeSystem      = colour{"#A9A7C1", "7"}
	cHomeBar         = colour{"#6B6789", "8"}
	cHomeReady       = colour{"#67C999", "10"}
	cHomeStatus      = colour{"#A4A2BD", "7"}
	cHomeFrame       = colour{"#615F75", "8"}
	cHomeChevron     = colour{"#CBC5EA", "7"}
	cHomePanelTitle  = colour{"#D2CCEA", "15"}
	cHomeUnderline   = colour{"#1D1D28", "8"}
	cHomeRule        = colour{"#634B95", "5"}
	cHomeRuleLow     = colour{"#513A86", "5"}
	cHomeNumber      = colour{"#BBBCC3", "7"}
	cHomeAsk         = colour{"#D7D8E2", "15"}
	cHomeChosenLine  = colour{"#8073A7", "5"}
	cHomeChosenFill  = colour{"#181228", ""}
	cHomeChosenName  = colour{"#CCC2E8", "13"}
	cHomeChosenVer   = colour{"#AEA3CF", "5"}
	cHomeOtherLine   = colour{"#17181F", "8"}
	cHomeOtherName   = colour{"#C8C9D5", "7"}
	cHomeOtherVer    = colour{"#848598", "8"}
	cHomeRadioOn     = colour{"#9480C5", "13"}
	cHomeRadioDot    = colour{"#A381F1", "13"}
	cHomeRadioOff    = colour{"#807E9B", "8"}
	cHomeField       = colour{"#0F1317", ""}
	cHomeFieldTop    = colour{"#ACA5C0", "7"}
	cHomeFieldLow    = colour{"#76758A", "8"}
	cHomeDivider     = colour{"#494957", "8"}
	cHomeFolder      = colour{"#9790B0", "7"}
	cHomePath        = colour{"#C1C1CF", "7"}
	cHomeButton      = colour{"#9C80FB", "13"}
	cHomeButtonText  = colour{"#24145C", "0"}
	cHomeSection     = colour{"#787596", "5"}
	cHomeSectionIcon = colour{"#B3AFCB", "7"}
	cHomeClock       = colour{"#B8B1CF", "7"}
	cHomeStarOn      = colour{"#9D81EF", "13"}
	cHomeStarOff     = colour{"#707086", "8"}
	cHomeName        = colour{"#CECFD9", "7"}
	cHomeNameSel     = colour{"#E9E7F5", "15"}
	cHomeRowPath     = colour{"#747688", "8"}
	cHomeRowPathSel  = colour{"#908FAA", "8"}
	cHomeAge         = colour{"#717385", "8"}
	cHomeAgeSel      = colour{"#8B8AA7", "8"}
	cHomeBand        = colour{"#16171F", ""}
	cHomeBandTop     = colour{"#282834", "8"}
	cHomeBandSide    = colour{"#1D1E26", "8"}
	cHomeSeparator   = colour{"#14171C", "8"}
	cHomeFootLine    = colour{"#0B0C17", ""}
	cHomeFootChevron = colour{"#CFBBFD", "13"}
	cHomeShortcuts   = colour{"#B6A7D9", "13"}
	cHomeCapLine     = colour{"#20232B", "8"}
	cHomeCapFill     = colour{"#0F1016", ""}
	cHomeCapText     = colour{"#BBB9CF", "7"}
	cHomeAction      = colour{"#83829D", "8"}
	cHomeFootBar     = colour{"#6E638B", "8"}
	// What the image does not show (plan 51, designs D6 and D10): START
	// SESSION dimmed, halfway to the page, and its label; the focus's
	// outline, the radio's purple, and START SESSION's, the selected name's
	// white; the list's band with the focus and its top line, a shade
	// brighter than without.
	cHomeButtonDim     = colour{"#53468A", "5"}
	cHomeButtonTextDim = colour{"#A9A0CB", "8"}
	cHomeFocus         = colour{"#A381F1", "13"}
	cHomeFocusOnButton = colour{"#E9E7F5", "15"}
	cHomeBandFocus     = colour{"#22222E", ""}
	cHomeBandFocusTop  = colour{"#3A3A4C", "8"}
)

// laneColour is the Git Graph's lanes' colours, taken in turn (plan 62,
// design D9): the families of the palette's others, each apart from the
// next on the dark ground and on 16 colours.
var laneColour = [laneColours]colour{{"#60A5FA", "12"}, {"#F472B6", "13"}, {"#4ADE80", "10"}, {"#FBBF24", "11"},
	{"#A78BFA", "5"}, {"#2DD4BF", "14"}, {"#FB923C", "3"}, {"#F87171", "9"}}

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

	// Zone C's rule under its header and the line between its panels, in
	// the title colour, not bold (plan 27, the owner's close-ups; design
	// D7).
	stPanelRule = new(lipgloss.Style)

	// The file viewer's frame, its cursor line (each class on zone A's
	// cursor row background, the line numbers likewise) and its search
	// matches on an amber background, the current one bold (plan 17, The
	// window).
	stFileFrame, stFileRowNum, stFileRowMark, stMatch, stMatchCur = new(lipgloss.Style), new(lipgloss.Style), new(lipgloss.Style), new(lipgloss.Style), new(lipgloss.Style)
	// The Explorer's cursor row: a dimmed name and git's markers on zone A's
	// cursor row background (plan 17, amendment 12); its names' since plan
	// 43 are stExpRowDirName and stExpRowFileName.
	stExpRowGrey, stExpRowProgress, stExpRowDone, stExpRowFailed = new(lipgloss.Style), new(lipgloss.Style), new(lipgloss.Style), new(lipgloss.Style)
	synRow                                                       = func() (out []*lipgloss.Style) {
		for range SynHeading + 1 {
			out = append(out, new(lipgloss.Style))
		}
		return out
	}()

	// The model selector (plan 34): its frame, its bold header, "Current
	// session", the descriptions, the cursor's "▶", the separators, and the
	// text toggle's rounded ends on, off and unavailable and its knob (plan
	// 35).
	stModelFrame, stModelHead, stModelAside, stModelDesc, stModelMark, stModelRule = new(lipgloss.Style), new(lipgloss.Style), new(lipgloss.Style), new(lipgloss.Style), new(lipgloss.Style), new(lipgloss.Style)
	stModelTrackEnd, stModelOffEnd, stModelDimEnd, stModelKnob                     = new(lipgloss.Style), new(lipgloss.Style), new(lipgloss.Style), new(lipgloss.Style)
	// The effort slider in text (plan 37, design D2): its ends on the track,
	// the fill, a held fill and the highest level's left and right; its
	// dots on the track, the fill and the gradient; a held thumb.
	stSliderTrackEnd, stSliderFillEnd, stSliderLockEnd, stSliderTopLeft, stSliderTopRight = new(lipgloss.Style), new(lipgloss.Style), new(lipgloss.Style), new(lipgloss.Style), new(lipgloss.Style)
	stSliderDot, stSliderDotFill, stSliderDotTop, stSliderLockKnob                        = new(lipgloss.Style), new(lipgloss.Style), new(lipgloss.Style), new(lipgloss.Style)
	// The modes selector (plan 39): its frame, its heading and the
	// confirmation's, "Current session", the halves' band, a title, a
	// description, an unavailable option's text, the amber, the selected
	// option's title and description, an icon in text.
	stModesFrame, stModesHead, stModesWarnHead, stModesAside, stModesHalf, stModesTitle = new(lipgloss.Style), new(lipgloss.Style), new(lipgloss.Style), new(lipgloss.Style), new(lipgloss.Style), new(lipgloss.Style)
	stModesDesc, stModesOff, stModesWarn, stModesSelTitle, stModesSelDesc, stModesIcon  = new(lipgloss.Style), new(lipgloss.Style), new(lipgloss.Style), new(lipgloss.Style), new(lipgloss.Style), new(lipgloss.Style)
	// The activity sidebar's tiles in text (plan 43, design D3): an icon
	// inactive and lit, the selected tile's accent bar.
	stSideIcon, stSideIconLit, stSideAccent = new(lipgloss.Style), new(lipgloss.Style), new(lipgloss.Style)
	// The Explorer panel's folder and file icons, and both on zone A's
	// cursor row background (plan 43, design D6).
	stExpFolder, stExpFile, stExpRowFolder, stExpRowFile = new(lipgloss.Style), new(lipgloss.Style), new(lipgloss.Style), new(lipgloss.Style)
	// Its folders' and files' names, and both on the cursor row's
	// background (design D6).
	stExpDirName, stExpFileName, stExpRowDirName, stExpRowFileName = new(lipgloss.Style), new(lipgloss.Style), new(lipgloss.Style), new(lipgloss.Style)
	// Source Control's section titles and a type change's letter (plan 44,
	// design D4); the diff's added and removed text (design D8).
	stTitleBold, stInfo, stDiffAdd, stDiffDel = new(lipgloss.Style), new(lipgloss.Style), new(lipgloss.Style), new(lipgloss.Style)
	// The commit controls in text (plan 45, design D5): the message box's
	// outline, with the focus, its placeholder; the button's outline, with
	// the focus, dimmed, its label and separator, both dimmed; the
	// dropdown's outline and text.
	stMsgOutline, stMsgFocus, stMsgHolder                                = new(lipgloss.Style), new(lipgloss.Style), new(lipgloss.Style)
	stBtnOutline, stBtnFocus, stBtnOutlineDim, stBtnLabel, stBtnLabelDim = new(lipgloss.Style), new(lipgloss.Style), new(lipgloss.Style), new(lipgloss.Style), new(lipgloss.Style)
	stBtnSep, stBtnSepDim, stMenuOutline, stMenuText                     = new(lipgloss.Style), new(lipgloss.Style), new(lipgloss.Style), new(lipgloss.Style)
	// The home screen's project path in its field (plan 51, design D4).
	stHomePath = new(lipgloss.Style)
	// The Git Graph's lanes, each in its colour (plan 62, design D9).
	stLane = func() (out [laneColours]*lipgloss.Style) {
		for i := range out {
			out[i] = new(lipgloss.Style)
		}
		return out
	}()

	// The summary suggestion's card (plan 70, brief amendment 73): its frame
	// in the thinking card's violet, softer than the pink of an approval,
	// which asks for an answer; its focused button's label bright, bold and
	// underlined, on no ground.
	stSummaryFrame = stThinkBorder
	stSummaryFocus = new(lipgloss.Style)

	stMdHeading = stTitle
	stMdMark    = stGrey // a quote's bars, a table's borders, a rule (plan 71)

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
	paintStyle(stPanelRule, fg(cTitle))

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
	paintStyle(stSummaryFocus, stBrightBold.Underline(true))
	paintStyle(stModelFrame, fg(cModelFrame))
	paintStyle(stModelHead, fg(cModelFrame).Bold(true))
	paintStyle(stModelAside, fg(cModelAside))
	paintStyle(stModelDesc, fg(cModelDesc))
	paintStyle(stModelMark, fg(cModelMark))
	paintStyle(stModelRule, fg(cModelRule))
	paintStyle(stModelTrackEnd, fg(cModelToggle))
	paintStyle(stModelOffEnd, fg(cModelOff))
	paintStyle(stModelDimEnd, fg(cModelDim))
	paintStyle(stModelKnob, fg(cModelKnob))
	paintStyle(stSliderTrackEnd, fg(cSliderTrack))
	paintStyle(stSliderFillEnd, fg(cSliderFill))
	paintStyle(stSliderLockEnd, fg(cSliderLockFill))
	paintStyle(stSliderTopLeft, fg(cSliderTopLeft))
	paintStyle(stSliderTopRight, fg(cSliderTopRight))
	paintStyle(stSliderDot, fg(cSliderDot))
	paintStyle(stSliderDotFill, fg(cSliderDotFill))
	paintStyle(stSliderDotTop, fg(cSliderDotTop))
	paintStyle(stSliderLockKnob, fg(cSliderLockKnob))
	paintStyle(stModesFrame, fg(cModesFrame))
	paintStyle(stModesHead, fg(cModesHead).Bold(true))
	paintStyle(stModesWarnHead, fg(cModesWarn).Bold(true))
	paintStyle(stModesAside, fg(cModesAside))
	paintStyle(stModesHalf, fg(cModesBand))
	paintStyle(stModesTitle, fg(cModesTitle))
	paintStyle(stModesDesc, fg(cModesDesc))
	paintStyle(stModesOff, fg(cModesOff))
	paintStyle(stModesWarn, fg(cModesWarn))
	paintStyle(stModesSelTitle, fg(cModesSelTitle))
	paintStyle(stModesSelDesc, fg(cModesSelDesc))
	paintStyle(stModesIcon, fg(cModesIcon))
	paintStyle(stSideIcon, fg(cSideIcon))
	paintStyle(stSideIconLit, fg(cSideIconLit))
	paintStyle(stSideAccent, fg(cSideAccent))
	paintStyle(stExpFolder, fg(cExpFolder))
	paintStyle(stExpFile, fg(cExpFile))
	paintStyle(stExpRowFolder, stExpFolder.Background(cTreeRow.get()))
	paintStyle(stExpRowFile, stExpFile.Background(cTreeRow.get()))
	paintExpIcons() // the Nerd Font icons' (plan 55)
	paintStyle(stExpDirName, fg(cExpDirName))
	paintStyle(stExpFileName, fg(cExpFileName))
	paintStyle(stExpRowDirName, stExpDirName.Background(cTreeRow.get()))
	paintStyle(stExpRowFileName, stExpFileName.Background(cTreeRow.get()))
	paintStyle(stTitleBold, fg(cTitle).Bold(true))
	paintStyle(stInfo, fg(cLink))
	paintStyle(stDiffAdd, fg(cDiffAdd))
	paintStyle(stDiffDel, fg(cDiffDel))
	for i, c := range laneColour {
		paintStyle(stLane[i], fg(c))
	}
	paintStyle(stMsgOutline, fg(cMsgOutline))
	paintStyle(stMsgFocus, fg(cMsgFocus))
	paintStyle(stMsgHolder, fg(cMsgHolder))
	paintStyle(stBtnOutline, fg(cBtnOutline))
	paintStyle(stBtnFocus, fg(cTitle))
	paintStyle(stBtnOutlineDim, fg(cBtnOutlineDim))
	paintStyle(stBtnLabel, fg(cBtnLabel))
	paintStyle(stBtnLabelDim, fg(cBtnLabelDim))
	paintStyle(stBtnSep, fg(cBtnSep))
	paintStyle(stBtnSepDim, fg(cBtnSepDim))
	paintStyle(stMenuOutline, fg(cMenuOutline))
	paintStyle(stMenuText, fg(cMenuText))
	paintStyle(stHomePath, fg(cHomePath))
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
	for k, st := range homeStyles {
		paintStyle(st, homeStyleOf(k))
	}
	for k, st := range mdStyles { // after their bases (plan 71)
		paintStyle(st, mdStyleOf(k))
	}
	for base, st := range cursorStyles { // after theirs, Markdown's among them
		paintStyle(st, base.Background(cTreeRow.get()))
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
	// activeGround is the "+" menu's: the band while the menu is open and
	// the item under the mouse (plan 30, design D2 and D3).
	activeGround = &ground{c: cActive, tints: map[*lipgloss.Style]*lipgloss.Style{}}
	// The model selector's grounds (plan 34): the cursor's band, the text
	// toggle's track on, off and unavailable (plan 35), the text slider's
	// track, fill and held fill (plan 37; its gradient's in slider.go).
	bandGround        = &ground{c: cModelBand, tints: map[*lipgloss.Style]*lipgloss.Style{}}
	trackGround       = &ground{c: cModelToggle, tints: map[*lipgloss.Style]*lipgloss.Style{}}
	offGround         = &ground{c: cModelOff, tints: map[*lipgloss.Style]*lipgloss.Style{}}
	dimGround         = &ground{c: cModelDim, tints: map[*lipgloss.Style]*lipgloss.Style{}}
	sliderTrackGround = &ground{c: cSliderTrack, tints: map[*lipgloss.Style]*lipgloss.Style{}}
	sliderFillGround  = &ground{c: cSliderFill, tints: map[*lipgloss.Style]*lipgloss.Style{}}
	sliderLockGround  = &ground{c: cSliderLockFill, tints: map[*lipgloss.Style]*lipgloss.Style{}}
	// The modes selector's grounds (plan 39): the box's and the band's.
	modesGround     = &ground{c: cModesGround, tints: map[*lipgloss.Style]*lipgloss.Style{}}
	modesBandGround = &ground{c: cModesBand, tints: map[*lipgloss.Style]*lipgloss.Style{}}
	// The sidebar's selected tile's ground, and a shade lighter while
	// hovered (plan 43, design D3); a hovered tile takes the model
	// selector's band, bandGround.
	sideSelGround      = &ground{c: cSideSel, tints: map[*lipgloss.Style]*lipgloss.Style{}}
	sideSelHoverGround = &ground{c: cSideSelHover, tints: map[*lipgloss.Style]*lipgloss.Style{}}
	// Source Control's hovered row, and the diff's added and removed lines'
	// grounds (plan 44, designs D4 and D8).
	sourceHoverGround = &ground{c: cSourceHover, tints: map[*lipgloss.Style]*lipgloss.Style{}}
	diffAddGround     = &ground{c: cDiffAddGround, tints: map[*lipgloss.Style]*lipgloss.Style{}}
	diffDelGround     = &ground{c: cDiffDelGround, tints: map[*lipgloss.Style]*lipgloss.Style{}}
	// The commit controls' grounds in text (plan 45, design D5): the
	// message box's inside, the button's fill and dimmed fill, the
	// dropdown's ground and band.
	msgGround      = &ground{c: cMsgInside, tints: map[*lipgloss.Style]*lipgloss.Style{}}
	btnGround      = &ground{c: cBtnFill, tints: map[*lipgloss.Style]*lipgloss.Style{}}
	btnDimGround   = &ground{c: cBtnFillDim, tints: map[*lipgloss.Style]*lipgloss.Style{}}
	menuGround     = &ground{c: cMenuGround, tints: map[*lipgloss.Style]*lipgloss.Style{}}
	menuBandGround = &ground{c: cMenuBand, tints: map[*lipgloss.Style]*lipgloss.Style{}}
	// The home screen's page and its project field's inside (plan 51); in
	// grid mode, the chosen option's inside, START SESSION's fill and its
	// dimmed fill, the list's band without and with the focus, a key cap's
	// inside.
	pageGround      = &ground{c: cHomePage, tints: map[*lipgloss.Style]*lipgloss.Style{}}
	fieldGround     = &ground{c: cHomeField, tints: map[*lipgloss.Style]*lipgloss.Style{}}
	chosenGround    = &ground{c: cHomeChosenFill, tints: map[*lipgloss.Style]*lipgloss.Style{}}
	buttonGround    = &ground{c: cHomeButton, tints: map[*lipgloss.Style]*lipgloss.Style{}}
	buttonDimGround = &ground{c: cHomeButtonDim, tints: map[*lipgloss.Style]*lipgloss.Style{}}
	rowBandGround   = &ground{c: cHomeBand, tints: map[*lipgloss.Style]*lipgloss.Style{}}
	rowFocusGround  = &ground{c: cHomeBandTop, tints: map[*lipgloss.Style]*lipgloss.Style{}}
	capGround       = &ground{c: cHomeCapFill, tints: map[*lipgloss.Style]*lipgloss.Style{}}
	grounds         = []*ground{toolGround, activeGround, bandGround, trackGround, offGround, dimGround, sliderTrackGround, sliderFillGround, sliderLockGround, modesGround, modesBandGround, sideSelGround, sideSelHoverGround, sourceHoverGround, diffAddGround, diffDelGround, msgGround, btnGround, btnDimGround, menuGround, menuBandGround, pageGround, fieldGround, chosenGround, buttonGround, buttonDimGround, rowBandGround, rowFocusGround, capGround}
)

// homeKey is a home screen text style: its colour, semi-bold or not.
type homeKey struct {
	c    colour
	bold bool
}

// homeStyles are the home screen's text styles in grid mode (plan 51),
// made on first use and painted again with the palette.
var homeStyles = map[homeKey]*lipgloss.Style{}

// homeStyle is the shared style of colour c, bold or not.
func homeStyle(c colour, bold bool) *lipgloss.Style {
	k := homeKey{c, bold}
	if st := homeStyles[k]; st != nil {
		return st
	}
	st := new(lipgloss.Style)
	paintStyle(st, homeStyleOf(k))
	homeStyles[k] = st
	return st
}

func homeStyleOf(k homeKey) lipgloss.Style { return fg(k.c).Bold(k.bold) }

// active is st on the active ground (plan 30).
func active(st *lipgloss.Style) *lipgloss.Style { return activeGround.on(st) }

// cursorStyles are styles on the file viewer's cursor line, its background
// zone A's cursor row's on every palette, as synRow's are (plan 71, design
// D5: a Markdown file's preview).
var cursorStyles = map[*lipgloss.Style]*lipgloss.Style{}

// onCursorRow is st on the viewer's cursor line, made on first use.
func onCursorRow(st *lipgloss.Style) *lipgloss.Style {
	if t := cursorStyles[st]; t != nil {
		return t
	}
	t := new(lipgloss.Style)
	paintStyle(t, st.Background(cTreeRow.get()))
	cursorStyles[st] = t
	return t
}

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
