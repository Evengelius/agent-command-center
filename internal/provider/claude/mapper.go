// Package claude is the Claude Code adapter: `claude -p` with stream-json in
// and out and the stdio control protocol (doc/architecture.md, Claude Code
// adapter). The control-request wire form and `--permission-prompt-tool
// stdio` are undocumented (INT 1, INT 4).
package claude

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/Evengelius/agent-command-center/internal/event"
)

// ProviderID is the adapter id.
const ProviderID = "claude-code"

// Mapper turns Claude Code stream-json lines into normalised events. It keeps
// the little state the wire form needs (which request an id belongs to,
// whether an interrupt is outstanding), so the same lines always give the
// same events.
type Mapper struct {
	session     string
	pending     map[string]string // request_id -> subtype of our own requests
	interrupted bool
	// The main thread's last message: its model and the tokens its usage
	// puts in the context, for the derived context (reliability.md,
	// Telemetry sources: Context).
	lastModel   string
	lastContext float64
	models      map[string]string // each subagent's last model sent (plan 5, item 7)
	// Each agent's last message_start id, by agent key ("" for the main
	// thread): the id the agent's following stream events take (plan 21,
	// item 3).
	streamIDs map[string]string
	// The usage reads the dashboard asked for with "/usage" and the CLI has
	// not answered yet (plan 24, amendment 23; P24u): the CLI answers each
	// in order, as a turn of its own, which the mapper keeps out of the
	// conversation: usageStarted once its turn start was seen, usageOpen
	// once its answer was.
	usageReads   int
	usageStarted bool
	usageOpen    bool
	// The session's models and their settings (plan 34, models.go): the
	// initialize response's models; the model the last set_model asked for
	// and the one set_model last took; the changes written and not yet
	// answered; whether a change's answer calls for a read, and the reads
	// written for one (by request id); the last read; the effort variable
	// in the CLI's environment.
	catalogue   []event.ModelEntry
	asked       string
	chosen      string
	changes     int
	readDue     bool
	readAnswers bool
	answering   map[string]bool
	inForce     event.Settings
	effortVar   string
}

// NewMapper returns an empty mapper.
func NewMapper() *Mapper {
	return &Mapper{pending: map[string]string{}, models: map[string]string{}, streamIDs: map[string]string{}, answering: map[string]bool{}}
}

type line struct {
	Type            string          `json:"type"`
	Subtype         string          `json:"subtype"`
	SessionID       string          `json:"session_id"`
	ParentToolUseID *string         `json:"parent_tool_use_id"`
	AgentID         string          `json:"agent_id"`
	Message         json.RawMessage `json:"message"`
	Event           json.RawMessage `json:"event"`
	APIMessageID    string          `json:"api_message_id"`
	ToolResultMeta  json.RawMessage `json:"tool_result_meta"`
	UsageReport     json.RawMessage `json:"usage_report"` // on the assistant message answering "/usage" (P24u)
	RequestID       string          `json:"request_id"`
	Request         json.RawMessage `json:"request"`
	Response        json.RawMessage `json:"response"`
	Error           json.RawMessage `json:"error"`
}

// Out maps one line read from the CLI's stdout.
func (m *Mapper) Out(raw []byte) []event.Event {
	var l line
	if err := json.Unmarshal(raw, &l); err != nil {
		return []event.Event{m.ev(event.Error, event.Observed, "", raw, event.Problem{Message: "unreadable line from Claude Code: " + err.Error()})}
	}
	if l.SessionID != "" && l.SessionID != "default" {
		m.session = l.SessionID
	}
	agent := ""
	if l.ParentToolUseID != nil && *l.ParentToolUseID != "" {
		agent = l.AgentID
		if agent == "" {
			agent = "tool:" + *l.ParentToolUseID
		}
	}
	switch l.Type {
	case "system":
		return m.system(l, raw)
	case "stream_event":
		if agent == "" && m.usageReads > 0 && !m.usageOpen {
			// A streamed part of the dashboard's own usage read (P24u showed
			// none): never a message of the conversation.
			return []event.Event{m.ev(event.StatusChanged, event.Observed, "", raw, event.Status{Detail: "usage report: streaming"})}
		}
		return m.stream(l, agent, raw)
	case "assistant":
		return m.assistant(l, agent, raw)
	case "user":
		return m.user(l, agent, raw)
	case "rate_limit_event":
		return m.rateLimit(raw)
	case "result":
		return m.result(raw)
	case "control_request":
		return m.controlRequest(l, raw)
	case "control_response":
		return m.controlResponse(l, raw)
	case "control_cancel_request":
		return []event.Event{m.ev(event.ApprovalResolved, event.Observed, "", raw, event.Approval{ID: l.RequestID, Decision: "expired"})}
	}
	return []event.Event{m.unknown(raw, l.Type, l.Subtype)}
}

// In maps one line the dashboard wrote to the CLI's stdin: the owner's
// message, an approval answer, or a control request of its own.
func (m *Mapper) In(raw []byte) []event.Event {
	var l line
	if err := json.Unmarshal(raw, &l); err != nil {
		return []event.Event{m.ev(event.Error, event.Observed, "", raw, event.Problem{Message: "unreadable line written to Claude Code: " + err.Error()})}
	}
	switch l.Type {
	case "user":
		var msg struct {
			Content json.RawMessage `json:"content"`
		}
		_ = json.Unmarshal(l.Message, &msg)
		imgs := contentImages(msg.Content)
		if len(imgs) > 0 {
			raw = withoutImageData(raw) // the event keeps no image bytes (plan 31)
		}
		return []event.Event{m.ev(event.UserMessage, event.Observed, "", raw, event.User{Text: contentText(msg.Content), Images: imgs})}
	case "control_request":
		var req struct {
			Subtype string `json:"subtype"`
		}
		_ = json.Unmarshal(l.Request, &req)
		m.pending[l.RequestID] = req.Subtype
		switch req.Subtype {
		case "interrupt":
			m.interrupted = true
		case "set_model", "apply_flag_settings":
			m.change(req.Subtype, l.Request)
		case "get_settings":
			if m.readDue { // the read a change's answer called for (plan 34)
				m.answering[l.RequestID], m.readDue = m.readAnswers, false
			}
		}
		return []event.Event{m.ev(event.Sent, event.Observed, "", raw, event.Write{What: req.Subtype, RequestID: l.RequestID})}
	case "control_response":
		var resp struct {
			RequestID string `json:"request_id"`
			Response  struct {
				Behavior string `json:"behavior"`
			} `json:"response"`
		}
		_ = json.Unmarshal(l.Response, &resp)
		return []event.Event{m.ev(event.ApprovalResolved, event.Observed, "", raw, event.Approval{ID: resp.RequestID, Decision: resp.Response.Behavior})}
	}
	return []event.Event{m.ev(event.Sent, event.Observed, "", raw, event.Write{What: l.Type})}
}

func (m *Mapper) ev(k event.Kind, o event.Origin, agent string, raw []byte, p any) event.Event {
	return event.Event{Provider: ProviderID, Session: m.session, Kind: k, Origin: o, Agent: agent, Payload: p, Raw: string(raw)}
}

func (m *Mapper) unknown(raw []byte, typ, sub string) event.Event {
	t := typ
	if sub != "" {
		t += "/" + sub
	}
	return m.ev(event.Unrecognised, event.Observed, "", raw, event.Unknown{Type: t})
}

func (m *Mapper) system(l line, raw []byte) []event.Event {
	switch l.Subtype {
	case "init":
		var s struct {
			Cwd            string `json:"cwd"`
			Model          string `json:"model"`
			PermissionMode string `json:"permissionMode"`
			Version        string `json:"claude_code_version"`
			APIKeySource   string `json:"apiKeySource"`
			Plugins        []struct {
				Name string `json:"name"`
			} `json:"plugins"`
		}
		_ = json.Unmarshal(raw, &s)
		st := event.Started{SessionID: l.SessionID, Model: s.Model, Cwd: s.Cwd, PermissionMode: s.PermissionMode, Version: s.Version, ApprovalMode: approvalMode(s.PermissionMode)}
		for _, p := range s.Plugins {
			st.Plugins = append(st.Plugins, p.Name)
		}
		out := []event.Event{m.ev(event.SessionStarted, event.Observed, "", raw, st)}
		if s.APIKeySource != "" && s.APIKeySource != "none" {
			// An API key has no subscription window (reliability.md,
			// Telemetry sources: 5-hour, 7-day quota).
			const why = "API key: no subscription window"
			out = append(out, m.ev(event.Telemetry, event.Observed, "", raw, event.Metrics{Items: []event.Metric{
				{Name: "quota.5h", Documented: true, Source: "system/init.apiKeySource", Reason: why},
				{Name: "quota.7d", Documented: true, Source: "system/init.apiKeySource", Reason: why},
			}}))
		}
		return out
	case "status":
		var s struct {
			Status         *string `json:"status"`
			PermissionMode string  `json:"permissionMode"`
		}
		_ = json.Unmarshal(raw, &s)
		st := event.Status{Detail: s.PermissionMode}
		if s.Status != nil {
			st.Status = *s.Status
		}
		out := []event.Event{m.ev(event.StatusChanged, event.Observed, "", raw, st)}
		if s.PermissionMode != "" {
			out = append(out, m.ev(event.ApprovalMode, event.Observed, "", raw, event.Mode{Mode: approvalMode(s.PermissionMode)}))
		}
		return out
	case "session_state_changed":
		var s struct {
			State string `json:"state"`
		}
		_ = json.Unmarshal(raw, &s)
		if s.State == "running" {
			if m.usageReads > 0 && !m.usageStarted {
				// The dashboard's own usage read runs as a turn of the CLI's:
				// no turn of the owner's opens for it (plan 24).
				m.usageStarted = true
				return []event.Event{m.ev(event.StatusChanged, event.Observed, "", raw, event.Status{Detail: "usage report: reading"})}
			}
			return []event.Event{m.ev(event.TurnStarted, event.Observed, "", raw, nil)}
		}
		return []event.Event{m.ev(event.StatusChanged, event.Observed, "", raw, event.Status{Status: s.State})}
	case "compact_boundary":
		return []event.Event{m.ev(event.StatusChanged, event.Observed, "", raw, event.Status{Status: "compacted"})}
	case "api_retry":
		return []event.Event{m.ev(event.Retry, event.Observed, "", raw, event.Problem{Message: "API retry", WillRetry: true})}
	case "commands_changed":
		// The slash commands and skills the session offers (claude-code.md,
		// section 6, "Secondary"; P3 and P3b). No MVP zone lists them
		// (amendment 4), so it is known and logged with no status: nothing
		// on screen changes, and it is not counted as unrecognised.
		var s struct {
			Commands []json.RawMessage `json:"commands"`
		}
		_ = json.Unmarshal(raw, &s)
		return []event.Event{m.ev(event.StatusChanged, event.Observed, "", raw, event.Status{Detail: fmt.Sprintf("%d commands available", len(s.Commands))})}
	case "permission_check_status":
		// A tool's permission being checked, "checking" then "done"
		// (undocumented; P13c, Claude Code 2.1.302). The verdict comes with
		// the tool's result (verdicts), so it is known and logged with no
		// status, as commands_changed.
		var s struct {
			ToolUseID string `json:"tool_use_id"`
			Status    string `json:"status"`
		}
		_ = json.Unmarshal(raw, &s)
		return []event.Event{m.ev(event.StatusChanged, event.Observed, "", raw, event.Status{Detail: "permission check " + s.Status + ": " + s.ToolUseID})}
	case "permission_denied":
		return []event.Event{m.ev(event.Error, event.Observed, "", raw, event.Problem{Message: "permission denied"})}
	case "thinking_tokens":
		// The running estimate of the open thinking block's tokens, cumulative,
		// main thread only, with no agent field (plan 22, C2). Recorded in
		// testdata/probes/p303b-claude-subagent-approval.jsonl (lines 12, 15
		// and 17: 50, 300, 650) and p12c-block-claude-auto-boundary-edit.jsonl
		// (lines 73 to 93: the second phase's 50 to 2,700). Documented: the
		// Agent SDK TypeScript reference lists it as SDKThinkingTokensMessage,
		// "a running estimate of the thinking tokens generated so far in the
		// current block", Claude Code 2.1.153 or later
		// (code.claude.com/docs/en/agent-sdk/typescript, checked 2026-10-21).
		// An estimate, never the exact count (C5).
		var s struct {
			Estimated float64 `json:"estimated_tokens"`
		}
		_ = json.Unmarshal(raw, &s)
		return []event.Event{m.ev(event.Telemetry, event.Observed, "", raw, event.Metrics{Items: []event.Metric{{Name: event.ThinkingEstimate, Value: s.Estimated, Unit: "tokens", Documented: true, Source: "system/thinking_tokens"}}})}
	case "hook_started", "post_turn_summary", "task_summary":
		// Known, and they give no event: nothing in the MVP draws a hook's start
		// or the turn's summary line (plan 22, amendment 20). P22o, Claude Code
		// 2.1.303 with the owner's SessionStart hook: hook_started at line 3,
		// post_turn_summary at line 55, each counted as unrecognised before.
		// task_summary (undocumented; plan 25, scenario 1 through the logging
		// wrapper on 2.1.303 with Fable 5.1, 2026-10-22): once right after
		// task_started, its detail the orchestrator's narration before the
		// Agent call, which zone B already shows; once right after the turn's
		// result, its detail null. The 2.1.303 probes of 2026-10-19 had none.
		return nil
	case "hook_response":
		return m.hookResponse(raw)
	case "task_started", "task_progress", "task_updated", "task_notification":
		return m.task(l, raw)
	case "background_tasks_changed":
		var s struct {
			Tasks []struct {
				TaskID string `json:"task_id"`
			} `json:"tasks"`
		}
		_ = json.Unmarshal(raw, &s)
		keys := []string{}
		for _, t := range s.Tasks {
			keys = append(keys, t.TaskID)
		}
		return []event.Event{m.ev(event.AgentSet, event.Observed, "", raw, event.AgentKeys{Keys: keys})}
	}
	return []event.Event{m.unknown(raw, l.Type, l.Subtype)}
}

// hookResponse maps system/hook_response, a hook's end. A hook that succeeded
// draws nothing: its outcome is "success" and its exit code, when it reports
// one, is 0. Any other end is one observed notice naming the hook, its exit
// code (else its outcome) and the first non-blank line of its stderr, left out
// when stderr is blank, so a failing hook stays visible (plan 22, amendment
// 20; AGENTS.md, keep failures visible). It is a notice, not an error: the CLI
// carries on after a failing hook, so the dashboard must not end the session,
// and a SessionStart hook's end comes before the initialize response (P22o,
// lines 4 and 5), where the reducer takes an error for a failed handshake and
// closes the session (reduce.go, Problem). The fields are those of P22o's line
// 4, the owner's SessionStart hook: hook_name "SessionStart:startup", outcome
// "success", exit_code 0, stderr "". No recording holds a failing hook, so that
// case is synthetic; the outcomes other than "success" are not documented in
// this repository, and the word is carried as sent. A response with no
// exit_code is judged by its outcome alone.
func (m *Mapper) hookResponse(raw []byte) []event.Event {
	var h struct {
		Name     string `json:"hook_name"`
		Outcome  string `json:"outcome"`
		ExitCode *int   `json:"exit_code"`
		Stderr   string `json:"stderr"`
	}
	_ = json.Unmarshal(raw, &h)
	if h.Outcome == "success" && (h.ExitCode == nil || *h.ExitCode == 0) {
		return nil
	}
	msg := "hook"
	if h.Name != "" {
		msg += " " + h.Name
	}
	msg += " failed: "
	switch {
	case h.ExitCode != nil && *h.ExitCode != 0:
		msg += fmt.Sprintf("exit %d", *h.ExitCode)
	case h.Outcome != "":
		msg += "outcome " + h.Outcome
	default:
		msg += "no outcome reported"
	}
	if l := firstLine(h.Stderr); l != "" {
		msg += ": " + l
	}
	return []event.Event{m.ev(event.StatusChanged, event.Observed, "", raw, event.Status{Status: "notice", Detail: msg})}
}

// firstLine is the first line of s with text in it, trimmed; "" when none.
func firstLine(s string) string {
	for l := range strings.SplitSeq(s, "\n") {
		if l = strings.TrimSpace(l); l != "" {
			return l
		}
	}
	return ""
}

func (m *Mapper) task(l line, raw []byte) []event.Event {
	var t struct {
		TaskID       string `json:"task_id"`
		ParentTaskID string `json:"parent_task_id"`
		Description  string `json:"description"`
		SubagentType string `json:"subagent_type"`
		TaskType     string `json:"task_type"`
		Ambient      bool   `json:"ambient"`
		Backgrounded bool   `json:"is_backgrounded"`
		Status       string `json:"status"`
		Summary      string `json:"summary"`
		LastTool     string `json:"last_tool_name"`
		Usage        *struct {
			TotalTokens int64 `json:"total_tokens"`
			ToolUses    int   `json:"tool_uses"`
		} `json:"usage"`
		Patch struct {
			Status string `json:"status"`
		} `json:"patch"`
	}
	_ = json.Unmarshal(raw, &t)
	a := event.Agent{Key: t.TaskID, Parent: t.ParentTaskID, Label: t.SubagentType, Description: t.Description,
		Backgrounded: t.Backgrounded, LastTool: t.LastTool, Summary: t.Summary}
	if t.Usage != nil {
		a.Tokens, a.HasTokens, a.ToolUses = t.Usage.TotalTokens, true, t.Usage.ToolUses
	}
	switch l.Subtype {
	case "task_started":
		a.Type = "subagent"
		if t.TaskType == "local_workflow" {
			a.Type = "workflow"
		}
		if t.Ambient {
			a.Type = "ambient"
		}
		a.Status = "running"
		return []event.Event{m.ev(event.AgentStarted, event.Observed, t.TaskID, raw, a)}
	case "task_progress":
		return []event.Event{m.ev(event.AgentProgress, event.Observed, t.TaskID, raw, a)}
	case "task_updated":
		a.Status = t.Patch.Status
		return []event.Event{m.ev(agentKind(t.Patch.Status), event.Observed, t.TaskID, raw, a)}
	default: // task_notification
		a.Status = t.Status
		return []event.Event{m.ev(agentKind(t.Status), event.Observed, t.TaskID, raw, a)}
	}
}

func agentKind(status string) event.Kind {
	switch status {
	case "completed":
		return event.AgentCompleted
	case "failed", "stopped", "killed":
		return event.AgentFailed
	}
	return event.AgentProgress
}

func (m *Mapper) stream(l line, agent string, raw []byte) []event.Event {
	var se struct {
		Type    string `json:"type"`
		Index   int    `json:"index"`
		Message struct {
			ID    string `json:"id"`
			Model string `json:"model"`
		} `json:"message"`
		ContentBlock struct {
			Type string `json:"type"`
		} `json:"content_block"`
		Delta struct {
			Type        string `json:"type"`
			Text        string `json:"text"`
			Thinking    string `json:"thinking"`
			PartialJSON string `json:"partial_json"`
			StopReason  string `json:"stop_reason"`
		} `json:"delta"`
	}
	if err := json.Unmarshal(l.Event, &se); err != nil {
		return []event.Event{m.unknown(raw, l.Type, "")}
	}
	// The message a stream event belongs to: the id of its agent's last
	// message_start (event.message.id, the Anthropic raw stream event carried
	// by the documented SDKPartialAssistantMessage.event). The line's
	// top-level api_message_id is UNDOCUMENTED: it is not in
	// SDKPartialAssistantMessage (code.claude.com/docs/en/agent-sdk/typescript,
	// read 2026-10-20), though present in every recording P1 to P303e and equal
	// there to the last message_start's id (claude-code.md, stream_event). It is
	// used only when no message_start came first for this agent (plan 21,
	// item 3).
	if se.Type == "message_start" {
		m.streamIDs[agent] = se.Message.ID // an empty id drops a stale one
	}
	id := m.streamIDs[agent]
	if id == "" {
		id = l.APIMessageID
	}
	switch se.Type {
	case "content_block_delta":
		d := event.Delta{MessageID: id, Index: se.Index}
		switch se.Delta.Type {
		case "text_delta":
			d.Block, d.Text = "text", se.Delta.Text
		case "thinking_delta":
			d.Block, d.Text = "thinking", se.Delta.Thinking
		case "input_json_delta":
			d.Block, d.Text = "tool_input", se.Delta.PartialJSON
		case "signature_delta":
			d.Block = "signature"
		default:
			d.Block = se.Delta.Type
		}
		return []event.Event{m.ev(event.MessageDelta, event.AI, agent, raw, d)}
	case "message_start":
		return []event.Event{m.ev(event.MessageMarker, event.Observed, agent, raw, event.Marker{MessageID: id, Name: se.Type, Model: se.Message.Model})}
	case "content_block_start", "content_block_stop":
		// The block's index in its message: a stop finds its start by it (plan
		// 22, C1). A thinking block is always index 0 in the recordings, a
		// later block 1 or more: p303b-claude-subagent-approval.jsonl lines
		// 11 and 22 (a thinking block), 23 and 79 (the tool use at 1 after it);
		// p12c-block-claude-auto-boundary-edit.jsonl lines 99, 107 and 108
		// (text at 1, a tool use at 2).
		return []event.Event{m.ev(event.MessageMarker, event.Observed, agent, raw, event.Marker{MessageID: id, Name: se.Type, Block: se.ContentBlock.Type, Index: se.Index})}
	case "message_stop":
		return []event.Event{m.ev(event.MessageMarker, event.Observed, agent, raw, event.Marker{MessageID: id, Name: se.Type, Block: se.ContentBlock.Type})}
	case "message_delta":
		thinking, has := thinkingTokens(l.Event)
		return []event.Event{m.ev(event.MessageMarker, event.Observed, agent, raw, event.Marker{MessageID: id, Name: se.Type, StopReason: se.Delta.StopReason, Thinking: thinking, HasThinking: has})}
	}
	return []event.Event{m.unknown(raw, l.Type, se.Type)}
}

// thinkingTokens reads a message_delta's usage.output_tokens_details.
// thinking_tokens, the message's exact thinking-token count (plan 22, C3):
// 719, 281 and 268 on the three messages of
// testdata/probes/p303b-claude-subagent-approval.jsonl (lines 82, 146 and
// 195), 717, 2,852, 17 and 354 on the four of
// p12c-block-claude-auto-boundary-edit.jsonl (lines 65, 121, 154 and 205), 0
// for a message with no thinking. Documented: "When streaming, this
// breakdown appears only on the final message_delta event"
// (platform.claude.com/docs/en/build-with-claude/extended-thinking, checked
// 2026-10-21). has is false when the line gives none, which is never a zero:
// a line without the field, or with another shape, reports no count. It
// reads the line apart from the stream event's other fields so that an odd
// usage cannot turn the message_delta into an unrecognised line.
func thinkingTokens(ev json.RawMessage) (n int64, has bool) {
	var u struct {
		Usage struct {
			Details struct {
				Thinking *float64 `json:"thinking_tokens"`
			} `json:"output_tokens_details"`
		} `json:"usage"`
	}
	if json.Unmarshal(ev, &u) != nil || u.Usage.Details.Thinking == nil {
		return 0, false
	}
	return int64(*u.Usage.Details.Thinking), true
}

type block struct {
	Type      string          `json:"type"`
	Text      string          `json:"text"`
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Input     json.RawMessage `json:"input"`
	ToolUseID string          `json:"tool_use_id"`
	Content   json.RawMessage `json:"content"`
	IsError   bool            `json:"is_error"`
}

func (m *Mapper) assistant(l line, agent string, raw []byte) []event.Event {
	var msg struct {
		ID      string  `json:"id"`
		Model   string  `json:"model"`
		Content []block `json:"content"`
		Usage   *struct {
			Input         float64 `json:"input_tokens"`
			CacheRead     float64 `json:"cache_read_input_tokens"`
			CacheCreation float64 `json:"cache_creation_input_tokens"`
		} `json:"usage"`
	}
	_ = json.Unmarshal(l.Message, &msg)
	// The answer to "/usage" (P24u): its usage_report gives the quota
	// windows; when the dashboard asked, the message itself is not shown.
	report := m.usageReport(l.UsageReport, raw)
	if agent == "" && m.usageReads > 0 {
		m.usageOpen = true
		if report == nil {
			report = []event.Event{m.ev(event.StatusChanged, event.Observed, "", raw, event.Status{Detail: "usage report: the answer holds none"})}
		}
		return report
	}
	if u := msg.Usage; agent == "" && u != nil {
		// What the request put in the context (claude-code.md, section 9:
		// input + cache creation + cache read).
		m.lastModel, m.lastContext = msg.Model, u.Input+u.CacheRead+u.CacheCreation
	}
	var out, pre []event.Event
	if agent != "" && msg.Model != "" && !strings.HasPrefix(msg.Model, "<") && m.models[agent] != msg.Model {
		// A subagent's own model, sent when it first appears or changes; a
		// name in angle brackets marks a message the CLI made, not a model.
		m.models[agent] = msg.Model
		pre = append(pre, m.ev(event.AgentProgress, event.Observed, agent, raw, event.Agent{Key: agent, Model: msg.Model}))
	}
	c := event.Completed{MessageID: msg.ID, Model: msg.Model}
	var texts []string
	hasText := false
	for _, b := range msg.Content {
		switch b.Type {
		case "text":
			texts, hasText = append(texts, b.Text), true
		case "thinking", "redacted_thinking":
			c.Thinking = true
		case "tool_use", "server_tool_use":
			e := m.ev(event.ToolStarted, event.Observed, agent, raw, event.Tool{ToolID: b.ID, Name: b.Name, Input: b.Input})
			e.Item = b.ID
			out = append(out, e)
		}
	}
	if hasText || c.Thinking || len(out) == 0 {
		c.Text = strings.Join(texts, "")
		e := m.ev(event.MessageCompleted, event.AI, agent, raw, c)
		e.Item = msg.ID
		out = append([]event.Event{e}, out...)
	}
	out = append(pre, out...)
	if len(l.Error) > 0 && string(l.Error) != "null" {
		out = append(out, m.ev(event.Error, event.Observed, agent, raw, event.Problem{Message: "assistant error: " + strings.Trim(string(l.Error), `"`)}))
	}
	return append(report, out...)
}

// usageReport maps the usage_report of an assistant message, documented as
// experimental (doc agent-sdk/typescript#sdkusagereport; P24u), to the quota
// windows: its rate_limits.limits[] rows by kind, session to quota.5h,
// weekly_all to quota.7d, weekly_scoped to quota.7d.<model> from its
// scope.model.display_name ("Fable"), each with its percent and reset time;
// nil without a report or a row.
func (m *Mapper) usageReport(rep json.RawMessage, raw []byte) []event.Event {
	if len(rep) == 0 || string(rep) == "null" {
		return nil
	}
	var r struct {
		RateLimits struct {
			Limits []struct {
				Kind     string  `json:"kind"`
				Percent  float64 `json:"percent"`
				ResetsAt string  `json:"resets_at"`
				Scope    *struct {
					Model *struct {
						DisplayName string `json:"display_name"`
					} `json:"model"`
				} `json:"scope"`
			} `json:"limits"`
		} `json:"rate_limits"`
	}
	if json.Unmarshal(rep, &r) != nil {
		return nil
	}
	var ms []event.Metric
	for _, w := range r.RateLimits.Limits {
		name := ""
		switch w.Kind {
		case "session":
			name = "quota.5h"
		case "weekly_all":
			name = "quota.7d"
		case "weekly_scoped":
			model := ""
			if w.Scope != nil && w.Scope.Model != nil {
				model = scopeSlug(w.Scope.Model.DisplayName)
			}
			if model == "" {
				model = "scoped"
			}
			name = "quota.7d." + model
		default:
			continue // a kind the SDK does not document today: not shown
		}
		var reset time.Time
		if t, err := time.Parse(time.RFC3339Nano, w.ResetsAt); err == nil {
			reset = t.UTC()
		}
		ms = append(ms, event.Metric{Name: name, Value: w.Percent, Unit: "%", ResetsAt: reset, Documented: true, Source: "usage_report (" + w.Kind + ")"})
	}
	if len(ms) == 0 {
		return nil
	}
	return []event.Event{m.ev(event.Telemetry, event.Observed, "", raw, event.Metrics{Items: ms})}
}

// scopeSlug is a model's display name as a metric name's part: its letters
// and digits, lower case ("Fable" gives "fable").
func scopeSlug(name string) string {
	var b strings.Builder
	for _, c := range strings.ToLower(name) {
		if c >= 'a' && c <= 'z' || c >= '0' && c <= '9' {
			b.WriteRune(c)
		}
	}
	return b.String()
}

// InInternal maps a line the dashboard wrote for itself, "/usage" (plan 24,
// amendment 23): logged as a write, never the owner's message, and the
// CLI's answer to it is kept out of the conversation (usageReads).
func (m *Mapper) InInternal(raw []byte) []event.Event {
	m.usageReads++
	return []event.Event{m.ev(event.Sent, event.Observed, "", raw, event.Write{What: "usage"})}
}

func (m *Mapper) user(l line, agent string, raw []byte) []event.Event {
	var msg struct {
		Content json.RawMessage `json:"content"`
	}
	_ = json.Unmarshal(l.Message, &msg)
	var blocks []block
	if json.Unmarshal(msg.Content, &blocks) != nil {
		// A plain string echoed back: the CLI's copy of a user message.
		return []event.Event{m.ev(event.StatusChanged, event.Observed, agent, raw, event.Status{Status: "notice", Detail: contentText(msg.Content)})}
	}
	var out []event.Event
	var results []string
	for _, b := range blocks {
		switch b.Type {
		case "tool_result":
			results = append(results, b.ToolUseID)
			st := "completed"
			if b.IsError {
				st = "failed"
			}
			e := m.ev(event.ToolCompleted, event.Observed, agent, raw, event.Tool{ToolID: b.ToolUseID, IsError: b.IsError, Status: st, Output: contentText(b.Content)})
			e.Item = b.ToolUseID
			out = append(out, e)
		case "text":
			out = append(out, m.ev(event.StatusChanged, event.Observed, agent, raw, event.Status{Status: "notice", Detail: b.Text}))
		}
	}
	out = append(out, m.verdicts(l, agent, raw, results)...)
	if len(out) == 0 {
		out = append(out, m.unknown(raw, "user", "empty"))
	}
	return out
}

// approvalMode is a Claude Code permissionMode as the dashboard names it:
// default is Manual, auto is Auto, any other as given (plan 12, item 2).
func approvalMode(mode string) string {
	switch mode {
	case "default":
		return event.ModeManual
	case "auto":
		return event.ModeAuto
	}
	return mode
}

// verdicts reads the line's tool_result_meta (undocumented): a decision the
// classifier made is a tool.decision; any other reason adds nothing. Every
// recording from 2.1.300 to 2.1.303 gives each tool result its
// permission_decision, except a tool use an interrupt rejected, whose entry
// has non_execution_kind "user-rejected" and no decision (P303e; the live
// gate on 2.1.300 and 2.1.303). Any other result without a decision counts
// as unrecognised: a CLI update that drops or renames the field shows in
// zone D instead of silently hiding the classifier's verdicts
// (doc/evaluation.md, After a CLI update).
func (m *Mapper) verdicts(l line, agent string, raw []byte, results []string) []event.Event {
	var metas []struct {
		ID           string `json:"id"`
		NonExecution string `json:"non_execution_kind"`
		Decision     struct {
			Decision   string `json:"decision"`
			ReasonType string `json:"reason_type"`
		} `json:"permission_decision"`
	}
	_ = json.Unmarshal(l.ToolResultMeta, &metas) // absent or another shape: no entry
	var out []event.Event
	for _, id := range results {
		found := false
		for _, x := range metas {
			found = found || (x.ID == id && (x.Decision.Decision != "" || x.NonExecution == "user-rejected"))
		}
		if !found {
			out = append(out, m.ev(event.Unrecognised, event.Observed, "", raw, event.Unknown{Type: "user/tool_result_meta", Reason: "tool result without tool_result_meta permission_decision (undocumented)"}))
		}
	}
	for _, x := range metas {
		if x.Decision.ReasonType != "classifier" {
			continue
		}
		e := m.ev(event.ToolDecision, event.Observed, agent, raw, event.Verdict{ToolID: x.ID, Decision: x.Decision.Decision, ReasonType: x.Decision.ReasonType})
		e.Item = x.ID
		out = append(out, e)
	}
	return out
}

func (m *Mapper) rateLimit(raw []byte) []event.Event {
	var r struct {
		Info struct {
			Status        string   `json:"status"`
			ResetsAt      int64    `json:"resetsAt"`
			RateLimitType string   `json:"rateLimitType"`
			Utilization   *float64 `json:"utilization"`
			Windows       map[string]struct {
				Utilization float64 `json:"utilization"`
				ResetsAt    int64   `json:"resetsAt"`
			} `json:"unifiedWindows"`
			OverageStatus  string `json:"overageStatus"`
			OverageReason  string `json:"overageDisabledReason"`
			IsUsingOverage bool   `json:"isUsingOverage"`
		} `json:"rate_limit_info"`
	}
	_ = json.Unmarshal(raw, &r)
	var ms []event.Metric
	// The windows in a fixed order: five_hour, seven_day, then the
	// model-scoped weekly windows by name (plan 24, design D8).
	keys := make([]string, 0, len(r.Info.Windows))
	for key := range r.Info.Windows {
		if _, ok := windowMetric(key); ok {
			keys = append(keys, key)
		}
	}
	sort.Slice(keys, func(i, j int) bool {
		rank := func(k string) int {
			switch k {
			case "five_hour":
				return 0
			case "seven_day":
				return 1
			}
			return 2
		}
		if a, b := rank(keys[i]), rank(keys[j]); a != b {
			return a < b
		}
		return keys[i] < keys[j]
	})
	for _, key := range keys {
		w := r.Info.Windows[key]
		name, _ := windowMetric(key)
		ms = append(ms, event.Metric{Name: name, Value: w.Utilization * 100, Unit: "%", ResetsAt: unix(w.ResetsAt), Documented: false, Source: "rate_limit_event.unifiedWindows"})
	}
	// The event's own window, documented: the one whose status changed. With
	// a utilization it is recorded whenever unifiedWindows did not give it,
	// so a model-scoped window (seven_day_<model>, which unifiedWindows has
	// never carried: every probe holds five_hour and seven_day only; plan
	// 24, the owner's live session of 2026-10-22) is kept when it comes.
	if n, ok := windowMetric(r.Info.RateLimitType); ok {
		seen := false
		for _, m := range ms {
			seen = seen || m.Name == n
		}
		switch {
		case seen:
		case r.Info.Utilization != nil:
			ms = append(ms, event.Metric{Name: n, Value: *r.Info.Utilization * 100, Unit: "%", ResetsAt: unix(r.Info.ResetsAt), Documented: true, Source: "rate_limit_event"})
		case r.Info.Status != "" && len(ms) == 0:
			// The status word alone, with no percentage (reliability.md,
			// Telemetry sources: 5-hour, 7-day quota).
			ms = append(ms, event.Metric{Name: n, Text: r.Info.Status, ResetsAt: unix(r.Info.ResetsAt), Documented: true, Source: "rate_limit_event.status"})
		}
	}
	ms = append(ms, event.Metric{Name: "quota.status", Text: r.Info.Status, Documented: true, Source: "rate_limit_event"})
	// Overage, kept apart from the plan's quota (claude-code.md, section 9).
	switch {
	case r.Info.IsUsingOverage:
		ms = append(ms, event.Metric{Name: "overage", Text: "in use", Documented: true, Source: "rate_limit_event.isUsingOverage"})
	case r.Info.OverageStatus != "":
		t := r.Info.OverageStatus
		if r.Info.OverageReason != "" {
			t += " (" + r.Info.OverageReason + ")"
		}
		ms = append(ms, event.Metric{Name: "overage", Text: t, Documented: true, Source: "rate_limit_event.overageStatus"})
	}
	return []event.Event{m.ev(event.Telemetry, event.Observed, "", raw, event.Metrics{Items: ms})}
}

func unix(s int64) time.Time {
	if s == 0 {
		return time.Time{}
	}
	return time.Unix(s, 0).UTC()
}

// windowMetric names the metric of a rate-limit window: five_hour is
// quota.5h, seven_day quota.7d, and a model-scoped weekly window,
// seven_day_<model> (seven_day_opus and seven_day_sonnet are documented as
// rateLimitType values; claude-code.md, section 7), quota.7d.<model>, shown
// as "<Model> limit" (plan 24, design D8). Any other key is no window.
func windowMetric(key string) (string, bool) {
	switch {
	case key == "five_hour":
		return "quota.5h", true
	case key == "seven_day":
		return "quota.7d", true
	case strings.HasPrefix(key, "seven_day_") && len(key) > len("seven_day_"):
		m := strings.TrimPrefix(key, "seven_day_")
		for _, c := range m {
			if (c < 'a' || c > 'z') && (c < '0' || c > '9') {
				return "", false
			}
		}
		return "quota.7d." + m, true
	}
	return "", false
}

func (m *Mapper) result(raw []byte) []event.Event {
	var r struct {
		Subtype    string   `json:"subtype"`
		IsError    bool     `json:"is_error"`
		DurationMS int64    `json:"duration_ms"`
		CostUSD    *float64 `json:"total_cost_usd"`
		Result     string   `json:"result"`
		Errors     []string `json:"errors"`
		Usage      struct {
			Input         int64 `json:"input_tokens"`
			Output        int64 `json:"output_tokens"`
			CacheRead     int64 `json:"cache_read_input_tokens"`
			CacheCreation int64 `json:"cache_creation_input_tokens"`
		} `json:"usage"`
		ModelUsage map[string]struct {
			Input         float64 `json:"inputTokens"`
			Output        float64 `json:"outputTokens"`
			CacheRead     float64 `json:"cacheReadInputTokens"`
			CacheCreation float64 `json:"cacheCreationInputTokens"`
			ContextWindow float64 `json:"contextWindow"`
		} `json:"modelUsage"`
	}
	_ = json.Unmarshal(raw, &r)
	if m.usageReads > 0 {
		// The result of the dashboard's own usage read: no turn ended, and
		// its figures (no model call: 0 ms, 0 tokens, no cost) are not the
		// session's (P24u).
		open := m.usageOpen
		m.usageReads, m.usageStarted, m.usageOpen = m.usageReads-1, false, false
		if open {
			return []event.Event{m.ev(event.StatusChanged, event.Observed, "", raw, event.Status{Detail: "usage report: read"})}
		}
		why := r.Subtype
		if len(r.Errors) > 0 && r.Errors[0] != "" {
			why = strings.Join(r.Errors, "; ")
		} else if r.Result != "" {
			why = r.Result
		}
		return []event.Event{m.ev(event.Error, event.Observed, "", raw, event.Problem{Message: "usage report not read: " + why})}
	}
	end := event.TurnEnd{Subtype: r.Subtype, DurationMS: r.DurationMS}
	switch {
	case r.Subtype == "success" && !r.IsError:
		end.Outcome = "completed"
	case m.interrupted && r.Subtype == "error_during_execution":
		end.Outcome = "interrupted"
	default:
		// The error as received: its errors, else its result text, else
		// the subtype (claude-code.md, section 9: Errors).
		end.Outcome, end.Error = "failed", r.Subtype
		if len(r.Errors) > 0 && r.Errors[0] != "" {
			end.Error = strings.Join(r.Errors, "\n")
		} else if r.IsError && r.Result != "" {
			end.Error = r.Result
		}
	}
	m.interrupted = false
	ms := []event.Metric{{Name: "duration.ms", Value: float64(r.DurationMS), Unit: "ms", Documented: true, Source: "result.duration_ms"}}
	if r.CostUSD != nil {
		ms = append(ms, event.Metric{Name: "cost.usd", Value: *r.CostUSD, Unit: "USD", Documented: true, Source: "result.total_cost_usd (API-equivalent estimate)"})
	}
	tokens := r.Usage.Input + r.Usage.Output + r.Usage.CacheRead + r.Usage.CacheCreation
	ms = append(ms, event.Metric{Name: "tokens.turn", Value: float64(tokens), Unit: "tokens", Documented: true, Source: "result.usage"})
	if len(r.ModelUsage) > 0 {
		// The running total, subagents included: read, not summed over turns
		// (claude-code.md, section 9: Tokens).
		var total float64
		for _, u := range r.ModelUsage {
			total += u.Input + u.Output + u.CacheRead + u.CacheCreation
		}
		ms = append(ms, event.Metric{Name: "tokens.session", Value: total, Unit: "tokens", Documented: true, Source: "result.modelUsage (running total, subagents included)"})
	}
	// The derived context: the main thread's last request over its model's
	// window. A get_context_usage reply after this result replaces it.
	u, ok := r.ModelUsage[m.lastModel]
	if !ok && len(r.ModelUsage) == 1 {
		for _, only := range r.ModelUsage {
			u, ok = only, true
		}
	}
	if ok && m.lastContext > 0 && u.ContextWindow > 0 {
		const src = "last assistant usage over result.modelUsage.contextWindow"
		ms = append(ms,
			event.Metric{Name: "context.used", Value: m.lastContext, Unit: "tokens", Documented: true, Source: src, Derived: true},
			event.Metric{Name: "context.window", Value: u.ContextWindow, Unit: "tokens", Documented: true, Source: src, Derived: true})
	}
	return []event.Event{
		m.ev(event.TurnCompleted, event.Observed, "", raw, end),
		m.ev(event.Telemetry, event.Observed, "", raw, event.Metrics{Items: ms}),
	}
}

func (m *Mapper) controlRequest(l line, raw []byte) []event.Event {
	var req struct {
		Subtype     string          `json:"subtype"`
		ToolName    string          `json:"tool_name"`
		DisplayName string          `json:"display_name"`
		Input       json.RawMessage `json:"input"`
		Description string          `json:"description"`
		ToolUseID   string          `json:"tool_use_id"`
		AgentID     string          `json:"agent_id"`
	}
	err := json.Unmarshal(l.Request, &req)
	if req.Subtype != "can_use_tool" && err == nil {
		u := m.unknown(raw, "control_request", req.Subtype)
		u.Payload = event.Unknown{Type: "control_request/" + req.Subtype, Reason: "control request the dashboard does not handle"}
		return []event.Event{u}
	}
	a := event.Approval{ID: l.RequestID, Tool: req.ToolName, Display: req.DisplayName, Summary: req.Description, ToolID: req.ToolUseID, Input: req.Input}
	switch {
	case err != nil:
		a.Problem = "unreadable request: " + err.Error()
	case l.RequestID == "":
		a.Problem = "request without request_id"
	case req.ToolName == "":
		a.Problem = "request without tool_name"
	}
	k := event.ApprovalRequest
	if req.ToolName == "AskUserQuestion" {
		k = event.Question
	}
	e := m.ev(k, event.Observed, req.AgentID, raw, a)
	e.Item = req.ToolUseID
	return []event.Event{e}
}

func (m *Mapper) controlResponse(l line, raw []byte) []event.Event {
	var resp struct {
		Subtype   string          `json:"subtype"`
		RequestID string          `json:"request_id"`
		Error     string          `json:"error"`
		Response  json.RawMessage `json:"response"`
	}
	_ = json.Unmarshal(l.Response, &resp)
	what := m.pending[resp.RequestID]
	delete(m.pending, resp.RequestID)
	if resp.Subtype == "error" {
		switch what {
		case "set_permission_mode":
			// Zone B shows the refusal; the mode in force is unchanged (plan 12, item 2).
			return []event.Event{m.ev(event.ApprovalMode, event.Observed, "", raw, event.Mode{Refused: true, Text: what + " refused by Claude Code: " + resp.Error})}
		case "set_model", "apply_flag_settings":
			return m.changeAnswered(what, true, resp.Error, raw)
		case "get_settings":
			// The settings in force are not known; a change waiting for
			// this read is settled with it (plan 34).
			s := event.Settings{Source: what, Undocumented: true, Text: what + " refused by Claude Code: " + resp.Error}
			if m.answering[resp.RequestID] {
				delete(m.answering, resp.RequestID)
				s.Answers = m.changes == 0
			}
			return []event.Event{m.ev(event.ModelSettings, event.Observed, "", raw, s)}
		}
		return []event.Event{m.ev(event.Error, event.Observed, "", raw, event.Problem{Message: what + " refused by Claude Code: " + resp.Error})}
	}
	switch what {
	case "initialize":
		evs := []event.Event{m.ev(event.SessionReady, event.Observed, "", raw, event.Ready{Detail: "initialize response"})}
		if c := catalogue(resp.Response); len(c) > 0 {
			// The session's models (plan 34): the Agent SDK's ModelInfo.
			m.catalogue = c
			evs = append(evs, m.ev(event.ModelsListed, event.Observed, "", raw, event.Models{Entries: c, Source: "initialize"}))
		}
		return evs
	case "set_model", "apply_flag_settings":
		return m.changeAnswered(what, false, "", raw)
	case "get_settings":
		s, line := m.settingsRead(resp.RequestID, resp.Response)
		if m.answering[resp.RequestID] {
			delete(m.answering, resp.RequestID)
			s.Answers = m.changes == 0
		}
		return []event.Event{m.ev(event.ModelSettings, event.Observed, "", line, s)}
	case "get_context_usage":
		var c struct {
			Total     float64 `json:"totalTokens"`
			Max       float64 `json:"maxTokens"`
			Threshold float64 `json:"autoCompactThreshold"`
		}
		_ = json.Unmarshal(resp.Response, &c)
		ms := []event.Metric{
			{Name: "context.used", Value: c.Total, Unit: "tokens", Documented: false, Source: "get_context_usage"},
			{Name: "context.window", Value: c.Max, Unit: "tokens", Documented: false, Source: "get_context_usage"},
		}
		if c.Threshold > 0 {
			ms = append(ms, event.Metric{Name: "context.compact_at", Value: c.Threshold, Unit: "tokens", Documented: false, Source: "get_context_usage"})
		}
		return []event.Event{m.ev(event.Telemetry, event.Observed, "", raw, event.Metrics{Items: ms})}
	case "interrupt":
		return []event.Event{m.ev(event.StatusChanged, event.Observed, "", raw, event.Status{Status: "interrupt acknowledged"})}
	case "set_permission_mode":
		var r struct {
			Mode string `json:"mode"`
		}
		_ = json.Unmarshal(resp.Response, &r)
		if r.Mode == "" {
			return []event.Event{m.ev(event.ApprovalMode, event.Observed, "", raw, event.Mode{Refused: true, Text: what + " answered without a mode"})}
		}
		return []event.Event{m.ev(event.ApprovalMode, event.Observed, "", raw, event.Mode{Mode: approvalMode(r.Mode), Answers: true})}
	}
	return []event.Event{m.unknown(raw, "control_response", "unmatched")}
}

// contentText returns a message content as text: a string as is, or the text
// of its text blocks joined.
func contentText(c json.RawMessage) string {
	var s string
	if json.Unmarshal(c, &s) == nil {
		return s
	}
	var blocks []block
	if json.Unmarshal(c, &blocks) != nil {
		return ""
	}
	var b strings.Builder
	for _, x := range blocks {
		if x.Type == "text" {
			b.WriteString(x.Text)
		}
	}
	return b.String()
}
