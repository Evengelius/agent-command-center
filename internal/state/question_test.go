package state_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/Evengelius/agent-command-center/internal/event"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// Plan 14, item 1: a question is allowed only with an answer to each of its
// questions; any other approval ignores answers.
func TestQuestionAnswerRules(t *testing.T) {
	const input = `{"questions":[{"question":"Which colour?","header":"Colour","options":[{"label":"Red","description":""},{"label":"Blue","description":""}],"multiSelect":false},{"question":"Which days?","header":"Days","options":[{"label":"Mon","description":""},{"label":"Tue","description":""}],"multiSelect":true}]}`
	setup := func() *state.State {
		s := start(t, event.Started{SessionID: "s", ApprovalMode: event.ModeManual})
		apply(s, 10, event.Question, event.Approval{ID: "q1", Tool: "AskUserQuestion", Input: json.RawMessage(input)})
		apply(s, 11, event.ApprovalRequest, event.Approval{ID: "w1", Tool: "Write", Input: json.RawMessage(`{"file_path":"x"}`)})
		return s
	}
	last := func(s *state.State) string {
		if len(s.Notes) == 0 {
			return ""
		}
		return s.Notes[len(s.Notes)-1]
	}
	for _, c := range []struct {
		name    string
		answers map[string]string
	}{
		{"none", nil},
		{"one of two", map[string]string{"Which colour?": "Red"}},
		{"an empty one", map[string]string{"Which colour?": "Red", "Which days?": ""}},
		{"another question", map[string]string{"Which colour?": "Red", "Other?": "x"}},
	} {
		s := setup()
		if cmds := s.Do(state.Answer{ID: "q1", Decision: "allow", Answers: c.answers}); len(cmds) != 0 {
			t.Errorf("%s: sent %v", c.name, cmds)
		}
		if ap := s.Approvals["q1"]; ap.Status != state.Pending || ap.Answers != nil {
			t.Errorf("%s: approval %+v, want pending", c.name, ap)
		}
		if n := last(s); !strings.Contains(n, "answer refused: ") {
			t.Errorf("%s: note %q", c.name, n)
		}
	}
	s := setup()
	answers := map[string]string{"Which colour?": "Red", "Which days?": "Mon, Tue"}
	cmds := s.Do(state.Answer{ID: "q1", Decision: "allow", Answers: answers})
	if len(cmds) != 1 || cmds[0].Kind != state.CmdAnswer || cmds[0].ID != "q1" || cmds[0].Decision != "allow" || cmds[0].Answers["Which days?"] != "Mon, Tue" || len(cmds[0].Answers) != 2 {
		t.Fatalf("commands %+v", cmds)
	}
	if ap := s.Approvals["q1"]; ap.Status != state.Answering || ap.Answers["Which colour?"] != "Red" {
		t.Errorf("approval %+v, want answering with the answers", ap)
	}
	apply(s, 12, event.ApprovalResolved, event.Approval{ID: "q1", Decision: "allow"})
	if ap := s.Approvals["q1"]; ap.Status != state.Allowed || ap.Answer != "allow" {
		t.Errorf("approval %+v, want allowed", ap)
	}
	// A deny of a question needs no answers; a non-question ignores them.
	s = setup()
	if cmds := s.Do(state.Answer{ID: "q1", Decision: "deny"}); len(cmds) != 1 || cmds[0].Answers != nil {
		t.Errorf("deny %+v", cmds)
	}
	cmds = s.Do(state.Answer{ID: "w1", Decision: "allow", Answers: answers})
	if len(cmds) != 1 || cmds[0].Answers != nil || s.Approvals["w1"].Answers != nil {
		t.Errorf("allow of a non-question %+v", cmds)
	}
}
