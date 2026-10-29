package state

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/Evengelius/agent-command-center/internal/event"
)

// reviewReady is a ready Claude Code session whose turns are reviewed
// (plan 65, design D2).
func reviewReady(t *testing.T) *State {
	t.Helper()
	s := New()
	s.Context = Context{Provider: "claude-code"}
	s.Startup.Phase = Starting
	reviewEvents(s, event.Event{Kind: event.SessionSpawned, Payload: event.Spawned{}}, event.Event{Kind: event.SessionReady, Payload: event.Ready{}})
	if s.Session.Phase != Ready {
		t.Fatalf("session %q", s.Session.Phase)
	}
	s.Do(ReviewStore{On: true})
	return s
}

// reviewEvents applies evs as the orchestrator's, observed, and returns the
// commands they gave.
func reviewEvents(s *State, evs ...event.Event) []Command {
	var cmds []Command
	for _, ev := range evs {
		ev.Seq, ev.At, ev.Provider, ev.Origin = uint64(len(s.Log)+1), time.Date(2026, 10, 29, 12, 0, len(s.Log), 0, time.UTC), "claude-code", event.Observed
		cmds = append(cmds, s.Apply(ev)...)
	}
	return cmds
}

// turnOpen is what a turn's start sends; turnClose its end with outcome.
var turnOpen = event.Event{Kind: event.MessageMarker, Payload: event.Marker{Name: "message_start"}}

func turnClose(outcome string) event.Event {
	return event.Event{Kind: event.TurnCompleted, Payload: event.TurnEnd{Outcome: outcome}}
}

// kinds lists the commands of the review among cmds, as "kind turn end".
func reviewCmds(cmds []Command) []string {
	var out []string
	for _, c := range cmds {
		switch c.Kind {
		case CmdTurnSnapshot:
			out = append(out, fmt.Sprintf("snapshot %d end=%v", c.Turn, c.End))
		case CmdTurnFiles:
			out = append(out, fmt.Sprintf("files %d %s..%s", c.Turn, c.From, c.To))
		}
	}
	return out
}

func TestReviewSend(t *testing.T) {
	// The owner's message: the start's snapshot first, then the write, in
	// that order, for the stage it opens (design D4).
	s := reviewReady(t)
	cmds := s.Do(Send{Text: "do it"})
	if len(cmds) != 2 || cmds[0].Kind != CmdTurnSnapshot || cmds[0].Turn != 1 || cmds[0].End || cmds[1].Kind != CmdWrite {
		t.Fatalf("send's commands: %+v", cmds)
	}
	if len(s.Stages) != 1 || !s.Stages[0].Review.Asked || s.Turn.ReviewAsked {
		t.Fatalf("stage %+v, turn asked %v", s.Stages, s.Turn.ReviewAsked)
	}
	// The editor's row says so until the start's snapshot answers.
	if s.Turn.Notice != "the turn's snapshot…" {
		t.Fatalf("notice %q", s.Turn.Notice)
	}
	if s.Do(TurnSnapshot{Turn: 1, Tree: "t1"}); s.Turn.Notice != "" {
		t.Fatalf("notice %q after the snapshot", s.Turn.Notice)
	}
	// Without the store, the write alone.
	off := reviewReady(t)
	off.Do(ReviewStore{})
	if cmds := off.Do(Send{Text: "do it"}); len(cmds) != 1 || cmds[0].Kind != CmdWrite {
		t.Fatalf("send without the review: %+v", cmds)
	}
}

func TestReviewTurn(t *testing.T) {
	for _, outcome := range []string{"completed", "failed", "interrupted"} {
		t.Run(outcome, func(t *testing.T) {
			s := reviewReady(t)
			s.Do(Send{Text: "do it"})
			if got := reviewCmds(s.Do(TurnSnapshot{Turn: 1, Tree: "t1"})); got != nil {
				t.Fatalf("the start alone asked %v", got)
			}
			reviewEvents(s, turnOpen)
			got := reviewCmds(reviewEvents(s, turnClose(outcome)))
			if want := []string{"snapshot 1 end=true"}; !reflect.DeepEqual(got, want) {
				t.Fatalf("turn end asked %v, want %v", got, want)
			}
			if s.Stages[0].Open {
				t.Fatal("the stage is still open")
			}
			got = reviewCmds(s.Do(TurnSnapshot{Turn: 1, End: true, Tree: "t2"}))
			if want := []string{"files 1 t1..t2"}; !reflect.DeepEqual(got, want) {
				t.Fatalf("the end asked %v, want %v", got, want)
			}
			s.Do(TurnFiles{Turn: 1, Files: []ReviewFile{{Status: 'M', Path: "a.go", Added: 2, Removed: 1}}, Total: 1})
			if r := s.Stages[0].Review; !r.Done || !r.Shown() || r.Total != 1 || r.Problem != "" {
				t.Fatalf("review %+v", r)
			}
		})
	}
}

func TestReviewNothingChanged(t *testing.T) {
	// Equal trees changed nothing: no files read, no row; a file left out
	// whose size changed is still shown, without its content (design D6).
	s := reviewReady(t)
	s.Do(Send{Text: "do it"})
	big := ReviewLeft{Path: "big.bin", Why: "over 32.0 MiB", Size: 40 << 20, ModTime: time.Unix(100, 0)}
	s.Do(TurnSnapshot{Turn: 1, Tree: "t1", Left: []ReviewLeft{big}})
	reviewEvents(s, turnOpen, turnClose("completed"))
	if got := reviewCmds(s.Do(TurnSnapshot{Turn: 1, End: true, Tree: "t1", Left: []ReviewLeft{big}})); got != nil {
		t.Fatalf("equal trees asked %v", got)
	}
	if r := s.Stages[0].Review; !r.Done || r.Shown() {
		t.Fatalf("an unchanged turn's review %+v", r)
	}
	s.Do(Send{Text: "again"})
	s.Do(TurnSnapshot{Turn: 2, Tree: "t1", Left: []ReviewLeft{big}})
	reviewEvents(s, turnOpen, turnClose("completed"))
	grown := big
	grown.Size++
	s.Do(TurnSnapshot{Turn: 2, End: true, Tree: "t1", Left: []ReviewLeft{grown, {Path: "new.bin", Why: "over 32.0 MiB"}}})
	if r := s.Stages[1].Review; !r.Shown() || len(r.Left) != 2 || r.Total != 0 {
		t.Fatalf("a turn whose left-out files changed: %+v", r)
	}
}

func TestReviewProblems(t *testing.T) {
	// A start that failed: the turn has no review, said, and no end is
	// asked for (design D7).
	s := reviewReady(t)
	s.Do(Send{Text: "do it"})
	s.Do(TurnSnapshot{Turn: 1, Problem: "its start's snapshot took more than 5 s"})
	reviewEvents(s, turnOpen)
	if got := reviewCmds(reviewEvents(s, turnClose("completed"))); got != nil {
		t.Fatalf("a failed start's end asked %v", got)
	}
	if r := s.Stages[0].Review; !r.Shown() || r.Problem != "its start's snapshot took more than 5 s" {
		t.Fatalf("review %+v", r)
	}
	// An end that failed, the store full: its row says why.
	s.Do(Send{Text: "again"})
	s.Do(TurnSnapshot{Turn: 2, Tree: "t1"})
	reviewEvents(s, turnOpen, turnClose("completed"))
	s.Do(TurnSnapshot{Turn: 2, End: true, Problem: "the review's store reached its 256.0 MiB for this session"})
	if r := s.Stages[1].Review; !r.Shown() || r.Problem == "" || r.Files != nil {
		t.Fatalf("review %+v", r)
	}
	// The store that could not be made: each turn says so at its end.
	f := reviewReady(t)
	f.Do(ReviewStore{Problem: "the review's store could not be made: denied"})
	f.Do(Send{Text: "do it"})
	reviewEvents(f, turnOpen, turnClose("completed"))
	if r := f.Stages[0].Review; !r.Shown() || r.Problem != "the review's store could not be made: denied" {
		t.Fatalf("review %+v", r)
	}
	// Outside a git repository: nothing asked, nothing shown.
	g := reviewReady(t)
	g.Do(ReviewStore{})
	g.Do(Send{Text: "do it"})
	if got := reviewCmds(reviewEvents(g, turnOpen, turnClose("completed"))); got != nil || g.Stages[0].Review.Shown() {
		t.Fatalf("no store: asked %v, review %+v", got, g.Stages[0].Review)
	}
}

func TestReviewCLITurn(t *testing.T) {
	// A turn the CLI opens starts where the turn before ended (design D4),
	// also when that end's snapshot comes after it opened.
	s := reviewReady(t)
	s.Do(Send{Text: "do it"})
	s.Do(TurnSnapshot{Turn: 1, Tree: "t1"})
	reviewEvents(s, turnOpen, turnClose("completed"))
	reviewEvents(s, turnOpen) // the CLI's own turn, before turn 1's end snapshot came
	if st := s.Stages[1]; !st.ByCLI || !st.Review.Prev || st.Review.Asked || st.Review.From != "" {
		t.Fatalf("the CLI's stage %+v", st)
	}
	got := reviewCmds(s.Do(TurnSnapshot{Turn: 1, End: true, Tree: "t2"}))
	if want := []string{"files 1 t1..t2"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("turn 1's end asked %v, want %v", got, want)
	}
	if s.Stages[1].Review.From != "t2" {
		t.Fatalf("turn 2 starts at %q", s.Stages[1].Review.From)
	}
	got = reviewCmds(reviewEvents(s, turnClose("completed")))
	if want := []string{"snapshot 2 end=true"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("turn 2's end asked %v, want %v", got, want)
	}
	got = reviewCmds(s.Do(TurnSnapshot{Turn: 2, End: true, Tree: "t3"}))
	if want := []string{"files 2 t2..t3"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("turn 2's snapshot asked %v, want %v", got, want)
	}
}

func TestReviewCLITurnNoStart(t *testing.T) {
	// The turn before failed its start: the CLI's turn after it has no
	// start either, and says so rather than waiting for one.
	s := reviewReady(t)
	s.Do(Send{Text: "do it"})
	s.Do(TurnSnapshot{Turn: 1, Problem: "its start's snapshot took more than 5s"})
	reviewEvents(s, turnOpen, turnClose("completed"))
	reviewEvents(s, turnOpen) // the CLI's own turn
	if r := s.Stages[1].Review; !r.Shown() || r.Problem != "no snapshot at its start: its start's snapshot took more than 5s" {
		t.Fatalf("the CLI's turn's review %+v", r)
	}
	if got := reviewCmds(reviewEvents(s, turnClose("completed"))); got != nil {
		t.Fatalf("its end asked %v", got)
	}
}

func TestReviewMock(t *testing.T) {
	// A prompt the dashboard did not send, a mock file's: the start's
	// snapshot is asked for when its stage opens (design D4).
	s := reviewReady(t)
	got := reviewCmds(reviewEvents(s, event.Event{Kind: event.UserMessage, Payload: event.User{Text: "replayed"}}))
	if want := []string{"snapshot 1 end=false"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("a replayed prompt asked %v, want %v", got, want)
	}
	if got := reviewCmds(reviewEvents(s, turnOpen)); got != nil {
		t.Fatalf("the turn's start asked again %v", got)
	}
}

func TestReviewCrash(t *testing.T) {
	// The session ends during the turn: its end's snapshot is taken then.
	s := reviewReady(t)
	s.Do(Send{Text: "do it"})
	s.Do(TurnSnapshot{Turn: 1, Tree: "t1"})
	reviewEvents(s, turnOpen)
	got := reviewCmds(reviewEvents(s, event.Event{Kind: event.SessionEnded, Payload: event.Ended{Code: 1, HasCode: true}}))
	if want := []string{"snapshot 1 end=true"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("a crash asked %v, want %v", got, want)
	}
}

func TestReviewFileCap(t *testing.T) {
	s := reviewReady(t)
	s.Do(Send{Text: "do it"})
	s.Do(TurnSnapshot{Turn: 1, Tree: "t1"})
	reviewEvents(s, turnOpen, turnClose("completed"))
	s.Do(TurnSnapshot{Turn: 1, End: true, Tree: "t2"})
	files := make([]ReviewFile, ReviewFileCap+500)
	s.Do(TurnFiles{Turn: 1, Files: files, Total: len(files)})
	if r := s.Stages[0].Review; len(r.Files) != ReviewFileCap || r.Total != ReviewFileCap+500 {
		t.Fatalf("kept %d of %d", len(r.Files), r.Total)
	}
	// An answer for no turn is noted, never applied.
	if s.Do(TurnSnapshot{Turn: 9, Tree: "x"}); len(s.Notes) == 0 {
		t.Fatal("no note for a snapshot of no turn")
	}
}
