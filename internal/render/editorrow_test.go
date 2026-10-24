package render

import (
	"fmt"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"

	"github.com/Evengelius/agent-command-center/internal/event"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// editorTail draws the dashboard and returns zone B's last four inner rows
// as text (without the frame's cells and its padding), the same rows as
// lines, and the cell the inner text starts at.
func editorTail(t *testing.T, s *state.State, ui UI) (inner []string, ls []line, off int) {
	t.Helper()
	sc := Draw(s, ui)
	rows := strings.Split(strings.TrimSuffix(texts(sc.lines), "\n"), "\n")
	b, ok := zoneBoxes(rows)["B"]
	if !ok {
		t.Fatalf("no zone B box:\n%s", strings.Join(rows, "\n"))
	}
	for y := b.bottom - 4; y < b.bottom; y++ {
		cells := []rune(rows[y])
		inner = append(inner, string(cells[b.left+2:b.right-1]))
		ls = append(ls, sc.lines[y])
	}
	return inner, ls, b.left + 2
}

// spanFrom is the first span of l holding s that starts at cell from or
// after it: zone A shares the row left of B.
func spanFrom(l line, from int, s string) (span, bool) {
	cell := 0
	for _, x := range l {
		if cell >= from && strings.Contains(x.text, s) {
			return x, true
		}
		cell += textWidth(x.text)
	}
	return span{}, false
}

// groundCells says, cell by cell, whether a row's span lies on the tool
// rows' ground.
func groundCells(l line) []bool {
	var out []bool
	for _, x := range l {
		on := false
		if x.st != nil {
			_, none := x.st.GetBackground().(lipgloss.NoColor)
			on = !none
		}
		for range textWidth(x.text) {
			out = append(out, on)
		}
	}
	return out
}

// quiet is the quota mock's end state with nothing to say: its notices
// cleared, the turn idle.
func quiet(t *testing.T) *state.State {
	t.Helper()
	s, _ := stampedMock(t, "claude-quota-windows.jsonl", appClock)
	s.Turn.Notice, s.Session.Notice = "", ""
	s.Turn.Phase = state.Idle
	return s
}

// highUltracode reports High and Ultracode in force, the owner's image's
// pill (plan 29), as a settings read would (plan 34).
func highUltracode(s *state.State) *state.State {
	s.Models.InForce = event.Settings{Model: s.Session.Model, EffortKnown: true, Effort: "high",
		UltracodeKnown: true, Ultracode: true, UltracodeAvailable: true}
	return s
}

// TestEditorRow is plan 29's criterion 29.1 (brief amendment 29, the owner's
// image of the input editor): under the draft a separator, then the row
// with the "+" and the model pill on the ground, the middle text when the
// status line had something to say, and "↯ Auto" at the right end. Since
// plan 34 the pill's effort and Ultracode are the ones in force: the
// owner's image's "High · Ultracode" once a read reports them, the model
// alone before. Since plan 39 the right end is the mode pill, the mode in
// force (here Manual), on the ground as the model pill, and one cell.
func TestEditorRow(t *testing.T) {
	defer UsePalette(pickTrue)
	now := appClock
	bare := quiet(t)
	model := modelName(bare.Session.Model)
	if model == "" || model == bare.Session.Model || strings.Contains(model, "claude") {
		t.Fatalf("the quota mock's model %q names %q", bare.Session.Model, model)
	}
	const modePill = " ✋ Manual "
	bareRows, _, _ := editorTail(t, bare, UI{Width: 120, Height: 40, Now: now, Focus: "B"})
	if head := "  +     " + model + " "; !strings.HasPrefix(bareRows[3], head) || strings.TrimSpace(bareRows[3][len(head):]) != strings.TrimSpace(modePill) {
		t.Errorf("no settings read: the row %q, want %q, spaces, %q", bareRows[3], head, modePill)
	}
	idle := highUltracode(quiet(t))
	pill := " " + model + " High · Ultracode "
	for _, size := range [][2]int{{80, 24}, {120, 40}, {208, 59}} {
		ui := UI{Width: size[0], Height: size[1], Now: now, Focus: "B"}
		rows, ls, off := editorTail(t, idle, ui)
		if strings.Trim(strings.TrimSpace(rows[0]), "─") != "" || !strings.HasPrefix(rows[1], ">") || !strings.Contains(rows[1], "Ask Mock data anything") {
			t.Errorf("%v: the editor's rule and prompt rows %q, %q", size, rows[0], rows[1])
		}
		if strings.Trim(strings.TrimSpace(rows[2]), "─") != "" {
			t.Errorf("%v: no separator under the draft: %q", size, rows[2])
		} else if x, ok := spanFrom(ls[2], off, "───"); !ok || x.st != stFrame {
			t.Errorf("%v: the separator styled %v, want stFrame", size, x.st)
		}
		row := rows[3]
		head := "  +    " + pill
		if !strings.HasPrefix(row, head) || !strings.HasSuffix(row, modePill+"  ") || strings.TrimSpace(row[len(head):]) != strings.TrimSpace(modePill) {
			t.Errorf("%v: the row %q, want %q, spaces, %q, a cell and the margin", size, row, head, modePill)
		}
		// The "+", the pill and the mode pill on the ground, the margins, the
		// gaps and the cell after the mode pill not.
		g := groundCells(ls[3])
		end := textWidth(strings.TrimRight(row, " ")) + 1 // the mode pill's last cell, its space, and one
		for i := off; i < len(g); i++ {
			c := i - off
			want := c >= 1 && c <= 3 || c >= 7 && c < 7+textWidth(pill) || c >= end-textWidth(modePill) && c < end
			if g[i] != want {
				t.Errorf("%v: cell %d of the row on the ground %v, want %v:\n%s", size, c, g[i], want, row)
				break
			}
		}
		for _, c := range []struct {
			text string
			st   *lipgloss.Style
		}{{" + ", tint(stText)}, {" " + model + " High", tint(stText)}, {" · ", tint(stMdBullet)}, {"Ultracode ", tint(stYou)}, {modePill, tint(stText)}} {
			if x, ok := spanFrom(ls[3], off, c.text); !ok || x.st != c.st {
				t.Errorf("%v: %q styled %v, want %v", size, c.text, x.st, c.st)
			}
		}
	}
	// A 12-line draft: the count three cells after the pill, in grey.
	var lines []string
	for i := range 12 {
		lines = append(lines, fmt.Sprintf("line %d", i+1))
	}
	ui := UI{Width: 120, Height: 40, Now: now, Focus: "B", Editor: EditorView{Lines: lines, Total: 12, Line: 11}}
	rows, ls, off := editorTail(t, idle, ui)
	if !strings.Contains(rows[3], "Ultracode    12 lines") || !strings.HasSuffix(strings.TrimRight(rows[3], " "), strings.TrimSpace(modePill)) {
		t.Errorf("a 12-line draft: the row %q", rows[3])
	} else if x, _ := spanFrom(ls[3], off, "12 lines"); x.st != stGrey {
		t.Errorf("the count styled %v, want stGrey", x.st)
	}
	// The turn running: the status in the middle where it fits beside the
	// pills, alone where it does not; with a draft of two lines, the count
	// before it.
	busy := highUltracode(quiet(t))
	busy.Turn.Phase = state.Running
	const running = "turn running: the draft waits · Ctrl+C interrupt"
	rows, _, _ = editorTail(t, busy, UI{Width: 208, Height: 59, Now: now, Focus: "B"})
	if !strings.Contains(rows[3], "Ultracode    "+running) || !strings.HasSuffix(strings.TrimRight(rows[3], " "), strings.TrimSpace(modePill)) {
		t.Errorf("208x59, the turn running: the row %q", rows[3])
	}
	rows, _, _ = editorTail(t, busy, UI{Width: 208, Height: 59, Now: now, Focus: "B", Editor: EditorView{Lines: []string{"a", "b"}, Total: 2, Line: 1}})
	if !strings.Contains(rows[3], "Ultracode    2 lines · "+running) {
		t.Errorf("208x59, the turn running with two lines: the row %q", rows[3])
	}
	rows, _, _ = editorTail(t, busy, UI{Width: 120, Height: 40, Now: now, Focus: "B"})
	if strings.TrimRight(rows[3], " ") != "  "+running || strings.Trim(strings.TrimSpace(rows[2]), "─") != "" {
		t.Errorf("120x40, the turn running: the row %q, want the status alone under the separator", rows[3])
	}
	// A paste in progress.
	rows, _, _ = editorTail(t, idle, UI{Width: 208, Height: 59, Now: now, Focus: "B", Editor: EditorView{Pasting: true}})
	if !strings.Contains(rows[3], "Ultracode    pasting…") {
		t.Errorf("pasting: the row %q", rows[3])
	}
	// The configuration file's problem keeps its red row alone (plan 21).
	rows, ls, off = editorTail(t, idle, UI{Width: 208, Height: 59, Now: now, Focus: "B", KeysProblem: "keys.json: bad"})
	if strings.TrimRight(rows[3], " ") != "  keys.json: bad" || strings.Trim(strings.TrimSpace(rows[2]), "─") != "" {
		t.Errorf("the keys problem: the row %q", rows[3])
	} else if x, _ := spanFrom(ls[3], off, "keys.json: bad"); x.st != stFailed {
		t.Errorf("the keys problem styled %v, want stFailed", x.st)
	}
	// A 40-line draft leaves the conversation at least half of B.
	lines = lines[:0]
	for i := range 40 {
		lines = append(lines, fmt.Sprintf("line %d", i+1))
	}
	for _, size := range [][2]int{{120, 40}, {200, 60}} {
		sc := Draw(idle, UI{Width: size[0], Height: size[1], Now: now, Focus: "B", Editor: EditorView{Lines: lines, Total: 40, Line: 39}})
		frame := strings.Split(strings.TrimSuffix(texts(sc.lines), "\n"), "\n")
		b := zoneBoxes(frame)["B"]
		rule := -1
		for y := b.bottom - 3; y > b.top; y-- {
			if txt := strings.TrimSpace(string([]rune(frame[y])[b.left+1 : b.right])); txt != "" && strings.Trim(txt, "─") == "" {
				rule = y
				break
			}
		}
		if inner, editor := b.bottom-b.top-1, b.bottom-rule; rule < 0 || editor > inner/2 || inner-editor < inner/2 {
			t.Errorf("%v: the editor takes %d of B's %d inner rows with a 40-line draft", size, editor, inner)
		}
	}
}

// TestEditorRowGap is plan 34's review: a middle text beside the pills
// keeps three cells before the mode pill (plan 39; "↯ Auto" before it), as
// after the model pill; one that would come closer has the row alone.
func TestEditorRowGap(t *testing.T) {
	s := highUltracode(quiet(t))
	for w := 60; w <= 130; w++ {
		for n := 1; n <= w; n++ {
			middle := strings.Repeat("x", n)
			text := strings.TrimSuffix(texts([]line{inputRow(s, middle, w, false, false)}), "\n")
			i := strings.Index(text, " ✋") // the mode pill's first cell
			if i < 0 {
				if strings.TrimSpace(text) != middle {
					t.Fatalf("width %d, a middle of %d: the row alone %q", w, n, text)
				}
				continue
			}
			before := strings.TrimRight(text[:i], " ")
			if !strings.HasSuffix(before, middle) || len(text[:i])-len(before) < 3 {
				t.Fatalf("width %d, a middle of %d: %q", w, n, text)
			}
		}
	}
}
