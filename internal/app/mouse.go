package app

import (
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/Evengelius/agent-command-center/internal/input"
	"github.com/Evengelius/agent-command-center/internal/render"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// Mouse timing and steps (ui-design.md, Selection and copy; Scrolling).
const (
	multiClick = 400 * time.Millisecond // a second press within it, on the same cell, is a double click
	wheelRows  = 3
)

// drag is a mouse selection in progress: the panel it started in and the
// unit (character, word or line) it started on.
type drag struct {
	on     bool
	panel  byte
	a0, a1 render.Pos
	x, y   int  // the cell pressed
	moved  bool // the pointer left that cell: a plain click selects nothing
}

// clicks counts presses on one cell for double and triple clicks.
type clicks struct {
	at   time.Time
	x, y int
	n    int
}

// clickFold is a fold a click toggled in zone B, and B's view as it was
// before (plan 42).
type clickFold struct {
	key       string
	hold      bool
	top, seen int
}

// mouse handles a mouse message on the dashboard.
func (m *Model) mouse(msg tea.MouseMsg) []tea.Cmd {
	if m.s.Screen != state.ScreenDashboard || m.screen == nil || m.wt != nil {
		return nil // the Worktrees view takes no mouse: the terminal selects
	}
	if render.WindowOpen(m.ui) && !m.ui.Keys {
		return m.fileMouse(msg) // the window floats over the dashboard (plan 17)
	}
	mo := msg.Mouse()
	switch msg.(type) {
	case tea.MouseClickMsg:
		// The "+" menu (plan 30): a left click on the band opens or closes
		// it; while it is open, a click on its item closes it and opens the
		// Open dialog (plan 31), a click anywhere else closes it and goes on.
		// A click on a chip's ✕ removes its image.
		if mo.Button == tea.MouseLeft && !m.ui.DraftWin.Open {
			switch p, item := m.screen.ItemAt(mo.X, mo.Y); {
			case p == render.SidePanel:
				// The sidebar (plan 43, design D5): the Explorer's tile shows or
				// hides its panel as F3 does, Source Control's its own (plan 44,
				// design D1), each closing a popup as a click outside it does;
				// the sidebar's frame changes nothing at all, no state, focus or
				// selection.
				if item != render.SideExplorer && item != render.SideSource {
					return nil
				}
				m.ui.Plus, m.ui.PlusHover = false, false
				m.ui.SideHover, m.ui.SideTipOff = item, true // the click answers the tooltip, until the pointer leaves the tile
				m.closeModels()
				m.closeModes()
				if item == render.SideSource {
					return m.toggleSource()
				}
				return m.toggleExplorer()
			case p == render.PillPanel:
				// The model pill opens and closes the model selector (plan 34),
				// which then knows the pointer lies outside it (plan 36).
				cmds := m.toggleModels()
				if m.ui.Models.Open {
					m.modelsPointer(mo.X, mo.Y)
				}
				return cmds
			case m.ui.Models.Open && p == render.SelectorPanel:
				m.modelsPointer(mo.X, mo.Y)
				return m.modelsClick(item)
			case p == render.ModePillPanel:
				// The mode pill opens and closes the modes selector (plan 39,
				// design D8), the same way.
				cmds := m.toggleModes()
				if m.ui.Modes.Open {
					m.modesPointer(mo.X, mo.Y)
				}
				return cmds
			case m.ui.Modes.Open && p == render.ModesPanel:
				m.modesPointer(mo.X, mo.Y)
				return m.modesClick(item)
			case p == '+':
				m.closeModels()
				m.closeModes()
				m.ui.Plus, m.ui.PlusHover = !m.ui.Plus, false
				m.ui.Focus = "B"
				return nil
			case m.ui.Models.Open || m.ui.Modes.Open:
				// A click outside closes the selector and goes on, as one
				// outside the "+" menu does (plan 34, design D9; plan 39,
				// design D8).
				m.closeModels()
				m.closeModes()
				if p == 'X' {
					m.ui.Focus = "B"
					return m.removeAttachment(item)
				}
			case m.ui.Plus:
				m.ui.Plus, m.ui.PlusHover = false, false
				if p == 'U' {
					return m.pickImages()
				}
			case p == 'X':
				m.ui.Focus = "B"
				return m.removeAttachment(item)
			}
		}
		// Over the draft window, only the window takes a click (amendment 13).
		if mo.Button == tea.MouseLeft && (!m.ui.DraftWin.Open || m.screen.PanelAt(mo.X, mo.Y) == 'E') {
			switch p, item := m.screen.ItemAt(mo.X, mo.Y); p {
			case render.ExplorerPanel:
				return m.explorerClick(item) // the panel's own, no selection (plan 43, design D7)
			case render.SourcePanel:
				return m.sourceClick(item) // the same for Source Control's (plan 44, design D5)
			}
			m.press(mo)
		}
	case tea.MouseMotionMsg:
		if !m.drag.on {
			// The sidebar's hover (plan 43, design D4): the tile under the
			// pointer, 0 off the tiles, a button held or not (a press on a
			// tile starts no drag); a report wandering within a tile leaves
			// it as it is, and it never changes what is selected.
			m.sideHover(m.sideTileAt(mo.X, mo.Y))
			m.ui.Source.Hover = m.sourceHoverAt(mo.X, mo.Y) // Source Control's row (plan 44, design D5)
		}
		if m.ui.Plus && !m.drag.on {
			m.ui.PlusHover = m.screen.PanelAt(mo.X, mo.Y) == 'U' // the hover (plan 30, design D3)
		}
		if m.ui.Models.Open && !m.drag.on {
			// The cursor follows the pointer onto another row (plan 34, design
			// D9), not a report of where it rests (plan 36, design D1).
			if item, moved := m.modelsPointer(mo.X, mo.Y); moved && item > 0 {
				m.modelsHover(item)
			}
		}
		if m.ui.Modes.Open && !m.drag.on {
			// The modes selector's the same way, its highlight leaving with
			// the pointer (plan 39, design D8).
			if item, moved := m.modesPointer(mo.X, mo.Y); moved {
				m.modesHover(item)
			}
		}
		if m.drag.on {
			m.drag.moved = m.drag.moved || mo.X != m.drag.x || mo.Y != m.drag.y
			m.extend(mo)
		}
	case tea.MouseReleaseMsg:
		m.sideHover(m.sideTileAt(mo.X, mo.Y)) // where the button came up
		m.ui.Source.Hover = m.sourceHoverAt(mo.X, mo.Y)
		if m.drag.on {
			m.drag.moved = m.drag.moved || mo.X != m.drag.x || mo.Y != m.drag.y
			m.extend(mo)
			m.drag.on = false
			return m.copySelection()
		}
	case tea.MouseWheelMsg:
		if m.ui.Models.Open && m.screen.PanelAt(mo.X, mo.Y) == render.SelectorPanel {
			// The wheel scrolls the selector's list (plan 34, design D8).
			n := wheelRows
			if mo.Button == tea.MouseWheelUp {
				n = -wheelRows
			}
			if mo.Button == tea.MouseWheelUp || mo.Button == tea.MouseWheelDown {
				m.scrollSelector(n)
			}
			return nil
		}
		if m.ui.Modes.Open && m.screen.PanelAt(mo.X, mo.Y) == render.ModesPanel {
			return nil // the modes box has nothing to scroll, and zone B under it stays (plan 39)
		}
		if m.ui.DraftWin.Open {
			if m.screen.PanelAt(mo.X, mo.Y) == 'E' && (mo.Button == tea.MouseWheelUp || mo.Button == tea.MouseWheelDown) {
				mv := input.Down
				if mo.Button == tea.MouseWheelUp {
					mv = input.Up
				}
				for range wheelRows {
					m.edit.Move(mv, false) // the wheel moves through the draft
				}
			}
			return nil
		}
		if p := m.screen.PanelAt(mo.X, mo.Y); p == 'I' {
			switch mo.Button {
			case tea.MouseWheelUp:
				m.inspectScroll(-wheelRows)
			case tea.MouseWheelDown:
				m.inspectScroll(wheelRows)
			}
		} else if p == 'B' || p == 'E' {
			switch mo.Button {
			case tea.MouseWheelUp:
				m.scroll(-wheelRows)
			case tea.MouseWheelDown:
				m.scroll(wheelRows)
			}
		} else if p == 'A' || p == 'C' || p == 'P' {
			// The pane under the pointer, whatever has the focus (plan 5,
			// item 5).
			switch mo.Button {
			case tea.MouseWheelUp:
				m.paneScroll(p, -wheelRows)
			case tea.MouseWheelDown:
				m.paneScroll(p, wheelRows)
			}
		} else if p == render.ExplorerPanel {
			m.explorerWheel(mo.Button) // over the panel only (plan 43, design D7)
		} else if p == render.SourcePanel {
			m.sourceWheel(mo.Button) // the same for Source Control's (plan 44, design D5)
		} else if p == render.DiffPanel {
			m.diffWheel(mo.Button) // the diff under the pointer (plan 44, design D9)
		}
	}
	return nil
}

// sideHoverable says the sidebar's tiles follow the pointer: the frame drew
// the sidebar (the dashboard from 100 columns, no full view or zoom over
// it), no floating window or draft window is open, the mouse is captured
// (plan 43, design D4).
func (m *Model) sideHoverable() bool {
	return m.screen != nil && m.screen.Side.W > 0 && m.s.Screen == state.ScreenDashboard && m.wt == nil &&
		!m.ui.MouseOff && !render.WindowOpen(m.ui) && !m.ui.DraftWin.Open
}

// sideHover sets the sidebar's hovered tile, its tooltip back once the
// pointer is on another tile or off them (plan 44, design D1).
func (m *Model) sideHover(tile int) {
	if tile != m.ui.SideHover {
		m.ui.SideHover, m.ui.SideTipOff = tile, false
	}
}

// sideTileAt is the sidebar's tile under the pointer at (x, y) on the frame
// drawn last, 0 off the tiles or while the tiles follow no pointer.
func (m *Model) sideTileAt(x, y int) int {
	if !m.sideHoverable() {
		return 0
	}
	return m.screen.SideTile(x, y)
}

// motionFilter is the program's filter (plan 43, design D4; tea.WithFilter,
// run.go): it drops, before Update and View, a motion report with no button
// that would change nothing: no popup is open, whose hover follows every
// motion (the "+" menu, the model selector, the modes selector), no drag
// runs, the sidebar's tile under the pointer, 0 off the tiles, is the one
// hovered already, and so is Source Control's row (plan 44, design D5). The
// all-motion reporting the hovers need thus costs a frame only when one
// changes. It runs on the program's event loop, as Update and View do, and
// reads the frame drawn last; the model counts what it drops.
func motionFilter(tm tea.Model, msg tea.Msg) tea.Msg {
	mo, ok := msg.(tea.MouseMotionMsg)
	if !ok || mo.Button != tea.MouseNone {
		return msg
	}
	m, ok := tm.(*Model)
	if !ok || m.ui.Plus || m.ui.Models.Open || m.ui.Modes.Open || m.drag.on || m.sideTileAt(mo.X, mo.Y) != m.ui.SideHover ||
		m.sourceHoverAt(mo.X, mo.Y) != m.ui.Source.Hover {
		return msg
	}
	m.motionDropped++
	return nil
}

// press focuses the panel under the pointer and starts a selection there:
// a character, a word on a double click, a line on a triple click; Shift
// extends the current one.
func (m *Model) press(mo tea.Mouse) {
	p := m.screen.PanelAt(mo.X, mo.Y)
	if p == 0 {
		return
	}
	switch p {
	case 'P':
		m.ui.Focus = "C"
	case 'E':
		m.ui.Focus = "B"
	case 'I', 'F':
		// The inspection or the file viewer covers the dashboard: the focus
		// stays the one Esc returns to.
	default:
		m.ui.Focus = string(p)
	}
	if mo.Mod&tea.ModShift != 0 && m.drag.panel == p {
		m.drag.on, m.drag.moved = true, true
		m.extend(mo)
		return
	}
	now := m.opt.Now()
	if m.click.n > 0 && now.Sub(m.click.at) <= multiClick && mo.X == m.click.x && mo.Y == m.click.y {
		m.click.n = m.click.n%3 + 1
	} else {
		m.click.n = 1
	}
	m.click.at, m.click.x, m.click.y = now, mo.X, mo.Y

	// Zone B's clicks on its foldable items and double clicks on its entries
	// (plan 19; plan 42).
	if c := m.screen.Conv; p == 'B' && m.convClick(c, mo.Y-c.Y) {
		m.ui.Sel = render.Selection{}
		return
	}
	// A click on a row of zone A with a node puts the cursor on it; on
	// TELEMETRY it opens the details, on a section's heading it folds or
	// unfolds the section (plan 24, design D6).
	if id, ok := m.screen.ARows[mo.Y]; p == 'A' && ok && m.click.n == 1 {
		m.ui.ACursor = id
		switch id {
		case render.NodeTelemetry:
			m.openDetails()
		case render.NodeUsage, render.NodeQuota:
			if m.ui.Fold == nil {
				m.ui.Fold = map[string]bool{}
			}
			m.ui.Fold[id] = !m.ui.Fold[id]
		default:
			break // a tree node: the cursor moved; the click may start a selection
		}
		if render.IsTelemetryNode(id) {
			m.ui.Sel = render.Selection{}
			return
		}
	}

	start, end, ok := m.screen.HitIn(p, mo.X, mo.Y, false)
	if !ok {
		m.ui.Sel = render.Selection{}
		return
	}
	switch m.click.n {
	case 1:
		end = start
	case 2:
		a, b := input.WordBounds(m.lineText(start), start.Off)
		start.Off, end.Off = a, b
	case 3:
		start.Off, end.Off = 0, len(m.lineText(start))
	}
	m.drag = drag{on: true, panel: p, a0: start, a1: end, x: mo.X, y: mo.Y, moved: m.click.n > 1}
	m.setSelection(start, end, true)
}

// convClick handles a press on zone B's conversation row r, counted from
// its first row shown (plan 42, brief amendment 43): a click on a foldable
// item's row, a run of calls (plan 19) or a thinking phase's summary row,
// unfolds it or folds it again; a double click on a row of an entry Enter
// shows whole opens it full screen, as Enter does, the fold its first click
// toggled back as it was; on a fold row a second click adds nothing.
// Neither puts the entry cursor anywhere: the mouse shows none of the
// keyboard's hints (design D3). false leaves the press to the selection.
func (m *Model) convClick(c render.ConvView, r int) bool {
	key, fold := c.Folds[r]
	id, n := "", 0
	if m.click.n == 2 {
		id, n = m.opens(c.Entries[r])
	}
	switch {
	case m.click.n == 1 && fold:
		m.clickFold = clickFold{key: key, hold: m.ui.ConvHold, top: m.ui.ConvTop, seen: m.ui.ConvSeen}
		m.unfold(key, !m.ui.Unfolded[key])
	case id != "" || n > 0:
		if f := m.clickFold; fold && f.key == key {
			m.unfold(key, !m.ui.Unfolded[key])
			m.ui.ConvHold, m.ui.ConvTop, m.ui.ConvSeen = f.hold, f.top, f.seen
		}
		m.openView(id, n)
		m.click.n = 0 // a third press starts afresh, on the full view
	case m.click.n >= 2 && fold:
		// The fold row's first click did it.
	default:
		return false
	}
	m.ui.Entry = 0
	return true
}

// opens is what a double click on conversation entry n (1-based; 0 none)
// shows whole, as Enter does: an approval by its id, as from its card; an
// entry the entry cursor stops on, by n; neither when Enter shows none.
func (m *Model) opens(n int) (id string, entry int) {
	if n < 1 || n > len(m.s.Conversation) {
		return "", 0
	}
	if e := m.s.Conversation[n-1]; e.Role == "approval" {
		if m.s.Approvals[e.MessageID] != nil {
			return e.MessageID, 0
		}
		return "", 0
	}
	if render.Targets(m.s, m.ui)[n-1] {
		return "", n
	}
	return "", 0
}

// extend moves the selection's free end to the pointer, clamped to the
// drag's panel; zone B scrolls when the pointer leaves it upward or
// downward.
func (m *Model) extend(mo tea.Mouse) {
	if !m.drag.moved {
		return // still on the pressed cell: the caret or the word stays
	}
	p := m.drag.panel
	if p == 'B' {
		if c := m.screen.Conv; mo.Y < c.Y {
			m.scroll(-1)
		} else if mo.Y >= c.Y+c.Room {
			m.scroll(1)
		}
	}
	_, end, ok := m.screen.HitIn(p, mo.X, mo.Y, true)
	if !ok {
		return
	}
	if !end.Before(m.drag.a0) {
		m.setSelection(m.drag.a0, maxPos(end, m.drag.a1), true)
		return
	}
	if start, _, ok := m.screen.HitIn(p, mo.X, mo.Y, false); ok && start.Before(m.drag.a0) {
		m.setSelection(start, m.drag.a1, false)
	}
}

func maxPos(a, b render.Pos) render.Pos {
	if a.Before(b) {
		return b
	}
	return a
}

// setSelection selects from..to in the drag's panel: the editor's own
// selection in the editor (its cursor at the free end), the mouse selection
// elsewhere.
func (m *Model) setSelection(from, to render.Pos, forward bool) {
	if m.drag.panel != 'E' {
		m.ui.Sel = render.Selection{From: from, To: to}
		m.edit.ClearSelection()
		return
	}
	m.ui.Sel = render.Selection{}
	a, b := m.edit.Offset(from.Line, from.Off), m.edit.Offset(to.Line, to.Off)
	if forward {
		m.edit.Select(a, b)
	} else {
		m.edit.Select(b, a)
	}
}

// lineText is the logical line p is on.
func (m *Model) lineText(p render.Pos) string {
	if p.Panel == 'E' {
		if ls := m.edit.LineRange(p.Line, 1); len(ls) == 1 {
			return ls[0]
		}
		return ""
	}
	return m.screen.LineText(m.s, p)
}

// copySelection copies the mouse selection, or the editor's after a drag in
// it (ui-design.md, Selection and copy: copy on release).
func (m *Model) copySelection() []tea.Cmd {
	if !m.ui.Sel.Empty() {
		return m.copy(m.screen.Text(m.s, m.ui.Sel))
	}
	if m.drag.panel == 'E' && m.edit.HasSelection() {
		return m.copy(m.edit.SelectedText())
	}
	return nil
}

// scroll moves the conversation d rows; scrolling up holds the view, and
// reaching the bottom follows again (ui-design.md, Zone B, Follow mode).
func (m *Model) scroll(d int) {
	if m.screen == nil || m.screen.Conv.Room == 0 {
		// No conversation drawn, the Explorer's panel in zone B's place (plan
		// 43, design D1): its view stays as it was.
		return
	}
	c := m.screen.Conv
	bottom := max(c.Rows-c.Room, 0)
	top := c.Top
	if m.ui.ConvHold {
		top = min(max(m.ui.ConvTop, 0), bottom)
	}
	top = max(top+d, 0)
	if top >= bottom {
		m.follow()
		return
	}
	if !m.ui.ConvHold {
		m.ui.ConvHold, m.ui.ConvSeen = true, len(m.s.Conversation)
	}
	m.ui.ConvTop = top
	m.screen.Conv.Top = top // a second step before the next frame starts here
}

// follow puts the conversation back at its end.
func (m *Model) follow() {
	m.ui.ConvHold, m.ui.ConvTop = false, 0
	if m.screen != nil {
		m.screen.Conv.Top = max(m.screen.Conv.Rows-m.screen.Conv.Room, 0)
	}
}

// page is a screenful of the conversation, less one row of context.
func (m *Model) page() int {
	if m.screen == nil {
		return 1
	}
	return max(m.screen.Conv.Room-1, 1)
}

// pageStep is a page down, or up.
func (m *Model) pageStep(down bool) int {
	if down {
		return m.page()
	}
	return -m.page()
}
