package claude

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Evengelius/agent-command-center/internal/event"
	"github.com/Evengelius/agent-command-center/internal/fixture"
	"github.com/Evengelius/agent-command-center/internal/provider"
)

// Plan 69, criteria 69.5 and 69.8 for Claude Code: the home screen's resume
// is a start's arguments with --resume (TestArgs), and Claude Code's refusal
// of an ID it has no session for, before the handshake's answer (P69a), is
// reported as for good for that resume only; the crash resume maps it as
// before.

func p69a(t *testing.T, proc string) []fixture.Record {
	t.Helper()
	recs, err := fixture.Read(filepath.Join("..", "..", "..", "testdata", "probes", "p69a-claude-resume.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	return fixture.Of(recs, proc)
}

// resumed is the ID a process of P69a resumed: its --resume argument.
func resumed(t *testing.T, recs []fixture.Record) string {
	t.Helper()
	var meta struct {
		Argv []string `json:"argv"`
	}
	if len(recs) == 0 || recs[0].Dir != "meta" || json.Unmarshal(recs[0].Msg, &meta) != nil || len(meta.Argv) < 2 || meta.Argv[len(meta.Argv)-2] != "--resume" {
		t.Fatalf("no --resume in the process's first record")
	}
	return meta.Argv[len(meta.Argv)-1]
}

// mapP69a maps a process's lines, its "/usage" written as the dashboard
// writes it.
func mapP69a(m *Mapper, recs []fixture.Record) []event.Event {
	var out []event.Event
	for _, r := range recs {
		switch r.Dir {
		case "out":
			out = append(out, m.Out(r.Msg)...)
		case "in":
			if strings.Contains(string(r.Msg), `"content":"/usage"`) {
				out = append(out, m.InInternal(r.Msg)...)
			} else {
				out = append(out, m.In(r.Msg)...)
			}
		}
	}
	return out
}

func problems(evs []event.Event) []event.Problem {
	var out []event.Problem
	for _, ev := range evs {
		if p, ok := ev.Payload.(event.Problem); ok && ev.Kind == event.Error {
			out = append(out, p)
		}
	}
	return out
}

func refusal(id string) []byte {
	return []byte(`{"type":"result","subtype":"error_during_execution","is_error":true,"num_turns":0,"session_id":"` + id + `","errors":["No conversation found with session ID: ` + id + `"]}`)
}

func TestResumeRefusal(t *testing.T) {
	unknown := p69a(t, "unknown")
	id := resumed(t, unknown)
	t.Run("the home screen's resume: for good", func(t *testing.T) {
		m := NewMapper()
		m.ResumeAsked(id)
		evs := mapP69a(m, unknown)
		if ps := problems(evs); len(ps) != 1 || !ps[0].NoSession || ps[0].Message != "Claude Code: No conversation found with session ID: "+id {
			t.Fatalf("problems %+v", ps)
		}
		if n := count(evs, event.TurnCompleted); n != 0 {
			t.Errorf("the refusal also mapped as %d turn ends", n)
		}
	})
	t.Run("the crash resume: as before", func(t *testing.T) {
		evs := mapP69a(NewMapper(), unknown)
		if ps := problems(evs); len(ps) != 0 {
			t.Errorf("problems %+v", ps)
		}
		ends := 0
		for _, ev := range evs {
			if end, ok := ev.Payload.(event.TurnEnd); ok && ev.Kind == event.TurnCompleted {
				ends++
				if end.Outcome != "failed" || end.Error != "No conversation found with session ID: "+id {
					t.Errorf("turn end %+v", end)
				}
			}
		}
		if ends != 1 {
			t.Errorf("%d turn ends, want the one failed", ends)
		}
	})
	t.Run("other words or another ID: another failure", func(t *testing.T) {
		for _, line := range [][]byte{
			[]byte(`{"type":"result","subtype":"error_during_execution","is_error":true,"num_turns":0,"session_id":"` + id + `","errors":["Session file damaged: ` + id + `"]}`),
			refusal("0f5e2a64-0000-4000-8000-00000000beef"),
		} {
			m := NewMapper()
			m.ResumeAsked(id)
			for _, p := range problems(m.Out(line)) {
				if p.NoSession {
					t.Errorf("%s read as for good", line)
				}
			}
		}
	})
	t.Run("after the handshake's answer: a turn's", func(t *testing.T) {
		recs := p69a(t, "resume")
		asked := resumed(t, recs)
		m := NewMapper()
		m.ResumeAsked(asked)
		evs := mapP69a(m, recs)
		if ps := problems(evs); len(ps) != 0 {
			t.Errorf("the resumed session's problems %+v", ps)
		}
		started := 0
		for _, ev := range evs {
			if st, ok := ev.Payload.(event.Started); ok {
				started++
				if st.SessionID != asked {
					t.Errorf("started %q, asked %q", st.SessionID, asked)
				}
			}
		}
		if started == 0 {
			t.Error("no session.started")
		}
		for _, p := range problems(m.Out(refusal(asked))) {
			if p.NoSession {
				t.Error("a refusal's words after the handshake read as the resume's")
			}
		}
	})
}

// TestResumeLastWired: the session context's home screen resume reaches the
// mapper, and the crash resume's does not (plan 69, design D7).
func TestResumeLastWired(t *testing.T) {
	const id = "0f5e2a64-0000-4000-8000-00000000dead"
	for _, last := range []bool{true, false} {
		r, _ := startCtx(t, "noconversation", nil, provider.SessionContext{ResumeID: id, ResumeLast: last})
		r.wait(event.SessionEnded)
		ps := problems(r.evs)
		if last && (len(ps) != 1 || !ps[0].NoSession || ps[0].Message != "Claude Code: No conversation found with session ID: "+id) {
			t.Errorf("home screen's resume: problems %+v", ps)
		}
		if !last && len(ps) != 0 {
			t.Errorf("crash resume: problems %+v", ps)
		}
	}
}
