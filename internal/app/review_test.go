package app

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/Evengelius/agent-command-center/internal/event"
	"github.com/Evengelius/agent-command-center/internal/input"
	"github.com/Evengelius/agent-command-center/internal/provider"
	"github.com/Evengelius/agent-command-center/internal/render"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// fakeTurns is a turn review's store (plan 65): its trees in order, each
// Take's error, its listing after an end, its files and sides; every call
// recorded. hold, when set, keeps Take waiting for it or its context.
type fakeTurns struct {
	mu       sync.Mutex
	trees    []string
	takes    int
	takeErr  map[int]error
	hold     chan struct{}
	changing []string
	files    input.CommitFiles
	sides    input.DiffPair
	calls    []string
	closed   bool
}

func (f *fakeTurns) Take(ctx context.Context) (input.TurnTree, error) {
	f.mu.Lock()
	f.takes++
	n, hold := f.takes, f.hold
	f.calls = append(f.calls, "take")
	f.mu.Unlock()
	if hold != nil {
		select {
		case <-hold:
		case <-ctx.Done():
			return input.TurnTree{}, ctx.Err()
		}
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := f.takeErr[n]; err != nil {
		return input.TurnTree{}, err
	}
	return input.TurnTree{Tree: f.trees[min(n, len(f.trees))-1]}, nil
}

func (f *fakeTurns) Changing(context.Context) ([]string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, "changing")
	return f.changing, nil
}

func (f *fakeTurns) Files(from, to string) (input.CommitFiles, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, "files "+from+".."+to)
	return f.files, nil
}

func (f *fakeTurns) Sides(from, to string, file input.CommitFile) (input.DiffPair, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, "sides "+from+".."+to+" "+file.Path)
	return f.sides, nil
}

func (f *fakeTurns) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.closed = true
	f.calls = append(f.calls, "close")
	return nil
}

func (f *fakeTurns) log() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.calls)
}

// twoFiles is a turn's files: one changed, one added.
var twoFiles = input.CommitFiles{Total: 2, Files: []input.CommitFile{
	{Status: 'M', Path: "internal/render/viewer.go", Added: 2, Removed: 1},
	{Status: 'A', Path: "notes.txt", Added: 1},
}}

// reviewRig is a dashboard on a scripted Claude Code session, ready, whose
// turn review's store is store, or openErr when it cannot be made.
func reviewRig(t *testing.T, store *fakeTurns, openErr error) (*harness, *scriptCLI) {
	t.Helper()
	cli := &scriptCLI{id: "claude-code", sess: newScriptSession()}
	h := newHarness(t, Options{Providers: []provider.Provider{cli}, Cwd: t.TempDir(), TurnStore: func(string) (TurnStore, error) {
		if openErr != nil {
			return nil, openErr
		}
		return store, nil
	}})
	h.settle()
	h.m.ui.Choice = "claude-code"
	h.send(key(tea.KeyEnter, 0))
	h.send(tea.WindowSizeMsg{Width: 120, Height: 40})
	pushEvents(h, "claude-code", evOf(event.SessionSpawned, event.Spawned{}), evOf(event.SessionReady, event.Ready{}))
	if s := h.m.State(); s.Session.Phase != state.Ready {
		t.Fatalf("session %q, want ready", s.Session.Phase)
	}
	return h, cli
}

// sendText types text and sends it.
func sendText(h *harness, text string) {
	for _, k := range typedText(text) {
		h.send(k)
	}
	h.send(key(tea.KeyEnter, 0))
}

// turnRound is a whole turn of the CLI's after a send: its start, a line
// of its answer, its end.
func turnRound(h *harness) {
	pushEvents(h, "claude-code",
		evOf(event.MessageMarker, event.Marker{Name: "message_start"}),
		evOf(event.MessageCompleted, event.Completed{Text: "Done."}),
		evOf(event.TurnCompleted, event.TurnEnd{Outcome: "completed", Subtype: "success"}))
}

func TestTurnReviewSend(t *testing.T) {
	// The owner's message goes once its turn's start is taken, never before
	// (design D4); then the turn's end, the listing once more, the files.
	synctest.Test(t, func(t *testing.T) {
		store := &fakeTurns{trees: []string{"t1", "t2"}, hold: make(chan struct{}), files: twoFiles}
		h, cli := reviewRig(t, store, nil)
		sendText(h, "do it")
		if sent, _, _ := cli.sess.calls(); len(sent) != 0 || !slices.Equal(store.log(), []string{"take"}) {
			t.Fatalf("sent %q while the snapshot ran; calls %v", sent, store.log())
		}
		if lineWith(h, "the turn's snapshot…") == "" {
			t.Fatalf("the editor's row does not say the snapshot runs:\n%s", h.frame())
		}
		close(store.hold)
		h.settle()
		if sent, _, _ := cli.sess.calls(); !slices.Equal(sent, []string{"do it"}) {
			t.Fatalf("sent %q after the snapshot", sent)
		}
		turnRound(h)
		if want := []string{"take", "take", "changing", "files t1..t2"}; !slices.Equal(store.log(), want) {
			t.Fatalf("calls %v, want %v", store.log(), want)
		}
		r := h.m.State().Stages[0].Review
		if !r.Shown() || r.Total != 2 || r.From != "t1" || r.To != "t2" || r.Changing {
			t.Fatalf("review %+v", r)
		}
		// The review is the dashboard's own: nothing more sent, no entry of
		// the conversation added (design D7).
		if sent, _, _ := cli.sess.calls(); len(sent) != 1 {
			t.Fatalf("sent %q", sent)
		}
		for _, e := range h.m.State().Conversation {
			if strings.Contains(e.Text, "files changed") {
				t.Fatalf("a conversation entry for the review: %+v", e)
			}
		}
		if l := lineWith(h, "files changed in turn 1"); !strings.Contains(l, "▸ 2 files changed in turn 1 · +3 −1 · Review") || !strings.Contains(l, "git") {
			t.Fatalf("the review's row %q", l)
		}
	})
}

func TestTurnReviewStartLimit(t *testing.T) {
	// A start's snapshot past 5 s: the message goes at 5 s, the turn says why
	// it has no review (designs D4 and D7).
	synctest.Test(t, func(t *testing.T) {
		store := &fakeTurns{trees: []string{"t1", "t2"}, hold: make(chan struct{})}
		h, cli := reviewRig(t, store, nil)
		start := time.Now()
		sendText(h, "do it")
		time.Sleep(turnStartLimit + time.Second)
		h.settle()
		if sent, _, _ := cli.sess.calls(); !slices.Equal(sent, []string{"do it"}) || time.Since(start) < turnStartLimit {
			t.Fatalf("sent %q after %s", sent, time.Since(start))
		}
		turnRound(h)
		r := h.m.State().Stages[0].Review
		if !r.Shown() || r.Problem != "its start's snapshot took more than 5s" {
			t.Fatalf("review %+v", r)
		}
		if l := lineWith(h, "No review of turn 1"); !strings.Contains(l, "✗ No review of turn 1: its start's snapshot took more than 5s") {
			t.Fatalf("the failure's row %q", l)
		}
		close(store.hold)
	})
}

func TestTurnReviewProblems(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		// The store full: the message sent all the same, the row says why.
		store := &fakeTurns{trees: []string{"t1"}, takeErr: map[int]error{1: input.ErrTurnStoreFull}}
		h, cli := reviewRig(t, store, nil)
		sendText(h, "do it")
		turnRound(h)
		if sent, _, _ := cli.sess.calls(); len(sent) != 1 {
			t.Fatalf("sent %q", sent)
		}
		if r := h.m.State().Stages[0].Review; r.Problem != "the review's store reached its 256.0 MiB for this session" {
			t.Fatalf("review %+v", r)
		}
		// Still changing after the end (design D5).
		store2 := &fakeTurns{trees: []string{"t1", "t2"}, changing: []string{"watch.log"}, files: twoFiles}
		h2, _ := reviewRig(t, store2, nil)
		sendText(h2, "do it")
		turnRound(h2)
		if l := lineWith(h2, "files changed in turn 1"); !strings.Contains(l, "· still changing") {
			t.Fatalf("the row %q", l)
		}
	})
}

func TestTurnReviewNoStore(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		// Outside a git repository: no snapshot, no row (design D6).
		h, cli := reviewRig(t, nil, input.NotRepoError{Dir: "x"})
		sendText(h, "do it")
		turnRound(h)
		if sent, _, _ := cli.sess.calls(); !slices.Equal(sent, []string{"do it"}) {
			t.Fatalf("sent %q", sent)
		}
		if h.m.State().Stages[0].Review.Shown() || lineWith(h, "turn 1") != "" && strings.Contains(lineWith(h, "turn 1"), "review") {
			t.Fatalf("a review outside a repository: %+v", h.m.State().Stages[0].Review)
		}
		// A store that could not be made: each turn says so.
		h2, _ := reviewRig(t, nil, errors.New("access is denied"))
		sendText(h2, "do it")
		turnRound(h2)
		if r := h2.m.State().Stages[0].Review; r.Problem != "the review's store could not be made: access is denied" {
			t.Fatalf("review %+v", r)
		}
	})
}

func TestTurnReviewCursor(t *testing.T) {
	// ↑ from the input on the review's summary, Enter unfolds, ↓ to a file,
	// Enter opens its diff "Turn 1: start → end", Esc back to the file;
	// ← folds the list, the cursor on the summary (design D8).
	synctest.Test(t, func(t *testing.T) {
		store := &fakeTurns{trees: []string{"t1", "t2"}, files: twoFiles,
			sides: input.DiffPair{Old: input.DiffSide{Name: "1a2b3c4", Path: "internal/render/viewer.go", Size: 4, Data: []byte("a\nb\n")}, New: input.DiffSide{Name: "5d6e7f8", Path: "internal/render/viewer.go", Size: 6, Data: []byte("a\nc\nd\n")}}}
		h, _ := reviewRig(t, store, nil)
		sendText(h, "do it")
		turnRound(h)
		h.send(key(tea.KeyUp, 0))
		if h.m.ui.Entry == 0 || h.m.ui.EntryReview != 1 {
			t.Fatalf("↑: entry %d, review %d", h.m.ui.Entry, h.m.ui.EntryReview)
		}
		if l := lineWith(h, "files changed in turn 1"); !strings.Contains(l, "Enter unfolds · ↑↓ entries · Esc back") {
			t.Fatalf("the summary under the cursor %q", l)
		}
		h.send(key(tea.KeyEnter, 0))
		if !h.m.ui.Unfolded[render.ReviewKey(1)] || lineWith(h, "notes.txt") == "" {
			t.Fatalf("Enter did not unfold: %v", h.m.ui.Unfolded)
		}
		h.send(key(tea.KeyDown, 0))
		if h.m.ui.EntryReview != 2 || !strings.Contains(lineWith(h, "viewer.go"), "Enter diff") {
			t.Fatalf("↓: review %d, row %q", h.m.ui.EntryReview, lineWith(h, "viewer.go"))
		}
		h.send(key(tea.KeyEnter, 0))
		if v := h.m.ui.Diff; !v.Open || v.Turn != 1 || v.Commit != "Turn 1" || h.m.ui.Focus != "D" {
			t.Fatalf("Enter on a file: diff %+v, focus %q", v, h.m.ui.Focus)
		}
		if got := store.log(); got[len(got)-1] != "sides t1..t2 internal/render/viewer.go" {
			t.Fatalf("calls %v", got)
		}
		if l := lineWith(h, "Turn 1: start → end"); l == "" {
			t.Fatalf("no path row with the turn:\n%s", h.frame())
		}
		h.send(key(tea.KeyEscape, 0))
		if h.m.ui.Diff.Open || h.m.ui.Focus != "B" || h.m.ui.EntryReview != 2 {
			t.Fatalf("Esc: diff %v, focus %q, review %d", h.m.ui.Diff.Open, h.m.ui.Focus, h.m.ui.EntryReview)
		}
		h.send(key(tea.KeyLeft, 0))
		if h.m.ui.Unfolded[render.ReviewKey(1)] || h.m.ui.EntryReview != 1 {
			t.Fatalf("←: unfolded %v, review %d", h.m.ui.Unfolded, h.m.ui.EntryReview)
		}
		h.send(key(tea.KeyEscape, 0))
		if h.m.ui.Entry != 0 || h.m.ui.EntryReview != 0 {
			t.Fatalf("Esc: entry %d, review %d", h.m.ui.Entry, h.m.ui.EntryReview)
		}
	})
}

func TestTurnReviewCursorReveals(t *testing.T) {
	// At 120x40, under a long answer, B held where the owner left it (PgUp,
	// then ↑ puts the cursor on the summary at the view's bottom edge): the
	// unfolded list's second file is below the view, as in Windows
	// Terminal's run (plan 65's evaluation), and ↓ onto it scrolls B the
	// least that shows it (design D8; the review of 2026-10-29).
	synctest.Test(t, func(t *testing.T) {
		store := &fakeTurns{trees: []string{"t1", "t2"}, files: twoFiles}
		h, _ := reviewRig(t, store, nil)
		sendText(h, "do it")
		var long strings.Builder
		for i := range 40 {
			fmt.Fprintf(&long, "line %d of a long answer\n\n", i+1)
		}
		pushEvents(h, "claude-code",
			evOf(event.MessageMarker, event.Marker{Name: "message_start"}),
			evOf(event.MessageCompleted, event.Completed{Text: long.String()}),
			evOf(event.TurnCompleted, event.TurnEnd{Outcome: "completed", Subtype: "success"}))
		// Each key drawn, as the program draws between messages: PgUp scrolls
		// by the frame drawn.
		h.m.View()
		h.draw(key(tea.KeyPgUp, 0))
		h.draw(key(tea.KeyUp, 0))
		h.draw(key(tea.KeyEnter, 0))
		if !h.m.ui.Unfolded[render.ReviewKey(1)] || lineWith(h, "files changed in turn 1") == "" {
			t.Fatalf("not unfolded on the summary: %v\n%s", h.m.ui.Unfolded, h.frame())
		}
		if lineWith(h, "notes.txt") != "" {
			t.Fatalf("the second file is in view at once: this test shows nothing\n%s", h.frame())
		}
		h.draw(key(tea.KeyDown, 0))
		h.draw(key(tea.KeyDown, 0))
		if l := lineWith(h, "notes.txt"); h.m.ui.EntryReview != 3 || !strings.Contains(l, "Enter diff") {
			t.Fatalf("↓ ↓: review %d, the second file's row %q\n%s", h.m.ui.EntryReview, l, h.frame())
		}
		if lineWith(h, "line 40 of a long answer") == "" {
			t.Errorf("B scrolled further than the second file needs:\n%s", h.frame())
		}
	})
}

func TestTurnReviewDiffReload(t *testing.T) {
	// "r" in a turn's diff reads its two sides again from the store, the
	// turn's label and the side-by-side layout kept (design D9).
	synctest.Test(t, func(t *testing.T) {
		store := &fakeTurns{trees: []string{"t1", "t2"}, files: twoFiles,
			sides: input.DiffPair{Old: input.DiffSide{Path: "internal/render/viewer.go", Size: 4, Data: []byte("a\nb\n")}, New: input.DiffSide{Path: "internal/render/viewer.go", Size: 6, Data: []byte("a\nc\nd\n")}}}
		h, _ := reviewRig(t, store, nil)
		sendText(h, "do it")
		turnRound(h)
		for _, k := range []tea.KeyPressMsg{key(tea.KeyUp, 0), key(tea.KeyEnter, 0), key(tea.KeyDown, 0), key(tea.KeyEnter, 0), typed('f')} {
			h.send(k)
		}
		if v := h.m.ui.Diff; !v.Open || v.Turn != 1 || !v.Wide || v.Added != 2 {
			t.Fatalf("the turn's diff, side by side: %+v", v)
		}
		store.mu.Lock()
		store.sides.New.Data, store.sides.New.Size = []byte("a\nc\nd\ne\n"), 8
		store.mu.Unlock()
		h.send(typed('r'))
		const read = "sides t1..t2 internal/render/viewer.go"
		if calls := store.log(); len(calls) < 2 || calls[len(calls)-1] != read || calls[len(calls)-2] != read {
			t.Fatalf("r: calls %v", calls)
		}
		if v := h.m.ui.Diff; !v.Open || v.Turn != 1 || v.Commit != "Turn 1" || !v.Wide || v.Added != 3 || v.Removed != 1 || h.m.ui.Focus != "D" {
			t.Fatalf("r: diff %+v, focus %q", v, h.m.ui.Focus)
		}
		if lineWith(h, "Turn 1: start → end") == "" {
			t.Fatalf("no path row with the turn after r:\n%s", h.frame())
		}
	})
}

func TestTurnReviewPanelEsc(t *testing.T) {
	// A turn's diff with a panel docked: Esc goes back to zone B, not to
	// the panel, as Source Control's own diff does (design D9).
	synctest.Test(t, func(t *testing.T) {
		store := &fakeTurns{trees: []string{"t1", "t2"}, files: twoFiles}
		h, _ := reviewRig(t, store, nil)
		h.send(tea.WindowSizeMsg{Width: 200, Height: 60}) // the panel beside zone B, not in its place
		sendText(h, "do it")
		turnRound(h)
		h.m.ui.Explorer.Open = true
		h.m.ui.Focus = "B"
		for _, k := range []tea.KeyPressMsg{key(tea.KeyUp, 0), key(tea.KeyEnter, 0), key(tea.KeyDown, 0), key(tea.KeyEnter, 0)} {
			h.send(k)
		}
		if !h.m.ui.Diff.Open || h.m.ui.Focus != "D" {
			t.Fatalf("diff %v, focus %q", h.m.ui.Diff.Open, h.m.ui.Focus)
		}
		h.send(key(tea.KeyEscape, 0))
		if h.m.ui.Focus != "B" || h.m.ui.EntryReview != 2 {
			t.Fatalf("Esc with a panel: focus %q, review %d", h.m.ui.Focus, h.m.ui.EntryReview)
		}
	})
}

func TestTurnReviewDiffNotes(t *testing.T) {
	// A binary or too large file is said as the diff view says it, for a
	// turn's diff as for Source Control's (plan 44, design D7).
	for _, c := range []struct {
		name string
		old  input.DiffSide
		want string
	}{
		{"binary", input.DiffSide{Size: 1200, Binary: true}, "Binary file: no textual diff"},
		{"too large", input.DiffSide{Size: 40 << 20, TooLarge: true}, "Too large to compare"},
	} {
		t.Run(c.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				store := &fakeTurns{trees: []string{"t1", "t2"}, files: twoFiles, sides: input.DiffPair{Old: c.old, New: input.DiffSide{Size: 10, Data: []byte("new text\n")}}}
				h, _ := reviewRig(t, store, nil)
				sendText(h, "do it")
				turnRound(h)
				for _, k := range []tea.KeyPressMsg{key(tea.KeyUp, 0), key(tea.KeyEnter, 0), key(tea.KeyDown, 0), key(tea.KeyEnter, 0)} {
					h.send(k)
				}
				if v := h.m.ui.Diff; !v.Open || v.Turn != 1 || !strings.HasPrefix(v.Note, c.want) {
					t.Fatalf("diff %+v, want the note %q", v, c.want)
				}
			})
		})
	}
}

func TestTurnReviewMouse(t *testing.T) {
	// A click on the summary unfolds the files; a double click on a file
	// opens its diff (design D8).
	synctest.Test(t, func(t *testing.T) {
		store := &fakeTurns{trees: []string{"t1", "t2"}, files: twoFiles}
		h, _ := reviewRig(t, store, nil)
		sendText(h, "do it")
		turnRound(h)
		at := func(part string) (int, int) {
			_ = h.m.View()
			c := h.m.screen.Conv
			for y, l := range strings.Split(h.frame(), "\n") {
				if strings.Contains(l, part) && y >= c.Y && y < c.Y+c.Room {
					return 60, y
				}
			}
			t.Fatalf("no row with %q", part)
			return 0, 0
		}
		press := func(x, y int) {
			_ = h.m.View()
			_, cmd := h.m.Update(click(x, y))
			h.exec(cmd)
			h.settle()
		}
		x, y := at("files changed in turn 1")
		press(x, y)
		if !h.m.ui.Unfolded[render.ReviewKey(1)] {
			t.Fatal("a click did not unfold the review")
		}
		x, y = at("notes.txt")
		press(x, y)
		press(x, y)
		if v := h.m.ui.Diff; !v.Open || v.Turn != 1 || v.Path != "notes.txt" {
			t.Fatalf("a double click on a file: diff %+v", v)
		}
	})
}

// panickyTurns is a store whose Take panics.
type panickyTurns struct{ fakeTurns }

func (p *panickyTurns) Take(context.Context) (input.TurnTree, error) { panic("store broke") }

func TestTurnReviewPanic(t *testing.T) {
	// A panic in a snapshot is its turn's problem: the message goes, the
	// session stays ready, the row says why (plan 48, design D5).
	synctest.Test(t, func(t *testing.T) {
		cli := &scriptCLI{id: "claude-code", sess: newScriptSession()}
		store := &panickyTurns{}
		h := newHarness(t, Options{Providers: []provider.Provider{cli}, Cwd: t.TempDir(), TurnStore: func(string) (TurnStore, error) { return store, nil }})
		h.settle()
		h.m.ui.Choice = "claude-code"
		h.send(key(tea.KeyEnter, 0))
		h.send(tea.WindowSizeMsg{Width: 120, Height: 40})
		pushEvents(h, "claude-code", evOf(event.SessionSpawned, event.Spawned{}), evOf(event.SessionReady, event.Ready{}))
		start := time.Now()
		sendText(h, "do it")
		if sent, _, _ := cli.sess.calls(); !slices.Equal(sent, []string{"do it"}) || time.Since(start) >= turnStartLimit {
			t.Fatalf("sent %q after %s", sent, time.Since(start))
		}
		turnRound(h)
		s := h.m.State()
		if s.Session.Phase != state.Ready || !strings.HasPrefix(s.Stages[0].Review.Problem, "the turn review failed: panic: store broke") {
			t.Fatalf("session %q, review %+v", s.Session.Phase, s.Stages[0].Review)
		}
	})
}

func TestTurnReviewQueueFull(t *testing.T) {
	// Past 16 snapshots waiting, one more is refused: its turn says so, and
	// a start's gate opens at once, so that the message is not held.
	synctest.Test(t, func(t *testing.T) {
		store := &fakeTurns{trees: []string{"t1"}, hold: make(chan struct{})}
		h, _ := reviewRig(t, store, nil)
		for range turnQueue + 1 { // the first runs and waits; 16 fill the queue
			h.m.turnSnapshot(state.Command{Kind: state.CmdTurnSnapshot, Turn: 1, End: true})
		}
		cmd := h.m.turnSnapshot(state.Command{Kind: state.CmdTurnSnapshot, Turn: 2})
		select {
		case <-h.m.turnGate.done:
		default:
			t.Fatal("a refused start's gate is still closed")
		}
		if msg, ok := cmd().(actionMsg); !ok || msg.a.(state.TurnSnapshot).Problem != "too many snapshots waiting" {
			t.Fatalf("the refused snapshot's answer: %#v", cmd())
		}
		close(store.hold)
	})
}

// TestTurnReviewFrameCost is criterion 65.9's frame: a turn of 1,000 files,
// unfolded, the cursor walking its rows, each key to its frame within the
// keystroke's target (judged in the timing step; doc/evaluation.md).
func TestTurnReviewFrameCost(t *testing.T) {
	files := input.CommitFiles{Total: 1000}
	for i := range 1000 {
		files.Files = append(files.Files, input.CommitFile{Status: 'M', Path: fmt.Sprintf("internal/pkg%02d/file%04d.go", i%40, i), Added: i % 7, Removed: i % 3})
	}
	store := &fakeTurns{trees: []string{"t1", "t2"}, files: files}
	h, _ := reviewRig(t, store, nil)
	sendText(h, "do it")
	turnRound(h)
	h.send(key(tea.KeyUp, 0))
	h.send(key(tea.KeyEnter, 0))
	if r := h.m.State().Stages[0].Review; len(r.Files) != 1000 || !h.m.ui.Unfolded[render.ReviewKey(1)] {
		t.Fatalf("%d files, unfolded %v", len(r.Files), h.m.ui.Unfolded)
	}
	var ds []time.Duration
	for i := range 400 {
		k := key(tea.KeyDown, 0)
		if i%100 >= 50 {
			k = key(tea.KeyUp, 0)
		}
		ds = append(ds, frameTime(h, k))
	}
	if h.m.ui.EntryReview < 1 {
		t.Fatalf("the cursor left the review: %d", h.m.ui.EntryReview)
	}
	p50, p95 := percentile(ds, 0.50), percentile(ds, 0.95)
	t.Logf("a key to its frame, a turn of 1,000 files unfolded: p50 %v, p95 %v (target p95 ≤ %v)", p50, p95, keystrokeTarget)
	if p95 > keystrokeTarget {
		overTarget(t, "p95 %v over the %v target", p95, keystrokeTarget)
	}
}

func TestTurnReviewClose(t *testing.T) {
	// The dashboard's end closes the store, its snapshots stopped (design D3).
	synctest.Test(t, func(t *testing.T) {
		store := &fakeTurns{trees: []string{"t1"}}
		h, _ := reviewRig(t, store, nil)
		h.m.Shutdown()
		if !store.closed || h.m.turns != nil {
			t.Fatalf("closed %v, turns %v", store.closed, h.m.turns)
		}
	})
}
