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
// that one for an Explorer since closed is dropped. Esc goes back one level;
// F3 closes every window.

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
// folder shown open read again, git's status asked.
func (m *Model) openExplorer() []tea.Cmd {
	m.explorerGen++
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
		v.Git.Note = "git status…"
	}
	m.dropFileSelection()
	cmds := []tea.Cmd{m.readDir("")}
	for _, r := range render.ExplorerRows(*v) {
		if r.Open && !r.Deleted {
			cmds = append(cmds, m.readDir(r.Path))
		}
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

// closeExplorer closes the Explorer, keeping its folders and cursor for the
// next F3; a read under way is dropped.
func (m *Model) closeExplorer() {
	m.explorerGen++
	m.ui.Explorer.Open = false
}

// closeWindows closes the quick open, the viewer and the Explorer: F3 in
// any of them (plan 17, amendment 12).
func (m *Model) closeWindows() {
	m.closeFinder()
	m.closeFile()
	m.closeExplorer()
}

// backToExplorer closes the viewer and shows the Explorer, its cursor on
// the file, each folder above it open; a file outside the project leaves
// the cursor where it was.
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
	return m.openExplorer()
}

// explorerKey handles a key while the Explorer is in front: all but the
// global ones are its own.
func (m *Model) explorerKey(k tea.KeyPressMsg) (cmds []tea.Cmd, done bool) {
	s := k.String()
	switch s {
	case "f1", "f4", "f12", "ctrl+c", "ctrl+shift+c", "ctrl+q":
		return nil, false
	}
	if k.Text == "?" {
		return nil, false // the key overlay
	}
	v := &m.ui.Explorer
	rows := render.ExplorerRows(*v)
	cur := render.ExplorerCursor(rows, v.Cursor)
	room := render.FileRows(m.ui)
	switch s {
	case "esc":
		if cur < 0 {
			m.closeExplorer()
			return nil, true
		}
		if p := render.ParentPath(rows[cur].Path); p != "" {
			v.Opened[p] = false
			m.explorerTo(p)
		} else {
			m.closeExplorer()
		}
	case "f3":
		m.closeWindows()
	case "f2":
		m.closeWindows()
		return m.focusCard("", true), true
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
	v.Top = render.ListTop(v.Top, max(render.ExplorerCursor(rows, path), 0), render.FileRows(m.ui), len(rows))
}

// explorerMouse handles the mouse while the Explorer is in front: the wheel
// moves the cursor, a click puts it on a row, a click on the cursor's row
// is Enter. It selects nothing: the terminal's own selection, with F12,
// copies from it.
func (m *Model) explorerMouse(msg tea.MouseMsg) []tea.Cmd {
	mo := msg.Mouse()
	v := &m.ui.Explorer
	rows := render.ExplorerRows(*v)
	cur := render.ExplorerCursor(rows, v.Cursor)
	room := render.FileRows(m.ui)
	switch msg.(type) {
	case tea.MouseWheelMsg:
		switch mo.Button {
		case tea.MouseWheelUp:
			m.explorerMove(rows, cur, -wheelRows, room)
		case tea.MouseWheelDown:
			m.explorerMove(rows, cur, wheelRows, room)
		}
	case tea.MouseClickMsg:
		x, y, ww, _ := render.FileRect(m.ui.Width, m.ui.Height, false)
		row := mo.Y - (y + 2) // under the top edge and the path row
		if mo.Button != tea.MouseLeft || mo.X <= x || mo.X >= x+ww-1 || row < 0 || row >= room {
			return nil
		}
		i := render.ListTop(v.Top, max(cur, 0), room, len(rows)) + row
		if i >= len(rows) || rows[i].Path == "" {
			return nil
		}
		if i == cur {
			return m.explorerEnter(rows[i])
		}
		v.Cursor, v.Top = rows[i].Path, i-row
	}
	return nil
}
