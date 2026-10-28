package input

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
)

// The file viewer's git blame (plan 60, brief amendment 62): one read of
// the file's commits, given the very bytes the viewer shows (design D5).

// BlameGit is what the file viewer's blame asks of git; tests pass a fake.
type BlameGit interface {
	// Blame is git blame of the file at path, relative to dir, for content
	// as given: the bytes the viewer shows, as read (design D5).
	Blame(dir, path string, content []byte) (Blame, error)
}

var _ BlameGit = GitCLI{}

// Blame is git blame's answer: each line's commit, by its index in Commits;
// a line git names no commit for is -1.
type Blame struct {
	Lines   []int
	Commits []BlameCommit
}

// BlameCommit is a commit as the blame names it.
type BlameCommit struct {
	// Hash is the full hash; Uncommitted says it is git's all-zero one, a
	// line that differs from the last commit, whoever changed it.
	Hash        string
	Uncommitted bool
	// Author is the commit's author as git records it, not necessarily who
	// wrote the line; Time is the author's time, in the author's zone.
	Author string
	Time   time.Time
	// Summary is the commit's subject line.
	Summary string
}

// zeroHash is git blame's commit for a line not committed yet.
const zeroHash = "0000000000000000000000000000000000000000"

// blameLimit is a blame's time limit, a read's; a variable so that tests can
// lower it.
var blameLimit = readLimit

// Blame is git blame --porcelain --contents - -- path, the content on its
// standard input, so that the blame is of the bytes given, CRs kept, git
// converting line endings as it does from the disk. Git's refusals come back
// as the viewer names them (design D6): an untracked file, a repository
// with no commit, a folder outside git, the time limit; any other in git's
// words.
func (g GitCLI) Blame(dir, path string, content []byte) (Blame, error) {
	var b Blame
	err := g.stream(blameLimit, dir, bytes.NewReader(content), func(r *bufio.Reader) error {
		var perr error
		b, perr = parseBlame(r)
		return perr
	}, "blame", "--porcelain", "--contents", "-", "--", path)
	if err == nil {
		return b, nil
	}
	var ge *gitError
	switch msg := strings.ToLower(err.Error()); {
	case errors.Is(err, context.DeadlineExceeded):
		return Blame{}, Refusal{"timeout", fmt.Sprintf("blame stopped after %d s", int(blameLimit.Seconds()))}
	case !errors.As(err, &ge):
		return Blame{}, err
	case strings.Contains(msg, "no such path") && strings.Contains(msg, "in head"):
		return Blame{}, Refusal{"untracked", "not in git yet: no history"}
	case strings.Contains(msg, "no such ref: head"):
		return Blame{}, Refusal{"no-commits", "no commits yet"}
	case strings.Contains(msg, "not a git repository"):
		return Blame{}, NotRepoError{Dir: dir}
	}
	return Blame{}, err
}

// parseBlame reads git blame's porcelain: each run of lines a header line
// "<hash> <orig> <final> [<count>]", a commit's fields after its first
// header only, then each line's text after a tab.
func parseBlame(r *bufio.Reader) (Blame, error) {
	var b Blame
	index := map[string]int{}
	type when struct {
		unix int64
		tz   string
	}
	var times []when
	cur, final := -1, 0 // the commit of the line being read, and its number
	for {
		l, err := r.ReadString('\n')
		l = strings.TrimSuffix(l, "\n")
		switch {
		case l == "" && err != nil:
		case strings.HasPrefix(l, "\t"):
			// The line's text: its entry ends.
			if cur < 0 || final < 1 {
				return Blame{}, fmt.Errorf("git blame: a line's text with no header")
			}
			for len(b.Lines) < final {
				b.Lines = append(b.Lines, -1)
			}
			b.Lines[final-1] = cur
			cur = -1
		case cur < 0:
			f := strings.Fields(l)
			n := 0
			if len(f) >= 3 {
				n, _ = strconv.Atoi(f[2])
			}
			if len(f) < 3 || len(f[0]) != len(zeroHash) || n < 1 {
				return Blame{}, fmt.Errorf("git blame: unexpected line %q", l)
			}
			i, ok := index[f[0]]
			if !ok {
				i = len(b.Commits)
				index[f[0]] = i
				b.Commits = append(b.Commits, BlameCommit{Hash: f[0], Uncommitted: f[0] == zeroHash})
				times = append(times, when{})
			}
			cur, final = i, n
		default:
			k, v, _ := strings.Cut(l, " ")
			c := &b.Commits[cur]
			switch k {
			case "author":
				c.Author = v
			case "author-time":
				times[cur].unix, _ = strconv.ParseInt(v, 10, 64)
			case "author-tz":
				times[cur].tz = v
			case "summary":
				c.Summary = v
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return Blame{}, err
		}
	}
	for i, t := range times {
		b.Commits[i].Time = time.Unix(t.unix, 0).In(zoneOf(t.tz))
	}
	return b, nil
}

// zoneOf is git's "+0100" as a zone; UTC when it reads none.
func zoneOf(tz string) *time.Location {
	if len(tz) != 5 || tz[0] != '+' && tz[0] != '-' {
		return time.UTC
	}
	h, err1 := strconv.Atoi(tz[1:3])
	m, err2 := strconv.Atoi(tz[3:])
	if err1 != nil || err2 != nil {
		return time.UTC
	}
	off := (h*60 + m) * 60
	if tz[0] == '-' {
		off = -off
	}
	return time.FixedZone(tz, off)
}
