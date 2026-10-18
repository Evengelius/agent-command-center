package input

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// gitEnv makes git ignore the owner's and the system's configuration, so
// the tests run the same anywhere and never sign or hook a commit.
func gitEnv(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	empty := filepath.Join(t.TempDir(), "gitconfig")
	if err := os.WriteFile(empty, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GIT_CONFIG_GLOBAL", empty)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_AUTHOR_NAME", "test")
	t.Setenv("GIT_AUTHOR_EMAIL", "test@example.invalid")
	t.Setenv("GIT_COMMITTER_NAME", "test")
	t.Setenv("GIT_COMMITTER_EMAIL", "test@example.invalid")
}

// gitT runs git in dir and fails the test if it fails.
func gitT(t *testing.T, dir string, args ...string) string {
	t.Helper()
	out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return strings.TrimSpace(string(out))
}

func write(t *testing.T, path, text string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
}

// testRepo is a repository "shop" on branch main with one commit.
func testRepo(t *testing.T) string {
	t.Helper()
	gitEnv(t)
	main := filepath.Join(t.TempDir(), "shop")
	if err := os.Mkdir(main, 0o700); err != nil {
		t.Fatal(err)
	}
	gitT(t, main, "init", "-q", "-b", "main")
	write(t, filepath.Join(main, "a.txt"), "one\ntwo\n")
	gitT(t, main, "add", ".")
	gitT(t, main, "commit", "-q", "-m", "first")
	return main
}

// commit writes name in dir and commits it.
func commit(t *testing.T, dir, name, text string) {
	t.Helper()
	write(t, filepath.Join(dir, name), text)
	gitT(t, dir, "add", name)
	gitT(t, dir, "commit", "-q", "-m", name)
}

func samePath(a, b string) bool {
	a, b = filepath.Clean(a), filepath.Clean(b)
	if runtime.GOOS == "windows" {
		return strings.EqualFold(a, b)
	}
	return a == b
}

func exists(p string) bool { _, err := os.Lstat(p); return err == nil }

func TestWorktreePath(t *testing.T) {
	main := filepath.Join("C:", "x", "shop")
	for _, c := range []struct{ branch, want string }{
		{"feature/login", filepath.Join("C:", "x", "shop.worktrees", "login")},
		{"fix invoices", filepath.Join("C:", "x", "shop.worktrees", "fix-invoices")},
		{"a/b/c_d.e-f", filepath.Join("C:", "x", "shop.worktrees", "c_d.e-f")},
		{"feature/", ""},
		{"..", ""},
		{"é", ""},
	} {
		if got := WorktreePath(main, c.branch); got != c.want {
			t.Errorf("WorktreePath(%q) = %q, want %q", c.branch, got, c.want)
		}
	}
}

func TestGitWorktrees(t *testing.T) {
	main := testRepo(t)
	g := GitCLI{}
	login := filepath.Join(filepath.Dir(main), "shop.worktrees", "login")
	fix := filepath.Join(filepath.Dir(main), "shop.worktrees", "fix")
	loose := filepath.Join(filepath.Dir(main), "shop.worktrees", "loose")
	gitT(t, main, "worktree", "add", "-q", "-b", "feature/login", login)
	commit(t, login, "l1.txt", "1")
	commit(t, login, "l2.txt", "2")
	gitT(t, main, "worktree", "add", "-q", "-b", "fix", fix)
	write(t, filepath.Join(fix, "a.txt"), "changed\n")
	write(t, filepath.Join(fix, "new.txt"), "new\n")
	gitT(t, main, "worktree", "add", "-q", "--detach", loose, "HEAD")

	for _, from := range []string{main, login, filepath.Join(main)} {
		wts, err := g.Worktrees(from)
		if err != nil {
			t.Fatal(err)
		}
		if len(wts) != 4 {
			t.Fatalf("from %s: %d worktrees, want 4: %+v", from, len(wts), wts)
		}
		want := []struct {
			path, branch   string
			main           bool
			changes, ahead int
		}{
			{main, "main", true, 0, -1},
			{fix, "fix", false, 2, 0}, // git lists the linked ones by path
			{login, "feature/login", false, 0, 2},
			{loose, "", false, 0, -1},
		}
		for i, w := range want {
			got := wts[i]
			if !samePath(got.Path, w.path) || got.Branch != w.branch || got.Main != w.main || got.Changes != w.changes || got.Ahead != w.ahead || got.Problem != "" {
				t.Errorf("row %d = %+v, want %+v", i, got, w)
			}
			if runtime.GOOS == "windows" && strings.Contains(got.Path, "/") {
				t.Errorf("row %d path %q not in Windows form", i, got.Path)
			}
		}
	}

	outside := t.TempDir()
	_, err := g.Worktrees(outside)
	var nr NotRepoError
	if !errors.As(err, &nr) || !strings.Contains(err.Error(), "Not a git repository: "+outside) {
		t.Errorf("outside a repository: %v", err)
	}
}

func TestGitAdd(t *testing.T) {
	main := testRepo(t)
	g := GitCLI{}
	head := gitT(t, main, "rev-parse", "HEAD")

	p := WorktreePath(main, "feature/login")
	if err := g.Add(main, p, "feature/login"); err != nil {
		t.Fatal(err)
	}
	if got := gitT(t, p, "rev-parse", "HEAD"); got != head {
		t.Errorf("new branch at %s, want main's HEAD %s", got, head)
	}
	if got := gitT(t, p, "branch", "--show-current"); got != "feature/login" {
		t.Errorf("checked out %q", got)
	}

	gitT(t, main, "branch", "old")
	p = WorktreePath(main, "old")
	if err := g.Add(main, p, "old"); err != nil {
		t.Fatal(err)
	}
	if got := gitT(t, p, "branch", "--show-current"); got != "old" {
		t.Errorf("existing branch: checked out %q", got)
	}

	for _, c := range []struct{ branch, path, want string }{
		{"bad..name", WorktreePath(main, "bad..name"), "not a valid branch name"},
		{"feature/login", WorktreePath(main, "feature/login"), "already exists"},
		{"main", WorktreePath(main, "other/main"), "main"},
	} {
		before := exists(c.path)
		err := g.Add(main, c.path, c.branch)
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("Add(%q) = %v, want an error with %q", c.branch, err, c.want)
		}
		if !before && exists(c.path) {
			t.Errorf("Add(%q) created %s", c.branch, c.path)
		}
	}
	if n := len(lines(gitT(t, main, "worktree", "list"))); n != 3 {
		t.Errorf("%d worktrees after the refusals, want 3", n)
	}
}

func TestGitMerge(t *testing.T) {
	main := testRepo(t)
	g := GitCLI{}
	p := WorktreePath(main, "feature/login")
	gitT(t, main, "worktree", "add", "-q", "-b", "feature/login", p)
	commit(t, p, "login.txt", "login\n")
	if err := g.Merge(main, "feature/login"); err != nil {
		t.Fatal(err)
	}
	if n := gitT(t, main, "rev-list", "--count", "main..feature/login"); n != "0" {
		t.Errorf("after the merge feature/login is %s ahead", n)
	}

	q := WorktreePath(main, "clash")
	gitT(t, main, "worktree", "add", "-q", "-b", "clash", q)
	commit(t, q, "a.txt", "theirs\n")
	commit(t, main, "a.txt", "ours\n")
	head := gitT(t, main, "rev-parse", "HEAD")
	err := g.Merge(main, "clash")
	if err == nil || !strings.Contains(err.Error(), "conflict") {
		t.Fatalf("conflict: %v", err)
	}
	if got := gitT(t, main, "rev-parse", "HEAD"); got != head {
		t.Errorf("main moved to %s", got)
	}
	if out, err := exec.Command("git", "-C", main, "rev-parse", "-q", "--verify", "MERGE_HEAD").CombinedOutput(); err == nil {
		t.Errorf("MERGE_HEAD left: %s", out)
	}
	if st := gitT(t, main, "status", "--porcelain"); st != "" {
		t.Errorf("main not clean: %q", st)
	}
}

func TestGitRemove(t *testing.T) {
	main := testRepo(t)
	var calls [][]string
	g := GitCLI{Trace: func(a []string) { calls = append(calls, a) }}

	merged := WorktreePath(main, "merged")
	gitT(t, main, "worktree", "add", "-q", "-b", "merged", merged)
	note, err := g.Remove(main, merged, "merged")
	if err != nil || note != "branch merged deleted" || exists(merged) {
		t.Errorf("merged: note %q, err %v, still there %v", note, err, exists(merged))
	}
	if out, err := exec.Command("git", "-C", main, "show-ref", "--verify", "refs/heads/merged").CombinedOutput(); err == nil {
		t.Errorf("branch merged kept: %s", out)
	}

	open := WorktreePath(main, "open")
	gitT(t, main, "worktree", "add", "-q", "-b", "open", open)
	commit(t, open, "o.txt", "o\n")
	note, err = g.Remove(main, open, "open")
	if err != nil || !strings.HasPrefix(note, "branch kept: ") || exists(open) {
		t.Errorf("unmerged: note %q, err %v, still there %v", note, err, exists(open))
	}
	gitT(t, main, "show-ref", "--verify", "refs/heads/open")

	dirty := WorktreePath(main, "dirty")
	gitT(t, main, "worktree", "add", "-q", "-b", "dirty", dirty)
	write(t, filepath.Join(dirty, "wip.txt"), "wip\n")
	if _, err := g.Remove(main, dirty, "dirty"); err == nil {
		t.Error("a dirty worktree was removed")
	}
	if !exists(filepath.Join(dirty, "wip.txt")) {
		t.Error("the dirty worktree's file is gone")
	}
	gitT(t, main, "show-ref", "--verify", "refs/heads/dirty")

	if len(calls) == 0 {
		t.Fatal("no call traced")
	}
	for _, c := range calls {
		for _, a := range c {
			if a == "--force" || a == "-f" || a == "-D" {
				t.Errorf("forced call: %v", c)
			}
		}
	}
}

func TestGitDiff(t *testing.T) {
	main := testRepo(t)
	g := GitCLI{}
	p := WorktreePath(main, "work")
	gitT(t, main, "worktree", "add", "-q", "-b", "work", p)
	commit(t, p, "a.txt", "one\ncommitted\n")
	write(t, filepath.Join(p, "a.txt"), "one\ncommitted\nuncommitted\n")
	write(t, filepath.Join(p, "c.txt"), "untracked\n")
	commit(t, main, "m.txt", "on main after the branch\n")

	got, err := g.Diff(p, "main")
	if err != nil {
		t.Fatal(err)
	}
	text := strings.Join(got, "\n")
	for _, w := range []string{"-two", "+committed", "+uncommitted", "Untracked files:\n  c.txt"} {
		if !strings.Contains(text, w) {
			t.Errorf("diff lacks %q:\n%s", w, text)
		}
	}
	if strings.Contains(text, "m.txt") {
		t.Errorf("diff holds main's own commit:\n%s", text)
	}

	write(t, filepath.Join(main, "a.txt"), "one\ntwo\nthree\n")
	got, err = g.Diff(main, "")
	if err != nil {
		t.Fatal(err)
	}
	if text := strings.Join(got, "\n"); !strings.Contains(text, "+three") || strings.Contains(text, "committed") {
		t.Errorf("main's uncommitted diff:\n%s", text)
	}

	defer func(n int) { DiffLimit = n }(DiffLimit)
	DiffLimit = 3
	got, err = g.Diff(p, "main")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 4 || !strings.HasPrefix(got[3], "… ") || !strings.HasSuffix(got[3], " lines left out") {
		t.Errorf("capped diff: %q", got)
	}
}

func TestGitMissing(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	g := GitCLI{}
	dir := t.TempDir()
	_, e1 := g.Worktrees(dir)
	e2 := g.Add(dir, filepath.Join(dir, "x"), "x")
	_, e3 := g.Diff(dir, "")
	e4 := g.Merge(dir, "x")
	_, e5 := g.Remove(dir, filepath.Join(dir, "x"), "x")
	for i, err := range []error{e1, e2, e3, e4, e5} {
		if !errors.Is(err, ErrNoGit) || !strings.Contains(err.Error(), "git not found") {
			t.Errorf("call %d: %v", i+1, err)
		}
	}
}

func TestSessionRegistry(t *testing.T) {
	s := Sessions{Dir: t.TempDir()}
	if runtime.GOOS != "windows" {
		if _, err := s.Live(); err != nil && !errors.Is(err, ErrSessionsUnsupported) {
			t.Fatal(err)
		}
		t.Skip("an open file is told from a closed one on Windows only")
	}
	l, err := s.Register(`C:\work\shop`)
	if err != nil {
		t.Fatal(err)
	}
	live, err := s.Live()
	if err != nil || len(live) != 1 || live[0].Project != `C:\work\shop` || live[0].ID != l.ID {
		t.Fatalf("live = %+v, %v", live, err)
	}

	stale := filepath.Join(s.Dir, "session-stale.lock")
	write(t, stale, `C:\gone`)
	live, err = s.Live()
	if err != nil || len(live) != 1 || exists(stale) {
		t.Errorf("a stale file: live %+v, %v, still there %v", live, err, exists(stale))
	}

	if err := l.Release(); err != nil {
		t.Fatal(err)
	}
	if live, err := s.Live(); err != nil || len(live) != 0 {
		t.Errorf("after Release: %+v, %v", live, err)
	}
	if es, _ := os.ReadDir(s.Dir); len(es) != 0 {
		t.Errorf("files left: %v", es)
	}
}
