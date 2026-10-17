package render

import (
	"strings"
	"testing"
	"time"

	"charm.land/lipgloss/v2"

	"github.com/Evengelius/agent-command-center/internal/event"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// failState is a conversation of five calls, from events through the
// reducer: Bash failed with an error after two blank lines, Read completed,
// Grep failed without output, Edit denied at its approval, Glob running. ok
// completes Bash without the error instead.
func failState(ok bool) *state.State {
	s := state.New()
	s.Screen = state.ScreenDashboard
	s.Context.Provider = "claude-code"
	at := time.Date(2026, 10, 17, 9, 30, 5, 0, time.Local)
	seq := uint64(0)
	apply := func(k event.Kind, p any) {
		seq++
		s.Apply(event.Event{Seq: seq, At: at.Add(time.Duration(seq) * time.Second), Kind: k, Origin: event.Observed, Payload: p})
	}
	apply(event.ToolStarted, event.Tool{ToolID: "t1", Name: "Bash"})
	apply(event.ToolCompleted, event.Tool{ToolID: "t1", IsError: !ok, Output: "\n  \r\nexit status 1: FAIL pkg/a  \r\nmore output"})
	apply(event.ToolStarted, event.Tool{ToolID: "t2", Name: "Read"})
	apply(event.ToolCompleted, event.Tool{ToolID: "t2", Output: "120 lines"})
	apply(event.ToolStarted, event.Tool{ToolID: "t3", Name: "Grep"})
	apply(event.ToolCompleted, event.Tool{ToolID: "t3", IsError: true})
	apply(event.ToolStarted, event.Tool{ToolID: "t4", Name: "Edit"})
	apply(event.ApprovalRequest, event.Approval{ID: "a4", Tool: "Edit", ToolID: "t4"})
	apply(event.ApprovalResolved, event.Approval{ID: "a4", Decision: "deny"})
	apply(event.ToolStarted, event.Tool{ToolID: "t5", Name: "Glob"})
	return s
}

// rowOf is the frame's row holding s, and its index.
func rowOf(t *testing.T, sc *Screen, s string) (line, int) {
	t.Helper()
	for i, l := range sc.lines {
		if strings.Contains(texts([]line{l}), s) {
			return l, i
		}
	}
	t.Fatalf("%q not in the frame:\n%s", s, texts(sc.lines))
	return nil, 0
}

// styleOf is the style of the span holding s on row l.
func styleOf(l line, s string) *lipgloss.Style {
	for _, sp := range l {
		if strings.Contains(sp.text, s) {
			return sp.st
		}
	}
	return nil
}

// TestToolFailureRow is plan 5, item 6, criteria 1, 2, 3, 5 and 6: a failed
// call's row in B is "✗ name · error" with the error's first non-blank line,
// one row cut to the width; a call without output reads "failed", a denied
// one "denied"; the others keep "▸"; the entry cursor keeps the error.
func TestToolFailureRow(t *testing.T) {
	s := failState(false)
	for _, size := range [][2]int{{120, 40}, {200, 60}, {80, 24}} {
		// B's tail and, held, its top: at 80x24 B holds five rows.
		sc := Draw(s, UI{Width: size[0], Height: size[1], Focus: "B"})
		top := Draw(s, UI{Width: size[0], Height: size[1], Focus: "B", ConvHold: true, ConvSeen: len(s.Conversation)})
		sc.lines = append(sc.lines, top.lines...)
		for _, c := range []struct{ has, not string }{
			{"  ✗ Bash · exit status 1: FAIL pkg/a", "more output"},
			{"  ▸ Read", "✗"},
			{"  ✗ Grep · failed", "▸"},
			{"  ✗ Edit · denied", "▸"},
			{"  ▸ Glob", "✗"},
		} {
			l, _ := rowOf(t, sc, c.has)
			if txt := texts([]line{l}); strings.Contains(txt, c.not) {
				t.Errorf("%v: row %q holds %q", size, txt, c.not)
			}
		}
		l, _ := rowOf(t, sc, "✗ Bash")
		if styleOf(l, "✗") != stFailed || styleOf(l, "Bash") != stGrey || styleOf(l, "exit status 1") != stFailed {
			t.Errorf("%v: Bash row styles ✗ %p, name %p, error %p", size, styleOf(l, "✗"), styleOf(l, "Bash"), styleOf(l, "exit status 1"))
		}
		if txt := texts([]line{l}); size[0] >= 120 && !strings.Contains(txt[strings.Index(txt, "FAIL pkg/a"):], "observed") {
			t.Errorf("%v: no origin label on %q", size, texts([]line{l}))
		}
		for i, r := range strings.Split(sc.String(), "\n") {
			if w := lipgloss.Width(r); w > size[0] {
				t.Errorf("%v: row %d is %d wide", size, i, w)
			}
		}
	}

	// A long error is cut to the width, on one row.
	long := failState(false)
	long.Tools["t1"].Output = strings.Repeat("error text ", 40)
	sc := Draw(long, UI{Width: 80, Height: 24, Focus: "B", ConvHold: true, ConvSeen: len(long.Conversation)})
	_, y := rowOf(t, sc, "✗ Bash · error text")
	if strings.Contains(texts(sc.lines[y+1:y+2]), "error text") {
		t.Error("the long error wraps to a second row")
	}

	// Criterion 6, second half; criterion 3: a call that did not fail keeps
	// its "▸" row.
	sc = Draw(failState(true), UI{Width: 120, Height: 40, Focus: "B"})
	if l, _ := rowOf(t, sc, "Bash"); !strings.Contains(texts([]line{l}), "  ▸ Bash") || strings.Contains(texts([]line{l}), "exit status") {
		t.Errorf("completed Bash row %q", texts([]line{l}))
	}

	// Criterion 5: the cursor on the failed row.
	ui := UI{Width: 120, Height: 40, Focus: "B", ConvHold: true, ConvSeen: len(s.Conversation), Entry: 1}
	sc = Draw(s, ui)
	l, _ := rowOf(t, sc, "Bash")
	txt := texts([]line{l})
	if !strings.Contains(txt, "  ▸ Bash · exit status 1: FAIL pkg/a") || !strings.Contains(txt, "Enter full view") {
		t.Errorf("marked row %q", txt)
	}
	if styleOf(l, "▸") != stAttentionBold || styleOf(l, "exit status 1") != stFailed || styleOf(l, "Enter full view") != stAttention {
		t.Errorf("marked row styles ▸ %p, error %p, hint %p", styleOf(l, "▸"), styleOf(l, "exit status 1"), styleOf(l, "Enter full view"))
	}
}

// TestCodexCommandExit is plan 6, item 1, criteria 1 to 3: a Codex command
// that exited non-zero shows "✗ commandExecution · exit N · " and the last
// non-blank line of its output, "exit N" alone when the output is blank; any
// other failed call keeps its first non-blank line; the viewer's text is the
// whole output.
func TestCodexCommandExit(t *testing.T) {
	build := func(out string, code int, has bool, name string) *state.State {
		s := state.New()
		s.Screen = state.ScreenDashboard
		s.Context.Provider = "codex"
		at := time.Date(2026, 10, 17, 9, 30, 5, 0, time.Local)
		s.Apply(event.Event{Seq: 1, At: at, Kind: event.ToolStarted, Origin: event.Observed, Payload: event.Tool{ToolID: "c1", Name: name}})
		s.Apply(event.Event{Seq: 2, At: at.Add(time.Second), Kind: event.ToolCompleted, Origin: event.Observed,
			Payload: event.Tool{ToolID: "c1", Name: name, IsError: true, Output: out, ExitCode: code, HasExitCode: has}})
		return s
	}
	frame := func(s *state.State) string {
		sc := Draw(s, UI{Width: 120, Height: 40, Focus: "B"})
		return texts(sc.lines)
	}
	if f := frame(build("first line\r\nreal error  \n\n", 2, true, "commandExecution")); !strings.Contains(f, "✗ commandExecution · exit 2 · real error") || strings.Contains(f, "first line") {
		t.Errorf("last line: %q", f)
	}
	if f := frame(build(" \n\t\n", 3, true, "commandExecution")); !strings.Contains(f, "✗ commandExecution · exit 3") || strings.Contains(f, "exit 3 ·") {
		t.Errorf("blank output: %q", f)
	}
	// Without an exit code, or for another tool, as before.
	if f := frame(build("first line\nlast line", 0, false, "commandExecution")); !strings.Contains(f, "✗ commandExecution · first line") || strings.Contains(f, "exit") {
		t.Errorf("no exit code: %q", f)
	}
	if f := frame(build("first line\nlast line", 2, true, "Bash")); !strings.Contains(f, "✗ Bash · first line") || strings.Contains(f, "exit") {
		t.Errorf("other tool: %q", f)
	}
	// The whole output stays in the state the viewer reads.
	s := build("first line\nlast line", 2, true, "commandExecution")
	if s.Tools["c1"].Output != "first line\nlast line" || s.Tools["c1"].ExitCode != 2 {
		t.Errorf("state %+v", s.Tools["c1"])
	}
}
