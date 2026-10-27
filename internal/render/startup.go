package render

import "strings"

// supported are the agent CLIs, in the order the startup screen lists them.
var supported = []struct{ id, name string }{
	{"claude-code", "Claude Code"},
	{"codex", "OpenAI Codex"},
}

// The startup screen is the home screen (plan 51): home.go, homepaint.go
// and hometile.go draw its picture, homegrid.go its text.

// ProjectRow is one entry of the startup screen's project list (plan 15).
type ProjectRow struct {
	Path      string
	Age       string // since its last start: "3h ago"
	Favourite bool
}

func padTo(s string, w int) string {
	if n := w - textWidth(s); n > 0 {
		return s + strings.Repeat(" ", n)
	}
	return s + " "
}
