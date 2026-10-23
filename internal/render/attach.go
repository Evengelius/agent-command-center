package render

import (
	"fmt"
	"slices"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/Evengelius/agent-command-center/internal/event"
)

// Attached images (plan 31, brief amendment 31; ui-design.md, Input editor
// and Zone B): the draft's as chips on a row of their own under the
// editor's rule, each removable by its ✕; a sent message's as rows under
// its text in zone B.

// attachPanel is the panel of a chip's ✕: a click on it removes the
// attachment its span's item names.
const attachPanel = 'X'

// attachRow is the draft's attachments, w cells wide: a margin cell, then a
// chip each, " name · 12 KB ✕ " on the tool rows' ground, two cells between
// chips; the chips past the width fold into "+N more". The chips carry no
// icon, to keep two of them on a 120-column screen's row; the sent
// message's rows in zone B have it.
func attachRow(chips []Attachment, w int) line {
	l, _ := attachLayout(chips, w)
	return l
}

// attachLayout is attachRow's row and the cell where each chip it shows
// starts, from the row's first cell.
func attachLayout(chips []Attachment, w int) (line, []int) {
	out := line{deco(" ", stNone)}
	var starts []int
	for i, c := range chips {
		chip := attachChip(i, c)
		gap, more := 0, 0
		if i > 0 {
			gap = 2
		}
		if rest := len(chips) - i - 1; rest > 0 {
			more = textWidth(fmt.Sprintf("  +%d more", rest))
		}
		if out.width()+gap+chip.width()+more > w {
			out = append(out, deco(fmt.Sprintf("%*s+%d more", gap, "", len(chips)-i), stGrey))
			break
		}
		if gap > 0 {
			out = append(out, deco("  ", stNone))
		}
		starts = append(starts, out.width())
		out = append(out, chip...)
	}
	return out, starts
}

// The preview (plan 32, brief amendment 32; reference-design.md, The image
// preview): when the terminal draws Sixel images, each attached image's
// thumbnail above its chip. The frame keeps the thumbnail's cells blank,
// marked as its (thumbPanel), and the dashboard draws the image over them
// once the frame is out (design D5).

// thumbPanel is the panel of a thumbnail's cells; the span's item is the
// attachment's index.
const thumbPanel = 'T'

// ThumbCells is a thumbnail's size in cells on a terminal of h rows (plan
// 32, design D4): 14 by 7 from 50 rows, 10 by 5 from 30, none under 30.
// At SixelCellW by SixelCellH pixels a cell, both are squares.
func ThumbCells(h int) (cols, rows int) {
	switch {
	case h >= 50:
		return 14, 7
	case h >= 30:
		return 10, 5
	}
	return 0, 0
}

// thumbGap is the cells between two thumbnails: one, about the chat's
// gap between two attachments (the owner, on the screenshots: the space
// between the images "should be way shorter").
const thumbGap = 1

// thumbStrip is the preview's rows, w cells wide: the attachments'
// thumbnails side by side in their order, cols by rows cells each,
// thumbGap apart, the first above the first chip at its first cell; a
// thumbnail past the width is left out, and so are those after it.
func thumbStrip(chips []Attachment, starts []int, cols, rows, w int) []line {
	type slot struct{ i, x int }
	var slots []slot
	x := 1 // the chips' row's margin cell
	if len(starts) > 0 {
		x = starts[0]
	}
	for i := range chips {
		if x+cols > w {
			break
		}
		slots = append(slots, slot{i, x})
		x += cols + thumbGap
	}
	out := make([]line, rows)
	for r := range rows {
		var l line
		at := 0
		for _, s := range slots {
			if s.x > at {
				l = append(l, deco(strings.Repeat(" ", s.x-at), stNone))
			}
			l = append(l, thumbRow(chips[s.i], s.i, r, cols, rows)...)
			at = s.x + cols
		}
		if at < w {
			l = append(l, deco(strings.Repeat(" ", w-at), stNone))
		}
		out[r] = l
	}
	return out
}

// thumbRow is row r of attachment i's thumbnail: its cells blank, marked
// as its, for the dashboard to draw the image over; for an image with no
// pixels to show (a WebP, which nothing here decodes), a box with its
// format's name in it.
func thumbRow(c Attachment, i, r, cols, rows int) line {
	mark := func(text string, st *lipgloss.Style) span {
		return span{text: text, st: st, ref: ref{panel: thumbPanel, kind: refDeco, item: i}}
	}
	if c.Pixels {
		return line{mark(strings.Repeat(" ", cols), stNone)}
	}
	inner := cols - 2
	switch r {
	case 0:
		return line{mark("╭"+strings.Repeat("─", inner)+"╮", stFrame)}
	case rows - 1:
		return line{mark("╰"+strings.Repeat("─", inner)+"╯", stFrame)}
	case (rows - 1) / 2:
		label := fitLabel(c.Format, inner)
		pad := inner - textWidth(label)
		return line{mark("│", stFrame), mark(strings.Repeat(" ", pad/2)+label+strings.Repeat(" ", pad-pad/2), stGrey), mark("│", stFrame)}
	}
	return line{mark("│", stFrame), mark(strings.Repeat(" ", inner), stNone), mark("│", stFrame)}
}

// Thumb is a thumbnail as the frame placed it: the attachment's index, its
// first cell, its width and height in cells, and whether it is a box of
// text rather than cells for an image.
type Thumb struct {
	Item       int
	X, Y, W, H int
	Text       bool
}

// Thumbs are the preview's thumbnails in the frame, in their items'
// order; none when the frame shows no preview.
func (sc *Screen) Thumbs() []Thumb {
	var out []Thumb
	at := map[int]int{}
	for y, l := range sc.lines {
		x := 0
		for _, s := range l {
			w := textWidth(s.text)
			if s.ref.panel == thumbPanel && w > 0 {
				i, ok := at[s.ref.item]
				if !ok {
					i = len(out)
					at[s.ref.item] = i
					out = append(out, Thumb{Item: s.ref.item, X: x, Y: y})
				}
				t := &out[i]
				t.W = max(t.W, x+w-t.X)
				t.H = max(t.H, y+1-t.Y)
				t.Text = t.Text || strings.TrimSpace(s.text) != ""
			}
			x += w
		}
	}
	slices.SortFunc(out, func(a, b Thumb) int { return a.Item - b.Item })
	return out
}

// Row is the frame's row y as the terminal shows it, once the frame is
// drawn (StringAfter); "" outside the frame.
func (sc *Screen) Row(y int) string {
	if y < 0 || y >= len(sc.rows) {
		return ""
	}
	return sc.rows[y]
}

// Region is the cells [x, x+w) of the frame's row y as the terminal shows
// them, styles included: what the dashboard writes over the place of a
// thumbnail no longer there, so its image goes and the cells hold what the
// renderer believes they hold (plan 32, design D5). Spaces stand for a
// character an edge cuts, and past the frame.
func (sc *Screen) Region(x, y, w int) string {
	if w <= 0 {
		return ""
	}
	if y < 0 || y >= len(sc.lines) {
		return strings.Repeat(" ", w)
	}
	var out line
	cx := 0
	for _, s := range sc.lines[y] {
		sw := textWidth(s.text)
		if lo, hi := max(x, cx), min(x+w, cx+sw); lo < hi {
			c := s
			c.text = cellRange(s.text, lo-cx, hi-cx)
			out = append(out, c)
		}
		cx += sw
	}
	return out.pad(w).String()
}

// cellRange is the part of s in its cells [from, to); a character across
// an edge gives a space for each of its cells inside.
func cellRange(s string, from, to int) string {
	var b strings.Builder
	n := 0
	for _, r := range s {
		end := n + runeWidth(r)
		switch {
		case end <= from && end > n, n >= to:
		case n >= from && end <= to:
			b.WriteRune(r)
		case end > n:
			b.WriteString(strings.Repeat(" ", min(end, to)-max(n, from)))
		}
		n = end
	}
	return b.String()
}

// attachChip is one chip: the name cut at 28 cells, the size and the ✕
// that removes it, all on the ground.
func attachChip(i int, c Attachment) line {
	l := line{deco(" ", stText), deco(fitLabel(c.Name, 28), stText), deco(" · "+c.Size+" ", stGrey),
		{text: "✕", st: stGrey, ref: ref{panel: attachPanel, kind: refDeco, item: i}}, deco(" ", stNone)}
	return grounded(l, l.width())
}

// imageRow is an image a sent message carried, under its text: "🗀  name ·
// 12 KB", the file's name when the dashboard sent it, else its media type.
func imageRow(img event.Image, w int) line {
	name := img.Name
	if name == "" {
		name = img.Media
	}
	if name == "" {
		name = "image"
	}
	l := line{deco("🗀  ", stMuted), deco(fitLabel(name, max(w-16, 8)), stMuted)}
	if img.Size > 0 {
		l = append(l, deco(" · "+byteSize(int64(img.Size)), stGrey))
	}
	return l
}

// byteSize writes a size as the chips do: "456 B", "12 KB", "1.2 MB".
func byteSize(n int64) string {
	switch {
	case n < 1000:
		return fmt.Sprintf("%d B", n)
	case n < 1000*1000:
		return fmt.Sprintf("%d KB", (n+500)/1000)
	}
	return strings.TrimSuffix(fmt.Sprintf("%.1f", float64(n)/1e6), ".0") + " MB"
}
