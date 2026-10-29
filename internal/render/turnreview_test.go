package render

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/Evengelius/agent-command-center/internal/event"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// reviewConv is a conversation of one turn, the owner's message and the
// CLI's answer, whose review is r (plan 65, design D7).
func reviewConv(r state.TurnReview, answer bool) *state.State {
	b := newConvBench("claude-code", "claude-haiku-5-5")
	b.ev(event.UserMessage, event.User{Text: "do it"})
	b.ev(event.MessageMarker, event.Marker{Name: "message_start"})
	if answer {
		b.ev(event.MessageCompleted, event.Completed{Text: "Done."})
	}
	b.ev(event.TurnCompleted, event.TurnEnd{Outcome: "completed", Subtype: "success"})
	r.Done = true
	b.s.Stages[0].Review = r
	return b.s
}

// rowsWith are the zone B rows holding part, as text.
func rowsWith(rows []string, part string) []string {
	var out []string
	for _, r := range rows {
		if strings.Contains(r, part) {
			out = append(out, r)
		}
	}
	return out
}

func TestReviewRows(t *testing.T) {
	files := []state.ReviewFile{{Status: 'M', Path: "internal/render/viewer.go", Added: 8, Removed: 2}, {Status: 'R', Path: "doc/new.md", Orig: "doc/old.md", Added: 1, Removed: 1}, {Status: 'A', Path: "logo.png", Binary: true}}
	big := state.ReviewLeft{Path: "data/big.bin", Why: "over 32.0 MiB", Size: 40 << 20, ModTime: time.Unix(1, 0)}
	r := state.TurnReview{From: "t1", To: "t2", Files: files, Total: 3, Left: []state.ReviewLeft{big}}

	// Folded: one row right under the turn's last row, a lone answer kept in
	// the CLI's block, so in that block (the result box's case is the golden
	// frames' review-*): the count with the file left out, the lines of the
	// files listed, "git" at its right.
	s := reviewConv(r, true)
	rows := rowTexts(bRows(s, UI{}, 76))
	got := rowsWith(rows, "files changed")
	if len(got) != 1 || !strings.Contains(got[0], "▸ 4 files changed in turn 1 · +9 −3 · Review") || !strings.HasSuffix(got[0], "git") {
		t.Fatalf("the folded row: %q", got)
	}
	answer := -1
	for i, row := range rows {
		if strings.Contains(row, "Done.") {
			answer = i
		}
	}
	if answer < 0 || answer+1 >= len(rows) || !strings.HasPrefix(rows[answer+1], "▎") || !strings.Contains(rows[answer+1], "files changed") {
		t.Fatalf("the row is not the CLI block's next row:\n%s", strings.Join(rows, "\n"))
	}

	// Unfolded: the files as the Git Graph lists a commit's, then the file
	// left out with why, never its content (design D6); the rows counted
	// are the rows drawn, so that the scroll and the cursor find them.
	ui := UI{Unfolded: map[string]bool{ReviewKey(1): true}}
	for _, u := range []UI{{}, ui} {
		if view, n, _, _ := conversation(s, u, 76, 1<<20); n != len(view)+len(sessionState(s, u, 76)) {
			t.Fatalf("%d rows counted, %d drawn", n, len(view))
		}
	}
	rows = rowTexts(bRows(s, ui, 76))
	for _, want := range []string{"M  viewer.go internal/render/", "R  new.md ← doc/old.md doc/", "A  logo.png", "·  big.bin data/ · over 32.0 MiB, not compared"} {
		if len(rowsWith(rows, want)) != 1 {
			t.Errorf("no row with %q:\n%s", want, strings.Join(rows, "\n"))
		}
	}
	if got := rowsWith(rows, "logo.png"); len(got) != 1 || !strings.HasSuffix(got[0], "binary") {
		t.Errorf("the binary file's row: %q", got)
	}
	if got := rowsWith(rows, "viewer.go"); len(got) != 1 || !strings.HasSuffix(got[0], "+8 −2") {
		t.Errorf("the file's counts: %q", got)
	}
	if n := ReviewStops(s, ui, len(s.Conversation)); n != 5 {
		t.Errorf("%d stops, want the summary, three files and the one left out", n)
	}
	if f := ReviewFile(s, len(s.Conversation), 3); f != 1 {
		t.Errorf("stop 3 is file %d, want 1", f)
	}
	if f := ReviewFile(s, len(s.Conversation), 5); f != -1 {
		t.Errorf("the file left out opens file %d", f)
	}
}

func TestReviewRowsProblem(t *testing.T) {
	// No review: one row saying why, in the failure colour; no stop.
	s := reviewConv(state.TurnReview{Problem: "its start's snapshot took more than 5s"}, true)
	rows := rowTexts(bRows(s, UI{}, 76))
	if got := rowsWith(rows, "No review"); len(got) != 1 || !strings.Contains(got[0], "✗ No review of turn 1: its start's snapshot took more than 5s") {
		t.Fatalf("the failure's row: %q", got)
	}
	if n := ReviewStops(s, UI{}, len(s.Conversation)); n != 0 {
		t.Errorf("a failure's row has %d stops", n)
	}
	// A turn that changed nothing: no row.
	s = reviewConv(state.TurnReview{From: "t1", To: "t1"}, true)
	if got := rowsWith(rowTexts(bRows(s, UI{}, 76)), "files changed"); got != nil {
		t.Fatalf("an unchanged turn's rows: %q", got)
	}
}

func TestReviewRowsCap(t *testing.T) {
	// Past the state's 1,000 files, the note of how many there were.
	files := make([]state.ReviewFile, state.ReviewFileCap)
	for i := range files {
		files[i] = state.ReviewFile{Status: 'M', Path: fmt.Sprintf("f%04d.go", i), Added: 1}
	}
	s := reviewConv(state.TurnReview{From: "t1", To: "t2", Files: files, Total: 2345}, true)
	rows := rowTexts(bRows(s, UI{Unfolded: map[string]bool{ReviewKey(1): true}}, 76))
	if got := rowsWith(rows, "first 1,000 of 2,345 files"); len(got) != 1 {
		t.Fatalf("no note of the files past the cap")
	}
	// The count grouped by thousands, as the note is (the review of
	// 2026-10-29).
	if got := rowsWith(rows, "files changed"); len(got) != 1 || !strings.Contains(got[0], "2,345 files changed") {
		t.Fatalf("the count: %q", got)
	}
}

func TestReviewRowsBlock(t *testing.T) {
	// Under the CLI's block when there is no result box: its bar; under the
	// owner's message alone, no block (design D7).
	s := reviewConv(state.TurnReview{From: "t1", To: "t2", Files: []state.ReviewFile{{Status: 'A', Path: "x.txt", Added: 1}}, Total: 1}, false)
	got := rowsWith(rowTexts(bRows(s, UI{}, 76)), "file changed")
	if len(got) != 1 || strings.HasPrefix(got[0], "▎") {
		t.Fatalf("under the owner's message: %q", got)
	}
	if !strings.Contains(got[0], "▸ 1 file changed in turn 1 · +1 −0 · Review") {
		t.Fatalf("one file: %q", got)
	}
}

func TestReviewCursorOffItsEntry(t *testing.T) {
	// A turn whose last entry is a tool call, which shows the cursor itself:
	// with the cursor on the review, the call's row is not marked too.
	b := newConvBench("claude-code", "claude-haiku-5-5")
	b.ev(event.UserMessage, event.User{Text: "do it"})
	b.ev(event.MessageMarker, event.Marker{Name: "message_start"})
	b.call("c1", "Read", map[string]string{"file_path": `C:\work\proj\a.go`}, true, false)
	b.end("completed")
	b.s.Stages[0].Review = state.TurnReview{From: "t1", To: "t2", Files: []state.ReviewFile{{Status: 'M', Path: "a.go", Added: 1}}, Total: 1, Done: true}
	n := len(b.s.Conversation)
	rows := rowTexts(bRows(b.s, UI{Focus: "B", Entry: n, EntryReview: 1}, 76))
	if got := rowsWith(rows, "Read"); len(got) != 1 || strings.Contains(got[0], "entries") {
		t.Fatalf("the call's row with the cursor on the review: %q", got)
	}
	if got := rowsWith(rowTexts(bRows(b.s, UI{Focus: "B", Entry: n}, 76)), "Read"); len(got) != 1 || !strings.Contains(got[0], "entries") {
		t.Fatalf("the call's row with the cursor on it: %q", got)
	}
}

func TestReviewCursorRows(t *testing.T) {
	// The cursor on the summary: its mark and keys; on a file: the file's
	// keys; the entry's own row never marked meanwhile (design D8).
	s := reviewConv(state.TurnReview{From: "t1", To: "t2", Files: []state.ReviewFile{{Status: 'M', Path: "a.go", Added: 1}}, Total: 1}, true)
	n := len(s.Conversation)
	ui := UI{Focus: "B", Entry: n, EntryReview: 1}
	rows := rowTexts(bRows(s, ui, 76))
	if got := rowsWith(rows, "file changed"); len(got) != 1 || !strings.Contains(got[0], "▸ ▸ 1 file changed") || !strings.Contains(got[0], "Enter unfolds · ↑↓ entries · Esc back") {
		t.Fatalf("the summary under the cursor: %q", got)
	}
	if got := rowsWith(rows, "Done."); len(got) != 1 || strings.Contains(got[0], "▸") || strings.Contains(got[0], "entries") {
		t.Fatalf("the entry's own row marked too: %q", got)
	}
	if got := entryFooter(s, ui); got != "Enter or → unfold · ↑↓ entries · Esc back to the input · F1 keys" {
		t.Fatalf("footer %q", got)
	}
	ui.Unfolded, ui.EntryReview = map[string]bool{ReviewKey(1): true}, 2
	rows = rowTexts(bRows(s, ui, 76))
	if got := rowsWith(rows, "a.go"); len(got) != 1 || !strings.Contains(got[0], "Enter diff · ← fold · ↑↓ entries · Esc back") {
		t.Fatalf("the file under the cursor: %q", got)
	}
	if got := entryFooter(s, ui); got != "Enter diff · ← fold · ↑↓ entries · Esc back to the input · F1 keys" {
		t.Fatalf("footer %q", got)
	}
	if got := ReviewRowsAfter(s, ui, n, 2); got != 1 {
		t.Fatalf("%d rows after the file, want the conversation's last blank row", got)
	}
}
