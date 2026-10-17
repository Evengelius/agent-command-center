//go:build timing

package app

import (
	"encoding/json"
	"fmt"
	"runtime"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/Evengelius/agent-command-center/internal/event"
	"github.com/Evengelius/agent-command-center/internal/provider/claude"
)

// TestLongSession is test 11 (reliability.md, Test plan; plan 4, step 7,
// criterion 6): two hours of generated events applied at speed through the
// model, a frame drawn every frameEvery events. It reports the heap and the
// sizes of the lists the state keeps, and fails when the frame time at the
// end is over the 16 ms target at the 95th percentile. Run with the timing
// step, alone (evaluation.md, Keystroke timing).
func TestLongSession(t *testing.T) {
	const (
		span       = 2 * time.Hour
		turns      = 240 // one every 30 s: a busy session
		agents     = 3   // subagents per turn, each with its own tools
		tools      = 20  // the orchestrator's tools per turn
		deltas     = 30  // answer lines per turn, one per delta
		frameEvery = 50
		endFrames  = 400
	)
	h, cli := startScript(t, claude.ProviderID)
	h.send(tea.WindowSizeMsg{Width: 200, Height: 60})
	t0 := time.Now().Add(-span)
	seq, n := uint64(1_000_000), 0
	var during []time.Duration
	send := func(at time.Time, k event.Kind, agent string, p any) {
		seq++
		n++
		msg := eventsMsg{sess: h.m.sess, evs: []event.Event{{Seq: seq, At: at, Provider: cli.id, Kind: k, Origin: event.Observed, Agent: agent, Payload: p}}}
		if n%frameEvery == 0 {
			during = append(during, frameTime(h, msg))
			return
		}
		h.m.Update(msg)
	}
	input := json.RawMessage(`{"file_path":"internal/render/telemetry.go","pattern":"func "}`)

	runtime.GC()
	var before runtime.MemStats
	runtime.ReadMemStats(&before)
	start := time.Now()
	for turn := 0; turn < turns; turn++ {
		at := t0.Add(time.Duration(turn) * span / turns)
		tick := func() time.Time { at = at.Add(100 * time.Millisecond); return at }
		id := fmt.Sprintf("m%d", turn)
		send(tick(), event.UserMessage, "", event.User{Text: fmt.Sprintf("Turn %d: look at the telemetry code and tell me what changed.", turn)})
		send(tick(), event.TurnStarted, "", nil)
		for j := 0; j < agents; j++ {
			key := fmt.Sprintf("a%d-%d", turn, j)
			send(tick(), event.AgentStarted, "", event.Agent{Key: key, Label: "explorer", Type: "subagent", Description: "scan the render package"})
			for k := 0; k < 3; k++ {
				tid := fmt.Sprintf("t%d-%d-%d", turn, j, k)
				send(tick(), event.ToolStarted, key, event.Tool{ToolID: tid, Name: "Grep", Input: input})
				send(tick(), event.ToolCompleted, key, event.Tool{ToolID: tid, Name: "Grep", Output: "internal/render/telemetry.go:42: func missing("})
				send(tick(), event.AgentProgress, key, event.Agent{Key: key, Tokens: int64(1000 * (k + 1)), HasTokens: true, ToolUses: k + 1, LastTool: "Grep"})
			}
			send(tick(), event.AgentCompleted, "", event.Agent{Key: key, Status: "completed", Summary: "Found the telemetry rows."})
		}
		for k := 0; k < tools; k++ {
			tid := fmt.Sprintf("o%d-%d", turn, k)
			send(tick(), event.ToolStarted, "", event.Tool{ToolID: tid, Name: "Read", Input: input})
			send(tick(), event.ToolCompleted, "", event.Tool{ToolID: tid, Name: "Read", Output: "120 lines"})
		}
		send(tick(), event.DiffUpdated, "", event.Diff{Path: fmt.Sprintf("internal/file%02d.go", turn%20), Text: "@@ -1 +1 @@\n-old\n+new\n"})
		for k := 0; k < deltas; k++ {
			send(tick(), event.MessageDelta, "", event.Delta{MessageID: id, Block: "text", Text: answerLine(k) + "\n"})
		}
		send(tick(), event.MessageCompleted, "", event.Completed{MessageID: id, Model: "claude-haiku-5-5"})
		send(tick(), event.Telemetry, "", event.Metrics{Items: []event.Metric{
			{Name: "quota.5h", Value: float64(turn % 100), Unit: "%", ResetsAt: at.Add(time.Hour), Source: "rate_limit_event"},
			{Name: "tokens.session", Value: float64(10_000 * (turn + 1)), Unit: "tokens", Documented: true, Source: "result.modelUsage"},
			{Name: "cost.usd", Value: 0.01 * float64(turn+1), Unit: "USD", Documented: true, Source: "result.total_cost_usd"},
		}})
		if turn%10 == 9 {
			send(tick(), event.Retry, "", event.Problem{Message: "API Error: 529 Overloaded", WillRetry: true})
			send(tick(), event.Error, "", event.Problem{Message: fmt.Sprintf("tool failed in turn %d", turn)})
		}
		send(tick(), event.TurnCompleted, "", event.TurnEnd{Outcome: "completed", DurationMS: 25_000})
	}
	elapsed := time.Since(start)
	runtime.GC()
	var after runtime.MemStats
	runtime.ReadMemStats(&after)

	// The frame time at the end: an answer streamed into the two-hour state.
	at := time.Now()
	send(at, event.TurnStarted, "", nil)
	var end []time.Duration
	for k := 0; k < endFrames; k++ {
		seq++
		end = append(end, frameTime(h, eventsMsg{sess: h.m.sess, evs: []event.Event{{Seq: seq, At: at, Provider: cli.id, Kind: event.MessageDelta, Origin: event.AI, Payload: event.Delta{MessageID: "end", Block: "text", Text: answerLine(k) + "\n"}}}}))
	}

	s := h.m.State()
	if len(s.Errors) == 0 || len(s.Agents) != turns*agents {
		t.Fatalf("the generated session was not applied: %d errors, %d agents", len(s.Errors), len(s.Agents))
	}
	t.Logf("%d events over %v of session time, applied in %v (%.0f events/s), a frame every %d", n, span, elapsed.Round(time.Millisecond), float64(n)/elapsed.Seconds(), frameEvery)
	t.Logf("heap in use %.1f MiB after (%.1f MiB before), %d GC cycles", float64(after.HeapAlloc)/(1<<20), float64(before.HeapAlloc)/(1<<20), after.NumGC-before.NumGC)
	t.Logf("kept: conversation %d, stages %d, agents %d, tools %d, diffs %d, log %d, errors %d, notes %d", len(s.Conversation), len(s.Stages), len(s.Agents), len(s.Tools), len(s.Diffs), len(s.Log), len(s.Errors), len(s.Notes))
	first, last := during[:len(during)/10], during[len(during)-len(during)/10:]
	t.Logf("frames during: first tenth p95 %v, last tenth p95 %v", percentile(first, 0.95), percentile(last, 0.95))
	p50, p95, worst := percentile(end, 0.50), percentile(end, 0.95), percentile(end, 1)
	t.Logf("delta-to-frame at the end, 200x60, %d deltas: p50 %v, p95 %v, max %v (target p95 ≤ %v)", endFrames, p50, p95, worst, streamTarget)
	if p95 > streamTarget {
		overTarget(t, "p95 %v over the %v target", p95, streamTarget)
	}
}
