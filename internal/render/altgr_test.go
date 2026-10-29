package render

import (
	"strings"
	"testing"
)

// TestKeyTableNoCtrlAlt is criterion 67.3's shortcuts (plan 67): no key the
// overlay lists is a Ctrl+Alt chord, the modifiers AltGr's characters come
// with, so that typing one never fires a binding.
func TestKeyTableNoCtrlAlt(t *testing.T) {
	n := 0
	for _, g := range KeyTable {
		for _, k := range g.keys {
			n++
			if s := strings.ToLower(k[0]); strings.Contains(s, "ctrl+alt") || strings.Contains(s, "alt+ctrl") {
				t.Errorf("%s: %q is a Ctrl+Alt chord", g.title, k[0])
			}
		}
	}
	if n == 0 {
		t.Fatal("no key read from the overlay's table")
	}
}
