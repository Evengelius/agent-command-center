package render

import (
	"fmt"
	"strings"

	"github.com/Evengelius/agent-command-center/internal/state"
)

// Zone C after the owner's image (plan 27, brief amendment 26) and its
// close-ups (the second pass, amendment 27; reference-design.md, Zone C
// with the execution timeline): the heading inside the frame, the agents'
// panel on the left, as wide as its cards, the execution timeline on the
// right, the selected agent's status at the timeline heading's right end.
// Under 100 columns the narrow layout keeps plan 18's one-row list
// (zoneCNarrow).
const (
	compactC       = 4 // no agent in play: the edges, the header row, "no agents yet"
	narrowCompactC = 3 // the narrow layout's, plan 18: the edges and one row
	narrowC        = 5 // the narrow layout's most rows
	cHeadRows      = 3 // the header row, its rule, the panels' headings
	// The panel's most share of the inner width in percent (the image: two
	// fifths), its least width (a card of the least columns) and the
	// timeline's least width (design D9).
	listShare   = 39
	listLeast   = cardMargins + labelMin + toolMin
	timelineMin = 20
	// The card (design D8): the margin cell, the symbol and two cells; three
	// cells, the bar and two cells; two cells at the end: 12 cells around
	// its two columns. The label column holds the label or the second row's
	// clock and tokens, at least "00:41   30.1k tok"; the tool column the
	// model or "tool: <name>", from "tool: Write" to "tool:
	// commandExecution", and "tool: " with ten cells where the panel must
	// give way first (D9).
	cardMargins = 12
	labelMin    = 17
	toolMin     = 11
	toolMax     = 22
	toolTight   = 16
	// The timeline's entries the pane is sized for, the image's four, two
	// rows each and a rule between them.
	timelineEntries = 4
	timelineRows    = timelineEntries*2 + timelineEntries - 1
	// From 100 columns, with an agent in play, C never has less than one
	// agent's two rows, the headings and the edges, A and B giving the rows
	// (plan 21, item 4).
	agentC    = 2 + cHeadRows + 2
	roleWidth = 9 // the timeline's role column: "assistant"
)

// cHeight is zone C's height (plan 18, plan 20, plan 27): with no agent in
// play, compactC; else its edges, its three heading rows and the larger of
// the list's rows (two an agent) and the timeline's eleven; the list's rows
// take up to half the rows A, B and C share (plan 18), the timeline's up to
// a third of them; B keeps minB, and C never has less than agentC. The
// narrow layout keeps plan 18's: its edges and one row an agent, at most
// narrowC.
func cHeight(n, region, minB int, narrow bool) int {
	if narrow {
		if n == 0 {
			return narrowCompactC
		}
		return min(max(2+n, narrowCompactC), narrowC)
	}
	if n == 0 {
		return compactC
	}
	list := 2 + cHeadRows + 2*n
	need := 2 + cHeadRows + max(2*n, timelineRows)
	most := max(min(list, region/2), min(need, region/3), agentC)
	return min(need, max(min(most, region-minB), agentC))
}

// cardCols are the card's columns over every listed agent, so they hold
// while scrolling (plan 7, item 1), and the panel's width (design D9); the
// "*" footnote of a requested model for the header.
type cardCols struct {
	label, tool, width int
	requested          bool
}

// toolInfo is an agent's current tool for its card (design D8).
type toolInfo struct {
	name   string
	failed bool
}

// currentTools is each listed agent's current tool: its latest call by
// sequence, a call waiting for the owner included (P3b's "Write"), else the
// tool the CLI reported (plan 7, item 1); one pass over the calls.
func currentTools(s *state.State, agents []*state.Agent) map[string]toolInfo {
	out := make(map[string]toolInfo, len(agents))
	last := make(map[string]*state.ToolCall, len(agents))
	for _, a := range agents {
		out[a.Key] = toolInfo{name: a.LastTool}
	}
	for _, tc := range s.Tools { // a map: the latest by sequence, then by ID, as agentTools orders them
		if _, listed := out[tc.Agent]; !listed {
			continue
		}
		if b := last[tc.Agent]; b != nil && (tc.Seq < b.Seq || tc.Seq == b.Seq && tc.ID < b.ID) {
			continue
		}
		last[tc.Agent] = tc
	}
	for k, tc := range last {
		out[k] = toolInfo{name: tc.Name, failed: tc.Status == state.ToolFailed}
	}
	return out
}

// toolText is the card's tool text: "tool: " and the name, "none yet"
// without one.
func toolText(t toolInfo) string {
	if t.name == "" {
		return "tool: none yet"
	}
	return "tool: " + t.name
}

// cardColumns sizes the cards (design D9): the label column is the longest
// label or second-row content, at least labelMin; the tool column the
// longest model with its mark or "tool: <name>", toolMin to toolMax; the
// panel is cardMargins more, never over the larger of listShare of the
// inner width and listLeast, nor over the inner width less the rule and
// timelineMin: over that, the tool column gives way to toolTight, then the
// label column to labelMin, then the tool column to toolMin.
func cardColumns(agents []*state.Agent, tools map[string]toolInfo, ui UI, inner int) cardCols {
	c := cardCols{label: labelMin, tool: toolMin}
	for _, a := range agents {
		c.label = max(c.label, textWidth(a.Label), cardClock(a, ui).width())
		c.tool = max(c.tool, textWidth(cardModel(a)), textWidth(toolText(tools[a.Key])))
		c.requested = c.requested || requested(a.Model)
	}
	c.tool = min(c.tool, toolMax)
	most := min(max((inner*listShare+50)/100, listLeast), inner-3-timelineMin)
	if over := c.label + c.tool + cardMargins - most; over > 0 {
		give := min(over, max(c.tool-toolTight, 0))
		c.tool, over = c.tool-give, over-give
		give = min(over, max(c.label-labelMin, 0))
		c.label, over = c.label-give, over-give
		give = min(over, max(c.tool-toolMin, 0))
		c.tool -= give
	}
	c.width = c.label + c.tool + cardMargins
	return c
}

// cardClock is the card's second row before the bar: the elapsed time as a
// clock, the retries "↻ 2" in amber when any (AM2 C7; plan 21, item 5.6a),
// then the tokens, "tok N/A" without a count.
func cardClock(a *state.Agent, ui UI) line {
	tok := "tok N/A"
	if a.Tokens.Present {
		tok = count(a.Tokens.V) + " tok"
	}
	l := line{sp(clockElapsed(a.StartedAt, a.EndedAt, ui.Now), stGrey)}
	l = append(l, agentRetries(a)...)
	return append(l, sp("   "+tok, stGrey))
}

// padCell pads s with spaces to exactly w cells when it is narrower, and
// adds none when it fills them (padTo adds one then, for a title).
func padCell(s string, w int) string {
	return s + strings.Repeat(" ", max(w-textWidth(s), 0))
}

// cardModel is the card's model column: Claude's family name without its
// prefix (plan 7), "*" after a requested model (plan 6, item 5), "model
// N/A" without one.
func cardModel(a *state.Agent) string {
	if !a.Model.Present {
		return "model N/A"
	}
	m := strings.TrimPrefix(a.Model.Text, "claude-")
	if requested(a.Model) {
		m += "*"
	}
	return m
}

// agentCard is an agent's two rows (design D8), cols.width cells: the
// status symbol, the label in bold and, after the grey bar, the model in
// grey; then the clock, the retries and the tokens and, after the bar,
// "tool: " with the current tool in bold, "none yet" in grey without one,
// a failed call's "✗". The selected agent's rows lie on the tool rows'
// ground (plan 19) between the margin cells; on 16 colours, which have no
// ground, nothing marks it: both palettes draw the same characters.
func agentCard(a *state.Agent, tool toolInfo, ui UI, cols cardCols) []line {
	iw := cols.width - 2 // between the margin cells
	bar := line{deco("   ", stNone), deco("│", stGrey), deco("  ", stNone)}
	first := line{listSymbol(a.Status), deco("  ", stNone), sp(padCell(fitLabel(a.Label, cols.label), cols.label), stBold)}
	first = append(append(first, bar...), sp(fitLabel(cardModel(a), cols.tool), stGrey))
	second := append(line{deco("   ", stNone)}, cardClock(a, ui)...).pad(3 + cols.label)
	second = append(append(second, bar...), sp("tool: ", stGrey))
	switch {
	case tool.name == "":
		second = append(second, sp("none yet", stGrey))
	case tool.failed:
		second = append(second, sp(fitLabel(tool.name, cols.tool-8), stBold), deco(" ", stNone), deco("✗", stFailed))
	default:
		second = append(second, sp(fitLabel(tool.name, cols.tool-6), stBold))
	}
	rows := []line{first.fit(iw), second.fit(iw)}
	if a.Key == ui.Selected { // on 16 colours tint gives the style itself: no ground, no mark
		for i := range rows {
			rows[i] = grounded(rows[i], iw)
		}
	}
	for i := range rows {
		rows[i] = append(append(line{deco(" ", stNone)}, rows[i]...), deco(" ", stNone))
	}
	return rows
}

// zoneC is zone C, w cells wide and h rows tall: the whole screen's width
// (plan 20). Row 1 inside the frame is the header, "C · LIVE AGENTS" with
// the counts at its right end; row 2 a rule; row 3 the panels' headings;
// then the cards' rows left of the panels' line and the timeline's right
// of it (design D1 to D3, D7 to D9).
func zoneC(s *state.State, ui UI, w, h int, sc *Screen) []line {
	if ui.Width < standardWidth { // the screen's layout, not the zone's width
		return zoneCNarrow(s, ui, w, h, sc)
	}
	agents := agentsByActivity(s)
	active, done := agentCounts(agents) // the agents it shows (plan 18)
	inner := w - 4
	focused := ui.Focus == "C"
	counts := line{sp(fmt.Sprintf("%d ACTIVE · %d DONE", active, done), stGrey)}
	if len(agents) == 0 {
		sc.List, sc.Preview = PaneView{}, PaneView{}
		body := []line{
			spread(decoLine("C · LIVE AGENTS", stTitle), counts, inner),
			plain("no agents yet: they appear as the CLI starts them", stGrey),
		}
		return box('C', nil, nil, body, w, h, borderFor(focused))
	}
	if s.Agents[ui.Selected] == nil { // the first listed agent, as the timeline shows it
		ui.Selected = agents[0].Key
	}
	tools := currentTools(s, agents)
	cols := cardColumns(agents, tools, ui, inner)
	lw := cols.width
	tw := inner - lw - 3
	if cols.requested { // the footnote, as the "†" of the details view
		counts = append(line{sp("* requested model · ", stGrey)}, counts...)
	}
	rows := max(h-2-cHeadRows, 0)
	room := rows / 2 // agents in view, two rows each
	from := 0        // the selected agent stays in view
	for i, a := range agents {
		if a.Key == ui.Selected && i >= room {
			from = i - room + 1
		}
	}
	from = paneTop(ui.List, len(agents), room, from)
	sc.List = PaneView{Rows: len(agents), Room: room, Top: from}
	for _, a := range agents {
		sc.List.IDs = append(sc.List.IDs, a.Key)
	}
	var list []line
	for _, a := range agents[from:min(from+room, len(agents))] { // only the rows the panel can show
		list = append(list, agentCard(a, tools[a.Key], ui, cols)...)
	}
	// The agents out of view, before the counts (plan 20).
	var out []string
	if from > 0 {
		out = append(out, fmt.Sprintf("↑ %d above", from))
	}
	if below := len(agents) - from - room; below > 0 {
		out = append(out, fmt.Sprintf("↓ %d below", below))
	}
	if len(out) > 0 {
		counts = append(line{sp(strings.Join(out, " · ")+" · ", stProgress)}, counts...)
	}
	status, tl := timeline(s, ui, tw, rows, sc)
	// The close-ups' lines: the rule under the header and the line between
	// the panels in the title colour, the headings in it, bold (D7).
	sep := deco(" │ ", stPanelRule)
	body := []line{
		spread(decoLine("C · LIVE AGENTS", stTitle), counts, inner),
		decoLine(strings.Repeat("─", inner), stPanelRule),
		append(line{deco(padTo("AGENTS", lw), stTitle), sep}, spread(decoLine("EXECUTION TIMELINE", stTitle), status, tw)...),
	}
	for i := 0; i < rows; i++ {
		var l, r line
		if i < len(list) {
			l = list[i]
		}
		if i < len(tl) {
			r = tl[i]
		}
		body = append(body, append(append(l.fit(lw), sep), r.fit(tw)...))
	}
	return box('C', nil, nil, body, w, h, borderFor(focused))
}

// zoneCNarrow is zone C under 100 columns as plan 18 drew it: its title in
// the edge, the counts at the edge's right, one row an agent, no timeline.
func zoneCNarrow(s *state.State, ui UI, w, h int, sc *Screen) []line {
	agents := agentsByActivity(s)
	active, done := agentCounts(agents)
	right := plain(fmt.Sprintf("%d ACTIVE · %d DONE", active, done), stText)
	lw := w - 4
	cols := agentColumns(agents, lw, 21)
	if cols.requested {
		right = append(line{sp("* requested model · ", stGrey)}, right...)
	}
	var body []line
	if len(agents) == 0 {
		body = append(body, plain("no agents yet: they appear as the CLI starts them", stGrey))
	}
	room := max(h-2, 0)
	from := 0
	for i, a := range agents {
		if a.Key == ui.Selected && i >= room {
			from = i - room + 1
		}
	}
	from = paneTop(ui.List, len(agents), room, from)
	sc.List = PaneView{Rows: len(agents), Room: room, Top: from}
	sc.Preview = PaneView{}
	for _, a := range agents {
		sc.List.IDs = append(sc.List.IDs, a.Key)
	}
	for _, a := range agents[from:min(from+room, len(agents))] {
		body = append(body, agentRow(a, ui, lw, cols))
	}
	var out []string
	if from > 0 {
		out = append(out, fmt.Sprintf("↑ %d above", from))
	}
	if below := len(agents) - from - room; below > 0 {
		out = append(out, fmt.Sprintf("↓ %d below", below))
	}
	if len(out) > 0 {
		right = append(line{sp(strings.Join(out, " · ")+" · ", stProgress)}, right...)
	}
	return box('C', zoneTitle("C · AGENTS", ui.Focus == "C"), right, body, w, h, borderFor(ui.Focus == "C"))
}

// agentColumns are the narrow row's columns over every agent, so the
// columns hold while scrolling (plan 7, item 1): the label column, the
// longest label and a space, at least 14, never so wide that the activity
// keeps fewer than 12 cells after the model's 12, the effort's and the
// right part's cells (21 for the narrow row); the effort column when any
// agent has one; the "*" footnote of a requested model.
func agentColumns(agents []*state.Agent, lw, right int) agentCols {
	cols := agentCols{label: 14}
	for _, a := range agents {
		cols.label = max(cols.label, textWidth(a.Label)+1)
		if a.Effort.Present {
			cols.effort = max(cols.effort, textWidth(a.Effort.Text)+1)
		}
		cols.requested = cols.requested || requested(a.Model)
	}
	cols.label = min(cols.label, max(14, lw-3-12-cols.effort-right-12))
	return cols
}

// timeline is the selected agent's execution timeline, w cells wide and h
// rows tall (design D3): its status for the heading's right end, or the
// rows out of view while the pane is held away from its tail, and the rows
// in view: one entry each of the agent's messages, then its history lines,
// the newest last, a rule between entries. The rows are panel P's, which
// the wheel scrolls.
func timeline(s *state.State, ui UI, w, h int, sc *Screen) (status line, rows []line) {
	a := s.Agents[ui.Selected]
	if a == nil {
		if ag := agentsByActivity(s); len(ag) > 0 {
			a = ag[0]
		}
	}
	if a == nil || h < 1 || w < 1 {
		sc.Preview = PaneView{}
		return nil, nil
	}
	var all []line
	for _, e := range a.Messages {
		if len(all) > 0 {
			all = append(all, timelineRule(w))
		}
		all = append(all, timelineEntry(e, w)...)
	}
	for _, hline := range a.History {
		if len(all) > 0 {
			all = append(all, timelineRule(w))
		}
		for _, l := range wrapTwo(hline, w) {
			all = append(all, plain(l, stGrey))
		}
	}
	if len(all) == 0 {
		all = append(all, plain("no activity yet", stGrey))
	}
	top := paneTop(ui.Preview, len(all), h, max(len(all)-h, 0))
	sc.Preview = PaneView{Rows: len(all), Room: h, Top: top}
	status = agentStatus(a)
	if ui.Preview.Hold && top < len(all)-h {
		status = heldHint(top, len(all)-top-h, "End follows", w-textWidth("EXECUTION TIMELINE")-2)
	}
	rows = all[top:min(top+h, len(all))]
	for i := range rows {
		rows[i] = rows[i].fit(w)
	}
	return status, in('P', rows)
}

// timelineRule separates two entries: a thin line across the panel in the
// frame's colour, the close-ups' dim rule.
func timelineRule(w int) line { return decoLine(strings.Repeat("─", max(w, 0)), stFrame) }

// timelineEntry is one entry's rows: the time, the role in its colour
// (zone B's speaker blue for a notice, its result green for the assistant,
// grey otherwise) padded to the role column, a grey vertical bar, then the
// text wrapped onto at most two rows, the bar drawn on both, the second cut
// with "…" when the text goes on (design D3, D7).
func timelineEntry(e state.Entry, w int) []line {
	st := stGrey
	switch e.Role {
	case "notice":
		st = stSpeaker
	case "assistant":
		st = stResult
	}
	role := e.Role + strings.Repeat(" ", max(roleWidth-textWidth(e.Role), 0)+1)
	lead := line{sp(clockSec(e.At)+"  ", stGrey), sp(role, st), deco("│ ", stGrey)}
	indent := lead.width()
	ls := wrapTwo(strings.Join(strings.Fields(e.Text), " "), max(w-indent, 8))
	out := []line{append(lead, sp(ls[0], stText))}
	if len(ls) > 1 {
		out = append(out, line{deco(strings.Repeat(" ", indent-2), stNone), deco("│ ", stGrey), sp(ls[1], stText)})
	}
	return out
}

// wrapTwo wraps text onto at most two lines of w cells; when more would
// follow, the second ends with "…".
func wrapTwo(text string, w int) []string {
	ls := wrap(text, w)
	if len(ls) == 0 {
		return []string{""}
	}
	if len(ls) > 2 {
		ls = ls[:2]
		if textWidth(ls[1])+1 <= w {
			ls[1] += "…"
		} else {
			ls[1] = cutRight(ls[1], w)
		}
	}
	return ls
}

// agentStatus is the selected agent's status for the timeline heading's
// right end: its symbol and its state in capitals, in the symbol's colour
// (the image's "✓ COMPLETED"; "● WAITING" for an approval, "◉ RUNNING").
func agentStatus(a *state.Agent) line {
	sym := listSymbol(a.Status)
	word := strings.ToUpper(string(a.Status))
	if a.Status == state.AgentWaiting {
		word = "WAITING"
	}
	return line{sym, sp(" "+word, sym.st)}
}
