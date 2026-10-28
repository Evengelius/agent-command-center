package app

import "github.com/Evengelius/agent-command-center/internal/render"

// closePopups closes every popup: the "+" menu, the model and modes
// selectors, the branches popup, the Git Graph's scope popup, the commit
// dropdown and the staging question, as a click outside each of them does.
// At most one popup is open, and opening any layer, a popup, a window, the
// key overlay or the Worktrees view, calls it first: a popup never stays
// open under what opened over it (plan 49, design D7; audit finding 4).
func (m *Model) closePopups() {
	m.ui.Plus, m.ui.PlusHover = false, false
	m.closeModels()
	m.closeModes()
	m.closeBranches()
	m.closeScope()
	m.dropSourcePopups()
}

// closeUnshown closes a popup open that the frame drawn last did not show:
// one a view covers, or one with no room in the frame, so no key or click
// acts on what is not on screen (plan 34, design D9; plan 39, design D9;
// plan 49, design D7). Nothing of it was drawn, so the frame stands.
func (m *Model) closeUnshown(sc *render.Screen) {
	if m.ui.Models.Open && !sc.Selector.Shown {
		m.closeModels()
	}
	if m.ui.Modes.Open && !sc.Modes.Shown {
		m.closeModes()
	}
	if m.ui.Branches.Open && !sc.Branches.Shown {
		m.closeBranches() // no room under the project bar, or a view over it (plan 59)
	}
	if m.ui.Scope.Open && !sc.Scope.Shown {
		m.closeScope() // no room under the scope row, or a view over it (plan 62)
	}
	if m.ui.Plus && !sc.Plus {
		m.ui.Plus, m.ui.PlusHover = false, false
	}
	if m.ui.Source.Menu && !sc.Dropdown || m.ui.Source.Ask > 0 && !sc.Layers.Shown(render.LayerAsk) {
		m.dropSourcePopups()
	}
}

// windowed says the layer on top is a window, whose keys and clicks are
// its own: no popup opens under it.
func windowed(top render.Layer) bool { return top == render.LayerWindow || top == render.LayerDraft }
