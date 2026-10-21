package codex

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"github.com/Evengelius/agent-command-center/internal/event"
	"github.com/Evengelius/agent-command-center/internal/fixture"
)

// The orchestrator's thread and turn in P7 (testdata/probes/p7-codex-subagents.jsonl).
const (
	p7Main = "01a1419a-23c6-74f1-837d-b5da2cbe1495"
	p7Turn = "01a1419a-253e-7461-9514-7e3acbeebd84"
)

// stepMarkers are the events of a model step's start.
func stepMarkers(evs []event.Event) []event.Event {
	var out []event.Event
	for _, e := range evs {
		if p, ok := e.Payload.(event.Marker); ok && e.Kind == event.MessageMarker && p.Name == event.StepStarted {
			out = append(out, e)
		}
	}
	return out
}

// reasoningLast are the ReasoningLast metrics of the usage updates, by agent
// ("" is the orchestrator), in order.
func reasoningLast(evs []event.Event) map[string][]float64 {
	out := map[string][]float64{}
	for _, e := range evs {
		m, ok := e.Payload.(event.Metrics)
		if !ok {
			continue
		}
		for _, it := range m.Items {
			if it.Name == event.ReasoningLast {
				out[e.Agent] = append(out[e.Agent], it.Value)
			}
		}
	}
	return out
}

// Plan 22, criterion 1.2 (design 4, X5): P7's orchestrator gets a step_started
// marker, inferred, at its userMessage item and at each usage update of a step
// that made a call, once the step's tools had completed: 5 markers at 3,380,
// 7,120, 9,632, 13,471 and 24,076 ms. The last step answered with a message
// alone (its usage at 25,329 ms) and starts none; no child thread has one.
func TestP7StepStarted(t *testing.T) {
	_, per := replay(t, "p7-codex-subagents.jsonl")
	all := fixture.Flatten(per)
	got := stepMarkers(all)
	want := []int64{3380, 7120, 9632, 13471, 24076}
	if len(got) != len(want) {
		t.Fatalf("%d step markers, want %d", len(got), len(want))
	}
	for i, e := range got {
		if at := e.At.Sub(fixture.Epoch).Milliseconds(); at != want[i] {
			t.Errorf("marker %d at %d ms, want %d", i, at, want[i])
		}
		if e.Origin != event.Inferred || e.Agent != "" || e.Turn != p7Turn || e.Item != "" || e.Provider != ProviderID {
			t.Errorf("marker %d: origin %q, agent %q, turn %q, item %q, provider %q; want inferred, the orchestrator, %s, no item", i, e.Origin, e.Agent, e.Turn, e.Item, e.Provider, p7Turn)
		}
		if p := e.Payload.(event.Marker); p.MessageID != "" || p.Block != "" || p.Index != 0 || p.HasThinking {
			t.Errorf("marker %d payload %+v, want a bare step_started", i, p)
		}
	}
	// Each comes right after the event that gave it away, in the same record:
	// the userMessage item first, then the orchestrator's usage updates.
	n := 0
	for i, evs := range per {
		for j, e := range evs {
			p, ok := e.Payload.(event.Marker)
			if !ok || p.Name != event.StepStarted {
				continue
			}
			n++
			if j != len(evs)-1 || j == 0 {
				t.Errorf("record %d: the step marker is event %d of %d, want it last after its trigger", i+1, j, len(evs))
				continue
			}
			trigger := evs[j-1]
			switch {
			case n == 1:
				if tr, ok := trigger.Payload.(event.Marker); !ok || tr.Name != "item_started" || tr.Block != "user" {
					t.Errorf("first marker follows %v %+v, want the userMessage item_started", trigger.Kind, trigger.Payload)
				}
			default:
				if trigger.Kind != event.Telemetry || trigger.Agent != "" || !strings.Contains(trigger.Raw, `"method":"thread/tokenUsage/updated"`) || !strings.Contains(trigger.Raw, p7Main) {
					t.Errorf("marker %d follows %v of agent %q, want the orchestrator's usage update", n, trigger.Kind, trigger.Agent)
				}
			}
			if e.Raw != trigger.Raw {
				t.Errorf("marker %d keeps another line than its trigger's", n)
			}
		}
	}
}

// Plan 22, criterion 1.2 (X2): every usage update gives its last
// reasoningOutputTokens as a metric, by thread: P7's orchestrator 83, 0, 8, 7,
// 0; its children their own, tagged with their thread.
func TestP7ReasoningLast(t *testing.T) {
	_, per := replay(t, "p7-codex-subagents.jsonl")
	all := fixture.Flatten(per)
	got := reasoningLast(all)
	if want := []float64{83, 0, 8, 7, 0}; !slices.Equal(got[""], want) {
		t.Errorf("orchestrator reasoning tokens %v, want %v", got[""], want)
	}
	const first, second = "01a1419a-384f-7d13-bd22-b375637ac31b", "01a1419a-4212-7712-b732-30b9c9bd385c"
	if want := []float64{14, 12}; !slices.Equal(got[first], want) {
		t.Errorf("first child's reasoning tokens %v, want %v", got[first], want)
	}
	if want := []float64{25, 10, 15, 12}; !slices.Equal(got[second], want) {
		t.Errorf("second child's reasoning tokens %v, want %v", got[second], want)
	}
	if len(got) != 3 {
		t.Errorf("reasoning tokens for %d threads, want 3", len(got))
	}
	for _, e := range all {
		m, ok := e.Payload.(event.Metrics)
		if !ok {
			continue
		}
		for _, it := range m.Items {
			if it.Name != event.ReasoningLast {
				continue
			}
			if it.Unit != "tokens" || !it.Documented || it.Source != "thread/tokenUsage/updated last.reasoningOutputTokens" || it.Reason != "" || it.Derived {
				t.Errorf("metric %+v", it)
			}
		}
	}
	// A child's usage update produces no step marker.
	for _, e := range stepMarkers(all) {
		if e.Agent != "" {
			t.Errorf("step marker for agent %q", e.Agent)
		}
	}
}

// Every usage update of every app-server recording gives its reasoning tokens
// exactly as the line says, read apart from the mapper; an update without the
// field gives none, and so does one whose total repeats its thread's last
// (P6b line 87, Codex 0.145.0 on an interrupt: plan 22, the review).
func TestReasoningLastMatchesEveryUsageLine(t *testing.T) {
	total, repeats := 0, 0
	for _, name := range appServer {
		recs, per := replay(t, name)
		last := map[string]string{} // each thread's last total, as sent
		for i, r := range recs {
			if r.Dir != "out" || !strings.Contains(string(r.Msg), `"thread/tokenUsage/updated"`) {
				continue
			}
			var l struct {
				Params struct {
					ThreadID   string `json:"threadId"`
					TokenUsage struct {
						Total json.RawMessage `json:"total"`
						Last  map[string]any  `json:"last"`
					} `json:"tokenUsage"`
				} `json:"params"`
			}
			if err := json.Unmarshal(r.Msg, &l); err != nil {
				t.Fatalf("%s line %d: %v", name, i+1, err)
			}
			raw, has := l.Params.TokenUsage.Last["reasoningOutputTokens"].(float64)
			th, tot := l.Params.ThreadID, string(l.Params.TokenUsage.Total)
			if prev, seen := last[th]; seen && prev == tot {
				has = false // a repeat: no count
				repeats++
			}
			last[th] = tot
			var vals []float64
			for _, e := range per[i] {
				if m, ok := e.Payload.(event.Metrics); ok {
					for _, it := range m.Items {
						if it.Name == event.ReasoningLast {
							vals = append(vals, it.Value)
						}
					}
				}
			}
			switch {
			case has && (len(vals) != 1 || vals[0] != raw):
				t.Errorf("%s line %d: reasoning tokens %v, the line says %v", name, i+1, vals, raw)
			case !has && len(vals) != 0:
				t.Errorf("%s line %d: reasoning tokens %v from a line with none", name, i+1, vals)
			}
			total++
		}
	}
	if total == 0 {
		t.Fatal("no usage update in any recording")
	}
	if repeats != 1 {
		t.Errorf("%d repeated updates, want 1 (P6b line 87)", repeats)
	}
	t.Logf("%d usage updates in %d recordings, %d repeated", total, len(appServer), repeats)
}

// Every step marker of every app-server recording is the orchestrator's,
// inferred, last in the record of one of the three moments, and the first of a
// turn comes with its userMessage; a turn never has more markers than its
// orchestrator's usage updates plus one. The count per turn is only logged:
// no recording is read for its content here.
func TestStepMarkersInRecordings(t *testing.T) {
	for _, name := range appServer {
		recs, per := replay(t, name)
		var main string
		for _, e := range fixture.Flatten(per) {
			if e.Kind == event.SessionStarted {
				main = e.Payload.(event.Started).SessionID
			}
		}
		if main == "" {
			t.Fatalf("%s: no main thread", name)
		}
		var turns []string                                    // the turns in order of their first marker
		markers, usages := map[string]int{}, map[string]int{} // by turn
		for i, evs := range per {
			var l struct {
				Method string `json:"method"`
				Params struct {
					ThreadID string `json:"threadId"`
					TurnID   string `json:"turnId"`
					Item     struct {
						Type string `json:"type"`
					} `json:"item"`
				} `json:"params"`
			}
			if recs[i].Dir == "out" {
				_ = json.Unmarshal(recs[i].Msg, &l)
			}
			if l.Method == "thread/tokenUsage/updated" && l.Params.ThreadID == main {
				usages[l.Params.TurnID]++
			}
			for j, e := range evs {
				p, ok := e.Payload.(event.Marker)
				if !ok || p.Name != event.StepStarted {
					continue
				}
				if e.Origin != event.Inferred || e.Agent != "" || j != len(evs)-1 || j == 0 {
					t.Errorf("%s line %d: marker origin %q, agent %q, event %d of %d", name, i+1, e.Origin, e.Agent, j, len(evs))
				}
				if l.Params.ThreadID != main || e.Turn != l.Params.TurnID || e.Turn == "" {
					t.Errorf("%s line %d: marker for turn %q from a line of thread %q, turn %q", name, i+1, e.Turn, l.Params.ThreadID, l.Params.TurnID)
				}
				switch l.Method {
				case "item/started":
					if l.Params.Item.Type != "userMessage" {
						t.Errorf("%s line %d: a marker at an item/started of type %q", name, i+1, l.Params.Item.Type)
					}
					if markers[e.Turn] != 0 {
						t.Errorf("%s line %d: a second userMessage marker in turn %s", name, i+1, e.Turn)
					}
				case "item/completed", "thread/tokenUsage/updated":
					if markers[e.Turn] == 0 {
						t.Errorf("%s line %d: the turn's first marker is not at its userMessage", name, i+1)
					}
				default:
					t.Errorf("%s line %d: a marker at %q", name, i+1, l.Method)
				}
				if markers[e.Turn] == 0 {
					turns = append(turns, e.Turn)
				}
				markers[e.Turn]++
			}
		}
		for k, turn := range turns {
			if markers[turn] > usages[turn]+1 {
				t.Errorf("%s turn %d: %d markers for %d usage updates", name, k+1, markers[turn], usages[turn])
			}
			t.Logf("%s: turn %d: %d step markers, %d orchestrator usage updates", name, k+1, markers[turn], usages[turn])
		}
	}
}

// stepSeq builds the lines of a synthetic orchestrator, one notification each,
// and says after which a step marker is due (plan 22, design 4).
type stepSeq struct {
	t     *testing.T
	lines []stepLine
}

type stepLine struct {
	raw    []byte
	turn   string
	marker bool
	why    string
}

const stepThread, stepChild, stepTurn = "thr-main", "thr-child", "turn-1"

func (s *stepSeq) note(method, thread, turn string, extra map[string]any) []byte {
	s.t.Helper()
	p := map[string]any{"threadId": thread}
	if turn != "" {
		p["turnId"] = turn
	}
	for k, v := range extra {
		p[k] = v
	}
	b, err := json.Marshal(map[string]any{"method": method, "params": p})
	if err != nil {
		s.t.Fatal(err)
	}
	return b
}

// add appends a line; marker is whether a step marker is due after it.
func (s *stepSeq) add(raw []byte, turn string, marker bool, why string) {
	s.lines = append(s.lines, stepLine{raw: raw, turn: turn, marker: marker, why: why})
}

func (s *stepSeq) turnStart(thread, turn string) {
	s.add(s.note("turn/started", thread, turn, map[string]any{"turn": map[string]any{"id": turn}}), turn, false, "turn/started")
}

func (s *stepSeq) turnEnd(thread, turn, status string) {
	s.add(s.note("turn/completed", thread, turn, map[string]any{"turn": map[string]any{"id": turn, "status": status}}), turn, false, "turn/completed")
}

func (s *stepSeq) item(method, thread, turn string, it map[string]any, marker bool, why string) {
	s.add(s.note(method, thread, turn, map[string]any{"item": it}), turn, marker, why)
}

// user is the owner's message item, which begins the turn's first step.
func (s *stepSeq) user(thread, turn string, marker bool) {
	s.item("item/started", thread, turn, map[string]any{"type": "userMessage", "id": "u-" + turn}, marker, "userMessage item/started")
	s.item("item/completed", thread, turn, map[string]any{"type": "userMessage", "id": "u-" + turn}, false, "userMessage item/completed")
}

func (s *stepSeq) message(thread, turn, id string) {
	s.item("item/started", thread, turn, map[string]any{"type": "agentMessage", "id": id}, false, "agentMessage item/started")
	s.item("item/completed", thread, turn, map[string]any{"type": "agentMessage", "id": id, "text": "done"}, false, "agentMessage item/completed")
}

func (s *stepSeq) reasoning(thread, turn, id string) {
	s.item("item/started", thread, turn, map[string]any{"type": "reasoning", "id": id}, false, "reasoning item/started")
	s.item("item/completed", thread, turn, map[string]any{"type": "reasoning", "id": id}, false, "reasoning item/completed")
}

func (s *stepSeq) toolStart(thread, turn, typ, id string) {
	s.item("item/started", thread, turn, map[string]any{"type": typ, "id": id, "status": "inProgress"}, false, typ+" item/started")
}

func (s *stepSeq) toolDone(thread, turn, typ, id string, marker bool, why string) {
	s.item("item/completed", thread, turn, map[string]any{"type": typ, "id": id, "status": "completed"}, marker, why)
}

func (s *stepSeq) spawn(method, thread, turn, id, kind string) {
	s.item(method, thread, turn, map[string]any{"type": "subAgentActivity", "id": id, "kind": kind, "agentThreadId": stepChild, "agentPath": "/root/child"}, false, "subAgentActivity "+kind+" "+method)
}

// usage adds a usage update; each one's total grows, as a model response's
// does, so none reads as a repeat of the one before (plan 22, the review).
func (s *stepSeq) usage(thread, turn string, reasoning any, marker bool, why string) {
	last := map[string]any{"totalTokens": 100}
	if reasoning != nil {
		last["reasoningOutputTokens"] = reasoning
	}
	total := map[string]any{"totalTokens": 100 * (len(s.lines) + 1)}
	s.add(s.note("thread/tokenUsage/updated", thread, turn, map[string]any{"tokenUsage": map[string]any{"total": total, "last": last}}), turn, marker, why)
}

// TestRepeatedUsageCountsNothing: Codex 0.145.0 sends a thread's last usage
// update again when a turn is interrupted (P6b line 87, the same as line 64):
// the repeat reports no model response, so it gives no reasoning count and
// begins no step (plan 22, the review of the change).
func TestRepeatedUsageCountsNothing(t *testing.T) {
	m := NewMapper()
	main := `{"method":"thread/started","params":{"thread":{"id":"t-main"}}}`
	m.Out([]byte(main))
	m.Out([]byte(`{"method":"turn/started","params":{"threadId":"t-main","turn":{"id":"u1"}}}`))
	up := `{"method":"thread/tokenUsage/updated","params":{"threadId":"t-main","turnId":"u1","tokenUsage":{"total":{"totalTokens":500,"reasoningOutputTokens":40},"last":{"totalTokens":500,"reasoningOutputTokens":40}}}}`
	count := func(evs []event.Event) (n int) {
		for _, e := range evs {
			if ms, ok := e.Payload.(event.Metrics); ok {
				for _, it := range ms.Items {
					if it.Name == event.ReasoningLast {
						n++
					}
				}
			}
		}
		return n
	}
	if n := count(m.Out([]byte(up))); n != 1 {
		t.Fatalf("the first update: %d reasoning counts, want 1", n)
	}
	if evs := m.Out([]byte(up)); count(evs) != 0 || len(stepMarkers(evs)) != 0 {
		t.Errorf("the repeat: %d reasoning counts, %d step markers; want none", count(evs), len(stepMarkers(evs)))
	}
	grown := strings.Replace(up, `"total":{"totalTokens":500`, `"total":{"totalTokens":900`, 1)
	if n := count(m.Out([]byte(grown))); n != 1 {
		t.Errorf("the next update, its total grown: %d reasoning counts, want 1", n)
	}
}

// run feeds the lines to a fresh mapper whose main thread is stepThread and
// checks, line by line, that a step marker follows exactly where due: last in
// the line's events, inferred, the orchestrator's, with the line's turn and
// line.
func (s *stepSeq) run() {
	s.t.Helper()
	m := NewMapper()
	m.Out([]byte(`{"method":"thread/started","params":{"thread":{"id":"` + stepThread + `"}}}`))
	for i, l := range s.lines {
		evs := m.Out(l.raw)
		got := stepMarkers(evs)
		if !l.marker {
			if len(got) != 0 {
				s.t.Errorf("line %d (%s): %d step markers, want none", i+1, l.why, len(got))
			}
			continue
		}
		if len(got) != 1 {
			s.t.Errorf("line %d (%s): %d step markers, want 1", i+1, l.why, len(got))
			continue
		}
		e := got[0]
		if last := len(evs) - 1; last < 1 || stepMarkers(evs[last:]) == nil {
			s.t.Errorf("line %d (%s): the marker is not the last of %d events, after its trigger", i+1, l.why, len(evs))
		}
		if e.Origin != event.Inferred || e.Agent != "" || e.Turn != l.turn || e.Raw != string(l.raw) {
			s.t.Errorf("line %d (%s): marker origin %q, agent %q, turn %q; want inferred, the orchestrator, %q", i+1, l.why, e.Origin, e.Agent, e.Turn, l.turn)
		}
	}
}

// Plan 22, criterion 1.2: a step that made no call, only a message, begins no
// other, so the turn's last usage update is followed by no marker.
func TestStepWithoutCallStartsNone(t *testing.T) {
	s := &stepSeq{t: t}
	s.turnStart(stepThread, stepTurn)
	s.user(stepThread, stepTurn, true)
	s.reasoning(stepThread, stepTurn, "r1")
	s.message(stepThread, stepTurn, "m1")
	s.usage(stepThread, stepTurn, 12, false, "usage of a step that made no call")
	s.usage(stepThread, stepTurn, 0, false, "a second usage, still no call")
	s.turnEnd(stepThread, stepTurn, "completed")
	s.run()
}

// Plan 22, design 4 (b): the usual order, the step's tools completing and the
// usage update following (4 to 87 ms later, X2): the marker comes with the
// usage, not with the tool; the next step's message ends the turn.
func TestStepStartsAtTheUsageAfterItsTools(t *testing.T) {
	s := &stepSeq{t: t}
	s.turnStart(stepThread, stepTurn)
	s.user(stepThread, stepTurn, true)
	s.reasoning(stepThread, stepTurn, "r1")
	s.toolStart(stepThread, stepTurn, "commandExecution", "c1")
	s.toolDone(stepThread, stepTurn, "commandExecution", "c1", false, "the only tool completes before the usage")
	s.usage(stepThread, stepTurn, 9, true, "usage after the tools")
	s.message(stepThread, stepTurn, "m1")
	s.usage(stepThread, stepTurn, 0, false, "usage of the last step, a message")
	s.turnEnd(stepThread, stepTurn, "completed")
	s.run()
}

// Plan 22, design 4 (c): a usage update with a tool still running starts
// nothing; the step begins when the last running tool completes.
func TestStepStartsAtTheLastToolAfterTheUsage(t *testing.T) {
	s := &stepSeq{t: t}
	s.turnStart(stepThread, stepTurn)
	s.user(stepThread, stepTurn, true)
	s.toolStart(stepThread, stepTurn, "commandExecution", "c1")
	s.toolStart(stepThread, stepTurn, "mcpToolCall", "c2")
	s.usage(stepThread, stepTurn, 5, false, "usage with two tools running")
	s.toolDone(stepThread, stepTurn, "commandExecution", "c1", false, "one of two tools completes")
	s.toolDone(stepThread, stepTurn, "mcpToolCall", "c2", true, "the last running tool completes")
	s.message(stepThread, stepTurn, "m1")
	s.usage(stepThread, stepTurn, 0, false, "usage of the last step, a message")
	s.turnEnd(stepThread, stepTurn, "completed")
	s.run()
}

// An item/completed with no item/started is not running: it is a call and
// ends nothing, so it neither blocks the usage's marker nor makes one itself.
func TestToolCompletedWithoutStartIsNotRunning(t *testing.T) {
	s := &stepSeq{t: t}
	s.turnStart(stepThread, stepTurn)
	s.user(stepThread, stepTurn, true)
	s.toolStart(stepThread, stepTurn, "fileChange", "c1")
	s.usage(stepThread, stepTurn, 3, false, "usage with a tool running")
	s.toolDone(stepThread, stepTurn, "webSearch", "never-started", false, "a completed item that never started")
	s.toolDone(stepThread, stepTurn, "fileChange", "c1", true, "the running tool completes")
	s.toolDone(stepThread, stepTurn, "sleep", "late", false, "a completed item that never started, in the next step")
	s.usage(stepThread, stepTurn, 4, true, "the next step made a call, the late item")
	s.turnEnd(stepThread, stepTurn, "completed")
	s.run()
}

// Plan 22, design 4: a spawn, a subAgentActivity item of the main thread of
// kind "started", is a call: Codex 0.145.0's (P7, an item/completed with no
// item/started) and 0.163.0's (both items). A child's end told on its parent
// (P7b, "completed"), or any other kind, is not the orchestrator's call: a
// last step of a message alone with a child ending in it starts no step after
// its usage, which would draw a card after the turn's answer. A child's own
// items are its thread's, not the orchestrator's.
func TestSubAgentActivityIsACall(t *testing.T) {
	s := &stepSeq{t: t}
	s.turnStart(stepThread, stepTurn)
	s.user(stepThread, stepTurn, true)
	s.spawn("item/completed", stepThread, stepTurn, "a1", "started") // Codex 0.145.0: no item/started
	s.usage(stepThread, stepTurn, 83, true, "usage of a step with a spawn")
	s.spawn("item/started", stepThread, stepTurn, "a2", "started") // Codex 0.163.0: started, then completed
	s.spawn("item/completed", stepThread, stepTurn, "a2", "started")
	s.usage(stepThread, stepTurn, 7, true, "usage of the next step, a spawn again")
	s.message(stepThread, stepTurn, "m1")
	s.usage(stepThread, stepTurn, 0, false, "usage of the last step, a message alone")
	s.turnEnd(stepThread, stepTurn, "completed")
	s.run()
	for _, kind := range []string{"completed", "interacted", "interrupted"} {
		t.Run(kind, func(t *testing.T) {
			s := &stepSeq{t: t}
			s.turnStart(stepThread, stepTurn)
			s.user(stepThread, stepTurn, true)
			s.spawn("item/started", stepThread, stepTurn, "a1", kind) // a child's activity told on its parent
			s.spawn("item/completed", stepThread, stepTurn, "a1", kind)
			s.message(stepThread, stepTurn, "m1")
			s.usage(stepThread, stepTurn, 0, false, "usage of a last step, a message and a child's "+kind)
			s.turnEnd(stepThread, stepTurn, "completed")
			s.run()
		})
	}
	s = &stepSeq{t: t}
	s.turnStart(stepThread, stepTurn)
	s.user(stepThread, stepTurn, true)
	s.spawn("item/started", stepChild, "child-turn", "a3", "started") // an activity in a child's thread
	s.spawn("item/completed", stepChild, "child-turn", "a3", "started")
	s.usage(stepThread, stepTurn, 1, false, "the orchestrator made no call: its child's activity is not its own")
	s.run()
}

// Every item type tool() handles is a call.
func TestEveryToolTypeIsACall(t *testing.T) {
	for _, typ := range []string{"commandExecution", "fileChange", "mcpToolCall", "dynamicToolCall", "webSearch", "imageView", "imageGeneration", "sleep", "collabAgentToolCall"} {
		t.Run(typ, func(t *testing.T) {
			s := &stepSeq{t: t}
			s.turnStart(stepThread, stepTurn)
			s.user(stepThread, stepTurn, true)
			s.toolStart(stepThread, stepTurn, typ, "c1")
			s.toolDone(stepThread, stepTurn, typ, "c1", false, typ+" completes")
			s.usage(stepThread, stepTurn, 1, true, "usage after a "+typ)
			s.run()
		})
	}
}

// A child thread's events never begin a step, make a call or keep one
// running for the orchestrator (X4).
func TestChildThreadsNeverStartAStep(t *testing.T) {
	s := &stepSeq{t: t}
	s.turnStart(stepThread, stepTurn)
	s.user(stepThread, stepTurn, true)
	s.turnStart(stepChild, "")
	s.user(stepChild, "child-turn", false)
	s.toolStart(stepChild, "child-turn", "commandExecution", "k1")
	s.toolDone(stepChild, "child-turn", "commandExecution", "k1", false, "a child's tool completes")
	s.usage(stepChild, "child-turn", 14, false, "a child's usage")
	s.message(stepThread, stepTurn, "m1")
	s.usage(stepThread, stepTurn, 0, false, "the orchestrator made no call: its children's tools are not its")
	s.run()

	// A child's tool still running does not hold the orchestrator's marker back.
	s = &stepSeq{t: t}
	s.turnStart(stepThread, stepTurn)
	s.user(stepThread, stepTurn, true)
	s.toolStart(stepChild, "child-turn", "commandExecution", "k1")
	s.toolStart(stepThread, stepTurn, "collabAgentToolCall", "w1")
	s.toolDone(stepThread, stepTurn, "collabAgentToolCall", "w1", false, "the orchestrator's wait completes")
	s.usage(stepChild, "child-turn", 3, false, "a child's usage")
	s.usage(stepThread, stepTurn, 8, true, "the orchestrator's usage, a child's tool still running")
	s.turnEnd(stepChild, "child-turn", "completed")
	s.run()
}

// The turn bounds the steps: nothing is inferred outside it, and nothing
// carries over from one turn to the next or from an interrupted one.
func TestStepsFollowTheTurn(t *testing.T) {
	s := &stepSeq{t: t}
	s.usage(stepThread, "turn-0", 5, false, "a usage before any turn")
	s.toolDone(stepThread, "turn-0", "commandExecution", "c0", false, "a tool completing before any turn")
	s.user(stepThread, "turn-0", false) // outside a turn: replayed, not the owner's new message
	s.turnStart(stepThread, "turn-1")
	s.user(stepThread, "turn-1", true)
	s.toolStart(stepThread, "turn-1", "commandExecution", "c1")
	s.turnEnd(stepThread, "turn-1", "interrupted")
	s.toolDone(stepThread, "turn-1", "commandExecution", "c1", false, "a tool completing after the turn ended")
	s.usage(stepThread, "turn-1", 2, false, "a usage after the turn ended")
	s.turnStart(stepThread, "turn-2")
	s.user(stepThread, "turn-2", true)
	s.message(stepThread, "turn-2", "m2")
	s.usage(stepThread, "turn-2", 0, false, "turn 2 made no call: turn 1's does not carry over")
	s.turnEnd(stepThread, "turn-2", "completed")
	s.run()

	// A turn ended while a tool still ran leaves nothing running for the next.
	s = &stepSeq{t: t}
	s.turnStart(stepThread, "turn-1")
	s.user(stepThread, "turn-1", true)
	s.toolStart(stepThread, "turn-1", "commandExecution", "c1")
	s.turnEnd(stepThread, "turn-1", "failed")
	s.turnStart(stepThread, "turn-2")
	s.user(stepThread, "turn-2", true)
	s.toolStart(stepThread, "turn-2", "commandExecution", "c2")
	s.toolDone(stepThread, "turn-2", "commandExecution", "c2", false, "turn 2's tool completes")
	s.usage(stepThread, "turn-2", 1, true, "turn 2's usage: turn 1's tool is not running")
	s.run()
}

// The reasoning metric is the line's value, 0 included; a line without the
// field gives none, never a zero.
func TestReasoningLastMetric(t *testing.T) {
	s := &stepSeq{t: t}
	for _, c := range []struct {
		reasoning any
		want      []float64
	}{{83, []float64{83}}, {0, []float64{0}}, {nil, nil}, {"7", nil}} {
		s.lines = nil
		s.usage(stepThread, stepTurn, c.reasoning, false, "usage")
		m := NewMapper()
		evs := m.Out(s.lines[0].raw)
		if len(evs) != 1 || evs[0].Kind != event.Telemetry {
			t.Fatalf("reasoning %v: %d events, want 1 telemetry", c.reasoning, len(evs))
		}
		var got []float64
		for _, it := range evs[0].Payload.(event.Metrics).Items {
			if it.Name == event.ReasoningLast {
				got = append(got, it.Value)
			}
		}
		if !slices.Equal(got, c.want) {
			t.Errorf("reasoning %v: metrics %v, want %v", c.reasoning, got, c.want)
		}
	}
}
