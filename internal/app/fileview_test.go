package app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"testing/synctest"

	tea "charm.land/bubbletea/v2"

	"github.com/Evengelius/agent-command-center/internal/event"
	"github.com/Evengelius/agent-command-center/internal/input"
	"github.com/Evengelius/agent-command-center/internal/render"
)

// fileRig is a scripted session whose project holds a hundred-line Go
// file, a TypeScript, a Java and a Markdown file, a credential-named file
// and a binary one.
func fileRig(t *testing.T) (inputRig, *fakeClipboard, string) {
	r, clip := copyRig(t, nil)
	root := r.h.m.State().Context.Project
	lines := make([]string, 100)
	for i := range lines {
		lines[i] = fmt.Sprintf("// line %d", i+1)
	}
	for name, body := range map[string]string{
		"main.go":        strings.Join(lines, "\n") + "\n",
		"web/app.ts":     "export const a: number = 1;\n",
		"src/Main.java":  "class Main {}\n",
		".env":           "NAME=value\n",
		"data.bin":       "abc\x00def",
		"docs/readme.md": "# Readme\n",
	} {
		p := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return r, clip, root
}

// openPath opens a file in the viewer as the quick open would, and lets
// the read and the colouring finish.
func openPath(h *harness, path string) {
	for _, c := range h.m.openFile(render.FileTarget{Path: path, Opened: "quick open"}) {
		h.exec(c)
	}
	h.settle()
}

// toolInput is a tool call's input as JSON: built with json.Marshal, the
// project's Windows paths escaped.
func toolInput(t *testing.T, v any) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// outside is the frame with the window's rectangle blanked.
func outside(h *harness) string {
	x, y, w, hh := render.FileRect(h.m.ui.Width, h.m.ui.Height, h.m.ui.File.Full)
	rows := strings.Split(plainText(h.m.View().Content), "\n")
	for i := y; i < y+hh && i < len(rows); i++ {
		r := []rune(rows[i])
		for j := x; j < x+w && j < len(r); j++ {
			r[j] = ' '
		}
		rows[i] = string(r)
	}
	return strings.Join(rows, "\n")
}

// TestFileOverDashboard is plan 17, criterion 3.1, in the application: the
// window floats over the dashboard, which is drawn outside it as without it
// and keeps updating behind it.
func TestFileOverDashboard(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r, _, _ := fileRig(t)
		h := r.h
		h.m.ui.File.Open = true // the rectangle, to blank the same cells
		without := outside(h)
		h.m.ui.File.Open = false
		openPath(h, "main.go")
		if !h.m.ui.File.Open || h.m.ui.File.State != "" {
			t.Fatalf("the viewer: %+v", h.m.ui.File.State)
		}
		if got := outside(h); got != without {
			t.Errorf("the dashboard outside the window changed:\n%s\n---\n%s", got, without)
		}
		r.emit(event.TurnStarted, nil)
		h.settle()
		if got := outside(h); got == without {
			t.Errorf("a turn started and nothing changed outside the window:\n%s", got)
		}
		if !h.m.ui.File.Open {
			t.Error("an event closed the viewer")
		}
	})
}

// TestFileFromEntry is plan 17, criterion 3.2: F3 on a tool entry opens the
// file it read or changed, at the place, the changed lines marked; on an
// entry naming no file it opens the Explorer (amendment 12).
func TestFileFromEntry(t *testing.T) {
	for _, c := range []struct {
		name   string
		tool   string
		input  func(root string) any
		cursor int
		mark   [2]int
	}{
		{"Read at an offset", "Read", func(root string) any {
			return map[string]any{"file_path": filepath.Join(root, "main.go"), "offset": 40, "limit": 5}
		}, 39, [2]int{}},
		{"Edit", "Edit", func(root string) any {
			return map[string]any{"file_path": filepath.Join(root, "main.go"), "old_string": "x", "new_string": "// line 50\n// line 51"}
		}, 49, [2]int{49, 51}},
		{"fileChange", "fileChange", func(root string) any {
			return map[string]any{"changes": []map[string]string{{"path": filepath.Join(root, "main.go"),
				"diff": "@@ -10,2 +10,3 @@\n // line 10\n+// line 11\n+// line 12\n // line 13\n"}}}
		}, 10, [2]int{10, 12}},
	} {
		t.Run(c.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				r, _, root := fileRig(t)
				h := r.h
				r.emit(event.ToolStarted, event.Tool{ToolID: "t1", Name: c.tool, Input: toolInput(t, c.input(root))})
				h.settle()
				h.send(key(tea.KeyUp, 0))
				if h.m.ui.Entry == 0 {
					t.Fatal("↑ chose no entry")
				}
				h.send(key(tea.KeyF3, 0))
				f := h.m.ui.File
				if !f.Open || f.Rel != "main.go" || f.State != "" || f.Cursor != c.cursor || f.Mark != c.mark || h.m.ui.Finder.Open {
					t.Fatalf("the viewer: open %v, %q, state %q, cursor %d, mark %v", f.Open, f.Rel, f.State, f.Cursor, f.Mark)
				}
				if f.Top > f.Cursor || f.Cursor >= f.Top+render.FileRows(h.m.ui) {
					t.Errorf("the cursor %d is out of the view from %d", f.Cursor, f.Top)
				}
				if c.mark != [2]int{} && !strings.Contains(plainText(h.m.View().Content), "▌") {
					t.Error("no marked line drawn")
				}
			})
		})
	}
	t.Run("an entry naming no file", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			r, _, _ := fileRig(t)
			h := r.h
			r.emit(event.ToolStarted, event.Tool{ToolID: "t1", Name: "Bash", Input: toolInput(t, map[string]any{"command": "ls"})})
			h.settle()
			h.send(key(tea.KeyUp, 0))
			h.send(key(tea.KeyF3, 0))
			v := h.m.ui.Explorer
			if h.m.ui.File.Open || h.m.ui.Finder.Open || !v.Open || v.Dirs[""] == nil || len(v.Dirs[""].Entries) != 5 {
				t.Errorf("F3 on a Bash entry: file %v, quick open %v, Explorer %+v", h.m.ui.File.Open, h.m.ui.Finder.Open, v)
			}
		})
	})
}

// TestQuickOpen is plan 17, criterion 3.3: / in the Explorer (amendment
// 12) lists the project's files,
// sorted, without credential files; typing narrows the list; Enter opens
// the file chosen; a cut listing says so.
func TestQuickOpen(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r, _, _ := fileRig(t)
		h := r.h
		h.send(key(tea.KeyF3, 0))
		h.send(typed('/'))
		v := h.m.ui.Finder
		want := []string{"data.bin", "docs/readme.md", "main.go", "src/Main.java", "web/app.ts"}
		if !v.Open || v.Total != 5 || !reflect.DeepEqual(v.Files, want) || v.Truncated {
			t.Fatalf("the quick open: %+v", v)
		}
		for _, k := range typedText("ma") {
			h.send(k)
		}
		if got := h.m.ui.Finder.Files; !reflect.DeepEqual(got, []string{"main.go", "src/Main.java"}) {
			t.Fatalf("\"ma\" keeps %q", got)
		}
		h.send(key(tea.KeyEnter, 0))
		if f := h.m.ui.File; !f.Open || f.Rel != "main.go" || f.Opened != "quick open" || h.m.ui.Finder.Open || len(f.Lines) != 100 {
			t.Fatalf("Enter opened %+v", f.Rel)
		}
		if !strings.Contains(plainText(h.m.View().Content), "quick open") {
			t.Error("the window does not say how the file was opened")
		}
	})
	synctest.Test(t, func(t *testing.T) {
		old := input.MaxListed
		input.MaxListed = 2
		t.Cleanup(func() { input.MaxListed = old })
		r, _, _ := fileRig(t)
		h := r.h
		h.send(key(tea.KeyF3, 0))
		h.send(typed('/'))
		if v := h.m.ui.Finder; !v.Truncated || v.Total != 2 || !strings.Contains(plainText(h.m.View().Content), "first 2 files") {
			t.Errorf("a cut listing: %+v\n%s", v, plainText(h.m.View().Content))
		}
	})
}

// TestFilterFiles: a path whose name holds the letters comes before a
// shorter one that holds them only across folders.
func TestFilterFiles(t *testing.T) {
	all := []string{"ab/c.go", "x/abc.go", "zz.go", "y/z/ABC.txt"}
	if got, want := filterFiles(all, "abc"), []string{"x/abc.go", "y/z/ABC.txt", "ab/c.go"}; !reflect.DeepEqual(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
	if got := filterFiles(all, ""); !reflect.DeepEqual(got, all) {
		t.Errorf("an empty query keeps %q", got)
	}
	if got := filterFiles(all, "qq"); len(got) != 0 {
		t.Errorf("\"qq\" keeps %q", got)
	}
}

// TestFileKeys is plan 17, criterion 3.4: the viewer's keys move, search
// and go to a line; Esc cancels a field, then goes back to the Explorer on
// the file (amendment 12), then gives zone B the focus, which at 120x40
// hides the Explorer's panel in zone B's place (plan 43, design D7 and the
// compact rule); the draft is kept.
func TestFileKeys(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r, _, _ := fileRig(t)
		h := r.h
		for _, k := range typedText("draft") {
			h.send(k)
		}
		h.send(key(tea.KeyF3, 0))
		h.send(typed('/'))
		for _, k := range typedText("main.go") {
			h.send(k)
		}
		h.send(key(tea.KeyEnter, 0))
		f := &h.m.ui.File
		if !f.Open || f.Rel != "main.go" {
			t.Fatalf("not opened: %+v", f.State)
		}
		rows := render.FileRows(h.m.ui)
		for _, c := range []struct {
			k      tea.KeyPressMsg
			cursor int
			left   int
		}{
			{key(tea.KeyDown, 0), 1, 0},
			{key(tea.KeyUp, 0), 0, 0},
			{key(tea.KeyPgDown, 0), rows, 0},
			{key(tea.KeyEnd, 0), 99, 0},
			{key(tea.KeyHome, 0), 0, 0},
			{key(tea.KeyRight, 0), 0, shiftCells},
			{key(tea.KeyLeft, 0), 0, 0},
		} {
			h.send(c.k)
			if f.Cursor != c.cursor || f.Left != c.left {
				t.Errorf("%s: cursor %d, left %d; want %d, %d", c.k, f.Cursor, f.Left, c.cursor, c.left)
			}
			if f.Top > f.Cursor || f.Cursor >= f.Top+rows {
				t.Errorf("%s: the cursor %d is out of the view from %d", c.k, f.Cursor, f.Top)
			}
		}
		h.send(typed('g'))
		for _, k := range typedText("4x2") {
			h.send(k)
		}
		if f.Field != 'g' || f.Typed != "42" {
			t.Errorf("the go-to field: %q %q", f.Field, f.Typed)
		}
		h.send(key(tea.KeyEnter, 0))
		if f.Field != 0 || f.Cursor != 41 {
			t.Errorf("g 42: cursor %d", f.Cursor)
		}
		h.send(typed('/'))
		for _, k := range typedText("line 7") {
			h.send(k)
		}
		h.send(key(tea.KeyEnter, 0))
		if len(f.Matches) != 11 || f.Match != 1 || f.Cursor != 69 || f.Searching {
			t.Fatalf("/line 7: %d matches, match %d, cursor %d", len(f.Matches), f.Match, f.Cursor)
		}
		h.send(typed('n'))
		if f.Match != 2 || f.Cursor != 70 {
			t.Errorf("n: match %d, cursor %d", f.Match, f.Cursor)
		}
		h.send(typed('N'))
		if f.Match != 1 || f.Cursor != 69 || !strings.Contains(plainText(h.m.View().Content), "match 2 of 11") {
			t.Errorf("N: match %d, cursor %d", f.Match, f.Cursor)
		}
		h.send(typed('/'))
		for _, k := range typedText("zz") {
			h.send(k)
		}
		h.send(key(tea.KeyEscape, 0))
		if f.Field != 0 || f.Query != "line 7" || !f.Open {
			t.Errorf("Esc in the field: field %q, query %q, open %v", f.Field, f.Query, f.Open)
		}
		h.send(typed('f'))
		// Full screen under the title bar (plan 21, item 5.7).
		if rows := strings.Split(plainText(h.m.View().Content), "\n"); !f.Full || !strings.Contains(rows[0], "AGENT COMMAND CENTER") || !strings.HasPrefix(rows[1], "╭") {
			t.Error("f: not full screen under the title bar")
		}
		h.send(typed('f'))
		if f.Full {
			t.Error("f again: still full screen")
		}
		h.send(key(tea.KeyF1, 0))
		if !h.m.ui.Keys {
			t.Error("F1: no key overlay")
		}
		h.send(key(tea.KeyEscape, 0))
		if h.m.ui.Keys || !f.Open {
			t.Errorf("Esc on the overlay: overlay %v, viewer %v", h.m.ui.Keys, f.Open)
		}
		h.send(key(tea.KeyEscape, 0))
		if v := h.m.ui.Explorer; f.Open || !v.Open || v.Cursor != "main.go" || h.m.ui.Focus != "R" {
			t.Errorf("Esc: viewer %v, Explorer %v on %q, focus %q", f.Open, v.Open, v.Cursor, h.m.ui.Focus)
		}
		// Esc at the top: zone B's focus, the Explorer's panel, in zone B's
		// place at 120x40, hidden (plan 43's compact rule), no window.
		h.send(key(tea.KeyEscape, 0))
		if h.m.ui.Explorer.Open || h.m.ui.Focus != "B" || render.WindowOpen(h.m.ui) {
			t.Errorf("Esc at the top: Explorer %v, focus %q, a window %v", h.m.ui.Explorer.Open, h.m.ui.Focus, render.WindowOpen(h.m.ui))
		}
		if got := h.m.edit.Text(); got != "draft" {
			t.Errorf("the draft is %q", got)
		}
	})
}

// TestFileStatesApp is plan 17, criterion 3.5: a file outside the project,
// a credential file, a binary, a too large and a missing file are refused
// with their reason; "reading…" and "colouring…" show while they run.
func TestFileStatesApp(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r, _, _ := fileRig(t)
		h := r.h
		elsewhere := filepath.Join(t.TempDir(), "x.go")
		if err := os.WriteFile(elsewhere, []byte("package x\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		for _, c := range []struct{ path, prefix string }{
			{elsewhere, "outside the project: "},
			{".env", "a credential file: not opened"},
			{"data.bin", "binary file, not shown · "},
			{"missing.go", "not found: "},
		} {
			openPath(h, c.path)
			if f := h.m.ui.File; !f.Open || !strings.HasPrefix(f.State, c.prefix) || len(f.Lines) != 0 {
				t.Errorf("%s: state %q, %d lines", c.path, f.State, len(f.Lines))
			} else if !strings.Contains(plainText(h.m.View().Content), c.prefix) {
				t.Errorf("%s: %q not drawn", c.path, c.prefix)
			}
		}
		old := input.MaxFileSize
		input.MaxFileSize = 10
		openPath(h, "main.go")
		input.MaxFileSize = old
		if f := h.m.ui.File; !strings.HasPrefix(f.State, "too large to show: ") || !f.Failed {
			t.Errorf("a large file: %q", f.State)
		}

		// Each step's command held back, to see the state it leaves.
		cmds := h.m.openFile(render.FileTarget{Path: "main.go", Opened: "quick open"})
		if f := h.m.ui.File; f.State != "reading…" || !strings.Contains(plainText(h.m.View().Content), "reading…") {
			t.Errorf("before the read: %q", f.State)
		}
		var read []tea.Msg
		for _, c := range cmds {
			read = append(read, msgsOf(c)...)
		}
		if len(read) != 1 {
			t.Fatalf("the read gave %d messages", len(read))
		}
		_, cmd := h.m.Update(read[0])
		if f := h.m.ui.File; f.State != "" || !f.Colouring || !strings.Contains(plainText(h.m.View().Content), "colouring…") {
			t.Errorf("before the colouring: state %q, colouring %v", f.State, f.Colouring)
		}
		for _, msg := range msgsOf(cmd) {
			h.m.Update(msg)
		}
		if f := h.m.ui.File; f.Colouring || f.Hl.Lang != "Go" {
			t.Errorf("after the colouring: colouring %v, language %q", f.Colouring, f.Hl.Lang)
		}
	})
}

// TestFileStale is plan 17, criterion 3.6: a read, colouring, search or
// listing asked for a file or a quick open since replaced is dropped.
func TestFileStale(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r, _, _ := fileRig(t)
		h := r.h
		first := h.m.openFile(render.FileTarget{Path: "web/app.ts", Opened: "quick open"})
		openPath(h, "main.go")
		gen := h.m.fileGen
		for _, c := range first {
			for _, msg := range msgsOf(c) {
				h.m.Update(msg)
			}
		}
		f := &h.m.ui.File
		if f.Rel != "main.go" || len(f.Lines) != 100 || f.Hl.Lang != "Go" {
			t.Fatalf("an older read replaced the file: %q, %d lines, %q", f.Rel, len(f.Lines), f.Hl.Lang)
		}
		h.m.Update(hlMsg{gen: gen - 1, hl: render.Highlighted{Lang: "TypeScript"}})
		if f.Hl.Lang != "Go" {
			t.Errorf("an older colouring was taken: %q", f.Hl.Lang)
		}
		h.m.search("line 9")
		h.m.Update(searchMsg{gen: gen, search: h.m.searchGen - 1, query: "old", matches: []render.FileMatch{{Line: 3, N: 1}}})
		if !f.Searching || len(f.Matches) != 0 {
			t.Errorf("an older search was taken: %d matches", len(f.Matches))
		}
		h.m.Update(searchMsg{gen: gen - 1, search: h.m.searchGen, query: "line 9", matches: []render.FileMatch{{Line: 3, N: 1}}})
		if !f.Searching || len(f.Matches) != 0 {
			t.Errorf("a search of an older file was taken: %d matches", len(f.Matches))
		}
		h.m.openFinder()
		fg := h.m.finderGen
		h.m.Update(filesMsg{gen: fg - 1, files: []string{"old.go"}})
		if v := h.m.ui.Finder; v.State != "listing…" || len(v.Files) != 0 {
			t.Errorf("an older listing was taken: %+v", v)
		}
		h.m.Update(filesMsg{gen: fg, files: []string{"new.go"}})
		if v := h.m.ui.Finder; v.State != "" || !reflect.DeepEqual(v.Files, []string{"new.go"}) {
			t.Errorf("the current listing: %+v", v)
		}
	})
}

// TestFileMouse is plan 17, criterion 3.7: a drag in the window copies the
// file's text exactly; a right click behind it does nothing (a left click
// closes it since plan 58: TestFileClickOutside); the wheel scrolls it.
func TestFileMouse(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r, clip, _ := fileRig(t)
		h := r.h
		openPath(h, "main.go")
		x0, y0 := at(t, h, "// line 3", 0)
		x1, y1 := at(t, h, "// line 5", 0)
		dragCopy(t, h, x0, y0, x1+len("// line 5")-1, y1)
		if got, want := clip.last(), "// line 3\n// line 4\n// line 5"; got != want {
			t.Errorf("copied %q, want %q", got, want)
		}
		h.send(key(tea.KeyEscape, 0)) // the selection cleared, the viewer kept
		if !h.m.ui.Sel.Empty() || !h.m.ui.File.Open {
			t.Fatalf("Esc: selection %v, viewer %v", h.m.ui.Sel, h.m.ui.File.Open)
		}
		focus := h.m.ui.Focus
		mouseAt(t, h, tea.MouseClickMsg{X: 0, Y: 0, Button: tea.MouseRight})
		mouseAt(t, h, tea.MouseReleaseMsg{X: 0, Y: 0, Button: tea.MouseRight})
		if !h.m.ui.Sel.Empty() || h.m.ui.Focus != focus || !h.m.ui.File.Open {
			t.Errorf("a right click behind: selection %v, focus %q, viewer %v", h.m.ui.Sel, h.m.ui.Focus, h.m.ui.File.Open)
		}
		rows := render.FileRows(h.m.ui)
		top := render.FileTop(h.m.ui.File, rows)
		x, y := at(t, h, "// line 10", 0)
		mouseAt(t, h, tea.MouseWheelMsg{X: x, Y: y, Button: tea.MouseWheelDown})
		if got := render.FileTop(h.m.ui.File, rows); got != top+wheelRows {
			t.Errorf("the wheel: top %d, want %d", got, top+wheelRows)
		}
	})
}

// uiDiff names the fields of two interfaces that differ, with their values,
// "" when none does.
func uiDiff(a, b render.UI) string {
	va, vb := reflect.ValueOf(a), reflect.ValueOf(b)
	var out []string
	for i := range va.NumField() {
		if fa, fb := va.Field(i).Interface(), vb.Field(i).Interface(); !reflect.DeepEqual(fa, fb) {
			out = append(out, fmt.Sprintf("%s %+v, not %+v", va.Type().Field(i).Name, fa, fb))
		}
	}
	return strings.Join(out, "; ")
}

// TestFileClickOutside is plan 58 (brief amendment 60): a left click outside
// the window closes it as Esc's last step does and does nothing else, the
// interface it leaves the one Esc leaves: the viewer back to the Explorer's
// panel with the focus, its cursor on the file (criterion 58.1); the quick
// open closed, the viewer behind it in front again (58.3). A click on the
// frame, the footer or the text, a right or middle click, a motion or a
// release outside, the wheel and a drag released outside leave it open
// (58.2); full screen, the first row is outside it and the second inside
// (design D3).
func TestFileClickOutside(t *testing.T) {
	// shut opens a window as open does in a rig of its own, then closes it
	// with Esc when cell is nil, else with a left click on the cell outside
	// the window cell picks, and gives the interface left. What is the rig's
	// own is left out: its project folder, a placeholder wherever the
	// interface names it, and the frames' memo, never a source of what is
	// drawn.
	shut := func(t *testing.T, open func(h *harness), cell func(w render.Box) (int, int)) render.UI {
		r, _, root := fileRig(t)
		h := r.h
		open(h)
		h.m.View()
		if cell == nil {
			h.send(key(tea.KeyEscape, 0))
		} else {
			x, y := cell(h.m.screen.Window)
			if h.m.screen.Window.Holds(x, y) || x < 0 || y < 0 || x >= h.m.ui.Width || y >= h.m.ui.Height {
				t.Fatalf("(%d, %d) is not on the screen outside the window %+v", x, y, h.m.screen.Window)
			}
			mouseAt(t, h, click(x, y))
			mouseAt(t, h, tea.MouseReleaseMsg{X: x, Y: y, Button: tea.MouseLeft})
		}
		ui := h.m.ui
		ui.Path = strings.Replace(ui.Path, root, "<root>", 1)
		ui.Resolved = strings.Replace(ui.Resolved, root, "<root>", 1)
		ui.File.Path = strings.Replace(ui.File.Path, root, "<root>", 1)
		if ui.Explorer.Name == filepath.Base(root) {
			ui.Explorer.Name = "<root>"
		}
		ui.Memo = nil
		return ui
	}
	viewer := func(h *harness) { openPath(h, "src/Main.java") }
	finder := func(h *harness) {
		for _, c := range h.m.openFinder() {
			h.exec(c)
		}
		h.settle()
	}
	cells := map[string]func(w render.Box) (int, int){
		"the title bar": func(render.Box) (int, int) { return 0, 0 },
		"left of it":    func(w render.Box) (int, int) { return w.X - 1, w.Y + w.H/2 },
		"right of it":   func(w render.Box) (int, int) { return w.X + w.W, w.Y + w.H/2 },
		"under it":      func(w render.Box) (int, int) { return w.X + w.W/2, w.Y + w.H },
	}
	for _, o := range []struct {
		name  string
		open  func(h *harness)
		check func(ui render.UI) string // "" when the window closed as it should
	}{
		{"the viewer", viewer, func(ui render.UI) string {
			if v := ui.Explorer; ui.File.Open || !v.Open || v.Cursor != "src/Main.java" || !v.Opened["src"] || ui.Focus != "R" {
				return fmt.Sprintf("viewer %v, Explorer %v, cursor %q, src open %v, focus %q", ui.File.Open, v.Open, v.Cursor, v.Opened["src"], ui.Focus)
			}
			return ""
		}},
		{"the quick open", finder, func(ui render.UI) string {
			if render.WindowOpen(ui) {
				return fmt.Sprintf("viewer %v, quick open %v", ui.File.Open, ui.Finder.Open)
			}
			return ""
		}},
		{"the quick open over the viewer", func(h *harness) { viewer(h); finder(h) }, func(ui render.UI) string {
			if ui.Finder.Open || !ui.File.Open || ui.File.Rel != "src/Main.java" {
				return fmt.Sprintf("quick open %v, viewer %v %q", ui.Finder.Open, ui.File.Open, ui.File.Rel)
			}
			return ""
		}},
	} {
		for name, cell := range cells {
			t.Run(o.name+", "+name, func(t *testing.T) {
				synctest.Test(t, func(t *testing.T) {
					esc := shut(t, o.open, nil)
					got := shut(t, o.open, cell)
					if why := o.check(got); why != "" {
						t.Errorf("after the click: %s", why)
					}
					if d := uiDiff(got, esc); d != "" {
						t.Errorf("the click left another interface than Esc: %s", d)
					}
				})
			})
		}
	}

	t.Run("inside, and the rest", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			r, clip, _ := fileRig(t)
			h := r.h
			openPath(h, "main.go")
			h.m.View()
			w := h.m.screen.Window
			// The place recorded is the window drawn: its title's corner, and
			// the corner opposite.
			if rows := strings.Split(h.frame(), "\n"); !windowTitled(h, w.X, w.Y, "main.go") || len(rows) <= w.Y+w.H-1 ||
				len([]rune(rows[w.Y+w.H-1])) <= w.X+w.W-1 || []rune(rows[w.Y+w.H-1])[w.X+w.W-1] != '╯' {
				t.Fatalf("the window recorded, %+v, is not the one drawn:\n%s", w, h.frame())
			}
			x3, y3 := at(t, h, "// line 3", 0)
			x5, y5 := at(t, h, "// line 5", 0)
			// A drag on the text released outside, left of the window, copies:
			// the release clamps the selection to the window, as before.
			mouseAt(t, h, click(x3, y3))
			mouseAt(t, h, tea.MouseMotionMsg{X: x5 + len("// line 5") - 1, Y: y5, Button: tea.MouseLeft})
			if _, ok := mouseAt(t, h, tea.MouseReleaseMsg{X: w.X - 1, Y: y5, Button: tea.MouseLeft}); !ok || !h.m.ui.File.Open {
				t.Fatalf("a drag released outside: copied %v, viewer %v", ok, h.m.ui.File.Open)
			}
			if got := clip.last(); !strings.HasPrefix(got, "// line 3\n// line 4\n") {
				t.Errorf("copied %q, want lines 3 and 4 at least", got)
			}
			h.send(key(tea.KeyEscape, 0)) // the selection cleared, the viewer kept
			focus := h.m.ui.Focus
			release := func(x, y int, b tea.MouseButton) tea.Msg { return tea.MouseReleaseMsg{X: x, Y: y, Button: b} }
			for _, c := range []struct {
				name string
				msgs []tea.Msg
			}{
				{"a click on the frame", []tea.Msg{click(w.X, w.Y), release(w.X, w.Y, tea.MouseLeft)}},
				{"a click on the footer", []tea.Msg{click(w.X+w.W/2, w.Y+w.H-1), release(w.X+w.W/2, w.Y+w.H-1, tea.MouseLeft)}},
				{"a click on the text", []tea.Msg{click(x3, y3), release(x3, y3, tea.MouseLeft)}},
				{"a right click outside", []tea.Msg{tea.MouseClickMsg{Button: tea.MouseRight}, release(0, 0, tea.MouseRight)}},
				{"a middle click outside", []tea.Msg{tea.MouseClickMsg{Button: tea.MouseMiddle}, release(0, 0, tea.MouseMiddle)}},
				{"a motion outside", []tea.Msg{tea.MouseMotionMsg{}}},
				{"a release outside", []tea.Msg{release(0, 0, tea.MouseLeft)}},
			} {
				for _, msg := range c.msgs {
					mouseAt(t, h, msg)
				}
				if !h.m.ui.File.Open || h.m.ui.Focus != focus || !h.m.ui.Sel.Empty() {
					t.Errorf("%s: viewer %v, focus %q, selection %v", c.name, h.m.ui.File.Open, h.m.ui.Focus, h.m.ui.Sel)
				}
			}
			rows := render.FileRows(h.m.ui)
			top := render.FileTop(h.m.ui.File, rows)
			mouseAt(t, h, tea.MouseWheelMsg{Button: tea.MouseWheelDown})
			if got := render.FileTop(h.m.ui.File, rows); !h.m.ui.File.Open || got != top+wheelRows {
				t.Errorf("the wheel outside: viewer %v, top %d, want %d", h.m.ui.File.Open, got, top+wheelRows)
			}
		})
	})

	t.Run("full screen", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			r, _, _ := fileRig(t)
			h := r.h
			openPath(h, "main.go")
			h.send(typed('f'))
			h.m.View()
			if w := h.m.screen.Window; !h.m.ui.File.Full || w != (render.Box{X: 0, Y: 1, W: h.m.ui.Width, H: h.m.ui.Height - 1}) {
				t.Fatalf("full screen %v, the window %+v", h.m.ui.File.Full, w)
			}
			mouseAt(t, h, click(5, 1))
			mouseAt(t, h, tea.MouseReleaseMsg{X: 5, Y: 1, Button: tea.MouseLeft})
			if !h.m.ui.File.Open {
				t.Fatal("a click on the window's top row closed it")
			}
			mouseAt(t, h, click(5, 0))
			mouseAt(t, h, tea.MouseReleaseMsg{X: 5, Y: 0, Button: tea.MouseLeft})
			if h.m.ui.File.Open || !h.m.ui.Explorer.Open || h.m.ui.Explorer.Cursor != "main.go" || h.m.ui.Focus != "R" {
				t.Errorf("a click on the first row: viewer %v, Explorer %v, cursor %q, focus %q",
					h.m.ui.File.Open, h.m.ui.Explorer.Open, h.m.ui.Explorer.Cursor, h.m.ui.Focus)
			}
		})
	})
}
