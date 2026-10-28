package app

import (
	"errors"
	"fmt"
	"time"

	tea "charm.land/bubbletea/v2"
)

// Windows' notifications for the ping (plan 64, brief amendment 66;
// ui-design.md, The ping): at each ring, a notification of the most urgent
// need rung, its text generic (design D3); taken out of Windows' list once
// nothing waits and when the dashboard quits (design D4). The calls go
// through Options.Notify and Options.Unnotify, the input part's notifier in
// the real run, one at a time in the order Update asks for them.

// notifyQueue is how many notification calls may wait; one more is refused
// rather than blocking the screen.
const notifyQueue = 16

// notifyClearWait bounds the clear the dashboard waits for when it quits.
const notifyClearWait = 2 * time.Second

// noticeTexts are a notification's title and text by need, %s the CLI's
// name: nothing of the session's content, no command, path, question or
// project (design D3).
var noticeTexts = [...][2]string{
	needApproval: {"Approval required", "%s is waiting for your approval."},
	needQuestion: {"Question waiting", "%s has a question for you."},
	needCrash:    {"Session ended", "%s stopped unexpectedly."},
	needFailed:   {"Turn failed", "%s's turn ended with an error."},
	needTurn:     {"Turn completed", "%s has finished its task."},
}

// notification is a need's notification: its title and text, with the
// CLI's name as the startup screen shows it.
func (m *Model) notification(kind int) (title, text string) {
	name := m.s.Context.ProviderName
	switch {
	case m.s.Context.Mock:
		name = "The mock session"
	case name == "":
		name = "The agent CLI"
	}
	return noticeTexts[kind][0], fmt.Sprintf(noticeTexts[kind][1], name)
}

type notifyJob struct {
	run  func() error
	done chan error
}

// notifiedMsg is a notification call's result: a failure is shown in the
// toast.
type notifiedMsg struct{ err error }

// notifySetUpMsg is Options.NotifySetUp's answer: the calls, or none and
// why.
type notifySetUpMsg struct {
	notify   func(title, text string) error
	unnotify func() error
	problem  string
}

// notifyCall queues a notification call, to run after every one queued
// before it: Bubble Tea runs a batch's commands concurrently, and a clear
// must never overtake the notification it clears. Called from Update only.
func (m *Model) notifyCall(run func() error) tea.Cmd {
	if m.notifies == nil {
		m.notifies = make(chan notifyJob, notifyQueue)
		go func(jobs <-chan notifyJob) {
			for j := range jobs {
				j.done <- runNotify(j.run)
			}
		}(m.notifies)
	}
	j := notifyJob{run: run, done: make(chan error, 1)}
	select {
	case m.notifies <- j:
	default:
		return func() tea.Msg { return notifiedMsg{errors.New("too many notifications waiting")} }
	}
	return func() tea.Msg { return notifiedMsg{<-j.done} }
}

// runNotify runs one call; a panic in it is its failure, never the
// program's end.
func runNotify(run func() error) (err error) {
	defer func() {
		if v := recover(); v != nil {
			err = fmt.Errorf("panic: %v", v)
		}
	}()
	return run()
}

// notify puts up the notification for a need.
func (m *Model) notify(kind int) tea.Cmd {
	title, text := m.notification(kind)
	show := m.opt.Notify
	m.ping.notified = true
	return m.notifyCall(func() error { return show(title, text) })
}

// unnotify takes ACC's notification out of Windows' list.
func (m *Model) unnotify() tea.Cmd {
	m.ping.notified = false
	if m.opt.Unnotify == nil {
		return nil
	}
	return m.notifyCall(m.opt.Unnotify)
}

// stopNotifying is the dashboard's end: the notification still up taken
// out, waited for a while, then the queue closed (design D4).
func (m *Model) stopNotifying() {
	if m.notifies == nil {
		return
	}
	if m.ping.notified && m.opt.Unnotify != nil {
		m.ping.notified = false
		j := notifyJob{run: m.opt.Unnotify, done: make(chan error, 1)}
		select {
		case m.notifies <- j:
			select {
			case <-j.done:
			case <-time.After(notifyClearWait):
			}
		default:
		}
	}
	close(m.notifies)
	m.notifies = nil
}
