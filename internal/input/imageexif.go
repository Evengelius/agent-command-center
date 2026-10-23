package input

import (
	"encoding/binary"
	"image"
)

// A JPEG's EXIF orientation (plan 31, design D1): the providers read no
// metadata, so a photo stored on its side is turned here before it is
// sent, by the tag's eight values.

// exifOrientation is the Orientation tag of the JPEG's EXIF segment, 0 when
// there is none.
func exifOrientation(b []byte) int {
	for i := 2; i+4 <= len(b); {
		if b[i] != 0xFF {
			return 0
		}
		marker := b[i+1]
		if marker == 0xD8 || marker == 0x01 || marker >= 0xD0 && marker <= 0xD7 {
			i += 2
			continue
		}
		if marker == 0xDA || marker == 0xD9 {
			return 0 // the scan or the end: no EXIF after it
		}
		n := int(binary.BigEndian.Uint16(b[i+2 : i+4]))
		if n < 2 || i+2+n > len(b) {
			return 0
		}
		seg := b[i+4 : i+2+n]
		if marker == 0xE1 && len(seg) > 6 && string(seg[:6]) == "Exif\x00\x00" {
			return tiffOrientation(seg[6:])
		}
		i += 2 + n
	}
	return 0
}

// tiffOrientation reads the Orientation tag of IFD0.
func tiffOrientation(t []byte) int {
	if len(t) < 8 {
		return 0
	}
	var bo binary.ByteOrder
	switch string(t[0:2]) {
	case "II":
		bo = binary.LittleEndian
	case "MM":
		bo = binary.BigEndian
	default:
		return 0
	}
	ifd := int(bo.Uint32(t[4:8]))
	if ifd < 8 || ifd+2 > len(t) {
		return 0
	}
	n := int(bo.Uint16(t[ifd : ifd+2]))
	for i := range n {
		e := ifd + 2 + i*12
		if e+12 > len(t) {
			return 0
		}
		if bo.Uint16(t[e:e+2]) == 0x0112 && bo.Uint16(t[e+2:e+4]) == 3 {
			return int(bo.Uint16(t[e+8 : e+10]))
		}
	}
	return 0
}

// orient turns the image as the orientation tag asks, so it shows upright:
// 2 mirrors, 3 turns half, 4 flips, 5 transposes, 6 turns a quarter
// clockwise, 7 transverses, 8 turns a quarter counter-clockwise.
func orient(im image.Image, o int) *image.NRGBA {
	b := im.Bounds()
	w, h := b.Dx(), b.Dy()
	ow, oh := w, h
	if o >= 5 {
		ow, oh = h, w
	}
	out := image.NewNRGBA(image.Rect(0, 0, ow, oh))
	for y := range h {
		for x := range w {
			var dx, dy int
			switch o {
			case 2:
				dx, dy = w-1-x, y
			case 3:
				dx, dy = w-1-x, h-1-y
			case 4:
				dx, dy = x, h-1-y
			case 5:
				dx, dy = y, x
			case 6:
				dx, dy = h-1-y, x
			case 7:
				dx, dy = h-1-y, w-1-x
			case 8:
				dx, dy = y, w-1-x
			default:
				dx, dy = x, y
			}
			out.Set(dx, dy, im.At(b.Min.X+x, b.Min.Y+y))
		}
	}
	return out
}
