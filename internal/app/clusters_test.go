package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/synctest"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/Evengelius/agent-command-center/internal/event"
	"github.com/Evengelius/agent-command-center/internal/render"
)

// clusterText holds a cluster of several code points of each kind plan 49
// keeps whole (design D11): a heart with VS16, a skin tone, a family of
// three, a flag, a letter and its mark, a keycap, a tag sequence, Hangul
// jamo, an Indic conjunct; then CJK.
const clusterText = "\u2764\uFE0F \U0001F44D\U0001F3FD \U0001F468\u200D\U0001F469\u200D\U0001F466 \U0001F1EB\U0001F1F7 " +
	"e\u0301 1\uFE0F\u20E3 \U0001F3F4\U000E0067\U000E0062\U000E0073\U000E0063\U000E0074\U000E007F " +
	"\u1100\u1161\u11A8 \u0915\u094D\u0937 \u4E2D\u6587"

// TestClusterFrames is plan 49's criterion 49.12 (design D11): a message, a
// tool row, a file and a commit message holding clusterText, every row of
// every frame exactly the screen's width by lipgloss.Width, at 200x60 and
// 120x40.
func TestClusterFrames(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r, _, root := sourceRig(t)
		h := r.h
		var bad []string
		look := func(step string) {
			t.Helper()
			for _, size := range [][2]int{{120, 40}, {200, 60}} {
				frames := 0
				h.check = func(frame string) {
					frames++
					for i, l := range strings.Split(frame, "\n") {
						if n := lipgloss.Width(l); n != size[0] {
							bad = append(bad, fmt.Sprintf("%s, %dx%d, row %d: %d columns: %s", step, size[0], size[1], i+1, n, plainText(l)))
						}
					}
				}
				h.send(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
				if frames == 0 {
					t.Fatalf("%s: no frame drawn", step)
				}
			}
			h.check = func(string) {}
		}

		r.emit(event.TurnStarted, nil)
		r.emit(event.ToolStarted, event.Tool{ToolID: "t1", Name: "Bash", Input: toolInput(t, map[string]any{"command": "echo " + clusterText})})
		r.emit(event.ToolCompleted, event.Tool{ToolID: "t1", Name: "Bash", Output: clusterText})
		r.emit(event.MessageCompleted, event.Completed{MessageID: "m1", Text: clusterText + "\n\n" + strings.Repeat(clusterText+" ", 6)})
		r.emit(event.TurnCompleted, event.TurnEnd{Outcome: "completed", Subtype: "success"})
		h.settle()
		if !strings.Contains(plainText(h.frame()), "\U0001F1EB\U0001F1F7") {
			t.Fatalf("the message is not drawn:\n%s", plainText(h.frame()))
		}
		look("the conversation and the tool row")

		path := filepath.Join(root, "clusters.txt")
		body := clusterText + "\n\t" + clusterText + "\n" + strings.Repeat(clusterText+" ", 8) + "\n"
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		openPath(h, path)
		if !render.WindowOpen(h.m.ui) {
			t.Fatal("the file is not open")
		}
		look("the file")
		h.send(key(tea.KeyEscape, 0))
		if render.WindowOpen(h.m.ui) {
			t.Fatal("Esc left the file open")
		}

		showSource(t, h)
		refresh(h)
		clickOn(h, render.SourcePanel, render.ItemMsg)
		h.send(tea.PasteMsg{Content: clusterText})
		if got := h.m.msg.Text(); got != clusterText {
			t.Fatalf("the commit message holds %q", got)
		}
		look("the commit message")

		if len(bad) > 0 {
			t.Errorf("%d rows not the screen's width:\n%s", len(bad), strings.Join(bad, "\n"))
		}
	})
}
