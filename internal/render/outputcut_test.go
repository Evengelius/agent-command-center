package render

import (
	"strings"
	"testing"
	"time"

	"github.com/Evengelius/agent-command-center/internal/state"
)

// TestOutputCutNote is plan 48's criterion 48.4, its views (design D2): the
// inspection and the full view open a running call's output with how much
// of it the state did not keep, and say nothing once nothing was dropped.
func TestOutputCutNote(t *testing.T) {
	s := state.New()
	tc := &state.ToolCall{ID: "c1", Agent: "a1", Name: "commandExecution", Status: state.ToolRunning, Output: "the tail\n", OutputCut: 139_264}
	s.Tools["c1"] = tc
	a := &state.Agent{Key: "a1"}
	e := state.Entry{Role: "tool", MessageID: "c1", Text: "commandExecution"}
	const note = "… 139 KB of earlier output not kept"
	views := func() (string, string) {
		_, body := toolView(s, UI{Now: time.Time{}}, e, span{}, 100)
		return texts(inspectTools(s, a, 100, time.Time{})), texts(body)
	}
	in, full := views()
	if !strings.Contains(in, "→ "+note) || !strings.Contains(in, "the tail") || strings.Contains(in, "→ the tail") {
		t.Errorf("the inspection:\n%s", in)
	}
	if !strings.Contains(full, note) || !strings.Contains(full, "the tail") {
		t.Errorf("the full view:\n%s", full)
	}
	tc.OutputCut = 0
	if in, full = views(); strings.Contains(in, "not kept") || strings.Contains(full, "not kept") || !strings.Contains(in, "→ the tail") {
		t.Errorf("nothing dropped:\n%s\n%s", in, full)
	}
}
