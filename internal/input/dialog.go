package input

import "strings"

// ImageExtensions are the extensions the Open dialog's "Images" filter
// shows (plan 31, design D2): every format the owner listed, those the
// dashboard sends and those it refuses by name, so a refusal is said rather
// than the file hidden.
var ImageExtensions = []string{
	"png", "apng", "jpg", "jpeg", "jpe", "jfif", "webp", "gif", "bmp", "dib", "tif", "tiff",
	"heic", "heif", "avif", "jxl", "svg", "svgz", "ico",
	"jp2", "j2k", "psd", "tga", "dds", "hdr", "exr", "dng", "cr2", "nef", "arw",
}

// pickFilter is the dialog's filter: its pairs of a name and patterns, each
// ended by a NUL, the list by another.
func pickFilter() string {
	pats := make([]string, len(ImageExtensions))
	for i, e := range ImageExtensions {
		pats[i] = "*." + e
	}
	return "Images\x00" + strings.Join(pats, ";") + "\x00All files\x00*.*\x00"
}
