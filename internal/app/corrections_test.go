package app

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/Evengelius/agent-command-center/internal/event"
	"github.com/Evengelius/agent-command-center/internal/provider/claude"
	"github.com/Evengelius/agent-command-center/internal/provider/codex"
	"github.com/Evengelius/agent-command-center/internal/render"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// Plan 22, corrections C3, C6, C7, C8, C9 and C10 at the application (plan.md,
// Corrections C1 to C12; doc/plan22-investigation.md, section 2): the
// recordings of both CLIs replayed with a frame every 50 ms, the owner's
// prompt, the entry cursor's reveal.

// C3: a prompt sent while the turn is Idle ends a phase still live, so no 50 ms
// frame is asked for once its exit is over: before, a thinking block opened
// outside a turn (P12c-block's end, a late content_block_start) and the next
// turn's events and end left it live, and the application asked for a frame
// every 50 ms for the rest of the session (reliability.md, test 9).
func TestCorrectionC3NoFramesAfterStrayPhase(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := thinkStart(t, "p12c-block-claude-auto-boundary-edit.jsonl", claude.ProviderID)
		r.h.draw(tea.WindowSizeMsg{Width: 120, Height: 40})
		r.until(26400) // after the result (26,339 ms), before the recorded process exit
		inject := func(ms int, kind event.Kind, payload any) {
			*r.now = thinkAt(ms)
			r.h.draw(eventsMsg{sess: r.h.m.sess, evs: []event.Event{{Seq: uint64(100000 + ms), At: thinkAt(ms), Provider: claude.ProviderID, Kind: kind, Origin: event.Observed, Payload: payload}}})
		}
		inject(26500, event.MessageMarker, event.Marker{MessageID: "msg_late", Name: "content_block_start", Block: "thinking", Index: 0})
		if th := thought(t, r.h.m.State(), 5); !th.Thought.Live() {
			t.Fatalf("not the scenario: the late block is not live: %+v", th.Thought)
		}
		// The owner sends the next prompt; the turn runs and ends.
		r.h.draw(tea.PasteMsg{Content: "next"})
		r.h.draw(key(tea.KeyEnter, 0))
		if th := thought(t, r.h.m.State(), 5); th.Thought.Live() || !th.Thought.EndInferred || th.Streaming {
			t.Errorf("the prompt left the stray phase live: %+v (streaming %v)", th.Thought, th.Streaming)
		}
		inject(60000, event.UserMessage, event.User{Text: "next"})
		inject(60001, event.TurnStarted, nil)
		inject(70000, event.TurnCompleted, event.TurnEnd{Outcome: "completed"})
		var asks []string
		r.h.m.opt.After = func(d time.Duration, msg tea.Msg) tea.Cmd {
			asks = append(asks, fmt.Sprintf("%T %v", msg, d))
			return func() tea.Msg { return nil }
		}
		r.h.m.framing, r.h.m.spinning, r.h.m.clocking = false, false, false
		for i := 0; i < 3; i++ {
			*r.now = thinkAt(3_600_000 + 50*i) // an hour later, the dashboard idle
			r.h.draw(animMsg{})
		}
		for _, e := range r.h.m.State().Conversation {
			if e.Role == "thinking" && e.Thought.Live() {
				t.Errorf("a phase is still live an hour later: %+v", e.Thought)
			}
		}
		if render.Animating(r.h.m.State(), r.h.m.ui) {
			t.Error("an idle dashboard animates")
		}
		for _, a := range asks {
			if !strings.HasPrefix(a, "app.clockMsg") {
				t.Errorf("an idle dashboard asked for %s", a)
			}
		}
	})
}

// recordedProbe is a recording of either CLI and the CLI that wrote it.
type recordedProbe struct{ path, id string }

// everyRecording lists the driver recordings of both CLIs in testdata/probes
// and doc/research/probes, once each by name: a Codex one is named for it. The
// one-shot `codex exec` recordings are left out: the dashboard drives Codex
// through its app-server.
func everyRecording(t *testing.T) []recordedProbe {
	t.Helper()
	seen := map[string]bool{}
	var out []recordedProbe
	for _, dir := range []string{filepath.Join("..", "..", "testdata", "probes"), filepath.Join("..", "..", "doc", "research", "probes")} {
		files, err := os.ReadDir(dir)
		if err != nil {
			t.Fatal(err)
		}
		for _, f := range files {
			n := f.Name()
			if !strings.HasSuffix(n, ".jsonl") || strings.Contains(n, "exec") || seen[n] {
				continue
			}
			seen[n] = true
			id := claude.ProviderID
			if strings.Contains(n, "codex") {
				id = codex.ProviderID
			}
			out = append(out, recordedProbe{filepath.Join(dir, n), id})
		}
	}
	return out
}

// activityRow is the activity line's first row: a spinner, an expression and
// the turn's time with nothing between (a card's rows sit inside their border,
// and a 2-row card's carry its count before the time).
var activityRow = regexp.MustCompile(`▎ [◜◠◝◞◡◟◌] (Crunching|Vibing|Connecting dots|Smoothing|Pondering|Untangling|Brewing ideas|Cooking|Piecing it together|Making magic)\.\.\. +(--:--|[0-9]{2}:[0-9]{2}|[0-9]+:[0-9]{2}:[0-9]{2})( |$|│)`)

// exitRow is a card's exit head at 80x24, the 2-row card: the mark, the
// summary's words, the count, the time it took.
var exitRow = regexp.MustCompile(`▎ [✓✗] (Thought|Thinking)[^▸]*$`)

// summaryLine is a phase's summary row.
var summaryLine = regexp.MustCompile(`▸ [✓✗] (Thought|Thinking)`)

// plainFast is plainText for a frame, which holds only SGR sequences, without
// the regular expression: much faster, which a replay of every recording at
// two sizes needs. watch checks it against plainText on its first frames.
func plainFast(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); {
		if s[i] != 0x1b {
			b.WriteByte(s[i])
			i++
			continue
		}
		i++
		if i >= len(s) {
			break
		}
		switch s[i] {
		case '[': // a control sequence: parameters, then a final byte from @ to ~
			for i++; i < len(s) && (s[i] < 0x40 || s[i] > 0x7e); i++ {
			}
			i++
		case ']': // an operating system command, up to BEL or ESC \
			for i++; i < len(s) && s[i] != 0x07 && !(s[i] == 0x1b && i+1 < len(s) && s[i+1] == '\\'); i++ {
			}
			if i < len(s) && s[i] == 0x07 {
				i++
			} else {
				i += 2
			}
		default:
			i++
		}
	}
	return b.String()
}

// watch checks every frame of a replay against the corrections of the
// activity line (C6), the 2-row card's exit (C7, C8) and the summary row at 80
// columns (C9).
type watch struct {
	t      *testing.T
	name   string
	r      *thinkSession
	narrow bool      // 80x24: the 2-row card, rows of zone B alone
	began  time.Time // when the activity line's condition began, by the test's own reading
	// What the replay showed, to know the check saw something.
	frames, held, drawn, shortRuns int
	errs                           int
}

func (w *watch) errorf(format string, args ...any) {
	w.t.Helper()
	if w.errs++; w.errs <= 8 {
		w.t.Errorf("%s: "+format, append([]any{w.name}, args...)...)
	}
}

// frame is called with every frame the application draws.
func (w *watch) frame(content string) {
	w.t.Helper()
	m, now := w.r.h.m, *w.r.now
	s := m.State()
	w.frames++
	var text string
	plain := func() string {
		if text == "" {
			text = plainFast(content)
			if w.frames <= 20 && text != plainText(content) {
				w.errorf("plainFast differs from plainText on frame %d", w.frames)
			}
		}
		return text
	}

	// C6: the activity line is drawn only once its condition has held 200 ms,
	// counted from the first frame that saw it hold, and then it is drawn.
	cond := render.ActivityOn(s, m.ui)
	switch {
	case !cond:
		if !w.began.IsZero() && now.Sub(w.began) < 200*time.Millisecond {
			w.shortRuns++ // a gap the line would have flickered in
		}
		w.began = time.Time{}
	case w.began.IsZero():
		w.began = now
	}
	if cond {
		w.held++
		since := now.Sub(w.began)
		// Every expression ends in "..."; a frame without it has no line.
		line := strings.Contains(content, "...") && activityRow.MatchString(plain())
		switch {
		case line && since < 200*time.Millisecond:
			w.errorf("the activity line is drawn %v after its condition began (at %v):\n%s", since, now.Sub(thinkAt(0)), plain())
		case !line && since >= 200*time.Millisecond:
			w.errorf("the activity line is not drawn %v after its condition began (at %v):\n%s", since, now.Sub(thinkAt(0)), plain())
		}
		if line {
			w.drawn++
		}
	}

	// C8: in a card's exit, Claude Code's count is the exact count once it came
	// and never "live estimate"; "token usage pending" never once a count came or
	// can no longer come (a message cut short, a turn or session ended).
	exiting, live := 0, 0
	settled := false
	for _, e := range s.Conversation {
		if e.Role != "thinking" || e.Thought == nil {
			continue
		}
		x := e.Thought
		if x.Live() && !x.Dropped {
			live++
		} else if m.ui.Live[uint64(x.ID)] && !x.End.IsZero() && now.Sub(x.End) < 200*time.Millisecond && !x.Inferred {
			exiting++
			settled = settled || x.HasTokens || x.Cut || x.Closed
		}
	}
	if exiting > 0 && live == 0 {
		if strings.Contains(content, "live estimate") {
			w.errorf("a phase's exit says \"live estimate\":\n%s", plain())
		}
		if settled && strings.Contains(content, "token usage pending") {
			w.errorf("a phase's exit says \"token usage pending\" with its count reported or no longer to come:\n%s", plain())
		}
	}

	// C7 and C9, at 80x24: no row of a phase's exit, no summary row, is cut.
	if !w.narrow || !strings.Contains(content, "Thought") && !strings.Contains(content, "Thinking") {
		return
	}
	for _, l := range strings.Split(plain(), "\n") {
		switch {
		case summaryLine.MatchString(l) && strings.Contains(l, "…"):
			w.errorf("a phase's summary row is cut: %q", l)
		case exitRow.MatchString(l) && !summaryLine.MatchString(l) && strings.Contains(l, "…"):
			w.errorf("a phase's exit row is cut: %q", l)
		}
	}
}

// C6, C7, C8 and C9 on every recording of both CLIs, replayed with a frame
// every 50 ms (and a frame at each recorded moment): no frame draws the
// activity line less than 200 ms after its condition began, in the gaps the
// investigation lists (the Codex turn ends, 27 to 145 ms; Claude Code's, 0 to
// 27 ms; before and between step cards; around approvals); and every frame
// from +200 ms draws it. At 80x24 no exit row and no summary row of a phase is
// cut, and no exit says "live estimate", or "token usage pending" once its count
// is reported or can no longer come.
func TestCorrectionsEveryRecording(t *testing.T) {
	var total watch
	for _, p := range everyRecording(t) {
		for _, size := range [][2]int{{120, 40}, {80, 24}} {
			name := fmt.Sprintf("%s-%dx%d", filepath.Base(p.path), size[0], size[1])
			t.Run(name, func(t *testing.T) {
				synctest.Test(t, func(t *testing.T) {
					r := thinkStartFile(t, p.path, p.id)
					r.h.draw(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
					w := &watch{t: t, name: name, r: r, narrow: size[0] < 100}
					r.onFrame = w.frame
					end := int(r.recs[len(r.recs)-1].TMs) + 3000
					for ms := 0; ms <= end; ms += 50 {
						before := r.next
						r.until(ms)
						if m := r.h.m; r.next == before && m.State().Turn.Phase == state.Idle && !render.Animating(m.State(), m.ui) {
							continue // nothing came and nothing moves: the application draws no frame (reliability.md, test 9)
						}
						r.at(ms)
					}
					total.frames += w.frames
					total.held += w.held
					total.drawn += w.drawn
					total.shortRuns += w.shortRuns
				})
			})
		}
	}
	// The check saw something: frames in which the line was due and drawn, and
	// the gaps it must stay out of.
	t.Logf("%d frames, %d with the activity line's condition, %d drawing it, %d gaps shorter than 200 ms", total.frames, total.held, total.drawn, total.shortRuns)
	if total.drawn < 200 || total.shortRuns < 40 {
		t.Errorf("%d frames drew the line and %d gaps were crossed: the check saw too little", total.drawn, total.shortRuns)
	}
}

// C6, the boundary on P12c-block: between its second phase's exit and the next
// phase, the model at work with no card, the activity line is absent from the
// first frame that sees its condition hold to +150 ms and drawn at the first
// frame from +200 ms.
func TestCorrectionC6Boundary(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := thinkStart(t, "p12c-block-claude-auto-boundary-edit.jsonl", claude.ProviderID)
		r.h.draw(tea.WindowSizeMsg{Width: 120, Height: 40})
		type frame struct {
			at         time.Time
			cond, line bool
		}
		var frames []frame
		r.onFrame = func(content string) {
			frames = append(frames, frame{*r.now, render.ActivityOn(r.h.m.State(), r.h.m.ui), strings.Contains(content, "...") && activityRow.MatchString(plainFast(content))})
		}
		for ms := 0; ms <= 22500; ms += 50 {
			r.until(ms)
			r.at(ms)
		}
		// The run of frames in which the condition holds, the first after the second
		// phase ended at 21,351 ms.
		start := -1
		for i, f := range frames {
			if f.cond && (i == 0 || !frames[i-1].cond) && f.at.After(thinkAt(21351)) {
				start = i
				break
			}
		}
		if start < 0 {
			t.Fatal("the condition never held after the second phase's end")
		}
		began, near, after := frames[start].at, 0, 0
		for i := start; i < len(frames) && frames[i].cond; i++ {
			d := frames[i].at.Sub(began)
			if want := d >= 200*time.Millisecond; frames[i].line != want {
				t.Errorf("+%v after the condition began (at %v): the activity line drawn %v, want %v", d, frames[i].at.Sub(thinkAt(0)), frames[i].line, want)
			}
			switch {
			case d >= 150*time.Millisecond && d < 200*time.Millisecond:
				near++
			case d >= 200*time.Millisecond:
				after++
			}
		}
		if near == 0 || after == 0 {
			t.Errorf("the run began at %v: %d frames in [150, 200) ms and %d from 200 ms; want both", began.Sub(thinkAt(0)), near, after)
		}
	})
}

// C6, after the owner's prompt: not before +200 ms. The condition begins when
// the prompt is sent (the turn Submitted, the earlier answer last); the
// application asks for no timer of its own for it.
func TestCorrectionC6AfterPrompt(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := thinkStart(t, "p12c-block-claude-auto-boundary-edit.jsonl", claude.ProviderID)
		r.h.draw(tea.WindowSizeMsg{Width: 120, Height: 40})
		r.until(26400) // the recorded session at rest, its last turn over
		asks := map[string]int{}
		r.h.m.opt.After = func(d time.Duration, msg tea.Msg) tea.Cmd {
			asks[fmt.Sprintf("%T %v", msg, d)]++
			return func() tea.Msg { return nil } // asked for, never delivered: the test delivers the frames
		}
		r.h.m.framing, r.h.m.spinning, r.h.m.clocking = false, false, false
		const t0 = 40000
		*r.now = thinkAt(t0)
		r.h.draw(tea.PasteMsg{Content: "next"})
		r.h.draw(key(tea.KeyEnter, 0))
		if r.h.m.State().Turn.Phase != state.Submitted {
			t.Fatalf("turn %q after Enter, want Submitted", r.h.m.State().Turn.Phase)
		}
		echo := func(ms int, kind event.Kind, payload any) {
			*r.now = thinkAt(ms)
			r.draw(eventsMsg{sess: r.h.m.sess, evs: []event.Event{{Seq: uint64(200000 + ms), At: thinkAt(ms), Provider: claude.ProviderID, Kind: kind, Origin: event.Observed, Payload: payload}}})
		}
		drawn := func(ms int) bool {
			*r.now = thinkAt(ms)
			r.h.draw(animMsg{})
			return activityRow.MatchString(plainText(r.h.m.View().Content))
		}
		echo(t0+10, event.UserMessage, event.User{Text: "next"})
		echo(t0+20, event.TurnStarted, nil)
		for _, c := range []struct {
			d    int
			want bool
		}{{50, false}, {100, false}, {150, false}, {199, false}, {200, true}, {250, true}, {300, true}} {
			if got := drawn(t0 + c.d); got != c.want {
				t.Errorf("%d ms after the prompt: the activity line drawn %v, want %v", c.d, got, c.want)
			}
		}
		// No timer of its own: only the 50 ms frames of a card, the 100 ms
		// spinner and the 1 s clock were asked for.
		for a := range asks {
			switch a {
			case "app.animMsg 50ms", "app.spinMsg 100ms", "app.clockMsg 1s":
			default:
				t.Errorf("asked for %s", a)
			}
		}
		if asks["app.spinMsg 100ms"] == 0 {
			t.Errorf("the 100 ms spinner was not asked for while the turn is under way: %v", asks)
		}
	})
}

// C10: with zone B held and only a Codex step that did not reason new, no "↓ N
// new" badge shows (render.NewShown is 0), and revealing an entry already in
// view scrolls nothing: reveal kept a row free for a badge that is not shown,
// and moved the view by a row (ConvTop 217 to 218) for an entry ending on the
// last row. With a new entry zone B does draw, the badge shows and the row is
// kept free, as before.
func TestCorrectionC10RevealCountsWhatTheBadgeCounts(t *testing.T) {
	// run holds the view up rows, then lets one new entry come: a step that did
	// not reason (dropped by its turn's end), or an answer.
	run := func(t *testing.T, up int, answer bool) (moved, checked int) {
		synctest.Test(t, func(t *testing.T) {
			h, cli := startScript(t, codex.ProviderID)
			h.send(tea.WindowSizeMsg{Width: 120, Height: 40})
			now := testNow
			h.m.opt.Now = func() time.Time { return now }
			seq := uint64(9000)
			apply := func(d time.Duration, k event.Kind, p any) {
				seq++
				now = now.Add(d)
				h.draw(eventsMsg{sess: h.m.sess, evs: []event.Event{{Seq: seq, At: now, Provider: cli.id, Kind: k, Origin: event.Inferred, Payload: p}}})
			}
			apply(0, event.UserMessage, event.User{Text: "go"})
			apply(10*time.Millisecond, event.TurnStarted, nil)
			for i := 0; i < 40; i++ {
				id := fmt.Sprintf("c%d", i)
				apply(100*time.Millisecond, event.ToolStarted, event.Tool{ToolID: id, Name: "commandExecution", Input: []byte(`{"command":"ls"}`)})
				apply(100*time.Millisecond, event.ToolCompleted, event.Tool{ToolID: id, Name: "commandExecution", Output: "x"})
				apply(10*time.Millisecond, event.MessageCompleted, event.Completed{MessageID: fmt.Sprintf("m%d", i), Text: "line one\nline two\nline three"})
			}
			h.m.scroll(-up) // held, a little up
			h.draw(animMsg{})
			seen := h.m.ui.ConvSeen
			if !h.m.ui.ConvHold || seen != len(h.m.State().Conversation) {
				t.Fatalf("not held: hold %v, seen %d of %d", h.m.ui.ConvHold, seen, len(h.m.State().Conversation))
			}
			if answer {
				apply(50*time.Millisecond, event.MessageCompleted, event.Completed{MessageID: "late", Text: "one more answer"})
			} else {
				apply(50*time.Millisecond, event.MessageMarker, event.Marker{Name: event.StepStarted})
				apply(500*time.Millisecond, event.TurnCompleted, event.TurnEnd{Outcome: "completed"})
			}
			now = now.Add(time.Second)
			h.draw(animMsg{})
			s := h.m.State()
			badge := strings.Contains(h.frame(), " new")
			if n := len(s.Conversation) - seen; n != 1 {
				t.Fatalf("%d entries new, want 1", n)
			}
			if shown := render.NewShown(s, seen); (shown == 1) != answer || badge != answer {
				t.Fatalf("answer %v: NewShown %d, badge drawn %v", answer, shown, badge)
			}
			c := h.m.screen.Conv
			top := min(max(h.m.ui.ConvTop, 0), max(c.Rows-c.Room, 0))
			targets := render.Targets(s, h.m.ui)
			for n := 1; n <= len(s.Conversation); n++ {
				// An entry in view that ends on the view's last row: the one the
				// kept-free row would push up.
				if !targets[n-1] || c.Starts[n-1] < top || c.Starts[n] != top+c.Room {
					continue
				}
				checked++
				before := h.m.ui.ConvTop
				h.m.reveal(n)
				if h.m.ui.ConvTop != before {
					moved++
				}
				h.m.ui.ConvTop = before
			}
		})
		return moved, checked
	}
	checkedDropped := 0
	for up := 1; up <= 8; up++ {
		moved, checked := run(t, up, false)
		checkedDropped += checked
		if moved != 0 {
			t.Errorf("held %d rows up, only a dropped step new: reveal scrolled for %d of %d entries already in view", up, moved, checked)
		}
	}
	if checkedDropped == 0 {
		t.Error("no entry ending on the view's last row was tried: the check saw nothing")
	}
	// A new entry zone B draws: the badge is shown and the row stays free, so an
	// entry ending on the last row is scrolled up by it.
	movedAnswer, checkedAnswer := 0, 0
	for up := 1; up <= 8; up++ {
		moved, checked := run(t, up, true)
		movedAnswer += moved
		checkedAnswer += checked
	}
	if checkedAnswer == 0 || movedAnswer != checkedAnswer {
		t.Errorf("with a new answer and its badge, reveal scrolled for %d of %d entries ending on the last row, want all", movedAnswer, checkedAnswer)
	}
}
