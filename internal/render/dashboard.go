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
	// Zone A's share of the width from 100 columns, in percent, and its
	// least width (plan 24: the owner's image draws it at about 59 of 208
	// columns; its usage section needs the least width).
	zoneAShare = 28
	zoneAMin   = 36
)

// zoneAWidth is zone A's width at w columns from 100 (plan 24): 28% of
// it, rounded, at least 36.
func zoneAWidth(w int) int { return max((w*zoneAShare+50)/100, zoneAMin) }

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

	if ui.Zoom && (ui.Focus == "A" || ui.Focus == "C") {
		// The focused zone over the middle and bottom (ui-design.md,
		// Scrolling, collapsing, full screen); the bars stay.
		zh := h - len(ls) - 1
		keys := "↑↓ select · Enter inspect"
		if ui.Focus == "A" {
			keys = "↑↓ move · ←→ fold · Enter inspect or details"
			sc.AY = len(ls)
			ls = append(ls, zoneA(s, ui, w, zh, sc)...)
		} else {
			ls = append(ls, zoneC(s, ui, w, zh, sc)...)
		}
		return append(ls, keyHints(w, ui, "z or Esc restore · "+keys+" · F6 zone · F1 keys"))
	}

	// Plan 20 (brief amendment 16): A and B share the upper row, their bottom
	// borders on one row; C spans the whole width under both. C is sized by
	// the agents in play (plan 18). Zone D is gone since plan 24: its rows
	// are A's and B's, its values in A's telemetry pane.
	inPlay := s.AgentsInPlay()
	if w < standardWidth {
		// The fallback under 100 columns: A as one row, B and C full width.
		ls = append(ls, zoneALine(s, w))
		region := h - len(ls) - 1
		ch := cHeight(inPlay, region, 0, true)
		bh := region - ch
		if ui.Focus == "A" { // the tree takes B's and C's place until F6 leaves it
			sc.AY = len(ls)
			ls = append(ls, zoneA(s, ui, w, bh+ch, sc)...)
			return append(ls, keyHints(w, ui, "↑↓ move · ←→ fold · Enter inspect or details · F6 zone · Esc back to the input"))
		}
		sc.Conv.Y = len(ls) + 1
		ls = append(ls, zoneB(s, ui, w, bh, editorMinRows, sc)...)
		ls = append(ls, zoneC(s, ui, w, ch, sc)...)
		return append(ls, keyHints(w, ui, footer(s, ui, sendNewline(ui)+" · F6 zone · F2 approve · F3 Explorer · F1 keys")))
	}

	aw, minB := zoneAWidth(w), 12
	if wide {
		minB = 14
	}
	region := h - len(ls) - 1 // A and B, then C
	ch := cHeight(inPlay, region, minB, false)
	top := region - ch
	sc.Conv.Y = len(ls) + 1
	sc.AY = len(ls)
	ls = append(ls, beside(zoneA(s, ui, aw, top, sc), zoneB(s, ui, w-aw, top, editorMinRows, sc))...)
	ls = append(ls, zoneC(s, ui, w, ch, sc)...) // the whole width, its timeline inside (plan 27)
	hints := sendNewline(ui) + " · Ctrl+C interrupt · Ctrl+G draft · F6 zone · F2 approvals · F3 Explorer · F1 keys · Ctrl+Q quit"
	if wide {
		hints = sendNewline(ui) + " · Ctrl+C copy or interrupt · Ctrl+G draft · F6 / Shift+F6 zone · F2 approvals · F3 Explorer · F4 Manual/Auto · z zoom · F12 mouse capture · F1 keys · Ctrl+Q quit"
	}
	if lipgloss.Width(hints)+1 > w {
		// The key page (F1) lists the newline key: the room goes to the end
		// of the line, Ctrl+Q quit (plan 17, F3 in the footer; plan 29).
		hints = strings.Replace(hints, ui.newlineKeys()[0]+" newline · ", "", 1)
	}
	return append(ls, keyHints(w, ui, footer(s, ui, hints)))
}

// editorMinRows is the input's least height: one row under the rule, then
// the status row (plan 20, the templates' compact input).
const editorMinRows = 1

// footer is the bottom line's key hints: B's entry cursor's while it is on,
// C's while C has the focus (plan 20: C has no hint row of its own), A's
// while A has it (plan 24: A has none either), else the dashboard's.
func footer(s *state.State, ui UI, hints string) string {
	switch {
	case ui.Entry > 0 && ui.Focus == "B" && ui.Card == "":
		return entryFooter(s, ui)
	case ui.Focus == "C":
		return "↑↓ select · Enter inspect · z zoom · PgUp PgDn Home End scroll · F6 zone · Esc back to the input · F1 keys"
	case ui.Focus == "A":
		return aFooter
	}
	return hints
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
	// A CLI newer than its adapter was probed with (doc/evaluation.md, After
	// a CLI update): the dashboard may meet events it does not know.
	newer := s.NewerThanTested()
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
	// The right end is the name and version alone while approvals are
	// Manual with no change asked, the owner's image of the top right (plan
	// 28, brief amendment 28); the mode shows only while Auto or changing.
	short := false // at the narrowest, the approvals mode alone, or the name alone without one
	right := func() line {
		mode := approvalsSpans(s)
		if short && len(mode) > 0 {
			return append(mode, sp(" ", stText))
		}
		rest := strings.Join(append([]string{""}, extras...), " · ")
		if short {
			rest = ""
		}
		if len(mode) > 0 {
			rest += " · "
		}
		r := plain(name+rest, stText)
		if newer {
			r = line{sp(name, stText), sp(" newer than tested", stAttention), sp(rest, stText)}
		}
		return append(append(r, mode...), sp(" ", stText))
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

// approvalsSpans is the project bar's approvals mode (plan 12, item 2):
// nothing while Manual is in force with no change asked (plan 28, the
// owner's image); else "approvals " and the mode in force, Auto in the
// attention colour, and a change asked for and not yet in force after an
// arrow. Codex applies it with its next turn; Claude Code answers at once.
func approvalsSpans(s *state.State) line {
	cur, req := s.Mode(), s.Session.ApprovalRequested
	if (cur == "" || cur == state.ModeManual) && req == "" {
		return nil
	}
	l := line{sp("approvals ", stGrey), sp(state.ModeLabel(cur), modeStyle(cur))}
	if req != "" {
		suffix := " …"
		if s.CLI() == "codex" {
			suffix = " (next turn)"
		}
		l = append(l, sp(" → ", stGrey), sp(state.ModeLabel(req), modeStyle(req)), sp(suffix, stGrey))
	}
	return l
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

// zoneB is the conversation, its notices and the editor. The conversation
// follows its end unless the owner scrolled up (ui-design.md, Zone B, Follow
// mode); sc, when given, records what was shown for scrolling and the mouse.
func zoneB(s *state.State, ui UI, w, h, editorRows int, sc *Screen) []line {
	inner := w - 4
	right := turnBadge(s, ui)
	// The editor grows up to half of B, then scrolls (ui-design.md, Input
	// editor, Size); its rule, its separator and its row count in that half
	// (plan 29).
	editor := editorLines(s, ui, inner, editorRows, (h-2)/2-3)
	room := max(h-2-len(editor), 0)
	conv, rows, top, starts := conversation(s, ui, inner, room)
	var shown []line // only the rows shown are drawn
	var folds map[int]string
	for j, c := range conv {
		shown = append(shown, c.draw())
		if c.fold != "" {
			if folds == nil {
				folds = map[int]string{}
			}
			folds[j] = c.fold
		}
	}
	if len(shown) < room {
		shown = append(shown, blank(0, room-len(shown))...)
	}
	if n := NewShown(s, ui.ConvSeen); ui.ConvHold && n > 0 && room > 0 {
		// The badge takes the right end of the last row only; the row's start
		// stays, cut before it (plan 6, item 2).
		shown[room-1] = spread(trimTail(shown[room-1]), decoLine(fmt.Sprintf("↓ %d new", n), stProgress), inner)
	}
	if sc != nil {
		sc.Conv = ConvView{Rows: rows, Room: room, Top: top, Y: sc.Conv.Y, Held: ui.ConvHold, Starts: starts, Folds: folds}
	}
	body := append(shown, editor...)
	focused := ui.Focus == "B" || ui.Focus == ""
	return box('B', zoneTitle("B · CONVERSATION", focused), right, body, w, h, borderFor(focused))
}

// NewShown counts the entries from seen on that zone B draws: a Codex step
// that did not reason leaves no row, and is no news (plan 22, the review). It
// is the "↓ N new" badge's count, and the application reveals an entry by it
// (correction C10).
func NewShown(s *state.State, seen int) int {
	n := 0
	for _, e := range s.Conversation[min(max(seen, 0), len(s.Conversation)):] {
		if e.Role != "thinking" || e.Thought == nil || !e.Thought.Dropped {
			n++
		}
	}
	return n
}

// conversation is the orchestrator's record, then the activity line while
// the turn works (plan 22, design 9), then the session's state: the rows in
// view, at most room from top, and the count of all rows. It follows the end
// unless the owner scrolled up. Only the entries in view are drawn; the
// others are counted, from the cache once finished (test 11). A thinking
// card's size follows room (plan 22, design 2).
func conversation(s *state.State, ui UI, w, room int) (view []crow, rows, top int, starts []int) {
	lk, act := lookAll(s, ui)
	size := cardSize(room)
	starts = make([]int, len(s.Conversation)+1)
	for i, e := range s.Conversation {
		lk[i].size = size
		rows += ui.Wrap.rowCount(i, e, w, lk[i].shape(), func() int { return len(entryRows(s, ui, nil, i, e, w, lk[i])) })
		starts[i+1] = rows
	}
	counts := func(i int) int { return starts[i+1] - starts[i] }
	var activity []crow
	if act {
		activity = activityRows(s, ui, w, lk)
	}
	tail := sessionState(s, ui, w)
	rows += len(activity) + len(tail)
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
			rs := entryRows(s, ui, nil, i, e, w, lk[i])
			view = append(view, rs[min(max(top-at, 0), len(rs)):min(end-at, len(rs))]...)
		}
		at += counts(i)
	}
	for _, c := range activity {
		if at >= top && at < end {
			view = append(view, c)
		}
		at++
	}
	for _, l := range tail {
		if at >= top && at < end {
			view = append(view, crow{l: l})
		}
		at++
	}
	// A rule between two messages shows only with something of both in
	// view: at the window's edge, scrolled, it would sit against B's border
	// or the input's rule as a second line (amendment 17, 2.5). Its row
	// stays, blank, so nothing moves.
	for _, edge := range []int{firstSolid(view, 1), firstSolid(view, -1)} {
		if edge >= 0 && view[edge].sep {
			view[edge] = crow{l: line{}}
		}
	}
	return view, rows, top, starts
}

// firstSolid is the index of the first row of view that shows something,
// from its start (d 1) or its end (d -1); -1 for none.
func firstSolid(view []crow, d int) int {
	i := 0
	if d < 0 {
		i = len(view) - 1
	}
	for ; i >= 0 && i < len(view); i += d {
		if view[i].solid() {
			return i
		}
	}
	return -1
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

// placeholder is the empty draft's grey text: while a turn runs, that the
// CLI works on the request (plan 22, design 10, the owner's image 1).
func placeholder(s *state.State) string {
	switch {
	case s.Session.Phase != state.Ready:
		return ""
	case s.Turn.Phase != state.Idle:
		// Any turn under way, a request waiting or an interrupt asked for
		// included: sending is refused until it ends (plan 22, the review).
		return s.Context.ProviderName + " is processing your request…"
	}
	return "Ask " + s.Context.ProviderName + " anything…"
}

func editorStatus(s *state.State, ui UI) string {
	switch {
	case s.QuitAsked:
		// From the state now: a turn that ended since is no longer named.
		if w := s.Running(); w != "" {
			return "quit? " + w + " will be stopped · Ctrl+Q again to quit · Esc to stay"
		}
		return "quit? nothing runs now · Ctrl+Q again to quit · Esc to stay"
	case keysProblemShown(s, ui):
		return ui.KeysProblem // a mock run has no startup screen (plan 21, item 5.1)
	case s.Turn.Notice != "":
		return s.Turn.Notice
	case s.Session.Notice != "":
		return s.Session.Notice
	case s.Session.Phase == state.Spawning || s.Session.Phase == state.Handshaking:
		return "send available when the session is ready"
	case s.Session.Phase != state.Ready:
		return "the session is " + strings.ToLower(string(s.Session.Phase)) + ": nothing can be sent"
	case s.Turn.Phase == state.Idle:
		return idleStatus(s, ui)
	case s.Turn.Phase == state.Interrupting:
		return "interrupting…"
	}
	return "turn running: the draft waits · Ctrl+C interrupt"
}

// agentsByActivity are zone C's agents, those in play (plan 18: each new
// prompt clears the others, which zone A's tree keeps), in its order
// (ui-design.md, Zone C; ASM-22): waiting for an approval, then running, then
// ended; most recent first within each group.
func agentsByActivity(s *state.State) []*state.Agent {
	out := make([]*state.Agent, 0, len(s.Order))
	for _, k := range s.InPlay() {
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

// activeAgents counts the session's agents running or waiting for an
// approval, and those ended; an agent only announced is neither.
func activeAgents(s *state.State) (active, done int) {
	all := make([]*state.Agent, 0, len(s.Agents))
	for _, a := range s.Agents {
		all = append(all, a)
	}
	return agentCounts(all)
}

// agentCounts counts agents as activeAgents does.
func agentCounts(agents []*state.Agent) (active, done int) {
	for _, a := range agents {
		switch {
		case a.Status == state.AgentRunning || a.Status == state.AgentWaiting:
			active++
		case a.Status.Final():
			done++
		}
	}
	return active, done
}

// agentRetries is an agent's retry count after its time in C, "↻ 2" in amber, and
// nothing while it has none (ui-design.md, Zone C; AM2 C7; plan 21, item
// 5.6a). Codex reports a thread's retries; Claude Code's are the
// orchestrator's, counted in D.
func agentRetries(a *state.Agent) line {
	if a.Retries < 1 {
		return nil
	}
	return line{sp(fmt.Sprintf("  ↻ %d", a.Retries), stProgress)}
}

// clockElapsed is from start to end, or to now while running, as a clock:
// 01:11, 1:02:03 (plan 20, Template 2's agent rows).
func clockElapsed(start, end, now time.Time) string {
	if start.IsZero() {
		return "--:--"
	}
	if end.IsZero() {
		end = now
	}
	if end.IsZero() || end.Before(start) {
		return "--:--"
	}
	d := int(end.Sub(start).Round(time.Second).Seconds())
	if d >= 3600 {
		return fmt.Sprintf("%d:%02d:%02d", d/3600, d/60%60, d%60)
	}
	return fmt.Sprintf("%02d:%02d", d/60, d%60)
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
	right := append(append(plain(fmt.Sprintf("  %7s", elapsed(a.StartedAt, a.EndedAt, ui.Now)), stGrey), agentRetries(a)...),
		sp(fmt.Sprintf("  %10s", tok), stGrey))
	left := line{deco(sel, stTitle), listSymbol(a.Status), sp(" "+padTo(fitLabel(a.Label, cols.label-1), cols.label), stText), sp(m, modelStyle(m))}
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

// listSymbol is an agent's status in zone C: zone A's, except Template 2's
// pink dot for an agent waiting for the owner, where zone A keeps its "!".
func listSymbol(st state.AgentStatus) span {
	if st == state.AgentWaiting {
		return deco("●", stAttention)
	}
	return agentSymbol(st)
}

// fitLabel cuts a label longer than w cells with "…"; the preview and the
// inspection show it whole (plan 7, item 1).
func fitLabel(s string, w int) string {
	if textWidth(s) <= w {
		return s
	}
	return cut(s, w-1) + "…"
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
// turns while the CLI reports a phase of thinking (plan 22), while
// an answer streams, or while the turn runs otherwise (a tool, a subagent, a
// Codex step), and "Idle" says the CLI waits for a message (ui-design.md,
// Zone B; plan 4, step 8).
func turnBadge(s *state.State, ui UI) line {
	spin := spinner[ui.Spinner%len(spinner)]
	switch s.Turn.Phase {
	case state.Running, state.Submitted:
		if thinkingLive(s) {
			return plain(spin+" Thinking", stProgress)
		}
		if n := len(s.Conversation); n > 0 && s.Conversation[n-1].Streaming && s.Conversation[n-1].Role != "thinking" {
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
