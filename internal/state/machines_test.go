package state

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Evengelius/agent-command-center/internal/event"
)

// Test 3 (proposal step 0, item 7): one case per transition and per invalid
// transition of doc/events-and-state.md, State machines. The doc is parsed:
// every table row must match exactly one case, and every clause of each
// "Invalid:" paragraph exactly one invalid case, so a row added to the doc
// without its case fails here.

const project = `C:\work\proj`

type bench struct {
	t    *testing.T
	s    *State
	seq  uint64
	cmds []Command
}

func newBench(t *testing.T) *bench { return &bench{t: t, s: New()} }

func (b *bench) ev(kind event.Kind, agent string, p any) []Command {
	b.seq++
	b.cmds = b.s.Apply(event.Event{Seq: b.seq, At: time.Date(2026, 10, 16, 0, 0, int(b.seq), 0, time.UTC),
		Provider: "test", Kind: kind, Origin: event.Observed, Agent: agent, Payload: p})
	return b.cmds
}

func (b *bench) do(a Action) []Command {
	b.cmds = b.s.Do(a)
	return b.cmds
}

func (b *bench) asking() *bench {
	b.do(Launch{})
	b.do(DetectionDone{Installed: []string{"claude-code", "codex"}})
	return b
}

func (b *bench) validating() *bench {
	b.asking().do(Confirm{Provider: "claude-code", Project: project})
	return b
}

func (b *bench) starting() *bench {
	b.validating().do(ValidationOK{Provider: "claude-code", ProviderName: "Claude Code", Project: project})
	return b
}

func (b *bench) handshaking() *bench {
	b.starting().ev(event.SessionSpawned, "", event.Spawned{Dir: project})
	return b
}

func (b *bench) ready() *bench {
	b.handshaking().ev(event.SessionReady, "", event.Ready{})
	return b
}

func (b *bench) submitted() *bench {
	b.ready().do(Send{Text: "hi"})
	b.ev(event.UserMessage, "", event.User{Text: "hi"})
	return b
}

func (b *bench) running() *bench {
	b.submitted().ev(event.TurnStarted, "", nil)
	return b
}

func (b *bench) ask(agent, id, toolID string) []Command {
	return b.ev(event.ApprovalRequest, agent, event.Approval{ID: id, Tool: "Bash", ToolID: toolID})
}

func (b *bench) resolve(id, decision string) []Command {
	return b.ev(event.ApprovalResolved, "", event.Approval{ID: id, Decision: decision})
}

func (b *bench) turnEnd(agent, outcome string) []Command {
	return b.ev(event.TurnCompleted, agent, event.TurnEnd{Outcome: outcome})
}

func (b *bench) exit(code int) []Command {
	return b.ev(event.SessionEnded, "", event.Ended{Code: code, HasCode: true})
}

func (b *bench) eq(what string, got, want any) {
	b.t.Helper()
	if got != want {
		b.t.Errorf("%s: got %v, want %v", what, got, want)
	}
}

// kinds returns the command kinds of the last step.
func (b *bench) kinds() string {
	var k []string
	for _, c := range b.cmds {
		k = append(k, string(c.Kind))
	}
	return strings.Join(k, ",")
}

func (b *bench) agent(key string) *Agent {
	b.t.Helper()
	a := b.s.Agents[key]
	if a == nil {
		b.t.Fatalf("no agent %q", key)
	}
	return a
}

func (b *bench) noted(part string) bool {
	for _, n := range b.s.Notes {
		if strings.Contains(n, part) {
			return true
		}
	}
	return false
}

type machineCase struct {
	machine string
	row     string // "From | trigger", a prefix of the doc row
	run     func(b *bench)
}

var transitions = []machineCase{
	// Startup machine
	{"Startup", "(launch) | application start", func(b *bench) {
		b.do(Launch{})
		b.eq("phase", b.s.Startup.Phase, Detecting)
		b.eq("screen", b.s.Screen, ScreenStartup)
	}},
	{"Startup", "(launch) | application start with `--mock <file>`", func(b *bench) {
		b.do(Launch{Mock: "demo.jsonl"})
		b.eq("phase", b.s.Startup.Phase, Starting)
		b.eq("context", b.s.Context, Context{Provider: MockProvider, ProviderName: "Mock data", Project: "demo.jsonl", Mock: true})
		b.eq("session", b.s.Session.Phase, Spawning)
		b.eq("screen", b.s.Screen, ScreenDashboard)
		b.eq("commands", b.kinds(), "start-session")
	}},
	{"Startup", "`Detecting` | both detections returned or timed out", func(b *bench) {
		b.asking()
		b.eq("phase", b.s.Startup.Phase, Asking)
		b.eq("installed", strings.Join(b.s.Startup.Installed, ","), "claude-code,codex")
	}},
	{"Startup", "`Asking` | Confirm pressed", func(b *bench) {
		b.validating()
		b.eq("phase", b.s.Startup.Phase, Validating)
		b.eq("commands", b.kinds(), "validate")
	}},
	{"Startup", "`Validating` | path or provider invalid", func(b *bench) {
		b.validating().do(ValidationFailed{Field: "project", Message: "Not found"})
		b.eq("phase", b.s.Startup.Phase, Asking)
		b.eq("field", b.s.Startup.Field, "project")
		b.eq("message", b.s.Startup.Message, "Not found")
		b.eq("answer kept", b.s.Startup.Project, project)
	}},
	{"Startup", "`Validating` | both valid", func(b *bench) {
		b.starting()
		b.eq("phase", b.s.Startup.Phase, Starting)
		b.eq("screen", b.s.Screen, ScreenDashboard)
		b.eq("session", b.s.Session.Phase, Spawning)
		b.eq("project", b.s.Context.Project, project)
		b.eq("commands", b.kinds(), "start-session")
	}},
	{"Startup", "`Starting` | handshake done (`session.ready`)", func(b *bench) {
		b.ready()
		b.eq("phase", b.s.Startup.Phase, StartReady)
		b.eq("session", b.s.Session.Phase, Ready)
	}},
	{"Startup", "`Starting` | quit", func(b *bench) {
		b.handshaking().do(Quit{})
		b.eq("session", b.s.Session.Phase, Closing)
		b.eq("commands", b.kinds(), "close")
		b.exit(0)
		b.eq("session", b.s.Session.Phase, Closed)
		b.eq("commands", b.kinds(), "exit")
	}},
	{"Startup", "`Starting` | spawn error, handshake timeout, process exit", func(b *bench) {
		b.starting().do(SpawnFailed{Message: "not found"})
		b.eq("spawn error", b.s.Startup.Phase, StartFailed)
		b.eq("session", b.s.Session.Phase, Crashed)

		c := newBench(b.t).handshaking()
		c.do(HandshakeTimeout{})
		c.eq("timeout", c.s.Startup.Phase, StartFailed)
		c.eq("commands", c.kinds(), "close")

		d := newBench(b.t).handshaking()
		d.exit(1)
		d.eq("exit", d.s.Startup.Phase, StartFailed)
		d.eq("exit code", d.s.Session.ExitCode, 1)
	}},
	{"Startup", "`StartFailed` | owner chooses \"back\"", func(b *bench) {
		b.handshaking().exit(1)
		b.do(Back{})
		b.eq("phase", b.s.Startup.Phase, Asking)
		b.eq("screen", b.s.Screen, ScreenStartup)
		b.eq("provider kept", b.s.Startup.Provider, "claude-code")
		b.eq("project kept", b.s.Startup.Project, project)
		b.eq("session reset", b.s.Session.Phase, SessionPhase(""))

		m := newBench(b.t)
		m.do(Launch{Mock: "demo.jsonl"})
		m.do(SpawnFailed{Message: "bad file"})
		m.do(Back{})
		m.eq("mock stays", m.s.Startup.Phase, StartFailed)
		m.eq("mock screen", m.s.Screen, ScreenDashboard)
		m.eq("mock context kept", m.s.Context.Mock, true)
	}},
	{"Startup", "`Ready` | quit", func(b *bench) {
		b.ready().do(Quit{})
		b.eq("session", b.s.Session.Phase, Closing)
		b.eq("commands", b.kinds(), "close")
		b.exit(0)
		b.eq("commands", b.kinds(), "exit")
	}},
	{"Startup", "`Detecting`, `Asking`, `Validating`, `StartFailed` | quit", func(b *bench) {
		setups := map[string]func(*bench){
			"Detecting":   func(c *bench) { c.do(Launch{}) },
			"Asking":      func(c *bench) { c.asking() },
			"Validating":  func(c *bench) { c.validating() },
			"StartFailed": func(c *bench) { c.starting().do(SpawnFailed{Message: "x"}) },
		}
		for name, setup := range setups {
			c := newBench(b.t)
			setup(c)
			c.do(Quit{})
			c.eq(name+" phase", c.s.Startup.Phase, Exited)
			c.eq(name+" commands", c.kinds(), "exit")
		}
	}},

	// Session machine
	{"Session", "`Spawning` | process started", func(b *bench) {
		b.handshaking()
		b.eq("session", b.s.Session.Phase, Handshaking)
	}},
	{"Session", "`Spawning` | spawn error", func(b *bench) {
		b.starting().do(SpawnFailed{Message: "access denied"})
		b.eq("session", b.s.Session.Phase, Crashed)
		b.eq("reason", strings.Contains(b.s.Session.Reason, "access denied"), true)
	}},
	{"Session", "`Handshaking` | `initialize` response", func(b *bench) {
		b.ready()
		b.eq("session", b.s.Session.Phase, Ready)
	}},
	{"Session", "`Handshaking` | 60 s timeout, error response, `cwd` mismatch", func(b *bench) {
		b.handshaking().do(HandshakeTimeout{})
		b.eq("timeout", b.s.Session.Phase, Crashed)
		b.eq("timeout commands", b.kinds(), "close")

		c := newBench(b.t).handshaking()
		c.ev(event.Error, "", event.Problem{Message: "unsupported version"})
		c.eq("error response", c.s.Session.Phase, Crashed)
		c.eq("error commands", c.kinds(), "close")

		d := newBench(b.t).handshaking()
		d.ev(event.SessionStarted, "", event.Started{SessionID: "s1", Cwd: `C:\elsewhere`})
		d.eq("cwd mismatch", d.s.Session.Phase, Crashed)
		d.eq("cwd commands", d.kinds(), "close")
		d.eq("start failed", d.s.Startup.Phase, StartFailed)

		m := newBench(b.t)
		m.do(Launch{Mock: "demo.jsonl"})
		m.ev(event.SessionSpawned, "", event.Spawned{})
		m.ev(event.SessionStarted, "", event.Started{SessionID: "s1", Cwd: `C:\recorded`})
		m.eq("mock has no working directory", m.s.Session.Phase, Handshaking)
	}},
	{"Session", "`Ready` | `session.started` with a `cwd` other than the project", func(b *bench) {
		b.ready().ev(event.SessionStarted, "", event.Started{SessionID: "s1", Cwd: "c:/work/PROJ/"})
		b.eq("same path, other spelling", b.s.Session.Phase, Ready)
		b.eq("session id", b.s.Session.ID, "s1")
		b.ev(event.SessionStarted, "", event.Started{SessionID: "s1", Cwd: `C:\work\proj2`})
		b.eq("mismatch", b.s.Session.Phase, Crashed)
		b.eq("commands", b.kinds(), "close")
	}},
	{"Session", "`Ready` | owner quits", func(b *bench) {
		b.ready().do(Quit{})
		b.eq("session", b.s.Session.Phase, Closing)
		b.eq("commands", b.kinds(), "close")
	}},
	{"Session", "`Closing` | process exited", func(b *bench) {
		b.ready().do(Quit{})
		b.exit(0)
		b.eq("session", b.s.Session.Phase, Closed)
	}},
	{"Session", "`Closing` | 5 s, then 5 s more, elapsed", func(b *bench) {
		b.ready().do(Quit{})
		b.do(CloseTimeout{})
		b.eq("commands", b.kinds(), "kill")
		b.eq("still closing", b.s.Session.Phase, Closing)
		b.ev(event.SessionEnded, "", event.Ended{Reason: "killed"})
		b.eq("session", b.s.Session.Phase, Closed)
	}},
	{"Session", "`Ready`, `Handshaking` | process exited unexpectedly or stdout EOF", func(b *bench) {
		b.running().exit(3)
		b.eq("from Ready", b.s.Session.Phase, Crashed)
		b.eq("exit code", b.s.Session.ExitCode, 3)
		b.eq("turn", b.s.Turn.Phase, Idle)

		c := newBench(b.t).handshaking()
		c.ev(event.SessionEnded, "", event.Ended{Reason: "stdout EOF"})
		c.eq("from Handshaking", c.s.Session.Phase, Crashed)
	}},

	// Turn machine
	{"Turn", "`Idle` | owner sends a message", func(b *bench) {
		b.ready().do(Send{Text: "line 1\nline 2"})
		b.eq("turn", b.s.Turn.Phase, Submitted)
		b.eq("commands", b.kinds(), "write")
		b.eq("text byte for byte", b.cmds[0].Text, "line 1\nline 2")
		b.eq("count", b.s.Turn.Count, 1)
	}},
	{"Turn", "`Submitted` | first event of the turn", func(b *bench) {
		b.submitted().ev(event.MessageDelta, "", event.Delta{MessageID: "m1", Block: "text", Text: "Hel"})
		b.eq("Claude Code", b.s.Turn.Phase, Running)
		b.eq("commands", b.kinds(), "draft-accepted")
		b.eq("draft", b.s.Turn.Draft, "")
		last := b.s.Conversation[len(b.s.Conversation)-1]
		b.eq("the first event is applied too", last.Text, "Hel")

		c := newBench(b.t).submitted()
		c.ev(event.TurnStarted, "", nil)
		c.eq("Codex", c.s.Turn.Phase, Running)
		c.eq("commands", c.kinds(), "draft-accepted")

		d := newBench(b.t).submitted()
		d.ev(event.TurnStarted, "child", nil)
		d.eq("a child's event does not start the turn", d.s.Turn.Phase, Submitted)
	}},
	{"Turn", "`Submitted` | write to stdin fails", func(b *bench) {
		b.ready().do(Send{Text: "hi"})
		b.do(WriteFailed{Message: "pipe closed"})
		b.eq("turn", b.s.Turn.Phase, Idle)
		b.eq("outcome", b.s.Turn.Outcome, "failed")
		b.eq("commands", b.kinds(), "restore-draft")
		b.eq("draft", b.cmds[0].Text, "hi")
	}},
	{"Turn", "`Running` | `approval.requested` for the orchestrator or any agent", func(b *bench) {
		b.running().ask("", "a1", "")
		b.eq("orchestrator", b.s.Turn.Phase, WaitingApproval)

		c := newBench(b.t).running()
		c.ask("sub1", "a2", "")
		c.eq("agent", c.s.Turn.Phase, WaitingApproval)
	}},
	{"Turn", "`WaitingApproval` | every pending approval answered", func(b *bench) {
		b.running().ask("", "a1", "")
		b.ask("", "a2", "")
		b.resolve("a1", "allow")
		b.eq("one left", b.s.Turn.Phase, WaitingApproval)
		b.resolve("a2", "deny")
		b.eq("all answered", b.s.Turn.Phase, Running)
	}},
	{"Turn", "`Running`, `WaitingApproval` | owner interrupts", func(b *bench) {
		b.running().do(Interrupt{})
		b.eq("from Running", b.s.Turn.Phase, Interrupting)
		b.eq("commands", b.kinds(), "interrupt")

		c := newBench(b.t).running()
		c.ask("", "a1", "")
		c.do(Interrupt{})
		c.eq("from WaitingApproval", c.s.Turn.Phase, Interrupting)
		c.eq("commands", c.kinds(), "answer,interrupt")
		c.eq("decision", c.cmds[0].Decision, "cancel")
	}},
	{"Turn", "`Interrupting` | turn end reported", func(b *bench) {
		b.running().do(Interrupt{})
		b.turnEnd("", "failed")
		b.eq("turn", b.s.Turn.Phase, Idle)
		b.eq("outcome", b.s.Turn.Outcome, "interrupted")
	}},
	{"Turn", "`Interrupting` | no turn end within 10 s", func(b *bench) {
		b.running().do(Interrupt{})
		b.do(InterruptTimeout{})
		b.eq("turn", b.s.Turn.Phase, Interrupting)
		b.eq("offered", b.s.Turn.ForceStopOffered, true)
	}},
	{"Turn", "`Interrupting` | owner forces stop", func(b *bench) {
		b.running().do(Interrupt{})
		b.do(InterruptTimeout{})
		b.do(ForceStop{})
		b.eq("turn", b.s.Turn.Phase, Idle)
		b.eq("outcome", b.s.Turn.Outcome, "interrupted")
		b.eq("session", b.s.Session.Phase, Closing)
		b.eq("commands", b.kinds(), "close")
	}},
	{"Turn", "`WaitingApproval` | turn end reported", func(b *bench) {
		b.running().ask("", "a1", "")
		b.turnEnd("", "completed")
		b.eq("turn", b.s.Turn.Phase, Idle)
		b.eq("outcome", b.s.Turn.Outcome, "completed")
		b.eq("approval", b.s.Approvals["a1"].Status, Expired)
		b.resolve("a1", "resolved") // Codex's serverRequest/resolved after the turn end
		b.eq("stays expired", b.s.Approvals["a1"].Status, Expired)
	}},
	{"Turn", "`Idle` | approval from a background agent", func(b *bench) {
		b.ready().ev(event.AgentStarted, "", event.Agent{Key: "bg", Type: "subagent", Backgrounded: true})
		b.ask("bg", "a1", "")
		b.eq("turn", b.s.Turn.Phase, Idle)
		b.eq("agent", b.agent("bg").Status, AgentWaiting)
		b.eq("queued", strings.Join(b.s.Queue, ","), "a1")
	}},
	{"Turn", "`Idle` | the CLI starts a turn itself", func(b *bench) {
		b.ready().ev(event.MessageDelta, "", event.Delta{MessageID: "m0", Block: "text", Text: "late"})
		b.eq("a late delta opens nothing", b.s.Turn.Phase, Idle)
		b.ev(event.MessageMarker, "", event.Marker{MessageID: "m1", Name: "message_start"})
		b.eq("Claude Code", b.s.Turn.Phase, Running)
		b.eq("by the CLI", b.s.Turn.ByCLI, true)
		b.eq("no draft accepted", b.kinds(), "")
		b.eq("count", b.s.Turn.Count, 0)
		b.turnEnd("", "completed")
		b.eq("ends normally", b.s.Turn.Outcome, "completed")
		b.eq("not unexpected", b.noted("unexpected turn end"), false)
		b.do(Send{Text: "next"})
		b.eq("owner's turn", b.s.Turn.ByCLI, false)

		c := newBench(b.t).ready()
		c.ev(event.TurnStarted, "", nil)
		c.eq("Codex", c.s.Turn.Phase, Running)
		d := newBench(b.t).handshaking()
		d.ev(event.TurnStarted, "", nil)
		d.eq("not before Ready", d.s.Turn.Phase, Idle)
	}},
	{"Turn", "`Running` | turn end, success", func(b *bench) {
		b.running().turnEnd("", "completed")
		b.eq("turn", b.s.Turn.Phase, Idle)
		b.eq("outcome", b.s.Turn.Outcome, "completed")
		b.eq("commands", b.kinds(), "refresh")
	}},
	{"Turn", "`Running` | turn end, error", func(b *bench) {
		b.running().ev(event.TurnCompleted, "", event.TurnEnd{Outcome: "failed", Error: "model not supported"})
		b.eq("outcome", b.s.Turn.Outcome, "failed")
		b.eq("error shown", len(b.s.Errors), 1)
	}},
	{"Turn", "any | session `Crashed`", func(b *bench) {
		b.running().exit(1)
		b.eq("turn", b.s.Turn.Phase, Idle)
		b.eq("outcome", b.s.Turn.Outcome, "failed")
	}},

	// Agent machine
	{"Agent", "(none) | `agent.started`", func(b *bench) {
		b.running().ev(event.AgentStarted, "", event.Agent{Key: "t1", Type: "subagent", Label: "Explore"})
		b.eq("Claude Code", b.agent("t1").Status, AgentRunning)
		b.eq("label", b.agent("t1").Label, "Explore")
		b.ev(event.AgentStarted, "", event.Agent{Key: "th1", Type: "thread"})
		b.eq("Codex", b.agent("th1").Status, AgentAnnounced)
	}},
	{"Agent", "(none) | event with an unknown key", func(b *bench) {
		b.running().ev(event.ToolStarted, "x", event.Tool{ToolID: "tu1", Name: "Read"})
		b.eq("status", b.agent("x").Status, AgentRunning)
		b.eq("origin", b.agent("x").StatusOrigin, event.Inferred)
		b.eq("label", b.agent("x").Label, "unannounced agent")
	}},
	{"Agent", "`Announced` | child `thread/status/changed active` or `turn/started`", func(b *bench) {
		b.running().ev(event.AgentStarted, "", event.Agent{Key: "th1", Type: "thread"})
		b.ev(event.StatusChanged, "th1", event.Status{Status: "active"})
		b.eq("status changed", b.agent("th1").Status, AgentRunning)
		b.ev(event.AgentStarted, "", event.Agent{Key: "th2", Type: "thread"})
		b.ev(event.TurnStarted, "th2", nil)
		b.eq("turn started", b.agent("th2").Status, AgentRunning)
	}},
	{"Agent", "`Running` | `agent.progress`", func(b *bench) {
		b.running().ev(event.AgentStarted, "", event.Agent{Key: "t1", Type: "subagent"})
		b.ev(event.AgentProgress, "", event.Agent{Key: "t1", Tokens: 1200, HasTokens: true, LastTool: "Grep", ToolUses: 3})
		a := b.agent("t1")
		b.eq("status", a.Status, AgentRunning)
		b.eq("tokens", a.Tokens.V, 1200.0)
		b.eq("last tool", a.LastTool, "Grep")
		b.eq("tool uses", a.ToolUses, 3)
	}},
	{"Agent", "`Running` | approval request carrying this key", func(b *bench) {
		b.running().ev(event.AgentStarted, "", event.Agent{Key: "t1", Type: "subagent"})
		b.ask("t1", "a1", "")
		b.eq("status", b.agent("t1").Status, AgentWaiting)
	}},
	{"Agent", "`WaitingApproval` | its approvals answered", func(b *bench) {
		b.running().ev(event.AgentStarted, "", event.Agent{Key: "t1", Type: "subagent"})
		b.ask("t1", "a1", "")
		b.resolve("a1", "allow")
		b.eq("status", b.agent("t1").Status, AgentRunning)
	}},
	{"Agent", "`Running`, `WaitingApproval` | completed", func(b *bench) {
		b.running().ev(event.AgentStarted, "", event.Agent{Key: "t1", Type: "subagent"})
		b.ev(event.AgentCompleted, "", event.Agent{Key: "t1", Summary: "done"})
		b.eq("from Running", b.agent("t1").Status, AgentCompleted)
		b.eq("summary", b.agent("t1").Summary, "done")
		b.ev(event.AgentCompleted, "", event.Agent{Key: "t1"}) // task_updated, then task_notification
		b.eq("idempotent", b.agent("t1").Runs, 1)

		b.ev(event.AgentStarted, "", event.Agent{Key: "th1", Type: "thread"})
		b.ev(event.TurnStarted, "th1", nil)
		b.ask("th1", "a1", "")
		b.turnEnd("th1", "completed")
		b.eq("from WaitingApproval", b.agent("th1").Status, AgentCompleted)
		b.eq("its approval", b.s.Approvals["a1"].Status, Expired)
	}},
	{"Agent", "`Running`, `WaitingApproval` | failed", func(b *bench) {
		b.running().ev(event.AgentStarted, "", event.Agent{Key: "t1", Type: "subagent"})
		b.ev(event.AgentFailed, "", event.Agent{Key: "t1", Status: "failed"})
		b.eq("status", b.agent("t1").Status, AgentFailed)
		b.ev(event.AgentStarted, "", event.Agent{Key: "th1", Type: "thread"})
		b.ev(event.TurnStarted, "th1", nil)
		b.turnEnd("th1", "failed")
		b.eq("child turn", b.agent("th1").Status, AgentFailed)
	}},
	{"Agent", "`Running`, `WaitingApproval` | stopped or killed", func(b *bench) {
		b.running().ev(event.AgentStarted, "", event.Agent{Key: "t1", Type: "subagent"})
		b.ev(event.AgentFailed, "", event.Agent{Key: "t1", Status: "killed"})
		b.eq("killed", b.agent("t1").Status, AgentStopped)
		b.ev(event.AgentStarted, "", event.Agent{Key: "th1", Type: "thread"})
		b.ev(event.TurnStarted, "th1", nil)
		b.turnEnd("th1", "interrupted")
		b.eq("child interrupted", b.agent("th1").Status, AgentStopped)
	}},
	{"Agent", "`Running` | absent from `background_tasks_changed`", func(b *bench) {
		b.running().ev(event.AgentSet, "", event.AgentKeys{Keys: []string{"a", "b"}})
		b.ev(event.AgentSet, "", event.AgentKeys{Keys: []string{"b"}})
		b.eq("absent", b.agent("a").Status, AgentEnded)
		b.eq("origin", b.agent("a").StatusOrigin, event.Inferred)
		b.eq("present", b.agent("b").Status, AgentRunning)
	}},
	{"Agent", "`Completed` | the same key starts again", func(b *bench) {
		b.running().ev(event.AgentStarted, "", event.Agent{Key: "t1", Type: "subagent"})
		b.ev(event.AgentCompleted, "", event.Agent{Key: "t1"})
		b.ev(event.AgentStarted, "", event.Agent{Key: "t1", Type: "subagent"})
		b.eq("Claude Code", b.agent("t1").Status, AgentRunning)
		b.eq("runs", b.agent("t1").Runs, 2)

		b.ev(event.AgentStarted, "", event.Agent{Key: "th1", Type: "thread"})
		b.ev(event.TurnStarted, "th1", nil)
		b.turnEnd("th1", "completed")
		b.ev(event.TurnStarted, "th1", nil)
		b.eq("Codex", b.agent("th1").Status, AgentRunning)
		b.eq("runs", b.agent("th1").Runs, 2)
	}},
	{"Agent", "any non-final | session `Crashed` or `Closed`", func(b *bench) {
		b.running().ev(event.AgentStarted, "", event.Agent{Key: "t1", Type: "subagent"})
		b.ev(event.AgentStarted, "", event.Agent{Key: "t2", Type: "subagent"})
		b.ev(event.AgentCompleted, "", event.Agent{Key: "t2"})
		b.exit(1)
		b.eq("non-final", b.agent("t1").Status, AgentEnded)
		b.eq("origin", b.agent("t1").StatusOrigin, event.Inferred)
		b.eq("final kept", b.agent("t2").Status, AgentCompleted)

		c := newBench(b.t).running()
		c.ev(event.AgentStarted, "", event.Agent{Key: "t1", Type: "subagent"})
		c.do(Quit{})
		c.exit(0)
		c.eq("closed", c.agent("t1").Status, AgentEnded)
	}},

	// Approval machine
	{"Approval", "(none) | `approval.requested`", func(b *bench) {
		b.running().ask("", "a1", "")
		b.eq("status", b.s.Approvals["a1"].Status, Pending)
		b.eq("queue", strings.Join(b.s.Queue, ","), "a1")
		b.eq("no command", b.kinds(), "")
	}},
	{"Approval", "`Pending` | owner allows (once)", func(b *bench) {
		b.running().ask("", "a1", "")
		b.do(Answer{ID: "a1", Decision: "allow"})
		b.eq("answering", b.s.Approvals["a1"].Status, Answering)
		b.eq("commands", b.kinds(), "answer")
		b.resolve("a1", "allow")
		b.eq("allowed", b.s.Approvals["a1"].Status, Allowed)
	}},
	{"Approval", "`Pending` | owner allows for the session", func(b *bench) {
		b.running().ask("", "a1", "")
		b.do(Answer{ID: "a1", Decision: "acceptForSession"})
		b.eq("decision", b.cmds[0].Decision, "acceptForSession")
		b.resolve("a1", "acceptForSession")
		b.eq("allowed", b.s.Approvals["a1"].Status, Allowed)
	}},
	{"Approval", "`Pending` | owner denies", func(b *bench) {
		b.running().ev(event.ToolStarted, "", event.Tool{ToolID: "tu1", Name: "Bash"})
		b.ask("", "a1", "tu1")
		b.do(Answer{ID: "a1", Decision: "deny"})
		b.resolve("a1", "deny")
		b.eq("denied", b.s.Approvals["a1"].Status, Denied)
	}},
	{"Approval", "`Pending` | owner cancels the turn", func(b *bench) {
		b.running().ask("", "a1", "")
		b.do(Interrupt{})
		b.eq("answering", b.s.Approvals["a1"].Status, Answering)
		b.resolve("a1", "cancel")
		b.eq("denied", b.s.Approvals["a1"].Status, Denied)
		b.eq("turn", b.s.Turn.Phase, Interrupting)
	}},
	{"Approval", "`Pending` | the request cannot be parsed", func(b *bench) {
		b.running().ev(event.ApprovalRequest, "", event.Approval{ID: "a1", Problem: "no tool name"})
		b.eq("status", b.s.Approvals["a1"].Status, Denied)
		b.eq("commands", b.kinds(), "answer")
		b.eq("decision", b.cmds[0].Decision, "deny")
		b.eq("turn not waiting", b.s.Turn.Phase, Running)
	}},
	{"Approval", "`Answering` | writing the answer fails", func(b *bench) {
		b.running().ask("", "a1", "")
		b.do(Answer{ID: "a1", Decision: "allow"})
		b.do(AnswerFailed{ID: "a1", Message: "pipe closed"})
		b.eq("approval", b.s.Approvals["a1"].Status, Expired)
		b.eq("session", b.s.Session.Phase, Crashed)
	}},
	{"Approval", "`Pending` | turn ends, CLI resolves it, or session ends", func(b *bench) {
		b.running().ask("", "a1", "")
		b.turnEnd("", "completed")
		b.eq("turn ends", b.s.Approvals["a1"].Status, Expired)

		c := newBench(b.t).running()
		c.ask("", "a1", "")
		c.resolve("a1", "resolved")
		c.eq("CLI resolves", c.s.Approvals["a1"].Status, Expired)
		c.eq("turn", c.s.Turn.Phase, Running)

		d := newBench(b.t).running()
		d.ask("", "a1", "")
		d.exit(1)
		d.eq("session ends", d.s.Approvals["a1"].Status, Expired)
	}},

	// Tool call machine
	{"Tool call", "(none) | `tool.started`", func(b *bench) {
		b.running().ev(event.ToolStarted, "", event.Tool{ToolID: "tu1", Name: "Bash"})
		b.eq("status", b.s.Tools["tu1"].Status, ToolStarted)
	}},
	{"Tool call", "`Started` | output or progress", func(b *bench) {
		b.running().ev(event.ToolStarted, "", event.Tool{ToolID: "tu1", Name: "Bash"})
		b.ev(event.ToolProgress, "", event.Tool{ToolID: "tu1", Output: "line\n"})
		b.eq("status", b.s.Tools["tu1"].Status, ToolRunning)
		b.eq("output", b.s.Tools["tu1"].Output, "line\n")
	}},
	{"Tool call", "`Started`, `Running` | `tool.completed` without error", func(b *bench) {
		b.running().ev(event.ToolStarted, "", event.Tool{ToolID: "tu1", Name: "Bash"})
		b.ev(event.ToolCompleted, "", event.Tool{ToolID: "tu1", Output: "ok"})
		b.eq("from Started", b.s.Tools["tu1"].Status, ToolCompleted)
		b.ev(event.ToolStarted, "", event.Tool{ToolID: "tu2", Name: "Bash"})
		b.ev(event.ToolProgress, "", event.Tool{ToolID: "tu2"})
		b.ev(event.ToolCompleted, "", event.Tool{ToolID: "tu2"})
		b.eq("from Running", b.s.Tools["tu2"].Status, ToolCompleted)
	}},
	{"Tool call", "`Started`, `Running` | `tool.completed` with an error", func(b *bench) {
		b.running().ev(event.ToolStarted, "", event.Tool{ToolID: "tu1", Name: "Bash"})
		b.ev(event.ToolCompleted, "", event.Tool{ToolID: "tu1", IsError: true})
		b.eq("status", b.s.Tools["tu1"].Status, ToolFailed)
	}},
	{"Tool call", "`Started` | its approval denied", func(b *bench) {
		b.running().ev(event.ToolStarted, "", event.Tool{ToolID: "tu1", Name: "Bash"})
		b.ask("", "a1", "tu1")
		b.resolve("a1", "decline")
		b.eq("status", b.s.Tools["tu1"].Status, ToolDenied)
	}},
	{"Tool call", "`Started`, `Running` | turn or agent ends first", func(b *bench) {
		b.running().ev(event.ToolStarted, "", event.Tool{ToolID: "tu1", Name: "Bash"})
		b.turnEnd("", "interrupted")
		b.eq("turn ends", b.s.Tools["tu1"].Status, ToolUnfinished)
		b.eq("origin", b.s.Tools["tu1"].Origin, event.Inferred)

		b.ev(event.AgentStarted, "", event.Agent{Key: "t1", Type: "subagent"})
		b.ev(event.ToolStarted, "t1", event.Tool{ToolID: "tu2", Name: "Read"})
		b.ev(event.AgentCompleted, "", event.Agent{Key: "t1"})
		b.eq("agent ends", b.s.Tools["tu2"].Status, ToolUnfinished)
	}},
}

var invalids = []machineCase{
	{"Startup", "Confirm while `Detecting` is ignored", func(b *bench) {
		b.do(Launch{})
		b.do(Confirm{Provider: "claude-code", Project: project})
		b.eq("phase", b.s.Startup.Phase, Detecting)
		b.eq("commands", b.kinds(), "")
	}},
	{"Startup", "a second Confirm while `Validating` or `Starting` is ignored", func(b *bench) {
		b.validating().do(Confirm{Provider: "codex", Project: `C:\other`})
		b.eq("validating", b.s.Startup.Phase, Validating)
		b.eq("answer unchanged", b.s.Startup.Project, project)
		b.eq("commands", b.kinds(), "")
		c := newBench(b.t).starting()
		c.do(Confirm{Provider: "codex", Project: `C:\other`})
		c.eq("starting", c.s.Startup.Phase, Starting)
		c.eq("commands", c.kinds(), "")
	}},
	{"Startup", "choosing a provider shown \"not installed\" is impossible", func(b *bench) {
		b.do(Launch{})
		b.do(DetectionDone{Installed: []string{"claude-code"}})
		b.do(Confirm{Provider: "codex", Project: project})
		b.eq("phase", b.s.Startup.Phase, Asking)
		b.eq("field", b.s.Startup.Field, "provider")
		b.eq("commands", b.kinds(), "")
		// With neither installed, Start is disabled: nothing can be chosen.
		c := newBench(b.t)
		c.do(Launch{})
		c.do(DetectionDone{})
		c.do(Confirm{Provider: "", Project: project})
		c.eq("none: phase", c.s.Startup.Phase, Asking)
		c.eq("none: commands", c.kinds(), "")
	}},
	{"Startup", "No transition leads from `Ready` back to `Asking`", func(b *bench) {
		b.ready().do(Back{})
		b.eq("phase", b.s.Startup.Phase, StartReady)
		b.eq("screen", b.s.Screen, ScreenDashboard)
	}},

	{"Session", "a user message, approval answer or interrupt while not `Ready` is refused", func(b *bench) {
		b.handshaking().ask("", "a1", "")
		b.do(Send{Text: "early"})
		b.eq("send", b.kinds(), "")
		b.eq("draft kept", b.s.Turn.Draft, "early")
		b.eq("notice", b.s.Turn.Notice != "", true)
		b.do(Answer{ID: "a1", Decision: "allow"})
		b.eq("answer", b.kinds(), "")
		b.eq("still pending", b.s.Approvals["a1"].Status, Pending)
		b.do(Interrupt{})
		b.eq("interrupt", b.kinds(), "")
	}},
	{"Session", "events read after `Closed` are logged, not applied", func(b *bench) {
		b.ready().do(Quit{})
		b.exit(0)
		n := len(b.s.Log)
		b.ev(event.MessageCompleted, "", event.Completed{MessageID: "m9", Text: "late"})
		b.eq("conversation", len(b.s.Conversation), 0)
		b.eq("logged", len(b.s.Log), n+1)
		b.eq("noted", b.noted("after the session closed"), true)
	}},
	{"Session", "`Crashed` is final for the process", func(b *bench) {
		b.ready().exit(1)
		b.ev(event.SessionReady, "", event.Ready{})
		b.eq("ready ignored", b.s.Session.Phase, Crashed)
		b.ev(event.SessionEnded, "", event.Ended{})
		b.eq("second exit ignored", b.s.Session.Phase, Crashed)
	}},
	{"Session", "A crash before `Ready` is the startup's `StartFailed`", func(b *bench) {
		b.handshaking().exit(1)
		b.eq("startup", b.s.Startup.Phase, StartFailed)
		b.do(Resume{})
		b.eq("no resume", b.kinds(), "")
	}},
	{"Session", "After `Ready`, the dashboard shows the exit code and stderr tail", func(b *bench) {
		b.ready().ev(event.SessionStarted, "", event.Started{SessionID: "s1", Cwd: project})
		b.ev(event.Stderr, "", event.Line{Text: "panic: boom"})
		b.exit(2)
		b.eq("crashed", b.s.Session.Phase, Crashed)
		b.eq("exit code", b.s.Session.ExitCode, 2)
		b.eq("stderr tail", strings.Join(b.s.Session.StderrTail, "|"), "panic: boom")
		b.eq("startup stays Ready", b.s.Startup.Phase, StartReady)
		b.do(Resume{})
		b.eq("resume", b.kinds(), "start-session")
		b.eq("same CLI session", b.cmds[0].ID, "s1")
		b.eq("new machine", b.s.Session.Phase, Spawning)

		c := newBench(b.t).ready()
		c.exit(2)
		c.do(Quit{})
		c.eq("quit", c.kinds(), "exit")
	}},

	{"Turn", "sending while a turn is not `Idle` is refused in the MVP", func(b *bench) {
		b.running().do(Send{Text: "second"})
		b.eq("commands", b.kinds(), "")
		b.eq("hint", b.s.Turn.Notice, "turn running: wait, or interrupt")
		b.eq("draft kept", b.s.Turn.Draft, "second")
		b.eq("turn", b.s.Turn.Phase, Running)
	}},
	{"Turn", "an interrupt while `Idle` or `Submitted`-without-events is a no-op with a notice", func(b *bench) {
		b.ready().do(Interrupt{})
		b.eq("idle", b.kinds(), "")
		b.eq("notice", b.s.Turn.Notice, "nothing to interrupt")
		c := newBench(b.t).submitted()
		c.do(Interrupt{})
		c.eq("submitted", c.kinds(), "")
		c.eq("turn", c.s.Turn.Phase, Submitted)
	}},
	{"Turn", "a turn-end event while `Idle` is applied to telemetry and logged as \"unexpected turn end\"", func(b *bench) {
		b.ready().ev(event.Telemetry, "", event.Metrics{Items: []event.Metric{{Name: "duration.ms", Value: 900, Documented: true}}})
		b.turnEnd("", "completed")
		b.eq("telemetry applied", b.s.Metrics["duration.ms"].V, 900.0)
		b.eq("noted", b.noted("unexpected turn end"), true)
		b.eq("outcome untouched", b.s.Turn.Outcome, "")
	}},
	{"Turn", "Background agents may keep running after the turn is `Idle`", func(b *bench) {
		b.running().ev(event.AgentStarted, "", event.Agent{Key: "bg", Type: "subagent", Backgrounded: true})
		b.turnEnd("", "completed")
		b.eq("turn", b.s.Turn.Phase, Idle)
		b.eq("agent", b.agent("bg").Status, AgentRunning)
		b.ev(event.AgentProgress, "", event.Agent{Key: "bg", ToolUses: 2})
		b.eq("progress applied", b.agent("bg").ToolUses, 2)
	}},

	{"Agent", "a progress or tool event for an agent in a final state is logged", func(b *bench) {
		b.running().ev(event.AgentStarted, "", event.Agent{Key: "t1", Type: "subagent"})
		b.ev(event.AgentCompleted, "", event.Agent{Key: "t1"})
		b.ev(event.AgentProgress, "", event.Agent{Key: "t1", ToolUses: 9})
		b.ev(event.ToolStarted, "t1", event.Tool{ToolID: "tu9", Name: "Read"})
		a := b.agent("t1")
		b.eq("status", a.Status, AgentCompleted)
		b.eq("history", strings.Join(a.History, "|"), "progress after end|tool after end: Read")
	}},
	{"Agent", "a terminal event for an unknown key creates the node directly in that state", func(b *bench) {
		b.running().ev(event.AgentFailed, "", event.Agent{Key: "ghost", Status: "failed"})
		b.eq("status", b.agent("ghost").Status, AgentFailed)
		b.eq("origin", b.agent("ghost").StatusOrigin, event.Inferred)
	}},

	{"Approval", "a second answer to the same id is ignored and logged", func(b *bench) {
		b.running().ask("", "a1", "")
		b.do(Answer{ID: "a1", Decision: "allow"})
		b.do(Answer{ID: "a1", Decision: "deny"})
		b.eq("second answer", b.kinds(), "")
		b.resolve("a1", "allow")
		b.resolve("a1", "deny")
		b.eq("status", b.s.Approvals["a1"].Status, Allowed)
		b.eq("noted", b.noted("second answer ignored"), true)
	}},
	{"Approval", "an answer for an unknown id shows an error", func(b *bench) {
		b.running().do(Answer{ID: "zz", Decision: "allow"})
		b.eq("commands", b.kinds(), "")
		b.resolve("zz", "allow")
		b.eq("errors", len(b.s.Errors), 2)
	}},
	{"Approval", "nothing is ever answered without the owner's action except unparseable requests", func(b *bench) {
		b.running().ask("", "a1", "")
		b.ask("t1", "a2", "")
		var all []Command
		all = append(all, b.ev(event.MessageDelta, "", event.Delta{MessageID: "m1", Block: "text", Text: "x"})...)
		all = append(all, b.ev(event.Telemetry, "", event.Metrics{})...)
		all = append(all, b.do(Tick{At: time.Date(2026, 10, 17, 0, 0, 0, 0, time.UTC)})...)
		all = append(all, b.turnEnd("", "completed")...)
		for _, c := range all {
			if c.Kind == CmdAnswer {
				b.t.Errorf("answered without the owner: %+v", c)
			}
		}
		b.ev(event.ApprovalRequest, "", event.Approval{ID: "a3", Problem: "unreadable"})
		b.eq("unparseable", b.cmds[0].Decision, "deny")
	}},
	{"Approval", "\"Allow and remember\" options that persist a rule", func(b *bench) {
		b.running().ask("", "a1", "")
		b.do(Answer{ID: "a1", Decision: "acceptWithExecpolicyAmendment"})
		b.eq("commands", b.kinds(), "")
		b.eq("status", b.s.Approvals["a1"].Status, Pending)
	}},

	{"Tool call", "a completion for an unknown tool id creates the call in its final state", func(b *bench) {
		b.running().ev(event.ToolCompleted, "", event.Tool{ToolID: "tu7", IsError: true})
		b.eq("status", b.s.Tools["tu7"].Status, ToolFailed)
		b.eq("origin", b.s.Tools["tu7"].Origin, event.Inferred)
	}},
	{"Tool call", "a second completion is logged and ignored", func(b *bench) {
		b.running().ev(event.ToolStarted, "", event.Tool{ToolID: "tu1", Name: "Bash"})
		b.ev(event.ToolCompleted, "", event.Tool{ToolID: "tu1", Output: "first"})
		b.ev(event.ToolCompleted, "", event.Tool{ToolID: "tu1", Output: "second", IsError: true})
		t := b.s.Tools["tu1"]
		b.eq("status", t.Status, ToolCompleted)
		b.eq("output", t.Output, "first")
		b.eq("history", strings.Join(t.History, "|"), "second completion ignored")
	}},
}

// docMachines parses the State machines section: per machine, its table rows
// as "From | Trigger" and its Invalid paragraph's clauses.
func docMachines(t *testing.T) (rows, clauses map[string][]string) {
	t.Helper()
	raw, err := os.ReadFile("../../doc/events-and-state.md")
	if err != nil {
		t.Fatal(err)
	}
	text := strings.ReplaceAll(string(raw), "\r\n", "\n")
	start := strings.Index(text, "\n## State machines")
	end := strings.Index(text[start+1:], "\n## ")
	section := text[start : start+1+end]
	rows, clauses = map[string][]string{}, map[string][]string{}
	for _, part := range strings.Split(section, "\n### ")[1:] {
		name := strings.TrimSpace(strings.SplitN(part, "\n", 2)[0])
		name = strings.TrimSuffix(strings.SplitN(name, " (", 2)[0], " machine")
		var invalid []string
		inInvalid := false
		for _, line := range strings.Split(part, "\n") {
			switch {
			case strings.HasPrefix(line, "| ") && !strings.HasPrefix(line, "| From") && !strings.HasPrefix(line, "| - "):
				cells := strings.Split(line, " | ")
				rows[name] = append(rows[name], strings.TrimPrefix(cells[0], "| ")+" | "+cells[1])
			case strings.HasPrefix(line, "Invalid:"):
				inInvalid = true
				invalid = append(invalid, strings.TrimPrefix(line, "Invalid:"))
			case inInvalid && strings.TrimSpace(line) == "":
				inInvalid = false
			case inInvalid:
				invalid = append(invalid, line)
			}
		}
		clauses[name] = splitClauses(strings.Join(invalid, " "))
	}
	return rows, clauses
}

// splitClauses splits prose on "; " and ". " outside parentheses and brackets.
func splitClauses(p string) []string {
	var out []string
	depth, from := 0, 0
	for i := 0; i < len(p); i++ {
		switch p[i] {
		case '(', '[':
			depth++
		case ')', ']':
			depth--
		case ';', '.':
			if depth == 0 && (i+1 == len(p) || p[i+1] == ' ') {
				if c := strings.TrimSpace(p[from:i]); c != "" {
					out = append(out, c)
				}
				from = i + 1
			}
		}
	}
	if c := strings.TrimSpace(p[from:]); c != "" {
		out = append(out, c)
	}
	return out
}

func TestStateMachines(t *testing.T) {
	rows, clauses := docMachines(t)
	machines := []string{"Startup", "Session", "Turn", "Agent", "Approval", "Tool call"}
	if len(rows) != len(machines) {
		t.Fatalf("doc has machines %v, want %v", keys(rows), machines)
	}
	cover := func(kind string, doc map[string][]string, cases []machineCase, match func(docItem, caseKey string) bool) {
		total := 0
		for _, m := range machines {
			matched := map[string]int{}
			n := 0
			for _, c := range cases {
				if c.machine != m {
					continue
				}
				n++
				hits := 0
				for _, d := range doc[m] {
					if match(d, c.row) {
						hits++
						matched[d]++
					}
				}
				if hits != 1 {
					t.Errorf("%s %s case %q matches %d doc items, want 1", m, kind, c.row, hits)
				}
			}
			for _, d := range doc[m] {
				if matched[d] != 1 {
					t.Errorf("%s %s %q has %d cases, want 1", m, kind, d, matched[d])
				}
			}
			if n != len(doc[m]) {
				t.Errorf("%s: %d %s cases, %d in the doc", m, n, kind, len(doc[m]))
			}
			total += n
		}
		t.Logf("%s: %d cases", kind, total)
	}
	// A case names a row by a prefix of "From | Trigger"; a row equal to the
	// prefix wins over longer rows that start with it.
	exact := map[string]bool{}
	for _, list := range rows {
		for _, d := range list {
			exact[d] = true
		}
	}
	cover("transition", rows, transitions, func(d, k string) bool {
		return d == k || (!exact[k] && strings.HasPrefix(d, k))
	})
	cover("invalid", clauses, invalids, strings.Contains)

	for _, list := range [][]machineCase{transitions, invalids} {
		for _, c := range list {
			t.Run(c.machine+": "+c.row, func(t *testing.T) { c.run(newBench(t)) })
		}
	}
}

func keys(m map[string][]string) []string {
	var k []string
	for n := range m {
		k = append(k, n)
	}
	return k
}
