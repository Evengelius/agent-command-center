package render

import (
	"image"
	"slices"
)

// HomeShown is the home screen's picture the frame asked for (plan 51,
// design D13), zero outside picture mode: the application paints Look and
// writes it after the frame.
type HomeShown struct {
	Shown bool
	Look  HomeLook
}

// HomeRects are the cells the picture covers, in as few rectangles as
// rows of the same shape allow: the whole screen but the project field's
// text, which stays terminal text (design D4). The last bottomRows rows are
// a rectangle of their own (BottomBlock). In grid mode, the banner's rows.
func HomeRects(l HomeLook) []image.Rectangle {
	w, h := l.Geom.W, l.Geom.H
	if !l.Geom.Picture {
		// Grid mode: the banner's rows alone (design D2).
		if !l.Geom.Banner {
			return nil
		}
		return []image.Rectangle{image.Rect(0, gridBannerTop, w, gridBannerTop+l.Geom.BannerRows)}
	}
	row, c0, c1 := l.FieldRow, l.FieldCols[0], l.FieldCols[1]
	bottom := max(h-bottomRows, 0)
	if row < 0 || row >= bottom || c1 <= c0 {
		return []image.Rectangle{image.Rect(0, 0, w, bottom), image.Rect(0, bottom, w, h)}
	}
	var out []image.Rectangle
	for _, r := range []image.Rectangle{
		image.Rect(0, 0, w, row),
		image.Rect(0, row, c0, row+1),
		image.Rect(c1, row, w, row+1),
		image.Rect(0, row+1, w, bottom),
		image.Rect(0, bottom, w, h),
	} {
		if !r.Empty() {
			out = append(out, r)
		}
	}
	return out
}

// bottomRows are how many rows an image on the screen's last row spans: a
// Sixel image is drawn in bands of six pixels, and one whose last band
// passes the screen's bottom moves the cursor below it, which scrolls the
// screen a row (plan 51, step 2); three rows are 60 pixels, ten bands.
const bottomRows = 3

// BottomBlock is c, cells of a change, grown to the last bottomRows rows of
// a screen h rows high when it reaches them, so that its image ends on a
// band (bottomRows).
func BottomBlock(c image.Rectangle, h int) image.Rectangle {
	if c.Max.Y > h-bottomRows {
		c.Min.Y, c.Max.Y = min(c.Min.Y, max(h-bottomRows, 0)), h
	}
	return c
}

// HomeChanges are the cells whose pixels differ between old and now, both
// the whole screen in image pixels, within the rectangles of cells rects:
// on each row, the runs of changed cells, those less than three cells apart
// joined.
func HomeChanges(old, now *image.NRGBA, rects []image.Rectangle) []image.Rectangle {
	var out []image.Rectangle
	for _, r := range rects {
		for y := r.Min.Y; y < r.Max.Y; y++ {
			start, last := -1, -1
			for x := r.Min.X; x < r.Max.X; x++ {
				if !cellSame(old, now, x, y) {
					if start >= 0 && x-last > 3 {
						out = append(out, image.Rect(start, y, last+1, y+1))
						start = -1
					}
					if start < 0 {
						start = x
					}
					last = x
				}
			}
			if start >= 0 {
				out = append(out, image.Rect(start, y, last+1, y+1))
			}
		}
	}
	return out
}

// cellSame says the cell (x, y)'s pixels are the same in a and b.
func cellSame(a, b *image.NRGBA, x, y int) bool {
	for py := y * SixelCellH; py < (y+1)*SixelCellH; py++ {
		i := a.PixOffset(x*SixelCellW, py)
		j := b.PixOffset(x*SixelCellW, py)
		if !slices.Equal(a.Pix[i:i+SixelCellW*4], b.Pix[j:j+SixelCellW*4]) {
			return false
		}
	}
	return true
}

// HomeSixel is the picture's pixels under the cells c as a Sixel image,
// its homeExact commonest colours exact: its grounds and fills, the same
// in every tile, meet without a seam (design D13).
func HomeSixel(pic *image.NRGBA, c image.Rectangle) string {
	px := image.Rect(c.Min.X*SixelCellW, c.Min.Y*SixelCellH, c.Max.X*SixelCellW, c.Max.Y*SixelCellH)
	return sixel(pic.SubImage(px).(*image.NRGBA), homeExact)
}

// homeExact is how many of a tile's commonest colours HomeSixel keeps
// exact.
const homeExact = 32
