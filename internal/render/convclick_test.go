package render

import (
	"regexp"
	"strings"
	"testing"

	"github.com/Evengelius/agent-command-center/internal/provider/claude"
)

// edgeRow is a row of zone B with no text: blank, a block's bar, a card's
// edge.
var edgeRow = regexp.MustCompile(`^[\s▎─╭╮╰╯│]*$`)

// TestConvClickMarks is plan 42's design D4 (brief amendment 43): zone B's
// rows in view say what a click and a double click on them do. A thinking
// phase's summary row carries its details' key and its entry, an Enter
// target; each row of an entry's own content carries its entry, a blank
// row or a card's edge inside it too; a speaker's label, a rule between two
// messages and the rows after the entries (the session's state) carry
// neither.
func TestConvClickMarks(t *testing.T) {
	s, now := replayed(claude.ProviderID, claude.NewMapper(), records(t, "p12c-block-claude-auto-boundary-edit.jsonl"))
	ui := UI{Width: 160, Height: 50, Now: now, Focus: "B"}
	sc := Draw(s, ui)
	rows := strings.Split(texts(sc.lines), "\n")
	c := sc.Conv
	targets := Targets(s, ui)
	zoneB := func(r int) string {
		_, b, _ := strings.Cut(rows[c.Y+r], "││")
		return strings.TrimRight(b, " │")
	}
	// inside says row r lies in entry n's own rows: the nearest row with
	// text above it is n's, or, just under a label, the nearest below is. A
	// blank row before an entry's body (a rule's, a gap's) has another
	// entry's text above it.
	inside := func(r, n int) bool {
		near := func(d int) (int, bool) {
			for q := r + d; q >= 0 && q < c.Room; q += d {
				if !edgeRow.MatchString(zoneB(q)) {
					m, ok := c.Entries[q]
					return m, ok
				}
			}
			return 0, false
		}
		up, okUp := near(-1)
		if okUp {
			return up == n
		}
		down, okDown := near(1)
		return okDown && down == n
	}
	end := c.Starts[len(s.Conversation)] // the entries' rows end here
	summaries, content, tail := 0, 0, 0
	for r := 0; r < c.Room; r++ {
		b := zoneB(r)
		text := strings.Trim(b, " ▎")
		n, entry := c.Entries[r]
		key, fold := c.Folds[r]
		switch {
		case c.Top+r >= end:
			tail++
			if entry || fold {
				t.Errorf("row %d, after the entries: entry %d, fold %q: %q", r, n, key, b)
			}
		case strings.Contains(b, "Thought for"):
			summaries++
			if !entry || !fold || s.Conversation[n-1].Role != "thinking" || key != ThoughtKey(s.Conversation[n-1]) || !targets[n-1] {
				t.Errorf("row %d, a summary row: entry %d (%v), fold %q (%v), want a thinking target and its key: %q", r, n, entry, key, fold, b)
			}
		case text != "" && strings.Trim(text, "─") == "":
			if entry || fold {
				t.Errorf("row %d, a rule between two messages: entry %d, fold %q", r, n, key)
			}
		case edgeRow.MatchString(b):
			if fold || entry && !inside(r, n) {
				t.Errorf("row %d, no text: entry %d outside it, fold %q: %q", r, n, key, b)
			}
		case strings.HasPrefix(text, "CLAUDE"):
			if entry || fold {
				t.Errorf("row %d, a speaker's label: entry %d, fold %q: %q", r, n, key, b)
			}
		default:
			content++
			if !entry || n < 1 || n > len(s.Conversation) {
				t.Errorf("row %d, an entry's content with no entry (%d): %q", r, n, b)
			}
		}
	}
	if summaries == 0 || content == 0 || tail == 0 {
		t.Fatalf("%d summary rows, %d content rows and %d after the entries in view:\n%s", summaries, content, tail, strings.Join(rows[c.Y:c.Y+c.Room], "\n"))
	}
}
