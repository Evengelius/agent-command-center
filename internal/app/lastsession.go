package app

import (
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/Evengelius/agent-command-center/internal/input"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// The last session's resumption from the home screen (plan 69): its record
// in the project store, the note and the key that resume it, and the
// record's writes.

// lastSession is the chosen agent CLI's last session in the project at path,
// the field's resolved (plan 69, design D3): none without a store, or once
// its CLI refused it for good and its record is being cleared, or could not
// be (design D7). A mock run never shows the home screen.
func (m *Model) lastSession(path string) (input.Project, input.Session, bool) {
	if m.projects == nil || path == "" {
		return input.Project{}, input.Session{}, false
	}
	e, s, ok := m.projects.p.LastSession(m.projectList, path, m.ui.Choice)
	if ok && m.s.Startup.ResumeGone && s.ID == m.s.Startup.Resume {
		return input.Project{}, input.Session{}, false
	}
	return e, s, ok
}

// resumeLast is Ctrl+R on the home screen and a click on the last session's
// note (plan 69, design D4): that session resumed in a new process, the
// project validated as START SESSION validates it, the state refusing it
// outside Asking as it refuses START SESSION; nothing without one.
func (m *Model) resumeLast() []tea.Cmd {
	r := input.ResolvePath(m.path.Text(), m.env)
	if r.Problem != "" {
		return nil
	}
	e, sess, ok := m.lastSession(r.Path)
	if !ok {
		return nil
	}
	cmds := m.do(state.Confirm{Provider: m.ui.Choice, Project: m.path.Text(), Resume: sess.ID, ResumeProject: e.Path})
	m.focusMessage()
	return cmds
}

// resumeNotes are the home screen's two notes of the last session for the
// project at path (plan 69, designs D4 and D7): the chosen CLI's, with when
// its last turn ended as the project list writes a time; and the last
// resume's failure, while its CLI and project are the chosen ones.
func (m *Model) resumeNotes(path string) (offer, failure string) {
	if _, sess, ok := m.lastSession(path); ok {
		offer = cliName(m.ui.Choice) + "'s last session here: " + age(m.ui.Now.Sub(sess.Ended), sess.Ended) + " · Ctrl+R continues it"
	}
	st := m.s.Startup
	if st.ResumeFailure != "" && st.Provider == m.ui.Choice && path != "" && state.SamePath(st.ResumeProject, path) {
		failure = "Not resumed: " + st.ResumeFailure
		if st.ResumeGone {
			failure = "Not resumed: " + cliName(st.Provider) + " no longer has its last session here"
		}
	}
	return offer, failure
}

// cliName is an agent CLI's name, as the home screen shows it.
func cliName(id string) string {
	for _, c := range supported {
		if c.id == id {
			return c.name
		}
	}
	return id
}

// recordSession makes id the last session of the session's CLI in its
// project, its turn ended at at (plan 69, design D2).
func (m *Model) recordSession(id string, at time.Time) tea.Cmd {
	path, cli := m.s.Context.Project, m.s.Context.Provider
	return m.projectsCmd(func(p input.Projects, l []input.Project) []input.Project {
		return p.RecordSession(l, path, cli, id, at)
	})
}

// forgetSession drops the project's last session id of the session's CLI,
// which its CLI no longer has (plan 69, design D7).
func (m *Model) forgetSession(id string) tea.Cmd {
	path, cli := m.s.Context.Project, m.s.Context.Provider
	return m.projectsCmd(func(p input.Projects, l []input.Project) []input.Project { return p.ForgetSession(l, path, cli, id) })
}
