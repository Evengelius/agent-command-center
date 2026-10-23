package input

import (
	"image"
	"image/color"
	"image/draw"
	"math"
)

// Thumbnails for the preview above the chips (plan 32, design D4;
// reference-design.md, The image preview): the image cut from its centre
// to a square, as the owner's chat shows it, scaled by area averaging,
// its corners rounded and transparent, a 1-pixel border.

// Thumbnail is im's square thumbnail, side pixels a side: its centre cut
// to a square and scaled, the corners rounded by radius pixels and
// transparent outside them, the edge a 1-pixel border of the colour given.
// A pixel of the image more than half transparent is transparent in the
// thumbnail and the others are opaque: a Sixel pixel is drawn or not.
func Thumbnail(im image.Image, side, radius int, border color.NRGBA) *image.NRGBA {
	b := im.Bounds()
	s := min(b.Dx(), b.Dy())
	from := image.Pt(b.Min.X+(b.Dx()-s)/2, b.Min.Y+(b.Dy()-s)/2)
	var sq *image.RGBA
	if n := 4 * side; s > n {
		// A large image is sampled to four times the thumbnail first, so a
		// 50-megapixel photo is never copied whole; the averaging below
		// then smooths the samples.
		sq = image.NewRGBA(image.Rect(0, 0, n, n))
		for y := range n {
			for x := range n {
				sq.Set(x, y, im.At(from.X+(2*x+1)*s/(2*n), from.Y+(2*y+1)*s/(2*n)))
			}
		}
	} else {
		sq = image.NewRGBA(image.Rect(0, 0, s, s))
		draw.Draw(sq, sq.Bounds(), im, from, draw.Src)
	}
	small := shrink(sq, side, side)
	out := image.NewNRGBA(image.Rect(0, 0, side, side))
	radius = max(radius, 1)
	for y := range side {
		for x := range side {
			inside, edge := roundedPixel(x, y, side, radius)
			o := out.Pix[y*out.Stride+x*4 : y*out.Stride+x*4+4]
			switch {
			case !inside:
				// Transparent: left at zero.
			case edge:
				o[0], o[1], o[2], o[3] = border.R, border.G, border.B, 255
			default:
				p := small.Pix[y*small.Stride+x*4 : y*small.Stride+x*4+4]
				if a := p[3]; a >= 128 {
					o[0], o[1], o[2], o[3] = straight(p[0], a), straight(p[1], a), straight(p[2], a), 255
				}
			}
		}
	}
	return out
}

// straight is a premultiplied channel unpremultiplied.
func straight(c, a uint8) uint8 { return uint8(min(int(c)*255/int(a), 255)) }

// roundedPixel says whether the pixel (x, y) lies in the square of side
// pixels with its corners rounded by r, and whether on its 1-pixel edge:
// that shape is the square r inside each side, grown by r.
func roundedPixel(x, y, side, r int) (inside, edge bool) {
	fx, fy := float64(x)+0.5, float64(y)+0.5
	lo, hi := float64(r), float64(side-r)
	dx := math.Max(math.Max(lo-fx, fx-hi), 0)
	dy := math.Max(math.Max(lo-fy, fy-hi), 0)
	d := math.Hypot(dx, dy)
	return d <= float64(r), d > float64(r)-1
}
