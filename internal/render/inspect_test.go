package render

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Evengelius/agent-command-center/internal/event"
	"github.com/Evengelius/agent-command-center/internal/fixture"
	"github.com/Evengelius/agent-command-center/internal/provider/claude"
	"github.com/Evengelius/agent-command-center/internal/provider/codex"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// Plan 4, step 6, checks 2 to 5: zone A's tree, zone C's order and the
// agent inspection, from the recordings.

const wide = 10000 // no row is cut: the tests read whole rows

func probeEvents(t *testing.T, name, provider string, m fixture.Mapper) []event.Event {
	t.Helper()
	recs, err := fixture.Read(filepath.Join("..", "..", "testdata", "probes", name))
	if err != nil {
		t.Fatal(err)
	}
	return fixture.Flatten(fixture.Replay(provider, recs, m))
}

func probeState(t *testing.T, name, provider string, m fixture.Mapper) *state.State {
	t.Helper()
	s := state.New()
	for _, ev := range probeEvents(t, name, provider, m) {
		s.Apply(ev)
	}
	return s
}

func texts(ls []line) string {
	var b strings.Builder
	for _, l := range ls {
		for _, s := range l {
			b.WriteString(s.text)
		}
		b.WriteString("\n")
	}
	return b.String()
}

func agentByLabel(t *testing.T, s *state.State, label string) *state.Agent {
	t.Helper()
	for _, a := range s.Agents {
		if a.Label == label {
			return a
		}
	}
	t.Fatalf("no agent %q", label)
	return nil
}

// P7: each tab holds only its agent's events, both ways.
func TestInspectP7(t *testing.T) {
	s := probeState(t, "p7-codex-subagents.jsonl", codex.ProviderID, codex.NewMapper())
	list, create := agentByLabel(t, s, "/root/list_files"), agentByLabel(t, s, "/root/create_found")
	now := s.Agents[list.Key].EndedAt

	lt, ct := texts(inspectTools(s, nil, list, wide, now)), texts(inspectTools(s, nil, create, wide, now))
	if !strings.Contains(lt, "Get-ChildItem") || strings.Contains(lt, "Join-Path") || strings.Contains(lt, "found.txt") {
		t.Errorf("list_files Tools:\n%s", lt)
	}
	if strings.Contains(ct, "Get-ChildItem") || !strings.Contains(ct, "Join-Path") {
		t.Errorf("create_found Tools:\n%s", ct)
	}

	ld, cd := texts(inspectDiff(s, list, wide)), texts(inspectDiff(s, create, wide))
	if !strings.Contains(ld, "no file change by this agent") {
		t.Errorf("list_files Diff:\n%s", ld)
	}
	if !strings.Contains(cd, "found.txt") || !strings.Contains(cd, "observed") || strings.Contains(cd, "derived") || strings.Contains(cd, "no file change") {
		t.Errorf("create_found Diff:\n%s", cd)
	}

	for _, a := range []*state.Agent{list, create} {
		other := create
		if a == create {
			other = list
		}
		raw := inspectRaw(s, a, wide)
		n := 0
		for _, ev := range s.Log {
			if ev.Agent == a.Key {
				n++
			}
		}
		if n == 0 || len(raw) != n {
			t.Errorf("%s: %d raw rows, %d events of its own", a.Label, len(raw), n)
		}
		if r := texts(raw); !strings.Contains(r, a.Key) || strings.Contains(r, other.Key) {
			t.Errorf("%s: raw rows name %s or miss %s", a.Label, other.Key, a.Key)
		}
		m := texts(inspectMessages(s, nil, a, wide))
		if !strings.Contains(m, "started by orchestrator") || !strings.Contains(m, "completed") {
			t.Errorf("%s Messages:\n%s", a.Label, m)
		}
	}
}

// P3b: the subagent's two approvals under its Bash and Write, the Write
// derived in Diff.
func TestInspectP3b(t *testing.T) {
	s := probeState(t, "p3b-claude-subagent-approval.jsonl", claude.ProviderID, claude.NewMapper())
	a := s.Agents["a9dcbbc8d63adf34c"]
	if a == nil {
		t.Fatal("no subagent")
	}
	tools := texts(inspectTools(s, nil, a, wide, a.EndedAt))
	for _, want := range []string{"✓ Bash", "✓ Write", "approval Bash · allowed by you", "approval Write · allowed by you"} {
		if strings.Count(tools, want) != 1 {
			t.Errorf("Tools: %q %d times, want 1:\n%s", want, strings.Count(tools, want), tools)
		}
	}
	if n := strings.Count(tools, "approval "); n != 2 {
		t.Errorf("Tools: %d approvals, want 2", n)
	}
	diff := texts(inspectDiff(s, a, wide))
	if !strings.Contains(diff, "found.txt  Write") || !strings.Contains(diff, "derived") || strings.Contains(diff, "observed") || !strings.Contains(diff, "+") {
		t.Errorf("Diff:\n%s", diff)
	}
	// The orchestrator's Agent tool belongs to no agent's inspection.
	if strings.Contains(tools, "Agent") {
		t.Errorf("Tools hold the orchestrator's Agent tool:\n%s", tools)
	}
}

// TestAgentOrder: waiting, then running, then ended, most recent first in
// each group (ui-design.md, Zone C).
func TestAgentOrder(t *testing.T) {
	t0 := time.Date(2026, 10, 16, 10, 0, 0, 0, time.UTC)
	at := func(m int) time.Time { return t0.Add(time.Duration(m) * time.Minute) }
	type ag struct {
		key        string
		st         state.AgentStatus
		start, end int // minutes; end -1 none
	}
	tests := []struct {
		name   string
		agents []ag
		want   string
	}{
		{"by status", []ag{{"done", state.AgentCompleted, 0, 1}, {"run", state.AgentRunning, 2, -1}, {"wait", state.AgentWaiting, 3, -1}}, "wait,run,done"},
		{"waiting first even if older", []ag{{"run", state.AgentRunning, 9, -1}, {"wait", state.AgentWaiting, 1, -1}}, "wait,run"},
		{"running, most recent first", []ag{{"r1", state.AgentRunning, 1, -1}, {"r2", state.AgentRunning, 5, -1}, {"r3", state.AgentRunning, 3, -1}}, "r2,r3,r1"},
		{"ended by end time", []ag{{"e1", state.AgentCompleted, 0, 9}, {"e2", state.AgentFailed, 5, 6}, {"e3", state.AgentStopped, 1, 7}}, "e1,e3,e2"},
		{"announced with running", []ag{{"ann", state.AgentAnnounced, 4, -1}, {"run", state.AgentRunning, 2, -1}, {"end", state.AgentEnded, 0, 8}}, "ann,run,end"},
		{"same time, later event first", []ag{{"a", state.AgentRunning, 1, -1}, {"b", state.AgentRunning, 1, -1}}, "b,a"},
	}
	for _, tt := range tests {
		s := state.New()
		for i, x := range tt.agents {
			a := &state.Agent{Key: x.key, Label: x.key, Status: x.st, StartedAt: at(x.start), Seq: uint64(i + 1)}
			if x.end >= 0 {
				a.EndedAt = at(x.end)
			}
			s.Agents[x.key] = a
			s.Order = append(s.Order, x.key)
		}
		if got := strings.Join(AgentOrder(s), ","); got != tt.want {
			t.Errorf("%s: order %s, want %s", tt.name, got, tt.want)
		}
		active, _ := activeAgents(s)
		n := 0
		for _, x := range tt.agents {
			if x.st == state.AgentRunning || x.st == state.AgentWaiting {
				n++
			}
		}
		if active != n {
			t.Errorf("%s: %d active, want %d (running and waiting)", tt.name, active, n)
		}
	}
}

// TestTreeP3b: one stage per turn, earlier ones folded; unfolded, the Agent
// tool and the subagent under turn 1, each with its origin.
func TestTreeP3b(t *testing.T) {
	s := probeState(t, "p3b-claude-subagent-approval.jsonl", claude.ProviderID, claude.NewMapper())
	ui := UI{Width: 120, Height: 40}
	ids := func(ns []Node) string {
		var out []string
		for _, n := range ns {
			out = append(out, n.ID)
		}
		return strings.Join(out, ",")
	}
	if got := ids(Tree(s, ui)); got != "s:1,s:2" {
		t.Fatalf("nodes %s, want s:1,s:2 (turn 1 folded)", got)
	}
	ui.Fold = map[string]bool{"s:1": false}
	nodes := Tree(s, ui)
	if len(nodes) != 4 || nodes[0].ID != "s:1" || !strings.HasPrefix(nodes[1].ID, "t:") || nodes[2].ID != "a:a9dcbbc8d63adf34c" || nodes[2].Agent != "a9dcbbc8d63adf34c" || nodes[3].ID != "s:2" {
		t.Fatalf("nodes %s", ids(nodes))
	}
	if nodes[1].Agent != "" {
		t.Errorf("the Agent tool inspects %q, want the orchestrator's", nodes[1].Agent)
	}
	rows := texts(func() (ls []line) {
		for _, r := range treeRows(s, ui, 60) {
			ls = append(ls, r.line(s, ui, 60))
		}
		return
	}())
	for _, want := range []string{"✓ TURN 1", "Use the Agent tool", "✓ Agent", "✓ agent  general-purpose", "✓ TURN 2", "started by the CLI"} {
		if !strings.Contains(rows, want) {
			t.Errorf("tree lacks %q:\n%s", want, rows)
		}
	}
	if strings.Count(rows, "observed") != 2 || strings.Contains(rows, "more as events arrive") || strings.Contains(rows, "completed") {
		t.Errorf("want the tool and the agent observed, no 'more' once ended, no 'completed' word (plan 24):\n%s", rows)
	}
	a := texts(zoneA(s, ui, 36, 30, &Screen{}))
	if !strings.Contains(a, "STAGES · inferred from activity") || !strings.Contains(a, "A · ORCHESTRATOR") {
		t.Errorf("zone A heading:\n%s", a)
	}
	// Folding the agent hides nothing more: it has nothing under it once ended.
	ui.Fold["a:a9dcbbc8d63adf34c"] = true
	if got := len(Tree(s, ui)); got != 4 {
		t.Errorf("%d nodes with the agent folded, want 4", got)
	}
}

// TestTreeP7Running: while the turn runs, "more as events arrive" closes the
// stage and both children sit directly under it; a plan update turns the
// heading to PLAN, observed.
func TestTreeP7Running(t *testing.T) {
	evs := probeEvents(t, "p7-codex-subagents.jsonl", codex.ProviderID, codex.NewMapper())
	s := state.New()
	ui := UI{Width: 120, Height: 40}
	mid := false
	for _, ev := range evs {
		s.Apply(ev)
		if s.Turn.Phase != state.Idle && labelled(s) == 2 && !mid {
			mid = true
			rows := texts(zoneA(s, ui, 60, 30, &Screen{}))
			if !strings.Contains(rows, "○ (more as events arrive)") {
				t.Errorf("running turn without 'more':\n%s", rows)
			}
			if strings.Count(rows, "child  /root/") != 2 || !strings.Contains(rows, "  ├─ ") {
				t.Errorf("children not under the stage:\n%s", rows)
			}

			p := *s
			p.Stages = append([]state.Stage(nil), s.Stages...)
			p.Apply(event.Event{Seq: 1 << 40, Kind: event.StageChanged, Origin: event.Observed,
				Payload: event.Plan{Steps: []event.PlanStep{{Step: "list files", Status: "completed"}, {Step: "create found.txt", Status: "inProgress"}, {Step: "reply", Status: "pending"}}}})
			plan := texts(zoneA(&p, ui, 60, 30, &Screen{}))
			for _, want := range []string{"PLAN · observed (turn/plan/updated)", "✓ 1 list files", "◉ 2 create found.txt", "○ 3 reply", "○ (more as events arrive)"} {
				if !strings.Contains(plan, want) {
					t.Errorf("plan lacks %q:\n%s", want, plan)
				}
			}
		}
	}
	if !mid {
		t.Fatal("never saw both children during the turn")
	}
	if rows := texts(zoneA(s, ui, 60, 30, &Screen{})); strings.Contains(rows, "more as events arrive") {
		t.Errorf("'more' after the turn ended:\n%s", rows)
	}
}

// labelled counts the agents whose spawn was seen.
func labelled(s *state.State) int {
	n := 0
	for _, a := range s.Agents {
		if strings.HasPrefix(a.Label, "/root/") {
			n++
		}
	}
	return n
}

// TestTreeWindowKeepsCursor: the cursor's row stays in the window.
func TestTreeWindowKeepsCursor(t *testing.T) {
	var rows []treeRow
	for i := range 20 {
		rows = append(rows, treeRow{node: Node{ID: string(rune('a' + i))}})
	}
	for _, c := range []string{"", "a", "e", "t"} {
		win := treeWindow(rows, c, 5)
		found := c == ""
		for _, r := range win {
			found = found || r.node.ID == c
		}
		if len(win) != 5 || !found {
			t.Errorf("cursor %q: window %d rows, cursor shown %v", c, len(win), found)
		}
	}
}
