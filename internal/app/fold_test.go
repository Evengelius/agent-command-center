package app

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/Evengelius/agent-command-center/internal/event"
	"github.com/Evengelius/agent-command-center/internal/provider/claude"
)

// TestToolFoldKeys is plan 19, criterion 2.3, through the application: ↑
// from the empty input stops on a folded run's row, never on a call it
// folds; Enter expands it, ← on one of its calls folds it again, the cursor
// back on its row; a click on the row expands it and folds it.
func TestToolFoldKeys(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := scriptRig(t, claude.ProviderID)
		h := r.h
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
		// B's call rows, not zone A's tree rows ("├─ ✓ Read").
		calls := func(f string) int { return strings.Count(f, "▎  ✓ Read  ") + strings.Count(f, "▎▸ ✓ Read  ") } // in the CLI card (plan 20)
		f := h.frame()
		if !strings.Contains(f, "▸ 9 earlier tool events · Expand") || calls(f) != 3 {
			t.Fatalf("not folded:\n%s", f)
		}
		first := 2 // the prompt, then the run's first call

		// ↑ goes through the three calls in view, then stops on the run's row.
		for _, want := range []string{"f11.go", "f10.go", "f09.go"} {
			h.draw(key(tea.KeyUp, 0))
			if f := h.frame(); !rowWith(f, "▎▸ ✓ Read  C:\\work\\proj\\"+want, "Enter full view") {
				t.Fatalf("↑ to %s:\n%s", want, f)
			}
		}
		h.draw(key(tea.KeyUp, 0))
		if f := h.frame(); h.m.ui.Entry != first || !rowWith(f, "▎▸ ▸ 9 earlier tool events", "Enter expands · ↑↓ entries · Esc back") {
			t.Fatalf("↑ to the fold row: entry %d:\n%s", h.m.ui.Entry, f)
		}
		if f := h.frame(); !strings.Contains(f, "Enter or → expand · ↑↓ entries") {
			t.Errorf("the fold row's key hints:\n%s", f)
		}
		h.draw(key(tea.KeyUp, 0)) // nothing older to expand: it stays
		if h.m.ui.Entry != first {
			t.Errorf("↑ past the run: entry %d, want %d", h.m.ui.Entry, first)
		}

		// Enter expands it: its row reads Fold, every call shown.
		h.draw(key(tea.KeyEnter, 0))
		f = h.frame()
		if !strings.Contains(f, "▾ 9 earlier tool events · Fold") || calls(f) != 12 || h.m.ui.View != "" || h.m.ui.ViewEntry != 0 {
			t.Fatalf("Enter on the fold row:\n%s", f)
		}
		if !rowWith(f, "▎▸ ✓ Read  C:\\work\\proj\\f00.go", "← fold · ↑↓ entries") {
			t.Errorf("the cursor after Enter:\n%s", f)
		}
		// ↓ to a call it had folded; ← folds the run, the cursor on its row.
		h.draw(key(tea.KeyDown, 0))
		if f := h.frame(); !rowWith(f, "▎▸ ✓ Read  C:\\work\\proj\\f01.go", "Enter full view") {
			t.Fatalf("↓ in the expanded run:\n%s", f)
		}
		// → on a call opens nothing: no view, the cursor and the run as they
		// were (the owner's live review: → unfolds, only Enter opens).
		h.draw(key(tea.KeyRight, 0))
		if f := h.frame(); h.m.ui.View != "" || h.m.ui.ViewEntry != 0 || calls(f) != 12 || !rowWith(f, "▎▸ ✓ Read  C:\\work\\proj\\f01.go", "Enter full view") {
			t.Fatalf("→ on a call: view %q %d:\n%s", h.m.ui.View, h.m.ui.ViewEntry, f)
		}
		h.draw(key(tea.KeyLeft, 0))
		f = h.frame()
		if h.m.ui.Entry != first || calls(f) != 3 || !rowWith(f, "▎▸ ▸ 9 earlier tool events", "Enter expands") {
			t.Fatalf("← on a call: entry %d:\n%s", h.m.ui.Entry, f)
		}
		// → on the run's row expands it, as Enter does; ← folds it again.
		h.draw(key(tea.KeyRight, 0))
		if f := h.frame(); calls(f) != 12 || h.m.ui.ViewEntry != 0 || !strings.Contains(f, "▾ 9 earlier tool events · Fold") {
			t.Fatalf("→ on the run's row:\n%s", f)
		}
		h.draw(key(tea.KeyLeft, 0))
		if f := h.frame(); calls(f) != 3 || h.m.ui.Entry != first {
			t.Fatalf("← after →: entry %d:\n%s", h.m.ui.Entry, f)
		}
		// ← on a call of a folded run leaves the cursor for the input, as before.
		h.draw(key(tea.KeyDown, 0))
		h.draw(key(tea.KeyLeft, 0))
		if h.m.ui.Entry != 0 {
			t.Errorf("← on a folded run's call: entry %d, want the input", h.m.ui.Entry)
		}

		// At 80x24 the expanded run is taller than B: B stays where it was,
		// the run's row in view and the calls opening under it.
		h.send(tea.WindowSizeMsg{Width: 80, Height: 24})
		h.draw(key(tea.KeyEscape, 0))
		for range 4 {
			h.draw(key(tea.KeyUp, 0))
		}
		if h.m.ui.Entry != first {
			t.Fatalf("80x24, ↑ four times: entry %d, want the run's row", h.m.ui.Entry)
		}
		h.draw(key(tea.KeyEnter, 0))
		// (The marked call's hint takes its path's room at 80 columns.)
		if f := h.frame(); !strings.Contains(f, "▾ 9 earlier tool events") || !rowWith(f, "▎▸ ✓ Read", "Enter full view") || !strings.Contains(f, "f01.go") {
			t.Fatalf("80x24, expanded: the run's row or its first calls out of view:\n%s", f)
		}
		h.draw(key(tea.KeyLeft, 0))
		h.draw(key(tea.KeyEscape, 0))
		h.send(tea.WindowSizeMsg{Width: 160, Height: 50})

		// A click on the row expands it, a second one folds it; the clock
		// moves between them, so they are two clicks, not a double click.
		h.m.opt.Now = time.Now
		for _, want := range []string{"▾ 9 earlier tool events · Fold", "▸ 9 earlier tool events · Expand"} {
			time.Sleep(time.Second) // not a double click
			x, y := at(t, h, "9 earlier tool events", 0)
			mouseAt(t, h, click(x, y))
			mouseAt(t, h, tea.MouseReleaseMsg{X: x, Y: y, Button: tea.MouseLeft})
			if f := h.frame(); !strings.Contains(f, want) {
				t.Fatalf("click: want %q:\n%s", want, f)
			}
		}
	})
}
