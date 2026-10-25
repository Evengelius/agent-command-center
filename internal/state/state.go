// Package state is the dashboard's state and its reducer (doc/events-and-state.md,
// State model and State machines). The reducer is pure: it reads no file,
// starts no process, opens no connection and never reads the clock; the time
// of every step comes from the event or the action. The same inputs always
// give the same state.
package state

import (
	"encoding/json"
	"regexp"
	"strconv"
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
	Tested                  string // the last version probed, "" when unknown
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
	// ApprovalMode is the mode in force: ModeManual, ModeAuto or the CLI's own
	// word; "" before the CLI reports, which reads as Manual, the mode the
	// dashboard starts with. ApprovalRequested is a change asked for and not
	// yet in force, "" when none (plan 12, item 2); ModeNextTurn says it
	// goes with the next message (Codex during a turn, or without
	// thread/settings/update; plan 39).
	ApprovalMode      string
	ApprovalRequested string
	ModeNextTurn      bool
	// ModesUnavailable are the session's modes its CLI's policy leaves out,
	// with why (Codex's requirements; plan 39).
	ModesUnavailable map[string]string
}

// Attachment is an image attached to the owner's message (plan 31): its
// file's path, which the application reads when it sends, its name, its
// size and media type as sent, what the sending changes ("converted from
// BMP"), and the SHA-256 of the file's bytes when it was attached, to tell
// a file changed since (plan 33, design D7). The image's bytes are never in
// the state.
type Attachment struct {
	Path  string
	Name  string
	Size  int64
	Media string
	Note  string
	Sum   [32]byte
}

// pending says a sent message waits to be accepted: its text or its images.
func (t TurnState) pending() bool { return t.Draft != "" || len(t.DraftImages) > 0 }

// TurnState is the turn machine.
type TurnState struct {
	Phase   TurnPhase
	Outcome string // completed, failed or interrupted: the last turn's
	Count   int    // turns submitted
	ByCLI   bool   // the CLI started the current or last turn itself
	Draft   string // the message written and not yet accepted
	// DraftImages are its attached images, until it is accepted (plan 31):
	// a failed write gives them back to the editor with the text.
	DraftImages      []Attachment
	ForceStopOffered bool
	Notice           string
	SlashHeld        string // a Codex slash command held back once: the same text again is sent
}

// Entry is one item of a conversation record.
type Entry struct {
	Seq       uint64
	Role      string // user, assistant, tool, notice, approval, error, dashboard, thinking
	Agent     string
	MessageID string
	Text      string
	Origin    event.Origin
	Streaming bool // still coming; for a thinking entry, the phase is live
	Thinking  bool
	At        time.Time
	// Status is a review entry's status as reported (inProgress, approved,
	// denied, timedOut, aborted); "" for any other entry (plan 12, item 2).
	Status string
	// Stage is the turn the entry belongs to, Stages[Stage-1]: the one open
	// when it came, for a prompt the one it opens; after a turn ended, that
	// turn; 0 before any (plan 19, zone B's RESULT).
	Stage int
	// Thought is a thinking entry's phase; nil for any other role (plan 22).
	Thought *Thought
	// Images are the images a user entry's message carried as image input,
	// named from the attachments when the dashboard sent them (plan 31).
	Images []event.Image
}

// Thought is one thinking phase of the orchestrator, the entry of role
// "thinking" that zone B draws as its live card, then as its summary row
// (plan 22, brief amendment 18). Claude Code reports a phase's start and
// end, a thinking block's; Codex only that its model step reasoned, once
// the reasoning ended: the step's start stands for the phase's, inferred,
// until a reasoning item confirms it (plan.md, plan 22, C1 and X1).
type Thought struct {
	// ID numbers the phases of the conversation record from 1, kept across a
	// resume (State.Thoughts), where an event's Seq starts again: zone B's
	// keys for a phase, its details open and drawn live, are its ID.
	ID    int
	Block int       // the thinking block's index in its message (Claude Code)
	Item  string    // the reasoning item's id (Codex)
	Start time.Time // the block's start; a Codex step's start, inferred; zero when unknown
	End   time.Time // when it ended; zero while it runs
	// EndInferred marks an End the CLI did not report as the phase's own (its
	// block's stop, its reasoning item's completion): the end of its
	// message, of its turn, or of the session.
	EndInferred bool
	// EndNext marks an inferred End that is the start of the next Codex step: a
	// reasoning item whose completion never came is over there, and its details
	// say so, not "when its turn ended" (plan 22, correction C4).
	EndNext bool
	// ItemStart is when the Codex reasoning item of the step started (its
	// item/started, observed), zero while none did; Claude Code's phases have
	// none. A turn that asks for reasoning summaries starts the item while the
	// model works (P22s: 2.5 and 3.4 s before it completes); without them the
	// item starts and completes within 10 ms. Start stays the step's, inferred,
	// and the phase's time stays "up to" (plan 22, amendment 20).
	ItemStart time.Time
	// Summary is the item's summary, Codex's own text, as its completion
	// carried it: the parts joined by a blank line; "" when Codex sent none
	// (plan 22, amendment 20).
	Summary string
	// Inferred marks a Codex model step: Start is the step's, and it is a
	// phase of thinking only once Confirmed by a reasoning item; Dropped
	// when the step ended with no reasoning, which leaves no row.
	Inferred, Confirmed, Dropped bool
	Outcome                      string // "" ended normally; "interrupted" or "failed"
	// Estimate is the CLI's live estimate, cumulative, never lowered (Claude
	// Code's system/thinking_tokens, C2); 0 before the first. EstimateFrom is
	// the value before the last change, EstimateAt the change's time: the bar
	// grows from one to the other.
	Estimate, EstimateFrom int64
	EstimateAt             time.Time
	// Tokens is the exact count once reported (HasTokens), at TokensAt, from
	// Source. Shared > 1 when it is the count of a message or a step with
	// that many phases, set on its last phase only. Counted marks a Codex
	// thought a usage update was applied to, so the next update goes to the
	// next thoughts only.
	Tokens    int64
	HasTokens bool
	TokensAt  time.Time
	Source    string
	Shared    int
	Counted   bool
	// Closed marks a phase whose turn or session ended with no exact count
	// reported: none can come any more, so no later count (a resume's replay of
	// the last usage, the next turn's update, a late message_delta) is applied
	// to it, and its details say so (plan 22, correction C2).
	Closed bool
	// Cut marks a Claude Code phase whose message was cut short (stop_reason
	// null: an interrupt, P303a and P2). The CLI's message_delta then carries a
	// thinking_tokens of 0, the usage the message had when it began, not a
	// measure; CutTokens keeps that value as reported. The phase has no exact
	// count (HasTokens stays false), so nothing that reads one shows the 0 as
	// a count, the exit's count row included (plan 22, correction C1).
	Cut       bool
	CutTokens int64
	// Budget is a thinking-token budget the CLI reported; 0 for none. No
	// supported CLI reports one (C8, X3): mode A of the bar is tested only.
	Budget int64
}

// Live reports whether the phase still runs.
func (t *Thought) Live() bool { return t != nil && t.End.IsZero() }

// Agent is one node of the agent tree.
type Agent struct {
	Key          string
	Parent       string
	Label        string
	Kind         string // subagent, workflow, thread
	Description  string
	Model        Value
	Effort       Value // the reasoning effort (plan 6, item 6)
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
	EndStage     int    // the stage when it last ended; 0 while it runs (plan 18)
	Seq          uint64 // the event that created it, for ordering in zone A
	// LabelRequested marks Label as a spawn request's, which the agent's own
	// label replaces (plan 10, item 1).
	LabelRequested bool
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
	ForSession  bool // the request takes acceptForSession (Codex; plan 5, item 8; plan 8)
	Problem     string
	Raw         string
	Status      ApprovalStatus
	Answer      event.Decision
	Answers     map[string]string // a question answered: question text to answer, set when sent
	Cancelled   bool              // answered by a cancel (c, Ctrl+C, quitting), not by d (plan 5, item 10)
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
	OutputCut int64 // bytes of a running call's output not kept (plan 48, design D2)
	History   []string
	StartedAt time.Time
	EndedAt   time.Time
	Stage     int    // the last stage when it appeared, 1-based; 0 before any turn
	Seq       uint64 // the event that created it

	// ExitCode is a command's exit code from the event (plan 6, item 1).
	ExitCode    int
	HasExitCode bool

	// Reviewed is who decided the call's permission when the CLI says an
	// auto-review did: "approved by auto-review" or "denied by auto-review";
	// "" otherwise (plan 12, item 2).
	Reviewed string
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

// UnknownEntry is one line the mapper could not map (plan 25, D1): when,
// what the mapper called it (event.Unknown.Type), why when it said, and
// the line itself. The details view lists the last MaxUnrecognised.
type UnknownEntry struct {
	At     time.Time
	Type   string
	Reason string
	Line   string
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
	Screen       Screen
	Startup      StartupState
	Context      Context
	Session      SessionState
	Models       Models // the session's models and their settings (plan 34)
	Turn         TurnState
	Stages       []Stage // one per turn, in order
	Conversation []Entry
	Agents       map[string]*Agent
	Order        []string // agents in the order they appeared
	Ambient      map[string]bool
	Approvals    map[string]*Approval
	Queue        []string // approval ids in the order they arrived
	Tools        map[string]*ToolCall
	// ToolOrder is every tool call's ID in the order it was added, and
	// ParentRev counts the times an agent's parent changed: render's memo
	// takes in the calls added since its last frame, and builds its index
	// of the turns' items again when an agent was added or its parent
	// changed (plan 49, design D5).
	ToolOrder        []string
	ParentRev        int
	Metrics          map[string]Value
	Diffs            map[string]string
	Errors           []ErrorEntry
	Retries          int
	Log              []event.Event // the last MaxLog events, for the inspection view
	Thoughts         int           // the thinking phases opened: the last Thought.ID (plan 22)
	Applied          int           // events applied; the first one's time is FirstAt
	FirstAt          time.Time     // the session time counts from it, kept when Log drops events and on a resume (plan 21, item 1)
	Unrecognised     int
	UnrecognisedLast []UnknownEntry
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

// InPlay lists, in their order of creation, the agents zone C shows and is
// sized for (plan 18, brief amendment 14): those that run, or appeared or
// ended in the stage of the owner's last prompt or after it, so that each new
// prompt clears the others, which zone A's tree keeps. A stage the CLI opens
// by itself, such as Claude Code's for a background agent's notification, is
// not a prompt. Before any prompt, every agent is in play.
func (s *State) InPlay() []string {
	from := 0
	for _, st := range s.Stages {
		if !st.ByCLI {
			from = st.N
		}
	}
	var keys []string
	for _, k := range s.Order {
		if a := s.Agents[k]; a != nil && (!a.Status.Final() || a.Stage >= from || a.EndStage >= from) {
			keys = append(keys, k)
		}
	}
	return keys
}

// AgentsInPlay counts the agents in play; with none, zone C is an empty strip.
func (s *State) AgentsInPlay() int { return len(s.InPlay()) }

// NewerThanTested reports whether the session's CLI is newer than the last
// version its adapter was probed with (doc/evaluation.md, After a CLI
// update). It is false in mock mode, and when either version is unknown or
// unparsable.
func (s *State) NewerThanTested() bool {
	if s.Context.Mock {
		return false
	}
	for _, c := range s.Startup.CLIs {
		if c.ID == s.Context.Provider {
			return newerVersion(s.Session.Version, c.Tested)
		}
	}
	return false
}

// versionRe reads the leading major.minor.patch of a CLI version.
var versionRe = regexp.MustCompile(`^\s*v?([0-9]+)\.([0-9]+)\.([0-9]+)`)

// newerVersion compares the leading major.minor.patch of v and tested; a
// suffix after them is ignored.
func newerVersion(v, tested string) bool {
	a, b := versionRe.FindStringSubmatch(v), versionRe.FindStringSubmatch(tested)
	if a == nil || b == nil {
		return false
	}
	for i := 1; i <= 3; i++ {
		x, errX := strconv.Atoi(a[i])
		y, errY := strconv.Atoi(b[i])
		if errX != nil || errY != nil {
			return false
		}
		if x != y {
			return x > y
		}
	}
	return false
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
	CmdSetApprovals  CommandKind = "set-approvals"  // ask for Mode, one of the session's CLI's modes (plan 39)
	CmdModels        CommandKind = "models"         // ask for the model settings of Change (plan 34)
)

// The approval modes (plan 12, item 2; plan 39, modes.go).
const (
	ModeManual = event.ModeManual
	ModeAuto   = event.ModeAuto
)

// Command is one effect.
type Command struct {
	Kind     CommandKind
	ID       string
	Text     string
	Decision event.Decision    // CmdAnswer: the owner's, in the dashboard's words (plan 49, design D10)
	Mode     string            // CmdSetApprovals: the mode asked for (plan 39)
	Answers  map[string]string // CmdAnswer of a question: question text to answer
	Images   []Attachment      // CmdWrite and CmdRestoreDraft: the message's images (plan 31)
	Change   ModelChange       // CmdModels: the model settings asked for (plan 34)
}

// Mode is the approval mode in force; Manual until the CLI reports another.
func (s *State) Mode() string {
	if s.Session.ApprovalMode == "" {
		return ModeManual
	}
	return s.Session.ApprovalMode
}
