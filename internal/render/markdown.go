package render

import (
	"strings"

	"charm.land/lipgloss/v2"
)

// AI text as Markdown (ui-design.md, Zone B, Markdown): styles only. Every
// character is drawn as received, markers included, so a row's text, its
// wrap and what selection copies are those of the plain text; the styles
// change, nothing else. Styling works line by line: a line's block kind,
// from its own text and the fences above it, then its inline runs, computed
// only for the rows a frame shows.

// markdownOn draws AI text styled; tests turn it off to compare the text.
var markdownOn = true

// mdKind is a source line's block kind.
type mdKind uint8

const (
	mdPlain mdKind = iota
	mdHeading
	mdQuote
	mdList
	mdTable
	mdRule
	mdFence // an opening or closing fence line
	mdCode  // inside a fence; an unclosed fence runs to the end
)

// mdLine is a line's kind and, for code and fences, its language; comment
// marks a code line that starts inside a block comment (plan 5, item 12).
type mdLine struct {
	kind    mdKind
	lang    string
	comment bool
}

// mdLines classifies an entry's lines; fences, and block comments within a
// fence, carry over from line to line.
func mdLines(lines []string) []mdLine {
	out := make([]mdLine, len(lines))
	var fence string // the open fence's marker, "" outside one
	var lang string
	comment := false // inside a block comment, in the open fence
	for j, l := range lines {
		t := strings.TrimLeft(l, " ")
		indent := len(l) - len(t)
		if fence != "" {
			if indent < 4 && strings.HasPrefix(t, fence) && strings.Trim(strings.TrimRight(t, " \t"), fence[:1]) == "" {
				out[j], fence = mdLine{kind: mdFence, lang: lang}, ""
				continue
			}
			out[j] = mdLine{mdCode, lang, comment}
			if lang != "" {
				comment = codeRuns(nil, l, lang, comment)
			}
			continue
		}
		if indent < 4 {
			if m := fenceMarker(t); m != "" {
				fence, comment = m, false
				lang, _, _ = strings.Cut(strings.TrimSpace(t[len(m):]), " ")
				out[j] = mdLine{kind: mdFence, lang: lang}
				continue
			}
		}
		out[j] = mdLine{kind: blockKind(t, indent)}
	}
	return out
}

// fenceMarker is the run of three or more backticks or tildes t starts
// with, or "".
func fenceMarker(t string) string {
	if len(t) < 3 || (t[0] != '`' && t[0] != '~') {
		return ""
	}
	n := 0
	for n < len(t) && t[n] == t[0] {
		n++
	}
	if n < 3 || t[0] == '`' && strings.IndexByte(t[n:], '`') >= 0 {
		return ""
	}
	return t[:n]
}

func blockKind(t string, indent int) mdKind {
	switch {
	case indent >= 4 || t == "":
		return mdPlain
	case t[0] == '#':
		n := 0
		for n < len(t) && t[n] == '#' {
			n++
		}
		if n <= 6 && (n == len(t) || t[n] == ' ') {
			return mdHeading
		}
	case t[0] == '>':
		return mdQuote
	case t[0] == '|':
		return mdTable
	case isRule(t):
		return mdRule
	}
	if listMarker(t) > 0 {
		return mdList
	}
	return mdPlain
}

// isRule reports a thematic break: three or more of -, * or _, spaces
// between.
func isRule(t string) bool {
	n := 0
	for i := 0; i < len(t); i++ {
		switch {
		case t[i] == t[0] && (t[0] == '-' || t[0] == '*' || t[0] == '_'):
			n++
		case t[i] != ' ':
			return false
		}
	}
	return n >= 3
}

// listMarker is the length of t's list marker with its space ("- ", "12. "),
// or 0.
func listMarker(t string) int {
	if len(t) >= 2 && (t[0] == '-' || t[0] == '*' || t[0] == '+') && t[1] == ' ' {
		return 2
	}
	n := 0
	for n < len(t) && n < 9 && t[n] >= '0' && t[n] <= '9' {
		n++
	}
	if n > 0 && n+1 < len(t) && (t[n] == '.' || t[n] == ')') && t[n+1] == ' ' {
		return n + 2
	}
	return 0
}

// styleRun styles src's bytes up to to, from the previous run's end.
type styleRun struct {
	to int
	st *lipgloss.Style
}

// runs builds a line's style runs; adjacent runs of one style merge.
type runs []styleRun

func (r *runs) add(to int, st *lipgloss.Style) {
	if n := len(*r); n > 0 && ((*r)[n-1].st == st || (*r)[n-1].to >= to) {
		if (*r)[n-1].to < to {
			(*r)[n-1].to = to
		}
		return
	}
	*r = append(*r, styleRun{to, st})
}

// mdRuns styles src, a line of kind k: runs covering its bytes in order.
func mdRuns(src string, k mdLine) []styleRun {
	var r runs
	t := strings.TrimLeft(src, " ")
	at := len(src) - len(t)
	r.add(at, stText)
	switch k.kind {
	case mdHeading:
		n := strings.IndexByte(t, ' ')
		if n < 0 {
			n = len(t)
		}
		r.add(at+n, stMdMark)
		r.add(len(src), stMdHeading)
	case mdRule:
		r.add(len(src), stMdMark)
	case mdFence:
		m := fenceMarker(t)
		if m == "" { // a closing fence
			m = t
		}
		r.add(at+len(m), stMdMark)
		if k.lang != "" && strings.HasPrefix(strings.TrimLeft(t[len(m):], " "), k.lang) {
			i := at + len(m) + strings.Index(t[len(m):], k.lang)
			r.add(i, stMdMark)
			r.add(i+len(k.lang), stMdLang)
		}
		r.add(len(src), stMdMark)
	case mdCode:
		if k.lang == "" {
			r.add(len(src), stMdCode)
		} else {
			codeRuns(&r, src, k.lang, k.comment)
		}
	case mdQuote:
		r.add(at+1, stMdMark)
		inline(&r, src, at+1, stMdEm, false)
	case mdList:
		r.add(at+listMarker(t), stMdBullet)
		inline(&r, src, at+listMarker(t), stText, false)
	case mdTable:
		inline(&r, src, at, stText, true)
	default:
		inline(&r, src, at, stText, false)
	}
	return r
}

// inline styles src from i: code spans, strong and emphasis, links, and in
// a table its pipes. Markers are drawn in grey, never removed.
func inline(r *runs, src string, i int, base *lipgloss.Style, table bool) {
	if table && strings.Trim(src, " |-:") == "" { // the header's separator row
		r.add(len(src), stMdMark)
		return
	}
	for i < len(src) {
		c := src[i]
		switch {
		case c == '|' && table:
			r.add(i+1, stMdMark)
			i++
			continue
		case c == '`':
			n := 1
			for i+n < len(src) && src[i+n] == '`' {
				n++
			}
			if j := strings.Index(src[i+n:], src[i:i+n]); j >= 0 {
				r.add(i+n, stMdMark)
				r.add(i+n+j, stMdCode)
				r.add(i+n+j+n, stMdMark)
				i += n + j + n
				continue
			}
			r.add(i+n, base)
			i += n
			continue
		case (c == '*' || c == '_') && i+1 < len(src) && src[i+1] == c:
			m := src[i : i+2]
			if j := strings.Index(src[i+2:], m); j > 0 && src[i+2] != ' ' && (c == '*' || !wordByte(src, i-1)) {
				r.add(i+2, stMdMark)
				r.add(i+2+j, stMdStrong)
				r.add(i+4+j, stMdMark)
				i += 4 + j
				continue
			}
		case c == '*' || c == '_':
			if i+1 < len(src) && src[i+1] != ' ' && (c == '*' || !wordByte(src, i-1)) {
				if j := strings.IndexByte(src[i+1:], c); j > 0 && src[i+j] != ' ' && (c == '*' || !wordByte(src, i+j+2)) {
					r.add(i+1, stMdMark)
					r.add(i+1+j, stMdEm)
					r.add(i+2+j, stMdMark)
					i += 2 + j
					continue
				}
			}
		case c == '[':
			if j := strings.Index(src[i:], "]("); j > 1 {
				if k := strings.IndexByte(src[i+j+2:], ')'); k >= 0 {
					end := i + j + 2 + k
					r.add(i+1, stMdMark)
					r.add(i+j, stMdLink)
					r.add(i+j+2, stMdMark)
					r.add(end, stMdComment)
					r.add(end+1, stMdMark)
					i = end + 1
					continue
				}
			}
		}
		r.add(i+1, base)
		i++
	}
}

// wordByte reports whether src[i] is a letter or digit; out of range is not.
func wordByte(src string, i int) bool {
	if i < 0 || i >= len(src) {
		return false
	}
	c := src[i]
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c >= 0x80
}

// codeRuns styles a line of code in lang: keywords, strings, comments and
// numbers. A string ends with its line; a block comment runs on to its */
// (plan 5, item 12): in tells that the line starts inside one, and codeRuns
// reports whether it ends inside one. A nil r only scans for that.
func codeRuns(r *runs, src, lang string, in bool) bool {
	hash, dash := hashComment[strings.ToLower(lang)], dashComment[strings.ToLower(lang)]
	add := func(to int, st *lipgloss.Style) {
		if r != nil {
			r.add(to, st)
		}
	}
	i := 0
	if in {
		j := strings.Index(src, "*/")
		if j < 0 {
			add(len(src), stMdComment)
			return true
		}
		i = j + 2
		add(i, stMdComment)
	}
	for i < len(src) {
		c := src[i]
		switch {
		case hash && c == '#',
			dash && strings.HasPrefix(src[i:], "--"),
			!hash && !dash && strings.HasPrefix(src[i:], "//"):
			add(len(src), stMdComment)
			return false
		case !hash && strings.HasPrefix(src[i:], "/*"):
			j := strings.Index(src[i+2:], "*/")
			if j < 0 {
				add(len(src), stMdComment)
				return true
			}
			i += 2 + j + 2
			add(i, stMdComment)
		case c == '"' || c == '\'' || c == '`':
			j := i + 1
			for j < len(src) && src[j] != c {
				if src[j] == '\\' {
					j++
				}
				j++
			}
			end := min(j+1, len(src))
			add(end, stMdString)
			i = end
		case c >= '0' && c <= '9' && !wordByte(src, i-1) && (i == 0 || src[i-1] != '_'):
			j := i
			for j < len(src) && (wordByte(src, j) || src[j] == '.' || src[j] == '_') {
				j++
			}
			add(j, stMdNumber)
			i = j
		case wordByte(src, i) || c == '_':
			j := i
			for j < len(src) && (wordByte(src, j) || src[j] == '_') {
				j++
			}
			st := stMdCode
			if r != nil && keywords[src[i:j]] {
				st = stMdKeyword
			}
			add(j, st)
			i = j
		default:
			add(i+1, stMdCode)
			i++
		}
	}
	return false
}

// Languages whose line comments start with # or --; the others use //.
var (
	hashComment = set("python", "py", "sh", "bash", "shell", "zsh", "console", "yaml", "yml", "toml", "ruby", "rb", "perl", "r", "powershell", "ps1", "pwsh", "dockerfile", "makefile", "make", "ini", "conf", "nix", "elixir")
	dashComment = set("sql", "lua", "haskell", "hs")
)

// keywords are the common ones of the languages an agent writes most; one
// set for all, since a line is short and the cost of a word wrongly
// coloured is a colour.
var keywords = set(
	// Go
	"break", "case", "chan", "const", "continue", "default", "defer", "else", "fallthrough", "for", "func", "go", "goto", "if", "import", "interface", "map", "package", "range", "return", "select", "struct", "switch", "type", "var", "nil", "true", "false",
	// JavaScript, TypeScript, Java, C, C#, Rust
	"async", "await", "class", "catch", "delete", "do", "enum", "export", "extends", "finally", "from", "function", "implements", "in", "instanceof", "let", "new", "null", "of", "private", "protected", "public", "static", "super", "this", "throw", "throws", "try", "typeof", "undefined", "void", "while", "yield", "abstract", "final", "int", "long", "double", "float", "boolean", "char", "string", "bool", "fn", "impl", "mut", "pub", "use", "mod", "match", "trait", "where", "self", "Self", "crate", "struct", "unsafe", "namespace", "using", "readonly", "override", "virtual",
	// Python, shell, SQL
	"def", "elif", "except", "lambda", "pass", "raise", "with", "as", "is", "not", "and", "or", "None", "True", "False", "global", "nonlocal", "assert", "then", "fi", "esac", "done", "echo", "local", "SELECT", "FROM", "WHERE", "INSERT", "UPDATE", "DELETE", "JOIN", "ORDER", "BY", "GROUP", "CREATE", "TABLE",
)

func set(words ...string) map[string]bool {
	m := make(map[string]bool, len(words))
	for _, w := range words {
		m[w] = true
	}
	return m
}
