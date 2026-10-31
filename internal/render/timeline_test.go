package render

import (
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/Evengelius/agent-command-center/internal/event"
	"github.com/Evengelius/agent-command-center/internal/fixture"
	"github.com/Evengelius/agent-command-center/internal/provider/codex"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// Plan 40 (brief amendment 41): zone C's execution timeline and the
// inspection's Messages tab draw no assistant entry with no text, as zone B
// (plan 22, design 1).

// p40Child is P7b's first child thread and its turn; p22sItem P22s's
// reasoning item with a summary, on its thread and turn (lines 67 to 70).
const (
	p40Child, p40ChildTurn = "01a14937-2992-7d51-8e12-7ff3613a8069", "01a14937-29c6-7ee3-8b48-22c96aaeff60"
	p22sThread, p22sTurn   = "01a15f60-9b59-7933-a3df-5560cc788a16", "01a15f60-b9e7-7940-8061-d848e70ac86f"
	p22sItem               = "rs_0bb989c0e5f502b3016ad8edc2bbfc8191bb725e7d8dc5d1bc"
)

// p40Records is P7b's run, Codex subagents, with P22s's reasoning item
// replayed on the first child's thread after each of the child's two
// reasoning items (P7b asked for no summary), as a child streams its
// reasoning summaries since plan 22 asks for them: the same notifications
// (item/started, summaryPartAdded, summaryTextDelta, item/completed), the
// child's thread and turn, each item its own id.
func p40Records(t *testing.T) []fixture.Record {
	t.Helper()
	var item []fixture.Record
	for _, r := range records(t, "p22s-codex-reasoning-summary.jsonl") {
		if r.Dir == "out" && strings.Contains(string(r.Msg), p22sItem) {
			item = append(item, r)
		}
	}
	if len(item) != 4 {
		t.Fatalf("P22s's reasoning item: %d lines, want 4", len(item))
	}
	var out []fixture.Record
	n := 0
	for _, r := range records(t, "p7b-codex-subagents.jsonl") {
		out = append(out, r)
		msg := string(r.Msg)
		if r.Dir != "out" || !strings.Contains(msg, `"method":"item/completed"`) || !strings.Contains(msg, `"type":"reasoning"`) || !strings.Contains(msg, `"threadId":"`+p40Child+`"`) {
			continue
		}
		on := strings.NewReplacer(p22sThread, p40Child, p22sTurn, p40ChildTurn, p22sItem, fmt.Sprintf("rs_p40_%d", n))
		for _, it := range item {
			out = append(out, fixture.Record{TMs: r.TMs, Dir: "out", Msg: []byte(on.Replace(string(it.Msg)))})
		}
		n++
	}
	if n != 2 {
		t.Fatalf("the child's reasoning items in P7b: %d, want 2", n)
	}
	return out
}

var (
	// timelineRow is an entry's first row: the time, the role, the bar
	// and what follows it.
	timelineRow = regexp.MustCompile(`^(\d\d:\d\d:\d\d)  (\S+)\s+│(.*)$`)
	clockRow    = regexp.MustCompile(`^\d\d:\d\d:\d\d(  |$)`)
)

// checkTimeline checks a timeline's rows (40.2): every entry row has text,
// the entries are want's, in order with their times, roles and first words,
// and a rule only between two entries, never first, last or twice in a row.
func checkTimeline(t *testing.T, name string, rows []string, want []state.Entry) {
	t.Helper()
	var entries []string
	for i, r := range rows {
		if strings.Trim(r, "─") == "" {
			if i == 0 || i == len(rows)-1 || strings.Trim(rows[i-1], "─") == "" {
				t.Errorf("%s: a rule at row %d of %d", name, i, len(rows))
			}
			continue
		}
		m := timelineRow.FindStringSubmatch(r)
		if m == nil {
			continue
		}
		if strings.TrimSpace(m[3]) == "" {
			t.Errorf("%s: an entry's row with no text: %q", name, r)
		}
		entries = append(entries, r)
	}
	if len(entries) != len(want) {
		t.Errorf("%s: %d entry rows, want %d:\n%s", name, len(entries), len(want), strings.Join(rows, "\n"))
		return
	}
	for i, e := range want {
		m := timelineRow.FindStringSubmatch(entries[i])
		if first := strings.Fields(e.Text)[0]; m[1] != clockSec(e.At) || m[2] != e.Role || !strings.HasPrefix(strings.TrimSpace(m[3]), first) {
			t.Errorf("%s: entry row %d %q, want %s, %q and %q", name, i, entries[i], clockSec(e.At), e.Role, first)
		}
	}
}

// checkInspection checks the inspection's Messages tab (40.3): no timed row
// with no text, and want of them: the start, the messages with text, the
// end of an agent that ended.
func checkInspection(t *testing.T, name string, s *state.State, a *state.Agent, want int) {
	t.Helper()
	ls := inspectMessages(s, nil, a, 120)
	timedRows := 0
	for _, l := range ls {
		r := strings.TrimRight(texts([]line{l}), " \n")
		if !clockRow.MatchString(r) {
			continue
		}
		timedRows++
		body := strings.TrimSpace(r[len("00:00:00"):])
		for _, o := range []event.Origin{event.AI, event.Observed, event.Derived, event.Inferred, event.Mock} {
			if body == originLabel(o) {
				t.Errorf("%s: the inspection's timed row with no text: %q", name, r)
			}
		}
	}
	if timedRows != want {
		t.Errorf("%s: the inspection's %d timed rows, want %d:\n%s", name, timedRows, want, texts(ls))
	}
}

// TestTimelineNoEmptyRows is criteria 40.1 to 40.3: the reproduction gives
// the child two entries holding only a thinking mark; zone C's timeline
// draws every entry with text and none without, alone and in the drawn
// frame at both sizes; the inspection's Messages tab no timed row with no
// text; with no message that has text, "no activity yet" and "no message
// from this agent yet".
func TestTimelineNoEmptyRows(t *testing.T) {
	s, now := replayed(codex.ProviderID, codex.NewMapper(), p40Records(t))
	a := s.Agents[p40Child]
	if a == nil {
		t.Fatalf("no agent %s; agents %v", p40Child, s.Agents)
	}
	var drawn, marks []state.Entry
	for _, e := range a.Messages {
		if e.Role == "assistant" && strings.TrimSpace(e.Text) == "" {
			marks = append(marks, e)
			continue
		}
		drawn = append(drawn, e)
	}
	if len(marks) != 2 || len(drawn) == 0 || len(a.History) != 0 {
		t.Fatalf("the reproduction: %d entries with no text, %d with text, %d history lines; want 2, some and none", len(marks), len(drawn), len(a.History))
	}
	for _, e := range marks {
		if !e.Thinking || e.Origin != event.AI {
			t.Errorf("an entry with no text that is no thinking mark: %+v", e)
		}
	}

	// The whole timeline, 60 cells wide.
	_, rows := timeline(s, UI{Selected: p40Child}, 60, 100, &Screen{})
	checkTimeline(t, "the timeline", strings.Split(strings.TrimSuffix(texts(rows), "\n"), "\n"), drawn)

	// In the frame, the child selected: the panel's right part from the
	// first row under the headings.
	for _, size := range [][2]int{{120, 40}, {200, 60}} {
		inner, _, _, lw := zoneCInner(t, s, UI{Width: size[0], Height: size[1], Now: now, Focus: "C", Selected: p40Child})
		var right []string
		for _, r := range inner[3:] {
			if r := strings.TrimRight(string([]rune(r)[lw+3:]), " "); r != "" {
				right = append(right, r)
			}
		}
		checkTimeline(t, fmt.Sprintf("the frame at %dx%d", size[0], size[1]), right, drawn)
	}

	if !a.Status.Final() {
		t.Fatalf("the child %s, want it ended", a.Status)
	}
	checkInspection(t, "the child", s, a, 1+len(drawn)+1)

	// With nothing but thinking so far.
	thinking := *a
	thinking.Messages = marks
	s.Agents[p40Child] = &thinking
	if _, rows := timeline(s, UI{Selected: p40Child}, 60, 10, &Screen{}); strings.TrimSpace(texts(rows)) != "no activity yet" {
		t.Errorf("only thinking: the timeline\n%s", texts(rows))
	}
	if got := texts(inspectMessages(s, nil, &thinking, 120)); !strings.Contains(got, "no message from this agent yet") {
		t.Errorf("only thinking: the inspection\n%s", got)
	}
}

// TestTimelineNoTextEntries is 40.2 and 40.3 on a hand-made agent: a
// message of spaces, a notice with no text and a message that ended empty
// are drawn nowhere; the entries with text are, a rule between them.
func TestTimelineNoTextEntries(t *testing.T) {
	s := state.New()
	a := &state.Agent{Key: "a1", Label: "quiet", Status: state.AgentRunning, StartedAt: appClock, Messages: []state.Entry{
		{Role: "assistant", At: appClock, Text: " \n\t "},
		{Role: "notice", At: appClock.Add(time.Second), Text: "Working directory: /tmp/p40"},
		{Role: "notice", At: appClock.Add(2 * time.Second)},
		{Role: "assistant", At: appClock.Add(3 * time.Second), Text: "Found two files."},
		{Role: "assistant", At: appClock.Add(4 * time.Second)},
	}}
	s.Agents[a.Key] = a
	_, rows := timeline(s, UI{Selected: a.Key}, 60, 20, &Screen{})
	checkTimeline(t, "hand-made", strings.Split(strings.TrimSuffix(texts(rows), "\n"), "\n"), []state.Entry{a.Messages[1], a.Messages[3]})
	checkInspection(t, "hand-made", s, a, 1+2)
}
