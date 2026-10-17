package state

import (
	"fmt"
	"strings"
	"time"

	"github.com/Evengelius/agent-command-center/internal/event"
)

// Action is something the owner or the application did: a key, a button, a
// command's result, a timer. Events come from the CLI; actions from here.
type Action interface{ action() }

// The actions.
type (
	Launch        struct{ Mock string } // Mock: the --mock file, "" for a normal start
	DetectionDone struct {
		Installed []string
		CLIs      []CLI // what detection found, for the startup screen
	}
	Confirm          struct{ Provider, Project string }
	ValidationFailed struct{ Field, Message string }
	ValidationOK     struct{ Provider, ProviderName, Project, Branch string } // Project as validated
	SpawnFailed      struct{ Message string }
	HandshakeTimeout struct{}
	Back             struct{}
	Quit             struct{}
	Stay             struct{} // Esc while Ctrl+Q asks to confirm: the session goes on
	Resume           struct{} // start the crashed session's CLI session again in a new process
	Send             struct{ Text string }
	WriteFailed      struct{ Message string }
	Interrupt        struct{}
	InterruptTimeout struct{}
	ForceStop        struct{}
	Answer           struct{ ID, Decision string }
	AnswerFailed     struct{ ID, Message string }
	CancelTurn       struct{ ID string } // c on an approval card: cancel it, then interrupt the turn
	CloseTimeout     struct{}
	Tick             struct{ At time.Time }
)

func (Launch) action()           {}
func (DetectionDone) action()    {}
func (Confirm) action()          {}
func (ValidationFailed) action() {}
func (ValidationOK) action()     {}
func (SpawnFailed) action()      {}
func (HandshakeTimeout) action() {}
func (Back) action()             {}
func (Quit) action()             {}
func (Stay) action()             {}
func (Resume) action()           {}
func (Send) action()             {}
func (WriteFailed) action()      {}
func (Interrupt) action()        {}
func (InterruptTimeout) action() {}
func (ForceStop) action()        {}
func (Answer) action()           {}
func (AnswerFailed) action()     {}
func (CancelTurn) action()       {}
func (CloseTimeout) action()     {}
func (Tick) action()             {}

// MockProvider is the provider id of mock mode.
const MockProvider = "mock"

// Do applies one action and returns the commands it calls for.
func (s *State) Do(a Action) []Command {
	defer s.syncStage(time.Time{}) // an action can start or end a turn
	switch a := a.(type) {
	case Launch:
		if a.Mock == "" {
			s.Startup.Phase = Detecting
			return nil
		}
		s.Context = Context{Provider: MockProvider, ProviderName: "Mock data", Project: a.Mock, Mock: true}
		return s.start()
	case DetectionDone:
		if s.Startup.Phase == Detecting {
			s.Startup.Phase, s.Startup.Installed, s.Startup.CLIs = Asking, a.Installed, a.CLIs
		}
	case Confirm:
		if s.Startup.Phase != Asking {
			s.doNote("confirm ignored while " + string(s.Startup.Phase))
			return nil
		}
		s.Startup.Provider, s.Startup.Project = a.Provider, a.Project
		if !contains(s.Startup.Installed, a.Provider) {
			s.Startup.Field, s.Startup.Message = "provider", "not installed"
			return nil
		}
		s.Startup.Phase, s.Startup.Field, s.Startup.Message = Validating, "", ""
		return []Command{{Kind: CmdValidate}}
	case ValidationFailed:
		if s.Startup.Phase == Validating {
			s.Startup.Phase, s.Startup.Field, s.Startup.Message = Asking, a.Field, a.Message
		}
	case ValidationOK:
		if s.Startup.Phase != Validating {
			return nil
		}
		s.Context = Context{Provider: a.Provider, ProviderName: a.ProviderName, Project: a.Project, Branch: a.Branch}
		return s.start()
	case SpawnFailed:
		if s.Session.Phase == Spawning {
			s.Session.Reason = a.Message
			return s.crash(event.Event{}, "the CLI could not start: "+a.Message, false)
		}
	case HandshakeTimeout:
		if s.Session.Phase == Handshaking {
			return s.crash(event.Event{}, "no handshake within 60 s", true)
		}
	case Back:
		if s.Context.Mock {
			// A mock run has no startup screen: going back would open a
			// path to a live session (reliability.md, Mock mode).
			s.doNote("back ignored: a mock run ends with Ctrl+Q")
			return nil
		}
		if s.Startup.Phase == StartFailed {
			startup := s.Startup
			*s = *New()
			s.Startup = startup
			s.Startup.Phase, s.Startup.Failure = Asking, ""
		}
	case Quit:
		return s.quit()
	case Stay:
		s.QuitAsked = false
	case Resume:
		if s.Session.Phase != Crashed || s.Session.ID == "" || s.Startup.Phase == StartFailed || s.Context.Mock {
			s.doNote("nothing to resume")
			return nil
		}
		id := s.Session.ID
		s.Session = SessionState{Phase: Spawning, ID: id}
		return []Command{{Kind: CmdStartSession, ID: id}}
	case Send:
		return s.send(a)
	case WriteFailed:
		if s.Turn.Phase == Submitted {
			s.Turn.Phase, s.Turn.Outcome, s.Turn.Notice = Idle, "failed", "the message could not be sent: "+a.Message
			return []Command{{Kind: CmdRestoreDraft, Text: s.Turn.Draft}}
		}
	case Interrupt:
		return s.interrupt()
	case InterruptTimeout:
		if s.Turn.Phase == Interrupting {
			s.Turn.ForceStopOffered, s.Turn.Notice = true, "no turn end after the interrupt: force stop closes the session"
		}
	case ForceStop:
		if s.Turn.Phase == Interrupting {
			s.Turn.Phase, s.Turn.Outcome, s.Turn.ForceStopOffered = Idle, "interrupted", false
			if s.Session.Phase == Ready {
				s.Session.Phase = Closing
			}
			return []Command{{Kind: CmdClose}}
		}
	case Answer:
		return s.answer(a)
	case CancelTurn:
		return s.cancelTurn(a)
	case AnswerFailed:
		if ap := s.Approvals[a.ID]; ap != nil && ap.Status == Answering {
			ap.Status = Expired
			return s.crash(event.Event{}, "the answer could not be written: "+a.Message, true)
		}
	case CloseTimeout:
		if s.Session.Phase == Closing {
			return []Command{{Kind: CmdKill}}
		}
	case Tick:
		for k, v := range s.Metrics {
			if v.Present && !v.ResetsAt.IsZero() && !a.At.Before(v.ResetsAt) {
				s.Metrics[k] = Value{Unit: v.Unit, Source: v.Source, Documented: v.Documented, Reason: "window reset; waiting for the next update"}
			}
		}
	}
	return nil
}

func (s *State) start() []Command {
	s.Startup.Phase, s.Screen = Starting, ScreenDashboard
	s.Session = SessionState{Phase: Spawning}
	return []Command{{Kind: CmdStartSession}}
}

func (s *State) quit() []Command {
	switch s.Startup.Phase {
	case Detecting, Asking, Validating, StartFailed, "":
		s.Startup.Phase = Exited
		return []Command{{Kind: CmdExit}}
	}
	if s.Running() != "" && !s.QuitAsked {
		// ui-design.md, Keys: Ctrl+Q asks to confirm while a turn or an
		// agent runs (reliability.md, Interruption and cancellation).
		s.QuitAsked = true // the editor status asks (render)
		return nil
	}
	s.QuitAsked = false
	s.Startup.Phase = Exited
	switch s.Session.Phase {
	case Ready:
		s.Session.Phase = Closing
		return append(s.stopWork(), Command{Kind: CmdClose})
	case Spawning, Handshaking:
		s.Session.Phase = Closing
		return []Command{{Kind: CmdClose}}
	case Closing:
		return []Command{{Kind: CmdKill}}
	}
	return []Command{{Kind: CmdExit}}
}

// Running says what quitting would stop: the turn and the agents still
// running; "" when nothing runs.
func (s *State) Running() string {
	if s.Session.Phase != Ready {
		return ""
	}
	n := 0
	for _, k := range s.Order {
		if !s.Agents[k].Status.Final() {
			n++
		}
	}
	agents := ""
	switch {
	case n == 1:
		agents = "1 agent still running"
	case n > 1:
		agents = fmt.Sprintf("%d agents still running", n)
	}
	switch {
	case s.Turn.Phase != Idle && agents != "":
		return "the running turn and " + agents
	case s.Turn.Phase != Idle:
		return "the running turn"
	}
	return agents
}

// stopWork is the confirmed quit of a ready session: every pending approval
// is cancelled and the turn interrupted before the CLI's input is closed,
// so nothing the owner did not approve runs while it shuts down
// (reliability.md, Interruption and cancellation). The commands come in
// the order they must be written. A message not yet accepted stays
// Submitted: if the CLI ends before taking it, it goes back to the editor
// and the stored draft.
func (s *State) stopWork() []Command {
	var cmds []Command
	for _, id := range s.Queue {
		if ap := s.Approvals[id]; ap.Status == Pending {
			ap.Status, ap.Cancelled = Answering, true
			cmds = append(cmds, Command{Kind: CmdAnswer, ID: id, Decision: "cancel"})
		}
	}
	switch s.Turn.Phase {
	case Running, WaitingApproval:
		s.Turn.Phase = Interrupting
		cmds = append(cmds, Command{Kind: CmdInterrupt})
	case Submitted:
		cmds = append(cmds, Command{Kind: CmdInterrupt})
	}
	return cmds
}

func (s *State) send(a Send) []Command {
	s.Turn.Draft = a.Text
	if s.Session.Phase != Ready {
		s.Turn.Notice = "not sent: the session is " + string(s.Session.Phase) + "; the draft is kept"
		return nil
	}
	if s.Turn.Phase != Idle {
		s.Turn.Notice = "turn running: wait, or interrupt"
		return nil
	}
	if cmd := slashCommand(a.Text); cmd != "" && s.CLI() == "codex" && a.Text != s.Turn.SlashHeld {
		// Codex runs its slash commands in its own terminal only: through
		// app-server the text would reach the model as a message (L5). It is
		// held back once, with a note of the dashboard's; the draft stays.
		s.Turn.SlashHeld = a.Text
		s.Conversation = append(s.Conversation, Entry{Seq: s.lastSeq(), Role: "dashboard", Text: slashNote(cmd)})
		return nil
	}
	s.Turn.SlashHeld = ""
	s.Turn.Phase, s.Turn.Count, s.Turn.Outcome, s.Turn.Notice, s.Turn.ByCLI = Submitted, s.Turn.Count+1, "", "", false
	return []Command{{Kind: CmdWrite, Text: a.Text}}
}

func (s *State) interrupt() []Command {
	if s.Session.Phase != Ready {
		s.Turn.Notice = "no interrupt: the session is " + string(s.Session.Phase)
		return nil
	}
	if s.Turn.Phase != Running && s.Turn.Phase != WaitingApproval {
		s.Turn.Notice = "nothing to interrupt · Ctrl+Q quits"
		return nil
	}
	var cmds []Command
	// Cancelling the turn first denies the orchestrator's pending approvals.
	for _, id := range s.Queue {
		if ap := s.Approvals[id]; ap.Agent == "" && ap.Status == Pending {
			ap.Status, ap.Cancelled = Answering, true
			cmds = append(cmds, Command{Kind: CmdAnswer, ID: id, Decision: "cancel"})
		}
	}
	s.Turn.Phase, s.Turn.Notice = Interrupting, ""
	return append(cmds, Command{Kind: CmdInterrupt})
}

func (s *State) answer(a Answer) []Command {
	ap := s.Approvals[a.ID]
	if ap == nil {
		s.Errors = appendBounded(s.Errors, ErrorEntry{Message: "answer for an unknown approval: " + a.ID}, MaxErrors)
		return nil
	}
	if !contains(answers, a.Decision) {
		// Answers that persist a rule (permission_suggestions, execpolicy or
		// network amendments) are never offered (events-and-state.md, Approval machine).
		s.doNote("answer refused: " + a.Decision)
		return nil
	}
	if a.Decision == "acceptForSession" && !ap.ForSession {
		s.doNote("answer refused: acceptForSession not offered by " + a.ID) // plan 5, item 8
		return nil
	}
	if ap.Status != Pending {
		s.doNote("second answer ignored: " + a.ID)
		return nil
	}
	if s.Session.Phase != Ready {
		s.Turn.Notice = "not answered: the session is " + string(s.Session.Phase)
		return nil
	}
	ap.Status = Answering
	return []Command{{Kind: CmdAnswer, ID: a.ID, Decision: a.Decision}}
}

// answers are the decisions the owner can give; none persists a rule.
var answers = []string{"allow", "deny", "accept", "acceptForSession", "decline", "cancel"}

func (s *State) doNote(msg string) {
	s.Notes = appendBounded(s.Notes, "action: "+msg, MaxErrors)
}

func contains(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

// cancelTurn is c on an approval card: the card is cancelled (Claude Code
// writes a deny), then the turn is interrupted, which cancels the
// orchestrator's other pending approvals first (ui-design.md, Approvals).
// The commands come in the order they must be written.
func (s *State) cancelTurn(a CancelTurn) []Command {
	ap := s.Approvals[a.ID]
	if ap == nil || ap.Status != Pending {
		s.doNote("cancel ignored: " + a.ID + " is not pending")
		return nil
	}
	if s.Session.Phase != Ready || (s.Turn.Phase != Running && s.Turn.Phase != WaitingApproval) {
		return s.interrupt() // says why nothing is interrupted
	}
	ap.Status, ap.Cancelled = Answering, true
	cmds := []Command{{Kind: CmdAnswer, ID: a.ID, Decision: "cancel"}}
	return append(cmds, s.interrupt()...)
}

// slashCommand is the command a message starts with, as Codex's terminal
// reads one ("/status", "/model"), or "".
func slashCommand(text string) string {
	w := strings.TrimSpace(text)
	if i := strings.IndexAny(w, " \t\r\n"); i >= 0 {
		w = w[:i]
	}
	if len(w) < 2 || w[0] != '/' || w[1] < 'a' || w[1] > 'z' {
		return ""
	}
	for _, c := range w[2:] {
		if (c < 'a' || c > 'z') && c != '-' {
			return ""
		}
	}
	return w
}

// slashNote is the dashboard's note for a held slash command.
func slashNote(cmd string) string {
	n := cmd + " is a command of Codex's own terminal; it does not work here, so nothing was sent."
	if cmd == "/status" {
		n += " Zone D shows the 5h and weekly limits."
	}
	return n + "\nPress Enter again to send it to the model as a message."
}

// lastSeq is the last event's sequence number, for an entry the dashboard
// adds between events.
func (s *State) lastSeq() uint64 {
	if n := len(s.Log); n > 0 {
		return s.Log[n-1].Seq
	}
	return 0
}
