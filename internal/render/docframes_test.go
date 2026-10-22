package render

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestDocFrames writes the frames ui-design.md's Layouts section shows, P3b
// at its Write request at 80x24 (plan 28), 120x40 and 200x60, into the
// folder ACC_DOC_FRAMES names, so the document is redrawn from the code,
// not by hand (plan 27); skipped without the variable.
func TestDocFrames(t *testing.T) {
	dir := os.Getenv("ACC_DOC_FRAMES")
	if dir == "" {
		t.Skip("ACC_DOC_FRAMES not set")
	}
	s := p3bAtWrite(t)
	for _, size := range [][2]int{{80, 24}, {120, 40}, {200, 60}} {
		ui := UI{Width: size[0], Height: size[1], Now: appClock.Add(13 * time.Second), Focus: "B"}
		name := filepath.Join(dir, fmt.Sprintf("p3b-write-%dx%d.txt", size[0], size[1]))
		if err := os.WriteFile(name, []byte(texts(Draw(s, ui).lines)), 0o644); err != nil {
			t.Fatal(err)
		}
		t.Logf("wrote %s", name)
	}
}
