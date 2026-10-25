package render

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"charm.land/lipgloss/v2"

	"github.com/Evengelius/agent-command-center/internal/state"
)

// Zone B's thinking (plan 22, brief amendment 18; plan.md, Plan 22, Design).
// Each phase of the orchestrator's thinking is a conversation entry of role
// "thinking" with its times and counts (state.Thought). While the phase runs,
// zone B draws its live card in the CLI's block, a box in the violet of
// thinking: the spinner, a playful expression and the elapsed time; the
// count; the bar. When it ends, a 300 ms exit, then its summary row among the
// tool rows, which opens on its details. While the turn works with no card,
// the activity line. Every animation is a function of the phase's times and
// the frame's (ui.Now): the frame never writes the chronology, and a phase
// that began and ended between two frames is its summary row at once
// (ui.Live, design 7).

// thinkStage is how a frame draws a thinking entry (plan 22, design 1).
type thinkStage uint8

const (
	thinkNone    thinkStage = iota // not a thinking entry
	thinkHidden                    // no row: a step that did not reason, or one no frame drew live past its end
	thinkLive                      // its live card
	thinkExit                      // its exit, its first 200 ms after the phase ended
	thinkSummary                   // its summary row
	thinkSettle                    // the exit's last 100 ms: the summary row, lit, laid out as one, so nothing moves when the exit ends
)

// The animations' times (plan 22, design 6).
const (
	spinStep     = 100 * time.Millisecond  // a spinner frame
	wordSlot     = 4 * time.Second         // an expression's turn
	wordDim      = 200 * time.Millisecond  // a new expression, dimmed
	enterStep    = 70 * time.Millisecond   // each of the entrance's four stages
	enterTime    = 4 * enterStep           // the entrance
	exitStep     = 100 * time.Millisecond  // each of the exit's three stages
	exitTime     = 3 * exitStep            // the exit
	growTime     = 300 * time.Millisecond  // the fill grows to a new value
	settleTime   = 700 * time.Millisecond  // its leading cell back to the fill's colour
	countLit     = 400 * time.Millisecond  // a new count, lit
	scaleLit     = time.Second             // a new scale's label, lit
	shimmerEvery = 2 * time.Second         // a shimmer crosses the fill
	shimmerTime  = 600 * time.Millisecond  // in this long
	sweepTime    = 1600 * time.Millisecond // mode C's segment crosses the track
	reconcileLit = 600 * time.Millisecond  // an exact count that came after its phase, lit
	frameAhead   = 50 * time.Millisecond   // the next frame while something animates
	// The activity line shows once its condition has held this long: longer than
	// every gap the recordings show between a response and the next card, or
	// before a turn's end (27 to 145 ms; plan 22, correction C6).
	activityDelay = 200 * time.Millisecond
)

// thinkSpin are the thinking spinner's six frames; stillSpin stands for it
// while a request waits for the owner (plan 22, designs 6 and 9).
var thinkSpin = [...]string{"◜", "◠", "◝", "◞", "◡", "◟"}

const stillSpin = "◌"

// expressions are the owner's ten, word for word (amendment 18, 4.2).
var expressions = [...]string{
	"Crunching...", "Vibing...", "Connecting dots...", "Smoothing...", "Pondering...",
	"Untangling...", "Brewing ideas...", "Cooking...", "Piecing it together...", "Making magic...",
}

// thoughtStage is how the frame draws thinking entry e (plan 22, designs 1,
// 4 and 7): its live card while the phase runs, a Codex step not yet
// confirmed included; its exit for 300 ms after it ended, only when a frame
// drew it live (ui.Live); then its summary row, for a phase of thinking;
// else nothing.
func thoughtStage(ui UI, e state.Entry) thinkStage {
	t := e.Thought
	switch {
	case t == nil:
		return thinkHidden
	case t.Live() && !t.Dropped:
		return thinkLive
	case ui.Live[uint64(t.ID)] && !t.End.IsZero() && !ui.Now.IsZero() && ui.Now.Sub(t.End) < exitTime && (ui.Now.Sub(t.End) < 2*exitStep || !thoughtShown(t)):
		return thinkExit // a step that did not reason closes over the whole 300 ms
	case ui.Live[uint64(t.ID)] && thoughtShown(t) && !ui.Now.IsZero() && ui.Now.Sub(t.End) < exitTime:
		return thinkSettle // the exit's last stage: the summary row, lit, in its place
	case thoughtShown(t):
		return thinkSummary
	}
	return thinkHidden
}

// thoughtShown says a phase has a summary row: it ended, was not dropped,
// and a Codex step was confirmed by a reasoning item (plan 22, design 4).
func thoughtShown(t *state.Thought) bool {
	return t != nil && !t.Live() && !t.Dropped && (!t.Inferred || t.Confirmed)
}

// ThoughtKey is a phase's key in UI.Unfolded, its details open (plan 22,
// design 8): "thought:" and its ID, unique across a resume, where an
// entry's Seq starts again.
func ThoughtKey(e state.Entry) string {
	if e.Thought == nil {
		return "thought:"
	}
	return "thought:" + strconv.Itoa(e.Thought.ID)
}

// cardSize is the live card's rows for zone B's conversation room (plan 22,
// design 2): seven, bordered, a blank row between its rows, from 20 rows;
// five, bordered, from 12; two without a border below (80x24).
func cardSize(room int) int {
	switch {
	case room >= 20:
		return 7
	case room >= 12:
		return 5
	}
	return 2
}

// spinGlyph is the spinner's frame d after it started, one each 100 ms.
func spinGlyph(d time.Duration) string {
	return thinkSpin[int(max(d, 0)/spinStep)%len(thinkSpin)]
}

// expression is the expression d after its card began, seeded by seed (plan
// 22, design 6): the seed's own, then every 4 s the one three on, never the
// same twice in a row; dim says a new one's first 200 ms.
func expression(seed uint64, d time.Duration) (string, bool) {
	d = max(d, 0)
	n := uint64(d / wordSlot)
	return expressions[(seed%10+3*n)%10], n > 0 && d%wordSlot < wordDim
}

// stopwatch is d as mm:ss, h:mm:ss past an hour (plan 22, design 2).
func stopwatch(d time.Duration) string {
	n := int(max(d, 0) / time.Second)
	if n >= 3600 {
		return fmt.Sprintf("%d:%02d:%02d", n/3600, n/60%60, n%60)
	}
	return fmt.Sprintf("%02d:%02d", n/60, n%60)
}

// thoughtFor is a phase's duration in words (plan 22, design 8): under 10 s
// to a tenth, rounded half up ("0.1s" for 56 ms, "3.5s" for 3,476 ms); under
// a minute to the second ("14s" for 13,902 ms); then "1m 05s".
func thoughtFor(d time.Duration) string {
	ms := max(d.Milliseconds(), 0)
	if t := (ms + 50) / 100; t < 100 {
		return fmt.Sprintf("%d.%ds", t/10, t%10)
	}
	n := (ms + 500) / 1000
	if n < 60 {
		return fmt.Sprintf("%ds", n)
	}
	return fmt.Sprintf("%dm %02ds", n/60, n%60)
}

// thoughtDuration is how long a phase took; false when its start or its end
// is not known.
func thoughtDuration(t *state.Thought) (time.Duration, bool) {
	if t.Start.IsZero() || t.End.IsZero() || t.End.Before(t.Start) {
		return 0, false
	}
	return t.End.Sub(t.Start), true
}

// thoughtSummary is a phase's summary (plan 22, design 8): its text, where
// its words end ("Thought for 2.8s"), and its exact count's bytes [n0, n1),
// equal when none. Claude Code's count is its thinking block's, or its
// message's over its phases; its estimate is marked as one; a Codex step's
// time is an upper bound, "up to", its count the step's reasoning tokens.
// Never a number the CLI did not report.
func thoughtSummary(t *state.Thought) (text string, lead, n0, n1 int) {
	d, timed := thoughtDuration(t)
	after := " after "
	if t.Inferred {
		after = " after up to " // a Codex step's time, its start inferred
	}
	var b strings.Builder
	switch t.Outcome {
	case "interrupted":
		b.WriteString("Thinking interrupted")
		if timed {
			b.WriteString(after + thoughtFor(d))
		}
	case "failed":
		b.WriteString("Thinking stopped")
		if timed {
			b.WriteString(after + thoughtFor(d))
		}
	default:
		b.WriteString("Thought")
		switch {
		case timed && t.Inferred:
			b.WriteString(" for up to " + thoughtFor(d))
		case timed:
			b.WriteString(" for " + thoughtFor(d))
		}
	}
	lead = b.Len()
	unit := " thinking tokens"
	if t.Inferred {
		unit = " reasoning tokens"
	}
	switch {
	case t.Outcome == "failed":
		b.WriteString(" · the turn failed")
	case t.HasTokens:
		b.WriteString(" · ")
		n0 = b.Len()
		b.WriteString(grouped(int(t.Tokens)))
		n1 = b.Len()
		b.WriteString(unit)
		if t.Shared > 1 {
			fmt.Fprintf(&b, " over %d phases", t.Shared)
		}
	case t.Estimate > 0 && !t.Inferred && t.Outcome == "":
		b.WriteString(" · ≈ " + grouped(int(t.Estimate)) + " thinking tokens (estimate)")
	}
	return b.String(), lead, n0, n1
}

// thoughtDetails are a phase's details (plan 22, design 8): when it ran,
// observed or inferred; each value, its scope and its source; what the CLI
// does not send. An end the CLI did not report as the phase's own (its
// message's, its turn's, a crash's) says so, and a count that can no longer
// come is not reported, not "not reported yet" (plan 22, the review): a phase
// whose turn or session ended with none is closed (correction C2). The 0 a
// message cut short reports is named for what it is, no exact count
// (correction C1).
//
// A Codex phase (plan 22, amendment 20) gives, in this order: its step's line;
// its reasoning item's own span, observed, when the item's start is known; the
// count; what Codex does not report live; and Codex's summary, its own text,
// each line of it a line of the entry, else that Codex sent none. The item's
// span and summary are the item's own; the phase's time stays the step's,
// inferred.
func thoughtDetails(t *state.Thought) []string {
	at := func(x time.Time) string {
		if x.IsZero() {
			return "--:--:--.---"
		}
		return x.Local().Format("15:04:05.000")
	}
	count := "exact count not reported yet"
	if t.Outcome != "" || t.EndInferred || t.Closed {
		count = "no exact count reported"
	}
	if t.Inferred {
		step := "the model step, its start not known,"
		if !t.Start.IsZero() {
			step = "the model step from " + at(t.Start) + " (inferred)"
		}
		if t.HasTokens {
			scope := "for this step"
			if t.Shared > 1 {
				scope = fmt.Sprintf("for the step's %d phases", t.Shared)
			}
			count = grouped(int(t.Tokens)) + " reasoning tokens · exact, " + scope + " · thread/tokenUsage/updated last.reasoningOutputTokens"
		}
		end := " to its reasoning's end at " + at(t.End) + " · reasoning item"
		switch {
		case t.EndNext:
			end = " to " + at(t.End) + ", when the next step began (inferred)"
		case t.EndInferred:
			end = " to " + at(t.End) + ", when its turn ended (inferred)"
		}
		out := []string{step + end}
		if !t.ItemStart.IsZero() {
			item := "the reasoning item from " + at(t.ItemStart)
			if t.EndInferred {
				item += " · observed, its completion not reported"
			} else {
				item += " to " + at(t.End) + " · observed"
			}
			out = append(out, item)
		}
		out = append(out, count, "no live estimate: Codex reports reasoning tokens when the step ends")
		if t.Summary == "" {
			return append(out, "no reasoning summary: Codex sent none")
		}
		// Codex's own lines: a summary of several parts has a blank line between
		// them, and a line of an entry holds no line break.
		lines := strings.Split(t.Summary, "\n")
		out = append(out, "Codex's summary, its own text: "+lines[0])
		return append(out, lines[1:]...)
	}
	switch {
	case t.HasTokens:
		scope := "for this phase"
		if t.Shared > 1 {
			scope = fmt.Sprintf("for the message's %d phases", t.Shared)
		}
		count = grouped(int(t.Tokens)) + " thinking tokens · exact, " + scope + " · message_delta usage.output_tokens_details.thinking_tokens"
	case t.Cut:
		count = "no exact count: the CLI reports " + grouped(int(t.CutTokens)) + " for an interrupted message"
	}
	span := "from " + at(t.Start) + " to " + at(t.End) + " · a thinking block, observed"
	if t.EndInferred {
		span = "from " + at(t.Start) + " (observed) to " + at(t.End) + " · a thinking block whose stop was not reported: its end its message's or its turn's"
	}
	out := []string{span, count}
	if t.Estimate > 0 {
		out = append(out, "live estimate "+grouped(int(t.Estimate))+" at its last update · system/thinking_tokens, estimated by the CLI")
	}
	return append(out, "no thinking text: the CLI sends none")
}

// thoughtMark is a phase's outcome: ✓ green; ✗ red when it was interrupted
// or the turn failed (plan 22, design 8).
func thoughtMark(t *state.Thought) span {
	if t.Outcome == "interrupted" || t.Outcome == "failed" {
		return deco("✗", stFailed)
	}
	return deco("✓", stDone)
}

// motionNow is the time a frame draws the turn's motion at (plan 22, design
// 9): the frame's; while a request waits for the owner, when it began to
// wait, so that what moves stands still.
func motionNow(s *state.State, ui UI) time.Time {
	at := ui.Now
	if s.Turn.Phase != state.WaitingApproval {
		return at
	}
	for _, id := range s.Queue {
		if ap := s.Approvals[id]; ap != nil && !ap.Status.Final() && !ap.RequestedAt.IsZero() && ap.RequestedAt.Before(at) {
			at = ap.RequestedAt
		}
	}
	return at
}

// thinkCard is what a frame draws a live or ending phase from (plan 22,
// design 2): the phase, the card's width and rows, and its clocks.
type thinkCard struct {
	e      state.Entry
	t      *state.Thought
	w      int           // the card's width: its block's content
	size   int           // its rows, steady: 7, 5 or 2
	start  time.Time     // the phase's start; the entry's time when not known
	now    time.Time     // the frame's time: the values and their transitions
	motion time.Time     // the time its motion is drawn at, frozen while a request waits
	since  time.Duration // from start to now: the timer, the entrance
	moved  time.Duration // from start to motion: the spinner, the expressions, the sweep
	still  bool          // a request waits for the owner: the spinner stands still
}

func newCard(s *state.State, ui UI, e state.Entry, w, size int) *thinkCard {
	c := &thinkCard{e: e, t: e.Thought, w: max(w, 1), size: size, start: e.Thought.Start, now: ui.Now,
		motion: motionNow(s, ui), still: s.Turn.Phase == state.WaitingApproval}
	if c.size == 0 {
		c.size = 7
	}
	if c.start.IsZero() {
		c.start = e.At // a step whose start is not known: from the card's own
	}
	c.since, c.moved = max(c.now.Sub(c.start), 0), max(c.motion.Sub(c.start), 0)
	return c
}

// inner is the width of the bordered card's rows.
func (c *thinkCard) inner() int { return max(c.w-4, 1) }

// barWidth is the bar's row's width: the card's, inside its border.
func (c *thinkCard) barWidth() int {
	if c.size == 2 {
		return c.w
	}
	return c.inner()
}

// rows are the card's rows at the frame (plan 22, designs 2 and 6): its
// entrance, steady, or its exit, whose last stage is the summary row of
// entry i drawn with its look k.
func (c *thinkCard) rows(s *state.State, ui UI, i int, k look) []line {
	switch {
	case k.think == thinkExit:
		return c.exit(s, ui, i, k)
	case c.since < enterTime:
		return c.entrance()
	}
	h := c.spinHead()
	return c.layout(h, c.countRow(h, false), c.barRow(false), stThinkBorder)
}

// headRow is the card's first row: its left part, and what is at its right.
type headRow struct{ left, right line }

// layout places the card's rows (plan 22, design 2): seven, the spinner's,
// the count's and the bar's in its border, a blank row between two; five
// without the blank rows; two without a border, the count after the
// expression. nil rows are blank.
func (c *thinkCard) layout(h headRow, count, bar line, border *lipgloss.Style) []line {
	if c.size == 2 {
		left := h.left
		if count != nil {
			left = append(append(append(line{}, h.left...), deco("  ", stNone)), count...)
		}
		return []line{spread(left, h.right, c.w), bar}
	}
	head := spread(h.left, h.right, c.inner())
	rows := []line{head, count, bar}
	if c.size == 7 {
		rows = []line{head, nil, count, nil, bar}
	}
	return c.box(rows, border)
}

// box borders rows in st, rounded, a cell of padding inside each edge.
func (c *thinkCard) box(rows []line, st *lipgloss.Style) []line {
	out := make([]line, 0, len(rows)+2)
	out = append(out, c.edge("╭", "╮", st))
	for _, r := range rows {
		out = append(out, append(append(line{deco("│ ", st)}, r.fit(c.inner())...), deco(" │", st)))
	}
	return append(out, c.edge("╰", "╯", st))
}

// edge is the border's top or bottom edge.
func (c *thinkCard) edge(a, b string, st *lipgloss.Style) line {
	return decoLine(a+strings.Repeat("─", max(c.w-2, 0))+b, st)
}

// spinHead is the spinner's row (plan 22, designs 2 and 6): the spinner,
// still while a request waits; the expression in lavender bold, a new one
// dim for its first 200 ms; the elapsed time at the right.
func (c *thinkCard) spinHead() headRow {
	glyph := stillSpin
	if !c.still {
		glyph = spinGlyph(c.moved)
	}
	word, dim := expression(c.e.Seq, c.moved)
	st := stThinkWord
	if dim {
		st = stThinkWordDim
	}
	return headRow{line{deco(glyph, stThinkMid), deco(" ", stNone), deco(word, st)}, c.timer(c.since)}
}

// exitHead is the spinner's row as the phase ends (plan 22, design 6): ✓,
// or ✗ when it was interrupted or failed; the summary's words, lit; the
// time it took.
func (c *thinkCard) exitHead() headRow {
	text, lead, _, _ := thoughtSummary(c.t)
	return headRow{line{thoughtMark(c.t), deco(" ", stNone), deco(text[:lead], stThinkLit)}, c.timer(max(c.t.End.Sub(c.t.Start), 0))}
}

// timer is the elapsed time d at the card's right, "--:--" when the phase's
// start is not known.
func (c *thinkCard) timer(d time.Duration) line {
	if c.t.Start.IsZero() {
		return decoLine("--:--", stGrey)
	}
	return decoLine(stopwatch(d), stGrey)
}

// countRow is the count's row (plan 22, designs 3 to 5), never a number the
// CLI did not report: mode A, a budget, the count and the budget; mode B the
// CLI's live estimate; mode C what is not reported yet. h is the head the row
// is drawn under, and exit says the card is in its exit. The 2-row card puts
// the count after the head: its short form when the long one would not fit the
// row (80x24; plan 22, the review), both measured against the head actually
// drawn, the exit's longer than the spinner's; and none when even the short
// form would not fit, so the head and the time stay whole and the count is left
// out of the row, never cut (correction C7).
func (c *thinkCard) countRow(h headRow, exit bool) line {
	l := c.countText(false, exit)
	if c.size != 2 {
		return l
	}
	fits := func(l line) bool { return h.left.width()+2+l.width()+1+h.right.width() <= c.w }
	if fits(l) {
		return l
	}
	if l = c.countText(true, exit); fits(l) {
		return l
	}
	return nil
}

// countText is the count's row, long or short (plan 22, designs 3 to 5): a
// Codex step once its reasoning item came is known to think, its count still
// to come. In its exit, Claude Code's row reads the exact count once it came
// (over its phases when it is a message's), else the estimate without "live",
// the phase being over; with neither, "no exact count" when none can come any
// more (a message cut short, a turn or session ended), else the usage still
// pending (correction C8).
func (c *thinkCard) countText(short, exit bool) line {
	t := c.t
	pick := func(long, s string) string {
		if short {
			return s
		}
		return long
	}
	switch {
	case t.Budget > 0:
		v, at := t.Estimate, t.EstimateAt
		if t.HasTokens {
			v, at = t.Tokens, t.TokensAt
		}
		return line{deco(pick("Thinking · ", ""), stBright), deco(grouped(int(v)), c.countStyle(at)), deco(" / "+grouped(int(t.Budget))+" tokens", stGrey)}
	case exit && !t.Inferred && t.HasTokens:
		over := ""
		if t.Shared > 1 {
			over = fmt.Sprintf(" over %d phases", t.Shared)
		}
		return line{deco(grouped(int(t.Tokens)), c.countStyle(t.TokensAt)), deco(pick(" thinking tokens", " tokens")+over, stGrey)}
	case exit && !t.Inferred && t.Estimate > 0:
		return line{deco("≈ "+grouped(int(t.Estimate)), c.countStyle(t.EstimateAt)), deco(pick(" thinking tokens · estimate", " tokens · estimate"), stGrey)}
	case exit && !t.Inferred && (t.Cut || t.Closed):
		return line{deco("Thinking", stBright), deco(pick(" · no exact count reported", " · no count"), stGrey)}
	case t.Estimate > 0:
		return line{deco("≈ "+grouped(int(t.Estimate)), c.countStyle(t.EstimateAt)), deco(pick(" thinking tokens · live estimate", " tokens · estimate"), stGrey)}
	case t.Inferred && t.Confirmed:
		return line{deco("Thinking", stBright), deco(pick(" · reasoning tokens reported when the step ends", " · tokens after the step"), stGrey)}
	case t.Inferred:
		return line{deco("Model working", stBright), deco(pick(" · reasoning tokens reported when the step ends", " · tokens after the step"), stGrey)}
	}
	return line{deco("Thinking", stBright), deco(pick(" · token usage pending", " · usage pending"), stGrey)}
}

// countStyle is a count's style: lit for 400 ms after it changed at at, then
// bright; the number itself never moves between two values (plan 22,
// criterion 2.6).
func (c *thinkCard) countStyle(at time.Time) *lipgloss.Style {
	if within(c.now, at, countLit) {
		return stThinkLit
	}
	return stBrightBold
}

// within says now is in [at, at+d).
func within(now, at time.Time, d time.Duration) bool {
	return !at.IsZero() && !now.Before(at) && now.Sub(at) < d
}

// barRow is the bar's row (plan 22, designs 3 to 6): modes A and B the fill
// on its track, mode B's scale at its right, lit for a second after it
// changed; mode C a segment sweeping the track, measuring nothing. frozen is
// the exit's: the value as it ended, nothing moving, never full because the
// phase ended.
func (c *thinkCard) barRow(frozen bool) line {
	t, w := c.t, c.barWidth()
	switch {
	case t.Budget > 0:
		if t.HasTokens {
			return c.fill(w, t.Tokens, t.Tokens, time.Time{}, t.Budget, frozen)
		}
		return c.fill(w, t.Estimate, t.EstimateFrom, t.EstimateAt, t.Budget, frozen)
	case t.Estimate > 0:
		sc := barScale(t.Estimate)
		label := scaleLabel(sc)
		st := stGrey
		if barScale(t.EstimateFrom) != sc && within(c.now, t.EstimateAt, scaleLit) {
			st = stThinkEdge
		}
		return append(c.fill(max(w-textWidth(label), 1), t.Estimate, t.EstimateFrom, t.EstimateAt, sc, frozen), deco(label, st))
	case frozen:
		return track(w)
	}
	return sweep(w, c.moved)
}

// fill is a bar w cells wide filled to value on scale (plan 22, design 6):
// after a new value at at, the fill grows from the last one over 300 ms,
// ease out, its leading cell lit, then in the mid shade until 700 ms; from
// then a shimmer crosses the fill every 2 s, the fill unchanged. frozen
// draws the value as it is, nothing moving.
func (c *thinkCard) fill(w int, value, from int64, at time.Time, scale int64, frozen bool) line {
	v, lead := float64(value), stThinkFill
	if !frozen && !at.IsZero() {
		switch d := c.now.Sub(at); {
		case d < growTime:
			v, lead = float64(from)+float64(value-from)*ease(float64(d)/float64(growTime)), stThinkEdge
		case d < settleTime:
			lead = stThinkMid
		}
	}
	halves := fillHalves(w, v, scale)
	shim := -1
	if !frozen {
		shim = c.shimmer((halves+1)/2, at)
	}
	return barCells(w, halves, lead, shim)
}

// shimmer is the filled cell the shimmer is on, -1 for none (plan 22,
// design 6): from 700 ms after the last value, or after the entrance without
// one, once every 2 s, across the cells filled in 600 ms; none while a
// request waits.
func (c *thinkCard) shimmer(cells int, at time.Time) int {
	base := c.start.Add(enterTime)
	if !at.IsZero() {
		base = at.Add(settleTime)
	}
	d := c.motion.Sub(base)
	if c.still || cells < 1 || d < 0 {
		return -1
	}
	if d %= shimmerEvery; d >= shimmerTime {
		return -1
	}
	return int(int64(cells) * int64(d) / int64(shimmerTime))
}

// entrance is the card's first 280 ms (plan 22, design 6), its rows the
// steady card's from the first frame: the border dim, a light sweeping its
// top edge, the rows blank; then the spinner's row dim; then it is normal
// and the count's row dim; then the count normal and the bar revealed from
// the left. Without a border, the same stages.
func (c *thinkCard) entrance() []line {
	h := c.spinHead()
	switch c.since / enterStep {
	case 0:
		rows := c.layout(headRow{}, nil, nil, stThinkDim)
		if c.size > 2 {
			rows[0] = c.sweptEdge()
		}
		return rows
	case 1:
		return c.layout(headRow{dimmed(h.left), dimmed(h.right)}, nil, nil, stThinkBorder)
	case 2:
		count := c.countRow(h, false)
		if count != nil {
			count = dimmed(count)
		}
		return c.layout(h, count, nil, stThinkBorder)
	}
	shown := int(int64(c.barWidth()) * int64(c.since-3*enterStep) / int64(enterStep))
	return c.layout(h, c.countRow(h, false), clip(c.barRow(false), shown), stThinkBorder)
}

// sweptEdge is the card's top edge in the dim shade, a light of four cells
// crossing it from left to right in the entrance's first 70 ms.
func (c *thinkCard) sweptEdge() line {
	rs := []rune("╭" + strings.Repeat("─", max(c.w-2, 0)) + "╮")
	seg := min(4, len(rs))
	at := min(int(int64(len(rs)-seg+1)*int64(c.since)/int64(enterStep)), len(rs)-seg)
	return line{deco(string(rs[:at]), stThinkDim), deco(string(rs[at:at+seg]), stThinkEdge), deco(string(rs[at+seg:]), stThinkDim)}
}

// exit is the card's 300 ms after its phase ended, for a phase a frame drew
// live (plan 22, design 6): ✓ for the spinner and the summary's words for the
// expression, the bar as it ended; then the ✓ row alone in the border; then
// the summary row, lit, settling to its colours. A step that did not reason
// dims, then closes away.
func (c *thinkCard) exit(s *state.State, ui UI, i int, k look) []line {
	d := max(c.now.Sub(c.t.End), 0)
	c.since = max(c.t.End.Sub(c.start), 0) // what moved stops as it was when the phase ended
	c.moved = min(c.moved, c.since)
	stage := d / exitStep
	if !thoughtShown(c.t) {
		switch stage {
		case 0:
			h := c.spinHead()
			rows := c.layout(h, c.countRow(h, true), c.barRow(true), stThinkDim)
			for j := range rows {
				rows[j] = dimmed(rows[j])
			}
			return rows
		case 1:
			if c.size > 2 {
				return []line{c.edge("╭", "╮", stThinkDim), nil}
			}
		}
		return []line{nil}
	}
	h := c.exitHead()
	if stage == 0 {
		return c.layout(h, c.countRow(h, true), c.barRow(true), stThinkBorder)
	}
	if c.size == 2 {
		return []line{spread(h.left, h.right, c.w)}
	}
	return c.box([]line{spread(h.left, h.right, c.inner())}, stThinkBorder)
}

// settleLit is how lit the summary row is in the exit's last 100 ms (plan
// 22, design 6): lit, then half way, settling to its colours.
func settleLit(ui UI, t *state.Thought) int {
	if ui.Now.Sub(t.End) >= exitTime-exitStep/2 {
		return 1
	}
	return 2
}

// barScale is mode B's scale (plan 22, design 3): 2,000 tokens, doubled
// until it holds v. The estimate is never lowered, so a phase's scale never
// goes down.
func barScale(v int64) int64 {
	sc := int64(2000)
	for sc < v {
		sc *= 2
	}
	return sc
}

// scaleLabel is the scale at the bar's right: " 2k", " 4k", " 8k"…
func scaleLabel(sc int64) string { return " " + strconv.FormatInt(sc/1000, 10) + "k" }

// ease is the growth's ease out, 1 − (1 − t)³, t clamped to [0, 1].
func ease(t float64) float64 {
	u := 1 - min(max(t, 0), 1)
	return 1 - u*u*u
}

// fillHalves is the half cells value fills of a bar w cells wide on scale:
// w × value / scale to half a cell, never past the bar.
func fillHalves(w int, value float64, scale int64) int {
	if w <= 0 || scale <= 0 || value <= 0 {
		return 0
	}
	return min(int(2*float64(w)*value/float64(scale)), 2*w)
}

// cellRuns draws w cells, each one's glyph and style from cell, a span for
// each run of equal cells.
func cellRuns(w int, cell func(int) (string, *lipgloss.Style)) line {
	var l line
	from, g, st := 0, "", (*lipgloss.Style)(nil)
	for i := 0; i <= w; i++ {
		var gi string
		var sti *lipgloss.Style
		if i < w {
			gi, sti = cell(i)
		}
		if i == w || gi != g || sti != st {
			if i > from {
				l = append(l, deco(strings.Repeat(g, i-from), st))
			}
			from, g, st = i, gi, sti
		}
	}
	return l
}

// barCells is a bar w cells wide, halves half cells of it filled (plan 22,
// design 6): the fill in its violet, a last half cell "╸", the rest of the
// track dim; the leading filled cell in lead, the shimmer's cell, shim, in
// the mid shade.
func barCells(w, halves int, lead *lipgloss.Style, shim int) line {
	full, n := halves/2, (halves+1)/2
	return cellRuns(w, func(i int) (string, *lipgloss.Style) {
		g := "━"
		if i == full {
			g = "╸"
		}
		switch {
		case i >= n:
			return "━", stThinkTrack
		case i == n-1 && lead != stThinkFill:
			return g, lead
		case i == shim:
			return g, stThinkMid
		}
		return g, stThinkFill
	})
}

// track is a bar's track alone, w cells.
func track(w int) line { return decoLine(strings.Repeat("━", max(w, 0)), stThinkTrack) }

// sweep is mode C's bar, w cells wide (plan 22, design 6): a segment of
// max(6, w/6) cells in the fill's violet, its two ends lit, crossing the
// track from left to right every 1.6 s, back in at the left as it leaves at
// the right; the rest of the track dim. It measures nothing.
func sweep(w int, moved time.Duration) line {
	if w <= 0 {
		return nil
	}
	seg := min(max(6, w/6), w)
	head := int(int64(w) * int64(max(moved, 0)%sweepTime) / int64(sweepTime))
	return cellRuns(w, func(i int) (string, *lipgloss.Style) {
		switch k := (i - head + w) % w; {
		case k >= seg:
			return "━", stThinkTrack
		case k == 0 || k == seg-1:
			return "━", stThinkEdge
		}
		return "━", stThinkFill
	})
}

// dimmed is l in the dim shade.
func dimmed(l line) line {
	out := make(line, len(l))
	for i, x := range l {
		x.st = stThinkDim
		out[i] = x
	}
	return out
}

// clip is l's first n cells, cut without a mark.
func clip(l line, n int) line {
	var out line
	for _, x := range l {
		if n <= 0 {
			break
		}
		x.text = cut(x.text, n)
		out = append(out, x)
		n -= textWidth(x.text)
	}
	return out
}

// thoughtRows adds thinking entry i's rows in its block fr (plan 22, design
// 1): its live card or its exit, a blank row of the block between it and the
// speaker's label above it, as the owner's drawing has; or its summary row,
// then its details when open.
func thoughtRows(s *state.State, ui UI, out []crow, i int, e state.Entry, w int, k look, fr *frame, focused bool) []crow {
	cw := cardContent(k.card, w)
	switch k.think {
	case thinkLive, thinkExit:
		if k.head {
			out = append(out, crow{l: line{}, fr: fr})
		}
		for _, l := range newCard(s, ui, e, cw, k.size).rows(s, ui, i, k) {
			out = append(out, crow{l: l, fr: fr})
		}
	case thinkSettle:
		out = append(out, crow{l: summaryRow(s, ui, i, e, cw, k, false, settleLit(ui, e.Thought)), fr: fr})
	case thinkSummary:
		// A click anywhere on the row opens or folds its details (plan 42).
		out = append(out, crow{fn: func() line { return summaryRow(s, ui, i, e, cw, k, focused, 0) }, fold: ThoughtKey(e), fr: fr, cursor: focused})
		if k.details {
			out = detailRows(out, ui.Wrap, i, e, cw, fr)
		}
	}
	return out
}

// detailRows adds a phase's details under its summary row (plan 22, design
// 8): lines 1… of its entry, wrapped four cells in, muted, on the tool rows'
// ground, each drawn only when shown.
func detailRows(out []crow, wc *WrapCache, i int, e state.Entry, w int, fr *frame) []crow {
	tw := max(w-4, 1)
	for j, src := range thoughtDetails(e.Thought) {
		at := ref{panel: 'B', item: i, line: j + 1}
		for _, r := range wc.ranges(src, tw, tw) {
			out = append(out, crow{fn: func() line {
				return grounded(append(line{deco("    ", stNone)}, rowLine(src, r, at, stMuted, nil)...), w)
			}, fr: fr})
		}
	}
	return out
}

// summaryRow is a phase's summary row, w cells wide, drawn as a tool row on
// its ground (plan 22, design 8): ▸ (▾ open) in the fold mark's blue, ✓ or
// ✗, the summary, the entry's first line, its words in the lavender of
// thinking and the rest muted; its end time and origin at the right, the
// entry cursor's hint in their place. lit is the exit's last stage: 2 lit,
// 1 half way, 0 settled; settled, an exact count come after the phase is lit
// for 600 ms.
func summaryRow(s *state.State, ui UI, i int, e state.Entry, w int, k look, focused bool, lit int) line {
	t := e.Thought
	text, lead, n0, n1 := thoughtSummary(t)
	words, rest := stThinkMid, stMuted
	switch lit {
	case 2:
		words, rest = stThinkLit, stBright
	case 1:
		words, rest = stThinkWord, stText
	}
	num := rest
	if lit == 0 && t.HasTokens && t.TokensAt.After(t.End) && within(ui.Now, t.TokensAt, reconcileLit) {
		num = stThinkLit
	}
	name := line{thoughtMark(t), deco(" ", stNone)}
	add := func(a, b int, st *lipgloss.Style) {
		if a < b {
			name = append(name, span{text: text[a:b], st: st, ref: ref{panel: 'B', kind: refText, item: i, off: a}})
		}
	}
	add(0, lead, words)
	if n1 > n0 {
		add(lead, n0, rest)
		add(n0, n1, num)
		add(n1, len(text), rest)
	} else {
		add(lead, len(text), rest)
	}
	mark := deco("▸", stFoldMark)
	if k.details {
		mark = deco("▾", stFoldMark)
	}
	outcome, origin := decoLine(clockSec(t.End), stGrey), originLabel(e.Origin)
	if focused {
		hint := strings.Replace(entryHint, "↑↓ entries", thoughtKeys(k.details, k.open)+"↑↓ entries", 1)
		outcome, origin = decoLine(hint, stAttention), ""
	}
	// A summary too long leaves toolLine no room for its end time and origin
	// (80 columns), and it cuts the origin ("obse…"): the origin is dropped
	// whole instead, the end time kept. The test is toolLine's own, by its
	// parts; the tool rows are as they were (plan 22, correction C9).
	if origin != "" && toolRight(outcome, origin).width() > toolRoom(toolLeft(mark, name), w) {
		origin = ""
	}
	return toolLine(mark, name, nil, k.nameW, nil, false, outcome, origin, w)
}

// thoughtView is a phase whole, full screen (plan 22, design 8): its head,
// its outcome's mark, its end time and its origin; its summary, then its
// details.
func thoughtView(e state.Entry, back span, w int) (line, []line) {
	ls := entryLines(nil, e)
	if len(ls) == 0 {
		why := "this model step did not reason: it has no summary"
		if e.Thought.Live() {
			why = "this phase still runs: its summary and its details come when it ends"
		}
		return line{back, deco(stillSpin+" THINKING", stBold)}, []line{plain(why, stGrey)}
	}
	t := e.Thought
	head := spread(line{back, thoughtMark(t), deco(" THINKING", stBold), deco("  "+clockSec(t.End), stGrey)}, decoLine(originLabel(e.Origin)+" ", stGrey), w)
	body := append(wrapped(ls[0], w-2, stText, ""), line{})
	for _, l := range ls[1:] {
		body = append(body, wrapped(l, w-2, stMuted, "  ")...)
	}
	return head, body
}

// ActivityOn says the activity line's condition holds at this frame (plan 22,
// design 9): the turn works (activityOn) and no thinking card is live or in
// its exit, so the line is never drawn under a card. The application notes in
// UI.ActivitySince when it began, and the drawing uses it too (lookAll), so
// the two never disagree (correction C6).
func ActivityOn(s *state.State, ui UI) bool {
	if !activityOn(s) {
		return false
	}
	// The thinking entries from the end, those only (plan 49, design D5): a
	// card live or in its exit is most often the last entry.
	at := thoughts(s, ui.Memo)
	for j := len(at) - 1; j >= 0; j-- {
		e := &s.Conversation[at[j]]
		ui.Memo.visited()
		if k := thoughtStage(ui, *e); k == thinkLive || k == thinkExit {
			return false
		}
	}
	return true
}

// activityDue says the activity line's condition has held for activityDelay at
// this frame: its drawing, and the frames its expressions ask for, wait for it
// (plan 22, correction C6). The application's ActivitySince is zero while the
// condition does not hold, which is never due.
func activityDue(ui UI) bool {
	return !ui.ActivitySince.IsZero() && ui.Now.Sub(ui.ActivitySince) >= activityDelay
}

// activityOn says the activity line follows the conversation (plan 22,
// design 9): the session ready and its turn running or submitted, the last
// entry not an answer's text streaming, no request waiting for the owner;
// ActivityOn adds that no thinking card is live.
func activityOn(s *state.State) bool {
	n := len(s.Conversation)
	if n == 0 || s.Session.Phase != state.Ready || s.Turn.Phase != state.Running && s.Turn.Phase != state.Submitted {
		return false
	}
	if e := &s.Conversation[n-1]; e.Role == "assistant" && e.Streaming && !thinkingMark(e) {
		return false
	}
	return !requestWaits(s)
}

// requestWaits says a request waits for the owner's answer.
func requestWaits(s *state.State) bool {
	for _, id := range s.Queue {
		if ap := s.Approvals[id]; ap != nil && !ap.Status.Final() {
			return true
		}
	}
	return false
}

// activityRows are the activity line's rows after the conversation (plan 22,
// design 9), decoration in the CLI's block: a blank row of the block, then
// the spinner, an expression and the turn's time, then the agents at work;
// after the rule and the speaker's label when the exchange has no entry of
// the CLI drawn yet, as a new message's; a blank row last, before the
// input's rule.
func activityRows(s *state.State, ui UI, w int, lk []look) []crow {
	fr := &frame{kind: cardAI, w: w}
	start, n := turnStart(s)
	last := len(lk) - 1 // the last entry drawn
	for last >= 0 && lk[last].hidden && lk[last].fold == 0 {
		last--
	}
	var out []crow
	if last >= 0 && lk[last].card == cardAI {
		out = append(out, crow{l: line{}, fr: fr})
	} else {
		out = append(out, crow{l: line{}}, crow{l: messageRule(w), sep: true}, crow{l: line{}}, crow{l: speakerLabel(s, state.Entry{At: start}), fr: fr})
	}
	var since time.Duration
	right := decoLine("--:--", stGrey)
	if !start.IsZero() {
		since = max(ui.Now.Sub(start), 0)
		right = decoLine(stopwatch(since), stGrey)
	}
	word, dim := expression(uint64(max(n, 0)), since)
	st := stThinkWord
	if dim {
		st = stThinkWordDim
	}
	left := line{deco(spinGlyph(since), stThinkMid), deco(" ", stNone), deco(word, st)}
	out = append(out, crow{l: spread(left, right, cardContent(cardAI, w)), fr: fr})
	if active := workingAgents(s); active > 0 {
		what := "subagents"
		if active == 1 {
			what = "subagent"
		}
		out = append(out, crow{l: line{deco(fmt.Sprintf("◇ %d %s working", active, what), stMuted)}, fr: fr})
	}
	return append(out, crow{l: line{}})
}

// turnStart is when the turn the activity line times began, and its number:
// the open stage's; before it opens, the last prompt's time.
func turnStart(s *state.State) (time.Time, int) {
	n := len(s.Stages)
	if n > 0 && s.Stages[n-1].Open {
		return s.Stages[n-1].StartedAt, s.Stages[n-1].N
	}
	for i := len(s.Conversation) - 1; i >= 0; i-- {
		if e := s.Conversation[i]; e.Role == "user" {
			return e.At, n + 1
		}
	}
	return time.Time{}, n + 1
}

// workingAgents counts the agents at work, zone C's active ones (running or
// waiting for the owner). One waiting for the owner makes the turn wait
// too, and the activity line is then off: no count of them (plan 22, the
// review).
func workingAgents(s *state.State) (active int) {
	for _, a := range s.Agents {
		if a.Status == state.AgentRunning || a.Status == state.AgentWaiting {
			active++
		}
	}
	return active
}

// thinkingLive says a phase of thinking the CLI reported runs in the last
// exchange: Claude Code's thinking block; a Codex step is not known to think
// until it ended (plan 22).
func thinkingLive(s *state.State) bool {
	for i := len(s.Conversation) - 1; i >= 0; i-- {
		e := s.Conversation[i]
		if e.Role == "user" {
			return false
		}
		if e.Role == "thinking" && e.Thought.Live() && !e.Thought.Inferred && !e.Thought.Dropped {
			return true
		}
	}
	return false
}

// Animating reports whether a frame 50 ms later would draw zone B's thinking
// otherwise (plan 22, design 6): a live card while no request waits; an
// entrance, an exit; a value growing or its edge settling, a count or a
// scale lit; an exact count come after its phase, lit; the activity line's
// next expression, dimmed, while the line is drawn (correction C6). The
// application then draws every 50 ms, else at its 100 ms spinner.
func Animating(s *state.State, ui UI) bool {
	still, now := s.Turn.Phase == state.WaitingApproval, ui.Now
	at := thoughts(s, ui.Memo) // the thinking entries only (plan 49, design D5)
	for j := len(at) - 1; j >= 0; j-- {
		e := &s.Conversation[at[j]]
		ui.Memo.visited()
		if e.Thought == nil {
			continue
		}
		t := e.Thought
		switch thoughtStage(ui, *e) {
		case thinkLive:
			start := t.Start
			if start.IsZero() {
				start = e.At
			}
			if !still || now.Sub(start) < enterTime || within(now, t.EstimateAt, settleTime) ||
				barScale(t.EstimateFrom) != barScale(t.Estimate) && within(now, t.EstimateAt, scaleLit) {
				return true
			}
		case thinkExit, thinkSettle:
			return true
		case thinkSummary:
			if t.HasTokens && t.TokensAt.After(t.End) && within(now, t.TokensAt, reconcileLit) {
				return true
			}
		}
	}
	// A card live (a request waiting) or in its exit is no activity line; the
	// line's frames wait for the line to be drawn (plan 22, correction C6).
	if !ActivityOn(s, ui) || !activityDue(ui) {
		return false
	}
	start, _ := turnStart(s)
	if start.IsZero() {
		return false
	}
	// A new expression's first 200 ms, or the next one due within 50 ms.
	d := max(now.Sub(start), 0)
	return d >= wordSlot && d%wordSlot < wordDim || (d+frameAhead)/wordSlot > d/wordSlot
}
