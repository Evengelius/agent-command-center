package app

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/Evengelius/agent-command-center/internal/event"
	"github.com/Evengelius/agent-command-center/internal/provider/claude"
	"github.com/Evengelius/agent-command-center/internal/render"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// Plan 42 (brief amendment 43): the mouse in zone B. A click on a foldable
// item unfolds it or folds it again, a double click on an entry Enter shows
// whole opens its full view, and neither puts the entry cursor anywhere.

// mouseClick presses and releases the left button at x, y.
func mouseClick(t *testing.T, h *harness, x, y int) {
	t.Helper()
	mouseAt(t, h, click(x, y))
	mouseAt(t, h, tea.MouseReleaseMsg{X: x, Y: y, Button: tea.MouseLeft})
}

// lastOn is the cell the last occurrence of s starts on in the frame's row y.
func lastOn(t *testing.T, h *harness, y int, s string) int {
	t.Helper()
	l := strings.Split(h.frame(), "\n")[y]
	i := strings.LastIndex(l, s)
	if i < 0 {
		t.Fatalf("no %q on row %d: %q", s, y, l)
	}
	return lipgloss.Width(l[:i])
}

// noKeyHints fails when the frame shows the entry cursor or its hints.
func noKeyHints(t *testing.T, h *harness, when string) {
	t.Helper()
	f := h.frame()
	if h.m.ui.Entry != 0 || strings.Contains(f, "Enter full view") || strings.Contains(f, "← fold") || strings.Contains(f, "Esc back") {
		t.Errorf("%s: the entry cursor %d or its hints:\n%s", when, h.m.ui.Entry, f)
	}
}

// TestThinkingClicks is criteria 42.1 and 42.3 on a thinking phase (P12c's
// last, "Thought for 2.2s"): a click on its summary row's text, its time or
// its ▸ unfolds its details or folds them again, with no key hint; a double
// click shows it whole, its details as they were, and Esc returns with no
// cursor; a double click on a detail row shows it whole too.
func TestThinkingClicks(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := thinkStart(t, "p12c-block-claude-auto-boundary-edit.jsonl", claude.ProviderID)
		h := r.h
		h.draw(tea.WindowSizeMsg{Width: 160, Height: 50})
		r.until(27000)
		r.at(27000)
		last := thought(t, h.m.State(), 4)
		want := 0
		for i, e := range h.m.State().Conversation {
			if e.Seq == last.Seq {
				want = i + 1
			}
		}
		k := render.ThoughtKey(last)
		rows := h.m.screen.Conv.Rows
		later := func() { *r.now = r.now.Add(time.Second) } // apart: two clicks, not a double click
		open := func(when string, wantOpen bool) {
			t.Helper()
			f := h.frame()
			mark, n := "▸ ✓ Thought for 2.2s", rows
			if wantOpen {
				mark, n = "▾ ✓ Thought for 2.2s", rows+4 // its four details, a row each at 160 columns
			}
			if h.m.ui.Unfolded[k] != wantOpen || !strings.Contains(f, mark) || h.m.screen.Conv.Rows != n || h.m.ui.ViewEntry != 0 {
				t.Fatalf("%s: open %v, want %v; rows %d, want %d; view %d\n%s", when, h.m.ui.Unfolded[k], wantOpen, h.m.screen.Conv.Rows, n, h.m.ui.ViewEntry, f)
			}
			noKeyHints(t, h, when)
		}
		// The keyboard's cursor on the newest entry, its hint shown; B follows
		// its end.
		h.draw(key(tea.KeyUp, 0))
		if h.m.ui.Entry == 0 || !strings.Contains(h.frame(), "Enter full view") {
			t.Fatalf("↑: entry %d\n%s", h.m.ui.Entry, h.frame())
		}
		hold := h.m.ui.ConvHold

		// A double click: the phase whole, its details folded as they were,
		// the cursor gone; Esc returns to B as it was, with no hint.
		x, y := at(t, h, "Thought for 2.2s", 0)
		mouseClick(t, h, x+2, y)
		mouseClick(t, h, x+2, y)
		if f := h.frame(); h.m.ui.ViewEntry != want || !strings.Contains(f, "THINKING") || h.m.ui.Unfolded[k] || h.m.ui.Entry != 0 {
			t.Fatalf("a double click: view %d, want %d; open %v; entry %d\n%s", h.m.ui.ViewEntry, want, h.m.ui.Unfolded[k], h.m.ui.Entry, f)
		}
		h.draw(key(tea.KeyEscape, 0))
		if h.m.ui.ViewEntry != 0 || h.m.ui.ConvHold != hold {
			t.Fatalf("Esc: view %d; B held %v, was %v", h.m.ui.ViewEntry, h.m.ui.ConvHold, hold)
		}
		open("Esc after the double click", false)
		later()

		// Clicks: on its text, on its origin at the right, on its ▸.
		x, y = at(t, h, "Thought for 2.2s", 0)
		mouseClick(t, h, x+2, y)
		open("a click on its text", true)
		later()
		mouseClick(t, h, lastOn(t, h, y, "observed")+1, y)
		open("a click on its origin, at the right", false)
		later()
		xm, ym := at(t, h, "▸ ✓ Thought for 2.2s", 0)
		mouseClick(t, h, xm, ym)
		open("a click on its ▸", true)
		later()
		mouseClick(t, h, xm, ym)
		open("a second click on its ▾", false)
		later()

		// Its details open, a double click on one of their rows shows the
		// phase whole and leaves them open.
		mouseClick(t, h, x+2, y)
		open("a click to open the details again", true)
		later()
		xd, yd := at(t, h, "thinking tokens · exact", 0)
		mouseClick(t, h, xd+2, yd)
		mouseClick(t, h, xd+2, yd)
		if h.m.ui.ViewEntry != want || !h.m.ui.Unfolded[k] {
			t.Fatalf("a double click on a detail row: view %d, want %d; open %v\n%s", h.m.ui.ViewEntry, want, h.m.ui.Unfolded[k], h.frame())
		}
	})
}

// clickRig is a Claude Code session whose zone B holds every kind of row
// plan 42 meets: a message of the owner's longer than its fold, an Agent
// call, a Write call, an approval the owner answered (43.png's Write row),
// a review, the answer and a failure; the copies go to clip, the clock is
// now.
func clickRig(t *testing.T) (inputRig, *fakeClipboard, *time.Time) {
	t.Helper()
	r := scriptRig(t, claude.ProviderID)
	clip := &fakeClipboard{}
	r.h.m.opt.Clipboard = clip
	r.h.m.opt.Getenv = func(string) string { return "" }
	now := testNow
	r.h.m.opt.Now = func() time.Time { return now }
	r.h.send(tea.WindowSizeMsg{Width: 160, Height: 50})
	var long []string
	for i := range 40 {
		long = append(long, fmt.Sprintf("line %02d of a long message", i+1))
	}
	r.emit(event.UserMessage, event.User{Text: strings.Join(long, "\n")})
	r.emit(event.TurnStarted, nil)
	agentIn, _ := json.Marshal(map[string]string{"subagent_type": "general-purpose", "description": "List dir, write found.txt", "prompt": "list the files"})
	r.emit(event.ToolStarted, event.Tool{ToolID: "a1", Name: "Agent", Input: agentIn})
	r.emit(event.ToolCompleted, event.Tool{ToolID: "a1", Name: "Agent", Output: "alpha.txt and beta.md"})
	writeIn, _ := json.Marshal(map[string]string{"file_path": `C:\work\acc-demo\found.txt`, "content": "alpha.txt\nbeta.md"})
	r.emit(event.ToolStarted, event.Tool{ToolID: "w1", Name: "Write", Input: writeIn})
	r.emit(event.ToolCompleted, event.Tool{ToolID: "w1", Name: "Write"})
	notesIn, _ := json.Marshal(map[string]string{"file_path": `C:\work\acc-demo\notes.txt`, "content": "notes"})
	r.emit(event.ApprovalRequest, event.Approval{ID: "ap1", Tool: "Write", ToolID: "w2", Input: notesIn})
	r.h.settle()
	r.h.send(key(tea.KeyF2, 0)) // the card's focus, then allow once
	r.h.send(typed('a'))
	r.h.settle()
	r.emit(event.ApprovalResolved, event.Approval{ID: "ap1", Decision: "allow"}) // as the CLI acknowledges it
	r.emit(event.ReviewUpdated, event.Review{ReviewID: "rv1", Command: "Set-Content found.txt", Status: "approved", Risk: "medium", Authorization: "high", Rationale: "a write the owner asked for"})
	r.emit(event.MessageCompleted, event.Completed{MessageID: "m1", Text: "Found alpha.txt and beta.md; both names are in found.txt."})
	r.emit(event.TurnCompleted, event.TurnEnd{Outcome: "failed", Error: "the model stopped: overloaded"})
	r.h.settle()
	if f := r.h.frame(); !rowWith(f, "notes.txt", "allowed by you") {
		t.Fatalf("no answered approval's row:\n%s", f)
	}
	return r, clip, &now
}

// TestEntryClicks is criteria 42.2 to 42.5 on clickRig's rows: a click on
// one opens and unfolds nothing; a double click on a call, a review, a
// failure or the long message opens the view Enter opens, and Esc returns
// with no cursor; a double click on the answer selects a word, a triple
// click its line.
func TestEntryClicks(t *testing.T) {
	entryOf := func(h *harness, role, id string) int {
		for i, e := range h.m.State().Conversation {
			if e.Role == role && (id == "" || e.MessageID == id) {
				return i + 1
			}
		}
		t.Fatalf("no %s entry %q", role, id)
		return 0
	}
	targets := []struct {
		name, text, role, id, view string // view: a text the full view shows
	}{
		{"the Agent call", "List dir, write found.txt", "tool", "a1", "alpha.txt and beta.md"},
		{"the Write call", `C:\work\acc-demo\found.txt`, "tool", "w1", "Write"},
		{"the review", "a write the owner asked for", "review", "rv1", "auto-review approved"},
		{"the failure", "the model stopped: overloaded", "error", "", "the model stopped: overloaded"},
		{"the long message", "line 03 of a long message", "user", "", "line 40 of a long message"},
	}
	t.Run("a click opens nothing", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			r, _, now := clickRig(t)
			h := r.h
			others := []struct{ name, text, role, id, view string }{{"the answer", "both names are in", "", "", ""}, {"the answered approval", "notes.txt", "", "", ""}}
			for _, c := range append(targets[:len(targets):len(targets)], others...) {
				*now = now.Add(time.Second)
				x, y := at(t, h, c.text, 0)
				mouseClick(t, h, x+2, y)
				if h.m.ui.ViewEntry != 0 || h.m.ui.View != "" || len(h.m.ui.Unfolded) != 0 {
					t.Errorf("a click on %s: view %d %q, unfolded %v", c.name, h.m.ui.ViewEntry, h.m.ui.View, h.m.ui.Unfolded)
				}
				noKeyHints(t, h, "a click on "+c.name)
			}
		})
	})
	for _, c := range targets {
		t.Run("a double click on "+c.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				r, _, now := clickRig(t)
				h := r.h
				n := entryOf(h, c.role, c.id)
				// The view Enter opens from the entry cursor, then the double
				// click's: the same frame.
				for i := 0; i < 12 && h.m.ui.Entry != n; i++ {
					h.draw(key(tea.KeyUp, 0))
				}
				if h.m.ui.Entry != n {
					t.Fatalf("↑ never reached %s (entry %d): entry %d", c.name, n, h.m.ui.Entry)
				}
				h.draw(key(tea.KeyEnter, 0))
				byKey := h.frame()
				h.draw(key(tea.KeyEscape, 0)) // back, the cursor on the entry
				h.draw(key(tea.KeyEscape, 0)) // the cursor off
				*now = now.Add(time.Second)
				x, y := at(t, h, c.text, 0)
				mouseClick(t, h, x+2, y)
				mouseClick(t, h, x+2, y)
				f := h.frame()
				if h.m.ui.ViewEntry != n || !strings.Contains(f, "◀ back (Esc)") || !strings.Contains(f, c.view) || h.m.ui.Entry != 0 {
					t.Fatalf("view %d, want %d; entry %d\n%s", h.m.ui.ViewEntry, n, h.m.ui.Entry, f)
				}
				if f != byKey {
					t.Errorf("the double click's view differs from Enter's:\n%s\n---\n%s", f, byKey)
				}
				h.draw(key(tea.KeyEscape, 0))
				if h.m.ui.ViewEntry != 0 {
					t.Fatalf("Esc: view %d", h.m.ui.ViewEntry)
				}
				noKeyHints(t, h, "Esc after the double click")
			})
		})
	}
	t.Run("a double click on the answered approval", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			r, _, now := clickRig(t)
			h := r.h
			*now = now.Add(time.Second)
			x, y := at(t, h, "notes.txt", 0)
			mouseClick(t, h, x+2, y)
			mouseClick(t, h, x+2, y)
			f := h.frame()
			if h.m.ui.View != "ap1" || h.m.ui.ViewEntry != 0 || !rowWith(f, "◀ back (Esc)", "orchestrator · Write", "allowed by you") || h.m.ui.Entry != 0 {
				t.Fatalf("view %q, entry view %d, entry %d\n%s", h.m.ui.View, h.m.ui.ViewEntry, h.m.ui.Entry, f)
			}
			h.draw(key(tea.KeyEscape, 0))
			if h.m.ui.View != "" {
				t.Fatalf("Esc: view %q", h.m.ui.View)
			}
			noKeyHints(t, h, "Esc after the double click")
		})
	})
	t.Run("a double click on a pending approval", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			r, _, now := clickRig(t)
			h := r.h
			r.emit(event.TurnStarted, nil)
			in, _ := json.Marshal(map[string]string{"command": "Remove-Item scratch.txt"})
			r.emit(event.ApprovalRequest, event.Approval{ID: "ap2", Tool: "Bash", ToolID: "b2", Input: in})
			h.settle()
			*now = now.Add(time.Second)
			x, y := at(t, h, "Remove-Item scratch.txt", 0)
			mouseClick(t, h, x+2, y)
			mouseClick(t, h, x+2, y)
			f := h.frame()
			if h.m.ui.View != "ap2" || !strings.Contains(f, "◀ back (Esc)") || !strings.Contains(f, "Remove-Item scratch.txt") {
				t.Fatalf("view %q\n%s", h.m.ui.View, f)
			}
			if ap := h.m.State().Approvals["ap2"]; ap.Status != state.Pending {
				t.Errorf("the double click answered it: %q", ap.Status)
			}
		})
	})
	t.Run("the answer's words and lines", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			r, clip, now := clickRig(t)
			h := r.h
			*now = now.Add(time.Second)
			x, y := at(t, h, "both names", 0)
			mouseClick(t, h, x+1, y)
			mouseClick(t, h, x+1, y)
			if clip.last() != "both" || h.m.ui.ViewEntry != 0 {
				t.Errorf("a double click on the answer: copied %q, view %d", clip.last(), h.m.ui.ViewEntry)
			}
			mouseClick(t, h, x+1, y)
			if want := "Found alpha.txt and beta.md; both names are in found.txt."; clip.last() != want || h.m.ui.ViewEntry != 0 {
				t.Errorf("a triple click on the answer: copied %q, want %q; view %d", clip.last(), want, h.m.ui.ViewEntry)
			}
		})
	})
}

// TestFoldRowDoubleClick is criterion 42.4 on a run of calls (plan 19): a
// double click on its row toggles it once, opens nothing and copies
// nothing.
func TestFoldRowDoubleClick(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := scriptRig(t, claude.ProviderID)
		h := r.h
		clip := &fakeClipboard{}
		h.m.opt.Clipboard = clip
		h.m.opt.Getenv = func(string) string { return "" }
		now := testNow
		h.m.opt.Now = func() time.Time { return now }
		r.emit(event.UserMessage, event.User{Text: "read twelve files"})
		r.emit(event.TurnStarted, nil)
		for i := range 12 {
			in, _ := json.Marshal(map[string]string{"file_path": fmt.Sprintf(`C:\work\proj\f%02d.go`, i)})
			r.emit(event.ToolStarted, event.Tool{ToolID: fmt.Sprintf("c%02d", i), Name: "Read", Input: in})
			r.emit(event.ToolCompleted, event.Tool{ToolID: fmt.Sprintf("c%02d", i), Name: "Read"})
		}
		r.emit(event.MessageCompleted, event.Completed{MessageID: "m1", Text: "Read all twelve."})
		r.emit(event.TurnCompleted, event.TurnEnd{Outcome: "completed"})
		h.settle()
		h.send(tea.WindowSizeMsg{Width: 160, Height: 50})
		x, y := at(t, h, "9 earlier tool events", 0)
		mouseClick(t, h, x+2, y)
		mouseClick(t, h, x+2, y)
		if f := h.frame(); !strings.Contains(f, "▾ 9 earlier tool events · Fold") || h.m.ui.ViewEntry != 0 || h.m.ui.View != "" || clip.last() != "" {
			t.Fatalf("a double click on the run's row: view %d, copied %q\n%s", h.m.ui.ViewEntry, clip.last(), f)
		}
		noKeyHints(t, h, "a double click on the run's row")
	})
}
