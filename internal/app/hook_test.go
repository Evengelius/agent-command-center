package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/synctest"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/Evengelius/agent-command-center/internal/fixture"
	"github.com/Evengelius/agent-command-center/internal/provider/claude"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// Plan 22, amendment 20: a failing SessionStart hook in the application. The
// CLI carries on after it, so the session must stay usable; the hook's end is a
// notice in zone B (observed), drawn before any prompt. P22o is the recording
// (the owner's hook; its end is line 4, the handshake's response line 5), with
// that end made a failure: outcome "error", exit code 1, stderr "pxw: not found".

const failedHook = "hook SessionStart:startup failed: exit 1: pxw: not found"

// failingHookFile writes P22o with its hook_response made a failure, and gives
// its path.
func failingHookFile(t *testing.T) string {
	t.Helper()
	recs, err := fixture.Read(filepath.Join("..", "..", "testdata", "probes", p22oProbe))
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join("..", "..", "testdata", "probes", p22oProbe))
	if err != nil {
		t.Fatal(err)
	}
	line := string(recs[3].Msg)
	failing := line
	for _, c := range [][2]string{{`"stderr":""`, `"stderr":"pxw: not found\n"`}, {`"exit_code":0`, `"exit_code":1`}, {`"outcome":"success"`, `"outcome":"error"`}} {
		if !strings.Contains(failing, c[0]) {
			t.Fatalf("P22o line 4 lacks %s: %s", c[0], line)
		}
		failing = strings.Replace(failing, c[0], c[1], 1)
	}
	s := strings.Replace(string(b), line, failing, 1)
	if s == string(b) {
		t.Fatal("P22o's line 4 was not found in the file")
	}
	p := filepath.Join(t.TempDir(), "failing-"+p22oProbe)
	if err := os.WriteFile(p, []byte(s), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

// zoneBLeft is the column the conversation zone's left border is at, read from
// its top edge; -1 when the frame has none.
func zoneBLeft(frame string) int {
	for _, l := range strings.Split(frame, "\n") {
		if i := strings.Index(l, "╭─ B"); i >= 0 {
			return lipgloss.Width(l[:i])
		}
	}
	return -1
}

// A notice before any prompt, in the zone that holds it: every line of the
// frame is as high and as wide as the terminal, every row of a zone ends on its
// border, and the notice's text is drawn in zone B.
func checkNoticeFrame(t *testing.T, frame string, w, h int) {
	t.Helper()
	lines := strings.Split(frame, "\n")
	if len(lines) != h {
		t.Errorf("%dx%d: %d lines, want %d", w, h, len(lines), h)
	}
	for i, l := range lines {
		if n := lipgloss.Width(l); n > w {
			t.Errorf("%dx%d: line %d is %d columns wide: %q", w, h, i+1, n, l)
		}
		if strings.HasPrefix(l, "│") || strings.HasPrefix(l, "╭") || strings.HasPrefix(l, "╰") {
			if n := lipgloss.Width(l); n != w {
				t.Errorf("%dx%d: zone row %d is %d columns wide, want %d (its border off): %q", w, h, i+1, n, w, l)
			}
		}
	}
	left := zoneBLeft(frame)
	for _, l := range lines {
		if i := strings.Index(l, failedHook); i >= 0 {
			if x := lipgloss.Width(l[:i]); left < 0 || x <= left {
				t.Errorf("%dx%d: the notice starts at column %d, zone B at %d: not in zone B\n%s", w, h, x, left, frame)
			}
			return
		}
	}
	t.Errorf("%dx%d: no %q in\n%s", w, h, failedHook, frame)
}

// After the handshake and before any prompt, the notice is in zone B at 120x40
// and at 80x24, the session is ready and not closed, and a draft is sent to it.
func TestFailingHookNoticeBeforeAnyPrompt(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := thinkStartFile(t, failingHookFile(t), claude.ProviderID)
		sess := r.h.m.sess.(*scriptSession)
		for _, size := range [][2]int{{120, 40}, {80, 24}, {120, 40}} {
			r.h.draw(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
			r.until(1458) // the response to the handshake is at 1,455 ms, the owner's prompt at 1,459
			r.at(1458)
			checkNoticeFrame(t, plainText(r.h.m.View().Content), size[0], size[1])
		}
		s := r.h.m.State()
		if s.Session.Phase != state.Ready || s.Startup.Phase != state.StartReady || s.Screen != state.ScreenDashboard {
			t.Fatalf("session %q (%s), startup %q, screen %q; want a ready session on the dashboard", s.Session.Phase, s.Session.Reason, s.Startup.Phase, s.Screen)
		}
		if len(s.Errors) != 0 || s.Unrecognised != 0 {
			t.Errorf("%d errors, %d unrecognised events", len(s.Errors), s.Unrecognised)
		}
		if len(s.Conversation) != 1 || s.Conversation[0].Role != "notice" || s.Conversation[0].Text != failedHook {
			t.Errorf("conversation %+v, want the one notice", s.Conversation)
		}

		// The session is usable: a draft goes to the CLI.
		r.h.send(tea.PasteMsg{Content: "hello"})
		r.h.send(key(tea.KeyEnter, 0))
		if sent, _, _ := sess.calls(); len(sent) != 1 || sent[0] != "hello" {
			t.Errorf("writes %q, want the draft once", sent)
		}
		if s.Session.Phase != state.Ready || s.Turn.Phase == state.Idle {
			t.Errorf("after the draft: session %q, turn %q; want a ready session and a turn under way", s.Session.Phase, s.Turn.Phase)
		}
	})
}

// The whole recording after the failing hook: the turn runs and ends, the
// notice stays at the top of zone B ahead of the prompt, and zone D counts no
// error.
func TestFailingHookTurnRuns(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := thinkStartFile(t, failingHookFile(t), claude.ProviderID)
		r.h.draw(tea.WindowSizeMsg{Width: 200, Height: 60}) // zone A's telemetry pane in its full form
		text := r.frameAt(14400)                            // after the result line (14,361 ms)
		s := r.h.m.State()
		if s.Session.Phase != state.Ready || s.Turn.Phase != state.Idle || s.Turn.Outcome != "completed" {
			t.Errorf("session %q (%s), turn %q (%q); want a ready session and a completed turn", s.Session.Phase, s.Session.Reason, s.Turn.Phase, s.Turn.Outcome)
		}
		var roles []string
		for _, e := range s.Conversation {
			roles = append(roles, e.Role)
		}
		if strings.Join(roles, " ") != "notice user thinking assistant" {
			t.Errorf("conversation %v", roles)
		}
		for _, want := range []string{failedHook, "Thought for 8.0s · 928 thinking tokens", "**59**"} {
			if !strings.Contains(text, want) {
				t.Errorf("no %q in\n%s", want, text)
			}
		}
		if strings.Contains(text, "errors ") || strings.Contains(text, "unrecognised ") || len(s.Errors) != 0 || s.Unrecognised != 0 {
			t.Errorf("errors %d, unrecognised %d, or a note of them (none is drawn at 0):\n%s", len(s.Errors), s.Unrecognised, text)
		}
		if i, j := strings.Index(text, failedHook), strings.Index(text, "Without using any tool"); i < 0 || j < 0 || i > j {
			t.Errorf("the notice (%d) does not come before the prompt (%d)", i, j)
		}
	})
}
