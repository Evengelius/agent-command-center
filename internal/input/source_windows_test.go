package input

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestSourceJunction: a junction needs no privilege, unlike a symbolic link,
// so the link cases of plan 44's criterion 44.6 always run on Windows: a
// file reached through a junction to outside the project is refused, the
// junction itself is no file, and the untracked listing does not enter it.
func TestSourceJunction(t *testing.T) {
	root, outside := t.TempDir(), t.TempDir()
	writeAt(t, outside, "far.txt", "far\n")
	writeAt(t, root, "dir/near.txt", "near\n")
	out, err := exec.Command("cmd", "/c", "mklink", "/J", filepath.Join(root, "dir", "junction"), outside).CombinedOutput()
	if err != nil {
		t.Fatalf("mklink /J: %v: %s", err, out)
	}
	s, err := workingSide(root, root, "dir/junction/far.txt")
	if err == nil || !strings.Contains(err.Error(), "outside the project") || s.Data != nil {
		t.Errorf("through a junction: %+v, %v", s, err)
	}
	s, err = workingSide(root, root, "dir/junction")
	if err == nil || s.Data != nil {
		t.Errorf("the junction itself: %+v, %v", s, err)
	}
	if s, err = workingSide(root, root, "dir/near.txt"); err != nil || string(s.Data) != "near\n" {
		t.Errorf("a file beside it: %+v, %v", s, err)
	}
	files, truncated, err := UntrackedFiles(root, "dir/", 100)
	if err != nil || truncated || len(files) != 1 || files[0] != "dir/near.txt" {
		t.Errorf("UntrackedFiles = %q, truncated %v, %v", files, truncated, err)
	}
	if _, _, err := UntrackedFiles(root, "dir/junction", 100); err == nil || !strings.Contains(err.Error(), "outside the project") {
		t.Errorf("UntrackedFiles through a junction: %v", err)
	}
}
