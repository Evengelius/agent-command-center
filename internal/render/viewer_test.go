package render

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"charm.land/lipgloss/v2"

	"github.com/Evengelius/agent-command-center/internal/state"
)

// viewerRows are the viewer body's rows as text, with each row's styles.
func viewerRows(s *state.State, ap *state.Approval, w int) (rows []string, styles [][]*lipgloss.Style) {
	for _, l := range viewerBody(s, ap, w) {
		t := ""
		var st []*lipgloss.Style
		for _, sp := range l {
			t += sp.text
			st = append(st, sp.st)
		}
		rows, styles = append(rows, t), append(styles, st)
	}
	return rows, styles
}

// TestViewerWhole is plan 5, item 3, criterion 2: every line of the request
// is shown, none cut; a line wider than the screen wraps; the target is whole.
func TestViewerWhole(t *testing.T) {
	var cmd []string
	for i := 1; i <= 40; i++ {
		cmd = append(cmd, fmt.Sprintf("echo %d", i))
	}
	wide := "echo " + strings.Repeat("x", 190) + "END"
	cmd = append(cmd, wide)
	path := `C:\Users\someone\AppData\Local\Temp\` + strings.Repeat("deep\\", 20) + "target.txt"
	in, _ := json.Marshal(map[string]string{"command": strings.Join(cmd, "\n"), "path": path})
	ap := &state.Approval{ID: "r1", Tool: "Bash", Input: in, Status: state.Pending, RequestedAt: time.Now()}
	s := &state.State{Approvals: map[string]*state.Approval{"r1": ap}}
	const w = 80
	rows, _ := viewerRows(s, ap, w)
	all := strings.Join(rows, "\n")
	n := 0
	for _, r := range rows {
		if strings.HasPrefix(r, "$ ") {
			n++
		}
		if lipgloss.Width(r) > w {
			t.Errorf("row wider than %d: %q", w, r)
		}
	}
	if n != 41 {
		t.Errorf("%d \"$ \" rows, want 41 (40 lines and the wide one):\n%s", n, all)
	}
	for i := 1; i <= 40; i++ {
		if !strings.Contains(all, fmt.Sprintf("$ echo %d\n", i)) {
			t.Errorf("line %d missing:\n%s", i, all)
		}
	}
	if strings.Contains(all, "more lines") || strings.Count(all, "…") != 1 { // the one is the user folder, hidden
		t.Errorf("the request is cut:\n%s", all)
	}
	if strings.Count(all, strings.Repeat("x", 78)) != 2 || !strings.Contains(all, strings.Repeat("x", 34)+"END") { // 190 x on rows of 78
		t.Errorf("the wide line is not whole once its rows are joined:\n%s", all)
	}
	if !strings.Contains(strings.ReplaceAll(all, "\n", ""), `\target.txt`) || !strings.Contains(all, "deep") {
		t.Errorf("the target is not whole:\n%s", all)
	}
	if strings.Contains(all, "someone") {
		t.Errorf("the user folder shows:\n%s", all)
	}
}

// TestViewerDiff is criterion 3: Claude Code's Edit, MultiEdit and Write and
// Codex's fileChange (from its tool call) are coloured diffs.
func TestViewerDiff(t *testing.T) {
	edit, _ := json.Marshal(map[string]string{"file_path": "a.go", "old_string": "old one\nold two", "new_string": "new one"})
	multi, _ := json.Marshal(map[string]any{"file_path": "m.go", "edits": []map[string]string{{"old_string": "old one", "new_string": "new one"}}})
	write, _ := json.Marshal(map[string]string{"file_path": "w.txt", "content": "new one\n"})
	change, _ := json.Marshal(map[string]any{"changes": []map[string]any{
		{"path": "hello.txt", "kind": map[string]string{"type": "add"}, "diff": "new one\n"}, // P6b's shape
		{"path": "b.txt", "kind": map[string]string{"type": "update"}, "diff": "@@ -1 +1 @@\n-old one\n+new one\n"},
	}})
	for _, c := range []struct {
		tool   string
		input  json.RawMessage
		header []string // the file headers drawn; Claude Code's file is the target, not repeated
		old    bool
	}{
		{"Edit", edit, nil, true},
		{"MultiEdit", multi, nil, true},
		{"Write", write, nil, false},
		{"fileChange", nil, []string{"hello.txt  add", "b.txt  update"}, true},
	} {
		t.Run(c.tool, func(t *testing.T) {
			ap := &state.Approval{ID: "r1", Tool: c.tool, Input: c.input, ToolID: "t1", Status: state.Pending}
			s := &state.State{Approvals: map[string]*state.Approval{"r1": ap}, Tools: map[string]*state.ToolCall{}}
			if c.tool == "fileChange" {
				ap.Input = json.RawMessage(`{"itemId":"t1","reason":null}`)
				s.Tools["t1"] = &state.ToolCall{ID: "t1", Name: "fileChange", Input: change}
			}
			rows, styles := viewerRows(s, ap, 80)
			all := strings.Join(rows, "\n")
			var plus, minus bool
			for i, r := range rows {
				if len(styles[i]) == 0 {
					continue
				}
				last := styles[i][len(styles[i])-1]
				switch {
				case strings.HasPrefix(r, "+new one"):
					plus = true
					if last != stDone {
						t.Errorf("%q not in the done colour", r)
					}
				case strings.HasPrefix(r, "-old one"):
					minus = true
					if last != stFailed {
						t.Errorf("%q not in the failed colour", r)
					}
				}
			}
			if !plus || minus != c.old {
				t.Errorf("+ row %v, - row %v (want %v):\n%s", plus, minus, c.old, all)
			}
			for _, h := range c.header {
				if !strings.Contains(all, h) {
					t.Errorf("no header %q:\n%s", h, all)
				}
			}
			if c.header == nil && strings.Count(all, ".") != 1 { // the path once, as the target
				t.Errorf("the file is named more than once:\n%s", all)
			}
		})
	}
}

// TestViewerHead is criteria 1 and 7: the head names the approval, who asks
// and the tool, then its outcome once resolved; the hints follow.
func TestViewerHead(t *testing.T) {
	at := time.Date(2026, 10, 17, 9, 30, 5, 0, time.Local)
	in, _ := json.Marshal(map[string]string{"command": "ls"})
	for _, c := range []struct {
		status          state.ApprovalStatus
		question        bool
		label, outcome  string
		keys, notInKeys string
	}{
		{state.Pending, false, "! APPROVAL", "requested " + clock(at), "a allow once", ""},
		{state.Pending, true, "? QUESTION", "requested " + clock(at), "d deny", "a allow once"},
		{state.Allowed, false, "✓ ALLOWED", "allowed by you " + clockSec(at), "Esc back", "a allow once"},
		{state.Denied, false, "✗ DENIED", "denied by you " + clockSec(at), "Esc back", "d deny"},
		{state.Expired, false, "· EXPIRED", "expired, not answered", "Esc back", "d deny"},
	} {
		ap := &state.Approval{ID: "r1", Tool: "Bash", Input: in, Question: c.question, Status: c.status, RequestedAt: at, AnsweredAt: at}
		s := &state.State{Approvals: map[string]*state.Approval{"r1": ap}}
		ls := viewer(s, UI{Width: 120, Height: 30, View: "r1"}, ap, &Screen{})
		text := func(l line) string {
			t := ""
			for _, sp := range l {
				t += sp.text
			}
			return t
		}
		var all []string
		for _, l := range ls {
			all = append(all, text(l))
		}
		head := strings.Join(all[:3], "\n")
		for _, want := range []string{"◀ back (Esc)", c.label, "orchestrator · Bash", c.outcome} {
			if !strings.Contains(head, want) {
				t.Errorf("%s: head lacks %q:\n%s", c.status, want, head)
			}
		}
		foot := all[len(all)-2] + all[len(all)-1]
		if !strings.Contains(foot, c.keys) || (c.notInKeys != "" && strings.Contains(foot, c.notInKeys)) {
			t.Errorf("%s, question %v: hints %q; want %q, not %q", c.status, c.question, foot, c.keys, c.notInKeys)
		}
	}
	// Plan 5, item 8, criteria 4 and 5: s in the hints when the request
	// offers it; the session answer named once answered.
	ap := &state.Approval{ID: "r1", Tool: "commandExecution", Input: in, ForSession: true, Status: state.Pending, RequestedAt: at}
	s := &state.State{Approvals: map[string]*state.Approval{"r1": ap}}
	if got := texts(viewer(s, UI{Width: 120, Height: 30, View: "r1"}, ap, &Screen{})); !strings.Contains(got, "a allow once · s allow for the session · d deny") {
		t.Errorf("offered, pending: hints lack s:\n%s", got)
	}
	ap.Status, ap.Answer, ap.AnsweredAt = state.Allowed, "acceptForSession", at
	if got := texts(viewer(s, UI{Width: 120, Height: 30, View: "r1"}, ap, &Screen{})); !strings.Contains(got, "allowed for the session by you "+clockSec(at)) {
		t.Errorf("allowed for the session: head:\n%s", got)
	}
	if !strings.Contains(texts(keysBody(78)), "allow for the session, when a Codex request offers it") { // criterion 6
		t.Error("the key overlay lacks s")
	}
	// Plan 5, item 10, criterion 3: a denied question says where its answer
	// goes; a denied approval, or a cancelled question, does not.
	for _, c := range []struct{ q, cancelled bool }{{true, false}, {false, false}, {true, true}} {
		ap.Question, ap.Cancelled, ap.Status, ap.Answer = c.q, c.cancelled, state.Denied, "deny"
		got := texts(viewer(s, UI{Width: 120, Height: 30, View: "r1"}, ap, &Screen{}))
		if strings.Contains(got, "denied by you "+clockSec(at)+" · answer in your next message") != (c.q && !c.cancelled) || !strings.Contains(got, "denied by you") {
			t.Errorf("denied, %+v: head:\n%s", c, got)
		}
	}
}

// entryState is a conversation with an entry of each kind; the expandable
// ones are a 30-line message (2), a tool call (4) and an error (5).
func entryState() *state.State {
	s := state.New()
	s.Screen = state.ScreenDashboard
	s.Context.Provider = "claude-code"
	at := time.Date(2026, 10, 17, 9, 30, 5, 0, time.Local)
	var paste []string
	for i := 1; i <= 30; i++ {
		paste = append(paste, fmt.Sprintf("pasted line %d", i))
	}
	in, _ := json.Marshal(map[string]string{"command": "go test ./...\ngo vet ./..."})
	s.Tools["t1"] = &state.ToolCall{ID: "t1", Name: "Bash", Input: in, Status: state.ToolCompleted, Output: "ok  pkg/a\nok  pkg/b\n", StartedAt: at, EndedAt: at.Add(2 * time.Second)}
	for _, e := range []state.Entry{
		{Role: "user", Text: "a short question"},
		{Role: "user", Text: strings.Join(paste, "\n")},
		{Role: "assistant", Text: "an answer\nover two lines"},
		{Role: "tool", MessageID: "t1", Text: "Bash"},
		{Role: "error", Text: "the API refused the request\nstatus 529: overloaded"},
		{Role: "dashboard", Text: "a note"},
	} {
		e.Seq, e.At = uint64(len(s.Conversation)+1), at
		s.Conversation = append(s.Conversation, e)
	}
	return s
}

// TestEntryCursor is plan 5, item 4, criterion 1: the expandable entries are
// the folded message, the tool call and the error; the cursor marks its
// entry's row in the attention colour, adds no row, and shows only while B
// has the focus.
func TestEntryCursor(t *testing.T) {
	s := entryState()
	var got []int
	for i, e := range s.Conversation {
		if Expandable(e) {
			got = append(got, i+1)
		}
	}
	if fmt.Sprint(got) != "[2 4 5]" {
		t.Fatalf("expandable %v, want [2 4 5]", got)
	}
	for _, size := range [][2]int{{120, 40}, {200, 60}, {80, 24}} {
		ui := UI{Width: size[0], Height: size[1], Focus: "B", ConvHold: true, ConvSeen: len(s.Conversation)}
		cv := Draw(s, ui).Conv
		starts := cv.Starts
		for _, n := range got {
			ui.ConvTop = max(starts[n-1], starts[n]-cv.Room) // in view, its end first, as the app scrolls it
			base := Draw(s, ui)
			ui.Entry = n
			sc := Draw(s, ui)
			if sc.Conv.Rows != base.Conv.Rows || len(sc.Conv.Starts) != len(s.Conversation)+1 {
				t.Errorf("%v, entry %d: %d rows, want %d; %d starts", size, n, sc.Conv.Rows, base.Conv.Rows, len(sc.Conv.Starts))
			}
			var diff []string
			attention := false
			for i, l := range sc.lines {
				if a, b := texts(base.lines[i:i+1]), texts(sc.lines[i:i+1]); a != b {
					diff = append(diff, b)
					for _, sp := range l {
						// A tool row's hint is on the tool rows' ground (plan 19).
						attention = attention || (sp.st == stAttention || sp.st == tint(stAttention)) && strings.Contains(sp.text, "Enter full view")
					}
				}
			}
			if !attention {
				t.Errorf("%v, entry %d: the hint is not in the attention colour", size, n)
			}
			// The marked row, and the key hints at the bottom.
			if len(diff) != 2 || !strings.Contains(diff[0], "Enter full view · ↑↓ entries · Esc back") || !strings.Contains(diff[1], "Enter full view · ↑↓ entries · Esc back to the input") {
				t.Errorf("%v, entry %d: changed rows %q", size, n, diff)
			}
			for _, focus := range []string{"A", "C"} {
				ui.Focus = focus
				off := Draw(s, ui)
				ui.Entry = 0
				if want := Draw(s, ui).String(); off.String() != want {
					t.Errorf("%v, entry %d: the cursor shows with focus %s", size, n, focus)
				}
				ui.Entry, ui.Focus = n, "B"
			}
			ui.Entry = 0
		}
	}
}

// TestEntryViewerBody is criteria 3 and 5: each entry whole, its rows never
// cut; a tool's input, then its output as the state holds it now.
func TestEntryViewerBody(t *testing.T) {
	s := entryState()
	view := func(n int) string {
		return texts(entryViewer(s, UI{Width: 100, Height: 300, ViewEntry: n, InspectHold: true}, n-1, &Screen{}))
	}
	paste := view(2)
	for _, want := range []string{"◀ back (Esc)", "YOU", clock(s.Conversation[1].At), "30 lines"} {
		if !strings.Contains(paste, want) {
			t.Errorf("message head lacks %q:\n%s", want, paste)
		}
	}
	for i := 1; i <= 30; i++ {
		if !strings.Contains(paste, fmt.Sprintf("pasted line %d\n", i)) && !strings.Contains(paste, fmt.Sprintf("pasted line %d ", i)) {
			t.Errorf("line %d missing:\n%s", i, paste)
		}
	}
	if strings.Contains(paste, "more lines") {
		t.Errorf("the message is cut:\n%s", paste)
	}

	tool := view(4)
	for _, want := range []string{"✓ Bash", "completed · 2s", "$ go test ./...", "go vet ./...", "OUTPUT", "ok  pkg/a", "ok  pkg/b"} {
		if !strings.Contains(tool, want) {
			t.Errorf("tool view lacks %q:\n%s", want, tool)
		}
	}
	tc := s.Tools["t1"]
	tc.Output, tc.Status = "", state.ToolRunning
	if tool := view(4); !strings.Contains(tool, "no output") || !strings.Contains(tool, "running") {
		t.Errorf("a running call:\n%s", tool)
	}
	tc.Input = json.RawMessage(`{"alpha":1,"beta":[2,3]}`)
	if tool := view(4); !strings.Contains(tool, `{"alpha":1,"beta":[2,3]}`) {
		t.Errorf("an input of no known shape is not shown as received:\n%s", tool)
	}
	tc.Name, tc.Input = "Edit", json.RawMessage(`{"file_path":"a.go","old_string":"old one","new_string":"new one"}`)
	if tool := view(4); !strings.Contains(tool, "-old one") || !strings.Contains(tool, "+new one") {
		t.Errorf("an edit is not its diff:\n%s", tool)
	}
	delete(s.Tools, "t1")
	if tool := view(4); !strings.Contains(tool, "no call recorded") {
		t.Errorf("an entry without its call:\n%s", tool)
	}

	failed := view(5)
	for _, want := range []string{"✗ TURN FAILED", "the API refused the request", "status 529: overloaded"} {
		if !strings.Contains(failed, want) {
			t.Errorf("error view lacks %q:\n%s", want, failed)
		}
	}
}
