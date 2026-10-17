package render

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/Evengelius/agent-command-center/internal/state"
)

// Agent inspection (ui-design.md, Agent inspection): one agent full screen,
// its own events only, in four tabs. A rendering of the same state, so it
// updates live.

// Tabs are the inspection's tabs, in order.
var Tabs = []string{"Messages", "Tools", "Diff", "Raw events"}

// inspect draws the inspected agent full screen; sc records the body's
// scrolling.
func inspect(s *state.State, ui UI, a *state.Agent, sc *Screen) []line {
	w, h := ui.Width, ui.Height
	ls := []line{titleBar(s, w, badge(s), nil), inspectHeader(a, ui, w), rule(w)}
	var tabs line
	tabs = append(tabs, deco("  ", stNone))
	for i, t := range Tabs {
		if i > 0 {
			tabs = append(tabs, deco(" │ ", stFrame))
		}
		st := stGrey
		if i == ui.Tab {
			st = stTitle
		}
		tabs = append(tabs, deco(t, st))
	}
	ls = append(ls, tabs, rule(w))

	var body []line
	switch ui.Tab {
	case 1:
		body = inspectTools(s, a, w-2, ui.Now)
	case 2:
		body = inspectDiff(s, a, w-2)
	case 3:
		body = inspectRaw(s, a, w-2)
	default:
		body = inspectMessages(s, a, w-2)
	}
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
	hints := "←→ 1-4 tab · ↑↓ PgUp PgDn scroll · End follow · [ ] agent · Esc back"
	if ui.InspectHold && top < bottom {
		hints = fmt.Sprintf("↓ %d more · End follow · ", bottom-top) + hints
	}
	return append(ls, keyHints(w, ui, hints))
}

// content marks a body row's text spans as row item of panel p, for
// selection and copy; the rest is decoration.
func content(p byte, item int, l line) line {
	off := 0
	for k := range l {
		r := &l[k].ref
		if r.kind == refNone {
			*r = ref{kind: refText, item: item, off: off}
			off += len(l[k].text)
		}
		r.panel = p
	}
	return l
}

func inspectHeader(a *state.Agent, ui UI, w int) line {
	m := "model N/A"
	if a.Model.Present {
		m = a.Model.Text
	}
	tok := "tok N/A"
	if a.Tokens.Present {
		tok = count(a.Tokens.V) + " tok"
	}
	l := line{deco(" ◀ back (Esc)   ", stTitle), deco(a.Label, stBold), deco(" · ", stGrey), deco(m, modelStyle(m)), deco(" · ", stGrey),
		agentSymbol(a.Status), deco(" "+statusWord(a.Status)+" "+elapsed(a.StartedAt, a.EndedAt, ui.Now), stText), deco(" · "+tok, stGrey)}
	return spread(l, decoLine("  [ ] prev/next ", stGrey), w)
}

func statusWord(st state.AgentStatus) string {
	if st == state.AgentWaiting {
		return "waiting: approval"
	}
	return strings.ToLower(string(st))
}

// timed is one timed row: the time, the text, its origin on the right.
func timed(at string, text line, origin string, w int) line {
	l := append(line{deco(at+"  ", stGrey)}, text...)
	if origin == "" {
		return l.fit(w)
	}
	return spread(l, decoLine(" "+origin, stGrey), w)
}

// detail is an indented line under a row.
func detail(text string, w int) line {
	return line{deco("          ", stNone), sp(text, stGrey)}.fit(w)
}

// inspectMessages is the agent's start, its messages as the CLI delivered
// them, and its end.
func inspectMessages(s *state.State, a *state.Agent, w int) []line {
	by := "orchestrator"
	if p := s.Agents[a.Parent]; p != nil {
		by = p.Label
	}
	start := "started by " + by
	if a.Description != "" {
		start += " · " + firstLine(a.Description)
	}
	out := []line{timed(clockSec(a.StartedAt), plain(start, stText), "observed", w)}
	for _, e := range a.Messages {
		rows := wrap(e.Text, max(w-10, 1))
		for i, r := range rows {
			if i == 0 {
				out = append(out, timed(clockSec(e.At), plain(r, stText), originLabel(e.Origin), w))
				continue
			}
			out = append(out, detail(r, w))
		}
	}
	if a.Status.Final() {
		end := line{agentSymbol(a.Status), sp(" "+statusWord(a.Status), stText)}
		if a.Summary != "" {
			end = append(end, sp(" · "+firstLine(a.Summary), stText))
		}
		out = append(out, timed(clockSec(a.EndedAt), end, originLabel(a.StatusOrigin), w))
	}
	if a.Error != "" {
		out = append(out, line{deco("          ", stNone), sp("✗ "+firstLine(a.Error), stFailed)}.fit(w))
	}
	for _, hl := range a.History {
		out = append(out, detail(hl, w))
	}
	if len(a.Messages) == 0 {
		out = append(out, detail("no message from this agent yet", w))
	}
	return out
}

// agentTools are the agent's own tool calls in the order they started.
func agentTools(s *state.State, key string) []*state.ToolCall {
	var out []*state.ToolCall
	for _, tc := range s.Tools {
		if tc.Agent == key {
			out = append(out, tc)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Seq != out[j].Seq {
			return out[i].Seq < out[j].Seq
		}
		return out[i].ID < out[j].ID
	})
	return out
}

// inspectTools is each call with its input, status, duration and result,
// and the approvals asked for it.
func inspectTools(s *state.State, a *state.Agent, w int, now time.Time) []line {
	var out []line
	used := map[string]bool{}
	for _, tc := range agentTools(s, a.Key) {
		head := line{toolSymbol(s, tc, deco("◉", stProgress)), sp(" "+tc.Name, stText)}
		target, _, body := approvalDetail(&state.Approval{Input: tc.Input})
		if t := toolTarget(tc.Input); t != "" {
			head = append(head, sp("  "+t, stText))
		}
		head = append(head, sp("  "+strings.ToLower(string(tc.Status))+" · "+elapsed(tc.StartedAt, tc.EndedAt, now), stGrey))
		out = append(out, timed(clockSec(tc.StartedAt), head, originLabel(tc.Origin), w))
		if target == "" && len(body) == 0 && toolTarget(tc.Input) == "" && len(tc.Input) > 0 && string(tc.Input) != "null" {
			body = []string{string(tc.Input)}
		}
		for _, l := range body {
			out = append(out, detail(l, w))
		}
		if tc.Output != "" {
			for i, l := range strings.Split(strings.TrimRight(tc.Output, "\n"), "\n") {
				mark := "  "
				if i == 0 {
					mark = "→ "
				}
				out = append(out, detail(mark+l, w))
			}
		}
		for _, id := range s.Queue {
			if ap := s.Approvals[id]; ap != nil && ap.ToolID == tc.ID && tc.ID != "" {
				used[id] = true
				out = append(out, approvalRow(ap, originLabel(tc.Origin), w))
			}
		}
	}
	for _, id := range s.Queue {
		if ap := s.Approvals[id]; ap != nil && ap.Agent == a.Key && !used[id] {
			out = append(out, approvalRow(ap, "observed", w))
		}
	}
	if len(out) == 0 {
		out = append(out, detail("no tool call by this agent", w))
	}
	return out
}

// approvalRow is an approval under its tool call and how it was answered.
func approvalRow(ap *state.Approval, origin string, w int) line {
	l := line{deco("          ", stNone)}
	switch ap.Status {
	case state.Allowed:
		l = append(l, deco("✓", stDone), sp(" approval "+ap.Tool+" · allowed by you "+clockSec(ap.AnsweredAt), stText))
	case state.Denied:
		l = append(l, deco("✗", stFailed), sp(" approval "+ap.Tool+" · denied by you "+clockSec(ap.AnsweredAt), stText))
	case state.Expired:
		l = append(l, deco("·", stGrey), sp(" approval "+ap.Tool+" · expired, not answered", stGrey))
	default:
		l = append(l, deco("!", stAttention), sp(" approval "+ap.Tool+" · waiting for you since "+clockSec(ap.RequestedAt), stAttention))
	}
	return spread(l, decoLine(" "+origin, stGrey), w)
}

// inspectDiff is the agent's file changes: Codex fileChange diffs as
// observed, Claude Code's Edit, MultiEdit and Write inputs as derived.
func inspectDiff(s *state.State, a *state.Agent, w int) []line {
	var out []line
	for _, tc := range agentTools(s, a.Key) {
		switch tc.Name {
		case "fileChange":
			var in struct {
				Changes []struct {
					Path string          `json:"path"`
					Kind json.RawMessage `json:"kind"`
					Diff string          `json:"diff"`
				} `json:"changes"`
			}
			if json.Unmarshal(tc.Input, &in) != nil {
				continue
			}
			for _, c := range in.Changes {
				var k struct {
					Type string `json:"type"`
				}
				kind := strings.Trim(string(c.Kind), `"`)
				if json.Unmarshal(c.Kind, &k) == nil && k.Type != "" {
					kind = k.Type
				}
				out = append(out, timed(clockSec(tc.StartedAt), line{sp(hideUser(c.Path), stBold), sp("  "+kind, stGrey)}, originLabel(tc.Origin), w))
				for _, l := range strings.Split(strings.TrimRight(c.Diff, "\n"), "\n") {
					out = append(out, diffLine(l, w))
				}
			}
		case "Edit", "MultiEdit", "Write":
			var in struct {
				Path    string `json:"file_path"`
				Old     string `json:"old_string"`
				New     string `json:"new_string"`
				Content string `json:"content"`
				Edits   []struct {
					Old string `json:"old_string"`
					New string `json:"new_string"`
				} `json:"edits"`
			}
			if json.Unmarshal(tc.Input, &in) != nil {
				continue
			}
			out = append(out, timed(clockSec(tc.StartedAt), line{sp(hideUser(in.Path), stBold), sp("  "+tc.Name, stGrey)}, "derived", w))
			if tc.Name == "Write" {
				for _, l := range strings.Split(strings.TrimRight(in.Content, "\n"), "\n") {
					out = append(out, diffLine("+"+l, w))
				}
				continue
			}
			edits := in.Edits
			if tc.Name == "Edit" {
				edits = append(edits[:0:0], struct {
					Old string `json:"old_string"`
					New string `json:"new_string"`
				}{in.Old, in.New})
			}
			for _, e := range edits {
				for _, l := range strings.Split(e.Old, "\n") {
					out = append(out, diffLine("-"+l, w))
				}
				for _, l := range strings.Split(e.New, "\n") {
					out = append(out, diffLine("+"+l, w))
				}
			}
		}
	}
	if len(out) == 0 {
		out = append(out, detail("no file change by this agent", w))
	}
	return out
}

func diffLine(l string, w int) line {
	st := stText
	switch {
	case strings.HasPrefix(l, "+"):
		st = stDone
	case strings.HasPrefix(l, "-"):
		st = stFailed
	}
	return line{deco("          ", stNone), sp(l, st)}.fit(w)
}

// inspectRaw is the agent's raw event lines, as received.
func inspectRaw(s *state.State, a *state.Agent, w int) []line {
	var out []line
	for _, ev := range s.Log {
		if ev.Agent != a.Key {
			continue
		}
		out = append(out, line{deco(clockSec(ev.At)+"  ", stGrey), deco(padTo(string(ev.Kind), 22)+" ", stGrey), sp(firstLine(ev.Raw), stText)}.fit(w))
	}
	if len(out) == 0 {
		out = append(out, detail("no raw event of this agent in the log", w))
	}
	return out
}
