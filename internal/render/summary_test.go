package render

import (
	"strings"
	"testing"
	"time"

	"github.com/Evengelius/agent-command-center/internal/state"
)

// Plan 22, amendment 20, criterion 20.5: the details of a Codex phase when its
// reasoning item started (observed) and when Codex sent a summary. P22s is
// Codex 0.163.0 asked for summaries: its phases' items ran 6,882 to 9,388 ms
// and 11,397 to 14,833 ms; the second has the summary "**Computing f(10)**".

const p22sProbe = "p22s-codex-reasoning-summary.jsonl"

// detailClock is the recording's time ms as the details print it.
func detailClock(ms int) string { return thinkAt(ms).Local().Format("15:04:05.000") }

// 20.5: the details of both P22s phases, word for word, and the phases' summary
// rows unchanged ("Thought for up to 4.1s · 28 reasoning tokens", "… up to 5.0s ·
// 36 reasoning tokens"); open under their rows, the lines are the entry's own.
func TestCodexSummaryDetails(t *testing.T) {
	s := replayProbe(t, p22sProbe)
	idx := thoughtEntries(s)
	if len(idx) != 2 {
		t.Fatalf("%d phases, want 2", len(idx))
	}
	const tokens = " reasoning tokens · exact, for this step · thread/tokenUsage/updated last.reasoningOutputTokens"
	for n, c := range []struct {
		row     string
		details []string
	}{
		{"Thought for up to 4.1s · 28 reasoning tokens", []string{
			"the model step from " + detailClock(5252) + " (inferred) to its reasoning's end at " + detailClock(9388) + " · reasoning item",
			"the reasoning item from " + detailClock(6882) + " to " + detailClock(9388) + " · observed",
			"28" + tokens,
			"no live estimate: Codex reports reasoning tokens when the step ends",
			"no reasoning summary: Codex sent none",
		}},
		{"Thought for up to 5.0s · 36 reasoning tokens", []string{
			"the model step from " + detailClock(9849) + " (inferred) to its reasoning's end at " + detailClock(14833) + " · reasoning item",
			"the reasoning item from " + detailClock(11397) + " to " + detailClock(14833) + " · observed",
			"36" + tokens,
			"no live estimate: Codex reports reasoning tokens when the step ends",
			"Codex's summary, its own text: **Computing f(10)**",
		}},
	} {
		e := s.Conversation[idx[n]]
		if got, _, _, _ := thoughtSummary(e.Thought); got != c.row {
			t.Errorf("phase %d: summary row %q, want %q", n+1, got, c.row)
		}
		if got := thoughtDetails(e.Thought); strings.Join(got, "\n") != strings.Join(c.details, "\n") {
			t.Errorf("phase %d: details\n%s\nwant\n%s", n+1, strings.Join(got, "\n"), strings.Join(c.details, "\n"))
		}
		// The entry's lines, both drawn and copied: the summary, then the details.
		if got := entryLines(s, e); len(got) != 1+len(c.details) || got[0] != c.row || strings.Join(got[1:], "\n") != strings.Join(c.details, "\n") {
			t.Errorf("phase %d: entry lines %q", n+1, got)
		}
	}

	// Open under the rows, in zone B at 200 columns: each detail a row.
	open := map[string]bool{}
	for _, i := range idx {
		open[ThoughtKey(s.Conversation[i])] = true
	}
	closed := rowTexts(bRows(s, UI{}, 200))
	rows := rowTexts(bRows(s, UI{Unfolded: open}, 200))
	if len(rows) != len(closed)+10 {
		t.Errorf("open: %d rows, closed %d, want 10 more", len(rows), len(closed))
	}
	for n, row := range []string{"Thought for up to 4.1s · 28 reasoning tokens", "Thought for up to 5.0s · 36 reasoning tokens"} {
		found := false
		for i, r := range rows {
			if !strings.HasPrefix(r, "▎  ▾ ✓ "+row) {
				continue
			}
			found = true
			want := thoughtDetails(s.Conversation[idx[n]].Thought)
			for j, w := range want {
				if i+1+j >= len(rows) || rows[i+1+j] != "▎     "+w {
					t.Errorf("phase %d: detail %d %q, want %q", n+1, j, rows[min(i+1+j, len(rows)-1)], "▎     "+w)
				}
			}
		}
		if !found {
			t.Errorf("no open row for %q in\n%s", row, strings.Join(rows, "\n"))
		}
	}
}

// The reasoning item's line says what the item reported: its start and its end
// when it completed; its start and that its completion was not reported when
// the phase ended some other way (the next step, the turn); and nothing when
// its start is not known. The count, the live estimate and the summary lines
// keep their order.
func TestCodexItemLine(t *testing.T) {
	base := state.Thought{Start: thinkAt(1000), ItemStart: thinkAt(2500), End: thinkAt(4000), Inferred: true, Confirmed: true}
	for _, c := range []struct {
		name  string
		mut   func(th *state.Thought)
		step  string
		item  string // "" for no line
		count string
	}{
		{"completed", func(th *state.Thought) {},
			"the model step from " + detailClock(1000) + " (inferred) to its reasoning's end at " + detailClock(4000) + " · reasoning item",
			"the reasoning item from " + detailClock(2500) + " to " + detailClock(4000) + " · observed", "exact count not reported yet"},
		{"ended by the next step", func(th *state.Thought) { th.EndInferred, th.EndNext = true, true },
			"the model step from " + detailClock(1000) + " (inferred) to " + detailClock(4000) + ", when the next step began (inferred)",
			"the reasoning item from " + detailClock(2500) + " · observed, its completion not reported", "no exact count reported"},
		{"ended by its turn", func(th *state.Thought) { th.EndInferred = true; th.Outcome = "interrupted" },
			"the model step from " + detailClock(1000) + " (inferred) to " + detailClock(4000) + ", when its turn ended (inferred)",
			"the reasoning item from " + detailClock(2500) + " · observed, its completion not reported", "no exact count reported"},
		{"its start not known", func(th *state.Thought) { th.ItemStart = time.Time{} },
			"the model step from " + detailClock(1000) + " (inferred) to its reasoning's end at " + detailClock(4000) + " · reasoning item",
			"", "exact count not reported yet"},
	} {
		th := base
		c.mut(&th)
		got := thoughtDetails(&th)
		want := []string{c.step}
		if c.item != "" {
			want = append(want, c.item)
		}
		want = append(want, c.count, "no live estimate: Codex reports reasoning tokens when the step ends", "no reasoning summary: Codex sent none")
		if strings.Join(got, "\n") != strings.Join(want, "\n") {
			t.Errorf("%s: details\n%s\nwant\n%s", c.name, strings.Join(got, "\n"), strings.Join(want, "\n"))
		}
	}
	// With a count, the count comes after the item's line and before the rest.
	th := base
	th.HasTokens, th.Tokens = true, 28
	d := thoughtDetails(&th)
	if len(d) != 5 || !strings.HasPrefix(d[1], "the reasoning item from ") || !strings.HasPrefix(d[2], "28 reasoning tokens · exact, for this step") || !strings.HasPrefix(d[3], "no live estimate") || !strings.HasPrefix(d[4], "no reasoning summary") {
		t.Errorf("details with a count: %q", d)
	}
}

// A summary of several parts is Codex's text with a blank line between them: the
// entry's lines hold no line break, so the first part follows the label, a blank
// line stands for the break, and each other part is a line of its own, drawn
// under the label and copied as the text is.
func TestCodexSummaryLines(t *testing.T) {
	th := state.Thought{Start: thinkAt(1000), ItemStart: thinkAt(2500), End: thinkAt(4000), Inferred: true, Confirmed: true,
		Summary: "**Planning the change**\n\nI will read the file first,\nthen edit it.\n\n**Checking**"}
	d := thoughtDetails(&th)
	if len(d) < 6 {
		t.Fatalf("details %q: no summary lines", d)
	}
	tail := d[len(d)-6:]
	want := []string{
		"Codex's summary, its own text: **Planning the change**",
		"",
		"I will read the file first,",
		"then edit it.",
		"",
		"**Checking**",
	}
	if strings.Join(tail, "|") != strings.Join(want, "|") {
		t.Errorf("summary lines %q, want %q", tail, want)
	}
	for _, l := range d {
		if strings.Contains(l, "\n") {
			t.Errorf("a detail line holds a line break: %q", l)
		}
	}
	// No "no reasoning summary" line beside a summary.
	for _, l := range d {
		if strings.HasPrefix(l, "no reasoning summary") {
			t.Errorf("%q beside a summary", l)
		}
	}
	// Drawn open: the label's row, a blank detail row, then the next part.
	s := thinkBench(th)
	e := s.Conversation[1]
	rows := rowTexts(bRows(s, UI{Unfolded: map[string]bool{ThoughtKey(e): true}}, 200))
	for i, r := range rows {
		if r != "▎     Codex's summary, its own text: **Planning the change**" {
			continue
		}
		if i+5 >= len(rows) || strings.TrimRight(rows[i+1], " ▎") != "" || rows[i+2] != "▎     I will read the file first," || rows[i+3] != "▎     then edit it." || strings.TrimRight(rows[i+4], " ▎") != "" || rows[i+5] != "▎     **Checking**" {
			t.Errorf("rows from the label:\n%s", strings.Join(rows[i:min(i+6, len(rows))], "\n"))
		}
		return
	}
	t.Errorf("no row for the summary's label in\n%s", strings.Join(rows, "\n"))
}

// Copied from zone B, an open summary comes out as the entry's lines do: the
// summary row, the details, and Codex's text with its own line breaks and blank
// line, as it sent it.
func TestCodexSummaryCopies(t *testing.T) {
	th := state.Thought{Start: thinkAt(1000), ItemStart: thinkAt(2500), End: thinkAt(4000), Inferred: true, Confirmed: true,
		Summary: "**Planning the change**\n\nI will read the file first,\nthen edit it."}
	s := thinkBench(th)
	e := s.Conversation[1]
	ls := entryLines(s, e)
	ui := UI{Width: 200, Height: 60, Unfolded: map[string]bool{ThoughtKey(e): true}}
	sc := Draw(s, ui)
	got := sc.Text(s, Selection{From: Pos{Panel: 'B', Item: 1}, To: Pos{Panel: 'B', Item: 1, Line: len(ls) - 1, Off: len(ls[len(ls)-1])}})
	if want := strings.Join(ls, "\n"); got != want {
		t.Errorf("copied\n%q\nwant\n%q", got, want)
	}
	if !strings.HasSuffix(got, "Codex's summary, its own text: **Planning the change**\n\nI will read the file first,\nthen edit it.") {
		t.Errorf("the copy does not end with Codex's text as sent: %q", got)
	}
}

// A Claude Code phase has no reasoning item and no Codex summary: its details
// are the four they were (plan 22, design 8), whatever a stray field holds.
func TestClaudeDetailsAreUnchanged(t *testing.T) {
	th := state.Thought{Start: thinkAt(940), End: thinkAt(3756), Estimate: 650, Tokens: 719, HasTokens: true, TokensAt: thinkAt(4300), Confirmed: true}
	want := []string{
		"from " + detailClock(940) + " to " + detailClock(3756) + " · a thinking block, observed",
		"719 thinking tokens · exact, for this phase · message_delta usage.output_tokens_details.thinking_tokens",
		"live estimate 650 at its last update · system/thinking_tokens, estimated by the CLI",
		"no thinking text: the CLI sends none",
	}
	if got := thoughtDetails(&th); strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("details\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	th.ItemStart, th.Summary = thinkAt(500), "stray"
	if got := thoughtDetails(&th); strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("a Claude Code phase with Codex's fields set:\n%s", strings.Join(got, "\n"))
	}
}
