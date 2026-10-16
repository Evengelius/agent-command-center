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
func dashboard(s *state.State, ui UI) []line {
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
		ls = append(ls, zoneB(s, ui, w, bh, 3)...)
		ls = append(ls, zoneC(s, ui, w, ch)...)
		ls = append(ls, zoneD(s, ui, w, dh, 1)...)
		return append(ls, keyHints(w, "Enter send · Ctrl+J newline · F6 zone · F2 approve · F1 keys"))
	}

	aw, dh, rows := zoneAStandard, 4, 2
	if wide {
		aw, dh, rows = zoneAWide, 6, 3
	}
	mid := h - len(ls) - dh - 1
	bh := (mid*3 + 2) / 5
	rw := w - aw
	right := zoneB(s, ui, rw, bh, 3+2*boolInt(wide))
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
	return append(ls, keyHints(w, hints))
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func keyHints(w int, text string) line { return plain(" "+text, stGrey).fit(w) }

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
		return sp("◉", stProgress)
	case state.AgentWaiting:
		return sp("!", stAttention)
	case state.AgentCompleted, state.AgentEnded:
		return sp("✓", stDone)
	case state.AgentFailed:
		return sp("✗", stFailed)
	case state.AgentStopped:
		return sp("○", stGrey)
	}
	return sp("○", stGrey)
}

func turnSymbol(s *state.State) span {
	switch s.Turn.Phase {
	case state.Running, state.Submitted, state.Interrupting:
		return sp("◉", stProgress)
	case state.WaitingApproval:
		return sp("!", stAttention)
	}
	switch s.Turn.Outcome {
	case "failed":
		return sp("✗", stFailed)
	case "interrupted":
		return sp("○", stGrey)
	}
	return sp("✓", stDone)
}

// zoneALine is zone A at narrow width: the current path in one line.
func zoneALine(s *state.State, w int) line {
	l := line{sp(" A ", stTitle)}
	if s.Turn.Count == 0 && len(s.Order) == 0 {
		l = append(l, sp("no turn yet", stGrey))
	} else {
		l = append(l, turnSymbol(s), sp(fmt.Sprintf(" Turn %d", s.Turn.Count), stText))
		for _, k := range s.Order {
			a := s.Agents[k]
			l = append(l, sp(" › ", stGrey), agentSymbol(a.Status), sp(" "+a.Label, stText))
		}
	}
	return spread(l, plain("   (from events)   F6: tree ", stGrey), w)
}

// zoneA is the orchestrator and its tree.
func zoneA(s *state.State, ui UI, w, h int) []line {
	var body []line
	m := model(s)
	if m == "" {
		body = append(body, plain("model N/A ("+naReason(s)+")", stGrey))
	} else {
		body = append(body, line{sp(m, modelStyle(m)), sp(" · orchestrator", stGrey)})
	}
	switch {
	case s.Turn.Count == 0 && !s.Turn.ByCLI:
		body = append(body, plain("no turn yet", stGrey))
	default:
		body = append(body, plain(fmt.Sprintf("turn %d · %s", s.Turn.Count, turnWord(s)), stText))
	}
	body = append(body, line{}, line{sp("AGENTS  ", stBold), sp("observed from events", stGrey)})
	if s.Turn.Count > 0 || s.Turn.ByCLI {
		body = append(body, line{turnSymbol(s), sp(fmt.Sprintf(" Turn %d", s.Turn.Count), stText)})
	}
	keys := children(s, "")
	if len(keys) == 0 {
		body = append(body, plain("  ○ (agents appear as events arrive)", stGrey))
	}
	body = append(body, tree(s, keys, "  ")...)
	hint := plain("↑↓ move ←→ fold Enter inspect", stGrey)
	for len(body) < h-3 {
		body = append(body, line{})
	}
	body = append(body[:h-3], hint)
	return box(zoneTitle("A · ORCHESTRATOR", ui.Focus == "A"), nil, body, w, h, borderFor(ui.Focus == "A"))
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

func tree(s *state.State, keys []string, indent string) []line {
	var out []line
	for i, k := range keys {
		a := s.Agents[k]
		branch, next := "├─ ", "│  "
		if i == len(keys)-1 {
			branch, next = "└─ ", "   "
		}
		l := line{sp(indent+branch, stFrame), agentSymbol(a.Status), sp(" "+a.Kind+"  ", stGrey), sp(a.Label, stText)}
		out = append(out, l)
		if a.LastTool != "" && !a.Status.Final() {
			out = append(out, line{sp(indent+next+"└─ ", stFrame), sp("▸ "+a.LastTool, stGrey)})
		}
		out = append(out, tree(s, children(s, k), indent+next)...)
	}
	return out
}

func zoneTitle(t string, focused bool) line {
	if focused {
		return plain(t, stTitle)
	}
	return plain(t, stBold)
}

func borderFor(focused bool) lipgloss.Style {
	if focused {
		return stTitle
	}
	return stFrame
}

// zoneB is the conversation, its notices and the editor.
func zoneB(s *state.State, ui UI, w, h, editorRows int) []line {
	inner := w - 4
	var right line
	if s.Turn.Phase == state.Running || s.Turn.Phase == state.Submitted {
		right = plain("◉ Streaming", stProgress)
	}
	editor := editorLines(s, ui, inner, editorRows)
	room := h - 2 - len(editor)
	conv := conversation(s, ui, inner)
	if len(conv) < room {
		conv = append(conv, blank(0, room-len(conv))...)
	}
	body := append(tail(conv, room), editor...)
	focused := ui.Focus == "B" || ui.Focus == ""
	return box(zoneTitle("B · CONVERSATION", focused), right, body, w, h, borderFor(focused))
}

// conversation is the orchestrator's record as lines, or the session's state
// when there is nothing yet.
func conversation(s *state.State, ui UI, w int) []line {
	var out []line
	who := providerLabel(s)
	for _, e := range s.Conversation {
		switch e.Role {
		case "user":
			out = append(out, line{sp("YOU", stAttention.Bold(true)), sp("  "+clock(e.At), stGrey)})
			out = append(out, indented(e.Text, w, stText)...)
		case "assistant":
			if e.Thinking {
				out = append(out, indented("∴ "+e.Text, w, stGrey)...)
				continue
			}
			head := line{sp(who, stTitle)}
			if m := model(s); m != "" {
				head = append(head, sp("  "+m, modelStyle(m)))
			}
			out = append(out, head)
			text := e.Text
			if e.Streaming {
				text += "▌"
			}
			out = append(out, indented(text, w, stText)...)
		case "tool":
			out = append(out, spread(plain("  ▸ "+firstLine(e.Text), stGrey), plain(originLabel(e.Origin), stGrey), w))
		case "approval":
			out = append(out, plain("! APPROVAL "+firstLine(e.Text), stAttention))
		default:
			out = append(out, indented("· "+e.Text, w, stProgress)...)
		}
	}
	return append(out, sessionState(s, ui, w)...)
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
	out := []line{plain("✗ "+title, stFailed.Bold(true))}
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

func indented(text string, w int, st lipgloss.Style) []line {
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

// editorLines is the editor at B's bottom edge: a rule, the draft, a status.
func editorLines(s *state.State, ui UI, w, rows int) []line {
	out := []line{rule(w)}
	lines := strings.Split(ui.Draft, "\n")
	shown := lines
	if len(shown) > rows {
		shown = shown[len(shown)-rows:]
	}
	for i := 0; i < rows; i++ {
		prefix := "  "
		if i == 0 {
			prefix = "> "
		}
		var l line
		if i < len(shown) {
			l = line{sp(prefix, stTitle), sp(shown[i], stText)}
			if i == len(shown)-1 && (ui.Focus == "B" || ui.Focus == "") {
				l = append(l, sp("█", stTitle))
			}
		} else if i == 0 {
			l = line{sp(prefix, stTitle), sp(placeholder(s), stGrey)}
		}
		out = append(out, l)
	}
	n := len(lines)
	count := fmt.Sprintf("%d line", n)
	if n != 1 {
		count += "s"
	}
	return append(out, spread(plain("  "+editorStatus(s), stGrey), plain(count+" · Ctrl+G editor", stGrey), w))
}

func placeholder(s *state.State) string {
	if s.Session.Phase == state.Ready {
		return "Ask " + s.Context.ProviderName + " anything…"
	}
	return ""
}

func editorStatus(s *state.State) string {
	switch {
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

// agentsByActivity are the agents, active first, then most recent.
func agentsByActivity(s *state.State) []*state.Agent {
	var out []*state.Agent
	for _, k := range s.Order {
		out = append(out, s.Agents[k])
	}
	sort.SliceStable(out, func(i, j int) bool {
		fi, fj := out[i].Status.Final(), out[j].Status.Final()
		if fi != fj {
			return !fi
		}
		return out[i].StartedAt.After(out[j].StartedAt)
	})
	return out
}

// zoneC is the live agents list.
func zoneC(s *state.State, ui UI, w, h int) []line {
	agents := agentsByActivity(s)
	active, done := 0, 0
	for _, a := range agents {
		if a.Status.Final() {
			done++
		} else {
			active++
		}
	}
	right := plain(fmt.Sprintf("%d ACTIVE · %d DONE", active, done), stText)
	var body []line
	if len(agents) == 0 {
		body = append(body, plain("no agents yet: they appear as the CLI starts them", stGrey))
	}
	for _, a := range agents {
		body = append(body, agentRow(a, ui, w-4))
	}
	if h-2 >= 4 {
		for len(body) < h-3 {
			body = append(body, line{})
		}
		body = append(body[:h-3], plain("↑↓ select · Enter inspect · z zoom · F6 next zone", stGrey))
	}
	title := "C · LIVE AGENTS"
	if w < standardWidth {
		title = "C · AGENTS"
	}
	return box(zoneTitle(title, ui.Focus == "C"), right, body, w, h, borderFor(ui.Focus == "C"))
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
		activity += " · " + a.Summary
	}
	tok := "tok N/A"
	if a.Tokens.Present {
		tok = count(a.Tokens.V) + " tok"
	}
	right := plain(fmt.Sprintf("  %7s  %10s", elapsed(a.StartedAt, a.EndedAt, ui.Now), tok), stGrey)
	left := line{sp(sel, stTitle), agentSymbol(a.Status), sp(" "+padTo(a.Label, 14), stText), sp(padTo(m, 12), modelStyle(m)), sp(activity, stGrey)}
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
		return box(plain("PREVIEW", stBold), nil, []line{plain("select an agent to preview its activity", stGrey)}, w, h, stFrame)
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
	body = append(body, plain("Enter full screen · [ ] previous / next agent", stGrey))
	right := plain("model N/A (not reported)", stGrey)
	if a.Model.Present {
		right = plain(a.Model.Text, modelStyle(a.Model.Text))
	}
	return box(line{sp(a.Label+" · ", stBold), sp("LIVE", stDone)}, right, body, w, h, stFrame)
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
	return box(zoneTitle(title, ui.Focus == "D"), plain("† undocumented field", stGrey), body, w, h, borderFor(ui.Focus == "D"))
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
	return line{sp(strings.Repeat("▰", full), st), sp(strings.Repeat("▱", n-full), stFrame)}
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
	n := 0
	for _, a := range s.Agents {
		if !a.Status.Final() {
			n++
		}
	}
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
