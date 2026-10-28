package app

import (
	"slices"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"

	"github.com/Evengelius/agent-command-center/internal/input"
	"github.com/Evengelius/agent-command-center/internal/render"
)

// The Git Graph's scope popup (plan 62, design D3): All, Auto, or the
// references ticked; Enter applies the scope chosen, Esc and a click
// outside keep the one there was. It only filters the history shown: it
// never switches a branch, fetches or writes.

// openScope opens the scope popup over the panel, the scope in force
// ticked, the references as read last (read again when none were).
func (m *Model) openScope() []tea.Cmd {
	m.closePopups() // the one popup open (plan 49, design D7)
	g := m.graph
	v := render.ScopeView{Open: true, All: g.scope.All, Checked: map[string]bool{}, Folded: [render.ScopeGroups]bool{false, true, true}, Cursor: "u"}
	for _, r := range g.scope.Refs {
		v.Checked[r] = true
	}
	switch {
	case g.scope.All:
		v.Cursor = "a"
	case len(g.scope.Refs) > 0:
		v.Cursor = "r:" + g.scope.Refs[0]
		_, kind := refShort(g.scope.Refs[0])
		v.Folded[scopeGroupOf(kind)] = false // the first reference ticked in view
	}
	m.ui.Scope = v
	m.fillScope()
	if !g.refsOK {
		m.ui.Scope.Reading = true
		if c := m.readGraphRefs(0); c != nil {
			return []tea.Cmd{c}
		}
	}
	return nil
}

// scopeGroupOf is the popup's group of a reference's kind.
func scopeGroupOf(kind byte) int {
	switch kind {
	case 'r':
		return render.ScopeRemote
	case 't':
		return render.ScopeTags
	}
	return render.ScopeLocal
}

// fillScope lists the references read in the popup's groups, each with its
// commit's short hash.
func (m *Model) fillScope() {
	v := &m.ui.Scope
	if !m.graph.refsOK {
		return
	}
	v.Reading = false
	var groups [render.ScopeGroups][]render.ScopeRef
	for _, r := range m.graph.refs.Refs {
		short, kind := refShort(r.Name)
		if kind == 0 {
			continue
		}
		g := scopeGroupOf(kind)
		groups[g] = append(groups[g], render.ScopeRef{Name: r.Name, Short: short, Hash: r.Commit[:min(7, len(r.Commit))]})
	}
	v.Groups = groups
}

// closeScope closes the popup, the scope as it was.
func (m *Model) closeScope() { m.ui.Scope = render.ScopeView{} }

// scopeKey handles a key while the popup is open (design D3); a key it does
// not take closes it and goes on, as the branches popup's.
func (m *Model) scopeKey(k tea.KeyPressMsg) (cmds []tea.Cmd, done bool) {
	v := &m.ui.Scope
	rows := render.ScopeRows(*v)
	cur := render.ScopeCursor(rows, v.Cursor)
	move := m.scopeMove
	room := 1 // a page of the list as the last frame showed it, as the branches popup's
	if m.screen != nil {
		room = max(m.screen.Scope.Height, 1)
	}
	switch s := k.String(); s {
	case "esc":
		if v.Query != "" {
			v.Query = ""
			return nil, true
		}
		m.closeScope()
		return nil, true
	case "enter":
		return m.applyScope(), true
	case "up":
		move(-1)
	case "down":
		move(1)
	case "pgup":
		move(-room)
	case "pgdown":
		move(room)
	case "home":
		move(-len(rows))
	case "end":
		move(len(rows))
	case "space":
		if cur >= 0 {
			m.scopeToggle(rows[cur])
		}
	case "right", "left":
		if cur >= 0 && rows[cur].Kind == 'g' && v.Query == "" {
			v.Folded[rows[cur].Group] = s == "left"
		}
	case "backspace":
		if _, n := utf8.DecodeLastRuneInString(v.Query); n > 0 {
			v.Query = v.Query[:len(v.Query)-n]
		}
		m.scopeFirst()
	case "ctrl+q", "f1", "f2", "f3", "f4", "f6", "shift+f6", "tab", "shift+tab", "alt+g", "alt+b", "alt+p", "f12":
		m.closeScope() // a key the popup does not take closes it and goes on
		return nil, false
	default:
		if k.Text != "" && k.Text != " " {
			v.Query += firstLine(k.Text)
			m.scopeFirst()
			return nil, true
		}
		if k.Text == " " {
			if cur >= 0 {
				m.scopeToggle(rows[cur])
			}
			return nil, true
		}
		return nil, true // the popup keeps the keys it has no use for
	}
	return nil, true
}

// scopeFirst puts the cursor on the first reference the search keeps, else
// on Auto.
func (m *Model) scopeFirst() {
	v := &m.ui.Scope
	for _, r := range render.ScopeRows(*v) {
		if r.Kind == 'r' {
			v.Cursor = r.Key()
			return
		}
	}
	v.Cursor = "u"
}

// scopeToggle is Space or a click on row it (design D3): All or Auto made
// the mode, every box cleared; a reference ticked or cleared, the last one
// cleared giving Auto; a heading folded or opened.
func (m *Model) scopeToggle(it render.ScopeItem) {
	v := &m.ui.Scope
	switch it.Kind {
	case 'a':
		v.All, v.Checked = true, map[string]bool{}
	case 'u':
		v.All, v.Checked = false, map[string]bool{}
	case 'r':
		v.All = false
		if v.Checked[it.Ref.Name] {
			delete(v.Checked, it.Ref.Name)
		} else {
			v.Checked[it.Ref.Name] = true
		}
	case 'g':
		if v.Query == "" {
			v.Folded[it.Group] = !v.Folded[it.Group]
		}
	}
}

// applyScope makes the scope ticked the history's and closes the popup;
// the history is read again when its tips change, the cursor and the
// unfolded commits kept where they remain (design D7).
func (m *Model) applyScope() []tea.Cmd {
	v := m.ui.Scope
	s := input.GraphScope{All: v.All}
	if !v.All {
		for _, r := range m.graph.refs.Refs {
			if v.Checked[r.Name] {
				s.Refs = append(s.Refs, r.Name)
			}
		}
	}
	m.closeScope()
	g := &m.graph
	if s.All == g.scope.All && slices.Equal(s.Refs, g.scope.Refs) {
		return nil
	}
	g.scope = s
	if !g.refsOK {
		return nil
	}
	return m.graphRefsRead(graphRefsMsg{gen: g.refsGen, refs: g.refs})
}

// scopeClick is a left click on the popup, item the cell's: on a row the
// cursor goes there and the row is toggled, as Space toggles it; the frame
// does nothing.
func (m *Model) scopeClick(item int) []tea.Cmd {
	v := &m.ui.Scope
	rows := render.ScopeRows(*v)
	i := item - render.ItemScopeRow
	if i < 0 || i >= len(rows) || rows[i].Kind == 'm' {
		return nil
	}
	v.Cursor = rows[i].Key()
	m.scopeToggle(rows[i])
	return nil
}

// scopeMove moves the popup's cursor d rows, past the messages, which take
// none.
func (m *Model) scopeMove(d int) {
	v := &m.ui.Scope
	rows := render.ScopeRows(*v)
	if len(rows) == 0 {
		return
	}
	next := min(max(render.ScopeCursor(rows, v.Cursor)+d, 0), len(rows)-1)
	step := 1
	if d < 0 {
		step = -1
	}
	for next >= 0 && next < len(rows) && rows[next].Kind == 'm' {
		next += step
	}
	if next >= 0 && next < len(rows) {
		v.Cursor = rows[next].Key()
	}
}

// scopeWheel moves the popup's cursor wheelRows rows.
func (m *Model) scopeWheel(b tea.MouseButton) {
	switch b {
	case tea.MouseWheelUp:
		m.scopeMove(-wheelRows)
	case tea.MouseWheelDown:
		m.scopeMove(wheelRows)
	}
}

// scopePaste puts a paste's first line in the search.
func (m *Model) scopePaste(text string) {
	m.ui.Scope.Query += firstLine(text)
	m.scopeFirst()
}
