package app

import (
	"strings"
	"testing"
	"testing/synctest"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/Evengelius/agent-command-center/internal/event"
)

// TestSessionStateMouse is plan 5, item 9, criterion 3: a drag from an
// answer into the failure card under it copies both, a blank line between;
// double-click selects a word of the card and triple-click a line, a cut
// stderr line whole.
func TestSessionStateMouse(t *testing.T) {
	cut := strings.Repeat("a long stderr line ", 12) + "TAIL"
	rig := func(t *testing.T) (*harness, *fakeClipboard, *time.Time) {
		r, clip := copyRig(t, nil)
		now := testNow
		r.h.m.opt.Now = func() time.Time { return now }
		r.emit(event.MessageCompleted, event.Completed{MessageID: "m1", Text: "an answer"})
		for _, l := range []string{"first", "second", cut} {
			r.emit(event.Stderr, event.Line{Text: l})
		}
		r.emit(event.SessionEnded, event.Ended{Code: 3, HasCode: true})
		r.h.settle()
		return r.h, clip, &now
	}
	clicks := func(h *harness, x, y, n int) {
		for i := 0; i < n; i++ {
			mouseAt(t, h, click(x, y))
			mouseAt(t, h, tea.MouseReleaseMsg{X: x, Y: y, Button: tea.MouseLeft})
		}
	}
	t.Run("drag from an answer into the card", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			h, clip, _ := rig(t)
			x0, y0 := at(t, h, "an answer", 0)
			_, y1 := at(t, h, "a long stderr", 0)
			dragCopy(t, h, x0, y0, 200, y1)
			want := "an answer\n\nThe session ended unexpectedly\nthe CLI exited unexpectedly\nexit code 3\nstderr:\nfirst\nsecond\n" + cut
			if clip.last() != want {
				t.Errorf("copied\n%q\nwant\n%q", clip.last(), want)
			}
		})
	})
	t.Run("double click a word, triple click a cut line", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			h, clip, now := rig(t)
			x, y := at(t, h, "exited", 0)
			clicks(h, x+1, y, 2)
			if clip.last() != "exited" {
				t.Errorf("double click copied %q", clip.last())
			}
			*now = testNow.Add(time.Minute)
			x, y = at(t, h, "a long stderr", 0)
			clicks(h, x+3, y, 3)
			if clip.last() != cut {
				t.Errorf("triple click copied %q, want %q", clip.last(), cut)
			}
		})
	})
}
