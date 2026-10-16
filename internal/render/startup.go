package render

import (
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/Evengelius/agent-command-center/internal/state"
)

// supported are the agent CLIs, in the order the startup screen lists them.
var supported = []struct{ id, name string }{
	{"claude-code", "Claude Code"},
	{"codex", "OpenAI Codex"},
}

// startup draws the startup screen (ui-design.md, Startup screen).
func startup(s *state.State, ui UI) []line {
	w, h := ui.Width, ui.Height
	cw := min(w-8, 76) // the content column
	margin := strings.Repeat(" ", (w-cw)/2)
	in := func(l line) line { return append(line{sp(margin, stText)}, l...) }
	sub := func(l line) line { return in(append(line{sp("   ", stText)}, l...)) }
	// note is a message under a field, wrapped whole, never cut: a failure
	// must be readable at 80 columns.
	note := func(mark, text string, st lipgloss.Style) []line {
		var out []line
		for i, t := range wrap(text, cw-3-textWidth(mark)) {
			if i > 0 {
				mark = strings.Repeat(" ", textWidth(mark))
			}
			out = append(out, sub(plain(mark+t, st)))
		}
		return out
	}

	ls := []line{
		titleBar(s, w, plain("New session ", stGrey), nil),
		rule(w),
		{},
		in(plain("Configure your workspace", stTitle)),
		in(plain("Select an agent CLI and the project to work on.", stGrey)),
		{},
		in(question("1. Agent CLI", ui.Field == 0)),
	}
	detecting := s.Startup.Phase == state.Detecting || s.Startup.Phase == ""
	for _, c := range supported {
		for _, l := range cliRow(s, ui, c.id, c.name, detecting, cw-3) {
			ls = append(ls, sub(l))
		}
	}
	none := !detecting && len(s.Startup.Installed) == 0 // Start stays disabled
	if none {
		ls = append(ls, note("✗ ", "No supported agent CLI found. Install Claude Code or Codex CLI, then restart.", stFailed)...)
	} else if s.Startup.Field == "provider" && s.Startup.Message != "" {
		ls = append(ls, note("✗ ", s.Startup.Message, stFailed)...)
	}
	ls = append(ls, in(plain(strings.Repeat("─", cw), stFrame)))
	ls = append(ls, in(question("2. Project directory", ui.Field == 1)))

	fw := cw - 6 // the field's frame
	border := stFrame
	if ui.Field == 1 {
		border = stTitle
	}
	text := ui.Path
	cursor := ""
	if ui.Field == 1 {
		cursor = "█"
	}
	if textWidth(text)+1 > fw-4 {
		text = "…" + string(tailRunes([]rune(text), fw-6))
	}
	ls = append(ls,
		sub(plain("╭"+strings.Repeat("─", fw-2)+"╮", border)),
		sub(line{sp("│ ", border), sp(text, stText), sp(cursor, stTitle)}.fit(fw-1).pad(fw-1).append(sp("│", border))),
		sub(plain("╰"+strings.Repeat("─", fw-2)+"╯", border)),
	)
	switch {
	case s.Startup.Field == "project" && s.Startup.Message != "":
		ls = append(ls, note("✗ ", hideUser(s.Startup.Message), stFailed)...)
	case ui.PathProblem != "":
		// Live, before Start: grey until Start makes it a failure.
		ls = append(ls, note("✗ ", hideUser(ui.PathProblem), stGrey)...)
	case ui.Resolved != "":
		l := line{sp("→ ", stGrey), sp(elide(ui.Resolved, cw-20), stText)}
		if ui.Branch != "" {
			l = append(l, sp("   git: "+ui.Branch, stGrey))
		}
		ls = append(ls, sub(l))
		if ui.PathWarning != "" {
			ls = append(ls, note("! ", ui.PathWarning, stAttention)...)
		}
	default:
		ls = append(ls, sub(plain("→ type a directory", stGrey)))
	}
	ls = append(ls, sub(plain("Type a full path; ~ and /mnt/c/… are expanded. Default: current dir.", stGrey)))
	ls = append(ls, in(plain(strings.Repeat("─", cw), stFrame)))

	bw := min(44, cw)
	bm := strings.Repeat(" ", (w-bw)/2)
	label := "Start session  →"
	switch {
	case detecting:
		label = spinner[ui.Spinner%len(spinner)] + " detecting agent CLIs…"
	case s.Startup.Phase == state.Validating:
		label = spinner[ui.Spinner%len(spinner)] + " validating…"
	}
	bst, lst := stFrame, stBold
	switch {
	case none:
		bst, lst = stFrame, stGrey
	case ui.Field == 2:
		bst = stTitle
	}
	lpad := (bw - 2 - textWidth(label)) / 2
	ls = append(ls,
		line{sp(bm, stText), sp("╭"+strings.Repeat("─", bw-2)+"╮", bst)},
		line{sp(bm, stText), sp("│", bst), sp(strings.Repeat(" ", lpad)+label, lst), sp("", stText)}.fit(len(bm)+bw-1).append(sp("│", bst)),
		line{sp(bm, stText), sp("╰"+strings.Repeat("─", bw-2)+"╯", bst)},
	)
	// Wrapped messages can make the screen taller than the terminal: the
	// blank spacing lines go first, so the Start button stays whole.
	for i := len(ls) - 1; i >= 0 && len(ls) > h-1; i-- {
		if len(ls[i]) == 0 {
			ls = append(ls[:i], ls[i+1:]...)
		}
	}
	for len(ls) < h-1 {
		ls = append(ls, line{})
	}
	ls = ls[:h-1]
	return append(ls, plain(" ↑↓ agent · Tab next field · Enter start · Ctrl+Q quit", stGrey))
}

func question(text string, focused bool) line {
	st := stBold
	if focused {
		st = stTitle
	}
	return line{sp(text+" ", st), sp("*", stFailed)}
}

// cliRow is one radio button: name, version and path, or why it cannot be
// chosen, wrapped under the version column within w cells.
func cliRow(s *state.State, ui UI, id, name string, detecting bool, w int) []line {
	radio := "( ) "
	if ui.Choice == id && !detecting {
		radio = "(•) "
	}
	l := line{sp(radio, stTitle), sp(padTo(name, 16), stBold)}
	if detecting {
		return []line{append(l, sp(spinner[ui.Spinner%len(spinner)]+" detecting…", stGrey))}
	}
	for _, c := range s.Startup.CLIs {
		if c.ID != id {
			continue
		}
		if !c.Installed {
			why := "not installed"
			if c.Problem != "" {
				why += ": " + c.Problem
			}
			var out []line
			for i, t := range wrap(why, w-20) {
				lead := line{sp("( ) ", stGrey), sp(padTo(name, 16), stGrey)}
				if i > 0 {
					lead = line{sp(strings.Repeat(" ", 20), stGrey)}
				}
				out = append(out, append(lead, sp(t, stGrey)))
			}
			return out
		}
		return []line{append(l, sp(padTo(c.Version, 10), stText), sp(elide(c.Path, w-30), stGrey))}
	}
	for _, inst := range s.Startup.Installed {
		if inst == id {
			return []line{append(l, sp("installed", stText))}
		}
	}
	return []line{{sp("( ) ", stGrey), sp(padTo(name, 16), stGrey), sp("not installed", stGrey)}}
}

func (l line) append(s ...span) line { return append(l, s...) }

func padTo(s string, w int) string {
	if n := w - textWidth(s); n > 0 {
		return s + strings.Repeat(" ", n)
	}
	return s + " "
}

func tailRunes(rs []rune, w int) []rune {
	for len(rs) > 0 && textWidth(string(rs)) > w {
		rs = rs[1:]
	}
	return rs
}
