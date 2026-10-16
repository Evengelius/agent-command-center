package input

import "testing"

func TestEditorKeepsPasteByteForByte(t *testing.T) {
	var e Editor
	paste := "line 1\r\n\tline 2 é 中\n\x00end\n"
	e.Insert("> ")
	e.Insert(paste)
	if got, want := e.Text(), "> "+paste; got != want {
		t.Fatalf("text: got %q, want %q", got, want)
	}
	if e.Cursor() != len(e.Text()) {
		t.Fatalf("cursor %d, want %d", e.Cursor(), len(e.Text()))
	}
}

func TestEditorRuneEditing(t *testing.T) {
	var e Editor
	e.SetText("aé中")
	e.Backspace()
	if e.Text() != "aé" {
		t.Fatalf("backspace: %q", e.Text())
	}
	e.Left()
	e.Delete()
	if e.Text() != "a" || e.Cursor() != 1 {
		t.Fatalf("left+delete: %q at %d", e.Text(), e.Cursor())
	}
	e.Newline()
	e.Insert("b")
	e.Home()
	if e.Cursor() != 2 {
		t.Fatalf("home: %d", e.Cursor())
	}
	e.End()
	if e.Cursor() != 3 || e.Lines() != 2 {
		t.Fatalf("end: %d, lines %d", e.Cursor(), e.Lines())
	}
}

func TestCleanPath(t *testing.T) {
	for in, want := range map[string]string{
		"\"C:\\dev\\app\"\r\n": `C:\dev\app`,
		"  ~/proj  ":           "~/proj",
		"C:\\a\nC:\\b":         `C:\a`,
		"\"":                   "\"",
	} {
		if got := CleanPath(in); got != want {
			t.Errorf("CleanPath(%q) = %q, want %q", in, got, want)
		}
	}
}
