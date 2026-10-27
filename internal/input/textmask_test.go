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

// TestGlyph is plan 51's design D3: a character for the home screen's
// picture as GDI draws it in Cascadia Mono (Consolas where it is missing),
// drawn four times larger and averaged down: a mask 3em/2 high and as wide
// as the character's cell, rounded up, the character in its middle, its
// capitals from about 0.3em down to 1.0em, smooth (many partial pixels);
// semi-bold heavier; a space no ink.
func TestGlyph(t *testing.T) {
	m, err := Glyph('H', 20, false)
	if err != nil {
		t.Fatal(err)
	}
	b := m.Bounds()
	if b.Dy() != 30 || b.Dx() < 10 || b.Dx() > 13 {
		t.Fatalf("H at 20: bounds %v", b)
	}
	top, bottom, ink, partial := inkRows(m)
	if top < 4 || top > 8 || bottom < 18 || bottom > 21 || partial*4 < ink {
		t.Errorf("H at 20: rows %d to %d, %d inked pixels, %d partial", top, bottom, ink, partial)
	}
	left, right := b.Dx(), -1
	for y := range b.Dy() {
		for x := range b.Dx() {
			if m.AlphaAt(x, y).A > 0 {
				left, right = min(left, x), max(right, x)
			}
		}
	}
	if d := left - (b.Dx() - 1 - right); d < -1 || d > 1 {
		t.Errorf("H at 20: ink from column %d to %d of %d, not centred", left, right, b.Dx())
	}
	sum := func(m *image.Alpha) (n int) {
		for _, a := range m.Pix {
			n += int(a)
		}
		return n
	}
	bold, err := Glyph('H', 20, true)
	if err != nil || sum(bold) <= sum(m) {
		t.Errorf("semi-bold H: %d of ink, regular %d (%v)", sum(bold), sum(m), err)
	}
	if sp, err := Glyph(' ', 20, false); err != nil || sum(sp) != 0 {
		t.Errorf("space: %d of ink (%v)", sum(sp), err)
	}
}
