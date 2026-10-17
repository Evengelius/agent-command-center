// Package codex is the Codex CLI adapter: `codex app-server` over stdio, JSON-RPC
// 2.0 lines in both directions (doc/architecture.md, Codex adapter). The
// app-server is documented but marked experimental by Codex; the wire shapes
// here are those of the v2 schema and of the plan 1 recordings (P6, P6b, P7).
// It does not read `codex exec --json` output (a different protocol).
package codex

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/Evengelius/agent-command-center/internal/event"
)

// ProviderID is the adapter id.
const ProviderID = "codex"

// Mapper turns app-server lines into normalised events. It keeps the little
// state the wire form needs (which method a response answers, which thread is
// the main one, which server requests are open, whether an interrupt is
// outstanding), so the same lines always give the same events.
type Mapper struct {
	session     string
	main        string               // the orchestrator's thread id
	pending     map[string]string    // id of our own requests -> method
	server      map[string]serverReq // id of the CLI's requests -> what they are about
	files       map[string]string    // fileChange item id -> paths, to describe its approval
	interrupted bool
}

type serverReq struct {
	agent, turn, item, tool string
}

// NewMapper returns an empty mapper.
func NewMapper() *Mapper {
	return &Mapper{pending: map[string]string{}, server: map[string]serverReq{}, files: map[string]string{}}
}

type rpc struct {
	ID     json.RawMessage `json:"id"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`
	Result json.RawMessage `json:"result"`
	Error  json.RawMessage `json:"error"`
	Type   string          `json:"type"` // only exec-style lines, which this mapper does not read
}

func (r rpc) hasID() bool { return len(r.ID) > 0 && string(r.ID) != "null" }

func idString(raw json.RawMessage) string {
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s
	}
	return strings.TrimSpace(string(raw))
}

// ids are the thread, turn and item a notification names.
type ids struct {
	ThreadID string `json:"threadId"`
	TurnID   string `json:"turnId"`
	ItemID   string `json:"itemId"`
}

// Out maps one line read from the CLI's stdout.
func (m *Mapper) Out(raw []byte) []event.Event {
	raw = redactCredits(raw)
	var r rpc
	if err := json.Unmarshal(raw, &r); err != nil {
		return []event.Event{m.ev(event.Error, event.Observed, "", "", "", raw, event.Problem{Message: "unreadable line from Codex: " + err.Error()})}
	}
	switch {
	case r.Method != "" && r.hasID():
		return m.serverRequest(r, raw)
	case r.Method != "":
		return m.notification(r, raw)
	case r.hasID():
		return m.response(r, raw)
	}
	t := r.Type
	if t == "" {
		t = "line"
	}
	return []event.Event{m.unknown(raw, t, "")}
}

// In maps one line the dashboard wrote to the CLI's stdin: a request, a
// notification, or the answer to one of the CLI's own requests.
func (m *Mapper) In(raw []byte) []event.Event {
	var r rpc
	if err := json.Unmarshal(raw, &r); err != nil {
		return []event.Event{m.ev(event.Error, event.Observed, "", "", "", raw, event.Problem{Message: "unreadable line written to Codex: " + err.Error()})}
	}
	switch {
	case r.Method != "":
		id := ""
		if r.hasID() {
			id = idString(r.ID)
			m.pending[id] = r.Method
		}
		var p struct {
			ids
			Input []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"input"`
		}
		_ = json.Unmarshal(r.Params, &p)
		switch r.Method {
		case "thread/start": // a new main thread: its thread/started names it
			m.main = ""
		case "thread/resume":
			m.main, m.session = p.ThreadID, p.ThreadID
		}
		agent := m.agentOf(p.ThreadID)
		var out []event.Event
		switch r.Method {
		case "turn/start":
			var texts []string
			for _, in := range p.Input {
				if in.Type == "text" {
					texts = append(texts, in.Text)
				}
			}
			out = append(out, m.ev(event.UserMessage, event.Observed, agent, "", "", raw, event.User{Text: strings.Join(texts, "\n")}))
		case "turn/interrupt":
			m.interrupted = true
		}
		return append(out, m.ev(event.Sent, event.Observed, agent, p.TurnID, "", raw, event.Write{What: r.Method, RequestID: id}))
	case r.hasID():
		id := idString(r.ID)
		req, ok := m.server[id]
		if !ok && len(r.Error) > 0 && string(r.Error) != "null" {
			// The error the dashboard returns for a request it does not handle.
			return []event.Event{m.ev(event.Sent, event.Observed, "", "", "", raw, event.Write{What: "error reply", RequestID: id})}
		}
		delete(m.server, id)
		a := event.Approval{ID: id, Tool: req.tool, ToolID: req.item}
		if len(r.Error) > 0 && string(r.Error) != "null" {
			a.Decision = "error"
		} else {
			a.Decision = decision(r.Result)
		}
		return []event.Event{m.ev(event.ApprovalResolved, event.Observed, req.agent, req.turn, req.item, raw, a)}
	}
	return []event.Event{m.ev(event.Sent, event.Observed, "", "", "", raw, event.Write{What: "unknown"})}
}

// decision is the decision of an approval answer, as given: a string, or the
// name of the object form ({"acceptWithExecpolicyAmendment": {...}}).
func decision(result json.RawMessage) string {
	var d struct {
		Decision json.RawMessage `json:"decision"`
	}
	_ = json.Unmarshal(result, &d)
	var s string
	if json.Unmarshal(d.Decision, &s) == nil {
		return s
	}
	var o map[string]json.RawMessage
	if json.Unmarshal(d.Decision, &o) == nil {
		for k := range o {
			return k
		}
	}
	return ""
}

func (m *Mapper) ev(k event.Kind, o event.Origin, agent, turn, item string, raw []byte, p any) event.Event {
	return event.Event{Provider: ProviderID, Session: m.session, Kind: k, Origin: o, Agent: agent, Turn: turn, Item: item, Payload: p, Raw: string(raw)}
}

func (m *Mapper) unknown(raw []byte, typ, reason string) event.Event {
	return m.ev(event.Unrecognised, event.Observed, "", "", "", raw, event.Unknown{Type: typ, Reason: reason})
}

// agentOf is "" for the orchestrator's thread and the thread id for any other.
func (m *Mapper) agentOf(thread string) string {
	if thread == "" || m.main == "" || thread == m.main {
		return ""
	}
	return thread
}

func (m *Mapper) status(agent string, raw []byte, s, detail string) event.Event {
	return m.ev(event.StatusChanged, event.Observed, agent, "", "", raw, event.Status{Status: s, Detail: detail})
}

func problemMessage(raw json.RawMessage) string {
	var e struct {
		Message string `json:"message"`
	}
	if json.Unmarshal(raw, &e) != nil || e.Message == "" {
		return strings.Trim(string(raw), `"`)
	}
	return unwrap(e.Message)
}

// unwrap digs the message out of an API error the CLI passes on as JSON text.
func unwrap(s string) string {
	var e struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if strings.HasPrefix(s, "{") && json.Unmarshal([]byte(s), &e) == nil && e.Error.Message != "" {
		return e.Error.Message
	}
	return s
}

// response maps the answer to one of the dashboard's own requests.
func (m *Mapper) response(r rpc, raw []byte) []event.Event {
	id := idString(r.ID)
	method, known := m.pending[id]
	delete(m.pending, id)
	if len(r.Error) > 0 && string(r.Error) != "null" {
		what := method
		if what == "" {
			what = "request " + id
		}
		msg := problemMessage(r.Error)
		refused := m.ev(event.Error, event.Observed, "", "", "", raw, event.Problem{Message: what + " refused by Codex: " + msg})
		if method == "account/rateLimits/read" {
			return m.rateLimitsRefused(msg, refused, raw)
		}
		return []event.Event{refused}
	}
	if !known {
		return []event.Event{m.unknown(raw, "response/unmatched", "response to a request the dashboard did not send")}
	}
	switch method {
	case "initialize":
		var s struct {
			UserAgent string `json:"userAgent"`
		}
		_ = json.Unmarshal(r.Result, &s)
		return []event.Event{m.status("", raw, "initialized", s.UserAgent)}
	case "thread/start", "thread/resume":
		return m.threadStart(r, raw)
	case "account/rateLimits/read":
		var s struct {
			RateLimits json.RawMessage `json:"rateLimits"`
		}
		_ = json.Unmarshal(r.Result, &s)
		return []event.Event{m.rateLimits(s.RateLimits, "account/rateLimits/read", raw)}
	case "turn/start":
		var s struct {
			Turn struct {
				ID string `json:"id"`
			} `json:"turn"`
		}
		_ = json.Unmarshal(r.Result, &s)
		e := m.status("", raw, "turn accepted", "")
		e.Turn = s.Turn.ID
		return []event.Event{e}
	case "turn/interrupt":
		return []event.Event{m.status("", raw, "interrupt acknowledged", "")}
	}
	return []event.Event{m.status("", raw, "response", method)}
}

func (m *Mapper) threadStart(r rpc, raw []byte) []event.Event {
	var s struct {
		Thread struct {
			ID         string `json:"id"`
			SessionID  string `json:"sessionId"`
			Cwd        string `json:"cwd"`
			CliVersion string `json:"cliVersion"`
		} `json:"thread"`
		Model          string          `json:"model"`
		Cwd            string          `json:"cwd"`
		CliVersion     string          `json:"cliVersion"`
		Sandbox        json.RawMessage `json:"sandbox"`
		ApprovalPolicy json.RawMessage `json:"approvalPolicy"`
	}
	if err := json.Unmarshal(r.Result, &s); err != nil || s.Thread.ID == "" {
		return []event.Event{m.ev(event.Error, event.Observed, "", "", "", raw, event.Problem{Message: "thread/start response without a thread id"})}
	}
	m.main, m.session = s.Thread.ID, s.Thread.ID
	st := event.Started{SessionID: s.Thread.ID, Model: s.Model, Cwd: first(s.Cwd, s.Thread.Cwd), Version: first(s.Thread.CliVersion, s.CliVersion),
		Sandbox: namedValue(s.Sandbox), ApprovalPolicy: namedValue(s.ApprovalPolicy)}
	return []event.Event{
		m.ev(event.SessionReady, event.Observed, "", "", "", raw, event.Ready{Detail: "thread/start response"}),
		m.ev(event.SessionStarted, event.Observed, "", "", "", raw, st),
	}
}

func first(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

// namedValue reads a value given as a string or as an object with a "type".
func namedValue(raw json.RawMessage) string {
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s
	}
	var o struct {
		Type string `json:"type"`
	}
	if json.Unmarshal(raw, &o) == nil {
		return o.Type
	}
	return ""
}

// serverRequest maps a request the CLI sends to the dashboard.
func (m *Mapper) serverRequest(r rpc, raw []byte) []event.Event {
	tool := ""
	switch r.Method {
	case "item/commandExecution/requestApproval":
		tool = "commandExecution"
	case "item/fileChange/requestApproval":
		tool = "fileChange"
	default:
		return []event.Event{m.ev(event.Unrecognised, event.Observed, "", "", "", raw, event.Unknown{Type: "serverRequest/" + r.Method, Reason: "server request the dashboard does not handle"})}
	}
	id := idString(r.ID)
	var p struct {
		ids
		Reason  *string           `json:"reason"`
		Command string            `json:"command"`
		Offered []json.RawMessage `json:"availableDecisions"` // app-server README; not in 0.163.0's generated schema
	}
	err := json.Unmarshal(r.Params, &p)
	a := event.Approval{ID: id, Tool: tool, ToolID: p.ItemID, Input: r.Params}
	switch {
	case err != nil:
		a.Problem = "unreadable request: " + err.Error()
	case p.ItemID == "":
		a.Problem = "request without itemId"
	case p.ThreadID == "":
		a.Problem = "request without threadId"
	}
	reason := ""
	if p.Reason != nil {
		reason = *p.Reason
	}
	switch {
	case reason != "":
		a.Summary = reason
	case tool == "commandExecution" && p.Command != "":
		a.Summary = p.Command
	case tool == "fileChange" && m.files[p.ItemID] != "":
		a.Summary = m.files[p.ItemID]
	default:
		a.Summary = tool
	}
	if tool == "commandExecution" {
		a.Display = p.Command
	}
	for _, d := range p.Offered {
		var s string
		if json.Unmarshal(d, &s) == nil { // the amendments are objects: never kept, never offered
			a.Offered = append(a.Offered, s)
		}
	}
	agent := m.agentOf(p.ThreadID)
	m.server[id] = serverReq{agent: agent, turn: p.TurnID, item: p.ItemID, tool: tool}
	return []event.Event{m.ev(event.ApprovalRequest, event.Observed, agent, p.TurnID, p.ItemID, raw, a)}
}

// notification maps a notification from the CLI.
func (m *Mapper) notification(r rpc, raw []byte) []event.Event {
	var p struct {
		ids
		Item      json.RawMessage `json:"item"`
		Turn      json.RawMessage `json:"turn"`
		Thread    json.RawMessage `json:"thread"`
		Status    json.RawMessage `json:"status"`
		Delta     string          `json:"delta"`
		Diff      string          `json:"diff"`
		Message   string          `json:"message"`
		Error     json.RawMessage `json:"error"`
		WillRetry bool            `json:"willRetry"`
		RequestID json.RawMessage `json:"requestId"`
		Name      string          `json:"name"`
		Failure   string          `json:"failureReason"`
		TokenUse  json.RawMessage `json:"tokenUsage"`
		Limits    json.RawMessage `json:"rateLimits"`
	}
	if len(r.Params) > 0 {
		if err := json.Unmarshal(r.Params, &p); err != nil {
			return []event.Event{m.unknown(raw, r.Method, "params not readable: "+err.Error())}
		}
	}
	agent := m.agentOf(p.ThreadID)
	switch r.Method {
	case "thread/started":
		return m.threadStarted(p.Thread, raw)
	case "thread/status/changed":
		var s struct {
			Type        string   `json:"type"`
			ActiveFlags []string `json:"activeFlags"`
		}
		_ = json.Unmarshal(p.Status, &s)
		return []event.Event{m.status(agent, raw, s.Type, strings.Join(s.ActiveFlags, ","))}
	case "turn/started":
		var t struct {
			ID string `json:"id"`
		}
		_ = json.Unmarshal(p.Turn, &t)
		return []event.Event{m.ev(event.TurnStarted, event.Observed, agent, first(t.ID, p.TurnID), "", raw, nil)}
	case "turn/completed":
		return m.turnCompleted(p.Turn, agent, p.TurnID, raw)
	case "item/started", "item/completed":
		return m.item(r.Method == "item/started", p.Item, agent, p.TurnID, raw)
	case "item/agentMessage/delta":
		return []event.Event{m.ev(event.MessageDelta, event.AI, agent, p.TurnID, p.ItemID, raw, event.Delta{MessageID: p.ItemID, Block: "text", Text: p.Delta})}
	case "item/reasoning/summaryTextDelta", "item/reasoning/textDelta":
		return []event.Event{m.ev(event.MessageDelta, event.AI, agent, p.TurnID, p.ItemID, raw, event.Delta{MessageID: p.ItemID, Block: "thinking", Text: p.Delta})}
	case "item/commandExecution/outputDelta":
		// First seen in P7b (Codex 0.163.0): a running command's output
		// (events-and-state.md, Tool lifecycle).
		return []event.Event{m.ev(event.ToolProgress, event.Observed, agent, p.TurnID, p.ItemID, raw, event.Tool{ToolID: p.ItemID, Output: p.Delta})}
	case "thread/tokenUsage/updated":
		return []event.Event{m.tokenUsage(p.TokenUse, agent, p.TurnID, raw)}
	case "account/rateLimits/updated":
		return []event.Event{m.rateLimits(p.Limits, "account/rateLimits/updated", raw)}
	case "turn/plan/updated":
		return []event.Event{m.plan(r.Params, agent, p.TurnID, raw)}
	case "turn/diff/updated":
		return []event.Event{m.ev(event.DiffUpdated, event.Observed, agent, p.TurnID, "", raw, event.Diff{Text: p.Diff})}
	case "serverRequest/resolved":
		id := idString(p.RequestID)
		req, ok := m.server[id]
		delete(m.server, id)
		if !ok {
			req = serverReq{agent: agent, turn: p.TurnID}
		}
		return []event.Event{m.ev(event.ApprovalResolved, event.Observed, req.agent, req.turn, req.item, raw, event.Approval{ID: id, Tool: req.tool, ToolID: req.item, Decision: "resolved"})}
	case "error":
		msg := problemMessage(p.Error)
		k := event.Error
		if p.WillRetry {
			k = event.Retry
		}
		return []event.Event{m.ev(k, event.Observed, agent, p.TurnID, "", raw, event.Problem{Message: msg, WillRetry: p.WillRetry})}
	case "warning":
		return []event.Event{m.status(agent, raw, "warning", p.Message)}
	case "mcpServer/startupStatus/updated":
		var s struct {
			Status string `json:"status"`
			Error  string `json:"error"`
		}
		_ = json.Unmarshal(r.Params, &s)
		return []event.Event{m.status(agent, raw, "mcp "+p.Name+" "+s.Status, first(s.Error, p.Failure))}
	case "remoteControl/status/changed":
		var s struct {
			Status string `json:"status"`
		}
		_ = json.Unmarshal(r.Params, &s)
		return []event.Event{m.status("", raw, "remote control "+s.Status, "")}
	case "skills/changed":
		return []event.Event{m.status("", raw, "skills changed", "")}
	case "account/updated":
		// First seen in P7b (Codex 0.163.0): the sign-in mode and plan.
		return []event.Event{m.status("", raw, "account updated", "")}
	}
	return []event.Event{m.unknown(raw, r.Method, "")}
}

func (m *Mapper) threadStarted(raw json.RawMessage, line []byte) []event.Event {
	var t struct {
		ID             string `json:"id"`
		ParentThreadID string `json:"parentThreadId"`
		AgentNickname  string `json:"agentNickname"`
		AgentRole      string `json:"agentRole"`
		Cwd            string `json:"cwd"`
	}
	_ = json.Unmarshal(raw, &t)
	if t.ID == "" {
		return []event.Event{m.unknown(line, "thread/started", "thread without an id")}
	}
	if m.main == "" {
		m.main, m.session = t.ID, t.ID
	}
	if t.ID == m.main {
		return []event.Event{m.status("", line, "thread started", t.Cwd)}
	}
	a := event.Agent{Key: t.ID, Parent: m.agentOf(first(t.ParentThreadID, m.main)), Label: first(t.AgentNickname, t.AgentRole), Type: "thread", Status: "running"}
	return []event.Event{m.ev(event.AgentStarted, event.Observed, t.ID, "", "", line, a)}
}

func (m *Mapper) turnCompleted(raw json.RawMessage, agent, turnID string, line []byte) []event.Event {
	var t struct {
		ID         string          `json:"id"`
		Status     string          `json:"status"`
		Error      json.RawMessage `json:"error"`
		DurationMs int64           `json:"durationMs"`
	}
	_ = json.Unmarshal(raw, &t)
	end := event.TurnEnd{Subtype: t.Status, DurationMS: t.DurationMs}
	switch {
	case t.Status == "completed":
		end.Outcome = "completed"
	case t.Status == "interrupted" || (m.interrupted && t.Status != "failed"):
		end.Outcome = "interrupted"
	default:
		end.Outcome = "failed"
		if len(t.Error) > 0 && string(t.Error) != "null" {
			end.Error = problemMessage(t.Error)
		} else {
			end.Error = "turn " + t.Status
		}
	}
	m.interrupted = false
	out := []event.Event{m.ev(event.TurnCompleted, event.Observed, agent, first(t.ID, turnID), "", line, end)}
	if agent == "" && t.DurationMs > 0 {
		out = append(out, m.ev(event.Telemetry, event.Observed, "", first(t.ID, turnID), "", line, event.Metrics{Items: []event.Metric{
			{Name: "duration.ms", Value: float64(t.DurationMs), Unit: "ms", Documented: true, Source: "turn/completed.durationMs"}}}))
	}
	if agent != "" {
		// A child thread's turn is the work its parent handed it.
		a := event.Agent{Key: agent, Type: "thread", Status: end.Outcome}
		k := event.AgentCompleted
		if end.Outcome != "completed" {
			k = event.AgentFailed
			a.Summary = end.Error
		}
		out = append(out, m.ev(k, event.Observed, agent, first(t.ID, turnID), "", line, a))
	}
	return out
}

type item struct {
	Type              string          `json:"type"`
	ID                string          `json:"id"`
	Text              string          `json:"text"`
	Status            string          `json:"status"`
	Command           string          `json:"command"`
	AggregatedOutput  *string         `json:"aggregatedOutput"`
	ExitCode          *int            `json:"exitCode"`
	Changes           []change        `json:"changes"`
	Server            string          `json:"server"`
	Tool              string          `json:"tool"`
	Query             string          `json:"query"`
	Error             json.RawMessage `json:"error"`
	Kind              string          `json:"kind"`
	AgentThreadID     string          `json:"agentThreadId"`
	AgentPath         string          `json:"agentPath"`
	Model             string          `json:"model"` // subAgentActivity, Codex 0.163.0 (P7b); absent before
	SenderThreadID    string          `json:"senderThreadId"`
	ReceiverThreadIDs []string        `json:"receiverThreadIds"`
	AgentsStates      map[string]struct {
		Status  string  `json:"status"`
		Message *string `json:"message"`
	} `json:"agentsStates"`
}

type change struct {
	Path string `json:"path"`
	Diff string `json:"diff"`
}

func (m *Mapper) item(started bool, raw json.RawMessage, agent, turn string, line []byte) []event.Event {
	var it item
	if err := json.Unmarshal(raw, &it); err != nil || it.Type == "" {
		return []event.Event{m.unknown(line, "item", "item not readable")}
	}
	name := "item_completed"
	if started {
		name = "item_started"
	}
	switch it.Type {
	case "userMessage":
		// The owner's message is already a UserMessage, from the write.
		return []event.Event{m.ev(event.MessageMarker, event.Observed, agent, turn, it.ID, line, event.Marker{MessageID: it.ID, Name: name, Block: "user"})}
	case "reasoning":
		return []event.Event{m.ev(event.MessageMarker, event.Observed, agent, turn, it.ID, line, event.Marker{MessageID: it.ID, Name: name, Block: "thinking"})}
	case "agentMessage":
		if started {
			return []event.Event{m.ev(event.MessageMarker, event.Observed, agent, turn, it.ID, line, event.Marker{MessageID: it.ID, Name: name, Block: "text"})}
		}
		return []event.Event{m.ev(event.MessageCompleted, event.AI, agent, turn, it.ID, line, event.Completed{MessageID: it.ID, Text: it.Text})}
	case "subAgentActivity":
		return m.subAgent(it, agent, turn, line)
	case "commandExecution", "fileChange", "mcpToolCall", "dynamicToolCall", "webSearch", "imageView", "imageGeneration", "sleep", "collabAgentToolCall":
		return m.tool(started, it, raw, agent, turn, line)
	}
	return []event.Event{m.unknown(line, "item/"+it.Type, "")}
}

// subAgent maps a child's activity; the child hangs under the thread that
// emitted it, "" for the orchestrator's (events-and-state.md, Agent tree).
func (m *Mapper) subAgent(it item, parent, turn string, line []byte) []event.Event {
	if it.AgentThreadID == "" {
		return []event.Event{m.unknown(line, "item/subAgentActivity", "activity without agentThreadId")}
	}
	if it.AgentThreadID == m.main {
		// Codex 0.163.0: a child reporting to the orchestrator emits
		// "interacted" toward "/root" in its own thread (P7b). The
		// orchestrator is not an agent of zone C.
		return []event.Event{m.status(parent, line, it.Kind+" "+first(it.AgentPath, "orchestrator"), "")}
	}
	a := event.Agent{Key: it.AgentThreadID, Parent: parent, Label: it.AgentPath, Type: "thread", Status: it.Kind, Model: it.Model}
	k := event.AgentProgress
	switch it.Kind {
	case "started":
		k, a.Status = event.AgentStarted, "running"
	case "completed":
		// Codex 0.163.0 also marks a child's end on the parent (P7b).
		k = event.AgentCompleted
	case "interrupted":
		k = event.AgentFailed
	}
	return []event.Event{m.ev(k, event.Observed, it.AgentThreadID, turn, it.ID, line, a)}
}

func (m *Mapper) tool(started bool, it item, raw json.RawMessage, agent, turn string, line []byte) []event.Event {
	name := it.Type
	switch it.Type {
	case "mcpToolCall":
		name = "mcp:" + it.Server + "/" + it.Tool
	case "dynamicToolCall":
		name = "dynamic:" + it.Tool
	case "collabAgentToolCall":
		name = "collabAgent:" + it.Tool
	}
	if it.Type == "fileChange" {
		var paths []string
		for _, c := range it.Changes {
			paths = append(paths, c.Path)
		}
		m.files[it.ID] = strings.Join(paths, ", ")
	}
	t := event.Tool{ToolID: it.ID, Name: name, Status: it.Status}
	if started {
		t.Input = append(json.RawMessage(nil), raw...)
		return []event.Event{m.ev(event.ToolStarted, event.Observed, agent, turn, it.ID, line, t)}
	}
	t.IsError = it.Status == "failed" || it.Status == "declined" || (it.ExitCode != nil && *it.ExitCode != 0)
	switch {
	case it.AggregatedOutput != nil:
		t.Output = *it.AggregatedOutput
	case len(it.Changes) > 0:
		var diffs []string
		for _, c := range it.Changes {
			diffs = append(diffs, c.Diff)
		}
		t.Output = strings.Join(diffs, "\n")
	case len(it.Error) > 0 && string(it.Error) != "null":
		t.Output = problemMessage(it.Error)
	}
	out := []event.Event{m.ev(event.ToolCompleted, event.Observed, agent, turn, it.ID, line, t)}
	if it.Type == "collabAgentToolCall" {
		// The states of the agents the call concerns, in a stable order.
		keys := make([]string, 0, len(it.AgentsStates))
		for k := range it.AgentsStates {
			keys = append(keys, k)
		}
		sortStrings(keys)
		for _, k := range keys {
			s := it.AgentsStates[k]
			a := event.Agent{Key: k, Parent: agent, Type: "thread", Status: s.Status}
			if s.Message != nil {
				a.Summary = *s.Message
			}
			kind := event.AgentProgress
			switch s.Status {
			case "completed":
				kind = event.AgentCompleted
			case "errored", "interrupted", "shutdown", "notFound":
				kind = event.AgentFailed
			}
			out = append(out, m.ev(kind, event.Observed, k, turn, it.ID, line, a))
		}
	}
	return out
}

func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}

func (m *Mapper) tokenUsage(raw json.RawMessage, agent, turn string, line []byte) event.Event {
	var u struct {
		Total struct {
			TotalTokens float64 `json:"totalTokens"`
		} `json:"total"`
		Last struct {
			TotalTokens float64 `json:"totalTokens"`
		} `json:"last"`
		Window *float64 `json:"modelContextWindow"`
	}
	_ = json.Unmarshal(raw, &u)
	const src = "thread/tokenUsage/updated"
	ms := []event.Metric{
		{Name: "tokens.total", Value: u.Total.TotalTokens, Unit: "tokens", Documented: true, Source: src},
		{Name: "tokens.last", Value: u.Last.TotalTokens, Unit: "tokens", Documented: true, Source: src},
	}
	if u.Window != nil {
		// The context: the last call's total over the window, the raw value
		// (telemetry-sources.md, Codex: Context).
		ms = append(ms,
			event.Metric{Name: "context.used", Value: u.Last.TotalTokens, Unit: "tokens", Documented: true, Source: src + ".last.totalTokens"},
			event.Metric{Name: "context.window", Value: *u.Window, Unit: "tokens", Documented: true, Source: src})
	}
	return m.ev(event.Telemetry, event.Observed, agent, turn, "", line, event.Metrics{Items: ms})
}

type window struct {
	UsedPercent float64 `json:"usedPercent"`
	Mins        int64   `json:"windowDurationMins"`
	ResetsAt    int64   `json:"resetsAt"`
}

// rateLimitsRefused maps a refused `account/rateLimits/read`: an API-key
// session is not an error, the values are N/A for that reason; any other
// refusal is an error, and the values turn N/A keeping the last pushed one
// (reliability.md, Telemetry sources: 5-hour, 7-day quota).
func (m *Mapper) rateLimitsRefused(msg string, refused event.Event, line []byte) []event.Event {
	why, names := "read failed: "+msg, []string{"quota.5h", "quota.7d"}
	// The refusal an API-key session gets (telemetry-sources.md, Codex:
	// 5-hour quota; account_processor.rs:1016-1030).
	apiKey := strings.Contains(strings.ToLower(msg), "chatgpt authentication required")
	if apiKey {
		why, names = "API key: ChatGPT sign-in needed", []string{"quota.5h", "quota.7d", "plan", "credits"}
	}
	var ms []event.Metric
	for _, n := range names {
		ms = append(ms, event.Metric{Name: n, Documented: true, Source: "account/rateLimits/read", Reason: why})
	}
	t := m.ev(event.Telemetry, event.Observed, "", "", "", line, event.Metrics{Items: ms})
	if apiKey {
		return []event.Event{t}
	}
	return []event.Event{refused, t}
}

func (m *Mapper) rateLimits(raw json.RawMessage, source string, line []byte) event.Event {
	var l struct {
		Primary   *window `json:"primary"`
		Secondary *window `json:"secondary"`
		PlanType  string  `json:"planType"`
		Credits   *struct {
			HasCredits bool   `json:"hasCredits"`
			Unlimited  bool   `json:"unlimited"`
			Balance    string `json:"balance"`
		} `json:"credits"`
	}
	_ = json.Unmarshal(raw, &l)
	var ms []event.Metric
	if (l.Primary != nil || l.Secondary != nil) && !(l.Primary != nil && l.Primary.Mins == 300) && !(l.Secondary != nil && l.Secondary.Mins == 300) {
		ms = append(ms, event.Metric{Name: "quota.5h", Documented: true, Source: source, Reason: "plan has no 5-hour limit"})
	}
	for _, w := range []*window{l.Primary, l.Secondary} {
		if w == nil {
			continue
		}
		name := ""
		switch w.Mins {
		case 300:
			name = "quota.5h"
		case 10080:
			name = "quota.7d"
		default:
			name = "quota." + strconv.FormatInt(w.Mins, 10) + "m"
		}
		var resets time.Time
		if w.ResetsAt != 0 {
			resets = time.Unix(w.ResetsAt, 0).UTC()
		}
		ms = append(ms, event.Metric{Name: name, Value: w.UsedPercent, Unit: "%", ResetsAt: resets, Documented: true, Source: source})
	}
	if l.PlanType != "" {
		ms = append(ms, event.Metric{Name: "plan", Text: l.PlanType, Documented: true, Source: source})
	}
	if c := l.Credits; c != nil {
		t := "none"
		switch {
		case c.Unlimited:
			t = "unlimited"
		case c.HasCredits:
			t = "balance " + c.Balance
		}
		ms = append(ms, event.Metric{Name: "credits", Text: t, Documented: true, Source: source + ".credits"})
	}
	return m.ev(event.Telemetry, event.Observed, "", "", "", line, event.Metrics{Items: ms})
}

// plan maps `turn/plan/updated`: the turn's whole plan, observed. Schema
// only, not recorded: built from the v2 schema's TurnPlanUpdatedNotification
// (p0-codex-app-server-schema-v2.json); no probe saw one (codex-cli.md,
// section 6).
func (m *Mapper) plan(params json.RawMessage, agent, turn string, line []byte) event.Event {
	var p struct {
		Explanation *string `json:"explanation"`
		Plan        []struct {
			Step   string `json:"step"`
			Status string `json:"status"`
		} `json:"plan"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return m.unknown(line, "turn/plan/updated", "plan not readable: "+err.Error())
	}
	var out event.Plan
	if p.Explanation != nil {
		out.Explanation = *p.Explanation
	}
	for _, s := range p.Plan {
		out.Steps = append(out.Steps, event.PlanStep{Step: s.Step, Status: s.Status})
	}
	return m.ev(event.StageChanged, event.Observed, agent, turn, "", line, out)
}
