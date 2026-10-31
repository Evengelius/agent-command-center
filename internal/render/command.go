package render

import (
	"encoding/json"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"
)

// The commands the CLIs run (plan 72, D1 to D4 and D8; ui-design.md, Zone
// B, Commands): the shell a command runs in, the command shown, out of
// Codex's launcher when that is exact, its layout for display only, and its
// colours, made off the draw path. The text shown is never changed: the
// layout breaks it into rows, and the copy takes its bytes.

// cmdIn is what an input says of a command, read with its other facts
// (Memo.input): the command field as the CLI sent it, the working folder and
// the reason sent with it, and, for a known launcher's invocation, the
// launcher's shell and the script it runs, "" when the invocation is not
// undone.
type cmdIn struct {
	sent, cwd, reason string
	shell, script     string
}

// readCmd reads in's command field, a string; zero for none.
func readCmd(in json.RawMessage) cmdIn {
	var c struct {
		Command json.RawMessage `json:"command"`
		Cwd     string          `json:"cwd"`
		Reason  string          `json:"reason"`
	}
	if len(in) == 0 || json.Unmarshal(in, &c) != nil {
		return cmdIn{}
	}
	var sent string
	if json.Unmarshal(c.Command, &sent) != nil || sent == "" {
		return cmdIn{}
	}
	out := cmdIn{sent: sent, cwd: c.Cwd, reason: c.Reason}
	out.shell, out.script = unwrap(sent)
	return out
}

// unwrap undoes a launcher's invocation (plan 72, D2). Codex's command
// field is its argv joined with POSIX shell quoting: single quotes, double
// quotes with a backslash escaped, and both in turn within one word (V1:
// every recorded invocation, and Codex's own parse of each where it gives
// one). The field split back into its words, with nothing left over, is
// that argv exactly; when the words are a known launcher's, PowerShell's
// [exe, flags…, -Command, script], bash, sh or zsh's [exe, -c or -lc,
// script] or cmd's [exe, /c, script], the script is the command. shell is
// the executable's shell even when the words are no launcher's; script ""
// then, the invocation shown whole.
func unwrap(sent string) (shell, script string) {
	words, ok := shellWords(sent)
	if !ok || len(words) < 3 {
		if len(words) > 0 { // its executable, its first word, read whole
			return exeShell(words[0]), ""
		}
		return "", ""
	}
	shell = exeShell(words[0])
	n := len(words)
	switch shell {
	case "PowerShell":
		if !strings.EqualFold(words[n-2], "-Command") && !strings.EqualFold(words[n-2], "-c") {
			return shell, ""
		}
		for _, f := range words[1 : n-2] {
			if !strings.HasPrefix(f, "-") || strings.EqualFold(f, "-Command") || strings.EqualFold(f, "-c") {
				return shell, ""
			}
		}
		return shell, words[n-1]
	case "bash", "sh", "zsh":
		if n == 3 && (words[1] == "-c" || words[1] == "-lc") {
			return shell, words[2]
		}
	case "cmd":
		if n == 3 && strings.EqualFold(words[1], "/c") {
			return shell, words[2]
		}
	}
	return shell, ""
}

// shellWords splits s into its words by POSIX shell quoting: blanks between
// words; '…' literal; "…" literal but for a backslash before $ ` " \ or a
// newline; outside quotes, a backslash quoting the next byte. ok is false for
// a quote not closed, or for anything a shell would read as more than a
// word: an expansion in double quotes ($ or ` not escaped), or outside
// quotes a byte other than a letter, a digit or one of _ - . / : @ % + , =
// and the non-ASCII ones; words are then those read whole before it.
func shellWords(s string) ([]string, bool) {
	var words []string
	var cur strings.Builder
	in := false
	for i := 0; i < len(s); i++ {
		switch c := s[i]; {
		case c == ' ' || c == '\t':
			if in {
				words = append(words, cur.String())
				cur.Reset()
				in = false
			}
		case c == '\'':
			j := strings.IndexByte(s[i+1:], '\'')
			if j < 0 {
				return words, false
			}
			cur.WriteString(s[i+1 : i+1+j])
			i, in = i+1+j, true
		case c == '"':
			for i++; ; i++ {
				if i >= len(s) {
					return words, false
				}
				b := s[i]
				if b == '"' {
					break
				}
				if b == '$' || b == '`' {
					return words, false
				}
				if b == '\\' && i+1 < len(s) && strings.IndexByte("$`\"\\\n", s[i+1]) >= 0 {
					if i++; s[i] == '\n' {
						continue // a line continued: neither byte is the word's
					}
					b = s[i]
				}
				cur.WriteByte(b)
			}
			in = true
		case c == '\\':
			if i+1 >= len(s) {
				return words, false
			}
			if i++; s[i] != '\n' { // a line continued: neither byte is the word's
				cur.WriteByte(s[i])
				in = true
			}
		case c >= 0x80 || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || strings.IndexByte("_-./:@%+,=", c) >= 0:
			cur.WriteByte(c)
			in = true
		default:
			return words, false
		}
	}
	if in {
		words = append(words, cur.String())
	}
	return words, true
}

// exeShell is the shell an executable runs, by its file's name: PowerShell
// for powershell and pwsh, bash, sh, zsh and cmd by theirs; "" for any other.
func exeShell(exe string) string {
	base := strings.ToLower(exe[strings.LastIndexAny(exe, `\/`)+1:])
	base = strings.TrimSuffix(base, ".exe")
	switch base {
	case "powershell", "pwsh":
		return "PowerShell"
	case "bash", "sh", "zsh", "cmd":
		return base
	}
	return ""
}

// command is what a call or a request shows of its command (plan 72, D1,
// D2); ok is false for a tool that runs none.
type command struct {
	ok        bool
	shell     string // the name shown: Claude's tool, the launcher's shell, else "command"
	lexer     string // Chroma's lexer for text; "" for plain text
	rules     string // the shell whose quoting text follows, for its layout; "" POSIX's
	text      string // the command shown
	sent      string // the command field as the CLI sent it
	cwd       string // the working folder sent with it; "" for none
	reason    string // Codex's reason sent with a request; "" for none
	unwrapped bool   // text is the script out of Codex's launcher, sent its invocation
}

// commandOf is the command tool name runs from c, its input's.
func commandOf(name string, c cmdIn) command {
	if c.sent == "" {
		return command{}
	}
	out := command{ok: true, text: c.sent, sent: c.sent, cwd: c.cwd, reason: c.reason}
	switch name {
	case "Bash":
		out.shell, out.lexer, out.rules = "Bash", "bash", "bash"
	case "PowerShell":
		out.shell, out.lexer, out.rules = "PowerShell", "powershell", "PowerShell"
	case "commandExecution":
		// Codex's invocation whole is its argv joined with POSIX quoting.
		out.shell = c.shell
		if out.shell == "" {
			out.shell = "command"
		}
		if c.script != "" {
			out.text, out.unwrapped, out.lexer, out.rules = c.script, true, lexerFor(c.shell), c.shell
		}
	default:
		return command{}
	}
	return out
}

// cmdFacts is a command's facts for the rows that show it, kept with its
// input's (Memo.command): the command, for the tool it was read for, its
// width drawn, and its layouts, by width, the last maxLays (plan 72, P2).
type cmdFacts struct {
	in    cmdIn
	name  string
	named bool
	c     command
	cells int
	lays  []cmdLay
}

// cmdLay is a command's layout at a width.
type cmdLay struct {
	w    int
	rows []cmdRow
}

// maxLays is how many widths a command keeps its layout at: zone B's, a
// card's, a full view's, and one a resize left.
const maxLays = 4

// of is the command tool name runs from the facts' input.
func (f *cmdFacts) of(name string) command {
	if !f.named || f.name != name {
		f.name, f.named, f.c, f.lays = name, true, commandOf(name, f.in), nil
		f.cells = cellsOf(f.c.text, 0, len(f.c.text))
	}
	return f.c
}

// layout is the command's layout w cells wide, laid out once a width.
func (f *cmdFacts) layout(w int) []cmdRow {
	for _, l := range f.lays {
		if l.w == w {
			return l.rows
		}
	}
	rows := cmdLayout(f.c.text, f.c.rules, w)
	if len(f.lays) == maxLays {
		f.lays = append(f.lays[:0], f.lays[1:]...)
	}
	f.lays = append(f.lays, cmdLay{w, rows})
	return rows
}

// want is the command's colours to ask for.
func (f *cmdFacts) want() CmdWant { return CmdWant{f.c.lexer, f.c.text} }

// lexerFor is Chroma's lexer for a shell's scripts; "" for none.
func lexerFor(shell string) string {
	switch shell {
	case "PowerShell":
		return "powershell"
	case "bash", "sh", "zsh":
		return "bash"
	case "cmd":
		return "batch"
	}
	return ""
}

// cmdRow is a row of a command's layout: its text's bytes [from, to), ind
// cells in (plan 72, D3).
type cmdRow struct{ from, to, ind int }

// cmdLayout lays text out in rows at most w cells wide, for display only
// (plan 72, D3). One command a row: a break after each top-level ; (and cmd's
// &) and before each top-level && and ||, that row 2 cells in; a row still
// too wide breaks before top-level |, as few times as needed, filled from
// the left, 2 cells further in; a row still too wide wraps at a blank outside
// quotes, a top-level one first, 4 cells further in, else at the width. Top
// level: outside quotes, (), {} and []. The command's own line breaks stay;
// the blanks at a break are left out of both rows. Joined, the rows and the
// bytes between them give text back.
func cmdLayout(text, shell string, w int) []cmdRow {
	w = max(w, 8)
	marks := scanCmd(text, shell)
	var rows []cmdRow
	for ls := 0; ls <= len(text); {
		le := strings.IndexByte(text[ls:], '\n')
		if le < 0 {
			le = len(text)
		} else {
			le += ls
		}
		rows = layLine(rows, text, marks, ls, le, w)
		ls = le + 1
	}
	return rows
}

// Marks of a command's bytes, scanCmd's.
const (
	markTop  uint8 = 1 << iota // outside quotes and brackets
	markOut                    // outside quotes
	markStop                   // a statement ends after it: ; or cmd's &
	markAnd                    // && or || begins at it
	markPipe                   // a single | at it
)

// scanCmd marks text's bytes by the shell's quoting: ' and " quote, a
// backslash (a backtick in PowerShell, a caret in cmd) escapes the next byte
// outside single quotes, ( { [ and ) } ] nest.
func scanCmd(text, shell string) []uint8 {
	esc := byte('\\')
	switch shell {
	case "PowerShell":
		esc = '`'
	case "cmd":
		esc = '^'
	}
	marks := make([]uint8, len(text))
	q, d := byte(0), 0
	for i := 0; i < len(text); i++ {
		c := text[i]
		switch {
		case q == '\'':
			if c == '\'' {
				q = 0
			}
			continue
		case q == '"':
			if c == esc && shell != "cmd" && i+1 < len(text) {
				i++
			} else if c == '"' {
				q = 0
			}
			continue
		}
		marks[i] = markOut
		if d == 0 {
			marks[i] |= markTop
		}
		switch c {
		case esc:
			if i+1 < len(text) {
				i++
				marks[i] = markOut
			}
		case '\'', '"':
			q = c
		case '(', '{', '[':
			d++
		case ')', '}', ']':
			d = max(d-1, 0)
		case ';':
			if d == 0 {
				marks[i] |= markStop
			}
		case '&', '|':
			switch {
			case d > 0:
			case i+1 < len(text) && text[i+1] == c:
				marks[i] |= markAnd
				i++
				marks[i] = markOut | markTop
			case c == '&' && shell == "cmd":
				marks[i] |= markStop
			case c == '|':
				marks[i] |= markPipe
			}
		}
	}
	return marks
}

// layLine lays out text's line [ls, le), its marks those of scanCmd.
func layLine(rows []cmdRow, text string, marks []uint8, ls, le, w int) []cmdRow {
	if ls == le {
		return append(rows, cmdRow{ls, le, 0})
	}
	// Its commands: [a, b) each, ind cells in.
	a, ind := ls, 0
	part := func(b int) {
		from, to := trimBlanks(text, a, b, a != ls)
		if from < to || a == ls {
			rows = layPart(rows, text, marks, from, to, ind, w)
		}
	}
	for i := ls; i < le; i++ {
		switch m := marks[i]; {
		case m&markStop != 0:
			part(i + 1)
			a, ind = i+1, 0
		case m&markAnd != 0 && i > a:
			part(i)
			a, ind = i, 2
		}
	}
	part(le)
	return rows
}

// indent is ind cells, as many as leave a row minRoom cells of text.
func indent(ind, w int) int { return min(ind, max(w-minRoom, 0)) }

// minRoom is the least a row keeps for its text when its layout indents it.
const minRoom = 8

// trimBlanks is [a, b) without its blanks at the end, and at its start when
// lead: a line's own indent stays.
func trimBlanks(text string, a, b int, lead bool) (int, int) {
	for lead && a < b && (text[a] == ' ' || text[a] == '\t') {
		a++
	}
	for b > a && (text[b-1] == ' ' || text[b-1] == '\t') {
		b--
	}
	return a, b
}

// layPart lays out one command [a, b), ind cells in: whole when it fits,
// else broken before its pipes, else wrapped.
func layPart(rows []cmdRow, text string, marks []uint8, a, b, ind, w int) []cmdRow {
	ind = indent(ind, w)
	if ind+cellsOf(text, a, b) <= w {
		return append(rows, cmdRow{a, b, ind})
	}
	var stages []int // where each stage after the first starts: its |
	for i := a + 1; i < b; i++ {
		if marks[i]&markPipe != 0 {
			stages = append(stages, i)
		}
	}
	cur, at := a, ind
	for _, p := range stages {
		next := indent(ind+2, w)
		_, end := trimBlanks(text, cur, p, false)
		if at+cellsOf(text, cur, nextStageEnd(stages, p, b, text)) <= w {
			continue // the next stage still fits on this row
		}
		rows = wrapPart(rows, text, marks, cur, end, at, w)
		cur, at = p, next
	}
	return wrapPart(rows, text, marks, cur, b, at, w)
}

// nextStageEnd is where the stage starting at p ends: before the next |, its
// blanks left out, or b.
func nextStageEnd(stages []int, p, b int, text string) int {
	for _, q := range stages {
		if q > p {
			_, e := trimBlanks(text, p, q, false)
			return e
		}
	}
	return b
}

// wrapPart adds [a, b) ind cells in, wrapped at its blanks outside quotes, a
// top-level one first, its next rows 4 cells further in; a part with no
// blank where it must break, at the width.
func wrapPart(rows []cmdRow, text string, marks []uint8, a, b, ind, w int) []cmdRow {
	for at := indent(ind, w); a < b; at = indent(ind+4, w) {
		room := max(w-at, 1)
		if cellsOf(text, a, b) <= room {
			return append(rows, cmdRow{a, b, at})
		}
		// The furthest blank that leaves the row within its room, a top-level
		// one when there is one, and the byte the width ends at.
		top, out, cells, end := -1, -1, 0, a
		for i := a; i < b; {
			n, cw := cmdCellSpan(text, i)
			if cells+cw > room {
				break
			}
			if text[i] == ' ' && i > a {
				if marks[i]&markTop != 0 {
					top = i
				}
				if marks[i]&markOut != 0 {
					out = i
				}
			}
			cells, i = cells+cw, i+n
			end = i
		}
		cut := end
		switch {
		case top > a:
			cut = top
		case out > a:
			cut = out
		case end == a: // not even one unit fits: it takes the row alone
			n, _ := cmdCellSpan(text, a)
			cut = a + n
		}
		rows = append(rows, cmdRow{a, cut, at})
		a, _ = trimBlanks(text, cut, b, true)
	}
	return rows
}

// cellsOf is how many cells text's bytes [a, b) take, drawn.
func cellsOf(text string, a, b int) int {
	n := 0
	for i := a; i < b; {
		k, w := cmdCellSpan(text, i)
		n, i = n+w, i+k
	}
	return n
}

// CmdWant is a command a frame drew in plain text, its colours not made
// yet: its lexer and its text (plan 72, D8, P3).
type CmdWant struct{ Lexer, Text string }

// hlSpan is a part of a command's text in one syntax class.
type hlSpan struct {
	from, to int
	class    SynClass
}

// Coloured is a command's colours as ColourCommand makes them: ok when its
// tokens join back into its text byte for byte, else none, the text plain.
type Coloured struct {
	want  CmdWant
	spans []hlSpan
	ok    bool
}

// ColourCommand colours a command with Chroma's lexer, from the file
// viewer's classes (plan 72, D8): pure computation, run by the application
// in a command, never while drawing (P3). A token stream that does not give
// the text back byte for byte leaves it plain.
func ColourCommand(w CmdWant) Coloured {
	out := Coloured{want: w}
	lx := lexers.Get(w.Lexer)
	if lx == nil || w.Text == "" {
		return out
	}
	it, err := chroma.Coalesce(lx).Tokenise(&chroma.TokeniseOptions{State: "root", EnsureLF: true}, w.Text)
	if err != nil {
		return out
	}
	off := 0
	var spans []hlSpan
	for _, tok := range it.Tokens() {
		v := tok.Value
		if off == len(w.Text) && v == "\n" {
			continue // EnsureLF's own
		}
		if !strings.HasPrefix(w.Text[off:], v) {
			return out
		}
		if c := classOf(tok.Type); c != SynText && v != "" {
			if k := len(spans); k > 0 && spans[k-1].class == c && spans[k-1].to == off {
				spans[k-1].to = off + len(v)
			} else {
				spans = append(spans, hlSpan{off, off + len(v), c})
			}
		}
		off += len(v)
	}
	if off != len(w.Text) {
		return out
	}
	out.spans, out.ok = spans, true
	return out
}

// CmdColors keeps commands' colours by lexer and text, made off the draw
// path (plan 72, P3): the application asks for each command as its call or
// request comes (Ask), and for any a frame drew plain (Wants), and Put keeps
// what ColourCommand made; until then, a command is drawn plain, in the
// same layout. Two generations of maxCmdColors/2 commands each, the older
// dropped when the newer fills, a command used moved to the newer: a memo,
// never a source of what is drawn. It lives on the program's event loop.
type CmdColors struct {
	got, old map[CmdWant][]hlSpan
	wants    map[CmdWant]bool // drawn plain, not asked for yet
	asked    map[CmdWant]bool // asked for, not put yet
}

const maxCmdColors = 2048

// NewCmdColors is an empty store.
func NewCmdColors() *CmdColors { return &CmdColors{} }

// colours is w's colours and whether they are made, plain text's always; a
// miss is wanted.
func (c *CmdColors) colours(w CmdWant) ([]hlSpan, bool) {
	if w.Lexer == "" || w.Text == "" {
		return nil, true
	}
	if c == nil {
		return nil, false
	}
	if sp, ok := c.got[w]; ok {
		return sp, true
	}
	if sp, ok := c.old[w]; ok {
		c.keep(w, sp)
		return sp, true
	}
	if !c.asked[w] {
		if c.wants == nil {
			c.wants = map[CmdWant]bool{}
		}
		c.wants[w] = true
	}
	return nil, false
}

// Ask is ws less those made or asked for already, now asked for, in their
// order.
func (c *CmdColors) Ask(ws []CmdWant) []CmdWant {
	if c == nil {
		return nil
	}
	var out []CmdWant
	for _, w := range ws {
		if w.Lexer == "" || w.Text == "" || c.asked[w] {
			continue
		}
		if _, ok := c.got[w]; ok {
			continue
		}
		if _, ok := c.old[w]; ok {
			continue
		}
		if c.asked == nil {
			c.asked = map[CmdWant]bool{}
		}
		c.asked[w] = true
		out = append(out, w)
	}
	return out
}

// Wants takes the commands frames drew plain since the last call, now asked
// for.
func (c *CmdColors) Wants() []CmdWant {
	if c == nil || len(c.wants) == 0 {
		return nil
	}
	ws := make([]CmdWant, 0, len(c.wants))
	for w := range c.wants {
		ws = append(ws, w)
	}
	c.wants = nil
	return c.Ask(ws)
}

// Put keeps a command's colours.
func (c *CmdColors) Put(x Coloured) {
	if c == nil {
		return
	}
	delete(c.asked, x.want)
	c.keep(x.want, x.spans)
}

// keep puts w's colours in the newer generation.
func (c *CmdColors) keep(w CmdWant, sp []hlSpan) {
	if c.got == nil || len(c.got) >= maxCmdColors/2 {
		c.old, c.got = c.got, make(map[CmdWant][]hlSpan, maxCmdColors/2)
	}
	c.got[w] = sp
}

// cmdRuns are a command's style runs over its whole text: its colours'
// classes, base st elsewhere.
func cmdRuns(text string, spans []hlSpan, st *lipgloss.Style) []styleRun {
	if len(spans) == 0 {
		return nil
	}
	out := make([]styleRun, 0, 2*len(spans)+1)
	at := 0
	for _, s := range spans {
		if s.from > at {
			out = append(out, styleRun{s.from, st})
		}
		out = append(out, styleRun{s.to, s.class.style()})
		at = s.to
	}
	if at < len(text) {
		out = append(out, styleRun{len(text), st})
	}
	return out
}

// cmdSpans is row r of a command's text drawn in st, coloured by runs, each
// span's reference at's with its offset in the whole text. In zone B a
// command is the search's alone, never copied (plan 72, C1): its spans are
// targets, a tab or a control character a cut's unit for its byte. In a
// full view it is the copy's (C2): text and units as the source's.
func cmdSpans(text string, r cmdRow, at ref, target bool, st *lipgloss.Style, runs []styleRun) line {
	var l line
	if r.from == r.to {
		a := at
		a.kind, a.off = refText, r.from
		l = line{{st: st, ref: a}} // an empty line still takes a click
	} else {
		l = cellSpans(text, cmdCells(text, r.from, r.to), at, st, runs)
	}
	if !target {
		return l
	}
	for k := range l {
		switch x := &l[k].ref; x.kind {
		case refText:
			x.kind = refTarget
		case refAtom:
			x.kind = refTargetCut
		}
	}
	return l
}
