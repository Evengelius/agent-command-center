package render

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Evengelius/agent-command-center/internal/state"
)

// Zone B's visual language (plan 19, ui-design.md, Zone B): who speaks,
// which message is an exchange's result, and which runs of tool calls fold.
// What the neighbours of an entry and the turns decide, for the whole
// conversation in one pass each frame; the entries themselves are drawn by
// entryRows.

// Runs of tool calls fold from foldFrom calls, keeping their last foldKeep
// in view (plan 19, the owner's third image).
const (
	foldFrom = 5
	foldKeep = 3
)

// look is how zone B draws one entry, besides the entry itself.
type look struct {
	head   bool     // the speaker's label row above it: its exchange's first message or call
	result bool     // the RESULT box around it: its exchange's last message, completed
	hidden bool     // no row of its own: a folded call, or a thinking mark that ended
	fold   int      // > 0: it draws its run's fold row, for this many calls
	end    int      // with fold: the first entry after the calls it folds
	open   bool     // its run is expanded
	run    string   // the key of the folding run of calls it is in; "" for none
	first  int      // that run's first entry, the one drawing its fold row
	card   cardKind // the block it is drawn in (plan 20)
	sep    bool     // a new message: a rule above it, a blank row on each side (amendment 17)
	gap    bool     // a blank row above it: the result's box, or a part of its message
	inner  bool     // with gap: the row is the block's, between two of its parts
	top    bool     // a blank row above it: the conversation's first entry drawn
	tail   bool     // a blank row after it: the conversation's last entry drawn
	nameW  int      // its group of tool rows' name column, for their targets to align
}

// shape is what of a look changes the entry's row count, for the cache.
func (k look) shape() uint16 {
	b := uint16(k.card) << 8
	for i, on := range []bool{k.head, k.result, k.hidden, k.fold > 0, k.gap, k.sep, k.top, k.tail} {
		if on {
			b |= 1 << i
		}
	}
	return b
}

// looks is each conversation entry's look.
func looks(s *state.State, ui UI) []look {
	conv := s.Conversation
	out := make([]look, len(conv))
	// A thinking block that ended is not drawn: only one streaming without
	// text yet shows, as "∴ thinking" (plan 20, the owner's section 11).
	for i, e := range conv {
		out[i].hidden = thinkingMark(e) && !e.Streaming
	}
	// Runs of tool calls: consecutive calls, with the thinking marks between
	// two of them, which are not drawn.
	for i := 0; i < len(conv); {
		if conv[i].Role != "tool" {
			i++
			continue
		}
		calls, last, j := []int{i}, i, i+1
		for ; j < len(conv); j++ {
			if conv[j].Role == "tool" {
				calls, last = append(calls, j), j
			} else if !thinkingMark(conv[j]) {
				break
			}
		}
		for k := i; k < last; k++ {
			out[k].hidden = thinkingMark(conv[k])
		}
		if len(calls) >= foldFrom {
			key := runKey(conv[i])
			open, cut := ui.Unfolded[key], calls[len(calls)-foldKeep]
			for k := i; k <= last; k++ {
				out[k].run, out[k].first, out[k].open = key, i, open
				if !open && k < cut {
					out[k].hidden = true
				}
			}
			out[i].fold, out[i].end = len(calls)-foldKeep, cut
		}
		i = last + 1
	}
	// Exchanges: a prompt and all that follows it up to the next prompt.
	from := 0
	undone := undoneByStage(s)
	for i := 1; i <= len(conv); i++ {
		if i == len(conv) || conv[i].Role == "user" {
			exchange(s, out, from, i, undone)
			from = i
		}
	}
	cards(s, out)
	return out
}

// exchange marks the speaker's label, RESULT and the cards in entries [a,
// b), one exchange: the prompt's card, the CLI's card from its first
// message or call, the result's box. RESULT goes to its last message once
// the exchange is over, its last turn completed and every agent started in
// it completed, when a call, an approval, a review or another message came
// before it: a lone answer keeps the speaker's label. An agent still
// running, failed or stopped leaves it narration (plan 20, the owner's
// section 6.1). undone is undoneByStage's, counted once a frame.
func exchange(s *state.State, out []look, a, b int, undone []int) {
	conv := s.Conversation
	if conv[a].Role == "user" {
		out[a].card = cardYou
	}
	head, msg := -1, -1
	for k := a; k < b; k++ {
		e := conv[k]
		if (e.Role == "assistant" || e.Role == "tool") && head < 0 && (!out[k].hidden || out[k].fold > 0) {
			head = k
		}
		if e.Role == "assistant" && e.Text != "" {
			msg = k
		}
	}
	if head < 0 {
		return
	}
	last, over := lastTurn(s, a, b, msg)
	result := msg >= 0 && !conv[msg].Streaming && over && agentsDone(undone, conv[a].Stage, last)
	if result {
		result = false
		for k := a; k < msg; k++ {
			switch e := conv[k]; e.Role {
			case "tool", "approval", "review":
				result = true
			case "assistant":
				result = result || e.Text != ""
			}
		}
	}
	out[head].head = !result || head != msg
	for k := head; k < b; k++ {
		switch {
		case result && k == msg:
			out[k].card = cardResult
		case result && k > msg: // after the result's box: no card
		default:
			out[k].card = cardAI
		}
	}
	if result {
		out[msg].result = true
	}
}

// lastTurn is exchange [a, b)'s last turn, the one before the next prompt's
// or the last one, and whether it is over and completed.
func lastTurn(s *state.State, a, b, msg int) (int, bool) {
	conv := s.Conversation
	n := len(s.Stages)
	if b < len(conv) {
		n = conv[b].Stage - 1
		if conv[b].Stage == 0 && msg >= 0 {
			n = conv[msg].Stage
		}
	} else if s.Turn.Phase != state.Idle {
		return n, false
	}
	return n, n >= 1 && n <= len(s.Stages) && !s.Stages[n-1].Open && s.Stages[n-1].Outcome == "completed"
}

// undoneByStage counts, for each stage i from 0 to the last, the agents not
// completed whose stage is at most i. One pass over the agents a frame:
// agentsDone walked every agent for every exchange, so a frame's cost grew
// with exchanges times agents, past the 16 ms target in the long session
// (plan 21, evaluation: TestLongSession p95 16.69 ms, then 4.19 ms).
func undoneByStage(s *state.State) []int {
	n := len(s.Stages)
	c := make([]int, n+1)
	for _, k := range s.Order {
		if a := s.Agents[k]; a != nil && a.Status != state.AgentCompleted {
			c[min(max(a.Stage, 0), n)]++
		}
	}
	for i := 1; i <= n; i++ {
		c[i] += c[i-1]
	}
	return c
}

// agentsDone says every agent started in turns from to last completed: none
// of them is counted in undone between the two.
func agentsDone(undone []int, from, last int) bool {
	last = min(last, len(undone)-1)
	if last < 0 || last < from {
		return true
	}
	below := 0
	if from > 0 {
		below = undone[from-1]
	}
	return undone[last]-below == 0
}

// cards marks where a card starts, for the blank row above it, and gives
// each group of tool rows its name column (plan 20).
func cards(s *state.State, out []look) {
	prev := -1 // the last entry drawn
	group := []int{}
	flush := func() {
		w := 0
		for _, k := range group {
			w = max(w, nameWidth(s, s.Conversation[k]))
		}
		for _, k := range group {
			out[k].nameW = min(w, 32)
		}
		group = group[:0]
	}
	for i, e := range s.Conversation {
		k := &out[i]
		if k.hidden && k.fold == 0 {
			continue
		}
		switch p := prev; {
		case p < 0:
			k.top = true
		case k.card == cardYou || k.card == cardAI && k.head:
			// A new message, the owner's or the CLI's part of an exchange
			// under its header: the rule between two messages (amendment 17).
			k.sep = true
		case k.card != out[p].card || k.card == cardResult:
			k.gap = true
		default:
			// Inside the CLI's part, narration, a group of tool rows and a
			// request waiting in its box are a row apart (plan 20, the owner's
			// section 7.3): the row is the block's, its bar.
			if a, b := part(s, s.Conversation[p]), part(s, e); k.card == cardAI && (a != b || b == partBox) {
				k.gap, k.inner = true, true
			}
		}
		prev = i
		switch {
		case k.fold > 0 && k.hidden: // a folded run's row: no name column, the group goes on
		case !asToolRow(s, e):
			flush()
		default:
			// A group is one card's rows of one kind: calls align with calls,
			// requests ("general-purpose · Bash") with requests.
			if len(group) > 0 && (out[group[0]].card != k.card || s.Conversation[group[0]].Role != e.Role) {
				flush()
			}
			group = append(group, i)
		}
	}
	flush()
	if prev >= 0 {
		out[prev].tail = true
	}
}

// The parts of the CLI's card, for the rows between them.
const (
	partText  = iota // narration, notes, errors
	partTools        // tool rows, a fold's row
	partBox          // a request waiting for the owner, in its box
)

// part is which part of a card entry e draws.
func part(s *state.State, e state.Entry) int {
	switch {
	case asToolRow(s, e):
		return partTools
	case e.Role == "approval":
		return partBox
	}
	return partText
}

// asToolRow says entry e is drawn as a tool row: a call, or an approval once
// answered.
func asToolRow(s *state.State, e state.Entry) bool {
	switch e.Role {
	case "tool":
		return true
	case "approval":
		ap := s.Approvals[e.MessageID]
		return ap != nil && ap.Status.Final()
	}
	return false
}

// nameWidth is the width of a tool row's name part: the call's name, or who
// asked and the tool for an approval.
func nameWidth(s *state.State, e state.Entry) int {
	name, _, _ := strings.Cut(e.Text, "\n")
	if e.Role == "approval" {
		if ap := s.Approvals[e.MessageID]; ap != nil {
			return textWidth(approvalWho(s, ap) + " · " + name)
		}
	}
	return textWidth(name)
}

// thinkingMark is an assistant entry with nothing but its thinking mark.
func thinkingMark(e state.Entry) bool {
	return e.Role == "assistant" && e.Text == "" && e.Thinking
}

// runKey names a run of calls by its first call, stable as the run grows.
func runKey(e state.Entry) string {
	if e.MessageID != "" {
		return e.MessageID
	}
	return fmt.Sprintf("#%d", e.Seq)
}

// Targets says, for each conversation entry, whether zone B's entry cursor
// stops on it: an entry B shows in part, or a folded run's row; never a
// folded call (plan 19).
func Targets(s *state.State, ui UI) []bool {
	lk := looks(s, ui)
	out := make([]bool, len(lk))
	for i, k := range lk {
		out[i] = k.fold > 0 && k.hidden || !k.hidden && Expandable(s.Conversation[i])
	}
	return out
}

// FoldOf is the folding run of calls entry n (1-based) is in: its key, its
// first entry (1-based), whether the run is expanded, and whether entry n is
// folded away, its run's row standing for it; ok false for none.
func FoldOf(s *state.State, ui UI, n int) (key string, first int, open, folded, ok bool) {
	if n < 1 || n > len(s.Conversation) {
		return "", 0, false, false, false
	}
	k := looks(s, ui)[n-1]
	if k.run == "" {
		return "", 0, false, false, false
	}
	return k.run, k.first + 1, k.open, !k.open && k.hidden, true
}

// modelName is how zone B's speaker label names the model:
// claude-<family>-<major>-<minor>, dated or not, as "<Family> <major>.<minor>"
// (plan 19, the owner's drawing); any other id as the CLI gives it.
func modelName(id string) string {
	p := strings.Split(id, "-")
	if len(p) < 4 || len(p) > 5 || p[0] != "claude" {
		return id
	}
	fam := map[string]string{"opus": "Opus", "sonnet": "Sonnet", "haiku": "Haiku"}[p[1]]
	if fam == "" || !digits(p[2], 2) || !digits(p[3], 2) || len(p) == 5 && (len(p[4]) != 8 || !digits(p[4], 8)) {
		return id
	}
	return fam + " " + p[2] + "." + p[3]
}

// digits says s is 1 to n decimal digits.
func digits(s string, n int) bool {
	if s == "" || len(s) > n {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// toolDetail is what a tool row shows after the call's name: its target (a
// path, a URL, a pattern, a command's first line), for an agent its type
// and description; path says the text is a path, elided from its left.
func toolDetail(tc *state.ToolCall) (text string, path bool) {
	if tc == nil || len(tc.Input) == 0 {
		return "", false
	}
	var in struct {
		Type        string `json:"subagent_type"`
		Description string `json:"description"`
	}
	if json.Unmarshal(tc.Input, &in) == nil && in.Type != "" {
		text = in.Type
		if in.Description != "" {
			text += " · " + in.Description
		}
	} else if t, p, _ := approvalDetail(&state.Approval{Input: tc.Input}); t != "" {
		text, path = t, p
	} else {
		text = toolTarget(tc.Input)
	}
	text, _, _ = strings.Cut(text, "\n")
	return text, path
}
