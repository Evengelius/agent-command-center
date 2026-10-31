package render

import (
	"fmt"
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
	tool := ap.Tool
	cmd := ui.Memo.command(ap.Tool, ap.Input)
	if cmd != nil {
		tool = cmd.c.shell // plan 72, D1
	}
	head := spread(line{deco(" ◀ back (Esc)   ", stTitle), deco(label, st), deco("  "+revealed(approvalWho(s, ap)+" · "+tool), stBold)}, outcome, w)
	if cmd != nil {
		// A command's sections, which copy their source (plan 72, C2), then
		// what its CLI sent with it.
		var body []line
		body, sc.srcs = commandSections(nil, ui, cmd, w-2)
		ps := contextParts(s, ap, cmd, false)
		col := 0
		for _, p := range ps {
			col = max(col, len(p.label)+2)
		}
		if len(ps) > 0 {
			body = append(body, line{})
		}
		for _, p := range ps {
			body = append(body, sectionRow(strings.ToUpper(p.label), stBold, col, revealed(p.text), stText, w-2)...)
		}
		if ap.Problem != "" {
			body = append(append(body, line{}), wrapped(revealed(ap.Problem), w-2, stFailed, "")...)
		}
		return fullScreen(s, ui, sc, head, body, strings.Replace(keys, "↑↓", copyKey+"↑↓", 1))
	}
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
		head, body, sc.srcs = toolView(s, ui, e, back, w)
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
	if e.Role == "tool" && viewCommand(s, ui) != nil {
		keys = copyKey + keys // plan 72, C3
	}
	return fullScreen(s, ui, sc, head.fit(w), body, keys)
}

// toolView is a tool call's head, its status and origin, and its body: the
// input as the approval viewer shows a request, then the whole output. A
// command's call shows its shell, its exit code, then its command's
// sections, the folder it ran in, how it was decided and when it ran (plan
// 72, D9); srcs are the sections that copy their source (C2).
func toolView(s *state.State, ui UI, e state.Entry, back span, w int) (line, []line, []viewSrc) {
	tc := s.Tools[e.MessageID]
	if tc == nil {
		return line{back, deco("▸ "+e.Text, stBold)}, []line{plain("no call recorded under this entry", stGrey)}, nil
	}
	f := ui.Memo.command(tc.Name, tc.Input)
	if f != nil {
		return commandView(s, ui, tc, f, back, w)
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
		body = append(body, line{}, line{deco("PERMISSION", stBold)}, plain(tc.Reviewed+" · "+classifierNote, stText))
	}
	return head, outputRows(body, tc, w), nil
}

// classifierNote says where Claude's classifier's verdict comes from (plan
// 12, item 2).
const classifierNote = "tool_result_meta.permission_decision, an undocumented field of Claude Code"

// outputRows adds a call's OUTPUT section, its whole output, w cells wide.
func outputRows(body []line, tc *state.ToolCall, w int) []line {
	body = append(body, line{}, line{deco("OUTPUT", stBold)})
	if tc.Output == "" {
		return append(body, plain("no output", stGrey))
	}
	if tc.OutputCut > 0 {
		body = append(body, plain(cutNote(tc.OutputCut), stGrey))
	}
	for _, l := range strings.Split(strings.TrimRight(tc.Output, "\n"), "\n") {
		body = append(body, wrapped(l, w, stText, "  ")...)
	}
	return body
}

// viewSrc is a full view's section that copies its source, not its rows
// (plan 72, C2): body rows first to last, whose spans hold item first and
// their bytes' offsets in text; a wrap puts no line break in a copy, an
// indent no space, and a tab is copied as a tab.
type viewSrc struct {
	first, last int
	text        string
}

// sectionIndent is where a full view's section's rows start: under its
// heading's text.
const sectionIndent = 2

// commandView is a command call's full view (plan 72, D9): its head, the
// symbol, the shell, its status and time, its exit code when the CLI
// reports one, its origin; then COMMAND laid out for reading, AS SENT
// verbatim for a command taken out of Codex's launcher, IN the folder it
// was sent with, APPROVAL when a request or a review came, then OUTPUT.
func commandView(s *state.State, ui UI, tc *state.ToolCall, f *cmdFacts, back span, w int) (line, []line, []viewSrc) {
	d := decisionOf(s, tc)
	sym, status := toolSymbol(s, tc, deco("◉", stProgress)), strings.ToLower(string(tc.Status))
	if d.kind == decidedDenied {
		sym, status = deco("✗", stFailed), "denied" // nothing ran (D5)
	}
	what := "  " + status + " · " + elapsed(tc.StartedAt, tc.EndedAt, ui.Now)
	if tc.HasExitCode && d.kind != decidedDenied {
		what += fmt.Sprintf(" · exit %d", tc.ExitCode)
	}
	head := spread(line{back, sym, deco(" "+f.c.shell, stBold), deco(what, stGrey)}, decoLine(originLabel(tc.Origin)+" ", stGrey), w)
	w -= 2
	body, srcs := commandSections(nil, ui, f, w)
	if cwd := f.c.cwd; cwd != "" {
		body = append(append(body, line{}), sectionRow("IN", stBold, inCol, hideUser(cwd), stText, w)...)
	}
	if hist := approvalHistory(s, ui.Memo, tc, d, w); len(hist) > 0 {
		body = append(append(body, line{}, line{deco("APPROVAL", stBold)}), hist...)
	}
	return head, outputRows(body, tc, w), srcs
}

// commandSections adds a command's COMMAND section, laid out for reading
// (D3) and coloured (D8), and its AS SENT section, verbatim, for a command
// taken out of Codex's launcher, w cells wide; each copies its source
// (C2), recorded in srcs.
func commandSections(body []line, ui UI, f *cmdFacts, w int) ([]line, []viewSrc) {
	var srcs []viewSrc
	add := func(text string, rows []cmdRow, runs []styleRun) {
		first := len(body)
		for _, r := range rows {
			body = append(body, append(spaces(sectionIndent+r.ind), cmdSpans(text, r, ref{item: first}, false, stText, runs)...))
		}
		srcs = append(srcs, viewSrc{first, len(body) - 1, text})
	}
	lw := max(w-sectionIndent, 1)
	body = append(body, line{deco("COMMAND", stBold), deco(", laid out for reading", stGrey)})
	add(f.c.text, f.layout(lw), cmdColours(ui, f, stText))
	if f.c.unwrapped {
		body = append(body, line{}, line{deco("AS SENT", stBold), deco(", verbatim (Codex's command field)", stGrey)})
		add(f.c.sent, verbatimRows(f.c.sent, lw), nil)
	}
	return body, srcs
}

// verbatimRows lays text out in rows at most w cells wide as it is: its own
// line breaks, and a break at a blank, else at the width, the blank in
// neither row.
func verbatimRows(text string, w int) []cmdRow {
	var rows []cmdRow
	for ls := 0; ls <= len(text); {
		le := strings.IndexByte(text[ls:], '\n')
		if le < 0 {
			le = len(text)
		} else {
			le += ls
		}
		a := ls
		for {
			cells, end, blank := 0, a, -1
			for i := a; i < le; {
				n, cw := cmdCellSpan(text, i)
				if cells+cw > w {
					break
				}
				if text[i] == ' ' && i > a {
					blank = i
				}
				cells, i = cells+cw, i+n
				end = i
			}
			if end == le {
				rows = append(rows, cmdRow{a, le, 0})
				break
			}
			cut := end
			switch {
			case blank > a:
				cut = blank
			case end == a:
				n, _ := cmdCellSpan(text, a)
				cut = a + n
			}
			rows = append(rows, cmdRow{a, cut, 0})
			if a = cut; a < le && text[a] == ' ' {
				a++ // the blank at the break
			}
		}
		ls = le + 1
	}
	return rows
}

// sectionRow is a row of a full view's section: its label, in lst, padded to
// col cells, its text wrapped under that column.
func sectionRow(label string, lst *lipgloss.Style, col int, text string, st *lipgloss.Style, w int) []line {
	var out []line
	for j, l := range wrap(text, max(w-col, 1)) {
		lead := strings.Repeat(" ", col)
		if j == 0 {
			lead = label + strings.Repeat(" ", max(col-len(label), 1))
		}
		out = append(out, line{deco(lead, lst), sp(l, st)})
	}
	return out
}

// The columns of a full view's rows' texts: IN's beside the sections'
// headings, APPROVAL's rows' under its heading (plan 72, D9).
const (
	inCol   = len("IN         ")
	histCol = len("  description  ")
)

// approvalHistory is how call tc was decided, w cells wide (plan 72, D9):
// each request naming it, when and who asked, the reason or the
// description, the answer and when; Codex's review, its verdict, risk,
// authorization and rationale; Claude's classifier's verdict; then when it
// ran, or "not run" once denied. None for a call no one decided.
func approvalHistory(s *state.State, m *Memo, tc *state.ToolCall, d decision, w int) []line {
	var out []line
	row := func(label, text string, st *lipgloss.Style) {
		out = append(out, sectionRow("  "+label, stMuted, histCol, text, st, w)...)
	}
	note := func(text string) { out = append(out, sectionRow("", stNone, histCol, text, stGrey, w)...) }
	waiting := false
	for _, id := range s.Queue {
		ap := s.Approvals[id]
		if ap == nil || ap.ToolID != tc.ID || id == "" {
			continue
		}
		row("requested", clockSec(ap.RequestedAt)+"  by the "+approvalWho(s, ap), stText)
		if f := m.command(ap.Tool, ap.Input); f != nil && f.c.reason != "" {
			row("reason", revealed(f.c.reason), stText)
		} else if ap.Summary != "" && ap.Tool != "commandExecution" && (f == nil || ap.Summary != f.c.sent) {
			row("description", revealed(ap.Summary), stText)
		}
		switch {
		case ap.Status == state.Allowed && ap.Answer == event.AllowSession:
			row("decision", clockSec(ap.AnsweredAt)+"  allowed by you, for the session", stDone)
		case ap.Status == state.Allowed:
			row("decision", clockSec(ap.AnsweredAt)+"  allowed by you, once", stDone)
		case ap.Status == state.Denied && ap.Problem != "":
			row("decision", clockSec(ap.AnsweredAt)+"  denied automatically: request not readable", stFailed)
		case ap.Status == state.Denied:
			row("decision", clockSec(ap.AnsweredAt)+"  denied by you", stFailed)
		case ap.Status == state.Expired:
			row("decision", "expired, not answered", stGrey)
		case ap.Status == state.Answering:
			row("decision", "sending answer…", stProgress)
			waiting = true
		default:
			row("decision", "waiting for you", stAttention)
			waiting = true
		}
	}
	if r := tc.Review; r != nil {
		verdict := r.Status
		if p := reviewPart(r); p != "" {
			verdict += " · " + p
		}
		row("review", clockSec(r.At)+"  auto-review "+verdict, stText)
		note("observed from Codex's item/autoApprovalReview, a notification marked unstable in its schema")
	}
	if tc.Reviewed != "" {
		row("decision", clockSec(tc.ReviewedAt)+"  "+tc.Reviewed+", Claude's classifier", stText)
		note(classifierNote)
	}
	if len(out) == 0 {
		return nil
	}
	reviewing := tc.Review != nil && tc.Review.Status == "inProgress" && commandTool(tc.Name) && !tc.Status.Final()
	switch {
	case d.kind == decidedDenied:
		row("ran", "not run", stGrey)
	case waiting, reviewing, tc.StartedAt.IsZero(), d.kind == decidedExpired && !goesOn(tc):
		// not known to have run
	default:
		// It ran once the owner answered, or once Codex's review approved
		// it; Claude Code's classifier gives its verdict with the result,
		// after the run.
		from := tc.StartedAt
		if ap := s.Approvals[tc.Approval]; ap != nil && merges(ap) {
			if ap.AnsweredAt.After(from) {
				from = ap.AnsweredAt
			}
		} else if r := tc.Review; r != nil && r.Status == "approved" && commandTool(tc.Name) && r.At.After(from) {
			from = r.At
		}
		if tc.EndedAt.IsZero() {
			row("ran", clockSec(from)+", running", stProgress)
		} else {
			row("ran", clockSec(from)+" to "+clockSec(tc.EndedAt), stText)
		}
	}
	return out
}

// entryFooter is the key hints while B's entry cursor is on: F3 among them
// when the entry names a file (plan 17, Opening).
func entryFooter(s *state.State, ui UI) string {
	if ui.EntryReview > 0 {
		return reviewFooter(s, ui) // on a turn's review rows (plan 65, design D8)
	}
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
	if _, unfolded, block := CommandFold(s, ui, layout(s, ui).bw-4, ui.Entry); block {
		// A command's block: → unfolds it, ← folds it, then its run (plan
		// 72, K1, K2).
		if unfolded {
			return "← fold · " + entryFooterFor(s, ui)
		}
		if ok && open {
			return "→ unfold · " + strings.Replace(entryFooterFor(s, ui), "↑↓ entries", "← fold · ↑↓ entries", 1)
		}
		return "→ unfold · " + entryFooterFor(s, ui)
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
