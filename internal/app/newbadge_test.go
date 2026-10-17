package app

import (
	"fmt"
	"strings"
	"testing"
	"testing/synctest"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/Evengelius/agent-command-center/internal/event"
)

// TestNewBadgeRow is plan 6, item 2, criteria 1 to 3: while B is held with new
// entries, "↓ N new" takes only the right end of B's last row, whose start
// stays drawn and copies as text, never the badge; moving the entry cursor
// does not leave the marked row on that row.
func TestNewBadgeRow(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r, clip, _ := selectRig(t)
		h := r.h
		var b strings.Builder
		for i := range 60 {
			fmt.Fprintf(&b, "line %02d of the long answer\n", i)
		}
		r.emit(event.MessageCompleted, event.Completed{MessageID: "m2", Text: strings.TrimSuffix(b.String(), "\n")})
		r.emit(event.ToolStarted, event.Tool{ToolID: "t2", Name: "Read"})
		r.emit(event.MessageCompleted, event.Completed{MessageID: "m3", Text: strings.TrimSuffix(b.String(), "\n")})
		h.settle()
		x, y := at(t, h, "of the long answer", 0)
		for range 15 {
			mouseAt(t, h, tea.MouseWheelMsg{X: x, Y: y, Button: tea.MouseWheelUp})
		}
		r.emit(event.MessageCompleted, event.Completed{MessageID: "m4", Text: "a newer answer"})
		h.settle()
		rows := strings.Split(plainText(h.m.View().Content), "\n")
		badge := -1
		for i, l := range rows {
			if strings.Contains(l, "↓ 1 new") {
				badge = i
			}
		}
		if badge < 0 {
			t.Fatalf("no badge:\n%s", strings.Join(rows, "\n"))
		}
		// Criterion 1: the row's start is drawn, cut before the badge.
		if row := rows[badge]; !strings.Contains(row, "line ") || strings.Contains(row, "…") && !strings.Contains(row, "answer…") {
			t.Fatalf("the badge row lost its text: %q", row)
		}
		// Criterion 3: a selection across the row copies its text only.
		cell := func(s string) int { return lipgloss.Width(rows[badge][:strings.Index(rows[badge], s)]) }
		dragCopy(t, h, cell("line "), badge, cell("↓ 1 new")+3, badge)
		if got := clip.last(); !strings.HasPrefix(got, "line ") || strings.Contains(got, "new") || strings.Contains(got, "↓") {
			t.Errorf("copied %q", got)
		}
		// Criterion 2: from the top, ↑ moves the cursor to the newest tool,
		// below the view; its row is not the badge row.
		for range 40 {
			mouseAt(t, h, tea.MouseWheelMsg{X: x, Y: y, Button: tea.MouseWheelUp})
		}
		h.draw(key(tea.KeyUp, 0))
		rows = strings.Split(plainText(h.m.View().Content), "\n")
		mark, badge := -1, -1
		for i, l := range rows {
			if strings.Contains(l, "Enter full view") {
				mark = i
			}
			if strings.Contains(l, "↓ 1 new") {
				badge = i
			}
		}
		if mark < 0 || badge < 0 || mark >= badge || strings.Contains(rows[mark], "new") {
			t.Errorf("marked row %d, badge row %d:\n%s", mark, badge, strings.Join(rows, "\n"))
		}
	})
}
