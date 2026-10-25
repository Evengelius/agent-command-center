//go:build timing

package app

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/Evengelius/agent-command-center/internal/event"
	"github.com/Evengelius/agent-command-center/internal/provider/claude"
)

// TestToolsToFrame is plan 49's criterion 49.7 (design D6; audit finding 7):
// a large tool history while agents run. One open turn of 2,000 tool calls,
// 30 of them Write calls of 100 KB, the turn's stage unfolded as an open
// one is, while 8 agents run with 50 calls each. The last tenth of the
// events are each timed from Update to their frame, at 200x60; the target
// is the timing step's 16 ms at the 95th percentile. Run with the timing
// step, alone (evaluation.md, Keystroke timing).
func TestToolsToFrame(t *testing.T) {
	const (
		calls  = 2000
		writes = 30
		agents = 8
		each   = 50
	)
	h, cli := startScript(t, claude.ProviderID)
	h.send(tea.WindowSizeMsg{Width: 200, Height: 60})
	at := time.Date(2026, 10, 25, 9, 0, 0, 0, time.UTC)
	seq := uint64(0)
	var evs []event.Event
	add := func(k event.Kind, origin event.Origin, agent string, p any) {
		seq++
		at = at.Add(10 * time.Millisecond)
		evs = append(evs, event.Event{Seq: seq, At: at, Provider: cli.id, Kind: k, Origin: origin, Agent: agent, Payload: p})
	}
	add(event.SessionSpawned, event.Observed, "", event.Spawned{})
	add(event.SessionReady, event.Observed, "", event.Ready{})
	add(event.SessionStarted, event.Observed, "", event.Started{SessionID: "s1", Version: "2.1.300", PermissionMode: "default", ApprovalMode: event.ModeManual})
	add(event.UserMessage, event.Observed, "", event.User{Text: "Write the generated files and check them."})
	add(event.TurnStarted, event.Observed, "", nil)
	for j := range agents {
		add(event.AgentStarted, event.Observed, "", event.Agent{Key: fmt.Sprintf("a%d", j), Label: "checker", Type: "subagent", Description: "check a package"})
	}
	content := strings.Repeat("// a generated line of the file, long enough to wrap in no row\n", 100_000/64)
	read := json.RawMessage(`{"file_path":"internal/render/telemetry.go"}`)
	agentCall := 0
	for k := range calls {
		in := read
		name := "Read"
		if k%(calls/writes) == 0 {
			b, _ := json.Marshal(map[string]string{"file_path": fmt.Sprintf("internal/gen/file%02d.go", k/(calls/writes)), "content": content})
			in, name = b, "Write"
		}
		id := fmt.Sprintf("o%d", k)
		add(event.ToolStarted, event.Observed, "", event.Tool{ToolID: id, Name: name, Input: in})
		add(event.ToolCompleted, event.Observed, "", event.Tool{ToolID: id, Name: name, Output: "ok"})
		if k%(calls/(agents*each)) == 0 && agentCall < agents*each {
			key, tid := fmt.Sprintf("a%d", agentCall%agents), fmt.Sprintf("c%d", agentCall)
			add(event.ToolStarted, event.Observed, key, event.Tool{ToolID: tid, Name: "Grep", Input: json.RawMessage(`{"pattern":"func "}`)})
			add(event.ToolCompleted, event.Observed, key, event.Tool{ToolID: tid, Name: "Grep", Output: "3 matches"})
			agentCall++
		}
	}
	timed := len(evs) / 10
	var ds []time.Duration
	for i, ev := range evs {
		msg := eventsMsg{sess: h.m.sess, evs: []event.Event{ev}}
		if i < len(evs)-timed {
			h.m.Update(msg)
			continue
		}
		ds = append(ds, frameTime(h, msg))
	}
	s := h.m.State()
	if len(s.Tools) != calls+agents*each || len(s.Agents) != agents || s.Stages[0].Open != true {
		t.Fatalf("the generated turn was not applied: %d calls, %d agents", len(s.Tools), len(s.Agents))
	}
	p50, p95, worst := percentile(ds, 0.50), percentile(ds, 0.95), percentile(ds, 1)
	t.Logf("event-to-frame, %d calls (%d Write calls of %d KB) in one open turn, %d agents running, 200x60, the last %d events: p50 %v, p95 %v, max %v (target p95 ≤ %v)",
		len(s.Tools), writes, len(content)/1000, agents, timed, p50, p95, worst, streamTarget)
	if p95 > streamTarget {
		overTarget(t, "p95 %v over the %v target", p95, streamTarget)
	}
}
