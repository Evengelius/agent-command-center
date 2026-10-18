package input

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Worktree is one worktree of a repository, as the Worktrees view lists it
// (plan 16).
type Worktree struct {
	Path    string // in the system's form
	Branch  string // "" when its HEAD is detached
	Head    string
	Main    bool // the repository's main worktree, listed first
	Bare    bool
	Locked  bool
	Missing bool   // its directory is gone (git calls it prunable)
	Changes int    // uncommitted changes: lines of git status --porcelain
	Ahead   int    // commits ahead of the main worktree's branch; -1 unknown
	Problem string // its state could not be read: git's reason
}

// Git runs the worktree operations of the Worktrees view (plan 16); tests
// pass a fake.
type Git interface {
	// Worktrees lists the worktrees of the repository holding dir, the main
	// one first.
	Worktrees(dir string) ([]Worktree, error)
	// Add creates a worktree at path for branch, an existing branch or a new
	// one from the main worktree's HEAD.
	Add(main, path, branch string) error
	// Diff is dir's changes against its merge base with base (committed and
	// uncommitted, then its untracked files), or its uncommitted changes
	// when base is "", at most DiffLimit lines.
	Diff(dir, base string) ([]string, error)
	// Merge merges branch into the main worktree; a conflict is aborted.
	Merge(main, branch string) error
	// Remove removes the worktree at path, then deletes branch if git finds
	// it merged; the note says what became of the branch.
	Remove(main, path, branch string) (note string, err error)
}

// ErrNoGit is every call's error when git is not on the PATH.
var ErrNoGit = errors.New("git not found: worktrees unavailable")

// NotRepoError is a directory outside any git repository.
type NotRepoError struct{ Dir string }

func (e NotRepoError) Error() string { return "Not a git repository: " + e.Dir }

// DiffLimit is the most lines Diff returns; then "… N lines left out".
var DiffLimit = 5000

// The time limits of a git call: a read, and an operation the owner asked
// for (a merge may run hooks).
const (
	readLimit = 15 * time.Second
	opLimit   = 2 * time.Minute
)

// GitCLI runs the git found on the PATH. It never passes --force, -f or -D:
// git's own refusals stand (brief, amendment 10).
type GitCLI struct {
	Trace func(args []string) // sees every call's arguments; for tests
}

// gitError is git's failure: its exit code and what it said.
type gitError struct {
	code int
	msg  string
}

func (e *gitError) Error() string { return e.msg }

// run is git -C dir args, with no standard input, no prompt and no
// optional lock (a read never writes), in English, within limit.
func (g GitCLI) run(limit time.Duration, dir string, args ...string) (string, error) {
	exe, err := exec.LookPath("git")
	if err != nil {
		return "", ErrNoGit
	}
	full := append([]string{"-C", dir, "-c", "core.quotepath=off"}, args...)
	if g.Trace != nil {
		g.Trace(full)
	}
	ctx, cancel := context.WithTimeout(context.Background(), limit)
	defer cancel()
	cmd := exec.CommandContext(ctx, exe, full...)
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0", "GIT_OPTIONAL_LOCKS=0",
		"GIT_MERGE_AUTOEDIT=no", "LC_ALL=C", "LANGUAGE=C")
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	err = cmd.Run()
	if ctx.Err() == context.DeadlineExceeded {
		return "", fmt.Errorf("git %s took more than %s", args[0], limit)
	}
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		msg := strings.Join(strings.Fields(strings.TrimSpace(errb.String())), " ")
		if msg == "" {
			msg = fmt.Sprintf("git %s failed (exit %d)", args[0], ee.ExitCode())
		}
		return out.String(), &gitError{code: ee.ExitCode(), msg: msg}
	}
	if err != nil {
		return "", err
	}
	return out.String(), nil
}

func (g GitCLI) Worktrees(dir string) ([]Worktree, error) {
	out, err := g.run(readLimit, dir, "worktree", "list", "--porcelain")
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "not a git repository") {
			return nil, NotRepoError{Dir: dir}
		}
		return nil, err
	}
	wts := parseWorktrees(out)
	if len(wts) == 0 {
		return nil, NotRepoError{Dir: dir}
	}
	main := wts[0]
	for i := range wts {
		w := &wts[i]
		w.Ahead = -1
		if w.Bare || w.Missing {
			continue
		}
		st, err := g.run(readLimit, w.Path, "status", "--porcelain")
		if err != nil {
			w.Problem = err.Error()
			continue
		}
		w.Changes = len(lines(st))
		if w.Main || w.Branch == "" || main.Branch == "" {
			continue
		}
		n, err := g.run(readLimit, main.Path, "rev-list", "--count", "refs/heads/"+main.Branch+"..refs/heads/"+w.Branch)
		if err != nil {
			w.Problem = err.Error()
			continue
		}
		if v, err := strconv.Atoi(strings.TrimSpace(n)); err == nil {
			w.Ahead = v
		}
	}
	return wts, nil
}

// parseWorktrees reads git worktree list --porcelain: one block per
// worktree, the main one first.
func parseWorktrees(out string) []Worktree {
	var wts []Worktree
	for _, block := range strings.Split(strings.ReplaceAll(out, "\r\n", "\n"), "\n\n") {
		var w Worktree
		for _, l := range strings.Split(block, "\n") {
			k, v, _ := strings.Cut(l, " ")
			switch k {
			case "worktree":
				w.Path = filepath.Clean(filepath.FromSlash(v))
			case "HEAD":
				w.Head = v
			case "branch":
				w.Branch = strings.TrimPrefix(v, "refs/heads/")
			case "bare":
				w.Bare = true
			case "locked":
				w.Locked = true
			case "prunable":
				w.Missing = true
			}
		}
		if w.Path != "" {
			w.Main = len(wts) == 0
			wts = append(wts, w)
		}
	}
	return wts
}

// lines is s's non-empty lines.
func lines(s string) []string {
	var out []string
	for _, l := range strings.Split(strings.ReplaceAll(s, "\r\n", "\n"), "\n") {
		if l != "" {
			out = append(out, l)
		}
	}
	return out
}

// WorktreePath is where a new worktree for branch goes:
// <main>.worktrees\<name>, beside the repository (amendment 10). name is the
// branch name's last segment, every character but letters, digits, ".", "_"
// and "-" replaced by "-"; "" when nothing is left.
func WorktreePath(main, branch string) string {
	seg := branch[strings.LastIndex(branch, "/")+1:]
	name := []rune(seg)
	for i, r := range name {
		ok := r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '.' || r == '_' || r == '-'
		if !ok {
			name[i] = '-'
		}
	}
	n := strings.Trim(string(name), ".")
	if strings.Trim(n, "-") == "" {
		return ""
	}
	return filepath.Join(filepath.Clean(main)+".worktrees", n)
}

func (g GitCLI) Add(main, path, branch string) error {
	if _, err := g.run(readLimit, main, "check-ref-format", "--branch", branch); err != nil {
		if errors.Is(err, ErrNoGit) {
			return err
		}
		return fmt.Errorf("not a valid branch name: %q", branch)
	}
	if path == "" {
		return fmt.Errorf("no directory name in %q", branch)
	}
	if _, err := os.Lstat(path); err == nil {
		return fmt.Errorf("already exists: %s", path)
	}
	_, err := g.run(readLimit, main, "show-ref", "--verify", "--quiet", "refs/heads/"+branch)
	var ge *gitError
	switch {
	case err == nil:
		_, err = g.run(opLimit, main, "worktree", "add", path, branch)
	case errors.As(err, &ge) && ge.code == 1:
		_, err = g.run(opLimit, main, "worktree", "add", "-b", branch, path, "HEAD")
	}
	return err
}

func (g GitCLI) Diff(dir, base string) ([]string, error) {
	from := "HEAD"
	if base != "" {
		mb, err := g.run(readLimit, dir, "merge-base", "refs/heads/"+base, "HEAD")
		if err != nil {
			return nil, err
		}
		from = strings.TrimSpace(mb)
	}
	d, err := g.run(readLimit, dir, "diff", "--no-color", "--no-ext-diff", "--no-textconv", from, "--")
	if err != nil {
		return nil, err
	}
	out := lines(d)
	st, err := g.run(readLimit, dir, "status", "--porcelain")
	if err != nil {
		return nil, err
	}
	var untracked []string
	for _, l := range lines(st) {
		if p, ok := strings.CutPrefix(l, "?? "); ok {
			untracked = append(untracked, "  "+p)
		}
	}
	if len(untracked) > 0 {
		out = append(append(out, "Untracked files:"), untracked...)
	}
	if len(out) > DiffLimit {
		left := len(out) - DiffLimit
		out = append(out[:DiffLimit:DiffLimit], fmt.Sprintf("… %d lines left out", left))
	}
	return out, nil
}

func (g GitCLI) Merge(main, branch string) error {
	_, err := g.run(opLimit, main, "merge", "--no-edit", branch)
	if err == nil || errors.Is(err, ErrNoGit) {
		return err
	}
	if _, merging := g.run(readLimit, main, "rev-parse", "-q", "--verify", "MERGE_HEAD"); merging != nil {
		return err // git refused before merging anything
	}
	if _, aerr := g.run(opLimit, main, "merge", "--abort"); aerr != nil {
		return fmt.Errorf("conflict, and git merge --abort failed: %v; the main worktree needs attention", aerr)
	}
	return errors.New("conflict: merge aborted, main unchanged")
}

func (g GitCLI) Remove(main, path, branch string) (string, error) {
	if _, err := g.run(opLimit, main, "worktree", "remove", path); err != nil {
		return "", err
	}
	if branch == "" {
		return "", nil
	}
	if _, err := g.run(opLimit, main, "branch", "-d", branch); err != nil {
		return "branch kept: " + err.Error(), nil
	}
	return "branch " + branch + " deleted", nil
}
