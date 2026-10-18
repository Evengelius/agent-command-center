package app

import (
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"

	"github.com/Evengelius/agent-command-center/internal/input"
	"github.com/Evengelius/agent-command-center/internal/render"
)

// The file viewer and the quick open (plan 17), opened from the Explorer
// (explorer.go, amendment 12): the file is read, coloured
// and searched in commands, each result carrying the generation it was
// asked under, so that one for a file since closed or replaced is dropped.

// Colouring limits (plan 17, Highlighting): a file up to wholeColour bytes
// is coloured whole, a larger one colourMargin lines either side of the view.
const (
	wholeColour  = 1 << 20
	colourMargin = 1000
	shiftCells   = 8 // ←→ shift the code by this many cells
)

// Messages.
type (
	fileMsg struct {
		gen    int
		ft     input.FileText
		err    error
		cursor int
		mark   [2]int
	}
	hlMsg struct {
		gen int
		hl  render.Highlighted
	}
	searchMsg struct {
		gen, search int
		query       string
		matches     []render.FileMatch
		capped      bool
	}
	filesMsg struct {
		gen       int
		files     []string
		truncated bool
		err       error
	}
)

// fileRoot is the project the viewer reads in: the chosen directory, or in
// a mock run the mock file's folder.
func (m *Model) fileRoot() string {
	root := m.s.Context.Project
	if m.s.Context.Mock {
		root = filepath.Dir(root)
	}
	if a, err := filepath.Abs(root); err == nil {
		root = a
	}
	return root
}

// f3 opens the file the entry or the request on screen names, else the
// Explorer (plan 17, Opening; amendment 12).
func (m *Model) f3() []tea.Cmd {
	conv := m.s.Conversation
	switch {
	case m.ui.View != "" && m.s.Approvals[m.ui.View] != nil:
		ap := m.s.Approvals[m.ui.View]
		if t, ok := render.ToolFile(ap.Tool, ap.Input); ok {
			return m.openFile(t)
		}
	case m.ui.ViewEntry > 0 && m.ui.ViewEntry <= len(conv):
		if t, ok := render.EntryFile(m.s, conv[m.ui.ViewEntry-1]); ok {
			return m.openFile(t)
		}
	case m.ui.Entry > 0 && m.ui.Entry <= len(conv) && m.ui.Focus == "B" && m.ui.Card == "":
		if t, ok := render.EntryFile(m.s, conv[m.ui.Entry-1]); ok {
			return m.openFile(t)
		}
	}
	return m.openExplorer()
}

// openFile shows the target's file: read in a command, the window saying
// so meanwhile.
func (m *Model) openFile(t render.FileTarget) []tea.Cmd {
	m.fileGen++
	m.closeFinder()
	m.ui.File = render.FileView{Open: true, Full: m.ui.File.Full, Path: t.Path, Opened: t.Opened, State: "reading…", Match: -1}
	m.dropFileSelection()
	gen, root := m.fileGen, m.fileRoot()
	return []tea.Cmd{func() tea.Msg {
		ft, err := input.ReadFile(root, t.Path)
		msg := fileMsg{gen: gen, ft: ft, err: err}
		if err == nil && !ft.Binary {
			msg.cursor, msg.mark = t.Resolve(ft.Lines)
		}
		return msg
	}}
}

// fileRead shows a file read, or why it is not shown.
func (m *Model) fileRead(msg fileMsg) []tea.Cmd {
	f := &m.ui.File
	if msg.gen != m.fileGen || !f.Open {
		return nil
	}
	if msg.err != nil {
		f.State, f.Failed = msg.err.Error(), true
		return nil
	}
	f.Path, f.Rel = msg.ft.Path, msg.ft.Rel
	if msg.ft.Binary {
		f.State = "binary file, not shown · " + input.SizeText(msg.ft.Size)
		return nil
	}
	f.State, f.Lines, f.Cursor, f.Mark = "", msg.ft.Lines, msg.cursor, msg.mark
	m.fileBytes = msg.ft.Size
	f.Top = max(f.Cursor-render.FileRows(m.ui)/3, 0) // some context above the place shown
	f.Windowed = m.fileBytes > wholeColour
	return m.colour()
}

// colour asks for the colours the view lacks: the whole file once, or a
// window around the view whenever it leaves the one coloured.
func (m *Model) colour() []tea.Cmd {
	f := &m.ui.File
	if !f.Open || f.State != "" || len(f.Lines) == 0 || f.Colouring {
		return nil
	}
	from, to := 0, len(f.Lines)
	if f.Windowed {
		rows := render.FileRows(m.ui)
		top := render.FileTop(*f, rows)
		if h := f.Hl; len(h.Lines) > 0 && top >= h.From && min(top+rows, len(f.Lines)) <= h.From+len(h.Lines) {
			return nil
		}
		from, to = max(top-colourMargin, 0), min(top+rows+colourMargin, len(f.Lines))
	} else if f.Hl.Lang != "" {
		return nil
	}
	f.Colouring = true
	gen, name, lines, hl := m.fileGen, f.Path, f.Lines, m.opt.Highlight
	return []tea.Cmd{func() tea.Msg { return hlMsg{gen, hl(name, lines, from, to)} }}
}

// coloured takes a colouring, then asks for the next if the view moved on.
func (m *Model) coloured(msg hlMsg) []tea.Cmd {
	if msg.gen != m.fileGen || !m.ui.File.Open {
		return nil
	}
	m.ui.File.Hl, m.ui.File.Colouring = msg.hl, false
	return m.colour()
}

// closeFile closes the viewer; a read or colouring under way is dropped.
func (m *Model) closeFile() {
	m.fileGen++
	m.ui.File = render.FileView{Full: m.ui.File.Full}
	m.dropFileSelection()
}

// dropFileSelection clears a selection in the window.
func (m *Model) dropFileSelection() {
	if m.ui.Sel.From.Panel == 'F' {
		m.ui.Sel = render.Selection{}
	}
	if m.drag.panel == 'F' {
		m.drag.on = false
	}
}

// fileKey handles a key while the viewer is open: all but the global ones
// are its own (plan 17, Keys).
func (m *Model) fileKey(k tea.KeyPressMsg) (cmds []tea.Cmd, done bool) {
	s := k.String()
	switch s {
	case "f1", "f4", "f12", "ctrl+c", "ctrl+shift+c", "ctrl+q":
		return nil, false
	}
	f := &m.ui.File
	if f.Field != 0 {
		return m.fieldKey(k), true
	}
	if k.Text == "?" {
		return nil, false // the key overlay
	}
	rows := render.FileRows(m.ui)
	switch s {
	case "esc":
		if !m.ui.Sel.Empty() {
			m.ui.Sel = render.Selection{}
			return nil, true
		}
		return m.backToExplorer(), true // one level back (amendment 12)
	case "f2":
		m.closeWindows()
		return m.focusCard("", true), true
	case "f3":
		m.closeWindows()
		return nil, true
	case "f":
		f.Full = !f.Full
	case "/":
		f.Field, f.Typed = '/', f.Query
	case "g":
		if len(f.Lines) > 0 {
			f.Field, f.Typed = 'g', ""
		}
	case "n", "N":
		if len(f.Matches) > 0 {
			step := 1
			if s == "N" {
				step = len(f.Matches) - 1
			}
			m.showMatch((max(f.Match, 0) + step) % len(f.Matches))
		}
	case "left":
		f.Left = max(f.Left-shiftCells, 0)
	case "right":
		f.Left += shiftCells
	case "up":
		m.moveCursor(-1, 0)
	case "down":
		m.moveCursor(1, 0)
	case "pgup":
		m.moveCursor(-rows, -rows)
	case "pgdown":
		m.moveCursor(rows, rows)
	case "home", "ctrl+home":
		m.moveCursor(-len(f.Lines), 0)
		f.Left = 0
	case "end", "ctrl+end":
		m.moveCursor(len(f.Lines), 0)
	}
	return m.colour(), true
}

// moveCursor moves the cursor d lines and the view dt lines, kept in the
// file, the cursor in view.
func (m *Model) moveCursor(d, dt int) {
	f := &m.ui.File
	if len(f.Lines) == 0 {
		return
	}
	rows := render.FileRows(m.ui)
	f.Cursor = min(max(f.Cursor+d, 0), len(f.Lines)-1)
	f.Top = render.FileTop(render.FileView{Top: max(render.FileTop(*f, rows)+dt, 0), Cursor: f.Cursor, Lines: f.Lines}, rows)
}

// fieldKey types in the search or go-to-line field.
func (m *Model) fieldKey(k tea.KeyPressMsg) []tea.Cmd {
	f := &m.ui.File
	switch k.String() {
	case "esc":
		f.Field, f.Typed = 0, ""
	case "enter":
		field, text := f.Field, f.Typed
		f.Field, f.Typed = 0, ""
		if field == '/' {
			return m.search(text)
		}
		if n, err := strconv.Atoi(strings.TrimSpace(text)); err == nil && len(f.Lines) > 0 {
			f.Cursor = min(max(n-1, 0), len(f.Lines)-1)
			f.Top = max(f.Cursor-render.FileRows(m.ui)/2, 0)
			f.Left = 0
			return m.colour()
		}
	case "backspace":
		if _, n := utf8.DecodeLastRuneInString(f.Typed); n > 0 {
			f.Typed = f.Typed[:len(f.Typed)-n]
		}
	default:
		m.typeField(k.Text)
	}
	return nil
}

// typeField adds text to the field: one line, digits only to go to a line.
func (m *Model) typeField(text string) {
	f := &m.ui.File
	text = firstLine(text)
	if f.Field == 'g' {
		text = strings.Map(func(r rune) rune {
			if r >= '0' && r <= '9' {
				return r
			}
			return -1
		}, text)
	}
	f.Typed += text
}

// search finds q in the file in a command: case ignored unless q has a
// capital, at most render.MaxMatches matches.
func (m *Model) search(q string) []tea.Cmd {
	f := &m.ui.File
	f.Query, f.Matches, f.Match, f.Capped, f.Searching = q, nil, -1, false, q != ""
	m.searchGen++
	if q == "" {
		return nil
	}
	gen, search, lines := m.fileGen, m.searchGen, f.Lines
	return []tea.Cmd{func() tea.Msg {
		ms, capped := findAll(lines, q)
		return searchMsg{gen, search, q, ms, capped}
	}}
}

// findAll is every match of q in lines, in order, and whether it stopped
// at the limit.
func findAll(lines []string, q string) ([]render.FileMatch, bool) {
	pat := regexp.QuoteMeta(q)
	if q == strings.ToLower(q) {
		pat = "(?i)" + pat
	}
	re := regexp.MustCompile(pat)
	var out []render.FileMatch
	for i, l := range lines {
		for _, loc := range re.FindAllStringIndex(l, -1) {
			if len(out) == render.MaxMatches {
				return out, true
			}
			out = append(out, render.FileMatch{Line: i, Off: loc[0], N: loc[1] - loc[0]})
		}
	}
	return out, false
}

// searched shows a search's matches, the first at or after the cursor.
func (m *Model) searched(msg searchMsg) []tea.Cmd {
	f := &m.ui.File
	if msg.gen != m.fileGen || msg.search != m.searchGen || !f.Open {
		return nil
	}
	f.Matches, f.Capped, f.Searching = msg.matches, msg.capped, false
	if len(f.Matches) == 0 {
		return nil
	}
	i := sort.Search(len(f.Matches), func(j int) bool { return f.Matches[j].Line >= f.Cursor })
	m.showMatch(i % len(f.Matches))
	return m.colour()
}

// showMatch makes match i the current one, its line the cursor's and its
// text in view.
func (m *Model) showMatch(i int) {
	f := &m.ui.File
	f.Match = i
	mt := f.Matches[i]
	m.moveCursor(mt.Line-f.Cursor, 0)
	cw := render.FileCodeWidth(m.ui)
	c0 := render.FileColumn(f.Lines[mt.Line], mt.Off)
	c1 := render.FileColumn(f.Lines[mt.Line], mt.Off+mt.N)
	if c0 < f.Left || c1 > f.Left+cw {
		f.Left = max(c0-cw/3, 0)
	}
}

// openFinder opens the quick open, from the Explorer, and lists the
// project in a command.
func (m *Model) openFinder() []tea.Cmd {
	m.finderGen++
	m.ui.Finder = render.FinderView{Open: true, State: "listing…"}
	m.finderAll = nil
	m.dropFileSelection()
	gen, root := m.finderGen, m.fileRoot()
	return []tea.Cmd{func() tea.Msg {
		files, truncated, err := input.ListFiles(root)
		return filesMsg{gen, files, truncated, err}
	}}
}

// closeFinder closes the quick open; a listing under way is dropped.
func (m *Model) closeFinder() {
	m.finderGen++
	m.ui.Finder, m.finderAll = render.FinderView{}, nil
}

// listed shows the project's files.
func (m *Model) listed(msg filesMsg) {
	v := &m.ui.Finder
	if msg.gen != m.finderGen || !v.Open {
		return
	}
	if msg.err != nil {
		v.State, v.Failed = "the project cannot be listed: "+msg.err.Error(), true
		return
	}
	m.finderAll = msg.files
	v.State, v.Total, v.Truncated = "", len(msg.files), msg.truncated
	m.filter()
}

// filter keeps the files the query matches, best first.
func (m *Model) filter() {
	v := &m.ui.Finder
	v.Files, v.Sel, v.Top = filterFiles(m.finderAll, v.Query), 0, 0
}

// filterFiles keeps the paths holding q's characters in order, case
// ignored: those whose name holds them first, then the shortest.
func filterFiles(all []string, q string) []string {
	if q == "" {
		return all
	}
	q = strings.ToLower(q)
	type hit struct {
		p    string
		name bool
	}
	var hits []hit
	for _, p := range all {
		lp := strings.ToLower(p)
		if !subsequence(lp, q) {
			continue
		}
		hits = append(hits, hit{p, subsequence(lp[strings.LastIndex(lp, "/")+1:], q)})
	}
	sort.SliceStable(hits, func(i, j int) bool {
		if hits[i].name != hits[j].name {
			return hits[i].name
		}
		return len(hits[i].p) < len(hits[j].p)
	})
	out := make([]string, len(hits))
	for i, h := range hits {
		out[i] = h.p
	}
	return out
}

// subsequence reports whether s holds q's runes in order.
func subsequence(s, q string) bool {
	for _, r := range q {
		i := strings.IndexRune(s, r)
		if i < 0 {
			return false
		}
		s = s[i+utf8.RuneLen(r):]
	}
	return true
}

// finderKey handles a key while the quick open is open; Esc goes back to
// the Explorer under it.
func (m *Model) finderKey(k tea.KeyPressMsg) (cmds []tea.Cmd, done bool) {
	s := k.String()
	switch s {
	case "f1", "f4", "f12", "ctrl+c", "ctrl+shift+c", "ctrl+q":
		return nil, false
	}
	if k.Text == "?" {
		return nil, false // the key overlay; no Windows file name holds one
	}
	v := &m.ui.Finder
	rows := render.FileRows(m.ui)
	move := func(d int) {
		v.Sel = min(max(v.Sel+d, 0), max(len(v.Files)-1, 0))
		v.Top = render.ListTop(v.Top, v.Sel, rows, len(v.Files))
	}
	switch s {
	case "esc":
		m.closeFinder()
	case "f3":
		m.closeWindows()
	case "f2":
		m.closeWindows()
		return m.focusCard("", true), true
	case "enter":
		if v.Sel < len(v.Files) {
			return m.openFile(render.FileTarget{Path: v.Files[v.Sel], Opened: "quick open"}), true
		}
	case "up":
		move(-1)
	case "down":
		move(1)
	case "pgup":
		move(-rows)
	case "pgdown":
		move(rows)
	case "home", "ctrl+home":
		move(-len(v.Files))
	case "end", "ctrl+end":
		move(len(v.Files))
	case "backspace":
		if _, n := utf8.DecodeLastRuneInString(v.Query); n > 0 {
			v.Query = v.Query[:len(v.Query)-n]
			m.filter()
		}
	default:
		if k.Text != "" {
			v.Query += firstLine(k.Text)
			m.filter()
		}
	}
	return nil, true
}

// filePaste puts a paste in the quick open's query or the viewer's field;
// with neither, the window drops it, the Explorer too.
func (m *Model) filePaste(text string) {
	switch {
	case m.ui.Finder.Open:
		m.ui.Finder.Query += firstLine(text)
		m.filter()
	case m.ui.File.Field != 0:
		m.typeField(text)
	}
}

// fileMouse handles the mouse while a window is open: the wheel scrolls it,
// a press in the file starts a selection, nothing outside it; the Explorer
// in front has its own.
func (m *Model) fileMouse(msg tea.MouseMsg) []tea.Cmd {
	if !m.ui.File.Open && !m.ui.Finder.Open {
		return m.explorerMouse(msg)
	}
	mo := msg.Mouse()
	switch msg.(type) {
	case tea.MouseWheelMsg:
		d := wheelRows
		if mo.Button == tea.MouseWheelUp {
			d = -wheelRows
		} else if mo.Button != tea.MouseWheelDown {
			return nil
		}
		if v := &m.ui.Finder; v.Open {
			v.Sel = min(max(v.Sel+d, 0), max(len(v.Files)-1, 0))
			v.Top = render.ListTop(v.Top, v.Sel, render.FileRows(m.ui), len(v.Files))
			return nil
		}
		m.scrollFile(d)
		return m.colour()
	case tea.MouseClickMsg:
		if mo.Button == tea.MouseLeft && !m.ui.Finder.Open && m.screen.PanelAt(mo.X, mo.Y) == 'F' {
			m.press(mo)
		}
	case tea.MouseMotionMsg, tea.MouseReleaseMsg:
		if !m.drag.on || m.drag.panel != 'F' {
			return nil
		}
		m.drag.moved = m.drag.moved || mo.X != m.drag.x || mo.Y != m.drag.y
		if m.drag.moved {
			// Past the code rows, the file scrolls under the selection.
			_, y, _, _ := render.FileRect(m.ui.Width, m.ui.Height, m.ui.File.Full)
			if mo.Y < y+2 {
				m.scrollFile(-1)
			} else if mo.Y > y+1+render.FileRows(m.ui) {
				m.scrollFile(1)
			}
		}
		m.extend(mo)
		if _, ok := msg.(tea.MouseReleaseMsg); ok {
			m.drag.on = false
			return append(m.copySelection(), m.colour()...)
		}
		return m.colour()
	}
	return nil
}

// scrollFile moves the view d lines, the cursor dragged into it.
func (m *Model) scrollFile(d int) {
	f := &m.ui.File
	if len(f.Lines) == 0 {
		return
	}
	rows := render.FileRows(m.ui)
	top := min(max(render.FileTop(*f, rows)+d, 0), max(len(f.Lines)-rows, 0))
	f.Top, f.Cursor = top, min(max(f.Cursor, top), min(top+rows, len(f.Lines))-1)
}
