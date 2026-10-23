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

// Message is the owner's message: the text as typed and the images attached
// to it, read, checked and converted by the application (plan 31, brief
// amendment 31). An adapter sends each image as image input in the CLI's
// supported interface, never as a path in the text
// (doc/research/image-input.md).
type Message struct {
	Text   string
	Images []Image
}

// Image is one attached image as sent: its file's name, the media type the
// models take (image/png, image/jpeg, image/gif, image/webp) and its bytes.
// Adapters keep its bytes out of events, logs and the screen.
type Image struct {
	Name  string
	Media string
	Data  []byte
}

// ImageLimits are a CLI's limits on the images of one message
// (doc/research/image-input.md); a zero field is no limit, except MaxCount:
// 0 means the CLI takes no image.
type ImageLimits struct {
	MaxCount   int   // images a message
	MaxBytes   int64 // an image as sent
	MaxTotal   int64 // all the images of a message as sent
	MaxSide    int   // pixels a side
	MaxPatches int   // patches of PatchSize px an image
	PatchSize  int
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
	// Images are the CLI's limits on a message's images (plan 31); a zero
	// MaxCount is a CLI that takes none.
	Images ImageLimits
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
	// Send writes the owner's message, its text byte for byte and its images
	// as image input (plan 31).
	Send(msg Message) error
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
