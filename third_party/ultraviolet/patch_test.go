// This file is not upstream: it is agent-command-center's test of the three
// changes in PATCHES.md. It replaces the removed upstream subtest
// TestWin32Functions/ensureKeyCase with the same four cases (named alike),
// fed as win32-input-mode sequences through the scanner, plus the cases
// patches 1 and 2 fix, and renders frames through the renderer for patch 3.
// It uses the standard library only.

package uv

import (
	"bytes"
	"strings"
	"testing"
)

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

// patchFrames renders each frame, its rows drawn from the left, the way
// Bubble Tea does: the buffer cleared first, which touches every line. It
// returns what each render wrote, and what a render at the next width
// wrote, when grow is set.
func patchFrames(t *testing.T, grapheme, grow bool, frames ...[]string) []string {
	const w, h = 20, 3
	var buf bytes.Buffer
	r := NewTerminalRenderer(&buf, []string{"TERM=xterm-256color"})
	r.SetFullscreen(true)
	r.SetGraphemeWidth(grapheme)
	r.Resize(w, h)
	scr := NewScreenBuffer(w, h)
	var out []string
	render := func(rows []string) {
		scr.Clear()
		for y, row := range rows {
			NewStyledString(row).Draw(scr, Rect(0, y, scr.Width(), 1))
		}
		r.Render(scr.RenderBuffer)
		if err := r.Flush(); err != nil {
			t.Fatalf("flush: %v", err)
		}
		out = append(out, buf.String())
		buf.Reset()
	}
	for _, rows := range frames {
		render(rows)
	}
	if grow {
		r.Resize(w+2, h)
		scr = NewScreenBuffer(w+2, h)
		render(frames[len(frames)-1])
	}
	return out
}

// TestPatchedDriftRows is patch 3: a row holding a wide cell, which the
// renderer repaints rather than diffs, is not written at all by a frame that
// leaves it as it was, and only up to its last changed cell by one that
// changes it, with no erase; the cells after the change stay as the screen
// shows them, an image drawn over them included. A resize still repaints it
// whole. With and without grapheme widths, the renderer's two drift paths.
func TestPatchedDriftRows(t *testing.T) {
	const (
		one   = "✅ one          TAIL" // 20 columns, the check mark two
		two   = "✅ two          TAIL"
		wide  = "ab✅ x          TAIL"
		wide2 = "ac世 x          TAIL"
		plain = "plain"
		erase = "\x1b[K"
	)
	for _, grapheme := range []bool{false, true} {
		t.Run(map[bool]string{false: "wcwidth", true: "grapheme"}[grapheme], func(t *testing.T) {
			out := patchFrames(t, grapheme, false, []string{one, plain}, []string{one, plain})
			if !strings.Contains(out[0], "✅") || !strings.Contains(out[0], "TAIL") {
				t.Fatalf("the first frame did not paint the row: %q", out[0])
			}
			if strings.Contains(out[1], "✅") || strings.Contains(out[1], "TAIL") || strings.Contains(out[1], erase) {
				t.Errorf("a frame that changed nothing wrote the row again: %q", out[1])
			}

			out = patchFrames(t, grapheme, false, []string{one, plain}, []string{two, plain}, []string{two, plain})
			if !strings.Contains(out[1], "✅ two") || strings.Contains(out[1], "TAIL") || strings.Contains(out[1], erase) {
				t.Errorf("a change at the left: want the row from column 0 to the change, no erase, the tail left; got %q", out[1])
			}
			if strings.Contains(out[2], "two") || strings.Contains(out[2], "TAIL") {
				t.Errorf("the model did not follow the partial repaint, the next frame wrote the row: %q", out[2])
			}

			out = patchFrames(t, grapheme, false, []string{wide, plain}, []string{wide2, plain})
			if !strings.Contains(out[1], "ac世") || strings.Contains(out[1], "x") || strings.Contains(out[1], erase) {
				t.Errorf("a change ending on a wide cell: want it written whole, nothing after it; got %q", out[1])
			}

			out = patchFrames(t, grapheme, false, []string{one, plain}, []string{"✅ one          TAIX", plain})
			if !strings.Contains(out[1], "✅ one") || !strings.Contains(out[1], "TAIX") {
				t.Errorf("a change in the last column: want the whole row written; got %q", out[1])
			}

			out = patchFrames(t, grapheme, true, []string{one, plain})
			if !strings.Contains(out[1], "✅") || !strings.Contains(out[1], "TAIL") {
				t.Errorf("a resize did not repaint the row whole: %q", out[1])
			}
		})
	}
}
