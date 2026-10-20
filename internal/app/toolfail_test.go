package app

import (
	"strings"
	"testing"
	"testing/synctest"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/Evengelius/agent-command-center/internal/event"
)

// TestFailedToolRow is plan 5, item 6, criteria 4 and 5, through the
// application: the failed call's row in B copies its name and its error as
// two lines, a double click selects a word of the error, and the entry cursor
// opens item 4's viewer on the call.
func TestFailedToolRow(t *testing.T) {
	const errLine = "exit status 1: FAIL pkg/a"
	rig := func(t *testing.T) (inputRig, *fakeClipboard, *time.Time) {
		r, clip, now := selectRig(t)
		r.emit(event.ToolCompleted, event.Tool{ToolID: "t1", IsError: true, Output: errLine + "\nmore output"})
		r.h.settle()
		if f := r.h.frame(); !rowWith(f, "✗ Bash · "+errLine, "observed") {
			t.Fatalf("no failed row:\n%s", f)
		}
		return r, clip, now
	}
	t.Run("copy", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			r, clip, _ := rig(t)
			h := r.h
			x0, y := at(t, h, "Bash · exit", 0) // B's row, not A's
			x1, _ := at(t, h, "pkg/a", 0)
			dragCopy(t, h, x0-4, y, x1+5, y) // from the "✗" to past the error
			if want := "Bash\n" + errLine; clip.last() != want {
				t.Errorf("copied %q, want %q", clip.last(), want)
			}
		})
	})
	t.Run("double click", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			r, clip, _ := rig(t)
			h := r.h
			x, y := at(t, h, "status", 0)
			for range 2 {
				mouseAt(t, h, click(x+2, y))
				mouseAt(t, h, tea.MouseReleaseMsg{X: x + 2, Y: y, Button: tea.MouseLeft})
			}
			if clip.last() != "status" {
				t.Errorf("copied %q, want %q", clip.last(), "status")
			}
		})
	})
	t.Run("cursor and viewer", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			r, _, _ := rig(t)
			h := r.h
			h.draw(key(tea.KeyUp, 0))
			if f := h.frame(); !rowWith(f, "▸ ✗ Bash · "+errLine, "Enter full view") {
				t.Fatalf("↑: the failed row is not marked:\n%s", f)
			}
			h.draw(key(tea.KeyEnter, 0))
			f := h.frame()
			for _, want := range []string{"◀ back (Esc)", "✗ Bash", "failed ·", errLine, "more output"} {
				if !strings.Contains(f, want) {
					t.Errorf("the viewer lacks %q:\n%s", want, f)
				}
			}
		})
	})
}
