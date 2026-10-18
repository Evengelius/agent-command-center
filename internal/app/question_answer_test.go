package app

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"testing/synctest"

	tea "charm.land/bubbletea/v2"

	"github.com/Evengelius/agent-command-center/internal/state"
)

// Plan 14, item 1: a question is answered on its card and in its full view,
// with the keys, and the answers reach Claude Code as P14a proved.

// p14aQuestion is the request of probe P14a, as recorded.
func p14aQuestion(t *testing.T) (id, raw string) {
	t.Helper()
	for _, r := range probe(t, "p14a-claude-question-answer.jsonl") {
		if r.Dir == "out" && bytes.Contains(r.Msg, []byte(`"tool_name":"AskUserQuestion"`)) {
			var m struct {
				RequestID string `json:"request_id"`
			}
			if err := json.Unmarshal(r.Msg, &m); err != nil {
				t.Fatal(err)
			}
			return m.RequestID, string(r.Msg)
		}
	}
	t.Fatal("no AskUserQuestion request in P14a")
	return "", ""
}

func startQuestion(t *testing.T) (h *harness, w *wireClaude, id string) {
	id, raw := p14aQuestion(t)
	w = &wireClaude{}
	h = wireRig(t, w, Options{})
	h.send(tea.WindowSizeMsg{Width: 160, Height: 50})
	h.send(tea.PasteMsg{Content: "go"})
	h.send(key(tea.KeyEnter, 0))
	w.last().emit(`{"type":"system","subtype":"init","cwd":`+quote(h.m.State().Context.Project)+`,"session_id":"s-q","model":"claude-haiku-5-5","permissionMode":"default"}`, raw)
	h.settle()
	h.send(key(tea.KeyF2, 0))
	return h, w, id
}

// updatedInput is the allow written for id, or nil.
func updatedInput(t *testing.T, w *wireClaude, id string) map[string]json.RawMessage {
	t.Helper()
	got := answersFor(w, id)
	if len(got) != 1 || got[0]["behavior"] != "allow" {
		t.Fatalf("answers %v, want one allow", got)
	}
	b, _ := json.Marshal(got[0]["updatedInput"])
	var in map[string]json.RawMessage
	if err := json.Unmarshal(b, &in); err != nil {
		t.Fatal(err)
	}
	return in
}

func TestQuestionAnswerKeys(t *testing.T) {
	for _, inView := range []bool{false, true} {
		name := "card"
		if inView {
			name = "view"
		}
		t.Run(name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				h, w, id := startQuestion(t)
				f := frameText(h)
				for _, want := range []string{"? QUESTION", "▸ Colour · Which colour?", "1 ○ Rouge café ✓ — Select this colour.", "2 ○ Blue", "Days · Which days?", "1 [ ] Monday", "3 [ ] Friday", "1-4 choose · ← → question · a send the answers"} {
					if !strings.Contains(f, want) && !inView {
						t.Fatalf("no %q:\n%s", want, f)
					}
				}
				if inView {
					h.send(key(tea.KeyEnter, 0))
					if f := frameText(h); !strings.Contains(f, "1-4 choose · ← → question · a send the answers · d deny") || !strings.Contains(f, "1 [ ] Monday") {
						t.Fatalf("no question view:\n%s", f)
					}
				}
				h.send(typed('3')) // no third colour: nothing
				if h.m.ui.Pick.At != 0 {
					t.Errorf("a digit beyond the options moved the selection to %d", h.m.ui.Pick.At)
				}
				h.send(typed('a')) // refused: no choice yet
				if f := frameText(h); !strings.Contains(f, "choose an option for each question first") {
					t.Fatalf("no toast:\n%s", f)
				}
				h.settle()
				if got := answersFor(w, id); len(got) != 0 {
					t.Fatalf("a answered %v", got)
				}
				h.send(typed('1')) // single-select: chosen, then the next question
				if p := h.m.ui.Pick; p.At != 1 || !p.Chosen[0][0] || p.Chosen[0][1] {
					t.Fatalf("pick %+v after 1", p)
				}
				h.send(typed('2')) // multiSelect toggles
				h.send(typed('3'))
				h.send(typed('2'))
				h.send(typed('1'))
				if p := h.m.ui.Pick; p.At != 1 || !p.Chosen[1][0] || p.Chosen[1][1] || !p.Chosen[1][2] {
					t.Fatalf("pick %+v after toggling", p)
				}
				h.send(key(tea.KeyLeft, 0))
				h.send(typed('2')) // Blue replaces Rouge
				if p := h.m.ui.Pick; !p.Chosen[0][1] || p.Chosen[0][0] || p.At != 1 {
					t.Fatalf("pick %+v after replacing", p)
				}
				h.send(key(tea.KeyLeft, 0))
				h.send(key(tea.KeyLeft, 0)) // stays on the first
				h.send(typed('1'))
				h.send(key(tea.KeyRight, 0))
				h.send(key(tea.KeyRight, 0)) // stays on the last
				if p := h.m.ui.Pick; p.At != 1 {
					t.Fatalf("selected %d, want 1", p.At)
				}
				if f := frameText(h); !strings.Contains(f, "1 ● Rouge café ✓") || !strings.Contains(f, "1 [x] Monday") || !strings.Contains(f, "3 [x] Friday") {
					t.Errorf("chosen options not marked:\n%s", f)
				}
				h.send(typed('a'))
				h.settle()
				in := updatedInput(t, w, id)
				if string(in["answers"]) != `{"Which colour?":"Rouge café ✓","Which days?":"Monday, Friday"}` {
					t.Errorf("answers %s", in["answers"])
				}
				var qs []struct {
					Question string `json:"question"`
				}
				if json.Unmarshal(in["questions"], &qs) != nil || len(qs) != 2 || len(in) != 2 {
					t.Errorf("updatedInput %v", in)
				}
				ap := h.m.State().Approvals[id]
				if ap.Status != state.Allowed || ap.Answer != "allow" || ap.Answers["Which colour?"] != "Rouge café ✓" {
					t.Errorf("approval %+v", ap)
				}
				f = frameText(h)
				if !strings.Contains(f, "answered by you") || !strings.Contains(f, "Which colour? → Rouge café ✓") || !strings.Contains(f, "Which days? → Monday, Friday") {
					t.Errorf("answered card:\n%s", f)
				}
			})
		})
	}
}
