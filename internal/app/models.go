package app

import (
	tea "charm.land/bubbletea/v2"

	"github.com/Evengelius/agent-command-center/internal/provider"
	"github.com/Evengelius/agent-command-center/internal/render"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// The model selector (plan 34, brief amendment 34; ui-design.md, Input
// editor, The model selector): a click on the editor row's model pill, or
// Alt+P, opens and closes it; while it is open its keys and the mouse work
// it, and every choice goes to the state, which asks the session.

// caps are the session's capabilities; none before a session starts.
func (m *Model) caps() provider.Capabilities {
	if m.sess == nil {
		return provider.Capabilities{}
	}
	return m.sess.Capabilities()
}

// toggleModels opens the selector, or closes it when it is open. It opens
// only on a session that lists its models, once the list came, and on a
// screen with room for it above the editor's rule; otherwise a toast says
// why. The focus stays where it is (design D10).
func (m *Model) toggleModels() []tea.Cmd {
	if m.ui.Models.Open {
		m.closeModels()
		return nil
	}
	c := m.caps()
	switch {
	case !c.Models:
		return m.toast("no model selection with this session")
	case len(m.s.Models.Entries) == 0 && m.s.Models.ListProblem != "":
		return m.toast("no model list: " + m.s.Models.ListProblem)
	case len(m.s.Models.Entries) == 0:
		return m.toast("the session has not listed its models yet")
	}
	sc := m.screen
	if sc == nil {
		sc = render.Draw(m.s, m.ui) // the screen the next frame draws on; m.screen stays the terminal's
	}
	h := sc.SelectorHeight(m.s, render.ModelsView{Open: true, Ultracode: c.Ultracode})
	switch {
	case !sc.PillShown():
		// A full view, or a turn's status too long to sit beside the pill.
		return m.toast("the model selector needs the model pill on screen")
	case h == 0:
		return m.toast("no room for the model selector above the editor")
	}
	m.ui.Plus, m.ui.PlusHover = false, false
	m.closeModes()
	m.ui.Models = render.OpenSelector(m.s, c.Ultracode, h)
	return nil
}

func (m *Model) closeModels() { m.ui.Models = render.ModelsView{} }

// selectorHeight is the rows of the list the last frame leaves the
// selector; the collapsed list's when no frame shows the pill yet.
func (m *Model) selectorHeight() int {
	if m.screen != nil {
		if m.screen.Selector.Shown {
			return m.screen.Selector.Height
		}
		if h := m.screen.SelectorHeight(m.s, m.ui.Models); h > 0 {
			return h
		}
	}
	return 17
}

// scrollSelector scrolls the list by n rows from where the last frame
// showed it; the list then stays there (Free) until the cursor moves.
func (m *Model) scrollSelector(n int) {
	v := &m.ui.Models
	if m.screen != nil && m.screen.Selector.Shown {
		v.Top = m.screen.Selector.Top
	}
	*v = render.ScrollList(m.s, *v, n, m.selectorHeight())
	v.Free = true
}

// modelsKey works the open selector (plan 34, design D9): ↑↓ move the
// cursor, Enter and Space act on its row, ← and → move the effort slider
// while the cursor is on Effort (plan 37, design D4), PgUp and PgDn
// scroll, Esc and Alt+P close it. A send key the configuration names acts
// as Enter does. Any other key closes it and goes on, as plan 30's menu;
// no send key reaches the editor, so the popup never sends.
func (m *Model) modelsKey(k tea.KeyPressMsg) ([]tea.Cmd, bool) {
	if m.opt.Keys.IsSend(k.String()) {
		return m.modelsAct(), true
	}
	v := &m.ui.Models
	items := render.SelectorItems(m.s, *v)
	h := m.selectorHeight()
	switch k.String() {
	case "esc", "alt+p":
		m.closeModels()
	case "up", "down":
		step := 1
		if k.String() == "up" {
			step = -1
		}
		v.Cursor = max(0, min(v.Cursor+step, len(items)-1))
		v.Free = false
		*v = render.CursorInView(m.s, *v, h)
	case "left", "right":
		if v.Cursor >= 0 && v.Cursor < len(items) && items[v.Cursor].Kind == 'e' {
			step := 1
			if k.String() == "left" {
				step = -1
			}
			return m.stepEffort(step), true
		}
		m.closeModels()
		return nil, false
	case "home", "end":
		v.Cursor = 0
		if k.String() == "end" {
			v.Cursor = len(items) - 1
		}
		v.Free = false
		*v = render.CursorInView(m.s, *v, h)
	case "pgup", "pgdown":
		n := h
		if k.String() == "pgup" {
			n = -h
		}
		m.scrollSelector(n)
	case "enter", "space":
		return m.modelsAct(), true
	default:
		m.closeModels()
		return nil, false
	}
	return nil, true
}

// modelsAct acts on the row under the cursor; on Effort nothing, the
// slider moving with ← and → (plan 37, design D4).
func (m *Model) modelsAct() []tea.Cmd {
	v := &m.ui.Models
	items := render.SelectorItems(m.s, *v)
	if v.Cursor < 0 || v.Cursor >= len(items) {
		return nil
	}
	switch it := items[v.Cursor]; it.Kind {
	case 'm':
		return m.do(state.ChooseModel{ID: m.s.Models.Entries[it.Model].ID, NextTurn: m.caps().ModelsNextTurn})
	case 'o':
		m.toggleMore()
	case 'u':
		return m.do(state.ChooseUltracode{On: !m.s.Models.InForce.Ultracode})
	}
	return nil
}

// toggleMore expands or collapses More models; expanded, the list scrolls
// to put its row at the top.
func (m *Model) toggleMore() {
	v := &m.ui.Models
	v.More, v.Free = !v.More, false
	m.cursorTo('o', 0)
	if v.More {
		v.Top = render.ItemTop(m.s, *v, v.Cursor)
	}
	*v = render.CursorInView(m.s, *v, m.selectorHeight())
}

// stepEffort asks for the level before (step -1) or after (+1) the one the
// slider shows, a level asked for and waiting counting as shown; nothing
// past the ends, nor going down from none known. A held effort asks for
// nothing: the state says why (plan 37, design D4).
func (m *Model) stepEffort(step int) []tea.Cmd {
	e := render.EffortModel(m.s)
	if e == nil || len(e.Efforts) == 0 {
		return nil
	}
	if !render.EffortChangeable(m.s) {
		return m.do(state.ChooseEffort{Level: m.s.Models.InForce.Effort, NextTurn: m.caps().ModelsNextTurn})
	}
	cur := m.s.Models.InForce.Effort
	if a := m.s.Models.Asked; a != nil && a.Effort != "" {
		cur = a.Effort
	}
	at := -1
	for i, l := range e.Efforts {
		if l == cur {
			at = i
		}
	}
	return m.chooseLevel(at + step)
}

// chooseLevel asks for the slider's level i of the model's levels; none
// outside them.
func (m *Model) chooseLevel(i int) []tea.Cmd {
	e := render.EffortModel(m.s)
	if e == nil || i < 0 || i >= len(e.Efforts) {
		return nil
	}
	return m.do(state.ChooseEffort{Level: e.Efforts[i], NextTurn: m.caps().ModelsNextTurn})
}

// modelsClick acts on the selector's item a click landed on: a model, More
// models, a slider's cell (its level), Ultracode; on the Effort row's label
// or description it moves the cursor there; on the frame or a blank row it
// does nothing.
func (m *Model) modelsClick(item int) []tea.Cmd {
	switch {
	case item >= render.ItemLevel:
		m.cursorTo('e', 0)
		return m.chooseLevel(item - render.ItemLevel)
	case item >= render.ItemModel:
		if m.cursorTo('m', item-render.ItemModel) {
			return m.modelsAct()
		}
	case item == render.ItemMore:
		m.toggleMore()
	case item == render.ItemEffort:
		m.cursorTo('e', 0)
	case item == render.ItemUltracode:
		if m.cursorTo('u', 0) {
			return m.modelsAct()
		}
	}
	return nil
}

// modelsPointer records the selector's item under the pointer at (x, y),
// -1 outside the selector, and says whether the pointer came onto another
// item than the last report's. A terminal reports a pointer moved by
// software, even one at rest, once the selector asks for every motion,
// and those reports wander a cell; so only a change of item is a hover,
// and the selector opened by a key, which does not know where the pointer
// is, takes its first report as where it rests (plan 36, design D1).
func (m *Model) modelsPointer(x, y int) (item int, moved bool) {
	v := &m.ui.Models
	item = -1
	if p, it := m.screen.ItemAt(x, y); p == render.SelectorPanel {
		item = it
	}
	moved = v.PointerKnown && item != v.Pointer
	v.Pointer, v.PointerKnown = item, true
	return item, moved
}

// modelsHover moves the cursor to the row of the item under the pointer
// (design D9), a slider's cell being Effort's.
func (m *Model) modelsHover(item int) {
	switch {
	case item >= render.ItemLevel:
		m.cursorTo('e', 0)
	case item >= render.ItemModel:
		m.cursorTo('m', item-render.ItemModel)
	case item == render.ItemMore:
		m.cursorTo('o', 0)
	case item == render.ItemEffort:
		m.cursorTo('e', 0)
	case item == render.ItemUltracode:
		m.cursorTo('u', 0)
	}
}

// cursorTo puts the cursor on the selector's row of that kind (a model's,
// by its index); false when there is none.
func (m *Model) cursorTo(kind byte, model int) bool {
	for i, it := range render.SelectorItems(m.s, m.ui.Models) {
		if it.Kind == kind && (kind != 'm' || it.Model == model) {
			m.ui.Models.Cursor = i
			return true
		}
	}
	return false
}

// modelsCmd hands a change to the session through the write queue, in
// order with the owner's messages; the session writes it at once or keeps
// it for the next one (plan 38).
func (m *Model) modelsCmd(c state.ModelChange) tea.Cmd {
	sess := m.sess
	req := provider.ModelRequest{Model: c.Choice, Effort: c.Effort, Ultracode: c.Ultracode}
	return m.write(func() tea.Msg {
		if sess == nil {
			return actionMsg{state.ModelsFailed{Message: "no session"}}
		}
		if err := sess.SetModelSettings(req); err != nil {
			return actionMsg{state.ModelsFailed{Message: err.Error()}}
		}
		return nil
	}, actionMsg{state.ModelsFailed{Message: "too many writes waiting"}})
}
