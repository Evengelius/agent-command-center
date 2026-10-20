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
)

// TestLongConversation is plan 20's scenario E (the owner's section 12): a
// long conversation in the cards, eight exchanges of seven Read calls each,
// one failing, on deep Windows paths, and long answers. Each run folds with
// its failure counted; a path keeps its file name; B scrolls to the first
// prompt and back; a mouse selection copies the conversation's own text,
// without the cards' bars; and after each resize the zones keep plan 20's
// geometry, no row wider than the screen.
func TestLongConversation(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := scriptRig(t, claude.ProviderID)
		h := r.h
		clip := &fakeClipboard{}
		h.m.opt.Clipboard = clip
		now := testNow
		h.m.opt.Now = func() time.Time { return now }
		answer := strings.Repeat("A long answer that wraps over several rows of the card. ", 10)
		deep := `C:\work\clients\northwind\services\billing\internal\ledger\reconciliation\f%d%d.go`
		const exchanges = 8
		for x := range exchanges {
			r.emit(event.UserMessage, event.User{Text: fmt.Sprintf("Exchange %d: read the seven files.", x)})
			r.emit(event.TurnStarted, nil)
			r.emit(event.MessageCompleted, event.Completed{MessageID: fmt.Sprintf("n%d", x), Text: fmt.Sprintf("Reading the files of exchange %d.", x)})
			for i := range 7 {
				id := fmt.Sprintf("c%d%d", x, i)
				in, _ := json.Marshal(map[string]string{"file_path": fmt.Sprintf(deep, x, i)})
				r.emit(event.ToolStarted, event.Tool{ToolID: id, Name: "Read", Input: in})
				if i == 1 {
					r.emit(event.ToolCompleted, event.Tool{ToolID: id, IsError: true, Status: "error", Output: "File does not exist."})
				} else {
					r.emit(event.ToolCompleted, event.Tool{ToolID: id, Status: "completed", Output: "package ledger"})
				}
			}
			r.emit(event.MessageCompleted, event.Completed{MessageID: fmt.Sprintf("a%d", x), Text: fmt.Sprintf("Answer %d. %s", x, answer)})
			r.emit(event.TurnCompleted, event.TurnEnd{Outcome: "completed"})
		}
		h.settle()

		h.send(tea.WindowSizeMsg{Width: 156, Height: 44})
		f := h.frame()
		// Each run of seven folds its first four, the failed one counted;
		// the newest exchange's last calls show their file names.
		if !strings.Contains(f, "▸ 4 earlier tool events · 1 failed · Expand") {
			t.Errorf("no folded run with its failure:\n%s", f)
		}
		if !rowWith(f, "✓ Read", `\f76.go`) {
			t.Errorf("the last call's path lost its file name:\n%s", f)
		}
		if !strings.Contains(f, "Answer 7.") || strings.Contains(f, "Exchange 0:") {
			t.Fatalf("B does not follow the newest exchange:\n%s", f)
		}
		longGeometry(t, f, 156, 44)

		// B scrolls up to the first prompt, then down to the newest answer.
		for range 40 {
			if strings.Contains(h.frame(), "Exchange 0: read the seven files.") {
				break
			}
			h.draw(key(tea.KeyPgUp, 0))
		}
		if f := h.frame(); !strings.Contains(f, "▎ YOU") || !strings.Contains(f, "Exchange 0: read the seven files.") {
			t.Fatalf("PgUp never reached the first prompt:\n%s", f)
		}
		for range 40 {
			if strings.Contains(h.frame(), "Answer 7.") {
				break
			}
			h.draw(key(tea.KeyPgDown, 0))
		}
		if f := h.frame(); !strings.Contains(f, "Answer 7.") {
			t.Fatalf("PgDn never came back to the newest answer:\n%s", f)
		}

		// A selection over a narration row copies its text alone.
		const said = "Reading the files of exchange 7."
		x0, y0 := at(t, h, said, 0)
		dragCopy(t, h, x0, y0, x0+len(said)-1, y0)
		if got := clip.last(); got != said {
			t.Errorf("copied %q, want %q", got, said)
		}

		// Every size keeps the geometry; the newest answer stays in view.
		for _, size := range [][2]int{{80, 24}, {120, 40}, {200, 60}, {100, 30}, {156, 44}} {
			h.send(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
			f := h.frame()
			longGeometry(t, f, size[0], size[1])
			if !strings.Contains(f, "Answer 7.") && !strings.Contains(f, "wraps over several rows") {
				t.Errorf("%v: the newest answer out of view after the resize:\n%s", size, f)
			}
		}
	})
}

// longGeometry checks a frame against plan 20's layout: no row wider than
// the screen; from 100 columns A and B on the third row with their bottoms
// on one row, C and D full width under them, the footer last; under 100, B,
// C and D full width.
func longGeometry(t *testing.T, f string, w, h int) {
	t.Helper()
	rows := strings.Split(strings.TrimSuffix(f, "\n"), "\n")
	if len(rows) != h {
		t.Errorf("%dx%d: %d rows", w, h, len(rows))
	}
	type box struct{ top, bottom, left, right int }
	boxes := map[string]box{}
	for y, r := range rows {
		if lw := lipgloss.Width(r); lw > w {
			t.Errorf("%dx%d: row %d is %d wide", w, h, y, lw)
		}
		cells := []rune(r)
		for x := 0; x+3 < len(cells); x++ {
			if cells[x] != '╭' || cells[x+1] != '─' || cells[x+2] != ' ' {
				continue
			}
			z := string(cells[x+3])
			if _, seen := boxes[z]; seen || !strings.Contains("ABCD", z) {
				continue
			}
			right := x + 1
			for right < len(cells) && cells[right] != '╮' {
				right++
			}
			bottom := y + 1
			for bottom < len(rows) && (x >= len([]rune(rows[bottom])) || []rune(rows[bottom])[x] != '╰') {
				bottom++
			}
			boxes[z] = box{y, bottom, x, right}
		}
	}
	a, b, c, d := boxes["A"], boxes["B"], boxes["C"], boxes["D"]
	if w >= 100 {
		if _, ok := boxes["A"]; !ok || a.top != 2 || b.top != 2 || a.bottom != b.bottom || b.left != a.right+1 {
			t.Errorf("%dx%d: A %v and B %v not side by side on one row", w, h, a, b)
		}
	} else if _, ok := boxes["A"]; ok {
		t.Errorf("%dx%d: zone A boxed under 100 columns", w, h)
	}
	if c.top != b.bottom+1 || c.left != 0 || c.right != w-1 || d.top != c.bottom+1 || d.left != 0 || d.right != w-1 || d.bottom != h-2 {
		t.Errorf("%dx%d: C %v and D %v not full width under B %v, to the footer", w, h, c, d, b)
	}
}
