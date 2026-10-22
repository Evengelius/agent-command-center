package render

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/Evengelius/agent-command-center/internal/state"
)

// Zone A since plan 24 (brief amendment 22, the owner's image
// references/03-zone-a-telemetry.webp; ui-design.md, Zone A): its heading
// inside the frame, the orchestrator's model with its status at the row's
// right end, a rule, the stages, then the telemetry pane at its bottom,
// which took zone D's values.

// minTree is the least rows the tree keeps when the telemetry pane shrinks
// (design D6); on a terminal too short for the pane's tight form beside
// them, the tree keeps fewTree rows and the pane the rest.
const (
	minTree = 6
	fewTree = 3
)

// zoneA draws the zone w by h cells; sc records what the tree showed, for
// scrolling, and the node under each screen row, for the mouse (sc.AY is
// the frame's top row on the screen).
func zoneA(s *state.State, ui UI, w, h int, sc *Screen) []line {
	inner := w - 4
	focused := ui.Focus == "A"
	head := []line{
		decoLine("A · ORCHESTRATOR", stTitle),
		modelRow(s, ui, inner),
		teleRule(inner),
		stagesHeading(s),
	}
	avail := max(h-2-len(head), 0)
	tele, teleNodes := telemetryPane(s, ui, inner, avail-minTree, avail-fewTree)
	room := max(avail-len(tele), 0)
	rows := treeRows(s, ui, inner)
	if len(rows) == 0 {
		rows = []treeRow{{l: plain("○ (stages appear as turns run)", stGrey)}}
	}
	top := paneTop(ui.Tree, len(rows), room, treeTop(rows, ui.ACursor, room))
	sc.Tree = PaneView{Rows: len(rows), Room: room, Top: top}
	for _, r := range rows {
		sc.Tree.IDs = append(sc.Tree.IDs, r.node.ID)
	}
	body := head
	nodes := make([]string, len(head))
	for _, r := range rows[top:min(top+room, len(rows))] {
		l := r.l
		if focused && r.node.ID != "" && r.node.ID == ui.ACursor {
			l = cursorLine(l.fit(inner), inner)
		}
		body = append(body, l)
		nodes = append(nodes, r.node.ID)
	}
	for len(body) < len(head)+room {
		body = append(body, line{})
		nodes = append(nodes, "")
	}
	for i, l := range tele {
		if focused && teleNodes[i] != "" && teleNodes[i] == ui.ACursor {
			l = cursorLine(l.fit(inner), inner)
		}
		body = append(body, l)
		nodes = append(nodes, teleNodes[i])
	}
	if sc.ARows == nil {
		sc.ARows = map[int]string{}
	}
	for i, id := range nodes {
		if id != "" {
			sc.ARows[sc.AY+1+i] = id
		}
	}
	// The rows out of view while the tree is held, in the top edge (design
	// D2), as zone C's since plan 20.
	var right line
	if ui.Tree.Hold && top < len(rows)-room {
		right = heldHint(top, len(rows)-top-room, "End follows", inner)
	}
	return box('A', nil, right, body, w, h, borderFor(focused))
}

// stagesHeading names the tree's origin: inferred stages, or a Codex plan
// observed (events-and-state.md, Agent tree).
func stagesHeading(s *state.State) line {
	if n := len(s.Stages); n > 0 && len(s.Stages[n-1].Plan) > 0 {
		return line{deco("PLAN", stBold), deco(" · observed (turn/plan/updated)", stGrey)}
	}
	return line{deco("STAGES", stBold), deco(" · inferred from activity", stGrey)}
}

// modelRow is the orchestrator's model, with its icon, and its status at
// the row's right end (design D3): the model's title, or zone B's short
// name when both do not fit, or the title cut.
func modelRow(s *state.State, ui UI, w int) line {
	status := orchestratorStatus(s, ui)
	left := modelLead(s, true)
	if left.width()+1+status.width() > w {
		left = modelLead(s, false)
	}
	return spread(left, status, w)
}

// modelLead is the icon and the model's name: "✿ Claude Haiku 5.5", or the
// short "✿ Haiku 5.5"; "model N/A (<reason>)" before the model is known.
func modelLead(s *state.State, full bool) line {
	m := model(s)
	if m == "" {
		return plain("model N/A ("+naReason(s)+")", stGrey)
	}
	name := modelName(m)
	if full {
		name = modelTitle(m)
	}
	return line{deco(modelIcon(m)+" ", modelStyle(m)), sp(name, stBold)}
}

// modelIcon is "✿" for a Claude model, "◆" for any other (design D3).
func modelIcon(id string) string {
	if strings.HasPrefix(strings.ToLower(id), "claude") {
		return "✿"
	}
	return "◆"
}

// modelTitle is the model's title: a Claude id as "Claude <Family>
// <major>.<minor>", an id starting with "gpt-" as "GPT-" and the rest, any
// other as the CLI gives it (design D3).
func modelTitle(id string) string {
	if n := modelName(id); n != id {
		return "Claude " + n
	}
	if strings.HasPrefix(strings.ToLower(id), "gpt-") {
		return "GPT-" + id[4:]
	}
	return id
}

// orchestratorStatus is the status the model row ends with (design D4),
// from the state alone: the session's phase, then the turn's, then the
// agents still working once the turn is idle, then the turn's outcome.
func orchestratorStatus(s *state.State, ui UI) line {
	spin := spinner[ui.Spinner%len(spinner)]
	switch s.Session.Phase {
	case state.Spawning, state.Handshaking, "":
		return plain("◉ STARTING", stProgress)
	case state.Closing:
		return plain("◉ CLOSING", stProgress)
	case state.Crashed:
		return plain("✗ ENDED", stFailed)
	case state.Closed:
		return plain("○ CLOSED", stGrey)
	}
	switch s.Turn.Phase {
	case state.WaitingApproval:
		return plain("! WAITING", stAttention)
	case state.Interrupting:
		return plain("◉ INTERRUPTING", stProgress)
	case state.Submitted, state.Running:
		if thinkingLive(s) {
			return plain(spin+" THINKING", stTitle)
		}
		return plain(spin+" RUNNING", stProgress)
	}
	waiting, running := 0, 0
	for _, k := range s.Order {
		switch a := s.Agents[k]; {
		case a == nil:
		case a.Status == state.AgentWaiting:
			waiting++
		case !a.Status.Final() && a.Status != state.AgentAnnounced:
			running++
		}
	}
	switch {
	case waiting > 0:
		return plain("! AGENT WAITING", stAttention)
	case running == 1:
		return plain("◉ AGENT RUNNING", stProgress)
	case running > 1:
		return plain(fmt.Sprintf("◉ %d AGENTS RUNNING", running), stProgress)
	}
	switch s.Turn.Outcome {
	case "completed":
		return plain("✓ COMPLETED", stDone)
	case "failed":
		return plain("✗ FAILED", stFailed)
	case "interrupted":
		return plain("○ INTERRUPTED", stProgress)
	}
	return plain("○ IDLE", stGrey)
}

// Nodes are zone A's nodes in cursor order: the tree's, then the
// telemetry pane's three headings (design D6): what ↑ and ↓ move through.
func Nodes(s *state.State, ui UI) []Node {
	return append(Tree(s, ui), TelemetryNodes(ui)...)
}

// IsTelemetryNode says a node id is one of the telemetry pane's.
func IsTelemetryNode(id string) bool { return strings.HasPrefix(id, "u:") }

// aFooter is the footer's line while zone A has the focus (design D11).
const aFooter = "↑↓ move · ←→ fold · Enter inspect or details · z zoom · PgUp PgDn Home End scroll · F6 zone · Esc back to the input · F1 keys"

// zoneTitle styles a zone's title in its top edge, for the zones that keep
// one there (B and C).
func zoneTitle(t string, focused bool) line {
	if focused {
		return plain(t, stTitle)
	}
	return plain(t, stBold)
}

func borderFor(focused bool) *lipgloss.Style {
	if focused {
		return stTitle
	}
	return stFrame
}
