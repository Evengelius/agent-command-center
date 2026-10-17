package codex

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Evengelius/agent-command-center/internal/event"
	"github.com/Evengelius/agent-command-center/internal/proc"
	"github.com/Evengelius/agent-command-center/internal/provider"
	"github.com/Evengelius/agent-command-center/internal/provider/pipe"
)

// MaxLine bounds one stdout line; a longer one is skipped with an error event
// and reading continues (ASM-20).
const MaxLine = 64 << 20

// maxStderrLine bounds one stderr line kept for the error tail.
const maxStderrLine = 4 << 10

// CloseStep is each wait of the close escalation: stdin closed, then the CLI
// terminated, then its whole tree killed (reliability.md, Close).
const CloseStep = 5 * time.Second

// drainWait bounds the wait for the readers after the exit, when something
// the CLI started still holds its pipes open.
const drainWait = 2 * time.Second

// Sandbox and ApprovalPolicy are the only ones the MVP sends: P6b and P7 ran
// with them and the configuration unchanged, and every write then asks
// through the dashboard (architecture.md, Codex adapter).
const (
	Sandbox        = "read-only"
	ApprovalPolicy = "on-request"
)

// clientInfo names the dashboard in initialize (CX-4.1).
var clientInfo = map[string]string{"name": "agent_command_center", "title": "Agent Command Center", "version": "dev"}

// answers are the decisions an approval request takes (v2 schema; INT 4).
// No answer that persists a rule is ever sent (reliability.md, Approvals).
var answers = map[string]bool{"accept": true, "acceptForSession": true, "decline": true, "cancel": true}

// threadParams are thread/start's and thread/resume's parameters. There is
// no cwd field: the process runs in the project, and sending cwd is what
// could write trust_level into the owner's config.toml (CX-10 q2).
type threadParams struct {
	ThreadID       string `json:"threadId,omitempty"`
	Model          string `json:"model,omitempty"`
	Sandbox        string `json:"sandbox"`
	ApprovalPolicy string `json:"approvalPolicy"`
}

// Session is one running `codex app-server` process.
type Session struct {
	prov  *Provider
	p     proc.Process
	stdin io.WriteCloser
	step  time.Duration

	mu        sync.Mutex            // the mapper, the order of events and what follows
	m         *Mapper               //
	ids       int                   // requests sent
	calls     map[string]string     // id of a request awaiting its response -> method
	approvals map[string]pendingReq // pending approval id -> its request
	resume    string                // the thread to resume, until thread/resume is sent
	pin       string                // the model passed in thread/start, "" for the configured one
	thread    string                // the main thread, "" while none is usable
	turn      string                // the running main turn
	model     string                // the model the main thread reports
	refused   bool                  // the running turn failed with the model refused
	fellBack  bool                  // the model fallback ran: never twice
	from      string                // the refused model, until the new thread starts
	fallback  bool                  // the fallback is due once the line's events are queued
	dir       string                // the project directory, the cwd of config/read

	wmu      sync.Mutex // one write at a time
	inClosed atomic.Bool

	q       *pipe.Queue
	exited  chan struct{}
	closing atomic.Bool
	once    sync.Once
}

// pendingReq is a pending approval: its id exactly as received, which the
// answer echoes, and whether the request could not be read, which only a
// decline or a cancel answers (INT 4 fallback).
type pendingReq struct {
	id         json.RawMessage
	unreadable bool
}

func (p *Provider) start(ctx context.Context, sc provider.SessionContext) (*Session, error) {
	sp := p.Spawner
	if sp == nil {
		sp = proc.Exec{}
	}
	argv := append([]string{"app-server"}, p.Args...)
	pr, err := sp.Spawn(proc.Spec{Path: sc.Executable, Args: argv, Dir: sc.ProjectPath})
	if err != nil {
		return nil, fmt.Errorf("Codex did not start: %w", err)
	}
	step := p.CloseStep
	if step == 0 {
		step = CloseStep
	}
	now := p.Now
	if now == nil {
		now = time.Now
	}
	s := &Session{prov: p, p: pr, stdin: pr.Stdin(), step: step, m: NewMapper(), calls: map[string]string{}, dir: sc.ProjectPath,
		approvals: map[string]pendingReq{}, resume: sc.ResumeID, pin: p.fallbackModel(),
		q: pipe.NewQueue(now), exited: make(chan struct{})}
	s.q.Push(event.Event{Provider: ProviderID, Kind: event.SessionSpawned, Origin: event.Observed,
		Payload: event.Spawned{Argv: append([]string{sc.Executable}, argv...), Dir: sc.ProjectPath}})
	go s.q.Forward()

	out, errs := make(chan struct{}), make(chan struct{})
	go func() { defer close(out); s.read(pr.Stdout()) }()
	go func() { defer close(errs); s.readStderr(pr.Stderr()) }()
	go s.wait(out, errs)
	go func() {
		select {
		case <-ctx.Done(): // the application's forced kill
			_ = pr.Kill()
		case <-s.exited:
		}
	}()
	if err := s.request("initialize", map[string]any{"clientInfo": clientInfo}); err != nil {
		s.problem("initialize not sent: " + err.Error())
	}
	return s, nil
}

// push queues one event of the adapter's own.
func (s *Session) push(ev event.Event) {
	ev.Provider = ProviderID
	s.mu.Lock()
	s.q.Push(ev)
	s.mu.Unlock()
}

func (s *Session) problem(msg string) {
	s.push(event.Event{Kind: event.Error, Origin: event.Observed, Payload: event.Problem{Message: msg}})
}

// read maps stdout line by line until end of file. What a line calls for
// (the next handshake step, a reply) is written apart: the reader never
// waits on stdin, which the CLI may not be reading while its stdout is full.
func (s *Session) read(r io.Reader) {
	br := bufio.NewReaderSize(r, 64<<10)
	for {
		l, size, over, err := pipe.ReadLine(br, MaxLine)
		switch {
		case over:
			s.problem(fmt.Sprintf("a line of %d bytes from Codex was skipped (limit %d MiB)", size, MaxLine>>20))
		case len(bytes.TrimSpace(l)) > 0:
			s.mu.Lock()
			evs := s.m.Out(l)
			for _, ev := range evs {
				s.track(ev)
				s.q.Push(ev)
			}
			next := s.react(l, evs)
			s.mu.Unlock()
			if next != nil {
				go next()
			}
		}
		if err != nil {
			if !errors.Is(err, io.EOF) && !s.closing.Load() {
				s.problem("reading Codex's output: " + err.Error())
			}
			return
		}
	}
}

// track follows the main thread, its running turn and the model refusal,
// from the events of every line read or written. Called with mu held.
func (s *Session) track(ev event.Event) {
	switch p := ev.Payload.(type) {
	case event.Approval:
		if ev.Kind == event.ApprovalResolved {
			delete(s.approvals, p.ID)
		}
	case event.Started:
		s.thread, s.model = p.SessionID, p.Model
	case event.Status:
		if p.Status == "turn accepted" && ev.Agent == "" && ev.Turn != "" {
			s.turn = ev.Turn
		}
	case event.Problem:
		// The refusal is recognised by its text, undocumented: the error
		// names the model in quotes and says it is not supported (P5, P6).
		msg := strings.ToLower(p.Message)
		if ev.Kind == event.Error && ev.Agent == "" && !p.WillRetry && s.model != "" &&
			strings.Contains(msg, "'"+strings.ToLower(s.model)+"'") && strings.Contains(msg, "not supported") {
			s.refused = true
		}
	}
	switch {
	case ev.Kind == event.TurnStarted && ev.Agent == "" && ev.Turn != "":
		s.turn = ev.Turn
	case ev.Kind == event.TurnCompleted && ev.Agent == "":
		s.turn = ""
		if s.refused && !s.fellBack && s.pin == "" {
			s.fellBack, s.from, s.thread, s.fallback = true, s.model, "", true
		}
		s.refused = false
	}
}

// wire is what react reads of a line: enough to tell a request, a response
// and a notification apart.
type wire struct {
	ID     json.RawMessage `json:"id"`
	Method string          `json:"method"`
	Result json.RawMessage `json:"result"`
	Error  json.RawMessage `json:"error"`
}

// react returns what a line read calls for, to run outside the lock: the
// handshake's next step, a reply to an unknown request, the model fallback.
// Called with mu held.
func (s *Session) react(l []byte, evs []event.Event) func() {
	var w wire
	if json.Unmarshal(l, &w) != nil {
		return nil
	}
	hasID := len(w.ID) > 0 && string(w.ID) != "null"
	failed := len(w.Error) > 0 && string(w.Error) != "null"
	var next func()
	switch {
	case w.Method != "" && hasID:
		if !approvalMethod(w.Method) {
			id, method := w.ID, w.Method
			next = func() {
				if err := s.write(map[string]any{"id": id, "error": map[string]any{
					"code": -32601, "message": "Agent Command Center does not handle " + method}}); err != nil {
					s.problem("the reply to " + method + " was not sent: " + err.Error())
				}
			}
			break
		}
		for _, ev := range evs {
			if a, ok := ev.Payload.(event.Approval); ok && ev.Kind == event.ApprovalRequest && a.ID != "" {
				s.approvals[a.ID] = pendingReq{id: w.ID, unreadable: a.Problem != ""}
			}
		}
	case w.Method == "" && hasID:
		key := idString(w.ID)
		method := s.calls[key]
		delete(s.calls, key)
		if failed {
			break // the mapper shows the refusal as received
		}
		switch method {
		case "initialize":
			next = func() {
				if err := s.notify("initialized"); err != nil {
					s.problem("initialized not sent: " + err.Error())
					return
				}
				// Only the concurrency limit is read from the reply (reliability.md,
				// Telemetry sources: Concurrency limit; probe 2026-10-17, Codex CLI 0.163.0).
				if err := s.request("config/read", map[string]any{"cwd": s.dir, "includeLayers": false}); err != nil {
					s.problem("config/read not sent: " + err.Error())
				}
				s.startThread()
			}
		case "thread/start", "thread/resume":
			notice := ""
			if s.from != "" {
				notice = "Codex refused the model " + s.from + ": a new thread uses " + s.pin +
					", the default model/list reports. Send your message again."
				s.from = ""
			}
			next = func() {
				if notice != "" {
					s.push(event.Event{Kind: event.StatusChanged, Origin: event.Observed, Payload: event.Status{Status: "notice", Detail: notice}})
				}
				if err := s.Refresh(); err != nil {
					s.problem("account/rateLimits/read not sent: " + err.Error())
				}
			}
		case "model/list":
			model := defaultModel(w.Result)
			if model == "" {
				from := s.from
				s.from = ""
				next = func() {
					s.problem("Codex refused the model " + from + " and model/list names no default model: no new thread was started")
				}
				break
			}
			s.pin = model
			s.prov.setFallbackModel(model)
			next = s.startThread
		}
	}
	if s.fallback {
		s.fallback = false
		fall := func() {
			if err := s.request("model/list", map[string]any{}); err != nil {
				s.problem("model/list not sent: " + err.Error())
			}
		}
		if prev := next; prev != nil {
			next = func() { prev(); fall() }
		} else {
			next = fall
		}
	}
	return next
}

func approvalMethod(m string) bool {
	return m == "item/commandExecution/requestApproval" || m == "item/fileChange/requestApproval"
}

// defaultModel is the model model/list marks as default and does not hide.
func defaultModel(result json.RawMessage) string {
	var r struct {
		Data []struct {
			ID        string `json:"id"`
			Model     string `json:"model"`
			Hidden    bool   `json:"hidden"`
			IsDefault bool   `json:"isDefault"`
		} `json:"data"`
	}
	if json.Unmarshal(result, &r) != nil {
		return ""
	}
	for _, d := range r.Data {
		if d.IsDefault && !d.Hidden {
			return first(d.Model, d.ID)
		}
	}
	return ""
}

// startThread starts the main thread, or resumes the one the session context
// names, with the fallback model once there is one.
func (s *Session) startThread() {
	s.mu.Lock()
	p := threadParams{ThreadID: s.resume, Model: s.pin, Sandbox: Sandbox, ApprovalPolicy: ApprovalPolicy}
	s.resume = ""
	s.mu.Unlock()
	method := "thread/start"
	if p.ThreadID != "" {
		method = "thread/resume"
	}
	if err := s.request(method, p); err != nil {
		s.problem(method + " not sent: " + err.Error())
	}
}

func (s *Session) readStderr(r io.Reader) {
	br := bufio.NewReaderSize(r, 4<<10)
	for {
		l, size, over, err := pipe.ReadLine(br, maxStderrLine)
		if over {
			l = []byte(fmt.Sprintf("(a stderr line of %d bytes, not kept)", size))
		}
		if len(bytes.TrimSpace(l)) > 0 {
			s.push(event.Event{Kind: event.Stderr, Origin: event.Observed, Payload: event.Line{Text: string(l)}, Raw: string(l)})
		}
		if err != nil {
			return
		}
	}
}

// wait reports the exit as session.ended, the last event, once the readers
// are done or drainWait has passed (as for Claude Code).
func (s *Session) wait(out, errs <-chan struct{}) {
	code, err := s.p.Wait()
	close(s.exited)
	timer := time.NewTimer(drainWait)
	for out != nil || errs != nil {
		select {
		case <-out:
			out = nil
		case <-errs:
			errs = nil
		case <-timer.C:
			s.problem("Codex's output stayed open after its exit: a process it started still holds it")
			out, errs = nil, nil
		}
	}
	timer.Stop()
	s.p.Release()
	end := event.Ended{Code: code, HasCode: code >= 0}
	switch {
	case err != nil:
		end.Reason = "Codex ended: " + err.Error()
	case s.closing.Load():
		end.Reason = "Codex closed (exit code " + strconv.Itoa(code) + ")"
	default:
		end.Reason = "Codex exited with code " + strconv.Itoa(code)
	}
	s.push(event.Event{Kind: event.SessionEnded, Origin: event.Observed, Payload: end})
	s.q.Close()
}

// Events are the session's events; the channel closes after session.ended.
func (s *Session) Events() <-chan event.Event { return s.q.Out() }

// write sends one line. Its events are mapped before the write, so a reply
// can never be mapped first, and released only once the line is written.
func (s *Session) write(msg any) error {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(msg); err != nil {
		return err
	}
	s.wmu.Lock()
	defer s.wmu.Unlock()
	if s.inClosed.Load() {
		return errors.New("Codex's input is closed")
	}
	s.mu.Lock()
	evs := s.m.In(bytes.TrimRight(b.Bytes(), "\n"))
	for _, ev := range evs {
		s.track(ev)
	}
	held := s.q.Hold(evs)
	s.mu.Unlock()
	_, err := s.stdin.Write(b.Bytes())
	s.q.Release(held, err == nil)
	if err != nil {
		return fmt.Errorf("not delivered to Codex: %w", err)
	}
	return nil
}

// request sends one request; params nil sends none.
func (s *Session) request(method string, params any) error {
	s.mu.Lock()
	s.ids++
	id := s.ids
	s.calls[strconv.Itoa(id)] = method
	s.mu.Unlock()
	msg := map[string]any{"id": id, "method": method}
	if params != nil {
		msg["params"] = params
	}
	return s.write(msg)
}

func (s *Session) notify(method string) error {
	return s.write(map[string]any{"method": method})
}

// Send starts a turn on the main thread with the owner's message, byte for
// byte.
func (s *Session) Send(text string) error {
	s.mu.Lock()
	thread := s.thread
	s.mu.Unlock()
	if thread == "" {
		return errors.New("no Codex thread to send to yet")
	}
	return s.request("turn/start", map[string]any{"threadId": thread,
		"input": []map[string]string{{"type": "text", "text": text}}})
}

// Answer answers a pending approval with one of Codex's decisions, echoing
// the request's id as received; Claude Code's allow and deny are refused.
func (s *Session) Answer(id, decision string) error {
	if !answers[decision] {
		return fmt.Errorf("Codex takes no answer %q", decision)
	}
	s.mu.Lock()
	req, ok := s.approvals[id]
	s.mu.Unlock()
	if !ok {
		return fmt.Errorf("no pending request %q", id)
	}
	if req.unreadable && (decision == "accept" || decision == "acceptForSession") {
		return fmt.Errorf("request %q was not readable: only a decline answers it", id)
	}
	return s.write(map[string]any{"id": req.id, "result": map[string]string{"decision": decision}})
}

// Interrupt asks Codex to end the running turn.
func (s *Session) Interrupt() error {
	s.mu.Lock()
	thread, turn := s.thread, s.turn
	s.mu.Unlock()
	if thread == "" || turn == "" {
		return errors.New("no running Codex turn")
	}
	return s.request("turn/interrupt", map[string]any{"threadId": thread, "turnId": turn})
}

// Refresh asks for the rate limits, which need no turn (TEL CX; P6).
func (s *Session) Refresh() error { return s.request("account/rateLimits/read", nil) }

// Capabilities of Codex CLI 0.145 through app-server. Allow for the session
// is decided per request by the mapper: offered unless the request's
// availableDecisions omits acceptForSession (plan 5, item 8; plan 8).
func (s *Session) Capabilities() provider.Capabilities {
	return provider.Capabilities{SubagentText: true, QuotaWithoutTurn: true}
}

// Close closes stdin, then terminates the CLI, then kills its tree, each
// after CloseStep; it returns when the process has exited or ctx ends, and
// the escalation goes on in the background.
func (s *Session) Close(ctx context.Context) error {
	s.once.Do(func() {
		s.closing.Store(true)
		go s.escalate()
	})
	select {
	case <-s.exited:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *Session) escalate() {
	// A write blocked on a full pipe holds the file: closing it waits for the
	// write, so it runs apart from the escalation.
	go func() { s.inClosed.Store(true); _ = s.stdin.Close() }()
	if s.waitExit() {
		return
	}
	_ = s.p.Terminate()
	if s.waitExit() {
		return
	}
	_ = s.p.Kill()
}

func (s *Session) waitExit() bool {
	t := time.NewTimer(s.step)
	defer t.Stop()
	select {
	case <-s.exited:
		return true
	case <-t.C:
		return false
	}
}
