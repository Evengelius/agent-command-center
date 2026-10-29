package app

import (
	"errors"
	"os"
	"strconv"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/Evengelius/agent-command-center/internal/input"
	"github.com/Evengelius/agent-command-center/internal/render"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// The branches popup (plan 59, brief amendment 61; ui-design.md, Branches):
// a click on the project bar's branch name, or Alt+B, opens and closes it
// over the dashboard; while it is open its keys and the mouse work it. Its
// git runs in commands (input.BranchGit): the lists, read when it opens and
// after a fetch; a switch, asked by Enter or a double click, confirmed when
// uncommitted changes would come along; a fetch, which Esc and FetchLimit
// stop. A switch or a fetch outlives the popup: opened again, it shows it
// running. The header's branch follows the HEAD file (design D9).

// Messages.
type (
	branchesMsg struct {
		gen int
		b   input.Branches
		err error
	}
	// branchChangesMsg is the uncommitted changes counted before the switch
	// sw (design D5).
	branchChangesMsg struct {
		sw  *branchSwitch
		n   int
		err error
	}
	// branchSwitchedMsg is the switch sw's end: from the branch the header
	// named to the one now checked out.
	branchSwitchedMsg struct {
		sw       *branchSwitch
		from, to string
		err      error
	}
	// fetchedMsg is the fetch of stop ended; fetchLimitMsg its FetchLimit
	// reached, or a refused stop tried again (design D7).
	fetchedMsg struct {
		stop *input.GitStop
		err  error
	}
	fetchLimitMsg struct{ stop *input.GitStop }
	// headFileMsg is the HEAD file's path and time, resolved once (design
	// D9); headTickMsg a check due, headCheckMsg its result: the branch read
	// again when the time changed.
	headFileMsg struct {
		gen  int
		path string
		mod  time.Time
		err  error
	}
	headTickMsg  struct{ gen int }
	headCheckMsg struct {
		gen     int
		mod     time.Time
		changed bool
		branch  string
	}
)

// branchSwitch is a switch the popup asked (design D5): to the branch name,
// a remote branch made a tracking one; the changes counted, the
// confirmation waiting, git's switch running.
type branchSwitch struct {
	name    string
	remote  bool
	n       int
	confirm bool
	running bool
}

// fetchRetry is how soon FetchLimit's stop, refused while git holds a lock,
// is tried again.
const fetchRetry = 200 * time.Millisecond

// branchGit is the run's git for the popup; false when it has none.
func (m *Model) branchGit() (input.BranchGit, bool) {
	bg, ok := m.git().(input.BranchGit)
	return bg, ok
}

// toggleBranches opens the popup, or closes it when it is open (design D1).
// It opens on a project in a git repository whose branch the project bar
// shows; otherwise a toast says why. Its lists are read in a command,
// Recent and Local open, Remote and Tags folded.
func (m *Model) toggleBranches() []tea.Cmd {
	if m.ui.Branches.Open {
		m.closeBranches()
		return nil
	}
	if m.ui.Branch == "" {
		return m.toast("no branches: the project is not in a git repository")
	}
	open := m.ui
	open.Branches = render.BranchesView{Open: true}
	sc := m.screen
	if sc == nil {
		sc = render.Draw(m.s, m.ui) // the screen the next frame draws on; m.screen stays the terminal's
	}
	if !render.Layers(m.s, open).Shown(render.LayerBranches) || !sc.BranchShown() {
		// A full view, a window or the key overlay over the dashboard, or a
		// project bar too narrow for the branch.
		return m.toast("the branches popup needs the project bar's branch on screen")
	}
	m.closePopups() // the one popup open (plan 49, design D7)
	m.ui.Branches = render.BranchesView{Open: true, Reading: true}
	m.ui.Branches.Folded[render.GroupRemote] = true
	m.ui.Branches.Folded[render.GroupTags] = true
	return m.readBranches()
}

// closeBranches closes the popup: its read under way is dropped, and so is
// a switch not yet running; a switch or a fetch running goes on.
func (m *Model) closeBranches() {
	if m.ui.Branches.Open {
		m.branchGen++
	}
	m.ui.Branches = render.BranchesView{}
	m.branchFind.Clear()
	if m.sw != nil && !m.sw.running {
		m.sw = nil
	}
}

// readBranches reads the popup's lists in a command (design D4); the lists
// shown stay until it answers.
func (m *Model) readBranches() []tea.Cmd {
	v := &m.ui.Branches
	bg, ok := m.branchGit()
	if !ok {
		v.Reading, v.Problem = false, "no git in this run"
		return nil
	}
	m.branchGen++
	gen, dir := m.branchGen, m.fileRoot()
	return []tea.Cmd{func() tea.Msg {
		b, err := bg.Branches(dir)
		return branchesMsg{gen, b, err}
	}}
}

// branchesRead shows the lists read, or why there are none (design D3).
// Recent's rows are Local's, with their upstreams.
func (m *Model) branchesRead(msg branchesMsg) {
	v := &m.ui.Branches
	if msg.gen != m.branchGen || !v.Open {
		return // closed, or an older read
	}
	v.Reading, v.Problem = false, ""
	if msg.err != nil {
		v.Problem = gitProblem(msg.err)
		return
	}
	b := msg.b
	h := b.Head
	v.Head, v.ID, v.Detached, v.NoCommits = h.Name, h.Head, h.Detached, h.NoCommits
	v.Upstream, v.Ahead, v.Behind, v.Gone = h.Upstream, h.Ahead, h.Behind, h.Gone
	local := branchRows(b.Local)
	byName := map[string]render.BranchRow{}
	for _, r := range local {
		byName[r.Name] = r
	}
	recent := make([]render.BranchRow, 0, len(b.Recent))
	for _, n := range b.Recent {
		r, ok := byName[n]
		if !ok {
			r = render.BranchRow{Name: n} // past Local's cap
		}
		recent = append(recent, r)
	}
	v.Groups = [render.BranchGroups][]render.BranchRow{recent, local, branchRows(b.Remote), branchRows(b.Tags)}
	v.Totals = [render.BranchGroups]int{len(recent), b.LocalTotal, b.RemoteTotal, b.TagsTotal}
	m.fixBranchCursor()
}

// branchRows is git's refs as the popup's rows.
func branchRows(refs []input.Ref) []render.BranchRow {
	out := make([]render.BranchRow, 0, len(refs))
	for _, r := range refs {
		out = append(out, render.BranchRow{Name: r.Name, Upstream: r.Upstream, Ahead: r.Ahead, Behind: r.Behind,
			Gone: r.Gone, Current: r.Current, Elsewhere: r.Elsewhere})
	}
	return out
}

// gitProblem is a git error as the popup shows it (design D3).
func gitProblem(err error) string {
	switch {
	case errors.Is(err, input.ErrNoGit):
		return "git not found"
	case errors.As(err, new(input.NotRepoError)):
		return "not a git repository"
	}
	return err.Error()
}

// fixBranchCursor keeps the cursor on a row it can stop on: where it is,
// else the current branch, else the first.
func (m *Model) fixBranchCursor() {
	v := &m.ui.Branches
	rows := render.BranchRows(*v)
	if render.BranchCursor(rows, v.Cursor) >= 0 {
		return
	}
	v.Cursor = ""
	for _, it := range rows {
		if it.Kind == 'b' && it.Group == render.GroupLocal && it.Row.Current {
			v.Cursor = it.Key()
			return
		}
	}
	for _, it := range rows {
		if it.Kind != 'm' {
			v.Cursor = it.Key()
			return
		}
	}
}

// setBranchQuery searches q (design D10): the cursor on the first branch or
// tag that matches; with the search cleared it stays on its row if shown.
func (m *Model) setBranchQuery(q string) {
	v := &m.ui.Branches
	v.Query, v.Top = q, 0
	if q != "" {
		v.Cursor = ""
		for _, it := range render.BranchRows(*v) {
			if it.Kind == 'b' || it.Kind == 't' {
				v.Cursor = it.Key()
				break
			}
		}
	}
	m.fixBranchCursor()
}

// branchPaste types a paste's first line in the search (design D10), its
// control characters left out.
func (m *Model) branchPaste(text string) {
	q := strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f {
			return -1
		}
		return r
	}, firstLine(text))
	if q != "" {
		m.branchFind.Insert(q)
		m.setBranchQuery(m.branchFind.Text())
	}
}

// branchTopSync takes the list's first row from the frame drawn last, so
// that the cursor moves the list the least (render.ListTop).
func (m *Model) branchTopSync() {
	if m.screen != nil && m.screen.Branches.Shown {
		m.ui.Branches.Top = m.screen.Branches.Top
	}
}

// branchRoom is the rows of the list the frame drawn last showed.
func (m *Model) branchRoom() int {
	if m.screen != nil && m.screen.Branches.Shown {
		return max(m.screen.Branches.Height, 1)
	}
	return 1
}

// cursorItem is the row under the cursor; ok false when none.
func (m *Model) cursorItem() (it render.BranchItem, ok bool) {
	rows := render.BranchRows(m.ui.Branches)
	i := render.BranchCursor(rows, m.ui.Branches.Cursor)
	if i < 0 {
		return render.BranchItem{}, false
	}
	return rows[i], true
}

// moveBranchCursor moves the cursor d of the rows it stops on, a heading,
// a branch or a tag, within the list.
func (m *Model) moveBranchCursor(d int) {
	v := &m.ui.Branches
	rows := render.BranchRows(*v)
	cur := render.BranchCursor(rows, v.Cursor)
	var stops []int
	at := 0
	for i, it := range rows {
		if it.Kind == 'm' {
			continue
		}
		if i <= cur {
			at = len(stops)
		}
		stops = append(stops, i)
	}
	if len(stops) == 0 {
		return
	}
	v.Cursor = rows[stops[min(max(at+d, 0), len(stops)-1)]].Key()
}

// foldBranches folds the group g, the cursor on its heading, or opens it.
// A search shows every group open: nothing folds meanwhile.
func (m *Model) foldBranches(g int, fold bool) {
	v := &m.ui.Branches
	if v.Query != "" {
		return
	}
	v.Folded[g] = fold
	if fold {
		m.toHeading(g)
	}
}

// toHeading puts the cursor on the group g's heading.
func (m *Model) toHeading(g int) {
	for _, it := range render.BranchRows(m.ui.Branches) {
		if it.Kind == 'g' && it.Group == g {
			m.ui.Branches.Cursor = it.Key()
		}
	}
}

// branchesKey works the open popup (design D10): the search typed, the
// cursor moved, the groups folded and opened, Enter on a branch switching
// to it, Ctrl+C copying its name, F5 fetching, Esc stopping a fetch, then
// cancelling a confirmation, then clearing the search, then closing. A
// send key acts as Enter does, so the popup never sends. Any other key
// closes it and goes on, as plan 34's selector; a modifier's own press is
// kept (plan 67).
func (m *Model) branchesKey(k tea.KeyPressMsg) ([]tea.Cmd, bool) {
	key := k.String()
	enter := key == "enter" || m.opt.Keys.IsSend(key)
	if sw := m.sw; sw != nil && sw.confirm {
		switch {
		case enter:
			return m.runSwitch(), true
		case key == "esc":
			m.sw = nil
			return nil, true
		}
		m.sw = nil // any other key cancels the confirmation, then acts
	}
	m.branchTopSync()
	v := &m.ui.Branches
	switch {
	case enter:
		return m.branchEnter(), true
	case key == "esc":
		switch {
		case m.fetch != nil && !m.fetchStopping:
			return m.stopFetch(), true
		case v.Query != "":
			m.branchFind.Clear()
			m.setBranchQuery("")
		default:
			m.closeBranches()
		}
	case key == "alt+b":
		m.closeBranches()
	case key == "up":
		m.moveBranchCursor(-1)
	case key == "down":
		m.moveBranchCursor(1)
	case key == "pgup":
		m.moveBranchCursor(-m.branchRoom())
	case key == "pgdown":
		m.moveBranchCursor(m.branchRoom())
	case key == "home":
		m.moveBranchCursor(-len(render.BranchRows(*v)))
	case key == "end":
		m.moveBranchCursor(len(render.BranchRows(*v)))
	case key == "right":
		if it, ok := m.cursorItem(); ok && it.Kind == 'g' {
			m.foldBranches(it.Group, false)
		}
	case key == "left":
		if it, ok := m.cursorItem(); ok && it.Kind == 'g' {
			m.foldBranches(it.Group, true)
		} else if ok {
			m.toHeading(it.Group) // a row: its group's heading
		}
	case key == "backspace":
		if m.branchFind.Text() != "" {
			m.branchFind.Backspace()
			m.setBranchQuery(m.branchFind.Text())
		}
	case key == "ctrl+c" || key == "ctrl+shift+c":
		if it, ok := m.cursorItem(); ok && (it.Kind == 'b' || it.Kind == 't') {
			return m.copy(it.Row.Name), true // its name (design D8); never an interrupt
		}
	case key == "ctrl+v" || key == "ctrl+shift+v":
		// A terminal that sends a raw Ctrl+V: the clipboard's text in the
		// search, as a bracketed paste.
		if cb := m.opt.Clipboard; cb != nil {
			return []tea.Cmd{func() tea.Msg {
				text, err := cb.Read()
				if err != nil {
					return toastMsg{"paste failed: " + err.Error()}
				}
				return pasteMsg{text}
			}}, true
		}
	case key == "f5":
		return m.fetchBranches(), true
	case modifierKey(k):
		// Kept: a modifier's own press comes before its key, AltGr's left
		// Ctrl and right Alt before its character, which reaches the search
		// (plan 67, design D2).
	case keyText(k) != "" && printable(k.Text): // AltGr's characters too (plan 67)
		m.branchFind.Insert(k.Text)
		m.setBranchQuery(m.branchFind.Text())
	default:
		m.closeBranches()
		return nil, false
	}
	return nil, true
}

// printable says s holds no control character.
func printable(s string) bool {
	for _, r := range s {
		if r < 0x20 || r == 0x7f {
			return false
		}
	}
	return true
}

// branchEnter is Enter on the cursor's row: a heading folds or opens, a
// branch is switched to (design D5); a tag or a message does nothing.
func (m *Model) branchEnter() []tea.Cmd {
	it, ok := m.cursorItem()
	switch {
	case !ok:
	case it.Kind == 'g':
		m.foldBranches(it.Group, !m.ui.Branches.Folded[it.Group])
	case it.Kind == 'b':
		return m.askSwitch(it)
	}
	return nil
}

// branchesClick is a left click on the popup, at (x, y) on its item (design
// D10): on "fetch" a fetch; on a heading its group folds or opens, a second
// click adding nothing; on a branch or a tag the cursor goes there, and a
// double click on a branch switches to it. The frame and the messages do
// nothing; a click cancels a confirmation waiting.
func (m *Model) branchesClick(item, x, y int) []tea.Cmd {
	m.countClick(x, y)
	m.branchTopSync()
	v := &m.ui.Branches
	if item == render.ItemBranchFetch {
		return m.fetchBranches()
	}
	rows := render.BranchRows(*v)
	i := item - render.ItemBranchRow
	if item < render.ItemBranchRow || v.Problem != "" || v.Reading || i >= len(rows) || rows[i].Kind == 'm' {
		return nil
	}
	if m.sw != nil && m.sw.confirm {
		m.sw = nil
	}
	it := rows[i]
	v.Cursor = it.Key()
	switch {
	case it.Kind == 'g' && m.click.n == 1:
		m.foldBranches(it.Group, !v.Folded[it.Group])
	case it.Kind == 'b' && m.click.n == 2:
		return m.askSwitch(it)
	}
	return nil
}

// branchesWheel moves the cursor three rows, the list with it, as the
// Explorer's wheel does; a confirmation waiting is cancelled.
func (m *Model) branchesWheel(b tea.MouseButton) {
	if b != tea.MouseWheelUp && b != tea.MouseWheelDown {
		return
	}
	if m.sw != nil && m.sw.confirm {
		m.sw = nil
	}
	m.branchTopSync()
	if b == tea.MouseWheelUp {
		m.moveBranchCursor(-wheelRows)
	} else {
		m.moveBranchCursor(wheelRows)
	}
}

// branchOpRunning says the popup's git changes something now: a switch or
// a fetch, which the quit question names (plan 47, design D1).
func (m *Model) branchOpRunning() bool { return m.sw != nil && m.sw.running || m.fetch != nil }

// switchRefused is why no switch may run now (design D5): the agent working
// in the folder, or Source Control's git; "" when one may.
func (m *Model) switchRefused() string {
	switch {
	case m.s.Running() != "":
		return "the agent is working in this folder: switch once its turn ends"
	case m.op != nil:
		return "Source Control's git is running: switch once it ends"
	}
	return ""
}

// askSwitch asks to switch to the branch of row it (design D5): nothing for
// the current branch or while the popup's git runs; refused, with a note,
// for a branch in another worktree and while switchRefused; else the
// changes are counted in a command, and with some a confirmation comes
// first.
func (m *Model) askSwitch(it render.BranchItem) []tea.Cmd {
	v := &m.ui.Branches
	r := it.Row
	if r.Current || m.sw != nil || m.fetch != nil {
		return nil
	}
	why := m.switchRefused()
	if r.Elsewhere != "" {
		why = r.Name + " is checked out in another worktree: " + r.Elsewhere
	}
	if why != "" {
		v.Note, v.NoteBad = why, true
		return nil
	}
	bg, ok := m.branchGit()
	if !ok {
		return nil
	}
	sw := &branchSwitch{name: r.Name, remote: it.Group == render.GroupRemote}
	m.sw, v.Note, v.NoteBad = sw, "", false
	dir := m.fileRoot()
	return []tea.Cmd{func() tea.Msg {
		n, err := bg.Changes(dir)
		return branchChangesMsg{sw, n, err}
	}}
}

// branchChanges is the changes counted: none, the switch runs; some, the
// confirmation waits for Enter. A switch since dropped stays dropped.
func (m *Model) branchChanges(msg branchChangesMsg) []tea.Cmd {
	sw := msg.sw
	if sw != m.sw {
		return nil
	}
	if msg.err != nil {
		m.sw = nil
		m.ui.Branches.Note, m.ui.Branches.NoteBad = gitProblem(msg.err), true
		return nil
	}
	sw.n = msg.n
	if sw.n > 0 {
		sw.confirm = true
		return nil
	}
	return m.runSwitch()
}

// runSwitch runs git switch in a command, the guards checked again: the
// agent may have started since the confirmation came.
func (m *Model) runSwitch() []tea.Cmd {
	sw, v := m.sw, &m.ui.Branches
	bg, ok := m.branchGit()
	why := m.switchRefused()
	if sw == nil || !ok || why != "" {
		m.sw = nil
		v.Note, v.NoteBad = why, why != ""
		return nil
	}
	sw.confirm, sw.running = false, true
	dir, from, name, remote := m.fileRoot(), m.s.Context.Branch, sw.name, sw.remote
	return []tea.Cmd{func() tea.Msg {
		to, err := bg.Switch(dir, name, remote)
		return branchSwitchedMsg{sw, from, to, err}
	}}
}

// branchSwitched is a switch's end (design D6). Refused, git's words in the
// popup, or in a toast once it is closed; nothing changed. Done, the popup
// closes, the state records the switch and its notice, the draft, when
// empty, gets a note for the agent, and what shows the files is read again.
func (m *Model) branchSwitched(msg branchSwitchedMsg) []tea.Cmd {
	if msg.sw != m.sw {
		return nil
	}
	m.sw = nil
	if msg.err != nil {
		if m.ui.Branches.Open {
			m.ui.Branches.Note, m.ui.Branches.NoteBad = gitProblem(msg.err), true
			return nil
		}
		return m.toast("switch failed: " + gitProblem(msg.err))
	}
	m.closeBranches()
	cmds := m.do(state.BranchSwitched{From: msg.from, To: msg.to, Carried: msg.sw.n})
	if m.edit.Text() == "" && len(m.attach) == 0 && m.s.Session.Phase == state.Ready {
		m.edit.Insert("I switched the project from " + msg.from + " to " + msg.to + "; check the files you rely on before going on.")
	}
	cmds = append(cmds, m.graphRecheck()...) // the Git Graph's references (plan 62, design D4b)
	if m.ui.Explorer.Open {
		return append(cmds, m.openExplorer()...) // its folders and git's markers
	}
	return append(cmds, m.sourceEvent()...) // Source Control or the diff, while shown
}

// fetchBranches runs git fetch --all in a command (design D7), stopped by
// Esc and at FetchLimit; nothing while the popup's git runs.
func (m *Model) fetchBranches() []tea.Cmd {
	bg, ok := m.branchGit()
	if !ok || m.sw != nil || m.fetch != nil {
		return nil
	}
	stop := input.NewGitStop()
	m.fetch, m.fetchHint, m.fetchLate, m.fetchStopping = stop, "", false, false
	m.ui.Branches.Note, m.ui.Branches.NoteBad = "", false
	dir := m.fileRoot()
	return []tea.Cmd{func() tea.Msg {
		return fetchedMsg{stop, bg.Fetch(dir, stop)}
	}, m.opt.After(input.FetchLimit, fetchLimitMsg{stop})}
}

// stopFetch asks the running fetch to stop (design D7); a stop refused
// while git holds a lock says why, and FetchLimit's is tried again.
func (m *Model) stopFetch() []tea.Cmd {
	ok, why := m.fetch.Stop()
	switch {
	case ok:
		m.fetchStopping, m.fetchHint = true, ""
	case why != "":
		m.fetchHint = why
		if m.fetchLate {
			return []tea.Cmd{m.opt.After(fetchRetry, fetchLimitMsg{m.fetch})}
		}
	}
	return nil
}

// fetchLimit is FetchLimit reached by the fetch of msg's stop.
func (m *Model) fetchLimit(msg fetchLimitMsg) []tea.Cmd {
	if msg.stop != m.fetch || m.fetchStopping {
		return nil
	}
	m.fetchLate = true
	return m.stopFetch()
}

// fetched is a fetch's end: its result in the popup and the lists read
// again, a failure too, since some remotes may have answered; in a toast
// once the popup is closed.
func (m *Model) fetched(msg fetchedMsg) []tea.Cmd {
	if msg.stop != m.fetch {
		return nil
	}
	late := m.fetchLate
	m.fetch, m.fetchHint, m.fetchLate, m.fetchStopping = nil, "", false, false
	note, bad := "Fetched every remote.", false
	var r input.Refusal
	stopped := errors.As(msg.err, &r) && r.Kind == "cancelled"
	switch {
	case stopped && late:
		note, bad = "The fetch ran past its "+strconv.Itoa(int(input.FetchLimit.Minutes()))+" minutes and was stopped.", true
	case stopped:
		note = r.Msg // the owner's Esc: no failure
	case msg.err != nil:
		note, bad = gitProblem(msg.err), true
	}
	recheck := m.graphRecheck() // the Git Graph's references (plan 62, design D4b)
	if !m.ui.Branches.Open {
		return append(recheck, m.toast(note)...)
	}
	m.ui.Branches.Note, m.ui.Branches.NoteBad = note, bad
	return append(recheck, m.readBranches()...)
}

// syncBranches shows what the popup's git does (syncUI): the quit
// question's name for it, and in the popup the confirmation waiting or
// what runs.
func (m *Model) syncBranches() {
	if m.ui.GitRunning == "" {
		switch {
		case m.sw != nil && m.sw.running:
			m.ui.GitRunning = "the switch"
		case m.fetch != nil:
			m.ui.GitRunning = "the fetch"
		}
	}
	v := &m.ui.Branches
	if !v.Open {
		return
	}
	v.Busy, v.Confirm, v.Changes = "", "", 0
	switch sw := m.sw; {
	case sw != nil && sw.confirm:
		v.Confirm, v.Changes = sw.name, sw.n
	case sw != nil:
		v.Busy = "switching to " + sw.name + "…"
	case m.fetch != nil && m.fetchStopping:
		v.Busy = "fetching… stopping"
	case m.fetch != nil && m.fetchHint != "":
		v.Busy = "fetching… " + m.fetchHint
	case m.fetch != nil:
		v.Busy = "fetching… Esc stops"
	}
}

// watchHead starts the watch of the project's HEAD file (design D9): its
// path resolved once in a command, then its time checked every
// BranchWatch. No repository, no git or no period: no watch.
func (m *Model) watchHead() []tea.Cmd {
	m.headGen++
	m.headPath, m.headTime = "", time.Time{}
	bg, ok := m.branchGit()
	if m.opt.BranchWatch <= 0 || !ok || m.s.Context.Branch == "" {
		return nil
	}
	gen, dir := m.headGen, m.fileRoot()
	return []tea.Cmd{func() tea.Msg {
		path, err := bg.HeadFile(dir)
		if err != nil {
			return headFileMsg{gen: gen, err: err}
		}
		fi, err := os.Stat(path)
		if err != nil {
			return headFileMsg{gen: gen, err: err}
		}
		return headFileMsg{gen: gen, path: path, mod: fi.ModTime()}
	}}
}

// headFound is the HEAD file resolved: the checks start; an error leaves
// the header as it is, unwatched.
func (m *Model) headFound(msg headFileMsg) []tea.Cmd {
	if msg.gen != m.headGen || msg.err != nil {
		return nil
	}
	m.headPath, m.headTime = msg.path, msg.mod
	return []tea.Cmd{m.opt.After(m.opt.BranchWatch, headTickMsg{msg.gen})}
}

// headTick checks the HEAD file's time in a command, without git, and
// reads the branch again, as the validation did (readBranch), when it
// changed. The watch ends with the dashboard.
func (m *Model) headTick(msg headTickMsg) []tea.Cmd {
	if msg.gen != m.headGen || m.headPath == "" || m.s.Screen != state.ScreenDashboard {
		return nil
	}
	gen, path, last, dir := m.headGen, m.headPath, m.headTime, m.fileRoot()
	return []tea.Cmd{func() tea.Msg {
		fi, err := os.Stat(path)
		if err != nil || fi.ModTime().Equal(last) {
			return headCheckMsg{gen: gen, mod: last} // git renames HEAD.lock over it: a check missed is the next one's
		}
		return headCheckMsg{gen: gen, mod: fi.ModTime(), changed: true, branch: readBranch(dir)}
	}}
}

// headChecked is a check's result: a branch changed shows in the header,
// and in the popup's lists while it is open; the next check is due.
func (m *Model) headChecked(msg headCheckMsg) []tea.Cmd {
	if msg.gen != m.headGen {
		return nil
	}
	var cmds []tea.Cmd
	if msg.changed {
		m.headTime = msg.mod
		if msg.branch != m.s.Context.Branch {
			cmds = append(cmds, m.do(state.BranchRead{Branch: msg.branch})...)
			if m.ui.Branches.Open && !m.branchOpRunning() {
				cmds = append(cmds, m.readBranches()...)
			}
		}
	}
	return append(cmds, m.opt.After(m.opt.BranchWatch, headTickMsg{msg.gen}))
}
