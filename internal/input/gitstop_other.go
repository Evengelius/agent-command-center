//go:build !windows

package input

import (
	"errors"
	"os/exec"
	"syscall"
)

// ownGroup starts cmd in a process group of its own, so that the stop
// reaches git and its children and nothing else (plan 45, design D15).
func ownGroup(cmd *exec.Cmd) { cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true} }

// breakGroup interrupts the process group pid leads. Group 0 is never sent
// to.
func breakGroup(pid int) error {
	if pid <= 0 {
		return errors.New("no process")
	}
	return syscall.Kill(-pid, syscall.SIGINT)
}

// holdPid has nothing to hold: a signal to -pid reaches only the members of
// pid's group, never its parent.
func holdPid(int) (release func(), err error) { return func() {}, nil }
