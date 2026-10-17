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
