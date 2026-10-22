package render

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"charm.land/lipgloss/v2"

	"github.com/Evengelius/agent-command-center/internal/event"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// The telemetry pane of zone A (plan 24, brief amendment 22; ui-design.md,
// Zone A: Telemetry) and its details view (ui-design.md, Zone A: Errors and
// telemetry; reliability.md, Telemetry sources). A value is shown as
// received, or as "N/A" with its reason: never 0 for a value no source gave.
// Zone D, which held these values until plan 23, is gone.

// The pane's nodes, after the tree's in zone A's cursor order (plan 24,
// design D6): Enter on any of them opens the details view; ← → fold the two
// sections.
const (
	NodeTelemetry = "u:telemetry"
	NodeUsage     = "u:usage"
	NodeQuota     = "u:quota"
)

// TelemetryNodes are the pane's nodes in order, with the owner's folds.
func TelemetryNodes(ui UI) []Node {
	return []Node{
		{ID: NodeTelemetry},
		{ID: NodeUsage, Foldable: true, Folded: folded(ui, NodeUsage, false)},
		{ID: NodeQuota, Foldable: true, Folded: folded(ui, NodeQuota, false)},
	}
}

// teleForm is one of the pane's three forms (design D6): the largest that
// leaves the tree its rows is drawn.
type teleForm int

const (
	formFull    teleForm = iota // the image's: blank rows, three rows a window, two rows of notes
	formCompact                 // no blank rows, one row of notes
	formTight                   // the usage on two rows, one row a window without its bar, one row of notes
)

// The pane's indent under a section's heading, and the least width the
// usage section draws its three columns at (design D7).
const (
	teleIndent  = "  "
	columnsFrom = 41
)

// telemetryPane draws the pane w cells wide in at most limit rows, the
// largest form that fits; the tight form may take up to most rows, the
// tree keeping its last few, and is cut short when even that is too much.
// Each row comes with the node it belongs to ("" for none): the heading
// rows, so that the cursor and a click find them.
func telemetryPane(s *state.State, ui UI, w, limit, most int) ([]line, []string) {
	limit, most = max(limit, 0), max(most, 0)
	for _, f := range []teleForm{formFull, formCompact, formTight} {
		rows, nodes := teleRows(s, ui, w, f)
		if len(rows) <= limit {
			return rows, nodes
		}
		if f == formTight {
			n := min(len(rows), most)
			return rows[:n], nodes[:n]
		}
	}
	return nil, nil
}

// teleRows is the pane in form f: a rule, the heading, the usage section,
// the quota section, then the notes after a rule in the full form.
func teleRows(s *state.State, ui UI, w int, f teleForm) ([]line, []string) {
	var rows []line
	var nodes []string
	add := func(l line, node string) {
		rows = append(rows, l)
		nodes = append(nodes, node)
	}
	add(teleRule(w), "")
	// The way to the rest at the heading's right end, as zone D's edge said
	// it: Enter on the heading, or a click, opens the details view.
	add(spread(line{deco("✦ ", stTitle), deco("TELEMETRY", stTitle)}, decoLine("Enter details", stGrey), w), NodeTelemetry)
	usageOpen := !folded(ui, NodeUsage, false)
	add(sectionHead("SESSION USAGE", usageOpen), NodeUsage)
	if usageOpen {
		for _, l := range usageRows(s, ui, w, f) {
			add(l, "")
		}
	}
	if f == formFull {
		add(line{}, "")
	}
	quotaOpen := !folded(ui, NodeQuota, false)
	add(sectionHead("QUOTA WINDOWS", quotaOpen), NodeQuota)
	if quotaOpen {
		for i, q := range windows(s, ui) {
			if f == formFull && i > 0 {
				add(line{}, "")
			}
			for _, l := range q.rows(ui, w, f) {
				add(l, "")
			}
		}
	}
	n := 1
	if f == formFull {
		add(teleRule(w), "")
		n = 2
	}
	for _, l := range noteRows(s, ui, w, n) {
		add(l, "")
	}
	return rows, nodes
}

// teleRule is a rule across the pane, decoration.
func teleRule(w int) line { return decoLine(strings.Repeat("─", max(w, 0)), stFrame) }

// sectionHead is a section's heading with its fold mark, ▾ open, ▸ folded.
func sectionHead(name string, open bool) line {
	mark := "▸ "
	if open {
		mark = "▾ "
	}
	return line{deco(mark, stFoldMark), deco(name, stBold)}
}

// usageCol is one of the usage section's three values: its label, the value,
// and its support, the context's bar or a scope label, or the reason the
// value is N/A.
type usageCol struct {
	label, value, support string
	na                    bool
	bar                   float64 // the context's used share of its window; -1 for none
}

// usageCols are the context, the tokens and the cost (design D7).
func usageCols(s *state.State, ui UI, w int) [3]usageCol {
	ctx := usageCol{label: "CONTEXT", bar: -1}
	used, win := s.Metrics["context.used"], s.Metrics["context.window"]
	if used.Present && win.Present && win.V > 0 {
		ctx.value = fmt.Sprintf("%.0f%% of %s", used.V/win.V*100, count(win.V))
		if !used.Documented || !win.Documented {
			ctx.value += "†"
		}
		ctx.bar = used.V / win.V
	} else {
		name, v := "context.used", used
		if used.Present {
			name, v = "context.window", win
		}
		ctx.value, ctx.support, ctx.na = "N/A", missing(s, ui, name, v, nil), true
	}
	tok := usageCol{label: "TOKENS", support: "session total", bar: -1}
	if v := s.Metrics["tokens.session"]; v.Present {
		tok.value = count(v.V)
		if !v.Documented {
			tok.value += "†"
		}
	} else {
		tok.value, tok.support, tok.na = "N/A", missing(s, ui, "tokens.session", v, tokens), true
	}
	cost := usageCol{label: "EST. COST", support: "API-equivalent", bar: -1}
	if w < 45 {
		cost.support = "API-equiv."
	}
	if v := s.Metrics["cost.usd"]; v.Present {
		cost.value = fmt.Sprintf("$%.2f", v.V)
		if !v.Documented {
			cost.value += "†"
		}
	} else {
		cost.value, cost.support, cost.na = "N/A", missing(s, ui, "cost.usd", v, nil), true
	}
	return [3]usageCol{ctx, tok, cost}
}

// usageRows is the usage section's body in form f: three columns from
// columnsFrom cells, else one row a value; in the tight form one row for
// the three.
func usageRows(s *state.State, ui UI, w int, f teleForm) []line {
	cols := usageCols(s, ui, w)
	indent := deco(teleIndent, stNone)
	if f == formTight {
		// Two rows: the context with its bar, then the tokens and the cost.
		first := line{indent, sp("Context ", stGrey), sp(cols[0].value, stBold)}
		if cols[0].bar >= 0 {
			first = append(first, deco(" ", stNone))
			first = append(first, usageBar(cols[0].bar, max(w-first.width(), 4))...)
		}
		second := line{indent, sp("Tokens ", stGrey), sp(cols[1].value, stBold), deco(" · ", stGrey), sp("Cost ", stGrey), sp(cols[2].value, stBold)}
		return []line{first.fit(w), second.fit(w)}
	}
	inner := w - len(teleIndent)
	if w < columnsFrom {
		var out []line
		for _, c := range cols {
			label := c.label[:1] + strings.ToLower(c.label[1:])
			l := line{indent, sp(label+" ", stGrey), sp(c.value, stBold)}
			if c.bar >= 0 {
				l = append(l, deco(" ", stNone))
				l = append(l, usageBar(c.bar, max(inner-l.width()+len(teleIndent), 4))...)
			} else {
				l = append(l, sp(" "+c.support, stGrey))
			}
			out = append(out, l.fit(w))
		}
		return out
	}
	// Three columns: the context's at least 10 cells, the tokens' 13, the
	// cost's its support; the cells left widen the context's bar.
	ws := [3]int{max(10, textWidth(cols[0].value)), max(13, textWidth(cols[1].value), textWidth(cols[1].support)),
		max(textWidth(cols[2].label), textWidth(cols[2].value), textWidth(cols[2].support))}
	if cols[0].na {
		ws[0] = max(ws[0], textWidth(cols[0].support))
	}
	if extra := inner - ws[0] - ws[1] - ws[2] - 6; extra > 0 {
		ws[0] += extra
	}
	sep := deco(" │ ", stFrame)
	row := func(cells [3]line) line {
		l := line{indent}
		for i, c := range cells {
			if i > 0 {
				l = append(l, sep)
			}
			l = append(l, c.fit(ws[i])...)
		}
		return l.fit(w)
	}
	var labels, values, support [3]line
	for i, c := range cols {
		labels[i] = plain(c.label, stGrey)
		values[i] = plain(c.value, stBold)
		support[i] = plain(c.support, stGrey)
		if c.bar >= 0 {
			support[i] = usageBar(c.bar, ws[i])
		}
	}
	return []line{row(labels), row(values), row(support)}
}

// usageBar is the context's bar, share of n cells filled, in the title
// colour on the track (design D7).
func usageBar(share float64, n int) line { return fillBar(share*100, n, stTitle) }

// fillBar is n cells of "█" and "░": the fill round(p × n), clamped to the
// bar, in st, the track in the thinking card's track colour (design D8).
func fillBar(p float64, n int, st *lipgloss.Style) line {
	n = max(n, 0)
	full := int(math.Round(p / 100 * float64(n)))
	full = max(0, min(n, full))
	return line{deco(strings.Repeat("█", full), st), deco(strings.Repeat("░", n-full), stThinkTrack)}
}

// quotaWindow is one quota window the pane shows: its metric, its label,
// its value and, when it shows no value, why.
type quotaWindow struct {
	key, label string
	v          state.Value
	na         string // why it is N/A; "" when a value shows
}

// windows are the quota windows in the pane's order (design D8): the
// 5-hour and 7-day windows always, then each model-scoped weekly window the
// CLI reported, then any other window by its duration. A window never
// reported, but documented for the CLI, reads N/A with its reason; a
// model-scoped one exists only once reported.
func windows(s *state.State, ui UI) []quotaWindow {
	keys := []string{"quota.5h", "quota.7d"}
	var scoped, other []string
	for k := range s.Metrics {
		switch {
		case strings.HasPrefix(k, "quota.7d."):
			scoped = append(scoped, k)
		case k == "quota.5h" || k == "quota.7d" || k == "quota.status":
		case strings.HasPrefix(k, "quota.") && strings.HasSuffix(k, "m"):
			other = append(other, k)
		}
	}
	sort.Strings(scoped)
	sort.Slice(other, func(i, j int) bool { return windowMinutes(other[i]) < windowMinutes(other[j]) })
	keys = append(append(keys, scoped...), other...)
	out := make([]quotaWindow, 0, len(keys))
	for _, k := range keys {
		v := s.Metrics[k]
		q := quotaWindow{key: k, label: windowLabel(k), v: v}
		switch {
		case !v.Present:
			q.na = missing(s, ui, k, v, percent)
		case !v.ResetsAt.IsZero() && !ui.Now.IsZero() && !ui.Now.Before(v.ResetsAt):
			q.na = "window reset; waiting for the next update" // the clock tick's words (state.Tick), a frame before it
		}
		out = append(out, q)
	}
	return out
}

// windowMinutes reads the duration of a "quota.<N>m" metric.
func windowMinutes(key string) int {
	n, _ := strconv.Atoi(strings.TrimSuffix(strings.TrimPrefix(key, "quota."), "m"))
	return n
}

// windowLabel names a quota window as the pane shows it.
func windowLabel(key string) string {
	switch {
	case key == "quota.5h":
		return "Session (5hr)"
	case key == "quota.7d":
		return "Weekly (7 day)"
	case strings.HasPrefix(key, "quota.7d."):
		m := strings.TrimPrefix(key, "quota.7d.")
		return strings.ToUpper(m[:1]) + m[1:] + " limit"
	}
	n := windowMinutes(key)
	switch {
	case n > 0 && n%1440 == 0:
		return fmt.Sprintf("%d-day window", n/1440)
	case n > 0 && n%60 == 0:
		return fmt.Sprintf("%dh window", n/60)
	}
	return fmt.Sprintf("%d min window", n)
}

// pct is the window's percentage with "†" when undocumented.
func (q quotaWindow) pct() string {
	p := percent(q.v)
	if !q.v.Documented {
		p += "†"
	}
	return p
}

// right is what ends the window's label row: its percentage, its status
// word when the CLI gave no percentage, or N/A.
func (q quotaWindow) right() line {
	switch {
	case q.na != "":
		return plain("N/A", stGrey)
	case q.v.Unit != "%":
		return plain(q.v.Text, stText)
	}
	return plain(q.pct(), stText)
}

// second is the window's second line of text: its reset countdown, or why
// it shows no percentage.
func (q quotaWindow) second(ui UI) string {
	switch {
	case q.na != "":
		return q.na
	case q.v.Unit != "%":
		return "no percentage reported"
	}
	return resetIn(q.v.ResetsAt, ui.Now)
}

// barStyle is the fill's colour: the title's, red from 90% (ASM-13).
func (q quotaWindow) barStyle() *lipgloss.Style {
	if q.v.V >= 90 {
		return stFailed
	}
	return stTitle
}

// rows draws the window in form f, w cells wide: its label row with the
// percentage at the right end, its bar, its reset countdown (design D8);
// in the tight form one row, the label, the percentage and the countdown.
func (q quotaWindow) rows(ui UI, w int, f teleForm) []line {
	indent := deco(teleIndent, stNone)
	inner := w - len(teleIndent)
	withBar := q.na == "" && q.v.Unit == "%"
	if f == formTight {
		l := append(line{indent, sp(q.label+"  ", stText)}, q.right()...)
		return []line{append(l, sp(" · "+strings.TrimPrefix(q.second(ui), "Resets in "), stGrey)).fit(w)}
	}
	out := []line{spread(line{indent, sp(q.label, stText)}, q.right(), w)}
	if withBar {
		out = append(out, append(line{indent}, fillBar(q.v.V, inner, q.barStyle())...))
	}
	return append(out, line{indent, sp(q.second(ui), stGrey)})
}

// resetIn is the countdown to a window's reset from the frame's time, as
// the pane shows it: "Resets in 5d 3h", "Resets in 4h 12m", "Resets in
// 12m", "Resets in <1m"; "Reset time unavailable" without a reset time.
// A reset that passed reads "<1m" until the clock tick turns the window
// N/A, never a negative time (design D8).
func resetIn(reset, now time.Time) string {
	if reset.IsZero() || now.IsZero() {
		return "Reset time unavailable"
	}
	return "Resets in " + countdown(reset.Sub(now))
}

// countdown writes d in its two largest units: "5d 3h", "5d", "4h 12m",
// "4h", "12m", "<1m".
func countdown(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	days, hours, mins := int(d.Hours())/24, int(d.Hours())%24, int(d.Minutes())%60
	switch {
	case days > 0 && hours > 0:
		return fmt.Sprintf("%dd %dh", days, hours)
	case days > 0:
		return fmt.Sprintf("%dd", days)
	case hours > 0 && mins > 0:
		return fmt.Sprintf("%dh %dm", hours, mins)
	case hours > 0:
		return fmt.Sprintf("%dh", hours)
	case mins > 0:
		return fmt.Sprintf("%dm", mins)
	}
	return "<1m"
}

// noteRows are the pane's notes (design D9): "ⓘ" then each part that
// applies, joined by " · ", on at most n rows, the last cut with "…"; with
// no part, "ⓘ values from the CLI's events".
func noteRows(s *state.State, ui UI, w, n int) []line {
	parts := noteParts(s, ui)
	if len(parts) == 0 {
		parts = []line{plain("values from the CLI's events", stGrey)}
	}
	var out []line
	cur := line{deco("ⓘ ", stGrey)}
	for i, p := range parts {
		next := p
		if i > 0 {
			next = append(line{deco(" · ", stGrey)}, p...)
		}
		if cur.width()+next.width() > w && len(out) < n-1 && len(cur) > 1 {
			out = append(out, cur.fit(w))
			cur = line{deco(teleIndent, stNone)}
			next = p
		}
		cur = append(cur, next...)
	}
	return append(out, cur.fit(w))
}

// noteParts are the notes' parts, each only when it applies: what went
// wrong first, then where the values come from, then the way to the rest.
func noteParts(s *state.State, ui UI) []line {
	var parts []line
	say := func(text string, st *lipgloss.Style) { parts = append(parts, plain(text, st)) }
	if n := len(s.Errors); n > 0 {
		say(fmt.Sprintf("errors %d", n), stFailed)
	}
	if s.Retries > 0 {
		say(fmt.Sprintf("retries %d", s.Retries), stProgress)
	}
	if s.Unrecognised > 0 {
		say(fmt.Sprintf("unrecognised %d", s.Unrecognised), stProgress)
	}
	if f := freshness(s, ui); f != "" {
		say(f, stGrey)
	}
	if v := s.Metrics["context.used"]; v.Present && v.Origin == event.Derived {
		say("context derived from usage", stGrey)
	}
	if v := s.Metrics["tokens.session"]; v.Present && v.Origin == event.Derived {
		say("tokens summed over threads", stGrey)
	}
	if undocumentedShown(s) {
		say("† undocumented field", stGrey)
	}
	if s.Metrics["cost.usd"].Present {
		say("cost at API rates", stGrey) // never the subscription's charge (AM2 D5)
	}
	if v := s.Metrics["overage"]; v.Present && v.Text == "in use" {
		say("overage in use", stProgress)
	}
	if v := s.Metrics["plan"]; v.Present && v.Text != "" {
		say("plan "+v.Text, stGrey)
	}
	if v := s.Metrics["credits"]; v.Present && v.Text != "" {
		say("credits "+v.Text, stGrey)
	}
	return parts
}

// undocumentedShown says a value the pane draws is marked "†".
func undocumentedShown(s *state.State) bool {
	for k, v := range s.Metrics {
		if v.Present && !v.Documented && (strings.HasPrefix(k, "quota.") && k != "quota.status" ||
			k == "context.used" || k == "context.window" || k == "tokens.session" || k == "cost.usd") {
			return true
		}
	}
	return false
}

// The forms of a value.
func text(v state.Value) string    { return v.Text }
func tokens(v state.Value) string  { return count(v.V) }
func percent(v state.Value) string { return fmt.Sprintf("%.0f%%", v.V) }
func seconds(v state.Value) string {
	return elapsed(time.Unix(0, 0), time.Unix(0, 0).Add(time.Duration(v.V)*time.Millisecond), time.Time{})
}

// missing says why value name is N/A: the source's own reason, with the last
// value and its age when there was one; else what has not happened yet, by
// CLI (reliability.md, Telemetry sources: the two value tables).
func missing(s *state.State, ui UI, name string, v state.Value, f func(state.Value) string) string {
	if v.Reason != "" {
		if v.Last && f != nil {
			return v.Reason + "; last " + f(v) + ago(v.ReceivedAt, ui.Now)
		}
		return v.Reason
	}
	codex := s.CLI() == "codex"
	// What the CLI never reports stays so, whatever the session's state.
	switch {
	case codex && name == "cost.usd":
		return "Codex reports none"
	case !codex && name == "plan":
		return "not reported"
	}
	if s.Session.Phase != state.Ready {
		return naReason(s)
	}
	switch name {
	case "quota.5h", "quota.7d", "credits", "plan":
		return "no report yet"
	case "overage":
		return "not reported"
	case "context.used", "context.window":
		if codex {
			return "no model call yet"
		}
		// Claude Code's comes with a turn's result, or the get_context_usage
		// reply read after it: once a turn has started, what is missing is
		// that result (the owner's live review, 2026-10-20: "no turn yet"
		// read on through the first turn).
		if s.Turn.Count > 0 || s.Turn.Phase != state.Idle {
			return "no result yet"
		}
		return "no turn yet"
	case "tokens.session":
		if codex {
			return "no update yet"
		}
		return "no result yet"
	case "cost.usd":
		return "no result yet"
	case "duration.ms":
		if codex {
			return "no turn ended yet"
		}
		return "no result yet"
	}
	return "not reported yet"
}

// ago is " 4m 02s ago", or "" without both times.
func ago(at, now time.Time) string {
	if at.IsZero() || now.IsZero() || now.Before(at) {
		return ""
	}
	return " " + elapsed(at, time.Time{}, now) + " ago"
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

// freshness is when the quota was last received, so a stale value shows.
func freshness(s *state.State, ui UI) string {
	var at time.Time
	for k, v := range s.Metrics {
		if strings.HasPrefix(k, "quota.") && k != "quota.status" && v.Present && v.ReceivedAt.After(at) {
			at = v.ReceivedAt
		}
	}
	if a := ago(at, ui.Now); a != "" {
		return "quota updated" + a
	}
	return ""
}

// agentCount is the agents running against the CLI's limit: Claude Code
// reports none; Codex's comes from config/read, of which only the limit is
// kept (probe 2026-10-17, Codex CLI 0.163.0; reliability.md, Telemetry
// sources: Concurrency limit).
func agentCount(s *state.State) string {
	switch s.Session.Phase {
	case state.Spawning, state.Handshaking, "":
		return "N/A (" + naReason(s) + ")"
	}
	n, _ := activeAgents(s)
	limit := "N/A (not reported)"
	if s.CLI() == "codex" {
		limit = "N/A (not read)"
		if v := s.Metrics[event.LimitMetric]; v.Present {
			switch v.Text {
			case event.LimitNotSet:
				limit = "N/A (not set; default not reported)"
			case event.LimitPerBackend:
				limit = "N/A (set per backend; backend not reported)"
			default:
				limit = fmt.Sprintf("%.0f (config)", v.V)
			}
		}
	}
	return fmt.Sprintf("%d active / limit %s", n, limit)
}

func retries(s *state.State) string {
	if s.Retries == 0 {
		return "N/A (none reported)"
	}
	return fmt.Sprint(s.Retries)
}

// sessionTime counts from the first event the state applied, not from the
// oldest the log keeps, which moves once MaxLog events were applied (plan
// 21, item 1).
func sessionTime(s *state.State, ui UI) string {
	if s.Applied == 0 || ui.Now.IsZero() {
		return "N/A (" + naReason(s) + ")"
	}
	return elapsed(s.FirstAt, time.Time{}, ui.Now) + " session"
}

// details is the telemetry full screen, from Enter on the pane's headings
// (plan 24) as from Enter on zone D before: the last errors, newest first,
// whole; then every value with its source and age (ui-design.md, Zone A:
// Errors and telemetry). A rendering of the same state, so it updates live;
// sc records the body's scrolling, as for an inspection.
func details(s *state.State, ui UI, sc *Screen) []line {
	w := ui.Width
	head := spread(line{deco(" ◀ back (Esc)   ", stTitle), deco("ERRORS AND TELEMETRY", stBold)},
		decoLine(fmt.Sprintf("%d errors · last %d kept ", len(s.Errors), state.MaxErrors), stGrey), w)
	return fullScreen(s, ui, sc, head, detailsBody(s, ui, w-2), scrollKeys)
}

// scrollKeys are the hints of a view that only scrolls.
const scrollKeys = "↑↓ PgUp PgDn scroll · Home top · End bottom · Esc back"

// fullScreen draws a view that covers the dashboard: the title bar, head, a
// rule, then body scrolled as ui.InspectTop and ui.InspectHold say; at the
// bottom unless held. keys are its key hints.
func fullScreen(s *state.State, ui UI, sc *Screen, head line, body []line, keys string) []line {
	w, h := ui.Width, ui.Height
	ls := []line{titleBar(s, w, badge(s), nil), head, rule(w)}
	room := max(h-len(ls)-1, 1)
	bottom := max(len(body)-room, 0)
	top := bottom
	if ui.InspectHold {
		top = min(max(ui.InspectTop, 0), bottom)
	}
	sc.Inspect = ConvView{Rows: len(body), Room: room, Top: top, Y: len(ls), Held: ui.InspectHold}
	for i := top; i < top+room; i++ {
		var l line
		if i < len(body) {
			l = body[i]
		}
		ls = append(ls, content('I', i, append(line{deco(" ", stNone)}, l...).fit(w)))
	}
	hints := keys
	if top < bottom {
		hints = fmt.Sprintf("↓ %d more · ", bottom-top) + hints
	}
	return append(ls, keyHints(w, ui, hints))
}

// errorIndent is where an error's message starts: the time and who.
const errorIndent = 22

func detailsBody(s *state.State, ui UI, w int) []line {
	body := []line{{deco("ERRORS  ", stBold), deco("newest first", stGrey)}}
	if len(s.Errors) == 0 {
		body = append(body, plain("no errors", stGrey))
	}
	for i := len(s.Errors) - 1; i >= 0; i-- {
		e := s.Errors[i]
		who := "session"
		if e.Agent != "" {
			who = e.Agent
			if a := s.Agents[e.Agent]; a != nil && a.Label != "" {
				who = a.Label
			}
		}
		if e.Retry {
			who += " · retry"
		}
		indent := line{deco(strings.Repeat(" ", errorIndent), stNone)}
		lead := line{deco(clockSec(e.At)+"  ", stGrey), deco(padTo(who, errorIndent-10), stText)}
		if textWidth(who) > errorIndent-11 {
			// A long label keeps its own row, whole.
			body = append(body, append(lead[:1:1], deco(who, stText)).fit(w))
			lead = indent
		}
		for j, l := range wrap(e.Message, max(w-errorIndent, 10)) {
			if j > 0 {
				lead = indent
			}
			body = append(body, append(lead, sp(l, stFailed)))
		}
	}
	// The lines the mapper could not map, newest first: when, what the
	// mapper called them, why when it said, and the line itself cut to the
	// width (plan 25, D2). Zone A's note "unrecognised N" says how many;
	// this says which, so a CLI change is read here, not guessed.
	body = append(body, line{}, line{deco("UNRECOGNISED  ", stBold), deco("lines the dashboard could not map, newest first", stGrey)})
	if len(s.UnrecognisedLast) == 0 {
		body = append(body, plain("none", stGrey))
	}
	for i := len(s.UnrecognisedLast) - 1; i >= 0; i-- {
		u := s.UnrecognisedLast[i]
		head := line{deco(clockSec(u.At)+"  ", stGrey), deco(u.Type, stText)}
		if u.Reason != "" {
			head = append(head, deco(" · "+u.Reason, stGrey))
		}
		body = append(body, head.fit(w))
		body = append(body, line{deco(strings.Repeat(" ", errorIndent), stNone), deco(cutRight(u.Line, max(w-errorIndent, 10)), stGrey)}.fit(w))
	}
	body = append(body, line{}, line{deco("TELEMETRY  ", stBold), deco("value · source · received", stGrey)})
	// Every value received, and the documented values the CLI has not
	// given yet, each N/A with its reason, as zone D listed them.
	seen := map[string]bool{}
	for k := range s.Metrics {
		seen[k] = true
	}
	for _, k := range documentedValues(s) {
		seen[k] = true
	}
	names := make([]string, 0, len(seen))
	for k := range seen {
		names = append(names, k)
	}
	sort.Strings(names)
	for _, k := range names {
		v := s.Metrics[k]
		val := "N/A (" + missing(s, ui, k, v, valueForm(k)) + ")"
		st := stGrey
		if v.Present {
			val, st = valueForm(k)(v), stText
			if !v.Documented {
				val += "†"
			}
			if v.Origin == event.Derived {
				val += " (derived)"
			}
		}
		src := v.Source
		if src == "" {
			src = "no source"
		}
		at := v.ReceivedAt
		if !v.Present {
			at = v.ReasonAt
		}
		body = append(body, line{deco(padTo(k, 20), stGrey), sp(val, st), deco("  · "+src+ago(at, ui.Now), stGrey)}.fit(w))
	}
	// The counters zone D drew until plan 23, derived from the state rather
	// than received: the agents against the CLI's limit, the session's time,
	// the retries and the unrecognised events.
	body = append(body, line{}, line{deco("COUNTERS  ", stBold), deco("derived from the events", stGrey)})
	for _, c := range [][2]string{
		{"agents", agentCount(s)},
		{"session time", sessionTime(s, ui)},
		{"retries", retries(s)},
		{"unrecognised events", fmt.Sprint(s.Unrecognised)},
	} {
		st := stText
		if strings.HasPrefix(c[1], "N/A") {
			st = stGrey
		}
		body = append(body, line{deco(padTo(c[0], 20), stGrey), sp(c[1], st)}.fit(w))
	}
	return body
}

// cutRight keeps the start of text within w cells and ends it with "…" when
// it was longer; a long line is first cut to twice w runes, so a 64 KiB
// line costs no more than a short one.
func cutRight(text string, w int) string {
	if textWidth(text) <= w {
		return text
	}
	if w <= 1 {
		return "…"
	}
	rs := []rune(text)
	if len(rs) > 2*w {
		rs = rs[:2*w]
	}
	for len(rs) > 0 && textWidth(string(rs))+1 > w {
		rs = rs[:len(rs)-1]
	}
	return string(rs) + "…"
}

// documentedValues are the values the session's CLI documents, listed in the
// details view even before they are received (reliability.md, Telemetry
// sources: the two value tables).
func documentedValues(s *state.State) []string {
	names := []string{"quota.5h", "quota.7d", "context.used", "context.window", "tokens.session", "cost.usd", "duration.ms"}
	if s.CLI() == "codex" {
		return append(names, "plan", "credits")
	}
	return append(names, "overage")
}

// valueForm is how value name reads.
func valueForm(name string) func(state.Value) string {
	switch {
	case strings.HasPrefix(name, "quota.") && name != "quota.status":
		return func(v state.Value) string {
			if v.Unit != "%" {
				return v.Text
			}
			return percent(v)
		}
	case name == "duration.ms":
		return seconds
	case name == "cost.usd":
		return func(v state.Value) string { return fmt.Sprintf("$%.4f API-equiv. estimate", v.V) }
	}
	return func(v state.Value) string {
		if v.Text != "" {
			return v.Text
		}
		return grouped(int(math.Round(v.V))) + unitSuffix(v.Unit)
	}
}

func unitSuffix(u string) string {
	if u == "" || u == "%" {
		return u
	}
	return " " + u
}
