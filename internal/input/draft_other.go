//go:build !windows

package input

import "os"

// openExclusive: outside Windows a file cannot be held without advisory
// locks, so two dashboards on one project are not kept apart there (plan
// 33, design D1).
func openExclusive(path string, create bool) (*os.File, error) {
	flag := os.O_RDWR
	if create {
		flag |= os.O_CREATE | os.O_EXCL
	}
	return os.OpenFile(path, flag, 0o600)
}

// transient: no other system here refuses a rename for a file held open.
func transient(error) bool { return false }
