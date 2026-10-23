package input

import (
	"errors"
	"fmt"
)

// The WebP header (plan 31, design D1): the dashboard reads the container
// and the first chunk's header for the image's size and its flags, and
// sends the file as it is, both providers taking WebP. It decodes no
// pixels: an animated WebP is refused, since its first frame cannot be
// taken without a decoder.

// inspectWebP validates the container and the first chunk's header.
func inspectWebP(img *Image, b []byte) ([]byte, error) {
	if len(b) < 30 {
		return nil, errors.New("WebP: truncated")
	}
	if riff := int(le.Uint32(b[4:8])); riff+8 > len(b) {
		return nil, fmt.Errorf("WebP: truncated (%d of %d bytes)", len(b), riff+8)
	}
	chunk := string(b[12:16])
	size := int(le.Uint32(b[16:20]))
	p := b[20:]
	if size > len(p) {
		return nil, errors.New("WebP: truncated chunk")
	}
	switch chunk {
	case "VP8 ":
		// The key frame's header: a 3-byte frame tag, the start code, then
		// 14 bits of width and of height.
		if size < 10 || p[3] != 0x9d || p[4] != 0x01 || p[5] != 0x2a {
			return nil, errors.New("WebP: no VP8 key frame")
		}
		img.Width, img.Height = int(le.Uint16(p[6:8])&0x3fff), int(le.Uint16(p[8:10])&0x3fff)
	case "VP8L":
		if size < 5 || p[0] != 0x2f {
			return nil, errors.New("WebP: no VP8L signature")
		}
		bits := le.Uint32(p[1:5])
		img.Width, img.Height = int(bits&0x3fff)+1, int((bits>>14)&0x3fff)+1
	case "VP8X":
		if size < 10 {
			return nil, errors.New("WebP: truncated VP8X")
		}
		if p[0]&0x02 != 0 {
			return nil, errors.New("animated WebP: no frame can be taken without a decoder; convert it to PNG or JPEG first")
		}
		img.Width = int(p[4]) | int(p[5])<<8 | int(p[6])<<16 + 1
		img.Height = int(p[7]) | int(p[8])<<8 | int(p[9])<<16 + 1
	default:
		return nil, fmt.Errorf("WebP: an unknown first chunk %q", chunk)
	}
	if err := tooBig(img.Width, img.Height); err != nil {
		return nil, err
	}
	img.Media = "image/webp"
	return b, nil
}
