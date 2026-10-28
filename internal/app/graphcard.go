package app

import (
	"strconv"

	tea "charm.land/bubbletea/v2"

	"github.com/Evengelius/agent-command-center/internal/render"
)

// The Git Graph's card (plan 62, design D13): with the pointer on a row, or
// i for the cursor's, the card beside it; a commit's whole message and its
// files read in commands the first time, kept for the cards after.

// graphCardKept bounds the messages the cards keep, and the files read for
// them: past it, those of the commits not unfolded go.
const graphCardKept = 1024

// graphMessageMsg is a commit's whole message, read for its card.
type graphMessageMsg struct {
	hash [20]byte
	msg  string
	err  error
}

// graphHoverAt is the Git Graph's row under the frame's cell (x, y), plus
// one; 0 off its rows.
func (m *Model) graphHoverAt(x, y int) int {
	if m.screen == nil || !m.ui.Graph.Open {
		return 0
	}
	if p, item := m.screen.ItemAt(x, y); p == render.GraphPanel && item > 0 {
		return item
	}
	return 0
}

// graphHover follows the pointer onto row item-1, 0 off the rows: the card
// beside the row under it.
func (m *Model) graphHover(item int) []tea.Cmd {
	if item == m.ui.Graph.Hover {
		return nil
	}
	m.ui.Graph.Hover = item
	return m.graphCardShow()
}

// graphCardPin is i: the cursor's card shown until i again.
func (m *Model) graphCardPin() []tea.Cmd {
	m.graph.pinned = !m.graph.pinned
	return m.graphCardShow()
}

// graphCardShow puts the card on the row under the pointer, else on the
// cursor's while i pinned it, else takes it away; what it shows of a commit
// read the first time.
func (m *Model) graphCardShow() []tea.Cmd {
	g, v := &m.graph, &m.ui.Graph
	row := -1
	switch {
	case v.Hover > 0:
		row = v.Hover - 1
	case g.pinned:
		row = v.Cursor
	}
	if row < 0 || row >= len(v.Rows) || v.State != "" {
		v.Card = nil
		return nil
	}
	r := v.Rows[row]
	e := v.Commits[r.Commit]
	c := &render.GraphCard{Row: row}
	var cmds []tea.Cmd
	if r.File < 0 {
		if msg, ok := g.messages[e.Hash]; ok {
			c.Message = msg
		} else {
			c.State = "reading…"
			cmds = append(cmds, m.readCommitMessage(e)...)
		}
		switch fs := g.files[e.Hash]; {
		case fs != nil:
			c.Files = filesSummary(fs, e.Parents > 1)
		case !g.filesAsked[e.Hash]:
			g.filesAsked[e.Hash] = true
			cmds = append(cmds, m.readCommitFiles(e)...)
		}
	}
	v.Card = c
	return cmds
}

// readCommitMessage reads commit e's whole message in a command, once at a
// time.
func (m *Model) readCommitMessage(e render.GraphEntry) []tea.Cmd {
	gg, ok := m.graphGit()
	if !ok || m.graph.messageAsked[e.Hash] {
		return nil
	}
	m.graph.messageAsked[e.Hash] = true
	hash, commit, root := e.Hash, e.HashText(), m.fileRoot()
	return []tea.Cmd{func() tea.Msg {
		msg, err := gg.CommitMessage(root, commit)
		return graphMessageMsg{hash: hash, msg: msg, err: err}
	}}
}

// graphMessageRead takes a commit's message: kept for its cards, a failure
// said on the card and read again the next time.
func (m *Model) graphMessageRead(msg graphMessageMsg) {
	g := &m.graph
	delete(g.messageAsked, msg.hash)
	c := m.ui.Graph.Card
	if msg.err != nil {
		if c != nil && c.Row < len(m.ui.Graph.Rows) && m.ui.Graph.Commits[m.ui.Graph.Rows[c.Row].Commit].Hash == msg.hash {
			c.State, c.Failed = graphProblem(msg.err)
		}
		return
	}
	if len(g.messages) >= graphCardKept {
		clear(g.messages)
	}
	g.messages[msg.hash] = msg.msg
	m.graphCardShow()
}

// filesSummary is a commit's files summed up for its card: "3 files
// changed, +12 −3", a merge's "against its first parent", as its rows say
// it when it changed none.
func filesSummary(fs *render.GraphFiles, merge bool) string {
	switch {
	case fs.State != "":
		return fs.State
	case fs.Total == 0 && merge:
		return "no change against its first parent"
	case fs.Total == 0:
		return "no file changed"
	}
	added, removed := 0, 0
	for _, f := range fs.Files {
		added += f.Added
		removed += f.Removed
	}
	files := "files"
	if fs.Total == 1 {
		files = "file"
	}
	against := ""
	if merge {
		against = " against its first parent"
	}
	return grouped(fs.Total) + " " + files + " changed" + against + ", +" + strconv.Itoa(added) + " −" + strconv.Itoa(removed)
}
