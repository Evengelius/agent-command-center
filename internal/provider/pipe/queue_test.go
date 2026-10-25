package pipe

import (
	"fmt"
	"strconv"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/Evengelius/agent-command-center/internal/event"
)

// Plan 49's criterion 49.1 (design D1; audit finding 12): the queue between
// an adapter's readers and the application, pinned before the adapters'
// lifecycle moves onto it (design D8). Each test runs in a bubble, where a
// deadlock fails the test, and a reader blocked on Out is seen to wait.

// line is an event named by its text.
func line(text string) event.Event {
	return event.Event{Kind: event.Stderr, Payload: event.Line{Text: text}}
}

func textOf(e event.Event) string { return e.Payload.(event.Line).Text }

// drain reads every event until Out closes.
func drain(q *Queue) []event.Event {
	var out []event.Event
	for e := range q.Out() {
		out = append(out, e)
	}
	return out
}

// quiet says nothing can be read from Out once every goroutine of the
// bubble waits: the events behind a hold stay behind it.
func quiet(t *testing.T, q *Queue) {
	t.Helper()
	synctest.Wait()
	select {
	case e, ok := <-q.Out():
		if !ok {
			t.Fatal("Out closed")
		}
		t.Fatalf("%q left", textOf(e))
	default:
	}
}

// next reads one event, which must be text with sequence number seq.
func next(t *testing.T, q *Queue, text string, seq uint64) {
	t.Helper()
	e, ok := <-q.Out()
	if !ok || textOf(e) != text || e.Seq != seq || e.At.IsZero() {
		t.Fatalf("read %+v (open %v), want %q with Seq %d and a time", e, ok, text, seq)
	}
}

// TestQueueProducers: eight producers pushing at once; every event leaves
// once, each producer's in the order it pushed them, numbered from 1 with no
// gap.
func TestQueueProducers(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		q := NewQueue(time.Now)
		go q.Forward()
		got := make(chan []event.Event)
		go func() { got <- drain(q) }()
		const producers, each = 8, 1000
		var wg sync.WaitGroup
		for p := range producers {
			wg.Go(func() {
				for i := range each {
					q.Push(line(fmt.Sprintf("%d %d", p, i)))
				}
			})
		}
		wg.Wait()
		q.Close()
		out := <-got
		if len(out) != producers*each {
			t.Fatalf("%d events left, want %d", len(out), producers*each)
		}
		last := make([]int, producers)
		for i := range last {
			last[i] = -1
		}
		for i, e := range out {
			var p, n int
			if _, err := fmt.Sscanf(textOf(e), "%d %d", &p, &n); err != nil {
				t.Fatal(err)
			}
			if e.Seq != uint64(i+1) {
				t.Fatalf("event %d has Seq %d", i, e.Seq)
			}
			if n != last[p]+1 {
				t.Fatalf("producer %d: event %d after %d", p, n, last[p])
			}
			last[p] = n
		}
	})
}

// TestQueueHold: a hold keeps the events after it waiting; released, its
// events leave in their place; dropped, they never leave, and the numbering
// has no gap.
func TestQueueHold(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		q := NewQueue(time.Now)
		go q.Forward()
		q.Push(line("a"))
		held := q.Hold([]event.Event{line("b"), line("c")})
		q.Push(line("d"))
		next(t, q, "a", 1)
		quiet(t, q)
		q.Release(held, true)
		next(t, q, "b", 2)
		next(t, q, "c", 3)
		next(t, q, "d", 4)

		dropped := q.Hold([]event.Event{line("e")})
		q.Push(line("f"))
		quiet(t, q)
		q.Release(dropped, false)
		next(t, q, "f", 5)
		q.Close()
		if rest := drain(q); len(rest) != 0 {
			t.Fatalf("after the close: %d events", len(rest))
		}
	})
}

// TestQueueClose: a close lets the queued events leave, then closes Out; a
// push after it is dropped, and a hold after it queues nothing.
func TestQueueClose(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		q := NewQueue(time.Now)
		q.Push(line("a"))
		q.Push(line("b"))
		q.Close()
		q.Push(line("c"))
		late := q.Hold([]event.Event{line("held")})
		go q.Forward()
		out := drain(q)
		if len(out) != 2 || textOf(out[0]) != "a" || textOf(out[1]) != "b" || out[1].Seq != 2 {
			t.Fatalf("after the close: %+v, want a then b", out)
		}
		q.Release(late, true) // harmless: it was never queued
	})
}

// TestQueueCloseWaitsForHold: a hold not yet released keeps Out open after
// the close, until its release lets its events go or drops them: a write in
// progress when the session ends still has its events placed (every write's
// path releases its hold).
func TestQueueCloseWaitsForHold(t *testing.T) {
	for _, keep := range []bool{true, false} {
		t.Run(strconv.FormatBool(keep), func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				q := NewQueue(time.Now)
				go q.Forward()
				held := q.Hold([]event.Event{line("x")})
				q.Close()
				quiet(t, q)
				q.Release(held, keep)
				out := drain(q)
				if keep && (len(out) != 1 || textOf(out[0]) != "x") || !keep && len(out) != 0 {
					t.Fatalf("released with keep %v: %+v", keep, out)
				}
			})
		})
	}
}

// TestQueueSlowReader: with nothing reading Out, the pushes never wait for a
// reader, so an adapter's readers never wait for the screen; once read, the
// events come in their order.
func TestQueueSlowReader(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		q := NewQueue(time.Now)
		go q.Forward()
		const n = 100_000
		for i := range n {
			q.Push(line(strconv.Itoa(i)))
		}
		q.Close()
		out := drain(q)
		if len(out) != n {
			t.Fatalf("%d events, want %d", len(out), n)
		}
		for i, e := range out {
			if textOf(e) != strconv.Itoa(i) || e.Seq != uint64(i+1) {
				t.Fatalf("event %d: %q, Seq %d", i, textOf(e), e.Seq)
			}
		}
	})
}
