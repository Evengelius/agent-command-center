package app

import (
	tea "charm.land/bubbletea/v2"

	"github.com/Evengelius/agent-command-center/internal/render"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// Zone A's tree, zone C's list and the agent inspection (ui-design.md, Zone
// A; Zone C; Agent inspection).

// openInspect shows agent key full screen, on its Messages tab, following.
func (m *Model) openInspect(key string) {
	m.ui.Inspect, m.ui.Tab = key, 0
	m.selectAgent(key)
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
			m.ui.Inspect = key
			m.selectAgent(key)
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
// an agent's; on the telemetry pane's headings Enter opens the details and
// ← → fold its sections (plan 24, design D6).
func (m *Model) treeKey(k tea.KeyPressMsg) []tea.Cmd {
	nodes := render.Nodes(m.s, m.ui)
	if len(nodes) == 0 {
		return nil
	}
	// No cursor yet, or its node is gone: the last tree node, the newest
	// activity (TELEMETRY when the tree is empty); the telemetry pane's
	// headings come after it (plan 24).
	i := 0
	for j, n := range nodes {
		if !render.IsTelemetryNode(n.ID) {
			i = j
		}
	}
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
		if render.IsTelemetryNode(nodes[i].ID) {
			m.openDetails()
			return nil
		}
		if a := nodes[i].Agent; a != "" {
			m.openInspect(a)
			return nil
		}
		return m.toast("the orchestrator's own: its messages are in zone B")
	default:
		return nil
	}
	m.ui.ACursor = nodes[i].ID
	if k.String() == "up" || k.String() == "down" {
		m.paneReveal('A', m.ui.ACursor)
	}
	return nil
}

// listKey moves zone C's selection in the order the zone shows, and inspects
// the selected agent.
func (m *Model) listKey(k tea.KeyPressMsg) []tea.Cmd {
	order := render.AgentOrder(m.s)
	switch k.String() {
	case "up", "[":
		m.selectAgent(step(order, m.ui.Selected, false))
		m.paneReveal('C', m.ui.Selected)
	case "down", "]":
		m.selectAgent(step(order, m.ui.Selected, true))
		m.paneReveal('C', m.ui.Selected)
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

// openDetails shows the telemetry full screen at its top: the newest error
// first (ui-design.md, Zone A: Errors and telemetry); zone A keeps the
// focus, so Esc returns to it.
func (m *Model) openDetails() {
	m.ui.Focus = "A"
	m.ui.Details = true
	m.ui.Sel = render.Selection{}
	m.ui.InspectHold, m.ui.InspectTop = true, 0
	if m.screen != nil {
		m.screen.Inspect.Top = 0
	}
}

// openOverlay shows the key overlay from its top. The view under it keeps
// its scroll, given back when the overlay closes.
func (m *Model) openOverlay() {
	m.closePopups() // nothing stays open under it (plan 49, design D7)
	if !m.ui.Keys {
		m.under = scrollPos{m.ui.InspectTop, m.ui.InspectHold}
	}
	m.ui.Keys = true
	m.ui.Sel = render.Selection{}
	m.ui.InspectHold, m.ui.InspectTop = true, 0
	if m.screen != nil {
		m.screen.Inspect.Top = 0
	}
}

// overlayKey handles a key while the key overlay is shown; done is false
// for the keys that keep their dashboard meaning.
func (m *Model) overlayKey(k tea.KeyPressMsg) (cmds []tea.Cmd, done bool) {
	switch s := k.String(); s {
	case "esc", "f1", "?":
		if s == "esc" && !m.ui.Sel.Empty() {
			m.ui.Sel = render.Selection{} // Esc clears a selection first
			return nil, true
		}
		m.ui.Keys = false
		m.ui.Sel = render.Selection{}
		m.ui.InspectTop, m.ui.InspectHold = m.under.top, m.under.hold
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
	case "ctrl+c", "ctrl+shift+c", "f12", "ctrl+o", "ctrl+r", "F":
		return nil, false
	}
	return nil, true
}

// scrollPos is a full-screen view's scroll: InspectTop and InspectHold.
type scrollPos struct {
	top  int
	hold bool
}

// detailsKey handles a key while the telemetry is full screen; done is
// false for the keys that keep their dashboard meaning, as in an
// inspection.
func (m *Model) detailsKey(k tea.KeyPressMsg) (cmds []tea.Cmd, done bool) {
	switch s := k.String(); s {
	case "esc":
		if !m.ui.Sel.Empty() {
			m.ui.Sel = render.Selection{} // Esc clears a selection first
			return nil, true
		}
		m.ui.Details = false // back to A, still focused, its cursor as it was
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

// openView shows approval id, or else conversation entry n (1-based), whole,
// full screen, from its top (ui-design.md, Approvals: Enter; Zone B,
// Expanding).
func (m *Model) openView(id string, n int) {
	m.ui.View, m.ui.ViewEntry = id, n
	m.ui.Sel = render.Selection{}
	m.ui.InspectHold, m.ui.InspectTop = true, 0
	if m.screen != nil {
		m.screen.Inspect.Top = 0
	}
}

// viewKey handles a key while an approval or an entry is shown whole: a, d
// and c answer an approval as on its card, which keeps the focus; done is
// false for the keys that keep their dashboard meaning.
func (m *Model) viewKey(k tea.KeyPressMsg) (cmds []tea.Cmd, done bool) {
	if m.ui.View != "" && m.s.Approvals[m.ui.View] == nil || m.ui.ViewEntry > len(m.s.Conversation) { // gone with a resumed session
		m.closeView()
		return nil, false
	}
	if ap := m.s.Approvals[m.ui.View]; m.ui.View != "" && ap.Question && ap.Status == state.Pending && m.ui.Card == ap.ID {
		switch k.String() {
		case "1", "2", "3", "4", "left", "right":
			return m.questionKey(ap, k.String())
		case "a": // refused while a question has no choice: the view stays open
			if !m.pickComplete(ap) {
				return m.toast("choose an option for each question first"), true
			}
			m.closeView()
			return m.questionKey(ap, "a")
		}
	}
	switch s := k.String(); s {
	case "esc":
		if !m.ui.Sel.Empty() {
			m.ui.Sel = render.Selection{} // Esc clears a selection first
			return nil, true
		}
		m.closeView() // back to the card, or to the input once it is answered
	case "a", "d", "c", "s":
		if m.ui.View == "" || m.ui.Card != m.ui.View || m.s.Approvals[m.ui.View].Status != state.Pending {
			return nil, true // answered or expired: nothing to answer
		}
		m.closeView()
		return m.cardKey(k)
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
	case "ctrl+c", "ctrl+shift+c", "f12", "ctrl+o", "shift+pgup", "shift+pgdown", "ctrl+r", "F":
		return nil, false
	}
	return nil, true
}

func (m *Model) closeView() {
	m.ui.View, m.ui.ViewEntry = "", 0 // an entry's cursor stays on it
	m.ui.Sel = render.Selection{}
	m.ui.InspectHold, m.ui.InspectTop = false, 0
	if m.ui.Card == "" {
		m.ui.Focus = "B"
	}
}

// entryKey handles a key while zone B's entry cursor is on (ui-design.md,
// Zone B, Expanding): ↑ ↓ move it, Enter shows its entry whole, → expands a
// folded run and opens nothing, Esc returns to the input. On a thinking
// phase's summary row, → opens its details and ← folds them (plan 22, design
// 8). done is false for the keys that keep their meaning; any other key
// leaves the cursor and reaches the input, so typing types.
func (m *Model) entryKey(k tea.KeyPressMsg) (cmds []tea.Cmd, done bool) {
	if m.ui.Entry > len(m.s.Conversation) || m.ui.Focus != "B" || m.ui.Card != "" {
		m.ui.Entry, m.ui.EntryReview = 0, 0
		return nil, false
	}
	if m.ui.EntryReview > 0 {
		return m.reviewKey(k) // on a turn's review rows (plan 65, design D8)
	}
	// On a folded call, the cursor is its run's row: the run's first entry,
	// also when the call was folded after the cursor came to it (plan 19).
	key, first, open, folded, inRun := render.FoldOf(m.s, m.ui, m.ui.Entry)
	if folded {
		m.ui.Entry = first
	}
	// A thinking phase drawn as its summary row (ended and shown): the key of
	// its details in ui.Unfolded; "" for any other entry (plan 22, design 8).
	thought := ""
	if e := m.s.Conversation[m.ui.Entry-1]; e.Role == "thinking" && render.Expandable(e) {
		thought = render.ThoughtKey(e)
	}
	switch k.String() {
	case "up":
		m.stepCursor(-1)
	case "down":
		if !m.stepCursor(1) {
			m.pastNewest()
		}
	case "enter":
		if folded {
			m.unfold(key, true)
			break
		}
		m.openView("", m.ui.Entry)
	case "right":
		// → unfolds, as in zone A's tree; only Enter opens (the owner's live
		// review, 2026-10-20: → opening an entry, as Enter does, confused it
		// with unfolding).
		switch {
		case folded:
			m.unfold(key, true)
		case thought != "":
			m.unfold(thought, true)
		}
	case "left":
		if thought != "" && m.ui.Unfolded[thought] {
			m.unfold(thought, false) // the details fold; the cursor stays on the row
			break
		}
		if !inRun || !open {
			m.ui.Entry = 0
			return nil, false
		}
		m.unfold(key, false)
		m.ui.Entry = first
		m.reveal(first)
	case "esc":
		if !m.ui.Sel.Empty() {
			m.ui.Sel = render.Selection{} // Esc clears a selection first
			return nil, true
		}
		m.ui.Entry = 0
	case "pgup", "pgdown", "shift+pgup", "shift+pgdown", "ctrl+c", "ctrl+shift+c", "f12", "ctrl+o", "ctrl+r":
		return nil, false
	default:
		m.ui.Entry = 0
		return nil, false
	}
	return nil, true
}

// stepCursor moves the entry cursor to its next stop (d 1) or its previous
// one (d -1) and scrolls B to it; false, the cursor where it was, when
// there is none. An entry's stops are the entry when the cursor stops on it
// (moveEntry's), then its turn's review rows, in that order (plan 65,
// design D8).
func (m *Model) stepCursor(d int) bool {
	n, k := m.ui.Entry, m.ui.EntryReview
	targets := render.Targets(m.s, m.ui)
	reviews := render.ReviewStopsAll(m.s, m.ui)
	switch {
	case d > 0 && k < reviews[n]:
		return m.putCursor(n, k+1)
	case d < 0 && k > 1:
		return m.putCursor(n, k-1)
	case d < 0 && k == 1 && targets[n-1]:
		return m.putCursor(n, 0)
	}
	for n += d; n >= 1 && n <= len(m.s.Conversation); n += d {
		switch {
		case d > 0 && targets[n-1]:
			return m.putCursor(n, 0)
		case reviews[n] > 0 && d > 0:
			return m.putCursor(n, 1)
		case reviews[n] > 0:
			return m.putCursor(n, reviews[n])
		case targets[n-1]:
			return m.putCursor(n, 0)
		}
	}
	return false
}

// pastNewest is ↓ past zone B's newest stop: the input, or the summary
// suggestion while it is shown (plan 70, design D3).
func (m *Model) pastNewest() {
	m.ui.Entry, m.ui.EntryReview = 0, 0
	if render.SummaryShown(m.s) {
		m.focusSummary()
	}
}

// lastStop puts the entry cursor on zone B's newest stop: ↑ from an empty
// input (ui-design.md, Zone B, Expanding); false for none.
func (m *Model) lastStop() bool {
	m.ui.Entry, m.ui.EntryReview = len(m.s.Conversation)+1, 0
	if !m.stepCursor(-1) {
		m.ui.Entry = 0
		return false
	}
	return true
}

// putCursor puts the entry cursor on entry n's stop k and scrolls B to it:
// the entry whole, or the review's row.
func (m *Model) putCursor(n, k int) bool {
	m.ui.Entry, m.ui.EntryReview = n, k
	if k == 0 {
		m.reveal(n)
		return true
	}
	if m.screen != nil && n < len(m.screen.Conv.Starts) {
		m.revealRow(m.screen.Conv.Starts[n] - 1 - render.ReviewRowsAfter(m.s, m.ui, n, k))
	}
	return true
}

// revealRow scrolls B the least that shows its row r.
func (m *Model) revealRow(r int) {
	c := m.screen.Conv
	top := c.Top
	if m.ui.ConvHold {
		top = min(max(m.ui.ConvTop, 0), max(c.Rows-c.Room, 0))
	}
	room := c.Room
	if m.ui.ConvHold && render.NewShown(m.s, m.ui.ConvSeen) > 0 {
		room = max(room-1, 1) // the "↓ N new" badge covers the last row (reveal)
	}
	to := top
	switch {
	case r < top:
		to = r
	case r >= top+room:
		to = r - room + 1
	}
	if to != top {
		m.scroll(to - top)
	}
}

// reviewKey handles a key while the entry cursor is on a turn's review rows
// (plan 65, design D8): ↑ ↓ move it; Enter or → on the summary unfolds the
// files, Enter or ← folds them; Enter on a file opens its diff, ← there
// folds the list, the cursor back on the summary; Esc returns to the input.
func (m *Model) reviewKey(k tea.KeyPressMsg) (cmds []tea.Cmd, done bool) {
	t := render.ReviewOf(m.s, m.ui.Entry)
	if t == 0 || m.ui.EntryReview > render.ReviewStops(m.s, m.ui, m.ui.Entry) {
		m.ui.EntryReview = 0 // its review gone or folded under it: the entry's own stop
		if t == 0 || !render.Targets(m.s, m.ui)[m.ui.Entry-1] {
			m.ui.Entry = 0
		}
		return nil, false
	}
	key := render.ReviewKey(t)
	switch k.String() {
	case "up":
		m.stepCursor(-1)
	case "down":
		if !m.stepCursor(1) {
			m.pastNewest()
		}
	case "enter":
		if m.ui.EntryReview == 1 {
			m.unfold(key, !m.ui.Unfolded[key])
			break
		}
		if f := render.ReviewFile(m.s, m.ui.Entry, m.ui.EntryReview); f >= 0 {
			return m.openTurnDiff(t, f), true
		}
	case "right":
		if m.ui.EntryReview == 1 {
			m.unfold(key, true)
		}
	case "left":
		switch {
		case m.ui.EntryReview > 1:
			m.unfold(key, false)
			m.putCursor(m.ui.Entry, 1)
		case m.ui.Unfolded[key]:
			m.unfold(key, false)
		default:
			m.ui.Entry, m.ui.EntryReview = 0, 0
			return nil, false
		}
	case "esc":
		if !m.ui.Sel.Empty() {
			m.ui.Sel = render.Selection{} // Esc clears a selection first
			return nil, true
		}
		m.ui.Entry, m.ui.EntryReview = 0, 0
	case "pgup", "pgdown", "shift+pgup", "shift+pgdown", "ctrl+c", "ctrl+shift+c", "f12", "ctrl+o", "ctrl+r":
		return nil, false
	default:
		m.ui.Entry, m.ui.EntryReview = 0, 0
		return nil, false
	}
	return nil, true
}

// moveEntry puts the entry cursor on the first expandable entry from n
// (1-based) in direction d, and scrolls B to it; false if there is none. A
// folded run's row is one; the calls it folds are not (plan 19).
func (m *Model) moveEntry(n, d int) bool {
	targets := render.Targets(m.s, m.ui)
	for ; n >= 1 && n <= len(m.s.Conversation); n += d {
		if targets[n-1] {
			m.ui.Entry = n
			m.reveal(n)
			return true
		}
	}
	return false
}

// unfold expands zone B's run of calls key, or a thinking phase's details
// (render.ThoughtKey, plan 22), or folds it again (plan 19). Expanding holds
// B's view where it is, so the calls open under the run's row instead of
// pushing it up out of view while B follows its end.
func (m *Model) unfold(key string, open bool) {
	if m.ui.Unfolded == nil {
		m.ui.Unfolded = map[string]bool{}
	}
	if !open {
		delete(m.ui.Unfolded, key)
		return
	}
	m.ui.Unfolded[key] = true
	if m.screen != nil && !m.ui.ConvHold {
		m.ui.ConvHold, m.ui.ConvTop, m.ui.ConvSeen = true, m.screen.Conv.Top, len(m.s.Conversation)
	}
}

// reveal scrolls B the least that shows entry n (1-based) whole, its end
// when it is taller than B: the cursor's row is its last.
func (m *Model) reveal(n int) {
	if m.screen == nil || n >= len(m.screen.Conv.Starts) {
		return
	}
	c := m.screen.Conv
	top := c.Top
	if m.ui.ConvHold {
		top = min(max(m.ui.ConvTop, 0), max(c.Rows-c.Room, 0))
	}
	// The "↓ N new" badge covers B's last row while the view is held with new
	// entries: one row less to show the marked entry in (plan 6, item 2). It
	// counts the entries zone B draws, as render.NewShown does, so a Codex step
	// that left no row is no news (plan 22, correction C10).
	room := c.Room
	if m.ui.ConvHold && render.NewShown(m.s, m.ui.ConvSeen) > 0 {
		room = max(room-1, 1)
	}
	to := min(top, c.Starts[n-1])
	if end := c.Starts[n]; end > to+room {
		to = end - room
	}
	if to != top {
		m.scroll(to - top)
	}
}
