package app

import (
	"strings"
	"testing"
	"testing/synctest"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/Evengelius/agent-command-center/internal/fixture"
	"github.com/Evengelius/agent-command-center/internal/provider/claude"
	"github.com/Evengelius/agent-command-center/internal/provider/codex"
	"github.com/Evengelius/agent-command-center/internal/render"
)

// Plan 22, amendment 20 at the application: P22s (Codex 0.163.0 asked for
// reasoning summaries) and P22o (Claude Code 2.1.303 with the owner's
// SessionStart hook) replayed as a live run receives them, one recorded moment
// per update with the clock at its time (thinkStart).

const (
	p22sProbe = "p22s-codex-reasoning-summary.jsonl"
	p22oProbe = "p22o-claude-opus-thinking.jsonl"
)

// frameAt is the text of a frame at the recorded time ms, after every record up
// to it.
func (r *thinkSession) frameAt(ms int) string {
	r.until(ms)
	r.at(ms)
	return plainText(r.h.m.View().Content)
}

// 20.4: the card reads "Model working · …" from the step's start (5,252 ms)
// until the reasoning item starts (6,882 ms), and "Thinking · reasoning tokens
// reported when the step ends" from then until the item completes (9,388 ms),
// for the second turn the same from 9,849 and 11,397 to 14,833 ms: 2.5 and 3.4
// s in P22s, where it was 0 to 10 ms. The card itself is as before.
func TestCodexCardReadsThinkingFromTheItemsStart(t *testing.T) {
	const working = "Model working · reasoning tokens reported when the step ends"
	const thinking = "Thinking · reasoning tokens reported when the step ends"
	synctest.Test(t, func(t *testing.T) {
		r := thinkStart(t, p22sProbe, codex.ProviderID)
		r.h.draw(tea.WindowSizeMsg{Width: 200, Height: 60})
		for _, c := range []struct {
			ms   int
			want string // the card's text; "" for no card
			also string // a text the frame holds too
		}{
			{3000, "", ""}, // before the turn's first step
			{5500, working, ""}, {6000, working, ""}, {6881, working, ""},
			{6882, thinking, ""}, {8000, thinking, ""}, {9387, thinking, ""},
			// Past the item's completion (9,388 ms) the card is over, and the phase is
			// its summary row, "up to" the step's time, as it was.
			{9800, "", "Thought for up to 4.1s · 28 reasoning tokens"},
			{10000, working, ""}, {11396, working, ""}, // the second turn's step
			{11397, thinking, ""}, {13000, thinking, ""}, {14832, thinking, ""},
		} {
			text := r.frameAt(c.ms)
			has := func(s string) bool { return strings.Contains(text, s) }
			switch c.want {
			case working:
				if !has(working) || has(thinking) {
					t.Errorf("at %d ms: want %q only in\n%s", c.ms, working, text)
				}
			case thinking:
				if !has(thinking) || has(working) {
					t.Errorf("at %d ms: want %q only in\n%s", c.ms, thinking, text)
				}
			default:
				if has(working) || has(thinking) {
					t.Errorf("at %d ms: a card where there is none:\n%s", c.ms, text)
				}
			}
			if c.also != "" && !has(c.also) {
				t.Errorf("at %d ms: no %q in\n%s", c.ms, c.also, text)
			}
		}
	})
}

// The details of P22s's phases in the application: opened under their rows they
// give the reasoning item's own span, the live-estimate line, and the summary
// Codex sent for the second item, "**Computing f(10)**", as its own text; the
// first says Codex sent none. Zone D counts nothing unrecognised (it counted 1).
func TestCodexSummaryInTheApplication(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := thinkStart(t, p22sProbe, codex.ProviderID)
		r.h.draw(tea.WindowSizeMsg{Width: 200, Height: 60})
		r.until(15000)
		for n := 1; n <= 2; n++ {
			r.h.m.unfold(render.ThoughtKey(thought(t, r.h.m.State(), n)), true)
		}
		r.h.m.ui.ConvHold = false
		r.at(15300) // past the second phase's exit (14,833 ms + 300)
		text := plainText(r.h.m.View().Content)
		clock := func(ms int) string {
			return fixture.Epoch.Add(time.Duration(ms) * time.Millisecond).Local().Format("15:04:05.000")
		}
		for _, want := range []string{
			"▾ ✓ Thought for up to 4.1s · 28 reasoning tokens",
			"the reasoning item from " + clock(6882) + " to " + clock(9388) + " · observed",
			"no reasoning summary: Codex sent none",
			"▾ ✓ Thought for up to 5.0s · 36 reasoning tokens",
			"the reasoning item from " + clock(11397) + " to " + clock(14833) + " · observed",
			"Codex's summary, its own text: **Computing f(10)**",
		} {
			if !strings.Contains(text, want) {
				t.Errorf("no %q in\n%s", want, text)
			}
		}
		if strings.Contains(text, "unrecognised ") {
			t.Errorf("an unrecognised note (the pane notes a count above 0 only):\n%s", text)
		}
		if n := strings.Count(text, "no live estimate: Codex reports reasoning tokens when the step ends"); n != 2 {
			t.Errorf("%d \"no live estimate\" lines, want one for each phase", n)
		}
		for _, gone := range []string{"Codex reports reasoning once it ended", "the CLI sends none"} {
			if strings.Contains(text, gone) {
				t.Errorf("the old line %q is still drawn", gone)
			}
		}
		if s := r.h.m.State(); s.Unrecognised != 0 {
			t.Errorf("%d unrecognised events", s.Unrecognised)
		}
	})
}

// 20.7 at the application: P22o (the owner's SessionStart hook, the turn's
// summary line) replayed counts nothing unrecognised in zone D (it counted 3),
// and its conversation is the prompt, one thinking phase and the answer.
func TestClaudeHookLinesAreNotUnrecognised(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := thinkStart(t, p22oProbe, claude.ProviderID)
		r.h.draw(tea.WindowSizeMsg{Width: 200, Height: 60})
		text := r.frameAt(14400) // after the result line (14,361 ms)
		if strings.Contains(text, "unrecognised ") {
			t.Errorf("an unrecognised note (none is drawn at 0):\n%s", text)
		}
		s := r.h.m.State()
		if s.Unrecognised != 0 || len(s.Errors) != 0 {
			t.Errorf("%d unrecognised events, %d errors", s.Unrecognised, len(s.Errors))
		}
		var roles []string
		for _, e := range s.Conversation {
			roles = append(roles, e.Role)
		}
		if strings.Join(roles, " ") != "user thinking assistant" {
			t.Errorf("conversation %v", roles)
		}
		if !strings.Contains(text, "Thought for 8.0s · 928 thinking tokens") || !strings.Contains(text, "**59**") {
			t.Errorf("the thinking row or the answer is missing:\n%s", text)
		}
	})
}
