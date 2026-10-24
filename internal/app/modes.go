package app

import (
	"image"

	tea "charm.land/bubbletea/v2"

	"github.com/Evengelius/agent-command-center/internal/input"
	"github.com/Evengelius/agent-command-center/internal/render"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// The modes selector (plan 39, brief amendment 40): the mode pill's click
// and F4 open it above the pill, the cursor on the mode in force; ↑ ↓ Home
// End move the cursor, Enter or Space chooses, Esc, F4 or a click outside
// closes it; the pointer's hover moves the cursor onto another option only
// (plan 36's rule), and back to the mode in force when the pointer leaves
// the options. Codex's Full access goes through a confirmation in the
// box's place (design D4). The focus, the draft and its images stay as
// they are (design D8).

// toggleModes opens the selector, or closes it when it is open: only on a
// session that takes modes and on a screen with room for it above the
// editor's rule; otherwise a toast says why (design D9).
func (m *Model) toggleModes() []tea.Cmd {
	if m.ui.Modes.Open {
		m.closeModes()
		return nil
	}
	if !m.caps().Modes || len(m.s.ModeOptions()) == 0 {
		return m.toast("no modes with this session")
	}
	sc := m.screen
	if sc == nil {
		sc = render.Draw(m.s, m.ui) // the screen the next frame draws on; m.screen stays the terminal's
	}
	v := render.OpenModes(m.s)
	switch {
	case !sc.ModePillOn():
		// A full view, or a turn's status too long to sit beside the pill.
		return m.toast("the modes need the mode pill on screen")
	case !sc.ModesFit(m.s, v, m.ui.Width):
		return m.toast("no room for the modes above the editor")
	}
	m.ui.Plus, m.ui.PlusHover = false, false
	m.closeModels()
	m.ui.Modes = v
	return nil
}

func (m *Model) closeModes() { m.ui.Modes = render.ModesView{} }

// modesKey works the open selector (design D8): ↑↓ Home End move the
// cursor over the options that take a choice, Enter and Space (and a send
// key the configuration names) choose, Esc closes the confirmation back to
// the modes or the selector, F4 closes it; any other key closes it and
// goes on, so no send key reaches the editor while it is open.
func (m *Model) modesKey(k tea.KeyPressMsg) ([]tea.Cmd, bool) {
	if m.opt.Keys.IsSend(k.String()) {
		return m.modesAct(), true
	}
	v := &m.ui.Modes
	n := render.ModesCount(m.s, *v)
	switch k.String() {
	case "esc":
		if v.Confirm {
			m.leaveConfirm()
			return nil, true
		}
		m.closeModes()
	case "f4":
		m.closeModes()
	case "up":
		m.modesStep(v.Cursor-1, -1)
	case "down":
		m.modesStep(max(v.Cursor+1, 0), 1)
	case "home":
		m.modesStep(0, 1)
	case "end":
		m.modesStep(n-1, -1)
	case "enter", "space":
		return m.modesAct(), true
	default:
		m.closeModes()
		return nil, false
	}
	return nil, true
}

// modesStep puts the cursor on the first option from i on, going by step,
// that takes a choice; nowhere past the ends, the cursor then staying.
func (m *Model) modesStep(i, step int) {
	v := &m.ui.Modes
	if v.Cursor < 0 && step < 0 {
		i = render.ModesCount(m.s, *v) - 1 // ↑ with no cursor: from the last
	}
	for ; i >= 0 && i < render.ModesCount(m.s, *v); i += step {
		if render.ModesOff(m.s, *v, i) == "" {
			v.Cursor = i
			return
		}
	}
}

// modesAct chooses the option under the cursor: a mode is asked for and
// the selector closes; Full access first opens the confirmation, whose
// Cancel goes back to the modes and whose Turn on Full access asks for it
// (design D4). An option the CLI's policy leaves out asks for nothing and
// the selector stays: the state says why.
func (m *Model) modesAct() []tea.Cmd {
	v := &m.ui.Modes
	if v.Cursor < 0 || v.Cursor >= render.ModesCount(m.s, *v) {
		return nil
	}
	c := m.caps()
	a := state.ChooseMode{Mode: render.ModesMode(m.s, *v, v.Cursor), Capable: c.Modes, NextTurn: c.ModelsNextTurn}
	switch {
	case v.Confirm && a.Mode == "":
		m.leaveConfirm()
		return nil
	case v.Confirm:
		m.closeModes()
		a.Confirmed = true
		return m.do(a)
	case render.ModesOff(m.s, *v, v.Cursor) != "":
		return m.do(a)
	case a.Mode == state.ModeFull && m.s.Mode() != state.ModeFull:
		// The rows change under the pointer: its next report is where it
		// rests, not a hover (plan 36, design D1).
		m.ui.Modes = render.OpenConfirm(m.s)
		return nil
	}
	m.closeModes()
	return m.do(a)
}

// leaveConfirm goes back from the confirmation to the modes, the cursor on
// Full access; the pointer's next report is where it rests.
func (m *Model) leaveConfirm() {
	v := render.OpenModes(m.s)
	for i := range render.ModesCount(m.s, v) {
		if render.ModesMode(m.s, v, i) == state.ModeFull {
			v.Cursor = i
		}
	}
	m.ui.Modes = v
}

// modesClick chooses the option a click landed on; on the frame, the
// heading or a half row it does nothing.
func (m *Model) modesClick(item int) []tea.Cmd {
	if item < render.ItemModeOpt {
		return nil
	}
	m.ui.Modes.Cursor = item - render.ItemModeOpt
	return m.modesAct()
}

// modesPointer records the item under the pointer at (x, y), -1 outside
// the box, and says whether the pointer came onto another item than the
// last report's (plan 36, design D1).
func (m *Model) modesPointer(x, y int) (item int, moved bool) {
	v := &m.ui.Modes
	item = -1
	if p, it := m.screen.ItemAt(x, y); p == render.ModesPanel {
		item = it
	}
	moved = v.PointerKnown && item != v.Pointer
	v.Pointer, v.PointerKnown = item, true
	return item, moved
}

// modesHover follows the pointer that came onto item (design D8): onto an
// option that takes a choice, the cursor goes there; onto the heading, the
// frame or out of the box, back to the mode in force, so the pointer
// leaving an option takes its highlight away (criterion 39.7); a half row
// between two options, or a greyed option, leaves it where it is.
func (m *Model) modesHover(item int) {
	v := &m.ui.Modes
	switch {
	case item >= render.ItemModeOpt:
		if render.ModesOff(m.s, *v, item-render.ItemModeOpt) == "" {
			v.Cursor = item - render.ItemModeOpt
		}
	case item == render.ItemModesHalf:
	default:
		v.Cursor = render.ModesHome(m.s, *v)
	}
}

// pillLabel is the mode pill's title as GDI draws it, rasterised once a
// title (plan 39, design D7).
func (m *Model) pillLabel(title string) (*image.Alpha, error) {
	if l, ok := m.pillLabels[title]; ok {
		return l, nil
	}
	l, err := input.TextMask(title, pillLabelEm, pillLabelTop, render.SixelCellH)
	if err != nil {
		return nil, err
	}
	if m.pillLabels == nil {
		m.pillLabels = map[string]*image.Alpha{}
	}
	m.pillLabels[title] = l
	return l, nil
}

// The pill's title: 13 of the cell's 20 pixels a character's height (8 a
// character's width, render.ModePillCells'), one pixel down from the row's
// top, so its letters sit around the pill's middle (rows 6 to 14) and its
// descenders above the outline (row 17 at most); smaller than the
// terminal's own text (about 17 pixels), as the owner's third image draws
// its label inside the pill.
const (
	pillLabelEm  = 13
	pillLabelTop = 1
)
