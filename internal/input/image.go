package input

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"image"
	"image/draw"
	"image/gif"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strings"
)

// An image attached to the draft (plan 31, brief amendment 31; design D1):
// found by its bytes, never its extension, decoded to be sure of it,
// converted to PNG when the providers do not take its format, and refused
// by name when no decoder here reads it. The decoders are Go's standard
// library (PNG, JPEG, GIF) and the dashboard's own (BMP, baseline TIFF,
// ICO, the WebP header): the dependencies stay the approved ones. The
// original file is never written; nothing of its bytes is kept after an
// inspection or logged.

// ImageLimits are a provider's limits on the images it takes
// (doc/research/image-input.md); 0 is no limit.
type ImageLimits struct {
	MaxBytes   int64 // a file as sent
	MaxCount   int   // images a message
	MaxSide    int   // pixels a side
	MaxPatches int   // patches of PatchSize px an image
	PatchSize  int   // the patch's side
}

// Image is an attached image as inspected.
type Image struct {
	Path   string
	Name   string // the file's name
	Size   int64  // the file's bytes
	Sent   int64  // the bytes as sent, after a conversion
	Format string // the file's format: PNG, JPEG, GIF, WebP, BMP, TIFF, ICO
	Media  string // the media type as sent: image/png, image/jpeg, image/gif, image/webp
	Width  int
	Height int
	Note   string   // what the sending changes: "animated: …", "orientation applied", "converted from BMP"
	Sum    [32]byte // the SHA-256 of the file's or the pasted bytes, as read (plan 33, design D7)
}

const (
	maxInspect = 64 << 20    // bytes a file may have to be read as an image at all
	maxPixels  = 60_000_000  // pixels an image may have to be decoded: a 50 MP camera image
	pngMedia   = "image/png" // the media type of a conversion
	noteAnim   = "animated: the first frame is what the model sees"
	noteFirst  = "animated: the first frame is sent"
	noteOrient = "orientation applied"
)

// InspectImage reads and validates the file: its format by its bytes, its
// decoding, its conversion when the providers do not take it as is, and
// the limits; it keeps nothing of the bytes.
func InspectImage(path string, lim ImageLimits) (Image, error) {
	img, _, _, err := readImage(path, lim)
	return img, err
}

// InspectPreview is InspectImage with the decoded image, for the preview's
// thumbnail (plan 32, design D4); nil for a WebP, which nothing here
// decodes.
func InspectPreview(path string, lim ImageLimits) (Image, image.Image, error) {
	img, _, im, err := readImage(path, lim)
	return img, im, err
}

// LoadImage is the image's bytes as sent, read and checked again now: the
// file may have changed or gone since it was attached.
func LoadImage(path string, lim ImageLimits) (Image, []byte, error) {
	img, data, _, err := readImage(path, lim)
	return img, data, err
}

// InspectData reads and checks a pasted image's bytes as InspectImage does
// a file's (plan 32, design D3): name is what it is shown and sent as; the
// bytes as sent and the decoded image come back with it. Nothing is
// written anywhere.
func InspectData(name string, b []byte, lim ImageLimits) (Image, []byte, image.Image, error) {
	img := Image{Name: name, Size: int64(len(b)), Sum: sha256.Sum256(b)}
	switch {
	case len(b) == 0:
		return img, nil, nil, errors.New("an empty image")
	case img.Size > maxInspect:
		return img, nil, nil, fmt.Errorf("%s: too large to read as an image (over %s)", FmtBytes(img.Size), FmtBytes(maxInspect))
	}
	return decodeImage(img, b, name, lim)
}

// Unreachable says a file's drive or share cannot be reached at all (a
// network share gone offline, a drive unplugged), as against a file gone
// from a folder that answers (plan 33). Windows reports a server it cannot
// find as a path that does not exist, so a failed read alone cannot tell.
func Unreachable(path string) bool {
	vol := filepath.VolumeName(path)
	if vol == "" {
		return false
	}
	_, err := os.Stat(vol + string(filepath.Separator))
	return err != nil
}

// readImage reads the file and does what InspectImage and LoadImage share.
func readImage(path string, lim ImageLimits) (Image, []byte, image.Image, error) {
	img := Image{Path: path, Name: filepath.Base(path)}
	st, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return img, nil, nil, errors.New("the file is gone")
		}
		return img, nil, nil, errors.New("the file cannot be read: " + err.Error())
	}
	if st.IsDir() {
		return img, nil, nil, errors.New("a folder, not an image")
	}
	img.Size = st.Size()
	if img.Size == 0 {
		return img, nil, nil, errors.New("an empty file")
	}
	if img.Size > maxInspect {
		return img, nil, nil, fmt.Errorf("%s: too large to read as an image (over %s)", FmtBytes(img.Size), FmtBytes(maxInspect))
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return img, nil, nil, errors.New("the file cannot be read: " + err.Error())
	}
	img.Sum = sha256.Sum256(b)
	return decodeImage(img, b, path, lim)
}

// decodeImage is what a file's reading and a pasted image's share: the
// format by the bytes, the decoding and the conversion, the limits. The
// decoded image comes back too, nil for a WebP.
func decodeImage(img Image, b []byte, path string, lim ImageLimits) (Image, []byte, image.Image, error) {
	format, err := sniffImage(b, path)
	if err != nil {
		return img, nil, nil, err
	}
	img.Format = format
	var data []byte
	var im image.Image // the decoded image, nil for a WebP: none is decoded
	switch format {
	case "PNG":
		data, im, err = inspectPNG(&img, b)
	case "JPEG":
		data, im, err = inspectJPEG(&img, b)
	case "GIF":
		data, im, err = inspectGIF(&img, b)
	case "WebP":
		data, err = inspectWebP(&img, b)
	case "BMP":
		data, im, err = convertWith(&img, b, "BMP", decodeBMP)
	case "TIFF":
		data, im, err = convertWith(&img, b, "TIFF", decodeTIFF)
	case "ICO":
		data, im, err = convertWith(&img, b, "ICO", decodeICO)
	default:
		err = errors.New("not an image")
	}
	if err != nil {
		return img, nil, nil, err
	}
	img.Sent = int64(len(data))
	if err := img.check(lim); err != nil {
		// Over a limit a smaller copy can meet: the image is scaled down to
		// the provider's limits, as the models would scale it anyway, and
		// said so; a WebP, which nothing here decodes, stays refused.
		if im == nil {
			return img, nil, nil, err
		}
		if data, err = fit(&img, im, lim); err != nil {
			return img, nil, nil, err
		}
		img.Sent = int64(len(data))
		if err := img.check(lim); err != nil {
			return img, nil, nil, err
		}
	}
	return img, data, im, nil
}

// check applies the provider's limits.
func (img Image) check(lim ImageLimits) error {
	if lim.MaxSide > 0 && (img.Width > lim.MaxSide || img.Height > lim.MaxSide) {
		return fmt.Errorf("%dx%d px: over the provider's %d px a side", img.Width, img.Height, lim.MaxSide)
	}
	if lim.MaxPatches > 0 && lim.PatchSize > 0 {
		p := ceilDiv(img.Width, lim.PatchSize) * ceilDiv(img.Height, lim.PatchSize)
		if p > lim.MaxPatches {
			return fmt.Errorf("%dx%d px, %d patches of %d px: over the provider's %d", img.Width, img.Height, p, lim.PatchSize, lim.MaxPatches)
		}
	}
	if lim.MaxBytes > 0 && img.Sent > lim.MaxBytes {
		return fmt.Errorf("%s as sent: over the provider's %s a file", FmtBytes(img.Sent), FmtBytes(lim.MaxBytes))
	}
	return nil
}

func ceilDiv(a, b int) int { return (a + b - 1) / b }

// unsupported is the refusal of a format no decoder here reads.
func unsupported(format string) error {
	return fmt.Errorf("%s: no decoder here; convert it to PNG or JPEG first", format)
}

// sniffImage names the file's format by its first bytes, or refuses it: a
// format no decoder here reads by name, SVG as never rendered, anything
// else as not an image. TGA, which has no signature, is known by its
// extension; the camera raws on TIFF's signature by theirs or their marks.
func sniffImage(b []byte, path string) (string, error) {
	ext := strings.ToLower(filepath.Ext(path))
	has := func(off int, sig string) bool { return len(b) >= off+len(sig) && string(b[off:off+len(sig)]) == sig }
	switch {
	case has(0, "\x89PNG\r\n\x1a\n"):
		return "PNG", nil
	case has(0, "\xff\xd8\xff"):
		return "JPEG", nil
	case has(0, "GIF87a") || has(0, "GIF89a"):
		return "GIF", nil
	case has(0, "RIFF") && has(8, "WEBP"):
		return "WebP", nil
	case has(0, "BM"):
		return "BMP", nil
	case has(0, "II*\x00") || has(0, "MM\x00*"):
		if has(8, "CR\x02") {
			return "", unsupported("Canon CR2 raw")
		}
		if raw := map[string]string{".dng": "DNG raw", ".nef": "Nikon NEF raw", ".arw": "Sony ARW raw", ".cr2": "Canon CR2 raw"}[ext]; raw != "" {
			return "", unsupported(raw)
		}
		return "TIFF", nil
	case has(0, "\x00\x00\x01\x00"):
		return "ICO", nil
	case has(4, "ftyp") && len(b) >= 12:
		brand := string(b[8:12])
		switch brand {
		case "avif", "avis":
			return "", unsupported("AVIF")
		case "heic", "heix", "hevc", "hevx", "heim", "heis", "hevm", "hevs", "mif1", "msf1":
			return "", unsupported("HEIC/HEIF")
		}
		return "", unsupported("an ISO media file (" + strings.TrimSpace(brand) + ")")
	case has(0, "\x00\x00\x00\x0cjP  \r\n\x87\n") || has(0, "\xff\x4f\xff\x51"):
		return "", unsupported("JPEG 2000")
	case has(0, "\xff\x0a") || has(0, "\x00\x00\x00\x0cJXL \r\n\x87\n"):
		return "", unsupported("JPEG XL")
	case has(0, "8BPS"):
		return "", unsupported("Photoshop PSD")
	case has(0, "DDS "):
		return "", unsupported("DDS")
	case has(0, "#?RADIANCE") || has(0, "#?RGBE"):
		return "", unsupported("Radiance HDR")
	case has(0, "\x76\x2f\x31\x01"):
		return "", unsupported("OpenEXR")
	case isSVG(b) || has(0, "\x1f\x8b") && ext == ".svgz":
		return "", errors.New("SVG is not an image the providers take, and is never rendered or executed here")
	case ext == ".tga":
		return "", unsupported("TGA")
	}
	return "", errors.New("not an image")
}

// isSVG says the bytes start an SVG document: XML or an svg element.
func isSVG(b []byte) bool {
	s := strings.TrimLeft(string(bytes.TrimPrefix(b[:min(len(b), 512)], []byte("\xef\xbb\xbf"))), " \t\r\n")
	l := strings.ToLower(s)
	if strings.HasPrefix(l, "<svg") || strings.HasPrefix(l, "<!doctype svg") {
		return true
	}
	return strings.HasPrefix(l, "<?xml") && strings.Contains(l, "<svg")
}

// tooBig refuses an image whose header's size would take too much memory
// to decode, before decoding.
func tooBig(w, h int) error {
	if w <= 0 || h <= 0 {
		return fmt.Errorf("%dx%d px: not a size", w, h)
	}
	if w*h > maxPixels {
		return fmt.Errorf("%dx%d px: over %d megapixels", w, h, maxPixels/1_000_000)
	}
	return nil
}

func inspectPNG(img *Image, b []byte) ([]byte, image.Image, error) {
	cfg, err := png.DecodeConfig(bytes.NewReader(b))
	if err != nil {
		return nil, nil, errors.New("PNG: " + err.Error())
	}
	if err := tooBig(cfg.Width, cfg.Height); err != nil {
		return nil, nil, err
	}
	im, err := png.Decode(bytes.NewReader(b))
	if err != nil {
		return nil, nil, errors.New("PNG: " + err.Error())
	}
	img.Width, img.Height, img.Media = cfg.Width, cfg.Height, pngMedia
	if hasChunk(b, "acTL") {
		img.Note = noteAnim
	}
	return b, im, nil
}

// hasChunk says the PNG holds a chunk of the type given.
func hasChunk(b []byte, typ string) bool {
	for i := 8; i+8 <= len(b); {
		n := int(uint32(b[i])<<24 | uint32(b[i+1])<<16 | uint32(b[i+2])<<8 | uint32(b[i+3]))
		if string(b[i+4:i+8]) == typ {
			return true
		}
		if string(b[i+4:i+8]) == "IEND" || n < 0 {
			return false
		}
		i += 12 + n
	}
	return false
}

func inspectJPEG(img *Image, b []byte) ([]byte, image.Image, error) {
	cfg, err := jpeg.DecodeConfig(bytes.NewReader(b))
	if err != nil {
		return nil, nil, errors.New("JPEG: " + err.Error())
	}
	if err := tooBig(cfg.Width, cfg.Height); err != nil {
		return nil, nil, err
	}
	im, err := jpeg.Decode(bytes.NewReader(b))
	if err != nil {
		return nil, nil, errors.New("JPEG: " + err.Error())
	}
	img.Width, img.Height, img.Media = cfg.Width, cfg.Height, "image/jpeg"
	if o := exifOrientation(b); o >= 2 && o <= 8 {
		// The providers read no metadata: the rotation is applied here.
		turned := orient(im, o)
		var buf bytes.Buffer
		if err := jpeg.Encode(&buf, turned, &jpeg.Options{Quality: 95}); err != nil {
			return nil, nil, errors.New("JPEG: " + err.Error())
		}
		img.Width, img.Height, img.Note = turned.Bounds().Dx(), turned.Bounds().Dy(), noteOrient
		return buf.Bytes(), turned, nil
	}
	return b, im, nil
}

func inspectGIF(img *Image, b []byte) ([]byte, image.Image, error) {
	cfg, err := gif.DecodeConfig(bytes.NewReader(b))
	if err != nil {
		return nil, nil, errors.New("GIF: " + err.Error())
	}
	if err := tooBig(cfg.Width, cfg.Height); err != nil {
		return nil, nil, err
	}
	// The first frame decoded, the others only walked (plan 49, design D12).
	first, err := gif.Decode(bytes.NewReader(b))
	if err != nil {
		return nil, nil, errors.New("GIF: " + err.Error())
	}
	frames, err := gifFrames(b)
	if err != nil {
		return nil, nil, errors.New("GIF: " + err.Error())
	}
	img.Width, img.Height = cfg.Width, cfg.Height
	// The first frame on the canvas: what is sent of an animated GIF, as PNG,
	// said so, and what a resize starts from.
	canvas := image.NewNRGBA(image.Rect(0, 0, cfg.Width, cfg.Height))
	draw.Draw(canvas, first.Bounds(), first, first.Bounds().Min, draw.Over)
	if frames == 1 {
		img.Media = "image/gif"
		return b, canvas, nil
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, canvas); err != nil {
		return nil, nil, errors.New("GIF: " + err.Error())
	}
	img.Media, img.Note = pngMedia, noteFirst
	return buf.Bytes(), canvas, nil
}

// convertWith decodes the file with one of the dashboard's own decoders and
// gives its PNG.
func convertWith(img *Image, b []byte, format string, decode func([]byte) (*image.NRGBA, error)) ([]byte, image.Image, error) {
	im, err := decode(b)
	if err != nil {
		return nil, nil, errors.New(format + ": " + err.Error())
	}
	if err := tooBig(im.Bounds().Dx(), im.Bounds().Dy()); err != nil {
		return nil, nil, err
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, im); err != nil {
		return nil, nil, errors.New(format + ": " + err.Error())
	}
	img.Width, img.Height, img.Media, img.Note = im.Bounds().Dx(), im.Bounds().Dy(), pngMedia, "converted from "+format
	return buf.Bytes(), im, nil
}

// FmtBytes writes a size as the chips show it: "456 B", "12 KB", "1.2 MB".
func FmtBytes(n int64) string {
	switch {
	case n < 1000:
		return fmt.Sprintf("%d B", n)
	case n < 1000*1000:
		return fmt.Sprintf("%d KB", (n+500)/1000)
	default:
		return fmt.Sprintf("%.1f MB", float64(n)/1e6)
	}
}
