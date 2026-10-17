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

// Zone D, quota and telemetry, and its details view (ui-design.md, Zone D;
// reliability.md, Telemetry sources). A value is shown as received, or as
// "N/A" with its reason: never 0 for a value no source gave.

// zoneD is quota and telemetry, rows lines high.
func zoneD(s *state.State, ui UI, w, h, rows int) []line {
	q5, q7 := quota(s, ui, "quota.5h", "5h", stTitle), quota(s, ui, "quota.7d", "7d", stProgress)
	ctx := contextValue(s, ui)
	tok := metric(s, ui, "tokens.session", tokens)
	cost := costText(s, ui)
	errs := fmt.Sprintf("errors %d", len(s.Errors))
	var body []line
	switch rows {
	case 1:
		body = []line{join(q5.short(5), q7.short(5), "ctx "+ctx.short, "tok "+tok.short, cost.short, errs)}
	case 2:
		body = []line{
			join(q5.long(20), q7.long(20), extra(s, ui), "model "+orNA(model(s), naReason(s))),
			join("context "+ctx.long, "tokens "+tok.long, "cost "+cost.long, errs, "agents "+agentCount(s), "time "+sessionTime(s, ui)),
		}
	default:
		body = []line{
			join(q5.long(30), q7.long(30), "plan "+metric(s, ui, "plan", text).long, extra(s, ui)),
			join("model "+orNA(model(s), naReason(s)), "context "+ctx.long, "tokens "+tok.long, "cost "+cost.long, "last turn "+metric(s, ui, "duration.ms", seconds).long),
			join("agents "+agentCount(s), "time "+sessionTime(s, ui), errs, retries(s), fmt.Sprintf("unrecognised events %d", s.Unrecognised)),
		}
	}
	title := "D · QUOTA & TELEMETRY"
	if rows == 1 {
		title = "D"
	}
	right := "† undocumented field"
	if f := freshness(s, ui); f != "" && rows > 1 {
		right = f + " · " + right
	}
	if ui.Focus == "D" {
		right = "Enter details · " + right
	}
	return box('D', zoneTitle(title, ui.Focus == "D"), plain(right, stGrey), body, w, h, borderFor(ui.Focus == "D"))
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

// The forms of a value.
func text(v state.Value) string    { return v.Text }
func tokens(v state.Value) string  { return count(v.V) }
func percent(v state.Value) string { return fmt.Sprintf("%.0f%%", v.V) }
func seconds(v state.Value) string {
	return elapsed(time.Unix(0, 0), time.Unix(0, 0).Add(time.Duration(v.V)*time.Millisecond), time.Time{})
}

// metric is value name in its short and long forms, or N/A with its reason.
func metric(s *state.State, ui UI, name string, f func(state.Value) string) shown {
	v := s.Metrics[name]
	if !v.Present {
		return shown{"N/A", "N/A (" + missing(s, ui, name, v, f) + ")"}
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

func orNA(v, reason string) string {
	if v == "" {
		return "N/A (" + reason + ")"
	}
	return v
}

type quotaShown struct {
	label string
	v     state.Value
	na    string // why it is N/A; "" when shown
	st    *lipgloss.Style
}

// quota is a 5-hour or 7-day window: its bar in st (5-hour violet, 7-day
// amber, red from 90%), the status word alone when the CLI gave no
// percentage, or N/A, also once its reset time has passed (ui-design.md,
// Zone D, Bars).
func quota(s *state.State, ui UI, key, label string, st *lipgloss.Style) quotaShown {
	v := s.Metrics[key]
	q := quotaShown{label: label, v: v, st: st}
	switch {
	case !v.Present:
		q.na = missing(s, ui, key, v, percent)
	case !v.ResetsAt.IsZero() && !ui.Now.IsZero() && !ui.Now.Before(v.ResetsAt):
		q.na = "window reset; waiting for the next report"
	}
	return q
}

func (q quotaShown) bar(n int) line {
	full := int(math.Round(q.v.V / 100 * float64(n)))
	full = max(0, min(n, full))
	st := q.st
	if q.v.V >= 90 {
		st = stFailed
	}
	return line{deco(strings.Repeat("▰", full), st), deco(strings.Repeat("▱", n-full), stFrame)}
}

func (q quotaShown) pct() string {
	p := percent(q.v)
	if !q.v.Documented {
		p += "†"
	}
	return p
}

func (q quotaShown) short(n int) line {
	switch {
	case q.na != "":
		return plain(q.label+" N/A", stGrey)
	case q.v.Unit != "%":
		return plain(q.label+" "+q.v.Text, stText)
	}
	return append(append(line{sp(q.label+" ", stText)}, q.bar(n)...), sp(" "+q.pct(), stText))
}

func (q quotaShown) long(n int) line {
	var l line
	switch {
	case q.na != "":
		return plain(q.label+" N/A ("+q.na+")", stGrey)
	case q.v.Unit != "%":
		l = line{sp(q.label+" "+q.v.Text, stText), sp(" (no percentage reported)", stGrey)}
	default:
		l = append(append(line{sp(q.label+" ", stText)}, q.bar(n)...), sp(" "+q.pct(), stText))
	}
	if !q.v.ResetsAt.IsZero() {
		l = append(l, sp("  resets "+q.v.ResetsAt.Local().Format("Mon 15:04"), stGrey))
	}
	return l
}

// freshness is when the quota was last received, so a stale value shows.
func freshness(s *state.State, ui UI) string {
	var at time.Time
	for _, k := range []string{"quota.5h", "quota.7d"} {
		if v := s.Metrics[k]; v.Present && v.ReceivedAt.After(at) {
			at = v.ReceivedAt
		}
	}
	if a := ago(at, ui.Now); a != "" {
		return "quota updated" + a
	}
	return ""
}

// extra is what each CLI reports beside its plan's quota: Claude Code's
// overage, Codex's credits.
func extra(s *state.State, ui UI) string {
	if s.CLI() == "codex" {
		return "credits " + metric(s, ui, "credits", text).long
	}
	return "overage " + metric(s, ui, "overage", text).long
}

func contextValue(s *state.State, ui UI) shown {
	used, win := s.Metrics["context.used"], s.Metrics["context.window"]
	if !used.Present || !win.Present || win.V <= 0 {
		name, v := "context.used", used
		if used.Present {
			name, v = "context.window", win
		}
		return shown{"N/A", "N/A (" + missing(s, ui, name, v, nil) + ")"}
	}
	p := fmt.Sprintf("%.0f%%", used.V/win.V*100)
	if !used.Documented || !win.Documented {
		p += "†"
	}
	long := p + " of " + count(win.V)
	if used.Origin == event.Derived {
		long += " (derived)"
	}
	return shown{p, long}
}

func costText(s *state.State, ui UI) shown {
	v := s.Metrics["cost.usd"]
	if !v.Present {
		return shown{"$ N/A", "N/A (" + missing(s, ui, "cost.usd", v, nil) + ")"}
	}
	return shown{fmt.Sprintf("$%.2f est.", v.V), fmt.Sprintf("$%.2f API-equiv. estimate", v.V)}
}

// agentCount is the agents running against the CLI's limit: Claude Code
// reports none; Codex's agents.max_threads is only in config/read, which is
// not called because the effective configuration it returns can hold
// secrets (reliability.md, Telemetry sources: Concurrency limit).
func agentCount(s *state.State) string {
	switch s.Session.Phase {
	case state.Spawning, state.Handshaking, "":
		return "N/A (" + naReason(s) + ")"
	}
	n, _ := activeAgents(s)
	limit := "N/A (not reported)"
	if s.CLI() == "codex" {
		limit = "N/A (not read)"
	}
	return fmt.Sprintf("%d active / limit %s", n, limit)
}

func retries(s *state.State) string {
	if s.Retries == 0 {
		return "retries N/A (none reported)"
	}
	return fmt.Sprintf("retries %d", s.Retries)
}

func sessionTime(s *state.State, ui UI) string {
	if len(s.Log) == 0 || ui.Now.IsZero() {
		return "N/A (" + naReason(s) + ")"
	}
	return elapsed(s.Log[0].At, time.Time{}, ui.Now) + " session"
}

// details is zone D full screen, from Enter on D: the last errors, newest
// first, whole; then every value with its source and age (ui-design.md,
// Zone D: Errors). A rendering of the same state, so it updates live; sc
// records the body's scrolling, as for an inspection.
func details(s *state.State, ui UI, sc *Screen) []line {
	w, h := ui.Width, ui.Height
	head := spread(line{deco(" ◀ back (Esc)   ", stTitle), deco("D · ERRORS AND TELEMETRY", stBold)},
		decoLine(fmt.Sprintf("%d errors · last %d kept ", len(s.Errors), state.MaxErrors), stGrey), w)
	ls := []line{titleBar(s, w, badge(s), nil), head, rule(w)}
	body := detailsBody(s, ui, w-2)
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
	hints := "↑↓ PgUp PgDn scroll · Home top · End bottom · Esc back"
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
	body = append(body, line{}, line{deco("TELEMETRY  ", stBold), deco("value · source · received", stGrey)})
	names := make([]string, 0, len(s.Metrics))
	for k := range s.Metrics {
		names = append(names, k)
	}
	sort.Strings(names)
	if len(names) == 0 {
		body = append(body, plain("no value received yet ("+naReason(s)+")", stGrey))
	}
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
	return body
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
