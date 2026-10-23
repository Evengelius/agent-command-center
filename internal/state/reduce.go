package state

import (
	"path"
	"strconv"
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
	if ev.Agent != "" && ev.Agent == s.Session.ID {
		// The session's own key is the orchestrator's, never an agent of
		// zone C, whatever a CLI version sends (L5: Codex 0.163.0's "/root").
		ev.Agent = ""
	}
	var cmds []Command
	if ev.Agent == "" && s.Turn.Phase == Submitted && turnActivity(ev.Kind) {
		cmds = s.toRunning()
	} else if ev.Agent == "" && s.Turn.Phase == Idle && s.Session.Phase == Ready && turnOpens(ev) {
		// Claude Code starts a turn by itself when a background agent's
		// task_notification arrives (P3 lines 80-98): no message was sent.
		s.Turn.Phase, s.Turn.Outcome, s.Turn.ByCLI = Running, "", true
		s.Turn.Notice = "turn started by the CLI"
	}
	n := len(s.Conversation)
	s.syncStage(ev.At)
	cmds = append(cmds, s.apply(ev)...)
	s.syncStage(ev.At)
	// After the stage the event may open: a prompt belongs to the turn it
	// opens (Entry.Stage).
	for i := n; i < len(s.Conversation); i++ {
		s.Conversation[i].Stage = s.stage()
	}
	return cmds
}

// syncStage opens a stage when a turn starts and closes it with the turn's
// outcome when it ends: zone A's stages are inferred from the turn machine
// (events-and-state.md, Agent tree). at is zero for an action; the stage
// takes its start from the first event after.
func (s *State) syncStage(at time.Time) {
	n := len(s.Stages)
	open := n > 0 && s.Stages[n-1].Open
	switch {
	case s.Turn.Phase != Idle && !open:
		s.Stages = append(s.Stages, Stage{N: n + 1, ByCLI: s.Turn.ByCLI, Open: true, StartedAt: at, Prompt: s.prompt()})
	case s.Turn.Phase == Idle && open:
		st := &s.Stages[n-1]
		st.Open, st.Outcome, st.EndedAt = false, s.Turn.Outcome, at
	case open && s.Stages[n-1].StartedAt.IsZero():
		s.Stages[n-1].StartedAt = at
	}
}

// prompt is the message that opens a turn: the draft being sent, or in a
// replay the user entry the event just added; "" for a turn the CLI opens.
func (s *State) prompt() string {
	switch n := len(s.Conversation); {
	case s.Turn.ByCLI:
		return ""
	case s.Turn.Draft != "":
		return s.Turn.Draft
	case n > 0 && s.Conversation[n-1].Role == "user" && s.Conversation[n-1].Agent == "":
		return s.Conversation[n-1].Text
	}
	return ""
}

// stage is the stage a new tool or agent belongs to: the last one, open or
// not, since a background agent outlives its turn; 0 before any turn.
func (s *State) stage() int {
	return len(s.Stages)
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
	s.Turn.Draft, s.Turn.DraftImages = "", nil
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
		if ev.Kind == event.ReviewWarning {
			return nil // Codex's guardianWarning: in the event log only (plan 12, item 2)
		}
		s.Session.StderrTail = appendBounded(s.Session.StderrTail, p.Text, MaxStderr)
		return nil
	case event.User:
		s.Conversation = append(s.Conversation, Entry{Seq: ev.Seq, Role: "user", Text: p.Text, Origin: ev.Origin, At: ev.At, Images: s.namedImages(p.Images)})
		if s.Turn.Phase == Idle {
			// A prompt the dashboard did not send, a mock file's: it opens the
			// turn as Send does, the last turn's notice gone.
			s.Turn.Phase, s.Turn.Count, s.Turn.ByCLI, s.Turn.Notice = Submitted, s.Turn.Count+1, false, ""
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
		s.marker(ev, p)
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
	case event.Plan:
		// The turn's plan as a whole, replaced on each update; a child
		// thread's plan is noted on it, not drawn (plan 4, step 6, check 3).
		if ev.Agent != "" {
			a := s.ensureAgent(ev.Agent, ev)
			a.History = append(a.History, "plan updated")
			return nil
		}
		if n := len(s.Stages); n > 0 {
			s.Stages[n-1].Plan = p.Steps
		} else {
			s.note(ev, "plan outside a turn")
		}
		return nil
	case event.Problem:
		if ev.Kind == event.Error && s.Session.Phase == Handshaking {
			return s.crash(ev, "handshake failed: "+p.Message, true)
		}
		s.Errors = appendBounded(s.Errors, ErrorEntry{Seq: ev.Seq, At: ev.At, Agent: ev.Agent, Message: p.Message, Retry: ev.Kind == event.Retry}, MaxErrors)
		if p.TurnRefused && s.Turn.Phase == Submitted {
			// The CLI refused the message: no turn started, and it goes back
			// to the editor, as after a failed write (plan 34). Zone B says so
			// under the owner's entry, which the write had added.
			s.Turn.Phase, s.Turn.Outcome, s.Turn.Notice = Idle, "failed", "the message could not be sent: "+p.Message
			s.Conversation = append(s.Conversation, Entry{Seq: ev.Seq, Role: "notice", Text: "the message was not sent: " + p.Message + "; it is back in the editor",
				Origin: ev.Origin, At: ev.At, Stage: s.stage()})
			if s.Turn.pending() {
				return []Command{s.restoreDraft()}
			}
		}
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
	case event.Mode:
		s.approvalMode(ev, p)
		return nil
	case event.Models:
		s.models(p)
		return nil
	case event.Settings:
		s.modelSettings(ev, p)
		return nil
	case event.Verdict:
		s.verdict(ev, p)
		return nil
	case event.Review:
		s.review(ev, p)
		return nil
	case event.Unknown:
		s.Unrecognised++
		s.UnrecognisedLast = appendBounded(s.UnrecognisedLast, UnknownEntry{At: ev.At, Type: p.Type, Reason: p.Reason, Line: ev.Raw}, MaxUnrecognised)
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
		// A new process lists its models and reports its settings itself
		// (plan 34, design D10): nothing of an earlier one is shown.
		s.Models = Models{}
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
	if p.ApprovalMode != "" {
		s.inForce(p.ApprovalMode)
	}
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
		var cmds []Command
		if s.Turn.Phase == Submitted && s.Turn.pending() {
			// Closed before the CLI took the message: it goes back to the
			// editor, so the draft saved on the way out keeps it (ASM-23).
			cmds = append(cmds, s.restoreDraft())
		}
		s.dropModelChange(ev)
		// Closed by the owner: what runs is interrupted, not failed (plan 22,
		// criterion 1.6).
		s.endAll(ev, "interrupted")
		if s.Startup.Phase == Exited {
			cmds = append(cmds, Command{Kind: CmdExit})
		}
		return cmds
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
	if s.Turn.Phase == Submitted && s.Turn.pending() {
		// Not accepted before the crash: the message goes back to the editor,
		// as after a failed write (ASM-23).
		cmds = append(cmds, s.restoreDraft())
	}
	s.dropModelChange(ev)
	if s.Turn.Phase != Idle {
		s.Turn.Phase, s.Turn.Outcome = Idle, "failed"
	}
	s.endAll(ev, "failed")
	if close {
		cmds = append(cmds, Command{Kind: CmdClose})
	}
	return cmds
}

// endAll ends what cannot outlive the session: agents, approvals, tools and
// the orchestrator's thinking phases, these with outcome: "failed" with a
// crashed session, "interrupted" with one the owner closed (plan 22,
// criterion 1.6).
func (s *State) endAll(ev event.Event, outcome string) {
	s.endThoughts(ev.At, outcome)
	for _, k := range s.Order {
		if a := s.Agents[k]; !a.Status.Final() {
			a.Status, a.StatusOrigin, a.EndedAt, a.EndStage = AgentEnded, event.Inferred, ev.At, s.stage()
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
	if ev.Agent == "" {
		if p.Block == "thinking" {
			// The orchestrator's thinking is its thinking entry, from the
			// block's markers or the step's: no assistant entry for it (plan
			// 22, item 9). A subagent's message keeps its mark.
			return
		}
		s.dropStep(ev.At) // text is a step's first output (plan 22, criterion 1.5)
	}
	e := s.entry(ev.Agent, p.MessageID)
	if e == nil {
		list := s.messages(ev.Agent)
		*list = append(*list, Entry{Seq: ev.Seq, Role: "assistant", Agent: ev.Agent, MessageID: p.MessageID, Origin: ev.Origin, Streaming: true, At: ev.At})
		e = &(*list)[len(*list)-1]
	}
	e.Streaming = true // a block's completed line may have come before this one's deltas
	if p.Block == "thinking" {
		e.Thinking = true
		return
	}
	e.Text += p.Text
}

func (s *State) completed(ev event.Event, p event.Completed) {
	if ev.Agent == "" && p.Text != "" {
		s.dropStep(ev.At) // a message is a step's first output (plan 22, criterion 1.5)
	}
	e := s.entry(ev.Agent, p.MessageID)
	if e == nil {
		if p.Text == "" {
			return
		}
		list := s.messages(ev.Agent)
		*list = append(*list, Entry{Seq: ev.Seq, Role: "assistant", Agent: ev.Agent, MessageID: p.MessageID, Text: p.Text, Origin: ev.Origin, Thinking: p.Thinking, At: ev.At})
		return
	}
	// Claude Code sends one completed line per content block, with the
	// message's id (testdata/probes/p2-claude-conversation.jsonl: the thinking
	// block's line, then the text's): a line never takes away text another
	// block streamed. Codex sends the whole message once.
	e.Thinking = e.Thinking || p.Thinking
	switch {
	case p.Text == "": // a thinking or tool block's line: the text stays
	case e.Text == "" || strings.HasPrefix(p.Text, e.Text):
		e.Text = p.Text
	case strings.Contains(e.Text, p.Text): // one block of a message already streamed
	default:
		s.note(ev, "streamed text differs from the completed message; the completed message wins")
		e.Text = p.Text
	}
	e.Streaming = false
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
		s.Tools[id] = &ToolCall{ID: id, Agent: ev.Agent, Name: p.Name, Input: p.Input, Status: ToolStarted, Origin: ev.Origin, StartedAt: ev.At, Stage: s.stage(), Seq: ev.Seq}
		if ev.Agent == "" {
			s.dropStep(ev.At) // a call is a step's first output (plan 22, criterion 1.5)
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
			s.Tools[id] = &ToolCall{ID: id, Agent: ev.Agent, Name: p.Name, Status: final, Origin: event.Inferred, Output: p.Output, ExitCode: p.ExitCode, HasExitCode: p.HasExitCode, EndedAt: ev.At, Stage: s.stage(), Seq: ev.Seq}
			return nil
		}
		if t.Status.Final() {
			t.History = append(t.History, "second completion ignored")
			return nil
		}
		t.Status, t.Output, t.EndedAt, t.Origin = final, p.Output, ev.At, ev.Origin
		t.ExitCode, t.HasExitCode = p.ExitCode, p.HasExitCode
	}
	return nil
}

// ensureAgent returns the node of a key, creating an unannounced one.
func (s *State) ensureAgent(key string, ev event.Event) *Agent {
	if a := s.Agents[key]; a != nil {
		return a
	}
	a := &Agent{Key: key, Label: "unannounced agent", Status: AgentRunning, StatusOrigin: event.Inferred, StartedAt: ev.At, Runs: 1, Stage: s.stage(), Seq: ev.Seq}
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
	if key == s.Session.ID {
		s.note(ev, "agent event for the orchestrator's own key: not an agent")
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
			a = &Agent{Key: key, StartedAt: ev.At, Stage: s.stage(), Seq: ev.Seq}
			s.Agents[key] = a
			s.Order = append(s.Order, key)
		} else if !a.Status.Final() && a.StatusOrigin != event.Inferred {
			s.update(a, p)
			return nil
		}
		a.Runs++
		a.Status, a.StatusOrigin, a.EndedAt, a.EndStage = AgentRunning, ev.Origin, time.Time{}, 0
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
			// The model of a message that ended late, or a spawn request's
			// model, effort and label with its parent (plan 6, items 4 and 6;
			// plan 10, item 1): no progress.
			if p == (event.Agent{Key: p.Key, Parent: p.Parent, Label: p.Label, Description: p.Description, Model: p.Model, Effort: p.Effort, ModelRequested: p.ModelRequested}) {
				s.update(a, p)
				return nil
			}
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
			a = &Agent{Key: key, Label: "unannounced agent", Runs: 1, Stage: s.stage(), Seq: ev.Seq}
			s.Agents[key] = a
			s.Order = append(s.Order, key)
			s.update(a, p)
			a.Status, a.StatusOrigin, a.EndedAt, a.EndStage = st, event.Inferred, ev.At, s.stage()
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
	a.Status, a.StatusOrigin, a.EndedAt, a.EndStage = st, ev.Origin, ev.At, s.stage()
	for _, id := range a.Tools {
		if t := s.Tools[id]; t != nil && !t.Status.Final() {
			t.Status, t.Origin, t.EndedAt = ToolUnfinished, event.Inferred, ev.At
		}
	}
	s.expireFor(a.Key)
}

func (s *State) update(a *Agent, p event.Agent) {
	// A spawn request's label and description fill only an agent with none
	// of its own, whose own label then replaces them, in either order (plan
	// 10, item 1).
	if p.Label != "" && p.ModelRequested {
		if a.Label == "" || a.Label == "unannounced agent" || a.LabelRequested {
			a.Label, a.LabelRequested = p.Label, true
		}
	} else if p.Label != "" {
		a.Label, a.LabelRequested = p.Label, false
	}
	if p.Description != "" && (!p.ModelRequested || a.Description == "") {
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
	// Only the agent's own model, never the session's (plan 5, item 7); a
	// requested one (a spawn call's) fills only an empty model, so the agent's
	// report wins in either order (plan 6, item 4).
	if p.Model != "" && p.ModelRequested && !a.Model.Present {
		a.Model = Value{Present: true, Text: p.Model, Origin: event.Observed, Documented: true, Source: "spawn request"}
	} else if p.Model != "" && !p.ModelRequested {
		a.Model = Value{Present: true, Text: p.Model, Origin: event.Observed, Documented: true, Source: "agent model"}
	}
	// The reasoning effort, by the same rule (plan 6, item 6).
	if p.Effort != "" && p.ModelRequested && !a.Effort.Present {
		a.Effort = Value{Present: true, Text: p.Effort, Origin: event.Observed, Documented: true, Source: "spawn request"}
	} else if p.Effort != "" && !p.ModelRequested {
		a.Effort = Value{Present: true, Text: p.Effort, Origin: event.Observed, Documented: true, Source: "agent effort"}
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
		Input: p.Input, Question: ev.Kind == event.Question, ForSession: p.ForSession, Problem: p.Problem, Raw: ev.Raw, Status: Pending, RequestedAt: ev.At}
	s.Approvals[id] = a
	s.Queue = append(s.Queue, id)
	if ev.Agent == "" {
		s.dropStep(ev.At) // a request is a step's first output (plan 22, criterion 1.5)
	}
	s.Conversation = append(s.Conversation, Entry{Seq: ev.Seq, Role: "approval", Agent: ev.Agent, MessageID: id, Text: p.Tool, Origin: ev.Origin, At: ev.At})
	if p.Problem != "" {
		// An unreadable request is denied, never allowed (INT 4 fallback),
		// in the CLI's word: Codex declines (reliability.md, Approvals).
		deny := "deny"
		if s.Context.Provider == "codex" {
			deny = "decline"
		}
		a.Status, a.Answer, a.AnsweredAt = Denied, deny, ev.At
		if p.ID == "" {
			return nil
		}
		return []Command{{Kind: CmdAnswer, ID: p.ID, Decision: deny}}
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
			a.Status, a.StatusOrigin, a.EndedAt, a.EndStage = AgentRunning, ev.Origin, time.Time{}, 0
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
			// Codex 0.163.0 ends a child on its parent first, then with the
			// child's own turn end (P7b): a matching end only confirms it.
			if p.Outcome != "completed" || a.Status != AgentCompleted {
				a.History = append(a.History, "turn end after end")
			}
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
		s.endThoughts(ev.At, p.Outcome) // a phase must not outlive a turn end, whatever the machine says
		return nil
	}
	// A turn the CLI completed stays completed, Ctrl+C or not: the interrupt
	// came after its end (events-and-state.md, Turn).
	outcome := p.Outcome
	if s.Turn.Phase == Interrupting && outcome != "completed" {
		outcome = "interrupted"
	}
	s.Turn.Phase, s.Turn.Outcome, s.Turn.ForceStopOffered = Idle, outcome, false
	s.endThoughts(ev.At, outcome)
	if outcome == "failed" && p.Error != "" {
		s.Errors = appendBounded(s.Errors, ErrorEntry{Seq: ev.Seq, At: ev.At, Message: "turn failed: " + p.Error}, MaxErrors)
		// The error as received, in B (ui-design.md, States: Error); B shows
		// its first line, the errors list in D the whole of it.
		s.Conversation = append(s.Conversation, Entry{Seq: ev.Seq, Role: "error", Text: p.Error, Origin: ev.Origin, At: ev.At})
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
		if m.Derived {
			v.Origin = event.Derived
		}
		if ev.Agent != "" {
			a := s.ensureAgent(ev.Agent, ev)
			if m.Name == "tokens.total" {
				a.Tokens = v
			}
			continue
		}
		if m.Reason != "" {
			// N/A for that reason; the last value and its time stay, for
			// "read failed" (reliability.md, Telemetry sources).
			prev := s.Metrics[m.Name]
			v = Value{Reason: m.Reason, Source: m.Source, Documented: m.Documented, Origin: v.Origin, ReasonAt: ev.At,
				Last: prev.Present || prev.Last, V: prev.V, Text: prev.Text, Unit: prev.Unit, ReceivedAt: prev.ReceivedAt}
		}
		s.Metrics[m.Name] = v
		if m.Reason == "" {
			switch m.Name {
			case event.ThinkingEstimate:
				s.estimate(ev, m.Value)
			case event.ReasoningLast:
				s.reasoningTokens(ev, m.Value)
			}
		}
	}
	s.deriveTokens(ev)
}

// deriveTokens keeps Codex's "tokens.session": the main thread's last total
// plus each child thread's. Claude Code reports its running total itself
// (result.modelUsage).
func (s *State) deriveTokens(ev event.Event) {
	main, ok := s.Metrics["tokens.total"]
	if !ok {
		return
	}
	sum := main.V
	for _, k := range s.Order {
		if a := s.Agents[k]; a.Kind == "thread" && a.Tokens.Present {
			sum += a.Tokens.V
		}
	}
	s.Metrics["tokens.session"] = Value{Present: true, V: sum, Unit: "tokens", Origin: event.Derived, Source: "sum of each thread's last tokens.total", Documented: true, ReceivedAt: ev.At}
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
	if s.Applied == 0 {
		s.FirstAt = ev.At
	}
	s.Applied++
	s.Log = appendBounded(s.Log, ev, MaxLog)
}

func (s *State) note(ev event.Event, msg string) {
	s.Notes = appendBounded(s.Notes, "#"+itoa(ev.Seq)+" "+string(ev.Kind)+": "+msg, MaxErrors)
}

// appendBounded keeps the last max items. The window slides over its array,
// which append replaces only once full, copying max items: a full list costs
// a few copies per item, not max (test 11). Items already in it never change.
func appendBounded[T any](list []T, v T, max int) []T {
	list = append(list, v)
	if len(list) > max {
		list = list[len(list)-max:]
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

// inForce records the mode the CLI reports in force; a request for it is
// met.
func (s *State) inForce(mode string) {
	s.Session.ApprovalMode = mode
	if s.Session.ApprovalRequested == mode {
		s.Session.ApprovalRequested = ""
	}
}

// approvalMode applies a mode report: in force, an answer to the request
// (met or not), or a refusal, which leaves the mode as it was (plan 12,
// item 2).
func (s *State) approvalMode(ev event.Event, p event.Mode) {
	if p.Refused {
		s.Session.ApprovalRequested = ""
		list := s.messages(ev.Agent)
		*list = append(*list, Entry{Seq: ev.Seq, Role: "notice", Agent: ev.Agent, Text: p.Text, Origin: ev.Origin, At: ev.At})
		return
	}
	s.inForce(p.Mode)
	if p.Sandbox != "" {
		s.Session.Sandbox = p.Sandbox
	}
	if p.Answers {
		s.Session.ApprovalRequested = ""
	}
}

// verdict marks a tool call decided by an auto-review: only the classifier
// is one; any other reason adds nothing.
func (s *State) verdict(ev event.Event, p event.Verdict) {
	t := s.Tools[p.ToolID]
	if t == nil || p.ReasonType != "classifier" {
		return
	}
	switch p.Decision {
	case "accept":
		t.Reviewed = "approved by auto-review"
	case "reject":
		t.Reviewed = "denied by auto-review"
	}
}

// review is one auto-review's entry in zone B: one per review id, updated in
// place from started to completed.
func (s *State) review(ev event.Event, p event.Review) {
	text := reviewText(p)
	list := s.messages(ev.Agent)
	for i := len(*list) - 1; i >= 0; i-- {
		if e := &(*list)[i]; e.Role == "review" && e.MessageID == p.ReviewID {
			e.Text, e.Status = text, p.Status
			return
		}
	}
	*list = append(*list, Entry{Seq: ev.Seq, Role: "review", Agent: ev.Agent, MessageID: p.ReviewID, Text: text, Status: p.Status, Origin: ev.Origin, At: ev.At})
}

func reviewText(p event.Review) string {
	if p.Status == "inProgress" {
		return "auto-review in progress: " + p.Command
	}
	word := p.Status
	switch p.Status {
	case "timedOut":
		word = "timed out"
	case "":
		word = "reported"
	}
	t := "auto-review " + word
	if p.Risk != "" {
		t += " · risk " + p.Risk
	}
	if p.Authorization != "" {
		t += " · authorization " + p.Authorization
	}
	if p.Rationale != "" {
		t += ": " + p.Rationale
	}
	return t
}

// The orchestrator's thinking phases (plan 22, brief amendment 18;
// events-and-state.md, Conversation record). Each is an entry of role
// "thinking" with its Thought, appended where the phase starts, so it comes
// before the text and the calls of its step. Only the orchestrator's events
// make one: a subagent's or a child thread's thinking is no entry of zone B
// (plan 22, C6 and X4). Every time is the event's.

// The sources a Thought records for its exact count (plan 22, C3 and X2).
const (
	sourceMessageDelta = "message_delta usage.output_tokens_details.thinking_tokens"
	sourceTokenUsage   = "thread/tokenUsage/updated last.reasoningOutputTokens"
)

// marker applies a stream marker: a message's end, and the starts, ends and
// counts of the orchestrator's thinking, which Claude Code's blocks and
// message_delta carry, and Codex's inferred steps and reasoning items.
func (s *State) marker(ev event.Event, p event.Marker) {
	if p.Name == "message_stop" {
		if e := s.entry(ev.Agent, p.MessageID); e != nil {
			e.Streaming = false
		}
		if ev.Agent == "" {
			s.closeBlocks(p.MessageID, false, ev.At)
		}
		return
	}
	if ev.Agent != "" {
		return
	}
	switch p.Name {
	case "message_start":
		s.closeBlocks(p.MessageID, true, ev.At)
	case "content_block_start":
		if p.Block == "thinking" || p.Block == "redacted_thinking" {
			s.blockStarted(ev, p)
		}
	case "content_block_stop":
		s.blockStopped(ev, p)
	case "message_delta":
		if p.HasThinking {
			s.messageCount(ev, p)
		}
		s.closeBlocks(p.MessageID, false, ev.At)
	case event.StepStarted:
		s.stepStarted(ev)
	case "item_started":
		switch p.Block {
		case "thinking":
			s.reasoningStarted(ev, p)
		case "text":
			s.dropStep(ev.At) // a message is a step's first output (plan 22, criterion 1.5)
		}
	case "item_completed":
		if p.Block == "thinking" {
			s.reasoningCompleted(ev, p)
		}
	}
}

// block is the entry of Claude Code's thinking block index in message id, nil
// when none came since the last prompt. A message's entries all come after
// the previous message's, so the search stops at the first thinking entry of
// another message.
func (s *State) block(id string, index int) *Entry {
	for i := len(s.Conversation) - 1; i >= 0; i-- {
		e := &s.Conversation[i]
		if e.Role == "user" {
			return nil
		}
		if t := e.Thought; e.Role == "thinking" && t != nil && !t.Inferred {
			if e.MessageID != id {
				return nil
			}
			if t.Block == index {
				return e
			}
		}
	}
	return nil
}

// blockStarted opens the phase of a thinking block, observed: its start is
// the marker's (plan 22, C1). A block already started, same message and
// index, is a repeated line and opens nothing; so is one after the session
// crashed, which no end would ever close (plan 22, the review).
func (s *State) blockStarted(ev event.Event, p event.Marker) {
	if p.MessageID != "" && s.block(p.MessageID, p.Index) != nil {
		s.note(ev, "thinking block started twice: "+p.MessageID+" "+strconv.Itoa(p.Index))
		return
	}
	if s.Session.Phase == Crashed {
		s.note(ev, "thinking block after the session crashed: no phase opened")
		return
	}
	s.Conversation = append(s.Conversation, Entry{Seq: ev.Seq, Role: "thinking", MessageID: p.MessageID, Origin: ev.Origin, Streaming: true, At: ev.At,
		Thought: s.newThought(Thought{Block: p.Index, Start: ev.At, Confirmed: true})})
}

// newThought numbers phase t, the record's next (State.Thoughts): its ID
// stays unique across a resume, where events are numbered from 1 again.
func (s *State) newThought(t Thought) *Thought {
	s.Thoughts++
	t.ID = s.Thoughts
	return &t
}

// closeBlocks ends the live thinking blocks of message id, or of every
// message but id when others, at at, their end inferred: a block whose stop
// never came ends with its message, or when the next message starts, so it
// never stays live beside the next one (plan 22, the review).
func (s *State) closeBlocks(id string, others bool, at time.Time) {
	for i := len(s.Conversation) - 1; i >= 0; i-- {
		e := &s.Conversation[i]
		if e.Role == "user" {
			return
		}
		t := e.Thought
		if e.Role != "thinking" || t == nil || t.Inferred || !t.Live() || (e.MessageID == id) == others {
			continue
		}
		t.End, t.EndInferred, e.Streaming = at, true, false
	}
}

// blockStopped ends the phase of the block that stopped: the stop of the same
// message and index (plan 22, C1).
func (s *State) blockStopped(ev event.Event, p event.Marker) {
	if e := s.block(p.MessageID, p.Index); e != nil && e.Thought.Live() {
		e.Thought.End, e.Streaming = ev.At, false
	}
}

// estimate applies Claude Code's running estimate (system/thinking_tokens,
// cumulative: plan 22, C2) to the live thinking block, the last one: a value
// not higher than the phase's changes nothing, so it never goes down, and with
// no phase live there is none to take it.
func (s *State) estimate(ev event.Event, v float64) {
	for i := len(s.Conversation) - 1; i >= 0; i-- {
		e := &s.Conversation[i]
		if e.Role == "user" {
			return
		}
		if t := e.Thought; e.Role == "thinking" && t.Live() && t.Confirmed && !t.Inferred {
			if n := int64(v); n > t.Estimate {
				t.EstimateFrom, t.Estimate, t.EstimateAt = t.Estimate, n, ev.At
			}
			return
		}
	}
}

// messageCount applies a message_delta's exact thinking count (plan 22, C3)
// to the message's phases: to its only phase, or to the last of several,
// marked shared. A message cut short (stop_reason null, C7) marks every phase
// interrupted, and ends one still live, but gives its last phase no exact
// count: the 0 the CLI sends with it is the usage the message had when it
// began (P303a, P2), kept apart as reported in Cut and CutTokens (plan 22,
// correction C1). A phase whose turn or session ended with no count takes
// none later (correction C2, both CLIs).
func (s *State) messageCount(ev event.Event, p event.Marker) {
	var last *Entry
	k := 0
	for i := len(s.Conversation) - 1; i >= 0; i-- {
		e := &s.Conversation[i]
		if e.Role == "user" {
			break
		}
		t := e.Thought
		if e.Role != "thinking" || t == nil || t.Inferred {
			continue
		}
		if e.MessageID != p.MessageID {
			break
		}
		k++
		if last == nil {
			last = e
		}
		if p.StopReason == "" && !t.Closed {
			t.Outcome = "interrupted"
			if t.Live() {
				t.End, t.EndInferred, e.Streaming = ev.At, true, false
			}
		}
	}
	if last == nil || last.Thought.Closed {
		return
	}
	t := last.Thought
	if p.StopReason == "" {
		t.Cut, t.CutTokens = true, p.Thinking
		return
	}
	t.Tokens, t.HasTokens, t.TokensAt, t.Source = p.Thinking, true, ev.At, sourceMessageDelta
	if k > 1 {
		t.Shared = k
	}
}

// openStep is the Codex model step that runs and no reasoning item has
// confirmed: the last thinking entry, when live, inferred and unconfirmed. Any
// other output of the orchestrator ends one (dropStep), so the search stops at
// the first entry such an output made.
func (s *State) openStep() *Entry {
	for i := len(s.Conversation) - 1; i >= 0; i-- {
		e := &s.Conversation[i]
		switch {
		case e.Role == "thinking":
			if t := e.Thought; t.Live() && t.Inferred && !t.Confirmed {
				return e
			}
			return nil
		case e.Role == "user", e.Agent == "" && (e.Role == "assistant" || e.Role == "tool" || e.Role == "approval"):
			return nil
		}
	}
	return nil
}

// stepStarted opens the phase of a Codex model step, inferred: the adapter
// marks where each begins (plan 22, X5; event.StepStarted). Only a turn at
// work has steps. A step that made a spawn and no reasoning is still open when
// the next marker comes, and stays open into it (criterion 1.5).
func (s *State) stepStarted(ev event.Event) {
	switch s.Turn.Phase {
	case Submitted, Running, WaitingApproval:
	default:
		s.note(ev, "model step started while the turn is "+string(s.Turn.Phase)+": no phase opened")
		return
	}
	if s.openStep() != nil {
		return
	}
	// A reasoning item whose completion never came is over once the next step
	// begins: it ends there, its end inferred, so two phases are never live at
	// once (plan 22, correction C4).
	for i := len(s.Conversation) - 1; i >= 0; i-- {
		e := &s.Conversation[i]
		if e.Role == "user" {
			break
		}
		if t := e.Thought; e.Role == "thinking" && t.Live() && t.Inferred && t.Confirmed {
			t.End, t.EndInferred, t.EndNext, e.Streaming = ev.At, true, true, false
		}
	}
	s.Conversation = append(s.Conversation, Entry{Seq: ev.Seq, Role: "thinking", MessageID: "step:" + itoa(ev.Seq), Origin: stepOrigin(ev), Streaming: true, At: ev.At,
		Thought: s.newThought(Thought{Start: ev.At, Inferred: true})})
}

// stepOrigin is the origin of a Codex step: inferred, as its adapter marks it;
// mock where every event is (events-and-state.md, Origins).
func stepOrigin(ev event.Event) event.Origin {
	if ev.Origin == event.Mock {
		return event.Mock
	}
	return event.Inferred
}

// dropStep ends the open Codex step as dropped, at at: its first output was
// not reasoning, so it leaves no row (plan 22, design 4). A step a reasoning
// item confirmed is never dropped.
func (s *State) dropStep(at time.Time) {
	if e := s.openStep(); e != nil {
		e.Thought.End, e.Thought.Dropped, e.Streaming = at, true, false
	}
}

// reasoning is the phase of Codex reasoning item id, nil when none came since
// the last prompt.
func (s *State) reasoning(id string) *Entry {
	if id == "" {
		return nil
	}
	for i := len(s.Conversation) - 1; i >= 0; i-- {
		e := &s.Conversation[i]
		if e.Role == "user" {
			return nil
		}
		if t := e.Thought; e.Role == "thinking" && t != nil && t.Item == id {
			return e
		}
	}
	return nil
}

// reasoningStarted applies a reasoning item's start: the model reasoned in its
// step (plan 22, X1), which confirms the open step, whose start stays the
// step's, inferred. With none open, a phase whose start is not known. Either
// way the item's own start, observed, is kept (plan 22, amendment 20).
func (s *State) reasoningStarted(ev event.Event, p event.Marker) {
	if s.reasoning(p.MessageID) != nil {
		s.note(ev, "reasoning item started twice: "+p.MessageID)
		return
	}
	if e := s.openStep(); e != nil {
		e.Thought.Confirmed, e.Thought.Item, e.Thought.ItemStart = true, p.MessageID, ev.At
		return
	}
	if s.Session.Phase == Crashed {
		s.note(ev, "reasoning item after the session crashed: no phase opened")
		return
	}
	s.Conversation = append(s.Conversation, Entry{Seq: ev.Seq, Role: "thinking", MessageID: "step:" + itoa(ev.Seq), Origin: ev.Origin, Streaming: true, At: ev.At,
		Thought: s.newThought(Thought{Item: p.MessageID, Inferred: true, Confirmed: true, Start: s.lastResponseEnd(), ItemStart: ev.At})})
}

// lastResponseEnd is when the turn's last model response ended, by its usage
// update; zero when none came since the prompt. A reasoning item no step
// marker announced (the step before it made no call, so the adapter began
// none) belongs to the request that followed that response at once, so its
// step's time runs from there, "up to" like every other Codex phase's (plan
// 22, correction C5).
func (s *State) lastResponseEnd() time.Time {
	v, ok := s.Metrics[event.ReasoningLast]
	if !ok || !v.Present {
		return time.Time{}
	}
	for i := len(s.Conversation) - 1; i >= 0; i-- {
		if e := s.Conversation[i]; e.Role == "user" {
			if !v.ReceivedAt.Before(e.At) {
				return v.ReceivedAt
			}
			break
		}
	}
	return time.Time{}
}

// reasoningCompleted ends the phase of the reasoning item that completed; the
// item's two lines come 0 to 10 ms apart when no summary was asked for (plan
// 22, X1), 2.5 to 3.4 s apart when one was (P22s). The item's summary, Codex's
// own text, goes to the phase it belongs to, still live or not; a completion
// without one never clears it (plan 22, amendment 20).
func (s *State) reasoningCompleted(ev event.Event, p event.Marker) {
	e := s.reasoning(p.MessageID)
	if e == nil {
		return
	}
	if p.Summary != "" {
		e.Thought.Summary = p.Summary
	}
	if e.Thought.Live() {
		e.Thought.End, e.Streaming = ev.At, false
	}
}

// reasoningTokens applies a usage update's reasoning tokens
// (last.reasoningOutputTokens, the model response it closes: plan 22, X2) to
// the orchestrator's confirmed thoughts that no update has counted: to the only
// one, or to the last of several, marked shared. Each is counted, so the next
// update goes to the thoughts after them. With none, nothing: a step with no
// reasoning item reports 0. Only the orchestrator's updates come here; a child
// thread's goes to its agent (telemetry).
func (s *State) reasoningTokens(ev event.Event, v float64) {
	var last *Entry
	k := 0
	for i := len(s.Conversation) - 1; i >= 0; i-- {
		e := &s.Conversation[i]
		if e.Role == "user" {
			break
		}
		t := e.Thought
		if e.Role != "thinking" || t == nil || !t.Inferred || !t.Confirmed || t.Counted || t.Closed {
			continue
		}
		k++
		if last == nil {
			last = e
		}
		t.Counted = true
	}
	if last == nil {
		return
	}
	t := last.Thought
	t.Tokens, t.HasTokens, t.TokensAt, t.Source = int64(v), true, ev.At, sourceTokenUsage
	if k > 1 {
		t.Shared = k
	}
}

// endThoughts ends the orchestrator's live thinking phases at at, the end of
// the turn or of the session (plan 22, criterion 1.6): a Codex step no
// reasoning item confirmed is dropped, any other phase takes the outcome when
// it is interrupted or failed. A session ended by an action has no time: the
// last event's. Every phase of the exchange that has no exact count by then
// is closed: none can come, and none is applied to it later (plan 22,
// correction C2).
func (s *State) endThoughts(at time.Time, outcome string) {
	if at.IsZero() {
		if n := len(s.Log); n > 0 {
			at = s.Log[n-1].At
		}
	}
	for i := len(s.Conversation) - 1; i >= 0; i-- {
		e := &s.Conversation[i]
		if e.Role == "user" {
			return
		}
		t := e.Thought
		if e.Role == "thinking" && t != nil && !t.HasTokens {
			t.Closed = true
		}
		if e.Role != "thinking" || !t.Live() {
			continue
		}
		t.End, t.EndInferred, e.Streaming = at, true, false
		switch {
		case t.Inferred && !t.Confirmed:
			t.Dropped = true
		case outcome == "interrupted" || outcome == "failed":
			t.Outcome = outcome
		}
	}
}

// namedImages are a message's images with their files' names: the line a
// CLI was written carries no name, so while the dashboard's own message
// waits to be accepted, its attachments name the images, in order, when
// they are as many (plan 31).
func (s *State) namedImages(imgs []event.Image) []event.Image {
	if len(imgs) == 0 {
		return nil
	}
	out := append([]event.Image(nil), imgs...)
	if d := s.Turn.DraftImages; s.Turn.Phase == Submitted && len(d) == len(out) {
		for i := range out {
			if out[i].Name == "" {
				out[i].Name = d[i].Name
			}
		}
	}
	return out
}
