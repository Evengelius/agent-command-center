package render

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/Evengelius/agent-command-center/internal/provider/claude"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// Plan 18, item 2: zone C's height follows the agents in play (brief
// amendment 14): a strip without them, else the rows they need, at most half
// the right column.
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
	cTitle := regexp.MustCompile(`C · (LIVE )?AGENTS`)
	dTop := regexp.MustCompile(`^╭─ D[ ─]`) // "D" alone under 100 columns
	for _, c := range []struct {
		name     string
		s        *state.State
		w, h     int
		b, c     int    // B's and C's rows
		preview  bool   // the selected agent's preview beside the list
		agentRow bool   // P3b's agent listed in C
		hint     bool   // C's key hint
		focus    string // "" is B
	}{
		{"2.1 120x40 no agent", quiet, 120, 40, 29, 4, false, false, false, ""},
		{"2.1 120x40 one agent", one, 120, 40, 29, 4, false, true, true, ""},
		{"2.1 120x40 six agents", six, 120, 40, 24, 9, false, true, true, ""},
		{"2.1 120x40 twenty agents, half", twenty, 120, 40, 17, 16, false, false, true, ""},
		{"2.1 120x40 its agent done, a new prompt: cleared", after, 120, 40, 29, 4, false, false, false, ""},
		{"2.2 200x60 no agent", quiet, 200, 60, 47, 4, false, false, false, ""},
		{"2.2 200x60 one agent, room for the preview", one, 200, 60, 43, 8, true, true, true, ""},
		{"2.2 200x60 twenty agents, as they need", twenty, 200, 60, 28, 23, true, true, true, ""},
		{"2.2 200x60 forty agents, half", forty, 200, 60, 26, 25, true, false, true, ""},
		{"2.2 200x60 its agent done, a new prompt: cleared", after, 200, 60, 47, 4, false, false, false, ""},
		{"2.3 80x24 no agent", quiet, 80, 24, 14, 3, false, false, false, ""},
		{"2.3 80x24 one agent", one, 80, 24, 13, 4, false, true, true, ""},
		{"2.3 80x24 six agents, 5 rows", six, 80, 24, 12, 5, false, false, false, ""},
		{"2.4 100x24 twenty agents, B keeps 12", twenty, 100, 24, 12, 5, false, false, false, ""},
		{"2.4 100x24 no agent", quiet, 100, 24, 13, 4, false, false, false, ""},
		{"2.5 120x40 twenty done earlier: cleared", earlier, 120, 40, 29, 4, false, false, false, ""},
		{"2.5 120x40 twenty done earlier, C focused: cleared", earlier, 120, 40, 29, 4, false, false, false, "C"},
		{"2.5 200x60 twenty done earlier, C focused: cleared", earlier, 200, 60, 47, 4, false, false, false, "C"},
		{"2.5 80x24 twenty done earlier, C focused: cleared", earlier, 80, 24, 14, 3, false, false, false, "C"},
		{"2.5 120x40 six agents, C focused: as unfocused", six, 120, 40, 24, 9, false, true, true, "C"},
	} {
		focus := c.focus
		if focus == "" {
			focus = "B"
		}
		rows := strings.Split(texts(Draw(c.s, UI{Width: c.w, Height: c.h, Focus: focus}).lines), "\n")
		b, cRow, d := -1, -1, -1
		for i, r := range rows {
			switch {
			case b < 0 && strings.Contains(r, "B · CONVERSATION"):
				b = i
			case cRow < 0 && cTitle.MatchString(r):
				cRow = i
			case d < 0 && dTop.MatchString(r):
				d = i
			}
		}
		if b < 0 || cRow < 0 || d < 0 {
			t.Errorf("%s: zones not found (B %d, C %d, D %d)", c.name, b, cRow, d)
			continue
		}
		// C's own columns, from its box's corner: zone A beside it lists the
		// agents too and has its own hint.
		at := len([]rune(rows[cRow][:strings.Index(rows[cRow], "╭─ C")]))
		var cols []string
		for _, r := range rows[cRow:d] {
			cols = append(cols, string([]rune(r)[min(at, len([]rune(r))):]))
		}
		zoneC := strings.Join(cols, "\n")
		preview := strings.Contains(zoneC, "previous / next agent")
		// P3b's own agent, "general-purpose", cut to 14 cells in narrow
		// columns; the extras are "extra-…".
		agentRow := strings.Contains(zoneC, "general-p")
		hint := strings.Contains(zoneC, "z zoom · F6 next zone")
		if cRow-b != c.b || d-cRow != c.c || preview != c.preview || agentRow != c.agentRow || hint != c.hint {
			t.Errorf("%s: B %d rows, C %d, preview %v, agent row %v, hint %v; want %d, %d, %v, %v, %v",
				c.name, cRow-b, d-cRow, preview, agentRow, hint, c.b, c.c, c.preview, c.agentRow, c.hint)
		}
		// Nothing in play: C is empty, its counts at zero (plan 18, the
		// owner's third message); else it lists them.
		empty := strings.Contains(zoneC, "no agents yet") && strings.Contains(zoneC, "0 ACTIVE · 0 DONE")
		if want := c.s.AgentsInPlay() == 0; empty != want {
			t.Errorf("%s: empty %v, want %v", c.name, empty, want)
		}
	}
}
