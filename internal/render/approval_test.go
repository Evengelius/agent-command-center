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
		for w := 60; w <= 200; w += 7 {
			ap := &state.Approval{ID: "r1", Tool: tt.tool, Input: tt.input, Status: tt.status, RequestedAt: at, AnsweredAt: at}
			s := &state.State{Approvals: map[string]*state.Approval{"r1": ap}}
			e := state.Entry{Role: "approval", Text: tt.tool, MessageID: "r1"}
			l := approvalCard(s, UI{}, 0, e, w)[0]
			got := ""
			for _, sp := range l {
				got += sp.text
			}
			if !strings.HasSuffix(got, tt.tail) || !strings.Contains(got, tt.keep) || l.width() > w {
				t.Errorf("%s, w %d: %q (%d wide); want it to end with %q, keep %q and fit", tt.name, w, got, l.width(), tt.tail, tt.keep)
			}
			if strings.Contains(got, "……") {
				t.Errorf("%s, w %d: a doubled ellipsis: %q", tt.name, w, got)
			}
			if strings.Contains(got, "someone") {
				t.Errorf("%s, w %d: the user folder shows: %q", tt.name, w, got)
			}
		}
	}
}
