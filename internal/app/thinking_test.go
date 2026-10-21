package app

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/Evengelius/agent-command-center/internal/event"
	"github.com/Evengelius/agent-command-center/internal/fixture"
	"github.com/Evengelius/agent-command-center/internal/provider/claude"
	"github.com/Evengelius/agent-command-center/internal/provider/codex"
	"github.com/Evengelius/agent-command-center/internal/render"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// Plan 22 at the application: the thinking phases replayed from recordings
// as a live run receives them, one recorded moment per update with the clock
// at its time, so the phases a frame drew live are marked (ui.Live); the
// frames, the animation's ticks, the keys.

// thinkAt is the recordings' time ms milliseconds in.
func thinkAt(ms int) time.Time { return fixture.Epoch.Add(time.Duration(ms) * time.Millisecond) }

// thinkSession is a recording replayed through a scripted session: its
// harness, the clock it reads, and the recording's events by record.
type thinkSession struct {
	h    *harness
	now  *time.Time
	recs []fixture.Record
	per  [][]event.Event
	next int // the next record to apply
	// onFrame, when set, is given every frame until and at draw (plan 22, the
	// corrections' check of every frame of a replay).
	onFrame func(frame string)
}

// draw delivers msg and draws the frame, as the harness does, handing it to
// onFrame.
func (r *thinkSession) draw(msg tea.Msg) {
	if r.onFrame == nil {
		r.h.draw(msg)
		return
	}
	// A replay of thousands of frames: the update and the view alone. The commands
	// it returns (the listener, the timers, the refresh after a turn) change
	// nothing a frame shows, and delivering them costs more than the frame.
	r.h.m.Update(msg)
	r.onFrame(r.h.m.View().Content)
}

// thinkStart opens the dashboard on a scripted session of the CLI id and
// reads probe, replayed through that CLI's own mapper.
func thinkStart(t *testing.T, probe, id string) *thinkSession {
	t.Helper()
	return thinkStartFile(t, filepath.Join("..", "..", "testdata", "probes", probe), id)
}

// thinkStartFile is thinkStart for the recording at path (plan 22, the
// corrections' test of every recording, which includes doc/research/probes).
func thinkStartFile(t *testing.T, path, id string) *thinkSession {
	t.Helper()
	recs, err := fixture.Read(path)
	if err != nil {
		t.Fatal(err)
	}
	var per [][]event.Event
	if id == codex.ProviderID {
		per = fixture.Replay(id, recs, codex.NewMapper())
	} else {
		per = fixture.Replay(id, recs, claude.NewMapper())
	}
	h, _ := startScript(t, id)
	now := testNow
	h.m.opt.Now = func() time.Time { return now }
	return &thinkSession{h: h, now: &now, recs: recs, per: per}
}

// until applies the records up to the recorded time ms, each record's events
// as one update with the clock at the record's time, as the listener hands
// them to the application.
func (r *thinkSession) until(ms int) {
	for ; r.next < len(r.recs) && r.recs[r.next].TMs <= int64(ms); r.next++ {
		evs := r.per[r.next]
		if len(evs) == 0 {
			continue
		}
		for i := range evs {
			if st, ok := evs[i].Payload.(event.Started); ok {
				// The probe ran in its own directory; a live run reports the
				// chosen one (a mismatch closes the session).
				st.Cwd = r.h.m.State().Context.Project
				evs[i].Payload = st
			}
		}
		*r.now = thinkAt(int(r.recs[r.next].TMs))
		r.draw(eventsMsg{sess: r.h.m.sess, evs: evs})
	}
}

// at moves the clock to the recorded time ms and draws a frame there.
func (r *thinkSession) at(ms int) {
	*r.now = thinkAt(ms)
	r.draw(animMsg{})
}

// thought is the conversation's thinking entry n (1-based), in order.
func thought(t *testing.T, s *state.State, n int) state.Entry {
	t.Helper()
	k := 0
	for _, e := range s.Conversation {
		if e.Role == "thinking" {
			if k++; k == n {
				return e
			}
		}
	}
	t.Fatalf("no thinking entry %d", n)
	return state.Entry{}
}

// TestThinkingFrames is criteria 2.1, 2.4, 2.11 and 2.13 drawn by the
// application, golden frames at the four sizes (plan 22, criterion 4.3):
// P12c-block's second phase live (mode B, then its scale doubled), before
// its first estimate (mode C), the four summary rows once over, one opened;
// P7's first Codex step and its thought; P303b's activity line while its
// subagent works.
func TestThinkingFrames(t *testing.T) {
	type rig struct {
		probe, id string
		until, at int
		open      int // the thinking entry whose details are open; 0 none
	}
	for _, sc := range []struct {
		name string
		r    rig
		want []string
	}{
		{"thinking-live", rig{"p12c-block-claude-auto-boundary-edit.jsonl", claude.ProviderID, 11000, 11000, 0},
			[]string{"≈ 650 thinking tokens · live estimate", " 2k", "00:03", "⠋ Thinking"}},
		{"thinking-scale", rig{"p12c-block-claude-auto-boundary-edit.jsonl", claude.ProviderID, 19900, 19900, 0},
			[]string{"≈ 2,450 thinking tokens · live estimate", " 4k"}},
		{"thinking-pending", rig{"p12c-block-claude-auto-boundary-edit.jsonl", claude.ProviderID, 7780, 7780, 0},
			[]string{"Thinking · token usage pending"}},
		{"thinking-done", rig{"p12c-block-claude-auto-boundary-edit.jsonl", claude.ProviderID, 27000, 27000, 0},
			[]string{"Thought for 2.2s · 354 thinking tokens"}},
		{"thinking-open", rig{"p12c-block-claude-auto-boundary-edit.jsonl", claude.ProviderID, 27000, 27000, 4},
			[]string{"Thought for 2.2s · 354 thinking tokens", "a thinking block, observed"}},
		{"codex-step", rig{"p7-codex-subagents.jsonl", codex.ProviderID, 5000, 5000, 0},
			[]string{"Model working · reasoning tokens reported when the step ends"}},
		{"codex-thought", rig{"p7-codex-subagents.jsonl", codex.ProviderID, 7200, 7200, 0},
			[]string{"Thought for up to 3.1s · 83 reasoning tokens"}},
		{"activity", rig{"p303b-claude-subagent-approval.jsonl", claude.ProviderID, 12000, 12000, 0},
			[]string{"◇ 1 subagent working", "Thought for 2.8s · 719 thinking tokens"}},
	} {
		for _, size := range sizes {
			w, hgt := size[0], size[1]
			name := fmt.Sprintf("%s-%dx%d", sc.name, w, hgt)
			t.Run(name, func(t *testing.T) {
				synctest.Test(t, func(t *testing.T) {
					r := thinkStart(t, sc.r.probe, sc.r.id)
					r.h.draw(tea.WindowSizeMsg{Width: w, Height: hgt})
					r.until(sc.r.until)
					if sc.r.open > 0 {
						e := thought(t, r.h.m.State(), sc.r.open)
						r.h.m.unfold(render.ThoughtKey(e), true)
						r.h.m.ui.ConvHold = false // the frame follows the end, its details in view
					}
					r.at(sc.r.at)
					frame := r.h.m.View().Content
					lines := strings.Split(frame, "\n")
					if len(lines) != hgt {
						t.Errorf("%d lines, want %d", len(lines), hgt)
					}
					for i, l := range lines {
						if n := lipgloss.Width(l); n > w {
							t.Errorf("line %d is %d columns wide, terminal %d: %q", i+1, n, w, plainText(l))
						}
					}
					text := plainText(frame)
					if w >= render.MinWidth && hgt >= 40 {
						for _, want := range sc.want {
							if !strings.Contains(text, want) {
								t.Errorf("no %q in\n%s", want, text)
							}
						}
					}
					golden(t, name, fixedProject(text))
				})
			})
		}
	}
}

// TestThinkingTicks is criterion 3.1: while a card is live the application
// asks for a frame every 50 ms, never two at once; animMsg moves only the
// clock, never the 100 ms spinner; once the phase's exit is over and the
// turn idle, neither a frame nor the spinner is asked for.
func TestThinkingTicks(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := thinkStart(t, "p12c-block-claude-auto-boundary-edit.jsonl", claude.ProviderID)
		type ask struct {
			d   time.Duration
			msg string
		}
		var asks []ask
		r.h.m.opt.After = func(d time.Duration, msg tea.Msg) tea.Cmd {
			asks = append(asks, ask{d, fmt.Sprintf("%T", msg)})
			return func() tea.Msg { return nil } // asked for, never delivered: the test delivers it
		}
		count := func(msg string, d time.Duration) int {
			n := 0
			for _, a := range asks {
				if a.msg == msg && a.d == d {
					n++
				}
			}
			return n
		}
		r.until(7448)
		asks = nil // the first phase's frames: not this check's
		r.h.m.framing, r.h.m.spinning, r.h.m.clocking = false, false, false
		r.until(7449) // the second phase's start: a card live
		if !render.Animating(r.h.m.State(), r.h.m.ui) {
			t.Fatal("a live card does not animate")
		}
		if n := count("app.animMsg", 50*time.Millisecond); n != 1 {
			t.Fatalf("%d frames of 50 ms asked for with a card live, want 1: %v", n, asks)
		}
		spin := r.h.m.ui.Spinner
		for i := 1; i <= 10; i++ {
			*r.now = thinkAt(7449 + 50*i)
			r.h.draw(animMsg{})
		}
		if n := count("app.animMsg", 50*time.Millisecond); n != 11 {
			t.Errorf("%d frames asked for after ten frames, want 11 (one at a time)", n)
		}
		if r.h.m.ui.Spinner != spin {
			t.Errorf("animMsg moved the spinner: %d, want %d", r.h.m.ui.Spinner, spin)
		}
		if !r.h.m.ui.Live[uint64(thought(t, r.h.m.State(), 2).Thought.ID)] {
			t.Error("the live phase is not marked as drawn live")
		}
		// Between two phases, the Glob done and the model at work, the count's
		// highlight over: no card, so no 50 ms frame, but the turn runs: the
		// 100 ms spinner (the review: no test asked for it).
		r.until(22400)
		r.at(22400)
		asks = nil
		r.h.m.framing, r.h.m.spinning = false, false
		r.h.draw(spinMsg{})
		if count("app.spinMsg", spinnerEvery) != 1 || count("app.animMsg", 50*time.Millisecond) != 0 {
			t.Errorf("a turn at work with no card: %v asked for, want the 100 ms spinner alone", asks)
		}
		// The turn's end, then the exit over: no frame asked for any more, and
		// no spinner either, the turn idle.
		r.until(26400)
		r.at(27000)
		if r.h.m.State().Turn.Phase != state.Idle {
			t.Fatalf("turn %q after the recording", r.h.m.State().Turn.Phase)
		}
		asks = nil
		r.h.m.framing, r.h.m.spinning = false, false
		r.h.draw(animMsg{})
		r.h.draw(spinMsg{})
		if n := count("app.animMsg", 50*time.Millisecond) + count("app.spinMsg", spinnerEvery); n != 0 {
			t.Errorf("idle after the turn: %v asked for, want no frame and no spinner", asks)
		}
	})
}

// TestIdleDrawsNothing is reliability.md's frame-count test (test 9, ASM-18;
// plan 22, criterion 3.2): a dashboard whose session is ready and whose turn
// is idle, with no transition left, asks for no frame and no spinner; only
// the 1 s clock ticks, for the ages zone D shows.
func TestIdleDrawsNothing(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := thinkStart(t, "p12c-block-claude-auto-boundary-edit.jsonl", claude.ProviderID)
		r.until(30000)
		r.at(31000)
		var asks []string
		r.h.m.opt.After = func(d time.Duration, msg tea.Msg) tea.Cmd {
			asks = append(asks, fmt.Sprintf("%T %v", msg, d))
			return func() tea.Msg { return nil }
		}
		r.h.m.framing, r.h.m.spinning, r.h.m.clocking = false, false, false
		for i := 0; i < 5; i++ {
			*r.now = thinkAt(31000 + 1000*i)
			r.h.draw(clockMsg{})
		}
		for _, a := range asks {
			if !strings.HasPrefix(a, "app.clockMsg") {
				t.Errorf("an idle dashboard asked for %s", a)
			}
		}
		if len(asks) != 5 {
			t.Errorf("%d asks, want the clock's 5: %v", len(asks), asks)
		}
	})
}

// TestThinkingHoldsView is criterion 2.15: B scrolled up while a card
// animates, 50 frames 50 ms apart change none of the focus, the held view's
// top, the hold, the entry cursor.
func TestThinkingHoldsView(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := thinkStart(t, "p12c-block-claude-auto-boundary-edit.jsonl", claude.ProviderID)
		r.h.draw(tea.WindowSizeMsg{Width: 120, Height: 30})
		r.until(11000)
		r.h.draw(key(tea.KeyPgUp, 0))
		if !r.h.m.ui.ConvHold {
			t.Fatalf("PgUp did not hold B: rows %d, room %d", r.h.m.screen.Conv.Rows, r.h.m.screen.Conv.Room)
		}
		focus, top, hold, entry := r.h.m.ui.Focus, r.h.m.ui.ConvTop, r.h.m.ui.ConvHold, r.h.m.ui.Entry
		for i := 1; i <= 50; i++ {
			r.at(11000 + 50*i)
			if r.h.m.ui.Focus != focus || r.h.m.ui.ConvTop != top || r.h.m.ui.ConvHold != hold || r.h.m.ui.Entry != entry {
				t.Fatalf("frame %d: focus %q, top %d, hold %v, entry %d; want %q, %d, %v, %d", i, r.h.m.ui.Focus, r.h.m.ui.ConvTop, r.h.m.ui.ConvHold, r.h.m.ui.Entry, focus, top, hold, entry)
			}
		}
		if !thought(t, r.h.m.State(), 2).Streaming {
			t.Fatal("the second phase is no longer live: the frames did not animate a card")
		}
	})
}

// TestThinkingReplayInOneBatch is criterion 1.7 (plan 22, design 7): a whole
// recording applied as one update, as a replay or a resume would, then frames
// at +0, +100 and +300 ms: no card, no entrance and no exit, every phase its
// summary row at once, none marked as drawn live.
func TestThinkingReplayInOneBatch(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := thinkStart(t, "p12c-block-claude-auto-boundary-edit.jsonl", claude.ProviderID)
		r.h.draw(tea.WindowSizeMsg{Width: 200, Height: 60})
		var all []event.Event
		for i := range r.per {
			for _, ev := range r.per[i] {
				if st, ok := ev.Payload.(event.Started); ok {
					st.Cwd = r.h.m.State().Context.Project
					ev.Payload = st
				}
				all = append(all, ev)
			}
		}
		end := int(r.recs[len(r.recs)-1].TMs)
		*r.now = thinkAt(end)
		r.h.draw(eventsMsg{sess: r.h.m.sess, evs: all})
		// What only a card draws: its count lines and the owner's expressions.
		card := []string{"live estimate", "token usage pending", "Model working", "Crunching...", "Vibing...", "Connecting dots...", "Smoothing...",
			"Pondering...", "Untangling...", "Brewing ideas...", "Cooking...", "Piecing it together...", "Making magic..."}
		for _, d := range []int{0, 100, 300} {
			r.at(end + d)
			f := r.h.frame()
			for _, no := range card {
				if strings.Contains(f, no) {
					t.Errorf("+%d ms: %q in\n%s", d, no, f)
				}
			}
			for _, want := range []string{"Thought for 3.5s · 717 thinking tokens", "Thought for 14s · 2,852 thinking tokens", "Thought for 0.1s · 17 thinking tokens", "Thought for 2.2s · 354 thinking tokens"} {
				if !strings.Contains(f, want) {
					t.Errorf("+%d ms: no %q in\n%s", d, want, f)
				}
			}
		}
		// By the phase's ID, as markLive keys it (plan 22, correction C11): the
		// entry's Seq is another number, so a lookup by it could never fail.
		for i := 1; i <= 4; i++ {
			if e := thought(t, r.h.m.State(), i); r.h.m.ui.Live[uint64(e.Thought.ID)] {
				t.Errorf("phase %d marked as drawn live", i)
			}
		}
		if render.Animating(r.h.m.State(), r.h.m.ui) {
			t.Error("a replay in one batch animates")
		}
	})
}

// TestThinkingKeys is design 8: on a phase's summary row, → opens its
// details under it, ← folds them, Enter shows the phase whole.
func TestThinkingKeys(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := thinkStart(t, "p12c-block-claude-auto-boundary-edit.jsonl", claude.ProviderID)
		r.h.draw(tea.WindowSizeMsg{Width: 160, Height: 50})
		r.until(27000)
		r.at(27000)
		last := thought(t, r.h.m.State(), 4)
		want := 0
		for i, e := range r.h.m.State().Conversation {
			if e.Seq == last.Seq {
				want = i + 1
			}
		}
		for i := 0; i < 10 && r.h.m.ui.Entry != want; i++ {
			r.h.draw(key(tea.KeyUp, 0))
		}
		if r.h.m.ui.Entry != want {
			t.Fatalf("↑ never reached the last phase's row (entry %d): entry %d", want, r.h.m.ui.Entry)
		}
		k := render.ThoughtKey(last)
		rows := r.h.m.screen.Conv.Rows
		r.h.draw(key(tea.KeyRight, 0))
		// Its four details: its times, its exact count, its last estimate, no
		// thinking text (render's thoughtDetails), a row each at 160 columns.
		const details = 4
		if f := r.h.frame(); !r.h.m.ui.Unfolded[k] || !strings.Contains(f, "▾ ✓ Thought for 2.2s") || r.h.m.screen.Conv.Rows != rows+details {
			t.Fatalf("→: open %v, rows %d then %d, %d details\n%s", r.h.m.ui.Unfolded[k], rows, r.h.m.screen.Conv.Rows, details, f)
		}
		r.h.draw(key(tea.KeyLeft, 0))
		if f := r.h.frame(); r.h.m.ui.Unfolded[k] || !strings.Contains(f, "▸ ✓ Thought for 2.2s") || r.h.m.screen.Conv.Rows != rows || r.h.m.ui.Entry != want {
			t.Fatalf("←: open %v, rows %d, want %d, entry %d\n%s", r.h.m.ui.Unfolded[k], r.h.m.screen.Conv.Rows, rows, r.h.m.ui.Entry, f)
		}
		r.h.draw(key(tea.KeyEnter, 0))
		if f := r.h.frame(); r.h.m.ui.ViewEntry != want || !strings.Contains(f, "THINKING") || !strings.Contains(f, "354 thinking tokens · exact, for this phase") {
			t.Fatalf("Enter: view %d, want %d\n%s", r.h.m.ui.ViewEntry, want, f)
		}
		if r.h.m.ui.Unfolded[k] {
			t.Errorf("Enter opened the details: %v", r.h.m.ui.Unfolded)
		}
	})
}
