package input

import (
	"io/fs"
	"syscall"

	"golang.org/x/sys/windows"
)

// hiddenEntry is true for a hidden or system directory, as Explorer hides it.
func hiddenEntry(e fs.DirEntry) bool {
	fi, err := e.Info()
	if err != nil {
		return false
	}
	a, ok := fi.Sys().(*syscall.Win32FileAttributeData)
	return ok && a.FileAttributes&(windows.FILE_ATTRIBUTE_HIDDEN|windows.FILE_ATTRIBUTE_SYSTEM) != 0
}

// drives is the roots of the drive letters in use.
func drives() []string {
	mask, err := windows.GetLogicalDrives()
	if err != nil {
		return nil
	}
	var out []string
	for i := range 26 {
		if mask&(1<<i) != 0 {
			out = append(out, string(rune('A'+i))+`:\`)
		}
	}
	return out
}
