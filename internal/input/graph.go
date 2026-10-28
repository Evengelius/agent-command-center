package input

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"path"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// The Git Graph's reads (plan 62, brief amendment 64): the references, the
// history from fixed tips in topological order, a commit's files and a
// file's two sides in a commit, all read only (design D14). Every git call
// is for-each-ref, rev-parse, log, diff-tree or cat-file, with no optional
// lock and no file system monitor.

// GraphGit is what the Git Graph asks of git; tests pass a fake.
type GraphGit interface {
	// GraphRefs reads where HEAD is and every branch, remote branch and tag
	// with its commit (designs D3 and D4).
	GraphRefs(dir string) (GraphRefs, error)
	// GraphLog starts reading the history reachable from tips, commit
	// hashes, in topological order (design D4). The reader holds one git
	// process until Close or its last commit; no tips, none.
	GraphLog(dir string, tips []string) (GraphReader, error)
	// CommitFiles reads the files commit changed against parent, its first
	// parent, or against nothing when parent is "", a root (design D5).
	CommitFiles(dir, commit, parent string) (CommitFiles, error)
	// CommitSides reads f's two sides in commit: parent's version against
	// commit's (design D6).
	CommitSides(dir, commit, parent string, f CommitFile) (DiffPair, error)
	// CommitMessage reads commit's whole message, for its card (design D13).
	CommitMessage(dir, commit string) (string, error)
}

var _ GraphGit = GitCLI{}

// GraphRefs is where HEAD is and the repository's references (design D3).
type GraphRefs struct {
	// Head is HEAD's commit, "" when it has none (no commit yet).
	Head string
	// Branch is the branch HEAD is on, by its full name ("refs/heads/main"),
	// "" while detached or with no commit; Upstream its upstream's full
	// name ("refs/remotes/origin/main"), "" when it has none.
	Branch, Upstream string
	// Refs are every branch, remote branch and tag, by full name, in
	// for-each-ref's order (refs/heads, then refs/remotes, then refs/tags);
	// a remote's HEAD and a reference naming no commit left out.
	Refs []GraphRef
}

// GraphRef is a reference and the commit it names.
type GraphRef struct {
	// Name is the full name: "refs/heads/main", "refs/remotes/origin/main",
	// "refs/tags/v1.0".
	Name string
	// Commit is the commit it names, a tag peeled to its commit.
	Commit string
}

// GraphScope is the history's scope (design D3): All, the references
// chosen, or, neither, Auto.
type GraphScope struct {
	// All is every branch, remote branch and tag, and HEAD.
	All bool
	// Refs are the chosen references' full names; with All false and Refs
	// empty, the scope is Auto: HEAD and its branch's upstream.
	Refs []string
}

// Auto says the scope is Auto.
func (s GraphScope) Auto() bool { return !s.All && len(s.Refs) == 0 }

// Tips are the commits a history read of scope s starts from (design D4),
// each once, HEAD's first; gone are the chosen references r no longer has.
// Auto is HEAD and its upstream when the upstream exists.
func (r GraphRefs) Tips(s GraphScope) (tips, gone []string) {
	add := func(c string) {
		if c != "" && !slices.Contains(tips, c) {
			tips = append(tips, c)
		}
	}
	commit := func(name string) (string, bool) {
		for _, ref := range r.Refs {
			if ref.Name == name {
				return ref.Commit, true
			}
		}
		return "", false
	}
	switch {
	case s.All:
		add(r.Head)
		for _, ref := range r.Refs {
			add(ref.Commit)
		}
	case len(s.Refs) > 0:
		for _, name := range s.Refs {
			c, ok := commit(name)
			if !ok {
				gone = append(gone, name)
				continue
			}
			add(c)
		}
	default:
		add(r.Head)
		if r.Upstream != "" {
			if c, ok := commit(r.Upstream); ok {
				add(c)
			}
		}
	}
	return tips, gone
}

// GraphCommit is a commit as the history reads it.
type GraphCommit struct {
	// Hash is the commit's full hash; Parents its parents', first parent
	// first, none for a root.
	Hash    string
	Parents []string
	// Author is the author's name as git records it; Time the author's
	// time, in Unix seconds.
	Author string
	Time   int64
	// Subject is the message's first line.
	Subject string
}

// GraphReader reads a history a page at a time (design D4).
type GraphReader interface {
	// Next reads up to n more commits; done says the history ended. After
	// Close, or once it failed, it gives the error again.
	Next(n int) (commits []GraphCommit, done bool, err error)
	// Close ends the reading, git stopped and waited for; at any time, more
	// than once.
	Close()
}

// CommitFile is a file a commit changed (design D5).
type CommitFile struct {
	// Status is git's letter: 'A', 'M', 'D', 'R' or 'T' (a type changed).
	Status byte
	// Path is the file's path in the commit, from the repository's top
	// level, with forward slashes; Orig a rename's source, "" otherwise.
	Path, Orig string
	// Added and Removed count its lines; Binary says git counts none.
	Added, Removed int
	Binary         bool
}

// CommitFiles is a commit's files, in git's order.
type CommitFiles struct {
	Files []CommitFile
	// Total is how many there were: more than GraphFileCap lists the first
	// GraphFileCap (design D5).
	Total int
}

// GraphFileCap is the guard on a commit's files listed (design D5); a
// variable so that tests can lower it.
var GraphFileCap = 100_000

// graphPageLimit is how long a page of the history may take: past it the
// reader ends itself with a *timeoutError, so that a stuck git never holds a
// caller for good (design D4). A variable so that tests can lower it.
var graphPageLimit = 60 * time.Second

// graphRecordMax is the most bytes of a commit's record a history read keeps:
// a longer subject is cut there, with "…", so that a repository whose commit
// has a message of gigabytes cannot fill the memory (plan 48's rule: no
// unbounded growth). A variable so that tests can lower it.
var graphRecordMax = 1 << 20

// ErrGraphClosed is what a history reader gives once Close ended it, to the
// Next that was waiting and to every later one.
var ErrGraphClosed = errors.New("the history read was closed")

// notRepo is err, or NotRepoError when git says dir is no repository, as
// Blame names it.
func notRepo(dir string, err error) error {
	if strings.Contains(strings.ToLower(err.Error()), "not a git repository") {
		return NotRepoError{Dir: dir}
	}
	return err
}

// asRevision refuses what git would take for an option or for nothing: a
// revision is only ever a hash (design D14: an option such as --output writes
// a file).
func asRevision(rev string) error {
	if rev == "" || strings.HasPrefix(rev, "-") {
		return fmt.Errorf("not a commit: %q", rev)
	}
	return nil
}

// graphRefsFormat is for-each-ref's line for each reference (design D4): its
// full name, its object and that object's type, the object a tag peels to
// and its type, "*" for the branch HEAD is on, its upstream's full name, and
// what a symbolic reference names.
const graphRefsFormat = "--format=%(refname)%00%(objectname)%00%(objecttype)%00%(*objectname)%00%(*objecttype)%00%(HEAD)%00%(upstream)%00%(symref)"

// GraphRefs implements GraphGit: one for-each-ref of refs/heads,
// refs/remotes and refs/tags, and, when HEAD is on no branch that has a
// commit, one rev-parse of HEAD (designs D3 and D4). A tag stands for the
// commit it peels to; what does not end in a commit, and a symbolic
// reference (a remote's HEAD), is left out. Git's refusal of a folder
// outside any repository is a NotRepoError.
func (g GitCLI) GraphRefs(dir string) (GraphRefs, error) {
	g.noFetch = true // nothing fetched (design D14)
	out, err := g.runRead(readLimit, dir, "for-each-ref", graphRefsFormat, "refs/heads", "refs/remotes", "refs/tags")
	if err != nil {
		return GraphRefs{}, notRepo(dir, err)
	}
	var refs GraphRefs
	onBranch := false
	for _, l := range lines(out) {
		f := strings.Split(l, "\x00")
		if len(f) != 8 {
			return GraphRefs{}, fmt.Errorf("git for-each-ref: unexpected line %q", l)
		}
		name, commit, kind := f[0], f[1], f[2]
		if f[7] != "" {
			continue
		}
		if kind == "tag" {
			commit, kind = f[3], f[4]
		}
		if kind != "commit" {
			continue
		}
		refs.Refs = append(refs.Refs, GraphRef{Name: name, Commit: commit})
		if f[5] == "*" {
			refs.Head, refs.Branch, refs.Upstream, onBranch = commit, name, f[6], true
		}
	}
	if onBranch {
		return refs, nil
	}
	// Detached, or on a branch with no commit yet: git's exit 1, without a
	// word, says HEAD names no commit.
	out, err = g.runRead(readLimit, dir, "rev-parse", "--verify", "--quiet", "HEAD^{commit}")
	var ge *gitError
	switch {
	case err == nil:
		refs.Head = strings.TrimSpace(out)
	case errors.As(err, &ge) && ge.code == 1:
	default:
		return GraphRefs{}, notRepo(dir, err)
	}
	return refs, nil
}

// graphLogFormat is git log's record of a commit (design D4): its hash, its
// parents, its author's time and name, then its subject, last because it may
// hold anything but a NUL, which ends the record (-z).
const graphLogFormat = "--format=%H%x1f%P%x1f%at%x1f%an%x1f%s"

// GraphLog implements GraphGit: one git log process for the whole read, its
// pages taken from its pipe as they are asked for (design D4). The tips are
// hashes, so the read is the same whatever the references do meanwhile. With
// none there is nothing to read and no process.
func (g GitCLI) GraphLog(dir string, tips []string) (GraphReader, error) {
	g.noFetch = true
	if len(tips) == 0 {
		return &emptyGraph{}, nil
	}
	for _, tip := range tips {
		if err := asRevision(tip); err != nil {
			return nil, err
		}
	}
	args := []string{"log", "--topo-order", "--no-show-signature", "-z", graphLogFormat}
	args = append(append(args, tips...), "--")
	// The process lives across the pages, so its context has no time limit
	// (call and stream give theirs): Close and a late page cancel it.
	ctx, cancel := context.WithCancel(context.Background())
	cmd, err := g.command(ctx, dir, true, args)
	if err != nil {
		cancel()
		return nil, err
	}
	r := &graphReader{cmd: cmd, args: args, cancel: cancel, exited: make(chan struct{})}
	// Git's stdout comes through a pipe of the dashboard's, as stream's does,
	// with Wait beside the reads to close it when git has gone (plan 48,
	// design D4; audit finding 3c).
	pr, pw := io.Pipe()
	r.pr, r.br = pr, bufio.NewReaderSize(pr, 64<<10)
	cmd.Stdout, cmd.Stderr = pw, &r.stderr
	if err := cmd.Start(); err != nil {
		cancel()
		return nil, err
	}
	go r.wait(pw)
	return r, nil
}

// emptyGraph is the history of no tips: done at once, with no process; closed,
// it says so as any reader does.
type emptyGraph struct{ closed atomic.Bool }

func (e *emptyGraph) Next(int) ([]GraphCommit, bool, error) {
	if e.closed.Load() {
		return nil, false, ErrGraphClosed
	}
	return nil, true, nil
}

func (e *emptyGraph) Close() { e.closed.Store(true) }

// graphReader is a git log process and the pipe its pages come from.
type graphReader struct {
	cmd    *exec.Cmd
	args   []string // git's, for its failure's words
	cancel context.CancelFunc
	pr     *io.PipeReader
	br     *bufio.Reader // over pr; used by Next alone, under next
	stderr bytes.Buffer  // git's, read once exited is closed

	// wait sets waitErr, then closes exited.
	exited  chan struct{}
	waitErr error

	next sync.Mutex // one Next at a time
	mu   sync.Mutex // guards err and done
	// err is why the reading ended before git's last commit: the first
	// reason stays, whoever gives the next. done says it came to the end.
	err  error
	done bool
}

// wait is the goroutine that waits for git: when it has gone, the context is
// released and the pipe ends, so that a Next waiting sees the end of git's
// output. It is the reader's one goroutine (a late page's timer runs end in
// another until git is gone), and it ends with git.
func (r *graphReader) wait(pw *io.PipeWriter) {
	r.waitErr = r.cmd.Wait()
	r.cancel()
	pw.Close()
	close(r.exited)
}

// end stops the reading for reason, unless it ended already for another:
// git is stopped and waited for, a Next waiting is let go. It returns the
// reason that stands.
func (r *graphReader) end(reason error) error {
	r.mu.Lock()
	if r.err == nil {
		r.err = reason
	}
	err := r.err
	r.mu.Unlock()
	r.cancel()
	r.pr.CloseWithError(err)
	<-r.exited
	return err
}

// Close implements GraphReader.
func (r *graphReader) Close() { r.end(ErrGraphClosed) }

// Next implements GraphReader, one call at a time. It waits at most
// graphPageLimit: a page that took longer ends the reader, whatever it got,
// with a *timeoutError. An error comes with no commits, git's failure in its
// words once its output was read; the page that holds the last commit is done.
func (r *graphReader) Next(n int) ([]GraphCommit, bool, error) {
	r.next.Lock()
	defer r.next.Unlock()
	r.mu.Lock()
	err, done := r.err, r.done
	r.mu.Unlock()
	switch {
	case err != nil:
		return nil, false, err
	case done:
		return nil, true, nil
	case n < 1:
		return nil, false, nil
	}

	limit := graphPageLimit
	late := &timeoutError{"log", limit}
	timer := time.AfterFunc(limit, func() { r.end(late) })
	commits, eof, err := r.page(n)
	if !timer.Stop() {
		return nil, false, r.end(late)
	}
	switch {
	case err != nil:
		return nil, false, r.end(err)
	case eof:
		if err := r.finish(); err != nil {
			return nil, false, err
		}
		return commits, true, nil
	}
	return commits, false, nil
}

// page reads up to n records; eof says git's output ended. A page of exactly
// n looks one byte further, so that the page holding the last commit says
// so, and the process is waited for with it.
func (r *graphReader) page(n int) (commits []GraphCommit, eof bool, err error) {
	commits = make([]GraphCommit, 0, min(n, 256))
	for len(commits) < n {
		rec, cut, rerr := nextRecord(r.br)
		if rerr == io.EOF {
			return commits, true, nil
		}
		if rerr != nil {
			return nil, false, rerr
		}
		c, perr := parseGraphRecord(rec, cut)
		if perr != nil {
			return nil, false, perr
		}
		commits = append(commits, c)
	}
	if _, perr := r.br.Peek(1); perr == io.EOF {
		return commits, true, nil
	} else if perr != nil {
		return nil, false, perr
	}
	return commits, false, nil
}

// finish settles a read that came to the end of git's output: git's exit,
// waited for, says whether the history was whole, in git's words when it was
// not.
func (r *graphReader) finish() error {
	<-r.exited
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.err != nil {
		return r.err
	}
	err := r.waitErr
	if errors.Is(err, exec.ErrWaitDelay) {
		err = nil // see gitWaitDelay
	}
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		err = failure(r.args, r.stderr.String(), ee)
	}
	if err != nil {
		r.err = err
		return err
	}
	r.done = true
	return nil
}

// nextRecord reads the next record of git log -z, without its NUL; the last
// may end at the end of the output instead. It is io.EOF when there is none.
// A record is kept up to graphRecordMax bytes: the rest is read and dropped,
// and cut says so.
func nextRecord(br *bufio.Reader) (string, bool, error) {
	var buf []byte
	cut := false
	for {
		chunk, err := br.ReadSlice(0)
		ended := err == nil // the chunk ends with the NUL
		if ended {
			chunk = chunk[:len(chunk)-1]
		}
		if room := graphRecordMax - len(buf); len(chunk) > room {
			chunk, cut = chunk[:room], true
		}
		buf = append(buf, chunk...)
		switch {
		case ended:
			return string(buf), cut, nil
		case err == bufio.ErrBufferFull:
		case err == io.EOF && len(buf) == 0:
			return "", false, io.EOF
		case err == io.EOF:
			return string(buf), cut, nil
		default:
			return "", false, err
		}
	}
}

// parseGraphRecord reads a record of graphLogFormat: five fields at most,
// split at the unit separator, the subject last whatever it holds. A cut
// record has its subject end in "…". Git's bytes that are no UTF-8, such as
// an author's name in Latin-1, become U+FFFD, as the viewer's lines do.
func parseGraphRecord(rec string, cut bool) (GraphCommit, error) {
	f := strings.SplitN(rec, "\x1f", 5)
	if len(f) != 5 || !graphHash(f[0]) {
		return GraphCommit{}, fmt.Errorf("git log: unexpected record %q", rec[:min(len(rec), 80)])
	}
	c := GraphCommit{Hash: f[0], Author: strings.ToValidUTF8(f[3], "�"), Subject: strings.ToValidUTF8(f[4], "�")}
	if f[1] != "" {
		c.Parents = strings.Fields(f[1])
		for _, p := range c.Parents {
			if !graphHash(p) {
				return GraphCommit{}, fmt.Errorf("git log: unexpected parent %q of %s", p, c.Hash)
			}
		}
	}
	t, err := strconv.ParseInt(f[2], 10, 64)
	if err != nil {
		return GraphCommit{}, fmt.Errorf("git log: unexpected time %q of %s", f[2], c.Hash)
	}
	c.Time = t
	if cut {
		c.Subject += "…"
	}
	return c, nil
}

// graphHash reports whether s is a full object name, SHA-1's or SHA-256's.
func graphHash(s string) bool { return isHex(s, 40) || isHex(s, 64) }

// CommitFiles implements GraphGit: one diff-tree of parent against commit,
// or of the root commit against nothing, whose raw records and line counts
// are read as git writes them (design D5). It lists the first GraphFileCap
// files and counts them all.
func (g GitCLI) CommitFiles(dir, commit, parent string) (CommitFiles, error) {
	g.noFetch = true // a partial clone's missing blobs stay out (design D14)
	if err := asRevision(commit); err != nil {
		return CommitFiles{}, err
	}
	args := []string{"diff-tree", "-r", "-z", "--raw", "--numstat", "-M", "--no-commit-id"}
	if parent == "" {
		args = append(args, "--root", commit)
	} else {
		if err := asRevision(parent); err != nil {
			return CommitFiles{}, err
		}
		args = append(args, parent, commit)
	}
	var files CommitFiles
	err := g.stream(readLimit, dir, nil, func(r *bufio.Reader) error {
		var perr error
		files, perr = parseCommitFiles(r, GraphFileCap)
		return perr
	}, args...)
	if err != nil {
		return CommitFiles{}, err
	}
	return files, nil
}

// parseCommitFiles reads git diff-tree -z --raw --numstat: first the raw
// record of each file, ":modes ids STATUS" and its path, or its source and
// its destination for a rename or a copy; then, in the same order, its
// counts, "added TAB removed TAB path", the path being left to the two
// names that follow for a rename, and both counts "-" for a binary file.
// The first limit files are kept, all are counted.
func parseCommitFiles(r *bufio.Reader, limit int) (CommitFiles, error) {
	cf, err := readCommitFiles(r, limit)
	if err != nil {
		return CommitFiles{}, fmt.Errorf("git diff-tree: %w", err)
	}
	return cf, nil
}

func readCommitFiles(r *bufio.Reader, limit int) (CommitFiles, error) {
	var cf CommitFiles
	for { // the raw records, each starting with ":"
		b, err := r.Peek(1)
		if err == io.EOF || err == nil && b[0] != ':' {
			break
		}
		if err != nil {
			return cf, err
		}
		head, err := nulFields(r, 1)
		if err != nil {
			return cf, err
		}
		meta := strings.Fields(head[0])
		if len(meta) != 5 {
			return cf, fmt.Errorf("unexpected record %q", head[0])
		}
		f := CommitFile{Status: meta[4][0]}
		n := 1
		if f.Status == 'R' || f.Status == 'C' {
			n = 2 // its source, then its destination
		}
		names, err := nulFields(r, n)
		if err != nil {
			return cf, err
		}
		f.Path = names[n-1]
		if n == 2 {
			f.Orig = names[0]
		}
		if cf.Total++; cf.Total <= limit {
			cf.Files = append(cf.Files, f)
		}
	}
	counted := 0
	for ; ; counted++ { // their counts, in the same order
		if _, err := r.Peek(1); err == io.EOF {
			break
		} else if err != nil {
			return cf, err
		}
		rec, err := nulFields(r, 1)
		if err != nil {
			return cf, err
		}
		added, rest, _ := strings.Cut(rec[0], "\t")
		removed, p, ok := strings.Cut(rest, "\t")
		if !ok {
			return cf, fmt.Errorf("unexpected counts %q", rec[0])
		}
		orig := ""
		if p == "" { // a rename or a copy: its source and destination follow
			names, err := nulFields(r, 2)
			if err != nil {
				return cf, err
			}
			orig, p = names[0], names[1]
		}
		if counted >= len(cf.Files) {
			continue
		}
		f := &cf.Files[counted]
		if f.Path != p || f.Orig != orig {
			return cf, fmt.Errorf("the counts of %q come at the file %q", p, f.Path)
		}
		if added == "-" && removed == "-" {
			f.Binary = true
			continue
		}
		var aerr, rerr error
		f.Added, aerr = strconv.Atoi(added)
		f.Removed, rerr = strconv.Atoi(removed)
		if aerr != nil || rerr != nil {
			return cf, fmt.Errorf("unexpected counts %q", rec[0])
		}
	}
	if counted != cf.Total {
		return cf, fmt.Errorf("counts for %d of %d files", counted, cf.Total)
	}
	return cf, nil
}

// nulFields reads n fields of git's -z output, each without its NUL.
func nulFields(r *bufio.Reader, n int) ([]string, error) {
	fields := make([]string, n)
	for i := range fields {
		s, err := r.ReadString(0)
		if err != nil {
			return nil, errors.New("the output ends in a record")
		}
		fields[i] = s[:len(s)-1]
	}
	return fields, nil
}

// CommitSides implements GraphGit, as Sides does for the Source Control
// (design D6): f's old side is parent's version, at f's source for a rename,
// and its new side commit's, both from one git cat-file --batch; nothing
// against an added file's old side or a deleted file's new one, nor against
// a root's, whose parent is "". The sides are named by their commits' first
// 7 characters. A credential file is ErrCredential and nothing is read, a
// path outside the repository a *FileError.
func (g GitCLI) CommitSides(dir, commit, parent string, f CommitFile) (DiffPair, error) {
	g.noFetch = true
	if f.Path == "" || strings.HasSuffix(f.Path, "/") {
		return DiffPair{}, &FileError{"a folder, not a file"}
	}
	for _, p := range []string{f.Path, f.Orig} {
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
	// An empty commit would name the index's file, not the commit's.
	if err := asRevision(commit); err != nil {
		return DiffPair{}, err
	}
	if parent != "" {
		if err := asRevision(parent); err != nil {
			return DiffPair{}, err
		}
	}

	from := f.Path // the source of a rename is where the parent has the file
	if f.Orig != "" {
		from = f.Orig
	}
	oldName, newName := shortHash(parent), shortHash(commit)
	var pair DiffPair
	var oldSrc, newSrc []sideSource
	if f.Status == 'A' || parent == "" {
		pair.Old = DiffSide{Name: oldName, Path: from, Missing: true}
	} else {
		oldSrc = []sideSource{{oldName, from, parent + ":" + from}}
	}
	if f.Status == 'D' {
		pair.New = DiffSide{Name: newName, Path: f.Path, Missing: true}
	} else {
		newSrc = []sideSource{{newName, f.Path, commit + ":" + f.Path}}
	}
	var specs []string
	for _, c := range append(append([]sideSource(nil), oldSrc...), newSrc...) {
		specs = append(specs, c.spec)
	}
	if len(specs) == 0 {
		return pair, nil
	}
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
	return pair, nil
}

// graphMessageMax is how much of a commit's message the card keeps: far more
// than its 20 lines show.
const graphMessageMax = 64 << 10

// CommitMessage implements GraphGit: commit's whole message as `git log -1
// --format=%B` gives it, its last line breaks cut, invalid UTF-8 as U+FFFD,
// kept up to graphMessageMax bytes (design D13).
func (g GitCLI) CommitMessage(dir, commit string) (string, error) {
	g.noFetch = true
	if err := asRevision(commit); err != nil {
		return "", err
	}
	out, err := g.runRead(readLimit, dir, "log", "-1", "--no-show-signature", "--format=%B", commit, "--")
	if err != nil {
		return "", err
	}
	if len(out) > graphMessageMax {
		out = out[:graphMessageMax]
	}
	return strings.ToValidUTF8(strings.TrimRight(out, "\n"), "�"), nil
}

// shortHash is the first 7 characters of a hash, the sides' names.
func shortHash(h string) string { return h[:min(len(h), 7)] }
