package input

import (
	"bytes"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Evengelius/agent-command-center/internal/input/imagetest"
)

// claudeLimits are Claude Code's limits as the adapter states them.
var claudeLimits = ImageLimits{MaxCount: 100, MaxBytes: 7_400_000, MaxSide: 8000}

// decodeSent decodes the bytes as sent by their media type.
func decodeSent(t *testing.T, media string, b []byte) image.Image {
	t.Helper()
	var im image.Image
	var err error
	switch media {
	case "image/png":
		im, err = png.Decode(bytes.NewReader(b))
	case "image/jpeg":
		im, err = jpeg.Decode(bytes.NewReader(b))
	case "image/gif":
		im, err = gif.Decode(bytes.NewReader(b))
	default:
		t.Fatalf("no decoder for %s", media)
	}
	if err != nil {
		t.Fatalf("%s as sent does not decode: %v", media, err)
	}
	return im
}

// near says the colour is within tol of want on each channel.
func near(c color.Color, want color.NRGBA, tol int) bool {
	n := color.NRGBAModel.Convert(c).(color.NRGBA)
	d := func(a, b uint8) bool { x := int(a) - int(b); return x >= -tol && x <= tol }
	return d(n.R, want.R) && d(n.G, want.G) && d(n.B, want.B)
}

// checkSquares checks the three squares' centres in the image as sent, at
// its size, scaled from the drawn 320x160.
func checkSquares(t *testing.T, name string, im image.Image, tol int) {
	t.Helper()
	b := im.Bounds()
	for _, s := range imagetest.Squares {
		c := s.Rect.Min.Add(s.Rect.Max).Div(2)
		x, y := b.Min.X+c.X*b.Dx()/imagetest.Width, b.Min.Y+c.Y*b.Dy()/imagetest.Height
		if got := im.At(x, y); !near(got, s.RGB, tol) {
			t.Errorf("%s: the square at (%d, %d) is %v, want %v", name, x, y, color.NRGBAModel.Convert(got), s.RGB)
		}
	}
}

// TestImageInspect is plan 31's criterion 31.1 (design D1): the generated
// test image in every written format and the checked-in WebP are accepted
// with the right media type and size, their conversions decode to the
// drawn squares, a file is known by its bytes, and the bad files are
// refused with the designed reasons.
func TestImageInspect(t *testing.T) {
	dir := t.TempDir()
	files, err := imagetest.WriteAll(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		key, format, media, note string
		tol                      int
	}{
		{"png", "PNG", "image/png", "", 0},
		{"jpg", "JPEG", "image/jpeg", "", 24},
		{"gif", "GIF", "image/gif", "", 0},
		{"bmp", "BMP", "image/png", "converted from BMP", 0},
		{"tif", "TIFF", "image/png", "converted from TIFF", 0},
		{"ico", "ICO", "image/png", "converted from ICO", 0},
		{"apng", "PNG", "image/png", noteAnim, 0},
		{"jpg-oriented", "JPEG", "image/jpeg", noteOrient, 32},
		{"gif-animated", "GIF", "image/png", noteFirst, 0},
		{"png-as-jpg", "PNG", "image/png", "", 0},
		{"unicode", "PNG", "image/png", "", 0},
		{"spaces", "PNG", "image/png", "", 0},
	} {
		path := files[c.key]
		img, err := InspectImage(path, claudeLimits)
		if err != nil {
			t.Errorf("%s: refused: %v", c.key, err)
			continue
		}
		if img.Format != c.format || img.Media != c.media || img.Note != c.note || img.Width != imagetest.Width || img.Height != imagetest.Height {
			t.Errorf("%s: %+v, want %s sent as %s, %dx%d, note %q", c.key, img, c.format, c.media, imagetest.Width, imagetest.Height, c.note)
		}
		if img.Name != filepath.Base(path) || img.Size == 0 || img.Sent == 0 {
			t.Errorf("%s: name %q, size %d, sent %d", c.key, img.Name, img.Size, img.Sent)
		}
		got, data, err := LoadImage(path, claudeLimits)
		if err != nil || got.Sent != int64(len(data)) {
			t.Errorf("%s: load: %v, %d bytes for %d", c.key, err, len(data), got.Sent)
			continue
		}
		checkSquares(t, c.key, decodeSent(t, got.Media, data), c.tol)
	}
	// The original files are never written.
	before, _ := os.ReadFile(files["jpg-oriented"])
	if _, _, err := LoadImage(files["jpg-oriented"], claudeLimits); err != nil {
		t.Fatal(err)
	}
	if after, _ := os.ReadFile(files["jpg-oriented"]); !bytes.Equal(before, after) {
		t.Error("the oriented JPEG was changed on disk")
	}

	// The checked-in WebP, which the standard library cannot write: its
	// header read, the file sent as it is.
	webp := filepath.Join("..", "..", "testdata", "images", "acc-image-upload-test.webp")
	if img, data, err := LoadImage(webp, claudeLimits); err != nil || img.Media != "image/webp" || img.Width != imagetest.Width || img.Height != imagetest.Height {
		t.Errorf("the WebP: %+v, %v", img, err)
	} else if raw, _ := os.ReadFile(webp); !bytes.Equal(raw, data) {
		t.Error("the WebP is not sent as it is")
	}

	// Refusals, by the bytes, with their reasons.
	for key, want := range map[string]string{
		"empty":          "an empty file",
		"text":           "not an image",
		"png-truncated":  "PNG: ",
		"png-corrupt":    "PNG: ",
		"svg":            "SVG is not an image the providers take, and is never rendered or executed here",
		"heic":           "HEIC/HEIF: no decoder here; convert it to PNG or JPEG first",
		"tga":            "TGA: no decoder here",
		"psd":            "Photoshop PSD: no decoder here",
		"tif-compressed": "TIFF: LZW compression: not read",
		"webp-animated":  "animated WebP: no frame can be taken without a decoder",
		"png-huge":       "9000x9000 px: over 60 megapixels",
	} {
		_, err := InspectImage(files[key], claudeLimits)
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%s: %v, want %q", key, err, want)
		}
	}
	if _, err := InspectImage(filepath.Join(dir, "gone.png"), claudeLimits); err == nil || err.Error() != "the file is gone" {
		t.Errorf("a missing file: %v", err)
	}
	if _, err := InspectImage(dir, claudeLimits); err == nil || !strings.Contains(err.Error(), "a folder") {
		t.Errorf("a folder: %v", err)
	}
}

// TestImageFit is design D1's fitting: an image over a side, its patches or
// its bytes is scaled down to the limits and said so; a WebP, which nothing
// here decodes, is refused instead.
func TestImageFit(t *testing.T) {
	files, err := imagetest.WriteAll(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	img, data, err := LoadImage(files["png"], ImageLimits{MaxSide: 100})
	if err != nil || img.Width != 100 || img.Height != 50 || img.Note != "resized from 320x160 to 100x50 for the provider's limits" {
		t.Fatalf("over the side: %+v, %v", img, err)
	}
	checkSquares(t, "side", decodeSent(t, img.Media, data), 8)
	img, _, err = LoadImage(files["jpg"], ImageLimits{MaxPatches: 10, PatchSize: 32})
	if err != nil || ceilDiv(img.Width, 32)*ceilDiv(img.Height, 32) > 10 || img.Media != "image/jpeg" {
		t.Errorf("over the patches: %+v, %v", img, err)
	}
	img, data, err = LoadImage(files["png"], ImageLimits{MaxBytes: 600}) // the PNG is 1,177 bytes
	if err != nil || int64(len(data)) > 600 || img.Width >= imagetest.Width {
		t.Errorf("over the bytes: %+v, %d bytes, %v", img, len(data), err)
	}
	webp := filepath.Join("..", "..", "testdata", "images", "acc-image-upload-test.webp")
	if _, err := InspectImage(webp, ImageLimits{MaxSide: 100}); err == nil || !strings.Contains(err.Error(), "over the provider's 100 px a side") {
		t.Errorf("a WebP over the side: %v", err)
	}
}

// TestDraftImages is design D6: the stored draft's images are listed beside
// it, read back in order, and gone with the draft.
func TestDraftImages(t *testing.T) {
	d := Drafts{Dir: t.TempDir()}
	paths := []string{`C:\photos\a b.png`, `C:\photos\é.jpg`}
	if err := d.Save("proj", "text"); err != nil {
		t.Fatal(err)
	}
	if err := d.SaveImages("proj", paths); err != nil {
		t.Fatal(err)
	}
	got, err := d.LoadImages("proj")
	if err != nil || strings.Join(got, "|") != strings.Join(paths, "|") {
		t.Fatalf("loaded %q, %v", got, err)
	}
	if err := d.Delete("proj"); err != nil {
		t.Fatal(err)
	}
	if got, err := d.LoadImages("proj"); err != nil || got != nil {
		t.Errorf("after Delete: %q, %v", got, err)
	}
	if got, err := d.LoadImages("other"); err != nil || got != nil {
		t.Errorf("no list: %q, %v", got, err)
	}
}

// TestWriteTestImages writes the generated test images into the folder
// ACC_TEST_IMAGES names, for the Windows Terminal harness and the live test
// (plan 31, D7); skipped without the variable.
func TestWriteTestImages(t *testing.T) {
	dir := os.Getenv("ACC_TEST_IMAGES")
	if dir == "" {
		t.Skip("ACC_TEST_IMAGES not set")
	}
	files, err := imagetest.WriteAll(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("wrote %d files into %s", len(files), dir)
}

// TestSniffRefusals covers the matrix's refused formats by their bytes or,
// for TGA and the TIFF-based raws, their extension: each is named with the
// reason, none decoded (plan 31, design D1).
func TestSniffRefusals(t *testing.T) {
	tiff := []byte("II*\x00\x08\x00\x00\x00")
	ftyp := func(brand string) []byte {
		return append([]byte("\x00\x00\x00\x18ftyp"+brand+"\x00\x00\x00\x00mif1"), make([]byte, 16)...)
	}
	for _, c := range []struct {
		name string
		b    []byte
		want string
	}{
		{"photo.avif", ftyp("avif"), "AVIF: no decoder here"},
		{"photo.heif", ftyp("mif1"), "HEIC/HEIF: no decoder here"},
		{"image.jxl", []byte("\xff\x0a\x00\x00"), "JPEG XL: no decoder here"},
		{"box.jxl", []byte("\x00\x00\x00\x0cJXL \r\n\x87\n"), "JPEG XL: no decoder here"},
		{"image.jp2", []byte("\x00\x00\x00\x0cjP  \r\n\x87\n"), "JPEG 2000: no decoder here"},
		{"image.j2k", []byte("\xff\x4f\xff\x51\x00\x00"), "JPEG 2000: no decoder here"},
		{"texture.dds", []byte("DDS \x7c\x00\x00\x00"), "DDS: no decoder here"},
		{"sky.hdr", []byte("#?RADIANCE\n"), "Radiance HDR: no decoder here"},
		{"render.exr", []byte("\x76\x2f\x31\x01\x02\x00\x00\x00"), "OpenEXR: no decoder here"},
		{"raw.cr2", append(append([]byte{}, tiff...), 'C', 'R', 2, 0), "Canon CR2 raw: no decoder here"},
		{"raw.dng", tiff, "DNG raw: no decoder here"},
		{"raw.nef", tiff, "Nikon NEF raw: no decoder here"},
		{"raw.arw", tiff, "Sony ARW raw: no decoder here"},
		{"drawing.svgz", []byte("\x1f\x8b\x08\x00"), "SVG is not an image the providers take"},
		{"drawing.svg", []byte("\xef\xbb\xbf  <svg xmlns=\"http://www.w3.org/2000/svg\"/>"), "SVG is not an image the providers take"},
		{"bitmap.bmp", append([]byte("BM"), make([]byte, 60)...), "BMP: "},
		{"notes.png", []byte("PK\x03\x04 a zip"), "not an image"},
	} {
		p := filepath.Join(t.TempDir(), c.name)
		if err := os.WriteFile(p, c.b, 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := InspectImage(p, claudeLimits); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: %v, want %q", c.name, err, c.want)
		}
	}
}
