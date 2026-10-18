package input

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

// names is entries as "name" for a file and "name/" for a folder.
func names(entries []TreeEntry) []string {
	var out []string
	for _, e := range entries {
		if e.Dir {
			out = append(out, e.Name+"/")
		} else {
			out = append(out, e.Name)
		}
	}
	return out
}

// TestListDir covers plan 17's criterion 5.1.
func TestListDir(t *testing.T) {
	root := t.TempDir()
	for _, rel := range []string{"b.go", "A.md", "a.txt", "zeta/x.go", "Alpha/y.go", "src/main/App.java",
		".git/config", ".env", "id_rsa", ".env.example", "empty/.keep"} {
		writeAt(t, root, rel, "x")
	}
	if err := os.Remove(filepath.Join(root, "empty", ".keep")); err != nil {
		t.Fatal(err)
	}
	outside := t.TempDir()
	writeAt(t, outside, "far.go", "x")
	if os.Symlink(filepath.Join(outside, "far.go"), filepath.Join(root, "far.go")) != nil {
		t.Logf("symbolic links not made: the link case is skipped")
	}
	_ = os.Symlink(outside, filepath.Join(root, "farDir"))

	entries, truncated, err := ListDir(root, "")
	if err != nil || truncated {
		t.Fatalf("ListDir root: truncated %v, %v", truncated, err)
	}
	want := []string{"Alpha/", "empty/", "src/", "zeta/", ".env.example", "A.md", "a.txt", "b.go"}
	if got := names(entries); !reflect.DeepEqual(got, want) {
		t.Errorf("root = %q, want %q", got, want)
	}
	if entries, _, err = ListDir(root, "src/main"); err != nil || !reflect.DeepEqual(names(entries), []string{"App.java"}) {
		t.Errorf("src/main = %q, %v", names(entries), err)
	}
	if entries, _, err = ListDir(root, "empty"); err != nil || len(entries) != 0 {
		t.Errorf("empty = %q, %v", names(entries), err)
	}

	for rel, why := range map[string]string{
		"missing": "not found: missing",
		"..":      "outside the project: ..",
		"../x":    "outside the project: ../x",
	} {
		if _, _, err := ListDir(root, rel); reason(err) != why {
			t.Errorf("ListDir %q: %q, want %q", rel, reason(err), why)
		}
	}
	if _, _, err := ListDir(root, "b.go"); err == nil || !strings.HasPrefix(reason(err), "cannot be read") {
		t.Errorf("a file: %q", reason(err))
	}

	defer func(n int) { MaxDirEntries = n }(MaxDirEntries)
	MaxDirEntries = 3
	entries, truncated, err = ListDir(root, "")
	if err != nil || !truncated || !reflect.DeepEqual(names(entries), want[:3]) {
		t.Errorf("limited: %q, truncated %v, %v", names(entries), truncated, err)
	}
	MaxDirEntries = len(want)
	if _, truncated, _ = ListDir(root, ""); truncated {
		t.Errorf("exactly at the limit: truncated")
	}
}

// TestParseStatus covers plan 17's criterion 5.2: every code of git's
// porcelain v1, a rename's source skipped, the prefix taken off.
func TestParseStatus(t *testing.T) {
	out := strings.Join([]string{
		" M web/a.ts", "M  b.go", "MM c.go", " T d.sh", "A  new.go", "AM new2.go",
		"R  web/to.ts", "web/from.ts", "C  copy.go", "orig.go", " D gone.go", "D  gone2.go",
		"UU both.go", "AA add.go", "DD del.go", "AU au.go", "UA ua.go", "DU du.go", "UD ud.go",
		"?? notes.txt", "?? tmp/", "!! build/", "!! app.log", "",
	}, "\x00")
	got := ParseStatus(out, "")
	wantFiles := map[string]byte{
		"web/a.ts": 'M', "b.go": 'M', "c.go": 'M', "d.sh": 'M', "new.go": 'A', "new2.go": 'A',
		"web/to.ts": 'R', "copy.go": 'A', "gone.go": 'D', "gone2.go": 'D',
		"both.go": '!', "add.go": '!', "del.go": '!', "au.go": '!', "ua.go": '!', "du.go": '!', "ud.go": '!',
		"notes.txt": 'U', "tmp/": 'U',
	}
	if !reflect.DeepEqual(got.Files, wantFiles) {
		t.Errorf("Files = %q,\nwant %q", got.Files, wantFiles)
	}
	if want := map[string]bool{"build/": true, "app.log": true}; !reflect.DeepEqual(got.Ignored, want) {
		t.Errorf("Ignored = %v", got.Ignored)
	}

	sub := ParseStatus(strings.Join([]string{" M Sub/a.go", "?? sub/n/", "!! other/x", "R  sub/to.go", "sub/from.go", " M sub/", ""}, "\x00"), "sub/")
	if want := map[string]byte{"a.go": 'M', "n/": 'U', "to.go": 'R'}; !reflect.DeepEqual(sub.Files, want) || len(sub.Ignored) != 0 {
		t.Errorf("under a prefix: %q, %v", sub.Files, sub.Ignored)
	}
	if e := ParseStatus("", ""); len(e.Files) != 0 || len(e.Ignored) != 0 {
		t.Errorf("no output: %v", e)
	}
}

// TestGitStatus covers plan 17's criterion 5.2 on a real repository: the
// markers (plan 17, criterion 6.4: a file and a folder deleted, each file
// under it a D), a project in a subfolder of the repository, and git writing
// nothing (its index unchanged though a file's time changed).
func TestGitStatus(t *testing.T) {
	repo := testRepo(t)
	write(t, filepath.Join(repo, ".gitignore"), "build/\n*.log\n")
	commit(t, repo, "keep.txt", "keep\n")
	writeAt(t, repo, "sub/s.go", "package s\n")
	writeAt(t, repo, "old/x.go", "package old\n")
	writeAt(t, repo, "old/deep/y.go", "package deep\n")
	gitT(t, repo, "add", ".")
	gitT(t, repo, "commit", "-q", "-m", "more")
	write(t, filepath.Join(repo, "a.txt"), "changed\n")
	write(t, filepath.Join(repo, "staged.go"), "package x\n")
	gitT(t, repo, "add", "staged.go")
	write(t, filepath.Join(repo, "notes.txt"), "new\n")
	writeAt(t, repo, "fresh/one.go", "x")
	writeAt(t, repo, "build/out.bin", "x")
	write(t, filepath.Join(repo, "run.log"), "x")
	writeAt(t, repo, "sub/t.go", "package s\n")
	writeAt(t, repo, "sub/s.go", "package s // changed\n")
	if err := os.Remove(filepath.Join(repo, "keep.txt")); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(filepath.Join(repo, "old")); err != nil { // a folder deleted whole: each file a D
		t.Fatal(err)
	}

	// A file's time moved without a change: git would refresh its index, if
	// it were allowed to write.
	later := time.Now().Add(time.Hour)
	if err := os.Chtimes(filepath.Join(repo, ".gitignore"), later, later); err != nil {
		t.Fatal(err)
	}
	index := filepath.Join(repo, ".git", "index")
	before, err := os.ReadFile(index)
	if err != nil {
		t.Fatal(err)
	}
	info, _ := os.Stat(index)

	var calls [][]string
	g := GitCLI{Trace: func(a []string) { calls = append(calls, a) }}
	gm, err := g.Status(repo)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]byte{"a.txt": 'M', "staged.go": 'A', "notes.txt": 'U', "fresh/": 'U',
		"sub/t.go": 'U', "sub/s.go": 'M', "keep.txt": 'D', "old/x.go": 'D', "old/deep/y.go": 'D'}
	if !reflect.DeepEqual(gm.Files, want) {
		t.Errorf("Files = %q,\nwant %q", gm.Files, want)
	}
	if want := map[string]bool{"build/": true, "run.log": true}; !reflect.DeepEqual(gm.Ignored, want) {
		t.Errorf("Ignored = %v", gm.Ignored)
	}

	gm, err = g.Status(filepath.Join(repo, "sub"))
	if err != nil {
		t.Fatal(err)
	}
	if want := map[string]byte{"t.go": 'U', "s.go": 'M'}; !reflect.DeepEqual(gm.Files, want) || len(gm.Ignored) != 0 {
		t.Errorf("in sub: %q, %v", gm.Files, gm.Ignored)
	}

	after, _ := os.ReadFile(index)
	info2, _ := os.Stat(index)
	if !bytes.Equal(before, after) || !info.ModTime().Equal(info2.ModTime()) {
		t.Errorf("git status rewrote the index")
	}
	for _, c := range calls {
		if j := strings.Join(c, " "); strings.Contains(j, "--force") || !strings.Contains(j, "rev-parse") && !strings.Contains(j, "status --porcelain=v1 -z") {
			t.Errorf("unexpected call: %s", j)
		}
	}

	if _, err := g.Status(t.TempDir()); !errors.As(err, new(NotRepoError)) {
		t.Errorf("outside a repository: %v", err)
	}
}
