package imagetest

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

// PixelsSHA256 is the SHA-256 of the drawn image's pixels, RGBA row by row:
// the WebP fixture, drawn by a browser with the same rectangles and font,
// decodes to these pixels (testdata/images/README.md).
const PixelsSHA256 = "1e89b6e2a41bfb8f95cdb875268b17206b79502c0bba1e8ba36bd3df897c1f5c"

// TestDrawPixels pins the drawing: its size, its squares, and its pixels'
// SHA-256, which the checked-in WebP was checked against.
func TestDrawPixels(t *testing.T) {
	img := Draw()
	if b := img.Bounds(); b.Dx() != Width || b.Dy() != Height {
		t.Fatalf("drawn %v", b)
	}
	for _, s := range Squares {
		c := s.Rect.Min.Add(s.Rect.Max).Div(2)
		if got := img.NRGBAAt(c.X, c.Y); got != s.RGB {
			t.Errorf("the square at %v is %v, want %v", c, got, s.RGB)
		}
	}
	sum := sha256.Sum256(img.Pix)
	if got := hex.EncodeToString(sum[:]); got != PixelsSHA256 {
		t.Errorf("the pixels' SHA-256 is %s, want %s", got, PixelsSHA256)
	}
}
