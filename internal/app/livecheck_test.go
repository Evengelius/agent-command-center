package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"testing/iotest"
	"testing/synctest"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/Evengelius/agent-command-center/internal/event"
	"github.com/Evengelius/agent-command-center/internal/provider"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// What L2 (live_test.go) checks, kept here without the live tag so that
// deterministic fixtures check it before the owner's quota does (the owner,
// approving the third L2 run, 2026-10-16).

// liveTurn caps every L2 turn (the scope the owner approved).
const liveTurn = 2 * time.Minute

// streamStats counts what the dashboard received during one turn: batches
// of events, the orchestrator's deltas, and when its text deltas came. The
// streamed text stays in memory, for the coherence check.
type streamStats struct {
	start                     time.Time
	batches, events, maxBatch int
	text, thinking            int
	textBytes                 int
	firstText, lastText       time.Duration
	streamed                  strings.Builder
	end                       *event.TurnEnd // the orchestrator turn's end, as the CLI reported it
}

func (st *streamStats) observe(msg tea.Msg) {
	em, ok := msg.(eventsMsg)
	if !ok {
		return
	}
	st.batches++
	st.events += len(em.evs)
	st.maxBatch = max(st.maxBatch, len(em.evs))
	for _, ev := range em.evs {
		if e, ok := ev.Payload.(event.TurnEnd); ok && ev.Agent == "" {
			st.end = &e
		}
		d, ok := ev.Payload.(event.Delta)
		if !ok || ev.Agent != "" {
			continue
		}
		switch d.Block {
		case "thinking":
			st.thinking++
		case "text":
			at := time.Since(st.start)
			if st.text == 0 {
				st.firstText = at
			}
			st.text++
			st.textBytes += len(d.Text)
			st.streamed.WriteString(d.Text)
			st.lastText = at
		}
	}
}

func (st *streamStats) String() string {
	return fmt.Sprintf("%d batches, %d events (largest %d), %d thinking deltas, %d text deltas (%d bytes) from %v to %v after Enter",
		st.batches, st.events, st.maxBatch, st.thinking, st.text, st.textBytes, st.firstText.Round(time.Millisecond), st.lastText.Round(time.Millisecond))
}

// observe, when set, sees every message until delivers.
var observe func(tea.Msg)

// until delivers messages as Bubble Tea would until ok holds; false when d
// passes first.
func (h *harness) until(ok func(s *state.State) bool, d time.Duration) bool {
	deadline := time.After(d)
	for !ok(h.m.State()) {
		select {
		case msg := <-h.msgs:
			if observe != nil {
				observe(msg)
			}
			h.deliver(msg)
		case <-deadline:
			return false
		}
	}
	return true
}

// quitWithin delivers messages until the program quits or d passes. Ctrl+Q
// closes the CLI's input and leaves once the CLI has ended (ui-design.md,
// Keys: "close the CLI, restore the terminal"); Claude Code took about 1.8 s
// to end in the third L2 run, longer than settle's quiet (2026-10-16).
func (h *harness) quitWithin(d time.Duration) bool {
	deadline := time.After(d)
	for !h.quit {
		select {
		case msg := <-h.msgs:
			h.deliver(msg)
		case <-deadline:
			return false
		}
	}
	return true
}

// resumeArgv is what Ctrl+R must spawn after first: the same arguments, with
// --resume id after the session's own flags and before the adapter's extra
// ones (claude.Provider.Args); nil when first does not end with extra.
func resumeArgv(first []string, id string, extra []string) []string {
	n := len(first) - len(extra)
	if n < 0 || !slices.Equal(first[n:], extra) {
		return nil
	}
	out := append(append([]string(nil), first[:n]...), "--resume", id)
	return append(out, extra...)
}

// say pastes text and presses Enter without settling: a streamed answer
// never leaves the 300 ms quiet that settle waits for, so the turn would run
// inside it, out of the checks' sight (the first L2 run, 2026-10-16).
func say(h *harness, text string) {
	h.send(tea.PasteMsg{Content: text})
	h.deliver(key(tea.KeyEnter, 0))
}

func pending(s *state.State) *state.Approval {
	for _, ap := range s.Approvals {
		if ap.Status == state.Pending {
			return ap
		}
	}
	return nil
}

func idle(s *state.State) bool {
	return s.Session.Phase != state.Ready || s.Turn.Phase == state.Idle
}

// lastAssistant is the text of the last orchestrator answer. A message that
// had a thinking block keeps its answer in Text (ui-design.md, Thinking).
func lastAssistant(s *state.State) string {
	for i := len(s.Conversation) - 1; i >= 0; i-- {
		if e := s.Conversation[i]; e.Role == "assistant" && e.Agent == "" && e.Text != "" {
			return e.Text
		}
	}
	return ""
}

// textStreaming reports an orchestrator answer from entry from on whose text
// is streaming: the answer has begun and has not ended.
func textStreaming(s *state.State, from int) bool {
	for _, e := range s.Conversation[min(from, len(s.Conversation)):] {
		if e.Role == "assistant" && e.Agent == "" && e.Streaming && e.Text != "" {
			return true
		}
	}
	return false
}

// oneEntryPerMessage fails when an answer from entry from has no message id
// or shares one with another entry: the deltas would show as fragments.
func oneEntryPerMessage(s *state.State, from int) error {
	seen := map[string]bool{}
	for _, e := range s.Conversation[min(from, len(s.Conversation)):] {
		if e.Role != "assistant" || e.Agent != "" {
			continue
		}
		if e.MessageID == "" || seen[e.MessageID] {
			return fmt.Errorf("assistant entry with message id %q (empty or repeated)", e.MessageID)
		}
		seen[e.MessageID] = true
	}
	return nil
}

// tokenMetrics are the token and context values a turn of either CLI gives
// (telemetry-sources.md); one at least must be present after a turn.
var tokenMetrics = []string{"tokens.turn", "tokens.last", "tokens.total", "tokens.session", "context.used"}

// activityShown fails unless the dashboard shows, after a completed turn,
// what steps 6 to 8 draw from its events (plan 4, Done): the owner's last
// turn as a stage of zone A, closed and completed, with a completed tool
// under it; every telemetry value real, from a named source, or N/A with its
// reason (never a silent zero); a token or context value present; B's badge
// Idle.
func activityShown(s *state.State, frame string) error {
	i := len(s.Stages) - 1
	for i >= 0 && s.Stages[i].ByCLI { // a turn the CLI opened, with no prompt
		i--
	}
	if i < 0 {
		return fmt.Errorf("no stage the owner opened in zone A")
	}
	st := s.Stages[i]
	if st.Open || st.Outcome != "completed" {
		return fmt.Errorf("stage %d open %v, outcome %q; want closed and completed", st.N, st.Open, st.Outcome)
	}
	if want := fmt.Sprintf("✓ TURN %d", st.N); !strings.Contains(frame, want) { // the stage's head row (plan 24, design D5)
		return fmt.Errorf("zone A lacks %q", want)
	}
	tool := false
	for _, tc := range s.Tools {
		tool = tool || tc.Stage == st.N && tc.Status == state.ToolCompleted
	}
	if !tool {
		return fmt.Errorf("no completed tool under stage %d", st.N)
	}
	tokens := false
	for _, k := range slices.Sorted(maps.Keys(s.Metrics)) {
		v := s.Metrics[k]
		switch {
		case v.Present && v.Source == "":
			return fmt.Errorf("metric %s has a value without a source", k)
		case !v.Present && v.Reason == "":
			return fmt.Errorf("metric %s is neither a value nor N/A with a reason", k)
		}
		tokens = tokens || v.Present && v.V > 0 && slices.Contains(tokenMetrics, k)
	}
	if !tokens {
		return fmt.Errorf("no token or context value present (%v)", tokenMetrics)
	}
	if !strings.Contains(frame, "Idle") {
		return fmt.Errorf("zone B's badge is not Idle")
	}
	return nil
}

// TestActivityShown checks activityShown on the recordings of both CLIs, and
// that it fails on what it guards against.
func TestActivityShown(t *testing.T) {
	for _, name := range []string{"claude-subagent-approval.jsonl", "codex-subagents.jsonl"} {
		t.Run(name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				h := newHarness(t, Options{
					Config:    Config{Mock: filepath.Join("..", "..", "testdata", "mock", name)},
					Providers: []provider.Provider{spawnTrap{t}},
					Cwd:       t.TempDir(),
				})
				h.deliver(tea.WindowSizeMsg{Width: 160, Height: 60})
				h.settle()
				s, f := h.m.State(), frameText(h)
				if err := activityShown(s, f); err != nil {
					t.Fatalf("%v\n%s", err, f)
				}
				for _, m := range []struct {
					name   string
					change func() func()
				}{
					{"stages open", func() func() {
						old := slices.Clone(s.Stages)
						for i := range s.Stages {
							s.Stages[i].Open = true
						}
						return func() { s.Stages = old }
					}},
					{"only CLI stages", func() func() {
						old := slices.Clone(s.Stages)
						for i := range s.Stages {
							s.Stages[i].ByCLI = true
						}
						return func() { s.Stages = old }
					}},
					{"no tool", func() func() {
						old := s.Tools
						s.Tools = map[string]*state.ToolCall{}
						return func() { s.Tools = old }
					}},
					{"silent N/A", func() func() {
						s.Metrics["x"] = state.Value{}
						return func() { delete(s.Metrics, "x") }
					}},
					{"value without source", func() func() {
						s.Metrics["x"] = state.Value{Present: true, V: 1}
						return func() { delete(s.Metrics, "x") }
					}},
					{"no tokens", func() func() {
						old := maps.Clone(s.Metrics)
						for _, k := range tokenMetrics {
							delete(s.Metrics, k)
						}
						return func() { s.Metrics = old }
					}},
				} {
					undo := m.change()
					if activityShown(s, f) == nil {
						t.Errorf("%s: activityShown passed", m.name)
					}
					undo()
				}
				if activityShown(s, strings.ReplaceAll(f, "Idle", "Busy")) == nil {
					t.Error("not Idle: activityShown passed")
				}
				if activityShown(s, strings.ReplaceAll(f, "TURN", "STEP")) == nil {
					t.Error("no stage drawn: activityShown passed")
				}
			})
		})
	}
}

// coherent fails unless the turn from entry from shows its streamed text as
// one answer: a single orchestrator entry with text, holding all of it.
func coherent(s *state.State, from int, streamed string) error {
	var texts []string
	for _, e := range s.Conversation[min(from, len(s.Conversation)):] {
		if e.Role == "assistant" && e.Agent == "" && e.Text != "" {
			texts = append(texts, e.Text)
		}
	}
	switch {
	case len(texts) != 1:
		return fmt.Errorf("%d answers with text, want 1 (fragments?)", len(texts))
	case streamed == "" || !strings.HasPrefix(texts[0], streamed):
		return fmt.Errorf("the answer (%d bytes) does not begin with the %d bytes streamed", len(texts[0]), len(streamed))
	}
	return nil
}

// streamCut is what interruptMidStream saw.
type streamCut struct {
	stats *streamStats
	frame string        // the frame when Ctrl+C was pressed
	took  time.Duration // from Ctrl+C to the turn's end
}

// interruptMidStream is L2's turn 2: it sends prompt, waits up to limit for
// the orchestrator's answer to stream, presses Ctrl+C at once, and checks
// the turn ends interrupted within InterruptTimeout, the session Ready, its
// text one answer.
func interruptMidStream(h *harness, prompt string, limit time.Duration) (streamCut, error) {
	from := len(h.m.State().Conversation)
	c := streamCut{stats: &streamStats{start: time.Now()}}
	observe = c.stats.observe
	defer func() { observe = nil }()
	say(h, prompt)
	if !h.until(func(s *state.State) bool { return idle(s) || textStreaming(s, from) }, limit) || idle(h.m.State()) {
		return c, fmt.Errorf("no streamed text before the turn ended (%q)", h.m.State().Turn.Outcome)
	}
	if p := pending(h.m.State()); p != nil {
		return c, fmt.Errorf("unexpected approval %s %s", p.Tool, p.Input)
	}
	c.frame = frameText(h)
	if !strings.Contains(c.frame, "▌") {
		return c, fmt.Errorf("text streaming but no streaming cursor in the frame")
	}
	at := time.Now()
	h.deliver(key('c', tea.ModCtrl))
	if !h.until(idle, InterruptTimeout) {
		return c, fmt.Errorf("not idle %v after Ctrl+C (%q)", InterruptTimeout, h.m.State().Turn.Phase)
	}
	c.took = time.Since(at)
	s := h.m.State()
	if s.Turn.Outcome != "interrupted" || s.Session.Phase != state.Ready {
		return c, fmt.Errorf("outcome %q, session %q; want interrupted, Ready", s.Turn.Outcome, s.Session.Phase)
	}
	// The dashboard's outcome comes from the CLI's result (events-and-state.md,
	// Turn); the result itself says Ctrl+C stopped the turn, not a failure.
	if e := c.stats.end; e == nil || e.Outcome != "interrupted" {
		return c, fmt.Errorf("the CLI ended the turn %s, not interrupted", describeEnd(e))
	}
	if err := oneEntryPerMessage(s, from); err != nil {
		return c, err
	}
	return c, coherent(s, from, c.stats.streamed.String())
}

// p2Turn plays the CLI's side of P2's interrupted turn (lines 61 to 78)
// through p, once the prompt is written: the thinking block and its
// assistant line, then a text delta every pace (P3 and P3b: about 20 ms)
// until the interrupt is written or the deltas have all gone; with burst, the
// whole answer is written at once. The answer ends as P2's does: the
// interrupt's response, the assistant line with the text streamed, the
// interrupted message, the message's end and its result. It returns the text
// it streamed.
func p2Turn(t *testing.T, w *wireClaude, p *wireProc, project string, pace time.Duration, deltas int, interruptible bool) string {
	recs := probe(t, "p2-claude-conversation.jsonl")
	out := func(n int) map[string]any {
		var m map[string]any
		if err := json.Unmarshal(recs[n-1].Msg, &m); err != nil {
			t.Fatal(err)
		}
		return m
	}
	line := func(m map[string]any) string { b, _ := json.Marshal(m); return string(b) }
	for len(w.written("user")) == 0 {
		time.Sleep(time.Millisecond)
	}
	init := out(61)
	init["cwd"] = project
	p.emit(line(init))
	for _, n := range []int{62, 63, 64, 65, 67, 68, 69, 70} { // 66 is the driver's interrupt
		time.Sleep(2 * time.Millisecond)
		p.emit(line(out(n)))
	}
	var streamed strings.Builder
	var burst []string
	for k := 1; k <= deltas; k++ {
		if ir := w.written("control_request/interrupt"); interruptible && len(ir) > 0 {
			var req struct {
				RequestID string `json:"request_id"`
			}
			_ = json.Unmarshal([]byte(ir[0]), &req)
			resp := out(72)
			resp["response"].(map[string]any)["request_id"] = req.RequestID
			a := out(73)
			a["message"].(map[string]any)["content"].([]any)[0].(map[string]any)["text"] = streamed.String()
			p.emit(line(resp), line(a), line(out(74)), line(out(75)), line(out(76)), line(out(77)), line(out(78)))
			return streamed.String()
		}
		d := out(71)
		text := fmt.Sprintf("%d\n", k)
		d["event"].(map[string]any)["delta"].(map[string]any)["text"] = text
		streamed.WriteString(text)
		if pace == 0 {
			burst = append(burst, line(d))
			continue
		}
		p.emit(line(d))
		time.Sleep(pace)
	}
	// Not interrupted: the message ends and the turn succeeds (P2, lines 56 to 59).
	a := out(73)
	a["message"].(map[string]any)["content"].([]any)[0].(map[string]any)["text"] = streamed.String()
	p.emit(append(burst, line(a), line(out(75)), line(out(76)), line(out(77)), line(out(59)))...)
	return streamed.String()
}

// TestInterruptMidStream checks L2's turn 2 on P2's interrupted turn, paced
// as Claude Code paces it, and that it fails where it should.
func TestInterruptMidStream(t *testing.T) {
	const prompt = "Write the numbers from one to three hundred in words, one per line, and nothing else."
	run := func(t *testing.T, pace time.Duration, deltas int, interruptible bool) (streamCut, error, string, string) {
		w := &wireClaude{}
		h := wireRig(t, w, Options{})
		if h.m.State().Session.Phase != state.Ready {
			t.Fatalf("session %q", h.m.State().Session.Phase)
		}
		var streamed string
		done := make(chan struct{})
		project := h.m.State().Context.Project
		go func() {
			defer close(done)
			streamed = p2Turn(t, w, w.last(), project, pace, deltas, interruptible)
		}()
		c, err := interruptMidStream(h, prompt, liveTurn)
		h.until(idle, liveTurn) // the CLI's side finishes either way
		<-done
		return c, err, streamed, h.m.State().Turn.Outcome
	}

	t.Run("paced", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			c, err, streamed, _ := run(t, 20*time.Millisecond, 300, true)
			if err != nil {
				t.Fatalf("%v; %s", err, c.stats)
			}
			if c.stats.text == 0 || c.stats.text >= 300 || c.stats.thinking != 1 {
				t.Errorf("%s; want one thinking delta and the text cut before its 300 deltas", c.stats)
			}
			if c.stats.streamed.String() != streamed {
				t.Errorf("the dashboard received %q, the CLI streamed %q", c.stats.streamed.String(), streamed)
			}
			if c.took > time.Second {
				t.Errorf("%v from Ctrl+C to the turn's end", c.took)
			}
			t.Logf("%s; idle %v after Ctrl+C", c.stats, c.took)
		})
	})

	// A turn whose whole answer comes at once has ended before Ctrl+C can
	// stop it: the check fails, whether or not the dashboard saw the text
	// stream on its way in.
	t.Run("burst", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			_, err, _, outcome := run(t, 0, 300, false)
			if err == nil || !strings.Contains(err.Error(), "no streamed text") && !strings.Contains(err.Error(), `outcome "completed"`) {
				t.Errorf("error %v, want no streamed text or the completed turn", err)
			}
			if outcome != "completed" {
				t.Errorf("outcome %q, want completed", outcome)
			}
		})
	})

	// A turn the interrupt does not stop ends completed, and the dashboard
	// says so (owner, 2026-10-16): the check fails.
	t.Run("not interrupted", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			_, err, _, outcome := run(t, 20*time.Millisecond, 30, false)
			if err == nil || !strings.Contains(err.Error(), `outcome "completed"`) {
				t.Errorf("error %v, want the completed turn", err)
			}
			if outcome != "completed" {
				t.Errorf("outcome %q after a turn Claude Code completed, want completed", outcome)
			}
		})
	})
}

// TestFragmentChecks checks that the fragment checks fail on fragments.
func TestFragmentChecks(t *testing.T) {
	answer := func(id, text string) state.Entry {
		return state.Entry{Role: "assistant", MessageID: id, Text: text}
	}
	user := state.Entry{Role: "user", Text: "go"}
	for _, tc := range []struct {
		name     string
		conv     []state.Entry
		one, coh bool // oneEntryPerMessage and coherent pass
	}{
		{"one answer", []state.Entry{user, answer("m1", "1\n2\n3\n")}, true, true},
		{"thinking entry of another message", []state.Entry{user, answer("m0", ""), answer("m1", "1\n2\n3\n")}, true, true},
		{"deltas without an id", []state.Entry{user, answer("", "1\n"), answer("", "2\n3\n")}, false, false},
		{"one id, two entries", []state.Entry{user, answer("m1", "1\n"), answer("m1", "2\n3\n")}, false, false},
		{"two messages", []state.Entry{user, answer("m1", "1\n"), answer("m2", "2\n3\n")}, true, false},
		{"answer lost its text", []state.Entry{user, answer("m1", "3\n")}, true, false},
		{"an earlier turn's answer", []state.Entry{answer("m0", "x"), user, answer("m1", "1\n2\n3\n")}, true, true},
	} {
		s := &state.State{Conversation: tc.conv}
		from := 0
		for i, e := range tc.conv {
			if e.Role == "user" {
				from = i
			}
		}
		if err := oneEntryPerMessage(s, from); (err == nil) != tc.one {
			t.Errorf("%s: oneEntryPerMessage %v, want pass %v", tc.name, err, tc.one)
		}
		if err := coherent(s, from, "1\n2\n3\n"); (err == nil) != tc.coh {
			t.Errorf("%s: coherent %v, want pass %v", tc.name, err, tc.coh)
		}
	}
}

func describeEnd(e *event.TurnEnd) string {
	if e == nil {
		return "without a result"
	}
	return fmt.Sprintf("%q (%s)", e.Outcome, e.Subtype)
}

// shapeReader passes its reads on, logging each read and each line's shape.
type shapeReader struct {
	r            io.Reader
	reads, lines io.Writer
	t0           time.Time
	buf          []byte
	ids          map[string]int // message ids, numbered as they appear
}

func (t *shapeReader) Read(b []byte) (int, error) {
	n, err := t.r.Read(b)
	if n > 0 {
		ms := time.Since(t.t0).Milliseconds()
		fmt.Fprintf(t.reads, "%d %d\n", ms, n)
		t.buf = append(t.buf, b[:n]...)
		for {
			i := bytes.IndexByte(t.buf, '\n')
			if i < 0 {
				break
			}
			t.shape(ms, t.buf[:i])
			t.buf = t.buf[i+1:]
		}
	}
	return n, err
}

// shape writes a line's time and size, its type and subtype, the stream
// event's type, block index and delta type, the size of the text it carries,
// and which message it belongs to (1 for the first id seen).
func (t *shapeReader) shape(ms int64, l []byte) {
	var m struct {
		Type, Subtype string
		APIMessageID  string `json:"api_message_id"`
		Event         struct {
			Type  string
			Index *int
			Delta struct{ Type, Text, Thinking string }
		}
		Message struct {
			ID      string
			Content []struct{ Type, Text, Thinking string }
		}
	}
	if json.Unmarshal(l, &m) != nil {
		fmt.Fprintf(t.lines, "%d\t%d\tnot json\t\t\t\t\t\t\n", ms, len(l))
		return
	}
	ev, index, size, id := m.Event.Type, "", len(m.Event.Delta.Text)+len(m.Event.Delta.Thinking), m.APIMessageID
	if m.Event.Index != nil {
		index = strconv.Itoa(*m.Event.Index)
	}
	if m.Type == "assistant" {
		var types []string
		for _, c := range m.Message.Content {
			types = append(types, c.Type)
			size += len(c.Text) + len(c.Thinking)
		}
		ev, id = strings.Join(types, "+"), m.Message.ID
	}
	msg := ""
	if id != "" {
		if _, ok := t.ids[id]; !ok {
			t.ids[id] = len(t.ids) + 1
		}
		msg = strconv.Itoa(t.ids[id])
	}
	fmt.Fprintf(t.lines, "%d\t%d\t%s\t%s\t%s\t%s\t%s\t%d\t%s\n", ms, len(l), m.Type, m.Subtype, ev, index, m.Event.Delta.Type, size, msg)
}

// TestShapeReader checks the L2 recorder keeps the stream's shape and none
// of its content: on P2, read in small pieces, it passes every byte on and
// writes one row per stdout line, made of numbers and the protocol's own
// lower-case names only (no text, no id, no path).
func TestShapeReader(t *testing.T) {
	var in bytes.Buffer
	outs := 0
	for _, r := range probe(t, "p2-claude-conversation.jsonl") {
		if r.Dir == "out" {
			in.Write(r.Msg)
			in.WriteByte('\n')
			outs++
		}
	}
	want := in.String()
	var reads, lines strings.Builder
	sr := &shapeReader{r: iotest.HalfReader(&in), reads: &reads, lines: &lines, t0: time.Now(), ids: map[string]int{}}
	got, err := io.ReadAll(sr)
	if err != nil || string(got) != want {
		t.Fatalf("passed on %d of %d bytes (%v)", len(got), len(want), err)
	}
	rows := strings.Split(strings.TrimSuffix(lines.String(), "\n"), "\n")
	if len(rows) != outs {
		t.Errorf("%d rows, want %d", len(rows), outs)
	}
	for i, row := range rows {
		if strings.Count(row, "\t") != 8 || strings.TrimLeft(row, "0123456789abcdefghijklmnopqrstuvwxyz_+\t") != "" {
			t.Errorf("row %d %q: not 9 columns of numbers and names", i+1, row)
		}
	}
	// P2, line 54: the text delta "again", in the session's third message.
	if r := "\tstream_event\t\tcontent_block_delta\t0\ttext_delta\t5\t3"; !strings.Contains(lines.String(), r) {
		t.Errorf("no row ending %q:\n%s", r, lines.String())
	}
	total := 0
	for _, read := range strings.Split(strings.TrimSuffix(reads.String(), "\n"), "\n") {
		var ms, n int
		if _, err := fmt.Sscanf(read, "%d %d", &ms, &n); err != nil {
			t.Fatalf("read %q: %v", read, err)
		}
		total += n
	}
	if total != len(want) {
		t.Errorf("reads logged for %d bytes, want %d", total, len(want))
	}
}

// TestResumeAndQuitChecks checks the live test's last two steps on the fake
// CLI, as the third L2 run met them: the adapter's extra arguments, and a CLI
// that takes its time to end once its input closes.
func TestResumeAndQuitChecks(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		extra := []string{"--model", "haiku"}
		recs := probe(t, "p2-claude-conversation.jsonl")
		w := &wireClaude{onEOF: func() { time.Sleep(1800 * time.Millisecond) }}
		h := wireRigArgs(t, w, Options{}, extra)
		text, at := firstUser(t, recs)
		h.send(tea.PasteMsg{Content: text})
		h.send(key(tea.KeyEnter, 0))
		replay(t, h, w.last(), recs, at+1, 48) // the first turn, its approval allowed
		id := h.m.State().Session.ID
		if id == "" || !idle(h.m.State()) {
			t.Fatalf("session %q, turn %q after the first turn", id, h.m.State().Turn.Phase)
		}

		w.last().exit(1, "")
		h.settle()
		h.send(key('r', tea.ModCtrl))
		argv := w.spawns()
		if len(argv) != 2 || h.m.State().Session.Phase != state.Ready {
			t.Fatalf("%d spawns, session %q after Ctrl+R", len(argv), h.m.State().Session.Phase)
		}
		if want := resumeArgv(argv[0], id, extra); want == nil || !slices.Equal(argv[1], want) {
			t.Errorf("resume argv\n got %q\nwant %q", argv[1], want)
		}
		if old := append(append([]string(nil), argv[0]...), "--resume", id); slices.Equal(argv[1], old) {
			t.Errorf("--resume came after the extra arguments: %q", argv[1])
		}
		if resumeArgv([]string{"claude", "-p"}, id, extra) != nil {
			t.Error("resumeArgv accepted arguments without the extra ones")
		}

		h.send(key('q', tea.ModCtrl))
		if h.quit || h.m.State().Session.Phase != state.Closing {
			t.Fatalf("quit %v, session %q once settled; want Closing while the CLI ends", h.quit, h.m.State().Session.Phase)
		}
		at0 := time.Now()
		if !h.quitWithin(CloseTimeout) {
			t.Fatalf("no quit within %v; session %q", CloseTimeout, h.m.State().Session.Phase)
		}
		if took := time.Since(at0); h.m.State().Session.Phase != state.Closed || took > 2*time.Second {
			t.Errorf("session %q, quit %v after settling; want Closed once the CLI ended", h.m.State().Session.Phase, took)
		}
		if s := h.m.State(); !s.Session.HasExitCode || s.Session.ExitCode != 0 {
			t.Errorf("exit code %d (%v) in the state; want 0, as live_test.go checks", s.Session.ExitCode, s.Session.HasExitCode)
		}
		if code, _ := w.last().Wait(); code != 0 {
			t.Errorf("the CLI ended with %d, want 0: its input closed, no kill", code)
		}
	})
}

// insideCodex reports whether a file-change approval names only paths in the
// project. Codex's request carries no path: the card's summary is the paths
// of the fileChange item, or the request's reason when it gives one, which
// is then not trusted as inside.
func insideCodex(dir string, ap *state.Approval) bool {
	if ap.Tool != "fileChange" || ap.Summary == "" || ap.Summary == ap.Tool {
		return false
	}
	for _, p := range strings.Split(ap.Summary, ", ") {
		if !filepath.IsAbs(p) {
			p = filepath.Join(dir, p)
		}
		rel, err := filepath.Rel(strings.ToLower(filepath.Clean(dir)), strings.ToLower(filepath.Clean(p)))
		if err != nil || filepath.IsAbs(rel) || strings.HasPrefix(rel, "..") {
			return false
		}
	}
	return true
}

// TestInsideCodex: L4 accepts a file change only when every path its card
// names is in the project; a reason instead of paths is never trusted.
func TestInsideCodex(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "proj")
	for _, c := range []struct {
		tool, summary string
		want          bool
	}{
		{"fileChange", filepath.Join(dir, "allow.txt"), true},
		{"fileChange", "allow.txt", true},
		{"fileChange", filepath.Join(strings.ToUpper(dir), "a.txt") + ", " + filepath.Join(dir, "sub", "b.txt"), true},
		{"fileChange", filepath.Join(dir, "a.txt") + ", " + filepath.Join(dir, "..", "b.txt"), false},
		{"fileChange", filepath.Join(dir+"x", "a.txt"), false},
		{"fileChange", filepath.Join("..", "a.txt"), false},
		{"fileChange", "fileChange", false},
		{"fileChange", "", false},
		{"commandExecution", filepath.Join(dir, "a.txt"), false},
	} {
		if got := insideCodex(dir, &state.Approval{Tool: c.tool, Summary: c.summary}); got != c.want {
			t.Errorf("insideCodex(%s %q) = %v, want %v", c.tool, c.summary, got, c.want)
		}
	}
}
