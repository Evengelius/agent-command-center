//go:build !windows

package codex

import "syscall"

// alive reports whether a process id still runs.
func alive(pid int) bool { return syscall.Kill(pid, 0) == nil }
