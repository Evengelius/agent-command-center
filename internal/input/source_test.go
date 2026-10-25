package input

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/Evengelius/agent-command-center/internal/diff"
)

// recs joins git status records the way -z writes them.
func recs(parts ...string) string { return strings.Join(parts, "\x00") + "\x00" }

// ent is a GitEntry whose repository paths are its paths under prefix.
func ent(prefix string, x, y byte, path, orig string) GitEntry {
	e := GitEntry{X: x, Y: y, Path: path, Orig: orig, RepoPath: prefix + path}
	if orig != "" {
		e.RepoOrig = prefix + orig
	}
	return e
}

// TestParseSnapshot covers plan 44's criterion 44.2: recorded porcelain
// records (the `git status --porcelain=v1 -z --branch` of git 2.56.0 on
// temporary repositories, and the same letters for the cases git makes only
// in rarer states) give each entry's X and Y, its path, a rename's or copy's
// source and the branch, while the Explorer's markers stay what they were.
func TestParseSnapshot(t *testing.T) {
	t.Run("every letter", func(t *testing.T) {
		out := recs("## main",
			" M web/a.ts", "M  b.go", "MM c.go", " T d.sh", "T  e.sh", "A  new.go", "AM new2.go", " A intent.go",
			"R  web/to.ts", "web/from.ts", "RM web/to2.ts", "web/from2.ts", "C  copy.go", "orig.go",
			" D gone.go", "D  gone2.go", "AD addgone.go",
			"UU both.go", "AA add.go", "DD del.go", "AU au.go", "UA ua.go", "DU du.go", "UD ud.go",
			"?? notes.txt", "?? tmp/", "!! build/", "!! app.log")
		entries, branch, marks := ParseSnapshot(out, "")
		want := []GitEntry{
			ent("", ' ', 'M', "web/a.ts", ""), ent("", 'M', ' ', "b.go", ""), ent("", 'M', 'M', "c.go", ""),
			ent("", ' ', 'T', "d.sh", ""), ent("", 'T', ' ', "e.sh", ""),
			ent("", 'A', ' ', "new.go", ""), ent("", 'A', 'M', "new2.go", ""), ent("", ' ', 'A', "intent.go", ""),
			ent("", 'R', ' ', "web/to.ts", "web/from.ts"), ent("", 'R', 'M', "web/to2.ts", "web/from2.ts"),
			ent("", 'C', ' ', "copy.go", "orig.go"),
			ent("", ' ', 'D', "gone.go", ""), ent("", 'D', ' ', "gone2.go", ""), ent("", 'A', 'D', "addgone.go", ""),
			ent("", 'U', 'U', "both.go", ""), ent("", 'A', 'A', "add.go", ""), ent("", 'D', 'D', "del.go", ""),
			ent("", 'A', 'U', "au.go", ""), ent("", 'U', 'A', "ua.go", ""), ent("", 'D', 'U', "du.go", ""),
			ent("", 'U', 'D', "ud.go", ""),
			ent("", '?', '?', "notes.txt", ""), ent("", '?', '?', "tmp/", ""),
		}
		if !reflect.DeepEqual(entries, want) {
			for i := range max(len(entries), len(want)) {
				if i >= len(entries) || i >= len(want) || entries[i] != want[i] {
					t.Errorf("entry %d = %+v, want %+v", i, entryAt(entries, i), entryAt(want, i))
				}
			}
		}
		if branch != (GitBranch{Name: "main"}) {
			t.Errorf("branch = %+v", branch)
		}
		wantFiles := map[string]byte{
			"web/a.ts": 'M', "b.go": 'M', "c.go": 'M', "d.sh": 'M', "e.sh": 'M', "new.go": 'A', "new2.go": 'A', "intent.go": 'M',
			"web/to.ts": 'R', "web/to2.ts": 'R', "copy.go": 'A', "gone.go": 'D', "gone2.go": 'D', "addgone.go": 'A',
			"both.go": '!', "add.go": '!', "del.go": '!', "au.go": '!', "ua.go": '!', "du.go": '!', "ud.go": '!',
			"notes.txt": 'U', "tmp/": 'U',
		}
		if !reflect.DeepEqual(marks.Files, wantFiles) {
			t.Errorf("markers = %q,\nwant %q", marks.Files, wantFiles)
		}
		if want := map[string]bool{"build/": true, "app.log": true}; !reflect.DeepEqual(marks.Ignored, want) {
			t.Errorf("Ignored = %v", marks.Ignored)
		}
		for _, e := range entries {
			if e.X == '!' || strings.HasPrefix(e.Path, "build") || e.Path == "app.log" {
				t.Errorf("an ignored path is an entry: %+v", e)
			}
		}
	})

	// Recorded: a rename with changes and a copy (status.renames=copies), two
	// files, an untracked folder with spaces, a Unicode name, two ignored,
	// on a branch with its upstream.
	t.Run("recorded renames, spaces and Unicode", func(t *testing.T) {
		out := "## main...origin/main\x00C  a-copy.txt\x00a.txt\x00M  a.txt\x00R  sub/t.txt\x00sub/s.txt\x00" +
			"?? .gitignore\x00?? dir with space/\x00?? é à.txt\x00!! build/\x00!! x.log\x00"
		entries, branch, marks := ParseSnapshot(out, "")
		want := []GitEntry{
			ent("", 'C', ' ', "a-copy.txt", "a.txt"), ent("", 'M', ' ', "a.txt", ""),
			ent("", 'R', ' ', "sub/t.txt", "sub/s.txt"),
			ent("", '?', '?', ".gitignore", ""), ent("", '?', '?', "dir with space/", ""), ent("", '?', '?', "é à.txt", ""),
		}
		if !reflect.DeepEqual(entries, want) {
			t.Errorf("entries = %+v,\nwant %+v", entries, want)
		}
		if branch != (GitBranch{Name: "main", Upstream: "origin/main"}) {
			t.Errorf("branch = %+v", branch)
		}
		if len(marks.Files) != 6 || marks.Files["é à.txt"] != 'U' || marks.Files["sub/t.txt"] != 'R' || len(marks.Ignored) != 2 {
			t.Errorf("markers %q, ignored %v", marks.Files, marks.Ignored)
		}
	})

	t.Run("the four branch forms and their variants", func(t *testing.T) {
		for _, c := range []struct {
			record string
			want   GitBranch
		}{
			{"## main", GitBranch{Name: "main"}},
			{"## feature/login-2", GitBranch{Name: "feature/login-2"}},
			{"## main...origin/main", GitBranch{Name: "main", Upstream: "origin/main"}},
			{"## main...origin/main [ahead 1]", GitBranch{Name: "main", Upstream: "origin/main", Ahead: 1}},
			{"## main...origin/main [behind 2]", GitBranch{Name: "main", Upstream: "origin/main", Behind: 2}},
			{"## main...origin/main [ahead 1, behind 2]", GitBranch{Name: "main", Upstream: "origin/main", Ahead: 1, Behind: 2}},
			{"## feature/x...fork/feature/x [ahead 12, behind 30]", GitBranch{Name: "feature/x", Upstream: "fork/feature/x", Ahead: 12, Behind: 30}},
			{"## main...origin/main [gone]", GitBranch{Name: "main", Upstream: "origin/main", Gone: true}},
			{"## HEAD (no branch)", GitBranch{Name: "HEAD", Detached: true}},
			{"## No commits yet on main", GitBranch{Name: "main", NoCommits: true}},
			{"## No commits yet on main...origin/main", GitBranch{Name: "main", NoCommits: true, Upstream: "origin/main"}},
			{"## No commits yet on main...origin/main [gone]", GitBranch{Name: "main", NoCommits: true, Upstream: "origin/main", Gone: true}},
			{"## Initial commit on master", GitBranch{Name: "master", NoCommits: true}},
		} {
			// The branch record comes first, with an entry after it, and it
			// is never a marker or an entry.
			entries, branch, marks := ParseSnapshot(recs(c.record, " M x.go"), "")
			if branch != c.want {
				t.Errorf("%q: branch = %+v, want %+v", c.record, branch, c.want)
			}
			if len(entries) != 1 || entries[0] != ent("", ' ', 'M', "x.go", "") || len(marks.Files) != 1 {
				t.Errorf("%q: entries %+v, markers %v", c.record, entries, marks.Files)
			}
			if entries, branch, marks := ParseSnapshot(recs(c.record), ""); len(entries) != 0 || len(marks.Files) != 0 || branch != c.want {
				t.Errorf("%q alone: entries %+v, markers %v, branch %+v", c.record, entries, marks.Files, branch)
			}
		}
		if _, branch, _ := ParseSnapshot("", ""); branch != (GitBranch{}) {
			t.Errorf("no output: branch %+v", branch)
		}
	})

	t.Run("a project in a folder of the repository", func(t *testing.T) {
		out := recs("## main", " M Sub/a.go", "?? sub/n/", "!! other/x", "R  sub/to.go", "sub/from.go", " M sub/", "A  elsewhere/z.go", "?? sub/dir with space/f.txt")
		entries, _, marks := ParseSnapshot(out, "sub/")
		want := []GitEntry{
			{X: ' ', Y: 'M', Path: "a.go", RepoPath: "Sub/a.go"},
			{X: '?', Y: '?', Path: "n/", RepoPath: "sub/n/"},
			{X: 'R', Y: ' ', Path: "to.go", Orig: "from.go", RepoPath: "sub/to.go", RepoOrig: "sub/from.go"},
			{X: '?', Y: '?', Path: "dir with space/f.txt", RepoPath: "sub/dir with space/f.txt"},
		}
		if !reflect.DeepEqual(entries, want) {
			t.Errorf("entries = %+v,\nwant %+v", entries, want)
		}
		if want := map[string]byte{"a.go": 'M', "n/": 'U', "to.go": 'R', "dir with space/f.txt": 'U'}; !reflect.DeepEqual(marks.Files, want) || len(marks.Ignored) != 0 {
			t.Errorf("markers %q, ignored %v", marks.Files, marks.Ignored)
		}
	})

	// Recorded: git lists the ignored files of an untracked folder apart, and
	// IsIgnored finds a path by itself or by a folder above it.
	t.Run("ignored inside an untracked folder", func(t *testing.T) {
		out := "## main\x00?? dir/\x00!! dir/build/\x00!! dir/sub/y.log\x00!! dir/x.log\x00!! onlyign/\x00"
		entries, _, marks := ParseSnapshot(out, "")
		if want := []GitEntry{ent("", '?', '?', "dir/", "")}; !reflect.DeepEqual(entries, want) {
			t.Errorf("entries = %+v", entries)
		}
		for p, want := range map[string]bool{
			"dir/build/o.bin": true, "dir/build/deep/x": true, "dir/build": true, "dir/x.log": true, "dir/sub/y.log": true,
			"onlyign/z.log": true, "dir/a.txt": false, "dir/sub/s.txt": false, "dir/x.logger": false, "dir/buildx/o": false,
			"dir": false, "dir/": false, "": false, "other/build/o": false,
		} {
			if got := marks.IsIgnored(p); got != want {
				t.Errorf("IsIgnored(%q) = %v, want %v", p, got, want)
			}
		}
		if (GitMarks{}).IsIgnored("x") {
			t.Errorf("IsIgnored on no marks")
		}
	})

	t.Run("odd input", func(t *testing.T) {
		for _, out := range []string{"", "\x00", "\x00\x00", "M", "M x", "?? ", "garbage\x00more"} {
			if entries, _, marks := ParseSnapshot(out, ""); len(entries) != 0 || len(marks.Files) != 0 {
				t.Errorf("%q: %+v %v", out, entries, marks.Files)
			}
		}
		// A rename cut short: the entry stands, without a source.
		if entries, _, _ := ParseSnapshot("R  new.go", ""); len(entries) != 1 || entries[0].Orig != "" {
			t.Errorf("a rename without its source: %+v", entries)
		}
		// A path starting with "## " is still a path: its record starts with XY.
		if entries, _, _ := ParseSnapshot(recs("?? ## odd.txt"), ""); len(entries) != 1 || entries[0].Path != "## odd.txt" {
			t.Errorf("a name starting with ##: %+v", entries)
		}
	})
}

func entryAt(es []GitEntry, i int) any {
	if i < len(es) {
		return es[i]
	}
	return "<none>"
}

// TestConflictName covers the seven names of D6 and the rest.
func TestConflictName(t *testing.T) {
	for xy, want := range map[string]string{
		"UU": "both modified", "AU": "added by us", "UA": "added by them", "DU": "deleted by us",
		"UD": "deleted by them", "AA": "both added", "DD": "both deleted",
		" M": "", "M ": "", "MM": "", "??": "", "A ": "", "R ": "", " D": "", "AM": "", "!!": "", "UM": "", "DA": "",
	} {
		if got := ConflictName(xy[0], xy[1]); got != want {
			t.Errorf("ConflictName(%q) = %q, want %q", xy, got, want)
		}
	}
}

// srcRepo is an empty repository on branch main, with the identity and the
// line end setting of the tests local to it.
func srcRepo(t *testing.T) string {
	t.Helper()
	gitEnv(t)
	dir := filepath.Join(t.TempDir(), "proj")
	if err := os.Mkdir(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	gitT(t, dir, "init", "-q", "-b", "main")
	gitT(t, dir, "config", "user.name", "test")
	gitT(t, dir, "config", "user.email", "test@example.invalid")
	gitT(t, dir, "config", "core.autocrlf", "false")
	return dir
}

const (
	renOld = "f1\nf2\nf3\nf4\nf5\nf6\nf7\nf8\n"
	renNew = "f1\nf2\nf3\nF4\nf5\nf6\nf7\nf8\n"
	cpSrc  = "g1\ng2\ng3\ng4\ng5\ng6\ng7\ng8\n"
	lfsOld = "version https://git-lfs.github.com/spec/v1\noid sha256:1111111111111111111111111111111111111111111111111111111111111111\nsize 100\n"
	lfsNew = "version https://git-lfs.github.com/spec/v1\noid sha256:2222222222222222222222222222222222222222222222222222222222222222\nsize 200\n"
)

// statesRepo makes a repository with a change of each kind and returns its
// path: see the table of TestSourceStates.
func statesRepo(t *testing.T) string {
	t.Helper()
	repo := srcRepo(t)
	put := func(rel, data string) { writeAt(t, repo, rel, data) }
	put(".gitignore", "*.log\nbuild/\n")
	put("a-mod.txt", "a1\na2\na3\n")
	put("b-staged.txt", "b1\nb2\n")
	put("c-both.txt", "c1\nc2\nc3\n")
	put("d-del.txt", "d\n")
	put("e-delstaged.txt", "e\n")
	put("f-ren-old.txt", renOld)
	put("g-copy-src.txt", cpSrc)
	put("sub/h-nested.txt", "h\n")
	put("sp ace.txt", "s\n")
	put("é-ünï.txt", "u\n")
	put("t-type.txt", "t\n")
	put("crlf.txt", "x\ny\n")
	put("nonl.txt", "n1\nn2\n")
	put("bin.dat", "\x00\x01\x02bin")
	put("lfs.bin", lfsOld)
	put(".env", "SECRET=1\n")
	put("big.txt", "0123456789\nabcdefghij\n")
	gitT(t, repo, "add", "-A")
	gitT(t, repo, "commit", "-q", "-m", "base")
	gitT(t, repo, "config", "status.renames", "copies")

	// The index's versions.
	put("b-staged.txt", "b1\nB2\n")
	put("c-both.txt", "c1\nC2\nc3\n")
	gitT(t, repo, "mv", "f-ren-old.txt", "f-ren-new.txt")
	put("f-ren-new.txt", renNew)
	put("g-copy-dst.txt", cpSrc)
	put("g-copy-src.txt", cpSrc+"g9\n")
	put("new-staged.txt", "n\n")
	put("new-am.txt", "p1\n")
	gitT(t, repo, "rm", "-q", "e-delstaged.txt")
	gitT(t, repo, "add", "b-staged.txt", "c-both.txt", "f-ren-new.txt", "g-copy-dst.txt", "g-copy-src.txt", "new-staged.txt", "new-am.txt")
	// A type change in the index: the file becomes a link to "t-type-target".
	target := filepath.Join(t.TempDir(), "target")
	write(t, target, "t-type-target")
	blob := gitT(t, repo, "hash-object", "-w", target)
	gitT(t, repo, "update-index", "--cacheinfo", "120000,"+blob+",t-type.txt")

	// The working tree's.
	put("c-both.txt", "c1\nC2\nC3\n")
	put("new-am.txt", "p1\np2\n")
	put("a-mod.txt", "a1\nA2\na3\n")
	if err := os.Remove(filepath.Join(repo, "d-del.txt")); err != nil {
		t.Fatal(err)
	}
	put("sub/h-nested.txt", "h2\n")
	put("sp ace.txt", "s2\n")
	put("é-ünï.txt", "u2\n")
	put("crlf.txt", "x\r\ny\r\n")
	put("nonl.txt", "n1\nn2")
	put("bin.dat", "\x00\x01\x02BIN!")
	put("lfs.bin", lfsNew)
	put(".env", "SECRET=2\n")
	put("big.txt", "0123456789\nABCDEFGHIJ\n")
	put("untracked.txt", "u1\nu2\n")
	put("fresh/inner.txt", "i\n")
	put("fresh/z.log", "ignored inside an untracked folder\n")
	put("x.log", "l\n")
	put("build/o.bin", "o\n")
	return repo
}

// fileState is what the read-only check compares.
type fileState struct {
	size int64
	mod  time.Time
	sum  [sha256.Size]byte
}

// treeState is every file under root, .git included, with its size, time
// and the hash of its bytes.
func treeState(t *testing.T, root string) map[string]fileState {
	t.Helper()
	out := map[string]fileState{}
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, p)
		out[filepath.ToSlash(rel)] = fileState{info.Size(), info.ModTime(), sha256.Sum256(data)}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// subcommand is the git command of a traced call: the first argument after
// "-C dir" and the "-c key=value" pairs.
func subcommand(args []string) string {
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-C", "-c":
			i++
		default:
			return args[i]
		}
	}
	return ""
}

// byPath maps a snapshot's entries to their paths (one each in these tests).
func byPath(t *testing.T, entries []GitEntry) map[string]GitEntry {
	t.Helper()
	m := map[string]GitEntry{}
	for _, e := range entries {
		if _, dup := m[e.Path]; dup {
			t.Fatalf("two entries for %s", e.Path)
		}
		m[e.Path] = e
	}
	return m
}

// TestSourceStates covers plan 44's criteria 44.3, 44.6 and 44.10 on one
// temporary repository holding a change of each kind: the snapshot git
// gives, the two sides of each file's diff with their exact content and
// flags, and the read-only guarantee over all of it.
func TestSourceStates(t *testing.T) {
	repo := statesRepo(t)
	// A tracked file whose time moved without a change: git would refresh
	// the index, if it were allowed to write.
	later := time.Now().Add(time.Hour)
	if err := os.Chtimes(filepath.Join(repo, ".gitignore"), later, later); err != nil {
		t.Fatal(err)
	}
	before := treeState(t, repo)

	var calls [][]string
	g := GitCLI{Trace: func(a []string) { calls = append(calls, a) }}
	snap, err := g.Snapshot(repo)
	if err != nil {
		t.Fatal(err)
	}
	if !samePath(snap.Top, repo) || snap.Prefix != "" || strings.Contains(snap.Top, "/") && os.PathSeparator == '\\' {
		t.Errorf("Top %q, Prefix %q for %q", snap.Top, snap.Prefix, repo)
	}
	if snap.Branch != (GitBranch{Name: "main"}) {
		t.Errorf("Branch = %+v", snap.Branch)
	}

	t.Run("the entries", func(t *testing.T) {
		want := []GitEntry{
			ent("", ' ', 'M', ".env", ""),
			ent("", ' ', 'M', "a-mod.txt", ""),
			ent("", 'M', ' ', "b-staged.txt", ""),
			ent("", ' ', 'M', "big.txt", ""),
			ent("", ' ', 'M', "bin.dat", ""),
			ent("", 'M', 'M', "c-both.txt", ""),
			ent("", ' ', 'M', "crlf.txt", ""),
			ent("", ' ', 'D', "d-del.txt", ""),
			ent("", 'D', ' ', "e-delstaged.txt", ""),
			ent("", 'R', ' ', "f-ren-new.txt", "f-ren-old.txt"),
			ent("", 'C', ' ', "g-copy-dst.txt", "g-copy-src.txt"),
			ent("", 'M', ' ', "g-copy-src.txt", ""),
			ent("", ' ', 'M', "lfs.bin", ""),
			ent("", 'A', 'M', "new-am.txt", ""),
			ent("", 'A', ' ', "new-staged.txt", ""),
			ent("", ' ', 'M', "nonl.txt", ""),
			ent("", ' ', 'M', "sp ace.txt", ""),
			ent("", ' ', 'M', "sub/h-nested.txt", ""),
			ent("", 'T', 'M', "t-type.txt", ""),
			ent("", ' ', 'M', "é-ünï.txt", ""),
			ent("", '?', '?', "fresh/", ""),
			ent("", '?', '?', "untracked.txt", ""),
		}
		got := append([]GitEntry(nil), snap.Entries...)
		// The staged entries' blobs (plan 45, design D14): the index's, "-"
		// for a staged deletion, none for an entry not staged.
		for i := range got {
			e := &got[i]
			want := ""
			switch {
			case e.X == 'D':
				want = "-"
			case isStaged(*e):
				want = gitT(t, repo, "rev-parse", ":0:"+e.RepoPath)
			}
			if e.Index != want {
				t.Errorf("%s: blob %q, want %q", e.Path, e.Index, want)
			}
			e.Index = ""
		}
		// The type change's Y depends on whether the system makes links.
		for i := range got {
			if got[i].Path == "t-type.txt" {
				if got[i].X != 'T' {
					t.Errorf("t-type.txt: X = %q", got[i].X)
				}
				got[i].Y = 'M'
			}
		}
		if !reflect.DeepEqual(got, want) {
			for i := range max(len(got), len(want)) {
				if i >= len(got) || i >= len(want) || got[i] != want[i] {
					t.Errorf("entry %d = %+v, want %+v", i, entryAt(got, i), entryAt(want, i))
				}
			}
		}
	})

	t.Run("the Explorer's markers are the same read", func(t *testing.T) {
		marks, err := g.Status(repo)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(marks, snap.Marks) {
			t.Errorf("Status = %+v,\nSnapshot.Marks = %+v", marks, snap.Marks)
		}
		wantFiles := map[string]byte{
			".env": 'M', "a-mod.txt": 'M', "b-staged.txt": 'M', "big.txt": 'M', "bin.dat": 'M', "c-both.txt": 'M',
			"crlf.txt": 'M', "d-del.txt": 'D', "e-delstaged.txt": 'D', "f-ren-new.txt": 'R', "g-copy-dst.txt": 'A',
			"g-copy-src.txt": 'M', "lfs.bin": 'M', "new-am.txt": 'A', "new-staged.txt": 'A', "nonl.txt": 'M',
			"sp ace.txt": 'M', "sub/h-nested.txt": 'M', "t-type.txt": 'M', "é-ünï.txt": 'M',
			"fresh/": 'U', "untracked.txt": 'U',
		}
		if !reflect.DeepEqual(snap.Marks.Files, wantFiles) {
			t.Errorf("markers = %q,\nwant %q", snap.Marks.Files, wantFiles)
		}
		if want := map[string]bool{"build/": true, "x.log": true, "fresh/z.log": true}; !reflect.DeepEqual(snap.Marks.Ignored, want) {
			t.Errorf("Ignored = %v", snap.Marks.Ignored)
		}
		// The files of an untracked folder, less those git ignores.
		files, truncated, err := UntrackedFiles(repo, "fresh/", 100)
		if err != nil || truncated || !reflect.DeepEqual(files, []string{"fresh/inner.txt", "fresh/z.log"}) {
			t.Fatalf("UntrackedFiles = %q, %v, %v", files, truncated, err)
		}
		var kept []string
		for _, f := range files {
			if !snap.Marks.IsIgnored(f) {
				kept = append(kept, f)
			}
		}
		if !reflect.DeepEqual(kept, []string{"fresh/inner.txt"}) {
			t.Errorf("not ignored: %q", kept)
		}
	})

	entries := byPath(t, snap.Entries)
	id := func(spec string) string { return gitT(t, repo, "rev-parse", spec) }
	// side checks a side whole, its modification time apart: the working
	// file's, set unless the file is missing.
	side := func(t *testing.T, got DiffSide, want DiffSide) {
		t.Helper()
		if wantMod := want.Name == "working tree" && !want.Missing; got.ModTime.IsZero() == wantMod {
			t.Errorf("%s side: ModTime %v, want one: %v", got.Name, got.ModTime, wantMod)
		}
		got.ModTime = time.Time{}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("side = %+v,\nwant   %+v", got, want)
		}
	}
	sides := func(t *testing.T, path string, staged bool) DiffPair {
		t.Helper()
		e, ok := entries[path]
		if !ok {
			t.Fatalf("no entry for %s", path)
		}
		p, err := g.Sides(repo, snap, e, staged)
		if err != nil {
			t.Fatalf("Sides(%s, staged %v): %v", path, staged, err)
		}
		return p
	}

	t.Run("unstaged: the index against the working file", func(t *testing.T) {
		p := sides(t, "a-mod.txt", false)
		side(t, p.Old, DiffSide{Name: "index", Path: "a-mod.txt", ID: id(":a-mod.txt"), Size: 9, Data: []byte("a1\na2\na3\n")})
		side(t, p.New, DiffSide{Name: "working tree", Path: "a-mod.txt", Size: 9, Data: []byte("a1\nA2\na3\n")})
		if p.Conflict != "" {
			t.Errorf("Conflict = %q", p.Conflict)
		}
		res := diff.Compare(p.Old.Data, p.New.Data, 3)
		if len(res.Hunks) != 1 || res.Added != 1 || res.Removed != 1 {
			t.Errorf("the comparison: %+v", res)
		}
	})

	t.Run("staged: HEAD against the index", func(t *testing.T) {
		p := sides(t, "b-staged.txt", true)
		side(t, p.Old, DiffSide{Name: "HEAD", Path: "b-staged.txt", ID: id("HEAD:b-staged.txt"), Size: 6, Data: []byte("b1\nb2\n")})
		side(t, p.New, DiffSide{Name: "index", Path: "b-staged.txt", ID: id(":b-staged.txt"), Size: 6, Data: []byte("b1\nB2\n")})
	})

	t.Run("a file both staged and unstaged: two different pairs", func(t *testing.T) {
		staged := sides(t, "c-both.txt", true)
		side(t, staged.Old, DiffSide{Name: "HEAD", Path: "c-both.txt", ID: id("HEAD:c-both.txt"), Size: 9, Data: []byte("c1\nc2\nc3\n")})
		side(t, staged.New, DiffSide{Name: "index", Path: "c-both.txt", ID: id(":c-both.txt"), Size: 9, Data: []byte("c1\nC2\nc3\n")})
		unstaged := sides(t, "c-both.txt", false)
		side(t, unstaged.Old, DiffSide{Name: "index", Path: "c-both.txt", ID: id(":c-both.txt"), Size: 9, Data: []byte("c1\nC2\nc3\n")})
		side(t, unstaged.New, DiffSide{Name: "working tree", Path: "c-both.txt", Size: 9, Data: []byte("c1\nC2\nC3\n")})
		if bytes.Equal(staged.Old.Data, unstaged.Old.Data) || staged.Old.ID == unstaged.Old.ID {
			t.Errorf("the two comparisons start from the same side")
		}
	})

	t.Run("a staged add and an added file changed again", func(t *testing.T) {
		p := sides(t, "new-staged.txt", true)
		side(t, p.Old, DiffSide{Name: "HEAD", Path: "new-staged.txt", Missing: true})
		side(t, p.New, DiffSide{Name: "index", Path: "new-staged.txt", ID: id(":new-staged.txt"), Size: 2, Data: []byte("n\n")})
		p = sides(t, "new-am.txt", true)
		side(t, p.Old, DiffSide{Name: "HEAD", Path: "new-am.txt", Missing: true})
		side(t, p.New, DiffSide{Name: "index", Path: "new-am.txt", ID: id(":new-am.txt"), Size: 3, Data: []byte("p1\n")})
		p = sides(t, "new-am.txt", false)
		side(t, p.Old, DiffSide{Name: "index", Path: "new-am.txt", ID: id(":new-am.txt"), Size: 3, Data: []byte("p1\n")})
		side(t, p.New, DiffSide{Name: "working tree", Path: "new-am.txt", Size: 6, Data: []byte("p1\np2\n")})
	})

	t.Run("untracked: nothing against the working file", func(t *testing.T) {
		p := sides(t, "untracked.txt", false)
		side(t, p.Old, DiffSide{Path: "untracked.txt", Missing: true})
		side(t, p.New, DiffSide{Name: "working tree", Path: "untracked.txt", Size: 6, Data: []byte("u1\nu2\n")})
		// A file of an untracked folder, as the panel builds its entry.
		e := GitEntry{X: '?', Y: '?', Path: "fresh/inner.txt", RepoPath: "fresh/inner.txt"}
		in, err := g.Sides(repo, snap, e, false)
		if err != nil {
			t.Fatal(err)
		}
		side(t, in.New, DiffSide{Name: "working tree", Path: "fresh/inner.txt", Size: 2, Data: []byte("i\n")})
		if !in.Old.Missing {
			t.Errorf("old side of an untracked file: %+v", in.Old)
		}
	})

	t.Run("deleted unstaged and staged", func(t *testing.T) {
		p := sides(t, "d-del.txt", false)
		side(t, p.Old, DiffSide{Name: "index", Path: "d-del.txt", ID: id(":d-del.txt"), Size: 2, Data: []byte("d\n")})
		side(t, p.New, DiffSide{Name: "working tree", Path: "d-del.txt", Missing: true})
		p = sides(t, "e-delstaged.txt", true)
		side(t, p.Old, DiffSide{Name: "HEAD", Path: "e-delstaged.txt", ID: id("HEAD:e-delstaged.txt"), Size: 2, Data: []byte("e\n")})
		side(t, p.New, DiffSide{Name: "index", Path: "e-delstaged.txt", Missing: true})
	})

	t.Run("renamed and changed: HEAD at the old path against the index", func(t *testing.T) {
		p := sides(t, "f-ren-new.txt", true)
		side(t, p.Old, DiffSide{Name: "HEAD", Path: "f-ren-old.txt", ID: id("HEAD:f-ren-old.txt"), Size: int64(len(renOld)), Data: []byte(renOld)})
		side(t, p.New, DiffSide{Name: "index", Path: "f-ren-new.txt", ID: id(":f-ren-new.txt"), Size: int64(len(renNew)), Data: []byte(renNew)})
		res := diff.Compare(p.Old.Data, p.New.Data, 3)
		if res.Added != 1 || res.Removed != 1 {
			t.Errorf("the rename's change: %+v", res)
		}
		// A copy reads its source in HEAD.
		p = sides(t, "g-copy-dst.txt", true)
		side(t, p.Old, DiffSide{Name: "HEAD", Path: "g-copy-src.txt", ID: id("HEAD:g-copy-src.txt"), Size: int64(len(cpSrc)), Data: []byte(cpSrc)})
		side(t, p.New, DiffSide{Name: "index", Path: "g-copy-dst.txt", ID: id(":g-copy-dst.txt"), Size: int64(len(cpSrc)), Data: []byte(cpSrc)})
	})

	t.Run("names with spaces, Unicode and folders", func(t *testing.T) {
		for _, c := range []struct{ path, old, cur string }{
			{"sp ace.txt", "s\n", "s2\n"}, {"é-ünï.txt", "u\n", "u2\n"}, {"sub/h-nested.txt", "h\n", "h2\n"},
		} {
			p := sides(t, c.path, false)
			side(t, p.Old, DiffSide{Name: "index", Path: c.path, ID: id(":" + c.path), Size: int64(len(c.old)), Data: []byte(c.old)})
			side(t, p.New, DiffSide{Name: "working tree", Path: c.path, Size: int64(len(c.cur)), Data: []byte(c.cur)})
		}
	})

	t.Run("a type change in the index", func(t *testing.T) {
		p := sides(t, "t-type.txt", true)
		side(t, p.Old, DiffSide{Name: "HEAD", Path: "t-type.txt", ID: id("HEAD:t-type.txt"), Size: 2, Data: []byte("t\n")})
		side(t, p.New, DiffSide{Name: "index", Path: "t-type.txt", ID: id(":t-type.txt"), Size: 13, Data: []byte("t-type-target")})
	})

	t.Run("line endings only", func(t *testing.T) {
		p := sides(t, "crlf.txt", false)
		side(t, p.Old, DiffSide{Name: "index", Path: "crlf.txt", ID: id(":crlf.txt"), Size: 4, Data: []byte("x\ny\n")})
		side(t, p.New, DiffSide{Name: "working tree", Path: "crlf.txt", Size: 6, Data: []byte("x\r\ny\r\n")})
		res := diff.Compare(p.Old.Data, p.New.Data, 3)
		if !res.EOLOnly || len(res.Hunks) != 0 || res.NoNewlineOld || res.NoNewlineNew {
			t.Errorf("the comparison: %+v", res)
		}
	})

	t.Run("no newline at the end", func(t *testing.T) {
		p := sides(t, "nonl.txt", false)
		side(t, p.Old, DiffSide{Name: "index", Path: "nonl.txt", ID: id(":nonl.txt"), Size: 6, Data: []byte("n1\nn2\n")})
		side(t, p.New, DiffSide{Name: "working tree", Path: "nonl.txt", Size: 5, Data: []byte("n1\nn2")})
		res := diff.Compare(p.Old.Data, p.New.Data, 3)
		if res.NoNewlineOld || !res.NoNewlineNew || res.Added != 1 || res.Removed != 1 || res.EOLOnly {
			t.Errorf("the comparison: %+v", res)
		}
	})

	t.Run("binary, with both sizes and no lines", func(t *testing.T) {
		p := sides(t, "bin.dat", false)
		side(t, p.Old, DiffSide{Name: "index", Path: "bin.dat", ID: id(":bin.dat"), Size: 6, Binary: true})
		side(t, p.New, DiffSide{Name: "working tree", Path: "bin.dat", Size: 7, Binary: true})
	})

	t.Run("an LFS pointer", func(t *testing.T) {
		p := sides(t, "lfs.bin", false)
		side(t, p.Old, DiffSide{Name: "index", Path: "lfs.bin", ID: id(":lfs.bin"), Size: int64(len(lfsOld)), LFS: true})
		side(t, p.New, DiffSide{Name: "working tree", Path: "lfs.bin", Size: int64(len(lfsNew)), LFS: true})
	})

	t.Run("too large, at the limit and over", func(t *testing.T) {
		defer func(n int64) { MaxFileSize = n }(MaxFileSize)
		MaxFileSize = 22 // big.txt is 22 bytes on both sides
		p := sides(t, "big.txt", false)
		side(t, p.Old, DiffSide{Name: "index", Path: "big.txt", ID: id(":big.txt"), Size: 22, Data: []byte("0123456789\nabcdefghij\n")})
		side(t, p.New, DiffSide{Name: "working tree", Path: "big.txt", Size: 22, Data: []byte("0123456789\nABCDEFGHIJ\n")})
		MaxFileSize = 21
		p = sides(t, "big.txt", false)
		side(t, p.Old, DiffSide{Name: "index", Path: "big.txt", ID: id(":big.txt"), Size: 22, TooLarge: true})
		side(t, p.New, DiffSide{Name: "working tree", Path: "big.txt", Size: 22, TooLarge: true})
		// Others stay small enough.
		p = sides(t, "sp ace.txt", false)
		if p.Old.TooLarge || p.New.TooLarge || p.New.Data == nil {
			t.Errorf("a small file: %+v", p)
		}
		// The blob alone, staged.
		MaxFileSize = 3
		p = sides(t, "b-staged.txt", true)
		if !p.Old.TooLarge || !p.New.TooLarge || p.Old.Data != nil || p.Old.Size != 6 {
			t.Errorf("staged: %+v", p)
		}
	})

	t.Run("a credential file is refused before anything is read", func(t *testing.T) {
		n := len(calls)
		for _, staged := range []bool{false, true} {
			p, err := g.Sides(repo, snap, entries[".env"], staged)
			if !errors.Is(err, ErrCredential) || p.Old.Data != nil || p.New.Data != nil {
				t.Errorf("staged %v: %v, %+v", staged, err, p)
			}
		}
		renamed := ent("", 'R', ' ', "safe.txt", ".env")
		if _, err := g.Sides(repo, snap, renamed, true); !errors.Is(err, ErrCredential) {
			t.Errorf("a rename from a credential file: %v", err)
		}
		if len(calls) != n {
			t.Errorf("git was called for a credential file: %v", calls[n:])
		}
	})

	t.Run("refusals", func(t *testing.T) {
		n := len(calls)
		for _, c := range []struct {
			e    GitEntry
			dir  string
			want string
		}{
			{ent("", '?', '?', "fresh/", ""), repo, "a folder, not a file"},
			{GitEntry{X: ' ', Y: 'M'}, repo, "a folder, not a file"},
			{GitEntry{X: ' ', Y: 'M', Path: "../x.txt", RepoPath: "../x.txt"}, repo, "outside the repository"},
			{GitEntry{X: ' ', Y: 'M', Path: "a/../../x", RepoPath: "a/../../x"}, repo, "outside the repository"},
			{GitEntry{X: ' ', Y: 'M', Path: "/etc/x", RepoPath: "/etc/x"}, repo, "outside the repository"},
			// The repository's file is outside a project that is another folder.
			{entries["a-mod.txt"], filepath.Join(repo, "sub"), "outside the project: a-mod.txt"},
		} {
			p, err := g.Sides(c.dir, snap, c.e, false)
			if err == nil || !strings.Contains(err.Error(), c.want) || p.New.Data != nil {
				t.Errorf("%+v in %s: %v, %+v; want %q", c.e, c.dir, err, p, c.want)
			}
		}
		if _, err := g.Sides(repo, GitSnapshot{}, entries["a-mod.txt"], false); err == nil {
			t.Errorf("a snapshot with no top level")
		}
		if len(calls) < n {
			t.Fatal("calls lost")
		}
	})

	t.Run("a project in a folder of the repository", func(t *testing.T) {
		sub := filepath.Join(repo, "sub")
		ss, err := g.Snapshot(sub)
		if err != nil {
			t.Fatal(err)
		}
		if !samePath(ss.Top, repo) || ss.Prefix != "sub/" || ss.Branch.Name != "main" {
			t.Errorf("Top %q, Prefix %q, Branch %+v", ss.Top, ss.Prefix, ss.Branch)
		}
		if want := []GitEntry{{X: ' ', Y: 'M', Path: "h-nested.txt", RepoPath: "sub/h-nested.txt"}}; !reflect.DeepEqual(ss.Entries, want) {
			t.Fatalf("entries = %+v, want %+v", ss.Entries, want)
		}
		if want := map[string]byte{"h-nested.txt": 'M'}; !reflect.DeepEqual(ss.Marks.Files, want) {
			t.Errorf("markers = %v", ss.Marks.Files)
		}
		p, err := g.Sides(sub, ss, ss.Entries[0], false)
		if err != nil {
			t.Fatal(err)
		}
		side(t, p.Old, DiffSide{Name: "index", Path: "sub/h-nested.txt", ID: id(":sub/h-nested.txt"), Size: 2, Data: []byte("h\n")})
		side(t, p.New, DiffSide{Name: "working tree", Path: "sub/h-nested.txt", Size: 3, Data: []byte("h2\n")})
	})

	t.Run("read only: the calls, the index and the working tree", func(t *testing.T) {
		if len(calls) == 0 {
			t.Fatal("no call traced")
		}
		seen := map[string]int{}
		for _, c := range calls {
			sub := subcommand(c)
			seen[sub]++
			if sub != "rev-parse" && sub != "status" && sub != "cat-file" {
				t.Errorf("a call that may write: %v", c)
			}
			j := " " + strings.Join(c, " ") + " "
			if !strings.Contains(j, " -c core.fsmonitor=false ") || !strings.Contains(j, " -c core.quotepath=off ") {
				t.Errorf("without the safety settings: %v", c)
			}
			for _, a := range c {
				if a == "--force" || a == "-f" || a == "--textconv" || a == "--filters" || a == "--no-index" {
					t.Errorf("a risky option in %v", c)
				}
			}
		}
		if seen["rev-parse"] == 0 || seen["status"] == 0 || seen["cat-file"] == 0 {
			t.Errorf("calls seen: %v", seen)
		}
		after := treeState(t, repo)
		var diffs []string
		for p, s := range after {
			if b, ok := before[p]; !ok || b.size != s.size || !b.mod.Equal(s.mod) || b.sum != s.sum {
				diffs = append(diffs, p)
			}
		}
		for p := range before {
			if _, ok := after[p]; !ok {
				diffs = append(diffs, p+" (gone)")
			}
		}
		if sort.Strings(diffs); len(diffs) > 0 {
			t.Errorf("the repository changed: %v", diffs)
		}
		if before[".git/index"].size == 0 {
			t.Errorf("the index was not looked at")
		}
		if _, err := os.Stat(filepath.Join(repo, ".git", "index.lock")); err == nil {
			t.Errorf("index.lock left")
		}
	})
}

// TestSourceConflicts covers criteria 44.3 and 44.6 on a merge with the
// seven conflicts of git, made from two branches: each is a conflict entry,
// named, with "ours" (stage 2, else HEAD) against the working file.
func TestSourceConflicts(t *testing.T) {
	repo := srcRepo(t)
	ten := "1\n2\n3\n4\n5\n6\n7\n8\n9\n10\n"
	with3 := func(v string) string { return strings.Replace(ten, "3\n", v+"\n", 1) }
	for _, f := range []string{"uu.txt", "du.txt", "ud.txt", "dd.txt"} {
		writeAt(t, repo, f, ten)
	}
	writeAt(t, repo, "plain.txt", "plain\n")
	gitT(t, repo, "add", "-A")
	gitT(t, repo, "commit", "-q", "-m", "base")
	gitT(t, repo, "checkout", "-q", "-b", "other")
	writeAt(t, repo, "uu.txt", with3("theirs3"))
	writeAt(t, repo, "du.txt", with3("theirs3"))
	writeAt(t, repo, "aa.txt", "theirs\n")
	gitT(t, repo, "rm", "-q", "ud.txt")
	gitT(t, repo, "mv", "dd.txt", "dd-theirs.txt")
	gitT(t, repo, "add", "-A")
	gitT(t, repo, "commit", "-q", "-m", "theirs")
	gitT(t, repo, "checkout", "-q", "main")
	writeAt(t, repo, "uu.txt", with3("ours3"))
	writeAt(t, repo, "ud.txt", with3("ours3"))
	writeAt(t, repo, "aa.txt", "ours\n")
	gitT(t, repo, "rm", "-q", "du.txt")
	gitT(t, repo, "mv", "dd.txt", "dd-ours.txt")
	gitT(t, repo, "add", "-A")
	gitT(t, repo, "commit", "-q", "-m", "ours")
	if out, err := exec.Command("git", "-C", repo, "merge", "other").CombinedOutput(); err == nil {
		t.Fatalf("the merge did not conflict: %s", out)
	}

	g := GitCLI{}
	snap, err := g.Snapshot(repo)
	if err != nil {
		t.Fatal(err)
	}
	want := []GitEntry{
		ent("", 'A', 'A', "aa.txt", ""), ent("", 'A', 'U', "dd-ours.txt", ""), ent("", 'U', 'A', "dd-theirs.txt", ""),
		ent("", 'D', 'D', "dd.txt", ""), ent("", 'D', 'U', "du.txt", ""), ent("", 'U', 'D', "ud.txt", ""),
		ent("", 'U', 'U', "uu.txt", ""),
	}
	if !reflect.DeepEqual(snap.Entries, want) {
		t.Fatalf("entries = %+v,\nwant %+v", snap.Entries, want)
	}
	for p, m := range snap.Marks.Files {
		if m != '!' {
			t.Errorf("%s marked %q, want a conflict", p, m)
		}
	}
	if snap.Branch.Name != "main" || snap.Branch.Detached {
		t.Errorf("Branch = %+v", snap.Branch)
	}

	id := func(spec string) string { return gitT(t, repo, "rev-parse", spec) }
	markers := "1\n2\n<<<<<<< HEAD\nours3\n=======\ntheirs3\n>>>>>>> other\n4\n5\n6\n7\n8\n9\n10\n"
	oursSide := func(path, data string) DiffSide {
		return DiffSide{Name: "ours", Path: path, ID: id(":2:" + path), Size: int64(len(data)), Data: []byte(data)}
	}
	workSide := func(path, data string) DiffSide {
		return DiffSide{Name: "working tree", Path: path, Size: int64(len(data)), Data: []byte(data)}
	}
	addMarkers := "<<<<<<< HEAD\nours\n=======\ntheirs\n>>>>>>> other\n"
	for _, c := range []struct {
		path, conflict string
		old, cur       DiffSide
	}{
		{"uu.txt", "both modified", oursSide("uu.txt", with3("ours3")), workSide("uu.txt", markers)},
		{"aa.txt", "both added", oursSide("aa.txt", "ours\n"), workSide("aa.txt", addMarkers)},
		{"dd-ours.txt", "added by us", oursSide("dd-ours.txt", ten), workSide("dd-ours.txt", ten)},
		{"dd-theirs.txt", "added by them", DiffSide{Name: "HEAD", Path: "dd-theirs.txt", Missing: true}, workSide("dd-theirs.txt", ten)},
		{"dd.txt", "both deleted", DiffSide{Name: "HEAD", Path: "dd.txt", Missing: true}, DiffSide{Name: "working tree", Path: "dd.txt", Missing: true}},
		{"du.txt", "deleted by us", DiffSide{Name: "HEAD", Path: "du.txt", Missing: true}, workSide("du.txt", with3("theirs3"))},
		{"ud.txt", "deleted by them", oursSide("ud.txt", with3("ours3")), workSide("ud.txt", with3("ours3"))},
	} {
		e := byPath(t, snap.Entries)[c.path]
		for _, staged := range []bool{false, true} { // a conflict is the same whichever section opens it
			p, err := g.Sides(repo, snap, e, staged)
			if err != nil {
				t.Fatalf("%s: %v", c.path, err)
			}
			if p.Conflict != c.conflict {
				t.Errorf("%s: Conflict = %q, want %q", c.path, p.Conflict, c.conflict)
			}
			if p.New.ModTime.IsZero() != p.New.Missing {
				t.Errorf("%s: ModTime %v for Missing %v", c.path, p.New.ModTime, p.New.Missing)
			}
			p.New.ModTime = time.Time{}
			if !reflect.DeepEqual(p.Old, c.old) || !reflect.DeepEqual(p.New, c.cur) {
				t.Errorf("%s: sides\n old %+v\n new %+v\nwant\n old %+v\n new %+v", c.path, p.Old, p.New, c.old, c.cur)
			}
		}
	}

	// A resolved file, marked in the index, is no longer a conflict.
	writeAt(t, repo, "uu.txt", with3("resolved"))
	gitT(t, repo, "add", "uu.txt")
	snap, err = g.Snapshot(repo)
	if err != nil {
		t.Fatal(err)
	}
	if e := byPath(t, snap.Entries)["uu.txt"]; e.X != 'M' || e.Y != ' ' {
		t.Errorf("uu.txt resolved: %+v", e)
	}
	p, err := g.Sides(repo, snap, byPath(t, snap.Entries)["uu.txt"], true)
	if err != nil || p.Conflict != "" || string(p.New.Data) != with3("resolved") || p.Old.Name != "HEAD" {
		t.Errorf("resolved: %v, %+v", err, p)
	}
}

// TestSourceBranch covers the branch line on real repositories: no commits
// yet, a branch with its upstream, a detached HEAD with its short id.
func TestSourceBranch(t *testing.T) {
	repo := srcRepo(t)
	var calls [][]string
	g := GitCLI{Trace: func(a []string) { calls = append(calls, a) }}

	writeAt(t, repo, "first.txt", "first\n")
	snap, err := g.Snapshot(repo)
	if err != nil {
		t.Fatal(err)
	}
	if snap.Branch != (GitBranch{Name: "main", NoCommits: true}) {
		t.Errorf("no commits yet: %+v", snap.Branch)
	}
	if want := []GitEntry{ent("", '?', '?', "first.txt", "")}; !reflect.DeepEqual(snap.Entries, want) {
		t.Errorf("entries = %+v", snap.Entries)
	}
	// With HEAD unborn, a staged add has nothing in HEAD.
	gitT(t, repo, "add", "first.txt")
	snap, err = g.Snapshot(repo)
	if err != nil {
		t.Fatal(err)
	}
	p, err := g.Sides(repo, snap, snap.Entries[0], true)
	if err != nil || !p.Old.Missing || string(p.New.Data) != "first\n" {
		t.Errorf("a staged add before the first commit: %v, %+v", err, p)
	}
	gitT(t, repo, "commit", "-q", "-m", "first")
	// And an untracked or unstaged file works as ever.
	writeAt(t, repo, "first.txt", "changed\n")
	if snap, err = g.Snapshot(repo); err != nil || snap.Branch != (GitBranch{Name: "main"}) {
		t.Errorf("after the commit: %+v, %v", snap.Branch, err)
	}

	// A branch with an upstream, ahead of it.
	remote := filepath.Join(t.TempDir(), "remote.git")
	gitT(t, repo, "init", "-q", "--bare", "-b", "main", remote)
	gitT(t, repo, "remote", "add", "origin", remote)
	gitT(t, repo, "checkout", "-q", "--", "first.txt")
	gitT(t, repo, "push", "-q", "-u", "origin", "main")
	commit(t, repo, "second.txt", "second\n")
	if out := gitT(t, repo, "status", "--porcelain=v1", "--branch"); !strings.Contains(out, "[ahead 1]") {
		t.Fatalf("not ahead: %s", out)
	}
	if snap, err = g.Snapshot(repo); err != nil || snap.Branch != (GitBranch{Name: "main", Upstream: "origin/main", Ahead: 1}) {
		t.Errorf("ahead of origin/main: %+v, %v", snap.Branch, err)
	}
	for _, c := range calls[len(calls)-2:] {
		if subcommand(c) == "rev-parse" && strings.Contains(strings.Join(c, " "), "--short") {
			t.Errorf("a short id was asked for on a branch: %v", c)
		}
	}

	// A detached HEAD: the short id comes from one more rev-parse.
	gitT(t, repo, "checkout", "-q", "--detach", "HEAD~1")
	n := len(calls)
	if snap, err = g.Snapshot(repo); err != nil {
		t.Fatal(err)
	}
	want := GitBranch{Name: "HEAD", Detached: true, Head: gitT(t, repo, "rev-parse", "--short", "HEAD")}
	if snap.Branch != want || len(want.Head) < 7 {
		t.Errorf("detached: %+v, want %+v", snap.Branch, want)
	}
	if got := len(calls) - n; got != 3 || subcommand(calls[len(calls)-1]) != "rev-parse" {
		t.Errorf("a detached snapshot made %d calls: %v", got, calls[n:])
	}
}

// TestSnapshotUntrackedProject covers a project whose folder git does not
// track, inside a repository: git names the folder once, by the prefix, so
// the snapshot says it (Untracked) rather than listing no change; a tracked
// project, or one with a tracked file, does not.
func TestSnapshotUntrackedProject(t *testing.T) {
	repo := srcRepo(t)
	writeAt(t, repo, "top.txt", "top\n")
	gitT(t, repo, "add", "-A")
	gitT(t, repo, "commit", "-q", "-m", "one")
	writeAt(t, repo, "sub/a.txt", "a\n")
	writeAt(t, repo, "sub/b/c.txt", "c\n")
	sub := filepath.Join(repo, "sub")
	snap, err := GitCLI{}.Snapshot(sub)
	if err != nil || !snap.Untracked || snap.Prefix != "sub/" || len(snap.Entries) != 0 {
		t.Fatalf("an untracked project: %v, untracked %v, prefix %q, %d entries", err, snap.Untracked, snap.Prefix, len(snap.Entries))
	}
	gitT(t, repo, "add", "sub/a.txt")
	snap, err = GitCLI{}.Snapshot(sub)
	if err != nil || snap.Untracked || len(snap.Entries) != 2 {
		t.Errorf("a file of it staged: %v, untracked %v, %+v", err, snap.Untracked, snap.Entries)
	}
	if snap, err := (GitCLI{}).Snapshot(repo); err != nil || snap.Untracked {
		t.Errorf("the repository's top: %v, untracked %v", err, snap.Untracked)
	}
}

// TestSourceNoRepo covers criterion 44.11's reading side: a folder outside
// any repository is NotRepoError, git missing is ErrNoGit, and nothing else
// is run.
func TestSourceNoRepo(t *testing.T) {
	gitEnv(t)
	var calls [][]string
	g := GitCLI{Trace: func(a []string) { calls = append(calls, a) }}
	dir := t.TempDir()
	_, err := g.Snapshot(dir)
	var nr NotRepoError
	if !errors.As(err, &nr) || nr.Dir != dir {
		t.Errorf("outside a repository: %v", err)
	}
	if len(calls) != 1 || subcommand(calls[0]) != "rev-parse" {
		t.Errorf("calls: %v", calls)
	}

	t.Setenv("PATH", t.TempDir())
	if _, err := g.Snapshot(dir); !errors.Is(err, ErrNoGit) {
		t.Errorf("Snapshot without git: %v", err)
	}
	_, err = g.Sides(dir, GitSnapshot{Top: dir}, GitEntry{X: ' ', Y: 'M', Path: "x", RepoPath: "x"}, false)
	if !errors.Is(err, ErrNoGit) {
		t.Errorf("Sides without git: %v", err)
	}
}

// TestCatFileNames covers a name with a line break, sent NUL terminated, and
// the answers git gives for a name that is missing, a folder and a file.
func TestCatFileNames(t *testing.T) {
	repo := srcRepo(t)
	writeAt(t, repo, "a.txt", "alpha\n")
	writeAt(t, repo, "d/b.txt", "beta\n")
	gitT(t, repo, "add", "-A")
	gitT(t, repo, "commit", "-q", "-m", "x")
	g := GitCLI{}
	objs, err := g.catFile(repo, []string{"HEAD:a.txt", "HEAD:nope", "HEAD:d", "HEAD:d/b.txt", ":a.txt", ":2:a.txt", "HEAD:no\nsuch", "HEAD:a.txt"})
	if err != nil {
		t.Fatal(err)
	}
	if len(objs) != 8 {
		t.Fatalf("%d answers", len(objs))
	}
	for i, wantMissing := range []bool{false, true, true, false, false, true, true, false} {
		if objs[i].missing != wantMissing {
			t.Errorf("answer %d: missing = %v", i, objs[i].missing)
		}
	}
	if string(objs[0].data) != "alpha\n" || string(objs[3].data) != "beta\n" || string(objs[4].data) != "alpha\n" || string(objs[7].data) != "alpha\n" {
		t.Errorf("contents: %q %q %q %q", objs[0].data, objs[3].data, objs[4].data, objs[7].data)
	}
	if objs[0].id != gitT(t, repo, "rev-parse", "HEAD:a.txt") || objs[2].data != nil {
		t.Errorf("id %s, a folder's data %q", objs[0].id, objs[2].data)
	}
}

// TestCatFileSubmodule covers a gitlink: told as git diff tells it.
func TestCatFileSubmodule(t *testing.T) {
	repo := srcRepo(t)
	writeAt(t, repo, "a.txt", "a\n")
	gitT(t, repo, "add", "-A")
	const sha = "1111111111111111111111111111111111111111"
	gitT(t, repo, "update-index", "--add", "--cacheinfo", "160000,"+sha+",sub")
	gitT(t, repo, "commit", "-q", "-m", "link")
	g := GitCLI{}
	snap := GitSnapshot{Top: repo}
	p, err := g.Sides(repo, snap, ent("", 'M', ' ', "sub", ""), true)
	if err != nil {
		t.Fatal(err)
	}
	if string(p.Old.Data) != "Subproject commit "+sha+"\n" || p.Old.ID != sha || string(p.New.Data) != string(p.Old.Data) {
		t.Errorf("a submodule: %+v", p)
	}
}

// TestWorkingSide covers the working file's refusals, read apart from git.
func TestWorkingSide(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	writeAt(t, root, "ok.txt", "fine\n")
	writeAt(t, root, "dir/inner.txt", "inner\n")
	writeAt(t, root, "id_rsa", "KEY")
	writeAt(t, outside, "far.txt", "far\n")
	linked := os.Symlink(filepath.Join(outside, "far.txt"), filepath.Join(root, "link.txt")) == nil
	dirLinked := os.Symlink(outside, filepath.Join(root, "linkdir")) == nil
	if !linked || !dirLinked {
		t.Logf("symbolic links not made: the link cases are skipped")
	}

	s, err := workingSide(root, root, "ok.txt")
	if err != nil || string(s.Data) != "fine\n" || s.Size != 5 || s.ModTime.IsZero() || s.Name != "working tree" {
		t.Errorf("a file: %+v, %v", s, err)
	}
	if s, err = workingSide(root, root, "gone.txt"); err != nil || !s.Missing || !s.ModTime.IsZero() {
		t.Errorf("a missing file: %+v, %v", s, err)
	}
	if s, err = workingSide(root, root, "ok.txt/under"); err != nil || !s.Missing {
		t.Errorf("under a file: %+v, %v", s, err)
	}
	if _, err = workingSide(root, root, "dir"); err == nil || !strings.Contains(err.Error(), "a directory") {
		t.Errorf("a directory: %v", err)
	}
	if _, err = workingSide(root, root, "id_rsa"); !errors.Is(err, ErrCredential) {
		t.Errorf("a credential file: %v", err)
	}
	if _, err = workingSide(filepath.Join(root, "dir"), root, "ok.txt"); err == nil || !strings.Contains(err.Error(), "outside the project") {
		t.Errorf("outside the project folder: %v", err)
	}
	if s, err = workingSide(filepath.Join(root, "dir"), root, "dir/inner.txt"); err != nil || string(s.Data) != "inner\n" {
		t.Errorf("inside the project folder: %+v, %v", s, err)
	}
	if linked {
		// A link is read as git reads it: its target as the content, whatever it points to.
		s, err := workingSide(root, root, "link.txt")
		if err != nil || !strings.HasSuffix(string(s.Data), "far.txt") || s.Size != int64(len(s.Data)) {
			t.Errorf("a link: %+v, %v", s, err)
		}
		if strings.Contains(string(s.Data), "far\n") {
			t.Errorf("the link's target was opened: %q", s.Data)
		}
	}
	if dirLinked {
		if _, err = workingSide(root, root, "linkdir/far.txt"); err == nil || !strings.Contains(err.Error(), "outside the project") {
			t.Errorf("through a linked folder: %v", err)
		}
	}
	writeAt(t, root, "nul.bin", "ab\x00cd")
	writeAt(t, root, "lfs.txt", lfsOld)
	writeAt(t, root, "empty.txt", "")
	if s, _ = workingSide(root, root, "nul.bin"); !s.Binary || s.Data != nil || s.Size != 5 {
		t.Errorf("binary: %+v", s)
	}
	if s, _ = workingSide(root, root, "lfs.txt"); !s.LFS || s.Data != nil {
		t.Errorf("LFS: %+v", s)
	}
	if s, _ = workingSide(root, root, "empty.txt"); s.Data == nil || len(s.Data) != 0 || s.Missing || s.Size != 0 {
		t.Errorf("empty: %+v", s)
	}
	// Binary is a NUL in the first 8,000 bytes (D6): the 8,000th is in, the
	// 8,001st is not.
	writeAt(t, root, "late.txt", strings.Repeat("x", 8000)+"\x00")
	if s, _ = workingSide(root, root, "late.txt"); s.Binary || len(s.Data) != 8001 {
		t.Errorf("a NUL as the 8,001st byte: Binary %v", s.Binary)
	}
	writeAt(t, root, "early.txt", strings.Repeat("x", 7999)+"\x00")
	if s, _ = workingSide(root, root, "early.txt"); !s.Binary {
		t.Errorf("a NUL as the 8,000th byte: not binary")
	}
}

// TestUntrackedFiles covers the files of an untracked folder: recursive,
// sorted, limited, and ListDir's rules.
func TestUntrackedFiles(t *testing.T) {
	root := t.TempDir()
	for _, rel := range []string{"dir/a.txt", "dir/B.txt", "dir/sub/c.txt", "dir/sub/deeper/d.txt", "dir/.git/config",
		"dir/.env", "dir/sub/id_rsa", "dir/empty/.keep", "other/z.txt", "dir/é x.txt"} {
		writeAt(t, root, rel, "x")
	}
	if err := os.Remove(filepath.Join(root, "dir", "empty", ".keep")); err != nil {
		t.Fatal(err)
	}
	outside := t.TempDir()
	writeAt(t, outside, "far.txt", "x")
	if os.Symlink(filepath.Join(outside, "far.txt"), filepath.Join(root, "dir", "link.txt")) != nil {
		t.Logf("symbolic links not made: the link cases are skipped")
	}
	_ = os.Symlink(outside, filepath.Join(root, "dir", "linkdir"))

	want := []string{"dir/B.txt", "dir/a.txt", "dir/sub/c.txt", "dir/sub/deeper/d.txt", "dir/é x.txt"}
	for _, rel := range []string{"dir/", "dir", "dir//"} {
		files, truncated, err := UntrackedFiles(root, rel, 1000)
		if err != nil || truncated || !reflect.DeepEqual(files, want) {
			t.Errorf("%q: %q, truncated %v, %v; want %q", rel, files, truncated, err, want)
		}
	}
	files, truncated, err := UntrackedFiles(root, "dir/sub/", 1000)
	if err != nil || truncated || !reflect.DeepEqual(files, []string{"dir/sub/c.txt", "dir/sub/deeper/d.txt"}) {
		t.Errorf("dir/sub: %q, %v, %v", files, truncated, err)
	}
	files, truncated, err = UntrackedFiles(root, "dir/empty/", 1000)
	if err != nil || truncated || len(files) != 0 {
		t.Errorf("dir/empty: %q, %v, %v", files, truncated, err)
	}

	// The limit: exactly the number of files is not truncated, one under is.
	if files, truncated, err = UntrackedFiles(root, "dir/", len(want)); err != nil || truncated || len(files) != len(want) {
		t.Errorf("exactly the limit: %q, %v, %v", files, truncated, err)
	}
	files, truncated, err = UntrackedFiles(root, "dir/", 2)
	if err != nil || !truncated || len(files) != 2 || !sort.StringsAreSorted(files) {
		t.Errorf("limit 2: %q, %v, %v", files, truncated, err)
	}
	again, _, _ := UntrackedFiles(root, "dir/", 2)
	if !reflect.DeepEqual(files, again) {
		t.Errorf("two reads differ: %q %q", files, again)
	}
	if files, _, _ = UntrackedFiles(root, "dir/", -1); len(files) != len(want) {
		t.Errorf("no limit: %q", files)
	}

	for rel, why := range map[string]string{
		"nope/":     "not found: nope",
		"../":       "outside the project",
		"dir/a.txt": "cannot be read",
	} {
		if _, _, err := UntrackedFiles(root, rel, 10); err == nil || !strings.Contains(err.Error(), why) {
			t.Errorf("%q: %v, want %q", rel, err, why)
		}
	}
	// The whole project.
	if files, _, err = UntrackedFiles(root, "", 1000); err != nil || len(files) != len(want)+1 || files[len(files)-1] == "" {
		t.Errorf("the root: %q, %v", files, err)
	}
	if files, _, err = UntrackedFiles(root, ".", 1000); err != nil || len(files) != len(want)+1 {
		t.Errorf("the root as dot: %q, %v", files, err)
	}
}
