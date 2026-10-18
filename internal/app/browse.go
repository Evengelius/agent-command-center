package app

import (
	"errors"
	"io/fs"
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/Evengelius/agent-command-center/internal/input"
	"github.com/Evengelius/agent-command-center/internal/render"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// browser is the project browser over the startup screen (plan 15): it
// lists directories, off the update loop, until one is chosen.
type browser struct {
	dir     string // the directory shown; "" the drives
	from    string // the child just left, selected once listed
	entries []input.DirEntry
	drives  []string
	problem string // the directory cannot be listed
	reading bool
	gen     int // the last listing asked for
	filter  []rune
	sel     int // in rows()
	hidden  bool
	windows bool // a root's parent is the drives
}

// browseMsg is a listing; an older gen than the browser's is dropped.
type browseMsg struct {
	gen     int
	dir     string
	entries []input.DirEntry
	drives  []string
	err     error
}

// browseRow is one row: choose the directory shown, go up, or a directory
// or a drive to open.
type browseRow struct {
	kind  byte // 'u' use, 'p' parent, 'd' directory, 'r' drive
	name  string
	entry input.DirEntry
}

// openBrowser opens at the field's directory, else its nearest existing
// parent, else the home directory (criterion 3.2).
func (m *Model) openBrowser() tea.Cmd {
	start := input.ResolvePath(m.path.Text(), m.env).Path
	if start == "" {
		start = m.env.Cwd
	}
	m.browse = &browser{windows: m.env.Windows}
	return m.browseTo(start, "", true)
}

// browseTo lists dir ("" the drives); nearest walks up past directories that
// do not exist, then falls back to home.
func (m *Model) browseTo(dir, from string, nearest bool) tea.Cmd {
	b := m.browse
	b.gen++
	b.dir, b.from, b.entries, b.drives, b.problem, b.reading = dir, from, nil, nil, "", true
	b.filter, b.sel = nil, 0
	gen, l, home := b.gen, m.opt.Lister, m.env.Home
	return func() tea.Msg {
		if dir == "" {
			return browseMsg{gen: gen, drives: l.Drives()}
		}
		for {
			es, err := l.List(dir)
			if err == nil || !nearest || !errors.Is(err, fs.ErrNotExist) {
				return browseMsg{gen: gen, dir: dir, entries: es, err: err}
			}
			up := filepath.Dir(dir)
			if up == dir {
				dir, nearest = home, false
				continue
			}
			dir = up
		}
	}
}

func (m *Model) browseDone(msg browseMsg) {
	b := m.browse
	if b == nil || msg.gen != b.gen {
		return
	}
	b.dir, b.entries, b.drives, b.reading = msg.dir, msg.entries, msg.drives, false
	if msg.err != nil {
		b.problem = listProblem(msg.dir, msg.err)
	}
	if b.from != "" {
		for i, r := range b.rows() {
			if r.kind != 'u' && r.kind != 'p' && strings.EqualFold(r.name, b.from) {
				b.sel = i
			}
		}
	}
}

// listProblem is why a directory cannot be listed, as validation words it
// (ui-design.md, Validation and messages).
func listProblem(dir string, err error) string {
	switch {
	case dir == "":
		return "No directory to open: the home directory is unknown"
	case errors.Is(err, fs.ErrPermission):
		return "Not readable (permission denied): " + dir
	case errors.Is(err, fs.ErrNotExist):
		return "Does not exist: " + dir
	}
	return "Not readable: " + dir + ": " + cause(err)
}

// rows is what the browser lists: the directory shown, its parent, then its
// subdirectories that match the filter; the drives instead at the top.
func (b *browser) rows() []browseRow {
	if b.reading {
		return nil // shown once listed
	}
	var out []browseRow
	f := strings.ToLower(string(b.filter))
	if b.dir == "" {
		for _, d := range b.drives {
			if strings.Contains(strings.ToLower(d), f) {
				out = append(out, browseRow{kind: 'r', name: d})
			}
		}
		return out
	}
	if b.problem == "" {
		out = append(out, browseRow{kind: 'u', name: b.dir})
	}
	if filepath.Dir(b.dir) != b.dir || b.windows {
		out = append(out, browseRow{kind: 'p', name: ".."})
	}
	for _, e := range b.entries {
		if (e.Hidden && !b.hidden) || !strings.Contains(strings.ToLower(e.Name), f) {
			continue
		}
		out = append(out, browseRow{kind: 'd', name: e.Name, entry: e})
	}
	return out
}

// path is the directory a row leads to; "" for the drives.
func (b *browser) path(r browseRow) string {
	switch r.kind {
	case 'u':
		return b.dir
	case 'p':
		up := filepath.Dir(b.dir)
		if up == b.dir {
			return ""
		}
		return up
	case 'r':
		return r.name
	}
	return filepath.Join(b.dir, r.name)
}

// browseKey handles a key while the browser is open (criterion 3.3).
func (m *Model) browseKey(k tea.KeyPressMsg) []tea.Cmd {
	b := m.browse
	rows := b.rows()
	move := func(to int) { b.sel = max(min(to, len(rows)-1), 0) }
	page := max(m.ui.Height-12, 1)
	switch k.String() {
	case "esc":
		if len(b.filter) > 0 {
			b.filter, b.sel = nil, 0
			return nil
		}
		m.browse = nil
	case "up":
		move(b.sel - 1)
	case "down":
		move(b.sel + 1)
	case "pgup":
		move(b.sel - page)
	case "pgdown":
		move(b.sel + page)
	case "home":
		move(0)
	case "end":
		move(len(rows) - 1)
	case "left":
		if len(b.filter) == 0 {
			return m.browseUp()
		}
	case "backspace":
		if len(b.filter) == 0 {
			return m.browseUp()
		}
		b.setFilter(b.filter[:len(b.filter)-1])
	case "ctrl+h":
		b.hidden = !b.hidden
		b.sel = 0
	case "ctrl+f":
		if b.sel < len(rows) && rows[b.sel].kind != 'p' {
			if p := b.path(rows[b.sel]); p != "" {
				return []tea.Cmd{m.starPath(p)}
			}
		}
	case "enter", "right":
		if b.sel >= len(rows) {
			return nil
		}
		r := rows[b.sel]
		if r.kind == 'u' {
			if k.String() == "right" {
				return nil
			}
			m.browse = nil
			m.path.SetText(r.name)
			m.ui.Field = 1
			cmds := m.do(state.Confirm{Provider: m.ui.Choice, Project: m.path.Text()})
			m.focusMessage()
			return cmds
		}
		if r.kind == 'p' {
			return m.browseUp()
		}
		return []tea.Cmd{m.browseTo(b.path(r), "", false)}
	default:
		if t := k.Text; t != "" && k.Mod&^tea.ModShift == 0 {
			b.setFilter(append(b.filter, []rune(t)...))
		}
	}
	return nil
}

// setFilter changes the filter and selects its first matching directory.
func (b *browser) setFilter(f []rune) {
	b.filter, b.sel = f, 0
	for i, r := range b.rows() {
		if r.kind == 'd' || r.kind == 'r' {
			b.sel = i
			return
		}
	}
}

// paste adds a paste's first line to the filter.
func (b *browser) paste(text string) {
	t, _, _ := strings.Cut(strings.ReplaceAll(text, "\r", "\n"), "\n")
	b.setFilter(append(b.filter, []rune(t)...))
}

// browseUp lists the parent, the child left selected; at a root, the drives
// on Windows.
func (m *Model) browseUp() []tea.Cmd {
	b := m.browse
	if b.dir == "" {
		return nil
	}
	up := filepath.Dir(b.dir)
	if up == b.dir {
		if !b.windows {
			return nil
		}
		return []tea.Cmd{m.browseTo("", b.dir, false)}
	}
	return []tea.Cmd{m.browseTo(up, filepath.Base(b.dir), false)}
}

// starPath stars or unstars a directory in the store; without a store it
// does nothing.
func (m *Model) starPath(path string) tea.Cmd {
	if m.projects == nil {
		return nil
	}
	on := !m.favourite(path)
	m.projectList = m.projects.p.Star(m.projectList, path, on) // shown at once
	return m.projectsCmd(func(p input.Projects, l []input.Project) []input.Project { return p.Star(l, path, on) })
}

func (m *Model) favourite(path string) bool {
	for _, e := range m.projectList {
		if e.Favourite && m.projects.p.Same(e.Path, path) {
			return true
		}
	}
	return false
}

// browserView is the browser as the renderer draws it.
func (m *Model) browserView() render.BrowserView {
	b := m.browse
	if b == nil {
		return render.BrowserView{}
	}
	v := render.BrowserView{
		Open: true, Dir: b.dir, Filter: string(b.filter), Reading: b.reading,
		Problem: b.problem, ShowHidden: b.hidden, Store: m.projects != nil,
	}
	rows := b.rows()
	b.sel = max(min(b.sel, len(rows)-1), 0)
	v.Sel = b.sel
	for _, r := range rows {
		br := render.BrowserRow{Kind: r.kind, Name: r.name, Git: r.entry.Git, Link: r.entry.Link}
		if r.kind != 'p' && m.projects != nil {
			br.Favourite = m.favourite(b.path(r))
		}
		if r.kind == 'd' {
			v.Shown++
		}
		v.Rows = append(v.Rows, br)
	}
	for _, e := range b.entries {
		if e.Hidden && !b.hidden {
			v.Hidden++
		} else {
			v.Total++
		}
	}
	return v
}
