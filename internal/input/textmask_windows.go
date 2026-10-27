//go:build windows

package input

import (
	"errors"
	"image"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"unicode/utf16"
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
	procAddFontResourceExW    = gdi32.NewProc("AddFontResourceExW")
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
	fwSemiBold        = 600
	frPrivate         = 0x10 // AddFontResourceExW: for this process alone
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

// glyphScale is how many times larger Glyph draws a character before
// averaging it down: GDI's hinting snaps stems to whole pixels, which at
// the home screen's sizes draws text crisper and heavier than the
// design's smooth text (plan 51, step 3).
const glyphScale = 4

// Glyph is r drawn by GDI white on black at em pixels, semi-bold when
// bold, smoothly (glyphScale): a coverage mask as wide as GDI measures the
// character, rounded up, the character centred in it, and 3em/2 high, its
// cell's top at the mask's top (plan 51, design D3).
func Glyph(r rune, em int, bold bool) (*image.Alpha, error) {
	dc, _, _ := procCreateCompatibleDC.Call(0)
	if dc == 0 {
		return nil, errGDI
	}
	defer procDeleteDC.Call(dc)
	weight := fwNormal
	if bold {
		weight = fwSemiBold
	}
	font, err := faceFont(dc, em*glyphScale, weight)
	if err != nil {
		return nil, err
	}
	defer procDeleteObject.Call(font)
	u := utf16.Encode([]rune{r})
	var size struct{ cx, cy int32 }
	if r, _, _ := procGetTextExtentPoint32W.Call(dc, uintptr(unsafe.Pointer(&u[0])), uintptr(len(u)), uintptr(unsafe.Pointer(&size))); r == 0 || size.cx <= 0 {
		return nil, errGDI
	}
	sw, sh := (int(size.cx)+glyphScale-1)/glyphScale, em*3/2 // the mask's size
	w, h := sw*glyphScale, sh*glyphScale
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
	if r, _, _ := procTextOutW.Call(dc, uintptr((w-int(size.cx))/2), 0, uintptr(unsafe.Pointer(&u[0])), uintptr(len(u))); r == 0 {
		return nil, errGDI
	}
	procGdiFlush.Call()
	px := unsafe.Slice((*byte)(bits), w*h*4)
	mask := image.NewAlpha(image.Rect(0, 0, sw, sh))
	for y := range sh {
		for x := range sw {
			sum := 0 // the block's coverage
			for by := y * glyphScale; by < (y+1)*glyphScale; by++ {
				for bx := x * glyphScale; bx < (x+1)*glyphScale; bx++ {
					i := (by*w + bx) * 4
					sum += int(max(px[i], px[i+1], px[i+2]))
				}
			}
			mask.Pix[y*mask.Stride+x] = uint8((sum + glyphScale*glyphScale/2) / (glyphScale * glyphScale))
		}
	}
	return mask, nil
}

// terminalFont loads, once, the Cascadia Mono of the Windows Terminal this
// process runs in, for this process alone (AddFontResourceExW, FR_PRIVATE):
// the terminal ships it in its package, where GDI does not look (plan 51,
// design D3). The terminal is the first WindowsTerminal.exe among the
// process's ancestors; without one nothing is loaded.
var terminalFont = sync.OnceFunc(func() {
	dir := terminalDir()
	if dir == "" {
		return
	}
	p, err := syscall.UTF16PtrFromString(filepath.Join(dir, "CascadiaMono.ttf"))
	if err != nil {
		return
	}
	procAddFontResourceExW.Call(uintptr(unsafe.Pointer(p)), frPrivate, 0)
})

// terminalDir is the folder of the WindowsTerminal.exe among this process's
// ancestors, "" when there is none: each parent checked to have started
// before its child, so that a reused process id is not taken for one.
func terminalDir() string {
	snap, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return ""
	}
	defer windows.CloseHandle(snap)
	type proc struct {
		parent uint32
		exe    string
	}
	procs := map[uint32]proc{}
	var e windows.ProcessEntry32
	e.Size = uint32(unsafe.Sizeof(e))
	for err = windows.Process32First(snap, &e); err == nil; err = windows.Process32Next(snap, &e) {
		procs[e.ProcessID] = proc{e.ParentProcessID, windows.UTF16ToString(e.ExeFile[:])}
	}
	started := func(pid uint32) (windows.Filetime, string, bool) {
		h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, pid)
		if err != nil {
			return windows.Filetime{}, "", false
		}
		defer windows.CloseHandle(h)
		var c, x, k, u windows.Filetime
		if windows.GetProcessTimes(h, &c, &x, &k, &u) != nil {
			return windows.Filetime{}, "", false
		}
		buf := make([]uint16, windows.MAX_LONG_PATH)
		n := uint32(len(buf))
		path := ""
		if windows.QueryFullProcessImageName(h, 0, &buf[0], &n) == nil {
			path = windows.UTF16ToString(buf[:n])
		}
		return c, path, true
	}
	pid := uint32(os.Getpid())
	child, _, ok := started(pid)
	for range 16 {
		p, found := procs[pid]
		if !found || !ok {
			return ""
		}
		at, path, alive := started(p.parent)
		if !alive || at.Nanoseconds() > child.Nanoseconds() {
			return "" // the parent is gone, or a later process took its id
		}
		if strings.EqualFold(procs[p.parent].exe, "WindowsTerminal.exe") && path != "" {
			return filepath.Dir(path)
		}
		pid, child = p.parent, at
	}
	return ""
}

// textFont is the first of textFaces GDI has, em pixels its character
// height, selected into dc.
func textFont(dc uintptr, em int) (uintptr, error) {
	return faceFont(dc, em, fwNormal)
}

// faceFont is textFont at a weight, the terminal's font loaded first.
func faceFont(dc uintptr, em, weight int) (uintptr, error) {
	terminalFont()
	for _, face := range textFaces {
		name, _ := syscall.UTF16PtrFromString(face)
		f, _, _ := procCreateFontW.Call(uintptr(-em), 0, 0, 0, uintptr(weight), 0, 0, 0, defaultCharset, outTTPrecis, clipDefault,
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
