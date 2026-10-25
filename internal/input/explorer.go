package input

import (
	"errors"
	"fmt"
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

// IsIgnored reports whether git ignores the project-relative path p: p
// itself, or a folder above it, is in Ignored. The files under an untracked
// folder that git ignores are listed there and not among the changes (plan
// 44, design D4).
func (m GitMarks) IsIgnored(p string) bool {
	if m.Ignored[p] || m.Ignored[p+"/"] {
		return true
	}
	for i := 0; i < len(p); i++ {
		if p[i] == '/' && m.Ignored[p[:i+1]] {
			return true
		}
	}
	return false
}

// Status is git's markers for the project dir: the markers of one Snapshot
// (plan 44, design D2), a single code path for the Explorer and the Source
// Control. Outside a repository it is NotRepoError; without git, ErrNoGit.
func (g GitCLI) Status(dir string) (GitMarks, error) {
	snap, err := g.Snapshot(dir)
	if err != nil {
		return GitMarks{}, err
	}
	return snap.Marks, nil
}

// Snapshot is git's view of the project dir for the Explorer's markers and
// the Source Control's lists (plan 44, design D2), read only: git rev-parse
// for the project's place in the repository, then one git status that writes
// nothing (no optional lock, no file system monitor), its paths made
// relative to dir; a detached HEAD costs a third call for its short id.
// Outside a repository it is NotRepoError; without git, ErrNoGit.
func (g GitCLI) Snapshot(dir string) (GitSnapshot, error) {
	out, err := g.runRead(readLimit, dir, "rev-parse", "--show-prefix", "--show-toplevel")
	if err != nil {
		if errors.Is(err, ErrNoGit) {
			return GitSnapshot{}, err
		}
		if strings.Contains(strings.ToLower(err.Error()), "not a git repository") {
			return GitSnapshot{}, NotRepoError{Dir: dir}
		}
		return GitSnapshot{}, err
	}
	prefix, top, _ := strings.Cut(out, "\n")
	prefix, top = strings.TrimSuffix(prefix, "\r"), strings.TrimRight(top, "\r\n")
	if top == "" {
		return GitSnapshot{}, fmt.Errorf("git has no top level for %s", dir)
	}
	st, err := g.runRead(readLimit, dir, "status", "--porcelain=v1", "-z", "--branch", "--untracked-files=normal", "--ignored", "--", ".")
	if err != nil {
		return GitSnapshot{}, err
	}
	snap := GitSnapshot{Top: filepath.FromSlash(top), Prefix: prefix}
	snap.Entries, snap.Branch, snap.Marks = ParseSnapshot(st, prefix)
	for _, r := range strings.Split(st, "\x00") {
		// The project's folder untracked as a whole: its one record, the
		// prefix itself, which ParseSnapshot leaves out.
		if p, ok := strings.CutPrefix(r, "?? "); ok && prefix != "" && strings.EqualFold(p, prefix) {
			snap.Untracked = true
		}
	}
	if snap.Branch.Detached {
		id, err := g.runRead(readLimit, dir, "rev-parse", "--short", "HEAD")
		if err != nil {
			return GitSnapshot{}, err
		}
		snap.Branch.Head = strings.TrimSpace(id)
	}
	return snap, nil
}

// ParseStatus reads git status --porcelain=v1 -z: the Explorer's markers
// of ParseSnapshot.
func ParseStatus(out, prefix string) GitMarks {
	_, _, marks := ParseSnapshot(out, prefix)
	return marks
}

// ParseSnapshot reads git status --porcelain=v1 -z --branch (plan 44, design
// D2): the branch record "## ..." if there is one, then "XY path", a rename
// or a copy followed by its source path, each ended by a NUL, the paths
// relative to the repository. prefix, the project's place in it, is taken
// off and a path outside it left out. The entries are the changes in git's
// order, the ignored never among them; the markers are the Explorer's, one
// for a path with X and Y merged, a rename's source skipped, the ignored
// paths apart.
func ParseSnapshot(out, prefix string) (entries []GitEntry, branch GitBranch, marks GitMarks) {
	marks = GitMarks{Files: map[string]byte{}, Ignored: map[string]bool{}}
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
		if s, ok := strings.CutPrefix(r, "## "); ok { // no entry starts with two #
			branch = parseBranch(s)
			continue
		}
		if len(r) < 4 || r[2] != ' ' {
			continue
		}
		x, y := r[0], r[1]
		from := ""
		if x == 'R' || x == 'C' || y == 'R' || y == 'C' {
			i++ // the source path
			if i < len(recs) {
				from = recs[i]
			}
		}
		p, ok := rel(r[3:])
		if !ok {
			continue
		}
		switch {
		case x == '!' && y == '!':
			marks.Ignored[p] = true
			continue
		case x == '?' && y == '?':
			marks.Files[p] = 'U'
		case x == 'U' || y == 'U' || x == 'A' && y == 'A' || x == 'D' && y == 'D':
			marks.Files[p] = '!'
		case x == 'R':
			marks.Files[p] = 'R'
		case x == 'A' || x == 'C':
			marks.Files[p] = 'A'
		case x == 'D' || y == 'D':
			marks.Files[p] = 'D'
		default:
			marks.Files[p] = 'M'
		}
		e := GitEntry{X: x, Y: y, Path: p, RepoPath: r[3:], Orig: from, RepoOrig: from}
		if o, ok := rel(from); ok {
			e.Orig = o // a source outside the project, which git does not list, stays as it wrote it
		}
		entries = append(entries, e)
	}
	return entries, branch, marks
}

// parseBranch reads the branch record of git status --branch, after its
// "## ": "main", "main...origin/main [ahead 1]", "HEAD (no branch)" or "No
// commits yet on main" (git before 2.14 wrote "Initial commit on").
func parseBranch(s string) GitBranch {
	name := func(s string) string {
		n, _, _ := strings.Cut(s, "...") // a branch name never holds "..."
		return n
	}
	for _, lead := range []string{"No commits yet on ", "Initial commit on "} {
		if rest, ok := strings.CutPrefix(s, lead); ok {
			return GitBranch{Name: name(rest), NoCommits: true}
		}
	}
	if s == "HEAD (no branch)" {
		return GitBranch{Name: "HEAD", Detached: true}
	}
	return GitBranch{Name: name(s)}
}
