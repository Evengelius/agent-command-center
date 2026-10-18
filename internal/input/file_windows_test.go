package input

import (
	"os/exec"
	"path/filepath"
	"testing"
)

// TestReadFileJunction: a junction needs no privilege, unlike a symbolic
// link, so the link cases of plan 17's criteria 1.2 and 1.5 always run on
// Windows: a file reached through a junction to outside the project is
// refused, and the listing does not enter it.
func TestReadFileJunction(t *testing.T) {
	root, outside := t.TempDir(), t.TempDir()
	writeAt(t, outside, "far.go", "package far\n")
	writeAt(t, root, "near.go", "package near\n")
	out, err := exec.Command("cmd", "/c", "mklink", "/J", filepath.Join(root, "junction"), outside).CombinedOutput()
	if err != nil {
		t.Fatalf("mklink /J: %v: %s", err, out)
	}
	if _, err := ReadFile(root, "junction/far.go"); reason(err) != "outside the project: junction/far.go" {
		t.Errorf("through a junction: %q", reason(err))
	}
	files, truncated, err := ListFiles(root)
	if err != nil || truncated || len(files) != 1 || files[0] != "near.go" {
		t.Errorf("ListFiles = %q, truncated %v, %v", files, truncated, err)
	}
	// The Explorer (plan 17, amendment 12) neither lists nor opens it.
	if entries, _, err := ListDir(root, ""); err != nil || len(entries) != 1 || entries[0].Name != "near.go" {
		t.Errorf("ListDir = %v, %v", entries, err)
	}
	if _, _, err := ListDir(root, "junction"); reason(err) != "outside the project: junction" {
		t.Errorf("ListDir through a junction: %q", reason(err))
	}
}
