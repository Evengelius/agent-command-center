package state_test

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/Evengelius/agent-command-center/internal/event"
	"github.com/Evengelius/agent-command-center/internal/fixture"
	"github.com/Evengelius/agent-command-center/internal/provider/claude"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// Plan 22, amendment 20: a failing SessionStart hook. Claude Code carries on
// after it, and its end comes before the initialize response (P22o: the hook's
// end is line 4, the response line 5), where an error is taken for a failed
// handshake and closes the session (reduce.go, Problem). The adapter gives it as
// a notice, which the reducer draws in zone B in any phase.

// failingHook is P22o with its hook_response (line 4) made a failure: outcome
// "error", exit code 1, stderr "pxw: not found".
func failingHook(t *testing.T) []fixture.Record {
	t.Helper()
	recs, err := fixture.Read(filepath.Join("../../testdata/probes", p22o))
	if err != nil {
		t.Fatal(err)
	}
	line := string(recs[3].Msg)
	for _, from := range []string{`"stderr":""`, `"exit_code":0`, `"outcome":"success"`} {
		if !strings.Contains(line, from) {
			t.Fatalf("P22o line 4 lacks %s: %s", from, line)
		}
	}
	line = strings.Replace(line, `"stderr":""`, `"stderr":"pxw: not found\n"`, 1)
	line = strings.Replace(line, `"exit_code":0`, `"exit_code":1`, 1)
	line = strings.Replace(line, `"outcome":"success"`, `"outcome":"error"`, 1)
	recs[3].Msg = []byte(line)
	return recs
}

// startedReplay replays records through the Claude Code adapter and the reducer,
// as the application does at its startup screen (a start waiting for the
// handshake), up to the turn's end: the process's own exit, the recording's last
// record, is not what is looked at. It checks after every event that the session
// has not crashed, and gives the commands the reducer asked for.
func startedReplay(t *testing.T, recs []fixture.Record) (*state.State, []state.Command) {
	t.Helper()
	per := fixture.Replay(claude.ProviderID, recs, claude.NewMapper())
	s := state.New()
	s.Startup.Phase = state.Starting
	var cmds []state.Command
	for i, evs := range per[:len(per)-1] {
		for _, ev := range evs {
			cmds = append(cmds, s.Apply(ev)...)
			if s.Session.Phase == state.Crashed || s.Startup.Phase == state.StartFailed {
				t.Fatalf("record %d (%s): session %q, startup %q: %s", i+1, ev.Kind, s.Session.Phase, s.Startup.Phase, s.Session.Reason)
			}
		}
	}
	return s, cmds
}

// A failing hook before the initialize response: the session reaches Ready and
// stays, the reducer asks for no close, no error is kept, and the hook's end is
// exactly one notice, observed, ahead of the owner's prompt, with the text the
// adapter gave. The rest of the conversation is what it is with the success.
func TestFailingHookDoesNotEndTheSession(t *testing.T) {
	const notice = "hook SessionStart:startup failed: exit 1: pxw: not found"
	s, cmds := startedReplay(t, failingHook(t))
	if s.Session.Phase != state.Ready || s.Startup.Phase != state.StartReady {
		t.Errorf("session %q (%s), startup %q; want ready", s.Session.Phase, s.Session.Reason, s.Startup.Phase)
	}
	for _, c := range cmds {
		if c.Kind == state.CmdClose {
			t.Errorf("the reducer asked to close the session: %+v", c)
		}
	}
	if len(s.Errors) != 0 {
		t.Errorf("errors kept: %+v", s.Errors)
	}
	if s.Turn.Phase != state.Idle || s.Turn.Outcome != "completed" {
		t.Errorf("turn %q (%q), want idle, completed: the turn ran", s.Turn.Phase, s.Turn.Outcome)
	}

	notices := 0
	for i, e := range s.Conversation {
		if e.Role != "notice" {
			continue
		}
		notices++
		if e.Text != notice || e.Origin != event.Observed || e.Agent != "" || e.Stage != 0 || i != 0 {
			t.Errorf("notice %d: %q (origin %s, agent %q, turn %d), want %q observed, the orchestrator's, before any turn, the first entry", i+1, e.Text, e.Origin, e.Agent, e.Stage, notice)
		}
	}
	if notices != 1 {
		t.Fatalf("%d notices, want 1", notices)
	}
	if len(s.Conversation) < 2 || s.Conversation[1].Role != "user" {
		t.Fatalf("the entry after the notice is not the owner's prompt: %+v", s.Conversation)
	}

	// The same recording with the hook a success: the same conversation, minus
	// the notice (the events after it are numbered one apart).
	ok, _ := startedReplay(t, mustRead(t, p22o))
	same := func(es []state.Entry) []state.Entry {
		out := append([]state.Entry(nil), es...)
		for i := range out {
			out[i].Seq = 0
		}
		return out
	}
	if got, want := same(s.Conversation[1:]), same(ok.Conversation); !reflect.DeepEqual(got, want) {
		t.Errorf("the conversation after the notice differs from the success's:\n%+v\nwant\n%+v", got, want)
	}
	if len(ok.Conversation) != len(s.Conversation)-1 || ok.Session.Phase != state.Ready || len(ok.Errors) != 0 {
		t.Errorf("the success: %d entries, session %q, %d errors", len(ok.Conversation), ok.Session.Phase, len(ok.Errors))
	}
}

// mustRead is a recording of testdata/probes.
func mustRead(t *testing.T, name string) []fixture.Record {
	t.Helper()
	recs, err := fixture.Read(filepath.Join("../../testdata/probes", name))
	if err != nil {
		t.Fatal(err)
	}
	return recs
}
