package render

import (
	"fmt"
	"strings"

	"github.com/Evengelius/agent-command-center/internal/state"
)

// BrowserView is the project browser over the startup screen (plan 15).
type BrowserView struct {
	Open       bool
	Dir        string // the directory shown; "" the drives
	Rows       []BrowserRow
	Sel        int
	Filter     string
	Shown      int // the subdirectories listed
	Total      int // those the filter can match
	Hidden     int // the hidden ones left out
	ShowHidden bool
	Reading    bool
	Problem    string // the directory cannot be listed
	Store      bool   // a project list to star into
}

// BrowserRow is one row: Kind 'u' the directory shown, 'p' its parent, 'd' a
// subdirectory, 'r' a drive.
type BrowserRow struct {
	Kind      byte
	Name      string
	Git       bool
	Link      bool
	Favourite bool
}

// browserScreen draws the browser full screen: the directory, the filter,
// the rows around the selected one, the counts and the keys.
func browserScreen(s *state.State, ui UI) []line {
	w, h, v := ui.Width, ui.Height, ui.Browser
	cw := min(w-8, 100)
	margin := strings.Repeat(" ", (w-cw)/2)
	in := func(l line) line { return append(line{sp(margin, stText)}, l...) }

	where := line{sp("Drives", stText)}
	if v.Dir != "" {
		where = line{sp(elide(v.Dir, cw-4), stText)}
	}
	filter := line{sp("Filter  ", stGrey), sp(v.Filter, stText), sp("█", stTitle)}
	if v.Filter == "" {
		filter = line{sp("Filter  ", stGrey), sp("█", stTitle), sp(" type to filter", stGrey)}
	}
	top := []line{
		titleBar(s, w, plain("Project browser ", stGrey), nil),
		rule(w),
		{},
		in(plain("Choose a project directory", stTitle)),
		in(append(line{sp("In      ", stGrey)}, where...)),
		in(filter),
		in(plain(strings.Repeat("─", cw), stFrame)),
	}
	var notes, rows []line
	for i, t := range wrap(hideUser(v.Problem), cw-4) {
		if v.Problem == "" {
			break
		}
		mark := "  "
		if i == 0 {
			mark = "✗ "
		}
		notes = append(notes, in(plain(mark+t, stFailed)))
	}
	if v.Reading {
		notes = append(notes, in(plain("  reading…", stGrey)))
	}
	for i, r := range v.Rows {
		rows = append(rows, in(browserRow(r, i == v.Sel, cw)))
	}
	if !v.Reading && v.Problem == "" && v.Shown == 0 && v.Dir != "" {
		empty := "  no subdirectory"
		if v.Filter != "" {
			empty = "  no directory matches the filter"
		}
		rows = append(rows, in(plain(empty, stGrey)))
	}

	count := plural(v.Total, "directory", "directories")
	switch {
	case v.Reading:
		count = ""
	case v.Dir == "":
		count = plural(len(v.Rows), "drive", "drives")
	case v.Filter != "":
		count = fmt.Sprintf("%d of %s match", v.Shown, count)
	}
	switch {
	case v.Reading || v.Dir == "":
	case v.Hidden > 0:
		count += fmt.Sprintf(" · %d hidden, Ctrl+H shows them", v.Hidden)
	case v.ShowHidden:
		count += " · hidden shown, Ctrl+H hides them"
	}
	bottom := []line{in(plain(strings.Repeat("─", cw), stFrame)), nil}

	first, n := 0, max(h-1-len(top)-len(notes)-len(bottom), 1)
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
	for len(ls) < h-1-len(bottom) {
		ls = append(ls, line{})
	}
	ls = append(ls[:min(len(ls), h-1-len(bottom))], bottom...)

	keys := " Enter open"
	if v.Sel < len(v.Rows) && v.Rows[v.Sel].Kind == 'u' {
		keys = " Enter start here"
	}
	keys += " · ↑↓ move · ← up"
	if v.Store {
		keys += " · Ctrl+F star"
	}
	keys += " · Ctrl+H hidden"
	if v.Filter != "" {
		keys += " · Esc clear"
	} else {
		keys += " · Esc back"
	}
	return append(ls, plain(keys, stGrey))
}

// plural is n and its noun.
func plural(n int, one, many string) string {
	if n == 1 {
		return "1 " + one
	}
	return fmt.Sprintf("%d %s", n, many)
}

// browserRow is one row within w cells: the selection, the star, the name,
// and on the right what it is.
func browserRow(r BrowserRow, selected bool, w int) line {
	mark, nst := "  ", stText
	if selected {
		mark, nst = "› ", stTitle
	}
	star := "  "
	if r.Favourite {
		star = "★ "
	}
	l := line{sp(mark, stTitle), sp(star, stAttention)}
	var right line
	switch r.Kind {
	case 'u':
		if !selected {
			nst = stBold
		}
		l = append(l, sp("Use this directory", nst))
		right = plain("start the session here", stGrey)
	case 'p':
		l = append(l, sp("..", nst))
		right = plain("parent", stGrey)
	case 'r':
		l = append(l, sp(r.Name, nst))
		right = plain("drive", stGrey)
	default:
		l = append(l, line{sp(r.Name, nst)}.fit(w-4-12)...)
		if r.Link {
			right = append(right, sp("link", stGrey))
		}
		if r.Git {
			if len(right) > 0 {
				right = append(right, sp("  ", stText))
			}
			right = append(right, sp("git", stDone))
		}
	}
	return spread(l, right, w)
}
