//go:build windows

package input

import (
	"errors"
	"fmt"
	"runtime"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// The system's folder dialog (plan 52, design D1): the common item dialog,
// IFileOpenDialog, picking a file system folder, as Explorer's applications
// show it for "Select Folder". Like plan 31's Open dialog it is owned by the
// window in front at the click, the terminal, so it opens over it and keeps
// it until it closes, and it runs on its own locked OS thread with COM
// initialised for it, in a command: the screen keeps drawing.

var (
	ole32                           = windows.NewLazySystemDLL("ole32.dll")
	procCoCreateInstance            = ole32.NewProc("CoCreateInstance")
	shell32                         = windows.NewLazySystemDLL("shell32.dll")
	procSHCreateItemFromParsingName = shell32.NewProc("SHCreateItemFromParsingName")
)

// The dialog's class and the two interfaces used (ShObjIdl_core.h).
var (
	clsidFileOpenDialog = windows.GUID{Data1: 0xDC1C5A9C, Data2: 0xE88A, Data3: 0x4DDE, Data4: [8]byte{0xA5, 0xA1, 0x60, 0xF8, 0x2A, 0x20, 0xAE, 0xF7}}
	iidFileOpenDialog   = windows.GUID{Data1: 0xD57C7288, Data2: 0xD4AD, Data3: 0x4768, Data4: [8]byte{0xBE, 0x02, 0x9D, 0x96, 0x95, 0x32, 0xD9, 0x60}}
	iidShellItem        = windows.GUID{Data1: 0x43826D1E, Data2: 0xE718, Data3: 0x42EE, Data4: [8]byte{0xBC, 0x55, 0xA1, 0xE2, 0x61, 0xC3, 0x7B, 0xFE}}
)

// The methods called, by their place in the interfaces' method tables:
// IUnknown's, IModalWindow's Show, IFileDialog's, and IShellItem's
// GetDisplayName.
const (
	mRelease          = 2
	mShow             = 3
	mSetOptions       = 9
	mGetOptions       = 10
	mSetFolder        = 12
	mGetFolder        = 13
	mSetTitle         = 17
	mGetResult        = 20
	mGetDisplayName   = 5
	clsctxInprocSrv   = 0x1
	sigdnFileSysPath  = 0x80058000
	hrCancelled       = 0x800704C7 // HRESULT_FROM_WIN32(ERROR_CANCELLED): the owner closed it
	folderDialogTitle = "Choose a project directory"
)

// The dialog's options: a folder, from the file system, that exists, kept
// out of the recent items; the working directory unchanged.
const (
	fosNoChangeDir      = 0x00000008
	fosPickFolders      = 0x00000020
	fosForceFileSystem  = 0x00000040
	fosPathMustExist    = 0x00000800
	fosDontAddToRecent  = 0x02000000
	folderDialogOptions = fosPickFolders | fosForceFileSystem | fosPathMustExist | fosNoChangeDir | fosDontAddToRecent
)

// comObject is a COM interface: its first word is its method table.
type comObject struct{ vtbl *[29]uintptr }

// call calls the interface's method i, the interface first, and gives its
// HRESULT. As with LazyProc.Call, a pointer converted to uintptr among args
// is moved to the heap and kept alive for the call.
//
//go:uintptrescapes
func (o *comObject) call(i int, args ...uintptr) uintptr {
	r, _, _ := syscall.SyscallN(o.vtbl[i], append([]uintptr{uintptr(unsafe.Pointer(o))}, args...)...)
	return r
}

func (o *comObject) release() { o.call(mRelease) }

// failed is an HRESULT's failure bit.
func failed(hr uintptr) bool { return int32(uint32(hr)) < 0 }

func hresultError(what string, hr uintptr) error {
	return fmt.Errorf("%s (HRESULT 0x%08X)", what, uint32(hr))
}

// PickFolder opens the system's folder dialog over the window in front, at
// start when it is a folder, and gives the folder chosen; cancelling gives
// "" and no error.
func PickFolder(start string) (string, error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	// COM in an apartment of this thread, as the dialog needs; S_FALSE,
	// already initialised, is undone as well.
	switch err := windows.CoInitializeEx(0, windows.COINIT_APARTMENTTHREADED|windows.COINIT_DISABLE_OLE1DDE); {
	case err == nil, errors.Is(err, syscall.Errno(1)):
		defer windows.CoUninitialize()
	}
	d, err := newFolderDialog(start)
	if err != nil {
		return "", err
	}
	defer d.release()
	switch hr := d.call(mShow, uintptr(windows.GetForegroundWindow())); {
	case hr == hrCancelled:
		return "", nil
	case failed(hr):
		return "", hresultError("the dialog failed", hr)
	}
	var item *comObject
	if hr := d.call(mGetResult, uintptr(unsafe.Pointer(&item))); failed(hr) {
		return "", hresultError("no folder from the dialog", hr)
	}
	defer item.release()
	return item.fileSysPath()
}

// newFolderDialog is the dialog set up as PickFolder shows it, not shown:
// its options, its title, and start as the folder it opens at, when start
// is one. The caller releases it.
func newFolderDialog(start string) (*comObject, error) {
	var d *comObject
	hr, _, _ := procCoCreateInstance.Call(uintptr(unsafe.Pointer(&clsidFileOpenDialog)), 0, clsctxInprocSrv,
		uintptr(unsafe.Pointer(&iidFileOpenDialog)), uintptr(unsafe.Pointer(&d)))
	if failed(hr) {
		return nil, hresultError("the folder dialog is not available", hr)
	}
	var opts uint32
	if hr := d.call(mGetOptions, uintptr(unsafe.Pointer(&opts))); failed(hr) {
		d.release()
		return nil, hresultError("the folder dialog's options", hr)
	}
	if hr := d.call(mSetOptions, uintptr(opts|folderDialogOptions)); failed(hr) {
		d.release()
		return nil, hresultError("the folder dialog's options", hr)
	}
	title, err := windows.UTF16PtrFromString(folderDialogTitle)
	if err != nil {
		d.release()
		return nil, err
	}
	d.call(mSetTitle, uintptr(unsafe.Pointer(title))) // the system's title otherwise
	if start != "" {
		// A start that is no folder leaves the dialog where it opens by itself.
		if folder, err := shellItem(start); err == nil {
			d.call(mSetFolder, uintptr(unsafe.Pointer(folder)))
			folder.release()
		}
	}
	return d, nil
}

// shellItem is the shell's item for path; the caller releases it.
func shellItem(path string) (*comObject, error) {
	p, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return nil, err
	}
	var item *comObject
	hr, _, _ := procSHCreateItemFromParsingName.Call(uintptr(unsafe.Pointer(p)), 0,
		uintptr(unsafe.Pointer(&iidShellItem)), uintptr(unsafe.Pointer(&item)))
	if failed(hr) {
		return nil, hresultError("no shell item for "+path, hr)
	}
	return item, nil
}

// fileSysPath is a shell item's file system path.
func (o *comObject) fileSysPath() (string, error) {
	var p *uint16
	if hr := o.call(mGetDisplayName, sigdnFileSysPath, uintptr(unsafe.Pointer(&p))); failed(hr) {
		return "", hresultError("the folder's path", hr)
	}
	defer windows.CoTaskMemFree(unsafe.Pointer(p))
	return windows.UTF16PtrToString(p), nil
}
