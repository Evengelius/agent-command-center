package render

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"charm.land/lipgloss/v2"

	"github.com/Evengelius/agent-command-center/internal/event"
	"github.com/Evengelius/agent-command-center/internal/provider/claude"
	"github.com/Evengelius/agent-command-center/internal/provider/codex"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// Plan 22, corrections C1 to C10 at the render level (plan.md, Corrections C1
// to C12; doc/plan22-investigation.md, section 2): the thinking phase's rows,
// details, card and activity line, from the recordings and from phases built
// by hand at chosen frame times. The reducer's side of C1 to C5 is tested in
// internal/state, the application's in internal/app.

// cardAt is entry i's card or exit as drawn in zone B's inner width w, size
// rows when steady, without its block.
func cardAt(s *state.State, ui UI, i, w, size int) []line {
	lk, _ := lookAll(s, ui)
	k := lk[i]
	k.size = size
	return newCard(s, ui, s.Conversation[i], cardContent(k.card, w), size).rows(s, ui, i, k)
}

// replayProbe is a recording of testdata/probes replayed through its CLI's mapper.
func replayProbe(t *testing.T, name string) *state.State {
	t.Helper()
	if strings.Contains(name, "codex") {
		return probeState(t, name, codex.ProviderID, codex.NewMapper())
	}
	return probeState(t, name, claude.ProviderID, claude.NewMapper())
}

// thoughtEntries are the indexes of the thinking entries of s.
func thoughtEntries(s *state.State) []int {
	var out []int
	for i, e := range s.Conversation {
		if e.Role == "thinking" && e.Thought != nil {
			out = append(out, i)
		}
	}
	return out
}

// cutEntry is the index of the entry of the phase whose message was cut short.
func cutEntry(t *testing.T, s *state.State) int {
	t.Helper()
	for _, i := range thoughtEntries(s) {
		if s.Conversation[i].Thought.Cut {
			return i
		}
	}
	t.Fatal("no phase of a message cut short")
	return -1
}

// zeroCount finds "0 thinking tokens" standing alone as a count, not as the
// end of "50 thinking tokens".
var zeroCount = regexp.MustCompile(`(^|[^0-9,.])0 thinking tokens`)

// C1: P303a's interrupted phase (and P2's): its summary row reads "✗
// Thinking interrupted after 0.2s" with nothing after it; its details say there
// is no exact count and why, and keep the live estimate; its exit shows no "0
// thinking tokens" at any frame, in any size of the card.
func TestCorrectionC1Rows(t *testing.T) {
	for _, c := range []struct {
		probe   string
		summary string
		details func(x *state.Thought) []string
		est     bool
	}{
		{"p303a-claude-conversation.jsonl", "✗ Thinking interrupted after 0.2s", func(x *state.Thought) []string {
			return []string{
				"from " + x.Start.Local().Format("15:04:05.000") + " to " + x.End.Local().Format("15:04:05.000") + " · a thinking block, observed",
				"no exact count: the CLI reports 0 for an interrupted message",
				"live estimate 50 at its last update · system/thinking_tokens, estimated by the CLI",
				"no thinking text: the CLI sends none",
			}
		}, true},
		{"p2-claude-conversation.jsonl", "✗ Thinking interrupted after 0.2s", func(x *state.Thought) []string {
			return []string{
				"from " + x.Start.Local().Format("15:04:05.000") + " to " + x.End.Local().Format("15:04:05.000") + " · a thinking block, observed",
				"no exact count: the CLI reports 0 for an interrupted message",
				"no thinking text: the CLI sends none",
			}
		}, false},
	} {
		s := replayProbe(t, c.probe)
		i := cutEntry(t, s)
		e := s.Conversation[i]
		x := e.Thought
		text, _, _, _ := thoughtSummary(x)
		if got := "✗ " + text; got != c.summary {
			t.Errorf("%s: summary %q, want %q", c.probe, got, c.summary)
		}
		// Drawn as a tool row: the summary, then only the end time and the origin.
		var row string
		for _, r := range bRows(s, UI{}, 100) {
			if strings.Contains(plainRow(r), "Thinking interrupted") {
				row = strings.TrimRight(plainRow(r), " ")
			}
		}
		if !regexp.MustCompile(`^▎  ▸ ✗ Thinking interrupted after 0\.2s +[0-9:]{8}  observed$`).MatchString(row) {
			t.Errorf("%s: the row %q", c.probe, row)
		}
		if got := thoughtDetails(x); strings.Join(got, "\n") != strings.Join(c.details(x), "\n") {
			t.Errorf("%s: details\n%s\nwant\n%s", c.probe, strings.Join(got, "\n"), strings.Join(c.details(x), "\n"))
		}
		// The details of the full view and of the open row are the same lines.
		if got := entryLines(s, e); strings.Join(got[1:], "\n") != strings.Join(c.details(x), "\n") || got[0] != text {
			t.Errorf("%s: entry lines %q", c.probe, got)
		}
		// Its exit, every 10 ms of its first 100 and at every size of the card.
		ui := UI{Live: map[uint64]bool{uint64(x.ID): true}}
		for _, size := range []int{7, 5, 2} {
			for _, w := range []int{bInner(80), bInner(120), bInner(200)} {
				for ms := 0; ms < 300; ms += 10 {
					ui.Now = x.End.Add(time.Duration(ms) * time.Millisecond)
					for _, l := range cardAt(s, ui, i, w, size) {
						if r := plainRow(l); zeroCount.MatchString(r) {
							t.Errorf("%s: size %d, width %d, End+%d ms: a zero count in %q", c.probe, size, w, ms, r)
						}
					}
				}
			}
		}
	}
}

// C2: a phase closed by its turn's end, or the session's, is not reported as
// "not reported yet" (its count can no longer come); a phase whose turn is open
// still is. In P6b, whose last Codex phase ended on its own in a turn
// interrupted later, no phase reads "exact count not reported yet" once its
// turn ended (it was 1 of 66 rows over the recordings).
func TestCorrectionC2Details(t *testing.T) {
	// A recorded Codex phase has the line of its reasoning item between its
	// time and its count (plan 22, amendment 20): the count is the one after it.
	countDetail := func(d []string) string {
		if len(d) > 2 && strings.HasPrefix(d[1], "the reasoning item from ") {
			return d[2]
		}
		return d[1]
	}
	open := &state.Thought{Start: thinkAt(1000), End: thinkAt(4000), Confirmed: true}
	if d := thoughtDetails(open); d[1] != "exact count not reported yet" {
		t.Errorf("a phase whose count may still come: %q", d)
	}
	closed := *open
	closed.Closed = true
	if d := thoughtDetails(&closed); d[1] != "no exact count reported" {
		t.Errorf("a closed Claude Code phase: %q", d)
	}
	step := &state.Thought{Start: thinkAt(1000), End: thinkAt(4000), Inferred: true, Confirmed: true, Closed: true}
	if d := thoughtDetails(step); d[1] != "no exact count reported" {
		t.Errorf("a closed Codex phase: %q", d)
	}
	counted := &state.Thought{Start: thinkAt(1000), End: thinkAt(4000), Inferred: true, Confirmed: true, HasTokens: true, Tokens: 7}
	if d := thoughtDetails(counted); !strings.HasPrefix(d[1], "7 reasoning tokens · exact") {
		t.Errorf("a counted phase: %q", d)
	}

	s := replayProbe(t, "p6b-codex-app-server.jsonl")
	last := -1
	for _, i := range thoughtEntries(s) {
		x := s.Conversation[i].Thought
		if d := thoughtDetails(x); countDetail(d) == "exact count not reported yet" {
			t.Errorf("P6b: entry %d reads %q once its turn ended: %+v", i+1, countDetail(d), x)
		}
		if thoughtShown(x) && !x.HasTokens {
			last = i
		}
	}
	if last < 0 {
		t.Fatal("P6b: no phase without a count")
	}
	if d := thoughtDetails(s.Conversation[last].Thought); countDetail(d) != "no exact count reported" {
		t.Errorf("P6b's phase without a count: %q", d)
	}

	// Every recording, every phase: whatever its turn's end, none says "not
	// reported yet" once the turn is over.
	files, err := filepath.Glob(filepath.Join("..", "..", "testdata", "probes", "*.jsonl"))
	if err != nil || len(files) < 20 {
		t.Fatalf("recordings: %v, %d files", err, len(files))
	}
	rows := 0
	for _, f := range files {
		rs := replayProbe(t, filepath.Base(f))
		for _, i := range thoughtEntries(rs) {
			e := rs.Conversation[i]
			if !thoughtShown(e.Thought) {
				continue
			}
			rows++
			over := e.Stage > 0 && !rs.Stages[e.Stage-1].Open
			if d := thoughtDetails(e.Thought); over && countDetail(d) == "exact count not reported yet" {
				t.Errorf("%s: entry %d of an ended turn reads %q", filepath.Base(f), i+1, countDetail(d))
			}
		}
	}
	if rows < 40 {
		t.Errorf("%d phase rows in the recordings: the check saw too little", rows)
	}
}

// bench6 is a turn at work whose last entry is a finished phase and a running
// tool, with a subagent at work: the activity line's condition holds.
func bench6() *state.State {
	s := thinkBench(state.Thought{Start: thinkAt(2000), End: thinkAt(4000), Tokens: 120, HasTokens: true})
	s.Tools["t1"] = &state.ToolCall{ID: "t1", Name: "Bash", Status: state.ToolRunning, StartedAt: thinkAt(4100), Origin: event.Observed}
	s.Conversation = append(s.Conversation, state.Entry{Seq: 3, Role: "tool", MessageID: "t1", Text: "Bash", Origin: event.Observed, At: thinkAt(4100), Stage: 1})
	s.Agents["a1"] = &state.Agent{Key: "a1", Label: "general-purpose", Status: state.AgentRunning, StartedAt: thinkAt(4200), Stage: 1}
	s.Order = []string{"a1"}
	return s
}

// C6: the activity line is drawn once its condition has held 200 ms: not when
// the application noted no beginning, not at +150 ms, drawn at +200 ms; the
// condition is one function for the drawing and for the application, false
// under a card live or in its exit.
func TestCorrectionC6ActivityDelay(t *testing.T) {
	s := bench6()
	since := thinkAt(23000)
	drawn := func(ui UI) bool {
		return strings.Contains(strings.Join(rowTexts(bRows(s, ui, 100)), "\n"), "subagent working")
	}
	if drawn(UI{Now: thinkAt(24000)}) {
		t.Error("the line is drawn with no beginning noted")
	}
	for _, c := range []struct {
		d    time.Duration
		want bool
	}{{0, false}, {50, false}, {100, false}, {150, false}, {199, false}, {200, true}, {250, true}, {5000, true}} {
		ui := UI{Now: since.Add(c.d * time.Millisecond), ActivitySince: since}
		if got := drawn(ui); got != c.want {
			t.Errorf("+%d ms: drawn %v, want %v", c.d, got, c.want)
		}
		// One function for both: the drawing's choice is the condition, and the
		// beginning having held 200 ms.
		_, act := lookAll(s, ui)
		if want := ActivityOn(s, ui) && c.d >= 200; act != want {
			t.Errorf("+%d ms: lookAll says %v, the condition %v", c.d, act, ActivityOn(s, ui))
		}
	}
	// A beginning in the future, or a clock gone back, draws nothing.
	if drawn(UI{Now: since, ActivitySince: since.Add(time.Second)}) {
		t.Error("the line is drawn before its beginning")
	}

	// The condition: the turn works, and no card is live or in its exit.
	if !ActivityOn(s, UI{Now: thinkAt(24000)}) {
		t.Error("the condition does not hold on a turn at work")
	}
	live := bench6()
	live.Conversation = append(live.Conversation, state.Entry{Seq: 4, Role: "thinking", Streaming: true, At: thinkAt(20000), Stage: 1, Thought: &state.Thought{ID: 4, Start: thinkAt(20000)}})
	if ActivityOn(live, UI{Now: thinkAt(24000)}) {
		t.Error("the condition holds under a live card")
	}
	exit := bench6()
	exit.Conversation = append(exit.Conversation, state.Entry{Seq: 4, Role: "thinking", At: thinkAt(23800), Stage: 1, Thought: &state.Thought{ID: 4, Start: thinkAt(23800), End: thinkAt(23900)}})
	for _, c := range []struct {
		ms   int
		want bool
	}{{24000, false}, {24099, false}, {24100, true}} {
		if got := ActivityOn(exit, UI{Now: thinkAt(c.ms), Live: map[uint64]bool{4: true}}); got != c.want {
			t.Errorf("a card in its exit, %d ms: %v, want %v", c.ms, got, c.want)
		}
	}
	for _, mut := range []func(s *state.State){
		func(s *state.State) { s.Turn.Phase = state.Idle },
		func(s *state.State) { s.Session.Phase = state.Crashed },
		func(s *state.State) {
			s.Turn.Phase = state.WaitingApproval
			s.Approvals["ap"] = &state.Approval{ID: "ap", Tool: "Bash", Status: state.Pending, RequestedAt: thinkAt(5000)}
			s.Queue = []string{"ap"}
		},
		func(s *state.State) {
			s.Conversation = append(s.Conversation, state.Entry{Seq: 9, Role: "assistant", Text: "Here is", Streaming: true, At: thinkAt(6000), Stage: 1})
		},
	} {
		x := bench6()
		mut(x)
		if ActivityOn(x, UI{Now: thinkAt(24000)}) {
			t.Errorf("the condition holds although the turn does not work: turn %q session %q", x.Turn.Phase, x.Session.Phase)
		}
	}
}

// C7: the 2-row card chooses the count's form against the head it draws. An
// interrupted phase's exit head, "✗ Thinking interrupted after 0.2s", is longer
// than the spinner's, and at 80x24 an estimate of 2,700 in its long form ran
// past the row ("…"); P303a's own 50 fits. In the exit, Claude Code's count is
// the estimate without "live" (C8).
func TestCorrectionC7ExitRowNotCut(t *testing.T) {
	w := bInner(80) // 80x24
	for _, c := range []struct {
		name  string
		th    state.Thought
		short string // what the row must hold
	}{
		{"an estimate of 2,700", state.Thought{Start: thinkAt(1000), End: thinkAt(1200), Outcome: "interrupted", Cut: true, Estimate: 2700, EstimateFrom: 2450, EstimateAt: thinkAt(1100)},
			"✗ Thinking interrupted after 0.2s  ≈ 2,700 tokens · estimate"},
		{"an estimate of 50", state.Thought{Start: thinkAt(1000), End: thinkAt(1200), Outcome: "interrupted", Cut: true, Estimate: 50, EstimateAt: thinkAt(1100)},
			"✗ Thinking interrupted after 0.2s  ≈ 50 thinking tokens · estimate"},
		{"a Codex phase", state.Thought{Start: thinkAt(1000), End: thinkAt(1200), Inferred: true, Confirmed: true},
			"✓ Thought for up to 0.2s  Thinking · tokens after the step"},
	} {
		s := thinkBench(c.th)
		for ms := 0; ms < 100; ms += 10 {
			ui := UI{Now: thinkAt(1200 + ms), Live: map[uint64]bool{2: true}}
			rows := card(s, ui, w, 2)
			if len(rows) != 2 {
				t.Fatalf("%s: %d rows", c.name, len(rows))
			}
			r := plainRow(rows[0])
			if strings.Contains(r, "…") || !strings.HasPrefix(r, c.short) || textWidth(r) > cardContent(cardAI, w) {
				t.Errorf("%s, End+%d ms: the exit row %q (%d wide of %d), want %q … 00:00", c.name, ms, r, textWidth(r), cardContent(cardAI, w), c.short)
			}
		}
	}
	// P303a at 80x24, the criterion's own case.
	p := replayProbe(t, "p303a-claude-conversation.jsonl")
	i := cutEntry(t, p)
	x := p.Conversation[i].Thought
	for ms := 0; ms < 100; ms += 10 {
		ui := UI{Now: x.End.Add(time.Duration(ms) * time.Millisecond), Live: map[uint64]bool{uint64(x.ID): true}}
		r := plainRow(cardAt(p, ui, i, w, 2)[0])
		if strings.Contains(r, "…") || strings.Contains(r, "live") || !strings.HasPrefix(r, "✗ Thinking interrupted after 0.2s  ≈ 50 thinking tokens · estimate") {
			t.Errorf("P303a at 80x24, End+%d ms: %q", ms, r)
		}
	}
}

// C8: in its exit, Claude Code's count row reads the exact count once it came,
// else the live text without "live"; the live card, Codex's exit and mode A's
// are as they were. Never "live estimate" or "token usage pending" in an exit
// once a count is reported.
func TestCorrectionC8ExitCountRow(t *testing.T) {
	end := thinkAt(21351)
	claude := func(mut func(th *state.Thought)) *state.State {
		th := state.Thought{Start: thinkAt(7449), End: end, Estimate: 2700, EstimateFrom: 2450, EstimateAt: thinkAt(20632)}
		mut(&th)
		return thinkBench(th)
	}
	countOf := func(s *state.State, ms, size, w int) string {
		ui := UI{Now: end.Add(time.Duration(ms) * time.Millisecond), Live: map[uint64]bool{2: true}}
		rows := card(s, ui, w, size)
		if size == 2 {
			return plainRow(rows[0])
		}
		return strings.TrimSpace(strings.Trim(plainRow(rows[map[int]int{7: 3, 5: 2}[size]]), "│ "))
	}
	exact := claude(func(th *state.Thought) {
		th.Tokens, th.HasTokens, th.TokensAt = 2852, true, end.Add(24*time.Millisecond)
	})
	for _, c := range []struct {
		name string
		s    *state.State
		size int
		w    int
		ms   int
		want string // the count row, or the 2-row card's row from its head
	}{
		{"the exact count came, long", exact, 7, 90, 50, "2,852 thinking tokens"},
		{"the exact count came, 5 rows", exact, 5, 90, 99, "2,852 thinking tokens"},
		{"the exact count came, short", exact, 2, 40, 50, "✓ Thought for 14s  2,852 tokens"},
		{"the exact count came, 2 rows, room", exact, 2, 80, 50, "✓ Thought for 14s  2,852 thinking tokens"},
		{"only an estimate, long", claude(func(*state.Thought) {}), 7, 90, 50, "≈ 2,700 thinking tokens · estimate"},
		{"only an estimate, short", claude(func(*state.Thought) {}), 2, 56, 50, "✓ Thought for 14s  ≈ 2,700 tokens · estimate"},
		{"only an estimate, 2 rows, room", claude(func(*state.Thought) {}), 2, 80, 50, "✓ Thought for 14s  ≈ 2,700 thinking tokens · estimate"},
		{"neither", claude(func(th *state.Thought) { th.Estimate, th.EstimateFrom, th.EstimateAt = 0, 0, time.Time{} }), 7, 90, 50, "Thinking · token usage pending"},
		{"a budget (mode A)", claude(func(th *state.Thought) { th.Budget = 4096 }), 7, 90, 50, "Thinking · 2,700 / 4,096 tokens"},
		{"a budget, the count came", claude(func(th *state.Thought) { th.Budget, th.Tokens, th.HasTokens = 4096, 2852, true }), 7, 90, 50, "Thinking · 2,852 / 4,096 tokens"},
	} {
		got := countOf(c.s, c.ms, c.size, c.w)
		if c.size == 2 {
			if !strings.HasPrefix(got, c.want) || strings.Contains(got, "…") {
				t.Errorf("%s: the row %q, want %q …", c.name, got, c.want)
			}
		} else if !strings.HasPrefix(got, c.want) {
			t.Errorf("%s: the count row %q, want %q", c.name, got, c.want)
		}
	}
	// The exact count is lit while it is new (within 400 ms of its arrival),
	// bright after; the estimate keeps its own highlight.
	for _, c := range []struct {
		ms int
		st *lipgloss.Style
	}{{50, stThinkLit}, {99, stThinkLit}} {
		ui := UI{Now: end.Add(time.Duration(c.ms) * time.Millisecond), Live: map[uint64]bool{2: true}}
		x, ok := spanOf(card(exact, ui, 90, 7)[3], "2,852")
		if !ok || x.st != c.st {
			t.Errorf("End+%d ms: the exact count %q styled %p, want lit", c.ms, x.text, x.st)
		}
	}
	old := claude(func(th *state.Thought) { th.Tokens, th.HasTokens, th.TokensAt = 2852, true, end.Add(-2*time.Second) })
	ui := UI{Now: end.Add(50 * time.Millisecond), Live: map[uint64]bool{2: true}}
	if x, ok := spanOf(card(old, ui, 90, 7)[3], "2,852"); !ok || x.st != stBrightBold {
		t.Errorf("an exact count that came long before: %q styled %p, want bright", x.text, x.st)
	}

	// The live card is as it was: the estimate with "live", whatever the count.
	live := thinkBench(state.Thought{Start: thinkAt(7449), Estimate: 2700, EstimateFrom: 2450, EstimateAt: thinkAt(20632)})
	if got := strings.TrimSpace(strings.Trim(plainRow(card(live, UI{Now: thinkAt(21000)}, 90, 7)[3]), "│ ")); got != "≈ 2,700 thinking tokens · live estimate" {
		t.Errorf("the live card: %q", got)
	}
	// Codex's exit is as it was.
	codexExit := thinkBench(state.Thought{Start: thinkAt(1000), End: thinkAt(4000), Inferred: true, Confirmed: true, Tokens: 83, HasTokens: true})
	ui = UI{Now: thinkAt(4050), Live: map[uint64]bool{2: true}}
	if got := plainRow(card(codexExit, ui, 90, 7)[3]); !strings.Contains(got, "Thinking · reasoning tokens reported when the step ends") {
		t.Errorf("Codex's exit: %q", got)
	}
}

// C9: at 80 columns a phase's summary row that is too long for its end time and
// origin drops the origin whole, never "obse…"; a row that fits keeps both; a
// tool row is cut as it always was.
func TestCorrectionC9SummaryOrigin(t *testing.T) {
	w := cardContent(cardAI, bInner(80)) // 80x24
	long := state.Thought{Start: thinkAt(0), End: thinkAt(13902), Estimate: 2700, EstimateAt: thinkAt(13000)}
	short := state.Thought{Start: thinkAt(0), End: thinkAt(13902), Tokens: 2852, HasTokens: true}
	for _, c := range []struct {
		name   string
		th     state.Thought
		w      int
		origin bool
	}{
		{"the estimate form at 80x24", long, w, false},
		{"the count at 80x24", short, w, true},
		{"the estimate form, room", long, 120, true},
		{"the count, only just room", short, 65, true},
		{"the count, one cell short", short, 64, false},
	} {
		s := thinkBench(c.th)
		s.Turn.Phase = state.Idle
		row := strings.TrimRight(plainRow(summaryRow(s, UI{}, 1, s.Conversation[1], c.w, look{}, false, 0)), " ")
		clock := clockSec(c.th.End)
		if textWidth(row) > c.w {
			t.Errorf("%s: the row is %d wide for %d: %q", c.name, textWidth(row), c.w, row)
		}
		switch {
		case strings.Contains(row, "…"):
			t.Errorf("%s: a cut row %q", c.name, row)
		case c.origin && !strings.HasSuffix(row, clock+"  observed"):
			t.Errorf("%s: %q, want the end time and the origin at the right", c.name, row)
		case !c.origin && !strings.HasSuffix(row, clock):
			t.Errorf("%s: %q, want the end time alone at the right", c.name, row)
		case !c.origin && strings.Contains(row, "observed"):
			t.Errorf("%s: the origin is kept: %q", c.name, row)
		}
	}
	// So narrow that even the end time does not fit: the existing cut applies.
	s := thinkBench(long)
	s.Turn.Phase = state.Idle
	if row := plainRow(summaryRow(s, UI{}, 1, s.Conversation[1], 50, look{}, false, 0)); textWidth(strings.TrimRight(row, " ")) > 50 {
		t.Errorf("a row wider than its room: %q", row)
	}
	// The entry cursor's hint replaces the end time and the origin, as before.
	if row := plainRow(summaryRow(s, UI{}, 1, s.Conversation[1], 74, look{}, true, 0)); !strings.Contains(row, "Enter") || strings.Contains(row, "observed") {
		t.Errorf("the focused row: %q", row)
	}

	// A tool row is cut as it always was: its name whole, the right part cut.
	got := plainRow(toolLine(deco("▸", stFoldMark), line{deco(strings.Repeat("x", 58), stText)}, nil, 58, nil, false, decoLine("02:00:09", stGrey), "observed", 78))
	if !strings.HasSuffix(strings.TrimRight(got, " "), "02:00:09  obse…") {
		t.Errorf("a long tool row: %q, want its origin cut as it always was", got)
	}
}

// C9 on the recordings: at 80x24 no summary row of a phase is cut, in any
// recording of both CLIs, at the end of the replay (the application's own check
// on every frame is in internal/app).
func TestCorrectionC9Recordings(t *testing.T) {
	files, err := os.ReadDir(filepath.Join("..", "..", "testdata", "probes"))
	if err != nil {
		t.Fatal(err)
	}
	seen := 0
	for _, f := range files {
		if !strings.HasSuffix(f.Name(), ".jsonl") {
			continue
		}
		s := replayProbe(t, f.Name())
		for _, r := range bRows(s, UI{Width: 80, Height: 24}, bInner(80)) {
			if row := plainRow(r); strings.Contains(row, "▸ ✓ Thought") || strings.Contains(row, "▸ ✗ Thinking") {
				seen++
				if strings.Contains(row, "…") {
					t.Errorf("%s: a cut summary row %q", f.Name(), row)
				}
			}
		}
	}
	if seen < 30 {
		t.Errorf("%d summary rows in the recordings: the check saw too little", seen)
	}
}

// C8 on the recordings where the message's count comes within 100 ms of the
// block's stop: P303e's second phase (46 tokens, 11 ms after), whose exit read
// "Thinking · token usage pending" with the count already reported, reads the
// count, in the 7-row card and in the 2-row card. P303a's interrupted phase
// (D9's 0 ms, now a message cut short with no count) reads its estimate without
// "live"; P2's, with no estimate either, says that no exact count is reported.
// None says "live estimate" or "token usage pending".
func TestCorrectionC8RecordedExits(t *testing.T) {
	check := func(probe string, i int, now time.Time, want, want2 string) {
		t.Helper()
		s := replayProbe(t, probe)
		x := s.Conversation[i].Thought
		ui := UI{Now: now, Live: map[uint64]bool{uint64(x.ID): true}}
		if got := strings.TrimSpace(strings.Trim(plainRow(cardAt(s, ui, i, 90, 7)[3]), "│ ")); !strings.HasPrefix(got, want) {
			t.Errorf("%s: the exit's count row %q, want %q", probe, got, want)
		}
		row := plainRow(cardAt(s, ui, i, bInner(80), 2)[0])
		if !strings.Contains(row, want2) || strings.Contains(row, "…") {
			t.Errorf("%s: the 2-row exit %q, want %q", probe, row, want2)
		}
		for _, size := range []int{7, 5, 2} {
			for _, l := range cardAt(s, ui, i, 90, size) {
				if r := plainRow(l); strings.Contains(r, "live estimate") || strings.Contains(r, "token usage pending") {
					t.Errorf("%s: size %d: %q", probe, size, r)
				}
			}
		}
	}
	e := replayProbe(t, "p303e-claude-interrupt-tool.jsonl")
	counted := 0
	for _, i := range thoughtEntries(e) {
		if x := e.Conversation[i].Thought; x.HasTokens && x.TokensAt.Sub(x.End) < 100*time.Millisecond {
			counted++
			check("p303e-claude-interrupt-tool.jsonl", i, x.TokensAt.Add(10*time.Millisecond), "46 thinking tokens", "46 thinking tokens")
		}
	}
	if counted != 1 {
		t.Errorf("P303e: %d phases with a count within 100 ms of their stop, want 1", counted)
	}
	a := replayProbe(t, "p303a-claude-conversation.jsonl")
	i := cutEntry(t, a)
	check("p303a-claude-conversation.jsonl", i, a.Conversation[i].Thought.End.Add(50*time.Millisecond), "≈ 50 thinking tokens · estimate", "≈ 50 thinking tokens · estimate")
	// P2's message cut short has no estimate and no count to come: its exit says
	// so, not that the usage is pending (the 2-row card at 80x24 in its short form).
	p := replayProbe(t, "p2-claude-conversation.jsonl")
	j := cutEntry(t, p)
	check("p2-claude-conversation.jsonl", j, p.Conversation[j].Thought.End.Add(50*time.Millisecond), "Thinking · no exact count reported", "Thinking · no count")
}

// C5 at the render level: P9a's second phase, a reasoning item that found no
// step open, reads "Thought for up to 1.4s · 118 reasoning tokens" with the
// step's start in its details, and its exit is timed (00:01), not "--:--";
// P6b's second phase gains its start the same way.
func TestCorrectionC5Rows(t *testing.T) {
	for _, c := range []struct {
		probe   string
		entry   int // the thinking entry, 1-based
		summary string
		timer   string
	}{
		{"p9a-codex-spawn-request.jsonl", 2, "Thought for up to 1.4s · 118 reasoning tokens", "00:01"},
		{"p6b-codex-app-server.jsonl", 2, "Thought for up to 1.6s · 7 reasoning tokens", "00:01"},
	} {
		s := replayProbe(t, c.probe)
		i := thoughtEntries(s)[c.entry-1]
		x := s.Conversation[i].Thought
		if text, _, _, _ := thoughtSummary(x); text != c.summary {
			t.Errorf("%s: summary %q, want %q", c.probe, text, c.summary)
		}
		if d := thoughtDetails(x); !strings.HasPrefix(d[0], "the model step from ") || strings.Contains(d[0], "its start not known") {
			t.Errorf("%s: details %q", c.probe, d[0])
		}
		for ms := 0; ms < 100; ms += 20 {
			ui := UI{Now: x.End.Add(time.Duration(ms) * time.Millisecond), Live: map[uint64]bool{uint64(x.ID): true}}
			for _, size := range []int{7, 2} {
				rows := cardAt(s, ui, i, bInner(120), size)
				head := map[int]int{7: 1, 2: 0}[size]
				if r := plainRow(rows[head]); strings.Contains(r, "--:--") || !strings.Contains(r, c.timer) {
					t.Errorf("%s: size %d, End+%d ms: the exit's head %q, want the time %s", c.probe, size, ms, r, c.timer)
				}
			}
		}
	}
}

// C4 at the render level: a reasoning item that the next step's start ended
// says so in its details, "when the next step began (inferred)"; a phase its
// turn's end cut says "when its turn ended (inferred)"; one that completed
// gives its item's end.
func TestCorrectionC4Details(t *testing.T) {
	base := state.Thought{Start: thinkAt(1000), End: thinkAt(4000), Inferred: true, Confirmed: true}
	at := func(x time.Time) string { return x.Local().Format("15:04:05.000") }
	for _, c := range []struct {
		name string
		mut  func(th *state.Thought)
		want string
	}{
		{"the next step's start", func(th *state.Thought) { th.EndInferred, th.EndNext = true, true },
			"the model step from " + at(base.Start) + " (inferred) to " + at(base.End) + ", when the next step began (inferred)"},
		{"its turn's end", func(th *state.Thought) { th.EndInferred = true },
			"the model step from " + at(base.Start) + " (inferred) to " + at(base.End) + ", when its turn ended (inferred)"},
		{"its own completion", func(th *state.Thought) {},
			"the model step from " + at(base.Start) + " (inferred) to its reasoning's end at " + at(base.End) + " · reasoning item"},
	} {
		th := base
		c.mut(&th)
		if got := thoughtDetails(&th)[0]; got != c.want {
			t.Errorf("%s: %q, want %q", c.name, got, c.want)
		}
	}
}

// C7 completed: when neither the long nor the short form of the count fits
// beside the head the 2-row card draws, the count is left out of the row, never
// cut: the head and the time stay whole. A Codex phase interrupted or failed at
// 80x24 (its short form "Thinking · tokens after the step" does not fit beside
// "Thinking interrupted after up to 0.2s"), and a Claude Code phase cut short
// after an hour with an estimate of 12,345,678.
func TestCorrectionC7CountLeftOut(t *testing.T) {
	w := bInner(80)
	codexOver := func(outcome string) state.Thought {
		return state.Thought{Start: thinkAt(1000), End: thinkAt(1200), Outcome: outcome, Inferred: true, Confirmed: true, EndInferred: true}
	}
	for _, c := range []struct {
		name   string
		th     state.Thought
		head   string // the head, whole
		timer  string
		absent string // a part of the count that must not be drawn; "" for none
	}{
		{"a Codex phase interrupted", codexOver("interrupted"), "✗ Thinking interrupted after up to 0.2s", "00:00", "Thinking ·"},
		{"a Codex phase failed", codexOver("failed"), "✗ Thinking stopped after up to 0.2s", "00:00", "Thinking ·"},
		{"a cut phase after an hour", state.Thought{Start: thinkAt(0), End: thinkAt(3665000), Outcome: "interrupted", Cut: true, Estimate: 12345678, EstimateAt: thinkAt(3660000)},
			"✗ Thinking interrupted after 61m 05s", "1:01:05", "tokens"},
	} {
		s := thinkBench(c.th)
		for ms := 0; ms < 100; ms += 10 {
			ui := UI{Now: c.th.End.Add(time.Duration(ms) * time.Millisecond), Live: map[uint64]bool{2: true}}
			rows := card(s, ui, w, 2)
			r := plainRow(rows[0])
			if strings.Contains(r, "…") || !strings.HasPrefix(r, c.head) || !strings.HasSuffix(strings.TrimRight(r, " "), c.timer) || textWidth(r) > cardContent(cardAI, w) {
				t.Errorf("%s, End+%d ms: the exit row %q (%d wide of %d), want %q … %s whole", c.name, ms, r, textWidth(r), cardContent(cardAI, w), c.head, c.timer)
			}
			if strings.Contains(r, c.absent) {
				t.Errorf("%s, End+%d ms: the count is in the row %q", c.name, ms, r)
			}
		}
	}
	// The same phases in a room where the short form fits keep it.
	s := thinkBench(codexOver("interrupted"))
	ui := UI{Now: thinkAt(1250), Live: map[uint64]bool{2: true}}
	if r := plainRow(card(s, ui, 100, 2)[0]); !strings.Contains(r, "Thinking · tokens after the step") || strings.Contains(r, "…") {
		t.Errorf("a Codex phase interrupted with room: %q", r)
	}
	// The live card's first row, with its count in the same short form, is as it was.
	live := thinkBench(state.Thought{Start: thinkAt(1000), Inferred: true, Confirmed: true})
	if r := plainRow(card(live, UI{Now: thinkAt(3000)}, w, 2)[0]); !strings.Contains(r, "Thinking · tokens after the step") || strings.Contains(r, "…") {
		t.Errorf("a Codex step live at 80x24: %q", r)
	}
}

// C8 completed: in its exit, a Claude Code phase whose count can no longer come
// (a message cut short, a turn or session ended) and that has no estimate reads
// "Thinking · no exact count reported", short "Thinking · no count", not "token
// usage pending"; a phase still waiting for its message_delta keeps "token usage
// pending"; an estimate, a count, Codex's and mode A's texts win or stay.
func TestCorrectionC8NoExactCount(t *testing.T) {
	end := thinkAt(5000)
	base := state.Thought{Start: thinkAt(1000), End: end}
	rowOf := func(th state.Thought, size, w int) string {
		s := thinkBench(th)
		rows := card(s, UI{Now: end.Add(50 * time.Millisecond), Live: map[uint64]bool{2: true}}, w, size)
		if size == 2 {
			return plainRow(rows[0])
		}
		return strings.TrimSpace(strings.Trim(plainRow(rows[map[int]int{7: 3, 5: 2}[size]]), "│ "))
	}
	with := func(mut func(th *state.Thought)) state.Thought { th := base; mut(&th); return th }
	for _, c := range []struct {
		name string
		th   state.Thought
		long string
		// The 2-row card in a room that holds only the short form.
		short string
	}{
		{"a message cut short", with(func(th *state.Thought) { th.Cut, th.Outcome = true, "interrupted" }), "Thinking · no exact count reported", "Thinking · no count"},
		{"a turn that ended", with(func(th *state.Thought) { th.Closed = true }), "Thinking · no exact count reported", "Thinking · no count"},
		{"a turn that failed", with(func(th *state.Thought) { th.Closed, th.Outcome = true, "failed" }), "Thinking · no exact count reported", "Thinking · no count"},
		{"still waiting for its message_delta", with(func(th *state.Thought) {}), "Thinking · token usage pending", "Thinking · usage pending"},
		{"cut short with an estimate", with(func(th *state.Thought) { th.Cut, th.Estimate = true, 650 }), "≈ 650 thinking tokens · estimate", "≈ 650 tokens · estimate"},
		{"closed with a count", with(func(th *state.Thought) { th.Closed, th.HasTokens, th.Tokens = true, true, 90 }), "90 thinking tokens", "90 tokens"},
		{"a Codex step closed", with(func(th *state.Thought) { th.Inferred, th.Confirmed, th.Closed = true, true, true }),
			"Thinking · reasoning tokens reported when the step ends", "Thinking · tokens after the step"},
		{"mode A, closed", with(func(th *state.Thought) { th.Budget, th.Estimate, th.Closed = 4096, 900, true }), "Thinking · 900 / 4,096 tokens", "900 / 4,096 tokens"},
	} {
		if got := rowOf(c.th, 7, 90); !strings.HasPrefix(got, c.long) {
			t.Errorf("%s: the count row %q, want %q", c.name, got, c.long)
		}
		// The 2-row card: the long form with room, then in a room that holds only the
		// short one, the short form, the row whole.
		if got := rowOf(c.th, 2, 100); !strings.Contains(got, c.long) {
			t.Errorf("%s: the 2-row exit %q, want %q", c.name, got, c.long)
		}
		found := false
		for cw := 100; cw >= 20 && !found; cw-- {
			r := rowOf(c.th, 2, cw)
			found = strings.Contains(r, c.short) && !strings.Contains(r, c.long) && !strings.Contains(r, "…")
		}
		if !found {
			t.Errorf("%s: no width gives the short form %q whole", c.name, c.short)
		}
	}
}

// twoBlockMessage is TestThinkingMessageWithTwoBlocks's synthetic message, run
// through the reducer: m1 with a thinking block at index 0 (1,000 to 2,000 ms)
// and another at index 2 (3,000 to 4,000 ms), and one count of 700 for both,
// 5 ms after the second block's stop.
func twoBlockMessage(t *testing.T) *state.State {
	t.Helper()
	s := state.New()
	var seq uint64
	apply := func(ms int, kind event.Kind, p any) {
		seq++
		s.Apply(event.Event{Seq: seq, At: thinkAt(ms), Provider: "test", Kind: kind, Origin: event.Observed, Payload: p})
	}
	s.Do(state.Launch{Mock: "x"})
	apply(0, event.SessionSpawned, event.Spawned{})
	apply(0, event.SessionReady, event.Ready{})
	s.Do(state.Send{Text: "think"})
	apply(0, event.UserMessage, event.User{Text: "think"})
	apply(0, event.TurnStarted, nil)
	apply(1000, event.MessageMarker, event.Marker{MessageID: "m1", Name: "content_block_start", Block: "thinking", Index: 0})
	apply(2000, event.MessageMarker, event.Marker{MessageID: "m1", Name: "content_block_stop", Index: 0})
	apply(3000, event.MessageMarker, event.Marker{MessageID: "m1", Name: "content_block_start", Block: "thinking", Index: 2})
	apply(4000, event.MessageMarker, event.Marker{MessageID: "m1", Name: "content_block_stop", Index: 2})
	apply(4005, event.MessageMarker, event.Marker{MessageID: "m1", Name: "message_delta", StopReason: "tool_use", Thinking: 700, HasThinking: true})
	return s
}

// C8 completed: in the exit, an exact count shared by several phases reads as
// the summary row says it, "700 thinking tokens over 2 phases", short "700
// tokens over 2 phases".
func TestCorrectionC8SharedCount(t *testing.T) {
	s := twoBlockMessage(t)
	th := thoughtEntries(s)
	if len(th) != 2 {
		t.Fatalf("%d phases, want 2", len(th))
	}
	i := th[1]
	x := s.Conversation[i].Thought
	if !x.HasTokens || x.Tokens != 700 || x.Shared != 2 || s.Conversation[th[0]].Thought.HasTokens {
		t.Fatalf("not the scenario: %+v", x)
	}
	if text, _, _, _ := thoughtSummary(x); !strings.HasSuffix(text, " · 700 thinking tokens over 2 phases") {
		t.Errorf("summary %q", text)
	}
	ui := UI{Now: x.End.Add(20 * time.Millisecond), Live: map[uint64]bool{uint64(x.ID): true}}
	const long, short = "700 thinking tokens over 2 phases", "700 tokens over 2 phases"
	if got := strings.TrimSpace(strings.Trim(plainRow(cardAt(s, ui, i, 90, 7)[3]), "│ ")); !strings.HasPrefix(got, long) {
		t.Errorf("the 7-row exit's count row %q, want %q", got, long)
	}
	if got := plainRow(cardAt(s, ui, i, 100, 2)[0]); !strings.Contains(got, long) || strings.Contains(got, "…") {
		t.Errorf("the 2-row exit with room: %q, want %q", got, long)
	}
	found := false
	for w := 100; w >= 20 && !found; w-- {
		r := plainRow(cardAt(s, ui, i, w, 2)[0])
		found = strings.Contains(r, short) && !strings.Contains(r, long) && !strings.Contains(r, "…")
	}
	if !found {
		t.Errorf("no width gives the short form %q whole", short)
	}
	// A count that is one phase's own says nothing of phases.
	own := thinkBench(state.Thought{Start: thinkAt(1000), End: thinkAt(2000), Tokens: 700, HasTokens: true, TokensAt: thinkAt(2005)})
	if got := strings.TrimSpace(strings.Trim(plainRow(card(own, UI{Now: thinkAt(2020), Live: map[uint64]bool{2: true}}, 90, 7)[3]), "│ ")); !strings.HasPrefix(got, "700 thinking tokens") || strings.Contains(got, "phases") {
		t.Errorf("a phase's own count: %q", got)
	}
}

// C9 follows toolLine: for every width, a phase's summary row is the row
// toolLine draws when the origin fits, and the row toolLine draws without an
// origin when it would not, so the rule cannot drift between the two. The
// reference is toolLine itself; the origin is cut when its text is not whole.
func TestCorrectionC9FollowsToolLine(t *testing.T) {
	s := thinkBench(state.Thought{Start: thinkAt(0), End: thinkAt(13902), Estimate: 2700, EstimateAt: thinkAt(13000)})
	s.Turn.Phase = state.Idle
	e := s.Conversation[1]
	text, _, _, _ := thoughtSummary(e.Thought)
	mark := deco("▸", stFoldMark)
	name := line{thoughtMark(e.Thought), deco(" ", stNone), deco(text, stText)}
	outcome := decoLine(clockSec(e.Thought.End), stGrey)
	kept, dropped := 0, 0
	for w := 30; w <= 140; w++ {
		row := plainRow(summaryRow(s, UI{}, 1, e, w, look{}, false, 0))
		with := plainRow(toolLine(mark, name, nil, 0, nil, false, outcome, "observed", w))
		want := plainRow(toolLine(mark, name, nil, 0, nil, false, outcome, "", w))
		if strings.Contains(with, "observed") {
			kept++
			want = with
		} else {
			dropped++
		}
		if row != want {
			t.Errorf("width %d: the summary row\n%q\nwant\n%q", w, row, want)
		}
	}
	if kept < 10 || dropped < 10 {
		t.Errorf("%d widths keep the origin and %d drop it: the check saw too little", kept, dropped)
	}
}
