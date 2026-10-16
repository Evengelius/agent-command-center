// Package event defines the normalised events every adapter produces and the
// reducer consumes (doc/events-and-state.md). It has no behaviour: types only.
package event

import (
	"encoding/json"
	"time"
)

// Kind is one conceptual event (events-and-state.md, Conceptual events).
type Kind string

// The conceptual events of events-and-state.md, plus the few the code needs to
// replay a whole recording: the process start, the handshake, the user's own
// message as written, other writes to the CLI, stream markers and stderr.
const (
	MessageDelta     Kind = "conversation.message.delta"
	MessageCompleted Kind = "conversation.message.completed"
	MessageMarker    Kind = "conversation.message.marker" // stream start/stop markers; no text
	UserMessage      Kind = "conversation.user"           // the owner's message as written to the CLI
	ToolStarted      Kind = "tool.started"
	ToolProgress     Kind = "tool.progress" // output or a heartbeat of a running tool
	ToolCompleted    Kind = "tool.completed"
	StageChanged     Kind = "workflow.stage.changed"
	AgentStarted     Kind = "agent.started"
	AgentProgress    Kind = "agent.progress"
	AgentCompleted   Kind = "agent.completed"
	AgentFailed      Kind = "agent.failed"
	AgentSet         Kind = "agent.set" // the full set of live background tasks (replace, not merge)
	ApprovalRequest  Kind = "approval.requested"
	ApprovalResolved Kind = "approval.resolved"
	Telemetry        Kind = "telemetry.updated"
	SessionSpawned   Kind = "session.spawned" // the process started
	SessionReady     Kind = "session.ready"   // the handshake completed
	SessionStarted   Kind = "session.started" // session id, model, cwd known
	SessionEnded     Kind = "session.ended"
	TurnStarted      Kind = "turn.started"
	TurnCompleted    Kind = "turn.completed"
	Question         Kind = "question.requested"
	DiffUpdated      Kind = "diff.updated"
	Retry            Kind = "retry"
	Error            Kind = "error"
	StatusChanged    Kind = "status.changed"
	Sent             Kind = "sent"   // a control write of the dashboard's own (initialize, interrupt...)
	Stderr           Kind = "stderr" // one line of the CLI's stderr, kept for the error tail
	Unrecognised     Kind = "unrecognised"
)

// Origin says where a fact comes from (events-and-state.md, Origins).
type Origin string

// The five origins.
const (
	AI       Origin = "ai"
	Observed Origin = "observed"
	Derived  Origin = "derived"
	Inferred Origin = "inferred"
	Mock     Origin = "mock"
)

// Event is the normalised envelope. Seq and At are set by the collector, in
// stream order; adapters' mappers fill the rest.
type Event struct {
	Seq      uint64
	At       time.Time
	Provider string
	Session  string
	Kind     Kind
	Origin   Origin
	Agent    string // "" is the orchestrator
	Turn     string
	Item     string
	Payload  any
	Raw      string // the original line
}

// Spawned is the payload of SessionSpawned.
type Spawned struct {
	Argv []string
	Dir  string
}

// Ready is the payload of SessionReady.
type Ready struct {
	Detail string // what completed the handshake
}

// Started is the payload of SessionStarted.
type Started struct {
	SessionID      string
	Model          string
	Cwd            string
	PermissionMode string
	Version        string
	Plugins        []string
	Sandbox        string
	ApprovalPolicy string
}

// Ended is the payload of SessionEnded.
type Ended struct {
	Code    int
	HasCode bool
	Reason  string
}

// User is the payload of UserMessage: the text exactly as written.
type User struct {
	Text string
}

// Delta is the payload of MessageDelta. Block is "text", "thinking",
// "tool_input" or "signature"; only "text" belongs to the conversation.
type Delta struct {
	MessageID string
	Index     int
	Block     string
	Text      string
}

// Marker is the payload of MessageMarker.
type Marker struct {
	MessageID  string
	Name       string // message_start, content_block_start, content_block_stop, message_delta, message_stop, item_started
	Block      string
	Model      string
	StopReason string
}

// Completed is the payload of MessageCompleted.
type Completed struct {
	MessageID string
	Model     string
	Text      string // the text blocks, joined
	Thinking  bool   // a thinking block was present
}

// Tool is the payload of ToolStarted and ToolCompleted.
type Tool struct {
	ToolID  string
	Name    string
	Input   json.RawMessage
	IsError bool
	Status  string
	Output  string
}

// Agent is the payload of the agent events.
type Agent struct {
	Key          string
	Parent       string
	Label        string
	Type         string // subagent, workflow, thread
	Description  string
	Backgrounded bool
	Status       string // as reported
	Tokens       int64
	HasTokens    bool
	ToolUses     int
	LastTool     string
	Summary      string
}

// AgentKeys is the payload of AgentSet.
type AgentKeys struct {
	Keys []string
}

// Approval is the payload of ApprovalRequest and ApprovalResolved.
type Approval struct {
	ID       string
	Tool     string
	Display  string
	Summary  string
	ToolID   string
	Input    json.RawMessage
	Decision string // allow, deny (resolved only)
	Problem  string // why the request could not be read, when it could not
}

// Metric is one telemetry value. Documented is false for values from an
// undocumented field (shown with †).
type Metric struct {
	Name       string // quota.5h, quota.7d, tokens.total, cost.usd, context.used, context.window, model, duration.ms...
	Value      float64
	Text       string
	Unit       string
	ResetsAt   time.Time
	Documented bool
	Source     string
}

// Metrics is the payload of Telemetry.
type Metrics struct {
	Items []Metric
}

// TurnEnd is the payload of TurnCompleted. Outcome is completed, failed or
// interrupted.
type TurnEnd struct {
	Outcome    string
	Subtype    string
	DurationMS int64
	Error      string
}

// Diff is the payload of DiffUpdated.
type Diff struct {
	Path string
	Text string
}

// Plan is the payload of StageChanged: a Codex `turn/plan/updated`, the
// turn's plan as a whole (replace, not merge). Schema only, not recorded:
// v2 schema `TurnPlanUpdatedNotification`; no probe saw one (codex-cli.md,
// section 6).
type Plan struct {
	Explanation string
	Steps       []PlanStep
}

// PlanStep is one step of a Plan; Status is pending, inProgress or
// completed, as reported.
type PlanStep struct {
	Step   string
	Status string
}

// Problem is the payload of Error and Retry.
type Problem struct {
	Message   string
	WillRetry bool
}

// Status is the payload of StatusChanged.
type Status struct {
	Status string
	Detail string
}

// Write is the payload of Sent.
type Write struct {
	What      string
	RequestID string
}

// Line is the payload of Stderr.
type Line struct {
	Text string
}

// Unknown is the payload of Unrecognised.
type Unknown struct {
	Type   string
	Reason string
}
