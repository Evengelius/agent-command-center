package render

import (
	"strings"
	"testing"
	"time"

	"charm.land/lipgloss/v2"

	"github.com/Evengelius/agent-command-center/internal/event"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// TestAgentModelColumn is plan 5, item 7, criterion 3: zone C shows each
// agent's model in its colour, Claude's without "claude-", "model N/A"
// without one; the preview header shows the name in full.
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
			l, _ := rowOf(t, sc, "◉ "+padTo(c.label, 14)+padTo(c.model, 12))
			if st := styleOf(l, c.model); st != modelStyle(c.model) {
				t.Errorf("%v: %s's model %q in style %p, want %p", size, c.label, c.model, st, modelStyle(c.model))
			}
			if txt := texts([]line{l}); strings.Contains(txt, "claude-haiku") {
				t.Errorf("%v: row %q keeps the prefix", size, txt)
			}
		}
		if size[0] >= 200 { // the preview, at wide width
			if l, _ := rowOf(t, sc, "explore · LIVE"); !strings.Contains(texts([]line{l}), "claude-haiku-5-5") {
				t.Errorf("%v: preview header %q", size, texts([]line{l}))
			}
		}
	}
	if modelStyle("haiku-5-5") != modelStyle("claude-haiku-5-5") || modelStyle("gpt-6-sol") == stText {
		t.Error("the short name loses its colour")
	}
}

// TestAgentSourceAndEffort is plan 6, items 5 and 6: an agent's model with
// its source and its reasoning effort, in zone C, the preview and the
// inspection header; "N/A (not reported)" where the inspection shows them.
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
		sc := Draw(s, UI{Width: size[0], Height: size[1], Focus: "C", Selected: "k2"})
		for i, r := range strings.Split(sc.String(), "\n") {
			if w := lipgloss.Width(r); w > size[0] {
				t.Errorf("%v: row %d is %d wide", size, i, w)
			}
		}
		// The effort column as wide as the longest effort; "*" after a
		// requested model, its footnote in the header, as D's "†".
		for _, c := range []struct{ label, model, mark, effort string }{
			{"/root/a", "gpt-6-sol", "", "high"}, {"/root/b", "gpt-6-mini", "*", "low"}, {"/root/c", "model N/A", "", ""},
		} {
			l, _ := rowOf(t, sc, "◉ "+padTo(c.label, 14)+padTo(c.model+c.mark, 12)+padTo(c.effort, 5)+"announced")
			for _, x := range []string{c.effort, c.mark} {
				if x != "" && styleOf(l, x) != stGrey {
					t.Errorf("%v: %s's %q not grey", size, c.label, x)
				}
			}
		}
		if l, _ := rowOf(t, sc, "C · LIVE AGENTS"); !strings.Contains(texts([]line{l}), "* requested model · 0 ACTIVE") {
			t.Errorf("%v: C header %q", size, texts([]line{l}))
		}
		if size[0] >= 200 { // the preview, at wide width
			if l, _ := rowOf(t, sc, "/root/b · LIVE"); !strings.Contains(texts([]line{l}), "gpt-6-mini (requested) · effort low (requested)") {
				t.Errorf("%v: preview header %q", size, texts([]line{l}))
			}
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
	// Without an effort or a requested model in the list, no column is added.
	s.Agents["k1"].Effort, s.Agents["k2"].Effort = state.Value{}, state.Value{}
	s.Agents["k2"].Model.Source = "agent model"
	sc := Draw(s, UI{Width: 120, Height: 40, Focus: "C"})
	rowOf(t, sc, "◉ "+padTo("/root/b", 14)+padTo("gpt-6-mini", 12)+"announced")
	if l, _ := rowOf(t, sc, "C · LIVE AGENTS"); strings.Contains(texts([]line{l}), "requested") {
		t.Errorf("C header %q", texts([]line{l}))
	}
}

// TestAgentLabelColumn is plan 7, item 1: zone C's label column is the
// longest label and a space, at least 14 cells, so every row's model starts
// at the same cell; it never leaves the activity under 12 cells, and a label
// longer than the column ends in "…", whole in the inspection.
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
		sc := Draw(s, UI{Width: size[0], Height: size[1], Focus: "C"})
		row := func(model string) string { // C's row, not the preview's header
			for _, l := range sc.lines {
				if txt := texts([]line{l}); strings.Contains(txt, model+" ") && !strings.Contains(txt, "· LIVE") {
					return txt
				}
			}
			t.Fatalf("%v: no row for %s:\n%s", size, model, texts(sc.lines))
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
		// The running agent's activity keeps at least 12 cells.
		if txt := row("mod-2"); !strings.Contains(txt, "abcdefghijk") {
			t.Errorf("%v: activity cut under 12 cells: %q", size, txt)
		}
		// The 40-cell label is cut with "…" in C, and only when it must be.
		if txt := row("mod-4"); strings.Contains(txt, long) || !strings.Contains(txt, "…") {
			t.Errorf("%v: long label %q", size, txt)
		}
		if txt := row("mod-3"); !strings.Contains(txt, "/root/create_found") && size[0] < 160 {
			t.Errorf("%v: an 18-cell label cut where C has room: %q", size, txt)
		}
		ins := Draw(s, UI{Width: size[0], Height: size[1], Inspect: "k4"})
		if h := texts([]line{ins.lines[1]}); !strings.Contains(h, long) {
			t.Errorf("%v: inspection header %q lacks the whole label", size, h)
		}
	}
	// Labels that fit in 13 cells keep today's 14-cell column.
	short := state.New()
	short.Screen = state.ScreenDashboard
	short.Apply(event.Event{Seq: 1, At: at, Kind: event.AgentStarted, Origin: event.Observed, Agent: "k1",
		Payload: event.Agent{Key: "k1", Type: "thread", Label: "/root/abcdefg", Model: "mod-1"}})
	rowOf(t, Draw(short, UI{Width: 120, Height: 40}), "◉ "+padTo("/root/abcdefg", 14)+"mod-1")
}

// TestPreviewHeaderFallback is plan 7, item 2: the preview's right-hand
// header is the first that fits of the model, its source and the effort; the
// model and its source; the model alone; nothing. Read from the frame at
// every width from 160 to 220: the form drawn is one of these, the model
// whole, and the next longer form would not have fitted in its place.
func TestPreviewHeaderFallback(t *testing.T) {
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
	forms := map[string][]string{ // longest first
		"k1": {"gpt-6-mini (requested) · effort low (requested)", "gpt-6-mini (requested)", "gpt-6-mini"},
		"k2": {"gpt-6-sol (reported) · effort high (reported)", "gpt-6-sol (reported)", "gpt-6-sol"},
		"k3": {"model N/A (not reported)", "model N/A"},
	}
	seen := map[string]bool{}
	for k, want := range forms {
		label := s.Agents[k].Label
		for w := 160; w <= 220; w++ {
			sc := Draw(s, UI{Width: w, Height: 50, Focus: "C", Selected: k})
			l, _ := rowOf(t, sc, label+" · LIVE")
			row := texts([]line{l})
			top := strings.TrimSuffix(row[strings.LastIndex(row, "╭─ "):], "\n")
			rest := strings.TrimPrefix(top, "╭─ "+label+" · LIVE ")
			if rest == top {
				t.Fatalf("%s at %d: preview title cut: %q", k, w, top)
			}
			dashes := strings.TrimLeft(rest, "─")
			fill := lipgloss.Width(rest) - lipgloss.Width(dashes)
			right := strings.TrimSuffix(strings.TrimPrefix(dashes, " "), " ─╮")
			var next string // the next longer form, which must not fit
			room := fill + lipgloss.Width(right) - 1
			switch i := index(want, right); {
			case dashes == "╮": // no right-hand header: "╭─ title ────╮"
				next, room = want[len(want)-1], fill-4
			case i < 0:
				t.Fatalf("%s at %d: right-hand header %q is none of %q", k, w, right, want)
			case i > 0:
				next = want[i-1]
			}
			seen[k+"|"+right] = true
			if next != "" && lipgloss.Width(next) <= room {
				t.Errorf("%s at %d: %q drawn where %q fits (%d cells)", k, w, right, next, room)
			}
		}
		for _, f := range want { // every form is reached between 160 and 220
			if !seen[k+"|"+f] {
				t.Errorf("%s: %q never drawn between 160 and 220", k, f)
			}
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
