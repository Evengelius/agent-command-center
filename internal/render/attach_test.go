package render

import (
	"strings"
	"testing"

	"github.com/Evengelius/agent-command-center/internal/event"
)

// TestAttachRow is plan 31's design D4: the chips on the tool rows' ground,
// two cells apart, each ✕ a span of panel X naming its attachment; the
// chips past the width fold into "+N more"; a sent message's image rows.
func TestAttachRow(t *testing.T) {
	defer UsePalette(pickTrue)
	chips := []Attachment{{Name: "one.png", Size: "1 KB"}, {Name: "two.jpg", Size: "6 KB"}, {Name: "three.gif", Size: "2 KB"}}
	l := attachRow(chips, 80)
	if got := strings.TrimRight(texts([]line{l}), "\n"); got != "  one.png · 1 KB ✕    two.jpg · 6 KB ✕    three.gif · 2 KB ✕ " {
		t.Errorf("the row %q", got)
	}
	x := 0
	n := 0
	for _, s := range l {
		if s.text == "✕" {
			if s.ref.panel != attachPanel || s.ref.item != n {
				t.Errorf("✕ %d: panel %q item %d", n, s.ref.panel, s.ref.item)
			}
			n++
		}
		if strings.Contains(s.text, "one.png") && s.st != tint(stText) {
			t.Errorf("a name styled %v, want on the ground", s.st)
		}
		x += textWidth(s.text)
	}
	if n != 3 {
		t.Errorf("%d ✕, want 3", n)
	}
	// Narrower: the chips that do not fit fold into "+N more".
	if got := strings.TrimRight(texts([]line{attachRow(chips, 50)}), "\n"); got != "  one.png · 1 KB ✕    two.jpg · 6 KB ✕   +1 more" {
		t.Errorf("folded %q", got)
	}
	if got := strings.TrimRight(texts([]line{attachRow(chips, 30)}), "\n"); got != "  one.png · 1 KB ✕   +2 more" {
		t.Errorf("folded to one %q", got)
	}
	// The Screen finds each chip's ✕ by its cell.
	sc := &Screen{lines: []line{attachRow(chips, 80)}}
	cell := 0
	for _, s := range sc.lines[0] {
		if s.text == "✕" {
			if p, item := sc.ItemAt(cell, 0); p != attachPanel || item != s.ref.item {
				t.Errorf("ItemAt(%d) = %q %d, want X %d", cell, p, item, s.ref.item)
			}
		}
		cell += textWidth(s.text)
	}
	// A sent message's rows: the name, else the media type.
	if got := texts([]line{imageRow(event.Image{Name: "one.png", Media: "image/png", Size: 1177}, 60)}); got != "🗀  one.png · 1 KB\n" {
		t.Errorf("an image row %q", got)
	}
	if got := texts([]line{imageRow(event.Image{Media: "image/webp", Size: 2_500_000}, 60)}); got != "🗀  image/webp · 2.5 MB\n" {
		t.Errorf("an unnamed image row %q", got)
	}
}
