package render

import (
	"github.com/Evengelius/agent-command-center/internal/state"
)

// The editor's send and newline keys are the only configurable bindings
// (plan 21, item 5.1). The screen names them from UI.SendKeys and
// UI.NewlineKeys; none given is today's keys.
var (
	defaultSendKeys    = []string{"Enter"}
	defaultNewlineKeys = []string{"Ctrl+J", "Shift+Enter", "Alt+Enter"}
)

// sendKeys are the send keys in force, as named on screen.
func (ui UI) sendKeys() []string {
	if len(ui.SendKeys) == 0 {
		return defaultSendKeys
	}
	return ui.SendKeys
}

// newlineKeys are the newline keys in force, as named on screen.
func (ui UI) newlineKeys() []string {
	if len(ui.NewlineKeys) == 0 {
		return defaultNewlineKeys
	}
	return ui.NewlineKeys
}

// sendNewline is the hint for the two keys the hints name: the first send key
// "send", the first newline key "newline".
func sendNewline(ui UI) string {
	return ui.sendKeys()[0] + " send · " + ui.newlineKeys()[0] + " newline"
}

// idleHint is the editor's status while a send key sends.
func idleHint(ui UI) string { return sendNewline(ui) }

// keysProblemShown reports whether the editor's status row shows the
// configuration file's problem in place of the idle hint: a mock run has no
// startup screen, so it shows it there, until the first message is sent.
func keysProblemShown(s *state.State, ui UI) bool { return ui.KeysProblem != "" && s.Context.Mock }

// idleStatus is the editor's status when it has nothing to say: the idle
// hint, or the configuration file's problem in a mock run.
func idleStatus(s *state.State, ui UI) string {
	if keysProblemShown(s, ui) {
		return ui.KeysProblem
	}
	return idleHint(ui)
}
