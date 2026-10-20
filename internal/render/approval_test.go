package render

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/Evengelius/agent-command-center/internal/state"
)

// TestApprovalLineKeepsOutcome checks an approval's first line keeps its
// outcome or clock whatever the target's length (ui-design.md, Approvals):
// at 160 columns, L2's Write to a scratch folder cut "allowed by you" off.
// A path keeps its file name, a command its start, and the line fits.
func TestApprovalLineKeepsOutcome(t *testing.T) {
	long := `C:\Users\someone\AppData\Local\Temp\claude\c--javadev-projets-agent-command-center\9773140c-c4c3-4795-90ba-b9eb23a58bd8\scratchpad\l2-project\allow.txt`
	at := time.Date(2026, 10, 16, 19, 37, 5, 0, time.Local)
	write, _ := json.Marshal(map[string]string{"file_path": long, "content": "ok"})
	bash, _ := json.Marshal(map[string]string{"command": "ping -n 30 127.0.0.1 " + strings.Repeat("x", 200)})
	tests := []struct {
		name, tool string
		input      json.RawMessage
		status     state.ApprovalStatus
		tail, keep string
	}{
		{"allowed path", "Write", write, state.Allowed, "allowed by you " + clockSec(at), `\allow.txt`},
		{"denied path", "Write", write, state.Denied, "denied by you " + clockSec(at), `\allow.txt`},
		{"expired path", "Write", write, state.Expired, "expired, not answered", `\allow.txt`},
		{"pending path", "Write", write, state.Pending, clock(at), `\allow.txt`},
		{"pending command", "Bash", bash, state.Pending, clock(at), "  ping -n 30"},
	}
	for _, tt := range tests {
		// An answered request is a row of the tool block, three cells in, its
		// ground to the edge (plan 19): from 63 it has the room 60 gave it.
		from := 60
		if tt.status != state.Pending {
			from = 63
		}
		for w := from; w <= 200; w += 7 {
			ap := &state.Approval{ID: "r1", Tool: tt.tool, Input: tt.input, Status: tt.status, RequestedAt: at, AnsweredAt: at}
			s := &state.State{Approvals: map[string]*state.Approval{"r1": ap}}
			e := state.Entry{Role: "approval", Text: tt.tool, MessageID: "r1"}
			// A pending request is a box since plan 20: its heading is its
			// second row, between the box's edges.
			ls := approvalCard(s, UI{}, 0, e, w, look{})
			l := ls[0]
			target := l
			if tt.status == state.Pending { // the heading, then the target on its own row (section 5.2)
				l, target = ls[1], ls[2]
			}
			got := ""
			for _, sp := range l {
				got += sp.text
			}
			if tt.status == state.Pending {
				got = strings.TrimSuffix(strings.TrimPrefix(got, "│ "), " │")
			}
			got = strings.TrimRight(got, " ")
			kept := texts([]line{target})
			if !strings.HasSuffix(got, tt.tail) || !strings.Contains(kept, strings.TrimSpace(tt.keep)) || l.width() > w || target.width() > w {
				t.Errorf("%s, w %d: %q (%d wide); want it to end with %q, keep %q and fit", tt.name, w, got, l.width(), tt.tail, tt.keep)
			}
			if strings.Contains(got, "……") || strings.Contains(kept, "……") {
				t.Errorf("%s, w %d: a doubled ellipsis: %q %q", tt.name, w, got, kept)
			}
			if strings.Contains(got, "someone") || strings.Contains(kept, "someone") {
				t.Errorf("%s, w %d: the user folder shows: %q %q", tt.name, w, got, kept)
			}
		}
	}
}

// TestApprovalEffect checks a pending card's sentence on what allowing does
// (plan 20, Template 2's "This will create or overwrite the file at …"):
// only for Claude Code's Write and Edit, whose effect its tools reference
// documents, from the request's own target; none for another tool or CLI.
func TestApprovalEffect(t *testing.T) {
	at := time.Date(2026, 10, 20, 1, 56, 0, 0, time.Local)
	for _, c := range []struct {
		cli, tool, input, want string
	}{
		{"claude-code", "Write", `{"file_path":"C:\\work\\found.txt","content":"alpha.txt\nbeta.md\n"}`, "This will create or overwrite the file at `C:\\work\\found.txt` with the above content."},
		{"claude-code", "Write", `{"file_path":"C:\\work\\empty.txt","content":""}`, "This will create or overwrite the file at `C:\\work\\empty.txt`, leaving it empty."},
		{"claude-code", "Edit", `{"file_path":"C:\\work\\a.go","old_string":"x := 1","new_string":"x := 2"}`, "This will replace the text marked - with the text marked + in `C:\\work\\a.go`."},
		{"claude-code", "Edit", `{"file_path":"C:\\work\\a.go","old_string":"x","new_string":"y","replace_all":true}`, "This will replace every occurrence of the text marked - with the text marked + in `C:\\work\\a.go`."},
		{"claude-code", "Bash", `{"command":"ls -p"}`, ""},
		{"claude-code", "mcp__fs__write", `{"file_path":"C:\\work\\found.txt","content":"x"}`, ""},
		{"codex", "fileChange", `{"file_path":"C:\\work\\found.txt","content":"x"}`, ""},
	} {
		ap := &state.Approval{ID: "r1", Tool: c.tool, Input: json.RawMessage(c.input), Status: state.Pending, RequestedAt: at}
		s := &state.State{Approvals: map[string]*state.Approval{"r1": ap}, Context: state.Context{Provider: c.cli}}
		e := state.Entry{Role: "approval", Text: c.tool, MessageID: "r1"}
		for _, w := range []int{97, 160} {
			// The card's text, its edges out, its rows joined: the sentence wraps.
			got := strings.Join(strings.Fields(strings.ReplaceAll(texts(approvalCard(s, UI{}, 0, e, w, look{})), "│", "")), " ")
			if c.want == "" && strings.Contains(got, "This will") || c.want != "" && !strings.Contains(got, c.want) {
				t.Errorf("%s %s at %d: %q; want the sentence %q", c.cli, c.tool, w, got, c.want)
			}
		}
	}
}
