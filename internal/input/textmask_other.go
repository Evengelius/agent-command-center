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
