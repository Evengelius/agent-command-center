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

	mu     sync.Mutex        // the mapper and the order of events
	m      *Mapper           //
	inputs map[string][]byte // pending approval id -> tool input
	reqs   int               // control requests sent

	wmu      sync.Mutex // one write at a time
	inClosed atomic.Bool

	q       *queue
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
	s := &Session{p: pr, stdin: pr.Stdin(), step: step, m: NewMapper(), inputs: map[string][]byte{},
		q: newQueue(now), exited: make(chan struct{})}
	s.q.push(event.Event{Provider: ProviderID, Kind: event.SessionSpawned, Origin: event.Observed,
		Payload: event.Spawned{Argv: append([]string{sc.Executable}, argv...), Dir: sc.ProjectPath}})
	go s.q.forward()

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
	s.q.push(ev)
	s.mu.Unlock()
}

// read maps stdout line by line until end of file.
func (s *Session) read(r io.Reader) {
	br := bufio.NewReaderSize(r, 64<<10)
	for {
		l, size, over, err := readLine(br, MaxLine)
		switch {
		case over:
			s.push(event.Event{Kind: event.Error, Origin: event.Observed, Payload: event.Problem{
				Message: fmt.Sprintf("a line of %d bytes from Claude Code was skipped (limit %d MiB)", size, MaxLine>>20)}})
		case len(bytes.TrimSpace(l)) > 0:
			s.mu.Lock()
			evs := s.m.Out(l)
			for _, ev := range evs {
				s.track(ev)
				s.q.push(ev)
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

// track keeps each pending approval's input, which an allow echoes back.
func (s *Session) track(ev event.Event) {
	a, ok := ev.Payload.(event.Approval)
	if !ok {
		return
	}
	switch ev.Kind {
	case event.ApprovalRequest, event.Question:
		if a.Problem == "" {
			s.inputs[a.ID] = a.Input
		}
	case event.ApprovalResolved:
		delete(s.inputs, a.ID)
	}
}

func (s *Session) readStderr(r io.Reader) {
	br := bufio.NewReaderSize(r, 4<<10)
	for {
		l, size, over, err := readLine(br, maxStderrLine)
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

// readLine reads one line without its end of line. A line longer than limit
// is read to its end and dropped: over is true and size is its length.
func readLine(br *bufio.Reader, limit int) (l []byte, size int, over bool, err error) {
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
	s.q.close()
}

// Events are the session's events; the channel closes after session.ended.
func (s *Session) Events() <-chan event.Event { return s.q.out }

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
	held := s.q.hold(evs)
	s.mu.Unlock()
	_, err := s.stdin.Write(b.Bytes())
	s.q.release(held, err == nil)
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

// Answer answers a pending approval: allow echoes the tool's input
// unchanged, anything else denies.
func (s *Session) Answer(id, decision string) error {
	s.mu.Lock()
	input, ok := s.inputs[id]
	s.mu.Unlock()
	if !ok {
		return fmt.Errorf("no pending request %q", id)
	}
	resp := map[string]any{"behavior": "deny", "message": DenyMessage}
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
	return provider.Capabilities{SubagentText: true, APICost: true, Questions: true, ResumeHistory: true}
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

// queue is unbounded, so the readers never wait for the screen, and keeps
// the order events were mapped in. A written line's events are held until
// the write ends: released if it succeeded, dropped if not. Seq is set when
// an event leaves, so a dropped one leaves no gap.
type queue struct {
	mu     sync.Mutex
	cond   *sync.Cond
	items  []*item
	closed bool
	now    func() time.Time
	out    chan event.Event
	seq    uint64
}

type item struct {
	ev   event.Event
	held bool
	drop bool
}

func newQueue(now func() time.Time) *queue {
	q := &queue{now: now, out: make(chan event.Event, 256)}
	q.cond = sync.NewCond(&q.mu)
	return q
}

func (q *queue) push(ev event.Event) {
	q.mu.Lock()
	if q.closed { // after session.ended
		q.mu.Unlock()
		return
	}
	ev.At = q.now()
	q.items = append(q.items, &item{ev: ev})
	q.mu.Unlock()
	q.cond.Signal()
}

func (q *queue) hold(evs []event.Event) []*item {
	q.mu.Lock()
	defer q.mu.Unlock()
	held := make([]*item, len(evs))
	for i, ev := range evs {
		ev.At = q.now()
		held[i] = &item{ev: ev, held: true}
		if !q.closed {
			q.items = append(q.items, held[i])
		}
	}
	return held
}

func (q *queue) release(held []*item, keep bool) {
	q.mu.Lock()
	for _, it := range held {
		it.held, it.drop = false, !keep
	}
	q.mu.Unlock()
	q.cond.Signal()
}

func (q *queue) close() {
	q.mu.Lock()
	q.closed = true
	q.mu.Unlock()
	q.cond.Signal()
}

// forward sends the events in order and closes the channel after the last.
func (q *queue) forward() {
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
