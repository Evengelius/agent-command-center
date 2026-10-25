package app

import (
	"bytes"
	"errors"
	"fmt"
	"path/filepath"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/Evengelius/agent-command-center/internal/diff"
	"github.com/Evengelius/agent-command-center/internal/input"
	"github.com/Evengelius/agent-command-center/internal/render"
)

// Source Control (plan 44, brief amendment 45): git's status read in
// commands, one at a time, when its panel shows, on "r", after a tool call
// or a turn ends, and while a panel or a diff shows every few seconds
// (design D3); a file's two sides read and compared in a command, a newer
// request dropping an older result (designs D6 and D7). Nothing here
// writes to git (design D10).

// Messages.
type (
	snapshotMsg struct {
		gen  int
		snap input.GitSnapshot
		err  error
		took time.Duration
	}
	pollMsg   struct{ gen int }
	folderMsg struct {
		gen       int
		rel       string
		files     []string
		truncated bool
		err       error
	}
	diffMsg struct {
		gen     int
		path    string
		staged  bool
		pair    input.DiffPair
		res     diff.Result
		err     error
		refresh bool
		// stamp is the sides' stamp (sideStamp), same that it is the one
		// compared last: nothing was compared again.
		stamp string
		same  bool
		took  time.Duration
	}
)

// The refresh's pace (design D3): every pollEvery while a panel or a diff
// shows, ten times a slow read's duration, at most pollMost.
const (
	pollEvery = 3 * time.Second
	pollMost  = 30 * time.Second
)

// diffContext is the lines of context around a change (design D7), and
// changeLead the rows n and N show above a change: its context and the
// hunk's header.
const (
	diffContext = 3
	changeLead  = diffContext + 1
)

// untrackedMost is how many files an untracked folder lists (design D4).
const untrackedMost = 1000

// sourceGit is the git Source Control asks: the options' when it reads
// snapshots, else none.
func (m *Model) sourceGit() (input.SourceGit, bool) {
	sg, ok := m.git().(input.SourceGit)
	return sg, ok
}

// watching says a status is to be kept current: a panel or a diff shows.
func (m *Model) watching() bool { return render.PanelShown(m.ui) || m.ui.Diff.Open }

// readSnapshot asks git's status in a command, unless one is under way:
// then one more is asked once it answers.
func (m *Model) readSnapshot() tea.Cmd {
	sg, ok := m.sourceGit()
	if !ok {
		return nil
	}
	if m.statusBusy {
		m.statusAgain = true
		return nil
	}
	m.statusBusy, m.statusAgain = true, false
	m.statusGen++
	gen, root, now := m.statusGen, m.fileRoot(), m.opt.Now
	return func() tea.Msg {
		start := now()
		snap, err := sg.Snapshot(root)
		return snapshotMsg{gen: gen, snap: snap, err: err, took: now().Sub(start)}
	}
}

// snapshotRead takes a status read: both views updated from it, the next
// read asked or timed, and the open diff compared again.
func (m *Model) snapshotRead(msg snapshotMsg) []tea.Cmd {
	if msg.gen != m.statusGen {
		return nil
	}
	m.statusBusy, m.statusTook = false, msg.took
	m.applySnapshot(msg.snap, msg.err)
	var cmds []tea.Cmd
	if m.statusAgain {
		return append(cmds, m.readSnapshot())
	}
	cmds = append(cmds, m.armPoll())
	if m.ui.Diff.Open && msg.err == nil {
		cmds = append(cmds, m.compareOpen(true))
	}
	return cmds
}

// armPoll times the next read while a panel or a diff shows, ten times the
// last read's and comparison's time apart when that is longer; a later one
// replaces it.
func (m *Model) armPoll() tea.Cmd {
	if !m.watching() {
		return nil
	}
	m.pollGen++
	every := min(max(pollEvery, 10*(m.statusTook+m.diffTook)), pollMost)
	return m.opt.After(every, pollMsg{gen: m.pollGen})
}

// polled reads the status again when the poll is still the last one timed
// and a panel or a diff shows.
func (m *Model) polled(msg pollMsg) []tea.Cmd {
	if msg.gen != m.pollGen || !m.watching() {
		return nil
	}
	return []tea.Cmd{m.readSnapshot()}
}

// sourceEvent asks a read after a session event that may have changed the
// project's files: a tool call's end or a turn's.
func (m *Model) sourceEvent() []tea.Cmd {
	if !m.watching() {
		return nil
	}
	return []tea.Cmd{m.readSnapshot()}
}

// applySnapshot puts a status read in both views (design D2): the
// Explorer's markers while it shows, Source Control's lists and notes.
func (m *Model) applySnapshot(snap input.GitSnapshot, err error) {
	if m.ui.Explorer.Open {
		m.gitRead(statusMsg{gen: m.explorerGen, marks: snap.Marks, err: err})
	}
	v := &m.ui.Source
	v.Note, v.Failed = "", false
	switch {
	case err == nil:
		s := snap
		m.snap = &s
	case errors.As(err, new(input.NotRepoError)):
		m.snap = nil
		v.Note, v.Repo, v.Branch, v.Entries = "No Git repository found for this project.", "", "", nil
		return
	case errors.Is(err, input.ErrNoGit):
		m.snap = nil
		v.Note, v.Repo, v.Branch, v.Entries = "git not found", "", "", nil
		return
	default:
		v.Note, v.Failed = "git: "+err.Error(), true
		return
	}
	v.Repo = filepath.Base(snap.Top)
	b := snap.Branch
	switch {
	case b.NoCommits:
		v.Branch = "no commits yet on " + b.Name
	case b.Detached:
		v.Branch = "detached at " + b.Head
	default:
		v.Branch = b.Name
	}
	if snap.Untracked {
		// The project's folder untracked in a larger repository: git lists
		// none of its files apart, which is no "No changes" (design D11).
		v.Note = "The project's folder is untracked in this repository: git lists none of its files apart."
	}
	path := ""
	if rows := render.SourceRows(*v); len(rows) > 0 {
		if i := render.SourceCursor(rows, v.Cursor); i >= 0 {
			path = rows[i].Path // the cursor's file, kept if it moves to another section
		}
	}
	v.Entries = make([]render.SourceEntry, 0, len(snap.Entries))
	untracked := map[string]bool{}
	for _, e := range snap.Entries {
		v.Entries = append(v.Entries, render.SourceEntry{X: e.X, Y: e.Y, Path: e.Path, Orig: e.Orig})
		if e.X == '?' && e.Y == '?' {
			untracked[e.Path] = true
		}
	}
	for p := range v.Folders {
		if !untracked[p] {
			delete(v.Folders, p) // no longer untracked: read again if it comes back
			delete(v.Opened, p)
		}
	}
	rows := render.SourceRows(*v)
	if i := render.SourceCursor(rows, v.Cursor); path != "" && (i < 0 || rows[i].Key != v.Cursor) {
		for _, r := range rows {
			if r.Path == path && r.Selectable() {
				v.Cursor = r.Key
				break
			}
		}
	}
}

// toggleSource shows Source Control, or hides it while shown: a click on its
// tile or on its "⌄" (design D1).
func (m *Model) toggleSource() []tea.Cmd {
	if m.ui.Source.Open {
		m.closeSource()
		return nil
	}
	return m.showSource()
}

// showSource shows Source Control in the docked place, the Explorer's state
// kept for the next F3, gives it the focus and reads git's status.
func (m *Model) showSource() []tea.Cmd {
	if m.ui.Explorer.Open {
		m.explorerGen++ // its reads under way are dropped
		m.ui.Explorer.Open = false
	}
	v := &m.ui.Source
	v.Open = true
	if v.Folded == nil {
		v.Folded, v.Opened, v.Folders = map[string]bool{}, map[string]bool{}, map[string]*render.SourceFolder{}
	}
	if m.snap == nil && v.Note == "" {
		v.Note = "reading…"
	}
	m.dropFileSelection()
	m.ui.Focus = "R"
	return []tea.Cmd{m.readSnapshot()}
}

// closeSource hides Source Control, its sections and cursor kept; zone B
// takes the focus back.
func (m *Model) closeSource() {
	if m.ui.Source.Open {
		m.ui.Source.Open = false
		m.ui.Source.Menu, m.ui.Source.MenuHot, m.ui.Source.Ask, m.ask = false, false, 0, nil // the dropdown and the question go with it (plan 45)
		if m.ui.Focus == "R" {
			m.ui.Focus = "B"
		}
	}
}

// closePanels hides whichever panel shows: the compact rule's.
func (m *Model) closePanels() {
	m.closeExplorer()
	m.closeSource()
}

// sourceKey handles a key while Source Control has the focus (design D5):
// its keys, the global ones passed on as beside the Explorer's panel.
func (m *Model) sourceKey(k tea.KeyPressMsg) (cmds []tea.Cmd, done bool) {
	// The commit controls first (plan 45): the dropdown, the question, the
	// field and the button take their keys and pass the dashboard's on.
	if cmds, done := m.commitKey(k); done {
		return cmds, true
	} else if m.ui.Source.Place != "" || m.ui.Source.Ask > 0 {
		return nil, false
	}
	s := k.String()
	switch s {
	case "f1", "f3", "f4", "f6", "shift+f6", "tab", "shift+tab", "alt+p", "f12", "ctrl+c", "ctrl+shift+c", "ctrl+o", "ctrl+r",
		"shift+pgup", "shift+pgdown", "ctrl+q", "F":
		return nil, false
	}
	if k.Text == "?" {
		return nil, false // the key overlay
	}
	v := &m.ui.Source
	rows := render.SourceRows(*v)
	cur := render.SourceCursor(rows, v.Cursor)
	room := render.SourceRoom(m.s, m.ui)
	switch s {
	case "esc":
		m.ui.Focus = "B" // the panel kept (design D5)
	case "f2":
		return m.focusCard("", true), true
	case "alt+v", "ctrl+v":
		if m.draftBeside() {
			return m.editorKey(k), true
		}
	case "r":
		return []tea.Cmd{m.readSnapshot()}, true
	case "up":
		if m.controlsShown() && (cur < 0 || render.SourceCursor(rows, "") == cur) {
			v.Place = render.PlaceBtn // up from the list's first row: the button (plan 45, design D3)
			break
		}
		m.sourceMove(rows, cur, -1, room)
	case "down":
		m.sourceMove(rows, cur, 1, room)
	case "pgup":
		m.sourceMove(rows, cur, -room, room)
	case "pgdown":
		m.sourceMove(rows, cur, room, room)
	case "home", "ctrl+home":
		m.sourceMove(rows, cur, -len(rows), room)
	case "end", "ctrl+end":
		m.sourceMove(rows, cur, len(rows), room)
	case "right":
		if cur >= 0 {
			return m.sourceOpenRow(rows[cur], true, false), true
		}
	case "left":
		if cur >= 0 {
			m.sourceFold(rows, cur)
		}
	case "enter":
		if cur >= 0 {
			return m.sourceOpenRow(rows[cur], false, true), true
		}
	}
	return nil, true
}

// sourceMove moves the cursor d selectable rows, kept in view.
func (m *Model) sourceMove(rows []render.SourceRow, cur, d, room int) {
	var sel []int
	at := 0
	for i, r := range rows {
		if r.Selectable() {
			if i == cur {
				at = len(sel)
			}
			sel = append(sel, i)
		}
	}
	if len(sel) == 0 {
		return
	}
	i := sel[min(max(at+d, 0), len(sel)-1)]
	v := &m.ui.Source
	v.Cursor = rows[i].Key
	v.Top = render.ListTop(v.Top, i, room, len(rows))
}

// sourceFold folds a section or an opened untracked folder; on a file, the
// cursor goes to its section's header.
func (m *Model) sourceFold(rows []render.SourceRow, cur int) {
	v := &m.ui.Source
	r := rows[cur]
	switch {
	case r.Kind == 'h' && r.Open:
		v.Folded[r.Section] = true
	case r.Kind == 'd' && r.Open:
		v.Opened[r.Path] = false
	default:
		for i := cur; i >= 0; i-- {
			if rows[i].Kind == 'h' {
				v.Cursor = rows[i].Key
				return
			}
		}
	}
}

// sourceOpenRow acts on a row: a header unfolds (and folds, by Enter), an
// untracked folder opens (and closes, by Enter), a file opens its diff,
// with Enter or a click (open).
func (m *Model) sourceOpenRow(r render.SourceRow, onlyOpen, open bool) []tea.Cmd {
	v := &m.ui.Source
	v.Cursor = r.Key
	switch r.Kind {
	case 'h':
		if v.Folded[r.Section] || !onlyOpen {
			v.Folded[r.Section] = !v.Folded[r.Section]
		}
		return nil
	case 'd':
		if v.Opened[r.Path] && !onlyOpen {
			v.Opened[r.Path] = false
			return nil
		}
		v.Opened[r.Path] = true
		if v.Folders[r.Path] == nil {
			return []tea.Cmd{m.readFolder(r.Path)}
		}
		return nil
	case 'f', 'u':
		if !open {
			return nil
		}
		return m.openDiff(r)
	}
	return nil
}

// readFolder lists an untracked folder's files in a command (design D4),
// those git ignores left out, as its status lists them apart.
func (m *Model) readFolder(rel string) tea.Cmd {
	gen, root := m.statusGen, m.fileRoot()
	var marks input.GitMarks
	if m.snap != nil {
		marks = m.snap.Marks
	}
	return func() tea.Msg {
		files, truncated, err := input.UntrackedFiles(root, rel, untrackedMost)
		kept := files[:0]
		for _, f := range files {
			if !marks.IsIgnored(f) {
				kept = append(kept, f)
			}
		}
		return folderMsg{gen: gen, rel: rel, files: kept, truncated: truncated, err: err}
	}
}

// folderRead keeps an untracked folder's files, or why there are none.
func (m *Model) folderRead(msg folderMsg) {
	v := &m.ui.Source
	if v.Folders == nil || !v.Opened[msg.rel] {
		return
	}
	d := &render.SourceFolder{Files: msg.files, Truncated: msg.truncated}
	if msg.err != nil {
		d.Note, d.Failed = msg.err.Error(), true
	} else if len(msg.files) == 0 {
		d.Note = "no files"
	}
	v.Folders[msg.rel] = d
}

// sourceClick is a click on the panel: its "⌄" hides it; a row takes the
// cursor and acts as Enter, a file opening its diff (design D5).
func (m *Model) sourceClick(item, x, y int) []tea.Cmd {
	m.ui.Sel = render.Selection{}
	if item == render.ItemPanelHide {
		m.closeSource()
		return nil
	}
	m.ui.Focus = "R"
	m.countClick(x, y)
	if cmds, done := m.commitClick(item, x, y); done {
		return cmds // the commit controls' (plan 45)
	}
	m.ui.Source.Place = "" // a click on the list gives it the keys
	rows := render.SourceRows(m.ui.Source)
	i := item - 1
	if i < 0 || i >= len(rows) || !rows[i].Selectable() {
		return nil
	}
	return m.sourceOpenRow(rows[i], false, true)
}

// sourceWheel moves the panel's cursor wheelRows rows, the pointer over it.
func (m *Model) sourceWheel(b tea.MouseButton) {
	v := &m.ui.Source
	rows := render.SourceRows(*v)
	cur := render.SourceCursor(rows, v.Cursor)
	switch b {
	case tea.MouseWheelUp:
		m.sourceMove(rows, cur, -wheelRows, render.SourceRoom(m.s, m.ui))
	case tea.MouseWheelDown:
		m.sourceMove(rows, cur, wheelRows, render.SourceRoom(m.s, m.ui))
	}
}

// sourceHoverAt is the panel's selectable row under the pointer plus one, 0
// off them (design D5).
func (m *Model) sourceHoverAt(x, y int) int {
	if m.screen == nil || !m.ui.Source.Open {
		return 0
	}
	p, item := m.screen.ItemAt(x, y)
	if p != render.SourcePanel || item < 1 {
		return 0
	}
	rows := render.SourceRows(m.ui.Source)
	if item > len(rows) || !rows[item-1].Selectable() {
		return 0
	}
	return item
}

// diffEntry is the git entry a row opens: its status's, or an untracked
// folder's file's.
func (m *Model) diffEntry(r render.SourceRow) (input.GitEntry, bool) {
	if m.snap == nil {
		return input.GitEntry{}, false
	}
	if r.Kind == 'u' {
		return input.GitEntry{X: '?', Y: '?', Path: r.Path, RepoPath: m.snap.Prefix + r.Path}, true
	}
	if r.Entry < 0 || r.Entry >= len(m.snap.Entries) || m.snap.Entries[r.Entry].Path != r.Path {
		return input.GitEntry{}, false
	}
	return m.snap.Entries[r.Entry], true
}

// openDiff opens a row's diff in zone B's place, the comparison of its
// section, and compares its sides in a command (designs D6 and D8).
func (m *Model) openDiff(r render.SourceRow) []tea.Cmd {
	e, ok := m.diffEntry(r)
	if !ok {
		return m.toast("the file changed: read again")
	}
	staged := r.Section == render.SectionStaged
	se := render.SourceEntry{X: e.X, Y: e.Y, Path: e.Path, Orig: e.Orig}
	m.ui.Diff = render.DiffView{Open: true, Path: e.Path, Orig: r.Orig, Staged: staged, Note: "comparing…",
		Both: len(render.Sections(se)) == 2, Conflict: render.Conflict(se)}
	m.diffEntryOpen, m.diffStamp = e, ""
	m.ui.Focus = "D"
	return []tea.Cmd{m.compareOpen(false)}
}

// compareOpen reads the open diff's sides in a command and compares them;
// a refresh whose sides have the stamp compared last compares nothing
// (design D3).
func (m *Model) compareOpen(refresh bool) tea.Cmd {
	sg, ok := m.sourceGit()
	if !ok || m.snap == nil {
		return nil
	}
	m.diffGen++
	gen, root, snap, e, staged, now := m.diffGen, m.fileRoot(), *m.snap, m.diffEntryOpen, m.ui.Diff.Staged, m.opt.Now
	last := ""
	if refresh {
		last = m.diffStamp
		for _, n := range snap.Entries {
			if n.Path == e.Path {
				e = n // its status now
			}
		}
		m.diffEntryOpen = e
	}
	return func() tea.Msg {
		start := now()
		pair, err := sg.Sides(root, snap, e, staged)
		msg := diffMsg{gen: gen, path: e.Path, staged: staged, pair: pair, err: err, refresh: refresh,
			stamp: sideStamp(pair.Old) + "|" + sideStamp(pair.New)}
		switch {
		case err == nil && last != "" && msg.stamp == last:
			msg.same = true
		case err == nil && textual(pair.Old) && textual(pair.New):
			msg.res = diff.Compare(pair.Old.Data, pair.New.Data, diffContext)
		}
		msg.took = now().Sub(start)
		return msg
	}
}

// textual says a side has lines to compare: missing (none) or read as text.
func textual(s input.DiffSide) bool { return !s.Binary && !s.LFS && !s.TooLarge }

// diffRead shows a comparison, or why there are no lines (design D6); a
// refresh whose sides did not change leaves the view as it is.
func (m *Model) diffRead(msg diffMsg) {
	v := &m.ui.Diff
	if msg.gen != m.diffGen || !v.Open || msg.path != v.Path || msg.staged != v.Staged {
		return
	}
	m.diffTook = msg.took
	if msg.same {
		return
	}
	m.diffStamp = msg.stamp
	if msg.err != nil {
		m.diffStamp = "" // compared again at the next read
	}
	top, left, wide := v.Top, v.Left, v.Wide
	nv := render.DiffView{Open: true, Path: v.Path, Orig: v.Orig, Staged: v.Staged, Both: v.Both, Wide: wide,
		OldName: msg.pair.Old.Name, NewName: msg.pair.New.Name, Conflict: msg.pair.Conflict}
	if nv.Conflict == "" {
		nv.Conflict = v.Conflict
	}
	if msg.pair.Old.Missing {
		nv.OldName = ""
	}
	if msg.pair.New.Missing {
		nv.NewName = ""
	}
	var refused *input.FileError
	switch {
	case errors.As(msg.err, &refused):
		// The viewer's refusals (a credential file, outside the project, not
		// a file), said as the viewer says them.
		nv.Note, nv.Failed = refused.Reason, true
	case msg.err != nil:
		nv.Note, nv.Failed = "git: "+msg.err.Error(), true
	case msg.pair.Old.Binary || msg.pair.New.Binary:
		nv.Note = fmt.Sprintf("Binary file: no textual diff (%s → %s)", sideSize(msg.pair.Old), sideSize(msg.pair.New))
	case msg.pair.Old.LFS != msg.pair.New.LFS:
		nv.Note = "Git LFS file: its content is kept outside git, no textual diff"
	case msg.pair.Old.TooLarge || msg.pair.New.TooLarge:
		nv.Note = fmt.Sprintf("Too large to compare (%s → %s, %s at most)", sideSize(msg.pair.Old), sideSize(msg.pair.New), input.SizeText(input.MaxFileSize))
	case msg.res.EOLOnly:
		nv.Note = "Only the line endings differ"
	case len(msg.res.Hunks) == 0:
		nv.Note = "No textual changes"
	}
	for _, h := range msg.res.Hunks {
		dh := render.DiffHunk{OldStart: h.OldStart, OldLines: h.OldLines, NewStart: h.NewStart, NewLines: h.NewLines,
			Lines: make([]render.DiffLine, len(h.Lines))}
		for i, l := range h.Lines {
			dh.Lines[i] = render.DiffLine{Kind: byte(l.Kind), Old: l.Old, New: l.New, Text: l.Text}
		}
		nv.Hunks = append(nv.Hunks, dh)
	}
	nv.Added, nv.Removed, nv.Whole = msg.res.Added, msg.res.Removed, msg.res.Whole
	// A side's missing line end is said where its last line is shown, as git
	// says it under that line.
	nv.NoNewlineOld = msg.res.NoNewlineOld && lastShown(nv.Hunks, lineCount(msg.pair.Old.Data), true)
	nv.NoNewlineNew = msg.res.NoNewlineNew && lastShown(nv.Hunks, lineCount(msg.pair.New.Data), false)
	if nv.Note == "" {
		nv.Layout = render.LayoutDiff(nv)
	}
	if msg.refresh {
		nv.Top, nv.Left = top, left
	}
	*v = nv
	m.clampDiff()
}

// lineCount is a text's lines, a last one with no line end counted.
func lineCount(data []byte) int {
	n := bytes.Count(data, []byte{'\n'})
	if len(data) > 0 && data[len(data)-1] != '\n' {
		n++
	}
	return n
}

// lastShown says the hunks show line n of a side, the old or the new.
func lastShown(hunks []render.DiffHunk, n int, old bool) bool {
	if len(hunks) == 0 || n == 0 {
		return false
	}
	for _, l := range hunks[len(hunks)-1].Lines {
		if old && l.Old == n || !old && l.New == n {
			return true
		}
	}
	return false
}

// sideStamp is what says a side changed: its blob, or the working file's
// size and time.
func sideStamp(s input.DiffSide) string {
	return fmt.Sprintf("%s:%s:%d:%d:%t", s.Name, s.ID, s.Size, s.ModTime.UnixNano(), s.Missing)
}

// sideSize is a side's size as the viewer says it, "nothing" for none.
func sideSize(s input.DiffSide) string {
	if s.Missing {
		return "nothing"
	}
	return input.SizeText(s.Size)
}

// diffInner is the diff box's inner width on this screen.
func (m *Model) diffInner() int { return render.DiffBox(m.s, m.ui).W - 4 }

// clampDiff keeps the diff's first row and its shift in range: the last
// row at the bottom at most, the longest line's end in view.
func (m *Model) clampDiff() {
	v := &m.ui.Diff
	w := m.diffInner()
	rows, _ := render.DiffRows(*v, w)
	v.Top = min(max(v.Top, 0), max(len(rows)-render.DiffRoom(m.s, m.ui), 0))
	v.Left = min(max(v.Left, 0), render.DiffMaxLeft(*v, w))
}

// diffViewKey handles a key while the diff has the focus (design D9).
func (m *Model) diffViewKey(k tea.KeyPressMsg) (cmds []tea.Cmd, done bool) {
	s := k.String()
	switch s {
	case "f1", "f2", "f3", "f4", "f6", "shift+f6", "shift+tab", "alt+p", "f12", "ctrl+c", "ctrl+shift+c", "ctrl+o", "ctrl+r",
		"shift+pgup", "shift+pgdown", "ctrl+q", "F":
		return nil, false
	}
	if k.Text == "?" {
		return nil, false
	}
	v := &m.ui.Diff
	room := render.DiffRoom(m.s, m.ui)
	rows, changes := render.DiffRows(*v, m.diffInner())
	switch s {
	case "esc":
		m.closeDiff()
	case "up":
		v.Top--
	case "down":
		v.Top++
	case "pgup":
		v.Top -= max(room-1, 1)
	case "pgdown":
		v.Top += max(room-1, 1)
	case "home", "ctrl+home":
		v.Top = 0
	case "end", "ctrl+end":
		v.Top = len(rows)
	case "left":
		v.Left -= 8
	case "right":
		v.Left += 8
	case "n":
		// The next change, under its hunk's header or the context before it.
		for _, c := range changes {
			if c-changeLead > v.Top {
				v.Top = c - changeLead
				break
			}
		}
	case "N":
		for i := len(changes) - 1; i >= 0; i-- {
			if c := changes[i]; c-changeLead < v.Top {
				v.Top = max(c-changeLead, 0)
				break
			}
		}
	case "tab":
		if !v.Both {
			return nil, false // the next zone, as outside the editor
		}
		v.Staged, v.Note, v.Layout, v.Hunks, v.Top, v.Left = !v.Staged, "comparing…", nil, nil, 0, 0
		e := m.diffEntryOpen
		sec := render.SectionChanges
		if v.Staged {
			sec = render.SectionStaged
		}
		v.Orig = ""
		if l := render.SourceLetter(render.SourceEntry{X: e.X, Y: e.Y}, sec); l == 'R' || l == 'C' {
			v.Orig = e.Orig // the rename's source, as the section's row names it
		}
		m.ui.Source.Cursor = "f:" + sec + ":" + e.Path // the panel's cursor on the row shown
		m.diffStamp = ""
		return []tea.Cmd{m.compareOpen(false)}, true
	case "f":
		v.Wide = !v.Wide
	case "r":
		// Git's status read again, and the file compared again with it
		// (design D9).
		m.diffStamp = ""
		return []tea.Cmd{m.readSnapshot(), m.compareOpen(false)}, true
	}
	m.clampDiff()
	return nil, true
}

// diffWheel scrolls the diff wheelRows rows, the pointer over it.
func (m *Model) diffWheel(b tea.MouseButton) {
	switch b {
	case tea.MouseWheelUp:
		m.ui.Diff.Top -= wheelRows
	case tea.MouseWheelDown:
		m.ui.Diff.Top += wheelRows
	}
	m.clampDiff()
}

// closeDiff closes the diff: zone B comes back as it was, the focus to the
// docked panel, its cursor on the file, or to zone B with no panel.
func (m *Model) closeDiff() {
	m.ui.Diff = render.DiffView{}
	m.diffGen++
	if m.ui.Focus == "D" {
		m.ui.Focus = "B"
		if render.PanelShown(m.ui) {
			m.ui.Focus = "R"
		}
	}
}
