package render

import (
	"fmt"
	"slices"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/Evengelius/agent-command-center/internal/state"
)

// The conversation's search (plan 66, brief amendment 68): Ctrl+F's bar at
// the foot of zone B finds a query in what B holds as text, the newest
// match first; ui-design.md, Zone B, Search.

// ConvMatch is one match of a search of the conversation (plan 66, design
// D3): Item the entry, Line its line as entryLines gives them, Off and N
// its bytes in that line. A match in a tool row's or an approval's target
// has Target set: Off is then in the target as its row writes it, before
// the row cuts it to its room, and Line is 0, the row's own.
type ConvMatch struct {
	Item, Line, Off, N int
	Target             bool
}

// SearchMemo keeps what a search reads but the state holds as no text:
// each tool call's target and each approval's, read from its input once
// while the bar is open (plan 66, design D4: the 4,800 calls of a two-hour
// session take 5 to 9 ms to read, too long for every keystroke). The
// application makes one when the bar opens and drops it when it closes:
// one string a call or a request.
type SearchMemo struct{ targets, approvals map[string]string }

// target is tc's target as its row writes it before the cut; "" for none.
func (m *SearchMemo) target(tc *state.ToolCall) string {
	if m == nil {
		d, _ := toolDetail(tc)
		return d
	}
	if d, ok := m.targets[tc.ID]; ok {
		return d
	}
	if m.targets == nil {
		m.targets = map[string]string{}
	}
	d, _ := toolDetail(tc)
	m.targets[tc.ID] = d
	return d
}

// approvalTarget is ap's target as its row, or its box's target row,
// writes it before the cut (approvalDetail's, its format characters drawn:
// plan 48, design D6); "" for none.
func (m *SearchMemo) approvalTarget(ap *state.Approval) string {
	read := func() string {
		d, _, _ := approvalDetail(ap)
		return revealed(d)
	}
	if m == nil {
		return read()
	}
	if d, ok := m.approvals[ap.ID]; ok {
		return d
	}
	if m.approvals == nil {
		m.approvals = map[string]string{}
	}
	d := read()
	m.approvals[ap.ID] = d
	return d
}

// ConvMatches is every match of q in the conversation, the newest first,
// MaxMatches at most, and whether it stopped there (plan 66, designs D3 and
// D4): the query as typed, its case ignored unless it has a capital, as the
// file viewer's search does. It reads the state as it is, no copy of its
// text: entries change under Update, and a command reading them would race.
func ConvMatches(s *state.State, ui UI, q string, m *SearchMemo) ([]ConvMatch, bool) {
	if q == "" {
		return nil, false
	}
	fold := q == strings.ToLower(q)
	var out, buf []ConvMatch
	for i := len(s.Conversation) - 1; i >= 0; i-- {
		e := &s.Conversation[i] // by reference: every keystroke reads them all (plan 49, design D5)
		if !searchable(ui, e) {
			continue
		}
		buf = buf[:0]
		entryParts(s, e, m, func(line int, target bool, text string) {
			findIn(text, q, fold, func(off int) {
				buf = append(buf, ConvMatch{Item: i, Line: line, Off: off, N: len(q), Target: target})
			})
		})
		for k := len(buf) - 1; k >= 0; k-- { // the entry's newest first
			if len(out) == MaxMatches {
				return out, true
			}
			out = append(out, buf[k])
		}
	}
	return out, false
}

// searchable says whether B draws entry e, or folds it in a run, the
// entries a search goes through (design D3): not an assistant entry holding
// only a thinking mark, which B never draws, nor a thinking phase B shows
// as no row or as its live card (lookAll's hidden ones and the card).
func searchable(ui UI, e *state.Entry) bool {
	if e.Role == "thinking" {
		st := thoughtStage(ui, *e)
		return st != thinkHidden && st != thinkLive
	}
	return !thinkingMark(e)
}

// entryParts calls f with each part of e a search reads, in the order its
// rows draw them: its lines (entryLines'), a tool row's target between its
// name and its error, an approval's after its name. A message's lines are
// read in place, never split.
func entryParts(s *state.State, e *state.Entry, m *SearchMemo, f func(line int, target bool, text string)) {
	switch e.Role {
	case "tool":
		name, _, _ := strings.Cut(e.Text, "\n")
		f(0, false, name)
		if tc := s.Tools[e.MessageID]; tc != nil {
			if d := m.target(tc); d != "" {
				f(0, true, d)
			}
		}
		if err := toolError(s, *e); err != "" {
			f(1, false, err)
		}
	case "approval":
		for j, l := range entryLines(s, *e) {
			f(j, false, l)
		}
		if ap := s.Approvals[e.MessageID]; ap != nil {
			if d := m.approvalTarget(ap); d != "" {
				f(0, true, d)
			}
		}
	case "thinking", "error":
		for j, l := range entryLines(s, *e) {
			f(j, false, l)
		}
	default:
		for j, text := 0, e.Text; ; j++ {
			l, rest, more := strings.Cut(text, "\n")
			f(j, false, l)
			if !more {
				return
			}
			text = rest
		}
	}
}

// findIn calls emit with the offset of each match of q in l, left to
// right, none overlapping the one before (plan 66, design D4). With fold,
// the case is ignored: each place q's first byte starts in either case,
// for an ASCII letter, or as it is, is compared with strings.EqualFold over
// q's length, so a letter whose two cases begin with the same byte (é and
// É) matches. Each case's next place is kept, so a line full of one letter
// is read once.
func findIn(l, q string, fold bool, emit func(off int)) {
	if len(q) == 0 || len(q) > len(l) {
		return
	}
	if !fold {
		for at := 0; at+len(q) <= len(l); {
			k := strings.Index(l[at:], q)
			if k < 0 {
				return
			}
			emit(at + k)
			at += k + len(q)
		}
		return
	}
	lo, up := q[0], q[0]
	if lo >= 'a' && lo <= 'z' {
		up = lo - 'a' + 'A'
	}
	next := func(b byte, at int) int {
		if k := strings.IndexByte(l[at:], b); k >= 0 {
			return at + k
		}
		return len(l)
	}
	nl, nu := -1, len(l) // the next place of each case; -1 to look for, len(l) none
	if up != lo {
		nu = -1
	}
	for at := 0; at+len(q) <= len(l); {
		if nl < at {
			nl = next(lo, at)
		}
		if up != lo && nu < at {
			nu = next(up, at)
		}
		p := min(nl, nu)
		if p+len(q) > len(l) {
			return
		}
		if strings.EqualFold(l[p:p+len(q)], q) {
			emit(p)
			at = p + len(q)
			continue
		}
		at = p + 1
	}
}

// SearchView is the conversation's search bar (plan 66, design D1): open,
// its query and the field's cursor, a byte offset in it, the matches found
// for it, the newest first, the current one (an index in Matches, -1 for
// none), whether the cap left older ones out, and the targets' memo.
type SearchView struct {
	Open    bool
	Query   string
	Cursor  int
	Matches []ConvMatch
	Cur     int
	Capped  bool
	Memo    *SearchMemo
}

// searchFooter is the bottom row's hints while the bar is open, their
// directions spelled out (plan 66, design D1).
const searchFooter = "↑ Enter older · ↓ Shift+Enter newer · Tab whole · Esc close · F1 keys"

// searchBar is the bar's row, w cells wide (plan 66, design D1): "/ " in the
// title colour, the query with its cursor, and at the right where the search
// stands. A long query keeps its end; the scope's note goes before the count
// does when the row is short.
func searchBar(v SearchView, w int) line {
	q, c := v.Query, min(max(v.Cursor, 0), len(v.Query))
	var right line
	switch {
	case q == "":
		right = decoLine("type to search the conversation", stGrey)
	case len(v.Matches) == 0:
		right = line{deco("no match", stAttention), deco(" · calls' inputs and outputs not searched", stGrey)}
	case v.Capped:
		right = decoLine(fmt.Sprintf("match %s of the newest %s", grouped(v.Cur+1), grouped(len(v.Matches))), stProgress)
	default:
		right = decoLine(fmt.Sprintf("match %s of %s", grouped(v.Cur+1), grouped(len(v.Matches))), stProgress)
	}
	field := line{deco(q[:c], stText), deco("▌", stText), deco(q[c:], stText)}
	lead := line{deco(" ", stNone), deco("/", stTitle), deco(" ", stNone)}
	room := w - lead.width() - right.width() - 2
	if room < 8 && len(right) > 1 {
		right = right[:1] // "no match" alone
		room = w - lead.width() - right.width() - 2
	}
	if f := field.width(); f > room && room > 1 {
		field = tailFit(field, room)
	}
	return spread(append(lead, field...), append(right, deco(" ", stNone)), w)
}

// tailFit keeps the end of l in w cells, "…" in place of its start.
func tailFit(l line, w int) line {
	var text strings.Builder
	for _, x := range l {
		text.WriteString(x.text)
	}
	s := tailCut(text.String(), max(w-1, 0))
	return line{deco("…", stGrey), deco(s, stText)}
}

// SearchPos is a place of the conversation a search compares its matches
// with (plan 66, design D5): the entry, its line and a byte in it; Line
// AllLines stands for the whole entry.
type SearchPos struct{ Item, Line, Off int }

// AllLines is a SearchPos's Line for a row that shows its whole entry: a
// tool row, an approval's or an error's, a long paste's fold row.
const AllLines = 1 << 30

// AtOrBefore says match m is at or before place p, in the conversation's
// order.
func (m ConvMatch) AtOrBefore(p SearchPos) bool {
	switch {
	case m.Item != p.Item:
		return m.Item < p.Item
	case p.Line == AllLines:
		return true
	case m.Line != p.Line:
		return m.Line < p.Line
	}
	return m.Off <= p.Off
}

// rowEnd is the place up to which row c, of entry i in role role, shows its
// entry, its last byte shown; ok is false for a row that shows none of it
// (a label, a blank).
func rowEnd(c crow, role string) (SearchPos, bool) {
	if c.entry == 0 {
		return SearchPos{}, false
	}
	i := c.entry - 1
	switch {
	case c.src != "":
		return SearchPos{i, c.line, c.r.to - 1}, true // a match from c.r.to on starts on the next row
	case c.more:
		return SearchPos{i, AllLines, 0}, true // a long paste's fold row: its folded lines
	}
	switch role {
	case "user", "assistant", "notice", "dashboard", "review":
		return SearchPos{}, false // a label or a blank row of a message
	}
	return SearchPos{i, AllLines, 0}, true
}

// MatchFolds is the folds a match is behind (plan 66, design D5): its run of
// calls when B folds its entry away, a phase's details when the match is in
// them; the keys of ui.Unfolded to open.
func MatchFolds(s *state.State, ui UI, m ConvMatch) []string {
	if m.Item < 0 || m.Item >= len(s.Conversation) {
		return nil
	}
	var keys []string
	lk, _ := lookAll(s, ui)
	if k := lk[m.Item]; k.hidden && k.run != "" && !k.open {
		keys = append(keys, k.run)
	}
	if e := s.Conversation[m.Item]; e.Role == "thinking" && m.Line > 0 && !ui.Unfolded[ThoughtKey(e)] {
		keys = append(keys, ThoughtKey(e))
	}
	return keys
}

// ConvMatchRow is the conversation's row match m is on, w cells wide with
// room rows to show them, and the conversation's rows, for ui as it is (its
// folds open): the row of its line holding its offset; a tool row, an
// approval's or an error's own row; a long paste's fold row for its folded
// lines (plan 66, design D5). ok is false when B draws none of its entry.
func ConvMatchRow(s *state.State, ui UI, w, room int, m ConvMatch) (row, rows int, ok bool) {
	if m.Item < 0 || m.Item >= len(s.Conversation) {
		return 0, 0, false
	}
	lk, _ := lookAll(s, ui)
	starts, rows := convStarts(s, ui, w, room, lk)
	e := s.Conversation[m.Item]
	whole, last := -1, -1 // the entry's first row showing it whole; the match's line's last row
	for j, c := range entryRows(s, ui, nil, m.Item, e, w, lk[m.Item]) {
		if c.entry != m.Item+1 {
			continue
		}
		switch {
		case c.src != "" && !m.Target && c.line == m.Line:
			if m.Off < max(c.r.to, c.r.from+1) {
				return starts[m.Item] + j, rows, true
			}
			last = j // should the offset pass the line's last row
		case c.more && !m.Target && m.Line >= foldShow:
			return starts[m.Item] + j, rows, true
		case whole < 0:
			if p, ok := rowEnd(c, e.Role); ok && p.Line == AllLines {
				whole = j
			}
		}
	}
	switch {
	case last >= 0:
		return starts[m.Item] + last, rows, true
	case whole >= 0:
		return starts[m.Item] + whole, rows, true
	}
	return 0, rows, false
}

// EntryViewRow is the full view's body row of match m (plan 66, design D8),
// counted as entryViewer wraps the entry's lines, ui.Width less its margin:
// the row of m's line holding its offset, where the view shows the entry's
// lines (a message, a failure); 0, the top, for any other entry.
func EntryViewRow(s *state.State, ui UI, m ConvMatch) (int, bool) {
	if m.Item < 0 || m.Item >= len(s.Conversation) {
		return 0, false
	}
	e := s.Conversation[m.Item]
	var ls []string
	switch e.Role {
	case "user":
		ls = entryLines(s, e)
	case "tool", "thinking", "review":
		return 0, true
	default:
		ls = strings.Split(e.Text, "\n")
	}
	w, row := ui.Width-2, 0
	for j, l := range ls {
		for _, para := range paragraphs(l) {
			rs := wrapPara(para, w)
			if j == m.Line {
				off := viewOffset(l, m.Off)
				for k, r := range rs {
					if off < r.to || k == len(rs)-1 {
						return row + k, true
					}
				}
			}
			row += len(rs)
		}
	}
	return row, true
}

// viewOffset is byte off of line l in the line as wrap draws it
// (paragraphs): a tab before it four spaces, a control character dropped.
func viewOffset(l string, off int) int {
	out := off
	for _, r := range l[:min(max(off, 0), len(l))] {
		switch {
		case r == '\t':
			out += 3
		case r < 0x20 || r == 0x7f:
			out--
		}
	}
	return out
}

// markTarget gives tool row i's target the references a search lights it
// by (plan 66, design D7): each span its offset in the whole target, of a
// kind no selection takes and no copy copies.
func markTarget(i int, d line) line {
	if len(d) == 0 {
		return d // no target: nil stays nil, so that toolLine draws no column for it
	}
	out := make(line, len(d))
	off := 0
	for k, x := range d {
		x.ref = ref{panel: 'B', kind: refTarget, item: i, off: off}
		off += len(x.text)
		out[k] = x
	}
	return out
}

// retarget is cut, fitDetail's cut of target d, with true references (plan
// 66, design D7): a path's left end cut gives "…", one unit for the bytes it
// hides, then the rest at its offset; anything else's right end cut keeps
// the spans drawn at theirs, then the "…" for the bytes past them.
func retarget(d, cut line) line {
	i, full := d[0].ref.item, 0
	for _, x := range d {
		full += len(x.text)
	}
	if len(cut) == 1 && len(d) == 1 && strings.HasPrefix(cut[0].text, "…") && !strings.HasPrefix(d[0].text, "…") {
		if rest := cut[0].text[len("…"):]; strings.HasSuffix(d[0].text, rest) {
			hid := len(d[0].text) - len(rest)
			a, b := cut[0], cut[0]
			a.text, a.ref = "…", ref{panel: 'B', kind: refTargetCut, item: i, n: hid}
			b.text, b.ref = rest, ref{panel: 'B', kind: refTarget, item: i, off: hid}
			return line{a, b}
		}
	}
	out := append(line(nil), cut...)
	at, k := 0, 0
	for k < len(out) && k < len(d) && strings.HasPrefix(d[k].text, out[k].text) {
		out[k].ref = ref{panel: 'B', kind: refTarget, item: i, off: at}
		whole := out[k].text == d[k].text
		at += len(out[k].text)
		k++
		if !whole {
			break // cut inside this span: the "…" follows
		}
	}
	if k < len(out) && out[k].text == "…" && at < full {
		out[k].ref = ref{panel: 'B', kind: refTargetCut, item: i, off: at, n: full - at}
	}
	return out
}

// lightSearch lights the search's matches in the rows drawn (plan 66,
// design D7), each on the file viewer's amber (stMatch), the current one in
// its current-match style (stMatchCur), before a selection is lit over
// them: zone B's rows from their sources, as the selection is lit; a tool
// row's target where the row draws it, the "…" of its cut for what it
// hides, its name for a target it has no room for; a long paste's fold row
// for its folded lines. Over an entry's full view, its rows (lightView).
func lightSearch(sc *Screen, s *state.State, ui UI) {
	v := ui.Search
	if !v.Open || len(v.Matches) == 0 {
		return
	}
	var cur ConvMatch
	hasCur := v.Cur >= 0 && v.Cur < len(v.Matches)
	if hasCur {
		cur = v.Matches[v.Cur]
	}
	if ui.ViewEntry > 0 {
		lightView(sc, s, ui, cur, hasCur)
		return
	}
	shown, drawn := map[int]bool{}, map[int]bool{} // the entries B's rows show; those whose target they draw
	for _, l := range sc.lines {
		for _, x := range l {
			switch {
			case x.ref.panel != 'B':
			case x.ref.kind == refText || x.ref.kind == refAtom || x.ref.kind == refFold:
				shown[x.ref.item] = true
			case x.ref.kind == refTarget || x.ref.kind == refTargetCut:
				shown[x.ref.item], drawn[x.ref.item] = true, true
			}
		}
	}
	by := map[int][]ConvMatch{}
	for _, m := range v.Matches {
		if shown[m.Item] {
			by[m.Item] = append(by[m.Item], m)
		}
	}
	if len(by) == 0 {
		return
	}
	style := func(m ConvMatch) *lipgloss.Style {
		if hasCur && m == cur {
			return stMatchCur
		}
		return stMatch
	}
	for y, l := range sc.lines {
		var out line
		for _, x := range l {
			ms := by[x.ref.item]
			if x.ref.panel != 'B' || len(ms) == 0 {
				out = append(out, x)
				continue
			}
			r := x.ref
			var lit []litRange
			var whole *lipgloss.Style // the span lit whole, in this style
			for _, m := range ms {
				switch r.kind {
				case refText:
					if m.Target && r.line == 0 && !drawn[r.item] && hasTarget(s.Conversation[r.item].Role) {
						whole = stronger(whole, style(m)) // the name, for a target not drawn
					} else if !m.Target && m.Line == r.line {
						if lo, hi := max(m.Off, r.off), min(m.Off+m.N, r.off+len(x.text)); lo < hi {
							lit = append(lit, litRange{lo - r.off, hi - r.off, style(m)})
						}
					}
				case refAtom:
					if !m.Target && m.Line == r.line && m.Off < r.off+r.n && m.Off+m.N > r.off {
						whole = stronger(whole, style(m))
					}
				case refTarget:
					if lo, hi := max(m.Off, r.off), min(m.Off+m.N, r.off+len(x.text)); m.Target && lo < hi {
						lit = append(lit, litRange{lo - r.off, hi - r.off, style(m)})
					}
				case refTargetCut:
					if m.Target && m.Off < r.off+r.n && m.Off+m.N > r.off {
						whole = stronger(whole, style(m))
					}
				case refFold:
					if !m.Target && m.Line >= foldShow {
						whole = stronger(whole, style(m))
					}
				}
			}
			if whole != nil {
				x.st = whole
				out = append(out, x)
				continue
			}
			out = append(out, splitLit(x, lit)...)
		}
		sc.lines[y] = out
	}
}

// hasTarget says an entry of role role draws a target beside its name: a
// tool row, an approval's row or its box.
func hasTarget(role string) bool { return role == "tool" || role == "approval" }

// litRange is a part of a span to light, its bytes from and to, and how.
type litRange struct {
	from, to int
	st       *lipgloss.Style
}

// stronger is the current match's style when either is, else a match's.
func stronger(a, b *lipgloss.Style) *lipgloss.Style {
	if a == stMatchCur || b == stMatchCur {
		return stMatchCur
	}
	return stMatch
}

// splitLit is x with its parts in lit restyled, the rest as it was.
func splitLit(x span, lit []litRange) line {
	if len(lit) == 0 {
		return line{x}
	}
	slices.SortFunc(lit, func(a, b litRange) int { return a.from - b.from })
	var out line
	at := 0
	part := func(from, to int, st *lipgloss.Style) {
		if from >= to {
			return
		}
		y := x
		y.text, y.st = x.text[from:to], st
		y.ref.off = x.ref.off + from
		out = append(out, y)
	}
	for _, r := range lit {
		from := max(r.from, at)
		part(at, from, x.st)
		part(from, r.to, r.st)
		at = max(at, r.to)
	}
	part(at, len(x.text), x.st)
	return out
}

// lightView lights the query in an entry's full view, the search's entry
// there (plan 66, design D8): for the view of an entry's lines (a message,
// a failure), the matches of each line as the view draws it, across the
// rows its wrap breaks it into, the bar's current one in its style; for any
// other entry, the query in each row drawn.
func lightView(sc *Screen, s *state.State, ui UI, cur ConvMatch, hasCur bool) {
	i, q := ui.ViewEntry-1, ui.Search.Query
	if i < 0 || i >= len(s.Conversation) || q == "" {
		return
	}
	fold := q == strings.ToLower(q)
	e := s.Conversation[i]
	rows := map[int][]litRange{} // by the view's row: what to light
	var ls []string
	switch e.Role {
	case "user":
		ls = entryLines(s, e)
	case "tool", "thinking", "review":
	default:
		ls = strings.Split(e.Text, "\n")
	}
	if ls != nil {
		row := 0
		for j, l := range ls {
			for _, para := range paragraphs(l) {
				rs := wrapPara(para, ui.Width-2)
				findIn(para, q, fold, func(off int) {
					st := stMatch
					if hasCur && cur.Item == i && !cur.Target && cur.Line == j && viewOffset(l, cur.Off) == off {
						st = stMatchCur
					}
					for k, r := range rs {
						if lo, hi := max(off, r.from), min(off+len(q), r.to); lo < hi {
							rows[row+k] = append(rows[row+k], litRange{lo - r.from, hi - r.from, st})
						}
					}
				})
				row += len(rs)
			}
		}
	}
	for y, l := range sc.lines {
		var text strings.Builder
		item := -1
		for _, x := range l {
			if x.ref.panel == 'I' && x.ref.kind == refText {
				item = x.ref.item
				text.WriteString(x.text)
			}
		}
		if item < 0 {
			continue
		}
		lit := rows[item]
		if ls == nil {
			findIn(text.String(), q, fold, func(off int) { lit = append(lit, litRange{off, off + len(q), stMatch}) })
		}
		if len(lit) == 0 {
			continue
		}
		var out line
		for _, x := range l {
			if x.ref.panel != 'I' || x.ref.kind != refText {
				out = append(out, x)
				continue
			}
			var mine []litRange
			for _, r := range lit {
				if lo, hi := max(r.from, x.ref.off), min(r.to, x.ref.off+len(x.text)); lo < hi {
					mine = append(mine, litRange{lo - x.ref.off, hi - x.ref.off, r.st})
				}
			}
			out = append(out, splitLit(x, mine)...)
		}
		sc.lines[y] = out
	}
}
