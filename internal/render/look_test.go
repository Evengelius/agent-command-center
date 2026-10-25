package render

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"charm.land/lipgloss/v2"

	"github.com/Evengelius/agent-command-center/internal/event"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// Plans 19 and 20: zone B's visual language (TestConversationLook) and
// its runs of tool calls (TestToolFold, with the keys in internal/app).
// Conversations are built from events through the reducer.

var lookAt = time.Date(2026, 10, 20, 11, 30, 0, 0, time.Local)

// convBench builds a session's conversation event by event.
type convBench struct {
	s   *state.State
	seq uint64
}

func newConvBench(provider, model string) *convBench {
	s := state.New()
	s.Screen = state.ScreenDashboard
	s.Context.Provider = provider
	s.Session.Phase = state.Ready
	s.Session.Model = model
	return &convBench{s: s}
}

func (b *convBench) ev(k event.Kind, p any) {
	b.seq++
	b.s.Apply(event.Event{Seq: b.seq, At: lookAt.Add(time.Duration(b.seq) * time.Second), Kind: k, Origin: event.Observed, Payload: p})
}

// prompt is the owner's message and the turn it opens.
func (b *convBench) prompt(text string) {
	b.ev(event.UserMessage, event.User{Text: text})
	b.ev(event.TurnStarted, nil)
}

func (b *convBench) msg(id, text string) {
	b.ev(event.MessageCompleted, event.Completed{MessageID: id, Text: text})
}

// think is a message with nothing but a thinking block.
func (b *convBench) think(id string) {
	b.ev(event.MessageDelta, event.Delta{MessageID: id, Block: "thinking"})
	b.ev(event.MessageCompleted, event.Completed{MessageID: id, Thinking: true})
}

// call is a tool call started; done, it ends, failed with fail.
func (b *convBench) call(id, name string, in map[string]string, done, fail bool) {
	raw, _ := json.Marshal(in)
	b.ev(event.ToolStarted, event.Tool{ToolID: id, Name: name, Input: raw})
	if done {
		b.ev(event.ToolCompleted, event.Tool{ToolID: id, Name: name, IsError: fail})
	}
}

func (b *convBench) calls(from, n int) {
	for i := from; i < from+n; i++ {
		b.call(fmt.Sprintf("c%d", i), "Read", map[string]string{"file_path": fmt.Sprintf(`C:\work\proj\f%d.go`, i)}, true, false)
	}
}

func (b *convBench) end(outcome string) {
	b.ev(event.TurnCompleted, event.TurnEnd{Outcome: outcome})
}

// bRows are every row of zone B's conversation, w cells wide.
func bRows(s *state.State, ui UI, w int) []line {
	view, _, _, _ := conversation(s, ui, w, 1<<20)
	out := make([]line, len(view))
	for i, c := range view {
		out[i] = c.draw()
	}
	return out
}

// rowText is a row's text, its trailing spaces dropped.
func rowText(l line) string { return strings.TrimRight(texts([]line{l}), " \n") }

func rowTexts(ls []line) []string {
	out := make([]string, len(ls))
	for i, l := range ls {
		out[i] = rowText(l)
	}
	return out
}

// spanOf is the span of l holding s.
func spanOf(l line, s string) (span, bool) {
	for _, x := range l {
		if strings.Contains(x.text, s) {
			return x, true
		}
	}
	return span{}, false
}

// agent starts a subagent in the turn running; done ends it, failed with
// fail.
func (b *convBench) agent(key string, done, fail bool) {
	b.ev(event.AgentStarted, event.Agent{Key: key, Type: "subagent", Label: "general-purpose"})
	if done {
		b.agentEnd(key, fail)
	}
}

func (b *convBench) agentEnd(key string, fail bool) {
	k := event.AgentCompleted
	if fail {
		k = event.AgentFailed
	}
	b.ev(k, event.Agent{Key: key})
}

// groundOf is a span's background: "card", "tool" or "" for none.
func groundOf(x span) string {
	switch x.st.GetBackground() {
	case lipgloss.Color("#13162A"):
		return "card"
	case lipgloss.Color("#1E2133"):
		return "tool"
	}
	return ""
}

// groundMap is the backgrounds of a row's cells, left to right, one letter
// per cell: c card, t tool, - none.
func groundMap(l line) string {
	var b strings.Builder
	for _, x := range l {
		c := map[string]string{"card": "c", "tool": "t", "": "-"}[groundOf(x)]
		b.WriteString(strings.Repeat(c, textWidth(x.text)))
	}
	return b.String()
}

// TestConversationLook is plan 20's item 3 (with plan 19's): zone B's
// cards, labels, tool rows, approvals, RESULT and thinking marks, from
// conversations built through the reducer.
func TestConversationLook(t *testing.T) {
	defer UsePalette(pickTrue)
	const w = 100 // zone B's inner width: labels beside their text from 90

	t.Run("3.1-3.4 cards, labels, gaps and tool rows", func(t *testing.T) {
		b := newConvBench("claude-code", "claude-haiku-5-5")
		b.prompt("Read the two files")
		b.msg("m1", "I'll read them.")
		b.call("t1", "Read", map[string]string{"file_path": `C:\work\proj\a.txt`}, true, false)
		b.think("m2") // between two calls: not drawn
		b.call("t2", "Agent", map[string]string{"subagent_type": "general-purpose", "description": "List directory and write found.txt", "prompt": "List the files."}, true, false)
		b.think("m3") // ended: not drawn either (plan 20)
		b.msg("m4", "I read both.")
		b.msg("m5", "Both files read.")
		b.end("completed")
		b.prompt("Say ok")
		b.msg("m6", "ok")
		b.end("completed")
		ls := bRows(b.s, UI{}, w)
		got := rowTexts(ls)
		at := clock(b.s.Conversation[0].At)
		hr := strings.Repeat("─", w)
		// Amendment 17: each message a block, its header on its own row and
		// its body under it from the same column; a rule between two
		// messages, a blank row on each side; a blank row first and last.
		want := []string{
			"",
			"▎ YOU  " + at, "▎ Read the two files",
			"", hr, "",
			"▎ CLAUDE  Haiku 5.5  " + at, "▎ I'll read them.", "▎", // the block's row between narration and calls
			"▎  ✓ Read   C:\\work\\proj\\a.txt", "▎  ✓ Agent  general-purpose · List directory and write found.txt", "▎",
			"▎ I read both.", "",
			"╭" + strings.Repeat("─", w-2) + "╮", "│ RESULT  " + at, "│ Both files read.", "╰" + strings.Repeat("─", w-2) + "╯",
			"", hr, "",
			"▎ YOU  " + at, "▎ Say ok",
			"", hr, "",
			"▎ CLAUDE  Haiku 5.5  " + at, "▎ ok",
			"",
		}
		if len(got) != len(want) {
			t.Fatalf("%d rows, want %d:\n%s", len(got), len(want), strings.Join(got, "\n"))
		}
		for i := range want {
			switch {
			case strings.HasPrefix(want[i], "▎  ✓"): // a call: its time and origin at the right
				if !strings.HasPrefix(got[i], want[i]+"  ") || !strings.HasSuffix(got[i], "  observed") {
					t.Errorf("row %d %q, want %q … observed", i, got[i], want[i])
				}
			case strings.HasPrefix(want[i], "│ "):
				if !strings.HasPrefix(got[i], want[i]) || !strings.HasSuffix(got[i], "│") {
					t.Errorf("row %d %q, want %q … │", i, got[i], want[i])
				}
			default:
				if got[i] != want[i] {
					t.Errorf("row %d %q, want %q", i, got[i], want[i])
				}
			}
		}
		// 3.1, 3.2 the bars', labels' and text's colours, on no ground: the
		// model and the time muted metadata.
		for _, c := range []struct {
			row  int
			text string
			st   *lipgloss.Style
			fg   string
		}{
			{1, "▎", stYou, "#A78BFA"}, {1, "YOU", stYou, "#A78BFA"}, {2, "▎", stYou, "#A78BFA"}, {2, "Read the two files", stBright, "#F9FAFB"},
			{6, "▎", stSpeaker, "#60A5FA"}, {6, "CLAUDE", stSpeaker, "#60A5FA"}, {6, "Haiku 5.5", stMuted, "#A1A9C0"},
			{7, "▎", stSpeaker, "#60A5FA"}, {7, "I'll read them.", stText, "#E5E7EB"},
			{15, "RESULT", stResult, "#4ADE80"}, {16, "Both files read.", stText, "#E5E7EB"},
			{4, "─", stFrame, "#3B3F51"}, // the rule between messages, the frames' colour
		} {
			x, ok := spanOf(ls[c.row], c.text)
			if !ok || x.st != c.st || x.st.GetForeground() != lipgloss.Color(c.fg) {
				t.Errorf("row %d %q: style %v, want %s", c.row, c.text, x.st, c.fg)
			}
		}
		for _, row := range []int{14, 17} { // the result box's edges, green
			if x, _ := spanOf(ls[row], "─"); x.st != stDone {
				t.Errorf("row %d: the edge's style %v, want green", row, x.st)
			}
		}
		// 3.1 no ground behind a message (amendment 17); 3.4 the tool rows'
		// from column 2 to one cell before the edge; 3.3 the gaps on none.
		none := strings.Repeat("-", w)
		for _, c := range []struct {
			row  int
			want string
		}{
			{0, ""}, {1, none}, {2, none}, {3, ""}, {4, none}, {6, none}, {7, none},
			{8, none}, {11, none}, // the block's rows between its parts
			{9, "--" + strings.Repeat("t", w-3) + "-"}, {10, "--" + strings.Repeat("t", w-3) + "-"},
			{15, none}, {16, none}, {28, ""},
		} {
			if g := groundMap(ls[c.row]); g != c.want {
				t.Errorf("row %d grounds %q, want %q", c.row, g, c.want)
			}
		}
		// 3.4 the tool name light blue and bold, the target readable.
		for _, c := range []struct{ text, fg string }{{"Read", "#93C5FD"}, {`C:\work\proj\a.txt`, "#A1A9C0"}} {
			if x, _ := spanOf(ls[9], c.text); x.st.GetForeground() != lipgloss.Color(c.fg) || groundOf(x) != "tool" {
				t.Errorf("%q: %v, want %s on the tool ground", c.text, x.st, c.fg)
			}
		}
		if x, _ := spanOf(ls[9], "Read"); !x.st.GetBold() {
			t.Error("the tool's name is not bold")
		}
		// Labels, rules and edges are decoration; the prompt and the answer
		// content.
		for _, row := range []int{1, 4, 6, 8, 14, 15, 17} {
			for _, x := range ls[row] {
				if x.ref.kind != refDeco {
					t.Errorf("row %d: %q is not decoration", row, x.text)
				}
			}
		}
		for _, c := range []struct {
			row  int
			text string
		}{{2, "Read the two"}, {16, "Both files"}, {7, "I'll read"}} {
			if x, _ := spanOf(ls[c.row], c.text); x.ref.kind != refText {
				t.Errorf("row %d: %q is not content", c.row, c.text)
			}
		}
	})

	t.Run("3.1 the same layout at every width", func(t *testing.T) {
		b := newConvBench("claude-code", "claude-haiku-5-5")
		b.prompt("hi there")
		b.msg("m1", "hello")
		b.end("completed")
		at := clock(b.s.Conversation[0].At)
		for _, width := range []int{60, 76, 100, 160} {
			got := rowTexts(bRows(b.s, UI{}, width))
			hr := strings.Repeat("─", width)
			want := []string{"", "▎ YOU  " + at, "▎ hi there", "", hr, "", "▎ CLAUDE  Haiku 5.5  " + at, "▎ hello", ""}
			if strings.Join(got, "\n") != strings.Join(want, "\n") {
				t.Errorf("%d: rows\n%s\nwant\n%s", width, strings.Join(got, "\n"), strings.Join(want, "\n"))
			}
		}
		// A long prompt wraps at the block's content column, under the label.
		long := newConvBench("claude-code", "claude-haiku-5-5")
		long.prompt(strings.Repeat("word ", 40))
		for _, r := range rowTexts(bRows(long.s, UI{}, 60)) {
			if strings.Contains(r, "word") && !strings.HasPrefix(r, "▎ word") {
				t.Errorf("a wrapped row %q not at the content column", r)
			}
		}
	})

	t.Run("3.2 a Codex session", func(t *testing.T) {
		b := newConvBench("codex", "gpt-5.5")
		b.prompt("hi")
		b.msg("m1", "hello")
		b.end("completed")
		if got := rowTexts(bRows(b.s, UI{}, w)); len(got) != 9 || got[6] != "▎ CODEX  gpt-5.5  "+clock(b.s.Conversation[1].At) || got[7] != "▎ hello" {
			t.Errorf("rows %q, want ▎ CODEX  gpt-5.5 at 6 and the answer under it", got)
		}
	})

	t.Run("3.3 the rules between messages", func(t *testing.T) {
		// Two prompts in a row, a CLI turn of its own, consecutive messages:
		// a rule only where a message starts, none inside one, none twice.
		b := newConvBench("claude-code", "claude-haiku-5-5")
		b.prompt("first")
		b.end("completed")
		b.prompt("second")
		b.msg("m1", "one")
		b.call("t1", "Bash", map[string]string{"command": "ls"}, true, false)
		b.msg("m2", "two")
		b.end("completed")
		b.ev(event.TurnStarted, nil) // the CLI's own turn: the same exchange
		b.msg("m3", "three")
		b.end("completed")
		got := rowTexts(bRows(b.s, UI{}, w))
		hr := strings.Repeat("─", w)
		if n := strings.Count(strings.Join(got, "\n"), hr); n != 2 {
			t.Errorf("%d rules, want 2 (before the second prompt, before CLAUDE):\n%s", n, strings.Join(got, "\n"))
		}
		for i := 1; i < len(got); i++ {
			if got[i] == hr && got[i-1] == hr {
				t.Errorf("two rules in a row at %d", i)
			}
			if got[i] == hr && (got[i-1] != "" || i+1 >= len(got) || got[i+1] != "") {
				t.Errorf("rule %d without a blank row on each side:\n%s", i, strings.Join(got, "\n"))
			}
		}
		if got[0] == hr || got[len(got)-1] == hr {
			t.Error("a rule at the conversation's start or end")
		}
	})

	t.Run("3.3 no rule at the window's edges", func(t *testing.T) {
		// Rows: blank, YOU, its body, blank, rule, blank, CLAUDE, ok, blank.
		// Scrolled so that the window ends after the rule, or starts with it,
		// the rule is a blank row: it would double B's border or the input's
		// rule (amendment 17, 2.5); with both messages in view it stays.
		b := newConvBench("claude-code", "claude-haiku-5-5")
		b.prompt("Reply with only the word ok.")
		b.msg("m1", "ok")
		b.end("completed")
		hr := strings.Repeat("─", w)
		for _, c := range []struct {
			top, room int
			rule      bool
		}{{0, 5, false}, {4, 5, false}, {3, 4, false}, {2, 6, true}, {0, 9, true}} {
			view, _, _, _ := conversation(b.s, UI{ConvHold: true, ConvTop: c.top}, w, c.room)
			var got []string
			for _, r := range view {
				got = append(got, rowText(r.draw()))
			}
			if has := strings.Contains(strings.Join(got, "\n"), hr); has != c.rule || len(got) != c.room {
				t.Errorf("top %d, room %d: rule %v in %q, want %v and %d rows", c.top, c.room, has, got, c.rule, c.room)
			}
		}
	})

	results := func(s *state.State) (out []string) {
		rows := rowTexts(bRows(s, UI{}, w))
		for i, r := range rows {
			if !strings.HasPrefix(r, "│ RESULT") {
				continue
			}
			var body []string // the box's rows under its label, to its bottom edge
			for _, b := range rows[i+1:] {
				if strings.HasPrefix(b, "╰") {
					break
				}
				body = append(body, strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(b, "│"), "│")))
			}
			out = append(out, strings.TrimSpace(strings.Join(body, " ")))
		}
		return out
	}
	t.Run("3.8 RESULT", func(t *testing.T) {
		one := func(outcome string, end bool) *convBench {
			b := newConvBench("claude-code", "claude-haiku-5-5")
			b.prompt("go")
			b.call("t1", "Bash", map[string]string{"command": "ls"}, true, false)
			b.msg("m1", "the answer")
			if end {
				b.end(outcome)
			}
			return b
		}
		for _, c := range []struct {
			name    string
			b       *convBench
			results string
		}{
			{"completed", one("completed", true), "[the answer]"},
			{"running", one("", false), "[]"},
			{"interrupted", one("interrupted", true), "[]"},
			{"failed", one("failed", true), "[]"},
		} {
			if got := fmt.Sprint(results(c.b.s)); got != c.results {
				t.Errorf("%s: RESULT over %s, want %s", c.name, got, c.results)
			}
		}
		// A lone answer keeps the speaker's label.
		b := newConvBench("claude-code", "claude-haiku-5-5")
		b.prompt("ok?")
		b.msg("m1", "ok")
		b.end("completed")
		if got := results(b.s); len(got) != 0 {
			t.Errorf("a lone answer: RESULT over %q", got)
		}
		// The owner's section 6.1: while an agent started in the exchange
		// runs, "I've launched the subagent…" stays narration; its report,
		// once the agent completed and the CLI's turn ended, is the result.
		b = newConvBench("claude-code", "claude-haiku-5-5")
		b.prompt("launch an agent")
		b.call("t1", "Agent", map[string]string{"subagent_type": "general-purpose", "description": "scan"}, true, false)
		b.agent("a1", false, false)
		b.msg("m1", "I've launched the subagent and will report its findings.")
		b.end("completed")
		if got := results(b.s); len(got) != 0 {
			t.Errorf("an agent running: RESULT over %q", got)
		}
		b.agentEnd("a1", false)
		b.ev(event.TurnStarted, nil)
		if got := results(b.s); len(got) != 0 {
			t.Errorf("the CLI's turn running: RESULT over %q", got)
		}
		b.msg("m2", "The agent found two files.")
		b.end("completed")
		if got := fmt.Sprint(results(b.s)); got != "[The agent found two files.]" {
			t.Errorf("after the agent and the CLI's turn: RESULT over %s", got)
		}
		b.prompt("next")
		if got := fmt.Sprint(results(b.s)); got != "[The agent found two files.]" {
			t.Errorf("after the next prompt: RESULT over %s", got)
		}
		// An agent failed: neutral, no RESULT.
		b = newConvBench("claude-code", "claude-haiku-5-5")
		b.prompt("launch an agent")
		b.call("t1", "Agent", map[string]string{"subagent_type": "general-purpose", "description": "scan"}, true, false)
		b.agent("a1", true, true)
		b.msg("m1", "The agent failed.")
		b.end("completed")
		if got := results(b.s); len(got) != 0 {
			t.Errorf("an agent failed: RESULT over %q", got)
		}
	})

	t.Run("3.4 symbols, outcomes and targets", func(t *testing.T) {
		b := newConvBench("claude-code", "claude-haiku-5-5")
		b.prompt("go")
		b.call("t1", "Read", map[string]string{"file_path": `C:\work\proj\a.txt`}, true, false)
		b.call("t2", "Bash", map[string]string{"command": "go vet ./..."}, true, true)
		b.call("t3", "Write", map[string]string{"file_path": `C:\work\proj\b.txt`, "content": "x"}, false, false)
		b.ev(event.ApprovalRequest, event.Approval{ID: "a3", Tool: "Write", ToolID: "t3"})
		b.call("t4", "Glob", map[string]string{"pattern": "**/*.go"}, false, false)
		ls := bRows(b.s, UI{}, w)
		for _, c := range []struct {
			prefix, outcome string
			st              *lipgloss.Style
		}{
			{"✓ Read   C:\\work\\proj\\a.txt", clockSec(b.s.Tools["t1"].EndedAt), stDone},
			{"✗ Bash", "failed " + clockSec(b.s.Tools["t2"].EndedAt), stFailed},
			{"! Write  C:\\work\\proj\\b.txt", "waiting for you", stAttention},
			{"◉ Glob  **/*.go", "running --", stProgress}, // waiting for an approval, the turn does not run
		} {
			found := false
			for _, l := range ls {
				r := rowText(l)
				if !strings.HasPrefix(r, "▎  "+c.prefix) {
					continue
				}
				found = true
				if !strings.HasSuffix(r, c.outcome+"  observed") {
					t.Errorf("%q: want the outcome %q", r, c.outcome)
				}
				if x, _ := spanOf(l, string([]rune(c.prefix)[0])); x.st != tint(c.st) {
					t.Errorf("%q: symbol style %v, want %v on the tool ground", r, x.st, c.st)
				}
			}
			if !found {
				t.Errorf("no row %q in:\n%s", c.prefix, strings.Join(rowTexts(ls), "\n"))
			}
		}
		// While the turn runs, a running call has the spinner and its time.
		r := newConvBench("claude-code", "claude-haiku-5-5")
		r.prompt("go")
		r.call("t1", "Bash", map[string]string{"command": "go test ./..."}, false, false)
		if got := strings.Join(rowTexts(bRows(r.s, UI{Spinner: 3}, w)), "\n"); !strings.Contains(got, "▎  "+spinner[3]+" Bash  go test ./...") {
			t.Errorf("no spinner on the running call:\n%s", got)
		}
		// An unfinished call, once the turn ends without it.
		b.end("interrupted")
		if got := strings.Join(rowTexts(bRows(b.s, UI{}, w)), "\n"); !strings.Contains(got, "▎  ○ Glob   **/*.go") && !strings.Contains(got, "▎  ○ Glob  **/*.go") {
			t.Errorf("no unfinished Glob in:\n%s", got)
		}
		// On 16 colours, no ground.
		UsePalette(pickANSI)
		for _, l := range bRows(b.s, UI{}, w) {
			for _, x := range l {
				if _, none := x.st.GetBackground().(lipgloss.NoColor); !none {
					t.Errorf("16 colours: %q on a ground", x.text)
				}
			}
		}
		UsePalette(pickTrue)
	})

	t.Run("3.6, 3.7 approvals", func(t *testing.T) {
		b := newConvBench("claude-code", "claude-haiku-5-5")
		b.prompt("go")
		for i, d := range []event.Decision{"", event.Allow, event.Deny} {
			id := fmt.Sprint(i)
			b.call("t"+id, "Bash", map[string]string{"command": "ls -" + id}, false, false)
			b.ev(event.ApprovalRequest, event.Approval{ID: "a" + id, Tool: "Bash", ToolID: "t" + id, Input: json.RawMessage(`{"command":"ls -` + id + `"}`)})
			if d != "" {
				b.ev(event.ApprovalResolved, event.Approval{ID: "a" + id, Decision: d})
			}
		}
		// A Write with its content, pending, for the preview.
		content := json.RawMessage(`{"file_path":"C:\\work\\proj\\found.txt","content":"alpha.txt\nbeta.md\n"}`)
		b.call("tw", "Write", nil, false, false)
		b.ev(event.ApprovalRequest, event.Approval{ID: "aw", Tool: "Write", ToolID: "tw", Input: content})
		ls := bRows(b.s, UI{}, w)
		got := rowTexts(ls)
		joined := strings.Join(got, "\n")
		// 3.6 the pending boxes: the pink edges, the heading, the preview
		// numbered on its inset ground, the pill; nothing green.
		at := clock(b.s.Approvals["aw"].RequestedAt)
		for _, want := range []string{
			"▎ ╭" + strings.Repeat("─", w-5) + "╮",
			"▎ │ ! APPROVAL · orchestrator · Bash",
			"▎ │   ls -0",
			"▎ │ ! APPROVAL · orchestrator · Write",
			"▎ │   C:\\work\\proj\\found.txt",
			"▎ │ ╭" + strings.Repeat("─", w-9) + "╮ │", // the preview in its own box, from 90 cells
			"▎ │ │ ▸ Preview of found.txt (2 lines)",
			"▎ │ │  1 │ alpha.txt",
			"▎ │ │  2 │ beta.md",
			"▎ │ ╰" + strings.Repeat("─", w-9) + "╯ │",
			"▎ │ This will create or overwrite the file at `C:\\work\\proj\\found.txt` with the above content.",
			"▎ │ ▐ F2 to answer ▌",
			"▎ ╰" + strings.Repeat("─", w-5) + "╯",
		} {
			if !strings.Contains(joined, want) {
				t.Errorf("no %q in:\n%s", want, joined)
			}
		}
		for i, r := range got {
			if strings.Contains(r, "! APPROVAL · orchestrator · Write") && !strings.HasSuffix(r, at+"  observed │") {
				t.Errorf("the Write's heading %q lacks its time and origin", r)
			}
			if !strings.Contains(r, "│") || strings.Contains(r, "allowed") {
				continue
			}
			for _, x := range ls[i] {
				if x.st.GetForeground() == lipgloss.Color("#4ADE80") {
					t.Errorf("green in a pending box: %q in %q", x.text, r)
				}
			}
			if strings.Contains(r, "alpha.txt") {
				if x, _ := spanOf(ls[i], "alpha.txt"); groundOf(x) != "tool" {
					t.Errorf("the preview's line %q not on the inset ground", r)
				}
			}
		}
		// 3.7 answered: one row each, the outcome in its colour.
		for _, c := range []struct {
			id, text string
			st       *lipgloss.Style
		}{{"a1", "allowed by you", stDone}, {"a2", "denied by you", stFailed}} {
			sym := map[string]string{"allowed by you": "✓", "denied by you": "✗"}[c.text]
			want := "▎  " + sym + " orchestrator · Bash  ls -" + c.id[1:]
			found := false
			for i, r := range got {
				if !strings.HasPrefix(r, want) {
					continue
				}
				found = true
				if !strings.HasSuffix(r, c.text+" "+clockSec(b.s.Approvals[c.id].AnsweredAt)+"  observed") {
					t.Errorf("%q: want %q and its time", r, c.text)
				}
				if x, _ := spanOf(ls[i], c.text); x.st != tint(c.st) {
					t.Errorf("%q: outcome style %v, want %v", r, x.st, c.st)
				}
			}
			if !found {
				t.Errorf("no row %q in:\n%s", want, joined)
			}
		}
		// The auto-review's verdict in its outcome's colour.
		b.call("t9", "Write", map[string]string{"file_path": `C:\work\proj\c.txt`, "content": "x"}, true, false)
		b.ev(event.ToolDecision, event.Verdict{ToolID: "t9", ReasonType: "classifier", Decision: event.Allow})
		for _, l := range bRows(b.s, UI{}, w) {
			if x, ok := spanOf(l, "approved by auto-review"); ok {
				if x.st != tint(stDone) {
					t.Errorf("the verdict's style %v, want green", x.st)
				}
				return
			}
		}
		t.Error("no auto-review verdict drawn")
	})

	t.Run("3.9 thinking", func(t *testing.T) {
		b := newConvBench("claude-code", "claude-haiku-5-5")
		b.prompt("go")
		b.ev(event.MessageDelta, event.Delta{MessageID: "m1", Block: "thinking"})
		// Plan 22: no mark; the thinking entry's card draws the phase.
		if got := strings.Join(rowTexts(bRows(b.s, UI{}, w)), "\n"); strings.Contains(got, "∴") {
			t.Errorf("a thinking block streaming: a mark in\n%s", got)
		}
		b.ev(event.MessageDelta, event.Delta{MessageID: "m1", Block: "text", Text: "Done"})
		if got := strings.Join(rowTexts(bRows(b.s, UI{}, w)), "\n"); strings.Contains(got, "∴ thinking") {
			t.Errorf("its text streaming: a thinking mark in\n%s", got)
		}
		b.ev(event.MessageCompleted, event.Completed{MessageID: "m1", Text: "Done", Thinking: true})
		b.think("m2")
		if got := strings.Join(rowTexts(bRows(b.s, UI{}, w)), "\n"); strings.Contains(got, "∴") {
			t.Errorf("thinking ended: a mark in\n%s", got)
		}
	})
}

// TestModelName is criterion 1.3 of plan 19.
func TestModelName(t *testing.T) {
	for id, want := range map[string]string{
		"claude-haiku-5-5": "Haiku 5.5", "claude-sonnet-4-5-20250929": "Sonnet 4.5", "claude-opus-5-5": "Opus 5.5",
		"claude-opus-5-5[1m]": "claude-opus-5-5[1m]", "gpt-5.5": "gpt-5.5", "claude-3-5-haiku-20241022": "claude-3-5-haiku-20241022",
		"claude-haiku-5-5-2026": "claude-haiku-5-5-2026", "": "",
	} {
		if got := modelName(id); got != want {
			t.Errorf("modelName(%q) = %q, want %q", id, got, want)
		}
	}
}

// TestToolFold is plan 19 criteria 2.1, 2.2, 2.4 and 2.5, and plan 20 3.5:
// the fold rows on the tool ground in the CLI card; 2.3, the keys and the
// click, is TestToolFoldKeys in internal/app.
func TestToolFold(t *testing.T) {
	const w = 100
	folds := func(s *state.State, ui UI) (rows []string, calls int) {
		for _, r := range rowTexts(bRows(s, ui, w)) {
			switch {
			case strings.HasPrefix(r, "▎  ▸ ") || strings.HasPrefix(r, "▎  ▾ "):
				rows = append(rows, strings.TrimSpace(strings.TrimPrefix(r, "▎")))
			case strings.HasPrefix(r, "▎  ✓ Read") || strings.HasPrefix(r, "▎  ✗ Read") || strings.HasPrefix(r, "▎  ◉ Read") || strings.HasPrefix(r, "▎  ⠋ Read"):
				calls++
			}
		}
		return rows, calls
	}
	run := func(n int) *convBench {
		b := newConvBench("claude-code", "claude-haiku-5-5")
		b.prompt("go")
		b.calls(0, n)
		b.msg("m1", "done")
		b.end("completed")
		return b
	}

	t.Run("2.1 the fold rows", func(t *testing.T) {
		for _, c := range []struct {
			n     int
			rows  string
			calls int
		}{
			{4, "[]", 4},
			{5, "[▸ 2 earlier tool events · Expand]", 3},
		} {
			rows, calls := folds(run(c.n).s, UI{})
			if fmt.Sprint(rows) != c.rows || calls != c.calls {
				t.Errorf("%d calls: fold rows %q and %d calls drawn, want %s and %d", c.n, rows, calls, c.rows, c.calls)
			}
		}
		// Twelve calls, one failed and one running among the first nine.
		b := newConvBench("claude-code", "claude-haiku-5-5")
		b.prompt("go")
		b.calls(0, 3)
		b.call("c3", "Read", map[string]string{"file_path": `C:\work\proj\f3.go`}, true, true)
		b.call("c4", "Read", map[string]string{"file_path": `C:\work\proj\f4.go`}, false, false)
		b.calls(5, 7)
		rows, calls := folds(b.s, UI{})
		if fmt.Sprint(rows) != "[▸ 9 earlier tool events · 1 failed · 1 running · Expand]" || calls != 3 {
			t.Errorf("12 calls: fold rows %q and %d calls drawn", rows, calls)
		}
		// The fold row's counts in their colours, on the ground.
		for _, l := range bRows(b.s, UI{}, w) {
			if x, ok := spanOf(l, "1 failed"); ok {
				y, _ := spanOf(l, "1 running")
				if x.st != tint(stFailed) || y.st != tint(stProgress) {
					t.Errorf("counts styled %p and %p", x.st, y.st)
				}
			}
		}
		// Expanded: its row reads Fold, every call drawn.
		ui := UI{Unfolded: map[string]bool{"c0": true}}
		if rows, calls := folds(b.s, ui); fmt.Sprint(rows) != "[▾ 9 earlier tool events · 1 failed · 1 running · Fold]" || calls != 12 {
			t.Errorf("expanded: fold rows %q and %d calls drawn", rows, calls)
		}
	})

	t.Run("2.2 what ends a run", func(t *testing.T) {
		b := newConvBench("claude-code", "claude-haiku-5-5")
		b.prompt("go")
		b.calls(0, 3)
		b.msg("m1", "halfway")
		b.calls(3, 3)
		b.call("w1", "Write", map[string]string{"file_path": `C:\work\proj\x.go`, "content": "x"}, false, false)
		b.ev(event.ApprovalRequest, event.Approval{ID: "ap", Tool: "Write", ToolID: "w1"})
		b.calls(6, 3)
		rows, calls := folds(b.s, UI{})
		if len(rows) != 0 || calls != 9 {
			t.Errorf("runs split by a message and an approval: fold rows %q, %d calls drawn", rows, calls)
		}
		// A pending card between two long runs stays whole.
		b = newConvBench("claude-code", "claude-haiku-5-5")
		b.prompt("go")
		b.calls(0, 6)
		b.call("w1", "Write", map[string]string{"file_path": `C:\work\proj\x.go`, "content": "x"}, false, false)
		b.ev(event.ApprovalRequest, event.Approval{ID: "ap", Tool: "Write", ToolID: "w1"})
		b.calls(6, 6)
		got := strings.Join(rowTexts(bRows(b.s, UI{}, w)), "\n")
		if strings.Count(got, "▸ ") != 2 || !strings.Contains(got, "! APPROVAL · orchestrator · Write") || !strings.Contains(got, "F2 to answer") {
			t.Errorf("a pending card between two runs:\n%s", got)
		}
	})

	t.Run("2.4 a selection copies every call", func(t *testing.T) {
		b := run(12)
		s := b.s
		var parts []string
		for _, e := range s.Conversation {
			parts = append(parts, strings.Join(entryLines(s, e), "\n"))
		}
		sc := Draw(s, UI{Width: 160, Height: 50})
		last := len(s.Conversation) - 1
		end := entryLines(s, s.Conversation[last])
		got := sc.Text(s, Selection{From: Pos{Panel: 'B'}, To: Pos{Panel: 'B', Item: last, Line: len(end) - 1, Off: len(end[len(end)-1])}})
		if got != strings.Join(parts, "\n\n") || strings.Count(got, "Read") != 12 {
			t.Errorf("copied %q", got)
		}
	})

	t.Run("2.5 the cache counts as without it", func(t *testing.T) {
		b := newConvBench("claude-code", "claude-haiku-5-5")
		cache := NewWrapCache()
		check := func(step string, ui UI) {
			t.Helper()
			for _, width := range []int{100, 61} {
				ui.Wrap = nil
				_, want, _, _ := conversation(b.s, ui, width, 30)
				ui.Wrap = cache
				_, got, _, _ := conversation(b.s, ui, width, 30)
				cache.sweep()
				if got != want {
					t.Errorf("%s, width %d: %d rows with the cache, %d without", step, width, got, want)
				}
			}
		}
		b.prompt("go")
		check("prompt", UI{})
		b.msg("m1", "first")
		check("message", UI{})
		for i := range 7 {
			b.calls(i, 1)
			check(fmt.Sprintf("call %d", i+1), UI{})
			if i == 2 {
				b.think("t" + fmt.Sprint(i))
				check("thinking after the third call", UI{})
			}
		}
		b.msg("m2", "the answer")
		check("the answer", UI{})
		b.end("completed")
		check("the turn's end", UI{})
		open := UI{Unfolded: map[string]bool{"c0": true}}
		check("expanded", open)
		check("folded again", UI{})
		b.ev(event.TurnStarted, nil)
		check("the CLI's turn", UI{})
		b.msg("m3", "the report")
		b.end("completed")
		check("its end", UI{})
		b.prompt("next")
		check("the next prompt", open)
	})
}
