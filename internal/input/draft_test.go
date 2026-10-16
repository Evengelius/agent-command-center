package input

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestDrafts is the store of proposal step 4, item 4: a draft is kept byte
// for byte, one per project, replaced whole, deleted by an empty text, and a
// missing draft is no error.
func TestDrafts(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "drafts") // created by the first save
	d := Drafts{Dir: dir}
	a, b := filepath.Join(t.TempDir(), "secret-project-a"), filepath.Join(t.TempDir(), "project-b")
	text := "\tline 1  \r\nline 2\n日本語 👍🏽\x1b[0m\x00end"

	if got, err := d.Load(a); got != "" || err != nil {
		t.Fatalf("no draft: got %q, %v", got, err)
	}
	if err := d.Delete(a); err != nil {
		t.Fatalf("deleting no draft: %v", err)
	}
	if err := d.Save(a, text); err != nil {
		t.Fatal(err)
	}
	if err := d.Save(b, "other"); err != nil {
		t.Fatal(err)
	}
	if got, err := d.Load(a); got != text || err != nil {
		t.Fatalf("byte for byte: got %q, %v; want %q", got, err, text)
	}
	if err := d.Save(a, "shorter"); err != nil {
		t.Fatal(err)
	}
	if got, _ := d.Load(a); got != "shorter" {
		t.Fatalf("replaced whole: got %q", got)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("files %v; want one per project, no temporary file left", entries)
	}
	for _, e := range entries {
		if strings.Contains(e.Name(), "secret") || !strings.HasSuffix(e.Name(), ".txt") || len(e.Name()) != 64+4 {
			t.Errorf("file name %q: want the path's SHA-256 only", e.Name())
		}
	}

	if err := d.Save(a, ""); err != nil {
		t.Fatal(err)
	}
	if p, _ := d.Path(a); fileExists(p) {
		t.Error("an empty draft left its file")
	}
	if got, _ := d.Load(b); got != "other" {
		t.Errorf("the other project's draft: got %q", got)
	}
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}
