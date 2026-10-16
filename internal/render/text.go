package render

import (
	"strings"

	"charm.land/lipgloss/v2"
)

// span is a run of plain text in one style. Widths are measured on the plain
// text, so a line is cut and padded before any escape code is added.
type span struct {
	text string
	st   lipgloss.Style
}

// line is one screen line under construction.
type line []span

func sp(text string, st lipgloss.Style) span { return span{text, st} }

// plain is a line of one span.
func plain(text string, st lipgloss.Style) line { return line{sp(text, st)} }

func (l line) width() int {
	w := 0
	for _, s := range l {
		w += textWidth(s.text)
	}
	return w
}

func (l line) String() string {
	var b strings.Builder
	for _, s := range l {
		if s.text != "" {
			b.WriteString(s.st.Render(s.text))
		}
	}
	return b.String()
}

// fit cuts the line to w cells, ending with "…" when cut, and pads it with
// spaces to exactly w.
func (l line) fit(w int) line {
	if w <= 0 {
		return nil
	}
	if l.width() <= w {
		return l.pad(w)
	}
	var out line
	left := w - 1
	for _, s := range l {
		if left <= 0 {
			break
		}
		t := cut(s.text, left)
		out = append(out, sp(t, s.st))
		left -= textWidth(t)
	}
	st := lipgloss.NewStyle()
	if len(out) > 0 {
		st = out[len(out)-1].st
	}
	return append(out, sp("…", st)).pad(w)
}

func (l line) pad(w int) line {
	if n := w - l.width(); n > 0 {
		return append(l, sp(strings.Repeat(" ", n), lipgloss.NewStyle()))
	}
	return l
}

// spread puts right at the right edge of a w-wide line after left, cutting
// left first when both do not fit.
func spread(left, right line, w int) line {
	rw := right.width()
	if rw >= w {
		return right.fit(w)
	}
	return append(left.fit(w-rw), right...)
}

// textWidth is the number of terminal cells s takes.
func textWidth(s string) int {
	n := 0
	for _, r := range s {
		n += runeWidth(r)
	}
	return n
}

func runeWidth(r rune) int {
	if r < 0x300 {
		if r < 0x20 || r == 0x7f {
			return 0
		}
		return 1
	}
	return lipgloss.Width(string(r))
}

// cut returns the longest prefix of s at most w cells wide.
func cut(s string, w int) string {
	n := 0
	for i, r := range s {
		rw := runeWidth(r)
		if n+rw > w {
			return s[:i]
		}
		n += rw
	}
	return s
}

// wrap breaks text into lines at most w cells wide, at spaces where it can.
// Tabs become four spaces; other control characters are dropped.
func wrap(text string, w int) []string {
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
				out = append(out, head)
				word = word[len(head):]
			}
			cur, cw = word, textWidth(word)
		}
		out = append(out, strings.TrimRight(cur, " "))
	}
	return out
}

// box draws a rounded frame w wide and h high around body, with a title on
// the left of the top edge and an optional label on its right.
func box(title, right line, body []line, w, h int, border lipgloss.Style) []line {
	if h < 2 || w < 5 { // the smallest top edge is "╭─  ╮"
		return blank(w, h)
	}
	inner := w - 4
	top := line{sp("╭─ ", border)}
	top = append(top, title...)
	if right != nil {
		// "╭─ " title " " fill " " right " ─╮"
		fill := w - 3 - title.width() - 2 - right.width() - 3
		if fill < 1 {
			right = nil
		} else {
			top = append(top, sp(" "+strings.Repeat("─", fill)+" ", border))
			top = append(top, right...)
			top = append(top, sp(" ─╮", border))
		}
	}
	if right == nil {
		fill := w - 3 - title.width() - 2
		if fill < 0 {
			top = append(line{sp("╭─ ", border)}, title.fit(w-5)...)
			fill = 0
		}
		top = append(top, sp(" "+strings.Repeat("─", fill)+"╮", border))
	}
	out := []line{top.fit(w)}
	for i := 0; i < h-2; i++ {
		var b line
		if i < len(body) {
			b = body[i]
		}
		row := line{sp("│ ", border)}
		row = append(row, b.fit(inner)...)
		out = append(out, append(row, sp(" │", border)))
	}
	return append(out, plain("╰"+strings.Repeat("─", w-2)+"╯", border))
}

// blank is h empty lines w wide.
func blank(w, h int) []line {
	out := make([]line, 0, h)
	for i := 0; i < h; i++ {
		out = append(out, line{}.pad(w))
	}
	return out
}

// beside joins columns of equal height side by side.
func beside(cols ...[]line) []line {
	if len(cols) == 0 {
		return nil
	}
	out := make([]line, len(cols[0]))
	for i := range out {
		for _, c := range cols {
			if i < len(c) {
				out[i] = append(out[i], c[i]...)
			}
		}
	}
	return out
}

// tail keeps the last n lines.
func tail(ls []line, n int) []line {
	if len(ls) > n {
		return ls[len(ls)-n:]
	}
	return ls
}
