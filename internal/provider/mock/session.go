package mock

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/Evengelius/agent-command-center/internal/event"
	"github.com/Evengelius/agent-command-center/internal/provider"
)

// MaxGap bounds the wait between two replayed events, so a recording's idle
// minutes do not stall the screen.
const MaxGap = 2 * time.Second

// Provider replays one mock file. It is never detected and never offered on
// the startup screen.
type Provider struct {
	Events []Timed
	Speed  float64          // 1 replays at the recorded pace; 0 sends every event at once
	Now    func() time.Time // the time stamped on each event; time.Now when nil
}

// ID is the provider id.
func (p *Provider) ID() string { return ProviderID }

// DisplayName is the name shown in the project bar.
func (p *Provider) DisplayName() string { return "Mock data" }

// Detect reports the mock provider as installed: it needs no executable.
func (p *Provider) Detect(context.Context) provider.Detection {
	return provider.Detection{ID: ProviderID, Name: p.DisplayName(), Installed: true}
}

// Start begins the replay. It starts no process.
func (p *Provider) Start(_ context.Context, sc provider.SessionContext) (provider.Session, error) {
	if !sc.Mock {
		return nil, errors.New("the mock provider runs only in mock mode")
	}
	now := p.Now
	if now == nil {
		now = time.Now
	}
	s := &Session{ch: make(chan event.Event, 4096), done: make(chan struct{}), now: now}
	if p.Speed <= 0 && len(p.Events) <= cap(s.ch) {
		// At once, literally: every event queued before the session starts, so
		// the application reads them as one batch, the same in every run. From
		// a goroutine, where the batches fell depended on the scheduler, and
		// with them which thinking phase a frame drew live (plan 22, design 7).
		s.replay(p.Events, 0)
		return s, nil
	}
	go s.replay(p.Events, p.Speed)
	return s, nil
}

// Session is a mock session.
type Session struct {
	mu     sync.Mutex
	ch     chan event.Event
	done   chan struct{}
	closed bool
	seq    uint64
	now    func() time.Time
}

func (s *Session) replay(evs []Timed, speed float64) {
	var last time.Duration
	for _, t := range evs {
		if speed > 0 {
			gap := time.Duration(float64(t.After-last) / speed)
			if gap > MaxGap {
				gap = MaxGap
			}
			if gap > 0 {
				timer := time.NewTimer(gap)
				select {
				case <-timer.C:
				case <-s.done:
					timer.Stop()
					return
				}
			}
		}
		last = t.After
		if !s.emit(t.Event) {
			return
		}
	}
}

// emit stamps and sends one event; false once the session is closed. The
// channel is read until it closes, so holding the lock while sending only
// waits for the reader.
func (s *Session) emit(ev event.Event) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.emitLocked(ev)
}

func (s *Session) emitLocked(ev event.Event) bool {
	if s.closed {
		return false
	}
	s.seq++
	ev.Seq, ev.At, ev.Origin = s.seq, s.now(), event.Mock
	if ev.Provider == "" {
		ev.Provider = ProviderID
	}
	s.ch <- ev
	return true
}

// Events are the replayed events, then the ones Send and Answer add.
func (s *Session) Events() <-chan event.Event { return s.ch }

// Send goes nowhere: it records the message and a notice saying so, as a
// turn that ends at once.
func (s *Session) Send(text string) error {
	ok := s.emit(event.Event{Kind: event.UserMessage, Payload: event.User{Text: text}}) &&
		s.emit(event.Event{Kind: event.TurnStarted}) &&
		s.emit(event.Event{Kind: event.StatusChanged, Payload: event.Status{Status: "notice", Detail: "mock session: the message was not sent to any CLI"}}) &&
		s.emit(event.Event{Kind: event.TurnCompleted, Payload: event.TurnEnd{Outcome: "completed", Subtype: "mock"}})
	if !ok {
		return errors.New("the mock session is closed")
	}
	return nil
}

// Answer records the answer; no CLI receives it.
func (s *Session) Answer(id, decision string) error {
	if !s.emit(event.Event{Kind: event.ApprovalResolved, Payload: event.Approval{ID: id, Decision: decision}}) {
		return errors.New("the mock session is closed")
	}
	return nil
}

// AnswerQuestion records the answer as an allow; no CLI receives it.
func (s *Session) AnswerQuestion(id string, answers map[string]string) error {
	return s.Answer(id, "allow")
}

// Interrupt ends the running turn as interrupted.
func (s *Session) Interrupt() error {
	if !s.emit(event.Event{Kind: event.TurnCompleted, Payload: event.TurnEnd{Outcome: "interrupted", Subtype: "mock"}}) {
		return errors.New("the mock session is closed")
	}
	return nil
}

// SetApprovalMode: a mock session has no approvals to switch.
func (s *Session) SetApprovalMode(bool) error {
	return errors.New("the mock session has no approval mode")
}

// Refresh has nothing to read.
func (s *Session) Refresh() error { return nil }

// Capabilities: a mock session offers nothing beyond what its file shows.
func (s *Session) Capabilities() provider.Capabilities { return provider.Capabilities{} }

// Close stops the replay and ends the session with session.ended, the last
// event: the lock is held from that event to closing the channel.
func (s *Session) Close(context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil
	}
	s.emitLocked(event.Event{Kind: event.SessionEnded, Payload: event.Ended{Code: 0, HasCode: true, Reason: "mock session closed"}})
	s.closed = true
	close(s.done)
	close(s.ch)
	return nil
}
