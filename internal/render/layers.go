package render

import "github.com/Evengelius/agent-command-center/internal/state"

// Layer is one of the interaction layers over the dashboard: its popups,
// its windows and its full-screen views, in the order they stack (plan 49,
// design D7; audit finding 4). One rule, Layers, says which the frame shows
// and which is on top; Draw, the keys, the mouse, the motion filter and the
// closing of a popup the frame does not show all take it from there. A new
// panel or popup is a Layer here, its place in Layers, and its handlers.
type Layer uint8

const (
	LayerNone      Layer = iota // the dashboard itself, its zones, panels and full views
	LayerSearch                 // the conversation's search bar (plan 66)
	LayerCommit                 // the commit button's dropdown (plan 45)
	LayerAsk                    // the staging question (plan 45)
	LayerBranches               // the branches popup (plan 59)
	LayerScope                  // the Git Graph's scope popup (plan 62)
	LayerModes                  // the modes selector (plan 39)
	LayerModels                 // the model selector (plan 34)
	LayerPlus                   // the "+" menu (plan 30)
	LayerWindow                 // the file viewer or the quick open (plan 17)
	LayerDraft                  // the draft window (amendment 13)
	LayerKeys                   // the key overlay
	LayerWorktrees              // the Worktrees view (plan 16)
)

// Popup says the layer is one of the popups: at most one is open, opening
// one closes the others, and one the frame does not show closes.
func (l Layer) Popup() bool { return l >= LayerCommit && l <= LayerPlus }

// Hover says the layer's hover follows every motion of the pointer: the
// all-motion mouse mode, and motions the filter lets through (plans 30, 34,
// 39).
func (l Layer) Hover() bool { return l == LayerPlus || l == LayerModels || l == LayerModes }

// Stack is what the frame shows of the layers open, and the top one, which
// takes the keys and the clicks.
type Stack struct {
	Top   Layer
	shown uint16
	// Tip is the sidebar's tooltip: shown under no window, taking no input
	// (plan 44, design D1). Card is the Git Graph's card: shown under no
	// popup, window or full view, taking no input (plan 62, design D13).
	Tip, Card bool
}

// Shown says the frame shows layer l.
func (k Stack) Shown(l Layer) bool { return k.shown&(1<<l) != 0 }

// show puts l on top.
func (k *Stack) show(l Layer) { k.shown |= 1 << l; k.Top = l }

// Layers is the layers the frame shows for the state and the UI, from the
// bottom:
//   - the Worktrees view, on either screen, alone;
//   - on the dashboard only, at a size it draws: the key overlay, alone;
//   - else one popup at most, over the dashboard itself (no full view) and
//     under no window: the "+" menu, the model selector, the modes
//     selector, the branches popup, the Git Graph's scope popup while its
//     panel shows, the staging question, the commit dropdown, the first
//     open of these;
//   - the conversation's search bar, over the dashboard, under them all;
//   - the file window, then the draft window over it.
func Layers(s *state.State, ui UI) Stack {
	var k Stack
	if ui.Worktrees.Open {
		k.show(LayerWorktrees)
		return k
	}
	if s.Screen != state.ScreenDashboard || ui.Width < MinWidth || ui.Height < MinHeight {
		return k
	}
	if ui.Keys {
		k.show(LayerKeys)
		return k
	}
	window := WindowOpen(ui) || ui.DraftWin.Open
	if !window && !fullView(s, ui) {
		switch {
		case ui.Plus:
			k.show(LayerPlus)
		case ui.Models.Open:
			k.show(LayerModels)
		case ui.Modes.Open:
			k.show(LayerModes)
		case ui.Branches.Open:
			k.show(LayerBranches)
		case ui.Graph.Open && ui.Scope.Open:
			k.show(LayerScope)
		case ui.Source.Open && ui.Source.Ask > 0:
			k.show(LayerAsk)
		case ui.Source.Open && ui.Source.Menu:
			k.show(LayerCommit)
		}
		k.Card = ui.Graph.Open && ui.Graph.Card != nil && k.Top == LayerNone
		if ui.Search.Open && k.Top == LayerNone {
			// Over the dashboard, under any popup, window or full view, which
			// take the keys first as before (plan 66, design D2).
			k.show(LayerSearch)
		}
	}
	if WindowOpen(ui) {
		k.show(LayerWindow)
	}
	if ui.DraftWin.Open {
		k.show(LayerDraft)
	}
	k.Tip = ui.SideHover != 0 && !window
	return k
}

// FullView is fullView, for the keys a full view keeps its own (plan 66,
// design D2: Ctrl+F opens no search bar over one).
func FullView(s *state.State, ui UI) bool { return fullView(s, ui) }

// fullView says a full-screen view replaces the dashboard: an approval's or
// an entry's, an agent's inspection, the telemetry's details.
func fullView(s *state.State, ui UI) bool {
	return ui.View != "" && s.Approvals[ui.View] != nil || ui.ViewEntry > 0 && ui.ViewEntry <= len(s.Conversation) ||
		ui.Inspect != "" && s.Agents[ui.Inspect] != nil || ui.Details
}

// only is ui with the popups the stack does not show closed, for the frame
// it draws: nothing of them is drawn, neither their hints nor their images.
func (k Stack) only(ui UI) UI {
	if ui.Plus && !k.Shown(LayerPlus) {
		ui.Plus, ui.PlusHover = false, false
	}
	if ui.Models.Open && !k.Shown(LayerModels) {
		ui.Models = ModelsView{}
	}
	if ui.Modes.Open && !k.Shown(LayerModes) {
		ui.Modes = ModesView{}
	}
	if ui.Branches.Open && !k.Shown(LayerBranches) {
		ui.Branches = BranchesView{}
	}
	if ui.Scope.Open && !k.Shown(LayerScope) {
		ui.Scope = ScopeView{}
	}
	if ui.Source.Ask > 0 && !k.Shown(LayerAsk) {
		ui.Source.Ask = 0
	}
	if ui.Source.Menu && !k.Shown(LayerCommit) {
		ui.Source.Menu, ui.Source.MenuHot = false, false
	}
	return ui
}
