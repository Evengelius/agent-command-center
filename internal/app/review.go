package app

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/Evengelius/agent-command-center/internal/diff"
	"github.com/Evengelius/agent-command-center/internal/input"
	"github.com/Evengelius/agent-command-center/internal/provider"
	"github.com/Evengelius/agent-command-center/internal/render"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// The turn review (plan 65, brief amendment 67): the session's store of the
// project's snapshots, made off the first frame's path; the snapshots taken
// in an ordered queue of their own, when the state asks for them; the
// owner's message written once its turn's start is taken, or 5 s after the
// send at most (design D4); a turn's files read once both its snapshots
// are; a file's change of the turn opened in the diff view (design D9).

// TurnStore is the turn review's store, input.TurnStore; tests pass a fake.
type TurnStore interface {
	Take(ctx context.Context) (input.TurnTree, error)
	Changing(ctx context.Context) ([]string, error)
	Files(from, to string) (input.CommitFiles, error)
	Sides(from, to string, f input.CommitFile) (input.DiffPair, error)
	Close() error
}

const (
	// turnStartLimit is how long the owner's message waits for its turn's
	// start, from the send (design D4); a snapshot not taken by then is not
	// taken, and the turn has no review.
	turnStartLimit = 5 * time.Second
	// turnEndLimit is how long a turn's end snapshot may take: no message
	// waits for it.
	turnEndLimit = 30 * time.Second
	// turnQueue is how many snapshots may wait; one more is refused, its
	// turn without a review, rather than the screen blocked.
	turnQueue = 16
	// turnsCloseWait is how long the quit waits for a snapshot running to
	// stop before the store is removed.
	turnsCloseWait = 3 * time.Second
)

// turnJob is one snapshot and the message its result becomes; fail is its
// message when it panics.
type turnJob struct {
	run  func() tea.Msg
	fail func(problem string) tea.Msg
	done chan tea.Msg
}

// turnGate is the start's snapshot the owner's message waits for: open once
// it is taken or refused, its deadline the send's time and turnStartLimit.
type turnGate struct {
	done     chan struct{}
	once     sync.Once
	deadline time.Time
}

// open lets the message go; twice is once.
func (g *turnGate) open() { g.once.Do(func() { close(g.done) }) }

// wait holds the message until the snapshot is taken or the deadline; nil
// waits for nothing.
func (g *turnGate) wait() {
	if g == nil {
		return
	}
	t := time.NewTimer(time.Until(g.deadline))
	defer t.Stop()
	select {
	case <-g.done:
	case <-t.C:
	}
}

// turnStoreMsg is the session's store made, or why not.
type turnStoreMsg struct {
	store TurnStore
	err   error
}

// openTurns makes the session's store in a command, once: its first call in
// a process copies the repository's index and reads git's configuration
// (design D2).
func (m *Model) openTurns() tea.Cmd {
	open, dir := m.opt.TurnStore, m.fileRoot()
	if open == nil || m.turns != nil || m.turnsOpening {
		return nil
	}
	m.turnsOpening = true
	return func() tea.Msg {
		st, err := open(dir)
		return turnStoreMsg{store: st, err: err}
	}
}

// turnStoreRead keeps the session's store, or tells the state why there is
// none: nothing outside a git repository or without git (design D6).
func (m *Model) turnStoreRead(msg turnStoreMsg) []tea.Cmd {
	m.turnsOpening = false
	if msg.err != nil || msg.store == nil {
		var nr input.NotRepoError
		if msg.err == nil || errors.As(msg.err, &nr) || errors.Is(msg.err, input.ErrNoGit) {
			return m.do(state.ReviewStore{})
		}
		return m.do(state.ReviewStore{Problem: "the review's store could not be made: " + msg.err.Error()})
	}
	if m.turnsClosed {
		_ = msg.store.Close() // the dashboard quit meanwhile
		return nil
	}
	m.turns = msg.store
	m.turnsCtx, m.turnsStop = context.WithCancel(context.Background())
	return m.do(state.ReviewStore{On: true})
}

// turnCall queues a snapshot after every one queued before it, so that a
// turn's end is taken before the next turn's start (design D4); fail is its
// turn's answer when it cannot run or panics.
func (m *Model) turnCall(run func() tea.Msg, fail func(problem string) tea.Msg) (cmd tea.Cmd, queued bool) {
	if m.turnJobs == nil {
		m.turnJobs, m.turnsDone = make(chan turnJob, turnQueue), make(chan struct{})
		go func(jobs <-chan turnJob, done chan<- struct{}) {
			defer close(done)
			for j := range jobs {
				j.done <- runTurn(j)
			}
		}(m.turnJobs, m.turnsDone)
	}
	j := turnJob{run: run, fail: fail, done: make(chan tea.Msg, 1)}
	select {
	case m.turnJobs <- j:
	default:
		return func() tea.Msg { return fail("too many snapshots waiting") }, false
	}
	return func() tea.Msg { return <-j.done }, true
}

// runTurn runs one snapshot; a panic in it is its turn's problem, said in
// its row, never the session's or the program's end (plan 48, design D5).
func runTurn(j turnJob) (msg tea.Msg) {
	defer func() {
		if v := recover(); v != nil {
			msg = j.fail("the turn review failed: " + provider.Recovered(v))
		}
	}()
	return j.run()
}

// turnSnapshot takes the snapshot the state asks for, turn c.Turn's start
// or its end (designs D4 and D5). The start's is the owner's message's
// gate when a write follows it: the next CmdWrite waits for it.
func (m *Model) turnSnapshot(c state.Command) tea.Cmd {
	turn, end := c.Turn, c.End
	answer := func(problem string) tea.Msg {
		return actionMsg{state.TurnSnapshot{Turn: turn, End: end, Problem: problem}}
	}
	store, base := m.turns, m.turnsCtx
	if store == nil {
		return func() tea.Msg { return answer("the review's store is not open") }
	}
	limit, gate := turnEndLimit, (*turnGate)(nil)
	if !end {
		limit = turnStartLimit
		gate = &turnGate{done: make(chan struct{}), deadline: time.Now().Add(turnStartLimit)}
		m.turnGate = gate
	}
	deadline := time.Now().Add(limit)
	cmd, queued := m.turnCall(func() tea.Msg {
		if gate != nil {
			defer gate.open()
		}
		ctx, cancel := context.WithDeadline(base, deadline)
		defer cancel()
		if ctx.Err() != nil {
			// Queued behind a slow snapshot past its turn's limit: the message
			// went without it, and none is taken now.
			return answer(snapshotProblem(ctx.Err(), end))
		}
		tree, err := store.Take(ctx)
		if err != nil {
			return answer(snapshotProblem(err, end))
		}
		snap := state.TurnSnapshot{Turn: turn, End: end, Tree: tree.Tree, Left: reviewLeft(tree.Left)}
		if end {
			// The listing once more: a file written while the snapshot was
			// taken or just after it (design D5).
			if ch, err := store.Changing(ctx); err == nil && len(ch) > 0 {
				snap.Changing = true
			}
		}
		return actionMsg{snap}
	}, answer)
	if !queued && gate != nil {
		gate.open() // refused: the message goes at once
	}
	return cmd
}

// snapshotProblem is why a turn has no review, said in its row (design D7).
func snapshotProblem(err error, end bool) string {
	which, limit := "start's", turnStartLimit
	if end {
		which, limit = "end's", turnEndLimit
	}
	switch {
	case errors.Is(err, input.ErrTurnStoreFull):
		return fmt.Sprintf("the review's store reached its %s for this session", input.SizeText(input.TurnStoreLimit))
	case errors.Is(err, context.DeadlineExceeded):
		return fmt.Sprintf("its %s snapshot took more than %s", which, limit)
	case errors.Is(err, context.Canceled):
		return fmt.Sprintf("its %s snapshot was stopped", which)
	}
	return fmt.Sprintf("its %s snapshot failed: %s", which, err)
}

// reviewLeft is the store's files left out, in the state's words.
func reviewLeft(ls []input.TurnLeft) []state.ReviewLeft {
	if len(ls) == 0 {
		return nil
	}
	out := make([]state.ReviewLeft, len(ls))
	for i, l := range ls {
		out[i] = state.ReviewLeft{Path: l.Path, Why: l.Why, Size: l.Size, ModTime: l.ModTime}
	}
	return out
}

// turnFiles reads the files between a turn's two trees (design D4), in a
// command: one diff-tree.
func (m *Model) turnFiles(c state.Command) tea.Cmd {
	store, turn, from, to := m.turns, c.Turn, c.From, c.To
	if store == nil {
		return nil
	}
	return func() tea.Msg {
		fs, err := store.Files(from, to)
		if err != nil {
			return actionMsg{state.TurnFiles{Turn: turn, Problem: "its files could not be read: " + err.Error()}}
		}
		files := make([]state.ReviewFile, 0, min(len(fs.Files), state.ReviewFileCap))
		for _, f := range fs.Files[:min(len(fs.Files), state.ReviewFileCap)] {
			files = append(files, state.ReviewFile{Status: f.Status, Path: f.Path, Orig: f.Orig, Added: f.Added, Removed: f.Removed, Binary: f.Binary})
		}
		return actionMsg{state.TurnFiles{Turn: turn, Files: files, Total: fs.Total}}
	}
}

// openTurnDiff shows file i of turn t's review in the diff view, in zone B's
// place: its start's version against its end's, both from the store
// (design D9). It writes nothing and sends nothing to the CLI.
func (m *Model) openTurnDiff(t, i int) []tea.Cmd {
	if t < 1 || t > len(m.s.Stages) || m.turns == nil {
		return nil
	}
	r := m.s.Stages[t-1].Review
	if i < 0 || i >= len(r.Files) || r.From == "" || r.To == "" {
		return nil
	}
	f := r.Files[i]
	m.ui.Diff = render.DiffView{Open: true, Path: f.Path, Orig: f.Orig, Note: "comparing…", Commit: fmt.Sprintf("Turn %d", t), OldName: "start", NewName: "end", Turn: t}
	m.ui.Focus = "D"
	m.turnDiff = turnDiffOpen{turn: t, file: i}
	m.diffGen++
	gen, store, from, to, now := m.diffGen, m.turns, r.From, r.To, m.opt.Now
	file := input.CommitFile{Status: f.Status, Path: f.Path, Orig: f.Orig, Added: f.Added, Removed: f.Removed, Binary: f.Binary}
	return []tea.Cmd{func() tea.Msg {
		start := now()
		pair, err := store.Sides(from, to, file)
		// The sides named by the turn, not by their trees' hashes.
		pair.Old.Name, pair.New.Name = "start", "end"
		msg := diffMsg{gen: gen, path: file.Path, pair: pair, err: err}
		if err == nil && textual(pair.Old) && textual(pair.New) {
			msg.res = diff.Compare(pair.Old.Data, pair.New.Data, diffContext)
		}
		msg.took = now().Sub(start)
		return msg
	}}
}

// turnDiffOpen is the turn review's file the diff view shows: "r" reads it
// again.
type turnDiffOpen struct{ turn, file int }

// closeTurns stops the snapshots and removes the store, at the dashboard's
// end (design D3): a snapshot running is stopped, waited for a moment, and
// the store removed once no git of its runs.
func (m *Model) closeTurns() {
	m.turnsClosed = true
	if m.turnsStop != nil {
		m.turnsStop()
	}
	if m.turnJobs != nil {
		close(m.turnJobs)
		select {
		case <-m.turnsDone:
		case <-time.After(turnsCloseWait):
		}
		m.turnJobs = nil
	}
	if m.turns != nil {
		_ = m.turns.Close()
		m.turns = nil
	}
}
