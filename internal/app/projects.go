package app

import (
	"fmt"
	"sync"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/Evengelius/agent-command-center/internal/input"
	"github.com/Evengelius/agent-command-center/internal/render"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// projectsMsg is the project list after a read or a change of the store
// (plan 15); gen orders them, an older one never replaces a newer list.
type projectsMsg struct {
	gen  int
	list []input.Project
	err  error
	read bool // the first read, not a change
}

// projectStore runs the reads and changes of projects.json outside Update,
// one at a time, so two changes never write over each other.
type projectStore struct {
	p  input.Projects
	mu sync.Mutex
}

// projectsCmd reads the list (change nil) or changes it in the store.
func (m *Model) projectsCmd(change func(input.Projects, []input.Project) []input.Project) tea.Cmd {
	if m.projects == nil {
		return nil
	}
	m.projectsGen++
	gen, ps := m.projectsGen, m.projects
	return func() tea.Msg {
		ps.mu.Lock()
		defer ps.mu.Unlock()
		if change == nil {
			list, err := ps.p.Load()
			return projectsMsg{gen: gen, list: list, err: err, read: true}
		}
		list, err := ps.p.Update(func(l []input.Project) []input.Project { return change(ps.p, l) })
		return projectsMsg{gen: gen, list: list, err: err}
	}
}

func (m *Model) projectsDone(msg projectsMsg) []tea.Cmd {
	if msg.gen < m.projectsSeen {
		return nil
	}
	m.projectsSeen = msg.gen
	if msg.err == nil {
		m.projectList, m.ui.ProjectsProblem = msg.list, ""
		return nil
	}
	// Never an empty list in its place (AGENTS.md, Keep failures visible).
	if msg.read {
		m.ui.ProjectsProblem = "Recent projects unavailable: " + msg.err.Error()
		return nil
	}
	m.ui.ProjectsProblem = "Recent projects not saved: " + msg.err.Error()
	if m.s.Screen == state.ScreenDashboard {
		return m.toast("recent projects not saved: " + msg.err.Error())
	}
	return nil
}

// recordProject puts the project a session starts in first in the list.
func (m *Model) recordProject(path string) tea.Cmd {
	at := m.opt.Now()
	return m.projectsCmd(func(p input.Projects, l []input.Project) []input.Project { return p.Record(l, path, at) })
}

// forgetProject removes the selected project from the list.
func (m *Model) forgetProject() tea.Cmd {
	if m.ui.ProjectSel >= len(m.projectList) {
		return nil
	}
	path := m.projectList[m.ui.ProjectSel].Path
	m.projectList = m.projects.p.Forget(m.projectList, path) // shown at once
	return m.projectsCmd(func(p input.Projects, l []input.Project) []input.Project { return p.Forget(l, path) })
}

// starProject stars or unstars the selected project; the selection follows
// it to its new place.
func (m *Model) starProject() tea.Cmd {
	if m.ui.ProjectSel >= len(m.projectList) {
		return nil
	}
	e := m.projectList[m.ui.ProjectSel]
	path, on := e.Path, !e.Favourite
	m.projectList = m.projects.p.Star(m.projectList, path, on) // shown at once
	for i, e := range m.projectList {
		if m.projects.p.Same(e.Path, path) {
			m.ui.ProjectSel = i
		}
	}
	return m.projectsCmd(func(p input.Projects, l []input.Project) []input.Project { return p.Star(l, path, on) })
}

// projectRows is the list as the startup screen shows it.
func (m *Model) projectRows() []render.ProjectRow {
	rows := make([]render.ProjectRow, len(m.projectList))
	for i, e := range m.projectList {
		a := "never" // starred in the browser, no session started yet
		if !e.Used.IsZero() {
			a = age(m.ui.Now.Sub(e.Used), e.Used)
		}
		rows[i] = render.ProjectRow{Path: e.Path, Age: a, Favourite: e.Favourite}
	}
	return rows
}

// age is how long ago a project's last session started, as the home screen
// writes it (plan 51, design D7): "now", "8m ago", "2h ago", "1d ago".
func age(d time.Duration, at time.Time) string {
	switch {
	case d < time.Minute:
		return "now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d/time.Minute))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d/time.Hour))
	case d < 30*24*time.Hour:
		return fmt.Sprintf("%dd ago", int(d/(24*time.Hour)))
	}
	return at.Local().Format("2006-01-02")
}

// startupFields is the Tab order: agent, project, the list when it has a
// row, Start.
func (m *Model) startupFields() []int {
	if len(m.projectList) > 0 {
		return []int{0, 1, 3, 2}
	}
	return []int{0, 1, 2}
}

func (m *Model) nextField(step int) {
	fs := m.startupFields()
	i := 0
	for j, f := range fs {
		if f == m.ui.Field {
			i = j
		}
	}
	m.ui.Field = fs[(i+step+len(fs))%len(fs)]
}
