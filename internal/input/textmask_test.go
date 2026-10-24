//go:build windows

package input

import (
	"image"
	"testing"
)

// inkRows is the first and last rows of m with ink, and its inked and
// partly inked pixels.
func inkRows(m *image.Alpha) (top, bottom, ink, partial int) {
	top, bottom = m.Bounds().Dy(), -1
	for y := range m.Bounds().Dy() {
		for x := range m.Bounds().Dx() {
			a := m.AlphaAt(x, y).A
			if a == 0 {
				continue
			}
			ink++
			if a < 255 {
				partial++
			}
			top, bottom = min(top, y), max(bottom, y)
		}
	}
	return top, bottom, ink, partial
}

// TestTextMask is plan 39's design D7: the mode pill's title as GDI draws
// it in Cascadia Mono (Consolas where it is missing), at the application's
// 13 pixels a character from one pixel down: a coverage mask as tall as
// asked, as wide as GDI measures the text, 8 pixels a character; the
// capitals around the row's middle, the descenders above its last two rows
// (the pill's outline), grey antialiased (partial coverage on the strokes'
// edges); an empty title is no pixel.
func TestTextMask(t *testing.T) {
	m, err := TextMask("Auto", 13, 1, 20)
	if err != nil {
		t.Fatal(err)
	}
	if b := m.Bounds(); b.Dy() != 20 || b.Dx() != 4*8 {
		t.Fatalf("bounds %v, want 32x20", b)
	}
	top, bottom, ink, partial := inkRows(m)
	if ink < 50 || partial == 0 || top < 5 || bottom > 15 || (top+bottom)/2 != 10 {
		t.Errorf("Auto: %d inked pixels, %d partial, rows %d to %d", ink, partial, top, bottom)
	}
	d, err := TextMask("Approve for me", 13, 1, 20)
	if err != nil || d.Bounds().Dx() != 14*8 {
		t.Fatalf("14 characters %v, want 112 pixels (%v)", d.Bounds(), err)
	}
	if _, bottom, _, _ := inkRows(d); bottom != 17 {
		t.Errorf("Approve's descenders down to row %d, want 17", bottom)
	}
	if e, err := TextMask("", 13, 1, 20); err != nil || e.Bounds() != image.Rect(0, 0, 0, 20) {
		t.Errorf("empty: %v %v", e.Bounds(), err)
	}
}
