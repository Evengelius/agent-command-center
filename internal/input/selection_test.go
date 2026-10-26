package input

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// Plan 50, criteria 50.4 to 50.8 (design D7, D8): a selection's commit on
// temporary repositories, real git. No test touches the repository these
// files are in.

// shownOf is the snapshot's entries of the project dir for paths, as the
// panel hands a selection over (plan 50, design D1).
func shownOf(t *testing.T, g GitCLI, dir string, paths ...string) []GitEntry {
	t.Helper()
	snap, err := g.Snapshot(dir)
	if err != nil {
		t.Fatal(err)
	}
	var out []GitEntry
	for _, p := range paths {
		n := len(out)
		for _, e := range snap.Entries {
			if e.Path == p {
				out = append(out, e)
			}
		}
		if len(out) == n {
			t.Fatalf("%s is no change of the snapshot: %+v", p, snap.Entries)
		}
	}
	return out
}

// sha is the hex SHA-256 of b.
func sha(b []byte) string { s := sha256.Sum256(b); return hex.EncodeToString(s[:]) }

// indexBytes is the SHA-256 of the repository's index file.
func indexBytes(t *testing.T, repo string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(repo, ".git", "index"))
	if err != nil {
		t.Fatal(err)
	}
	return sha(b)
}

// treeFiles is every working-tree file of root but .git's, its SHA-256 by
// its slash path.
func treeFiles(t *testing.T, root string) map[string]string {
	t.Helper()
	files := map[string]string{}
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, p)
		files[filepath.ToSlash(rel)] = sha(b)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return files
}

// indexEntries is git ls-files -s: each path's "mode blob stage".
func indexEntries(t *testing.T, repo string) map[string]string {
	t.Helper()
	m := map[string]string{}
	for _, rec := range strings.Split(gitRaw(t, repo, "ls-files", "-s", "-z"), "\x00") {
		if meta, p, ok := strings.Cut(rec, "\t"); ok {
			m[p] = meta
		}
	}
	return m
}

// committed is the paths commit rev changes, sorted, renames as their two
// sides.
func committed(t *testing.T, repo, rev string, root bool) []string {
	t.Helper()
	args := []string{"diff-tree", "-r", "-z", "--no-commit-id", "--name-only", "--no-renames"}
	if root {
		args = append(args, "--root")
	}
	var out []string
	for _, p := range strings.Split(gitRaw(t, repo, append(args, rev)...), "\x00") {
		if p != "" {
			out = append(out, p)
		}
	}
	slices.Sort(out)
	return out
}

// differ is the paths, of those asked, whose entry differs between a and b.
func differ(a, b map[string]string, paths ...string) []string {
	var d []string
	for _, p := range paths {
		if a[p] != b[p] {
			d = append(d, p)
		}
	}
	return d
}

func TestSelectionCommit(t *testing.T) {
	t.Parallel()
	t.Run("a staged file alone, the others as they were", func(t *testing.T) {
		t.Parallel()
		repo := testRepo(t)
		g, _ := tracedGit(t)
		commit(t, repo, "b.txt", "b\n")
		commit(t, repo, "c.txt", "c\n")
		commit(t, repo, "p.txt", "p1\np2\n")
		write(t, filepath.Join(repo, "a.txt"), "ONE\ntwo\n")
		gitT(t, repo, "add", "a.txt")
		write(t, filepath.Join(repo, "b.txt"), "B\n") // staged, not selected
		gitT(t, repo, "add", "b.txt")
		write(t, filepath.Join(repo, "c.txt"), "C\n") // unstaged
		write(t, filepath.Join(repo, "p.txt"), "P1\np2\n")
		gitT(t, repo, "add", "p.txt")
		write(t, filepath.Join(repo, "p.txt"), "P1\nP2\n") // partly staged
		write(t, filepath.Join(repo, "u.txt"), "u\n")      // untracked
		old := gitT(t, repo, "rev-parse", "HEAD")
		idx, files := indexEntries(t, repo), treeFiles(t, repo)
		res, err := g.CommitSelection(repo, "only a", shownOf(t, g, repo, "a.txt"), nil)
		if err != nil {
			t.Fatal(err)
		}
		if head := gitT(t, repo, "rev-parse", "HEAD"); head != res.Hash || gitT(t, repo, "rev-parse", "HEAD^1") != old {
			t.Errorf("result %+v, HEAD %s on %s, want on %s", res, head, gitT(t, repo, "rev-parse", "HEAD^1"), old)
		}
		if got := committed(t, repo, "HEAD", false); !slices.Equal(got, []string{"a.txt"}) {
			t.Errorf("the commit changes %q, want a.txt alone", got)
		}
		if got := gitRaw(t, repo, "show", "HEAD:a.txt"); got != "ONE\ntwo\n" {
			t.Errorf("HEAD:a.txt = %q", got)
		}
		if d := differ(idx, indexEntries(t, repo), "a.txt", "b.txt", "c.txt", "p.txt", "u.txt"); d != nil {
			t.Errorf("index entries changed: %q", d)
		}
		if d := differ(files, treeFiles(t, repo), "a.txt", "b.txt", "c.txt", "p.txt", "u.txt"); d != nil {
			t.Errorf("working-tree files changed: %q", d)
		}
		if st := gitRaw(t, repo, "status", "--porcelain"); st != "M  b.txt\n M c.txt\nMM p.txt\n?? u.txt\n" {
			t.Errorf("status after:\n%s", st)
		}
	})

	t.Run("an unstaged and an untracked file, staged by the question", func(t *testing.T) {
		t.Parallel()
		repo := testRepo(t)
		g, _ := tracedGit(t)
		commit(t, repo, "b.txt", "b\n")
		commit(t, repo, "c.txt", "c\n")
		write(t, filepath.Join(repo, "b.txt"), "B\n")
		gitT(t, repo, "add", "b.txt")
		write(t, filepath.Join(repo, "c.txt"), "C\n")
		write(t, filepath.Join(repo, "u.txt"), "u\n")
		sel := shownOf(t, g, repo, "c.txt", "u.txt")
		if err := g.CheckSelection(repo, sel); err != nil {
			t.Fatalf("the check refuses files to stage: %v", err)
		}
		b := indexEntries(t, repo)["b.txt"]
		if err := g.Stage(repo, []string{"c.txt", "u.txt"}); err != nil {
			t.Fatal(err)
		}
		if _, err := g.CommitSelection(repo, "c and u", sel, []string{"c.txt", "u.txt"}); err != nil {
			t.Fatal(err)
		}
		if got := committed(t, repo, "HEAD", false); !slices.Equal(got, []string{"c.txt", "u.txt"}) {
			t.Errorf("the commit changes %q", got)
		}
		if got := indexEntries(t, repo)["b.txt"]; got != b {
			t.Errorf("b.txt's index entry %q, want %q", got, b)
		}
		if st := gitRaw(t, repo, "status", "--porcelain"); st != "M  b.txt\n" {
			t.Errorf("status after:\n%s", st)
		}
	})

	t.Run("a file of an untracked folder, its other file left", func(t *testing.T) {
		t.Parallel()
		repo := testRepo(t)
		g, _ := tracedGit(t)
		if err := os.Mkdir(filepath.Join(repo, "dir"), 0o700); err != nil {
			t.Fatal(err)
		}
		write(t, filepath.Join(repo, "dir", "1.txt"), "1\n")
		write(t, filepath.Join(repo, "dir", "2.txt"), "2\n")
		sel := []GitEntry{{X: '?', Y: '?', Path: "dir/1.txt", RepoPath: "dir/1.txt"}} // the panel's 'u' row
		if err := g.CheckSelection(repo, sel); err != nil {
			t.Fatal(err)
		}
		if err := g.Stage(repo, []string{"dir/1.txt"}); err != nil {
			t.Fatal(err)
		}
		if _, err := g.CommitSelection(repo, "one", sel, []string{"dir/1.txt"}); err != nil {
			t.Fatal(err)
		}
		if got := committed(t, repo, "HEAD", false); !slices.Equal(got, []string{"dir/1.txt"}) {
			t.Errorf("the commit changes %q", got)
		}
		if st := gitRaw(t, repo, "status", "--porcelain", "--untracked-files=all"); st != "?? dir/2.txt\n" {
			t.Errorf("status after:\n%s", st)
		}
	})

	t.Run("a staged deletion and an unstaged one", func(t *testing.T) {
		t.Parallel()
		repo := testRepo(t)
		g, _ := tracedGit(t)
		commit(t, repo, "d1.txt", "1\n")
		commit(t, repo, "d2.txt", "2\n")
		gitT(t, repo, "rm", "-q", "d1.txt")
		if err := os.Remove(filepath.Join(repo, "d2.txt")); err != nil {
			t.Fatal(err)
		}
		sel := shownOf(t, g, repo, "d1.txt", "d2.txt")
		if err := g.Stage(repo, []string{"d2.txt"}); err != nil {
			t.Fatal(err)
		}
		if _, err := g.CommitSelection(repo, "gone", sel, []string{"d2.txt"}); err != nil {
			t.Fatal(err)
		}
		if got := committed(t, repo, "HEAD", false); !slices.Equal(got, []string{"d1.txt", "d2.txt"}) {
			t.Errorf("the commit changes %q", got)
		}
		if files := gitT(t, repo, "ls-tree", "--name-only", "HEAD"); files != "a.txt" {
			t.Errorf("HEAD holds %q", files)
		}
	})

	t.Run("a rename, both sides", func(t *testing.T) {
		t.Parallel()
		repo := testRepo(t)
		g, _ := tracedGit(t)
		commit(t, repo, "old.txt", strings.Repeat("a line of the file\n", 20))
		gitT(t, repo, "mv", "old.txt", "new.txt")
		sel := shownOf(t, g, repo, "new.txt")
		if sel[0].X != 'R' || sel[0].Orig != "old.txt" {
			t.Fatalf("the panel shows %+v, not a rename", sel[0])
		}
		if _, err := g.CommitSelection(repo, "rename", sel, nil); err != nil {
			t.Fatal(err)
		}
		if got := committed(t, repo, "HEAD", false); !slices.Equal(got, []string{"new.txt", "old.txt"}) {
			t.Errorf("the commit changes %q, want both sides", got)
		}
		if st := gitRaw(t, repo, "status", "--porcelain"); st != "" {
			t.Errorf("status after:\n%s", st)
		}
	})

	// A file moved on disk shows as its source deleted and the file
	// untracked, either side perhaps staged already; once the question has
	// staged the rest, git pairs them as a rename, which the check reads
	// path by path (the review of plan 50).
	for _, c := range []struct {
		name, shown string
		before      func(t *testing.T, repo string) // after old.txt moved to new.txt
		stage       []string                        // the question's
	}{
		{"neither side staged", " D old.txt\n?? new.txt\n", func(*testing.T, string) {}, []string{"new.txt", "old.txt"}},
		{"the file staged", "A  new.txt\n D old.txt\n", func(t *testing.T, repo string) { gitT(t, repo, "add", "new.txt") }, []string{"old.txt"}},
		{"the deletion staged", "D  old.txt\n?? new.txt\n", func(t *testing.T, repo string) { gitT(t, repo, "rm", "-q", "old.txt") }, []string{"new.txt"}},
	} {
		t.Run("a file moved on disk, both sides selected, "+c.name, func(t *testing.T) {
			t.Parallel()
			repo := testRepo(t)
			g, _ := tracedGit(t)
			commit(t, repo, "old.txt", strings.Repeat("a line of the file\n", 20))
			if err := os.Rename(filepath.Join(repo, "old.txt"), filepath.Join(repo, "new.txt")); err != nil {
				t.Fatal(err)
			}
			c.before(t, repo)
			if st := gitRaw(t, repo, "status", "--porcelain"); st != c.shown {
				t.Fatalf("status before:\n%s", st)
			}
			sel := shownOf(t, g, repo, "new.txt", "old.txt")
			if err := g.CheckSelection(repo, sel); err != nil {
				t.Fatal(err)
			}
			if err := g.Stage(repo, c.stage); err != nil {
				t.Fatal(err)
			}
			if _, err := g.CommitSelection(repo, "moved", sel, c.stage); err != nil {
				t.Fatal(err)
			}
			if got := gitT(t, repo, "diff-tree", "--no-commit-id", "--name-status", "-r", "-M", "HEAD"); got != "R100\told.txt\tnew.txt" {
				t.Errorf("the commit changes %q, want the rename", got)
			}
			if st := gitRaw(t, repo, "status", "--porcelain"); st != "" {
				t.Errorf("status after:\n%s", st)
			}
		})
	}

	t.Run("names taken as they are, not as patterns", func(t *testing.T) {
		t.Parallel()
		repo := testRepo(t)
		g, _ := tracedGit(t)
		names := []string{"a b.txt", "[x].txt", "-dash.txt"}
		for _, n := range append(names, "x.txt") { // "[x].txt" as a pattern matches x.txt
			write(t, filepath.Join(repo, n), n+"\n")
			gitT(t, repo, "--literal-pathspecs", "add", "--", n)
		}
		x := indexEntries(t, repo)["x.txt"]
		if _, err := g.CommitSelection(repo, "names", shownOf(t, g, repo, names...), nil); err != nil {
			t.Fatal(err)
		}
		want := slices.Clone(names)
		slices.Sort(want)
		if got := committed(t, repo, "HEAD", false); !slices.Equal(got, want) {
			t.Errorf("the commit changes %q, want %q", got, want)
		}
		if got := indexEntries(t, repo)["x.txt"]; got != x || !strings.Contains(gitRaw(t, repo, "status", "--porcelain"), "A  x.txt") {
			t.Errorf("x.txt not left staged: %q", got)
		}
	})

	t.Run("a project in a subfolder: a file staged outside it left staged", func(t *testing.T) {
		t.Parallel()
		repo := testRepo(t)
		g, _ := tracedGit(t)
		sub := filepath.Join(repo, "sub")
		if err := os.Mkdir(sub, 0o700); err != nil {
			t.Fatal(err)
		}
		commit(t, repo, "sub/s.txt", "s\n")
		commit(t, repo, "top.txt", "t\n")
		write(t, filepath.Join(sub, "s.txt"), "S\n")
		write(t, filepath.Join(repo, "top.txt"), "T\n")
		gitT(t, repo, "add", "sub/s.txt", "top.txt")
		top := indexEntries(t, repo)["top.txt"]
		if _, err := g.CommitSelection(sub, "sub", shownOf(t, g, sub, "s.txt"), nil); err != nil {
			t.Fatal(err)
		}
		if got := committed(t, repo, "HEAD", false); !slices.Equal(got, []string{"sub/s.txt"}) {
			t.Errorf("the commit changes %q", got)
		}
		if got := indexEntries(t, repo)["top.txt"]; got != top || gitRaw(t, repo, "status", "--porcelain") != "M  top.txt\n" {
			t.Errorf("top.txt not left staged: %q, status %q", got, gitRaw(t, repo, "status", "--porcelain"))
		}
	})

	t.Run("a branch's first commit", func(t *testing.T) {
		t.Parallel()
		gitEnv(t)
		repo := t.TempDir()
		gitT(t, repo, "init", "-q", "-b", "main")
		write(t, filepath.Join(repo, "f.txt"), "f\n")
		write(t, filepath.Join(repo, "g.txt"), "g\n")
		gitT(t, repo, "add", "f.txt", "g.txt")
		g, _ := tracedGit(t)
		res, err := g.CommitSelection(repo, "root", shownOf(t, g, repo, "f.txt"), nil)
		if err != nil {
			t.Fatal(err)
		}
		if head := gitT(t, repo, "rev-parse", "HEAD"); head != res.Hash {
			t.Errorf("HEAD %s, result %+v", head, res)
		}
		if got := committed(t, repo, "HEAD", true); !slices.Equal(got, []string{"f.txt"}) {
			t.Errorf("the root commit holds %q", got)
		}
		if st := gitRaw(t, repo, "status", "--porcelain"); st != "A  g.txt\n" {
			t.Errorf("status after:\n%s", st)
		}
	})
}

// TestSelectionHooks is criteria 50.7 and 50.8 at git's level: the
// pre-commit hook runs once and sees the selection alone staged; one that
// refuses makes no commit, and the files the question staged stay staged,
// as git leaves them, until a commit without the hook takes exactly them.
func TestSelectionHooks(t *testing.T) {
	t.Parallel()
	t.Run("the hook sees the selection alone", func(t *testing.T) {
		t.Parallel()
		repo := testRepo(t)
		g, _ := tracedGit(t)
		commit(t, repo, "b.txt", "b\n")
		log := filepath.ToSlash(filepath.Join(t.TempDir(), "hook.log"))
		hook(t, filepath.Join(repo, ".git", "hooks"), "pre-commit", "git diff --cached --name-only >> '"+log+"'")
		write(t, filepath.Join(repo, "a.txt"), "A\n")
		write(t, filepath.Join(repo, "b.txt"), "B\n")
		gitT(t, repo, "add", "a.txt", "b.txt")
		if _, err := g.CommitSelection(repo, "a", shownOf(t, g, repo, "a.txt"), nil); err != nil {
			t.Fatal(err)
		}
		if got, err := os.ReadFile(filepath.FromSlash(log)); err != nil || string(got) != "a.txt\n" {
			t.Errorf("the hook saw %q (%v), want a.txt once", got, err)
		}
	})

	t.Run("a refusing hook: no commit, the question's files stay staged", func(t *testing.T) {
		t.Parallel()
		repo := testRepo(t)
		g, _ := tracedGit(t)
		commit(t, repo, "b.txt", "b\n")
		commit(t, repo, "c.txt", "c\n")
		pre := filepath.Join(repo, ".git", "hooks", "pre-commit")
		hook(t, filepath.Join(repo, ".git", "hooks"), "pre-commit", "echo 'not today' >&2\nexit 1")
		write(t, filepath.Join(repo, "b.txt"), "B\n") // staged, not selected
		gitT(t, repo, "add", "b.txt")
		write(t, filepath.Join(repo, "c.txt"), "C\n")
		write(t, filepath.Join(repo, "u.txt"), "u\n")
		sel := shownOf(t, g, repo, "c.txt", "u.txt")
		old := gitT(t, repo, "rev-parse", "HEAD")
		files := treeFiles(t, repo)
		if err := g.Stage(repo, []string{"c.txt", "u.txt"}); err != nil {
			t.Fatal(err)
		}
		idx := indexEntries(t, repo) // the question's staging done
		_, err := g.CommitSelection(repo, "refused", sel, []string{"c.txt", "u.txt"})
		if err == nil || !strings.Contains(err.Error(), "not today") {
			t.Fatalf("error %v, want the hook's words", err)
		}
		if head := gitT(t, repo, "rev-parse", "HEAD"); head != old {
			t.Errorf("HEAD moved to %s", head)
		}
		now := indexEntries(t, repo)
		if d := differ(idx, now, "b.txt", "c.txt", "u.txt"); d != nil {
			t.Errorf("index entries changed by the failed commit: %q", d)
		}
		for _, f := range []string{"c.txt", "u.txt"} {
			if blob := gitT(t, repo, "hash-object", f); !strings.Contains(now[f], blob) {
				t.Errorf("%s's index entry %q, want its working-tree blob %s", f, now[f], blob)
			}
		}
		if d := differ(files, treeFiles(t, repo), "a.txt", "b.txt", "c.txt", "u.txt"); d != nil {
			t.Errorf("working-tree files changed: %q", d)
		}
		if st := gitRaw(t, repo, "status", "--porcelain"); st != "M  b.txt\nM  c.txt\nA  u.txt\n" {
			t.Errorf("status after the failure:\n%s", st)
		}
		// The hook gone, a commit of the selection as now shown asks nothing
		// and takes exactly the two files.
		if err := os.Remove(pre); err != nil {
			t.Fatal(err)
		}
		if _, err := g.CommitSelection(repo, "again", shownOf(t, g, repo, "c.txt", "u.txt"), nil); err != nil {
			t.Fatal(err)
		}
		if got := committed(t, repo, "HEAD", false); !slices.Equal(got, []string{"c.txt", "u.txt"}) {
			t.Errorf("the retry changes %q", got)
		}
		if st := gitRaw(t, repo, "status", "--porcelain"); st != "M  b.txt\n" {
			t.Errorf("status after the retry:\n%s", st)
		}
	})
}

// TestSelectionRefusals is criterion 50.6 at git's level: each refusal
// says why, the index file and HEAD byte for byte as they were.
func TestSelectionRefusals(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name, kind, msg string
		// set makes the state and returns the selection as the panel shows it.
		set func(t *testing.T, g GitCLI, repo string) (dir string, sel []GitEntry)
	}{
		{"a partly staged file", "partial",
			"p.txt has staged and unstaged changes, and a selection commits whole files: leave it out of the selection, or stage the rest in a terminal.",
			func(t *testing.T, g GitCLI, repo string) (string, []GitEntry) {
				commit(t, repo, "p.txt", "p1\np2\n")
				write(t, filepath.Join(repo, "p.txt"), "P1\np2\n")
				gitT(t, repo, "add", "p.txt")
				write(t, filepath.Join(repo, "p.txt"), "P1\nP2\n")
				return repo, shownOf(t, g, repo, "p.txt")
			}},
		{"a file deleted from the index, still on disk", "partial",
			"k.txt is deleted in the index but still on disk: leave it out of the selection, or commit it from a terminal.",
			func(t *testing.T, g GitCLI, repo string) (string, []GitEntry) {
				commit(t, repo, "k.txt", "k\n")
				gitT(t, repo, "rm", "-q", "--cached", "k.txt")
				return repo, shownOf(t, g, repo, "k.txt")
			}},
		{"a conflict", "conflict", "Resolve the merge conflicts first.",
			func(t *testing.T, g GitCLI, repo string) (string, []GitEntry) {
				write(t, filepath.Join(repo, "a.txt"), "stashed\ntwo\n")
				gitT(t, repo, "stash", "-q")
				commit(t, repo, "a.txt", "committed\ntwo\n")
				if _, err := gitTry(repo, "stash", "apply"); err == nil {
					t.Fatal("the stash applied without a conflict")
				}
				write(t, filepath.Join(repo, "b.txt"), "b\n")
				gitT(t, repo, "add", "b.txt")
				return repo, shownOf(t, g, repo, "b.txt")
			}},
		{"a merge in progress", "merge", "A merge is in progress: commit it whole, without a selection.",
			func(t *testing.T, g GitCLI, repo string) (string, []GitEntry) {
				gitT(t, repo, "checkout", "-q", "-b", "side")
				commit(t, repo, "s.txt", "s\n")
				gitT(t, repo, "checkout", "-q", "main")
				commit(t, repo, "m.txt", "m\n")
				gitT(t, repo, "merge", "-q", "--no-ff", "--no-commit", "side")
				return repo, shownOf(t, g, repo, "s.txt")
			}},
		{"a rebase in progress", "rebase", "A rebase is in progress: finish it in a terminal first.",
			func(t *testing.T, g GitCLI, repo string) (string, []GitEntry) {
				write(t, filepath.Join(repo, "a.txt"), "A\n")
				gitT(t, repo, "add", "a.txt")
				sel := shownOf(t, g, repo, "a.txt")
				if err := os.Mkdir(filepath.Join(repo, ".git", "rebase-merge"), 0o700); err != nil {
					t.Fatal(err)
				}
				return repo, sel
			}},
		{"a file staged since it was shown", "changed", "c.txt changed since it was shown: check the selection, then commit again.",
			func(t *testing.T, g GitCLI, repo string) (string, []GitEntry) {
				commit(t, repo, "c.txt", "c\n")
				write(t, filepath.Join(repo, "c.txt"), "C\n")
				sel := shownOf(t, g, repo, "c.txt")
				gitT(t, repo, "add", "c.txt")
				return repo, sel
			}},
		{"a file staged again with other content", "changed", "a.txt changed since it was shown: check the selection, then commit again.",
			func(t *testing.T, g GitCLI, repo string) (string, []GitEntry) {
				write(t, filepath.Join(repo, "a.txt"), "A1\n")
				gitT(t, repo, "add", "a.txt")
				sel := shownOf(t, g, repo, "a.txt")
				write(t, filepath.Join(repo, "a.txt"), "A2\n")
				gitT(t, repo, "add", "a.txt")
				return repo, sel
			}},
		{"a rename's source outside the project", "outside", "new.txt was renamed from top-old.txt, outside this folder: commit it from a terminal.",
			func(t *testing.T, g GitCLI, repo string) (string, []GitEntry) {
				commit(t, repo, "top-old.txt", strings.Repeat("a line of the file\n", 20))
				if err := os.Mkdir(filepath.Join(repo, "sub"), 0o700); err != nil {
					t.Fatal(err)
				}
				gitT(t, repo, "mv", "top-old.txt", "sub/new.txt")
				// As ParseSnapshot keeps such a rename: its source outside, as git wrote it.
				return filepath.Join(repo, "sub"), []GitEntry{{X: 'R', Y: ' ', Path: "new.txt", Orig: "top-old.txt", RepoPath: "sub/new.txt", RepoOrig: "top-old.txt"}}
			}},
		{"a rename's source back on disk", "partial", "old.txt, the source of the rename to new.txt, is back on disk: commit it from a terminal.",
			func(t *testing.T, g GitCLI, repo string) (string, []GitEntry) {
				commit(t, repo, "old.txt", strings.Repeat("a line of the file\n", 20))
				gitT(t, repo, "mv", "old.txt", "new.txt")
				sel := shownOf(t, g, repo, "new.txt")
				write(t, filepath.Join(repo, "old.txt"), "back\n")
				return repo, sel
			}},
		{"a rename's source back in the index", "changed", "new.txt changed since it was shown: check the selection, then commit again.",
			func(t *testing.T, g GitCLI, repo string) (string, []GitEntry) {
				commit(t, repo, "old.txt", strings.Repeat("a line of the file\n", 20))
				gitT(t, repo, "mv", "old.txt", "new.txt")
				sel := shownOf(t, g, repo, "new.txt")
				gitT(t, repo, "restore", "--staged", "old.txt") // a commit would take its deletion on disk
				return repo, sel
			}},
		{"nothing selected", "nothing-selected", "No file is selected.",
			func(t *testing.T, g GitCLI, repo string) (string, []GitEntry) { return repo, nil }},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			repo := testRepo(t)
			g, _ := tracedGit(t)
			dir, sel := c.set(t, g, repo)
			idx, head := indexBytes(t, repo), gitT(t, repo, "rev-parse", "HEAD")
			wantRefusal(t, g.CheckSelection(dir, sel), c.kind, c.msg)
			_, err := g.CommitSelection(dir, "refused", sel, nil)
			wantRefusal(t, err, c.kind, c.msg)
			if got := indexBytes(t, repo); got != idx {
				t.Error("the index file changed")
			}
			if got := gitT(t, repo, "rev-parse", "HEAD"); got != head {
				t.Errorf("HEAD moved to %s", got)
			}
		})
	}
}

// TestSelectionUnexpected is the check after a selection's commit (plan
// 50, design D7): a hook that stages another file, or that changes a
// selected one, makes a commit other than the selection, which is told,
// the commit kept.
func TestSelectionUnexpected(t *testing.T) {
	t.Parallel()
	for _, c := range []struct{ name, body, what string }{
		{"another file staged by the hook", "git add other.txt", "it also holds other.txt, which was not selected"},
		{"the selected file changed by the hook", "printf 'hooked\\n' > a.txt && git add a.txt", "a.txt changed during the commit, which holds the newer content"},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			repo := testRepo(t)
			g, _ := tracedGit(t)
			hook(t, filepath.Join(repo, ".git", "hooks"), "pre-commit", c.body)
			write(t, filepath.Join(repo, "other.txt"), "other\n")
			write(t, filepath.Join(repo, "a.txt"), "A\n")
			gitT(t, repo, "add", "a.txt")
			res, err := g.CommitSelection(repo, "hooked", shownOf(t, g, repo, "a.txt"), nil)
			var u Unexpected
			if !errors.As(err, &u) {
				t.Fatalf("error %v (%T), want an Unexpected", err, err)
			}
			if head := gitT(t, repo, "rev-parse", "HEAD"); u.Commit.Hash != head || res != u.Commit || u.What != c.what {
				t.Errorf("Unexpected %+v, result %+v, HEAD %s; want What %q", u, res, head, c.what)
			}
			if want := "Committed " + u.Commit.Short + ", but " + c.what + ": check it in a terminal."; u.Error() != want {
				t.Errorf("message %q, want %q", u.Error(), want)
			}
		})
	}
}
