package state

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/Evengelius/agent-command-center/internal/event"
)

// Plan 34 (brief amendment 34): the session's models, the settings in
// force, a change asked for and its answer.

var claudeList = []event.ModelEntry{
	{ID: "default", Model: "claude-opus-5-5", Name: "Default (recommended)", Efforts: []string{"low", "medium", "high", "xhigh", "max"}},
	{ID: "fable", Model: "claude-fable-5-1", Name: "Fable 5.1", Efforts: []string{"low", "medium", "high", "xhigh", "max"}},
	{ID: "claude-haiku-4-5-20251001", Model: "claude-haiku-4-5-20251001", Name: "Haiku 4.5"},
	{ID: "claude-opus-4-6", Model: "claude-opus-4-6", Name: "Opus 4.6", Efforts: []string{"low", "medium", "high", "max"}},
}

var codexList = []event.ModelEntry{
	{ID: "gpt-6.1-sol", Model: "gpt-6.1-sol", Name: "GPT-6.1-Sol", Efforts: []string{"low", "medium", "high", "xhigh", "max", "ultra"}, DefaultEffort: "low", Default: true},
	{ID: "gpt-6-sol", Model: "gpt-6-sol", Name: "GPT-6-Sol", Efforts: []string{"low", "medium", "high", "xhigh", "max", "ultra"}, DefaultEffort: "medium"},
	{ID: "gpt-6-luna", Model: "gpt-6-luna", Name: "GPT-6-Luna", Efforts: []string{"low", "medium", "high", "xhigh", "max"}, DefaultEffort: "medium"},
}

// modelsReady is a ready session of provider with the list and the
// settings in force applied as events.
func modelsReady(t *testing.T, provider string, list []event.ModelEntry, in event.Settings) *State {
	t.Helper()
	s := New()
	s.Context = Context{Provider: provider}
	s.Startup.Phase = Starting
	at := time.Date(2026, 10, 23, 12, 0, 0, 0, time.UTC)
	for i, ev := range []event.Event{
		{Kind: event.SessionSpawned, Payload: event.Spawned{}},
		{Kind: event.SessionReady, Payload: event.Ready{}},
		{Kind: event.ModelsListed, Payload: event.Models{Entries: list, Source: "test"}},
		{Kind: event.ModelSettings, Payload: in},
	} {
		ev.Seq, ev.At, ev.Provider, ev.Origin = uint64(i+1), at, provider, event.Observed
		s.Apply(ev)
	}
	if s.Session.Phase != Ready || len(s.Models.Entries) != len(list) {
		t.Fatalf("session %q with %d models", s.Session.Phase, len(s.Models.Entries))
	}
	return s
}

func settle(s *State, p event.Settings) {
	s.Apply(event.Event{Seq: uint64(len(s.Log) + 1), At: time.Date(2026, 10, 23, 12, 0, 1, 0, time.UTC), Kind: event.ModelSettings, Origin: event.Observed, Payload: p})
}

func notices(s *State) []string {
	var out []string
	for _, e := range s.Conversation {
		if e.Role == "notice" {
			out = append(out, e.Text)
		}
	}
	return out
}

var fableMax = event.Settings{Model: "claude-fable-5-1", Choice: "fable", Effort: "max", EffortKnown: true, Ultracode: true, UltracodeAvailable: true,
	UltracodeKnown: true, Source: "get_settings", Undocumented: true}

// TestChooseClaudeModel is criteria 7, 12 and 13 in the state: a choice is
// asked for and handed to the session; until its answer the settings in
// force stay, and a second choice waits; the answer puts the new model in
// force, zone A's and zone B's model with it.
func TestChooseClaudeModel(t *testing.T) {
	s := modelsReady(t, "claude-code", claudeList, fableMax)
	if s.Session.Model != "claude-fable-5-1" || s.Models.Current().ID != "fable" {
		t.Fatalf("at the start: session %q, current %+v", s.Session.Model, s.Models.Current())
	}
	cmds := s.Do(ChooseModel{ID: "claude-opus-4-6"})
	if len(cmds) != 1 || cmds[0].Kind != CmdModels || cmds[0].Change != (ModelChange{Choice: "claude-opus-4-6"}) {
		t.Fatalf("commands %+v", cmds)
	}
	if s.Models.Current().ID != "fable" || s.Session.Model != "claude-fable-5-1" {
		t.Error("the model changed before the answer")
	}
	if cmds := s.Do(ChooseModel{ID: "default"}); cmds != nil {
		t.Errorf("a second choice while the first waits: %+v", cmds)
	}
	settle(s, event.Settings{Model: "claude-opus-4-6", Choice: "claude-opus-4-6", Effort: "max", EffortKnown: true, UltracodeKnown: true,
		Source: "get_settings", Undocumented: true, Answers: true})
	if s.Models.Asked != nil || s.Models.Current().ID != "claude-opus-4-6" || s.Session.Model != "claude-opus-4-6" {
		t.Errorf("after the answer: asked %+v, current %+v, session %q", s.Models.Asked, s.Models.Current(), s.Session.Model)
	}
	if v := s.Metrics["model"]; v.Text != "claude-opus-4-6" || v.Documented || v.Source != "get_settings" {
		t.Errorf("the details view's model %+v", v)
	}
	if n := notices(s); len(n) != 0 {
		t.Errorf("notices %q", n)
	}
}

// TestRefusedAndKept is criterion 12: a refusal leaves the settings in
// force and says why in zone B; an answer that keeps another value than
// the one asked for says what the CLI kept.
func TestRefusedAndKept(t *testing.T) {
	s := modelsReady(t, "claude-code", claudeList, fableMax)
	s.Do(ChooseModel{ID: "claude-opus-4-6"})
	settle(s, event.Settings{Source: "set_model", Answers: true, Refused: true, Text: "set_model refused by Claude Code: Model 'x' not found"})
	if s.Models.Asked != nil || s.Models.InForce.Model != "claude-fable-5-1" {
		t.Errorf("after the refusal: asked %+v, in force %+v", s.Models.Asked, s.Models.InForce)
	}
	if n := notices(s); len(n) != 1 || !strings.Contains(n[0], "Model 'x' not found") {
		t.Errorf("notices %q", n)
	}
	s.Do(ChooseEffort{Level: "low"})
	settle(s, event.Settings{Model: "claude-fable-5-1", Choice: "fable", Effort: "max", EffortKnown: true, Source: "get_settings", Answers: true})
	if n := notices(s); len(n) != 2 || n[1] != "Claude Code kept the effort Max (asked: Low)" {
		t.Errorf("notices %q", n)
	}
	if s.Models.InForce.Effort != "max" {
		t.Errorf("effort in force %q", s.Models.InForce.Effort)
	}
}

// TestEffortChoices is criteria 6 and 8: only a level the model lists is
// asked for; a model without levels takes none; an effort a variable holds
// is not asked for, and zone B says why.
func TestEffortChoices(t *testing.T) {
	s := modelsReady(t, "claude-code", claudeList, event.Settings{Model: "claude-opus-4-6", Choice: "claude-opus-4-6", Effort: "high", EffortKnown: true})
	if cmds := s.Do(ChooseEffort{Level: "xhigh"}); cmds != nil {
		t.Errorf("xhigh asked for on Opus 4.6: %+v", cmds)
	}
	if cmds := s.Do(ChooseEffort{Level: "max"}); len(cmds) != 1 || cmds[0].Change.Effort != "max" {
		t.Errorf("max on Opus 4.6: %+v", cmds)
	}
	s = modelsReady(t, "claude-code", claudeList, event.Settings{Model: "claude-haiku-4-5-20251001", Choice: "claude-haiku-4-5-20251001", EffortKnown: true})
	if cmds := s.Do(ChooseEffort{Level: "low"}); cmds != nil {
		t.Errorf("a level asked for on Haiku 4.5: %+v", cmds)
	}
	locked := fableMax
	locked.EffortLock = "CLAUDE_CODE_EFFORT_LEVEL"
	s = modelsReady(t, "claude-code", claudeList, locked)
	if cmds := s.Do(ChooseEffort{Level: "low"}); cmds != nil {
		t.Errorf("an effort the variable holds asked for: %+v", cmds)
	}
	if n := notices(s); len(n) != 1 || n[0] != "the effort is set by CLAUDE_CODE_EFFORT_LEVEL (Max): the dashboard does not change it" {
		t.Errorf("notices %q", n)
	}
	if v := s.Metrics["model.effort"]; v.Text != "Max, held by CLAUDE_CODE_EFFORT_LEVEL" || v.Documented {
		t.Errorf("the details view's effort %+v", v)
	}
}

// TestUltracodeChoices is criteria 10 and 11: Ultracode is asked for alone,
// only while the CLI reports it available; otherwise zone B says why.
func TestUltracodeChoices(t *testing.T) {
	s := modelsReady(t, "claude-code", claudeList, fableMax)
	cmds := s.Do(ChooseUltracode{On: false})
	if len(cmds) != 1 || cmds[0].Change.Ultracode == nil || *cmds[0].Change.Ultracode || cmds[0].Change.Effort != "" || cmds[0].Change.Choice != "" {
		t.Errorf("Ultracode off: %+v", cmds)
	}
	s = modelsReady(t, "claude-code", claudeList, event.Settings{Model: "claude-opus-4-6", Choice: "claude-opus-4-6", Effort: "high", EffortKnown: true,
		UltracodeKnown: true})
	if cmds := s.Do(ChooseUltracode{On: true}); cmds != nil {
		t.Errorf("Ultracode asked for while unavailable: %+v", cmds)
	}
	if n := notices(s); len(n) != 1 || n[0] != "Claude Code reports Ultracode unavailable with Opus 4.6" {
		t.Errorf("notices %q", n)
	}
	s = modelsReady(t, "claude-code", claudeList, event.Settings{Model: "claude-fable-5-1", Choice: "fable"})
	s.Do(ChooseUltracode{On: true})
	if n := notices(s); len(n) != 1 || n[0] != "Ultracode's state is not known" {
		t.Errorf("notices %q", n)
	}
}

// TestCodexNextTurn is criterion 9 in the state: a Codex choice waits for
// the next message, shown on the editor row; a new model takes the effort
// in force when it lists it, else its default; choosing what is in force
// cancels the wait; the answer puts the change in force.
func TestCodexNextTurn(t *testing.T) {
	s := modelsReady(t, "codex", codexList, event.Settings{Model: "gpt-6-sol", Choice: "gpt-6-sol", Effort: "ultra", EffortKnown: true, Source: "thread/start"})
	cmds := s.Do(ChooseModel{ID: "gpt-6-luna", NextTurn: true})
	if len(cmds) != 1 || cmds[0].Change != (ModelChange{Choice: "gpt-6-luna", Effort: "medium", NextTurn: true}) {
		t.Fatalf("commands %+v", cmds)
	}
	if got := s.PendingLabel(); got != "→ GPT-6-Luna Medium (next turn)" {
		t.Errorf("pending %q", got)
	}
	cmds = s.Do(ChooseEffort{Level: "high", NextTurn: true})
	if len(cmds) != 1 || cmds[0].Change != (ModelChange{Choice: "gpt-6-luna", Effort: "high", NextTurn: true}) {
		t.Errorf("the effort added: %+v", cmds)
	}
	if cmds := s.Do(ChooseEffort{Level: "ultra", NextTurn: true}); cmds != nil {
		t.Errorf("ultra asked for on GPT-6-Luna: %+v", cmds)
	}
	s.Do(ChooseModel{ID: "gpt-6.1-sol", NextTurn: true})
	if got := s.PendingLabel(); got != "→ GPT-6.1-Sol High (next turn)" {
		t.Errorf("pending %q", got)
	}
	s.Do(ChooseModel{ID: "gpt-6-sol", NextTurn: true})
	s.Do(ChooseEffort{Level: "ultra", NextTurn: true})
	if s.Models.Asked != nil || s.PendingLabel() != "" {
		t.Errorf("choosing what is in force left %+v", s.Models.Asked)
	}
	s.Do(ChooseModel{ID: "gpt-6.1-sol", NextTurn: true})
	settle(s, event.Settings{Model: "gpt-6.1-sol", Choice: "gpt-6.1-sol", Effort: "ultra", EffortKnown: true, Source: "thread/read", Answers: true})
	if s.Models.Asked != nil || s.Models.Current().ID != "gpt-6.1-sol" || s.Session.Model != "gpt-6.1-sol" || len(notices(s)) != 0 {
		t.Errorf("after the answer: asked %+v, current %+v, notices %q", s.Models.Asked, s.Models.Current(), notices(s))
	}
}

// TestModelsNewProcess is design D10: a new process lists its models and
// reports its settings itself; nothing of the earlier one stays.
func TestModelsNewProcess(t *testing.T) {
	s := modelsReady(t, "claude-code", claudeList, fableMax)
	s.Session.Phase = Spawning
	s.Apply(event.Event{Seq: 9, Kind: event.SessionSpawned, Origin: event.Observed, Payload: event.Spawned{}})
	if len(s.Models.Entries) != 0 || s.Models.InForce != (event.Settings{}) {
		t.Errorf("after a new process: %+v", s.Models)
	}
}

func TestEffortLabel(t *testing.T) {
	for in, want := range map[string]string{"low": "Low", "medium": "Medium", "high": "High", "xhigh": "XHigh", "max": "Max", "ultra": "Ultra", "": ""} {
		if got := EffortLabel(in); got != want {
			t.Errorf("EffortLabel(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestTurnRefused is plan 34's review: Codex refusing the turn/start that
// carried a model change starts no turn, so the message goes back to the
// editor, the turn idle and sendable again, and zone B says the change was
// not taken.
func TestTurnRefused(t *testing.T) {
	s := modelsReady(t, "codex", codexList, event.Settings{Model: "gpt-6-sol", Choice: "gpt-6-sol", Effort: "high", EffortKnown: true, Source: "thread/start"})
	s.Do(ChooseModel{ID: "gpt-6.1-sol", NextTurn: true})
	if cmds := s.Do(Send{Text: "hello"}); len(cmds) != 1 || cmds[0].Kind != CmdWrite || s.Turn.Phase != Submitted {
		t.Fatalf("send: %+v, turn %q", cmds, s.Turn.Phase)
	}
	at := time.Date(2026, 10, 23, 12, 0, 2, 0, time.UTC)
	const why = "turn/start refused by Codex: invalid thread settings override"
	cmds := s.Apply(event.Event{Seq: 20, At: at, Kind: event.Error, Origin: event.Observed, Provider: "codex",
		Payload: event.Problem{Message: why, TurnRefused: true}})
	if len(cmds) != 1 || cmds[0].Kind != CmdRestoreDraft || cmds[0].Text != "hello" {
		t.Errorf("commands %+v, want the draft given back", cmds)
	}
	if s.Turn.Phase != Idle || s.Turn.Notice != "the message could not be sent: "+why {
		t.Errorf("turn %q, notice %q", s.Turn.Phase, s.Turn.Notice)
	}
	s.Apply(event.Event{Seq: 21, At: at, Kind: event.ModelSettings, Origin: event.Observed, Provider: "codex",
		Payload: event.Settings{Source: "turn/start", Answers: true, Refused: true, Text: "the model change was not taken: " + why}})
	if s.Models.Asked != nil || s.PendingLabel() != "" || s.Models.InForce.Model != "gpt-6-sol" {
		t.Errorf("after the refusal: asked %+v, in force %+v", s.Models.Asked, s.Models.InForce)
	}
	want := []string{"the message was not sent: " + why + "; it is back in the editor", "the model change was not taken: " + why}
	if n := notices(s); !reflect.DeepEqual(n, want) {
		t.Errorf("notices %q, want %q", n, want)
	}
	if cmds := s.Do(Send{Text: "hello"}); len(cmds) != 1 || cmds[0].Kind != CmdWrite {
		t.Errorf("sending again: %+v, notice %q", cmds, s.Turn.Notice)
	}
}

// TestChoiceDuringTurn is plan 34's review: a Claude Code change, applied at
// once, waits for the turn's end, zone B saying so; a Codex one, which the
// next message carries, is taken.
func TestChoiceDuringTurn(t *testing.T) {
	s := modelsReady(t, "claude-code", claudeList, fableMax)
	s.Do(Send{Text: "hello"})
	for _, a := range []Action{ChooseModel{ID: "claude-opus-4-6"}, ChooseEffort{Level: "low"}, ChooseUltracode{On: false}} {
		if cmds := s.Do(a); cmds != nil {
			t.Errorf("%T during a turn: %+v", a, cmds)
		}
	}
	// One notice for the three: the same words again are not repeated.
	if n := notices(s); len(n) != 1 || n[0] != "Claude Code's model settings change between turns: wait for the turn to end" {
		t.Errorf("notices %q", n)
	}
	c := modelsReady(t, "codex", codexList, event.Settings{Model: "gpt-6-sol", Choice: "gpt-6-sol", Effort: "high", EffortKnown: true})
	c.Do(Send{Text: "hello"})
	if cmds := c.Do(ChooseModel{ID: "gpt-6-luna", NextTurn: true}); len(cmds) != 1 || c.PendingLabel() != "→ GPT-6-Luna High (next turn)" {
		t.Errorf("Codex during a turn: %+v, %q", cmds, c.PendingLabel())
	}
}

// TestChangeDroppedAtEnd is plan 34's review: a change still waiting when
// the session ends is dropped, its label gone, zone B saying so.
func TestChangeDroppedAtEnd(t *testing.T) {
	s := modelsReady(t, "codex", codexList, event.Settings{Model: "gpt-6-sol", Choice: "gpt-6-sol", Effort: "high", EffortKnown: true})
	s.Do(ChooseModel{ID: "gpt-6-luna", NextTurn: true})
	s.Apply(event.Event{Seq: 20, At: time.Date(2026, 10, 23, 12, 0, 2, 0, time.UTC), Kind: event.SessionEnded, Origin: event.Observed, Provider: "codex",
		Payload: event.Ended{Code: 1, HasCode: true}})
	if s.Session.Phase != Crashed || s.Models.Asked != nil || s.PendingLabel() != "" {
		t.Errorf("after the crash: phase %q, asked %+v, label %q", s.Session.Phase, s.Models.Asked, s.PendingLabel())
	}
	if n := notices(s); len(n) != 1 || n[0] != "the model change asked for was not applied: the session ended" {
		t.Errorf("notices %q", n)
	}
}
