//go:build timing

package app

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/Evengelius/agent-command-center/internal/event"
	"github.com/Evengelius/agent-command-center/internal/provider/claude"
	"github.com/Evengelius/agent-command-center/internal/render"
)

// TestThinkingFrame is plan 22, criterion 4.2: the frame time while zone B's
// thinking card animates at the end of a long session. Two hours of turns,
// each with three thinking phases (their estimates and exact counts), twenty
// calls and a streamed answer, applied at speed; then a phase live at the
// end, a frame every 50 ms of its clock over 4 s, as the application's
// animation tick draws them (ASM-18), its estimate rising every second.
// Each frame is an update and a view, timed; the target is test 6's, 16 ms at
// the 95th percentile. Run with the timing step, alone (evaluation.md,
// Keystroke timing).
func TestThinkingFrame(t *testing.T) {
	const (
		span   = 2 * time.Hour
		turns  = 240
		phases = 3  // thinking phases per turn, each its own message
		tools  = 20 // the orchestrator's calls per turn
		deltas = 30 // answer lines per turn
		frames = 80 // 4 s of the card, one every 50 ms
	)
	h, cli := startScript(t, claude.ProviderID)
	h.send(tea.WindowSizeMsg{Width: 200, Height: 60})
	t0 := time.Now().Add(-span)
	seq := uint64(1_000_000)
	apply := func(at time.Time, k event.Kind, p any) {
		seq++
		h.m.Update(eventsMsg{sess: h.m.sess, evs: []event.Event{{Seq: seq, At: at, Provider: cli.id, Kind: k, Origin: event.Observed, Payload: p}}})
	}
	// phase is one thinking block of message id, from at: its start, three
	// estimates, its stop and its message's count; the next time after it.
	phase := func(at time.Time, id string) time.Time {
		tick := func(d time.Duration) time.Time { at = at.Add(d); return at }
		apply(tick(0), event.MessageMarker, event.Marker{MessageID: id, Name: "message_start", Model: "claude-haiku-5-5"})
		apply(tick(time.Millisecond), event.MessageMarker, event.Marker{MessageID: id, Name: "content_block_start", Block: "thinking"})
		for k, v := range []float64{50, 350, 650} {
			apply(tick(time.Duration(200+1100*k)*time.Millisecond), event.Telemetry, event.Metrics{Items: []event.Metric{{Name: event.ThinkingEstimate, Value: v, Unit: "tokens", Documented: true, Source: "system/thinking_tokens"}}})
		}
		apply(tick(300*time.Millisecond), event.MessageMarker, event.Marker{MessageID: id, Name: "content_block_stop"})
		apply(tick(500*time.Millisecond), event.MessageMarker, event.Marker{MessageID: id, Name: "message_delta", StopReason: "tool_use", Thinking: 719, HasThinking: true})
		return at
	}
	input := json.RawMessage(`{"file_path":"internal/render/telemetry.go","pattern":"func "}`)
	for turn := 0; turn < turns; turn++ {
		at := t0.Add(time.Duration(turn) * span / turns)
		tick := func() time.Time { at = at.Add(100 * time.Millisecond); return at }
		apply(tick(), event.UserMessage, event.User{Text: fmt.Sprintf("Turn %d: look at the telemetry code and tell me what changed.", turn)})
		apply(tick(), event.TurnStarted, nil)
		for p := 0; p < phases; p++ {
			at = phase(tick(), fmt.Sprintf("m%d-%d", turn, p))
			for k := 0; k < tools/phases; k++ {
				tid := fmt.Sprintf("o%d-%d-%d", turn, p, k)
				apply(tick(), event.ToolStarted, event.Tool{ToolID: tid, Name: "Read", Input: input})
				apply(tick(), event.ToolCompleted, event.Tool{ToolID: tid, Name: "Read", Output: "120 lines"})
			}
		}
		id := fmt.Sprintf("a%d", turn)
		for k := 0; k < deltas; k++ {
			apply(tick(), event.MessageDelta, event.Delta{MessageID: id, Block: "text", Text: answerLine(k) + "\n"})
		}
		apply(tick(), event.MessageCompleted, event.Completed{MessageID: id, Model: "claude-haiku-5-5"})
		apply(tick(), event.TurnCompleted, event.TurnEnd{Outcome: "completed", DurationMS: 25_000})
	}

	// The card at the end: a turn, its thinking block live, its estimate
	// rising every second while the frames come every 50 ms.
	start := time.Now()
	apply(start, event.UserMessage, event.User{Text: "One more turn: think it over."})
	apply(start, event.TurnStarted, nil)
	apply(start, event.MessageMarker, event.Marker{MessageID: "end", Name: "message_start", Model: "claude-haiku-5-5"})
	apply(start, event.MessageMarker, event.Marker{MessageID: "end", Name: "content_block_start", Block: "thinking"})
	var times []time.Duration
	clock := start
	h.m.opt.Now = func() time.Time { return clock }
	for k := 0; k < frames; k++ {
		clock = start.Add(time.Duration(k) * 50 * time.Millisecond)
		var msg tea.Msg = animMsg{}
		if k%20 == 10 { // an estimate every second, its growth drawn by the next frames
			seq++
			msg = eventsMsg{sess: h.m.sess, evs: []event.Event{{Seq: seq, At: clock, Provider: cli.id, Kind: event.Telemetry, Origin: event.Observed,
				Payload: event.Metrics{Items: []event.Metric{{Name: event.ThinkingEstimate, Value: float64(50 + 300*k/20), Unit: "tokens", Documented: true, Source: "system/thinking_tokens"}}}}}}
		}
		times = append(times, frameTime(h, msg))
	}
	s := h.m.State()
	if !render.Animating(s, h.m.ui) || !s.Conversation[len(s.Conversation)-1].Streaming {
		t.Fatal("no card animated at the end")
	}
	thoughts := 0
	for _, e := range s.Conversation {
		if e.Role == "thinking" {
			thoughts++
		}
	}
	if thoughts != turns*phases+1 {
		t.Fatalf("%d thinking entries, want %d", thoughts, turns*phases+1)
	}
	p50, p95, worst := percentile(times, 0.50), percentile(times, 0.95), percentile(times, 1)
	t.Logf("conversation %d entries, %d of them thinking; card frames at 200x60, %d over 4 s: p50 %v, p95 %v, max %v (target p95 ≤ %v)",
		len(s.Conversation), thoughts, frames, p50, p95, worst, streamTarget)
	if p95 > streamTarget {
		overTarget(t, "p95 %v over the %v target", p95, streamTarget)
	}
}
