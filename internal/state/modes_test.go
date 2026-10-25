package state_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/Evengelius/agent-command-center/internal/event"
	"github.com/Evengelius/agent-command-center/internal/fixture"
	"github.com/Evengelius/agent-command-center/internal/provider/claude"
	"github.com/Evengelius/agent-command-center/internal/provider/codex"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// The modes selector's state (plan 39, brief amendment 40).

// onCLI is a ready session of cli, not a mock, in mode.
func onCLI(t *testing.T, cli, mode string) *state.State {
	t.Helper()
	s := start(t, event.Started{SessionID: "s", ApprovalMode: mode})
	s.Context.Mock, s.Context.Provider = false, cli
	return s
}

// TestModeCatalogs is criterion 39.2: each CLI's modes, never the other's,
// in the images' order, the owner's titles and descriptions word for word,
// broken where the images break them; Full access alone in the warning
// colour; no modes for a session of neither CLI.
func TestModeCatalogs(t *testing.T) {
	type opt struct {
		mode, title, desc, icon string
		warning                 bool
	}
	for cli, want := range map[string][]opt{
		"claude-code": {
			{state.ModeManual, "Manual", "Claude will ask for approval|before making each edit", state.IconHand, false},
			{state.ModeAcceptEdits, "Edit automatically", "Claude will edit your selected|text or the whole file", state.IconCode, false},
			{state.ModePlan, "Plan", "Claude will explore the code and|present a plan before editing", state.IconPlan, false},
			{state.ModeAuto, "Auto", "Claude will approve actions that|pass a safety check and pause|for anything risky", state.IconBolt, false},
		},
		"codex": {
			{state.ModeManual, "Ask for approval", "Always ask to edit external files|and use the internet", state.IconHand, false},
			{state.ModeAuto, "Approve for me", "Only ask for actions detected|as potentially unsafe", state.IconShield, false},
			{state.ModeFull, "Full access", "Unrestricted access to the internet|and any file on your computer", state.IconHexagon, true},
		},
	} {
		s := onCLI(t, cli, event.ModeManual)
		got := s.ModeOptions()
		if len(got) != len(want) {
			t.Fatalf("%s: %d options, want %d", cli, len(got), len(want))
		}
		for i, w := range want {
			g := got[i]
			if g.Mode != w.mode || g.Title != w.title || strings.Join(g.Desc, "|") != w.desc || g.Icon != w.icon || g.Warning != w.warning {
				t.Errorf("%s option %d: %+v, want %+v", cli, i, g, w)
			}
			if s.ModeTitle(w.mode) != w.title {
				t.Errorf("%s: title of %q is %q", cli, w.mode, s.ModeTitle(w.mode))
			}
		}
	}
	for _, cli := range []string{"", "gemini"} {
		if s := onCLI(t, cli, ""); len(s.ModeOptions()) != 0 {
			t.Errorf("%q: options %+v", cli, s.ModeOptions())
		}
	}
	// A mode the catalog has not: named as Codex's Custom or the CLI's word.
	if s := onCLI(t, "codex", ""); s.ModeTitle(state.ModeCustom) != "Custom" || s.ModeTitle("") != "Ask for approval" {
		t.Errorf("Codex: %q, %q", s.ModeTitle(state.ModeCustom), s.ModeTitle(""))
	}
	if s := onCLI(t, "claude-code", ""); s.ModeTitle("bypassPermissions") != "bypassPermissions" || s.ModeOption("bypassPermissions") != nil {
		t.Errorf("Claude Code: %q", s.ModeTitle("bypassPermissions"))
	}
}

// modesThrough replays a probe into a state and gives the mode in force
// after each mode event, with the state at the end.
func modesThrough(t *testing.T, name, provider string, m fixture.Mapper) ([]string, *state.State) {
	t.Helper()
	s := state.New()
	var modes []string
	for _, e := range replayEvents(t, name, provider, m) {
		s.Apply(e)
		if e.Kind == event.ApprovalMode || e.Kind == event.SessionStarted {
			modes = append(modes, s.Mode())
		}
	}
	return modes, s
}

// TestModesP39c is criterion 39.3 on Claude Code 2.1.303 (P39c, which sent
// no prompt, so no system/init; plan 12's tests read the init's): the mode
// in force from its answers to set_permission_mode and its system/status,
// Manual, Edit automatically, Plan, Auto, a refused word leaving Auto, and
// Manual again; the refusal in zone B in the CLI's words.
func TestModesP39c(t *testing.T) {
	modes, s := modesThrough(t, "p39c-claude-permission-modes.jsonl", claude.ProviderID, claudeMapper())
	// Each change is answered, then reported by system/status (but the
	// first, the mode already in force): the mode after each event.
	want := []string{state.ModeManual, state.ModeAcceptEdits, state.ModeAcceptEdits, state.ModePlan, state.ModePlan,
		state.ModeAuto, state.ModeAuto, state.ModeAuto, state.ModeManual, state.ModeManual}
	if !slices.Equal(modes, want) {
		t.Errorf("modes %v\nwant %v", modes, want)
	}
	refused := false
	for _, e := range s.Conversation {
		refused = refused || e.Role == "notice" && strings.Contains(e.Text, "must be one of acceptEdits, auto, bypassPermissions, default, dontAsk, plan")
	}
	if !refused || s.Mode() != state.ModeManual {
		t.Errorf("refusal shown %v, mode %q at the end", refused, s.Mode())
	}
}

// TestModesP39x is criterion 39.3 on Codex 0.163.0 (P39x): the thread's start
// Ask for approval; VS Code's workspace-write "Ask for approval", which is
// none of the dashboard's three, Custom; Approve for me; Full access; Ask
// for approval again; a change of nothing reports nothing. The requirements
// leave no mode out.
func TestModesP39x(t *testing.T) {
	modes, s := modesThrough(t, "p39x-codex-permission-modes.jsonl", codex.ProviderID, codexMapper())
	want := []string{state.ModeManual, state.ModeCustom, state.ModeAuto, state.ModeFull, state.ModeManual}
	if !slices.Equal(modes, want) {
		t.Errorf("modes %v\nwant %v", modes, want)
	}
	if s.Session.Sandbox != "readOnly" || len(s.Session.ModesUnavailable) != 0 || s.Session.ApprovalRequested != "" {
		t.Errorf("sandbox %q, modes left out %v, requested %q", s.Session.Sandbox, s.Session.ModesUnavailable, s.Session.ApprovalRequested)
	}
}

// TestChooseMode is criteria 39.3 to 39.5 in the state: a choice asks for
// one of the session's CLI's modes and changes nothing shown until the CLI
// reports it; never another CLI's, nor Full access unconfirmed, nor a mode
// the requirements leave out (zone B saying why); a refusal leaves the
// mode, a change Codex keeps for the next message stays asked for.
func TestChooseMode(t *testing.T) {
	s := onCLI(t, "claude-code", state.ModeManual)
	for _, m := range []string{state.ModeAcceptEdits, state.ModePlan, state.ModeAuto} {
		cmds := s.Do(state.ChooseMode{Mode: m, Capable: true})
		if len(cmds) != 1 || cmds[0].Kind != state.CmdSetApprovals || cmds[0].Mode != m || s.Session.ApprovalRequested != m || s.Mode() != state.ModeManual {
			t.Fatalf("%s: %+v, requested %q, mode %q", m, cmds, s.Session.ApprovalRequested, s.Mode())
		}
		s.Do(state.ChooseMode{Mode: state.ModeManual, Capable: true}) // back: the request cancelled
	}
	if cmds := s.Do(state.ChooseMode{Mode: state.ModeFull, Confirmed: true, Capable: true}); cmds != nil || s.Session.ApprovalRequested != "" {
		t.Errorf("Claude Code took Codex's Full access: %+v", cmds)
	}

	x := onCLI(t, "codex", state.ModeManual)
	for _, m := range []string{state.ModeAcceptEdits, state.ModePlan, state.ModeCustom} {
		if cmds := x.Do(state.ChooseMode{Mode: m, Confirmed: true, Capable: true}); cmds != nil {
			t.Errorf("Codex took %q: %+v", m, cmds)
		}
	}
	if cmds := x.Do(state.ChooseMode{Mode: state.ModeFull, Capable: true}); cmds != nil || x.Session.ApprovalRequested != "" {
		t.Fatalf("Full access unconfirmed: %+v, requested %q", cmds, x.Session.ApprovalRequested)
	}
	cmds := x.Do(state.ChooseMode{Mode: state.ModeFull, Confirmed: true, Capable: true})
	if len(cmds) != 1 || cmds[0].Mode != state.ModeFull || x.Session.ApprovalRequested != state.ModeFull || x.Session.ModeNextTurn || x.Mode() != state.ModeManual {
		t.Fatalf("Full access confirmed: %+v, requested %q, next turn %v, mode %q", cmds, x.Session.ApprovalRequested, x.Session.ModeNextTurn, x.Mode())
	}
	// Codex refuses: the mode stays, the request goes, zone B says why.
	apply(x, 10, event.ApprovalMode, event.Mode{Refused: true, Text: "the mode was not changed: boom"})
	if x.Mode() != state.ModeManual || x.Session.ApprovalRequested != "" || x.Conversation[len(x.Conversation)-1].Text != "the mode was not changed: boom" {
		t.Errorf("refused: mode %q, requested %q", x.Mode(), x.Session.ApprovalRequested)
	}
	// Codex takes it with the next message: still asked for, so marked.
	x.Do(state.ChooseMode{Mode: state.ModeAuto, Capable: true})
	apply(x, 11, event.ApprovalMode, event.Mode{NextTurn: true, Text: "Codex takes mode changes with the next message: unsupported"})
	if x.Mode() != state.ModeManual || x.Session.ApprovalRequested != state.ModeAuto || !x.Session.ModeNextTurn {
		t.Errorf("next turn: mode %q, requested %q, next turn %v", x.Mode(), x.Session.ApprovalRequested, x.Session.ModeNextTurn)
	}
	apply(x, 12, event.ApprovalMode, event.Mode{Mode: state.ModeAuto, Sandbox: "workspaceWrite"})
	if x.Mode() != state.ModeAuto || x.Session.ApprovalRequested != "" || x.Session.Sandbox != "workspaceWrite" {
		t.Errorf("in force: mode %q, requested %q, sandbox %q", x.Mode(), x.Session.ApprovalRequested, x.Session.Sandbox)
	}
	// The requirements leave Full access out: greyed, no choice, zone B says why.
	apply(x, 13, event.ModesAllowed, event.ModeLimits{Unavailable: map[string]string{state.ModeFull: "Codex's requirements do not allow the approval policy never"}})
	if cmds := x.Do(state.ChooseMode{Mode: state.ModeFull, Confirmed: true, Capable: true}); cmds != nil ||
		x.Turn.Notice != "Full access is not available: Codex's requirements do not allow the approval policy never" {
		t.Errorf("left out: %+v, notice %q", cmds, x.Turn.Notice)
	}
}
