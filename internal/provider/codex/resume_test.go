package codex

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/Evengelius/agent-command-center/internal/event"
	"github.com/Evengelius/agent-command-center/internal/fixture"
	"github.com/Evengelius/agent-command-center/internal/provider"
)

// Plan 69, criteria 69.5 and 69.8 for Codex: the home screen's resume asks
// for the thread without its turns, the crash resume's request as before
// (TestResume); Codex's refusal for want of the thread (P69b) is reported as
// for good for that resume only, and no thread is started after it; the
// notification that follows every resume is logged, not unrecognised.

func p69b(t *testing.T, proc string) []fixture.Record {
	t.Helper()
	recs, err := fixture.Read(probe("p69b-codex-resume.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	return fixture.Of(recs, proc)
}

func refusals(evs []event.Event) []event.Problem {
	var out []event.Problem
	for _, ev := range evs {
		if p, ok := ev.Payload.(event.Problem); ok && ev.Kind == event.Error {
			out = append(out, p)
		}
	}
	return out
}

// TestResumeLast: the home screen's resume writes thread/resume with
// excludeTurns, the rest of the handshake as a crash resume's.
func TestResumeLast(t *testing.T) {
	recs := []string{
		handshake[0], handshake[1], handshake[2],
		`{"dir":"in","msg":{"id":2,"method":"thread/resume","params":{"threadId":"th-9"}}}`,
		`{"dir":"out","msg":{"id":2,"result":{"thread":{"id":"th-9"},"model":"gpt-5.6-sol","cwd":"<probe-work>"}}}`,
		handshake[5], handshake[6],
	}
	p := &Provider{}
	p.setFallbackModel("gpt-5.6-sol")
	r, _ := startCtx(t, "replay", p, provider.SessionContext{ResumeID: "th-9", ResumeLast: true}, script(t, recs...))
	if st := r.wait(event.SessionStarted).Payload.(event.Started); st.SessionID != "th-9" {
		t.Errorf("started %+v", st)
	}
	r.limits()
	r.close()
	r.expect([]string{
		initLine,
		`{"method":"initialized"}`,
		`{"id":2,"method":"config/read","params":{"cwd":"<dir>","includeLayers":false}}`,
		`{"id":3,"method":"configRequirements/read"}`,
		`{"id":4,"method":"model/list","params":{}}`,
		`{"id":5,"method":"thread/resume","params":{"threadId":"th-9","model":"gpt-5.6-sol","sandbox":"read-only","approvalPolicy":"on-request","approvalsReviewer":"user","excludeTurns":true}}`,
		`{"id":6,"method":"account/rateLimits/read"}`,
	})
}

// TestResumeLastRefused: with no such thread the home screen's resume is
// refused for good; another refusal, or one naming another thread, is
// another failure; neither is followed by a thread/start.
func TestResumeLastRefused(t *testing.T) {
	for _, c := range []struct {
		err  string
		gone bool
	}{
		{`{"code":-32600,"message":"no rollout found for thread id th-9"}`, true},
		{`{"code":-32600,"message":"invalid session id: invalid character: expected an optional prefix of ` + "`urn:uuid:`" + ` followed by [0-9a-fA-F-], found ` + "`t`" + ` at 1"}`, true},
		{`{"code":-32600,"message":"thread th-9 is locked"}`, false},
		{`{"code":-32600,"message":"no rollout found for thread id th-8"}`, false},
		{`{"code":-32603,"message":"no rollout found for thread id th-9"}`, false},
	} {
		recs := []string{handshake[0], handshake[1], handshake[2],
			`{"dir":"in","msg":{"id":2,"method":"thread/resume","params":{"threadId":"th-9"}}}`,
			`{"dir":"out","msg":{"id":2,"error":` + c.err + `}}`}
		r, _ := startCtx(t, "replay", nil, provider.SessionContext{ResumeID: "th-9", ResumeLast: true}, script(t, recs...))
		p := r.wait(event.Error).Payload.(event.Problem)
		if p.NoSession != c.gone || !strings.HasPrefix(p.Message, "thread/resume refused by Codex: ") {
			t.Errorf("%s: %+v", c.err, p)
		}
		r.close()
		for _, l := range r.written() {
			if strings.Contains(l, `"thread/start"`) {
				t.Errorf("%s: a thread started after the refused resume: %s", c.err, l)
			}
		}
	}
}

// TestResumeRefusalP69b: P69b's unknown thread and malformed id, as Codex
// 0.163.0 refused them: for good only for the home screen's resume of that
// id, never for the crash resume's.
func TestResumeRefusalP69b(t *testing.T) {
	recs := p69b(t, "unknown")
	var starts []int
	var ids []string
	for i, r := range recs {
		var m struct {
			Method string `json:"method"`
			Params struct {
				ThreadID string `json:"threadId"`
			} `json:"params"`
		}
		if r.Dir == "in" && json.Unmarshal(r.Msg, &m) == nil && m.Method == "thread/resume" {
			starts, ids = append(starts, i), append(ids, m.Params.ThreadID)
		}
	}
	if len(ids) != 2 || ids[1] != "not-a-thread-69" {
		t.Fatalf("thread/resume ids %q", ids)
	}
	// Each request alone after the handshake, as one session sends it.
	alone := func(i int) []fixture.Record {
		end := len(recs)
		if i+1 < len(starts) {
			end = starts[i+1]
		}
		return append(append([]fixture.Record(nil), recs[:starts[0]]...), recs[starts[i]:end]...)
	}
	for i, id := range ids {
		m := NewMapper()
		m.ResumeAsked(id)
		if ps := refusals(fixture.Flatten(fixture.Replay(ProviderID, alone(i), m))); len(ps) != 1 || !ps[0].NoSession {
			t.Errorf("%s, the home screen's resume: %+v", id, ps)
		}
		if ps := refusals(fixture.Flatten(fixture.Replay(ProviderID, alone(i), NewMapper()))); len(ps) != 1 || ps[0].NoSession {
			t.Errorf("%s, the crash resume: %+v", id, ps)
		}
	}
}

// TestGoalCleared: thread/goal/cleared after each of P69b's resumes is a
// status; the full-history resume's deprecation notice stays unrecognised,
// as the crash resume has it today.
func TestGoalCleared(t *testing.T) {
	for _, proc := range []string{"full", "exclude"} {
		goal, deprecated := 0, 0
		for _, ev := range fixture.Flatten(fixture.Replay(ProviderID, p69b(t, proc), NewMapper())) {
			switch p := ev.Payload.(type) {
			case event.Status:
				if p.Status == "goal cleared" {
					goal++
				}
			case event.Unknown:
				switch p.Type {
				case "thread/goal/cleared":
					t.Errorf("%s: thread/goal/cleared unrecognised", proc)
				case "deprecationNotice":
					deprecated++
				}
			}
		}
		want := 0
		if proc == "full" {
			want = 1
		}
		if goal != 1 || deprecated != want {
			t.Errorf("%s: %d goal cleared, %d deprecation notices, want 1 and %d", proc, goal, deprecated, want)
		}
	}
}
