package app

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/Evengelius/agent-command-center/internal/event"
	"github.com/Evengelius/agent-command-center/internal/input"
	"github.com/Evengelius/agent-command-center/internal/provider"
	"github.com/Evengelius/agent-command-center/internal/provider/claude"
	"github.com/Evengelius/agent-command-center/internal/render"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// Plan 59 (brief amendment 61): the branches popup through the application,
// git's answers from the test double; the HEAD watch on real repositories.

// fakeBranches is the popup's git over fakeGit: its lists, the changes it
// counts, its switch's and fetch's answers, the HEAD file's path; each call
// recorded. With stopper, a fetch runs until a stop is asked, stopper its
// way to stop.
type fakeBranches struct {
	*fakeGit
	b         input.Branches
	readErr   error
	changes   int
	switchErr error
	fetchErr  error
	stopper   func() (bool, string)
	head      string
}

func (f *fakeBranches) Branches(dir string) (input.Branches, error) {
	f.wait()
	f.record("branches")
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.b, f.readErr
}

func (f *fakeBranches) Changes(dir string) (int, error) {
	f.record("changes")
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.changes, nil
}

func (f *fakeBranches) Switch(dir, name string, remote bool) (string, error) {
	f.record(fmt.Sprintf("switch %s %v", name, remote))
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.switchErr != nil {
		return "", f.switchErr
	}
	if remote {
		_, name, _ = strings.Cut(name, "/")
	}
	return name, nil
}

func (f *fakeBranches) Fetch(dir string, stop *input.GitStop) error {
	f.record("fetch")
	f.mu.Lock()
	stopper, err := f.stopper, f.fetchErr
	f.mu.Unlock()
	if stopper == nil {
		return err
	}
	stop.Set(stopper)
	<-stop.Asked()
	stop.Set(nil)
	return input.Refusal{Kind: "cancelled", Msg: "The fetch was stopped."}
}

func (f *fakeBranches) HeadFile(dir string) (string, error) {
	f.record("head")
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.head == "" {
		return "", errors.New("no HEAD file")
	}
	return f.head, nil
}

// sampleBranches is a repository on main, ahead 2 and behind 1 of
// origin/main: feat and fix/login, last switched to; wtb checked out in
// another worktree; two remotes; three tags, newest first.
func sampleBranches() input.Branches {
	return input.Branches{
		Head:   input.GitBranch{Name: "main", Upstream: "origin/main", Ahead: 2, Behind: 1},
		Recent: []string{"feat", "fix/login"},
		Local: []input.Ref{
			{Name: "feat", Upstream: "origin/feat"},
			{Name: "fix/login"},
			{Name: "main", Upstream: "origin/main", Ahead: 2, Behind: 1, Current: true},
			{Name: "wtb", Elsewhere: `C:\work\wtb`},
		},
		Remote:     []input.Ref{{Name: "backup/main"}, {Name: "origin/away"}, {Name: "origin/feat"}, {Name: "origin/main"}},
		Tags:       []input.Ref{{Name: "v2.0"}, {Name: "v1.1"}, {Name: "v1.0"}},
		LocalTotal: 4, RemoteTotal: 4, TagsTotal: 3,
	}
}

// branchesRig is the scripted Codex session's dashboard at 120x40 on the
// file rig's project (fileview_test.go), which the fake says is a
// repository on main.
func branchesRig(t *testing.T) (inputRig, *fakeBranches, *fakeClipboard) {
	r, clip, _ := fileRig(t)
	f := &fakeBranches{fakeGit: &fakeGit{}, b: sampleBranches()}
	r.h.m.opt.Git = f
	r.h.send(actionMsg{state.BranchRead{Branch: "main"}})
	return r, f, clip
}

// altB is Alt+B.
var altB = key('b', tea.ModAlt)

// openBranches opens the popup with Alt+B, its lists read.
func openBranches(t *testing.T, h *harness) {
	t.Helper()
	h.send(altB)
	if v := h.m.ui.Branches; !v.Open || v.Reading {
		t.Fatalf("Alt+B: open %v, reading %v, problem %q", v.Open, v.Reading, v.Problem)
	}
}

// branchTo puts the popup's cursor on the row with key, by ↑ and ↓.
func branchTo(t *testing.T, h *harness, key string) {
	t.Helper()
	h.send(keyOf(tea.KeyHome))
	for range 40 {
		if h.m.ui.Branches.Cursor == key {
			return
		}
		h.send(keyOf(tea.KeyDown))
	}
	t.Fatalf("no row %q: the cursor on %q", key, h.m.ui.Branches.Cursor)
}

// TestBranchesOpenClose is plan 59, criterion 59.1 (design D1): a click on
// the branch name and Alt+B open the popup and close it again, Esc with the
// search empty closes it, Esc first clears a search; a click outside closes
// it and does nothing else; it closes the "+" menu; it opens over no window
// and no full view; outside a repository a toast says why. Its lists are
// read once each time it opens; nothing is watched without a period.
func TestBranchesOpenClose(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r, f, _ := branchesRig(t)
		h := r.h
		bx, by := at(t, h, "branch main", 0)
		before := h.m.ui
		mouseClick(t, h, bx+len("branch "), by)
		v := h.m.ui.Branches
		if !v.Open || v.Reading || v.Cursor != "b:1:main" || v.Folded != [render.BranchGroups]bool{false, false, true, true} {
			t.Fatalf("the click: open %v, reading %v, cursor %q, folded %v", v.Open, v.Reading, v.Cursor, v.Folded)
		}
		frame := h.frame()
		for _, want := range []string{"╭─ Branches ", "Search branches and tags", " main → origin/main ↑2 ↓1", "⌄ Recent", "⌄ Local", "› Remote", "› Tags", "✓ main"} {
			if !strings.Contains(frame, want) {
				t.Errorf("the popup lacks %q:\n%s", want, frame)
			}
		}
		mouseClick(t, h, bx+len("branch "), by)
		if h.m.ui.Branches.Open {
			t.Fatal("a second click on the name left the popup open")
		}
		openBranches(t, h)
		h.send(altB)
		if h.m.ui.Branches.Open {
			t.Fatal("a second Alt+B left the popup open")
		}
		openBranches(t, h)
		h.send(keyOf(tea.KeyEscape))
		if h.m.ui.Branches.Open {
			t.Fatal("Esc with the search empty left the popup open")
		}
		openBranches(t, h)
		for _, k := range typedText("fe") {
			h.send(k)
		}
		h.send(keyOf(tea.KeyEscape))
		if v := h.m.ui.Branches; !v.Open || v.Query != "" {
			t.Fatalf("Esc on a search: open %v, query %q", v.Open, v.Query)
		}
		h.send(keyOf(tea.KeyEscape))
		if h.m.ui.Branches.Open {
			t.Fatal("the second Esc left the popup open")
		}
		if got := f.called("branches"); len(got) != 4 {
			t.Errorf("the lists read %d times, want once each of 4 openings", len(got))
		}

		// A click outside closes it and does nothing else: on zone B's
		// text no selection starts, the interface as before it opened; on
		// the Explorer's tile no panel shows.
		openBranches(t, h)
		mouseClick(t, h, 20, 20)
		after := h.m.ui
		after.Memo, before.Memo, after.Wrap, before.Wrap = nil, nil, nil, nil
		if d := uiDiff(after, before); d != "" || h.m.drag.on {
			t.Errorf("a click outside changed %s; a drag %v", d, h.m.drag.on)
		}
		openBranches(t, h)
		tx, ty := itemCell(t, h, render.SidePanel, render.SideExplorer)
		mouseClick(t, h, tx, ty)
		if h.m.ui.Branches.Open || h.m.ui.Explorer.Open {
			t.Errorf("a click on the Explorer's tile: popup %v, Explorer %v", h.m.ui.Branches.Open, h.m.ui.Explorer.Open)
		}

		// It closes the "+" menu, and a click on "+" closes it without
		// opening the menu.
		px, py := at(t, h, "+     ", 0)
		mouseClick(t, h, px+1, py)
		if !h.m.ui.Plus {
			t.Fatal("the + menu did not open")
		}
		h.send(altB)
		if !h.m.ui.Branches.Open || h.m.ui.Plus {
			t.Errorf("Alt+B over the + menu: popup %v, menu %v", h.m.ui.Branches.Open, h.m.ui.Plus)
		}
		mouseClick(t, h, px+1, py)
		if h.m.ui.Branches.Open || h.m.ui.Plus {
			t.Errorf("a click on +: popup %v, menu %v", h.m.ui.Branches.Open, h.m.ui.Plus)
		}

		// No popup over a window: Alt+B in the file viewer, a click on the
		// name outside it, which closes the window only (plan 58).
		openPath(h, "main.go")
		h.send(altB)
		if h.m.ui.Branches.Open {
			t.Error("Alt+B opened the popup over the file viewer")
		}
		mouseClick(t, h, bx+len("branch "), by)
		if h.m.ui.Branches.Open || h.m.ui.File.Open {
			t.Errorf("a click on the name over the viewer: popup %v, viewer %v", h.m.ui.Branches.Open, h.m.ui.File.Open)
		}
		// Nor over a full view, drawn or not yet: the toast says why.
		h.m.openDetails()
		h.send(altB)
		if h.m.ui.Branches.Open || h.m.ui.Toast != "the branches popup needs the project bar's branch on screen" {
			t.Errorf("Alt+B over the details not drawn yet: popup %v, toast %q", h.m.ui.Branches.Open, h.m.ui.Toast)
		}
		h.m.View()
		h.send(altB)
		if h.m.ui.Branches.Open || h.m.ui.Toast != "the branches popup needs the project bar's branch on screen" {
			t.Errorf("Alt+B over the details: popup %v, toast %q", h.m.ui.Branches.Open, h.m.ui.Toast)
		}
		h.send(keyOf(tea.KeyEscape))

		// Outside a repository: no branch in the header, a toast.
		h.send(actionMsg{state.BranchRead{Branch: ""}})
		if strings.Contains(h.frame(), "branch main") {
			t.Error("the header still names a branch")
		}
		h.send(altB)
		if h.m.ui.Branches.Open || h.m.ui.Toast != "no branches: the project is not in a git repository" {
			t.Errorf("Alt+B outside a repository: popup %v, toast %q", h.m.ui.Branches.Open, h.m.ui.Toast)
		}
		if got := f.called("head"); len(got) != 0 {
			t.Errorf("no watch period, yet the HEAD file was asked: %v", got)
		}
	})
}

// TestBranchesFromPanels is criterion 59.1's "from any focus" (design D1):
// Alt+B opens the popup with the focus in the Explorer's panel, in Source
// Control's list, in its commit field and its button, and in a file's
// diff, each passing it on to the dashboard as it does Alt+P.
func TestBranchesFromPanels(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r, g, _ := sourceRig(t)
		h := r.h
		h.m.opt.Git = &fakeBranches{fakeGit: g, b: sampleBranches()}
		h.send(actionMsg{state.BranchRead{Branch: "main"}})
		check := func(where string) {
			t.Helper()
			h.send(altB)
			if !h.m.ui.Branches.Open {
				t.Errorf("%s: Alt+B left the popup closed (focus %q, place %q)", where, h.m.ui.Focus, h.m.ui.Source.Place)
			}
			h.send(altB)
		}
		h.send(keyOf(tea.KeyF3))
		if !h.m.ui.Explorer.Open || h.m.ui.Focus != "R" {
			t.Fatalf("F3: the Explorer %v, focus %q", h.m.ui.Explorer.Open, h.m.ui.Focus)
		}
		check("the Explorer")
		showSource(t, h)
		check("Source Control's list")
		for _, k := range []tea.KeyPressMsg{keyOf(tea.KeyHome), keyOf(tea.KeyUp), keyOf(tea.KeyUp)} {
			h.send(k)
		}
		if p := h.m.ui.Source.Place; p != render.PlaceMsg {
			t.Fatalf("the commit field: place %q", p)
		}
		check("the commit field")
		h.send(keyOf(tea.KeyDown))
		if p := h.m.ui.Source.Place; p != render.PlaceBtn {
			t.Fatalf("the commit button: place %q", p)
		}
		check("the commit button")
		openDiffOf(t, h, "f:changes:main.go")
		check("a file's diff")
	})
}

// TestBranchesKeys is criteria 59.2 and 59.3 in the application (design
// D10): the groups as read, Recent's rows with Local's upstreams; ↑↓, Home,
// End, → and ← over the headings and rows, Enter on a heading; Ctrl+C
// copies a name, never interrupts; the search typed, pasted and cleared,
// the cursor on its first match; a search matching nothing leaves no row
// to act on.
func TestBranchesKeys(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r, _, clip := branchesRig(t)
		h := r.h
		openBranches(t, h)
		v := h.m.ui.Branches
		if got := v.Groups[render.GroupRecent]; len(got) != 2 || got[0] != (render.BranchRow{Name: "feat", Upstream: "origin/feat"}) || got[1].Name != "fix/login" {
			t.Errorf("Recent %+v", got)
		}
		if v.Totals != [render.BranchGroups]int{2, 4, 4, 3} || v.Head != "main" || v.Upstream != "origin/main" || v.Ahead != 2 || v.Behind != 1 {
			t.Errorf("totals %v, head %q → %q ↑%d ↓%d", v.Totals, v.Head, v.Upstream, v.Ahead, v.Behind)
		}
		// The wheel over the popup moves the cursor three rows; a click on a
		// heading opens its group, a second click adding nothing.
		fx, fy := itemCell(t, h, render.BranchesPanel, render.ItemBranchFrame)
		mouseAt(t, h, tea.MouseWheelMsg{X: fx, Y: fy, Button: tea.MouseWheelDown})
		if got := h.m.ui.Branches.Cursor; got != "g:3" {
			t.Errorf("the wheel down from main: cursor %q, want g:3", got)
		}
		mouseAt(t, h, tea.MouseWheelMsg{X: fx, Y: fy, Button: tea.MouseWheelUp})
		rx, ry := itemCell(t, h, render.BranchesPanel, render.ItemBranchRow+render.BranchCursor(render.BranchRows(h.m.ui.Branches), "g:2"))
		mouseClick(t, h, rx, ry)
		mouseClick(t, h, rx, ry)
		if v := h.m.ui.Branches; v.Cursor != "g:2" || v.Folded != [render.BranchGroups]bool{false, false, false, true} {
			t.Errorf("clicks on Remote's heading: cursor %q, folded %v", v.Cursor, v.Folded)
		}
		h.send(keyOf(tea.KeyLeft))
		branchTo(t, h, "b:1:main")
		steps := []struct {
			key  tea.KeyPressMsg
			want string
		}{
			{keyOf(tea.KeyUp), "b:1:fix/login"}, {keyOf(tea.KeyDown), "b:1:main"}, {keyOf(tea.KeyDown), "b:1:wtb"},
			{keyOf(tea.KeyDown), "g:2"}, {keyOf(tea.KeyRight), "g:2"}, {keyOf(tea.KeyDown), "b:2:backup/main"},
			{keyOf(tea.KeyLeft), "g:2"}, {keyOf(tea.KeyLeft), "g:2"}, {keyOf(tea.KeyEnd), "g:3"},
			{keyOf(tea.KeyHome), "g:0"}, {keyOf(tea.KeyEnter), "g:0"}, {keyOf(tea.KeyDown), "g:1"},
		}
		for i, s := range steps {
			h.send(s.key)
			if got := h.m.ui.Branches.Cursor; got != s.want {
				t.Fatalf("step %d (%s): cursor %q, want %q", i, s.key, got, s.want)
			}
		}
		if f := h.m.ui.Branches.Folded; f != [render.BranchGroups]bool{true, false, true, true} {
			t.Errorf("folded %v after → and ← on Remote, Enter on Recent", f)
		}
		h.send(keyOf(tea.KeyEnter)) // Local folds, then opens again
		h.send(keyOf(tea.KeyEnter))
		branchTo(t, h, "g:0")
		h.send(keyOf(tea.KeyEnter)) // Recent opens again
		if f := h.m.ui.Branches.Folded; f != [render.BranchGroups]bool{false, false, true, true} {
			t.Errorf("folded %v after Enter on Local twice and Recent once", f)
		}

		// Ctrl+C copies the cursor's name, on a heading nothing; no
		// interrupt either way.
		branchTo(t, h, "b:1:main")
		h.send(ctrl('c'))
		h.send(keyOf(tea.KeyHome))
		h.send(ctrl('c'))
		if got := strings.Join(clip.writes, "|"); got != "main" || h.m.State().Turn.Phase != state.Idle {
			t.Errorf("Ctrl+C copied %q, turn %q", got, h.m.State().Turn.Phase)
		}

		// The search: typed, case ignored, the cursor on the first match;
		// cleared by Backspace, the cursor staying; pasted, its first line.
		for _, k := range typedText("LOG") {
			h.send(k)
		}
		if v := h.m.ui.Branches; v.Query != "LOG" || v.Cursor != "b:0:fix/login" {
			t.Fatalf("typed LOG: query %q, cursor %q", v.Query, v.Cursor)
		}
		for range 3 {
			h.send(keyOf(tea.KeyBackspace))
		}
		if v := h.m.ui.Branches; v.Query != "" || v.Cursor != "b:0:fix/login" {
			t.Fatalf("cleared: query %q, cursor %q", v.Query, v.Cursor)
		}
		// Ctrl+V's lone release, Windows Terminal's when the clipboard holds
		// no text, reads no image for the draft under the popup (plan 32);
		// before any paste, which the frozen clock would put just before it.
		toast := h.m.ui.Toast // Ctrl+C's: no timer clears it here
		h.send(tea.KeyReleaseMsg{Code: 'v', Mod: tea.ModCtrl})
		h.send(pasteWaitMsg{h.m.pasteWait})
		if h.m.ui.Toast != toast || len(h.m.attach) != 0 {
			t.Errorf("a lone Ctrl+V release: toast %q, %d attached", h.m.ui.Toast, len(h.m.attach))
		}
		h.send(tea.PasteMsg{Content: "v1\nnot this"})
		if v := h.m.ui.Branches; v.Query != "v1" || v.Cursor != "t:3:v1.1" || h.m.edit.Text() != "" {
			t.Fatalf("pasted: query %q, cursor %q, draft %q", v.Query, v.Cursor, h.m.edit.Text())
		}
		for _, k := range typedText("zz") {
			h.send(k)
		}
		if v := h.m.ui.Branches; v.Cursor != "" || !strings.Contains(h.frame(), "No branch or tag matches “v1zz”") {
			t.Fatalf("no match: cursor %q", v.Cursor)
		}
		h.send(keyOf(tea.KeyEnter))
		h.send(keyOf(tea.KeyEscape))
		if v := h.m.ui.Branches; !v.Open || v.Query != "" || v.Cursor == "" {
			t.Errorf("Esc: open %v, query %q, cursor %q", v.Open, v.Query, v.Cursor)
		}
		// A key the popup does not take closes it and goes on: F6 moves the
		// focus.
		h.send(keyOf(tea.KeyF6))
		if h.m.ui.Branches.Open || h.m.ui.Focus == "B" {
			t.Errorf("F6: popup %v, focus %q", h.m.ui.Branches.Open, h.m.ui.Focus)
		}
	})
}

// TestBranchesProblems is criterion 59.4 (design D3): "reading…" while git
// reads; git missing, not a repository, git's own error.
func TestBranchesProblems(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r, f, _ := branchesRig(t)
		h := r.h
		block := make(chan struct{})
		f.set(func(g *fakeGit) { g.block = block })
		h.send(altB)
		if v := h.m.ui.Branches; !v.Reading || !strings.Contains(h.frame(), "reading…") {
			t.Fatalf("while git reads: reading %v", v.Reading)
		}
		close(block)
		f.set(func(g *fakeGit) { g.block = nil })
		h.settle()
		if v := h.m.ui.Branches; v.Reading || len(v.Groups[render.GroupLocal]) != 4 {
			t.Fatalf("read: reading %v, %d local", v.Reading, len(v.Groups[render.GroupLocal]))
		}
		h.send(altB)
		for _, c := range []struct {
			err  error
			want string
		}{
			{input.ErrNoGit, "git not found"},
			{input.NotRepoError{Dir: "x"}, "not a git repository"},
			{errors.New("fatal: bad object HEAD"), "fatal: bad object HEAD"},
		} {
			f.mu.Lock()
			f.readErr = c.err
			f.mu.Unlock()
			h.send(altB)
			if v := h.m.ui.Branches; v.Problem != c.want || !strings.Contains(h.frame(), c.want) {
				t.Errorf("%v: problem %q", c.err, v.Problem)
			}
			h.send(altB)
		}
	})
}

// TestBranchesSwitch is criteria 59.5 and 59.6 (designs D5 and D6): a local
// branch, by Enter and by a double click; the confirmation with changes,
// Esc and any other key cancelling it; a remote branch made a tracking
// one; git's refusals in its words, nothing changed; a branch in another
// worktree, the current one and a tag not offered; refused while a turn
// runs. After a switch the popup closes, the header names the branch, the
// Explorer reads again, zone B has the dashboard's notice and the empty
// draft the note, nothing sent.
func TestBranchesSwitch(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r, f, _ := branchesRig(t)
		h := r.h
		h.send(tea.WindowSizeMsg{Width: 200, Height: 60}) // the Explorer docked beside zone B
		for _, c := range h.m.showExplorer() {
			h.exec(c)
		}
		h.settle()
		h.send(keyOf(tea.KeyEscape)) // the focus back to zone B, the panel kept
		if !h.m.ui.Explorer.Open || h.m.ui.Focus != "B" {
			t.Fatalf("the Explorer: open %v, focus %q", h.m.ui.Explorer.Open, h.m.ui.Focus)
		}
		statuses := len(f.called("status"))
		convs := len(h.m.State().Conversation)

		openBranches(t, h)
		branchTo(t, h, "b:1:feat")
		h.send(keyOf(tea.KeyEnter))
		if got := strings.Join(f.called("switch"), "|"); got != "switch feat false" {
			t.Fatalf("switch calls %q", got)
		}
		s := h.m.State()
		if h.m.ui.Branches.Open || s.Context.Branch != "feat" || !strings.Contains(h.frame(), "branch feat") {
			t.Errorf("after the switch: popup %v, branch %q", h.m.ui.Branches.Open, s.Context.Branch)
		}
		if len(s.Conversation) != convs+1 {
			t.Fatalf("%d entries added, want 1", len(s.Conversation)-convs)
		}
		if e := s.Conversation[convs]; e.Role != "dashboard" || e.Text != "Switched from main to feat · the agent's conversation still reflects main" {
			t.Errorf("the notice: %q %q", e.Role, e.Text)
		}
		const note = "I switched the project from main to feat; check the files you rely on before going on."
		if got := h.m.edit.Text(); got != note || len(r.sent()) != 0 {
			t.Errorf("the draft %q, sent %q", got, r.sent())
		}
		if got := len(f.called("status")); got <= statuses {
			t.Error("the Explorer's markers were not read again")
		}

		// With changes, a confirmation: Esc cancels it, another key too, Enter
		// switches; the draft holding text keeps it.
		f.mu.Lock()
		f.changes = 3
		f.mu.Unlock()
		openBranches(t, h)
		branchTo(t, h, "b:1:fix/login")
		h.send(keyOf(tea.KeyEnter))
		if v := h.m.ui.Branches; v.Confirm != "fix/login" || v.Changes != 3 ||
			!strings.Contains(h.frame(), "3 uncommitted changes. Git keeps them on fix/login") || !strings.Contains(h.frame(), "Enter switch · Esc cancel") {
			t.Fatalf("the confirmation: %q, %d", v.Confirm, v.Changes)
		}
		h.send(keyOf(tea.KeyEscape))
		h.send(keyOf(tea.KeyEnter))
		h.send(keyOf(tea.KeyDown))
		if v := h.m.ui.Branches; !v.Open || v.Confirm != "" || v.Cursor != "b:1:main" || len(f.called("switch")) != 1 {
			t.Fatalf("cancelled: open %v, confirm %q, cursor %q, switches %v", v.Open, v.Confirm, v.Cursor, f.called("switch"))
		}
		h.send(keyOf(tea.KeyUp))
		h.send(keyOf(tea.KeyEnter))
		h.send(keyOf(tea.KeyEnter))
		s = h.m.State()
		if got := f.called("switch"); len(got) != 2 || got[1] != "switch fix/login false" || s.Context.Branch != "fix/login" {
			t.Fatalf("confirmed: %v, branch %q", got, s.Context.Branch)
		}
		if e := s.Conversation[len(s.Conversation)-1]; e.Text != "Switched from feat to fix/login · 3 uncommitted changes came along · the agent's conversation still reflects feat" {
			t.Errorf("the notice %q", e.Text)
		}
		if got := h.m.edit.Text(); got != note {
			t.Errorf("a draft with text became %q", got)
		}

		// A remote branch, by a double click: a new tracking branch.
		f.mu.Lock()
		f.changes = 0
		f.mu.Unlock()
		openBranches(t, h)
		for _, k := range typedText("away") {
			h.send(k)
		}
		rows := render.BranchRows(h.m.ui.Branches)
		x, y := itemCell(t, h, render.BranchesPanel, render.ItemBranchRow+render.BranchCursor(rows, "b:2:origin/away"))
		mouseClick(t, h, x, y)
		mouseClick(t, h, x, y)
		if got := f.called("switch"); len(got) != 3 || got[2] != "switch origin/away true" || h.m.State().Context.Branch != "away" {
			t.Fatalf("the remote branch: %v, branch %q", got, h.m.State().Context.Branch)
		}

		// Git's refusals, in its words, nothing changed.
		for _, refusal := range []error{
			input.Refusal{Kind: "local-exists", Msg: "A local branch main exists already: switch to it in Local."},
			errors.New("error: Your local changes to the following files would be overwritten by checkout: a.txt"),
		} {
			f.mu.Lock()
			f.switchErr = refusal
			f.mu.Unlock()
			openBranches(t, h)
			for _, k := range typedText("origin/main") {
				h.send(k)
			}
			h.send(keyOf(tea.KeyEnter))
			if v := h.m.ui.Branches; !v.Open || v.Note != refusal.Error() || !v.NoteBad || h.m.State().Context.Branch != "away" {
				t.Errorf("refused: open %v, note %q, branch %q", v.Open, v.Note, h.m.State().Context.Branch)
			}
			h.send(altB)
		}
		f.mu.Lock()
		f.switchErr = nil
		f.mu.Unlock()

		// Not offered: another worktree's branch (a note), the current one
		// and a tag (nothing); refused while a turn runs.
		switches, counts := len(f.called("switch")), len(f.called("changes"))
		openBranches(t, h)
		branchTo(t, h, "b:1:wtb")
		h.send(keyOf(tea.KeyEnter))
		if v := h.m.ui.Branches; v.Note != `wtb is checked out in another worktree: C:\work\wtb` || !v.NoteBad {
			t.Errorf("another worktree's branch: note %q", v.Note)
		}
		branchTo(t, h, "b:1:main")
		h.send(keyOf(tea.KeyEnter))
		branchTo(t, h, "g:3")
		h.send(keyOf(tea.KeyRight))
		h.send(keyOf(tea.KeyDown))
		h.send(keyOf(tea.KeyEnter))
		r.emit(event.TurnStarted, nil)
		h.settle()
		branchTo(t, h, "b:1:feat")
		h.send(keyOf(tea.KeyEnter))
		if v := h.m.ui.Branches; v.Note != "the agent is working in this folder: switch once its turn ends" || !v.NoteBad {
			t.Errorf("during a turn: note %q", v.Note)
		}
		if len(f.called("switch")) != switches || len(f.called("changes")) != counts {
			t.Errorf("git asked: %v, %v", f.called("switch"), f.called("changes"))
		}
		r.end()
		h.send(keyOf(tea.KeyEscape))
		if h.m.ui.Branches.Open {
			t.Fatal("Esc left the popup open")
		}

		// Source Control shown in the Explorer's place is read again after a
		// switch too.
		for _, c := range h.m.showSource() {
			h.exec(c)
		}
		h.settle()
		if !h.m.ui.Source.Open || h.m.ui.Explorer.Open {
			t.Fatalf("Source Control: open %v, the Explorer %v", h.m.ui.Source.Open, h.m.ui.Explorer.Open)
		}
		statuses = len(f.called("status"))
		openBranches(t, h)
		branchTo(t, h, "b:1:feat")
		h.send(keyOf(tea.KeyEnter))
		if got := len(f.called("status")); got <= statuses || h.m.State().Context.Branch != "feat" {
			t.Errorf("a switch with Source Control shown: %d status reads, branch %q", got-statuses, h.m.State().Context.Branch)
		}
	})
}

// TestBranchesFetch is criterion 59.7 (design D7): F5 and a click on
// "fetch" fetch; Esc stops a fetch, and says why while git holds a lock;
// FetchLimit stops it; the lists are read again after each; a failure in
// git's words; a fetch outlives the popup and ends in a toast; the quit
// question names it.
func TestBranchesFetch(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r, f, _ := branchesRig(t)
		h := r.h
		running := func(stopper func() (bool, string)) {
			f.mu.Lock()
			f.stopper = stopper
			f.mu.Unlock()
		}
		ok := func() (bool, string) { return true, "" }
		openBranches(t, h)
		reads := len(f.called("branches"))

		running(ok)
		h.send(keyOf(tea.KeyF5))
		if v := h.m.ui.Branches; v.Busy != "fetching… Esc stops" || !strings.Contains(h.frame(), "fetching… ") || len(f.called("fetch")) != 1 {
			t.Fatalf("F5: busy %q, fetches %v", v.Busy, f.called("fetch"))
		}
		h.send(ctrl('q'))
		if !strings.Contains(h.frame(), "quit? the fetch is still running") {
			t.Error("the quit question does not name the fetch")
		}
		h.send(keyOf(tea.KeyEscape)) // stay
		h.send(keyOf(tea.KeyEscape)) // the stop
		if v := h.m.ui.Branches; !v.Open || v.Busy != "" || v.Note != "The fetch was stopped." || v.NoteBad || len(f.called("branches")) != reads+1 {
			t.Fatalf("Esc: open %v, busy %q, note %q bad %v, reads %d", v.Open, v.Busy, v.Note, v.NoteBad, len(f.called("branches"))-reads)
		}

		// A stop refused while git holds a lock says why; the next one stops.
		n := 0
		running(func() (bool, string) {
			if n++; n == 1 {
				return false, "git is updating the repository: try again in a moment."
			}
			return true, ""
		})
		// A click on "fetch", its command run as the program runs it (mouseAt
		// runs them inline, where this fetch would wait).
		x, y := itemCell(t, h, render.BranchesPanel, render.ItemBranchFetch)
		h.send(click(x, y))
		h.send(tea.MouseReleaseMsg{X: x, Y: y, Button: tea.MouseLeft})
		h.send(keyOf(tea.KeyEscape))
		if v := h.m.ui.Branches; v.Busy != "fetching… git is updating the repository: try again in a moment." {
			t.Fatalf("a refused stop: busy %q", v.Busy)
		}
		h.send(keyOf(tea.KeyEscape))
		if v := h.m.ui.Branches; v.Busy != "" || v.Note != "The fetch was stopped." {
			t.Fatalf("the second Esc: busy %q, note %q", v.Busy, v.Note)
		}

		// FetchLimit stops it, a failure.
		running(ok)
		h.send(keyOf(tea.KeyF5))
		h.send(fetchLimitMsg{h.m.fetch})
		if v := h.m.ui.Branches; v.Note != "The fetch ran past its 2 minutes and was stopped." || !v.NoteBad {
			t.Errorf("FetchLimit: note %q, bad %v", v.Note, v.NoteBad)
		}

		// A failure in git's words, a success; each read the lists again.
		running(nil)
		f.mu.Lock()
		f.fetchErr = errors.New("fatal: unable to access 'https://example.invalid/r.git/': Could not resolve host: example.invalid")
		f.mu.Unlock()
		reads = len(f.called("branches"))
		h.send(keyOf(tea.KeyF5))
		if v := h.m.ui.Branches; v.Note != f.fetchErr.Error() || !v.NoteBad || len(f.called("branches")) != reads+1 {
			t.Errorf("a failure: note %q, bad %v", v.Note, v.NoteBad)
		}
		f.mu.Lock()
		f.fetchErr = nil
		f.mu.Unlock()
		h.send(keyOf(tea.KeyF5))
		if v := h.m.ui.Branches; v.Note != "Fetched every remote." || v.NoteBad {
			t.Errorf("a success: note %q, bad %v", v.Note, v.NoteBad)
		}

		// The fetch outlives the popup: opened again it shows it; its end,
		// the popup closed, is a toast.
		running(ok)
		h.send(keyOf(tea.KeyF5))
		h.send(altB)
		openBranches(t, h)
		if v := h.m.ui.Branches; v.Busy != "fetching… Esc stops" {
			t.Errorf("reopened during a fetch: busy %q", v.Busy)
		}
		h.send(altB)
		h.send(fetchLimitMsg{h.m.fetch})
		if h.m.fetch != nil || h.m.ui.Toast != "The fetch ran past its 2 minutes and was stopped." {
			t.Errorf("the fetch's end, the popup closed: running %v, toast %q", h.m.fetch != nil, h.m.ui.Toast)
		}
	})
}

// headRig is a Claude Code session started through the startup screen on
// dir, with git itself and a watch every 2 s whose ticks the test delivers:
// due holds them.
func headRig(t *testing.T, dir string, git input.Git) (*harness, *[]tea.Msg) {
	t.Helper()
	cli := installedClaude{&claude.Provider{Spawner: &fakeClaude{}, Now: func() time.Time { return testNow }}}
	h := newHarness(t, Options{Providers: []provider.Provider{cli}, Cwd: dir, Git: git, BranchWatch: 2 * time.Second})
	due := &[]tea.Msg{}
	h.m.opt.After = func(_ time.Duration, msg tea.Msg) tea.Cmd {
		if _, ok := msg.(headTickMsg); ok {
			*due = append(*due, msg)
		}
		return nil
	}
	h.settle()
	h.m.ui.Choice = claude.ProviderID
	h.send(key(tea.KeyEnter, 0))
	h.send(tea.WindowSizeMsg{Width: 120, Height: 40})
	if s := h.m.State(); s.Session.Phase != state.Ready {
		t.Fatalf("session %q, startup %q", s.Session.Phase, s.Startup.Phase)
	}
	return h, due
}

// TestBranchesHeadWatch is criterion 59.8 (design D9) on real repositories:
// a switch made in another terminal, in the project and in a linked
// worktree of it, shows in the header at the next check; a check with no
// change changes nothing; without a repository nothing is watched.
func TestBranchesHeadWatch(t *testing.T) {
	isolateGit(t)
	// tick delivers the check due, until the next one is due: real git and
	// a file read run between.
	tick := func(t *testing.T, h *harness, due *[]tea.Msg) {
		t.Helper()
		n := len(*due)
		if n == 0 {
			t.Fatal("no check due")
		}
		h.send((*due)[n-1])
		for i := 0; i < 100 && len(*due) == n; i++ {
			h.settle()
		}
		if len(*due) == n {
			t.Fatal("no next check")
		}
	}
	started := func(t *testing.T, h *harness, due *[]tea.Msg) {
		t.Helper()
		for i := 0; i < 100 && len(*due) == 0; i++ {
			h.settle()
		}
		if len(*due) == 0 {
			t.Fatal("the watch never started")
		}
	}
	t.Run("project", func(t *testing.T) {
		dir := t.TempDir()
		gitRun(t, dir, "init", "-q", "-b", "main")
		gitRun(t, dir, "commit", "-q", "--allow-empty", "-m", "init")
		gitRun(t, dir, "branch", "feat")
		h, due := headRig(t, dir, input.GitCLI{})
		started(t, h, due)
		if h.m.ui.Branch != "main" {
			t.Fatalf("validated on %q", h.m.ui.Branch)
		}
		tick(t, h, due)
		if h.m.ui.Branch != "main" {
			t.Errorf("a check with no switch: %q", h.m.ui.Branch)
		}
		gitRun(t, dir, "switch", "-q", "feat")
		tick(t, h, due)
		if h.m.ui.Branch != "feat" || !strings.Contains(h.frame(), "branch feat") {
			t.Errorf("after a switch elsewhere: %q", h.m.ui.Branch)
		}
	})
	t.Run("worktree", func(t *testing.T) {
		main, dir := t.TempDir(), filepath.Join(t.TempDir(), "side")
		gitRun(t, main, "init", "-q", "-b", "main")
		gitRun(t, main, "commit", "-q", "--allow-empty", "-m", "init")
		gitRun(t, main, "worktree", "add", "-q", "-b", "side", dir)
		gitRun(t, main, "branch", "other")
		h, due := headRig(t, dir, input.GitCLI{})
		started(t, h, due)
		if h.m.ui.Branch != "side" {
			t.Fatalf("validated on %q", h.m.ui.Branch)
		}
		gitRun(t, main, "switch", "-q", "other") // the main worktree's own HEAD
		tick(t, h, due)
		if h.m.ui.Branch != "side" {
			t.Errorf("a switch in the main worktree: %q", h.m.ui.Branch)
		}
		gitRun(t, dir, "switch", "-q", "-c", "side-2")
		tick(t, h, due)
		if h.m.ui.Branch != "side-2" {
			t.Errorf("after a switch in the worktree: %q", h.m.ui.Branch)
		}
	})
	t.Run("no repository", func(t *testing.T) {
		f := &fakeBranches{fakeGit: &fakeGit{}, head: filepath.Join(t.TempDir(), "HEAD")}
		h, due := headRig(t, t.TempDir(), f)
		h.settle()
		if len(*due) != 0 || len(f.called("head")) != 0 || h.m.ui.Branch != "" {
			t.Errorf("checks %d, HEAD asked %v, branch %q", len(*due), f.called("head"), h.m.ui.Branch)
		}
	})
}

// TestBranchesFrameCost is criterion 59.10's timing (ui-design.md, Budget):
// a key in the popup with 3,000 refs, a thousand in each of Local, Remote
// and Tags, every group open, timed from Update to its frame; its 95th
// percentile within the keystroke's target, judged in the timing step.
func TestBranchesFrameCost(t *testing.T) {
	r, f, _ := branchesRig(t)
	h := r.h
	var b input.Branches
	b.Head = input.GitBranch{Name: "main"}
	for i := range input.BranchCap {
		b.Local = append(b.Local, input.Ref{Name: fmt.Sprintf("feature/topic-%04d", i), Upstream: fmt.Sprintf("origin/feature/topic-%04d", i), Ahead: i % 3})
		b.Remote = append(b.Remote, input.Ref{Name: fmt.Sprintf("origin/feature/topic-%04d", i)})
		b.Tags = append(b.Tags, input.Ref{Name: fmt.Sprintf("v1.%d.%d", i/100, i%100)})
	}
	b.Local[500].Current = true
	b.LocalTotal, b.RemoteTotal, b.TagsTotal = 1200, 1000, 1000
	f.mu.Lock()
	f.b = b
	f.mu.Unlock()
	openBranches(t, h)
	// Tags' heading is the last row, Remote's the one before while folded.
	for _, k := range []tea.KeyPressMsg{keyOf(tea.KeyEnd), keyOf(tea.KeyRight), keyOf(tea.KeyUp), keyOf(tea.KeyRight)} {
		h.send(k)
	}
	rows := render.BranchRows(h.m.ui.Branches)
	if v := h.m.ui.Branches; v.Folded != [render.BranchGroups]bool{} || len(rows) < 3000 || v.Cursor != "g:2" {
		t.Fatalf("folded %v, %d rows, cursor %q", v.Folded, len(rows), v.Cursor)
	}
	capped := false
	for _, it := range rows {
		capped = capped || it.Kind == 'm' && it.Group == render.GroupLocal && it.Text == "first 1,000 shown"
	}
	if !capped {
		t.Error("Local's cap is not noted")
	}
	branchTo(t, h, "b:1:feature/topic-0000")
	var ds []time.Duration
	for i := range 400 {
		k := keyOf(tea.KeyDown)
		if i%100 >= 50 {
			k = keyOf(tea.KeyUp)
		}
		ds = append(ds, frameTime(h, k))
	}
	p50, p95 := percentile(ds, 0.50), percentile(ds, 0.95)
	t.Logf("a key to its frame, 3,000 refs: p50 %v, p95 %v (target p95 ≤ %v)", p50, p95, keystrokeTarget)
	if p95 > keystrokeTarget {
		overTarget(t, "p95 %v over the %v target", p95, keystrokeTarget)
	}
}
