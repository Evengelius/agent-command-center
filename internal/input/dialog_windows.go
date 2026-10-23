//go:build windows

package input

import (
	"errors"
	"fmt"
	"path/filepath"
	"runtime"
	"syscall"
	"unicode/utf16"
	"unsafe"

	"golang.org/x/sys/windows"
)

// The system's Open dialog (plan 31, design D2): GetOpenFileNameW with the
// Explorer flag and no hook or template, which Windows shows as its common
// item dialog, the one Explorer's applications use. It is owned by the
// window in front at the click, the terminal, so it opens over it and keeps
// it until it closes. It runs on its own locked OS thread with COM
// initialised for it, in a command: the dashboard keeps drawing.

var (
	comdlg32                 = windows.NewLazySystemDLL("comdlg32.dll")
	procGetOpenFileNameW     = comdlg32.NewProc("GetOpenFileNameW")
	procCommDlgExtendedError = comdlg32.NewProc("CommDlgExtendedError")
)

// openFileName is OPENFILENAMEW, as laid out on 64-bit Windows.
type openFileName struct {
	structSize    uint32
	owner         uintptr
	instance      uintptr
	filter        *uint16
	customFilter  *uint16
	maxCustFilter uint32
	filterIndex   uint32
	file          *uint16
	maxFile       uint32
	fileTitle     *uint16
	maxFileTitle  uint32
	initialDir    *uint16
	title         *uint16
	flags         uint32
	fileOffset    uint16
	fileExtension uint16
	defExt        *uint16
	custData      uintptr
	hook          uintptr
	templateName  *uint16
	reserved      uintptr
	reserved2     uint32
	flagsEx       uint32
}

// The dialog's flags: the Explorer dialog, several files, existing ones
// only, the working directory unchanged, no read-only box, and the files
// chosen kept out of the recent documents.
const (
	ofnHideReadOnly     = 0x00000004
	ofnNoChangeDir      = 0x00000008
	ofnAllowMultiSelect = 0x00000200
	ofnPathMustExist    = 0x00000800
	ofnFileMustExist    = 0x00001000
	ofnExplorer         = 0x00080000
	ofnDontAddToRecent  = 0x02000000
	fnerrBufferTooSmall = 0x3003
	pickBuffer          = 1 << 16 // UTF-16 units for the chosen paths
)

// PickImages opens the system's Open dialog for images over the window in
// front and gives the files chosen; cancelling gives none and no error.
func PickImages() ([]string, error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	// COM in an apartment of this thread, as the dialog's shell views need;
	// S_FALSE, already initialised, is undone as well.
	switch err := windows.CoInitializeEx(0, windows.COINIT_APARTMENTTHREADED|windows.COINIT_DISABLE_OLE1DDE); {
	case err == nil, errors.Is(err, syscall.Errno(1)):
		defer windows.CoUninitialize()
	}
	filter := utf16With0(pickFilter())
	title := utf16With0("Upload from computer")
	buf := make([]uint16, pickBuffer)
	ofn := openFileName{
		owner:       uintptr(windows.GetForegroundWindow()),
		filter:      &filter[0],
		filterIndex: 1,
		file:        &buf[0],
		maxFile:     uint32(len(buf)),
		title:       &title[0],
		flags:       ofnExplorer | ofnAllowMultiSelect | ofnFileMustExist | ofnPathMustExist | ofnNoChangeDir | ofnHideReadOnly | ofnDontAddToRecent,
	}
	ofn.structSize = uint32(unsafe.Sizeof(ofn))
	ok, _, _ := procGetOpenFileNameW.Call(uintptr(unsafe.Pointer(&ofn)))
	runtime.KeepAlive(filter)
	runtime.KeepAlive(title)
	runtime.KeepAlive(buf)
	if ok == 0 {
		code, _, _ := procCommDlgExtendedError.Call()
		switch code {
		case 0:
			return nil, nil // cancelled
		case fnerrBufferTooSmall:
			return nil, errors.New("too many files chosen at once")
		}
		return nil, fmt.Errorf("the dialog failed (error 0x%04X)", code)
	}
	return pickedPaths(buf), nil
}

// pickedPaths reads the dialog's buffer: one full path, or a folder and
// the names chosen in it, each ended by a NUL, the list by another.
func pickedPaths(buf []uint16) []string {
	var parts []string
	for start, i := 0, 0; i < len(buf); i++ {
		if buf[i] != 0 {
			continue
		}
		if i == start {
			break
		}
		parts = append(parts, string(utf16.Decode(buf[start:i])))
		start = i + 1
	}
	switch len(parts) {
	case 0:
		return nil
	case 1:
		return parts
	}
	out := make([]string, 0, len(parts)-1)
	for _, name := range parts[1:] {
		out = append(out, filepath.Join(parts[0], name))
	}
	return out
}

// utf16With0 is s in UTF-16 with a closing NUL; its own NULs are kept, as
// the filter's pairs need.
func utf16With0(s string) []uint16 { return append(utf16.Encode([]rune(s)), 0) }
