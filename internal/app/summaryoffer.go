package app

import (
	tea "charm.land/bubbletea/v2"

	"github.com/Evengelius/agent-command-center/internal/render"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// The one-time summary suggestion after a resume from the home screen
// (plan 70): its keys, its buttons and the request it prepares.

// summaryRequest is the request Prepare summary puts in the draft, the
// owner's words (plan 70, design D7). It asks, and enforces nothing: the
// CLI's permissions and approvals still decide what the turn may do.
const summaryRequest = "Summarize the work we have done in this conversation so far, not the project in general: the goals, the important decisions, what was completed, what remains open, and the recommended next steps. Answer from this conversation as you have it now: do not read files, run tools or commands, or change anything. If parts of the earlier conversation are missing or only available to you as a compacted summary, say so rather than filling the gaps. Keep it concise."

// summaryBusy is the toast of Prepare summary over a draft that is not
// empty, which it leaves as it is (plan 70, design D5).
const summaryBusy = "The draft is not empty: send or clear it, then Prepare summary"

// focusSummary gives the suggestion the keyboard focus, Prepare summary
// selected, and shows it: it is zone B's last rows (plan 70, design D3).
func (m *Model) focusSummary() {
	m.ui.Summary = render.SummaryFocus{On: true}
	m.ui.Entry, m.ui.EntryReview = 0, 0
	m.follow()
}

// summaryKey handles a key while the suggestion has the focus (plan 70,
// design D3): ← and → select a button, Enter acts on it, Esc and ↓ return
// to the input, ↑ goes on to the newest entry's stop, where there is one.
// done is false for any other key, which leaves the focus and does what it
// does without it.
func (m *Model) summaryKey(k tea.KeyPressMsg) (cmds []tea.Cmd, done bool) {
	switch k.String() {
	case "left", "right":
		m.ui.Summary.NotNow = k.String() == "right"
		return nil, true
	case "enter":
		if m.ui.Summary.NotNow {
			return m.summaryNotNow(), true
		}
		return m.prepareSummary(), true
	case "esc", "down":
		m.ui.Summary = render.SummaryFocus{}
		return nil, true
	case "up":
		if m.lastStop() {
			m.ui.Summary = render.SummaryFocus{}
		}
		return nil, true
	}
	m.ui.Summary = render.SummaryFocus{}
	return nil, false
}

// summaryClick is a left click on one of the suggestion's buttons, which
// acts without the focus (plan 70, design D4).
func (m *Model) summaryClick(item int) []tea.Cmd {
	m.ui.Focus = "B"
	switch item {
	case render.SummaryPrepare:
		return m.prepareSummary()
	case render.SummaryNotNow:
		return m.summaryNotNow()
	}
	return nil
}

// prepareSummary puts the summary request in an empty draft, the cursor
// at its end, and the offer is answered; nothing is written to the CLI
// before the owner's Enter (plan 70, design D5). A draft with text or an
// image, one restored at the session's start among them, stays as it is,
// and so does the suggestion: a toast says why.
func (m *Model) prepareSummary() []tea.Cmd {
	if m.edit.Text() != "" || len(m.attach) > 0 {
		return m.toast(summaryBusy)
	}
	m.edit.Insert(summaryRequest)
	m.ui.Summary = render.SummaryFocus{}
	return m.do(state.SummaryPrepared{})
}

// summaryNotNow answers the offer with Not now: the draft, its images and
// the session unchanged (plan 70, design D6).
func (m *Model) summaryNotNow() []tea.Cmd {
	m.ui.Summary = render.SummaryFocus{}
	return m.do(state.SummaryDismissed{})
}
