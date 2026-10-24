package render

import (
	"reflect"
	"regexp"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"

	"github.com/Evengelius/agent-command-center/internal/event"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// Plan 34 (brief amendment 34, references/11-model-selector.webp;
// reference-design.md, The model selector): the selector's columns, rows
// and colours, read cell by cell from the drawn screen.

var testModels = []event.ModelEntry{
	{ID: "default", Model: "claude-opus-5-5", Name: "Default (recommended)", Description: "Opus 5.5 · Best for everyday, complex tasks", Efforts: []string{"low", "medium", "high", "xhigh", "max"}},
	{ID: "opus", Model: "claude-opus-5-5", Name: "Opus 5.5", Description: "For complex work and everyday tasks", Efforts: []string{"low", "medium", "high", "xhigh", "max"}},
	{ID: "fable", Model: "claude-fable-5-1", Name: "Fable 5.1", Description: "For your toughest challenges", Efforts: []string{"low", "medium", "high", "xhigh", "max"}},
	{ID: "sonnet", Model: "claude-sonnet-5-5", Name: "Sonnet 5.5", Description: "Most efficient for simpler tasks", Efforts: []string{"low", "medium", "high", "xhigh", "max"}},
	{ID: "haiku", Model: "claude-haiku-5-5", Name: "Haiku 5.5", Description: "Fastest for quick answers", Efforts: []string{"low", "medium", "high", "xhigh", "max"}},
	{ID: "claude-haiku-4-5-20251001", Model: "claude-haiku-4-5-20251001", Name: "Haiku 4.5", Description: "Fastest for quick answers"},
}

// selectorState is the dashboard of plan 34's mock of the CLI (a session
// with no agent yet, as P34c's and P34x's), with the given models listed
// and in force the given settings.
func selectorState(t *testing.T, cli string, models []event.ModelEntry, in event.Settings) *state.State {
	t.Helper()
	mock := "claude-models.jsonl"
	if cli == "codex" {
		mock = "codex-models.jsonl"
	}
	s, _ := stampedMock(t, mock, appClock)
	s.Models = state.Models{Entries: models, InForce: in}
	s.Session.Model = in.Model
	return s
}

// spanAt is the span that covers column x of l, and the column it starts at.
func spanAt(l line, x int) (span, int) {
	c := 0
	for _, s := range l {
		w := textWidth(s.text)
		if x >= c && x < c+w {
			return s, c
		}
		c += w
	}
	return span{}, -1
}

// boxOf finds the selector on the screen: its rows' text, from its top edge
// to its bottom edge, and its left column and top row.
func boxOf(t *testing.T, sc *Screen) (rows []string, x, y int) {
	t.Helper()
	var lines [][]rune // every cell of these frames is one rune wide
	for _, l := range strings.Split(strings.TrimSuffix(texts(sc.lines), "\n"), "\n") {
		lines = append(lines, []rune(l))
	}
	for i := 0; i+1 < len(lines); i++ {
		for x, r := range lines[i] {
			next := string(lines[i+1][x:])
			if r != '╭' || !strings.HasPrefix(next, "│ CLAUDE MODELS") && !strings.HasPrefix(next, "│ CODEX MODELS") {
				continue
			}
			w := strings.IndexRune(string(lines[i][x:]), '╮')
			w = len([]rune(string(lines[i][x:])[:w])) + 1
			for k := i; k < len(lines); k++ {
				rows = append(rows, string(lines[k][x:x+w]))
				if lines[k][x] == '╰' {
					return rows, x, i
				}
			}
		}
	}
	t.Fatalf("no selector on the screen:\n%s", texts(sc.lines))
	return nil, 0, 0
}

// col is the column of s in the box's row r, -1 when it is not there.
func col(r, s string) int {
	if i := strings.Index(r, s); i >= 0 {
		return textWidth(r[:i])
	}
	return -1
}

// TestSelectorGeometry is criterion 20's cell part: at 120x40, Claude Code's
// selector is the reference's: 63 cells wide (the longest description
// needs less), 25 rows, the header, the names, "▶", "✓", the separators,
// More models, the effort box and Ultracode at their columns, the bottom
// edge right under Ultracode with "▾" above the pill's first cell, no
// footer; the band from column 2 to W-3 on the cursor's two rows.
func TestSelectorGeometry(t *testing.T) {
	defer UsePalette(pickTrue)
	in := event.Settings{Model: "claude-fable-5-1", Choice: "fable", Effort: "xhigh", EffortKnown: true, Ultracode: true, UltracodeAvailable: true, UltracodeKnown: true}
	s := selectorState(t, "claude-code", testModels, in)
	ui := UI{Width: 120, Height: 40, Now: appClock, Focus: "B"}
	ui.Models = OpenSelector(s, true, 17)
	sc := Draw(s, ui)
	rows, bx, by := boxOf(t, sc)
	const w = 63
	if len(rows) != 25 {
		t.Fatalf("%d rows, want 25:\n%s", len(rows), strings.Join(rows, "\n"))
	}
	for i, r := range rows {
		if n := textWidth(r); n != w {
			t.Errorf("row %d is %d cells, want %d: %q", i, n, w, r)
		}
	}
	at := func(row int, s string, want int) {
		t.Helper()
		if got := col(rows[row], s); got != want {
			t.Errorf("row %d: %q at column %d, want %d: %q", row, s, got, want, rows[row])
		}
	}
	at(1, "CLAUDE MODELS", 2)
	at(1, "Current session", w-3-len("Current session")+1)
	at(3, "Default (recommended)", 5)
	at(4, "Opus 5.5 · Best", 5)
	at(9, "▶ Fable 5.1", 3)
	at(9, "✓", w-5)
	at(10, "For your toughest challenges", 5)
	at(16, "Fastest for quick answers", 5)
	at(17, "───", 2)
	at(18, "▶ More models", 3)
	at(18, "›", w-5)
	at(19, "───", 2)
	at(21, "Effort:", 3)
	at(21, "XHigh", 14)
	at(21, "⌄", 25)
	at(23, "Ultracode", 3)
	at(23, "Enables dynamic workflows", 13)
	at(23, toggleOnText, w-8) // the text toggle, on, without Sixel (plan 35, design D3)
	if !strings.HasPrefix(rows[24], "╰▾─") || !strings.HasSuffix(rows[24], "╯") || strings.TrimSpace(strings.Trim(rows[22], "│")) != "" {
		t.Errorf("the rows under Effort: %q, %q, %q", rows[22], rows[23], rows[24])
	}
	for _, r := range []int{2, 5, 20, 22} {
		if strings.TrimSpace(strings.Trim(rows[r], "│┃")) != "" {
			t.Errorf("row %d not blank: %q", r, rows[r])
		}
	}
	// The pill's first cell is under the tail; the bottom edge is the
	// editor's rule.
	px, py, _ := sc.find(PillPanel)
	if bx+1 != px || sc.editorRule(py) != by+24 {
		t.Errorf("box at %d,%d; pill at %d,%d; the rule at %d", bx, by, px, py, sc.editorRule(py))
	}
	// The colours: the frame and the header lavender, the band on the
	// cursor's rows from column 2 to column W-3, the pink "▶".
	frame, _ := spanAt(sc.lines[by], bx)
	head, _ := spanAt(sc.lines[by+1], bx+2)
	mark, _ := spanAt(sc.lines[by+9], bx+3)
	if frame.st != stModelFrame || head.st != stModelHead || mark.st != styleBand(stModelMark) {
		t.Errorf("styles: frame %v, header %v, mark %v", frame.st, head.st, mark.st)
	}
	for _, r := range []int{9, 10} {
		for c := 1; c < w-1; c++ {
			sp, _ := spanAt(sc.lines[by+r], bx+c)
			_, none := sp.st.GetBackground().(lipgloss.NoColor)
			if band := c >= 2 && c <= w-3; band == none {
				t.Errorf("row %d column %d on the band %v, want %v", r, c, !none, band)
				break
			}
		}
	}
	if box, _ := spanAt(sc.lines[by+21], bx+13); box.st != boxGround.on(stBright) {
		t.Errorf("the effort box styled %v", box.st)
	}
	// The description, word for word (amendment 34).
	if !strings.Contains(rows[23], " Ultracode Enables dynamic workflows ") || strings.Contains(texts(sc.lines), "XHigh effort") {
		t.Errorf("the Ultracode row %q", rows[23])
	}
}

func styleBand(st *lipgloss.Style) *lipgloss.Style { return bandGround.on(st) }

// TestSelectorStates is criteria 6, 8, 11 and the Codex form: a model
// without levels and an effort a variable holds show their box without a
// chevron; an unavailable Ultracode is dimmed and off; Codex's selector has
// no Ultracode row, its bottom edge under Effort, and marks its default
// model and a change waiting for the next message.
func TestSelectorStates(t *testing.T) {
	defer UsePalette(pickTrue)
	ui := UI{Width: 120, Height: 40, Now: appClock, Focus: "B"}
	draw := func(s *state.State, ultracode bool) []string {
		ui.Models = OpenSelector(s, ultracode, 17)
		rows, _, _ := boxOf(t, Draw(s, ui))
		return rows
	}
	haiku := selectorState(t, "claude-code", testModels, event.Settings{Model: "claude-haiku-4-5-20251001", Choice: "claude-haiku-4-5-20251001",
		EffortKnown: true, UltracodeKnown: true})
	rows := draw(haiku, true)
	effort := rows[len(rows)-4]
	if col(effort, "N/A") != 14 || strings.Contains(effort, "⌄") {
		t.Errorf("Haiku 4.5's effort row %q", effort)
	}
	ult := rows[len(rows)-2]
	sc := Draw(haiku, ui)
	_, bx, by := boxOf(t, sc)
	label, _ := spanAt(sc.lines[by+len(rows)-2], bx+3)
	if label.st != stGrey || col(ult, toggleOffText) != textWidth(ult)-8 {
		t.Errorf("the unavailable Ultracode row %q styled %v", ult, label.st)
	}
	locked := selectorState(t, "claude-code", testModels, event.Settings{Model: "claude-fable-5-1", Choice: "fable", Effort: "max", EffortKnown: true,
		EffortLock: "CLAUDE_CODE_EFFORT_LEVEL", UltracodeKnown: true, UltracodeAvailable: true})
	rows = draw(locked, true)
	if e := rows[len(rows)-4]; col(e, "Max") != 14 || strings.Contains(e, "⌄") {
		t.Errorf("the held effort's row %q", e)
	}
	codex := []event.ModelEntry{
		{ID: "gpt-6.1-sol", Model: "gpt-6.1-sol", Name: "GPT-6.1-Sol", Description: "Latest workhorse model for coding and everyday work.", Efforts: []string{"low", "medium", "high"}, DefaultEffort: "low", Default: true},
		{ID: "gpt-6-sol", Model: "gpt-6-sol", Name: "GPT-6-Sol", Description: "Previous generation workhorse model.", Efforts: []string{"low", "medium", "high"}, DefaultEffort: "medium"},
	}
	cs := selectorState(t, "codex", codex, event.Settings{Model: "gpt-6-sol", Choice: "gpt-6-sol", Effort: "high", EffortKnown: true})
	cs.Models.Asked = &state.ModelChange{Choice: "gpt-6.1-sol", Effort: "high", NextTurn: true}
	rows = draw(cs, false)
	all := strings.Join(rows, "\n")
	if !strings.Contains(rows[1], "CODEX MODELS") || strings.Contains(all, "Ultracode") || strings.Contains(all, "CLAUDE") {
		t.Errorf("Codex's selector:\n%s", all)
	}
	if !strings.Contains(rows[len(rows)-2], "Effort:") || !strings.HasPrefix(rows[len(rows)-1], "╰▾") {
		t.Errorf("Codex's last rows %q, %q", rows[len(rows)-2], rows[len(rows)-1])
	}
	if !strings.Contains(all, "GPT-6.1-Sol (default)") || col(rows[3], "…") != textWidth(rows[3])-5 || col(rows[6], "✓") != textWidth(rows[6])-5 {
		t.Errorf("the default model, the waiting change and the model in force:\n%s", all)
	}
	if f := texts(Draw(cs, UI{Width: 120, Height: 40, Now: appClock, Focus: "B"}).lines); !strings.Contains(f, "GPT-6-Sol High") ||
		!strings.Contains(f, "→ GPT-6.1-Sol High (next turn)") {
		t.Errorf("the editor row with Codex's change waiting:\n%s", f)
	}
}

// TestModelPill is criterion 3 and design D7 on the pill: the model in
// force by its entry's name (never Default's), the effort, and Ultracode
// only while it is in force.
func TestModelPill(t *testing.T) {
	for _, c := range []struct {
		in   event.Settings
		want string
	}{
		{event.Settings{Model: "claude-fable-5-1", Effort: "max", EffortKnown: true, Ultracode: true, UltracodeKnown: true}, " Fable 5.1 Max · Ultracode "},
		{event.Settings{Model: "claude-opus-5-5", Choice: "default", Effort: "medium", EffortKnown: true, UltracodeKnown: true}, " Opus 5.5 Medium "},
		{event.Settings{Model: "claude-sonnet-5-5", EffortKnown: true}, " Sonnet 5.5 "},
		{event.Settings{Model: "claude-opus-4-8"}, " Opus 4.8 "},
		{event.Settings{}, " model N/A "},
	} {
		s := state.New()
		s.Models = state.Models{Entries: testModels, InForce: c.in}
		var b strings.Builder
		for _, sp := range modelPill(s) {
			b.WriteString(sp.text)
			if sp.ref.panel != PillPanel {
				t.Errorf("%q: a cell outside panel M", c.want)
			}
		}
		if b.String() != c.want {
			t.Errorf("pill %q, want %q", b.String(), c.want)
		}
	}
}

// levelRow is a row of the effort list: its box's edge, a level and
// Codex's "(default)", then "✓" on the level in force or "…" on Codex's
// level asked for.
var levelRow = regexp.MustCompile(`│ ((?:Low|Medium|High|XHigh|Max|Ultra)(?: \(default\))?) +([✓… ]) │`)

// TestEffortList is criterion 6 on the screen: the effort list holds the
// model's own levels in the CLI's order, the level in force checked;
// Codex's default level reads "(default)", Claude Code names none. A Codex
// level waiting for the next message has a "…" and the box keeps the level
// in force, as the pill does (plan 34's review).
func TestEffortList(t *testing.T) {
	defer UsePalette(pickTrue)
	list := func(s *state.State, ultracode bool) (labels []string, checked, asked, box string) {
		t.Helper()
		ui := UI{Width: 120, Height: 40, Now: appClock, Focus: "B"}
		ui.Models = OpenSelector(s, ultracode, 17)
		ui.Models.Effort = true
		for _, l := range strings.Split(texts(Draw(s, ui).lines), "\n") {
			if m := levelRow.FindStringSubmatch(l); m != nil {
				labels = append(labels, m[1])
				switch m[2] {
				case "✓":
					checked = m[1]
				case "…":
					asked = m[1]
				}
			}
			if i := strings.Index(l, "Effort:"); i >= 0 {
				box = strings.Fields(l[i+len("Effort:"):])[0]
			}
		}
		return labels, checked, asked, box
	}
	five := []string{"low", "medium", "high", "xhigh", "max"}
	codex := []event.ModelEntry{ // P34x's
		{ID: "gpt-6-sol", Model: "gpt-6-sol", Name: "GPT-6-Sol", Description: "Previous generation workhorse model.", Efforts: append(five, "ultra"), DefaultEffort: "medium"},
		{ID: "gpt-6-luna", Model: "gpt-6-luna", Name: "GPT-6-Luna", Description: "Fast and affordable model for easier tasks.", Efforts: five, DefaultEffort: "medium"},
	}
	opus46 := append(testModels, event.ModelEntry{ID: "claude-opus-4-6", Model: "claude-opus-4-6", Name: "Opus 4.6",
		Description: "Best for everyday, complex tasks", Efforts: []string{"low", "medium", "high", "max"}})
	solHigh := event.Settings{Model: "gpt-6-sol", Choice: "gpt-6-sol", Effort: "high", EffortKnown: true}
	waiting := selectorState(t, "codex", codex, solHigh)
	waiting.Models.Asked = &state.ModelChange{Effort: "low", NextTurn: true}
	for _, c := range []struct {
		what                      string
		s                         *state.State
		claude                    bool
		want                      []string
		checked, asked, boxShowed string
	}{
		{"GPT-6-Sol", selectorState(t, "codex", codex, solHigh), false,
			[]string{"Low", "Medium (default)", "High", "XHigh", "Max", "Ultra"}, "High", "", "High"},
		{"GPT-6-Luna", selectorState(t, "codex", codex, event.Settings{Model: "gpt-6-luna", Choice: "gpt-6-luna", Effort: "medium", EffortKnown: true}), false,
			[]string{"Low", "Medium (default)", "High", "XHigh", "Max"}, "Medium (default)", "", "Medium"},
		{"GPT-6-Sol, Low for the next turn", waiting, false,
			[]string{"Low", "Medium (default)", "High", "XHigh", "Max", "Ultra"}, "High", "Low", "High"},
		{"Fable 5.1", selectorState(t, "claude-code", testModels, event.Settings{Model: "claude-fable-5-1", Choice: "fable", Effort: "max", EffortKnown: true,
			UltracodeKnown: true, UltracodeAvailable: true}), true, []string{"Low", "Medium", "High", "XHigh", "Max"}, "Max", "", "Max"},
		{"Opus 4.6", selectorState(t, "claude-code", opus46, event.Settings{Model: "claude-opus-4-6", Choice: "claude-opus-4-6", Effort: "high", EffortKnown: true,
			UltracodeKnown: true}), true, []string{"Low", "Medium", "High", "Max"}, "High", "", "High"},
	} {
		labels, checked, asked, box := list(c.s, c.claude)
		if !reflect.DeepEqual(labels, c.want) || checked != c.checked || asked != c.asked || box != c.boxShowed {
			t.Errorf("%s: levels %q, %q checked, %q asked, the box %q; want %q, %q, %q, %q", c.what, labels, checked, asked, box,
				c.want, c.checked, c.asked, c.boxShowed)
		}
	}
}

// TestSelectorMarks is plan 34's review: the model in force keeps its "✓"
// while a choice of it again waits for its answer; another model asked for
// has the grey "…".
func TestSelectorMarks(t *testing.T) {
	defer UsePalette(pickTrue)
	fable := event.Settings{Model: "claude-fable-5-1", Choice: "fable", Effort: "max", EffortKnown: true, UltracodeKnown: true, UltracodeAvailable: true}
	for _, c := range []struct {
		asked, fable, sonnet string
	}{{"fable", "✓", " "}, {"sonnet", "✓", "…"}} {
		s := selectorState(t, "claude-code", testModels, fable)
		s.Models.Asked = &state.ModelChange{Choice: c.asked}
		ui := UI{Width: 120, Height: 40, Now: appClock, Focus: "B"}
		ui.Models = OpenSelector(s, true, 17)
		rows, _, _ := boxOf(t, Draw(s, ui))
		mark := func(name string) string {
			for _, r := range rows {
				if col(r, name) == 5 {
					return string([]rune(r)[len([]rune(r))-5])
				}
			}
			return "none"
		}
		if f, so := mark("Fable 5.1"), mark("Sonnet 5.5"); f != c.fable || so != c.sonnet {
			t.Errorf("%s asked: Fable 5.1's mark %q, Sonnet 5.5's %q; want %q, %q", c.asked, f, so, c.fable, c.sonnet)
		}
	}
}

// TestSelectorTail is plan 34's review: the bottom edge's "▾" lies above the
// pill's first cell at every width, the box pushed left by the screen's
// right edge included.
func TestSelectorTail(t *testing.T) {
	defer UsePalette(pickTrue)
	in := event.Settings{Model: "claude-fable-5-1", Choice: "fable", Effort: "max", EffortKnown: true, Ultracode: true, UltracodeKnown: true, UltracodeAvailable: true}
	long := append(testModels, event.ModelEntry{ID: "x", Model: "x", Name: "X", Description: strings.Repeat("d", 60)}) // a box 72 cells wide
	pushed := 0
	for _, models := range [][]event.ModelEntry{testModels, long} {
		for _, w := range []int{80, 90, 100, 110, 120, 160, 200} {
			s := selectorState(t, "claude-code", models, in)
			ui := UI{Width: w, Height: 40, Now: appClock, Focus: "B"}
			ui.Models = OpenSelector(s, true, 17)
			sc := Draw(s, ui)
			rows, bx, _ := boxOf(t, sc)
			px, _, _ := sc.find(PillPanel)
			if got := bx + col(rows[len(rows)-1], "▾"); got != px {
				t.Errorf("%d models at width %d: the tail at column %d, the pill at %d", len(models), w, got, px)
			}
			if bx < px-1 {
				pushed++
			}
		}
	}
	if pushed == 0 {
		t.Error("no width pushed the box left of the pill")
	}
}
