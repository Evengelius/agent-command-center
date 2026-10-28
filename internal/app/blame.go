package app

import (
	"errors"
	"slices"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"

	"github.com/Evengelius/agent-command-center/internal/input"
	"github.com/Evengelius/agent-command-center/internal/render"
)

// The file viewer's git blame (plan 60, brief amendment 62; ui-design.md,
// Git blame): `b` turns it on and off for the rest of the session, and each
// file shown while it is on is blamed in a command, git given the very bytes
// the viewer shows (design D5). An answer for a file since closed or
// replaced, or for a read that `b` has since replaced, is dropped: each read
// carries the file's generation and a number of its own.

// blameMsg is a blame's answer: git's, or that the file changed since it was
// shown.
type blameMsg struct {
	gen, seq int
	b        input.Blame
	err      error
	changed  bool
}

// toggleBlame is `b` in the viewer (design D1): on, the file shown is
// blamed, read again first and blamed only if it is still as shown; off,
// the column goes and a read under way is dropped.
func (m *Model) toggleBlame() []tea.Cmd {
	m.blameOn = !m.blameOn
	m.blameSeq++
	f := &m.ui.File
	f.Blame = render.FileBlame{On: m.blameOn}
	if !m.blameOn || f.State != "" || len(f.Lines) == 0 {
		return nil // off, or the read showing the file asks for it once it ends
	}
	bg, ok := m.git().(input.BlameGit)
	if !ok {
		f.Blame.Note, f.Blame.NoteBad = "no git in this run", true
		return nil
	}
	f.Blame.Note = "blame…"
	gen, seq, root, path, shown := m.fileGen, m.blameSeq, m.fileRoot(), f.Path, f.Lines
	return []tea.Cmd{func() tea.Msg {
		ft, err := input.ReadFile(root, path)
		if err != nil || ft.Binary || !slices.Equal(ft.Lines, shown) {
			return blameMsg{gen: gen, seq: seq, changed: true}
		}
		b, err := bg.Blame(root, ft.Rel, ft.Raw)
		return blameMsg{gen: gen, seq: seq, b: b, err: err}
	}}
}

// blameFile blames the file just shown, from the bytes its read gave.
func (m *Model) blameFile(raw []byte) []tea.Cmd {
	f := &m.ui.File
	bg, ok := m.git().(input.BlameGit)
	if !ok {
		f.Blame.Note, f.Blame.NoteBad = "no git in this run", true
		return nil
	}
	m.blameSeq++
	f.Blame.Note = "blame…"
	gen, seq, root, rel := m.fileGen, m.blameSeq, m.fileRoot(), f.Rel
	return []tea.Cmd{func() tea.Msg {
		b, err := bg.Blame(root, rel, raw)
		return blameMsg{gen: gen, seq: seq, b: b, err: err}
	}}
}

// blameRead shows a blame's answer (designs D2 and D6): its lines and
// commits, or why there are none.
func (m *Model) blameRead(msg blameMsg) {
	f := &m.ui.File
	if msg.gen != m.fileGen || msg.seq != m.blameSeq || !f.Open || !m.blameOn {
		return // a file closed or replaced, a read replaced, or the blame off
	}
	f.Blame = render.FileBlame{On: true}
	switch {
	case msg.changed || msg.err == nil && len(msg.b.Lines) != len(f.Lines):
		f.Blame.Note, f.Blame.NoteBad = "the file changed after it was read: show it again", true
	case msg.err != nil:
		f.Blame.Note, f.Blame.NoteBad = blameProblem(msg.err), true
	default:
		f.Blame.Lines, f.Blame.Commits = msg.b.Lines, m.blameCommits(msg.b.Commits)
	}
}

// blameProblem is a blame's failure as the footer says it (design D6).
func blameProblem(err error) string {
	var r input.Refusal
	switch {
	case errors.As(err, &r):
		return r.Msg
	case errors.Is(err, input.ErrNoGit):
		return "git not found"
	case errors.As(err, new(input.NotRepoError)):
		return "not a git repository"
	}
	return err.Error()
}

// blameCommits are git's commits with the texts the viewer draws, made
// once: the short hash, the initials, the dates in the machine's zone.
func (m *Model) blameCommits(cs []input.BlameCommit) []render.BlameCommit {
	loc := m.opt.Location
	if loc == nil {
		loc = time.Local
	}
	out := make([]render.BlameCommit, len(cs))
	for i, c := range cs {
		if c.Uncommitted {
			out[i] = render.BlameCommit{Uncommitted: true}
			continue
		}
		t := c.Time.In(loc)
		out[i] = render.BlameCommit{Short: c.Hash[:min(len(c.Hash), 7)], Author: c.Author, Initials: initials(c.Author),
			Summary: c.Summary, Date: t.Format("02/01/2006"), DateShort: t.Format("02/01/06"), When: t.Format("02/01/2006 15:04")}
	}
	return out
}

// initials is a name's words' first letters, upper case, three at most.
func initials(name string) string {
	var b strings.Builder
	for i, w := range strings.Fields(name) {
		if i == 3 {
			break
		}
		r, _ := utf8.DecodeRuneInString(w)
		b.WriteRune(unicode.ToUpper(r))
	}
	return b.String()
}
