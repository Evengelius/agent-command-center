package render

import (
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"
	"unicode"
	"unicode/utf8"

	"charm.land/lipgloss/v2"

	"github.com/Evengelius/agent-command-center/internal/state"
)

// searchConv is a conversation built entry by entry for the matcher's
// tests, its tool calls with it.
type searchConv struct{ s *state.State }

func newSearchConv() *searchConv {
	s := state.New()
	s.Screen = state.ScreenDashboard
	return &searchConv{s: s}
}

func (c *searchConv) add(e state.Entry) int {
	c.s.Conversation = append(c.s.Conversation, e)
	return len(c.s.Conversation) - 1
}

func (c *searchConv) tool(id, name, input string, status state.ToolStatus, out string) int {
	c.s.Tools[id] = &state.ToolCall{ID: id, Name: name, Input: json.RawMessage(input), Status: status, Output: out}
	return c.add(state.Entry{Role: "tool", MessageID: id, Text: name})
}

// found is the matches of q, newest first.
func (c *searchConv) found(q string) []ConvMatch {
	ms, _ := ConvMatches(c.s, UI{}, q, nil)
	return ms
}

// TestConvMatchesKinds is criterion 66.1: a query found in each kind of
// entry design D3 names, with its entry, line and offset, a long paste's
// folded line, a folded run's call, a tool row's target by a word its cut
// would hide; nothing found where D3 says no search goes.
func TestConvMatchesKinds(t *testing.T) {
	c := newSearchConv()
	user := c.add(state.Entry{Role: "user", Text: "please find the alpha bug"})
	var paste []string
	for i := range 30 {
		paste = append(paste, fmt.Sprintf("log line %d", i))
	}
	paste[12] = "connection refused on port 5432"
	long := c.add(state.Entry{Role: "user", Text: strings.Join(paste, "\n")})
	answer := c.add(state.Entry{Role: "assistant", Text: "The **beta** report\nis ready"})
	read := c.tool("r1", "Read", `{"file_path":"C:\\Users\\name\\proj\\internal\\gamma\\viewer.go"}`, state.ToolCompleted, "")
	bash := c.tool("b1", "Bash", `{"command":"go test ./delta/..."}`, state.ToolCompleted, "pi output")
	failed := c.tool("f1", "Bash", `{"command":"curl localhost"}`, state.ToolFailed, "\n  epsilon: connection reset\nmore")
	agent := c.tool("a1", "Agent", `{"subagent_type":"general-purpose","description":"zeta survey"}`, state.ToolCompleted, "")
	write := c.tool("w1", "Write", `{"file_path":"notes.md","content":"omicron text"}`, state.ToolCompleted, "")
	t0 := time.Date(2026, 10, 29, 12, 0, 0, 0, time.UTC)
	thought := c.add(state.Entry{Role: "thinking", Thought: &state.Thought{ID: 1, Start: t0, End: t0.Add(2 * time.Second)}})
	c.add(state.Entry{Role: "thinking", Thought: &state.Thought{ID: 2, Start: t0}}) // live: its card, not searched
	approval := c.add(state.Entry{Role: "approval", Text: "eta request\nhidden detail mu"})
	// An answered request and one waiting: each its target too, as its row
	// or its box writes it, the user folder hidden; not what it would write.
	c.s.Approvals["ap1"] = &state.Approval{ID: "ap1", Tool: "Write", Status: state.Allowed,
		Input: json.RawMessage(`{"file_path":"C:\\Users\\name\\proj\\upsilon.txt","content":"chi body"}`)}
	asked := c.add(state.Entry{Role: "approval", MessageID: "ap1", Text: "Write"})
	c.s.Approvals["ap2"] = &state.Approval{ID: "ap2", Tool: "Bash", Status: state.Pending, Input: json.RawMessage(`{"command":"phi --check"}`)}
	waiting := c.add(state.Entry{Role: "approval", MessageID: "ap2", Text: "Bash"})
	notice := c.add(state.Entry{Role: "notice", Text: "theta note"})
	failure := c.add(state.Entry{Role: "error", Text: "iota failed\ntrace nu"})
	note := c.add(state.Entry{Role: "dashboard", Text: "· kappa command held"})
	c.add(state.Entry{Role: "assistant", Thinking: true}) // a thinking mark: never drawn
	var run []int
	for k := range 6 { // a run of six calls: B folds the first three
		run = append(run, c.tool(fmt.Sprintf("l%d", k), "Read", fmt.Sprintf(`{"file_path":"lambda%d.go"}`, k), state.ToolCompleted, ""))
	}
	c.s.Stages = []state.Stage{{N: 1, Review: state.TurnReview{Files: []state.ReviewFile{{Status: 'M', Path: "rho.go"}}}}}
	c.s.Agents["x"] = &state.Agent{Key: "x", Summary: "sigma summary"}

	details := entryLines(c.s, c.s.Conversation[thought])
	if len(details) < 2 {
		t.Fatalf("the phase has no details: %q", details)
	}
	word := strings.Fields(details[1])[0]
	for _, tc := range []struct {
		q    string
		want []ConvMatch
	}{
		{"alpha", []ConvMatch{{Item: user, Line: 0, Off: 16, N: 5}}},
		{"refused", []ConvMatch{{Item: long, Line: 12, Off: 11, N: 7}}},
		{"beta", []ConvMatch{{Item: answer, Line: 0, Off: 6, N: 4}}},
		{"gamma", []ConvMatch{{Item: read, Line: 0, Off: strings.Index(`C:\Users\…\proj\internal\gamma\viewer.go`, "gamma"), N: 5, Target: true}}},
		{"delta", []ConvMatch{{Item: bash, Line: 0, Off: 10, N: 5, Target: true}}},
		{"epsilon", []ConvMatch{{Item: failed, Line: 1, Off: 2, N: 7}}}, // the output's line as received, its indent kept
		{"zeta", []ConvMatch{{Item: agent, Line: 0, Off: len("general-purpose · "), N: 4, Target: true}}},
		{"notes.md", []ConvMatch{{Item: write, Line: 0, Off: 0, N: 8, Target: true}}},
		{"eta request", []ConvMatch{{Item: approval, Line: 0, Off: 0, N: 11}}},
		{"upsilon", []ConvMatch{{Item: asked, Line: 0, Off: strings.Index(`C:\Users\…\proj\upsilon.txt`, "upsilon"), N: 7, Target: true}}},
		{"phi --", []ConvMatch{{Item: waiting, Line: 0, Off: 0, N: 6, Target: true}}},
		{"theta", []ConvMatch{{Item: notice, Line: 0, Off: 0, N: 5}}},
		{"iota", []ConvMatch{{Item: failure, Line: 0, Off: 0, N: 4}}},
		{"kappa", []ConvMatch{{Item: note, Line: 0, Off: len("· "), N: 5}}},
		{"lambda0", []ConvMatch{{Item: run[0], Line: 0, Off: 0, N: 7, Target: true}}},
		{strings.ToLower(word), []ConvMatch{{Item: thought, Line: 1, Off: strings.Index(strings.ToLower(details[1]), strings.ToLower(word)), N: len(word)}}},
	} {
		if got := c.found(tc.q); !slices.Equal(got, tc.want) {
			t.Errorf("%q: %+v, want %+v", tc.q, got, tc.want)
		}
	}
	// Not searched (design D3): a call's input past its target and its
	// output, an approval's and an error's later lines, the turn review's
	// rows, zone C, a hidden user folder.
	for _, q := range []string{"omicron", "pi output", "mu", "nu", "rho", "sigma", "name", "chi body"} {
		if got := c.found(q); len(got) != 0 {
			t.Errorf("%q found: %+v", q, got)
		}
	}
	// Nor the draft, in the editor's view, nor the labels B draws around the
	// entries.
	ui := UI{Editor: EditorView{Lines: []string{"tau draft"}, Total: 1}}
	for _, q := range []string{"tau", "YOU", "CLAUDE", "RESULT"} {
		if got, _ := ConvMatches(c.s, ui, q, nil); len(got) != 0 {
			t.Errorf("%q found: %+v", q, got)
		}
	}
}

// TestSearchPlaces is design D5's places, where a search starts and what is
// at or above it: a row shown up to its last byte, so that a match from the
// next byte on, on the next row, is below it; a long paste's fold row for
// its folded lines; a call's row for its whole entry; a label for none.
func TestSearchPlaces(t *testing.T) {
	if p, ok := rowEnd(crow{entry: 3, src: "abcdef", line: 2, r: rowRange{0, 6}}, "assistant"); !ok || p != (SearchPos{2, 2, 5}) {
		t.Errorf("a text row's place %+v (%v), want entry 2, line 2, its last byte 5", p, ok)
	}
	if p, ok := rowEnd(crow{entry: 3, more: true}, "user"); !ok || p != (SearchPos{2, AllLines, 0}) {
		t.Errorf("a long paste's fold row's place %+v (%v), want its whole entry", p, ok)
	}
	if p, ok := rowEnd(crow{entry: 3}, "tool"); !ok || p != (SearchPos{2, AllLines, 0}) {
		t.Errorf("a call's row's place %+v (%v), want its whole entry", p, ok)
	}
	if _, ok := rowEnd(crow{entry: 3}, "user"); ok {
		t.Error("a message's label row has a place")
	}
	at := SearchPos{Item: 2, Line: 2, Off: 5}
	for _, tc := range []struct {
		m    ConvMatch
		want bool
	}{
		{ConvMatch{Item: 2, Line: 2, Off: 5}, true},  // at the place
		{ConvMatch{Item: 2, Line: 2, Off: 6}, false}, // the next byte, on the next row
		{ConvMatch{Item: 2, Line: 1, Off: 99}, true}, // an earlier line
		{ConvMatch{Item: 2, Line: 3, Off: 0}, false}, // a later line
		{ConvMatch{Item: 1, Line: 50, Off: 0}, true}, // an earlier entry
		{ConvMatch{Item: 3, Line: 0, Off: 0}, false}, // a later entry
	} {
		if got := tc.m.AtOrBefore(at); got != tc.want {
			t.Errorf("%+v at or before %+v: %v, want %v", tc.m, at, got, tc.want)
		}
	}
	if !(ConvMatch{Item: 2, Line: 40}).AtOrBefore(SearchPos{2, AllLines, 0}) {
		t.Error("a folded line not at or before its fold row's place")
	}
}

// TestConvMatchesCase is criterion 66.2's case: a lower-case query ignores
// case, ASCII and the letters whose cases begin with the same byte; a
// capital keeps it.
func TestConvMatchesCase(t *testing.T) {
	c := newSearchConv()
	c.add(state.Entry{Role: "assistant", Text: "ERROR Error error\nÉcole école\nДом дом\nΑλφα αλφα"})
	for _, tc := range []struct {
		q string
		n int
	}{{"error", 3}, {"Error", 1}, {"ERROR", 1}, {"école", 2}, {"École", 1}, {"дом", 2}, {"Дом", 1}, {"αλφα", 2}, {"Αλφα", 1}} {
		if got := c.found(tc.q); len(got) != tc.n {
			t.Errorf("%q: %d matches %+v, want %d", tc.q, len(got), got, tc.n)
		}
	}
}

// TestConvMatchesLiteral is criterion 66.2's syntax: no pattern, every
// character as typed, the matches never overlapping.
func TestConvMatchesLiteral(t *testing.T) {
	c := newSearchConv()
	c.add(state.Entry{Role: "assistant", Text: `a.b axb (x) \d d * ** eeeee`})
	for _, tc := range []struct {
		q    string
		offs []int
	}{{"a.b", []int{0}}, {"(x)", []int{8}}, {`\d`, []int{12}}, {"*", []int{20, 19, 17}}, {"ee", []int{24, 22}}, {"", nil}, {strings.Repeat("e", 40), nil}} {
		var offs []int
		for _, m := range c.found(tc.q) {
			offs = append(offs, m.Off)
		}
		if !slices.Equal(offs, tc.offs) {
			t.Errorf("%q: offsets %v, want %v (newest first)", tc.q, offs, tc.offs)
		}
	}
}

// TestConvMatchesOrder is design D5's order: the newest first, a later
// entry before an earlier one, and within a tool row its error, then its
// target, then its name, the reverse of how the row draws them.
func TestConvMatchesOrder(t *testing.T) {
	c := newSearchConv()
	first := c.add(state.Entry{Role: "user", Text: "omega one\nomega two"})
	call := c.tool("o1", "omega", `{"command":"omega run"}`, state.ToolFailed, "omega broke")
	got := c.found("omega")
	want := []ConvMatch{
		{Item: call, Line: 1, Off: 0, N: 5},
		{Item: call, Line: 0, Off: 0, N: 5, Target: true},
		{Item: call, Line: 0, Off: 0, N: 5},
		{Item: first, Line: 1, Off: 0, N: 5},
		{Item: first, Line: 0, Off: 0, N: 5},
	}
	if !slices.Equal(got, want) {
		t.Errorf("the order: %+v\nwant %+v", got, want)
	}
}

// TestConvMatchesCap is criterion 66.2's cap: past MaxMatches the newest
// are the ones kept, where the search starts (design D4).
func TestConvMatchesCap(t *testing.T) {
	c := newSearchConv()
	for i := range 120 {
		c.add(state.Entry{Role: "assistant", Text: strings.Repeat("x ", 100) + fmt.Sprint(i)})
	}
	ms, capped := ConvMatches(c.s, UI{}, "x", nil)
	if len(ms) != MaxMatches || !capped {
		t.Fatalf("%d matches, capped %v: want %d and capped", len(ms), capped, MaxMatches)
	}
	if ms[0].Item != 119 || ms[0].Off != 198 || ms[len(ms)-1].Item != 20 {
		t.Errorf("kept from %+v to %+v: want the newest, entry 119's last x to entry 20's first", ms[0], ms[len(ms)-1])
	}
	if ms, capped := ConvMatches(c.s, UI{}, "119", nil); len(ms) != 1 || capped {
		t.Errorf("a query under the cap: %d, capped %v", len(ms), capped)
	}
}

// TestConvMatchesMemo: the targets are read once while the bar is open,
// one a call, and found the same with the memo as without it.
func TestConvMatchesMemo(t *testing.T) {
	c := newSearchConv()
	c.tool("r1", "Read", `{"file_path":"one.go"}`, state.ToolCompleted, "")
	c.tool("r2", "Read", `{"file_path":"two.go"}`, state.ToolCompleted, "")
	m := &SearchMemo{}
	with, _ := ConvMatches(c.s, UI{}, ".go", m)
	without, _ := ConvMatches(c.s, UI{}, ".go", nil)
	if !slices.Equal(with, without) || len(with) != 2 {
		t.Fatalf("with the memo %+v, without %+v", with, without)
	}
	if len(m.targets) != 2 || m.targets["r1"] != "one.go" {
		t.Errorf("the memo holds %v, want one target a call", m.targets)
	}
	m.targets["r1"] = "kept.go" // what the memo holds is what the next search reads
	if again, _ := ConvMatches(c.s, UI{}, "kept", m); len(again) != 1 {
		t.Errorf("the memo not read: %+v", again)
	}
}

// TestConvMatchesLikeRegexp is criterion 66.2's reference: on sessions of
// the probe's shape and random queries, exactly the matches the file
// viewer's way finds (a literal regexp, (?i) without a capital) over the
// same parts of the same entries.
func TestConvMatchesLikeRegexp(t *testing.T) {
	b := newConvBench("claude-code", "claude-haiku-5-5")
	for turn := range 12 {
		b.prompt(fmt.Sprintf("Turn %d: look at the telemetry CODE and tell me what changed.", turn))
		b.calls(turn*6, 6)
		var ans strings.Builder
		for k := range 30 {
			ans.WriteString(searchAnswerLine(turn*30+k) + "\n")
		}
		b.msg(fmt.Sprintf("m%d", turn), ans.String())
		b.end("completed")
	}
	s := b.s
	type part struct {
		item, line int
		target     bool
		text       string
	}
	var parts []part
	for i := range s.Conversation {
		e := &s.Conversation[i]
		if !searchable(UI{}, e) {
			continue
		}
		switch e.Role {
		case "tool":
			parts = append(parts, part{i, 0, false, e.Text})
			if d, _ := toolDetail(s.Tools[e.MessageID]); d != "" {
				parts = append(parts, part{i, 0, true, d})
			}
		default:
			for j, l := range strings.Split(e.Text, "\n") {
				parts = append(parts, part{i, j, false, l})
			}
		}
	}
	reference := func(q string) []ConvMatch {
		pat := regexp.QuoteMeta(q)
		if q == strings.ToLower(q) {
			pat = "(?i)" + pat
		}
		re := regexp.MustCompile(pat)
		var out []ConvMatch
		for _, p := range parts {
			for _, loc := range re.FindAllStringIndex(p.text, -1) {
				out = append(out, ConvMatch{Item: p.item, Line: p.line, Off: loc[0], N: loc[1] - loc[0], Target: p.target})
			}
		}
		return out
	}
	key := func(m ConvMatch) string {
		return fmt.Sprintf("%06d %04d %v %06d %d", m.Item, m.Line, m.Target, m.Off, m.N)
	}
	sorted := func(ms []ConvMatch) []string {
		var out []string
		for _, m := range ms {
			out = append(out, key(m))
		}
		slices.Sort(out)
		return out
	}
	r := rand.New(rand.NewPCG(66, 2026))
	queries := []string{"telemetry", "TELEMETRY", "code", "CODE", "fox", "日本語", "**bold**", "`code`", "f1.go", "e", "the quick"}
	for range 300 {
		p := parts[r.IntN(len(parts))].text
		if p == "" {
			continue
		}
		from := r.IntN(len(p))
		for from > 0 && !utf8.RuneStart(p[from]) {
			from--
		}
		to := min(from+1+r.IntN(12), len(p))
		for to < len(p) && !utf8.RuneStart(p[to]) {
			to++
		}
		q := p[from:to]
		if r.IntN(3) == 0 {
			r0, size := utf8.DecodeRuneInString(q)
			q = string(unicode.ToUpper(r0)) + q[size:]
		} else if r.IntN(2) == 0 {
			q = strings.ToLower(q)
		}
		queries = append(queries, q)
	}
	for _, q := range queries {
		got, capped := ConvMatches(s, UI{}, q, nil)
		want := reference(q)
		if capped || !slices.Equal(sorted(got), sorted(want)) {
			t.Errorf("%q: %d matches, the regexp's %d (capped %v)", q, len(got), len(want), capped)
		}
	}
}

// searchAnswerLine is an answer's line of the probe's session (plan 66,
// What was found; after internal/app's answerLine): headings, code,
// Markdown and CJK text among them.
func searchAnswerLine(i int) string {
	switch {
	case i%100 == 0:
		return fmt.Sprintf("## Section %d", i/100)
	case i%100 >= 40 && i%100 < 60:
		return fmt.Sprintf("\tif err := step(%d); err != nil { return err }", i)
	case i%7 == 0:
		return fmt.Sprintf("- item %d: **bold** and `code` and a link [doc](https://example.com/%d)", i, i)
	}
	return fmt.Sprintf("Line %d of the answer: the quick brown fox jumps over the lazy dog, 日本語のテキスト, and then some more words.", i)
}

// TestFindIn: the scan's edges, a line of one letter read once.
func TestFindIn(t *testing.T) {
	collect := func(l, q string, fold bool) []int {
		var out []int
		findIn(l, q, fold, func(off int) { out = append(out, off) })
		return out
	}
	for _, tc := range []struct {
		l, q string
		fold bool
		want []int
	}{
		{"eeeee", "ee", true, []int{0, 2}},
		{"EeEeE", "ee", true, []int{0, 2}},
		{"EeEeE", "Ee", false, []int{0, 2}},
		{"abc", "abcd", true, nil},
		{"", "a", true, nil},
		{"xAx", "a", true, []int{1}},
		{"aXa", "X", false, []int{1}},
		{"école ÉCOLE", "école", true, []int{0, 7}},
		{strings.Repeat("a", 10000) + "b", "ab", true, []int{9999}},
	} {
		if got := collect(tc.l, tc.q, tc.fold); !slices.Equal(got, tc.want) {
			t.Errorf("findIn(%q, %q, %v) = %v, want %v", tc.l, tc.q, tc.fold, got, tc.want)
		}
	}
}

// litSpans is the text of each span of the frame drawn in st.
func litSpans(sc *Screen, st *lipgloss.Style) []string {
	var out []string
	for _, l := range sc.lines {
		for _, x := range l {
			if x.st == st {
				out = append(out, x.text)
			}
		}
	}
	return out
}

// TestSearchLit is criterion 66.6: every match in view on the amber, the
// current one in its style: in a message, inside bold text, code and a
// link's text, in CJK text, in a tool row's target where it is drawn, the
// "…" of a target's cut for a match it hides, a failure's error, a long
// paste's fold row for its folded lines; a selection over a match; the copy
// as without the search; nothing lit once the bar is closed.
func TestSearchLit(t *testing.T) {
	c := newSearchConv()
	c.s.Session.Phase = state.Ready
	c.add(state.Entry{Role: "user", Text: "find the needle please"})
	c.add(state.Entry{Role: "assistant", Text: "A **bold needle**, a `code needle`, a [needle link](https://example.com), 日本語のneedleテキスト"})
	long := `{"file_path":"C:\\work\\aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\\needle\\bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb\\cccccccccccccccccccc\\file.go"}`
	c.tool("r1", "Read", long, state.ToolCompleted, "")
	c.tool("r2", "Read", `{"file_path":"x\\needle.go"}`, state.ToolCompleted, "")
	c.tool("f1", "Bash", `{"command":"run"}`, state.ToolFailed, "needle failed")
	var paste []string
	for i := range 30 {
		paste = append(paste, fmt.Sprintf("log %d", i))
	}
	paste[12] = "the needle in the paste"
	c.add(state.Entry{Role: "user", Text: strings.Join(paste, "\n")})
	ms, _ := ConvMatches(c.s, UI{}, "needle", nil)
	ui := UI{Width: 120, Height: 60, Focus: "B", Search: SearchView{Open: true, Query: "needle", Matches: ms, Cur: 0}} // all of it in view
	sc := Draw(c.s, ui)
	lit, cur := litSpans(sc, stMatch), litSpans(sc, stMatchCur)
	if n := strings.Count(strings.Join(lit, "|"), "needle"); n != 7 {
		t.Errorf("%d matches lit, want 7 (message, bold, code, link, CJK, a visible target, an error): %q", n, lit)
	}
	if !slices.Contains(lit, "…") {
		t.Errorf("the cut target's \"…\" not lit for the match it hides: %q", lit)
	}
	if len(cur) != 1 || !strings.Contains(cur[0], "more lines") {
		t.Errorf("the current match, in the paste's folded lines, lit as %q: want its fold row", cur)
	}
	var cutRow string
	for y := range sc.lines {
		if row := plainRow(sc.lines[y]); strings.Contains(row, "file.go") {
			cutRow = row
		}
	}
	if !strings.Contains(cutRow, "…") || strings.Contains(cutRow, "needle") {
		t.Fatalf("the long target's row %q: want its needle folder cut", cutRow)
	}

	// A selection over a match: the selection's style wins.
	sel := Selection{From: Pos{'B', 0, 0, 0}, To: Pos{'B', 0, 0, 22}}
	ui.Sel = sel
	if got := litSpans(Draw(c.s, ui), stMatch); strings.Count(strings.Join(got, "|"), "needle") != 6 {
		t.Errorf("under a selection, the message's match still lit: %q", got)
	}
	// The copy is as without the search, a tool row's included.
	all := Selection{From: Pos{'B', 0, 0, 0}, To: Pos{'B', len(c.s.Conversation) - 1, 29, 10}}
	with, without := ui, ui
	with.Sel, without.Sel, without.Search = all, all, SearchView{}
	if a, b := Draw(c.s, with).Text(c.s, all), Draw(c.s, without).Text(c.s, all); a != b {
		t.Errorf("the copy changed with the search:\n%q\nwant\n%q", a, b)
	}
	// Closed: nothing lit.
	ui.Sel, ui.Search = Selection{}, SearchView{}
	if got := append(litSpans(Draw(c.s, ui), stMatch), litSpans(Draw(c.s, ui), stMatchCur)...); len(got) != 0 {
		t.Errorf("lit with the bar closed: %q", got)
	}
}

// TestSearchLitName is design D7's last case: a tool row with no room for
// its target lights its name for a match in the target.
func TestSearchLitName(t *testing.T) {
	c := newSearchConv()
	c.tool("r1", "Read", `{"file_path":"needle.go"}`, state.ToolCompleted, "")
	ms, _ := ConvMatches(c.s, UI{}, "needle", nil)
	sc := &Screen{lines: []line{{deco(" ✓ ", stDone), span{text: "Read", st: stToolName, ref: ref{panel: 'B', kind: refText, item: 0}}}}}
	lightSearch(sc, c.s, UI{Search: SearchView{Open: true, Query: "needle", Matches: ms, Cur: 0}})
	if got := litSpans(sc, stMatchCur); !slices.Equal(got, []string{"Read"}) {
		t.Errorf("a target not drawn: lit %q, want the name", got)
	}
}

// TestSearchLitApproval is criterion 66.6 for a request's target: an
// answered one's row lights its visible characters, or the "…" of its cut
// for a match it hides; a waiting one's box its target row; the copy as
// without the search.
func TestSearchLitApproval(t *testing.T) {
	c := newSearchConv()
	c.s.Session.Phase = state.Ready
	at := time.Date(2026, 10, 29, 11, 30, 0, 0, time.UTC)
	ask := func(id, tool, input string, st state.ApprovalStatus) {
		c.s.Approvals[id] = &state.Approval{ID: id, Agent: "", Tool: tool, Status: st, Input: json.RawMessage(input), RequestedAt: at, AnsweredAt: at}
		c.add(state.Entry{Role: "approval", MessageID: id, Text: tool})
	}
	ask("a1", "Write", `{"file_path":"x\\needle.txt","content":"z"}`, state.Allowed)
	ask("a2", "Write", `{"file_path":"C:\\work\\needle\\aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\\found.txt","content":"z"}`, state.Allowed)
	ask("a3", "Bash", `{"command":"echo needle"}`, state.Pending)
	c.s.Queue = []string{"a3"}
	ms, _ := ConvMatches(c.s, UI{}, "needle", nil)
	if len(ms) != 3 {
		t.Fatalf("%d matches, want the three targets: %+v", len(ms), ms)
	}
	ui := UI{Width: 120, Height: 40, Focus: "B", Search: SearchView{Open: true, Query: "needle", Matches: ms, Cur: 0}}
	sc := Draw(c.s, ui)
	lit, cur := litSpans(sc, stMatch), litSpans(sc, stMatchCur)
	if !slices.Equal(cur, []string{"needle"}) {
		t.Errorf("the current match, in the waiting request's target row: lit %q", cur)
	}
	if !slices.Contains(lit, "needle") || !slices.Contains(lit, "…") {
		t.Errorf("the answered rows: lit %q, want the visible target's needle and the cut one's \"…\"", lit)
	}
	all := Selection{From: Pos{'B', 0, 0, 0}, To: Pos{'B', len(c.s.Conversation) - 1, 0, 10}}
	with, without := ui, ui
	with.Sel, without.Sel, without.Search = all, all, SearchView{}
	if a, b := Draw(c.s, with).Text(c.s, all), Draw(c.s, without).Text(c.s, all); a != b {
		t.Errorf("the copy changed with the search:\n%q\nwant\n%q", a, b)
	}
}

// TestRetarget is design D7's references for a cut target: a path's left
// cut, its "…" standing for the bytes it hides; anything else's right cut.
func TestRetarget(t *testing.T) {
	p := `C:\work\needle\deep\folder\file.go`
	d := markTarget(4, line{deco(p, stMuted)})
	got := fitDetail(d, true, 18)
	if len(got) != 2 || got[0].text != "…" || got[0].ref.kind != refTargetCut || got[0].ref.n != len(p)-len(got[1].text) ||
		got[1].ref.kind != refTarget || got[1].ref.off != got[0].ref.n || !strings.HasSuffix(p, got[1].text) {
		t.Errorf("a path's left cut: %+v", got)
	}
	a := markTarget(5, line{deco("general-purpose", stFoldMark), deco(" · find the needle now", stMuted)})
	got = fitDetail(a, false, 22)
	last := got[len(got)-1]
	for last.text != "…" && len(got) > 1 { // the padding after the ellipsis
		got = got[:len(got)-1]
		last = got[len(got)-1]
	}
	drawn := 0
	for _, x := range got[:len(got)-1] {
		if x.ref.kind != refTarget || x.ref.off != drawn {
			t.Errorf("a drawn span %+v at %d", x, drawn)
		}
		drawn += len(x.text)
	}
	if last.ref.kind != refTargetCut || last.ref.off != drawn || last.ref.n != len("general-purpose · find the needle now")-drawn {
		t.Errorf("a right cut's \"…\": %+v (drawn %d)", last, drawn)
	}
	if fitted := fitDetail(markTarget(6, line{deco("short.go", stMuted)}), true, 40); len(fitted) != 1 || fitted[0].ref.kind != refTarget || fitted[0].ref.off != 0 {
		t.Errorf("a target that fits: %+v", fitted)
	}
}

// TestSearchLitView is criterion 66.7's light: in an entry's full view the
// query's matches, one the wrap splits across two rows lit on both.
func TestSearchLitView(t *testing.T) {
	c := newSearchConv()
	c.s.Session.Phase = state.Ready
	w := 80
	pad := strings.Repeat("x", w-2-3)
	var paste []string
	for i := range 30 {
		paste = append(paste, fmt.Sprintf("log %d", i))
	}
	paste[20] = pad + "needle and more"
	c.add(state.Entry{Role: "user", Text: strings.Join(paste, "\n")})
	ms, _ := ConvMatches(c.s, UI{}, "needle", nil)
	ui := UI{Width: w, Height: 40, Focus: "B", ViewEntry: 1, Search: SearchView{Open: true, Query: "needle", Matches: ms, Cur: 0}}
	if row, ok := EntryViewRow(c.s, ui, ms[0]); !ok || row != 20 {
		t.Fatalf("the match's row in the view %d, %v; want 20, the long line's first", row, ok)
	}
	ui.InspectHold, ui.InspectTop = true, 18
	sc := Draw(c.s, ui)
	got := litSpans(sc, stMatchCur)
	if strings.Join(got, "") != "needle" || len(got) != 2 {
		t.Errorf("the match the wrap splits: lit %q, want its two parts", got)
	}
}
