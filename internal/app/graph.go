package app

import (
	"encoding/hex"
	"errors"
	"fmt"
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/Evengelius/agent-command-center/internal/diff"
	"github.com/Evengelius/agent-command-center/internal/input"
	"github.com/Evengelius/agent-command-center/internal/render"
)

// The Git Graph (plan 62, brief amendment 64; ui-design.md, Git Graph): a
// panel of the docked place, the history of the scope's references read a
// page at a time from fixed tips by one git process, a commit unfolded into
// its files, a file's diff in that commit in the diff viewer, the scope
// chosen in a popup (graphscope.go). Read only: git is asked for its
// references, its log, a commit's files and a file's two sides, nothing
// else (design D14).

// graphPage is how many commits a page reads, graphNear how close to the
// last row read the cursor or the view comes before the next page is asked
// (design D4).
const (
	graphPage = 200
	graphNear = 50
)

// graphGuard is the most commits the history keeps, a guard against a
// runaway (design D4); a variable so that tests can lower it.
var graphGuard = 1_000_000

// graphState is what the application keeps of the Git Graph beside its view.
type graphState struct {
	// scope is the history's (design D3); refs the references read last,
	// refsOK once read.
	scope  input.GraphScope
	refs   input.GraphRefs
	refsOK bool
	// tips are the commits the read under way started from (design D4).
	tips []string
	// reader is that read's git process, nil when none runs; gen its number,
	// an answer of an older read dropped; reading says a page was asked and
	// has not come; done that the history ended.
	reader  input.GraphReader
	gen     int
	reading bool
	done    bool
	// index finds a commit's row by its hash; authors shares a name's string
	// among its commits; labels are the references on each commit.
	index   map[[20]byte]int32
	authors map[string]string
	labels  map[[20]byte][]render.GraphLabel
	// lanes lay the read's commits out (design D9); parents is the room
	// graphEntry decodes a commit's parents in.
	lanes   *render.GraphLanes
	parents [][20]byte
	// unfolded are the commits unfolded, files their files read, both by
	// hash, for the session (designs D5 and D7).
	unfolded map[[20]byte]bool
	files    map[[20]byte]*render.GraphFiles
	// pinned says i keeps the cursor's card shown; messages are the
	// messages read for the cards, messageAsked and filesAsked the reads on
	// their way (design D13).
	pinned                   bool
	messages                 map[[20]byte]string
	messageAsked, filesAsked map[[20]byte]bool
	// keep is the cursor and the first row in view, by hash, to put back
	// once a history read again reaches them (design D7).
	keep graphPlace
	// refsGen numbers the references' reads, watchGen the watch's chain of
	// checks (design D4b).
	refsGen, watchGen int
	// keptDiff is Source Control's diff while the Git Graph shows, shown
	// again with Source Control (design D6).
	keptDiff render.DiffView
}

// graphPlace is a place in the history by hash: the cursor's commit, and
// its file's path when on one; the first row in view.
type graphPlace struct {
	set           bool
	cursor, first [20]byte
	file          string
}

type (
	// graphRefsMsg is the references read: for a scope's tips, and with
	// watch the watch's check.
	graphRefsMsg struct {
		gen, watch int
		refs       input.GraphRefs
		err        error
	}
	// graphPageMsg is a page of the history; first carries the reader its
	// read started; moved says the references moved before it was read,
	// refs as they are now (design D4b).
	graphPageMsg struct {
		gen     int
		reader  input.GraphReader
		first   bool
		commits []input.GraphCommit
		done    bool
		err     error
		moved   bool
		refs    input.GraphRefs
	}
	// graphFilesMsg is a commit's files read.
	graphFilesMsg struct {
		hash  [20]byte
		files input.CommitFiles
		err   error
	}
	// graphTickMsg is the watch's next check (design D4b).
	graphTickMsg struct{ gen int }
)

// graphGit is the Git Graph's git: the one the dashboard runs, when it reads
// a history.
func (m *Model) graphGit() (input.GraphGit, bool) {
	gg, ok := m.git().(input.GraphGit)
	return gg, ok
}

// toggleGraph shows the Git Graph, or hides it while shown: a click on its
// tile or on its "⌄", Alt+G (design D1).
func (m *Model) toggleGraph() []tea.Cmd {
	if m.ui.Graph.Open {
		m.closeGraph()
		return nil
	}
	return m.showGraph()
}

// showGraph shows the Git Graph in the docked place with the focus, the
// other panels' states kept, Source Control's diff kept for Source Control,
// and reads the references, then the history again (designs D1, D6 and D7).
func (m *Model) showGraph() []tea.Cmd {
	if m.ui.Explorer.Open {
		m.explorerGen++
		m.ui.Explorer.Open = false
	}
	m.ui.Source.Open = false
	m.dropSourcePopups()
	if m.ui.Diff.Open && m.ui.Diff.Commit == "" {
		m.graph.keptDiff = m.ui.Diff
		m.ui.Diff = render.DiffView{}
		m.diffGen++
	}
	g := &m.graph
	if g.index == nil {
		g.index, g.authors, g.labels = map[[20]byte]int32{}, map[string]string{}, map[[20]byte][]render.GraphLabel{}
		g.unfolded, g.files = map[[20]byte]bool{}, map[[20]byte]*render.GraphFiles{}
		g.messages, g.messageAsked, g.filesAsked = map[[20]byte]string{}, map[[20]byte]bool{}, map[[20]byte]bool{}
	}
	v := &m.ui.Graph
	v.Open = true
	if len(v.Rows) == 0 && v.State == "" {
		v.State = "reading…"
	}
	m.dropFileSelection()
	m.ui.Focus = "R"
	cmds := []tea.Cmd{m.readGraphRefs(0)}
	return append(cmds, m.watchGraph()...)
}

// closeGraph hides the Git Graph, its history and places kept: its read
// ends, git stopped, its scope popup and its diff close, Source Control's
// diff waits for Source Control; zone B takes the focus back.
func (m *Model) closeGraph() {
	if !m.ui.Graph.Open {
		return
	}
	m.ui.Graph.Open = false
	m.ui.Scope = render.ScopeView{}
	m.stopGraphRead()
	m.graph.watchGen++
	if m.ui.Diff.Open && m.ui.Diff.Commit != "" {
		m.closeDiff()
	}
	if m.ui.Focus == "R" {
		m.ui.Focus = "B"
	}
}

// stopGraphRead ends the history's read under way: git stopped and waited
// for, its pages to come dropped (design D4).
func (m *Model) stopGraphRead() {
	g := &m.graph
	if g.reader != nil {
		g.reader.Close()
		g.reader = nil
	}
	g.gen++
	g.reading = false
}

// restoreSourceDiff gives Source Control back the diff it had open when the
// Git Graph showed (design D6).
func (m *Model) restoreSourceDiff() {
	if k := m.graph.keptDiff; k.Open && !m.ui.Diff.Open {
		m.ui.Diff = k
	}
	m.graph.keptDiff = render.DiffView{}
}

// readGraphRefs reads the references in a command; watch is the watch's
// chain when the check is its own, 0 otherwise.
func (m *Model) readGraphRefs(watch int) tea.Cmd {
	gg, ok := m.graphGit()
	if !ok {
		return nil
	}
	m.graph.refsGen++
	gen, root := m.graph.refsGen, m.fileRoot()
	return func() tea.Msg {
		refs, err := gg.GraphRefs(root)
		return graphRefsMsg{gen: gen, watch: watch, refs: refs, err: err}
	}
}

// graphRecheck reads the references again while the panel shows, after the
// dashboard's own git operations (design D4b).
func (m *Model) graphRecheck() []tea.Cmd {
	if !m.ui.Graph.Open {
		return nil
	}
	if c := m.readGraphRefs(0); c != nil {
		return []tea.Cmd{c}
	}
	return nil
}

// watchGraph starts the watch's checks while the panel shows (design D4b):
// every GraphWatch, the references read again; none without a period.
func (m *Model) watchGraph() []tea.Cmd {
	m.graph.watchGen++
	if m.opt.GraphWatch <= 0 || m.opt.After == nil {
		return nil
	}
	return []tea.Cmd{m.opt.After(m.opt.GraphWatch, graphTickMsg{m.graph.watchGen})}
}

// graphTick is the watch's check: the references read again while the panel
// shows on the dashboard.
func (m *Model) graphTick(msg graphTickMsg) []tea.Cmd {
	if msg.gen != m.graph.watchGen || !m.ui.Graph.Open {
		return nil
	}
	if c := m.readGraphRefs(msg.gen); c != nil {
		return []tea.Cmd{c}
	}
	return nil
}

// graphRefsRead takes the references read (design D4b): the scope's tips
// from them, the labels and the scope row; tips that moved, or a first
// read, read the history again from them. The watch's check asks the next.
func (m *Model) graphRefsRead(msg graphRefsMsg) []tea.Cmd {
	g, v := &m.graph, &m.ui.Graph
	var cmds []tea.Cmd
	if msg.watch != 0 && msg.watch == g.watchGen && v.Open && m.opt.GraphWatch > 0 && m.opt.After != nil {
		cmds = append(cmds, m.opt.After(m.opt.GraphWatch, graphTickMsg{msg.watch}))
	}
	if msg.gen != g.refsGen || !v.Open {
		return cmds
	}
	if msg.err != nil {
		if msg.watch == 0 || len(v.Rows) == 0 {
			m.stopGraphRead()
			v.Rows, v.Commits = nil, nil
			v.State, v.Failed = graphProblem(msg.err)
		}
		return cmds
	}
	g.refs, g.refsOK = msg.refs, true
	if m.ui.Scope.Open {
		m.fillScope()
	}
	tips, gone := msg.refs.Tips(g.scope)
	v.ScopeNote = ""
	if len(gone) > 0 {
		if len(gone) == len(g.scope.Refs) {
			g.scope = input.GraphScope{} // every reference chosen gone: Auto, said (design D3)
			tips, _ = msg.refs.Tips(g.scope)
			v.ScopeNote = "the references chosen are gone"
		} else {
			v.ScopeNote = plural(len(gone), "reference") + " gone"
		}
	}
	m.graphLabels()
	v.Scope, v.Follows = graphScopeText(g.scope, msg.refs)
	switch {
	case len(tips) == 0:
		m.stopGraphRead()
		v.Rows, v.Commits, v.Labels, v.More = nil, nil, nil, ""
		g.tips, g.done = nil, true
		clear(g.index)
		v.State, v.Failed = "no commits yet", false
	case !slices.Equal(tips, g.tips) || len(v.Rows) == 0 && !g.reading || !g.done && g.reader == nil && !g.reading:
		// The tips moved, nothing read yet, or a read the panel's hiding ended
		// before the history's end: read again, the place kept (design D7).
		cmds = append(cmds, m.readHistory(tips)...)
	default:
		m.relabel()
	}
	return cmds
}

// graphProblem is a read's failure as the panel says it, and whether it is
// a failure or a plain note.
func graphProblem(err error) (string, bool) {
	switch {
	case errors.As(err, new(input.NotRepoError)):
		return "not a git repository", false
	case errors.Is(err, input.ErrNoGit):
		return "git not found", false
	}
	return "git: " + err.Error(), true
}

// readHistory reads the history again from tips (design D4): the read under
// way ended, the place kept by hash, the first page asked of a new git
// process; the rows shown stay until it comes.
func (m *Model) readHistory(tips []string) []tea.Cmd {
	gg, ok := m.graphGit()
	if !ok {
		return nil
	}
	g, v := &m.graph, &m.ui.Graph
	m.keepGraphPlace()
	m.stopGraphRead()
	g.tips, g.done, g.reading = tips, false, true
	if len(v.Rows) == 0 {
		v.State, v.Failed = "reading…", false
	}
	gen, root := g.gen, m.fileRoot()
	return []tea.Cmd{func() tea.Msg {
		r, err := gg.GraphLog(root, tips)
		if err != nil {
			return graphPageMsg{gen: gen, first: true, err: err}
		}
		commits, done, err := r.Next(graphPage)
		return graphPageMsg{gen: gen, reader: r, first: true, commits: commits, done: done, err: err}
	}}
}

// keepGraphPlace records the cursor and the first row in view by hash, for
// the history read again to put back; a place already kept stays.
func (m *Model) keepGraphPlace() {
	g, v := &m.graph, m.ui.Graph
	if g.keep.set || len(v.Rows) == 0 {
		return
	}
	cur := v.Rows[min(max(v.Cursor, 0), len(v.Rows)-1)]
	g.keep = graphPlace{set: true, cursor: v.Commits[cur.Commit].Hash}
	if cur.File >= 0 {
		if fs := v.Files[cur.Commit]; fs != nil && int(cur.File) < len(fs.Files) {
			g.keep.file = fs.Files[cur.File].Path
		}
	}
	top := render.ListTop(v.Top, max(v.Cursor, 0), m.graphRoom(), len(v.Rows))
	if top < len(v.Rows) {
		g.keep.first = v.Commits[v.Rows[top].Commit].Hash
	}
}

// graphPaged takes a page of the history: its commits added once each,
// their rows and labels, the place kept put back when reached, the next
// page asked while the place is still to come (design D7) or the view near
// the end.
func (m *Model) graphPaged(msg graphPageMsg) []tea.Cmd {
	g, v := &m.graph, &m.ui.Graph
	if msg.gen != g.gen || !v.Open {
		if msg.first && msg.reader != nil {
			msg.reader.Close() // a read replaced before its first page came: git stopped
		}
		return nil
	}
	g.reading = false
	if msg.moved {
		// The references moved before this page: the history read again from
		// them, the place kept (design D4b).
		return m.graphRefsRead(graphRefsMsg{gen: g.refsGen, refs: msg.refs})
	}
	v.More = ""
	if msg.first {
		// A new read's first page: the rows it replaces go, its lanes laid
		// out afresh.
		g.reader = msg.reader
		g.lanes = &render.GraphLanes{}
		v.Lanes, v.Loc = g.lanes, m.opt.Location
		v.Card, v.Hover = nil, 0 // the rows it was about go
		clear(g.index)
		v.Commits, v.Rows, v.Cursor, v.Top = nil, nil, 0, 0
		v.Labels, v.Files = map[int32][]render.GraphLabel{}, map[int32]*render.GraphFiles{}
	}
	if msg.err != nil {
		m.stopGraphRead()
		g.done = true
		if len(v.Commits) == 0 {
			v.State, v.Failed = graphProblem(msg.err)
		} else {
			v.More = "git: " + msg.err.Error()
		}
		return nil
	}
	v.State, v.Failed = "", false
	for _, c := range msg.commits {
		e, ok := m.graphEntry(c)
		if !ok {
			m.stopGraphRead()
			g.done = true
			v.More = "a commit's hash git gives in a form the graph does not read: " + c.Hash
			break
		}
		if _, seen := g.index[e.Hash]; seen {
			continue // never twice (design D4)
		}
		i := int32(len(v.Commits))
		v.Commits = append(v.Commits, e)
		g.index[e.Hash] = i
		g.lanes.Add(e.Hash, g.parents)
		if ls := g.labels[e.Hash]; ls != nil {
			v.Labels[i] = ls
		}
		v.Rows = append(v.Rows, render.GraphRowRef{Commit: i, File: -1})
		if g.unfolded[e.Hash] {
			if fs := g.files[e.Hash]; fs != nil {
				v.Files[i] = fs
			}
			v.Rows = append(v.Rows, fileRows(i, v.Files[i])...)
		}
		if len(v.Commits) >= graphGuard {
			m.stopGraphRead()
			g.done = true
			v.More = "first " + plural(graphGuard, "commit")
			break
		}
	}
	if msg.done && g.reader != nil {
		g.reader.Close() // the history ended: git has, and is waited for
		g.reader = nil
		g.done = true
	}
	if g.keep.set {
		if !m.putGraphPlace(g.done) {
			return m.nextGraphPage(false) // the place kept is further on: read on to it
		}
		g.keep = graphPlace{}
	}
	return m.graphMore()
}

// graphEntry is commit c in the history's compact form, its author's name
// shared, its parents decoded into m.graph.parents for its lanes; false
// for a hash that is not 40 hex digits (a SHA-256 repository's, which the
// graph does not read).
func (m *Model) graphEntry(c input.GraphCommit) (render.GraphEntry, bool) {
	var e render.GraphEntry
	if len(c.Hash) != 40 {
		return e, false
	}
	if _, err := hex.Decode(e.Hash[:], []byte(c.Hash)); err != nil {
		return e, false
	}
	g := &m.graph
	g.parents = g.parents[:0]
	for _, p := range c.Parents {
		var h [20]byte
		if len(p) != 40 {
			return e, false
		}
		if _, err := hex.Decode(h[:], []byte(p)); err != nil {
			return e, false
		}
		g.parents = append(g.parents, h)
	}
	if len(g.parents) > 0 {
		e.Parent = g.parents[0]
	}
	e.Parents = uint8(min(len(c.Parents), 255))
	a, ok := m.graph.authors[c.Author]
	if !ok {
		a = c.Author
		m.graph.authors[a] = a
	}
	e.Author, e.Time, e.Subject = a, c.Time, c.Subject
	return e, true
}

// fileRows are commit i's file rows: its state's row while read or failed,
// or the note that it changed none; its files, and the guard's note past
// them.
func fileRows(i int32, fs *render.GraphFiles) []render.GraphRowRef {
	if fs == nil || fs.State != "" || fs.Total == 0 {
		return []render.GraphRowRef{{Commit: i, File: 0}}
	}
	n := len(fs.Files)
	if fs.Total > n {
		n++
	}
	out := make([]render.GraphRowRef, n)
	for k := range out {
		out[k] = render.GraphRowRef{Commit: i, File: int32(k)}
	}
	return out
}

// putGraphPlace puts the cursor and the view back on the place kept once
// both its commits are read, or, final, the history read whole, as much of
// it as remains: a cursor commit gone, the cursor on the first row kept, or
// the history's first (design D7). False while more is to be read.
func (m *Model) putGraphPlace(final bool) bool {
	g, v := &m.graph, &m.ui.Graph
	first, okFirst := g.index[g.keep.first]
	cur, okCur := g.index[g.keep.cursor]
	if !(okFirst && okCur) && !final {
		return false
	}
	if okFirst {
		v.Top = m.graphRowOf(first, "")
	}
	switch {
	case okCur:
		v.Cursor = m.graphRowOf(cur, g.keep.file)
	case okFirst:
		v.Cursor = v.Top
	}
	return true
}

// graphRowOf is the row of commit c, or of its file at path when unfolded.
func (m *Model) graphRowOf(c int32, path string) int {
	v := m.ui.Graph
	for i, r := range v.Rows {
		if r.Commit != c {
			continue
		}
		if path == "" || r.File < 0 {
			if path == "" {
				return i
			}
			continue
		}
		if fs := v.Files[c]; fs != nil && int(r.File) < len(fs.Files) && fs.Files[r.File].Path == path {
			return i
		}
	}
	for i, r := range v.Rows {
		if r.Commit == c && r.File < 0 {
			return i
		}
	}
	return 0
}

// graphMore asks the next page when the cursor or the view comes within
// graphNear rows of the last row read (design D4).
func (m *Model) graphMore() []tea.Cmd {
	g, v := m.graph, m.ui.Graph
	if g.reading || g.done || g.reader == nil {
		return nil
	}
	last := len(v.Rows) - 1
	if v.Cursor < last-graphNear && render.ListTop(v.Top, max(v.Cursor, 0), m.graphRoom(), len(v.Rows))+m.graphRoom() < last-graphNear {
		return nil
	}
	return m.nextGraphPage(true)
}

// nextGraphPage reads the next page in a command; with check, the
// references are read first and, when the scope's tips moved, the history is
// read again from them instead (design D4b).
func (m *Model) nextGraphPage(check bool) []tea.Cmd {
	g := &m.graph
	gg, ok := m.graphGit()
	if !ok || g.reader == nil || g.reading {
		return nil
	}
	g.reading = true
	m.ui.Graph.More = "reading…"
	gen, r, root, scope, tips := g.gen, g.reader, m.fileRoot(), g.scope, slices.Clone(g.tips)
	return []tea.Cmd{func() tea.Msg {
		if check {
			if refs, err := gg.GraphRefs(root); err == nil {
				if now, _ := refs.Tips(scope); !slices.Equal(now, tips) {
					return graphPageMsg{gen: gen, moved: true, refs: refs}
				}
			}
		}
		commits, done, err := r.Next(graphPage)
		return graphPageMsg{gen: gen, commits: commits, done: done, err: err}
	}}
}

// graphLabels makes each commit's labels from the references read (design
// D11): HEAD's branch first and marked, then the local branches, the remote
// branches and the tags; a detached HEAD's "HEAD".
func (m *Model) graphLabels() {
	g := &m.graph
	clear(g.labels)
	refs := g.refs
	add := func(commit string, lb render.GraphLabel) {
		var h [20]byte
		if len(commit) != 40 {
			return
		}
		if _, err := hex.Decode(h[:], []byte(commit)); err != nil {
			return
		}
		g.labels[h] = append(g.labels[h], lb)
	}
	if refs.Branch == "" && refs.Head != "" {
		add(refs.Head, render.GraphLabel{Name: "HEAD", Kind: 'h', Head: true})
	}
	for _, kind := range []byte{'h', 'r', 't'} {
		for _, r := range refs.Refs {
			short, k := refShort(r.Name)
			if k != kind {
				continue
			}
			lb := render.GraphLabel{Name: short, Kind: k, Head: r.Name == refs.Branch}
			if lb.Head {
				var h [20]byte
				if _, err := hex.Decode(h[:], []byte(r.Commit)); err == nil {
					g.labels[h] = append([]render.GraphLabel{lb}, g.labels[h]...)
				}
				continue
			}
			add(r.Commit, lb)
		}
	}
}

// relabel puts the labels read on the commits the history holds.
func (m *Model) relabel() {
	v := &m.ui.Graph
	v.Labels = map[int32][]render.GraphLabel{}
	for h, ls := range m.graph.labels {
		if i, ok := m.graph.index[h]; ok {
			v.Labels[i] = ls
		}
	}
}

// refShort is a reference's short name and its kind: 'h' a local branch,
// 'r' a remote branch, 't' a tag, 0 another.
func refShort(name string) (string, byte) {
	switch {
	case strings.HasPrefix(name, "refs/heads/"):
		return strings.TrimPrefix(name, "refs/heads/"), 'h'
	case strings.HasPrefix(name, "refs/remotes/"):
		return strings.TrimPrefix(name, "refs/remotes/"), 'r'
	case strings.HasPrefix(name, "refs/tags/"):
		return strings.TrimPrefix(name, "refs/tags/"), 't'
	}
	return name, 0
}

// graphScopeText is the scope row's name and, for Auto, what it follows
// (design D2).
func graphScopeText(s input.GraphScope, refs input.GraphRefs) (scope, follows string) {
	switch {
	case s.All:
		return "All", ""
	case len(s.Refs) == 1:
		short, _ := refShort(s.Refs[0])
		return short, ""
	case len(s.Refs) > 1:
		return plural(len(s.Refs), "reference"), ""
	}
	switch {
	case refs.Branch != "":
		branch, _ := refShort(refs.Branch)
		if refs.Upstream != "" && slices.ContainsFunc(refs.Refs, func(r input.GraphRef) bool { return r.Name == refs.Upstream }) {
			up, _ := refShort(refs.Upstream)
			return "Auto", branch + " → " + up
		}
		return "Auto", branch + ", no upstream"
	case refs.Head != "":
		return "Auto", "detached at " + refs.Head[:min(7, len(refs.Head))]
	}
	return "Auto", ""
}

// plural is n and its noun: "1 reference", "3 references".
func plural(n int, noun string) string {
	if n == 1 {
		return "1 " + noun
	}
	return grouped(n) + " " + noun + "s"
}

// grouped is n with its thousands apart: "1,000,000".
func grouped(n int) string {
	s := fmt.Sprint(n)
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	return s
}

// graphRoom is how many rows of the history the panel shows.
func (m *Model) graphRoom() int { return max(render.GraphRoom(m.s, m.ui), 1) }

// graphMove moves the cursor d rows, kept in the rows and in view, and asks
// the next page near the end.
func (m *Model) graphMove(d int) []tea.Cmd {
	v := &m.ui.Graph
	if len(v.Rows) == 0 {
		return nil
	}
	v.Cursor = min(max(v.Cursor+d, 0), len(v.Rows)-1)
	v.Top = render.ListTop(v.Top, v.Cursor, m.graphRoom(), len(v.Rows))
	return m.graphMore()
}

// graphKey handles a key while the Git Graph has the focus (design D8): its
// keys, the global ones passed on as beside the other panels.
func (m *Model) graphKey(k tea.KeyPressMsg) (cmds []tea.Cmd, done bool) {
	s := k.String()
	switch s {
	case "f1", "f3", "f4", "f6", "shift+f6", "tab", "shift+tab", "alt+p", "alt+b", "alt+g", "f12", "ctrl+shift+c", "ctrl+o", "ctrl+r",
		"shift+pgup", "shift+pgdown", "ctrl+q", "F":
		return nil, false
	}
	if k.Text == "?" {
		return nil, false // the key overlay
	}
	v := &m.ui.Graph
	room := m.graphRoom()
	switch s {
	case "esc":
		m.ui.Focus = "B" // back to the input, the panel kept beside it, as the Explorer's
		return nil, true
	case "up":
		return m.graphMove(-1), true
	case "down":
		return m.graphMove(1), true
	case "pgup":
		return m.graphMove(-room), true
	case "pgdown":
		return m.graphMove(room), true
	case "home", "ctrl+home":
		return m.graphMove(-len(v.Rows)), true
	case "end", "ctrl+end":
		return m.graphMove(len(v.Rows)), true
	case "s":
		return m.openScope(), true
	case "f":
		v.Wide = !v.Wide // zone A's columns too, or back (design D10)
		return nil, true
	case "i":
		return m.graphCardPin(), true // the cursor's card (design D13)
	case "ctrl+c":
		return m.graphCopy(), true
	}
	if v.Cursor < 0 || v.Cursor >= len(v.Rows) {
		return nil, true
	}
	r := v.Rows[v.Cursor]
	switch s {
	case "right":
		if r.File < 0 {
			return m.unfoldCommit(r.Commit, true), true
		}
	case "left":
		if r.File >= 0 {
			v.Cursor = m.graphRowOf(r.Commit, "")
			v.Top = render.ListTop(v.Top, v.Cursor, room, len(v.Rows))
		} else {
			m.unfoldCommit(r.Commit, false)
		}
	case "enter":
		if r.File < 0 {
			return m.unfoldCommit(r.Commit, !m.graph.unfolded[v.Commits[r.Commit].Hash]), true
		}
		return m.openGraphDiff(r), true
	}
	return nil, true
}

// graphCopy copies the cursor's commit's full hash, or its file's path.
func (m *Model) graphCopy() []tea.Cmd {
	v := m.ui.Graph
	if v.Cursor < 0 || v.Cursor >= len(v.Rows) {
		return nil
	}
	r := v.Rows[v.Cursor]
	if fs := v.Files[r.Commit]; r.File >= 0 && fs != nil && int(r.File) < len(fs.Files) {
		return m.copy(fs.Files[r.File].Path)
	}
	return m.copy(v.Commits[r.Commit].HashText())
}

// unfoldCommit unfolds commit c into its files, reading them in a command
// the first time, or folds it (design D5); the cursor stays on its row.
func (m *Model) unfoldCommit(c int32, open bool) []tea.Cmd {
	g, v := &m.graph, &m.ui.Graph
	e := v.Commits[c]
	if g.unfolded[e.Hash] == open {
		return nil
	}
	row := m.graphRowOf(c, "")
	if !open {
		delete(g.unfolded, e.Hash)
		end := row + 1
		for end < len(v.Rows) && v.Rows[end].Commit == c {
			end++
		}
		v.Rows = slices.Delete(v.Rows, row+1, end)
		v.Cursor = row
		v.Top = render.ListTop(v.Top, v.Cursor, m.graphRoom(), len(v.Rows))
		return nil
	}
	g.unfolded[e.Hash] = true
	var cmds []tea.Cmd
	fs := g.files[e.Hash]
	if fs == nil {
		fs = &render.GraphFiles{State: "reading…"}
		cmds = m.readCommitFiles(e)
	}
	v.Files[c] = fs
	v.Rows = slices.Insert(v.Rows, row+1, fileRows(c, fs)...)
	v.Cursor = row
	return cmds
}

// readCommitFiles reads commit e's files in a command, against its first
// parent, a root's against nothing (design D5).
func (m *Model) readCommitFiles(e render.GraphEntry) []tea.Cmd {
	gg, ok := m.graphGit()
	if !ok {
		return nil
	}
	hash, commit, parent, root := e.Hash, e.HashText(), e.ParentText(), m.fileRoot()
	return []tea.Cmd{func() tea.Msg {
		files, err := gg.CommitFiles(root, commit, parent)
		return graphFilesMsg{hash: hash, files: files, err: err}
	}}
}

// graphFilesRead takes a commit's files: kept for the session, and in its
// rows when it is still unfolded.
func (m *Model) graphFilesRead(msg graphFilesMsg) {
	g, v := &m.graph, &m.ui.Graph
	delete(g.filesAsked, msg.hash)
	if v.Card != nil {
		defer m.graphCardShow() // the card's summary of them
	}
	if len(g.files) >= graphCardKept {
		// The files the cards read, past the bound: those of the commits
		// not unfolded go.
		for h := range g.files {
			if !g.unfolded[h] {
				delete(g.files, h)
			}
		}
	}
	fs := &render.GraphFiles{Total: msg.files.Total}
	if msg.err != nil {
		fs.State, fs.Failed = graphProblem(msg.err)
	} else {
		fs.Files = make([]render.GraphFile, len(msg.files.Files))
		for i, f := range msg.files.Files {
			fs.Files[i] = render.GraphFile{Status: f.Status, Path: f.Path, Orig: f.Orig, Added: f.Added, Removed: f.Removed, Binary: f.Binary}
		}
		g.files[msg.hash] = fs // a failure is read again at the next unfolding
	}
	c, ok := g.index[msg.hash]
	if !ok || !g.unfolded[msg.hash] {
		return
	}
	v.Files[c] = fs
	row := m.graphRowOf(c, "")
	end := row + 1
	for end < len(v.Rows) && v.Rows[end].Commit == c {
		end++
	}
	v.Rows = slices.Replace(v.Rows, row+1, end, fileRows(c, fs)...)
	if v.Cursor >= end {
		v.Cursor += len(fileRows(c, fs)) - (end - row - 1)
	}
}

// openGraphDiff shows what the commit of row r changed in its file, in the
// diff viewer, read only: the first parent's version against the commit's,
// compared in a command (design D6). Source Control's diff, kept since the
// panel showed, waits for Source Control.
func (m *Model) openGraphDiff(r render.GraphRowRef) []tea.Cmd {
	gg, ok := m.graphGit()
	v := m.ui.Graph
	fs := v.Files[r.Commit]
	if !ok || fs == nil || r.File < 0 || int(r.File) >= len(fs.Files) {
		return nil
	}
	e, f := v.Commits[r.Commit], fs.Files[r.File]
	what := "Commit " + e.Short()
	switch {
	case e.Parents == 0:
		what = "Root commit " + e.Short()
	case e.Parents > 1:
		what = "Merge " + e.Short() + ", against its first parent " + e.ParentText()[:7]
	}
	m.ui.Diff = render.DiffView{Open: true, Path: f.Path, Orig: f.Orig, Note: "comparing…", Commit: what}
	m.ui.Focus = "D"
	m.diffGen++
	gen, root, commit, parent, now := m.diffGen, m.fileRoot(), e.HashText(), e.ParentText(), m.opt.Now
	file := input.CommitFile{Status: f.Status, Path: f.Path, Orig: f.Orig, Added: f.Added, Removed: f.Removed, Binary: f.Binary}
	return []tea.Cmd{func() tea.Msg {
		start := now()
		pair, err := gg.CommitSides(root, commit, parent, file)
		msg := diffMsg{gen: gen, path: file.Path, pair: pair, err: err}
		if err == nil && textual(pair.Old) && textual(pair.New) {
			msg.res = diff.Compare(pair.Old.Data, pair.New.Data, diffContext)
		}
		msg.took = now().Sub(start)
		return msg
	}}
}

// graphClick handles a left click on the panel, item the cell's (design
// D8): the panel takes the focus; on a row the cursor goes to it, on the
// cursor's commit it unfolds or folds it, on the cursor's file it opens its
// diff, as the Explorer's rule; the scope row opens the scope popup; the
// header's "⌄" hides the panel.
func (m *Model) graphClick(item int) []tea.Cmd {
	m.ui.Sel = render.Selection{}
	switch item {
	case render.ItemPanelHide:
		m.closeGraph()
		return nil
	case render.ItemGraphScope:
		m.ui.Focus = "R"
		return m.openScope()
	}
	m.ui.Focus = "R"
	v := &m.ui.Graph
	i := item - 1
	if i < 0 || i >= len(v.Rows) {
		return nil
	}
	if i != v.Cursor {
		v.Cursor = i
		return m.graphMore()
	}
	r := v.Rows[i]
	if r.File < 0 {
		return m.unfoldCommit(r.Commit, !m.graph.unfolded[v.Commits[r.Commit].Hash])
	}
	return m.openGraphDiff(r)
}

// graphWheel moves the cursor wheelRows rows, the pointer over the panel.
func (m *Model) graphWheel(b tea.MouseButton) []tea.Cmd {
	switch b {
	case tea.MouseWheelUp:
		return m.graphMove(-wheelRows)
	case tea.MouseWheelDown:
		return m.graphMove(wheelRows)
	}
	return nil
}
