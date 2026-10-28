package state

import "testing"

// TestBranchSwitched is plan 59, design D6, in the reducer: a switch sets
// the branch and adds the dashboard's notice, its changes in the singular
// or the plural, the agent's conversation named while a session runs;
// BranchRead sets the branch alone.
func TestBranchSwitched(t *testing.T) {
	for _, c := range []struct {
		carried int
		ready   bool
		want    string
	}{
		{0, true, "Switched from main to feat · the agent's conversation still reflects main"},
		{1, true, "Switched from main to feat · 1 uncommitted change came along · the agent's conversation still reflects main"},
		{3, false, "Switched from main to feat · 3 uncommitted changes came along"},
	} {
		s := New()
		s.Context.Branch = "main"
		if c.ready {
			s.Session.Phase = Ready
		}
		if cmds := s.Do(BranchSwitched{From: "main", To: "feat", Carried: c.carried}); cmds != nil {
			t.Errorf("commands %v", cmds)
		}
		n := len(s.Conversation)
		if s.Context.Branch != "feat" || n != 1 || s.Conversation[0].Role != "dashboard" || s.Conversation[0].Text != c.want {
			t.Errorf("carried %d, ready %v: branch %q, %d entries %+v", c.carried, c.ready, s.Context.Branch, n, s.Conversation)
		}
	}
	s := New()
	s.Do(BranchRead{Branch: "fix/login"})
	if s.Context.Branch != "fix/login" || len(s.Conversation) != 0 {
		t.Errorf("BranchRead: branch %q, %d entries", s.Context.Branch, len(s.Conversation))
	}
}
