package app

import (
	"slices"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/Evengelius/agent-command-center/internal/input"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// The ping (plan 63, brief amendment 65; ui-design.md, The ping): when the
// agent needs the owner while the dashboard's window, or its tab, lacks the
// focus, a BEL between frames in ACC's own Windows Terminal profile, whose
// bellStyle "taskbar" flashes the taskbar button, a Windows notification
// where the dashboard has a notifier (plan 64, notify.go), and the reason in
// the window's title. Nothing is drawn in the frame.

// windowTitle is the window's title while nothing waits.
const windowTitle = "Agent Command Center"

// pingEvery is the shortest time between two BELs (design D3).
const pingEvery = 5 * time.Second

// bel is the bell character: Terminal does with it what the profile's
// bellStyle says.
const bel = "\a"

// The needs, the most urgent first: the title names the first that waits
// (design D4).
const (
	needApproval = iota
	needQuestion
	needCrash
	needFailed
	needTurn
)

// needTitles are the window's titles, by need.
var needTitles = [...]string{
	needApproval: "◆ Approval waiting · " + windowTitle,
	needQuestion: "◆ Question waiting · " + windowTitle,
	needCrash:    "✗ Session ended · " + windowTitle,
	needFailed:   "✗ Turn failed · " + windowTitle,
	needTurn:     "✓ Turn done · " + windowTitle,
}

// need is one thing the agent needs the owner for, noticed while the window
// lacked the focus: an approval or a question, by its id, the session's
// end, a turn failed or completed, by its number (state.Stage.N). It is
// rung once its BEL is written, or once none is due: no bell here, or the
// focus back first.
type need struct {
	kind  int
	id    string
	stage int
	rung  bool
}

// pingState is the ping's part of the model: how far the approvals and the
// turns' ends were looked at (s.Queue and s.Stages only grow), whether the
// session's end was, the needs that wait, the last ring's time, whether a
// pingMsg is pending, and whether a notification is up (plan 64).
type pingState struct {
	queued, stages int
	crashed        bool
	needs          []need
	last           time.Time
	due            bool
	notified       bool
}

// pingMsg says the 5 s since the last BEL are up (design D3).
type pingMsg struct{}

// pings notices what needs the owner since the last message (design D1),
// drops what no longer waits, and gives the BEL when one is due. Needs
// noticed while the window has the focus are seen: nothing rings for them
// and the title does not name them.
func (m *Model) pings() []tea.Cmd {
	p, s := &m.ping, m.s
	for ; p.queued < len(s.Queue); p.queued++ {
		// An approval is pending from its request on (state.Approval).
		if a := s.Approvals[s.Queue[p.queued]]; a != nil && a.Status == state.Pending && m.blurred {
			kind := needApproval
			if a.Question {
				kind = needQuestion
			}
			p.needs = append(p.needs, need{kind: kind, id: a.ID})
		}
	}
	for ; p.stages < len(s.Stages) && !s.Stages[p.stages].Open; p.stages++ {
		// An interrupted turn is the owner's own doing.
		switch st := s.Stages[p.stages]; {
		case !m.blurred:
		case st.Outcome == "completed":
			p.needs = append(p.needs, need{kind: needTurn, stage: st.N})
		case st.Outcome == "failed":
			p.needs = append(p.needs, need{kind: needFailed, stage: st.N})
		}
	}
	if s.Session.Phase == state.Crashed && !p.crashed {
		p.crashed = true
		if m.blurred {
			p.needs = append(p.needs, need{kind: needCrash})
		}
	}
	p.needs = slices.DeleteFunc(p.needs, func(n need) bool { return !m.waits(n) })
	var cmds []tea.Cmd
	if len(p.needs) == 0 && p.notified {
		// Nothing waits: the notification leaves Windows' list as the title
		// clears (plan 64, design D4).
		if c := m.unnotify(); c != nil {
			cmds = append(cmds, c)
		}
	}
	return append(cmds, m.ring()...)
}

// waits says whether what a need is for still waits for the owner: an
// approval or a question until answered, a turn's end while no turn has
// started since (one the CLI opens itself for a background agent's report),
// the session's end until the focus is back (pingFocused).
func (m *Model) waits(n need) bool {
	switch n.kind {
	case needApproval, needQuestion:
		a := m.s.Approvals[n.id]
		return a != nil && a.Status == state.Pending
	case needFailed, needTurn:
		return n.stage == len(m.s.Stages) // Stage.N counts them from 1
	}
	return true
}

// pingFocused is the focus back: the session's end and a turn's leave the
// title, an approval or a question stays until answered, and nothing that
// waits rings any more (designs D3 and D4).
func (m *Model) pingFocused() {
	p := &m.ping
	p.needs = slices.DeleteFunc(p.needs, func(n need) bool { return n.kind != needApproval && n.kind != needQuestion })
	p.rungAll()
}

// ring rings once for the needs not yet rung while the window lacks the
// focus: a BEL in ACC's profile only (design D2), and a notification of the
// most urgent of them where the dashboard has a notifier, in that profile
// or not (plan 64, design D2). At once when the last ring is 5 s old or
// more, else when those 5 s are up, the needs noticed meanwhile sharing it
// (design D3).
func (m *Model) ring() []tea.Cmd {
	p := &m.ping
	if !m.blurred || !slices.ContainsFunc(p.needs, func(n need) bool { return !n.rung }) {
		return nil
	}
	bell, notify := m.bellHere(), m.opt.Notify != nil
	if !bell && !notify {
		p.rungAll() // the title alone says it
		return nil
	}
	if wait := p.last.Add(pingEvery).Sub(m.ui.Now); !p.last.IsZero() && wait > 0 {
		if !p.due {
			if c := m.opt.After(wait, pingMsg{}); c != nil {
				p.due = true
				return []tea.Cmd{c}
			}
		}
		return nil
	}
	kind := -1
	for _, n := range p.needs {
		if !n.rung && (kind < 0 || n.kind < kind) {
			kind = n.kind
		}
	}
	p.last = m.ui.Now
	p.rungAll()
	var cmds []tea.Cmd
	if bell {
		cmds = append(cmds, tea.Raw(bel))
	}
	if notify {
		cmds = append(cmds, m.notify(kind))
	}
	return cmds
}

// rungAll says no need that waits now rings any more.
func (p *pingState) rungAll() {
	for i := range p.needs {
		p.needs[i].rung = true
	}
}

// bellHere says whether the dashboard runs in ACC's own Windows Terminal
// profile, whose fragment scripts/acc.ps1 writes with bellStyle "taskbar"
// (design D2): Terminal gives each tab its profile's id in WT_PROFILE_ID.
// Anywhere else a BEL could sound.
func (m *Model) bellHere() bool {
	return strings.EqualFold(m.opt.Getenv("WT_PROFILE_ID"), input.AccProfile)
}

// title is the window's title: the most urgent need that waits before the
// dashboard's name, the name alone when none does (design D4). Bubble Tea
// clears the title it set when the program ends, a panic's end too, so the
// tab's own comes back (cursedRenderer.close).
func (m *Model) title() string {
	kind := -1
	for _, n := range m.ping.needs {
		if kind < 0 || n.kind < kind {
			kind = n.kind
		}
	}
	if kind < 0 {
		return windowTitle
	}
	return needTitles[kind]
}
