package render

import (
	"fmt"
	"math"
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

	if w < standardWidth {
		ls = append(ls, zoneALine(s, w))
		dh, ch := 3, 5
		bh := h - len(ls) - ch - dh - 1
		if ui.Focus == "A" { // the tree takes B's and C's place until F6 leaves it
			ls = append(ls, zoneA(s, ui, w, bh+ch)...)
			ls = append(ls, zoneD(s, ui, w, dh, 1)...)
			return append(ls, keyHints(w, ui, "↑↓ move · ←→ fold · Enter inspect · F6 zone · Esc back to the input"))
		}
		sc.Conv.Y = len(ls) + 1
		ls = append(ls, zoneB(s, ui, w, bh, 3, sc)...)
		ls = append(ls, zoneC(s, ui, w, ch)...)
		ls = append(ls, zoneD(s, ui, w, dh, 1)...)
		return append(ls, keyHints(w, ui, "Enter send · Ctrl+J newline · F6 zone · F2 approve · F1 keys"))
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
		right = append(right, beside(zoneC(s, ui, listWide, mid-bh), preview(s, ui, rw-listWide, mid-bh))...)
	} else {
		right = append(right, zoneC(s, ui, rw, mid-bh)...)
	}
	ls = append(ls, beside(zoneA(s, ui, aw, mid), right)...)
	ls = append(ls, zoneD(s, ui, w, dh, rows)...)
	hints := "Enter send · Ctrl+J newline · Ctrl+C interrupt · Ctrl+G editor · F6 zone · F2 approvals · F1 keys · Ctrl+Q quit"
	if wide {
		hints = "Enter send · Ctrl+J newline · Ctrl+C copy or interrupt · Ctrl+G editor · F6 / Shift+F6 zone · F2 approvals · z zoom · F12 mouse capture · F1 keys · Ctrl+Q quit"
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
	if ui.Branch != "" {
		left = append(left, sp("   branch ", stGrey), sp(ui.Branch, stText))
	}
	name := s.Context.ProviderName
	if s.Session.Version != "" {
		name += " " + s.Session.Version
	}
	if s.Context.Provider == "codex" {
		for _, x := range []struct{ k, v string }{{"", s.Session.Model}, {"sandbox ", s.Session.Sandbox}, {"approvals ", s.Session.ApprovalPolicy}} {
			if x.v != "" {
				name += " · " + x.k + x.v
			}
		}
	}
	right := plain(name+" ", stText)
	if ui.Width >= standardWidth {
		room := ui.Width - left.width() - right.width() - 6
		if room > 10 {
			left = append(left, sp("   "+elide(s.Context.Project, room), stGrey))
		}
	}
	return spread(left, right, ui.Width)
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
func zoneA(s *state.State, ui UI, w, h int) []line {
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
	for _, r := range treeWindow(rows, ui.ACursor, room) {
		if ui.Focus == "A" && r.node.ID != "" && r.node.ID == ui.ACursor {
			r.l = cursorLine(r.l.fit(w-4), w-4)
		}
		body = append(body, r.l)
	}
	for len(body) < h-3 {
		body = append(body, line{})
	}
	body = append(body[:max(h-3, 0)], decoLine("↑↓ move ←→ fold Enter inspect", stGrey))
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
	var right line
	switch s.Turn.Phase {
	case state.Running, state.Submitted:
		right = plain("◉ Streaming", stProgress)
	case state.WaitingApproval:
		right = plain("! Waiting for approval", stAttention)
	case state.Interrupting:
		right = plain("Interrupting…", stProgress)
	}
	// The editor grows up to half of B, then scrolls (ui-design.md, Input
	// editor, Size); its rule and status line count in that half.
	editor := editorLines(s, ui, inner, editorRows, (h-2)/2-2)
	room := max(h-2-len(editor), 0)
	conv := conversation(s, ui, inner)
	bottom := max(len(conv)-room, 0)
	top := bottom
	if ui.ConvHold {
		top = min(max(ui.ConvTop, 0), bottom)
	}
	var shown []line // only the rows shown are drawn
	for _, c := range conv[top:min(top+room, len(conv))] {
		shown = append(shown, c.draw())
	}
	if len(shown) < room {
		shown = append(shown, blank(0, room-len(shown))...)
	}
	if n := len(s.Conversation) - ui.ConvSeen; ui.ConvHold && n > 0 && room > 0 {
		shown[room-1] = spread(nil, decoLine(fmt.Sprintf("↓ %d new", n), stProgress), inner)
	}
	if sc != nil {
		sc.Conv = ConvView{Rows: len(conv), Room: room, Top: top, Y: sc.Conv.Y, Held: ui.ConvHold}
	}
	body := append(shown, editor...)
	focused := ui.Focus == "B" || ui.Focus == ""
	return box('B', zoneTitle("B · CONVERSATION", focused), right, body, w, h, borderFor(focused))
}

// conversation is the orchestrator's record as lines, or the session's state
// when there is nothing yet. Content comes from entryLines; everything else
// on these lines is decoration.
func conversation(s *state.State, ui UI, w int) []crow {
	var out []crow
	wc := ui.Wrap
	who := providerLabel(s)
	for i, e := range s.Conversation {
		switch e.Role {
		case "user":
			out = drawn(out, line{deco("YOU", stAttentionBold), deco("  "+clock(e.At), stGrey)})
			ls := entryLines(e)
			if len(ls) > foldOver {
				out = contentRows(out, wc, i, ls[:foldShow], w, stText)
				out = drawn(out, decoLine(fmt.Sprintf("  … %s more lines", grouped(len(ls)-foldShow)), stGrey))
				break
			}
			out = contentRows(out, wc, i, ls, w, stText)
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
				out = contentRows(out, wc, i, entryLines(e), w, stText)
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
			l := append(line{deco("  ▸ ", stGrey)}, oneRow(i, e, stGrey)...)
			out = drawn(out, spread(l, decoLine(originLabel(e.Origin), stGrey), w))
		case "approval":
			out = drawn(out, approvalCard(s, ui, i, e, w)...)
		default:
			p := &rowStyle{lead0: deco("  · ", stProgress), lead: deco("  ", stProgress), at: ref{panel: 'B', item: i}, st: stProgress}
			for j, l := range entryLines(e) {
				out = sourceRows(out, wc, p, j, l, w-4, w-2)
			}
		}
	}
	return drawn(out, sessionState(s, ui, w)...)
}

// oneRow is a tool or approval entry's first line on one row, marked when
// more lines follow.
func oneRow(i int, e state.Entry, st *lipgloss.Style) line {
	l := wrapSource(entryLines(e)[0], 1<<30, 1<<30, ref{panel: 'B', item: i}, st)[0]
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

// sessionState is what zone B shows about the session itself.
func sessionState(s *state.State, ui UI, w int) []line {
	name := s.Context.ProviderName
	if s.Session.Version != "" {
		name += " " + s.Session.Version
	}
	switch {
	case s.Startup.Phase == state.StartFailed && s.Context.Mock:
		return failureCard(s, w, "The mock replay could not start", "Ctrl+Q quit")
	case s.Startup.Phase == state.StartFailed:
		return failureCard(s, w, "The session could not start", "Esc back to startup (answers kept) · Ctrl+Q quit")
	case s.Session.Phase == state.Crashed && s.Session.ID != "" && !s.Context.Mock:
		return failureCard(s, w, "The session ended unexpectedly", "Ctrl+R resume · Ctrl+Q quit")
	case s.Session.Phase == state.Crashed:
		return failureCard(s, w, "The session ended unexpectedly", "Ctrl+Q quit")
	case s.Session.Phase == state.Spawning || s.Session.Phase == state.Handshaking:
		return indented(spinner[ui.Spinner%len(spinner)]+" Starting "+name+" in "+hideUser(s.Context.Project)+"…", w, stProgress)
	case s.Session.Phase == state.Closed:
		return []line{plain("Session closed.", stGrey)}
	case s.Session.Phase == state.Ready && len(s.Conversation) == 0:
		return []line{plain("Waiting for instructions", stTitle), plain("Ready to work on this project.", stGrey)}
	}
	return nil
}

func failureCard(s *state.State, w int, title, keys string) []line {
	out := []line{plain("✗ "+title, stFailedBold)}
	if s.Session.Reason != "" {
		out = append(out, indented(s.Session.Reason, w, stFailed)...)
	}
	if s.Session.HasExitCode {
		out = append(out, plain(fmt.Sprintf("  exit code %d", s.Session.ExitCode), stText))
	}
	if n := len(s.Session.StderrTail); n > 0 {
		out = append(out, plain("  stderr:", stGrey))
		for _, l := range s.Session.StderrTail[max(0, n-5):] {
			out = append(out, plain("    "+l, stGrey))
		}
	}
	return append(out, plain(keys, stGrey))
}

func indented(text string, w int, st *lipgloss.Style) []line {
	var out []line
	for _, l := range wrap(text, w-2) {
		out = append(out, plain("  "+l, st))
	}
	return out
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i] + " …"
	}
	return s
}

func providerLabel(s *state.State) string {
	switch s.Context.Provider {
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
func zoneC(s *state.State, ui UI, w, h int) []line {
	agents := agentsByActivity(s)
	active, done := activeAgents(s)
	right := plain(fmt.Sprintf("%d ACTIVE · %d DONE", active, done), stText)
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
	for _, a := range agents[from:] {
		body = append(body, agentRow(a, ui, w-4))
	}
	if h-2 >= 4 {
		for len(body) < h-3 {
			body = append(body, line{})
		}
		body = append(body[:h-3], decoLine("↑↓ select · Enter inspect · z zoom · F6 next zone", stGrey))
	}
	title := "C · LIVE AGENTS"
	if w < standardWidth {
		title = "C · AGENTS"
	}
	return box('C', zoneTitle(title, ui.Focus == "C"), right, body, w, h, borderFor(ui.Focus == "C"))
}

func agentRow(a *state.Agent, ui UI, w int) line {
	sel := " "
	if ui.Selected == a.Key {
		sel = "▸"
	}
	m := "model N/A"
	if a.Model.Present {
		m = a.Model.Text
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
	left := line{deco(sel, stTitle), agentSymbol(a.Status), sp(" "+padTo(a.Label, 14), stText), sp(padTo(m, 12), modelStyle(m)), sp(activity, stGrey)}
	return spread(left, right, w)
}

// preview is the selected agent's last events, at wide width.
func preview(s *state.State, ui UI, w, h int) []line {
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
	body = tail(body, h-3)
	for len(body) < h-3 {
		body = append(body, line{})
	}
	body = append(body, decoLine("Enter full screen · [ ] previous / next agent", stGrey))
	right := plain("model N/A (not reported)", stGrey)
	if a.Model.Present {
		right = plain(a.Model.Text, modelStyle(a.Model.Text))
	}
	return box('P', line{sp(a.Label+" · ", stBold), sp("LIVE", stDone)}, right, body, w, h, stFrame)
}

func clockSec(t time.Time) string {
	if t.IsZero() {
		return "--:--:--"
	}
	return t.Local().Format("15:04:05")
}

// zoneD is quota and telemetry, rows lines high.
func zoneD(s *state.State, ui UI, w, h, rows int) []line {
	m := s.Metrics
	q5, q7 := quota(s, "quota.5h", "5h"), quota(s, "quota.7d", "7d")
	ctx := contextValue(s)
	tok := valueText(s, m["tokens.session"], func(v state.Value) string { return count(v.V) })
	cost := costText(s)
	errs := fmt.Sprintf("errors %d", len(s.Errors))
	var body []line
	switch rows {
	case 1:
		body = []line{join(q5.short(10), q7.short(10), "ctx "+ctx.short, "tok "+tok.short, cost.short, errs)}
	case 2:
		body = []line{
			join(q5.long(20), q7.long(20), "model "+orNA(model(s), naReason(s))),
			join("context "+ctx.long, "tokens "+tok.long, "cost "+cost.long, "agents "+agentCount(s), "time "+sessionTime(s, ui), errs),
		}
	default:
		body = []line{
			join(q5.long(30), q7.long(30), "plan "+valueText(s, m["plan"], func(v state.Value) string { return v.Text }).long),
			join("model "+orNA(model(s), naReason(s)), "context "+ctx.long, "tokens "+tok.long, "cost "+cost.long),
			join("agents "+agentCount(s), "time "+sessionTime(s, ui), errs, fmt.Sprintf("retries %d", s.Retries), fmt.Sprintf("unrecognised events %d", s.Unrecognised)),
		}
	}
	title := "D · QUOTA & TELEMETRY"
	if rows == 1 {
		title = "D"
	}
	return box('D', zoneTitle(title, ui.Focus == "D"), plain("† undocumented field", stGrey), body, w, h, borderFor(ui.Focus == "D"))
}

// shown is a value in its short and long forms.
type shown struct{ short, long string }

func join(parts ...any) line {
	var l line
	for i, p := range parts {
		if i > 0 {
			l = append(l, sp("   ", stText))
		}
		switch p := p.(type) {
		case string:
			st := stText
			if strings.Contains(p, "N/A") {
				st = stGrey
			}
			l = append(l, sp(p, st))
		case line:
			l = append(l, p...)
		}
	}
	return l
}

type quotaShown struct {
	label   string
	v       state.Value
	present bool
	reason  string
}

func quota(s *state.State, key, label string) quotaShown {
	v := s.Metrics[key]
	return quotaShown{label: label, v: v, present: v.Present, reason: reasonOf(s, v)}
}

func (q quotaShown) bar(n int) line {
	full := int(math.Round(q.v.V / 100 * float64(n)))
	full = max(0, min(n, full))
	st := stDone
	switch {
	case q.v.V >= 90:
		st = stFailed
	case q.v.V >= 70:
		st = stProgress
	}
	return line{deco(strings.Repeat("▰", full), st), deco(strings.Repeat("▱", n-full), stFrame)}
}

func (q quotaShown) pct() string {
	p := fmt.Sprintf("%.0f%%", q.v.V)
	if !q.v.Documented {
		p += "†"
	}
	return p
}

func (q quotaShown) short(n int) line {
	if !q.present {
		return plain(q.label+" N/A", stGrey)
	}
	return append(append(line{sp(q.label+" ", stText)}, q.bar(n)...), sp(" "+q.pct(), stText))
}

func (q quotaShown) long(n int) line {
	if !q.present {
		return plain(q.label+" N/A ("+q.reason+")", stGrey)
	}
	l := append(append(line{sp(q.label+" ", stText)}, q.bar(n)...), sp(" "+q.pct(), stText))
	if !q.v.ResetsAt.IsZero() {
		l = append(l, sp("  resets "+q.v.ResetsAt.Local().Format("Mon 15:04"), stGrey))
	}
	return l
}

func valueText(s *state.State, v state.Value, f func(state.Value) string) shown {
	if !v.Present {
		return shown{"N/A", "N/A (" + reasonOf(s, v) + ")"}
	}
	t := f(v)
	if !v.Documented {
		t += "†"
	}
	if v.Origin == event.Derived {
		return shown{t, t + " (derived)"}
	}
	return shown{t, t}
}

func contextValue(s *state.State) shown {
	used, win := s.Metrics["context.used"], s.Metrics["context.window"]
	if !used.Present || !win.Present || win.V <= 0 {
		missing := used
		if used.Present {
			missing = win
		}
		return shown{"N/A", "N/A (" + reasonOf(s, missing) + ")"}
	}
	p := fmt.Sprintf("%.0f%%", used.V/win.V*100)
	if !used.Documented || !win.Documented {
		p += "†"
	}
	return shown{p, p + " of " + count(win.V)}
}

func costText(s *state.State) shown {
	v := s.Metrics["cost.usd"]
	if !v.Present {
		if s.Context.Provider == "codex" {
			return shown{"$ N/A", "N/A (Codex reports none)"}
		}
		return shown{"$ N/A", "N/A (" + reasonOf(s, v) + ")"}
	}
	return shown{fmt.Sprintf("$%.2f est.", v.V), fmt.Sprintf("$%.2f API-equiv. estimate", v.V)}
}

func agentCount(s *state.State) string {
	n, _ := activeAgents(s)
	return fmt.Sprintf("%d active / limit N/A (not reported)", n)
}

func sessionTime(s *state.State, ui UI) string {
	if len(s.Log) == 0 || ui.Now.IsZero() {
		return "N/A (" + naReason(s) + ")"
	}
	return elapsed(s.Log[0].At, time.Time{}, ui.Now) + " session"
}

func reasonOf(s *state.State, v state.Value) string {
	if v.Reason != "" {
		return v.Reason
	}
	return naReason(s)
}

// naReason says why a value is missing, from the session's state.
func naReason(s *state.State) string {
	switch s.Session.Phase {
	case state.Spawning, state.Handshaking, "":
		return "session starting"
	case state.Ready:
		return "not reported yet"
	}
	return "session ended"
}

func orNA(v, reason string) string {
	if v == "" {
		return "N/A (" + reason + ")"
	}
	return v
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
