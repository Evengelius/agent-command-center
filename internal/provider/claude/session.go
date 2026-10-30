package claude

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/Evengelius/agent-command-center/internal/event"
	"github.com/Evengelius/agent-command-center/internal/proc"
	"github.com/Evengelius/agent-command-center/internal/provider"
	"github.com/Evengelius/agent-command-center/internal/provider/pipe"
)

// CloseStep is each wait of the close escalation (pipe.CloseStep;
// reliability.md, Close).
const CloseStep = pipe.CloseStep

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

// envValue is the variable's value in env, the names compared without case
// as Windows does; the dashboard's own environment when env is nil, which
// the child then inherits.
func envValue(env []string, name string) string {
	if env == nil {
		env = os.Environ()
	}
	v := ""
	for _, kv := range env {
		if k, val, ok := strings.Cut(kv, "="); ok && strings.EqualFold(k, name) {
			v = val // the last one wins, as in the process the child gets
		}
	}
	return v
}

// Session is one running `claude -p` process.
type Session struct {
	cli *pipe.Proc // the process: its events, readers, writes and close (plan 49, design D8)

	mu     sync.Mutex            // the mapper and the order of events
	m      *Mapper               //
	inputs map[string]pendingReq // pending approval id -> its request
	reqs   int                   // control requests sent
	reads  int                   // get_settings requests sent (plan 34)
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
	s := &Session{m: NewMapper(), inputs: map[string]pendingReq{}}
	s.m.SetEffortVar(envValue(env, EffortVar))
	if sc.ResumeLast && sc.ResumeID != "" {
		s.m.ResumeAsked(sc.ResumeID) // its refusal for want of the session is for good (plan 69, design D7)
	}
	s.cli = pipe.New(pipe.Config{Name: "Claude Code", Provider: ProviderID, Step: p.CloseStep, Now: p.Now, Mu: &s.mu, Line: s.line}, pr)
	s.cli.Run(ctx, append([]string{sc.Executable}, argv...), sc.ProjectPath)
	if err := s.control("initialize", `"hooks":null`); err != nil {
		s.cli.Problem("initialize not sent: " + err.Error())
	}
	return s, nil
}

// line maps one line of stdout, for the shared reader (plan 49, design D8),
// and has what it calls for written apart: the reader never waits on stdin,
// which Claude Code may not be reading while its stdout is full (plan 48,
// design D3; audit finding 9).
func (s *Session) line(l []byte) {
	switch held, ready, due := s.mapOut(l); {
	case ready:
		// The handshake's answer: its events are held while the settings read
		// and "/usage" are written, so both are asked for before the owner can
		// write anything, and their Sent events follow session.ready (plan 24,
		// amendment 23; P24u; plan 34).
		s.cli.Apart("asking for the handshake's reads", func() { s.handshakeReads(held) })
	case due:
		// A change was answered: the read that says what is in force (plan
		// 34).
		s.cli.Apart("asking for the model settings", s.readSettings)
	}
}

// mapOut maps one line of stdout under the lock, released however the
// mapping ends (plan 48, design D5), and queues its events: held, and ready
// true, when the line is the handshake's answer; due when a settings read is
// due.
func (s *Session) mapOut(l []byte) (held []*pipe.Item, ready, due bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	evs := s.m.Out(l)
	for _, ev := range evs {
		s.track(ev)
		ready = ready || ev.Kind == event.SessionReady
	}
	if ready {
		return s.cli.Queue().Hold(evs), true, false
	}
	for _, ev := range evs {
		s.cli.Queue().Push(ev)
	}
	return nil, false, s.m.ReadDue()
}

// handshakeReads writes the handshake's settings read and "/usage", then lets
// the handshake's held events go; a failed write is an event of its own.
func (s *Session) handshakeReads(held []*pipe.Item) {
	serr := s.control("get_settings", "")
	werr := s.readUsage()
	s.cli.Queue().Release(held, true)
	if serr != nil {
		s.cli.Problem("model settings not asked for: " + serr.Error())
	}
	if werr != nil {
		s.cli.Problem("usage report not asked for: " + werr.Error())
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

// Events are the session's events; the channel closes after session.ended.
func (s *Session) Events() <-chan event.Event { return s.cli.Events() }

// write sends one line. Its events are mapped before the write, so a reply
// can never be mapped first, and released only once the line is written.
func (s *Session) write(msg any) error { return s.writeWith(msg, s.m.In) }

// writeWith is write with the mapping of the line's own events: the
// dashboard's, In, or an internal line's, InInternal.
func (s *Session) writeWith(msg any, in func([]byte) []event.Event) error {
	return s.cli.Write(msg, func(l []byte) []*pipe.Item { return s.holdIn(l, in) })
}

// holdIn maps a line the dashboard writes under the lock, released however
// the mapping ends (plan 48, design D5); its events are held until the line
// is written.
func (s *Session) holdIn(l []byte, in func([]byte) []event.Event) []*pipe.Item {
	s.mu.Lock()
	defer s.mu.Unlock()
	evs := in(l)
	for _, ev := range evs {
		s.track(ev)
	}
	return s.cli.Queue().Hold(evs)
}

// control sends one control request; body is its fields after the subtype.
func (s *Session) control(subtype, body string) error {
	s.mu.Lock()
	var id string
	if subtype == "get_settings" {
		// The settings reads have their own numbers (plan 34), so they never
		// shift the numbers of the session's other requests.
		s.reads++
		id = "read_" + strconv.Itoa(s.reads)
	} else {
		s.reqs++
		id = "req_" + strconv.Itoa(s.reqs)
	}
	s.mu.Unlock()
	req := `{"subtype":"` + subtype + `"`
	if body != "" {
		req += "," + body
	}
	return s.write(map[string]any{"type": "control_request", "request_id": id, "request": json.RawMessage(req + "}")})
}

// imageLimits are Claude Code's limits on a message's images
// (doc/research/image-input.md): the Claude API's 10 MB an image once
// base64-encoded, 7.4 MB as sent, 8000 px a side, 100 images a request on
// a 200k-context model; the dashboard's own 7.4 MB a message keeps a
// written line under the 10 MB of piped input the headless page names.
var imageLimits = provider.ImageLimits{MaxCount: 100, MaxBytes: 7_400_000, MaxTotal: 7_400_000, MaxSide: 8000}

// Send writes the owner's message: its text byte for byte, and its images
// as base64 image blocks before the text, as the Agent SDK's streaming
// input takes them and the vision guide advises (plan 31;
// doc/research/image-input.md).
func (s *Session) Send(msg provider.Message) error { return s.write(userContent(msg)) }

// userContent is the owner's message as Claude Code reads it: a string
// content without images, content blocks with them, the images first.
func userContent(msg provider.Message) map[string]any {
	m := userMessage(msg.Text)
	if len(msg.Images) == 0 {
		return m
	}
	blocks := make([]map[string]any, 0, len(msg.Images)+1)
	for _, img := range msg.Images {
		blocks = append(blocks, map[string]any{"type": "image", "source": map[string]any{
			"type": "base64", "media_type": img.Media, "data": base64.StdEncoding.EncodeToString(img.Data)}})
	}
	if msg.Text != "" {
		blocks = append(blocks, map[string]any{"type": "text", "text": msg.Text})
	}
	m["message"] = map[string]any{"role": "user", "content": blocks}
	return m
}

// userMessage is a user message as Claude Code reads it.
func userMessage(text string) map[string]any {
	return map[string]any{
		"type":               "user",
		"message":            map[string]any{"role": "user", "content": text},
		"parent_tool_use_id": nil,
		"session_id":         "default",
	}
}

// readUsage asks for the usage report: "/usage" as a user message, which
// Claude Code answers itself with no model call (P24u, plan 24: the
// session, weekly and model-scoped windows). The mapper keeps the exchange
// out of the conversation.
func (s *Session) readUsage() error { return s.writeWith(userMessage("/usage"), s.m.InInternal) }

// DenyMessage is what Claude Code is told when the owner denies a request.
const DenyMessage = "The user denied this in Agent Command Center."

// QuestionDenyMessage denies a question the owner saw: the answer comes as
// their next message, since the updatedInput round trip is unprobed (CC-10
// q3; plan 5, item 10).
const QuestionDenyMessage = "The user saw these questions in Agent Command Center, which cannot answer them here, and will answer them in their next message. Wait for that message; do not ask them again."

// Answer answers a pending approval in Claude Code's words (plan 49, design
// D10): allow echoes the tool's input unchanged; deny and cancel deny it, a
// readable question's deny with QuestionDenyMessage. A question is answered
// by AnswerQuestion, never by an allow; Claude Code takes no allow for the
// session (CC-4).
func (s *Session) Answer(id string, d event.Decision) error {
	s.mu.Lock()
	req, ok := s.inputs[id]
	s.mu.Unlock()
	if !ok {
		return fmt.Errorf("%w %q", provider.ErrNoPending, id)
	}
	switch d {
	case event.Allow, event.Deny, event.Cancel:
	default:
		return fmt.Errorf("Claude Code takes no answer %q", d)
	}
	if d == event.Allow && req.question {
		return fmt.Errorf("request %q is a question: it is answered with AnswerQuestion", id)
	}
	if d == event.Allow && req.unreadable {
		return fmt.Errorf("request %q was not readable: only a deny answers it", id)
	}
	input := req.input
	msg := DenyMessage
	if d == event.Deny && req.question && !req.unreadable { // not a cancel; an unreadable question was never shown
		msg = QuestionDenyMessage
	}
	resp := map[string]any{"behavior": "deny", "message": msg}
	if d == event.Allow {
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

// Refresh asks for the context usage (undocumented, INT 4), the model
// settings in force (undocumented, plan 34) and the usage report (plan
// 24): all three after each turn.
func (s *Session) Refresh() error {
	if err := s.control("get_context_usage", ""); err != nil {
		return err
	}
	if err := s.control("get_settings", ""); err != nil {
		return err
	}
	return s.readUsage()
}

// readSettings asks for the model settings in force: get_settings, which
// Claude Code answers itself with no model call (plan 34, P34c). A write
// that fails while the session is still open is an event of its own.
func (s *Session) readSettings() {
	if err := s.control("get_settings", ""); err != nil && !s.cli.Closing() && !s.cli.InputClosed() {
		s.cli.Problem("model settings not asked for: " + err.Error())
	}
}

// SetModelSettings asks for a model, an effort or Ultracode, for this
// session only (plan 34): set_model for the model, apply_flag_settings for
// the effort and Ultracode, both documented. An effort carries Ultracode as
// the last read gave it, since a level asked for alone turns Ultracode off
// (P34c3). The read after each answer says what is in force.
func (s *Session) SetModelSettings(r provider.ModelRequest) error {
	if r.Model != "" {
		b, _ := json.Marshal(r.Model)
		if err := s.control("set_model", `"model":`+string(b)); err != nil {
			return err
		}
	}
	flags := map[string]any{}
	if r.Effort != "" {
		flags["effortLevel"] = r.Effort
		if r.Ultracode == nil {
			s.mu.Lock()
			on, known := s.m.UltracodeInForce()
			s.mu.Unlock()
			if known {
				flags["ultracode"] = on
			}
		}
	}
	if r.Ultracode != nil {
		flags["ultracode"] = *r.Ultracode
	}
	if len(flags) == 0 {
		return nil
	}
	b, _ := json.Marshal(flags)
	return s.control("apply_flag_settings", `"settings":`+string(b))
}

// Capabilities of Claude Code 2.1 in way A.
func (s *Session) Capabilities() provider.Capabilities {
	return provider.Capabilities{SubagentText: true, APICost: true, Questions: true, ResumeHistory: true, Modes: true, Images: imageLimits,
		Models: true, Ultracode: true}
}

// Close closes stdin, then terminates the CLI, then kills its tree, each
// after CloseStep; it returns when the process has exited or ctx ends, and
// the escalation goes on in the background (pipe.Proc).
func (s *Session) Close(ctx context.Context) error { return s.cli.Close(ctx) }

// permissionModes are Claude Code's permission modes by the dashboard's
// words: the four of the modes selector (plan 39; P39c, 2.1.303 takes each).
var permissionModes = map[string]string{event.ModeManual: "default", event.ModeAcceptEdits: "acceptEdits", event.ModePlan: "plan", event.ModeAuto: "auto"}

// SetApprovalMode asks Claude Code for one of its permission modes at once;
// the response and the system/status that follows say which is in force
// (plan 12, item 2; P12c-switch; plan 39, P39c).
func (s *Session) SetApprovalMode(mode string) error {
	m, ok := permissionModes[mode]
	if !ok {
		return fmt.Errorf("Claude Code has no mode %q", mode)
	}
	return s.control("set_permission_mode", `"mode":"`+m+`"`)
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
		return fmt.Errorf("%w %q", provider.ErrNoPending, id)
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
