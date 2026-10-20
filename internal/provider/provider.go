// Package provider is the interface every agent CLI adapter implements
// (doc/architecture.md, Provider interface). Nothing outside an adapter names
// a CLI; the application talks to this interface only.
package provider

import (
	"context"
	"time"

	"github.com/Evengelius/agent-command-center/internal/event"
)

// Detection is what detecting one CLI found.
type Detection struct {
	ID          string // provider id
	Name        string // display name
	Installed   bool
	Path        string // the executable
	Version     string
	Tested      string // the last version probed (doc/evaluation.md, After a CLI update)
	Problem     string // why it is not usable, when it is not
	ElapsedTime time.Duration
}

// SessionContext is built once by the startup screen and is immutable for the
// session (architecture.md, Session context).
type SessionContext struct {
	Provider    string
	Name        string
	Version     string
	Executable  string
	ProjectName string
	ProjectPath string
	Branch      string
	Mock        bool
	StartedAt   time.Time
	ResumeID    string // a CLI session id to resume, "" for a new session
}

// Capabilities are what one CLI and version support; the screen shows "N/A"
// or hides an action when a flag is off.
type Capabilities struct {
	SubagentText     bool
	SubagentModel    bool
	QuotaWithoutTurn bool
	APICost          bool
	Questions        bool
	PlanSteps        bool
	ResumeHistory    bool
	AllowForSession  bool
	AutoApprovals    bool // the session can switch between Manual and Auto approvals (plan 12)
}

// Provider is one agent CLI.
type Provider interface {
	ID() string
	DisplayName() string
	Detect(ctx context.Context) Detection
	Start(ctx context.Context, sc SessionContext) (Session, error)
}

// Session is one running CLI session. Events is closed after the session's
// last event (session.ended).
type Session interface {
	Events() <-chan event.Event
	Send(text string) error
	Answer(approvalID, decision string) error
	// AnswerQuestion answers a pending question request with the answer to each
	// question, by its text (plan 14, item 1); Capabilities.Questions says whether
	// the provider takes it.
	AnswerQuestion(approvalID string, answers map[string]string) error
	Interrupt() error
	Refresh() error
	// SetApprovalMode asks for Auto (true) or Manual (false) approvals; the mode
	// in force is reported by events (plan 12, item 2).
	SetApprovalMode(auto bool) error
	Capabilities() Capabilities
	Close(ctx context.Context) error
}
