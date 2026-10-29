package input

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"reflect"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

// Plan 65's criteria 65.1 to 65.3 for the snapshot store (internal/input): on
// real git in temporary repositories, the stores in temporary folders of their
// own. A test that sets TurnStoreLimit, MaxFileSize or turnsRoot is not
// parallel.

// turnCtx is a context for a call of the store, with a time limit no test
// should reach.
func turnCtx(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	t.Cleanup(cancel)
	return ctx
}

// turnProject is a repository whose first commit holds files, path by path.
func turnProject(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := srcRepo(t)
	for rel, data := range files {
		writeAt(t, dir, rel, data)
	}
	gitT(t, dir, "add", "-A")
	gitT(t, dir, "commit", "-q", "-m", "base")
	return dir
}

// openTurns opens a store on dir, in a stores' folder of its own, closed when
// the test ends.
func openTurns(t *testing.T, dir string) *TurnStore {
	t.Helper()
	s, err := openTurnStore(dir, filepath.Join(t.TempDir(), "turns"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

// traceTurns records the git calls of the store.
func traceTurns(s *TurnStore) *graphTrace {
	l := &graphTrace{}
	s.git.Trace = l.add
	return l
}

// takeTree takes a snapshot that must work.
func takeTree(t *testing.T, s *TurnStore) TurnTree {
	t.Helper()
	tt, err := s.Take(turnCtx(t))
	if err != nil {
		t.Fatal(err)
	}
	return tt
}

// storeGit runs git on the store's objects, the repository's behind them, as
// the test's own reader: no help from the store's code.
func storeGit(t *testing.T, s *TurnStore, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", s.top}, args...)...)
	cmd.Env = append(os.Environ(), "GIT_OBJECT_DIRECTORY="+filepath.Join(s.dir, turnObjects),
		"GIT_ALTERNATE_OBJECT_DIRECTORIES="+s.objects)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git %s: %v", strings.Join(args, " "), err)
	}
	return string(out)
}

// blobIn is the content of path in tree, as the store holds it.
func blobIn(t *testing.T, s *TurnStore, tree, path string) string {
	t.Helper()
	return storeGit(t, s, "cat-file", "blob", tree+":"+path)
}

// pathsIn are the files of tree, sorted.
func pathsIn(t *testing.T, s *TurnStore, tree string) []string {
	t.Helper()
	return lines(storeGit(t, s, "ls-tree", "-r", "--name-only", tree))
}

// storeObjects is every object of the store, as git prints them: its own
// object folder alone, no alternates. A blob's content is in it as it is, a
// tree's file names too.
func storeObjects(t *testing.T, s *TurnStore) string {
	t.Helper()
	cmd := exec.Command("git", "-C", s.top, "cat-file", "--batch-all-objects", "--batch")
	cmd.Env = append(os.Environ(), "GIT_OBJECT_DIRECTORY="+filepath.Join(s.dir, turnObjects), "GIT_ALTERNATE_OBJECT_DIRECTORIES=")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git cat-file --batch-all-objects: %v", err)
	}
	return string(out)
}

// storeFiles are the files under the store's object folder with their sizes.
func storeFiles(t *testing.T, s *TurnStore) map[string]int64 {
	t.Helper()
	out := map[string]int64{}
	root := filepath.Join(s.dir, turnObjects)
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, p)
		out[filepath.ToSlash(rel)] = info.Size()
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// sameFile says two states of a file are one.
func sameFile(a, b fileState) bool { return a.size == b.size && a.mod.Equal(b.mod) && a.sum == b.sum }

// unchanged reports every file whose path, size, time or bytes differ between
// two states of a tree.
func unchanged(t *testing.T, step string, before, after map[string]fileState) {
	t.Helper()
	for p, b := range before {
		if a, ok := after[p]; !ok || !sameFile(a, b) {
			t.Errorf("%s: %s changed: %+v then %+v", step, p, b, a)
		}
	}
	for p := range after {
		if _, ok := before[p]; !ok {
			t.Errorf("%s: %s appeared", step, p)
		}
	}
}

// TestTurnUnchanged is criterion 65.1's first sentence: a folder that did not
// change gives the same tree twice, the second from the listing alone, and a
// clean checkout's first tree is HEAD's.
func TestTurnUnchanged(t *testing.T) {
	t.Parallel()
	dir := turnProject(t, map[string]string{"a.txt": "one\n", "sub/b.txt": "two\n"})
	s := openTurns(t, dir)
	calls := traceTurns(s)

	t1 := takeTree(t, s)
	first := calls.subs()
	if !slices.Equal(first, []string{"status", "write-tree"}) {
		t.Errorf("the first Take with nothing to take ran %q, want a status and a write-tree", first)
	}
	if want := gitT(t, dir, "rev-parse", "HEAD^{tree}"); t1.Tree != want || len(t1.Left) != 0 {
		t.Errorf("the first tree of a clean checkout is %s with %v left out, want HEAD's %s", t1.Tree, t1.Left, want)
	}
	t2 := takeTree(t, s)
	if t2.Tree != t1.Tree {
		t.Errorf("an unchanged folder gave %s then %s", t1.Tree, t2.Tree)
	}
	if got := calls.subs()[len(first):]; !slices.Equal(got, []string{"status"}) {
		t.Errorf("the second Take ran %q, want the listing alone", got)
	}
	if n := s.Size(); n != s.measure() {
		t.Errorf("Size is %d, the objects hold %d", n, s.measure())
	}
}

// TestTurnFiles is criterion 65.1's second sentence: a turn that modifies,
// adds, removes and renames files and writes a binary one is exactly those
// files, with git's letters and counts and an R from -M, and each file's sides
// are the turn's.
func TestTurnFiles(t *testing.T) {
	t.Parallel()
	const (
		mod0 = "m1\nm2\nm3\nm4\nm5\nm6\nm7\nm8\n"
		mod1 = "m1\nM2\nm3\nm4\nm5\nm6\nm7\nm8\nm9\n"
		ren0 = "r1\nr2\nr3\nr4\nr5\nr6\nr7\nr8\n"
		ren1 = "r1\nr2\nr3\nR4\nr5\nr6\nr7\nr8\n"
	)
	dir := turnProject(t, map[string]string{"mod.txt": mod0, "del.txt": "gone\n", "ren-old.txt": ren0, "keep.txt": "same\n",
		"sub/in.txt": "in\n", "sub/é ünï.txt": "u0\n"})
	s := openTurns(t, dir)
	start := takeTree(t, s)

	writeAt(t, dir, "sub/é ünï.txt", "u1\n")
	writeAt(t, dir, "mod.txt", mod1)
	writeAt(t, dir, "add.txt", "a1\na2\na3\n")
	writeAt(t, dir, "sub/bin.dat", "x\x00y\x00z")
	if err := os.Remove(filepath.Join(dir, "del.txt")); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(dir, "ren-old.txt")); err != nil {
		t.Fatal(err)
	}
	writeAt(t, dir, "ren new.txt", ren1)
	end := takeTree(t, s)
	if end.Tree == start.Tree {
		t.Fatal("a turn that changed files gave the same tree")
	}

	cf, err := s.Files(start.Tree, end.Tree)
	if err != nil {
		t.Fatal(err)
	}
	want := []CommitFile{
		{Status: 'A', Path: "add.txt", Added: 3},
		{Status: 'D', Path: "del.txt", Removed: 1},
		{Status: 'M', Path: "mod.txt", Added: 2, Removed: 1},
		{Status: 'R', Path: "ren new.txt", Orig: "ren-old.txt", Added: 1, Removed: 1},
		{Status: 'A', Path: "sub/bin.dat", Binary: true},
		{Status: 'M', Path: "sub/é ünï.txt", Added: 1, Removed: 1},
	}
	if !reflect.DeepEqual(cf.Files, want) || cf.Total != len(want) {
		t.Errorf("Files:\n got %+v (of %d)\nwant %+v", cf.Files, cf.Total, want)
	}

	text := func(data string) DiffSide { return DiffSide{Size: int64(len(data)), Data: []byte(data)} }
	for _, c := range []struct {
		file     CommitFile
		old, new DiffSide
	}{
		{want[0], DiffSide{Missing: true}, text("a1\na2\na3\n")},
		{want[1], text("gone\n"), DiffSide{Missing: true}},
		{want[2], text(mod0), text(mod1)},
		{want[3], text(ren0), text(ren1)},
		{want[4], DiffSide{Missing: true}, DiffSide{Size: 5, Binary: true}},
		{want[5], text("u0\n"), text("u1\n")},
	} {
		pair, err := s.Sides(start.Tree, end.Tree, c.file)
		if err != nil {
			t.Errorf("%s: %v", c.file.Path, err)
			continue
		}
		for _, side := range []struct {
			name      string
			got, want DiffSide
		}{{"old", pair.Old, c.old}, {"new", pair.New, c.new}} {
			g := side.got
			if g.Missing != side.want.Missing || g.Binary != side.want.Binary || g.Size != side.want.Size || !bytes.Equal(g.Data, side.want.Data) {
				t.Errorf("%s, %s side: missing %v binary %v size %d %q; want missing %v binary %v size %d %q", c.file.Path, side.name,
					g.Missing, g.Binary, g.Size, g.Data, side.want.Missing, side.want.Binary, side.want.Size, side.want.Data)
			}
		}
	}

	// A credential file is refused by its name, a tree that is none by its form.
	if _, err := s.Sides(start.Tree, end.Tree, CommitFile{Status: 'M', Path: "config/.env"}); !errors.Is(err, ErrCredential) {
		t.Errorf("Sides of a credential file: %v", err)
	}
	for _, c := range [][2]string{{"", end.Tree}, {start.Tree, "HEAD"}, {"--output=x", end.Tree}, {start.Tree, "main"}} {
		if _, err := s.Files(c[0], c[1]); err == nil {
			t.Errorf("Files(%q, %q) was read", c[0], c[1])
		}
		if _, err := s.Sides(c[0], c[1], want[0]); err == nil {
			t.Errorf("Sides(%q, %q) was read", c[0], c[1])
		}
	}
}

// TestTurnWritesNothing is criterion 65.1's third sentence (design D10): in a
// main worktree and in a linked one, with the owner's changes pending, every
// file of the project and of its .git, path, size, time and bytes, is as it
// was across the store's opening, Take, Changing, Files and Sides. The
// repository's objects are in a pack, and a file of the project repeats a
// packed blob: written with the repository's objects behind it, git would
// refresh the pack's time.
func TestTurnWritesNothing(t *testing.T) {
	t.Parallel()
	for _, linked := range []bool{false, true} {
		name := "main worktree"
		if linked {
			name = "linked worktree"
		}
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			const packed = "the same lines\nin a packed blob\n"
			main := turnProject(t, map[string]string{"keep.txt": packed, "mod.txt": "m\n", "gone.txt": "g\n",
				"touched.txt": "t\n", "sub/in.txt": "in\n"})
			gitT(t, main, "gc", "-q")
			dir, roots := main, []string{main}
			if linked {
				dir = filepath.Join(filepath.Dir(main), "linked")
				gitT(t, main, "worktree", "add", "-q", "-b", "linked", dir)
				roots = append(roots, dir)
			}
			// With this set, git would write a shared index of its own in .git. (A
			// split index that exists already has its shared file's time refreshed
			// by any read of it, the owner's git commands' included.)
			gitT(t, main, "config", "core.splitIndex", "true")
			// The owner's changes before the first turn: a file changed, one made,
			// one made that repeats a packed blob, one removed, one touched.
			writeAt(t, dir, "mod.txt", "changed\n")
			writeAt(t, dir, "new.txt", "untracked\n")
			writeAt(t, dir, "twin.txt", packed)
			if err := os.Remove(filepath.Join(dir, "gone.txt")); err != nil {
				t.Fatal(err)
			}
			later := time.Now().Add(time.Hour)
			if err := os.Chtimes(filepath.Join(dir, "touched.txt"), later, later); err != nil {
				t.Fatal(err)
			}
			state := func() map[string]fileState {
				all := map[string]fileState{}
				for i, root := range roots {
					for p, st := range treeState(t, root) {
						all[strconv.Itoa(i)+"/"+p] = st
					}
				}
				return all
			}

			before := state()
			s := openTurns(t, dir)
			t1 := takeTree(t, s)
			unchanged(t, "OpenTurnStore and the first Take", before, state())

			writeAt(t, dir, "mod.txt", "changed in the turn\n")
			writeAt(t, dir, "more.txt", "more\n")
			before = state()
			t2 := takeTree(t, s)
			if _, err := s.Changing(turnCtx(t)); err != nil {
				t.Fatal(err)
			}
			cf, err := s.Files(t1.Tree, t2.Tree)
			if err != nil {
				t.Fatal(err)
			}
			if got := filesOf(cf); !slices.Equal(got, []string{"M mod.txt", "A more.txt"}) {
				t.Errorf("the turn's files: %q", got)
			}
			for _, f := range cf.Files {
				if _, err := s.Sides(t1.Tree, t2.Tree, f); err != nil {
					t.Errorf("Sides of %s: %v", f.Path, err)
				}
			}
			unchanged(t, "the second Take, Changing, Files and Sides", before, state())
		})
	}
}

// filesOf are the letters and paths of CommitFiles, in git's order.
func filesOf(cf CommitFiles) []string {
	var out []string
	for _, f := range cf.Files {
		out = append(out, string(f.Status)+" "+f.Path)
	}
	return out
}

// TestTurnSplitIndex is the review of 2026-10-29 on criterion 65.1's third
// sentence: in a repository whose index is split already, git refreshes the
// time of its shared index file each time it reads it (git's documentation of
// core.splitIndex), and the store's first snapshot reads it through its copy
// of the index. That time is all that changes: the shared index's bytes, and
// every other file of the project and of its .git, path, size, time and
// bytes, the index and the references among them, stay as they were. The
// first snapshot writes the store's index whole, so no later call reads the
// shared one again, and nothing changes at all.
func TestTurnSplitIndex(t *testing.T) {
	t.Parallel()
	dir := turnProject(t, map[string]string{"keep.txt": "keep\n", "mod.txt": "m\n"})
	gitT(t, dir, "update-index", "--split-index")
	found, err := filepath.Glob(filepath.Join(dir, ".git", "sharedindex.*"))
	if err != nil || len(found) != 1 {
		t.Fatalf("the shared index: %v, %v", found, err)
	}
	shared := ".git/" + filepath.Base(found[0])
	// An hour back, so that a refresh shows whatever the clock's grain.
	earlier := time.Now().Add(-time.Hour)
	if err := os.Chtimes(found[0], earlier, earlier); err != nil {
		t.Fatal(err)
	}
	writeAt(t, dir, "mod.txt", "changed\n") // the owner's change before the first turn

	before := treeState(t, dir)
	s := openTurns(t, dir)
	t1 := takeTree(t, s)
	after := treeState(t, dir)
	if b, a := before[shared], after[shared]; a.size != b.size || a.sum != b.sum {
		t.Errorf("the shared index's bytes changed: %+v then %+v", b, a)
	} else if a.mod.Equal(b.mod) {
		t.Error("the shared index's time did not change: git no longer refreshes it on a read, and reliability.md's exception can go")
	}
	delete(before, shared)
	delete(after, shared)
	unchanged(t, "OpenTurnStore and the first Take, the shared index's time aside", before, after)

	writeAt(t, dir, "mod.txt", "changed in the turn\n")
	before = treeState(t, dir)
	t2 := takeTree(t, s)
	if _, err := s.Changing(turnCtx(t)); err != nil {
		t.Fatal(err)
	}
	cf, err := s.Files(t1.Tree, t2.Tree)
	if err != nil {
		t.Fatal(err)
	}
	if got := filesOf(cf); !slices.Equal(got, []string{"M mod.txt"}) {
		t.Errorf("the turn's files: %q", got)
	}
	for _, f := range cf.Files {
		if _, err := s.Sides(t1.Tree, t2.Tree, f); err != nil {
			t.Errorf("Sides of %s: %v", f.Path, err)
		}
	}
	unchanged(t, "the second Take, Changing, Files and Sides", before, treeState(t, dir))
}

// TestTurnFiltersHooksLineEndings is criterion 65.1's fourth and fifth
// sentences (designs D2 and D5): the repository's filter driver and its
// post-index-change hook are not run by Take and are by git's own add, the
// control; a CRLF file under "* text eol=crlf" with core.autocrlf true is
// stored as LF, as git's own add stores it, and a filtered file as its bytes.
func TestTurnFiltersHooksLineEndings(t *testing.T) {
	t.Parallel()
	dir := srcRepo(t)
	marks := t.TempDir()
	mark := func(n string) string { return filepath.Join(marks, n) }
	slashed := filepath.ToSlash(marks)
	gitT(t, dir, "config", "core.autocrlf", "true")
	gitT(t, dir, "config", "filter.mark.clean", "echo clean >> '"+slashed+"/clean'; tr a-z A-Z")
	gitT(t, dir, "config", "filter.mark.smudge", "cat")
	writeAt(t, dir, ".git/hooks/post-index-change", "#!/bin/sh\necho hook >> '"+slashed+"/hook'\n")
	writeAt(t, dir, ".gitattributes", "* text eol=crlf\n*.bin filter=mark\n")
	writeAt(t, dir, "base.md", "base\n")
	gitT(t, dir, "add", "-A")
	gitT(t, dir, "commit", "-q", "-m", "base")
	_ = os.Remove(mark("clean"))
	_ = os.Remove(mark("hook"))

	s := openTurns(t, dir)
	takeTree(t, s)
	writeAt(t, dir, "a.bin", "lower case\n")
	writeAt(t, dir, "crlf.txt", "crlf\r\nhere\r\n")
	tt := takeTree(t, s)
	for _, n := range []string{"clean", "hook"} {
		if exists(mark(n)) {
			t.Errorf("Take ran the repository's %s", n)
		}
	}
	if got := blobIn(t, s, tt.Tree, "crlf.txt"); got != "crlf\nhere\n" {
		t.Errorf("crlf.txt is stored as %q, want LF", got)
	}
	if got := blobIn(t, s, tt.Tree, "a.bin"); got != "lower case\n" {
		t.Errorf("the filtered file is stored as %q, want its bytes", got)
	}
	// The listing reads the files as the Take stored them: neither the
	// converted file nor the filtered one is changing.
	if got, err := s.Changing(turnCtx(t)); err != nil || len(got) != 0 {
		t.Errorf("Changing lists %v, %v after the Take", got, err)
	}
	if exists(mark("clean")) || exists(mark("hook")) {
		t.Error("Changing ran the repository's filter or hook")
	}

	// The control: git's own add runs both, and stores the CRLF file as the
	// store did; the filtered file it stores as the filter made it.
	gitT(t, dir, "add", "-A")
	for _, n := range []string{"clean", "hook"} {
		if !exists(mark(n)) {
			t.Errorf("git's own add did not run the %s: the marks prove nothing", n)
		}
	}
	stored := func(p string) string { return strings.TrimSpace(storeGit(t, s, "rev-parse", tt.Tree+":"+p)) }
	if own := gitT(t, dir, "rev-parse", ":crlf.txt"); own != stored("crlf.txt") {
		t.Errorf("crlf.txt: git's own add stores %s, the store %s", own, stored("crlf.txt"))
	}
	if own := gitT(t, dir, "rev-parse", ":a.bin"); own == stored("a.bin") {
		t.Errorf("a.bin: git's own add and the store both store %s: the filter did nothing", own)
	}

	// core.safecrlf=true makes git's own add refuse a file whose line endings it
	// cannot round trip; the store's calls turn the check off.
	gitT(t, dir, "config", "core.safecrlf", "true")
	writeAt(t, dir, "mixed.txt", "one\r\ntwo\n")
	if _, err := gitTry(dir, "add", "mixed.txt"); err == nil {
		t.Error("git's own add took a file with mixed line endings under core.safecrlf: the control fails")
	}
	tt = takeTree(t, s)
	if got := blobIn(t, s, tt.Tree, "mixed.txt"); got != "one\ntwo\n" {
		t.Errorf("mixed.txt is stored as %q, want LF", got)
	}
}

// TestTurnDirtyStart is criterion 65.1's last sentence: a tracked file the
// owner changed before the first turn and an untracked one made before it, both
// changed again by the turn, show the turn's change alone, never the earlier
// one: the first tree holds the owner's versions.
func TestTurnDirtyStart(t *testing.T) {
	t.Parallel()
	dir := turnProject(t, map[string]string{"tracked.txt": "t0\n", "keep.txt": "k\n"})
	writeAt(t, dir, "tracked.txt", "t1\n")
	writeAt(t, dir, "made.txt", "n1\n")
	s := openTurns(t, dir)
	first := takeTree(t, s)
	if got := blobIn(t, s, first.Tree, "tracked.txt"); got != "t1\n" {
		t.Errorf("the first tree holds tracked.txt as %q, want the owner's %q", got, "t1\n")
	}
	if got := blobIn(t, s, first.Tree, "made.txt"); got != "n1\n" {
		t.Errorf("the first tree holds made.txt as %q, want the owner's %q", got, "n1\n")
	}

	writeAt(t, dir, "tracked.txt", "t2\n")
	writeAt(t, dir, "made.txt", "n2\n")
	second := takeTree(t, s)
	cf, err := s.Files(first.Tree, second.Tree)
	if err != nil {
		t.Fatal(err)
	}
	want := []CommitFile{{Status: 'M', Path: "made.txt", Added: 1, Removed: 1}, {Status: 'M', Path: "tracked.txt", Added: 1, Removed: 1}}
	if !reflect.DeepEqual(cf.Files, want) {
		t.Errorf("Files:\n got %+v\nwant %+v", cf.Files, want)
	}
	for i, old := range []string{"n1\n", "t1\n"} {
		pair, err := s.Sides(first.Tree, second.Tree, want[i])
		if err != nil {
			t.Fatal(err)
		}
		if string(pair.Old.Data) != old || string(pair.New.Data) != strings.ReplaceAll(old, "1", "2") {
			t.Errorf("%s: sides %q and %q, want %q and %q", want[i].Path, pair.Old.Data, pair.New.Data, old, strings.ReplaceAll(old, "1", "2"))
		}
	}
}

// credentialFiles are the names the viewer refuses, one per entry of its
// tables, and the three examples of .env that the snapshot leaves out too
// (design D2).
func credentialFiles() []string {
	var names []string
	for n := range credentialExact {
		names = append(names, n)
	}
	for e := range credentialExt {
		names = append(names, "server"+e)
	}
	for stem := range credentialStem {
		for e := range credentialStemExt {
			names = append(names, stem+e)
		}
	}
	names = append(names, ".env.local", ".env.production", ".env.example", ".env.sample", ".env.template")
	slices.Sort(names)
	return names
}

// TestTurnNeverStored is criterion 65.2 on the files a normal repository
// holds: each file that must not reach the store holds a marker, looked for
// in every object the store holds afterwards, none found; the files that may
// are found, so that the search proves something.
func TestTurnNeverStored(t *testing.T) {
	t.Parallel()
	dir := srcRepo(t)
	excludes := filepath.Join(t.TempDir(), "global-excludes")
	write(t, excludes, "global-only.txt\n")
	gitT(t, dir, "config", "core.excludesFile", excludes)
	writeAt(t, dir, ".gitignore", "ignored.txt\nbuild/\n")
	writeAt(t, dir, ".git/info/exclude", "local-only.txt\n")
	writeAt(t, dir, "normal.txt", "normal\n")
	writeAt(t, dir, "config/auth.json", "old\n")
	gitT(t, dir, "add", "-A")
	gitT(t, dir, "commit", "-q", "-m", "base")
	s := openTurns(t, dir)
	takeTree(t, s)

	never, may := map[string]string{}, map[string]string{} // marker -> file
	put := func(markers map[string]string, rel string) {
		m := fmt.Sprintf("MARKER-%04d", len(never)+len(may)) // none a prefix of another
		markers[m] = rel
		writeAt(t, dir, rel, m+"\n")
	}
	for _, rel := range []string{"ignored.txt", "build/out.txt", "local-only.txt", "global-only.txt", "config/auth.json"} {
		put(never, rel) // by .gitignore, a folder of it, .git/info/exclude, core.excludesFile; a tracked credential file
	}
	for _, n := range credentialFiles() {
		put(never, n)
		put(never, "sub/"+strings.ToUpper(n))
	}
	// A name Go's case folding refuses and git's does not: the Kelvin sign
	// folds to "k". The pathspecs let it through, the viewer's checks do not.
	kelvin := "kelvin." + string(rune(0x212A)) + "ey"
	if !CredentialName(kelvin) {
		t.Fatalf("CredentialName(%q) is false: the test's premise fails", kelvin)
	}
	put(never, kelvin)
	for _, rel := range []string{"secret.md", "token.go", "tokenizer.json", "env.go", "authors.json", "docs/secrets/readme.md",
		"internal/token/parser.go", "docs/.env.d/notes.txt"} {
		put(may, rel) // near misses of the names, and folders called as they are
	}
	// A name that is a pattern for another: "[a]uth.json" matches auth.json, a
	// credential file, unless git takes its pathspecs literally.
	put(may, "[a]uth.json")

	tt := takeTree(t, s)
	all := storeObjects(t, s)
	for m, rel := range never {
		if strings.Contains(all, m) {
			t.Errorf("%s reached the store", rel)
		}
	}
	for m, rel := range may {
		if !strings.Contains(all, m) {
			t.Errorf("%s did not reach the store, or the search finds nothing", rel)
		}
	}
	for _, p := range pathsIn(t, s, tt.Tree) {
		if p == "config/auth.json" {
			continue // the repository's own, as it was committed
		}
		if base := path.Base(p); CredentialName(base) || strings.HasPrefix(strings.ToLower(base), ".env.") {
			t.Errorf("the tree lists %s", p)
		}
	}
	// Git itself leaves out what the pathspecs name: only the Kelvin file is
	// left out by the checks.
	if len(tt.Left) != 1 || tt.Left[0].Path != kelvin || tt.Left[0].Why != "a credential file" {
		t.Errorf("left out: %+v, want %q as a credential file", tt.Left, kelvin)
	}
}

// TestTurnLinks is criterion 65.2's links, where the system lets the test
// make them: a link to a credential file and a link to a file outside the
// project are left out and never read; a link inside is no problem.
func TestTurnLinks(t *testing.T) {
	t.Parallel()
	dir := turnProject(t, map[string]string{"normal.txt": "normal\n"})
	outside := t.TempDir()
	writeAt(t, outside, "far.txt", "MARKER-outside-link\n")
	writeAt(t, dir, ".env", "MARKER-credential-link\n")
	links := map[string]string{
		"settings.txt": filepath.Join(dir, ".env"),
		"far-link.txt": filepath.Join(outside, "far.txt"),
		"inner.txt":    filepath.Join(dir, "normal.txt"),
	}
	for name, target := range links {
		if err := os.Symlink(target, filepath.Join(dir, name)); err != nil {
			t.Skipf("symbolic links cannot be made here, their case is not verified: %v", err)
		}
	}
	s := openTurns(t, dir)
	tt := takeTree(t, s)
	all := storeObjects(t, s)
	for _, m := range []string{"MARKER-outside-link", "MARKER-credential-link"} {
		if strings.Contains(all, m) {
			t.Errorf("%s reached the store", m)
		}
	}
	got := map[string]string{}
	for _, l := range tt.Left {
		got[l.Path] = l.Why
	}
	if want := map[string]string{"settings.txt": "a credential file", "far-link.txt": "outside the project"}; !reflect.DeepEqual(got, want) {
		t.Errorf("left out %v, want %v", got, want)
	}
	paths := pathsIn(t, s, tt.Tree)
	if slices.Contains(paths, "settings.txt") || slices.Contains(paths, "far-link.txt") || !slices.Contains(paths, "inner.txt") {
		t.Errorf("the tree holds %v", paths)
	}
}

// TestTurnTooLarge is criterion 65.2's large file, with MaxFileSize lowered
// instead of a large fixture, and design D5's "still changing": a file over
// the limit, tracked or not, is left out with its size and time, its content
// never read; it counts as changing only when its size or time change; one
// brought under the limit is taken.
func TestTurnTooLarge(t *testing.T) {
	defer func(n int64) { MaxFileSize = n }(MaxFileSize)
	MaxFileSize = 100
	dir := turnProject(t, map[string]string{"small.txt": "s\n", "tracked-big.txt": "t\n"})
	s := openTurns(t, dir)
	takeTree(t, s)

	writeAt(t, dir, "big.bin", "MARKER-large-file\n"+strings.Repeat("x", 100))
	writeAt(t, dir, "tracked-big.txt", "MARKER-tracked-large\n"+strings.Repeat("y", 100))
	writeAt(t, dir, "edge.txt", strings.Repeat("e", 100))
	tt := takeTree(t, s)
	all := storeObjects(t, s)
	for _, m := range []string{"MARKER-large-file", "MARKER-tracked-large"} {
		if strings.Contains(all, m) {
			t.Errorf("%s reached the store", m)
		}
	}
	var want []TurnLeft
	for _, name := range []string{"big.bin", "tracked-big.txt"} {
		info, err := os.Lstat(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		want = append(want, TurnLeft{Path: name, Why: "over 100 bytes", Size: info.Size(), ModTime: info.ModTime()})
	}
	if len(tt.Left) != len(want) {
		t.Errorf("left out %+v, want %+v", tt.Left, want)
	}
	for i, l := range tt.Left[:min(len(tt.Left), len(want))] {
		if w := want[i]; l.Path != w.Path || l.Why != w.Why || l.Size != w.Size || !l.ModTime.Equal(w.ModTime) {
			t.Errorf("left out %+v, want %+v", l, w)
		}
	}
	if paths := pathsIn(t, s, tt.Tree); !slices.Contains(paths, "edge.txt") || !slices.Contains(paths, "small.txt") {
		t.Errorf("the tree holds %v: a file exactly at the limit is taken", paths)
	}
	if blob := blobIn(t, s, tt.Tree, "tracked-big.txt"); blob != "t\n" {
		t.Errorf("the tracked large file is stored as %q, want the repository's version", blob)
	}

	// Left out and unchanged is not changing; changed in size or in time is.
	changing := func() []string {
		t.Helper()
		got, err := s.Changing(turnCtx(t))
		if err != nil {
			t.Fatal(err)
		}
		return got
	}
	if got := changing(); len(got) != 0 {
		t.Errorf("Changing lists %v after a Take that left them out", got)
	}
	writeAt(t, dir, "big.bin", "MARKER-large-file\n"+strings.Repeat("x", 105))
	later := time.Now().Add(time.Hour)
	if err := os.Chtimes(filepath.Join(dir, "tracked-big.txt"), later, later); err != nil {
		t.Fatal(err)
	}
	if got := changing(); !slices.Equal(got, []string{"big.bin", "tracked-big.txt"}) {
		t.Errorf("Changing lists %v, want both large files, one longer and one touched", got)
	}
	if again := takeTree(t, s); again.Tree != tt.Tree || len(again.Left) != 2 {
		t.Errorf("the next Take: %s with %d left out, want the same tree and both left out again", again.Tree, len(again.Left))
	}
	if got := changing(); len(got) != 0 {
		t.Errorf("Changing lists %v after the next Take", got)
	}

	// Brought under the limit, it is taken.
	writeAt(t, dir, "big.bin", "small now\n")
	tt = takeTree(t, s)
	if len(tt.Left) != 1 || tt.Left[0].Path != "tracked-big.txt" || blobIn(t, s, tt.Tree, "big.bin") != "small now\n" {
		t.Errorf("after the shrink: left out %+v", tt.Left)
	}
}

// TestTurnChanging is design D5's listing after the end: the files written
// since the last Take, sorted, none once a Take has taken them.
func TestTurnChanging(t *testing.T) {
	t.Parallel()
	dir := turnProject(t, map[string]string{"a.txt": "a\n", "b.txt": "b\n", "sub/c.txt": "c\n"})
	s := openTurns(t, dir)
	changing := func() []string {
		t.Helper()
		got, err := s.Changing(turnCtx(t))
		if err != nil {
			t.Fatal(err)
		}
		return got
	}
	if got := changing(); len(got) != 0 {
		t.Errorf("a clean checkout is changing: %v", got)
	}
	takeTree(t, s)
	writeAt(t, dir, "sub/c.txt", "c2\n")
	writeAt(t, dir, "new.txt", "n\n")
	if err := os.Remove(filepath.Join(dir, "a.txt")); err != nil {
		t.Fatal(err)
	}
	if got := changing(); !slices.Equal(got, []string{"a.txt", "new.txt", "sub/c.txt"}) {
		t.Errorf("Changing lists %v", got)
	}
	takeTree(t, s)
	if got := changing(); len(got) != 0 {
		t.Errorf("Changing lists %v after the Take", got)
	}
	writeAt(t, dir, "after.txt", "late\n")
	if got := changing(); !slices.Equal(got, []string{"after.txt"}) {
		t.Errorf("a file written after the Take: %v", got)
	}
}

// TestTurnGrowth is criterion 65.3's growth: eight turns over a project of
// about 2 MiB of tracked files, with no change, one file changed and every
// file touched. The store grows by the changed file alone, within what the
// projection counts for it; a turn with no change or only touched files
// adds nothing, from the listing alone, and the listing refreshes the store's
// index so that a touched file is hashed once.
func TestTurnGrowth(t *testing.T) {
	t.Parallel()
	files := map[string]string{}
	for i := 0; i < 20; i++ {
		var b strings.Builder
		for l := 0; l < 1400; l++ {
			fmt.Fprintf(&b, "file %d line %d %s\n", i, l, strings.Repeat("x", 60))
		}
		files[fmt.Sprintf("f%02d.txt", i)] = b.String()
	}
	dir := turnProject(t, files)
	s := openTurns(t, dir)
	calls := traceTurns(s)
	prev := takeTree(t, s)
	base := s.Size()
	size := int64(len(files["f00.txt"]))
	if size < 90<<10 {
		t.Fatalf("a file is %d bytes", size)
	}
	// The seeded first snapshot takes nothing in: the tracked files stay in the
	// repository's objects, the store holds their trees.
	if base > 8<<10 || strings.Contains(storeObjects(t, s), "file 7 line 700") {
		t.Errorf("the first snapshot took %d bytes of 2 MiB of tracked files, or copied one", base)
	}

	var firstGrowth int64
	for turn, kind := range []string{"one", "none", "touched", "one", "none", "touched", "one", "none"} {
		before, mark := s.Size(), len(calls.all())
		indexBefore := readFile(t, filepath.Join(s.dir, turnIndex))
		switch kind {
		case "one":
			f := filepath.Join(dir, fmt.Sprintf("f%02d.txt", turn))
			appendTo(t, f, fmt.Sprintf("turn %d\n", turn))
		case "touched":
			at := time.Now().Add(time.Duration(turn+1) * time.Hour)
			for name := range files {
				if err := os.Chtimes(filepath.Join(dir, name), at, at); err != nil {
					t.Fatal(err)
				}
			}
		}
		tt := takeTree(t, s)
		growth, subs := s.Size()-before, calls.subs()[mark:]
		switch kind {
		case "one":
			if tt.Tree == prev.Tree {
				t.Errorf("turn %d (%s): the tree did not change", turn, kind)
			}
			if growth <= 0 || growth > turnCost(size) {
				t.Errorf("turn %d (%s): the store grew by %d bytes, want 1 to %d", turn, kind, growth, turnCost(size))
			}
			if firstGrowth == 0 {
				firstGrowth = growth
			} else if growth > 2*firstGrowth {
				t.Errorf("turn %d (%s): the store grew by %d bytes, the first such turn by %d", turn, kind, growth, firstGrowth)
			}
			if !slices.Equal(subs, []string{"status", "add", "write-tree"}) {
				t.Errorf("turn %d (%s): ran %q", turn, kind, subs)
			}
		default:
			if tt.Tree != prev.Tree || growth != 0 {
				t.Errorf("turn %d (%s): tree %s then %s, the store grew by %d bytes", turn, kind, prev.Tree, tt.Tree, growth)
			}
			if !slices.Equal(subs, []string{"status"}) {
				t.Errorf("turn %d (%s): ran %q, want the listing alone", turn, kind, subs)
			}
		}
		if kind == "touched" && readFile(t, filepath.Join(s.dir, turnIndex)) == indexBefore {
			t.Errorf("turn %d (%s): the listing did not refresh the store's index", turn, kind)
		}
		if got, err := s.Changing(turnCtx(t)); err != nil || len(got) != 0 {
			t.Errorf("turn %d (%s): Changing lists %v, %v", turn, kind, got, err)
		}
		prev = tt
	}
	if total := s.Size() - base; total <= 0 || total >= 1<<20 {
		t.Errorf("eight turns grew the store by %d bytes, want under 1 MiB", total)
	}
}

// appendTo adds text to the end of a file.
func appendTo(t *testing.T, path, text string) {
	t.Helper()
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString(text); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
}

// TestTurnLimit is criterion 65.3's limit, on a limit the test sets: a Take
// whose projection passes it is refused before git takes anything and leaves
// the store as it was; one that ends past it succeeds and every later Take is
// refused at once; the earlier trees stay readable.
func TestTurnLimit(t *testing.T) {
	defer func(n int64) { TurnStoreLimit = n }(TurnStoreLimit)
	files := map[string]string{"a.txt": "a\n", "b.txt": "b\n"}
	for i := 0; i < 400; i++ { // a folder whose tree weighs, when one file of it changes
		files[fmt.Sprintf("many/file-%03d.txt", i)] = fmt.Sprintf("x%d\n", i) // each its own blob: a tree of 400 distinct ids
	}
	dir := turnProject(t, files)
	s := openTurns(t, dir)
	calls := traceTurns(s)
	first := takeTree(t, s)
	base := s.Size()
	if base == 0 {
		t.Fatal("the first Take wrote nothing: the test's premise fails")
	}

	// The projection of a 3,000-byte file passes the limit.
	TurnStoreLimit = base + 2000
	writeAt(t, dir, "a.txt", strings.Repeat("0123456789abcdef", 190))
	objects, mark := storeFiles(t, s), len(calls.all())
	if _, err := s.Take(turnCtx(t)); !errors.Is(err, ErrTurnStoreFull) {
		t.Fatalf("Take: %v, want ErrTurnStoreFull", err)
	}
	if got := calls.subs()[mark:]; !slices.Equal(got, []string{"status"}) {
		t.Errorf("the refused Take ran %q, want the listing alone", got)
	}
	if s.Size() != base || !reflect.DeepEqual(storeFiles(t, s), objects) {
		t.Errorf("the refused Take changed the store: %d bytes, was %d", s.Size(), base)
	}

	// Nothing was lost: with room, the same Take works.
	TurnStoreLimit = 256 << 20
	second := takeTree(t, s)
	if second.Tree == first.Tree {
		t.Fatal("the Take after the refusal took nothing")
	}

	// A small file of the big folder passes the projection (1,026 bytes) and
	// rewrites the folder's tree, which is more than the room left: the store
	// ends past the limit, that Take succeeds, the next ones are refused.
	TurnStoreLimit = s.Size() + 4000
	writeAt(t, dir, "many/file-007.txt", "changed\n")
	third, err := s.Take(turnCtx(t))
	if err != nil {
		t.Fatalf("the Take that ends past the limit: %v", err)
	}
	if s.Size() <= TurnStoreLimit {
		t.Fatalf("the store is %d bytes, not past the limit %d: the test's premise fails", s.Size(), TurnStoreLimit)
	}
	mark = len(calls.all())
	for i := 0; i < 2; i++ {
		if _, err := s.Take(turnCtx(t)); !errors.Is(err, ErrTurnStoreFull) {
			t.Errorf("a Take after the limit: %v, want ErrTurnStoreFull", err)
		}
	}
	if n := len(calls.all()) - mark; n != 0 {
		t.Errorf("the refused Takes ran %d git calls, want none", n)
	}

	// The earlier trees are readable.
	cf, err := s.Files(first.Tree, third.Tree)
	if err != nil || !slices.Equal(filesOf(cf), []string{"M a.txt", "M many/file-007.txt"}) {
		t.Errorf("Files after the limit: %v, %v", filesOf(cf), err)
	}
	pair, err := s.Sides(second.Tree, third.Tree, CommitFile{Status: 'M', Path: "many/file-007.txt"})
	if err != nil || string(pair.Old.Data) != "x7\n" || string(pair.New.Data) != "changed\n" {
		t.Errorf("Sides after the limit: %q, %q, %v", pair.Old.Data, pair.New.Data, err)
	}
	if _, err := s.Changing(turnCtx(t)); err != nil {
		t.Errorf("Changing after the limit: %v", err)
	}
}

// TestTurnClose is criterion 65.3's end: Close removes the folder, every call
// after it says the store is closed, and Close waits for a call under way.
func TestTurnClose(t *testing.T) {
	t.Parallel()
	dir := turnProject(t, map[string]string{"a.txt": "a\n"})
	root := filepath.Join(t.TempDir(), "turns")
	s, err := openTurnStore(dir, root)
	if err != nil {
		t.Fatal(err)
	}
	tt := takeTree(t, s)
	if !exists(s.dir) {
		t.Fatal("the store has no folder")
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	if exists(s.dir) {
		t.Errorf("Close left %s", s.dir)
	}
	if es, err := os.ReadDir(root); err != nil || len(es) != 0 {
		t.Errorf("the stores' folder holds %v, %v", es, err)
	}
	closed := func(what string, err error) {
		t.Helper()
		if err == nil || !strings.Contains(err.Error(), "closed") {
			t.Errorf("%s after Close: %v, want an error saying the store is closed", what, err)
		}
	}
	_, err = s.Take(turnCtx(t))
	closed("Take", err)
	_, err = s.Changing(turnCtx(t))
	closed("Changing", err)
	_, err = s.Files(tt.Tree, tt.Tree)
	closed("Files", err)
	_, err = s.Sides(tt.Tree, tt.Tree, CommitFile{Status: 'M', Path: "a.txt"})
	closed("Sides", err)
	if err := s.Close(); err != nil {
		t.Errorf("Close twice: %v", err)
	}
}

// TestTurnCloseWaits: a Take under way is not cut by Close; Close returns once
// it has ended, and the folder is gone.
func TestTurnCloseWaits(t *testing.T) {
	t.Parallel()
	dir := turnProject(t, map[string]string{"a.txt": "a\n"})
	s := openTurns(t, dir)
	var once sync.Once
	started, gate := make(chan struct{}), make(chan struct{})
	s.git.Trace = func(args []string) {
		if subcommand(args) == "status" {
			once.Do(func() { close(started) })
			<-gate
		}
	}
	ctx := turnCtx(t)
	taken := make(chan error, 1)
	go func() {
		_, err := s.Take(ctx)
		taken <- err
	}()
	<-started
	closed := make(chan error, 1)
	go func() { closed <- s.Close() }()
	select {
	case err := <-closed:
		t.Fatalf("Close returned while a Take was under way: %v", err)
	case <-time.After(300 * time.Millisecond):
	}
	close(gate)
	if err := <-taken; err != nil {
		t.Errorf("the Take under way: %v", err)
	}
	if err := <-closed; err != nil {
		t.Errorf("Close: %v", err)
	}
	if exists(s.dir) {
		t.Errorf("Close left %s", s.dir)
	}
}

// TestTurnSweep is criterion 65.3's sweep (design D3): the next
// OpenTurnStore removes a store whose lock is not held, a folder without a
// lock a minute old and never one that is held, a younger one without a lock
// (being made), nor what is no store's.
func TestTurnSweep(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("a held file is Windows' (plan 33, design D1)")
	}
	t.Parallel()
	dir := turnProject(t, map[string]string{"a.txt": "a\n"})
	root := filepath.Join(t.TempDir(), "turns")
	open := func() *TurnStore {
		t.Helper()
		s, err := openTurnStore(dir, root)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { s.Close() })
		return s
	}
	crashed, held := open(), open()
	takeTree(t, held)
	heldFiles := storeFiles(t, held)
	// What a killed dashboard leaves: the folder, its lock released.
	if err := crashed.lock.Close(); err != nil {
		t.Fatal(err)
	}
	mk := func(name string) string {
		p := filepath.Join(root, name)
		if err := os.Mkdir(p, 0o700); err != nil {
			t.Fatal(err)
		}
		return p
	}
	old, young, foreign := mk("00000000000000a1"), mk("00000000000000a2"), mk("not-a-store")
	age(t, old, 2*time.Minute)
	age(t, foreign, 2*time.Minute)
	write(t, filepath.Join(root, "note.txt"), "x")

	third := open()
	for _, p := range []string{crashed.dir, old} {
		if exists(p) {
			t.Errorf("%s was not swept", filepath.Base(p))
		}
	}
	for _, p := range []string{held.dir, young, foreign, filepath.Join(root, "note.txt"), third.dir} {
		if !exists(p) {
			t.Errorf("%s was swept", filepath.Base(p))
		}
	}
	if !reflect.DeepEqual(storeFiles(t, held), heldFiles) {
		t.Error("the sweep touched the held store")
	}
	writeAt(t, dir, "b.txt", "b\n")
	if tt := takeTree(t, held); !slices.Contains(pathsIn(t, held, tt.Tree), "b.txt") {
		t.Error("the held store no longer works")
	}
}

// TestTurnDeadline is criterion 65.3's deadline (design D2): a Take whose
// context is past its deadline fails with a timeout, and the next Take works,
// with no index.lock left; so after a git that was stopped midway, which
// leaves the lock behind, and after an add whose write-tree did not follow.
func TestTurnDeadline(t *testing.T) {
	t.Parallel()
	dir := turnProject(t, map[string]string{"a.txt": "a\n"})
	s := openTurns(t, dir)
	takeTree(t, s)
	lock := filepath.Join(s.dir, "index.lock")

	writeAt(t, dir, "new.txt", "n\n")
	past, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()
	_, err := s.Take(past)
	var te *timeoutError
	if !errors.As(err, &te) || !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("a Take past its deadline: %v, want a timeout", err)
	}
	if exists(lock) {
		t.Error("index.lock left after the timeout")
	}
	if _, err := s.Changing(past); !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("Changing past its deadline: %v", err)
	}
	second := takeTree(t, s)
	if !slices.Contains(pathsIn(t, s, second.Tree), "new.txt") {
		t.Error("the Take after the timeout did not take new.txt")
	}

	// A git stopped midway leaves its lock: the add is stopped as it starts, the
	// lock written as a killed git leaves it.
	writeAt(t, dir, "more.txt", "m\n")
	ctx, stop := context.WithCancel(context.Background())
	defer stop()
	s.git.Trace = func(args []string) {
		if subcommand(args) == "add" {
			write(t, lock, "")
			stop()
		}
	}
	if _, err := s.Take(ctx); !errors.Is(err, context.Canceled) {
		t.Errorf("a Take stopped at its add: %v", err)
	}
	if exists(lock) {
		t.Error("index.lock left after the stopped add")
	}
	s.git.Trace = nil
	third := takeTree(t, s)
	cf, err := s.Files(second.Tree, third.Tree)
	if err != nil || !slices.Equal(filesOf(cf), []string{"A more.txt"}) {
		t.Errorf("the Take after the stopped add: %v, %v", filesOf(cf), err)
	}

	// The listing refreshes the index under its lock as well.
	ctx, stop3 := context.WithCancel(context.Background())
	defer stop3()
	s.git.Trace = func(args []string) {
		if subcommand(args) == "status" {
			write(t, lock, "")
			stop3()
		}
	}
	if _, err := s.Changing(ctx); !errors.Is(err, context.Canceled) {
		t.Errorf("Changing stopped at its listing: %v", err)
	}
	if exists(lock) {
		t.Error("index.lock left after the stopped listing")
	}
	s.git.Trace = nil

	// The index took the file in, the tree was not written: the next Take
	// writes it though the listing shows nothing.
	writeAt(t, dir, "last.txt", "l\n")
	ctx, stop2 := context.WithCancel(context.Background())
	defer stop2()
	s.git.Trace = func(args []string) {
		if subcommand(args) == "write-tree" {
			stop2()
		}
	}
	if _, err := s.Take(ctx); !errors.Is(err, context.Canceled) {
		t.Errorf("a Take stopped at its write-tree: %v", err)
	}
	s.git.Trace = nil
	fourth := takeTree(t, s)
	cf, err = s.Files(third.Tree, fourth.Tree)
	if err != nil || !slices.Equal(filesOf(cf), []string{"A last.txt"}) {
		t.Errorf("the Take after the stopped write-tree: %v, %v; the file is lost", filesOf(cf), err)
	}
}

// TestTurnOpen covers OpenTurnStore: the stores' folder is turnsRoot's, a
// repository with no commit and no index opens, a folder outside a repository
// is a NotRepoError and git missing is ErrNoGit, neither leaving a folder.
func TestTurnOpen(t *testing.T) {
	root := filepath.Join(t.TempDir(), "turns")
	defer func(r string) { turnsRoot = r }(turnsRoot)
	turnsRoot = root

	dir := srcRepo(t) // no commit, no index
	writeAt(t, dir, "first.txt", "first\n")
	s, err := OpenTurnStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Dir(s.dir) != root || !isHex(filepath.Base(s.dir), keyDigits) {
		t.Errorf("the store is in %s, want a folder of 16 hexadecimal digits in %s", s.dir, root)
	}
	if got := files(t, s.dir); !slices.Equal(got, []string{"hooks", "lock", "objects"}) {
		t.Errorf("a new store holds %v: no index when the repository has none", got)
	}
	if es, _ := os.ReadDir(filepath.Join(s.dir, turnHooks)); len(es) != 0 {
		t.Errorf("the hooks folder is not empty: %v", es)
	}
	tt := takeTree(t, s)
	if got := pathsIn(t, s, tt.Tree); !slices.Equal(got, []string{"first.txt"}) {
		t.Errorf("the first tree of a repository with no commit: %v", got)
	}
	if hooks := files(t, filepath.Join(s.dir, turnHooks)); len(hooks) != 0 {
		t.Errorf("the hooks folder was written: %v", hooks)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}

	plain := t.TempDir()
	var nr NotRepoError
	if _, err := OpenTurnStore(plain); !errors.As(err, &nr) || nr.Dir != plain {
		t.Errorf("outside a repository: %v, want a NotRepoError for %s", err, plain)
	}
	t.Setenv("PATH", t.TempDir())
	if _, err := OpenTurnStore(dir); !errors.Is(err, ErrNoGit) {
		t.Errorf("without git: %v, want ErrNoGit", err)
	}
	if es, err := os.ReadDir(root); err != nil || len(es) != 0 {
		t.Errorf("the failures left %v in the stores' folder, %v", es, err)
	}
}

// TestTurnSubfolder: a project that is a subfolder of its repository lists and
// takes its own files alone, the paths from the repository's top level, and a
// nested repository (git lists the folder, not its files) is left out as such
// without making the Take fail (it would make git add fail).
func TestTurnSubfolder(t *testing.T) {
	t.Parallel()
	// The second name holds the characters of a pattern, a space and an accent.
	for _, rel := range []string{"app", "[odd] é"} {
		t.Run(rel, func(t *testing.T) {
			t.Parallel()
			top := turnProject(t, map[string]string{"lib/x.txt": "x\n", "o é": "o\n", rel + "/y.txt": "y\n", rel + "/inner/z.txt": "z\n"})
			project := filepath.Join(top, rel)
			s := openTurns(t, project)
			start := takeTree(t, s)
			writeAt(t, top, "lib/x.txt", "x changed\n") // outside the project
			writeAt(t, top, "o é", "o changed\n")       // "[odd] é" is a pattern: it matches a file called "o é"
			writeAt(t, project, "y.txt", "y changed\n")
			writeAt(t, project, "new.txt", "n\n")
			writeAt(t, project, "auth.json", "MARKER-subfolder-credential\n")
			nested := filepath.Join(project, "vendor")
			if err := os.MkdirAll(nested, 0o755); err != nil {
				t.Fatal(err)
			}
			gitT(t, nested, "init", "-q", "-b", "main")
			writeAt(t, nested, "v.txt", "v\n")
			gitT(t, nested, "add", "-A")
			gitT(t, nested, "commit", "-q", "-m", "v")
			end := takeTree(t, s)

			cf, err := s.Files(start.Tree, end.Tree)
			if err != nil {
				t.Fatal(err)
			}
			if want := []string{"A " + rel + "/new.txt", "M " + rel + "/y.txt"}; !slices.Equal(filesOf(cf), want) {
				t.Errorf("Files: %q, want %q", filesOf(cf), want)
			}
			if strings.Contains(storeObjects(t, s), "MARKER-subfolder-credential") {
				t.Error("a credential file of the subfolder reached the store")
			}
			if len(end.Left) != 1 || end.Left[0].Path != rel+"/vendor/" || end.Left[0].Why != whyNested {
				t.Errorf("left out %+v, want %s/vendor/ as a nested repository", end.Left, rel)
			}
			if got, err := s.Changing(turnCtx(t)); err != nil || len(got) != 0 {
				t.Errorf("Changing: %v, %v", got, err)
			}
		})
	}
}

// TestTurnSubmodule: a submodule's record is taken without checks, its commit
// only, as git records it (design D6).
func TestTurnSubmodule(t *testing.T) {
	t.Parallel()
	sub := srcRepo(t)
	writeAt(t, sub, "s.txt", "s1\n")
	gitT(t, sub, "add", "-A")
	gitT(t, sub, "commit", "-q", "-m", "s1")
	dir := turnProject(t, map[string]string{"a.txt": "a\n"})
	gitT(t, dir, "-c", "protocol.file.allow=always", "submodule", "add", "-q", sub, "libs/sub")
	gitT(t, dir, "commit", "-q", "-m", "submodule")
	s := openTurns(t, dir)
	start := takeTree(t, s)

	inner := filepath.Join(dir, "libs", "sub")
	writeAt(t, inner, "s.txt", "s2\n")
	gitT(t, inner, "commit", "-q", "-am", "s2")
	writeAt(t, inner, "untracked.txt", "u\n") // dirty content of a submodule is not git's concern here
	end := takeTree(t, s)
	if end.Tree == start.Tree {
		t.Fatal("a submodule's new commit gave the same tree")
	}
	cf, err := s.Files(start.Tree, end.Tree)
	if err != nil || !slices.Equal(filesOf(cf), []string{"M libs/sub"}) || len(end.Left) != 0 {
		t.Errorf("Files: %q, %v; left out %+v", filesOf(cf), err, end.Left)
	}
	if got, err := s.Changing(turnCtx(t)); err != nil || len(got) != 0 {
		t.Errorf("Changing: %v, %v", got, err)
	}
}

// TestParseTurnStatus reads git status --porcelain=v2 -z as the listing
// needs it.
func TestParseTurnStatus(t *testing.T) {
	t.Parallel()
	const h = "1111111111111111111111111111111111111111"
	rec := func(fields ...string) string { return strings.Join(fields, " ") }
	out := strings.Join([]string{
		rec("1", ".M", "N...", "100644", "100644", "100644", h, h, "a b.txt"), // a path with a space
		rec("1", "M.", "N...", "100644", "100644", "100644", h, h, "staged.txt"),
		rec("1", "MD", "N...", "100644", "100644", "000000", h, h, "gone.txt"),
		rec("1", ".M", "SC..", "160000", "160000", "160000", h, h, "libs/sub"),
		rec("u", "UU", "N...", "100644", "100644", "100644", "100644", h, h, h, "conflict.txt"),
		rec("?", "new file.txt"),
		rec("?", "nested/"),
		rec("!", "ignored.txt"),
		"",
	}, "\x00")
	got, err := parseTurnStatus(out)
	if err != nil {
		t.Fatal(err)
	}
	want := []turnEntry{{path: "a b.txt"}, {path: "gone.txt"}, {path: "libs/sub", module: true}, {path: "conflict.txt"},
		{path: "new file.txt", untracked: true}, {path: "nested/", untracked: true, folder: true}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("parsed %+v, want %+v", got, want)
	}
	if got, err := parseTurnStatus(""); err != nil || len(got) != 0 {
		t.Errorf("an empty listing: %v, %v", got, err)
	}
	for _, bad := range []string{
		rec("2", "R.", "N...", "100644", "100644", "100644", h, h, "R100", "new.txt") + "\x00old.txt\x00",
		rec("1", ".M", "N..."),
		rec("1", "M", "N...", "100644", "100644", "100644", h, h, "x"),
		"?",
		"x something",
	} {
		if got, err := parseTurnStatus(bad); err == nil {
			t.Errorf("parseTurnStatus(%q) = %+v, want an error", bad, got)
		}
	}
}

// TestTurnCredentialExcludes: the pathspecs are the viewer's tables, each name
// without case in any folder, and ".env.*" besides.
func TestTurnCredentialExcludes(t *testing.T) {
	t.Parallel()
	got := credentialExcludes()
	if !slices.IsSorted(got) {
		t.Error("the pathspecs are not in a fixed order")
	}
	have := map[string]bool{}
	for _, g := range got {
		if !strings.HasPrefix(g, ":(exclude,icase,glob)**/") {
			t.Errorf("%q is no case-less exclusion in any folder", g)
		}
		have[strings.TrimPrefix(g, ":(exclude,icase,glob)**/")] = true
	}
	want := len(credentialExact) + len(credentialExt) + len(credentialStem)*len(credentialStemExt) + 1
	if len(got) != want || len(have) != want {
		t.Errorf("%d pathspecs (%d different), want %d", len(got), len(have), want)
	}
	for _, n := range append(credentialFiles(), "auth.json", "secrets", "token.txt") {
		covered := have[n] || have["*"+path.Ext(n)] || strings.HasPrefix(n, ".env.") && have[".env.*"]
		if !covered {
			t.Errorf("%q is named by no pathspec", n)
		}
	}
	for _, n := range []string{"secret.md", "token.go", "env.go", ".env.d"} {
		if have[n] {
			t.Errorf("%q is a pathspec", n)
		}
	}
}

// TestTurnCheck makes the checks on listed entries that no git listed: a file
// that came and went, a deletion, a folder, a submodule.
func TestTurnCheck(t *testing.T) {
	t.Parallel()
	dir := turnProject(t, map[string]string{"a.txt": "a\n"})
	s := openTurns(t, dir)
	writeAt(t, dir, "new.txt", "n\n")
	for _, d := range []string{"nest", "mod", "plain"} {
		if err := os.Mkdir(filepath.Join(dir, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	sc, err := s.check(turnCtx(t), []turnEntry{
		{path: "new.txt", untracked: true},
		{path: "ghost.txt", untracked: true}, // came and went: nothing to record
		{path: "removed.txt"},                // tracked, gone: a deletion
		{path: "nest/", untracked: true, folder: true},
		{path: "mod", module: true}, // a folder, but git records its commit
		{path: "plain"},             // a folder git listed as a file: replaced since
		{path: "under/ghost.txt"},   // below a folder that is gone too
		{path: "a.txt/child"},       // below a file: gone as well
	})
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"new.txt", "removed.txt", "mod", "under/ghost.txt", "a.txt/child"}; !slices.Equal(sc.take, want) {
		t.Errorf("to take: %q, want %q", sc.take, want)
	}
	if want := turnCost(2) + 4*turnCost(0); sc.need != want {
		t.Errorf("the projection is %d, want %d", sc.need, want)
	}
	if len(sc.left) != 2 || sc.left[0].Path != "nest/" || sc.left[0].Why != whyNested || sc.left[1].Path != "plain" || sc.left[1].Why != "a directory" {
		t.Errorf("left out: %+v", sc.left)
	}
	// A deadline passed stops the checks.
	past, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()
	if _, err := s.check(past, []turnEntry{{path: "new.txt", untracked: true}}); !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("checks past the deadline: %v", err)
	}
}

// TestTurnGitCalls is design D2's list of what every call carries and never
// does: only the five commands of the snapshot, none that forces, commits,
// stashes or makes a reference; the repository's hooks, filters, file system
// monitor, split index and line-ending check off on every call; the add given
// its paths on standard input; the listing carrying the project's pathspec
// and the credential names.
func TestTurnGitCalls(t *testing.T) {
	t.Parallel()
	dir := srcRepo(t)
	gitT(t, dir, "config", "filter.lfs.clean", "git-lfs clean -- %f")
	gitT(t, dir, "config", "filter.lfs.process", "git-lfs filter-process")
	gitT(t, dir, "config", "filter.My.Driver.smudge", "cat")
	writeAt(t, dir, "app/a.txt", "a\n")
	gitT(t, dir, "add", "-A")
	gitT(t, dir, "commit", "-q", "-m", "base")
	s := openTurns(t, filepath.Join(dir, "app"))
	calls := traceTurns(s)
	start := takeTree(t, s)
	writeAt(t, dir, "app/b.txt", "b\n")
	end := takeTree(t, s)
	if _, err := s.Changing(turnCtx(t)); err != nil {
		t.Fatal(err)
	}
	cf, err := s.Files(start.Tree, end.Tree)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range cf.Files {
		if _, err := s.Sides(start.Tree, end.Tree, f); err != nil {
			t.Fatal(err)
		}
	}

	hooks := filepath.ToSlash(filepath.Join(s.dir, turnHooks))
	for _, c := range calls.all() {
		sub := subcommand(c)
		if !slices.Contains([]string{"status", "add", "write-tree", "diff-tree", "cat-file"}, sub) {
			t.Errorf("a %s call: %q", sub, c)
		}
		for _, a := range c {
			if a == "-f" || a == "--force" || strings.HasPrefix(a, "--force") {
				t.Errorf("a forcing call: %q", c)
			}
		}
		var opts []string
		for i := 0; i+1 < len(c); i++ {
			if c[i] == "-c" {
				opts = append(opts, c[i+1])
			}
		}
		for _, want := range []string{"core.hooksPath=" + hooks, "core.fsmonitor=false", "core.splitIndex=false", "core.safecrlf=false",
			"filter.lfs.clean=", "filter.lfs.process=", "filter.lfs.required=false", "filter.My.Driver.clean=",
			"filter.My.Driver.process=", "filter.My.Driver.required=false"} {
			if !slices.Contains(opts, want) {
				t.Errorf("%s call without -c %s: %q", sub, want, c)
			}
		}
		if len(c) < 2 || c[0] != "-C" || !samePath(c[1], s.top) {
			t.Errorf("%s call not made from the top level: %q", sub, c)
		}
		switch sub {
		case "add":
			if got := c[len(c)-4:]; !slices.Equal(got, []string{"add", "-A", "--pathspec-from-file=-", "--pathspec-file-nul"}) {
				t.Errorf("the add: %q", c)
			}
		case "status":
			i := slices.Index(c, "--")
			if i < 0 || c[i+1] != ":(literal)app" || len(c[i+1:]) != 1+len(credentialExcludes()) || c[i+2] != credentialExcludes()[0] {
				t.Errorf("the listing's pathspecs: %q", c[max(i, 0):])
			}
			for _, want := range []string{"--porcelain=v2", "-z", "--untracked-files=all", "--ignore-submodules=dirty", "--no-renames"} {
				if !slices.Contains(c, want) {
					t.Errorf("the listing lacks %s: %q", want, c)
				}
			}
		}
	}
	if got := calls.subs(); !slices.Contains(got, "add") || !slices.Contains(got, "write-tree") || !slices.Contains(got, "diff-tree") || !slices.Contains(got, "cat-file") {
		t.Errorf("the calls: %q", got)
	}

	// The environment: the store's index and objects on every call; the
	// repository's objects behind them for reads only, whatever the process's
	// environment says; optional locks for the listing alone.
	cmdEnv := func(g GitCLI) []string {
		cmd, err := g.command(turnCtx(t), s.top, true, []string{"status"})
		if err != nil {
			t.Fatal(err)
		}
		return cmd.Env
	}
	last := func(env []string, key string) string {
		v := "<unset>"
		for _, e := range env {
			if k, val, ok := strings.Cut(e, "="); ok && strings.EqualFold(k, key) {
				v = val
			}
		}
		return v
	}
	for _, c := range []struct {
		name      string
		g         GitCLI
		alt, lock string
	}{
		{"reads", s.reads(), filepath.ToSlash(s.objects), "0"},
		{"the listing", s.reads("GIT_OPTIONAL_LOCKS=1"), filepath.ToSlash(s.objects), "1"},
		{"writes", s.writes("GIT_LITERAL_PATHSPECS=1"), "", "0"},
	} {
		env := cmdEnv(c.g)
		if got := last(env, "GIT_INDEX_FILE"); !samePath(got, filepath.Join(s.dir, turnIndex)) {
			t.Errorf("%s: GIT_INDEX_FILE=%s", c.name, got)
		}
		if got := last(env, "GIT_OBJECT_DIRECTORY"); !samePath(got, filepath.Join(s.dir, turnObjects)) {
			t.Errorf("%s: GIT_OBJECT_DIRECTORY=%s", c.name, got)
		}
		if got := last(env, "GIT_ALTERNATE_OBJECT_DIRECTORIES"); got != c.alt {
			t.Errorf("%s: GIT_ALTERNATE_OBJECT_DIRECTORIES=%q, want %q", c.name, got, c.alt)
		}
		if got := last(env, "GIT_OPTIONAL_LOCKS"); got != c.lock {
			t.Errorf("%s: GIT_OPTIONAL_LOCKS=%s, want %s", c.name, got, c.lock)
		}
	}
	if got := last(cmdEnv(s.writes("GIT_LITERAL_PATHSPECS=1")), "GIT_LITERAL_PATHSPECS"); got != "1" {
		t.Errorf("the add's GIT_LITERAL_PATHSPECS=%s", got)
	}
	if got := last(cmdEnv(s.writes()), "GIT_LITERAL_PATHSPECS"); got != "<unset>" {
		t.Errorf("a write-tree's GIT_LITERAL_PATHSPECS=%s", got)
	}
	// Every other caller of GitCLI is as it was: no option, no variable.
	plain := GitCLI{}
	cmd, err := plain.command(turnCtx(t), s.top, false, []string{"status"})
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"-C", s.top, "-c", "core.quotepath=off", "status"}; !slices.Equal(cmd.Args[1:], want) {
		t.Errorf("a plain call: %q, want %q", cmd.Args[1:], want)
	}
	if last(cmd.Env, "GIT_INDEX_FILE") != "<unset>" || last(cmd.Env, "GIT_OBJECT_DIRECTORY") != "<unset>" {
		t.Error("a plain call carries the store's variables")
	}
}

// TestTurnConcurrent: the calls of one store from several goroutines at once
// give what they give alone; Take and Changing run one at a time.
func TestTurnConcurrent(t *testing.T) {
	t.Parallel()
	dir := turnProject(t, map[string]string{"a.txt": "a\n", "b.txt": "b\n"})
	s := openTurns(t, dir)
	start := takeTree(t, s)
	writeAt(t, dir, "a.txt", "a2\n")
	end := takeTree(t, s)
	ctx := turnCtx(t)
	var wg sync.WaitGroup
	for i := 0; i < 6; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 2; j++ {
				switch (i + j) % 4 {
				case 0:
					if tt, err := s.Take(ctx); err != nil || tt.Tree != end.Tree {
						t.Errorf("Take: %s, %v", tt.Tree, err)
					}
				case 1:
					if got, err := s.Changing(ctx); err != nil || len(got) != 0 {
						t.Errorf("Changing: %v, %v", got, err)
					}
				case 2:
					if cf, err := s.Files(start.Tree, end.Tree); err != nil || !slices.Equal(filesOf(cf), []string{"M a.txt"}) {
						t.Errorf("Files: %v, %v", filesOf(cf), err)
					}
				case 3:
					if pair, err := s.Sides(start.Tree, end.Tree, CommitFile{Status: 'M', Path: "a.txt"}); err != nil || string(pair.New.Data) != "a2\n" {
						t.Errorf("Sides: %q, %v", pair.New.Data, err)
					}
				}
			}
		}()
	}
	wg.Wait()
}
