package render

import (
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/Evengelius/agent-command-center/internal/state"
)

// The modes selector (plan 39, brief amendment 40; reference-design.md, The
// modes selector): the editor row's mode pill, at its right end, opens a
// box above it headed "MODES", the session's CLI's own modes, each with its
// icon, its title and its description; the mode in force on the band with a
// check, the cursor's (the pointer's or the keys') on the band without one.
// Codex's Full access is asked for only through a confirmation in the box's
// place (design D4). With Sixel the icons, the check and the band's rounded
// ends are two images over blank cells that never change (design D6), so a
// hover changes only cells between them.

// ModesView is the selector as the owner works it; the application owns it.
type ModesView struct {
	Open bool
	// Cursor is the option under the cursor, an index into the box's
	// options (ModesCount), -1 for none.
	Cursor int
	// Confirm says the Full access confirmation stands in the box's place:
	// its options Turn on Full access (0) and Cancel (1), Cancel where Full
	// access was, so a double click on Full access never turns it on.
	Confirm bool
	// Pointer and PointerKnown are the model selector's (plan 36, design
	// D1): the item under the pointer as the last report gave it, -1
	// outside the box.
	Pointer      int
	PointerKnown bool
}

// ModesShown is what the frame showed of the box; zero when it is closed.
type ModesShown struct {
	Shown bool
	// Strips says its strips' cells are blank for the images the
	// application draws (Sixel): the left strip's columns from LeftX, the
	// right one's from RightX, both from row Y down.
	Strips           bool
	LeftX, RightX, Y int
	Left, Right      StripLook
}

// ModePillShown is where the frame left the mode pill's image to be drawn
// (design D7), zero when there is none.
type ModePillShown struct {
	Shown bool
	X, Y  int
	Look  ModePillLook
}

// The box's panel, the pill's, and the items its cells carry: each
// option's, a half row's, the frame's and the heading's.
const (
	ModesPanel     = 'O'
	ModePillPanel  = 'Q'
	ItemModeOpt    = 3000 // plus the option's index
	ItemModesHalf  = 2999
	ItemModesFrame = 2998
	// itemCompactPill marks the compact mode pill's cells, the icon alone.
	itemCompactPill = 1
)

// The box's columns (design D5), from its left edge: the left strip from
// column 1, ten cells; the titles and descriptions from column 11; the
// right strip the five cells before the right edge; two cells between the
// longest description and the right strip.
const (
	modesStripL    = 10
	modesStripR    = 5
	modesTextCol   = 11
	modesTextSlack = 2
)

// modesConfirmHead is the confirmation's heading (design D4), and
// modesConfirmText what Full access allows, the dashboard's words.
const modesConfirmHead = "TURN ON FULL ACCESS?"

var modesConfirmText = []string{"Codex will run commands and edit", "any file on this computer, with", "internet access, without asking."}

// modesOpt is one option of the box as drawn.
type modesOpt struct {
	title, icon string
	desc        []string
	warning     bool
	mode        string // the mode it asks for; "" for the confirmation's Cancel
	in          bool   // the mode in force: the band and the check
	off         string // why the CLI's policy leaves it out: greyed, no band
}

// modesOptions are the box's options: the session's CLI's modes, or the
// confirmation's two.
func modesOptions(s *state.State, v ModesView) []modesOpt {
	cur := s.Mode()
	if v.Confirm {
		keep := s.ModeOption(cur)
		icon, title := state.IconHand, s.ModeTitle(cur)
		if keep != nil {
			icon = keep.Icon
		}
		return []modesOpt{
			{title: "Turn on Full access", icon: state.IconHexagon, desc: modesConfirmText, warning: true, mode: state.ModeFull},
			{title: "Cancel", icon: icon, desc: []string{"Keep " + title}},
		}
	}
	var out []modesOpt
	for _, o := range s.ModeOptions() {
		out = append(out, modesOpt{title: o.Title, icon: o.Icon, desc: o.Desc, warning: o.Warning, mode: o.Mode,
			in: o.Mode == cur, off: s.Session.ModesUnavailable[o.Mode]})
	}
	return out
}

// ModesStripCells is a strip's width in cells, the right one's or the
// left one's.
func ModesStripCells(right bool) int {
	if right {
		return modesStripR
	}
	return modesStripL
}

// ModesCount is how many options the box shows.
func ModesCount(s *state.State, v ModesView) int { return len(modesOptions(s, v)) }

// ModesMode is the mode option i asks for: one of the CLI's, or, in the
// confirmation, "" for Cancel and Full access's word.
func ModesMode(s *state.State, v ModesView, i int) string {
	opts := modesOptions(s, v)
	if i < 0 || i >= len(opts) {
		return ""
	}
	return opts[i].mode
}

// ModesOff is why the CLI's policy leaves option i out, "" when it takes
// a choice.
func ModesOff(s *state.State, v ModesView, i int) string {
	opts := modesOptions(s, v)
	if i < 0 || i >= len(opts) {
		return ""
	}
	return opts[i].off
}

// ModesHome is the cursor's option when neither the keys nor the pointer
// put it on another: the mode in force's, the confirmation's Cancel, else
// none (-1).
func ModesHome(s *state.State, v ModesView) int {
	for i, o := range modesOptions(s, v) {
		if v.Confirm && o.mode == "" || !v.Confirm && o.in && o.off == "" {
			return i
		}
	}
	return -1
}

// OpenModes is the selector as it opens: the cursor on the mode in force.
func OpenModes(s *state.State) ModesView {
	v := ModesView{Open: true}
	v.Cursor = ModesHome(s, v)
	return v
}

// OpenConfirm is the Full access confirmation as it opens: the cursor on
// Cancel (design D4).
func OpenConfirm(s *state.State) ModesView {
	v := ModesView{Open: true, Confirm: true}
	v.Cursor = ModesHome(s, v)
	return v
}

// modesWidth is the box's width: 19 cells and the longest description line
// of the session's CLI's modes (51 for Claude Code's, 54 for Codex's): the
// text's column, the line, the slack, the right strip and the right edge.
func modesWidth(s *state.State) int {
	w := 0
	for _, o := range s.ModeOptions() {
		for _, l := range o.Desc {
			w = max(w, textWidth(l))
		}
		w = max(w, textWidth(o.Title))
	}
	return modesTextCol + w + modesTextSlack + modesStripR + 1
}

// modesRow is one row of the box between its edges: the header ('h'), a
// half row ('g') between the options above and below it (-1 for none), an
// option's title ('t') or one of its description lines ('d').
type modesRow struct {
	kind         byte
	opt, line    int
	above, below int
}

// modesLayout is the box's rows between its edges (design D5): the header,
// then each option after a half row, its title and its description lines,
// and a half row after the last.
func modesLayout(opts []modesOpt) []modesRow {
	rows := []modesRow{{kind: 'h', opt: -1, above: -1, below: -1}}
	for i, o := range opts {
		rows = append(rows, modesRow{kind: 'g', opt: -1, above: i - 1, below: i})
		rows = append(rows, modesRow{kind: 't', opt: i, above: -1, below: -1})
		for j := range o.desc {
			rows = append(rows, modesRow{kind: 'd', opt: i, line: j, above: -1, below: -1})
		}
	}
	return append(rows, modesRow{kind: 'g', opt: -1, above: len(opts) - 1, below: -1})
}

// ModesHeight is the box's height, its edges included.
func ModesHeight(s *state.State, v ModesView) int { return len(modesLayout(modesOptions(s, v))) + 2 }

// modesPlace is the box's left edge and top row above the mode pill: its
// right edge on the cell after the pill, zone B's margin left free, or on
// zone B's right edge, never over the Explorer's panel or the sidebar (plan
// 43, design D9); its bottom edge on the editor's rule (design D5). ok is
// false when the pill or the rule is not on screen, or the box does not
// fit between zone B's title and the rule (design D9).
func (sc *Screen) modesPlace(s *state.State, v ModesView, width int) (x, y, tail int, ok bool) {
	px, py, found := sc.find(ModePillPanel)
	if !found {
		return 0, 0, 0, false
	}
	ry := sc.editorRule(py)
	if ry < 0 {
		return 0, 0, 0, false
	}
	w, h := modesWidth(s), ModesHeight(s, v)
	pw := sc.spanWidth(ModePillPanel, py)
	x = min(px+pw, sc.bRight(width)-1) - (w - 1)
	y = ry + 1 - h
	if x < 0 || y < sc.selectorTop() {
		return 0, 0, 0, false
	}
	tail = min(max(px+pw/2-x, 1), w-2) // the "▾", above the pill's middle cell
	return x, y, tail, true
}

// ModesFit says the box fits above the mode pill on this screen (design D9).
func (sc *Screen) ModesFit(s *state.State, v ModesView, width int) bool {
	_, _, _, ok := sc.modesPlace(s, v, width)
	return ok && len(s.ModeOptions()) > 0
}

// ModePillOn says the frame shows the mode pill.
func (sc *Screen) ModePillOn() bool {
	_, _, ok := sc.find(ModePillPanel)
	return ok
}

// spanWidth is the width of row y's spans of panel p.
func (sc *Screen) spanWidth(p byte, y int) int {
	n := 0
	if y >= 0 && y < len(sc.lines) {
		for _, s := range sc.lines[y] {
			if s.ref.panel == p {
				n += textWidth(s.text)
			}
		}
	}
	return n
}

// modesBox draws the selector over the dashboard when it is open (design
// D5): its frame, "MODES" or the confirmation's heading and "Current
// session", the options with their half rows, and, with Sixel, the strips'
// blank cells whose images the application draws (design D6).
func modesBox(sc *Screen, s *state.State, ui UI) {
	v := ui.Modes
	x, y, tail, ok := sc.modesPlace(s, v, ui.Width)
	if !ok {
		return
	}
	opts := modesOptions(s, v)
	w := modesWidth(s)
	inner := w - 2
	lit := func(i int) bool {
		return i >= 0 && i < len(opts) && opts[i].off == "" && (opts[i].in && !v.Confirm || i == v.Cursor)
	}
	rows := modesLayout(opts)
	box := []line{{modesDeco("╭"+strings.Repeat("─", inner)+"╮", modesGround.on(stModesFrame), ItemModesFrame)}}
	for _, r := range rows {
		box = append(box, modesRowLine(r, opts, v, inner, lit, ui.Sixel))
	}
	box = append(box, line{modesDeco("╰"+strings.Repeat("─", tail-1)+"▾"+strings.Repeat("─", max(inner-tail, 0))+"╯", modesGround.on(stModesFrame), ItemModesFrame)})
	sc.place(box, x, y, w)
	sc.Modes = ModesShown{Shown: true}
	if ui.Sixel {
		left, right := StripLook{Rows: len(rows) - 1}, StripLook{Rows: len(rows) - 1, Right: true}
		for i, r := range rows[1:] {
			if r.kind != 't' {
				continue
			}
			o := opts[r.opt]
			selected := o.in && !v.Confirm && o.off == ""
			so := StripOption{Row: i, Rows: 1 + len(o.desc), Icon: o.icon, Warning: o.warning, Off: o.off != "",
				Selected: selected, Cursor: !selected && lit(r.opt)}
			left.Options = append(left.Options, so)
			right.Options = append(right.Options, so)
		}
		sc.Modes = ModesShown{Shown: true, Strips: true, LeftX: x + 1, RightX: x + w - 1 - modesStripR, Y: y + 2, Left: left, Right: right}
	}
}

// modesRowLine is one row of the box between its edges, w cells inside the
// frame: the strips' cells, blank on the box's ground with Sixel, the icon
// and the check in text without; between them the row's text on the band
// when its option is lit, or a half row's halves ("▀" the band of the
// option above, "▄" of the one below, "█" both), which run from edge to
// edge without Sixel.
func modesRowLine(r modesRow, opts []modesOpt, v ModesView, w int, lit func(int) bool, sixel bool) line {
	edge := modesDeco("│", modesGround.on(stModesFrame), ItemModesFrame)
	item := ItemModesHalf
	if r.opt >= 0 {
		item = ItemModeOpt + r.opt
	}
	if r.kind == 'h' {
		head, st := "MODES", stModesHead
		if v.Confirm {
			head, st = modesConfirmHead, stModesWarnHead
		}
		// The gap between the two on the box's ground, as every cell of it.
		left := line{modesDeco("  ", modesGround.on(stNone), ItemModesFrame), modesDeco(head, modesGround.on(st), ItemModesFrame)}
		right := line{modesDeco("Current session", modesGround.on(stModesAside), ItemModesFrame), modesDeco("  ", modesGround.on(stNone), ItemModesFrame)}
		gap := modesDeco(strings.Repeat(" ", max(w-left.width()-right.width(), 0)), modesGround.on(stNone), ItemModesFrame)
		return append(append(append(append(line{edge}, left...), gap), right...), edge)
	}
	textW := w - modesStripL - modesStripR // the cells between the strips
	left := modesDeco(strings.Repeat(" ", modesStripL), modesGround.on(stNone), item)
	right := modesDeco(strings.Repeat(" ", modesStripR), modesGround.on(stNone), item)
	var mid span
	if r.kind == 'g' {
		glyph := " "
		switch up, down := lit(r.above), lit(r.below); {
		case up && down:
			glyph = "█"
		case up:
			glyph = "▀"
		case down:
			glyph = "▄"
		}
		mid = modesDeco(strings.Repeat(glyph, textW), modesGround.on(stModesHalf), item)
		if !sixel {
			left = modesDeco(strings.Repeat(glyph, modesStripL), modesGround.on(stModesHalf), item)
			right = modesDeco(strings.Repeat(glyph, modesStripR), modesGround.on(stModesHalf), item)
		}
		return line{edge, left, mid, right, edge}
	}
	o := opts[r.opt]
	ground := modesGround
	if lit(r.opt) {
		ground = modesBandGround
	}
	text, st := o.title, stModesTitle
	if r.kind == 'd' {
		text, st = o.desc[r.line], stModesDesc
	}
	selected := o.in && !v.Confirm
	switch {
	case o.off != "":
		st = stModesOff
	case o.warning && r.kind == 't':
		st = stModesWarn
	case selected && r.kind == 't':
		st = stModesSelTitle
	case selected:
		st = stModesSelDesc
	}
	mid = modesDeco(padCell(text, textW), ground.on(st), item)
	if !sixel {
		left, right = modesTextStrip(r, opts, v, lit, item)
	}
	return line{edge, left, mid, right, edge}
}

// modesTextStrip is a row's strips in text, without Sixel: the icon on the
// title row, the check of the mode in force, on the band when the option
// is lit.
func modesTextStrip(r modesRow, opts []modesOpt, v ModesView, lit func(int) bool, item int) (span, span) {
	o := opts[r.opt]
	ground := modesGround
	if lit(r.opt) {
		ground = modesBandGround
	}
	icon, check := "", ""
	if r.kind == 't' {
		icon = modesTextIcon[o.icon]
		if o.in && !v.Confirm && o.off == "" {
			check = "✓"
		}
	}
	ist := stModesIcon
	if o.warning {
		ist = stModesWarn
	}
	left := modesDeco(padCell("   "+icon, modesStripL), ground.on(ist), item)
	right := modesDeco(padCell(" "+check, modesStripR), ground.on(stModesSelTitle), item)
	return left, right
}

// modesTextIcon is each icon in text, without Sixel.
var modesTextIcon = map[string]string{state.IconHand: "✋", state.IconCode: "</>", state.IconPlan: "▤",
	state.IconBolt: "↯", state.IconShield: "⛨", state.IconHexagon: "⚠"}

// modesDeco is a span of the box: decoration of panel O, carrying item.
func modesDeco(text string, st *lipgloss.Style, item int) span {
	return span{text: text, st: st, ref: ref{panel: ModesPanel, kind: refDeco, item: item}}
}

// modePillLook is the mode pill's look: the mode in force's icon and
// title, the warning colour for Full access.
func modePillLook(s *state.State) ModePillLook {
	mode := s.Mode()
	look := ModePillLook{Title: s.ModeTitle(mode)}
	if o := s.ModeOption(mode); o != nil {
		look.Icon, look.Warning = o.Icon, o.Warning
	}
	return look
}

// modePill is the editor row's right end (design D7): the mode pill and one
// cell, panel Q; compact, the icon alone, where a middle text leaves no room
// for the title. With Sixel its cells are blank, the pill's image drawn over
// them; without, " <icon> <title> " on the pill's ground.
func modePill(s *state.State, sixel, compact bool) line {
	look := modePillLook(s)
	ref := ref{panel: ModePillPanel, kind: refDeco}
	if compact {
		look.Title, ref.item = "", itemCompactPill
	}
	if sixel {
		return line{span{text: strings.Repeat(" ", ModePillCells(look)), st: stNone, ref: ref}, deco(" ", stNone)}
	}
	text := " " + look.Title + " "
	if icon := modesTextIcon[look.Icon]; icon != "" && look.Title == "" {
		text = " " + icon + " "
	} else if icon != "" {
		text = " " + icon + text
	}
	l := grounded(line{span{text: text, st: stText, ref: ref}}, textWidth(text))
	return append(l, deco(" ", stNone))
}
