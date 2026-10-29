//go:build timing

package app

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/Evengelius/agent-command-center/internal/event"
	"github.com/Evengelius/agent-command-center/internal/provider/claude"
)

// searchSession is the probe's session through the application (plan 66,
// What was found): TestLongSession's shape, turns of a prompt, 20 calls, a
// thinking phase and an answer of answer lines, then a 2,000-line paste
// whose "connection refused" is past its first lines; at 200x60, no frame
// drawn until the end.
func searchSession(t *testing.T, turns, answer int) (*harness, *scriptCLI) {
	t.Helper()
	h, cli := startScript(t, claude.ProviderID)
	h.send(tea.WindowSizeMsg{Width: 200, Height: 60})
	seq, at := uint64(1_000_000), time.Now().Add(-2*time.Hour)
	send := func(k event.Kind, p any) {
		seq++
		at = at.Add(100 * time.Millisecond)
		h.m.Update(eventsMsg{sess: h.m.sess, evs: []event.Event{{Seq: seq, At: at, Provider: cli.id, Kind: k, Origin: event.Observed, Payload: p}}})
	}
	for turn := range turns {
		id := fmt.Sprintf("m%d", turn)
		send(event.UserMessage, event.User{Text: fmt.Sprintf("Turn %d: look at the telemetry code and tell me what changed.", turn)})
		send(event.TurnStarted, nil)
		for k := range 20 {
			tid := fmt.Sprintf("o%d-%d", turn, k)
			in, _ := json.Marshal(map[string]string{"file_path": fmt.Sprintf(`C:\work\proj\internal\render\f%d.go`, k)})
			send(event.ToolStarted, event.Tool{ToolID: tid, Name: "Read", Input: in})
			send(event.ToolCompleted, event.Tool{ToolID: tid, Name: "Read", Output: "120 lines"})
		}
		send(event.MessageDelta, event.Delta{MessageID: id + "t", Block: "thinking"})
		send(event.MessageCompleted, event.Completed{MessageID: id + "t", Thinking: true})
		var ans strings.Builder
		for k := range answer {
			ans.WriteString(answerLine(turn*answer+k) + "\n")
		}
		send(event.MessageCompleted, event.Completed{MessageID: id, Text: ans.String(), Model: "claude-haiku-5-5"})
		send(event.TurnCompleted, event.TurnEnd{Outcome: "completed", DurationMS: 25_000})
	}
	paste := make([]string, 2000)
	for i := range paste {
		paste[i] = fmt.Sprintf("2026-10-29 12:%02d:%02d log line %d of the pasted log", i/60%60, i%60, i)
	}
	paste[1500] = "2026-10-29 12:25:00 dial tcp 10.0.0.5:5432: connection refused"
	send(event.UserMessage, event.User{Text: strings.Join(paste, "\n")})
	send(event.TurnStarted, nil)
	send(event.MessageCompleted, event.Completed{MessageID: "end", Text: "Seen.", Model: "claude-haiku-5-5"})
	send(event.TurnCompleted, event.TurnEnd{Outcome: "completed", DurationMS: 1_000})
	h.m.View()
	return h, cli
}

// TestSearchToFrame is criterion 66.9: the conversation's search, each key
// timed whole from Update to its frame (the query read, the matches found
// and placed, the rows lit, the frame drawn), on the probe's two sessions
// (plan 66, What was found), a 2,000-line paste's hidden match among
// them; then, the bar open, an answer streamed into the
// two-hour one, each delta to its frame. The target stays 16 ms at the 95th
// percentile (evaluation.md, Keystroke timing).
func TestSearchToFrame(t *testing.T) {
	for _, c := range []struct {
		name   string
		answer int
	}{{"two hours", 30}, {"answers ten times longer", 300}} {
		h, cli := searchSession(t, 240, c.answer)
		frameTime(h, key('f', tea.ModCtrl))
		if !h.m.ui.Search.Open {
			t.Fatal("Ctrl+F opened no bar")
		}
		// The keys: the query typed then taken back, round after round; a
		// prefix as short as one letter has the most matches, up to the cap.
		var keys []tea.KeyPressMsg
		for _, q := range []string{"telemetry", "refused"} {
			keys = append(keys, typedText(q)...)
			for range q {
				keys = append(keys, key(tea.KeyBackspace, 0))
			}
		}
		for i := 0; i < warmUp; i++ {
			frameTime(h, keys[i%len(keys)])
		}
		var ds []time.Duration
		for i := 0; i < 400; i++ {
			ds = append(ds, frameTime(h, keys[i%len(keys)]))
		}
		// The hidden match: "refused" is in the paste's folded lines alone,
		// typed into the field emptied first (the rounds can end mid-query).
		for range len([]rune(h.m.ui.Search.Query)) {
			frameTime(h, key(tea.KeyBackspace, 0))
		}
		for _, k := range typedText("refused") {
			ds = append(ds, frameTime(h, k))
		}
		if v := h.m.ui.Search; len(v.Matches) != 1 || v.Matches[0].Line != 1500 {
			t.Fatalf("the paste's hidden match: %+v", v.Matches)
		}
		p50, p95, worst := percentile(ds, 0.50), percentile(ds, 0.95), percentile(ds, 1)
		t.Logf("search, %s (%d entries), a key in the bar to its frame at 200x60, %d keys after %d warm-up: p50 %v, p95 %v, max %v (target p95 ≤ %v)",
			c.name, len(h.m.State().Conversation), len(ds), warmUp, p50, p95, worst, keystrokeTarget)
		if p95 > keystrokeTarget {
			overTarget(t, "%s: p95 %v over the %v target", c.name, p95, keystrokeTarget)
		}
		if c.answer != 30 {
			continue
		}
		// An answer streamed with the bar open: each delta finds the matches
		// again (design D6), "telemetry" in every tenth line.
		for range 7 {
			frameTime(h, key(tea.KeyBackspace, 0))
		}
		for _, k := range typedText("telemetry") {
			frameTime(h, k)
		}
		seq, at := uint64(9_000_000), time.Now()
		frameTime(h, eventsMsg{sess: h.m.sess, evs: []event.Event{{Seq: seq, At: at, Provider: cli.id, Kind: event.TurnStarted, Origin: event.Observed}}})
		var dd []time.Duration
		for k := range 400 {
			seq++
			text := answerLine(k)
			if k%10 == 0 {
				text = "the telemetry rows, line " + fmt.Sprint(k)
			}
			dd = append(dd, frameTime(h, eventsMsg{sess: h.m.sess, evs: []event.Event{{Seq: seq, At: at, Provider: cli.id, Kind: event.MessageDelta, Origin: event.AI, Payload: event.Delta{MessageID: "live", Block: "text", Text: text + "\n"}}}}))
		}
		p50, p95, worst = percentile(dd, 0.50), percentile(dd, 0.95), percentile(dd, 1)
		t.Logf("search, %s, the bar open, a delta to its frame at 200x60, %d deltas: p50 %v, p95 %v, max %v (target p95 ≤ %v)", c.name, len(dd), p50, p95, worst, keystrokeTarget)
		if p95 > keystrokeTarget {
			overTarget(t, "%s, streaming: p95 %v over the %v target", c.name, p95, keystrokeTarget)
		}
	}
}
