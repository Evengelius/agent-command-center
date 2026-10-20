package render

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"charm.land/lipgloss/v2"

	"github.com/Evengelius/agent-command-center/internal/event"
	"github.com/Evengelius/agent-command-center/internal/provider/claude"
	"github.com/Evengelius/agent-command-center/internal/provider/mock"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// zoneRows reads the dashboard's zones from a frame's text: each box's top
// and bottom rows and its left and right columns, by its title.
type zoneBox struct{ top, bottom, left, right int }

func zoneBoxes(rows []string) map[string]zoneBox {
	out := map[string]zoneBox{}
	for y, r := range rows {
		cells := []rune(r)
		for x := 0; x+3 < len(cells); x++ {
			if cells[x] != '╭' || cells[x+1] != '─' || cells[x+2] != ' ' {
				continue
			}
			name := string(cells[x+3])
			right := x
			for right < len(cells) && cells[right] != '╮' {
				right++
			}
			bottom := y
			for bottom+1 < len(rows) {
				bottom++
				if rc := []rune(rows[bottom]); x < len(rc) && rc[x] == '╰' {
					break
				}
			}
			if _, seen := out[name]; !seen {
				out[name] = zoneBox{y, bottom, x, right}
			}
		}
	}
	return out
}

// TestDashboardGeometry is plan 20's item 1 (brief amendment 16): A and B
// share the upper row, their bottom borders on one row; C spans the whole
// width under both; D under C; the footer last.
func TestDashboardGeometry(t *testing.T) {
	quiet := probeState(t, "p2-claude-conversation.jsonl", claude.ProviderID, claude.NewMapper())
	one := probeState(t, "p3b-claude-subagent-approval.jsonl", claude.ProviderID, claude.NewMapper())
	for _, s := range []*state.State{quiet, one} {
		for _, c := range []struct{ w, h, aw int }{{120, 40, 26}, {156, 44, 34}, {160, 45, 35}, {200, 60, 44}} {
			name := fmt.Sprintf("%dx%d, %d agents", c.w, c.h, s.AgentsInPlay())
			rows := strings.Split(strings.TrimSuffix(texts(Draw(s, UI{Width: c.w, Height: c.h, Focus: "B"}).lines), "\n"), "\n")
			z := zoneBoxes(rows)
			a, b, cz, d := z["A"], z["B"], z["C"], z["D"]
			// 1.1 A and B on row 3, A aw wide, B the rest, their bottoms on one row.
			if a.top != 2 || b.top != 2 || a.left != 0 || a.right != c.aw-1 || b.left != c.aw || b.right != c.w-1 || a.bottom != b.bottom {
				t.Errorf("%s: A %+v, B %+v; want both from row 2 to one row, A 0..%d, B %d..%d", name, a, b, c.aw-1, c.aw, c.w-1)
			}
			// 1.2 C full width under them, D under C, the footer last.
			if cz.top != a.bottom+1 || cz.left != 0 || cz.right != c.w-1 || d.top != cz.bottom+1 || d.left != 0 || d.right != c.w-1 || d.bottom != c.h-2 {
				t.Errorf("%s: C %+v, D %+v; want C from row %d full width, D under it to row %d", name, cz, d, a.bottom+1, c.h-2)
			}
			if !strings.HasPrefix(rows[c.h-1], " Enter send") {
				t.Errorf("%s: last row %q", name, rows[c.h-1])
			}
			for i, r := range rows {
				if n := len([]rune(r)); n > c.w {
					t.Errorf("%s: row %d is %d wide", name, i, n)
				}
			}
		}
	}
	// 1.3 At 80x24: A's row, then B, C and D full width; C compact.
	rows := strings.Split(strings.TrimSuffix(texts(Draw(quiet, UI{Width: 80, Height: 24, Focus: "B"}).lines), "\n"), "\n")
	z := zoneBoxes(rows)
	if !strings.HasPrefix(rows[2], " A ") || z["B"].top != 3 || z["B"].right != 79 || z["C"].top != z["B"].bottom+1 || z["C"].bottom-z["C"].top+1 != 3 || z["C"].right != 79 || z["D"].top != z["C"].bottom+1 {
		t.Errorf("80x24: row 2 %q, zones %+v", rows[2], z)
	}
}

// TestAdaptiveLayout is plan 18's item 2 and plan 20's item 2: zone C's
// height follows the agents in play, compact without them, two rows an
// agent from 100 columns, one under it; at most half the rows it shares
// with A and B, B keeping its least height; cleared by the owner's next
// prompt.
func TestAdaptiveLayout(t *testing.T) {
	p3b := func() *state.State {
		return probeState(t, "p3b-claude-subagent-approval.jsonl", claude.ProviderID, claude.NewMapper())
	}
	quiet := probeState(t, "p2-claude-conversation.jsonl", claude.ProviderID, claude.NewMapper())
	one := p3b() // its one agent, done, in play
	// P3b's agent, done, once the owner's next prompt opens its stage.
	after := p3b()
	after.Stages = append(after.Stages, state.Stage{N: len(after.Stages) + 1, Open: true})
	// P3b with more agents running in its turn.
	more := func(n int) *state.State {
		s := p3b()
		for i := 0; i < n-1; i++ {
			k := fmt.Sprintf("extra-%02d", i)
			s.Agents[k] = &state.Agent{Key: k, Label: k, Status: state.AgentRunning, Stage: len(s.Stages), Runs: 1}
			s.Order = append(s.Order, k)
		}
		return s
	}
	six, twenty, forty := more(6), more(20), more(40)
	// Twenty agents done in P3b's turn, then the owner's next prompt.
	earlier := more(20)
	for _, k := range earlier.Order {
		if a := earlier.Agents[k]; !a.Status.Final() {
			a.Status, a.EndStage = state.AgentCompleted, a.Stage
		}
	}
	earlier.Stages = append(earlier.Stages, state.Stage{N: len(earlier.Stages) + 1, Open: true})
	for _, c := range []struct {
		s *state.State
		n int
	}{{quiet, 0}, {one, 1}, {after, 0}, {six, 6}, {twenty, 20}, {forty, 40}, {earlier, 0}} {
		if got := c.s.AgentsInPlay(); got != c.n {
			t.Fatalf("agents in play %d, want %d", got, c.n)
		}
	}
	for _, c := range []struct {
		name     string
		s        *state.State
		w, h     int
		top, c   int  // the upper row's rows (A and B), C's
		preview  bool // the selected agent's preview beside the list
		agentRow bool // P3b's agent listed in C
		focus    string
	}{
		{"2.1 120x40 no agent", quiet, 120, 40, 30, 3, false, false, ""},
		{"2.2 120x40 one agent", one, 120, 40, 29, 4, false, true, ""},
		{"2.2 120x40 six agents", six, 120, 40, 19, 14, false, true, ""},
		{"2.2 120x40 twenty agents, half", twenty, 120, 40, 17, 16, false, false, ""},
		{"2.3 120x40 its agent done, a new prompt: cleared", after, 120, 40, 30, 3, false, false, ""},
		{"2.1 200x60 no agent", quiet, 200, 60, 48, 3, false, false, ""},
		{"2.2 200x60 one agent, room for the preview", one, 200, 60, 43, 8, true, true, ""},
		{"2.2 200x60 twenty agents, half", twenty, 200, 60, 26, 25, true, false, ""},
		{"2.2 200x60 forty agents, half", forty, 200, 60, 26, 25, true, false, ""},
		{"2.3 200x60 its agent done, a new prompt: cleared", after, 200, 60, 48, 3, false, false, ""},
		{"2.1 156x44 no agent", quiet, 156, 44, 34, 3, false, false, ""},
		{"2.2 156x44 one agent", one, 156, 44, 33, 4, false, true, ""},
		{"2.1 100x24 no agent", quiet, 100, 24, 14, 3, false, false, ""},
		{"2.2 100x24 twenty agents, B keeps 12", twenty, 100, 24, 12, 5, false, false, ""},
		{"2.3 120x40 twenty done earlier, C focused: cleared", earlier, 120, 40, 30, 3, false, false, "C"},
		{"2.2 120x40 six agents, C focused: as unfocused", six, 120, 40, 19, 14, false, true, "C"},
	} {
		focus := c.focus
		if focus == "" {
			focus = "B"
		}
		rows := strings.Split(strings.TrimSuffix(texts(Draw(c.s, UI{Width: c.w, Height: c.h, Focus: focus}).lines), "\n"), "\n")
		z := zoneBoxes(rows)
		top, ch := z["B"].bottom-z["B"].top+1, z["C"].bottom-z["C"].top+1
		zoneC := strings.Join(rows[z["C"].top:z["C"].bottom+1], "\n")
		// The preview is inside C since plan 20, right of the list after
		// " │ ", the only one inside its rows.
		preview := strings.Contains(strings.Join(rows[z["C"].top+1:z["C"].bottom], "\n"), " │ ")
		agentRow := strings.Contains(zoneC, "general-p")
		if top != c.top || ch != c.c || preview != c.preview || agentRow != c.agentRow {
			t.Errorf("%s: upper row %d rows, C %d, preview %v, agent row %v; want %d, %d, %v, %v:\n%s",
				c.name, top, ch, preview, agentRow, c.top, c.c, c.preview, c.agentRow, zoneC)
		}
		// Nothing in play: C is empty, its counts at zero (plan 18, the
		// owner's third message); else it lists them, without a hint row
		// (plan 20: the footer has C's keys).
		empty := strings.Contains(zoneC, "no agents yet") && strings.Contains(zoneC, "0 ACTIVE · 0 DONE")
		if want := c.s.AgentsInPlay() == 0; empty != want {
			t.Errorf("%s: empty %v, want %v", c.name, empty, want)
		}
		if strings.Contains(zoneC, "z zoom") {
			t.Errorf("%s: a key hint row in C:\n%s", c.name, zoneC)
		}
		if footer := rows[c.h-1]; (focus == "C") != strings.Contains(footer, "↑↓ select · Enter inspect") {
			t.Errorf("%s: footer %q with C focused %v", c.name, footer, focus == "C")
		}
	}
	// At 80x24, one row an agent, at most 5 rows for C.
	for _, c := range []struct {
		s    *state.State
		b, c int
	}{{quiet, 14, 3}, {one, 14, 3}, {six, 12, 5}} {
		rows := strings.Split(strings.TrimSuffix(texts(Draw(c.s, UI{Width: 80, Height: 24, Focus: "B"}).lines), "\n"), "\n")
		z := zoneBoxes(rows)
		if b, ch := z["B"].bottom-z["B"].top+1, z["C"].bottom-z["C"].top+1; b != c.b || ch != c.c {
			t.Errorf("80x24, %d agents: B %d, C %d; want %d, %d", c.s.AgentsInPlay(), b, ch, c.b, c.c)
		}
	}
}

// TestAgentCardTemplate2 is plan 20's criterion 2.4: zone C's two rows of
// the agent waiting on its Write, Template 2's, from P3b's events up to that
// request (its mock file, which holds the owner's answer to the Bash): the
// pink dot, the label, the model, "waiting: approval" in grey; then the
// time, the tokens, "tool: Bash ✓" and "next: Write (approval pending)" in
// grey.
func TestAgentCardTemplate2(t *testing.T) {
	defer UsePalette(pickTrue)
	f := filepath.Join("..", "..", "testdata", "mock", "claude-subagent-approval.jsonl")
	evs, err := mock.Load(f)
	if err != nil {
		t.Fatal(err)
	}
	n, cut := 0, len(evs)
	base := time.Date(2026, 10, 20, 1, 55, 0, 0, time.Local)
	for i, e := range evs {
		evs[i].Event.At = base.Add(e.After) // as mock mode stamps them, for the agent's time
		if e.Event.Kind == event.ApprovalRequest {
			if n++; n == 2 {
				cut = i + 1
				break
			}
		}
	}
	s, now := mockState(f, evs[:cut])
	if q := s.Queue; len(q) != 2 || !s.Approvals[q[0]].Status.Final() || s.Approvals[q[1]].Tool != "Write" || s.Approvals[q[1]].Status != state.Pending {
		t.Fatalf("queue %v: want the Bash answered, the Write pending", q)
	}
	for _, size := range [][2]int{{156, 44}, {120, 40}} {
		ls := Draw(s, UI{Width: size[0], Height: size[1], Now: now, Focus: "B"}).lines
		rows := strings.Split(texts(ls), "\n")
		c := zoneBoxes(rows)["C"]
		first, second := ls[c.top+1], ls[c.top+2]
		got := texts([]line{first, second})
		for _, want := range []string{"●  general-purpose", "haiku-5-5", "waiting: approval", "tool: Bash ✓", "next: Write (approval pending)", "1 ACTIVE · 0 DONE"} {
			if !strings.Contains(got, want) && !strings.Contains(rows[c.top], want) {
				t.Errorf("%v: no %q in C's rows:\n%s", size, want, got)
			}
		}
		if !regexp.MustCompile(`^│\s+\d\d:\d\d\s+[\d.]+k tok`).MatchString(texts([]line{second})) {
			t.Errorf("%v: the second row %q lacks the time and the tokens", size, texts([]line{second}))
		}
		for _, c := range []struct {
			l    line
			text string
			st   *lipgloss.Style
		}{{first, "●", stAttention}, {first, "waiting: approval", stMuted}, {second, "   next: Write (approval pending)", stMuted}, {second, "✓", stDone}} {
			if x, ok := spanOf(c.l, c.text); !ok || x.st != c.st {
				t.Errorf("%v: %q styled %v, want %v", size, c.text, x.st, c.st)
			}
		}
	}
	// Under 100 columns, the one row with the same pink dot.
	ls := Draw(s, UI{Width: 80, Height: 24, Now: now, Focus: "B"}).lines
	rows := strings.Split(texts(ls), "\n")
	c := zoneBoxes(rows)["C"]
	if row := ls[c.top+1]; !strings.Contains(texts([]line{row}), "● general-purpose") || !strings.Contains(texts([]line{row}), "waiting: approval") {
		t.Errorf("80x24: C's row %q", texts([]line{row}))
	} else if x, _ := spanOf(row, "●"); x.st != stAttention {
		t.Errorf("80x24: the dot styled %v", x.st)
	}
}

// TestCompactEditor is plan 20's criterion 3.10: under its rule the input
// is one row, then the status row, no empty row between them; a 3-line
// draft takes three input rows.
func TestCompactEditor(t *testing.T) {
	// The Claude mock's session, ready (a probe's ends with its process).
	f := filepath.Join("..", "..", "testdata", "mock", "claude-subagent-approval.jsonl")
	evs, err := mock.Load(f)
	if err != nil {
		t.Fatal(err)
	}
	s, _ := mockState(f, evs)
	for _, c := range []struct {
		lines []string
		rows  int
		count string
	}{{[]string{""}, 1, "1 line"}, {[]string{"one"}, 1, "1 line"}, {[]string{"one", "two", "three"}, 3, "3 lines"}} { // as the app sends them: an empty draft is one empty line
		for _, size := range [][2]int{{80, 24}, {120, 40}, {200, 60}} {
			ui := UI{Width: size[0], Height: size[1], Focus: "B", Editor: EditorView{Lines: c.lines, Total: max(len(c.lines), 1)}}
			rows := strings.Split(texts(Draw(s, ui).lines), "\n")
			b := zoneBoxes(rows)["B"]
			inner := func(y int) string {
				cells := []rune(rows[y])
				return strings.TrimSpace(string(cells[b.left+1 : b.right]))
			}
			rule := -1
			for y := b.bottom - 1; y > b.top && rule < 0; y-- {
				if txt := inner(y); txt != "" && strings.Trim(txt, "─") == "" {
					rule = y
				}
			}
			got := b.bottom - 1 - rule - 1 // the rows between the rule and the status row
			if rule < 0 || got != c.rows || !strings.HasPrefix(inner(rule+1), ">") || !strings.Contains(inner(b.bottom-1), c.count) {
				t.Errorf("%v, %d lines: %d input rows under the rule (row %d), want %d; first %q, status %q:\n%s", size, len(c.lines), got, rule, c.rows, inner(rule+1), inner(b.bottom-1), strings.Join(rows, "\n"))
			}
		}
	}
}
