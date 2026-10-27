package render

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/Evengelius/agent-command-center/internal/state"
)

// WorktreesView is the Worktrees view, full screen over either screen
// (plan 16, amendment 10).
type WorktreesView struct {
	Open     bool
	Startup  bool   // over the startup screen: Enter starts, n creates
	Repo     string // the main worktree, or the directory asked about
	Rows     []WorktreeRow
	Sel      int
	Reading  bool
	Busy     string // the operation running
	Problem  string // the list cannot be read
	Note     string // the last operation's outcome
	Failed   bool   // the note is a refusal or a failure
	Sessions string // why other sessions are unknown; "" known
	Mode     byte   // 0 the list, 'n' a new branch, 'd' a diff, 'm' or 'x' a question
	Question string
	Branch   string // the new branch field
	NewPath  string // the path it creates; "" none yet
	Diff     []string
	DiffTop  int
	DiffOf   string
}

// WorktreeRow is one worktree.
type WorktreeRow struct {
	Branch, Path, Problem       string
	Main, Bare, Locked, Missing bool
	Changes, Ahead              int // Ahead -1 unknown
	Here, Other                 bool
}

// worktreesScreen draws the view: the list, or a diff full screen.
func worktreesScreen(s *state.State, ui UI) []line {
	if ui.Worktrees.Mode == 'd' {
		return worktreeDiff(s, ui)
	}
	w, h, v := ui.Width, ui.Height, ui.Worktrees
	cw := min(w-8, 110)
	margin := strings.Repeat(" ", (w-cw)/2)
	in := func(l line) line { return append(line{sp(margin, stText)}, l...) }
	note := func(mark, text string, st *lipgloss.Style) []line {
		var out []line
		for i, t := range wrap(hideUser(text), cw-2) {
			if i > 0 {
				mark = "  "
			}
			out = append(out, in(plain(mark+t, st)))
		}
		return out
	}

	top := []line{
		titleBar(s, w, plain("Worktrees ", stGrey), nil),
		rule(w),
		{},
		in(plain("Git worktrees", stTitle)),
		in(line{sp("Repo    ", stGrey), sp(elide(v.Repo, cw-8), stText)}),
		in(plain(strings.Repeat("─", cw), stFrame)),
	}
	var notes, rows []line
	if v.Problem != "" {
		notes = append(notes, note("✗ ", v.Problem, stFailed)...)
	}
	if v.Reading {
		notes = append(notes, in(plain("  reading…", stGrey)))
	}
	if v.Busy != "" {
		notes = append(notes, note("  ", v.Busy, stTitle)...)
	}
	if v.Question != "" {
		notes = append(notes, note("? ", v.Question+"  y/n", stAttention)...)
	}
	if v.Note != "" {
		mark, st := "✓ ", stDone
		if v.Failed {
			mark, st = "✗ ", stFailed
		}
		notes = append(notes, note(mark, v.Note, st)...)
	}
	if v.Sel < len(v.Rows) && v.Rows[v.Sel].Problem != "" && !v.Reading {
		notes = append(notes, note("✗ ", v.Rows[v.Sel].Problem, stFailed)...)
	}
	for i, r := range v.Rows {
		rows = append(rows, in(worktreeRow(r, i == v.Sel, cw)))
	}

	var form []line
	if v.Mode == 'n' {
		creates := line{sp("Creates ", stGrey), sp("type an existing branch, or a new one from main", stGrey)}
		if v.NewPath != "" {
			creates = line{sp("Creates ", stGrey), sp(elide(v.NewPath, cw-8), stText)}
		}
		form = []line{
			in(plain(strings.Repeat("─", cw), stFrame)),
			in(plain("New worktree", stTitle)),
			in(line{sp("Branch  ", stGrey), sp(v.Branch, stText), sp("█", stTitle)}.fit(cw)),
			in(creates),
		}
	}

	count := plural(len(v.Rows), "worktree", "worktrees")
	if v.Reading || v.Problem != "" {
		count = ""
	}
	if v.Sessions != "" && count != "" {
		count += " · other sessions: " + v.Sessions
	}
	bottom := []line{in(plain(strings.Repeat("─", cw), stFrame)), nil}

	first, n := 0, max(h-1-len(top)-len(notes)-len(form)-len(bottom), 1)
	if len(rows) > n {
		first = min(max(v.Sel-n/2, 0), len(rows)-n)
		rows = rows[first : first+n]
	}
	right := line{}
	if len(rows) < len(v.Rows) {
		right = plain(fmt.Sprintf("%d–%d of %d", first+1, first+len(rows), len(v.Rows)), stGrey)
	}
	bottom[1] = in(spread(plain(count, stGrey), right, cw))

	ls := append(append(top, notes...), rows...)
	for len(ls) < h-1-len(form)-len(bottom) {
		ls = append(ls, line{})
	}
	ls = append(append(ls[:min(len(ls), max(h-1-len(form)-len(bottom), 0))], form...), bottom...)
	return append(ls, plain(worktreesKeys(v), stGrey))
}

// worktreesKeys is the view's footer, within 79 cells.
func worktreesKeys(v WorktreesView) string {
	switch {
	case v.Mode == 'n':
		return " Enter create · Esc cancel · Ctrl+Q quit"
	case v.Mode == 'm' || v.Mode == 'x':
		return " y yes · n no"
	case len(v.Rows) == 0:
		return " Esc back · Ctrl+Q quit"
	case v.Startup:
		return " Enter start here · n new · d diff · m merge into main · x remove · Esc back"
	}
	return " ↑↓ move · d diff · m merge into main · x remove · Esc back to the session"
}

// worktreeRow is one worktree within w cells: the selection, its branch,
// its state, its commits ahead of main, a session in it, its path.
func worktreeRow(r WorktreeRow, selected bool, w int) line {
	mark, bst := "  ", stText
	if r.Main {
		bst = stBold
	}
	if selected {
		mark, bst = "› ", stTitle
	}
	bw := 18
	if w >= 90 {
		bw = 26
	}
	branch := line{sp(r.Branch, bst)}
	if r.Branch == "" {
		branch = line{sp("detached", stGrey)}
	}
	status := sp("clean", stDone)
	switch {
	case r.Missing:
		status = sp("missing", stFailed)
	case r.Bare:
		status = sp("bare", stGrey)
	case r.Problem != "":
		status = sp("N/A", stFailed)
	case r.Changes > 0:
		status = sp(fmt.Sprintf("+%d", r.Changes), stAttention)
	}
	ahead := sp("", stGrey)
	switch {
	case r.Main || r.Missing || r.Bare:
	case r.Ahead < 0:
		ahead = sp("↑N/A", stGrey)
	case r.Ahead > 0:
		ahead = sp(fmt.Sprintf("↑%d", r.Ahead), stTitle)
	default:
		ahead = sp("↑0", stGrey)
	}
	who := sp("", stGrey)
	switch {
	case r.Here:
		who = sp("this session", stTitle)
	case r.Other:
		who = sp("session", stAttention)
	case r.Locked:
		who = sp("locked", stGrey)
	}
	l := line{sp(mark, stTitle)}
	l = append(l, branch.fit(bw).pad(bw)...)
	l = append(l, sp(" ", stText))
	l = append(l, line{status}.pad(8)...)
	l = append(l, line{ahead}.pad(6)...)
	l = append(l, line{who}.pad(13)...)
	rest := w - l.width()
	return append(l, sp(elide(r.Path, rest), stGrey)).fit(w)
}

// worktreeDiff draws a worktree's diff full screen; it scrolls with
// DiffTop (the application keeps it within the rows).
func worktreeDiff(s *state.State, ui UI) []line {
	w, h, v := ui.Width, ui.Height, ui.Worktrees
	ls := []line{
		titleBar(s, w, plain("Worktrees ", stGrey), nil),
		rule(w),
		{sp(" Diff  ", stTitle), sp(v.DiffOf, stText)},
		rule(w),
	}
	room := max(h-7, 1)
	top := max(min(v.DiffTop, len(v.Diff)-room), 0)
	shown := v.Diff[top:min(top+room, len(v.Diff))]
	for _, l := range shown {
		ls = append(ls, diffRow(l, w))
	}
	for len(ls) < 4+room {
		ls = append(ls, line{})
	}
	count := ""
	switch {
	case v.Busy != "":
		count = v.Busy
	case len(v.Diff) == 0:
		count = "no change"
	default:
		count = fmt.Sprintf("%d–%d of %s", top+1, top+len(shown), plural(len(v.Diff), "line", "lines"))
	}
	ls = append(ls[:4+room], rule(w), plain(" "+count, stGrey))
	return append(ls, plain(" ↑↓ PgUp PgDn Home End scroll · Esc back to the list", stGrey))
}

// diffRow is one diff line, coloured as git colours it, within w cells.
func diffRow(l string, w int) line {
	st := stText
	switch {
	case strings.HasPrefix(l, "diff --git"), strings.HasPrefix(l, "index "),
		strings.HasPrefix(l, "+++ "), strings.HasPrefix(l, "--- "):
		st = stBold
	case strings.HasPrefix(l, "@@"):
		st = stTitle
	case strings.HasPrefix(l, "+"):
		st = stDone
	case strings.HasPrefix(l, "-"):
		st = stFailed
	case l == "Untracked files:":
		st = stAttention
	case strings.HasPrefix(l, "… ") && strings.HasSuffix(l, " lines left out"):
		st = stGrey
	}
	// Tabs as four spaces, as wrap and the editor draw them: code keeps its
	// indentation.
	return line{deco(" ", stNone), sp(strings.ReplaceAll(l, "\t", "    "), st)}.fit(w)
}

// plural is n and its noun.
func plural(n int, one, many string) string {
	if n == 1 {
		return "1 " + one
	}
	return fmt.Sprintf("%d %s", n, many)
}
