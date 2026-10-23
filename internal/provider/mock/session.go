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
	// What the replay showed (plan 34): the CLI its events came from, the
	// models listed and the settings in force, which SetModelSettings
	// answers from.
	cli      string
	models   []event.ModelEntry
	settings event.Settings
	next     provider.ModelRequest // a Codex file's change, until the next message
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
	} else if s.cli == "" && ev.Provider != ProviderID {
		s.cli = ev.Provider
	}
	switch p := ev.Payload.(type) {
	case event.Models:
		s.models = p.Entries
	case event.Settings:
		if !p.Refused {
			s.settings = merged(s.settings, p)
		}
	}
	s.ch <- ev
	return true
}

// merged is the settings in force after a report: the fields it gives
// replace the earlier ones.
func merged(cur, p event.Settings) event.Settings {
	if p.Model != "" {
		cur.Model, cur.Choice = p.Model, p.Choice
	}
	if p.EffortKnown {
		cur.Effort, cur.EffortKnown, cur.EffortLock = p.Effort, true, p.EffortLock
	}
	if p.UltracodeKnown {
		cur.Ultracode, cur.UltracodeAvailable, cur.UltracodeKnown = p.Ultracode, p.UltracodeAvailable, true
	}
	return cur
}

// Events are the replayed events, then the ones Send and Answer add.
func (s *Session) Events() <-chan event.Event { return s.ch }

// Send goes nowhere: it records the message, its images named, and a
// notice saying so, as a turn that ends at once.
func (s *Session) Send(msg provider.Message) error {
	var imgs []event.Image
	for _, img := range msg.Images {
		imgs = append(imgs, event.Image{Name: img.Name, Media: img.Media, Size: len(img.Data)})
	}
	ok := s.emit(event.Event{Kind: event.UserMessage, Payload: event.User{Text: msg.Text, Images: imgs}}) &&
		s.emit(event.Event{Kind: event.TurnStarted}) &&
		s.emit(event.Event{Kind: event.StatusChanged, Payload: event.Status{Status: "notice", Detail: "mock session: the message was not sent to any CLI"}}) &&
		s.emit(event.Event{Kind: event.TurnCompleted, Payload: event.TurnEnd{Outcome: "completed", Subtype: "mock"}})
	if !ok {
		return errors.New("the mock session is closed")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if next := s.next; next != (provider.ModelRequest{}) {
		// A Codex file's change goes with the message (plan 34).
		s.next = provider.ModelRequest{}
		return s.applyLocked(next)
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

// SetModelSettings answers a change as the replayed CLI would, with no CLI
// (plan 34): a Claude Code file's at once, a Codex file's with the next
// message (Send). The settings asked for are reported in force, from the
// last ones the file or an earlier change gave. Every event it adds has
// origin mock.
func (s *Session) SetModelSettings(r provider.ModelRequest) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cli == "codex" {
		if r.Model != "" {
			s.next.Model = r.Model
		}
		if r.Effort != "" {
			s.next.Effort = r.Effort
		}
		return nil
	}
	return s.applyLocked(r)
}

// applyLocked reports the settings a change gives as in force.
func (s *Session) applyLocked(r provider.ModelRequest) error {
	cur := s.settings
	if r.Model != "" {
		cur.Model, cur.Choice = r.Model, r.Model
		for _, e := range s.models {
			if e.ID == r.Model {
				cur.Model = e.Model
				if len(e.Efforts) == 0 {
					cur.Effort = ""
				}
			}
		}
		cur.UltracodeAvailable = cur.UltracodeKnown && s.ultracodeOn(cur.Model)
		if !cur.UltracodeAvailable {
			cur.Ultracode = false
		}
	}
	if r.Effort != "" {
		cur.Effort, cur.EffortKnown = r.Effort, true
	}
	if r.Ultracode != nil && cur.UltracodeAvailable {
		cur.Ultracode = *r.Ultracode
	}
	cur.Source, cur.Undocumented, cur.Answers, cur.Refused, cur.Text = "mock", false, true, false, ""
	if !s.emitLocked(event.Event{Kind: event.ModelSettings, Payload: cur}) {
		return errors.New("the mock session is closed")
	}
	s.settings = cur
	return nil
}

// ultracodeOn says the model takes Ultracode in the mock: as Claude Code, a
// model whose levels include "xhigh" (model-config, "When ultracode is
// available").
func (s *Session) ultracodeOn(model string) bool {
	for _, e := range s.models {
		if e.Model == model {
			for _, l := range e.Efforts {
				if l == "xhigh" {
					return true
				}
			}
		}
	}
	return false
}

// Refresh has nothing to read.
func (s *Session) Refresh() error { return nil }

// Capabilities: a mock session offers nothing beyond what its file shows.
// Capabilities are none of a CLI's but the images: a mock run takes them
// with Claude Code's limits, the stricter, so the editor's attachments can
// be shown and sent nowhere (plan 31).
func (s *Session) Capabilities() provider.Capabilities {
	s.mu.Lock()
	cli := s.cli
	s.mu.Unlock()
	// The models are offered as the replayed CLI offers them (plan 34): a
	// change answered at once, Ultracode for Claude Code's files only.
	return provider.Capabilities{Images: provider.ImageLimits{MaxCount: 100, MaxBytes: 7_400_000, MaxTotal: 7_400_000, MaxSide: 8000},
		Models: true, ModelsNextTurn: cli == "codex", Ultracode: cli == "claude-code"}
}

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
