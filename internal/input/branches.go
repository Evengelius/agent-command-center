package input

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
)

// The branches popup's git (plan 59, brief amendment 61): its reads, all
// through runRead, and the two calls that change something, Switch and
// Fetch, each doing one thing and never forcing: no --force, -f,
// --discard-changes or --merge, no stash, no reset, no prune (designs D5
// and D7).

// BranchGit is what the branches popup asks of git; tests pass a fake.
type BranchGit interface {
	// Branches reads the repository's branches and tags for the popup
	// (design D4).
	Branches(dir string) (Branches, error)
	// Changes counts dir's uncommitted changes, untracked files included and
	// ignored ones not, for the switch's confirmation (design D5).
	Changes(dir string) (int, error)
	// Switch switches dir's worktree to the local branch name or, remote
	// true, to a new local branch tracking the remote branch name, and
	// gives the local branch now checked out (design D5). Git's refusal
	// comes back in its words; nothing is forced.
	Switch(dir, name string, remote bool) (string, error)
	// Fetch fetches every remote and prunes nothing; stop, when given, can
	// end it (design D7).
	Fetch(dir string, stop *GitStop) error
	// HeadFile is the path of the HEAD file of dir's worktree (design D9).
	HeadFile(dir string) (string, error)
}

var _ BranchGit = GitCLI{}

// BranchCap is the most rows a group holds; the popup then says so (design
// D3).
const BranchCap = 1000

// RecentMax is how many branches Recent names (design D4).
const RecentMax = 5

// FetchLimit is how long a fetch may run before the application stops it
// (design D7).
const FetchLimit = 2 * time.Minute

// Ref is a branch or a tag as the popup lists it.
type Ref struct {
	// Name is the short name: "main", "origin/main", "v1.0".
	Name string
	// Upstream is a local branch's upstream, short ("origin/main"), "" when
	// it has none; Ahead and Behind count the commits it has that the
	// upstream has not, and the reverse; Gone says the upstream is gone.
	Upstream      string
	Ahead, Behind int
	Gone          bool
	// Current says HEAD is on this branch; Elsewhere is the other worktree
	// that has it checked out, which the popup does not offer (design D5).
	Current   bool
	Elsewhere string
	// date is a tag's, for the order: newest first.
	date int64
}

// Branches is the popup's data (design D4).
type Branches struct {
	// Head is where HEAD is: its branch, or its commit while detached, with
	// the branch's upstream and counts.
	Head GitBranch
	// Recent names the local branches last switched to, newest first, the
	// current one left out, RecentMax at most.
	Recent []string
	// Local, Remote and Tags are the groups, BranchCap rows each at most;
	// the totals say how many there were.
	Local, Remote, Tags                []Ref
	LocalTotal, RemoteTotal, TagsTotal int
}

// refsFormat is each ref's line of for-each-ref: its full name, "*" when
// HEAD is on it, the worktree that has it checked out, its upstream and how
// it tracks it, its date.
const refsFormat = "--format=%(refname)%00%(HEAD)%00%(worktreepath)%00%(upstream:short)%00%(upstream:track,nobracket)%00%(creatordate:unix)"

// Branches reads the popup's data in five reads: the worktree's top level,
// every branch and tag, where HEAD is, and HEAD's reflog for Recent.
func (g GitCLI) Branches(dir string) (Branches, error) {
	top, err := g.runRead(readLimit, dir, "rev-parse", "--show-toplevel")
	if err != nil {
		return Branches{}, err
	}
	top = strings.TrimSpace(top)
	out, err := g.runRead(readLimit, dir, "for-each-ref", refsFormat, "refs/heads", "refs/remotes", "refs/tags")
	if err != nil {
		return Branches{}, err
	}
	var b Branches
	local := map[string]bool{}
	for _, l := range lines(out) {
		f := strings.Split(l, "\x00")
		if len(f) != 6 {
			continue
		}
		full, head, tree, up, track := f[0], f[1], f[2], f[3], f[4]
		switch {
		case strings.HasPrefix(full, "refs/heads/"):
			r := Ref{Name: strings.TrimPrefix(full, "refs/heads/"), Upstream: up, Current: head == "*"}
			r.Ahead, r.Behind, r.Gone = parseTrack(track)
			if tree != "" && !samePlace(tree, top) {
				r.Elsewhere = filepath.FromSlash(tree)
			}
			local[r.Name] = true
			b.Local = append(b.Local, r)
		case strings.HasPrefix(full, "refs/remotes/"):
			// A remote's HEAD names its default branch, which is listed already.
			if name := strings.TrimPrefix(full, "refs/remotes/"); !strings.HasSuffix(name, "/HEAD") {
				b.Remote = append(b.Remote, Ref{Name: name})
			}
		case strings.HasPrefix(full, "refs/tags/"):
			date, _ := strconv.ParseInt(f[5], 10, 64)
			b.Tags = append(b.Tags, Ref{Name: strings.TrimPrefix(full, "refs/tags/"), date: date})
		}
	}
	// Tags newest first; branches as for-each-ref sorts them, by name.
	sort.SliceStable(b.Tags, func(i, j int) bool { return b.Tags[i].date > b.Tags[j].date })
	if b.Head, err = g.headState(dir); err != nil {
		return Branches{}, err
	}
	for _, r := range b.Local {
		if r.Current {
			b.Head.Upstream, b.Head.Ahead, b.Head.Behind, b.Head.Gone = r.Upstream, r.Ahead, r.Behind, r.Gone
		}
	}
	b.Recent = g.recent(dir, b.Head, local)
	b.LocalTotal, b.RemoteTotal, b.TagsTotal = len(b.Local), len(b.Remote), len(b.Tags)
	b.Local, b.Remote, b.Tags = capRefs(b.Local), capRefs(b.Remote), capRefs(b.Tags)
	return b, nil
}

// headState is where HEAD is: on a branch, possibly with no commit yet, or
// detached at a commit.
func (g GitCLI) headState(dir string) (GitBranch, error) {
	name, err := g.runRead(readLimit, dir, "symbolic-ref", "-q", "--short", "HEAD")
	var ge *gitError
	detached := errors.As(err, &ge) && ge.code == 1
	if err != nil && !detached {
		return GitBranch{}, err
	}
	id, idErr := g.runRead(readLimit, dir, "rev-parse", "-q", "--verify", "--short", "HEAD")
	id = strings.TrimSpace(id)
	if detached {
		return GitBranch{Name: "HEAD", Detached: true, Head: id}, nil
	}
	return GitBranch{Name: strings.TrimSpace(name), NoCommits: idErr != nil || id == ""}, nil
}

// recent is HEAD's reflog's switches, newest first: the branches moved to
// that are still local branches, the current one left out, RecentMax at
// most. A repository with no reflog has none.
func (g GitCLI) recent(dir string, head GitBranch, local map[string]bool) []string {
	out, err := g.runRead(readLimit, dir, "reflog", "show", "--format=%gs", "-n", "300", "HEAD")
	if err != nil {
		return nil
	}
	var names []string
	seen := map[string]bool{}
	for _, l := range lines(out) {
		_, to, ok := strings.Cut(strings.TrimPrefix(l, "checkout: moving from "), " to ")
		if !ok || !strings.HasPrefix(l, "checkout: moving from ") {
			continue
		}
		if to == head.Name && !head.Detached || !local[to] || seen[to] {
			continue
		}
		seen[to] = true
		if names = append(names, to); len(names) == RecentMax {
			break
		}
	}
	return names
}

// parseTrack reads for-each-ref's "%(upstream:track,nobracket)": "ahead 2,
// behind 1", "gone", or nothing.
func parseTrack(track string) (ahead, behind int, gone bool) {
	if track == "gone" {
		return 0, 0, true
	}
	for _, w := range strings.Split(track, ", ") {
		if n, ok := strings.CutPrefix(w, "ahead "); ok {
			ahead, _ = strconv.Atoi(n)
		} else if n, ok := strings.CutPrefix(w, "behind "); ok {
			behind, _ = strconv.Atoi(n)
		}
	}
	return ahead, behind, false
}

// capRefs is refs' first BranchCap.
func capRefs(refs []Ref) []Ref {
	if len(refs) > BranchCap {
		return refs[:BranchCap]
	}
	return refs
}

// samePlace says two of git's paths name the same folder: forward slashes
// or not, and on Windows in any case.
func samePlace(a, b string) bool {
	a, b = filepath.Clean(filepath.FromSlash(a)), filepath.Clean(filepath.FromSlash(b))
	if runtime.GOOS == "windows" {
		return strings.EqualFold(a, b)
	}
	return a == b
}

// Changes counts the lines of git status --porcelain: each changed, staged,
// conflicted or untracked path once, the ignored never.
func (g GitCLI) Changes(dir string) (int, error) {
	out, err := g.runRead(readLimit, dir, "status", "--porcelain")
	if err != nil {
		return 0, err
	}
	return len(lines(out)), nil
}

// Switch is git switch name, or git switch --track name for a remote
// branch, refused when a local branch of its name exists (design D5).
func (g GitCLI) Switch(dir, name string, remote bool) (string, error) {
	local, args := name, []string{"switch", name}
	if remote {
		var err error
		if local, err = g.localName(dir, name); err != nil {
			return "", err
		}
		if _, err := g.runRead(readLimit, dir, "rev-parse", "-q", "--verify", "refs/heads/"+local); err == nil {
			return "", Refusal{"local-exists", fmt.Sprintf("A local branch %s exists already: switch to it in Local.", local)}
		}
		args = []string{"switch", "--track", name}
	}
	res := g.exec(dir, nil, nil, "switch", args, nil, nil)
	if res.err != nil {
		var ge *gitError
		if errors.As(res.err, &ge) {
			return "", &gitError{code: ge.code, msg: res.failed()}
		}
		return "", res.err
	}
	return local, nil
}

// localName is the local branch git switch --track makes of the remote
// branch name: its name without its remote's.
func (g GitCLI) localName(dir, name string) (string, error) {
	out, err := g.runRead(readLimit, dir, "remote")
	if err != nil {
		return "", err
	}
	best := ""
	for _, r := range lines(out) {
		if strings.HasPrefix(name, r+"/") && len(r) > len(best) {
			best = r
		}
	}
	if best == "" {
		return "", fmt.Errorf("%s belongs to no remote of this repository", name)
	}
	return strings.TrimPrefix(name, best+"/"), nil
}

// Fetch is git fetch --all --no-prune (design D7). Git never prompts: the
// runner's GIT_TERMINAL_PROMPT=0, and ssh's askpass as for Push. With stop,
// it runs in its own process group, and a stop waits while one of the lock
// files it could hold exists (packed-refs, shallow, a remote branch's), as
// Push's does; the application asks for the stop on Esc and at FetchLimit.
// No message shows a URL's user or password.
func (g GitCLI) Fetch(dir string, stop *GitStop) error {
	env := []string{"SSH_ASKPASS_REQUIRE=force"}
	if os.Getenv("SSH_ASKPASS") == "" {
		env = append(env, "SSH_ASKPASS=agent-command-center-no-askpass")
	}
	var locks []string
	if stop != nil {
		var err error
		if locks, err = g.fetchLocks(dir); err != nil {
			return err
		}
	}
	res := g.exec(dir, nil, env, "fetch", []string{"fetch", "--all", "--no-prune"}, stop, locks)
	switch {
	case stop.stopped():
		return Refusal{"cancelled", "The fetch was stopped."}
	case res.err != nil:
		var ge *gitError
		if errors.As(res.err, &ge) {
			return &gitError{code: ge.code, msg: scrubURLs(res.failed())}
		}
		return errors.New(scrubURLs(res.err.Error()))
	}
	return nil
}

// fetchLocks is the lock files a fetch could hold: packed-refs', shallow's
// and each remote branch's known now.
func (g GitCLI) fetchLocks(dir string) ([]string, error) {
	names := []string{"packed-refs.lock", "shallow.lock"}
	out, err := g.runRead(readLimit, dir, "for-each-ref", "--format=%(refname)", "refs/remotes")
	if err != nil {
		return nil, err
	}
	for _, r := range lines(out) {
		names = append(names, r+".lock")
	}
	return g.gitPaths(dir, names...)
}

// HeadFile is git rev-parse --git-path HEAD made absolute: a worktree's own
// HEAD (design D9).
func (g GitCLI) HeadFile(dir string) (string, error) {
	p, err := g.gitPaths(dir, "HEAD")
	if err != nil {
		return "", err
	}
	if len(p) != 1 {
		return "", errors.New("git gave no path for HEAD")
	}
	return p[0], nil
}
