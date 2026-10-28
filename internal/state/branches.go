package state

import "strconv"

// The branches popup's actions (plan 59, brief amendment 61): the
// project's branch read again, and a switch the popup made, with the
// dashboard's notice in zone B.

type (
	// BranchRead is the project's branch read again after its HEAD file
	// changed: a switch made elsewhere (design D9); "" outside a work tree.
	BranchRead struct{ Branch string }
	// BranchSwitched is a switch the popup made, From one branch To
	// another, with Carried uncommitted changes git kept (design D6).
	BranchSwitched struct {
		From, To string
		Carried  int
	}
)

func (BranchRead) action()     {}
func (BranchSwitched) action() {}

// branchSwitched records a switch: the header's branch, and the dashboard's
// notice in zone B, which is never sent to the CLI (design D6). While a
// session runs, its agent has seen the branch it started on.
func (s *State) branchSwitched(a BranchSwitched) {
	s.Context.Branch = a.To
	text := "Switched from " + a.From + " to " + a.To
	switch {
	case a.Carried == 1:
		text += " · 1 uncommitted change came along"
	case a.Carried > 1:
		text += " · " + strconv.Itoa(a.Carried) + " uncommitted changes came along"
	}
	if s.Session.Phase == Ready {
		text += " · the agent's conversation still reflects " + a.From
	}
	s.Conversation = append(s.Conversation, Entry{Seq: s.lastSeq(), Role: "dashboard", Text: text, Stage: s.stage()})
}
