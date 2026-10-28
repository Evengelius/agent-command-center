package app

import (
	"encoding/json"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/Evengelius/agent-command-center/internal/event"
	"github.com/Evengelius/agent-command-center/internal/input"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// Plan 63: the ping (ui-design.md, The ping), on a scripted Claude Code
// session.

// pingRig is a dashboard on a scripted session in the given Windows
// Terminal profile (WT_PROFILE_ID), its clock moved by the test; it counts
// the BELs written and keeps the delays the ping's timers were asked for,
// whose message the test delivers itself.
type pingRig struct {
	inputRig
	now    *time.Time
	bells  *int
	timers *[]time.Duration
}

func newPingRig(t *testing.T, profile string) pingRig {
	t.Helper()
	return newPingRigFor(t, "claude-code", profile)
}

// newPingRigFor is newPingRig on the given CLI's scripted session.
func newPingRigFor(t *testing.T, id, profile string) pingRig {
	t.Helper()
	r := scriptRig(t, id)
	now, bells, timers := testNow, 0, []time.Duration{}
	r.h.m.opt.Getenv = profileEnv(profile)
	r.h.m.opt.Now = func() time.Time { return now }
	r.h.m.opt.After = func(d time.Duration, msg tea.Msg) tea.Cmd {
		if _, ok := msg.(pingMsg); ok {
			timers = append(timers, d)
			return func() tea.Msg { return nil } // asked for, never delivered: the test delivers it
		}
		return nil
	}
	r.h.raw = func(text string) {
		if text == bel {
			bells++
		}
	}
	return pingRig{r, &now, &bells, &timers}
}

// profileEnv is an environment whose WT_PROFILE_ID is profile, "" for none.
func profileEnv(profile string) func(string) string {
	return func(k string) string {
		if k == "WT_PROFILE_ID" {
			return profile
		}
		return ""
	}
}

// push gives the dashboard the session's events and lets it take them in.
func (r pingRig) push(evs ...event.Event) {
	for _, ev := range evs {
		r.emit(ev.Kind, ev.Payload)
	}
	r.h.settle()
}

func (r pingRig) title() string { return r.h.m.View().WindowTitle }

func approvalEv(id string) event.Event {
	return evOf(event.ApprovalRequest, event.Approval{ID: id, Tool: "Bash", ToolID: "t-" + id, Input: json.RawMessage(`{"command":"ls"}`)})
}

func questionEv(id string) event.Event {
	return evOf(event.Question, event.Approval{ID: id, Tool: "AskUserQuestion", Input: json.RawMessage(`{"questions":[{"question":"Which colour?","header":"Colour","options":[{"label":"Red","description":"warm"},{"label":"Blue","description":"cold"}],"multiSelect":false}]}`)})
}

func resolvedEv(id string) event.Event {
	return evOf(event.ApprovalResolved, event.Approval{ID: id, Decision: "allow"})
}

func turnEndEv(outcome string) event.Event {
	return evOf(event.TurnCompleted, event.TurnEnd{Outcome: outcome})
}

var crashEv = evOf(event.SessionEnded, event.Ended{Code: 1, HasCode: true})

// TestPingNeeds is criterion 63.1: the window without the focus, in ACC's
// profile, one BEL for an approval, a question, a turn completed, a turn
// failed and the session's end; none for streamed text, a tool, a
// subagent's end or an interrupted turn. Focus reports are asked for on
// the dashboard (the home screen's: TestHomeBanner).
func TestPingNeeds(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := newPingRig(t, input.AccProfile)
		if !r.h.m.View().ReportFocus {
			t.Fatal("the dashboard does not ask for the terminal's focus reports")
		}
		r.h.send(tea.BlurMsg{})
		for _, c := range []struct {
			what  string
			evs   []event.Event
			bells int
			title string
		}{
			{"an approval", []event.Event{evOf(event.TurnStarted, nil), approvalEv("a1")}, 1, needTitles[needApproval]},
			{"streamed text, the approval rung", []event.Event{evOf(event.MessageDelta, event.Delta{MessageID: "m1", Block: "text", Text: "Listing."})}, 0, needTitles[needApproval]},
			{"a tool, the approval answered", []event.Event{resolvedEv("a1"), evOf(event.ToolStarted, event.Tool{ToolID: "t-a1", Name: "Bash"}),
				evOf(event.ToolCompleted, event.Tool{ToolID: "t-a1", Name: "Bash", Output: "a.txt"})}, 0, windowTitle},
			{"a subagent's end", []event.Event{evOf(event.AgentStarted, event.Agent{Key: "k1", Label: "explorer", Type: "subagent", Description: "scan"}),
				evOf(event.AgentCompleted, event.Agent{Key: "k1", Status: "completed", Summary: "done"})}, 0, windowTitle},
			{"a question", []event.Event{questionEv("q1")}, 1, needTitles[needQuestion]},
			{"a turn completed", []event.Event{resolvedEv("q1"), turnEndEv("completed")}, 1, needTitles[needTurn]},
			{"a turn failed", []event.Event{evOf(event.TurnStarted, nil), turnEndEv("failed")}, 1, needTitles[needFailed]},
			{"a turn interrupted", []event.Event{evOf(event.TurnStarted, nil), turnEndEv("interrupted")}, 0, windowTitle},
			{"the session's end", []event.Event{crashEv}, 1, needTitles[needCrash]},
		} {
			*r.now = r.now.Add(time.Minute) // past the bell's limit
			was := *r.bells
			r.push(c.evs...)
			if got := *r.bells - was; got != c.bells {
				t.Errorf("%s: %d BELs, want %d", c.what, got, c.bells)
			}
			if got := r.title(); got != c.title {
				t.Errorf("%s: title %q, want %q", c.what, got, c.title)
			}
		}
		if s := r.h.m.State(); s.Session.Phase != state.Crashed || len(*r.timers) != 0 {
			t.Errorf("session %q, timers %v: want crashed, none", s.Session.Phase, *r.timers)
		}
	})
}

// TestPingFocused is criterion 63.1's last part: the window has the focus
// until Terminal says otherwise, and with it nothing rings or changes the
// title, then or once the focus is gone; an approval rung once does not
// ring again after the focus came and went.
func TestPingFocused(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := newPingRig(t, input.AccProfile)
		r.push(evOf(event.TurnStarted, nil), approvalEv("a1"))
		r.push(questionEv("q1"))
		r.push(resolvedEv("a1"), resolvedEv("q1"), turnEndEv("completed"))
		r.push(evOf(event.TurnStarted, nil), approvalEv("a2"))
		r.push(turnEndEv("failed"))
		r.push(evOf(event.TurnStarted, nil), approvalEv("a3"))
		if *r.bells != 0 || r.title() != windowTitle {
			t.Fatalf("with the focus: %d BELs, title %q", *r.bells, r.title())
		}
		*r.now = r.now.Add(time.Minute)
		r.h.send(tea.BlurMsg{})
		r.push(evOf(event.MessageDelta, event.Delta{MessageID: "m1", Block: "text", Text: "Working."}))
		if *r.bells != 0 || r.title() != windowTitle {
			t.Errorf("the focus gone after: %d BELs, title %q", *r.bells, r.title())
		}
		r.push(crashEv)
		r.h.send(tea.FocusMsg{})
		if *r.bells != 1 || r.title() != windowTitle {
			t.Errorf("the session's end, then the focus: %d BELs, title %q; want 1, %q", *r.bells, r.title(), windowTitle)
		}
	})
	synctest.Test(t, func(t *testing.T) {
		r := newPingRig(t, input.AccProfile)
		r.h.send(tea.BlurMsg{})
		r.push(evOf(event.TurnStarted, nil), approvalEv("a1"))
		r.h.send(tea.FocusMsg{})
		*r.now = r.now.Add(time.Minute)
		r.h.send(tea.BlurMsg{})
		r.push(evOf(event.MessageDelta, event.Delta{MessageID: "m1", Block: "text", Text: "Waiting."}))
		if *r.bells != 1 || r.title() != needTitles[needApproval] {
			t.Errorf("an approval rung, the focus there and gone: %d BELs, title %q; want 1, %q", *r.bells, r.title(), needTitles[needApproval])
		}
	})
}

// TestPingBell is criterion 63.2's first part: a BEL in ACC's profile only,
// its id compared without case; elsewhere none, the title changing all the
// same.
func TestPingBell(t *testing.T) {
	for _, c := range []struct {
		profile string
		bells   int
	}{
		{"", 0}, // not in Windows Terminal
		{"{0caa0dad-35be-5f56-a8ff-afceeeaa6101}", 0}, // another profile's
		{input.AccProfile, 1},
		{strings.ToUpper(input.AccProfile), 1},
	} {
		synctest.Test(t, func(t *testing.T) {
			r := newPingRig(t, c.profile)
			r.h.send(tea.BlurMsg{})
			r.push(evOf(event.TurnStarted, nil), approvalEv("a1"))
			if *r.bells != c.bells || len(*r.timers) != 0 || r.title() != needTitles[needApproval] {
				t.Errorf("profile %q: %d BELs, timers %v, title %q; want %d, none, %q", c.profile, *r.bells, *r.timers, r.title(), c.bells, needTitles[needApproval])
			}
		})
	}
}

// TestPingLimit is criterion 63.2's limit: a need within 5 s of a BEL rings
// when the 5 s are up, two there sharing that BEL; one answered, or the
// focus back, before the mark rings none; 5 s after a BEL, at once again.
func TestPingLimit(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := newPingRig(t, input.AccProfile)
		r.h.send(tea.BlurMsg{})
		check := func(when string, bells int, timers ...time.Duration) {
			t.Helper()
			if *r.bells != bells || !slices.Equal(*r.timers, timers) {
				t.Fatalf("%s: %d BELs, timers %v; want %d, %v", when, *r.bells, *r.timers, bells, timers)
			}
		}
		after := func(d time.Duration) { *r.now = r.now.Add(d) }
		r.push(evOf(event.TurnStarted, nil), turnEndEv("completed"))
		check("a turn's end", 1)
		after(2 * time.Second)
		r.push(evOf(event.TurnStarted, nil), questionEv("q1"))
		check("a question 2 s after", 1, 3*time.Second)
		after(time.Second)
		r.push(approvalEv("a1"))
		check("an approval 1 s later", 1, 3*time.Second)
		after(2 * time.Second)
		r.h.send(pingMsg{})
		check("the 5 s up", 2, 3*time.Second)
		if r.title() != needTitles[needApproval] {
			t.Errorf("the 5 s up: title %q", r.title())
		}
		after(time.Second)
		r.push(approvalEv("a2"))
		check("another approval 1 s after", 2, 3*time.Second, 4*time.Second)
		r.h.send(actionMsg{state.Answer{ID: "a2", Decision: "allow"}})
		after(4 * time.Second)
		r.h.send(pingMsg{})
		check("that approval answered before the mark", 2, 3*time.Second, 4*time.Second)
		r.push(approvalEv("a3"))
		check("5 s after the last BEL", 3, 3*time.Second, 4*time.Second)
		after(time.Second)
		r.push(approvalEv("a4"))
		check("an approval 1 s after", 3, 3*time.Second, 4*time.Second, 4*time.Second)
		r.h.send(tea.FocusMsg{})
		r.h.send(tea.BlurMsg{})
		after(4 * time.Second)
		r.h.send(pingMsg{})
		check("the focus back before the mark", 3, 3*time.Second, 4*time.Second, 4*time.Second)
	})
}

// TestPingTitle is criterion 63.3: each reason's title while it waits, the
// most urgent when several do, the dashboard's name once answered, with the
// focus back for a turn's end or the session's, and once another turn
// runs; outside ACC's profile, as no BEL changes it.
func TestPingTitle(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := newPingRig(t, "")
		title := func(when, want string) {
			t.Helper()
			if got := r.title(); got != want {
				t.Errorf("%s: title %q, want %q", when, got, want)
			}
		}
		title("at the start", "Agent Command Center")
		r.h.send(tea.BlurMsg{})
		r.push(evOf(event.TurnStarted, nil), questionEv("q1"))
		title("a question", "◆ Question waiting · Agent Command Center")
		r.push(approvalEv("a1"))
		title("an approval with it", "◆ Approval waiting · Agent Command Center")
		r.h.send(tea.FocusMsg{})
		title("the focus back", "◆ Approval waiting · Agent Command Center")
		r.h.send(actionMsg{state.Answer{ID: "a1", Decision: "allow"}})
		title("the approval answered", "◆ Question waiting · Agent Command Center")
		r.h.send(actionMsg{state.Answer{ID: "q1", Decision: "allow", Answers: map[string]string{"Which colour?": "Red"}}})
		title("the question answered", "Agent Command Center")
		r.h.send(tea.BlurMsg{})
		r.push(turnEndEv("completed"))
		title("the turn completed", "✓ Turn done · Agent Command Center")
		r.h.send(tea.FocusMsg{})
		title("the focus back after it", "Agent Command Center")
		r.h.send(tea.BlurMsg{})
		r.push(evOf(event.TurnStarted, nil), turnEndEv("failed"))
		title("a turn failed", "✗ Turn failed · Agent Command Center")
		r.push(evOf(event.TurnStarted, nil))
		title("another turn running", "Agent Command Center")
		r.push(crashEv)
		title("the session's end, its turn failed", "✗ Session ended · Agent Command Center")
		r.h.send(tea.FocusMsg{})
		title("the focus back at the end", "Agent Command Center")
		if *r.bells != 0 {
			t.Errorf("%d BELs outside ACC's profile", *r.bells)
		}
	})
}

// TestPingTitleCleared is criterion 63.3's end: Bubble Tea clears the title
// a view set when the program ends, so a reason never outlives the
// dashboard and the tab's own title comes back; its close does the same
// after a recovered panic (cursedRenderer.close).
func TestPingTitleCleared(t *testing.T) {
	set := "\x1b]2;" + needTitles[needApproval] + "\a"
	out := runTea(t, "TTY_FORCE=1 WT_SESSION=1", titleModel{needTitles[needApproval]})
	i := strings.LastIndex(out, set)
	if i < 0 || !strings.Contains(out[i+len(set):], "\x1b]2;\a") {
		t.Fatalf("the title not set, then cleared:\n%q", out)
	}
}

// titleModel shows one frame under a title and quits.
type titleModel struct{ title string }

func (m titleModel) Init() tea.Cmd                       { return tea.Quit }
func (m titleModel) Update(tea.Msg) (tea.Model, tea.Cmd) { return m, nil }
func (m titleModel) View() tea.View {
	v := tea.NewView("a frame")
	v.AltScreen, v.WindowTitle = true, m.title
	return v
}

// TestPingMock is design D1's last line: a mock session pings the same way.
// The replay comes all at once here: its first need rings, those after it
// come within its 5 s and wait for a timer the harness never fires.
func TestPingMock(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newHarness(t, Options{Config: Config{Mock: filepath.Join("..", "..", "testdata", "mock", "claude-subagent-approval.jsonl")}, Getenv: profileEnv(input.AccProfile)})
		bells := 0
		h.raw = func(text string) {
			if text == bel {
				bells++
			}
		}
		h.deliver(tea.BlurMsg{}) // before the replay's first event
		h.settle()
		h.send(tea.WindowSizeMsg{Width: 120, Height: 40})
		s := h.m.State()
		if len(s.Stages) != 2 || s.Turn.Phase != state.Idle {
			t.Fatalf("%d turns, the turn %q: want the replay's 2, idle", len(s.Stages), s.Turn.Phase)
		}
		if got := h.m.View().WindowTitle; bells != 1 || got != needTitles[needTurn] {
			t.Errorf("%d BELs, title %q; want 1, %q", bells, got, needTitles[needTurn])
		}
	})
}
