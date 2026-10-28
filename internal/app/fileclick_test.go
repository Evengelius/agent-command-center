package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/synctest"

	tea "charm.land/bubbletea/v2"

	"github.com/Evengelius/agent-command-center/internal/render"
)

// Plan 61 (brief amendment 63): a click on a line of the file viewer puts
// the cursor on it; in the quick open, a click on a file selects it and a
// click on the selected one opens it.

// shown is the frame drawn now and what its window's rows showed (plan 61,
// design D5).
func shown(h *harness) render.ListShown {
	h.m.View()
	return h.m.screen.WindowList
}

// TestFileClickLine is criterion 61.1 (design D1): at 120x40, 80x24 and full
// screen, a click on the middle, the last and the first line in view puts
// the cursor there, on the code, the blank after it, the number and the
// row's edge; the view and the sideways shift stay; with the blame on, a
// click on its column, the footer naming that line's commit.
func TestFileClickLine(t *testing.T) {
	for _, c := range []struct {
		name string
		w, h int
		full bool
	}{{"120x40", 120, 40, false}, {"80x24", 80, 24, false}, {"full screen", 120, 40, true}} {
		t.Run(c.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				r, _, _ := fileRig(t)
				h := r.h
				h.send(tea.WindowSizeMsg{Width: c.w, Height: c.h})
				openPath(h, "main.go")
				if c.full {
					h.send(typed('f'))
				}
				h.send(key(tea.KeyDown, 0))
				h.send(key(tea.KeyRight, 0)) // a shift, which a click keeps
				l := shown(h)
				left, nw := h.m.ui.File.Left, len("100")
				if l.First != 0 || l.N < 10 {
					t.Fatalf("the lines shown: %+v", l)
				}
				for _, k := range []struct {
					row, x int
					what   string
				}{
					{l.N / 2, l.X + 1 + nw + 2, "the code"},
					{l.N - 1, l.X + l.W - 2, "the blank after the code"},
					{0, l.X + 1, "the number"},
					{l.N / 3, l.X, "the row's edge"},
				} {
					mouseClick(t, h, k.x, l.Y+k.row)
					f := h.m.ui.File
					if f.Cursor != k.row || f.Left != left || !h.m.ui.Sel.Empty() {
						t.Errorf("a click on %s of row %d: cursor %d, shift %d (was %d), selection %v", k.what, k.row, f.Cursor, f.Left, left, h.m.ui.Sel)
					}
					if after := shown(h); after.First != l.First || !strings.Contains(h.frame(), fmt.Sprintf("line %d/100", k.row+1)) {
						t.Errorf("a click on %s of row %d: the view from line %d, was %d; footer %q", k.what, k.row, after.First+1, l.First+1, footerOf(h))
					}
				}
			})
		})
	}

	// A key moving the view after the frame and before the click: the click
	// is on the line drawn, and the view goes back to the rows drawn (design
	// D5).
	t.Run("a key after the frame", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			r, _, _ := fileRig(t)
			h := r.h
			h.send(tea.WindowSizeMsg{Width: 120, Height: 40})
			openPath(h, "main.go")
			l := shown(h)
			h.send(key(tea.KeyPgDown, 0)) // not drawn
			h.deliver(click(l.X+8, l.Y+5))
			h.deliver(tea.MouseReleaseMsg{X: l.X + 8, Y: l.Y + 5, Button: tea.MouseLeft})
			h.settle()
			if f := h.m.ui.File; f.Cursor != l.First+5 || shown(h).First != l.First {
				t.Errorf("a click on line %d as drawn after PgDn: cursor on line %d, the view from line %d", l.First+6, f.Cursor+1, shown(h).First+1)
			}
		})
	})

	t.Run("the blame", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			r, _, _, _ := blameRig(t)
			h := r.h
			h.send(tea.WindowSizeMsg{Width: 120, Height: 40})
			openPath(h, "main.go")
			h.send(typed('b'))
			l := shown(h)
			for _, c := range []struct {
				line int
				want string
			}{
				{22, "9a1b2c3 · Bo Li · 02/03/2023 14:30 · Rework main.go (read 1)"},
				{5, "f52c343 · Ana Hubaut · 15/11/2022 10:00 · Add main.go"},
			} {
				mouseClick(t, h, l.X+3, l.Y+c.line-1) // on the blame's column
				if f := h.m.ui.File; f.Cursor != c.line-1 || !strings.Contains(h.frame(), c.want) {
					t.Errorf("a click on line %d's blame: cursor %d, footer %q", c.line, f.Cursor+1, footerOf(h))
				}
			}
		})
	})
}

// footerOf is the frame's row holding the viewer's line count.
func footerOf(h *harness) string {
	for _, row := range strings.Split(h.frame(), "\n") {
		if strings.Contains(row, "line ") && strings.Contains(row, "/") {
			return strings.TrimSpace(row)
		}
	}
	return ""
}

// TestFileClickNotALine is criterion 61.2 (design D1): a click on the path
// row, the footer, the frame, a row past a short file's end, and in the
// window while the file is read, empty or binary, leaves the cursor.
func TestFileClickNotALine(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r, _, root := fileRig(t)
		h := r.h
		h.send(tea.WindowSizeMsg{Width: 120, Height: 40})
		openPath(h, "main.go")
		for range 4 {
			h.send(key(tea.KeyDown, 0))
		}
		l, w := shown(h), h.m.screen.Window
		for _, c := range []struct {
			what string
			x, y int
		}{
			{"the path row", l.X + 5, l.Y - 1},
			{"the footer", l.X + 5, w.Y + w.H - 2},
			{"the top edge", l.X + 5, w.Y},
			{"the bottom edge", l.X + 5, w.Y + w.H - 1},
			{"the left edge", w.X, l.Y + 1},
			{"the right edge", w.X + w.W - 1, l.Y + 1},
		} {
			mouseClick(t, h, c.x, c.y)
			if f := h.m.ui.File; !f.Open || f.Cursor != 4 {
				t.Errorf("a click on %s: viewer %v, cursor on line %d, want 5", c.what, f.Open, f.Cursor+1)
			}
		}

		if err := os.WriteFile(filepath.Join(root, "short.txt"), []byte("a\nb\nc\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		openPath(h, "short.txt")
		h.send(key(tea.KeyDown, 0))
		h.send(key(tea.KeyDown, 0))
		if l = shown(h); l.N != 3 {
			t.Fatalf("short.txt: %+v", l)
		}
		mouseClick(t, h, l.X+5, l.Y+5)
		if f := h.m.ui.File; f.Cursor != 2 {
			t.Errorf("a click under short.txt's last line: cursor on line %d, want 3", f.Cursor+1)
		}

		if err := os.WriteFile(filepath.Join(root, "empty.txt"), nil, 0o644); err != nil {
			t.Fatal(err)
		}
		for _, p := range []string{"empty.txt", "data.bin"} {
			openPath(h, p)
			if l = shown(h); l.N != 0 {
				t.Errorf("%s: rows shown %+v", p, l)
			}
			mouseClick(t, h, l.X+5, l.Y+1)
			if f := h.m.ui.File; !f.Open || f.Cursor != 0 {
				t.Errorf("a click in %s: viewer %v, cursor %d", p, f.Open, f.Cursor)
			}
		}
		// While the file is read: its read not run yet.
		h.m.openFile(render.FileTarget{Path: "main.go", Opened: "quick open"})
		if l = shown(h); l.N != 0 || h.m.ui.File.State == "" {
			t.Fatalf("reading: %+v, state %q", l, h.m.ui.File.State)
		}
		mouseClick(t, h, l.X+5, l.Y+3)
		if f := h.m.ui.File; !f.Open || f.Cursor != 0 || len(f.Lines) != 0 {
			t.Errorf("a click while reading: viewer %v, cursor %d, %d lines", f.Open, f.Cursor, len(f.Lines))
		}

		// A click on rows drawn for a file another has replaced since, its
		// read not back: no line of the new one (design D5).
		openPath(h, "main.go")
		l = shown(h)
		h.m.openFile(render.FileTarget{Path: "short.txt", Opened: "quick open"}) // not drawn, not read
		h.deliver(click(l.X+5, l.Y+20))
		h.deliver(tea.MouseReleaseMsg{X: l.X + 5, Y: l.Y + 20, Button: tea.MouseLeft})
		if f := h.m.ui.File; f.Cursor != 0 || len(f.Lines) != 0 || !strings.Contains(h.frame(), "short.txt") {
			t.Errorf("a click on main.go's rows after short.txt was opened: cursor %d, %d lines", f.Cursor, len(f.Lines))
		}
	})
}

// TestFileClickSelection is criterion 61.3 (design D2): the click still
// starts a selection, the cursor on the line pressed: a drag copies the code
// alone, a double click a word, a triple click the line, Shift+click
// extends; a right or a middle click moves nothing.
func TestFileClickSelection(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r, clip, _ := fileRig(t)
		h := r.h
		h.send(tea.WindowSizeMsg{Width: 120, Height: 40})
		openPath(h, "main.go")

		x3, y3 := at(t, h, "// line 3", 0)
		x5, y5 := at(t, h, "// line 5", 0)
		dragCopy(t, h, x3, y3, x5+len("// line 5"), y5)
		if got := clip.last(); got != "// line 3\n// line 4\n// line 5" || h.m.ui.File.Cursor != 2 {
			t.Errorf("a drag from line 3: copied %q, cursor on line %d", got, h.m.ui.File.Cursor+1)
		}
		h.send(key(tea.KeyEscape, 0)) // the selection cleared

		// The clock stands still: clicks on one cell are a double or a triple
		// click.
		x7, y7 := at(t, h, "// line 7", 0)
		mouseClick(t, h, x7+4, y7)
		mouseClick(t, h, x7+4, y7)
		if got := clip.last(); got != "line" || h.m.ui.File.Cursor != 6 {
			t.Errorf("a double click on line 7: copied %q, cursor on line %d", got, h.m.ui.File.Cursor+1)
		}
		h.send(key(tea.KeyEscape, 0))
		x9, y9 := at(t, h, "// line 9", 0)
		for range 3 {
			mouseClick(t, h, x9+1, y9)
		}
		if got := clip.last(); got != "// line 9" || h.m.ui.File.Cursor != 8 {
			t.Errorf("a triple click on line 9: copied %q, cursor on line %d", got, h.m.ui.File.Cursor+1)
		}
		h.send(key(tea.KeyEscape, 0))

		x2, y2 := at(t, h, "// line 2", 0)
		x6, y6 := at(t, h, "// line 6", 0)
		mouseClick(t, h, x2, y2)
		mouseAt(t, h, tea.MouseClickMsg{X: x6 + len("// line 6"), Y: y6, Button: tea.MouseLeft, Mod: tea.ModShift})
		mouseAt(t, h, tea.MouseReleaseMsg{X: x6 + len("// line 6"), Y: y6, Button: tea.MouseLeft, Mod: tea.ModShift})
		if got := clip.last(); !strings.HasPrefix(got, "// line 2\n") || !strings.HasSuffix(got, "// line 6") || h.m.ui.File.Cursor != 5 {
			t.Errorf("Shift+click on line 6 after a click on line 2: copied %q, cursor on line %d", got, h.m.ui.File.Cursor+1)
		}
		h.send(key(tea.KeyEscape, 0))

		x8, y8 := at(t, h, "// line 8", 0)
		for _, b := range []tea.MouseButton{tea.MouseRight, tea.MouseMiddle} {
			mouseAt(t, h, tea.MouseClickMsg{X: x8, Y: y8, Button: b})
			mouseAt(t, h, tea.MouseReleaseMsg{X: x8, Y: y8, Button: b})
			if f := h.m.ui.File; !f.Open || f.Cursor != 5 {
				t.Errorf("button %v on line 8: viewer %v, cursor on line %d, want 6", b, f.Open, f.Cursor+1)
			}
		}
	})
}

// TestFileClickField is criterion 61.4 (design D3): with the search or the
// go-to field open, a click on a line moves the cursor and leaves the field
// open with its text; Enter then acts as before.
func TestFileClickField(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r, _, _ := fileRig(t)
		h := r.h
		h.send(tea.WindowSizeMsg{Width: 120, Height: 40})
		openPath(h, "main.go")

		h.send(typed('/'))
		for _, k := range typedText("line 9") {
			h.send(k)
		}
		x4, y4 := at(t, h, "// line 4", 0)
		mouseClick(t, h, x4, y4)
		if f := h.m.ui.File; f.Cursor != 3 || f.Field != '/' || f.Typed != "line 9" {
			t.Errorf("a click while searching: cursor on line %d, field %q %q", f.Cursor+1, f.Field, f.Typed)
		}
		h.send(key(tea.KeyEnter, 0))
		if f := h.m.ui.File; f.Field != 0 || len(f.Matches) == 0 || f.Query != "line 9" {
			t.Errorf("Enter after the click: field %q, %d matches, query %q", f.Field, len(f.Matches), f.Query)
		}

		h.send(typed('g'))
		h.send(typed('1'))
		x6, y6 := at(t, h, "// line 6", 0)
		mouseClick(t, h, x6, y6)
		if f := h.m.ui.File; f.Cursor != 5 || f.Field != 'g' || f.Typed != "1" {
			t.Errorf("a click while going to a line: cursor on line %d, field %q %q", f.Cursor+1, f.Field, f.Typed)
		}
		h.send(key(tea.KeyEnter, 0))
		if f := h.m.ui.File; f.Field != 0 || f.Cursor != 0 {
			t.Errorf("Enter after the click: field %q, cursor on line %d", f.Field, f.Cursor+1)
		}
	})
}

// TestFinderClick is criterion 61.5 (design D4): in the quick open, a click
// on a file selects it, a click on the selected one opens it, so does a
// double click; on a scrolled list, the file under the pointer; the field
// row, the footer, a row past the list and its states do nothing.
func TestFinderClick(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r, _, root := fileRig(t)
		h := r.h
		h.send(tea.WindowSizeMsg{Width: 120, Height: 40})
		if err := os.MkdirAll(filepath.Join(root, "gen"), 0o755); err != nil {
			t.Fatal(err)
		}
		for i := range 60 {
			if err := os.WriteFile(filepath.Join(root, "gen", fmt.Sprintf("f%02d.txt", i)), []byte("x\n"), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		finder := func() {
			for _, c := range h.m.openFinder() {
				h.exec(c)
			}
			h.settle()
		}
		finder()
		v := h.m.ui.Finder
		if v.State != "" || len(v.Files) < 60 {
			t.Fatalf("the quick open: %q, %d files", v.State, len(v.Files))
		}
		l := shown(h)
		mouseClick(t, h, l.X+5, l.Y+2)
		if v := h.m.ui.Finder; !v.Open || v.Sel != l.First+2 || !strings.Contains(h.frame(), fmt.Sprintf("3 of %d", len(v.Files))) {
			t.Errorf("a click on the third file: open %v, selected %d", v.Open, v.Sel)
		}
		want := v.Files[2]
		mouseClick(t, h, l.X+5, l.Y+2)
		if f := h.m.ui.File; h.m.ui.Finder.Open || !f.Open || f.Rel != want || f.Opened != "quick open" {
			t.Errorf("a click on the selected file: quick open %v, viewer %v %q (%s), want %q", h.m.ui.Finder.Open, f.Open, f.Rel, f.Opened, want)
		}

		finder()
		want = h.m.ui.Finder.Files[4]
		l = shown(h)
		mouseClick(t, h, l.X+8, l.Y+4)
		mouseClick(t, h, l.X+8, l.Y+4)
		if f := h.m.ui.File; h.m.ui.Finder.Open || f.Rel != want {
			t.Errorf("a double click on the fifth file: quick open %v, viewer %q, want %q", h.m.ui.Finder.Open, f.Rel, want)
		}

		// Scrolled: the file under the pointer, the rows where they were.
		finder()
		h.send(key(tea.KeyEnd, 0))
		l = shown(h)
		if l.First == 0 {
			t.Fatalf("End did not scroll the list: %+v", l)
		}
		mouseClick(t, h, l.X+5, l.Y+1)
		if v := h.m.ui.Finder; v.Sel != l.First+1 || shown(h).First != l.First {
			t.Errorf("a click on a scrolled list: selected %d, want %d; first shown %d, was %d", v.Sel, l.First+1, shown(h).First, l.First)
		}

		// A key moving the list after the frame and before the click: the
		// file drawn under the pointer, the rows as drawn (design D5).
		h.send(key(tea.KeyHome, 0))
		l = shown(h)
		h.send(key(tea.KeyPgDown, 0)) // not drawn
		h.deliver(click(l.X+5, l.Y+3))
		h.deliver(tea.MouseReleaseMsg{X: l.X + 5, Y: l.Y + 3, Button: tea.MouseLeft})
		h.settle()
		if v := h.m.ui.Finder; v.Sel != l.First+3 || shown(h).First != l.First {
			t.Errorf("a click on file %d as drawn after PgDn: selected %d, the list from %d", l.First+3, v.Sel, shown(h).First)
		}

		w := h.m.screen.Window
		sel := h.m.ui.Finder.Sel
		for _, c := range []struct {
			what string
			x, y int
		}{
			{"the field row", l.X + 5, l.Y - 1},
			{"the footer", l.X + 5, w.Y + w.H - 2},
			{"the left edge", w.X, l.Y + 1},
		} {
			mouseClick(t, h, c.x, c.y)
			if v := h.m.ui.Finder; !v.Open || v.Sel != sel || h.m.ui.File.Rel != want {
				t.Errorf("a click on %s: open %v, selected %d, want %d; viewer %q", c.what, v.Open, v.Sel, sel, h.m.ui.File.Rel)
			}
		}
		// A click on a row drawn before a query narrowed the list, not drawn
		// since: no file past the list (design D5).
		l = shown(h)
		for _, k := range typedText("f00") {
			h.send(k)
		}
		h.deliver(click(l.X+5, l.Y+10))
		h.deliver(tea.MouseReleaseMsg{X: l.X + 5, Y: l.Y + 10, Button: tea.MouseLeft})
		if v := h.m.ui.Finder; !v.Open || v.Sel != 0 || len(v.Files) != 1 {
			t.Errorf("a click on a row of the list before f00: open %v, selected %d of %d", v.Open, v.Sel, len(v.Files))
		}
		if l = shown(h); l.N != 1 {
			t.Fatalf("f00: %+v", l)
		}
		mouseClick(t, h, l.X+5, l.Y+3)
		if v := h.m.ui.Finder; !v.Open || v.Sel != 0 {
			t.Errorf("a click under the list: open %v, selected %d", v.Open, v.Sel)
		}
		for _, k := range typedText("zz") {
			h.send(k)
		}
		mouseClick(t, h, l.X+5, l.Y)
		if v := h.m.ui.Finder; !v.Open || len(v.Files) != 0 || !strings.Contains(h.frame(), "no file matches") {
			t.Errorf("a click on \"no file matches\": open %v, %d files", v.Open, len(v.Files))
		}
		h.send(key(tea.KeyEscape, 0))

		// Still listing: the listing not run yet.
		h.m.openFinder()
		if l = shown(h); l.N != 0 {
			t.Fatalf("listing: %+v", l)
		}
		mouseClick(t, h, l.X+5, l.Y+1)
		if v := h.m.ui.Finder; !v.Open || v.State != "listing…" {
			t.Errorf("a click while listing: open %v, state %q", v.Open, v.State)
		}
	})
}
