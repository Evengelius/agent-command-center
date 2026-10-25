package render

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
)

// clusterCases are texts and their grapheme clusters as UAX #29 makes them
// (plan 49, criterion 49.12).
var clusterCases = []struct {
	name     string
	clusters []string
}{
	{"combining marks", []string{"e\u0301", "a\u0308\u0323", "x"}},
	{"VS15", []string{"\u2764\uFE0E", " "}},
	{"VS16", []string{"\u2764\uFE0F", "\u263A\uFE0F"}},
	{"skin tone", []string{"\U0001F44D\U0001F3FD", "\U0001F44B\U0001F3FF"}},
	{"ZWJ family", []string{"\U0001F468\u200D\U0001F469\u200D\U0001F466", "!"}},
	{"ZWJ rainbow flag", []string{"\U0001F3F3\uFE0F\u200D\U0001F308"}},
	{"flags", []string{"\U0001F1EB\U0001F1F7", "\U0001F1EF\U0001F1F5", "\U0001F1E9"}},
	{"tag sequence", []string{"\U0001F3F4\U000E0067\U000E0062\U000E0073\U000E0063\U000E0074\U000E007F", "a"}},
	{"keycaps", []string{"1\uFE0F\u20E3", "#\uFE0F\u20E3"}},
	{"Hangul jamo", []string{"\u1100\u1161\u11A8", "\uAC00\u11A8", "\uAC01"}},
	{"Indic conjuncts", []string{"\u0915\u094D\u0937", "\u0915\u094D\u200D\u0928\u093F", "\u0915\u094D\u200D", " ", "\u0928\u093F"}},
	{"CJK", []string{"\u4E2D", "\u6587", "\uFF21"}},
	{"prepend", []string{"\u0600" + "1", "2"}},
	{"CR LF", []string{"a", "\r\n", "b"}},
	{"ASCII", []string{"a", "b", " ", "~"}},
}

// TestClusterWidths is plan 49's criterion 49.12 (design D11): each text is
// cut into its clusters, measured as lipgloss measures it, and cut and
// tailCut keep the most whole clusters that fit, with ASCII on either side
// too.
func TestClusterWidths(t *testing.T) {
	for _, c := range clusterCases {
		t.Run(c.name, func(t *testing.T) {
			for _, around := range [][2]string{{"", ""}, {"ab", ""}, {"", "yz"}, {"ab", "yz"}} {
				clusters := c.clusters
				if around[0] != "" {
					clusters = append([]string{"a", "b"}, clusters...)
				}
				if around[1] != "" {
					clusters = append(append([]string(nil), clusters...), "y", "z")
				}
				s := strings.Join(clusters, "")
				var widths []int
				at := 0
				for _, cl := range clusters {
					end, w := next(s, at)
					if s[at:end] != cl {
						t.Fatalf("%+q: the cluster at %d is %+q, want %+q", s, at, s[at:end], cl)
					}
					if want := lipgloss.Width(cl); w != want {
						t.Errorf("%+q: width %d, lipgloss %d", cl, w, want)
					}
					widths = append(widths, w)
					at = end
				}
				total := textWidth(s)
				// lipgloss measures each line apart: CR LF's case has two.
				if want := lipgloss.Width(s); total != want && !strings.Contains(s, "\n") {
					t.Errorf("%+q: textWidth %d, lipgloss %d", s, total, want)
				}
				for w := 0; w <= total+1; w++ {
					head, n := "", 0
					for i, cl := range clusters {
						if n+widths[i] > w {
							break
						}
						head, n = head+cl, n+widths[i]
					}
					if got := cut(s, w); got != head {
						t.Errorf("cut(%+q, %d) = %+q, want %+q", s, w, got, head)
					}
					tl, n := "", 0
					for i := len(clusters) - 1; i >= 0; i-- {
						if n+widths[i] > w {
							break
						}
						tl, n = clusters[i]+tl, n+widths[i]
					}
					if got := tailCut(s, w); got != tl {
						t.Errorf("tailCut(%+q, %d) = %+q, want %+q", s, w, got, tl)
					}
				}
			}
		})
	}
}

// FuzzTextWidth is plan 49's criterion 49.12: on any text the frame draws
// (UTF-8 without control characters, which it draws as atoms), textWidth is
// lipgloss's, and cut and tailCut keep the most that fits, ending where
// lipgloss's clusters end.
func FuzzTextWidth(f *testing.F) {
	for _, c := range clusterCases {
		f.Add(strings.Join(c.clusters, ""), 3)
	}
	f.Add("plain ASCII text", 7)
	f.Fuzz(func(t *testing.T, in string, w int) {
		s := strings.Map(func(r rune) rune {
			if r < 0x20 || r >= 0x7f && r < 0xa0 {
				return -1
			}
			return r
		}, strings.ToValidUTF8(in, ""))
		total := textWidth(s)
		if want := lipgloss.Width(s); total != want {
			t.Fatalf("textWidth(%+q) = %d, lipgloss %d", s, total, want)
		}
		w = min(max(w, 0), total+1)
		head := cut(s, w)
		rest := s[len(head):]
		switch {
		case !strings.HasPrefix(s, head):
			t.Fatalf("cut(%+q, %d) = %+q, not a prefix", s, w, head)
		case textWidth(head) > w:
			t.Fatalf("cut(%+q, %d) = %+q, %d cells", s, w, head, textWidth(head))
		case lipgloss.Width(head)+lipgloss.Width(rest) != total:
			t.Fatalf("cut(%+q, %d) = %+q splits a cluster", s, w, head)
		}
		if rest != "" {
			if _, cw := next(s, len(head)); textWidth(head)+cw <= w {
				t.Fatalf("cut(%+q, %d) = %+q: the next cluster fits", s, w, head)
			}
		}
		tl := tailCut(s, w)
		lead := s[:len(s)-len(tl)]
		switch {
		case !strings.HasSuffix(s, tl):
			t.Fatalf("tailCut(%+q, %d) = %+q, not a suffix", s, w, tl)
		case textWidth(tl) > w:
			t.Fatalf("tailCut(%+q, %d) = %+q, %d cells", s, w, tl, textWidth(tl))
		case lipgloss.Width(lead)+lipgloss.Width(tl) != total:
			t.Fatalf("tailCut(%+q, %d) = %+q splits a cluster", s, w, tl)
		}
	})
}

// BenchmarkTextWidth measures a row of ASCII, and one with CJK and emoji
// (design D11: ASCII takes a path of its own).
func BenchmarkTextWidth(b *testing.B) {
	for _, c := range []struct{ name, s string }{
		{"ascii", strings.Repeat("The quick brown fox jumps over the lazy dog. ", 4)},
		{"mixed", strings.Repeat("\u4E2D\u6587 text \U0001F44D \u00E9t\u00E9 ", 8)},
	} {
		b.Run(c.name, func(b *testing.B) {
			for b.Loop() {
				textWidth(c.s)
			}
		})
	}
}

// TestClusterUnits is plan 49's design D11 for the hit-testing: a cluster is
// one unit, as wide as it is drawn, its bytes all in it.
func TestClusterUnits(t *testing.T) {
	family := "\U0001F468\u200D\U0001F469\u200D\U0001F466"
	sc := &Screen{lines: []line{{span{text: "a" + family + "e\u0301b", ref: ref{panel: 'B', kind: refText}}}}}
	var got []string
	for _, u := range sc.units('B', 0) {
		got = append(got, fmt.Sprintf("%d-%d:%d-%d", u.x0, u.x1, u.start.Off, u.end.Off))
	}
	if want := []string{"0-1:0-1", "1-3:1-19", "3-4:19-22", "4-5:22-23"}; !slices.Equal(got, want) {
		t.Errorf("units %v, want %v", got, want)
	}
}

// TestClusterCells is plan 49's design D11 for the cells: the editor's and
// the conversation's cells are clusters, so the wrap fills a row with as
// many as fit.
func TestClusterCells(t *testing.T) {
	family := "\U0001F468\u200D\U0001F469\u200D\U0001F466"
	var got []string
	for _, c := range cells("a" + family + "e\u0301\t\x01") {
		got = append(got, fmt.Sprintf("%+q=%d", c.text, c.w))
	}
	want := []string{`"a"=1`, fmt.Sprintf("%+q=2", family), `"e\u0301"=1`, `"    "=4`, `"^A"=2`}
	if !slices.Equal(got, want) {
		t.Errorf("cells %v, want %v", got, want)
	}
	line := strings.Repeat(family, 10)
	if rows := wrapRanges(line, 6, 6); len(rows) != 4 {
		t.Errorf("ten families at 6 cells: %d rows, want 4 (three, three, three, one)", len(rows))
	}
	if n := rowCount(line, 6); n != 4 {
		t.Errorf("the editor counts %d rows for ten families at 6 cells, want 4", n)
	}
	if rs := rowsOf(0, line, 6); len(rs) != 4 {
		t.Errorf("the editor wraps ten families at 6 cells into %d rows, want 4", len(rs))
	}
}
