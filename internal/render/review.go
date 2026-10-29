package render

import (
	"fmt"
	"strconv"

	"charm.land/lipgloss/v2"

	"github.com/Evengelius/agent-command-center/internal/state"
)

// The turn review in zone B (plan 65, brief amendment 67, design D7): under
// a turn's last row, its RESULT box or else its block's last row, rows of
// the dashboard's own drawn from the stage's review, never a message of the
// conversation. Folded, one row: "▸ 3 files changed in turn 2 · +12 −4 ·
// Review"; unfolded, the files under it as the Git Graph lists a commit's
// (plan 62, design D5). A turn without a review says why in one row.

// ReviewKey is a turn's review in ui.Unfolded: unfolded, its files show.
func ReviewKey(turn int) string { return "review:" + strconv.Itoa(turn) }

// TurnFile is a review's file row: the turn and the file, 1-based, among
// the review's files (design D8: Enter or a double click opens its diff).
type TurnFile struct{ Turn, File int }

// reviewAnchors are the entries zone B draws a turn's review under, by
// index: each turn's last entry, for the turns whose review shows.
func reviewAnchors(s *state.State) map[int]int {
	var last map[int]int // turn -> its last entry
	for i := range s.Conversation {
		if t := s.Conversation[i].Stage; t > 0 && t <= len(s.Stages) && s.Stages[t-1].Review.Shown() {
			if last == nil {
				last = map[int]int{}
			}
			last[t] = i
		}
	}
	if last == nil {
		return nil
	}
	out := make(map[int]int, len(last))
	for t, i := range last {
		out[i] = t
	}
	return out
}

// ReviewOf is the turn whose review zone B draws under entry n (1-based);
// 0 for none.
func ReviewOf(s *state.State, n int) int {
	return reviewAnchors(s)[n-1]
}

// ReviewStops is how many entry cursor stops the review under entry n has
// (design D8): its summary, then, unfolded, each file it lists; none for a
// review that only says why it has none.
func ReviewStops(s *state.State, ui UI, n int) int {
	t := ReviewOf(s, n)
	if t == 0 {
		return 0
	}
	r := s.Stages[t-1].Review
	if r.Problem != "" {
		return 0
	}
	if !ui.Unfolded[ReviewKey(t)] {
		return 1
	}
	return 1 + len(r.Files) + len(r.Left)
}

// ReviewStopsAll is ReviewStops for every entry with a review, by entry
// (1-based): the entry cursor's walk counts them in one pass.
func ReviewStopsAll(s *state.State, ui UI) map[int]int {
	out := map[int]int{}
	for i, t := range reviewAnchors(s) {
		if r := s.Stages[t-1].Review; r.Problem == "" {
			out[i+1] = 1
			if ui.Unfolded[ReviewKey(t)] {
				out[i+1] += len(r.Files) + len(r.Left)
			}
		}
	}
	return out
}

// ReviewRowsAfter is how many of entry n's rows follow its review's stop k:
// the review's rows after it, and the blank row that ends the conversation
// when n is its last; the cursor's row is its span's end less these.
func ReviewRowsAfter(s *state.State, ui UI, n, k int) int {
	t := ReviewOf(s, n)
	if t == 0 {
		return 0
	}
	after := reviewCount(s, ui, t) - k
	if looks(s, ui)[n-1].tail {
		after++
	}
	return max(after, 0)
}

// ReviewFile is the file under stop k of the review under entry n: its
// index among the review's files, from 0; -1 on the summary or on a file
// left out, which opens nothing (design D8).
func ReviewFile(s *state.State, n, k int) int {
	t := ReviewOf(s, n)
	if t == 0 || k < 2 || k-2 >= len(s.Stages[t-1].Review.Files) {
		return -1
	}
	return k - 2
}

// reviewCount is how many rows reviewRows draws for turn t's review.
func reviewCount(s *state.State, ui UI, t int) int {
	r := s.Stages[t-1].Review
	if r.Problem != "" || !ui.Unfolded[ReviewKey(t)] {
		return 1
	}
	n := 1 + len(r.Files) + len(r.Left)
	if r.Total > len(r.Files) {
		n++ // "first 1,000 of N files"
	}
	return n
}

// reviewRows are turn t's review rows under entry i, drawn with its look k:
// in the CLI's block, or with no block after a result box or the owner's
// message.
func reviewRows(s *state.State, ui UI, t, i int, k look, w int) []crow {
	r := s.Stages[t-1].Review
	fr := &frame{kind: k.card, w: w}
	if k.result || k.card == cardYou {
		fr.kind = cardNone
	}
	cw := cardContent(fr.kind, w)
	cursor := ui.Card == "" && (ui.Focus == "B" || ui.Focus == "") && ui.Entry == i+1
	on := func(stop int) bool { return cursor && ui.EntryReview == stop }
	if r.Problem != "" {
		return []crow{{fn: func() line { return reviewProblem(t, r.Problem, cw) }, fr: fr}}
	}
	open := ui.Unfolded[ReviewKey(t)]
	out := []crow{{fn: func() line { return reviewSummary(t, r, open, on(1), cw) }, fr: fr, fold: ReviewKey(t), cursor: on(1)}}
	if !open {
		return out
	}
	for j := range r.Files {
		f, stop := r.Files[j], 2+j
		out = append(out, crow{fn: func() line { return reviewFileRow(f, on(stop), cw) }, fr: fr, cursor: on(stop), turn: t, file: j + 1})
	}
	for j := range r.Left {
		l, stop := r.Left[j], 2+len(r.Files)+j
		out = append(out, crow{fn: func() line { return reviewLeftRow(l, on(stop), cw) }, fr: fr, cursor: on(stop)})
	}
	if r.Total > len(r.Files) {
		note := fmt.Sprintf("first %s of %s files", grouped(len(r.Files)), grouped(r.Total))
		out = append(out, crow{l: grounded(append(spaces(reviewIndent), deco(note, stGrey)), cw), fr: fr})
	}
	return out
}

// withReview is an entry's rows with its turn's review rows after them,
// before the blank row that ends the conversation when tail.
func withReview(rs, review []crow, tail bool) []crow {
	if !tail || len(rs) == 0 {
		return append(rs, review...)
	}
	out := make([]crow, 0, len(rs)+len(review))
	out = append(append(out, rs[:len(rs)-1]...), review...)
	return append(out, rs[len(rs)-1])
}

// reviewIndent is where a file's letter stands: under the summary's count.
const reviewIndent = 3

// reviewSummary is the summary row, w cells on the tool rows' ground: how
// many files the turn changed, its lines added and removed, "still
// changing" when the listing after its end found a file written meanwhile
// (design D5), "git" at its right, what read it; on, the entry cursor's
// keys there instead.
func reviewSummary(t int, r state.TurnReview, open, on bool, w int) line {
	n := r.Total + len(r.Left)
	what := fmt.Sprintf("%s files changed in turn %d", grouped(n), t) // "2,345", as the cap's note (the review of 2026-10-29)
	if n == 1 {
		what = fmt.Sprintf("1 file changed in turn %d", t)
	}
	mark, act, keys := "▸ ", " · Review", " Enter unfolds · ↑↓ entries · Esc back "
	if open {
		mark, act, keys = "▾ ", " · Fold", " Enter folds · ↑↓ entries · Esc back "
	}
	l := line{deco(" ", stNone), deco(mark, stFoldMark), deco(what, stMuted)}
	var added, removed int
	for _, f := range r.Files {
		added, removed = added+f.Added, removed+f.Removed
	}
	if added > 0 || removed > 0 {
		l = append(l, deco(" · ", stGrey), deco("+"+strconv.Itoa(added), stDiffAdd), deco(" −"+strconv.Itoa(removed), stDiffDel))
	}
	if r.Changing {
		l = append(l, deco(" · ", stGrey), deco("still changing", stAttention))
	}
	right := decoLine("git", stGrey)
	if on {
		right = decoLine(keys, stAttention)
	} else {
		l = append(l, deco(act, stGrey))
	}
	return grounded(spread(l, right, w), w)
}

// reviewProblem is the row of a turn without a review: why, in the failure
// colour (design D7).
func reviewProblem(t int, why string, w int) line {
	l := line{deco(" ", stNone), deco("✗ ", stFailed), deco(fmt.Sprintf("No review of turn %d: %s", t, why), stFailed)}
	return grounded(spread(l, decoLine("git", stGrey), w), w)
}

// reviewFileRow is a file the turn changed, as the Git Graph lists one; on,
// the entry cursor's keys in place of its counts.
func reviewFileRow(f state.ReviewFile, on bool, w int) line {
	g := GraphFile{Status: f.Status, Path: f.Path, Orig: f.Orig, Added: f.Added, Removed: f.Removed, Binary: f.Binary}
	same := func(st *lipgloss.Style) *lipgloss.Style { return st }
	if !on {
		return grounded(graphFileRow(g, spaces(reviewIndent), same, w), w)
	}
	keys := decoLine(" Enter diff · ← fold · ↑↓ entries · Esc back ", stAttention)
	g.Added, g.Removed, g.Binary = 0, 0, false
	return grounded(spread(graphFileRow(g, spaces(reviewIndent), same, max(w-keys.width()-1, 0)), keys, w), w)
}

// reviewLeftRow is a file left out that the turn changed: its name, its
// folder, why, never its content (design D6); on, the cursor's keys.
func reviewLeftRow(f state.ReviewLeft, on bool, w int) line {
	dir, name := splitPath(f.Path)
	l := append(spaces(reviewIndent), deco("·", stGrey), deco("  ", stNone), deco(name, stExpFileName))
	if dir != "" {
		l = append(l, deco(" "+dir, stMuted))
	}
	l = append(l, deco(" · "+f.Why+", not compared", stGrey))
	var right line
	if on {
		right = decoLine(" ← fold · ↑↓ entries · Esc back ", stAttention)
	}
	return grounded(spread(l, right, w), w)
}

// reviewFooter is the key hints while the entry cursor is on a review's
// stop k (design D8).
func reviewFooter(s *state.State, ui UI) string {
	t := ReviewOf(s, ui.Entry)
	switch {
	case t == 0:
		return entryKeys
	case ui.EntryReview == 1 && ui.Unfolded[ReviewKey(t)]:
		return "Enter or ← fold · ↑↓ entries · Esc back to the input · F1 keys"
	case ui.EntryReview == 1:
		return "Enter or → unfold · ↑↓ entries · Esc back to the input · F1 keys"
	case ReviewFile(s, ui.Entry, ui.EntryReview) >= 0:
		return "Enter diff · ← fold · ↑↓ entries · Esc back to the input · F1 keys"
	}
	return "← fold · ↑↓ entries · Esc back to the input · F1 keys"
}
