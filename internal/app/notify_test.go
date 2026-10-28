package app

import (
	"errors"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/Evengelius/agent-command-center/internal/event"
	"github.com/Evengelius/agent-command-center/internal/input"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// Plan 64: Windows' notifications for the ping (ui-design.md, The ping), on
// a fake notifier.

// fakeNotifier is Windows' notification list as the dashboard sees it: the
// one notification up, and every call in the order it came.
type fakeNotifier struct {
	mu    sync.Mutex
	calls []string // "show <title> | <text>" or "clear"
	up    string   // the notification up, "" none
	err   error    // Show's failure when set
}

func (f *fakeNotifier) show(title, text string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, "show "+title+" | "+text)
	if f.err != nil {
		return f.err
	}
	f.up = title + " | " + text // one tag: a newer replaces the older
	return nil
}

func (f *fakeNotifier) clear() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, "clear")
	f.up = ""
	return nil
}

func (f *fakeNotifier) state() (calls []string, up string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.calls), f.up
}

func withNotifier(r pingRig) *fakeNotifier {
	f := &fakeNotifier{}
	r.h.m.opt.Notify, r.h.m.opt.Unnotify = f.show, f.clear
	return f
}

const (
	approvalShown = "show Approval required | Claude Code is waiting for your approval."
	questionShown = "show Question waiting | Claude Code has a question for you."
	turnShown     = "show Turn completed | Claude Code has finished its task."
)

// TestNotifyWhen is criterion 64.2: one notification at each ring, in ACC's
// profile with its BEL and outside it alone; none while the window has the
// focus; none for what the ping does not ring for; the 5 s limit shared, a
// need within it notified when the 5 s are up.
func TestNotifyWhen(t *testing.T) {
	for _, profile := range []string{input.AccProfile, ""} {
		synctest.Test(t, func(t *testing.T) {
			r := newPingRig(t, profile)
			f := withNotifier(r)
			r.push(evOf(event.TurnStarted, nil), approvalEv("a1")) // the window has the focus
			r.push(resolvedEv("a1"), turnEndEv("completed"))
			if calls, _ := f.state(); len(calls) != 0 {
				t.Fatalf("profile %q, with the focus: %v", profile, calls)
			}
			r.h.send(tea.BlurMsg{})
			*r.now = r.now.Add(time.Minute)
			r.push(evOf(event.TurnStarted, nil), evOf(event.MessageDelta, event.Delta{MessageID: "m1", Block: "text", Text: "Listing."}),
				evOf(event.ToolStarted, event.Tool{ToolID: "t1", Name: "Bash"}), evOf(event.ToolCompleted, event.Tool{ToolID: "t1", Name: "Bash"}), turnEndEv("interrupted"))
			if calls, _ := f.state(); len(calls) != 0 {
				t.Fatalf("profile %q, text, a tool, an interrupted turn: %v", profile, calls)
			}
			r.push(evOf(event.TurnStarted, nil), turnEndEv("completed"))
			bells := 0
			if profile != "" {
				bells = 1
			}
			if calls, up := f.state(); !slices.Equal(calls, []string{turnShown}) || up == "" || *r.bells != bells {
				t.Fatalf("profile %q, a turn's end: calls %v, up %q, %d BELs", profile, calls, up, *r.bells)
			}
			*r.now = r.now.Add(2 * time.Second)
			r.push(evOf(event.TurnStarted, nil), questionEv("q1"))
			if calls, _ := f.state(); len(calls) != 1 || !slices.Equal(*r.timers, []time.Duration{3 * time.Second}) {
				t.Fatalf("profile %q, a question 2 s after: calls %v, timers %v", profile, calls, *r.timers)
			}
			*r.now = r.now.Add(3 * time.Second)
			r.h.send(pingMsg{})
			if calls, up := f.state(); !slices.Equal(calls, []string{turnShown, questionShown}) || up != strings.TrimPrefix(questionShown, "show ") || *r.bells != 2*bells {
				t.Errorf("profile %q, the 5 s up: calls %v, up %q, %d BELs", profile, calls, up, *r.bells)
			}
		})
	}
}

// TestNotifyText is criterion 64.3: each need's title and text, for Claude
// Code and Codex, and nothing of the session in them: no command, path,
// question or project.
func TestNotifyText(t *testing.T) {
	for _, c := range []struct{ id, name string }{{"claude-code", "Claude Code"}, {"codex", "OpenAI Codex"}} {
		synctest.Test(t, func(t *testing.T) {
			r := newPingRigFor(t, c.id, "")
			f := withNotifier(r)
			r.h.send(tea.BlurMsg{})
			step := func(evs ...event.Event) {
				*r.now = r.now.Add(time.Minute)
				r.push(evs...)
			}
			step(evOf(event.TurnStarted, nil), approvalEv("a1"))
			step(resolvedEv("a1"), questionEv("q1"))
			step(resolvedEv("q1"), turnEndEv("completed"))
			step(evOf(event.TurnStarted, nil), turnEndEv("failed"))
			step(evOf(event.TurnStarted, nil), crashEv)
			calls, _ := f.state()
			var shown []string
			for _, call := range calls {
				if strings.HasPrefix(call, "show ") {
					shown = append(shown, call)
				}
			}
			want := []string{
				"show Approval required | " + c.name + " is waiting for your approval.",
				"show Question waiting | " + c.name + " has a question for you.",
				"show Turn completed | " + c.name + " has finished its task.",
				"show Turn failed | " + c.name + "'s turn ended with an error.",
				"show Session ended | " + c.name + " stopped unexpectedly.",
			}
			if !slices.Equal(shown, want) {
				t.Errorf("%s: shown\n%s\nwant\n%s", c.id, strings.Join(shown, "\n"), strings.Join(want, "\n"))
			}
			project := r.h.m.State().Context.Project
			for _, call := range calls {
				if strings.Contains(call, "ls") || strings.Contains(call, "colour") || project != "" && strings.Contains(call, filepath.Base(project)) {
					t.Errorf("%s: the session's content in %q", c.id, call)
				}
			}
		})
	}
	synctest.Test(t, func(t *testing.T) {
		h := newHarness(t, Options{Config: Config{Mock: filepath.Join("..", "..", "testdata", "mock", "claude-subagent-approval.jsonl")}})
		f := &fakeNotifier{}
		h.m.opt.Notify, h.m.opt.Unnotify = f.show, f.clear
		h.deliver(tea.BlurMsg{})
		h.settle()
		if calls, _ := f.state(); len(calls) == 0 || calls[0] != "show Turn completed | The mock session has finished its task." {
			t.Errorf("a mock: %v", calls)
		}
	})
}

// TestNotifyClear is criterion 64.4: a newer notification replaces the
// older; it is taken out once nothing waits (an approval answered, the
// focus back after a turn's end) and when the dashboard quits, a clear
// never before the notification it clears, none with nothing up.
func TestNotifyClear(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := newPingRig(t, "")
		f := withNotifier(r)
		r.h.send(tea.BlurMsg{})
		r.push(evOf(event.TurnStarted, nil), approvalEv("a1"))
		*r.now = r.now.Add(time.Minute)
		r.push(approvalEv("a2"))
		if calls, up := f.state(); !slices.Equal(calls, []string{approvalShown, approvalShown}) || up == "" {
			t.Fatalf("two approvals: calls %v, up %q", calls, up)
		}
		r.h.send(actionMsg{state.Answer{ID: "a1", Decision: "allow"}})
		if calls, _ := f.state(); len(calls) != 2 {
			t.Fatalf("one of two answered: %v", calls)
		}
		r.h.send(actionMsg{state.Answer{ID: "a2", Decision: "allow"}})
		if calls, up := f.state(); !slices.Equal(calls, []string{approvalShown, approvalShown, "clear"}) || up != "" {
			t.Fatalf("both answered: calls %v, up %q", calls, up)
		}
		*r.now = r.now.Add(time.Minute)
		r.push(turnEndEv("completed"))
		r.h.send(tea.FocusMsg{})
		if calls, up := f.state(); !slices.Equal(calls[3:], []string{turnShown, "clear"}) || up != "" {
			t.Fatalf("a turn's end, then the focus: calls %v, up %q", calls, up)
		}
		r.h.send(tea.BlurMsg{})
		*r.now = r.now.Add(time.Minute)
		r.push(evOf(event.TurnStarted, nil), turnEndEv("completed"))
		r.h.m.Shutdown()
		if calls, up := f.state(); !slices.Equal(calls[5:], []string{turnShown, "clear"}) || up != "" {
			t.Errorf("quitting: calls %v, up %q", calls, up)
		}
	})
	synctest.Test(t, func(t *testing.T) {
		r := newPingRig(t, "")
		f := withNotifier(r)
		r.push(evOf(event.TurnStarted, nil), turnEndEv("completed"))
		r.h.m.Shutdown()
		if calls, _ := f.state(); len(calls) != 0 {
			t.Errorf("nothing up, quitting: %v", calls)
		}
	})
}

// TestNotifyFailure: a notification Windows refuses is said in the toast,
// and the ping goes on.
func TestNotifyFailure(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := newPingRig(t, input.AccProfile)
		f := withNotifier(r)
		f.err = errors.New("CreateToastNotifierWithId: HRESULT 0x80070490")
		r.h.send(tea.BlurMsg{})
		r.push(evOf(event.TurnStarted, nil), turnEndEv("completed"))
		if !strings.Contains(r.h.m.ui.Toast, "the Windows notification failed: CreateToastNotifierWithId: HRESULT 0x80070490") || *r.bells != 1 {
			t.Errorf("toast %q, %d BELs", r.h.m.ui.Toast, *r.bells)
		}
	})
}

// TestNotifySetUp: the notifications are made ready in a command at the
// start, off the first frame's path; their calls come in a message, or the
// reason there are none, which the startup screen shows (design D5).
func TestNotifySetUp(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		f := &fakeNotifier{}
		calls := 0
		h := newHarness(t, Options{Cwd: t.TempDir(), NotifySetUp: func() (func(string, string) error, func() error, string) {
			calls++
			return f.show, f.clear, ""
		}})
		if h.m.opt.Notify != nil {
			t.Fatal("the calls there before the set-up's message")
		}
		h.settle()
		if calls != 1 || h.m.opt.Notify == nil || h.m.opt.Unnotify == nil || h.m.ui.NotifyProblem != "" {
			t.Errorf("ready: %d set-ups, notify %v, unnotify %v, problem %q", calls, h.m.opt.Notify != nil, h.m.opt.Unnotify != nil, h.m.ui.NotifyProblem)
		}
	})
	synctest.Test(t, func(t *testing.T) {
		const why = `notifications off: the Start menu's "Agent Command Center" shortcut is not ACC's (its app ID is "Someone.Else"); left as it is`
		h := newHarness(t, Options{Cwd: t.TempDir(), NotifySetUp: func() (func(string, string) error, func() error, string) { return nil, nil, why }})
		h.settle()
		h.send(tea.WindowSizeMsg{Width: 120, Height: 40})
		if h.m.opt.Notify != nil || h.m.ui.NotifyProblem != why {
			t.Errorf("not ACC's: notify %v, problem %q", h.m.opt.Notify != nil, h.m.ui.NotifyProblem)
		}
	})
}

// TestNotifyMostUrgentRung is design D3's "most urgent need rung": a
// question rung while an approval rung earlier still waits is notified as
// the question, though the title names the approval.
func TestNotifyMostUrgentRung(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := newPingRig(t, "")
		f := withNotifier(r)
		r.h.send(tea.BlurMsg{})
		r.push(evOf(event.TurnStarted, nil), approvalEv("a1"))
		*r.now = r.now.Add(time.Minute)
		r.push(questionEv("q1"))
		if calls, _ := f.state(); !slices.Equal(calls, []string{approvalShown, questionShown}) || r.title() != needTitles[needApproval] {
			t.Errorf("calls %v, title %q", calls, r.title())
		}
	})
}
