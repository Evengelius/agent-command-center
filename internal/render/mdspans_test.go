package render

import (
	"slices"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"

	"github.com/Evengelius/agent-command-center/internal/event"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// Plan 71, after the owner's acceptance run in Windows Terminal: a span of
// strong, emphasis or strikethrough over a paragraph's lines was shown with
// its markers, in AGENTS.md's preview and in a Codex answer. A paragraph's
// lines now pair the runs each leaves open (spans).

// The AGENTS.md paragraphs the owner's run found, word for word.
var (
	agentsWhat = []string{
		`A personal, interactive terminal workspace, a "mission control", for`,
		"multi-agent coding with agent CLIs: **Claude Code first, then OpenAI Codex",
		"CLI**, the only two supported (amendment 5; the provider interface stays open",
		"to others later). In one persistent screen of three zones (layout in the",
	}
	agentsStatus = []string{
		"**Status: plan 71 active since 2026-10-30, consistent Markdown rendering",
		"across ACC (brief amendment 74).** Plans 1 to 70 are closed, one entry",
		"each in `plan.md`, Closed plans; the MVP was approved on",
		"2026-10-17. `plan.md` holds the active plan's",
	}
	// The Codex answer the owner's run found.
	codexSpans = []string{
		"**This bold span begins on one physical line",
		"and ends on the next physical line.**",
		"",
		"*This italic span also begins on one line",
		"and ends on another.*",
	}
)

// acrossCase is a paragraph's lines, each as Markdown shows it, and the
// styles of some of their bytes.
type acrossCase struct {
	name        string
	lines, want []string
	at          []acrossAt
}

// acrossAt is the style of text's first byte, at its first occurrence on
// line.
type acrossAt struct {
	line int
	text string
	st   *lipgloss.Style
}

// acrossCases are the paragraphs of TestMarkdownAcrossLines.
func acrossCases() []acrossCase {
	bold := mdStyle(stText, true, false, false)
	italic := mdStyle(stText, false, true, false)
	both := mdStyle(stText, true, true, false)
	strike := mdStyle(stText, false, false, true)
	boldStrike := mdStyle(stText, true, false, true)
	boldCode := mdStyle(stMdCode, true, false, false)
	emBold := mdStyle(stMdEm, true, false, false)
	type at = acrossAt
	rule := strings.Repeat("─", 40)
	return []acrossCase{
		{"AGENTS.md, the two CLIs", agentsWhat, []string{
			`A personal, interactive terminal workspace, a "mission control", for`,
			"multi-agent coding with agent CLIs: Claude Code first, then OpenAI Codex",
			"CLI, the only two supported (amendment 5; the provider interface stays open",
			"to others later). In one persistent screen of three zones (layout in the",
		}, []at{{1, "multi-agent", stText}, {1, "Claude Code first", bold}, {1, "Codex", bold}, {2, "CLI", bold}, {2, ", the only", stText}, {3, "to others", stText}}},
		{"AGENTS.md, its status", agentsStatus, []string{
			"Status: plan 71 active since 2026-10-30, consistent Markdown rendering",
			"across ACC (brief amendment 74). Plans 1 to 70 are closed, one entry",
			"each in plan.md, Closed plans; the MVP was approved on",
			"2026-10-17. plan.md holds the active plan's",
		}, []at{{0, "Status", bold}, {1, "across ACC", bold}, {1, " Plans 1", stText}, {2, "plan.md", stMdCode}, {2, ", Closed", stText}}},
		{"Codex's bold and italic", codexSpans, []string{
			"This bold span begins on one physical line", "and ends on the next physical line.", "",
			"This italic span also begins on one line", "and ends on another.",
		}, []at{{0, "This bold", bold}, {1, "and ends", bold}, {3, "This italic", italic}, {4, "and ends", italic}}},
		{"strikethrough", []string{"~~struck over", "two lines~~ then plain"}, []string{"struck over", "two lines then plain"},
			[]at{{0, "struck", strike}, {1, "two lines", strike}, {1, " then", stText}}},
		{"over three lines", []string{"a **bold", "middle", "end** done"}, []string{"a bold", "middle", "end done"},
			[]at{{0, "a ", stText}, {0, "bold", bold}, {1, "middle", bold}, {2, "end", bold}, {2, " done", stText}}},
		{"nested", []string{"**bold *both", "both* bold** after"}, []string{"bold both", "both bold after"},
			[]at{{0, "bold", bold}, {0, "both", both}, {1, "both", both}, {1, " bold", bold}, {1, " after", stText}}},
		{"strong emphasis", []string{"***both", "lines***"}, []string{"both", "lines"}, []at{{0, "both", both}, {1, "lines", both}}},
		{"underscores", []string{"_italic", "here_ and snake_case", "names_too"}, []string{"italic", "here and snake_case", "names_too"},
			[]at{{0, "italic", italic}, {1, "here", italic}, {1, " and", stText}}},
		{"a list item's lines", []string{"- **item that", "  goes on** here", "- next **one"}, []string{"• item that", "  goes on here", "• next **one"},
			[]at{{0, "item", bold}, {1, "goes on", bold}, {1, " here", stText}}},
		{"a quote's lines", []string{"> quoted **bold", "> still bold** done"}, []string{"│ quoted bold", "│ still bold done"},
			[]at{{0, "bold", emBold}, {1, "still bold", emBold}, {1, " done", stMdEm}}},
		{"a quote's item's lines", []string{"> - **a", "> b** c"}, []string{"│ • a", "│ b c"},
			[]at{{0, "a", emBold}, {1, "b", emBold}, {1, " c", stMdEm}}},
		{"beside a span on its line", []string{"a **b** c **d", "e** f"}, []string{"a b c d", "e f"},
			[]at{{0, "b", bold}, {0, " c ", stText}, {0, "d", bold}, {1, "e", bold}, {1, " f", stText}}},
		{"a code span's markers literal", []string{"`**a**` and **b", "c**"}, []string{"**a** and b", "c"},
			[]at{{0, "**a**", stMdCode}, {0, " and ", stText}, {0, "b", bold}, {1, "c", bold}}},
		{"a code span over lines, its text literal", []string{"`**code", "span**` after"}, []string{"**code", "span** after"},
			[]at{{0, "**code", stMdCode}, {1, "span**", stMdCode}, {1, " after", stText}}},
		{"fenced code literal", []string{"```", "**a", "b**", "```"}, []string{"", "**a", "b**", ""}, []at{{1, "**a", stMdCode}, {2, "b**", stMdCode}}},
		// A closer its line's own parse would open a span with, and a code
		// span's closer before its line's own code spans.
		{"a closer before punctuation, a span after it", []string{"**Claude Code first, then OpenAI Codex", "CLI**, the only **two** supported"},
			[]string{"Claude Code first, then OpenAI Codex", "CLI, the only two supported"},
			[]at{{0, "Claude", bold}, {1, "CLI", bold}, {1, ", the only ", stText}, {1, "two", bold}, {1, " supported", stText}}},
		{"a closer before a letter", []string{"**a", "b**c d**"}, []string{"a", "bc d**"}, []at{{1, "b", bold}, {1, "c d**", stText}}},
		{"a closer after another's", []string{"**a ~~b", "c~~**"}, []string{"a b", "c"},
			[]at{{0, "a ", bold}, {0, "b", boldStrike}, {1, "c", boldStrike}}},
		{"a code span's closer, then its line's own", []string{"`go test", "./...` and `go vet`"}, []string{"go test", "./... and go vet"},
			[]at{{0, "go test", stMdCode}, {1, "./...", stMdCode}, {1, " and ", stText}, {1, "go vet", stMdCode}}},
		{"a code span over three lines", []string{"`a", "b ** c", "d` e"}, []string{"a", "b ** c", "d e"},
			[]at{{0, "a", stMdCode}, {1, "b ** c", stMdCode}, {2, "d", stMdCode}, {2, " e", stText}}},
		{"a code span in a span", []string{"**a `b", "c` d**"}, []string{"a b", "c d"},
			[]at{{0, "a ", bold}, {0, "b", boldCode}, {1, "c", boldCode}, {1, " d", bold}}},
		{"a code span of two backticks", []string{"``a ` b", "c``"}, []string{"a ` b", "c"},
			[]at{{0, "a ` b", stMdCode}, {1, "c", stMdCode}}},
		{"a span from inside the span its closer's line opens", []string{"**a", "b**c ~~d** e", "f~~"}, []string{"a", "bc d** e", "f"},
			[]at{{1, "b", bold}, {1, "c ", stText}, {1, "d", strike}, {2, "f", strike}}},
		{"a span of the line's own, a closer on the next", []string{"a **b** c", "d** e"}, []string{"a b c", "d** e"},
			[]at{{0, "b", bold}, {0, " c", stText}, {1, "d", stText}}},
		{"an opener between dropped", []string{"**a ~~b", "c** d~~"}, []string{"a ~~b", "c d~~"},
			[]at{{0, "a", bold}, {1, "c", bold}, {1, " d~~", stText}}},
		// A paragraph's end: no span over it.
		{"across a blank line", []string{"**not", "", "across**"}, []string{"**not", "", "across**"}, nil},
		{"across a line of spaces", []string{"**not", "   ", "across**"}, []string{"**not", "   ", "across**"}, nil},
		{"across a heading", []string{"**not", "# heading**"}, []string{"**not", "heading**"}, nil},
		{"across a list item", []string{"**not", "- item**"}, []string{"**not", "• item**"}, nil},
		{"across a quote", []string{"**not", "> quoted**"}, []string{"**not", "│ quoted**"}, nil},
		{"across a fence", []string{"**not", "```", "code**", "```", "after**"}, []string{"**not", "", "code**", "", "after**"}, nil},
		{"across a rule", []string{"**not", "---", "after**"}, []string{"**not", rule, "after**"}, nil},
		{"across a table", []string{"**not", "| a | b |", "|---|---|", "| c** | d |"}, []string{"**not", "│ a   │ b │", "├─────┼───┤", "│ c** │ d │"}, nil},
		{"across two list items", []string{"- **one", "- two**"}, []string{"• **one", "• two**"}, nil},
		{"across quotes of two depths", []string{"> **one", "> > two**"}, []string{"│ **one", "│ │ two**"}, nil},
		{"across a quote's heading", []string{"> **one", "> # two**"}, []string{"│ **one", "│ two**"}, nil},
		{"across a quote's item", []string{"> **one", "> - two**"}, []string{"│ **one", "│ • two**"}, nil},
		// Escaped, unmatched and malformed: as written.
		{"escaped", []string{`\*\*escaped`, `not\*\*`}, []string{"**escaped", "not**"}, nil},
		{"unclosed", []string{"**unclosed", "never"}, []string{"**unclosed", "never"}, nil},
		{"lengths that differ", []string{"**a", "b*"}, []string{"**a", "b*"}, nil},
		{"a closer at a line's start", []string{"**a", "** b"}, []string{"**a", "** b"}, nil},
		{"an opener at a line's end", []string{"a **", "b**"}, []string{"a **", "b**"}, nil},
		{"spaces inside", []string{"2 * 3", "* 4"}, []string{"2 * 3", "• 4"}, nil},
		{"a closer at a quote's text's start", []string{"> **a", ">** b"}, []string{"│ **a", "│** b"}, nil},
		{"an unmatched backtick", []string{"`unmatched **bold", "text** c"}, []string{"`unmatched bold", "text c"},
			[]at{{0, "`unmatched", stText}, {0, "bold", bold}, {1, "text", bold}, {1, " c", stText}}},
		{"an underscore in a word opens nothing", []string{"snake_case and", "more_ here"}, []string{"snake_case and", "more_ here"}, nil},
		{"an underscore in a word closes nothing", []string{"_italic", "snake_case"}, []string{"_italic", "snake_case"}, nil},
	}
}

// TestMarkdownAcrossLines: spans over a paragraph's lines, each line as
// shown and styled; none over a paragraph's end; escaped, unmatched and
// coded markers as before.
func TestMarkdownAcrossLines(t *testing.T) {
	for _, c := range acrossCases() {
		t.Run(c.name, func(t *testing.T) {
			ks := mdLines(c.lines)
			for j, l := range c.lines {
				sh := mdShow(l, ks[j], 40, stText)
				if got := mdShownText(sh); got != c.want[j] {
					t.Errorf("line %d %q shown %q, want %q", j, l, got, c.want[j])
				}
			}
			for _, a := range c.at {
				l := c.lines[a.line]
				i := strings.Index(l, a.text)
				if i < 0 {
					t.Fatalf("no %q on line %d", a.text, a.line)
				}
				if got := mdByteStyle(mdShow(l, ks[a.line], 40, stText), i); got != a.st {
					t.Errorf("line %d %q styled %v, want %v", a.line, a.text, got, a.st)
				}
			}
		})
	}
}

// TestMarkdownAcrossLinesCache: a paragraph's kinds read through a frame's
// cache, in a frame and the next, are those read anew; its lines' runs are
// dropped two frames after their last use.
func TestMarkdownAcrossLinesCache(t *testing.T) {
	wc := NewWrapCache()
	for frame := range 2 {
		for i, c := range acrossCases() {
			d := wc.markdown(mdKey{panel: 'B', i: i}, strings.Join(c.lines, "\n"), splitLines)
			if got, want := d.md, mdLines(c.lines); !slices.Equal(got, want) {
				t.Errorf("frame %d, %s: %+v, want %+v", frame, c.name, got, want)
			}
		}
		wc.sweep()
	}
	wc.sweep()
	if len(wc.delims) != 0 || len(wc.delimsOld) != 0 || len(wc.texts) != 0 || len(wc.textsOld) != 0 {
		t.Errorf("%d and %d runs, %d and %d texts kept after two frames without them", len(wc.delims), len(wc.delimsOld), len(wc.texts), len(wc.textsOld))
	}
	// Streamed in zone B, 9 bytes a delta: every frame with the cache the
	// frame without it.
	b := newConvBench("codex", "a-model")
	b.prompt("Answer.")
	text := strings.Join(slices.Concat(agentsWhat, codexSpans, []string{"", "**a", "b**c ~~d** e", "f~~"}), "\n")
	for n, rest := 0, text; rest != ""; n++ {
		k := min(9, len(rest))
		b.ev(event.MessageDelta, event.Delta{MessageID: "m1", Block: "text", Text: rest[:k]})
		rest = rest[k:]
		ui := UI{Width: 120, Height: 40, Focus: "B"}
		cached := ui
		cached.Wrap = wc
		if got, want := Draw(b.s, cached).String(), Draw(b.s, ui).String(); got != want {
			t.Fatalf("delta %d: the cached frame differs:\n%s\nwant\n%s", n, got, want)
		}
	}
}

// TestMarkdownAcrossLinesRaw: what is raw stays as written over lines too:
// the owner's message, a Markdown file's source view, a diff's lines.
func TestMarkdownAcrossLinesRaw(t *testing.T) {
	b := newConvBench("claude-code", "a-model")
	b.prompt(strings.Join(agentsWhat, "\n"))
	frame := texts(Draw(b.s, UI{Width: 200, Height: 60, Focus: "B"}).lines)
	f := mdViewer("AGENTS.md", true)
	f.Lines = agentsWhat
	_, rows := blameFrame(f, 200, 60)
	for i, l := range agentsWhat[1:3] {
		if !strings.Contains(frame, l) {
			t.Errorf("the owner's message: %q not drawn as written:\n%s", l, frame)
		}
		if rowWith(rows, l) < 0 {
			t.Errorf("the source view: %q not drawn as written:\n%s", l, strings.Join(rows, "\n"))
		}
		if got := plainOf(diffText(DiffLine{'+', 0, i + 1, l}, 0, 200)); !strings.Contains(got, l) {
			t.Errorf("the diff line %q drawn %q", l, got)
		}
	}
}

// TestMarkdownAcrossLinesSurfaces: the same spans in every surface that
// renders Markdown, as one renderer draws them: zone B for each CLI, a
// Markdown file's preview, the inspection's Messages tab, Codex's summary
// in a phase's details, zone C's timeline.
func TestMarkdownAcrossLinesSurfaces(t *testing.T) {
	bold := mdStyle(stText, true, false, false)
	text := strings.Join(append(append([]string{}, agentsWhat...), append([]string{""}, agentsStatus...)...), "\n")
	for _, provider := range []string{"claude-code", "codex"} {
		b := newConvBench(provider, "a-model")
		b.prompt("Show the two paragraphs.")
		b.msg("m1", text)
		sc := Draw(b.s, UI{Width: 200, Height: 60, Focus: "B"})
		zone := strings.Join(mdZoneB(sc), "\n")
		if strings.Contains(zone, "**") || !strings.Contains(zone, "Claude Code first, then OpenAI Codex") || !strings.Contains(zone, "across ACC (brief amendment 74).") {
			t.Errorf("%s: zone B\n%s", provider, zone)
		}
		for _, span := range []string{"CLI", "Status: plan 71 active since 2026-10-30, consistent Markdown rendering", "across ACC (brief amendment 74)."} {
			if _, _, st, ok := mdSpanAt(sc, span); !ok || st != bold {
				t.Errorf("%s: %q drawn %v, %v", provider, span, ok, st)
			}
		}
	}
	// The preview of AGENTS.md's lines.
	f := mdViewer("AGENTS.md", false)
	f.Lines = strings.Split(text, "\n")
	sc, rows := blameFrame(f, 200, 60)
	if rowWith(rows, "**") >= 0 || rowWith(rows, "multi-agent coding with agent CLIs: Claude Code first, then OpenAI Codex") < 0 {
		t.Errorf("the preview:\n%s", strings.Join(rows, "\n"))
	}
	if y := rowWith(rows, "CLI, the only two supported"); y < 0 || blameStyle(sc, y, "CLI") != bold {
		t.Errorf("the preview's CLI not bold")
	}
	// The inspection's Messages tab.
	s := state.New()
	s.Screen = state.ScreenDashboard
	s.Session.Phase = state.Ready
	s.Agents["a1"] = &state.Agent{Key: "a1", Label: "worker", Status: state.AgentCompleted, Messages: []state.Entry{{Role: "assistant", Text: text}}}
	sc = Draw(s, UI{Width: 200, Height: 60, Inspect: "a1"})
	if all := texts(sc.lines); strings.Contains(all, "**") || panelStyle(sc, 'I', "CLI") != bold {
		t.Errorf("the inspection:\n%s", all)
	}
	// Codex's summary in a phase's details.
	th := state.Thought{Start: thinkAt(1000), ItemStart: thinkAt(2500), End: thinkAt(4000), Inferred: true, Confirmed: true,
		Summary: "**Planning the\nchange** now"}
	ts := thinkBench(th)
	open := rowTexts(bRows(ts, UI{Unfolded: map[string]bool{ThoughtKey(ts.Conversation[1]): true}}, 200))
	if all := strings.Join(open, "\n"); strings.Contains(all, "**") || !strings.Contains(all, codexSummaryLead+"Planning the") || !strings.Contains(all, "change now") {
		t.Errorf("Codex's summary:\n%s", all)
	}
	// Zone C's timeline.
	if line := texts(timelineEntry(state.Entry{Role: "assistant", Text: strings.Join(agentsStatus, "\n")}, 160)); strings.Contains(line, "**") || !strings.Contains(line, "Status: plan 71 active") {
		t.Errorf("the timeline: %q", line)
	}
}

// TestMarkdownAcrossLinesStreaming: Codex's answer streamed in chunks of 1,
// 3, 7 and 11 bytes on each CLI: at each step its rows are those of the
// same text completed; an opener is shown as written until its closer
// comes on a later line, then both hide.
func TestMarkdownAcrossLinesStreaming(t *testing.T) {
	text := strings.Join(codexSpans, "\n")
	for _, provider := range []string{"claude-code", "codex"} {
		for _, size := range []int{1, 3, 7, 11} {
			b := newConvBench(provider, "a-model")
			b.prompt("Answer.")
			for n, rest := 0, text; rest != ""; n++ {
				k := min(size, len(rest))
				b.ev(event.MessageDelta, event.Delta{MessageID: "m1", Block: "text", Text: rest[:k]})
				rest = rest[k:]
				ai := len(b.s.Conversation) - 1
				done := *b.s
				done.Conversation = slices.Clone(b.s.Conversation)
				done.Conversation[ai].Streaming = false
				rows := answerRows(b.s, ai, 76)
				if d := answerRows(&done, ai, 76); !slices.Equal(rows, d) {
					t.Fatalf("%s, %d bytes, step %d: rows\n%s\nwant\n%s", provider, size, n, strings.Join(rows, "\n"), strings.Join(d, "\n"))
				}
				sofar := b.s.Conversation[ai].Text
				closed := strings.Contains(sofar, "physical line.**")
				if shown := strings.Join(rows, "\n"); strings.HasPrefix(sofar, "**This") && !closed && !strings.Contains(shown, "**This") || closed && strings.Contains(shown, "**This") {
					t.Fatalf("%s, %d bytes, step %d, closer come %v:\n%s", provider, size, n, closed, shown)
				}
			}
			b.ev(event.MessageCompleted, event.Completed{MessageID: "m1", Text: text})
			ai := len(b.s.Conversation) - 1
			if rows := strings.Join(answerRows(b.s, ai, 76), "\n"); strings.ContainsAny(rows, "*") {
				t.Errorf("%s, %d bytes: the completed answer's rows\n%s", provider, size, rows)
			}
		}
	}
}

// TestMarkdownAcrossLinesSource: the source stays the content: every span
// drawn is its source's bytes at its offset, a click on a span's cell its
// byte, the copy the source, widened over a span's markers when it is taken
// in whole, a word of it as the word; the search finds the source and
// lights what is shown.
func TestMarkdownAcrossLinesSource(t *testing.T) {
	b := newConvBench("claude-code", "a-model")
	b.prompt("Show the paragraph.")
	text := strings.Join(agentsWhat, "\n")
	b.msg("m1", text)
	s := b.s
	ai := len(s.Conversation) - 1
	sc := Draw(s, UI{Width: 200, Height: 60, Focus: "B"})
	ls := entryLines(s, s.Conversation[ai])
	if strings.Join(ls, "\n") != text {
		t.Fatalf("the entry's lines are not its text: %q", ls)
	}
	// Each text span is its line's bytes at its offset.
	for y, l := range sc.lines {
		for _, x := range l {
			if r := x.ref; r.panel == 'B' && r.kind == refText && r.item == ai && x.text != "" {
				if r.line >= len(ls) || r.off+len(x.text) > len(ls[r.line]) || ls[r.line][r.off:r.off+len(x.text)] != x.text {
					t.Errorf("row %d: span %q at line %d, byte %d, is not its source", y, x.text, r.line, r.off)
				}
			}
		}
	}
	// A click on CLI's first cell: its byte, line 2, offset 0.
	if x, y, _, ok := mdSpanAt(sc, "CLI"); !ok {
		t.Error("CLI not drawn as a span")
	} else if p, _, hit := sc.HitIn('B', x, y, false); !hit || p.Item != ai || p.Line != 2 || p.Off != 0 {
		t.Errorf("CLI's first cell hits %+v", p)
	}
	at := strings.Index(agentsWhat[1], "Claude")
	sel := func(l0, o0, l1, o1 int) string {
		return sc.Text(s, Selection{From: Pos{Panel: 'B', Item: ai, Line: l0, Off: o0}, To: Pos{Panel: 'B', Item: ai, Line: l1, Off: o1}})
	}
	for _, c := range []struct {
		name           string
		l0, o0, l1, o1 int
		want           string
	}{
		{"the span whole", 1, at, 2, 3, "**Claude Code first, then OpenAI Codex\nCLI**"},
		{"a word of it", 1, strings.Index(agentsWhat[1], "Codex"), 1, len(agentsWhat[1]), "Codex"},
		{"short of its end", 1, at, 2, 2, "Claude Code first, then OpenAI Codex\nCL"},
		{"from its middle", 1, strings.Index(agentsWhat[1], "first"), 2, 3, "first, then OpenAI Codex\nCLI"},
		{"the lines whole", 0, 0, 3, len(agentsWhat[3]), text},
	} {
		if got := sel(c.l0, c.o0, c.l1, c.o1); got != c.want {
			t.Errorf("%s: copied %q, want %q", c.name, got, c.want)
		}
	}
	// The search finds the source and lights what is shown.
	ms, _ := ConvMatches(s, UI{}, "OpenAI Codex", nil)
	if len(ms) != 1 || ms[0].Line != 1 || ms[0].Off != strings.Index(agentsWhat[1], "OpenAI Codex") {
		t.Fatalf("matches %+v", ms)
	}
	lit := litSpans(Draw(s, UI{Width: 200, Height: 60, Focus: "B", Search: SearchView{Open: true, Query: "OpenAI Codex", Matches: ms, Cur: 0}}), stMatchCur)
	if strings.Join(lit, "") != "OpenAI Codex" {
		t.Errorf("lit %q", lit)
	}
	stars, _ := ConvMatches(s, UI{}, "**", nil)
	lit = litSpans(Draw(s, UI{Width: 200, Height: 60, Focus: "B", Search: SearchView{Open: true, Query: "**", Matches: stars, Cur: 0}}), stMatchCur)
	if len(stars) != 2 || strings.Contains(strings.Join(lit, ""), "*") {
		t.Errorf("the markers: %d matches, lit %q", len(stars), lit)
	}
	// The preview's copy.
	f := mdViewer("AGENTS.md", false)
	f.Lines = agentsWhat
	fs, _ := blameFrame(f, 200, 60)
	if got := fs.Text(nil, Selection{From: Pos{Panel: 'F', Item: 1, Off: at}, To: Pos{Panel: 'F', Item: 2, Off: 3}}); got != "**Claude Code first, then OpenAI Codex\nCLI**" {
		t.Errorf("the preview's copy %q", got)
	}
}

// TestMarkdownAcrossLinesRandom: on random paragraphs, every span over lines
// is within one paragraph, its markers hidden, and every line's own parse
// begins where its paragraph's text does.
func TestMarkdownAcrossLinesRandom(t *testing.T) {
	parts := []string{"a", "word", " ", "**", "*", "_", "__", "~~", "`", "``", "\\", "- ", "> ", "# ", "```", "|", "\n", "\n", "\n\n"}
	found := 0 // spans over lines, so that the test is not empty
	for n := range 3000 {
		var sb strings.Builder
		for k := range 40 {
			sb.WriteString(parts[(n*7+k*13+k*k*n)%len(parts)])
		}
		lines := strings.Split(sb.String(), "\n")
		ks := mdLines(lines)
		for _, sp := range crossSpans(ks) {
			found++
			if sp.l0 >= sp.l1 {
				t.Fatalf("%q: a span %+v not over lines", lines, sp)
			}
			for l := sp.l0; l <= sp.l1; l++ {
				if _, part, _ := mdSegment(lines[l], ks[l]); part == mdParaNone {
					t.Fatalf("%q: a span %+v over line %d, no paragraph's", lines, sp, l)
				}
			}
			for _, e := range [][3]int{{sp.l0, sp.o, sp.n}, {sp.l1, sp.c, sp.n}} {
				sh := mdShow(lines[e[0]], ks[e[0]], 40, stText)
				for i := e[1]; i < e[1]+e[2]; i++ {
					if mdCovered(sh, i) {
						t.Fatalf("%q: a span %+v's marker byte %d of line %d shown", lines, sp, i, e[0])
					}
				}
			}
		}
		for j, l := range lines {
			b := mdBuild{src: l, trace: true}
			b.show(ks[j], 40, stText)
			if from, part, _ := mdSegment(l, ks[j]); part != mdParaNone && !slices.Equal(b.paraAt, []int{from}) {
				t.Fatalf("%q, line %d %q: its parse began at %v, its paragraph's text at %d", lines, j, l, b.paraAt, from)
			}
		}
	}
	if found < 100 {
		t.Errorf("%d spans over lines in the random paragraphs", found)
	}
}
