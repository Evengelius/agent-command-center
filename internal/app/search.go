package app

import (
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/Evengelius/agent-command-center/internal/input"
	"github.com/Evengelius/agent-command-center/internal/render"
)

// The conversation's search (plan 66, brief amendment 68): Ctrl+F's bar at
// the foot of zone B, its field, the matches in turn, the whole entry by
// Tab, and Esc back where the search began; ui-design.md, Zone B, Search.

// searchState is what the application keeps of the bar beside its view
// (render.SearchView): the field, the focus Ctrl+F found and B's entry
// cursor with it, given back by Esc (design D9), and the events applied
// when the matches were found (design D6). The query lives here and in the
// view alone: never in a file, a log or the draft, never sent to a CLI
// (design D2).
type searchState struct {
	field         input.Editor
	prev          string
	entry, review int
	applied       int
}

// openSearch is Ctrl+F on the dashboard (plan 66, design D2): a popup
// closed, the diff in zone B's place closed as Esc closes it, and a panel
// in zone B's place hidden by the compact rule once B has the focus
// (syncUI); the bar opens with its field empty, the entry cursor off, the
// draft as it is.
func (m *Model) openSearch() []tea.Cmd {
	m.closePopups()
	prev := m.ui.Focus
	if m.ui.Diff.Open {
		m.closeDiff()
	}
	m.convFind = searchState{prev: prev, entry: m.ui.Entry, review: m.ui.EntryReview, applied: -1}
	m.ui.Search = render.SearchView{Open: true, Cur: -1, Memo: &render.SearchMemo{}}
	m.ui.Entry, m.ui.EntryReview = 0, 0
	m.ui.Focus = "B"
	return nil
}

// closeSearch is Esc on the bar (design D9): the matches cleared, zone B
// left where the search took it, the focus back where Ctrl+F found it while
// that place is on screen, else the input's.
func (m *Model) closeSearch() {
	if !m.ui.Search.Open {
		return
	}
	prev := m.convFind.prev
	m.ui.Search, m.convFind = render.SearchView{}, searchState{}
	switch {
	case prev == "A" || prev == "C":
		m.ui.Focus = prev
	case prev == "R" && render.PanelShown(m.ui) && !render.PanelCompact(m.s, m.ui):
		m.ui.Focus = "R"
	default:
		m.ui.Focus = "B"
	}
}

// searchKey is a key while the bar is on top (design D1): the field's
// editing, the matches in turn, Tab's whole entry, Esc. A key that leaves
// the conversation (F2, F3, F4, F6, Alt+P, Alt+B, Alt+G) closes the bar and
// goes on; F1, F12, Ctrl+C, Ctrl+O and Shift+PgUp/PgDn go on with the bar
// kept; any other key is the bar's, none reaching the draft.
func (m *Model) searchKey(k tea.KeyPressMsg) ([]tea.Cmd, bool) {
	f := &m.convFind.field
	switch s := k.String(); s {
	case "esc":
		c := m.convFind
		m.closeSearch()
		if c.prev == "B" && m.ui.Focus == "B" && c.entry > 0 && c.entry <= len(m.s.Conversation) {
			// B's entries, where Ctrl+F found them: their cursor back on its
			// entry, on its review's row while that row is there (plan 65).
			m.ui.Entry, m.ui.EntryReview = c.entry, c.review
			if c.review > render.ReviewStops(m.s, m.ui, c.entry) {
				m.ui.EntryReview = 0
			}
		}
		return nil, true
	case "up", "enter":
		m.stepMatch(1) // older: the matches are the newest first
	case "down", "shift+enter":
		m.stepMatch(-1)
	case "tab":
		m.searchWhole()
	case "left":
		f.Left()
	case "right":
		f.Right()
	case "home":
		f.Home()
	case "end":
		f.End()
	case "backspace":
		f.Backspace()
		m.searchAgain(true)
	case "delete":
		f.Delete()
		m.searchAgain(true)
	case "ctrl+v", "ctrl+shift+v":
		// A terminal that sends a raw Ctrl+V: the clipboard's text in the
		// field, as a bracketed paste (the branches popup's way).
		if cb := m.opt.Clipboard; cb != nil {
			return []tea.Cmd{func() tea.Msg {
				text, err := cb.Read()
				if err != nil {
					return toastMsg{"paste failed: " + err.Error()}
				}
				return pasteMsg{text}
			}}, true
		}
	case "f2", "f3", "f4", "f6", "shift+f6", "alt+p", "alt+b", "alt+g":
		m.closeSearch()
		return nil, false
	case "f1", "f12", "ctrl+c", "ctrl+shift+c", "ctrl+o", "shift+pgup", "shift+pgdown":
		return nil, false
	default:
		if text := keyText(k); text != "" && printable(text) {
			f.Insert(text)
			m.searchAgain(true)
		}
	}
	m.syncSearch()
	return nil, true
}

// searchPaste is a paste while the bar is on top: its first line, its
// control characters dropped, as the branches popup's field takes one.
func (m *Model) searchPaste(text string) {
	q := strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f {
			return -1
		}
		return r
	}, firstLine(text))
	if q != "" {
		m.convFind.field.Insert(q)
		m.searchAgain(true)
	}
}

// syncSearch copies the field into the bar's view.
func (m *Model) syncSearch() {
	if m.ui.Search.Open {
		m.ui.Search.Query, m.ui.Search.Cursor = m.convFind.field.Text(), m.convFind.field.Cursor()
	}
}

// searchAgain finds the query's matches in the state as it is (design D4)
// and keeps the current one (designs D5 and D6): the same match while it is
// there (by its entry, line, offset and part), else the newest at or above
// its place; with none yet, the newest at or above zone B's last row shown.
// typed says the query changed: zone B then shows the current match, while
// matches found again after events never move it.
func (m *Model) searchAgain(typed bool) {
	v := &m.ui.Search
	if !v.Open {
		return
	}
	var keep *render.ConvMatch
	if v.Cur >= 0 && v.Cur < len(v.Matches) {
		c := v.Matches[v.Cur]
		keep = &c
	}
	v.Query, v.Cursor = m.convFind.field.Text(), m.convFind.field.Cursor()
	v.Matches, v.Capped = render.ConvMatches(m.s, m.ui, v.Query, v.Memo)
	m.convFind.applied = m.s.Applied
	v.Cur = -1
	if len(v.Matches) == 0 {
		return
	}
	var at render.SearchPos
	found := false
	switch {
	case keep != nil:
		at, found = render.SearchPos{Item: keep.Item, Line: keep.Line, Off: keep.Off}, true
		for i, c := range v.Matches {
			if c.Item == keep.Item && c.Line == keep.Line && c.Off == keep.Off && c.Target == keep.Target {
				v.Cur = i
				break
			}
		}
	case m.screen != nil && m.screen.Conv.HasBottom:
		at, found = m.screen.Conv.Bottom, true
	}
	if v.Cur < 0 {
		v.Cur = 0 // the newest, when none is at or above the place
		if found {
			for i, c := range v.Matches {
				if c.AtOrBefore(at) {
					v.Cur = i
					break
				}
			}
		}
	}
	if typed {
		m.revealMatch()
	}
}

// stepMatch makes the next match current, d 1 the older and -1 the newer,
// round at either end (design D5), and shows it.
func (m *Model) stepMatch(d int) {
	v := &m.ui.Search
	if n := len(v.Matches); n > 0 {
		v.Cur = ((v.Cur+d)%n + n) % n
		m.revealMatch()
	}
}

// revealMatch shows the current match (design D5): the folds it is behind
// opened, as a click opens them, then zone B scrolled the least that shows
// its row and held there, "↓ N new" counting what comes, as the entry
// cursor's reveal does (revealRow), from the frame drawn last.
func (m *Model) revealMatch() {
	v := m.ui.Search
	if v.Cur < 0 || v.Cur >= len(v.Matches) || m.screen == nil {
		return
	}
	mt := v.Matches[v.Cur]
	for _, key := range render.MatchFolds(m.s, m.ui, mt) {
		if m.ui.Unfolded == nil {
			m.ui.Unfolded = map[string]bool{}
		}
		m.ui.Unfolded[key] = true
	}
	c := m.screen.Conv
	if c.Room <= 0 || c.W <= 0 {
		return // no conversation drawn: nothing to scroll
	}
	row, rows, ok := render.ConvMatchRow(m.s, m.ui, c.W, c.Room, mt)
	if !ok {
		return
	}
	bottom := max(rows-c.Room, 0)
	top := bottom
	if m.ui.ConvHold {
		top = min(max(m.ui.ConvTop, 0), bottom)
	}
	to := top
	switch {
	case row < top:
		to = row
	case row >= top+c.Room:
		to = row - c.Room + 1
	}
	to = min(max(to, 0), bottom)
	if !m.ui.ConvHold {
		m.ui.ConvSeen = len(m.s.Conversation)
	}
	m.ui.ConvHold, m.ui.ConvTop = true, to
	m.screen.Conv.Top, m.screen.Conv.Rows = to, rows // a second step before the next frame starts here
}

// searchWhole is Tab (design D8): the current match's entry whole, in its
// full view, as Enter shows it from the entry cursor, with the match's row
// in view; an entry B already shows whole has no full view. Esc there comes
// back to zone B and the bar.
func (m *Model) searchWhole() {
	v := m.ui.Search
	if v.Cur < 0 || v.Cur >= len(v.Matches) {
		return
	}
	mt := v.Matches[v.Cur]
	if mt.Item >= len(m.s.Conversation) || !render.Expandable(m.s.Conversation[mt.Item]) {
		return
	}
	m.openView("", mt.Item+1)
	if row, ok := render.EntryViewRow(m.s, m.ui, mt); ok {
		m.ui.InspectTop = max(row-2, 0) // two rows above it, as a reveal leaves room
	}
}

// searchAfterMouse closes the bar once a click opened a popup or a window
// over it, or gave another zone the focus (design D9); the focus stays where
// the click put it.
func (m *Model) searchAfterMouse(msg tea.MouseMsg) {
	if _, click := msg.(tea.MouseClickMsg); !click || !m.ui.Search.Open {
		return
	}
	top := render.Layers(m.s, m.ui).Top
	if top != render.LayerSearch && top != render.LayerNone || m.ui.Focus != "B" {
		focus := m.ui.Focus
		m.closeSearch()
		m.ui.Focus = focus
	}
}
