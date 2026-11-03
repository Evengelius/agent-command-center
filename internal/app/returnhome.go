package app

import (
	"sync"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/Evengelius/agent-command-center/internal/render"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// The return to the home screen (plan 75, brief amendments 89 to 91): Esc where it
// does nothing else on the dashboard asks the question (design D7), never
// while the turn or a git operation runs (design D8); Yes closes the CLI as
// the quit does, and once it has exited the session's part of the
// application ends and the home screen shows, its answers kept (design D9).
// One run of the dashboard holds several sessions, one after another
// (design D11).

// sessionMsg is a message of the session the dashboard had when its command
// was made: a timer (a handshake's 60 s, an interrupt's 10 s, a close's 5 s),
// a write's result or failure, a turn review's snapshot or files, an image
// read for the draft. Once another session, or none, is the dashboard's
// (sessGen), it is dropped, a failure it reports only said in the toast:
// nothing of a session that ended reaches the next one, nor the state the
// return left (plan 75, design D9).
type sessionMsg struct {
	gen int
	msg tea.Msg
}

// sessionAfter is opt.After for a timer of the session now.
func (m *Model) sessionAfter(d time.Duration, a state.Action) tea.Cmd {
	return m.opt.After(d, sessionMsg{gen: m.sessGen, msg: actionMsg{a}})
}

// sessionCmd is c, its message numbered with the session now; nil for nil.
// c must give a message of its own, never a batch's.
func (m *Model) sessionCmd(c tea.Cmd) tea.Cmd {
	if c == nil {
		return nil
	}
	gen := m.sessGen
	return func() tea.Msg {
		msg := c()
		if msg == nil {
			return nil
		}
		return sessionMsg{gen: gen, msg: msg}
	}
}

// lateMessage is a message of a session that ended: dropped; a failure it
// reports, a write's or a panic in one, said in the toast, the state left as
// it is.
func (m *Model) lateMessage(msg tea.Msg) []tea.Cmd {
	a, ok := msg.(actionMsg)
	if r, isRefused := msg.(refusedMsg); isRefused {
		a, ok = actionMsg{r.a}, true
	}
	if !ok {
		return nil
	}
	why := ""
	switch a := a.a.(type) {
	case state.WriteFailed:
		why = a.Message
	case state.AnswerFailed:
		why = a.Message
	case state.ApprovalFailed:
		why = a.Message
	case state.Panicked:
		why = a.Message
	}
	if why == "" {
		return nil
	}
	return m.toast("the session that ended: " + why)
}

// gitRunning names the git operation running as the quit question does
// ("the commit", "the fetch"), "" when none runs: Source Control's, or the
// branches popup's switch or fetch.
func (m *Model) gitRunning() string {
	if m.op == nil && !m.branchOpRunning() {
		return ""
	}
	if m.ui.GitRunning != "" {
		return m.ui.GitRunning
	}
	return "the git operation"
}

// askReturn is Esc at the end of the dashboard's ladder, where the focus is
// already on zone B and nothing is open or selected (design D7): the
// question opens, No selected, unless the session cannot go back (a mock
// run, a start under way, a close under way) or something runs, which the
// input row then says, nothing written to the CLI (design D8).
func (m *Model) askReturn() []tea.Cmd {
	if render.Layers(m.s, m.ui).Top != render.LayerNone {
		// A popup or a question still open, the staging question among them,
		// whatever has the focus: Esc does what it did, nothing (design D7).
		return nil
	}
	if m.graph.pinned && m.ui.Graph.Card != nil {
		return nil // a Git Graph card pinned, which is no layer: the same (design D7)
	}
	git := m.gitRunning()
	cmds := m.do(state.AskReturn{Git: git})
	if m.s.CanReturn() && m.s.ReturnRefusal(git) == "" {
		m.ui.Return = render.ReturnAsk{Open: true}
	}
	return cmds
}

// returnKey takes every key while the question is open (design D7): ← and
// → select Yes or No, Enter answers, Esc is No; Ctrl+Q is the dashboard's
// before it comes here, and every other key does nothing.
func (m *Model) returnKey(k tea.KeyPressMsg) []tea.Cmd {
	switch k.String() {
	case "left":
		m.ui.Return.Yes = true
	case "right":
		m.ui.Return.Yes = false
	case "enter":
		return m.answerReturn(m.ui.Return.Yes)
	case "esc":
		return m.answerReturn(false)
	}
	return nil
}

// returnMouse takes the mouse while the question is open: a left click on a
// button answers with it, one outside the card is No; anything else does
// nothing.
func (m *Model) returnMouse(msg tea.MouseMsg) []tea.Cmd {
	click, ok := msg.(tea.MouseClickMsg)
	if !ok || click.Button != tea.MouseLeft || m.screen == nil {
		return nil
	}
	if p, item := m.screen.ItemAt(click.X, click.Y); p == render.ReturnPanel {
		switch item {
		case render.ReturnYes:
			return m.answerReturn(true)
		case render.ReturnNo:
			return m.answerReturn(false)
		}
	}
	if !m.screen.Return.Holds(click.X, click.Y) {
		return m.answerReturn(false)
	}
	return nil
}

// answerReturn closes the question; Yes asks the state for the return, which
// it refuses, saying why, if something has started to run meanwhile.
func (m *Model) answerReturn(yes bool) []tea.Cmd {
	m.ui.Return = render.ReturnAsk{}
	if !yes {
		return nil
	}
	return m.do(state.ReturnHome{Git: m.gitRunning()})
}

// goHome is the state's CmdHome: the session is over and the state is New's,
// its answers kept (design D9). The home screen comes at once: the session's
// part of Shutdown, which waits (the notification taken out, the Git Graph's
// git stopped, the turn review's snapshots ended and its store removed), and
// the draft's write and release for the project's next start are taken from
// the model and run in a command, off the event loop (sessionEnd); then
// closeSession (the windows, the attachments), the session's views and popups
// closed, the write queue's worker ended, and every field of the session
// given a fresh model's value (resetSession). The next start waits for the
// command (stillClosing). The registration lock follows the screen
// (lockSync), and the home screen's picture is drawn whole again, as after
// Back.
func (m *Model) goHome() []tea.Cmd {
	var end sessionEnd
	m.endNotifying(&end)
	m.endGraphRead(&end)
	m.endTurns(&end)
	m.endDraft(&end)
	m.closeSearch()
	m.closePopups()
	m.closePanels()
	m.closeDiff()
	cmds := m.closeSession()
	if m.writes != nil {
		close(m.writes) // the worker ends once the writes queued, the close among them, are done
		m.writes = nil
	}
	m.resetSession()
	m.focusMessage()
	return append(cmds, m.runEnd(end))
}

// returnGraceMsg is the quit's kill grace over during a return's close, a
// session's timer (sessionMsg).
type returnGraceMsg struct{}

// sessionEnd is what a session's end waits for, taken from the model on the
// event loop and run off it (plan 75): the notification still up taken out
// (2 s at most), the Git Graph's read stopped and waited for, the turn
// review's snapshot worker ended (3 s at most) and its store removed, the
// draft written and let go.
type sessionEnd struct {
	notifies  chan notifyJob
	unnotify  func() error
	reader    interface{ Close() }
	turnJobs  chan turnJob
	turnsDone chan struct{}
	turns     TurnStore
	draft     *draftStore
	save      bool
	draftGen  int
	text      string
	images    []string
	pasted    map[string][]byte
}

// endNotifying takes the notification queue from the model: stopNotifying
// off the event loop (plan 64, design D4).
func (m *Model) endNotifying(e *sessionEnd) {
	e.notifies = m.notifies
	if m.ping.notified && m.opt.Unnotify != nil {
		e.unnotify = m.opt.Unnotify
	}
	m.notifies, m.ping.notified = nil, false
}

// endGraphRead takes the Git Graph's read from the model, its pages to come
// dropped: stopGraphRead's wait off the event loop (plan 62, design D4).
func (m *Model) endGraphRead(e *sessionEnd) {
	g := &m.graph
	if g.reader != nil {
		e.reader = g.reader
		g.reader = nil
	}
	g.gen++
	g.reading = false
}

// endTurns takes the turn review from the model: its snapshots stopped, a
// store still being made removed as it comes (turnsGen), the worker's end
// and the store's removal off the event loop (plan 65, design D3). The next
// session makes its own store.
func (m *Model) endTurns(e *sessionEnd) {
	if m.turnsStop != nil {
		m.turnsStop()
	}
	e.turnJobs, e.turnsDone, e.turns = m.turnJobs, m.turnsDone, m.turns
	m.turns, m.turnJobs, m.turnsDone, m.turnsCtx, m.turnsStop, m.turnGate = nil, nil, nil, nil, nil, nil
	m.turnsOpening = false
	m.turnsGen++
}

// endDraft takes the draft from the model, what it holds now to be written
// and the draft let go off the event loop, for the project's next start
// (plan 33, designs D1 and D2).
func (m *Model) endDraft(e *sessionEnd) {
	if m.draft == nil {
		return
	}
	e.draft = m.draft
	if m.draftReady {
		m.draftGen++
		paths := m.storedPaths()
		e.save, e.draftGen, e.text = true, m.draftGen, m.storedText()
		e.images, e.pasted = m.draftEntries(paths)
	}
	m.draft, m.draftReady, m.draftTimer, m.unread = nil, false, false, nil
}

// run does what the session's end waits for; the draft's error, if any.
func (e sessionEnd) run() error {
	if e.notifies != nil {
		if e.unnotify != nil {
			j := notifyJob{run: e.unnotify, done: make(chan error, 1)}
			select {
			case e.notifies <- j:
				select {
				case <-j.done:
				case <-time.After(notifyClearWait):
				}
			default:
			}
		}
		close(e.notifies)
	}
	if e.reader != nil {
		e.reader.Close()
	}
	if e.turnJobs != nil {
		close(e.turnJobs)
		select {
		case <-e.turnsDone:
		case <-time.After(turnsCloseWait):
		}
	}
	if e.turns != nil {
		_ = e.turns.Close()
	}
	if e.draft == nil {
		return nil
	}
	var err error
	if e.save {
		err = e.draft.save(e.draftGen, e.text, e.images, e.pasted)
	}
	if rerr := e.draft.release(); err == nil {
		err = rerr
	}
	return err
}

// endJob is a session's end to run once, by its command or, should the
// program end before the command started, by Shutdown; prev is the end
// before it, run first.
type endJob struct {
	once sync.Once
	end  sessionEnd
	prev *endJob
	err  error
}

// run runs the end once, the one before it first; a second call waits for
// the first, which every wait of the end bounds (notifyClearWait,
// turnsCloseWait).
func (j *endJob) run() error {
	j.once.Do(func() {
		if j.prev != nil {
			j.prev.run()
			j.prev = nil // done: nothing kept of it
		}
		j.err = j.end.run()
	})
	return j.err
}

// sessionEndMsg is a session's end done, err the draft's error.
type sessionEndMsg struct {
	job *endJob
	err error
}

// runEnd runs a session's end in a command, after the one before it if it is
// still running; until it is done, m.ending says the last session is still
// closing, and Shutdown runs it, or waits for it, before the program ends.
func (m *Model) runEnd(e sessionEnd) tea.Cmd {
	j := &endJob{end: e, prev: m.ending}
	m.ending = j
	return func() tea.Msg { return sessionEndMsg{job: j, err: j.run()} }
}

// sessionEnded is a session's end done: the next start may go; the draft's
// failure in the toast.
func (m *Model) sessionEnded(msg sessionEndMsg) []tea.Cmd {
	if msg.job == m.ending {
		m.ending, m.closingSaid = nil, false
	}
	if msg.err != nil {
		return m.toast("the draft could not be saved: " + msg.err.Error())
	}
	return nil
}

// returningNote is what a git operation asked for while the return closes
// the session says: none starts, so that none is dropped unseen (plan 75).
const returningNote = "Returning to the home screen"

// stillClosingNote is what a start says while the last session's end runs.
const stillClosingNote = "The last session is still closing"

// stillClosing says the last session's end still runs: START SESSION, Ctrl+R
// and the history's resume then do nothing and say so, so that its draft and
// store are settled before a new session takes them.
func (m *Model) stillClosing() bool {
	if m.ending == nil {
		return false
	}
	m.closingSaid = true
	return true
}

// resetSession gives every field of the session a fresh model's value
// (criterion 75.14): the model is made anew, and what belongs to the
// application, not to a session, is kept. Kept: the options and the state
// (reset by the reducer), the startup screen's field, detection, the project
// list, the folder dialog and Alt+H's check, the registration lock (lockSync
// releases it), the session's end still running (ending), the timers
// already asked for (the spinner, the clock, the frames, the banner, the
// bell's limit), the terminal's input and images (pastes under way, images
// drawn, GDI's glyphs and labels, the home screen's picture, the toast's
// number), and the generations, each one past the session's, so that an
// answer to the session's reads is dropped. The interface's view is made
// anew, its caches with it, keeping the terminal's size and capabilities,
// the startup screen's places and notes and the toast.
func (m *Model) resetSession() {
	old := m.ui
	ui := m.newUI()
	ui.Width, ui.Height, ui.Now, ui.Spinner = old.Width, old.Height, old.Now, old.Spinner
	ui.Field, ui.Choice, ui.ProjectSel, ui.ProjectsProblem, ui.FolderProblem = old.Field, old.Choice, old.ProjectSel, old.ProjectsProblem, old.FolderProblem
	ui.Sixel, ui.PillNoFont, ui.NotifyProblem, ui.MouseOff, ui.Toast = old.Sixel, old.PillNoFont, old.NotifyProblem, old.MouseOff, old.Toast

	opt, s, path, env, detected, check := m.opt, m.s, m.path, m.env, m.detected, m.check
	projects, projectList, projectsGen, projectsSeen := m.projects, m.projectList, m.projectsGen, m.projectsSeen
	folderOpen, histRefused, histRefusedFor := m.folderOpen, m.histRefused, m.histRefusedFor
	lock, locking, ending, closingSaid := m.lock, m.locking, m.ending, m.closingSaid
	spinning, clocking, framing, bannering, blurred := m.spinning, m.clocking, m.framing, m.bannering, m.blurred
	pingLast, pingDue := m.ping.last, m.ping.due
	pasting, pasteN, ctrlVDown, pasteCount, lastPaste, pasteWait, waitPastes := m.pasting, m.pasteN, m.ctrlVDown, m.pasteCount, m.lastPaste, m.pasteWait, m.waitPastes
	drawn, send, kicks, pillLabels, liveLabel, toastGen := m.drawn, m.send, m.kicks, m.pillLabels, m.liveLabel, m.toastGen
	home, glyphs, homeSecond, resized, screen, repaint, motionDropped := m.home, m.glyphs, m.homeSecond, m.resized, m.screen, m.repaint, m.motionDropped
	// The generations, one past the session's.
	gens := [...]int{m.wtGen, m.histGen, m.fileGen, m.searchGen, m.finderGen, m.explorerGen, m.blameSeq, m.statusGen, m.pollGen,
		m.diffGen, m.folderSeq, m.opGen, m.opSeq, m.branchGen, m.headGen, m.graph.gen, m.graph.refsGen, m.graph.watchGen,
		m.draftGen, m.thumbGen, m.turnsGen, m.sessGen}
	for i := range gens {
		gens[i]++
	}

	*m = Model{}

	m.opt, m.s, m.ui, m.path, m.env, m.detected, m.check = opt, s, ui, path, env, detected, check
	m.projects, m.projectList, m.projectsGen, m.projectsSeen = projects, projectList, projectsGen, projectsSeen
	m.folderOpen, m.histRefused, m.histRefusedFor = folderOpen, histRefused, histRefusedFor
	m.lock, m.locking, m.ending, m.closingSaid = lock, locking, ending, closingSaid
	m.spinning, m.clocking, m.framing, m.bannering, m.blurred = spinning, clocking, framing, bannering, blurred
	m.ping.last, m.ping.due = pingLast, pingDue
	m.pasting, m.pasteN, m.ctrlVDown, m.pasteCount, m.lastPaste, m.pasteWait, m.waitPastes = pasting, pasteN, ctrlVDown, pasteCount, lastPaste, pasteWait, waitPastes
	m.drawn, m.send, m.kicks, m.pillLabels, m.liveLabel = drawn, send, kicks, pillLabels, liveLabel
	m.toastGen = toastGen // the toast on screen still goes when its time is up
	m.home, m.glyphs, m.homeSecond, m.resized, m.screen, m.repaint, m.motionDropped = home, glyphs, homeSecond, resized, screen, repaint, motionDropped
	m.wtGen, m.histGen, m.fileGen, m.searchGen, m.finderGen, m.explorerGen, m.blameSeq, m.statusGen, m.pollGen = gens[0], gens[1], gens[2], gens[3], gens[4], gens[5], gens[6], gens[7], gens[8]
	m.diffGen, m.folderSeq, m.opGen, m.opSeq, m.branchGen, m.headGen, m.graph.gen, m.graph.refsGen, m.graph.watchGen = gens[9], gens[10], gens[11], gens[12], gens[13], gens[14], gens[15], gens[16], gens[17]
	m.draftGen, m.thumbGen, m.turnsGen, m.sessGen = gens[18], gens[19], gens[20], gens[21]
}
