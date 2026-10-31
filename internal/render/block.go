package render

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"charm.land/lipgloss/v2"

	"github.com/Evengelius/agent-command-center/internal/event"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// Zone B's commands (plan 72, D4 to D6; ui-design.md, Zone B, Commands): a
// call's command stays on its row when it fits the room the row leaves,
// else goes under it at an unfolded thought's indent, folded from blockFold
// lines to its first blockShow and a counter row; under the command, a row
// of Codex's review of the call, then a row of a failure's error, each cut
// to one row while folded. ▸ before the status marks a block with
// something hidden, ▾ once unfolded; a click on the call's row or the
// counter folds and unfolds it, as on plan 42's fold rows. The call's row
// says how the call was decided, in D5's words.

const (
	blockIndent = 4 // an unfolded thought's details' indent (plan 22)
	blockFold   = 4 // a command of this many lines or more folds…
	blockShow   = 2 // …to its first lines
)

// CommandKey is a call's command block in UI.Unfolded: unfolded, all of it
// shows (plan 72, S4).
func CommandKey(id string) string { return commandPrefix + id }

const commandPrefix = "cmd:"

// decision is how a call was decided, as D5 words it: the owner's answer
// to its request, Codex's auto-review, Claude's classifier.
type decision struct {
	kind  decisionKind
	words string
	st    *lipgloss.Style
	at    time.Time // when, for a denial's words
}

type decisionKind uint8

const (
	decidedNone    decisionKind = iota // no one decided it
	decidedRan                         // the call's own outcome follows the words
	decidedDenied                      // nothing ran: the denial's time follows
	decidedExpired                     // the call's own outcome follows only when it ran
)

// merges says an answered request is drawn on its call's row, the call's
// in zone B, no row of its own (plan 72, D5): a request the owner answered,
// or that expired, neither a question, whose answers show under its row,
// nor one denied for being unreadable.
func merges(ap *state.Approval) bool {
	return ap.Status.Final() && !ap.Question && ap.Problem == ""
}

// decisionOf is call tc's decision: its request's answer when it merges,
// else its auto-review's verdict, Codex's or Claude's classifier's; a
// request answered wins over a review.
func decisionOf(s *state.State, tc *state.ToolCall) decision {
	if ap := s.Approvals[tc.Approval]; ap != nil && merges(ap) {
		switch {
		case ap.Status == state.Allowed && ap.Answer == event.AllowSession:
			return decision{decidedRan, "allowed for the session by you", stDone, ap.AnsweredAt}
		case ap.Status == state.Allowed:
			return decision{decidedRan, "allowed by you", stDone, ap.AnsweredAt}
		case ap.Status == state.Denied:
			return decision{decidedDenied, "denied by you", stFailed, ap.AnsweredAt}
		default:
			return decision{decidedExpired, "expired, not answered", stGrey, ap.AnsweredAt}
		}
	}
	if r := tc.Review; r != nil && commandTool(tc.Name) {
		// Codex's words for its verdict (plan 12, item 2): any but approved
		// shown as a failure; a review of another call keeps its entry.
		switch r.Status {
		case "approved":
			return decision{decidedRan, "approved by auto-review", stDone, r.At}
		case "denied":
			return decision{decidedDenied, "denied by auto-review", stFailed, r.At}
		case "inProgress":
			return decision{decidedRan, "auto-review in progress", stProgress, r.At}
		case "timedOut":
			return decision{decidedRan, "auto-review timed out", stFailed, r.At}
		case "":
			return decision{decidedRan, "auto-review reported", stFailed, r.At}
		}
		return decision{decidedRan, "auto-review " + r.Status, stFailed, r.At}
	}
	switch tc.Reviewed {
	case "approved by auto-review":
		return decision{decidedRan, tc.Reviewed, stDone, tc.ReviewedAt}
	case "denied by auto-review":
		at := tc.ReviewedAt
		if at.IsZero() {
			at = tc.EndedAt
		}
		return decision{decidedDenied, tc.Reviewed, stFailed, at}
	}
	return decision{}
}

// deniedCall says call entry e was denied: nothing of what the CLI reports
// after shows (D5).
func deniedCall(s *state.State, e state.Entry) bool {
	tc := s.Tools[e.MessageID]
	return tc != nil && decisionOf(s, tc).kind == decidedDenied
}

// ownOutcome is how the call itself went, in today's words (plan 20):
// running and for how long, failed, denied, unfinished, or its end time.
func ownOutcome(ui UI, tc *state.ToolCall) line {
	switch {
	case !tc.Status.Final():
		return decoLine("running "+elapsed(tc.StartedAt, tc.EndedAt, ui.Now), stProgress)
	case tc.Status == state.ToolFailed:
		return decoLine("failed "+clockSec(tc.EndedAt), stFailed)
	case tc.Status == state.ToolDenied:
		return decoLine("denied "+clockSec(tc.EndedAt), stFailed)
	case tc.Status == state.ToolUnfinished:
		return decoLine("unfinished", stGrey)
	}
	return decoLine(clockSec(tc.EndedAt), stGrey)
}

// ran says the call reported a result: completed or failed.
func ran(tc *state.ToolCall) bool {
	return tc.Status == state.ToolCompleted || tc.Status == state.ToolFailed
}

// outcomeRoom is the most cells a call's outcome takes for decision d: its
// words and the longest of the call's own ("failed 15:04:05"). A command
// keeps to its row by it, not by the outcome drawn, so that it does not
// move as the outcome changes or the entry cursor's hint replaces it (plan
// 72, D4; V6).
func outcomeRoom(d decision) int {
	const own = len("failed 15:04:05")
	switch d.kind {
	case decidedNone:
		return own
	case decidedDenied:
		return textWidth(d.words) + len(" 15:04:05")
	}
	return textWidth(d.words) + len(" · ") - 1 + own // "·" is two bytes, one cell
}

// originRoom is the cells toolRight gives a row's origin.
func originRoom(o event.Origin) int {
	if originLabel(o) != "" {
		return len("  observed ")
	}
	return 1
}

// callName is the name entry e's row shows: for a command, the shell it
// runs in (plan 72, D1), else the tool's, as the CLI names it. m may be
// nil, s too (the entry's own text then).
func callName(s *state.State, m *Memo, e *state.Entry) string {
	name, _, _ := strings.Cut(e.Text, "\n")
	if s == nil || name != "commandExecution" {
		return name // the entry holds its tool's name: only Codex's commands change it
	}
	switch e.Role {
	case "tool":
		if tc := s.Tools[e.MessageID]; tc != nil && tc.Name == "commandExecution" {
			if f := m.command(tc.Name, tc.Input); f != nil {
				return f.c.shell
			}
		}
	case "approval":
		if ap := s.Approvals[e.MessageID]; ap != nil && ap.Tool == "commandExecution" {
			if f := m.command(ap.Tool, ap.Input); f != nil {
				return f.c.shell
			}
		}
	}
	return name
}

// reviewPart is what a call's block shows of Codex's review of it (plan
// 72, D6): its risk, the authorization with it when the risk is not low
// (plan 41), then its rationale; "" for none.
func reviewPart(r *state.Review) string {
	if r == nil {
		return ""
	}
	var head []string
	if r.Risk != "" {
		head = append(head, "risk "+r.Risk)
	}
	if r.Authorization != "" && r.Risk != "low" {
		head = append(head, "authorization "+r.Authorization)
	}
	h := strings.Join(head, " · ")
	switch {
	case r.Rationale == "":
		return h
	case h == "":
		return r.Rationale
	}
	return h + ": " + r.Rationale
}

// elsewhere says zone B draws entry e on another's row, or leaves it out
// while a card shows it (plan 72, D5, D6): a call while a request naming it
// waits, waiting's (waitingCalls), its card the call's; an answered
// request, on its call's row, zone B's; a review, under the command call it
// names. The search, too, finds it there (C4).
func elsewhere(s *state.State, e *state.Entry, waiting map[string]bool) bool {
	switch e.Role {
	case "tool":
		return waiting[e.MessageID]
	case "approval":
		ap := s.Approvals[e.MessageID]
		return ap != nil && ap.Agent == "" && merges(ap) && callEntry(s, s.Tools[ap.ToolID]) >= 0
	case "review":
		return attached(s, e)
	}
	return false
}

// waitingCalls are the calls a request waiting for the owner names, its
// card shown, read from the request queue once a pass; nil for none.
func waitingCalls(s *state.State) map[string]bool {
	var out map[string]bool
	for _, id := range s.Queue {
		if ap := s.Approvals[id]; ap != nil && ap.ToolID != "" && (ap.Status == state.Pending || ap.Status == state.Answering) {
			if out == nil {
				out = map[string]bool{}
			}
			out[ap.ToolID] = true
		}
	}
	return out
}

// attached says review entry e is drawn on the command call it names, zone
// B's (plan 72, D6): its entry is then not drawn. A review of another call
// keeps its entry.
func attached(s *state.State, e *state.Entry) bool {
	if e.Role != "review" || e.Target == "" {
		return false
	}
	tc := s.Tools[e.Target]
	return tc != nil && commandTool(tc.Name) && callEntry(s, tc) >= 0
}

// callEntry is the index of call tc's entry in the conversation, -1 when
// zone B draws none: a subagent's call, whose rows are zone C's (V8), or one
// known only from its end. Entries come in the order of their events.
func callEntry(s *state.State, tc *state.ToolCall) int {
	if tc == nil || tc.Agent != "" {
		return -1
	}
	conv := s.Conversation
	lo, hi := 0, len(conv)
	for lo < hi {
		if mid := (lo + hi) / 2; conv[mid].Seq < tc.Seq {
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	for i := lo; i < len(conv) && conv[i].Seq == tc.Seq; i++ {
		if conv[i].Role == "tool" && conv[i].MessageID == tc.ID {
			return i
		}
	}
	return -1
}

// blockKey is what a command call's rows count on besides its entry, for
// the row cache (plan 72, P1): its input, by identity, so its command; its
// decision's words, so its outcome's room; its status and, failed, its
// output and exit, so its error; its review; its name column; its origin;
// its block unfolded. Zero for any other entry.
type blockKey struct {
	in      *byte
	n       int
	words   string
	status  state.ToolStatus
	out     string
	exit    int
	hasExit bool
	review  *state.Review
	nameW   int
	origin  event.Origin
	open    bool
}

// blockKeyOf is entry e's blockKey, drawn with look k.
func blockKeyOf(s *state.State, e *state.Entry, k look) blockKey {
	if name, _, _ := strings.Cut(e.Text, "\n"); e.Role != "tool" || k.hidden || !commandTool(name) {
		return blockKey{} // the entry holds its tool's name
	}
	tc := s.Tools[e.MessageID]
	if tc == nil || len(tc.Input) == 0 || !commandTool(tc.Name) {
		return blockKey{}
	}
	b := blockKey{in: &tc.Input[0], n: len(tc.Input), words: decisionOf(s, tc).words, status: tc.Status, review: tc.Review, nameW: k.nameW, origin: e.Origin, open: k.cmdOpen}
	if tc.Status == state.ToolFailed {
		b.out, b.exit, b.hasExit = tc.Output, tc.ExitCode, tc.HasExitCode
	}
	return b
}

// commandTool says a tool runs a command: commandOf's tools.
func commandTool(name string) bool {
	return name == "Bash" || name == "PowerShell" || name == "commandExecution"
}

// cmdBlock is what a call's rows show of its command, its row w cells wide
// (plan 72, D4).
type cmdBlock struct {
	f      *cmdFacts
	tc     *state.ToolCall
	d      decision
	name   string   // the shell's name
	under  bool     // the command under its row, else on it
	rows   []cmdRow // its layout under the row
	review string   // the review's part, under the command; "" none
	err    string   // the failure's error, under it; "" none
	mark   bool     // something hidden while folded: ▸, or ▾ unfolded
	open   bool     // unfolded
	bw     int      // the width under the row
}

// blockOf is call entry e's block, drawn with look k in a row w cells
// wide; nil for a call that runs no command.
func blockOf(s *state.State, ui UI, e *state.Entry, w int, k look) *cmdBlock {
	tc := s.Tools[e.MessageID]
	if tc == nil {
		return nil
	}
	f := ui.Memo.command(tc.Name, tc.Input)
	if f == nil {
		return nil
	}
	b := &cmdBlock{f: f, tc: tc, d: decisionOf(s, tc), name: f.c.shell, open: k.cmdOpen, bw: max(w-blockIndent, 1)}
	b.review = reviewPart(tc.Review)
	if b.d.kind != decidedDenied { // a denial shows no error (D5)
		b.err = toolError(s, *e)
	}
	cut := b.review != "" && 2+srcCells(b.review) > b.bw || b.err != "" && srcCells(b.err) > b.bw
	markW := 0
	if cut {
		markW = 2
	}
	left := 3 + max(k.nameW, markW+textWidth(b.name))
	room := w - left - outcomeRoom(b.d) - originRoom(e.Origin) - 4
	if f.cells > room || !rowPlain(f.c.text) {
		b.under, b.rows = true, f.layout(b.bw)
		cut = cut || len(b.rows) >= blockFold
	}
	b.mark = cut
	return b
}

// rowPlain says a command can be drawn on its call's row: one line, no tab,
// control or format character, each of which the row's cut would not keep
// whole.
func rowPlain(text string) bool {
	for i := 0; i < len(text); {
		c := cmdCell(text, i)
		if c.ctrl || text[i] == '\t' {
			return false
		}
		i += c.n
	}
	return true
}

// reviewRanges are the review row's rows: whole unfolded, else its first
// with room for the "…" of its cut.
func (b *cmdBlock) reviewRanges() []rowRange {
	w := max(b.bw-2, 1)
	if !b.open && srcCells(b.review) > w {
		return wrapRanges(b.review, w-1, w-1)[:1]
	}
	if !b.open {
		return []rowRange{{0, len(b.review)}}
	}
	return wrapRanges(b.review, w, w)
}

// errRanges are the error row's rows, as reviewRanges.
func (b *cmdBlock) errRanges() []rowRange {
	if !b.open && srcCells(b.err) > b.bw {
		return wrapRanges(b.err, b.bw-1, b.bw-1)[:1]
	}
	if !b.open {
		return []rowRange{{0, len(b.err)}}
	}
	return wrapRanges(b.err, b.bw, b.bw)
}

// srcCells is how many cells s takes drawn as content (cells).
func srcCells(s string) int {
	n := 0
	for i := 0; i < len(s); {
		k, w := cellSpan(s, i)
		n, i = n+w, i+k
	}
	return n
}

// blockRows adds the rows under call entry i's row, in its card fr: the
// command's rows, the counter row while folded, the review's rows, the
// error's. The command and the review are the search's targets, never
// selected nor copied (plan 72, C1); the error is the entry's line 1, as on
// its row before (plan 5, item 6).
func (b *cmdBlock) blockRows(out []crow, ui UI, i int, fr *frame) []crow {
	key := CommandKey(b.tc.ID)
	if b.under {
		shown := len(b.rows)
		if !b.open && shown >= blockFold {
			shown = blockShow
		}
		for _, r := range b.rows[:shown] {
			out = append(out, crow{fn: func() line { return b.cmdRow(ui, i, r) }, fr: fr, tr: &targetRow{0, r.from, r.to}})
		}
		if shown < len(b.rows) {
			from := b.rows[shown].from
			out = append(out, crow{fn: func() line { return b.counter(i, len(b.rows)-shown, from) }, fold: key, fr: fr, tr: &targetRow{0, from, len(b.f.c.text)}})
		}
	}
	if b.review != "" {
		for _, r := range b.reviewRanges() {
			to := r.to
			if !b.open {
				to = len(b.review) // its row stands for what its cut hides
			}
			out = append(out, crow{fn: func() line { return b.reviewRow(i, r) }, fr: fr, tr: &targetRow{2, r.from, to}})
		}
	}
	if b.err != "" {
		for _, r := range b.errRanges() {
			to := r.to
			if !b.open {
				to = len(b.err)
			}
			out = append(out, crow{fn: func() line { return b.errRow(i, r) }, fr: fr, tr: &targetRow{1, r.from, to}})
		}
	}
	return out
}

// targetRow is the part of its entry a block's row shows, for a search's
// match (plan 72, C4): line 0 the call's command, its target; 1 its error,
// the entry's line 1; 2 its review's part, a target too; bytes [from, to).
type targetRow struct{ line, from, to int }

// cmdRow is a command's row under its call's row, on the tool rows' ground.
func (b *cmdBlock) cmdRow(ui UI, i int, r cmdRow) line {
	l := append(spaces(blockIndent+r.ind), cmdSpans(b.f.c.text, r, ref{panel: 'B', item: i}, true, stMuted, cmdColours(ui, b.f, stMuted))...)
	return grounded(l, b.bw+blockIndent)
}

// counter is a folded block's "… N more lines", standing for its hidden
// lines, from byte from on, which a search lights it for.
func (b *cmdBlock) counter(i, n, from int) line {
	more := deco(fmt.Sprintf("… %s more lines", grouped(n)), stGrey)
	more.ref = ref{panel: 'B', kind: refTargetCut, item: i, off: from, n: len(b.f.c.text) - from}
	return grounded(line{deco(strings.Repeat(" ", blockIndent), stNone), more}, b.bw+blockIndent)
}

// reviewRow is a row of the review's part, "◈ risk low: …" in the review's
// colour, its cut's "…" standing for the rest.
func (b *cmdBlock) reviewRow(i int, r rowRange) line {
	st := stProgress
	if s := b.tc.Review.Status; s != "inProgress" && s != "approved" {
		st = stFailed // a verdict other than approved (plan 12, item 2)
	}
	lead := "  "
	if r.from == 0 {
		lead = "◈ "
	}
	l := append(line{deco(strings.Repeat(" ", blockIndent), stNone), deco(lead, st)}, cmdSpans(b.review, cmdRow{r.from, r.to, 0}, ref{panel: 'B', item: i, line: 2}, true, stMuted, nil)...)
	if !b.open && r.to < len(b.review) {
		cut := deco("…", stMuted)
		cut.ref = ref{panel: 'B', kind: refTargetCut, item: i, line: 2, off: r.to, n: len(b.review) - r.to}
		l = append(l, cut)
	}
	return grounded(l, b.bw+blockIndent)
}

// errRow is a row of the failure's error, the entry's line 1, in red.
func (b *cmdBlock) errRow(i int, r rowRange) line {
	l := append(spaces(blockIndent), rowLine(b.err, r, ref{panel: 'B', item: i, line: 1}, stFailed, nil)...)
	if !b.open && r.to < len(b.err) {
		l = append(l, deco("…", stFailed))
	}
	return grounded(l, b.bw+blockIndent)
}

// cmdColours are a command's style runs, its colours once the application
// made them (plan 72, D8, P3), on base elsewhere; none, the command plain,
// before then, on 16 colours, or when Chroma's tokens did not rejoin.
func cmdColours(ui UI, f *cmdFacts, base *lipgloss.Style) []styleRun {
	if sixteen {
		return nil
	}
	sp, _ := ui.Colors.colours(f.want())
	return cmdRuns(f.c.text, sp, base)
}

// cmdCell is the drawn unit at byte i of a command: as cellAt, but a format
// character that could make it read otherwise than it runs drawn as
// <U+XXXX>, as a request's text is (plan 48, design D6).
func cmdCell(text string, i int) cell {
	if r, n := utf8.DecodeRuneInString(text[i:]); hiddenFormat(r) {
		return cell{fmt.Sprintf("<U+%04X>", r), len("<U+0000>"), i, n, true, false}
	}
	return cellAt(text, i)
}

// cmdCellSpan is the bytes and the width of cmdCell(text, i).
func cmdCellSpan(text string, i int) (int, int) {
	if r, n := utf8.DecodeRuneInString(text[i:]); hiddenFormat(r) {
		return n, len("<U+0000>")
	}
	return cellSpan(text, i)
}

// cmdCells are the cells of a command's bytes [a, b).
func cmdCells(text string, a, b int) []cell {
	out := make([]cell, 0, b-a)
	for i := a; i < b; {
		c := cmdCell(text, i)
		if i+c.n > b { // a cluster across the row's end: the row ends it
			c.text, c.n = text[i:b], b-i
		}
		out = append(out, c)
		i += c.n
	}
	return out
}

// CommandFold is the fold of entry n's command block (1-based), zone B's
// conversation w cells wide: its key in UI.Unfolded and whether it is
// unfolded; ok only for a block with something to fold (plan 72, K2).
func CommandFold(s *state.State, ui UI, w, n int) (key string, open, ok bool) {
	if n < 1 || n > len(s.Conversation) || s.Conversation[n-1].Role != "tool" {
		return "", false, false
	}
	lk, _ := lookAll(s, ui)
	k := lk[n-1]
	if k.hidden {
		return "", false, false
	}
	e := s.Conversation[n-1]
	b := blockOf(s, ui, &e, cardContent(k.card, w), k)
	if b == nil || !b.mark {
		return "", false, false
	}
	return CommandKey(b.tc.ID), b.open, true
}

// CommandSent is the command the full view shows, as its CLI sent it:
// Claude's command, Codex's invocation, byte for byte (plan 72, C3); ok
// false when the view shows no command.
func CommandSent(s *state.State, ui UI) (string, bool) {
	if f := viewCommand(s, ui); f != nil {
		return f.c.sent, true
	}
	return "", false
}

// viewCommand is the command of the full view shown, a request's or a
// call's; nil for none.
func viewCommand(s *state.State, ui UI) *cmdFacts {
	switch {
	case ui.View != "":
		if ap := s.Approvals[ui.View]; ap != nil {
			return ui.Memo.command(ap.Tool, ap.Input)
		}
	case ui.ViewEntry > 0 && ui.ViewEntry <= len(s.Conversation):
		if e := s.Conversation[ui.ViewEntry-1]; e.Role == "tool" {
			if tc := s.Tools[e.MessageID]; tc != nil {
				return ui.Memo.command(tc.Name, tc.Input)
			}
		}
	}
	return nil
}

// CommandWants are the colours to make for the commands of the orchestrator's
// calls from s.ToolOrder[tools] on and of the requests from s.Queue[asks]
// on: the application asks for each command as it comes (plan 72, P3).
func CommandWants(s *state.State, tools, asks int) []CmdWant {
	var out []CmdWant
	add := func(name string, in json.RawMessage) {
		if !commandTool(name) {
			return
		}
		if c := commandOf(name, readCmd(in)); c.ok && c.lexer != "" {
			out = append(out, CmdWant{c.lexer, c.text})
		}
	}
	for _, id := range s.ToolOrder[min(max(tools, 0), len(s.ToolOrder)):] {
		if tc := s.Tools[id]; tc != nil && tc.Agent == "" {
			add(tc.Name, tc.Input)
		}
	}
	for _, id := range s.Queue[min(max(asks, 0), len(s.Queue)):] {
		if ap := s.Approvals[id]; ap != nil {
			add(ap.Tool, ap.Input)
		}
	}
	return out
}

// copyKey is a command's full view's key hint for C3's copy.
const copyKey = "y copy the command · "

// cwdShown is the working folder a request names, when it is not the
// project's: "" for the project folder or none (plan 72, D7).
func cwdShown(s *state.State, cwd string) string {
	if cwd == "" {
		return ""
	}
	p := s.Context.Project
	if p != "" && strings.EqualFold(filepath.Clean(cwd), filepath.Clean(p)) {
		return ""
	}
	return cwd
}
