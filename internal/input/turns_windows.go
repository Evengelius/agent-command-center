package input

import (
	"os"
	"unsafe"

	"golang.org/x/sys/windows"
)

// makeStoreDir makes a store's folder with an access list of its own, set at
// its creation: the current user alone, in full, and nothing inherited
// (plan 65, design D3). The rest of %LOCALAPPDATA% also lets SYSTEM, the
// administrators and, on some machines, a local group in; what the folder
// holds inherits this list. SDDL "D:P(A;OICI;FA;;;<user>)": a protected
// DACL of one allow entry, inherited by files and folders, file all access.
func makeStoreDir(path string) error {
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		return err
	}
	sd, err := windows.SecurityDescriptorFromString("D:P(A;OICI;FA;;;" + user.User.Sid.String() + ")")
	if err != nil {
		return err
	}
	p, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return err
	}
	sa := &windows.SecurityAttributes{Length: uint32(unsafe.Sizeof(windows.SecurityAttributes{})), SecurityDescriptor: sd}
	if err := windows.CreateDirectory(p, sa); err != nil {
		return &os.PathError{Op: "mkdir", Path: path, Err: err}
	}
	return nil
}

// linkCount is the number of names the checked file has, its hard links
// included (plan 65, the review of 2026-10-29). It is opened for its
// attributes alone, which no sharing mode refuses, never for its content.
func linkCount(cf checkedFile) (uint32, error) {
	p, err := windows.UTF16PtrFromString(cf.real)
	if err != nil {
		return 0, err
	}
	h, err := windows.CreateFile(p, windows.FILE_READ_ATTRIBUTES,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE, nil,
		windows.OPEN_EXISTING, windows.FILE_ATTRIBUTE_NORMAL, 0)
	if err != nil {
		return 0, &os.PathError{Op: "open", Path: cf.real, Err: err}
	}
	defer windows.CloseHandle(h)
	var fi windows.ByHandleFileInformation
	if err := windows.GetFileInformationByHandle(h, &fi); err != nil {
		return 0, &os.PathError{Op: "stat", Path: cf.real, Err: err}
	}
	return fi.NumberOfLinks, nil
}

// openShared opens path to read it while other programs may rename or delete
// it: the repository's index, which a git of the owner's replaces when it
// writes it (plan 65, design D3). Go's own open does not allow the deletion.
func openShared(path string) (*os.File, error) {
	p, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return nil, err
	}
	h, err := windows.CreateFile(p, windows.GENERIC_READ,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE, nil,
		windows.OPEN_EXISTING, windows.FILE_ATTRIBUTE_NORMAL, 0)
	if err != nil {
		return nil, &os.PathError{Op: "open", Path: path, Err: err}
	}
	return os.NewFile(uintptr(h), path), nil
}
