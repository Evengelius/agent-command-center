package render

import (
	"strings"
	"testing"
	"time"

	"charm.land/lipgloss/v2"

	"github.com/Evengelius/agent-command-center/internal/event"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// TestAgentModelColumn is plan 5, item 7, criterion 3, as plan 27's second
// pass draws the card (design D8): zone C shows each agent's model,
// Claude's without "claude-", "model N/A" without one, after the card's
// bar in grey from 100 columns, and in its family's colour on the narrow
// row under 100; the inspection header shows the name in full.
func TestAgentModelColumn(t *testing.T) {
	s := state.New()
	s.Screen = state.ScreenDashboard
	s.Context.Provider = "claude-code"
	at := time.Date(2026, 10, 17, 9, 30, 5, 0, time.Local)
	seq := uint64(0)
	apply := func(k event.Kind, agent string, p any) {
		seq++
		s.Apply(event.Event{Seq: seq, At: at.Add(time.Duration(seq) * time.Second), Kind: k, Origin: event.Observed, Agent: agent, Payload: p})
	}
	apply(event.SessionStarted, "", event.Started{SessionID: "s1", Model: "claude-opus-5-5"})
	apply(event.AgentStarted, "k1", event.Agent{Key: "k1", Type: "subagent", Label: "explore"})
	apply(event.AgentProgress, "k1", event.Agent{Key: "k1", Model: "claude-haiku-5-5"})
	apply(event.AgentStarted, "k2", event.Agent{Key: "k2", Type: "thread", Label: "/root/scan", Model: "gpt-6-sol"})
	apply(event.AgentStarted, "k3", event.Agent{Key: "k3", Type: "subagent", Label: "review"})

	for _, size := range [][2]int{{80, 24}, {120, 40}, {200, 60}} {
		sc := Draw(s, UI{Width: size[0], Height: size[1], Focus: "C", Selected: "k1"})
		for i, r := range strings.Split(sc.String(), "\n") {
			if w := lipgloss.Width(r); w > size[0] {
				t.Errorf("%v: row %d is %d wide", size, i, w)
			}
		}
		for _, c := range []struct{ label, model string }{{"explore", "haiku-5-5"}, {"/root/scan", "gpt-6-sol"}, {"review", "model N/A"}} {
			var l line
			if size[0] >= standardWidth {
				// The card: the label padded to the 17-cell column, the bar,
				// the model in grey (the selected agent's row on the ground).
				l, _ = rowOf(t, sc, cardLead("◉", c.label, 17)+c.model)
				if st := styleOf(l, c.model); st != stGrey && st != tint(stGrey) {
					t.Errorf("%v: %s's model %q in style %p, want grey", size, c.label, c.model, st)
				}
			} else {
				l, _ = rowOf(t, sc, agentLead(size[0], "◉", c.label, 14)+padTo(c.model, 12))
				if st := styleOf(l, c.model); st != modelStyle(c.model) {
					t.Errorf("%v: %s's model %q in style %p, want %p", size, c.label, c.model, st, modelStyle(c.model))
				}
			}
			if txt := texts([]line{l}); strings.Contains(txt, "claude-haiku") {
				t.Errorf("%v: row %q keeps the prefix", size, txt)
			}
		}
		// The timeline's heading stands right of the list from 100 columns
		// (plan 27); the model in full is the inspection's.
		if size[0] >= standardWidth {
			if l, _ := rowOf(t, sc, "EXECUTION TIMELINE"); !strings.Contains(texts([]line{l}), "AGENTS") {
				t.Errorf("%v: the panels' headings %q", size, texts([]line{l}))
			}
		}
	}
	if modelStyle("haiku-5-5") != modelStyle("claude-haiku-5-5") || modelStyle("gpt-6-sol") == stText {
		t.Error("the short name loses its colour")
	}
}

// TestAgentSourceAndEffort is plan 6, items 5 and 6: an agent's model with
// its source and its reasoning effort. Since plan 27's second pass the card
// shows the model with a requested model's "*" and nothing else (design
// D8); the source and the effort are the inspection header's, "N/A (not
// reported)" where it has none.
func TestAgentSourceAndEffort(t *testing.T) {
	s := state.New()
	s.Screen = state.ScreenDashboard
	s.Context.Provider = "codex"
	at := time.Date(2026, 10, 17, 9, 30, 5, 0, time.Local)
	seq := uint64(0)
	apply := func(k event.Kind, agent string, p any) {
		seq++
		s.Apply(event.Event{Seq: seq, At: at.Add(time.Duration(seq) * time.Second), Kind: k, Origin: event.Observed, Agent: agent, Payload: p})
	}
	apply(event.SessionStarted, "", event.Started{SessionID: "s1", Model: "gpt-6-sol"})
	apply(event.AgentStarted, "k1", event.Agent{Key: "k1", Type: "thread", Label: "/root/a", Model: "gpt-6-sol", Effort: "high"})
	apply(event.AgentStarted, "k2", event.Agent{Key: "k2", Type: "thread", Label: "/root/b"})
	apply(event.AgentProgress, "k2", event.Agent{Key: "k2", Model: "gpt-6-mini", Effort: "low", ModelRequested: true})
	apply(event.AgentStarted, "k3", event.Agent{Key: "k3", Type: "thread", Label: "/root/c"})

	for _, size := range [][2]int{{120, 40}, {200, 60}} {
		ui := UI{Width: size[0], Height: size[1], Focus: "C", Selected: "k2"}
		sc := Draw(s, ui)
		for i, r := range strings.Split(sc.String(), "\n") {
			if w := lipgloss.Width(r); w > size[0] {
				t.Errorf("%v: row %d is %d wide", size, i, w)
			}
		}
		// "*" after a requested model, its footnote in the header, as D's
		// "†"; no effort on the card.
		lw := panelWidth(s, ui)
		for _, c := range []struct{ label, model, mark string }{
			{"/root/a", "gpt-6-sol", ""}, {"/root/b", "gpt-6-mini", "*"}, {"/root/c", "model N/A", ""},
		} {
			l, _ := rowOf(t, sc, cardLead("◉", c.label, 17)+c.model+c.mark)
			if st := styleOf(l, c.model+c.mark); st != stGrey && st != tint(stGrey) {
				t.Errorf("%v: %s's model %q not grey", size, c.label, c.model+c.mark)
			}
			if card := listPart(string([]rune(texts([]line{l}))[2:]), lw); strings.Contains(card, "high") || strings.Contains(card, "low") {
				t.Errorf("%v: the effort on %s's card: %q", size, c.label, card)
			}
		}
		if l, _ := rowOf(t, sc, "C · LIVE AGENTS"); !strings.Contains(texts([]line{l}), "* requested model · 0 ACTIVE") {
			t.Errorf("%v: C header %q", size, texts([]line{l}))
		}
		// The selected agent's status at the timeline heading's right end
		// (plan 27); its model's source and its effort are the inspection's.
		if l, _ := rowOf(t, sc, "EXECUTION TIMELINE"); !strings.HasSuffix(strings.TrimRight(texts([]line{l}), " │\n"), "◉ ANNOUNCED") {
			t.Errorf("%v: the timeline heading %q", size, texts([]line{l}))
		}
		for k, want := range map[string]string{
			"k1": "/root/a · gpt-6-sol (reported) · ◉ announced",
			"k2": "/root/b · gpt-6-mini (requested) · ◉ announced",
			"k3": "/root/c · model N/A (not reported) · ◉ announced",
		} {
			sc := Draw(s, UI{Width: size[0], Height: size[1], Inspect: k})
			h := texts([]line{sc.lines[1]})
			e := map[string]string{"k1": "effort high (reported)", "k2": "effort low (requested)", "k3": "effort N/A (not reported)"}[k]
			if !strings.Contains(h, want) || !strings.Contains(h, e) {
				t.Errorf("%v: %s's header %q, want %q and %q", size, k, h, want, e)
			}
		}
	}
	// Without a requested model in the list, no footnote.
	s.Agents["k1"].Effort, s.Agents["k2"].Effort = state.Value{}, state.Value{}
	s.Agents["k2"].Model.Source = "agent model"
	sc := Draw(s, UI{Width: 120, Height: 40, Focus: "C"})
	rowOf(t, sc, cardLead("◉", "/root/b", 17)+"gpt-6-mini")
	if l, _ := rowOf(t, sc, "C · LIVE AGENTS"); strings.Contains(texts([]line{l}), "requested") {
		t.Errorf("C header %q", texts([]line{l}))
	}
}

// TestAgentLabelColumn is plan 7, item 1, as plan 27's second pass sizes
// the card (design D9): the label column is the longest label over the
// agents, at least 17 cells, so every row's model starts at the same cell;
// the panel gives way in steps when the columns would exceed its most
// width, the tool column first down to 16, then the label column down to
// 17; a label longer than the column ends in "…", whole in the inspection,
// and so does a tool name longer than the tool column.
func TestAgentLabelColumn(t *testing.T) {
	long := "/root/" + strings.Repeat("x", 33) + "z" // 40 cells
	labels := []string{"a", "/root/abcdefg", "/root/create_found", long}
	s := state.New()
	s.Screen = state.ScreenDashboard
	s.Context.Provider = "codex"
	at := time.Date(2026, 10, 17, 9, 30, 5, 0, time.Local)
	for i, l := range labels {
		k := string(rune('1' + i))
		s.Apply(event.Event{Seq: uint64(i + 1), At: at, Kind: event.AgentStarted, Origin: event.Observed, Agent: "k" + k,
			Payload: event.Agent{Key: "k" + k, Type: "thread", Label: l, Model: "mod-" + k}})
	}
	a := s.Agents["k2"]
	a.Status, a.LastTool = state.AgentRunning, "abcdefghijklmnopqrstuvwxyz"

	for _, size := range [][2]int{{120, 40}, {160, 50}, {200, 60}} {
		ui := UI{Width: size[0], Height: size[1], Focus: "C"}
		sc := Draw(s, ui)
		agents := agentsByActivity(s)
		cols := cardColumns(agents, currentTools(s, agents), ui, size[0]-4)
		// 27.11: the panel 45, 61 and 74 cells; the label column 17, 33 and
		// 40; the tool column 16, 16 and 22.
		if want := map[int][3]int{120: {45, 17, 16}, 160: {61, 33, 16}, 200: {74, 40, 22}}[size[0]]; [3]int{cols.width, cols.label, cols.tool} != want {
			t.Errorf("%v: the card's columns %+v, want width %d, label %d, tool %d", size, cols, want[0], want[1], want[2])
		}
		row := func(model string) string { // the card's first row holding the model: the panel's part
			for _, l := range sc.lines {
				rs := []rune(strings.TrimSuffix(texts([]line{l}), "\n"))
				if len(rs) < 2+cols.width {
					continue
				}
				if card := string(rs[2 : 2+cols.width]); strings.Contains(card, "│  "+model) {
					return card
				}
			}
			t.Fatalf("%v: no card for %s:\n%s", size, model, texts(sc.lines))
			return ""
		}
		col := -1
		for i := range labels {
			model := "mod-" + string(rune('1'+i))
			txt := row(model)
			c := lipgloss.Width(txt[:strings.Index(txt, model)])
			if col < 0 {
				col = c
			} else if c != col {
				t.Errorf("%v: %s's model at cell %d, the first row's at %d:\n%s", size, labels[i], c, col, txt)
			}
		}
		// The running agent's tool, cut to the tool column with "…".
		if wantTool := "tool: " + fitLabel(a.LastTool, cols.tool-6); !strings.Contains(texts(sc.lines), wantTool) {
			t.Errorf("%v: no %q:\n%s", size, wantTool, texts(sc.lines))
		}
		// The 40-cell label is whole only where the label column reaches 40
		// (200 columns), the 18-cell one where it reaches 18 (160 and 200);
		// elsewhere they end in "…", whole in the inspection.
		if txt := row("mod-4"); cols.label >= 40 && (!strings.Contains(txt, long) || strings.Contains(txt, "…")) || cols.label < 40 && !strings.Contains(txt, "…") {
			t.Errorf("%v: long label %q (column %d)", size, txt, cols.label)
		}
		if txt := row("mod-3"); cols.label >= 18 && !strings.Contains(txt, "/root/create_found") || cols.label < 18 && strings.Contains(txt, "/root/create_found") {
			t.Errorf("%v: an 18-cell label %q (column %d)", size, txt, cols.label)
		}
		ins := Draw(s, UI{Width: size[0], Height: size[1], Inspect: "k4"})
		if h := texts([]line{ins.lines[1]}); !strings.Contains(h, long) {
			t.Errorf("%v: inspection header %q lacks the whole label", size, h)
		}
	}
	// At 80 columns, one row an agent: the long label is cut, the running
	// agent's activity keeps its 12 cells.
	narrow := Draw(s, UI{Width: 80, Height: 24, Focus: "C"})
	for _, l := range narrow.lines {
		txt := texts([]line{l})
		if strings.Contains(txt, "mod-4") && (strings.Contains(txt, long) || !strings.Contains(txt, "…")) {
			t.Errorf("80x24: long label %q", txt)
		}
		if strings.Contains(txt, "mod-2") && !strings.Contains(txt, "abcdefghijk") {
			t.Errorf("80x24: activity cut under 12 cells: %q", txt)
		}
	}
	// Labels that fit in 17 cells keep the 17-cell column.
	short := state.New()
	short.Screen = state.ScreenDashboard
	short.Apply(event.Event{Seq: 1, At: at, Kind: event.AgentStarted, Origin: event.Observed, Agent: "k1",
		Payload: event.Agent{Key: "k1", Type: "thread", Label: "/root/abcdefg", Model: "mod-1"}})
	rowOf(t, Draw(short, UI{Width: 120, Height: 40}), cardLead("◉", "/root/abcdefg", 17)+"mod-1")
}

// agentLead is the start of an agent's row in zone C under 100 columns,
// with a label column col cells wide: the symbol, a cell, the label padded
// to the column (plan 18's one-row list).
func agentLead(w int, sym, label string, col int) string {
	if w >= standardWidth {
		return cardLead(sym, label, col)
	}
	return sym + " " + padTo(label, col)
}

// cardLead is the start of an agent's card row from 100 columns (plan 27,
// design D8): the margin cell, the symbol, two cells, the label padded to
// the label column, three cells, the bar and two cells.
func cardLead(sym, label string, col int) string {
	return " " + sym + "  " + label + strings.Repeat(" ", max(col-textWidth(label), 0)) + "   │  "
}

// TestListModelAcrossWidths stands where plan 7's preview-heading fallback
// test stood (the heading went with plan 27): from 160 to 220 columns the
// card's model column shows each agent's model short, "*" after a requested
// one and "model N/A" without one, C's header carries the "* requested
// model" footnote, and the timeline's heading the selected agent's status;
// the model's source and the effort are the inspection's
// (TestAgentSourceAndEffort).
func TestListModelAcrossWidths(t *testing.T) {
	s := state.New()
	s.Screen = state.ScreenDashboard
	s.Context.Provider = "codex"
	at := time.Date(2026, 10, 17, 9, 30, 5, 0, time.Local)
	seq := uint64(0)
	apply := func(k event.Kind, agent string, p any) {
		seq++
		s.Apply(event.Event{Seq: seq, At: at, Kind: k, Origin: event.Observed, Agent: agent, Payload: p})
	}
	apply(event.AgentStarted, "k1", event.Agent{Key: "k1", Type: "thread", Label: "/root/create_found"})
	apply(event.AgentProgress, "k1", event.Agent{Key: "k1", Model: "gpt-6-mini", Effort: "low", ModelRequested: true})
	apply(event.AgentStarted, "k2", event.Agent{Key: "k2", Type: "thread", Label: "/root/list_files", Model: "gpt-6-sol", Effort: "high"})
	apply(event.AgentStarted, "k3", event.Agent{Key: "k3", Type: "thread", Label: "/root/c"})
	want := map[string]string{"k1": "gpt-6-mini*", "k2": "gpt-6-sol", "k3": "model N/A"}
	for w := 160; w <= 220; w++ {
		ui := UI{Width: w, Height: 50, Focus: "C", Selected: "k1"}
		sc := Draw(s, ui)
		lw := panelWidth(s, ui)
		for k, m := range want {
			// The card's first row: the label in the panel's part (zone A's
			// tree names the agent too, after "├─" or "└─").
			txt := ""
			for _, l := range sc.lines {
				x := strings.TrimSuffix(texts([]line{l}), "\n")
				rs := []rune(x)
				if len(rs) < 2+lw || strings.Contains(x, "├─") || strings.Contains(x, "└─") {
					continue
				}
				if card := string(rs[2 : 2+lw]); strings.Contains(card, s.Agents[k].Label) {
					txt = card
					break
				}
			}
			if !strings.Contains(txt, "│  "+m) {
				t.Errorf("%d: %s's card %q lacks %q after the bar", w, k, txt, m)
			}
		}
		if l, _ := rowOf(t, sc, "C · LIVE AGENTS"); !strings.Contains(texts([]line{l}), "* requested model · 0 ACTIVE") {
			t.Errorf("%d: C's header %q", w, texts([]line{l}))
		}
		if l, _ := rowOf(t, sc, "EXECUTION TIMELINE"); !strings.HasSuffix(strings.TrimRight(texts([]line{l}), " │\n"), "◉ ANNOUNCED") {
			t.Errorf("%d: the timeline heading %q", w, texts([]line{l}))
		}
	}
}
func index(xs []string, x string) int {
	for i, v := range xs {
		if v == x {
			return i
		}
	}
	return -1
}
