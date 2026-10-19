// Package claude is the Claude Code adapter: `claude -p` with stream-json in
// and out and the stdio control protocol (doc/architecture.md, Claude Code
// adapter). The control-request wire form and `--permission-prompt-tool
// stdio` are undocumented (INT 1, INT 4).
package claude

import (
	"encoding/json"
	"fmt"
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
}

// NewMapper returns an empty mapper.
func NewMapper() *Mapper {
	return &Mapper{pending: map[string]string{}, models: map[string]string{}}
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
		return []event.Event{m.ev(event.UserMessage, event.Observed, "", raw, event.User{Text: contentText(msg.Content)})}
	case "control_request":
		var req struct {
			Subtype string `json:"subtype"`
		}
		_ = json.Unmarshal(l.Request, &req)
		m.pending[l.RequestID] = req.Subtype
		if req.Subtype == "interrupt" {
			m.interrupted = true
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
		var s struct {
			Estimated float64 `json:"estimated_tokens"`
		}
		_ = json.Unmarshal(raw, &s)
		return []event.Event{m.ev(event.Telemetry, event.Observed, "", raw, event.Metrics{Items: []event.Metric{{Name: "thinking.estimate", Value: s.Estimated, Unit: "tokens", Source: "system/thinking_tokens"}}})}
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
	id := l.APIMessageID
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
		if se.Message.ID != "" {
			id = se.Message.ID
		}
		return []event.Event{m.ev(event.MessageMarker, event.Observed, agent, raw, event.Marker{MessageID: id, Name: se.Type, Model: se.Message.Model})}
	case "content_block_start", "content_block_stop", "message_stop":
		return []event.Event{m.ev(event.MessageMarker, event.Observed, agent, raw, event.Marker{MessageID: id, Name: se.Type, Block: se.ContentBlock.Type})}
	case "message_delta":
		return []event.Event{m.ev(event.MessageMarker, event.Observed, agent, raw, event.Marker{MessageID: id, Name: se.Type, StopReason: se.Delta.StopReason})}
	}
	return []event.Event{m.unknown(raw, l.Type, se.Type)}
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
	return out
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
	names := map[string]string{"five_hour": "quota.5h", "seven_day": "quota.7d"}
	for _, key := range []string{"five_hour", "seven_day"} {
		if w, ok := r.Info.Windows[key]; ok {
			ms = append(ms, event.Metric{Name: names[key], Value: w.Utilization * 100, Unit: "%", ResetsAt: unix(w.ResetsAt), Documented: false, Source: "rate_limit_event.unifiedWindows"})
		}
	}
	if n, ok := names[r.Info.RateLimitType]; ok && len(ms) == 0 {
		if r.Info.Utilization != nil {
			ms = append(ms, event.Metric{Name: n, Value: *r.Info.Utilization * 100, Unit: "%", ResetsAt: unix(r.Info.ResetsAt), Documented: true, Source: "rate_limit_event"})
		} else if r.Info.Status != "" {
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
		if what == "set_permission_mode" {
			// Zone B shows the refusal; the mode in force is unchanged (plan 12, item 2).
			return []event.Event{m.ev(event.ApprovalMode, event.Observed, "", raw, event.Mode{Refused: true, Text: what + " refused by Claude Code: " + resp.Error})}
		}
		return []event.Event{m.ev(event.Error, event.Observed, "", raw, event.Problem{Message: what + " refused by Claude Code: " + resp.Error})}
	}
	switch what {
	case "initialize":
		return []event.Event{m.ev(event.SessionReady, event.Observed, "", raw, event.Ready{Detail: "initialize response"})}
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
