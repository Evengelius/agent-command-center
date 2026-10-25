// Package pipe is what every adapter's session needs between a CLI's pipes
// and the application: an ordered, unbounded event queue, a bounded line
// reader (doc/reliability.md, Backpressure), and the process's lifecycle the
// sessions share (Proc; plan 49, design D8). It knows no CLI: a session
// gives its CLI's name and what each line means.
package pipe

import (
	"bufio"
	"bytes"
	"errors"
	"sync"
	"time"

	"github.com/Evengelius/agent-command-center/internal/event"
)

// Queue is unbounded, so the readers never wait for the screen, and keeps
// the order events were mapped in. A written line's events are held until
// the write ends: released if it succeeded, dropped if not. Seq is set when
// an event leaves, so a dropped one leaves no gap.
type Queue struct {
	mu     sync.Mutex
	cond   *sync.Cond
	items  []*Item
	closed bool
	now    func() time.Time
	out    chan event.Event
	seq    uint64
}

// Item is one queued event; Hold returns them so Release can find them.
type Item struct {
	ev   event.Event
	held bool
	drop bool
}

// NewQueue returns an empty queue stamping events with now.
func NewQueue(now func() time.Time) *Queue {
	q := &Queue{now: now, out: make(chan event.Event, 256)}
	q.cond = sync.NewCond(&q.mu)
	return q
}

// Out is the channel Forward sends to; it closes after the last event.
func (q *Queue) Out() <-chan event.Event { return q.out }

// Push queues one event, ready to leave.
func (q *Queue) Push(ev event.Event) {
	q.mu.Lock()
	if q.closed { // after session.ended
		q.mu.Unlock()
		return
	}
	ev.At = q.now()
	q.items = append(q.items, &Item{ev: ev})
	q.mu.Unlock()
	q.cond.Signal()
}

// Hold queues events that wait for Release.
func (q *Queue) Hold(evs []event.Event) []*Item {
	q.mu.Lock()
	defer q.mu.Unlock()
	held := make([]*Item, len(evs))
	for i, ev := range evs {
		ev.At = q.now()
		held[i] = &Item{ev: ev, held: true}
		if !q.closed {
			q.items = append(q.items, held[i])
		}
	}
	return held
}

// Release lets held events leave, or drops them when keep is false.
func (q *Queue) Release(held []*Item, keep bool) {
	q.mu.Lock()
	for _, it := range held {
		it.held, it.drop = false, !keep
	}
	q.mu.Unlock()
	q.cond.Signal()
}

// Close ends the queue: what is queued still leaves, nothing more enters.
func (q *Queue) Close() {
	q.mu.Lock()
	q.closed = true
	q.mu.Unlock()
	q.cond.Signal()
}

// Forward sends the events in order and closes the channel after the last.
func (q *Queue) Forward() {
	for {
		q.mu.Lock()
		for !(len(q.items) > 0 && !q.items[0].held) && !(q.closed && len(q.items) == 0) {
			q.cond.Wait()
		}
		if len(q.items) == 0 {
			q.mu.Unlock()
			close(q.out)
			return
		}
		it := q.items[0]
		q.items[0] = nil
		q.items = q.items[1:]
		if it.drop {
			q.mu.Unlock()
			continue
		}
		q.seq++
		it.ev.Seq = q.seq
		q.mu.Unlock()
		q.out <- it.ev
	}
}

// ReadLine reads one line without its end of line. A line longer than limit
// is read to its end and dropped: over is true and size is its length.
func ReadLine(br *bufio.Reader, limit int) (l []byte, size int, over bool, err error) {
	for {
		var chunk []byte
		chunk, err = br.ReadSlice('\n')
		size += len(chunk)
		if !over {
			l = append(l, chunk...)
			if len(bytes.TrimRight(l, "\r\n")) > limit {
				over, l = true, nil
			}
		}
		if errors.Is(err, bufio.ErrBufferFull) {
			continue
		}
		return bytes.TrimRight(l, "\r\n"), size, over, err
	}
}
