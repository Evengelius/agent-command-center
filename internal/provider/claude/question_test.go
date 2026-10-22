package claude

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Evengelius/agent-command-center/internal/event"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// lastLine waits for the log to hold n lines and returns the last.
func lastLine(t *testing.T, log string, n int) string {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		b, _ := os.ReadFile(log)
		if lines := strings.Split(strings.TrimSuffix(string(b), "\n"), "\n"); len(lines) >= n {
			return lines[len(lines)-1]
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("fewer than %d lines written", n)
	return ""
}

// TestAnswerQuestion is plan 14, item 1: a question is answered with the
// request's input unchanged and an "answers" key (probe P14a); an allow
// through Answer is refused.
func TestAnswerQuestion(t *testing.T) {
	log := filepath.Join(t.TempDir(), "stdin.jsonl")
	t.Setenv("ACC_FAKE_LOG", log)
	r, _ := start(t, "question", nil)
	r.wait(event.SessionReady)
	if err := r.sess.Send("ask"); err != nil {
		t.Fatal(err)
	}
	q := r.wait(event.Question).Payload.(event.Approval)
	if q.ID != "q14" {
		t.Fatalf("question id %q", q.ID)
	}
	if err := r.sess.Answer("q14", "allow"); err == nil {
		t.Error("Answer allow on a question was accepted")
	}
	if err := r.sess.AnswerQuestion("nope", map[string]string{}); err == nil {
		t.Error("an unknown request was answered")
	}
	answers := map[string]string{"Which colour?": "Rouge café ✓", "Which days?": "Monday, Tuesday"}
	if err := r.sess.AnswerQuestion("q14", answers); err != nil {
		t.Fatal(err)
	}
	line := lastLine(t, log, 4) // initialize, the usage read (plan 24), the message, then the answer
	var got struct {
		Type     string `json:"type"`
		Response struct {
			Subtype   string `json:"subtype"`
			RequestID string `json:"request_id"`
			Response  struct {
				Behavior     string                     `json:"behavior"`
				UpdatedInput map[string]json.RawMessage `json:"updatedInput"`
			} `json:"response"`
		} `json:"response"`
	}
	if err := json.Unmarshal([]byte(line), &got); err != nil {
		t.Fatalf("%v: %s", err, line)
	}
	if got.Type != "control_response" || got.Response.RequestID != "q14" || got.Response.Response.Behavior != "allow" {
		t.Fatalf("written %s", line)
	}
	in := got.Response.Response.UpdatedInput
	const questions = `[{"question":"Which colour?","header":"Colour","options":[{"label":"Rouge café ✓","description":"Select this colour."},{"label":"Blue","description":"Select this colour."}],"multiSelect":false},{"question":"Which days?","header":"Days","options":[{"label":"Monday","description":"Select Monday."},{"label":"Tuesday","description":"Select Tuesday."}],"multiSelect":true}]`
	if string(in["questions"]) != questions {
		t.Errorf("questions changed:\n got %s\nwant %s", in["questions"], questions)
	}
	if string(in["metadata"]) != `{"source":"x"}` {
		t.Errorf("another key changed: %s", in["metadata"])
	}
	if string(in["answers"]) != `{"Which colour?":"Rouge café ✓","Which days?":"Monday, Tuesday"}` {
		t.Errorf("answers %s", in["answers"])
	}
	if len(in) != 3 {
		t.Errorf("%d keys in updatedInput", len(in))
	}
	if !strings.Contains(line, "Rouge café ✓") { // byte for byte, not escaped
		t.Errorf("answer escaped: %s", line)
	}
	r.until(func(ev event.Event) bool { return ev.Kind == event.ApprovalResolved }, 5*time.Second)
	ap := r.st.Approvals["q14"]
	if ap == nil || ap.Status != state.Allowed || ap.Answer != "allow" {
		t.Errorf("approval %+v, want Allowed with answer allow", ap)
	}
	if err := r.sess.AnswerQuestion("q14", answers); err == nil {
		t.Error("an answered question was answered again")
	}
}

// TestEchoUnicode is CC-10 q13 deterministically: a message with non-ASCII
// text and both kinds of line break reaches the CLI, and comes back, byte for
// byte.
func TestEchoUnicode(t *testing.T) {
	r, _ := start(t, "echo", nil)
	r.wait(event.SessionReady)
	const text = "café ✓ — 日本\r\nline 2\n"
	if err := r.sess.Send(text); err != nil {
		t.Fatal(err)
	}
	var got string
	r.until(func(ev event.Event) bool {
		if c, ok := ev.Payload.(event.Completed); ok && ev.Kind == event.MessageCompleted {
			got = c.Text
			return true
		}
		return false
	}, 10*time.Second)
	if got != text {
		t.Errorf("assistant text %q, want %q", got, text)
	}
}
