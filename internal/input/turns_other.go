//go:build !windows

package input

import (
	"fmt"
	"os"
	"syscall"
)

// makeStoreDir makes a store's folder, which only its owner can open (plan
// 65, design D3). Outside Windows a file cannot be held without advisory
// locks (draft_other.go), so a dashboard's sweep cannot tell a running
// dashboard's store from a dead one's: the review is Windows's.
func makeStoreDir(path string) error { return os.Mkdir(path, 0o700) }

// linkCount is the number of names the checked file has, its hard links
// included (plan 65, the review of 2026-10-29), from its status.
func linkCount(cf checkedFile) (uint32, error) {
	st, ok := cf.info.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, fmt.Errorf("no link count for %s", cf.real)
	}
	return uint32(st.Nlink), nil
}

// openShared opens path to read it.
func openShared(path string) (*os.File, error) { return os.Open(path) }
