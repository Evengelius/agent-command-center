package input

import (
	"io/fs"
	"strings"
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

// realPath is p with every symbolic link and junction resolved, which
// filepath.EvalSymlinks does not do for a junction (plan 17, Reading a file):
// the system's final path of a handle opened for its attributes only, so the
// file's content is never opened.
func realPath(p string) (string, error) {
	const volumeNameDOS = 0 // FILE_NAME_NORMALIZED | VOLUME_NAME_DOS, a drive letter path
	name, err := windows.UTF16PtrFromString(p)
	if err != nil {
		return "", err
	}
	h, err := windows.CreateFile(name, windows.FILE_READ_ATTRIBUTES,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE, nil,
		windows.OPEN_EXISTING, windows.FILE_FLAG_BACKUP_SEMANTICS, 0)
	if err != nil {
		return "", &fs.PathError{Op: "open", Path: p, Err: err}
	}
	defer windows.CloseHandle(h)
	buf := make([]uint16, windows.MAX_LONG_PATH)
	n, err := windows.GetFinalPathNameByHandle(h, &buf[0], uint32(len(buf)), volumeNameDOS)
	if err != nil {
		return "", err
	}
	if int(n) > len(buf) {
		return "", syscall.ENAMETOOLONG
	}
	return dosPath(windows.UTF16ToString(buf[:n])), nil
}

// dosPath is a final path name without the `\\?\` prefix the system gives
// it: a drive letter path, or `\\server\share\…` for `\\?\UNC\server\share\…`
// (plan 47, design D5).
func dosPath(s string) string {
	if rest, ok := strings.CutPrefix(s, `\\?\UNC\`); ok {
		return `\\` + rest
	}
	return strings.TrimPrefix(s, `\\?\`)
}
