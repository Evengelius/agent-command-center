package input

import (
	"bufio"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"
)

// gitAs runs git in dir as the author name at date, its committer the same.
func gitAs(t *testing.T, dir, name, date string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME="+name, "GIT_AUTHOR_DATE="+date,
		"GIT_COMMITTER_NAME="+name, "GIT_COMMITTER_DATE="+date)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
}

// blameRepo is a repository where f.txt was added by Ana Hubaut, then
// reworked by Bo Li, in two zones.
func blameRepo(t *testing.T) string {
	t.Helper()
	gitEnv(t)
	dir := t.TempDir()
	gitT(t, dir, "init", "-q", "-b", "main")
	write(t, filepath.Join(dir, "f.txt"), "one\ntwo\nthree\n")
	gitT(t, dir, "add", "f.txt")
	gitAs(t, dir, "Ana Hubaut", "2022-11-15T10:00:00+01:00", "commit", "-q", "-m", "Add f.txt")
	write(t, filepath.Join(dir, "f.txt"), "one\nTWO by Bo\nthree\nfour\n")
	gitAs(t, dir, "Bo Li", "2023-03-02T08:30:00-05:00", "commit", "-q", "-am", "Rework f.txt")
	return dir
}

// TestBlame is plan 60, criterion 60.7, in git: the blame of the bytes
// given, each line's commit, its author, time and zone and subject, a
// commit's fields read once and its later lines by hash, the changed and
// added lines not committed; the call a read fed by its standard input.
func TestBlame(t *testing.T) {
	dir := blameRepo(t)
	var mu sync.Mutex
	var calls [][]string
	g := GitCLI{Trace: func(a []string) { mu.Lock(); calls = append(calls, a); mu.Unlock() }}

	// The working tree's bytes: line 3 changed, line 5 added.
	content := "one\nTWO by Bo\nthree changed\nfour\nfive\n"
	write(t, filepath.Join(dir, "f.txt"), content)
	b, err := g.Blame(dir, "f.txt", []byte(content))
	if err != nil {
		t.Fatal(err)
	}
	if len(b.Commits) != 3 {
		t.Fatalf("%d commits, want 3: %+v", len(b.Commits), b.Commits)
	}
	at := func(line int) BlameCommit { return b.Commits[b.Lines[line]] }
	ana, bo := at(0), at(1)
	if ana.Author != "Ana Hubaut" || ana.Summary != "Add f.txt" || ana.Uncommitted ||
		!ana.Time.Equal(time.Date(2022, 11, 15, 9, 0, 0, 0, time.UTC)) || ana.Time.Format("-0700") != "+0100" {
		t.Errorf("line 1's commit %+v", ana)
	}
	if bo.Author != "Bo Li" || bo.Summary != "Rework f.txt" || !bo.Time.Equal(time.Date(2023, 3, 2, 13, 30, 0, 0, time.UTC)) ||
		bo.Time.Format("-0700") != "-0500" || len(bo.Hash) != 40 {
		t.Errorf("line 2's commit %+v", bo)
	}
	if b.Lines[3] != b.Lines[1] {
		t.Errorf("line 4 is Bo's commit again by its hash: %v", b.Lines)
	}
	for _, l := range []int{2, 4} {
		if c := at(l); !c.Uncommitted || c.Hash != zeroHash {
			t.Errorf("line %d: %+v, want not committed", l+1, c)
		}
	}
	if len(b.Lines) != 5 {
		t.Errorf("%d lines, want 5", len(b.Lines))
	}
	mu.Lock()
	last := calls[len(calls)-1]
	mu.Unlock()
	if s := strings.Join(last, " "); !strings.HasSuffix(s, "blame --porcelain --contents - -- f.txt") ||
		!strings.Contains(s, "-c core.fsmonitor=false") {
		t.Errorf("the call %q", s)
	}
	// The read's environment, command's as every call's: no optional lock,
	// so that git never refreshes the index, and git's words in English.
	if cmd, err := g.command(context.Background(), dir, true, []string{"blame"}); err != nil ||
		!slices.Contains(cmd.Env, "GIT_OPTIONAL_LOCKS=0") || !slices.Contains(cmd.Env, "LC_ALL=C") {
		t.Errorf("the read's environment: %v", err)
	}

	// The bytes given are blamed, not the disk's: the first commit's text.
	b, err = g.Blame(dir, "f.txt", []byte("one\ntwo\nthree\n"))
	if err != nil {
		t.Fatal(err)
	}
	// A line not committed is known by its hash: with --contents git names
	// its author "External file (--contents)", which is never shown.
	var who []string
	for _, i := range b.Lines {
		c := b.Commits[i]
		if c.Uncommitted {
			who = append(who, "not committed")
			continue
		}
		who = append(who, c.Author)
	}
	if want := []string{"Ana Hubaut", "not committed", "Ana Hubaut"}; !reflect.DeepEqual(who, want) {
		t.Errorf("the first commit's text blamed as %q, want %q", who, want)
	}
}

// TestBlameCRLF is criterion 60.7's line endings: a file committed with
// CRLF, its bytes given as read, CRs kept, is blamed as committed; its
// lines without their CRs would all differ.
func TestBlameCRLF(t *testing.T) {
	gitEnv(t)
	dir := t.TempDir()
	gitT(t, dir, "init", "-q", "-b", "main")
	gitT(t, dir, "config", "core.autocrlf", "false")
	raw := "x\r\ny\r\n"
	write(t, filepath.Join(dir, "c.txt"), raw)
	gitT(t, dir, "add", "c.txt")
	gitT(t, dir, "commit", "-q", "-m", "CRLF")
	g := GitCLI{}
	for _, c := range []struct {
		content     string
		uncommitted bool
	}{{raw, false}, {"x\ny\n", true}} {
		b, err := g.Blame(dir, "c.txt", []byte(c.content))
		if err != nil {
			t.Fatal(err)
		}
		if len(b.Lines) != 2 {
			t.Fatalf("%q: %d lines", c.content, len(b.Lines))
		}
		for i, l := range b.Lines {
			if b.Commits[l].Uncommitted != c.uncommitted {
				t.Errorf("%q, line %d: not committed %v, want %v", c.content, i+1, b.Commits[l].Uncommitted, c.uncommitted)
			}
		}
	}
}

// TestBlameRefusals is criterion 60.5 in git (design D6): an untracked
// file, a repository with no commit, a folder outside git, the time limit.
func TestBlameRefusals(t *testing.T) {
	dir := blameRepo(t)
	g := GitCLI{}
	write(t, filepath.Join(dir, "u.txt"), "new\n")
	var r Refusal
	if _, err := g.Blame(dir, "u.txt", []byte("new\n")); !errors.As(err, &r) || r.Kind != "untracked" || r.Msg != "not in git yet: no history" {
		t.Errorf("an untracked file: %v", err)
	}
	empty := t.TempDir()
	gitT(t, empty, "init", "-q", "-b", "main")
	write(t, filepath.Join(empty, "a.txt"), "a\n")
	gitT(t, empty, "add", "a.txt")
	if _, err := g.Blame(empty, "a.txt", []byte("a\n")); !errors.As(err, &r) || r.Kind != "no-commits" || r.Msg != "no commits yet" {
		t.Errorf("no commits: %v", err)
	}
	if _, err := g.Blame(t.TempDir(), "a.txt", []byte("a\n")); !errors.As(err, new(NotRepoError)) {
		t.Errorf("outside git: %v", err)
	}
	old := blameLimit
	blameLimit = time.Nanosecond
	defer func() { blameLimit = old }()
	if _, err := g.Blame(dir, "f.txt", []byte("one\n")); !errors.As(err, &r) || r.Kind != "timeout" || !strings.HasPrefix(r.Msg, "blame stopped after ") {
		t.Errorf("the time limit: %v", err)
	}
	if e := (&timeoutError{"blame", 15 * time.Second}); e.Error() != "git blame took more than 15s" || !errors.Is(e, context.DeadlineExceeded) {
		t.Errorf("the time limit's error %q", e.Error())
	}
}

// TestParseBlame is the porcelain's reading: a commit's fields once, its
// later lines by hash, the fields the viewer does not use skipped, lines
// numbered as git numbers them; a line that is no part of the format is an
// error.
func TestParseBlame(t *testing.T) {
	a, z := strings.Repeat("a", 40), zeroHash
	out := a + " 1 2 1\nauthor A B\nauthor-mail <a@x>\nauthor-time 1668502800\nauthor-tz +0100\n" +
		"committer A B\nsummary First\nboundary\nfilename f.txt\n\ttwo\n" +
		z + " 2 1 1\nauthor Not Committed Yet\nauthor-time 1793180108\nauthor-tz +0100\nsummary Version of f.txt\n" +
		"previous " + a + " f.txt\nfilename f.txt\n\tone\n" +
		a + " 3 3\n\tthree\n"
	b, err := parseBlame(bufio.NewReader(strings.NewReader(out)))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(b.Lines, []int{1, 0, 0}) || len(b.Commits) != 2 || b.Commits[0].Author != "A B" ||
		b.Commits[0].Summary != "First" || !b.Commits[1].Uncommitted || b.Commits[0].Time.Unix() != 1668502800 {
		t.Errorf("parsed %+v", b)
	}
	if _, err := parseBlame(bufio.NewReader(strings.NewReader("not a header\n"))); err == nil {
		t.Error("a line outside the format was read")
	}
	if b, err := parseBlame(bufio.NewReader(strings.NewReader(""))); err != nil || len(b.Lines) != 0 {
		t.Errorf("an empty file's blame: %+v, %v", b, err)
	}
}
