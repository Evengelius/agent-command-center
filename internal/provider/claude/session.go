package claude

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

// GitBashDefault is where Git for Windows installs its bash.
const GitBashDefault = `C:\Program Files\Git\bin\bash.exe`

// args is way A: one long-lived process, stream-json in and out, approvals
// over stdio (architecture.md, Claude Code adapter). Never --bare, never an
// empty system prompt: the owner's own setup applies.
func args(resume string, extra []string) []string {
	a := []string{"-p", "--input-format", "stream-json", "--output-format", "stream-json",
		"--verbose", "--include-partial-messages", "--forward-subagent-text",
		"--permission-prompt-tool", "stdio", "--permission-mode", "default"}
	if resume != "" {
		a = append(a, "--resume", resume)
	}
	return append(a, extra...)
}

// childEnv is the dashboard's environment with session state events on, and
// Git Bash named when the PATH's bash is not Git's (WSL's, P3b). A variable
// the owner set is kept.
func childEnv(base []string, h provider.Host) []string {
	env := append([]string(nil), base...)
	has := func(name string) bool {
		for _, kv := range env {
			if k, _, ok := strings.Cut(kv, "="); ok && strings.EqualFold(k, name) {
				return true
			}
		}
		return false
	}
	if !has("CLAUDE_CODE_EMIT_SESSION_STATE_EVENTS") {
		env = append(env, "CLAUDE_CODE_EMIT_SESSION_STATE_EVENTS=1")
	}
	if h.Windows && !has("CLAUDE_CODE_GIT_BASH_PATH") && provider.IsFile(GitBashDefault) &&
		!strings.EqualFold(h.Find("bash.exe"), GitBashDefault) {
		env = append(env, "CLAUDE_CODE_GIT_BASH_PATH="+GitBashDefault)
	}
	return env
}

// Session is one running `claude -p` process.
type Session struct {
	p     proc.Process
	stdin io.WriteCloser
	step  time.Duration

	mu     sync.Mutex            // the mapper and the order of events
	m      *Mapper               //
	inputs map[string]pendingReq // pending approval id -> its request
	reqs   int                   // control requests sent

	wmu      sync.Mutex // one write at a time
	inClosed atomic.Bool

	q       *pipe.Queue
	exited  chan struct{}
	closing atomic.Bool
	once    sync.Once
}

func (p *Provider) start(ctx context.Context, sc provider.SessionContext, env []string) (*Session, error) {
	sp := p.Spawner
	if sp == nil {
		sp = proc.Exec{}
	}
	argv := args(sc.ResumeID, p.Args)
	pr, err := sp.Spawn(proc.Spec{Path: sc.Executable, Args: argv, Dir: sc.ProjectPath, Env: env})
	if err != nil {
		return nil, fmt.Errorf("Claude Code did not start: %w", err)
	}
	step := p.CloseStep
	if step == 0 {
		step = CloseStep
	}
	now := p.Now
	if now == nil {
		now = time.Now
	}
	s := &Session{p: pr, stdin: pr.Stdin(), step: step, m: NewMapper(), inputs: map[string]pendingReq{},
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
	if err := s.control("initialize", `"hooks":null`); err != nil {
		s.push(event.Event{Kind: event.Error, Origin: event.Observed, Payload: event.Problem{Message: "initialize not sent: " + err.Error()}})
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

// read maps stdout line by line until end of file.
func (s *Session) read(r io.Reader) {
	br := bufio.NewReaderSize(r, 64<<10)
	for {
		l, size, over, err := pipe.ReadLine(br, MaxLine)
		switch {
		case over:
			s.push(event.Event{Kind: event.Error, Origin: event.Observed, Payload: event.Problem{
				Message: fmt.Sprintf("a line of %d bytes from Claude Code was skipped (limit %d MiB)", size, MaxLine>>20)}})
		case len(bytes.TrimSpace(l)) > 0:
			s.mu.Lock()
			evs := s.m.Out(l)
			for _, ev := range evs {
				s.track(ev)
				s.q.Push(ev)
			}
			s.mu.Unlock()
		}
		if err != nil {
			if !errors.Is(err, io.EOF) && !s.closing.Load() {
				s.push(event.Event{Kind: event.Error, Origin: event.Observed, Payload: event.Problem{Message: "reading Claude Code's output: " + err.Error()}})
			}
			return
		}
	}
}

// pendingReq is a pending approval: its input, which an allow echoes back,
// whether the request could not be read, which only a deny answers (INT 4
// fallback), and whether it is a question (AskUserQuestion).
type pendingReq struct {
	input      []byte
	unreadable bool
	question   bool
}

// track keeps each pending approval's request until it is answered.
func (s *Session) track(ev event.Event) {
	a, ok := ev.Payload.(event.Approval)
	if !ok {
		return
	}
	switch ev.Kind {
	case event.ApprovalRequest, event.Question:
		if a.ID != "" {
			s.inputs[a.ID] = pendingReq{input: a.Input, unreadable: a.Problem != "", question: ev.Kind == event.Question}
		}
	case event.ApprovalResolved:
		delete(s.inputs, a.ID)
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
// are done. A process the CLI started and the job object did not end could
// hold the pipes open; a blocked read on Windows does not return when its
// file is closed, so after drainWait the session ends without the readers,
// saying so, and anything they read later is dropped.
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
			s.push(event.Event{Kind: event.Error, Origin: event.Observed, Payload: event.Problem{
				Message: "Claude Code's output stayed open after its exit: a process it started still holds it"}})
			out, errs = nil, nil
		}
	}
	timer.Stop()
	s.p.Release()
	end := event.Ended{Code: code, HasCode: code >= 0}
	switch {
	case err != nil:
		end.Reason = "Claude Code ended: " + err.Error()
	case s.closing.Load():
		end.Reason = "Claude Code closed (exit code " + strconv.Itoa(code) + ")"
	default:
		end.Reason = "Claude Code exited with code " + strconv.Itoa(code)
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
		return errors.New("Claude Code's input is closed")
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
		return fmt.Errorf("not delivered to Claude Code: %w", err)
	}
	return nil
}

// control sends one control request; body is its fields after the subtype.
func (s *Session) control(subtype, body string) error {
	s.mu.Lock()
	s.reqs++
	id := "req_" + strconv.Itoa(s.reqs)
	s.mu.Unlock()
	req := `{"subtype":"` + subtype + `"`
	if body != "" {
		req += "," + body
	}
	return s.write(map[string]any{"type": "control_request", "request_id": id, "request": json.RawMessage(req + "}")})
}

// Send writes the owner's message, byte for byte.
func (s *Session) Send(text string) error {
	return s.write(map[string]any{
		"type":               "user",
		"message":            map[string]any{"role": "user", "content": text},
		"parent_tool_use_id": nil,
		"session_id":         "default",
	})
}

// DenyMessage is what Claude Code is told when the owner denies a request.
const DenyMessage = "The user denied this in Agent Command Center."

// QuestionDenyMessage denies a question the owner saw: the answer comes as
// their next message, since the updatedInput round trip is unprobed (CC-10
// q3; plan 5, item 10).
const QuestionDenyMessage = "The user saw these questions in Agent Command Center, which cannot answer them here, and will answer them in their next message. Wait for that message; do not ask them again."

// Answer answers a pending approval: allow echoes the tool's input
// unchanged, anything else denies; a readable question is denied with
// QuestionDenyMessage, and is answered by AnswerQuestion, never by an allow.
func (s *Session) Answer(id, decision string) error {
	s.mu.Lock()
	req, ok := s.inputs[id]
	s.mu.Unlock()
	if !ok {
		return fmt.Errorf("no pending request %q", id)
	}
	if decision == "allow" && req.question {
		return fmt.Errorf("request %q is a question: it is answered with AnswerQuestion", id)
	}
	if decision == "allow" && req.unreadable {
		return fmt.Errorf("request %q was not readable: only a deny answers it", id)
	}
	input := req.input
	msg := DenyMessage
	if decision == "deny" && req.question && !req.unreadable { // not a cancel; an unreadable question was never shown
		msg = QuestionDenyMessage
	}
	resp := map[string]any{"behavior": "deny", "message": msg}
	if decision == "allow" {
		if len(bytes.TrimSpace(input)) == 0 || string(input) == "null" {
			input = []byte("{}")
		}
		resp = map[string]any{"behavior": "allow", "updatedInput": json.RawMessage(input)}
	}
	return s.write(map[string]any{"type": "control_response",
		"response": map[string]any{"subtype": "success", "request_id": id, "response": resp}})
}

// Interrupt asks Claude Code to end the running turn.
func (s *Session) Interrupt() error { return s.control("interrupt", "") }

// Refresh asks for the context usage (undocumented, INT 4).
func (s *Session) Refresh() error { return s.control("get_context_usage", "") }

// Capabilities of Claude Code 2.1 in way A.
func (s *Session) Capabilities() provider.Capabilities {
	return provider.Capabilities{SubagentText: true, APICost: true, Questions: true, ResumeHistory: true, AutoApprovals: true}
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

// SetApprovalMode asks Claude Code for the auto permission mode or, for
// Manual, the default one, at once; the response says which is in force
// (plan 12, item 2; P12c-switch).
func (s *Session) SetApprovalMode(auto bool) error {
	mode := "default"
	if auto {
		mode = "auto"
	}
	return s.control("set_permission_mode", `"mode":"`+mode+`"`)
}

// AnswerQuestion answers a pending, readable question request: it allows it
// with the request's input, every key unchanged, and an "answers" key (question
// text to answer) added; Claude Code then returns the answers to the model as
// the tool's result (plan 14, item 1; probe P14a).
func (s *Session) AnswerQuestion(id string, answers map[string]string) error {
	s.mu.Lock()
	req, ok := s.inputs[id]
	s.mu.Unlock()
	switch {
	case !ok:
		return fmt.Errorf("no pending request %q", id)
	case !req.question:
		return fmt.Errorf("request %q is not a question", id)
	case req.unreadable:
		return fmt.Errorf("request %q was not readable: only a deny answers it", id)
	}
	var in map[string]json.RawMessage
	if err := json.Unmarshal(req.input, &in); err != nil || in == nil {
		return fmt.Errorf("the questions of request %q are not readable", id)
	}
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(answers); err != nil {
		return err
	}
	in["answers"] = json.RawMessage(bytes.TrimRight(b.Bytes(), "\n"))
	resp := map[string]any{"behavior": "allow", "updatedInput": in}
	return s.write(map[string]any{"type": "control_response",
		"response": map[string]any{"subtype": "success", "request_id": id, "response": resp}})
}
