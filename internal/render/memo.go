package render

import (
	"encoding/json"
	"slices"
	"sort"

	"github.com/Evengelius/agent-command-center/internal/state"
)

// Memo keeps what the frames derive from the state, so a frame does not do
// it again (plan 49, designs D4 and D5; audit finding 7): what a tool's input
// shows, read once; each turn's items, each agent's calls and each agent's
// children, added to as calls come; the conversation's thinking entries,
// found as entries come. Like WrapCache it is a memo, never a source of what
// is drawn: a nil Memo derives everything again each frame, with the same
// frame. It holds what it took from one state, and starts again from
// another. Render runs on the program's event loop, and so does a Memo.
type Memo struct {
	cur, old map[inputKey]inputFacts
	misses   int // the inputs read, for the tests (criterion 49.4)
	items    itemIndex
	thinking thinkIndex
	file     fileMemo
}

// fileMemo is the Markdown kinds of the file the viewer previews, by its
// lines' identity: their first element's address and their number, a file
// read again being new lines (plan 71, design D5); idle is how many frames
// ended since one used them, which are dropped at the second, the viewer
// closed or on another file (plan 71, the cache).
type fileMemo struct {
	first *string
	n     int
	md    []mdLine
	idle  int
}

// fileMd is lines' Markdown kinds, classified once for as long as the
// viewer shows the same file.
func (m *Memo) fileMd(lines []string) []mdLine {
	if m == nil || len(lines) == 0 {
		return mdLines(lines)
	}
	if !m.holdsFile(lines) {
		m.file = fileMemo{first: &lines[0], n: len(lines), md: mdLines(lines)}
	}
	m.file.idle = 0
	return m.file.md
}

// keepFile keeps the kinds of lines, the file the viewer shows as its
// source, for its preview again.
func (m *Memo) keepFile(lines []string) {
	if m != nil && m.holdsFile(lines) {
		m.file.idle = 0
	}
}

// holdsFile reports the memo holding the kinds of lines.
func (m *Memo) holdsFile(lines []string) bool {
	return len(lines) > 0 && m.file.first == &lines[0] && m.file.n == len(lines)
}

// NewMemo is an empty memo.
func NewMemo() *Memo { return &Memo{cur: map[inputKey]inputFacts{}} }

// inputKey is an input's identity, never its contents: its first byte's
// address and its length. An event's input is a slice of its own that
// nothing writes into, so an input changed is a new key, and its row shows
// the change at once (design D4).
type inputKey struct {
	p        *byte
	n        int
	question bool
}

// inputFacts is what the rows show of an input, as approvalDetail,
// toolTarget and toolDetail read it.
type inputFacts struct {
	target     string // approvalDetail's
	path       bool
	body       []string
	tool       string // toolTarget's
	detail     string // toolDetail's
	detailPath bool
}

// readInput reads in as the three readers do, question saying it is a
// question's request.
func readInput(in json.RawMessage, question bool) inputFacts {
	var f inputFacts
	f.target, f.path, f.body = approvalDetail(&state.Approval{Input: in, Question: question})
	f.tool = toolTarget(in)
	f.detail, f.detailPath = toolDetail(&state.ToolCall{Input: in})
	return f
}

// input is in's facts, read once for as long as the frames use them: what
// a frame did not use is dropped after the next one, so the memo holds the
// inputs of two frames at most.
func (m *Memo) input(in json.RawMessage, question bool) inputFacts {
	if m == nil || len(in) == 0 {
		return readInput(in, question)
	}
	k := inputKey{&in[0], len(in), question}
	if f, ok := m.cur[k]; ok {
		return f
	}
	f, ok := m.old[k]
	if !ok {
		f = readInput(in, question)
		m.misses++
	}
	m.cur[k] = f
	return f
}

// sweep ends a frame: the inputs it did not use are dropped at the next one,
// and so are the file's kinds.
func (m *Memo) sweep() {
	if m != nil {
		m.old, m.cur = m.cur, make(map[inputKey]inputFacts, len(m.cur))
		if m.file.md != nil {
			if m.file.idle++; m.file.idle >= 2 {
				m.file = fileMemo{}
			}
		}
	}
}

// itemIndex is, for the state it was taken from, each turn's items
// (indexItems'), each agent's calls in the order they started (agentTools'),
// and each agent's children (children's). It is built whole once, and again
// when an agent known before changed its parent (State.ParentRev); the
// agents and the calls added since are then put in their places (design
// D5). A call never changes its turn, agent or sequence, nor an agent its
// turn or sequence.
type itemIndex struct {
	s       *state.State
	tools   int // the calls of ToolOrder taken in
	agents  int // the agents of Order taken in
	parents int // ParentRev when built
	stages  map[int][]item
	calls   map[string][]*state.ToolCall
	kids    map[string][]string
	waiting map[string][]string // the top agents whose parent is not known yet, by that parent
	visits  int                 // the calls visited, for the tests (criterion 49.6)
}

// before orders items as indexItems does: by sequence, then by ID.
func before(aSeq uint64, aID string, bSeq uint64, bID string) bool {
	if aSeq != bSeq {
		return aSeq < bSeq
	}
	return aID < bID
}

// index is the state's index, taken in up to its last agent and call.
func (m *Memo) index(s *state.State) *itemIndex {
	x := &m.items
	if x.s != s || x.parents != s.ParentRev || x.agents > len(s.Order) || x.tools > len(s.ToolOrder) {
		*x = itemIndex{s: s, parents: s.ParentRev, visits: x.visits, stages: map[int][]item{},
			calls: map[string][]*state.ToolCall{}, kids: map[string][]string{}, waiting: map[string][]string{}}
	}
	for _, k := range s.Order[x.agents:] {
		x.addAgent(s, k)
	}
	x.agents = len(s.Order)
	for _, id := range s.ToolOrder[x.tools:] {
		if tc := s.Tools[id]; tc != nil {
			x.add(tc)
		}
	}
	x.tools = len(s.ToolOrder)
	return x
}

// addAgent takes in an agent added since: its parent's child, a top item
// of its turn while its parent is not known, and the parent of the top
// agents that were waiting for it, which are top items no more.
func (x *itemIndex) addAgent(s *state.State, key string) {
	a := s.Agents[key]
	if a == nil {
		return
	}
	x.kids[a.Parent] = append(x.kids[a.Parent], key)
	for _, c := range x.waiting[key] {
		if ca := s.Agents[c]; ca != nil {
			list := x.stages[ca.Stage]
			if i := slices.IndexFunc(list, func(it item) bool { return it.agent == ca }); i >= 0 {
				x.stages[ca.Stage] = slices.Delete(list, i, i+1)
			}
		}
	}
	delete(x.waiting, key)
	if a.Parent == "" || s.Agents[a.Parent] == nil {
		x.insert(a.Stage, item{seq: a.Seq, id: key, agent: a})
		if a.Parent != "" {
			x.waiting[a.Parent] = append(x.waiting[a.Parent], key)
		}
	}
}

// insert puts it in its place among turn n's items: they come in their
// order, so almost always at the end.
func (x *itemIndex) insert(n int, it item) {
	list := x.stages[n]
	i := sort.Search(len(list), func(i int) bool { return before(it.seq, it.id, list[i].seq, list[i].id) })
	x.stages[n] = slices.Insert(list, i, it)
}

// add puts a call added since in its place: in its turn's items when it is
// the orchestrator's, in its agent's calls otherwise.
func (x *itemIndex) add(tc *state.ToolCall) {
	x.visits++
	if tc.Agent == "" {
		x.insert(tc.Stage, item{seq: tc.Seq, id: tc.ID, tool: tc})
		return
	}
	list := x.calls[tc.Agent]
	i := sort.Search(len(list), func(i int) bool { return before(tc.Seq, tc.ID, list[i].Seq, list[i].ID) })
	x.calls[tc.Agent] = slices.Insert(list, i, tc)
}

// stageIndex is every turn's items, from the memo when there is one.
func stageIndex(s *state.State, ui UI) map[int][]item {
	if ui.Memo == nil {
		return indexItems(s)
	}
	return ui.Memo.index(s).stages
}

// callsOf is the agent's calls in the order they started; key "" is the
// orchestrator's, which the index keeps with the turns' items.
func callsOf(s *state.State, m *Memo, key string) []*state.ToolCall {
	if m == nil || key == "" {
		return agentTools(s, key)
	}
	return m.index(s).calls[key]
}

// kidsOf is the agents whose parent is key, in order of appearance.
func kidsOf(s *state.State, m *Memo, key string) []string {
	if m == nil {
		return children(s, key)
	}
	return m.index(s).kids[key]
}

// thinkIndex is where the conversation's thinking entries are, for the
// state it was taken from: the conversation only grows, and an entry keeps
// its role, so the entries added since are all it looks at (design D5).
type thinkIndex struct {
	s      *state.State
	seen   int
	at     []int
	visits int // the entries Animating and ActivityOn looked at, for the tests
}

// thoughts are the indexes of the conversation's thinking entries, in order.
func thoughts(s *state.State, m *Memo) []int {
	if m == nil {
		var at []int
		for i := range s.Conversation {
			if s.Conversation[i].Role == "thinking" {
				at = append(at, i)
			}
		}
		return at
	}
	x := &m.thinking
	if x.s != s || x.seen > len(s.Conversation) {
		*x = thinkIndex{s: s, visits: x.visits}
	}
	for i := x.seen; i < len(s.Conversation); i++ {
		if s.Conversation[i].Role == "thinking" {
			x.at = append(x.at, i)
		}
	}
	x.seen = len(s.Conversation)
	return x.at
}

// visited counts an entry Animating or ActivityOn looked at, for the tests
// (criterion 49.6).
func (m *Memo) visited() {
	if m != nil {
		m.thinking.visits++
	}
}
