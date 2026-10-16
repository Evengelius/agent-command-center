package state

import (
	"path"
	"strings"
	"time"

	"github.com/Evengelius/agent-command-center/internal/event"
)

// Apply applies one event from the CLI and returns the commands it calls for.
func (s *State) Apply(ev event.Event) []Command {
	s.log(ev)
	if s.Session.Phase == Closed {
		s.note(ev, "event after the session closed: logged, not applied")
		return nil
	}
	if ev.Agent != "" && s.Ambient[ev.Agent] {
		return nil
	}
	if ev.Agent == "" && s.Turn.Phase == Submitted && turnActivity(ev.Kind) {
		return append(s.toRunning(), s.apply(ev)...)
	}
	if ev.Agent == "" && s.Turn.Phase == Idle && s.Session.Phase == Ready && turnOpens(ev) {
		// Claude Code starts a turn by itself when a background agent's
		// task_notification arrives (P3 lines 80-98): no message was sent.
		s.Turn.Phase, s.Turn.Outcome, s.Turn.ByCLI = Running, "", true
		s.Turn.Notice = "turn started by the CLI"
	}
	return s.apply(ev)
}

// turnOpens is an event that only the start of a turn sends; deltas and
// completions can arrive late and never open one.
func turnOpens(ev event.Event) bool {
	if ev.Kind == event.TurnStarted {
		return true
	}
	m, ok := ev.Payload.(event.Marker)
	return ok && ev.Kind == event.MessageMarker && m.Name == "message_start"
}

// turnActivity is an event that only a turn the CLI took sends, which
// accepts the sent message ("Submitted" until the first event, reliability.md).
// A turn end counts: a turn can end with nothing streamed before its result.
func turnActivity(k event.Kind) bool {
	switch k {
	case event.TurnStarted, event.MessageDelta, event.MessageMarker, event.MessageCompleted, event.ToolStarted, event.TurnCompleted:
		return true
	}
	return false
}

func (s *State) toRunning() []Command {
	s.Turn.Phase = Running
	s.Turn.Draft = ""
	return []Command{{Kind: CmdDraftAccepted}}
}

func (s *State) apply(ev event.Event) []Command {
	switch p := ev.Payload.(type) {
	case event.Spawned:
		if ev.Kind == event.SessionSpawned {
			return s.spawned()
		}
	case event.Ready:
		return s.ready(ev)
	case event.Started:
		return s.started(ev, p)
	case event.Ended:
		return s.ended(ev, p)
	case event.Line:
		s.Session.StderrTail = appendBounded(s.Session.StderrTail, p.Text, MaxStderr)
		return nil
	case event.User:
		s.Conversation = append(s.Conversation, Entry{Seq: ev.Seq, Role: "user", Text: p.Text, Origin: ev.Origin, At: ev.At})
		if s.Turn.Phase == Idle {
			s.Turn.Phase, s.Turn.Count, s.Turn.ByCLI = Submitted, s.Turn.Count+1, false
		}
		return nil
	case event.Write:
		if (p.What == "interrupt" || p.What == "turn/interrupt") && (s.Turn.Phase == Running || s.Turn.Phase == WaitingApproval) {
			s.Turn.Phase = Interrupting
		}
		return nil
	case event.Delta:
		s.delta(ev, p)
		return nil
	case event.Marker:
		if p.Name == "message_stop" {
			if e := s.entry(ev.Agent, p.MessageID); e != nil {
				e.Streaming = false
			}
		}
		return nil
	case event.Completed:
		s.completed(ev, p)
		return nil
	case event.Tool:
		return s.tool(ev, p)
	case event.Agent:
		return s.agent(ev, p)
	case event.AgentKeys:
		s.agentSet(ev, p)
		return nil
	case event.Approval:
		if ev.Kind == event.ApprovalResolved {
			return s.resolved(ev, p)
		}
		return s.requested(ev, p)
	case event.Metrics:
		s.telemetry(ev, p)
		return nil
	case event.TurnEnd:
		return s.turnEnd(ev, p)
	case event.Diff:
		s.Diffs[p.Path] = p.Text
		return nil
	case event.Problem:
		if ev.Kind == event.Error && s.Session.Phase == Handshaking {
			return s.crash(ev, "handshake failed: "+p.Message, true)
		}
		s.Errors = appendBounded(s.Errors, ErrorEntry{Seq: ev.Seq, At: ev.At, Agent: ev.Agent, Message: p.Message, Retry: ev.Kind == event.Retry}, MaxErrors)
		if ev.Kind == event.Retry {
			s.Retries++
			if a := s.Agents[ev.Agent]; a != nil {
				a.Retries++
			}
		}
		return nil
	case event.Status:
		s.status(ev, p)
		return nil
	case event.Unknown:
		s.Unrecognised++
		s.UnrecognisedLast = appendBounded(s.UnrecognisedLast, ev.Raw, MaxUnrecognised)
		return nil
	}
	if ev.Kind == event.TurnStarted {
		return s.turnStarted(ev)
	}
	s.note(ev, "event without a known payload")
	return nil
}

func (s *State) spawned() []Command {
	if s.Session.Phase == "" || s.Session.Phase == Spawning {
		s.Session.Phase = Handshaking
		s.Screen = ScreenDashboard
	}
	return nil
}

func (s *State) ready(ev event.Event) []Command {
	if s.Session.Phase != Handshaking {
		s.note(ev, "handshake completed while "+string(s.Session.Phase))
		return nil
	}
	s.Session.Phase = Ready
	if s.Startup.Phase == Starting {
		s.Startup.Phase = StartReady
	}
	return nil
}

func (s *State) started(ev event.Event, p event.Started) []Command {
	s.Session.ID = p.SessionID
	s.Session.Model, s.Session.Cwd, s.Session.PermissionMode = p.Model, p.Cwd, p.PermissionMode
	s.Session.Version, s.Session.Plugins = p.Version, p.Plugins
	if p.Sandbox != "" {
		s.Session.Sandbox = p.Sandbox
	}
	if p.ApprovalPolicy != "" {
		s.Session.ApprovalPolicy = p.ApprovalPolicy
	}
	if p.Model != "" {
		s.Metrics["model"] = Value{Present: true, Text: p.Model, Origin: ev.Origin, Source: string(ev.Kind), Documented: true, ReceivedAt: ev.At}
	}
	// Claude Code reports its cwd only with the first turn (system/init),
	// after the handshake, so the check runs on every session.started. A mock
	// session has no working directory: its "project" is the mock file.
	if !s.Context.Mock && s.Context.Project != "" && p.Cwd != "" && !SamePath(s.Context.Project, p.Cwd) {
		return s.crash(ev, "working directory mismatch: the CLI reports "+p.Cwd+", the chosen project is "+s.Context.Project, true)
	}
	return nil
}

// SamePath compares two Windows paths without touching the disk: cleaned,
// separators unified, case ignored. It uses package path, not path/filepath,
// which would bring os into the reducer.
func SamePath(a, b string) bool {
	norm := func(p string) string {
		p = path.Clean(strings.ReplaceAll(p, `\`, "/"))
		return strings.TrimRight(strings.ToLower(p), "/")
	}
	return norm(a) == norm(b)
}

func (s *State) ended(ev event.Event, p event.Ended) []Command {
	s.Session.ExitCode, s.Session.HasExitCode = p.Code, p.HasCode
	if s.Session.Phase == Closing {
		s.Session.Phase = Closed
		s.endAll(ev)
		if s.Startup.Phase == Exited {
			return []Command{{Kind: CmdExit}}
		}
		return nil
	}
	return s.crash(ev, "the CLI exited unexpectedly", false)
}

// crash moves the session to Crashed with its consequences on the other
// machines. close asks the application to stop the process.
func (s *State) crash(ev event.Event, reason string, close bool) []Command {
	if s.Session.Phase == Crashed || s.Session.Phase == Closed {
		return nil
	}
	s.Session.Phase, s.Session.Reason = Crashed, reason
	s.Errors = appendBounded(s.Errors, ErrorEntry{Seq: ev.Seq, At: ev.At, Message: reason}, MaxErrors)
	if s.Startup.Phase == Starting {
		s.Startup.Phase, s.Startup.Failure = StartFailed, reason
	}
	var cmds []Command
	if s.Turn.Phase == Submitted && s.Turn.Draft != "" {
		// Not accepted before the crash: the message goes back to the editor,
		// as after a failed write (ASM-23).
		cmds = append(cmds, Command{Kind: CmdRestoreDraft, Text: s.Turn.Draft})
		s.Turn.Draft = ""
	}
	if s.Turn.Phase != Idle {
		s.Turn.Phase, s.Turn.Outcome = Idle, "failed"
	}
	s.endAll(ev)
	if close {
		cmds = append(cmds, Command{Kind: CmdClose})
	}
	return cmds
}

// endAll ends what cannot outlive the session: agents, approvals, tools.
func (s *State) endAll(ev event.Event) {
	for _, k := range s.Order {
		if a := s.Agents[k]; !a.Status.Final() {
			a.Status, a.StatusOrigin, a.EndedAt = AgentEnded, event.Inferred, ev.At
		}
	}
	for _, id := range s.Queue {
		if a := s.Approvals[id]; !a.Status.Final() {
			a.Status = Expired
		}
	}
	for _, t := range s.Tools {
		if !t.Status.Final() {
			t.Status, t.Origin, t.EndedAt = ToolUnfinished, event.Inferred, ev.At
		}
	}
	if s.Turn.Phase != Idle {
		s.Turn.Phase = Idle
		if s.Turn.Outcome == "" {
			s.Turn.Outcome = "interrupted"
		}
	}
}

// messages returns the record an agent key writes to.
func (s *State) messages(key string) *[]Entry {
	if key == "" {
		return &s.Conversation
	}
	return &s.ensureAgent(key, event.Event{}).Messages
}

func (s *State) entry(agent, id string) *Entry {
	if id == "" {
		return nil
	}
	list := s.messages(agent)
	for i := len(*list) - 1; i >= 0; i-- {
		if (*list)[i].MessageID == id && (*list)[i].Role == "assistant" {
			return &(*list)[i]
		}
	}
	return nil
}

func (s *State) delta(ev event.Event, p event.Delta) {
	if ev.Agent != "" {
		s.ensureAgent(ev.Agent, ev)
		if a := s.Agents[ev.Agent]; a.Status.Final() {
			a.History = append(a.History, "message after end")
		}
	}
	if p.Block != "text" && p.Block != "thinking" {
		return
	}
	e := s.entry(ev.Agent, p.MessageID)
	if e == nil {
		list := s.messages(ev.Agent)
		*list = append(*list, Entry{Seq: ev.Seq, Role: "assistant", Agent: ev.Agent, MessageID: p.MessageID, Origin: ev.Origin, Streaming: true, At: ev.At})
		e = &(*list)[len(*list)-1]
	}
	if p.Block == "thinking" {
		e.Thinking = true
		return
	}
	e.Text += p.Text
}

func (s *State) completed(ev event.Event, p event.Completed) {
	e := s.entry(ev.Agent, p.MessageID)
	if e == nil {
		if p.Text == "" {
			return
		}
		list := s.messages(ev.Agent)
		*list = append(*list, Entry{Seq: ev.Seq, Role: "assistant", Agent: ev.Agent, MessageID: p.MessageID, Text: p.Text, Origin: ev.Origin, Thinking: p.Thinking, At: ev.At})
		return
	}
	if e.Text != p.Text && e.Text != "" {
		s.note(ev, "streamed text differs from the completed message; the completed message wins")
	}
	e.Text, e.Streaming, e.Thinking = p.Text, false, e.Thinking || p.Thinking
}

func (s *State) tool(ev event.Event, p event.Tool) []Command {
	id := p.ToolID
	if id == "" {
		id = ev.Item
	}
	t := s.Tools[id]
	switch ev.Kind {
	case event.ToolStarted:
		if t != nil {
			s.note(ev, "tool started twice: "+id)
			return nil
		}
		s.Tools[id] = &ToolCall{ID: id, Agent: ev.Agent, Name: p.Name, Input: p.Input, Status: ToolStarted, Origin: ev.Origin, StartedAt: ev.At}
		if ev.Agent == "" {
			s.Conversation = append(s.Conversation, Entry{Seq: ev.Seq, Role: "tool", MessageID: id, Text: p.Name, Origin: ev.Origin, At: ev.At})
		} else {
			a := s.ensureAgent(ev.Agent, ev)
			if a.Status.Final() {
				a.History = append(a.History, "tool after end: "+p.Name)
			}
			a.Tools, a.LastTool = append(a.Tools, id), p.Name
		}
	case event.ToolProgress:
		if t == nil {
			s.note(ev, "progress for an unknown tool: "+id)
			return nil
		}
		if t.Status == ToolStarted {
			t.Status = ToolRunning
		}
		if p.Output != "" {
			t.Output += p.Output
		}
	case event.ToolCompleted:
		final := ToolCompleted
		if p.IsError {
			final = ToolFailed
		}
		if t == nil {
			s.Tools[id] = &ToolCall{ID: id, Agent: ev.Agent, Name: p.Name, Status: final, Origin: event.Inferred, Output: p.Output, EndedAt: ev.At}
			return nil
		}
		if t.Status.Final() {
			t.History = append(t.History, "second completion ignored")
			return nil
		}
		t.Status, t.Output, t.EndedAt, t.Origin = final, p.Output, ev.At, ev.Origin
	}
	return nil
}

// ensureAgent returns the node of a key, creating an unannounced one.
func (s *State) ensureAgent(key string, ev event.Event) *Agent {
	if a := s.Agents[key]; a != nil {
		return a
	}
	a := &Agent{Key: key, Label: "unannounced agent", Status: AgentRunning, StatusOrigin: event.Inferred, StartedAt: ev.At, Runs: 1}
	s.Agents[key] = a
	s.Order = append(s.Order, key)
	return a
}

func (s *State) agent(ev event.Event, p event.Agent) []Command {
	key := p.Key
	if key == "" {
		key = ev.Agent
	}
	if key == "" {
		s.note(ev, "agent event without a key: attached to the orchestrator, unattributed")
		return nil
	}
	if p.Type == "ambient" {
		s.Ambient[key] = true
		return nil
	}
	if s.Ambient[key] {
		return nil
	}
	a, known := s.Agents[key]
	switch ev.Kind {
	case event.AgentStarted:
		if !known {
			a = &Agent{Key: key, StartedAt: ev.At}
			s.Agents[key] = a
			s.Order = append(s.Order, key)
		} else if !a.Status.Final() && a.StatusOrigin != event.Inferred {
			s.update(a, p)
			return nil
		}
		a.Runs++
		a.Status, a.StatusOrigin, a.EndedAt = AgentRunning, ev.Origin, time.Time{}
		if p.Type == "thread" {
			a.Status = AgentAnnounced
		}
		a.Kind, a.Parent = p.Type, p.Parent
		s.update(a, p)
	case event.AgentProgress:
		if !known {
			a = s.ensureAgent(key, ev)
		}
		if a.Status.Final() {
			a.History = append(a.History, "progress after end")
			return nil
		}
		s.update(a, p)
	default: // AgentCompleted, AgentFailed
		st := AgentCompleted
		if ev.Kind == event.AgentFailed {
			st = AgentFailed
			if p.Status == "stopped" || p.Status == "killed" || p.Status == "interrupted" {
				st = AgentStopped
			}
		}
		if !known {
			a = &Agent{Key: key, Label: "unannounced agent", Runs: 1}
			s.Agents[key] = a
			s.Order = append(s.Order, key)
			s.update(a, p)
			a.Status, a.StatusOrigin, a.EndedAt = st, event.Inferred, ev.At
			return nil
		}
		s.update(a, p)
		if a.Status.Final() {
			return nil
		}
		s.finishAgent(a, st, ev)
	}
	return nil
}

func (s *State) finishAgent(a *Agent, st AgentStatus, ev event.Event) {
	a.Status, a.StatusOrigin, a.EndedAt = st, ev.Origin, ev.At
	for _, id := range a.Tools {
		if t := s.Tools[id]; t != nil && !t.Status.Final() {
			t.Status, t.Origin, t.EndedAt = ToolUnfinished, event.Inferred, ev.At
		}
	}
	s.expireFor(a.Key)
}

func (s *State) update(a *Agent, p event.Agent) {
	if p.Label != "" {
		a.Label = p.Label
	}
	if p.Description != "" {
		a.Description = p.Description
	}
	if p.Parent != "" {
		a.Parent = p.Parent
	}
	if p.Backgrounded {
		a.Backgrounded = true
	}
	if p.LastTool != "" {
		a.LastTool = p.LastTool
	}
	if p.ToolUses > a.ToolUses {
		a.ToolUses = p.ToolUses
	}
	if p.HasTokens {
		a.Tokens = Value{Present: true, V: float64(p.Tokens), Unit: "tokens", Origin: event.Observed, Documented: true, Source: "agent usage"}
	}
	if p.Summary != "" {
		a.Summary = p.Summary
	}
}

// agentSet applies Claude Code's background task set: replace, not merge.
func (s *State) agentSet(ev event.Event, p event.AgentKeys) {
	in := map[string]bool{}
	for _, k := range p.Keys {
		in[k] = true
		if s.Ambient[k] {
			continue
		}
		s.ensureAgent(k, ev).InSet = true
	}
	for _, k := range s.Order {
		a := s.Agents[k]
		if a.InSet && !in[k] && !a.Status.Final() {
			s.finishAgent(a, AgentEnded, ev)
			a.StatusOrigin = event.Inferred
			a.History = append(a.History, "ended, status not reported")
		}
	}
}

func (s *State) requested(ev event.Event, p event.Approval) []Command {
	if p.ID != "" && s.Approvals[p.ID] != nil {
		s.note(ev, "approval requested twice: "+p.ID)
		return nil
	}
	id := p.ID
	if id == "" {
		id = "unreadable:" + itoa(ev.Seq)
	}
	a := &Approval{ID: id, Agent: ev.Agent, Tool: p.Tool, Display: p.Display, Summary: p.Summary, ToolID: p.ToolID,
		Input: p.Input, Question: ev.Kind == event.Question, Problem: p.Problem, Raw: ev.Raw, Status: Pending, RequestedAt: ev.At}
	s.Approvals[id] = a
	s.Queue = append(s.Queue, id)
	s.Conversation = append(s.Conversation, Entry{Seq: ev.Seq, Role: "approval", Agent: ev.Agent, MessageID: id, Text: p.Tool, Origin: ev.Origin, At: ev.At})
	if p.Problem != "" {
		// An unreadable request is denied, never allowed (INT 4 fallback).
		a.Status, a.Answer, a.AnsweredAt = Denied, "deny", ev.At
		if p.ID == "" {
			return nil
		}
		return []Command{{Kind: CmdAnswer, ID: p.ID, Decision: "deny"}}
	}
	if ev.Agent != "" {
		ag := s.ensureAgent(ev.Agent, ev)
		if ag.Status == AgentRunning || ag.Status == AgentAnnounced {
			ag.Status = AgentWaiting
		}
	}
	if s.Turn.Phase == Running {
		s.Turn.Phase = WaitingApproval
	}
	return nil
}

func (s *State) resolved(ev event.Event, p event.Approval) []Command {
	a := s.Approvals[p.ID]
	if a == nil {
		s.Errors = appendBounded(s.Errors, ErrorEntry{Seq: ev.Seq, At: ev.At, Message: "answer for an unknown approval: " + p.ID}, MaxErrors)
		return nil
	}
	if a.Status.Final() {
		if p.Decision != "" && p.Decision != "resolved" {
			s.note(ev, "second answer ignored: "+p.ID)
		}
		return nil
	}
	switch p.Decision {
	case "allow", "accept", "acceptForSession", "approved":
		a.Status = Allowed
	case "deny", "decline", "cancel", "denied", "abort":
		a.Status = Denied
	default: // expired, or resolved by the CLI without the owner's answer
		a.Status = Expired
	}
	a.Answer, a.AnsweredAt = p.Decision, ev.At
	if a.Status == Denied {
		if t := s.Tools[a.ToolID]; t != nil && t.Status == ToolStarted {
			t.Status, t.EndedAt = ToolDenied, ev.At
		}
	}
	s.settleWaiting()
	return nil
}

// settleWaiting moves the turn and agents back to Running once none of
// their approvals is pending.
func (s *State) settleWaiting() {
	pending := map[string]bool{}
	for _, id := range s.Queue {
		if a := s.Approvals[id]; !a.Status.Final() {
			pending[a.Agent] = true
		}
	}
	if s.Turn.Phase == WaitingApproval && len(pending) == 0 {
		s.Turn.Phase = Running
	}
	for _, k := range s.Order {
		if a := s.Agents[k]; a.Status == AgentWaiting && !pending[k] {
			a.Status = AgentRunning
		}
	}
}

func (s *State) turnStarted(ev event.Event) []Command {
	if ev.Agent != "" {
		a := s.ensureAgent(ev.Agent, ev)
		switch {
		case a.Status == AgentAnnounced:
			a.Status, a.StatusOrigin = AgentRunning, ev.Origin
		case a.Status.Final():
			a.Runs++
			a.Status, a.StatusOrigin, a.EndedAt = AgentRunning, ev.Origin, time.Time{}
		}
		return nil
	}
	if s.Turn.Phase != Running && s.Turn.Phase != WaitingApproval && s.Turn.Phase != Interrupting {
		s.note(ev, "turn started while "+string(s.Turn.Phase))
	}
	return nil
}

func (s *State) turnEnd(ev event.Event, p event.TurnEnd) []Command {
	if ev.Agent != "" {
		a := s.ensureAgent(ev.Agent, ev)
		if a.Status.Final() {
			a.History = append(a.History, "turn end after end")
			return nil
		}
		switch p.Outcome {
		case "completed":
			s.finishAgent(a, AgentCompleted, ev)
		case "interrupted":
			s.finishAgent(a, AgentStopped, ev)
		default:
			s.finishAgent(a, AgentFailed, ev)
			a.Error = p.Error
		}
		return nil
	}
	if s.Turn.Phase == Idle {
		s.note(ev, "unexpected turn end")
		return nil
	}
	outcome := p.Outcome
	if s.Turn.Phase == Interrupting {
		outcome = "interrupted"
	}
	s.Turn.Phase, s.Turn.Outcome, s.Turn.ForceStopOffered = Idle, outcome, false
	if outcome == "failed" && p.Error != "" {
		s.Errors = appendBounded(s.Errors, ErrorEntry{Seq: ev.Seq, At: ev.At, Message: "turn failed: " + p.Error}, MaxErrors)
	}
	s.expireFor("")
	for _, t := range s.Tools {
		if t.Agent == "" && !t.Status.Final() {
			t.Status, t.Origin, t.EndedAt = ToolUnfinished, event.Inferred, ev.At
		}
	}
	return []Command{{Kind: CmdRefresh}}
}

// expireFor expires the pending approvals of one agent key ("" is the
// orchestrator): its turn ended before they were answered.
func (s *State) expireFor(key string) {
	for _, id := range s.Queue {
		if a := s.Approvals[id]; a.Agent == key && !a.Status.Final() {
			a.Status = Expired
		}
	}
	s.settleWaiting()
}

func (s *State) telemetry(ev event.Event, p event.Metrics) {
	for _, m := range p.Items {
		v := Value{Present: true, V: m.Value, Text: m.Text, Unit: m.Unit, Origin: ev.Origin, Source: m.Source,
			Documented: m.Documented, ReceivedAt: ev.At, ResetsAt: m.ResetsAt}
		if ev.Agent != "" {
			a := s.ensureAgent(ev.Agent, ev)
			if m.Name == "tokens.total" {
				a.Tokens = v
			}
			continue
		}
		s.Metrics[m.Name] = v
	}
	s.deriveTokens(ev)
}

// deriveTokens keeps "tokens.session": Codex, the main thread's last total
// plus each child thread's; Claude Code, the sum of each turn's result usage.
func (s *State) deriveTokens(ev event.Event) {
	if main, ok := s.Metrics["tokens.total"]; ok {
		sum := main.V
		for _, k := range s.Order {
			if a := s.Agents[k]; a.Kind == "thread" && a.Tokens.Present {
				sum += a.Tokens.V
			}
		}
		s.Metrics["tokens.session"] = Value{Present: true, V: sum, Unit: "tokens", Origin: event.Derived, Source: "sum of each thread's last tokens.total", Documented: true, ReceivedAt: ev.At}
		return
	}
	if t, ok := s.Metrics["tokens.turn"]; ok && t.ReceivedAt.Equal(ev.At) {
		prev := s.Metrics["tokens.session"]
		s.Metrics["tokens.session"] = Value{Present: true, V: prev.V + t.V, Unit: "tokens", Origin: event.Derived, Source: "sum of result.usage over turns", Documented: true, ReceivedAt: ev.At}
	}
}

func (s *State) status(ev event.Event, p event.Status) {
	if p.Status == "notice" {
		list := s.messages(ev.Agent)
		*list = append(*list, Entry{Seq: ev.Seq, Role: "notice", Agent: ev.Agent, Text: p.Detail, Origin: ev.Origin, At: ev.At})
		return
	}
	if ev.Agent != "" {
		if a := s.ensureAgent(ev.Agent, ev); a.Status == AgentAnnounced && (p.Status == "active" || p.Status == "running") {
			a.Status, a.StatusOrigin = AgentRunning, ev.Origin
		}
		return
	}
	if p.Status != "" {
		s.Session.Status = p.Status
	}
}

func (s *State) log(ev event.Event) {
	s.Log = appendBounded(s.Log, ev, MaxLog)
}

func (s *State) note(ev event.Event, msg string) {
	s.Notes = appendBounded(s.Notes, "#"+itoa(ev.Seq)+" "+string(ev.Kind)+": "+msg, MaxErrors)
}

func appendBounded[T any](list []T, v T, max int) []T {
	list = append(list, v)
	if len(list) > max {
		list = append(list[:0:0], list[len(list)-max:]...)
	}
	return list
}

func itoa(n uint64) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
