package render

import (
	"regexp"
	"sort"
	"strings"
	"testing"
	"time"

	"charm.land/lipgloss/v2"

	"github.com/Evengelius/agent-command-center/internal/event"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// zoneCInner draws the dashboard and returns zone C's rows inside its
// frame as text, without the two cells of border on each side, the same
// rows as lines (with their border spans), the frame and the panel's
// width.
func zoneCInner(t *testing.T, s *state.State, ui UI) (inner []string, rows []line, frame []string, lw int) {
	t.Helper()
	sc := Draw(s, ui)
	frame = strings.Split(strings.TrimSuffix(texts(sc.lines), "\n"), "\n")
	z, ok := zoneBoxes(frame)["C"]
	if !ok {
		t.Fatalf("no zone C box:\n%s", strings.Join(frame, "\n"))
	}
	for _, r := range frame[z.top+1 : z.bottom] {
		rs := []rune(r)
		inner = append(inner, string(rs[2:len(rs)-2]))
	}
	return inner, sc.lines[z.top+1 : z.bottom], frame, panelWidth(s, ui)
}

// panelWidth is zone C's panel width for the state at the screen's size
// (design D9).
func panelWidth(s *state.State, ui UI) int {
	agents := agentsByActivity(s)
	return cardColumns(agents, currentTools(s, nil, agents), ui, ui.Width-4).width
}

// listPart is the panel's part of an inner row: its first lw cells.
func listPart(row string, lw int) string {
	rs := []rune(row)
	return string(rs[:min(lw, len(rs))])
}

// checkGround checks a card row's cells between the margin cells (the
// panel's cells 1 to lw-2, after the frame's two) lie on the tool rows'
// ground, or do not, and that the margin cells, the frame and the panels'
// line never do (design D8).
func checkGround(t *testing.T, name string, l line, lw int, want bool) {
	t.Helper()
	cell := 0
	for _, x := range l {
		w := textWidth(x.text)
		from, to := cell, cell+w
		cell = to
		if w == 0 || from >= 2+lw+3 {
			continue
		}
		none := x.st == nil
		if !none {
			_, none = x.st.GetBackground().(lipgloss.NoColor)
		}
		inside := from >= 3 && to <= 2+lw-1
		switch {
		case inside && want && none:
			t.Errorf("%s: %q at cell %d is not on the ground:\n%s", name, x.text, from, texts([]line{l}))
		case inside && !want && !none:
			t.Errorf("%s: %q at cell %d is on the ground:\n%s", name, x.text, from, texts([]line{l}))
		case !inside && !none:
			t.Errorf("%s: %q at cell %d, outside the card, is on the ground:\n%s", name, x.text, from, texts([]line{l}))
		}
	}
}

// TestZoneCDesign is plan 27's criteria 27.1, 27.3 and 27.10 (brief
// amendments 26 and 27, the owner's image and its close-ups): the header
// row inside the frame, the rule, the panels' headings with the agent's
// status, the line between the panels, the panel's width, the card's two
// columns on their ground, and the timeline's entries.
func TestZoneCDesign(t *testing.T) {
	defer UsePalette(pickTrue)
	waiting := p3bAtWrite(t)
	done, now := stampedMock(t, "claude-quota-windows.jsonl", appClock)
	for _, c := range []struct {
		name                string
		s                   *state.State
		sym, counts, status string
	}{
		{"P3b at its Write", waiting, "●", "1 ACTIVE · 0 DONE", "● WAITING"},
		{"the quota mock", done, "✓", "0 ACTIVE · 1 DONE", "✓ COMPLETED"},
	} {
		for _, size := range [][2]int{{120, 40}, {208, 59}} {
			ui := UI{Width: size[0], Height: size[1], Now: now, Focus: "B"}
			rows, ls, frame, lw := zoneCInner(t, c.s, ui)
			z := zoneBoxes(frame)["C"]
			if strings.Contains(frame[z.top], "C · ") {
				t.Errorf("%s %v: a title in C's top edge: %q", c.name, size, frame[z.top])
			}
			if head := strings.TrimRight(rows[0], " "); !strings.HasPrefix(head, "C · LIVE AGENTS") || !strings.HasSuffix(head, c.counts) {
				t.Errorf("%s %v: C's header row %q, want \"C · LIVE AGENTS\" … %q", c.name, size, head, c.counts)
			}
			if strings.Trim(rows[1], "─") != "" {
				t.Errorf("%s %v: C's second row %q is not a rule", c.name, size, rows[1])
			}
			// 27.10 The panel as wide as its card, 40 cells at both sizes
			// (D9); the header in the title colour, the rule and the panels'
			// line in stPanelRule (D7).
			if lw != 40 {
				t.Errorf("%s %v: the panel is %d cells wide, want 40", c.name, size, lw)
			}
			if x, ok := spanOf(ls[0], "C · LIVE AGENTS"); !ok || x.st != stTitle {
				t.Errorf("%s %v: the header styled %v, want stTitle", c.name, size, x.st)
			}
			if x, ok := spanOf(ls[1], "────"); !ok || x.st != stPanelRule {
				t.Errorf("%s %v: the rule styled %v, want stPanelRule", c.name, size, x.st)
			}
			r3 := []rune(rows[2])
			if !strings.HasPrefix(rows[2], "AGENTS") || string(r3[lw:lw+3]) != " │ " || !strings.HasPrefix(string(r3[lw+3:]), "EXECUTION TIMELINE") || !strings.HasSuffix(strings.TrimRight(rows[2], " "), c.status) {
				t.Errorf("%s %v: the panels' headings %q, want AGENTS, the line at %d, EXECUTION TIMELINE and %q at the end", c.name, size, rows[2], lw, c.status)
			}
			for i := 2; i < len(rows); i++ {
				if rs := []rune(rows[i]); len(rs) < lw+3 || string(rs[lw:lw+3]) != " │ " {
					t.Errorf("%s %v: row %d has no panels' line at %d: %q", c.name, size, i+1, lw, rows[i])
				}
				if x, ok := spanOf(ls[i], " │ "); !ok || x.st != stPanelRule {
					t.Errorf("%s %v: row %d's panels' line styled %v, want stPanelRule", c.name, size, i+1, x.st)
				}
			}
			// 27.10 The card (D8): the symbol, the label in bold, the bar, the
			// model in grey; the clock, the tokens, the bar, "tool: " and the
			// tool in bold; nothing else; the only agent is the selected one,
			// its rows on the ground between the margin cells.
			first, second := listPart(rows[3], lw), listPart(rows[4], lw)
			if want := " " + c.sym + "  general-purpose     │  haiku-5-5"; strings.TrimRight(first, " ") != want {
				t.Errorf("%s %v: the card's first row %q, want %q", c.name, size, first, want)
			}
			if !regexp.MustCompile(`^    \d\d:\d\d   [\d.]+k tok   │  tool: Write *$`).MatchString(second) {
				t.Errorf("%s %v: the card's second row %q", c.name, size, second)
			}
			for _, bad := range []string{"waiting", "completed", "running", "next:", "approval", "Files", "Bash"} {
				if strings.Contains(first+second, bad) {
					t.Errorf("%s %v: %q on the card's rows:\n%s\n%s", c.name, size, bad, first, second)
				}
			}
			for _, l := range []line{ls[3], ls[4]} {
				checkGround(t, c.name, l, lw, true)
			}
			for _, x := range []struct {
				l    line
				text string
				st   *lipgloss.Style
			}{{ls[3], "general-purpose", stBold}, {ls[3], "haiku-5-5", stGrey}, {ls[4], "tool: ", stGrey}, {ls[4], "Write", stBold}} {
				if st := styleOf(x.l, x.text); st != tint(x.st) {
					t.Errorf("%s %v: %q styled %v, want %v on the ground", c.name, size, x.text, st, tint(x.st))
				}
			}
			bars := 0
			for _, l := range []line{ls[3], ls[4]} {
				for _, x := range l {
					if x.text == "│" && x.st == tint(stGrey) {
						bars++
					}
				}
			}
			if bars != 2 {
				t.Errorf("%s %v: %d grey bars on the card's rows, want one on each", c.name, size, bars)
			}
			// 27.3 The timeline's entries, the newest last, rules between them
			// and none after the last; "no activity yet" for an agent with no
			// entry (P3b's subagent has none at its Write request).
			right := func(i int) string { return string([]rune(rows[i])[lw+3:]) }
			if a := agentsByActivity(c.s)[0]; len(a.Messages) == 0 {
				if !strings.HasPrefix(right(3), "no activity yet") {
					t.Errorf("%s %v: an agent with no entry reads %q", c.name, size, right(3))
				}
			} else {
				entry := regexp.MustCompile(`^(\d\d:\d\d:\d\d)  (\S+)\s+│ \S`)
				var times []string
				rules := 0
				for i := 3; i < len(rows); i++ {
					if m := entry.FindStringSubmatch(right(i)); m != nil {
						times = append(times, m[1])
						if len(m[2]) > roleWidth {
							t.Errorf("%s %v: the role %q is wider than the column", c.name, size, m[2])
						}
					} else if r := right(i); strings.TrimRight(r, " ") != "" && strings.Trim(r, "─") == "" {
						rules++
					}
				}
				if len(times) == 0 || !sort.StringsAreSorted(times) {
					t.Errorf("%s %v: the timeline's entries %v, want some, the newest last:\n%s", c.name, size, times, strings.Join(rows, "\n"))
				}
				if rules < len(times)-1 || rules > len(times) {
					t.Errorf("%s %v: %d rules for %d entries", c.name, size, rules, len(times))
				}
				if last := right(len(rows) - 1); strings.TrimRight(last, " ") != "" && strings.Trim(last, "─") == "" {
					t.Errorf("%s %v: a rule after the last entry", c.name, size)
				}
			}
			// The old preview's heading carried the model's source after
			// "<label> · LIVE"; C's header "C · LIVE AGENTS" stays.
			for _, old := range []string{"(reported)", "(requested)", "(not reported)", "-purpose · LIVE"} {
				if strings.Contains(strings.Join(rows, "\n"), old) {
					t.Errorf("%s %v: the old preview's heading drawn (%q):\n%s", c.name, size, old, strings.Join(rows, "\n"))
				}
			}
		}
	}

	// 27.3 One entry's rows on a hand-made agent: the role in its colour, the
	// text on two rows at most, the bar on both, the second cut with "…";
	// the running status; the held hint; "no activity yet".
	s := state.New()
	long := "Working directory: " + strings.Repeat("x ", 200)
	s.Agents["a1"] = &state.Agent{Key: "a1", Label: "long", Status: state.AgentRunning, Messages: []state.Entry{
		{Role: "notice", At: appClock, Text: long},
		{Role: "assistant", At: appClock.Add(time.Second), Text: "short\nsecond line"},
	}}
	s.Agents["a2"] = &state.Agent{Key: "a2", Label: "quiet", Status: state.AgentCompleted}
	sc := &Screen{}
	status, rows := timeline(s, UI{Selected: "a1"}, 60, 10, sc)
	if got := texts([]line{status}); strings.TrimSpace(got) != "◉ RUNNING" {
		t.Errorf("the running agent's status %q", got)
	}
	if len(rows) != 4 || sc.Preview.Rows != 4 {
		t.Fatalf("%d rows for two entries (%d in the pane):\n%s", len(rows), sc.Preview.Rows, texts(rows))
	}
	first, second, third, fourth := texts([]line{rows[0]}), texts([]line{rows[1]}), texts([]line{rows[2]}), texts([]line{rows[3]})
	if !regexp.MustCompile(`^\d\d:\d\d:\d\d  notice    │ Working directory: x( x)*\s*\n$`).MatchString(first) {
		t.Errorf("the entry's first row %q", first)
	}
	if !regexp.MustCompile(`^ {20}│ (x )*x…\s*\n$`).MatchString(second) {
		t.Errorf("the entry's second row %q, want the bar, the text cut with …", second)
	}
	if strings.Trim(strings.TrimSpace(third), "─") != "" {
		t.Errorf("no rule between the entries: %q", third)
	}
	if !strings.Contains(fourth, "assistant │ short second line") {
		t.Errorf("the second entry's row %q", fourth)
	}
	if x, ok := spanOf(rows[0], "notice    "); !ok || x.st != stSpeaker {
		t.Errorf("the notice's role styled %v", x.st)
	}
	if x, ok := spanOf(rows[3], "assistant "); !ok || x.st != stResult {
		t.Errorf("the assistant's role styled %v", x.st)
	}
	// The bar before the text in grey, on both rows (D7).
	for i, r := range rows[:2] {
		if x, ok := spanOf(r, "│ "); !ok || x.st != stGrey {
			t.Errorf("row %d's bar styled %v, want stGrey", i, x.st)
		}
	}
	for _, r := range rows {
		if w := r.width(); w != 60 {
			t.Errorf("a timeline row %d cells wide: %q", w, texts([]line{r}))
		}
	}
	if status, rows := timeline(s, UI{Selected: "a1", Preview: Pane{Hold: true, Top: 0}}, 60, 2, &Screen{}); len(rows) != 2 || !strings.Contains(texts([]line{status}), "↓ 2 · End follows") {
		t.Errorf("held at the top: %d rows, the heading's right %q", len(rows), texts([]line{status}))
	}
	if status, rows := timeline(s, UI{Selected: "a2"}, 60, 3, &Screen{}); len(rows) != 1 || !strings.Contains(texts(rows), "no activity yet") || !strings.Contains(texts([]line{status}), "✓ COMPLETED") {
		t.Errorf("an agent with no entry: %q, status %q", texts(rows), texts([]line{status}))
	}

	// 27.10 Two agents, the later one selected: its rows on the ground, the
	// other's not; an agent without a tool reads "tool: none yet"; on 16
	// colours the same characters and no ground.
	two := state.New()
	two.Screen = state.ScreenDashboard
	two.Context.Provider = "claude-code"
	at := time.Date(2026, 10, 22, 19, 7, 0, 0, time.Local)
	for i, k := range []string{"k1", "k2"} {
		two.Apply(event.Event{Seq: uint64(i + 1), At: at.Add(time.Duration(i) * time.Second), Kind: event.AgentStarted, Origin: event.Observed, Agent: k,
			Payload: event.Agent{Key: k, Type: "subagent", Label: "agent-" + k}})
	}
	ui := UI{Width: 120, Height: 40, Now: at.Add(5 * time.Second), Focus: "C", Selected: "k2"}
	inner, ls, _, lw := zoneCInner(t, two, ui)
	if !strings.Contains(inner[3], "agent-k2") || !strings.Contains(inner[5], "agent-k1") {
		t.Fatalf("two agents: rows\n%s", strings.Join(inner[3:7], "\n"))
	}
	if !strings.Contains(listPart(inner[4], lw), "│  tool: none yet") {
		t.Errorf("an agent without a tool: %q", listPart(inner[4], lw))
	}
	for i, want := range map[int]bool{3: true, 4: true, 5: false, 6: false} {
		checkGround(t, "two agents, row "+string(rune('0'+i)), ls[i], lw, want)
	}
	before := strings.Join(inner, "\n")
	UsePalette(pickANSI)
	inner, ls, _, lw = zoneCInner(t, two, ui)
	if strings.Join(inner, "\n") != before {
		t.Errorf("16 colours draw other characters:\n%s\n--- true colour ---\n%s", strings.Join(inner, "\n"), before)
	}
	for i := 3; i <= 6; i++ {
		checkGround(t, "16 colours, row "+string(rune('0'+i)), ls[i], lw, false)
	}
	UsePalette(pickTrue)
}

// TestCardColumns is criterion 27.11's rule on the function itself (design
// D9): the columns' least widths, the longest label and tool over the
// agents, and the three steps the panel gives way in.
func TestCardColumns(t *testing.T) {
	ag := func(label, model, tool string) *state.Agent {
		a := &state.Agent{Key: label, Label: label, LastTool: tool}
		if model != "" {
			a.Model = state.Value{Present: true, Text: model, Source: "agent model"}
		}
		return a
	}
	tools := func(as []*state.Agent) map[string]toolInfo { return currentTools(state.New(), nil, as) }
	one := []*state.Agent{ag("general-purpose", "claude-haiku-5-5", "Write")}
	for _, inner := range []int{96, 116, 152, 204} {
		if c := cardColumns(one, tools(one), UI{}, inner); c.width != 40 || c.label != 17 || c.tool != 11 {
			t.Errorf("inner %d: P3b's card %+v, want 40 = 12 + 17 + 11", inner, c)
		}
	}
	long := []*state.Agent{ag("/root/"+strings.Repeat("x", 33)+"z", "mod-4", ""), ag("/root/abcdefg", "mod-2", "abcdefghijklmnopqrstuvwxyz")}
	for _, c := range []struct{ inner, width, label, tool int }{{116, 45, 17, 16}, {156, 61, 33, 16}, {196, 74, 40, 22}} {
		if got := cardColumns(long, tools(long), UI{}, c.inner); got.width != c.width || got.label != c.label || got.tool != c.tool {
			t.Errorf("inner %d: %+v, want width %d, label %d, tool %d", c.inner, got, c.width, c.label, c.tool)
		}
	}
	codex := []*state.Agent{ag("/root/create_found", "", "commandExecution")}
	if c := cardColumns(codex, tools(codex), UI{}, 204); c.width != 52 || c.label != 18 || c.tool != 22 {
		t.Errorf("a Codex card at 208 columns: %+v, want 52 = 12 + 18 + 22", c)
	}
	if c := cardColumns(codex, tools(codex), UI{}, 116); c.width != 45 || c.label != 17 || c.tool != 16 {
		t.Errorf("a Codex card at 120 columns: %+v, want 45 = 12 + 17 + 16", c)
	}
	if c := cardColumns(nil, nil, UI{}, 96); c.width != 40 {
		t.Errorf("no agent: %+v", c)
	}
}

// TestCHeight is criterion 27.4's rule (design D4): C's rows by the agents
// in play and the rows A, B and C share.
func TestCHeight(t *testing.T) {
	for _, c := range []struct{ n, region, minB, want int }{
		{0, 56, 14, 4}, {1, 56, 14, 16}, {1, 57, 14, 16}, {1, 41, 12, 13}, {1, 37, 12, 12},
		{1, 21, 14, 7}, {1, 22, 14, 7}, {1, 23, 14, 7}, {1, 24, 14, 8}, {6, 37, 12, 17}, {20, 37, 12, 18}, {20, 57, 14, 28}, {20, 21, 12, 9},
	} {
		if got := cHeight(c.n, c.region, c.minB, false); got != c.want {
			t.Errorf("cHeight(%d agents, %d rows, B at least %d) = %d, want %d", c.n, c.region, c.minB, got, c.want)
		}
	}
	for _, c := range []struct{ n, region, want int }{{0, 21, 3}, {1, 21, 3}, {6, 21, 5}} {
		if got := cHeight(c.n, c.region, 0, true); got != c.want {
			t.Errorf("narrow cHeight(%d, %d) = %d, want %d", c.n, c.region, got, c.want)
		}
	}
}
