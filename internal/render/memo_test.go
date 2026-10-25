package render

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"reflect"
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

// Plan 49's criteria 49.4 to 49.6 (designs D4 and D5; audit finding 7): the
// memo changes no frame, reads an input once, and takes in only what was
// added.

// twin is a UI drawn with the memo and the same UI drawn without, each
// with its own wrap cache, as the application holds them across frames.
type twin struct{ with, without UI }

func newTwin(w, h int) twin {
	return twin{
		with:    UI{Width: w, Height: h, Focus: "B", Wrap: NewWrapCache(), Memo: NewMemo()},
		without: UI{Width: w, Height: h, Focus: "B", Wrap: NewWrapCache()},
	}
}

// agree draws s both ways at now, then, with unfolded, again with every
// stage unfolded and the first agent inspected, and fails on a frame that
// differs.
func (tw twin) agree(t *testing.T, where string, s *state.State, now time.Time, unfolded bool) {
	t.Helper()
	tw.with.Now, tw.without.Now = now, now
	for _, unfold := range []bool{false, true} {
		if unfold && !unfolded {
			break
		}
		a, b := tw.with, tw.without
		if unfold {
			fold := map[string]bool{}
			for _, st := range s.Stages {
				fold[fmt.Sprintf("s:%d", st.N)] = false
			}
			a.Fold, b.Fold = fold, fold
			if len(s.Order) > 0 {
				a.Inspect, b.Inspect = s.Order[0], s.Order[0]
			}
		}
		if x, y := Draw(s, a).String(), Draw(s, b).String(); x != y {
			t.Fatalf("%s, %dx%d, unfolded %v: the memo's frame differs:\n%s\n--- without it ---\n%s", where, a.Width, a.Height, unfold, x, y)
		}
	}
}

// TestMemoChangesNoFrame is criterion 49.5: every mock file and every
// probe recording, replayed event by event, drawn after each with the memo
// and without it: the same frames. Render draws on one goroutine, so the
// files run one after the other: every event drawn at 200x60, and every
// third and the last, unfolded with an agent inspected, and at 80x24.
func TestMemoChangesNoFrame(t *testing.T) {
	mocks, _ := filepath.Glob(filepath.Join("..", "..", "testdata", "mock", "*.jsonl"))
	probes, _ := filepath.Glob(filepath.Join("..", "..", "testdata", "probes", "p*.jsonl"))
	if len(mocks) == 0 || len(probes) == 0 {
		t.Fatalf("%d mock files, %d recordings", len(mocks), len(probes))
	}
	replay := func(t *testing.T, name string, s *state.State, evs []event.Event) {
		wide, narrow := newTwin(200, 60), newTwin(80, 24)
		s.Screen = state.ScreenDashboard
		for i, e := range evs {
			s.Apply(e)
			where, now, all := fmt.Sprintf("%s, event %d", name, i), e.At.Add(time.Second), i%3 == 0 || i == len(evs)-1
			wide.agree(t, where, s, now, all)
			if all {
				narrow.agree(t, where, s, now, false)
			}
		}
	}
	for _, f := range mocks {
		t.Run(filepath.Base(f), func(t *testing.T) {
			timed, err := mock.Load(f)
			if err != nil {
				t.Fatal(err)
			}
			evs := make([]event.Event, len(timed))
			for i, e := range timed {
				evs[i] = e.Event
			}
			s := state.New()
			s.Context = state.Context{Provider: state.MockProvider, ProviderName: "Mock data", Project: f, Mock: true}
			replay(t, filepath.Base(f), s, evs)
		})
	}
	for _, f := range probes {
		t.Run(filepath.Base(f), func(t *testing.T) {
			recs, err := fixture.Read(f)
			if err != nil {
				t.Fatal(err)
			}
			s := state.New()
			var per [][]event.Event
			if strings.Contains(filepath.Base(f), "codex") {
				per = fixture.Replay(codex.ProviderID, recs, codex.NewMapper())
				s.Context = state.Context{Provider: codex.ProviderID, ProviderName: "OpenAI Codex", Project: `C:\Users\amolimom\probe`}
			} else {
				per = fixture.Replay(claude.ProviderID, recs, claude.NewMapper())
				s.Context = state.Context{Provider: claude.ProviderID, ProviderName: "Claude Code", Project: `C:\Users\amolimom\probe`}
			}
			replay(t, filepath.Base(f), s, fixture.Flatten(per))
		})
	}
}

// longState is a session of n turns, each with an orchestrator's calls and
// three agents with calls of their own, as TestLongSession builds one.
func longState(n int) *state.State {
	s := state.New()
	s.Context = state.Context{Provider: claude.ProviderID, ProviderName: "Claude Code", Project: `C:\Users\amolimom\proj`}
	s.Screen = state.ScreenDashboard
	at := time.Date(2026, 10, 25, 9, 0, 0, 0, time.UTC)
	seq := uint64(0)
	apply := func(k event.Kind, agent string, p any) {
		seq++
		at = at.Add(100 * time.Millisecond)
		s.Apply(event.Event{Seq: seq, At: at, Provider: claude.ProviderID, Kind: k, Origin: event.Observed, Agent: agent, Payload: p})
	}
	apply(event.SessionSpawned, "", event.Spawned{})
	apply(event.SessionReady, "", event.Ready{})
	in := json.RawMessage(`{"file_path":"internal/render/telemetry.go"}`)
	for turn := range n {
		apply(event.UserMessage, "", event.User{Text: fmt.Sprintf("Turn %d: look at the telemetry code.", turn)})
		apply(event.TurnStarted, "", nil)
		for j := range 3 {
			key := fmt.Sprintf("a%d-%d", turn, j)
			apply(event.AgentStarted, "", event.Agent{Key: key, Label: "explorer", Type: "subagent"})
			for k := range 3 {
				id := fmt.Sprintf("t%d-%d-%d", turn, j, k)
				apply(event.ToolStarted, key, event.Tool{ToolID: id, Name: "Grep", Input: in})
				apply(event.ToolCompleted, key, event.Tool{ToolID: id, Name: "Grep", Output: "1 match"})
			}
			apply(event.AgentCompleted, "", event.Agent{Key: key, Status: "completed"})
		}
		for k := range 20 {
			id := fmt.Sprintf("o%d-%d", turn, k)
			apply(event.ToolStarted, "", event.Tool{ToolID: id, Name: "Read", Input: in})
			apply(event.ToolCompleted, "", event.Tool{ToolID: id, Name: "Read", Output: "120 lines"})
		}
		apply(event.TurnCompleted, "", event.TurnEnd{Outcome: "completed"})
	}
	return s
}

// TestMemoInputs is criterion 49.4: a frame of 30 Write calls with inputs
// of 100 KB, drawn again with nothing changed, reads no input; a call's new
// input and its new status show in the next frame; an input no frame uses
// is no longer held two frames later.
func TestMemoInputs(t *testing.T) {
	s := longState(0)
	at := time.Date(2026, 10, 25, 10, 0, 0, 0, time.UTC)
	s.Apply(event.Event{Seq: 100, At: at, Provider: claude.ProviderID, Kind: event.TurnStarted, Origin: event.Observed})
	content := strings.Repeat("a line of the file\n", 100_000/19)
	for k := range 30 {
		b, _ := json.Marshal(map[string]string{"file_path": fmt.Sprintf("gen/file%02d.go", k), "content": content})
		s.Apply(event.Event{Seq: uint64(101 + k), At: at, Provider: claude.ProviderID, Kind: event.ToolStarted, Origin: event.Observed,
			Payload: event.Tool{ToolID: fmt.Sprintf("w%d", k), Name: "Write", Input: b}})
	}
	m := NewMemo()
	ui := UI{Width: 200, Height: 60, Now: at, Focus: "B", Wrap: NewWrapCache(), Memo: m}
	Draw(s, ui)
	read := m.misses
	if read < 30 {
		t.Fatalf("the first frame read %d inputs, want the 30 calls' at least", read)
	}
	Draw(s, ui)
	if m.misses != read {
		t.Fatalf("drawn again with nothing changed, the memo read %d more inputs", m.misses-read)
	}
	tc := s.Tools["w29"]
	old := tc.Input
	tc.Input = json.RawMessage(`{"file_path":"gen/renamed.go","content":"x"}`)
	tc.Status = state.ToolFailed
	f := plainFrame(Draw(s, ui))
	if !strings.Contains(f, "renamed.go") || !strings.Contains(f, "✗ Write") {
		t.Fatalf("the call's new input or status not shown:\n%s", f)
	}
	Draw(s, ui)
	k := inputKey{&old[0], len(old), false}
	if _, ok := m.cur[k]; ok {
		t.Error("the replaced input is still held two frames later")
	} else if _, ok := m.old[k]; ok {
		t.Error("the replaced input is still held two frames later")
	}
}

// plainFrame is the frame's text without its escape codes.
func plainFrame(sc *Screen) string {
	var b strings.Builder
	for _, l := range sc.lines {
		for _, x := range l {
			b.WriteString(x.text)
		}
		b.WriteString("\n")
	}
	return b.String()
}

// TestMemoIndex is criterion 49.6, the index: once built, a call added is
// taken in by visiting it alone, and the index always equals indexItems,
// agentTools and children; an agent re-parented builds it again.
func TestMemoIndex(t *testing.T) {
	s := longState(40)
	m := NewMemo()
	same := func(when string) {
		t.Helper()
		ui := UI{Memo: m}
		if got, want := stageIndex(s, ui), indexItems(s); !reflect.DeepEqual(got, want) {
			t.Fatalf("%s: the memo's items differ from indexItems", when)
		}
		for _, k := range s.Order {
			if got, want := callsOf(s, m, k), agentTools(s, k); !reflect.DeepEqual(got, want) {
				t.Fatalf("%s: agent %s's calls differ", when, k)
			}
			if got, want := kidsOf(s, m, k), children(s, k); !reflect.DeepEqual(got, want) {
				t.Fatalf("%s: agent %s's children differ", when, k)
			}
		}
	}
	same("built")
	built := m.items.visits
	at := time.Date(2026, 10, 26, 9, 0, 0, 0, time.UTC)
	s.Apply(event.Event{Seq: 1_000_000, At: at, Provider: claude.ProviderID, Kind: event.ToolStarted, Origin: event.Observed,
		Payload: event.Tool{ToolID: "late", Name: "Read", Input: json.RawMessage(`{}`)}})
	same("a call added")
	if v := m.items.visits - built; v != 1 {
		t.Errorf("a call added visited %d calls, want 1", v)
	}
	// A child before its parent, then the parent: a top item, then not.
	s.Apply(event.Event{Seq: 1_000_001, At: at, Provider: claude.ProviderID, Kind: event.AgentStarted, Origin: event.Observed,
		Payload: event.Agent{Key: "child", Parent: "parent", Label: "child", Type: "thread"}})
	same("a child before its parent")
	s.Apply(event.Event{Seq: 1_000_002, At: at, Provider: claude.ProviderID, Kind: event.AgentStarted, Origin: event.Observed,
		Payload: event.Agent{Key: "parent", Label: "parent", Type: "thread"}})
	same("its parent")
	before := m.items.visits
	s.Apply(event.Event{Seq: 1_000_003, At: at, Provider: claude.ProviderID, Kind: event.AgentProgress, Origin: event.Observed,
		Payload: event.Agent{Key: "child", Parent: s.Order[0]}})
	same("the child re-parented")
	if m.items.visits-before < len(s.ToolOrder) {
		t.Errorf("a re-parented agent visited %d calls, want the index built again", m.items.visits-before)
	}
}

// TestMemoThinking is criterion 49.6, the thinking scans: Animating and
// ActivityOn look at the thinking entries only.
func TestMemoThinking(t *testing.T) {
	s := longState(30)
	at := time.Date(2026, 10, 26, 9, 0, 0, 0, time.UTC)
	for i := range 5 {
		s.Conversation = append(s.Conversation, state.Entry{Role: "thinking", At: at, Thought: &state.Thought{ID: i + 1, Start: at, End: at}})
	}
	m := NewMemo()
	ui := UI{Width: 200, Height: 60, Now: at.Add(time.Minute), Memo: m}
	Animating(s, ui)
	ActivityOn(s, ui)
	if m.thinking.visits > 10 {
		t.Errorf("Animating and ActivityOn looked at %d entries of %d, want the 5 thinking ones at most twice", m.thinking.visits, len(s.Conversation))
	}
	if got, want := thoughts(s, m), thoughts(s, nil); !reflect.DeepEqual(got, want) || len(got) != 5 {
		t.Errorf("the thinking entries %v, want %v", got, want)
	}
}

// TestTreeRowsLazy is criterion 49.6, zone A: the tree's rows make no
// line; zone A makes those of its window only.
func TestTreeRowsLazy(t *testing.T) {
	s := longState(240)
	ui := UI{Width: 200, Height: 60, Memo: NewMemo()}
	made, lazy := 0, 0
	for _, r := range treeRows(s, ui, 44) {
		switch {
		case r.kind != rowMade && r.l != nil:
			t.Fatalf("a row of kind %d made its line before it shows", r.kind)
		case r.kind == rowMade && r.l != nil:
			made++
		case r.kind != rowMade:
			lazy++
		}
	}
	// The blank rows between the turns are the only lines made: one less
	// than the turns.
	if made > 1 || lazy < 3*240 {
		t.Errorf("%d rows made and %d left to make, want none made and every turn's", made, lazy)
	}
	// Every turn unfolded: its calls and its agents are rows too, none made.
	ui.Fold = map[string]bool{}
	for i := range s.Stages {
		ui.Fold[fmt.Sprintf("s:%d", s.Stages[i].N)] = false
	}
	kinds := map[rowKind]int{}
	for _, r := range treeRows(s, ui, 44) {
		if r.kind != rowMade && r.l != nil {
			t.Fatalf("unfolded, a row of kind %d made its line before it shows", r.kind)
		}
		kinds[r.kind]++
	}
	if kinds[rowTool] < 20*240 || kinds[rowAgent] < 3*240 {
		t.Errorf("unfolded: %d call rows and %d agent rows, want every turn's", kinds[rowTool], kinds[rowAgent])
	}
}
