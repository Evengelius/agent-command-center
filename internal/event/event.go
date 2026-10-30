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
	ApprovalMode     Kind = "approvals.mode"    // the approval mode in force (plan 12, item 2)
	ModesAllowed     Kind = "approvals.allowed" // the modes the CLI's policy leaves out (plan 39)
	ModelsListed     Kind = "models.listed"     // the models the CLI offers the session (plan 34)
	ModelSettings    Kind = "models.settings"   // the model, effort and Ultracode in force, or a change's answer (plan 34)
	ToolDecision     Kind = "tool.decision"     // who decided a tool call's permission (plan 12, item 2; undocumented for Claude Code)
	ReviewUpdated    Kind = "approvals.review"  // an auto-review's progress or verdict ([UNSTABLE] in Codex)
	ReviewWarning    Kind = "approvals.warning" // Codex's guardianWarning: kept in the event log only
	Sent             Kind = "sent"              // a control write of the dashboard's own (initialize, interrupt...)
	Stderr           Kind = "stderr"            // one line of the CLI's stderr, kept for the error tail
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
	// ApprovalMode is the mode in force the CLI reported: manual, auto, or
	// the CLI's own word for another; "" when it reported none (plan 12).
	ApprovalMode string `json:",omitempty"`
}

// Ended is the payload of SessionEnded.
type Ended struct {
	Code    int
	HasCode bool
	Reason  string
}

// User is the payload of UserMessage: the text exactly as written, and the
// images the message carried as image input (plan 31).
type User struct {
	Text   string
	Images []Image `json:",omitempty"` // left out of a mock file's line when none
}

// Image is an image a message carried: its file's name when the dashboard
// knows it ("" when the line gives none), its media type and its bytes as
// sent. The bytes themselves are never kept.
type Image struct {
	Name  string `json:",omitempty"`
	Media string `json:",omitempty"`
	Size  int    `json:",omitempty"`
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
	Name       string // message_start, content_block_start, content_block_stop, message_delta, message_stop, item_started, item_completed, step_started
	Block      string
	Model      string
	StopReason string
	// Index is a content block's index in its message, on Claude Code's
	// content_block_start and content_block_stop, so a block's stop finds
	// its start (plan 22).
	Index int `json:",omitempty"`
	// Thinking is the message's exact thinking-token count, a message_delta's
	// usage.output_tokens_details.thinking_tokens (Claude Code); HasThinking
	// is false when the line has none (plan 22, C3).
	Thinking    int64 `json:",omitempty"`
	HasThinking bool  `json:",omitempty"`
	// Summary is the summary of a Codex reasoning item, on its item_completed,
	// as Codex sent it: the item's parts joined by a blank line ("\n\n"); ""
	// when the item has none, as every item of a turn that did not ask for one
	// (plan 22, amendment 20; codex-cli.md, section 12).
	Summary string `json:",omitempty"`
}

// StepStarted names the marker of a model step's start: Codex reports a
// reasoning item only once its reasoning ended, so its adapter infers when
// each step began, from its items and its usage (plan 22, X5).
const StepStarted = "step_started"

// The thinking metrics (plan 22): Claude Code's live estimate of the open
// thinking block, cumulative (system/thinking_tokens, C2), and a Codex
// model step's reasoning tokens (thread/tokenUsage/updated
// last.reasoningOutputTokens, X2), by thread.
const (
	ThinkingEstimate = "thinking.estimate"
	ReasoningLast    = "reasoning.last"
)

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
	// ExitCode is a Codex command's exit code, as the CLI reports it (plan 6,
	// item 1); HasExitCode is false when it gave none.
	ExitCode    int  `json:",omitempty"`
	HasExitCode bool `json:",omitempty"`
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
	Model        string // the agent's own model as its CLI reports it; "" when not reported
	// Effort is the agent's reasoning effort, as Model; "" when not
	// reported (plan 6, item 6).
	Effort string `json:",omitempty"`
	// ModelRequested marks Model and Effort as the ones a spawn call asked
	// for, not the agent's own report: each fills only an empty value (plan
	// 6, items 4 and 6). It marks Label and Description the same way: a
	// spawn request's label fills only an agent without its own (plan 10,
	// item 1).
	ModelRequested bool `json:",omitempty"`
}

// AgentKeys is the payload of AgentSet.
type AgentKeys struct {
	Keys []string
}

// Decision is an approval's answer in the dashboard's words: the owner's
// four, and the CLI's withdrawal (plan 49, design D10; audit finding 11).
// Each adapter writes the owner's in its CLI's words, exactly as its CLI
// takes them, and reads its CLI's back into these; a word it does not
// know stays as it is, which the reducer keeps as an expiry.
type Decision string

const (
	Allow        Decision = "allow"         // once: Claude Code's allow, Codex's accept
	AllowSession Decision = "allow-session" // for the session: Codex's acceptForSession
	Deny         Decision = "deny"          // Claude Code's deny, Codex's decline
	Cancel       Decision = "cancel"        // deny, the turn interrupted after: Claude Code's deny, Codex's cancel
	Withdrawn    Decision = "withdrawn"     // the CLI ended the request itself
)

// Approval is the payload of ApprovalRequest and ApprovalResolved.
type Approval struct {
	ID         string
	Tool       string
	Display    string
	Summary    string
	ToolID     string
	Input      json.RawMessage
	Decision   Decision // the answer in the dashboard's words (resolved only; plan 49, design D10)
	Problem    string   // why the request could not be read, when it could not
	Offered    []string // the decisions the request lists (Codex availableDecisions, strings only); nil when it lists none
	ForSession bool     // the request takes an allow for the session (Codex acceptForSession; plan 8)
}

// Metric is one telemetry value. Documented is false for values from an
// undocumented field (shown with †). A non-empty Reason makes the value N/A
// for that reason (reliability.md, Fallback: N/A and its reason); Derived
// marks a value computed from other fields rather than read.
type Metric struct {
	Name       string // quota.5h, quota.7d, tokens.total, cost.usd, context.used, context.window, model, duration.ms...
	Value      float64
	Text       string
	Unit       string
	ResetsAt   time.Time
	Documented bool
	Source     string
	Reason     string `json:",omitempty"`
	Derived    bool   `json:",omitempty"`
}

// LimitMetric names the metric that carries Codex's concurrency limit: a
// positive Value, Text LimitNotSet when config/read reports none, or Text
// LimitPerBackend when Codex's two agent backends have different limits.
const (
	LimitMetric     = "agents.limit"
	LimitNotSet     = "not set"
	LimitPerBackend = "set per backend"
)

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

// Problem is the payload of Error and Retry. TurnRefused marks the CLI's
// refusal of the owner's message: no turn started, so it goes back to the
// editor (plan 34).
type Problem struct {
	Message     string
	WillRetry   bool
	TurnRefused bool `json:",omitempty"`
	// Optional says the refused request is one the session works without
	// (plan 46, design D2): its Error never ends the handshake.
	Optional bool `json:",omitempty"`
	// Fatal says the adapter recovered a panic (plan 48, design D5): the
	// session ends, its CLI closed, the message as the reason.
	Fatal bool `json:",omitempty"`
	// NoSession says the CLI has no such session to resume (plan 69, design
	// D7): the resume from the home screen is refused for good.
	NoSession bool `json:",omitempty"`
}

// The approval modes the adapters name (plan 12, item 2; plan 39): Manual
// (Claude Code's default, Codex's Ask for approval), Auto (Claude Code's
// auto, Codex's Approve for me), Claude Code's acceptEdits and plan, Codex's
// Full access, and Custom for a Codex combination outside its three; any
// other word of a CLI is carried as given.
const (
	ModeManual      = "manual"
	ModeAuto        = "auto"
	ModeAcceptEdits = "acceptEdits"
	ModePlan        = "plan"
	ModeFull        = "full"
	ModeCustom      = "custom"
)

// Mode is the payload of ApprovalMode: the mode in force, manual, auto, or
// the CLI's own word for another.
type Mode struct {
	Mode string
	// Answers marks the response to a request for a mode: the request is
	// settled whatever the mode. Refused marks a refusal, with its Text for
	// zone B; the mode in force is unchanged.
	Answers bool   `json:",omitempty"`
	Refused bool   `json:",omitempty"`
	Text    string `json:",omitempty"`
	// Sandbox is the sandbox that comes with the mode, when the dashboard
	// names one per turn (Codex, plan 12 addendum); empty leaves it as is.
	Sandbox string `json:",omitempty"`
	// NextTurn says the CLI takes the mode asked for with the next message,
	// not at once, its Text for zone B: a Codex that refuses
	// thread/settings/update (plan 39). It answers nothing: the change waits.
	NextTurn bool `json:",omitempty"`
}

// ModeLimits is the payload of ModesAllowed (plan 39): the session's modes
// its CLI does not allow, each with the CLI's reason (Codex's managed
// requirements); none listed, every mode of the catalog is allowed.
type ModeLimits struct {
	Unavailable map[string]string `json:",omitempty"`
	Source      string
}

// Models is the payload of ModelsListed (plan 34): the models the CLI offers
// the session, in its order, and the request they come from; Problem says
// why the list is cut short or missing, in the CLI's words.
type Models struct {
	Entries []ModelEntry
	Source  string // "initialize" (Claude Code) or "model/list" (Codex)
	Problem string `json:",omitempty"`
}

// ModelEntry is one model the CLI offers: what selects it (Claude Code's
// value, Codex's id), the model it runs (Claude Code's resolvedModel,
// Codex's model), its name and description as the CLI gives them, its
// effort levels in the CLI's order (none when it takes none), the level the
// CLI names its default and whether the CLI marks it the default model
// (both Codex's only).
type ModelEntry struct {
	ID            string
	Model         string
	Name          string
	Description   string
	Efforts       []string `json:",omitempty"`
	DefaultEffort string   `json:",omitempty"`
	Default       bool     `json:",omitempty"`
}

// Settings is the payload of ModelSettings (plan 34): the model settings the
// CLI reports in force, each as far as the read gives it, or the answer to a
// change the dashboard asked for.
type Settings struct {
	// Model is the model id in force; Choice the ModelEntry.ID that selects
	// it, "" when no entry does or the read cannot tell.
	Model  string `json:",omitempty"`
	Choice string `json:",omitempty"`
	// Effort is the level in force when EffortKnown; "" then means the CLI
	// sends none. EffortLock names what sets it outside the dashboard (the
	// CLAUDE_CODE_EFFORT_LEVEL variable), which the dashboard does not change.
	Effort      string `json:",omitempty"`
	EffortKnown bool   `json:",omitempty"`
	EffortLock  string `json:",omitempty"`
	// Ultracode in force and whether it can be turned on, when
	// UltracodeKnown (Claude Code only).
	Ultracode          bool `json:",omitempty"`
	UltracodeAvailable bool `json:",omitempty"`
	UltracodeKnown     bool `json:",omitempty"`
	// Source is the read the values come from; Undocumented marks one the
	// CLI's documentation does not list (Claude Code's get_settings).
	Source       string `json:",omitempty"`
	Undocumented bool   `json:",omitempty"`
	// Answers marks the answer to a change asked for: the request is settled
	// whatever it gives. Refused marks a refusal, its Text for zone B; the
	// settings in force are unchanged.
	Answers bool   `json:",omitempty"`
	Refused bool   `json:",omitempty"`
	Text    string `json:",omitempty"`
	// NextTurn says the CLI takes the change asked for, and every later one,
	// with the next message, not at once, its Text for zone B: a Codex that
	// refuses thread/settings/update (plan 38, design D4). It answers
	// nothing: the change still waits.
	NextTurn bool `json:",omitempty"`
}

// Verdict is the payload of ToolDecision: the permission decision the CLI
// recorded for a tool call (Claude Code tool_result_meta, undocumented).
// Decision is Allow or Deny in the dashboard's words, as the adapter reads
// its CLI's accept and reject (plan 49, design D10); ReasonType names who
// decided.
type Verdict struct {
	ToolID     string
	Decision   Decision
	ReasonType string
}

// Review is the payload of ReviewUpdated: one auto-review, started then
// completed (Codex item/autoApprovalReview/*, [UNSTABLE]). Status is as
// reported: inProgress, approved, denied, timedOut, aborted.
type Review struct {
	ReviewID      string
	Command       string
	Status        string
	Risk          string
	Authorization string
	Rationale     string
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
