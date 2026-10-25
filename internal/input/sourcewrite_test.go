package input

import (
	"errors"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// Plan 45, criteria 45.5 to 45.8: the Source Control's staging, commit and
// push, on temporary repositories whose remotes are local bare repositories.
// No test touches the repository these files are in.

// traceLog records the arguments of every git call of a GitCLI.
type traceLog struct {
	mu    sync.Mutex
	calls [][]string
}

func (l *traceLog) add(args []string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.calls = append(l.calls, append([]string(nil), args...))
}

// ran reports whether any call has tok as one of its arguments.
func (l *traceLog) ran(tok string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, c := range l.calls {
		for _, a := range c {
			if a == tok {
				return true
			}
		}
	}
	return false
}

// forbiddenArg is the argument of a call that no Source Control call may
// hold, "" when there is none (plan 45, criteria 45.6 and 45.7).
func forbiddenArg(args []string) string {
	for _, a := range args {
		switch a {
		case "-f", "--all", "--tags", "--mirror", "--delete", "--allow-empty", "--amend", "--no-verify":
			return a
		}
		if strings.HasPrefix(a, "--force") || strings.HasPrefix(a, "+") || strings.Contains(a, ":+") {
			return a
		}
	}
	return ""
}

// tracedGit is a GitCLI whose every call the test checks at its end for the
// forbidden flags.
func tracedGit(t *testing.T) (GitCLI, *traceLog) {
	t.Helper()
	l := &traceLog{}
	t.Cleanup(func() {
		l.mu.Lock()
		defer l.mu.Unlock()
		if len(l.calls) == 0 {
			return
		}
		for _, c := range l.calls {
			if bad := forbiddenArg(c); bad != "" {
				t.Errorf("a call holds %q: git %s", bad, strings.Join(c, " "))
			}
		}
	})
	return GitCLI{Trace: l.add}, l
}

// gitTry runs git in dir and returns its output and error; a failure is no
// failure of the test.
func gitTry(dir string, args ...string) (string, error) {
	out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

// gitRaw is git's standard output untouched.
func gitRaw(t *testing.T, dir string, args ...string) string {
	t.Helper()
	out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).Output()
	if err != nil {
		t.Fatalf("git %s: %v", strings.Join(args, " "), err)
	}
	return string(out)
}

// hook writes an executable hook script of the repository whose hooks folder
// is hooks.
func hook(t *testing.T, hooks, name, body string) {
	t.Helper()
	if err := os.MkdirAll(hooks, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(hooks, name), []byte("#!/bin/sh\n"+body+"\n"), 0o700); err != nil {
		t.Fatal(err)
	}
}

// bareRemote makes a bare repository, adds it to repo as name and pushes
// main to it with the upstream set. It returns the bare repository's path.
func bareRemote(t *testing.T, repo, name string) string {
	t.Helper()
	bare := filepath.Join(t.TempDir(), name+".git")
	gitT(t, repo, "init", "--bare", "-q", "-b", "main", bare)
	gitT(t, repo, "remote", "add", name, bare)
	gitT(t, repo, "push", "-q", "-u", name, "main")
	return bare
}

// stagedOf is the staged entries of the project dir as the panel would list
// them.
func stagedOf(t *testing.T, g GitCLI, dir string) []GitEntry {
	t.Helper()
	snap, err := g.Snapshot(dir)
	if err != nil {
		t.Fatal(err)
	}
	var out []GitEntry
	for _, e := range snap.Entries {
		if e.X != ' ' && e.X != '?' && e.X != '!' {
			out = append(out, e)
		}
	}
	return out
}

// refusal is err as a Refusal of the given kind with the given message.
func wantRefusal(t *testing.T, err error, kind, msg string) {
	t.Helper()
	var r Refusal
	if !errors.As(err, &r) {
		t.Fatalf("error %v (%T) is no Refusal, want %s", err, err, kind)
	}
	if r.Kind != kind {
		t.Errorf("refusal kind %q, want %q (%s)", r.Kind, kind, r.Msg)
	}
	if msg != "" && r.Msg != msg || r.Error() != r.Msg {
		t.Errorf("refusal message %q, want %q", r.Msg, msg)
	}
}

// divergent is a repository on main whose main and feature both changed a.txt
// in the same line.
func divergent(t *testing.T) string {
	t.Helper()
	repo := testRepo(t)
	gitT(t, repo, "checkout", "-q", "-b", "feature")
	commit(t, repo, "a.txt", "feature\ntwo\n")
	gitT(t, repo, "checkout", "-q", "main")
	commit(t, repo, "a.txt", "main\ntwo\n")
	return repo
}

func TestScrubURLs(t *testing.T) {
	for _, c := range []struct{ in, want string }{
		{"https://user:secret@host/x.git", "https://host/x.git"},
		{"fatal: unable to access 'https://user:p@ss@example.com:8443/x.git/': Could not resolve", "fatal: unable to access 'https://example.com:8443/x.git/': Could not resolve"},
		{"ssh://git@host/x", "ssh://host/x"},
		{"https://host/x@y", "https://host/x@y"},
		{"no url, a@b", "no url, a@b"},
	} {
		if got := scrubURLs(c.in); got != c.want {
			t.Errorf("scrubURLs(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestSourceStage(t *testing.T) {
	cached := func(t *testing.T, repo string) string {
		return gitT(t, repo, "diff", "--cached", "--name-status")
	}

	t.Run("a modified, a deleted and an untracked file", func(t *testing.T) {
		repo := testRepo(t)
		commit(t, repo, "b.txt", "b\n")
		g, _ := tracedGit(t)
		write(t, filepath.Join(repo, "a.txt"), "changed\n")
		if err := os.Remove(filepath.Join(repo, "b.txt")); err != nil {
			t.Fatal(err)
		}
		write(t, filepath.Join(repo, "c.txt"), "new\n")
		if err := g.Stage(repo, []string{"a.txt", "b.txt", "c.txt"}); err != nil {
			t.Fatal(err)
		}
		if got, want := cached(t, repo), "M\ta.txt\nD\tb.txt\nA\tc.txt"; got != want {
			t.Errorf("staged:\n%s\nwant:\n%s", got, want)
		}
	})

	t.Run("an untracked folder adds its files, never an ignored one", func(t *testing.T) {
		repo := testRepo(t)
		g, _ := tracedGit(t)
		write(t, filepath.Join(repo, ".gitignore"), "*.log\n")
		gitT(t, repo, "add", ".gitignore")
		gitT(t, repo, "commit", "-q", "-m", "ignore")
		if err := os.MkdirAll(filepath.Join(repo, "dir", "deep"), 0o700); err != nil {
			t.Fatal(err)
		}
		write(t, filepath.Join(repo, "dir", "x.txt"), "x\n")
		write(t, filepath.Join(repo, "dir", "deep", "z.txt"), "z\n")
		write(t, filepath.Join(repo, "dir", "y.log"), "y\n")
		if err := g.Stage(repo, []string{"dir/"}); err != nil {
			t.Fatal(err)
		}
		if got, want := cached(t, repo), "A\tdir/deep/z.txt\nA\tdir/x.txt"; got != want {
			t.Errorf("staged:\n%s\nwant:\n%s", got, want)
		}
	})

	t.Run("a space and a bracket are taken literally", func(t *testing.T) {
		repo := testRepo(t)
		g, _ := tracedGit(t)
		for _, n := range []string{"a b.txt", "f[1].txt", "f1.txt", "-dash.txt"} {
			write(t, filepath.Join(repo, n), n+"\n")
		}
		if err := g.Stage(repo, []string{"a b.txt", "f[1].txt", "-dash.txt"}); err != nil {
			t.Fatal(err)
		}
		if got, want := cached(t, repo), "A\t-dash.txt\nA\ta b.txt\nA\tf[1].txt"; got != want {
			t.Errorf("staged:\n%s\nwant:\n%s (f1.txt must stay unstaged)", got, want)
		}
	})

	t.Run("exactly the listed paths", func(t *testing.T) {
		repo := testRepo(t)
		commit(t, repo, "b.txt", "b\n")
		g, _ := tracedGit(t)
		write(t, filepath.Join(repo, "a.txt"), "changed\n")
		write(t, filepath.Join(repo, "b.txt"), "changed\n")
		if err := g.Stage(repo, []string{"a.txt"}); err != nil {
			t.Fatal(err)
		}
		if got := cached(t, repo); got != "M\ta.txt" {
			t.Errorf("staged: %q, want only a.txt", got)
		}
	})

	t.Run("a project folder, paths relative to it", func(t *testing.T) {
		repo := testRepo(t)
		g, _ := tracedGit(t)
		if err := os.Mkdir(filepath.Join(repo, "sub"), 0o700); err != nil {
			t.Fatal(err)
		}
		write(t, filepath.Join(repo, "sub", "n.txt"), "n\n")
		write(t, filepath.Join(repo, "n.txt"), "outer\n")
		if err := g.Stage(filepath.Join(repo, "sub"), []string{"n.txt"}); err != nil {
			t.Fatal(err)
		}
		if got := cached(t, repo); got != "A\tsub/n.txt" {
			t.Errorf("staged: %q, want sub/n.txt only", got)
		}
	})

	t.Run("no path runs no git", func(t *testing.T) {
		repo := testRepo(t)
		g, log := tracedGit(t)
		if err := g.Stage(repo, nil); err != nil {
			t.Fatal(err)
		}
		if len(log.calls) != 0 {
			t.Errorf("git ran: %v", log.calls)
		}
	})

	t.Run("git's refusal is the error", func(t *testing.T) {
		repo := testRepo(t)
		g, _ := tracedGit(t)
		if err := g.Stage(repo, []string{"missing.txt"}); err == nil {
			t.Error("staging a path that does not exist succeeded")
		}
	})
}

func TestSourceCommit(t *testing.T) {
	t.Run("the index alone: the unstaged part of a file stays", func(t *testing.T) {
		repo := testRepo(t)
		g, _ := tracedGit(t)
		old := gitT(t, repo, "rev-parse", "HEAD")
		write(t, filepath.Join(repo, "a.txt"), "ONE\ntwo\n")
		gitT(t, repo, "add", "a.txt")
		write(t, filepath.Join(repo, "a.txt"), "ONE\ntwo\nthree\n")
		res, err := g.Commit(repo, "second", stagedOf(t, g, repo))
		if err != nil {
			t.Fatal(err)
		}
		head := gitT(t, repo, "rev-parse", "HEAD")
		if res.Hash != head || head == old || !strings.HasPrefix(head, res.Short) || len(res.Short) < 7 {
			t.Errorf("result %+v, HEAD %s, old %s", res, head, old)
		}
		if got := gitRaw(t, repo, "show", "HEAD:a.txt"); got != "ONE\ntwo\n" {
			t.Errorf("HEAD:a.txt = %q, want the staged content", got)
		}
		if d := gitT(t, repo, "diff"); !strings.Contains(d, "+three") {
			t.Errorf("git diff lost the unstaged part:\n%s", d)
		}
		if p := gitT(t, repo, "rev-parse", "HEAD^1"); p != old {
			t.Errorf("parent %s, want the old HEAD %s", p, old)
		}
		if n := gitT(t, repo, "rev-list", "--count", "HEAD"); n != "2" {
			t.Errorf("%s commits, want 2", n)
		}
	})

	t.Run("the message as git cleans it", func(t *testing.T) {
		repo := testRepo(t)
		g, _ := tracedGit(t)
		write(t, filepath.Join(repo, "a.txt"), "new\n")
		gitT(t, repo, "add", "a.txt")
		if _, err := g.Commit(repo, "subject  \n\nbody  \n\n# not a comment\n\n  \n", stagedOf(t, g, repo)); err != nil {
			t.Fatal(err)
		}
		raw := gitRaw(t, repo, "cat-file", "commit", "HEAD")
		_, msg, _ := strings.Cut(raw, "\n\n")
		if want := "subject\n\nbody\n\n# not a comment\n"; msg != want {
			t.Errorf("stored message %q, want %q", msg, want)
		}
	})

	t.Run("a multibyte message is kept byte for byte", func(t *testing.T) {
		repo := testRepo(t)
		g, _ := tracedGit(t)
		write(t, filepath.Join(repo, "a.txt"), "new\n")
		gitT(t, repo, "add", "a.txt")
		if _, err := g.Commit(repo, "Café ✓ 日本\r\nsecond line", stagedOf(t, g, repo)); err != nil {
			t.Fatal(err)
		}
		raw := gitRaw(t, repo, "cat-file", "commit", "HEAD")
		if _, msg, _ := strings.Cut(raw, "\n\n"); !strings.HasPrefix(msg, "Café ✓ 日本") {
			t.Errorf("stored message %q", msg)
		}
	})

	t.Run("a root commit in an empty repository", func(t *testing.T) {
		gitEnv(t)
		repo := t.TempDir()
		gitT(t, repo, "init", "-q", "-b", "main")
		write(t, filepath.Join(repo, "f.txt"), "f\n")
		gitT(t, repo, "add", "f.txt")
		g, _ := tracedGit(t)
		res, err := g.Commit(repo, "root", stagedOf(t, g, repo))
		if err != nil {
			t.Fatal(err)
		}
		if head := gitT(t, repo, "rev-parse", "HEAD"); head != res.Hash {
			t.Errorf("HEAD %s, result %+v", head, res)
		}
		if p := gitT(t, repo, "cat-file", "-p", "HEAD"); strings.Contains(p, "\nparent ") {
			t.Errorf("a root commit has a parent:\n%s", p)
		}
	})

	t.Run("a rename found once both sides are staged", func(t *testing.T) {
		repo := testRepo(t)
		g, _ := tracedGit(t)
		body := strings.Repeat("a line of the file\n", 20)
		commit(t, repo, "old.txt", body)
		if err := os.Rename(filepath.Join(repo, "old.txt"), filepath.Join(repo, "new.txt")); err != nil {
			t.Fatal(err)
		}
		// What the panel showed: the deletion and the untracked file.
		shown := []GitEntry{{X: ' ', Y: 'D', Path: "old.txt"}, {X: '?', Y: '?', Path: "new.txt"}}
		if err := g.Stage(repo, []string{"old.txt", "new.txt"}); err != nil {
			t.Fatal(err)
		}
		if st := gitT(t, repo, "status", "--porcelain"); !strings.HasPrefix(st, "R  old.txt -> new.txt") {
			t.Fatalf("git does not see a rename: %q", st)
		}
		if _, err := g.Commit(repo, "rename", shown); err != nil {
			t.Fatal(err)
		}
		if gitT(t, repo, "status", "--porcelain") != "" {
			t.Error("changes left after the commit")
		}
	})

	t.Run("an untracked folder staged matches its folder", func(t *testing.T) {
		repo := testRepo(t)
		g, _ := tracedGit(t)
		if err := os.Mkdir(filepath.Join(repo, "dir"), 0o700); err != nil {
			t.Fatal(err)
		}
		write(t, filepath.Join(repo, "dir", "1.txt"), "1\n")
		write(t, filepath.Join(repo, "dir", "2.txt"), "2\n")
		shown := []GitEntry{{X: '?', Y: '?', Path: "dir/"}}
		if err := g.Stage(repo, []string{"dir/"}); err != nil {
			t.Fatal(err)
		}
		if _, err := g.Commit(repo, "dir", shown); err != nil {
			t.Fatal(err)
		}
		if files := gitT(t, repo, "ls-tree", "-r", "--name-only", "HEAD"); files != "a.txt\ndir/1.txt\ndir/2.txt" {
			t.Errorf("HEAD holds %q", files)
		}
	})

	t.Run("a merge with its conflicts resolved has two parents", func(t *testing.T) {
		repo := divergent(t)
		g, _ := tracedGit(t)
		if _, err := gitTry(repo, "merge", "feature"); err == nil {
			t.Fatal("the merge did not conflict")
		}
		write(t, filepath.Join(repo, "a.txt"), "both\ntwo\n")
		gitT(t, repo, "add", "a.txt")
		if _, err := g.Commit(repo, "Merge feature", stagedOf(t, g, repo)); err != nil {
			t.Fatal(err)
		}
		if p := gitT(t, repo, "log", "-1", "--format=%p"); len(strings.Fields(p)) != 2 {
			t.Errorf("parents %q, want two", p)
		}
		if _, err := os.Stat(filepath.Join(repo, ".git", "MERGE_HEAD")); err == nil {
			t.Error("the merge is still in progress")
		}
	})

	t.Run("nothing staged", func(t *testing.T) {
		repo := testRepo(t)
		g, log := tracedGit(t)
		old := gitT(t, repo, "rev-parse", "HEAD")
		write(t, filepath.Join(repo, "a.txt"), "unstaged\n")
		_, err := g.Commit(repo, "x", nil)
		wantRefusal(t, err, "nothing-staged", "No changes to commit.")
		if gitT(t, repo, "rev-parse", "HEAD") != old {
			t.Error("HEAD moved")
		}
		if log.ran("commit") {
			t.Error("git commit ran")
		}
	})

	t.Run("a pre-commit hook that fails", func(t *testing.T) {
		repo := testRepo(t)
		g, _ := tracedGit(t)
		write(t, filepath.Join(repo, "a.txt"), "staged\n")
		gitT(t, repo, "add", "a.txt")
		hook(t, filepath.Join(repo, ".git", "hooks"), "pre-commit", "echo 'lint says no'\nexit 1")
		old, cachedBefore := gitT(t, repo, "rev-parse", "HEAD"), gitT(t, repo, "diff", "--cached")
		_, err := g.Commit(repo, "msg", stagedOf(t, g, repo))
		if err == nil || !strings.Contains(err.Error(), "lint says no") {
			t.Fatalf("error %v, want the hook's output", err)
		}
		var r Refusal
		if errors.As(err, &r) {
			t.Errorf("a hook's failure is no refusal: %+v", r)
		}
		if gitT(t, repo, "rev-parse", "HEAD") != old || gitT(t, repo, "diff", "--cached") != cachedBefore || cachedBefore == "" {
			t.Error("HEAD or the index changed")
		}
	})

	t.Run("a commit-msg hook that rejects", func(t *testing.T) {
		repo := testRepo(t)
		g, _ := tracedGit(t)
		write(t, filepath.Join(repo, "a.txt"), "staged\n")
		gitT(t, repo, "add", "a.txt")
		hook(t, filepath.Join(repo, ".git", "hooks"), "commit-msg", "echo 'subject too vague' >&2\nexit 1")
		old, cachedBefore := gitT(t, repo, "rev-parse", "HEAD"), gitT(t, repo, "diff", "--cached")
		_, err := g.Commit(repo, "msg", stagedOf(t, g, repo))
		if err == nil || !strings.Contains(err.Error(), "subject too vague") {
			t.Fatalf("error %v, want the hook's output", err)
		}
		if gitT(t, repo, "rev-parse", "HEAD") != old || gitT(t, repo, "diff", "--cached") != cachedBefore {
			t.Error("HEAD or the index changed")
		}
	})

	t.Run("no identity", func(t *testing.T) {
		repo := testRepo(t)
		g, _ := tracedGit(t)
		write(t, filepath.Join(repo, "a.txt"), "staged\n")
		gitT(t, repo, "add", "a.txt")
		for _, v := range []string{"GIT_AUTHOR_NAME", "GIT_AUTHOR_EMAIL", "GIT_COMMITTER_NAME", "GIT_COMMITTER_EMAIL", "EMAIL"} {
			os.Unsetenv(v)
		}
		old := gitT(t, repo, "rev-parse", "HEAD")
		_, err := g.Commit(repo, "msg", stagedOf(t, g, repo))
		wantRefusal(t, err, "identity", "Git has no user name and email for this repository: set user.name and user.email in a terminal, then commit.")
		if gitT(t, repo, "rev-parse", "HEAD") != old {
			t.Error("a commit was made")
		}
	})

	t.Run("a merge conflict", func(t *testing.T) {
		repo := divergent(t)
		g, _ := tracedGit(t)
		if _, err := gitTry(repo, "merge", "feature"); err == nil {
			t.Fatal("the merge did not conflict")
		}
		_, err := g.Commit(repo, "msg", nil)
		wantRefusal(t, err, "conflict", "Resolve the merge conflicts first.")
	})

	t.Run("staged changes outside the project folder", func(t *testing.T) {
		repo := testRepo(t)
		g, _ := tracedGit(t)
		sub := filepath.Join(repo, "sub")
		if err := os.Mkdir(sub, 0o700); err != nil {
			t.Fatal(err)
		}
		write(t, filepath.Join(sub, "in.txt"), "in\n")
		write(t, filepath.Join(repo, "out1.txt"), "1\n")
		gitT(t, repo, "add", "sub/in.txt", "out1.txt")
		old := gitT(t, repo, "rev-parse", "HEAD")
		_, err := g.Commit(sub, "msg", []GitEntry{{X: 'A', Y: ' ', Path: "in.txt"}})
		wantRefusal(t, err, "outside", "1 staged change outside this folder would be committed too: commit them from a terminal, or unstage them.")
		write(t, filepath.Join(repo, "out2.txt"), "2\n")
		write(t, filepath.Join(repo, "out3.txt"), "3\n")
		gitT(t, repo, "add", "out2.txt", "out3.txt")
		_, err = g.Commit(sub, "msg", []GitEntry{{X: 'A', Y: ' ', Path: "in.txt"}})
		wantRefusal(t, err, "outside", "3 staged changes outside this folder would be committed too: commit them from a terminal, or unstage them.")
		if gitT(t, repo, "rev-parse", "HEAD") != old {
			t.Error("a commit was made")
		}
	})

	t.Run("a project folder commits its own changes", func(t *testing.T) {
		repo := testRepo(t)
		g, _ := tracedGit(t)
		sub := filepath.Join(repo, "sub")
		if err := os.Mkdir(sub, 0o700); err != nil {
			t.Fatal(err)
		}
		write(t, filepath.Join(sub, "in.txt"), "in\n")
		gitT(t, repo, "add", "sub/in.txt")
		if _, err := g.Commit(sub, "msg", stagedOf(t, g, sub)); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("the staged set is not the expected one", func(t *testing.T) {
		repo := testRepo(t)
		commit(t, repo, "b.txt", "b\n")
		g, _ := tracedGit(t)
		write(t, filepath.Join(repo, "a.txt"), "a2\n")
		write(t, filepath.Join(repo, "b.txt"), "b2\n")
		gitT(t, repo, "add", "a.txt", "b.txt")
		old := gitT(t, repo, "rev-parse", "HEAD")
		const msg = "The staged changes changed since they were shown: check them, then commit again."
		for name, expect := range map[string][]GitEntry{
			"a path missing":  {{X: 'M', Path: "a.txt"}},
			"an extra path":   {{X: 'M', Path: "a.txt"}, {X: 'M', Path: "b.txt"}, {X: 'M', Path: "c.txt"}},
			"nothing":         nil,
			"another path":    {{X: 'M', Path: "a.txt"}, {X: 'M', Path: "c.txt"}},
			"a folder unused": {{X: 'M', Path: "a.txt"}, {X: 'M', Path: "b.txt"}, {X: '?', Y: '?', Path: "dir/"}},
		} {
			_, err := g.Commit(repo, "msg", expect)
			wantRefusal(t, err, "changed", msg)
			if err != nil && gitT(t, repo, "rev-parse", "HEAD") != old {
				t.Errorf("%s: a commit was made", name)
			}
		}
	})

	t.Run("a rebase in progress", func(t *testing.T) {
		repo := divergent(t)
		g, _ := tracedGit(t)
		gitT(t, repo, "checkout", "-q", "feature")
		if _, err := gitTry(repo, "rebase", "main"); err == nil {
			t.Fatal("the rebase did not conflict")
		}
		_, err := g.Commit(repo, "msg", nil)
		wantRefusal(t, err, "rebase", "A rebase is in progress: finish it in a terminal first.")
	})

	t.Run("a cherry-pick in progress", func(t *testing.T) {
		repo := divergent(t)
		g, _ := tracedGit(t)
		if _, err := gitTry(repo, "cherry-pick", "feature"); err == nil {
			t.Fatal("the cherry-pick did not conflict")
		}
		_, err := g.Commit(repo, "msg", nil)
		wantRefusal(t, err, "cherry-pick", "A cherry-pick is in progress: finish it in a terminal first.")
	})

	t.Run("a revert in progress", func(t *testing.T) {
		repo := testRepo(t)
		g, _ := tracedGit(t)
		commit(t, repo, "a.txt", "x\ntwo\n")
		first := gitT(t, repo, "rev-parse", "HEAD")
		commit(t, repo, "a.txt", "y\ntwo\n")
		if _, err := gitTry(repo, "revert", "--no-edit", first); err == nil {
			t.Fatal("the revert did not conflict")
		}
		_, err := g.Commit(repo, "msg", nil)
		wantRefusal(t, err, "revert", "A revert is in progress: finish it in a terminal first.")
	})

	t.Run("an am session in progress", func(t *testing.T) {
		repo := divergent(t)
		g, _ := tracedGit(t)
		patch := filepath.Join(t.TempDir(), "p.patch")
		write(t, patch, gitRaw(t, repo, "format-patch", "-1", "feature", "--stdout"))
		if _, err := gitTry(repo, "am", patch); err == nil {
			t.Fatal("the patch applied")
		}
		_, err := g.Commit(repo, "msg", nil)
		wantRefusal(t, err, "am", "An am session is in progress: finish it in a terminal first.")
	})
}

func TestSourcePushSucceeds(t *testing.T) {
	t.Run("the remote's branch is the local HEAD", func(t *testing.T) {
		repo := testRepo(t)
		bare := bareRemote(t, repo, "origin")
		g, _ := tracedGit(t)
		commit(t, repo, "b.txt", "b\n")
		res, err := g.Push(repo, nil)
		if err != nil {
			t.Fatal(err)
		}
		if want := (PushTarget{Branch: "main", Remote: "origin", RemoteRef: "refs/heads/main", Upstream: "origin/main", Ahead: 1, tracking: "refs/remotes/origin/main"}); res.Target != want {
			t.Errorf("target %+v, want %+v", res.Target, want)
		}
		if remote, local := gitT(t, bare, "rev-parse", "refs/heads/main"), gitT(t, repo, "rev-parse", "HEAD"); remote != local {
			t.Errorf("remote main %s, local HEAD %s", remote, local)
		}
		snap, err := g.Snapshot(repo)
		if err != nil {
			t.Fatal(err)
		}
		if b := snap.Branch; b.Upstream != "origin/main" || b.Ahead != 0 || b.Behind != 0 || b.Gone {
			t.Errorf("branch line after the push: %+v", b)
		}
	})

	t.Run("two commits ahead, pushed at once", func(t *testing.T) {
		repo := testRepo(t)
		bare := bareRemote(t, repo, "origin")
		g, _ := tracedGit(t)
		commit(t, repo, "b.txt", "b\n")
		commit(t, repo, "c.txt", "c\n")
		snap, err := g.Snapshot(repo)
		if err != nil || snap.Branch.Ahead != 2 || snap.Branch.Upstream != "origin/main" {
			t.Fatalf("before: %+v, %v", snap.Branch, err)
		}
		tg, err := g.PushTarget(repo)
		if err != nil || tg.Ahead != 2 || tg.Behind != 0 {
			t.Fatalf("target %+v, %v", tg, err)
		}
		if _, err := g.Push(repo, nil); err != nil {
			t.Fatal(err)
		}
		if n := gitT(t, bare, "rev-list", "--count", "refs/heads/main"); n != "3" {
			t.Errorf("the remote has %s commits, want 3", n)
		}
	})

	t.Run("a branch with another name on the remote", func(t *testing.T) {
		repo := testRepo(t)
		bare := bareRemote(t, repo, "origin")
		g, _ := tracedGit(t)
		gitT(t, repo, "checkout", "-q", "-b", "work")
		commit(t, repo, "w.txt", "w\n")
		gitT(t, repo, "push", "-q", "-u", "origin", "work:topic")
		commit(t, repo, "w2.txt", "w2\n")
		// push.default=simple would refuse a different name; upstream sends it
		// where the branch tracks.
		gitT(t, repo, "config", "push.default", "upstream")
		res, err := g.Push(repo, nil)
		if err != nil {
			t.Fatal(err)
		}
		if res.Target.RemoteRef != "refs/heads/topic" || res.Target.Upstream != "origin/topic" {
			t.Errorf("target %+v", res.Target)
		}
		if remote, local := gitT(t, bare, "rev-parse", "refs/heads/topic"), gitT(t, repo, "rev-parse", "HEAD"); remote != local {
			t.Errorf("remote topic %s, local HEAD %s", remote, local)
		}
	})

	t.Run("push.followTags does not send tags", func(t *testing.T) {
		repo := testRepo(t)
		bare := bareRemote(t, repo, "origin")
		g, _ := tracedGit(t)
		gitT(t, repo, "config", "push.followTags", "true")
		commit(t, repo, "b.txt", "b\n")
		gitT(t, repo, "tag", "-a", "-m", "v1", "v1")
		if _, err := g.Push(repo, nil); err != nil {
			t.Fatal(err)
		}
		if tags := gitT(t, bare, "tag"); tags != "" {
			t.Errorf("the remote holds tags %q", tags)
		}
	})
}

func TestSourcePushRefused(t *testing.T) {
	cases := []struct {
		name  string
		setup func(t *testing.T) string
		kind  string
		msg   string
	}{
		{"no remote", func(t *testing.T) string {
			repo := testRepo(t)
			commit(t, repo, "b.txt", "b\n")
			return repo
		}, "no-remote", "No remote is configured for this repository. Add one in a terminal: git remote add <name> <url>"},
		{"no upstream, one remote", func(t *testing.T) string {
			repo := testRepo(t)
			bareRemote(t, repo, "origin")
			gitT(t, repo, "checkout", "-q", "-b", "feature")
			commit(t, repo, "f.txt", "f\n")
			return repo
		}, "no-upstream", "The branch feature has no upstream. Set one in a terminal: git push -u origin feature"},
		{"no upstream, two remotes", func(t *testing.T) string {
			repo := testRepo(t)
			bareRemote(t, repo, "origin")
			bareRemote(t, repo, "fork")
			gitT(t, repo, "checkout", "-q", "-b", "feature")
			commit(t, repo, "f.txt", "f\n")
			return repo
		}, "no-upstream", "The branch feature has no upstream. Set one in a terminal: git push -u <remote> feature"},
		{"a local upstream", func(t *testing.T) string {
			repo := testRepo(t)
			bareRemote(t, repo, "origin")
			gitT(t, repo, "checkout", "-q", "-b", "dev")
			gitT(t, repo, "branch", "--set-upstream-to=main")
			commit(t, repo, "d.txt", "d\n")
			return repo
		}, "local-upstream", "The upstream of dev is the local branch main: there is no remote to push to."},
		{"a gone upstream", func(t *testing.T) string {
			repo := testRepo(t)
			bare := bareRemote(t, repo, "origin")
			gitT(t, repo, "checkout", "-q", "-b", "feature")
			commit(t, repo, "f.txt", "f\n")
			gitT(t, repo, "push", "-q", "-u", "origin", "feature")
			gitT(t, bare, "update-ref", "-d", "refs/heads/feature")
			gitT(t, repo, "fetch", "-q", "--prune")
			commit(t, repo, "g.txt", "g\n")
			return repo
		}, "gone", "The upstream origin/feature is gone from its remote: push from a terminal to create it again."},
		{"a pushRemote elsewhere", func(t *testing.T) string {
			repo := testRepo(t)
			bareRemote(t, repo, "origin")
			bareRemote(t, repo, "fork")
			gitT(t, repo, "branch", "--set-upstream-to=origin/main")
			gitT(t, repo, "config", "branch.main.pushRemote", "fork")
			gitT(t, repo, "config", "push.default", "current")
			commit(t, repo, "b.txt", "b\n")
			return repo
		}, "elsewhere", "Git's push settings send main to fork/main, not to its upstream origin/main: push from a terminal."},
		{"a pushRemote under push.default=simple", func(t *testing.T) string {
			repo := testRepo(t)
			bareRemote(t, repo, "origin")
			bareRemote(t, repo, "fork")
			gitT(t, repo, "branch", "--set-upstream-to=origin/main")
			gitT(t, repo, "config", "branch.main.pushRemote", "fork")
			commit(t, repo, "b.txt", "b\n")
			return repo
		}, "elsewhere", "Git's push settings do not push main to its upstream origin/main: push from a terminal."},
		{"a pushRemote with no tracking branch yet", func(t *testing.T) string {
			repo := testRepo(t)
			bareRemote(t, repo, "origin")
			gitT(t, repo, "remote", "add", "fork", filepath.Join(t.TempDir(), "fork.git"))
			gitT(t, repo, "config", "branch.main.pushRemote", "fork")
			gitT(t, repo, "config", "push.default", "current")
			commit(t, repo, "b.txt", "b\n")
			return repo
		}, "elsewhere", "Git's push settings send main to fork/main, not to its upstream origin/main: push from a terminal."},
		{"push.default=nothing", func(t *testing.T) string {
			repo := testRepo(t)
			bareRemote(t, repo, "origin")
			gitT(t, repo, "config", "push.default", "nothing")
			commit(t, repo, "b.txt", "b\n")
			return repo
		}, "elsewhere", "Git's push settings do not push main to its upstream origin/main: push from a terminal."},
		{"simple with another name", func(t *testing.T) string {
			repo := testRepo(t)
			bareRemote(t, repo, "origin")
			gitT(t, repo, "checkout", "-q", "-b", "work")
			commit(t, repo, "w.txt", "w\n")
			gitT(t, repo, "push", "-q", "-u", "origin", "work:topic")
			commit(t, repo, "w2.txt", "w2\n")
			return repo
		}, "elsewhere", "Git's push settings do not push work to its upstream origin/topic: push from a terminal."},
		{"nothing to push", func(t *testing.T) string {
			repo := testRepo(t)
			bareRemote(t, repo, "origin")
			return repo
		}, "up-to-date", "Nothing to push: main is up to date with origin/main."},
		{"a detached HEAD", func(t *testing.T) string {
			repo := testRepo(t)
			bareRemote(t, repo, "origin")
			commit(t, repo, "b.txt", "b\n")
			gitT(t, repo, "checkout", "-q", "--detach")
			return repo
		}, "detached", "HEAD is detached: there is no branch to push."},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			repo := c.setup(t)
			g, log := tracedGit(t)
			_, err := g.Push(repo, nil)
			wantRefusal(t, err, c.kind, c.msg)
			if log.ran("push") {
				t.Error("git push ran")
			}
			if _, err := g.PushTarget(repo); err != nil && c.kind == "up-to-date" {
				t.Errorf("PushTarget refused an up-to-date branch: %v", err)
			} else if err == nil && c.kind != "up-to-date" {
				t.Error("PushTarget accepted what Push refused")
			}
		})
	}
}

func TestSourcePushFails(t *testing.T) {
	t.Run("another clone pushed first: fetch first, then non-fast-forward", func(t *testing.T) {
		repo := testRepo(t)
		bare := bareRemote(t, repo, "origin")
		other := filepath.Join(t.TempDir(), "other")
		gitT(t, repo, "clone", "-q", bare, other)
		commit(t, other, "o.txt", "o\n")
		gitT(t, other, "push", "-q", "origin", "main")
		commit(t, repo, "b.txt", "b\n")
		g, _ := tracedGit(t)
		head, remote := gitT(t, repo, "rev-parse", "HEAD"), gitT(t, bare, "rev-parse", "refs/heads/main")

		for _, reason := range []string{"fetch first", "non-fast-forward"} {
			_, err := g.Push(repo, nil)
			var pr PushRejected
			if !errors.As(err, &pr) || !pr.NonFastForward || pr.Remote || pr.Reason != reason {
				t.Fatalf("%s: error %v (%T), want a non-fast-forward rejection", reason, err, err)
			}
			if want := "The push was rejected: origin/main has commits that main does not (non-fast-forward). Pull or merge them in a terminal, then push."; err.Error() != want {
				t.Errorf("message %q, want %q", err.Error(), want)
			}
			if gitT(t, repo, "rev-parse", "HEAD") != head || gitT(t, bare, "rev-parse", "refs/heads/main") != remote {
				t.Errorf("%s: the local or the remote branch changed", reason)
			}
			if reason == "fetch first" {
				gitT(t, repo, "fetch", "-q")
			}
		}
	})

	t.Run("the remote's pre-receive hook rejects", func(t *testing.T) {
		repo := testRepo(t)
		bare := bareRemote(t, repo, "origin")
		hook(t, filepath.Join(bare, "hooks"), "pre-receive", "echo 'no pushes today' >&2\nexit 1")
		commit(t, repo, "b.txt", "b\n")
		g, _ := tracedGit(t)
		remote := gitT(t, bare, "rev-parse", "refs/heads/main")
		_, err := g.Push(repo, nil)
		var pr PushRejected
		if !errors.As(err, &pr) || !pr.Remote || pr.NonFastForward {
			t.Fatalf("error %v (%T), want a remote rejection", err, err)
		}
		if !strings.Contains(pr.Reason, "pre-receive hook declined") {
			t.Errorf("reason %q", pr.Reason)
		}
		if want := "The remote rejected the push: pre-receive hook declined."; err.Error() != want {
			t.Errorf("message %q, want %q", err.Error(), want)
		}
		if gitT(t, bare, "rev-parse", "refs/heads/main") != remote {
			t.Error("the remote changed")
		}
	})

	t.Run("an unreachable remote is git's error", func(t *testing.T) {
		repo := testRepo(t)
		bareRemote(t, repo, "origin")
		commit(t, repo, "b.txt", "b\n")
		gitT(t, repo, "remote", "set-url", "origin", filepath.Join(t.TempDir(), "missing.git"))
		g, _ := tracedGit(t)
		_, err := g.Push(repo, nil)
		var r Refusal
		var pr PushRejected
		if err == nil || errors.As(err, &r) || errors.As(err, &pr) {
			t.Fatalf("error %v (%T), want git's own", err, err)
		}
		if !strings.Contains(err.Error(), "fatal") || strings.Contains(err.Error(), "\n") {
			t.Errorf("message %q", err.Error())
		}
	})

	t.Run("a password in the URL is never shown", func(t *testing.T) {
		for _, k := range []string{"HTTPS_PROXY", "HTTP_PROXY", "ALL_PROXY", "https_proxy", "http_proxy", "all_proxy"} {
			t.Setenv(k, "")
		}
		t.Setenv("NO_PROXY", "*")
		t.Setenv("no_proxy", "*")
		repo := testRepo(t)
		bareRemote(t, repo, "origin")
		commit(t, repo, "b.txt", "b\n")
		gitT(t, repo, "remote", "set-url", "origin", "https://user:secret@127.0.0.1:1/x.git")
		g, _ := tracedGit(t)
		_, err := g.Push(repo, nil)
		if err == nil {
			t.Fatal("the push succeeded")
		}
		if strings.Contains(err.Error(), "secret") || strings.Contains(err.Error(), "user:") {
			t.Errorf("the message shows the credentials: %q", err.Error())
		}
		if !strings.Contains(err.Error(), "127.0.0.1") {
			t.Errorf("message %q does not name the host", err.Error())
		}
	})
}

func TestSourceTraceForbidsForce(t *testing.T) {
	for _, bad := range [][]string{{"push", "--force"}, {"push", "--force-with-lease=x"}, {"push", "-f"}, {"push", "o", "+refs/heads/main:refs/heads/main"},
		{"push", "o", "main:+x"}, {"push", "--mirror"}, {"push", "--all"}, {"push", "--tags"}, {"push", "--delete"},
		{"commit", "--allow-empty"}, {"commit", "--amend"}, {"commit", "--no-verify"}} {
		if forbiddenArg(bad) == "" {
			t.Errorf("%v is not seen as forbidden", bad)
		}
	}
	if a := forbiddenArg([]string{"-C", "x", "-c", "user.useConfigOnly=true", "commit", "--file=-", "push", "--porcelain", "origin", "refs/heads/main:refs/heads/main", "add", "-A", "--pathspec-file-nul"}); a != "" {
		t.Errorf("%q is seen as forbidden", a)
	}
}

// TestSourceStagedContents is plan 45's criterion 45.18 (design D14): the
// snapshot keeps each staged entry's blob in the index, "-" for a staged
// deletion, none for an entry not staged; a file staged again with other
// content after the snapshot refuses the commit, HEAD and the index as they
// were; staged again with the content shown, it commits.
func TestSourceStagedContents(t *testing.T) {
	repo := testRepo(t)
	commit(t, repo, "b.txt", "bee\n")
	write(t, filepath.Join(repo, "a.txt"), "one\ntwo\nthree\n")
	gitT(t, repo, "add", "a.txt")
	gitT(t, repo, "rm", "-q", "b.txt")
	write(t, filepath.Join(repo, "c.txt"), "sea\n")
	g, _ := tracedGit(t)
	snap, err := g.Snapshot(repo)
	if err != nil {
		t.Fatal(err)
	}
	blobs := map[string]string{}
	for _, e := range snap.Entries {
		blobs[e.Path] = e.Index
	}
	if want := gitT(t, repo, "rev-parse", ":0:a.txt"); blobs["a.txt"] != want || blobs["b.txt"] != "-" || blobs["c.txt"] != "" {
		t.Fatalf("blobs %q, want a.txt %s, b.txt -, c.txt none", blobs, want)
	}
	expect := stagedOf(t, g, repo)
	head := gitT(t, repo, "rev-parse", "HEAD")
	write(t, filepath.Join(repo, "a.txt"), "other\n")
	gitT(t, repo, "add", "a.txt")
	staged := gitT(t, repo, "rev-parse", ":0:a.txt")
	_, err = g.Commit(repo, "shown", expect)
	wantRefusal(t, err, "changed", "The staged changes changed since they were shown: check them, then commit again.")
	if gitT(t, repo, "rev-parse", "HEAD") != head || gitT(t, repo, "rev-parse", ":0:a.txt") != staged {
		t.Fatal("a refused commit moved HEAD or changed the index")
	}
	write(t, filepath.Join(repo, "a.txt"), "one\ntwo\nthree\n")
	gitT(t, repo, "add", "a.txt")
	res, err := g.Commit(repo, "shown", expect)
	if err != nil || res.Hash == head {
		t.Fatalf("the content shown staged again: %v, %+v", err, res)
	}
	if got := gitT(t, repo, "show", "HEAD:a.txt"); got != "one\ntwo\nthree" {
		t.Errorf("committed a.txt %q", got)
	}
}

// TestSourcePushStop is plan 45's criterion 45.19 (design D15): a push to a
// server that accepts and never answers; while one of the lock files a push
// could hold exists (the index's, HEAD's, packed-refs, the branch's and its
// remote-tracking ref's, each in turn), a stop is refused with its message
// and the push goes on; once none is, a stop ends the push within 5 s with
// the cancelled refusal, no lock file left and its remote-tracking ref as
// it was.
func TestSourcePushStop(t *testing.T) {
	repo := testRepo(t)
	for _, k := range []string{"HTTP_PROXY", "HTTPS_PROXY", "ALL_PROXY", "http_proxy", "https_proxy"} {
		t.Setenv(k, "")
	}
	t.Setenv("NO_PROXY", "*")
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	var conns sync.WaitGroup
	var connected atomic.Int32 // the connections a client holds open
	var mu sync.Mutex
	var open []net.Conn
	// closeAll ends the server: a git still waiting on it fails and ends.
	closeAll := func() {
		_ = ln.Close()
		mu.Lock()
		for _, c := range open {
			_ = c.Close()
		}
		mu.Unlock()
	}
	t.Cleanup(func() { closeAll(); conns.Wait() })
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			mu.Lock()
			open = append(open, c)
			mu.Unlock()
			conns.Add(1)
			connected.Add(1)
			go func() {
				defer conns.Done()
				defer connected.Add(-1)
				defer c.Close()
				b := make([]byte, 4096)
				for {
					if _, err := c.Read(b); err != nil {
						return
					}
				}
			}()
		}
	}()
	head := gitT(t, repo, "rev-parse", "HEAD")
	gitT(t, repo, "remote", "add", "slow", "http://"+ln.Addr().String()+"/x.git")
	gitT(t, repo, "update-ref", "refs/remotes/slow/main", head)
	gitT(t, repo, "config", "branch.main.remote", "slow")
	gitT(t, repo, "config", "branch.main.merge", "refs/heads/main")
	commit(t, repo, "b.txt", "bee\n")
	lock := filepath.Join(repo, ".git", "packed-refs.lock")
	write(t, lock, "")
	g, _ := tracedGit(t)
	stop := NewGitStop()
	done := make(chan error, 1)
	ended := make(chan struct{})
	go func() { defer close(ended); _, err := g.Push(repo, stop); done <- err }()
	t.Cleanup(func() {
		// A push a failure left running ends with the server, before its
		// repository is removed.
		closeAll()
		select {
		case <-ended:
		case <-time.After(10 * time.Second):
		}
	})
	deadline := time.Now().Add(15 * time.Second)
	for {
		ok, why := stop.Stop()
		if ok {
			t.Fatal("stopped with a lock file held")
		}
		if why == stopBusy {
			break
		}
		if strings.HasPrefix(why, "git could not be stopped") {
			t.Skipf("no console to send a Ctrl+Break to: %s", why)
		}
		if time.Now().After(deadline) {
			t.Fatal("the push never ran")
		}
		time.Sleep(50 * time.Millisecond)
	}
	if err := os.Remove(lock); err != nil {
		t.Fatal(err)
	}
	// Each of the other lock files a push could hold refuses the stop too.
	for _, name := range []string{"index.lock", "HEAD.lock", "refs/heads/main.lock", "refs/remotes/slow/main.lock"} {
		p := filepath.Join(repo, ".git", filepath.FromSlash(name))
		write(t, p, "")
		if ok, why := stop.Stop(); ok || why != stopBusy {
			t.Errorf("with %s: stopped %v, %q", name, ok, why)
		}
		if err := os.Remove(p); err != nil {
			t.Fatal(err)
		}
	}
	time.Sleep(500 * time.Millisecond)
	select {
	case err := <-done:
		t.Fatalf("the push ended while its stop was refused: %v", err)
	default:
	}
	if connected.Load() == 0 {
		t.Fatal("no connection to the server while the push ran")
	}
	if ok, why := stop.Stop(); !ok {
		t.Fatalf("the stop refused with no lock file: %q", why)
	}
	select {
	case err := <-done:
		wantRefusal(t, err, "cancelled", "The push was cancelled. The remote may have received it: Push again to check.")
	case <-time.After(5 * time.Second):
		t.Fatal("the push did not end 5 s after its stop")
	}
	// The transport, the process holding the server's connection, ended with
	// git: every connection closed, no git process left.
	for end := time.Now().Add(5 * time.Second); connected.Load() > 0 && time.Now().Before(end); {
		time.Sleep(50 * time.Millisecond)
	}
	if n := connected.Load(); n > 0 {
		t.Errorf("%d connections open 5 s after the stop: a git process left", n)
	}
	var left []string
	_ = filepath.WalkDir(filepath.Join(repo, ".git"), func(p string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() && strings.HasSuffix(p, ".lock") {
			left = append(left, p)
		}
		return nil
	})
	if len(left) > 0 || gitT(t, repo, "rev-parse", "refs/remotes/slow/main") != head {
		t.Errorf("after the stop: lock files %q, slow/main %s", left, gitT(t, repo, "rev-parse", "refs/remotes/slow/main"))
	}
	if ok, why := stop.Stop(); ok || why != "" {
		t.Errorf("a stop once git ended: %v %q", ok, why)
	}
}

// TestCancelledPushNamesLocks is design D15's cancelled refusal: a lock
// file git left behind is named, for the owner to delete, and left as it
// is; with none left, the refusal says only that the push was cancelled.
func TestCancelledPushNamesLocks(t *testing.T) {
	dir := t.TempDir()
	held, gone := filepath.Join(dir, "index.lock"), filepath.Join(dir, "HEAD.lock")
	write(t, held, "")
	plain := "The push was cancelled. The remote may have received it: Push again to check."
	if r := cancelledPush([]string{gone}); r.Kind != "cancelled" || r.Msg != plain {
		t.Errorf("no lock left: %+v", r)
	}
	want := plain + " git left " + held + " behind: delete it once no git command runs in this repository."
	if r := cancelledPush([]string{held, gone}); r.Kind != "cancelled" || r.Msg != want {
		t.Errorf("a lock left: %q\nwant %q", r.Msg, want)
	}
	if _, err := os.Stat(held); err != nil {
		t.Errorf("the lock left: %v", err)
	}
}

// TestGitStopUnderLock is design D15's order: a stop's way to stop runs
// under the lock git's end takes before letting its pid go (probe P45d), so
// the end waits for a stop under way; ended, a stop names nothing.
func TestGitStopUnderLock(t *testing.T) {
	s := NewGitStop()
	inside, leave := make(chan struct{}), make(chan struct{})
	s.Set(func() (bool, string) { close(inside); <-leave; return true, "" })
	stopped := make(chan bool, 1)
	go func() { ok, _ := s.Stop(); stopped <- ok }()
	<-inside
	ended := make(chan struct{})
	go func() { s.Set(nil); close(ended) }()
	select {
	case <-ended:
		t.Fatal("git's end did not wait for the stop under way")
	case <-time.After(200 * time.Millisecond):
	}
	close(leave)
	<-ended
	if !<-stopped {
		t.Error("the stop under way not asked")
	}
	select {
	case <-s.Asked():
	default:
		t.Error("Asked not closed after the stop")
	}
	if ok, why := s.Stop(); ok || why != "" {
		t.Errorf("a stop once git ended: %v %q", ok, why)
	}
}
