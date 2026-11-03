package app

import (
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/Evengelius/agent-command-center/internal/input"
	"github.com/Evengelius/agent-command-center/internal/render"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// The last session's resumption from the home screen (plan 69): its record
// in the project store, the note and the key that resume it, and the
// record's writes.

// lastSession is the chosen agent CLI's last session in the project at path,
// the field's resolved (plan 69, design D3): the newest of its entries in the
// project's history (plan 75, design D1); none without a store or a session,
// and not the one its CLI refused for good while its record is being cleared,
// or could not be (design D7). A mock run never shows the home screen.
func (m *Model) lastSession(path string) (input.Project, input.Session, bool) {
	if m.projects == nil || path == "" {
		return input.Project{}, input.Session{}, false
	}
	e, found := m.projects.p.Find(m.projectList, path)
	if !found {
		return input.Project{}, input.Session{}, false
	}
	// The CLI's newest session in the project's history, but for the one its
	// CLI has just refused for good, whose record is being cleared or could not
	// be: the one before it is offered then (plan 75, design D1).
	skip := ""
	if m.s.Startup.ResumeGone {
		skip = m.s.Startup.Resume
	}
	x, ok := e.Newest(m.ui.Choice, skip)
	return e, input.Session{ID: x.ID, Ended: x.Ended}, ok
}

// resumeLast is Ctrl+R on the home screen and a click on the last session's
// note (plan 69, design D4): that session resumed in a new process, the
// project validated as START SESSION validates it, the state refusing it
// outside Asking as it refuses START SESSION; nothing without one.
func (m *Model) resumeLast() []tea.Cmd {
	if m.stillClosing() {
		return nil // the last session's draft and store still being settled (plan 75)
	}
	r := input.ResolvePath(m.path.Text(), m.env)
	if r.Problem != "" {
		return nil
	}
	e, sess, ok := m.lastSession(r.Path)
	if !ok {
		return nil
	}
	m.olderResume = false
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
			if m.olderResume {
				failure = "Not resumed: " + cliName(st.Provider) + " no longer has that session" // plan 75: not the CLI's last
			}
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

// recordSession puts the session id of the session's CLI first in its
// project's history, its turn ended at at, with the model zone A names and the
// title of the session's first message prompt (plan 69, design D2; plan 75,
// designs D1 and D1a). The title is made here, with render's grapheme
// functions, for the store keeps it only the first time it records the ID.
func (m *Model) recordSession(id string, at time.Time, prompt string, partial bool) tea.Cmd {
	path := m.s.Context.Project
	e := input.Entry{CLI: m.s.Context.Provider, ID: id, Ended: at, Model: render.ModelOf(m.s), Title: render.SessionTitle(prompt, partial)}
	return m.projectsCmd(func(p input.Projects, l []input.Project) []input.Project {
		return p.RecordSession(l, path, e)
	})
}

// forgetSession drops the project's last session id of the session's CLI,
// which its CLI no longer has (plan 69, design D7).
func (m *Model) forgetSession(id string) tea.Cmd {
	path, cli := m.s.Context.Project, m.s.Context.Provider
	return m.projectsCmd(func(p input.Projects, l []input.Project) []input.Project { return p.ForgetSession(l, path, cli, id) })
}
