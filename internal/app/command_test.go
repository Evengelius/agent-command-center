package app

import (
	"encoding/json"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/Evengelius/agent-command-center/internal/event"
	"github.com/Evengelius/agent-command-center/internal/render"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// Plan 72's keys and mouse on zone B's commands (K1, K2, M1 to M6) and the
// full view's copy by keyboard (C3), on a scripted Codex session holding the
// approved frames' commands, from synthetic events labelled so.

// psLauncher is the launcher Codex runs a command with on Windows, as its
// command field writes it.
const psLauncher = `"C:\\WINDOWS\\System32\\WindowsPowerShell\\v1.0\\powershell.exe"`

// longScript is the reference frames' long command.
const longScript = `$files = Get-ChildItem -Recurse -Filter *.go -File; $todos = $files | Select-String -Pattern 'TODO|FIXME' | Sort-Object Path, LineNumber; $todos | ForEach-Object { "- $($_.Path):$($_.LineNumber) $($_.Line.Trim())" } | Set-Content -LiteralPath 'TODO.md' -Encoding utf8; '{0} items in {1} files' -f $todos.Count, ($todos | Select-Object -Unique Path).Count`

// invocation is a script as Codex's command field sends it: in the
// launcher's -Command, single-quoted, POSIX's way (plan 72, V1).
func invocation(script string) string {
	return psLauncher + " -Command '" + strings.ReplaceAll(script, "'", `'"'"'`) + "'"
}

// commandEvents are the frames' Codex commands: one short, one reviewed by
// Codex's auto-review, the long one asking for the owner's approval.
func commandEvents(t *testing.T) []event.Event {
	t.Helper()
	item := func(id, script string) json.RawMessage {
		raw, _ := json.Marshal(map[string]any{"type": "commandExecution", "id": id, "command": invocation(script), "cwd": `C:\work\acc-demo`, "status": "inProgress"})
		return raw
	}
	req, _ := json.Marshal(map[string]any{"threadId": "th1", "turnId": "tu1", "itemId": "c3", "reason": "Create TODO.md from the TODO and FIXME comments of the Go files",
		"command": invocation(longScript), "cwd": `C:\work\acc-demo`})
	return []event.Event{
		{Kind: event.UserMessage, Payload: event.User{Text: "Collect the TODO and FIXME comments of the Go sources into TODO.md."}},
		{Kind: event.TurnStarted},
		{Kind: event.ToolStarted, Payload: event.Tool{ToolID: "c1", Name: "commandExecution", Input: item("c1", "git status --short")}},
		{Kind: event.ToolCompleted, Payload: event.Tool{ToolID: "c1", Name: "commandExecution", Output: " M a.go"}},
		{Kind: event.ToolStarted, Payload: event.Tool{ToolID: "c2", Name: "commandExecution", Input: item("c2", "Get-ChildItem -Recurse -Filter *.go -File | Measure-Object")}},
		{Kind: event.ReviewUpdated, Item: "c2", Payload: event.Review{ReviewID: "rv1", Status: "approved", Risk: "low", Rationale: "Counts the Go files of the project and reads nothing else; a read-only check of the working tree."}},
		{Kind: event.ToolCompleted, Payload: event.Tool{ToolID: "c2", Name: "commandExecution", Output: "Count : 9"}},
		{Kind: event.ToolStarted, Payload: event.Tool{ToolID: "c3", Name: "commandExecution", Input: item("c3", longScript)}},
		{Kind: event.ApprovalRequest, Payload: event.Approval{ID: "r3", Tool: "commandExecution", ToolID: "c3", Input: req, Summary: "Create TODO.md from the TODO and FIXME comments of the Go files"}},
	}
}

// commandRig is a scripted Codex session at 120x40 whose zone B holds the
// frames' commands, the long one allowed through its card and completed;
// copies go to clip, the clock is now.
func commandRig(t *testing.T) (inputRig, *fakeClipboard, *time.Time) {
	t.Helper()
	r := scriptRig(t, "codex")
	clip := &fakeClipboard{}
	r.h.m.opt.Clipboard = clip
	r.h.m.opt.Getenv = func(string) string { return "" }
	now := testNow
	r.h.m.opt.Now = func() time.Time { return now }
	r.h.send(tea.WindowSizeMsg{Width: 120, Height: 40})
	for _, ev := range commandEvents(t) {
		ev.Provider, ev.Origin = "codex", event.Observed
		r.script.events <- ev
	}
	r.h.settle()
	if f := r.h.frame(); !strings.Contains(f, "! APPROVAL · orchestrator · PowerShell") {
		t.Fatalf("no card:\n%s", f)
	}
	r.h.send(key(tea.KeyF2, 0))
	r.h.send(typed('a'))
	r.emit(event.ApprovalResolved, event.Approval{ID: "r3", Decision: event.Allow}) // as Codex acknowledges it
	r.emit(event.ToolCompleted, event.Tool{ToolID: "c3", Name: "commandExecution", Output: "14 items in 9 files", ExitCode: 0, HasExitCode: true})
	r.h.settle()
	if f := r.h.frame(); !rowWith(f, "▸ ✓ PowerShell", "allowed by you · ") || !strings.Contains(f, "… 7 more lines") {
		t.Fatalf("no folded block:\n%s", f)
	}
	return r, clip, &now
}

// TestCommandKeys is K1, K2 and criterion 72.6: on a block the row's hint and
// the footer's name → and ←; → unfolds, ← folds, then leaves the cursor;
// Enter opens the full view, Esc returns, the fold kept; ↑ stops once per
// operation.
func TestCommandKeys(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r, _, _ := commandRig(t)
		h := r.h
		key3 := render.CommandKey("c3")
		h.draw(key(tea.KeyUp, 0))
		f := h.frame()
		if !rowWith(f, "PowerShell", "→ unfold · Enter full view · ↑↓ entries · Esc back") || !strings.Contains(f, "→ unfold · Enter full view · ↑↓ entries · Esc back to the input · F1 keys") {
			t.Fatalf("↑ to the block: its hints\n%s", f)
		}
		h.draw(key(tea.KeyRight, 0))
		if f := h.frame(); !h.m.ui.Unfolded[key3] || !strings.Contains(f, "($todos | Select-Object -Unique Path).Count") || !rowWith(f, "▾ ✓ PowerShell", "← fold · Enter full view") || !strings.Contains(f, "← fold · Enter full view · ↑↓ entries · Esc back to the input") {
			t.Fatalf("→: unfolded %v\n%s", h.m.ui.Unfolded[key3], f)
		}
		entry := h.m.ui.Entry
		h.draw(key(tea.KeyEnter, 0))
		if h.m.ui.ViewEntry != entry || !strings.Contains(h.frame(), "COMMAND, laid out for reading") {
			t.Fatalf("Enter: view %d, want %d\n%s", h.m.ui.ViewEntry, entry, h.frame())
		}
		h.draw(key(tea.KeyEscape, 0))
		if h.m.ui.ViewEntry != 0 || h.m.ui.Entry != entry || !h.m.ui.Unfolded[key3] {
			t.Fatalf("Esc: view %d, entry %d, unfolded %v", h.m.ui.ViewEntry, h.m.ui.Entry, h.m.ui.Unfolded[key3])
		}
		h.draw(key(tea.KeyLeft, 0))
		if h.m.ui.Unfolded[key3] || h.m.ui.Entry != entry || !strings.Contains(h.frame(), "… 7 more lines") {
			t.Fatalf("←: unfolded %v, entry %d\n%s", h.m.ui.Unfolded[key3], h.m.ui.Entry, h.frame())
		}
		h.draw(key(tea.KeyLeft, 0))
		if h.m.ui.Entry != 0 {
			t.Fatalf("← on a folded block in no run: the cursor stays on %d", h.m.ui.Entry)
		}
		// One stop per operation: the long command, the reviewed one, the
		// short one, no stop for the request or the review drawn on them.
		s := h.m.State()
		var stops []string
		h.draw(key(tea.KeyUp, 0))
		for range 3 {
			e := s.Conversation[h.m.ui.Entry-1]
			stops = append(stops, e.Role+" "+e.MessageID)
			h.draw(key(tea.KeyUp, 0))
		}
		if want := "tool c3,tool c2,tool c1"; strings.Join(stops, ",") != want {
			t.Errorf("↑ stops %q, want %q", strings.Join(stops, ","), want)
		}
	})
}

// TestCommandCopyKey is C3 and criterion 72.13's key: y in a command's full
// view copies the command as Codex sent it, byte for byte, with the copy's
// toast; with the native clipboard failing, OSC 52's; in a view with no
// command, nothing.
func TestCommandCopyKey(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r, clip, _ := commandRig(t)
		h := r.h
		h.draw(key(tea.KeyUp, 0))
		h.draw(key(tea.KeyEnter, 0))
		if !strings.Contains(h.frame(), "y copy the command · ") {
			t.Fatalf("the full view's hints lack y:\n%s", h.frame())
		}
		h.draw(typed('y'))
		if got, want := clip.last(), invocation(longScript); got != want {
			t.Fatalf("y copied %q, want %q", got, want)
		}
		if h.m.ui.Toast != "copied 1 line · native" || h.m.ui.ViewEntry == 0 {
			t.Errorf("after y: toast %q, view %d", h.m.ui.Toast, h.m.ui.ViewEntry)
		}
		clip.mu.Lock()
		clip.fail = true
		clip.mu.Unlock()
		h.draw(typed('y'))
		if h.m.ui.Toast != "copied · OSC 52 (terminal must allow it)" {
			t.Errorf("the native clipboard failing: toast %q", h.m.ui.Toast)
		}
		// The owner's message's full view: no command, y copies nothing.
		clip.mu.Lock()
		clip.fail, clip.writes = false, nil
		clip.mu.Unlock()
		h.draw(key(tea.KeyEscape, 0))
		h.m.ui.Entry = 0
		for i, e := range h.m.State().Conversation {
			if e.Role == "user" {
				h.m.openView("", i+1)
			}
		}
		h.draw(typed('y'))
		if clip.last() != "" || strings.Contains(h.frame(), "y copy the command") {
			t.Errorf("y in a message's view copied %q", clip.last())
		}
	})
}

// TestCommandClicks is M1 to M6 and criteria 72.7 and 72.8: a click on a
// folded block's row or its counter unfolds it, on an unfolded block's row
// folds it, with the focus anywhere and no cursor; a press on a command's
// line toggles nothing; a double click opens the full view, the fold as it
// was; a click on a card neither folds nor answers it; with F12's capture
// off, no mouse reporting, and the keys do it all.
func TestCommandClicks(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r, _, now := commandRig(t)
		h := r.h
		key3 := render.CommandKey("c3")
		later := func() { *now = now.Add(time.Second) } // apart: two clicks, not a double click
		open := func(when string, want bool) {
			t.Helper()
			if h.m.ui.Unfolded[key3] != want || h.m.ui.ViewEntry != 0 {
				t.Fatalf("%s: unfolded %v, want %v; view %d\n%s", when, h.m.ui.Unfolded[key3], want, h.m.ui.ViewEntry, h.frame())
			}
			noKeyHints(t, h, when)
		}
		// The focus in zone A first: the click gives B the focus (M3).
		h.m.ui.Focus = "A"
		x, y := at(t, h, "▸ ✓ PowerShell", 1)
		mouseClick(t, h, x+6, y)
		open("a click on the folded block's row, the focus in A", true)
		if h.m.ui.Focus != "B" {
			t.Errorf("the focus %q after the click", h.m.ui.Focus)
		}
		later()
		x, y = at(t, h, "▾ ✓ PowerShell", 0)
		mouseClick(t, h, x, y)
		open("a click on the unfolded block's row", false)
		later()
		x, y = at(t, h, "… 7 more lines", 0)
		mouseClick(t, h, x+2, y)
		open("a click on its counter", true)
		later()
		x, y = at(t, h, "| Sort-Object Path, LineNumber;", 0)
		mouseClick(t, h, x+4, y)
		open("a press on a command line", true)
		later()
		// A double click on a command line: the full view, the fold as it was.
		mouseClick(t, h, x+4, y)
		mouseClick(t, h, x+4, y)
		if h.m.ui.ViewEntry == 0 || !h.m.ui.Unfolded[key3] || !strings.Contains(h.frame(), "COMMAND, laid out for reading") {
			t.Fatalf("a double click on a command line: view %d, unfolded %v\n%s", h.m.ui.ViewEntry, h.m.ui.Unfolded[key3], h.frame())
		}
		h.draw(key(tea.KeyEscape, 0))
		later()
		x, y = at(t, h, "▾ ✓ PowerShell", 0)
		mouseClick(t, h, x, y)
		mouseClick(t, h, x, y)
		if h.m.ui.ViewEntry == 0 || !h.m.ui.Unfolded[key3] {
			t.Fatalf("a double click on the row: view %d, unfolded %v", h.m.ui.ViewEntry, h.m.ui.Unfolded[key3])
		}
		h.draw(key(tea.KeyEscape, 0))
		later()
		// F12: no mouse reporting; the keys still fold and unfold (72.8).
		h.draw(key(tea.KeyF12, 0))
		if v := h.m.View(); !h.m.ui.MouseOff || v.MouseMode != tea.MouseModeNone {
			t.Fatalf("F12: capture off %v, mouse mode %v", h.m.ui.MouseOff, v.MouseMode)
		}
		h.draw(key(tea.KeyUp, 0))
		h.draw(key(tea.KeyLeft, 0))
		if h.m.ui.Unfolded[key3] {
			t.Errorf("F12 off: ← did not fold")
		}
		h.draw(key(tea.KeyRight, 0))
		if !h.m.ui.Unfolded[key3] {
			t.Errorf("F12 off: → did not unfold")
		}
	})
}

// TestCommandMocks is criterion 72.17 on the Human Evaluation's mock
// sessions, real recordings with plan 72's lines put in (testdata/mock,
// codex-commands.jsonl on P12x, claude-commands.jsonl on P2): each replayed
// through the dashboard draws the frames' rows; answering Codex's waiting
// request puts the answer on its call's row.
func TestCommandMocks(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := mockDash(t, "codex-commands.jsonl")
		h.send(tea.WindowSizeMsg{Width: 200, Height: 150})
		f := h.frame()
		for _, want := range [][]string{
			{"✓ PowerShell", "git status --short"},
			{"▸ ✓ PowerShell", "approved by auto-review · "},
			{"◈ risk low: Creates the explicitly requested hello.txt"},
			{"▸ ✓ PowerShell", "allowed by you · "},
			{"$files = Get-ChildItem -Recurse -Filter *.go -File;"},
			{"… 3 more lines"},
			{"RESULT"},
			{"! APPROVAL · orchestrator · PowerShell"},
			{"$counts = Get-ChildItem -Recurse -Filter *.go -File"},
			{"reason  Write TODO-COUNT.md"},
			{"▐ F2 to answer ▌"},
		} {
			if !rowWith(f, want...) {
				t.Errorf("codex-commands: no row of %q:\n%s", want, f)
			}
		}
		if strings.Contains(f, "commandExecution  ") && strings.Contains(f, "orchestrator · commandExecution") {
			t.Errorf("a request's row or Codex's item type in zone B:\n%s", f)
		}
		h.send(key(tea.KeyF2, 0))
		h.send(typed('a'))
		if f := h.frame(); strings.Contains(f, "! APPROVAL · orchestrator · PowerShell") || !rowWith(f, "PowerShell", "allowed by you · running") {
			t.Errorf("the mock's request answered:\n%s", f)
		}
		c := mockDash(t, "claude-commands.jsonl")
		c.send(tea.WindowSizeMsg{Width: 200, Height: 150})
		f = c.frame()
		for _, want := range [][]string{
			{"✗ Bash", "go test ./...", "failed "},
			{"--- FAIL: TestMarkdownCacheBytes (0.65s)"},
			{"✗ Bash", "denied by you "},
			{"&& git pull --rebase origin main"},
			{"✓ Write", "allowed by you · "},
		} {
			if !rowWith(f, want...) {
				t.Errorf("claude-commands: no row of %q:\n%s", want, f)
			}
		}
		if strings.Contains(f, "doesn't want") || strings.Contains(f, "orchestrator · Bash") {
			t.Errorf("claude-commands: a denial's output or a request's row:\n%s", f)
		}
	})
}

// TestCommandColoursKept is P3 in the application: the colours made off the
// event loop reach the frame, the same text drawn coloured, and nothing is
// asked twice.
func TestCommandColoursKept(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r, _, _ := commandRig(t)
		h := r.h
		coloured := h.m.View().Content
		kept := h.m.ui.Colors
		h.m.ui.Colors = render.NewCmdColors() // none made: the commands plain
		plain := h.m.View().Content
		h.m.ui.Colors = kept
		if coloured == plain || plainText(coloured) != plainText(plain) {
			t.Errorf("the colours made: the frame coloured %v, its text the same %v", coloured != plain, plainText(coloured) == plainText(plain))
		}
		if ws := kept.Wants(); len(ws) != 0 {
			t.Errorf("a coloured frame asked again for %d commands", len(ws))
		}
	})
}

// TestCommandCardClicks is M4: a click on a card's command answers nothing
// and folds nothing; the card's keys answer it, as before.
func TestCommandCardClicks(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := scriptRig(t, "codex")
		h := r.h
		h.send(tea.WindowSizeMsg{Width: 120, Height: 40})
		for _, ev := range commandEvents(t) {
			ev.Provider, ev.Origin = "codex", event.Observed
			r.script.events <- ev
		}
		h.settle()
		x, y := at(t, h, "| Sort-Object Path, LineNumber;", 0)
		mouseClick(t, h, x+4, y)
		ap := h.m.State().Approvals["r3"]
		if ap.Status != state.Pending || len(h.m.ui.Unfolded) != 0 {
			t.Fatalf("a click on the card: status %q, folds %v", ap.Status, h.m.ui.Unfolded)
		}
		if _, answers, _ := r.script.calls(); len(answers) != 0 {
			t.Fatalf("a click answered: %v", answers)
		}
	})
}
