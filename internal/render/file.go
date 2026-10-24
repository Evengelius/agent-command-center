package render

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/Evengelius/agent-command-center/internal/state"
)

// The file viewer (plan 17, The window): a read-only window floating over
// the dashboard, which is drawn and updated behind it, and the quick open in
// a window of the same size. The application reads, colours and searches
// the file in commands; this draws what it holds.

// FileView is the file viewer as the application holds it.
type FileView struct {
	Open, Full bool
	Path       string // as named: absolute, or relative to the project
	Rel        string // relative to the project, once read
	Opened     string // what opened it: "read by Read", "quick open"…
	State      string // "reading…", or why the file is not shown; "" once shown
	Failed     bool   // State is a refusal or an error, not a step
	Lines      []string
	Hl         Highlighted // the colours of lines Hl.From on; the others uncoloured
	Colouring  bool        // a colouring is under way
	Windowed   bool        // too large to colour whole: coloured around the view
	Cursor     int         // the cursor line, 0-based
	Top        int         // the first line shown, before FileTop keeps the cursor in view
	Left       int         // the columns the code is shifted by
	Mark       [2]int      // the lines the agent edited, [from, to), 0-based
	Field      byte        // '/' or 'g' while the owner types it; 0 none
	Typed      string      // the field's text
	Query      string      // the search Matches are of; "" none
	Matches    []FileMatch // in file order
	Match      int         // the current match; -1 none
	Searching  bool
	Capped     bool // the search stopped at MaxMatches
}

// FileMatch is a search match: a line, 0-based, and a byte range in it.
type FileMatch struct{ Line, Off, N int }

// MaxMatches is how many matches a search keeps (plan 17, Keys).
const MaxMatches = 10_000

// FinderView is the quick open (plan 17, Opening).
type FinderView struct {
	Open      bool
	Query     string
	Files     []string // the files the query keeps, best first
	Total     int      // the files listed
	Sel, Top  int
	Truncated bool   // the listing stopped at its limit
	State     string // "listing…", or why there is no list; "" once listed
	Failed    bool   // State is an error
}

// FileRect is the window's place on a w by h screen: about 80% of it,
// centred, or full screen all of it under the title bar, as the inspection
// is, so the title bar and its MOCK badge stay on screen (plan 17, The
// window; plan 21, item 5.7).
func FileRect(w, h int, full bool) (x, y, ww, hh int) {
	if full {
		return 0, 1, w, h - 1
	}
	ww, hh = min(max(w*4/5, min(w-4, 84)), w), min(max(h*4/5, min(h-2, 22)), h)
	return (w - ww) / 2, (h - hh) / 2, ww, hh
}

// FileRows is how many lines of the file, or of the quick open's list, the
// window shows: its rows less the frame, the path or field row and the
// footer.
func FileRows(ui UI) int {
	_, _, _, hh := FileRect(ui.Width, ui.Height, fileFull(ui))
	return max(hh-4, 1)
}

// ListTop is the first of n rows shown in room rows: top moved the least to
// keep the cursor row cur in view.
func ListTop(top, cur, room, n int) int {
	top = min(max(top, cur-room+1), cur)
	return max(0, min(top, n-room))
}

// FileTop is the first line the window shows.
func FileTop(f FileView, rows int) int { return ListTop(f.Top, f.Cursor, rows, len(f.Lines)) }

// WindowOpen is true while the file viewer or the quick open is open. The
// Explorer's panel is docked, no window: F4, Alt+P, the "+" menu, the
// selectors and a paste work while it is shown (plan 43, design D8).
func WindowOpen(ui UI) bool { return ui.File.Open || ui.Finder.Open }

// fileFull is true while the window fills the screen: the viewer in front,
// full screen.
func fileFull(ui UI) bool { return ui.File.Open && ui.File.Full && !ui.Finder.Open }

// floatWindow draws the window in front over the frame drawn behind it: the
// quick open, else the viewer (plan 17; the Explorer docked since plan 43,
// design D8).
func floatWindow(s *state.State, ui UI, sc *Screen) {
	x, y, ww, hh := FileRect(ui.Width, ui.Height, fileFull(ui))
	var win []line
	if ui.Finder.Open {
		win = finderWindow(ui, ww, hh)
	} else {
		win = fileWindow(s, ui, ww, hh)
		sc.fileLines = ui.File.Lines
	}
	sc.place(win, x, y, ww)
}

// place draws the window win, ww wide, at (x, y) over the frame drawn behind
// it, each row built once in a line of its final length.
func (sc *Screen) place(win []line, x, y, ww int) {
	for i, l := range win {
		b := sc.lines[y+i]
		row := appendCols(make(line, 0, len(b)+len(l)+2), b, 0, x)
		sc.lines[y+i] = appendCols(append(row, l...), b, x+ww, b.width())
	}
}

// cols is the part of l between the columns from and to, exactly that wide:
// a wide character the edge cuts becomes spaces, and a span cut in part is
// decoration, its source offsets no longer true.
func cols(l line, from, to int) line { return appendCols(nil, l, from, to) }

// appendCols appends cols(l, from, to) to out.
func appendCols(out, l line, from, to int) line {
	x, n := 0, 0 // the column in l, the cells appended
	for _, s := range l {
		w := textWidth(s.text)
		switch {
		case x+w <= from || x >= to:
		case x >= from && x+w <= to:
			out = append(out, s)
			n += w
		default:
			c := s
			c.text, c.ref = cutCols(s.text, x, from, to), ref{panel: s.ref.panel, kind: refDeco}
			out = append(out, c)
			n += textWidth(c.text)
		}
		x += w
	}
	if n < to-from {
		out = append(out, deco(strings.Repeat(" ", to-from-n), stNone))
	}
	return out
}

// cutCols is the part of text, starting at column x, between the columns
// from and to: a slice of it unless the edge cuts a wide character, which
// becomes spaces.
func cutCols(text string, x, from, to int) string {
	start, end, cx := -1, len(text), x
	for i, r := range text {
		rw := runeWidth(r)
		switch {
		case rw == 0:
		case cx >= from && cx+rw <= to:
			if start < 0 {
				start = i
			}
		case cx < to && cx+rw > from:
			return cutWide(text, x, from, to)
		case cx >= to:
			end = i
		}
		if end < len(text) {
			break
		}
		cx += rw
	}
	if start < 0 {
		return ""
	}
	return text[start:end]
}

// cutWide is cutCols where the edge cuts a wide character.
func cutWide(text string, x, from, to int) string {
	var b strings.Builder
	cx, kept := x, false
	for _, r := range text {
		rw := runeWidth(r)
		switch {
		case rw == 0:
			if kept {
				b.WriteRune(r)
			}
		case cx >= from && cx+rw <= to:
			b.WriteRune(r)
			kept = true
		case cx < to && cx+rw > from:
			b.WriteString(strings.Repeat(" ", min(cx+rw, to)-max(cx, from)))
			kept = false
		default:
			kept = false
		}
		cx += rw
	}
	return b.String()
}

// frameIn makes every span of a window not marked otherwise decoration of
// panel F.
func frameIn(ls []line) []line {
	for _, l := range ls {
		for i := range l {
			if l[i].ref.kind == refNone {
				l[i].ref.kind = refDeco
			}
		}
	}
	return in('F', ls)
}

// fileWindow is the file viewer's window, ww by hh.
func fileWindow(s *state.State, ui UI, ww, hh int) []line {
	f := ui.File
	inner := ww - 4
	title := line{deco(baseName(f.Path), stTitle)}
	var right line
	if f.State == "" {
		r := grouped(len(f.Lines)) + " lines"
		if len(f.Lines) == 1 {
			r = "1 line"
		}
		if f.Hl.Lang != "" {
			r = f.Hl.Lang + " · " + r
		}
		right = decoLine(r, stGrey)
	}
	status := ""
	switch {
	case f.Colouring:
		status = "colouring…"
	case f.Windowed:
		status = "coloured around the view"
	}
	shown := f.Rel
	if shown == "" {
		shown = f.Path
	}
	opened := ""
	if f.Opened != "" {
		opened = " · " + f.Opened
	}
	room := inner - textWidth(opened) - textWidth(status) - 1
	body := []line{spread(line{deco(elide(shown, max(room, 8)), stText), deco(opened, stGrey)}, decoLine(status, stProgress), inner)}

	rows := hh - 4
	switch {
	case f.State != "":
		st := stGrey
		if f.Failed {
			st = stAttention
		}
		body = append(body, decoLine(f.State, st))
	case len(f.Lines) == 0:
		body = append(body, decoLine("empty file", stGrey))
	default:
		top := FileTop(f, rows)
		nw := len(strconv.Itoa(len(f.Lines)))
		for i := top; i < top+rows && i < len(f.Lines); i++ {
			body = append(body, codeRow(f, i, nw, inner))
		}
	}
	for len(body) < rows+1 {
		body = append(body, nil)
	}
	body = append(body[:rows+1], fileFooter(s, ui, inner))
	return frameIn(frameBox(title, right, body, ww, hh, stFileFrame))
}

// baseName is the last element of a path, Windows or slash form.
func baseName(p string) string {
	if i := strings.LastIndexAny(p, `/\`); i >= 0 {
		return p[i+1:]
	}
	return p
}

// codeRow is line i of the file w cells wide: its number right-aligned in nw
// cells, the edit mark, then the code shifted by f.Left.
func codeRow(f FileView, i, nw, w int) line {
	cur := i == f.Cursor
	marked := i >= f.Mark[0] && i < f.Mark[1]
	numSt, markSt, padSt := stGrey, stAttention, stNone
	if marked {
		numSt = stAttention
	}
	if cur {
		numSt, markSt, padSt = stFileRowNum, stFileRowMark, stTreeCursor
		if marked {
			numSt = stFileRowMark
		}
	}
	bar := " "
	if marked {
		bar = "▌"
	}
	l := line{deco(fmt.Sprintf("%*d", nw, i+1), numSt), deco(bar, markSt), deco(" ", padSt)}
	cw := max(w-nw-2, 1)
	code := codeSpans(f, i, cur, cw)
	if code.width() > cw {
		code = code.fit(cw)
	}
	l = append(l, code...)
	if n := w - l.width(); n > 0 {
		l = append(l, deco(strings.Repeat(" ", n), padSt))
	}
	return l
}

// codeSpans is line i's code from column f.Left, one cell more than cw when
// it goes on, so that fit ends it with "…": text spans in each run's class
// style, matches on an amber background, tabs to the next multiple of 4 and
// control characters as atoms.
func codeSpans(f FileView, i int, cur bool, cw int) line {
	text := f.Lines[i]
	runs := []HlRun{{text, SynText}}
	if k := i - f.Hl.From; k >= 0 && k < len(f.Hl.Lines) && len(f.Hl.Lines[k]) > 0 {
		runs = f.Hl.Lines[k]
	}
	m0 := sort.Search(len(f.Matches), func(j int) bool { return f.Matches[j].Line >= i })
	var out line
	start, startSt := -1, (*lipgloss.Style)(nil) // the text span being built
	flush := func(end int) {
		if start >= 0 {
			out = append(out, span{text: text[start:end], st: startSt, ref: ref{panel: 'F', kind: refText, item: i, off: start}})
			start = -1
		}
	}
	ri, rEnd := 0, len(runs[0].Text)
	mi := m0
	col, stop, end := 0, f.Left+cw+1, len(text)
	for off, r := range text {
		if col >= stop {
			end = off
			break
		}
		for ri < len(runs)-1 && off >= rEnd {
			ri++
			rEnd += len(runs[ri].Text)
		}
		st := runs[ri].Class.style()
		if cur {
			st = synRow[runs[ri].Class]
		}
		for mi < len(f.Matches) && f.Matches[mi].Line == i && off >= f.Matches[mi].Off+f.Matches[mi].N {
			mi++
		}
		if mi < len(f.Matches) && f.Matches[mi].Line == i && off >= f.Matches[mi].Off {
			st = stMatch
			if mi == f.Match {
				st = stMatchCur
			}
		}
		c := cellAt(text, off, r)
		if r == '\t' {
			c.text, c.w = strings.Repeat(" ", TabWidth-col%TabWidth), TabWidth-col%TabWidth
		}
		atom := c.ctrl || r == '\t'
		switch {
		case col+c.w <= f.Left && c.w > 0:
		case col < f.Left:
			flush(off)
			out = append(out, deco(strings.Repeat(" ", col+c.w-f.Left), st))
		case atom:
			flush(off)
			if c.ctrl {
				st = stControl
			}
			out = append(out, span{text: c.text, st: st, ref: ref{panel: 'F', kind: refAtom, item: i, off: off, n: c.n}})
		case c.w == 0 && start < 0 && len(out) == 0:
		default:
			if start >= 0 && st != startSt {
				flush(off)
			}
			if start < 0 {
				start, startSt = off, st
			}
		}
		col += c.w
	}
	flush(end)
	if len(out) == 0 { // an anchor, so that a click or a drag finds the line
		out = line{{st: stText, ref: ref{panel: 'F', kind: refText, item: i, off: len(text)}}}
	}
	return out
}

// fileKeys are the window's key hints, the longest first; the footer shows
// the first that fits.
var fileKeys = []string{
	"↑↓ PgUp PgDn Home End move · ←→ shift · / search · n N next, previous · g go to line · f full screen · F1 keys · Esc back · F3 close",
	"↑↓ PgUp PgDn · ←→ shift · / search · n N · g line · f full screen · Esc back · F3 close",
	"↑↓ PgUp PgDn · ←→ shift · / search · n N · g line · f full · Esc back · F3 close",
	"↑↓ PgUp PgDn · / search · n N · g line · f full · Esc back · F3 close",
	"↑↓ · / search · n N · g line · f full · Esc back · F3 close",
	"/ search · g line · f full · Esc back · F3 close",
}

// fileFooter is the window's footer, w cells: the keys, or a toast, the
// mouse capture or a waiting approval, and on the right the cursor line,
// the search or the field being typed.
func fileFooter(s *state.State, ui UI, w int) line {
	f := ui.File
	var right line
	keys := fileKeys
	switch {
	case f.Field == '/':
		right = line{deco("/", stTitle), deco(f.Typed, stText), deco("▌", stText)}
		keys = []string{"Enter find · Esc cancel"}
	case f.Field == 'g':
		right = line{deco("go to line ", stGrey), deco(f.Typed, stText), deco("▌", stText)}
		keys = []string{"Enter go · Esc cancel"}
	case f.Searching:
		right = decoLine("searching…", stGrey)
	case f.Query != "" && len(f.Matches) == 0:
		right = decoLine("no match", stAttention)
	case f.Query != "":
		n := grouped(len(f.Matches))
		if f.Capped {
			n += "+"
		}
		right = decoLine(fmt.Sprintf("match %s of %s", grouped(max(f.Match, 0)+1), n), stProgress)
	case f.State == "" && len(f.Lines) > 0:
		right = decoLine(fmt.Sprintf("line %s/%s", grouped(f.Cursor+1), grouped(len(f.Lines))), stGrey)
	}
	return spread(windowLeft(s, ui, keys, w-right.width()-1), right, w)
}

// windowLeft is a window footer's left side in room cells: a toast, the
// mouse capture off, a waiting approval, else the first keys that fit.
func windowLeft(s *state.State, ui UI, keys []string, room int) line {
	switch {
	case ui.Toast != "":
		return decoLine(ui.Toast, stProgress)
	case ui.MouseOff:
		return decoLine("mouse capture off · F12 back on", stProgress)
	case anyPending(s):
		return decoLine("◆ approval waiting · Esc, then F2", stAttentionBold)
	}
	for _, k := range keys {
		if textWidth(k) <= room {
			return decoLine(k, stGrey)
		}
	}
	return decoLine(keys[len(keys)-1], stGrey)
}

// anyPending is true while an approval waits for the owner.
func anyPending(s *state.State) bool {
	for _, ap := range s.Approvals {
		if ap.Status == state.Pending || ap.Status == state.Answering {
			return true
		}
	}
	return false
}

// finderKeys are the quick open's key hints, the longest first.
var finderKeys = []string{
	"type to filter · ↑↓ PgUp PgDn select · Enter open · F1 keys · Esc back · F3 close",
	"type to filter · ↑↓ select · Enter open · Esc back · F3 close",
	"↑↓ select · Enter open · Esc back · F3 close",
}

// finderWindow is the quick open's window, ww by hh: the field, the files
// the query keeps, the footer.
func finderWindow(ui UI, ww, hh int) []line {
	v := ui.Finder
	inner := ww - 4
	var right line
	if v.State == "" {
		n := grouped(v.Total) + " files"
		if v.Truncated {
			n = "first " + n
		}
		right = decoLine(n, stGrey)
	}
	body := []line{{deco("› ", stTitle), deco(v.Query, stText), deco("▌", stText)}}
	rows := hh - 4
	switch {
	case v.State != "":
		st := stGrey
		if v.Failed {
			st = stAttention
		}
		body = append(body, decoLine(v.State, st))
	case len(v.Files) == 0 && v.Total == 0:
		body = append(body, decoLine("no files in the project", stGrey))
	case len(v.Files) == 0:
		body = append(body, decoLine("no file matches", stGrey))
	default:
		top := ListTop(v.Top, v.Sel, rows, len(v.Files))
		for i := top; i < top+rows && i < len(v.Files); i++ {
			body = append(body, finderRow(v.Files[i], i == v.Sel, inner))
		}
	}
	for len(body) < rows+1 {
		body = append(body, nil)
	}
	var count line
	if v.State == "" && len(v.Files) > 0 {
		count = decoLine(fmt.Sprintf("%s of %s", grouped(v.Sel+1), grouped(len(v.Files))), stGrey)
	}
	footer := spread(windowLeft(state.New(), ui, finderKeys, inner-count.width()-1), count, inner)
	body = append(body[:rows+1], footer)
	return frameIn(frameBox(line{deco("Quick open", stTitle)}, right, body, ww, hh, stFileFrame))
}

// finderRow is a file of the quick open, w cells: its folder grey, its name
// in the text colour; the selected one on the cursor row's background.
func finderRow(p string, sel bool, w int) line {
	p = elide(p, w-2)
	dir, name := "", p
	if i := strings.LastIndex(p, "/"); i >= 0 {
		dir, name = p[:i+1], p[i+1:]
	}
	if sel {
		return line{deco("▸ "+dir+name, stTreeCursor)}.fit(w).restyle(stTreeCursor)
	}
	return line{deco("  "+dir, stGrey), deco(name, stText)}
}

// restyle gives every span of l the style st.
func (l line) restyle(st *lipgloss.Style) line {
	for i := range l {
		l[i].st = st
	}
	return l
}

// FileTarget is the file a conversation entry names, and where to show it
// (plan 17, Opening).
type FileTarget struct {
	Path   string
	Opened string
	Line   int      // the cursor line, 1-based; 0 the first
	Find   []string // texts whose first place in the file is marked, in order
	Mark   [2]int   // the lines marked, 1-based, first and last; 0 none
}

// EntryFile is the file entry e names: a tool entry whose call has a path.
func EntryFile(s *state.State, e state.Entry) (FileTarget, bool) {
	if e.Role != "tool" {
		return FileTarget{}, false
	}
	tc := s.Tools[e.MessageID]
	if tc == nil {
		return FileTarget{}, false
	}
	return ToolFile(tc.Name, tc.Input)
}

// ToolFile is the file a tool call's input names.
func ToolFile(name string, input json.RawMessage) (FileTarget, bool) {
	var in map[string]json.RawMessage
	if json.Unmarshal(input, &in) != nil {
		return FileTarget{}, false
	}
	str := func(k string) string {
		var v string
		_ = json.Unmarshal(in[k], &v)
		return v
	}
	var t FileTarget
	switch name {
	case "fileChange":
		var cs []struct {
			Path string `json:"path"`
			Diff string `json:"diff"`
		}
		if json.Unmarshal(in["changes"], &cs) != nil || len(cs) == 0 {
			return FileTarget{}, false
		}
		t = FileTarget{Path: cs[0].Path, Opened: "changed by fileChange"}
		t.Line, t.Mark = diffMark(cs[0].Diff)
	case "Read":
		t = FileTarget{Path: str("file_path"), Opened: "read by Read"}
		var n float64
		if json.Unmarshal(in["offset"], &n) == nil && n > 0 {
			t.Line = int(n)
		} else if k, err := strconv.Atoi(str("offset")); err == nil && k > 0 {
			t.Line = k
		}
	case "Edit", "MultiEdit":
		t = FileTarget{Path: str("file_path"), Opened: "edited by " + name}
		var edits []struct {
			Old string `json:"old_string"`
			New string `json:"new_string"`
		}
		if name == "Edit" {
			edits = append(edits, struct {
				Old string `json:"old_string"`
				New string `json:"new_string"`
			}{str("old_string"), str("new_string")})
		} else {
			_ = json.Unmarshal(in["edits"], &edits)
		}
		for _, e := range edits {
			t.Find = append(t.Find, e.New)
		}
		for _, e := range edits {
			t.Find = append(t.Find, e.Old)
		}
	case "Write":
		t = FileTarget{Path: str("file_path"), Opened: "written by Write"}
	default:
		for _, k := range []string{"file_path", "notebook_path", "path"} {
			if p := str(k); p != "" {
				t = FileTarget{Path: p, Opened: "named by " + name}
				break
			}
		}
	}
	return t, t.Path != ""
}

// diffMark is a change's cursor line and the first added lines of its
// first hunk, 1-based. A diff with no hunk is an added file's content: all
// of it is added, from line 1.
func diffMark(diff string) (int, [2]int) {
	if diff == "" {
		return 1, [2]int{}
	}
	lines := strings.Split(strings.TrimRight(diff, "\n"), "\n")
	hunk := -1
	for i, l := range lines {
		if strings.HasPrefix(l, "@@") {
			hunk = i
			break
		}
		if strings.HasPrefix(l, "diff --git") || strings.HasPrefix(l, "--- ") {
			hunk = -2
		}
	}
	switch hunk {
	case -1:
		return 1, [2]int{1, len(lines)}
	case -2:
		return 1, [2]int{}
	}
	// "@@ -a,b +c,d @@": the new file's lines start at c.
	c := 1
	if i := strings.Index(lines[hunk], " +"); i >= 0 {
		rest := lines[hunk][i+2:]
		if j := strings.IndexAny(rest, ", "); j >= 0 {
			rest = rest[:j]
		}
		if k, err := strconv.Atoi(rest); err == nil {
			c = k
		}
	}
	n, from, to := c, 0, 0
loop:
	for _, l := range lines[hunk+1:] {
		kind := byte(' ')
		if l != "" {
			kind = l[0]
		}
		switch kind {
		case '@':
			break loop
		case '+':
			if from == 0 {
				from = n
			}
			to = n
			n++
		case '-':
			if from > 0 {
				break loop
			}
		case '\\':
		default:
			if from > 0 {
				break loop
			}
			n++
		}
	}
	if from == 0 {
		return max(c, 1), [2]int{}
	}
	return from, [2]int{from, to}
}

// Resolve is the target's cursor line and marked lines, [from, to), 0-based,
// in a file of lines: the first text of Find found, else Mark, else Line.
func (t FileTarget) Resolve(lines []string) (int, [2]int) {
	last := max(len(lines)-1, 0)
	if len(t.Find) > 0 {
		text := strings.Join(lines, "\n")
		for _, f := range t.Find {
			f = strings.TrimRight(strings.ReplaceAll(f, "\r", ""), "\n")
			if f == "" {
				continue
			}
			if k := strings.Index(text, f); k >= 0 {
				from := strings.Count(text[:k], "\n")
				return from, [2]int{from, from + strings.Count(f, "\n") + 1}
			}
		}
	}
	if t.Mark[0] > 0 && t.Mark[0] <= len(lines) {
		from := t.Mark[0] - 1
		return from, [2]int{from, min(t.Mark[1], len(lines))}
	}
	return min(max(t.Line-1, 0), last), [2]int{}
}

// FileCodeWidth is how many cells of code a row of the file window shows.
func FileCodeWidth(ui UI) int {
	_, _, ww, _ := FileRect(ui.Width, ui.Height, fileFull(ui))
	return max(ww-4-len(strconv.Itoa(len(ui.File.Lines)))-2, 1)
}

// FileColumn is the cell column byte off of text starts at, as the window
// draws it: tabs to the next multiple of TabWidth, controls as their marks.
func FileColumn(text string, off int) int {
	col := 0
	for i, r := range text {
		if i >= off {
			break
		}
		if r == '\t' {
			col += TabWidth - col%TabWidth
		} else {
			col += cellAt(text, i, r).w
		}
	}
	return col
}
