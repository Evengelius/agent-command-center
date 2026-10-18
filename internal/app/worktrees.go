package app

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/Evengelius/agent-command-center/internal/input"
	"github.com/Evengelius/agent-command-center/internal/render"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// worktrees is the Worktrees view (plan 16, amendment 10): the worktrees of
// a repository, listed off the update loop, with create, diff, merge and
// remove. Git runs only in a command; a result with an older gen than the
// model's last request is dropped.
type worktrees struct {
	dir      string // the directory asked about
	rows     []input.Worktree
	live     []input.LiveSession // the other dashboard sessions
	sessions string              // why they are unknown; "" known
	problem  string              // the list cannot be read
	reading  bool
	busy     string // the operation running, as the view says it
	sel      int
	note     string // the last operation's outcome
	failed   bool   // the note is a refusal or a failure
	mode     byte   // 0 the list, 'n' a new branch, 'd' a diff, 'm' or 'x' a y/n question
	target   string // the path the question or the diff is about
	branch   input.Editor
	diff     []string
	diffOf   string // what the diff compares, as its title says
	diffTop  int
	selPath  string // select this path once listed (after a create)
}

// Results of the view's git commands.
type (
	wtListMsg struct {
		gen      int
		rows     []input.Worktree
		live     []input.LiveSession
		err      error
		sessions error
	}
	wtOpMsg struct {
		gen  int
		op   byte // 'n', 'm' or 'x'
		path string
		note string
		err  error
	}
	wtDiffMsg struct {
		gen   int
		lines []string
		err   error
	}
	lockMsg struct {
		lock    *input.SessionLock
		project string
		err     error
	}
)

// openWorktrees opens the view: at startup on the project field's directory
// (the current one when the field is empty), in session on the session's
// project, for a mock run on the mock file's directory.
func (m *Model) openWorktrees() []tea.Cmd {
	dir := input.ResolvePath(m.path.Text(), m.env).Path
	if dir == "" {
		dir = m.env.Cwd
	}
	if m.s.Screen == state.ScreenDashboard {
		dir = m.s.Context.Project
		if m.s.Context.Mock {
			dir = filepath.Dir(input.ResolvePath(m.opt.Mock, m.env).Path)
		}
	}
	m.wt = &worktrees{dir: dir}
	return []tea.Cmd{m.listWorktrees()}
}

// listWorktrees reads the list again, and the other sessions with it.
func (m *Model) listWorktrees() tea.Cmd {
	w := m.wt
	m.wtGen++
	w.reading, w.problem = true, ""
	gen, g, dir, reg := m.wtGen, m.git(), w.dir, m.opt.Sessions
	own := ""
	if m.lock != nil {
		own = m.lock.ID
	}
	return func() tea.Msg {
		rows, err := g.Worktrees(dir)
		msg := wtListMsg{gen: gen, rows: rows, err: err}
		if reg == nil {
			msg.sessions = errors.New("N/A: no session registry")
			return msg
		}
		live, serr := reg.Live()
		msg.sessions = serr
		for _, l := range live {
			if l.ID != own {
				msg.live = append(msg.live, l)
			}
		}
		return msg
	}
}

// git is the git the view runs.
func (m *Model) git() input.Git {
	if m.opt.Git == nil {
		return input.GitCLI{}
	}
	return m.opt.Git
}

func (m *Model) worktreesListed(msg wtListMsg) {
	w := m.wt
	if w == nil || msg.gen != m.wtGen {
		return // closed, or an older read
	}
	w.reading, w.rows, w.live, w.sessions = false, msg.rows, msg.live, ""
	if msg.err != nil {
		w.rows, w.problem = nil, msg.err.Error()
	}
	if msg.sessions != nil {
		w.sessions = msg.sessions.Error()
	}
	if w.selPath != "" {
		for i, r := range w.rows {
			if m.samePath(r.Path, w.selPath) {
				w.sel = i
			}
		}
		w.selPath = ""
	}
	w.sel = max(min(w.sel, len(w.rows)-1), 0)
}

func (m *Model) worktreeDone(msg wtOpMsg) []tea.Cmd {
	w := m.wt
	if w == nil || msg.gen != m.wtGen {
		return nil
	}
	w.busy = ""
	switch {
	case msg.err != nil:
		w.note, w.failed = msg.err.Error(), true
		if msg.op == 'n' {
			return nil // the name stays to correct; the list has not changed
		}
	case msg.op == 'n':
		w.note, w.failed, w.mode, w.selPath = "created "+msg.path, false, 0, msg.path
	case msg.op == 'm':
		w.note, w.failed = msg.note, false
	case msg.op == 'x':
		w.note, w.failed = "removed "+msg.path, false
		if msg.note != "" {
			w.note += "; " + msg.note
		}
	}
	return []tea.Cmd{m.listWorktrees()}
}

func (m *Model) worktreeDiffed(msg wtDiffMsg) {
	w := m.wt
	if w == nil || msg.gen != m.wtGen || w.mode != 'd' {
		return
	}
	w.busy = ""
	if msg.err != nil {
		w.mode, w.note, w.failed = 0, "diff: "+msg.err.Error(), true
		return
	}
	w.diff = msg.lines
}

// samePath compares two paths as the system does: without case on Windows.
func (m *Model) samePath(a, b string) bool {
	a, b = filepath.Clean(a), filepath.Clean(b)
	if m.env.Windows {
		return strings.EqualFold(a, b)
	}
	return a == b
}

// here is true for the worktree this session runs in.
func (m *Model) here(r input.Worktree) bool {
	return m.s.Screen == state.ScreenDashboard && !m.s.Context.Mock && m.samePath(r.Path, m.s.Context.Project)
}

// elsewhere is true for a worktree another dashboard session runs in.
func (m *Model) elsewhere(r input.Worktree) bool {
	for _, l := range m.wt.live {
		if m.samePath(r.Path, l.Project) {
			return true
		}
	}
	return false
}

// name is a worktree as the view's messages call it.
func name(r input.Worktree) string {
	if r.Branch == "" {
		return "the detached worktree " + r.Path
	}
	return r.Branch
}

// mergeRefusal is why m cannot merge row r into main; "" when it can
// (plan 16, design, The Worktrees view).
func (m *Model) mergeRefusal(r input.Worktree) string {
	w := m.wt
	main := w.rows[0]
	switch {
	case r.Main:
		return "this is main: m merges another worktree's branch into it"
	case r.Bare || r.Missing:
		return "this worktree's directory is not there"
	case r.Branch == "":
		return "detached HEAD: no branch to merge"
	case r.Problem != "":
		return "its state cannot be read: " + r.Problem
	case m.here(r):
		return "this session runs in " + r.Branch + ": merge it from another directory"
	case main.Branch == "":
		return "main is detached: no branch to merge into"
	case main.Problem != "":
		return "main's state cannot be read: " + main.Problem
	case r.Changes > 0:
		return fmt.Sprintf("%s has %s: commit them first", r.Branch, changes(r.Changes))
	case main.Changes > 0:
		return fmt.Sprintf("%s has %s: commit them first", main.Branch, changes(main.Changes))
	case r.Ahead < 0:
		return "its commits ahead of " + main.Branch + " cannot be counted"
	case r.Ahead == 0:
		return "nothing to merge: " + r.Branch + " has no commit ahead of " + main.Branch
	case m.elsewhere(main):
		return "another dashboard session runs in " + main.Branch
	case m.here(main) && m.s.Turn.Phase != state.Idle && m.s.Turn.Phase != "":
		return "this session's turn is running in " + main.Branch + ": wait for it to end"
	}
	return ""
}

// removeRefusal is why x cannot remove row r; "" when it can.
func (m *Model) removeRefusal(r input.Worktree) string {
	switch {
	case r.Main:
		return "the main worktree is the repository: it is never removed"
	case m.here(r):
		return "this session runs in this worktree"
	case r.Changes > 0:
		return fmt.Sprintf("%s has %s: commit or discard them first", name(r), changes(r.Changes))
	case r.Problem != "" && !r.Missing:
		return "its state cannot be read: " + r.Problem
	case m.elsewhere(r):
		return "another dashboard session runs in this worktree"
	}
	return ""
}

func changes(n int) string {
	if n == 1 {
		return "1 uncommitted change"
	}
	return fmt.Sprintf("%d uncommitted changes", n)
}

// refuse shows why an operation does not run.
func (w *worktrees) refuse(why string) {
	w.note, w.failed = why, true
}

// worktreesKey handles a key while the view is open.
func (m *Model) worktreesKey(k tea.KeyPressMsg) []tea.Cmd {
	w := m.wt
	switch w.mode {
	case 'n':
		return m.newBranchKey(k)
	case 'd':
		m.diffKey(k)
		return nil
	case 'm', 'x':
		return m.wtQuestionKey(k)
	}
	move := func(to int) { w.sel = max(min(to, len(w.rows)-1), 0) }
	page := max(m.ui.Height-12, 1)
	startup := m.s.Screen == state.ScreenStartup
	var row *input.Worktree
	if !w.reading && w.sel < len(w.rows) {
		row = &w.rows[w.sel]
	}
	switch k.String() {
	case "esc":
		if w.busy != "" {
			w.refuse("wait: " + w.busy)
			return nil
		}
		m.wt = nil
	case "up":
		move(w.sel - 1)
	case "down":
		move(w.sel + 1)
	case "pgup":
		move(w.sel - page)
	case "pgdown":
		move(w.sel + page)
	case "home":
		move(0)
	case "end":
		move(len(w.rows) - 1)
	case "enter":
		if !startup || row == nil || w.busy != "" {
			return nil
		}
		if row.Bare || row.Missing {
			w.refuse("this worktree's directory is not there")
			return nil
		}
		m.wt = nil
		m.path.SetText(row.Path)
		m.ui.Field = 1
		cmds := m.do(state.Confirm{Provider: m.ui.Choice, Project: m.path.Text()})
		m.focusMessage()
		return cmds
	case "n":
		switch {
		case !startup:
			w.refuse("a new worktree is created from the startup screen")
		case w.busy != "":
			w.refuse("wait: " + w.busy)
		case w.reading:
			w.refuse("wait: the list is being read")
		case len(w.rows) == 0:
			w.refuse("no repository to create a worktree in")
		default:
			w.mode, w.note, w.failed = 'n', "", false
			w.branch.Clear()
		}
	case "d":
		if row == nil {
			return nil
		}
		if w.busy != "" {
			w.refuse("wait: " + w.busy)
			return nil
		}
		if row.Bare || row.Missing {
			w.refuse("this worktree's directory is not there")
			return nil
		}
		return []tea.Cmd{m.diffWorktree(*row)}
	case "m", "x":
		if row == nil {
			return nil
		}
		if w.busy != "" {
			w.refuse("wait: " + w.busy)
			return nil
		}
		why := m.removeRefusal(*row)
		if k.String() == "m" {
			why = m.mergeRefusal(*row)
		}
		if why != "" {
			w.refuse(why)
			return nil
		}
		w.mode, w.target, w.note, w.failed = k.String()[0], row.Path, "", false
	}
	return nil
}

// newBranchKey edits the new branch's name; Enter creates the worktree.
func (m *Model) newBranchKey(k tea.KeyPressMsg) []tea.Cmd {
	w := m.wt
	switch k.String() {
	case "esc":
		if w.busy == "" {
			w.mode, w.note, w.failed = 0, "", false
		}
		return nil
	case "enter":
		if w.busy != "" {
			return nil
		}
		branch := strings.TrimSpace(w.branch.Text())
		if branch == "" {
			w.refuse("type the branch name: an existing branch, or a new one from main")
			return nil
		}
		main := w.rows[0].Path
		path := input.WorktreePath(main, branch)
		m.wtGen++
		w.busy, w.note, w.failed = "creating "+path+"…", "", false
		gen, g := m.wtGen, m.git()
		return []tea.Cmd{func() tea.Msg {
			return wtOpMsg{gen: gen, op: 'n', path: path, err: g.Add(main, path, branch)}
		}}
	case "tab", "shift+tab", "up", "down":
		return nil
	}
	if w.busy != "" {
		return nil
	}
	w.note, w.failed = "", false
	return m.editKey(&w.branch, k, true)
}

// wtQuestionKey answers the y/n question of a merge or a removal.
func (m *Model) wtQuestionKey(k tea.KeyPressMsg) []tea.Cmd {
	w := m.wt
	switch k.String() {
	case "n", "esc":
		w.mode = 0
		return nil
	case "y":
	default:
		return nil
	}
	op := w.mode
	w.mode = 0
	var row input.Worktree
	found := false
	for _, r := range w.rows {
		if m.samePath(r.Path, w.target) {
			row, found = r, true
		}
	}
	if !found {
		return nil
	}
	// Asked again: a session may have started meanwhile.
	why := m.removeRefusal(row)
	if op == 'm' {
		why = m.mergeRefusal(row)
	}
	if why != "" {
		w.refuse(why)
		return nil
	}
	m.wtGen++
	gen, g, main := m.wtGen, m.git(), w.rows[0]
	if op == 'm' {
		w.busy = "merging " + row.Branch + " into " + main.Branch + "…"
		return []tea.Cmd{func() tea.Msg {
			err := g.Merge(main.Path, row.Branch)
			return wtOpMsg{gen: gen, op: 'm', path: row.Path, note: "merged " + row.Branch + " into " + main.Branch, err: err}
		}}
	}
	w.busy = "removing " + row.Path + "…"
	return []tea.Cmd{func() tea.Msg {
		note, err := g.Remove(main.Path, row.Path, row.Branch)
		return wtOpMsg{gen: gen, op: 'x', path: row.Path, note: note, err: err}
	}}
}

// diffWorktree reads a worktree's diff: against its merge base with main's
// branch, or the main worktree's uncommitted changes.
func (m *Model) diffWorktree(r input.Worktree) tea.Cmd {
	w := m.wt
	main := w.rows[0]
	base := main.Branch
	switch {
	case r.Main:
		base = ""
		w.diffOf = name(r) + ": uncommitted changes"
	case base == "":
		w.diffOf = name(r) + ": uncommitted changes (main is detached)"
	default:
		w.diffOf = name(r) + " against " + base
	}
	w.mode, w.target, w.diff, w.diffTop, w.note, w.failed = 'd', r.Path, nil, 0, "", false
	m.wtGen++
	w.busy = "reading the diff…"
	gen, g, dir := m.wtGen, m.git(), r.Path
	return func() tea.Msg {
		lines, err := g.Diff(dir, base)
		return wtDiffMsg{gen: gen, lines: lines, err: err}
	}
}

// diffKey scrolls the diff; Esc goes back to the list.
func (m *Model) diffKey(k tea.KeyPressMsg) {
	w := m.wt
	room := max(m.ui.Height-7, 1) // render.WorktreesView's diff rows
	last := max(len(w.diff)-room, 0)
	to := w.diffTop
	switch k.String() {
	case "esc":
		w.mode, w.diff, w.busy = 0, nil, ""
		m.wtGen++ // a diff still being read is dropped
		return
	case "up":
		to--
	case "down":
		to++
	case "pgup":
		to -= room
	case "pgdown":
		to += room
	case "home":
		to = 0
	case "end":
		to = last
	}
	w.diffTop = max(min(to, last), 0)
}

// worktreesView is the view as the renderer draws it.
func (m *Model) worktreesView() render.WorktreesView {
	w := m.wt
	if w == nil {
		return render.WorktreesView{}
	}
	v := render.WorktreesView{
		Open: true, Startup: m.s.Screen == state.ScreenStartup, Repo: w.dir,
		Reading: w.reading, Busy: w.busy, Problem: w.problem, Note: w.note, Failed: w.failed,
		Sessions: w.sessions, Mode: w.mode, Sel: w.sel,
		Diff: w.diff, DiffTop: w.diffTop, DiffOf: w.diffOf,
	}
	if len(w.rows) > 0 {
		v.Repo = w.rows[0].Path
	}
	for _, r := range w.rows {
		v.Rows = append(v.Rows, render.WorktreeRow{
			Branch: r.Branch, Path: r.Path, Problem: r.Problem,
			Main: r.Main, Bare: r.Bare, Locked: r.Locked, Missing: r.Missing,
			Changes: r.Changes, Ahead: r.Ahead, Here: m.here(r), Other: m.elsewhere(r),
		})
	}
	switch w.mode {
	case 'n':
		v.Branch = w.branch.Text()
		if len(w.rows) > 0 {
			v.NewPath = input.WorktreePath(w.rows[0].Path, strings.TrimSpace(v.Branch))
		}
	case 'm', 'x':
		for _, r := range w.rows {
			if !m.samePath(r.Path, w.target) {
				continue
			}
			if w.mode == 'm' {
				v.Question = "Merge " + r.Branch + " into " + w.rows[0].Branch + "?"
			} else if r.Branch != "" {
				v.Question = "Remove " + r.Path + "? Its branch " + r.Branch + " is deleted only if merged."
			} else {
				v.Question = "Remove " + r.Path + "?"
			}
		}
	}
	return v
}

// lockSync keeps this session's registration in step with the screen: a
// live dashboard session holds one for its project, nothing else does
// (plan 16, Sessions). The file is written and removed in commands.
func (m *Model) lockSync() []tea.Cmd {
	if m.opt.Sessions == nil {
		return nil
	}
	want := m.s.Screen == state.ScreenDashboard && !m.s.Context.Mock && m.s.Context.Project != ""
	switch {
	case want && m.lock == nil && !m.locking:
		m.locking = true
		reg, project := *m.opt.Sessions, m.s.Context.Project
		return []tea.Cmd{func() tea.Msg {
			l, err := reg.Register(project)
			return lockMsg{lock: l, project: project, err: err}
		}}
	case !want && m.lock != nil:
		l := m.lock
		m.lock = nil
		return []tea.Cmd{func() tea.Msg { _ = l.Release(); return nil }}
	}
	return nil
}

func (m *Model) locked(msg lockMsg) []tea.Cmd {
	m.locking = false
	if msg.err != nil {
		return m.toast("session not registered: " + msg.err.Error())
	}
	m.lock = msg.lock
	if m.s.Screen != state.ScreenDashboard || m.s.Context.Project != msg.project {
		l := m.lock // left meanwhile: lockSync registers the new one
		m.lock = nil
		return []tea.Cmd{func() tea.Msg { _ = l.Release(); return nil }}
	}
	return nil
}

// firstLine is a paste's first line.
func firstLine(text string) string {
	t, _, _ := strings.Cut(strings.ReplaceAll(text, "\r", "\n"), "\n")
	return t
}
