package input

import (
	"path/filepath"
	"strings"
	"testing"
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
