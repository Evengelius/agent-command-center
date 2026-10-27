//go:build !windows

package input

import "path/filepath"

// realPath is p with every symbolic link resolved (plan 17, Reading a file).
func realPath(p string) (string, error) { return filepath.EvalSymlinks(p) }
