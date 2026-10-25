package input

import (
	"bytes"
	"errors"
	"image"
	"image/draw"
	"image/png"
)

// The dashboard's ICO reader (plan 31, design D1): the icon's largest image,
// a PNG by image/png or a DIB by the BMP reader, its AND mask applied.

// decodeICO decodes the largest image of an icon file.
func decodeICO(b []byte) (*image.NRGBA, error) {
	if len(b) < 6 {
		return nil, errors.New("truncated")
	}
	n := int(le.Uint16(b[4:6]))
	if n == 0 {
		return nil, errors.New("no image")
	}
	if 6+n*16 > len(b) {
		return nil, errors.New("truncated directory")
	}
	best, bestArea := -1, -1
	for i := range n {
		e := b[6+i*16:]
		w, h := int(e[0]), int(e[1])
		if w == 0 {
			w = 256
		}
		if h == 0 {
			h = 256
		}
		if w*h > bestArea {
			best, bestArea = i, w*h
		}
	}
	e := b[6+best*16:]
	size, off := int(le.Uint32(e[8:12])), int(le.Uint32(e[12:16]))
	if off < 0 || size < 0 || off+size > len(b) {
		return nil, errors.New("an image outside the file")
	}
	data := b[off : off+size]
	if bytes.HasPrefix(data, []byte("\x89PNG\r\n\x1a\n")) {
		// Its size first: decoding makes the whole image before reading its
		// data (plan 48, design D1; audit finding 2).
		cfg, err := png.DecodeConfig(bytes.NewReader(data))
		if err != nil {
			return nil, errors.New("its PNG: " + err.Error())
		}
		if err := tooBig(cfg.Width, cfg.Height); err != nil {
			return nil, err
		}
		im, err := png.Decode(bytes.NewReader(data))
		if err != nil {
			return nil, errors.New("its PNG: " + err.Error())
		}
		out := image.NewNRGBA(im.Bounds())
		draw.Draw(out, out.Bounds(), im, im.Bounds().Min, draw.Src)
		return out, nil
	}
	im, _, err := decodeDIB(data, 0, true)
	if err != nil {
		return nil, errors.New("its bitmap: " + err.Error())
	}
	return im, nil
}
