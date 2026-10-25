package render

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/Evengelius/agent-command-center/internal/event"
)

// TestApprovalRevealsFormat is plan 48's criterion 48.11 (design D6; audit
// finding 8): a request's bidi controls and zero-width spaces are drawn as
// <U+XXXX> on its card, a question's too, and in its full view, never as
// they are; the zero-width joiner stays, so an emoji sequence is drawn
// whole; an assistant message is drawn as before.
func TestApprovalRevealsFormat(t *testing.T) {
	rlo, zwsp, zwj := string(rune(0x202E)), string(rune(0x200B)), string(rune(0x200D))
	family := string(rune(0x1F468)) + zwj + string(rune(0x1F469)) + zwj + string(rune(0x1F467))
	cmd := "echo safe" + rlo + "txt.exe" + zwsp
	b := newConvBench("claude-code", "claude-haiku-5-5")
	b.prompt("go")
	b.msg("m1", "an answer with "+rlo+" and "+zwsp+" in it")
	in, _ := json.Marshal(map[string]string{"command": cmd})
	b.call("t1", "Bash", map[string]string{"command": cmd}, false, false)
	b.ev(event.ApprovalRequest, event.Approval{ID: "a1", Tool: "Bash", ToolID: "t1", Input: in, Summary: "lists the files" + zwsp + " of the " + family})
	qin, _ := json.Marshal(map[string]any{"questions": []map[string]any{{"question": "Which one" + rlo + "?", "header": "Pick",
		"options": []map[string]string{{"label": "this" + zwsp, "description": "the first"}}}}})
	b.ev(event.Question, event.Approval{ID: "q1", Tool: "AskUserQuestion", Input: qin})

	rows := rowTexts(bRows(b.s, UI{}, 120))
	box := func(label string) string {
		for i, r := range rows {
			if strings.Contains(r, label) {
				var out []string
				for _, c := range rows[i:] {
					out = append(out, c)
					if strings.Contains(c, "╰") {
						break
					}
				}
				return strings.Join(out, "\n")
			}
		}
		return ""
	}
	message := ""
	for _, r := range rows {
		if strings.Contains(r, "an answer with") {
			message = r
		}
	}
	full := texts(viewerBodyFor(b.s, UI{}, b.s.Approvals["a1"], 100))
	for _, c := range []struct {
		name, text string
		want       []string
	}{
		{"the card", box("! APPROVAL"), []string{"echo safe<U+202E>txt.exe<U+200B>", "lists the files<U+200B> of the " + family}},
		{"the full view", full, []string{"echo safe<U+202E>txt.exe<U+200B>", "lists the files<U+200B> of the " + family}},
		{"the question's card", box("? QUESTION"), []string{"Which one<U+202E>?", "this<U+200B> — the first"}},
	} {
		for _, w := range c.want {
			if !strings.Contains(c.text, w) {
				t.Errorf("%s lacks %q:\n%s", c.name, w, c.text)
			}
		}
		if strings.ContainsAny(c.text, rlo+zwsp) {
			t.Errorf("%s draws a raw format character:\n%s", c.name, c.text)
		}
	}
	if !strings.Contains(message, "an answer with "+rlo+" and "+zwsp+" in it") {
		t.Errorf("the assistant message changed: %q", message)
	}
}
