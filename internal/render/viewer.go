package render

import (
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/Evengelius/agent-command-center/internal/event"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// The approval viewer (ui-design.md, Approvals; Scrolling, collapsing, full
// screen): Enter on a focused card shows the whole request full screen, every
// line of the command, diff or content, wrapped and never cut. A rendering of
// the same state, so an answer or an expiry shows while it is open.

// viewKeys are the viewer's hints while its approval waits; the answer keys
// act as on the card.
const (
	viewKeys         = "a allow once · d deny · c cancel the turn · ↑↓ PgUp PgDn Home End scroll · Esc back to the card"
	viewQuestionKeys = "1-4 choose · ← → question · a send the answers · d deny · c cancel the turn · ↑↓ PgUp PgDn Home End scroll · Esc back to the card"
)

// viewer draws approval ap whole; sc records the body's scrolling, as for an
// inspection.
func viewer(s *state.State, ui UI, ap *state.Approval, sc *Screen) []line {
	w := ui.Width
	label, st := "! APPROVAL", stAttentionBold
	if ap.Question {
		label = "? QUESTION"
	}
	keys := strings.Replace(viewKeys, "a allow once · ", "a allow once · "+forSession(ap), 1)
	if ap.Question {
		keys = viewQuestionKeys
	}
	var outcome line
	switch ap.Status {
	case state.Allowed:
		if ap.Question {
			label, st, outcome = "✓ ANSWERED", stDone, decoLine("answered by you "+clockSec(ap.AnsweredAt)+" ", stDone)
			break
		}
		by := "allowed by you "
		if ap.Answer == event.AllowSession {
			by = "allowed for the session by you "
		}
		label, st, outcome = "✓ ALLOWED", stDone, decoLine(by+clockSec(ap.AnsweredAt)+" ", stDone)
	case state.Denied:
		label, st, outcome = "✗ DENIED", stFailed, decoLine("denied by you "+clockSec(ap.AnsweredAt)+" ", stFailed)
		if ap.Problem != "" {
			outcome = decoLine("denied automatically: request not readable ", stFailed)
		} else if ap.Question && !ap.Cancelled {
			outcome = decoLine("denied by you "+clockSec(ap.AnsweredAt)+" · answer in your next message ", stFailed)
		}
	case state.Expired:
		label, st, outcome = "· EXPIRED", stGrey, decoLine("expired, not answered ", stGrey)
	case state.Answering:
		outcome = decoLine("sending answer… ", stProgress)
	default:
		outcome = decoLine("requested "+clock(ap.RequestedAt)+" ", stGrey)
	}
	if ap.Status != state.Pending {
		keys = scrollKeys
	}
	head := spread(line{deco(" ◀ back (Esc)   ", stTitle), deco(label, st), deco("  "+revealed(approvalWho(s, ap)+" · "+ap.Tool), stBold)}, outcome, w)
	return fullScreen(s, ui, sc, head, viewerBodyFor(s, ui, ap, w-2), keys)
}

// viewerBody is the request: its target and summary, then its file changes
// as a coloured diff, else every line the card shows the first of.
func viewerBody(s *state.State, ap *state.Approval, w int) []line {
	return viewerBodyFor(s, UI{}, ap, w)
}

// viewerBodyFor is viewerBody with the owner's choices of a question.
func viewerBodyFor(s *state.State, ui UI, ap *state.Approval, w int) []line {
	raw, _, body := approvalDetail(ap)
	// The request's format characters are drawn, never applied (plan 48,
	// design D6), here and in a tool call's full view, which shows its input
	// as this view does.
	target, body := revealed(raw), revealedLines(body)
	var out []line
	if target != "" {
		out = append(out, wrapped(target, w, stBold, "")...)
	}
	if ap.Summary != "" && ap.Summary != raw {
		out = append(out, wrapped(revealed(ap.Summary), w, stText, "")...)
	}
	if ap.Problem != "" {
		out = append(out, wrapped(revealed(ap.Problem), w, stFailed, "")...)
	}
	if diffs := approvalDiffs(s, ap); len(diffs) > 0 {
		for _, d := range diffs {
			if len(out) > 0 {
				out = append(out, line{})
			}
			if hideUser(d.path) != raw { // Claude Code names the file as the target
				out = append(out, line{sp(revealed(hideUser(d.path)), stBold), sp("  "+d.kind, stGrey)}.fit(w))
			}
			for _, l := range d.lines {
				st := stText
				switch {
				case strings.HasPrefix(l, "+"):
					st = stDone
				case strings.HasPrefix(l, "-"):
					st = stFailed
				}
				out = append(out, wrapped(revealed(l), w, st, "  ")...)
			}
		}
		return out
	}
	if qs := QuestionsOf(ap); len(qs) > 0 {
		if len(out) > 0 {
			out = append(out, line{})
		}
		if ap.Status == state.Allowed || ap.Answers != nil {
			return append(out, answeredLines(qs, ap.Answers, w)...)
		}
		return append(out, questionLines(qs, pickOf(ui, ap, qs), ap.Status == state.Pending, w)...)
	}
	if len(body) > 0 && len(out) > 0 {
		out = append(out, line{})
	}
	for _, l := range body {
		out = append(out, wrapped(l, w, stText, "  ")...)
	}
	if len(out) == 0 {
		out = append(out, plain("the request carries no command, diff or content", stGrey))
	}
	return out
}

// approvalDiffs are the file changes an approval asks for: Claude Code's
// Edit, MultiEdit and Write from the request's input; Codex's fileChange from
// the tool call it names (the request itself has no diff).
func approvalDiffs(s *state.State, ap *state.Approval) []fileDiff {
	if ap.Tool == "fileChange" {
		if tc := s.Tools[ap.ToolID]; tc != nil {
			return fileDiffs(tc.Name, tc.Input)
		}
		return nil
	}
	return fileDiffs(ap.Tool, ap.Input)
}

// wrapped is text wrapped to w, its next rows under lead's width.
func wrapped(text string, w int, st *lipgloss.Style, lead string) []line {
	var out []line
	for j, l := range wrap(text, max(w-len(lead), 1)) {
		if j == 0 {
			out = append(out, line{sp(l, st)})
			continue
		}
		out = append(out, line{deco(lead, stNone), sp(l, st)})
	}
	return out
}

// The entry cursor and viewer (ui-design.md, Zone B, Expanding): ↑ from an
// empty input moves a cursor over the entries B shows in part, and Enter
// shows the one under it whole, full screen.
const (
	entryHint     = " Enter full view · ↑↓ entries · Esc back"
	entryFileHint = " Enter full view · F3 file · ↑↓ entries · Esc back"
	entryKeys     = "Enter full view · ↑↓ entries · Esc back to the input · F1 keys"
	entryFileKeys = "Enter full view · ↑↓ entries · Esc back to the input · F3 file · F1 keys"
)

// Expandable reports whether conversation entry e has more than B shows: a
// sent message folded to its first lines, a tool call (B shows its name), an
// error (B shows its first line, cut to the width), a phase of thinking that
// ended (B shows its summary; plan 22, design 8).
func Expandable(e state.Entry) bool {
	switch e.Role {
	case "user":
		return len(entryLines(nil, e)) > foldOver
	case "tool", "error", "review":
		return true
	case "thinking":
		return thoughtShown(e.Thought)
	}
	return false
}

// entryViewer draws conversation entry i whole; a rendering of the state, so
// a tool's output and status show as they arrive.
func entryViewer(s *state.State, ui UI, i int, sc *Screen) []line {
	w := ui.Width
	e := s.Conversation[i]
	back := deco(" ◀ back (Esc)   ", stTitle)
	var head line
	var body []line
	switch e.Role {
	case "user":
		ls := entryLines(s, e)
		head = spread(line{back, deco("YOU", stAttentionBold), deco("  "+clock(e.At), stGrey)}, decoLine(grouped(len(ls))+" lines ", stGrey), w)
		for _, l := range ls {
			body = append(body, wrapped(l, w-2, stText, "")...)
		}
	case "tool":
		head, body = toolView(s, ui, e, back, w)
	case "thinking":
		head, body = thoughtView(e, back, w)
	case "review":
		head = line{back, deco("◈ AUTO-REVIEW [UNSTABLE]", stBold), deco("  "+clockSec(e.At), stGrey)}
		for _, l := range strings.Split(e.Text, "\n") {
			body = append(body, wrapped(l, w-2, stText, "")...)
		}
		body = append(body, line{}, plain("observed from Codex's item/autoApprovalReview, a notification marked unstable in its schema", stGrey))
	default:
		head = line{back, deco("✗ TURN FAILED", stFailedBold)}
		if !e.At.IsZero() {
			head = append(head, deco("  "+clockSec(e.At), stGrey))
		}
		for _, l := range strings.Split(e.Text, "\n") {
			body = append(body, wrapped(l, w-2, stFailed, "")...)
		}
	}
	keys := scrollKeys
	if _, ok := EntryFile(s, e); ok {
		keys = "F3 file · " + scrollKeys // plan 17, Opening
	}
	return fullScreen(s, ui, sc, head.fit(w), body, keys)
}

// toolView is a tool call's head, its status and origin, and its body: the
// input as the approval viewer shows a request, then the whole output.
func toolView(s *state.State, ui UI, e state.Entry, back span, w int) (line, []line) {
	tc := s.Tools[e.MessageID]
	if tc == nil {
		return line{back, deco("▸ "+e.Text, stBold)}, []line{plain("no call recorded under this entry", stGrey)}
	}
	head := spread(line{back, toolSymbol(s, tc, deco("◉", stProgress)), deco(" "+tc.Name, stBold),
		deco("  "+strings.ToLower(string(tc.Status))+" · "+elapsed(tc.StartedAt, tc.EndedAt, ui.Now), stGrey)},
		decoLine(originLabel(tc.Origin)+" ", stGrey), w)
	w -= 2
	ap := &state.Approval{Tool: tc.Name, Input: tc.Input, ToolID: tc.ID}
	var body []line
	if target, _, detail := approvalDetail(ap); target != "" || len(detail) > 0 || len(approvalDiffs(s, ap)) > 0 {
		body = viewerBody(s, ap, w)
	} else if in := strings.TrimSpace(string(tc.Input)); in != "" && in != "null" && in != "{}" {
		body = wrapped(in, w, stText, "") // an input of no known shape, as received
	} else {
		body = []line{plain("no input", stGrey)}
	}
	if tc.Reviewed != "" {
		// The field is undocumented: said where the owner can see it (plan 12, item 2).
		body = append(body, line{}, line{deco("PERMISSION", stBold)}, plain(tc.Reviewed+" · tool_result_meta.permission_decision, an undocumented field of Claude Code", stText))
	}
	body = append(body, line{}, line{deco("OUTPUT", stBold)})
	if tc.Output == "" {
		return head, append(body, plain("no output", stGrey))
	}
	if tc.OutputCut > 0 {
		body = append(body, plain(cutNote(tc.OutputCut), stGrey))
	}
	for _, l := range strings.Split(strings.TrimRight(tc.Output, "\n"), "\n") {
		body = append(body, wrapped(l, w, stText, "  ")...)
	}
	return head, body
}

// entryFooter is the key hints while B's entry cursor is on: F3 among them
// when the entry names a file (plan 17, Opening).
func entryFooter(s *state.State, ui UI) string {
	// On a run of calls: its row expands it, the calls of an expanded one
	// fold it (plan 19).
	_, _, open, folded, ok := FoldOf(s, ui, ui.Entry)
	if folded {
		return "Enter or → expand · ↑↓ entries · Esc back to the input · F1 keys"
	}
	if i := ui.Entry - 1; i >= 0 && i < len(s.Conversation) && s.Conversation[i].Role == "thinking" {
		// A phase's summary row: → opens its details, ← folds them (plan 22,
		// design 8), then its run of calls.
		return strings.Replace(entryKeys, "↑↓ entries", thoughtKeys(ui.Unfolded[ThoughtKey(s.Conversation[i])], ok && open)+"↑↓ entries", 1)
	}
	if ok && open {
		return strings.Replace(entryFooterFor(s, ui), "↑↓ entries", "← fold · ↑↓ entries", 1)
	}
	return entryFooterFor(s, ui)
}

// thoughtKeys are the keys a summary row adds before "↑↓ entries": ← folds
// its details when open, → opens them, then ← folds its run of calls when it
// is in an expanded one (plan 22, design 8).
func thoughtKeys(details, run bool) string {
	switch {
	case details:
		return "← fold · "
	case run:
		return "→ details · ← fold · "
	}
	return "→ details · "
}

// entryFooterFor is the key hints for the entry under the cursor.
func entryFooterFor(s *state.State, ui UI) string {
	if i := ui.Entry - 1; i < len(s.Conversation) {
		if _, ok := EntryFile(s, s.Conversation[i]); ok {
			return entryFileKeys
		}
	}
	return entryKeys
}
