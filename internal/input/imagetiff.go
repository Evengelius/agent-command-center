package input

import (
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"image/color"
)

// The dashboard's TIFF reader (plan 31, design D1): baseline, uncompressed,
// 8 bits a sample, RGB, RGBA or grayscale, chunky, in strips. A compressed
// or any other TIFF is refused by name; a DNG is a camera raw, refused.

// tiffTags are the tags the reader uses: the size, the samples and their
// bits, the compression, the photometric interpretation, the strips, the
// planar configuration and DNG's version.
var tiffTags = map[uint16]bool{256: true, 257: true, 258: true, 259: true, 262: true, 273: true, 277: true, 279: true, 284: true, 50706: true}

// tiffEntry is the first directory entry of a tag the reader uses, its count
// values of size bytes read from src when used, never copied (plan 48, design
// D1; audit finding 2).
type tiffEntry struct {
	bo          binary.ByteOrder
	size, count int
	src         []byte
}

// at is the entry's value k, an integer.
func (e tiffEntry) at(k int) uint32 {
	switch e.size {
	case 1:
		return uint32(e.src[k])
	case 2:
		return uint32(e.bo.Uint16(e.src[k*2:]))
	}
	return e.bo.Uint32(e.src[k*4:])
}

// tiffCompression names the compression values a TIFF may carry.
var tiffCompression = map[uint32]string{2: "CCITT", 3: "T4", 4: "T6", 5: "LZW", 6: "JPEG", 7: "JPEG", 8: "Deflate", 32773: "PackBits", 32946: "Deflate"}

// decodeTIFF decodes the first image of a TIFF.
func decodeTIFF(b []byte) (*image.NRGBA, error) {
	if len(b) < 8 {
		return nil, errors.New("truncated")
	}
	var bo binary.ByteOrder
	switch string(b[0:2]) {
	case "II":
		bo = binary.LittleEndian
	case "MM":
		bo = binary.BigEndian
	default:
		return nil, errors.New("no byte order")
	}
	if bo.Uint16(b[2:4]) != 42 {
		return nil, errors.New("not a TIFF (no 42)")
	}
	ifd := int(bo.Uint32(b[4:8]))
	if ifd < 8 || ifd+2 > len(b) {
		return nil, errors.New("truncated IFD")
	}
	n := int(bo.Uint16(b[ifd : ifd+2]))
	if ifd+2+n*12 > len(b) {
		return nil, errors.New("truncated IFD")
	}
	// The tags the reader uses, the first entry of each, their values left in
	// the file; any other tag is skipped, its values never read (plan 48,
	// design D1; audit finding 2).
	tags := map[uint16]tiffEntry{}
	for i := range n {
		e := b[ifd+2+i*12:]
		tag, typ, count := bo.Uint16(e[0:2]), bo.Uint16(e[2:4]), int(bo.Uint32(e[4:8]))
		if _, seen := tags[tag]; seen || !tiffTags[tag] {
			continue
		}
		size := map[uint16]int{1: 1, 3: 2, 4: 4}[typ]
		if size == 0 || count <= 0 || count > 1<<20 {
			continue // a type the reader does not need (rationals, ASCII)
		}
		src := e[8:12]
		if size*count > 4 {
			off := int(bo.Uint32(e[8:12]))
			if off < 0 || off+size*count > len(b) {
				return nil, errors.New("truncated tag values")
			}
			src = b[off:]
		}
		tags[tag] = tiffEntry{bo: bo, size: size, count: count, src: src}
	}
	first := func(tag uint16, def uint32) uint32 {
		if v := tags[tag]; v.count > 0 {
			return v.at(0)
		}
		return def
	}
	if _, dng := tags[50706]; dng {
		return nil, errors.New("DNG raw: no decoder here; convert it to PNG or JPEG first")
	}
	width, height := int(first(256, 0)), int(first(257, 0))
	if err := tooBig(width, height); err != nil {
		return nil, err
	}
	if c := first(259, 1); c != 1 {
		name := tiffCompression[c]
		if name == "" {
			name = fmt.Sprintf("compression %d", c)
		}
		return nil, fmt.Errorf("%s compression: not read", name)
	}
	if first(284, 1) != 1 {
		return nil, errors.New("planar samples: not read")
	}
	samples := int(first(277, 1))
	for k := range tags[258].count {
		if bits := tags[258].at(k); bits != 8 {
			return nil, fmt.Errorf("%d bits a sample: not read", bits)
		}
	}
	photo := first(262, 2)
	switch {
	case photo == 2 && (samples == 3 || samples == 4):
	case (photo == 0 || photo == 1) && samples == 1:
	default:
		return nil, fmt.Errorf("photometric %d with %d samples: not read", photo, samples)
	}
	offsets, counts := tags[273], tags[279]
	if offsets.count == 0 || offsets.count != counts.count {
		return nil, errors.New("no strips")
	}
	rowBytes := width * samples
	img := image.NewNRGBA(image.Rect(0, 0, width, height))
	y := 0
	for i := range offsets.count {
		off, cnt := int(offsets.at(i)), int(counts.at(i))
		if off+cnt > len(b) || off < 0 {
			return nil, errors.New("truncated strip")
		}
		strip := b[off : off+cnt]
		for len(strip) >= rowBytes && y < height {
			for x := range width {
				p := strip[x*samples:]
				var c color.NRGBA
				switch {
				case samples == 1 && photo == 0:
					c = color.NRGBA{255 - p[0], 255 - p[0], 255 - p[0], 255}
				case samples == 1:
					c = color.NRGBA{p[0], p[0], p[0], 255}
				case samples == 3:
					c = color.NRGBA{p[0], p[1], p[2], 255}
				default:
					c = color.NRGBA{p[0], p[1], p[2], p[3]}
				}
				img.SetNRGBA(x, y, c)
			}
			strip = strip[rowBytes:]
			y++
		}
	}
	if y < height {
		return nil, fmt.Errorf("%d of %d rows: truncated", y, height)
	}
	return img, nil
}
