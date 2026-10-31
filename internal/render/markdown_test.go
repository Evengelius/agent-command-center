package render

import (
	"fmt"
	"math/rand/v2"
	"slices"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"

	"github.com/Evengelius/agent-command-center/internal/event"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// Plan 71, brief amendment 74: AI text drawn as Markdown renders it, its
// markers hidden and its bytes kept for the wrap, the entry cursor, the
// search and the copy (ui-design.md, Markdown).

// mdShownText is the text sh's cells show.
func mdShownText(sh mdShown) string {
	var b strings.Builder
	for _, c := range sh.cells {
		b.WriteString(c.text)
	}
	return b.String()
}

// mdByteStyle is the style of byte i of sh's line; nil past its runs.
func mdByteStyle(sh mdShown, i int) *lipgloss.Style {
	for _, r := range sh.runs {
		if r.to > i {
			return r.st
		}
	}
	return nil
}

// mdCovered reports whether sh shows byte i of its line, in a cell of its
// own or a glyph standing for it.
func mdCovered(sh mdShown, i int) bool {
	for _, c := range sh.cells {
		if i >= c.off && i < c.off+c.n {
			return true
		}
	}
	return false
}

// The block kinds: fences carry over lines and close only on a fence as
// long, an unclosed one running to the end; a list item at any indent, the
// other blocks under four spaces; a table only with its header's rule.
func TestMarkdownLines(t *testing.T) {
	for _, c := range []struct {
		lines []string
		want  string
	}{
		{[]string{"# h", "#nope", "> q", "- l", "1) l", "***", "x", "#", "####### seven"}, "HPQLLRPHP"},
		{[]string{"````go", "a", "```", "b", "`````", "c"}, "FCCCFP"},
		{[]string{"~~~~", "```", "~~~", "~~~~", "d"}, "FCCFP"},
		{[]string{"text", "```", "open", "    # still code"}, "PFCC"},
		{[]string{"    # indented", "``` not ` a fence", "    > nor a quote", "   > a quote"}, "PPPQ"},
		{[]string{"- a", "  - b", "    - c", "      1. d", "- - -", "* * *", "1.no", "10. ten"}, "LLLLRRPL"},
		{[]string{"| a | b |", "|---|:-:|", "| c | d |", "text", "| lone |", "plain"}, "TTTPPP"},
		{[]string{"| h |", "| no rule |"}, "PP"},
		{[]string{"\uFEFF# a file's first line, after its byte order mark", "\uFEFF# not a first line"}, "HP"},
	} {
		var got strings.Builder
		for _, k := range mdLines(c.lines) {
			got.WriteByte("PHQLTRFCN"[k.kind])
		}
		if got.String() != c.want {
			t.Errorf("%q: %s, want %s", c.lines, got.String(), c.want)
		}
	}
	// A table's rows share their columns' widths as shown; its first row is
	// its header.
	ks := mdLines([]string{"| a | **bold** |", "|---|---|", "| ccc | d |"})
	if ks[0].cols != "3,4" || ks[1].cols != "3,4" || ks[2].cols != "3,4" || !ks[0].head || ks[1].head || ks[2].head {
		t.Errorf("table kinds %+v", ks)
	}
}

// TestMarkdownShown is criterion 71.1 (designs D1, D4 and D7): each
// construct's shown text and styles, nested, mixed, escaped, unclosed and
// malformed; its markers hidden, no control character drawn as such.
func TestMarkdownShown(t *testing.T) {
	bold := mdStyle(stText, true, false, false)
	italic := mdStyle(stText, false, true, false)
	both := mdStyle(stText, true, true, false)
	strike := mdStyle(stText, false, false, true)
	type at struct {
		text string // its first occurrence in the line, by its first byte
		st   *lipgloss.Style
	}
	table := []string{"| Name | Value |", "|------|:-----:|", "| alpha | **42** |", "| b \\| c | `x|y` |"}
	for _, c := range []struct {
		name   string
		lines  []string
		j      int    // the line shown, in its lines
		want   string // as shown in 40 cells
		at     []at
		hidden []string // each occurrence hidden
	}{
		{"strong", []string{"Some **bold** words"}, 0, "Some bold words", []at{{"bold", bold}, {"Some", stText}, {"words", stText}}, []string{"**"}},
		{"the owner's labels", []string{"**Goal:** finish, **Decisions:** none"}, 0, "Goal: finish, Decisions: none",
			[]at{{"Goal:", bold}, {"Decisions:", bold}, {"finish", stText}}, []string{"**"}},
		{"emphasis", []string{"an *emphasis* and _another_"}, 0, "an emphasis and another", []at{{"emphasis", italic}, {"another", italic}}, []string{"*", "_"}},
		{"strong emphasis", []string{"***both*** and ___too___"}, 0, "both and too", []at{{"both", both}, {"too", both}}, []string{"***", "___"}},
		{"strikethrough", []string{"~~gone~~ but ~kept~"}, 0, "gone but ~kept~", []at{{"gone", strike}, {"kept", stText}}, []string{"~~"}},
		{"nested", []string{"**bold *both* bold** and *it **both** it*"}, 0, "bold both bold and it both it",
			[]at{{"bold", bold}, {"both", both}, {"it", italic}}, nil},
		{"code spans", []string{"a `code` and ``with ` tick`` here"}, 0, "a code and with ` tick here", []at{{"code", stMdCode}, {"with", stMdCode}, {"here", stText}}, nil},
		{"code in strong", []string{"**bold `code` bold**"}, 0, "bold code bold", []at{{"bold", bold}, {"code", mdStyle(stMdCode, true, false, false)}}, nil},
		{"a code span literal", []string{"`**not bold** \\*`"}, 0, "**not bold** \\*", []at{{"**not", stMdCode}}, nil},
		{"link", []string{"see [the docs](https://example.com/a_b) now"}, 0, "see the docs (https://example.com/a_b) now",
			[]at{{"the docs", stMdLink}, {"https", stMdComment}, {"now", stText}}, []string{"["}},
		{"image", []string{"![a chart](chart.png)"}, 0, "a chart (chart.png)", []at{{"a chart", stMdLink}, {"chart.png", stMdComment}}, []string{"!["}},
		{"strong link", []string{"[**strong** link](u)"}, 0, "strong link (u)", []at{{"strong", mdStyle(stMdLink, true, false, false)}, {"link", stMdLink}}, nil},
		{"escapes", []string{`\*not em\* \_nor\_ \\ \# \[x\](y)`}, 0, `*not em* _nor_ \ # [x](y)`, []at{{"not em", stText}}, nil},
		{"intraword underscores", []string{"snake_case_name and file_name.go"}, 0, "snake_case_name and file_name.go", nil, nil},
		{"no underscore opening in a word", []string{"snake_case_ and a_b_ c"}, 0, "snake_case_ and a_b_ c", nil, nil},
		{"unclosed", []string{"**unclosed bold, `open code, [label](open and ~~strike"}, 0, "**unclosed bold, `open code, [label](open and ~~strike", nil, nil},
		{"lone markers", []string{"2 * 3 * 4, a*b and a ~ b"}, 0, "2 * 3 * 4, a*b and a ~ b", nil, nil},
		{"no closer after a space", []string{"*a * b* and **c ** d**"}, 0, "a * b and c ** d", []at{{"a", italic}, {"b", italic}, {"c", bold}, {"d**", bold}}, nil},
		{"a file's byte order mark", []string{"\uFEFF# Title"}, 0, "Title", []at{{"Title", stMdHeading}}, []string{"\uFEFF", "#"}},
		{"a byte order mark before text", []string{"\uFEFF- **item**"}, 0, "• item", []at{{"-", stMdBullet}, {"item", bold}}, []string{"\uFEFF"}},
		{"heading", []string{"## Title with `code` ##"}, 0, "Title with code", []at{{"Title", stMdHeading}, {"code", stMdCode}}, []string{"##"}},
		{"no heading", []string{"#hashtag ####### seven"}, 0, "#hashtag ####### seven", nil, nil},
		{"quote", []string{"> quoted **bold** text"}, 0, "│ quoted bold text", []at{{">", stMdMark}, {"quoted", stMdEm}, {"bold", mdStyle(stMdEm, true, false, false)}}, nil},
		{"nested quote", []string{"> > deeper"}, 0, "│ │ deeper", []at{{"deeper", stMdEm}}, nil},
		{"a list in a quote", []string{"> - item"}, 0, "│ • item", []at{{"-", stMdBullet}, {"item", stMdEm}}, nil},
		{"a heading in a quote", []string{"> # Title"}, 0, "│ Title", []at{{"Title", stMdHeading}}, []string{"#"}},
		{"bullet", []string{"- first"}, 0, "• first", []at{{"-", stMdBullet}, {"first", stText}}, nil},
		{"nested bullet", []string{"  - nested"}, 0, "  ◦ nested", []at{{"-", stMdBullet}}, nil},
		{"deeper bullet", []string{"    - deeper"}, 0, "    ▪ deeper", nil, nil},
		{"deepest bullet", []string{"      * deepest"}, 0, "      ▪ deepest", nil, nil},
		{"plus bullet", []string{"+ plus"}, 0, "• plus", nil, nil},
		{"numbered", []string{"12. twelfth **bold**"}, 0, "12. twelfth bold", []at{{"12.", stMdBullet}, {"twelfth", stText}, {"bold", bold}}, nil},
		{"numbered, a parenthesis", []string{"3) third"}, 0, "3) third", []at{{"3)", stMdBullet}}, nil},
		{"a task list's box as written", []string{"- [ ] todo"}, 0, "• [ ] todo", nil, nil},
		{"rule", []string{"---"}, 0, strings.Repeat("─", 40), []at{{"-", stMdMark}}, nil},
		{"rule of stars", []string{"* * *"}, 0, strings.Repeat("─", 40), nil, nil},
		{"rule of underscores", []string{"___"}, 0, strings.Repeat("─", 40), nil, nil},
		{"fence", []string{"```go", "x := 1", "```"}, 0, "go", []at{{"go", stMdLang}}, []string{"```"}},
		{"fence with an info string", []string{"~~~ python extra", "pass", "~~~"}, 0, "python extra", []at{{"python", stMdLang}, {"extra", stGrey}}, []string{"~~~"}},
		{"closing fence", []string{"```go", "x := 1", "```"}, 2, "", nil, []string{"```"}},
		{"code literal", []string{"```", "**not bold** `x` # h"}, 1, "**not bold** `x` # h", []at{{"**not", stMdCode}}, nil},
		{"code in an unclosed fence", []string{"```", "# not a heading"}, 1, "# not a heading", []at{{"#", stMdCode}}, nil},
		{"table header", table, 0, "│ Name  │ Value │", []at{{"Name", stBold}, {"|", stMdMark}}, nil},
		{"table rule", table, 1, "├───────┼───────┤", nil, nil},
		{"table row", table, 2, "│ alpha │ 42    │", []at{{"alpha", stText}, {"42", bold}}, []string{"**"}},
		{"table row, pipes escaped and in code", table, 3, "│ b | c │ x|y   │", []at{{"x|y", stMdCode}}, nil},
		{"no table without its rule", []string{"| not | a table |", "plain"}, 0, "| not | a table |", []at{{"not", stText}}, nil},
		{"mixed", []string{"Text with **bold**, *em*, `code`, ~~strike~~ and [a link](u)."}, 0, "Text with bold, em, code, strike and a link (u).",
			[]at{{"Text", stText}, {"bold", bold}, {"em", italic}, {"code", stMdCode}, {"strike", strike}, {"a link", stMdLink}}, nil},
		{"wide characters", []string{"**日本語** text"}, 0, "日本語 text", []at{{"日本語", bold}}, nil},
		{"control characters drawn", []string{"a\x1b[31m**b**\x07"}, 0, "a^[[31mb^G", []at{{"b", bold}}, nil},
		{"a tab", []string{"**a**\tb"}, 0, "a    b", nil, nil},
		{"empty", []string{""}, 0, "", nil, nil},
	} {
		t.Run(c.name, func(t *testing.T) {
			src := c.lines[c.j]
			sh := mdShow(src, mdLines(c.lines)[c.j], 40, stText)
			if got := mdShownText(sh); got != c.want {
				t.Errorf("%q shown %q, want %q", src, got, c.want)
			}
			n := 0
			for _, x := range sh.cells {
				n += x.w
			}
			if n != textWidth(c.want) {
				t.Errorf("%q: %d cells wide, want %d", src, n, textWidth(c.want))
			}
			for _, a := range c.at {
				i := strings.Index(src, a.text)
				if i < 0 {
					t.Fatalf("no %q in %q", a.text, src)
				}
				if got := mdByteStyle(sh, i); got != a.st {
					t.Errorf("%q in %q styled %v, want %v", a.text, src, got, a.st)
				}
			}
			for _, h := range c.hidden {
				for i := 0; ; {
					j := strings.Index(src[i:], h)
					if j < 0 {
						break
					}
					for b := i + j; b < i+j+len(h); b++ {
						if mdCovered(sh, b) || mdByteStyle(sh, b) != stNone {
							t.Errorf("%q: byte %d of %q shown", src, b, h)
						}
					}
					i += j + len(h)
				}
			}
		})
	}
}

// TestMarkdownCells is criteria 71.1 and 71.2's ground, on 4,000 random
// lines of Markdown's pieces, controls and broken UTF-8: every cell stands
// for bytes of its line, in order and apart, shows them as they are unless
// it is a glyph, a tab or a control's mark, and draws no control
// character; every byte is styled; the wrap keeps every cell but the spaces
// at its rows' edges, each row as wide as its width allows; a copy's
// widening takes in hidden bytes only.
func TestMarkdownCells(t *testing.T) {
	parts := []string{"a", "word", " ", "  ", "**", "*", "_", "__", "~~", "~", "`", "``", "[", "]", "(", ")", "![", "\\", "#", "## ",
		"> ", "- ", "  - ", "1. ", "|", "|---|", "---", "```", "~~~", "\t", "\x1b", "\x07", "\x7f", "\xc3", "\u0085", "é", "界", "😀", "https://x.y", "\n"}
	r := rand.New(rand.NewPCG(71, 2026))
	for n := range 4000 {
		var b strings.Builder
		for range r.IntN(30) {
			b.WriteString(parts[r.IntN(len(parts))])
		}
		lines := strings.Split(b.String(), "\n")
		ks := mdLines(lines)
		w := 1 + n%50
		for j, l := range lines {
			sh := mdShow(l, ks[j], w, stText)
			end := 0
			for _, c := range sh.cells {
				if c.off < end || c.n <= 0 || c.off+c.n > len(l) {
					t.Fatalf("%q: cell %+v after byte %d", l, c, end)
				}
				if !c.sub && !c.ctrl && l[c.off] != '\t' && c.text != l[c.off:c.off+c.n] {
					t.Fatalf("%q: cell %+v not its bytes", l, c)
				}
				if strings.ContainsFunc(c.text, func(r rune) bool { return r < 0x20 || r == 0x7f || r >= 0x80 && r < 0xa0 }) {
					t.Fatalf("%q: cell %q draws a control character", l, c.text)
				}
				end = c.off + c.n
			}
			if len(l) > 0 && (len(sh.runs) == 0 || sh.runs[len(sh.runs)-1].to < len(l)) {
				t.Fatalf("%q: runs %v leave bytes unstyled", l, sh.runs)
			}
			// The wrap: rows in order, every cell in one but the spaces at
			// their edges, none wider than its width but a lone wide cell.
			rest := w - sh.hang.width()
			rows := wrapCells(l, sh.cells, w, rest)
			kept := 0
			for k, rr := range rows {
				cs := rowCells(&sh, rr)
				width := 0
				for _, c := range cs {
					width += c.w
				}
				room := max(w, 1)
				if k > 0 {
					room = max(rest, 1)
				}
				if width > room && len(cs) > 1 {
					t.Fatalf("%q at %d: row %d %d cells wide", l, w, k, width)
				}
				if k > 0 && rr.from < rows[k-1].to {
					t.Fatalf("%q at %d: rows %v out of order", l, w, rows)
				}
				kept += len(cs)
			}
			spaces := 0
			for _, c := range sh.cells {
				if c.text == " " {
					spaces++
				}
			}
			if kept > len(sh.cells) || kept < len(sh.cells)-spaces {
				t.Fatalf("%q at %d: %d of %d cells in rows", l, w, kept, len(sh.cells))
			}
			// A copy widens over hidden bytes only.
			o0 := r.IntN(len(l) + 1)
			o1 := min(o0+r.IntN(6), len(l))
			w0, w1 := widenLine(l, ks[j], o0, o1)
			if w0 > o0 || w1 < o1 || w0 < 0 || w1 > len(l) {
				t.Fatalf("%q: [%d, %d) widened to [%d, %d)", l, o0, o1, w0, w1)
			}
			for i := w0; i < o0; i++ {
				if mdCovered(sh, i) {
					t.Fatalf("%q: [%d, %d) widened to [%d, %d) over shown byte %d", l, o0, o1, w0, w1, i)
				}
			}
			for i := o1; i < w1; i++ {
				if mdCovered(sh, i) {
					t.Fatalf("%q: [%d, %d) widened to [%d, %d) over shown byte %d", l, o0, o1, w0, w1, i)
				}
			}
		}
	}
}

// TestMarkdownWiden is design D3: a selection copies the source of what it
// covers, a construct's markers with it when it takes in all the construct
// shows, a word of it as the word; over lines, from the first line's place
// to the last's.
func TestMarkdownWiden(t *testing.T) {
	table := []string{"| a | **b** |", "|---|---|"}
	for _, c := range []struct {
		lines     []string
		j         int
		sel, want string // the selection, by its source, and its copy
	}{
		{[]string{"**Goal:** finish"}, 0, "Goal:", "**Goal:**"},
		{[]string{"**Goal:** finish"}, 0, "Goal", "Goal"},
		{[]string{"**Goal:** finish"}, 0, "Goal:** finish", "**Goal:** finish"},
		{[]string{"**Goal:** finish"}, 0, "finish", "finish"},
		{[]string{"a **bold *both* bold** b"}, 0, "both", "*both*"},
		{[]string{"a **bold *both* bold** b"}, 0, "bold *both* bold", "**bold *both* bold**"},
		{[]string{"**`code`**"}, 0, "code", "**`code`**"},
		{[]string{"see [the docs](https://x) now"}, 0, "the docs", "the docs"},
		{[]string{"see [the docs](https://x) now"}, 0, "the docs](https://x)", "[the docs](https://x)"},
		{[]string{`a \*b\* c`}, 0, `*b\*`, `\*b\*`},
		{[]string{"## Title here"}, 0, "Title here", "## Title here"},
		{[]string{"## Title here"}, 0, "Title", "Title"},
		{[]string{"```go", "x"}, 0, "go", "```go"},
		{[]string{"- **item**"}, 0, "item", "**item**"},
		{table, 0, "b", "**b**"},
		{table, 0, "a", "a"},
	} {
		src := c.lines[c.j]
		i := strings.Index(src, c.sel)
		if i < 0 {
			t.Fatalf("no %q in %q", c.sel, src)
		}
		o0, o1 := mdWiden(c.lines, mdLines(c.lines), c.j, i, c.j, i+len(c.sel))
		if got := src[o0:o1]; got != c.want {
			t.Errorf("%q in %q copies %q, want %q", c.sel, src, got, c.want)
		}
	}
	// Over lines: from the first line's place to its end, and from the last
	// line's start to its place.
	ls := []string{"**Goal:** x", "middle", "- **Next** y"}
	o0, o1 := mdWiden(ls, mdLines(ls), 0, 2, 2, 8)
	if got := between(ls, 0, o0, 2, o1); got != "**Goal:** x\nmiddle\n- **Next**" {
		t.Errorf("over lines: %q", got)
	}
	// No Markdown, no widening.
	if o0, o1 := mdWiden(ls, nil, 0, 2, 0, 7); o0 != 2 || o1 != 7 {
		t.Errorf("without Markdown: [%d, %d)", o0, o1)
	}
}

// TestMarkdownWrap is design D2: the wrap counts the cells shown, its rows
// keeping their bytes; a list item's and a quote's later rows hang under
// their text, a rule is as wide as its row.
func TestMarkdownWrap(t *testing.T) {
	src := "**" + strings.Repeat("a", 20) + "**"
	if rs := (*WrapCache)(nil).mdRanges(src, mdLines([]string{src})[0], 20, stText); len(rs) != 1 {
		t.Errorf("a strong word of 20 cells in 20: %d rows, want 1", len(rs))
	}
	if rs := wrapRanges(src, 20, 20); len(rs) != 2 {
		t.Errorf("its source in 20: %d rows, want 2", len(rs))
	}
	words := strings.Repeat("word ", 9) + "end"
	for _, c := range []struct {
		src, first, hang string // the first row's start, the later rows' hang
	}{
		{"- " + words, "• ", "  "},
		{"  - " + words, "  ◦ ", "    "},
		{"10. " + words, "10. ", "    "},
		{"> " + words, "│ ", "│ "},
		{"> > " + words, "│ │ ", "│ │ "},
		{"> - " + words, "│ • ", "│   "},
		{"plain " + words, "plain ", ""},
	} {
		rows := mdWrapped(c.src, mdLines([]string{c.src})[0], 20, stText, "")
		if len(rows) < 3 {
			t.Fatalf("%q: %d rows", c.src, len(rows))
		}
		var got []string
		for k, l := range rows {
			s := strings.TrimSuffix(texts([]line{l}), "\n")
			if l.width() > 20 {
				t.Errorf("%q: row %d %q %d cells wide", c.src, k, s, l.width())
			}
			lead := c.hang
			if k == 0 {
				lead = c.first
			}
			text, ok := strings.CutPrefix(s, lead)
			if !ok || strings.HasPrefix(text, " ") {
				t.Errorf("%q: row %d %q, want %q then its text", c.src, k, s, lead)
			}
			got = append(got, strings.Fields(text)...)
		}
		if !slices.Equal(got, strings.Fields(words)) {
			t.Errorf("%q: rows hold %q", c.src, got)
		}
	}
	// A rule: one row, as wide as asked.
	if rows := mdWrapped("---", mdLines([]string{"---"})[0], 33, stText, ""); len(rows) != 1 || rows[0].width() != 33 {
		t.Errorf("a rule at 33: %d rows, %d cells", len(rows), rows[0].width())
	}
}

// mdZoneB is each frame row's zone B content: the text of its spans keyed to
// panel B.
func mdZoneB(sc *Screen) []string {
	var out []string
	for _, l := range sc.lines {
		var b strings.Builder
		for _, x := range l {
			if x.ref.panel == 'B' {
				b.WriteString(x.text)
			}
		}
		out = append(out, b.String())
	}
	return out
}

// mdSpanAt is where the frame draws panel B's span text, and its style.
func mdSpanAt(sc *Screen, text string) (x, y int, st *lipgloss.Style, ok bool) {
	for y, l := range sc.lines {
		x := 0
		for _, s := range l {
			if s.ref.panel == 'B' && s.text == text {
				return x, y, s.st, true
			}
			x += textWidth(s.text)
		}
	}
	return 0, 0, nil, false
}

// ownerAnswer is the answer of the owner's report (amendment 74), a summary
// after a resume.
var ownerAnswer = strings.Join([]string{
	"**Goal:** finish plan 70's card and close it.",
	"",
	"**Decisions:**",
	"- keep the blank row under the buttons",
	"- `m` toggles a `.md` file's source",
	"",
	"**Completed:** the card, its tests and the docs.",
	"",
	"**Open:** none.",
	"",
	"**Recommended next step:** run the review.",
}, "\n")

var ownerLabels = []string{"Goal:", "Decisions:", "Completed:", "Open:", "Recommended next step:"}

// ownerSession is a resumed session of provider whose CLI answered the
// summary request with ownerAnswer, streaming or completed, built from
// events through the reducer.
func ownerSession(provider string, streaming bool) (*state.State, int) {
	b := newConvBench(provider, "a-model")
	b.s.Session.Resumed = "5f2c0a9e-0000-4000-8000-000000000001"
	b.ev(event.SessionStarted, event.Started{SessionID: b.s.Session.Resumed})
	b.prompt("Summarize the work we have done in this conversation so far.")
	for rest := ownerAnswer; rest != ""; {
		n := min(7, len(rest)) // markers split across deltas
		b.ev(event.MessageDelta, event.Delta{MessageID: "m1", Block: "text", Text: rest[:n]})
		rest = rest[n:]
	}
	if !streaming {
		b.ev(event.MessageCompleted, event.Completed{MessageID: "m1", Text: ownerAnswer})
		b.ev(event.TurnCompleted, event.TurnEnd{Outcome: "completed", Subtype: "success"})
	}
	for i, e := range b.s.Conversation {
		if e.Role == "assistant" {
			return b.s, i
		}
	}
	return b.s, -1
}

// TestMarkdownZoneB is criteria 71.2 and 71.6 for the owner's answer: on
// each CLI, streaming and completed, at 80x24, 120x40 and 200x60, its labels
// bold without their markers, its list's bullets; every row within the
// frame; a cell's click its source byte; the copy its source, a label's its
// markers with it.
func TestMarkdownZoneB(t *testing.T) {
	bold := mdStyle(stText, true, false, false)
	for _, provider := range []string{"claude-code", "codex"} {
		for _, streaming := range []bool{false, true} {
			s, ai := ownerSession(provider, streaming)
			if ai < 0 || s.Conversation[ai].Text != ownerAnswer || s.Conversation[ai].Streaming != streaming {
				t.Fatalf("%s, streaming %v: no answer in %+v", provider, streaming, s.Conversation)
			}
			for _, size := range [][2]int{{80, 24}, {120, 40}, {200, 60}} {
				name := fmt.Sprintf("%s streaming %v %dx%d", provider, streaming, size[0], size[1])
				ui := UI{Width: size[0], Height: size[1], Focus: "B"}
				sc := Draw(s, ui)
				if len(sc.lines) != size[1] {
					t.Errorf("%s: %d rows", name, len(sc.lines))
				}
				for y, l := range sc.lines {
					if l.width() > size[0] {
						t.Errorf("%s: row %d %d cells wide", name, y, l.width())
					}
				}
				b := strings.Join(mdZoneB(sc), "\n")
				if strings.Contains(b, "**") || strings.Contains(b, "`") {
					t.Errorf("%s: a marker drawn:\n%s", name, b)
				}
				seen := 0
				for _, label := range ownerLabels {
					if _, _, st, ok := mdSpanAt(sc, label); ok {
						seen++
						if st != bold {
							t.Errorf("%s: %q styled %v, want bold", name, label, st)
						}
					}
				}
				// Zone B follows the conversation's end: the last label is in
				// view, all of them from 40 rows.
				if _, _, _, ok := mdSpanAt(sc, "Recommended next step:"); !ok || size[1] >= 40 && seen != len(ownerLabels) {
					t.Errorf("%s: %d labels in view:\n%s", name, seen, b)
				}
				if size[1] >= 40 && (!strings.Contains(b, "• keep the blank row under the buttons") || !strings.Contains(b, "• m toggles a .md file's source")) {
					t.Errorf("%s: the list not drawn with its bullets:\n%s", name, b)
				}
				// A click on a label's first cell is its source byte, after its
				// hidden markers.
				if x, y, _, ok := mdSpanAt(sc, "Recommended next step:"); ok {
					p, _, hit := sc.HitIn('B', x, y, false)
					if !hit || p.Item != ai || p.Line != 10 || p.Off != 2 {
						t.Errorf("%s: the label's first cell hits %+v", name, p)
					}
				}
				// The copy: the whole answer its source; a label with its markers;
				// a word of it as the word.
				ls := entryLines(s, s.Conversation[ai])
				all := Selection{From: Pos{Panel: 'B', Item: ai}, To: Pos{Panel: 'B', Item: ai, Line: len(ls) - 1, Off: len(ls[len(ls)-1])}}
				if got := sc.Text(s, all); got != ownerAnswer {
					t.Errorf("%s: the answer copies as\n%q", name, got)
				}
				label := Selection{From: Pos{Panel: 'B', Item: ai, Line: 0, Off: 2}, To: Pos{Panel: 'B', Item: ai, Line: 0, Off: 7}}
				word := Selection{From: Pos{Panel: 'B', Item: ai, Line: 0, Off: 2}, To: Pos{Panel: 'B', Item: ai, Line: 0, Off: 6}}
				if got, w := sc.Text(s, label), sc.Text(s, word); got != "**Goal:**" || w != "Goal" {
					t.Errorf("%s: the label copies as %q, its word as %q", name, got, w)
				}
			}
		}
	}
}

// answerRows are the rows of entry i among zone B's rows w cells wide, as
// the text of their spans keyed to panel B: the card, the hangs and the
// streaming cursor aside.
func answerRows(s *state.State, i, w int) []string {
	var out []string
	for _, l := range bRows(s, UI{}, w) {
		mine := false
		for _, x := range l {
			if x.ref.panel == 'B' && x.ref.item == i {
				mine = true
			}
		}
		if mine {
			var b strings.Builder
			for _, x := range l {
				if x.ref.panel == 'B' {
					b.WriteString(x.text)
				}
			}
			out = append(out, strings.TrimRight(b.String(), " "))
		}
	}
	return out
}

// TestMarkdownStreaming is criterion 71.6 (design D6): the owner's answer
// streamed seven bytes a delta, each marker split, on each CLI: at each
// step its rows are those of the same text completed, the cursor aside, and
// every row is within zone B at 80x24; a marker shows as written until its
// closer comes.
func TestMarkdownStreaming(t *testing.T) {
	for _, provider := range []string{"claude-code", "codex"} {
		b := newConvBench(provider, "a-model")
		b.prompt("Summarize.")
		for n, rest := 0, ownerAnswer; rest != ""; n++ {
			k := min(7, len(rest))
			b.ev(event.MessageDelta, event.Delta{MessageID: "m1", Block: "text", Text: rest[:k]})
			rest = rest[k:]
			ai := len(b.s.Conversation) - 1
			e := b.s.Conversation[ai]
			if e.Role != "assistant" || !e.Streaming {
				t.Fatalf("%s: step %d: entry %+v", provider, n, e)
			}
			done := *b.s
			done.Conversation = slices.Clone(b.s.Conversation)
			done.Conversation[ai].Streaming = false
			if a, d := answerRows(b.s, ai, 76), answerRows(&done, ai, 76); !slices.Equal(a, d) {
				t.Fatalf("%s: step %d: streaming rows\n%s\nwant\n%s", provider, n, strings.Join(a, "\n"), strings.Join(d, "\n"))
			}
			sc := Draw(b.s, UI{Width: 80, Height: 24, Focus: "B"})
			for y, l := range sc.lines {
				if l.width() > 80 {
					t.Fatalf("%s: step %d: row %d %d cells wide", provider, n, y, l.width())
				}
			}
			// "**Goal:" before its closer: the opener as written.
			if e.Text == ownerAnswer[:7] && !strings.Contains(strings.Join(answerRows(b.s, ai, 76), "\n"), "**Goal:") {
				t.Errorf("%s: the unclosed opener not drawn as written", provider)
			}
		}
	}
}

// TestMarkdownRecordings is criteria 71.2 and 71.3 on the CLIs' own
// recordings, each through its mapper: in zone B, Claude Code's streamed
// "**59**" (P22o) bold and Codex's answer linking the file it wrote, its
// destination in angle brackets, and naming its content in a code span
// (P12x); a Codex subagent's "**0 files**" (P7c) bold in the inspection's
// Messages tab and plain in zone C's timeline, without markers.
func TestMarkdownRecordings(t *testing.T) {
	bold := mdStyle(stText, true, false, false)
	for _, c := range []struct {
		probe, marked string
		shown         []string // spans drawn, with their styles
		st            []*lipgloss.Style
		row           string // the row's text, in part
	}{
		{"p22o-claude-opus-thinking.jsonl", "**59**", []string{"59"}, []*lipgloss.Style{bold}, "59: each remainder is one less than its divisor"},
		{"p12x-codex-auto-review.jsonl", "Created [hello.txt](<", []string{"hello.txt", "hello"}, []*lipgloss.Style{stMdLink, stMdCode},
			"Created hello.txt (<HOME>/AppData/Local/Temp/acc-p12/x/hello.txt) containing hello."},
	} {
		s := replayProbe(t, c.probe)
		ai := -1
		for i, e := range s.Conversation {
			if e.Role == "assistant" && strings.Contains(e.Text, c.marked) {
				ai = i
			}
		}
		if ai < 0 {
			t.Fatalf("%s: no answer holding %q", c.probe, c.marked)
		}
		// Zone B's rows 76 and 116 cells wide, as at 80 and 120 columns.
		for _, w := range []int{76, 116} {
			rows := answerRows(s, ai, w)
			joined := strings.Join(strings.Fields(strings.Join(rows, " ")), " ")
			if strings.Contains(joined, c.marked) || !strings.Contains(joined, c.row) {
				t.Errorf("%s at %d: the answer's rows\n%s", c.probe, w, strings.Join(rows, "\n"))
			}
			for k, text := range c.shown {
				var st *lipgloss.Style
				for _, l := range bRows(s, UI{}, w) {
					for _, x := range l {
						if st == nil && x.ref.panel == 'B' && x.ref.item == ai && x.text == text {
							st = x.st
						}
					}
				}
				if st != c.st[k] {
					t.Errorf("%s at %d: %q styled %v, want %v", c.probe, w, text, st, c.st[k])
				}
			}
		}
	}

	s := replayProbe(t, "p7c-codex-spawn-model.jsonl")
	var key string
	for k, a := range s.Agents {
		for _, e := range a.Messages {
			if strings.Contains(e.Text, "**0 files**") {
				key = k
			}
		}
	}
	if key == "" {
		t.Fatal("P7c: no agent message holding **0 files**")
	}
	sc := Draw(s, UI{Width: 120, Height: 40, Inspect: key})
	if text := texts(sc.lines); strings.Contains(text, "**0 files**") || !strings.Contains(text, "There are 0 files in the current working directory.") {
		t.Errorf("P7c's inspection:\n%s", text)
	}
	if st := panelStyle(sc, 'I', "0 files"); st != bold {
		t.Errorf("P7c's inspection: \"0 files\" styled %v", st)
	}
	_, rows := timeline(s, UI{Selected: key}, 100, 40, &Screen{})
	if text := texts(rows); strings.Contains(text, "**") || !strings.Contains(text, "There are 0 files in the current working directory.") {
		t.Errorf("P7c's timeline:\n%s", text)
	}
}

// TestMarkdownRaw is criterion 71.5: what is not AI text keeps its markers
// as written: the owner's message, the draft, a notice, a dashboard note, a
// tool's target and its error, an approval's preview, a diff's line of a
// Markdown file.
func TestMarkdownRaw(t *testing.T) {
	b := newConvBench("claude-code", "a-model")
	b.prompt("make **this** bold and `that` code")
	b.ev(event.ToolStarted, event.Tool{ToolID: "b1", Name: "Bash", Input: []byte(`{"command":"echo **x**"}`)})
	b.ev(event.ToolCompleted, event.Tool{ToolID: "b1", Name: "Bash", IsError: true, Output: "**bad** output"})
	b.s.Conversation = append(b.s.Conversation,
		state.Entry{Role: "notice", Text: "a **notice**"},
		state.Entry{Role: "dashboard", Text: "a **note**"})
	draft := EditorView{Lines: []string{"a **draft** with `ticks` and # a hash"}, Total: 1}
	sc := Draw(b.s, UI{Width: 200, Height: 60, Focus: "B", Editor: draft})
	frame := texts(sc.lines)
	for _, want := range []string{"make **this** bold and `that` code", "a **draft** with `ticks` and # a hash", "echo **x**", "**bad** output", "a **notice**", "a **note**"} {
		if !strings.Contains(frame, want) {
			t.Errorf("%q not drawn as written:\n%s", want, frame)
		}
	}
	// A Write's preview on its approval card, as written.
	in := []byte(`{"file_path":"notes.md","content":"# Title\n**bold** and *em*\n"}`)
	ap := &state.Approval{ID: "r1", Tool: "Write", Input: in, Status: state.Pending}
	rows, _ := viewerRows(&state.State{Approvals: map[string]*state.Approval{"r1": ap}}, ap, 80)
	if all := strings.Join(rows, "\n"); !strings.Contains(all, "# Title") || !strings.Contains(all, "**bold** and *em*") {
		t.Errorf("the approval's preview:\n%s", all)
	}
	// A diff of a Markdown file: its lines as written.
	for _, ln := range []DiffLine{{'+', 0, 1, "**bold** and `code`"}, {'-', 1, 0, "# Title"}, {' ', 2, 2, "- item"}} {
		if got := plainOf(diffText(ln, 0, 40)); !strings.Contains(got, ln.Text) {
			t.Errorf("the diff line %q drawn %q", ln.Text, got)
		}
	}
}

// TestBlockComment is plan 5, item 12, criteria 1 and 2: a block comment is
// a comment on every line to its */, and only there.
func TestBlockComment(t *testing.T) {
	lines := []string{
		"```c",
		"int a; /* opens here",
		"   still \"in\" the comment // too",
		"   ends */ return 42;",
		`x = "/* in a string" + 1;`,
		"y = 2; // line /* not opened",
		"z = 3; /* one line */ w = 4;",
		"int b; /* left open",
		"```",
		"```c",
		"int c = 5;",
		"```",
		"```python",
		"x = 1 /* not a comment",
		"y = 2",
		"```",
		"```",
		"a /* b",
		"c */ d",
		"```",
	}
	ks := mdLines(lines)
	// style is the style of the first occurrence of text on line j.
	style := func(j int, text string) *lipgloss.Style {
		at := strings.Index(lines[j], text)
		if at < 0 {
			t.Fatalf("no %q on line %d", text, j)
		}
		return mdByteStyle(mdShow(lines[j], ks[j], 80, stText), at)
	}
	for _, c := range []struct {
		line int
		text string
		st   *lipgloss.Style
	}{
		{1, "int", stMdKeyword},
		{1, "/* opens here", stMdComment},
		{2, "still", stMdComment}, // the comment's next lines,
		{2, `"in"`, stMdComment},  // a string in it,
		{2, "// too", stMdComment},
		{3, "ends */", stMdComment}, // its end,
		{3, "return", stMdKeyword},  // and the code after it
		{3, "42", stMdNumber},
		{4, `"/* in a string"`, stMdString},
		{5, "y", stMdCode},
		{6, "z", stMdCode}, // after a line comment holding /*
		{6, "/* one line */", stMdComment},
		{6, "w", stMdCode},
		{6, "4", stMdNumber},
		{7, "/* left open", stMdComment},
		{10, "int", stMdKeyword}, // a new fence starts as code
		{10, "5", stMdNumber},
		{13, "/", stMdCode}, // not a comment in Python
		{13, "not", stMdKeyword},
		{14, "2", stMdNumber},
		{17, "a /* b", stMdCode}, // a fence without a language
		{18, "c */ d", stMdCode},
	} {
		if got := style(c.line, c.text); got != c.st {
			t.Errorf("line %d %q styled %v, want %v", c.line, c.text, got, c.st)
		}
	}
	for j, want := range map[int]bool{1: false, 2: true, 3: true, 4: false, 5: false, 6: false, 7: false, 8: false, 10: false, 14: false, 18: false} {
		if (ks[j].open == "*/") != want {
			t.Errorf("line %d starts in a comment: %v, want %v", j, ks[j].open == "*/", want)
		}
	}
}

// TestMultiLineString is plan 6, item 3, criteria 1 to 3: a backtick string in
// Go, JavaScript and TypeScript, and a triple-quoted one in Python, run over
// the lines to their close; elsewhere a string ends with its line, one left
// open ends with its fence, a /* in a string is string and a backtick in a
// block comment is comment; the fences' markers hidden (plan 71).
func TestMultiLineString(t *testing.T) {
	lines := []string{
		"```go",                        // 0
		"q := `select *",               // 1
		"  from \"t\" // not one",      // 2
		"  /* nor this` + x",           // 3
		"```",                          // 4
		"```tsx",                       // 5
		"const a = `x ${y}",            // 6
		"still \\` inside`; let n = 3", // 7
		"```",                          // 8
		"```python",                    // 9
		`s = """doc`,                   // 10
		"  more # text",                // 11
		`""" ; k = 4`,                  // 12
		`t = '''open`,                  // 13
		"```",                          // 14
		"```python",                    // 15
		"n = 5",                        // 16
		"```",                          // 17
		"```sh",                        // 18
		"echo `date",                   // 19
		"x=1",                          // 20
		"```",                          // 21
		"```c",                         // 22
		"/* a ` b",                     // 23
		"c` */ d = 6;",                 // 24
		`s = "/* in a string`,          // 25
		"y = 7;",                       // 26
		"```",                          // 27
	}
	ks := mdLines(lines)
	style := func(j int, text string) *lipgloss.Style {
		at := strings.Index(lines[j], text)
		if at < 0 {
			t.Fatalf("no %q on line %d", text, j)
		}
		return mdByteStyle(mdShow(lines[j], ks[j], 80, stText), at)
	}
	for _, c := range []struct {
		line int
		text string
		st   *lipgloss.Style
	}{
		{1, "select", stMdString}, {2, "from", stMdString}, {2, "// not one", stMdString}, {3, "/* nor this`", stMdString},
		{3, "x", stMdCode}, {4, "```", stNone},
		{6, "x ${y}", stMdString}, {7, "still", stMdString}, {7, "inside", stMdString}, {7, "let", stMdKeyword}, {7, "3", stMdNumber},
		{10, "doc", stMdString}, {11, "more # text", stMdString}, {12, `"""`, stMdString}, {12, "k", stMdCode}, {12, "4", stMdNumber},
		{13, "open", stMdString},
		{16, "5", stMdNumber},                         // the next fence starts as code
		{19, "date", stMdString}, {20, "x", stMdCode}, // a shell backtick ends with its line
		{23, "a", stMdComment}, {24, "c", stMdComment}, {24, "d", stMdCode}, {24, "6", stMdNumber},
		{25, "/* in", stMdString}, {26, "y", stMdCode},
	} {
		if got := style(c.line, c.text); got != c.st {
			t.Errorf("line %d %q styled %v, want %v", c.line, c.text, got, c.st)
		}
	}
}

// TestMarkdownExcerpts is criterion 71.3's other surfaces: an agent's
// summary on its row; zone C's timeline line of an AI message, a notice's
// as written, a long message read only as far as its two rows need; Codex's
// summary in a phase's full view, the dashboard's own lines as they were.
func TestMarkdownExcerpts(t *testing.T) {
	a := &state.Agent{Key: "a1", Label: "worker", Status: state.AgentCompleted, Summary: "**Done**: `ok` and [a link](u)\nmore"}
	if row := texts([]line{agentRow(a, UI{}, 120, agentCols{label: 16})}); !strings.Contains(row, "completed · Done: ok and a link (u)") || strings.Contains(row, "**") {
		t.Errorf("the agent's row: %q", row)
	}
	at := thinkAt(1000)
	for _, c := range []struct {
		e    state.Entry
		want string
	}{
		{state.Entry{Role: "assistant", Text: "**Goal:** done\n- `a` and *b*", At: at}, "│ Goal: done • a and b"},
		{state.Entry{Role: "notice", Text: "a **notice** with `ticks`", At: at}, "│ a **notice** with `ticks`"},
		{state.Entry{Role: "assistant", Text: "**Start** " + strings.Repeat("filler words ", 2000) + "**end**", At: at}, "│ Start filler words"},
	} {
		rows := strings.Split(strings.TrimSuffix(texts(timelineEntry(c.e, 80)), "\n"), "\n")
		if !strings.Contains(rows[0], c.want) {
			t.Errorf("the timeline's line for %.30q: %q, want %q", c.e.Text, rows, c.want)
		}
		if strings.HasPrefix(c.e.Text, "**Start") && (len(rows) != 2 || !strings.HasSuffix(strings.TrimRight(rows[1], " "), "…")) {
			t.Errorf("a long message's rows: %q", rows)
		}
	}
	// The excerpt reads its lines until they hold the room's bytes, and one
	// more; without a room, all of them.
	many := strings.Repeat("a line\n", 5000)
	if got := strings.Count(mdText(many, 30), "\n"); got != 5 {
		t.Errorf("the excerpt of 30 bytes holds %d line breaks, want 5", got)
	}
	if got := strings.Count(mdText(many, 0), "\n"); got != 5000 {
		t.Errorf("the whole text holds %d line breaks", got)
	}
	// Codex's summary in a phase's full view.
	th := state.Thought{Start: thinkAt(1000), ItemStart: thinkAt(2500), End: thinkAt(4000), Inferred: true, Confirmed: true,
		Summary: "**Planning the change**\n\nI will `read` the file."}
	s := thinkBench(th)
	_, body := thoughtView(s.Conversation[1], span{}, 80)
	text := texts(body)
	for _, want := range []string{"Codex's summary, its own text: Planning the change", "I will read the file.", "the model step from"} {
		if !strings.Contains(text, want) {
			t.Errorf("the full view: no %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, "**") || strings.Contains(text, "`") {
		t.Errorf("the full view draws a marker:\n%s", text)
	}
}

// TestMarkdownCache is design D2's memo: the lines shown, kept from frame to
// frame with the wraps, draw the frames drawn without them, as an answer
// streams, the size changes and the scroll moves, in a Markdown file's
// preview and in the inspection; a line no frame draws is dropped after the
// next.
func TestMarkdownCache(t *testing.T) {
	b := newConvBench("codex", "a-model")
	b.prompt("Summarize.")
	long := strings.Repeat("a **long** line with `code` ", 400)
	text := ownerAnswer + "\n" + long + "\n| a | b |\n|---|---|\n| c | d |\n---"
	wc := NewWrapCache()
	check := func(step string, ui UI) {
		t.Helper()
		cached := ui
		cached.Wrap = wc
		if got, want := Draw(b.s, cached).String(), Draw(b.s, ui).String(); got != want {
			t.Fatalf("%s: the cached frame differs:\n%s\nwant\n%s", step, got, want)
		}
	}
	for n, rest := 0, text; rest != ""; n++ {
		k := min(97, len(rest))
		b.ev(event.MessageDelta, event.Delta{MessageID: "m1", Block: "text", Text: rest[:k]})
		rest = rest[k:]
		check(fmt.Sprintf("delta %d", n), UI{Width: 120, Height: 40, Focus: "B"})
	}
	b.ev(event.MessageCompleted, event.Completed{MessageID: "m1", Text: text})
	for _, size := range [][2]int{{80, 24}, {200, 60}, {120, 40}} {
		ui := UI{Width: size[0], Height: size[1], Focus: "B"}
		check(fmt.Sprintf("%dx%d", size[0], size[1]), ui)
		for _, top := range []int{0, 5, 1 << 20} {
			ui.ConvHold, ui.ConvTop = true, top
			check(fmt.Sprintf("%dx%d held at %d", size[0], size[1], top), ui)
		}
	}
	fileLine := "file " + long
	f := FileView{Open: true, Path: "README.md", Rel: "README.md", Match: -1, Lines: append(slices.Clone(mdFile), fileLine)}
	check("the preview", UI{Width: 120, Height: 40, Focus: "B", File: f})
	f.Left, f.Cursor = 30, len(f.Lines)-1
	check("the preview shifted", UI{Width: 120, Height: 40, Focus: "B", File: f})
	b.s.Agents["a1"] = &state.Agent{Key: "a1", Label: "w", Status: state.AgentCompleted, Messages: []state.Entry{{Role: "assistant", Text: text}}}
	check("the inspection", UI{Width: 120, Height: 40, Inspect: "a1"})
	// Two frames without the preview: none of the file's lines kept.
	for range 2 {
		Draw(b.s, UI{Width: 120, Height: 40, Focus: "B", Wrap: wc})
	}
	for k := range wc.shown {
		if k.src == fileLine {
			t.Error("the file's line kept by a frame that did not draw it")
		}
	}
	for k := range wc.shownOld {
		if k.src == fileLine {
			t.Error("the file's line kept two frames after it was drawn")
		}
	}
}

// TestMarkdownStyles is design D4's styles made from a block's base: each
// with its base's colours and its own attributes, made once, painted again
// with the palette as the shared styles are (TestPalette16); a preview's
// cursor line's on zone A's cursor row ground.
func TestMarkdownStyles(t *testing.T) {
	defer UsePalette(pickTrue)
	type made struct {
		st, base             *lipgloss.Style
		bold, italic, strike bool
	}
	ms := []made{
		{mdStyle(stText, true, false, false), stText, true, false, false},
		{mdStyle(stText, false, true, false), stText, false, true, false},
		{mdStyle(stText, true, true, false), stText, true, true, false},
		{mdStyle(stText, false, false, true), stText, false, false, true},
		{mdStyle(stMdEm, true, false, false), stMdEm, true, false, false},
		{mdStyle(stMdLink, true, false, false), stMdLink, true, false, false},
		{mdStyle(stMdCode, false, false, true), stMdCode, false, false, true},
		{mdStyle(stMuted, false, true, false), stMuted, false, true, false},
	}
	if mdStyle(stText, false, false, false) != stText {
		t.Error("no construct: not its base")
	}
	if mdStyle(stText, true, false, false) != ms[0].st {
		t.Error("a style made twice")
	}
	cur := onCursorRow(mdStyle(stText, true, false, false))
	check := func(when string) {
		t.Helper()
		for k, m := range ms {
			st := m.st
			if st.GetForeground() != m.base.GetForeground() || st.GetBold() != (m.bold || m.base.GetBold()) ||
				st.GetItalic() != (m.italic || m.base.GetItalic()) || st.GetStrikethrough() != m.strike || st.GetUnderline() != m.base.GetUnderline() {
				t.Errorf("%s: style %d draws %q from its base %q", when, k, st.Render("x"), m.base.Render("x"))
			}
		}
		if cur.GetForeground() != stText.GetForeground() || !cur.GetBold() || cur.GetBackground() != cTreeRow.get() {
			t.Errorf("%s: the cursor line's strong text draws %q", when, cur.Render("x"))
		}
	}
	check("true colour")
	UsePalette(pickANSI)
	check("16 colours")
	UsePalette(pickTrue)
	check("true colour again")
}

// TestMarkdownZoneBHang is design D2 in zone B: a list item's later rows
// hang under its text, a quote's behind its bar, at 80 columns.
func TestMarkdownZoneBHang(t *testing.T) {
	b := newConvBench("claude-code", "a-model")
	b.prompt("go")
	words := strings.Repeat("word ", 30) + "end"
	b.msg("m1", "- "+words+"\n> "+words)
	ai := len(b.s.Conversation) - 1
	var rows []string
	for _, l := range bRows(b.s, UI{}, 76) {
		for _, x := range l {
			if x.ref.panel == 'B' && x.ref.item == ai {
				rows = append(rows, rowText(l))
				break
			}
		}
	}
	list, quote := -1, -1
	for i, r := range rows {
		if strings.Contains(r, "• word") {
			list = i
		}
		if strings.Contains(r, "│ word") && quote < 0 {
			quote = i
		}
	}
	if list < 0 || quote <= list+1 {
		t.Fatalf("rows:\n%s", strings.Join(rows, "\n"))
	}
	lead := rows[list][:strings.Index(rows[list], "• word")]
	for i := list + 1; i < quote; i++ {
		if !strings.HasPrefix(rows[i], lead+"  word") && !strings.HasPrefix(rows[i], lead+"  end") {
			t.Errorf("the list item's row %d %q, want %q then a word", i, rows[i], lead+"  ")
		}
	}
	for i := quote; i < len(rows); i++ {
		if !strings.HasPrefix(rows[i], lead+"│ word") && !strings.HasPrefix(rows[i], lead+"│ end") {
			t.Errorf("the quote's row %d %q, want %q then a word", i, rows[i], lead+"│ ")
		}
	}
}
