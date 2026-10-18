package render

import (
	"path/filepath"
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"
)

// The file viewer's highlighting (plan 17, Highlighting): Chroma's lexers,
// coloured from the palette, not from a Chroma theme. Pure computation, run
// by the application in a command, never while drawing.

// SynClass is a syntax colour role (ui-design.md, The file viewer).
type SynClass uint8

const (
	SynText SynClass = iota
	SynKeyword
	SynString
	SynNumber
	SynComment
	SynFunc
	SynBuiltin
	SynTag
	SynAttr
	SynInserted
	SynDeleted
	SynHeading
)

// HlRun is a run of one line's text in one class.
type HlRun struct {
	Text  string
	Class SynClass
}

// Highlighted is lines[From:From+len(Lines)] of a file, coloured. Lang is the
// language Chroma recognised, or "Plain text".
type Highlighted struct {
	Lang  string
	From  int
	Lines [][]HlRun
}

// plainText is the language shown when no lexer matches.
const plainText = "Plain text"

// sniffLines is how many lines pick a lexer when the name does not.
const sniffLines = 200

// classOf maps a Chroma token type to a syntax class.
func classOf(t chroma.TokenType) SynClass {
	switch {
	case t == chroma.KeywordType:
		return SynBuiltin
	case t.InCategory(chroma.Keyword):
		return SynKeyword
	case t.InSubCategory(chroma.LiteralString):
		return SynString
	case t.InSubCategory(chroma.LiteralNumber):
		return SynNumber
	case t.InCategory(chroma.Comment):
		return SynComment
	case t.InSubCategory(chroma.NameFunction), t == chroma.NameClass, t == chroma.NameDecorator:
		return SynFunc
	case t.InSubCategory(chroma.NameBuiltin):
		return SynBuiltin
	case t == chroma.NameTag:
		return SynTag
	case t == chroma.NameAttribute:
		return SynAttr
	case t == chroma.GenericInserted:
		return SynInserted
	case t == chroma.GenericDeleted:
		return SynDeleted
	case t == chroma.GenericHeading, t == chroma.GenericSubheading:
		return SynHeading
	}
	return SynText
}

// pickLexer chooses the lexer by file name, then by content; nil is plain
// text. It depends only on the name and the lines, so windowed calls agree.
func pickLexer(name string, lines []string) chroma.Lexer {
	l := lexers.Match(filepath.Base(strings.ReplaceAll(name, `\`, "/")))
	if l == nil {
		l = lexers.Analyse(strings.Join(lines[:min(len(lines), sniffLines)], "\n"))
	}
	if l == nil || l.Config() == nil {
		return nil
	}
	if n := l.Config().Name; n == "" || n == "plaintext" || n == "fallback" {
		return nil
	}
	return l
}

// Highlight colours lines[from:to] of the file named name (a base name or a
// path). The runs of each line join to the line byte for byte; should
// Chroma's tokens not, the lines are given uncoloured.
func Highlight(name string, lines []string, from, to int) Highlighted {
	from = max(0, min(from, len(lines)))
	to = max(from, min(to, len(lines)))
	out := Highlighted{Lang: plainText, From: from, Lines: make([][]HlRun, to-from)}
	lx := pickLexer(name, lines)
	if lx == nil {
		return uncoloured(out, lines)
	}
	out.Lang = lx.Config().Name
	it, err := chroma.Coalesce(lx).Tokenise(&chroma.TokeniseOptions{State: "root", EnsureLF: true}, strings.Join(lines[from:to], "\n"))
	if err != nil {
		return uncoloured(out, lines)
	}
	n := 0 // the line being filled
	for _, tok := range it.Tokens() {
		c := classOf(tok.Type)
		for i, part := range strings.Split(tok.Value, "\n") {
			if i > 0 {
				n++
			}
			if part == "" {
				continue
			}
			if n >= len(out.Lines) {
				return uncoloured(out, lines)
			}
			l := out.Lines[n]
			if k := len(l); k > 0 && l[k-1].Class == c {
				l[k-1].Text += part
			} else {
				out.Lines[n] = append(l, HlRun{part, c})
			}
		}
	}
	for i, l := range out.Lines {
		var b strings.Builder
		for _, r := range l {
			b.WriteString(r.Text)
		}
		if b.String() != lines[from+i] {
			return uncoloured(out, lines)
		}
	}
	return out
}

// uncoloured gives every line of h as one text run, keeping its language.
func uncoloured(h Highlighted, lines []string) Highlighted {
	for i := range h.Lines {
		h.Lines[i] = []HlRun{{lines[h.From+i], SynText}}
	}
	return h
}
