// Package state is the dashboard's state and its reducer (doc/events-and-state.md,
// State model and State machines). The reducer is pure: it reads no file,
// starts no process, opens no connection and never reads the clock; the time
// of every step comes from the event or the action. The same inputs always
// give the same state.
package state

import (
	"encoding/json"
	"time"

	"github.com/Evengelius/agent-command-center/internal/event"
)

// Screen is the screen shown.
type Screen string

// The two screens.
const (
	ScreenStartup   Screen = "startup"
	ScreenDashboard Screen = "dashboard"
)

// StartupPhase is the startup machine's state.
type StartupPhase string

// Startup machine states; Exited is the application quitting.
const (
	Detecting   StartupPhase = "Detecting"
	Asking      StartupPhase = "Asking"
	Validating  StartupPhase = "Validating"
	Starting    StartupPhase = "Starting"
	StartReady  StartupPhase = "Ready"
	StartFailed StartupPhase = "StartFailed"
	Exited      StartupPhase = "Exited"
)

// SessionPhase is the session machine's state (the CLI process). "" is no
// session yet.
type SessionPhase string

// Session machine states.
const (
	Spawning    SessionPhase = "Spawning"
	Handshaking SessionPhase = "Handshaking"
	Ready       SessionPhase = "Ready"
	Closing     SessionPhase = "Closing"
	Closed      SessionPhase = "Closed"
	Crashed     SessionPhase = "Crashed"
)

// TurnPhase is the turn machine's state.
type TurnPhase string

// Turn machine states.
const (
	Idle            TurnPhase = "Idle"
	Submitted       TurnPhase = "Submitted"
	Running         TurnPhase = "Running"
	WaitingApproval TurnPhase = "WaitingApproval"
	Interrupting    TurnPhase = "Interrupting"
)

// AgentStatus is the agent machine's state.
type AgentStatus string

// Agent machine states.
const (
	AgentAnnounced AgentStatus = "Announced"
	AgentRunning   AgentStatus = "Running"
	AgentWaiting   AgentStatus = "WaitingApproval"
	AgentCompleted AgentStatus = "Completed"
	AgentFailed    AgentStatus = "Failed"
	AgentStopped   AgentStatus = "Stopped"
	AgentEnded     AgentStatus = "Ended"
)

// Final reports whether the agent has ended.
func (a AgentStatus) Final() bool {
	return a == AgentCompleted || a == AgentFailed || a == AgentStopped || a == AgentEnded
}

// ApprovalStatus is the approval machine's state.
type ApprovalStatus string

// Approval machine states.
const (
	Pending   ApprovalStatus = "Pending"
	Answering ApprovalStatus = "Answering"
	Allowed   ApprovalStatus = "Allowed"
	Denied    ApprovalStatus = "Denied"
	Expired   ApprovalStatus = "Expired"
)

// Final reports whether the approval is settled.
func (a ApprovalStatus) Final() bool { return a == Allowed || a == Denied || a == Expired }

// ToolStatus is the tool call machine's state.
type ToolStatus string

// Tool call machine states.
const (
	ToolStarted    ToolStatus = "Started"
	ToolRunning    ToolStatus = "Running"
	ToolCompleted  ToolStatus = "Completed"
	ToolFailed     ToolStatus = "Failed"
	ToolDenied     ToolStatus = "Denied"
	ToolUnfinished ToolStatus = "Unfinished"
)

// Final reports whether the call has ended.
func (t ToolStatus) Final() bool { return t != ToolStarted && t != ToolRunning }

// Context is the session context the startup screen builds; immutable for
// the session (architecture.md, Session context).
type Context struct {
	Provider     string // provider id
	ProviderName string
	Project      string // the validated project path
	Branch       string // its git branch, "" outside a work tree (display only)
	Mock         bool
}

// CLI is one agent CLI as detection found it.
type CLI struct {
	ID, Name, Version, Path string
	Installed               bool
	Problem                 string // why it is not installed or not usable
}

// StartupState is the startup machine with its answers.
type StartupState struct {
	Phase     StartupPhase
	Installed []string // provider ids detection found
	CLIs      []CLI
	Provider  string // the answers, kept when going back
	Project   string
	Field     string // the field a validation message belongs to
	Message   string
	Failure   string // why the start failed
}

// SessionState is the session machine with what the CLI reported.
type SessionState struct {
	Phase          SessionPhase
	ID             string
	Model          string
	Cwd            string
	PermissionMode string
	Version        string
	Sandbox        string
	ApprovalPolicy string
	Plugins        []string
	Status         string
	ExitCode       int
	HasExitCode    bool
	Reason         string   // why it ended or crashed
	StderrTail     []string // the last lines of stderr
	Notice         string   // the last refusal shown to the owner
}

// TurnState is the turn machine.
type TurnState struct {
	Phase            TurnPhase
	Outcome          string // completed, failed or interrupted: the last turn's
	Count            int    // turns submitted
	ByCLI            bool   // the CLI started the current or last turn itself
	Draft            string // the message written and not yet accepted
	ForceStopOffered bool
	Notice           string
	SlashHeld        string // a Codex slash command held back once: the same text again is sent
}

// Entry is one item of a conversation record.
type Entry struct {
	Seq       uint64
	Role      string // user, assistant, tool, notice, approval, error, dashboard
	Agent     string
	MessageID string
	Text      string
	Origin    event.Origin
	Streaming bool
	Thinking  bool
	At        time.Time
}

// Agent is one node of the agent tree.
type Agent struct {
	Key          string
	Parent       string
	Label        string
	Kind         string // subagent, workflow, thread
	Description  string
	Model        Value
	Status       AgentStatus
	StatusOrigin event.Origin
	Backgrounded bool
	InSet        bool // seen in a background task set
	LastTool     string
	ToolUses     int
	Tokens       Value
	Tools        []string
	Messages     []Entry
	History      []string // events applied after the agent ended, and other notes
	Summary      string
	StartedAt    time.Time
	EndedAt      time.Time
	Runs         int
	Retries      int
	Error        string
	Stage        int    // the last stage when it appeared, 1-based; 0 before any turn
	Seq          uint64 // the event that created it, for ordering in zone A
}

// Approval is one approval or question.
type Approval struct {
	ID          string
	Agent       string
	Tool        string
	Display     string
	Summary     string
	ToolID      string
	Input       json.RawMessage
	Question    bool
	Problem     string
	Raw         string
	Status      ApprovalStatus
	Answer      string
	RequestedAt time.Time
	AnsweredAt  time.Time
}

// ToolCall is one tool call.
type ToolCall struct {
	ID        string
	Agent     string
	Name      string
	Input     json.RawMessage
	Status    ToolStatus
	Origin    event.Origin
	Output    string
	History   []string
	StartedAt time.Time
	EndedAt   time.Time
	Stage     int    // the last stage when it appeared, 1-based; 0 before any turn
	Seq       uint64 // the event that created it
}

// Stage is one user turn as zone A shows it: an inferred stage whose
// children are the turn's tools and agents, or, with a Codex plan, the
// plan's steps, observed (events-and-state.md, Agent tree).
type Stage struct {
	N         int  // 1-based, in order; a turn the CLI opened counts too
	ByCLI     bool // the CLI opened it (a background agent's report)
	Open      bool
	Outcome   string // completed, failed or interrupted once closed
	StartedAt time.Time
	EndedAt   time.Time
	Plan      []event.PlanStep // the last turn/plan/updated of the turn, observed
	Prompt    string           // the owner's message that opened it; "" when the CLI opened it
}

// Value is one telemetry value. Present false renders "N/A" with Reason.
type Value struct {
	Present    bool
	V          float64
	Text       string
	Unit       string
	Origin     event.Origin
	Source     string
	Documented bool
	ReceivedAt time.Time
	ResetsAt   time.Time
	Reason     string
	ReasonAt   time.Time // when the source said N/A
	Last       bool      // N/A, and V, Text and ReceivedAt keep the last value
}

// ErrorEntry is one error shown in zone D.
type ErrorEntry struct {
	Seq     uint64
	At      time.Time
	Agent   string
	Message string
	Retry   bool
}

// Bounds of the lists kept in memory.
const (
	MaxErrors       = 50
	MaxLog          = 2000
	MaxUnrecognised = 20
	MaxStderr       = 20
)

// State is the whole dashboard state.
type State struct {
	Screen           Screen
	Startup          StartupState
	Context          Context
	Session          SessionState
	Turn             TurnState
	Stages           []Stage // one per turn, in order
	Conversation     []Entry
	Agents           map[string]*Agent
	Order            []string // agents in the order they appeared
	Ambient          map[string]bool
	Approvals        map[string]*Approval
	Queue            []string // approval ids in the order they arrived
	Tools            map[string]*ToolCall
	Metrics          map[string]Value
	Diffs            map[string]string
	Errors           []ErrorEntry
	Retries          int
	Log              []event.Event // the last MaxLog events, for the inspection view
	Unrecognised     int
	UnrecognisedLast []string
	Notes            []string // anomalies logged by the reducer (invalid transitions)
	QuitAsked        bool     // Ctrl+Q asked to confirm; a second one quits, Esc stays
}

// New returns the state before launch.
func New() *State {
	return &State{
		Screen:    ScreenStartup,
		Turn:      TurnState{Phase: Idle},
		Agents:    map[string]*Agent{},
		Ambient:   map[string]bool{},
		Approvals: map[string]*Approval{},
		Tools:     map[string]*ToolCall{},
		Metrics:   map[string]Value{},
		Diffs:     map[string]string{},
	}
}

// CLI is the agent CLI the session's events come from: the provider id, or in
// mock mode the provider the replayed events name; "" before any.
func (s *State) CLI() string {
	if !s.Context.Mock {
		return s.Context.Provider
	}
	for _, ev := range s.Log {
		if ev.Provider != "" {
			return ev.Provider
		}
	}
	return ""
}

// CommandKind is an effect the reducer asks the application to carry out.
type CommandKind string

// The commands.
const (
	CmdValidate      CommandKind = "validate"       // validate Provider and Project
	CmdStartSession  CommandKind = "start-session"  // spawn the CLI
	CmdWrite         CommandKind = "write"          // send Text as a user message
	CmdAnswer        CommandKind = "answer"         // answer approval ID with Decision
	CmdInterrupt     CommandKind = "interrupt"      // interrupt the turn
	CmdRefresh       CommandKind = "refresh"        // the reads after a turn end (events-and-state.md, Polling)
	CmdClose         CommandKind = "close"          // close the session
	CmdKill          CommandKind = "kill"           // force the process tree down
	CmdDraftAccepted CommandKind = "draft-accepted" // the CLI accepted the message: delete its draft
	CmdRestoreDraft  CommandKind = "restore-draft"  // put Text back in the editor
	CmdExit          CommandKind = "exit"           // leave the application
)

// Command is one effect.
type Command struct {
	Kind     CommandKind
	ID       string
	Text     string
	Decision string
}
