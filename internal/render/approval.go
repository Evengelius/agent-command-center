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
	target, body := approvalDetail(ap)
	head := func(mark string, st lipgloss.Style) line {
		l := line{deco(mark, st), deco(approvalWho(s, ap)+" · ", stGrey)}
		l = append(l, oneRow(i, e, st)...)
		if target != "" {
			l = append(l, deco("  "+target, stText))
		}
		return l
	}
	switch ap.Status {
	case state.Allowed:
		return []line{append(head("✓ ", stDone), deco("  allowed by you "+clockSec(ap.AnsweredAt), stDone))}
	case state.Denied:
		if ap.Problem != "" {
			return []line{
				append(head("✗ ", stFailed), deco("  denied automatically: request not readable", stFailed)),
				decoLine("  "+cut(ap.Problem, max(w-2, 1)), stGrey),
			}
		}
		return []line{append(head("✗ ", stFailed), deco("  denied by you "+clockSec(ap.AnsweredAt), stFailed))}
	case state.Expired:
		return []line{append(head("· ", stGrey), deco("  expired, not answered", stGrey))}
	}
	label := "! APPROVAL "
	if ap.Question {
		label = "? QUESTION "
	}
	out := []line{append(head(label, stAttention.Bold(true)), deco("  "+clock(ap.RequestedAt), stGrey))}
	if ap.Summary != "" && ap.Summary != target {
		out = append(out, indentedDeco(ap.Summary, w, stText)...)
	}
	for j, l := range body {
		if j == cardLines {
			out = append(out, decoLine(fmt.Sprintf("  │ … %s more lines", grouped(len(body)-cardLines)), stGrey))
			break
		}
		out = append(out, decoLine("  │ "+cut(l, max(w-4, 1)), stGrey))
	}
	focused := ui.Card == ap.ID
	switch {
	case ap.Status == state.Answering:
		out = append(out, decoLine("  sending answer…", stProgress))
	case focused && ap.Question:
		out = append(out, decoLine("  ▸ d deny, then answer in your next message · c cancel the turn · Esc later", stAttention))
	case focused:
		out = append(out, decoLine("  ▸ a allow once · d deny · c cancel the turn · Esc later · ↑ ↓ other approvals", stAttention))
	default:
		out = append(out, decoLine("  F2 to answer", stGrey))
	}
	return out
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
// and the lines a card shows from the request's input; an input it cannot
// read shows as it came.
func approvalDetail(ap *state.Approval) (target string, body []string) {
	var in map[string]any
	if len(ap.Input) == 0 || json.Unmarshal(ap.Input, &in) != nil {
		if len(ap.Input) > 0 && string(ap.Input) != "null" {
			body = strings.Split(string(ap.Input), "\n")
		}
		return "", body
	}
	str := func(k string) string { v, _ := in[k].(string); return v }
	for _, k := range []string{"file_path", "notebook_path", "path", "url", "pattern"} {
		if v := str(k); v != "" {
			target = hideUser(v)
			break
		}
	}
	switch {
	case str("command") != "":
		cmd := strings.Split(str("command"), "\n")
		if target == "" && len(cmd) == 1 {
			return cmd[0], nil
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
	return target, body
}

// indentedDeco is indented, wrapped decoration.
func indentedDeco(text string, w int, st lipgloss.Style) []line {
	var out []line
	for _, l := range wrap(text, w-2) {
		out = append(out, decoLine("  "+l, st))
	}
	return out
}
