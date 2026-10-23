package state_test

import (
	"encoding/json"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Evengelius/agent-command-center/internal/event"
	"github.com/Evengelius/agent-command-center/internal/fixture"
	"github.com/Evengelius/agent-command-center/internal/provider/claude"
	"github.com/Evengelius/agent-command-center/internal/provider/codex"
	"github.com/Evengelius/agent-command-center/internal/provider/mock"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// Plan 22, criteria 1.3 to 1.6: the orchestrator's thinking phases, from the
// recordings replayed through the adapters (times: fixture.Epoch and the
// recording's t_ms) and from synthetic events for what no recording holds.

const (
	p12cBlock = "p12c-block-claude-auto-boundary-edit.jsonl"
	p303a     = "p303a-claude-conversation.jsonl"
	p303b     = "p303b-claude-subagent-approval.jsonl"
	p7        = "p7-codex-subagents.jsonl"
	p6b       = "p6b-codex-app-server.jsonl"
)

// at is the time t ms from the epoch.
func at(t int64) time.Time { return fixture.Epoch.Add(time.Duration(t) * time.Millisecond) }

// since is how many ms after the epoch a time is.
func since(x time.Time) int64 { return x.Sub(fixture.Epoch).Milliseconds() }

// thoughts are the thinking entries of the orchestrator's record, in order.
func thoughts(s *state.State) []state.Entry {
	var out []state.Entry
	for _, e := range s.Conversation {
		if e.Role == "thinking" {
			out = append(out, e)
		}
	}
	return out
}

// span is how long a phase ran, in ms.
func span(th *state.Thought) int64 { return th.End.Sub(th.Start).Milliseconds() }

// resultsThinking are the thinking_tokens of a recording's result lines, one
// per turn: the turn's total (plan 22, C4).
func resultsThinking(t *testing.T, name string) []int64 {
	t.Helper()
	recs, err := fixture.Read(filepath.Join("../../testdata/probes", name))
	if err != nil {
		t.Fatal(err)
	}
	var out []int64
	for _, r := range recs {
		var m struct {
			Type  string `json:"type"`
			Usage struct {
				Details struct {
					Thinking *int64 `json:"thinking_tokens"`
				} `json:"output_tokens_details"`
			} `json:"usage"`
		}
		if r.Dir == "out" && json.Unmarshal(r.Msg, &m) == nil && m.Type == "result" && m.Usage.Details.Thinking != nil {
			out = append(out, *m.Usage.Details.Thinking)
		}
	}
	if len(out) == 0 {
		t.Fatalf("%s: no result line with thinking_tokens", name)
	}
	return out
}

// 1.3: P12c-block's four phases, each with the start and the end of its block,
// the second one's estimates as C2 lists them, the exact counts of C3.
func TestThinkingClaudeP12cBlock(t *testing.T) {
	s := state.New()
	type change struct{ from, to int64 }
	var seen []change // the second phase's estimate after each event that changed it
	var seenAt []int64
	for _, ev := range replayEvents(t, p12cBlock, claude.ProviderID, claude.NewMapper()) {
		s.Apply(ev)
		if th := thoughts(s); len(th) >= 2 {
			x := th[1].Thought
			if x.Estimate > 0 && (len(seen) == 0 || seen[len(seen)-1].to != x.Estimate) {
				seen = append(seen, change{x.EstimateFrom, x.Estimate})
				seenAt = append(seenAt, since(x.EstimateAt))
			}
		}
	}
	th := thoughts(s)
	if len(th) != 4 {
		t.Fatalf("%d thinking entries, want 4", len(th))
	}
	wantSpan := []int64{3476, 13902, 56, 2161}
	wantStart := []int64{2434, 7449, 22562, 23816}
	wantCount := []int64{717, 2852, 17, 354}
	for i, e := range th {
		x := e.Thought
		if e.Agent != "" || e.Streaming || e.Origin != event.Observed || !strings.HasPrefix(e.MessageID, "msg_") || e.Text != "" || e.Thinking {
			t.Errorf("phase %d: entry %+v", i+1, e)
		}
		if !x.Confirmed || x.Inferred || x.Dropped || x.Outcome != "" || x.Block != 0 || x.Item != "" || x.Shared != 0 || x.Counted || x.Budget != 0 {
			t.Errorf("phase %d: %+v", i+1, x)
		}
		if got := since(x.Start); got != wantStart[i] {
			t.Errorf("phase %d: start %d ms, want %d", i+1, got, wantStart[i])
		}
		if x.Live() || span(x) != wantSpan[i] {
			t.Errorf("phase %d: lasted %d ms (live %v), want %d", i+1, span(x), x.Live(), wantSpan[i])
		}
		if !x.HasTokens || x.Tokens != wantCount[i] || x.Source != "message_delta usage.output_tokens_details.thinking_tokens" || x.TokensAt.Before(x.End) {
			t.Errorf("phase %d: count %d (has %v) at %v from %q, want %d", i+1, x.Tokens, x.HasTokens, x.TokensAt, x.Source, wantCount[i])
		}
	}
	// C2: the second phase's estimate, each change from the one before.
	wantTo := []int64{50, 350, 650, 850, 1100, 1450, 1700, 1950, 2200, 2450, 2700}
	wantAt := []int64{7841, 9241, 10610, 11856, 13182, 14814, 16019, 17406, 18467, 19550, 20632}
	var want []change
	for i, to := range wantTo {
		from := int64(0)
		if i > 0 {
			from = wantTo[i-1]
		}
		want = append(want, change{from, to})
	}
	if !slices.Equal(seen, want) || !slices.Equal(seenAt, wantAt) {
		t.Errorf("second phase's estimates %v at %v, want %v at %v", seen, seenAt, want, wantAt)
	}
	// The first phase ends on the CLI's last, uneven estimate (734); the exact
	// count is another number (717): never mixed (C5).
	if x := th[0].Thought; x.Estimate != 734 || x.EstimateFrom != 450 || x.Tokens != 717 {
		t.Errorf("first phase: estimate %d from %d, count %d", x.Estimate, x.EstimateFrom, x.Tokens)
	}
	// The metric is still kept as it always was: the last estimate received.
	if v := s.Metrics[event.ThinkingEstimate]; !v.Present || v.V != 447 {
		t.Errorf("metric %+v, want the last estimate, 447", v)
	}
	// Two turns: the first phase belongs to the first, the others to the second.
	if th[0].Stage != 1 || th[1].Stage != 2 || th[2].Stage != 2 || th[3].Stage != 2 {
		t.Errorf("stages %d %d %d %d, want 1 2 2 2", th[0].Stage, th[1].Stage, th[2].Stage, th[3].Stage)
	}
}

// 1.3 and 1.4: P303b's three phases, 719, 281 and 268, which add up to the
// result's 1,268; none for the subagent's four thinking blocks.
func TestThinkingClaudeP303b(t *testing.T) {
	s := state.New()
	sub := 0
	for _, ev := range replayEvents(t, p303b, claude.ProviderID, claude.NewMapper()) {
		s.Apply(ev)
		if p, ok := ev.Payload.(event.Completed); ok && ev.Agent != "" && p.Thinking {
			sub++
		}
	}
	if sub != 4 {
		t.Fatalf("%d thinking blocks of the subagent in the recording, want 4", sub)
	}
	th := thoughts(s)
	if len(th) != 3 {
		t.Fatalf("%d thinking entries, want 3 (none for the subagent's)", len(th))
	}
	var sum int64
	for i, want := range []int64{719, 281, 268} {
		x := th[i].Thought
		if !x.HasTokens || x.Tokens != want || x.Shared != 0 {
			t.Errorf("phase %d: %d (has %v, shared %d), want %d", i+1, x.Tokens, x.HasTokens, x.Shared, want)
		}
		sum += x.Tokens
	}
	if res := resultsThinking(t, p303b); len(res) != 1 || sum != res[0] || sum != 1268 {
		t.Errorf("the phases add up to %d, the result says %v, want 1268", sum, res)
	}
	// The subagent's messages are its own: no thinking entry of the orchestrator.
	for _, e := range s.Conversation {
		if e.Role == "thinking" && e.Agent != "" {
			t.Errorf("a thinking entry of agent %q", e.Agent)
		}
	}
	if len(s.Order) != 1 || len(s.Agents[s.Order[0]].Messages) == 0 {
		t.Errorf("the subagent's messages: %+v", s.Order)
	}
	// The first phase: its estimates were 50, 300, 650; the count 719.
	if x := th[0].Thought; x.Estimate != 650 || x.EstimateFrom != 300 {
		t.Errorf("first phase's estimate %d from %d, want 650 from 300", x.Estimate, x.EstimateFrom)
	}
}

// bench is a synthetic session with a turn running, to drive thinking events
// at chosen times: the cases no recording holds.
type bench struct {
	t   *testing.T
	s   *state.State
	seq uint64
}

// newBench is a ready session whose turn runs, at 0 ms.
func newBench(t *testing.T) *bench {
	t.Helper()
	b := &bench{t: t, s: state.New()}
	b.s.Do(state.Launch{Mock: "x"})
	b.send(0, event.SessionSpawned, "", event.Spawned{})
	b.send(0, event.SessionReady, "", event.Ready{})
	b.s.Do(state.Send{Text: "think"})
	b.send(0, event.UserMessage, "", event.User{Text: "think"})
	b.send(0, event.TurnStarted, "", nil)
	if b.s.Turn.Phase != state.Running {
		t.Fatalf("turn %q, want Running", b.s.Turn.Phase)
	}
	return b
}

func (b *bench) send(t int64, kind event.Kind, agent string, p any) {
	b.sendAs(t, event.Observed, kind, agent, p)
}

func (b *bench) sendAs(t int64, org event.Origin, kind event.Kind, agent string, p any) {
	b.seq++
	b.s.Apply(event.Event{Seq: b.seq, At: at(t), Provider: "test", Kind: kind, Origin: org, Agent: agent, Payload: p})
}

func (b *bench) marker(t int64, agent string, m event.Marker) {
	b.send(t, event.MessageMarker, agent, m)
}

// blockStart and blockStop are Claude Code's thinking block, blockCount the
// message_delta that ends the message.
func (b *bench) blockStart(t int64, id string, index int) {
	b.marker(t, "", event.Marker{MessageID: id, Name: "content_block_start", Block: "thinking", Index: index})
}

func (b *bench) blockStop(t int64, id string, index int) {
	b.marker(t, "", event.Marker{MessageID: id, Name: "content_block_stop", Index: index})
}

func (b *bench) blockCount(t int64, id, stop string, n int64) {
	b.marker(t, "", event.Marker{MessageID: id, Name: "message_delta", StopReason: stop, Thinking: n, HasThinking: true})
}

func (b *bench) estimate(t int64, agent string, v float64) {
	b.send(t, event.Telemetry, agent, event.Metrics{Items: []event.Metric{{Name: event.ThinkingEstimate, Value: v, Unit: "tokens"}}})
}

// step is Codex's inferred start of a model step; reasoningStart and
// reasoningEnd its reasoning item; usage a thread's usage update; text the
// start of an agentMessage.
func (b *bench) step(t int64) {
	b.sendAs(t, event.Inferred, event.MessageMarker, "", event.Marker{Name: event.StepStarted})
}

func (b *bench) reasoningStart(t int64, agent, id string) {
	b.marker(t, agent, event.Marker{MessageID: id, Name: "item_started", Block: "thinking"})
}

func (b *bench) reasoningEnd(t int64, agent, id string) {
	b.marker(t, agent, event.Marker{MessageID: id, Name: "item_completed", Block: "thinking"})
}

func (b *bench) usage(t int64, agent string, v float64) {
	b.send(t, event.Telemetry, agent, event.Metrics{Items: []event.Metric{
		{Name: "tokens.last", Value: 15000, Unit: "tokens"},
		{Name: event.ReasoningLast, Value: v, Unit: "tokens"},
	}})
}

func (b *bench) text(t int64, agent, id string) {
	b.marker(t, agent, event.Marker{MessageID: id, Name: "item_started", Block: "text"})
}

func (b *bench) turnEnd(t int64, outcome string) {
	b.send(t, event.TurnCompleted, "", event.TurnEnd{Outcome: outcome})
}

// all are the thinking entries; one is the only one.
func (b *bench) all() []state.Entry { return thoughts(b.s) }

func (b *bench) one() state.Entry {
	b.t.Helper()
	th := b.all()
	if len(th) != 1 {
		b.t.Fatalf("%d thinking entries, want 1: %+v", len(th), th)
	}
	return th[0]
}

func (b *bench) noted(part string) bool {
	for _, n := range b.s.Notes {
		if strings.Contains(n, part) {
			return true
		}
	}
	return false
}

// 1.4: the CLI's estimate goes to the live phase and never lowers it; one with
// no live phase changes none; the count is the message's own and never mixes
// with it.
func TestThinkingEstimateRules(t *testing.T) {
	b := newBench(t)
	b.estimate(500, "", 40) // before any phase: nothing to take it
	if len(b.all()) != 0 {
		t.Fatal("an estimate made a phase")
	}
	b.blockStart(1000, "m1", 0)
	x := b.one().Thought
	if !x.Live() || x.Estimate != 0 || !x.Start.Equal(at(1000)) || !x.Confirmed || x.Inferred {
		t.Fatalf("a started block: %+v", x)
	}
	for _, c := range []struct {
		t      int64
		agent  string
		value  float64
		wantTo int64
		from   int64
		wantAt int64
	}{
		{1500, "", 300, 300, 0, 1500},
		{2000, "", 650, 650, 300, 2000},
		{2500, "", 400, 650, 300, 2000},    // lower: unchanged
		{3000, "", 650, 650, 300, 2000},    // equal: unchanged
		{3200, "", 0, 650, 300, 2000},      // zero: unchanged
		{3300, "a1", 9000, 650, 300, 2000}, // a subagent's: none of the orchestrator's
		{3500, "", 700, 700, 650, 3500},
	} {
		b.estimate(c.t, c.agent, c.value)
		x := b.one().Thought
		if x.Estimate != c.wantTo || x.EstimateFrom != c.from || !x.EstimateAt.Equal(at(c.wantAt)) {
			t.Errorf("after %v at %d ms: estimate %d from %d at %d ms, want %d from %d at %d ms", c.value, c.t, x.Estimate, x.EstimateFrom, since(x.EstimateAt), c.wantTo, c.from, c.wantAt)
		}
	}
	// The block stops: an estimate after it has no phase to take it.
	b.blockStop(4000, "m1", 0)
	b.estimate(4100, "", 900)
	x = b.one().Thought
	if x.Live() || x.Estimate != 700 || x.EstimateFrom != 650 || !x.End.Equal(at(4000)) {
		t.Errorf("after the stop: %+v", x)
	}
	if v := b.s.Metrics[event.ThinkingEstimate]; !v.Present || v.V != 900 {
		t.Errorf("metric %+v: the estimate is still kept, as received", v)
	}
	// The exact count is the message's and the estimate stays the CLI's.
	b.blockCount(4500, "m1", "end_turn", 800)
	x = b.one().Thought
	if x.Estimate != 700 || x.Tokens != 800 || !x.HasTokens || x.Shared != 0 {
		t.Errorf("after the count: estimate %d, count %d", x.Estimate, x.Tokens)
	}
	// The same line again counts nothing twice.
	b.blockCount(4600, "m1", "end_turn", 800)
	if y := b.one().Thought; y.Tokens != 800 || y.Shared != 0 {
		t.Errorf("a repeated count: %d, shared %d", y.Tokens, y.Shared)
	}
	// The turn's end changes no number of a phase that ended.
	before := *b.one().Thought
	b.turnEnd(5000, "completed")
	if after := *b.one().Thought; !reflect.DeepEqual(after, before) {
		t.Errorf("the turn's end changed a finished phase:\n%+v\n%+v", after, before)
	}
}

// 1.4: a message with two thinking blocks (synthetic) has one count, on the
// second, over 2 phases; a message with one keeps it unshared.
func TestThinkingMessageWithTwoBlocks(t *testing.T) {
	b := newBench(t)
	b.blockStart(1000, "m1", 0)
	b.blockStop(2000, "m1", 0)
	b.blockStart(3000, "m1", 2)
	b.blockStop(4000, "m1", 2)
	b.blockCount(4500, "m1", "tool_use", 700)
	th := b.all()
	if len(th) != 2 {
		t.Fatalf("%d phases, want 2", len(th))
	}
	if x := th[0].Thought; x.HasTokens || x.Tokens != 0 || x.Shared != 0 || span(x) != 1000 || x.Block != 0 {
		t.Errorf("first phase: %+v", x)
	}
	x := th[1].Thought
	if !x.HasTokens || x.Tokens != 700 || x.Shared != 2 || !x.TokensAt.Equal(at(4500)) || span(x) != 1000 || x.Block != 2 {
		t.Errorf("second phase: %+v, want 700 shared by 2 at 4500 ms", x)
	}
	// A second message, one block, is not shared and takes nothing of the others.
	b.blockStart(5000, "m2", 0)
	b.blockStop(5500, "m2", 0)
	b.blockCount(5600, "m2", "end_turn", 90)
	th = b.all()
	if len(th) != 3 || th[2].Thought.Tokens != 90 || th[2].Thought.Shared != 0 || th[1].Thought.Tokens != 700 || th[0].Thought.HasTokens {
		t.Errorf("three phases: %+v", th)
	}
	// A message with no thinking block (a count of 0, as the CLI sends) makes none.
	b.blockCount(6000, "m3", "end_turn", 0)
	if len(b.all()) != 3 {
		t.Error("a message with no thinking block made a phase")
	}
	// The block that stops is found by its message and its index: another
	// message's stop, or another index's, ends nothing.
	b.blockStart(7000, "m4", 0)
	b.blockStop(7100, "m5", 0)
	b.blockStop(7200, "m4", 1)
	if th = b.all(); !th[3].Thought.Live() {
		t.Error("a stop of another block ended the phase")
	}
	b.blockStart(7250, "m4", 0) // the same block again: a repeated line
	if len(b.all()) != 4 || !b.noted("started twice") {
		t.Errorf("a block started twice made %d phases, noted %v", len(b.all()), b.noted("started twice"))
	}
	b.blockStop(7300, "m4", 0)
	if th = b.all(); th[3].Thought.Live() || !th[3].Thought.End.Equal(at(7300)) {
		t.Errorf("the stop of the right block: %+v", th[3].Thought)
	}
	// A block the CLI redacts is a phase too.
	b.marker(8000, "", event.Marker{MessageID: "m6", Name: "content_block_start", Block: "redacted_thinking", Index: 0})
	if th = b.all(); len(th) != 5 || !th[4].Thought.Live() {
		t.Errorf("a redacted block: %d phases", len(th))
	}
	// A text or tool block starts none.
	b.marker(8100, "", event.Marker{MessageID: "m6", Name: "content_block_start", Block: "text", Index: 1})
	b.marker(8200, "", event.Marker{MessageID: "m6", Name: "content_block_start", Block: "tool_use", Index: 2})
	if len(b.all()) != 5 {
		t.Error("a block that is not thinking made a phase")
	}
}

// 1.6: P303a's message cut short (stop_reason null, C7) marks its phase
// interrupted and gives it no exact count: the 0 the CLI reports with it is
// the usage the message had when it began, kept apart as reported (plan 22,
// correction C1); the others are not.
func TestThinkingClaudeP303a(t *testing.T) {
	s, _ := replayState(t, p303a, claude.ProviderID, claude.NewMapper())
	th := thoughts(s)
	if len(th) != 2 {
		t.Fatalf("%d phases, want 2", len(th))
	}
	if x := th[0].Thought; x.Outcome != "" || x.Tokens != 178 || span(x) != 747 || x.Cut {
		t.Errorf("the first phase: %+v", x)
	}
	x := th[1].Thought
	if x.Outcome != "interrupted" || x.Live() || span(x) != 175 {
		t.Errorf("the interrupted phase: %+v, want interrupted, 175 ms", x)
	}
	if x.HasTokens || x.Tokens != 0 || !x.TokensAt.IsZero() || x.Source != "" || x.Shared != 0 || !x.Cut || x.CutTokens != 0 {
		t.Errorf("the interrupted phase: %+v, want no exact count and the CLI's 0 kept apart as reported", x)
	}
	if res := resultsThinking(t, p303a); res[len(res)-1] != 0 {
		t.Errorf("the interrupted turn's result says %v", res)
	}

	// The same message cut short while its block is still live ends the phase.
	b := newBench(t)
	b.blockStart(1000, "m1", 0)
	b.estimate(1200, "", 50)
	b.blockCount(1500, "m1", "", 0)
	y := b.one().Thought
	if y.Live() || !y.End.Equal(at(1500)) || y.Outcome != "interrupted" || y.Estimate != 50 || y.HasTokens || y.Tokens != 0 || !y.Cut || y.CutTokens != 0 {
		t.Errorf("cut short while live: %+v", y)
	}
	if b.one().Streaming {
		t.Error("the entry still streams")
	}
	// A message cut short marks every one of its phases; the value the CLI
	// reports is kept on the last, as reported, and counts for none.
	c := newBench(t)
	c.blockStart(1000, "m1", 0)
	c.blockStop(1100, "m1", 0)
	c.blockStart(1200, "m1", 2)
	c.blockCount(1300, "m1", "", 5)
	for i, e := range c.all() {
		if e.Thought.Outcome != "interrupted" || e.Thought.Live() || e.Thought.HasTokens || e.Thought.Tokens != 0 || e.Thought.Shared != 0 {
			t.Errorf("phase %d of a cut message: %+v", i+1, e.Thought)
		}
	}
	if all := c.all(); all[0].Thought.Cut || !all[1].Thought.Cut || all[1].Thought.CutTokens != 5 {
		t.Errorf("a cut message: %+v, %+v; want the CLI's 5 kept on the last phase only", all[0].Thought, all[1].Thought)
	}
	// A message ended with a stop reason is not interrupted.
	d := newBench(t)
	d.blockStart(1000, "m1", 0)
	d.blockStop(1100, "m1", 0)
	d.blockCount(1200, "m1", "end_turn", 5)
	if d.one().Thought.Outcome != "" {
		t.Error("a message ended with end_turn is interrupted")
	}
	// A message_delta that carries no count gives none, but its message is
	// over: a block whose stop never came ends with it, its end inferred, so
	// it never stays live beside the next message's (plan 22, the review).
	e := newBench(t)
	e.blockStart(1000, "m1", 0)
	e.marker(1100, "", event.Marker{MessageID: "m1", Name: "message_delta", StopReason: "tool_use"})
	if x := e.one().Thought; x.Live() || !x.End.Equal(at(1100)) || !x.EndInferred || x.Outcome != "" || x.HasTokens {
		t.Errorf("a message_delta with no count: %+v", x)
	}
}

// A block whose content_block_stop never comes (a stream cut, a retry) ends
// with its message: its message_stop, or the next message's start; its end
// is then inferred. One that stopped keeps its own end (plan 22, the review).
func TestThinkingBlockWithoutStop(t *testing.T) {
	b := newBench(t)
	b.blockStart(1000, "m1", 0)
	b.marker(1500, "", event.Marker{MessageID: "m1", Name: "message_stop"})
	if x := b.one().Thought; x.Live() || !x.End.Equal(at(1500)) || !x.EndInferred {
		t.Errorf("at its message_stop: %+v", x)
	}
	c := newBench(t)
	c.blockStart(1000, "m1", 0)
	c.marker(2000, "", event.Marker{MessageID: "m2", Name: "message_start"})
	c.blockStart(2001, "m2", 0)
	all := c.all()
	if len(all) != 2 || all[0].Thought.Live() || !all[0].Thought.End.Equal(at(2000)) || !all[0].Thought.EndInferred || !all[1].Thought.Live() {
		t.Fatalf("at the next message's start: %+v, %+v", all[0].Thought, all[1].Thought)
	}
	d := newBench(t)
	d.blockStart(1000, "m1", 0)
	d.blockStop(1200, "m1", 0)
	d.marker(1500, "", event.Marker{MessageID: "m1", Name: "message_stop"})
	if x := d.one().Thought; !x.End.Equal(at(1200)) || x.EndInferred {
		t.Errorf("a block that stopped: %+v", x)
	}
}

// No phase opens once the session crashed: nothing would ever end it, and
// its card would animate for good (plan 22, the review).
func TestThinkingAfterCrash(t *testing.T) {
	b := newBench(t)
	b.s.Session.Phase = state.Crashed
	b.blockStart(1000, "m1", 0)
	b.marker(1100, "", event.Marker{MessageID: "r1", Name: "item_started", Block: "thinking"})
	if n := len(b.all()); n != 0 {
		t.Errorf("%d phases opened after a crash", n)
	}
}

// Each phase gets the record's next ID, numbered on across a resume, where
// the events' Seq start again (plan 22, the review).
func TestThinkingIDs(t *testing.T) {
	b := newBench(t)
	b.blockStart(1000, "m1", 0)
	b.blockStop(1100, "m1", 0)
	b.s.Conversation[len(b.s.Conversation)-1].Seq = 7
	b.blockStart(2000, "m2", 0)
	all := b.all()
	if len(all) != 2 || all[0].Thought.ID != 1 || all[1].Thought.ID != 2 || b.s.Thoughts != 2 {
		t.Errorf("IDs %d and %d, State.Thoughts %d; want 1, 2, 2", all[0].Thought.ID, all[1].Thought.ID, b.s.Thoughts)
	}
}

// 1.6: a turn's end ends the live phase at the event's time, interrupted or
// failed as the turn is; a crash or the end of the session fails it; an open
// Codex step ends dropped.
func TestThinkingEnds(t *testing.T) {
	t.Run("a turn interrupted", func(t *testing.T) {
		b := newBench(t)
		b.blockStart(1000, "m1", 0)
		b.s.Do(state.Interrupt{})
		if b.s.Turn.Phase != state.Interrupting {
			t.Fatalf("turn %q", b.s.Turn.Phase)
		}
		b.turnEnd(2500, "interrupted")
		x := b.one().Thought
		if x.Live() || !x.End.Equal(at(2500)) || x.Outcome != "interrupted" || x.Dropped || b.one().Streaming {
			t.Errorf("%+v", x)
		}
	})
	t.Run("an interrupt the CLI reports as an error", func(t *testing.T) {
		// P303a: the result says error_during_execution; the machine's outcome
		// for an interrupted turn is interrupted whatever the CLI called it.
		b := newBench(t)
		b.blockStart(1000, "m1", 0)
		b.s.Do(state.Interrupt{})
		b.turnEnd(2500, "failed")
		if x := b.one().Thought; x.Outcome != "interrupted" {
			t.Errorf("outcome %q, want interrupted", x.Outcome)
		}
	})
	t.Run("a turn failed", func(t *testing.T) {
		b := newBench(t)
		b.blockStart(1000, "m1", 0)
		b.turnEnd(2500, "failed")
		x := b.one().Thought
		if x.Live() || !x.End.Equal(at(2500)) || x.Outcome != "failed" {
			t.Errorf("%+v", x)
		}
	})
	t.Run("a turn completed with a block that never stopped", func(t *testing.T) {
		b := newBench(t)
		b.blockStart(1000, "m1", 0)
		b.turnEnd(2500, "completed")
		x := b.one().Thought
		if x.Live() || !x.End.Equal(at(2500)) || x.Outcome != "" {
			t.Errorf("%+v", x)
		}
	})
	t.Run("a late turn end", func(t *testing.T) {
		// The machine is idle already (a force stop): no phase outlives a turn end.
		b := newBench(t)
		b.blockStart(1000, "m1", 0)
		b.turnEnd(2000, "completed")
		b.blockStart(2100, "m2", 0)
		b.turnEnd(2500, "failed")
		th := b.all()
		if len(th) != 2 || th[1].Thought.Live() || th[1].Thought.Outcome != "failed" || !b.noted("unexpected turn end") {
			t.Errorf("%+v", th)
		}
	})
	t.Run("a crash", func(t *testing.T) {
		b := newBench(t)
		b.blockStart(1000, "m1", 0)
		b.send(3000, event.SessionEnded, "", event.Ended{Code: 1, HasCode: true})
		x := b.one().Thought
		if b.s.Session.Phase != state.Crashed || x.Live() || !x.End.Equal(at(3000)) || x.Outcome != "failed" || b.one().Streaming {
			t.Errorf("session %q: %+v", b.s.Session.Phase, x)
		}
	})
	t.Run("a session the owner closed", func(t *testing.T) {
		// Closed by the owner, not crashed: the phase was interrupted.
		b := newBench(t)
		b.blockStart(1000, "m1", 0)
		b.s.Do(state.Quit{}) // asks to confirm: a turn runs
		b.s.Do(state.Quit{})
		b.send(3000, event.SessionEnded, "", event.Ended{Code: 0, HasCode: true})
		x := b.one().Thought
		if b.s.Session.Phase != state.Closed || x.Live() || !x.End.Equal(at(3000)) || x.Outcome != "interrupted" {
			t.Errorf("session %q: %+v", b.s.Session.Phase, x)
		}
	})
	t.Run("a turn force-stopped", func(t *testing.T) {
		// Interrupted, no turn end came, the owner forced the stop: no turn end
		// will come, so the phase ends with the turn, at the last event's time,
		// and no card stays live after it.
		b := newBench(t)
		b.blockStart(1000, "m1", 0)
		b.s.Do(state.Interrupt{})
		b.s.Do(state.InterruptTimeout{})
		b.s.Do(state.ForceStop{})
		x := b.one().Thought
		if b.s.Turn.Phase != state.Idle || x.Live() || !x.End.Equal(at(1000)) || x.Outcome != "interrupted" {
			t.Errorf("turn %q: %+v", b.s.Turn.Phase, x)
		}
	})
	t.Run("a crash an action causes has no time of its own", func(t *testing.T) {
		// A request being answered, the answer cannot be written: the session
		// crashes with no event. The phase ends at the last event's time.
		b := newBench(t)
		b.blockStart(1000, "m1", 0)
		b.send(1500, event.ApprovalRequest, "", event.Approval{ID: "ap1", Tool: "Bash"})
		b.s.Do(state.Answer{ID: "ap1", Decision: "allow"})
		b.s.Do(state.AnswerFailed{ID: "ap1", Message: "pipe closed"})
		x := b.one().Thought
		if b.s.Session.Phase != state.Crashed || x.Live() || !x.End.Equal(at(1500)) || x.Outcome != "failed" {
			t.Errorf("session %q: %+v", b.s.Session.Phase, x)
		}
	})
	t.Run("a step no reasoning confirmed", func(t *testing.T) {
		for _, outcome := range []string{"completed", "interrupted", "failed"} {
			b := newBench(t)
			b.step(1000)
			b.turnEnd(2500, outcome)
			x := b.one().Thought
			if x.Live() || !x.Dropped || x.Outcome != "" || !x.End.Equal(at(2500)) || b.one().Streaming {
				t.Errorf("%s: %+v", outcome, x)
			}
		}
		b := newBench(t)
		b.step(1000)
		b.send(3000, event.SessionEnded, "", event.Ended{})
		if x := b.one().Thought; x.Live() || !x.Dropped || !x.End.Equal(at(3000)) {
			t.Errorf("a crash: %+v", x)
		}
	})
	t.Run("a reasoning item the turn cut", func(t *testing.T) {
		b := newBench(t)
		b.step(1000)
		b.reasoningStart(1500, "", "rs1")
		b.turnEnd(2500, "interrupted")
		x := b.one().Thought
		if x.Live() || x.Dropped || x.Outcome != "interrupted" || !x.End.Equal(at(2500)) || !x.Start.Equal(at(1000)) {
			t.Errorf("%+v", x)
		}
	})
	t.Run("a child's turn end", func(t *testing.T) {
		b := newBench(t)
		b.blockStart(1000, "m1", 0)
		b.send(1500, event.TurnCompleted, "child", event.TurnEnd{Outcome: "failed"})
		if !b.one().Thought.Live() {
			t.Error("a child's turn end ended the orchestrator's phase")
		}
	})
}

// 1.5: P7's orchestrator: three thoughts, from the step's start to the
// reasoning item's end (3,140, 3,743 and 1,440 ms), with 83, 8 and 7 reasoning
// tokens, and one step dropped, the last, ended by the agentMessage; the
// children's items, tools, messages and usage change none of them.
func TestThinkingCodexP7(t *testing.T) {
	evs := replayEvents(t, p7, codex.ProviderID, codex.NewMapper())
	s := state.New()
	for _, ev := range evs {
		s.Apply(ev)
	}
	th := thoughts(s)
	if len(th) != 4 {
		t.Fatalf("%d entries, want 3 thoughts and a dropped step", len(th))
	}
	wantStart := []int64{3380, 7120, 13471}
	wantSpan := []int64{3140, 3743, 1440}
	wantTokens := []int64{83, 8, 7}
	for i := 0; i < 3; i++ {
		e, x := th[i], th[i].Thought
		if !x.Inferred || !x.Confirmed || x.Dropped || x.Live() || x.Outcome != "" || e.Streaming || x.Item == "" || x.Block != 0 {
			t.Errorf("thought %d: %+v", i+1, x)
		}
		if e.Origin != event.Inferred || !strings.HasPrefix(e.MessageID, "step:") || e.Agent != "" {
			t.Errorf("thought %d: entry %+v", i+1, e)
		}
		if got := since(x.Start); got != wantStart[i] || span(x) != wantSpan[i] {
			t.Errorf("thought %d: from %d ms for %d ms, want from %d for %d", i+1, got, span(x), wantStart[i], wantSpan[i])
		}
		if !x.HasTokens || x.Tokens != wantTokens[i] || x.Shared != 0 || !x.Counted || x.Source != "thread/tokenUsage/updated last.reasoningOutputTokens" {
			t.Errorf("thought %d: %d tokens (has %v, shared %d, counted %v) from %q, want %d", i+1, x.Tokens, x.HasTokens, x.Shared, x.Counted, x.Source, wantTokens[i])
		}
	}
	// The second thought is a step that made a spawn and no reasoning: it
	// stayed open into the next step (the marker at 9,632 ms opened none).
	// Event 35: the thread start's model settings are an event since plan 34.
	if th[1].MessageID != "step:35" {
		t.Errorf("the second thought is %q, want the step opened by event 35", th[1].MessageID)
	}
	d := th[3].Thought
	if !d.Dropped || d.Confirmed || !d.Inferred || d.Live() || d.HasTokens || d.Outcome != "" || d.Item != "" {
		t.Errorf("the dropped step: %+v", d)
	}
	if from, to := since(d.Start), since(d.End); from != 24076 || to != 25244 {
		t.Errorf("the dropped step ran %d to %d ms, want 24076 to 25244 (the agentMessage)", from, to)
	}
	if len(s.Order) != 2 {
		t.Errorf("%d children, want 2", len(s.Order))
	}

	// The children's events change none of the orchestrator's thoughts: the
	// record without them is the same.
	alone := state.New()
	for _, ev := range evs {
		if ev.Agent == "" {
			alone.Apply(ev)
		}
	}
	a := thoughts(alone)
	if len(a) != len(th) {
		t.Fatalf("%d thoughts without the children, %d with", len(a), len(th))
	}
	for i := range a {
		if !reflect.DeepEqual(a[i].Thought, th[i].Thought) {
			t.Errorf("thought %d changes with the children:\n%+v\n%+v", i+1, *a[i].Thought, *th[i].Thought)
		}
	}
}

// P6b (Codex 0.145.0): a reasoning item that comes before its step's marker (a
// file change's completion came late) is a thought that starts at its turn's
// last usage update, 19,363 ms (plan 22, correction C5); the next step is a
// thought of its own and takes the update that follows.
func TestThinkingCodexP6b(t *testing.T) {
	s, _ := replayState(t, p6b, codex.ProviderID, codex.NewMapper())
	th := thoughts(s)
	type want struct {
		dropped, confirmed bool
		start              int64 // -1: not known
		span               int64
		tokens             int64
		has                bool
	}
	wants := []want{
		{dropped: true, start: 4919, span: 1116},
		{confirmed: true, start: 19363, span: 1634, tokens: 7, has: true},
		{confirmed: true, start: 24530, span: 1053, tokens: 9, has: true},
		{dropped: true, start: 32420, span: 946},
		// The interrupted turn's update (line 87) repeats line 64's: it reports
		// no model response, so this thought gets no count (plan 22, the review).
		{confirmed: true, start: 34469, span: 1647},
	}
	if len(th) != len(wants) {
		t.Fatalf("%d entries, want %d", len(th), len(wants))
	}
	for i, w := range wants {
		x := th[i].Thought
		start := int64(-1)
		if !x.Start.IsZero() {
			start = since(x.Start)
		}
		if x.Dropped != w.dropped || x.Confirmed != w.confirmed || !x.Inferred || start != w.start || x.HasTokens != w.has || x.Tokens != w.tokens || x.Live() {
			t.Errorf("entry %d: %+v, want %+v", i+1, x, w)
		}
		if w.span > 0 && span(x) != w.span {
			t.Errorf("entry %d lasted %d ms, want %d", i+1, span(x), w.span)
		}
	}
	// The thought no step marker announced: observed, ended by its item.
	if e := th[1]; e.Origin != event.Observed || e.Thought.End.IsZero() || e.Thought.Item == "" {
		t.Errorf("the thought no step marker announced: %+v %+v", e, e.Thought)
	}
}

// 1.5 and the design: Codex's steps, one case at a time.
func TestThinkingCodexSteps(t *testing.T) {
	t.Run("a step confirmed, ended and counted", func(t *testing.T) {
		b := newBench(t)
		b.step(100)
		e := b.one()
		if !e.Streaming || e.Origin != event.Inferred || e.MessageID != "step:"+itoa(int(b.seq)) || e.Seq != b.seq || !e.Thought.Live() || !e.Thought.Inferred || e.Thought.Confirmed || !e.Thought.Start.Equal(at(100)) || !e.At.Equal(at(100)) {
			t.Fatalf("a step: %+v %+v", e, e.Thought)
		}
		b.reasoningStart(500, "", "rs1")
		x := b.one().Thought
		if !x.Live() || !x.Confirmed || x.Item != "rs1" || !x.Start.Equal(at(100)) {
			t.Fatalf("confirmed: %+v", x)
		}
		b.reasoningEnd(501, "", "rs1")
		x = b.one().Thought
		if x.Live() || !x.End.Equal(at(501)) || span(x) != 401 || b.one().Streaming || x.Dropped {
			t.Fatalf("ended: %+v", x)
		}
		b.usage(900, "", 12)
		x = b.one().Thought
		if !x.HasTokens || x.Tokens != 12 || !x.Counted || !x.TokensAt.Equal(at(900)) || x.Shared != 0 {
			t.Fatalf("counted: %+v", x)
		}
		// The next update, for a step with no reasoning, reports 0 and changes nothing.
		b.step(900)
		b.usage(1200, "", 0)
		th := b.all()
		if len(th) != 2 || th[0].Thought.Tokens != 12 || th[1].Thought.HasTokens || th[1].Thought.Counted {
			t.Fatalf("an update of 0 with no reasoning: %+v", th)
		}
	})

	t.Run("a confirmed thought is never dropped", func(t *testing.T) {
		b := newBench(t)
		b.step(100)
		b.reasoningStart(200, "", "rs1")
		b.text(250, "", "msg1") // while the reasoning item has not completed
		if x := b.one().Thought; !x.Live() || x.Dropped {
			t.Fatalf("a message dropped a confirmed thought: %+v", x)
		}
		b.reasoningEnd(300, "", "rs1")
		b.send(350, event.ToolStarted, "", event.Tool{ToolID: "t1", Name: "commandExecution"})
		b.send(360, event.ApprovalRequest, "", event.Approval{ID: "ap1", Tool: "commandExecution"})
		if x := b.one().Thought; x.Live() || x.Dropped || !x.Confirmed {
			t.Fatalf("a call dropped a confirmed thought: %+v", x)
		}
	})

	t.Run("the first output that is not reasoning drops the step", func(t *testing.T) {
		outputs := []struct {
			name string
			do   func(b *bench)
		}{
			{"an agentMessage", func(b *bench) { b.text(300, "", "msg1") }},
			{"a text delta", func(b *bench) {
				b.send(300, event.MessageDelta, "", event.Delta{MessageID: "msg1", Block: "text", Text: "hi"})
			}},
			{"a completed message", func(b *bench) {
				b.send(300, event.MessageCompleted, "", event.Completed{MessageID: "msg1", Text: "hi"})
			}},
			{"a call", func(b *bench) {
				b.send(300, event.ToolStarted, "", event.Tool{ToolID: "t1", Name: "commandExecution"})
			}},
			{"an approval", func(b *bench) {
				b.send(300, event.ApprovalRequest, "", event.Approval{ID: "ap1", Tool: "commandExecution"})
			}},
			{"a question", func(b *bench) {
				b.send(300, event.Question, "", event.Approval{ID: "q1", Tool: "AskUserQuestion"})
			}},
			{"a request nobody can read", func(b *bench) {
				b.send(300, event.ApprovalRequest, "", event.Approval{ID: "ap2", Problem: "no input"})
			}},
		}
		for _, o := range outputs {
			b := newBench(t)
			b.step(100)
			o.do(b)
			x := b.one().Thought
			if x.Live() || !x.Dropped || x.Confirmed || !x.End.Equal(at(300)) || b.one().Streaming {
				t.Errorf("%s: %+v", o.name, x)
			}
			// What follows does not reopen it, and the next step is another entry.
			b.step(400)
			if th := b.all(); len(th) != 2 || !th[1].Thought.Live() {
				t.Errorf("%s: the next step: %+v", o.name, th)
			}
		}
	})

	t.Run("what leaves the step open", func(t *testing.T) {
		c := newBench(t)
		c.step(100)
		// A message with no text, a spawn and everything of the children.
		c.send(150, event.MessageCompleted, "", event.Completed{MessageID: "m1", Thinking: true})
		c.send(160, event.AgentStarted, "child", event.Agent{Key: "child", Type: "thread", Status: "running"})
		c.send(170, event.AgentProgress, "child", event.Agent{Key: "child", Status: "running"})
		c.text(180, "child", "cm1")
		c.send(181, event.MessageDelta, "child", event.Delta{MessageID: "cm1", Block: "text", Text: "x"})
		c.send(182, event.MessageCompleted, "child", event.Completed{MessageID: "cm1", Text: "x"})
		c.send(183, event.ToolStarted, "child", event.Tool{ToolID: "ct", Name: "commandExecution"})
		c.send(184, event.ApprovalRequest, "child", event.Approval{ID: "cap", Tool: "commandExecution"})
		c.reasoningStart(185, "child", "crs")
		c.reasoningEnd(186, "child", "crs")
		c.usage(187, "child", 99)
		c.send(188, event.StatusChanged, "", event.Status{Status: "active"})
		c.marker(189, "", event.Marker{MessageID: "u1", Name: "item_started", Block: "user"})
		x := c.one().Thought
		if !x.Live() || x.Dropped || x.Confirmed || x.Item != "" || x.HasTokens {
			t.Fatalf("a spawn, a child's output or the owner's message touched the step: %+v", x)
		}
		// A step that only spawned stays open into the next: no second entry.
		c.step(200)
		if len(c.all()) != 1 {
			t.Fatalf("a second marker opened a second step: %d entries", len(c.all()))
		}
		// The reasoning item then confirms it, and its time runs from the first marker.
		c.reasoningStart(300, "", "rs1")
		c.reasoningEnd(301, "", "rs1")
		x = c.one().Thought
		if x.Live() || !x.Confirmed || !x.Start.Equal(at(100)) || span(x) != 201 {
			t.Errorf("the step that stayed open: %+v", x)
		}
		// The child's usage did not count for it; the orchestrator's does.
		c.usage(400, "child", 55)
		if c.one().Thought.HasTokens {
			t.Error("a child's usage counted for the orchestrator's thought")
		}
		c.usage(401, "", 8)
		if x = c.one().Thought; !x.HasTokens || x.Tokens != 8 {
			t.Errorf("the orchestrator's usage: %+v", x)
		}
	})

	t.Run("the orchestrator under the session's own key", func(t *testing.T) {
		// Codex 0.163.0 names the orchestrator's thread: its events are the
		// orchestrator's, not an agent's (L5).
		b := newBench(t)
		b.send(10, event.SessionStarted, "", event.Started{SessionID: "root-thread"})
		b.sendAs(100, event.Inferred, event.MessageMarker, "root-thread", event.Marker{Name: event.StepStarted})
		b.marker(200, "root-thread", event.Marker{MessageID: "rs1", Name: "item_started", Block: "thinking"})
		b.marker(201, "root-thread", event.Marker{MessageID: "rs1", Name: "item_completed", Block: "thinking"})
		b.send(300, event.Telemetry, "root-thread", event.Metrics{Items: []event.Metric{{Name: event.ReasoningLast, Value: 9}}})
		x := b.one().Thought
		if x.Live() || !x.Confirmed || span(x) != 101 || !x.HasTokens || x.Tokens != 9 {
			t.Errorf("%+v", x)
		}
	})

	t.Run("a step only when the turn works", func(t *testing.T) {
		b := newBench(t)
		b.turnEnd(100, "completed")
		b.step(200)
		if len(b.all()) != 0 || !b.noted("model step started while the turn is Idle") {
			t.Errorf("a step after the turn: %d entries, notes %v", len(b.all()), b.s.Notes)
		}
		c := newBench(t)
		c.s.Do(state.Interrupt{})
		c.step(200)
		if len(c.all()) != 0 || !c.noted("model step started while the turn is Interrupting") {
			t.Errorf("a step while interrupting: %d entries, notes %v", len(c.all()), c.s.Notes)
		}
		// A request waiting does not stop the model: the step is opened.
		d := newBench(t)
		d.send(50, event.ApprovalRequest, "", event.Approval{ID: "ap1", Tool: "x"})
		if d.s.Turn.Phase != state.WaitingApproval {
			t.Fatalf("turn %q", d.s.Turn.Phase)
		}
		d.step(200)
		if len(d.all()) != 1 {
			t.Error("no step while a request waits")
		}
	})

	t.Run("a reasoning item with no step open", func(t *testing.T) {
		b := newBench(t)
		b.reasoningStart(500, "", "rs1")
		e := b.one()
		x := e.Thought
		if !x.Live() || !x.Confirmed || !x.Inferred || !x.Start.IsZero() || x.Item != "rs1" || e.Origin != event.Observed || !e.At.Equal(at(500)) || !strings.HasPrefix(e.MessageID, "step:") {
			t.Fatalf("a thought with no known start: %+v %+v", e, x)
		}
		// The same item again is a repeated line.
		b.reasoningStart(501, "", "rs1")
		if len(b.all()) != 1 || !b.noted("started twice") {
			t.Fatalf("a repeated item: %d entries", len(b.all()))
		}
		// Another item's end, or an unknown one, ends nothing.
		b.reasoningEnd(502, "", "rs-other")
		if !b.one().Thought.Live() {
			t.Fatal("another item's end ended the thought")
		}
		b.reasoningEnd(503, "", "rs1")
		if x = b.one().Thought; x.Live() || !x.End.Equal(at(503)) || !x.Start.IsZero() {
			t.Fatalf("ended: %+v", x)
		}
		// Its end again changes nothing.
		b.reasoningEnd(900, "", "rs1")
		if x = b.one().Thought; !x.End.Equal(at(503)) {
			t.Fatalf("a second end moved it to %v", x.End)
		}
		// The next step opens as any other.
		b.step(600)
		if th := b.all(); len(th) != 2 || !th[1].Thought.Live() || th[1].Thought.Start.IsZero() {
			t.Fatalf("the step after: %+v", th)
		}
	})

	t.Run("one update for several thoughts", func(t *testing.T) {
		b := newBench(t)
		b.step(100)
		b.reasoningStart(200, "", "rs1")
		b.reasoningEnd(201, "", "rs1")
		b.reasoningStart(300, "", "rs2") // no step open: a thought of its own
		b.reasoningEnd(301, "", "rs2")
		b.usage(400, "", 30)
		th := b.all()
		if len(th) != 2 || th[0].Thought.HasTokens || !th[0].Thought.Counted || !th[1].Thought.HasTokens || th[1].Thought.Tokens != 30 || th[1].Thought.Shared != 2 || !th[1].Thought.Counted {
			t.Fatalf("one update, two thoughts: %+v %+v", th[0].Thought, th[1].Thought)
		}
		// The next update goes to the thoughts after them only.
		b.step(400)
		b.reasoningStart(500, "", "rs3")
		b.reasoningEnd(501, "", "rs3")
		b.usage(600, "", 7)
		th = b.all()
		if len(th) != 3 || th[2].Thought.Tokens != 7 || th[2].Thought.Shared != 0 || th[1].Thought.Tokens != 30 || th[0].Thought.HasTokens {
			t.Fatalf("the next update: %+v", th)
		}
		// An update with nothing to count is a step with no reasoning.
		b.usage(700, "", 0)
		if th = b.all(); th[2].Thought.Tokens != 7 || th[1].Thought.Tokens != 30 {
			t.Fatalf("an update with nothing to count: %+v", th)
		}
	})

	t.Run("an item of an earlier turn is not found", func(t *testing.T) {
		b := newBench(t)
		b.step(100)
		b.reasoningStart(200, "", "rs1")
		b.turnEnd(300, "interrupted") // the item never completed
		b.s.Do(state.Send{Text: "again"})
		b.send(400, event.UserMessage, "", event.User{Text: "again"})
		b.send(400, event.TurnStarted, "", nil)
		b.reasoningEnd(500, "", "rs1")
		if x := b.one().Thought; !x.End.Equal(at(300)) || x.Outcome != "interrupted" {
			t.Errorf("a late end of an earlier turn's item changed the thought: %+v", x)
		}
	})

	t.Run("an update is not carried to the next prompt", func(t *testing.T) {
		// A thought of an earlier turn that never got its update is not counted
		// by this turn's.
		b := newBench(t)
		b.step(100)
		b.reasoningStart(200, "", "rs1")
		b.reasoningEnd(201, "", "rs1")
		b.turnEnd(300, "completed")
		b.s.Do(state.Send{Text: "again"})
		b.send(400, event.UserMessage, "", event.User{Text: "again"})
		b.send(400, event.TurnStarted, "", nil)
		b.usage(500, "", 44)
		if th := b.all(); th[0].Thought.HasTokens {
			t.Error("an update counted a thought of an earlier turn")
		}
	})
}

// Item 9: the orchestrator's thinking deltas make no assistant entry; the
// thinking entry stands for them. In every recording no orchestrator message is
// a thinking mark with no text, and no thinking entry is a subagent's.
func TestThinkingMakesNoAssistantEntry(t *testing.T) {
	for _, r := range recordings {
		s, _ := replayState(t, r.name, r.provider, r.mapper())
		for i, e := range s.Conversation {
			if e.Role == "assistant" && e.Thinking && e.Text == "" {
				t.Errorf("%s: entry %d is a thinking mark with no text: %+v", r.name, i, e)
			}
			if e.Role == "thinking" && (e.Agent != "" || e.Thought == nil || e.Text != "") {
				t.Errorf("%s: entry %d: %+v", r.name, i, e)
			}
		}
	}
}

// Every phase has ended when a recording ends, whichever way it ended: none
// lives on, nothing streams, and the record keeps its order: a thinking entry
// comes before the text of its message and falls in the turn it belongs to.
func TestThinkingRecordedPhasesEnd(t *testing.T) {
	for _, r := range recordings {
		s, _ := replayState(t, r.name, r.provider, r.mapper())
		last := uint64(0)
		textAt := map[string]int{}  // message id: where its assistant entry is
		thinkAt := map[string]int{} // message id: where its thinking entry is
		for i, e := range s.Conversation {
			if e.Seq < last {
				t.Errorf("%s: entry %d (%s) has seq %d after %d", r.name, i, e.Role, e.Seq, last)
			}
			last = e.Seq
			switch e.Role {
			case "assistant":
				textAt[e.MessageID] = i
			case "thinking":
				x := e.Thought
				if x.Live() || e.Streaming || (x.Inferred && !x.Confirmed && !x.Dropped) || x.End.Before(x.Start) {
					t.Errorf("%s: entry %d is live or inconsistent: %+v", r.name, i, x)
				}
				if !x.Inferred {
					thinkAt[e.MessageID] = i
				}
				// The entry belongs to the turn it came in: its time falls in
				// the stage's.
				if e.Stage > 0 {
					st := s.Stages[e.Stage-1]
					if e.At.Before(st.StartedAt) || (!st.EndedAt.IsZero() && e.At.After(st.EndedAt)) {
						t.Errorf("%s: entry %d at %v is of stage %d, which ran %v to %v", r.name, i, e.At, e.Stage, st.StartedAt, st.EndedAt)
					}
				}
			}
		}
		for id, ti := range thinkAt {
			if ai, ok := textAt[id]; ok && ai < ti {
				t.Errorf("%s: message %s: its text (entry %d) comes before its thinking (entry %d)", r.name, id, ai, ti)
			}
		}
	}
}

// The order of the record: each thinking entry stands where its phase began, so
// before the text and the calls of its step. The roles in order, as recorded.
func TestThinkingOrder(t *testing.T) {
	roles := func(s *state.State) string {
		var out []string
		for _, e := range s.Conversation {
			out = append(out, e.Role)
		}
		return strings.Join(out, " ")
	}
	for _, c := range []struct {
		name string
		s    *state.State
		want string
	}{
		{p12cBlock, first(replayState(t, p12cBlock, claude.ProviderID, claude.NewMapper())),
			"user thinking assistant user thinking assistant tool thinking assistant tool thinking assistant"},
		{p303b, first(replayState(t, p303b, claude.ProviderID, claude.NewMapper())),
			"user thinking tool approval thinking tool thinking assistant"},
		// P303a ends on its interrupted turn: the thinking, then the notice.
		{p303a, first(replayState(t, p303a, claude.ProviderID, claude.NewMapper())),
			"user thinking tool approval assistant user assistant user thinking notice"},
		// P7: the second step stays open into the third marker; the last, dropped,
		// stands before the answer that dropped it.
		{p7, first(replayState(t, p7, codex.ProviderID, codex.NewMapper())),
			"user thinking thinking tool approval thinking tool thinking assistant"},
	} {
		if got := roles(c.s); got != c.want {
			t.Errorf("%s:\n got %s\nwant %s", c.name, got, c.want)
		}
	}
}

// first is the state of replayState, to use it in a literal.
func first(s *state.State, _ int) *state.State { return s }

// The mock files that carry thinking give the same phases as the recordings
// they were made from, labelled mock as every event of mock mode is.
func TestThinkingMockFiles(t *testing.T) {
	load := func(name string) *state.State {
		t.Helper()
		timed, err := mock.Load(filepath.Join("../../testdata/mock", name))
		if err != nil {
			t.Fatal(err)
		}
		s := state.New()
		for i, x := range timed {
			ev := x.Event
			ev.Seq, ev.At = uint64(i+1), fixture.Epoch.Add(x.After)
			s.Apply(ev)
		}
		return s
	}
	check := func(name string, spans, counts []int64) {
		t.Helper()
		th := thoughts(load(name))
		if len(th) != len(counts) {
			t.Fatalf("%s: %d phases, want %d", name, len(th), len(counts))
		}
		for i, e := range th {
			x := e.Thought
			if e.Origin != event.Mock || x.Live() || (spans != nil && span(x) != spans[i]) || !x.HasTokens || x.Tokens != counts[i] {
				t.Errorf("%s: phase %d: lasted %d ms, %d tokens (has %v), origin %q; want %v ms, %d", name, i+1, span(x), x.Tokens, x.HasTokens, e.Origin, spans, counts[i])
			}
		}
	}
	check("claude-thinking.jsonl", []int64{3476, 13902, 56, 2161}, []int64{717, 2852, 17, 354})
	check("claude-thinking-subagent.jsonl", nil, []int64{719, 281, 268})

	th := thoughts(load("codex-subagents.jsonl"))
	if len(th) != 4 {
		t.Fatalf("codex-subagents.jsonl: %d entries, want 3 thoughts and a step dropped", len(th))
	}
	for i, want := range []struct{ span, tokens int64 }{{3140, 83}, {3743, 8}, {1440, 7}} {
		x := th[i].Thought
		if th[i].Origin != event.Mock || !x.Confirmed || x.Dropped || span(x) != want.span || x.Tokens != want.tokens {
			t.Errorf("codex-subagents.jsonl: thought %d: %+v origin %q, want %d ms and %d tokens", i+1, x, th[i].Origin, want.span, want.tokens)
		}
	}
	if x := th[3].Thought; !x.Dropped || th[3].Origin != event.Mock {
		t.Errorf("codex-subagents.jsonl: the last step: %+v origin %q", x, th[3].Origin)
	}
}
