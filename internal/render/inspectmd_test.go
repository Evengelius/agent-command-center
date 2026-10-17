package render

import (
	"fmt"
	"math/rand/v2"
	"slices"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"charm.land/lipgloss/v2"

	"github.com/Evengelius/agent-command-center/internal/state"
)

// wrapBefore is wrap as it was before plan 5, item 11 split it into
// paragraphs and wrapPara, with the one change item 11 made: a rune wider
// than the row takes a row alone, where wrap looped for ever.
func wrapBefore(text string, w int) []string {
	if w < 1 {
		w = 1
	}
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\t", "    ")
	var out []string
	for _, para := range strings.Split(text, "\n") {
		para = strings.Map(func(r rune) rune {
			if r < 0x20 || r == 0x7f {
				return -1
			}
			return r
		}, para)
		if para == "" {
			out = append(out, "")
			continue
		}
		cur, cw := "", 0
		for _, word := range strings.SplitAfter(para, " ") {
			ww := textWidth(word)
			if cw+ww <= w {
				cur, cw = cur+word, cw+ww
				continue
			}
			if cur != "" {
				out = append(out, strings.TrimRight(cur, " "))
				cur, cw = "", 0
			}
			for textWidth(word) > w {
				head := cut(word, w)
				if head == "" {
					_, n := utf8.DecodeRuneInString(word)
					head = word[:n]
				}
				out = append(out, head)
				word = word[len(head):]
			}
			cur, cw = word, textWidth(word)
		}
		out = append(out, strings.TrimRight(cur, " "))
	}
	return out
}

// TestWrapUnchanged is plan 5, item 11, criterion 2's ground: wrap draws
// the rows it drew before, for texts of words, runs of spaces, tabs, line
// breaks, control and wide characters, at widths 1 to 30.
func TestWrapUnchanged(t *testing.T) {
	if got := wrap("a界b", 1); !slices.Equal(got, []string{"a", "界", "b"}) {
		t.Fatalf("wrap at 1 cell = %q", got) // a wide rune once hung wrap
	}
	parts := []string{"a", "bb", "word", "averyveryverylongword", " ", "  ", "   ", "\t", "\n", "\r\n", "\r", "\x01", "\x1f", "\x7f", "界", "é", "**x**", "`c`"}
	r := rand.New(rand.NewPCG(11, 2026))
	for n := range 3000 {
		var b strings.Builder
		for range r.IntN(24) {
			b.WriteString(parts[r.IntN(len(parts))])
		}
		text, w := b.String(), 1+n%30
		if got, want := wrap(text, w), wrapBefore(text, w); !slices.Equal(got, want) {
			t.Fatalf("wrap(%q, %d) = %q, want %q", text, w, got, want)
		}
	}
}

// TestStyledRow: a row takes each run's style over the bytes it shares with
// it, and the base style past the last run.
func TestStyledRow(t *testing.T) {
	runs := []styleRun{{to: 3, st: stMdMark}, {to: 6, st: stMdCode}}
	got := fmt.Sprint(spansOf(styledRow("ab cdefgh", rowRange{1, 8}, runs, stText)))
	want := fmt.Sprint(spansOf(line{sp("b ", stMdMark), sp("cde", stMdCode), sp("fg", stText)}))
	if got != want {
		t.Errorf("styledRow = %s, want %s", got, want)
	}
}

// inspectMDState is an inspected agent with the Markdown fixture's two AI
// messages around a long notice, an end, an error and a history line.
func inspectMDState() *state.State {
	at := time.Date(2026, 10, 17, 9, 30, 5, 0, time.Local)
	s := state.New()
	s.Screen = state.ScreenDashboard
	s.Session.Phase = state.Ready
	s.Agents["a1"] = &state.Agent{Key: "a1", Label: "general-purpose", Description: "fix the plan", Status: state.AgentCompleted,
		StartedAt: at, EndedAt: at.Add(time.Minute), Summary: "done", Error: "a late error", History: []string{"a history line"},
		Messages: []state.Entry{
			{Role: "assistant", Text: mdFixture[1].Text, At: at},
			{Role: "notice", Text: "a notice with **stars** and `ticks` that is long enough to wrap onto a second row at eighty columns", At: at},
			{Role: "assistant", Text: mdFixture[3].Text, At: at},
		}}
	return s
}

// TestInspectMarkdown is plan 5, item 11, criteria 1 to 3: the Messages
// tab styles an agent's AI messages as zone B does, draws and copies the
// same text as unstyled, and leaves the rest as it was.
func TestInspectMarkdown(t *testing.T) {
	defer func() { markdownOn = true }()
	s := inspectMDState()
	draw := func(ui UI, md bool) *Screen {
		markdownOn = md
		defer func() { markdownOn = true }()
		return Draw(s, ui)
	}
	// Criterion 2: the same characters at the same places, scrolled from
	// top to bottom; a selection over the shown rows copies the same text.
	for _, size := range [][2]int{{80, 24}, {120, 40}, {200, 60}} {
		ui := UI{Width: size[0], Height: size[1], Inspect: "a1", InspectHold: true}
		sc := draw(ui, true)
		for top := 0; ; top++ {
			ui.InspectTop = top
			md, plain := draw(ui, true), draw(ui, false)
			if a, b := texts(md.lines), texts(plain.lines); a != b {
				t.Fatalf("%v top %d: styled text differs from plain:\n%s\n--- plain ---\n%s", size, top, a, b)
			}
			// A cell hits from its row's units; every row's are compared, and
			// HitIn itself on every fifth column.
			for y := range size[1] {
				if a, b := md.units('I', y), plain.units('I', y); !slices.Equal(a, b) {
					t.Fatalf("%v top %d: row %d units %v, plain %v", size, top, y, a, b)
				}
				for x := 0; x < size[0]; x += 5 {
					p1, q1, ok1 := md.HitIn('I', x, y, false)
					p2, q2, ok2 := plain.HitIn('I', x, y, false)
					if p1 != p2 || q1 != q2 || ok1 != ok2 {
						t.Fatalf("%v top %d: cell %d,%d hits %v %v, plain %v %v", size, top, x, y, p1, q1, p2, q2)
					}
				}
			}
			v := md.Inspect
			sel := Selection{Pos{'I', v.Top, 0, 0}, Pos{'I', v.Top + v.Room - 1, 0, 1 << 20}}
			if a, b := md.Text(s, sel), plain.Text(s, sel); a != b || a == "" {
				t.Fatalf("%v top %d: copied %q, plain %q", size, top, a, b)
			}
			if top >= sc.Inspect.Rows-sc.Inspect.Room {
				break
			}
		}
	}
	// Criterion 1: each construct takes zone B's style. 80 columns wrap the
	// long code span; 200 rows show the whole tab.
	ui := UI{Width: 80, Height: 200, Inspect: "a1"}
	md, plain := draw(ui, true), draw(ui, false)
	for _, c := range []struct {
		text string
		st   *lipgloss.Style
	}{
		{"#", stMdMark},
		{"Plan for the fix", stMdHeading},
		{"bold", stMdStrong},
		{"emphasis", stMdEm},
		{"inline code", stMdCode},
		{"link", stMdLink},
		{"a quoted line", stMdEm},
		{"- ", stMdBullet},
		{"2. ", stMdBullet},
		{"|", stMdMark},
		{"|------|-------|", stMdMark},
		{"go", stMdLang},
		{"func", stMdKeyword},
		{"// a comment here", stMdComment},
		{`"a string"`, stMdString},
		{"1234", stMdNumber},
		{"over lines */", stMdComment}, // a block comment's second line (item 12)
		{"7", stMdNumber},              // after its end
		{"sure", stMdCode},             // a code span's row after the wrap
		{"def", stMdKeyword},           // in the unclosed fence
		{"# note", stMdComment},
	} {
		if got := panelStyle(md, 'I', c.text); got != c.st {
			t.Errorf("%q styled %v, want %v", c.text, got, c.st)
		}
		if got := panelStyle(plain, 'I', c.text); got != stText && got != stGrey {
			t.Errorf("plain %q styled %v", c.text, got)
		}
	}
	// Criterion 3: every row that is not an AI message's is drawn as before.
	// An AI message's rows are its wrapped rows, the first after the time,
	// the others after ten spaces, as the tab lays them out.
	ai, firsts := map[string]bool{}, []string{}
	for _, e := range s.Agents["a1"].Messages {
		if e.Role == "assistant" {
			rs := wrap(e.Text, 80-2-10)
			firsts = append(firsts, clockSec(e.At)+"  "+rs[0])
			for _, r := range rs[1:] {
				ai[strings.TrimRight("          "+r, " ")] = true
			}
		}
	}
	isAI := func(row string) bool {
		body := strings.TrimRight(row[1:], " \n")
		return ai[body] || slices.ContainsFunc(firsts, func(f string) bool { return strings.HasPrefix(body, f) })
	}
	others := 0
	for y, l := range md.lines {
		row := texts([]line{l})
		if isAI(row) {
			continue
		}
		if a, b := fmt.Sprint(spansOf(l)), fmt.Sprint(spansOf(plain.lines[y])); a != b {
			t.Errorf("row %d %q differs:\n%s\nplain\n%s", y, row, a, b)
		}
		others++
	}
	for _, want := range []string{"started by orchestrator", "a notice with **stars**", "second row at eighty columns", "completed", "a late error", "a history line"} {
		if !strings.Contains(texts(md.lines), want) {
			t.Errorf("no %q", want)
		}
	}
	if others < 8 {
		t.Errorf("only %d rows compared", others)
	}
}

// panelStyle is the style of panel p's first span whose text is text, or
// else that contains it.
func panelStyle(sc *Screen, p byte, text string) *lipgloss.Style {
	var near *lipgloss.Style
	for _, l := range sc.lines {
		for _, x := range l {
			if x.ref.panel != p {
				continue
			}
			if x.text == text {
				return x.st
			}
			if near == nil && strings.Contains(x.text, text) {
				near = x.st
			}
		}
	}
	return near
}

// spansOf is a row's spans: text, style and kind, to compare two rows.
func spansOf(l line) []string {
	var out []string
	for _, x := range l {
		st := "-"
		if x.st != nil {
			st = x.st.Render("x")
		}
		out = append(out, fmt.Sprintf("%q %s %d", x.text, st, x.ref.kind))
	}
	return out
}
