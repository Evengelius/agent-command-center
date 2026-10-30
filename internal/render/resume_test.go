package render

import (
	"bytes"
	"math"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Evengelius/agent-command-center/internal/event"
	"github.com/Evengelius/agent-command-center/internal/fixture"
	"github.com/Evengelius/agent-command-center/internal/provider/claude"
	"github.com/Evengelius/agent-command-center/internal/provider/codex"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// Plan 69, criterion 69.7: zone A's telemetry after a resume from the home
// screen, P69a's and P69b's lines through the mappers, the state and the
// pane, once the resume is answered and after the first new turn.

// usageAs is a mapper whose "/usage" lines go in as the dashboard writes
// them, its own read (plan 24).
type usageAs struct {
	fixture.Mapper
	internal func([]byte) []event.Event
}

func (m usageAs) In(line []byte) []event.Event {
	if m.internal != nil && bytes.Contains(line, []byte(`"content":"/usage"`)) {
		return m.internal(line)
	}
	return m.Mapper.In(line)
}

// resumedState is the state after recs, up to the first record at or after
// stop ms (0: all), in a session started from the home screen to resume
// asked ("" for a new session's start); the frame time is the last event's.
func resumedState(provider string, m fixture.Mapper, internal func([]byte) []event.Event, recs []fixture.Record, asked string, stop int64) (*state.State, time.Time) {
	var keep []fixture.Record
	for _, r := range recs {
		if stop > 0 && r.TMs > stop {
			break
		}
		if r.Dir != "exit" {
			keep = append(keep, r)
		}
	}
	s := state.New()
	s.Context.Provider = provider
	s.Session.Resumed = asked
	for _, ev := range fixture.Flatten(fixture.Replay(provider, keep, usageAs{m, internal})) {
		s.Apply(ev)
	}
	now := fixture.Epoch
	if len(s.Log) > 0 {
		now = s.Log[len(s.Log)-1].At
	}
	return s, now
}

// firstAt is the time of the first record of proc whose message holds text.
func firstAt(t *testing.T, recs []fixture.Record, dir, text string) int64 {
	t.Helper()
	for _, r := range recs {
		if r.Dir == dir && strings.Contains(string(r.Msg), text) {
			return r.TMs
		}
	}
	t.Fatalf("no %s record with %s", dir, text)
	return 0
}

// pane is zone A's usage values: "context …", "tokens …" and "cost …".
func pane(s *state.State, now time.Time) string {
	d := teleSummary(s, now)
	var out []string
	for _, l := range strings.Split(d, "\n") {
		for _, k := range []string{"context ", "tokens ", "cost "} {
			if strings.HasPrefix(l, k) {
				out = append(out, l)
			}
		}
	}
	return strings.Join(out, "\n")
}

// TestResumeNoteTarget is criterion 69.4's click in picture mode: the last
// session's note is a click target on the row of its line's middle and on
// no row its line does not touch, at every size and however many notes
// grow the panels. The Windows Terminal check at 120x40 found a line's own
// band, less than a row high, rounding to no row.
func TestResumeNoteTarget(t *testing.T) {
	n := 0
	for w := 100; w <= 240; w += 2 {
		for h := 30; h <= 70; h++ {
			for _, extra := range []string{"", "http://127.0.0.1:3128", strings.Repeat("a reason that wraps ", 6)} {
				s, ui := homeTestUI()
				ui.Width, ui.Height = w, h
				ui.LastSession = "Claude Code's last session here: now · Ctrl+R continues it"
				ui.CodexProxy = extra
				g := HomeLayout(w, h, true)
				if !g.Picture {
					continue
				}
				l := homeLook(s, ui, g)
				var rows []int
				for _, tg := range homeTargets(l) {
					if tg.item == HomeItemResume {
						for y := tg.r.Min.Y; y < tg.r.Max.Y; y++ {
							rows = append(rows, y)
						}
					}
				}
				if mid := int(math.Floor(g.Y(634.75-l.Grow) / SixelCellH)); !slices.Contains(rows, mid) {
					t.Errorf("%dx%d, %d more characters of notes: the note's row %d, targets %v", w, h, len(extra), mid, rows)
				}
				top, bottom := g.Y(634.75-l.Grow-noteLine/2), g.Y(634.75-l.Grow+noteLine/2)
				for _, r := range rows {
					if float64(r*SixelCellH) >= bottom || float64((r+1)*SixelCellH) <= top {
						t.Errorf("%dx%d, %d more characters of notes: row %d is a target the line does not touch", w, h, len(extra), r)
					}
				}
				n++
			}
		}
	}
	if n < 1000 {
		t.Fatalf("%d picture-mode cases", n)
	}
}

// TestResumeNoteValidating is criterion 69.4's "none while a start is
// validated": the note offers the last session while the owner chooses,
// and not while the start it began is validated.
func TestResumeNoteValidating(t *testing.T) {
	s, ui := homeTestUI()
	ui.LastSession = "Claude Code's last session here: now · Ctrl+R continues it"
	for _, c := range []struct {
		phase state.StartupPhase
		want  bool
	}{{state.Asking, true}, {state.Validating, false}} {
		s.Startup.Phase = c.phase
		offered := false
		for _, n := range homeNotes(s, ui) {
			if n.Item == HomeItemResume && n.Text == ui.LastSession {
				offered = true
			}
		}
		if offered != c.want {
			t.Errorf("%s: the note offered %t, want %t", c.phase, offered, c.want)
		}
	}
}

// TestNotResumedNote is criterion 69.8's home note: the last resume's
// failure a failure note on the home screen, under the offer when the
// session stays offered.
func TestNotResumedNote(t *testing.T) {
	s, ui := homeTestUI()
	ui.LastSession = "Claude Code's last session here: 5m ago · Ctrl+R continues it"
	ui.NotResumed = "Not resumed: the CLI exited unexpectedly"
	offer, failure := -1, -1
	for i, n := range homeNotes(s, ui) {
		switch {
		case n.Item == HomeItemResume:
			offer = i
		case n.Mark == "✗ " && n.Text == ui.NotResumed && n.Kind == 'f' && n.Item == 0:
			failure = i
		}
	}
	if offer < 0 || failure != offer+1 {
		t.Errorf("the offer at %d, the failure at %d: %+v", offer, failure, homeNotes(s, ui))
	}
}

// TestResumingLine is criterion 69.6's starting line: while the CLI starts
// on a session resumed from the home screen, zone B names the session or
// thread, its ID cut as zone B's note cuts it; a new session's line as
// before.
func TestResumingLine(t *testing.T) {
	for _, c := range []struct{ cli, name, resumed, want string }{
		{claude.ProviderID, "Claude Code", "21f3b775-162f-47e1-bb16-aa91f8a808b9", `Resuming Claude Code session 21f3b775 in C:\work\demo-project…`},
		{codex.ProviderID, "OpenAI Codex", "01a18936-726b-71c2-bda9-13b405475191", `Resuming OpenAI Codex thread 01a18936 in C:\work\demo-project…`},
		{claude.ProviderID, "Claude Code", "", `Starting Claude Code in C:\work\demo-project…`},
	} {
		for _, phase := range []state.SessionPhase{state.Spawning, state.Handshaking} {
			s := state.New()
			s.Screen = state.ScreenDashboard
			s.Context.Provider, s.Context.ProviderName, s.Context.Project = c.cli, c.name, `C:\work\demo-project`
			s.Startup.Phase = state.Starting
			s.Session.Phase, s.Session.Resumed = phase, c.resumed
			if got := sessionLines(s); !slices.Equal(got, []string{c.want}) {
				t.Errorf("%s, %s: %q, want %q", c.name, phase, got, c.want)
			}
		}
	}
}

// TestResumedUsageRows is criterion 69.7 on the screen, as the live run of
// 69.11 found it: at 208x59, where the usage section draws three columns,
// a resumed session's reasons would push the cost out of them; the section
// takes its three rows instead, every value and reason whole, until the
// values come; a new session's columns as before.
func TestResumedUsageRows(t *testing.T) {
	// usage is the section's three rows under its heading, three empty rows
	// when there is none.
	usage := func(s *state.State, now time.Time) []string {
		rows, _ := zoneARows(s, UI{Width: 208, Height: 59, Now: now, Focus: "B"})
		if i := indexOf(rows, "▾ SESSION USAGE"); i >= 0 && i+4 <= len(rows) {
			return rows[i+1 : i+4]
		}
		return make([]string, 3)
	}
	columns := regexp.MustCompile(`^  CONTEXT\s+│ TOKENS\s+│ EST\. COST$`)
	t.Run("Claude Code", func(t *testing.T) {
		resume := fixture.Of(records(t, "p69a-claude-resume.jsonl"), "resume")
		read := firstAt(t, resume, "out", `"type":"result"`)
		var turned int64
		for _, r := range resume {
			if r.Dir == "out" && strings.Contains(string(r.Msg), `"type":"result"`) {
				turned = r.TMs
			}
		}
		const session = "21f3b775-162f-47e1-bb16-aa91f8a808b9"
		m := claude.NewMapper()
		want := []string{"  Context N/A no turn since the resume", "  Tokens N/A no result since the resume", "  Est. cost N/A no result since the resume"}
		if got := usage(resumedState(claude.ProviderID, m, m.InInternal, resume, session, read)); !slices.Equal(got, want) {
			t.Errorf("the usage read answered:\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
		}
		m = claude.NewMapper()
		if got := usage(resumedState(claude.ProviderID, m, m.InInternal, resume, session, turned)); !columns.MatchString(got[0]) || !strings.Contains(got[1], "4% of 1M") {
			t.Errorf("after the first new turn:\n%s", strings.Join(got, "\n"))
		}
		m = claude.NewMapper()
		if got := usage(resumedState(claude.ProviderID, m, m.InInternal, resume, "", read)); !columns.MatchString(got[0]) || !strings.Contains(got[2], "no turn yet") {
			t.Errorf("a new session's columns changed:\n%s", strings.Join(got, "\n"))
		}
	})
	t.Run("Codex", func(t *testing.T) {
		const thread = "01a18936-726b-71c2-bda9-13b405475191"
		exclude := fixture.Of(records(t, "p69b-codex-resume.jsonl"), "exclude")
		answered := firstAt(t, exclude, "out", `"thread/tokenUsage/updated"`)
		if got := usage(resumedState(codex.ProviderID, codex.NewMapper(), nil, exclude, thread, answered)); !columns.MatchString(got[0]) || !strings.Contains(got[1], "6% of 258.4k") || !strings.Contains(got[2], "Codex reports none") {
			t.Errorf("answered with its usage replay:\n%s", strings.Join(got, "\n"))
		}
		var cut []fixture.Record
		for _, r := range exclude {
			if r.TMs != answered || !strings.Contains(string(r.Msg), `"thread/tokenUsage/updated"`) {
				cut = append(cut, r)
			}
		}
		want := []string{"  Context N/A no update since the resume", "  Tokens N/A no update since the resume", "  Est. cost N/A Codex reports none"}
		if got := usage(resumedState(codex.ProviderID, codex.NewMapper(), nil, cut, thread, answered)); !slices.Equal(got, want) {
			t.Errorf("answered without its usage replay:\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
		}
	})
}

func TestResumedTelemetry(t *testing.T) {
	const thread = "01a18936-726b-71c2-bda9-13b405475191"
	recs := func(proc string) []fixture.Record { return fixture.Of(records(t, "p69b-codex-resume.jsonl"), proc) }
	t.Run("Codex without its turns, as with them", func(t *testing.T) {
		exclude, full := recs("exclude"), recs("full")
		answered := firstAt(t, exclude, "out", `"thread/tokenUsage/updated"`)
		want := "context 6% of 258.4k\ntokens 16.4k\ncost N/A (Codex reports none)"
		if got := pane(resumedState(codex.ProviderID, codex.NewMapper(), nil, exclude, thread, answered)); got != want {
			t.Errorf("answered, without its turns:\n%s\nwant\n%s", got, want)
		}
		if got := pane(resumedState(codex.ProviderID, codex.NewMapper(), nil, full, thread, firstAt(t, full, "out", `"thread/tokenUsage/updated"`))); got != want {
			t.Errorf("answered, with its turns:\n%s\nwant\n%s", got, want)
		}
		s, now := resumedState(codex.ProviderID, codex.NewMapper(), nil, exclude, thread, firstAt(t, exclude, "out", `"turn/completed"`))
		if got, want := pane(s, now), "context 6% of 258.4k\ntokens 32.9k\ncost N/A (Codex reports none)"; got != want {
			t.Errorf("after the first new turn:\n%s\nwant\n%s", got, want)
		}
		if v := s.Metrics["tokens.total"]; v.V != 32897 {
			t.Errorf("the thread's total %v, want 32,897: 16,430 before and 16,467 for the turn", v.V)
		}
		if v := s.Metrics["duration.ms"]; !v.Present {
			t.Error("no turn duration after the turn")
		}
	})
	t.Run("Codex without its usage replay", func(t *testing.T) {
		exclude := recs("exclude")
		answered := firstAt(t, exclude, "out", `"thread/tokenUsage/updated"`)
		var cut []fixture.Record
		for _, r := range exclude {
			if r.TMs != answered || !strings.Contains(string(r.Msg), `"thread/tokenUsage/updated"`) {
				cut = append(cut, r)
			}
		}
		want := "context N/A (no update since the resume)\ntokens N/A (no update since the resume)\ncost N/A (Codex reports none)"
		if got := pane(resumedState(codex.ProviderID, codex.NewMapper(), nil, cut, thread, answered)); got != want {
			t.Errorf("answered:\n%s\nwant\n%s", got, want)
		}
		if got := pane(resumedState(codex.ProviderID, codex.NewMapper(), nil, cut, thread, firstAt(t, cut, "out", `"turn/completed"`))); !strings.Contains(got, "tokens 32.9k") {
			t.Errorf("after the first new turn:\n%s", got)
		}
		if got := pane(resumedState(codex.ProviderID, codex.NewMapper(), nil, cut, "", answered)); !strings.Contains(got, "context N/A (no model call yet)") {
			t.Errorf("a new session's reason changed:\n%s", got)
		}
	})
	t.Run("Claude Code", func(t *testing.T) {
		resume := fixture.Of(records(t, "p69a-claude-resume.jsonl"), "resume")
		read := firstAt(t, resume, "out", `"type":"result"`)
		var turned int64
		for _, r := range resume {
			if r.Dir == "out" && strings.Contains(string(r.Msg), `"type":"result"`) {
				turned = r.TMs
			}
		}
		const session = "21f3b775-162f-47e1-bb16-aa91f8a808b9"
		m := claude.NewMapper()
		want := "context N/A (no turn since the resume)\ntokens N/A (no result since the resume)\ncost N/A (no result since the resume)"
		if got := pane(resumedState(claude.ProviderID, m, m.InInternal, resume, session, read)); got != want {
			t.Errorf("the usage read answered:\n%s\nwant\n%s", got, want)
		}
		m = claude.NewMapper()
		if got, want := pane(resumedState(claude.ProviderID, m, m.InInternal, resume, session, turned)), "context 4% of 1M\ntokens 85.3k\ncost $0.01"; got != want {
			t.Errorf("after the first new turn:\n%s\nwant\n%s", got, want)
		}
		m = claude.NewMapper()
		if got, want := pane(resumedState(claude.ProviderID, m, m.InInternal, resume, "", read)), "context N/A (no turn yet)\ntokens N/A (no result yet)\ncost N/A (no result yet)"; got != want {
			t.Errorf("a new session's reasons:\n%s\nwant\n%s", got, want)
		}
	})
}
