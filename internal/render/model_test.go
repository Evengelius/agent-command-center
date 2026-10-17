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
