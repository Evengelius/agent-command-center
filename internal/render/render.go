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
	Field       int    // 0 agent CLI, 1 project directory, 2 Start
	Choice      string // the provider id the radio buttons select
	Path        string // the project field's text
	Resolved    string // the path the agent would receive, computed without I/O
	PathProblem string // why the typed form is refused, computed without I/O
	PathWarning string // a caution on an accepted path (UNC, WSL)
	Branch      string // the git branch, once read

	// Dashboard.
	Focus    string // "A", "B", "C" or "D"
	Draft    string // the editor's text
	Editor   EditorView
	Selected string // the agent selected in zone C
	Card     string // the approval card with the focus: it takes a, d, c (ui-design.md, Approvals)
	Spinner  int    // the spinner's frame
	Toast    string // replaces the key hints for a moment (ui-design.md, Clipboard)

	// Zone A's tree and agent inspection (ui-design.md, Zone A; Agent
	// inspection).
	ACursor     string          // the node under zone A's cursor (Node.ID); "" none
	Fold        map[string]bool // the owner's folds by node; a node not in it has its default
	Inspect     string          // the agent inspected full screen; "" none
	Tab         int             // the inspection's tab: 0 Messages, 1 Tools, 2 Diff, 3 Raw events
	InspectTop  int             // the first inspection row shown while held
	InspectHold bool            // scrolled up: the inspection stays where it is

	// Mouse selection and scrolling (ui-design.md, Selection and copy; Zone B,
	// Follow mode).
	Sel      Selection // the mouse selection, outside the editor
	ConvTop  int       // the first conversation row shown while held
	ConvHold bool      // scrolled up: the view stays where it is
	ConvSeen int       // the entries there were when the view was held
	MouseOff bool      // mouse capture off: the terminal selects

	// Wrap keeps the conversation's line breaks between frames; nil wraps
	// every line each frame, with the same result.
	Wrap *WrapCache
}

// Colours and symbols (ui-design.md, Colours and symbols).
var (
	cFrame     = lipgloss.Color("#3B3F51")
	cTitle     = lipgloss.Color("#C4B5FD")
	cText      = lipgloss.Color("#E5E7EB")
	cDone      = lipgloss.Color("#4ADE80")
	cProgress  = lipgloss.Color("#FBBF24")
	cAttention = lipgloss.Color("#F472B6")
	cFailed    = lipgloss.Color("#F87171")
	cGrey      = lipgloss.Color("#6B7280")

	stFrame     = style(lipgloss.NewStyle().Foreground(cFrame))
	stTitle     = style(lipgloss.NewStyle().Foreground(cTitle).Bold(true))
	stText      = style(lipgloss.NewStyle().Foreground(cText))
	stDone      = style(lipgloss.NewStyle().Foreground(cDone))
	stProgress  = style(lipgloss.NewStyle().Foreground(cProgress))
	stAttention = style(lipgloss.NewStyle().Foreground(cAttention))
	stFailed    = style(lipgloss.NewStyle().Foreground(cFailed))
	stGrey      = style(lipgloss.NewStyle().Foreground(cGrey))
	stBold      = style(lipgloss.NewStyle().Foreground(cText).Bold(true))
	stButton    = style(lipgloss.NewStyle().Foreground(lipgloss.Color("#111827")).Background(cTitle).Bold(true))

	stAttentionBold = style(stAttention.Bold(true))
	stFailedBold    = style(stFailed.Bold(true))
	stNone          = style(lipgloss.NewStyle())

	modelColours = [][2]string{{"opus", "#A78BFA"}, {"sonnet", "#FB923C"}, {"haiku", "#60A5FA"}, {"gpt", "#2DD4BF"}}
	modelStyles  = func() (out []*lipgloss.Style) {
		for _, kc := range modelColours {
			out = append(out, style(lipgloss.NewStyle().Foreground(lipgloss.Color(kc[1]))))
		}
		return out
	}()

	spinner = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
)

// style is a shared style. Spans hold a pointer to one, never a copy: a
// lipgloss.Style is hundreds of bytes, and spans equal by value let a frame
// reuse the rows a keystroke did not change (Screen.StringAfter).
func style(s lipgloss.Style) *lipgloss.Style { return &s }

// modelStyle colours a model name by its family.
func modelStyle(model string) *lipgloss.Style {
	m := strings.ToLower(model)
	for i, kc := range modelColours {
		if strings.Contains(m, kc[0]) {
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
