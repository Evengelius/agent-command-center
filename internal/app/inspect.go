package app

import (
	tea "charm.land/bubbletea/v2"

	"github.com/Evengelius/agent-command-center/internal/render"
)

// Zone A's tree, zone C's list and the agent inspection (ui-design.md, Zone
// A; Zone C; Agent inspection).

// openInspect shows agent key full screen, on its Messages tab, following.
func (m *Model) openInspect(key string) {
	m.ui.Inspect, m.ui.Tab = key, 0
	m.ui.Selected = key
	m.ui.Sel = render.Selection{}
	m.inspectFollow()
}

// inspectKey handles a key while an agent is inspected; done is false for the
// keys that keep their dashboard meaning (copy, F12, Ctrl+O, Shift+PgUp).
func (m *Model) inspectKey(k tea.KeyPressMsg) (cmds []tea.Cmd, done bool) {
	switch s := k.String(); s {
	case "esc":
		if !m.ui.Sel.Empty() {
			m.ui.Sel = render.Selection{} // Esc clears a selection first
			return nil, true
		}
		m.ui.Inspect = "" // the focus is the one the inspection was opened from
		m.ui.Sel = render.Selection{}
	case "left", "right":
		d := 1
		if s == "left" {
			d = len(render.Tabs) - 1
		}
		m.setTab((m.ui.Tab + d) % len(render.Tabs))
	case "1", "2", "3", "4":
		m.setTab(int(s[0] - '1'))
	case "[", "]":
		if key := step(render.AgentOrder(m.s), m.ui.Inspect, s == "]"); key != "" {
			m.ui.Inspect, m.ui.Selected = key, key
			m.ui.Sel = render.Selection{}
			m.inspectFollow()
		}
	case "up":
		m.inspectScroll(-1)
	case "down":
		m.inspectScroll(1)
	case "pgup", "pgdown":
		m.inspectPage(s == "pgdown")
	case "home":
		m.ui.InspectHold, m.ui.InspectTop = true, 0
	case "end":
		m.inspectFollow()
	case "f2":
		m.ui.Inspect = ""
		return m.focusCard("", true), true
	case "ctrl+c", "ctrl+shift+c", "f12", "ctrl+o", "shift+pgup", "shift+pgdown", "ctrl+r", "F":
		return nil, false
	}
	return nil, true
}

func (m *Model) setTab(t int) {
	m.ui.Tab = t
	m.ui.Sel = render.Selection{}
	m.inspectFollow()
}

// inspectScroll moves the inspection d rows; scrolling up holds the view, and
// reaching the bottom follows again, as in zone B.
func (m *Model) inspectScroll(d int) {
	if m.screen == nil {
		return
	}
	c := m.screen.Inspect
	bottom := max(c.Rows-c.Room, 0)
	top := c.Top
	if m.ui.InspectHold {
		top = min(max(m.ui.InspectTop, 0), bottom)
	}
	top = max(top+d, 0)
	if top >= bottom {
		m.inspectFollow()
		return
	}
	m.ui.InspectHold, m.ui.InspectTop = true, top
	m.screen.Inspect.Top = top // a second step before the next frame starts here
}

// inspectPage scrolls the inspection or the details view one page.
func (m *Model) inspectPage(down bool) {
	p := 1
	if m.screen != nil {
		p = max(m.screen.Inspect.Room-1, 1)
	}
	if !down {
		p = -p
	}
	m.inspectScroll(p)
}

func (m *Model) inspectFollow() {
	m.ui.InspectHold, m.ui.InspectTop = false, 0
	if m.screen != nil {
		m.screen.Inspect.Top = max(m.screen.Inspect.Rows-m.screen.Inspect.Room, 0)
	}
}

// treeKey moves zone A's cursor, folds and unfolds its nodes, and inspects
// an agent's.
func (m *Model) treeKey(k tea.KeyPressMsg) []tea.Cmd {
	nodes := render.Tree(m.s, m.ui)
	if len(nodes) == 0 {
		return nil
	}
	i := len(nodes) - 1 // no cursor yet, or its node is gone: the last one
	for j, n := range nodes {
		if n.ID == m.ui.ACursor {
			i = j
		}
	}
	known := m.ui.ACursor != "" && nodes[i].ID == m.ui.ACursor
	switch k.String() {
	case "up":
		if known {
			i = max(i-1, 0)
		}
	case "down":
		if known {
			i = min(i+1, len(nodes)-1)
		}
	case "left", "right":
		if n := nodes[i]; n.Foldable {
			if m.ui.Fold == nil {
				m.ui.Fold = map[string]bool{}
			}
			m.ui.Fold[n.ID] = k.String() == "left"
		}
	case "enter":
		m.ui.ACursor = nodes[i].ID
		if a := nodes[i].Agent; a != "" {
			m.openInspect(a)
			return nil
		}
		return m.toast("the orchestrator's own: its messages are in zone B")
	default:
		return nil
	}
	m.ui.ACursor = nodes[i].ID
	return nil
}

// listKey moves zone C's selection in the order the zone shows, and inspects
// the selected agent.
func (m *Model) listKey(k tea.KeyPressMsg) []tea.Cmd {
	order := render.AgentOrder(m.s)
	switch k.String() {
	case "up", "[":
		m.ui.Selected = step(order, m.ui.Selected, false)
	case "down", "]":
		m.ui.Selected = step(order, m.ui.Selected, true)
	case "enter":
		key := m.ui.Selected
		if m.s.Agents[key] == nil && len(order) > 0 {
			key = order[0]
		}
		if key != "" {
			m.openInspect(key)
		}
	}
	return nil
}

// openDetails shows zone D full screen at its top: the newest error first
// (ui-design.md, Zone D: Errors).
func (m *Model) openDetails() {
	m.ui.Details = true
	m.ui.Sel = render.Selection{}
	m.ui.InspectHold, m.ui.InspectTop = true, 0
	if m.screen != nil {
		m.screen.Inspect.Top = 0
	}
}

// detailsKey handles a key while zone D is full screen; done is false for
// the keys that keep their dashboard meaning, as in an inspection.
func (m *Model) detailsKey(k tea.KeyPressMsg) (cmds []tea.Cmd, done bool) {
	switch s := k.String(); s {
	case "esc":
		if !m.ui.Sel.Empty() {
			m.ui.Sel = render.Selection{} // Esc clears a selection first
			return nil, true
		}
		m.ui.Details = false // back to D, still focused
		m.ui.InspectHold, m.ui.InspectTop = false, 0
	case "up":
		m.inspectScroll(-1)
	case "down":
		m.inspectScroll(1)
	case "pgup", "pgdown":
		m.inspectPage(s == "pgdown")
	case "home":
		m.ui.InspectHold, m.ui.InspectTop = true, 0
	case "end":
		m.inspectFollow()
	case "f2":
		m.ui.Details = false
		return m.focusCard("", true), true
	case "ctrl+c", "ctrl+shift+c", "f12", "ctrl+o", "shift+pgup", "shift+pgdown", "ctrl+r", "F":
		return nil, false
	}
	return nil, true
}
