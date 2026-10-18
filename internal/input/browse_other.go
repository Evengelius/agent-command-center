//go:build !windows

package input

import (
	"io/fs"
	"path/filepath"
)

// hiddenEntry: outside Windows only a dot name hides a directory.
func hiddenEntry(fs.DirEntry) bool { return false }

// drives: outside Windows there is one root, reached by going up.
func drives() []string { return nil }

// realPath is p with every symbolic link resolved (plan 17, Reading a file).
func realPath(p string) (string, error) { return filepath.EvalSymlinks(p) }
