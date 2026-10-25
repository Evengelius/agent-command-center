package codex

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
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
// through the dashboard (architecture.md, Codex adapter). Sandbox is the
// thread's at start and resume; each turn/start then names its own
// (turnSandbox), a writable one only in Auto.
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
	// ApprovalsReviewer is "user" (Manual) or "auto_review" (Auto), sent with
	// every thread/start, thread/resume and turn/start (plan 12, item 2).
	ApprovalsReviewer string `json:"approvalsReviewer"`
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
	mode      string                // the mode turn/start carries (modePresets): manual, auto or full; "" is manual
	refused   bool                  // the running turn failed with the model refused
	fellBack  bool                  // the model fallback ran: never twice
	from      string                // the refused model, until the new thread starts
	fallback  bool                  // the fallback is due once the line's events are queued
	dir       string                // the project directory, the cwd of config/read
	change    provider.ModelRequest // the model and effort asked for and not yet answered: the next turn/start carries them (plan 34)
	// Plan 38 (models.go): instant while Codex takes thread/settings/update,
	// a change between turns then written at once; sent once the change is
	// written so and waits for its thread/settings/updated; starting from
	// the main turn/start written to its turn's end; flush when that turn
	// ended with a change waiting; inModel and inEffort what Codex last
	// reported in force.
	instant           bool
	sent              bool
	starting          bool
	flush             bool
	inModel, inEffort string
	now               func() time.Time
	// Plan 39: a mode asked for and not yet in force, written at once or at
	// the main turn's end as the model change is (modeSent once written).
	modeAsk  string
	modeSent bool

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

// proxyEnv is base, the dashboard's environment, with the proxy given to the
// CLI (plan 22, amendment 19): every HTTPS_PROXY and HTTP_PROXY entry, in any
// case, is dropped, then HTTPS_PROXY=proxy and HTTP_PROXY=proxy are appended.
// Every other entry stays as it is and in its place, so the commands the CLI
// runs inherit the same variables. base is not changed.
func proxyEnv(base []string, proxy string) []string {
	env := make([]string, 0, len(base)+2)
	for _, kv := range base {
		if name, _, _ := strings.Cut(kv, "="); strings.EqualFold(name, "HTTPS_PROXY") || strings.EqualFold(name, "HTTP_PROXY") {
			continue
		}
		env = append(env, kv)
	}
	return append(env, "HTTPS_PROXY="+proxy, "HTTP_PROXY="+proxy)
}

func (p *Provider) start(ctx context.Context, sc provider.SessionContext) (*Session, error) {
	sp := p.Spawner
	if sp == nil {
		sp = proc.Exec{}
	}
	argv := append([]string{"app-server"}, p.Args...)
	spec := proc.Spec{Path: sc.Executable, Args: argv, Dir: sc.ProjectPath}
	if p.Proxy != "" {
		// A start and a resume after a crash alike: the same provider, the same
		// environment. Without a proxy Env stays nil and the CLI inherits the
		// dashboard's, as before.
		spec.Env = proxyEnv(os.Environ(), p.Proxy)
	}
	pr, err := sp.Spawn(spec)
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
		approvals: map[string]pendingReq{}, resume: sc.ResumeID, pin: p.fallbackModel(), instant: true, now: now,
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
	// Codex's experimental API, for thread/settings/update (plan 38, design
	// D1), asked for as OpenAI's Codex extension for VS Code asks for it.
	if err := s.request("initialize", map[string]any{"clientInfo": clientInfo, "capabilities": initCapabilities}); err != nil {
		s.problem("initialize not sent: " + err.Error())
	}
	return s, nil
}

// initCapabilities are the capabilities initialize asks for: Codex's
// experimental API, which marks itself experimental (plan 38).
var initCapabilities = map[string]bool{"experimentalApi": true}

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
// A panic in the mapping is the session's fatal error, and the rest of stdout
// is read and dropped, so Codex never blocks on it before the dashboard
// closes it (plan 48, design D5; audit finding 6).
func (s *Session) read(r io.Reader) {
	br := bufio.NewReaderSize(r, 64<<10)
	defer func() {
		if v := recover(); v != nil {
			s.fatal("reading Codex's output failed", v)
			_, _ = io.Copy(io.Discard, br)
		}
	}()
	for {
		l, size, over, err := pipe.ReadLine(br, MaxLine)
		switch {
		case over:
			s.problem(fmt.Sprintf("a line of %d bytes from Codex was skipped (limit %d MiB)", size, MaxLine>>20))
		case len(bytes.TrimSpace(l)) > 0:
			if next := s.mapOut(l); next != nil {
				s.apart("replying to Codex", next)
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
		if p.TurnRefused {
			s.starting = false // no turn started
		}
	case event.Settings:
		s.settings(p)
	case event.Mode:
		if ev.Agent == "" {
			s.approvalMode(p)
		}
	}
	switch {
	case ev.Kind == event.TurnStarted && ev.Agent == "" && ev.Turn != "":
		s.turn = ev.Turn
	case ev.Kind == event.TurnCompleted && ev.Agent == "":
		s.turn, s.starting = "", false
		if s.refused && !s.fellBack && s.pin == "" {
			s.fellBack, s.from, s.thread, s.fallback = true, s.model, "", true
		}
		s.refused = false
		// A change asked for during the turn is written now (plan 38,
		// design D3); after a refused model it waits for the new thread.
		s.flush = s.instant && s.thread != "" && (s.change != (provider.ModelRequest{}) && !s.sent || s.modeAsk != "" && !s.modeSent)
	}
}

// settings follows what Codex reports of the model settings (plan 38): the
// values in force; a change answered, or refused, no longer waits; a
// refused thread/settings/update leaves every change to the next turn/start
// (design D4). Called with mu held.
func (s *Session) settings(p event.Settings) {
	if p.Model != "" {
		s.inModel = p.Model
	}
	if p.EffortKnown {
		s.inEffort = p.Effort
	}
	switch {
	case p.NextTurn:
		s.instant, s.sent = false, false
	case p.Source == "thread/settings/updated" && p.Answers, p.Source == "thread/settings/update" && p.Refused:
		s.change, s.sent = provider.ModelRequest{}, false
	}
}

// approvalMode follows what Codex reports of the mode (plan 39): one of the
// three in force is what turn/start carries from now on, a change answered
// or refused no longer waits, and a refused thread/settings/update leaves the
// mode asked for to the next turn/start. Called with mu held.
func (s *Session) approvalMode(p event.Mode) {
	switch {
	case p.NextTurn:
		s.instant = false
		if s.modeAsk != "" {
			s.mode, s.modeAsk, s.modeSent = s.modeAsk, "", false
		}
	case p.Refused:
		s.modeAsk, s.modeSent = "", false
	case p.Answers || p.Mode != "":
		if _, ok := modePresets[p.Mode]; ok {
			s.mode = p.Mode
		}
		if p.Answers {
			s.modeAsk, s.modeSent = "", false
		}
	}
}

// busy says the main turn runs: from its turn/start written to its end.
// Called with mu held.
func (s *Session) busy() bool { return s.starting || s.turn != "" }

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
	case w.Method == "currentTime/read" && hasID:
		// The time in whole Unix seconds, as OpenAI's Codex extension for VS
		// Code answers it (plan 38, design D5).
		id, at := w.ID, s.now().Unix()
		next = func() {
			if err := s.write(map[string]any{"id": id, "result": map[string]int64{"currentTimeAt": at}}); err != nil {
				s.problem("the reply to currentTime/read was not sent: " + err.Error())
			}
		}
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
				// The managed requirements, for the modes they leave out (plan
				// 39, design D1; P39x).
				if err := s.request("configRequirements/read", nil); err != nil {
					s.problem("configRequirements/read not sent: " + err.Error())
				}
				// The session's models, every page (plan 34, P34x).
				if err := s.request("model/list", map[string]any{}); err != nil {
					s.problem("model/list not sent: " + err.Error())
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
			if cursor := s.m.NextPage(); cursor != "" {
				// The next page (plan 34): the mapper keeps the list and stops
				// a cursor that repeats.
				next = func() {
					if err := s.request("model/list", map[string]any{"cursor": cursor}); err != nil {
						s.problem("model/list not sent: " + err.Error())
					}
				}
				break
			}
			if s.from == "" {
				break // the session's models: no refused model waits for the list
			}
			model := s.m.DefaultModel()
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
	if s.flush {
		// The main turn ended with a change asked for during it: written now
		// (plan 38, design D3).
		s.flush = false
		thread, change, mode := s.thread, provider.ModelRequest{}, ""
		if s.change != (provider.ModelRequest{}) && !s.sent {
			change, s.sent = s.change, true
		}
		if s.modeAsk != "" && !s.modeSent {
			mode, s.modeSent = s.modeAsk, true // and a mode asked for during it (plan 39)
		}
		up := func() {
			if err := s.writeUpdate(thread, change, mode); err != nil {
				s.problem("thread/settings/update not sent, the change waits for the next message: " + err.Error())
			}
		}
		if prev := next; prev != nil {
			next = func() { prev(); up() }
		} else {
			next = up
		}
	}
	if s.m.TakeReadDue() {
		// A turn took a model or effort change, or was rerouted: thread/read
		// says what the thread is configured with now (plan 34).
		read := func() {
			s.mu.Lock()
			thread := s.thread
			s.mu.Unlock()
			if thread == "" {
				return
			}
			if err := s.request("thread/read", map[string]any{"threadId": thread}); err != nil {
				s.problem("thread/read not sent: " + err.Error())
			}
		}
		if prev := next; prev != nil {
			next = func() { prev(); read() }
		} else {
			next = read
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

// startThread starts the main thread, or resumes the one the session context
// names, with the fallback model once there is one.
func (s *Session) startThread() {
	s.mu.Lock()
	p := threadParams{ThreadID: s.resume, Model: s.pin, Sandbox: Sandbox, ApprovalPolicy: ApprovalPolicy, ApprovalsReviewer: presetOf(s.mode).reviewer}
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
	held := s.holdIn(bytes.TrimRight(b.Bytes(), "\n"))
	_, err := s.stdin.Write(b.Bytes())
	s.q.Release(held, err == nil)
	if err != nil {
		return fmt.Errorf("not delivered to Codex: %w", err)
	}
	return nil
}

// holdIn maps a line the dashboard writes under the lock, released however
// the mapping ends (plan 48, design D5); its events are held until the line
// is written.
func (s *Session) holdIn(l []byte) []*pipe.Item {
	s.mu.Lock()
	defer s.mu.Unlock()
	evs := s.m.In(l)
	for _, ev := range evs {
		s.track(ev)
	}
	return s.q.Hold(evs)
}

// mapOut maps one line of stdout under the lock, released however the
// mapping ends (plan 48, design D5), queues its events, and returns what the
// line calls for, to be written apart.
func (s *Session) mapOut(l []byte) func() {
	s.mu.Lock()
	defer s.mu.Unlock()
	evs := s.m.Out(l)
	for _, ev := range evs {
		s.track(ev)
		s.q.Push(ev)
	}
	return s.react(l, evs)
}

// apart runs f in a goroutine of its own: the reader never waits on stdin. A
// panic in f is the session's fatal error, never the end of the program
// (plan 48, design D5).
func (s *Session) apart(what string, f func()) {
	go func() {
		defer func() {
			if v := recover(); v != nil {
				s.fatal(what+" failed", v)
			}
		}()
		f()
	}()
}

// fatal queues a recovered panic as the session's fatal error: the dashboard
// shows it as it ends the session and closes Codex (plan 48, design D5).
func (s *Session) fatal(what string, v any) {
	s.push(event.Event{Kind: event.Error, Origin: event.Observed, Payload: event.Problem{Message: what + ": " + provider.Recovered(v), Fatal: true}})
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

// turnSummary is the reasoning summary every turn/start asks for
// (TurnStartParams.summary, "auto"). Asked for, a reasoning item starts while
// the model works, 2.5 to 3.4 s before it completes in P22s (0 to 10 ms
// without), and it carries Codex's summary when Codex sends one (plan 22,
// amendment 20; codex-cli.md, section 12). Only turn/start carries it:
// thread/start and thread/resume are unchanged.
const turnSummary = "auto"

// imageLimits are Codex's limits on a message's images
// (doc/research/image-input.md): the OpenAI API's 30,000 patches of 32 px an
// image, rejected beyond; the dashboard's own 20 MB an image, 50 MB and 100
// images a message, well under the API's 512 MB and 1,500 a request.
var imageLimits = provider.ImageLimits{MaxCount: 100, MaxBytes: 20_000_000, MaxTotal: 50_000_000, MaxPatches: 30_000, PatchSize: 32}

// Send starts a turn on the main thread with the owner's message: its
// images as `image` inputs holding data URLs, then its text byte for byte
// (UserInput, the app-server v2 schema; plan 31).
func (s *Session) Send(msg provider.Message) error {
	s.mu.Lock()
	thread, change := s.thread, s.change
	// The change goes with this turn, also one written at once whose
	// thread/settings/updated has not come: never left waiting (plan 38,
	// design D2). From here to the turn's end, changes wait for it (D3). So
	// does a mode asked for and not yet in force (plan 39).
	s.change, s.sent, s.starting = provider.ModelRequest{}, false, thread != ""
	mode, asked := s.mode, s.modeAsk
	if asked != "" {
		mode, s.modeAsk, s.modeSent = asked, "", false
	}
	preset := presetOf(mode)
	s.mu.Unlock()
	if thread == "" {
		s.keepChange(change)
		s.keepMode(asked)
		return errors.New("no Codex thread to send to yet")
	}
	input := make([]map[string]string, 0, len(msg.Images)+1)
	for _, img := range msg.Images {
		input = append(input, map[string]string{"type": "image", "url": "data:" + img.Media + ";base64," + base64.StdEncoding.EncodeToString(img.Data)})
	}
	if msg.Text != "" || len(msg.Images) == 0 {
		input = append(input, map[string]string{"type": "text", "text": msg.Text})
	}
	params := map[string]any{"threadId": thread,
		"input": input, "approvalsReviewer": preset.reviewer, "approvalPolicy": preset.policy,
		"sandboxPolicy": map[string]string{"type": preset.sandbox}, "summary": turnSummary}
	// A model or effort change asked for since the last turn goes with this
	// one, "for this turn and subsequent turns" (plan 34).
	if change.Model != "" {
		params["model"] = change.Model
	}
	if change.Effort != "" {
		params["effort"] = change.Effort
	}
	err := s.request("turn/start", params)
	if err != nil {
		s.mu.Lock()
		s.starting = false
		s.mu.Unlock()
		s.keepChange(change)
		s.keepMode(asked)
	}
	return err
}

// keepMode puts back a mode a turn/start did not carry, unless another was
// asked for since (plan 39).
func (s *Session) keepMode(m string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.modeAsk == "" {
		s.modeAsk = m
	}
}

// keepChange puts back a change a turn/start did not carry, unless another
// was asked for since.
func (s *Session) keepChange(c provider.ModelRequest) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.change == (provider.ModelRequest{}) {
		s.change = c
	}
}

// SetModelSettings takes the change asked for, whole (plan 38; doc/research/
// codex-cli.md, section 13): between turns it writes it at once with
// thread/settings/update, answered by the thread/settings/updated that
// follows (design D2); during a turn it waits for the turn's end (D3); with
// a Codex that refused the update, it goes with the next turn/start, which
// is when Codex takes it (plan 34; D4). A change that asks only for what
// Codex reports in force cancels the one waiting. Codex has no Ultracode.
func (s *Session) SetModelSettings(r provider.ModelRequest) error {
	if r.Ultracode != nil {
		return errors.New("Codex has no Ultracode")
	}
	s.mu.Lock()
	if (r.Model == "" || r.Model == s.inModel) && (r.Effort == "" || r.Effort == s.inEffort) {
		r = provider.ModelRequest{} // Codex notifies no update that changes nothing (P38x)
	}
	s.change = r
	if !s.instant || s.busy() || s.thread == "" || r == (provider.ModelRequest{}) {
		s.mu.Unlock()
		return nil
	}
	thread := s.thread
	s.sent = true
	s.mu.Unlock()
	if err := s.writeUpdate(thread, r, ""); err != nil {
		s.mu.Lock()
		s.change = provider.ModelRequest{}
		s.mu.Unlock()
		return err
	}
	return nil
}

// settingsUpdate is thread/settings/update's params (experimental): the
// thread and only what the change asks for.
type settingsUpdate struct {
	ThreadID string `json:"threadId"`
	Model    string `json:"model,omitempty"`
	Effort   string `json:"effort,omitempty"`
	// A mode's three settings (plan 39), all or none.
	SandboxPolicy     *sandboxPolicy `json:"sandboxPolicy,omitempty"`
	ApprovalPolicy    string         `json:"approvalPolicy,omitempty"`
	ApprovalsReviewer string         `json:"approvalsReviewer,omitempty"`
}

// sandboxPolicy is a SandboxPolicy as the dashboard writes it: its type
// alone.
type sandboxPolicy struct {
	Type string `json:"type"`
}

// writeUpdate writes thread/settings/update for the change; one not written
// waits for the next turn/start instead.
func (s *Session) writeUpdate(thread string, c provider.ModelRequest, mode string) error {
	u := settingsUpdate{ThreadID: thread, Model: c.Model, Effort: c.Effort}
	if mode != "" {
		p := presetOf(mode)
		u.SandboxPolicy, u.ApprovalPolicy, u.ApprovalsReviewer = &sandboxPolicy{Type: p.sandbox}, p.policy, p.reviewer
	}
	err := s.request("thread/settings/update", u)
	if err != nil {
		s.mu.Lock()
		if c != (provider.ModelRequest{}) {
			s.sent = false
		}
		if mode != "" {
			s.modeSent = false
		}
		s.mu.Unlock()
	}
	return err
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
		return fmt.Errorf("%w %q", provider.ErrNoPending, id)
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
// availableDecisions omits acceptForSession (plan 5, item 8; plan 8). A
// model change applies at once between turns, from the next turn while the
// main turn runs or for good once Codex refused thread/settings/update
// (plan 38, designs D3 and D4).
func (s *Session) Capabilities() provider.Capabilities {
	s.mu.Lock()
	next := !s.instant || s.busy()
	s.mu.Unlock()
	return provider.Capabilities{SubagentText: true, QuotaWithoutTurn: true, Modes: true, Images: imageLimits,
		Models: true, ModelsNextTurn: next}
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

// The approvals reviewers Codex takes: the owner, or Codex's auto-review
// (plan 12, item 2; P12x).
const (
	reviewerUser = "user"
	reviewerAuto = "auto_review"
)

// modePreset is the settings that make one of Codex's modes: its sandbox,
// its approval policy and who reviews approvals.
type modePreset struct{ sandbox, policy, reviewer string }

// modePresets are Codex's three modes of the modes selector, as OpenAI's
// Codex extension for VS Code sets them (plan 39, design D1; P39x): Ask for
// approval the read-only sandbox the dashboard starts with (plan 12's
// Manual), Approve for me Codex's own auto-review in the workspace-writing
// sandbox (plan 12's Auto: codex --approve-for-me sets the same), Full
// access no sandbox and no approval, which the state asks for only once the
// owner confirms it (design D4). Codex keeps a turn's settings for the
// turns after it, so every turn/start carries the mode's three.
var modePresets = map[string]modePreset{
	event.ModeManual: {"readOnly", "on-request", reviewerUser},
	event.ModeAuto:   {"workspaceWrite", "on-request", reviewerAuto},
	event.ModeFull:   {"dangerFullAccess", "never", reviewerUser},
}

// presetOf is the mode's settings, Ask for approval's for "" or a word
// Codex has no preset for.
func presetOf(mode string) modePreset {
	if p, ok := modePresets[mode]; ok {
		return p
	}
	return modePresets[event.ModeManual]
}

// codexMode names the mode three settings make, as VS Code's extension
// reads them (plan 39, design D2): one of the presets, an unnamed policy
// read as on-request (plan 12's turn/start sent none), Codex's other
// auto-reviewer ("guardian_subagent") as auto_review, Custom otherwise; ""
// when no sandbox and no reviewer is given. A turn/start that names no
// sandbox (plan 12's before its addendum, P12x) is read by its reviewer
// alone, as plan 12 read it.
func codexMode(sandbox, policy, reviewer string) string {
	switch reviewer {
	case "guardian_subagent":
		reviewer = reviewerAuto
	case "":
		if sandbox == "" {
			return ""
		}
		reviewer = reviewerUser
	}
	if sandbox == "" {
		switch reviewer {
		case reviewerUser:
			return event.ModeManual
		case reviewerAuto:
			return event.ModeAuto
		}
		return event.ModeCustom
	}
	norm := func(v string) string { return strings.ToLower(strings.ReplaceAll(v, "-", "")) }
	if policy == "" {
		policy = "on-request"
	}
	for mode, p := range modePresets {
		if norm(sandbox) == norm(p.sandbox) && policy == p.policy && reviewer == p.reviewer {
			return mode
		}
	}
	return event.ModeCustom
}

// SetApprovalMode asks for one of Codex's modes (plan 39, design D3):
// between turns it writes thread/settings/update with the mode's settings
// at once, answered by the thread/settings/updated that follows (plan 38's
// path); during the main turn it waits for the turn's end; with a Codex
// that refused the update it goes with the next turn/start, which is when
// the change is in force (plan 12, item 2).
func (s *Session) SetApprovalMode(mode string) error {
	if _, ok := modePresets[mode]; !ok {
		return fmt.Errorf("Codex has no mode %q", mode)
	}
	s.mu.Lock()
	if !s.instant {
		s.mode, s.modeAsk = mode, ""
		s.mu.Unlock()
		return nil
	}
	s.modeAsk = mode
	if s.busy() || s.thread == "" {
		s.mu.Unlock()
		return nil
	}
	thread := s.thread
	s.modeSent = true
	s.mu.Unlock()
	if err := s.writeUpdate(thread, provider.ModelRequest{}, mode); err != nil {
		s.mu.Lock()
		s.modeAsk = ""
		s.mu.Unlock()
		return err
	}
	return nil
}

// AnswerQuestion: Codex has no question request here (Capabilities.Questions
// is false).
func (s *Session) AnswerQuestion(id string, answers map[string]string) error {
	return fmt.Errorf("Codex does not take answers to questions (request %q)", id)
}
