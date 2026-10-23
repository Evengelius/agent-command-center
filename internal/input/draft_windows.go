package input

import (
	"errors"
	"os"

	"golang.org/x/sys/windows"
)

// openExclusive opens path for reading and writing with no sharing, making
// it when create is set: while the file is held nobody else can open it,
// and Windows closes it when the process ends, killed or not (plan 33,
// design D1; plan 16 tells a running session the same way).
func openExclusive(path string, create bool) (*os.File, error) {
	p, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return nil, err
	}
	disposition := uint32(windows.OPEN_EXISTING)
	if create {
		disposition = windows.CREATE_NEW
	}
	h, err := windows.CreateFile(p, windows.GENERIC_READ|windows.GENERIC_WRITE, 0, nil, disposition, windows.FILE_ATTRIBUTE_NORMAL, 0)
	if err != nil {
		return nil, &os.PathError{Op: "open", Path: path, Err: err}
	}
	return os.NewFile(uintptr(h), path), nil
}

// transient says Windows refused a rename or a delete because another
// program has the file open: an antivirus, an indexer, a reader not
// sharing deletion. It lets go soon.
func transient(err error) bool {
	return errors.Is(err, windows.ERROR_ACCESS_DENIED) || errors.Is(err, windows.ERROR_SHARING_VIOLATION) || errors.Is(err, windows.ERROR_LOCK_VIOLATION)
}
