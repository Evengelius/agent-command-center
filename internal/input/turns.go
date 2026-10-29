package input

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

// The turn review's snapshots (plan 65, brief amendment 67): the project's
// files as git would store them, recorded as a tree in a store of the
// dashboard's own, so that two trees give a turn's changes. The store is an
// index, seeded once from the repository's, and an object folder; every git
// call of the review gives them as its own, with the hooks and filters of the
// repository turned off, so that the project, its .git and its index are
// never written (design D2). Nothing here commits, stashes or makes a
// reference.

// TurnStoreLimit is the most the store may hold on disk (plan 65, design D3);
// a variable so that tests can lower it.
var TurnStoreLimit int64 = 256 << 20

// ErrTurnStoreFull is Take's answer when the snapshot would pass
// TurnStoreLimit, and when an earlier one did.
var ErrTurnStoreFull = errors.New("the review's store is full for this session")

// errTurnStoreClosed is every call's error once Close was called.
var errTurnStoreClosed = errors.New("the review's store is closed")

// TurnTree is one snapshot (design D2).
type TurnTree struct {
	Tree string     // the snapshot's tree hash
	Left []TurnLeft // the listed files it left out, sorted by path
}

// TurnLeft is a file a snapshot left out, never opened.
type TurnLeft struct {
	Path    string // from the repository's top level, forward slashes
	Why     string // "a credential file", "outside the project", "over 32.0 MiB" (SizeText(MaxFileSize)), "a hard link"
	Size    int64
	ModTime time.Time // from os.Lstat
}

// What a snapshot leaves out besides the files the viewer's checks refuse
// (file.go): an untracked folder, which git lists when it holds a repository
// of its own, and a file with more than one name. The checks see a hard
// link's name alone, and no path shows its other names, which may be a
// credential file's or outside the project (plan 65, the review of
// 2026-10-29).
const (
	whyNested = "a nested repository"
	whyLinked = "a hard link"
)

// The files of a store's folder (design D3); the lock's name is a draft's.
const (
	turnObjects = "objects"
	turnHooks   = "hooks"
	turnIndex   = "index"
)

// turnsRoot is the folder of the stores; "" is turnsDir's own. Tests set it to
// a temporary folder, so that they never touch the real one.
var turnsRoot string

// turnsDir is where the stores live: agent-command-center\turns in the user's
// cache directory, %LOCALAPPDATA% on Windows.
func turnsDir() (string, error) {
	if turnsRoot != "" {
		return turnsRoot, nil
	}
	d, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "agent-command-center", "turns"), nil
}

// TurnStore is the session's store (design D3): a folder of the dashboard's
// own, held as a draft's is (plan 33, design D1) by a lock it keeps open with
// no sharing, holding an index, an object folder and an empty hooks folder.
// It is safe for concurrent use: Take and Changing run one at a time, and
// Close waits for the calls under way.
type TurnStore struct {
	dir     string // the store's folder
	project string // the project folder, absolute and cleaned
	top     string // the repository's top level, in the system's form
	prefix  string // the project's place in it: "" or "sub/dir/"
	// base is the pathspec of the project, and excludes those of the
	// credential names, which every listing carries.
	base     string
	excludes []string
	// objects is the repository's object folder, behind the store's for
	// reads only (design D2).
	objects string
	lock    *os.File
	// git carries the store's index, object folder and options; its Trace
	// is for tests.
	git GitCLI

	life   sync.RWMutex // every call reads it, Close writes it
	closed bool

	mu   sync.Mutex // Take and Changing: one at a time; guards what follows
	last string     // the last tree taken, "" before the first
	left map[string]TurnLeft
	// ahead says the store's index may hold files the last tree does not: an
	// add ran and its write-tree did not follow.
	ahead bool
	full  bool // the store passed TurnStoreLimit

	size atomic.Int64 // the objects' size after the last Take
}

// OpenTurnStore makes the session's store for the project dir, which may be
// a subfolder of a repository or a linked worktree (design D3). It first
// sweeps the stores no running dashboard holds. ErrNoGit without git, a
// NotRepoError outside a repository.
func OpenTurnStore(dir string) (*TurnStore, error) {
	root, err := turnsDir()
	if err != nil {
		return nil, err
	}
	return openTurnStore(dir, root)
}

// openTurnStore is OpenTurnStore with the stores' folder given.
func openTurnStore(dir, root string) (*TurnStore, error) {
	sweepTurns(root)
	project, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	repo, err := readTurnRepo(project)
	if err != nil {
		return nil, notRepo(dir, err)
	}
	s := &TurnStore{project: project, top: repo.top, prefix: repo.prefix, objects: repo.objects,
		base: ".", excludes: credentialExcludes()}
	if repo.prefix != "" {
		s.base = ":(literal)" + strings.TrimSuffix(repo.prefix, "/")
	}
	if err := s.create(root, repo.index, repo.drivers); err != nil {
		return nil, err
	}
	return s, nil
}

// turnRepo is what the store reads of the repository once, with git's own
// environment.
type turnRepo struct {
	top, prefix    string
	index, objects string   // the repository's index, which may not exist yet, and objects
	drivers        []string // the filter drivers of git's configuration, once each
}

// readTurnRepo reads the project's place in its repository and the paths the
// store copies or reads from, then its filter drivers (design D2).
func readTurnRepo(project string) (turnRepo, error) {
	g := GitCLI{}
	out, err := g.runRead(readLimit, project, "rev-parse", "--path-format=absolute", "--show-prefix", "--show-toplevel",
		"--git-path", "index", "--git-path", "objects")
	if err != nil {
		return turnRepo{}, err
	}
	f := strings.Split(strings.TrimSuffix(out, "\n"), "\n") // the first line is empty at the top level
	for i := range f {
		f[i] = strings.TrimSuffix(f[i], "\r")
	}
	if len(f) != 4 || f[1] == "" || f[2] == "" || f[3] == "" {
		return turnRepo{}, fmt.Errorf("git rev-parse: unexpected answer %q", out)
	}
	repo := turnRepo{prefix: f[0], top: filepath.FromSlash(f[1]), index: filepath.FromSlash(f[2]), objects: filepath.FromSlash(f[3])}

	// A driver's name is what stands between "filter." and the variable's name
	// (it may hold dots); git exits 1 when there is none.
	out, err = g.run(readLimit, repo.top, "config", "--name-only", "--get-regexp", `^filter\..*\.(clean|smudge|process)$`)
	var ge *gitError
	if err != nil && !(errors.As(err, &ge) && ge.code == 1) {
		return turnRepo{}, err
	}
	for _, name := range lines(out) {
		name = strings.TrimSpace(name)
		if d, ok := strings.CutPrefix(name, "filter."); ok {
			if i := strings.LastIndexByte(d, '.'); i > 0 {
				repo.drivers = append(repo.drivers, d[:i])
			}
		}
	}
	sort.Strings(repo.drivers)
	repo.drivers = slices.Compact(repo.drivers)
	return repo, nil
}

// credentialExcludes are the viewer's credential names (file.go) as excluded
// pathspecs without case, in any folder, which the listing carries so that
// git never lists such a file: ".env.*" is every name that starts with ".env."
// (design D2), the three examples of .env included.
func credentialExcludes() []string {
	var names []string
	for n := range credentialExact {
		names = append(names, n)
	}
	for e := range credentialExt {
		names = append(names, "*"+e)
	}
	for stem := range credentialStem {
		for e := range credentialStemExt {
			names = append(names, stem+e)
		}
	}
	names = append(names, ".env.*")
	sort.Strings(names)
	for i, n := range names {
		names[i] = ":(exclude,icase,glob)**/" + n
	}
	return names
}

// create makes the store's folder, its lock and its contents (design D3),
// and gives s the git calls that use them. Any failure removes the folder.
func (s *TurnStore) create(root, repoIndex string, drivers []string) (err error) {
	if err := os.MkdirAll(root, 0o700); err != nil {
		return err
	}
	var key [keyDigits / 2]byte
	if _, err := rand.Read(key[:]); err != nil {
		return err
	}
	dir := filepath.Join(root, hex.EncodeToString(key[:]))
	if err := makeStoreDir(dir); err != nil {
		return err
	}
	s.dir = dir
	defer func() {
		if err != nil {
			if s.lock != nil {
				s.lock.Close()
			}
			_ = removeStore(dir)
		}
	}()
	if s.lock, err = openExclusive(filepath.Join(dir, lockName), true); err != nil {
		return err
	}
	for _, n := range []string{turnObjects, turnHooks} {
		if err := os.Mkdir(filepath.Join(dir, n), 0o700); err != nil {
			return err
		}
	}
	if err := copyIndex(repoIndex, filepath.Join(dir, turnIndex)); err != nil {
		return err
	}

	// Every call of the store: its index and object folder, the hooks pointed
	// at the empty folder, the file system monitor, the split index and the
	// line-ending check off, and each filter driver given empty commands
	// (design D2). Git takes a path in either form; in an option a backslash
	// could be read as an escape, so the options' go with forward slashes.
	s.git.env = []string{
		"GIT_INDEX_FILE=" + filepath.Join(dir, turnIndex),
		"GIT_OBJECT_DIRECTORY=" + filepath.Join(dir, turnObjects),
	}
	s.git.opts = []string{"core.hooksPath=" + filepath.ToSlash(filepath.Join(dir, turnHooks)),
		"core.fsmonitor=false", "core.splitIndex=false", "core.safecrlf=false", "core.quotepath=off"}
	for _, d := range drivers {
		s.git.opts = append(s.git.opts, "filter."+d+".clean=", "filter."+d+".process=", "filter."+d+".required=false")
	}
	return nil
}

// copyIndex copies the repository's index to the store's: the file is only
// read, and other programs may rename it meanwhile (a git of the owner's
// replaces it). A repository with no index yet gives none.
func copyIndex(from, to string) error {
	in, err := openShared(from)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(to, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	_, err = io.Copy(out, in)
	return errors.Join(err, out.Close())
}

// sweepTurns removes the stores no running dashboard holds (design D3): in
// root, each folder whose lock opens with no sharing is held by nobody; one
// without a lock is being made by another dashboard, and goes when it is a
// minute old. What it cannot remove, the next start does.
func sweepTurns(root string) {
	es, err := os.ReadDir(root)
	if err != nil {
		return
	}
	now := time.Now()
	for _, e := range es {
		if !e.IsDir() || !isHex(e.Name(), keyDigits) {
			continue // not a store's
		}
		p := filepath.Join(root, e.Name())
		lock, err := openExclusive(filepath.Join(p, lockName), false)
		switch {
		case errors.Is(err, fs.ErrNotExist):
			if !older(e, now) {
				continue // being made
			}
		case err != nil:
			continue // held
		}
		if lock != nil {
			lock.Close()
		}
		_ = removeStore(p)
	}
}

// removeStore removes a store's folder, again while Windows refuses because
// another program has a file open (draft.go's retried).
func removeStore(dir string) error {
	return retried(func() error { return os.RemoveAll(dir) })
}

// reads is the store's git for the calls that read objects: the repository's
// behind the store's (design D2). extra is appended to the environment.
func (s *TurnStore) reads(extra ...string) GitCLI {
	g := s.git
	g.env = append(slices.Clone(g.env), "GIT_ALTERNATE_OBJECT_DIRECTORIES="+filepath.ToSlash(s.objects))
	g.env = append(g.env, extra...)
	return g
}

// writes is the store's git for the calls that write objects or the index:
// no alternates, whatever the environment says, because git refreshed a pack
// file's time in the repository with them there (design D2).
func (s *TurnStore) writes(extra ...string) GitCLI {
	g := s.git
	g.env = append(slices.Clone(g.env), "GIT_ALTERNATE_OBJECT_DIRECTORIES=")
	g.env = append(g.env, extra...)
	return g
}

// run runs git from the repository's top level within ctx, whose deadline
// stops it, and returns its standard output. Errors are GitCLI's own: git's
// failure, or a *timeoutError when the deadline stopped it.
func (s *TurnStore) run(ctx context.Context, g GitCLI, stdin io.Reader, args ...string) (string, error) {
	start := time.Now()
	cmd, err := g.command(ctx, s.top, true, args)
	if err != nil {
		return "", err
	}
	var out, errb bytes.Buffer
	cmd.Stdin, cmd.Stdout, cmd.Stderr = stdin, &out, &errb
	err = cmd.Run()
	if errors.Is(err, exec.ErrWaitDelay) {
		err = nil // see gitWaitDelay
	}
	if err != nil && ctx.Err() != nil {
		return "", turnStopped(ctx, args[0], start)
	}
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		return out.String(), failure(args, errb.String(), ee)
	}
	if err != nil {
		return "", err
	}
	return out.String(), nil
}

// turnStopped is why a call that ctx ended did not finish: a *timeoutError
// when the deadline passed, with the time the call was given; else ctx's own.
func turnStopped(ctx context.Context, sub string, start time.Time) error {
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		limit := time.Duration(0)
		if dl, ok := ctx.Deadline(); ok {
			limit = max(0, dl.Sub(start)).Round(100 * time.Millisecond)
		}
		return &timeoutError{sub, limit}
	}
	return ctx.Err()
}

// unlock removes the index.lock a git stopped midway leaves behind, so that
// the next call works: none of the store's gits runs when it is called, and
// nobody else uses the store's index (design D2).
func (s *TurnStore) unlock() {
	_ = retried(func() error {
		if err := os.Remove(filepath.Join(s.dir, turnIndex+".lock")); !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		return nil
	})
}

// turnEntry is one record of the listing.
type turnEntry struct {
	path      string // from the repository's top level
	module    bool   // a submodule's: git records its commit only
	untracked bool   // the store's index does not hold it
	folder    bool   // an untracked folder, git's way to list a nested repository
}

// list is the project's files that differ from the store's index, credential
// names excluded by git itself (design D2): the working tree against the
// index, not the index against HEAD. GIT_OPTIONAL_LOCKS=1 lets git refresh
// the store's own index, so that a touched file is hashed once.
func (s *TurnStore) list(ctx context.Context) ([]turnEntry, error) {
	args := append([]string{"status", "--porcelain=v2", "-z", "--untracked-files=all", "--ignore-submodules=dirty",
		"--no-renames", "--", s.base}, s.excludes...)
	out, err := s.run(ctx, s.reads("GIT_OPTIONAL_LOCKS=1"), nil, args...)
	if err != nil {
		s.unlock()
		return nil, err
	}
	return parseTurnStatus(out)
}

// parseTurnStatus reads git status --porcelain=v2 -z --no-renames: "1 XY sub
// mH mI mW hH hI path" for a changed file, "u XY sub m1 m2 m3 mW h1 h2 h3
// path" for an unmerged one and "? path" for an untracked one, each ended by a
// NUL. A file whose Y is "." has its working tree as the index has it: it
// changed against HEAD, not since the store's last snapshot. A rename's "2"
// record cannot come with --no-renames.
func parseTurnStatus(out string) ([]turnEntry, error) {
	var entries []turnEntry
	for _, rec := range strings.Split(out, "\x00") {
		kind, rest, _ := strings.Cut(rec, " ")
		switch kind {
		case "":
		case "1", "u":
			n := 9
			if kind == "u" {
				n = 11
			}
			f := strings.SplitN(rec, " ", n)
			if len(f) != n || len(f[1]) != 2 || f[n-1] == "" {
				return nil, fmt.Errorf("git status: unexpected record %q", rec)
			}
			if kind == "1" && f[1][1] == '.' {
				continue
			}
			entries = append(entries, turnEntry{path: f[n-1], module: !strings.HasPrefix(f[2], "N")})
		case "?":
			if rest == "" {
				return nil, fmt.Errorf("git status: unexpected record %q", rec)
			}
			entries = append(entries, turnEntry{path: rest, untracked: true, folder: strings.HasSuffix(rest, "/")})
		case "2":
			return nil, fmt.Errorf("git status: unexpected rename record %q", rec)
		case "!", "#":
		default:
			return nil, fmt.Errorf("git status: unexpected record %q", rec)
		}
	}
	return entries, nil
}

// abs is the path of a listed file on disk, in the project's own form.
func (s *TurnStore) abs(repoPath string) string {
	if n := len(s.prefix); len(repoPath) >= n && strings.EqualFold(repoPath[:n], s.prefix) {
		return filepath.Join(s.project, filepath.FromSlash(repoPath[n:]))
	}
	return filepath.Join(s.top, filepath.FromSlash(repoPath))
}

// turnScan is the listing once its files passed the viewer's checks.
type turnScan struct {
	take []string   // what git adds: the files that passed, the deleted ones and the submodules
	need int64      // what taking them adds to the store at most (design D3)
	left []TurnLeft // what was left out, sorted by path
}

// turnCost is the most a file of size bytes adds to the store: a compressed
// copy with git's header, which the size and a thousandth of it cover, and the
// trees above it.
func turnCost(size int64) int64 { return size + size/1000 + 1024 }

// scan lists what changed since the store's last snapshot, then checks it.
func (s *TurnStore) scan(ctx context.Context) (turnScan, error) {
	entries, err := s.list(ctx)
	if err != nil {
		return turnScan{}, err
	}
	return s.check(ctx, entries)
}

// check makes the file viewer's checks on each listed file before git reads
// it (design D2): no credential name, its own or its link's target's, inside
// the project once links and junctions are resolved, not a folder, MaxFileSize
// at most; then one name alone, no hard link (linkCount). A file that fails
// is left out, never opened for its content; one gone is a deletion, which
// git records without reading anything.
func (s *TurnStore) check(ctx context.Context, entries []turnEntry) (turnScan, error) {
	var sc turnScan
	realRoot := ""
	leave := func(path, why string, info fs.FileInfo) {
		l := TurnLeft{Path: path, Why: why}
		if info != nil {
			l.Size, l.ModTime = info.Size(), info.ModTime()
		}
		sc.left = append(sc.left, l)
	}
	start := time.Now()
	for _, e := range entries {
		if ctx.Err() != nil {
			return turnScan{}, turnStopped(ctx, "status", start)
		}
		if e.module {
			sc.take, sc.need = append(sc.take, e.path), sc.need+turnCost(0)
			continue
		}
		abs := s.abs(e.path)
		info, err := os.Lstat(abs)
		switch {
		case errors.Is(err, fs.ErrNotExist) || errors.Is(err, syscall.ENOTDIR):
			// Gone since the listing. An untracked file that came and went has
			// nothing to record, and git add refuses a path it knows nowhere.
			if !e.untracked {
				sc.take, sc.need = append(sc.take, e.path), sc.need+turnCost(0)
			}
			continue
		case err != nil:
			leave(e.path, whyUnreadable, nil)
			continue
		case e.folder:
			leave(e.path, whyNested, info)
			continue
		}
		if realRoot == "" {
			if realRoot, err = realPath(s.project); err != nil {
				return turnScan{}, fmt.Errorf("the project cannot be read: %s", s.project)
			}
		}
		cf, why, err := checkFile(s.project, realRoot, e.path, abs)
		if err != nil {
			leave(e.path, why, info)
			continue
		}
		if n, err := linkCount(cf); err != nil {
			leave(e.path, whyUnreadable, info)
			continue
		} else if n > 1 {
			leave(e.path, whyLinked, info)
			continue
		}
		sc.take, sc.need = append(sc.take, e.path), sc.need+turnCost(cf.info.Size())
	}
	sort.Slice(sc.left, func(i, j int) bool { return sc.left[i].Path < sc.left[j].Path })
	return sc, nil
}

// Take records the project's files as a tree (design D2): the listing, the
// checks, then git add of the files that passed and git write-tree. A tree is
// the last one's when nothing changed, from the listing alone. ctx's deadline
// stops git. It is ErrTurnStoreFull, with nothing taken, when the files to
// take would pass TurnStoreLimit, and at once when an earlier Take left the
// store past it (design D3).
func (s *TurnStore) Take(ctx context.Context) (TurnTree, error) {
	s.life.RLock()
	defer s.life.RUnlock()
	if s.closed {
		return TurnTree{}, errTurnStoreClosed
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.full {
		return TurnTree{}, ErrTurnStoreFull
	}
	sc, err := s.scan(ctx)
	if err != nil {
		return TurnTree{}, err
	}
	if len(sc.take) == 0 && s.last != "" && !s.ahead {
		s.left = leftByPath(sc.left)
		return TurnTree{Tree: s.last, Left: sc.left}, nil
	}
	if len(sc.take) > 0 && s.size.Load()+sc.need > TurnStoreLimit {
		return TurnTree{}, ErrTurnStoreFull
	}

	// Git has written, or may have, from here: the size is measured again
	// whatever came of it.
	defer func() {
		n := s.measure()
		s.size.Store(n)
		s.full = n > TurnStoreLimit
	}()
	if len(sc.take) > 0 {
		// The index may take the files in even when this call fails after it
		// (its deadline passes between the two calls): the listing would not
		// show them again, so the next Take writes the tree whatever it lists.
		s.ahead = true
		// Literal pathspecs, from the listing: a name holds nothing git reads as
		// a pattern; NUL ends each one, as a name holds anything else.
		in := strings.Join(sc.take, "\x00") + "\x00"
		g := s.writes("GIT_LITERAL_PATHSPECS=1")
		if _, err := s.run(ctx, g, strings.NewReader(in), "add", "-A", "--pathspec-from-file=-", "--pathspec-file-nul"); err != nil {
			s.unlock()
			return TurnTree{}, err
		}
	}
	out, err := s.run(ctx, s.writes(), nil, "write-tree", "--missing-ok")
	if err != nil {
		s.unlock()
		return TurnTree{}, err
	}
	tree := strings.TrimSpace(out)
	if !graphHash(tree) {
		return TurnTree{}, fmt.Errorf("git write-tree: unexpected answer %q", out)
	}
	s.last, s.left, s.ahead = tree, leftByPath(sc.left), false
	return TurnTree{Tree: tree, Left: sc.left}, nil
}

// leftByPath indexes the files a snapshot left out, for Changing.
func leftByPath(left []TurnLeft) map[string]TurnLeft {
	m := make(map[string]TurnLeft, len(left))
	for _, l := range left {
		m[l.Path] = l
	}
	return m
}

// measure is the size of the files under the store's objects folder.
func (s *TurnStore) measure() int64 {
	var n int64
	_ = filepath.WalkDir(filepath.Join(s.dir, turnObjects), func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if info, err := d.Info(); err == nil {
			n += info.Size()
		}
		return nil
	})
	return n
}

// Size is the store's size on disk, the files under its object folder,
// measured after the last Take (design D3).
func (s *TurnStore) Size() int64 { return s.size.Load() }

// Changing lists the files that changed since the last Take (design D5): the
// listing once more, sorted. A file the last Take left out is not one while
// its size and time are what that Take found.
func (s *TurnStore) Changing(ctx context.Context) ([]string, error) {
	s.life.RLock()
	defer s.life.RUnlock()
	if s.closed {
		return nil, errTurnStoreClosed
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	entries, err := s.list(ctx)
	if err != nil {
		return nil, err
	}
	var paths []string
	for _, e := range entries {
		if l, ok := s.left[e.path]; ok {
			if info, err := os.Lstat(s.abs(e.path)); err == nil && info.Size() == l.Size && info.ModTime().Equal(l.ModTime) {
				continue
			}
		}
		paths = append(paths, e.path)
	}
	sort.Strings(paths)
	return paths, nil
}

// snapshotTree refuses what is no tree's name, so that a revision never
// reaches git as an option or as the repository's own reference.
func snapshotTree(trees ...string) error {
	for _, t := range trees {
		if !graphHash(t) {
			return fmt.Errorf("not a snapshot: %q", t)
		}
	}
	return nil
}

// Files lists the files that differ between two trees of Take, from's to
// to's, as the Git Graph lists a commit's (plan 62, design D5): letters,
// counts and renames, paths from the repository's top level.
func (s *TurnStore) Files(from, to string) (CommitFiles, error) {
	s.life.RLock()
	defer s.life.RUnlock()
	if s.closed {
		return CommitFiles{}, errTurnStoreClosed
	}
	if err := snapshotTree(from, to); err != nil {
		return CommitFiles{}, err
	}
	return s.reads().CommitFiles(s.top, to, from)
}

// Sides reads the two sides of f between two trees of Take, as the Git Graph
// does for a commit (plan 62, design D6): a credential file is refused by its
// name and a blob over MaxFileSize is not read.
func (s *TurnStore) Sides(from, to string, f CommitFile) (DiffPair, error) {
	s.life.RLock()
	defer s.life.RUnlock()
	if s.closed {
		return DiffPair{}, errTurnStoreClosed
	}
	if err := snapshotTree(from, to); err != nil {
		return DiffPair{}, err
	}
	return s.reads().CommitSides(s.top, to, from, f)
}

// Close lets the store go: the lock is released and the folder removed, once
// the calls under way have ended (design D3). Every call after it fails.
func (s *TurnStore) Close() error {
	s.life.Lock()
	defer s.life.Unlock()
	if s.closed {
		return nil
	}
	s.closed = true
	return errors.Join(s.lock.Close(), removeStore(s.dir))
}
