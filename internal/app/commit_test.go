package app

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/Evengelius/agent-command-center/internal/input"
	"github.com/Evengelius/agent-command-center/internal/render"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// Plan 45 (brief amendment 46): Commit, Push and Commit & Push in Source
// Control, through the application; git's writes from the test double
// below, and in TestCommitGit from git on a temporary repository with a
// local bare remote.

// fakeWrite is the test double's writes over plan 44's status: each call
// logged in order, the errors to give, and the status as git would leave
// it after a success (the staged entries committed, ahead counted, the
// push syncing).
type fakeWrite struct {
	*fakeGit
	hold                                    chan struct{}
	stageErr, commitErr, targetErr, pushErr error
	log                                     []string
	expects                                 [][]input.GitEntry
	made                                    int
	// stoppable makes a push the owner can stop, waiting until stopped or
	// pushHold closes; stopBusy refuses the stop, as a lock file would (plan
	// 45, design D15).
	stoppable, stopBusy bool
	pushHold            chan struct{}
}

// calledStatus is the status reads made so far.
func (f *fakeWrite) calledStatus() []string { return f.called("status ") }

// holdOn makes the writes wait until the channel closes.
func (f *fakeWrite) holdOn() chan struct{} {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.hold = make(chan struct{})
	return f.hold
}

func (f *fakeWrite) waitHold() {
	f.mu.Lock()
	h := f.hold
	f.mu.Unlock()
	if h != nil {
		<-h
	}
}

// logged is the writes logged so far whose words start with prefix.
func (f *fakeWrite) logged(prefix string) []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []string
	for _, l := range f.log {
		if strings.HasPrefix(l, prefix) {
			out = append(out, l)
		}
	}
	return out
}

func (f *fakeWrite) Stage(dir string, paths []string) error {
	f.waitHold()
	f.mu.Lock()
	defer f.mu.Unlock()
	f.log = append(f.log, "stage "+strings.Join(paths, ","))
	if f.stageErr != nil {
		return f.stageErr
	}
	for i, e := range f.snap.Entries {
		if slices.Contains(paths, e.Path) {
			if e.X == '?' {
				e.X, e.Y = 'A', ' '
			} else {
				e.X, e.Y = e.Y, ' '
			}
			f.snap.Entries[i] = e
		}
	}
	return nil
}

func (f *fakeWrite) Commit(dir, message string, expect []input.GitEntry) (input.CommitResult, error) {
	f.waitHold()
	f.mu.Lock()
	defer f.mu.Unlock()
	f.log = append(f.log, "commit "+message)
	f.expects = append(f.expects, slices.Clone(expect))
	if f.commitErr != nil {
		return input.CommitResult{}, f.commitErr
	}
	f.made++
	var left []input.GitEntry
	for _, e := range f.snap.Entries {
		if e.X != ' ' && e.X != '?' {
			if e.X = ' '; e.Y == ' ' {
				continue
			}
		}
		left = append(left, e)
	}
	f.snap.Entries = left
	if f.snap.Branch.Upstream != "" {
		f.snap.Branch.Ahead++
	}
	return input.CommitResult{Hash: fmt.Sprintf("%040d", f.made), Short: fmt.Sprintf("abc%04d", f.made)}, nil
}

func (f *fakeWrite) PushTarget(dir string) (input.PushTarget, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.log = append(f.log, "target")
	if f.targetErr != nil {
		return input.PushTarget{}, f.targetErr
	}
	b := f.snap.Branch
	return input.PushTarget{Branch: b.Name, Remote: "origin", RemoteRef: "refs/heads/" + b.Name, Upstream: b.Upstream, Ahead: b.Ahead}, nil
}

// The double is the application's writes.
var _ input.SourceWrite = (*fakeWrite)(nil)

func (f *fakeWrite) Push(dir string, stop *input.GitStop) (input.PushResult, error) {
	f.mu.Lock()
	stoppable, pushHold := f.stoppable, f.pushHold
	f.mu.Unlock()
	if stoppable {
		// A push the owner can stop (design D15): refused while stopBusy,
		// as a lock file would.
		stop.Set(func() (bool, string) {
			f.mu.Lock()
			defer f.mu.Unlock()
			if f.stopBusy {
				return false, "git is updating the repository: try again in a moment."
			}
			return true, ""
		})
		defer stop.Set(nil)
		select {
		case <-stop.Asked():
			f.mu.Lock()
			f.log = append(f.log, "push cancelled")
			f.mu.Unlock()
			return input.PushResult{}, input.Refusal{Kind: "cancelled", Msg: "The push was cancelled. The remote may have received it: Push again to check."}
		case <-pushHold:
		}
	} else {
		f.waitHold()
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.log = append(f.log, "push")
	if f.pushErr != nil {
		return input.PushResult{}, f.pushErr
	}
	b := f.snap.Branch
	f.snap.Branch.Ahead = 0
	return input.PushResult{Target: input.PushTarget{Branch: b.Name, Remote: "origin", RemoteRef: "refs/heads/" + b.Name, Upstream: b.Upstream, Ahead: b.Ahead}}, nil
}

// commitRig is sourceRig's repository (main.go and gone.txt changed,
// src/Main.java staged and changed again, docs/readme.md staged, web/
// untracked) on main with the upstream origin/main, its writes faked,
// Source Control shown at 200x60.
func commitRig(t *testing.T) (*harness, *fakeWrite) { return commitRigOn(t, "origin/main") }

// commitRigOn is commitRig with the upstream given, "" for none: a commit
// then leaves the button on Commit.
func commitRigOn(t *testing.T, upstream string) (*harness, *fakeWrite) {
	t.Helper()
	r, g, _ := sourceRig(t)
	g.snap.Branch.Upstream = upstream
	w := &fakeWrite{fakeGit: g}
	r.h.m.opt.Git = w
	showSource(t, r.h)
	return r.h, w
}

// refresh reads git's status again, as the poll does.
func refresh(h *harness) {
	h.exec(h.m.readSnapshot())
	h.settle()
}

// cellsFrom counts the cells from (x, y) rightwards carrying one of items.
func cellsFrom(h *harness, x, y int, items ...int) int {
	n := 0
	for ; ; x++ {
		p, it := h.m.screen.ItemAt(x, y)
		if p != render.SourcePanel || !slices.Contains(items, it) {
			return n
		}
		n++
	}
}

// clickItem clicks the first cell of the panel's item, plus dx and dy.
func clickItem(t *testing.T, h *harness, item, dx, dy int) {
	t.Helper()
	x, y := itemCell(t, h, render.SourcePanel, item)
	mouseClick(t, h, x+dx, y+dy)
}

// clickQueued clicks the panel's item as the program would, its commands
// run in their own goroutines (mouseAt runs them in the test's): a write
// held by the double then holds no test.
func clickQueued(t *testing.T, h *harness, item, dx, dy int) {
	t.Helper()
	x, y := itemCell(t, h, render.SourcePanel, item)
	h.send(click(x+dx, y+dy))
	h.send(tea.MouseReleaseMsg{X: x + dx, Y: y + dy, Button: tea.MouseLeft})
}

// typeMessage clicks the message field and types text, "\n" as Enter.
func typeMessage(t *testing.T, h *harness, text string) {
	t.Helper()
	clickItem(t, h, render.ItemMsg, 0, 0)
	if h.m.ui.Source.Place != render.PlaceMsg || h.m.ui.Focus != "R" {
		t.Fatalf("a click on the field: place %q, focus %q", h.m.ui.Source.Place, h.m.ui.Focus)
	}
	for _, r := range text {
		if r == '\n' {
			h.send(keyOf(tea.KeyEnter))
		} else {
			h.send(typed(r))
		}
	}
}

// TestCommitLayout is criterion 45.1 through the application (design D1):
// under the repository row the message box ("Message"), then the button
// ("✓ Commit", the separator, "⌄"), both spanning the panel's inner
// columns; no dropdown until the chevron opens it, then right under the
// button, its columns the button's; no controls without a repository.
func TestCommitLayout(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, w := commitRig(t)
		f := h.frame()
		for _, s := range []string{h.m.ui.Source.Repo + " · main", "Message", "✓ Commit", "⌄"} {
			if !strings.Contains(f, s) {
				t.Errorf("%q not on screen:\n%s", s, f)
			}
		}
		if strings.Contains(f, render.MenuLabel) {
			t.Error("the dropdown shows by default")
		}
		x0, _ := panelExtent(h, render.SourcePanel)
		mx, my := itemCell(t, h, render.SourcePanel, render.ItemMsg)
		bx, by := itemCell(t, h, render.SourcePanel, render.ItemCommitMain)
		_, ry := at(t, h, h.m.ui.Source.Repo+" · main", 0)
		if mx != x0+2 || bx != mx || my != ry+1 || by != my+3 {
			t.Errorf("message box at %d,%d, button at %d,%d, panel from %d, repository row %d", mx, my, bx, by, x0, ry)
		}
		bw := cellsFrom(h, bx, by, render.ItemCommitMain, render.ItemCommitMore)
		if inner := h.m.screen.Panel.W - 4; bw != inner {
			t.Errorf("the button %d wide, the panel's inner %d", bw, inner)
		}
		clickItem(t, h, render.ItemCommitMore, 0, 0)
		if !h.m.ui.Source.Menu {
			t.Fatal("the chevron opened nothing")
		}
		dx, dy := itemCell(t, h, render.SourcePanel, render.ItemCommitMenu)
		if dx != bx || dy != by+render.BtnRows {
			t.Errorf("the dropdown at %d,%d, the button at %d,%d", dx, dy, bx, by)
		}
		rows := strings.Split(h.frame(), "\n")
		if n := strings.Count(strings.Join(rows[:len(rows)-1], "\n"), render.MenuLabel); n != 1 {
			t.Errorf("%q %d times above the footer, want exactly one item", render.MenuLabel, n)
		}
		for r := range render.MenuRows {
			if got := cellsFrom(h, dx, dy+r, render.ItemCommitMenu); got != bw {
				t.Errorf("dropdown row %d: %d cells, the button %d", r, got, bw)
			}
		}
		if len(w.log) != 0 {
			t.Errorf("git written: %q", w.log)
		}
		h.send(keyOf(tea.KeyEscape))
		w.set(func(f *fakeGit) { f.statusErr = input.NotRepoError{Dir: "x"} })
		refresh(h)
		if f := h.frame(); strings.Contains(f, "✓ Commit") || h.m.ui.Source.Repo != "" {
			t.Errorf("controls without a repository:\n%s", f)
		}
	})
}

// TestCommitMessageField is criterion 45.3 (design D2): the editor's keys,
// Enter a new line, a bracketed paste kept byte for byte and never
// committing, the text kept across the Explorer, the panel hidden and the
// focus away; no git write.
func TestCommitMessageField(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, w := commitRig(t)
		typeMessage(t, h, "Fix\nbody")
		if got := h.m.msg.Text(); got != "Fix\nbody" {
			t.Fatalf("typed %q", got)
		}
		for range 4 {
			h.send(key(tea.KeyLeft, tea.ModShift))
		}
		h.send(typed('X'))
		if got := h.m.msg.Text(); got != "Fix\nX" {
			t.Errorf("a selection typed over: %q", got)
		}
		h.send(key('z', tea.ModCtrl))
		if got := h.m.msg.Text(); got != "Fix\nbody" {
			t.Errorf("Ctrl+Z: %q", got)
		}
		h.send(key('a', tea.ModCtrl))
		h.send(key('x', tea.ModCtrl))
		if got := h.m.msg.Text(); got != "" {
			t.Errorf("Ctrl+A, Ctrl+X: %q", got)
		}
		h.send(tea.PasteStartMsg{})
		h.send(tea.PasteMsg{Content: "one\ntwo\tthree\nfour  "})
		h.send(tea.PasteEndMsg{})
		if got := h.m.msg.Text(); got != "one\ntwo\tthree\nfour  " {
			t.Errorf("pasted %q", got)
		}
		if h.m.edit.Text() != "" {
			t.Errorf("zone B's draft took %q", h.m.edit.Text())
		}
		if h.m.draftBeside() || h.m.editorTakesPaste() {
			t.Error("the field has the keys, yet zone B's draft beside it takes pastes and images")
		}
		for _, r := range "r?zF" {
			h.send(typed(r)) // the panel's and the dashboard's letters type here
		}
		want := "one\ntwo\tthree\nfour  r?zF"
		if got := h.m.msg.Text(); got != want || h.m.ui.Zoom || h.m.ui.Keys {
			t.Errorf("letters: %q, zoom %v, overlay %v", got, h.m.ui.Zoom, h.m.ui.Keys)
		}
		x, y := itemCell(t, h, render.SidePanel, render.SideExplorer)
		mouseClick(t, h, x, y)
		x, y = itemCell(t, h, render.SidePanel, render.SideSource)
		mouseClick(t, h, x, y)
		h.send(keyOf(tea.KeyEscape))
		h.send(keyOf(tea.KeyEscape))
		if got := h.m.msg.Text(); got != want || h.m.ui.Focus != "B" {
			t.Errorf("after the Explorer and Esc: %q, focus %q", got, h.m.ui.Focus)
		}
		if f := h.frame(); !strings.Contains(f, "one") {
			t.Errorf("the message not shown:\n%s", f)
		}
		if len(w.log) != 0 {
			t.Errorf("git written while typing: %q", w.log)
		}
	})
}

// TestCommitValidation is criterion 45.4 (designs D6 and D8): an empty or
// blank message says so with no git call, the text kept; no change at all
// says "No changes to commit." with none, the button dimmed.
func TestCommitValidation(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, w := commitRig(t)
		clickItem(t, h, render.ItemCommitMain, 0, 0)
		if v := h.m.ui.Source; v.Notice != "Enter a commit message." || v.NoticeKind != render.NoticeCheck {
			t.Errorf("empty: %q", v.Notice)
		}
		typeMessage(t, h, "  \n  ")
		h.send(ctrlEnter)
		if got := h.m.msg.Text(); got != "  \n  " || h.m.ui.Source.Notice != "Enter a commit message." {
			t.Errorf("blank: kept %q, said %q", got, h.m.ui.Source.Notice)
		}
		if f := h.frame(); !strings.Contains(f, "Enter a commit message.") || !strings.Contains(f, "Message") && h.m.msg.Text() == "" {
			t.Errorf("the note or the field not shown:\n%s", f)
		}
		h.send(typed('x'))
		if h.m.ui.Source.Notice != "" {
			t.Error("editing the message left the note")
		}
		w.set(func(f *fakeGit) { f.snap.Entries = nil })
		h.send(keyOf(tea.KeyEscape))
		x, y := itemCell(t, h, render.SidePanel, render.SideSource)
		mouseClick(t, h, x, y) // hidden
		mouseClick(t, h, x, y) // shown again: a read
		if !h.m.ui.Source.Dim {
			t.Error("no change: the button not dimmed")
		}
		clickItem(t, h, render.ItemCommitMain, 0, 0)
		if got := h.m.ui.Source.Notice; got != "No changes to commit." {
			t.Errorf("no change: %q", got)
		}
		if len(w.log) != 0 {
			t.Errorf("git written: %q", w.log)
		}
	})
}

// TestCommitStaging is criterion 45.5 (design D6): staged changes commit
// the index alone, expected as shown; nothing staged asks the question,
// Esc and Cancel doing nothing, "Stage and commit" staging exactly the
// CHANGES paths, then committing them; a conflict refused with no call.
func TestCommitStaging(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, w := commitRigOn(t, "")
		// The staged entries' blobs as the snapshot read them (design D14).
		w.set(func(f *fakeGit) { f.snap.Entries[1].Index, f.snap.Entries[2].Index = "b1ob-main", "b1ob-readme" })
		refresh(h)
		typeMessage(t, h, "Staged only")
		h.send(ctrlEnter)
		if got := w.logged(""); !slices.Equal(got, []string{"commit Staged only"}) {
			t.Fatalf("writes %q", got)
		}
		var paths []string
		for _, e := range w.expects[0] {
			paths = append(paths, string(e.X)+string(e.Y)+" "+e.Path+" "+e.Index)
		}
		if !slices.Equal(paths, []string{"MM src/Main.java b1ob-main", "A  docs/readme.md b1ob-readme"}) {
			t.Errorf("expected %q", paths)
		}
		// Nothing staged now: main.go, src/Main.java and gone.txt changed, web/ untracked.
		typeMessage(t, h, "All of it")
		h.send(ctrlEnter)
		v := h.m.ui.Source
		want := "Nothing is staged. Stage the 4 changes listed under CHANGES and commit them?"
		if f := h.frame(); v.Ask != 4 || !strings.Contains(f, "Nothing is staged. Stage") || !strings.Contains(f, "the 4 changes listed") ||
			!strings.Contains(f, " Stage and commit ") || !strings.Contains(f, " Cancel ") {
			t.Fatalf("the question: %d, %q\n%s", v.Ask, render.AskText(v.Ask, v.AskPush), h.frame())
		}
		if render.AskText(4, false) != want {
			t.Errorf("the question's words: %q", render.AskText(4, false))
		}
		h.send(keyOf(tea.KeyEscape))
		h.send(ctrlEnter)
		clickItem(t, h, render.ItemAskNo, 0, 0)
		if h.m.ui.Source.Ask != 0 || len(w.logged("stage")) != 0 || len(w.logged("commit")) != 1 {
			t.Fatalf("Esc and Cancel: ask %d, writes %q", h.m.ui.Source.Ask, w.log)
		}
		h.send(ctrlEnter)
		clickItem(t, h, render.ItemAskYes, 0, 0)
		if got := w.logged(""); !slices.Equal(got, []string{"commit Staged only", "stage main.go,src/Main.java,gone.txt,web/", "commit All of it"}) {
			t.Errorf("stage and commit: %q", got)
		}
		if n := len(w.expects); n != 2 || len(w.expects[1]) != 4 {
			t.Errorf("the commit after staging expected %d entries", len(w.expects[n-1]))
		}
		w.set(func(f *fakeGit) {
			f.snap.Entries = []input.GitEntry{{X: 'U', Y: 'U', Path: "both.go", RepoPath: "both.go"}, {X: 'M', Y: ' ', Path: "a.go", RepoPath: "a.go"}}
		})
		refresh(h)
		typeMessage(t, h, "conflict")
		h.send(ctrlEnter)
		if got := h.m.ui.Source.Notice; got != "Resolve the merge conflicts first." || len(w.logged("commit")) != 2 {
			t.Errorf("a conflict: %q, writes %q", got, w.log)
		}
	})
}

// TestCommitExplorerOverAsk is criterion 47.6 (plan 47, design D4): the
// Explorer opened over the staging question drops it with Source Control's
// panel, so the next click acts: Source Control's tile shows the panel
// again, with no question, and nothing is staged.
func TestCommitExplorerOverAsk(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, w := commitRigOn(t, "")
		typeMessage(t, h, "Staged only")
		h.send(ctrlEnter) // the staged entries committed: nothing staged now
		typeMessage(t, h, "All of it")
		h.send(ctrlEnter)
		if h.m.ui.Source.Ask == 0 {
			t.Fatalf("no staging question:\n%s", h.frame())
		}
		h.send(keyOf(tea.KeyF3))
		if !h.m.ui.Explorer.Open || h.m.ui.Source.Open {
			t.Fatalf("F3: Explorer %v, Source Control %v", h.m.ui.Explorer.Open, h.m.ui.Source.Open)
		}
		x, y := itemCell(t, h, render.SidePanel, render.SideSource)
		mouseClick(t, h, x, y)
		if v := h.m.ui.Source; !v.Open || v.Ask != 0 || strings.Contains(h.frame(), "Nothing is staged") {
			t.Fatalf("a click on Source Control's tile: open %v, ask %d\n%s", v.Open, v.Ask, h.frame())
		}
		// The compact rule hides the panel through closeSource, the same
		// helper's other caller: zone B given the focus under 100 columns.
		typeMessage(t, h, " again")
		h.send(ctrlEnter)
		h.send(tea.WindowSizeMsg{Width: 90, Height: 40})
		if h.m.ui.Source.Ask == 0 || !h.m.ui.Source.Open {
			t.Fatalf("compact: ask %d, open %v", h.m.ui.Source.Ask, h.m.ui.Source.Open)
		}
		h.send(key(tea.KeyF6, tea.ModShift))
		if v := h.m.ui.Source; v.Open || v.Ask != 0 || h.m.ask != nil || h.m.ui.Focus != "B" {
			t.Fatalf("zone B in the compact layout: open %v, ask %d, focus %q", v.Open, v.Ask, h.m.ui.Focus)
		}
		if got := w.logged("stage"); len(got) != 0 {
			t.Errorf("staged: %q", got)
		}
	})
}

// TestCommitMainAction is criterion 45.9 (design D7): the button's action
// from the snapshot alone, a new model (a restart) included.
func TestCommitMainAction(t *testing.T) {
	for _, c := range []struct {
		name    string
		branch  input.GitBranch
		entries bool
		action  string
		dim     bool
	}{
		{"changes, no upstream", input.GitBranch{Name: "main"}, true, render.ActionCommit, false},
		{"nothing", input.GitBranch{Name: "main", Upstream: "origin/main"}, false, render.ActionCommit, true},
		{"ahead", input.GitBranch{Name: "main", Upstream: "origin/main", Ahead: 1}, false, render.ActionPush, false},
		{"ahead with changes", input.GitBranch{Name: "main", Upstream: "origin/main", Ahead: 1}, true, render.ActionPush, false},
		{"gone", input.GitBranch{Name: "main", Upstream: "origin/main", Ahead: 2, Gone: true}, true, render.ActionCommit, false},
		{"no remote", input.GitBranch{Name: "main", Ahead: 0}, false, render.ActionCommit, true},
	} {
		t.Run(c.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				r, g, _ := sourceRig(t) // a new model: the state is Git's from its first read
				g.snap.Branch = c.branch
				if !c.entries {
					g.snap.Entries = nil
				}
				r.h.m.opt.Git = &fakeWrite{fakeGit: g}
				showSource(t, r.h)
				if v := r.h.m.ui.Source; v.Action != c.action || v.Dim != c.dim {
					t.Errorf("action %q dim %v, want %q %v", v.Action, v.Dim, c.action, c.dim)
				}
				label := "✓ Commit"
				if c.action == render.ActionPush {
					label = "Push"
				}
				if f := r.h.frame(); !strings.Contains(f, label) {
					t.Errorf("%q not on the button:\n%s", label, f)
				}
			})
		})
	}
}

// TestCommitWorkflowA is criterion 45.10 with the double (designs D7, D8,
// D9 and D11): Commit makes one commit, clears the field, and the button
// says Push once the status read says ahead 1; Push pushes, the button says
// Commit; with new changes, still Commit.
func TestCommitWorkflowA(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, w := commitRig(t)
		typeMessage(t, h, "Add Source Control diff viewer")
		clickItem(t, h, render.ItemCommitMain, 0, 0)
		if got := w.logged(""); !slices.Equal(got, []string{"commit Add Source Control diff viewer"}) {
			t.Fatalf("writes %q", got)
		}
		v := h.m.ui.Source
		if h.m.msg.Text() != "" || v.Action != render.ActionPush || v.Notice != "Committed abc0001." || v.NoticeKind != render.NoticeDone {
			t.Fatalf("after the commit: field %q, action %q, note %q", h.m.msg.Text(), v.Action, v.Notice)
		}
		if f := h.frame(); !strings.Contains(f, "Push") || strings.Contains(f, "✓ Commit") {
			t.Errorf("the button not on Push:\n%s", f)
		}
		clickItem(t, h, render.ItemCommitMain, 0, 0)
		if got := w.logged(""); !slices.Equal(got, []string{"commit Add Source Control diff viewer", "push"}) {
			t.Fatalf("writes %q", got)
		}
		if v := h.m.ui.Source; v.Action != render.ActionCommit || v.Notice != "Pushed to origin/main." {
			t.Errorf("after the push: action %q, note %q", v.Action, v.Notice)
		}
		if f := h.frame(); !strings.Contains(f, "✓ Commit") {
			t.Errorf("the button not back on Commit:\n%s", f)
		}
		if h.m.ui.Source.Dim {
			t.Error("changes left, yet the button is dimmed")
		}
	})
}

// TestCommitWorkflowB is criterion 45.11 with the double (design D10):
// Commit & Push checks the destination, commits, then pushes, in that
// order, says so, closes the dropdown and clears the field; a failing
// commit runs no push and keeps the message; a failing push keeps the
// commit, says so, clears the field and shows Push, which then pushes once
// more and never commits again.
func TestCommitWorkflowB(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, w := commitRig(t)
		typeMessage(t, h, "Both")
		clickItem(t, h, render.ItemCommitMore, 0, 0)
		clickItem(t, h, render.ItemCommitMenu, 3, 1)
		if got := w.logged(""); !slices.Equal(got, []string{"target", "commit Both", "push"}) {
			t.Fatalf("writes %q", got)
		}
		v := h.m.ui.Source
		if v.Menu || h.m.msg.Text() != "" || v.Notice != "Committed abc0001 and pushed to origin/main." || v.Action != render.ActionCommit {
			t.Fatalf("after both: menu %v, field %q, note %q, action %q", v.Menu, h.m.msg.Text(), v.Notice, v.Action)
		}

		// The commit failing: no push, the message kept, git's words.
		w.set(func(f *fakeGit) {
			f.snap.Entries = append(f.snap.Entries, input.GitEntry{X: 'M', Y: ' ', Path: "b.go", RepoPath: "b.go"})
		})
		w.commitErr = errors.New("hook says no")
		refresh(h)
		typeMessage(t, h, "Fails")
		clickItem(t, h, render.ItemCommitMore, 0, 0)
		clickItem(t, h, render.ItemCommitMenu, 3, 1)
		if got := w.logged(""); !slices.Equal(got, []string{"target", "commit Both", "push", "target", "commit Fails"}) {
			t.Fatalf("writes %q", got)
		}
		if v := h.m.ui.Source; h.m.msg.Text() != "Fails" || v.Notice != "hook says no" || v.NoticeKind != render.NoticeFailed {
			t.Fatalf("a failed commit: field %q, note %q", h.m.msg.Text(), v.Notice)
		}

		// The push failing: the commit kept, said, Push for the push alone.
		w.commitErr = nil
		w.pushErr = input.PushRejected{Ref: "refs/heads/main:refs/heads/main", Reason: "pre-receive hook declined", Remote: true}
		clickItem(t, h, render.ItemCommitMore, 0, 0)
		clickItem(t, h, render.ItemCommitMenu, 3, 1)
		v = h.m.ui.Source
		if want := "Committed abc0002, but the push failed: The remote rejected the push: pre-receive hook declined."; v.Notice != want {
			t.Errorf("note %q\nwant %q", v.Notice, want)
		}
		if h.m.msg.Text() != "" || v.Action != render.ActionPush {
			t.Fatalf("after the failed push: field %q, action %q", h.m.msg.Text(), v.Action)
		}
		w.pushErr = nil
		clickItem(t, h, render.ItemCommitMain, 0, 0)
		if got := w.logged("commit"); len(got) != 3 {
			t.Errorf("commits %q", got)
		}
		if got := w.logged("push"); len(got) != 3 || w.made != 2 {
			t.Errorf("pushes %q, commits made %d", got, w.made)
		}
		if v := h.m.ui.Source; v.Action != render.ActionCommit || v.Notice != "Pushed to origin/main." {
			t.Errorf("after the retry: %q, %q", v.Action, v.Notice)
		}

		// A destination refused before any commit.
		w.targetErr = input.Refusal{Kind: "no-upstream", Msg: "The branch main has no upstream. Set one in a terminal: git push -u origin main"}
		w.set(func(f *fakeGit) {
			f.snap.Entries = append(f.snap.Entries, input.GitEntry{X: 'M', Y: ' ', Path: "c.go", RepoPath: "c.go"})
		})
		refresh(h)
		typeMessage(t, h, "No upstream")
		clickItem(t, h, render.ItemCommitMore, 0, 0)
		clickItem(t, h, render.ItemCommitMenu, 3, 1)
		if v := h.m.ui.Source; v.Notice != w.targetErr.Error() || v.NoticeKind != render.NoticeCheck || h.m.msg.Text() != "No upstream" {
			t.Errorf("refused: note %q, field %q", v.Notice, h.m.msg.Text())
		}
		if got := w.logged("commit"); len(got) != 3 {
			t.Errorf("a refused destination committed: %q", got)
		}
	})
}

// TestCommitOneAtATime is criterion 45.12 (design D11): repeated clicks and
// keys during an operation make one; the dropdown does not open and Commit
// & Push does nothing meanwhile; a failure is not retried.
func TestCommitOneAtATime(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, w := commitRigOn(t, "")
		typeMessage(t, h, "Once")
		hold := w.holdOn()
		for range 10 {
			clickQueued(t, h, render.ItemCommitMain, 0, 0)
		}
		for range 3 {
			h.send(ctrlEnter)
		}
		if v := h.m.ui.Source; v.Busy != "Committing…" || !strings.Contains(h.frame(), "Committing…") {
			t.Errorf("busy %q:\n%s", v.Busy, h.frame())
		}
		clickQueued(t, h, render.ItemCommitMore, 0, 0)
		if h.m.ui.Source.Menu {
			t.Error("the dropdown opened during an operation")
		}
		h.m.commitAction(true)
		close(hold)
		h.settle()
		if got := w.logged(""); !slices.Equal(got, []string{"commit Once"}) {
			t.Errorf("writes %q", got)
		}
		w.set(func(f *fakeGit) {
			f.snap.Entries = append(f.snap.Entries, input.GitEntry{X: 'A', Y: ' ', Path: "n.go", RepoPath: "n.go"})
		})
		refresh(h)
		w.commitErr = errors.New("index locked")
		typeMessage(t, h, "Fails once")
		h.send(ctrlEnter)
		h.settle()
		if got := w.logged("commit"); len(got) != 2 {
			t.Errorf("a failure retried: %q", got)
		}
	})
}

// TestCommitDropdown is criterion 45.13 (design D4): hidden by default; the
// button's left part acts and opens nothing; the chevron opens it; a click
// outside closes it and does nothing else; Esc closes it; choosing closes
// it; the pointer on the item puts it on the band, off it not; the
// button's look the same under the pointer.
func TestCommitDropdown(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, w := commitRig(t)
		h.send(PrimaryDeviceAttributesEvent{61, 4, 6, 7, 14}) // Sixel: the button's image and its look
		if h.m.ui.Source.Menu || strings.Contains(h.frame(), render.MenuLabel) {
			t.Fatal("the dropdown shows by default")
		}
		look := screenNow(h).Commit.Look
		bx, by := itemCell(t, h, render.SourcePanel, render.ItemCommitMain)
		mouseAt(t, h, moveTo(bx+3, by+1))
		if l := screenNow(h).Commit.Look; l != look || h.m.ui.Source.MenuHot {
			t.Errorf("the pointer over the button: look %+v, was %+v", l, look)
		}
		typeMessage(t, h, "Left part")
		clickItem(t, h, render.ItemCommitMain, 0, 0)
		if h.m.ui.Source.Menu || len(w.logged("commit")) != 1 {
			t.Fatalf("the left part: menu %v, writes %q", h.m.ui.Source.Menu, w.log)
		}
		clickItem(t, h, render.ItemCommitMore, 0, 0)
		if !h.m.ui.Source.Menu || !screenNow(h).Menu.Shown {
			t.Fatal("the chevron did not open it")
		}
		cur, focus := h.m.ui.Source.Cursor, h.m.ui.Focus
		mouseClick(t, h, 5, 10) // zone A
		if h.m.ui.Source.Menu || h.m.ui.Focus != focus || h.m.ui.Source.Cursor != cur || len(w.logged("")) != 1 {
			t.Errorf("a click outside: menu %v, focus %q, cursor %q, writes %q", h.m.ui.Source.Menu, h.m.ui.Focus, h.m.ui.Source.Cursor, w.log)
		}
		clickItem(t, h, render.ItemCommitMore, 0, 0)
		h.send(keyOf(tea.KeyEscape))
		if h.m.ui.Source.Menu {
			t.Error("Esc left it open")
		}
		clickItem(t, h, render.ItemCommitMore, 1, 0) // another cell: the frozen clock makes the same one a double click
		mx, my := itemCell(t, h, render.SourcePanel, render.ItemCommitMenu)
		mouseAt(t, h, moveTo(mx+4, my+1))
		if !h.m.ui.Source.MenuHot || !screenNow(h).Menu.Look.Hot {
			t.Error("the pointer on the item: no band")
		}
		mouseAt(t, h, moveTo(mx+4, my+5))
		if h.m.ui.Source.MenuHot || screenNow(h).Menu.Look.Hot {
			t.Error("the pointer off the item: still on the band")
		}
		h.send(keyOf(tea.KeyEscape)) // the dropdown closed, so the click reaches the field
		typeMessage(t, h, "x")
		w.set(func(f *fakeGit) {
			f.snap.Entries = append(f.snap.Entries, input.GitEntry{X: 'M', Y: ' ', Path: "z.go", RepoPath: "z.go"})
		})
		refresh(h)
		clickItem(t, h, render.ItemCommitMore, 0, 0)
		clickItem(t, h, render.ItemCommitMenu, 3, 1)
		if h.m.ui.Source.Menu || len(w.logged("push")) != 1 {
			t.Errorf("choosing: menu %v, writes %q", h.m.ui.Source.Menu, w.log)
		}
	})
}

// TestCommitKeyboard is criterion 45.14 (designs D2 and D3): ↑ from the
// list's first row to the button, ↑ to the field, ↓ back down to the list;
// Enter on the button acts; → opens the dropdown, Enter chooses; Esc
// leaves.
func TestCommitKeyboard(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, w := commitRig(t)
		h.send(keyOf(tea.KeyHome))
		h.send(keyOf(tea.KeyUp))
		if p := h.m.ui.Source.Place; p != render.PlaceBtn {
			t.Fatalf("↑ from the first row: %q", p)
		}
		if f := h.frame(); !strings.Contains(f, "Enter commit") {
			t.Errorf("the button's keys not in the footer:\n%s", f)
		}
		h.send(keyOf(tea.KeyUp))
		if p := h.m.ui.Source.Place; p != render.PlaceMsg {
			t.Fatalf("↑ from the button: %q", p)
		}
		for _, r := range "Keys" {
			h.send(typed(r))
		}
		h.send(keyOf(tea.KeyDown))
		h.send(keyOf(tea.KeyDown))
		if p := h.m.ui.Source.Place; p != "" {
			t.Fatalf("↓↓ from the field: %q", p)
		}
		h.send(keyOf(tea.KeyUp))
		h.send(keyOf(tea.KeyEnter))
		if got := w.logged(""); !slices.Equal(got, []string{"commit Keys"}) {
			t.Fatalf("Enter on the button: %q", got)
		}
		reads := len(w.calledStatus())
		w.set(func(f *fakeGit) {
			f.snap.Entries = append(f.snap.Entries, input.GitEntry{X: 'M', Y: ' ', Path: "k.go", RepoPath: "k.go"})
		})
		h.send(typed('r')) // on the button: nothing typed, nothing read
		if h.m.msg.Text() != "" || len(w.calledStatus()) != reads {
			t.Errorf("r on the button: typed %q, read again", h.m.msg.Text())
		}
		h.send(keyOf(tea.KeyDown))
		h.send(typed('r')) // on the list: a read
		h.send(keyOf(tea.KeyUp))
		h.send(keyOf(tea.KeyUp))
		for _, r := range "More" {
			h.send(typed(r))
		}
		h.send(keyOf(tea.KeyDown))
		h.send(keyOf(tea.KeyRight))
		if v := h.m.ui.Source; !v.Menu || !v.MenuHot {
			t.Fatalf("→ on the button: menu %v, hot %v", v.Menu, v.MenuHot)
		}
		h.send(keyOf(tea.KeyEnter))
		if got := w.logged(""); !slices.Equal(got, []string{"commit Keys", "target", "commit More", "push"}) {
			t.Errorf("→ then Enter: %q", got)
		}
		h.send(keyOf(tea.KeyEscape))
		if h.m.ui.Focus != "B" {
			t.Errorf("Esc: focus %q", h.m.ui.Focus)
		}
	})
}

// isolateGit makes git ignore the owner's and the system's configuration
// (no credential helper, hook or signing of theirs), its identity the
// test's, for TestCommitGit's real git.
func isolateGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	empty := filepath.Join(t.TempDir(), "gitconfig")
	if err := os.WriteFile(empty, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	for k, v := range map[string]string{"GIT_CONFIG_GLOBAL": empty, "GIT_CONFIG_NOSYSTEM": "1",
		"GIT_AUTHOR_NAME": "test", "GIT_AUTHOR_EMAIL": "test@example.invalid",
		"GIT_COMMITTER_NAME": "test", "GIT_COMMITTER_EMAIL": "test@example.invalid"} {
		t.Setenv(k, v)
	}
}

// gitRun runs git in dir, the test failing if it fails.
func gitRun(t *testing.T, dir string, args ...string) string {
	t.Helper()
	if dir != "" {
		args = append([]string{"-C", dir}, args...)
	}
	out, err := exec.Command("git", args...).CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return strings.TrimSpace(string(out))
}

// TestCommitGit is criteria 45.10 and 45.11 with git itself, on a
// temporary repository whose remote is a local bare repository: Commit,
// then Push, the button on Git's word; Commit & Push against a remote whose
// pre-receive hook refuses, the commit kept, then Push once the hook is
// gone, the remote holding the commit once. Nothing touches this
// repository.
func TestCommitGit(t *testing.T) {
	isolateGit(t)
	r := claudeRig(t)
	h := r.h
	root := h.m.State().Context.Project
	bare := filepath.Join(t.TempDir(), "remote.git")
	gitRun(t, "", "init", "-q", "--bare", "-b", "main", bare)
	gitRun(t, root, "init", "-q", "-b", "main")
	edit := func(text string) {
		if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte(text), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	edit("one\n")
	gitRun(t, root, "add", "a.txt")
	gitRun(t, root, "commit", "-q", "-m", "init")
	gitRun(t, root, "remote", "add", "origin", bare)
	gitRun(t, root, "push", "-q", "-u", "origin", "main")
	edit("one\ntwo\n")
	gitRun(t, root, "add", "a.txt")

	h.send(tea.WindowSizeMsg{Width: 200, Height: 60})
	for _, c := range h.m.showSource() {
		h.exec(c)
	}
	// until settles until ready holds, at most a minute: real git, under
	// the race detector's load, reads the status in more than the harness
	// waits for a message.
	until := func(ready func() bool) {
		for i := 0; i < 200 && !ready(); i++ {
			h.settle()
		}
	}
	commitable := func() bool { v := h.m.ui.Source; return v.Action == render.ActionCommit && !v.Dim }
	until(commitable)
	if v := h.m.ui.Source; v.Action != render.ActionCommit || v.Dim {
		t.Fatalf("before: action %q, dim %v, note %q", v.Action, v.Dim, v.Note)
	}
	// act sends a key, then waits for the operation it started to end: real
	// git takes longer than the harness waits for a message.
	act := func(k tea.KeyPressMsg) {
		h.send(k)
		for i := 0; i < 200 && h.m.op != nil; i++ {
			h.settle()
		}
		if h.m.op != nil {
			t.Fatal("the operation never ended")
		}
	}
	toField := func(msg string) {
		h.send(keyOf(tea.KeyHome))
		h.send(keyOf(tea.KeyUp))
		h.send(keyOf(tea.KeyUp))
		for _, k := range typedText(msg) {
			h.send(k)
		}
	}

	// Workflow A: Commit, then Push.
	toField("First change")
	act(ctrlEnter)
	if got := gitRun(t, root, "log", "-1", "--format=%s"); got != "First change" {
		t.Fatalf("HEAD's subject %q", got)
	}
	if v := h.m.ui.Source; v.Action != render.ActionPush || h.m.msg.Text() != "" || !strings.HasPrefix(v.Notice, "Committed ") {
		t.Fatalf("after the commit: action %q, field %q, note %q", v.Action, h.m.msg.Text(), v.Notice)
	}
	if !strings.Contains(h.frame(), "Push") {
		t.Errorf("the button not on Push:\n%s", h.frame())
	}
	act(ctrlEnter) // the main action, now Push
	if local, remote := gitRun(t, root, "rev-parse", "HEAD"), gitRun(t, bare, "rev-parse", "main"); local != remote {
		t.Fatalf("after Push: local %s, remote %s; note %q", local, remote, h.m.ui.Source.Notice)
	}
	if v := h.m.ui.Source; v.Action != render.ActionCommit || v.Notice != "Pushed to origin/main." {
		t.Errorf("after Push: action %q, note %q", v.Action, v.Notice)
	}

	// Workflow B against a refusing remote: the commit kept, Push after.
	hook := filepath.Join(bare, "hooks", "pre-receive")
	if err := os.WriteFile(hook, []byte("#!/bin/sh\necho 'refused by the test' >&2\nexit 1\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	edit("one\ntwo\nthree\n")
	gitRun(t, root, "add", "a.txt")
	h.exec(h.m.readSnapshot())
	until(commitable)
	toField("Second change")
	h.send(keyOf(tea.KeyDown))
	h.send(keyOf(tea.KeyRight))
	act(keyOf(tea.KeyEnter))
	v := h.m.ui.Source
	if !strings.HasPrefix(v.Notice, "Committed ") || !strings.Contains(v.Notice, "but the push failed: The remote rejected the push: pre-receive hook declined.") {
		t.Fatalf("note %q", v.Notice)
	}
	if v.Action != render.ActionPush || v.Menu || h.m.msg.Text() != "" {
		t.Fatalf("after the refused push: action %q, menu %v, field %q", v.Action, v.Menu, h.m.msg.Text())
	}
	if got := gitRun(t, root, "log", "-1", "--format=%s"); got != "Second change" {
		t.Fatalf("HEAD's subject %q", got)
	}
	if err := os.Remove(hook); err != nil {
		t.Fatal(err)
	}
	act(keyOf(tea.KeyEnter)) // the button has the keys: Push
	if local, remote := gitRun(t, root, "rev-parse", "HEAD"), gitRun(t, bare, "rev-parse", "main"); local != remote {
		t.Fatalf("after the second Push: local %s, remote %s; note %q", local, remote, h.m.ui.Source.Notice)
	}
	if got := gitRun(t, bare, "log", "--format=%s", "main"); got != "Second change\nFirst change\ninit" {
		t.Errorf("the remote's history:\n%s", got)
	}
	if got := gitRun(t, root, "status", "--porcelain"); got != "" {
		t.Errorf("the working tree after: %q", got)
	}
}

// TestCommitLongOperation is criterion 45.19 through the application
// (design D15): a running push's label carries its seconds from the second
// one ("Pushing… 12 s", then "Pushing… 1:05") and the note says Esc cancels
// it; Esc while a lock file is held says why and the push goes on; Esc then
// stops it, "Cancelling…", and the note says it was cancelled, the button
// left on Push; Commit & Push's cancelled push keeps its commit and says so;
// Esc during a commit leaves it running.
func TestCommitLongOperation(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, w := commitRig(t)
		typeMessage(t, h, "Long")
		h.send(ctrlEnter) // the commit: the button on Push
		w.mu.Lock()
		w.stoppable = true
		w.mu.Unlock()
		clickQueued(t, h, render.ItemCommitMain, 0, 0)
		v := &h.m.ui.Source
		if !h.m.pushing() || v.Busy != "Pushing…" || v.BusyHint != "" {
			t.Fatalf("the push: running %v, label %q, hint %q", h.m.pushing(), v.Busy, v.BusyHint)
		}
		for _, c := range []struct {
			d     time.Duration
			label string
		}{{12 * time.Second, "Pushing… 12 s"}, {65 * time.Second, "Pushing… 1:05"}} {
			h.m.op.started = h.m.ui.Now.Add(-c.d)
			h.m.syncCommit()
			if v.Busy != c.label || v.BusyHint != "Esc cancels the push." || v.BusyWarn {
				t.Errorf("after %v: label %q, hint %q", c.d, v.Busy, v.BusyHint)
			}
		}
		if f := h.frame(); !strings.Contains(f, "Pushing… 1:05") || !strings.Contains(f, "Esc cancels the push.") || !strings.Contains(f, "Esc cancel the push") {
			t.Errorf("the label, the note or the footer not on screen:\n%s", f)
		}
		w.mu.Lock()
		w.stopBusy = true
		w.mu.Unlock()
		h.send(keyOf(tea.KeyEscape))
		if !h.m.pushing() || h.m.op.cancelling || v.BusyHint != "git is updating the repository: try again in a moment." || !v.BusyWarn {
			t.Fatalf("a refused stop: running %v, hint %q", h.m.pushing(), v.BusyHint)
		}
		w.mu.Lock()
		w.stopBusy = false
		w.mu.Unlock()
		h.deliver(keyOf(tea.KeyEscape))
		if v.Busy != "Cancelling…" || v.BusyHint != "" {
			t.Errorf("stopping: label %q, hint %q", v.Busy, v.BusyHint)
		}
		h.settle()
		if v.Notice != "The push was cancelled. The remote may have received it: Push again to check." || v.NoticeKind != render.NoticeCheck ||
			h.m.op != nil || v.Action != render.ActionPush || len(w.logged("push cancelled")) != 1 {
			t.Fatalf("cancelled: note %q, action %q, writes %q", v.Notice, v.Action, w.log)
		}

		// Commit & Push, its push cancelled: the commit kept and said.
		w.set(func(f *fakeGit) {
			f.snap.Entries = append(f.snap.Entries, input.GitEntry{X: 'M', Y: ' ', Path: "x.go", RepoPath: "x.go"})
		})
		refresh(h)
		typeMessage(t, h, "Both")
		clickItem(t, h, render.ItemCommitMore, 0, 0)
		clickQueued(t, h, render.ItemCommitMenu, 3, 1)
		if !h.m.pushing() || h.m.msg.Text() != "" {
			t.Fatalf("Commit & Push: pushing %v, field %q, writes %q", h.m.pushing(), h.m.msg.Text(), w.log)
		}
		h.send(keyOf(tea.KeyEscape))
		if want := "Committed abc0002; the push was cancelled. The remote may have received it: Push again to check."; v.Notice != want {
			t.Errorf("note %q\nwant %q", v.Notice, want)
		}

		// Esc during a commit: it runs on, Esc doing what it does.
		hold := w.holdOn()
		w.set(func(f *fakeGit) {
			f.snap.Entries = append(f.snap.Entries, input.GitEntry{X: 'M', Y: ' ', Path: "y.go", RepoPath: "y.go"})
		})
		refresh(h)
		typeMessage(t, h, "Held")
		clickItem(t, h, render.ItemCommitMore, 1, 0)
		clickQueued(t, h, render.ItemCommitMenu, 3, 1)
		if h.m.op == nil || h.m.op.steps[0] != stepCommit {
			t.Fatalf("the commit not running: %+v", h.m.op)
		}
		h.send(keyOf(tea.KeyEscape))
		if h.m.op == nil || h.m.op.cancelling || h.m.ui.Focus != "B" {
			t.Fatalf("Esc during the commit: running %v, focus %q", h.m.op != nil, h.m.ui.Focus)
		}
		w.mu.Lock()
		w.stoppable = false
		w.mu.Unlock()
		close(hold)
		h.settle()
		if got := w.logged("commit Held"); len(got) != 1 || v.Notice != "Committed abc0003 and pushed to origin/main." {
			t.Errorf("after the held commit: writes %q, note %q", w.log, v.Notice)
		}
	})
}

// TestCommitTicks is design D15's tick: an operation asks one a second
// after it starts, and each of its ticks the next, the label then showing
// the seconds run; a tick of an operation that ended, or of an earlier
// one, asks none.
func TestCommitTicks(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, w := commitRig(t)
		var mu sync.Mutex
		var ticks []opTickMsg
		var waits []time.Duration
		h.m.opt.After = func(d time.Duration, msg tea.Msg) tea.Cmd {
			if tk, ok := msg.(opTickMsg); ok {
				mu.Lock()
				ticks, waits = append(ticks, tk), append(waits, d)
				mu.Unlock()
			}
			return nil
		}
		asked := func() []opTickMsg {
			mu.Lock()
			defer mu.Unlock()
			return slices.Clone(ticks)
		}
		var clockMu sync.Mutex
		now := testNow
		h.m.opt.Now = func() time.Time {
			clockMu.Lock()
			defer clockMu.Unlock()
			return now
		}
		advance := func(d time.Duration) {
			clockMu.Lock()
			now = now.Add(d)
			clockMu.Unlock()
		}
		typeMessage(t, h, "Tick")
		hold := w.holdOn()
		h.send(ctrlEnter) // the commit, held
		first := asked()
		if len(first) != 1 || h.m.op == nil || first[0].seq != h.m.op.seq || waits[0] != time.Second {
			t.Fatalf("the commit started: ticks %v after %v", first, waits)
		}
		advance(3 * time.Second)
		h.send(first[0])
		if got := asked(); len(got) != 2 || got[1] != first[0] || h.m.ui.Source.Busy != "Committing… 3 s" || !strings.Contains(h.frame(), "Committing… 3 s") {
			t.Errorf("a tick: ticks %v, label %q", got, h.m.ui.Source.Busy)
		}
		close(hold)
		h.settle()
		if h.m.op != nil || h.m.ui.Source.Action != render.ActionPush {
			t.Fatalf("after the commit: running %v, action %q", h.m.op != nil, h.m.ui.Source.Action)
		}
		h.send(first[0])
		if got := asked(); len(got) != 2 {
			t.Errorf("a tick of the commit ended asked another: %v", got)
		}

		// The push: its own tick; the commit's asks none.
		w.mu.Lock()
		w.stoppable = true
		w.mu.Unlock()
		clickQueued(t, h, render.ItemCommitMain, 0, 0)
		got := asked()
		if len(got) != 3 || h.m.op == nil || got[2].seq != h.m.op.seq || got[2] == first[0] {
			t.Fatalf("the push started: ticks %v", got)
		}
		h.send(first[0])
		if n := len(asked()); n != 3 {
			t.Errorf("the commit's tick during the push asked %d", n-3)
		}
		h.send(got[2])
		if n := len(asked()); n != 4 {
			t.Errorf("the push's tick asked %d", n-3)
		}
		h.send(keyOf(tea.KeyEscape))
		if h.m.op != nil {
			t.Fatal("the push not cancelled")
		}
	})
}

// TestCommitQuitAsks is plan 47's criterion 47.1 (design D1): Ctrl+Q while
// a commit or a push runs asks to confirm, naming it, and quits only on a
// second Ctrl+Q; Esc stays, the operation going on.
func TestCommitQuitAsks(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, w := commitRig(t)
		typeMessage(t, h, "Slow")
		hold := w.holdOn()
		h.send(ctrlEnter) // the commit, held
		h.send(ctrlQ)
		if !h.m.State().QuitAsked || h.quit {
			t.Fatalf("Ctrl+Q during the commit: asked %v, quit %v", h.m.State().QuitAsked, h.quit)
		}
		if f := h.frame(); !strings.Contains(f, "quit? the commit is still running · Ctrl+Q again to quit · Esc to stay") {
			t.Errorf("the question does not name the commit:\n%s", f)
		}
		h.send(keyOf(tea.KeyEscape))
		if h.m.State().QuitAsked || h.quit || h.m.op == nil {
			t.Fatalf("Esc: asked %v, quit %v, the commit running %v", h.m.State().QuitAsked, h.quit, h.m.op != nil)
		}
		w.mu.Lock()
		w.stoppable = true
		w.mu.Unlock()
		close(hold)
		h.settle()
		clickQueued(t, h, render.ItemCommitMain, 0, 0) // the push
		h.send(ctrlQ)
		if f := h.frame(); !h.m.State().QuitAsked || !strings.Contains(f, "quit? the push is still running · Ctrl+Q again to quit · Esc to stay") {
			t.Errorf("Ctrl+Q during the push: asked %v\n%s", h.m.State().QuitAsked, f)
		}
		h.send(ctrlQ)
		if s := h.m.State(); s.QuitAsked || s.Startup.Phase != state.Exited {
			t.Errorf("a second Ctrl+Q: asked %v, startup %q, want the quit under way", s.QuitAsked, s.Startup.Phase)
		}
		h.send(keyOf(tea.KeyEscape)) // the push stopped, nothing left waiting
	})
}
