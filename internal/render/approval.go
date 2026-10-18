package render

import (
	"encoding/json"
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/Evengelius/agent-command-center/internal/state"
)

// cardLines is how many lines of the command, diff or content a pending card
// shows; the rest is counted (ui-design.md, Approvals).
const cardLines = 6

// approvalCard is an approval entry: a card while it waits, one line once
// answered (ui-design.md, Approvals). Only the tool name is the entry's
// content; the rest is drawn from the approval and is decoration.
func approvalCard(s *state.State, ui UI, i int, e state.Entry, w int) []line {
	ap := s.Approvals[e.MessageID]
	if ap == nil {
		return []line{append(line{deco("! APPROVAL ", stAttention)}, oneRow(i, e, stAttention)...)}
	}
	target, path, body := approvalDetail(ap)
	// The target gets the room left once the outcome or the clock fits, so a
	// long path never hides how the request was answered: a path loses its
	// left end, a command its right end.
	head := func(mark string, st *lipgloss.Style, tail span) line {
		l := line{deco(mark, st), deco(approvalWho(s, ap)+" · ", stGrey)}
		l = append(l, oneRow(i, e, st)...)
		room := w - l.width() - line{tail}.width() - 2
		switch {
		case target == "" || room < 1:
		case path:
			l = append(l, deco("  "+elide(target, room), stText))
		case textWidth(target) > room:
			l = append(l, deco("  "+cut(target, room-1)+"…", stText))
		default:
			l = append(l, deco("  "+target, stText))
		}
		return append(l, tail)
	}
	switch ap.Status {
	case state.Allowed:
		if ap.Question {
			out := []line{head("✓ ", stDone, deco("  answered by you "+clockSec(ap.AnsweredAt), stDone))}
			return append(out, answeredLines(QuestionsOf(ap), ap.Answers, w)...)
		}
		if ap.Answer == "acceptForSession" {
			return []line{head("✓ ", stDone, deco("  allowed for the session by you "+clockSec(ap.AnsweredAt), stDone))}
		}
		return []line{head("✓ ", stDone, deco("  allowed by you "+clockSec(ap.AnsweredAt), stDone))}
	case state.Denied:
		if ap.Problem != "" {
			return []line{
				head("✗ ", stFailed, deco("  denied automatically: request not readable", stFailed)),
				decoLine("  "+cut(ap.Problem, max(w-2, 1)), stGrey),
			}
		}
		if ap.Question && !ap.Cancelled { // denied with QuestionDenyMessage (plan 5, item 10)
			return []line{head("✗ ", stFailed, deco("  denied by you "+clockSec(ap.AnsweredAt)+" · answer in your next message", stFailed))}
		}
		return []line{head("✗ ", stFailed, deco("  denied by you "+clockSec(ap.AnsweredAt), stFailed))}
	case state.Expired:
		return []line{head("· ", stGrey, deco("  expired, not answered", stGrey))}
	}
	label := "! APPROVAL "
	if ap.Question {
		label = "? QUESTION "
	}
	out := []line{head(label, stAttentionBold, deco("  "+clock(ap.RequestedAt), stGrey))}
	if ap.Summary != "" && ap.Summary != target {
		out = append(out, indentedDeco(ap.Summary, w, stText)...)
	}
	qs := QuestionsOf(ap)
	focused := ui.Card == ap.ID
	if len(qs) > 0 {
		body = nil // the questions are drawn whole, every option, below (plan 14, item 1)
		out = append(out, questionLines(qs, pickOf(ui, ap, qs), focused, w)...)
	}
	for j, l := range body {
		if j == cardLines {
			out = append(out, decoLine(fmt.Sprintf("  │ … %s more lines", grouped(len(body)-cardLines)), stGrey))
			break
		}
		out = append(out, decoLine("  │ "+cut(l, max(w-4, 1)), stGrey))
	}
	switch {
	case ap.Status == state.Answering:
		out = append(out, decoLine("  sending answer…", stProgress))
	case focused && ap.Question:
		out = append(out, decoLine("  ▸ 1-4 choose · ← → question · a send the answers · d deny, then answer in your next message · c cancel the turn · Enter full view · Esc later", stAttention))
	case focused:
		out = append(out, decoLine("  ▸ a allow once · "+forSession(ap)+"d deny · c cancel the turn · Enter full view · Esc later · ↑ ↓ other approvals", stAttention))
	default:
		out = append(out, decoLine("  F2 to answer", stGrey))
	}
	return out
}

// forSession is the hint for s, only when the request offers it (plan 5,
// item 8).
func forSession(ap *state.Approval) string {
	if ap.ForSession && !ap.Question {
		return "s allow for the session · "
	}
	return ""
}

// approvalWho is who asks: the orchestrator or the agent's label.
func approvalWho(s *state.State, ap *state.Approval) string {
	if ap.Agent == "" {
		return "orchestrator"
	}
	if ag := s.Agents[ap.Agent]; ag != nil && ag.Label != "" {
		return ag.Label
	}
	return "agent " + ap.Agent
}

// approvalDetail reads the tool's target (a path, a command's first line)
// and the lines a card shows from the request's input; path says the target
// is a file path. An input it cannot read shows as it came.
func approvalDetail(ap *state.Approval) (target string, path bool, body []string) {
	var in map[string]any
	if len(ap.Input) == 0 || json.Unmarshal(ap.Input, &in) != nil {
		if len(ap.Input) > 0 && string(ap.Input) != "null" {
			body = strings.Split(string(ap.Input), "\n")
		}
		return "", false, body
	}
	str := func(k string) string { v, _ := in[k].(string); return v }
	for _, k := range []string{"file_path", "notebook_path", "path"} {
		if v := str(k); v != "" {
			target, path = hideUser(v), true
			break
		}
	}
	for _, k := range []string{"url", "pattern"} {
		if v := str(k); v != "" && target == "" {
			target = v
		}
	}
	switch {
	case str("command") != "":
		cmd := strings.Split(str("command"), "\n")
		if target == "" && len(cmd) == 1 {
			return cmd[0], false, nil
		}
		for _, l := range cmd {
			body = append(body, "$ "+l)
		}
	case str("new_string") != "" || str("old_string") != "":
		for _, l := range strings.Split(str("old_string"), "\n") {
			body = append(body, "- "+l)
		}
		for _, l := range strings.Split(str("new_string"), "\n") {
			body = append(body, "+ "+l)
		}
	case str("content") != "":
		body = strings.Split(strings.TrimSuffix(str("content"), "\n"), "\n")
	case ap.Question:
		if qs, ok := in["questions"].([]any); ok {
			for _, q := range qs {
				if m, ok := q.(map[string]any); ok {
					if t, ok := m["question"].(string); ok {
						body = append(body, t)
					}
				}
			}
		}
	}
	return target, path, body
}

// indentedDeco is indented, wrapped decoration.
func indentedDeco(text string, w int, st *lipgloss.Style) []line {
	var out []line
	for _, l := range wrap(text, w-2) {
		out = append(out, decoLine("  "+l, st))
	}
	return out
}
