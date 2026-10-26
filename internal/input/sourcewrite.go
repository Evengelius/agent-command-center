package input

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// The Source Control's writing (plan 45, brief amendment 46): staging, the
// commit, a selection's commit (plan 50, brief amendment 51) and the push,
// the only git calls of the application that change a repository. Each
// does one thing, refuses with a Refusal before git runs when the state
// forbids it, and never forces: no --force, --force-with-lease, -f, "+"
// refspec, --mirror, --all, --tags, --delete, --allow-empty, --amend or
// --no-verify (design D6 to D11).

// SourceWrite is what the Source Control changes through git; tests pass a
// fake.
type SourceWrite interface {
	// Stage adds exactly paths (project-relative, forward slashes; an untracked
	// folder ends with "/") to the index, deletions and untracked files
	// included. With no path it does nothing.
	Stage(dir string, paths []string) error
	// Commit commits the index as it is with message, once the staged changes
	// inside the project are those of expect (design D8). Anything wrong with
	// the state is a Refusal.
	Commit(dir, message string, expect []GitEntry) (CommitResult, error)
	// CheckSelection refuses a commit of sel, the selected files as the panel
	// showed them, that the state forbids, before anything is written (plan
	// 50, design D7): it reads only.
	CheckSelection(dir string, sel []GitEntry) error
	// CommitSelection commits the staged content of sel's files and nothing
	// else, the rest of the index left as it is (plan 50, design D7). staged
	// names the paths the question's staging staged, taken as they now are;
	// the others must be as shown. A commit made whose content is not the
	// one asked for is an Unexpected.
	CommitSelection(dir, message string, sel []GitEntry, staged []string) (CommitResult, error)
	// PushTarget is where Push would send the current branch, or the Refusal
	// that says why it cannot (design D9).
	PushTarget(dir string) (PushTarget, error)
	// Push sends the current branch to its upstream and nothing else. A
	// refusal is a Refusal, a rejection by the remote a PushRejected; stop,
	// when given, can end it (plan 45, design D15).
	Push(dir string, stop *GitStop) (PushResult, error)
}

var _ SourceWrite = GitCLI{}

// CommitResult names the commit made: its full id and its short one.
type CommitResult struct{ Hash, Short string }

// PushTarget is where the current branch is pushed (plan 45, design D9).
type PushTarget struct {
	// Branch is the local branch, Remote the upstream's remote, RemoteRef its
	// full ref on that remote ("refs/heads/main"), Upstream its short name
	// ("origin/main").
	Branch, Remote, RemoteRef, Upstream string
	// Ahead and Behind count the commits the branch has that the upstream has
	// not, and the reverse.
	Ahead, Behind int
	// tracking is the upstream's remote-tracking ref, whose lock a push
	// could hold ("refs/remotes/origin/main").
	tracking string
}

// PushResult is a push that went through.
type PushResult struct{ Target PushTarget }

// Refusal is the application declining to run git, or git declining for a
// reason the owner can fix (plan 45, design D8 and D9). Kind is one of
// "nothing-staged", "conflict", "outside", "changed", "rebase", "am",
// "cherry-pick", "revert", "identity", "detached", "no-remote",
// "no-upstream", "local-upstream", "gone", "elsewhere" and "up-to-date", or,
// for a selection's commit (plan 50, design D7), "nothing-selected",
// "merge" and "partial"; Msg is what the panel shows as it is.
type Refusal struct{ Kind, Msg string }

func (r Refusal) Error() string { return r.Msg }

// Unexpected is a selection's commit made whose content is not the one
// asked for (plan 50, design D7): Commit names it, What says how it
// differs. The commit stays: nothing is rewritten.
type Unexpected struct {
	Commit CommitResult
	What   string
}

func (u Unexpected) Error() string {
	return fmt.Sprintf("Committed %s, but %s: check it in a terminal.", u.Commit.Short, u.What)
}

// PushRejected is the push git or the remote turned down, read from the
// porcelain line of the ref (design D9).
type PushRejected struct {
	// Ref is the porcelain's "<from>:<to>"; Reason the text in parentheses
	// ("fetch first", "non-fast-forward", "pre-receive hook declined").
	Ref, Reason string
	// NonFastForward says the remote has commits the branch does not; Remote
	// says the remote's own side refused (a hook, a protected branch).
	NonFastForward, Remote bool
	summary                string // the porcelain's summary: "[rejected]"
	upstream               string // the short upstream, for the message
}

func (p PushRejected) Error() string {
	switch {
	case p.NonFastForward:
		from, to, _ := strings.Cut(p.Ref, ":")
		up := p.upstream
		if up == "" {
			up = strings.TrimPrefix(to, "refs/heads/")
		}
		return fmt.Sprintf("The push was rejected: %s has commits that %s does not (non-fast-forward). Pull or merge them in a terminal, then push.",
			up, strings.TrimPrefix(from, "refs/heads/"))
	case p.Remote:
		reason := p.Reason
		if reason == "" {
			reason = "no reason given"
		}
		return "The remote rejected the push: " + reason + "."
	}
	if p.Reason == "" {
		return "The push was rejected: " + p.summary + "."
	}
	return "The push was rejected: " + p.summary + " (" + p.Reason + ")."
}

// userInfo is the user and password of a URL, which no message shows.
var userInfo = regexp.MustCompile(`(?i)([a-z][a-z0-9+.-]*://)[^/\s]*@`)

// scrubURLs removes the user info of every URL in s: scheme://user:pass@host
// becomes scheme://host.
func scrubURLs(s string) string { return userInfo.ReplaceAllString(s, "${1}") }

// gitResult is a finished git call: both outputs and its failure, a *gitError
// when git exited with a code.
type gitResult struct {
	out, errOut string
	err         error
}

// failed is what git said, stderr then stdout, on one line.
func (r gitResult) failed() string {
	return strings.Join(strings.Fields(r.errOut+"\n"+r.out), " ")
}

// exec is git -C dir args with stdin as its standard input, in env added to
// the runner's, with no time limit and no kill: a commit or a push ends when
// git ends it (plan 45, design D8 and D9). sub names the subcommand for a
// failure with nothing to say. With stop, git runs in its own process group
// and stop can end it while none of locks exists (design D15).
func (g GitCLI) exec(dir string, stdin []byte, env []string, sub string, args []string, stop *GitStop, locks []string) gitResult {
	cmd, err := g.command(context.Background(), dir, false, args)
	if err != nil {
		return gitResult{err: err}
	}
	cmd.Env = append(cmd.Env, env...)
	if stdin != nil {
		cmd.Stdin = bytes.NewReader(stdin)
	}
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	if stop != nil {
		ownGroup(cmd)
	}
	if err = cmd.Start(); err == nil {
		disarm := stop.arm(cmd.Process.Pid, locks)
		err = cmd.Wait()
		disarm()
		if errors.Is(err, exec.ErrWaitDelay) {
			err = nil // git ended with 0; a child kept its pipes (gitWaitDelay)
		}
	}
	res := gitResult{out: out.String(), errOut: errb.String()}
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		res.err = failure([]string{sub}, errb.String(), ee)
	} else {
		res.err = err
	}
	return res
}

// Stage adds exactly paths to the index (plan 45, design D6): git add -A with
// literal pathspecs read from standard input, so that a name with a space, a
// bracket or a leading dash is taken as it is, and an untracked folder adds
// its files as git adds them, the ignored ones never.
func (g GitCLI) Stage(dir string, paths []string) error {
	if len(paths) == 0 {
		return nil
	}
	in := strings.Join(paths, "\x00") + "\x00"
	res := g.exec(dir, []byte(in), []string{"GIT_LITERAL_PATHSPECS=1"}, "add",
		[]string{"add", "-A", "--pathspec-from-file=-", "--pathspec-file-nul"}, nil, nil)
	return res.err
}

// inProgress is the Refusal of a rebase, am, cherry-pick or revert under way
// in dir's repository (design D8); a merge is allowed.
func (g GitCLI) inProgress(dir string) error {
	out, err := g.runRead(readLimit, dir, "rev-parse", "--git-path", "rebase-merge", "--git-path", "rebase-apply",
		"--git-path", "rebase-apply/applying", "--git-path", "CHERRY_PICK_HEAD", "--git-path", "REVERT_HEAD")
	if err != nil {
		return err
	}
	ps := strings.Split(strings.TrimRight(strings.ReplaceAll(out, "\r\n", "\n"), "\n"), "\n")
	if len(ps) != 5 {
		return fmt.Errorf("git rev-parse --git-path said %d paths, want 5", len(ps))
	}
	has := func(i int) bool {
		p := filepath.FromSlash(ps[i])
		if !filepath.IsAbs(p) {
			p = filepath.Join(dir, p)
		}
		_, err := os.Lstat(p)
		return err == nil
	}
	switch {
	case has(2):
		return Refusal{"am", "An am session is in progress: finish it in a terminal first."}
	case has(1) || has(0):
		return Refusal{"rebase", "A rebase is in progress: finish it in a terminal first."}
	case has(3):
		return Refusal{"cherry-pick", "A cherry-pick is in progress: finish it in a terminal first."}
	case has(4):
		return Refusal{"revert", "A revert is in progress: finish it in a terminal first."}
	}
	return nil
}

// stagedEntries reads the whole repository's staged entries, paths relative
// to its top level, and the project's place in it (plan 45, design D8).
// conflict says an unmerged entry exists.
func (g GitCLI) stagedEntries(dir string) (staged []GitEntry, prefix string, conflict bool, err error) {
	pre, err := g.runRead(readLimit, dir, "rev-parse", "--show-prefix")
	if err != nil {
		return nil, "", false, err
	}
	prefix = strings.TrimRight(pre, "\r\n")
	st, err := g.runRead(readLimit, dir, "status", "--porcelain=v1", "-z", "--untracked-files=no")
	if err != nil {
		return nil, "", false, err
	}
	entries, _, _ := ParseSnapshot(st, "")
	for _, e := range entries {
		if ConflictName(e.X, e.Y) != "" {
			conflict = true
		} else if e.X != ' ' && e.X != '?' && e.X != '!' {
			staged = append(staged, e)
		}
	}
	return staged, prefix, conflict, nil
}

// Commit makes the commit of the index as it is (plan 45, design D8): the
// state is checked first, then git commit --file=- is run with the message
// on standard input as given, its own clean-up, hooks and signing; the
// result counts only if HEAD is a new commit on the old one.
func (g GitCLI) Commit(dir, message string, expect []GitEntry) (CommitResult, error) {
	if err := g.inProgress(dir); err != nil {
		return CommitResult{}, err
	}
	staged, prefix, conflict, err := g.stagedEntries(dir)
	if err != nil {
		return CommitResult{}, err
	}
	if conflict {
		return CommitResult{}, Refusal{"conflict", "Resolve the merge conflicts first."}
	}
	if len(staged) == 0 {
		return CommitResult{}, Refusal{"nothing-staged", "No changes to commit."}
	}
	// The staged entries inside the project, their paths relative to it; the
	// ones with a path outside it would be committed too.
	inside := func(p string) (string, bool) { return within(prefix, p) }
	outside := 0
	var touched []string
	for _, e := range staged {
		p, ok := inside(e.RepoPath)
		o := ""
		if e.RepoOrig != "" {
			var ok2 bool
			if o, ok2 = inside(e.RepoOrig); !ok2 {
				ok = false
			}
		}
		if !ok {
			outside++
			continue
		}
		touched = append(touched, p)
		if o != "" {
			touched = append(touched, o)
		}
	}
	if outside > 0 {
		what := "staged changes"
		if outside == 1 {
			what = "staged change"
		}
		return CommitResult{}, Refusal{"outside", fmt.Sprintf("%d %s outside this folder would be committed too: commit them from a terminal, or unstage them.", outside, what)}
	}
	changed := Refusal{"changed", "The staged changes changed since they were shown: check them, then commit again."}
	if !sameStaged(touched, expect) {
		return CommitResult{}, changed
	}
	// Their contents too: each expected path's blob, as the snapshot read
	// it, still the index's (plan 45, design D14).
	var shown []string
	for _, e := range expect {
		if e.Index != "" {
			shown = append(shown, e.RepoPath)
		}
	}
	if len(shown) > 0 {
		now, err := g.stagedBlobs(dir, shown)
		if err != nil {
			return CommitResult{}, err
		}
		for _, e := range expect {
			if b, ok := now[e.RepoPath]; ok && e.Index != "" && b != e.Index {
				return CommitResult{}, changed
			}
		}
	}

	old, err := g.head(dir)
	if err != nil {
		return CommitResult{}, err
	}
	res := g.exec(dir, []byte(message), nil, "commit", []string{"-c", "user.useConfigOnly=true", "commit", "--file=-"}, nil, nil)
	if res.err != nil {
		return CommitResult{}, commitFailed(res)
	}
	return g.madeOn(dir, old)
}

// commitFailed is a git commit that failed, as the panel says it: no
// identity is a Refusal, anything else git's own words (plan 45, design D8).
func commitFailed(res gitResult) error {
	text := res.failed()
	for _, s := range []string{"Please tell me who you are", "useConfigOnly", "auto-detection is disabled"} {
		if strings.Contains(text, s) {
			return Refusal{"identity", "Git has no user name and email for this repository: set user.name and user.email in a terminal, then commit."}
		}
	}
	var ge *gitError
	if errors.As(res.err, &ge) && text != "" {
		return &gitError{code: ge.code, msg: text}
	}
	return res.err
}

// madeOn is the commit git made once it exited 0: it counts only if HEAD is
// a new commit on old, "" for a branch's first (plan 45, design D8).
func (g GitCLI) madeOn(dir, old string) (CommitResult, error) {
	bad := errors.New("git exited 0 but HEAD is not a new commit on the old one")
	hash, err := g.head(dir)
	if err != nil {
		return CommitResult{}, err
	}
	if hash == "" || hash == old {
		return CommitResult{}, bad
	}
	parent, err := g.runRead(readLimit, dir, "rev-parse", "-q", "--verify", "HEAD^1")
	var ge *gitError
	switch {
	case old == "" && errors.As(err, &ge) && ge.code == 1:
		// a root commit: HEAD has no parent
	case old == "" || err != nil || strings.TrimSpace(parent) != old:
		return CommitResult{}, bad
	}
	short, err := g.runRead(readLimit, dir, "rev-parse", "--short", "HEAD")
	if err != nil {
		return CommitResult{}, err
	}
	return CommitResult{Hash: hash, Short: strings.TrimSpace(short)}, nil
}

// within is p, a path relative to the repository's top level, relative to
// the project at prefix instead, and whether the project holds it.
func within(prefix, p string) (string, bool) {
	if len(p) < len(prefix) || !strings.EqualFold(p[:len(prefix)], prefix) {
		return "", false
	}
	return p[len(prefix):], true
}

// head is the id of HEAD, "" when the branch has no commit yet.
func (g GitCLI) head(dir string) (string, error) {
	out, err := g.runRead(readLimit, dir, "rev-parse", "-q", "--verify", "HEAD")
	var ge *gitError
	if errors.As(err, &ge) && ge.code == 1 {
		return "", nil
	}
	return strings.TrimSpace(out), err
}

// sameStaged reports whether the project-relative paths touched by the staged
// entries are those of expect: its Path and Orig. An expect path ending in
// "/" is an untracked folder that was staged: it matches every staged path
// under it, and at least one (plan 45, design D8).
func sameStaged(touched []string, expect []GitEntry) bool {
	exact := map[string]bool{}
	var dirs []string
	add := func(p string) {
		switch {
		case p == "":
		case strings.HasSuffix(p, "/"):
			dirs = append(dirs, p)
		default:
			exact[p] = false
		}
	}
	for _, e := range expect {
		add(e.Path)
		add(e.Orig)
	}
	used := make([]bool, len(dirs))
	for _, p := range touched {
		if _, ok := exact[p]; ok {
			exact[p] = true
			continue
		}
		matched := false
		for i, d := range dirs {
			if strings.HasPrefix(p, d) {
				used[i], matched = true, true
			}
		}
		if !matched {
			return false
		}
	}
	for _, seen := range exact {
		if !seen {
			return false
		}
	}
	for _, u := range used {
		if !u {
			return false
		}
	}
	return true
}

// merging is the Refusal of a merge in progress in dir's repository for a
// selection's commit (plan 50, design D7): git commits no part of a merge
// ("cannot do a partial commit during a merge").
func (g GitCLI) merging(dir string) error {
	out, err := g.runRead(readLimit, dir, "rev-parse", "--git-path", "MERGE_HEAD")
	if err != nil {
		return err
	}
	p := filepath.FromSlash(strings.TrimRight(out, "\r\n"))
	if !filepath.IsAbs(p) {
		p = filepath.Join(dir, p)
	}
	if _, err := os.Lstat(p); err == nil {
		return Refusal{"merge", "A merge is in progress: commit it whole, without a selection."}
	}
	return nil
}

// selected is a selection checked against git's status (plan 50, design
// D7): the pathspecs of git commit --only, relative to the project, a
// rename's source beside its file; the paths the commit may change,
// relative to the repository's top level; and the index's blob of each,
// "-" for none, as the check read it.
type selected struct {
	specs, paths []string
	blobs        map[string]string
}

// WholeRefusal is the Refusal of a selected file, its entries in git's
// status, that a selection's commit cannot take whole, nil when it can
// (plan 50, design D7): staged and unstaged changes both, or a deletion in
// the index beside the same file untracked on disk. The panel asks it of its
// snapshot before the question, the commit of git's status read then.
func WholeRefusal(path string, entries []GitEntry) error {
	tracked, untracked := false, false
	for _, e := range entries {
		if e.X == '?' && e.Y == '?' {
			untracked = true
			continue
		}
		tracked = true
		if e.X != ' ' && e.Y != ' ' {
			return Refusal{"partial", path + " has staged and unstaged changes, and a selection commits whole files: leave it out of the selection, or stage the rest in a terminal."}
		}
	}
	if tracked && untracked {
		return Refusal{"partial", path + " is deleted in the index but still on disk: leave it out of the selection, or commit it from a terminal."}
	}
	return nil
}

// changedSince is the Refusal of a selected file no longer as it was shown.
func changedSince(e GitEntry) Refusal {
	return Refusal{"changed", e.Path + " changed since it was shown: check the selection, then commit again."}
}

// selection checks sel, the selected files as the panel showed them,
// against git's status read now (plan 50, design D7): no rebase, am,
// cherry-pick, revert or merge in progress, no conflict, and each file as
// shown, staged whole or not staged at all, a rename's source in the
// project. Before the staging (final false) a file not staged may stay so,
// the question staging it; for the commit (final true) every file is
// staged, those of staged as they now are, the others as shown, their
// blobs too. The status is read path by path, as git commit --only takes
// them: a rename shown is its file added and its source deleted, and a
// file moved on disk, both its sides selected, stays two paths once the
// question has staged them, where git would pair them.
func (g GitCLI) selection(dir string, sel []GitEntry, staged []string, final bool) (selected, error) {
	var s selected
	if len(sel) == 0 {
		return s, Refusal{"nothing-selected", "No file is selected."}
	}
	if err := g.inProgress(dir); err != nil {
		return s, err
	}
	if err := g.merging(dir); err != nil {
		return s, err
	}
	_, prefix, conflict, err := g.stagedEntries(dir)
	if err != nil {
		return s, err
	}
	if conflict {
		return s, Refusal{"conflict", "Resolve the merge conflicts first."}
	}
	// The paths as shown: each file, a rename's or a copy's as added, and a
	// rename's source as deleted from the index.
	type part struct {
		e      GitEntry // the selected file as shown
		spec   string   // the path relative to the project, its pathspec
		repo   string   // relative to the repository's top level
		x, y   byte     // its letters path by path
		source bool     // a rename's source
	}
	var parts []part
	for _, e := range sel {
		parts = append(parts, part{e: e, spec: e.Path, repo: e.RepoPath, x: added(e.X), y: added(e.Y)})
		if e.X == 'R' {
			o, ok := within(prefix, e.RepoOrig)
			if !ok {
				return s, Refusal{"outside", fmt.Sprintf("%s was renamed from %s, outside this folder: commit it from a terminal.", e.Path, e.RepoOrig)}
			}
			parts = append(parts, part{e: e, spec: o, repo: e.RepoOrig, x: 'D', y: ' ', source: true})
		}
	}
	// Their status with no rename paired, untracked files one by one, names
	// literal.
	args := []string{"--literal-pathspecs", "status", "--porcelain=v1", "-z", "--no-renames", "--untracked-files=all", "--"}
	for _, p := range parts {
		args = append(args, p.spec)
	}
	st, err := g.runRead(readLimit, dir, args...)
	if err != nil {
		return s, err
	}
	entries, _, _ := ParseSnapshot(st, prefix)
	now := map[string][]GitEntry{}
	for _, e := range entries {
		now[e.RepoPath] = append(now[e.RepoPath], e)
	}
	byUs := map[string]bool{}
	for _, p := range staged {
		byUs[p] = true
	}
	var shown []GitEntry // staged as shown: their blobs must be the ones shown
	for _, p := range parts {
		cur, e := now[p.repo], p.e
		if p.source {
			// Deleted from the index, not back on disk.
			for _, x := range cur {
				if x.X == '?' && x.Y == '?' {
					return s, Refusal{"partial", fmt.Sprintf("%s, the source of the rename to %s, is back on disk: commit it from a terminal.", p.spec, e.Path)}
				}
			}
			if len(cur) != 1 || cur[0].X != 'D' {
				return s, changedSince(e)
			}
			s.specs = append(s.specs, p.spec)
			s.paths = append(s.paths, p.repo)
			continue
		}
		if len(cur) == 0 {
			return s, changedSince(e)
		}
		if err := WholeRefusal(e.Path, cur); err != nil {
			return s, err
		}
		c := cur[0] // one entry: WholeRefusal refuses a deletion beside its file untracked
		stagedNow := c.X != ' ' && c.X != '?'
		switch {
		case final && byUs[e.Path]:
			if !stagedNow {
				return s, changedSince(e) // the staging did not take it
			}
		case c.X != p.x || c.Y != p.y:
			return s, changedSince(e)
		case final && !stagedNow:
			return s, changedSince(e) // not staged, and not by the question
		case e.Index != "":
			shown = append(shown, e)
		}
		s.specs = append(s.specs, p.spec)
		s.paths = append(s.paths, p.repo)
	}
	blobs, err := g.stagedBlobs(dir, s.paths)
	if err != nil {
		return s, err
	}
	for _, e := range shown {
		if b, ok := blobs[e.RepoPath]; ok && b != e.Index {
			return s, changedSince(e)
		}
	}
	s.blobs = blobs
	return s, nil
}

// added is a status letter of a file path by path: a rename's or a copy's
// file is an added one.
func added(c byte) byte {
	if c == 'R' || c == 'C' {
		return 'A'
	}
	return c
}

// CheckSelection refuses what a commit of sel cannot be, before anything is
// written (plan 50, design D7).
func (g GitCLI) CheckSelection(dir string, sel []GitEntry) error {
	_, err := g.selection(dir, sel, nil, false)
	return err
}

// CommitSelection commits the staged content of sel's files and nothing
// else (plan 50, design D7): the check again, then git commit --only with
// the files and a rename's source as literal pathspecs read from a file,
// the message on standard input as Commit gives it, git's own clean-up,
// hooks and signing. Git builds the commit from HEAD in its own transaction
// and leaves the rest of the index as it is; no selected file has an
// unstaged change, so the working tree's content --only takes is the staged
// one. It counts only as a new commit on the old HEAD; one that holds other
// than the selection is an Unexpected.
func (g GitCLI) CommitSelection(dir, message string, sel []GitEntry, staged []string) (CommitResult, error) {
	s, err := g.selection(dir, sel, staged, true)
	if err != nil {
		return CommitResult{}, err
	}
	old, err := g.head(dir)
	if err != nil {
		return CommitResult{}, err
	}
	f, err := os.CreateTemp("", "acc-selection-*.txt")
	if err != nil {
		return CommitResult{}, err
	}
	specs := f.Name()
	defer os.Remove(specs)
	_, werr := f.WriteString(strings.Join(s.specs, "\x00") + "\x00")
	if err := errors.Join(werr, f.Close()); err != nil {
		return CommitResult{}, err
	}
	res := g.exec(dir, []byte(message), []string{"GIT_LITERAL_PATHSPECS=1"}, "commit", []string{"-c", "user.useConfigOnly=true",
		"commit", "--only", "--file=-", "--pathspec-from-file=" + specs, "--pathspec-file-nul"}, nil, nil)
	if res.err != nil {
		return CommitResult{}, commitFailed(res)
	}
	r, err := g.madeOn(dir, old)
	if err != nil {
		return CommitResult{}, err
	}
	if what := g.holds(dir, r.Hash, old, s); what != "" {
		return r, Unexpected{Commit: r, What: what}
	}
	return r, nil
}

// holds says how commit hash, made on old ("" none), differs from s: ""
// when it changes exactly s's paths, each to the blob the check read from
// the index (plan 50, design D7).
func (g GitCLI) holds(dir, hash, old string, s selected) string {
	args := []string{"diff-tree", "-r", "-z", "--no-commit-id", "--name-only", "--no-renames"}
	if old == "" {
		args = append(args, "--root")
	}
	out, err := g.runRead(readLimit, dir, append(args, hash)...)
	if err != nil {
		return "its content could not be read (" + err.Error() + ")"
	}
	want := map[string]bool{}
	for _, p := range s.paths {
		want[p] = true
	}
	got := map[string]bool{}
	var extra, missing []string
	for _, p := range strings.Split(out, "\x00") {
		if p != "" {
			got[p] = true
			if !want[p] {
				extra = append(extra, p)
			}
		}
	}
	for _, p := range s.paths {
		if !got[p] {
			missing = append(missing, p)
		}
	}
	switch {
	case len(extra) > 0:
		return "it also holds " + some(extra) + ", which was not selected"
	case len(missing) > 0:
		return "it leaves out " + some(missing)
	}
	blobs, err := g.blobsAt(dir, hash, s.paths)
	if err != nil {
		return "its content could not be read (" + err.Error() + ")"
	}
	var changed []string
	for _, p := range s.paths {
		if b, ok := s.blobs[p]; ok && blobs[p] != b {
			changed = append(changed, p)
		}
	}
	if len(changed) > 0 {
		return some(changed) + " changed during the commit, which holds the newer content"
	}
	return ""
}

// some names the first of paths and how many more there are.
func some(paths []string) string {
	if len(paths) == 1 {
		return paths[0]
	}
	return fmt.Sprintf("%s and %d more", paths[0], len(paths)-1)
}

// PushTarget finds where the current branch is pushed, or why it cannot be
// (plan 45, design D9), from symbolic-ref, for-each-ref and remote: all
// reads, in the order detached HEAD, no remote, no upstream, a local
// upstream, a gone upstream, push settings sending the branch elsewhere.
func (g GitCLI) PushTarget(dir string) (PushTarget, error) {
	b, err := g.runRead(readLimit, dir, "symbolic-ref", "-q", "--short", "HEAD")
	var ge *gitError
	if errors.As(err, &ge) && ge.code == 1 {
		return PushTarget{}, Refusal{"detached", "HEAD is detached: there is no branch to push."}
	}
	if err != nil {
		return PushTarget{}, err
	}
	branch := strings.TrimRight(b, "\r\n")
	const format = "--format=%(refname)%00%(upstream)%00%(upstream:short)%00%(upstream:remotename)%00%(upstream:remoteref)%00" +
		"%(upstream:track,nobracket)%00%(push)%00%(push:remotename)%00%(push:short)"
	refs, err := g.runRead(readLimit, dir, "for-each-ref", format, "refs/heads/"+branch)
	if err != nil {
		return PushTarget{}, err
	}
	var f []string
	for _, l := range lines(refs) {
		// The pattern also matches the refs under the branch's name.
		if p := strings.Split(l, "\x00"); len(p) == 9 && p[0] == "refs/heads/"+branch {
			f = p
		}
	}
	rem, err := g.runRead(readLimit, dir, "remote")
	if err != nil {
		return PushTarget{}, err
	}
	remotes := lines(rem)
	if len(remotes) == 0 {
		return PushTarget{}, Refusal{"no-remote", "No remote is configured for this repository. Add one in a terminal: git remote add <name> <url>"}
	}
	if f == nil || f[1] == "" {
		name := "<remote>"
		if len(remotes) == 1 {
			name = remotes[0]
		}
		return PushTarget{}, Refusal{"no-upstream", fmt.Sprintf("The branch %s has no upstream. Set one in a terminal: git push -u %s %s", branch, name, branch)}
	}
	short, upRemote, upRef, track := f[2], f[3], f[4], f[5]
	if upRemote == "." {
		return PushTarget{}, Refusal{"local-upstream", fmt.Sprintf("The upstream of %s is the local branch %s: there is no remote to push to.", branch, short)}
	}
	if track == "gone" {
		return PushTarget{}, Refusal{"gone", fmt.Sprintf("The upstream %s is gone from its remote: push from a terminal to create it again.", short)}
	}
	// The push destination is git's remote-tracking name of it (%(push), "" when
	// the settings give none); %(push:remoteref) is empty in git 2.56, so the
	// destination is compared as that name, and its remote.
	pushRef, pushRemote, pushShort := f[6], f[7], f[8]
	switch {
	case pushRef == "":
		return PushTarget{}, Refusal{"elsewhere", fmt.Sprintf("Git's push settings do not push %s to its upstream %s: push from a terminal.", branch, short)}
	case pushRef != f[1] || pushRemote != upRemote:
		return PushTarget{}, Refusal{"elsewhere", fmt.Sprintf("Git's push settings send %s to %s, not to its upstream %s: push from a terminal.",
			branch, pushShort, short)}
	}
	t := PushTarget{Branch: branch, Remote: upRemote, RemoteRef: upRef, Upstream: short, tracking: f[1]}
	for _, w := range strings.Split(track, ", ") {
		if n, ok := strings.CutPrefix(w, "ahead "); ok {
			t.Ahead, _ = strconv.Atoi(n)
		} else if n, ok := strings.CutPrefix(w, "behind "); ok {
			t.Behind, _ = strconv.Atoi(n)
		}
	}
	return t, nil
}

// Push sends the current branch to its upstream, only that (plan 45, design
// D9): git push --porcelain <remote> refs/heads/<branch>:<upstream's ref>,
// after PushTarget has found nothing against it. Git never prompts on the
// console: GIT_TERMINAL_PROMPT=0 is the runner's, and ssh is told to use an
// askpass program that does not exist, unless the owner set one. No message
// shows a URL's user or password.
func (g GitCLI) Push(dir string, stop *GitStop) (PushResult, error) {
	t, err := g.PushTarget(dir)
	if err != nil {
		return PushResult{}, err
	}
	if t.Ahead == 0 {
		return PushResult{}, Refusal{"up-to-date", fmt.Sprintf("Nothing to push: %s is up to date with %s.", t.Branch, t.Upstream)}
	}
	env := []string{"SSH_ASKPASS_REQUIRE=force"}
	if os.Getenv("SSH_ASKPASS") == "" {
		env = append(env, "SSH_ASKPASS=agent-command-center-no-askpass")
	}
	var locks []string
	if stop != nil {
		// The lock files a push could hold, which a stop waits out (design
		// D15): the index's, HEAD's, packed-refs, the branch's and its
		// remote-tracking ref's.
		names := []string{"index.lock", "HEAD.lock", "packed-refs.lock", "refs/heads/" + t.Branch + ".lock"}
		if t.tracking != "" {
			names = append(names, t.tracking+".lock")
		}
		if locks, err = g.gitPaths(dir, names...); err != nil {
			return PushResult{}, err
		}
	}
	// push.followTags and push.recurseSubmodules would send more than the
	// branch: off for this call.
	res := g.exec(dir, nil, env, "push", []string{"-c", "push.followTags=false", "-c", "push.recurseSubmodules=no",
		"push", "--porcelain", t.Remote, "refs/heads/" + t.Branch + ":" + t.RemoteRef}, stop, locks)
	if stop.stopped() && res.err != nil {
		return PushResult{}, cancelledPush(locks)
	}
	refLine := false
	for _, l := range strings.Split(res.out, "\n") {
		l = strings.TrimSuffix(l, "\r")
		// A ref line: the flag, a tab, "<from>:<to>", a tab, the summary.
		if len(l) < 2 || l[1] != '\t' || !strings.ContainsRune(" +-*=!", rune(l[0])) {
			continue
		}
		parts := strings.SplitN(l[2:], "\t", 2)
		summary, reason := "", ""
		if len(parts) == 2 {
			summary = parts[1]
			if i := strings.LastIndex(summary, " ("); i >= 0 && strings.HasSuffix(summary, ")") {
				summary, reason = summary[:i], summary[i+2:len(summary)-1]
			}
		}
		if l[0] == '!' {
			return PushResult{}, PushRejected{
				Ref:            scrubURLs(parts[0]),
				Reason:         scrubURLs(reason),
				NonFastForward: summary == "[rejected]" && (reason == "non-fast-forward" || reason == "fetch first"),
				Remote:         summary == "[remote rejected]" || summary == "[remote failure]",
				summary:        scrubURLs(summary),
				upstream:       t.Upstream,
			}
		}
		refLine = true
	}
	if res.err != nil {
		text := scrubURLs(res.failed())
		var ge *gitError
		if errors.As(res.err, &ge) && text != "" {
			return PushResult{}, &gitError{code: ge.code, msg: text}
		}
		return PushResult{}, errors.New(scrubURLs(res.err.Error()))
	}
	if !refLine {
		return PushResult{}, fmt.Errorf("git push exited 0 but said nothing about %s", t.Upstream)
	}
	return PushResult{Target: t}, nil
}

// cancelledPush is the refusal of a push stopped before it ended (design
// D15): the remote may have taken it or not, and a lock file of locks git
// left is named, never deleted.
func cancelledPush(locks []string) Refusal {
	msg := "The push was cancelled. The remote may have received it: Push again to check."
	if left := heldLocks(locks); len(left) > 0 {
		msg += " git left " + strings.Join(left, ", ") + " behind: delete it once no git command runs in this repository."
	}
	return Refusal{"cancelled", msg}
}
