package input

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// GitStop lets the owner stop a running push (plan 45, design D15). The push
// gives it the way to stop its git while git runs. Git, stopped, leaves its
// lock files behind (probe P45a), so Stop first checks that none of those a
// push could hold exists; then it asks git's own process group to stop, a
// console Ctrl+Break on Windows, which ends git and its children (the
// transport, a credential helper) at once.
type GitStop struct {
	mu    sync.Mutex
	stop  func() (bool, string)
	asked chan struct{}
	once  sync.Once
}

// NewGitStop is a stop for one push.
func NewGitStop() *GitStop { return &GitStop{asked: make(chan struct{})} }

// stopBusy is Stop's answer while a lock file exists.
const stopBusy = "git is updating the repository: try again in a moment."

// Stop asks the running push to end: ok once asked; otherwise why it cannot
// be now, "" when no git runs. The way to stop runs under the lock, which
// git's end takes before letting its pid go (arm): a stop names the git
// that runs, or one that ended, never another process.
func (s *GitStop) Stop() (ok bool, why string) {
	if s == nil {
		return false, ""
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.stop == nil {
		return false, ""
	}
	if ok, why = s.stop(); ok {
		s.once.Do(func() { close(s.asked) })
	}
	return ok, why
}

// Asked is closed once a stop was asked.
func (s *GitStop) Asked() <-chan struct{} { return s.asked }

// Set gives the way to stop the git that runs now, nil once it ended. The
// push sets it; a test double sets its own.
func (s *GitStop) Set(f func() (bool, string)) {
	if s == nil {
		return
	}
	s.mu.Lock()
	s.stop = f
	s.mu.Unlock()
}

// stopped says a stop was asked.
func (s *GitStop) stopped() bool {
	if s == nil {
		return false
	}
	select {
	case <-s.asked:
		return true
	default:
		return false
	}
}

// arm gives s the way to stop the git of pid, which leads its own process
// group, and returns what takes it back once git ended. The pid is held
// meanwhile (holdPid), and taken back only after the way to stop: a
// Ctrl+Break sent to a pid that leads no group goes to that process's
// parent, the dashboard (probe P45d), and a pid freed by git's end could
// name such a process.
func (s *GitStop) arm(pid int, locks []string) (disarm func()) {
	if s == nil {
		return func() {}
	}
	release, err := holdPid(pid)
	if err != nil {
		s.Set(func() (bool, string) { return false, "git could not be stopped: " + err.Error() })
		return func() { s.Set(nil) }
	}
	s.Set(stopper(pid, locks))
	return func() {
		s.Set(nil)
		release()
	}
}

// stopper is the way to stop the git of pid: refused while one of locks
// exists, else its process group asked to stop.
func stopper(pid int, locks []string) func() (bool, string) {
	return func() (bool, string) {
		if len(heldLocks(locks)) > 0 {
			return false, stopBusy
		}
		if err := breakGroup(pid); err != nil {
			return false, "git could not be stopped: " + err.Error()
		}
		return true, ""
	}
}

// heldLocks is those of the lock files that exist.
func heldLocks(locks []string) []string {
	var held []string
	for _, l := range locks {
		if _, err := os.Lstat(l); err == nil {
			held = append(held, l)
		}
	}
	return held
}

// gitPaths is git's paths of names (git rev-parse --git-path), made
// absolute from dir where git gives them relative to it.
func (g GitCLI) gitPaths(dir string, names ...string) ([]string, error) {
	args := []string{"rev-parse"}
	for _, n := range names {
		args = append(args, "--git-path", n)
	}
	out, err := g.runRead(readLimit, dir, args...)
	if err != nil {
		return nil, err
	}
	var paths []string
	for _, l := range lines(out) {
		p := filepath.FromSlash(strings.TrimSpace(l))
		if !filepath.IsAbs(p) {
			p = filepath.Join(dir, p)
		}
		paths = append(paths, p)
	}
	return paths, nil
}
