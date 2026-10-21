package state

import (
	"testing"

	"github.com/Evengelius/agent-command-center/internal/event"
)

// Claude Code sends an assistant line per content block, each with the
// message's id, while the message goes on streaming (P2, line 63 then 69:
// the thinking block's line comes before the text deltas). The second L2 run
// (2026-10-16) streamed 300 lines into an entry such a line had marked
// finished. One message stays one entry, streaming until message_stop, and
// no block's line takes away text another block streamed.
func TestAnswerStreamsAcrossBlockLines(t *testing.T) {
	delta := func(b *bench, block, text string) {
		b.ev(event.MessageDelta, "", event.Delta{MessageID: "m1", Block: block, Text: text})
	}
	line := func(b *bench, text string, thinking bool) {
		b.ev(event.MessageCompleted, "", event.Completed{MessageID: "m1", Text: text, Thinking: thinking})
	}
	stop := func(b *bench) {
		b.ev(event.MessageMarker, "", event.Marker{MessageID: "m1", Name: "message_stop"})
	}
	answer := func(b *bench) (Entry, int) {
		b.t.Helper()
		var got []Entry
		for _, e := range b.s.Conversation {
			if e.Role == "assistant" {
				got = append(got, e)
			}
		}
		if len(got) != 1 {
			b.t.Fatalf("%d assistant entries, want 1: %+v", len(got), got)
		}
		return got[0], len(got)
	}

	t.Run("thinking, then text", func(t *testing.T) {
		b := newBench(t).running()
		delta(b, "thinking", "")
		line(b, "", true)
		// The orchestrator's thinking is a thinking entry of its own (plan 22,
		// item 9): a thinking delta and a thinking line make no assistant entry.
		for _, e := range b.s.Conversation {
			if e.Role == "assistant" {
				t.Fatalf("an assistant entry before any text: %+v", e)
			}
		}
		delta(b, "text", "one\n")
		e, _ := answer(b)
		b.eq("streaming after the thinking line and a text delta", e.Streaming, true)
		delta(b, "text", "two\n")
		e, _ = answer(b)
		b.eq("text", e.Text, "one\ntwo\n")
		b.eq("no thinking mark: the thinking entry stands for it", e.Thinking, false)
		line(b, "one\ntwo\n", false)
		stop(b)
		e, _ = answer(b)
		b.eq("text at the end", e.Text, "one\ntwo\n")
		b.eq("streaming after message_stop", e.Streaming, false)
		b.eq("noted a difference", b.noted("differs"), false)
	})

	t.Run("text, then a thinking line", func(t *testing.T) {
		b := newBench(t).running()
		delta(b, "text", "A")
		line(b, "A", false)
		line(b, "", true)
		e, _ := answer(b)
		b.eq("text kept", e.Text, "A")
		b.eq("thinking", e.Thinking, true)
	})

	// A subagent's thinking keeps its mark on its message (plan 22, item 9):
	// a delta or a line marks the message, and none reaches the orchestrator's
	// record.
	t.Run("a subagent's thinking", func(t *testing.T) {
		b := newBench(t).running()
		b.ev(event.MessageDelta, "a1", event.Delta{MessageID: "s1", Block: "thinking"})
		b.ev(event.MessageCompleted, "a1", event.Completed{MessageID: "s1", Thinking: true})
		b.ev(event.MessageDelta, "a1", event.Delta{MessageID: "s2", Block: "text", Text: "hi"})
		b.ev(event.MessageCompleted, "a1", event.Completed{MessageID: "s2", Thinking: true})
		msgs := b.agent("a1").Messages
		if len(msgs) != 2 {
			t.Fatalf("%d messages, want 2: %+v", len(msgs), msgs)
		}
		b.eq("the thinking delta marks its message", msgs[0].Thinking, true)
		b.eq("a thinking line marks the message it completes", msgs[1].Thinking, true)
		for _, e := range b.s.Conversation {
			if e.Role == "thinking" || e.Role == "assistant" {
				t.Errorf("a subagent's message in the orchestrator's record: %+v", e)
			}
		}
	})

	t.Run("two text blocks", func(t *testing.T) {
		b := newBench(t).running()
		delta(b, "text", "Let me look. ")
		line(b, "Let me look. ", false)
		delta(b, "text", "Done.")
		line(b, "Done.", false)
		stop(b)
		e, _ := answer(b)
		b.eq("text", e.Text, "Let me look. Done.")
		b.eq("noted a difference", b.noted("differs"), false)
	})

	t.Run("a line the stream contradicts", func(t *testing.T) {
		b := newBench(t).running()
		delta(b, "text", "garbled")
		line(b, "the answer", false)
		e, _ := answer(b)
		b.eq("the completed text wins", e.Text, "the answer")
		b.eq("noted a difference", b.noted("differs"), true)
	})

	t.Run("a whole message (Codex)", func(t *testing.T) {
		b := newBench(t).running()
		delta(b, "text", "Hel")
		line(b, "Hello", false)
		e, _ := answer(b)
		b.eq("text", e.Text, "Hello")
		b.eq("streaming after the completed message", e.Streaming, false)
		b.eq("noted a difference", b.noted("differs"), false)
	})
}
