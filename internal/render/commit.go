package render

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
)

// Source Control's commit controls (plan 45, brief amendment 46; the
// owner's image references/15-commit-push.png): under the repository row
// the message field, then the Commit button with its separator and
// chevron; under the button, only once the chevron opened it, the dropdown
// with its one item, "Commit & Push"; then the note or the question
// (designs D1 to D6, D12). The application owns the field's editor and the
// operations; this draws them, in text or, with Sixel, as images over blank
// cells (design D5).

// The controls' items in panel G (design D1); the list's rows are 1 and
// up, the header's "⌄" ItemPanelHide.
const (
	ItemMsg        = -100 // the message box's outline, padding and placeholder
	ItemMsgText    = -101 // the message's text, its line and offset in the span
	ItemCommitMain = -102 // the button's left part
	ItemCommitMore = -103 // the button's chevron part, the separator at its left
	ItemCommitMenu = -104 // the dropdown
	ItemAskYes     = -105 // the question's "Stage and commit"
	ItemAskNo      = -106 // the question's "Cancel"
	ItemNotice     = -107 // the note's rows
	ItemMsgHolder  = -108 // the placeholder, "Message"
)

// Where the panel's keys go (design D2, D3): the list (""), the message
// field, the button.
const (
	PlaceMsg = "msg"
	PlaceBtn = "btn"
)

// The button's actions (design D7).
const (
	ActionCommit = "commit"
	ActionPush   = "push"
)

// The notes' kinds (design D12): a validation, a success, a failure.
const (
	NoticeCheck  = 'v'
	NoticeDone   = 'd'
	NoticeFailed = 'x'
)

// The controls' sizes (designs D1, D3 and D4): the message's rows at most,
// the button's rows and the dropdown's, the button's chevron part in cells
// as an image (the owner's image: the separator at five sixths of the
// width) and in text (the chevron centred between the separator and the
// outline).
const (
	msgMaxRows     = 3
	BtnRows        = 3
	MenuRows       = 3
	moreImageCells = 4
	moreTextCells  = 5
)

// The words of the controls (designs D2 to D4, D6).
const (
	msgHolder   = "Message"
	commitLabel = "Commit" // after the check mark
	pushLabel   = "Push"
	MenuLabel   = "Commit & Push"
	askYes      = "Stage and commit"
	askNo       = "Cancel"
)

// CommitLook is the button as its image draws it (design D5): its width in
// cells, its label, the check mark before it, dimmed with nothing to do,
// its outline in the focus colour.
type CommitLook struct {
	Cells             int
	Label             string
	Check, Dim, Focus bool
	// Live says an operation runs: its label changes every second, so its
	// image is not kept (design D15).
	Live bool
}

// MenuLook is the dropdown as its image draws it: its width in cells, its
// item on the band.
type MenuLook struct {
	Cells int
	Hot   bool
}

// MsgLook is the message box's outline as its images draw it: its width in
// cells, its text rows, its outline in the focus colour.
type MsgLook struct {
	Cells, Rows int
	Focus       bool
}

// CommitShown, MenuShown and MsgShown are where the frame left the
// button's, the dropdown's and the message box's images to be drawn, their
// first cell and look; zero when none (no Sixel, the panel hidden, a window
// over them).
type (
	CommitShown struct {
		Shown bool
		X, Y  int
		Look  CommitLook
	}
	MenuShown struct {
		Shown bool
		X, Y  int
		Look  MenuLook
	}
	MsgShown struct {
		Shown bool
		X, Y  int
		Look  MsgLook
	}
)

// ButtonLabel is the button's label and whether the check mark leads it:
// the running operation's, "Push", else "Commit" after the check (design
// D3), "Commit N files" with a selection (plan 50, design D6).
func ButtonLabel(v SourceView) (string, bool) {
	switch {
	case v.Busy != "":
		return v.Busy, false
	case v.Action == ActionPush:
		return pushLabel, false
	case len(v.Selected) == 1:
		return commitLabel + " 1 file", true
	case len(v.Selected) > 1:
		return fmt.Sprintf("%s %d files", commitLabel, len(v.Selected)), true
	}
	return commitLabel, true
}

// commitControls are the rows between the repository row and the list,
// inner cells wide (design D1): the message box, the button, then the note
// or the question; none while the panel shows no repository.
func commitControls(ui UI, inner int) []line {
	v := ui.Source
	if v.Repo == "" || inner < 12 {
		return nil
	}
	out := msgBox(v, inner, ui.Sixel)
	out = append(out, commitButton(v, inner, ui.Sixel && !ui.PillNoFont)...)
	return append(out, commitNotice(v, inner)...)
}

// msgBox is the message field (design D2): its outline, its rows, one up to
// msgMaxRows, the text on the box's inside from its second cell, "Message"
// while it is empty; with Sixel the outline's cells blank for its images
// (design D5).
func msgBox(v SourceView, inner int, sixel bool) []line {
	focused := v.Place == PlaceMsg
	edge := either(focused, stMsgFocus, stMsgOutline)
	rows, show := shownRows(v.Msg, msgTextCells(inner), 1, msgMaxRows)
	empty := v.Msg.Total <= 1 && (len(v.Msg.Lines) == 0 || v.Msg.Lines[0] == "")
	frame := func(l, r string) line {
		if sixel {
			return line{msgSpan(strings.Repeat(" ", inner), stNone, ItemMsg)}
		}
		return line{msgSpan(l+strings.Repeat("─", inner-2)+r, edge, ItemMsg)}
	}
	side := msgSpan("│", edge, ItemMsg)
	if sixel {
		side = msgSpan(" ", stNone, ItemMsg)
	}
	out := []line{frame("╭", "╮")}
	for i := range show {
		var body line
		switch {
		case i == 0 && empty:
			if focused {
				body = append(body, msgSpan(" ", stCursor, ItemMsg))
			}
			body = append(body, msgSpan(msgHolder, msgGround.on(stMsgHolder), ItemMsgHolder))
		case i < len(rows):
			body = msgRow(drawRow(v.Msg, rows[i], focused))
		}
		if n := inner - 2 - body.width(); n > 0 {
			body = append(body, msgSpan(strings.Repeat(" ", n), msgGround.on(stNone), ItemMsg))
		}
		out = append(out, append(append(line{side}, body...), side))
	}
	return append(out, frame("╰", "╯"))
}

// msgTextCells is the cells a row of the message holds, inner being the
// box's width: the two edges out, and a cell for the cursor at a row's end.
func msgTextCells(inner int) int { return max(inner-3, 1) }

// msgRow is a row of the message as drawRow draws zone B's draft: its spans
// in panel G, the text's ItemMsgText with its line and offset kept for a
// click (MsgAt), an atom's n kept; the text on the box's inside.
func msgRow(l line) line {
	for i := range l {
		s := &l[i]
		switch s.st {
		case nil, stText:
			s.st = msgGround.on(stText)
		case stControl:
			s.st = msgGround.on(stControl)
		}
		item := ItemMsgText
		if s.ref.panel != 'E' {
			item = ItemMsg // the cursor's cell at a row's end
		}
		s.ref.panel, s.ref.kind, s.ref.item = SourcePanel, refDeco, item
	}
	return l
}

// msgSpan is a span of the message box: decoration, panel G, its item.
func msgSpan(text string, st *lipgloss.Style, item int) span {
	return span{text: text, st: st, ref: ref{panel: SourcePanel, kind: refDeco, item: item}}
}

// moreCells is the button's chevron part in cells, as an image or in text.
func moreCells(image bool) int {
	if image {
		return moreImageCells
	}
	return moreTextCells
}

// commitButton is the button, three rows inner cells wide (design D3): its
// left part, the label centred, and its chevron part, the separator then
// the chevron; as an image, its cells blank (design D5). No hover look.
func commitButton(v SourceView, inner int, image bool) []line {
	more := moreCells(image)
	main := inner - more
	mainSpan := func(text string, st *lipgloss.Style) span { return msgSpan(text, st, ItemCommitMain) }
	moreSpan := func(text string, st *lipgloss.Style) span { return msgSpan(text, st, ItemCommitMore) }
	if image {
		row := line{mainSpan(strings.Repeat(" ", main), stNone), moreSpan(strings.Repeat(" ", more), stNone)}
		return []line{row, row, row}
	}
	dim := v.Dim && v.Busy == ""
	outline, label, sep, fill := stBtnOutline, stBtnLabel, stBtnSep, btnGround
	switch {
	case v.Place == PlaceBtn:
		outline = stBtnFocus
	case dim:
		outline = stBtnOutlineDim
	}
	if dim {
		label, sep, fill = stBtnLabelDim, stBtnSepDim, btnDimGround
	}
	text, check := ButtonLabel(v)
	if check {
		text = "✓ " + text
	}
	edge := func(l, r string) line {
		return line{mainSpan(l+strings.Repeat("─", main-1), outline), moreSpan(strings.Repeat("─", more-1)+r, outline)}
	}
	room := main - 1 // the left part's cells inside the outline
	text = cut(text, room)
	pad := room - textWidth(text)
	middle := line{
		mainSpan("│", outline),
		mainSpan(strings.Repeat(" ", pad/2)+text+strings.Repeat(" ", pad-pad/2), fill.on(label)),
		moreSpan("│", fill.on(sep)),
		moreSpan(strings.Repeat(" ", (more-3)/2)+"⌄"+strings.Repeat(" ", more-2-1-(more-3)/2), fill.on(label)),
		moreSpan("│", outline),
	}
	return []line{edge("╭", "╮"), middle, edge("╰", "╯")}
}

// commitNotice is the question or the note under the button (designs D6
// and D12), wrapped to inner cells.
func commitNotice(v SourceView, inner int) []line {
	if v.Busy != "" {
		// While an operation runs: how to cancel a push, or why a stop could
		// not be (design D15); the note waits for its result.
		var out []line
		if v.BusyHint != "" {
			for _, l := range wrapDeco(v.BusyHint, inner, "", either(v.BusyWarn, stAttention, stGrey)) {
				out = append(out, noticeLine(l))
			}
		}
		return out
	}
	if v.Ask > 0 {
		var out []line
		text := AskText(v.Ask, v.AskPush)
		if v.AskSel > 0 {
			text = AskSelText(v.Ask, v.AskSel, v.AskPush)
		}
		for _, l := range wrapDeco(text, inner, "", stText) {
			out = append(out, noticeLine(l))
		}
		yes, no := " "+askYes+" ", " "+askNo+" "
		pick := func(text string, on bool, item int) span {
			return msgSpan(text, either(on, active(stBright), stGrey), item)
		}
		y, n := pick(yes, !v.AskNo, ItemAskYes), pick(no, v.AskNo, ItemAskNo)
		if textWidth(yes)+1+textWidth(no) <= inner {
			return append(out, line{y, msgSpan(" ", stNone, ItemNotice), n})
		}
		return append(out, line{y}, line{n})
	}
	if v.Notice == "" {
		return nil
	}
	st := stDone
	switch v.NoticeKind {
	case NoticeCheck:
		st = stAttention
	case NoticeFailed:
		st = stFailed
	}
	var out []line
	for _, l := range wrapDeco(v.Notice, inner, "", st) {
		out = append(out, noticeLine(l))
	}
	return out
}

// noticeLine puts a wrapped line of the note in panel G, ItemNotice.
func noticeLine(l line) line {
	for i := range l {
		l[i].ref = ref{panel: SourcePanel, kind: refDeco, item: ItemNotice}
	}
	return l
}

// AskSelText is the question of a selection's commit (plan 50, design D7):
// staging the n of its files that are not staged, then the commit of all
// of them, or the commit and the push.
func AskSelText(n, of int, push bool) string {
	then := "commit"
	if push {
		then = "commit and push"
	}
	switch {
	case n == 1 && of == 1:
		return "Stage the selected file and " + then + " it?"
	case n == of:
		return fmt.Sprintf("Stage the %d selected files and %s them?", n, then)
	case n == 1:
		return fmt.Sprintf("Stage the selected file that is not staged, and %s all %d?", then, of)
	}
	return fmt.Sprintf("Stage the %d selected files that are not staged, and %s all %d?", n, then, of)
}

// AskText is the question of design D6 for n changes: staging them, then
// the commit, or the commit and the push.
func AskText(n int, push bool) string {
	then := "commit them"
	if push {
		then = "commit and push them"
	}
	if n == 1 {
		then = strings.TrimSuffix(then, "them") + "it"
		return "Nothing is staged. Stage the change listed under CHANGES and " + then + "?"
	}
	return fmt.Sprintf("Nothing is staged. Stage the %d changes listed under CHANGES and %s?", n, then)
}

// buttonPlace is the button's first cell and width on the screen: the first
// row holding the cells of its left part; ok is false without one.
func (sc *Screen) buttonPlace() (x, y, w int, ok bool) {
	for y := range sc.lines {
		cx, first := 0, -1
		for _, s := range sc.lines[y] {
			sw := textWidth(s.text)
			if s.ref.panel == SourcePanel && (s.ref.item == ItemCommitMain || s.ref.item == ItemCommitMore) {
				if first < 0 {
					first = cx
				}
				w += sw
			}
			cx += sw
		}
		if first >= 0 {
			return first, y, w, true
		}
	}
	return 0, 0, 0, false
}

// cellText is the text of the cell (x, y), "" past the row.
func (sc *Screen) cellText(x, y int) string {
	if y < 0 || y >= len(sc.lines) {
		return ""
	}
	cx := 0
	for _, s := range sc.lines[y] {
		for i := 0; i < len(s.text); {
			j, w := next(s.text, i)
			if cx == x {
				return s.text[i:j]
			}
			cx += w
			if cx > x {
				return ""
			}
			i = j
		}
	}
	return ""
}

// commitMenu draws the dropdown over the panel while it is open (design
// D4): right under the button, its columns, over the panel's own rows below
// it; nothing when those rows are not inside the panel's frame, whose side
// is two cells left of the button.
func commitMenu(sc *Screen, ui UI) {
	x, y, w, ok := sc.buttonPlace()
	if !ok || x < 2 {
		return
	}
	top := y + BtnRows
	for r := range MenuRows {
		if top+r >= len(sc.lines) || sc.cellText(x-2, top+r) != "│" {
			return
		}
	}
	image := ui.Sixel && !ui.PillNoFont
	sc.place(menuLines(ui.Source, w, image), x, top, w)
	sc.Dropdown = true
}

// menuLines are the dropdown's rows, w cells wide: its outline, its item
// three cells in, on the band while hot; as an image, blank cells.
func menuLines(v SourceView, w int, image bool) []line {
	item := func(text string, st *lipgloss.Style) span { return msgSpan(text, st, ItemCommitMenu) }
	if image {
		row := line{item(strings.Repeat(" ", w), stNone)}
		return []line{row, row, row}
	}
	ground := menuGround
	if v.MenuHot {
		ground = menuBandGround
	}
	text := cut(MenuLabel, w-4)
	edge := func(l, r string) line { return line{item(l+strings.Repeat("─", w-2)+r, stMenuOutline)} }
	middle := line{
		item("│", stMenuOutline),
		item("  "+text+strings.Repeat(" ", max(w-4-textWidth(text), 0)), ground.on(stMenuText)),
		item("│", stMenuOutline),
	}
	return []line{edge("╭", "╮"), middle, edge("╰", "╯")}
}

// MenuItemAt says the cell (x, y) is on the dropdown's item: its middle
// row, the rows above and below it the dropdown's too.
func (sc *Screen) MenuItemAt(x, y int) bool {
	at := func(y int) bool {
		p, item := sc.ItemAt(x, y)
		return p == SourcePanel && item == ItemCommitMenu
	}
	return at(y) && at(y-1) && at(y+1)
}

// commitLook is the button's look on the screen, w cells wide.
func commitLook(v SourceView, w int) CommitLook {
	label, check := ButtonLabel(v)
	return CommitLook{Cells: w, Label: label, Check: check, Dim: v.Dim && v.Busy == "", Focus: v.Place == PlaceBtn, Live: v.Busy != ""}
}

// commitShown records where the frame left the controls' images to be
// drawn (design D5): each only where its cells are all there, no window
// over them; the button and the dropdown only with the labels' font.
func (sc *Screen) commitShown(ui UI) {
	v := ui.Source
	if !ui.Sixel || !v.Open {
		return
	}
	if x, y, w, ok := sc.buttonPlace(); ok && !ui.PillNoFont {
		main, whole := w-moreImageCells, true
		for r := range BtnRows {
			whole = whole && sc.cellsOf(x, y+r, main, SourcePanel, ItemCommitMain) &&
				sc.cellsOf(x+main, y+r, moreImageCells, SourcePanel, ItemCommitMore)
		}
		if whole {
			sc.Commit = CommitShown{Shown: true, X: x, Y: y, Look: commitLook(v, w)}
		}
		whole = v.Menu
		for r := range MenuRows {
			whole = whole && sc.cellsOf(x, y+BtnRows+r, w, SourcePanel, ItemCommitMenu)
		}
		if whole {
			sc.Menu = MenuShown{Shown: true, X: x, Y: y + BtnRows, Look: MenuLook{Cells: w, Hot: v.MenuHot}}
		}
	}
	if x, y, w, ok := sc.msgPlace(); ok {
		// Its rows as msgBox drew them: the outline above and below, the
		// edges of each text row.
		_, rows := shownRows(v.Msg, msgTextCells(w), 1, msgMaxRows)
		whole := sc.cellsOf(x, y, w, SourcePanel, ItemMsg) && sc.cellsOf(x, y+1+rows, w, SourcePanel, ItemMsg)
		for r := 1; r <= rows; r++ {
			whole = whole && sc.cellsOf(x, y+r, 1, SourcePanel, ItemMsg) && sc.cellsOf(x+w-1, y+r, 1, SourcePanel, ItemMsg)
		}
		if whole {
			sc.MsgBox = MsgShown{Shown: true, X: x, Y: y, Look: MsgLook{Cells: w, Rows: rows, Focus: v.Place == PlaceMsg}}
		}
	}
}

// msgPlace is the message box's top row: its first cell and width, the
// first row whose cells are all the box's outline.
func (sc *Screen) msgPlace() (x, y, w int, ok bool) {
	for y := range sc.lines {
		cx, first, n := 0, -1, 0
		for _, s := range sc.lines[y] {
			sw := textWidth(s.text)
			if s.ref.panel == SourcePanel && s.ref.item == ItemMsg {
				if first < 0 {
					first = cx
				}
				n += sw
			} else if first >= 0 {
				break
			}
			cx += sw
		}
		if first >= 0 {
			return first, y, n, true
		}
	}
	return 0, 0, 0, false
}

// MsgAt is the place in the message under the cell (x, y): its logical line
// and byte offset; past a row's text, the row's end. ok is false off the
// message's text rows.
func (sc *Screen) MsgAt(x, y int) (ln, off int, ok bool) {
	if y < 0 || y >= len(sc.lines) {
		return 0, 0, false
	}
	cx := 0
	for _, s := range sc.lines[y] {
		sw := textWidth(s.text)
		if s.ref.panel != SourcePanel || s.ref.item != ItemMsgText {
			cx += sw
			continue
		}
		ln, ok = s.ref.line, true
		end := s.ref.off + len(s.text)
		if s.ref.n > 0 {
			end = s.ref.off + s.ref.n // an atom: one unit for n bytes
		}
		if x < cx+sw {
			if s.ref.n > 0 || x < cx {
				return ln, s.ref.off, true
			}
			at := cx
			for i := 0; i < len(s.text); {
				j, w := next(s.text, i)
				if at+w > x {
					return ln, s.ref.off + i, true
				}
				at, i = at+w, j
			}
		}
		off = end
		cx += sw
	}
	return ln, off, ok
}

// commitHints are the footer's key hints while the controls have the focus
// (design D12), the longest first: the dropdown's, the question's, the
// field's, the button's; action names the main one.
func commitHints(v SourceView) []string {
	act := "commit"
	if v.Action == ActionPush {
		act = "push"
	}
	switch {
	case v.Busy != "" && strings.HasPrefix(v.Busy, "Pushing"):
		return []string{"Esc cancel the push · F1 keys", "Esc cancel the push"}
	case v.Menu:
		return []string{"Enter " + MenuLabel + " · Esc close · F1 keys", "Enter " + MenuLabel + " · Esc close"}
	case v.Ask > 0:
		return []string{"←→ choose · Enter confirm · Esc cancel · F1 keys", "←→ · Enter · Esc cancel"}
	case v.Place == PlaceMsg:
		return []string{
			"Ctrl+Enter " + act + " · Enter new line · ↓ button · Esc back to the input · F1 keys",
			"Ctrl+Enter " + act + " · ↓ button · Esc back",
			"Ctrl+Enter " + act,
		}
	case v.Place == PlaceBtn:
		return []string{
			"Enter " + act + " · → " + MenuLabel + " · ↑ message · ↓ changes · Esc back to the input · F1 keys",
			"Enter " + act + " · → " + MenuLabel + " · Esc back",
			"Enter " + act,
		}
	}
	return nil
}
