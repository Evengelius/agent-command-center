package app

import (
	"errors"
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/Evengelius/agent-command-center/internal/input"
	"github.com/Evengelius/agent-command-center/internal/render"
)

// The Explorer (plan 17, amendment 12): the project as a tree, each folder
// read in a command when it is opened and git's status asked once when the
// Explorer opens, each result carrying the generation it was asked under, so
// that one for an Explorer since closed is dropped. Since plan 43
// (amendment 44) it is a panel docked between zone B and the sidebar, with
// its own focus: F3, its tile and its "⌄" show and hide it (design D5); Esc
// goes back one level, then to zone B, the panel kept (design D7).

// Messages.
type (
	dirMsg struct {
		gen       int
		rel       string
		entries   []input.TreeEntry
		truncated bool
		err       error
	}
	statusMsg struct {
		gen   int
		marks input.GitMarks
		err   error
	}
)

// openExplorer opens the Explorer as it was left: the project and every
// folder shown open read again, git's status asked, Source Control hidden
// from the docked place (plan 44, design D1), its dropdown and question with
// it (plan 47, design D4). The status is the one Source Control reads too
// (design D2), its markers taken by applySnapshot.
func (m *Model) openExplorer() []tea.Cmd {
	m.explorerGen++
	m.ui.Source.Open = false
	m.dropSourcePopups()
	v := &m.ui.Explorer
	root := m.fileRoot()
	v.Open, v.Name = true, filepath.Base(root)
	if v.Dirs == nil {
		v.Dirs = map[string]*render.ExplorerDir{}
	}
	if v.Opened == nil {
		v.Opened = map[string]bool{}
	}
	if v.Git.Marks == nil && v.Git.Note == "" {
		v.Git.Note = render.GitPending
	}
	m.dropFileSelection()
	cmds := []tea.Cmd{m.readDir("")}
	for _, r := range render.ExplorerRows(*v) {
		if r.Open && !r.Deleted {
			cmds = append(cmds, m.readDir(r.Path))
		}
	}
	if _, ok := m.sourceGit(); ok {
		return append(cmds, m.readSnapshot())
	}
	gen, git := m.explorerGen, m.git()
	return append(cmds, func() tea.Msg {
		marks, err := git.Status(root)
		return statusMsg{gen, marks, err}
	})
}

// readDir reads the folder rel of the project in a command.
func (m *Model) readDir(rel string) tea.Cmd {
	gen, root := m.explorerGen, m.fileRoot()
	return func() tea.Msg {
		entries, truncated, err := input.ListDir(root, rel)
		return dirMsg{gen, rel, entries, truncated, err}
	}
}

// dirRead shows a folder read, or why it cannot be, and reads each folder
// in it left open but not read yet: the folders above a file the viewer
// goes back to.
func (m *Model) dirRead(msg dirMsg) []tea.Cmd {
	v := &m.ui.Explorer
	if msg.gen != m.explorerGen || !v.Open {
		return nil
	}
	d := &render.ExplorerDir{Truncated: msg.truncated}
	if msg.err != nil {
		d.State = msg.err.Error()
	}
	var cmds []tea.Cmd
	for _, e := range msg.entries {
		d.Entries = append(d.Entries, render.ExplorerEntry{Name: e.Name, Dir: e.Dir})
		p := e.Name
		if msg.rel != "" {
			p = msg.rel + "/" + e.Name
		}
		if e.Dir && v.Opened[p] && v.Dirs[p] == nil {
			cmds = append(cmds, m.readDir(p))
		}
	}
	v.Dirs[msg.rel] = d
	return cmds
}

// gitRead shows git's markers, or why there are none: not a repository and
// git missing are notes, any other error a failure.
func (m *Model) gitRead(msg statusMsg) {
	v := &m.ui.Explorer
	if msg.gen != m.explorerGen || !v.Open {
		return
	}
	switch {
	case msg.err == nil:
		v.Git = render.NewGitView(msg.marks.Files, msg.marks.Ignored)
	case errors.As(msg.err, new(input.NotRepoError)):
		v.Git = render.GitView{Note: "not a git repository"}
	case errors.Is(msg.err, input.ErrNoGit):
		v.Git = render.GitView{Note: "git not found"}
	default:
		v.Git = render.GitView{Note: "git: " + msg.err.Error(), Failed: true}
	}
}

// closeExplorer hides the Explorer's panel, keeping its folders and cursor
// for the next F3; a read under way is dropped. Zone B takes the focus
// back (plan 43, design D5).
func (m *Model) closeExplorer() {
	m.explorerGen++
	if m.ui.Explorer.Open {
		m.ui.Explorer.Open = false
		m.ui.Focus = "B"
	}
}

// toggleExplorer shows the Explorer's panel, or hides it while shown: F3 on
// the dashboard, a click on its tile or on its "⌄" (plan 43, design D5).
func (m *Model) toggleExplorer() []tea.Cmd {
	if m.ui.Explorer.Open {
		m.closeExplorer()
		return nil
	}
	return m.showExplorer()
}

// showExplorer shows the panel as it was left, its folders read again, and
// gives it the keyboard focus (plan 43, design D5).
func (m *Model) showExplorer() []tea.Cmd {
	var cmds []tea.Cmd
	if !m.ui.Explorer.Open {
		cmds = m.openExplorer()
	}
	m.ui.Focus = "R"
	return cmds
}

// closeWindows closes the floating windows, the quick open and the viewer:
// F2 in either (plan 17, amendment 12). The Explorer's panel is no window
// (plan 43, design D8): it stays, and the compact rule hides it when zone B
// takes the focus in its place.
func (m *Model) closeWindows() {
	m.closeFinder()
	m.closeFile()
}

// draftBeside says the draft takes the clipboard's keys and pastes: zone B
// has the focus, or the panel docked beside zone B has it, the draft on
// screen (plan 43, design D8; plan 44, design D5). The panel in zone B's
// place, or the diff there, leaves them to no one, as zones A and C do.
func (m *Model) draftBeside() bool {
	if m.ui.Diff.Open || m.msgFocused() {
		return false // the commit message takes the keys and pastes (plan 45, design D2)
	}
	return m.ui.Focus == "B" || m.ui.Focus == "R" && render.PanelShown(m.ui) && !render.PanelCompact(m.s, m.ui)
}

// backToExplorer closes the viewer and shows the Explorer with the focus,
// its cursor on the file, each folder above it open; a file outside the
// project leaves the cursor where it was (plan 43, design D7: Esc from the
// viewer returns to the panel).
func (m *Model) backToExplorer() []tea.Cmd {
	rel := m.ui.File.Rel
	if p := m.ui.File.Path; rel == "" && p != "" {
		if !filepath.IsAbs(p) {
			p = filepath.Join(m.fileRoot(), p)
		}
		if r, err := filepath.Rel(m.fileRoot(), p); err == nil && r != "." && !strings.HasPrefix(r, "..") && !filepath.IsAbs(r) {
			rel = filepath.ToSlash(r)
		}
	}
	m.closeFile()
	v := &m.ui.Explorer
	if v.Opened == nil {
		v.Opened = map[string]bool{}
	}
	if rel != "" {
		for p := render.ParentPath(rel); p != ""; p = render.ParentPath(p) {
			v.Opened[p] = true
		}
		v.Cursor = rel
	}
	cmds := m.openExplorer()
	m.ui.Focus = "R"
	return cmds
}

// explorerKey handles a key while the Explorer's panel has the focus: its
// keys as in plan 17, the global ones passed on, F6 and Tab to the next
// zone, Alt+P and F4 to the selectors (plan 43, designs D7 and D8), Alt+B
// to the branches popup (plan 59).
func (m *Model) explorerKey(k tea.KeyPressMsg) (cmds []tea.Cmd, done bool) {
	s := k.String()
	switch s {
	case "f1", "f4", "f6", "shift+f6", "tab", "shift+tab", "alt+p", "alt+b", "f12", "ctrl+c", "ctrl+shift+c", "ctrl+o", "ctrl+r",
		"shift+pgup", "shift+pgdown", "ctrl+q", "F":
		return nil, false
	}
	if k.Text == "?" {
		return nil, false // the key overlay
	}
	v := &m.ui.Explorer
	rows := render.ExplorerRows(*v)
	cur := render.ExplorerCursor(rows, v.Cursor)
	room := m.explorerRoom()
	switch s {
	case "esc":
		// Out of the folder the cursor is in; at the top level, back to zone
		// B, the panel kept (design D7).
		if cur >= 0 {
			if p := render.ParentPath(rows[cur].Path); p != "" {
				v.Opened[p] = false
				m.explorerTo(p)
				return nil, true
			}
		}
		m.ui.Focus = "B"
	case "f3":
		m.closeExplorer()
	case "f2":
		return m.focusCard("", true), true
	case "alt+v", "ctrl+v":
		// The images, or a raw Ctrl+V's clipboard, go to the draft beside
		// the docked panel, as the terminal's paste does (design D8).
		if m.draftBeside() {
			return m.editorKey(k), true
		}
	case "/":
		return m.openFinder(), true
	case "up":
		m.explorerMove(rows, cur, -1, room)
	case "down":
		m.explorerMove(rows, cur, 1, room)
	case "pgup":
		m.explorerMove(rows, cur, -room, room)
	case "pgdown":
		m.explorerMove(rows, cur, room, room)
	case "home", "ctrl+home":
		m.explorerMove(rows, cur, -len(rows), room)
	case "end", "ctrl+end":
		m.explorerMove(rows, cur, len(rows), room)
	case "right":
		if cur >= 0 && rows[cur].Dir {
			if !rows[cur].Open {
				return m.openDir(rows[cur]), true
			}
			if cur+1 < len(rows) && rows[cur+1].Path != "" && rows[cur+1].Depth > rows[cur].Depth {
				m.explorerTo(rows[cur+1].Path)
			}
		}
	case "left":
		switch {
		case cur < 0:
		case rows[cur].Open:
			v.Opened[rows[cur].Path] = false
		case render.ParentPath(rows[cur].Path) != "":
			m.explorerTo(render.ParentPath(rows[cur].Path))
		}
	case "enter":
		if cur >= 0 {
			return m.explorerEnter(rows[cur]), true
		}
	}
	return nil, true
}

// explorerEnter opens or closes a folder, or shows a file in the viewer; a
// deleted file has nothing to show, its path row says so.
func (m *Model) explorerEnter(r render.ExplorerRow) []tea.Cmd {
	m.ui.Explorer.Cursor = r.Path
	switch {
	case !r.Dir && r.Deleted:
		return nil
	case !r.Dir:
		return m.openFile(render.FileTarget{Path: r.Path, Opened: "Explorer"})
	case r.Open:
		m.ui.Explorer.Opened[r.Path] = false
		return nil
	}
	return m.openDir(r)
}

// openDir opens the folder r and reads it again; until the read arrives it
// shows what it held, or "reading…". A deleted folder is not read: its
// entries are git's.
func (m *Model) openDir(r render.ExplorerRow) []tea.Cmd {
	m.ui.Explorer.Opened[r.Path] = true
	m.ui.Explorer.Cursor = r.Path
	if r.Deleted {
		return nil
	}
	return []tea.Cmd{m.readDir(r.Path)}
}

// explorerMove moves the cursor d entries, notes skipped, kept in the tree
// and in view.
func (m *Model) explorerMove(rows []render.ExplorerRow, cur, d, room int) {
	var entries []int
	at := 0
	for i, r := range rows {
		if r.Path == "" {
			continue
		}
		if i <= cur {
			at = len(entries)
		}
		entries = append(entries, i)
	}
	if len(entries) == 0 {
		return
	}
	i := entries[min(max(at+d, 0), len(entries)-1)]
	m.ui.Explorer.Cursor = rows[i].Path
	m.ui.Explorer.Top = render.ListTop(m.ui.Explorer.Top, i, room, len(rows))
}

// explorerTo puts the cursor on the row at path, in view.
func (m *Model) explorerTo(path string) {
	v := &m.ui.Explorer
	v.Cursor = path
	rows := render.ExplorerRows(*v)
	v.Top = render.ListTop(v.Top, max(render.ExplorerCursor(rows, path), 0), m.explorerRoom(), len(rows))
}

// explorerRoom is the tree's rows in the panel on this screen, what PgUp
// and PgDn move by (plan 43, design D6).
func (m *Model) explorerRoom() int { return max(render.ExplorerRoom(m.s, m.ui), 1) }

// explorerClick handles a left click on the panel, item the cell's (plan
// 43, design D7): the panel takes the focus; on a row the cursor goes to
// it, on the cursor's row it is Enter (plan 17); the header's "⌄" hides the
// panel. It selects nothing: the terminal's own selection, with F12, copies
// from it.
func (m *Model) explorerClick(item int) []tea.Cmd {
	m.ui.Sel = render.Selection{}
	if item == render.ItemPanelHide {
		m.closeExplorer()
		return nil
	}
	m.ui.Focus = "R"
	v := &m.ui.Explorer
	rows := render.ExplorerRows(*v)
	i := item - 1 // the row's index, by the cell's item
	if i < 0 || i >= len(rows) || rows[i].Path == "" {
		return nil
	}
	cur := render.ExplorerCursor(rows, v.Cursor)
	if i == cur {
		return m.explorerEnter(rows[i])
	}
	// The cursor on the row clicked, the rows where the frame drew them.
	v.Top = render.ListTop(v.Top, max(cur, 0), m.explorerRoom(), len(rows))
	v.Cursor = rows[i].Path
	return nil
}

// explorerWheel moves the panel's cursor wheelRows entries, the pointer
// over the panel (plan 43, design D7).
func (m *Model) explorerWheel(b tea.MouseButton) {
	v := &m.ui.Explorer
	rows := render.ExplorerRows(*v)
	cur := render.ExplorerCursor(rows, v.Cursor)
	switch b {
	case tea.MouseWheelUp:
		m.explorerMove(rows, cur, -wheelRows, m.explorerRoom())
	case tea.MouseWheelDown:
		m.explorerMove(rows, cur, wheelRows, m.explorerRoom())
	}
}
