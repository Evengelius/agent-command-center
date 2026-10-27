package render

import (
	"math"
	"strings"

	"github.com/Evengelius/agent-command-center/internal/state"
)

// Grid mode (plan 51, design D1): the home screen's structure in terminal
// text, on whole rows of one size, frames in box drawing, in the design's
// colours (reference-design.md, The home screen): the header and a blank
// row, the banner's rows when it shows (design D2), the two panels side by
// side, a blank row and the footer.

const (
	// gridRows is the least screen in grid mode: the header and its blank
	// row, the panels' least height, the footer and the blank row above it.
	gridRows = 2 + gridPanelRows + 2
	// gridPanelRows are the panels' least height, their frames included:
	// NEW SESSION's rows with no spacing.
	gridPanelRows = 14
	// gridBannerTop is the banner's first row.
	gridBannerTop = 2
)

// gridRow is a row of a panel's inside: its line, or a blank row the
// panel's spare height may give, spacer its order (1 first), 0 for a row
// always drawn.
type gridRow struct {
	l      line
	spacer int
}

// homeGrid draws the home screen in grid mode on a w by h screen, and gives
// the geometry it drew: the banner gives the panels the rows they need,
// hidden below bannerRows (design D6, the panel growing into the banner).
func homeGrid(s *state.State, ui UI, g HomeGeom) ([]line, HomeGeom) {
	w, h := ui.Width, ui.Height
	lw := int(math.Round(float64(w-3) * 608.5 / 1308.5)) // the design's panels' widths
	rw := w - 3 - lw
	left := homeSessionRows(s, ui, lw-2)
	need := gridNeed(left)
	if need > h-2 {
		// Messages too long for NEW SESSION beside PROJECTS: a message is
		// never cut, so NEW SESSION takes the width, PROJECTS under it.
		return homeGridStacked(s, ui, g)
	}
	// The panels from the banner's end, or from under the header, down to
	// the blank row above the footer; taller when NEW SESSION needs it,
	// the banner giving its rows first, then the blank rows around.
	top, bottom := gridBannerTop, h-2
	if g.Banner {
		top += g.BannerRows
	}
	if over := need - (bottom - top); over > 0 && g.Banner {
		g.BannerRows -= over
		top -= over
		if g.BannerRows < bannerRows {
			g.Banner, g.BannerRows, top = false, 0, gridBannerTop
		}
	}
	if bottom-top < need {
		top = max(top-1, 1) // the header's blank row
	}
	if bottom-top < need {
		bottom = h - 1 // the footer's blank row
	}
	ph := bottom - top
	ls := make([]line, 0, h)
	ls = append(ls, homeGridHeader(s, ui, w))
	for len(ls) < top {
		ls = append(ls, homeBlank(w))
	}
	lp := homePanel("NEW SESSION", fitGridRows(left, ph-3), lw)
	rp := homePanel("PROJECTS", homeProjectRows(ui, rw-2, ph-3), rw)
	for i := range ph {
		l := line{sp(" ", pageGround.on(stNone))}
		l = append(append(l, lp[i]...), sp(" ", pageGround.on(stNone)))
		ls = append(ls, append(append(l, rp[i]...), sp(strings.Repeat(" ", max(w-3-lw-rw+1, 0)), pageGround.on(stNone))))
	}
	for len(ls) < h-1 {
		ls = append(ls, homeBlank(w))
	}
	return append(ls, homeGridFooter(ui, w)), g
}

// gridNeed is the height a panel's rows need, its frame and title
// included, without its spacers.
func gridNeed(rows []gridRow) int {
	need := 3
	for _, r := range rows {
		if r.spacer == 0 {
			need++
		}
	}
	return need
}

// homeGridStacked is grid mode when NEW SESSION's messages need more rows
// than the screen has beside PROJECTS: NEW SESSION across the screen under
// the header, PROJECTS under it in the rows left when they hold a project,
// no banner.
func homeGridStacked(s *state.State, ui UI, g HomeGeom) ([]line, HomeGeom) {
	w, h := ui.Width, ui.Height
	g.Banner, g.BannerRows = false, 0
	left := homeSessionRows(s, ui, w-4)
	lh := min(gridNeed(left), h-2)
	rest := h - 2 - lh // the rows from under the header to the footer
	ls := []line{homeGridHeader(s, ui, w)}
	panel := func(p []line, pw int) {
		for _, l := range p {
			ls = append(ls, append(append(line{sp(" ", pageGround.on(stNone))}, l...), sp(strings.Repeat(" ", max(w-1-pw, 0)), pageGround.on(stNone))))
		}
	}
	panel(homePanel("NEW SESSION", fitGridRows(left, lh-3), w-2), w-2)
	if rest >= 5 {
		panel(homePanel("PROJECTS", homeProjectRows(ui, w-4, rest-3), w-2), w-2)
	}
	for len(ls) < h-1 {
		ls = append(ls, homeBlank(w))
	}
	return append(ls, homeGridFooter(ui, w)), g
}

// homeTag gives every span of l but its first, the panel's margin, item:
// what a click there does (design D11).
func homeTag(l line, item int) line {
	for i := 1; i < len(l); i++ {
		l[i].ref = ref{panel: HomePanel, kind: refDeco, item: item}
	}
	return l
}

// homeBlank is a blank row on the page's ground.
func homeBlank(w int) line { return line{sp(strings.Repeat(" ", w), pageGround.on(stNone))} }

// homeOn is a span of text in colour c, bold or not, on ground gr.
func homeOn(text string, c colour, bold bool, gr *ground) span {
	return sp(text, gr.on(homeStyle(c, bold)))
}

// homeRow is a panel's inside row of iw cells: l, cut to fit, the rest
// blank on the page's ground.
func homeRow(l line, iw int) line {
	l = l.fit(iw)
	for i := range l {
		if l[i].st == stNone || l[i].st == nil {
			l[i].st = pageGround.on(stNone)
		}
	}
	return l
}

// homeGridHeader is the header's row: the logo and the title, and at the
// right the version, the system and the status between bars (design D5);
// a mock run's badge in the status's place.
func homeGridHeader(s *state.State, ui UI, w int) line {
	left := line{homeOn(" ⊞ ", cHomeLogo, false, pageGround), homeOn("AGENT COMMAND CENTER", cHomeTitle, false, pageGround)}
	status, kind := homeStatus(s)
	dot := cHomeReady
	switch kind {
	case 'a':
		dot = cProgress
	case 'r':
		dot = cFailed
	}
	bar := homeOn("  │  ", cHomeBar, false, pageGround)
	right := line{homeOn("│  ", cHomeBar, false, pageGround), homeOn("v"+ui.Version, cHomeVersion, false, pageGround), bar,
		homeOn(ui.System, cHomeSystem, false, pageGround), bar, homeOn("● ", dot, false, pageGround), homeOn(status+" ", cHomeStatus, false, pageGround)}
	if s.Context.Mock {
		right = line{sp(MockBadge+" ", pageGround.on(stProgress))}
	}
	return homeRow(spread(left, right, w), w)
}

// homeGridFooter is the footer's row (design D8): "Shortcuts:", then each
// shortcut's key on its cap and its action, bars between them.
func homeGridFooter(ui UI, w int) line {
	l := line{homeOn(" › ", cHomeFootChevron, false, pageGround), homeOn("Shortcuts:", cHomeShortcuts, true, pageGround), homeOn("  ", cHomePage, false, pageGround)}
	for i, k := range homeKeys {
		if i > 0 {
			l = append(l, homeOn("  │  ", cHomeFootBar, false, pageGround))
		}
		l = append(l, homeOn(" "+k.Key+" ", cHomeCapText, false, capGround), homeOn(" "+k.Action, cHomeAction, false, pageGround))
	}
	return homeRow(l, w)
}

// homePanel is a panel's rows, its frame around the inside rows of a w
// cells wide panel: its first inside row the title, a chevron before it
// and the purple rule after.
func homePanel(title string, rows []line, w int) []line {
	iw := w - 2
	frame := homeStyle(cHomeFrame, false)
	edge := func(l, r string) line {
		return line{sp(l+strings.Repeat("─", iw)+r, pageGround.on(frame))}
	}
	head := line{homeOn(" › ", cHomeChevron, false, pageGround), homeOn(title, cHomePanelTitle, false, pageGround), homeOn("  ", cHomePage, false, pageGround)}
	head = append(head, homeOn(strings.Repeat("─", max(iw-head.width()-1, 0)), cHomeRule, false, pageGround))
	out := []line{edge("╭", "╮"), append(append(line{sp("│", pageGround.on(frame))}, homeRow(head, iw)...), sp("│", pageGround.on(frame)))}
	for _, r := range rows {
		out = append(out, append(append(line{sp("│", pageGround.on(frame))}, homeRow(r, iw)...), sp("│", pageGround.on(frame))))
	}
	return append(out, edge("╰", "╯"))
}

// fitGridRows are a panel's rows in n: every row always drawn, then the
// spacers in their order while they fit, blank rows below the rest.
func fitGridRows(rows []gridRow, n int) []line {
	spare := n
	for _, r := range rows {
		if r.spacer == 0 {
			spare--
		}
	}
	take := map[int]bool{}
	for order := 1; spare > 0; order++ {
		found := false
		for i, r := range rows {
			if r.spacer == order && spare > 0 {
				take[i], spare, found = true, spare-1, true
			}
		}
		if !found && order > len(rows) {
			break
		}
	}
	var out []line
	for i, r := range rows {
		if r.spacer == 0 || take[i] {
			out = append(out, r.l)
		}
	}
	for len(out) < n {
		out = append(out, nil)
	}
	return out[:max(n, 0)]
}

// homeSessionRows are NEW SESSION's rows under its title, within iw
// cells: the questions, the agent CLIs' options, the project field, the
// messages and START SESSION (designs D6 and D10), and the blank rows
// spare height gives, as the design spaces them.
func homeSessionRows(s *state.State, ui UI, iw int) []gridRow {
	blank := func(order int) gridRow { return gridRow{spacer: order} }
	ask := func(n, text string, focused bool) gridRow {
		c := cHomeAsk
		if focused {
			c = cHomeFocus
		}
		return gridRow{l: line{homeOn("  "+n+"  ", cHomeNumber, false, pageGround), homeOn(text, c, focused, pageGround)}}
	}
	rows := []gridRow{blank(1), ask("1.", "Agent CLI", ui.Field == 0)}
	for i, c := range homeCLIs(s, ui) {
		if i > 0 {
			rows = append(rows, blank(3))
		}
		for _, l := range homeOption(c, ui.Field == 0 && c.Chosen, iw) {
			rows = append(rows, gridRow{l: homeTag(l, HomeItemOption+i)})
		}
	}
	rows = append(rows, blank(1), ask("2.", "Project directory", ui.Field == 1))
	for _, l := range homeField(ui, iw) {
		// The folder button's cells, from its divider, open the folder
		// dialog (plan 52).
		l = homeTag(l, HomeItemField)
		for x, i := 0, 0; i < len(l); i++ {
			if x >= iw-6 {
				l[i].ref.item = HomeItemFolder
			}
			x += textWidth(l[i].text)
		}
		rows = append(rows, gridRow{l: l})
	}
	rows = append(rows, blank(4))
	for _, l := range homeMessages(s, ui, iw) {
		rows = append(rows, gridRow{l: l})
	}
	rows = append(rows, blank(2), gridRow{l: homeTag(homeButton(s, ui, iw), HomeItemStart)}, blank(5))
	return rows
}

// homeOption is an agent CLI's option: its radio, its name and its version
// on the chosen one's ground, the version wrapped under itself when it is
// a reason; with the focus, the band's ends in the focus's purple.
func homeOption(c HomeCLI, focused bool, iw int) []line {
	gr, radio, rc := pageGround, "○", cHomeRadioOff
	name, ver := cHomeOtherName, cHomeOtherVer
	if c.Chosen {
		gr, radio, rc, name, ver = chosenGround, "◉", cHomeRadioDot, cHomeChosenName, cHomeChosenVer
	}
	if !c.Usable {
		name = cHomeOtherVer
	}
	left, right := " ", " "
	if focused {
		left, right = "▏", "▕"
	}
	// The version after the name, a reason going on under the name: the
	// first line's words those that fit after it.
	const nameCol = 6 // the band's start, its edge, the radio
	tw := iw - 3 - nameCol
	first, rest := c.Version, ""
	if w := tw - textWidth(c.Name) - 1; textWidth(c.Version) > w {
		words := strings.Fields(c.Version)
		n, used := 0, 0 // the first line's words and cells
		for ; n < len(words); n++ {
			add := textWidth(words[n]) + min(n, 1)
			if used+add > w {
				break
			}
			used += add
		}
		first, rest = strings.Join(words[:n], " "), strings.Join(words[n:], " ")
	}
	texts := []string{first}
	if rest != "" {
		texts = append(texts, wrap(rest, tw)...)
	}
	var out []line
	for i, t := range texts {
		l := line{sp(" ", pageGround.on(stNone)), homeOn(left, cHomeFocus, false, gr)}
		if i == 0 {
			l = append(l, homeOn(" "+radio+"  ", rc, false, gr), homeOn(c.Name+" ", name, false, gr))
		} else {
			l = append(l, homeOn(strings.Repeat(" ", nameCol-2), ver, false, gr))
		}
		l = append(l, homeOn(t, ver, false, gr))
		l = append(l, homeOn(strings.Repeat(" ", max(iw-2-l.width(), 0)), ver, false, gr), homeOn(right, cHomeFocus, false, gr))
		out = append(out, l)
	}
	return out
}

// homeField is the project field in box drawing (design D4's text, as
// today's): its path cut at its start, its cursor with the focus, the
// folder button at its right; its outline the focus's purple with it.
func homeField(ui UI, iw int) []line {
	top, low := cHomeFieldTop, cHomeFieldLow
	if ui.Field == 1 {
		top, low = cHomeFocus, cHomeFocus
	}
	fw := iw - 2 // from the box's left edge to its right
	tw := fw - 8 // the text's cells, between "│ " and " │ 🗀 │"
	text, cursor := ui.Path, ""
	if ui.Field == 1 {
		cursor = "█"
	}
	if textWidth(text)+textWidth(cursor) > tw {
		text = "…" + tailCut(text, tw-1-textWidth(cursor))
	}
	in := line{homeOn(text, cHomePath, false, fieldGround), homeOn(cursor, cHomeFocus, false, fieldGround)}
	in = append(in, homeOn(strings.Repeat(" ", max(tw-in.width(), 0)), cHomePath, false, fieldGround))
	side := func(c colour, s string) span { return homeOn(s, c, false, pageGround) }
	return []line{
		{sp(" ", pageGround.on(stNone)), side(top, "╭"+strings.Repeat("─", fw-6)+"┬───╮")},
		append(append(line{sp(" ", pageGround.on(stNone)), side(top, "│"), homeOn(" ", cHomePath, false, fieldGround)}, in...),
			homeOn(" ", cHomePath, false, fieldGround), side(top, "│"), homeOn(" 🗀 ", cHomeFolder, false, fieldGround), side(top, "│")),
		{sp(" ", pageGround.on(stNone)), side(low, "╰"+strings.Repeat("─", fw-6)+"┴───╯")},
	}
}

// homeMessages are the notes between the field and START SESSION (design
// D6), each wrapped whole, never cut, in the panel's width.
func homeMessages(s *state.State, ui UI, iw int) []line {
	var out []line
	for _, n := range homeNotes(s, ui) {
		out = append(out, homeNoteRows(n, iw)...)
	}
	return out
}

// homeNoteRows are a note's rows in a panel iw cells wide, from its third
// cell, its lines after the first under its text.
func homeNoteRows(n HomeNote, iw int) []line {
	st := stGrey
	switch n.Kind {
	case 'f':
		st = stFailed
	case 'a':
		st = stAttention
	}
	var out []line
	mark := n.Mark
	for i, t := range wrap(n.Text, iw-4-textWidth(mark)) {
		if i > 0 {
			mark = strings.Repeat(" ", textWidth(mark))
		}
		out = append(out, line{sp("  ", pageGround.on(stNone)), sp(mark+t, pageGround.on(st))})
	}
	return out
}

// homeButton is START SESSION, its label and arrow centred on its fill,
// dimmed while there is nothing to start or a start runs; with the focus,
// its ends in the focus's colour.
func homeButton(s *state.State, ui UI, iw int) line {
	gr, text := buttonGround, cHomeButtonText
	if homeDim(s) {
		gr, text = buttonDimGround, cHomeButtonTextDim
	}
	label := "START SESSION →"
	if sixteen {
		text = cHomeButton // no ground: the label in the button's colour
	}
	left, right := " ", " "
	if ui.Field == 2 {
		left, right = "▏", "▕"
	}
	bw := iw - 2
	pad := max(bw-2-textWidth(label), 0)
	return line{sp(" ", pageGround.on(stNone)), homeOn(left, cHomeFocusOnButton, false, gr),
		homeOn(strings.Repeat(" ", pad/2)+label+strings.Repeat(" ", pad-pad/2), text, true, gr), homeOn(right, cHomeFocusOnButton, false, gr)}
}

// homeProjectRows are PROJECTS' n rows under its title, within iw cells
// (design D7): FAVORITES then RECENT PROJECTS, a section with no project
// left out; the selected project on the band, brighter with the focus;
// separators between the rows while they all fit with them, the list
// following the selection when they do not, a slim mark at its right
// saying where it is; the store's problem, or "No projects yet".
func homeProjectRows(ui UI, iw, n int) []line {
	out := []line{nil}
	if note := homeListNote(ui); note.Text != "" {
		out = append(out, homeNoteRows(note, iw)...)
	}
	if len(ui.Projects) == 0 {
		return fitGridRows(gridLines(out), n)
	}
	type item struct {
		l       line
		project bool
		sel     bool
	}
	var items []item
	// The names from the sixth cell, as wide as the longest, from 12 to 24
	// cells; the ages as wide as the longest, up to 10, ending at the
	// band's end; the paths in what is left between, from 10 cells, else
	// none and the names wider.
	nameW, ageW := 12, 6
	for _, p := range ui.Projects {
		nameW, ageW = max(nameW, textWidth(base(hideUser(p.Path)))), max(ageW, textWidth(p.Age))
	}
	nameW, ageW = min(nameW, 24), min(ageW, 10)
	ageCol := iw - 2 - ageW
	pathCol := 5 + nameW + 2
	pathW := ageCol - 1 - pathCol
	if pathW < 10 {
		nameW, pathW = ageCol-1-5, 0
	}
	for _, fav := range []bool{true, false} {
		head := false
		for i, p := range ui.Projects {
			if p.Favourite != fav {
				continue
			}
			if !head {
				icon, ic, title := "☆", cHomeSectionIcon, "FAVORITES"
				if !fav {
					icon, ic, title = "◷", cHomeClock, "RECENT PROJECTS"
				}
				if len(items) > 0 {
					items = append(items, item{}) // a blank row between the sections
				}
				items = append(items, item{l: line{sp("  ", pageGround.on(stNone)), homeOn(icon+" ", ic, false, pageGround), homeOn(title, cHomeSection, false, pageGround)}})
				head = true
			}
			sel := i == ui.ProjectSel
			l := homeTag(homeProjectRow(p, sel, sel && ui.Field == 3, iw, nameW, pathCol, pathW, ageW), HomeItemRow+i)
			l[2].ref.item = HomeItemStar + i // the star
			items = append(items, item{l: l, project: true, sel: sel})
		}
	}
	// With separators between a section's rows when they all fit, else
	// without; else the rows around the selection.
	rest := n - len(out)
	var rows []line
	seps := 0
	for i := 1; i < len(items); i++ {
		if items[i].project && items[i-1].project {
			seps++
		}
	}
	if len(items)+seps <= rest {
		for i, it := range items {
			if i > 0 && it.project && items[i-1].project {
				sep := homeOn(strings.Repeat("─", iw-2), cHomeSeparator, false, pageGround)
				if it.sel || items[i-1].sel {
					sep = homeOn(strings.Repeat(" ", iw-2), cHomeSeparator, false, pageGround)
				}
				rows = append(rows, line{sp(" ", pageGround.on(stNone)), sep})
			}
			rows = append(rows, it.l)
		}
		return fitGridRows(gridLines(append(out, rows...)), n)
	}
	first := 0
	if len(items) > rest {
		at := 0
		for i, it := range items {
			if it.sel {
				at = i
			}
		}
		first = min(max(at-rest/2, 0), len(items)-rest)
		first = max(first, 0)
	}
	for i := first; i < len(items) && len(rows) < rest; i++ {
		rows = append(rows, items[i].l)
	}
	if len(items) > rest && rest > 0 {
		// The slim mark: its length and place those of the rows shown.
		size := max(rest*rest/len(items), 1)
		at := first * rest / len(items)
		for i := range rows {
			mark := " "
			if i >= at && i < at+size {
				mark = "▐"
			}
			rows[i] = append(homeRow(rows[i], iw-1), homeOn(mark, cHomeFrame, false, pageGround))
		}
	}
	return fitGridRows(gridLines(append(out, rows...)), n)
}

// gridLines are lines as rows always drawn.
func gridLines(ls []line) []gridRow {
	out := make([]gridRow, len(ls))
	for i, l := range ls {
		out[i] = gridRow{l: l}
	}
	return out
}

// homeProjectRow is a project's row: its star, its name, its path and its
// age at the right, on the band when selected, a brighter one with the
// focus.
func homeProjectRow(p ProjectRow, sel, focused bool, iw, nameW, pathCol, pathW, ageW int) line {
	gr, nc, pc, ac := pageGround, cHomeName, cHomeRowPath, cHomeAge
	switch {
	case focused:
		gr, nc, pc, ac = rowFocusGround, cHomeNameSel, cHomeRowPathSel, cHomeAgeSel
	case sel:
		gr, nc, pc, ac = rowBandGround, cHomeNameSel, cHomeRowPathSel, cHomeAgeSel
	}
	star, sc := "☆", cHomeStarOff
	if p.Favourite {
		star, sc = "★", cHomeStarOn
	}
	name := base(hideUser(p.Path))
	if name == "" {
		name = p.Path // a drive's root
	}
	left, right := " ", " "
	if focused {
		left, right = "▏", "▕"
	}
	l := line{sp(" ", pageGround.on(stNone)), homeOn(left, cHomeFocus, false, gr), homeOn(star+"  ", sc, false, gr), homeOn(fitChars(name, nameW), nc, focused, gr)}
	if pathW > 0 {
		l = append(l, homeOn(strings.Repeat(" ", max(pathCol-l.width(), 1)), nc, false, gr), homeOn(elide(p.Path, pathW), pc, false, gr))
	}
	age := fitChars(p.Age, ageW)
	l = append(l, homeOn(strings.Repeat(" ", max(iw-2-textWidth(age)-l.width(), 1)), ac, false, gr), homeOn(age, ac, false, gr))
	return append(l, homeOn(right, cHomeFocus, false, gr))
}
