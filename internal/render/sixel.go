package render

import (
	"fmt"
	"image"
	"image/color"
	"slices"
	"strconv"
	"strings"
)

// Sixel images (plan 32, design D5; doc/research/image-paste.md, Images):
// the preview's thumbnails, drawn by the terminal over cells the frame
// keeps blank. Windows Terminal gives a cell 10x20 of an image's pixels
// whatever the font (probe P32b), so a thumbnail's size in cells sets its
// size in pixels.

// SixelCellW and SixelCellH are the image pixels a cell holds, across and
// down, in Windows Terminal's Sixel images (probe P32b).
const (
	SixelCellW = 10
	SixelCellH = 20
)

// ThumbBorder is the colour of the thumbnails' 1-pixel border: the frame's
// (plan 32, design D4).
var ThumbBorder = hexColour(cFrame.tc)

// hexColour reads "#RRGGBB".
func hexColour(s string) color.NRGBA {
	v, _ := strconv.ParseUint(strings.TrimPrefix(s, "#"), 16, 32)
	return color.NRGBA{uint8(v >> 16), uint8(v >> 8), uint8(v), 0xFF}
}

// Sixel encodes img: its opaque pixels (alpha 128 and over) in up to 255
// colours chosen by median cut, its transparent ones left unset (P2 = 1,
// what lies under them shows), its size in its raster attributes, a band
// of 6 rows at a time, runs of 4 and more repeated ("!n").
func Sixel(img *image.NRGBA) string {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if w == 0 || h == 0 {
		return ""
	}
	idx, pal := quantize(img)
	var s strings.Builder
	fmt.Fprintf(&s, "\x1bP0;1;0q\"1;1;%d;%d", w, h)
	for i, c := range pal {
		fmt.Fprintf(&s, "#%d;2;%d;%d;%d", i, sixelPercent(c[0]), sixelPercent(c[1]), sixelPercent(c[2]))
	}
	used := make([]bool, len(pal))
	band := make([]byte, w)
	for y0 := 0; y0 < h; y0 += 6 {
		if y0 > 0 {
			s.WriteByte('-')
		}
		clear(used)
		for dy := 0; dy < 6 && y0+dy < h; dy++ {
			for x := range w {
				if c := idx[(y0+dy)*w+x]; c >= 0 {
					used[c] = true
				}
			}
		}
		first := true
		for c := range pal {
			if !used[c] {
				continue
			}
			if !first {
				s.WriteByte('$')
			}
			first = false
			fmt.Fprintf(&s, "#%d", c)
			last := 0
			for x := range w {
				var bits byte
				for dy := 0; dy < 6 && y0+dy < h; dy++ {
					if idx[(y0+dy)*w+x] == int16(c) {
						bits |= 1 << dy
					}
				}
				band[x] = 63 + bits
				if bits != 0 {
					last = x + 1
				}
			}
			writeRuns(&s, band[:last])
		}
	}
	s.WriteString("\x1b\\")
	return s.String()
}

// sixelPercent is a channel as Sixel's colour definitions take it, 0 to 100.
func sixelPercent(v uint8) int { return (int(v)*100 + 127) / 255 }

// writeRuns writes the sixel characters, a run of 4 or more as "!n" and
// the character.
func writeRuns(s *strings.Builder, b []byte) {
	for i := 0; i < len(b); {
		j := i
		for j < len(b) && b[j] == b[i] {
			j++
		}
		if n := j - i; n >= 4 {
			fmt.Fprintf(s, "!%d%c", n, b[i])
		} else {
			s.Write(b[i:j])
		}
		i = j
	}
}

// quantize gives each pixel its palette entry, -1 for a transparent one,
// and the palette: the opaque pixels' colours split by median cut, along
// the widest channel of the box with the widest one, until 255 boxes or no
// box can split; each entry is its box's mean weighted by pixel count.
func quantize(img *image.NRGBA) ([]int16, [][3]uint8) {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	counts := map[[3]uint8]int{}
	for y := range h {
		for x := range w {
			p := img.Pix[img.PixOffset(b.Min.X+x, b.Min.Y+y):]
			if p[3] >= 128 {
				counts[[3]uint8{p[0], p[1], p[2]}]++
			}
		}
	}
	all := make([]shade, 0, len(counts))
	for c, n := range counts {
		all = append(all, shade{c, n})
	}
	// The map's order is random: sorted, the palette is the same each run.
	slices.SortFunc(all, func(a, b shade) int { return pack(a.c) - pack(b.c) })
	var boxes []colourBox
	if len(all) > 0 {
		boxes = append(boxes, newColourBox(all))
	}
	for len(boxes) < 255 {
		bi := -1
		for i, bx := range boxes {
			if len(bx.shades) > 1 && bx.span > 0 && (bi < 0 || bx.span > boxes[bi].span) {
				bi = i
			}
		}
		if bi < 0 {
			break
		}
		bx := boxes[bi]
		ch := bx.ch
		slices.SortStableFunc(bx.shades, func(a, b shade) int { return int(a.c[ch]) - int(b.c[ch]) })
		total, acc, cut := 0, 0, 1
		for _, e := range bx.shades {
			total += e.n
		}
		for k, e := range bx.shades {
			acc += e.n
			if acc*2 >= total {
				cut = k + 1
				break
			}
		}
		cut = min(max(cut, 1), len(bx.shades)-1)
		boxes[bi] = newColourBox(bx.shades[:cut])
		boxes = append(boxes, newColourBox(bx.shades[cut:]))
	}
	pal := make([][3]uint8, len(boxes))
	index := make(map[[3]uint8]int16, len(counts))
	for i, bx := range boxes {
		var sum [3]int
		n := 0
		for _, e := range bx.shades {
			for k := range 3 {
				sum[k] += int(e.c[k]) * e.n
			}
			n += e.n
			index[e.c] = int16(i)
		}
		for k := range 3 {
			pal[i][k] = uint8((sum[k] + n/2) / n)
		}
	}
	idx := make([]int16, w*h)
	for y := range h {
		for x := range w {
			p := img.Pix[img.PixOffset(b.Min.X+x, b.Min.Y+y):]
			idx[y*w+x] = -1
			if p[3] >= 128 {
				idx[y*w+x] = index[[3]uint8{p[0], p[1], p[2]}]
			}
		}
	}
	return idx, pal
}

// pack is a colour as one number, for sorting.
func pack(c [3]uint8) int { return int(c[0])<<16 | int(c[1])<<8 | int(c[2]) }

// shade is one colour of the image and its pixel count.
type shade struct {
	c [3]uint8
	n int
}

// colourBox is a set of shades with its widest channel and that channel's span.
type colourBox struct {
	shades []shade
	ch     int
	span   int
}

func newColourBox(shades []shade) colourBox {
	bx := colourBox{shades: shades}
	for k := range 3 {
		lo, hi := 255, 0
		for _, e := range shades {
			lo, hi = min(lo, int(e.c[k])), max(hi, int(e.c[k]))
		}
		if hi-lo > bx.span {
			bx.ch, bx.span = k, hi-lo
		}
	}
	return bx
}
