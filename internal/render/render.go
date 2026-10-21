// Package render draws the state as a frame of text (doc/ui-design.md). It is
// a pure function of the state and the interface's own state: it imports no
// adapter and no process package, reads no clock and changes nothing.
package render

import (
	"fmt"
	"strings"
	"time"

	"charm.land/lipgloss/v2"

	"github.com/Evengelius/agent-command-center/internal/state"
)

// MinWidth and MinHeight are the smallest terminal the screens fit.
const (
	MinWidth  = 80
	MinHeight = 24
)

// MockBadge replaces "● LIVE" on every frame of a mock run.
const MockBadge = "◆ MOCK — not a live session"

// UI is what the frame needs beyond the state: the terminal, the focus and
// the text being edited. The application owns it.
type UI struct {
	Width, Height int
	Now           time.Time // the frame's time, for elapsed times

	// Startup screen.
	Field       int    // 0 agent CLI, 1 project directory, 2 Start, 3 the project list
	Choice      string // the provider id the radio buttons select
	Path        string // the project field's text
	Resolved    string // the path the agent would receive, computed without I/O
	PathProblem string // why the typed form is refused, computed without I/O
	PathWarning string // a caution on an accepted path (UNC, WSL)
	Branch      string // the git branch, once read
	// The project list (plan 15): favourites first, then recent projects.
	Projects        []ProjectRow
	ProjectSel      int           // the selected row
	ProjectsProblem string        // the list cannot be read or written
	Browser         BrowserView   // the project browser, over the screen while Open
	Worktrees       WorktreesView // the Worktrees view, over either screen while Open (plan 16)

	// Dashboard.
	Focus    string // "A", "B", "C" or "D"
	Draft    string // the editor's text
	Editor   EditorView
	Selected string // the agent selected in zone C
	Card     string // the approval card with the focus: it takes a, d, c (ui-design.md, Approvals)
	Pick     Pick   // the owner's choices on the focused question card (plan 14, item 1)
	Spinner  int    // the spinner's frame
	// Live holds the thinking phases a frame drew while they ran, by their
	// Thought.ID: only they play their exit, so a phase that began and ended
	// in one batch of events (a replay, a resume) never looks live (plan 22).
	Live map[uint64]bool
	// ActivitySince is when the activity line's condition began (ActivityOn),
	// zero while it does not hold. The application notes it; the line is drawn
	// once Now is 200 ms past it, so a gap of a few frames between two cards, or
	// before a turn's end, never shows it. Zero draws no line (plan 22,
	// correction C6).
	ActivitySince time.Time
	Toast         string // replaces the key hints for a moment (ui-design.md, Clipboard)
	// The editor's send and newline keys in force, as the screen names them
	// ("Ctrl+Enter"), in the configuration file's order (plan 21, item 5.1);
	// none is the defaults. KeysProblem is why the file's keys were not used,
	// "" when there is no problem: the startup screen shows it, and a mock
	// run's editor status row until the first message is sent.
	SendKeys, NewlineKeys []string
	KeysProblem           string
	// CodexProxy is the proxy Codex is started through, config.json's
	// codex.proxy (plan 22, amendment 19), "" when there is none; the startup
	// screen shows it on a grey line. CodexProxyProblem is why the codex section
	// was not used, "" when there is none: the startup screen shows it, under
	// KeysProblem, as a red line.
	CodexProxy, CodexProxyProblem string

	// Zone A's tree and agent inspection (ui-design.md, Zone A; Agent
	// inspection).
	ACursor     string          // the node under zone A's cursor (Node.ID); "" none
	Fold        map[string]bool // the owner's folds by node; a node not in it has its default
	Inspect     string          // the agent inspected full screen; "" none
	Tab         int             // the inspection's tab: 0 Messages, 1 Tools, 2 Diff, 3 Raw events
	InspectTop  int             // the first inspection row shown while held
	InspectHold bool            // scrolled up: the inspection stays where it is
	Details     bool            // zone D full screen (Enter on D); it scrolls with InspectTop and InspectHold
	Zoom        bool            // the focused zone, A, C or D, over the dashboard (z)
	Keys        bool            // the key overlay (F1); it scrolls with InspectTop and InspectHold
	View        string          // the approval shown whole, full screen (Enter on its card); "" none; it scrolls with InspectTop and InspectHold
	Entry       int             // zone B's entry cursor, Conversation[Entry-1]; 0 none (ui-design.md, Zone B, Expanding)
	ViewEntry   int             // the entry shown whole, full screen (Enter on it), as Entry; 0 none; it scrolls with InspectTop and InspectHold
	Unfolded    map[string]bool // zone B's runs of tool calls the owner expanded, by their first call's id (plan 19)
	Tree        Pane            // zone A's tree as the owner scrolled it (plan 5, item 5)
	List        Pane            // zone C's agent list, likewise
	Preview     Pane            // zone C's preview, likewise

	// Mouse selection and scrolling (ui-design.md, Selection and copy; Zone B,
	// Follow mode).
	Sel      Selection // the mouse selection, outside the editor
	ConvTop  int       // the first conversation row shown while held
	ConvHold bool      // scrolled up: the view stays where it is
	ConvSeen int       // the entries there were when the view was held
	MouseOff bool      // mouse capture off: the terminal selects

	// File is the file viewer, Finder the quick open and Explorer the
	// project's tree, floating over the dashboard (plan 17, amendment 12).
	File     FileView
	Finder   FinderView
	Explorer ExplorerView
	// DraftWin is the draft in a window, Ctrl+G (amendment 13).
	DraftWin DraftWindow

	// Wrap keeps the conversation's line breaks between frames; nil wraps
	// every line each frame, with the same result.
	Wrap *WrapCache
}

// Pane is a pane of zone A or C the owner scrolled (ui-design.md,
// Scrolling, collapsing, full screen). Not held, it shows what it always
// did: A's tree and the preview their tail, C's list its top, each with its
// cursor in view.
type Pane struct {
	Top  int  // the first row shown while held
	Hold bool // scrolled: the pane stays at Top as rows come and go
}

// Symbols (ui-design.md, Colours and symbols); the colours are in palette.go.
var spinner = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// style is a shared style. Spans hold a pointer to one, never a copy: a
// lipgloss.Style is hundreds of bytes, and spans equal by value let a frame
// reuse the rows a keystroke did not change (Screen.StringAfter).
func style(s lipgloss.Style) *lipgloss.Style { return &s }

// modelStyle colours a model name by its family.
func modelStyle(model string) *lipgloss.Style {
	m := strings.ToLower(model)
	for i, kc := range modelColours {
		if strings.Contains(m, kc.family) {
			return modelStyles[i]
		}
	}
	return stText
}

// Frame draws the whole screen: exactly ui.Height lines, none wider than
// ui.Width.
func Frame(s *state.State, ui UI) string { return Draw(s, ui).String() }

func tooSmall(s *state.State, ui UI) []line {
	ls := []line{
		plain(fmt.Sprintf(" Terminal too small: %dx%d needed, now %dx%d", MinWidth, MinHeight, ui.Width, ui.Height), stFailed),
		plain(" Ctrl+Q quit", stGrey),
	}
	if s.Context.Mock {
		ls = append([]line{plain(" "+MockBadge, stProgress)}, ls...)
	}
	return ls
}

// titleBar is the first line of both screens; right is the badge.
func titleBar(s *state.State, w int, right line, extra line) line {
	left := line{sp(" >_ ", stTitle), sp("AGENT COMMAND CENTER", stTitle)}
	if extra != nil {
		left = append(append(left, sp("     ", stText)), extra...)
	}
	if s.Context.Mock {
		right = plain(MockBadge+" ", stProgress)
	}
	return spread(left, right, w)
}

// rule is a full-width thin line.
func rule(w int) line { return plain(strings.Repeat("─", w), stFrame) }

// elide shortens a path from the left with "…" to fit w cells, and hides the
// user folder (C:\Users\name\ becomes C:\Users\…\).
func elide(p string, w int) string {
	p = hideUser(p)
	if textWidth(p) <= w {
		return p
	}
	if w <= 1 {
		return "…"
	}
	rs := []rune(p)
	for len(rs) > 0 && (textWidth(string(rs))+1 > w || rs[0] == '…') {
		rs = rs[1:] // never "……" where the cut meets the hidden user folder
	}
	return "…" + string(rs)
}

func hideUser(p string) string {
	for _, sep := range []string{`\`, "/"} {
		pre := "Users" + sep
		i := strings.Index(p, pre)
		if i < 0 {
			continue
		}
		rest := p[i+len(pre):]
		j := strings.Index(rest, sep)
		if j < 0 {
			return p[:i+len(pre)] + "…"
		}
		return p[:i+len(pre)] + "…" + rest[j:]
	}
	return p
}

// base is the last element of a path.
func base(p string) string {
	p = strings.TrimRight(p, `\/`)
	if i := strings.LastIndexAny(p, `\/`); i >= 0 {
		return p[i+1:]
	}
	return p
}
