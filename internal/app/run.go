package app

import (
	"errors"
	"fmt"

	tea "charm.land/bubbletea/v2"
)

// Exit codes (reliability.md, Separated parts; Resumption).
const (
	ExitOK    = 0
	ExitError = 1
	ExitPanic = 2
)

// Run runs the program to its end, then writes the draft, closes the
// session and says how to resume it. Bubble Tea recovers a panic in Update,
// in View or in a command, restores the terminal and returns
// tea.ErrProgramPanic; the same steps follow, and the code is ExitPanic
// (reliability.md, Separated parts, the dashboard's own crash). out is
// what to print once the terminal is back.
func Run(m *Model, opts ...tea.ProgramOption) (code int, out []string) {
	_, err := tea.NewProgram(m, opts...).Run()
	if derr := m.flushDraft(); derr != nil {
		out = append(out, "the draft could not be saved: "+derr.Error())
	}
	m.Shutdown()
	switch {
	case errors.Is(err, tea.ErrProgramPanic):
		code = ExitPanic
		out = append(out, "agent-command-center stopped on an internal error: "+err.Error())
	case err != nil:
		code = ExitError
		out = append(out, "agent-command-center: "+err.Error())
	}
	if l := m.resumeLine(); l != "" {
		out = append(out, l)
	}
	return code, out
}

// resumeLine names the CLI session and the CLI's own command that resumes
// it outside the dashboard (reliability.md, Resumption); "" without one.
func (m *Model) resumeLine() string {
	id := m.s.Session.ID
	if id == "" || m.s.Context.Mock {
		return ""
	}
	cmd := ResumeCommand(m.s.Context.Provider, id)
	if cmd == "" {
		return "session " + id
	}
	return fmt.Sprintf("session %s · resume it in %s with: %s", id, m.s.Context.Project, cmd)
}

// ResumeCommand is the CLI's command that resumes session id
// (reliability.md, Resumption); "" for a provider without one.
func ResumeCommand(provider, id string) string {
	switch provider {
	case "claude-code":
		return "claude --resume " + id
	case "codex":
		return "codex resume " + id
	}
	return ""
}
