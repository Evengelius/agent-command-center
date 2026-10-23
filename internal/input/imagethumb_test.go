package input

import (
	"image"
	"image/color"
	"testing"

	"github.com/Evengelius/agent-command-center/internal/input/imagetest"
)

// TestThumbnail is plan 32's criterion 32.4 (design D4): the 320x160
// drawing gives a square cut from its centre (the green square at its
// middle), its corners transparent, its edge the border's colour, at 140
// and at 100 pixels a side.
func TestThumbnail(t *testing.T) {
	border := color.NRGBA{0x3B, 0x3F, 0x51, 0xFF}
	for _, tc := range []struct{ side, radius int }{{140, 8}, {100, 6}} {
		th := Thumbnail(imagetest.Draw(), tc.side, tc.radius, border)
		if b := th.Bounds(); b.Dx() != tc.side || b.Dy() != tc.side {
			t.Fatalf("%d: %v", tc.side, b)
		}
		last := tc.side - 1
		for _, p := range [][2]int{{0, 0}, {last, 0}, {0, last}, {last, last}, {1, 1}} {
			if a := th.NRGBAAt(p[0], p[1]).A; a != 0 {
				t.Errorf("%d: corner pixel %v alpha %d, want transparent", tc.side, p, a)
			}
		}
		mid := tc.side / 2
		for _, p := range [][2]int{{mid, 0}, {0, mid}, {mid, last}, {last, mid}} {
			if c := th.NRGBAAt(p[0], p[1]); c != border {
				t.Errorf("%d: edge pixel %v is %v, want the border %v", tc.side, p, c, border)
			}
		}
		// The drawing's centre, (160, 80), lies in its green square.
		if c := th.NRGBAAt(mid, mid-tc.side/8); c.R > 8 || c.G < 247 || c.B > 8 || c.A != 255 {
			t.Errorf("%d: the centre %v, want the green square", tc.side, c)
		}
	}
	// A large image, its centre square over four times the thumbnail, is
	// sampled first: a 1200x900 image, blue with a red middle third.
	big := image.NewNRGBA(image.Rect(0, 0, 1200, 900))
	for y := range 900 {
		for x := range 1200 {
			c := color.NRGBA{0, 0, 255, 255}
			if x >= 450 && x < 750 && y >= 300 && y < 600 {
				c = color.NRGBA{255, 0, 0, 255}
			}
			big.SetNRGBA(x, y, c)
		}
	}
	th := Thumbnail(big, 140, 8, border)
	if c := th.NRGBAAt(70, 70); c != (color.NRGBA{255, 0, 0, 255}) {
		t.Errorf("the large image's centre %v, want red", c)
	}
	if c := th.NRGBAAt(20, 70); c != (color.NRGBA{0, 0, 255, 255}) {
		t.Errorf("the large image's left %v, want blue", c)
	}
}
