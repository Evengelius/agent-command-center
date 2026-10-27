package input

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"unsafe"

	"golang.org/x/sys/windows"
)

// TestFolderDialog is plan 52's criterion 52.3 without a window: the dialog
// created and set up as PickFolder shows it, folders only, from the file
// system, kept out of the recent items, opening at the start folder; a
// start that does not exist leaves it where it opens by itself. Showing it
// is the Windows Terminal run's.
func TestFolderDialog(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	switch err := windows.CoInitializeEx(0, windows.COINIT_APARTMENTTHREADED|windows.COINIT_DISABLE_OLE1DDE); {
	case err == nil, errors.Is(err, syscall.Errno(1)):
		defer windows.CoUninitialize()
	default:
		t.Fatalf("CoInitializeEx: %v", err)
	}
	start := filepath.Join(t.TempDir(), "my project é")
	if err := os.Mkdir(start, 0o755); err != nil {
		t.Fatal(err)
	}

	d, err := newFolderDialog(start)
	if err != nil {
		t.Fatal(err)
	}
	defer d.release()
	var opts uint32
	if hr := d.call(mGetOptions, uintptr(unsafe.Pointer(&opts))); failed(hr) {
		t.Fatal(hresultError("GetOptions", hr))
	}
	// FOS_PICKFOLDERS, FOS_FORCEFILESYSTEM, FOS_PATHMUSTEXIST and
	// FOS_DONTADDTORECENT (ShObjIdl_core.h), written out, not the code's.
	const want = 0x20 | 0x40 | 0x800 | 0x02000000
	if opts&want != want {
		t.Errorf("options 0x%08X lack 0x%08X", opts, uint32(want))
	}
	if got := folderOf(t, d); !sameFolder(t, got, start) {
		t.Errorf("the dialog opens at %q, want %q", got, start)
	}

	missing := filepath.Join(start, "missing")
	d2, err := newFolderDialog(missing)
	if err != nil {
		t.Fatal(err)
	}
	defer d2.release()
	if got := folderOf(t, d2); strings.EqualFold(got, missing) {
		t.Errorf("the dialog opens at the missing %q", got)
	}
}

// folderOf is the folder a dialog not shown yet will open at.
func folderOf(t *testing.T, d *comObject) string {
	t.Helper()
	var folder *comObject
	if hr := d.call(mGetFolder, uintptr(unsafe.Pointer(&folder))); failed(hr) || folder == nil {
		return "" // no folder set: the dialog chooses
	}
	defer folder.release()
	p, err := folder.fileSysPath()
	if err != nil {
		t.Fatal(err)
	}
	return p
}

// sameFolder compares two existing folders by their final paths, so that a
// short name or a letter's case does not differ.
func sameFolder(t *testing.T, a, b string) bool {
	t.Helper()
	ra, err := realPath(a)
	if err != nil {
		t.Fatal(err)
	}
	rb, err := realPath(b)
	if err != nil {
		t.Fatal(err)
	}
	return strings.EqualFold(ra, rb)
}
