package render

import (
	"sync"
	"sync/atomic"
	"unicode"
	"unicode/utf8"

	"charm.land/lipgloss/v2"
)

// Grapheme clusters (plan 49, design D11; audit finding 5): what a terminal
// measuring them draws as one character. Windows Terminal reports the
// Unicode core mode (2027), so Bubble Tea's cell buffer measures them too
// (third_party/ultraviolet/PATCHES.md, patch 3): a row's width here must be
// theirs, or its border shifts.
//
// The segmentation is the dashboard's own, from Go's Unicode tables (17.0),
// with no dependency added. Where UAX #29 might keep two code points
// together, it does: a cluster here is one or more of UAX #29's, never a
// part of one, so its width, lipgloss's for the whole, is never wrong; at
// worst a cut keeps two of UAX #29's clusters together.

// next is the end of the cluster that starts at byte i of s, and its width.
// A control is a cluster of its own and takes no cell; text in ASCII takes
// one cell a byte.
func next(s string, i int) (end, width int) {
	if b := s[i]; b < 0x80 && (i+1 == len(s) || s[i+1] < 0x80) {
		switch {
		case b == '\r' && i+1 < len(s) && s[i+1] == '\n':
			return i + 2, 0 // GB3
		case b < 0x20 || b == 0x7f:
			return i + 1, 0
		}
		return i + 1, 1
	}
	r, n := utf8.DecodeRuneInString(s[i:])
	end, width, _ = cluster(s, i, r, n)
	return end, width
}

// decoded is a code point and its length in bytes, 0 for none.
type decoded struct {
	r rune
	n int
}

// cluster is next for a cluster whose first code point, r, n bytes long, is
// decoded already; after is the code point it ends before, decoded, if it
// did decode one (not ASCII, not a byte that is not UTF-8).
func cluster(s string, i int, r rune, n int) (end, width int, after decoded) {
	end = i + n
	f := factsOf(r)
	if f&control != 0 || r == utf8.RuneError && n == 1 {
		return end, f.width(), after // a byte that is not UTF-8 is one cell, as before
	}
	// The state UAX #29 keeps: the regional indicators just before, and an
	// Indic linker since the last letter. Where a code point is kept that
	// UAX #29 would start a cluster with, the state is the one it would start
	// with, so the cluster ends where one of UAX #29's ends.
	prev, ris, linked := f, 0, false
	if f&regional != 0 {
		ris = 1
	}
	for end < len(s) {
		if s[end] < 0x80 && prev&prepend == 0 {
			break // only a prepended mark joins ASCII
		}
		r2, n2 := utf8.DecodeRuneInString(s[end:])
		if r2 == utf8.RuneError && n2 == 1 {
			break
		}
		f2 := factsOf(r2)
		if !joins(prev, f2, r2, ris, linked) {
			after = decoded{r2, n2}
			break
		}
		switch {
		case f2&prev&regional != 0:
			ris++
		case f2&regional != 0:
			ris = 1
		default:
			ris = 0
		}
		switch {
		case f2&linker != 0:
			linked = true
		case f2&mark == 0:
			linked = false
		}
		prev, end = f2, end+n2
	}
	if end == i+n {
		return end, f.width(), after
	}
	return end, clusterWidth(s[i:end]), after
}

// joins says r, of facts f, continues the cluster whose last code point has
// the facts prev, after ris regional indicators in a row, linked after an
// Indic linker.
func joins(prev, f facts, r rune, ris int, linked bool) bool {
	switch {
	case (prev|f)&control != 0:
		return false // a control stands alone (GB4, GB5)
	case prev&prepend != 0, prev&joiner != 0 && r >= 0x80:
		return true // GB9b; and what the joiner joins (GB11), whatever it is but ASCII
	case f&mark != 0:
		return true // GB9, GB9a
	case prev&f&regional != 0:
		return ris%2 == 1 // a flag's two indicators (GB12, GB13)
	case linked && f&letter != 0:
		return true // an Indic conjunct's consonant (GB9c)
	}
	// A Hangul syllable's jamo (GB6 to GB8).
	return prev&hangulL != 0 && f&(hangulL|hangulV|hangulLV|hangulLVT) != 0 ||
		prev&(hangulLV|hangulV) != 0 && f&(hangulV|hangulT) != 0 ||
		prev&(hangulLVT|hangulT) != 0 && f&hangulT != 0
}

// facts are what the segmentation and the widths read of a code point: its
// width, lipgloss's, in the two low bits, and its properties.
type facts uint32

const (
	known     facts = 1 << (iota + 2) // the facts are filled
	control                           // UAX #29's Control: it stands alone
	mark                              // it continues any cluster: an extending code point, the joiner, a spacing mark
	joiner                            // the zero-width joiner
	prepend                           // it joins the code point after it
	linker                            // a virama that links a conjunct
	letter                            // a letter past U+02FF
	regional                          // a regional indicator, half of a flag
	hangulL                           // a Hangul leading consonant
	hangulV                           // a vowel
	hangulT                           // a trailing consonant
	hangulLV                          // a syllable of two jamo
	hangulLVT                         // a syllable of three
)

func (f facts) width() int { return int(f & 3) }

// factsOf is r's facts, measured once: each frame measures its borders and
// its text again. Those under U+20000 are read without a lock; the program
// draws on one goroutine, and a measure made twice is the same.
func factsOf(r rune) facts {
	if uint32(r) < uint32(len(table)) {
		if f := facts(table[r].Load()); f != 0 {
			return f
		}
	}
	return measured(r)
}

// measured is factsOf's first look at r, and every look past U+1FFFF.
func measured(r rune) facts {
	if uint32(r) < uint32(len(table)) {
		f := measure(r)
		table[r].Store(uint32(f))
		return f
	}
	beyond.Lock()
	defer beyond.Unlock()
	f, ok := beyond.m[r]
	if !ok {
		f = measure(r)
		beyond.m[r] = f
	}
	return f
}

var (
	table  [0x20000]atomic.Uint32
	beyond = struct {
		sync.Mutex
		m map[rune]facts
	}{m: map[rune]facts{}}
)

// measure is r's facts, from lipgloss and Go's Unicode tables. A single
// code point's width is lipgloss's: the soft hyphen and the C1 controls take
// no cell (they took one before plan 49).
func measure(r rune) facts {
	f := known | facts(min(lipgloss.Width(string(r)), 2))
	switch {
	case isControl(r):
		f |= control
	case r == 0x200D:
		f |= mark | joiner
	case isExtend(r) || isSpacingMark(r):
		f |= mark
	}
	if isPrepend(r) {
		f |= prepend
	}
	if isLinker(r) {
		f |= linker
	}
	if r >= 0x300 && unicode.IsLetter(r) {
		f |= letter
	}
	if r >= 0x1F1E6 && r <= 0x1F1FF {
		f |= regional
	}
	switch {
	case r >= 0x1100 && r <= 0x115F, r >= 0xA960 && r <= 0xA97C:
		f |= hangulL
	case r >= 0x1160 && r <= 0x11A7, r >= 0xD7B0 && r <= 0xD7C6:
		f |= hangulV
	case r >= 0x11A8 && r <= 0x11FF, r >= 0xD7CB && r <= 0xD7FB:
		f |= hangulT
	case r >= 0xAC00 && r <= 0xD7A3 && (r-0xAC00)%28 == 0:
		f |= hangulLV
	case r >= 0xAC00 && r <= 0xD7A3:
		f |= hangulLVT
	}
	return f
}

// isControl is UAX #29's Control: the C0 and C1 controls, DEL, the line and
// paragraph separators, and the format characters but the joiners, the
// prepended marks and the tags.
func isControl(r rune) bool {
	switch {
	case r < 0x20 || r >= 0x7f && r < 0xa0 || r == 0x2028 || r == 0x2029:
		return true
	case r < 0xAD || r == 0x200D || r == 0x200C || isPrepend(r) || r >= 0xE0020 && r <= 0xE007F:
		return false
	}
	return unicode.Is(unicode.Cf, r)
}

// isExtend is a code point that extends the cluster before it: a mark (Mn,
// Me), the zero-width non-joiner, the halfwidth sound marks, an emoji
// modifier, a tag. Variation selectors are marks (Mn).
func isExtend(r rune) bool {
	switch {
	case r == 0x200C, r == 0xFF9E, r == 0xFF9F:
		return true
	case r >= 0x1F3FB && r <= 0x1F3FF, r >= 0xE0020 && r <= 0xE007F:
		return true
	}
	return unicode.In(r, unicode.Mn, unicode.Me)
}

// isSpacingMark is a spacing mark (Mc), and the two Lo the standard counts
// with them; those it does not count are kept with their base all the same.
func isSpacingMark(r rune) bool {
	return r == 0x0E33 || r == 0x0EB3 || unicode.Is(unicode.Mc, r)
}

// isPrepend is a code point that joins the one after it (GB9b).
func isPrepend(r rune) bool {
	switch {
	case r >= 0x0600 && r <= 0x0605, r == 0x06DD, r == 0x070F, r == 0x0890, r == 0x0891, r == 0x08E2, r == 0x0D4E:
		return true
	case r == 0x110BD, r == 0x110CD, r == 0x111C2, r == 0x111C3, r == 0x113D1, r == 0x1193F, r == 0x11941,
		r == 0x11A3A, r >= 0x11A84 && r <= 0x11A89, r == 0x11D46, r == 0x11F02:
		return true
	}
	return false
}

// isLinker is a virama that links a conjunct (GB9c's InCB=Linker, Unicode
// 17).
func isLinker(r rune) bool {
	switch r {
	case 0x094D, 0x09CD, 0x0ACD, 0x0B4D, 0x0C4D, 0x0D4D, 0x1039, 0x17D2, 0x1A60, 0x1B44, 0x1BAB, 0xA9C0, 0xAAF6,
		0x10A3F, 0x11133, 0x113D0, 0x1193E, 0x11A47, 0x11A99, 0x11F42:
		return true
	}
	return false
}

// clusterWidth is the width of a cluster of two code points or more:
// lipgloss's, which measures clusters as the terminal does, kept for the
// clusters met again; the cache starts again past maxClusters.
func clusterWidth(c string) int {
	clusters.Lock()
	defer clusters.Unlock()
	if w, ok := clusters.m[c]; ok {
		return w
	}
	if len(clusters.m) >= maxClusters {
		clear(clusters.m)
	}
	w := lipgloss.Width(c)
	clusters.m[c] = w
	return w
}

const maxClusters = 4096

var clusters = struct {
	sync.Mutex
	m map[string]int
}{m: map[string]int{}}
