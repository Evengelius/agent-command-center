package state

import (
	"testing"

	"github.com/Evengelius/agent-command-center/internal/event"
)

// Plan 38 (brief amendment 39): a Codex change between turns taken at once,
// as Claude Code's; during a turn, with the next message; the fallback when
// Codex refuses thread/settings/update.

var sixUltra = event.Settings{Model: "gpt-6-sol", Choice: "gpt-6-sol", Effort: "ultra", EffortKnown: true, Source: "thread/start"}

// TestCodexAtOnce is criterion 38.7: a Codex choice between turns (NextTurn
// false) asks once, with no label, and its answer puts it in force with no
// notice; a second before the answer asks nothing; a model choice carries
// an effort the model lists, the one in force if it lists it, else its
// default; a choice of what is in force asks nothing.
func TestCodexAtOnce(t *testing.T) {
	s := modelsReady(t, "codex", codexList, sixUltra)
	if cmds := s.Do(ChooseEffort{Level: "ultra"}); cmds != nil || s.Models.Asked != nil {
		t.Errorf("the effort in force asked: %+v", cmds)
	}
	if cmds := s.Do(ChooseModel{ID: "gpt-6-sol"}); cmds != nil || s.Models.Asked != nil {
		t.Errorf("the model in force asked: %+v", cmds)
	}
	answer := func(model, effort string) {
		settle(s, event.Settings{Model: model, Choice: model, Effort: effort, EffortKnown: true, Source: "thread/settings/updated", Answers: true})
		if s.Models.Asked != nil || s.Models.InForce.Model != model || s.Models.InForce.Effort != effort || s.Session.Model != model {
			t.Errorf("after the answer: asked %+v, in force %+v, session %q", s.Models.Asked, s.Models.InForce, s.Session.Model)
		}
	}
	// GPT-6-Luna lists no Ultra: its default goes with it.
	cmds := s.Do(ChooseModel{ID: "gpt-6-luna"})
	if len(cmds) != 1 || cmds[0].Kind != CmdModels || cmds[0].Change != (ModelChange{Choice: "gpt-6-luna", Effort: "medium"}) || s.PendingLabel() != "" {
		t.Fatalf("commands %+v, label %q", cmds, s.PendingLabel())
	}
	if cmds := s.Do(ChooseEffort{Level: "low"}); cmds != nil {
		t.Errorf("a second change before the answer: %+v", cmds)
	}
	answer("gpt-6-luna", "medium")
	// GPT-6-Sol lists Medium: the effort in force goes with it.
	if cmds := s.Do(ChooseModel{ID: "gpt-6-sol"}); len(cmds) != 1 || cmds[0].Change != (ModelChange{Choice: "gpt-6-sol", Effort: "medium"}) {
		t.Errorf("commands %+v", cmds)
	}
	answer("gpt-6-sol", "medium")
	if cmds := s.Do(ChooseEffort{Level: "max"}); len(cmds) != 1 || cmds[0].Change != (ModelChange{Effort: "max"}) {
		t.Errorf("commands %+v", cmds)
	}
	answer("gpt-6-sol", "max")
	if n := notices(s); len(n) != 0 {
		t.Errorf("notices %q", n)
	}
}

// TestCodexDuringTurn is criterion 38.7 during a turn: a choice the
// session takes with the next turn (NextTurn, as Codex's while its turn
// runs) waits, labelled, and the answer that follows the turn's end puts
// it in force; one taken at once (a session that ended its turn before the
// state did) gets zone B's notice, naming Codex.
func TestCodexDuringTurn(t *testing.T) {
	s := modelsReady(t, "codex", codexList, sixUltra)
	s.Do(Send{Text: "hello"})
	if cmds := s.Do(ChooseEffort{Level: "low"}); cmds != nil {
		t.Errorf("at once during a turn: %+v", cmds)
	}
	if n := notices(s); len(n) != 1 || n[0] != "Codex's model settings change between turns: wait for the turn to end" {
		t.Errorf("notices %q", n)
	}
	if cmds := s.Do(ChooseEffort{Level: "low", NextTurn: true}); len(cmds) != 1 || s.PendingLabel() != "→ GPT-6-Sol Low (next turn)" {
		t.Errorf("with the next turn: %+v, %q", cmds, s.PendingLabel())
	}
	settle(s, event.Settings{Model: "gpt-6-sol", Choice: "gpt-6-sol", Effort: "low", EffortKnown: true, Source: "thread/settings/updated", Answers: true})
	if s.Models.Asked != nil || s.PendingLabel() != "" || s.Models.InForce.Effort != "low" {
		t.Errorf("after the answer: asked %+v, label %q, in force %+v", s.Models.Asked, s.PendingLabel(), s.Models.InForce)
	}
}

// TestCodexFallback is criterion 38.7's fallback: the next-turn event turns
// the change waiting for its answer into one waiting for the next message,
// labelled, with one notice in Codex's words; the next message's answer
// puts it in force.
func TestCodexFallback(t *testing.T) {
	s := modelsReady(t, "codex", codexList, sixUltra)
	s.Do(ChooseEffort{Level: "max"})
	const why = "Codex takes model changes with the next message: thread/settings/update refused by Codex: thread/settings/update requires experimentalApi capability"
	fall := event.Settings{Source: "thread/settings/update", NextTurn: true, Text: why}
	settle(s, fall)
	if a := s.Models.Asked; a == nil || *a != (ModelChange{Effort: "max", NextTurn: true}) || s.PendingLabel() != "→ GPT-6-Sol Max (next turn)" {
		t.Errorf("after the fallback: asked %+v, label %q", a, s.PendingLabel())
	}
	if s.Models.InForce != sixUltra {
		t.Errorf("in force %+v", s.Models.InForce)
	}
	settle(s, fall) // the same words again, right after: not repeated
	if n := notices(s); len(n) != 1 || n[0] != why {
		t.Errorf("notices %q", n)
	}
	// Now with the next message: a new choice adds to the waiting change.
	if cmds := s.Do(ChooseEffort{Level: "xhigh", NextTurn: true}); len(cmds) != 1 || cmds[0].Change != (ModelChange{Effort: "xhigh", NextTurn: true}) {
		t.Errorf("commands %+v", cmds)
	}
	settle(s, event.Settings{Model: "gpt-6-sol", Choice: "gpt-6-sol", Effort: "xhigh", EffortKnown: true, Source: "thread/read", Answers: true})
	if s.Models.Asked != nil || s.Models.InForce.Effort != "xhigh" {
		t.Errorf("after the answer: asked %+v, in force %+v", s.Models.Asked, s.Models.InForce)
	}
}

// TestClaudeSameStillAsks is 38.7's amendment: Claude Code's choice of what
// is in force still asks, as before plan 38.
func TestClaudeSameStillAsks(t *testing.T) {
	s := modelsReady(t, "claude-code", claudeList, fableMax)
	if cmds := s.Do(ChooseModel{ID: "fable"}); len(cmds) != 1 || s.Models.Asked == nil {
		t.Errorf("Claude Code's model in force: %+v", cmds)
	}
}
