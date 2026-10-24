package render

import (
	"strings"

	"github.com/Evengelius/agent-command-center/internal/event"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// The model selector (plan 34, brief amendment 34 and the owner's image
// references/11-model-selector.webp; reference-design.md, The model
// selector): a click on the editor row's model pill, or Alt+P, opens a box
// over zone B, its bottom edge on the editor's rule, with the session's
// models, their effort and, for Claude Code, Ultracode. Columns are the
// reference's, counted from the box's left edge, W its width; a row's line
// is built inside the frame, its column c at the box's column c+1.

// ModelsView is the selector as the owner works it; the application owns it.
type ModelsView struct {
	Open   bool
	Cursor int  // the row under the cursor, an index into SelectorItems
	More   bool // More models expanded
	Top    int  // the first row of the list shown
	// Free says the owner scrolled the list (the wheel, PgUp, PgDn): it
	// stays at Top; otherwise each frame keeps the cursor's rows in view,
	// whatever the screen's height.
	Free      bool
	Ultracode bool // the CLI has Ultracode (Capabilities.Ultracode): its row is drawn
	// Pointer is the item under the mouse pointer as the last mouse report
	// gave it (0 on the frame or a blank row, -1 outside the selector);
	// PointerKnown says a report came since the selector opened. The
	// cursor follows the pointer onto another item only (plan 36, design
	// D1).
	Pointer      int
	PointerKnown bool
}

// SelectorShown is what the last frame showed of the selector's list: its
// first row and its height, for the next scroll (the frame may have moved
// it to keep the cursor in view). A selector open that the frame did not
// show is closed by the application (design D9).
type SelectorShown struct {
	Shown       bool
	Top, Height int
}

// The selector's panel, the pill's, and the items the selector's cells
// carry, for the mouse.
const (
	SelectorPanel = 'S'
	PillPanel     = 'M'

	ItemMore      = 1
	ItemEffort    = 2
	ItemUltracode = 3
	ItemModel     = 1000 // plus the model's index in the session's list
	ItemLevel     = 2000 // plus the level's index in the model's levels: a slider's cell
)

// MainModels is how many models the list shows before More models: the
// first five in the CLI's order (design D2).
const MainModels = 5

// The box's geometry (reference-design.md, The model selector).
const (
	selectorMinWidth = 63 // the image's width in cells
	selectorSlack    = 12 // the cells a model's row takes beyond its text
)

// EffortDescription is the Effort row's description, the owner's words
// (amendment 38).
const EffortDescription = "Controls reasoning depth"

// SelectorItem is one row the cursor can be on.
type SelectorItem struct {
	Kind  byte // 'm' a model, 'o' More models, 'e' Effort, 'u' Ultracode
	Model int  // the model's index in the session's list, for 'm'
}

// SelectorItems are the selector's rows in order: the first five models,
// More models and, expanded, the others; then Effort and, where the CLI
// has it, Ultracode.
func SelectorItems(s *state.State, v ModelsView) []SelectorItem {
	n := len(s.Models.Entries)
	var out []SelectorItem
	for i := 0; i < n && i < MainModels; i++ {
		out = append(out, SelectorItem{Kind: 'm', Model: i})
	}
	if n > MainModels {
		out = append(out, SelectorItem{Kind: 'o'})
		if v.More {
			for i := MainModels; i < n; i++ {
				out = append(out, SelectorItem{Kind: 'm', Model: i})
			}
		}
	}
	out = append(out, SelectorItem{Kind: 'e'})
	if v.Ultracode {
		out = append(out, SelectorItem{Kind: 'u'})
	}
	return out
}

// OpenSelector is the selector as it opens on a screen showing h rows of
// its list (SelectorHeight): the cursor on the model in force, More models
// expanded when it is one of them (design D2), the list scrolled to it.
func OpenSelector(s *state.State, ultracode bool, h int) ModelsView {
	v := ModelsView{Open: true, Ultracode: ultracode}
	at := -1
	if cur := s.Models.Current(); cur != nil {
		for i := range s.Models.Entries {
			if s.Models.Entries[i].ID == cur.ID {
				at = i
			}
		}
	}
	v.More = at >= MainModels
	for i, it := range SelectorItems(s, v) {
		if it.Kind == 'm' && it.Model == at {
			v.Cursor = i
		}
	}
	return CursorInView(s, v, h)
}

// CursorInView scrolls the list, h rows shown, so the cursor's rows are in
// view; the list kept in bounds.
func CursorInView(s *state.State, v ModelsView, h int) ModelsView {
	rows, first, last := selectorList(s, v, 0)
	if c := v.Cursor; c >= 0 && c < len(first) && first[c] >= 0 {
		if first[c] < v.Top {
			v.Top = first[c]
		}
		if last[c] >= v.Top+h {
			v.Top = last[c] - h + 1
		}
	}
	v.Top = max(0, min(v.Top, len(rows)-h))
	return v
}

// ItemTop is the first row of the list the item of SelectorItems takes, 0
// for one outside the list.
func ItemTop(s *state.State, v ModelsView, item int) int {
	_, first, _ := selectorList(s, v, 0)
	if item >= 0 && item < len(first) && first[item] >= 0 {
		return first[item]
	}
	return 0
}

// ScrollList moves the list by n rows, h shown, the cursor where it is.
func ScrollList(s *state.State, v ModelsView, n, h int) ModelsView {
	rows, _, _ := selectorList(s, v, 0)
	v.Top = max(0, min(v.Top+n, len(rows)-h))
	return v
}

// SelectorHeight is how many rows of the list the selector shows on this
// screen: as many as with More models collapsed, never more than zone B
// leaves between its top edge and the editor's rule (design D8), so the
// title bar and the project bar stay in view; 0 when the pill or the rule
// is not on screen, or fewer than two rows are left: the selector is then
// not drawn.
func (sc *Screen) SelectorHeight(s *state.State, v ModelsView) int {
	_, py, ok := sc.find(PillPanel)
	if !ok {
		return 0
	}
	ry := sc.editorRule(py)
	if ry < 0 {
		return 0
	}
	c := v
	c.More = false
	rows, _, _ := selectorList(s, c, 0)
	if h := min(len(rows), ry-sc.selectorTop()+1-selectorFixed(v)); h >= 2 {
		return h
	}
	return 0
}

// PillShown says the screen shows the model pill, which the selector opens
// from: not under a full view, nor while a status too long to sit beside
// it takes the editor row alone (plan 29, design D1).
func (sc *Screen) PillShown() bool {
	_, _, ok := sc.find(PillPanel)
	return ok
}

// selectorTop is the highest row the selector may take: zone B's first row
// under its title, never the title bar or the project bar.
func (sc *Screen) selectorTop() int { return max(sc.Conv.Y, 2) }

// selectorFixed is the box's rows outside its list: the top edge, the
// header and a blank row; a blank row and Effort; a blank row and
// Ultracode where drawn; the bottom edge.
func selectorFixed(v ModelsView) int {
	if v.Ultracode {
		return 8
	}
	return 6
}

// selRow is one row of the list, w cells wide inside the frame, and the
// item it belongs to, -1 for none.
type selRow struct {
	l    line
	item int
}

// selectorList is the list's rows, w cells wide inside the frame (0 for
// their structure only), and for each item of SelectorItems its first and
// last row, -1 for one outside the list: each model two rows with a blank
// row before all but the first; a line, More models and a line; the other
// models, expanded, before that second line.
func selectorList(s *state.State, v ModelsView, w int) (rows []selRow, first, last []int) {
	items := SelectorItems(s, v)
	first, last = make([]int, len(items)), make([]int, len(items))
	for i := range items {
		first[i], last[i] = -1, -1
	}
	add := func(l line, item int) {
		if item >= 0 {
			if first[item] < 0 {
				first[item] = len(rows)
			}
			last[item] = len(rows)
		}
		rows = append(rows, selRow{l: l, item: item})
	}
	shown := 0
	for i, it := range items {
		switch it.Kind {
		case 'm':
			if shown > 0 {
				add(blankRow(w), -1)
			}
			name, desc := modelRows(s, v, it.Model, i, w)
			add(name, i)
			add(desc, i)
			shown++
		case 'o':
			add(separator(w), -1)
			add(moreRow(v, i, w), i)
		}
	}
	add(separator(w), -1) // the list's last line, under More models or the last model
	return rows, first, last
}

func blankRow(w int) line { return line{deco(strings.Repeat(" ", max(w, 0)), stNone)} }

// modelRows are a model's two rows: its name from column 5, the cursor's
// pink "▶" at column 3 and the "✓" of the model in force at column W-5
// (a grey "…" while it is asked for); then its description from column 5;
// both on the band while the cursor is on it.
func modelRows(s *state.State, v ModelsView, i, item, w int) (line, line) {
	e := s.Models.Entries[i]
	cursor := v.Cursor == item
	text := max(w-9, 1) // columns 5 to W-7
	mark, check, checkSt := " ", " ", stBright
	if cursor {
		mark = "▶"
	}
	if cur := s.Models.Current(); cur != nil && cur.ID == e.ID {
		check = "✓"
	} else if a := s.Models.Asked; a != nil && a.Choice == e.ID {
		check, checkSt = "…", stModelAside // asked for, not yet in force (designs D3, D4)
	}
	name := line{deco("  ", stNone), deco(mark, stModelMark), deco(" ", stNone), deco(fitText(e.Name, text), stBright)}
	if aside := " (default)"; e.Default && textWidth(e.Name)+textWidth(aside) <= text {
		name = append(name, deco(aside, stModelDesc)) // Codex's default model (design D2)
	}
	name = append(padRow(name, w-4), deco(check, checkSt), deco("   ", stNone))
	descSt := stModelDesc
	if cursor {
		descSt = stMuted
	}
	desc := padRow(line{deco("    ", stNone), deco(fitText(e.Description, text), descSt)}, w)
	if cursor {
		name, desc = banded(name, w), banded(desc, w)
	}
	return items(name, ItemModel+i), items(desc, ItemModel+i)
}

// moreRow is More models: "▶" collapsed or "▼" expanded at column 3, its
// name from column 5, "›" at column W-5.
func moreRow(v ModelsView, item, w int) line {
	tri := "▶"
	if v.More {
		tri = "▼"
	}
	l := line{deco("  ", stNone), deco(tri, stBright), deco(" ", stNone), deco("More models", stBright)}
	l = append(padRow(l, w-4), deco("›", stBright), deco("   ", stNone))
	if v.Cursor == item {
		l = banded(l, w)
	}
	return items(l, ItemMore)
}

// separator is a line from column 2 to column W-3.
func separator(w int) line {
	if w < 2 {
		return blankRow(w)
	}
	return line{deco(" ", stNone), deco(strings.Repeat("─", w-2), stModelRule), deco(" ", stNone)}
}

// fitText is s cut to w cells, "…" in its last cell when it is cut.
func fitText(s string, w int) string {
	if textWidth(s) <= w {
		return s
	}
	if w <= 1 {
		return "…"
	}
	return cut(s, w-1) + "…"
}

// items gives each span of l the selector's panel and the item.
func items(l line, item int) line {
	for j := range l {
		l[j].ref = ref{panel: SelectorPanel, kind: refDeco, item: item}
	}
	return l
}

// padRow is l padded with spaces, or cut, to w cells.
func padRow(l line, w int) line {
	if d := w - l.width(); d > 0 {
		return append(l, deco(strings.Repeat(" ", d), stNone))
	}
	return l.fit(w)
}

// banded is a row on the cursor's band from column 2 to column W-3: its
// first and last cell inside the frame stay off it.
func banded(l line, w int) line {
	out := make(line, 0, len(l)+2)
	x := 0
	for _, s := range l {
		for _, p := range splitCols(s, x, 1, w-1) {
			if p.on {
				p.s.st = bandGround.on(p.s.st)
			}
			out = append(out, p.s)
		}
		x += textWidth(s.text)
	}
	return out
}

// colPiece is part of a span, on the band or not.
type colPiece struct {
	s  span
	on bool
}

// splitCols cuts s, which starts at column x, where columns from to to
// (excluded) begin and end.
func splitCols(s span, x, from, to int) []colPiece {
	w := textWidth(s.text)
	if x >= from && x+w <= to {
		return []colPiece{{s, true}}
	}
	if x+w <= from || x >= to {
		return []colPiece{{s, false}}
	}
	var out []colPiece
	var cur strings.Builder
	cx, on := x, x >= from && x < to
	flush := func() {
		if cur.Len() > 0 {
			p := s
			p.text = cur.String()
			out = append(out, colPiece{p, on})
			cur.Reset()
		}
	}
	for _, r := range s.text {
		if in := cx >= from && cx < to; in != on {
			flush()
			on = in
		}
		cur.WriteRune(r)
		cx += runeWidth(r)
	}
	flush()
	return out
}

// EffortModel is the model whose levels the effort row offers: Codex's
// model asked for, while it waits, else the model in force.
func EffortModel(s *state.State) *event.ModelEntry {
	if a := s.Models.Asked; a != nil && a.NextTurn && a.Choice != "" {
		if e := s.Models.Entry(a.Choice); e != nil {
			return e
		}
	}
	return s.Models.Current()
}

// effortLevel is the level the effort row shows: the one in force, as the
// pill shows it, a level asked for being only ringed on the slider (plan
// 37, design D5) and, Codex's, named on the editor row until the next
// turn takes it (design D4); "" when none is known.
func effortLevel(s *state.State) string {
	if in := s.Models.InForce; in.EffortKnown {
		return in.Effort
	}
	return ""
}

// EffortChangeable says the effort row can be changed: the model lists
// levels and no variable holds the effort (design D5).
func EffortChangeable(s *state.State) bool {
	e := EffortModel(s)
	return e != nil && len(e.Efforts) > 0 && s.Models.InForce.EffortLock == ""
}

// settingsLabelWidth is the settings rows' label column (plan 37, design
// D3): the longest label they can show, "Effort (<level>)" for any level of
// the session's models, "Effort (N/A)", and "Ultracode" where it is drawn,
// so that no level shown moves the descriptions.
func settingsLabelWidth(s *state.State, ultracode bool) int {
	w := textWidth("Effort (N/A)")
	for _, e := range s.Models.Entries {
		for _, l := range e.Efforts {
			w = max(w, textWidth("Effort ("+state.EffortLabel(l)+")"))
		}
	}
	if ultracode {
		w = max(w, textWidth("Ultracode"))
	}
	return w
}

// settingsDescCol is where the settings rows' descriptions start inside the
// frame: the label column from column 3, then two cells (design D3).
func settingsDescCol(s *state.State, ultracode bool) int {
	return 2 + settingsLabelWidth(s, ultracode) + 2
}

// settingsWidth is the box's width the settings rows take: their
// descriptions, two cells, the slider's pill, which ends at column W-4,
// and the three cells after it (design D3).
func settingsWidth(s *state.State, ultracode bool) int {
	desc := textWidth(EffortDescription)
	if ultracode {
		desc = max(desc, textWidth(UltracodeDescription))
	}
	return 1 + settingsDescCol(s, ultracode) + desc + 2 + SliderPillCells + 3
}

// sliderLook is what the Effort row's slider shows (design D1, D5): the
// levels of EffortModel, the thumb on the level in force (none while it is
// not known or not one of them), a ring on a level asked for and not in
// force, greyed while a variable holds the effort; false for a model that
// lists no level, which has no slider.
func sliderLook(s *state.State, band bool) (SliderLook, bool) {
	e := EffortModel(s)
	if e == nil || len(e.Efforts) == 0 {
		return SliderLook{}, false
	}
	look := SliderLook{Levels: len(e.Efforts), Pos: -1, Asked: -1, Locked: s.Models.InForce.EffortLock != "", Band: band}
	cur, asked := effortLevel(s), ""
	if a := s.Models.Asked; a != nil {
		asked = a.Effort
	}
	for i, l := range e.Efforts {
		if cur != "" && l == cur {
			look.Pos = float64(i)
		}
		if asked != "" && asked != cur && l == asked {
			look.Asked = i
		}
	}
	return look, true
}

// effortRow is the Effort row on the settings columns (plan 37, design D3):
// "Effort (<level in force>)" at column 3 ("Effort (N/A)" when the model
// lists no level or none is known), its description at the description
// column, its slider's pill ending at column W-4: with Sixel SliderCells
// blank cells (W-19 to W-3) for its image, whose last is the band's last,
// else the text slider (W-19 to W-4); each pill cell carries its nearest
// level for a click. A model with no level has no slider. It gives the
// slider's look, and whether there is one.
func effortRow(s *state.State, v ModelsView, item, w, descCol int, sixel bool) (line, SliderLook, bool) {
	cursor := v.Cursor == item
	look, ok := sliderLook(s, cursor)
	label := "Effort (N/A)"
	if lv := effortLevel(s); ok && lv != "" {
		label = "Effort (" + state.EffortLabel(lv) + ")"
	}
	l := padRow(line{deco("  ", stNone), deco(label, stBright)}, descCol)
	l = append(l, deco(EffortDescription, stModelDesc))
	if !ok {
		l = padRow(l, w)
		if cursor {
			l = banded(l, w)
		}
		return items(l, ItemEffort), look, false
	}
	// After the text slider two cells, the band reaching the first; after
	// the image one, the image painting the band's last cell.
	cells, after := SliderPillCells, 2
	if sixel {
		cells, after = SliderCells, 1
	}
	l = padRow(l, w-cells-after)
	tail := line{deco(strings.Repeat(" ", after), stNone)}
	if cursor {
		l = banded(l, w)
		if !sixel {
			tail = line{deco(" ", bandGround.on(stNone)), deco(" ", stNone)}
		}
	}
	l = append(items(l, ItemEffort), sliderCells(look, sixel)...)
	return append(l, items(tail, ItemEffort)...), look, true
}

// UltracodeDescription is the Ultracode row's description, the owner's
// words (amendment 34).
const UltracodeDescription = "Enables dynamic workflows"

// UltracodeUsable says the CLI reports Ultracode available.
func UltracodeUsable(s *state.State) bool {
	in := s.Models.InForce
	return in.UltracodeKnown && in.UltracodeAvailable
}

// ultracodeRow is "Ultracode" at column 3, its description at the settings
// rows' description column (plan 37, design D3), its toggle's pill ending
// at column W-4: with Sixel, 5 blank cells (W-7 to W-3) for its image,
// whose last is the band's last (plan 35, design D2), else the text toggle
// (W-8 to W-4, design D3); dimmed, the toggle off, while the CLI does not
// report it available (plan 34, design D6). On the band the toggle keeps
// its own colours, the band around it.
func ultracodeRow(s *state.State, v ModelsView, item, w, descCol int, sixel bool) line {
	on, usable := toggleState(s)
	label, desc := stBright, stModelDesc
	if !usable {
		label, desc = stGrey, stGrey
	}
	cursor := v.Cursor == item
	// After the text toggle two cells, the band reaching the first; after
	// the image one, the image painting the band's last cell.
	cells, after := toggleTextCells, 2
	if sixel {
		cells, after = ToggleCells, 1
	}
	l := padRow(line{deco("  ", stNone), deco("Ultracode", label)}, descCol)
	l = padRow(append(l, deco(UltracodeDescription, desc)), w-cells-after)
	tail := line{deco(strings.Repeat(" ", after), stNone)}
	if cursor {
		l = banded(l, w)
		if !sixel {
			tail = line{deco(" ", bandGround.on(stNone)), deco(" ", stNone)}
		}
	}
	l = append(append(l, toggleCells(on, usable, cursor, sixel)...), tail...)
	return items(l, ItemUltracode)
}

// selectorWidth is the box's width: 63 cells, the image's, or the longest
// name or description and the cells around it, or what the settings rows
// take (plan 37); never wider than the screen.
func selectorWidth(s *state.State, v ModelsView, width int) int {
	w := max(selectorMinWidth, settingsWidth(s, v.Ultracode))
	for _, e := range s.Models.Entries {
		name := textWidth(e.Name)
		if e.Default {
			name += textWidth(" (default)")
		}
		w = max(w, name+selectorSlack, textWidth(e.Description)+selectorSlack)
	}
	return min(w, width)
}

// SelectorHeading is the header's left part: the session's CLI's models.
func SelectorHeading(s *state.State) string {
	if s.CLI() == "codex" {
		return "CODEX MODELS"
	}
	return "CLAUDE MODELS"
}

// modelSelector draws the selector over the dashboard when it is open: its
// left edge one cell left of the pill, its bottom edge on the editor's rule
// with "▾" above the pill's first cell, at column 1 (design D8), further
// right when zone B's right edge pushes the box left, never over the
// Explorer's panel or the sidebar (plan 43, design D9). It draws nothing
// when the pill or the rule is not on screen, or the rows above the rule
// leave fewer than two for the list.
func modelSelector(sc *Screen, s *state.State, ui UI) {
	v := ui.Models
	px, py, ok := sc.find(PillPanel)
	h := sc.SelectorHeight(s, v)
	if !ok || h == 0 || len(s.Models.Entries) == 0 {
		return
	}
	ry := sc.editorRule(py)
	w := selectorWidth(s, v, ui.Width)
	inner := w - 2
	x := max(min(px-1, sc.bRight(ui.Width)-w), 0)
	y := ry + 1 - selectorFixed(v) - h
	tail := max(1, min(px-x, inner)) // the "▾", above the pill's first cell

	if !v.Free {
		v = CursorInView(s, v, h)
	}
	rows, _, _ := selectorList(s, v, inner)
	top := max(0, min(v.Top, len(rows)-h))
	sc.Selector = SelectorShown{Shown: true, Top: top, Height: h}
	at := func(kind byte) int {
		for i, it := range SelectorItems(s, v) {
			if it.Kind == kind {
				return i
			}
		}
		return -1
	}
	frame := func(l line, right string) line {
		out := line{deco("│", stModelFrame)}
		out = append(out, padRow(l, inner)...)
		return append(out, deco(right, stModelFrame))
	}
	box := []line{
		{deco("╭"+strings.Repeat("─", inner)+"╮", stModelFrame)},
		frame(spread(line{deco(" ", stNone), deco(SelectorHeading(s), stModelHead)},
			line{deco("Current session", stModelAside), deco(" ", stNone)}, inner), "│"),
		frame(blankRow(inner), "│"),
	}
	from, to := scrollThumb(len(rows), h, top)
	for i := 0; i < h; i++ {
		edge := "│"
		if i >= from && i < to {
			edge = "┃" // the list's scroll bar, on the right edge (design D8)
		}
		box = append(box, frame(rows[top+i].l, edge))
	}
	// The settings rows on their shared columns (plan 37, design D3).
	desc := settingsDescCol(s, v.Ultracode)
	box = append(box, frame(blankRow(inner), "│"))
	erow, look, slider := effortRow(s, v, at('e'), inner, desc, ui.Sixel)
	if slider && ui.Sixel {
		// The slider's image lies on its row's blank cells, drawn by the
		// application once the frame is out (plan 37, design D1).
		sc.Slider = SliderShown{Shown: true, X: x + 1 + inner - SliderCells - 1, Y: y + len(box), Look: look}
	}
	box = append(box, frame(erow, "│"))
	if v.Ultracode {
		box = append(box, frame(blankRow(inner), "│"))
		if ui.Sixel {
			// The toggle's image lies on its row's blank cells, drawn by the
			// application once the frame is out (plan 35, design D2).
			on, usable := toggleState(s)
			sc.Toggle = ToggleShown{Shown: true, X: x + 1 + inner - ToggleCells - 1, Y: y + len(box), On: on, Usable: usable,
				Band: v.Cursor == at('u')}
		}
		box = append(box, frame(ultracodeRow(s, v, at('u'), inner, desc, ui.Sixel), "│"))
	}
	box = append(box, line{deco("╰"+strings.Repeat("─", tail-1)+"▾"+strings.Repeat("─", max(inner-tail, 0))+"╯", stModelFrame)})
	for i := range box {
		for j := range box[i] {
			if box[i][j].ref.panel == 0 {
				box[i][j].ref = ref{panel: SelectorPanel, kind: refDeco}
			}
		}
	}
	sc.place(box, x, y, w)
}

// scrollThumb is the rows of the list's height h the scroll bar's thumb
// takes, from and to (excluded); none when the whole list shows.
func scrollThumb(n, h, top int) (int, int) {
	if n <= h || h <= 0 {
		return 0, 0
	}
	size := max(1, h*h/n)
	from := min(top*h/n, h-size)
	if top+h >= n {
		from = h - size
	}
	return from, from + size
}

// modelPill is the editor row's model pill (plan 34, design D7): " <model>
// <Effort>", then " · Ultracode" while it is in force; its cells are panel
// M, where a click opens the selector.
func modelPill(s *state.State) line {
	in := s.Models.InForce
	text := " " + pillModel(s)
	if in.EffortKnown && in.Effort != "" {
		text += " " + state.EffortLabel(in.Effort)
	}
	l := line{deco(text, stText)}
	if in.UltracodeKnown && in.Ultracode {
		l = append(l, deco(" · ", stMdBullet), deco("Ultracode ", stYou))
	} else {
		l = append(l, deco(" ", stText))
	}
	for j := range l {
		l[j].ref = ref{panel: PillPanel, kind: refDeco}
	}
	return l
}

// pillModel is the model in force by the name the CLI's list gives the
// entry that runs it (not Claude Code's Default, which names no model),
// else its id read as zone B's speaker reads it; "model N/A" when none is
// known.
func pillModel(s *state.State) string {
	m := s.Models.InForce.Model
	if m == "" {
		m = s.Session.Model
	}
	if m == "" {
		return "model N/A"
	}
	for _, e := range s.Models.Entries {
		if e.Model == m && e.ID != "default" {
			return e.Name
		}
	}
	return modelName(m)
}
