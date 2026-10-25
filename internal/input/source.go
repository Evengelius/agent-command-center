package input

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// The Source Control's reading (plan 44, brief amendment 45): the project's
// Git changes, staged and unstaged apart, and the two sides of each file's
// diff, all read only. Every git call is rev-parse, status or cat-file, with
// no optional lock and no file system monitor; the index and the working
// tree are never touched (design D10). The lines are compared by
// internal/diff, which is pure.

// GitEntry is one change of git status, as the Source Control lists it
// (plan 44, design D2).
type GitEntry struct {
	// X and Y are the two letters of git's porcelain v1: X the index against
	// HEAD, Y the working tree against the index; ' ' for none. '?', '?' is
	// an untracked path; a conflict has a 'U' on either side, or is "AA" or
	// "DD".
	X, Y byte
	// Path is relative to the project, with forward slashes; an untracked
	// folder ends with "/". Orig is the source of a rename or a copy, ""
	// otherwise.
	Path, Orig string
	// RepoPath and RepoOrig are the same relative to the repository's top
	// level, as git's objects name them (for cat-file).
	RepoPath, RepoOrig string
}

// GitBranch is the branch line of git status (plan 44, design D4).
type GitBranch struct {
	// Name is the branch; "HEAD" while detached.
	Name string
	// Detached says HEAD is at a commit, not on a branch; Head is then its
	// short id, "" otherwise.
	Detached bool
	// NoCommits says the branch has no commit yet.
	NoCommits bool
	Head      string
}

// GitSnapshot is git's view of a project at one moment (plan 44, design D2):
// the Explorer's markers and the Source Control's entries come from the same
// read, so that they never disagree.
type GitSnapshot struct {
	// Marks are the Explorer's, exactly as Status gives them.
	Marks GitMarks
	// Entries are the changes in git's order, the ignored never among them.
	Entries []GitEntry
	Branch  GitBranch
	// Top is the repository's top level, in the system's form; Prefix is the
	// project's place in the repository, "" or "dir/".
	Top, Prefix string
	// Untracked says the project's folder itself is untracked: git names it
	// once, by Prefix, and none of its files apart (plan 44, design D11).
	Untracked bool
}

// DiffSide is one side of a file's diff, read raw (plan 44, design D6): no
// filter, no text conversion.
type DiffSide struct {
	// Name says where it was read: "HEAD", "index", "ours" (a conflict's
	// stage 2), "working tree"; "" when the comparison has no such side (an
	// untracked file's old one).
	Name string
	// Path is the file's path in the repository, as read on this side.
	Path string
	// ID is the blob's id, "" for the working tree or no side.
	ID string
	// Size is the file's size in bytes, told when Data is nil too: 0 for a
	// Missing side.
	Size int64
	// Data is the content; nil when the side is Missing, Binary, LFS or
	// TooLarge, an empty slice for an empty file.
	Data []byte
	// Missing says there is no file on this side; Binary a NUL in its first
	// 8,000 bytes; LFS a Git LFS pointer, not the content; TooLarge more than
	// MaxFileSize bytes.
	Missing, Binary, LFS, TooLarge bool
	// ModTime is the working tree file's time of modification.
	ModTime time.Time
}

// DiffPair is the two sides of one comparison.
type DiffPair struct {
	Old, New DiffSide
	// Conflict names the unmerged state of a conflict: "both modified",
	// "added by us", "added by them", "deleted by us", "deleted by them",
	// "both added" or "both deleted"; "" for any other entry.
	Conflict string
}

// SourceGit is what the Source Control reads from git; tests pass a fake.
type SourceGit interface {
	// Snapshot is git's view of the project dir.
	Snapshot(dir string) (GitSnapshot, error)
	// Sides reads the two sides of the diff of entry e of snap, the project
	// being dir: the index against HEAD when staged, the working tree against
	// the index otherwise.
	Sides(dir string, snap GitSnapshot, e GitEntry, staged bool) (DiffPair, error)
}

var _ SourceGit = GitCLI{}

// ConflictName is the unmerged state git's two letters X and Y name, "" if
// they name none (plan 44, design D6).
func ConflictName(x, y byte) string {
	switch string([]byte{x, y}) {
	case "UU":
		return "both modified"
	case "AU":
		return "added by us"
	case "UA":
		return "added by them"
	case "DU":
		return "deleted by us"
	case "UD":
		return "deleted by them"
	case "AA":
		return "both added"
	case "DD":
		return "both deleted"
	}
	return ""
}

// diffProbe is how many leading bytes are searched for a NUL: git's own
// (plan 44, design D6).
const diffProbe = 8000

// lfsHeader starts a Git LFS pointer file.
var lfsHeader = []byte("version https://git-lfs.github.com/spec/v1")

// sideSource is a place to read one side from: a blob named spec, which is
// called name and path in the DiffSide it makes.
type sideSource struct{ name, path, spec string }

// Sides reads the two sides of e's diff (plan 44, design D6). The blobs of
// HEAD, the index and a conflict's stage 2 come from one git cat-file
// --batch, raw; the working file is read from the disk with the viewer's
// refusals: inside the project, no credential file, at most MaxFileSize
// bytes (more is TooLarge). Staged is HEAD, at the original path of a
// rename, against the index; unstaged is the index (stage 0) against the
// working file; an untracked path is nothing against the working file; a
// conflict is "ours" (stage 2, else HEAD) against the working file, named in
// Conflict, never called resolved. A credential file is ErrCredential and
// opens nothing; any other refusal is a *FileError.
func (g GitCLI) Sides(dir string, snap GitSnapshot, e GitEntry, staged bool) (DiffPair, error) {
	if e.RepoPath == "" || strings.HasSuffix(e.RepoPath, "/") {
		return DiffPair{}, &FileError{"a folder, not a file"}
	}
	for _, p := range []string{e.RepoPath, e.RepoOrig} {
		if p == "" {
			continue
		}
		if !repoRelative(p) {
			return DiffPair{}, &FileError{"outside the repository: " + p}
		}
		if CredentialName(path.Base(p)) {
			return DiffPair{}, ErrCredential
		}
	}
	if snap.Top == "" {
		return DiffPair{}, errors.New("the repository's top level is unknown")
	}

	pair := DiffPair{Conflict: ConflictName(e.X, e.Y)}
	p := e.RepoPath
	// Each side is made by the first of its sources git has, or is known missing
	// (the old side of an added file, the new side of a deleted one) or is
	// the working file. ":0:" names the index's stage 0 outright: a file
	// called "2:x" is not stage 2 of "x".
	var oldSrc, newSrc []sideSource
	working := false
	switch {
	case pair.Conflict != "":
		oldSrc = []sideSource{{"ours", p, ":2:" + p}, {"HEAD", p, "HEAD:" + p}}
		working = true
	case e.X == '?' && e.Y == '?':
		pair.Old = DiffSide{Path: p, Missing: true}
		working = true
	case staged:
		from := p // the source of a rename or a copy is where HEAD has the file
		if e.RepoOrig != "" {
			from = e.RepoOrig
		}
		if e.X == 'A' {
			pair.Old = DiffSide{Name: "HEAD", Path: from, Missing: true}
		} else {
			oldSrc = []sideSource{{"HEAD", from, "HEAD:" + from}}
		}
		if e.X == 'D' {
			pair.New = DiffSide{Name: "index", Path: p, Missing: true}
		} else {
			newSrc = []sideSource{{"index", p, ":0:" + p}}
		}
	default:
		oldSrc = []sideSource{{"index", p, ":0:" + p}}
		if e.Y == 'D' {
			pair.New = DiffSide{Name: "working tree", Path: p, Missing: true}
		} else {
			working = true
		}
	}

	var specs []string
	for _, c := range append(append([]sideSource(nil), oldSrc...), newSrc...) {
		specs = append(specs, c.spec)
	}
	if len(specs) > 0 {
		objs, err := g.catFile(dir, specs)
		if err != nil {
			return DiffPair{}, err
		}
		if len(oldSrc) > 0 {
			pair.Old = firstSide(oldSrc, objs[:len(oldSrc)])
		}
		if len(newSrc) > 0 {
			pair.New = firstSide(newSrc, objs[len(oldSrc):])
		}
	}
	if working {
		w, err := workingSide(dir, snap.Top, p)
		if err != nil {
			return DiffPair{}, err
		}
		pair.New = w
	}
	return pair, nil
}

// repoRelative reports whether p can name a file in the repository: no
// absolute path, no ".." step.
func repoRelative(p string) bool {
	if strings.HasPrefix(p, "/") || strings.ContainsRune(p, 0) || filepath.IsAbs(p) {
		return false
	}
	for _, step := range strings.Split(p, "/") {
		if step == ".." {
			return false
		}
	}
	return true
}

// firstSide is the side the first source git has makes; with none it is
// missing, as the last source names it.
func firstSide(sources []sideSource, objs []object) DiffSide {
	for i, c := range sources {
		if !objs[i].missing {
			return blobSide(c.name, c.path, objs[i])
		}
	}
	last := sources[len(sources)-1]
	return DiffSide{Name: last.name, Path: last.path, Missing: true}
}

// object is what git cat-file --batch says of one name.
type object struct {
	id       string
	size     int64
	data     []byte // a blob's content; nil when missing, too large or not a file
	missing  bool   // no such object, or not a file
	tooLarge bool   // a blob over MaxFileSize, skipped
}

// catFile reads the objects named by specs ("HEAD:path", ":path", ":2:path")
// with one git cat-file --batch fed on its standard input: raw, with no
// filter and no text conversion. A name with a line break is sent NUL
// terminated (git cat-file -z).
func (g GitCLI) catFile(dir string, specs []string) ([]object, error) {
	args, sep := []string{"cat-file", "--batch"}, "\n"
	for _, s := range specs {
		if strings.Contains(s, "\n") {
			args, sep = append(args, "-z"), "\x00"
			break
		}
	}
	var objs []object
	err := g.stream(readLimit, dir, strings.NewReader(strings.Join(specs, sep)+sep), func(r *bufio.Reader) error {
		for range specs {
			o, err := readObject(r)
			if err != nil {
				return err
			}
			objs = append(objs, o)
		}
		return nil
	}, args...)
	return objs, err
}

// readObject reads one answer of git cat-file --batch: "<id> <type> <size>"
// and the content, or "<name> missing", or "<id> submodule" for a path that
// names another repository's commit. A blob over MaxFileSize is skipped, not
// read; so is anything but a blob, which is no file.
func readObject(r *bufio.Reader) (object, error) {
	ended := errors.New("git cat-file ended before it answered")
	line, err := r.ReadString('\n')
	if err != nil {
		return object{}, ended
	}
	line = strings.TrimSuffix(line, "\n")
	for !isObjectHeader(line) && !strings.HasSuffix(line, " missing") && !strings.HasSuffix(line, " ambiguous") {
		// A name with a line break in it that git does not have is repeated
		// as it was sent, over several lines.
		more, err := r.ReadString('\n')
		if err != nil {
			return object{}, ended
		}
		line += "\n" + strings.TrimSuffix(more, "\n")
	}
	if strings.HasSuffix(line, " missing") || strings.HasSuffix(line, " ambiguous") {
		return object{missing: true}, nil
	}
	if id, ok := strings.CutSuffix(line, " submodule"); ok {
		// A submodule is told as git diff tells it.
		data := []byte("Subproject commit " + id + "\n")
		return object{id: id, size: int64(len(data)), data: data}, nil
	}
	f := strings.Fields(line)
	size, err := strconv.ParseInt(f[2], 10, 64)
	if err != nil || size < 0 {
		return object{}, fmt.Errorf("git cat-file said %q", line)
	}
	o := object{id: f[0], size: size}
	o.missing = f[1] != "blob" // a tree or a tag is no file
	o.tooLarge = !o.missing && size > MaxFileSize
	if o.missing || o.tooLarge {
		if _, err := io.CopyN(io.Discard, r, size+1); err != nil { // the content and its line feed
			return object{}, ended
		}
		return o, nil
	}
	o.data = make([]byte, size)
	if _, err := io.ReadFull(r, o.data); err != nil {
		return object{}, ended
	}
	if _, err := r.ReadByte(); err != nil { // the line feed after the content
		return object{}, ended
	}
	return o, nil
}

// isObjectHeader reports whether line is "<id> <type> <size>" or "<id>
// submodule", an object git has.
func isObjectHeader(line string) bool {
	f := strings.Split(line, " ")
	if len(f) < 2 || len(f) > 3 || len(f[0]) < 40 || strings.Trim(f[0], "0123456789abcdef") != "" {
		return false
	}
	if len(f) == 2 {
		return f[1] == "submodule"
	}
	_, err := strconv.ParseUint(f[2], 10, 63)
	return err == nil && (f[1] == "blob" || f[1] == "tree" || f[1] == "commit" || f[1] == "tag")
}

// blobSide is the side an object makes.
func blobSide(name, path string, o object) DiffSide {
	s := DiffSide{Name: name, Path: path}
	switch {
	case o.missing:
		s.Missing = true
	case o.tooLarge:
		s.ID, s.Size, s.TooLarge = o.id, o.size, true
	default:
		s.ID, s.Size = o.id, int64(len(o.data))
		classify(&s, o.data)
	}
	return s
}

// classify sets s's Data from the content data, or says why it has none: a
// Git LFS pointer, or a NUL in the first diffProbe bytes.
func classify(s *DiffSide, data []byte) {
	switch {
	case bytes.HasPrefix(data, lfsHeader):
		s.LFS = true
	case bytes.IndexByte(data[:min(len(data), diffProbe)], 0) >= 0:
		s.Binary = true
	default:
		s.Data = data
	}
}

// workingSide reads the file repoPath of the repository whose top level is
// top, which must be inside the project root: the viewer's refusals apply
// (plan 17, Reading a file). A link is read as git reads it, its target path
// as the content; what it points to is never opened.
func workingSide(root, top, repoPath string) (DiffSide, error) {
	s := DiffSide{Name: "working tree", Path: repoPath}
	abs := filepath.Join(top, filepath.FromSlash(repoPath))
	info, err := os.Lstat(abs)
	switch {
	case errors.Is(err, fs.ErrNotExist) || errors.Is(err, syscall.ENOTDIR):
		s.Missing = true
		return s, nil
	case err != nil:
		return s, &FileError{"cannot be read: " + repoPath}
	}
	s.ModTime = info.ModTime()
	realRoot, err := realPath(root)
	if err != nil {
		return s, &FileError{"the project cannot be read: " + root}
	}
	switch mode := info.Mode(); {
	case mode&fs.ModeSymlink != 0:
		dir, err := realPath(filepath.Dir(abs))
		if err != nil || !inside(realRoot, realRoot, dir) {
			return s, &FileError{"outside the project: " + repoPath}
		}
		target, err := os.Readlink(abs)
		if err != nil {
			return s, &FileError{"cannot be read: " + repoPath}
		}
		s.Size = int64(len(target))
		classify(&s, []byte(target))
		return s, nil
	case mode.IsDir():
		return s, &FileError{"a directory, not a file"}
	case !mode.IsRegular():
		return s, &FileError{"not a regular file: " + repoPath}
	}
	real, err := realPath(abs)
	if err != nil {
		return s, &FileError{"cannot be read: " + repoPath}
	}
	if CredentialName(filepath.Base(real)) {
		return s, ErrCredential
	}
	if !inside(realRoot, realRoot, real) {
		return s, &FileError{"outside the project: " + repoPath}
	}
	s.Size = info.Size()
	if s.Size > MaxFileSize {
		s.TooLarge = true
		return s, nil
	}
	f, err := os.Open(real)
	if err != nil {
		return s, &FileError{"cannot be read: " + repoPath}
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, MaxFileSize+1))
	if err != nil {
		return s, &FileError{"cannot be read: " + repoPath}
	}
	s.Size = int64(len(data))
	if s.Size > MaxFileSize {
		s.TooLarge = true
		return s, nil
	}
	classify(&s, data)
	return s, nil
}

// UntrackedFiles lists the files under the untracked folder rel of the
// project root (plan 44, design D4): recursive, with project-relative paths
// and forward slashes, sorted, at most max (a limit under 1 is MaxListed);
// truncated says the limit cut it. It follows ListDir's rules: no .git, no
// link or junction, no credential file. The folder rel itself is an error
// when it cannot be listed; one below it is left out.
func UntrackedFiles(root, rel string, max int) (files []string, truncated bool, err error) {
	if max < 1 {
		max = MaxListed
	}
	stack := []string{strings.Trim(filepath.ToSlash(rel), "/")}
	full := false
	for first := true; len(stack) > 0 && !full; first = false {
		cur := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		entries, cut, err := ListDir(root, cur)
		if err != nil {
			if first {
				return nil, false, err
			}
			continue
		}
		truncated = truncated || cut
		var dirs []string
		for _, e := range entries {
			if e.Dir {
				dirs = append(dirs, path.Join(cur, e.Name))
				continue
			}
			if len(files) == max {
				truncated, full = true, true
				break
			}
			files = append(files, path.Join(cur, e.Name))
		}
		for i := len(dirs) - 1; i >= 0; i-- { // the first folder is read next
			stack = append(stack, dirs[i])
		}
	}
	sort.Strings(files)
	return files, truncated, nil
}
