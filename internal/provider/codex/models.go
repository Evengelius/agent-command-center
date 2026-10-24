package codex

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/Evengelius/agent-command-center/internal/event"
)

// The session's models and their effort (plan 34; doc/research/codex-cli.md,
// section 13): the models from every page of model/list, hidden ones left
// out; the model and effort in force from the thread/start answer, then
// from thread/settings/updated, and from thread/read after a turn that
// carried a change. A change between turns is written at once with
// thread/settings/update, Codex's experimental API, which the session asks
// for at initialize as OpenAI's Codex extension for VS Code does (plan 38,
// design D1); it is answered by the thread/settings/updated that follows
// (P38x: 1 to 3 ms after the update's {}). A change during a turn waits
// for its end; with a Codex that refuses the update, every change goes with
// the next turn/start ("for this turn and subsequent turns", stable), as in
// plan 34 (design D4).

// listedModel is one model of a model/list page.
type listedModel struct {
	ID          string `json:"id"`
	Model       string `json:"model"`
	Name        string `json:"displayName"`
	Description string `json:"description"`
	Hidden      bool   `json:"hidden"`
	IsDefault   bool   `json:"isDefault"`
	Efforts     []struct {
		Effort string `json:"reasoningEffort"`
	} `json:"supportedReasoningEfforts"`
	DefaultEffort string `json:"defaultReasoningEffort"`
}

// modelPage reads one model/list answer: its visible models and the cursor
// of the next page, "" at the last.
func modelPage(result json.RawMessage) (models []event.ModelEntry, next string) {
	var r struct {
		Data       []listedModel `json:"data"`
		NextCursor *string       `json:"nextCursor"`
	}
	if json.Unmarshal(result, &r) != nil {
		return nil, ""
	}
	for _, d := range r.Data {
		if d.Hidden {
			continue // hidden from the default picker list (model/list's includeHidden)
		}
		e := event.ModelEntry{ID: first(d.ID, d.Model), Model: first(d.Model, d.ID), Name: first(d.Name, first(d.ID, d.Model)),
			Description: d.Description, DefaultEffort: d.DefaultEffort, Default: d.IsDefault}
		for _, ef := range d.Efforts {
			if ef.Effort != "" {
				e.Efforts = append(e.Efforts, ef.Effort)
			}
		}
		models = append(models, e)
	}
	if r.NextCursor != nil {
		next = *r.NextCursor
	}
	return models, next
}

// maxModelPages bounds a listing: a server whose cursors never end is not
// followed past it.
const maxModelPages = 100

// listPage maps a model/list answer: the page's models join the listing;
// the last page gives the whole list as the session's models. A cursor
// that came before, or a page past maxModelPages, ends the listing there,
// the models so far kept and the reason an error (plan 34).
func (m *Mapper) listPage(result json.RawMessage, raw []byte) []event.Event {
	models, next := modelPage(result)
	m.listing = append(m.listing, models...)
	if m.cursors == nil {
		m.cursors = map[string]bool{}
	}
	var problem string
	switch {
	case next != "" && m.cursors[next]:
		problem = "model/list: the page cursor " + strconv.Quote(next) + " came back; the list stops there"
	case next != "" && len(m.cursors) >= maxModelPages-1:
		problem = "model/list: more than " + strconv.Itoa(maxModelPages) + " pages; the list stops there"
	case next != "":
		m.cursors[next], m.nextPage = true, next
		return []event.Event{m.status("", raw, "", "model/list page")}
	}
	m.catalogue, m.listing, m.cursors, m.nextPage = m.listing, nil, nil, ""
	listed := m.ev(event.ModelsListed, event.Observed, "", "", "", raw, event.Models{Entries: m.catalogue, Source: "model/list", Problem: problem})
	if problem == "" {
		return []event.Event{listed}
	}
	return []event.Event{m.ev(event.Error, event.Observed, "", "", "", raw, event.Problem{Message: problem}), listed}
}

// NextPage is the cursor of the model/list page the session asks for next,
// "" when the listing ended; asking for it clears it.
func (m *Mapper) NextPage() string {
	next := m.nextPage
	m.nextPage = ""
	return next
}

// DefaultModel is the model the whole list marks as default, "" when none
// does or no list is complete.
func (m *Mapper) DefaultModel() string {
	for _, e := range m.catalogue {
		if e.Default {
			return e.Model
		}
	}
	return ""
}

// settingsOf is the model and effort a thread/start, thread/resume or
// thread/read answer gives (the effort known when the answer names one).
func settingsOf(model string, effort *string, source string) event.Settings {
	s := event.Settings{Model: model, Choice: model, Source: source}
	if effort != nil && *effort != "" {
		s.Effort, s.EffortKnown = *effort, true
	}
	return s
}

// threadRead maps a thread/read answer: the main thread's model and effort
// in force, the answer to a change when the read follows one.
func (m *Mapper) threadRead(id string, result json.RawMessage, raw []byte) []event.Event {
	var r struct {
		Thread struct {
			ID     string  `json:"id"`
			Model  string  `json:"model"`
			Effort *string `json:"reasoningEffort"`
		} `json:"thread"`
	}
	_ = json.Unmarshal(result, &r)
	// The read answers its change unless a thread/settings/updated did first
	// (plan 38).
	answers := m.answering[id] && m.outstanding
	delete(m.answering, id)
	if r.Thread.ID != "" && m.main != "" && r.Thread.ID != m.main {
		return []event.Event{m.status("", raw, "", "thread/read of another thread")}
	}
	s := settingsOf(r.Thread.Model, r.Thread.Effort, "thread/read")
	if answers {
		s.Answers, m.outstanding = true, false
	}
	return []event.Event{m.ev(event.ModelSettings, event.Observed, "", "", "", raw, s)}
}

// settingsUpdated maps thread/settings/updated (Codex's experimental API,
// plan 38): the main thread's model and effort in force, nothing else of
// its settings, the answer to the change written when one waits (design
// D2). A turn/start that changes another setting brings one too (P38t:
// its summary).
func (m *Mapper) settingsUpdated(params json.RawMessage, raw []byte) []event.Event {
	var p struct {
		ThreadID string `json:"threadId"`
		Settings struct {
			Model    string          `json:"model"`
			Effort   *string         `json:"effort"`
			Policy   json.RawMessage `json:"approvalPolicy"`
			Reviewer string          `json:"approvalsReviewer"`
			Sandbox  json.RawMessage `json:"sandboxPolicy"`
		} `json:"threadSettings"`
	}
	_ = json.Unmarshal(params, &p)
	if p.ThreadID != "" && m.main != "" && p.ThreadID != m.main {
		return []event.Event{m.status(m.agentOf(p.ThreadID), raw, "", "thread/settings/updated of another thread")}
	}
	s := settingsOf(p.Settings.Model, p.Settings.Effort, "thread/settings/updated")
	s.Answers, m.outstanding = m.outstanding, false
	out := []event.Event{m.ev(event.ModelSettings, event.Observed, "", "", "", raw, s)}
	// The mode in force, by its three settings, the answer to a mode written
	// when one waits (plan 39, design D2).
	sandbox := namedValue(p.Settings.Sandbox)
	if mode := codexMode(sandbox, policyValue(p.Settings.Policy), p.Settings.Reviewer); mode != "" {
		out = append(out, m.ev(event.ApprovalMode, event.Observed, "", "", "", raw, event.Mode{Mode: mode, Sandbox: sandbox, Answers: m.modeOutstanding}))
		m.modeOutstanding = false
	}
	return out
}

// policyValue is an approval policy as given, a word; an object (Codex's
// granular policy) is no preset's, read as Custom.
func policyValue(raw json.RawMessage) string {
	var v string
	if len(raw) == 0 || string(raw) == "null" {
		return ""
	}
	if json.Unmarshal(raw, &v) == nil {
		return v
	}
	return "granular"
}

// modeLimits reads configRequirements/read's answer (plan 39, design D1;
// P39x): a mode whose sandbox or approval policy the managed requirements
// leave out is unavailable, with why; none when no requirement is set.
func (m *Mapper) modeLimits(result json.RawMessage, raw []byte) event.Event {
	var r struct {
		Requirements *struct {
			Sandboxes []json.RawMessage `json:"allowedSandboxModes"`
			Policies  []json.RawMessage `json:"allowedApprovalPolicies"`
		} `json:"requirements"`
	}
	_ = json.Unmarshal(result, &r)
	limits := event.ModeLimits{Source: "configRequirements/read"}
	allowed := func(list []json.RawMessage, v string) bool {
		for _, x := range list {
			if policyValue(x) == v {
				return true
			}
		}
		return false
	}
	if q := r.Requirements; q != nil {
		for mode, p := range modePresets {
			var why []string
			if sb := kebab(p.sandbox); q.Sandboxes != nil && !allowed(q.Sandboxes, sb) {
				why = append(why, "the sandbox "+sb)
			}
			if q.Policies != nil && !allowed(q.Policies, p.policy) {
				why = append(why, "the approval policy "+p.policy)
			}
			if len(why) > 0 {
				if limits.Unavailable == nil {
					limits.Unavailable = map[string]string{}
				}
				limits.Unavailable[mode] = "Codex's requirements do not allow " + strings.Join(why, " or ")
			}
		}
	}
	return m.ev(event.ModesAllowed, event.Observed, "", "", "", raw, limits)
}

// kebab writes a sandbox type as the requirements name it: readOnly is
// read-only.
func kebab(v string) string {
	var b strings.Builder
	for _, r := range v {
		if r >= 'A' && r <= 'Z' {
			b.WriteByte('-')
			r += 'a' - 'A'
		}
		b.WriteRune(r)
	}
	return b.String()
}

// unsupported says Codex refused method as one it does not take: VS Code's
// rule (JSON-RPC -32601, "method not found", or "unknown method" or
// "unknown variant" naming the method; openai.chatgpt 26.1021.22027), or
// Codex 0.163.0's -32600 "... requires experimentalApi capability" when
// the opt-in is not taken (P34x). Plan 38, design D4.
func unsupported(raw json.RawMessage, method string) bool {
	var e struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	_ = json.Unmarshal(raw, &e)
	msg := strings.ToLower(e.Message)
	named := strings.Contains(msg, strings.ToLower(method))
	return e.Code == -32601 || strings.Contains(msg, "method not found") ||
		named && (strings.Contains(msg, "unknown method") || strings.Contains(msg, "unknown variant") || strings.Contains(msg, "requires experimentalapi"))
}

// TakeReadDue says a turn that carried a change was answered, or a turn
// was rerouted: the session writes thread/read, which the mapper takes as
// the change's answer in the first case (In).
func (m *Mapper) TakeReadDue() bool {
	due := m.readDue
	if due {
		m.readDue, m.readSent, m.readForChange = false, m.readForChange, false
	}
	return due
}
