package input

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// The Explorer's reading (plan 17, amendment 12): one folder at a time, when
// it is opened, and git's markers for the project, both read only.

// TreeEntry is an entry of a folder as the Explorer lists it.
type TreeEntry struct {
	Name string
	Dir  bool
}

// MaxDirEntries is how many entries a folder lists; a variable so tests can
// lower it.
var MaxDirEntries = 10_000

// ListDir lists the folder rel, relative to root with forward slashes, ""
// for root itself: its folders, then its files, each sorted without case;
// without .git, credential files (by name, never opened) or links and
// junctions; at most MaxDirEntries, truncated saying the limit cut it. The
// folder, its links resolved, must be inside root.
func ListDir(root, rel string) (entries []TreeEntry, truncated bool, err error) {
	dir := filepath.Join(root, filepath.FromSlash(rel))
	realRoot, err := realPath(root)
	if err != nil {
		return nil, false, &FileError{"the project cannot be read: " + root}
	}
	if !inside(realRoot, filepath.Clean(root), filepath.Clean(dir)) {
		return nil, false, &FileError{"outside the project: " + rel}
	}
	real, err := realPath(dir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, false, &FileError{"not found: " + rel}
		}
		return nil, false, &FileError{"cannot be read: " + rel}
	}
	if !inside(realRoot, realRoot, real) {
		return nil, false, &FileError{"outside the project: " + rel}
	}
	des, err := os.ReadDir(real)
	if err != nil && len(des) == 0 {
		if errors.Is(err, fs.ErrPermission) {
			return nil, false, &FileError{"access denied"}
		}
		return nil, false, &FileError{"cannot be read: " + err.Error()}
	}
	for _, d := range des {
		t := d.Type()
		switch {
		case t == fs.ModeDir: // not a link, a junction or another reparse point
			if !strings.EqualFold(d.Name(), ".git") {
				entries = append(entries, TreeEntry{Name: d.Name(), Dir: true})
			}
		case t.IsRegular() && !CredentialName(d.Name()):
			entries = append(entries, TreeEntry{Name: d.Name()})
		}
	}
	sort.SliceStable(entries, func(i, j int) bool {
		a, b := entries[i], entries[j]
		if a.Dir != b.Dir {
			return a.Dir
		}
		la, lb := strings.ToLower(a.Name), strings.ToLower(b.Name)
		if la != lb {
			return la < lb
		}
		return a.Name < b.Name
	})
	if len(entries) > MaxDirEntries {
		entries, truncated = entries[:MaxDirEntries:MaxDirEntries], true
	}
	return entries, truncated, nil
}

// GitMarks is git's view of a project's files (plan 17, amendment 12): every
// path relative to the project, with forward slashes; a folder's ends with
// "/".
type GitMarks struct {
	// Files is each changed path's marker: 'M' modified, 'A' added, 'R'
	// renamed, 'D' deleted, 'U' untracked (a folder: all of it), '!' in
	// conflict.
	Files map[string]byte
	// Ignored is every path git ignores (a folder: all of it).
	Ignored map[string]bool
}

// Status is git's markers for the project dir: one git status that writes
// nothing (no optional lock), its paths made relative to dir. Outside a
// repository it is NotRepoError; without git, ErrNoGit.
func (g GitCLI) Status(dir string) (GitMarks, error) {
	prefix, err := g.run(readLimit, dir, "rev-parse", "--show-prefix")
	if err != nil {
		if errors.Is(err, ErrNoGit) {
			return GitMarks{}, err
		}
		if strings.Contains(strings.ToLower(err.Error()), "not a git repository") {
			return GitMarks{}, NotRepoError{Dir: dir}
		}
		return GitMarks{}, err
	}
	out, err := g.run(readLimit, dir, "status", "--porcelain=v1", "-z", "--untracked-files=normal", "--ignored", "--", ".")
	if err != nil {
		return GitMarks{}, err
	}
	return ParseStatus(out, strings.TrimSpace(prefix)), nil
}

// ParseStatus reads git status --porcelain=v1 -z: "XY path", a rename or a
// copy followed by its source path, each ended by a NUL, the paths relative
// to the repository; prefix, the project's place in it, is taken off and a
// path outside it left out.
func ParseStatus(out, prefix string) GitMarks {
	gm := GitMarks{Files: map[string]byte{}, Ignored: map[string]bool{}}
	rel := func(p string) (string, bool) {
		if len(p) < len(prefix) || !strings.EqualFold(p[:len(prefix)], prefix) {
			return "", false
		}
		p = p[len(prefix):]
		return p, p != ""
	}
	recs := strings.Split(out, "\x00")
	for i := 0; i < len(recs); i++ {
		r := recs[i]
		if len(r) < 4 || r[2] != ' ' {
			continue
		}
		x, y := r[0], r[1]
		if x == 'R' || x == 'C' {
			i++ // the source path
		}
		p, ok := rel(r[3:])
		if !ok {
			continue
		}
		switch {
		case x == '!' && y == '!':
			gm.Ignored[p] = true
		case x == '?' && y == '?':
			gm.Files[p] = 'U'
		case x == 'U' || y == 'U' || x == 'A' && y == 'A' || x == 'D' && y == 'D':
			gm.Files[p] = '!'
		case x == 'R':
			gm.Files[p] = 'R'
		case x == 'A' || x == 'C':
			gm.Files[p] = 'A'
		case x == 'D' || y == 'D':
			gm.Files[p] = 'D'
		default:
			gm.Files[p] = 'M'
		}
	}
	return gm
}
