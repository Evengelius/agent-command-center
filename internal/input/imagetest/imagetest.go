// Package imagetest draws the dashboard's own test image, a red, a green
// and a blue square and the text "ACC IMAGE UPLOAD TEST", and writes it in
// the formats the dashboard reads, with its own writers where Go's standard
// library has none (plan 31, design D7, brief amendment 31: "Generate your
// own valid image files programmatically"). Tests of the input, app and
// provider packages use it; nothing of the dashboard itself does.
package imagetest

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"os"
	"path/filepath"
)

// Width and Height are the test image's size.
const (
	Width  = 320
	Height = 160
	Text   = "ACC IMAGE UPLOAD TEST"
)

// Squares are the three squares' rectangles and colours: red, green, blue,
// 64 px, on a row near the top.
var Squares = []struct {
	Rect image.Rectangle
	RGB  color.NRGBA
}{
	{image.Rect(24, 24, 88, 88), color.NRGBA{255, 0, 0, 255}},
	{image.Rect(128, 24, 192, 88), color.NRGBA{0, 255, 0, 255}},
	{image.Rect(232, 24, 296, 88), color.NRGBA{0, 0, 255, 255}},
}

// Draw is the test image: white, the three squares, the text in black in
// a 5x7 pixel font at twice its size.
func Draw() *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, Width, Height))
	draw.Draw(img, img.Bounds(), image.NewUniform(color.NRGBA{255, 255, 255, 255}), image.Point{}, draw.Src)
	for _, s := range Squares {
		draw.Draw(img, s.Rect, image.NewUniform(s.RGB), image.Point{}, draw.Src)
	}
	DrawText(img, Text, 34, 112, 2, color.NRGBA{0, 0, 0, 255})
	return img
}

// DrawText draws s in the pixel font at (x, y), each font pixel scale px
// wide, 6 font columns a character.
func DrawText(img *image.NRGBA, s string, x, y, scale int, c color.NRGBA) {
	for _, r := range s {
		rows, ok := font[r]
		if !ok {
			rows = font['?']
		}
		for dy, row := range rows {
			for dx := 0; dx < 5; dx++ {
				if row&(1<<(4-dx)) == 0 {
					continue
				}
				for sy := 0; sy < scale; sy++ {
					for sx := 0; sx < scale; sx++ {
						img.SetNRGBA(x+dx*scale+sx, y+dy*scale+sy, c)
					}
				}
			}
		}
		x += 6 * scale
	}
}

// font is a 5x7 pixel font for the letters the text needs, each row 5 bits.
var font = map[rune][7]uint8{
	' ': {0, 0, 0, 0, 0, 0, 0},
	'A': {0b01110, 0b10001, 0b10001, 0b11111, 0b10001, 0b10001, 0b10001},
	'C': {0b01110, 0b10001, 0b10000, 0b10000, 0b10000, 0b10001, 0b01110},
	'D': {0b11110, 0b10001, 0b10001, 0b10001, 0b10001, 0b10001, 0b11110},
	'E': {0b11111, 0b10000, 0b10000, 0b11110, 0b10000, 0b10000, 0b11111},
	'G': {0b01110, 0b10001, 0b10000, 0b10111, 0b10001, 0b10001, 0b01111},
	'I': {0b01110, 0b00100, 0b00100, 0b00100, 0b00100, 0b00100, 0b01110},
	'L': {0b10000, 0b10000, 0b10000, 0b10000, 0b10000, 0b10000, 0b11111},
	'M': {0b10001, 0b11011, 0b10101, 0b10101, 0b10001, 0b10001, 0b10001},
	'O': {0b01110, 0b10001, 0b10001, 0b10001, 0b10001, 0b10001, 0b01110},
	'P': {0b11110, 0b10001, 0b10001, 0b11110, 0b10000, 0b10000, 0b10000},
	'S': {0b01111, 0b10000, 0b10000, 0b01110, 0b00001, 0b00001, 0b11110},
	'T': {0b11111, 0b00100, 0b00100, 0b00100, 0b00100, 0b00100, 0b00100},
	'U': {0b10001, 0b10001, 0b10001, 0b10001, 0b10001, 0b10001, 0b01110},
	'?': {0b01110, 0b10001, 0b00001, 0b00010, 0b00100, 0b00000, 0b00100},
}

// Files are the test files WriteAll writes, by their short name.
type Files map[string]string

// WriteAll writes the test image and its variants into dir: the good files
// ("png", "jpg", "gif", "bmp", "tif", "ico", "apng", "jpg-oriented",
// "gif-animated", "png-as-jpg", "unicode", "spaces") and the bad ones
// ("empty", "text", "png-truncated", "png-corrupt", "svg", "heic", "tga",
// "psd", "tif-compressed", "webp-animated", "png-huge").
func WriteAll(dir string) (Files, error) {
	img := Draw()
	out := Files{}
	add := func(key, name string, enc func(w io.Writer) error) error {
		p := filepath.Join(dir, name)
		var buf bytes.Buffer
		if err := enc(&buf); err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		if err := os.WriteFile(p, buf.Bytes(), 0o644); err != nil {
			return err
		}
		out[key] = p
		return nil
	}
	steps := []struct {
		key, name string
		enc       func(w io.Writer) error
	}{
		{"png", "acc-image-upload-test.png", func(w io.Writer) error { return png.Encode(w, img) }},
		{"jpg", "acc-image-upload-test.jpg", func(w io.Writer) error { return jpeg.Encode(w, img, &jpeg.Options{Quality: 95}) }},
		{"gif", "acc-image-upload-test.gif", func(w io.Writer) error { return gif.Encode(w, img, nil) }},
		{"bmp", "acc-image-upload-test.bmp", func(w io.Writer) error { return EncodeBMP(w, img) }},
		{"tif", "acc-image-upload-test.tif", func(w io.Writer) error { return EncodeTIFF(w, img, 1) }},
		{"ico", "acc-image-upload-test.ico", func(w io.Writer) error { return EncodeICO(w, img) }},
		{"apng", "acc-image-upload-test.apng", func(w io.Writer) error { return EncodeAPNG(w, img) }},
		{"jpg-oriented", "acc-image-upload-test-oriented.jpg", func(w io.Writer) error { return EncodeJPEGOriented(w, img, 6) }},
		{"gif-animated", "acc-image-upload-test-animated.gif", func(w io.Writer) error { return EncodeAnimatedGIF(w, img) }},
		{"png-as-jpg", "acc-image-upload-test-renamed.jpg", func(w io.Writer) error { return png.Encode(w, img) }},
		{"unicode", "acc-image-upload-test-éè-日本.png", func(w io.Writer) error { return png.Encode(w, img) }},
		{"spaces", "acc image upload test.png", func(w io.Writer) error { return png.Encode(w, img) }},
		{"empty", "empty.png", func(w io.Writer) error { return nil }},
		{"text", "text.png", func(w io.Writer) error { _, err := io.WriteString(w, "this is not an image\n"); return err }},
		{"png-truncated", "truncated.png", func(w io.Writer) error { return truncatedPNG(w, img) }},
		{"png-corrupt", "corrupt.png", func(w io.Writer) error { return corruptPNG(w, img) }},
		{"svg", "drawing.svg", func(w io.Writer) error {
			_, err := io.WriteString(w, `<?xml version="1.0"?><svg xmlns="http://www.w3.org/2000/svg" width="10" height="10"><script>alert(1)</script><rect width="10" height="10" fill="red"/></svg>`)
			return err
		}},
		{"heic", "photo.heic", func(w io.Writer) error { _, err := w.Write(ftyp("heic")); return err }},
		{"tga", "texture.tga", func(w io.Writer) error { _, err := w.Write(make([]byte, 64)); return err }},
		{"psd", "layers.psd", func(w io.Writer) error { _, err := w.Write(append([]byte("8BPS"), make([]byte, 60)...)); return err }},
		{"tif-compressed", "compressed.tif", func(w io.Writer) error { return EncodeTIFF(w, img, 5) }},
		{"webp-animated", "animated.webp", func(w io.Writer) error { _, err := w.Write(AnimatedWebPHeader(Width, Height)); return err }},
		{"png-huge", "huge.png", func(w io.Writer) error { _, err := w.Write(pngHeaderOnly(9000, 9000)); return err }},
	}
	for _, s := range steps {
		if err := add(s.key, s.name, s.enc); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// EncodeBMP writes img as a 24-bit bottom-up BI_RGB bitmap.
func EncodeBMP(w io.Writer, img image.Image) error {
	b := img.Bounds()
	width, height := b.Dx(), b.Dy()
	stride := (width*3 + 3) &^ 3
	size := 14 + 40 + stride*height
	var buf bytes.Buffer
	buf.Write([]byte("BM"))
	le := binary.LittleEndian
	le.PutUint32(appendN(&buf, 4), uint32(size))
	le.PutUint32(appendN(&buf, 4), 0)
	le.PutUint32(appendN(&buf, 4), 54)
	le.PutUint32(appendN(&buf, 4), 40)
	le.PutUint32(appendN(&buf, 4), uint32(width))
	le.PutUint32(appendN(&buf, 4), uint32(height))
	le.PutUint16(appendN(&buf, 2), 1)
	le.PutUint16(appendN(&buf, 2), 24)
	le.PutUint32(appendN(&buf, 4), 0)
	le.PutUint32(appendN(&buf, 4), uint32(stride*height))
	le.PutUint32(appendN(&buf, 4), 2835)
	le.PutUint32(appendN(&buf, 4), 2835)
	le.PutUint32(appendN(&buf, 4), 0)
	le.PutUint32(appendN(&buf, 4), 0)
	row := make([]byte, stride)
	for y := b.Max.Y - 1; y >= b.Min.Y; y-- {
		for x := 0; x < width; x++ {
			r, g, bl, _ := img.At(b.Min.X+x, y).RGBA()
			row[x*3], row[x*3+1], row[x*3+2] = byte(bl>>8), byte(g>>8), byte(r>>8)
		}
		buf.Write(row)
	}
	_, err := w.Write(buf.Bytes())
	return err
}

// appendN grows buf by n zero bytes and returns them.
func appendN(buf *bytes.Buffer, n int) []byte {
	start := buf.Len()
	buf.Write(make([]byte, n))
	return buf.Bytes()[start : start+n]
}

// EncodeTIFF writes img as a little-endian baseline TIFF, 8-bit RGB in one
// strip, with the compression tag given: 1 for none, as the dashboard
// reads; another value with the same bytes, which the dashboard refuses
// by name.
func EncodeTIFF(w io.Writer, img image.Image, compression uint16) error {
	b := img.Bounds()
	width, height := b.Dx(), b.Dy()
	pixels := make([]byte, 0, width*height*3)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r, g, bl, _ := img.At(x, y).RGBA()
			pixels = append(pixels, byte(r>>8), byte(g>>8), byte(bl>>8))
		}
	}
	le := binary.LittleEndian
	var buf bytes.Buffer
	buf.Write([]byte{'I', 'I', 42, 0})
	le.PutUint32(appendN(&buf, 4), 8) // the IFD right after the header
	type entry struct {
		tag, typ uint16
		count    uint32
		value    uint32
	}
	// The bits-per-sample array (3 shorts, 6 bytes) and the pixels follow
	// the IFD: 2 + 11*12 + 4 = 138 bytes from offset 8.
	ifdEnd := uint32(8 + 2 + 11*12 + 4)
	bpsOff := ifdEnd
	dataOff := bpsOff + 6
	entries := []entry{
		{256, 4, 1, uint32(width)}, {257, 4, 1, uint32(height)}, {258, 3, 3, bpsOff}, {259, 3, 1, uint32(compression)},
		{262, 3, 1, 2}, {273, 4, 1, dataOff}, {277, 3, 1, 3}, {278, 4, 1, uint32(height)}, {279, 4, 1, uint32(len(pixels))},
		{282, 4, 1, 0}, {284, 3, 1, 1},
	}
	le.PutUint16(appendN(&buf, 2), uint16(len(entries)))
	for _, e := range entries {
		le.PutUint16(appendN(&buf, 2), e.tag)
		le.PutUint16(appendN(&buf, 2), e.typ)
		le.PutUint32(appendN(&buf, 4), e.count)
		if e.typ == 3 && e.count == 1 {
			le.PutUint16(appendN(&buf, 2), uint16(e.value))
			appendN(&buf, 2)
		} else {
			le.PutUint32(appendN(&buf, 4), e.value)
		}
	}
	le.PutUint32(appendN(&buf, 4), 0) // no next IFD
	for range 3 {
		le.PutUint16(appendN(&buf, 2), 8)
	}
	buf.Write(pixels)
	_, err := w.Write(buf.Bytes())
	return err
}

// EncodeICO writes img as an icon with one PNG image.
func EncodeICO(w io.Writer, img image.Image) error {
	var pngBuf bytes.Buffer
	if err := png.Encode(&pngBuf, img); err != nil {
		return err
	}
	b := img.Bounds()
	le := binary.LittleEndian
	var buf bytes.Buffer
	le.PutUint16(appendN(&buf, 2), 0)
	le.PutUint16(appendN(&buf, 2), 1)
	le.PutUint16(appendN(&buf, 2), 1)
	dim := func(n int) byte {
		if n >= 256 {
			return 0
		}
		return byte(n)
	}
	buf.WriteByte(dim(b.Dx()))
	buf.WriteByte(dim(b.Dy()))
	buf.WriteByte(0)
	buf.WriteByte(0)
	le.PutUint16(appendN(&buf, 2), 1)
	le.PutUint16(appendN(&buf, 2), 32)
	le.PutUint32(appendN(&buf, 4), uint32(pngBuf.Len()))
	le.PutUint32(appendN(&buf, 4), 22)
	buf.Write(pngBuf.Bytes())
	_, err := w.Write(buf.Bytes())
	return err
}

// EncodeAPNG writes img as a PNG with an acTL chunk after its IHDR: a
// one-frame animation by the APNG signature the dashboard notes, still a
// valid PNG for image/png.
func EncodeAPNG(w io.Writer, img image.Image) error {
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return err
	}
	b := buf.Bytes()
	// The IHDR chunk is the first: 8 (signature) + 4 (length) + 4 (type) + 13 + 4 (CRC).
	at := 8 + 4 + 4 + 13 + 4
	actl := chunk("acTL", []byte{0, 0, 0, 1, 0, 0, 0, 0}) // 1 frame, 0 plays
	out := append(append(append([]byte{}, b[:at]...), actl...), b[at:]...)
	_, err := w.Write(out)
	return err
}

// chunk is a PNG chunk: its length, type, data and CRC.
func chunk(typ string, data []byte) []byte {
	var buf bytes.Buffer
	binary.Write(&buf, binary.BigEndian, uint32(len(data)))
	buf.WriteString(typ)
	buf.Write(data)
	binary.Write(&buf, binary.BigEndian, crc32.ChecksumIEEE(append([]byte(typ), data...)))
	return buf.Bytes()
}

// EncodeJPEGOriented writes img rotated so that the EXIF orientation o
// turns it back: for 6, the stored image is img turned 90° clockwise and
// the tag says "rotate 90° clockwise to view" the other way round, so a
// reader applying the tag shows img.
func EncodeJPEGOriented(w io.Writer, img image.Image, o uint16) error {
	stored := img
	if o == 6 {
		stored = rotateCCW(img) // stored rotated; applying "rotate CW" restores img
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, stored, &jpeg.Options{Quality: 95}); err != nil {
		return err
	}
	b := buf.Bytes()
	if len(b) < 2 || b[0] != 0xFF || b[1] != 0xD8 {
		return fmt.Errorf("no SOI")
	}
	// APP1: FF E1, length, "Exif\0\0", a little-endian TIFF with one IFD0
	// entry, Orientation (0x0112, SHORT, 1).
	tiff := []byte{'I', 'I', 42, 0, 8, 0, 0, 0, 1, 0, 0x12, 0x01, 3, 0, 1, 0, 0, 0, byte(o), 0, 0, 0, 0, 0, 0, 0}
	payload := append([]byte("Exif\x00\x00"), tiff...)
	app1 := []byte{0xFF, 0xE1, byte((len(payload) + 2) >> 8), byte(len(payload) + 2)}
	app1 = append(app1, payload...)
	out := append(append(append([]byte{}, b[:2]...), app1...), b[2:]...)
	_, err := w.Write(out)
	return err
}

// rotateCCW turns img 90° counter-clockwise.
func rotateCCW(img image.Image) *image.NRGBA {
	b := img.Bounds()
	out := image.NewNRGBA(image.Rect(0, 0, b.Dy(), b.Dx()))
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			// (x, y) goes to (y, W-1-x) after a counter-clockwise turn.
			out.Set(y-b.Min.Y, b.Max.X-1-x, img.At(x, y))
		}
	}
	return out
}

// EncodeAnimatedGIF writes two frames: img, then img with the squares
// swapped, so the first frame is the test image.
func EncodeAnimatedGIF(w io.Writer, img image.Image) error {
	second := image.NewNRGBA(img.Bounds())
	draw.Draw(second, second.Bounds(), img, img.Bounds().Min, draw.Src)
	draw.Draw(second, Squares[0].Rect, image.NewUniform(Squares[2].RGB), image.Point{}, draw.Src)
	draw.Draw(second, Squares[2].Rect, image.NewUniform(Squares[0].RGB), image.Point{}, draw.Src)
	g := &gif.GIF{LoopCount: 0}
	for _, f := range []image.Image{img, second} {
		pal := image.NewPaletted(f.Bounds(), palette())
		draw.FloydSteinberg.Draw(pal, pal.Bounds(), f, f.Bounds().Min)
		g.Image = append(g.Image, pal)
		g.Delay = append(g.Delay, 50)
	}
	return gif.EncodeAll(w, g)
}

// palette is the colours the test image uses.
func palette() color.Palette {
	return color.Palette{color.NRGBA{255, 255, 255, 255}, color.NRGBA{0, 0, 0, 255}, Squares[0].RGB, Squares[1].RGB, Squares[2].RGB}
}

// truncatedPNG writes the first half of the PNG.
func truncatedPNG(w io.Writer, img image.Image) error {
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return err
	}
	_, err := w.Write(buf.Bytes()[:buf.Len()/2])
	return err
}

// corruptPNG writes the PNG with a byte of its first IDAT chunk changed,
// so its CRC fails.
func corruptPNG(w io.Writer, img image.Image) error {
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return err
	}
	b := buf.Bytes()
	i := bytes.Index(b, []byte("IDAT"))
	if i < 0 {
		return fmt.Errorf("no IDAT")
	}
	b[i+8] ^= 0xFF
	_, err := w.Write(b)
	return err
}

// ftyp is an ISO base media file with the brand given: what HEIC, HEIF
// and AVIF files start with.
func ftyp(brand string) []byte {
	box := []byte{0, 0, 0, 24, 'f', 't', 'y', 'p'}
	box = append(box, brand...)
	box = append(box, 0, 0, 0, 0)
	box = append(box, "mif1"...)
	return append(box, make([]byte, 64)...)
}

// AnimatedWebPHeader is a WebP container with a VP8X chunk whose animation
// flag is set and the canvas w by h: enough for the dashboard to refuse it.
func AnimatedWebPHeader(w, h int) []byte {
	le := binary.LittleEndian
	vp8x := make([]byte, 10)
	vp8x[0] = 0x02     // animation
	w1, h1 := w-1, h-1 // the canvas, 24 bits each, less one
	vp8x[4], vp8x[5], vp8x[6] = byte(w1), byte(w1>>8), byte(w1>>16)
	vp8x[7], vp8x[8], vp8x[9] = byte(h1), byte(h1>>8), byte(h1>>16)
	chunkBytes := append([]byte("VP8X"), 10, 0, 0, 0)
	chunkBytes = append(chunkBytes, vp8x...)
	anim := append([]byte("ANIM"), 6, 0, 0, 0, 0, 0, 0, 0, 0, 0)
	body := append(append([]byte("WEBP"), chunkBytes...), anim...)
	out := append([]byte("RIFF"), 0, 0, 0, 0)
	le.PutUint32(out[4:8], uint32(len(body)))
	return append(out, body...)
}

// pngHeaderOnly is a PNG signature and IHDR claiming w by h, and nothing
// else: its size is read, nothing decoded.
func pngHeaderOnly(w, h int) []byte {
	ihdr := make([]byte, 13)
	binary.BigEndian.PutUint32(ihdr[0:4], uint32(w))
	binary.BigEndian.PutUint32(ihdr[4:8], uint32(h))
	ihdr[8], ihdr[9] = 8, 6
	return append([]byte("\x89PNG\r\n\x1a\n"), chunk("IHDR", ihdr)...)
}
