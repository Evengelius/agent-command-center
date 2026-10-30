package state

import (
	"testing"

	"github.com/Evengelius/agent-command-center/internal/event"
)

// Plan 70, criterion 70.1: the one-time offer of a summary request after a
// resume from the home screen, set, never set, withdrawn and kept.

// resumedReady is a start from the home screen of cli resuming asked, the
// handshake done and the CLI naming id.
func (b *bench) resumedReady(cli, asked, id string) *bench {
	name := map[string]string{"claude-code": "Claude Code", "codex": "OpenAI Codex"}[cli]
	b.asking().do(Confirm{Provider: cli, Project: project, Resume: asked, ResumeProject: project})
	b.do(ValidationOK{Provider: cli, ProviderName: name, Project: project})
	b.ev(event.SessionSpawned, "", event.Spawned{Dir: project})
	b.ev(event.SessionReady, "", event.Ready{})
	b.ev(event.SessionStarted, "", event.Started{SessionID: id, Cwd: project})
	return b
}

func TestSummaryOfferSet(t *testing.T) {
	for _, cli := range []string{"claude-code", "codex"} {
		b := newBench(t).resumedReady(cli, "s-1", "s-1")
		b.eq(cli+": the session asked for", b.s.Session.SummaryOffered, true)
		b.eq(cli+": the session's phase", b.s.Session.Phase, Ready)
	}
	// Codex answers thread/resume before the handshake is done: the offer
	// waits for Ready, which zone B needs to show it.
	c := newBench(t).asking()
	c.do(Confirm{Provider: "codex", Project: project, Resume: "th-1", ResumeProject: project})
	c.do(ValidationOK{Provider: "codex", ProviderName: "OpenAI Codex", Project: project})
	c.ev(event.SessionSpawned, "", event.Spawned{Dir: project})
	c.ev(event.SessionStarted, "", event.Started{SessionID: "th-1", Cwd: project})
	c.eq("named while handshaking", c.s.Session.SummaryOffered, true)
	c.ev(event.SessionReady, "", event.Ready{})
	c.eq("ready", c.s.Session.Phase, Ready)
}

func TestSummaryOfferNever(t *testing.T) {
	// A message of the owner's before the CLI names the session.
	b := newBench(t).asking()
	b.do(Confirm{Provider: "claude-code", Project: project, Resume: "s-1", ResumeProject: project})
	b.do(ValidationOK{Provider: "claude-code", ProviderName: "Claude Code", Project: project})
	b.ev(event.SessionSpawned, "", event.Spawned{Dir: project})
	b.ev(event.SessionReady, "", event.Ready{})
	b.do(Send{Text: "hi"})
	b.ev(event.SessionStarted, "", event.Started{SessionID: "s-1", Cwd: project})
	b.eq("after the owner's message", b.s.Session.SummaryOffered, false)

	// A resume the CLI refuses, or that fails: no session.started.
	r := newBench(t).asking()
	r.do(Confirm{Provider: "claude-code", Project: project, Resume: "s-1", ResumeProject: project})
	r.do(ValidationOK{Provider: "claude-code", ProviderName: "Claude Code", Project: project})
	r.ev(event.SessionSpawned, "", event.Spawned{Dir: project})
	r.ev(event.Error, "", event.Problem{Message: "Claude Code: No conversation found with session ID: s-1", NoSession: true})
	r.eq("refused", r.s.Session.SummaryOffered, false)
	f := newBench(t).asking()
	f.do(Confirm{Provider: "codex", Project: project, Resume: "th-1", ResumeProject: project})
	f.do(ValidationOK{Provider: "codex", ProviderName: "OpenAI Codex", Project: project})
	f.ev(event.SessionSpawned, "", event.Spawned{Dir: project})
	f.do(HandshakeTimeout{})
	f.eq("failed", f.s.Session.SummaryOffered, false)

	// A crash resume (Ctrl+R): a new session machine, no resume asked of
	// the home screen, even with the earlier offer unanswered.
	c := newBench(t).resumedReady("claude-code", "s-1", "s-1")
	c.exit(1)
	c.do(Resume{})
	c.eq("crash resume", c.kinds(), "start-session")
	c.ev(event.SessionSpawned, "", event.Spawned{Dir: project})
	c.ev(event.SessionReady, "", event.Ready{})
	c.ev(event.SessionStarted, "", event.Started{SessionID: "s-1", Cwd: project})
	c.eq("crash resume: the offer", c.s.Session.SummaryOffered, false)

	// A mock run.
	m := newBench(t)
	m.do(Launch{Mock: "demo.jsonl"})
	m.ev(event.SessionSpawned, "", event.Spawned{})
	m.ev(event.SessionReady, "", event.Ready{})
	m.ev(event.SessionStarted, "", event.Started{SessionID: "s1"})
	m.eq("mock", m.s.Session.SummaryOffered, false)
}

func TestSummaryOfferKept(t *testing.T) {
	// The CLI's own reads: Claude Code's usage read answers with telemetry.
	b := newBench(t).resumedReady("claude-code", "s-1", "s-1")
	b.ev(event.Telemetry, "", event.Metrics{Items: []event.Metric{{Name: "quota.5h", Value: 12, Unit: "%", Documented: true, Source: "usage_report"}}})
	b.eq("a usage read", b.s.Session.SummaryOffered, true)

	// A send refused while a turn the CLI started runs; the offer stays
	// through that turn and after it.
	b.ev(event.MessageMarker, "", event.Marker{Name: "message_start"})
	b.eq("a turn of the CLI's", b.s.Turn.ByCLI, true)
	b.do(Send{Text: "hi"})
	b.eq("refused", b.kinds(), "")
	b.ev(event.TurnCompleted, "", event.TurnEnd{Outcome: "completed"})
	b.eq("turn", b.s.Turn.Phase, Idle)
	b.eq("a refused send, a turn of the CLI's", b.s.Session.SummaryOffered, true)

	// A Codex slash command held back.
	c := newBench(t).resumedReady("codex", "th-1", "th-1")
	c.do(Send{Text: "/model"})
	c.eq("held", c.kinds(), "")
	c.eq("a slash command held", c.s.Session.SummaryOffered, true)

	// A write that fails: the message never reached the CLI.
	w := newBench(t).resumedReady("claude-code", "s-1", "s-1")
	w.do(Send{Text: "hi"})
	w.do(WriteFailed{Message: "pipe closed"})
	w.eq("restored", w.kinds(), "restore-draft")
	w.eq("a write that failed", w.s.Session.SummaryOffered, true)

	// Accepted after all: gone.
	w.do(Send{Text: "hi"})
	w.ev(event.MessageDelta, "", event.Delta{MessageID: "m1", Block: "text", Text: "Hel"})
	w.eq("accepted", w.s.Session.SummaryOffered, false)
	w.ev(event.TurnCompleted, "", event.TurnEnd{Outcome: "completed"})
	w.eq("after the turn", w.s.Session.SummaryOffered, false)
}
