package state

import (
	"testing"
	"time"

	"github.com/Evengelius/agent-command-center/internal/event"
)

// Plan 72's associations (S1 to S3; events-and-state.md, Conversation
// record): a call and the requests naming it, Codex's review and the call it
// names, the classifier's time.

// assoc applies evs to s, numbered and timed one second apart.
func assoc(s *State, evs ...event.Event) {
	for _, ev := range evs {
		n := len(s.Log) + 1
		ev.Seq, ev.At, ev.Provider, ev.Origin = uint64(n), time.Date(2026, 10, 31, 14, 0, n, 0, time.UTC), "codex", event.Observed
		s.Apply(ev)
	}
}

// TestCallRequests is S1: a call keeps the last request naming it, matched
// whichever came first, answered or not; a request naming no call known
// matches nothing.
func TestCallRequests(t *testing.T) {
	s := New()
	s.Context = Context{Provider: "codex"}
	started := func(id string) event.Event {
		return event.Event{Kind: event.ToolStarted, Payload: event.Tool{ToolID: id, Name: "commandExecution"}}
	}
	ask := func(id, tool string) event.Event {
		return event.Event{Kind: event.ApprovalRequest, Payload: event.Approval{ID: id, Tool: "commandExecution", ToolID: tool}}
	}
	// The call first, as both CLIs report it (P3b, P6b, P12x).
	assoc(s, started("c1"), ask("r1", "c1"))
	if got := s.Tools["c1"].Approval; got != "r1" {
		t.Errorf("a request after its call: %q", got)
	}
	// The request first: matched when the call starts, the newest pending.
	assoc(s, ask("r2", "c2"), ask("r3", "c2"), started("c2"))
	if got := s.Tools["c2"].Approval; got != "r3" {
		t.Errorf("requests before their call: %q, want the newest", got)
	}
	// A second request for a call: the last.
	assoc(s, event.Event{Kind: event.ApprovalResolved, Payload: event.Approval{ID: "r1", Decision: event.Allow}}, ask("r4", "c1"))
	if got := s.Tools["c1"].Approval; got != "r4" {
		t.Errorf("a second request: %q", got)
	}
	// A request naming no call, then a call of another id: nothing joined.
	assoc(s, ask("r5", "nowhere"), started("c3"))
	if got := s.Tools["c3"].Approval; got != "" {
		t.Errorf("a call joined to a request naming another: %q", got)
	}
	// A request answered before its call started still names it.
	assoc(s, ask("r6", "c4"), event.Event{Kind: event.ApprovalResolved, Payload: event.Approval{ID: "r6", Decision: event.Deny}}, started("c4"))
	if got := s.Tools["c4"].Approval; got != "r6" {
		t.Errorf("a request answered before its call: %q", got)
	}
}

// TestCallReview is S2 and S3: Codex's review keeps the item it names, on
// its entry and on the call, updated in place from started to completed;
// the classifier's verdict keeps its time.
func TestCallReview(t *testing.T) {
	s := New()
	s.Context = Context{Provider: "codex"}
	assoc(s, event.Event{Kind: event.ToolStarted, Payload: event.Tool{ToolID: "c1", Name: "commandExecution"}},
		event.Event{Kind: event.ReviewUpdated, Item: "c1", Payload: event.Review{ReviewID: "rv1", Status: "inProgress", Command: "ls"}})
	tc := s.Tools["c1"]
	if tc.Review == nil || tc.Review.ID != "rv1" || tc.Review.Status != "inProgress" {
		t.Fatalf("the review in progress: %+v", tc.Review)
	}
	first := tc.Review
	assoc(s, event.Event{Kind: event.ReviewUpdated, Item: "c1", Payload: event.Review{ReviewID: "rv1", Status: "approved", Risk: "low", Authorization: "unknown", Rationale: "reads only"}})
	want := Review{ID: "rv1", Status: "approved", Risk: "low", Authorization: "unknown", Rationale: "reads only", At: s.Log[len(s.Log)-1].At}
	if tc.Review == first || *tc.Review != want {
		t.Errorf("the review completed: %+v, want a new %+v", tc.Review, want)
	}
	var reviews []Entry
	for _, e := range s.Conversation {
		if e.Role == "review" {
			reviews = append(reviews, e)
		}
	}
	if len(reviews) != 1 || reviews[0].Target != "c1" || reviews[0].Status != "approved" {
		t.Errorf("the review's entry: %+v", reviews)
	}
	// A review naming an item the state does not know: its entry, its target
	// kept, no call.
	assoc(s, event.Event{Kind: event.ReviewUpdated, Item: "c9", Payload: event.Review{ReviewID: "rv2", Status: "denied"}})
	if last := s.Conversation[len(s.Conversation)-1]; last.Target != "c9" || s.Tools["c9"] != nil {
		t.Errorf("a review of an unknown item: %+v, call %v", last, s.Tools["c9"])
	}
	// Claude Code's classifier: its verdict's time (S3).
	c := New()
	c.Context = Context{Provider: "claude-code"}
	assoc(c, event.Event{Kind: event.ToolStarted, Payload: event.Tool{ToolID: "t1", Name: "Bash"}},
		event.Event{Kind: event.ToolDecision, Payload: event.Verdict{ToolID: "t1", ReasonType: "classifier", Decision: event.Deny}})
	if t1 := c.Tools["t1"]; t1.Reviewed != "denied by auto-review" || !t1.ReviewedAt.Equal(c.Log[len(c.Log)-1].At) {
		t.Errorf("the classifier: %q at %v", t1.Reviewed, t1.ReviewedAt)
	}
}
