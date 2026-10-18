package input

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// DirEntry is a subdirectory as the project browser lists it (plan 15).
type DirEntry struct {
	Name   string
	Git    bool // it holds a .git: a work tree's root
	Hidden bool // a dot name, or hidden or system on Windows
	Link   bool // a symbolic link or a junction to a directory
}

// Lister reads directories for the project browser; tests pass a fake.
type Lister interface {
	// List is dir's subdirectories, sorted by name without case.
	List(dir string) ([]DirEntry, error)
	// Drives is the drive roots, "C:\" first; none outside Windows.
	Drives() []string
}

// Disk lists the file system.
type Disk struct{}

func (Disk) List(dir string) ([]DirEntry, error) {
	es, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var out []DirEntry
	for _, e := range es {
		full := filepath.Join(dir, e.Name())
		d := DirEntry{Name: e.Name()}
		if e.Type()&(fs.ModeSymlink|fs.ModeIrregular) != 0 {
			// A link or a junction (a reparse point since Go 1.23): listed
			// when it leads to a directory.
			fi, err := os.Stat(full)
			if err != nil || !fi.IsDir() {
				continue
			}
			d.Link = true
		} else if !e.IsDir() {
			continue
		}
		d.Hidden = strings.HasPrefix(d.Name, ".") || hiddenEntry(e)
		if _, err := os.Lstat(filepath.Join(full, ".git")); err == nil {
			d.Git = true // a directory, or a worktree's file
		}
		out = append(out, d)
	}
	SortEntries(out)
	return out, nil
}

func (Disk) Drives() []string { return drives() }

// SortEntries sorts by name without case, then with it.
func SortEntries(es []DirEntry) {
	sort.SliceStable(es, func(i, j int) bool {
		a, b := strings.ToLower(es[i].Name), strings.ToLower(es[j].Name)
		if a != b {
			return a < b
		}
		return es[i].Name < es[j].Name
	})
}
