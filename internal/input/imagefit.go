package input

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/draw"
	"image/jpeg"
	"image/png"
	"math"
)

// Fitting an image to a provider's limits (plan 31, design D1): over its
// pixels a side, its patches or its bytes, the image is scaled down by
// area averaging, as the models would scale it anyway, and re-encoded in
// the media type it is sent as (a JPEG as JPEG, anything else as PNG);
// its note says so. Nothing is cropped.

// fit scales the decoded image down until it meets the limits and gives
// its bytes; it updates the image's size, media type and note.
func fit(img *Image, im image.Image, lim ImageLimits) ([]byte, error) {
	w0, h0 := im.Bounds().Dx(), im.Bounds().Dy()
	scale := 1.0
	if lim.MaxSide > 0 && max(w0, h0) > lim.MaxSide {
		scale = float64(lim.MaxSide) / float64(max(w0, h0))
	}
	patches := func(s float64) int {
		w, h := max(int(float64(w0)*s), 1), max(int(float64(h0)*s), 1)
		return ceilDiv(w, lim.PatchSize) * ceilDiv(h, lim.PatchSize)
	}
	if lim.MaxPatches > 0 && lim.PatchSize > 0 {
		for range 8 {
			p := patches(scale)
			if p <= lim.MaxPatches {
				break
			}
			scale *= math.Sqrt(float64(lim.MaxPatches)/float64(p)) * 0.99
		}
	}
	jpg := img.Media == "image/jpeg"
	src := toRGBA(im)
	var data []byte
	for range 8 {
		w, h := max(int(float64(w0)*scale), 1), max(int(float64(h0)*scale), 1)
		out := src
		if w != w0 || h != h0 {
			out = shrink(src, w, h)
		}
		var buf bytes.Buffer
		var err error
		if jpg {
			err = jpeg.Encode(&buf, out, &jpeg.Options{Quality: 90})
		} else {
			err = png.Encode(&buf, out)
		}
		if err != nil {
			return nil, errors.New("resizing: " + err.Error())
		}
		data = buf.Bytes()
		if lim.MaxBytes <= 0 || int64(len(data)) <= lim.MaxBytes {
			note := fmt.Sprintf("resized from %dx%d to %dx%d for the provider's limits", w0, h0, w, h)
			if w == w0 && h == h0 {
				note = "re-encoded for the provider's limits"
			}
			if img.Note != "" {
				note = img.Note + "; " + note
			}
			img.Width, img.Height, img.Note = w, h, note
			if !jpg {
				img.Media = pngMedia
			}
			return data, nil
		}
		// Too many bytes still: smaller by the share over the limit.
		scale *= math.Sqrt(float64(lim.MaxBytes)/float64(len(data))) * 0.95
	}
	return nil, fmt.Errorf("%s even scaled down: over the provider's %s a file", FmtBytes(int64(len(data))), FmtBytes(lim.MaxBytes))
}

// toRGBA is the image as premultiplied RGBA from (0, 0).
func toRGBA(im image.Image) *image.RGBA {
	if r, ok := im.(*image.RGBA); ok && r.Bounds().Min == (image.Point{}) {
		return r
	}
	b := im.Bounds()
	out := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(out, out.Bounds(), im, b.Min, draw.Src)
	return out
}

// shrink scales src down to w by h, each target pixel the average of the
// source pixels it covers (a box filter on premultiplied colour, so
// transparent pixels lend no colour).
func shrink(src *image.RGBA, w, h int) *image.RGBA {
	sw, sh := src.Bounds().Dx(), src.Bounds().Dy()
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	xs := make([]int, w+1)
	for i := range xs {
		xs[i] = i * sw / w
	}
	for ty := range h {
		y0, y1 := ty*sh/h, (ty+1)*sh/h
		if y1 <= y0 {
			y1 = y0 + 1
		}
		for tx := range w {
			x0, x1 := xs[tx], xs[tx+1]
			if x1 <= x0 {
				x1 = x0 + 1
			}
			var r, g, b, a, n uint64
			for y := y0; y < y1; y++ {
				row := src.Pix[y*src.Stride:]
				for x := x0; x < x1; x++ {
					p := row[x*4 : x*4+4]
					r, g, b, a, n = r+uint64(p[0]), g+uint64(p[1]), b+uint64(p[2]), a+uint64(p[3]), n+1
				}
			}
			d := dst.Pix[ty*dst.Stride+tx*4:]
			d[0], d[1], d[2], d[3] = uint8(r/n), uint8(g/n), uint8(b/n), uint8(a/n)
		}
	}
	return dst
}
