package input

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
)

// gitDated runs git in dir with its author and committer dates set to date,
// so that the tags' order does not hang on the clock.
func gitDated(t *testing.T, dir, date string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = append(os.Environ(), "GIT_AUTHOR_DATE="+date, "GIT_COMMITTER_DATE="+date)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
}

// branchCalls is a GitCLI whose calls the test checks at its end: none
// holds a force, a stash, a reset, a merge or a prune (plan 59, criterion
// 59.5).
func branchCalls(t *testing.T) GitCLI {
	t.Helper()
	var mu sync.Mutex
	var calls [][]string
	t.Cleanup(func() {
		mu.Lock()
		defer mu.Unlock()
		for _, c := range calls {
			for _, a := range c {
				switch {
				case a == "-f", a == "-m", a == "-p", a == "stash", a == "reset", a == "--discard-changes", a == "--merge",
					a == "--prune", a == "--prune-tags", strings.HasPrefix(a, "--force"):
					t.Errorf("a call holds %q: git %s", a, strings.Join(c, " "))
				}
			}
		}
	})
	return GitCLI{Trace: func(args []string) {
		mu.Lock()
		calls = append(calls, append([]string(nil), args...))
		mu.Unlock()
	}}
}

func refNames(refs []Ref) []string {
	var n []string
	for _, r := range refs {
		n = append(n, r.Name)
	}
	return n
}

// TestBranchesRead is plan 59, criterion 59.2, in git: on a repository with
// local branches, one checked out in another worktree, two remotes, a
// remote's HEAD, tags of three dates and a reflog of switches.
func TestBranchesRead(t *testing.T) {
	repo := testRepo(t)
	gitT(t, repo, "branch", "feat")
	gitT(t, repo, "branch", "fix/login")
	gitT(t, repo, "branch", "old")
	for _, b := range []string{"feat", "fix/login", "old", "main", "feat", "main"} {
		gitT(t, repo, "switch", "-q", b)
	}
	wt := filepath.Join(filepath.Dir(repo), "wt")
	gitT(t, repo, "worktree", "add", "-q", "-b", "wtb", wt)
	bareRemote(t, repo, "origin")
	gitT(t, repo, "remote", "set-head", "origin", "main")
	backup := filepath.Join(t.TempDir(), "backup.git")
	gitT(t, repo, "init", "--bare", "-q", "-b", "main", backup)
	gitT(t, repo, "remote", "add", "backup", backup)
	gitT(t, repo, "push", "-q", "backup", "main", "feat")
	// main one commit behind origin/main and one ahead of it.
	commit(t, repo, "b.txt", "pushed\n")
	gitT(t, repo, "push", "-q", "origin", "main")
	gitT(t, repo, "reset", "-q", "--hard", "HEAD~1")
	commit(t, repo, "c.txt", "local\n")
	// A lightweight tag's date is its commit's, an annotated one's its own.
	gitDated(t, repo, "2020-01-01T00:00:00Z", "commit", "-q", "--allow-empty", "-m", "2020")
	gitT(t, repo, "tag", "v1")
	gitDated(t, repo, "2022-01-01T00:00:00Z", "tag", "-a", "v2", "-m", "two")
	gitDated(t, repo, "2021-01-01T00:00:00Z", "commit", "-q", "--allow-empty", "-m", "2021")
	gitT(t, repo, "tag", "v3")

	b, err := branchCalls(t).Branches(repo)
	if err != nil {
		t.Fatal(err)
	}
	if h := b.Head; h.Name != "main" || h.Detached || h.NoCommits || h.Upstream != "origin/main" || h.Ahead != 3 || h.Behind != 1 {
		t.Errorf("the head: %+v", h)
	}
	if got, want := refNames(b.Local), []string{"feat", "fix/login", "main", "old", "wtb"}; !reflect.DeepEqual(got, want) {
		t.Errorf("local %q, want %q", got, want)
	}
	for _, r := range b.Local {
		if r.Current != (r.Name == "main") {
			t.Errorf("%s current %v", r.Name, r.Current)
		}
		if (r.Elsewhere != "") != (r.Name == "wtb") || r.Name == "wtb" && !samePath(r.Elsewhere, wt) {
			t.Errorf("%s elsewhere %q, want %q for wtb only", r.Name, r.Elsewhere, wt)
		}
	}
	if got, want := refNames(b.Remote), []string{"backup/feat", "backup/main", "origin/main"}; !reflect.DeepEqual(got, want) {
		t.Errorf("remote %q, want %q", got, want)
	}
	if got, want := refNames(b.Tags), []string{"v2", "v3", "v1"}; !reflect.DeepEqual(got, want) {
		t.Errorf("tags %q, want %q (newest first)", got, want)
	}
	if want := []string{"feat", "old", "fix/login"}; !reflect.DeepEqual(b.Recent, want) {
		t.Errorf("recent %q, want %q", b.Recent, want)
	}
	if b.LocalTotal != 5 || b.RemoteTotal != 3 || b.TagsTotal != 3 {
		t.Errorf("totals %d, %d, %d", b.LocalTotal, b.RemoteTotal, b.TagsTotal)
	}
}

// TestBranchesHead is plan 59, criterion 59.4, in git: a detached HEAD, and a
// branch with no commit yet.
func TestBranchesHead(t *testing.T) {
	repo := testRepo(t)
	id := gitT(t, repo, "rev-parse", "--short", "HEAD")
	gitT(t, repo, "switch", "-q", "--detach", "HEAD")
	b, err := GitCLI{}.Branches(repo)
	if err != nil {
		t.Fatal(err)
	}
	if h := b.Head; !h.Detached || h.Name != "HEAD" || h.Head != id {
		t.Errorf("detached: %+v, want at %s", h, id)
	}
	if len(b.Recent) != 0 {
		t.Errorf("recent %q", b.Recent)
	}

	fresh := filepath.Join(t.TempDir(), "fresh")
	gitT(t, filepath.Dir(fresh), "init", "-q", "-b", "trunk", fresh)
	b, err = GitCLI{}.Branches(fresh)
	if err != nil {
		t.Fatal(err)
	}
	if h := b.Head; h.Name != "trunk" || !h.NoCommits || h.Detached || len(b.Local) != 0 || len(b.Recent) != 0 {
		t.Errorf("no commits: %+v, local %q, recent %q", h, refNames(b.Local), b.Recent)
	}
}

// TestBranchesCap is plan 59, criterion 59.4: a group holds 1,000 rows at
// most and says how many there were.
func TestBranchesCap(t *testing.T) {
	repo := testRepo(t)
	id := gitT(t, repo, "rev-parse", "HEAD")
	var in strings.Builder
	for i := range BranchCap + 1 {
		fmt.Fprintf(&in, "create refs/tags/t%04d %s\n", i, id)
	}
	cmd := exec.Command("git", "-C", repo, "update-ref", "--stdin")
	cmd.Stdin = strings.NewReader(in.String())
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("update-ref: %v\n%s", err, out)
	}
	b, err := GitCLI{}.Branches(repo)
	if err != nil {
		t.Fatal(err)
	}
	if len(b.Tags) != BranchCap || b.TagsTotal != BranchCap+1 {
		t.Errorf("tags: %d shown of %d", len(b.Tags), b.TagsTotal)
	}
}

// TestBranchSwitch is plan 59, criterion 59.5, in git: a local branch, a
// remote one made a tracking branch, one whose local name exists, git's
// refusal of changes that would be lost, changes git carries along, and the
// count of changes.
func TestBranchSwitch(t *testing.T) {
	repo := testRepo(t)
	g := branchCalls(t)
	bareRemote(t, repo, "origin")
	gitT(t, repo, "switch", "-q", "-c", "away")
	commit(t, repo, "away.txt", "away\n")
	gitT(t, repo, "push", "-q", "origin", "away")
	gitT(t, repo, "switch", "-q", "main")
	gitT(t, repo, "branch", "-q", "-D", "away")
	gitT(t, repo, "branch", "feat")

	head := func() string { return gitT(t, repo, "symbolic-ref", "--short", "HEAD") }
	if to, err := g.Switch(repo, "feat", false); err != nil || head() != "feat" || to != "feat" {
		t.Fatalf("switch to feat: %q, %v, on %s", to, err, head())
	}
	if to, err := g.Switch(repo, "origin/away", true); err != nil || head() != "away" || to != "away" {
		t.Fatalf("switch to origin/away: %q, %v, on %s", to, err, head())
	}
	if up := gitT(t, repo, "rev-parse", "--abbrev-ref", "away@{upstream}"); up != "origin/away" {
		t.Errorf("away tracks %q", up)
	}
	var r Refusal
	if _, err := g.Switch(repo, "origin/main", true); !errors.As(err, &r) || r.Kind != "local-exists" ||
		r.Msg != "A local branch main exists already: switch to it in Local." || head() != "away" {
		t.Errorf("origin/main with main local: %v, on %s", err, head())
	}

	// a.txt differs between away and feat: a change to it is refused.
	gitT(t, repo, "switch", "-q", "feat")
	commit(t, repo, "a.txt", "feat's\n")
	gitT(t, repo, "switch", "-q", "away")
	write(t, filepath.Join(repo, "a.txt"), "mine\n")
	write(t, filepath.Join(repo, "new.txt"), "untracked\n")
	if n, err := g.Changes(repo); err != nil || n != 2 {
		t.Errorf("changes: %d, %v; want 2", n, err)
	}
	_, err := g.Switch(repo, "feat", false)
	if err == nil || !strings.Contains(err.Error(), "would be overwritten") || head() != "away" {
		t.Fatalf("a change git would lose: %v, on %s", err, head())
	}
	if b, _ := os.ReadFile(filepath.Join(repo, "a.txt")); string(b) != "mine\n" {
		t.Errorf("the change after the refusal: %q", b)
	}
	// away.txt is on away only: a change to it would be lost on main.
	gitT(t, repo, "checkout", "-q", "--", "a.txt")
	write(t, filepath.Join(repo, "away.txt"), "changed\n")
	if _, err := g.Switch(repo, "main", false); err == nil || head() != "away" {
		t.Fatalf("a change to a file main lacks: %v, on %s", err, head())
	}
	// a.txt is the same on away and main: a change to it comes along, and so
	// does the untracked new.txt.
	gitT(t, repo, "checkout", "-q", "--", "away.txt")
	write(t, filepath.Join(repo, "a.txt"), "carried\n")
	if _, err := g.Switch(repo, "main", false); err != nil || head() != "main" {
		t.Fatalf("a change git carries: %v, on %s", err, head())
	}
	for name, want := range map[string]string{"a.txt": "carried\n", "new.txt": "untracked\n"} {
		if b, _ := os.ReadFile(filepath.Join(repo, name)); string(b) != want {
			t.Errorf("%s after the switch: %q", name, b)
		}
	}
}

// TestBranchFetch is plan 59, criterion 59.7, in git: a fetch brings a
// remote's new commit and branch, prunes nothing, and a remote that cannot
// be read is a failure in git's words.
func TestBranchFetch(t *testing.T) {
	repo := testRepo(t)
	bare := bareRemote(t, repo, "origin")
	gitT(t, repo, "push", "-q", "origin", "main:gone")
	gitT(t, repo, "fetch", "-q", "origin")
	other := filepath.Join(t.TempDir(), "other")
	gitT(t, filepath.Dir(other), "clone", "-q", bare, other)
	commit(t, other, "z.txt", "z\n")
	gitT(t, other, "push", "-q", "origin", "main", "main:fresh")
	gitT(t, other, "push", "-q", "origin", "--delete", "gone")

	g := branchCalls(t)
	if err := g.Fetch(repo, NewGitStop()); err != nil {
		t.Fatal(err)
	}
	if got, want := gitT(t, repo, "rev-parse", "origin/main"), gitT(t, other, "rev-parse", "main"); got != want {
		t.Errorf("origin/main %s, want %s", got, want)
	}
	for _, ref := range []string{"refs/remotes/origin/fresh", "refs/remotes/origin/gone"} {
		if _, err := gitTry(repo, "rev-parse", "-q", "--verify", ref); err != nil {
			t.Errorf("%s missing after the fetch (gone must not be pruned)", ref)
		}
	}
	gitT(t, repo, "remote", "add", "lost", filepath.Join(t.TempDir(), "missing.git"))
	if err := g.Fetch(repo, nil); err == nil || !strings.Contains(err.Error(), "missing.git") {
		t.Errorf("a remote that cannot be read: %v", err)
	}
}

// TestBranchHeadFile is plan 59, criterion 59.8, in git: each worktree's own
// HEAD file.
func TestBranchHeadFile(t *testing.T) {
	repo := testRepo(t)
	wt := filepath.Join(filepath.Dir(repo), "wt")
	gitT(t, repo, "worktree", "add", "-q", "-b", "side", wt)
	main, err := GitCLI{}.HeadFile(repo)
	if err != nil {
		t.Fatal(err)
	}
	side, err := GitCLI{}.HeadFile(wt)
	if err != nil {
		t.Fatal(err)
	}
	if !samePath(main, filepath.Join(repo, ".git", "HEAD")) || !exists(side) || samePath(main, side) {
		t.Errorf("HEAD files: %s and %s", main, side)
	}
	if b, _ := os.ReadFile(side); strings.TrimSpace(string(b)) != "ref: refs/heads/side" {
		t.Errorf("the worktree's HEAD holds %q", b)
	}
}
