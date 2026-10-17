package render

import (
	"fmt"
	"math"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"charm.land/lipgloss/v2"

	"github.com/Evengelius/agent-command-center/internal/event"
	"github.com/Evengelius/agent-command-center/internal/fixture"
	"github.com/Evengelius/agent-command-center/internal/provider/claude"
	"github.com/Evengelius/agent-command-center/internal/provider/codex"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// Plan 4, step 7: zone D's values from the recordings' exact lines, and every
// N/A case of reliability.md's two value tables with its reason.

func records(t *testing.T, name string) []fixture.Record {
	t.Helper()
	recs, err := fixture.Read(filepath.Join("..", "..", "testdata", "probes", name))
	if err != nil {
		t.Fatal(err)
	}
	return recs
}

// rec is a synthetic record at tms milliseconds, for a case no probe has.
func rec(tms int64, dir, msg string) fixture.Record {
	return fixture.Record{TMs: tms, Dir: dir, Msg: []byte(msg)}
}

// replayed is the state after recs, the frame time a second after the last.
func replayed(provider string, m fixture.Mapper, recs []fixture.Record) (*state.State, time.Time) {
	s := state.New()
	s.Context.Provider = provider // as the startup screen sets it
	for _, ev := range fixture.Flatten(fixture.Replay(provider, recs, m)) {
		s.Apply(ev)
	}
	now := fixture.Epoch
	if len(s.Log) > 0 {
		now = s.Log[len(s.Log)-1].At.Add(time.Second)
	}
	return s, now
}

// zoneDText is zone D at its tallest, wide enough for no row to be cut.
func zoneDText(s *state.State, now time.Time) string {
	return texts(zoneD(s, UI{Width: 400, Height: 60, Now: now}, 400, 5, 3))
}

func near(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

// Criterion 1: the values each recording's lines give, and how zone D shows
// them.
func TestTelemetryParsing(t *testing.T) {
	t.Run("P1 line 3: quota and overage", func(t *testing.T) {
		s, now := replayed("claude-code", claude.NewMapper(), records(t, "p1-claude-stream.jsonl")[:3])
		q5, q7 := s.Metrics["quota.5h"], s.Metrics["quota.7d"]
		if !q5.Present || !near(q5.V, 22) || q5.Documented || q5.ResetsAt.Unix() != 1792109400 {
			t.Errorf("quota.5h %+v", q5)
		}
		if !q7.Present || !near(q7.V, 45) || q7.Documented || q7.ResetsAt.Unix() != 1792216800 {
			t.Errorf("quota.7d %+v", q7)
		}
		if o := s.Metrics["overage"]; o.Text != "rejected (org_level_disabled)" {
			t.Errorf("overage %+v", o)
		}
		d := zoneDText(s, now)
		for _, want := range []string{"5h ", " 22%†", "7d ", " 45%†", "overage rejected (org_level_disabled)"} {
			if !strings.Contains(d, want) {
				t.Errorf("zone D lacks %q:\n%s", want, d)
			}
		}
		if strings.Contains(d, "22%† (") || strings.Contains(d, "45%†  overage") {
			t.Errorf("overage shown with the quota:\n%s", d)
		}
	})

	t.Run("P1 result: cost as an estimate only", func(t *testing.T) {
		s, now := replayed("claude-code", claude.NewMapper(), records(t, "p1-claude-stream.jsonl"))
		if c := s.Metrics["cost.usd"]; !c.Present || c.V != 0.00249374 {
			t.Errorf("cost.usd %+v", c)
		}
		d := zoneDText(s, now) + texts(detailsBody(s, UI{Width: wide, Now: now}, wide))
		all := strings.Count(d, "$")
		est := len(regexp.MustCompile(`\$[0-9.]+ API-equiv\. estimate`).FindAllString(d, -1))
		if all == 0 || est != all || !strings.Contains(d, "$0.0025 API-equiv. estimate") {
			t.Errorf("%d amounts, %d labelled estimates:\n%s", all, est, d)
		}
	})

	t.Run("P6b line 7: windows by duration, plan and credits", func(t *testing.T) {
		s, now := replayed(codex.ProviderID, codex.NewMapper(), records(t, "p6b-codex-app-server.jsonl")[:7])
		q5, q7 := s.Metrics["quota.5h"], s.Metrics["quota.7d"]
		if !q5.Present || q5.V != 0 || q5.ResetsAt.Unix() != 1792111426 {
			t.Errorf("quota.5h %+v", q5)
		}
		if !q7.Present || q7.V != 0 || q7.ResetsAt.Unix() != 1792657845 {
			t.Errorf("quota.7d %+v", q7)
		}
		if p := s.Metrics["plan"]; p.Text != "plus" {
			t.Errorf("plan %+v", p)
		}
		if c := s.Metrics["credits"]; c.Text != "none" {
			t.Errorf("credits %+v", c)
		}
		d := zoneDText(s, now)
		for _, want := range []string{" 0%  resets", "plan plus", "credits none"} {
			if !strings.Contains(d, want) {
				t.Errorf("zone D lacks %q:\n%s", want, d)
			}
		}
	})

	t.Run("P7: thread totals and the derived session total", func(t *testing.T) {
		s, now := replayed(codex.ProviderID, codex.NewMapper(), records(t, "p7-codex-subagents.jsonl"))
		got := map[float64]bool{s.Metrics["tokens.total"].V: true}
		for _, a := range s.Agents {
			if a.Kind == "thread" && a.Tokens.Present {
				got[a.Tokens.V] = true
			}
		}
		for _, want := range []float64{76521, 30455, 77336} {
			if !got[want] || len(got) != 3 {
				t.Errorf("thread totals %v, want 76521, 30455 and 77336", got)
			}
		}
		if v := s.Metrics["tokens.session"]; v.V != 184312 || v.Origin != event.Derived {
			t.Errorf("tokens.session %+v", v)
		}
		if d := zoneDText(s, now); !strings.Contains(d, "tokens 184.3k (derived)") {
			t.Errorf("zone D:\n%s", d)
		}
	})

	t.Run("P3b: the running total, subagents included", func(t *testing.T) {
		s, now := replayed("claude-code", claude.NewMapper(), records(t, "p3b-claude-subagent-approval.jsonl"))
		if v := s.Metrics["tokens.session"]; v.V != 159362 || v.Origin == event.Derived {
			t.Errorf("tokens.session %+v", v)
		}
		if d := zoneDText(s, now); !strings.Contains(d, "tokens 159.4k   ") {
			t.Errorf("zone D:\n%s", d)
		}
	})
}

// Criterion 2: each N/A case shows "N/A" and its reason in zone D, never 0.
func TestTelemetryNA(t *testing.T) {
	p2 := func(t *testing.T) []fixture.Record { return records(t, "p2-claude-conversation.jsonl") }
	p6b := func(t *testing.T) []fixture.Record { return records(t, "p6b-codex-app-server.jsonl") }
	upTo := func(recs []fixture.Record, n int, more ...fixture.Record) []fixture.Record {
		return append(append([]fixture.Record(nil), recs[:n]...), more...)
	}
	// P2 with get_context_usage refused instead of answered (line 6).
	refused := func(t *testing.T) []fixture.Record {
		recs := p2(t)
		if !strings.Contains(string(recs[5].Msg), `"request_id":"req_2"`) {
			t.Fatalf("line 6 is not get_context_usage's reply: %.80s", recs[5].Msg)
		}
		recs[5] = rec(recs[5].TMs, "out", `{"type":"control_response","response":{"subtype":"error","request_id":"req_2","error":"Unsupported control request subtype: get_context_usage"}}`)
		return recs
	}
	const p6bRead = `{"id":99,"method":"account/rateLimits/read"}`

	cases := []struct {
		name     string
		provider string
		mapper   func() fixture.Mapper
		recs     func(t *testing.T) []fixture.Record
		at       time.Duration // the frame time after the last event; default a second
		want     []string
		errors   int // the errors the case counts; -1 does not check
	}{
		// Claude Code (reliability.md, Telemetry sources: Claude Code's values).
		{name: "claude quota before the first event", provider: "claude-code",
			recs: func(t *testing.T) []fixture.Record { return upTo(p2(t), 3) },
			want: []string{"5h N/A (no report yet)", "7d N/A (no report yet)"}, errors: 0},
		{name: "claude API key", provider: "claude-code",
			recs: func(t *testing.T) []fixture.Record {
				return []fixture.Record{rec(1, "out", `{"type":"system","subtype":"init","session_id":"s","model":"claude-sonnet-5-5","apiKeySource":"ANTHROPIC_API_KEY"}`)}
			},
			want: []string{"5h N/A (API key: no subscription window)", "7d N/A (API key: no subscription window)"}, errors: 0},
		{name: "claude window past its reset", provider: "claude-code",
			recs: func(t *testing.T) []fixture.Record { return records(t, "p1-claude-stream.jsonl")[:3] },
			at:   time.Unix(1792216800, 0).Sub(fixture.Epoch),
			want: []string{"5h N/A (window reset; waiting for the next report)", "7d N/A (window reset; waiting for the next report)"}},
		{name: "claude status only", provider: "claude-code",
			recs: func(t *testing.T) []fixture.Record {
				return upTo(p2(t), 3, rec(7000, "out", `{"type":"rate_limit_event","rate_limit_info":{"status":"allowed_warning","rateLimitType":"seven_day"}}`))
			},
			want: []string{"7d allowed_warning (no percentage reported)", "5h N/A (no report yet)"}},
		{name: "claude overage not reported", provider: "claude-code",
			recs: func(t *testing.T) []fixture.Record { return upTo(p2(t), 3) },
			want: []string{"overage N/A (not reported)"}},
		{name: "claude context before the first turn", provider: "claude-code",
			recs: func(t *testing.T) []fixture.Record { return upTo(p2(t), 3) },
			want: []string{"context N/A (no turn yet)"}},
		{name: "claude context refused, before the first result", provider: "claude-code",
			recs: func(t *testing.T) []fixture.Record { return upTo(refused(t), 8) },
			want: []string{"context N/A (no turn yet)", "errors 1"}, errors: 1},
		{name: "claude context refused, derived after the result", provider: "claude-code",
			recs: func(t *testing.T) []fixture.Record { return upTo(refused(t), 48) },
			want: []string{"context 3% of 1M (derived)", "errors 1"}, errors: 1},
		{name: "claude tokens, cost, duration before the first result", provider: "claude-code",
			recs: func(t *testing.T) []fixture.Record { return upTo(p2(t), 8) },
			want: []string{"tokens N/A (no result yet)", "cost N/A (no result yet)", "last turn N/A (no result yet)"}},
		{name: "claude agents before the stream is up", provider: "claude-code",
			recs: func(t *testing.T) []fixture.Record { return upTo(p2(t), 1) },
			want: []string{"agents N/A (session starting)"}},
		{name: "claude limit and retries", provider: "claude-code",
			recs: p2,
			want: []string{"active / limit N/A (not reported)", "retries N/A (none reported)"}, errors: -1},

		// Codex (reliability.md, Telemetry sources: Codex's values).
		{name: "codex API key", provider: codex.ProviderID,
			recs: func(t *testing.T) []fixture.Record {
				return upTo(p6b(t), 4, rec(400, "in", p6bRead),
					rec(500, "out", `{"id":99,"error":{"code":-32600,"message":"chatgpt authentication required to read rate limits"}}`))
			},
			want: []string{"5h N/A (API key: ChatGPT sign-in needed)", "7d N/A (API key: ChatGPT sign-in needed)",
				"plan N/A (API key: ChatGPT sign-in needed)", "credits N/A (API key: ChatGPT sign-in needed)"}, errors: 0},
		{name: "codex no 5-hour window", provider: codex.ProviderID,
			recs: func(t *testing.T) []fixture.Record {
				return upTo(p6b(t), 4, rec(400, "in", p6bRead),
					rec(500, "out", `{"id":99,"result":{"rateLimits":{"primary":{"usedPercent":12,"windowDurationMins":10080,"resetsAt":1792657845},"planType":"plus"}}}`))
			},
			want: []string{"5h N/A (plan has no 5-hour limit)", "12%"}},
		{name: "codex read failed keeps the last value", provider: codex.ProviderID,
			recs: func(t *testing.T) []fixture.Record {
				return upTo(p6b(t), 7, rec(60300, "in", p6bRead),
					rec(60301, "out", `{"id":99,"error":{"code":-32603,"message":"upstream timeout"}}`))
			},
			want: []string{"5h N/A (read failed: upstream timeout; last 0% 1m 00s ago)", "7d N/A (read failed: upstream timeout; last 0% 1m 00s ago)", "errors 1"}, errors: 1},
		{name: "codex context before the first model call", provider: codex.ProviderID,
			recs: func(t *testing.T) []fixture.Record { return upTo(p6b(t), 20) },
			want: []string{"context N/A (no model call yet)"}},
		{name: "codex cost and limit", provider: codex.ProviderID,
			recs: p6b,
			want: []string{"cost N/A (Codex reports none)", "active / limit N/A (not read)"}, errors: -1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var m fixture.Mapper = claude.NewMapper()
			if c.provider == codex.ProviderID {
				m = codex.NewMapper()
			}
			s, now := replayed(c.provider, m, c.recs(t))
			if c.at != 0 {
				now = fixture.Epoch.Add(c.at)
			}
			d := zoneDText(s, now)
			for _, want := range c.want {
				if !strings.Contains(d, want) {
					t.Errorf("zone D lacks %q:\n%s", want, d)
				}
			}
			if c.errors >= 0 && len(s.Errors) != c.errors {
				t.Errorf("errors %d, want %d: %+v", len(s.Errors), c.errors, s.Errors)
			}
		})
	}
}

// failedLine is the "✗ turn failed:" row of zone B: its lead's and its
// text's styles, and the text.
func failedLine(s *state.State, now time.Time) (lead, msg *lipgloss.Style, text string, n int) {
	for _, l := range zoneB(s, UI{Width: 200, Height: 60, Now: now}, 200, 60, 1, nil) {
		for i, x := range l {
			if x.text == "✗ turn failed: " && i+1 < len(l) {
				lead, msg, text = x.st, l[i+1].st, l[i+1].text
				n++
			}
		}
	}
	return lead, msg, text, n
}

// Criterion 4: a failed turn shows in B "✗ turn failed: <first line of the
// error, as received>" in red, for each CLI.
func TestTurnFailedLine(t *testing.T) {
	t.Run("Codex P6's refusal", func(t *testing.T) {
		recs := records(t, "p6-codex-app-server.jsonl")
		var first string
		for _, ev := range fixture.Flatten(fixture.Replay(codex.ProviderID, recs, codex.NewMapper())) {
			if p, ok := ev.Payload.(event.TurnEnd); ok && p.Error != "" && first == "" {
				first = strings.SplitN(p.Error, "\n", 2)[0]
			}
		}
		if !strings.Contains(first, "not supported") {
			t.Fatalf("P6's turn error %q", first)
		}
		s, now := replayed(codex.ProviderID, codex.NewMapper(), recs)
		lead, msg, text, n := failedLine(s, now)
		if n != 2 || lead != stFailedBold || msg != stFailed || !strings.HasPrefix(first, strings.TrimRight(text, " ")) {
			t.Errorf("%d rows, lead %v, text %q in %v; want 2 rows of %q in red", n, lead == stFailedBold, text, msg == stFailed, first)
		}
	})

	// The documented schema (claude-code.md, section 9: Errors); no probe
	// failed a turn. P2 up to its turn in flight, then a failed result.
	for _, c := range []struct{ name, result, first, whole string }{
		{"Claude Code errors", `{"type":"result","subtype":"error_during_execution","is_error":true,"duration_ms":10,"errors":["API Error: 529 Overloaded","the request was not retried"]}`,
			"API Error: 529 Overloaded", "API Error: 529 Overloaded\nthe request was not retried"},
		{"Claude Code result text", `{"type":"result","subtype":"success","is_error":true,"duration_ms":10,"result":"Credit balance is too low"}`,
			"Credit balance is too low", "Credit balance is too low"},
	} {
		t.Run(c.name, func(t *testing.T) {
			recs := append(records(t, "p2-claude-conversation.jsonl")[:8], rec(9000, "out", c.result))
			s, now := replayed("claude-code", claude.NewMapper(), recs)
			lead, msg, text, n := failedLine(s, now)
			if n != 1 || lead != stFailedBold || msg != stFailed || strings.TrimRight(text, " ") != c.first {
				t.Errorf("%d rows, text %q; want one row of %q in red", n, text, c.first)
			}
			if len(s.Errors) != 1 || s.Errors[0].Message != "turn failed: "+c.whole {
				t.Errorf("errors %+v", s.Errors)
			}
		})
	}
}

// Criterion 5: the errors list, newest first, each with its time and who,
// the whole message wrapped.
func TestDetailsErrors(t *testing.T) {
	s := state.New()
	s.Context.Provider = "claude-code"
	at := fixture.Epoch
	s.Agents["a1"] = &state.Agent{Key: "a1", Label: "explorer"}
	s.Errors = []state.ErrorEntry{
		{Seq: 1, At: at.Add(time.Second), Message: "first error"},
		{Seq: 2, At: at.Add(2 * time.Second), Agent: "a1", Message: "second error, " + strings.Repeat("long ", 30) + "end", Retry: true},
		{Seq: 3, At: at.Add(3 * time.Second), Agent: "gone", Message: "line one\nline two"},
	}
	body := texts(detailsBody(s, UI{Width: 100, Now: at.Add(time.Minute)}, 98))
	i3, i2, i1 := strings.Index(body, "line one"), strings.Index(body, "second error"), strings.Index(body, "first error")
	if i3 < 0 || i2 < 0 || i1 < 0 || !(i3 < i2 && i2 < i1) {
		t.Errorf("not newest first:\n%s", body)
	}
	for _, want := range []string{clockSec(at.Add(time.Second)) + "  session", "explorer · retry", "gone", "line two", " end\n"} {
		if !strings.Contains(body, want) {
			t.Errorf("lacks %q:\n%s", want, body)
		}
	}
	for _, l := range strings.Split(body, "\n") {
		if textWidth(l) > 98 {
			t.Errorf("row wider than 98: %q", l)
		}
	}
	if !strings.Contains(texts(detailsBody(state.New(), UI{}, 98)), "no errors") {
		t.Error("no \"no errors\" when there are none")
	}
}

// Criterion 3: the account-linked reset credit ids Codex sends are kept
// nowhere: not in an event's Raw, not in the log, not on any screen
// (reliability.md, Telemetry sources: Plan and credits).
func TestCreditIDsNotKept(t *testing.T) {
	const secret = "RateLimitResetCredit_SECRET123"
	credits := `"rateLimitResetCredits":{"credits":[{"id":"` + secret + `","expiresAt":1792657845}]}`
	recs := append(records(t, "p6b-codex-app-server.jsonl")[:7],
		rec(20000, "out", `{"method":"account/rateLimits/updated","params":{"rateLimits":{"limitId":"codex","primary":{"usedPercent":3,"windowDurationMins":300,"resetsAt":1792111427},"planType":"plus",`+credits+`}}}`),
		rec(20100, "in", `{"id":98,"method":"account/rateLimits/read"}`),
		rec(20200, "out", `{"id":98,"result":{"rateLimits":{"limitId":"codex","primary":{"usedPercent":4,"windowDurationMins":300,"resetsAt":1792111427},"planType":"plus"},"rateLimitsByLimitId":{"codex":{`+credits+`}}}}`),
		rec(20300, "out", `{"method":"account/rateLimits/updated","params":{`+credits+`,`), // cut: unreadable
	)
	s, now := replayed("codex", codex.NewMapper(), recs)

	if q := s.Metrics["quota.5h"]; !q.Present || !near(q.V, 4) {
		t.Fatalf("the lines with credits were not read: quota.5h %+v", q)
	}
	if !strings.Contains(s.Log[len(s.Log)-1].Raw, "not kept") {
		t.Errorf("the unreadable line was kept: %q", s.Log[len(s.Log)-1].Raw)
	}
	for _, ev := range s.Log {
		if strings.Contains(ev.Raw, secret) || strings.Contains(fmt.Sprint(ev.Payload), secret) {
			t.Errorf("event %d (%s) keeps the id: %s", ev.Seq, ev.Kind, ev.Raw)
		}
	}
	for _, m := range s.Metrics {
		if strings.Contains(fmt.Sprint(m), secret) {
			t.Errorf("metric keeps the id: %+v", m)
		}
	}
	ansi := regexp.MustCompile(`\x1b\[[0-9;:]*[A-Za-z]`)
	for _, ui := range []UI{
		{Width: 200, Height: 60, Now: now},
		{Width: 200, Height: 60, Now: now, Details: true},
		{Width: 80, Height: 24, Now: now, Details: true},
	} {
		if f := ansi.ReplaceAllString(Frame(s, ui), ""); strings.Contains(f, secret) || strings.Contains(f, "SECRET") {
			t.Errorf("the frame at %dx%d (details %v) shows the id", ui.Width, ui.Height, ui.Details)
		}
	}
}
