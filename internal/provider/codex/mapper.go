// Package codex is the Codex CLI adapter: `codex app-server` over stdio, JSON-RPC
// 2.0 lines in both directions (doc/architecture.md, Codex adapter). The
// app-server is documented but marked experimental by Codex; the wire shapes
// here are those of the v2 schema and of the plan 1 recordings (P6, P6b, P7).
// It does not read `codex exec --json` output (a different protocol).
package codex

import (
	"encoding/json"
	"path/filepath"
	"slices"
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
	steps       steps             // where the orchestrator's model steps begin (plan 22, X5)
	totals      map[string]string // each thread's last usage total, as sent, to know a repeat (plan 22)
	// The session's models (plan 34, models.go): the pages read so far of
	// the listing under way, the last whole list; the turn/start requests
	// that carried a change; whether a thread/read is due (readForChange
	// when it answers a change), written (readSent), and the reads that
	// answer a change, by request id.
	listing       []event.ModelEntry
	cursors       map[string]bool // the listing's cursors so far, to stop one that repeats
	nextPage      string          // the cursor of the page the session asks for next, "" for none
	catalogue     []event.ModelEntry
	carried       map[string]bool
	readDue       bool
	readForChange bool
	readSent      bool
	answering     map[string]bool
	// A change written and not yet answered (plan 38, models.go): the
	// thread/settings/update requests by id, and whether a change sent by
	// one of them or by a turn/start still waits for its answer.
	updates     map[string]updateKind
	outstanding bool
	// modeOutstanding: a mode written by thread/settings/update and not yet
	// answered (plan 39).
	modeOutstanding bool
	// replies are the CLI's requests the session answers itself, other than
	// approvals, by id: their answers map as writes (currentTime/read).
	replies map[string]string
	// resumeAsked is the thread a resume from the home screen asked for,
	// until thread/resume answers (plan 69, design D7): Codex's refusal of it
	// for want of the thread is reported as for good.
	resumeAsked string
}

// ResumeAsked tells the mapper the thread asked for is the project's last,
// resumed from the home screen (plan 69, design D7).
func (m *Mapper) ResumeAsked(id string) { m.resumeAsked = id }

// noThread says error is Codex's refusal of thread/resume for want of the
// thread id (P69b, Codex 0.163.0): -32600 "no rollout found for thread id
// <id>", or "invalid session id: …" for an id no thread can have. The words
// are undocumented, matched as 0.163.0 writes them; others are another
// failure (plan 69, design D7).
func noThread(raw json.RawMessage, id string) bool {
	var e struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	if json.Unmarshal(raw, &e) != nil || e.Code != -32600 {
		return false
	}
	return e.Message == "no rollout found for thread id "+id || strings.HasPrefix(e.Message, "invalid session id: ")
}

// updateKind is what a thread/settings/update carried: a model or effort
// change (plan 38), a mode (plan 39), or both.
type updateKind struct{ model, mode bool }

type serverReq struct {
	agent, turn, item, tool string
}

// NewMapper returns an empty mapper.
func NewMapper() *Mapper {
	return &Mapper{pending: map[string]string{}, server: map[string]serverReq{}, files: map[string]string{}, totals: map[string]string{},
		carried: map[string]bool{}, answering: map[string]bool{}, updates: map[string]updateKind{}, replies: map[string]string{}}
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

// Out maps one line read from the CLI's stdout. The events keep no image:
// the app-server writes the owner's images back in its userMessage items
// (plan 31, seen live in 31.8), so each event's raw line loses its image
// data URLs once the line is mapped (keepNoImages).
func (m *Mapper) Out(raw []byte) []event.Event {
	return keepNoImages(m.out(raw))
}

func (m *Mapper) out(raw []byte) []event.Event {
	if evs, ok := m.configRead(raw); ok {
		return evs
	}
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
				URL  string `json:"url"`
				Path string `json:"path"`
			} `json:"input"`
			Reviewer string          `json:"approvalsReviewer"`
			Policy   string          `json:"approvalPolicy"`
			Sandbox  json.RawMessage `json:"sandboxPolicy"`
			Cursor   string          `json:"cursor"`
			Model    string          `json:"model"`
			Effort   string          `json:"effort"`
		}
		_ = json.Unmarshal(r.Params, &p)
		switch r.Method {
		case "thread/start": // a new main thread: its thread/started names it
			m.main = ""
		case "thread/resume":
			m.main, m.session = p.ThreadID, p.ThreadID
		case "model/list":
			if p.Cursor == "" { // a new listing begins (plan 34)
				m.listing, m.cursors = nil, map[string]bool{}
			}
		case "turn/start":
			if id != "" && (p.Model != "" || p.Effort != "") {
				m.carried[id] = true // a model or effort change goes with this turn (plan 34)
				m.outstanding = true
			}
		case "thread/settings/update": // a change written at once (plans 38 and 39)
			k := updateKind{model: p.Model != "" || p.Effort != "", mode: len(p.Sandbox) > 0 || p.Policy != "" || p.Reviewer != ""}
			if id != "" {
				m.updates[id] = k
			}
			m.outstanding = m.outstanding || k.model
			m.modeOutstanding = m.modeOutstanding || k.mode
		case "thread/read":
			if id != "" && m.readSent {
				m.answering[id], m.readSent = true, false
			}
		}
		agent := m.agentOf(p.ThreadID)
		var out []event.Event
		switch r.Method {
		case "turn/start":
			var texts []string
			var imgs []event.Image
			for _, in := range p.Input {
				switch in.Type {
				case "text":
					texts = append(texts, in.Text)
				case "image":
					imgs = append(imgs, dataURLImage(in.URL))
				case "localImage":
					imgs = append(imgs, event.Image{Name: filepath.Base(in.Path)})
				}
			}
			if len(imgs) > 0 {
				raw = withoutImageURLs(raw) // the events keep no image bytes (plan 31)
			}
			out = append(out, m.ev(event.UserMessage, event.Observed, agent, "", "", raw, event.User{Text: strings.Join(texts, "\n"), Images: imgs}))
		case "turn/interrupt":
			m.interrupted = true
		}
		out = append(out, m.ev(event.Sent, event.Observed, agent, p.TurnID, "", raw, event.Write{What: r.Method, RequestID: id}))
		if r.Method == "turn/start" && (p.Reviewer != "" || len(p.Sandbox) > 0) {
			// The mode in force for this turn (plan 12, item 2; plan 39): its
			// sandbox, policy and reviewer, with the sandbox the turn names
			// (plan 12 addendum).
			out = append(out, m.ev(event.ApprovalMode, event.Observed, agent, "", "", raw, event.Mode{Mode: codexMode(namedValue(p.Sandbox), p.Policy, p.Reviewer), Sandbox: namedValue(p.Sandbox)}))
		}
		return out
	case r.hasID():
		id := idString(r.ID)
		if what, ok := m.replies[id]; ok {
			// The session's own answer to a request that is no approval.
			delete(m.replies, id)
			return []event.Event{m.ev(event.Sent, event.Observed, "", "", "", raw, event.Write{What: what + " reply", RequestID: id})}
		}
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
			a.Decision = decisionOf(decision(r.Result))
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
		p := event.Problem{Message: what + " refused by Codex: " + msg}
		// A refused turn/start started no turn: the message goes back to the
		// editor (plan 34).
		p.TurnRefused = method == "turn/start"
		// The session works without its model list: a refused page never
		// ends the handshake (plan 46, design D2).
		p.Optional = method == "model/list"
		if method == "thread/resume" && m.resumeAsked != "" {
			// The home screen's resume refused: for good only when Codex has
			// no such thread (plan 69, design D7).
			p.NoSession = noThread(r.Error, m.resumeAsked)
			m.resumeAsked = ""
		}
		refused := m.ev(event.Error, event.Observed, "", "", "", raw, p)
		switch method {
		case "account/rateLimits/read":
			return m.rateLimitsRefused(msg, refused, raw)
		case "turn/start":
			if m.carried[id] {
				// The change it carried is not taken either (plan 34): zone B
				// says why; the settings in force are unchanged.
				delete(m.carried, id)
				m.outstanding = false
				return []event.Event{refused, m.ev(event.ModelSettings, event.Observed, "", "", "", raw, event.Settings{Source: method, Answers: true, Refused: true,
					Text: "the model change was not taken: " + p.Message})}
			}
		case "thread/settings/update":
			k, seen := m.updates[id]
			delete(m.updates, id)
			if !seen {
				k.model = true // as plan 38 read every update
			}
			if k.model {
				m.outstanding = false
			}
			if k.mode {
				m.modeOutstanding = false
			}
			var out []event.Event
			if unsupported(r.Error, method) {
				// No change at once with this Codex: the change waits for the
				// next turn/start, as every later one (plan 38, design D4;
				// plan 39).
				if k.model {
					out = append(out, m.ev(event.ModelSettings, event.Observed, "", "", "", raw, event.Settings{Source: method, NextTurn: true,
						Text: "Codex takes model changes with the next message: " + p.Message}))
				}
				if k.mode {
					out = append(out, m.ev(event.ApprovalMode, event.Observed, "", "", "", raw, event.Mode{NextTurn: true,
						Text: "Codex takes mode changes with the next message: " + p.Message}))
				}
				return out
			}
			out = append(out, refused)
			if k.model {
				out = append(out, m.ev(event.ModelSettings, event.Observed, "", "", "", raw, event.Settings{Source: method, Answers: true, Refused: true,
					Text: "the model change was not taken: " + p.Message}))
			}
			if k.mode {
				out = append(out, m.ev(event.ApprovalMode, event.Observed, "", "", "", raw, event.Mode{Refused: true,
					Text: "the mode was not changed: " + p.Message}))
			}
			return out
		case "configRequirements/read":
			// An older Codex without the read: every mode shown, Codex
			// refusing what it does not allow (plan 39, design D1).
			return []event.Event{m.status("", raw, "", "configRequirements/read refused: "+msg)}
		case "model/list":
			if len(m.catalogue) == 0 {
				// No list yet: the selector says why rather than wait for one
				// (plan 34).
				m.listing = nil
				return []event.Event{refused, m.ev(event.ModelsListed, event.Observed, "", "", "", raw, event.Models{Source: method, Problem: p.Message})}
			}
		case "thread/read":
			if m.answering[id] {
				delete(m.answering, id)
				return []event.Event{refused, m.ev(event.ModelSettings, event.Observed, "", "", "", raw, event.Settings{Source: method, Answers: true,
					Text: "the model and effort in force are not known: " + msg})}
			}
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
		if m.carried[id] {
			// The turn took the change it carried: thread/read says what
			// is in force (plan 34).
			delete(m.carried, id)
			m.readDue, m.readForChange = true, true
		}
		return []event.Event{e}
	case "model/list":
		return m.listPage(r.Result, raw)
	case "thread/read":
		return m.threadRead(id, r.Result, raw)
	case "turn/interrupt":
		return []event.Event{m.status("", raw, "interrupt acknowledged", "")}
	case "configRequirements/read":
		return []event.Event{m.modeLimits(r.Result, raw)}
	case "thread/settings/update":
		// Queued, not yet applied: thread/settings/updated answers the change
		// (plan 38, design D2; P38x).
		delete(m.updates, id)
		return []event.Event{m.status("", raw, "settings update acknowledged", "")}
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
		Effort         *string         `json:"reasoningEffort"`
		Cwd            string          `json:"cwd"`
		CliVersion     string          `json:"cliVersion"`
		Sandbox        json.RawMessage `json:"sandbox"`
		ApprovalPolicy json.RawMessage `json:"approvalPolicy"`
		Reviewer       string          `json:"approvalsReviewer"`
	}
	m.resumeAsked = "" // answered: a later refusal is no longer the resume's (plan 69)
	if err := json.Unmarshal(r.Result, &s); err != nil || s.Thread.ID == "" {
		return []event.Event{m.ev(event.Error, event.Observed, "", "", "", raw, event.Problem{Message: "thread/start response without a thread id"})}
	}
	m.main, m.session = s.Thread.ID, s.Thread.ID
	st := event.Started{SessionID: s.Thread.ID, Model: s.Model, Cwd: first(s.Cwd, s.Thread.Cwd), Version: first(s.Thread.CliVersion, s.CliVersion),
		Sandbox: namedValue(s.Sandbox), ApprovalPolicy: namedValue(s.ApprovalPolicy),
		ApprovalMode: codexMode(namedValue(s.Sandbox), namedValue(s.ApprovalPolicy), s.Reviewer)}
	return []event.Event{
		m.ev(event.SessionReady, event.Observed, "", "", "", raw, event.Ready{Detail: "thread/start response"}),
		m.ev(event.SessionStarted, event.Observed, "", "", "", raw, st),
		// The model and effort the thread starts with (plan 34, P34x).
		m.ev(event.ModelSettings, event.Observed, "", "", "", raw, settingsOf(s.Model, s.Effort, "thread/start")),
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
	case "currentTime/read":
		// Codex's experimental API (plan 38, design D5): the session answers
		// it with the time, as OpenAI's Codex extension for VS Code does.
		var p ids
		_ = json.Unmarshal(r.Params, &p)
		m.replies[idString(r.ID)] = r.Method
		return []event.Event{m.status(m.agentOf(p.ThreadID), raw, "current time asked", "")}
	default:
		return []event.Event{m.ev(event.Unrecognised, event.Observed, "", "", "", raw, event.Unknown{Type: "serverRequest/" + r.Method, Reason: "server request the dashboard does not handle"})}
	}
	id := idString(r.ID)
	var p struct {
		ids
		Reason  *string            `json:"reason"`
		Command string             `json:"command"`
		Offered *[]json.RawMessage `json:"availableDecisions"` // experimental field: sent, but not in 0.163.0's generated schema (P8)
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
	if p.Offered != nil {
		for _, d := range *p.Offered {
			var s string
			if json.Unmarshal(d, &s) == nil { // the amendments are objects: never kept, never offered
				a.Offered = append(a.Offered, s)
			}
		}
	}
	// Allow for the session: 0.163.0's generated schema documents
	// acceptForSession for both requests; a file-change request comes
	// without a list and takes it, a command request lists its decisions
	// (P8: accept, the amendment, cancel), and the list decides (plan 8).
	a.ForSession = a.Problem == "" && (p.Offered == nil || slices.Contains(a.Offered, "acceptForSession"))
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
		FromModel string          `json:"fromModel"`
		ToModel   string          `json:"toModel"`
		Reason    json.RawMessage `json:"reason"`
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
	case "model/rerouted":
		// Codex answered a turn with another model (plan 34): zone B says so,
		// and thread/read shows what the thread is configured with since.
		why := namedValue(p.Reason)
		text := "Codex answered with " + p.ToModel + " instead of " + p.FromModel
		if why != "" {
			text += " (" + why + ")"
		}
		if agent == "" {
			m.readDue = true
		}
		return []event.Event{m.ev(event.StatusChanged, event.Observed, agent, p.TurnID, "", raw, event.Status{Status: "notice", Detail: text})}
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
		if agent == "" {
			m.steps.turnStarted()
		}
		return []event.Event{m.ev(event.TurnStarted, event.Observed, agent, first(t.ID, p.TurnID), "", raw, nil)}
	case "turn/completed":
		return m.turnCompleted(p.Turn, agent, p.TurnID, raw)
	case "item/started", "item/completed":
		return m.item(r.Method == "item/started", p.Item, agent, p.TurnID, raw)
	case "item/agentMessage/delta":
		return []event.Event{m.ev(event.MessageDelta, event.AI, agent, p.TurnID, p.ItemID, raw, event.Delta{MessageID: p.ItemID, Block: "text", Text: p.Delta})}
	case "item/reasoning/summaryTextDelta", "item/reasoning/textDelta":
		return []event.Event{m.ev(event.MessageDelta, event.AI, agent, p.TurnID, p.ItemID, raw, event.Delta{MessageID: p.ItemID, Block: "thinking", Text: p.Delta})}
	case "item/reasoning/summaryPartAdded":
		// A new part of a reasoning item's summary begins (schema:
		// ReasoningSummaryPartAddedNotification, itemId and summaryIndex; P22s
		// line 68, 2026-10-21). Known, and it gives no event: the part's text
		// comes in the summaryTextDelta that follows and in the item's
		// completion, whose marker carries the whole summary (plan 22,
		// amendment 20).
		return nil
	case "item/commandExecution/outputDelta":
		// First seen in P7b (Codex 0.163.0): a running command's output
		// (events-and-state.md, Tool lifecycle).
		return []event.Event{m.ev(event.ToolProgress, event.Observed, agent, p.TurnID, p.ItemID, raw, event.Tool{ToolID: p.ItemID, Output: p.Delta})}
	case "thread/tokenUsage/updated":
		ev, repeat := m.tokenUsage(p.TokenUse, agent, p.TurnID, raw)
		out := []event.Event{ev}
		if agent == "" && !repeat && m.steps.usage() {
			// The step's response is complete and its tools are done: the next
			// step begins (plan 22, design 4, criterion 1.2).
			out = append(out, m.stepStarted(p.TurnID, raw))
		}
		return out
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
		return []event.Event{m.ev(event.ApprovalResolved, event.Observed, req.agent, req.turn, req.item, raw, event.Approval{ID: id, Tool: req.tool, ToolID: req.item, Decision: event.Withdrawn})}
	case "error":
		msg := problemMessage(p.Error)
		k := event.Error
		if p.WillRetry {
			k = event.Retry
		}
		return []event.Event{m.ev(k, event.Observed, agent, p.TurnID, "", raw, event.Problem{Message: msg, WillRetry: p.WillRetry})}
	case "warning":
		return []event.Event{m.status(agent, raw, "warning", p.Message)}
	case "item/autoApprovalReview/started", "item/autoApprovalReview/completed":
		// [UNSTABLE] in the v2 schema (P12x): one review's progress, then its verdict.
		var rv struct {
			ReviewID     string `json:"reviewId"`
			TargetItemID string `json:"targetItemId"`
			Review       struct {
				Status        string `json:"status"`
				RiskLevel     string `json:"riskLevel"`
				Authorization string `json:"userAuthorization"`
				Rationale     string `json:"rationale"`
			} `json:"review"`
			Action struct {
				Command string `json:"command"`
			} `json:"action"`
		}
		_ = json.Unmarshal(r.Params, &rv)
		return []event.Event{m.ev(event.ReviewUpdated, event.Observed, agent, p.TurnID, rv.TargetItemID, raw, event.Review{ReviewID: rv.ReviewID, Command: rv.Action.Command,
			Status: rv.Review.Status, Risk: rv.Review.RiskLevel, Authorization: rv.Review.Authorization, Rationale: rv.Review.Rationale})}
	case "guardianWarning":
		// [UNSTABLE]: repeats the review's verdict as a sentence; kept in the event log.
		return []event.Event{m.ev(event.ReviewWarning, event.Observed, agent, p.TurnID, "", raw, event.Line{Text: p.Message})}
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
	case "thread/settings/updated":
		return m.settingsUpdated(r.Params, raw)
	case "turn/moderationMetadata":
		// First seen in P38t (Codex 0.163.0), in the stable schema too: the
		// turn's moderation flags, logged (plan 38, design D5).
		return []event.Event{m.status(agent, raw, "moderation metadata", "")}
	case "thread/readState/changed":
		// First seen in P38t, in the stable schema too: the thread's first
		// unread position, for clients that mark threads read; logged.
		return []event.Event{m.status(agent, raw, "read state changed", "")}
	case "thread/goal/cleared":
		// First seen in P69b (Codex 0.163.0), in the stable schema too: sent
		// after every thread/resume, the thread's goal cleared; logged (plan
		// 69, design D5).
		return []event.Event{m.status(agent, raw, "goal cleared", "")}
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
	if agent == "" {
		m.steps.turnEnded()
	}
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
	Model             string          `json:"model"`           // subAgentActivity, Codex 0.163.0 (P7b); absent before
	ReasoningEffort   string          `json:"reasoningEffort"` // beside model (P7b; plan 6, item 6)
	Prompt            string          `json:"prompt"`          // a spawnAgent call's task (P9a; plan 10, item 1)
	Summary           json.RawMessage `json:"summary"`         // a reasoning item's summary, a list of strings (schema: ReasoningThreadItem; plan 22, amendment 20)
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
		out := []event.Event{m.ev(event.MessageMarker, event.Observed, agent, turn, it.ID, line, event.Marker{MessageID: it.ID, Name: name, Block: "user"})}
		if agent == "" && started && m.steps.message() {
			// The turn's first step begins (plan 22, design 4, criterion 1.2).
			out = append(out, m.stepStarted(turn, line))
		}
		return out
	case "reasoning":
		mk := event.Marker{MessageID: it.ID, Name: name, Block: "thinking"}
		unreadable := false
		if !started {
			// The item's summary rides on its completion (plan 22, amendment 20).
			mk.Summary, unreadable = reasoningSummary(it.Summary)
		}
		out := []event.Event{m.ev(event.MessageMarker, event.Observed, agent, turn, it.ID, line, mk)}
		if unreadable {
			// Kept visible, not shown as "Codex sent none": the marker stands.
			out = append(out, m.unknown(line, "item/reasoning", "summary is not a list of strings"))
		}
		return out
	case "agentMessage":
		if started {
			return []event.Event{m.ev(event.MessageMarker, event.Observed, agent, turn, it.ID, line, event.Marker{MessageID: it.ID, Name: name, Block: "text"})}
		}
		return []event.Event{m.ev(event.MessageCompleted, event.AI, agent, turn, it.ID, line, event.Completed{MessageID: it.ID, Text: it.Text})}
	case "subAgentActivity":
		if agent == "" {
			m.steps.subAgent(it.Kind)
		}
		return m.subAgent(it, agent, turn, line)
	case "commandExecution", "fileChange", "mcpToolCall", "dynamicToolCall", "webSearch", "imageView", "imageGeneration", "sleep", "collabAgentToolCall":
		out := m.tool(started, it, raw, agent, turn, line)
		if agent == "" && m.steps.tool(started, it.ID) {
			// The step's last tool is done and its response was already
			// complete: the next step begins (plan 22, design 4).
			out = append(out, m.stepStarted(turn, line))
		}
		return out
	}
	return []event.Event{m.unknown(line, "item/"+it.Type, "")}
}

// reasoningSummary is a reasoning item's summary as one text: the parts of its
// summary list (schema: ReasoningThreadItem.summary, a list of strings) joined
// by a blank line, each as Codex sent it; a part with no text in it is left
// out. A field that is missing, null or an empty list, and a list with no text
// in it, give "": Codex sent none. unreadable is true when the field is there
// and is not a list of strings (plan 22, amendment 20; P22s lines 23 and 70).
func reasoningSummary(raw json.RawMessage) (text string, unreadable bool) {
	if len(raw) == 0 {
		return "", false
	}
	var parts []string
	if json.Unmarshal(raw, &parts) != nil {
		return "", true
	}
	parts = slices.DeleteFunc(parts, func(p string) bool { return strings.TrimSpace(p) == "" })
	return strings.Join(parts, "\n\n"), false
}

// stepStarted is the marker of a model step's start: inferred, the
// orchestrator's, in the turn of the event that gave it away and with that
// event's line. It names no message (plan 22, X5; event.StepStarted).
func (m *Mapper) stepStarted(turn string, line []byte) event.Event {
	return m.ev(event.MessageMarker, event.Inferred, "", turn, "", line, event.Marker{Name: event.StepStarted})
}

// steps infers where the orchestrator's model steps begin. Codex reports no
// step's start, and a reasoning item only once its reasoning ended (plan 22,
// X1 and X5), so the adapter marks each start from what Codex does report:
// the owner's message, a step's usage and its tools (plan 22, design 4). It
// follows the main thread alone, where the turn runs; a child thread's events
// never reach it. A step's response is complete when its usage comes (X2);
// the next step begins once that and the step's tools are both done, and only
// if the step made a call. A step that made none, a message alone, begins
// nothing: the turn is over. Recording: P7's orchestrator gives 3,380 (the
// message), 7,120, 9,632, 13,471 and 24,076 ms, and none after the last
// step, a message alone (testdata/probes/p7-codex-subagents.jsonl lines 20,
// 30, 54, 118 and 161; the last step's usage is line 167).
type steps struct {
	turn     bool            // the main thread's turn runs
	running  map[string]bool // its tool items running, by item id
	complete bool            // the step's response is complete: its usage came
	call     bool            // the step made a call: a tool item or a spawn
}

// begin starts a step: nothing runs, no response is complete, no call made.
func (s *steps) begin() {
	clear(s.running)
	s.complete, s.call = false, false
}

// turnStarted follows the main thread's turn/started, turnEnded its
// turn/completed: the state starts and ends empty with the turn.
func (s *steps) turnStarted() { s.begin(); s.turn = true }
func (s *steps) turnEnded()   { s.begin(); s.turn = false }

// message follows an item/started userMessage of the main thread and reports
// whether the turn's first step begins: only while its turn runs, so a replayed
// message outside a turn starts none.
func (s *steps) message() bool {
	if !s.turn {
		return false
	}
	s.begin()
	return true
}

// subAgent follows a subAgentActivity item of the main thread, of kind kind:
// a spawn, kind "started", is a call (plan 22, criterion 1.2). Codex 0.145.0
// sends it as an item/completed of kind "started" (P7); 0.163.0 as both
// items, and also tells a child's end on its parent, kind "completed" (P7b,
// P7c), which is not the orchestrator's call: counted, it would start a step
// after a turn's last message, a card for nothing. Counting only "started"
// gives the same markers on every recording.
func (s *steps) subAgent(kind string) {
	if kind == "started" {
		s.call = true
	}
}

// tool follows a tool item of the main thread, started or completed, and
// reports whether the next step begins: its last running tool just completed
// after the response was complete, and the step made a call. An item/completed
// with no item/started was not running (a resume, a replay): it is a call
// and ends nothing.
func (s *steps) tool(started bool, id string) bool {
	s.call = true
	if started {
		if s.running == nil {
			s.running = map[string]bool{}
		}
		s.running[id] = true
		return false
	}
	if !s.running[id] {
		return false
	}
	delete(s.running, id)
	if !s.turn || !s.complete || len(s.running) > 0 {
		return false
	}
	s.begin()
	return true
}

// usage follows a thread/tokenUsage/updated of the main thread, the end of a
// step's response (X2), and reports whether the next step begins: no tool is
// still running and the step made a call. With a tool still running, the
// step begins when the last one completes (tool). Outside a turn, a usage
// replayed by a resume (X5) changes nothing.
func (s *steps) usage() bool {
	if !s.turn {
		return false
	}
	s.complete = true
	if len(s.running) > 0 || !s.call {
		return false
	}
	s.begin()
	return true
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
	a := event.Agent{Key: it.AgentThreadID, Parent: parent, Label: it.AgentPath, Type: "thread", Status: it.Kind, Model: it.Model, Effort: it.ReasoningEffort}
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
	if it.ExitCode != nil {
		t.ExitCode, t.HasExitCode = *it.ExitCode, true // plan 6, item 1
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
		// A spawn request's model and effort, for each agent it created (plan
		// 6, items 4 and 6; the schema's `model`, "Model requested for the
		// spawned agent", and `reasoningEffort` beside it), and the label and
		// description its prompt gives an agent Codex names nowhere else
		// (multi-agent v1, P9a; plan 10, item 1).
		label := spawnLabel(it.Prompt)
		if it.Tool == "spawnAgent" && (it.Model != "" || it.ReasoningEffort != "" || label != "") {
			for _, id := range it.ReceiverThreadIDs {
				a := event.Agent{Key: id, Parent: agent, Model: it.Model, Effort: it.ReasoningEffort, ModelRequested: true}
				if label != "" {
					a.Label, a.Description = label, strings.TrimSpace(it.Prompt)
				}
				out = append(out, m.ev(event.AgentProgress, event.Observed, id, turn, it.ID, line, a))
			}
		}
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

// spawnLabel is the label a spawn request's prompt gives its agent: "task: "
// and the prompt's first non-blank line, cut at 24 characters with "…"; ""
// for a blank prompt (plan 10, item 1).
func spawnLabel(prompt string) string {
	for _, l := range strings.Split(prompt, "\n") {
		if l = strings.TrimSpace(l); l != "" {
			if r := []rune(l); len(r) > 24 {
				l = strings.TrimRight(string(r[:24]), " ") + "…"
			}
			return "task: " + l
		}
	}
	return ""
}

func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}

// tokenUsage maps a thread's usage update; repeat is true when its total is
// the thread's previous one, as sent: Codex 0.145.0 sends the last update
// again when a turn is interrupted (testdata/probes/p6b-codex-app-server.jsonl
// line 87, the same as line 64), which reports no new model response.
func (m *Mapper) tokenUsage(raw json.RawMessage, agent, turn string, line []byte) (ev event.Event, repeat bool) {
	var u struct {
		TotalRaw json.RawMessage `json:"total"`
		Total    struct {
			TotalTokens float64 `json:"totalTokens"`
		} `json:"-"`
		Last struct {
			TotalTokens     float64 `json:"totalTokens"`
			ReasoningTokens any     `json:"reasoningOutputTokens"` // a number, else no count
		} `json:"last"`
		Window *float64 `json:"modelContextWindow"`
	}
	_ = json.Unmarshal(raw, &u)
	_ = json.Unmarshal(u.TotalRaw, &u.Total)
	if len(u.TotalRaw) > 0 {
		prev, seen := m.totals[agent]
		repeat = seen && prev == string(u.TotalRaw)
		m.totals[agent] = string(u.TotalRaw)
	}
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
	if n, ok := u.Last.ReasoningTokens.(float64); ok && !repeat {
		// The reasoning tokens of the model response this update closes, a
		// step's, by thread (plan 22, X2): the orchestrator of
		// testdata/probes/p7-codex-subagents.jsonl gives 83, 0, 8, 7 and 0
		// (lines 30, 54, 118, 161 and 167), its children their own. last is
		// the difference of two consecutive totals, so it never counts
		// twice. The field is defined in the app-server schema
		// (p0-codex-app-server-schema-v2.json, TokenUsageBreakdown,
		// required); when the update is sent is not documented: observed,
		// once per model response. A line without the number gives no
		// metric, never a zero. A repeated update reports no model response:
		// its count would be the previous step's again (plan 22, the review).
		ms = append(ms, event.Metric{Name: event.ReasoningLast, Value: n, Unit: "tokens", Documented: true, Source: src + " last.reasoningOutputTokens"})
	}
	return m.ev(event.Telemetry, event.Observed, agent, turn, "", line, event.Metrics{Items: ms}), repeat
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

// configRead maps the reply to the dashboard's `config/read`, before any
// other step touches the line. The reply is the effective configuration and
// can hold secrets (an MCP server's env, a provider's bearer token), so only
// agents.max_concurrent_threads_per_session (or max_threads) is decoded, and no
// event carries a byte of the result: Raw is a fixed placeholder. Unset, the
// reply has no `agents` key (probe 2026-10-17, Codex CLI 0.163.0). ok is
// false for any other line.
func (m *Mapper) configRead(raw []byte) ([]event.Event, bool) {
	var h struct {
		ID     json.RawMessage `json:"id"`
		Method string          `json:"method"`
	}
	if json.Unmarshal(raw, &h) != nil || h.Method != "" || len(h.ID) == 0 || string(h.ID) == "null" {
		return nil, false
	}
	id := idString(h.ID)
	if m.pending[id] != "config/read" {
		return nil, false
	}
	delete(m.pending, id)
	q, _ := json.Marshal(id)
	kept := []byte(`{"id":` + string(q) + `,"result":"config/read reply not kept: it can hold secrets"}`)
	var r struct {
		Result json.RawMessage `json:"result"`
		Error  json.RawMessage `json:"error"`
	}
	_ = json.Unmarshal(raw, &r)
	if len(r.Error) > 0 && string(r.Error) != "null" {
		// The error's message carries no configuration; it is shown as any
		// refusal, one the session works without (plan 46, design D2).
		msg := problemMessage(r.Error)
		return []event.Event{m.ev(event.Error, event.Observed, "", "", "", kept, event.Problem{Message: "config/read refused by Codex: " + msg, Optional: true})}, true
	}
	var c struct {
		Config struct {
			Agents *struct {
				PerSession *int `json:"max_concurrent_threads_per_session"`
				MaxThreads *int `json:"max_threads"`
			} `json:"agents"`
			Features struct {
				// true, false or a table; only the table's limit is decoded.
				V2 json.RawMessage `json:"multi_agent_v2"`
			} `json:"features"`
		} `json:"config"`
	}
	_ = json.Unmarshal(r.Result, &c)
	item := event.Metric{Name: event.LimitMetric, Text: event.LimitNotSet, Documented: true, Source: "config/read"}
	n := 0
	if a := c.Config.Agents; a != nil {
		p := a.PerSession
		if p == nil {
			p = a.MaxThreads
		}
		if p != nil && *p > 0 {
			n = *p
		}
	}
	if n > 0 {
		item.Text, item.Value, item.Unit = "", float64(n), "threads"
	}
	// The V2 backend's own limit counts the main thread (Codex 0.163.0,
	// codex-rs/core/src/config/mod.rs:1666-1680); it overrides agents' on V2
	// only, and the active backend is not reported: a limit only when both
	// backends agree (doc/plan5-evaluation.md, After the commit).
	var v2 struct {
		Max *int `json:"max_concurrent_threads_per_session"`
	}
	if json.Unmarshal(c.Config.Features.V2, &v2) == nil && v2.Max != nil && *v2.Max > 0 && (n == 0 || *v2.Max-1 != n) {
		item.Text, item.Value, item.Unit = event.LimitPerBackend, 0, ""
	}
	return []event.Event{m.ev(event.Telemetry, event.Observed, "", "", "", kept, event.Metrics{Items: []event.Metric{item}})}, true
}

// decisionOf is a decision in the dashboard's words (plan 49, design D10),
// app-server's and the older words the reducer took with them; a word it
// does not know, an amendment's or one to come, stays as it is, which the
// reducer keeps as an expiry.
func decisionOf(word string) event.Decision {
	switch word {
	case "accept", "approved":
		return event.Allow
	case "acceptForSession":
		return event.AllowSession
	case "decline", "denied":
		return event.Deny
	case "cancel", "abort":
		return event.Cancel
	}
	return event.Decision(word)
}
