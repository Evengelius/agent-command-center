//go:build windows

package input

import (
	"strings"
	"testing"
	"unicode/utf16"
	"unsafe"
)

// TestPickedPaths is plan 31's design D2 without the dialog itself, which
// the Windows Terminal harness opens: the buffer GetOpenFileNameW fills,
// one full path or a folder and its names, read back as paths, Unicode and
// spaces kept; the filter's pairs; OPENFILENAMEW's size on 64-bit Windows.
func TestPickedPaths(t *testing.T) {
	buf := func(parts ...string) []uint16 {
		var b []uint16
		for _, p := range parts {
			b = append(append(b, utf16.Encode([]rune(p))...), 0)
		}
		return append(b, 0, 0, 0)
	}
	if got := pickedPaths(buf(`C:\my pictures\é 日本.png`)); len(got) != 1 || got[0] != `C:\my pictures\é 日本.png` {
		t.Errorf("one file: %q", got)
	}
	if got := pickedPaths(buf(`C:\my pictures`, "one.png", "two 2.jpg")); len(got) != 2 || got[0] != `C:\my pictures\one.png` || got[1] != `C:\my pictures\two 2.jpg` {
		t.Errorf("two files: %q", got)
	}
	if got := pickedPaths(make([]uint16, 8)); got != nil {
		t.Errorf("none: %q", got)
	}
	f := pickFilter()
	if !strings.HasPrefix(f, "Images\x00*.png;*.apng;*.jpg;") || !strings.Contains(f, ";*.heic;") || !strings.HasSuffix(f, "\x00All files\x00*.*\x00") {
		t.Errorf("filter %q", f)
	}
	if n := unsafe.Sizeof(openFileName{}); unsafe.Sizeof(uintptr(0)) == 8 && n != 152 {
		t.Errorf("OPENFILENAMEW is %d bytes, want 152", n)
	}
}
