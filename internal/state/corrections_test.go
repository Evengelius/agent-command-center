package state_test

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/Evengelius/agent-command-center/internal/event"
	"github.com/Evengelius/agent-command-center/internal/fixture"
	"github.com/Evengelius/agent-command-center/internal/provider/claude"
	"github.com/Evengelius/agent-command-center/internal/provider/codex"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// Plan 22, corrections C1 to C5 (plan.md, Corrections C1 to C12; the
// investigation, doc/plan22-investigation.md, sections 2 and 5): the thinking
// phases' counts and ends, from the recordings and from the synthetic
// sequences no recording holds. The tests of C6 to C11 are in internal/render
// and internal/app.

const (
	p2          = "p2-claude-conversation.jsonl"
	p9a         = "p9a-codex-spawn-request.jsonl"
	p12xSandbox = "../../testdata/probes/p12x-sandbox-codex-turn-workspace-write.jsonl"
	p9n         = "../../doc/research/probes/p9n-codex-network.jsonl"
)

// C1: a message cut short gives its phase no exact count, in P303a and in P2;
// the 0 the CLI sends with it is kept apart as reported. The phases before it
// keep the counts their messages reported.
func TestCorrectionC1CutMessage(t *testing.T) {
	for _, c := range []struct {
		name       string
		start, end int64
		estimate   int64
		stage      int
		before     []int64 // the exact counts of the phases before it
	}{
		{p303a, 9309, 9484, 50, 3, []int64{178}},
		{p2, 12033, 12265, 0, 3, nil},
	} {
		s, _ := replayState(t, c.name, claude.ProviderID, claude.NewMapper())
		th := thoughts(s)
		if len(th) != len(c.before)+1 {
			t.Fatalf("%s: %d phases, want %d", c.name, len(th), len(c.before)+1)
		}
		for i, n := range c.before {
			if x := th[i].Thought; !x.HasTokens || x.Tokens != n || x.Cut {
				t.Errorf("%s: phase %d: %+v, want the exact count %d", c.name, i+1, x, n)
			}
		}
		e := th[len(th)-1]
		x := e.Thought
		if since(x.Start) != c.start || since(x.End) != c.end || x.Outcome != "interrupted" || x.Estimate != c.estimate || e.Stage != c.stage {
			t.Errorf("%s: the cut phase %d..%d ms, %q, estimate %d, stage %d; want %d..%d, interrupted, %d, %d", c.name, since(x.Start), since(x.End), x.Outcome, x.Estimate, e.Stage, c.start, c.end, c.estimate, c.stage)
		}
		// No exact count: nothing that reads one can show the 0, and the CLI's
		// value is kept, as reported, apart.
		if x.HasTokens || x.Tokens != 0 || !x.TokensAt.IsZero() || x.Source != "" || x.Shared != 0 || !x.Cut || x.CutTokens != 0 {
			t.Errorf("%s: the cut phase: %+v, want no exact count and Cut with the CLI's 0", c.name, x)
		}
	}
}

// C2: a phase whose turn or session ended with no exact count takes none later.
// Codex replays a resumed thread's last usage (documented for 0.145.0, present
// in 0.163.0's source, never recorded), and it used to land on the crashed
// step: P12x-sandbox cut at line 61 by a crash, line 57 replayed after the
// resume, read "up to 4.1s · 10 reasoning tokens" where the step's own count is
// 16; P9n cut at line 25, line 21 replayed, "0 reasoning tokens" for 25.
func TestCorrectionC2ResumeReplay(t *testing.T) {
	for _, c := range []struct {
		name        string
		path        string
		cut, replay int
		reported    float64 // what the replayed update reports: the previous step's count
	}{
		{"P12x-sandbox", p12xSandbox, 61, 57, 10},
		{"P9n", p9n, 25, 21, 0},
	} {
		t.Run(c.name, func(t *testing.T) {
			s, replayed := crashAndResume(t, c.path, c.cut, c.replay)
			if len(replayed) != 1 || replayed[0] != c.reported {
				t.Fatalf("the replayed update gave the reducer %v as reasoning.last, want [%v]", replayed, c.reported)
			}
			th := thoughts(s)
			last := th[len(th)-1].Thought
			if !last.Confirmed || !last.Inferred || last.Live() {
				t.Fatalf("not the scenario: the last phase must be a confirmed, ended Codex step: %+v", last)
			}
			if last.HasTokens || last.Tokens != 0 || last.Counted || last.Shared != 0 || !last.Closed {
				t.Errorf("the crashed step took the replay: %+v, want no count, closed", last)
			}
			// The resumed session's next turn counts its own phase, and not the
			// crashed one.
			s.Do(state.Send{Text: "next"})
			b := &bench{t: t, s: s, seq: 900000}
			b.send(70000, event.UserMessage, "", event.User{Text: "next"})
			b.send(70000, event.TurnStarted, "", nil)
			b.step(70100)
			b.reasoningStart(70200, "", "rs-next")
			b.reasoningEnd(70201, "", "rs-next")
			b.usage(70300, "", 33)
			th = thoughts(s)
			if x := th[len(th)-1].Thought; !x.HasTokens || x.Tokens != 33 || x.Shared != 0 {
				t.Errorf("the next turn's phase: %+v, want 33 reasoning tokens", x)
			}
			if x := th[len(th)-2].Thought; x.HasTokens || !x.Closed {
				t.Errorf("the crashed step took the next turn's count: %+v", x)
			}
		})
	}
}

// crashAndResume replays the first cut lines of a Codex recording, crashes the
// session (the process exits with code 1), resumes it as Ctrl+R does (a new
// mapper, events numbered from 1 again, thread/resume) and has Codex replay the
// usage update of line replay of the recording, byte for byte. It returns the
// state and the reasoning.last values the replay gave the reducer.
func crashAndResume(t *testing.T, path string, cut, replay int) (*state.State, []float64) {
	t.Helper()
	recs, err := fixture.Read(path)
	if err != nil {
		t.Fatal(err)
	}
	line := func(n int) fixture.Record {
		if n < 1 || n > len(recs) {
			t.Fatalf("%s: line %d of %d", filepath.Base(path), n, len(recs))
		}
		return recs[n-1]
	}
	s := state.New()
	m1 := codex.NewMapper()
	for _, ev := range fixture.Flatten(fixture.Replay(codex.ProviderID, recs[:cut], m1)) {
		s.Apply(ev)
	}
	crash := []fixture.Record{{TMs: line(cut).TMs + 50, Dir: "exit", Msg: json.RawMessage(`{"code":1}`)}}
	for _, ev := range fixture.Flatten(fixture.Replay(codex.ProviderID, crash, m1)) {
		ev.Seq += 100000
		s.Apply(ev)
	}
	if s.Session.Phase != state.Crashed {
		t.Fatalf("session %q after the exit, want Crashed", s.Session.Phase)
	}
	if th := thoughts(s); len(th) == 0 || th[len(th)-1].Thought.HasTokens || th[len(th)-1].Thought.Live() {
		t.Fatalf("not the scenario: the crashed step must be ended and uncounted: %+v", th)
	}
	if cmds := s.Do(state.Resume{}); len(cmds) != 1 || cmds[0].Kind != state.CmdStartSession {
		t.Fatalf("Ctrl+R: %+v", cmds)
	}
	// The new process: its handshake, thread/resume with the thread's id, the
	// recorded thread response, then the replayed usage (the recording's own
	// line).
	var thread string
	var threadResp json.RawMessage
	var resp struct {
		ID int `json:"id"`
	}
	for _, r := range recs {
		var m struct {
			ID     int `json:"id"`
			Result struct {
				Thread struct {
					ID string `json:"id"`
				} `json:"thread"`
			} `json:"result"`
		}
		if r.Dir == "out" && json.Unmarshal(r.Msg, &m) == nil && m.Result.Thread.ID != "" {
			thread, threadResp, resp.ID = m.Result.Thread.ID, r.Msg, m.ID
			break
		}
	}
	if thread == "" {
		t.Fatalf("%s: no thread response", filepath.Base(path))
	}
	base := line(cut).TMs + 5000
	var resume []fixture.Record
	add := func(dir string, msg json.RawMessage) {
		resume = append(resume, fixture.Record{TMs: base + int64(len(resume))*10, Dir: dir, Msg: msg})
	}
	add("argv", json.RawMessage(`["app-server"]`))
	add(line(2).Dir, line(2).Msg) // initialize
	add(line(3).Dir, line(3).Msg) // its response
	add(line(4).Dir, line(4).Msg) // initialized
	add("in", json.RawMessage(fmt.Sprintf(`{"id":%d,"method":"thread/resume","params":{"threadId":%q}}`, resp.ID, thread)))
	add("out", threadResp)
	if !strings.Contains(string(line(replay).Msg), `"thread/tokenUsage/updated"`) {
		t.Fatalf("%s: line %d is not a usage update", filepath.Base(path), replay)
	}
	add("out", line(replay).Msg)
	var got []float64
	for _, ev := range fixture.Flatten(fixture.Replay(codex.ProviderID, resume, codex.NewMapper())) {
		if ms, ok := ev.Payload.(event.Metrics); ok && ev.Agent == "" {
			for _, it := range ms.Items {
				if it.Name == event.ReasoningLast {
					got = append(got, it.Value)
				}
			}
		}
		s.Apply(ev)
	}
	return s, got
}

// C2: a turn with no prompt entry (one the CLI starts itself) never shares an
// earlier turn's phase: the count of its own phase is its own, not "over 2
// phases".
func TestCorrectionC2TurnWithoutPrompt(t *testing.T) {
	b := newBench(t)
	b.step(100)
	b.reasoningStart(200, "", "rs1")
	b.reasoningEnd(201, "", "rs1")
	b.turnEnd(300, "interrupted") // the first turn's count never came
	b.send(1000, event.TurnStarted, "", nil)
	if b.s.Turn.Phase != state.Running {
		t.Fatalf("the CLI's own turn: %q, want Running", b.s.Turn.Phase)
	}
	b.step(1100)
	b.reasoningStart(1200, "", "rs2")
	b.reasoningEnd(1201, "", "rs2")
	b.usage(1300, "", 40)
	th := b.all()
	if len(th) != 2 {
		t.Fatalf("%d phases, want 2", len(th))
	}
	if x := th[0].Thought; x.HasTokens || x.Shared != 0 || !x.Closed {
		t.Errorf("the first turn's phase took a count: %+v", x)
	}
	if x := th[1].Thought; !x.HasTokens || x.Tokens != 40 || x.Shared != 0 {
		t.Errorf("the CLI's own turn's phase: %+v, want 40 reasoning tokens, not shared", x)
	}
}

// C2: every phase of an ended turn either has its exact count or is closed,
// whichever way the turn ended, in every recording of both CLIs; a phase with a
// count is never closed; a phase of a turn still open is not.
func TestCorrectionC2RecordedPhasesClosed(t *testing.T) {
	extra := []struct {
		name     string
		provider string
		mapper   func() fixture.Mapper
	}{
		{p9a, codex.ProviderID, func() fixture.Mapper { return codex.NewMapper() }},
		{"p12x-sandbox-codex-turn-workspace-write.jsonl", codex.ProviderID, func() fixture.Mapper { return codex.NewMapper() }},
	}
	type rec = struct {
		name     string
		provider string
		mapper   func() fixture.Mapper
	}
	all := append([]rec(nil), extra...)
	for _, r := range recordings {
		all = append(all, r)
	}
	closed, counted := 0, 0
	for _, r := range all {
		s, _ := replayState(t, r.name, r.provider, r.mapper())
		for i, e := range thoughts(s) {
			x := e.Thought
			over := e.Stage > 0 && !s.Stages[e.Stage-1].Open
			switch {
			case x.HasTokens && x.Closed:
				t.Errorf("%s: phase %d has its count and is closed: %+v", r.name, i+1, x)
			case x.HasTokens:
				counted++
			case over && !x.Closed:
				t.Errorf("%s: phase %d of an ended turn has no count and is not closed: %+v", r.name, i+1, x)
			case !over && x.Closed:
				t.Errorf("%s: phase %d of an open turn is closed: %+v", r.name, i+1, x)
			case x.Closed:
				closed++
			}
		}
	}
	if closed < 10 || counted < 40 {
		t.Errorf("%d closed phases and %d counted in the recordings: the check saw too little", closed, counted)
	}
}

// C2: a count that arrives after its phase's turn ended is not applied to it,
// for Claude Code's message_delta (a cut message's value included) and for
// Codex's usage update; the session's crash closes a phase the same way.
func TestCorrectionC2LateCount(t *testing.T) {
	t.Run("a message_delta after the turn", func(t *testing.T) {
		b := newBench(t)
		b.blockStart(1000, "m1", 0)
		b.blockStop(1500, "m1", 0)
		b.turnEnd(2000, "completed")
		b.blockCount(2500, "m1", "end_turn", 40)
		if x := b.one().Thought; x.HasTokens || x.Tokens != 0 || !x.Closed {
			t.Errorf("a late count: %+v, want none, closed", x)
		}
	})
	t.Run("a message cut short after the turn", func(t *testing.T) {
		b := newBench(t)
		b.blockStart(1000, "m1", 0)
		b.blockStop(1500, "m1", 0)
		b.turnEnd(2000, "completed")
		b.blockCount(2500, "m1", "", 0)
		if x := b.one().Thought; x.HasTokens || x.Cut || x.Outcome != "" || !x.Closed {
			t.Errorf("a late cut message: %+v, want it left as the turn ended it", x)
		}
	})
	t.Run("a usage update after the turn", func(t *testing.T) {
		b := newBench(t)
		b.step(100)
		b.reasoningStart(200, "", "rs1")
		b.reasoningEnd(201, "", "rs1")
		b.turnEnd(300, "completed")
		b.usage(400, "", 9)
		if x := b.one().Thought; x.HasTokens || x.Tokens != 0 || x.Counted || !x.Closed {
			t.Errorf("a late usage: %+v, want none, closed", x)
		}
	})
	t.Run("a crash", func(t *testing.T) {
		b := newBench(t)
		b.blockStart(1000, "m1", 0)
		b.blockStop(1500, "m1", 0)
		b.send(2000, event.SessionEnded, "", event.Ended{Code: 1, HasCode: true})
		b.blockCount(2500, "m1", "tool_use", 40)
		if x := b.one().Thought; x.HasTokens || !x.Closed {
			t.Errorf("a count after the crash: %+v, want none, closed", x)
		}
	})
	t.Run("a phase with its count is not closed", func(t *testing.T) {
		b := newBench(t)
		b.blockStart(1000, "m1", 0)
		b.blockStop(1500, "m1", 0)
		b.blockCount(1600, "m1", "end_turn", 40)
		b.turnEnd(2000, "completed")
		if x := b.one().Thought; !x.HasTokens || x.Tokens != 40 || x.Closed {
			t.Errorf("a counted phase: %+v, want 40, not closed", x)
		}
	})
}

// C3: a prompt sent while the turn is Idle ends a phase still live, at the last
// event's time, its end inferred: a phase that began outside any turn, which no
// turn end would ever close, no longer stays live for good (Claude Code's block
// and Codex's reasoning item alike). A prompt with no such phase changes none.
func TestCorrectionC3PromptEndsStrayPhase(t *testing.T) {
	t.Run("a thinking block", func(t *testing.T) {
		b := newBench(t)
		b.blockStart(1000, "m1", 0)
		b.blockStop(1100, "m1", 0)
		b.blockCount(1200, "m1", "end_turn", 30)
		b.turnEnd(2000, "completed")
		b.blockStart(2100, "m2", 0) // outside any turn
		if th := b.all(); len(th) != 2 || !th[1].Thought.Live() || !th[1].Streaming {
			t.Fatalf("not the scenario: %+v", th)
		}
		before := *b.all()[0].Thought
		if cmds := b.s.Do(state.Send{Text: "again"}); len(cmds) != 1 || cmds[0].Kind != state.CmdWrite {
			t.Fatalf("the prompt: %+v", cmds)
		}
		th := b.all()
		x := th[1].Thought
		if x.Live() || !x.End.Equal(at(2100)) || !x.EndInferred || th[1].Streaming || x.Outcome != "" || !x.Closed || x.HasTokens {
			t.Errorf("the stray phase after the prompt: %+v (streaming %v), want it ended at 2,100 ms, its end inferred", x, th[1].Streaming)
		}
		if after := *th[0].Thought; !reflect.DeepEqual(after, before) {
			t.Errorf("the prompt changed a phase of the earlier turn:\n%+v\n%+v", after, before)
		}
	})
	t.Run("a reasoning item", func(t *testing.T) {
		b := newBench(t)
		b.turnEnd(2000, "completed")
		b.reasoningStart(2100, "", "rs-stray")
		if x := b.one().Thought; !x.Live() {
			t.Fatalf("not the scenario: %+v", x)
		}
		b.s.Do(state.Send{Text: "again"})
		x := b.one().Thought
		if x.Live() || !x.End.Equal(at(2100)) || !x.EndInferred || b.one().Streaming {
			t.Errorf("the stray item after the prompt: %+v", x)
		}
	})
	t.Run("a prompt with no stray phase", func(t *testing.T) {
		b := newBench(t)
		b.step(100)
		b.reasoningStart(200, "", "rs1")
		b.reasoningEnd(201, "", "rs1")
		b.usage(300, "", 12)
		b.turnEnd(400, "completed")
		before := *b.one().Thought
		b.s.Do(state.Send{Text: "again"})
		if after := *b.one().Thought; !reflect.DeepEqual(after, before) {
			t.Errorf("the prompt changed an ended phase:\n%+v\n%+v", after, before)
		}
	})
}

// liveCount is how many phases of the record run.
func liveCount(s *state.State) int {
	n := 0
	for _, e := range thoughts(s) {
		if e.Thought.Live() {
			n++
		}
	}
	return n
}

// C4: a Codex step's start ends a reasoning item still live, its end inferred
// at that start: never two live phases.
func TestCorrectionC4NextStepEndsItem(t *testing.T) {
	b := newBench(t)
	b.step(100)
	b.reasoningStart(200, "", "rs1") // its completion never comes
	if x := b.one().Thought; !x.Live() || !x.Confirmed {
		t.Fatalf("not the scenario: %+v", x)
	}
	b.step(500)
	th := b.all()
	if len(th) != 2 {
		t.Fatalf("%d phases after the next step, want 2", len(th))
	}
	x, y := th[0].Thought, th[1].Thought
	if x.Live() || !x.End.Equal(at(500)) || !x.EndInferred || th[0].Streaming || !x.Confirmed || x.Dropped || !x.Start.Equal(at(100)) {
		t.Errorf("the item still live at the next step: %+v (streaming %v), want it ended at 500 ms, its end inferred", x, th[0].Streaming)
	}
	// Its end is the next step's start, and says so: not its turn's.
	if !x.EndNext {
		t.Errorf("the item ended by the next step: %+v, want EndNext", x)
	}
	if !y.Live() || !y.Inferred || y.Confirmed || !th[1].Streaming {
		t.Errorf("the next step: %+v", y)
	}
	if n := liveCount(b.s); n != 1 {
		t.Errorf("%d phases live, want only the new step", n)
	}
	// Its own completion, late, changes nothing.
	b.reasoningEnd(600, "", "rs1")
	if x := b.all()[0].Thought; !x.End.Equal(at(500)) {
		t.Errorf("the late completion moved the end to %v", x.End)
	}
	// A step that is already open keeps the item beside it only until the item's
	// own end; a reasoning item that completed is not ended again.
	c := newBench(t)
	c.step(100)
	c.reasoningStart(200, "", "rs1")
	c.reasoningEnd(201, "", "rs1")
	c.step(500)
	if x := c.all()[0].Thought; !x.End.Equal(at(201)) || x.EndInferred || x.EndNext {
		t.Errorf("a completed item: %+v, want its own end", x)
	}
	// An item the turn's end cut is ended by the turn, not by a next step.
	d := newBench(t)
	d.step(100)
	d.reasoningStart(200, "", "rs1")
	d.turnEnd(500, "interrupted")
	if x := d.one().Thought; !x.EndInferred || x.EndNext || !x.End.Equal(at(500)) {
		t.Errorf("an item cut by its turn's end: %+v, want an inferred end that is not EndNext", x)
	}
}

// C5: a reasoning item that finds no step open (the step before it made no
// call, so the adapter began none) starts at its turn's last usage update, so
// its time is a bound like every other Codex phase's: P9a's second phase, which
// read "Thought · 118 reasoning tokens" with no duration, and P6b's second.
func TestCorrectionC5StartAtLastUsage(t *testing.T) {
	s, _ := replayState(t, p9a, codex.ProviderID, codex.NewMapper())
	th := thoughts(s)
	if len(th) != 4 {
		t.Fatalf("P9a: %d phases, want 4", len(th))
	}
	x := th[1].Thought
	if since(x.Start) != 4252 || since(x.End) != 5694 || !x.Confirmed || !x.Inferred || !x.HasTokens || x.Tokens != 118 || th[1].Origin != event.Observed {
		t.Errorf("P9a's second phase: %d..%d ms, %+v; want 4,252..5,694 ms and 118 reasoning tokens", since(x.Start), since(x.End), x)
	}
	// 4,252 ms is the first response's usage update (38 reasoning tokens): the
	// request that followed it at once.
	if v := s.Metrics[event.ReasoningLast]; !v.Present {
		t.Fatalf("no reasoning.last: %+v", v)
	}
	// No recorded Codex phase has an unknown start any more.
	for _, r := range recordings {
		if r.provider != codex.ProviderID {
			continue
		}
		rs, _ := replayState(t, r.name, r.provider, r.mapper())
		for i, e := range thoughts(rs) {
			if y := e.Thought; y.Confirmed && y.Start.IsZero() {
				t.Errorf("%s: phase %d has no start: %+v", r.name, i+1, y)
			}
		}
	}
	for i, e := range thoughts(s) {
		if y := e.Thought; y.Inferred && !y.Dropped && y.Start.IsZero() {
			t.Errorf("P9a: phase %d has no start", i+1)
		}
	}

	// The rule, on its own: the start is the turn's last usage update; none
	// before one; none from an earlier turn's; never a child's.
	b := newBench(t)
	b.reasoningStart(100, "", "rs0")
	if x := b.one().Thought; !x.Start.IsZero() {
		t.Errorf("no usage yet: start %v, want none", x.Start)
	}
	b.reasoningEnd(101, "", "rs0")
	b.usage(200, "child", 99) // a child's: not the orchestrator's response
	b.reasoningStart(300, "", "rs1")
	if x := b.all()[1].Thought; !x.Start.IsZero() {
		t.Errorf("after a child's usage: start %v, want none", x.Start)
	}
	b.reasoningEnd(301, "", "rs1")
	b.usage(400, "", 12)
	b.reasoningStart(500, "", "rs2")
	if x := b.all()[2].Thought; !x.Start.Equal(at(400)) || !x.Inferred || !x.Confirmed {
		t.Errorf("after the turn's usage at 400 ms: %+v, want it to start there", x)
	}
	b.reasoningEnd(501, "", "rs2")
	b.turnEnd(600, "completed")
	b.s.Do(state.Send{Text: "again"})
	b.send(700, event.UserMessage, "", event.User{Text: "again"})
	b.send(700, event.TurnStarted, "", nil)
	b.reasoningStart(800, "", "rs3")
	if x := b.all()[3].Thought; !x.Start.IsZero() {
		t.Errorf("an earlier turn's usage: start %v, want none", x.Start)
	}
	b.reasoningEnd(801, "", "rs3")
	b.usage(900, "", 5)
	b.reasoningStart(1000, "", "rs4")
	if x := b.all()[4].Thought; !x.Start.Equal(at(900)) {
		t.Errorf("after this turn's usage: start %v, want 900 ms", x.Start)
	}
}
