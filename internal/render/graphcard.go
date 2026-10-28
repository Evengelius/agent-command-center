package render

import (
	"strconv"
	"strings"
	"time"
)

// The Git Graph's details (plan 62, designs D12 and D13): the footer sums up
// the cursor's row while the panel has the focus; the card, with the pointer
// on a row or i for the cursor's, tells more beside it.

// GraphCard is the card the application holds for a row (design D13).
type GraphCard struct {
	// Row is the row in GraphView.Rows the card is about.
	Row int
	// Message is a commit's whole message, "" until read; Files its files
	// summed up ("3 files changed, +12 −3"), "" until read.
	Message, Files string
	// State is said in the message's place while it is read, or its read's
	// failure (Failed).
	State  string
	Failed bool
}

// graphCardMax is how many lines of a message the card shows (design D13).
const graphCardMax = 20

// graphFooter is the footer while the panel has the focus (designs D10 and
// D12): the cursor's commit summed up, its short hash, its author and when,
// or its file's full path and its status in words; then the note of lanes
// hidden and the keys, as many as fit.
func graphFooter(ui UI, w int) line {
	v := ui.Graph
	dot := deco(" · ", stGrey)
	l := line{deco(" ", stNone)}
	if v.State == "" && v.Cursor >= 0 && v.Cursor < len(v.Rows) {
		r := v.Rows[v.Cursor]
		e := v.Commits[r.Commit]
		switch fs := v.Files[r.Commit]; {
		case r.File < 0:
			l = append(l, deco(e.Short(), stTitle), dot, deco(e.Author, stText), dot, deco(commitAgo(e.Time, ui.Now, v.Loc), stGrey), dot)
		case fs != nil && int(r.File) < len(fs.Files):
			f := fs.Files[r.File]
			l = append(l, deco(f.Path, stText), dot, deco(statusWords(f), stGrey), dot)
		}
	}
	if room := w - l.width(); room > 0 {
		l = append(l, deco(graphKeys(room, v.hidden, GraphWide(ui)), stGrey))
	}
	return l.fit(w)
}

// statusWords is a file's status as the footer and the card say it.
func statusWords(f GraphFile) string {
	switch f.Status {
	case 'A':
		return "Added"
	case 'D':
		return "Deleted"
	case 'R':
		return "Renamed from " + f.Orig
	case 'T':
		return "Type changed"
	case 'M':
		return "Modified"
	}
	return string(f.Status)
}

// commitAgo is when a commit was made, Unix seconds at, from now: "just
// now", "5 minutes ago", "2 days ago", "3 months ago"; the date in loc's
// zone, nil the machine's, without now or for a time to come.
func commitAgo(at int64, now time.Time, loc *time.Location) string {
	t := time.Unix(at, 0)
	d := now.Sub(t)
	if now.IsZero() || d < 0 {
		return commitDate(at, loc)
	}
	n, unit := 0, ""
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		n, unit = int(d/time.Minute), "minute"
	case d < 24*time.Hour:
		n, unit = int(d/time.Hour), "hour"
	case d < 14*24*time.Hour:
		n, unit = int(d/(24*time.Hour)), "day"
	case d < 60*24*time.Hour:
		n, unit = int(d/(7*24*time.Hour)), "week"
	case d < 365*24*time.Hour:
		n, unit = int(d/(30*24*time.Hour)), "month"
	default:
		n, unit = int(d/(365*24*time.Hour)), "year"
	}
	if n != 1 {
		unit += "s"
	}
	return strconv.Itoa(n) + " " + unit + " ago"
}

// commitDate is the date and time of at in loc's zone, nil the machine's.
func commitDate(at int64, loc *time.Location) string {
	if loc == nil {
		loc = time.Local
	}
	return time.Unix(at, 0).In(loc).Format("02/01/2006 15:04")
}

// graphCard draws the card of ui.Graph.Card beside its row (design D13):
// left of the panel where room remains, else in the panel under the row or
// over it, never over the row itself; nothing when the row is not in view.
// A commit's: its author and when, relative and exact; its whole message
// wrapped, graphCardMax lines at most, then "…"; its files summed up; its
// labels; its short hash in the top edge. A file's: its full path and its
// status in words. Its cells are the panel's frame's: a click on it does
// nothing.
func graphCard(sc *Screen, ui UI) {
	v, c := ui.Graph, ui.Graph.Card
	if c == nil || c.Row < 0 || c.Row >= len(v.Rows) {
		return
	}
	_, y, ok := sc.findItem(GraphPanel, c.Row+1)
	if !ok {
		return
	}
	cw := min(56, max(ui.Width-2, 20)) - 4 // the text's cells
	r := v.Rows[c.Row]
	e := v.Commits[r.Commit]
	title := " " + e.Short() + " "
	var body []line
	if r.File < 0 {
		body = append(body, line{deco(fitText(e.Author, cw), stText)},
			line{deco(fitText(commitAgo(e.Time, ui.Now, v.Loc)+" ("+commitDate(e.Time, v.Loc)+")", cw), stGrey)}, nil)
		switch {
		case c.State != "":
			body = append(body, line{deco(fitText(c.State, cw), either(c.Failed, stAttention, stGrey))})
		default:
			var msg []line
			for _, p := range strings.Split(c.Message, "\n") {
				if strings.TrimSpace(p) == "" {
					msg = append(msg, nil)
					continue
				}
				msg = append(msg, wrapDeco(p, cw, "", stText)...)
			}
			if len(msg) > graphCardMax {
				msg = append(msg[:graphCardMax-1], line{deco("…", stGrey)})
			}
			body = append(body, msg...)
		}
		if c.Files != "" {
			body = append(body, nil, line{deco(fitText(c.Files, cw), stGrey)})
		}
		if ls := v.Labels[r.Commit]; len(ls) > 0 {
			var l line
			for i, lb := range ls {
				if i > 0 {
					l = append(l, deco("  ", stNone))
				}
				l = append(l, deco(labelText(lb), labelStyle(lb)))
			}
			body = append(body, l.fit(cw))
		}
	} else if fs := v.Files[r.Commit]; fs != nil && int(r.File) < len(fs.Files) {
		f := fs.Files[r.File]
		title = " " + string(f.Status) + " "
		body = append(body, wrapDeco(f.Path, cw, "", stText)...)
		body = append(body, line{deco(fitText(statusWords(f), cw), stGrey)})
	} else {
		return
	}
	w, h := cw+4, len(body)+2
	box := []line{{deco("╭─", stModelFrame), deco(title, stModelHead), deco(strings.Repeat("─", max(w-3-textWidth(title), 0))+"╮", stModelFrame)}}
	for _, l := range body {
		box = append(box, append(append(line{deco("│ ", stModelFrame)}, padRow(l, cw)...), deco(" │", stModelFrame)))
	}
	box = append(box, line{deco("╰"+strings.Repeat("─", w-2)+"╯", stModelFrame)})
	for i := range box {
		for j := range box[i] {
			box[i][j].ref = ref{panel: GraphPanel, kind: refDeco, item: 0}
		}
	}
	top, bottom := 2, len(sc.lines)-1 // under the bars, over the footer
	x := sc.Panel.X - w
	cy := min(max(y-1, top), bottom-h)
	if x < 0 {
		// No room left of the panel: in it, at its right, under the row or
		// over it.
		x = max(sc.Panel.X+sc.Panel.W-w, 0)
		switch {
		case y+1+h <= bottom:
			cy = y + 1
		case y-h >= top:
			cy = y - h
		default:
			return
		}
	}
	if cy < 0 || cy+h > len(sc.lines) {
		return
	}
	sc.place(box, x, cy, w)
}
