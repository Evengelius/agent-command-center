package input

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Evengelius/agent-command-center/internal/input/imagetest"
)

// FuzzInspectData is plan 48's criterion 48.3 (design D1; audit findings 2
// and 6): no bytes, under any name, make the image readers panic. It is
// seeded with every file imagetest writes and the checked-in WebP; go test
// runs the seeds, and the evaluation fuzzes it for a bounded time
// (doc/evaluation.md, Fuzzing).
func FuzzInspectData(f *testing.F) {
	files, err := imagetest.WriteAll(f.TempDir())
	if err != nil {
		f.Fatal(err)
	}
	for _, p := range files {
		b, err := os.ReadFile(p)
		if err != nil {
			f.Fatal(err)
		}
		f.Add(filepath.Base(p), b)
	}
	webp, err := os.ReadFile(filepath.Join("..", "..", "testdata", "images", "acc-image-upload-test.webp"))
	if err != nil {
		f.Fatal(err)
	}
	f.Add("acc-image-upload-test.webp", webp)
	f.Fuzz(func(t *testing.T, name string, b []byte) {
		_, _, _, _ = InspectData(name, b, claudeLimits)
	})
}
