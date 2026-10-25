package app

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/Evengelius/agent-command-center/internal/event"
	"github.com/Evengelius/agent-command-center/internal/provider/codex"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// TestOutputToFrame is plan 48's criterion 48.5 (design D2; audit finding
// 7a): a running command's output streamed in 8,192 deltas of 8 KiB, 64 MiB
// in all, each delta timed from Update to its frame; then the completion's
// output is the call's. Outside the timing step it streams 512 deltas, and
// the time is only logged.
func TestOutputToFrame(t *testing.T) {
	h, cli := startScript(t, codex.ProviderID)
	seq := uint64(1_000_000) // after anything the start applied
	ev := func(k event.Kind, p any) eventsMsg {
		seq++
		return eventsMsg{sess: h.m.sess, evs: []event.Event{{Seq: seq, Provider: cli.id, Kind: k, Origin: event.Observed, Payload: p}}}
	}
	h.m.Update(ev(event.SessionSpawned, event.Spawned{}))
	h.m.Update(ev(event.SessionReady, event.Ready{Detail: "scripted"}))
	h.m.Update(ev(event.TurnStarted, nil))
	h.m.Update(ev(event.ToolStarted, event.Tool{ToolID: "c1", Name: "commandExecution", Input: json.RawMessage(`{"command":"make all"}`)}))
	var chunk strings.Builder
	for i := 0; chunk.Len() < 8<<10-128; i++ {
		fmt.Fprintf(&chunk, "build step %04d: compiling package %04d of the project, %s\n", i, i, strings.Repeat("·", 20))
	}
	delta := chunk.String() + strings.Repeat("-", 8<<10-chunk.Len()-1) + "\n" // 8 KiB, whole lines
	deltas := 8_192
	if !timingBuild {
		deltas = 512
	}
	ds := make([]time.Duration, 0, deltas)
	for range deltas {
		ds = append(ds, frameTime(h, ev(event.ToolProgress, event.Tool{ToolID: "c1", Output: delta})))
	}
	p50, p95, worst := percentile(ds, 0.50), percentile(ds, 0.95), percentile(ds, 1)
	t.Logf("delta-to-frame, a command's output in %d deltas of %d bytes (%d MiB): p50 %v, p95 %v, max %v (target p95 ≤ %v)",
		deltas, len(delta), deltas*len(delta)>>20, p50, p95, worst, streamTarget)
	if p95 > streamTarget {
		overTarget(t, "p95 %v over the %v target", p95, streamTarget)
	}
	tc := h.m.State().Tools["c1"]
	if tc == nil || len(tc.Output) > state.MaxLiveOutput || !strings.HasSuffix(tc.Output, delta) || tc.OutputCut+int64(len(tc.Output)) != int64(deltas*len(delta)) {
		t.Fatalf("the running call's output: %d bytes kept, %d not kept", len(tc.Output), tc.OutputCut)
	}
	h.m.Update(ev(event.ToolCompleted, event.Tool{ToolID: "c1", Name: "commandExecution", Output: "built\n", ExitCode: 0, HasExitCode: true}))
	if tc := h.m.State().Tools["c1"]; tc.Output != "built\n" || tc.OutputCut != 0 {
		t.Errorf("after the completion: %q, %d bytes not kept", tc.Output, tc.OutputCut)
	}
}
