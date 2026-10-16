package app

import (
	tea "charm.land/bubbletea/v2"
)

// writeQueue is how many writes may wait for the session; one more is
// refused rather than blocking the screen.
const writeQueue = 64

// writeJob is one write to the session and the message its result becomes.
type writeJob struct {
	run  func() tea.Msg
	done chan tea.Msg
}

// write queues a write to the session, to run after every write queued
// before it. Bubble Tea runs a batch's commands concurrently, so without the
// queue an approval's answer and the interrupt that follows it could reach
// the CLI in either order (ui-design.md, Approvals: c is deny, then
// interrupt). Called from Update only; full is the message when the queue
// is full.
func (m *Model) write(run func() tea.Msg, full tea.Msg) tea.Cmd {
	if m.writes == nil {
		m.writes = make(chan writeJob, writeQueue)
		go func(jobs <-chan writeJob) {
			for j := range jobs {
				j.done <- j.run()
			}
		}(m.writes)
	}
	j := writeJob{run: run, done: make(chan tea.Msg, 1)}
	select {
	case m.writes <- j:
	default:
		return func() tea.Msg { return full }
	}
	return func() tea.Msg { return <-j.done }
}
