package input

import (
	"errors"
	"fmt"
	"io"
)

// gifFrames counts a GIF's frames by walking its blocks as image/gif's
// DecodeAll reads them, without decoding their pixels (plan 49, design D12;
// audit finding 9): DecodeAll kept every frame decoded, 61 MB for 200 frames
// of 640x480, to send the first. It refuses what DecodeAll refused of the
// file's structure, in its words; a later frame's pixel data, never used,
// is not read.
func gifFrames(b []byte) (int, error) {
	r := gifReader{b: b}
	head, err := r.take(13)
	if err != nil {
		return 0, fmt.Errorf("gif: reading header: %v", err)
	}
	w, h := int(head[6])|int(head[7])<<8, int(head[8])|int(head[9])<<8
	global := head[10]&0x80 != 0
	if global {
		if _, err := r.take(3 << (1 + head[10]&7)); err != nil {
			return 0, fmt.Errorf("gif: reading color table: %s", err)
		}
	}
	frames := 0
	for {
		c, err := r.byte()
		if err != nil {
			return 0, fmt.Errorf("gif: reading frames: %v", err)
		}
		switch c {
		case 0x21: // an extension
			if err := r.extension(); err != nil {
				return 0, err
			}
		case 0x2C: // an image descriptor
			if err := r.frame(w, h, global); err != nil {
				return 0, err
			}
			frames++
		case 0x3B: // the trailer
			if frames == 0 {
				return 0, errors.New("gif: missing image data")
			}
			return frames, nil
		default:
			return 0, fmt.Errorf("gif: unknown block type: 0x%.2x", c)
		}
	}
}

// gifReader reads a GIF's bytes in order.
type gifReader struct {
	b []byte
	i int
}

// take is the next n bytes.
func (r *gifReader) take(n int) ([]byte, error) {
	if len(r.b)-r.i < n {
		r.i = len(r.b)
		return nil, io.ErrUnexpectedEOF
	}
	p := r.b[r.i : r.i+n]
	r.i += n
	return p, nil
}

func (r *gifReader) byte() (byte, error) {
	p, err := r.take(1)
	if err != nil {
		return 0, err
	}
	return p[0], nil
}

// blocks passes data sub-blocks up to their terminator.
func (r *gifReader) blocks() error {
	for {
		n, err := r.byte()
		if err != nil {
			return err
		}
		if n == 0 {
			return nil
		}
		if _, err := r.take(int(n)); err != nil {
			return err
		}
	}
}

// extension passes an extension, its label read: DecodeAll's readExtension
// and readGraphicControl.
func (r *gifReader) extension() error {
	label, err := r.byte()
	if err != nil {
		return fmt.Errorf("gif: reading extension: %v", err)
	}
	size := 0
	switch label {
	case 0x01: // plain text
		size = 13
	case 0xF9: // graphic control
		p, err := r.take(6)
		if err != nil {
			return fmt.Errorf("gif: can't read graphic control: %s", err)
		}
		if p[0] != 4 {
			return fmt.Errorf("gif: invalid graphic control extension block size: %d", p[0])
		}
		if p[5] != 0 {
			return fmt.Errorf("gif: invalid graphic control extension block terminator: %d", p[5])
		}
		return nil
	case 0xFE: // comment
	case 0xFF: // application
		n, err := r.byte()
		if err != nil {
			return fmt.Errorf("gif: reading extension: %v", err)
		}
		size = int(n)
	default:
		return fmt.Errorf("gif: unknown extension 0x%.2x", label)
	}
	if _, err := r.take(size); err != nil {
		return fmt.Errorf("gif: reading extension: %v", err)
	}
	if err := r.blocks(); err != nil {
		return fmt.Errorf("gif: reading extension: %v", err)
	}
	return nil
}

// frame passes a frame, its separator read: DecodeAll's readImageDescriptor
// but the pixels, on a canvas w by h.
func (r *gifReader) frame(w, h int, global bool) error {
	p, err := r.take(9)
	if err != nil {
		return fmt.Errorf("gif: can't read image descriptor: %s", err)
	}
	left, top := int(p[0])|int(p[1])<<8, int(p[2])|int(p[3])<<8
	fw, fh := int(p[4])|int(p[5])<<8, int(p[6])|int(p[7])<<8
	if left+fw > w || top+fh > h {
		return errors.New("gif: frame bounds larger than image bounds")
	}
	if fields := p[8]; fields&0x80 != 0 {
		if _, err := r.take(3 << (1 + fields&7)); err != nil {
			return fmt.Errorf("gif: reading color table: %s", err)
		}
	} else if !global {
		return errors.New("gif: no color table")
	}
	lit, err := r.byte()
	if err != nil {
		return fmt.Errorf("gif: reading image data: %v", err)
	}
	if lit < 2 || lit > 8 {
		return fmt.Errorf("gif: pixel size in decode out of range: %d", lit)
	}
	if err := r.blocks(); err != nil {
		return errors.New("gif: not enough image data") // the file ends in the frame's data
	}
	return nil
}
