package app

import (
	"encoding/json"
	"slices"
	"strings"
	"sync"
	"testing"
	"testing/synctest"

	tea "charm.land/bubbletea/v2"

	"github.com/Evengelius/agent-command-center/internal/event"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// Plan 49's criterion 49.2 (design D2; audit finding 12): the application's
// write queue, which puts the writes to the session in the order Update
// asked for them, though Bubble Tea runs a batch's commands at once.

// askedScript is a ready Claude Code session whose turn runs with approval
// requests r1 and r2 pending, the first with its card in focus.
func askedScript(t *testing.T) (*harness, *scriptCLI) {
	t.Helper()
	h, cli := readyScript(t, "claude-code", false, event.Started{SessionID: "s1", Version: "2.1.300", PermissionMode: "default", ApprovalMode: event.ModeManual})
	in := json.RawMessage(`{"command":"ls"}`)
	pushEvents(h, "claude-code", evOf(event.TurnStarted, nil),
		evOf(event.ApprovalRequest, event.Approval{ID: "r1", Tool: "Bash", ToolID: "t1", Input: in}),
		evOf(event.ApprovalRequest, event.Approval{ID: "r2", Tool: "Bash", ToolID: "t2", Input: in}))
	h.send(key(tea.KeyF2, 0))
	if h.m.ui.Card != "r1" {
		t.Fatalf("card %q in focus, want r1", h.m.ui.Card)
	}
	return h, cli
}

// TestWriteOrder: c on an approval card cancels it, then interrupts the turn.
// While the CLI holds the answer, the interrupt is not written: it waits
// behind it, and comes after it.
func TestWriteOrder(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, cli := askedScript(t)
		gate := cli.sess.hold()
		h.send(typed('c'))
		if got := cli.sess.calledInOrder(); len(got) != 0 {
			t.Fatalf("written while the CLI holds the answer: %q", got)
		}
		close(gate)
		h.settle()
		// The turn's interrupt cancels the other pending request first
		// (ui-design.md, Approvals).
		want := []string{"answer r1 cancel", "answer r2 cancel", "interrupt"}
		if got := cli.sess.calledInOrder(); !slices.Equal(got, want) {
			t.Fatalf("written %q, want %q", got, want)
		}
	})
}

// TestQuitWritesInOrder: a confirmed quit cancels the pending requests and
// interrupts the turn before the CLI's input is closed, in that order, the
// close waiting behind the answers the CLI holds.
func TestQuitWritesInOrder(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, cli := askedScript(t)
		gate := cli.sess.hold()
		h.send(key('q', tea.ModCtrl))
		if !h.m.State().QuitAsked {
			t.Fatal("Ctrl+Q did not ask while the turn runs")
		}
		h.send(key('q', tea.ModCtrl))
		if got := cli.sess.calledInOrder(); len(got) != 0 {
			t.Fatalf("written while the CLI holds the answer: %q", got)
		}
		close(gate)
		h.settle()
		want := []string{"answer r1 cancel", "answer r2 cancel", "interrupt", "close"}
		if got := cli.sess.calledInOrder(); !slices.Equal(got, want) {
			t.Fatalf("written %q, want %q", got, want)
		}
	})
}

// gatedWrites queues n writes straight through Model.write, the first held
// until the returned channel closes; ran logs the ones that ran, in order.
func gatedWrites(h *harness, n int, full tea.Msg) (gate chan struct{}, cmds []tea.Cmd, ran func() []int) {
	gate = make(chan struct{})
	var mu sync.Mutex
	var log []int
	for i := range n {
		cmds = append(cmds, h.m.write(func() tea.Msg {
			if i == 0 {
				<-gate
			}
			mu.Lock()
			log = append(log, i)
			mu.Unlock()
			return nil
		}, full))
		if i == 0 {
			synctest.Wait() // the worker has taken it: it runs, the rest wait
		}
	}
	return gate, cmds, func() []int {
		mu.Lock()
		defer mu.Unlock()
		return slices.Clone(log)
	}
}

// TestWriteQueueFull: one write running and writeQueue waiting, the next is
// refused at once with its message; the accepted ones then run in order.
func TestWriteQueueFull(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, _ := startScript(t, "claude-code")
		full := actionMsg{state.WriteFailed{Message: "too many writes waiting"}}
		gate, cmds, ran := gatedWrites(h, writeQueue+2, full)
		if msg := cmds[writeQueue+1](); msg != full {
			t.Fatalf("the write past the queue gave %v, want the refusal", msg)
		}
		close(gate)
		for _, c := range cmds[:writeQueue+1] {
			if msg := c(); msg != nil {
				t.Fatalf("an accepted write gave %v", msg)
			}
		}
		want := make([]int, writeQueue+1)
		for i := range want {
			want[i] = i
		}
		if got := ran(); !slices.Equal(got, want) {
			t.Fatalf("ran %v, want 0 to %d in order", got, writeQueue)
		}
	})
}

// TestWriteFailureGoesOn: a write that fails and one that panics each give
// their message, and the writes after them still run.
func TestWriteFailureGoesOn(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, _ := startScript(t, "claude-code")
		failed := actionMsg{state.ApprovalFailed{Message: "the mode was refused"}}
		ranLast := false
		cmds := []tea.Cmd{
			h.m.write(func() tea.Msg { return failed }, nil),
			h.m.write(func() tea.Msg { panic("the adapter broke") }, nil),
			h.m.write(func() tea.Msg { ranLast = true; return nil }, nil),
		}
		if msg := cmds[0](); msg != failed {
			t.Fatalf("the failing write gave %v", msg)
		}
		p, ok := cmds[1]().(actionMsg)
		if a, isPanic := p.a.(state.Panicked); !ok || !isPanic || !strings.HasPrefix(a.Message, "writing to the CLI failed: panic: the adapter broke") {
			t.Fatalf("the panicking write gave %v", p)
		}
		if msg := cmds[2](); msg != nil || !ranLast {
			t.Fatalf("the write after them gave %v, ran %v", msg, ranLast)
		}
	})
}

// TestCloseWhenQueueFull: with the queue full, the quit's close is not
// queued behind writes the CLI is not reading: it comes at once.
func TestCloseWhenQueueFull(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, cli := readyScript(t, "claude-code", false, event.Started{SessionID: "s1", Version: "2.1.300", PermissionMode: "default", ApprovalMode: event.ModeManual})
		gate, _, _ := gatedWrites(h, writeQueue+1, nil)
		h.send(key('q', tea.ModCtrl)) // nothing runs: no question
		if got := cli.sess.calledInOrder(); !slices.Equal(got, []string{"close"}) {
			t.Fatalf("with the queue full: %q, want the close at once", got)
		}
		close(gate)
	})
}
