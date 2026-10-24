package render

import (
	"encoding/json"
	"fmt"
	"regexp"
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
// one "denied"; the entry cursor keeps the error. Since plan 19 the other
// calls show their status, ✓ done and ◉ running, where they had "▸", every
// row on the tool rows' ground, the name in the text colour.
func TestToolFailureRow(t *testing.T) {
	s := failState(false)
	for _, size := range [][2]int{{120, 40}, {200, 60}, {80, 24}} {
		// B's tail and, held, its top: at 80x24 B holds five rows.
		sc := Draw(s, UI{Width: size[0], Height: size[1], Focus: "B"})
		top := Draw(s, UI{Width: size[0], Height: size[1], Focus: "B", ConvHold: true, ConvSeen: len(s.Conversation)})
		sc.lines = append(sc.lines, top.lines...)
		for _, c := range []struct{ has, not string }{
			{"  ✗ Bash · exit status 1: FAIL pkg/a", "more output"},
			{"  ✓ Read", "✗"},
			{"  ✗ Grep", "▸"},
			{"  ✗ Edit", "▸"},
			{"  ◉ Glob", "✗"},
		} {
			l, _ := rowOf(t, sc, c.has)
			// B's part of the screen row: zone A's tree shares it.
			txt := texts([]line{l})
			if _, b, ok := strings.Cut(txt, "││"); ok {
				txt = b
			}
			if strings.Contains(txt, c.not) {
				t.Errorf("%v: row %q holds %q", size, txt, c.not)
			}
		}
		// A call without output reads "failed", a denied one "denied": in
		// the outcome column since plan 20, with their time.
		for name, word := range map[string]string{"✗ Grep": "failed 09:30", "✗ Edit": "denied 09:30"} {
			if l, _ := rowOf(t, sc, name); !strings.Contains(texts([]line{l}), word) {
				t.Errorf("%v: %s row %q lacks %q", size, name, texts([]line{l}), word)
			}
		}
		l, _ := rowOf(t, sc, "✗ Bash")
		if styleOf(l, "✗") != tint(stFailed) || styleOf(l, "Bash") != tint(stToolName) || styleOf(l, "exit status 1") != tint(stFailed) {
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
	// its row without an error, ✓ since plan 19.
	sc = Draw(failState(true), UI{Width: 120, Height: 40, Focus: "B"})
	if l, _ := rowOf(t, sc, "Bash"); !strings.Contains(texts([]line{l}), "  ✓ Bash") || strings.Contains(texts([]line{l}), "exit status") {
		t.Errorf("completed Bash row %q", texts([]line{l}))
	}

	// Criterion 5: the cursor on the failed row, in its margin since plan 19;
	// at 140 columns, where B's row holds the error and the hint whole (B is
	// 10 columns narrower at 120 since plan 24).
	ui := UI{Width: 140, Height: 40, Focus: "B", ConvHold: true, ConvSeen: len(s.Conversation), Entry: 1}
	sc = Draw(s, ui)
	l, _ := rowOf(t, sc, "Bash")
	txt := texts([]line{l})
	if !strings.Contains(txt, "▸ ✗ Bash · exit status 1: FAIL pkg/a") || !strings.Contains(txt, "Enter full view") {
		t.Errorf("marked row %q", txt)
	}
	if styleOf(l, "▸") != stAttentionBold || styleOf(l, "exit status 1") != tint(stFailed) || styleOf(l, "Enter full view") != tint(stAttention) {
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

// TestFailedRowNamesItsTarget is the owner's live review of 2026-10-20,
// scenario 2: twelve Read calls on deep paths, the fourth failing. The
// failed row names its file, then its error, and keeps "failed" and its
// time at the right; every path keeps its file name, cut at a separator; in
// zone A's tree too, where the rows all read "C:\User…".
func TestFailedRowNamesItsTarget(t *testing.T) {
	s := state.New()
	s.Screen = state.ScreenDashboard
	s.Context.Provider = "claude-code"
	at := time.Date(2026, 10, 20, 12, 15, 18, 0, time.Local)
	seq := uint64(0)
	apply := func(k event.Kind, p any) {
		seq++
		s.Apply(event.Event{Seq: seq, At: at.Add(time.Duration(seq) * time.Second), Kind: k, Origin: event.Observed, Payload: p})
	}
	apply(event.UserMessage, event.User{Text: "Read the twelve files"})
	apply(event.TurnStarted, nil)
	const dir = `C:\\Users\\someone\\AppData\\Local\\Temp\\acc-demo\\src\\`
	for i := range 12 {
		id := fmt.Sprintf("r%02d", i)
		apply(event.ToolStarted, event.Tool{ToolID: id, Name: "Read", Input: json.RawMessage(fmt.Sprintf(`{"file_path":"%sf%02d.go"}`, dir, i))})
		if i == 3 {
			apply(event.ToolCompleted, event.Tool{ToolID: id, IsError: true, Status: "error", Output: "File does not exist. Note: your current working directory is C:\\Users\\someone\\AppData\\Local\\Temp\\acc-demo."})
		} else {
			apply(event.ToolCompleted, event.Tool{ToolID: id, Status: "completed", Output: "package src"})
		}
	}
	apply(event.MessageCompleted, event.Completed{MessageID: "m1", Text: "I read 11 of the 12 files."})
	apply(event.TurnCompleted, event.TurnEnd{Outcome: "completed"})
	ui := func(w, h int) UI {
		return UI{Width: w, Height: h, Focus: "B", Unfolded: map[string]bool{"r00": true}} // the run expanded
	}
	for _, size := range [][2]int{{156, 44}, {120, 40}, {200, 60}, {80, 40}} {
		// The error as the row shows it: whole, except at 120 columns, where
		// zone B lost the sidebar's 8 columns (plan 43, design D1) and the
		// row is cut at its edge, the error first, "failed" and its time kept.
		shown := "· File does not exist."
		if size[0] == 120 {
			shown = "· File does not ex…"
		}
		// B's tail and, held, its top; 80 columns 40 rows tall, for the failed
		// row to be in one of them under the messages' spacing (amendment 17).
		sc := Draw(s, ui(size[0], size[1]))
		held := ui(size[0], size[1])
		held.ConvHold, held.ConvSeen = true, len(s.Conversation)
		sc.lines = append(sc.lines, Draw(s, held).lines...)
		frame := texts(sc.lines)
		for i, r := range strings.Split(frame, "\n") {
			if w := lipgloss.Width(r); w > size[0] {
				t.Errorf("%v: row %d is %d wide", size, i, w)
			}
		}
		// The failed row in B's part of the screen, not zone A's tree beside it.
		var l line
		row := ""
		for _, x := range sc.lines {
			txt := texts([]line{x})
			if _, b, ok := strings.Cut(txt, "││"); ok {
				txt = b
			}
			if strings.Contains(txt, "✗ Read") {
				l, row = x, txt
				break
			}
		}
		for _, want := range []string{"f03.go " + shown, "failed 12:15"} {
			if !strings.Contains(row, want) {
				t.Errorf("%v: the failed row %q lacks %q", size, row, want)
			}
		}
		if styleOf(l, "File does not ex") != tint(stFailed) {
			t.Errorf("%v: the error styled %p", size, styleOf(l, "File does not ex"))
		}
		// No path is cut inside an element, nor shows the user folder.
		for _, r := range strings.Split(frame, "\n") {
			if strings.Contains(r, "Read") && (strings.Contains(r, "someone") || regexp.MustCompile(`…[^\\ ]+\\…`).MatchString(r)) {
				t.Errorf("%v: a path cut inside an element or showing the user folder: %q", size, r)
			}
		}
		if size[0] >= standardWidth { // zone A's tree beside B
			if !strings.Contains(frame, "f00.go") || strings.Count(frame, "C:\\User… ") > 0 {
				t.Errorf("%v: zone A's Read rows lost their file names:\n%s", size, frame)
			}
		}
	}
}

// TestElidePath checks a path is cut at one of its elements, the hidden
// user folder never meeting the ellipsis, and as elide when even the file
// name does not fit.
func TestElidePath(t *testing.T) {
	p := `C:\Users\someone\AppData\Local\Temp\acc-demo\src\f00.go`
	for w := 6; w <= 60; w++ {
		got := elidePath(p, w)
		switch {
		case textWidth(got) > w:
			t.Errorf("%d: %q is %d wide", w, got, textWidth(got))
		case strings.Contains(got, "someone") || strings.Contains(got, "……"):
			t.Errorf("%d: %q", w, got)
		case w >= 8 && !strings.HasSuffix(got, "f00.go"):
			t.Errorf("%d: %q lost the file name", w, got)
		case got != hideUser(p) && w >= 8 && !strings.HasPrefix(got, "…\\"):
			t.Errorf("%d: %q not cut at an element", w, got)
		}
	}
	if got := elidePath(p, 44); got != `…\AppData\Local\Temp\acc-demo\src\f00.go` {
		t.Errorf("44: %q", got)
	}
}
