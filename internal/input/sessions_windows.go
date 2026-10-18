package input

import (
	"errors"
	"os"

	"golang.org/x/sys/windows"
)

// held is true while another handle keeps path open: opening it unshared
// then fails with a sharing violation (plan 16).
func held(path string) (bool, error) {
	p, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return false, err
	}
	h, err := windows.CreateFile(p, windows.GENERIC_READ, 0, nil, windows.OPEN_EXISTING, windows.FILE_ATTRIBUTE_NORMAL, 0)
	switch {
	case errors.Is(err, windows.ERROR_SHARING_VIOLATION):
		return true, nil
	case errors.Is(err, windows.ERROR_FILE_NOT_FOUND):
		return true, nil // released meanwhile: not ours to delete
	case err != nil:
		return false, &os.PathError{Op: "open", Path: path, Err: err}
	}
	windows.CloseHandle(h)
	return false, nil
}
