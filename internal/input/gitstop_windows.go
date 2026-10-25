//go:build windows

package input

import (
	"errors"
	"os/exec"
	"syscall"

	"golang.org/x/sys/windows"
)

// ownGroup starts cmd in a process group of its own, so that a console
// Ctrl+Break reaches git and its children and nothing else (plan 45,
// design D15).
func ownGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: windows.CREATE_NEW_PROCESS_GROUP}
}

// breakGroup sends a console Ctrl+Break to the process group pid leads.
// Group 0 would reach every process of the console, the dashboard's own:
// never sent. A pid that leads no group sends it to that process's parent
// (probe P45d): pid must be a group's root, held (holdPid).
func breakGroup(pid int) error {
	if pid <= 0 {
		return errors.New("no process")
	}
	return windows.GenerateConsoleCtrlEvent(windows.CTRL_BREAK_EVENT, uint32(pid))
}

// holdPid keeps pid naming its process until release: Windows gives a pid
// to another process only once every handle to the one it names is
// closed. Its group, once it exited, reaches only the members left
// (probe P45e).
func holdPid(pid int) (release func(), err error) {
	h, err := windows.OpenProcess(windows.SYNCHRONIZE, false, uint32(pid))
	if err != nil {
		return nil, err
	}
	return func() { _ = windows.CloseHandle(h) }, nil
}
