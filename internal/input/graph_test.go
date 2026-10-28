package input

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"runtime/pprof"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

// Plan 62, criteria 62.2 to 62.5 and 62.10, in git: the Git Graph's reads on
// temporary repositories (brief, amendment 64). The histories are made with
// git fast-import and fixed dates, so that no order hangs on the clock; the
// owner's repositories and configuration are never touched (TestMain).

// graphTrace records the calls of a GitCLI.
type graphTrace struct {
	mu    sync.Mutex
	calls [][]string
}

func (l *graphTrace) add(args []string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.calls = append(l.calls, slices.Clone(args))
}

// all are the calls so far, with their arguments.
func (l *graphTrace) all() [][]string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return slices.Clone(l.calls)
}

// reset forgets the calls so far.
func (l *graphTrace) reset() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.calls = nil
}

// subs are the git commands called so far, in order.
func (l *graphTrace) subs() []string {
	var s []string
	for _, c := range l.all() {
		s = append(s, subcommand(c))
	}
	return s
}

// graphViolation says why a traced call is no read of the Git Graph, "" if it
// is one: one of the five commands of design D14, the file system monitor
// off, and none of the options that write.
func graphViolation(args []string) string {
	switch subcommand(args) {
	case "for-each-ref", "rev-parse", "log", "diff-tree", "cat-file":
	default:
		return "not one of the five reads"
	}
	if !slices.Contains(args, "core.fsmonitor=false") {
		return "the file system monitor is not off"
	}
	for _, a := range args {
		if strings.HasPrefix(a, "--output") || strings.HasPrefix(a, "--force") || a == "-f" || a == "-D" || a == "--delete" {
			return "holds " + a
		}
	}
	return ""
}

// graphGit is a GitCLI whose every call is checked when the test ends
// (design D14): a read, and no write.
func graphGit(t *testing.T) (GitCLI, *graphTrace) {
	t.Helper()
	l := &graphTrace{}
	t.Cleanup(func() {
		for _, c := range l.all() {
			if why := graphViolation(c); why != "" {
				t.Errorf("git %s: %s", strings.Join(c, " "), why)
			}
		}
	})
	return GitCLI{Trace: l.add}, l
}

// graphEpoch is the first commit's author date; each commit's is a minute
// after the one before it.
const graphEpoch = 1_700_000_000

func graphWhen(mark int) int64 { return graphEpoch + 60*int64(mark) }

// graphHist is a history as a git fast-import stream: each commit has a mark,
// its number.
type graphHist struct {
	buf   bytes.Buffer
	marks int
}

// commit adds a commit to ref, whose parents are the commits of the marks
// given, the first parent first (a root has none), and gives its mark. The
// subject is the whole message.
func (h *graphHist) commit(ref, author, subject string, parents ...int) int {
	return h.commitWith(ref, author, subject, nil, parents...)
}

// commitWith is commit with files written in it, each a path and its content.
func (h *graphHist) commitWith(ref, author, subject string, files [][2]string, parents ...int) int {
	h.marks++
	when := graphWhen(h.marks)
	fmt.Fprintf(&h.buf, "commit %s\nmark :%d\nauthor %s <a@example.invalid> %d +0000\ncommitter %s <a@example.invalid> %d +0000\ndata %d\n%s\n",
		ref, h.marks, author, when, author, when, len(subject), subject)
	for i, p := range parents {
		verb := "merge"
		if i == 0 {
			verb = "from"
		}
		fmt.Fprintf(&h.buf, "%s :%d\n", verb, p)
	}
	for _, f := range files {
		fmt.Fprintf(&h.buf, "M 100644 inline %s\ndata %d\n%s\n", f[0], len(f[1]), f[1])
	}
	h.buf.WriteByte('\n')
	return h.marks
}

// run puts the history into the repository dir and gives each commit's hash
// by its mark.
func (h *graphHist) run(t *testing.T, dir string) []string {
	t.Helper()
	marks := filepath.Join(t.TempDir(), "marks")
	cmd := exec.Command("git", "-C", dir, "fast-import", "--quiet", "--export-marks="+marks)
	cmd.Stdin = &h.buf
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git fast-import: %v\n%s", err, out)
	}
	b, err := os.ReadFile(marks)
	if err != nil {
		t.Fatal(err)
	}
	hashes := make([]string, h.marks+1)
	for _, l := range lines(string(b)) {
		var n int
		var sha string
		if _, err := fmt.Sscanf(l, ":%d %s", &n, &sha); err != nil || n < 1 || n > h.marks {
			t.Fatalf("git fast-import's marks: %q", l)
		}
		hashes[n] = sha
	}
	return hashes
}

// graphBig makes a repository of total commits on main, every tenth a merge
// of a short side branch, the first commit the only root; it gives the
// repository and main's tip.
func graphBig(t *testing.T, total int) (dir, tip string) {
	t.Helper()
	gitEnv(t)
	dir = t.TempDir()
	gitT(t, dir, "init", "-q", "-b", "main")
	var h graphHist
	prev := h.commit("refs/heads/main", "Ana", "root")
	for h.marks < total {
		if h.marks%10 == 0 && h.marks+3 <= total {
			s1 := h.commit("refs/heads/side", "Bo", fmt.Sprintf("side %d a", h.marks), prev)
			s2 := h.commit("refs/heads/side", "Bo", fmt.Sprintf("side %d b", h.marks), s1)
			prev = h.commit("refs/heads/main", "Ana", fmt.Sprintf("merge %d", h.marks), prev, s2)
			continue
		}
		prev = h.commit("refs/heads/main", "Ana", fmt.Sprintf("commit %d", h.marks), prev)
	}
	return dir, h.run(t, dir)[prev]
}

// graphWant is git's own reading of the history from tips in topological
// order, one commit a line, its fields split at a character of its own: the
// reader's record proves nothing about it.
func graphWant(t *testing.T, dir string, tips ...string) []GraphCommit {
	t.Helper()
	args := append([]string{"log", "--topo-order", "--format=%H%x01%P%x01%at%x01%an%x01%s"}, tips...)
	var want []GraphCommit
	for _, l := range lines(gitRaw(t, dir, append(args, "--")...)) {
		f := strings.SplitN(l, "\x01", 5)
		if len(f) != 5 {
			t.Fatalf("git log's line %q", l)
		}
		c := GraphCommit{Hash: f[0], Author: f[3], Subject: f[4]}
		c.Time, _ = strconv.ParseInt(f[2], 10, 64)
		if f[1] != "" {
			c.Parents = strings.Fields(f[1])
		}
		want = append(want, c)
	}
	return want
}

// graphReadAll reads the whole history from tips in pages of page, which
// must end with done on the last call, and the calls be as few as the pages
// are many.
func graphReadAll(t *testing.T, g GitCLI, dir string, page int, tips ...string) []GraphCommit {
	t.Helper()
	r, err := g.GraphLog(dir, tips)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	var all []GraphCommit
	var sizes []int
	for {
		commits, done, err := r.Next(page)
		if err != nil {
			t.Fatalf("page %d: %v", len(sizes)+1, err)
		}
		if len(commits) > page || len(commits) == 0 && !done {
			t.Fatalf("page %d holds %d commits (done %v) for pages of %d", len(sizes)+1, len(commits), done, page)
		}
		sizes = append(sizes, len(commits))
		all = append(all, commits...)
		if done {
			break
		}
	}
	// Every page is full but the last, and the page of the last commit says
	// the history ended: no call for nothing.
	if want := (len(all) + page - 1) / page; len(sizes) != want && len(all) > 0 {
		t.Errorf("pages of %d took %d calls for %d commits, want %d: %v", page, len(sizes), len(all), want, sizes)
	}
	return all
}

// graphReaped fails the test unless the reader's git was waited for.
func graphReaped(t *testing.T, r GraphReader) {
	t.Helper()
	gr, ok := r.(*graphReader)
	if !ok {
		t.Fatalf("the reader is a %T", r)
	}
	select {
	case <-gr.exited:
	default:
		t.Error("git was not waited for")
		return
	}
	if gr.cmd.ProcessState == nil {
		t.Error("git has no exit status: it was not waited for")
	}
}

// noGraphGoroutines fails the test if a goroutine of a history reader is
// still there after 5 s: none may outlive git (design D4).
func noGraphGoroutines(t *testing.T) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		var buf bytes.Buffer
		_ = pprof.Lookup("goroutine").WriteTo(&buf, 2)
		var left []string
		for _, stack := range strings.Split(buf.String(), "\n\n") {
			if strings.Contains(stack, "(*graphReader).") {
				left = append(left, stack)
			}
		}
		if len(left) == 0 {
			return
		}
		if time.Now().After(deadline) {
			t.Errorf("%d goroutines of a reader left:\n%s", len(left), strings.Join(left, "\n\n"))
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// graphFolderFree fails the test if dir cannot be renamed within 10 s. On
// Windows a folder some process is working in, such as a git left running
// after the reader ended, cannot be renamed; elsewhere it always can.
func graphFolderFree(t *testing.T, dir string) {
	t.Helper()
	moved := dir + "-moved"
	deadline := time.Now().Add(10 * time.Second)
	for {
		err := os.Rename(dir, moved)
		if err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Errorf("the repository is in use after the reader ended: %v", err)
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	if err := os.Rename(moved, dir); err != nil {
		t.Fatal(err)
	}
}

// graphStuckGit puts a git that never answers first on the PATH: this test
// binary, which TestMain makes sleep 15 s without a word when ACC_FAKE_GIT is
// "child".
func graphStuckGit(t *testing.T) {
	t.Helper()
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	name := "git"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	bin := t.TempDir()
	src, err := os.Open(self)
	if err != nil {
		t.Fatal(err)
	}
	defer src.Close()
	dst, err := os.OpenFile(filepath.Join(bin, name), os.O_CREATE|os.O_WRONLY, 0o755)
	if err != nil {
		t.Fatal(err)
	}
	_, err = io.Copy(dst, src)
	if cerr := dst.Close(); err != nil || cerr != nil {
		t.Fatal(err, cerr)
	}
	t.Setenv("PATH", bin)
	t.Setenv("ACC_FAKE_GIT", "child")
}

// graphRefsRepo is a repository on main with three commits, a remote
// "origin" and every kind of reference the graph meets (plan 62, design D3):
//
//	main            c3, tracking origin/main (c1)
//	fix/login       c2, also pushed, without an upstream
//	origin/HEAD     a symbolic reference to origin/main
//	annot, nested   an annotated tag of c2, and a tag of that tag
//	light           a lightweight tag of c1
//	tree-light, tree-annot   tags of a tree
func graphRefsRepo(t *testing.T) (repo string, c1, c2, c3 string) {
	t.Helper()
	repo = testRepo(t)
	c1 = gitT(t, repo, "rev-parse", "HEAD")
	bareRemote(t, repo, "origin")
	gitT(t, repo, "remote", "set-head", "origin", "main")
	commit(t, repo, "b.txt", "two\n")
	c2 = gitT(t, repo, "rev-parse", "HEAD")
	gitT(t, repo, "branch", "fix/login")
	gitT(t, repo, "push", "-q", "origin", "fix/login")
	commit(t, repo, "c.txt", "three\n")
	c3 = gitT(t, repo, "rev-parse", "HEAD")
	gitT(t, repo, "tag", "light", c1)
	gitT(t, repo, "tag", "-a", "-m", "annotated", "annot", c2)
	gitT(t, repo, "tag", "-a", "-m", "nested", "nested", "annot")
	gitT(t, repo, "tag", "tree-light", "HEAD^{tree}")
	gitT(t, repo, "tag", "-a", "-m", "tree", "tree-annot", "HEAD^{tree}")
	all := gitT(t, repo, "for-each-ref", "--format=%(refname)")
	for _, name := range []string{"refs/remotes/origin/HEAD", "refs/tags/tree-light", "refs/tags/tree-annot"} {
		if !strings.Contains(all, name) {
			t.Fatalf("the fixture lacks %s:\n%s", name, all)
		}
	}
	return repo, c1, c2, c3
}

// TestGraphRefs is plan 62's criteria 62.2 and 62.3 for the references and
// the tips of each scope (designs D3 and D4), on one repository that changes
// from step to step. References: local and remote branches, a lightweight and
// an annotated tag peeled to their commits, a tag of a tag, HEAD's branch, its
// upstream and its commit; a remote's HEAD and the tags of a tree left out; a
// detached HEAD; a branch with no commit yet; a repository with none; a folder
// outside git. Tips: Auto with an upstream, without one and with one whose
// branch was deleted on the remote; All, each commit once and HEAD's first;
// chosen references, one of them gone.
func TestGraphRefs(t *testing.T) {
	t.Parallel()
	repo, c1, c2, c3 := graphRefsRepo(t)
	g, calls := graphGit(t)
	read := func() GraphRefs {
		t.Helper()
		refs, err := g.GraphRefs(repo)
		if err != nil {
			t.Fatal(err)
		}
		return refs
	}
	tips := func(name string, s GraphScope, wantTips, wantGone []string) {
		t.Helper()
		got, gone := read().Tips(s)
		if !slices.Equal(got, wantTips) || !slices.Equal(gone, wantGone) {
			t.Errorf("%s: tips %q gone %q, want %q and %q", name, got, gone, wantTips, wantGone)
		}
	}
	refs := []GraphRef{
		{"refs/heads/fix/login", c2},
		{"refs/heads/main", c3},
		{"refs/remotes/origin/fix/login", c2},
		{"refs/remotes/origin/main", c1},
		{"refs/tags/annot", c2},
		{"refs/tags/light", c1},
		{"refs/tags/nested", c2},
	}

	// On main.
	want := GraphRefs{Head: c3, Branch: "refs/heads/main", Upstream: "refs/remotes/origin/main", Refs: refs}
	if got := read(); !reflect.DeepEqual(got, want) {
		t.Errorf("on main:\n got %+v\nwant %+v", got, want)
	}
	// HEAD on a branch is known from the one for-each-ref.
	if subs := calls.subs(); !reflect.DeepEqual(subs, []string{"for-each-ref"}) {
		t.Errorf("calls %q, want one for-each-ref", subs)
	}
	call := strings.Join(calls.all()[0], " ")
	for _, w := range []string{"-c core.fsmonitor=false", "for-each-ref", "refs/heads refs/remotes refs/tags"} {
		if !strings.Contains(call, w) {
			t.Errorf("the call %q lacks %q", call, w)
		}
	}
	tips("Auto with an upstream", GraphScope{}, []string{c3, c1}, nil)
	tips("All, each once", GraphScope{All: true}, []string{c3, c2, c1}, nil)
	tips("chosen, one gone", GraphScope{Refs: []string{"refs/tags/annot", "refs/heads/main", "refs/heads/nope", "refs/tags/nested", "refs/remotes/origin/main"}},
		[]string{c2, c3, c1}, []string{"refs/heads/nope"})
	tips("chosen, all gone", GraphScope{Refs: []string{"refs/heads/nope"}}, nil, []string{"refs/heads/nope"})

	// A detached HEAD: its commit, on no branch, the references as they were;
	// HEAD's commit is the first tip.
	gitT(t, repo, "checkout", "-q", "--detach", c2)
	calls.reset()
	want = GraphRefs{Head: c2, Refs: refs}
	if got := read(); !reflect.DeepEqual(got, want) {
		t.Errorf("detached:\n got %+v\nwant %+v", got, want)
	}
	if subs := calls.subs(); !reflect.DeepEqual(subs, []string{"for-each-ref", "rev-parse"}) {
		t.Errorf("calls %q, want a for-each-ref and a rev-parse of HEAD", subs)
	} else if last := strings.Join(calls.all()[1], " "); !strings.HasSuffix(last, "rev-parse --verify --quiet HEAD^{commit}") {
		t.Errorf("the rev-parse: %q", last)
	}
	gitT(t, repo, "checkout", "-q", "--detach", c1)
	tips("All, detached", GraphScope{All: true}, []string{c1, c2, c3}, nil)
	tips("Auto, detached", GraphScope{}, []string{c1}, nil)

	// A branch without an upstream: HEAD alone.
	gitT(t, repo, "switch", "-q", "fix/login")
	if got := read(); got.Branch != "refs/heads/fix/login" || got.Upstream != "" || got.Head != c2 {
		t.Errorf("fix/login: %+v", got)
	}
	tips("Auto without an upstream", GraphScope{}, []string{c2}, nil)

	// An upstream whose branch was deleted on the remote and pruned here: the
	// configuration stays, the reference is gone, Auto is HEAD alone.
	gitT(t, repo, "switch", "-q", "-c", "topic")
	gitT(t, repo, "push", "-q", "-u", "origin", "topic")
	tips("Auto with an upstream", GraphScope{}, []string{c2}, nil) // topic and origin/topic are both c2
	gitT(t, repo, "push", "-q", "origin", "--delete", "topic")
	gitT(t, repo, "fetch", "-q", "--prune", "origin")
	got := read()
	if got.Branch != "refs/heads/topic" || got.Upstream != "refs/remotes/origin/topic" {
		t.Errorf("the branch and its upstream, gone: %q, %q", got.Branch, got.Upstream)
	}
	for _, r := range got.Refs {
		if r.Name == got.Upstream {
			t.Errorf("the upstream %s is listed though gone", r.Name)
		}
	}
	tips("Auto with an upstream gone", GraphScope{}, []string{c2}, nil)

	// A branch with no commit yet, the others still there: All is their
	// commits alone.
	gitT(t, repo, "checkout", "-q", "--orphan", "fresh")
	if got := read(); got.Head != "" || got.Branch != "" || got.Upstream != "" || len(got.Refs) != len(refs)+1 { // and topic
		t.Errorf("unborn branch: %+v", got)
	}
	tips("All, no commit yet", GraphScope{All: true}, []string{c2, c3, c1}, nil)
	tips("Auto, no commit yet", GraphScope{}, nil, nil)

	// A repository with no commit at all.
	empty := t.TempDir()
	gitT(t, empty, "init", "-q", "-b", "main")
	if got, err := g.GraphRefs(empty); err != nil || !reflect.DeepEqual(got, GraphRefs{}) {
		t.Errorf("no commit: %+v, %v", got, err)
	}

	// A folder outside git.
	outside := t.TempDir()
	var nr NotRepoError
	if _, err := g.GraphRefs(outside); !errors.As(err, &nr) || nr.Dir != outside {
		t.Errorf("outside git: %v", err)
	}
}

// TestGraphGitMissing: without git every read says so, but no tips is
// nothing to read, and no process (design D4).
func TestGraphGitMissing(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	g := GitCLI{}
	dir := t.TempDir()
	hash := strings.Repeat("a", 40)
	_, e1 := g.GraphRefs(dir)
	_, e2 := g.GraphLog(dir, []string{hash})
	_, e3 := g.CommitFiles(dir, hash, "")
	_, e4 := g.CommitSides(dir, hash, hash, CommitFile{Status: 'M', Path: "a.txt"})
	for i, err := range []error{e1, e2, e3, e4} {
		if !errors.Is(err, ErrNoGit) {
			t.Errorf("read %d: %v, want ErrNoGit", i+1, err)
		}
	}
	r, err := g.GraphLog(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	if commits, done, err := r.Next(10); commits != nil || !done || err != nil {
		t.Errorf("no tips: %v, %v, %v", commits, done, err)
	}
}

// graphTopology makes the history of the lanes' cases (design D9): two
// branches and the merge of them, a merge of three parents, and two roots,
// the second an unrelated history merged in. It has 14 commits, each with
// the subject "commit <name>"; c gives their hashes by name.
func graphTopology(t *testing.T) (dir string, c map[string]string) {
	t.Helper()
	gitEnv(t)
	dir = t.TempDir()
	gitT(t, dir, "init", "-q", "-b", "main")
	var h graphHist
	marks := map[string]int{}
	add := func(name, ref, author string, parents ...string) {
		var ps []int
		for _, p := range parents {
			ps = append(ps, marks[p])
		}
		marks[name] = h.commit(ref, author, "commit "+name, ps...)
	}
	add("r1", "refs/heads/main", "Ana")
	add("a1", "refs/heads/main", "Ana", "r1")
	add("b1", "refs/heads/feature", "Bo", "a1")
	add("a2", "refs/heads/main", "Ana", "a1")
	add("b2", "refs/heads/feature", "Bo", "b1")
	add("m1", "refs/heads/main", "Ana", "a2", "b2")
	add("c1", "refs/heads/topic", "Cy", "a2")
	add("a3", "refs/heads/main", "Ana", "m1")
	add("m2", "refs/heads/main", "Ana", "a3", "b2", "c1")
	add("a4", "refs/heads/main", "Ana", "m2")
	add("r2", "refs/heads/other", "Di")
	add("o1", "refs/heads/other", "Di", "r2")
	add("m3", "refs/heads/main", "Ana", "a4", "o1")
	add("a5", "refs/heads/main", "Ana", "m3")
	hashes := h.run(t, dir)
	c = map[string]string{}
	for name, m := range marks {
		c[name] = hashes[m]
	}
	return dir, c
}

// graphTopological fails the test unless every commit of the history is
// there once, before its parents, all of which are there.
func graphTopological(t *testing.T, history []GraphCommit) {
	t.Helper()
	at := map[string]int{}
	for i, c := range history {
		if _, twice := at[c.Hash]; twice {
			t.Errorf("%s is read twice", c.Hash)
		}
		at[c.Hash] = i
	}
	for i, c := range history {
		for _, p := range c.Parents {
			if j, ok := at[p]; !ok || j <= i {
				t.Errorf("parent %s of %s (row %d) is at row %d (read: %v)", p, c.Hash, i, j, ok)
			}
		}
	}
}

// TestGraphLogOrder is plan 62's criterion 62.2 for the history (design D4):
// the commits of the tips in git log's topological order, each with its
// parents, author, time and subject, in pages of any size, on a history with
// two branches, merges, a merge of three parents and two roots; and the call
// is the log of the design, a read.
func TestGraphLogOrder(t *testing.T) {
	t.Parallel()
	dir, c := graphTopology(t)
	g, calls := graphGit(t)
	for _, tc := range []struct {
		name string
		tips []string
	}{
		{"main", []string{c["a5"]}},
		{"two branches", []string{c["b2"], c["c1"]}},
		{"every branch", []string{c["a5"], c["b2"], c["c1"], c["o1"]}},
		{"a root", []string{c["r2"]}},
	} {
		want := graphWant(t, dir, tc.tips...)
		for _, page := range []int{1, 3, 100} {
			if got := graphReadAll(t, g, dir, page, tc.tips...); !reflect.DeepEqual(got, want) {
				t.Errorf("%s in pages of %d:\n got %+v\nwant %+v", tc.name, page, got, want)
			}
		}
	}

	// The shapes themselves, not git's word for them.
	all := graphReadAll(t, g, dir, 4, c["a5"])
	graphTopological(t, all)
	var roots []string
	for _, commit := range all {
		if len(commit.Parents) == 0 {
			roots = append(roots, commit.Hash)
		}
		switch commit.Hash {
		case c["m2"]:
			if want := []string{c["a3"], c["b2"], c["c1"]}; !slices.Equal(commit.Parents, want) {
				t.Errorf("the merge of three has parents %q, want %q", commit.Parents, want)
			}
		case c["m3"]:
			if want := []string{c["a4"], c["o1"]}; !slices.Equal(commit.Parents, want) {
				t.Errorf("the unrelated history's merge has parents %q, want %q", commit.Parents, want)
			}
		case c["a1"]:
			if commit.Subject != "commit a1" || commit.Author != "Ana" || commit.Time != graphWhen(2) {
				t.Errorf("a1: %+v", commit)
			}
		}
	}
	slices.Sort(roots)
	wantRoots := []string{c["r1"], c["r2"]}
	slices.Sort(wantRoots)
	if len(all) != 14 || !slices.Equal(roots, wantRoots) {
		t.Errorf("%d commits with roots %q, want 14 and %q", len(all), roots, wantRoots)
	}

	// Every call: git log, topological, no signature check, NUL terminated,
	// the format, the tips, then "--"; the file system monitor off.
	format := "--format=%H%x1f%P%x1f%at%x1f%an%x1f%s"
	logs := 0
	for _, call := range calls.all() {
		if subcommand(call) != "log" {
			t.Errorf("a call that is no log: %q", call)
			continue
		}
		logs++
		i := slices.Index(call, "log")
		if head := call[i : i+5]; !slices.Equal(head, []string{"log", "--topo-order", "--no-show-signature", "-z", format}) {
			t.Errorf("the log's options: %q", head)
		}
		if call[len(call)-1] != "--" || !slices.Contains(call[:i], "core.fsmonitor=false") {
			t.Errorf("the log's end or its monitor: %q", call)
		}
		for _, tip := range call[i+5 : len(call)-1] {
			if !graphHash(tip) {
				t.Errorf("the tip %q of %q is no hash", tip, call)
			}
		}
	}
	if logs != 4*3+1 { // a git log a read: 4 sets of tips in 3 page sizes, and one more
		t.Errorf("%d git logs, want 13", logs)
	}
}

// TestGraphLogDone is plan 62's criterion 62.2 for the end of a read (design
// D4): done comes with the page that holds the last commit, also when the
// pages divide the history exactly, git is waited for with it, and nothing
// is left, with no Close.
func TestGraphLogDone(t *testing.T) {
	dir, c := graphTopology(t)
	g, _ := graphGit(t)
	for _, page := range []int{14, 7, 5, 1} { // 14 commits: the first two divide them
		if got := graphReadAll(t, g, dir, page, c["a5"]); len(got) != 14 {
			t.Errorf("pages of %d: %d commits, want 14", page, len(got))
		}
	}

	r, err := g.GraphLog(dir, []string{c["a5"]})
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	if commits, done, err := r.Next(0); commits != nil || done || err != nil {
		t.Errorf("a page of nothing: %v, %v, %v", commits, done, err)
	}
	commits, done, err := r.Next(14)
	if len(commits) != 14 || !done || err != nil {
		t.Fatalf("a page of all: %d commits, done %v, %v", len(commits), done, err)
	}
	graphReaped(t, r)
	noGraphGoroutines(t)
	if commits, done, err = r.Next(5); commits != nil || !done || err != nil {
		t.Errorf("after the end: %v, %v, %v", commits, done, err)
	}
	r.Close()
	if _, _, err = r.Next(5); !errors.Is(err, ErrGraphClosed) {
		t.Errorf("after Close: %v", err)
	}
}

// TestGraphLogConcurrentNext: Next called from several goroutines takes the
// pages one at a time, none lost and none read twice (design D4).
func TestGraphLogConcurrentNext(t *testing.T) {
	dir, c := graphTopology(t)
	g, _ := graphGit(t)
	r, err := g.GraphLog(dir, []string{c["a5"]})
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	var mu sync.Mutex
	seen := map[string]int{}
	var wg sync.WaitGroup
	for range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				commits, done, err := r.Next(1)
				if err != nil {
					t.Error(err)
					return
				}
				mu.Lock()
				for _, commit := range commits {
					seen[commit.Hash]++
				}
				mu.Unlock()
				if done {
					return
				}
			}
		}()
	}
	wg.Wait()
	if len(seen) != 14 {
		t.Errorf("%d commits read, want 14", len(seen))
	}
	for hash, n := range seen {
		if n != 1 {
			t.Errorf("%s read %d times", hash, n)
		}
	}
	graphReaped(t, r)
}

// TestGraphLogBetweenPages is plan 62's criterion 62.2 for the tips fixed
// (design D4): a commit made on the branch while git waits on its pipe leaves
// the rest of the read as it was.
func TestGraphLogBetweenPages(t *testing.T) {
	t.Parallel()
	dir, tip := graphBig(t, 8000)
	g, _ := graphGit(t)
	want := graphWant(t, dir, tip)
	if len(want) != 8000 {
		t.Fatalf("%d commits made", len(want))
	}
	r, err := g.GraphLog(dir, []string{tip})
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	got, done, err := r.Next(100)
	if err != nil || done || len(got) != 100 {
		t.Fatalf("the first page: %d commits, done %v, %v", len(got), done, err)
	}
	// The premise: more than the pipes hold, so that git still writes.
	select {
	case <-r.(*graphReader).exited:
		t.Fatal("git had ended: nothing is proved")
	default:
	}
	gitT(t, dir, "commit", "-q", "--allow-empty", "-m", "late")
	if now := gitT(t, dir, "rev-parse", "main"); now == tip {
		t.Fatal("main did not move")
	}
	for !done {
		var page []GraphCommit
		if page, done, err = r.Next(250); err != nil {
			t.Fatal(err)
		}
		got = append(got, page...)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("%d commits read, want the %d of the tip, unchanged", len(got), len(want))
	}
}

// TestGraphLogText is plan 62's criterion 62.2 for what a commit holds
// (design D4): a subject with the unit separator in it and non-ASCII text
// kept whole, an empty one, a long one, authors in UTF-8 and in Latin-1,
// whose bytes that are no UTF-8 become U+FFFD.
func TestGraphLogText(t *testing.T) {
	t.Parallel()
	gitEnv(t)
	dir := t.TempDir()
	gitT(t, dir, "init", "-q", "-b", "main")
	long := strings.Repeat("très long ", 10_000) // 110,000 bytes: more than the reader's buffer
	type commitText struct{ author, subject, wantAuthor, wantSubject string }
	texts := []commitText{
		{"Ana", "plain", "Ana", "plain"},
		{"José Núñez", "tab\x1fsep — naïve ☃ 日本語 😀", "José Núñez", "tab\x1fsep — naïve ☃ 日本語 😀"},
		{"Bo", "", "Bo", ""},
		{"Cy", long, "Cy", long[:len(long)-1]}, // git trims the subject's end
		{"Jos\xe9", "a\x1f\x1f\x1fb", "Jos\ufffd", "a\x1f\x1f\x1fb"},
		{"Di", "caf\xe9", "Di", "caf\ufffd"},
	}
	var h graphHist
	prev := 0
	for _, tx := range texts {
		var parents []int
		if prev != 0 {
			parents = []int{prev}
		}
		prev = h.commit("refs/heads/main", tx.author, tx.subject, parents...)
	}
	hashes := h.run(t, dir)

	// Newest first: the history is a line.
	var want []GraphCommit
	for i := len(texts) - 1; i >= 0; i-- {
		c := GraphCommit{Hash: hashes[i+1], Author: texts[i].wantAuthor, Time: graphWhen(i + 1), Subject: texts[i].wantSubject}
		if i > 0 {
			c.Parents = []string{hashes[i]}
		}
		want = append(want, c)
	}
	g, _ := graphGit(t)
	for _, page := range []int{1, 2, 100} {
		got := graphReadAll(t, g, dir, page, hashes[len(texts)])
		if len(got) != len(want) {
			t.Errorf("pages of %d: %d commits, want %d", page, len(got), len(want))
			continue
		}
		for i := range want {
			if !reflect.DeepEqual(got[i], want[i]) {
				t.Errorf("pages of %d, commit %d:\n got %.200q\nwant %.200q", page, i, fmt.Sprintf("%+v", got[i]), fmt.Sprintf("%+v", want[i]))
			}
		}
	}
}

// TestGraphLogCut: a record past graphRecordMax is cut in its subject, which
// ends in "…", and the commits after it are read as they are (plan 48's rule:
// no unbounded growth).
func TestGraphLogCut(t *testing.T) {
	gitEnv(t)
	defer func(n int) { graphRecordMax = n }(graphRecordMax)
	graphRecordMax = 4096
	dir := t.TempDir()
	gitT(t, dir, "init", "-q", "-b", "main")
	var h graphHist
	a := h.commit("refs/heads/main", "Ana", "first")
	b := h.commit("refs/heads/main", "Ana", strings.Repeat("é", 10_000), a)
	c := h.commit("refs/heads/main", "Ana", "third", b)
	hashes := h.run(t, dir)
	g, _ := graphGit(t)
	got := graphReadAll(t, g, dir, 10, hashes[c])
	if len(got) != 3 || got[0].Subject != "third" || got[2].Subject != "first" || got[0].Hash != hashes[c] || got[2].Hash != hashes[a] {
		t.Fatalf("the commits around the cut one: %+v", got)
	}
	cut := got[1].Subject
	if !strings.HasSuffix(cut, "…") || !strings.HasPrefix(cut, "éééé") || len(cut) > graphRecordMax || len(cut) < 3000 {
		t.Errorf("the cut subject (%d bytes) %.20q…%.20q", len(cut), cut, cut[max(0, len(cut)-20):])
	}
}

// TestGraphLogFailures is plan 62's criterion 62.2 for git's failure (design
// D4): in git's words, the same error again, git waited for with no Close;
// and what git would take for an option never reaches it (design D14).
func TestGraphLogFailures(t *testing.T) {
	dir, c := graphTopology(t)
	// A commit whose parent the repository does not have: a history git
	// cannot walk.
	tree := gitT(t, dir, "rev-parse", c["a5"]+"^{tree}")
	crafted := filepath.Join(t.TempDir(), "commit.txt")
	write(t, crafted, "tree "+tree+"\nparent "+strings.Repeat("2", 40)+"\nauthor A <a@example.invalid> 1700000000 +0000\ncommitter A <a@example.invalid> 1700000000 +0000\n\ndangling\n")
	broken := gitT(t, dir, "hash-object", "-t", "commit", "-w", crafted)

	g, calls := graphGit(t)
	missing := strings.Repeat("1", 40)
	for _, tc := range []struct {
		name, dir string
		tips      []string
		want      string
	}{
		{"a tip git does not have", dir, []string{missing}, "bad object " + missing},
		{"a good tip and a missing one", dir, []string{c["a5"], missing}, "bad object " + missing},
		{"a history git cannot walk", dir, []string{broken}, "Could not read " + strings.Repeat("2", 40)},
		{"a folder outside git", t.TempDir(), []string{c["a5"]}, "not a git repository"},
	} {
		r, err := g.GraphLog(tc.dir, tc.tips)
		if err != nil {
			t.Errorf("%s: %v", tc.name, err)
			continue
		}
		commits, done, err1 := r.Next(10)
		var ge *gitError
		if commits != nil || done || !errors.As(err1, &ge) || ge.code != 128 || !strings.Contains(err1.Error(), tc.want) {
			t.Errorf("%s: %v, %v, %v; want git's refusal with %q", tc.name, commits, done, err1, tc.want)
		}
		// The failure stays, and git is gone without a Close.
		if _, _, err2 := r.Next(10); err2 != err1 {
			t.Errorf("%s: the second Next gave %v, want the first error", tc.name, err2)
		}
		graphReaped(t, r)
		noGraphGoroutines(t)
		r.Close()
		if _, _, err3 := r.Next(10); err3 != err1 {
			t.Errorf("%s: after Close %v, want the first error", tc.name, err3)
		}
	}

	before := len(calls.all())
	out := filepath.Join(t.TempDir(), "out.txt")
	for _, tip := range []string{"--output=" + out, "-1", ""} {
		if r, err := g.GraphLog(dir, []string{c["a5"], tip}); err == nil {
			r.Close()
			t.Errorf("the tip %q was taken", tip)
		}
	}
	if len(calls.all()) != before || exists(out) {
		t.Errorf("git ran for a tip that is no hash, or wrote %s", out)
	}
}

// TestGraphLogNoTips: no tips is a history that is done, with no git call and
// no process (design D4).
func TestGraphLogNoTips(t *testing.T) {
	g, calls := graphGit(t)
	for _, tips := range [][]string{nil, {}} {
		r, err := g.GraphLog(t.TempDir(), tips)
		if err != nil {
			t.Fatal(err)
		}
		for range 2 {
			if commits, done, err := r.Next(200); commits != nil || !done || err != nil {
				t.Errorf("no tips: %v, %v, %v", commits, done, err)
			}
		}
		r.Close()
		r.Close()
		if _, _, err := r.Next(200); !errors.Is(err, ErrGraphClosed) {
			t.Errorf("after Close: %v", err)
		}
	}
	if n := len(calls.all()); n != 0 {
		t.Errorf("%d git calls for no tips", n)
	}
	noGraphGoroutines(t)
}

// TestGraphLogClose is plan 62's criterion 62.2 for the end of a read (design
// D4): Close at any time stops git and waits for it, leaves no goroutine, is
// safe more than once and from other goroutines, and a Next that waits is let
// go with the error that says so.
func TestGraphLogClose(t *testing.T) {
	t.Run("before any page", func(t *testing.T) {
		dir, c := graphTopology(t)
		r, err := GitCLI{}.GraphLog(dir, []string{c["a5"]})
		if err != nil {
			t.Fatal(err)
		}
		r.Close()
		graphReaped(t, r)
		noGraphGoroutines(t)
		if commits, done, err := r.Next(5); commits != nil || done || !errors.Is(err, ErrGraphClosed) {
			t.Errorf("after Close: %v, %v, %v", commits, done, err)
		}
	})

	t.Run("in the middle of a long history", func(t *testing.T) {
		dir, tip := graphBig(t, 8000)
		r, err := GitCLI{}.GraphLog(dir, []string{tip})
		if err != nil {
			t.Fatal(err)
		}
		if commits, done, err := r.Next(100); len(commits) != 100 || done || err != nil {
			t.Fatalf("the first page: %d commits, %v, %v", len(commits), done, err)
		}
		select {
		case <-r.(*graphReader).exited:
			t.Fatal("git had ended: nothing is proved")
		default:
		}
		start := time.Now()
		var wg sync.WaitGroup
		for range 4 { // together, and each returns once git is gone
			wg.Add(1)
			go func() {
				defer wg.Done()
				r.Close()
				graphReaped(t, r)
			}()
		}
		closed := make(chan struct{})
		go func() { wg.Wait(); close(closed) }()
		select {
		case <-closed:
		case <-time.After(20 * time.Second):
			t.Fatal("Close does not return")
		}
		// Git is stopped, not left to give up: the pipe is closed at once, where
		// a pipe left open keeps Wait for gitWaitDelay, 2 s.
		d := time.Since(start)
		t.Logf("Close took %v", d.Round(time.Millisecond))
		if d > 1500*time.Millisecond {
			t.Errorf("Close took %v", d)
		}
		r.Close()
		if commits, done, err := r.Next(100); commits != nil || done || !errors.Is(err, ErrGraphClosed) {
			t.Errorf("after Close: %v, %v, %v", commits, done, err)
		}
		noGraphGoroutines(t)
		graphFolderFree(t, dir)
	})

	t.Run("while a Next waits", func(t *testing.T) {
		graphStuckGit(t)
		r, err := GitCLI{}.GraphLog(t.TempDir(), []string{strings.Repeat("a", 40)})
		if err != nil {
			t.Fatal(err)
		}
		defer r.Close()
		type page struct {
			commits []GraphCommit
			done    bool
			err     error
		}
		got := make(chan page, 1)
		go func() {
			c, d, e := r.Next(10)
			got <- page{c, d, e}
		}()
		time.Sleep(300 * time.Millisecond)
		select {
		case p := <-got:
			t.Fatalf("Next returned before Close: %+v", p)
		default:
		}
		start := time.Now()
		r.Close()
		select {
		case p := <-got:
			if p.commits != nil || p.done || !errors.Is(p.err, ErrGraphClosed) {
				t.Errorf("the Next that waited: %+v", p)
			}
		case <-time.After(10 * time.Second):
			t.Fatal("Next still waits after Close")
		}
		d := time.Since(start)
		t.Logf("Close took %v", d.Round(time.Millisecond))
		if d > 5*time.Second { // git sleeps 15 s
			t.Errorf("Close took %v", d)
		}
		graphReaped(t, r)
		noGraphGoroutines(t)
	})
}

// TestGraphLogTimeout is plan 62's criterion 62.2 for a git that does not
// answer (design D4): a page that takes longer than graphPageLimit ends the
// reader with an error that is context.DeadlineExceeded, git killed and
// waited for, and the same error afterwards.
func TestGraphLogTimeout(t *testing.T) {
	defer func(d time.Duration) { graphPageLimit = d }(graphPageLimit)

	t.Run("a limit of 1 ns", func(t *testing.T) {
		dir, c := graphTopology(t)
		graphPageLimit = time.Nanosecond
		r, err := GitCLI{}.GraphLog(dir, []string{c["a5"]})
		if err != nil {
			t.Fatal(err)
		}
		defer r.Close()
		commits, done, err1 := r.Next(5)
		var te *timeoutError
		if commits != nil || done || !errors.Is(err1, context.DeadlineExceeded) || !errors.As(err1, &te) {
			t.Fatalf("the late page: %v, %v, %v", commits, done, err1)
		}
		graphReaped(t, r)
		noGraphGoroutines(t)
		graphPageLimit = time.Minute
		if _, _, err2 := r.Next(5); err2 != err1 {
			t.Errorf("the next page gave %v, want %v", err2, err1)
		}
	})

	t.Run("a git that is stuck", func(t *testing.T) {
		graphStuckGit(t)
		graphPageLimit = 300 * time.Millisecond
		r, err := GitCLI{}.GraphLog(t.TempDir(), []string{strings.Repeat("a", 40)})
		if err != nil {
			t.Fatal(err)
		}
		defer r.Close()
		start := time.Now()
		commits, done, err := r.Next(5)
		took := time.Since(start)
		if commits != nil || done || !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("the stuck git: %v, %v, %v", commits, done, err)
		}
		if took < 300*time.Millisecond || took > 5*time.Second { // git sleeps 15 s
			t.Errorf("Next took %v for a limit of 300 ms", took)
		}
		graphReaped(t, r)
		noGraphGoroutines(t)
	})
}

// TestGraphLogLarge is plan 62's criterion 62.10 for the read (design D15):
// 50,000 commits made with git fast-import, the first page of 200 timed, then
// every page to the end: none twice, in git's order, the last the root; a
// commit made meanwhile changes nothing. The time is logged, not judged.
func TestGraphLogLarge(t *testing.T) {
	if testing.Short() {
		t.Skip("making 50,000 commits takes seconds")
	}
	made := time.Now()
	dir, tip := graphBig(t, 50_000)
	t.Logf("50,000 commits made in %v", time.Since(made).Round(time.Millisecond))
	g, _ := graphGit(t)
	var want []string
	for _, c := range graphWant(t, dir, tip) {
		want = append(want, c.Hash)
	}
	if len(want) != 50_000 {
		t.Fatalf("%d commits made", len(want))
	}

	start := time.Now()
	r, err := g.GraphLog(dir, []string{tip})
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	page, done, err := r.Next(200)
	first := time.Since(start)
	if err != nil || done || len(page) != 200 {
		t.Fatalf("the first page: %d commits, done %v, %v", len(page), done, err)
	}
	t.Logf("git log started and the first page of 200 read in %v", first.Round(time.Millisecond))

	gitT(t, dir, "commit", "-q", "--allow-empty", "-m", "late")
	got := append([]GraphCommit(nil), page...)
	pages := 1
	for !done {
		if page, done, err = r.Next(200); err != nil {
			t.Fatalf("page %d: %v", pages+1, err)
		}
		got = append(got, page...)
		pages++
	}
	t.Logf("%d commits in %d pages, read to the end %v after the start", len(got), pages, time.Since(start).Round(time.Millisecond))
	seen := make(map[string]bool, len(got))
	for i, c := range got {
		if seen[c.Hash] {
			t.Fatalf("%s is read twice (row %d)", c.Hash, i)
		}
		seen[c.Hash] = true
		if i < len(want) && c.Hash != want[i] {
			t.Fatalf("row %d is %s, git log's is %s", i, c.Hash, want[i])
		}
	}
	if len(got) != 50_000 || len(got[len(got)-1].Parents) != 0 || got[len(got)-1].Subject != "root" {
		t.Errorf("%d commits, the last %+v; want 50,000 and the root", len(got), got[len(got)-1])
	}
}

const (
	graphMod0      = "a1\na2\na3\na4\na5\na6\na7\na8\n"
	graphMod1      = "a1\nA2\na3\na4\na5\na6\na7\na8\na9\n"
	graphModSide   = "a1\na2\na3\na4\nA5\na6\na7\na8\n"
	graphModMerged = "a1\nA2\na3\na4\nA5\na6\na7\na8\na9\n"
	graphRen0      = "b1\nb2\nb3\nb4\nb5\nb6\nb7\nb8\n"
	graphRen1      = "b1\nb2\nb3\nb4\nB5\nb6\nb7\nb8\n"
)

// graphFiles is a repository whose commits change files in every way the
// graph lists (design D5): base is the first commit; top, on main, changes
// the files of base; side, from base, changes two others; merge, of top and
// side, is made without a working tree.
type graphFiles struct {
	dir                    string
	base, top, side, merge string
}

// graphFilesRepo makes it. In top: .env, mod.txt and a file in a folder with
// a space and accents are modified, add.txt is added, del.txt deleted,
// ren-old.txt renamed to "ren new.txt" and edited, bin.dat (binary) changed
// and tc.txt made a link, an index entry of mode 120000 with no link on disk.
func graphFilesRepo(t *testing.T) graphFiles {
	t.Helper()
	var f graphFiles
	f.dir = srcRepo(t)
	put := func(rel, data string) { writeAt(t, f.dir, rel, data) }
	put(".env", "SECRET=1\n")
	put("bin.dat", "x\x00y")
	put("del.txt", "del\n")
	put("mod.txt", graphMod0)
	put("ren-old.txt", graphRen0)
	put("sub/dir/é ünï.txt", "u\n")
	put("tc.txt", "one\ntwo\n")
	gitT(t, f.dir, "add", "-A")
	gitT(t, f.dir, "commit", "-q", "-m", "base")
	f.base = gitT(t, f.dir, "rev-parse", "HEAD")

	gitT(t, f.dir, "switch", "-q", "-c", "side")
	put("mod.txt", graphModSide)
	put("side.txt", "side\n")
	gitT(t, f.dir, "add", "-A")
	gitT(t, f.dir, "commit", "-q", "-m", "side")
	f.side = gitT(t, f.dir, "rev-parse", "HEAD")
	gitT(t, f.dir, "switch", "-q", "main")

	put("mod.txt", graphMod1)
	gitT(t, f.dir, "mv", "ren-old.txt", "ren new.txt")
	put("ren new.txt", graphRen1)
	gitT(t, f.dir, "rm", "-q", "del.txt")
	put("add.txt", "added\n1\n2\n")
	put("bin.dat", "x\x00y\x00z")
	put(".env", "SECRET=2\n")
	put("sub/dir/é ünï.txt", "u2\n")
	gitT(t, f.dir, "add", "-A")
	target := filepath.Join(t.TempDir(), "target")
	write(t, target, "some/target")
	blob := gitT(t, f.dir, "hash-object", "-w", target)
	gitT(t, f.dir, "update-index", "--add", "--cacheinfo", "120000,"+blob+",tc.txt")
	gitT(t, f.dir, "commit", "-q", "-m", "top")
	f.top = gitT(t, f.dir, "rev-parse", "HEAD")

	tree := gitT(t, f.dir, "merge-tree", "--write-tree", f.top, f.side)
	f.merge = gitT(t, f.dir, "commit-tree", tree, "-p", f.top, "-p", f.side, "-m", "merge")
	gitT(t, f.dir, "update-ref", "refs/heads/main", f.merge)
	return f
}

// TestGraphCommit is plan 62's criteria 62.4 and 62.5 on one repository
// (graphFilesRepo): a commit's files, a file's sides, and the reads' write of
// nothing.
func TestGraphCommit(t *testing.T) {
	t.Parallel()
	f := graphFilesRepo(t)
	t.Run("reads", func(t *testing.T) {
		t.Run("files", func(t *testing.T) { t.Parallel(); checkCommitFiles(t, f) })
		t.Run("sides", func(t *testing.T) { t.Parallel(); checkCommitSides(t, f) })
	})
	t.Run("writes nothing", func(t *testing.T) { checkReadsWriteNothing(t, f) })
}

// checkCommitFiles is plan 62's criterion 62.4 (design D5): a commit's files
// against its parent with their statuses, counts and renames; a binary file;
// a type change; the root against nothing; a merge against its first parent;
// paths from the top level, as git gives them; git's refusal of a commit it
// does not have; the call a read.
func checkCommitFiles(t *testing.T, f graphFiles) {
	g, calls := graphGit(t)

	for _, tc := range []struct {
		name           string
		commit, parent string
		want           []CommitFile
	}{
		{"a commit against its parent", f.top, f.base, []CommitFile{
			{Status: 'M', Path: ".env", Added: 1, Removed: 1},
			{Status: 'A', Path: "add.txt", Added: 3},
			{Status: 'M', Path: "bin.dat", Binary: true},
			{Status: 'D', Path: "del.txt", Removed: 1},
			{Status: 'M', Path: "mod.txt", Added: 2, Removed: 1},
			{Status: 'R', Path: "ren new.txt", Orig: "ren-old.txt", Added: 1, Removed: 1},
			{Status: 'M', Path: "sub/dir/é ünï.txt", Added: 1, Removed: 1},
			{Status: 'T', Path: "tc.txt", Added: 1, Removed: 2},
		}},
		{"the root against nothing", f.base, "", []CommitFile{
			{Status: 'A', Path: ".env", Added: 1},
			{Status: 'A', Path: "bin.dat", Binary: true},
			{Status: 'A', Path: "del.txt", Added: 1},
			{Status: 'A', Path: "mod.txt", Added: 8},
			{Status: 'A', Path: "ren-old.txt", Added: 8},
			{Status: 'A', Path: "sub/dir/é ünï.txt", Added: 1},
			{Status: 'A', Path: "tc.txt", Added: 2},
		}},
		{"a merge against its first parent", f.merge, f.top, []CommitFile{
			{Status: 'M', Path: "mod.txt", Added: 1, Removed: 1},
			{Status: 'A', Path: "side.txt", Added: 1},
		}},
		{"a merge against its second parent", f.merge, f.side, []CommitFile{
			{Status: 'M', Path: ".env", Added: 1, Removed: 1},
			{Status: 'A', Path: "add.txt", Added: 3},
			{Status: 'M', Path: "bin.dat", Binary: true},
			{Status: 'D', Path: "del.txt", Removed: 1},
			{Status: 'M', Path: "mod.txt", Added: 2, Removed: 1},
			{Status: 'R', Path: "ren new.txt", Orig: "ren-old.txt", Added: 1, Removed: 1},
			{Status: 'M', Path: "sub/dir/é ünï.txt", Added: 1, Removed: 1},
			{Status: 'T', Path: "tc.txt", Added: 1, Removed: 2},
		}},
		{"no change", f.top, f.top, nil},
	} {
		got, err := g.CommitFiles(f.dir, tc.commit, tc.parent)
		if err != nil {
			t.Errorf("%s: %v", tc.name, err)
			continue
		}
		if !slices.Equal(got.Files, tc.want) || got.Total != len(tc.want) {
			t.Errorf("%s:\n got %+v (of %d)\nwant %+v", tc.name, got.Files, got.Total, tc.want)
		}
	}

	// The calls: the root with --root, the others against their parent.
	var root, other int
	for _, c := range calls.all() {
		if !slices.Contains(c, "diff-tree") {
			t.Errorf("a call that is no diff-tree: %q", c)
			continue
		}
		i := slices.Index(c, "diff-tree")
		if !slices.Equal(c[i:i+7], []string{"diff-tree", "-r", "-z", "--raw", "--numstat", "-M", "--no-commit-id"}) {
			t.Errorf("the diff-tree's options: %q", c[i:])
		}
		if slices.Contains(c, "--root") {
			root++
		} else {
			other++
		}
	}
	if root != 1 || other != 4 {
		t.Errorf("%d roots and %d others, want 1 and 4", root, other)
	}

	// Git's refusals, in its words; what is no commit never reaches it.
	missing := strings.Repeat("1", 40)
	var ge *gitError
	if _, err := g.CommitFiles(f.dir, missing, f.base); !errors.As(err, &ge) || !strings.Contains(err.Error(), missing) {
		t.Errorf("a commit git does not have: %v", err)
	}
	n := len(calls.all())
	out := filepath.Join(t.TempDir(), "out.txt")
	for _, c := range [][2]string{{"", f.base}, {"--output=" + out, f.base}, {f.top, "--output=" + out}, {"-1", ""}} {
		if _, err := g.CommitFiles(f.dir, c[0], c[1]); err == nil {
			t.Errorf("CommitFiles(%q, %q) was taken", c[0], c[1])
		}
	}
	if len(calls.all()) != n || exists(out) {
		t.Errorf("git ran for what is no commit, or wrote %s", out)
	}
}

// TestCommitFilesMany is plan 62's criterion 62.4 for a commit of many files
// (design D5): 1,500 files all listed, in git's order, and the guard: the
// first GraphFileCap listed and all counted.
func TestCommitFilesMany(t *testing.T) {
	gitEnv(t)
	dir := t.TempDir()
	gitT(t, dir, "init", "-q", "-b", "main")
	var h graphHist
	seed := h.commitWith("refs/heads/main", "Ana", "seed", [][2]string{{"seed.txt", "seed\n"}})
	var files [][2]string
	var want []CommitFile
	for i := 1; i <= 1500; i++ {
		name := fmt.Sprintf("many/f%04d.txt", i)
		files = append(files, [2]string{name, "x\n"})
		want = append(want, CommitFile{Status: 'A', Path: name, Added: 1})
	}
	many := h.commitWith("refs/heads/main", "Ana", "many", files, seed)
	hashes := h.run(t, dir)

	g, _ := graphGit(t)
	got, err := g.CommitFiles(dir, hashes[many], hashes[seed])
	if err != nil {
		t.Fatal(err)
	}
	if got.Total != 1500 || !slices.Equal(got.Files, want) {
		t.Errorf("%d files listed of %d, want 1,500 of 1,500 in order", len(got.Files), got.Total)
	}

	defer func(n int) { GraphFileCap = n }(GraphFileCap)
	for _, limit := range []int{7, 1500, 0} {
		GraphFileCap = limit
		got, err = g.CommitFiles(dir, hashes[many], hashes[seed])
		if err != nil {
			t.Fatal(err)
		}
		if got.Total != 1500 || !slices.Equal(got.Files, want[:limit]) {
			t.Errorf("a guard of %d: %d files listed of %d, want the first %d of 1,500", limit, len(got.Files), got.Total, limit)
		}
	}
}

// checkCommitSides is plan 62's criterion 62.5 (design D6): a file's two
// sides in a commit, parent's version against commit's, from one cat-file: an
// added, a modified, a deleted, a renamed (its old path read from the
// parent), a binary and a type-changed file, the root's, a merge's against its
// first parent; the sides named by the 7 characters of their commits; a
// credential file and a path outside the repository refused before any read.
func checkCommitSides(t *testing.T, f graphFiles) {
	g, calls := graphGit(t)
	short := func(rev string) string { return rev[:min(7, len(rev))] }
	blob := func(rev, path string) string { return gitT(t, f.dir, "rev-parse", rev+":"+path) }
	text := func(rev, path, data string) DiffSide {
		return DiffSide{Name: short(rev), Path: path, ID: blob(rev, path), Size: int64(len(data)), Data: []byte(data)}
	}
	binary := func(rev, path string, size int) DiffSide {
		return DiffSide{Name: short(rev), Path: path, ID: blob(rev, path), Size: int64(size), Binary: true}
	}
	missing := func(rev, path string) DiffSide { return DiffSide{Name: short(rev), Path: path, Missing: true} }

	for _, tc := range []struct {
		name           string
		commit, parent string
		file           CommitFile
		old, new       DiffSide
	}{
		{"added", f.top, f.base, CommitFile{Status: 'A', Path: "add.txt"},
			missing(f.base, "add.txt"), text(f.top, "add.txt", "added\n1\n2\n")},
		{"modified", f.top, f.base, CommitFile{Status: 'M', Path: "mod.txt"},
			text(f.base, "mod.txt", graphMod0), text(f.top, "mod.txt", graphMod1)},
		{"deleted", f.top, f.base, CommitFile{Status: 'D', Path: "del.txt"},
			text(f.base, "del.txt", "del\n"), missing(f.top, "del.txt")},
		{"renamed", f.top, f.base, CommitFile{Status: 'R', Path: "ren new.txt", Orig: "ren-old.txt"},
			text(f.base, "ren-old.txt", graphRen0), text(f.top, "ren new.txt", graphRen1)},
		{"binary", f.top, f.base, CommitFile{Status: 'M', Path: "bin.dat", Binary: true},
			binary(f.base, "bin.dat", 3), binary(f.top, "bin.dat", 5)},
		{"type changed", f.top, f.base, CommitFile{Status: 'T', Path: "tc.txt"},
			text(f.base, "tc.txt", "one\ntwo\n"), text(f.top, "tc.txt", "some/target")},
		{"in a folder with a space and accents", f.top, f.base, CommitFile{Status: 'M', Path: "sub/dir/é ünï.txt"},
			text(f.base, "sub/dir/é ünï.txt", "u\n"), text(f.top, "sub/dir/é ünï.txt", "u2\n")},
		{"the root's file", f.base, "", CommitFile{Status: 'A', Path: "mod.txt"},
			missing("", "mod.txt"), text(f.base, "mod.txt", graphMod0)},
		{"a merge's file, against the first parent", f.merge, f.top, CommitFile{Status: 'M', Path: "mod.txt"},
			text(f.top, "mod.txt", graphMod1), text(f.merge, "mod.txt", graphModMerged)},
		{"a merge's added file", f.merge, f.top, CommitFile{Status: 'A', Path: "side.txt"},
			missing(f.top, "side.txt"), text(f.merge, "side.txt", "side\n")},
	} {
		got, err := g.CommitSides(f.dir, tc.commit, tc.parent, tc.file)
		if err != nil {
			t.Errorf("%s: %v", tc.name, err)
			continue
		}
		if !reflect.DeepEqual(got, DiffPair{Old: tc.old, New: tc.new}) {
			t.Errorf("%s:\n got %+v\nwant %+v", tc.name, got, DiffPair{Old: tc.old, New: tc.new})
		}
	}
	// One cat-file for the two sides (none for the sides known missing).
	for _, c := range calls.all() {
		if subcommand(c) != "cat-file" || !slices.Contains(c, "--batch") {
			t.Errorf("a call that is no cat-file --batch: %q", c)
		}
	}
	if n := len(calls.all()); n != 10 {
		t.Errorf("%d cat-file calls for 10 files", n)
	}

	// Refused before any read.
	n := len(calls.all())
	for _, file := range []CommitFile{
		{Status: 'M', Path: ".env"}, {Status: 'M', Path: "sub/.ENV"}, {Status: 'M', Path: "x/secrets.json"},
		{Status: 'R', Path: "config.txt", Orig: "id_rsa"}, {Status: 'D', Path: "keys/server.pem"},
	} {
		if _, err := g.CommitSides(f.dir, f.top, f.base, file); err != ErrCredential {
			t.Errorf("%+v: %v, want ErrCredential", file, err)
		}
	}
	for _, p := range []string{"../outside.txt", "a/../../b.txt", "/etc/passwd", "a/./../../x"} {
		var fe *FileError
		if _, err := g.CommitSides(f.dir, f.top, f.base, CommitFile{Status: 'M', Path: p}); !errors.As(err, &fe) || !strings.HasPrefix(fe.Reason, "outside the repository") {
			t.Errorf("%q: %v", p, err)
		}
	}
	for _, p := range []string{"", "dir/"} {
		var fe *FileError
		if _, err := g.CommitSides(f.dir, f.top, f.base, CommitFile{Status: 'M', Path: p}); !errors.As(err, &fe) || fe.Reason != "a folder, not a file" {
			t.Errorf("%q: %v", p, err)
		}
	}
	// An empty commit would name the index, a revision that is an option
	// nothing: neither is read.
	for _, c := range [][2]string{{"", f.base}, {"-x", f.base}, {f.top, "--output=x"}} {
		if _, err := g.CommitSides(f.dir, c[0], c[1], CommitFile{Status: 'M', Path: "mod.txt"}); err == nil {
			t.Errorf("CommitSides(%q, %q) was taken", c[0], c[1])
		}
	}
	if len(calls.all()) != n {
		t.Errorf("%d git calls for what is refused", len(calls.all())-n)
	}
}

// checkReadsWriteNothing is plan 62's design D14 and criterion 62.5: all four
// reads, on a repository whose working tree and index are out of step with
// its commits and whose tracked file's time has moved, leave every file of
// it, .git included, as it was.
func checkReadsWriteNothing(t *testing.T, f graphFiles) {
	g, _ := graphGit(t)
	write(t, filepath.Join(f.dir, "mod.txt"), "edited, not staged\n")
	later := time.Now().Add(time.Hour)
	if err := os.Chtimes(filepath.Join(f.dir, "bin.dat"), later, later); err != nil {
		t.Fatal(err)
	}
	before := treeState(t, f.dir)

	if _, err := g.GraphRefs(f.dir); err != nil {
		t.Fatal(err)
	}
	if got := graphReadAll(t, g, f.dir, 2, f.merge); len(got) != 4 {
		t.Errorf("%d commits", len(got))
	}
	for _, c := range [][2]string{{f.top, f.base}, {f.base, ""}, {f.merge, f.top}} {
		files, err := g.CommitFiles(f.dir, c[0], c[1])
		if err != nil {
			t.Fatal(err)
		}
		for _, file := range files.Files {
			if _, err := g.CommitSides(f.dir, c[0], c[1], file); err != nil && err != ErrCredential {
				t.Errorf("%+v: %v", file, err)
			}
		}
	}

	after := treeState(t, f.dir)
	if !reflect.DeepEqual(before, after) {
		for p, b := range before {
			if a, ok := after[p]; !ok || a != b {
				t.Errorf("%s changed: %+v then %+v", p, b, a)
			}
		}
		for p := range after {
			if _, ok := before[p]; !ok {
				t.Errorf("%s appeared", p)
			}
		}
	}
}

// TestGraphNoLazyFetch is design D14 on a partial clone made without its
// blobs: the graph's reads of a commit's files and sides fetch nothing, the
// blob the clone lacks still missing after them; Source Control's reads keep
// git's own behaviour.
func TestGraphNoLazyFetch(t *testing.T) {
	src := srcRepo(t)
	writeAt(t, src, "a.txt", "one\n")
	gitT(t, src, "add", "-A")
	gitT(t, src, "commit", "-q", "-m", "one")
	writeAt(t, src, "a.txt", "two\n")
	gitT(t, src, "commit", "-q", "-am", "two")
	gitT(t, src, "config", "uploadpack.allowFilter", "true")
	parent, top, blob := gitT(t, src, "rev-parse", "HEAD~1"), gitT(t, src, "rev-parse", "HEAD"), gitT(t, src, "rev-parse", "HEAD~1:a.txt")
	base := t.TempDir()
	gitT(t, base, "clone", "-q", "--filter=blob:none", "--no-checkout", "file:///"+filepath.ToSlash(src), "clone")
	clone := filepath.Join(base, "clone")
	has := func() bool {
		c := exec.Command("git", "-C", clone, "cat-file", "-e", blob)
		c.Env = append(os.Environ(), "GIT_NO_LAZY_FETCH=1")
		return c.Run() == nil
	}
	if has() {
		t.Fatal("the partial clone has the old blob from the start")
	}
	g, _ := graphGit(t)
	_, ferr := g.CommitFiles(clone, top, parent)
	pair, serr := g.CommitSides(clone, top, parent, CommitFile{Status: 'M', Path: "a.txt"})
	if has() {
		t.Errorf("the graph's reads fetched the old blob")
	}
	t.Logf("in the partial clone: the files' read %v; the sides' %v, the old one missing %v", ferr, serr, pair.Old.Missing)
	cmd, err := GitCLI{}.command(context.Background(), clone, true, []string{"status"})
	if err != nil || slices.Contains(cmd.Env, "GIT_NO_LAZY_FETCH=1") {
		t.Errorf("Source Control's reads: %v, their environment %v", err, cmd.Env)
	}
}

// TestCommitMessage is design D13's read: a commit's whole message, its
// lines and the blank one between, without its last line break; a name
// that is no hash refused before git; a commit git does not have, git's
// failure. Every call one of the graph's reads.
func TestCommitMessage(t *testing.T) {
	dir := srcRepo(t)
	writeAt(t, dir, "a.txt", "a\n")
	gitT(t, dir, "add", "-A")
	gitT(t, dir, "commit", "-q", "-m", "Name the footer's parts", "-m", "The short hash, the author\nand the date.")
	head := gitT(t, dir, "rev-parse", "HEAD")
	g, trace := graphGit(t)
	msg, err := g.CommitMessage(dir, head)
	if want := "Name the footer's parts\n\nThe short hash, the author\nand the date."; err != nil || msg != want {
		t.Errorf("the message %q, %v; want %q", msg, err, want)
	}
	trace.reset()
	if _, err := g.CommitMessage(dir, "--output=x"); err == nil || len(trace.all()) != 0 {
		t.Errorf("an option: %v, %d calls", err, len(trace.all()))
	}
	if _, err := g.CommitMessage(dir, strings.Repeat("0", 40)); err == nil {
		t.Error("a commit git does not have: no error")
	}
}

// TestGraphScopeRevList is criterion 62.3's comparison (design D3): on a
// history of twelve branches off one base, four merged into main, each
// scope's history, read page by page from its tips, is what git rev-list
// --topo-order gives for the same revisions: Auto, All, one branch, two.
func TestGraphScopeRevList(t *testing.T) {
	dir := srcRepo(t)
	var h graphHist
	graphRecLanesHist(&h)
	h.run(t, dir)
	g, _ := graphGit(t)
	refs, err := g.GraphRefs(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		name  string
		scope GraphScope
		n     int
	}{{"Auto", GraphScope{}, 13}, {"All", GraphScope{All: true}, 29}, {"b01", GraphScope{Refs: []string{"refs/heads/b01"}}, 3},
		{"b01 and b05", GraphScope{Refs: []string{"refs/heads/b01", "refs/heads/b05"}}, 5}} {
		tips, gone := refs.Tips(c.scope)
		if len(gone) != 0 {
			t.Fatalf("%s: gone %v", c.name, gone)
		}
		var got []string
		for _, cm := range graphReadAll(t, g, dir, 4, tips...) {
			got = append(got, cm.Hash)
		}
		want := lines(gitRaw(t, dir, append(append([]string{"rev-list", "--topo-order"}, tips...), "--")...))
		if !slices.Equal(got, want) || len(got) != c.n {
			t.Errorf("%s: %d commits read, rev-list's %d, want %d alike", c.name, len(got), len(want), c.n)
		}
	}
}
