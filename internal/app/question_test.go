package app

import (
	"strings"
	"testing"
	"testing/synctest"

	tea "charm.land/bubbletea/v2"

	"github.com/Evengelius/agent-command-center/internal/provider/claude"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// TestQuestionDeny is plan 5, item 10, criteria 1 to 4: a Claude Code
// question is denied with QuestionDenyMessage, never allowed; every other
// deny keeps DenyMessage. No probe recorded an AskUserQuestion request
// (CC-10 q3), so the requests are synthetic, in can_use_tool's shape (P2).
func TestQuestionDeny(t *testing.T) {
	const later = "answer in your next message"
	if claude.QuestionDenyMessage != "The user saw these questions in Agent Command Center, which cannot answer them here, and will answer them in their next message. Wait for that message; do not ask them again." {
		t.Errorf("QuestionDenyMessage is %q", claude.QuestionDenyMessage) // criterion 1, word for word
	}
	const question = `{"type":"control_request","request_id":"q1","request":{"subtype":"can_use_tool","tool_name":"AskUserQuestion","input":{"questions":[{"question":"Which colour?","header":"Colour","options":[{"label":"Red","description":"warm"},{"label":"Blue","description":"cold"}],"multiSelect":false}]},"tool_use_id":"toolu_q1"}}`
	start := func(t *testing.T, reqs ...string) (*harness, *wireClaude) {
		w := &wireClaude{}
		h := wireRig(t, w, Options{})
		h.send(tea.PasteMsg{Content: "go"})
		h.send(key(tea.KeyEnter, 0))
		w.last().emit(append([]string{`{"type":"system","subtype":"init","cwd":` + quote(h.m.State().Context.Project) + `,"session_id":"s-q","model":"claude-haiku-5-5","permissionMode":"default"}`}, reqs...)...)
		h.settle()
		return h, w
	}
	t.Run("question", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			h, w := start(t, question)
			if f := frameText(h); !strings.Contains(f, "? QUESTION") || !strings.Contains(f, "Which colour?") {
				t.Fatalf("no question card:\n%s", f)
			}
			// a sends nothing before every question has a choice, from the card or the
			// full view (criterion 4; plan 14, item 1: a toast says why).
			h.send(key(tea.KeyF2, 0))
			h.send(typed('a'))
			if f := frameText(h); !strings.Contains(f, "choose an option for each question first") {
				t.Fatalf("no toast after a:\n%s", f)
			}
			h.send(key(tea.KeyEnter, 0))
			if f := frameText(h); !strings.Contains(f, "back (Esc)") {
				t.Fatalf("no full view:\n%s", f)
			}
			h.send(typed('a'))
			h.settle()
			if got := answersFor(w, "q1"); len(got) != 0 {
				t.Fatalf("a answered %v", got)
			}
			if ap := h.m.State().Approvals["q1"]; ap.Status != state.Pending {
				t.Fatalf("question %q after a, want pending", ap.Status)
			}
			h.send(key(tea.KeyF2, 0))
			h.send(typed('d'))
			h.send(typed('d'))
			h.settle()
			got := answersFor(w, "q1")
			if len(got) != 1 || got[0]["behavior"] != "deny" || got[0]["message"] != claude.QuestionDenyMessage {
				t.Fatalf("answers %v; want one deny with the question's message", got)
			}
			if f := frameText(h); !strings.Contains(f, "denied by you") || !strings.Contains(f, later) {
				t.Errorf("denied question:\n%s", f)
			}
		})
	})
	t.Run("approval", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			h, w := start(t, `{"type":"control_request","request_id":"r1","request":{"subtype":"can_use_tool","tool_name":"Write","input":{"file_path":"x.txt","content":"x"},"tool_use_id":"toolu_r1"}}`)
			h.send(key(tea.KeyF2, 0))
			h.send(typed('d'))
			h.settle()
			got := answersFor(w, "r1")
			if len(got) != 1 || got[0]["behavior"] != "deny" || got[0]["message"] != claude.DenyMessage {
				t.Fatalf("answers %v; want one deny with DenyMessage", got)
			}
			if f := frameText(h); !strings.Contains(f, "denied by you") || strings.Contains(f, later) {
				t.Errorf("denied approval:\n%s", f)
			}
		})
	})
	t.Run("cancelled question", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			// c needs a running turn: the tool call comes first, as in a live turn.
			h, w := start(t, `{"type":"assistant","message":{"id":"m1","type":"message","role":"assistant","model":"claude-haiku-5-5","content":[{"type":"tool_use","id":"toolu_q1","name":"AskUserQuestion","input":{}}]},"parent_tool_use_id":null,"session_id":"s-q"}`, question)
			h.send(key(tea.KeyF2, 0))
			h.send(typed('c'))
			h.settle()
			got := answersFor(w, "q1")
			if len(got) != 1 || got[0]["behavior"] != "deny" || got[0]["message"] != claude.DenyMessage {
				t.Fatalf("answers %v; want one deny with DenyMessage", got)
			}
			if f := frameText(h); !strings.Contains(f, "denied by you") || strings.Contains(f, later) {
				t.Errorf("cancelled question:\n%s", f)
			}
		})
	})
	t.Run("unreadable question", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			h, w := start(t, strings.Replace(question, `"tool_use_id":"toolu_q1"`, `"tool_use_id":"toolu_q1","agent_id":7`, 1))
			got := answersFor(w, "q1")
			if len(got) != 1 || got[0]["behavior"] != "deny" || got[0]["message"] != claude.DenyMessage {
				t.Fatalf("answers %v; want one deny with DenyMessage", got)
			}
			if f := frameText(h); !strings.Contains(f, "denied automatically") || strings.Contains(f, later) {
				t.Errorf("unreadable question:\n%s", f)
			}
		})
	})
}
