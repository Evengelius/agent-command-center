package state

import (
	"strings"

	"github.com/Evengelius/agent-command-center/internal/event"
)

// The permission modes selector (plan 39, brief amendment 40): each CLI's
// own modes, never the other's, applied through the CLI and in force only
// once the CLI reports them.

// ModeOption is one option of the modes selector: the mode's word (the
// adapters', event.Mode*), its title and description as the owner's images
// write them, the description broken where the images break it, its icon,
// and whether it carries the warning colour.
type ModeOption struct {
	Mode    string
	Title   string
	Desc    []string
	Icon    string // hand, code, plan, bolt, shield, hexagon
	Warning bool
}

// The icons of the options (the owner's images).
const (
	IconHand    = "hand"
	IconCode    = "code"
	IconPlan    = "plan"
	IconBolt    = "bolt"
	IconShield  = "shield"
	IconHexagon = "hexagon"
)

// The modes beyond Manual and Auto (plan 39).
const (
	ModeAcceptEdits = event.ModeAcceptEdits
	ModePlan        = event.ModePlan
	ModeFull        = event.ModeFull
	ModeCustom      = event.ModeCustom
)

// claudeModes are Claude Code's four (P39c: 2.1.303 takes each of default,
// acceptEdits, plan and auto).
var claudeModes = []ModeOption{
	{ModeManual, "Manual", []string{"Claude will ask for approval", "before making each edit"}, IconHand, false},
	{ModeAcceptEdits, "Edit automatically", []string{"Claude will edit your selected", "text or the whole file"}, IconCode, false},
	{ModePlan, "Plan", []string{"Claude will explore the code and", "present a plan before editing"}, IconPlan, false},
	{ModeAuto, "Auto", []string{"Claude will approve actions that", "pass a safety check and pause", "for anything risky"}, IconBolt, false},
}

// codexModes are Codex's three (P39x: 0.163.0 takes each through
// thread/settings/update).
var codexModes = []ModeOption{
	{ModeManual, "Ask for approval", []string{"Always ask to edit external files", "and use the internet"}, IconHand, false},
	{ModeAuto, "Approve for me", []string{"Only ask for actions detected", "as potentially unsafe"}, IconShield, false},
	{ModeFull, "Full access", []string{"Unrestricted access to the internet", "and any file on your computer"}, IconHexagon, true},
}

// ModeOptions are the session's CLI's modes, in the selector's order; none
// for a CLI the dashboard has no modes of.
func (s *State) ModeOptions() []ModeOption {
	switch s.CLI() {
	case "codex":
		return codexModes
	case "claude-code":
		return claudeModes
	}
	return nil
}

// ModeOption is the session's CLI's option of the mode, nil when it has
// none.
func (s *State) ModeOption(mode string) *ModeOption {
	for i, o := range s.ModeOptions() {
		if o.Mode == mode {
			return &s.ModeOptions()[i]
		}
	}
	return nil
}

// ModeTitle is a mode as the session's CLI's selector names it: its
// option's title, "Custom" for a Codex combination outside its three, else
// the CLI's own word.
func (s *State) ModeTitle(mode string) string {
	if mode == "" {
		mode = ModeManual
	}
	if o := s.ModeOption(mode); o != nil {
		return o.Title
	}
	if mode == ModeCustom {
		return "Custom"
	}
	return mode
}

// chooseMode asks for one of the session's CLI's modes (plan 39, design
// D3): never another CLI's, nor one its policy leaves out (zone B's status
// says why), nor Codex's Full access unconfirmed (design D4). Choosing the
// mode in force asks nothing, or cancels a change waiting for Codex's next
// turn.
func (s *State) chooseMode(a ChooseMode) []Command {
	switch {
	case s.Session.Phase != Ready:
		s.Turn.Notice = "mode not changed: the session is " + strings.ToLower(string(s.Session.Phase))
		return nil
	case !a.Capable:
		s.Turn.Notice = "mode not changed: no modes here"
		return nil
	}
	o := s.ModeOption(a.Mode)
	if o == nil {
		return nil
	}
	if why := s.Session.ModesUnavailable[a.Mode]; why != "" {
		s.Turn.Notice = o.Title + " is not available: " + why
		return nil
	}
	if a.Mode == ModeFull && !a.Confirmed {
		return nil
	}
	cur := s.Mode()
	if a.Mode == cur && s.Session.ApprovalRequested == "" {
		return nil
	}
	s.Session.ApprovalRequested, s.Session.ModeNextTurn = a.Mode, a.NextTurn
	if a.Mode == cur {
		s.Session.ApprovalRequested = ""
	}
	s.Turn.Notice = ""
	return []Command{{Kind: CmdSetApprovals, Mode: a.Mode}}
}
