package app

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"testing/synctest"

	tea "charm.land/bubbletea/v2"

	"github.com/Evengelius/agent-command-center/internal/event"
	"github.com/Evengelius/agent-command-center/internal/input"
	"github.com/Evengelius/agent-command-center/internal/provider"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// fakeGit is a repository's worktrees in memory: every call is recorded, a
// creation adds a row, a merge brings the branch level with main, a removal
// drops the row. With block set, every call waits for it to close.
type fakeGit struct {
	mu         sync.Mutex
	rows       []input.Worktree
	listErr    error
	addErr     error
	mergeErr   error
	removeErr  error
	removeNote string
	diff       []string
	marks      input.GitMarks
	statusErr  error
	block      chan struct{}
	calls      []string
	// Source Control's (plan 44, source_test.go): the status's entries,
	// branch and top beside marks; each file's sides by "path staged", and
	// their error.
	snap     input.GitSnapshot
	sides    map[string]input.DiffPair
	sidesErr error
}

func (f *fakeGit) wait() {
	f.mu.Lock()
	b := f.block
	f.mu.Unlock()
	if b != nil {
		<-b
	}
}

func (f *fakeGit) record(c string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, c)
}

// called is the calls recorded that start with prefix.
func (f *fakeGit) called(prefix string) []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []string
	for _, c := range f.calls {
		if strings.HasPrefix(c, prefix) {
			out = append(out, c)
		}
	}
	return out
}

func (f *fakeGit) Worktrees(dir string) ([]input.Worktree, error) {
	f.wait()
	f.record("list " + dir)
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.listErr != nil {
		return nil, f.listErr
	}
	return slices.Clone(f.rows), nil
}

func (f *fakeGit) Add(main, path, branch string) error {
	f.wait()
	f.record("add " + main + " " + path + " " + branch)
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.addErr != nil {
		return f.addErr
	}
	f.rows = append(f.rows, input.Worktree{Path: path, Branch: branch})
	return nil
}

func (f *fakeGit) Diff(dir, base string) ([]string, error) {
	f.wait()
	f.record("diff " + dir + " " + base)
	return slices.Clone(f.diff), nil
}

func (f *fakeGit) Merge(main, branch string) error {
	f.wait()
	f.record("merge " + main + " " + branch)
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.mergeErr != nil {
		return f.mergeErr
	}
	for i := range f.rows {
		if f.rows[i].Branch == branch {
			f.rows[i].Ahead = 0
		}
	}
	return nil
}

func (f *fakeGit) Remove(main, path, branch string) (string, error) {
	f.wait()
	f.record("remove " + main + " " + path + " " + branch)
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.removeErr != nil {
		return "", f.removeErr
	}
	f.rows = slices.DeleteFunc(f.rows, func(r input.Worktree) bool { return r.Path == path })
	return f.removeNote, nil
}

func (f *fakeGit) Status(dir string) (input.GitMarks, error) {
	f.wait()
	f.record("status " + dir)
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.marks, f.statusErr
}

// repoRows is a main worktree at main and a linked one on feature, two
// commits ahead.
func repoRows(main string) []input.Worktree {
	return []input.Worktree{
		{Path: main, Branch: "main", Main: true},
		{Path: input.WorktreePath(main, "feature"), Branch: "feature", Ahead: 2},
	}
}

// wtStartup is the startup screen with a fake git on a repository at a
// temporary directory, which is also the current one; both worktree
// directories exist, so Enter can start in either.
func wtStartup(t *testing.T) (*harness, *fakeGit, *scriptCLI, string) {
	t.Helper()
	root := filepath.Join(t.TempDir(), "repo")
	fake := &fakeGit{rows: repoRows(root)}
	mkdirs(t, root, ".")
	mkdirs(t, fake.rows[1].Path, ".")
	cli := &scriptCLI{id: "claude-code", sess: newScriptSession()}
	h := newHarness(t, Options{Providers: []provider.Provider{cli}, Cwd: root, Git: fake, Sessions: &input.Sessions{Dir: t.TempDir()}})
	h.settle()
	h.m.ui.Choice = cli.id
	h.send(tea.WindowSizeMsg{Width: 120, Height: 40})
	return h, fake, cli, root
}

// wtSession is a dashboard session started in project, a worktree of the
// fake repository at root, with the view not yet open.
func wtSession(t *testing.T, root, project string, reg *input.Sessions) (*harness, *fakeGit, *scriptCLI) {
	t.Helper()
	fake := &fakeGit{rows: repoRows(root)}
	mkdirs(t, root, ".")
	mkdirs(t, fake.rows[1].Path, ".")
	cli := &scriptCLI{id: "claude-code", sess: newScriptSession()}
	h := newHarness(t, Options{Providers: []provider.Provider{cli}, Cwd: project, Git: fake, Sessions: reg})
	h.settle()
	h.m.ui.Choice = cli.id
	h.send(key(tea.KeyEnter, 0))
	if s := h.m.State(); s.Screen != state.ScreenDashboard || cli.starts() != 1 {
		t.Fatalf("screen %q, %d starts, phase %q, field %q: %s", s.Screen, cli.starts(), s.Startup.Phase, s.Startup.Field, s.Startup.Message)
	}
	h.send(tea.WindowSizeMsg{Width: 120, Height: 40})
	return h, fake, cli
}

func esc() tea.KeyPressMsg { return key(tea.KeyEscape, 0) }

// TestWorktreesStartup is criteria 2.1 and 2.2 of plan 16.
func TestWorktreesStartup(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, fake, cli, root := wtStartup(t)

		// The field empty: the current directory.
		h.m.path.SetText("")
		h.send(ctrl('w'))
		w := h.m.wt
		if w == nil || w.dir != root || w.reading || len(w.rows) != 2 || len(fake.called("list "+root)) != 1 {
			t.Fatalf("view %+v, calls %v", w, fake.called(""))
		}
		f := h.frame()
		for _, want := range []string{"Git worktrees", "main", "feature", "↑2", "clean", "2 worktrees", "Enter start here"} {
			if !strings.Contains(f, want) {
				t.Fatalf("frame lacks %q:\n%s", want, f)
			}
		}
		h.send(esc())
		if h.m.wt != nil || h.m.path.Text() != "" || h.m.State().Screen != state.ScreenStartup {
			t.Fatalf("Esc: view %v, field %q", h.m.wt, h.m.path.Text())
		}

		// The field's directory, and Esc leaves the field as it was.
		sub := filepath.Join(root, "internal")
		h.m.path.SetText(sub)
		h.send(ctrl('w'))
		if h.m.wt == nil || h.m.wt.dir != sub || len(fake.called("list "+sub)) != 1 {
			t.Fatalf("view on %v, calls %v", h.m.wt, fake.called(""))
		}
		h.send(esc())
		if h.m.wt != nil || h.m.path.Text() != sub {
			t.Fatalf("Esc: field %q", h.m.path.Text())
		}

		// Outside a repository, and no git.
		for _, c := range []struct {
			err  error
			want string
		}{{input.NotRepoError{Dir: `C:\elsewhere`}, `Not a git repository: C:\elsewhere`}, {input.ErrNoGit, "git not found: worktrees unavailable"}} {
			fake.mu.Lock()
			fake.listErr = c.err
			fake.mu.Unlock()
			h.send(ctrl('w'))
			if f := h.frame(); !strings.Contains(f, "✗ "+c.want) || len(h.m.wt.rows) != 0 || !strings.Contains(f, "Esc back · Ctrl+Q quit") {
				t.Fatalf("%v: frame\n%s", c.err, f)
			}
			h.send(typed('n'))
			if h.m.wt.mode != 0 {
				t.Fatal("n opened the form with no repository")
			}
			h.send(esc())
		}
		fake.mu.Lock()
		fake.listErr = nil
		fake.mu.Unlock()

		// 2.2: Enter on the linked row fills the field and starts there.
		h.m.path.SetText("")
		h.send(ctrl('w'))
		h.send(key(tea.KeyDown, 0))
		h.send(key(tea.KeyEnter, 0))
		linked := fake.rows[1].Path
		s := h.m.State()
		if h.m.wt != nil || h.m.path.Text() != linked || s.Screen != state.ScreenDashboard || cli.starts() != 1 || !h.m.samePath(s.Context.Project, linked) {
			t.Fatalf("Enter: view %v, field %q, screen %q, project %q, %d starts", h.m.wt, h.m.path.Text(), s.Screen, s.Context.Project, cli.starts())
		}
	})
}

// TestWorktreeCreate is criterion 2.3 of plan 16.
func TestWorktreeCreate(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, fake, _, root := wtStartup(t)
		h.send(ctrl('w'))

		// n shows the field and the path it creates.
		h.send(typed('n'))
		if h.m.wt.mode != 'n' || !strings.Contains(h.frame(), "New worktree") || !strings.Contains(h.frame(), "type an existing branch, or a new one from main") {
			t.Fatalf("n: mode %q\n%s", h.m.wt.mode, h.frame())
		}
		// Enter with no name is refused.
		h.send(key(tea.KeyEnter, 0))
		if !h.m.wt.failed || !strings.Contains(h.m.wt.note, "type the branch name") || len(fake.called("add")) != 0 {
			t.Fatalf("empty: note %q", h.m.wt.note)
		}
		for _, r := range "feat/login" {
			h.send(typed(r))
		}
		want := input.WorktreePath(root, "feat/login")
		if v := h.m.ui.Worktrees; v.Branch != "feat/login" || v.NewPath != want || !strings.HasSuffix(want, `repo.worktrees\login`) {
			t.Fatalf("form: branch %q, path %q", v.Branch, v.NewPath)
		}
		if f := h.frame(); !strings.Contains(f, "Branch  feat/login█") || !strings.Contains(f, `repo.worktrees\login`) {
			t.Fatalf("frame:\n%s", f)
		}

		// Enter creates it, lists again and selects the new row.
		lists := len(fake.called("list"))
		h.send(key(tea.KeyEnter, 0))
		w := h.m.wt
		if got := fake.called("add"); len(got) != 1 || got[0] != "add "+root+" "+want+" feat/login" {
			t.Fatalf("add calls %v", got)
		}
		if w.mode != 0 || w.failed || w.note != "created "+want || len(w.rows) != 3 || w.sel != 2 || len(fake.called("list")) != lists+1 {
			t.Fatalf("after create: mode %q, note %q, %d rows, row %d", w.mode, w.note, len(w.rows), w.sel)
		}
		if !strings.Contains(h.frame(), "✓ created") {
			t.Fatalf("frame:\n%s", h.frame())
		}

		// A refusal: the reason shows, the name stays, the list is unchanged.
		fake.mu.Lock()
		fake.addErr = errors.New("'bad..name' is not a valid branch name")
		fake.mu.Unlock()
		lists = len(fake.called("list"))
		h.send(typed('n'))
		for _, r := range "bad..name" {
			h.send(typed(r))
		}
		h.send(key(tea.KeyEnter, 0))
		w = h.m.wt
		if w.mode != 'n' || !w.failed || w.note != "'bad..name' is not a valid branch name" || w.branch.Text() != "bad..name" || len(w.rows) != 3 || len(fake.called("list")) != lists {
			t.Fatalf("refusal: mode %q, note %q, field %q, %d rows", w.mode, w.note, w.branch.Text(), len(w.rows))
		}
		if !strings.Contains(h.frame(), "✗ 'bad..name' is not a valid branch name") {
			t.Fatalf("frame:\n%s", h.frame())
		}

		// Esc cancels: no other call.
		adds := len(fake.called("add"))
		h.send(esc())
		if w.mode != 0 || len(fake.called("add")) != adds || h.m.wt == nil {
			t.Fatalf("Esc: mode %q, %d adds", w.mode, len(fake.called("add")))
		}
		// A paste takes its first line only.
		h.send(typed('n'))
		h.send(tea.PasteMsg{Content: "topic\r\nrm -rf"})
		if w.branch.Text() != "topic" {
			t.Fatalf("paste: %q", w.branch.Text())
		}
	})
}

// TestWorktreesSession is criterion 3.1 of plan 16.
func TestWorktreesSession(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		reg := &input.Sessions{Dir: t.TempDir()}
		root := filepath.Join(t.TempDir(), "repo")
		other := input.WorktreePath(root, "feature")
		// Another dashboard session, running in the linked worktree.
		lock, err := reg.Register(other)
		if err != nil {
			t.Fatal(err)
		}
		defer lock.Release()
		h, fake, _ := wtSession(t, root, root, reg)
		if h.m.lock == nil {
			t.Fatal("the session did not register")
		}

		h.send(ctrl('w'))
		w := h.m.wt
		if w == nil || w.dir != root || len(fake.called("list "+root)) != 1 {
			t.Fatalf("view %+v", w)
		}
		v := h.m.ui.Worktrees
		if len(v.Rows) != 2 || !v.Rows[0].Here || v.Rows[0].Other || v.Rows[1].Here || !v.Rows[1].Other || v.Sessions != "" {
			t.Fatalf("rows %+v, sessions %q", v.Rows, v.Sessions)
		}
		f := h.frame()
		for _, want := range []string{"this session", "session", "Esc back to the session", "2 worktrees"} {
			if !strings.Contains(f, want) {
				t.Fatalf("frame lacks %q:\n%s", want, f)
			}
		}
		// n is the startup screen's; the mouse is off.
		h.send(typed('n'))
		if w.mode != 0 || !strings.Contains(w.note, "from the startup screen") {
			t.Fatalf("n: mode %q, note %q", w.mode, w.note)
		}
		if h.m.View().MouseMode != tea.MouseModeNone {
			t.Fatal("the mouse is on while the view is open")
		}

		// An event meanwhile is in the state after Esc.
		pushEvents(h, "claude-code", evOf(event.SessionSpawned, event.Spawned{}), evOf(event.SessionReady, event.Ready{}))
		h.send(esc())
		if h.m.wt != nil || h.m.State().Screen != state.ScreenDashboard || h.m.State().Session.Phase != state.Ready {
			t.Fatalf("Esc: view %v, session %q", h.m.wt, h.m.State().Session.Phase)
		}
		if strings.Contains(h.frame(), "Git worktrees") {
			t.Fatal("the view is still drawn")
		}
	})
}

// TestWorktreeDiff is criterion 3.2 of plan 16.
func TestWorktreeDiff(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		root := filepath.Join(t.TempDir(), "repo")
		h, fake, _ := wtSession(t, root, root, nil)
		diff := []string{"diff --git a/x.go b/x.go", "index 1..2 100644", "--- a/x.go", "+++ b/x.go", "@@ -1,3 +1,3 @@"}
		for i := range 80 {
			diff = append(diff, fmt.Sprintf("+added %d", i), fmt.Sprintf("-removed %d", i))
		}
		diff = append(diff, "Untracked files:", "new.txt", "… 120 lines left out")
		fake.diff = diff
		h.send(ctrl('w'))
		h.send(key(tea.KeyDown, 0))
		h.send(typed('d'))
		w := h.m.wt
		linked := fake.rows[1].Path
		if got := fake.called("diff"); len(got) != 1 || got[0] != "diff "+linked+" main" {
			t.Fatalf("diff calls %v", got)
		}
		if w.mode != 'd' || len(w.diff) != len(diff) || w.diffOf != "feature against main" || w.busy != "" {
			t.Fatalf("diff: mode %q, %d lines, of %q", w.mode, len(w.diff), w.diffOf)
		}
		f := h.frame()
		if !strings.Contains(f, "Diff  feature against main") || !strings.Contains(f, "1–33 of 168 lines") {
			t.Fatalf("frame:\n%s", f)
		}
		// Coloured: an added line, a removed one and the hunk each in its style.
		raw := strings.Split(h.m.View().Content, "\n")
		style := func(text string) string {
			for _, l := range raw {
				if i := strings.Index(l, text); i > 0 && strings.Contains(plainText(l), text) {
					return l[:i]
				}
			}
			t.Fatalf("no line %q", text)
			return ""
		}
		if a, r, hunk := style("+added 0"), style("-removed 0"), style("@@ -1,3"); a == r || a == hunk || r == hunk {
			t.Fatalf("styles alike: %q %q %q", a, r, hunk)
		}

		// It scrolls, within its lines.
		h.send(key(tea.KeyDown, 0))
		if w.diffTop != 1 {
			t.Fatalf("down: top %d", w.diffTop)
		}
		h.send(key(tea.KeyPgDown, 0))
		if w.diffTop != 34 {
			t.Fatalf("pgdown: top %d", w.diffTop)
		}
		h.send(key(tea.KeyEnd, 0))
		if w.diffTop != len(diff)-33 || !strings.Contains(h.frame(), "… 120 lines left out") || !strings.Contains(h.frame(), "Untracked files:") {
			t.Fatalf("end: top %d\n%s", w.diffTop, h.frame())
		}
		h.send(key(tea.KeyDown, 0))
		if w.diffTop != len(diff)-33 {
			t.Fatalf("past the end: top %d", w.diffTop)
		}
		h.send(key(tea.KeyHome, 0))
		h.send(key(tea.KeyUp, 0))
		if w.diffTop != 0 {
			t.Fatalf("home: top %d", w.diffTop)
		}

		// Esc returns to the list; main's diff is its uncommitted changes.
		h.send(esc())
		if h.m.wt == nil || w.mode != 0 || w.diff != nil || !strings.Contains(h.frame(), "Git worktrees") {
			t.Fatalf("Esc: view %v, mode %q", h.m.wt, w.mode)
		}
		h.send(key(tea.KeyUp, 0))
		h.send(typed('d'))
		if got := fake.called("diff"); len(got) != 2 || got[1] != "diff "+root+" " || w.diffOf != "main: uncommitted changes" {
			t.Fatalf("main's diff: %v, of %q", got, w.diffOf)
		}
	})
}

// wtWith is a startup view whose rows are set, row sel selected.
func wtWith(t *testing.T, rows func(root string) []input.Worktree, sel int, reg func(root string) []string) (*harness, *fakeGit) {
	t.Helper()
	h, fake, _, root := wtStartup(t)
	fake.rows = rows(root)
	if reg != nil {
		for _, p := range reg(root) {
			l, err := h.m.opt.Sessions.Register(p)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { l.Release() })
		}
	}
	h.send(ctrl('w'))
	h.m.wt.sel = sel
	return h, fake
}

// TestWorktreeMerge is criterion 3.3 of plan 16.
func TestWorktreeMerge(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, fake := wtWith(t, repoRows, 1, nil)
		root := fake.rows[0].Path

		// m asks; n cancels.
		h.send(typed('m'))
		if h.m.wt.mode != 'm' || !strings.Contains(h.frame(), "? Merge feature into main?  y/n") || !strings.Contains(h.frame(), " y yes · n no") {
			t.Fatalf("m: mode %q\n%s", h.m.wt.mode, h.frame())
		}
		h.send(typed('n'))
		if h.m.wt.mode != 0 || len(fake.called("merge")) != 0 {
			t.Fatal("n merged")
		}
		// y merges, says so, lists again.
		lists := len(fake.called("list"))
		h.send(typed('m'))
		h.send(typed('y'))
		w := h.m.wt
		if got := fake.called("merge"); len(got) != 1 || got[0] != "merge "+root+" feature" {
			t.Fatalf("merge calls %v", got)
		}
		if w.failed || w.note != "merged feature into main" || len(fake.called("list")) != lists+1 || w.rows[1].Ahead != 0 {
			t.Fatalf("after merge: note %q, ahead %d", w.note, w.rows[1].Ahead)
		}
		if !strings.Contains(h.frame(), "✓ merged feature into main") {
			t.Fatalf("frame:\n%s", h.frame())
		}
		// A conflict.
		fake.mu.Lock()
		fake.rows[1].Ahead = 1
		fake.mergeErr = errors.New("conflict: merge aborted, main unchanged: CONFLICT (content): Merge conflict in x.go")
		fake.mu.Unlock()
		w.sel = 1
		w.rows[1].Ahead = 1
		h.send(typed('m'))
		h.send(typed('y'))
		if !w.failed || !strings.HasPrefix(w.note, "conflict: merge aborted") || !strings.Contains(h.frame(), "✗ conflict: merge aborted") {
			t.Fatalf("conflict: note %q", w.note)
		}

		// Each refusal of the design: its reason, no merge.
		dirty := func(i, n int) func(string) []input.Worktree {
			return func(root string) []input.Worktree { r := repoRows(root); r[i].Changes = n; return r }
		}
		for _, c := range []struct {
			name string
			rows func(string) []input.Worktree
			sel  int
			reg  func(string) []string
			want string
		}{
			{"main", repoRows, 0, nil, "this is main"},
			{"detached", func(root string) []input.Worktree { r := repoRows(root); r[1].Branch = ""; return r }, 1, nil, "detached HEAD: no branch to merge"},
			{"changes", dirty(1, 3), 1, nil, "feature has 3 uncommitted changes: commit them first"},
			{"main changes", dirty(0, 1), 1, nil, "main has 1 uncommitted change: commit them first"},
			{"nothing ahead", func(root string) []input.Worktree { r := repoRows(root); r[1].Ahead = 0; return r }, 1, nil, "nothing to merge: feature has no commit ahead of main"},
			{"ahead unknown", func(root string) []input.Worktree { r := repoRows(root); r[1].Ahead = -1; return r }, 1, nil, "cannot be counted"},
			{"main detached", func(root string) []input.Worktree { r := repoRows(root); r[0].Branch = ""; return r }, 1, nil, "main is detached: no branch to merge into"},
			{"missing", func(root string) []input.Worktree { r := repoRows(root); r[1].Missing = true; return r }, 1, nil, "directory is not there"},
			{"a session in main", repoRows, 1, func(root string) []string { return []string{root} }, "another dashboard session runs in main"},
		} {
			func() {
				h, fake := wtWith(t, c.rows, c.sel, c.reg)
				h.send(typed('m'))
				w := h.m.wt
				if w.mode != 0 || !w.failed || !strings.Contains(w.note, c.want) || !strings.Contains(h.frame(), "✗ ") || len(fake.called("merge")) != 0 {
					t.Fatalf("%s: mode %q, note %q, want %q", c.name, w.mode, w.note, c.want)
				}
			}()
		}

		// In session: in the worktree to merge, and in main during a turn.
		root2 := filepath.Join(t.TempDir(), "repo")
		h, fake, _ = wtSession(t, root2, input.WorktreePath(root2, "feature"), nil)
		h.send(ctrl('w'))
		h.m.wt.sel = 1
		h.send(typed('m'))
		if !strings.Contains(h.m.wt.note, "this session runs in feature") || len(fake.called("merge")) != 0 {
			t.Fatalf("here: note %q", h.m.wt.note)
		}
		root3 := filepath.Join(t.TempDir(), "repo")
		h, fake = readyIn(t, root3)
		h.m.State().Turn.Phase = state.Running // a turn: no longer idle
		if h.m.State().Turn.Phase == state.Idle {
			t.Fatalf("turn %q", h.m.State().Turn.Phase)
		}
		h.send(ctrl('w'))
		h.m.wt.sel = 1
		h.send(typed('m'))
		if !strings.Contains(h.m.wt.note, "this session's turn is running in main: wait for it to end") || len(fake.called("merge")) != 0 {
			t.Fatalf("turn: note %q", h.m.wt.note)
		}
	})
}

// readyIn is a ready session in the main worktree of a fake repository at
// root; its git is returned.
func readyIn(t *testing.T, root string) (*harness, *fakeGit) {
	t.Helper()
	h, fake, cli := wtSession(t, root, root, nil)
	pushEvents(h, cli.id, evOf(event.SessionSpawned, event.Spawned{}), evOf(event.SessionReady, event.Ready{}), evOf(event.SessionStarted, event.Started{}))
	if s := h.m.State(); s.Session.Phase != state.Ready {
		t.Fatalf("session %q", s.Session.Phase)
	}
	return h, fake
}

// TestWorktreeRemove is criterion 3.4 of plan 16.
func TestWorktreeRemove(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, fake := wtWith(t, repoRows, 1, nil)
		root := fake.rows[0].Path
		linked := fake.rows[1].Path
		fake.removeNote = "branch kept: error: the branch 'feature' is not fully merged"

		h.send(typed('x'))
		if h.m.wt.mode != 'x' || h.m.ui.Worktrees.Question != "Remove "+linked+"? Its branch feature is deleted only if merged." || !strings.Contains(h.frame(), "is deleted only if merged.  y/n") {
			t.Fatalf("x: mode %q\n%s", h.m.wt.mode, h.frame())
		}
		h.send(esc())
		if h.m.wt == nil || h.m.wt.mode != 0 || len(fake.called("remove")) != 0 {
			t.Fatal("Esc removed, or closed the view")
		}
		h.send(typed('x'))
		h.send(typed('y'))
		w := h.m.wt
		if got := fake.called("remove"); len(got) != 1 || got[0] != "remove "+root+" "+linked+" feature" {
			t.Fatalf("remove calls %v", got)
		}
		if w.failed || w.note != "removed "+linked+"; branch kept: error: the branch 'feature' is not fully merged" || len(w.rows) != 1 || w.sel != 0 {
			t.Fatalf("after remove: note %q, %d rows", w.note, len(w.rows))
		}

		// Each refusal of the design: its reason, no removal.
		for _, c := range []struct {
			name string
			rows func(string) []input.Worktree
			sel  int
			reg  func(string) []string
			want string
		}{
			{"main", repoRows, 0, nil, "the main worktree is the repository: it is never removed"},
			{"dirty", func(root string) []input.Worktree { r := repoRows(root); r[1].Changes = 2; return r }, 1, nil, "feature has 2 uncommitted changes: commit or discard them first"},
			{"unreadable", func(root string) []input.Worktree { r := repoRows(root); r[1].Problem = "fatal: bad object"; return r }, 1, nil, "its state cannot be read: fatal: bad object"},
			{"another session", repoRows, 1, func(root string) []string { return []string{input.WorktreePath(root, "feature")} }, "another dashboard session runs in this worktree"},
		} {
			func() {
				h, fake := wtWith(t, c.rows, c.sel, c.reg)
				h.send(typed('x'))
				w := h.m.wt
				if w.mode != 0 || !w.failed || !strings.Contains(w.note, c.want) || len(fake.called("remove")) != 0 {
					t.Fatalf("%s: mode %q, note %q, want %q", c.name, w.mode, w.note, c.want)
				}
			}()
		}
		// The session's own worktree.
		root2 := filepath.Join(t.TempDir(), "repo")
		h, fake, _ = wtSession(t, root2, input.WorktreePath(root2, "feature"), nil)
		h.send(ctrl('w'))
		h.m.wt.sel = 1
		h.send(typed('x'))
		if h.m.wt.note != "this session runs in this worktree" || len(fake.called("remove")) != 0 {
			t.Fatalf("here: note %q", h.m.wt.note)
		}
	})
}

// TestWorktreesSlow is criterion 3.5 of plan 16: a slow git shows what runs,
// keys stay handled, a stale result is dropped.
func TestWorktreesSlow(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, fake, _, _ := wtStartup(t)
		release := func() {
			fake.mu.Lock()
			close(fake.block)
			fake.block = nil
			fake.mu.Unlock()
			h.settle()
		}
		fake.block = make(chan struct{})
		h.send(ctrl('w'))
		if !h.m.wt.reading || !strings.Contains(h.frame(), "reading…") {
			t.Fatalf("reading: %+v\n%s", h.m.wt, h.frame())
		}
		h.send(key(tea.KeyDown, 0))
		h.send(typed('n'))
		if h.m.wt.mode != 0 || !strings.Contains(h.m.wt.note, "wait: the list is being read") {
			t.Fatalf("n while reading: mode %q, note %q", h.m.wt.mode, h.m.wt.note)
		}
		h.send(esc())
		if h.m.wt != nil {
			t.Fatal("Esc did not close the view while it read")
		}
		// The late list is dropped; a new view lists anew.
		h.send(ctrl('w'))
		release()
		if w := h.m.wt; w == nil || w.reading || len(w.rows) != 2 {
			t.Fatalf("after the list: %+v", w)
		}

		// An operation: it says what runs, Esc and the other keys wait.
		h.m.wt.sel = 1
		fake.mu.Lock()
		fake.block = make(chan struct{})
		fake.mu.Unlock()
		h.send(typed('m'))
		h.send(typed('y'))
		if !strings.Contains(h.frame(), "merging feature into main…") {
			t.Fatalf("frame:\n%s", h.frame())
		}
		for _, k := range []tea.KeyPressMsg{esc(), typed('d'), typed('x'), typed('n')} {
			h.send(k)
			if h.m.wt == nil || !strings.HasPrefix(h.m.wt.note, "wait: merging feature into main") {
				t.Fatalf("%v during the merge: view %v", k, h.m.wt)
			}
		}
		release()
		if w := h.m.wt; w.busy != "" || w.note != "merged feature into main" {
			t.Fatalf("after the merge: busy %q, note %q", w.busy, w.note)
		}

		// A diff read after its view is left is dropped.
		fake.mu.Lock()
		fake.block = make(chan struct{})
		fake.diff = []string{"+late"}
		fake.mu.Unlock()
		h.send(typed('d'))
		if !strings.Contains(h.frame(), "reading the diff…") {
			t.Fatalf("frame:\n%s", h.frame())
		}
		h.send(esc())
		release()
		if w := h.m.wt; w.mode != 0 || w.diff != nil {
			t.Fatalf("stale diff: mode %q, diff %v", w.mode, w.diff)
		}
	})
}

// TestSessionRegistered is criterion 3.6 of plan 16.
func TestSessionRegistered(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		reg := &input.Sessions{Dir: t.TempDir()}
		root := filepath.Join(t.TempDir(), "repo")
		h, _, _ := wtSession(t, root, root, reg)
		live, err := reg.Live()
		if err != nil || len(live) != 1 || !h.m.samePath(live[0].Project, root) || h.m.lock == nil || live[0].ID != h.m.lock.ID {
			t.Fatalf("live %v, %v", live, err)
		}
		h.m.Shutdown()
		if live, err := reg.Live(); err != nil || len(live) != 0 {
			t.Fatalf("after the end: live %v, %v", live, err)
		}
		if es, _ := os.ReadDir(reg.Dir); len(es) != 0 {
			t.Fatalf("files left: %v", es)
		}

		// A mock run registers nothing.
		reg = &input.Sessions{Dir: t.TempDir()}
		h = newHarness(t, Options{Config: Config{Mock: filepath.Join("..", "..", "testdata", "mock", "claude-subagent-approval.jsonl")}, Sessions: reg})
		h.settle()
		if es, _ := os.ReadDir(reg.Dir); h.m.State().Screen != state.ScreenDashboard || len(es) != 0 || h.m.lock != nil {
			t.Fatalf("mock: %v", es)
		}
		// No registry: nothing, and the view says so.
		root = filepath.Join(t.TempDir(), "repo")
		h, _, _ = wtSession(t, root, root, nil)
		h.send(ctrl('w'))
		if h.m.lock != nil || !strings.Contains(h.frame(), "other sessions: N/A: no session registry") {
			t.Fatalf("no registry:\n%s", h.frame())
		}
	})
}

// demoRepo is the golden frames' repository: fixed paths, as git would list
// them; the fake git ignores the directory asked about.
const demoRepo = `C:\javadev\projets\demo-project`

// demoWorktrees has one row of each kind the list draws: main, a branch
// ahead with another session in it, uncommitted changes, level with main,
// and a directory deleted by hand.
func demoWorktrees() []input.Worktree {
	return []input.Worktree{
		{Path: demoRepo, Branch: "main", Main: true},
		{Path: input.WorktreePath(demoRepo, "feature"), Branch: "feature", Ahead: 2},
		{Path: input.WorktreePath(demoRepo, "fix-login"), Branch: "fix-login", Changes: 3, Ahead: 1},
		{Path: input.WorktreePath(demoRepo, "docs"), Branch: "docs"},
		{Path: input.WorktreePath(demoRepo, "old-spike"), Branch: "old-spike", Missing: true, Ahead: -1},
	}
}

// demoDiff is feature's diff against main, as Diff returns it.
var demoDiff = []string{
	"diff --git a/internal/auth/login.go b/internal/auth/login.go",
	"index 3f1c2a0..9b8e7d4 100644",
	"--- a/internal/auth/login.go",
	"+++ b/internal/auth/login.go",
	"@@ -12,7 +12,9 @@ func Login(user, password string) (*Session, error) {",
	" 	if user == \"\" {",
	" 		return nil, ErrNoUser",
	" 	}",
	"-	s, err := store.Open(user)",
	"+	s, err := store.Open(user, store.WithTimeout(5*time.Second))",
	"+	if errors.Is(err, store.ErrLocked) {",
	"+		return nil, fmt.Errorf(\"%s is signed in elsewhere: %w\", user, err)",
	"+	}",
	" 	if err != nil {",
	" 		return nil, err",
	" 	}",
	"Untracked files:",
	"internal/auth/login_test.go",
}

// worktreesRig is the Worktrees view for the golden frames (plan 16, 2.4 and
// 3.7): over the startup screen, or over the mock dashboard, feature marked
// as another session's; keys are then sent to it.
func worktreesRig(startup bool, keys ...tea.KeyPressMsg) func(*testing.T) *harness {
	return func(t *testing.T) *harness {
		fake := &fakeGit{rows: demoWorktrees(), diff: demoDiff}
		reg := &input.Sessions{Dir: t.TempDir()}
		l, err := reg.Register(fake.rows[1].Path)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { l.Release() })
		opt := Options{Providers: bothCLIs[:1], Cwd: demoRepo, Git: fake, Sessions: reg}
		if !startup {
			opt = Options{Config: Config{Mock: filepath.Join("..", "..", "testdata", "mock", "claude-subagent-approval.jsonl")}, Git: fake, Sessions: reg}
		}
		h := newHarness(t, opt)
		h.settle()
		h.send(ctrl('w'))
		if h.m.wt == nil || len(h.m.wt.rows) != 5 || len(h.m.wt.live) != 1 {
			t.Fatalf("view %+v", h.m.wt)
		}
		for _, k := range keys {
			h.send(k)
		}
		return h
	}
}

// typedText is s typed key by key.
func typedText(s string) []tea.KeyPressMsg {
	var out []tea.KeyPressMsg
	for _, r := range s {
		out = append(out, typed(r))
	}
	return out
}

// TestKeysListWorktrees is criterion 3.7 of plan 16: F1's overlay lists
// Ctrl+W.
func TestKeysListWorktrees(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newHarness(t, Options{Config: Config{Mock: filepath.Join("..", "..", "testdata", "mock", "claude-subagent-approval.jsonl")}})
		h.settle()
		h.send(tea.WindowSizeMsg{Width: 120, Height: 40})
		h.send(key(tea.KeyF1, 0))
		if f := h.frame(); !strings.Contains(f, "Ctrl+W") || !strings.Contains(f, "worktrees: list, diff, merge, remove") {
			t.Fatalf("the overlay lacks Ctrl+W:\n%s", f)
		}
	})
}
