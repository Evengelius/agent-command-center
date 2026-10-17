package render

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/Evengelius/agent-command-center/internal/state"
)

// Zone A's tree (ui-design.md, Zone A; events-and-state.md, Agent tree): one
// stage per turn, inferred, or a Codex plan, observed; under each, the
// turn's tools and agents in the order they appeared.

// Node is one row of zone A's tree the cursor can stand on.
type Node struct {
	ID       string // "s:N" a stage, "p:I" a plan step, "t:ID" a tool, "a:KEY" an agent
	Agent    string // the agent Enter inspects; "" the orchestrator's own
	Foldable bool
	Folded   bool
}

// treeRow is a drawn row of the tree; rows without a node (a prompt, "more")
// are skipped by the cursor.
type treeRow struct {
	l    line
	node Node
}

var (
	stInferred   = style(stText.Italic(true))
	stTreeCursor = style(lipgloss.NewStyle().Foreground(cText).Background(lipgloss.Color("#2E3247")))
)

// Tree is zone A's nodes in the order they show, folded nodes' children
// left out: what ↑ and ↓ move through.
func Tree(s *state.State, ui UI) []Node {
	var out []Node
	for _, r := range treeRows(s, ui, zoneAStandard) {
		if r.node.ID != "" {
			out = append(out, r.node)
		}
	}
	return out
}

// folded is the owner's fold of a node, else its default.
func folded(ui UI, id string, def bool) bool {
	if v, ok := ui.Fold[id]; ok {
		return v
	}
	return def
}

// treeRows are the tree's rows at width w.
func treeRows(s *state.State, ui UI, w int) []treeRow {
	var out []treeRow
	if loose := stageItems(s, 0); len(loose) > 0 {
		out = itemRows(s, ui, out, loose, "", false, w)
	}
	n := len(s.Stages)
	if n == 0 {
		return out
	}
	cur := &s.Stages[n-1]
	if len(cur.Plan) == 0 {
		for i := range s.Stages {
			out = stageRows(s, ui, out, &s.Stages[i], i == n-1, w)
		}
		return out
	}
	attached := false
	for i, p := range cur.Plan {
		sym := deco("○", stGrey)
		switch p.Status {
		case "completed":
			sym = deco("✓", stDone)
		case "inProgress":
			sym = deco("◉", stProgress)
		}
		row := treeRow{line{sym, sp(fmt.Sprintf(" %d %s", i+1, p.Step), stText)}, Node{ID: fmt.Sprintf("p:%d", i)}}
		out = append(out, row)
		if p.Status == "inProgress" && !attached {
			attached = true
			out = itemRows(s, ui, out, stageItems(s, cur.N), "  ", more(s, cur), w)
		}
	}
	if !attached {
		out = stageRows(s, ui, out, cur, true, w)
	}
	if n > 1 {
		out = append(out, treeRow{})
	}
	for i := range s.Stages[:n-1] {
		out = stageRows(s, ui, out, &s.Stages[i], false, w)
	}
	return out
}

// more says the stage is still running: the tree never pretends to know the
// steps to come.
func more(s *state.State, st *state.Stage) bool {
	return st.Open && s.Turn.Phase != state.Idle
}

// stageRows is a stage's line, "TURN n ✓ completed · 31s", and, unfolded,
// its prompt and items. Every stage but the last starts folded.
func stageRows(s *state.State, ui UI, out []treeRow, st *state.Stage, last bool, w int) []treeRow {
	id := fmt.Sprintf("s:%d", st.N)
	f := folded(ui, id, !last)
	sym, word := stageStatus(s, st)
	head := line{sp(fmt.Sprintf("TURN %d  ", st.N), stInferred), sym, sp(" "+word+" · "+elapsed(st.StartedAt, st.EndedAt, ui.Now), stInferred)}
	out = append(out, treeRow{head.fit(w), Node{ID: id, Foldable: true, Folded: f}})
	if f {
		return out
	}
	if st.ByCLI {
		out = append(out, treeRow{l: plain("  started by the CLI", stGrey)})
	} else if p := firstLine(st.Prompt); p != "" {
		out = append(out, treeRow{l: plain("  "+p, stGrey)})
	}
	return itemRows(s, ui, out, stageItems(s, st.N), "  ", more(s, st), w)
}

// stageStatus is the stage's symbol and word: the turn's while it runs, its
// outcome once closed.
func stageStatus(s *state.State, st *state.Stage) (span, string) {
	if st.Open {
		return turnSymbol(s), turnWord(s)
	}
	switch st.Outcome {
	case "failed":
		return deco("✗", stFailed), st.Outcome
	case "interrupted":
		return deco("○", stGrey), st.Outcome
	case "":
		return deco("✓", stDone), "ended"
	}
	return deco("✓", stDone), st.Outcome
}

// item is a tool or an agent under a stage.
type item struct {
	seq   uint64
	id    string
	tool  *state.ToolCall
	agent *state.Agent
}

// stageItems are the orchestrator's tools and the top agents of stage n, in
// the order they appeared. An agent whose parent is unknown counts as a top
// agent: it is shown, never dropped.
func stageItems(s *state.State, n int) []item {
	var out []item
	for _, tc := range s.Tools {
		if tc.Agent == "" && tc.Stage == n {
			out = append(out, item{seq: tc.Seq, id: tc.ID, tool: tc})
		}
	}
	for _, k := range s.Order {
		a := s.Agents[k]
		if a == nil || a.Stage != n {
			continue
		}
		if a.Parent == "" || s.Agents[a.Parent] == nil {
			out = append(out, item{seq: a.Seq, id: k, agent: a})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].seq != out[j].seq {
			return out[i].seq < out[j].seq
		}
		return out[i].id < out[j].id
	})
	return out
}

// itemRows draws items with ├─ └─ │ connectors, and "○ (more as events
// arrive)" last while the stage runs.
func itemRows(s *state.State, ui UI, out []treeRow, items []item, indent string, more bool, w int) []treeRow {
	from := len(out)
	for i, it := range items {
		branch, next := "├─ ", "│  "
		if i == len(items)-1 && !more {
			branch, next = "└─ ", "   "
		}
		if it.tool != nil {
			out = append(out, treeRow{toolRow(s, ui, it.tool, indent+branch, w), Node{ID: "t:" + it.tool.ID, Agent: it.tool.Agent}})
			continue
		}
		out = agentRows(s, ui, out, it.agent, indent+branch, indent+next, w)
	}
	if more {
		out = append(out, treeRow{l: line{deco(indent+"└─ ", stFrame), sp("○ (more as events arrive)", stGrey)}})
		if turnRuns(s) {
			movingDot(out[from:], len([]rune(indent)), ui.Spinner)
		}
	}
	return out
}

// movingDot puts a dot on the running stage's connector, at column col of
// the rows' prefixes, one row further each frame: work flowing down the
// tree, as in the reference video (reference-design.md; plan 4, step 8).
func movingDot(rows []treeRow, col, frame int) {
	var trunk []int
	for i, r := range rows {
		if len(r.l) > 0 && r.l[0].ref.kind == refDeco {
			if p := []rune(r.l[0].text); col < len(p) && strings.ContainsRune("├│└", p[col]) {
				trunk = append(trunk, i)
			}
		}
	}
	if len(trunk) == 0 {
		return
	}
	r := &rows[trunk[frame%len(trunk)]]
	p := []rune(r.l[0].text)
	l := make(line, 0, len(r.l)+2)
	if col > 0 {
		l = append(l, deco(string(p[:col]), stFrame))
	}
	l = append(l, deco("•", stProgress))
	if col+1 < len(p) {
		l = append(l, deco(string(p[col+1:]), stFrame))
	}
	r.l = append(l, r.l[1:]...)
}

// toolRow is "✓ Write found.txt", observed.
func toolRow(s *state.State, ui UI, tc *state.ToolCall, prefix string, w int) line {
	l := line{deco(prefix, stFrame), toolSymbol(s, tc, running(s, ui)), sp(" "+tc.Name, stText)}
	if t := toolTarget(tc.Input); t != "" {
		l = append(l, sp("  "+t, stGrey))
	}
	return withOrigin(l, originLabel(tc.Origin), w)
}

// withOrigin puts the origin label at the right edge.
func withOrigin(l line, label string, w int) line {
	if label == "" {
		return l.fit(w)
	}
	return spread(l, plain(" "+label, stGrey), w)
}

// toolSymbol is a call's status; run is the symbol of a running call.
func toolSymbol(s *state.State, tc *state.ToolCall, run span) span {
	if pending(s, tc.ID) {
		return deco("!", stAttention)
	}
	switch tc.Status {
	case state.ToolCompleted:
		return deco("✓", stDone)
	case state.ToolFailed, state.ToolDenied:
		return deco("✗", stFailed)
	case state.ToolUnfinished:
		return deco("○", stGrey)
	}
	return run
}

// running is the symbol of a call or task under way: the spinner while the
// turn runs, so the tree shows work moving (plan 4, step 8); "◉" otherwise,
// when no frame turns it.
func running(s *state.State, ui UI) span {
	if turnRuns(s) {
		return deco(spinner[ui.Spinner%len(spinner)], stProgress)
	}
	return deco("◉", stProgress)
}

// turnRuns says the turn is under way, so the spinner's frames turn (the
// application animates the same phases).
func turnRuns(s *state.State) bool {
	switch s.Turn.Phase {
	case state.Submitted, state.Running, state.Interrupting:
		return true
	}
	return false
}

// pending says an approval for the tool call waits for the owner.
func pending(s *state.State, toolID string) bool {
	for _, id := range s.Queue {
		if ap := s.Approvals[id]; ap != nil && ap.ToolID == toolID && toolID != "" &&
			(ap.Status == state.Pending || ap.Status == state.Answering) {
			return true
		}
	}
	return false
}

// agentRows is an agent's line and, unfolded, its pending approval or
// current tool and its own agents.
func agentRows(s *state.State, ui UI, out []treeRow, a *state.Agent, prefix, indent string, w int) []treeRow {
	var under []line
	for _, id := range s.Queue {
		if ap := s.Approvals[id]; ap != nil && ap.Agent == a.Key && (ap.Status == state.Pending || ap.Status == state.Answering) {
			under = append(under, line{deco("!", stAttention), sp(" approval "+ap.Tool, stText)})
		}
	}
	if len(under) == 0 && !a.Status.Final() && a.LastTool != "" {
		under = append(under, line{running(s, ui), sp(" "+a.LastTool, stGrey)})
	}
	kids := children(s, a.Key)
	id := "a:" + a.Key
	f := folded(ui, id, false)
	l := line{deco(prefix, stFrame), agentSymbol(a.Status), sp(" ", stText)}
	if k := kindWord(a.Kind); k != "" {
		l = append(l, sp(k+"  ", stGrey))
	}
	l = append(l, sp(a.Label, stText))
	out = append(out, treeRow{withOrigin(l, originLabel(a.StatusOrigin), w), Node{ID: id, Agent: a.Key, Foldable: len(under)+len(kids) > 0, Folded: f}})
	if f {
		return out
	}
	for i, u := range under {
		branch := "├─ "
		if i == len(under)-1 && len(kids) == 0 {
			branch = "└─ "
		}
		out = append(out, treeRow{l: append(line{deco(indent+branch, stFrame)}, u...).fit(w)})
	}
	for i, k := range kids {
		branch, next := "├─ ", "│  "
		if i == len(kids)-1 {
			branch, next = "└─ ", "   "
		}
		out = agentRows(s, ui, out, s.Agents[k], indent+branch, indent+next, w)
	}
	return out
}

// kindWord names an agent's kind as the tree shows it.
func kindWord(kind string) string {
	switch kind {
	case "subagent":
		return "agent"
	case "thread":
		return "child"
	}
	return kind
}

// toolTarget is what a tool call acts on, from its input: a path, a
// command's first line, a Codex change's path, an agent's description.
func toolTarget(in json.RawMessage) string {
	if len(in) == 0 || in[0] != '{' {
		return ""
	}
	if t, _, body := approvalDetail(&state.Approval{Input: in}); t != "" {
		return t
	} else if len(body) > 0 && strings.HasPrefix(body[0], "$ ") {
		return strings.TrimPrefix(body[0], "$ ") + " …"
	}
	var m struct {
		Changes []struct {
			Path string `json:"path"`
		} `json:"changes"`
		Description string `json:"description"`
	}
	if json.Unmarshal(in, &m) != nil {
		return ""
	}
	if len(m.Changes) > 0 {
		t := hideUser(m.Changes[0].Path)
		if len(m.Changes) > 1 {
			t += fmt.Sprintf(" +%d", len(m.Changes)-1)
		}
		return t
	}
	return m.Description
}

// treeWindow keeps the cursor's row visible in room rows: the tail without
// a cursor, and the cursor's row at the top once it is above the tail.
func treeWindow(rows []treeRow, cursor string, room int) []treeRow {
	if len(rows) <= room {
		return rows
	}
	top := len(rows) - room
	for i, r := range rows {
		if cursor != "" && r.node.ID == cursor && i < top {
			top = i
		}
	}
	return rows[top : top+room]
}

// cursorLine restyles a row's content for the cursor; symbols keep their
// colours.
func cursorLine(l line, w int) line {
	out := make(line, 0, len(l)+1)
	for _, x := range l {
		if x.ref.kind == refNone || x.st == stNone {
			x.st = stTreeCursor
		}
		out = append(out, x)
	}
	if n := w - out.width(); n > 0 {
		out = append(out, deco(strings.Repeat(" ", n), stTreeCursor))
	}
	return out
}
