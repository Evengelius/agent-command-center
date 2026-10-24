package app

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"testing/synctest"

	tea "charm.land/bubbletea/v2"

	"github.com/Evengelius/agent-command-center/internal/event"
	"github.com/Evengelius/agent-command-center/internal/input"
	"github.com/Evengelius/agent-command-center/internal/render"
)

// explorerRig is fileRig with git's status given by a test double.
func explorerRig(t *testing.T, g *fakeGit) (*harness, string) {
	r, _, root := fileRig(t)
	r.h.m.opt.Git = g
	return r.h, root
}

// treeNames is the tree's rows as drawn, each name after its indent.
func treeNames(v render.ExplorerView) []string {
	var out []string
	for _, r := range render.ExplorerRows(v) {
		out = append(out, strings.Repeat("  ", r.Depth)+r.Name)
	}
	return out
}

// TestExplorerOpen is plan 17, criterion 5.3, docked since plan 43 (design
// D6): F3 with no file in front shows the Explorer's panel with the focus,
// the project's top level under the root row that names the project's
// folder, folders first, a credential file left out; its keys in the
// footer.
func TestExplorerOpen(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, root := explorerRig(t, &fakeGit{})
		h.send(key(tea.KeyF3, 0))
		v := h.m.ui.Explorer
		want := []string{"docs", "src", "web", "data.bin", "main.go"}
		if got := treeNames(v); !v.Open || strings.Join(got, ",") != strings.Join(want, ",") {
			t.Fatalf("the Explorer: open %v, rows %q", v.Open, got)
		}
		if v.Name != filepath.Base(root) || v.Cursor != "" || h.m.ui.File.Open || h.m.ui.Finder.Open || h.m.ui.Focus != "R" {
			t.Errorf("name %q, cursor %q, viewer %v, quick open %v, focus %q", v.Name, v.Cursor, h.m.ui.File.Open, h.m.ui.Finder.Open, h.m.ui.Focus)
		}
		frame := plainText(h.m.View().Content)
		for _, s := range []string{"EXPLORER", "⌄ 🗀  " + filepath.Base(root), "› 🗀  docs", "🗎  main.go", "Esc back", "F3 Explorer"} {
			if !strings.Contains(frame, s) {
				t.Errorf("%q not drawn:\n%s", s, frame)
			}
		}
		if strings.Contains(frame, ".env") {
			t.Error("the credential file is listed")
		}
	})
}

// TestExplorerKeys is plan 17, criteria 5.4 and 5.5, docked since plan 43
// (designs D5 and D7): → and Enter open a folder, ← closes it or goes to
// its folder, Esc closes the folder the cursor is in, then gives zone B the
// focus, which at 120x40, the panel in zone B's place, hides it (plan 43's
// compact rule; docked, the panel is kept: TestExplorerDocked); / opens the
// quick open, whose Esc comes back; F3 shows the panel as it was left and
// hides it; F3 in the viewer or the quick open closes it and hides the
// panel, F2 closes them all; a paste is dropped, the draft kept.
func TestExplorerKeys(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, _ := explorerRig(t, &fakeGit{})
		for _, k := range typedText("draft") {
			h.send(k)
		}
		h.send(key(tea.KeyF3, 0))
		v := &h.m.ui.Explorer
		h.send(key(tea.KeyRight, 0)) // docs opened
		if !v.Opened["docs"] || v.Cursor != "docs" || v.Dirs["docs"] == nil {
			t.Fatalf("→ on docs: opened %v, cursor %q", v.Opened, v.Cursor)
		}
		h.send(key(tea.KeyRight, 0)) // into it
		if v.Cursor != "docs/readme.md" {
			t.Fatalf("→ on an open folder: cursor %q", v.Cursor)
		}
		h.send(key(tea.KeyLeft, 0)) // to its folder
		if v.Cursor != "docs" || !v.Opened["docs"] {
			t.Fatalf("← on a file: cursor %q, docs open %v", v.Cursor, v.Opened["docs"])
		}
		h.send(key(tea.KeyLeft, 0)) // closed
		if v.Opened["docs"] || v.Cursor != "docs" {
			t.Fatalf("← on an open folder: open %v, cursor %q", v.Opened["docs"], v.Cursor)
		}
		h.send(key(tea.KeyDown, 0))
		h.send(key(tea.KeyEnter, 0)) // src opened
		h.send(key(tea.KeyDown, 0))  // src/Main.java
		if !v.Opened["src"] || v.Cursor != "src/Main.java" {
			t.Fatalf("Enter on src, ↓: open %v, cursor %q", v.Opened["src"], v.Cursor)
		}
		h.send(key(tea.KeyEscape, 0)) // src closed, the cursor on it
		if v.Opened["src"] || v.Cursor != "src" || !v.Open {
			t.Fatalf("Esc in src: open %v, cursor %q, Explorer %v", v.Opened["src"], v.Cursor, v.Open)
		}
		h.send(key(tea.KeyEnd, 0))
		if v.Cursor != "main.go" {
			t.Fatalf("End: cursor %q", v.Cursor)
		}
		h.send(key(tea.KeyHome, 0))
		if v.Cursor != "docs" {
			t.Fatalf("Home: cursor %q", v.Cursor)
		}
		h.send(tea.PasteMsg{Content: "pasted"})
		if got := h.m.edit.Text(); got != "draft" {
			t.Errorf("a paste in the Explorer reached the draft: %q", got)
		}
		h.send(typed('/'))
		if !h.m.ui.Finder.Open {
			t.Fatal("/ opened no quick open")
		}
		h.send(key(tea.KeyEscape, 0))
		if h.m.ui.Finder.Open || !v.Open {
			t.Fatalf("Esc in the quick open: quick open %v, Explorer %v", h.m.ui.Finder.Open, v.Open)
		}
		h.send(key(tea.KeyEscape, 0)) // at the top: zone B's focus, the panel in its place hidden
		if v.Open || h.m.ui.Focus != "B" || !strings.Contains(h.frame(), "B · CONVERSATION") {
			t.Fatalf("Esc at the top: Explorer %v, focus %q", v.Open, h.m.ui.Focus)
		}
		h.send(key(tea.KeyF3, 0)) // as it was left
		if !v.Open || v.Cursor != "docs" || h.m.ui.Focus != "R" {
			t.Fatalf("F3 from zone B: open %v, cursor %q, focus %q", v.Open, v.Cursor, h.m.ui.Focus)
		}
		h.send(key(tea.KeyF3, 0))
		if v.Open || h.m.ui.File.Open || h.m.ui.Finder.Open || h.m.ui.Focus != "B" {
			t.Fatalf("F3 in the Explorer: open %v, a window open, or the focus %q", v.Open, h.m.ui.Focus)
		}
		if got := h.m.edit.Text(); got != "draft" {
			t.Errorf("the draft: %q", got)
		}
	})
	for _, c := range []struct {
		name string
		to   func(h *harness)
	}{
		{"the Explorer", func(h *harness) {}},
		{"the viewer", func(h *harness) { h.send(key(tea.KeyEnd, 0)); h.send(key(tea.KeyEnter, 0)) }},
		{"the quick open", func(h *harness) { h.send(typed('/')) }},
	} {
		t.Run("F3 and F2 from "+c.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				for _, k := range []tea.KeyPressMsg{key(tea.KeyF3, 0), key(tea.KeyF2, 0)} {
					h, _ := explorerRig(t, &fakeGit{})
					h.send(key(tea.KeyF3, 0))
					c.to(h)
					if !h.m.ui.Explorer.Open || c.name != "the Explorer" && !render.WindowOpen(h.m.ui) {
						t.Fatalf("the panel shown %v, a window open %v", h.m.ui.Explorer.Open, render.WindowOpen(h.m.ui))
					}
					h.send(k)
					// F3 hides the panel too; F2 closes the windows and keeps the
					// panel, which is none (plan 43's review, design D8).
					if render.WindowOpen(h.m.ui) || h.m.ui.Explorer.Open != (k.String() == "f2") {
						t.Errorf("%s: viewer %v, quick open %v, Explorer %v", k.String(),
							h.m.ui.File.Open, h.m.ui.Finder.Open, h.m.ui.Explorer.Open)
					}
				}
			})
		})
	}
}

// TestExplorerBack is plan 17, criterion 5.5, docked since plan 43 (design
// D7): Esc in the viewer goes back to the Explorer's panel with the focus,
// its cursor on the file and each folder above it open, a file opened from
// a tool entry too; Esc at its top gives zone B the focus, which at 120x40
// hides the panel in zone B's place (plan 43's compact rule).
func TestExplorerBack(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, _ := explorerRig(t, &fakeGit{})
		h.send(key(tea.KeyF3, 0))
		h.send(key(tea.KeyDown, 0))
		h.send(key(tea.KeyRight, 0))
		h.send(key(tea.KeyDown, 0))
		h.send(key(tea.KeyEnter, 0))
		if f := h.m.ui.File; !f.Open || f.Rel != "src/Main.java" || f.Opened != "Explorer" {
			t.Fatalf("Enter on a file: %+v", f.Rel)
		}
		if !strings.Contains(plainText(h.m.View().Content), "Explorer") {
			t.Error("the viewer does not say it was opened from the Explorer")
		}
		h.send(key(tea.KeyEscape, 0))
		v := h.m.ui.Explorer
		if h.m.ui.File.Open || !v.Open || v.Cursor != "src/Main.java" || !v.Opened["src"] || h.m.ui.Focus != "R" {
			t.Fatalf("Esc in the viewer: viewer %v, Explorer %v, cursor %q, focus %q", h.m.ui.File.Open, v.Open, v.Cursor, h.m.ui.Focus)
		}
	})
	synctest.Test(t, func(t *testing.T) {
		r, _, root := fileRig(t)
		h := r.h
		h.m.opt.Git = &fakeGit{}
		r.emit(event.ToolStarted, event.Tool{ToolID: "t1", Name: "Read", Input: toolInput(t, map[string]any{"file_path": filepath.Join(root, "docs", "readme.md")})})
		h.settle()
		h.send(key(tea.KeyUp, 0))
		h.send(key(tea.KeyF3, 0))
		if f := h.m.ui.File; !f.Open || f.Rel != "docs/readme.md" || h.m.ui.Explorer.Open {
			t.Fatalf("F3 on the Read entry: viewer %v %q, Explorer %v", f.Open, f.Rel, h.m.ui.Explorer.Open)
		}
		h.send(key(tea.KeyEscape, 0))
		v := h.m.ui.Explorer
		if h.m.ui.File.Open || !v.Open || v.Cursor != "docs/readme.md" || !v.Opened["docs"] || v.Dirs["docs"] == nil || h.m.ui.Focus != "R" {
			t.Fatalf("Esc after a tool entry: viewer %v, Explorer %v, cursor %q, opened %v, focus %q", h.m.ui.File.Open, v.Open, v.Cursor, v.Opened, h.m.ui.Focus)
		}
		if !strings.Contains(plainText(h.m.View().Content), "readme.md") {
			t.Error("the file's row is not drawn")
		}
		h.send(key(tea.KeyEscape, 0)) // docs closed
		h.send(key(tea.KeyEscape, 0)) // zone B's focus, the panel in its place hidden
		if h.m.ui.Explorer.Open || h.m.ui.Explorer.Opened["docs"] || h.m.ui.Focus != "B" {
			t.Errorf("two Esc: Explorer %v, docs open %v, focus %q", h.m.ui.Explorer.Open, h.m.ui.Explorer.Opened["docs"], h.m.ui.Focus)
		}
	})
}

// TestExplorerMarks is plan 17, criterion 5.6: git's markers beside
// the rows, a folder holding a change dotted, an ignored file dimmed and
// listed; git missing, no repository or git failing are notes.
func TestExplorerMarks(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		g := &fakeGit{marks: input.GitMarks{
			Files:   map[string]byte{"src/Main.java": 'M', "web/": 'U', "main.go": '!'},
			Ignored: map[string]bool{"data.bin": true},
		}}
		h, root := explorerRig(t, g)
		h.send(key(tea.KeyF3, 0))
		v := h.m.ui.Explorer
		if v.Git.Note != "" || !v.Git.Dirty["src"] || v.Git.Marks["main.go"] != '!' {
			t.Fatalf("git's view: %+v", v.Git)
		}
		if got := g.calls; len(got) != 1 || got[0] != "status "+root {
			t.Errorf("git calls: %q", got)
		}
		frame := plainText(h.m.View().Content)
		for _, row := range []struct{ name, mark string }{{"src", "●"}, {"web", "U"}, {"main.go", "!"}} {
			if !rowHas(frame, row.name, row.mark) {
				t.Errorf("%s has no %s:\n%s", row.name, row.mark, frame)
			}
		}
		if !strings.Contains(frame, "data.bin") {
			t.Error("the ignored file is hidden")
		}
	})
	for _, c := range []struct {
		err  error
		note string
	}{
		{input.NotRepoError{Dir: "x"}, "not a git repository"},
		{input.ErrNoGit, "git not found"},
		{errors.New("index locked"), "git: index locked"},
	} {
		t.Run(c.note, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				h, _ := explorerRig(t, &fakeGit{statusErr: c.err})
				h.send(key(tea.KeyF3, 0))
				v := h.m.ui.Explorer
				if v.Git.Note != c.note || len(v.Dirs[""].Entries) != 5 {
					t.Errorf("note %q, want %q; %d entries", v.Git.Note, c.note, len(v.Dirs[""].Entries))
				}
				if !strings.Contains(plainText(h.m.View().Content), c.note) {
					t.Errorf("%q not drawn", c.note)
				}
			})
		})
	}
}

// rowHas is true when a row of frame holds name, then mark after it.
func rowHas(frame, name, mark string) bool {
	for _, l := range strings.Split(frame, "\n") {
		if i := strings.Index(l, " "+name+" "); i >= 0 && strings.Contains(l[i+len(name)+1:], " "+mark+" ") {
			return true
		}
	}
	return false
}

// TestExplorerStale is plan 17, criterion 5.7: a folder's read and git's
// status asked for an Explorer since closed are dropped; reopened, it keeps
// its open folders and its cursor.
func TestExplorerStale(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, _ := explorerRig(t, &fakeGit{})
		h.send(key(tea.KeyF3, 0))
		h.send(key(tea.KeyDown, 0))
		h.send(key(tea.KeyRight, 0)) // src opened
		gen := h.m.explorerGen
		h.m.Update(dirMsg{gen: gen - 1, rel: "", entries: []input.TreeEntry{{Name: "old.go"}}})
		h.m.Update(statusMsg{gen: gen - 1, marks: input.GitMarks{Files: map[string]byte{"main.go": 'M'}}})
		v := h.m.ui.Explorer
		if len(v.Dirs[""].Entries) != 5 || v.Git.Marks["main.go"] != 0 {
			t.Fatalf("an older read was taken: %d entries, marks %v", len(v.Dirs[""].Entries), v.Git.Marks)
		}
		h.send(key(tea.KeyF3, 0)) // closed
		h.m.Update(dirMsg{gen: gen, rel: "web", entries: []input.TreeEntry{{Name: "late.go"}}})
		h.m.Update(statusMsg{gen: gen, err: errors.New("late")})
		v = h.m.ui.Explorer
		if v.Dirs["web"] != nil || v.Git.Failed {
			t.Errorf("a read for the closed Explorer was taken: %+v, %+v", v.Dirs["web"], v.Git)
		}
		h.m.Update(dirMsg{gen: h.m.explorerGen, rel: "web", entries: []input.TreeEntry{{Name: "late.go"}}})
		if h.m.ui.Explorer.Dirs["web"] != nil {
			t.Error("a read was taken while the Explorer is closed")
		}
		h.send(key(tea.KeyF3, 0)) // reopened as it was left
		v = h.m.ui.Explorer
		if !v.Open || v.Cursor != "src" || !v.Opened["src"] || v.Dirs["src"] == nil || len(v.Dirs["src"].Entries) != 1 {
			t.Errorf("reopened: open %v, cursor %q, opened %v, src %+v", v.Open, v.Cursor, v.Opened, v.Dirs["src"])
		}
	})
}

// TestExplorerDeleted is plan 17, criterion 6.3: the deleted paths git
// reports listed; a deleted file's row with its D (the panel has no path
// row since plan 43, design D6), and Enter or a click on the cursor's row
// opens nothing; → opens a deleted folder without reading the disk, Esc
// closes it; the Explorer opened again reads no deleted folder.
func TestExplorerDeleted(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, _ := explorerRig(t, &fakeGit{marks: input.GitMarks{Files: map[string]byte{"gone/a.txt": 'D', "old.go": 'D', "src/lost.go": 'D'}}})
		h.send(key(tea.KeyF3, 0))
		v := &h.m.ui.Explorer
		want := "docs,gone,src,web,data.bin,main.go,old.go"
		if got := strings.Join(treeNames(*v), ","); got != want {
			t.Fatalf("rows %q, want %q", got, want)
		}
		h.send(key(tea.KeyEnd, 0))
		frame := plainText(h.m.View().Content)
		if v.Cursor != "old.go" || !rowHas(frame, "old.go", "D") || !rowHas(frame, "gone", "D") {
			t.Fatalf("the cursor on old.go: %q\n%s", v.Cursor, frame)
		}
		h.send(key(tea.KeyEnter, 0))
		x, y := at(t, h, "old.go", 0)
		mouseAt(t, h, click(x, y))
		if h.m.ui.File.Open || !v.Open || v.Cursor != "old.go" {
			t.Fatalf("Enter and a click on a deleted file: viewer %v, Explorer %v, cursor %q", h.m.ui.File.Open, v.Open, v.Cursor)
		}
		h.send(key(tea.KeyHome, 0))
		h.send(key(tea.KeyDown, 0))
		h.send(key(tea.KeyRight, 0)) // gone opened
		if !v.Opened["gone"] || v.Dirs["gone"] != nil {
			t.Fatalf("→ on a deleted folder: opened %v, read %v", v.Opened["gone"], v.Dirs["gone"] != nil)
		}
		h.send(key(tea.KeyRight, 0))
		if v.Cursor != "gone/a.txt" {
			t.Fatalf("→ in a deleted folder: cursor %q", v.Cursor)
		}
		h.send(key(tea.KeyF3, 0))
		h.send(key(tea.KeyF3, 0)) // opened again, gone still open
		if !v.Open || v.Dirs["gone"] != nil || !strings.Contains(strings.Join(treeNames(*v), ","), "gone,  a.txt") {
			t.Fatalf("reopened: open %v, gone read %v, rows %q", v.Open, v.Dirs["gone"] != nil, treeNames(*v))
		}
		h.send(key(tea.KeyEscape, 0))
		if v.Opened["gone"] || v.Cursor != "gone" {
			t.Errorf("Esc in a deleted folder: open %v, cursor %q", v.Opened["gone"], v.Cursor)
		}
		h.send(key(tea.KeyDown, 0))
		h.send(key(tea.KeyRight, 0)) // src, read from the disk, its deleted file among its entries
		if got := strings.Join(treeNames(*v), ","); v.Dirs["src"] == nil || !strings.Contains(got, "  lost.go") {
			t.Errorf("src opened: rows %q", got)
		}
	})
}

// TestExplorerMouse is plan 17, criterion 5.8, on the docked panel (plan
// 43, design D7): the wheel over it moves the cursor, a click puts it on a
// row, a click on the cursor's row is Enter; nothing is selected and a
// click outside every panel does nothing.
func TestExplorerMouse(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, _ := explorerRig(t, &fakeGit{})
		h.send(key(tea.KeyF3, 0))
		v := &h.m.ui.Explorer
		x, y := at(t, h, "main.go", 0)
		mouseAt(t, h, click(x, y))
		mouseAt(t, h, tea.MouseReleaseMsg{X: x, Y: y, Button: tea.MouseLeft})
		if v.Cursor != "main.go" || h.m.ui.File.Open || !h.m.ui.Sel.Empty() {
			t.Fatalf("a click on main.go: cursor %q, viewer %v, selection %v", v.Cursor, h.m.ui.File.Open, h.m.ui.Sel)
		}
		mouseAt(t, h, click(x, y))
		if f := h.m.ui.File; !f.Open || f.Rel != "main.go" || f.Opened != "Explorer" {
			t.Fatalf("a click on the cursor's row: viewer %v %q", f.Open, f.Rel)
		}
		h.send(key(tea.KeyEscape, 0))
		x, y = at(t, h, "docs", 0)
		mouseAt(t, h, tea.MouseWheelMsg{X: x, Y: y, Button: tea.MouseWheelUp})
		if v.Cursor != "src" { // wheelRows entries up from main.go
			t.Fatalf("the wheel up: cursor %q", v.Cursor)
		}
		mouseAt(t, h, tea.MouseWheelMsg{X: x, Y: y, Button: tea.MouseWheelDown})
		if v.Cursor != "main.go" {
			t.Errorf("the wheel down: cursor %q", v.Cursor)
		}
		mouseAt(t, h, click(x, y))
		if v.Cursor != "docs" || v.Opened["docs"] {
			t.Errorf("a click on docs: cursor %q, open %v", v.Cursor, v.Opened["docs"])
		}
		mouseAt(t, h, click(x, y))
		if !v.Opened["docs"] {
			t.Error("a click on the cursor's folder did not open it")
		}
		focus := h.m.ui.Focus
		mouseAt(t, h, click(0, 0))
		if !v.Open || h.m.ui.Focus != focus || v.Cursor != "docs" {
			t.Errorf("a click outside: Explorer %v, focus %q, cursor %q", v.Open, h.m.ui.Focus, v.Cursor)
		}
	})
}
