package input

import (
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"golang.org/x/sys/windows"
)

// TestRealPathDOS is plan 47's criterion 47.7 (design D5): realPath gives a
// drive letter path, without the `\\?\` prefix of the system's final path,
// absolute and the same when resolved again.
func TestRealPathDOS(t *testing.T) {
	dir := t.TempDir()
	got, err := realPath(dir)
	if err != nil {
		t.Fatal(err)
	}
	if strings.HasPrefix(got, `\\?\`) || !filepath.IsAbs(got) || len(filepath.VolumeName(got)) != 2 {
		t.Errorf("realPath(%q) = %q, want a drive letter path", dir, got)
	}
	if again, err := realPath(got); err != nil || again != got {
		t.Errorf("resolved again: %q, %v; want %q", again, err, got)
	}
	// A share's final path keeps its two leading backslashes.
	for in, want := range map[string]string{
		`\\?\C:\Users\x`:           `C:\Users\x`,
		`\\?\UNC\server\share\dir`: `\\server\share\dir`,
		`C:\plain`:                 `C:\plain`,
	} {
		if got := dosPath(in); got != want {
			t.Errorf("dosPath(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestDiskListWindows: hidden and system attributes, a junction, the drives
// (plan 15, criterion 3.1).
func TestDiskListWindows(t *testing.T) {
	root := t.TempDir()
	mkdirs(t, root, "attrHidden", "attrSystem", "plain", "target")
	for name, a := range map[string]uint32{"attrHidden": windows.FILE_ATTRIBUTE_HIDDEN, "attrSystem": windows.FILE_ATTRIBUTE_SYSTEM} {
		p, _ := windows.UTF16PtrFromString(filepath.Join(root, name))
		if err := windows.SetFileAttributes(p, a|windows.FILE_ATTRIBUTE_DIRECTORY); err != nil {
			t.Fatal(err)
		}
	}
	// A junction needs no privilege, unlike a symbolic link.
	out, err := exec.Command("cmd", "/c", "mklink", "/J", filepath.Join(root, "junction"), filepath.Join(root, "target")).CombinedOutput()
	if err != nil {
		t.Fatalf("mklink /J: %v: %s", err, out)
	}
	got, err := Disk{}.List(root)
	if err != nil {
		t.Fatal(err)
	}
	want := []DirEntry{
		{Name: "attrHidden", Hidden: true},
		{Name: "attrSystem", Hidden: true},
		{Name: "junction", Link: true},
		{Name: "plain"},
		{Name: "target"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("List:\n got %+v\nwant %+v", got, want)
	}

	ds := Disk{}.Drives()
	sysDir, err := windows.GetSystemDirectory()
	if err != nil {
		t.Fatal(err)
	}
	sys := strings.ToUpper(filepath.VolumeName(sysDir)) + `\`
	found := false
	for _, d := range ds {
		found = found || d == sys
	}
	if !found {
		t.Errorf("Drives %v lack the system drive %s", ds, sys)
	}
}
