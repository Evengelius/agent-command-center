package claude

import (
	"encoding/json"
	"strings"

	"github.com/Evengelius/agent-command-center/internal/event"
)

// The session's models and their settings (plan 34; doc/research/
// claude-code.md, section 13): the models come from the initialize
// response (the Agent SDK's documented ModelInfo); the settings in force
// from get_settings, which is in the CLI's own schema and not in its
// documentation (undocumented), read after the handshake, after each change
// and after each turn. set_model and apply_flag_settings, both documented,
// change them for the session only.

// EffortVar is the variable that outranks every other effort setting
// (model-config, "Adjust effort level"): while it is set, the CLI keeps the
// effort it gives (P34c), and the dashboard shows that effort and does not
// change it.
const EffortVar = "CLAUDE_CODE_EFFORT_LEVEL"

// modelInfo is one entry of the initialize response's models.
type modelInfo struct {
	Value          string   `json:"value"`
	Resolved       string   `json:"resolvedModel"`
	Name           string   `json:"displayName"`
	Description    string   `json:"description"`
	SupportsEffort bool     `json:"supportsEffort"`
	Levels         []string `json:"supportedEffortLevels"`
}

// catalogue reads the initialize response's models, in the CLI's order; nil
// when it has none.
func catalogue(resp json.RawMessage) []event.ModelEntry {
	var r struct {
		Models []modelInfo `json:"models"`
	}
	if json.Unmarshal(resp, &r) != nil {
		return nil
	}
	var out []event.ModelEntry
	for _, mi := range r.Models {
		if mi.Value == "" {
			continue
		}
		e := event.ModelEntry{ID: mi.Value, Model: or(mi.Resolved, mi.Value), Name: or(mi.Name, mi.Value), Description: mi.Description}
		if mi.SupportsEffort {
			e.Efforts = append([]string(nil), mi.Levels...)
		}
		out = append(out, e)
	}
	return out
}

func or(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

// applied is the part of get_settings' answer the dashboard reads: the
// values in force, the settings' model and the one variable of their env
// that holds the effort. The rest of the owner's settings is never kept.
type applied struct {
	Applied *struct {
		Model     string  `json:"model"`
		Effort    *string `json:"effort"`
		Ultracode *bool   `json:"ultracode"`
		Available *bool   `json:"ultracodeAvailable"`
	} `json:"applied"`
	Effective struct {
		Model string `json:"model"`
		Env   struct {
			Effort json.RawMessage `json:"CLAUDE_CODE_EFFORT_LEVEL"`
		} `json:"env"`
	} `json:"effective"`
}

// settingsRead reads a get_settings answer: the settings in force, with the
// catalogue entry that selects the model and what holds the effort; and the
// answer as the event keeps it, applied and the settings' model only.
func (m *Mapper) settingsRead(requestID string, resp json.RawMessage) (event.Settings, []byte) {
	var a applied
	_ = json.Unmarshal(resp, &a)
	s := event.Settings{Source: "get_settings", Undocumented: true}
	if a.Applied == nil {
		s.Text = "get_settings answered without the settings in force"
		return s, kept(requestID, nil, "")
	}
	s.Model = a.Applied.Model
	s.Choice = m.choiceOf(s.Model, a.Effective.Model)
	s.EffortKnown = true
	if a.Applied.Effort != nil {
		s.Effort = *a.Applied.Effort
	}
	var inSettings string
	_ = json.Unmarshal(a.Effective.Env.Effort, &inSettings)
	if strings.TrimSpace(m.effortVar) != "" || strings.TrimSpace(inSettings) != "" {
		s.EffortLock = EffortVar
	}
	if a.Applied.Ultracode != nil {
		s.UltracodeKnown = true
		s.Ultracode = *a.Applied.Ultracode
		s.UltracodeAvailable = a.Applied.Available != nil && *a.Applied.Available
	}
	m.inForce = s
	return s, kept(requestID, a.Applied, a.Effective.Model)
}

// kept is a get_settings answer as its event keeps it: its applied values
// and the settings' model, nothing else of the owner's settings.
func kept(requestID string, values any, model string) []byte {
	r := map[string]any{}
	if values != nil {
		r["applied"] = values
	}
	if model != "" {
		r["effective"] = map[string]any{"model": model}
	}
	b, _ := json.Marshal(map[string]any{"type": "control_response", "response": map[string]any{
		"subtype": "success", "request_id": requestID, "response": r}})
	return b
}

// choiceOf is the catalogue entry that selects the model in force: the one
// set_model last took; else the settings' model; else Default when the
// settings name none; else the first other entry that runs it; "" when
// none does (a model started outside the catalogue).
func (m *Mapper) choiceOf(model, setting string) string {
	runs := func(id string) bool {
		for _, e := range m.catalogue {
			if e.ID == id {
				return e.Model == model
			}
		}
		return false
	}
	switch {
	case m.chosen != "" && runs(m.chosen):
		return m.chosen
	case setting != "" && runs(setting):
		return setting
	case setting == "" && runs("default"):
		return "default"
	}
	for _, e := range m.catalogue {
		if e.ID != "default" && e.Model == model {
			return e.ID
		}
	}
	if runs("default") {
		return "default"
	}
	return ""
}

// change notes a change the dashboard wrote: set_model's model, or the
// flag settings asked for.
func (m *Mapper) change(subtype string, request json.RawMessage) {
	m.changes++
	if subtype == "set_model" {
		var r struct {
			Model string `json:"model"`
		}
		_ = json.Unmarshal(request, &r)
		m.asked = r.Model
	}
}

// changeAnswered maps the answer to a change: a refusal is the change's
// answer, the settings unchanged; an acceptance waits for the read that
// follows it, which the session sends (ReadDue), to say what is in force.
func (m *Mapper) changeAnswered(what string, failed bool, msg string, raw []byte) []event.Event {
	if m.changes > 0 {
		m.changes--
	}
	// A read follows either way, to show what is in force; only an
	// acceptance's read answers the change.
	m.readDue, m.readAnswers = true, !failed
	if failed {
		return []event.Event{m.ev(event.ModelSettings, event.Observed, "", raw, event.Settings{
			Source: what, Answers: true, Refused: true, Text: what + " refused by Claude Code: " + msg})}
	}
	if what == "set_model" {
		m.chosen = m.asked
	}
	return []event.Event{m.ev(event.StatusChanged, event.Observed, "", raw, event.Status{Detail: what + " accepted"})}
}

// ReadDue says a change was answered and the read that says what is in
// force is to be written; writing it clears it (In).
func (m *Mapper) ReadDue() bool { return m.readDue }

// UltracodeInForce is Ultracode as the last read gave it, and whether it
// gave it: an effort change carries it, since a level asked for alone turns
// Ultracode off (P34c3).
func (m *Mapper) UltracodeInForce() (on, known bool) {
	return m.inForce.Ultracode, m.inForce.UltracodeKnown
}

// SetEffortVar records the effort variable in the CLI's environment.
func (m *Mapper) SetEffortVar(v string) { m.effortVar = v }
