package app

import (
	"os"
	"path/filepath"

	tea "charm.land/bubbletea/v2"

	"github.com/Evengelius/agent-command-center/internal/input"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// The project folder from the system's dialog (plan 52, brief amendment 53;
// ui-design.md, Startup screen): the folder button and Ctrl+O open it at
// the field's directory, and the folder chosen goes into the field.

// chosenFolderMsg is the folder dialog's answer: the folder chosen, ""
// when it was cancelled.
type chosenFolderMsg struct {
	path string
	err  error
}

// openFolder opens the folder dialog in a command (design D1): one at a
// time, none while a start is validated.
func (m *Model) openFolder() []tea.Cmd {
	if m.folderOpen || m.s.Startup.Phase == state.Validating {
		return nil
	}
	m.ui.FolderProblem = ""
	if m.opt.PickFolder == nil {
		m.ui.FolderProblem = "No folder dialog in this run"
		return nil
	}
	m.folderOpen = true
	pick, dir, fallback := m.opt.PickFolder, input.ResolvePath(m.path.Text(), m.env).Path, m.startProject()
	return []tea.Cmd{func() tea.Msg {
		path, err := pick(folderStart(dir, fallback))
		return chosenFolderMsg{path, err}
	}}
}

// folderStart is where the dialog opens (design D2): dir when it is a
// directory, else its nearest existing parent, else fallback.
func folderStart(dir, fallback string) string {
	for d := dir; d != ""; {
		if fi, err := os.Stat(d); err == nil && fi.IsDir() {
			return d
		}
		up := filepath.Dir(d)
		if up == d {
			break
		}
		d = up
	}
	return fallback
}

// folderPicked puts the folder chosen in the field, which takes the focus
// (design D3); a cancelled dialog changes nothing, a failed one says why
// until the dialog opens again or Ctrl+N.
func (m *Model) folderPicked(msg chosenFolderMsg) {
	m.folderOpen = false
	switch {
	case msg.err != nil:
		m.ui.FolderProblem = "Folder dialog: " + msg.err.Error()
	case msg.path != "" && m.s.Screen == state.ScreenStartup && m.s.Startup.Phase != state.Validating:
		m.path.SetText(msg.path)
		m.ui.Field = 1
	}
}
