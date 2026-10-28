package app

import (
	"errors"
	"fmt"
	"image"
	"slices"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/Evengelius/agent-command-center/internal/input"
	"github.com/Evengelius/agent-command-center/internal/render"
)

// Source Control's commit controls (plan 45, brief amendment 46): the
// message field, the Commit button, its dropdown's Commit & Push and the
// question before staging (designs D2 to D12). Git's writes run in
// commands through input.SourceWrite, one step at a time, one operation at
// a time; the last step, or a failing one, reads git's status again in the
// same command, so the button changes on Git's word (design D11).

// The steps of an operation (designs D8 to D10), in their order: Commit &
// Push's destination checked, a selection checked before its staging (plan
// 50, design D7), the question's paths staged, the commit, the push.
const (
	stepTarget = "target"
	stepCheck  = "check"
	stepStage  = "stage"
	stepCommit = "commit"
	stepPush   = "push"
)

// gitOp is the operation running: its steps left, what they take, and what
// the done ones gave.
type gitOp struct {
	steps  []string
	stage  []string
	expect []input.GitEntry
	// sel is a selection's files, which its commit commits alone (plan 50,
	// design D7); nil, the whole index. staged says the stage step was done.
	sel     []input.GitEntry
	staged  bool
	message string
	push    bool // Commit & Push, or Push alone
	commit  *input.CommitResult
	pushed  *input.PushResult
	// The elapsed time and the push's stop (design D15): when it started,
	// its ticks' sequence, the stop its push takes, a stop asked, and why
	// the last one could not be.
	started    time.Time
	seq        int
	stop       *input.GitStop
	cancelling bool
	hint       string
}

// opTickMsg is the second's tick of the operation seq, which redraws its
// elapsed time (design D15).
type opTickMsg struct{ seq int }

// opTick is the tick a second apart.
const opTick = time.Second

// opMsg is a step's end: its error, what it gave, and, after the last step
// or a failure, git's status read again.
type opMsg struct {
	gen    int
	step   string
	err    error
	commit *input.CommitResult
	pushed *input.PushResult
	read   bool
	snap   input.GitSnapshot
	snapE  error
}

// asked is the question's offer (design D6): the CHANGES entries to stage,
// for Commit & Push or not; or a selection's (plan 50, design D7), its files
// and those of them to stage.
type asked struct {
	entries []input.GitEntry
	push    bool
	sel     []input.GitEntry
	stage   []string
}

// sourceWrite is the Source Control's writes, when the git in use has them.
func (m *Model) sourceWrite() (input.SourceWrite, bool) {
	sw, ok := m.git().(input.SourceWrite)
	return sw, ok
}

// busyLabel is the button's label while step runs (design D3).
func busyLabel(step string) string {
	switch step {
	case stepStage:
		return "Staging…"
	case stepPush:
		return "Pushing…"
	}
	return "Committing…"
}

// commitState is the button's action and whether it has nothing to do,
// from the snapshot read last (design D7): Push while the branch is ahead
// of an upstream that exists, else Commit, dimmed with no change.
func (m *Model) commitState() (action string, dim bool) {
	if m.snap == nil {
		return render.ActionCommit, true
	}
	if len(m.ui.Source.Selected) > 0 {
		return render.ActionCommit, false // a selection's, the branch ahead or not (plan 50, design D6)
	}
	if b := m.snap.Branch; b.Upstream != "" && !b.Gone && b.Ahead > 0 {
		return render.ActionPush, false
	}
	return render.ActionCommit, len(m.snap.Entries) == 0
}

// syncCommit copies the controls' state into the view (syncUI).
func (m *Model) syncCommit() {
	v := &m.ui.Source
	v.Action, v.Dim = m.commitState()
	v.Busy, v.BusyHint, v.BusyWarn = "", "", false
	m.ui.GitRunning = ""
	if op := m.op; op != nil && len(op.steps) > 0 {
		// What the quit question names (plan 47, design D1).
		m.ui.GitRunning = map[string]string{stepCheck: "the commit", stepStage: "the staging", stepCommit: "the commit", stepPush: "the push", stepTarget: "the push"}[op.steps[0]]
	}
	if op := m.op; op != nil && len(op.steps) > 0 {
		// The running step, its elapsed time from its second second; a push
		// says how to cancel it, or why a stop could not be (design D15).
		step, d := op.steps[0], m.ui.Now.Sub(op.started)
		v.Busy = busyLabel(step)
		switch {
		case op.cancelling:
			v.Busy = "Cancelling…"
		case d >= 2*time.Second:
			v.Busy += " " + elapsed(d)
		}
		switch {
		case op.hint != "":
			v.BusyHint, v.BusyWarn = op.hint, true
		case step == stepPush && !op.cancelling && d >= 2*time.Second:
			v.BusyHint = "Esc cancels the push."
		}
	}
	if v.Repo == "" {
		v.Place, v.Menu, v.MenuHot, v.Ask = "", false, false, 0 // no controls drawn
	}
	v.Msg = m.msgView()
}

// elapsed is an operation's time as its label shows it: "12 s", then
// "1:05" past a minute.
func elapsed(d time.Duration) string {
	s := int(d / time.Second)
	if s < 60 {
		return fmt.Sprintf("%d s", s)
	}
	return fmt.Sprintf("%d:%02d", s/60, s%60)
}

// opTicked redraws a running operation's elapsed time and asks the next
// tick; a tick of an operation that ended does nothing.
func (m *Model) opTicked(msg opTickMsg) []tea.Cmd {
	if m.op == nil || msg.seq != m.op.seq {
		return nil
	}
	return []tea.Cmd{m.opt.After(opTick, opTickMsg{seq: msg.seq})}
}

// cancelPush asks the running push to stop (design D15): "Cancelling…" once
// asked, else why it cannot be now.
func (m *Model) cancelPush() {
	op := m.op
	if op == nil || op.cancelling {
		return
	}
	ok, why := op.stop.Stop()
	switch {
	case ok:
		op.cancelling, op.hint = true, ""
	case why != "":
		op.hint = why
	}
}

// pushing says the running step is a push, which Esc can stop.
func (m *Model) pushing() bool {
	return m.op != nil && m.op.stop != nil && len(m.op.steps) > 0 && m.op.steps[0] == stepPush
}

// msgView is the message field as the frame draws it: zone B's editor's
// view of m.msg (editorView).
func (m *Model) msgView() render.EditorView {
	e := &m.msg
	cur := e.CursorLine()
	first := max(cur-msgWindow, 0)
	v := render.EditorView{Lines: e.LineRange(first, 2*msgWindow), First: first, Total: e.Lines(), Line: cur}
	text := e.Text()
	v.Col = e.Cursor() - (strings.LastIndexByte(text[:e.Cursor()], '\n') + 1)
	if from, to, ok := e.Selection(); ok {
		v.From, v.To = linePos(text, from), linePos(text, to)
	}
	return v
}

// msgWindow is the logical lines around the cursor the field's view holds:
// it shows three rows at most.
const msgWindow = 8

// notice puts the note under the button (design D12).
func (m *Model) notice(text string, kind byte) {
	m.ui.Source.Notice, m.ui.Source.NoticeKind = text, kind
}

// noticeErr is a step's error as the note says it: a refusal as a
// validation, git's failure as a failure.
func (m *Model) noticeErr(prefix string, err error) {
	var r input.Refusal
	if errors.As(err, &r) && prefix == "" {
		m.notice(r.Msg, render.NoticeCheck)
		return
	}
	m.notice(prefix+err.Error(), render.NoticeFailed)
}

// mainAction runs the button's action (design D3): Push, or Commit.
func (m *Model) mainAction() []tea.Cmd {
	if m.ui.Source.Action == render.ActionPush {
		return m.pushAction()
	}
	return m.commitAction(false)
}

// commitAction is Commit, or Commit & Push with push (designs D6, D8 and
// D10): the message, then the changes, checked before any git call; with
// nothing staged, the question.
func (m *Model) commitAction(push bool) []tea.Cmd {
	v := &m.ui.Source
	if m.op != nil {
		return nil // one operation at a time (design D11)
	}
	m.closePopups() // the question, when it comes, the one popup open (plan 49, design D7)
	if strings.TrimSpace(m.msg.Text()) == "" {
		m.notice("Enter a commit message.", render.NoticeCheck)
		return nil
	}
	if m.snap == nil {
		return nil
	}
	var staged, changes []input.GitEntry
	for _, e := range m.snap.Entries {
		switch {
		case input.ConflictName(e.X, e.Y) != "":
			m.notice("Resolve the merge conflicts first.", render.NoticeCheck)
			return nil
		case e.X == '?' && e.Y == '?':
			changes = append(changes, e)
		default:
			if e.X != ' ' {
				staged = append(staged, e)
			}
			if e.Y != ' ' {
				changes = append(changes, e)
			}
		}
	}
	if sel := m.selection(); len(sel) > 0 {
		return m.commitSelected(sel, push)
	}
	switch {
	case len(staged) > 0:
		return m.startOp(&gitOp{steps: commitSteps(push, false), expect: staged, message: m.msg.Text(), push: push})
	case len(changes) == 0:
		m.notice("No changes to commit.", render.NoticeCheck)
		return nil
	}
	m.ask = &asked{entries: changes, push: push}
	v.Ask, v.AskSel, v.AskPush, v.AskNo, v.Notice = len(changes), 0, push, false, ""
	return nil
}

// selection is the selected files as git's status last read them, sorted by
// path, a file of an untracked folder as untracked (plan 50, design D1).
func (m *Model) selection() []input.GitEntry {
	if len(m.ui.Source.Selected) == 0 || m.snap == nil {
		return nil
	}
	paths := make([]string, 0, len(m.ui.Source.Selected))
	for p := range m.ui.Source.Selected {
		paths = append(paths, p)
	}
	slices.Sort(paths)
	var sel []input.GitEntry
	for _, p := range paths {
		n := len(sel)
		for _, e := range m.snap.Entries {
			if e.Path == p {
				sel = append(sel, e)
			}
		}
		if len(sel) == n {
			sel = append(sel, input.GitEntry{X: '?', Y: '?', Path: p, RepoPath: m.snap.Prefix + p})
		}
	}
	return sel
}

// commitSelected is Commit, or Commit & Push, of the selection sel (plan 50,
// design D7): a file a commit cannot take whole refused at once, by the
// panel's status; files not staged, the question first; else the commit.
func (m *Model) commitSelected(sel []input.GitEntry, push bool) []tea.Cmd {
	var stage []string
	for i := 0; i < len(sel); {
		j := i + 1
		for j < len(sel) && sel[j].Path == sel[i].Path {
			j++ // a path's entries, side by side in sel
		}
		if err := input.WholeRefusal(sel[i].Path, sel[i:j]); err != nil {
			m.notice(err.Error(), render.NoticeCheck)
			return nil
		}
		if e := sel[i]; e.X == ' ' || e.X == '?' {
			stage = append(stage, e.Path)
		}
		i = j
	}
	if len(stage) > 0 {
		v := &m.ui.Source
		m.ask = &asked{push: push, sel: sel, stage: stage}
		v.Ask, v.AskSel, v.AskPush, v.AskNo, v.Notice = len(stage), len(v.Selected), push, false, ""
		return nil
	}
	return m.startOp(&gitOp{steps: selectionSteps(push, false), sel: sel, message: m.msg.Text(), push: push})
}

// selectionSteps are a selection's commit's steps (plan 50, design D7):
// with files to stage, a check before the staging, the only write before
// the commit.
func selectionSteps(push, stage bool) []string {
	steps := commitSteps(push, stage)
	if stage {
		steps = slices.Insert(steps, slices.Index(steps, stepStage), stepCheck)
	}
	return steps
}

// stayStaged says the changes a question's staging staged stay staged once
// the commit after it failed (plan 50, design D8; amendment 51).
func stayStaged(n int) string {
	if n == 1 {
		return "The change staged for it stays staged."
	}
	return fmt.Sprintf("The %d changes staged for it stay staged.", n)
}

// commitSteps are a commit's steps: the push's destination first for
// Commit & Push (design D10), the staging after the question's yes.
func commitSteps(push, stage bool) []string {
	var steps []string
	if push {
		steps = append(steps, stepTarget)
	}
	if stage {
		steps = append(steps, stepStage)
	}
	steps = append(steps, stepCommit)
	if push {
		steps = append(steps, stepPush)
	}
	return steps
}

// answer takes the question's answer (design D6): yes stages exactly the
// offered paths, then commits; no does nothing.
func (m *Model) answer(yes bool) []tea.Cmd {
	a := m.ask
	m.ask, m.ui.Source.Ask, m.ui.Source.AskSel = nil, 0, 0
	if !yes || a == nil || m.op != nil {
		return nil
	}
	if strings.TrimSpace(m.msg.Text()) == "" {
		m.notice("Enter a commit message.", render.NoticeCheck)
		return nil
	}
	if a.sel != nil {
		// A selection's (plan 50, design D7): checked, then exactly its
		// files not staged staged, then committed alone.
		return m.startOp(&gitOp{steps: selectionSteps(a.push, true), stage: a.stage, sel: a.sel, message: m.msg.Text(), push: a.push})
	}
	paths := make([]string, len(a.entries))
	for i, e := range a.entries {
		paths[i] = e.Path
	}
	return m.startOp(&gitOp{steps: commitSteps(a.push, true), stage: paths, expect: a.entries, message: m.msg.Text(), push: a.push})
}

// pushAction is Push (design D9).
func (m *Model) pushAction() []tea.Cmd {
	if m.op != nil {
		return nil
	}
	m.ui.Source.Menu, m.ui.Source.MenuHot = false, false
	return m.startOp(&gitOp{steps: []string{stepPush}, push: true})
}

// startOp starts an operation at its first step, the note cleared.
func (m *Model) startOp(op *gitOp) []tea.Cmd {
	if _, ok := m.sourceWrite(); !ok {
		return nil
	}
	m.op = op
	m.notice("", 0)
	m.opSeq++
	op.started, op.seq = m.ui.Now, m.opSeq
	if slices.Contains(op.steps, stepPush) {
		op.stop = input.NewGitStop() // its push can be stopped (design D15)
	}
	return []tea.Cmd{m.runStep(), m.opt.After(opTick, opTickMsg{seq: op.seq})}
}

// runStep runs the operation's next step in a command; the last one, or a
// failing one, reads git's status after it.
func (m *Model) runStep() tea.Cmd {
	sw, _ := m.sourceWrite()
	sg, _ := m.sourceGit()
	op := m.op
	m.opGen++
	gen, step, dir, last := m.opGen, op.steps[0], m.fileRoot(), len(op.steps) == 1
	stage, expect, message, stop, sel := op.stage, op.expect, op.message, op.stop, op.sel
	return func() tea.Msg {
		out := opMsg{gen: gen, step: step}
		switch step {
		case stepTarget:
			_, out.err = sw.PushTarget(dir)
		case stepCheck:
			out.err = sw.CheckSelection(dir, sel)
		case stepStage:
			out.err = sw.Stage(dir, stage)
		case stepCommit:
			var r input.CommitResult
			if sel != nil {
				// A commit made other than the selection is still made (plan
				// 50, design D7).
				var u input.Unexpected
				if r, out.err = sw.CommitSelection(dir, message, sel, stage); out.err == nil || errors.As(out.err, &u) {
					out.commit = &r
				}
			} else if r, out.err = sw.Commit(dir, message, expect); out.err == nil {
				out.commit = &r
			}
		case stepPush:
			var r input.PushResult
			if r, out.err = sw.Push(dir, stop); out.err == nil {
				out.pushed = &r
			}
		}
		if (last || out.err != nil) && sg != nil {
			out.read = true
			out.snap, out.snapE = sg.Snapshot(dir)
		}
		return out
	}
}

// stepDone takes a step's end: the next step, or the operation's end, its
// status read put in the panel before its result (design D11).
func (m *Model) stepDone(msg opMsg) []tea.Cmd {
	op := m.op
	if op == nil || msg.gen != m.opGen {
		return nil
	}
	if msg.commit != nil {
		op.commit = msg.commit
	}
	if msg.pushed != nil {
		op.pushed = msg.pushed
	}
	if msg.step == stepStage && msg.err == nil {
		op.staged = true
	}
	op.steps = op.steps[1:]
	if msg.err == nil && len(op.steps) > 0 {
		if msg.step == stepCommit {
			m.clearCommitted(op.message) // the commit made: the field cleared before the push (design D10)
			m.clearSelection(op)
		}
		return []tea.Cmd{m.runStep()}
	}
	m.op = nil
	var cmds []tea.Cmd
	if msg.read {
		// This read is newer than any under way: theirs are dropped.
		m.statusGen++
		m.statusBusy, m.statusAgain = false, false
		m.applySnapshot(msg.snap, msg.snapE)
		cmds = append(cmds, m.readFolders()...) // the opened folders too (plan 46, design D3)
		cmds = append(cmds, m.armPoll())
		if m.ui.Diff.Open && msg.snapE == nil {
			cmds = append(cmds, m.compareOpen(true))
		}
	}
	if op.commit != nil {
		m.clearCommitted(op.message)
		m.clearSelection(op)
	}
	switch {
	case msg.err != nil && msg.step == stepPush && op.commit != nil && cancelled(msg.err):
		m.notice(fmt.Sprintf("Committed %s; %s", op.commit.Short, lowerFirst(msg.err.Error())), render.NoticeCheck)
	case msg.err != nil && msg.step == stepPush && op.commit != nil:
		m.notice(fmt.Sprintf("Committed %s, but the push failed: %s", op.commit.Short, msg.err.Error()), render.NoticeFailed)
	case msg.err != nil:
		m.noticeErr("", msg.err)
		if msg.step == stepCommit && op.staged && op.commit == nil {
			// The question's staging is not undone (plan 50, design D8): git's
			// words, a full stop if they end with none, then that.
			n := strings.TrimRight(m.ui.Source.Notice, " ")
			if n != "" && !strings.ContainsAny(n[len(n)-1:], ".!?:") {
				n += "."
			}
			m.ui.Source.Notice = n + " " + stayStaged(len(op.stage))
		}
	case op.commit != nil && op.pushed != nil:
		m.notice(fmt.Sprintf("Committed %s and pushed to %s.", op.commit.Short, op.pushed.Target.Upstream), render.NoticeDone)
	case op.commit != nil:
		m.notice(fmt.Sprintf("Committed %s.", op.commit.Short), render.NoticeDone)
	case op.pushed != nil:
		m.notice(fmt.Sprintf("Pushed to %s.", op.pushed.Target.Upstream), render.NoticeDone)
	}
	return cmds
}

// cancelled says err is a push the owner stopped.
func cancelled(err error) bool {
	var r input.Refusal
	return errors.As(err, &r) && r.Kind == "cancelled"
}

// lowerFirst is s with its first letter in lower case.
func lowerFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToLower(s[:1]) + s[1:]
}

// clearSelection clears the selection a commit was made from (plan 50,
// design D9).
func (m *Model) clearSelection(op *gitOp) {
	if op.sel != nil {
		m.ui.Source.Selected = nil
	}
}

// clearCommitted clears the field once Git has confirmed the commit made
// from it, if it still holds the text committed (design D2).
func (m *Model) clearCommitted(message string) {
	if m.msg.Text() == message {
		m.msg.Clear()
	}
}

// msgFocused says the message field takes the keys and pastes.
func (m *Model) msgFocused() bool {
	return m.ui.Focus == "R" && m.ui.Source.Open && m.ui.Source.Place == render.PlaceMsg && m.ui.Source.Repo != ""
}

// controlsShown says the panel draws the commit controls (design D1).
func (m *Model) controlsShown() bool { return m.ui.Source.Open && m.ui.Source.Repo != "" }

// commitKey handles a key while the commit controls have the panel's keys:
// the dropdown's, the question's, the field's or the button's (designs D2
// to D4, D6). done is false for a key they pass on.
func (m *Model) commitKey(k tea.KeyPressMsg) (cmds []tea.Cmd, done bool) {
	v := &m.ui.Source
	s := k.String()
	if s == "esc" && m.pushing() {
		m.cancelPush() // Esc stops a running push (design D15); a commit runs on
		return nil, true
	}
	switch {
	case v.Menu:
		switch s {
		case "enter", "space":
			return m.commitAction(true), true
		case "esc":
			v.Menu, v.MenuHot = false, false
			return nil, true
		case "up", "down":
			v.MenuHot = true
			return nil, true
		}
		v.Menu, v.MenuHot = false, false // any other key closes it and goes on
	case v.Ask > 0:
		switch s {
		case "left", "right", "tab", "shift+tab":
			v.AskNo = !v.AskNo
			return nil, true
		case "enter", "space":
			return m.answer(!v.AskNo), true
		case "esc":
			return m.answer(false), true
		}
		if globalKey(k) {
			return nil, false
		}
		return nil, true
	}
	if s == "ctrl+enter" && (v.Place != "" || m.controlsShown()) {
		return m.mainAction(), true
	}
	switch v.Place {
	case render.PlaceMsg:
		return m.msgKey(k)
	case render.PlaceBtn:
		return m.btnKey(k)
	}
	return nil, false
}

// globalKey is a key the panel passes on to the dashboard: the function
// keys, the zones', the selectors' and the branches popup's (plan 59), and
// the window's.
func globalKey(k tea.KeyPressMsg) bool {
	switch k.String() {
	case "f1", "f2", "f3", "f4", "f6", "shift+f6", "tab", "shift+tab", "alt+p", "alt+b", "alt+g", "f12", "ctrl+o", "ctrl+r",
		"shift+pgup", "shift+pgdown", "ctrl+q":
		return true
	}
	return false
}

// msgKey handles a key in the message field (design D2): zone B's editor's
// keys, Enter a new line; ↓ from the last line to the button, Esc back to
// zone B. Editing the message clears the note.
func (m *Model) msgKey(k tea.KeyPressMsg) (cmds []tea.Cmd, done bool) {
	e := &m.msg
	before := e.Text()
	defer func() {
		if e.Text() != before {
			m.notice("", 0)
		}
	}()
	switch s := k.String(); s {
	case "ctrl+c", "ctrl+shift+c":
		if e.HasSelection() {
			return m.copy(e.SelectedText()), true
		}
		return nil, false
	case "esc":
		if e.HasSelection() {
			e.ClearSelection()
		} else {
			m.ui.Focus = "B" // the message kept (design D2)
		}
	case "enter", "shift+enter", "ctrl+j", "alt+enter":
		e.Newline()
	case "down":
		if e.CursorLine() >= e.Lines()-1 {
			e.ClearSelection()
			m.ui.Source.Place = render.PlaceBtn
		} else {
			e.Move(input.Down, false)
		}
	case "ctrl+a":
		e.SelectAll()
	case "ctrl+z":
		e.Undo()
	case "ctrl+y":
		e.Redo()
	case "ctrl+x":
		if e.HasSelection() {
			return m.copy(e.Cut()), true
		}
	case "ctrl+backspace", "alt+backspace", "ctrl+h":
		e.DeleteWordLeft()
	case "ctrl+delete", "alt+delete":
		e.DeleteWordRight()
	case "alt+v":
		// No image in a commit message.
	default:
		if globalKey(k) {
			return nil, false
		}
		if mo, extend, ok := motion(k); ok {
			e.Move(mo, extend)
			return nil, true
		}
		return m.editKey(e, k, false), true
	}
	return nil, true
}

// btnKey handles a key on the button (design D3).
func (m *Model) btnKey(k tea.KeyPressMsg) (cmds []tea.Cmd, done bool) {
	v := &m.ui.Source
	switch k.String() {
	case "enter", "space":
		return m.mainAction(), true
	case "right", "alt+down":
		if m.op == nil {
			m.closePopups() // the one popup open (plan 49, design D7)
			v.Menu, v.MenuHot = true, true
		}
	case "up":
		v.Place = render.PlaceMsg
	case "down":
		v.Place = ""
		rows := render.SourceRows(*v)
		if i := render.SourceCursor(rows, ""); i >= 0 {
			v.Cursor, v.Top = rows[i].Key, 0
		}
	case "esc":
		m.ui.Focus = "B"
	default:
		if globalKey(k) || k.String() == "ctrl+c" || k.String() == "ctrl+shift+c" {
			return nil, false
		}
	}
	return nil, true
}

// commitClick takes a click on the controls' items (designs D2 to D4, D6);
// done is false for another item.
func (m *Model) commitClick(item, x, y int) (cmds []tea.Cmd, done bool) {
	v := &m.ui.Source
	switch item {
	case render.ItemMsg, render.ItemMsgText, render.ItemMsgHolder:
		v.Place = render.PlaceMsg
		if ln, off, ok := m.screen.MsgAt(x, y); ok {
			m.msg.Select(m.msg.Offset(ln, off), m.msg.Offset(ln, off))
		} else if item != render.ItemMsgText {
			m.msg.Move(input.BufferEnd, false)
		}
		return nil, true
	case render.ItemCommitMain:
		return m.mainAction(), true
	case render.ItemCommitMore:
		if m.click.n <= 1 && m.op == nil {
			open := !v.Menu
			m.closePopups() // the one popup open (plan 49, design D7)
			v.Menu = open
		}
		return nil, true
	case render.ItemCommitMenu:
		if m.screen.MenuItemAt(x, y) {
			return m.commitAction(true), true
		}
		return nil, true // its outline
	case render.ItemAskYes:
		return m.answer(true), true
	case render.ItemAskNo:
		return m.answer(false), true
	case render.ItemNotice:
		return nil, true
	}
	return nil, false
}

// menuHotAt says the pointer at (x, y) is on the dropdown's item, its
// middle row, while it is open (design D4).
func (m *Model) menuHotAt(x, y int) bool {
	return m.ui.Source.Menu && m.screen != nil && m.screen.MenuItemAt(x, y)
}

// countClick counts a press for double clicks as press does, for the
// controls' own clicks.
func (m *Model) countClick(x, y int) {
	now := m.opt.Now()
	if m.click.n > 0 && now.Sub(m.click.at) <= multiClick && x == m.click.x && y == m.click.y {
		m.click.n = m.click.n%3 + 1
	} else {
		m.click.n = 1
	}
	m.click.at, m.click.x, m.click.y = now, x, y
}

// commitSequence writes the commit controls' images into draw (design D5),
// each over its cells written first, as the toggle's: their corners are
// transparent, where an image drawn before may lie. It gives where they now
// lie. The button and the dropdown need their labels' font; without it the
// next frame draws them in text.
func (m *Model) commitSequence(draw *strings.Builder) []render.Thumb {
	var now []render.Thumb
	region := func(x, y, w, h int) {
		for r := range h {
			fmt.Fprintf(draw, "\x1b[%d;%dH%s", y+1+r, x+1, m.screen.Region(x, y+r, w))
		}
	}
	if cm := m.screen.Commit; cm.Shown {
		if label, err := m.textLabel(cm.Look.Label, render.LabelTop, cm.Look.Live); err != nil {
			m.ui.PillNoFont = true
		} else {
			region(cm.X, cm.Y, cm.Look.Cells, render.BtnRows)
			fmt.Fprintf(draw, "\x1b[%d;%dH%s", cm.Y+1, cm.X+1, render.CommitSixel(cm.Look, label))
			now = append(now, render.Thumb{Item: -8, X: cm.X, Y: cm.Y, W: cm.Look.Cells, H: render.BtnRows})
		}
	}
	if mn := m.screen.Menu; mn.Shown {
		if label, err := m.textLabel(render.MenuLabel, render.MenuTextTop, false); err != nil {
			m.ui.PillNoFont = true
		} else {
			region(mn.X, mn.Y, mn.Look.Cells, render.MenuRows)
			fmt.Fprintf(draw, "\x1b[%d;%dH%s", mn.Y+1, mn.X+1, render.MenuSixel(mn.Look, label))
			now = append(now, render.Thumb{Item: -9, X: mn.X, Y: mn.Y, W: mn.Look.Cells, H: render.MenuRows})
		}
	}
	if mb := m.screen.MsgBox; mb.Shown {
		six, w, rows := render.MsgSixels(mb.Look), mb.Look.Cells, mb.Look.Rows
		places := [4]render.Thumb{
			{Item: -10, X: mb.X, Y: mb.Y, W: w, H: 1},
			{Item: -11, X: mb.X, Y: mb.Y + rows + 1, W: w, H: 1},
			{Item: -12, X: mb.X, Y: mb.Y + 1, W: 1, H: rows},
			{Item: -13, X: mb.X + w - 1, Y: mb.Y + 1, W: 1, H: rows},
		}
		for i, p := range places {
			region(p.X, p.Y, p.W, p.H)
			fmt.Fprintf(draw, "\x1b[%d;%dH%s", p.Y+1, p.X+1, six[i])
			now = append(now, p)
		}
	}
	return now
}

// textLabel is a control's label as GDI draws it in Cascadia Mono at the
// labels' em, top pixels down in an image as high as the button: drawn
// once a label; a live one, a running operation's, whose seconds change
// every second, kept only while it shows (design D15).
func (m *Model) textLabel(text string, top int, live bool) (*image.Alpha, error) {
	key := fmt.Sprintf("%s\x00%d", text, top)
	if l, ok := m.pillLabels[key]; ok {
		return l, nil
	}
	if live && m.liveLabel.mask != nil && m.liveLabel.key == key {
		return m.liveLabel.mask, nil
	}
	l, err := input.TextMask(text, render.LabelEm, top, render.BtnRows*render.SixelCellH)
	if err != nil {
		return nil, err
	}
	if live {
		m.liveLabel.key, m.liveLabel.mask = key, l
		return l, nil
	}
	if m.pillLabels == nil {
		m.pillLabels = map[string]*image.Alpha{}
	}
	m.pillLabels[key] = l
	return l, nil
}
