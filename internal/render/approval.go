package render

import (
	"encoding/json"
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/Evengelius/agent-command-center/internal/event"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// cardLines is how many lines of the command, diff or content a pending card
// shows; the rest is counted (ui-design.md, Approvals).
const cardLines = 6

// nestFrom is zone B's inner width from which a pending request's preview
// has its own grey box inside the card, Template 2's (plan 20).
const nestFrom = 90

// approvalCard is an approval entry's rows of content, w cells wide (plan
// 20): a pink box while it waits, Template 2's; once answered, a tool row
// with its outcome in its colour (ui-design.md, Approvals). Only the tool
// name is the entry's content; the rest is drawn from the approval and is
// decoration.
func approvalCard(s *state.State, ui UI, i int, e state.Entry, w int, k look) []line {
	ap := s.Approvals[e.MessageID]
	if ap == nil {
		return []line{append(line{deco("! APPROVAL ", stAttention)}, oneRow(i, e, stAttention)...)}
	}
	if ap.Status.Final() {
		return answeredRows(s, ui.Memo, i, e, ap, w, k)
	}
	return pendingBox(s, ui, i, e, ap, w)
}

// answeredRows are an answered request's rows on the tool rows' ground: who
// asked and the tool, its target, how it was answered and when; a
// question's answers, or why it was refused, under it.
func answeredRows(s *state.State, m *Memo, i int, e state.Entry, ap *state.Approval, w int, k look) []line {
	f := m.input(ap.Input, ap.Question)
	target, path := f.target, f.path
	target = revealed(target) // plan 48, design D6
	var detail line
	if target != "" {
		detail = line{deco(target, stMuted)}
	}
	e.Text = revealed(e.Text)
	name := append(line{deco(revealed(approvalWho(s, ap))+" · ", stMuted)}, oneRow(i, e, stToolName)...)
	at := clockSec(ap.AnsweredAt)
	sym, outcome, st := deco("✓", stDone), "allowed by you "+at, stDone
	var more []line
	switch {
	case ap.Status == state.Allowed && ap.Question:
		outcome = "answered by you " + at
		more = answeredLines(QuestionsOf(ap), ap.Answers, w-4)
	case ap.Status == state.Allowed && ap.Answer == event.AllowSession:
		outcome = "allowed for the session by you " + at
	case ap.Status == state.Denied && ap.Problem != "":
		sym, outcome, st = deco("✗", stFailed), "denied automatically: request not readable", stFailed
		more = []line{decoLine("  "+cut(ap.Problem, max(w-6, 1)), stGrey)}
	case ap.Status == state.Denied && ap.Question && !ap.Cancelled: // QuestionDenyMessage (plan 5, item 10)
		sym, outcome, st = deco("✗", stFailed), "denied by you "+at+" · answer in your next message", stFailed
	case ap.Status == state.Denied:
		sym, outcome, st = deco("✗", stFailed), "denied by you "+at, stFailed
	case ap.Status == state.Expired:
		sym, outcome, st = deco("·", stGrey), "expired, not answered", stGrey
	}
	out := []line{toolLine(sym, name, nil, k.nameW, markTarget(i, detail), path, decoLine(outcome, st), originLabel(e.Origin), w)}
	for _, l := range more {
		out = append(out, grounded(append(spaces(2), l...), w))
	}
	return out
}

// pendingBox is a request waiting for the owner, w cells wide (plan 20,
// Template 2): a pink box with its heading, who asks, the tool, its target,
// when and the origin; the request's own description when it says more;
// the questions, or a preview of what it would do; then the pill, or its
// keys once F2 gives it the focus. Never folded, never green.
func pendingBox(s *state.State, ui UI, i int, e state.Entry, ap *state.Approval, w int) []line {
	inner := max(w-4, 1)
	in := func(l line) line {
		return append(append(line{deco("│ ", stAttention)}, l.fit(inner)...), deco(" │", stAttention))
	}
	edge := func(a, b string) line {
		return decoLine(a+strings.Repeat("─", max(w-2, 0))+b, stAttention)
	}
	raw, path, body := approvalDetail(ap)
	// The request's format characters are drawn, never applied (plan 48,
	// design D6).
	target, body := revealed(raw), revealedLines(body)
	e.Text = revealed(e.Text)
	// The owner's structure (amendment 16, section 5.2): "! APPROVAL ·
	// general-purpose · Write", its time and origin at the right, then the
	// target on its own row: a path loses its left end, a command its right
	// end, only when the box is too narrow.
	label := "! APPROVAL"
	if ap.Question {
		label = "? QUESTION"
	}
	head := line{deco(label, stAttentionBold), deco(" · "+revealed(approvalWho(s, ap))+" · ", stMuted)}
	head = append(head, oneRow(i, e, stAttentionBold)...)
	right := line{deco("  "+clock(ap.RequestedAt), stGrey), deco("  "+originLabel(e.Origin), stAttention)}
	out := []line{edge("╭", "╮"), in(spread(head, right, inner))}
	if target != "" {
		out = append(out, in(append(spaces(2), fitDetail(markTarget(i, line{deco(target, stText)}), path, inner-2)...)))
	}
	qs := QuestionsOf(ap)
	focused := ui.Card == ap.ID
	// From zone B's width where labels have their text beside them, the
	// preview has its own grey box inside the card, Template 2's; narrower,
	// its rows go straight in the card, for the card to fit B.
	nest := w >= cardContent(cardAI, nestFrom)
	pw := inner
	if nest {
		pw = inner - 4
	}
	if len(qs) > 0 {
		// The questions are drawn whole, every option (plan 14, item 1).
		for _, l := range questionLines(qs, pickOf(ui, ap, qs), focused, inner) {
			out = append(out, in(l))
		}
	} else if title, rows := approvalPreview(ap, body, pw); title != "" {
		rows = append([]line{{deco("▸ ", stMuted), deco(revealed(title), stMuted)}}, rows...)
		if nest {
			sub := func(l line) line {
				return append(append(line{deco("│ ", stFrame)}, l.fit(pw)...), deco(" │", stFrame))
			}
			out = append(out, in(decoLine("╭"+strings.Repeat("─", inner-2)+"╮", stFrame)))
			for _, r := range rows {
				out = append(out, in(sub(r)))
			}
			out = append(out, in(decoLine("╰"+strings.Repeat("─", inner-2)+"╯", stFrame)))
		} else {
			for _, r := range rows {
				out = append(out, in(r))
			}
		}
	}
	if d := ap.Summary; d != "" && d != raw && d != baseOf(raw) {
		for _, l := range wrap(revealed(d), inner) {
			out = append(out, in(plain(l, stText)))
		}
	}
	if d := revealed(effect(s, ap, raw, len(body) > 0)); d != "" {
		for _, l := range wrap(d, inner) {
			out = append(out, in(plain(l, stMuted)))
		}
	}
	switch {
	case ap.Status == state.Answering:
		out = append(out, in(decoLine("sending answer…", stProgress)))
	case focused && ap.Question:
		out = append(out, in(decoLine("▸ 1-4 choose · ← → question · a send the answers · d deny, then answer in your next message · c cancel the turn · Enter full view · Esc later", stAttention)))
	case focused:
		out = append(out, in(decoLine("▸ a allow once · "+forSession(ap)+"d deny · c cancel the turn · Enter full view · Esc later · ↑ ↓ other approvals", stAttention)))
	default:
		out = append(out, in(pill("F2 to answer")))
	}
	return append(out, edge("╰", "╯"))
}

// pill is a pink button's label, its ends rounded by half blocks; on a
// 16-colour terminal, the label in brackets (plan 20, Template 2's "F2 to
// answer").
func pill(text string) line {
	if sixteen {
		return decoLine("[ "+text+" ]", stAttentionBold)
	}
	return line{deco("▐", stPillEnd), deco(" "+text+" ", stPill), deco("▌", stPillEnd)}
}

// approvalPreview is what a request would do, w cells wide: its title and
// at most cardLines rows on an inset ground, then how many more. A file's
// content is numbered (Template 2's "Preview of found.txt (2 lines)"), a
// change shows its lines removed and added, a command its lines.
func approvalPreview(ap *state.Approval, body []string, w int) (string, []line) {
	if len(body) == 0 {
		return "", nil
	}
	var in map[string]any
	_ = json.Unmarshal(ap.Input, &in)
	str := func(k string) string { v, _ := in[k].(string); return v }
	file := ""
	for _, k := range []string{"file_path", "notebook_path", "path"} {
		if v := str(k); v != "" {
			file = baseOf(v)
			break
		}
	}
	n := fmt.Sprintf("%s %s", grouped(len(body)), map[bool]string{true: "line", false: "lines"}[len(body) == 1])
	title, numbered := "Request ("+n+")", false
	switch {
	case str("content") != "":
		title, numbered = "Preview of "+file+" ("+n+")", true
	case str("new_string") != "" || str("old_string") != "":
		title = "Change to " + file + " (" + n + ")"
	case str("command") != "":
		title = "Command (" + n + ")"
	}
	gutter := len(fmt.Sprint(min(len(body), cardLines)))
	var rows []line
	for j, l := range body {
		if j == cardLines {
			rows = append(rows, grounded(decoLine(fmt.Sprintf("  … %s more lines", grouped(len(body)-cardLines)), stGrey), w))
			break
		}
		r := line{deco(" ", stNone)}
		if numbered {
			r = append(r, deco(fmt.Sprintf("%*d │ ", gutter, j+1), stGrey))
		}
		st := stText
		switch {
		case !numbered && strings.HasPrefix(l, "- "):
			st = stFailed
		case !numbered && strings.HasPrefix(l, "+ "):
			st = stDone
		}
		rows = append(rows, grounded(append(r, deco(l, st)), w))
	}
	return title, rows
}

// effect is what allowing a request does to its target, for the tools whose
// effect Claude Code documents (its tools reference,
// code.claude.com/docs/en/tools-reference.md, read on 2026-10-20): Write
// "creates a new file or overwrites an existing one with the full content
// provided", Edit "performs exact string replacement". Template 2's
// sentence, from the request's own tool and target; none for another tool
// or CLI. shown says the card shows the content above it.
func effect(s *state.State, ap *state.Approval, target string, shown bool) string {
	if s.CLI() != "claude-code" || target == "" {
		return ""
	}
	at := "`" + target + "`"
	switch ap.Tool {
	case "Write":
		if !shown {
			return "This will create or overwrite the file at " + at + ", leaving it empty."
		}
		return "This will create or overwrite the file at " + at + " with the above content."
	case "Edit":
		if !shown {
			return ""
		}
		var in struct {
			All bool `json:"replace_all"`
		}
		_ = json.Unmarshal(ap.Input, &in)
		if in.All {
			return "This will replace every occurrence of the text marked - with the text marked + in " + at + "."
		}
		return "This will replace the text marked - with the text marked + in " + at + "."
	}
	return ""
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
