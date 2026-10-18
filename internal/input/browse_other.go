//go:build !windows

package input

import "io/fs"

// hiddenEntry: outside Windows only a dot name hides a directory.
func hiddenEntry(fs.DirEntry) bool { return false }

// drives: outside Windows there is one root, reached by going up.
func drives() []string { return nil }
