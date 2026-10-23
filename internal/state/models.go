package state

import (
	"strings"
	"time"

	"github.com/Evengelius/agent-command-center/internal/event"
)

// Models are the session's models and their settings (plan 34, brief
// amendment 34): the models the CLI listed, the settings in force as the
// CLI reports them, and a change asked for and not yet answered.
type Models struct {
	Entries []event.ModelEntry
	Source  string // the request the list came from
	// InForce merges the reports: each field as the last report that gave it.
	InForce event.Settings
	// Asked is a change asked for, until the CLI answers it; nil when none.
	Asked *ModelChange
	// Reason says why the settings in force are not known: a refused read.
	Reason string
	// ListProblem says why the list is missing or cut short, in the CLI's
	// words: a refused page, a cursor that came back.
	ListProblem string
}

// ModelChange is a change of the model settings: an entry's ID, an effort
// level, Ultracode on or off; an empty field asks for nothing. NextTurn
// marks a change the CLI takes with the next message (Codex).
type ModelChange struct {
	Choice    string
	Effort    string
	Ultracode *bool
	NextTurn  bool
}

// The actions of the model selector (plan 34). NextTurn is the session's
// ModelsNextTurn: the change goes with the next message.
type (
	ChooseModel struct {
		ID       string
		NextTurn bool
	}
	ChooseEffort struct {
		Level    string
		NextTurn bool
	}
	ChooseUltracode struct{ On bool }
	ModelsFailed    struct{ Message string } // the change could not be handed to the session
)

func (ChooseModel) action()     {}
func (ChooseEffort) action()    {}
func (ChooseUltracode) action() {}
func (ModelsFailed) action()    {}

// Entry is the listed model id selects, nil when none.
func (m *Models) Entry(id string) *event.ModelEntry {
	for i := range m.Entries {
		if m.Entries[i].ID == id {
			return &m.Entries[i]
		}
	}
	return nil
}

// Current is the listed model in force: the one the CLI's report names, else
// the first whose model it runs; nil when no listed model runs it.
func (m *Models) Current() *event.ModelEntry {
	if e := m.Entry(m.InForce.Choice); e != nil && (m.InForce.Model == "" || e.Model == m.InForce.Model) {
		return e
	}
	for i := range m.Entries {
		if m.InForce.Model != "" && m.Entries[i].Model == m.InForce.Model {
			return &m.Entries[i]
		}
	}
	return nil
}

// EffortLabel is how the dashboard writes an effort level: the levels the
// CLIs name in their own capitals (amendment 34's "XHigh"), any other word
// with its first letter upper.
func EffortLabel(level string) string {
	switch level {
	case "xhigh":
		return "XHigh"
	case "":
		return ""
	}
	return strings.ToUpper(level[:1]) + level[1:]
}

// models applies ModelsListed.
func (s *State) models(p event.Models) {
	s.Models.Entries, s.Models.Source, s.Models.ListProblem = p.Entries, p.Source, p.Problem
}

// dropModelChange forgets the change asked for when the session ends: no
// message will carry it and no read will answer it, so zone B says so.
func (s *State) dropModelChange(ev event.Event) {
	if s.Models.Asked == nil {
		return
	}
	s.Models.Asked = nil
	s.modelNotice(ev, "the model change asked for was not applied: the session ended")
}

// modelSettings applies ModelSettings: the settings in force, an answer to
// the change asked for (a refusal, or what the CLI kept of it, said in zone
// B), or a read that could not tell.
func (s *State) modelSettings(ev event.Event, p event.Settings) {
	if p.Refused {
		s.Models.Asked = nil
		if p.Text != "" {
			s.modelNotice(ev, p.Text)
		}
		return
	}
	in := &s.Models.InForce
	known := p.Model != "" || p.EffortKnown || p.UltracodeKnown
	if p.Model != "" {
		in.Model, in.Choice = p.Model, p.Choice
		if s.Session.Model != p.Model {
			// Zone B's speaker and zone A's model follow the model in force
			// (amendment 34, "the orchestrator's displayed model").
			s.Session.Model = p.Model
			s.Metrics["model"] = Value{Present: true, Text: p.Model, Origin: ev.Origin, Source: p.Source, Documented: !p.Undocumented, ReceivedAt: ev.At}
		}
	}
	// The details view lists each value with its read, an undocumented one
	// marked (plan 34, design D1).
	value := func(text, reason string) Value {
		return Value{Present: text != "", Text: text, Reason: reason, Origin: ev.Origin, Source: p.Source,
			Documented: !p.Undocumented, ReceivedAt: ev.At, ReasonAt: ev.At}
	}
	if p.EffortKnown {
		in.Effort, in.EffortKnown, in.EffortLock = p.Effort, true, p.EffortLock
		text := EffortLabel(p.Effort)
		if text != "" && p.EffortLock != "" {
			text += ", held by " + p.EffortLock
		}
		s.Metrics["model.effort"] = value(text, "the CLI sends no effort")
	}
	if p.UltracodeKnown {
		in.Ultracode, in.UltracodeAvailable, in.UltracodeKnown = p.Ultracode, p.UltracodeAvailable, true
		text := onOff(p.Ultracode)
		if !p.UltracodeAvailable {
			text += ", unavailable"
		}
		s.Metrics["model.ultracode"] = value(text, "")
	}
	if known {
		in.Source, in.Undocumented = p.Source, p.Undocumented
		s.Models.Reason = ""
	} else if p.Text != "" {
		s.Models.Reason = p.Text
	}
	if !p.Answers || s.Models.Asked == nil {
		return
	}
	asked := *s.Models.Asked
	s.Models.Asked = nil
	if !known {
		if p.Text != "" {
			s.modelNotice(ev, p.Text)
		}
		return
	}
	// What the CLI kept of the change, said in zone B (plan 34, design D3).
	var kept []string
	if asked.Choice != "" && in.Choice != asked.Choice {
		kept = append(kept, "the model "+s.modelName(in.Model)+" (asked: "+s.entryName(asked.Choice)+")")
	}
	if asked.Effort != "" && in.EffortKnown && in.Effort != asked.Effort {
		got := EffortLabel(in.Effort)
		if got == "" {
			got = "none sent"
		}
		kept = append(kept, "the effort "+got+" (asked: "+EffortLabel(asked.Effort)+")")
	}
	if asked.Ultracode != nil && in.UltracodeKnown && in.Ultracode != *asked.Ultracode {
		kept = append(kept, "Ultracode "+onOff(in.Ultracode))
	}
	if len(kept) > 0 {
		s.modelNotice(ev, s.cliName()+" kept "+strings.Join(kept, ", "))
	}
}

// modelNotice is a notice of the model selector in zone B; the same notice
// again right after it, from a second click, is not repeated.
func (s *State) modelNotice(ev event.Event, text string) {
	list := s.messages("")
	if n := len(*list); n > 0 && (*list)[n-1].Role == "notice" && (*list)[n-1].Text == text {
		return
	}
	*list = append(*list, Entry{Seq: ev.Seq, Role: "notice", Text: text, Origin: ev.Origin, At: ev.At})
}

// chooseModel asks for a listed model. A Codex change takes, with it, an
// effort the new model lists: the one in force when it lists it, else its
// default (plan 34, design D4).
func (s *State) chooseModel(a ChooseModel) []Command {
	e := s.Models.Entry(a.ID)
	if e == nil || !s.modelsOpen(a.NextTurn) {
		return nil
	}
	c := s.change(a.NextTurn)
	c.Choice = e.ID
	if a.NextTurn {
		effort := c.Effort
		if effort == "" {
			effort = s.Models.InForce.Effort
		}
		if !contains(e.Efforts, effort) {
			effort = e.DefaultEffort
		}
		c.Effort = effort
	}
	return s.ask(c)
}

// chooseEffort asks for an effort level of the model in force or asked for;
// one the variable that outranks every setting holds is not asked for, and
// zone B says why (design D5).
func (s *State) chooseEffort(a ChooseEffort) []Command {
	if !s.modelsOpen(a.NextTurn) {
		return nil
	}
	in := s.Models.InForce
	if lock := in.EffortLock; lock != "" && !a.NextTurn {
		s.modelNotice(event.Event{At: s.lastAt()}, "the effort is set by "+lock+" ("+orNone(EffortLabel(in.Effort))+"): the dashboard does not change it")
		return nil
	}
	c := s.change(a.NextTurn)
	model := s.Models.Current()
	if c.Choice != "" {
		model = s.Models.Entry(c.Choice)
	}
	if model == nil || !contains(model.Efforts, a.Level) {
		return nil // only a level the model lists is asked for (design D5)
	}
	c.Effort = a.Level
	return s.ask(c)
}

// chooseUltracode turns Claude Code's Ultracode on or off, only while the
// CLI reports it available; otherwise zone B says why (design D6).
func (s *State) chooseUltracode(a ChooseUltracode) []Command {
	if !s.modelsOpen(false) {
		return nil
	}
	in := s.Models.InForce
	if !in.UltracodeKnown || !in.UltracodeAvailable {
		why := "Claude Code reports Ultracode unavailable with " + s.modelName(in.Model)
		if !in.UltracodeKnown {
			why = "Ultracode's state is not known"
			if s.Models.Reason != "" {
				why += ": " + s.Models.Reason
			}
		}
		s.modelNotice(event.Event{At: s.lastAt()}, why)
		return nil
	}
	on := a.On
	c := ModelChange{Ultracode: &on}
	return s.ask(c)
}

// modelsOpen says a change can be asked for now: a ready session, and no
// change of Claude Code's waiting for its answer. Claude Code's changes
// take effect at once, so they wait for the turn's end, which zone B says:
// what a change does to a running turn was not probed (plan 34, P34c sent
// no message).
func (s *State) modelsOpen(nextTurn bool) bool {
	if s.Session.Phase != Ready || len(s.Models.Entries) == 0 {
		return false
	}
	if nextTurn {
		return true
	}
	if s.Turn.Phase != Idle {
		s.modelNotice(event.Event{At: s.lastAt()}, "Claude Code's model settings change between turns: wait for the turn to end")
		return false
	}
	return s.Models.Asked == nil
}

// change is the change a choice adds to: Codex's waiting one, which the next
// message carries whole, or a new one.
func (s *State) change(nextTurn bool) ModelChange {
	if nextTurn && s.Models.Asked != nil {
		return *s.Models.Asked
	}
	return ModelChange{NextTurn: nextTurn}
}

// ask records the change and hands it to the session. A Codex change that
// asks for what is in force cancels the waiting one.
func (s *State) ask(c ModelChange) []Command {
	if c.NextTurn {
		in := s.Models.InForce
		if (c.Choice == "" || c.Choice == in.Choice) && (c.Effort == "" || c.Effort == in.Effort) {
			s.Models.Asked = nil
		} else {
			s.Models.Asked = &c
		}
	} else {
		s.Models.Asked = &c
	}
	return []Command{{Kind: CmdModels, Change: c}}
}

// modelsFailed: the session did not take the change.
func (s *State) modelsFailed(a ModelsFailed) {
	s.Models.Asked = nil
	s.modelNotice(event.Event{At: s.lastAt()}, "the model settings were not changed: "+a.Message)
}

// PendingLabel is a Codex change waiting for the next message as the editor
// row shows it: "→ <model> <Effort> (next turn)"; "" when none waits.
func (s *State) PendingLabel() string {
	a := s.Models.Asked
	if a == nil || !a.NextTurn {
		return ""
	}
	name := s.Models.InForce.Model
	if e := s.Models.Entry(a.Choice); e != nil {
		name = e.Name
	} else if cur := s.Models.Current(); cur != nil {
		name = cur.Name
	}
	effort := a.Effort
	if effort == "" {
		effort = s.Models.InForce.Effort
	}
	label := "→ " + name
	if effort != "" {
		label += " " + EffortLabel(effort)
	}
	return label + " (next turn)"
}

// modelName is a model id by the name the CLI's list gives the entry that
// runs it, else the id.
func (s *State) modelName(model string) string {
	for _, e := range s.Models.Entries {
		if e.Model == model && e.ID != "default" {
			return e.Name
		}
	}
	if model == "" {
		return "the model in force"
	}
	return model
}

// entryName is an entry's name, else its ID.
func (s *State) entryName(id string) string {
	if e := s.Models.Entry(id); e != nil {
		return e.Name
	}
	return id
}

// lastAt is the time of the last event applied: a notice an action adds
// takes it, the reducer reading no clock (architecture.md, The five parts).
func (s *State) lastAt() time.Time {
	if n := len(s.Log); n > 0 {
		return s.Log[n-1].At
	}
	return time.Time{}
}

// cliName is the session's CLI by name, for zone B's notices.
func (s *State) cliName() string {
	if s.CLI() == "codex" {
		return "Codex"
	}
	return "Claude Code"
}

func onOff(on bool) string {
	if on {
		return "on"
	}
	return "off"
}

func orNone(s string) string {
	if s == "" {
		return "none sent"
	}
	return s
}
