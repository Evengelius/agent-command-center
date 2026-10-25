package render

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

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
// are skipped by the cursor. A folded stage's block is several rows of one
// node: its head, then its continuation rows (plan 24, design D5), which
// the cursor's background covers with it.
type treeRow struct {
	l    line
	node Node
	cont bool // a continuation row of its node's block
	// What the row draws when its line is made only once it shows (plan
	// 49, design D5): by kind, from tc, a, st or items after prefix, its
	// connectors; dot is the moving dot's column in them, plus one, 0 for
	// none. A row of kind rowMade has its line in l.
	kind   rowKind
	prefix string
	tc     *state.ToolCall
	a      *state.Agent
	st     *state.Stage
	items  []item
	dot    int
}

// rowKind is what a tree row draws: its line made already, a stage's head,
// its origin or the names of its items, a call or an agent.
type rowKind uint8

const (
	rowMade rowKind = iota
	rowStage
	rowOrigin
	rowNames
	rowTool
	rowAgent
)

// line is the row's line at width w: zone A makes only the lines of the
// rows it shows, and Tree none (design D5).
func (r treeRow) line(s *state.State, ui UI, w int) line {
	l := r.l
	switch r.kind {
	case rowStage:
		l = stageHead(s, r.st, ui, w)
	case rowOrigin:
		l = plain("  "+originText(r.st), stGrey).fit(w)
	case rowNames:
		l = plain("  "+itemNames(r.items), stGrey).fit(w)
	case rowTool:
		l = toolRow(s, ui, r.tc, r.prefix, w)
	case rowAgent:
		l = agentLine(r.a, r.prefix, w)
	}
	if r.dot > 0 {
		l = dotted(l, r.dot-1)
	}
	return l
}

// lead is the row's first span as its line draws it at width w, and
// whether that span is decoration: what movingDot reads, without making
// the line. A call's or an agent's row fits its connectors, the symbol and
// the rest in what its origin label leaves of w, cutting from the end:
// the connectors stay whole while they leave a cell, and the "…" after.
func (r treeRow) lead(w int) (string, bool) {
	var origin string
	switch r.kind {
	case rowTool:
		origin = originLabel(r.tc.Origin)
	case rowAgent:
		origin = originLabel(r.a.StatusOrigin)
	case rowMade:
		if len(r.l) > 0 && r.l[0].ref.kind == refDeco {
			return r.l[0].text, true
		}
		return "", false
	default:
		return "", false
	}
	room := w
	if origin != "" {
		rw := textWidth(" " + origin)
		if rw >= w {
			return " " + origin, false // the label alone, cut
		}
		room = w - rw
	}
	switch {
	case room-1 <= 0:
		return "…", true
	case textWidth(r.prefix) <= room-1:
		return r.prefix, true
	}
	return cut(r.prefix, room-1), true
}

// dotted is l with the moving dot at column col of its first span.
func dotted(l line, col int) line {
	p := []rune(l[0].text)
	out := make(line, 0, len(l)+2)
	if col > 0 {
		out = append(out, deco(string(p[:col]), stFrame))
	}
	out = append(out, deco("•", stProgress))
	if col+1 < len(p) {
		out = append(out, deco(string(p[col+1:]), stFrame))
	}
	return append(out, l[1:]...)
}

// originText is what a stage's origin row says: the prompt's first line,
// or that the CLI started it; "" for no row.
func originText(st *state.Stage) string {
	if st.ByCLI {
		return "started by the CLI"
	}
	return firstLine(st.Prompt)
}

// named says itemNames names one item at least, without joining them.
func named(items []item) bool {
	for _, it := range items {
		switch {
		case it.tool != nil && it.tool.Name != "":
			return true
		case it.agent != nil && strings.TrimSpace(kindWord(it.agent.Kind)+" "+it.agent.Label) != "":
			return true
		}
	}
	return false
}

// Tree is zone A's nodes in the order they show, folded nodes' children
// left out: what ↑ and ↓ move through.
func Tree(s *state.State, ui UI) []Node {
	var out []Node
	for _, r := range treeRows(s, ui, zoneAMin) { // the nodes, whatever the width; no line made
		if r.node.ID != "" && !r.cont {
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
	idx := stageIndex(s, ui)
	if loose := idx[0]; len(loose) > 0 {
		out = itemRows(s, ui, out, loose, "", false, w)
	}
	n := len(s.Stages)
	if n == 0 {
		return out
	}
	cur := &s.Stages[n-1]
	if len(cur.Plan) == 0 {
		for i := range s.Stages {
			if i > 0 {
				out = append(out, treeRow{}) // a blank row between the turns' blocks (design D5)
			}
			out = stageRows(s, ui, out, &s.Stages[i], w, idx)
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
		row := treeRow{l: line{sym, sp(fmt.Sprintf(" %d %s", i+1, p.Step), stText)}, node: Node{ID: fmt.Sprintf("p:%d", i)}}
		out = append(out, row)
		if p.Status == "inProgress" && !attached {
			attached = true
			out = itemRows(s, ui, out, idx[cur.N], "  ", more(s, cur), w)
		}
	}
	if !attached {
		out = stageRows(s, ui, out, cur, w, idx)
	}
	for i := range s.Stages[:n-1] {
		out = append(out, treeRow{})
		out = stageRows(s, ui, out, &s.Stages[i], w, idx)
	}
	return out
}

// more says the stage is still running: the tree never pretends to know the
// steps to come.
func more(s *state.State, st *state.Stage) bool {
	return st.Open && s.Turn.Phase != state.Idle
}

// stageRows is a stage's block (plan 24, design D5, the owner's image): its
// head row, "✓ TURN 1" with its elapsed time at the right end and its word
// when it is not completed; then, unfolded, its prompt's first line and its
// items with their connectors; folded, the prompt's line and one row naming
// its items. A closed stage starts folded, an open one unfolded; the
// owner's → and ← decide after that. idx holds every stage's items.
func stageRows(s *state.State, ui UI, out []treeRow, st *state.Stage, w int, idx map[int][]item) []treeRow {
	id := fmt.Sprintf("s:%d", st.N)
	f := folded(ui, id, !st.Open)
	node := Node{ID: id, Foldable: true, Folded: f}
	out = append(out, treeRow{kind: rowStage, st: st, node: node})
	origin := originText(st) != ""
	if !f {
		if origin {
			out = append(out, treeRow{kind: rowOrigin, st: st})
		}
		return itemRows(s, ui, out, idx[st.N], "  ", more(s, st), w)
	}
	if origin {
		out = append(out, treeRow{kind: rowOrigin, st: st, node: node, cont: true})
	}
	if items := idx[st.N]; named(items) {
		out = append(out, treeRow{kind: rowNames, items: items, node: node, cont: true})
	}
	return out
}

// stageHead is a stage's head row: its symbol, "TURN n", its word unless
// it completed, and its elapsed time at the row's right end.
func stageHead(s *state.State, st *state.Stage, ui UI, w int) line {
	sym, word := stageStatus(s, st)
	l := line{sym, sp(fmt.Sprintf(" TURN %d", st.N), stBold)}
	if word != "completed" && word != "" {
		l = append(l, sp(" · "+word, stInferred))
	}
	return spread(l, plain(elapsed(st.StartedAt, st.EndedAt, ui.Now), stGrey), w)
}

// itemNames names a stage's items in order, each name once, joined by
// " · ": a tool by its name, an agent as "agent <label>" (design D5).
func itemNames(items []item) string {
	var names []string
	seen := map[string]bool{}
	for _, it := range items {
		name := ""
		switch {
		case it.tool != nil:
			name = it.tool.Name
		case it.agent != nil:
			name = strings.TrimSpace(kindWord(it.agent.Kind) + " " + it.agent.Label)
		}
		if name != "" && !seen[name] {
			seen[name] = true
			names = append(names, name)
		}
	}
	return strings.Join(names, " · ")
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
func stageItems(s *state.State, ui UI, n int) []item { return stageIndex(s, ui)[n] }

// indexItems is every stage's items, stageItems' for each stage, in one
// pass over the tools and the agents: since plan 24 a folded stage names
// its items too, so a frame needs them all, and a scan of the tools for
// each stage grew with the session (the timing step's long session).
func indexItems(s *state.State) map[int][]item {
	idx := map[int][]item{}
	for _, tc := range s.Tools {
		if tc.Agent == "" {
			idx[tc.Stage] = append(idx[tc.Stage], item{seq: tc.Seq, id: tc.ID, tool: tc})
		}
	}
	for _, k := range s.Order {
		a := s.Agents[k]
		if a == nil {
			continue
		}
		if a.Parent == "" || s.Agents[a.Parent] == nil {
			idx[a.Stage] = append(idx[a.Stage], item{seq: a.Seq, id: k, agent: a})
		}
	}
	for _, out := range idx {
		sort.Slice(out, func(i, j int) bool {
			if out[i].seq != out[j].seq {
				return out[i].seq < out[j].seq
			}
			return out[i].id < out[j].id
		})
	}
	return idx
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
			out = append(out, treeRow{kind: rowTool, tc: it.tool, prefix: indent + branch, node: Node{ID: "t:" + it.tool.ID, Agent: it.tool.Agent}})
			continue
		}
		out = agentRows(s, ui, out, it.agent, indent+branch, indent+next, w)
	}
	if more {
		out = append(out, treeRow{l: line{deco(indent+"└─ ", stFrame), sp("○ (more as events arrive)", stGrey)}})
		if turnRuns(s) {
			movingDot(out[from:], len([]rune(indent)), ui.Spinner, w)
		}
	}
	return out
}

// movingDot puts a dot on the running stage's connector, at column col of
// the rows' prefixes as drawn at width w, one row further each frame: work
// flowing down the tree, as in the reference video (reference-design.md;
// plan 4, step 8). The row draws it when its line is made (plan 49, design
// D5).
func movingDot(rows []treeRow, col, frame, w int) {
	var trunk []int
	for i, r := range rows {
		if t, decoration := r.lead(w); decoration {
			if p := []rune(t); col < len(p) && strings.ContainsRune("├│└", p[col]) {
				trunk = append(trunk, i)
			}
		}
	}
	if len(trunk) == 0 {
		return
	}
	rows[trunk[frame%len(trunk)]].dot = col + 1
}

// toolRow is "✓ Write found.txt", observed. A path too long for the row
// keeps its file name, cut at its left end (the owner's live review,
// 2026-10-20: twelve Read rows all read "C:\User…").
func toolRow(s *state.State, ui UI, tc *state.ToolCall, prefix string, w int) line {
	l := line{deco(prefix, stFrame), toolSymbol(s, tc, running(s, ui)), sp(" "+tc.Name, stText)}
	if f := ui.Memo.input(tc.Input, false); f.tool != "" {
		t := f.tool
		if f.path {
			room := w - l.width() - 2
			if o := originLabel(tc.Origin); o != "" {
				room -= textWidth(" " + o)
			}
			t = elidePath(t, max(room, 1))
		}
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
	kids := kidsOf(s, ui.Memo, a.Key)
	id := "a:" + a.Key
	f := folded(ui, id, false)
	out = append(out, treeRow{kind: rowAgent, a: a, prefix: prefix, node: Node{ID: id, Agent: a.Key, Foldable: len(under)+len(kids) > 0, Folded: f}})
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

// agentLine is an agent's row: its symbol, its kind and its label after
// prefix, its origin at the right edge.
func agentLine(a *state.Agent, prefix string, w int) line {
	l := line{deco(prefix, stFrame), agentSymbol(a.Status), sp(" ", stText)}
	if k := kindWord(a.Kind); k != "" {
		l = append(l, sp(k+"  ", stGrey))
	}
	l = append(l, sp(a.Label, stText))
	return withOrigin(l, originLabel(a.StatusOrigin), w)
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
	top := treeTop(rows, cursor, room)
	return rows[top:min(top+room, len(rows))]
}

// treeTop is treeWindow's first row.
func treeTop(rows []treeRow, cursor string, room int) int {
	if len(rows) <= room {
		return 0
	}
	top := len(rows) - room
	for i, r := range rows {
		if cursor != "" && r.node.ID == cursor && i < top {
			top = i
		}
	}
	return top
}

// paneTop is the first of rows a pane shows in room: its own top while held,
// kept in range, else rest (plan 5, item 5).
func paneTop(p Pane, rows, room, rest int) int {
	if !p.Hold {
		return rest
	}
	return min(max(p.Top, 0), max(rows-room, 0))
}

// heldHint is a held pane's hint row, w cells at most: the rows out of view,
// then the key that returns it to where it rests, left out where it does
// not fit (plan 20: zone A is narrower, 26 columns at 120).
func heldHint(above, below int, back string, w int) line {
	var parts []string
	if above > 0 {
		parts = append(parts, fmt.Sprintf("↑ %d", above))
	}
	if below > 0 {
		parts = append(parts, fmt.Sprintf("↓ %d", below))
	}
	text := strings.Join(parts, " ")
	if full := text + " · " + back; textWidth(full) <= w {
		text = full
	}
	return decoLine(text, stProgress)
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
