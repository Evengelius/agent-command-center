package render

import (
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
func Sixel(img *image.NRGBA) string { return sixel(img, 0) }

// sixel is Sixel, img's exact commonest colours each an entry of its own
// before the median cut splits the others (quantize).
func sixel(img *image.NRGBA, exact int) string {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if w == 0 || h == 0 {
		return ""
	}
	idx, pal := quantize(img, exact)
	return encodeSixel(w, h, idx, pal)
}

// encodeSixel is a w by h image as Sixel, its pixels' entries in pal by idx
// (-1 transparent), every entry defined. Each band of six rows is read
// once, each colour's sixels gathered as it goes, then written in the
// entries' order.
func encodeSixel(w, h int, idx []int16, pal [][3]uint8) string {
	var s strings.Builder
	s.WriteString("\x1bP0;1;0q\"1;1;")
	s.WriteString(strconv.Itoa(w))
	s.WriteByte(';')
	s.WriteString(strconv.Itoa(h))
	for i, c := range pal {
		s.WriteByte('#')
		s.WriteString(strconv.Itoa(i))
		s.WriteString(";2;")
		s.WriteString(strconv.Itoa(sixelPercent(c[0])))
		s.WriteByte(';')
		s.WriteString(strconv.Itoa(sixelPercent(c[1])))
		s.WriteByte(';')
		s.WriteString(strconv.Itoa(sixelPercent(c[2])))
	}
	bands := make([][]byte, len(pal)) // each colour's sixels across the band, zero outside its span
	span := make([][2]int, len(pal))  // the columns a colour takes in the band, first and last
	used := make([]bool, len(pal))
	var order []int
	for y0 := 0; y0 < h; y0 += 6 {
		if y0 > 0 {
			s.WriteByte('-')
		}
		order = order[:0]
		for dy := 0; dy < 6 && y0+dy < h; dy++ {
			for x, c := range idx[(y0+dy)*w : (y0+dy+1)*w] {
				if c < 0 {
					continue
				}
				if !used[c] {
					used[c] = true
					order = append(order, int(c))
					span[c] = [2]int{x, x}
					if bands[c] == nil {
						bands[c] = make([]byte, w)
					}
				}
				span[c][0], span[c][1] = min(span[c][0], x), max(span[c][1], x)
				bands[c][x] |= 1 << dy
			}
		}
		slices.Sort(order)
		for k, c := range order {
			if k > 0 {
				s.WriteByte('$')
			}
			s.WriteByte('#')
			s.WriteString(strconv.Itoa(c))
			// The columns before the colour's first: blank sixels, one run
			// as writeRuns writes it.
			band, first, last := bands[c], span[c][0], span[c][1]+1
			if first >= 4 {
				s.WriteByte('!')
				s.WriteString(strconv.Itoa(first))
				s.WriteByte(63)
			} else {
				s.WriteString("???"[:first])
			}
			for x := first; x < last; x++ {
				band[x] += 63
			}
			writeRuns(&s, band[first:last])
			clear(band[first:last])
			used[c] = false
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
			s.WriteByte('!')
			s.WriteString(strconv.Itoa(n))
			s.WriteByte(b[i])
		} else {
			s.Write(b[i:j])
		}
		i = j
	}
}

// quantize gives each pixel its palette entry, -1 for a transparent one,
// and the palette: when the opaque pixels have more than 255 colours, the
// exact commonest each an entry of its own; the others split by median
// cut, along the widest channel of the box with the widest one, until 255
// boxes or no box can split; each entry is its box's mean weighted by
// pixel count.
func quantize(img *image.NRGBA, exact int) ([]int16, [][3]uint8) {
	pal, index := quantizePalette(img, exact)
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	idx := make([]int16, w*h)
	for y := range h {
		for x := range w {
			p := img.Pix[img.PixOffset(b.Min.X+x, b.Min.Y+y):]
			idx[y*w+x] = -1
			if p[3] >= 128 {
				idx[y*w+x] = index[rgb(p)]
			}
		}
	}
	return idx, pal
}

// rgb is a pixel's colour as one key.
func rgb(p []uint8) uint32 { return uint32(p[0])<<16 | uint32(p[1])<<8 | uint32(p[2]) }

// quantizePalette is quantize's palette, and each colour of the image's
// opaque pixels by its key (rgb) to its entry.
func quantizePalette(img *image.NRGBA, exact int) ([][3]uint8, map[uint32]int16) {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	counts := map[uint32]int{}
	for y := range h {
		for x := range w {
			p := img.Pix[img.PixOffset(b.Min.X+x, b.Min.Y+y):]
			if p[3] >= 128 {
				counts[rgb(p)]++
			}
		}
	}
	all := make([]shade, 0, len(counts))
	for c, n := range counts {
		all = append(all, shade{[3]uint8{uint8(c >> 16), uint8(c >> 8), uint8(c)}, n})
	}
	// The map's order is random: sorted, the palette is the same each run.
	slices.SortFunc(all, func(a, b shade) int { return pack(a.c) - pack(b.c) })
	var boxes []colourBox
	if exact > 0 && len(all) > 255 {
		by := slices.Clone(all)
		slices.SortStableFunc(by, func(a, b shade) int { return b.n - a.n })
		kept := map[[3]uint8]bool{}
		for _, e := range by[:exact] {
			kept[e.c] = true
			boxes = append(boxes, newColourBox([]shade{e}))
		}
		all = slices.DeleteFunc(all, func(e shade) bool { return kept[e.c] })
	}
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
	index := make(map[uint32]int16, len(counts))
	for i, bx := range boxes {
		var sum [3]int
		n := 0
		for _, e := range bx.shades {
			for k := range 3 {
				sum[k] += int(e.c[k]) * e.n
			}
			n += e.n
			index[rgb(e.c[:])] = int16(i)
		}
		for k := range 3 {
			pal[i][k] = uint8((sum[k] + n/2) / n)
		}
	}
	return pal, index
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

// sixelPalette is a fixed palette made once from a sample of a picture's
// colours, each colour mapped to its entry once, a colour the sample did
// not hold to its nearest (plan 51, design D12: the banner's tiles, many a
// second, without a median cut each).
type sixelPalette struct {
	pal   [][3]uint8
	index map[uint32]int16 // a colour's entry by its key (rgb)
}

// newSixelPalette is the palette of sample's colours, its exact commonest
// each an entry of its own (quantize).
func newSixelPalette(sample *image.NRGBA, exact int) *sixelPalette {
	pal, index := quantizePalette(sample, exact)
	return &sixelPalette{pal: pal, index: index}
}

// entry is the entry of the colour of key c (rgb): its own, else the
// nearest, then kept.
func (p *sixelPalette) entry(c uint32) int16 {
	if i, ok := p.index[c]; ok {
		return i
	}
	best, bd := 0, 1<<30
	for i, q := range p.pal {
		dr, dg, db := int(c>>16)-int(q[0]), int(c>>8&0xFF)-int(q[1]), int(c&0xFF)-int(q[2])
		if d := dr*dr + dg*dg + db*db; d < bd {
			best, bd = i, d
		}
	}
	p.index[c] = int16(best)
	return int16(best)
}

// encode is img, opaque, as a Sixel image in the palette's colours, those
// it uses defined.
func (p *sixelPalette) encode(img *image.NRGBA) string { return p.encodeChanged(img, nil) }

// encodeChanged is encode with the pixels equal to was's, img's pixels as
// they were row by row, left transparent; "" when none changed.
func (p *sixelPalette) encodeChanged(img *image.NRGBA, was []byte) string {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if w == 0 || h == 0 {
		return ""
	}
	idx := make([]int16, w*h)
	var remap [256]int16 // an entry's place in the image's palette, plus one; 0 none yet
	var pal [][3]uint8
	key, e := uint32(1<<24), int16(0) // the last colour looked up, which runs repeat
	for y := range h {
		row := img.Pix[img.PixOffset(b.Min.X, b.Min.Y+y):]
		for x := range w {
			q := row[x*4:]
			if was != nil {
				if o := was[(y*w+x)*4:]; o[0] == q[0] && o[1] == q[1] && o[2] == q[2] {
					idx[y*w+x] = -1
					continue
				}
			}
			if k := rgb(q); k != key {
				key, e = k, p.entry(k)
			}
			if remap[e] == 0 {
				pal = append(pal, p.pal[e])
				remap[e] = int16(len(pal))
			}
			idx[y*w+x] = remap[e] - 1
		}
	}
	if len(pal) == 0 {
		return ""
	}
	return encodeSixel(w, h, idx, pal)
}
