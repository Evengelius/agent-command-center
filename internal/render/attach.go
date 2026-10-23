package render

import (
	"fmt"
	"strings"

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
	out := line{deco(" ", stNone)}
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
		out = append(out, chip...)
	}
	return out
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
