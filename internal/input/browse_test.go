package input

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// mkdirs makes each directory under root.
func mkdirs(t *testing.T, root string, dirs ...string) {
	t.Helper()
	for _, d := range dirs {
		if err := os.MkdirAll(filepath.Join(root, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
}

// TestDiskList: plan 15, criterion 3.1.
func TestDiskList(t *testing.T) {
	root := t.TempDir()
	mkdirs(t, root, "beta", "Alpha", "gamma", ".hidden", "repo/.git", "work/sub")
	if err := os.WriteFile(filepath.Join(root, "file.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	// A worktree's .git is a file.
	if err := os.WriteFile(filepath.Join(root, "work", ".git"), []byte("gitdir: x"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := Disk{}.List(root)
	if err != nil {
		t.Fatal(err)
	}
	want := []DirEntry{
		{Name: ".hidden", Hidden: true},
		{Name: "Alpha"},
		{Name: "beta"},
		{Name: "gamma"},
		{Name: "repo", Git: true},
		{Name: "work", Git: true},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("List:\n got %+v\nwant %+v", got, want)
	}

	// An empty directory lists nothing, without error.
	if got, err := (Disk{}).List(filepath.Join(root, "gamma")); err != nil || len(got) != 0 {
		t.Errorf("empty: %v, %v", got, err)
	}
	// A missing directory, or a file, gives its error.
	if _, err := (Disk{}).List(filepath.Join(root, "nope")); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("missing: %v, want not exist", err)
	}
	if _, err := (Disk{}).List(filepath.Join(root, "file.txt")); err == nil {
		t.Error("a file listed without error")
	}
}

// TestDiskListLink: a symbolic link to a directory is listed and marked; one
// to a file, or dangling, is not.
func TestDiskListLink(t *testing.T) {
	root := t.TempDir()
	mkdirs(t, root, "target")
	if err := os.WriteFile(filepath.Join(root, "f"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(root, "target"), filepath.Join(root, "link")); err != nil {
		t.Skip("symbolic links need a privilege here:", err)
	}
	_ = os.Symlink(filepath.Join(root, "f"), filepath.Join(root, "filelink"))
	_ = os.Symlink(filepath.Join(root, "gone"), filepath.Join(root, "dangling"))
	got, err := Disk{}.List(root)
	if err != nil {
		t.Fatal(err)
	}
	want := []DirEntry{{Name: "link", Link: true}, {Name: "target"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("List:\n got %+v\nwant %+v", got, want)
	}
}

func TestSortEntries(t *testing.T) {
	es := []DirEntry{{Name: "b"}, {Name: "B"}, {Name: "a"}, {Name: "_x"}, {Name: "A"}}
	SortEntries(es)
	var names []string
	for _, e := range es {
		names = append(names, e.Name)
	}
	if want := []string{"_x", "A", "a", "B", "b"}; !reflect.DeepEqual(names, want) {
		t.Errorf("sorted %v, want %v", names, want)
	}
}
