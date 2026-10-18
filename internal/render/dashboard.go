package render

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"charm.land/lipgloss/v2"

	"github.com/Evengelius/agent-command-center/internal/event"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// Layout widths (ui-design.md, Layouts at three sizes; ASM-12).
const (
	standardWidth = 100
	wideWidth     = 160
	zoneAStandard = 34
	zoneAWide     = 44
	listWide      = 74
)

// dashboard draws the working screen at the layout the width selects.
func dashboard(s *state.State, ui UI, sc *Screen) []line {
	w, h := ui.Width, ui.Height
	wide := w >= wideWidth
	var extra line
	if wide {
		if m := model(s); m != "" {
			extra = plain("■ "+m, modelStyle(m))
		}
	}
	ls := []line{titleBar(s, w, badge(s), extra), projectBar(s, ui)}

	if ui.Zoom && strings.Contains("ACD", ui.Focus) && ui.Focus != "" {
		// The focused zone over the middle and bottom (ui-design.md,
		// Scrolling, collapsing, full screen); the bars stay.
		zh := h - len(ls) - 1
		keys := "Enter details"
		switch ui.Focus {
		case "A":
			keys = "↑↓ move · ←→ fold · Enter inspect"
			ls = append(ls, zoneA(s, ui, w, zh, sc)...)
		case "C":
			keys = "↑↓ select · Enter inspect"
			if wide {
				ls = append(ls, beside(zoneC(s, ui, listWide, zh, sc), preview(s, ui, w-listWide, zh, sc))...)
			} else {
				ls = append(ls, zoneC(s, ui, w, zh, sc)...)
			}
		case "D":
			ls = append(ls, zoneD(s, ui, w, zh, zh-2)...)
		}
		return append(ls, keyHints(w, ui, "z or Esc restore · "+keys+" · F6 zone · F1 keys"))
	}

	if w < standardWidth {
		ls = append(ls, zoneALine(s, w))
		dh, ch := 3, 5
		bh := h - len(ls) - ch - dh - 1
		if ui.Focus == "A" { // the tree takes B's and C's place until F6 leaves it
			ls = append(ls, zoneA(s, ui, w, bh+ch, sc)...)
			ls = append(ls, zoneD(s, ui, w, dh, 1)...)
			return append(ls, keyHints(w, ui, "↑↓ move · ←→ fold · Enter inspect · F6 zone · Esc back to the input"))
		}
		sc.Conv.Y = len(ls) + 1
		ls = append(ls, zoneB(s, ui, w, bh, 3, sc)...)
		ls = append(ls, zoneC(s, ui, w, ch, sc)...)
		ls = append(ls, zoneD(s, ui, w, dh, 1)...)
		hints := "Enter send · Ctrl+J newline · F6 zone · F2 approve · F1 keys"
		if ui.Entry > 0 && ui.Focus == "B" && ui.Card == "" {
			hints = entryKeys
		}
		return append(ls, keyHints(w, ui, hints))
	}

	aw, dh, rows := zoneAStandard, 4, 2
	if wide {
		aw, dh, rows = zoneAWide, 6, 3
	}
	mid := h - len(ls) - dh - 1
	bh := (mid*3 + 2) / 5
	rw := w - aw
	right := zoneB(s, ui, rw, bh, 3+2*boolInt(wide), sc)
	sc.Conv.Y = len(ls) + 1
	if wide {
		right = append(right, beside(zoneC(s, ui, listWide, mid-bh, sc), preview(s, ui, rw-listWide, mid-bh, sc))...)
	} else {
		right = append(right, zoneC(s, ui, rw, mid-bh, sc)...)
	}
	ls = append(ls, beside(zoneA(s, ui, aw, mid, sc), right)...)
	ls = append(ls, zoneD(s, ui, w, dh, rows)...)
	hints := "Enter send · Ctrl+J newline · Ctrl+C interrupt · Ctrl+G editor · F6 zone · F2 approvals · F1 keys · Ctrl+Q quit"
	if wide {
		hints = "Enter send · Ctrl+J newline · Ctrl+C copy or interrupt · Ctrl+G editor · F6 / Shift+F6 zone · F2 approvals · F4 Manual/Auto · z zoom · F12 mouse capture · F1 keys · Ctrl+Q quit"
	}
	if ui.Entry > 0 && ui.Focus == "B" && ui.Card == "" {
		hints = entryKeys
	}
	return append(ls, keyHints(w, ui, hints))
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// keyHints is the bottom line; a toast takes its place for a moment.
func keyHints(w int, ui UI, text string) line {
	if ui.Toast != "" {
		return plain(" "+ui.Toast, stProgress).fit(w)
	}
	if ui.MouseOff {
		return plain(" mouse capture off: the terminal selects (whole screen, borders included) · F12 back on", stProgress).fit(w)
	}
	return plain(" "+text, stGrey).fit(w)
}

// badge is "● LIVE" while the session runs, else what it is.
func badge(s *state.State) line {
	switch s.Session.Phase {
	case state.Ready:
		return plain("● LIVE ", stDone)
	case state.Spawning, state.Handshaking:
		return plain("◉ STARTING ", stProgress)
	case state.Closing:
		return plain("◉ CLOSING ", stProgress)
	case state.Crashed:
		return plain("✗ ENDED ", stFailed)
	}
	return plain("○ CLOSED ", stGrey)
}

func model(s *state.State) string {
	if s.Session.Model != "" {
		return s.Session.Model
	}
	if v := s.Metrics["model"]; v.Present {
		return v.Text
	}
	return ""
}

// projectBar is the line under the title (ui-design.md, Project bar).
func projectBar(s *state.State, ui UI) line {
	left := line{sp(" ▣ ", stTitle), sp(base(s.Context.Project), stBold)}
	branch := ""
	if ui.Branch != "" {
		branch = ui.Branch
	}
	withBranch := func() line {
		if branch == "" {
			return left
		}
		return append(append(line{}, left...), sp("   branch ", stGrey), sp(branch, stText))
	}
	name := s.Context.ProviderName
	if s.Session.Version != "" {
		name += " " + s.Session.Version
	}
	// Codex's own settings, dropped from the right one when the bar is too
	// narrow; the approvals mode never is (plan 12, item 2).
	var extras []string
	if s.Context.Provider == "codex" {
		for _, x := range []struct{ k, v string }{{"", s.Session.Model}, {"sandbox ", s.Session.Sandbox}, {"policy ", s.Session.ApprovalPolicy}} {
			if x.v != "" {
				extras = append(extras, x.k+x.v)
			}
		}
	}
	short := false // at the narrowest, the approvals mode alone
	right := func() line {
		if short {
			return approvalsSpans(s)
		}
		r := plain(name+strings.Join(append([]string{""}, extras...), " · ")+" · ", stText)
		return append(r, approvalsSpans(s)...)
	}
	for len(extras) > 0 && withBranch().width()+right().width() >= ui.Width {
		extras = extras[:len(extras)-1]
	}
	if withBranch().width()+right().width() >= ui.Width {
		branch = ""
	}
	if withBranch().width()+right().width() >= ui.Width {
		short = true
	}
	left = withBranch()
	r := right()
	if ui.Width >= standardWidth {
		room := ui.Width - left.width() - r.width() - 6
		if room > 10 {
			left = append(left, sp("   "+elide(s.Context.Project, room), stGrey))
		}
	}
	return spread(left, r, ui.Width)
}

// approvalsSpans is the project bar's approvals mode: Auto in the attention
// colour, a change asked for and not yet in force after an arrow (plan 12,
// item 2). Codex applies it with its next turn; Claude Code answers at once.
func approvalsSpans(s *state.State) line {
	cur := s.Mode()
	l := line{sp("approvals ", stGrey), sp(state.ModeLabel(cur), modeStyle(cur))}
	if req := s.Session.ApprovalRequested; req != "" {
		suffix := " …"
		if s.CLI() == "codex" {
			suffix = " (next turn)"
		}
		l = append(l, sp(" → ", stGrey), sp(state.ModeLabel(req), modeStyle(req)), sp(suffix, stGrey))
	}
	return append(l, sp(" ", stText))
}

func modeStyle(mode string) *lipgloss.Style {
	if mode == state.ModeAuto {
		return stAttentionBold
	}
	return stText
}

// status symbols and their styles (ui-design.md, Colours and symbols).
func agentSymbol(st state.AgentStatus) span {
	switch st {
	case state.AgentRunning, state.AgentAnnounced:
		return deco("◉", stProgress)
	case state.AgentWaiting:
		return deco("!", stAttention)
	case state.AgentCompleted, state.AgentEnded:
		return deco("✓", stDone)
	case state.AgentFailed:
		return deco("✗", stFailed)
	case state.AgentStopped:
		return deco("○", stGrey)
	}
	return deco("○", stGrey)
}

func turnSymbol(s *state.State) span {
	switch s.Turn.Phase {
	case state.Running, state.Submitted, state.Interrupting:
		return deco("◉", stProgress)
	case state.WaitingApproval:
		return deco("!", stAttention)
	}
	switch s.Turn.Outcome {
	case "failed":
		return deco("✗", stFailed)
	case "interrupted":
		return deco("○", stGrey)
	}
	return deco("✓", stDone)
}

// zoneALine is zone A at narrow width: the current stage and its agents in
// one line; F6 opens the tree.
func zoneALine(s *state.State, w int) line {
	l := line{sp(" A ", stTitle)}
	if n := len(s.Stages); n == 0 {
		l = append(l, sp("no turn yet", stGrey))
	} else {
		st := &s.Stages[n-1]
		sym, _ := stageStatus(s, st)
		l = append(l, sym, sp(fmt.Sprintf(" Turn %d", st.N), stText))
		for _, it := range stageItems(s, st.N) {
			if a := it.agent; a != nil {
				l = append(l, sp(" › ", stGrey), agentSymbol(a.Status), sp(" "+a.Label, stText))
			}
		}
	}
	return spread(l, plain("   (inferred)   F6: tree ", stGrey), w)
}

// zoneA is the orchestrator and its stage tree (ui-design.md, Zone A).
func zoneA(s *state.State, ui UI, w, h int, sc *Screen) []line {
	var body []line
	m := model(s)
	if m == "" {
		body = append(body, plain("model N/A ("+naReason(s)+")", stGrey))
	} else {
		body = append(body, line{sp(m, modelStyle(m)), sp(" · orchestrator", stGrey)})
	}
	n := len(s.Stages)
	switch {
	case n == 0:
		body = append(body, plain("no turn yet", stGrey))
	case s.Stages[n-1].Open:
		body = append(body, plain(fmt.Sprintf("turn %d · %s %s", n, turnWord(s), elapsed(s.Stages[n-1].StartedAt, time.Time{}, ui.Now)), stText))
	default:
		_, word := stageStatus(s, &s.Stages[n-1])
		body = append(body, plain(fmt.Sprintf("turn %d · %s", n, word), stText))
	}
	body = append(body, line{})
	if n > 0 && len(s.Stages[n-1].Plan) > 0 {
		body = append(body, line{sp("PLAN  ", stBold), sp("observed (turn/plan/updated)", stGrey)})
	} else {
		body = append(body, line{sp("STAGES  ", stBold), sp("inferred from activity", stGrey)})
	}
	rows := treeRows(s, ui, w-4)
	if len(rows) == 0 {
		rows = []treeRow{{l: plain("○ (stages appear as turns run)", stGrey)}}
	}
	room := max(h-3-len(body), 0)
	top := paneTop(ui.Tree, len(rows), room, treeTop(rows, ui.ACursor, room))
	sc.Tree = PaneView{Rows: len(rows), Room: room, Top: top}
	for _, r := range rows {
		sc.Tree.IDs = append(sc.Tree.IDs, r.node.ID)
	}
	for _, r := range rows[top:min(top+room, len(rows))] {
		if ui.Focus == "A" && r.node.ID != "" && r.node.ID == ui.ACursor {
			r.l = cursorLine(r.l.fit(w-4), w-4)
		}
		body = append(body, r.l)
	}
	for len(body) < h-3 {
		body = append(body, line{})
	}
	hint := decoLine("↑↓ move ←→ fold Enter inspect", stGrey)
	if ui.Tree.Hold && top < len(rows)-room { // held away from the tail
		hint = heldHint(top, len(rows)-top-room, "End follows")
	}
	body = append(body[:max(h-3, 0)], hint)
	return box('A', zoneTitle("A · ORCHESTRATOR", ui.Focus == "A"), nil, body, w, h, borderFor(ui.Focus == "A"))
}

func turnWord(s *state.State) string {
	switch s.Turn.Phase {
	case state.Idle:
		if s.Turn.Outcome != "" {
			return s.Turn.Outcome
		}
		return "idle"
	case state.WaitingApproval:
		return "waiting for approval"
	}
	return strings.ToLower(string(s.Turn.Phase))
}

// children are the agents whose parent is key, in order of appearance.
func children(s *state.State, parent string) []string {
	var out []string
	for _, k := range s.Order {
		if a := s.Agents[k]; a != nil && a.Parent == parent {
			out = append(out, k)
		}
	}
	return out
}

func zoneTitle(t string, focused bool) line {
	if focused {
		return plain(t, stTitle)
	}
	return plain(t, stBold)
}

func borderFor(focused bool) *lipgloss.Style {
	if focused {
		return stTitle
	}
	return stFrame
}

// zoneB is the conversation, its notices and the editor. The conversation
// follows its end unless the owner scrolled up (ui-design.md, Zone B, Follow
// mode); sc, when given, records what was shown for scrolling and the mouse.
func zoneB(s *state.State, ui UI, w, h, editorRows int, sc *Screen) []line {
	inner := w - 4
	right := turnBadge(s, ui)
	// The editor grows up to half of B, then scrolls (ui-design.md, Input
	// editor, Size); its rule and status line count in that half.
	editor := editorLines(s, ui, inner, editorRows, (h-2)/2-2)
	room := max(h-2-len(editor), 0)
	conv, rows, top, starts := conversation(s, ui, inner, room)
	var shown []line // only the rows shown are drawn
	for _, c := range conv {
		shown = append(shown, c.draw())
	}
	if len(shown) < room {
		shown = append(shown, blank(0, room-len(shown))...)
	}
	if n := len(s.Conversation) - ui.ConvSeen; ui.ConvHold && n > 0 && room > 0 {
		// The badge takes the right end of the last row only; the row's start
		// stays, cut before it (plan 6, item 2).
		shown[room-1] = spread(trimTail(shown[room-1]), decoLine(fmt.Sprintf("↓ %d new", n), stProgress), inner)
	}
	if sc != nil {
		sc.Conv = ConvView{Rows: rows, Room: room, Top: top, Y: sc.Conv.Y, Held: ui.ConvHold, Starts: starts}
	}
	body := append(shown, editor...)
	focused := ui.Focus == "B" || ui.Focus == ""
	return box('B', zoneTitle("B · CONVERSATION", focused), right, body, w, h, borderFor(focused))
}

// conversation is the orchestrator's record, then the session's state: the
// rows in view, at most room from top, and the count of all rows. It follows
// the end unless the owner scrolled up. Only the entries in view are drawn;
// the others are counted, from the cache once finished (test 11).
func conversation(s *state.State, ui UI, w, room int) (view []crow, rows, top int, starts []int) {
	starts = make([]int, len(s.Conversation)+1)
	for i, e := range s.Conversation {
		rows += ui.Wrap.rowCount(i, e, w, func() int { return len(entryRows(s, ui, nil, i, e, w)) })
		starts[i+1] = rows
	}
	counts := func(i int) int { return starts[i+1] - starts[i] }
	tail := sessionState(s, ui, w)
	rows += len(tail)
	top = max(rows-room, 0)
	if ui.ConvHold {
		top = min(max(ui.ConvTop, 0), top)
	}
	at, end := 0, top+room
	for i, e := range s.Conversation {
		if at >= end {
			break
		}
		if at+counts(i) > top {
			rs := entryRows(s, ui, nil, i, e, w)
			view = append(view, rs[min(max(top-at, 0), len(rs)):min(end-at, len(rs))]...)
		}
		at += counts(i)
	}
	for _, l := range tail {
		if at >= top && at < end {
			view = append(view, crow{l: l})
		}
		at++
	}
	return view, rows, top, starts
}

// entryRows adds the rows of conversation entry i. Content comes from
// entryLines; everything else on these rows is decoration.
func entryRows(s *state.State, ui UI, out []crow, i int, e state.Entry, w int) []crow {
	wc := ui.Wrap
	who := providerLabel(s)
	// Its row shows the entry cursor, its rows as many as without it.
	focused := ui.Entry == i+1 && ui.Card == "" && (ui.Focus == "B" || ui.Focus == "")
	switch e.Role {
	case "user":
		out = drawn(out, line{deco("YOU", stAttentionBold), deco("  "+clock(e.At), stGrey)})
		ls := entryLines(s, e)
		if len(ls) > foldOver {
			out = contentRows(out, wc, i, ls[:foldShow], w, stText, false)
			more := fmt.Sprintf("… %s more lines", grouped(len(ls)-foldShow))
			if focused {
				out = drawn(out, spread(decoLine("  ▸ "+more, stAttentionBold), decoLine(entryHint, stAttention), w))
				break
			}
			out = drawn(out, decoLine("  "+more, stGrey))
			break
		}
		out = contentRows(out, wc, i, ls, w, stText, false)
	case "assistant":
		// Thinking marks a message that had a thinking block; its text is
		// the answer. The thinking is folded under it, without a token
		// count the state does not keep (ui-design.md, Markdown, Thinking).
		if e.Text != "" || !e.Thinking {
			head := line{deco(who, stTitle)}
			if m := model(s); m != "" {
				head = append(head, deco("  "+m, modelStyle(m)))
			}
			out = drawn(out, head)
			out = contentRows(out, wc, i, entryLines(s, e), w, stText, true)
			if e.Streaming {
				if last := out[len(out)-1].draw(); last.width() < w {
					out[len(out)-1] = crow{l: append(last, deco("▌", stText))}
				} else {
					out = drawn(out, decoLine("  ▌", stText))
				}
			}
		}
		if e.Thinking {
			out = drawn(out, decoLine("  ∴ thinking", stGrey))
		}
	case "tool":
		// Drawn only when shown: a long session has thousands (test 11). A
		// failed or denied call is marked ✗, with its error (plan 5, item 6).
		out = append(out, crow{fn: func() line {
			fail := toolFailure(s, i, e)
			if focused {
				l := append(append(append(line{deco("  ▸ ", stAttentionBold)}, oneRow(i, e, stText)...), fail...), toolReviewed(s, e)...)
				return spread(l, decoLine(entryHint, stAttention), w)
			}
			lead := deco("  ▸ ", stGrey)
			if fail != nil {
				lead = deco("  ✗ ", stFailed)
			}
			l := append(append(append(line{lead}, oneRow(i, e, stGrey)...), fail...), toolReviewed(s, e)...)
			return spread(l, decoLine(originLabel(e.Origin), stGrey), w)
		}})
	case "review":
		// An auto-review's entry, the reviewer's own, observed; a verdict other
		// than approved is shown as a failure (plan 12, item 2).
		st := stProgress
		switch e.Status {
		case "inProgress", "approved":
		default:
			st = stFailed
		}
		p := &rowStyle{lead0: deco("  ◈ ", st), lead: deco("    ", st), at: ref{panel: 'B', item: i}, st: st}
		for j, l := range entryLines(s, e) {
			out = sourceRows(out, wc, p, j, l, w-4, w-4)
		}
	case "approval":
		out = drawn(out, approvalCard(s, ui, i, e, w)...)
	case "error":
		// The error's first line, as received; Enter on D shows it
		// whole (ui-design.md, States: Error).
		out = append(out, crow{fn: func() line {
			l := append(line{deco("✗ turn failed: ", stFailedBold)}, oneRow(i, e, stFailed)...)
			if focused {
				return spread(l, decoLine(entryHint, stAttention), w)
			}
			return l.fit(w)
		}})
	case "dashboard":
		// The dashboard's own note, grey, never the CLI's (AGENTS.md: shown
		// distinctly); every row under the first's text.
		p := &rowStyle{lead0: deco("  · ", stGrey), lead: deco("    ", stGrey), at: ref{panel: 'B', item: i}, st: stGrey}
		for j, l := range entryLines(s, e) {
			out = sourceRows(out, wc, p, j, l, w-4, w-4)
		}
	default:
		p := &rowStyle{lead0: deco("  · ", stProgress), lead: deco("  ", stProgress), at: ref{panel: 'B', item: i}, st: stProgress}
		for j, l := range entryLines(s, e) {
			out = sourceRows(out, wc, p, j, l, w-4, w-2)
		}
	}
	return out
}

// toolReviewed marks a call the CLI says an auto-review decided: approved or
// denied by it (plan 12, item 2).
func toolReviewed(s *state.State, e state.Entry) line {
	tc := s.Tools[e.MessageID]
	if tc == nil || tc.Reviewed == "" {
		return nil
	}
	if tc.Status == state.ToolDenied || strings.HasPrefix(tc.Reviewed, "denied") {
		return line{deco(" · "+tc.Reviewed, stFailed)}
	}
	return line{deco(" · "+tc.Reviewed, stAttention)}
}

// toolFailure is what follows a failed or denied call's name in B: its error,
// the entry's second line (entryLines), or the status when it wrote none;
// nil for any other call.
func toolFailure(s *state.State, i int, e state.Entry) line {
	tc := s.Tools[e.MessageID]
	switch {
	case tc == nil:
		return nil
	case tc.Status == state.ToolDenied:
		return line{deco(" · denied", stFailed)}
	case tc.Status != state.ToolFailed:
		return nil
	}
	if ls := entryLines(s, e); len(ls) > 1 {
		return append(line{deco(" · ", stGrey)}, wrapSource(ls[1], 1<<30, 1<<30, ref{panel: 'B', item: i, line: 1}, stFailed)[0]...)
	}
	return line{deco(" · failed", stFailed)}
}

// oneRow is a tool or approval entry's first line on one row, marked when
// more lines follow.
func oneRow(i int, e state.Entry, st *lipgloss.Style) line {
	l := wrapSource(entryLines(nil, e)[0], 1<<30, 1<<30, ref{panel: 'B', item: i}, st)[0]
	if strings.IndexByte(e.Text, '\n') >= 0 {
		l = append(l, deco(" …", st))
	}
	return l
}

// grouped writes n with thousands separators: 9,995.
func grouped(n int) string {
	s := fmt.Sprint(n)
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	return s
}

// sessionPart is a row of zone B's session state: its lead and, unless it is
// a key hint, one logical line of content, copied with a selection like an
// entry's (plan 5, item 9). A wrapped part continues under its lead's indent.
type sessionPart struct {
	lead, text string
	st         *lipgloss.Style
	hint       bool // decoration: a key hint, never copied
	wrap       bool // wrapped to the width, else one row cut at its edge
}

// sessionParts is what zone B shows about the session itself, after the
// last entry; spin is the spinner's frame.
func sessionParts(s *state.State, spin int) []sessionPart {
	name := s.Context.ProviderName
	if s.Session.Version != "" {
		name += " " + s.Session.Version
	}
	switch {
	case s.Startup.Phase == state.StartFailed && s.Context.Mock:
		return failureCard(s, "The mock replay could not start", "Ctrl+Q quit")
	case s.Startup.Phase == state.StartFailed:
		return failureCard(s, "The session could not start", "Esc back to startup (answers kept) · Ctrl+Q quit")
	case s.Session.Phase == state.Crashed && s.Session.ID != "" && !s.Context.Mock:
		return failureCard(s, "The session ended unexpectedly", "Ctrl+R resume · Ctrl+Q quit")
	case s.Session.Phase == state.Crashed:
		return failureCard(s, "The session ended unexpectedly", "Ctrl+Q quit")
	case s.Session.Phase == state.Spawning || s.Session.Phase == state.Handshaking:
		return []sessionPart{{lead: "  " + spinner[spin%len(spinner)] + " ", text: "Starting " + name + " in " + hideUser(s.Context.Project) + "…", st: stProgress, wrap: true}}
	case s.Session.Phase == state.Closed:
		return []sessionPart{{text: "Session closed.", st: stGrey}}
	case s.Session.Phase == state.Ready && len(s.Conversation) == 0:
		return []sessionPart{{text: "Waiting for instructions", st: stTitle}, {text: "Ready to work on this project.", st: stGrey}}
	}
	return nil
}

func failureCard(s *state.State, title, keys string) []sessionPart {
	out := []sessionPart{{lead: "✗ ", text: title, st: stFailedBold}}
	if s.Session.Reason != "" {
		for _, l := range strings.Split(strings.ReplaceAll(s.Session.Reason, "\r\n", "\n"), "\n") {
			out = append(out, sessionPart{lead: "  ", text: l, st: stFailed, wrap: true})
		}
	}
	if s.Session.HasExitCode {
		out = append(out, sessionPart{lead: "  ", text: fmt.Sprintf("exit code %d", s.Session.ExitCode), st: stText})
	}
	if n := len(s.Session.StderrTail); n > 0 {
		out = append(out, sessionPart{lead: "  ", text: "stderr:", st: stGrey})
		for _, l := range s.Session.StderrTail[max(0, n-5):] {
			out = append(out, sessionPart{lead: "    ", text: l, st: stGrey})
		}
	}
	return append(out, sessionPart{text: keys, st: stGrey, hint: true})
}

// sessionLines are the session state's logical lines of content, item
// len(s.Conversation) of zone B for selection and copy.
func sessionLines(s *state.State) []string {
	var out []string
	for _, p := range sessionParts(s, 0) {
		if !p.hint {
			out = append(out, p.text)
		}
	}
	return out
}

// sessionState draws the session state w cells wide; its content spans
// refer to zone B's item after the last entry.
func sessionState(s *state.State, ui UI, w int) []line {
	var out []line
	at := ref{panel: 'B', item: len(s.Conversation)}
	for _, p := range sessionParts(s, ui.Spinner) {
		if p.hint {
			out = append(out, decoLine(p.text, p.st))
			continue
		}
		lead := deco(p.lead, p.st)
		first, rest := 1<<30, 1<<30
		if p.wrap {
			first, rest = w-textWidth(p.lead), w-2
		}
		for k, l := range wrapSource(p.text, first, rest, at, p.st) {
			if k > 0 {
				lead = deco("  ", p.st)
			}
			row := append(line{lead}, l...)
			if !p.wrap && row.width() > w {
				row = cutRow(row, w, at, len(p.text))
			}
			out = append(out, row)
		}
		at.line++
	}
	return out
}

// cutRow cuts a session state's row at w, its "…" standing for the rest of
// the source line at: a selection over it, or past it, copies the line whole
// (plan 5, item 9).
func cutRow(row line, w int, at ref, n int) line {
	row = row.fit(w)
	end := 0 // the source bytes the row still shows
	for _, s := range row {
		switch s.ref.kind {
		case refText:
			end = s.ref.off + len(s.text)
		case refAtom:
			end = s.ref.off + s.ref.n
		}
	}
	at.kind, at.off, at.n = refAtom, end, n-end
	row[len(row)-1].ref = at
	return row
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i] + " …"
	}
	return s
}

func providerLabel(s *state.State) string {
	switch s.CLI() { // in mock mode, the CLI the file recorded
	case "claude-code":
		return "CLAUDE"
	case "codex":
		return "CODEX"
	}
	return "AGENT"
}

// originLabel is how an origin reads on screen (ui-design.md, Origins on screen).
func originLabel(o event.Origin) string {
	if o == event.AI || o == "" {
		return ""
	}
	return string(o)
}

func clock(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Local().Format("15:04")
}

func placeholder(s *state.State) string {
	if s.Session.Phase == state.Ready {
		return "Ask " + s.Context.ProviderName + " anything…"
	}
	return ""
}

func editorStatus(s *state.State) string {
	switch {
	case s.QuitAsked:
		// From the state now: a turn that ended since is no longer named.
		if w := s.Running(); w != "" {
			return "quit? " + w + " will be stopped · Ctrl+Q again to quit · Esc to stay"
		}
		return "quit? nothing runs now · Ctrl+Q again to quit · Esc to stay"
	case s.Turn.Notice != "":
		return s.Turn.Notice
	case s.Session.Notice != "":
		return s.Session.Notice
	case s.Session.Phase == state.Spawning || s.Session.Phase == state.Handshaking:
		return "send available when the session is ready"
	case s.Session.Phase != state.Ready:
		return "the session is " + strings.ToLower(string(s.Session.Phase)) + ": nothing can be sent"
	case s.Turn.Phase == state.Idle:
		return "Enter send · Ctrl+J newline"
	case s.Turn.Phase == state.Interrupting:
		return "interrupting…"
	}
	return "turn running: the draft waits · Ctrl+C interrupt"
}

// agentsByActivity are the agents in zone C's order (ui-design.md, Zone C;
// ASM-22): waiting for an approval, then running, then ended; most recent
// first within each group.
func agentsByActivity(s *state.State) []*state.Agent {
	out := make([]*state.Agent, 0, len(s.Order))
	for _, k := range s.Order {
		if a := s.Agents[k]; a != nil {
			out = append(out, a)
		}
	}
	rank := func(a *state.Agent) int {
		switch {
		case a.Status == state.AgentWaiting:
			return 0
		case !a.Status.Final():
			return 1
		}
		return 2
	}
	recent := func(a *state.Agent) time.Time {
		if a.Status.Final() && !a.EndedAt.IsZero() {
			return a.EndedAt
		}
		return a.StartedAt
	}
	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if ra, rb := rank(a), rank(b); ra != rb {
			return ra < rb
		}
		if ta, tb := recent(a), recent(b); !ta.Equal(tb) {
			return ta.After(tb)
		}
		return a.Seq > b.Seq
	})
	return out
}

// AgentOrder is zone C's order of agent keys: what ↑ ↓ in C and [ ] in
// inspection move through.
func AgentOrder(s *state.State) []string {
	var out []string
	for _, a := range agentsByActivity(s) {
		out = append(out, a.Key)
	}
	return out
}

// activeAgents counts the agents running or waiting for an approval, and
// those ended; an agent only announced is neither.
func activeAgents(s *state.State) (active, done int) {
	for _, a := range s.Agents {
		switch {
		case a.Status == state.AgentRunning || a.Status == state.AgentWaiting:
			active++
		case a.Status.Final():
			done++
		}
	}
	return active, done
}

// zoneC is the live agents list.
func zoneC(s *state.State, ui UI, w, h int, sc *Screen) []line {
	agents := agentsByActivity(s)
	active, done := activeAgents(s)
	right := plain(fmt.Sprintf("%d ACTIVE · %d DONE", active, done), stText)
	cols := agentCols{label: 14} // over every agent, so the columns hold while scrolling
	for _, a := range agents {
		cols.label = max(cols.label, textWidth(a.Label)+1)
		if a.Effort.Present {
			cols.effort = max(cols.effort, textWidth(a.Effort.Text)+1)
		}
		cols.requested = cols.requested || requested(a.Model)
	}
	// The label column never leaves the activity under 12 cells: "▸✓ ",
	// the model's 12, the effort's, and the right part's 21 (plan 7, item 1).
	cols.label = min(cols.label, max(14, w-4-3-12-cols.effort-21-12))
	if cols.requested { // the footnote, as D's "†"
		right = append(line{sp("* requested model · ", stGrey)}, right...)
	}
	var body []line
	if len(agents) == 0 {
		body = append(body, plain("no agents yet: they appear as the CLI starts them", stGrey))
	}
	room := h - 2
	if room >= 4 {
		room--
	}
	from := 0 // the selected agent stays in view
	for i, a := range agents {
		if a.Key == ui.Selected && i >= room {
			from = i - room + 1
		}
	}
	from = paneTop(ui.List, len(agents), max(room, 0), from)
	sc.List = PaneView{Rows: len(agents), Room: max(room, 0), Top: from}
	for _, a := range agents {
		sc.List.IDs = append(sc.List.IDs, a.Key)
	}
	for _, a := range agents[from:min(from+max(room, 0), len(agents))] { // only the rows the box can show
		body = append(body, agentRow(a, ui, w-4, cols))
	}
	if h-2 >= 4 {
		for len(body) < h-3 {
			body = append(body, line{})
		}
		hint := decoLine("↑↓ select · Enter inspect · z zoom · F6 next zone", stGrey)
		if ui.List.Hold && from > 0 { // held away from the top
			hint = heldHint(from, len(agents)-from-room, "Home top")
		}
		body = append(body[:h-3], hint)
	}
	title := "C · LIVE AGENTS"
	if ui.Width < standardWidth { // the screen's layout, not the zone's width
		title = "C · AGENTS"
	}
	return box('C', zoneTitle(title, ui.Focus == "C"), right, body, w, h, borderFor(ui.Focus == "C"))
}

// agentCols are zone C's columns over every agent: the label's width, the
// longest label and a space, at least 14 (plan 7, item 1), and the optional
// parts drawn when any agent has a value for them (plan 6, items 5 and 6):
// the effort column's width, 0 for none, and the "*" footnote of a requested
// model.
type agentCols struct {
	label     int
	effort    int
	requested bool
}

// requested tells a model or effort a spawn call asked for from the agent's
// own report (plan 6, item 5).
func requested(v state.Value) bool { return v.Present && v.Source == "spawn request" }

// source names where an agent's model or effort came from (plan 6, item 5).
func source(v state.Value) string {
	if requested(v) {
		return "(requested)"
	}
	return "(reported)"
}

func agentRow(a *state.Agent, ui UI, w int, cols agentCols) line {
	sel := " "
	if ui.Selected == a.Key {
		sel = "▸"
	}
	m := "model N/A"
	if a.Model.Present {
		// The column is the mock-up's "haiku-5-5": Claude's family name
		// without its prefix; the preview and inspection show it in full.
		m = strings.TrimPrefix(a.Model.Text, "claude-")
	}
	activity := strings.ToLower(string(a.Status))
	switch {
	case a.Status == state.AgentWaiting:
		activity = "waiting: approval"
	case !a.Status.Final() && a.LastTool != "":
		activity = a.LastTool
	case a.Status.Final() && a.Summary != "":
		activity += " · " + firstLine(a.Summary)
	}
	tok := "tok N/A"
	if a.Tokens.Present {
		tok = count(a.Tokens.V) + " tok"
	}
	right := plain(fmt.Sprintf("  %7s  %10s", elapsed(a.StartedAt, a.EndedAt, ui.Now), tok), stGrey)
	left := line{deco(sel, stTitle), agentSymbol(a.Status), sp(" "+padTo(fitLabel(a.Label, cols.label-1), cols.label), stText), sp(m, modelStyle(m))}
	mark := ""
	if requested(a.Model) {
		mark = "*"
	}
	left = append(left, sp(padTo(mark, 12-textWidth(m)), stGrey))
	if cols.effort > 0 {
		left = append(left, sp(padTo(a.Effort.Text, cols.effort), stGrey))
	}
	left = append(left, sp(activity, stGrey))
	return spread(left, right, w)
}

// fitLabel cuts a label longer than w cells with "…"; the preview and the
// inspection show it whole (plan 7, item 1).
func fitLabel(s string, w int) string {
	if textWidth(s) <= w {
		return s
	}
	return cut(s, w-1) + "…"
}

// preview is the selected agent's last events, at wide width.
func preview(s *state.State, ui UI, w, h int, sc *Screen) []line {
	a := s.Agents[ui.Selected]
	if a == nil {
		if ag := agentsByActivity(s); len(ag) > 0 {
			a = ag[0]
		}
	}
	if a == nil {
		return box('P', plain("PREVIEW", stBold), nil, []line{plain("select an agent to preview its activity", stGrey)}, w, h, stFrame)
	}
	var body []line
	for _, e := range a.Messages {
		l := line{sp(clockSec(e.At)+"  ", stGrey), sp(e.Role+"  ", stGrey), sp(firstLine(e.Text), stText)}
		body = append(body, spread(l, plain(originLabel(e.Origin), stGrey), w-4))
	}
	for _, hline := range a.History {
		body = append(body, plain(hline, stGrey))
	}
	if len(body) == 0 {
		body = append(body, plain("no activity yet", stGrey))
	}
	room := max(h-3, 0)
	top := paneTop(ui.Preview, len(body), room, max(len(body)-room, 0))
	sc.Preview = PaneView{Rows: len(body), Room: room, Top: top}
	body = body[top:min(top+room, len(body))]
	for len(body) < h-3 {
		body = append(body, line{})
	}
	hint := decoLine("Enter full screen · [ ] previous / next agent", stGrey)
	if ui.Preview.Hold && top < sc.Preview.Rows-room { // held away from the tail
		hint = heldHint(top, sc.Preview.Rows-top-room, "scroll to the end follows")
	}
	body = append(body, hint)
	title := line{sp(a.Label+" · ", stBold), sp("LIVE", stDone)}
	return box('P', title, previewRight(a, w-9-title.width()), body, w, h, stFrame)
}

// previewRight is the preview's right-hand header in room cells: the first
// that fits of the model, its source and the effort; the model and its
// source; the model alone; nothing. The model is never cut (plan 7, item 2).
func previewRight(a *state.Agent, room int) line {
	model := plain("model N/A", stGrey)
	src := " (not reported)"
	if a.Model.Present {
		model, src = line{sp(a.Model.Text, modelStyle(a.Model.Text))}, " "+source(a.Model)
	}
	withSrc := append(append(line{}, model...), sp(src, stGrey))
	full := withSrc
	if a.Effort.Present {
		full = append(append(line{}, withSrc...), sp(" · effort "+a.Effort.Text+" "+source(a.Effort), stGrey))
	}
	for _, r := range []line{full, withSrc, model} {
		if r.width() <= room {
			return r
		}
	}
	return nil
}

func clockSec(t time.Time) string {
	if t.IsZero() {
		return "--:--:--"
	}
	return t.Local().Format("15:04:05")
}

// count shows a number of tokens: 950, 12.1k, 1.2M.
func count(v float64) string {
	switch {
	case v >= 1e6:
		return trimZero(fmt.Sprintf("%.1f", v/1e6)) + "M"
	case v >= 1e3:
		return trimZero(fmt.Sprintf("%.1f", v/1e3)) + "k"
	}
	return fmt.Sprintf("%.0f", v)
}

func trimZero(s string) string { return strings.TrimSuffix(s, ".0") }

// elapsed is from start to end, or to now while running.
func elapsed(start, end, now time.Time) string {
	if start.IsZero() {
		return "--"
	}
	if end.IsZero() {
		end = now
	}
	if end.IsZero() || end.Before(start) {
		return "--"
	}
	d := end.Sub(start).Round(time.Second)
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm %02ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	return fmt.Sprintf("%dh %02dm", int(d.Hours()), int(d.Minutes())%60)
}

// turnBadge is what the turn is doing, at zone B's top right: the spinner
// turns while an answer streams or the turn runs otherwise (a tool, a
// subagent, the model thinking), and "Idle" says the CLI waits for a message
// (ui-design.md, Zone B; plan 4, step 8).
func turnBadge(s *state.State, ui UI) line {
	spin := spinner[ui.Spinner%len(spinner)]
	switch s.Turn.Phase {
	case state.Running, state.Submitted:
		if n := len(s.Conversation); n > 0 && s.Conversation[n-1].Streaming {
			return plain(spin+" Streaming", stProgress)
		}
		return plain(spin+" Running", stProgress)
	case state.WaitingApproval:
		return plain("! Waiting for approval", stAttention)
	case state.Interrupting:
		return plain("Interrupting…", stProgress)
	case state.Idle:
		if s.Session.Phase == state.Ready {
			return plain("Idle", stGrey)
		}
	}
	return nil
}
