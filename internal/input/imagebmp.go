package input

import (
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"image/color"
)

// The dashboard's BMP reader (plan 31, design D1): a BITMAPINFOHEADER or a
// later header, 24 or 32 bits BI_RGB or 32 bits BI_BITFIELDS, 8 bits with a
// palette, bottom-up or top-down. Anything else is refused by name.

var le = binary.LittleEndian

// decodeBMP decodes a .bmp file: its file header, then the DIB.
func decodeBMP(b []byte) (*image.NRGBA, error) {
	if len(b) < 14 {
		return nil, errors.New("truncated")
	}
	off := int(le.Uint32(b[10:14]))
	if off < 14 || off > len(b) {
		return nil, errors.New("the pixel offset is outside the file")
	}
	img, _, err := decodeDIB(b[14:], off-14, false)
	return img, err
}

// decodeDIB decodes a device-independent bitmap at b: its header, its
// palette and its pixels, which start at pixOff when pixOff is positive and
// right after the palette otherwise. For an icon's image the height is
// doubled by an AND mask, which is applied for pixels without alpha. It
// returns the image and where the pixels end.
func decodeDIB(b []byte, pixOff int, icon bool) (*image.NRGBA, int, error) {
	if len(b) < 40 {
		return nil, 0, errors.New("truncated header")
	}
	hdr := int(le.Uint32(b[0:4]))
	if hdr < 40 {
		return nil, 0, fmt.Errorf("a %d-byte header: only BITMAPINFOHEADER and later are read", hdr)
	}
	if hdr > len(b) {
		return nil, 0, errors.New("truncated header")
	}
	width, height := int(int32(le.Uint32(b[4:8]))), int(int32(le.Uint32(b[8:12])))
	bpp := int(le.Uint16(b[14:16]))
	comp := le.Uint32(b[16:20])
	colors := int(le.Uint32(b[32:36]))
	topDown := height < 0
	if topDown {
		height = -height
	}
	if icon {
		height /= 2
	}
	if err := tooBig(width, height); err != nil {
		return nil, 0, err
	}
	switch {
	case comp == 1 || comp == 2:
		return nil, 0, errors.New("RLE compression: not read")
	case comp == 3 && bpp == 32, comp == 0 && (bpp == 24 || bpp == 32 || bpp == 8):
	case comp == 0:
		return nil, 0, fmt.Errorf("%d bits a pixel: not read", bpp)
	default:
		return nil, 0, fmt.Errorf("compression %d: not read", comp)
	}
	var pal []color.NRGBA
	palOff := hdr
	if bpp == 8 {
		if colors == 0 || colors > 256 {
			colors = 256
		}
		if palOff+colors*4 > len(b) {
			return nil, 0, errors.New("truncated palette")
		}
		for i := range colors {
			p := b[palOff+i*4:]
			pal = append(pal, color.NRGBA{p[2], p[1], p[0], 255})
		}
		palOff += colors * 4
	}
	var rm, gm, bm, am uint32
	if comp == 3 {
		if hdr >= 52 {
			rm, gm, bm = le.Uint32(b[40:44]), le.Uint32(b[44:48]), le.Uint32(b[48:52])
			if hdr >= 56 {
				am = le.Uint32(b[52:56])
			}
		} else {
			if len(b) < 52 {
				return nil, 0, errors.New("truncated masks")
			}
			rm, gm, bm = le.Uint32(b[40:44]), le.Uint32(b[44:48]), le.Uint32(b[48:52])
			palOff = 52
		}
		if rm == 0 || gm == 0 || bm == 0 {
			return nil, 0, errors.New("empty colour masks")
		}
	}
	if pixOff <= 0 {
		pixOff = palOff
	}
	stride := (bpp*width + 31) / 32 * 4
	if pixOff+stride*height > len(b) {
		return nil, 0, errors.New("truncated pixels")
	}
	img := image.NewNRGBA(image.Rect(0, 0, width, height))
	for y := range height {
		src := height - 1 - y
		if topDown {
			src = y
		}
		row := b[pixOff+src*stride:]
		for x := range width {
			var c color.NRGBA
			switch bpp {
			case 8:
				i := int(row[x])
				if i < len(pal) {
					c = pal[i]
				}
			case 24:
				c = color.NRGBA{row[x*3+2], row[x*3+1], row[x*3], 255}
			case 32:
				v := le.Uint32(row[x*4:])
				if comp == 3 {
					c = color.NRGBA{masked(v, rm), masked(v, gm), masked(v, bm), 255}
					if am != 0 {
						c.A = masked(v, am)
					}
				} else {
					c = color.NRGBA{row[x*4+2], row[x*4+1], row[x*4], 255}
				}
			}
			img.SetNRGBA(x, y, c)
		}
	}
	end := pixOff + stride*height
	if icon && (bpp < 32 || comp == 0) {
		// The AND mask: one bit a pixel, set where the pixel is transparent.
		ms := (width + 31) / 32 * 4
		if end+ms*height <= len(b) {
			for y := range height {
				src := height - 1 - y
				if topDown {
					src = y
				}
				mrow := b[end+src*ms:]
				for x := range width {
					if mrow[x/8]&(0x80>>(x%8)) != 0 {
						c := img.NRGBAAt(x, y)
						c.A = 0
						img.SetNRGBA(x, y, c)
					}
				}
			}
			end += ms * height
		}
	}
	return img, end, nil
}

// masked is the channel the mask selects, scaled to 8 bits.
func masked(v, mask uint32) uint8 {
	if mask == 0 {
		return 0
	}
	shift, bits := 0, 0
	for mask&(1<<shift) == 0 {
		shift++
	}
	for (mask>>shift)&(1<<bits) != 0 {
		bits++
	}
	x := (v & mask) >> shift
	if bits >= 8 {
		return uint8(x >> (bits - 8))
	}
	return uint8(x * 255 / (1<<bits - 1))
}
