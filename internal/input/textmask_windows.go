//go:build windows

package input

import (
	"errors"
	"image"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// The mode pill's title (plan 39, design D7): drawn by Windows' GDI in
// Cascadia Mono, the font Windows Terminal draws with, Consolas where it is
// missing, grey antialiased, as a coverage mask the render package puts in
// the pill's Sixel image. Text and an image cannot share a cell, so the
// pill's outline around its title needs the title in the image.

var (
	gdi32                     = windows.NewLazySystemDLL("gdi32.dll")
	procCreateCompatibleDC    = gdi32.NewProc("CreateCompatibleDC")
	procCreateDIBSection      = gdi32.NewProc("CreateDIBSection")
	procCreateFontW           = gdi32.NewProc("CreateFontW")
	procSelectObject          = gdi32.NewProc("SelectObject")
	procDeleteObject          = gdi32.NewProc("DeleteObject")
	procDeleteDC              = gdi32.NewProc("DeleteDC")
	procSetTextColor          = gdi32.NewProc("SetTextColor")
	procSetBkMode             = gdi32.NewProc("SetBkMode")
	procTextOutW              = gdi32.NewProc("TextOutW")
	procGdiFlush              = gdi32.NewProc("GdiFlush")
	procGetTextFaceW          = gdi32.NewProc("GetTextFaceW")
	procGetTextExtentPoint32W = gdi32.NewProc("GetTextExtentPoint32W")
	textFaces                 = []string{"Cascadia Mono", "Consolas"}
	errNoFace                 = errors.New("neither Cascadia Mono nor Consolas is installed")
	errGDI                    = errors.New("GDI could not draw the text")
)

// bitmapInfo is BITMAPINFO with its header alone, a 32-bit top-down DIB.
type bitmapInfo struct {
	size                   uint32
	width, height          int32
	planes, bitCount       uint16
	compression, sizeImage uint32
	xPels, yPels           int32
	clrUsed, clrImportant  uint32
	colours                [1]uint32
}

// GDI's constants used here.
const (
	fwNormal          = 400
	defaultCharset    = 1
	outTTPrecis       = 4
	clipDefault       = 0
	antialiasedQual   = 4 // grey antialiasing, never ClearType's colour fringes
	fixedPitchModern  = 0x31
	transparentBkMode = 1
)

// TextMask is text drawn by GDI white on black, em pixels its character
// height, from the mask's top-left corner moved down by top: each pixel's
// coverage, h pixels high and as wide as GDI measures the text.
func TextMask(text string, em, top, h int) (*image.Alpha, error) {
	if text == "" {
		return image.NewAlpha(image.Rect(0, 0, 0, h)), nil
	}
	dc, _, _ := procCreateCompatibleDC.Call(0)
	if dc == 0 {
		return nil, errGDI
	}
	defer procDeleteDC.Call(dc)
	font, err := textFont(dc, em)
	if err != nil {
		return nil, err
	}
	defer procDeleteObject.Call(font)
	u, err := syscall.UTF16FromString(text)
	if err != nil {
		return nil, err
	}
	n := len(u) - 1
	var size struct{ cx, cy int32 }
	if r, _, _ := procGetTextExtentPoint32W.Call(dc, uintptr(unsafe.Pointer(&u[0])), uintptr(n), uintptr(unsafe.Pointer(&size))); r == 0 || size.cx <= 0 {
		return nil, errGDI
	}
	w := int(size.cx)
	bi := bitmapInfo{size: 40, width: int32(w), height: -int32(h), planes: 1, bitCount: 32}
	var bits unsafe.Pointer
	dib, _, _ := procCreateDIBSection.Call(dc, uintptr(unsafe.Pointer(&bi)), 0, uintptr(unsafe.Pointer(&bits)), 0, 0)
	if dib == 0 || bits == nil {
		return nil, errGDI
	}
	defer procDeleteObject.Call(dib)
	procSelectObject.Call(dc, dib)
	procSelectObject.Call(dc, font)
	procSetTextColor.Call(dc, 0x00FFFFFF)
	procSetBkMode.Call(dc, transparentBkMode)
	if r, _, _ := procTextOutW.Call(dc, 0, uintptr(top), uintptr(unsafe.Pointer(&u[0])), uintptr(n)); r == 0 {
		return nil, errGDI
	}
	procGdiFlush.Call()
	px := unsafe.Slice((*byte)(bits), w*h*4)
	mask := image.NewAlpha(image.Rect(0, 0, w, h))
	for i := range w * h {
		b, g, r := px[i*4], px[i*4+1], px[i*4+2]
		mask.Pix[i] = max(r, g, b)
	}
	return mask, nil
}

// textFont is the first of textFaces GDI has, em pixels its character
// height, selected into dc.
func textFont(dc uintptr, em int) (uintptr, error) {
	for _, face := range textFaces {
		name, _ := syscall.UTF16PtrFromString(face)
		f, _, _ := procCreateFontW.Call(uintptr(-em), 0, 0, 0, fwNormal, 0, 0, 0, defaultCharset, outTTPrecis, clipDefault,
			antialiasedQual, fixedPitchModern, uintptr(unsafe.Pointer(name)))
		if f == 0 {
			continue
		}
		procSelectObject.Call(dc, f)
		var got [64]uint16
		procGetTextFaceW.Call(dc, uintptr(len(got)), uintptr(unsafe.Pointer(&got[0])))
		if syscall.UTF16ToString(got[:]) == face {
			return f, nil
		}
		procDeleteObject.Call(f)
	}
	return 0, errNoFace
}
