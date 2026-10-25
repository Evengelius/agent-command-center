package render

import (
	"fmt"
	"path/filepath"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/Evengelius/agent-command-center/internal/state"
)

// Zone B's messages (plan 20; brief amendments 16 and 17, the owner's
// templates and the image of 2026-10-20): each message is a block, a narrow
// bar of its colour on every row, its header on its own row ("YOU  14:29",
// "CLAUDE  Opus 5.5  14:29"), its body under it from the same column, on
// the zone's own dark background; a thin rule between two messages, a blank
// row on each side. The result and a pending approval are boxes, the tool
// rows on their ground. A frame draws the block around a row of content, so
// an entry counts as many rows however its block is drawn.

// cardKind is the block an entry is drawn in.
type cardKind uint8

const (
	cardNone   cardKind = iota // no block: a note before any exchange, an entry after a result
	cardYou                    // the owner's prompt, violet
	cardAI                     // the CLI's part of an exchange, blue
	cardResult                 // the exchange's result, in its green box
)

// frame draws a message's block around one of its rows of content.
type frame struct {
	kind cardKind
	w    int // zone B's inner width: the row's whole width
}

// cardContent is the width of a card's content in a row w wide: the bar
// and the margin before it, a cell after it; inside the result's box, its
// edges and their padding; without a card, a margin of two.
func cardContent(kind cardKind, w int) int {
	switch kind {
	case cardResult:
		return w - 4
	case cardNone:
		return w - 2
	}
	return w - 3
}

// wrap is content in its block: the bar, the margin (the entry cursor's ▸
// when cursor), the content, a cell to the edge; no ground behind it (the
// owner's amendment 17: no tinted bands behind messages).
func (f *frame) wrap(content line, cursor bool) line {
	cw := cardContent(f.kind, f.w)
	margin := deco(" ", stNone)
	if cursor {
		margin = deco("▸", stAttentionBold)
	}
	switch f.kind {
	case cardNone:
		return append(line{deco(" ", stNone), margin}, content.fit(cw)...)
	case cardResult:
		l := append(line{deco("│", stDone), margin}, content.fit(cw)...)
		return append(l, deco(" ", stNone), deco("│", stDone))
	}
	bar := stYou
	if f.kind == cardAI {
		bar = stSpeaker
	}
	l := append(line{deco("▎", bar), margin}, content.fit(cw)...)
	return append(l, deco(" ", stNone))
}

// messageRule is the thin rule between two messages, w cells wide, in the
// frames' colour: decoration, so a selection across it copies nothing of it.
func messageRule(w int) line {
	return decoLine(strings.Repeat("─", max(w, 0)), stFrame)
}

// spaces is n cells of decoration.
func spaces(n int) line {
	if n <= 0 {
		return nil
	}
	return line{deco(strings.Repeat(" ", n), stNone)}
}

// youLabel is "YOU  01:54", violet (plan 20).
func youLabel(e state.Entry) line {
	l := line{deco("YOU", stYou)}
	if c := clock(e.At); c != "" {
		l = append(l, deco("  "+c, stGrey))
	}
	return l
}

// speakerLabel labels the CLI's part of an exchange: "CLAUDE  Haiku 5.5
// 01:54", the model as plan 19 names it and the time of its first entry,
// both muted metadata (amendment 17).
func speakerLabel(s *state.State, e state.Entry) line {
	l := line{deco(providerLabel(s), stSpeaker)}
	if m := model(s); m != "" {
		l = append(l, deco("  "+modelName(m), stMuted))
	}
	if c := clock(e.At); c != "" {
		l = append(l, deco("  "+c, stGrey))
	}
	return l
}

// resultLabel is "RESULT  01:54", green.
func resultLabel(e state.Entry) line {
	l := line{deco("RESULT", stResult)}
	if c := clock(e.At); c != "" {
		l = append(l, deco("  "+c, stGrey))
	}
	return l
}

// resultEdge is the result box's top or bottom edge, w wide.
func resultEdge(w int, top bool) line {
	if top {
		return decoLine("╭"+strings.Repeat("─", max(w-2, 0))+"╮", stDone)
	}
	return decoLine("╰"+strings.Repeat("─", max(w-2, 0))+"╯", stDone)
}

// cardText adds entry i's lines as rows of a card's content width wide:
// the first behind lead0 (a label, or the text's column), the others behind
// lead, of the same width.
func cardText(out []crow, wc *WrapCache, fr *frame, i int, lines []string, width int, lead0, lead line, st *lipgloss.Style, markdown bool) []crow {
	p := &rowStyle{lead0: lead0, lead: lead, at: ref{panel: 'B', item: i}, st: st}
	if markdown && markdownOn {
		p.md = mdLines(lines)
	}
	tw := max(width-lead.width(), 1)
	for j, l := range lines {
		for k, r := range wc.ranges(l, tw, tw) {
			out = append(out, crow{src: l, r: r, line: j, first: k == 0, p: p, fr: fr})
		}
	}
	return out
}

// entryRows adds the rows of conversation entry i, drawn with its look k in
// its block (plan 20, amendment 17): what separates it from the entry
// before, its rows, and a blank row after the conversation's last.
// Content comes from entryLines; everything else on these rows is
// decoration: the labels, the bars, the rules, the symbols, the folds.
func entryRows(s *state.State, ui UI, out []crow, i int, e state.Entry, w int, k look) []crow {
	if k.hidden && k.fold == 0 {
		return out
	}
	fr := &frame{kind: k.card, w: w}
	if k.top { // a blank row under the zone's border
		out = append(out, crow{l: line{}})
	}
	switch {
	case k.sep: // a new message: the rule, a blank row on each side
		out = append(out, crow{l: line{}}, crow{l: messageRule(w), sep: true}, crow{l: line{}})
	case k.gap && k.inner: // between two parts of a message: the block's row
		out = append(out, crow{l: line{}, fr: fr})
	case k.gap:
		out = append(out, crow{l: line{}})
	}
	out = entryBody(s, ui, out, i, e, w, k, fr)
	if k.tail { // the conversation's last: a blank row before the input's rule
		out = append(out, crow{l: line{}})
	}
	return out
}

// entryBody is entry i's own rows in its block fr: a header on its own row,
// then the body under it from the block's content column, for every
// message (amendment 17: never beside its label).
func entryBody(s *state.State, ui UI, out []crow, i int, e state.Entry, w int, k look, fr *frame) []crow {
	wc := ui.Wrap
	// Its row shows the entry cursor, its rows as many as without it.
	cursor := ui.Card == "" && (ui.Focus == "B" || ui.Focus == "")
	focused := cursor && ui.Entry == i+1
	cw := cardContent(k.card, w)
	if k.result {
		out = append(out, crow{l: resultEdge(w, true)})
	}
	if k.head {
		out = append(out, crow{l: speakerLabel(s, e), fr: fr})
	}
	if k.fold > 0 {
		// A folded run's row takes the cursor of any call it folds.
		on := cursor && k.hidden && ui.Entry > i && ui.Entry <= k.end
		out = append(out, crow{fn: func() line { return foldRow(s, i, k, cw, on) }, fold: k.run, fr: fr, cursor: on})
		if k.hidden {
			return out
		}
	}
	body := len(out) // the entry's own rows start here, below its labels and fold row
	switch e.Role {
	case "user":
		ls := entryLines(s, e)
		out = append(out, crow{l: youLabel(e), fr: fr})
		shown := ls
		if len(ls) > foldOver {
			shown = ls[:foldShow]
		}
		out = cardText(out, wc, fr, i, shown, cw, nil, nil, stBright, false)
		if len(ls) > foldOver {
			more := fmt.Sprintf("… %s more lines", grouped(len(ls)-foldShow))
			row := decoLine(more, stGrey)
			if focused {
				row = spread(decoLine(more, stAttentionBold), decoLine(entryHint, stAttention), cw)
			}
			out = append(out, crow{l: row, fr: fr, cursor: focused})
		}
		// The images the message carried, one a row under its text (plan 31).
		for _, img := range e.Images {
			out = append(out, crow{l: imageRow(img, cw), fr: fr})
		}
	case "assistant":
		if k.result {
			cw = cardContent(cardResult, w)
			out = append(out, crow{l: resultLabel(e), fr: fr})
		}
		// The answer, Markdown styled; a thinking block it had is the thinking
		// entry's (plan 22): a message with nothing else is not drawn (looks).
		out = cardText(out, wc, fr, i, entryLines(s, e), cw, nil, nil, stText, true)
		if e.Streaming {
			if last := out[len(out)-1].content(); last.width() < cw {
				out[len(out)-1] = crow{l: append(last, deco("▌", stText)), fr: fr}
			} else {
				out = append(out, crow{l: line{deco("▌", stText)}, fr: fr})
			}
		}
		if k.result {
			out = append(out, crow{l: resultEdge(w, false)})
		}
	case "thinking":
		out = thoughtRows(s, ui, out, i, e, w, k, fr, focused)
	case "tool":
		// Drawn only when shown: a long session has thousands (test 11).
		out = append(out, crow{fn: func() line { return callRow(s, ui, i, e, cw, k, focused) }, fr: fr, cursor: focused})
	case "approval":
		for _, l := range approvalCard(s, ui, i, e, cw, k) {
			out = append(out, crow{l: l, fr: fr})
		}
	case "review":
		// An auto-review's entry, the reviewer's own, observed; a verdict other
		// than approved is shown as a failure (plan 12, item 2).
		st := stProgress
		switch e.Status {
		case "inProgress", "approved":
		default:
			st = stFailed
		}
		start := len(out)
		out = cardText(out, wc, fr, i, entryLines(s, e), cw, line{deco("◈ ", st)}, line{deco("  ", st)}, st, false)
		out[start].cursor = focused
	case "error":
		// The error's first line, as received; Enter on D shows it
		// whole (ui-design.md, States: Error).
		out = append(out, crow{fn: func() line {
			l := append(line{deco("✗ turn failed: ", stFailedBold)}, oneRow(i, e, stFailed)...)
			if focused {
				return spread(l, decoLine(entryHint, stAttention), cw)
			}
			return l.fit(cw)
		}, fr: fr, cursor: focused})
	case "dashboard":
		// The dashboard's own note, grey, never the CLI's (AGENTS.md: shown
		// distinctly); every row under the first's text.
		out = cardText(out, wc, fr, i, entryLines(s, e), cw, line{deco("· ", stGrey)}, line{deco("  ", stGrey)}, stGrey, false)
	default:
		out = cardText(out, wc, fr, i, entryLines(s, e), cw, line{deco("· ", stProgress)}, line{deco("  ", stProgress)}, stProgress, false)
	}
	// Its own rows say whose they are: a double click on one opens the entry
	// as Enter does (plan 42, design D4).
	for j := body; j < len(out); j++ {
		out[j].entry = i + 1
	}
	return out
}

// toolLeft is a tool row's left part: a cell, the status symbol, a cell and
// the name (plan 20, Template 1's columns).
func toolLeft(sym span, name line) line {
	return append(line{deco(" ", stNone), sym, deco(" ", stNone)}, name...)
}

// toolRight is a tool row's right part: the outcome, then the origin in its
// column, as wide as "observed" for the outcomes to align, or a cell alone.
func toolRight(outcome line, origin string) line {
	if origin != "" {
		return append(append(line{}, outcome...), deco(fmt.Sprintf("  %8s ", origin), stGrey))
	}
	return append(append(line{}, outcome...), deco(" ", stNone))
}

// toolRoom is the room a tool row leaves its right part, in a row w cells wide.
// The name stays whole, so a right part wider than this is cut. toolLine does
// it; a phase's summary row asks the same question of the same parts (toolLeft,
// toolRight, toolRoom) to drop its origin whole instead, so the rule is in one
// place (plan 22, correction C9).
func toolRoom(left line, w int) int { return w - left.width() - 2 }

// toolLine lays out a tool row w cells wide on the tool rows' ground: the
// status symbol, the name part padded to nameW, the detail cut to the room
// left, and at the right the outcome and the origin (plan 20, Template 1's
// columns).
func toolLine(sym span, name, fail line, nameW int, detail line, path bool, outcome line, origin string, w int) line {
	left, right := toolLeft(sym, name), toolRight(outcome, origin)
	// The name stays whole: a right part too long for the row (the entry
	// cursor's hint at 80 columns) is cut instead.
	if room := toolRoom(left, w); right.width() > room {
		right = right.fit(max(room, 0))
	}
	// Then the target, in its column, and a failure's error after it ("· exit
	// status 1: …"), or after the name when the call has no target. The
	// error keeps half the room, or its whole width when less, and both are
	// cut to the room the outcome leaves, the verdict never pushed off (plan
	// 13, item 1): a failed row names what failed, as Template 1's does (the
	// owner's live review, 2026-10-20).
	if detail != nil {
		left = append(left, spaces(nameW-name.width())...)
		room := w - left.width() - right.width() - 4
		if fail != nil {
			room -= min(fail.width(), max(room/2, 0))
		}
		if room >= 8 || fail == nil && room >= 4 {
			left = append(left, deco("  ", stNone))
			left = append(left, fitDetail(detail, path, room)...)
		}
	}
	if room := w - left.width() - right.width() - 2; fail != nil && room >= 4 {
		left = append(left, fail.fit(min(fail.width(), room))...)
	}
	return grounded(spread(left, right, w), w)
}

// fitDetail cuts a detail to w cells: a path loses its left end, so its
// file name stays; anything else its right end, an ellipsis in its place.
func fitDetail(d line, path bool, w int) line {
	if d.width() <= w {
		return d
	}
	if path && len(d) == 1 {
		x := d[0]
		x.text = elidePath(x.text, w)
		return line{x}
	}
	return d.fit(w) // its last cell an ellipsis
}

// elidePath is elide cut at a path's element, so what is kept starts at a
// separator: "…\Local\Temp\demo\src\f00.go", never "…rs\…\AppData…" (the
// owner's live review, 2026-10-20); when even the file name does not fit,
// as elide.
func elidePath(p string, w int) string {
	p = hideUser(p)
	if textWidth(p) <= w {
		return p
	}
	for i, r := range p {
		if r != '\\' && r != '/' || i == 0 {
			continue
		}
		rest := p[i:]
		if strings.HasPrefix(rest[1:], "…") { // the hidden user folder: keep what follows it
			continue
		}
		if textWidth(rest)+1 <= w {
			return "…" + rest
		}
	}
	return elide(p, w)
}

// callRow is a call's row in zone B (plan 20): its status, its name (the
// entry's content) in light blue, its target in a readable muted text or
// its error in red, and at the right how it ended, with its time, and its
// origin; the entry cursor's hint replaces them on its row, with ← when
// the call is in an expanded run.
func callRow(s *state.State, ui UI, i int, e state.Entry, w int, k look, focused bool) line {
	tc := s.Tools[e.MessageID]
	sym := deco("▸", stGrey) // no call recorded under it
	if tc != nil {
		sym = toolSymbol(s, tc, running(s, ui))
	}
	outcome, origin := callOutcome(s, ui, tc), originLabel(e.Origin)
	if focused { // the hint in place of the outcome and the origin
		hint := entryHint
		if _, ok := EntryFile(s, e); ok {
			hint = entryFileHint
		}
		if k.open {
			hint = strings.Replace(hint, "↑↓ entries", "← fold · ↑↓ entries", 1)
		}
		outcome, origin = decoLine(hint, stAttention), ""
	}
	// Every call shows its target; a failed one then its error, "✗ Read
	// …\src\f03.go · File does not exist.", or "✗ Bash · exit status 1: FAIL
	// pkg/a" without a target (plan 5, item 6; plan 6, item 1).
	var fail line
	if f := toolFailure(s, i, e); tc != nil && tc.Status == state.ToolFailed && len(f) > 1 {
		fail = f
	}
	detail, path := callDetail(ui.Memo, tc)
	return toolLine(sym, oneRow(i, e, stToolName), fail, k.nameW, detail, path, outcome, origin, w)
}

// callDetail is a call's target in a readable muted text (plan 20, the
// owner's section 7.2), path saying it is a file path; for an agent, its
// type in blue and its description.
func callDetail(m *Memo, tc *state.ToolCall) (d line, path bool) {
	var t string
	if tc != nil {
		f := m.input(tc.Input, false)
		t, path = f.detail, f.detailPath
	}
	if t == "" {
		return nil, false
	}
	if typ, desc, ok := strings.Cut(t, " · "); ok && agentLaunch(tc) {
		return line{deco(typ, stFoldMark), deco(" · "+desc, stMuted)}, false
	}
	return line{deco(t, stMuted)}, path
}

// agentLaunch says a call starts a subagent: its input names an agent type.
func agentLaunch(tc *state.ToolCall) bool {
	return tc != nil && strings.Contains(string(tc.Input), `"subagent_type"`)
}

// callOutcome is how a call ended, with its time, in that outcome's colour
// (plan 20): waiting for the owner, running and for how long, the
// auto-review's verdict, failed, denied, unfinished, or its end time.
func callOutcome(s *state.State, ui UI, tc *state.ToolCall) line {
	switch {
	case tc == nil:
		return nil
	case pending(s, tc.ID):
		return decoLine("waiting for you", stAttention)
	case tc.Reviewed != "" && (tc.Status == state.ToolDenied || strings.HasPrefix(tc.Reviewed, "denied")):
		return decoLine(tc.Reviewed+endTime(tc), stFailed)
	case tc.Reviewed != "": // decided before it ran: shown while it runs too (plan 12)
		return decoLine(tc.Reviewed+endTime(tc), stDone)
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

// endTime is " 01:54:35", a call's end, or "" while it runs.
func endTime(tc *state.ToolCall) string {
	if tc.EndedAt.IsZero() {
		return ""
	}
	return " " + clockSec(tc.EndedAt)
}

// foldRow is the row of a run of calls that folds (plan 19), w cells wide on
// the tool rows' ground: "▸ 9 earlier tool events · 1 failed · Expand", how
// many of them failed, run or wait for the owner, in their colours; on, the
// entry cursor is on it.
func foldRow(s *state.State, i int, k look, w int, on bool) line {
	var failed, runs, waits int
	for _, e := range s.Conversation[i:k.end] {
		tc := s.Tools[e.MessageID]
		switch {
		case e.Role != "tool" || tc == nil:
		case pending(s, tc.ID):
			waits++
		case tc.Status == state.ToolFailed || tc.Status == state.ToolDenied:
			failed++
		case !tc.Status.Final():
			runs++
		}
	}
	mark, act := "▸ ", " · Expand"
	if k.open {
		mark, act = "▾ ", " · Fold"
	}
	l := line{deco(" ", stNone), deco(mark, stFoldMark), deco(fmt.Sprintf("%d earlier tool events", k.fold), stMuted)}
	for _, c := range []struct {
		n    int
		what string
		st   *lipgloss.Style
	}{{failed, "failed", stFailed}, {runs, "running", stProgress}, {waits, "waiting for you", stAttention}} {
		if c.n > 0 {
			l = append(l, deco(" · ", stGrey), deco(fmt.Sprintf("%d %s", c.n, c.what), c.st))
		}
	}
	var right line
	if on {
		right = decoLine(" Enter expands · ↑↓ entries · Esc back ", stAttention)
	} else {
		l = append(l, deco(act, stGrey))
	}
	return grounded(spread(l, right, w), w)
}

// grounded is l on the tool rows' ground, w cells wide (plan 19).
func grounded(l line, w int) line {
	l = l.fit(w)
	out := make(line, len(l))
	for j, x := range l {
		x.st = tint(x.st)
		out[j] = x
	}
	return out
}

// baseOf is a path's last element, either separator.
func baseOf(p string) string {
	return filepath.Base(strings.ReplaceAll(p, `\`, "/"))
}
