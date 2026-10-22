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
	"github.com/Evengelius/agent-command-center/internal/fixture"
	"github.com/Evengelius/agent-command-center/internal/provider/claude"
	"github.com/Evengelius/agent-command-center/internal/provider/mock"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// Plan 24 (brief amendment 22, the owner's image): zone A with its heading
// inside the frame, the model row with the status at its end, the stages as
// blocks, and the telemetry pane at its bottom; zone D gone.

// appClock is the application tests' clock, so the synthetic quota mock's
// countdowns read as its golden frames do (mock_test.go, testNow).
var appClock = time.Date(2026, 10, 15, 9, 30, 0, 0, time.UTC)

// stampedMock is a mock file's state with its events stamped from base, as
// mock mode stamps them, and the frame time a second after the last.
func stampedMock(t *testing.T, name string, base time.Time) (*state.State, time.Time) {
	t.Helper()
	f := filepath.Join("..", "..", "testdata", "mock", name)
	evs, err := mock.Load(f)
	if err != nil {
		t.Fatal(err)
	}
	for i, e := range evs {
		evs[i].Event.At, evs[i].Event.Seq = base.Add(e.After), uint64(i+1) // as the mock session stamps them
	}
	return mockState(f, evs)
}

// zoneARows are zone A's inner rows of a dashboard frame, trimmed right,
// and the frame's rows.
func zoneARows(s *state.State, ui UI) (inner, frame []string) {
	frame = strings.Split(strings.TrimSuffix(texts(Draw(s, ui).lines), "\n"), "\n")
	a := zoneBoxes(frame)["A"]
	for _, r := range frame[a.top+1 : a.bottom] {
		cells := []rune(r)
		inner = append(inner, strings.TrimRight(string(cells[a.left+2:a.right-1]), " "))
	}
	return inner, frame
}

func indexOf(rows []string, prefix string) int {
	for i, r := range rows {
		if strings.HasPrefix(r, prefix) {
			return i
		}
	}
	return -1
}

// TestZoneAStructure is criteria 24.4 and 24.5: the rows in order, the room
// the tree keeps, the pane's forms, a folded section.
func TestZoneAStructure(t *testing.T) {
	s, now := stampedMock(t, "claude-subagent-approval.jsonl", appClock)
	for _, size := range [][2]int{{120, 40}, {208, 59}} {
		ui := UI{Width: size[0], Height: size[1], Now: now, Focus: "B"}
		rows, frame := zoneARows(s, ui)
		if !strings.HasPrefix(frame[2], "╭──") || strings.Contains(frame[2], "A · ") {
			t.Errorf("%v: zone A's top edge %q, want no title in it", size, frame[2])
		}
		if rows[0] != "A · ORCHESTRATOR" || !strings.HasPrefix(rows[1], "✿ ") || strings.Trim(rows[2], "─") != "" || rows[3] != "STAGES · inferred from activity" {
			t.Errorf("%v: zone A's first rows %q", size, rows[:4])
		}
		tele := indexOf(rows, "✦ TELEMETRY")
		usage, quota := indexOf(rows, "▾ SESSION USAGE"), indexOf(rows, "▾ QUOTA WINDOWS")
		if tele < 0 || usage != tele+1 || quota < usage || strings.Trim(rows[tele-1], "─") != "" {
			t.Errorf("%v: TELEMETRY at %d, SESSION USAGE at %d, QUOTA WINDOWS at %d:\n%s", size, tele, usage, quota, strings.Join(rows, "\n"))
		}
		// Plan 26: the pane ends with the last window's rows, no rule and no
		// notes after them; the tree has the 3 rows the full form's notes took.
		if tree := tele - 1 - 4; tree < 6 || tree < map[int]int{120: 9, 208: 24}[size[0]] {
			t.Errorf("%v: the tree has %d rows, want at least 6 and plan 24's 3 more", size, tree)
		}
		if last := rows[len(rows)-1]; !strings.HasPrefix(last, "  Resets in ") || strings.Contains(strings.Join(rows, "\n"), "ⓘ") {
			t.Errorf("%v: the pane's last row %q, want the last window's countdown and no notes:\n%s", size, last, strings.Join(rows, "\n"))
		}
		if n := strings.Count(strings.Join(rows, "\n"), "Resets in "); n != 2 {
			t.Errorf("%v: %d \"Resets in\" rows, want the full form's 2", size, n)
		}
		for i, r := range frame {
			if strings.Contains(r, "D · ") || strings.Contains(r, "QUOTA & TELEMETRY") {
				t.Errorf("%v: zone D's title on row %d", size, i)
			}
		}
	}
	// 24.5 At 100x24 the tight form: a window on one row with its percentage
	// and countdown; the tree keeps at least 3 rows.
	ui := UI{Width: 100, Height: 24, Now: now, Focus: "B"}
	rows, _ := zoneARows(s, ui)
	tele := indexOf(rows, "✦ TELEMETRY")
	tight := regexp.MustCompile(`^  Session \(5hr\)  \d+%†? · \S+`)
	found := false
	for _, r := range rows {
		found = found || tight.MatchString(r)
	}
	if !found || tele-1-4 < 3 || strings.Contains(strings.Join(rows, "\n"), "Resets in ") {
		t.Errorf("100x24: want the tight form and 3 tree rows, TELEMETRY at %d:\n%s", tele, strings.Join(rows, "\n"))
	}
	// SESSION USAGE folded: its three rows go to the tree.
	ui = UI{Width: 120, Height: 40, Now: now, Focus: "B"}
	open, _ := zoneARows(s, ui)
	ui.Fold = map[string]bool{NodeUsage: true}
	folded, _ := zoneARows(s, ui)
	if o, f := indexOf(open, "✦ TELEMETRY"), indexOf(folded, "✦ TELEMETRY"); f != o+3 || indexOf(folded, "▸ SESSION USAGE") != f+1 || strings.Contains(strings.Join(folded, "\n"), "Context") {
		t.Errorf("SESSION USAGE folded: TELEMETRY at %d, was %d:\n%s", f, o, strings.Join(folded, "\n"))
	}
}

// TestOrchestratorRow is criteria 24.6 to 24.9: the model's title and icon,
// the status at the row's end, each state's status, and the agents still
// working once the turn ended.
func TestOrchestratorRow(t *testing.T) {
	claudeMock, now := stampedMock(t, "claude-subagent-approval.jsonl", appClock)
	codexMock, _ := stampedMock(t, "codex-subagents.jsonl", appClock)
	for _, c := range []struct {
		s          *state.State
		lead, icon string
	}{{claudeMock, "✿ Claude Haiku 5.5", "✿"}, {codexMock, "◆ GPT-5.6-sol", "◆"}} {
		if got := texts([]line{modelLead(c.s, true)}); strings.TrimSpace(got) != c.lead {
			t.Errorf("model lead %q, want %q", got, c.lead)
		}
		if x, ok := spanOf(modelLead(c.s, true), c.icon+" "); !ok || x.st != modelStyle(model(c.s)) {
			t.Errorf("the icon's style %v, want the model's", x.st)
		}
	}
	starting := state.New()
	starting.Context.Provider = "claude-code"
	starting.Session.Phase = state.Spawning
	if got := texts([]line{modelLead(starting, true)}); strings.TrimSpace(got) != "model N/A (session starting)" {
		t.Errorf("before the model is known: %q", got)
	}
	for id, want := range map[string]string{"claude-sonnet-4-5-20250929": "Claude Sonnet 4.5", "claude-fable-5-1": "Claude Fable 5.1", "claude-mythos-5-1": "Claude Mythos 5.1", "claude-next": "claude-next"} {
		if got := modelTitle(id); got != want {
			t.Errorf("modelTitle(%s) = %q, want %q", id, got, want)
		}
	}

	// 24.7 The status ends the model row at every width; the short name
	// where the title does not fit.
	for _, w := range []int{32, 40, 52, 54} {
		row := modelRow(claudeMock, UI{Now: now}, w)
		if row.width() != w || !strings.HasSuffix(strings.TrimRight(texts([]line{row}), "\n"), "✓ COMPLETED") {
			t.Errorf("width %d: row %q", w, texts([]line{row}))
		}
	}
	waiting := p3bAtWrite(t)
	row := strings.TrimRight(texts([]line{modelRow(waiting, UI{Now: now}, 32)}), "\n")
	if !strings.HasPrefix(row, "✿ Haiku 5.5 ") || !strings.HasSuffix(row, "! AGENT WAITING") {
		t.Errorf("at 32 cells with an agent waiting: %q", row)
	}
	if row := strings.TrimRight(texts([]line{modelRow(waiting, UI{Now: now}, 54)}), "\n"); !strings.HasPrefix(row, "✿ Claude Haiku 5.5 ") {
		t.Errorf("at 54 cells: %q", row)
	}

	// 24.8 Each state's status, text and style.
	ready := func() *state.State {
		s := state.New()
		s.Context.Provider = "claude-code"
		s.Session.Phase = state.Ready
		return s
	}
	withAgent := func(st state.AgentStatus, n int) *state.State {
		s := ready()
		s.Turn.Outcome = "completed"
		for i := range n {
			k := fmt.Sprintf("a%d", i)
			s.Agents[k] = &state.Agent{Key: k, Label: k, Status: st}
			s.Order = append(s.Order, k)
		}
		return s
	}
	thinking := thinkBench(state.Thought{Start: thinkAt(7449)})
	for _, c := range []struct {
		name string
		s    *state.State
		want string
		st   *lipgloss.Style
	}{
		{"starting", func() *state.State { s := ready(); s.Session.Phase = state.Spawning; return s }(), "◉ STARTING", stProgress},
		{"closing", func() *state.State { s := ready(); s.Session.Phase = state.Closing; return s }(), "◉ CLOSING", stProgress},
		{"crashed", func() *state.State { s := ready(); s.Session.Phase = state.Crashed; return s }(), "✗ ENDED", stFailed},
		{"closed", func() *state.State { s := ready(); s.Session.Phase = state.Closed; return s }(), "○ CLOSED", stGrey},
		{"an approval pending", func() *state.State { s := ready(); s.Turn.Phase = state.WaitingApproval; return s }(), "! WAITING", stAttention},
		{"interrupting", func() *state.State { s := ready(); s.Turn.Phase = state.Interrupting; return s }(), "◉ INTERRUPTING", stProgress},
		{"thinking", thinking, "⠋ THINKING", stTitle},
		{"running", func() *state.State { s := ready(); s.Turn.Phase = state.Running; return s }(), "⠋ RUNNING", stProgress},
		{"an agent waiting", withAgent(state.AgentWaiting, 1), "! AGENT WAITING", stAttention},
		{"an agent running", withAgent(state.AgentRunning, 1), "◉ AGENT RUNNING", stProgress},
		{"two agents running", withAgent(state.AgentRunning, 2), "◉ 2 AGENTS RUNNING", stProgress},
		{"completed", func() *state.State { s := ready(); s.Turn.Outcome = "completed"; return s }(), "✓ COMPLETED", stDone},
		{"failed", func() *state.State { s := ready(); s.Turn.Outcome = "failed"; return s }(), "✗ FAILED", stFailed},
		{"interrupted", func() *state.State { s := ready(); s.Turn.Outcome = "interrupted"; return s }(), "○ INTERRUPTED", stProgress},
		{"no turn yet", ready(), "○ IDLE", stGrey},
	} {
		got := orchestratorStatus(c.s, UI{Now: now})
		if text := strings.TrimRight(texts([]line{got}), "\n"); text != c.want || got[0].st != c.st {
			t.Errorf("%s: %q styled %p, want %q styled %p", c.name, text, got[0].st, c.want, c.st)
		}
	}
	if !thinkingLive(thinking) {
		t.Fatal("the bench's phase is not live")
	}
	// 24.9 The whole mock, its agent done: completed.
	if got := texts([]line{orchestratorStatus(claudeMock, UI{Now: now})}); strings.TrimSpace(got) != "✓ COMPLETED" {
		t.Errorf("the mock whole: %q", got)
	}
}

// p3bAtWrite is the Claude mock's state at its subagent's Write request:
// the turn completed, the agent waiting (as TestAgentCardTemplate2 cuts it).
func p3bAtWrite(t *testing.T) *state.State {
	t.Helper()
	f := filepath.Join("..", "..", "testdata", "mock", "claude-subagent-approval.jsonl")
	evs, err := mock.Load(f)
	if err != nil {
		t.Fatal(err)
	}
	n, cut := 0, len(evs)
	for i, e := range evs {
		evs[i].Event.At, evs[i].Event.Seq = appClock.Add(e.After), uint64(i+1)
		if e.Event.Kind == event.ApprovalRequest {
			if n++; n == 2 {
				cut = i + 1
				break
			}
		}
	}
	s, _ := mockState(f, evs[:cut])
	if s.Turn.Phase != state.Idle || s.Turn.Outcome != "completed" {
		t.Fatalf("turn %s %q, want idle and completed", s.Turn.Phase, s.Turn.Outcome)
	}
	return s
}

// TestStageBlocks is criteria 24.10 and 24.11: a closed stage's block, its
// unfolded rows, and the cursor's background over the block.
func TestStageBlocks(t *testing.T) {
	s, now := stampedMock(t, "claude-subagent-approval.jsonl", appClock)
	ui := UI{Width: 208, Height: 59, Now: now, Focus: "A", ACursor: "s:1"}
	rows, _ := zoneARows(s, ui)
	head := regexp.MustCompile(`^✓ TURN 1\s+\S+$`)
	i := -1
	for j, r := range rows {
		if head.MatchString(r) {
			i = j
		}
	}
	if i < 0 {
		t.Fatalf("no \"✓ TURN 1\" head row with its time at the end:\n%s", strings.Join(rows, "\n"))
	}
	if !strings.HasPrefix(rows[i+1], "  Use the Agent tool to launch one gene") || rows[i+2] != "  Agent · agent general-purpose" || rows[i+3] != "" {
		t.Errorf("the block under the head: %q", rows[i+1:i+4])
	}
	if j := indexOf(rows, "✓ TURN 2"); j < 0 || rows[j+1] != "  started by the CLI" {
		t.Errorf("turn 2's block: %q", rows[j:min(j+2, len(rows))])
	}
	// The cursor's background on the block's three rows, not on the next.
	a := zoneBoxes(strings.Split(strings.TrimSuffix(texts(Draw(s, ui).lines), "\n"), "\n"))["A"]
	lines := Draw(s, ui).lines
	onCursor := func(y int) bool {
		for _, x := range lines[y] {
			if x.st == stTreeCursor {
				return true
			}
		}
		return false
	}
	for k := range 3 {
		if !onCursor(a.top + 1 + i + k) {
			t.Errorf("block row %d without the cursor's background", k)
		}
	}
	if onCursor(a.top + 1 + i + 3) {
		t.Errorf("the row after the block has the cursor's background")
	}
	// Unfolded: the connectors, the agent's row, the Agent tool's row; the
	// cursor's background on the head row alone.
	ui.Fold = map[string]bool{"s:1": false}
	rows, _ = zoneARows(s, ui)
	all := strings.Join(rows, "\n")
	for _, want := range []string{"├─ ✓ Agent  ", "└─ ✓ agent  general-purpose", "  Use the Agent tool to launch one general-purpose"} {
		if !strings.Contains(all, want) {
			t.Errorf("unfolded, zone A lacks %q:\n%s", want, all)
		}
	}
	if strings.Contains(all, "  Agent · agent general-purpose") {
		t.Errorf("unfolded, the items row is still drawn:\n%s", all)
	}
	lines = Draw(s, ui).lines
	i = -1
	for j, r := range rows {
		if head.MatchString(r) {
			i = j
		}
	}
	if !onCursor(a.top+1+i) || onCursor(a.top+1+i+1) {
		t.Errorf("unfolded: the cursor's background on rows %v, %v; want the head row alone", onCursor(a.top+1+i), onCursor(a.top+1+i+1))
	}
	// Both nodes, then the pane's three, in cursor order.
	var ids []string
	for _, n := range Nodes(s, UI{}) {
		ids = append(ids, n.ID)
	}
	if got := strings.Join(ids, ","); got != "s:1,s:2,u:telemetry,u:usage,u:quota" {
		t.Errorf("nodes %s", got)
	}
}

// TestUsageSection is criteria 24.13 to 24.15: the three columns from 41
// cells, the three rows under it, the values and their support, N/A with
// its reason, the context bar's fill.
func TestUsageSection(t *testing.T) {
	claudeMock, now := stampedMock(t, "claude-subagent-approval.jsonl", appClock)
	codexMock, _ := stampedMock(t, "codex-subagents.jsonl", appClock)
	tok, cost := count(claudeMock.Metrics["tokens.session"].V), fmt.Sprintf("$%.2f", claudeMock.Metrics["cost.usd"].V)
	rows, _ := zoneARows(claudeMock, UI{Width: 208, Height: 59, Now: now, Focus: "B"})
	i := indexOf(rows, "▾ SESSION USAGE")
	cols := regexp.MustCompile(`^  CONTEXT\s+│ TOKENS\s+│ EST\. COST$`)
	if i < 0 || !cols.MatchString(rows[i+1]) || !regexp.MustCompile(`^  4% of 1M\s+│ `+regexp.QuoteMeta(tok)+`\s+│ `+regexp.QuoteMeta(cost)+`$`).MatchString(rows[i+2]) ||
		!regexp.MustCompile(`^  █[█░]+\s+│ session total\s+│ API-equivalent$`).MatchString(rows[i+3]) {
		t.Errorf("the columns at 208x59:\n%s", strings.Join(rows[i:i+4], "\n"))
	}
	// At 120x40 (32 cells): three rows, the cost's support shortened.
	rows, _ = zoneARows(claudeMock, UI{Width: 120, Height: 40, Now: now, Focus: "B"})
	i = indexOf(rows, "▾ SESSION USAGE")
	if i < 0 || !regexp.MustCompile(`^  Context 4% of 1M [█░]+$`).MatchString(rows[i+1]) || rows[i+2] != "  Tokens "+tok+" session total" || rows[i+3] != "  Est. cost "+cost+" API-equiv." {
		t.Errorf("the rows at 120x40:\n%s", strings.Join(rows[i:i+4], "\n"))
	}
	c := usageCols(codexMock, UI{Now: now}, 54)
	if c[0].value != "6% of 258.4k" || c[1].value != "184.3k" || c[1].support != "session total" || c[2].value != "N/A" || c[2].support != "Codex reports none" || !c[2].na {
		t.Errorf("Codex's columns %+v", c)
	}
	// 24.14 Before the first result: N/A with its reason, no figure.
	s, now2 := replayed("claude-code", claude.NewMapper(), records(t, "p2-claude-conversation.jsonl")[:5])
	c = usageCols(s, UI{Now: now2}, 54)
	for k, want := range []string{"no result yet", "no result yet", "no result yet"} {
		if !c[k].na || c[k].value != "N/A" || c[k].support != want {
			t.Errorf("column %d before the first result: %+v", k, c[k])
		}
	}
	for _, l := range usageRows(s, UI{Now: now2}, 54, formFull) {
		if txt := texts([]line{l}); strings.Contains(txt, "0") || strings.Contains(txt, "$") {
			t.Errorf("a figure before the first result: %q", txt)
		}
	}
	// 24.15 The context bar: round(used / window × width) cells.
	if b := texts([]line{fillBar(3.5, 50, stTitle)}); strings.Count(b, "█") != 2 || strings.Count(b, "░") != 48 {
		t.Errorf("3.5%% of 50 cells: %q", b)
	}
	if x, ok := spanOf(fillBar(3.5, 50, stTitle), "██"); !ok || x.st != stTitle {
		t.Errorf("the fill's style %v, want the title colour", x.st)
	}
}

// TestQuotaWindows is criteria 24.16 to 24.20: the windows and their
// values, the synthetic three-window mock, the bars' fill and colour, a new
// report, the countdowns.
func TestQuotaWindows(t *testing.T) {
	// 24.16 P1 line 3; P6b line 7.
	s, now := replayed("claude-code", claude.NewMapper(), records(t, "p1-claude-stream.jsonl")[:3])
	ws := windows(s, UI{Now: now})
	if len(ws) != 2 || ws[0].label != "Session (5hr)" || ws[1].label != "Weekly (7 day)" {
		t.Fatalf("windows %+v", ws)
	}
	rows := ws[0].rows(UI{Now: now}, 52, formFull)
	if r := strings.TrimRight(texts([]line{rows[0]}), "\n"); !strings.HasPrefix(r, "  Session (5hr)") || !strings.HasSuffix(r, "22%†") || len([]rune(r)) != 52 {
		t.Errorf("the label row %q", r)
	}
	if b := texts([]line{rows[1]}); strings.Count(b, "█") != 11 || strings.Count(b, "░") != 39 {
		t.Errorf("22%% of 50 cells: %q", b)
	}
	if r := texts([]line{rows[2]}); !strings.HasPrefix(r, "  Resets in ") {
		t.Errorf("the reset row %q", r)
	}
	if b := texts([]line{ws[1].rows(UI{Now: now}, 52, formFull)[1]}); strings.Count(b, "█") != 23 {
		t.Errorf("45%% of 50 cells: %q (round half away from zero)", b)
	}
	// 24.17 The synthetic mock: three windows at the image's values.
	q, _ := stampedMock(t, "claude-quota-windows.jsonl", appClock)
	ws = windows(q, UI{Now: appClock})
	var got []string
	for _, w := range ws {
		got = append(got, w.label+" "+strings.TrimRight(texts([]line{w.right()}), "\n")+" ("+w.second(UI{Now: appClock})+")")
	}
	want := "Session (5hr) 0%† (Resets in 4h 12m)|Weekly (7 day) 17%† (Resets in 5d 3h)|Fable limit 0%† (Resets in 5d 3h)"
	if strings.Join(got, "|") != want {
		t.Errorf("the three windows:\n%s\nwant\n%s", strings.Join(got, "\n"), strings.ReplaceAll(want, "|", "\n"))
	}
	if b := texts([]line{ws[1].rows(UI{Now: appClock}, 52, formFull)[1]}); strings.Count(b, "█") != 9 {
		t.Errorf("17%% of 50 cells: %q", b)
	}
	if b := texts([]line{ws[2].rows(UI{Now: appClock}, 52, formFull)[1]}); strings.Count(b, "█") != 0 || strings.Count(b, "░") != 50 {
		t.Errorf("0%%: %q, want an empty track", b)
	}
	for _, name := range []string{"codex-subagents.jsonl", "claude-subagent-approval.jsonl"} {
		m, _ := stampedMock(t, name, appClock)
		if ws := windows(m, UI{Now: appClock}); len(ws) != 2 {
			t.Errorf("%s: %d windows, want the two reported", name, len(ws))
		}
	}
	// 24.18 The bars' colour and clamping.
	for _, c := range []struct {
		p    float64
		full int
		st   *lipgloss.Style
	}{{22, 11, stTitle}, {45, 23, stTitle}, {89, 45, stTitle}, {90, 45, stFailed}, {100, 50, stFailed}, {-5, 0, stTitle}, {150, 50, stFailed}} {
		q := quotaWindow{v: state.Value{Present: true, V: c.p, Unit: "%"}}
		b := fillBar(c.p, 50, q.barStyle())
		if n := strings.Count(texts([]line{b}), "█"); n != c.full || q.barStyle() != c.st {
			t.Errorf("%.0f%%: %d cells filled, style %p; want %d, %p", c.p, n, q.barStyle(), c.full, c.st)
		}
	}
	// 24.19 A new report replaces the value and the fill.
	s.Apply(event.Event{Seq: 1 << 40, At: now, Kind: event.Telemetry, Origin: event.Observed,
		Payload: event.Metrics{Items: []event.Metric{{Name: "quota.5h", Value: 60, Unit: "%", Documented: true, Source: "rate_limit_event", ResetsAt: now.Add(time.Hour)}}}})
	ws = windows(s, UI{Now: now})
	if ws[0].v.V != 60 || strings.Count(texts([]line{ws[0].rows(UI{Now: now}, 52, formFull)[1]}), "█") != 30 || strings.Contains(teleSummary(s, now), "22%") {
		t.Errorf("after the new report: %+v", ws[0].v)
	}
	// 24.20 The countdowns.
	for _, c := range []struct {
		d    time.Duration
		want string
	}{{5*24*time.Hour + 3*time.Hour + 20*time.Minute, "5d 3h"}, {5*24*time.Hour + 20*time.Minute, "5d"}, {4*time.Hour + 12*time.Minute + 30*time.Second, "4h 12m"},
		{4 * time.Hour, "4h"}, {12*time.Minute + 30*time.Second, "12m"}, {45 * time.Second, "<1m"}, {-time.Minute, "<1m"}} {
		if got := countdown(c.d); got != c.want {
			t.Errorf("countdown(%v) = %q, want %q", c.d, got, c.want)
		}
	}
	reset := appClock.Add(4*time.Hour + 12*time.Minute)
	if got := resetIn(reset, appClock.Add(time.Hour)); got != "Resets in 3h 12m" {
		t.Errorf("an hour later: %q", got)
	}
	if got := resetIn(time.Time{}, appClock); got != "Reset time unavailable" {
		t.Errorf("no reset time: %q", got)
	}
	// Past the reset: the clock tick turns the window N/A, nothing negative.
	q.Do(state.Tick{At: appClock.Add(5 * time.Hour)})
	w := windows(q, UI{Now: appClock.Add(5 * time.Hour)})[0]
	if w.na != "window reset; waiting for the next update" || strings.Contains(w.second(UI{Now: appClock.Add(5 * time.Hour)}), "-") {
		t.Errorf("past the reset: na %q, second %q", w.na, w.second(UI{Now: appClock.Add(5 * time.Hour)}))
	}
}

// TestNoNotes is plan 26's criterion 26.1 (amendment 25, the owner's
// standing decision): zone A draws values only, none of the notes it drew
// until plan 25, whatever the state holds; the details view keeps them.
func TestNoNotes(t *testing.T) {
	gone := []string{"ⓘ", "errors ", "retries ", "unrecognised", "quota updated", "derived", "undocumented field", "cost at API rates", "overage", "plan plus", "credits", "values from the CLI's events"}
	s, now := stampedMock(t, "claude-subagent-approval.jsonl", appClock)
	s.Errors = append(s.Errors, state.ErrorEntry{Message: "one"}, state.ErrorEntry{Message: "two"})
	s.Retries = 3
	s.Unrecognised = 3
	s.UnrecognisedLast = []state.UnknownEntry{{At: now, Type: "system/x", Line: `{"type":"system","subtype":"x"}`}}
	if v := s.Metrics["context.used"]; !v.Present || v.Origin != event.Derived {
		t.Fatalf("the mock's context is not derived: %+v", v)
	}
	if v := s.Metrics["quota.5h"]; !v.Present || v.Documented {
		t.Fatalf("the mock's quota is documented: %+v", v)
	}
	c, _ := stampedMock(t, "codex-subagents.jsonl", appClock)
	if rows, _ := teleRows(state.New(), UI{Width: 120, Now: now}, 54, formFull); strings.Contains(texts(rows), "ⓘ") || strings.Contains(texts(rows), "values from the CLI") {
		t.Errorf("an empty state draws a note:\n%s", texts(rows))
	}
	for _, tc := range []struct {
		name string
		s    *state.State
	}{{"claude", s}, {"codex", c}} {
		for _, size := range [][2]int{{120, 40}, {208, 59}} {
			rows, _ := zoneARows(tc.s, UI{Width: size[0], Height: size[1], Now: now, Focus: "B"})
			all := strings.Join(rows, "\n")
			for _, g := range gone {
				if strings.Contains(all, g) {
					t.Errorf("%s %v: zone A draws %q:\n%s", tc.name, size, g, all)
				}
			}
			if tele := indexOf(rows, "✦ TELEMETRY"); tele < 0 || !strings.HasSuffix(strings.TrimRight(rows[tele], " "), "Enter details") {
				t.Errorf("%s %v: the TELEMETRY heading row %d", tc.name, size, tele)
			}
		}
	}
	heading, _ := teleRows(s, UI{Now: now, Width: 208}, 54, formFull)
	if h := strings.TrimRight(texts(heading[1:2]), "\n"); !strings.HasPrefix(h, "✦ TELEMETRY") || !strings.HasSuffix(h, "Enter details") || len([]rune(h)) != 54 {
		t.Errorf("the heading row %q", h)
	}
	if last := texts(heading[len(heading)-1:]); !strings.HasPrefix(last, "  Resets in ") {
		t.Errorf("the full form's last row %q, want the last window's countdown", last)
	}
	// The details view keeps the counts, the legend and the marks.
	d := texts(detailsBody(s, UI{Width: 200, Now: now}, 198))
	for _, want := range []string{"value · source · received · † undocumented field", "unrecognised events 3", "retries             3", "%†", "(derived)", "system/x"} {
		if !strings.Contains(d, want) {
			t.Errorf("the details view lacks %q:\n%s", want, d)
		}
	}
}

// TestScopedWindows is criterion 24.30: the Claude adapter's model-scoped
// weekly windows, from unifiedWindows and from rateLimitType.
// TestOverageWindowSkipped (plan 25, D5; criterion 25.5): the
// unifiedWindows key seven_day_overage_included, which Claude Code 2.1.303
// sent in the logged turn of 2026-10-22 (the owner's plan, overage
// disabled), is no model's window: the two windows show, nothing else, and
// no value is kept for it.
func TestOverageWindowSkipped(t *testing.T) {
	p2 := records(t, "p2-claude-conversation.jsonl")[:3]
	s, _ := replayed("claude-code", claude.NewMapper(), append(p2, rec(7000, "out",
		`{"type":"rate_limit_event","rate_limit_info":{"status":"allowed","resetsAt":1792682400,"rateLimitType":"five_hour","overageStatus":"rejected","overageDisabledReason":"org_level_disabled","isUsingOverage":false,"unifiedWindows":{"five_hour":{"utilization":0.17,"resetsAt":1792682400},"seven_day":{"utilization":0.26,"resetsAt":1793091600},"seven_day_overage_included":{"utilization":0.17,"resetsAt":1793091600}}},"uuid":"ab0a3e5e-34ca-40c2-9d82-9aeb6cea838d","session_id":"5d15927c-da0a-46f1-9c41-d330ba488c5d"}`)))
	if v := s.Metrics["quota.5h"]; !v.Present || v.V != 17 {
		t.Errorf("quota.5h %+v", v)
	}
	if v := s.Metrics["quota.7d"]; !v.Present || v.V != 26 {
		t.Errorf("quota.7d %+v", v)
	}
	for name := range s.Metrics {
		if name == "" || strings.Contains(name, "overage_included") {
			t.Errorf("a value for the overage key: %q", name)
		}
	}
	if ws := windows(s, UI{}); len(ws) != 2 || ws[0].key != "quota.5h" || ws[1].key != "quota.7d" {
		t.Errorf("windows %+v", ws)
	}
	if s.Unrecognised != 0 {
		t.Errorf("%d unrecognised", s.Unrecognised)
	}
}

func TestScopedWindows(t *testing.T) {
	p2 := records(t, "p2-claude-conversation.jsonl")[:3]
	s, _ := replayed("claude-code", claude.NewMapper(), append(p2, rec(7000, "out",
		`{"type":"rate_limit_event","rate_limit_info":{"status":"allowed","rateLimitType":"five_hour","resetsAt":1792071720,"unifiedWindows":{"five_hour":{"utilization":0.1,"resetsAt":1792071720},"seven_day":{"utilization":0.2,"resetsAt":1792499400},"seven_day_fable":{"utilization":0.3,"resetsAt":1792499400}}}}`)))
	if v := s.Metrics["quota.7d.fable"]; !v.Present || v.V != 30 || v.Documented || v.ResetsAt.Unix() != 1792499400 || v.Unit != "%" {
		t.Errorf("quota.7d.fable %+v", v)
	}
	if ws := windows(s, UI{}); len(ws) != 3 || ws[2].key != "quota.7d.fable" || ws[2].label != "Fable limit" {
		t.Errorf("windows %+v", ws)
	}
	s, _ = replayed("claude-code", claude.NewMapper(), append(p2, rec(7000, "out",
		`{"type":"rate_limit_event","rate_limit_info":{"status":"allowed","rateLimitType":"seven_day_opus","utilization":0.5,"resetsAt":1792499400}}`)))
	if v := s.Metrics["quota.7d.opus"]; !v.Present || v.V != 50 || !v.Documented {
		t.Errorf("quota.7d.opus %+v", v)
	}
	if _, ok := s.Metrics["quota.7d.fable"]; ok {
		t.Errorf("a window the event did not carry")
	}
	// The event's own model-scoped window beside unifiedWindows' two: kept
	// (the owner's live session of 2026-10-22 showed unifiedWindows carrying
	// five_hour and seven_day only).
	s, _ = replayed("claude-code", claude.NewMapper(), append(p2, rec(7000, "out",
		`{"type":"rate_limit_event","rate_limit_info":{"status":"allowed","rateLimitType":"seven_day_fable","utilization":0.4,"resetsAt":1792499400,"unifiedWindows":{"five_hour":{"utilization":0.1,"resetsAt":1792071720},"seven_day":{"utilization":0.2,"resetsAt":1792499400}}}}`)))
	if v := s.Metrics["quota.7d.fable"]; !v.Present || v.V != 40 || !v.Documented || v.Source != "rate_limit_event" {
		t.Errorf("quota.7d.fable beside unifiedWindows %+v", v)
	}
	if ws := windows(s, UI{}); len(ws) != 3 || !near(ws[0].v.V, 10) || !near(ws[1].v.V, 20) {
		t.Errorf("windows %+v", ws)
	}
	for key, want := range map[string]string{"quota.7d.opus": "Opus limit", "quota.90m": "90 min window", "quota.1440m": "1-day window", "quota.120m": "2h window"} {
		if got := windowLabel(key); got != want {
			t.Errorf("windowLabel(%s) = %q, want %q", key, got, want)
		}
	}
}

// TestCountdownRedraw is criterion 24.29: a frame a second later differs
// only in zone A's rows (the quota's age), a minute later likewise, never
// in zone B's or C's.
func TestCountdownRedraw(t *testing.T) {
	s, _ := stampedMock(t, "claude-quota-windows.jsonl", appClock)
	draw := func(now time.Time) []string {
		return strings.Split(strings.TrimSuffix(texts(Draw(s, UI{Width: 120, Height: 40, Now: now, Focus: "B"}).lines), "\n"), "\n")
	}
	base := draw(appClock)
	aw := zoneAWidth(120)
	for _, c := range []struct {
		name string
		at   time.Time
	}{{"a second later", appClock.Add(time.Second)}, {"a minute later", appClock.Add(time.Minute)}} {
		later := draw(c.at)
		changedA := 0
		for y := range base {
			if base[y] == later[y] {
				continue
			}
			rb, rl := []rune(base[y]), []rune(later[y])
			if string(rb[aw:]) != string(rl[aw:]) || y > 36 {
				t.Errorf("%s: row %d differs outside zone A:\n%s\n%s", c.name, y, base[y], later[y])
			}
			changedA++
		}
		if changedA == 0 || c.name == "a minute later" && !strings.Contains(strings.Join(later, "\n"), "Resets in 4h 11m") {
			t.Errorf("%s: %d rows of zone A changed, countdown %v", c.name, changedA, strings.Contains(strings.Join(later, "\n"), "Resets in 4h 11m"))
		}
	}
	if f := fixture.Epoch; f.After(appClock) {
		t.Fatal("the fixture epoch is after the application clock")
	}
}
