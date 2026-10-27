//go:build !windows

package input

import (
	"errors"
	"image"
)

// TextMask needs Windows' GDI (textmask_windows.go): elsewhere the mode
// pill's image has no title.
func TextMask(text string, em, top, h int) (*image.Alpha, error) {
	return nil, errors.New("no text rasteriser outside Windows")
}

// Glyph needs Windows' GDI too: elsewhere the home screen is drawn in text
// (plan 51, design D3).
func Glyph(r rune, em int, bold bool) (*image.Alpha, error) {
	return nil, errors.New("no text rasteriser outside Windows")
}
