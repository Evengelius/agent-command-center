package state_test

import (
	"testing"

	"github.com/Evengelius/agent-command-center/internal/event"
	"github.com/Evengelius/agent-command-center/internal/provider/claude"
	"github.com/Evengelius/agent-command-center/internal/provider/codex"
)

// Plan 22, amendment 20: a Codex phase keeps when its reasoning item started
// (observed) and the item's summary, beside the step's inferred start.

const (
	p22s = "p22s-codex-reasoning-summary.jsonl"
	p22o = "p22o-claude-opus-thinking.jsonl"
)

// 20.3: P22s through the reducer. Two phases: their reasoning items started at
// 6,882 and 11,397 ms and completed at 9,388 and 14,833 ms; their starts are
// the steps', inferred, 5,252 and 9,849 ms; their summaries are none and
// "**Computing f(10)**", their counts 28 and 36. Times are ms from the epoch,
// the recording's t_ms.
func TestThinkingCodexP22s(t *testing.T) {
	s, _ := replayState(t, p22s, codex.ProviderID, codex.NewMapper())
	th := thoughts(s)
	if len(th) != 2 {
		t.Fatalf("%d phases, want 2", len(th))
	}
	for i, want := range []struct {
		start, itemStart, end, span, tokens int64
		summary, step                       string
	}{
		{5252, 6882, 9388, 4136, 28, "", "step:23"},
		{9849, 11397, 14833, 4984, 36, "**Computing f(10)**", "step:71"},
	} {
		e, x := th[i], th[i].Thought
		if !x.Inferred || !x.Confirmed || x.Dropped || x.Live() || x.Outcome != "" || x.EndInferred || x.EndNext || x.Closed || e.Streaming {
			t.Errorf("phase %d: %+v", i+1, x)
		}
		if e.Origin != event.Inferred || e.MessageID != want.step || e.Agent != "" {
			t.Errorf("phase %d: entry %q origin %q agent %q, want the %s step, inferred", i+1, e.MessageID, e.Origin, e.Agent, want.step)
		}
		if got := since(x.Start); got != want.start {
			t.Errorf("phase %d: start %d ms, want the step's %d", i+1, got, want.start)
		}
		if got := since(x.ItemStart); got != want.itemStart {
			t.Errorf("phase %d: reasoning item started at %d ms, want %d", i+1, got, want.itemStart)
		}
		if got := since(x.End); got != want.end || span(x) != want.span {
			t.Errorf("phase %d: ended at %d ms after %d ms, want %d after %d", i+1, got, span(x), want.end, want.span)
		}
		if x.Summary != want.summary {
			t.Errorf("phase %d: summary %q, want %q", i+1, x.Summary, want.summary)
		}
		if !x.HasTokens || x.Tokens != want.tokens || x.Shared != 0 || !x.Counted || x.Source != "thread/tokenUsage/updated last.reasoningOutputTokens" || x.Item == "" {
			t.Errorf("phase %d: %d tokens (has %v, shared %d) from %q, item %q", i+1, x.Tokens, x.HasTokens, x.Shared, x.Source, x.Item)
		}
		// The item starts after its step does and ends where the phase does.
		if !x.ItemStart.After(x.Start) || !x.ItemStart.Before(x.End) {
			t.Errorf("phase %d: the item (%v) is not inside the step (%v to %v)", i+1, x.ItemStart, x.Start, x.End)
		}
	}
}

// A Claude Code phase has no reasoning item and no Codex summary, and a Codex
// recording without summaries has items with a start and none.
func TestReasoningItemFieldsByCLI(t *testing.T) {
	for _, name := range []string{p303b, p12cBlock, p22o} {
		s, _ := replayState(t, name, claude.ProviderID, claude.NewMapper())
		for i, e := range thoughts(s) {
			if x := e.Thought; !x.ItemStart.IsZero() || x.Summary != "" {
				t.Errorf("%s phase %d: item start %v, summary %q; want none", name, i+1, x.ItemStart, x.Summary)
			}
		}
	}
	for _, name := range []string{p7, p6b, p9a} {
		s, _ := replayState(t, name, codex.ProviderID, codex.NewMapper())
		n := 0
		for i, e := range thoughts(s) {
			x := e.Thought
			if x.Summary != "" {
				t.Errorf("%s phase %d: summary %q, want none (the recording asked for none)", name, i+1, x.Summary)
			}
			if x.Confirmed {
				n++
				// Without summaries the item starts and completes within 10 ms.
				if x.ItemStart.IsZero() || x.End.Sub(x.ItemStart).Milliseconds() < 0 || x.End.Sub(x.ItemStart).Milliseconds() > 10 {
					t.Errorf("%s phase %d: the item ran %v to %v", name, i+1, x.ItemStart, x.End)
				}
			}
		}
		if n == 0 {
			t.Errorf("%s: no confirmed phase", name)
		}
	}
}

// The item's start is kept on both paths: the item confirms the step that is
// open (the phase keeps the step's start), or finds none open and is a phase of
// its own, whose start is the turn's last usage update when there was one (plan
// 22, correction C5); its summary comes with its completion, whenever that
// reaches the phase of that item, and a completion without one clears none.
func TestReasoningItemStartAndSummary(t *testing.T) {
	complete := func(b *bench, ms int64, id, summary string) {
		b.marker(ms, "", event.Marker{MessageID: id, Name: "item_completed", Block: "thinking", Summary: summary})
	}

	t.Run("an item confirming the open step", func(t *testing.T) {
		b := newBench(t)
		b.step(100)
		if x := b.one().Thought; !x.ItemStart.IsZero() || x.Summary != "" {
			t.Fatalf("a step no item confirmed: %+v", x)
		}
		b.reasoningStart(500, "", "rs1")
		x := b.one().Thought
		if !x.ItemStart.Equal(at(500)) || !x.Start.Equal(at(100)) || !x.Inferred || !x.Confirmed || !x.Live() {
			t.Fatalf("confirmed: item start %v, start %v: %+v", x.ItemStart, x.Start, x)
		}
		complete(b, 900, "rs1", "**Heading**\n\nbody")
		x = b.one().Thought
		if x.Live() || !x.End.Equal(at(900)) || !x.ItemStart.Equal(at(500)) || !x.Start.Equal(at(100)) || x.Summary != "**Heading**\n\nbody" {
			t.Errorf("completed: %+v", x)
		}
	})

	t.Run("an item with no step open", func(t *testing.T) {
		b := newBench(t)
		b.reasoningStart(500, "", "rs0")
		x := b.one().Thought
		if !x.ItemStart.Equal(at(500)) || !x.Start.IsZero() || !x.Inferred || !x.Confirmed {
			t.Fatalf("before any usage: item start %v, start %v: %+v", x.ItemStart, x.Start, x)
		}
		complete(b, 501, "rs0", "")
		b.usage(600, "", 12)
		b.reasoningStart(900, "", "rs1")
		x = b.all()[1].Thought
		if !x.ItemStart.Equal(at(900)) || !x.Start.Equal(at(600)) || !x.Inferred || !x.Confirmed {
			t.Fatalf("after the turn's usage at 600 ms: item start %v, start %v: %+v", x.ItemStart, x.Start, x)
		}
		complete(b, 1500, "rs1", "second item")
		if x = b.all()[1].Thought; x.Summary != "second item" || !x.End.Equal(at(1500)) || !x.ItemStart.Equal(at(900)) {
			t.Errorf("completed: %+v", x)
		}
		if b.all()[0].Thought.Summary != "" {
			t.Errorf("the first item took the second's summary: %+v", b.all()[0].Thought)
		}
	})

	t.Run("a completion goes to its own item", func(t *testing.T) {
		b := newBench(t)
		b.step(100)
		b.reasoningStart(200, "", "rs1")
		complete(b, 300, "rs-other", "not this one's") // an unknown item
		if x := b.one().Thought; x.Summary != "" || !x.Live() {
			t.Fatalf("another item's completion changed the phase: %+v", x)
		}
		complete(b, 400, "rs1", "this one's")
		if x := b.one().Thought; x.Summary != "this one's" || !x.End.Equal(at(400)) {
			t.Fatalf("its own: %+v", x)
		}
		// Again, with no summary: nothing is cleared, and the end stays.
		complete(b, 900, "rs1", "")
		if x := b.one().Thought; x.Summary != "this one's" || !x.End.Equal(at(400)) {
			t.Errorf("a repeated completion with no summary: %+v", x)
		}
	})

	t.Run("a child's item is no phase and keeps no summary", func(t *testing.T) {
		b := newBench(t)
		b.step(100)
		b.marker(200, "child", event.Marker{MessageID: "rsc", Name: "item_started", Block: "thinking"})
		b.marker(300, "child", event.Marker{MessageID: "rsc", Name: "item_completed", Block: "thinking", Summary: "the child's"})
		if x := b.one().Thought; x.Summary != "" || !x.ItemStart.IsZero() || x.Confirmed {
			t.Errorf("a child's item reached the orchestrator's step: %+v", x)
		}
	})

	t.Run("a completion after the phase's end inferred", func(t *testing.T) {
		// The turn's end cut the item: the phase ended there. Its completion comes
		// late, in the same exchange: the end stays the inferred one, the summary
		// is Codex's text for that item and is kept.
		b := newBench(t)
		b.step(100)
		b.reasoningStart(200, "", "rs1")
		b.turnEnd(500, "interrupted")
		complete(b, 700, "rs1", "late")
		x := b.one().Thought
		if !x.End.Equal(at(500)) || !x.EndInferred || x.Outcome != "interrupted" || x.Summary != "late" || !x.ItemStart.Equal(at(200)) {
			t.Errorf("late completion: %+v", x)
		}
	})
}

// The two recordings of amendment 20 are in the replay list (TestReplayDeterministic
// replays each twice to the same state, test 4); nothing in either is
// unrecognised.
func TestAmendment20RecordingsAreKnown(t *testing.T) {
	seen := 0
	for _, r := range recordings {
		if r.name != p22s && r.name != p22o {
			continue
		}
		seen++
		if s, n := replayState(t, r.name, r.provider, r.mapper()); s.Unrecognised != 0 || n == 0 {
			t.Errorf("%s: %d events, %d unrecognised", r.name, n, s.Unrecognised)
		}
	}
	if seen != 2 {
		t.Errorf("%d of the 2 recordings are in the replay list", seen)
	}
}
