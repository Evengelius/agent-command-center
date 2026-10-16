// This file is not upstream: it is agent-command-center's test of the two
// changes in PATCHES.md, and replaces the removed upstream subtest
// TestWin32Functions/ensureKeyCase with the same four cases (named alike),
// fed as win32-input-mode sequences through the scanner, plus the cases the
// patches fix. It uses the standard library only.

package uv

import "testing"

func TestPatchedWin32Input(t *testing.T) {
	tests := []struct {
		name string
		in   string // key-down records: ESC [ vk ; sc ; uc ; kd ; cks ; rc _
		text string
		code rune
		mod  KeyMod
	}{
		// The four cases of the removed subtest. "uppercase without shift"
		// is "a" upstream; the console's character is kept (patch 1).
		{"uppercase with shift", "\x1b[65;30;65;1;16;1_", "A", 'A', ModShift},
		{"lowercase without shift", "\x1b[65;30;97;1;0;1_", "a", 'a', 0},
		{"uppercase without shift", "\x1b[65;30;65;1;0;1_", "A", 'A', 0},
		{"non-letter", "\x1b[49;2;49;1;0;1_", "1", '1', 0},
		// Caps Lock and Shift cancel out: "A" upstream (patch 1).
		{"lowercase with caps lock and shift", "\x1b[65;30;97;1;144;1_", "a", 'a', ModShift | ModCapsLock},
		// An emoji's two halves, as Windows Terminal 1.23 sends them with
		// vk 255: dropped upstream (patch 2). The vk 0 form worked already.
		{"surrogate pair, vk 255", "\x1b[255;0;55357;1;32;1_\x1b[255;0;56898;1;32;1_", "\U0001F642", '\U0001F642', 0},
		{"surrogate pair, vk 0", "\x1b[0;0;55357;1;0;1_\x1b[0;0;56898;1;0;1_", "\U0001F642", '\U0001F642', 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// The byte count is not checked: upstream adds a surrogate pair's
			// decoded UTF-8 to it (40 for these 36 bytes), patched or not.
			_, events := newEventScanner().scanEvents([]byte(tt.in), true)
			if len(events) != 1 {
				t.Fatalf("%q: events %v; want one", tt.in, events)
			}
			k, ok := events[0].(KeyPressEvent)
			if !ok {
				t.Fatalf("%q: %T, want KeyPressEvent", tt.in, events[0])
			}
			if k.Text != tt.text || k.Code != tt.code || k.ShiftedCode != 0 || k.Mod != tt.mod {
				t.Errorf("%q: Text %q Code %q ShiftedCode %q Mod %v; want %q %q 0 %v",
					tt.in, k.Text, k.Code, k.ShiftedCode, k.Mod, tt.text, tt.code, tt.mod)
			}
		})
	}
}
