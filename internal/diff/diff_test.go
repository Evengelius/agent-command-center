package diff

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"testing"
	"time"
)

// unified formats res as git writes a unified diff, without the file
// header; the "\ No newline" marks are left to the flags.
func unified(res Result) string {
	span := func(start, n int) string {
		if n == 1 {
			return strconv.Itoa(start)
		}
		return fmt.Sprintf("%d,%d", start, n)
	}
	var sb strings.Builder
	for _, h := range res.Hunks {
		fmt.Fprintf(&sb, "@@ -%s +%s @@\n", span(h.OldStart, h.OldLines), span(h.NewStart, h.NewLines))
		for _, l := range h.Lines {
			sb.WriteByte(byte(l.Kind))
			sb.WriteString(l.Text)
			sb.WriteByte('\n')
		}
	}
	return sb.String()
}

// oracleLines is s cut at LF, a CR before the LF dropped, the empty line after
// a final LF not a line: written apart from the package's own split.
func oracleLines(s string) []string {
	var out []string
	for s != "" {
		l, rest, hadLF := strings.Cut(s, "\n")
		if hadLF {
			l = strings.TrimSuffix(l, "\r")
		}
		out = append(out, l)
		s = rest
	}
	return out
}

// valid is lines with invalid UTF-8 replaced, as Text shows them.
func valid(lines []string) []string {
	out := make([]string, len(lines))
	for i, l := range lines {
		out[i] = strings.ToValidUTF8(l, "�")
	}
	return out
}

// same reports whether two line lists are equal.
func same(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// replay applies res to the old lines and returns the new ones; it checks on
// the way every number, every header count and every line it can against the
// old side, as a patch tool would.
func replay(t *testing.T, a []string, res Result) []string {
	t.Helper()
	var out []string
	ai, added, removed := 0, 0, 0
	for hi, h := range res.Hunks {
		start := h.OldStart - 1
		if h.OldLines == 0 {
			start = h.OldStart
		}
		if start < ai || start > len(a) {
			t.Fatalf("hunk %d starts at old line %d, after %d lines were used of %d", hi, h.OldStart, ai, len(a))
		}
		out = append(out, a[ai:start]...)
		ai = start
		newStart := len(out) + 1
		if h.NewLines == 0 {
			newStart = len(out)
		}
		if h.NewStart != newStart {
			t.Fatalf("hunk %d: NewStart %d, want %d", hi, h.NewStart, newStart)
		}
		oldN, newN := 0, 0
		for li, l := range h.Lines {
			switch l.Kind {
			case Context:
				if l.Old != ai+1 || l.New != len(out)+1 || ai >= len(a) || l.Text != a[ai] {
					t.Fatalf("hunk %d line %d: context %+v, old line %d is %q", hi, li, l, ai+1, lineAt(a, ai))
				}
				out, ai, oldN, newN = append(out, l.Text), ai+1, oldN+1, newN+1
			case Removed:
				if l.Old != ai+1 || l.New != 0 || ai >= len(a) || l.Text != a[ai] {
					t.Fatalf("hunk %d line %d: removed %+v, old line %d is %q", hi, li, l, ai+1, lineAt(a, ai))
				}
				ai, oldN, removed = ai+1, oldN+1, removed+1
			case Added:
				if l.Old != 0 || l.New != len(out)+1 {
					t.Fatalf("hunk %d line %d: added %+v, new line should be %d", hi, li, l, len(out)+1)
				}
				out, newN, added = append(out, l.Text), newN+1, added+1
			default:
				t.Fatalf("hunk %d line %d: kind %q", hi, li, l.Kind)
			}
		}
		if h.OldLines != oldN || h.NewLines != newN {
			t.Fatalf("hunk %d: header counts %d/%d, lines %d/%d", hi, h.OldLines, h.NewLines, oldN, newN)
		}
	}
	if added != res.Added || removed != res.Removed {
		t.Fatalf("Added/Removed = %d/%d, lines %d/%d", res.Added, res.Removed, added, removed)
	}
	return append(out, a[ai:]...)
}

func lineAt(a []string, i int) string {
	if i < len(a) {
		return a[i]
	}
	return "<end>"
}

// lcs is the length of a longest common subsequence, by dynamic programming:
// the brute force the minimality is checked against.
func lcs(a, b []string) int {
	prev, cur := make([]int, len(b)+1), make([]int, len(b)+1)
	for i := 1; i <= len(a); i++ {
		for j := 1; j <= len(b); j++ {
			if a[i-1] == b[j-1] {
				cur[j] = prev[j-1] + 1
			} else {
				cur[j] = max(prev[j], cur[j-1])
			}
		}
		prev, cur = cur, prev
	}
	return prev[len(b)]
}

func join(lines []string) string {
	if len(lines) == 0 {
		return ""
	}
	return strings.Join(lines, "\n") + "\n"
}

func nums(from, to int) string {
	var sb strings.Builder
	for i := from; i <= to; i++ {
		sb.WriteString(strconv.Itoa(i))
		sb.WriteByte('\n')
	}
	return sb.String()
}

// TestCompareHunks covers plan 44's criterion 44.7 on exact hunks: the header
// numbers, the order of removed and added lines, the context and its merging.
func TestCompareHunks(t *testing.T) {
	ten := nums(1, 10)
	for _, c := range []struct {
		name, a, b string
		context    int
		want       string
	}{
		{"identical", ten, ten, 3, ""},
		{"both empty", "", "", 3, ""},
		{"replace", "a\nb\nc\n", "a\nB\nc\n", 3, "@@ -1,3 +1,3 @@\n a\n-b\n+B\n c\n"},
		{"removed before added", "a\nb\nc\nd\n", "a\nX\nY\nd\n", 3, "@@ -1,4 +1,4 @@\n a\n-b\n-c\n+X\n+Y\n d\n"},
		{"insert at the top", "a\nb\n", "N\na\nb\n", 3, "@@ -1,2 +1,3 @@\n+N\n a\n b\n"},
		{"insert at the end", "a\nb\n", "a\nb\nN\n", 3, "@@ -1,2 +1,3 @@\n a\n b\n+N\n"},
		{"delete the first", "a\nb\nc\n", "b\nc\n", 3, "@@ -1,3 +1,2 @@\n-a\n b\n c\n"},
		{"from nothing", "", "x\ny\n", 3, "@@ -0,0 +1,2 @@\n+x\n+y\n"},
		{"to nothing", "x\ny\n", "", 3, "@@ -1,2 +0,0 @@\n-x\n-y\n"},
		{"one line to one line", "x\n", "y\n", 3, "@@ -1 +1 @@\n-x\n+y\n"},
		{"context 0", ten, "1\n2\nX\n4\n5\n6\n7\nY\n9\n10\n", 0, "@@ -3 +3 @@\n-3\n+X\n@@ -8 +8 @@\n-8\n+Y\n"},
		{"context 0, a pure insertion", "a\nb\n", "a\nN\nb\n", 0, "@@ -1,0 +2 @@\n+N\n"},
		{"context 0, a pure deletion", "a\nN\nb\n", "a\nb\n", 0, "@@ -2 +1,0 @@\n-N\n"},
		{"a negative context is 0", "a\nN\nb\n", "a\nb\n", -5, "@@ -2 +1,0 @@\n-N\n"},
		{"context 1", ten, "1\n2\nX\n4\n5\n6\n7\nY\n9\n10\n", 1, "@@ -2,3 +2,3 @@\n 2\n-3\n+X\n 4\n@@ -7,3 +7,3 @@\n 7\n-8\n+Y\n 9\n"},
		// Two changes with 6 lines between them, 2*context: the contexts touch.
		{"touching contexts merge", nums(1, 12), "1\n2\nX\n4\n5\n6\n7\n8\n9\nY\n11\n12\n", 3,
			"@@ -1,12 +1,12 @@\n 1\n 2\n-3\n+X\n 4\n 5\n 6\n 7\n 8\n 9\n-10\n+Y\n 11\n 12\n"},
		// With 7 lines between them they do not.
		{"separate contexts split", nums(1, 13), "1\n2\nX\n4\n5\n6\n7\n8\n9\n10\nY\n12\n13\n", 3,
			"@@ -1,6 +1,6 @@\n 1\n 2\n-3\n+X\n 4\n 5\n 6\n@@ -8,6 +8,6 @@\n 8\n 9\n 10\n-11\n+Y\n 12\n 13\n"},
		{"final line without LF changed", "x\ny", "x\nz", 3, "@@ -1,2 +1,2 @@\n x\n-y\n+z\n"},
	} {
		t.Run(c.name, func(t *testing.T) {
			res := Compare([]byte(c.a), []byte(c.b), c.context)
			if got := unified(res); got != c.want {
				t.Errorf("hunks:\n%s\nwant:\n%s", got, c.want)
			}
			if res.Whole || res.EOLOnly {
				t.Errorf("Whole %v, EOLOnly %v", res.Whole, res.EOLOnly)
			}
			if got, want := replay(t, oracleLines(c.a), res), oracleLines(c.b); !same(got, want) {
				t.Errorf("replay = %q, want %q", got, want)
			}
		})
	}
}

// TestCompareEnds covers the line ends: a CR before the LF is dropped, the
// final newline is part of the last line, a difference of ends alone is told.
func TestCompareEnds(t *testing.T) {
	for _, c := range []struct {
		name, a, b                string
		want                      string
		noNLOld, noNLNew, eolOnly bool
	}{
		{name: "CRLF against LF only", a: "a\r\nb\r\n", b: "a\nb\n", eolOnly: true},
		{name: "LF against CRLF only", a: "a\nb\n", b: "a\r\nb\r\n", eolOnly: true},
		{name: "mixed ends only", a: "a\r\nb\n", b: "a\nb\r\n", eolOnly: true},
		{name: "CRLF and a real change", a: "a\r\nb\r\n", b: "a\nB\n", want: "@@ -1,2 +1,2 @@\n a\n-b\n+B\n"},
		{name: "a lone CR is kept", a: "x\ry\n", b: "x\rz\n", want: "@@ -1 +1 @@\n-x\ry\n+x\rz\n"},
		{name: "two CRs, one dropped", a: "a\r\r\n", b: "a\r\n", want: "@@ -1 +1 @@\n-a\r\n+a\n"},
		{name: "old lacks the final LF", a: "x\ny", b: "x\ny\n", want: "@@ -1,2 +1,2 @@\n x\n-y\n+y\n", noNLOld: true},
		{name: "new lacks the final LF", a: "x\ny\n", b: "x\ny", want: "@@ -1,2 +1,2 @@\n x\n-y\n+y\n", noNLNew: true},
		{name: "new adds a line, no final LF", a: "x\ny\n", b: "x\ny\nz", want: "@@ -1,2 +1,3 @@\n x\n y\n+z\n", noNLNew: true},
		{name: "old ends without LF, new continues", a: "x\ny", b: "x\ny\nz\n", want: "@@ -1,2 +1,3 @@\n x\n-y\n+y\n+z\n", noNLOld: true},
		{name: "neither ends with LF, a change above", a: "x\ny", b: "X\ny", want: "@@ -1,2 +1,2 @@\n-x\n+X\n y\n", noNLOld: true, noNLNew: true},
		{name: "neither ends with LF, equal", a: "x\ny", b: "x\ny", noNLOld: true, noNLNew: true},
		{name: "an empty line is a line", a: "\n", b: "\n\n", want: "@@ -1 +1,2 @@\n \n+\n"},
		{name: "empty against one empty line", a: "", b: "\n", want: "@@ -0,0 +1 @@\n+\n"},
	} {
		t.Run(c.name, func(t *testing.T) {
			res := Compare([]byte(c.a), []byte(c.b), 3)
			if got := unified(res); got != c.want {
				t.Errorf("hunks:\n%q\nwant:\n%q", got, c.want)
			}
			if res.NoNewlineOld != c.noNLOld || res.NoNewlineNew != c.noNLNew || res.EOLOnly != c.eolOnly || res.Whole {
				t.Errorf("flags: noNLOld %v noNLNew %v eolOnly %v whole %v", res.NoNewlineOld, res.NoNewlineNew, res.EOLOnly, res.Whole)
			}
			if res.EOLOnly && (len(res.Hunks) != 0 || res.Added != 0 || res.Removed != 0) {
				t.Errorf("EOLOnly with hunks: %+v", res)
			}
		})
	}
}

// TestCompareUTF8 covers invalid UTF-8: the bytes are compared, the text
// shown is valid.
func TestCompareUTF8(t *testing.T) {
	res := Compare([]byte("a\xffb\nsame\xfe\n"), []byte("a\xfeb\nsame\xfe\n"), 3)
	want := "@@ -1,2 +1,2 @@\n-a�b\n+a�b\n same�\n"
	if got := unified(res); got != want {
		t.Errorf("hunks %q, want %q", got, want)
	}
	res = Compare([]byte("é à\n日本\n"), []byte("é à\n日本語\n"), 3)
	if want := "@@ -1,2 +1,2 @@\n é à\n-日本\n+日本語\n"; unified(res) != want {
		t.Errorf("unicode: %q", unified(res))
	}
}

// checkContext checks a result's hunks against the rules of the context: each
// begins with ctx unchanged lines, or with all there are above it, and ends
// with ctx, or with all there are below it; the unchanged lines inside a hunk
// between two changes number at most 2*ctx; two hunks are more than 2*ctx
// unchanged lines apart.
func checkContext(t *testing.T, res Result, ctx, oldLines int) {
	t.Helper()
	prevEnd := 0 // the old line after the previous hunk, 0-based
	for hi, h := range res.Hunks {
		first, last := -1, -1
		for i, l := range h.Lines {
			if l.Kind != Context {
				if first < 0 {
					first = i
				}
				last = i
			}
		}
		if first < 0 {
			t.Fatalf("hunk %d has no change", hi)
		}
		start := h.OldStart - 1
		if h.OldLines == 0 {
			start = h.OldStart
		}
		// Both hunks hold ctx unchanged lines on this side of their changes,
		// so with no old line between them the changes are 2*ctx lines
		// apart: their contexts touch and they are one hunk.
		if hi > 0 && start-prevEnd <= 0 {
			t.Fatalf("hunk %d begins where the previous one ends: the contexts of %d touch", hi, ctx)
		}
		if lead := first; lead != ctx && start != 0 || lead > ctx {
			t.Fatalf("hunk %d: %d lines of leading context at old line %d, want %d", hi, lead, h.OldStart, ctx)
		}
		end := start + h.OldLines
		if trail := len(h.Lines) - 1 - last; trail != ctx && end != oldLines || trail > ctx {
			t.Fatalf("hunk %d: %d lines of trailing context ending at old line %d of %d, want %d", hi, trail, end, oldLines, ctx)
		}
		run := 0
		for i := first; i <= last; i++ {
			if h.Lines[i].Kind == Context {
				run++
				continue
			}
			if run > 2*ctx {
				t.Fatalf("hunk %d: %d unchanged lines inside, more than %d", hi, run, 2*ctx)
			}
			run = 0
		}
		prevEnd = end
	}
}

// TestCompareRandomScripts covers criterion 44.7: edits applied to a text,
// then the hunks replayed on it give the new text, with every number right,
// and no script is longer than a shortest one.
func TestCompareRandomScripts(t *testing.T) {
	rng := rand.New(rand.NewSource(44))
	word := func() string { return "w" + strconv.Itoa(rng.Intn(40)) }
	for n := 0; n < 3000; n++ {
		old := make([]string, rng.Intn(120))
		for i := range old {
			old[i] = word()
		}
		cur := append([]string(nil), old...)
		for e := rng.Intn(25); e > 0; e-- {
			i := 0
			if len(cur) > 0 {
				i = rng.Intn(len(cur) + 1)
			}
			switch rng.Intn(3) {
			case 0: // insert
				cur = append(cur[:i], append([]string{word()}, cur[i:]...)...)
			case 1: // delete
				if i < len(cur) {
					cur = append(cur[:i], cur[i+1:]...)
				}
			default: // replace
				if i < len(cur) {
					cur[i] = word()
				}
			}
		}
		ctx := rng.Intn(5)
		a, b := join(old), join(cur)
		res := Compare([]byte(a), []byte(b), ctx)
		if res.Whole {
			t.Fatalf("case %d: bounded", n)
		}
		if got := replay(t, old, res); !same(got, cur) {
			t.Fatalf("case %d: the replay differs from the new side", n)
		}
		if a == b && len(res.Hunks) != 0 {
			t.Fatalf("case %d: hunks for equal texts", n)
		}
		if want := len(old) + len(cur) - 2*lcs(old, cur); res.Added+res.Removed != want {
			t.Fatalf("case %d: %d edits, a shortest script has %d\nold %q\nnew %q", n, res.Added+res.Removed, want, old, cur)
		}
		checkContext(t, res, ctx, len(old))
	}
}

// TestCompareMinimal covers criterion 44.7's brute force: on every small pair
// the number of edits is the shortest there is, and the replay holds.
func TestCompareMinimal(t *testing.T) {
	rng := rand.New(rand.NewSource(7))
	for n := 0; n < 40000; n++ {
		alphabet := 1 + rng.Intn(4)
		gen := func() []string {
			s := make([]string, rng.Intn(13))
			for i := range s {
				s[i] = string(rune('a' + rng.Intn(alphabet)))
			}
			return s
		}
		x, y := gen(), gen()
		ctx := rng.Intn(4)
		res := Compare([]byte(join(x)), []byte(join(y)), ctx)
		if res.Whole {
			t.Fatalf("bounded: %q %q", x, y)
		}
		if want := len(x) + len(y) - 2*lcs(x, y); res.Added+res.Removed != want {
			t.Fatalf("%q -> %q: %d edits, shortest is %d\n%s", x, y, res.Added+res.Removed, want, unified(res))
		}
		if got := replay(t, x, res); !same(got, y) {
			t.Fatalf("%q -> %q: replay %q", x, y, got)
		}
		checkContext(t, res, ctx, len(x))
	}
}

// TestCompareMinimalSkewed is the same on sides of very different sizes, a
// narrow grid whose edges the search must not run off.
func TestCompareMinimalSkewed(t *testing.T) {
	rng := rand.New(rand.NewSource(77))
	for n := 0; n < 20000; n++ {
		alphabet := 1 + rng.Intn(5)
		gen := func(most int) []string {
			s := make([]string, rng.Intn(most+1))
			for i := range s {
				s[i] = string(rune('a' + rng.Intn(alphabet)))
			}
			return s
		}
		var x, y []string
		switch n % 3 {
		case 0:
			x, y = gen(40), gen(4)
		case 1:
			x, y = gen(4), gen(40)
		default:
			x, y = gen(25), gen(25)
		}
		res := Compare([]byte(join(x)), []byte(join(y)), 1+rng.Intn(3))
		if res.Whole {
			t.Fatalf("bounded: %q %q", x, y)
		}
		if want := len(x) + len(y) - 2*lcs(x, y); res.Added+res.Removed != want {
			t.Fatalf("%q -> %q: %d edits, shortest is %d\n%s", x, y, res.Added+res.Removed, want, unified(res))
		}
		if got := replay(t, x, res); !same(got, y) {
			t.Fatalf("%q -> %q: replay %q", x, y, got)
		}
	}
}

// TestCompareMinimalLarger is the same on texts of a few thousand lines and
// some hundreds of edits, whose search goes through many halvings.
func TestCompareMinimalLarger(t *testing.T) {
	rng := rand.New(rand.NewSource(11))
	for n := 0; n < 12; n++ {
		size := 1500 + rng.Intn(1500)
		alphabet := []int{3, 30, 400, 100000}[n%4]
		old := make([]string, size)
		for i := range old {
			old[i] = strconv.Itoa(rng.Intn(alphabet))
		}
		cur := make([]string, 0, size)
		for _, l := range old {
			switch r := rng.Intn(100); {
			case r < 6: // delete
			case r < 12: // replace
				cur = append(cur, strconv.Itoa(rng.Intn(alphabet)))
			case r < 18: // insert before
				cur = append(cur, strconv.Itoa(rng.Intn(alphabet)), l)
			default:
				cur = append(cur, l)
			}
		}
		res := Compare([]byte(join(old)), []byte(join(cur)), 3)
		if res.Whole {
			t.Fatalf("alphabet %d: bounded at %d edits", alphabet, res.Added+res.Removed)
		}
		if want := len(old) + len(cur) - 2*lcs(old, cur); res.Added+res.Removed != want {
			t.Fatalf("alphabet %d: %d edits, shortest is %d", alphabet, res.Added+res.Removed, want)
		}
		if got := replay(t, old, res); !same(got, cur) {
			t.Fatalf("alphabet %d: replay differs", alphabet)
		}
		checkContext(t, res, 3, len(old))
	}
}

// TestCompareBound covers criterion 44.7's bound: a middle over MaxLines, or
// over MaxEdits, the head and tail stay context and the middle is removed
// then added, said so; at the limits it is not, nor is a long file whose
// changes lie close together.
func TestCompareBound(t *testing.T) {
	t.Run("a middle over MaxLines", func(t *testing.T) {
		// A side over MaxLines, one line changed: its middle is that line,
		// compared as any.
		old := nums(1, MaxLines+1)
		cur := strings.Replace(old, "\n50000\n", "\nfifty thousand\n", 1)
		res := Compare([]byte(old), []byte(cur), 3)
		want := "@@ -49997,7 +49997,7 @@\n 49997\n 49998\n 49999\n-50000\n+fifty thousand\n 50001\n 50002\n 50003\n"
		if res.Whole || unified(res) != want || res.Added != 1 || res.Removed != 1 {
			t.Errorf("Whole %v, %d/%d\n%s", res.Whole, res.Added, res.Removed, unified(res))
		}
		// Lines 2 and MaxLines+1 changed: a middle of MaxLines lines, compared.
		old = nums(1, MaxLines+2)
		at := func(s string, line int, to string) string {
			return strings.Replace(s, fmt.Sprintf("\n%d\n", line), "\n"+to+"\n", 1)
		}
		cur = at(at(old, 2, "two"), MaxLines+1, "last but one")
		if res := Compare([]byte(old), []byte(cur), 3); res.Whole || res.Added != 2 || res.Removed != 2 || len(res.Hunks) != 2 {
			t.Errorf("a middle of MaxLines lines: Whole %v, %d/%d, %d hunks", res.Whole, res.Added, res.Removed, len(res.Hunks))
		}
		// Lines 2 and MaxLines+2: one more, bounded, on either side.
		cur = at(at(old, 2, "two"), MaxLines+2, "last")
		if res := Compare([]byte(old), []byte(cur), 3); !res.Whole || res.Added != MaxLines+1 || res.Removed != MaxLines+1 {
			t.Errorf("a middle of MaxLines+1 lines: Whole %v, %d/%d", res.Whole, res.Added, res.Removed)
		}
		if res := Compare([]byte(cur), []byte(old), 3); !res.Whole {
			t.Errorf("the same, sides swapped: not bounded")
		}
	})

	t.Run("the middle whole between its head and tail", func(t *testing.T) {
		var a, b strings.Builder
		a.WriteString("h1\nh2\nh3\nh4\n")
		b.WriteString("h1\nh2\nh3\nh4\n")
		for i := 0; i < 25_000; i++ {
			fmt.Fprintf(&a, "old %d\n", i)
			fmt.Fprintf(&b, "new %d\n", i)
		}
		a.WriteString("t1\nt2\nt3\nt4\n")
		b.WriteString("t1\nt2\nt3\nt4\n")
		res := Compare([]byte(a.String()), []byte(b.String()), 2)
		if !res.Whole || len(res.Hunks) != 1 || res.Removed != 25_000 || res.Added != 25_000 {
			t.Fatalf("Whole %v, %d hunks, %d/%d", res.Whole, len(res.Hunks), res.Added, res.Removed)
		}
		h := res.Hunks[0]
		if h.OldStart != 3 || h.OldLines != 25_004 || h.NewStart != 3 || h.NewLines != 25_004 || len(h.Lines) != 50_004 {
			t.Fatalf("header %d,%d %d,%d, %d lines", h.OldStart, h.OldLines, h.NewStart, h.NewLines, len(h.Lines))
		}
		kinds := func(from, to int) string {
			var sb strings.Builder
			for _, l := range h.Lines[from:to] {
				sb.WriteByte(byte(l.Kind))
			}
			return sb.String()
		}
		if kinds(0, 3) != "  -" || kinds(25_001, 25_003) != "-+" || kinds(len(h.Lines)-3, len(h.Lines)) != "+  " {
			t.Errorf("order: %q %q %q", kinds(0, 3), kinds(25_001, 25_003), kinds(len(h.Lines)-3, len(h.Lines)))
		}
		if h.Lines[0].Text != "h3" || h.Lines[1].Text != "h4" || h.Lines[len(h.Lines)-1].Text != "t2" {
			t.Errorf("context: %q %q %q", h.Lines[0].Text, h.Lines[1].Text, h.Lines[len(h.Lines)-1].Text)
		}
		if l := h.Lines[25_002]; l.Kind != Added || l.New != 5 || l.Old != 0 || l.Text != "new 0" {
			t.Errorf("first added line: %+v", l)
		}
		if got := replay(t, oracleLines(a.String()), res); !same(got, oracleLines(b.String())) {
			t.Errorf("the replay of the bounded result differs")
		}
	})

	t.Run("MaxEdits", func(t *testing.T) {
		pair := func(removed, added int) (string, string) {
			var a, b strings.Builder
			a.WriteString("top\n")
			b.WriteString("top\n")
			for i := 0; i < removed; i++ {
				fmt.Fprintf(&a, "old %d\n", i)
			}
			for i := 0; i < added; i++ {
				fmt.Fprintf(&b, "new %d\n", i)
			}
			a.WriteString("bottom\n")
			b.WriteString("bottom\n")
			return a.String(), b.String()
		}
		a, b := pair(MaxEdits/2, MaxEdits/2) // exactly MaxEdits edits
		if res := Compare([]byte(a), []byte(b), 3); res.Whole || res.Added+res.Removed != MaxEdits {
			t.Errorf("%d edits: Whole %v, %d", MaxEdits, res.Whole, res.Added+res.Removed)
		}
		a, b = pair(MaxEdits/2+1, MaxEdits/2) // one more
		if res := Compare([]byte(a), []byte(b), 3); !res.Whole || res.Added+res.Removed != MaxEdits+1 {
			t.Errorf("%d edits: Whole %v, %d", MaxEdits+1, res.Whole, res.Added+res.Removed)
		}
		a, b = pair(0, MaxEdits+1) // a pure insertion of more than MaxEdits lines
		if res := Compare([]byte(a), []byte(b), 3); !res.Whole || res.Added != MaxEdits+1 || res.Removed != 0 {
			t.Errorf("insertion of %d lines: Whole %v", MaxEdits+1, res.Whole)
		}
	})
}

// swaps is texts of n disjoint swaps of two neighbouring lines, 2n edits,
// every line common to both sides so that the reduction discards none.
func swaps(n int) (a, b string) {
	var x, y strings.Builder
	for i := 0; i < n; i++ {
		fmt.Fprintf(&x, "line %d\nline %d\n", 2*i, 2*i+1)
		fmt.Fprintf(&y, "line %d\nline %d\n", 2*i+1, 2*i)
	}
	return x.String(), y.String()
}

// TestCompareBoundSearch is the bound met by the search itself: lines both
// sides hold, in another order, so that only the edits say it.
func TestCompareBoundSearch(t *testing.T) {
	if testing.Short() || raceEnabled {
		t.Skip("about a second of search")
	}
	a, b := swaps(MaxEdits / 2) // exactly MaxEdits edits
	start := time.Now()
	res := Compare([]byte(a), []byte(b), 3)
	t.Logf("%d edits: %v", MaxEdits, time.Since(start))
	if res.Whole || res.Added+res.Removed != MaxEdits {
		t.Errorf("%d edits: Whole %v, %d", MaxEdits, res.Whole, res.Added+res.Removed)
	}
	a, b = swaps(MaxEdits/2 + 1)
	start = time.Now()
	res = Compare([]byte(a), []byte(b), 3)
	t.Logf("%d edits: %v", MaxEdits+2, time.Since(start))
	if !res.Whole {
		t.Errorf("%d edits: not bounded", MaxEdits+2)
	}
	// The lines only one side holds are edits too, and count against the
	// bound beside the search's: 10,000 edits of swaps, then 10,000 or 10,001
	// of lines the other side lacks.
	for _, c := range []struct {
		onlyOld, onlyNew int
		whole            bool
	}{{5000, 5000, false}, {5000, 5001, true}, {0, 10_000, false}, {10_001, 0, true}} {
		var x, y strings.Builder
		for i := 0; i < c.onlyOld; i++ {
			fmt.Fprintf(&x, "only in the old %d\n", i)
		}
		for i := 0; i < c.onlyNew; i++ {
			fmt.Fprintf(&y, "only in the new %d\n", i)
		}
		s, w := swaps(MaxEdits / 4) // 10,000 edits
		x.WriteString(s)
		y.WriteString(w)
		res := Compare([]byte(x.String()), []byte(y.String()), 3)
		if res.Whole != c.whole || !c.whole && res.Added+res.Removed != c.onlyOld+c.onlyNew+MaxEdits/2 {
			t.Errorf("%d + %d lines of one side and the swaps: Whole %v (want %v), %d edits", c.onlyOld, c.onlyNew, res.Whole, c.whole, res.Added+res.Removed)
		}
	}
	// Reversed: the worst case for the search, bounded by the edits.
	var x, y strings.Builder
	const n = 60_000
	for i := 0; i < n; i++ {
		fmt.Fprintf(&x, "line %d\n", i)
		fmt.Fprintf(&y, "line %d\n", n-1-i)
	}
	start = time.Now()
	res = Compare([]byte(x.String()), []byte(y.String()), 3)
	t.Logf("reversed %d lines: %v", n, time.Since(start))
	if !res.Whole || res.Removed != n || res.Added != n {
		t.Errorf("reversed: Whole %v, %d/%d", res.Whole, res.Added, res.Removed)
	}
}

// slow is how long a test gets, longer under the race detector.
func slow(d time.Duration) time.Duration {
	if raceEnabled {
		return 10 * d
	}
	return d
}

// TestCompareSpeed covers criterion 44.7's time: 100,000-line sides with 50
// scattered edits well under a second, and a fully different pair of that
// size answered at once by the bound.
func TestCompareSpeed(t *testing.T) {
	rng := rand.New(rand.NewSource(3))
	old := make([]string, MaxLines-20) // the insertions keep the new side within MaxLines
	for i := range old {
		switch {
		case i%7 == 0:
			old[i] = ""
		case i%11 == 0:
			old[i] = "}"
		default:
			old[i] = fmt.Sprintf("line %d of the file, with some text %d", i, rng.Intn(1000))
		}
	}
	cur := append([]string(nil), old...)
	for e := 49; e >= 0; e-- { // from the end, so that an insertion moves no later position
		i := e*2000 + 100 + rng.Intn(1000)
		switch e % 3 {
		case 0:
			cur[i] = "changed " + cur[i]
		case 1:
			cur[i] += " (and more)"
		default:
			cur = append(cur[:i+1], append([]string{"inserted " + strconv.Itoa(e)}, cur[i+1:]...)...)
		}
	}
	a, b := join(old), join(cur)

	start := time.Now()
	res := Compare([]byte(a), []byte(b), 3)
	took := time.Since(start)
	t.Logf("100,000 lines, 50 edits: %v, %d hunks, %d added, %d removed", took, len(res.Hunks), res.Added, res.Removed)
	if took > slow(time.Second) {
		t.Errorf("took %v", took)
	}
	if res.Whole || len(res.Hunks) != 50 {
		t.Errorf("Whole %v, %d hunks, want 50", res.Whole, len(res.Hunks))
	}
	if got := replay(t, old, res); !same(got, cur) {
		t.Errorf("the replay differs")
	}

	var x, y strings.Builder
	for i := 0; i < MaxLines; i++ {
		fmt.Fprintf(&x, "old line %d\n", i)
		fmt.Fprintf(&y, "new line %d\n", i)
	}
	start = time.Now()
	res = Compare([]byte(x.String()), []byte(y.String()), 3)
	took = time.Since(start)
	t.Logf("100,000 lines, all different: %v", took)
	if took > slow(time.Second) || !res.Whole || res.Removed != MaxLines || res.Added != MaxLines || len(res.Hunks) != 1 {
		t.Errorf("all different: %v, Whole %v, %d/%d, %d hunks", took, res.Whole, res.Added, res.Removed, len(res.Hunks))
	}
}

// TestCompareMessy runs the engine on shapes that stress the ends: CRs, LFs,
// invalid bytes, runs of empty lines, a missing final newline.
func TestCompareMessy(t *testing.T) {
	rng := rand.New(rand.NewSource(99))
	mk := func() string {
		var sb strings.Builder
		for i := rng.Intn(30); i > 0; i-- {
			sb.WriteString(strings.Repeat("x", rng.Intn(3)))
			switch rng.Intn(6) {
			case 0:
				sb.WriteString("\r\n")
			case 1:
				sb.WriteString("\r")
			case 2:
				sb.WriteString("\xff")
			default:
				sb.WriteString("\n")
			}
		}
		return sb.String()
	}
	for n := 0; n < 6000; n++ {
		a, b := mk(), mk()
		ctx := rng.Intn(4) - 1
		res := Compare([]byte(a), []byte(b), ctx)
		la, lb := oracleLines(a), oracleLines(b)
		if got := replay(t, valid(la), res); !same(got, valid(lb)) {
			t.Fatalf("%q -> %q: replay %q", a, b, got)
		}
		eol := a != b && same(la, lb) && (a == "" || a[len(a)-1] == '\n') == (b == "" || b[len(b)-1] == '\n')
		if res.EOLOnly != eol {
			t.Fatalf("%q -> %q: EOLOnly %v, want %v", a, b, res.EOLOnly, eol)
		}
		if res.NoNewlineOld != (a != "" && a[len(a)-1] != '\n') || res.NoNewlineNew != (b != "" && b[len(b)-1] != '\n') {
			t.Fatalf("%q -> %q: NoNewline %v/%v", a, b, res.NoNewlineOld, res.NoNewlineNew)
		}
	}
}
