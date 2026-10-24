package app

import (
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"testing/synctest"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/Evengelius/agent-command-center/internal/event"
	"github.com/Evengelius/agent-command-center/internal/input"
	"github.com/Evengelius/agent-command-center/internal/input/imagetest"
	"github.com/Evengelius/agent-command-center/internal/render"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// Plan 43 (brief amendment 44): the activity sidebar and the Explorer docked
// between it and zone B, through the application.

// itemCell is the first cell of panel p carrying item on the frame drawn
// now.
func itemCell(t *testing.T, h *harness, p byte, item int) (x, y int) {
	t.Helper()
	h.m.View()
	for y := range h.m.ui.Height {
		for x := range h.m.ui.Width {
			if q, it := h.m.screen.ItemAt(x, y); q == p && it == item {
				return x, y
			}
		}
	}
	t.Fatalf("no cell of panel %c, item %d:\n%s", p, item, h.frame())
	return 0, 0
}

// panelExtent is the first and last column of panel p's cells on the frame
// drawn now; -1 and -1 when it has none.
func panelExtent(h *harness, p byte) (x0, x1 int) {
	h.m.View()
	x0, x1 = -1, -1
	for y := range h.m.ui.Height {
		for x := range h.m.ui.Width {
			if q, _ := h.m.screen.ItemAt(x, y); q == p {
				if x0 < 0 || x < x0 {
					x0 = x
				}
				x1 = max(x1, x)
			}
		}
	}
	return x0, x1
}

// lastRow is the frame's footer.
func lastRow(h *harness) string {
	rows := strings.Split(h.frame(), "\n")
	return rows[len(rows)-1]
}

// TestExplorerToggle is criterion 43.6's showing and hiding (design D5):
// F3 and a click on the Explorer's tile show the panel, the focus on it and
// the tile selected, and hide it, the focus back to zone B and its width
// back; the panel's "⌄" hides it; F3 and the tile leave the same state.
func TestExplorerToggle(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, _ := explorerRig(t, &fakeGit{marks: input.GitMarks{Files: map[string]byte{"main.go": 'M'}}})
		h.send(tea.WindowSizeMsg{Width: 200, Height: 60})
		check := func(when string, open bool, focus string, bw int) {
			t.Helper()
			f := h.frame()
			selected := strings.Contains(f, "││▎ ≡   │") // the Explorer's tile in its selected look
			if h.m.ui.Explorer.Open != open || h.m.ui.Focus != focus || h.m.screen.B.W != bw || selected != open {
				t.Fatalf("%s: shown %v, focus %q, zone B %d columns, tile selected %v; want %v, %q, %d\n%s",
					when, h.m.ui.Explorer.Open, h.m.ui.Focus, h.m.screen.B.W, selected, open, focus, bw, f)
			}
		}
		// The Explorer's state as text: its rows, cursor, view, folders open and
		// marks.
		explorer := func() string {
			v := h.m.ui.Explorer
			return fmt.Sprintf("open %v %q rows %q cursor %q top %d opened %v marks %v note %q", v.Open, v.Name, treeNames(v), v.Cursor, v.Top, v.Opened, v.Git.Marks, v.Git.Note)
		}
		check("at first", false, "B", 136) // 200 - 56 - 8
		h.send(key(tea.KeyF3, 0))
		check("F3", true, "R", 106)  // the panel's 30 columns docked
		h.send(key(tea.KeyRight, 0)) // docs opened: the state to find again
		viaF3 := explorer()
		h.send(key(tea.KeyF3, 0))
		check("F3 again", false, "B", 136)

		x, y := itemCell(t, h, render.SidePanel, render.SideExplorer)
		mouseClick(t, h, x, y)
		check("a click on the tile", true, "R", 106)
		if got := explorer(); got != viaF3 {
			t.Errorf("the tile shows another state than F3:\n%s\n%s", got, viaF3)
		}
		mouseClick(t, h, x, y)
		check("a second click on the tile", false, "B", 136)

		h.send(key(tea.KeyF3, 0))
		hx, hy := itemCell(t, h, render.ExplorerPanel, render.ItemPanelHide)
		if l := strings.Split(h.frame(), "\n")[hy]; !strings.Contains(l, "EXPLORER") {
			t.Errorf("the ⌄ is not on the header's row: %q", l)
		}
		mouseClick(t, h, hx, hy)
		check("a click on the panel's ⌄", false, "B", 136)
	})
}

// TestSkillsTile is criterion 43.6's Skills (design D5): a click on its
// tile, the pointer on it already as a real one arrives (hovering it),
// changes nothing at all, the panel shown or hidden, a popup open or not:
// the same frame, focus, selection and draft, and no command.
func TestSkillsTile(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, s := selectorRig(t)
		h.send(tea.WindowSizeMsg{Width: 200, Height: 60})
		h.send(tea.PasteMsg{Content: "a draft kept"})
		h.send(key(tea.KeyLeft, tea.ModShift)) // a selection in the draft
		calls := func() string { a, b, c := s.calls(); return fmt.Sprint(a, b, c, s.modelChanges()) }
		for _, shown := range []bool{false, true} {
			if shown {
				h.send(key(tea.KeyF3, 0))
				h.send(key(tea.KeyEscape, 0)) // zone B's focus, the panel kept
			}
			for _, popup := range []bool{false, true} {
				if popup {
					h.send(key('p', tea.ModAlt))
				}
				name := fmt.Sprintf("panel shown %v, selector open %v", shown, popup)
				x, y := itemCell(t, h, render.SidePanel, render.SideSkills)
				mouseAt(t, h, moveTo(x, y))
				before, focus, ui, sent := h.m.View().Content, h.m.ui.Focus, h.m.ui, calls()
				from, to, sel := h.m.edit.Selection()
				for _, msg := range []tea.Msg{click(x, y), tea.MouseReleaseMsg{X: x, Y: y, Button: tea.MouseLeft}} {
					if _, cmd := h.m.Update(msg); cmd != nil {
						t.Errorf("%s: %T gave a command", name, msg)
					}
				}
				f2, t2, sel2 := h.m.edit.Selection()
				if h.m.View().Content != before || h.m.ui.Focus != focus || h.m.ui.Sel != ui.Sel || h.m.ui.Models.Open != ui.Models.Open ||
					h.m.ui.Explorer.Open != ui.Explorer.Open || h.m.ui.Plus != ui.Plus || calls() != sent || h.m.edit.Text() != "a draft kept" ||
					f2 != from || t2 != to || sel2 != sel {
					t.Errorf("%s: a click on Skills changed something", name)
				}
				if popup {
					h.send(key(tea.KeyEscape, 0))
				}
			}
		}
	})
}

// TestExplorerF3 is criterion 43.6's F3 (design D5): its file meaning kept
// in its three cases, the panel left as it was; from a full view naming no
// file, the inspection or the details view, F3 closes it and shows the
// panel; from the viewer or the quick open, it closes them and shows or
// hides the panel.
func TestExplorerF3(t *testing.T) {
	// The file's three cases: F3 opens main.go in the viewer, the panel
	// hidden; F3 in the viewer closes it and shows the panel.
	for _, c := range []struct {
		name, opened string
		emit         func(t *testing.T, r inputRig, root string)
		keys         []tea.KeyPressMsg
	}{
		{"an approval shown whole", "edited by Edit", func(t *testing.T, r inputRig, root string) {
			in := toolInput(t, map[string]any{"file_path": filepath.Join(root, "main.go"), "old_string": "x", "new_string": "// line 50"})
			r.emit(event.ApprovalRequest, event.Approval{ID: "ap1", Tool: "Edit", ToolID: "t1", Input: in})
		}, []tea.KeyPressMsg{key(tea.KeyF2, 0), key(tea.KeyEnter, 0)}},
		{"an entry shown whole", "read by Read", func(t *testing.T, r inputRig, root string) {
			r.emit(event.ToolStarted, event.Tool{ToolID: "t1", Name: "Read", Input: toolInput(t, map[string]any{"file_path": filepath.Join(root, "main.go")})})
		}, []tea.KeyPressMsg{key(tea.KeyUp, 0), key(tea.KeyEnter, 0)}},
		{"the entry cursor on a call", "read by Read", func(t *testing.T, r inputRig, root string) {
			r.emit(event.ToolStarted, event.Tool{ToolID: "t1", Name: "Read", Input: toolInput(t, map[string]any{"file_path": filepath.Join(root, "main.go")})})
		}, []tea.KeyPressMsg{key(tea.KeyUp, 0)}},
	} {
		t.Run(c.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				r, _, root := fileRig(t)
				h := r.h
				h.m.opt.Git = &fakeGit{}
				c.emit(t, r, root)
				h.settle()
				for _, k := range c.keys {
					h.send(k)
				}
				if h.m.ui.View == "" && h.m.ui.ViewEntry == 0 && h.m.ui.Entry == 0 {
					t.Fatal("nothing in front")
				}
				h.send(key(tea.KeyF3, 0))
				if f := h.m.ui.File; !f.Open || f.Rel != "main.go" || f.Opened != c.opened || h.m.ui.Explorer.Open {
					t.Fatalf("F3: viewer %v %q %q, panel %v", f.Open, f.Rel, f.Opened, h.m.ui.Explorer.Open)
				}
				h.send(key(tea.KeyF3, 0))
				if h.m.ui.File.Open || !h.m.ui.Explorer.Open || h.m.ui.Focus != "R" {
					t.Errorf("F3 in the viewer: viewer %v, panel %v, focus %q", h.m.ui.File.Open, h.m.ui.Explorer.Open, h.m.ui.Focus)
				}
			})
		})
	}
	t.Run("a full view naming no file, then the quick open", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			r, _, _ := fileRig(t)
			h := r.h
			h.m.opt.Git = &fakeGit{}
			r.emit(event.ToolStarted, event.Tool{ToolID: "t1", Name: "Bash", Input: toolInput(t, map[string]any{"command": "ls"})})
			h.settle()
			h.send(key(tea.KeyUp, 0))
			h.send(key(tea.KeyEnter, 0))
			if h.m.ui.ViewEntry == 0 {
				t.Fatal("Enter showed no entry whole")
			}
			h.send(key(tea.KeyF3, 0))
			if h.m.ui.ViewEntry != 0 || !h.m.ui.Explorer.Open || h.m.ui.Focus != "R" || h.m.ui.File.Open {
				t.Errorf("F3 on a Bash entry shown whole: view %d, panel %v, focus %q, viewer %v", h.m.ui.ViewEntry, h.m.ui.Explorer.Open, h.m.ui.Focus, h.m.ui.File.Open)
			}
			h.send(typed('/'))
			h.send(key(tea.KeyF3, 0))
			if h.m.ui.Finder.Open || h.m.ui.Explorer.Open || h.m.ui.Focus != "B" {
				t.Errorf("F3 in the quick open: quick open %v, panel %v, focus %q", h.m.ui.Finder.Open, h.m.ui.Explorer.Open, h.m.ui.Focus)
			}
		})
	})
	for _, c := range []struct {
		name string
		open func(*testing.T, *harness)
		gone func(*harness) bool
	}{
		{"the inspection", func(t *testing.T, h *harness) {
			h.send(key(tea.KeyF6, 0)) // zone C
			h.send(key(tea.KeyEnter, 0))
		}, func(h *harness) bool { return h.m.ui.Inspect == "" }},
		{"the details view", openDetailsView, func(h *harness) bool { return !h.m.ui.Details }},
	} {
		t.Run(c.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				h := mockDash(t, "claude-subagent-approval.jsonl")
				h.m.opt.Git = &fakeGit{}
				c.open(t, h)
				if c.gone(h) {
					t.Fatalf("%s not shown", c.name)
				}
				h.send(key(tea.KeyF3, 0))
				f := h.frame()
				if !c.gone(h) || !h.m.ui.Explorer.Open || h.m.ui.Focus != "R" || !strings.Contains(f, "EXPLORER") {
					t.Errorf("F3 from %s: closed %v, panel %v, focus %q\n%s", c.name, c.gone(h), h.m.ui.Explorer.Open, h.m.ui.Focus, f)
				}
			})
		})
	}
}

// TestExplorerDocked is criterion 43.8's new behaviour (design D7), the
// panel docked at 200x60: Esc on a top-level row gives zone B the focus,
// the panel kept; the wheel over zone B leaves the panel, over the panel
// moves its cursor; a click on the panel gives it the focus, one in zone B
// gives it back; the folders, the cursor and the marks survive hiding and
// showing.
func TestExplorerDocked(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, _ := explorerRig(t, &fakeGit{marks: input.GitMarks{Files: map[string]byte{"src/Main.java": 'M', "main.go": 'U'}}})
		h.send(tea.WindowSizeMsg{Width: 200, Height: 60})
		h.send(key(tea.KeyF3, 0))
		v := &h.m.ui.Explorer
		h.send(key(tea.KeyRight, 0)) // docs opened
		h.send(key(tea.KeyRight, 0)) // docs/readme.md
		h.send(key(tea.KeyEscape, 0))
		if !v.Open || v.Opened["docs"] || v.Cursor != "docs" || h.m.ui.Focus != "R" {
			t.Fatalf("Esc in docs: open %v, docs open %v, cursor %q, focus %q", v.Open, v.Opened["docs"], v.Cursor, h.m.ui.Focus)
		}
		h.send(key(tea.KeyEscape, 0))
		if !v.Open || h.m.ui.Focus != "B" {
			t.Fatalf("Esc on a top-level row: open %v, focus %q", v.Open, h.m.ui.Focus)
		}

		h.m.View()
		sc := h.m.screen
		bx, px, y := sc.B.X+5, sc.Panel.X+5, 10
		if p := sc.PanelAt(bx, y); p != 'B' {
			t.Fatalf("no zone B at %d,%d: %c", bx, y, p)
		}
		mouseAt(t, h, tea.MouseWheelMsg{X: bx, Y: y, Button: tea.MouseWheelDown})
		if v.Cursor != "docs" {
			t.Errorf("the wheel over zone B moved the panel's cursor to %q", v.Cursor)
		}
		mouseAt(t, h, tea.MouseWheelMsg{X: px, Y: y, Button: tea.MouseWheelDown})
		if v.Cursor != "data.bin" { // wheelRows entries down from docs: src, web, data.bin
			t.Errorf("the wheel over the panel: cursor %q", v.Cursor)
		}
		mouseClick(t, h, sc.Panel.X+sc.Panel.W-2, 2) // the panel's frame
		if h.m.ui.Focus != "R" || v.Cursor != "data.bin" {
			t.Errorf("a click on the panel's frame: focus %q, cursor %q", h.m.ui.Focus, v.Cursor)
		}
		mouseClick(t, h, bx, y)
		if h.m.ui.Focus != "B" || !v.Open {
			t.Errorf("a click in zone B: focus %q, panel %v", h.m.ui.Focus, v.Open)
		}

		// Hidden and shown again: the same folders, cursor and marks.
		x, yy := at(t, h, "› 🗀  src", 0)
		mouseClick(t, h, x, yy)
		mouseClick(t, h, x, yy) // src opened
		opened, cursor, marks := fmt.Sprint(v.Opened), v.Cursor, fmt.Sprint(v.Git.Marks)
		h.send(key(tea.KeyF3, 0))
		h.send(key(tea.KeyF3, 0))
		if !v.Open || fmt.Sprint(v.Opened) != opened || v.Cursor != cursor || fmt.Sprint(v.Git.Marks) != marks || !v.Opened["src"] {
			t.Errorf("shown again: opened %v, cursor %q, marks %v; want %s, %q, %s", v.Opened, v.Cursor, v.Git.Marks, opened, cursor, marks)
		}
		if f := h.frame(); !rowHas(f, "Main.java", "M") || !rowHas(f, "main.go", "U") {
			t.Errorf("the marks not drawn again:\n%s", f)
		}
	})
}

// TestExplorerContent is criterion 43.7 on a project read from the disk
// (design D6): the header, the rule, the root row with the project folder's
// own name, the tree under it one step deeper, folders with "›" and "⌄" and
// their icon, files with theirs.
func TestExplorerContent(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, root := explorerRig(t, &fakeGit{})
		h.send(tea.WindowSizeMsg{Width: 200, Height: 60})
		h.send(key(tea.KeyF3, 0))
		h.send(key(tea.KeyRight, 0)) // docs opened
		rows := strings.Split(h.frame(), "\n")
		panel := func(y int) string {
			sc := h.m.screen
			return strings.TrimRight(cellText(rows[y], sc.Panel.X, sc.Panel.X+sc.Panel.W), " ")
		}
		want := []string{ // the panel 30 columns at 200 (design D1)
			"╭" + strings.Repeat("─", 28) + "╮",
			"│ EXPLORER" + strings.Repeat(" ", 17) + "⌄ │",
			"│ " + strings.Repeat("─", 26) + " │",
			"│ ⌄ 🗀  " + filepath.Base(root),
			"│   ⌄ 🗀  docs",
			"│       🗎  readme.md",
			"│   › 🗀  src",
			"│   › 🗀  web",
			"│     🗎  data.bin",
			"│     🗎  main.go",
		}
		for i, w := range want {
			if got := panel(2 + i); !strings.HasPrefix(got, w) {
				t.Errorf("the panel's row %d %q, want %q…", i, got, w)
			}
		}
	})
}

// cellText is row's text between the cell columns from and to, a wide
// character counted as the cells it takes.
func cellText(row string, from, to int) string {
	var b strings.Builder
	x := 0
	for _, r := range row {
		w := lipgloss.Width(string(r))
		if x >= from && x+w <= to {
			b.WriteRune(r)
		}
		x += w
	}
	return b.String()
}

// TestExplorerNotWindow is criterion 43.9 (design D8): with the panel
// shown, F4 opens the modes selector, Alt+P the model selector, a click the
// "+" menu, from the panel's focus too, and a paste reaches the draft; the
// draft, its images and zone B's scroll are the same before showing it,
// while shown and after hiding it.
func TestExplorerNotWindow(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, s := selectorRig(t)
		s.auto = true
		pushEvents(h, "claude-code", evOf(event.SessionStarted, event.Started{SessionID: "s1", ApprovalMode: event.ModeAuto}))
		h.send(tea.WindowSizeMsg{Width: 200, Height: 60})
		files, err := imagetest.WriteAll(t.TempDir())
		if err != nil {
			t.Fatal(err)
		}
		h.m.opt.PickImages = (&picker{answers: [][]string{{files["png"]}}}).pick
		h.send(tea.PasteMsg{Content: "a draft kept"})
		upload(t, h)
		var long []string
		for i := range 200 {
			long = append(long, fmt.Sprintf("line %03d of a long answer", i))
		}
		pushEvents(h, "claude-code", evOf(event.MessageCompleted, event.Completed{MessageID: "m1", Text: strings.Join(long, "\n")}))
		h.m.View()
		h.send(key(tea.KeyPgUp, 0))
		h.send(key(tea.KeyPgUp, 0))
		kept := func() string {
			return fmt.Sprintf("draft %q, images %v, held %v at %d", h.m.edit.Text(), rcNames(h.m), h.m.ui.ConvHold, h.m.ui.ConvTop)
		}
		before := kept()
		if len(h.m.attach) != 1 || !h.m.ui.ConvHold {
			t.Fatalf("the rig: %s", before)
		}
		h.send(key(tea.KeyF3, 0))
		if !h.m.ui.Explorer.Open || render.WindowOpen(h.m.ui) || kept() != before {
			t.Fatalf("shown: panel %v, a window %v; %s, was %s", h.m.ui.Explorer.Open, render.WindowOpen(h.m.ui), kept(), before)
		}
		// From the panel's focus: F4 and Alt+P, each drawn.
		h.send(key(tea.KeyF4, 0))
		if h.m.View(); !h.m.ui.Modes.Open || !h.m.screen.Modes.Shown {
			t.Errorf("F4 with the panel focused: modes open %v, drawn %v", h.m.ui.Modes.Open, h.m.screen.Modes.Shown)
		}
		h.send(key(tea.KeyF4, 0))
		h.send(key('p', tea.ModAlt))
		if h.m.View(); !h.m.ui.Models.Open || !h.m.screen.Selector.Shown {
			t.Errorf("Alt+P with the panel focused: selector open %v, drawn %v", h.m.ui.Models.Open, h.m.screen.Selector.Shown)
		}
		h.send(key(tea.KeyEscape, 0))
		// The "+" menu by a click, then a paste into the draft.
		x, y := at(t, h, "+     ", 0)
		mouseClick(t, h, x+1, y)
		if mx, _ := panelExtent(h, 'U'); !h.m.ui.Plus || mx < 0 {
			t.Errorf("a click on +: menu %v, drawn %v", h.m.ui.Plus, mx >= 0)
		}
		mouseClick(t, h, x+1, y)
		if !h.m.ui.Explorer.Open || kept() != before || h.m.ui.Focus != "B" {
			t.Fatalf("after the popups: panel %v, focus %q; %s, was %s", h.m.ui.Explorer.Open, h.m.ui.Focus, kept(), before)
		}
		h.send(tea.PasteMsg{Content: " and pasted"})
		if got := h.m.edit.Text(); got != "a draft kept and pasted" {
			t.Errorf("a paste with the panel shown: draft %q", got)
		}
		before = kept()
		h.send(key(tea.KeyF3, 0))
		if h.m.ui.Explorer.Open || kept() != before {
			t.Errorf("hidden: panel %v; %s, was %s", h.m.ui.Explorer.Open, kept(), before)
		}
	})
}

// TestExplorerFocus is criterion 43.10 (design D7): F6 and Tab go through
// B, the panel while shown, C and A, Shift+F6 and Shift+Tab back; the
// footer shows the panel's keys while it has the focus; the footer and F1
// say "F3 Explorer".
func TestExplorerFocus(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, _ := explorerRig(t, &fakeGit{})
		h.send(tea.WindowSizeMsg{Width: 200, Height: 60})
		cycle := func(k tea.KeyPressMsg, want ...string) {
			t.Helper()
			var got []string
			for range want {
				h.send(k)
				got = append(got, h.m.ui.Focus)
			}
			if !reflect.DeepEqual(got, want) {
				t.Errorf("%s: %v, want %v", k.String(), got, want)
			}
		}
		cycle(key(tea.KeyF6, 0), "C", "A", "B") // hidden: no panel in the order
		if f := lastRow(h); !strings.Contains(f, "F3 Explorer") {
			t.Errorf("the dashboard's footer: %q", f)
		}
		h.send(key(tea.KeyF3, 0))
		if f := lastRow(h); h.m.ui.Focus != "R" || !strings.Contains(f, "→ open") || !strings.Contains(f, "Esc back") || !strings.Contains(f, "F3 Explorer") {
			t.Errorf("the panel's footer, focus %q: %q", h.m.ui.Focus, f)
		}
		cycle(key(tea.KeyF6, 0), "C", "A", "B", "R")
		cycle(key(tea.KeyF6, tea.ModShift), "B", "A", "C", "R")
		cycle(key(tea.KeyTab, 0), "C", "A")
		cycle(key(tea.KeyTab, tea.ModShift), "C", "R")
		h.send(key(tea.KeyF1, 0))
		seen := h.frame()
		for range 10 {
			h.send(key(tea.KeyPgDown, 0))
			seen += h.frame()
		}
		for _, want := range []string{"else the Explorer, shown or hidden", "THE EXPLORER (F3 shows or hides it, with the focus)", "hide the Explorer"} {
			if !strings.Contains(seen, want) {
				t.Errorf("F1 lacks %q", want)
			}
		}
	})
}

// TestPopupsInsideB is criterion 43.11 (design D9): the model selector, the
// modes selector and the "+" menu end at zone B's right edge or before it,
// the panel shown and hidden, from 100x40 to 208x59; one as wide as zone B
// or narrower starts at its left edge or after it, a wider one keeps its
// width and ends at zone B's right edge, over zone A, never off the screen
// (the model selector, 67 cells, at 100 columns and beside the panel
// docked at 136). At 120x40 the shown panel takes zone B's place, keeping
// the focus (zone B's would hide it, plan 43's compact rule), and no popup
// opens. At 100 and 112 columns zone B is narrower than the selector's
// place right of its pill: there the clamp to zone B's edge, not the
// screen's, keeps it off the sidebar.
func TestPopupsInsideB(t *testing.T) {
	for _, size := range [][2]int{{100, 40}, {105, 40}, {112, 40}, {120, 40}, {136, 40}, {156, 44}, {200, 60}, {208, 59}} {
		for _, shown := range []bool{false, true} {
			if shown && size[0] < 120 {
				continue // the panel in zone B's place, as at 120
			}
			t.Run(fmt.Sprintf("%dx%d shown %v", size[0], size[1], shown), func(t *testing.T) {
				synctest.Test(t, func(t *testing.T) {
					h, s := selectorRig(t)
					s.auto = true
					pushEvents(h, "claude-code", evOf(event.SessionStarted, event.Started{SessionID: "s1", ApprovalMode: event.ModeAuto}))
					h.send(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
					if shown {
						h.send(key(tea.KeyF3, 0))
						if size[0] > 120 {
							h.send(key(tea.KeyEscape, 0)) // zone B's focus, the panel kept docked
						}
					}
					h.m.View()
					right := h.m.screen.B.X + h.m.screen.B.W - 1
					inB := h.m.screen.B.W > 0
					if inB != !(shown && size[0] == 120) {
						t.Fatalf("zone B drawn %v", inB)
					}
					closePopup := func() {
						if inB { // nothing opens over the panel in zone B's place: no Esc to hide it
							h.send(key(tea.KeyEscape, 0))
						}
					}
					for _, c := range []struct {
						name  string
						panel byte
						open  func()
						close func()
					}{
						{"the model selector", render.SelectorPanel, func() { h.send(key('p', tea.ModAlt)) }, closePopup},
						{"the modes selector", render.ModesPanel, func() { h.send(key(tea.KeyF4, 0)) }, closePopup},
						{"the + menu", 'U', func() {
							if x, y, ok := plusCell(h); ok {
								mouseClick(t, h, x, y)
							}
						}, closePopup},
					} {
						c.open()
						x0, x1 := panelExtent(h, c.panel)
						wide := x1-x0+1 > h.m.screen.B.W
						switch {
						case !inB && x0 >= 0:
							t.Errorf("%s drawn with zone B in the panel's place", c.name)
						case inB && x0 < 0:
							t.Errorf("%s not drawn", c.name)
						case inB && x1 > right:
							t.Errorf("%s ends at column %d, past zone B's right edge %d", c.name, x1, right)
						case inB && !wide && x0 < h.m.screen.B.X:
							t.Errorf("%s, %d columns, starts at column %d, left of zone B's %d", c.name, x1-x0+1, x0, h.m.screen.B.X)
						case inB && wide && x1 != right:
							t.Errorf("%s, wider than zone B, ends at column %d, not at its right edge %d", c.name, x1, right)
						case c.panel == render.SelectorPanel && (size[0] == 100 || size[0] == 136 && shown) && !wide:
							t.Errorf("%s fits zone B: the case of one wider is no longer tested", c.name)
						}
						c.close()
						h.m.View() // the frame the program draws after each message
					}
				})
			})
		}
	}
}

// plusCell is the "+" band's cell, when the frame shows it.
func plusCell(h *harness) (x, y int, ok bool) {
	h.m.View()
	for y := range h.m.ui.Height {
		for x := range h.m.ui.Width {
			if p, _ := h.m.screen.ItemAt(x, y); p == '+' {
				return x, y, true
			}
		}
	}
	return 0, 0, false
}

// TestExplorerDraftKeys is plan 43's review on design D8: the docked panel
// is no window, so with the focus on it the draft beside it keeps the
// clipboard: the terminal's paste goes into it, Ctrl+C copies its selection
// instead of interrupting the running turn, Alt+V reads the clipboard's
// images for it. In zone B's place (120x40) the draft is not on screen: a
// paste goes nowhere and Ctrl+C interrupts, as from zones A and C.
func TestExplorerDraftKeys(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r, clip, _ := fileRig(t)
		h := r.h
		h.m.opt.Git = &fakeGit{}
		h.send(tea.WindowSizeMsg{Width: 200, Height: 60})
		h.send(tea.PasteMsg{Content: "hello world"})
		h.send(key(tea.KeyF3, 0))
		if h.m.ui.Focus != "R" || render.ExplorerCompact(h.m.State(), h.m.ui) {
			t.Fatalf("F3: focus %q, compact %v; want the panel docked with the focus", h.m.ui.Focus, render.ExplorerCompact(h.m.State(), h.m.ui))
		}
		h.send(tea.PasteMsg{Content: " again"})
		if got := h.m.edit.Text(); got != "hello world again" || h.m.ui.Focus != "R" {
			t.Errorf("a paste with the docked panel focused: draft %q, focus %q", got, h.m.ui.Focus)
		}
		h.send(key(tea.KeyEscape, 0)) // zone B's focus, the panel kept (design D7)
		for range 5 {
			h.send(key(tea.KeyLeft, tea.ModShift))
		}
		r.emit(event.TurnStarted, nil)
		h.settle()
		h.send(key(tea.KeyF6, 0)) // the panel's focus again
		if h.m.ui.Focus != "R" || h.m.edit.SelectedText() != "again" || h.m.State().Turn.Phase != state.Running {
			t.Fatalf("the rig: focus %q, selection %q, turn %s", h.m.ui.Focus, h.m.edit.SelectedText(), h.m.State().Turn.Phase)
		}
		h.send(key('c', tea.ModCtrl))
		if clip.last() != "again" || h.m.State().Turn.Phase != state.Running {
			t.Errorf("Ctrl+C with the docked panel focused: copied %q, turn %s; want the selection copied, the turn running", clip.last(), h.m.State().Turn.Phase)
		}
		h.send(key('v', tea.ModAlt))
		if h.m.ui.Toast != "no clipboard in this run" { // this rig's clipboard gives text only
			t.Errorf("Alt+V with the docked panel focused: toast %q, want the images asked for", h.m.ui.Toast)
		}

		h.send(tea.WindowSizeMsg{Width: 120, Height: 40})
		if !render.ExplorerCompact(h.m.State(), h.m.ui) || h.m.ui.Focus != "R" {
			t.Fatalf("120x40: compact %v, focus %q", render.ExplorerCompact(h.m.State(), h.m.ui), h.m.ui.Focus)
		}
		h.send(tea.PasteMsg{Content: " lost"})
		h.send(key('c', tea.ModCtrl))
		if got := h.m.edit.Text(); got != "hello world again" || h.m.State().Turn.Phase != state.Interrupting {
			t.Errorf("the panel in zone B's place: draft %q, turn %s; want the paste dropped and Ctrl+C interrupting", got, h.m.State().Turn.Phase)
		}
	})
}

// TestExplorerF2 is plan 43's review on designs D5 and D8: F2 from the
// docked panel, or from the viewer opened from it, leaves the panel shown,
// with no approval waiting saying so, else focusing the card; while the
// panel has the focus the footer names the waiting approval, in zone B's
// place too, where the card is hidden, and F2 there hides the panel for
// zone B and its card (the compact rule).
func TestExplorerF2(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r, _, _ := fileRig(t)
		h := r.h
		h.m.opt.Git = &fakeGit{}
		h.send(tea.WindowSizeMsg{Width: 200, Height: 60})
		h.send(key(tea.KeyF3, 0))
		h.send(key(tea.KeyF2, 0))
		if !h.m.ui.Explorer.Open || h.m.ui.Focus != "R" || h.m.ui.Toast != "no approval waiting" {
			t.Errorf("F2 from the panel, no approval: shown %v, focus %q, toast %q", h.m.ui.Explorer.Open, h.m.ui.Focus, h.m.ui.Toast)
		}
		h.send(key(tea.KeyEnd, 0))
		h.send(key(tea.KeyEnter, 0)) // the last file, in the viewer
		if !h.m.ui.File.Open {
			t.Fatal("the viewer did not open")
		}
		h.send(key(tea.KeyF2, 0))
		if h.m.ui.File.Open || !h.m.ui.Explorer.Open {
			t.Errorf("F2 from the viewer: viewer %v, panel %v; want the viewer closed, the panel kept", h.m.ui.File.Open, h.m.ui.Explorer.Open)
		}
		h.send(toastDoneMsg{h.m.toastGen}) // the toast over: the footer's own text

		in := []byte(`{"command":"echo hi"}`)
		r.emit(event.TurnStarted, nil)
		r.emit(event.ApprovalRequest, event.Approval{ID: "r1", Tool: "Bash", ToolID: "t1", Input: in})
		h.settle()
		for _, size := range []tea.WindowSizeMsg{{Width: 200, Height: 60}, {Width: 120, Height: 40}} {
			h.send(size)
			h.m.ui.Card = ""
			h.send(key(tea.KeyF3, 0))
			if !h.m.ui.Explorer.Open {
				h.send(key(tea.KeyF3, 0))
			}
			compact := render.ExplorerCompact(h.m.State(), h.m.ui)
			if f := lastRow(h); h.m.ui.Focus != "R" || !strings.Contains(f, "◆ approval waiting · F2 to answer") {
				t.Errorf("%dx%d, the panel focused: focus %q, footer %q", size.Width, size.Height, h.m.ui.Focus, f)
			}
			h.send(key(tea.KeyF2, 0))
			if h.m.ui.Card != "r1" || h.m.ui.Focus != "B" || h.m.ui.Explorer.Open == compact {
				t.Errorf("%dx%d, F2: card %q, focus %q, panel %v; want the card focused, the panel hidden only in zone B's place (%v)",
					size.Width, size.Height, h.m.ui.Card, h.m.ui.Focus, h.m.ui.Explorer.Open, compact)
			}
			if !strings.Contains(h.frame(), "echo hi") {
				t.Errorf("%dx%d: the card not on screen after F2:\n%s", size.Width, size.Height, h.frame())
			}
		}
	})
}

// TestExplorerHiddenB is plan 43's review on design D1's compact layout:
// where the panel takes zone B's place, zone B's editor status row and
// cards are hidden, so the footer shows what they would, whatever the
// focus: the quit question first, then a waiting approval.
func TestExplorerHiddenB(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r, _, _ := fileRig(t)
		h := r.h
		h.m.opt.Git = &fakeGit{}
		h.send(tea.WindowSizeMsg{Width: 120, Height: 40})
		r.emit(event.TurnStarted, nil)
		h.settle()
		h.send(key(tea.KeyF3, 0))
		if !render.ExplorerCompact(h.m.State(), h.m.ui) || h.m.ui.Focus != "R" {
			t.Fatalf("120x40, F3: compact %v, focus %q", render.ExplorerCompact(h.m.State(), h.m.ui), h.m.ui.Focus)
		}
		h.send(key('q', tea.ModCtrl))
		if f := lastRow(h); !h.m.State().QuitAsked || !strings.Contains(f, "quit? ") || !strings.Contains(f, "Ctrl+Q again to quit · Esc to stay") {
			t.Errorf("Ctrl+Q, a turn running: asked %v, footer %q", h.m.State().QuitAsked, f)
		}
		h.send(key(tea.KeyEscape, 0))
		if f := lastRow(h); h.m.State().QuitAsked || strings.Contains(f, "quit?") || !h.m.ui.Explorer.Open || h.m.ui.Focus != "R" {
			t.Errorf("Esc to stay: asked %v, panel %v, focus %q, footer %q", h.m.State().QuitAsked, h.m.ui.Explorer.Open, h.m.ui.Focus, f)
		}
		r.emit(event.ApprovalRequest, event.Approval{ID: "r1", Tool: "Bash", ToolID: "t1", Input: []byte(`{"command":"echo hi"}`)})
		h.settle()
		for _, focus := range []string{"R", "C", "A"} {
			for h.m.ui.Focus != focus {
				h.send(key(tea.KeyF6, 0))
			}
			if f := lastRow(h); !render.ExplorerCompact(h.m.State(), h.m.ui) || !strings.Contains(f, "◆ approval waiting · F2 to answer") {
				t.Errorf("focus %s: compact %v, footer %q", focus, render.ExplorerCompact(h.m.State(), h.m.ui), f)
			}
		}
	})
}

// TestExplorerZoomF3 is plan 43's review on design D5: a zoomed zone
// covers the dashboard, the panel and the sidebar with it, as the full
// views do: F3 there ends the zoom and shows the panel with the focus,
// whether it was shown or hidden before.
func TestExplorerZoomF3(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, _ := explorerRig(t, &fakeGit{})
		h.send(tea.WindowSizeMsg{Width: 200, Height: 40})
		for _, shown := range []bool{false, true} {
			if shown {
				h.send(key(tea.KeyF3, 0))
			}
			for h.m.ui.Focus != "C" {
				h.send(key(tea.KeyF6, 0))
			}
			h.send(key('z', 0))
			if !h.m.ui.Zoom || h.m.ui.Explorer.Open != shown {
				t.Fatalf("shown before %v: zoom %v, panel %v", shown, h.m.ui.Zoom, h.m.ui.Explorer.Open)
			}
			h.send(key(tea.KeyF3, 0))
			h.m.View()
			if h.m.ui.Zoom || !h.m.ui.Explorer.Open || h.m.ui.Focus != "R" || h.m.screen.Panel.W == 0 {
				t.Errorf("shown before %v, F3 in the zoom: zoom %v, panel %v, focus %q, drawn %d columns", shown, h.m.ui.Zoom, h.m.ui.Explorer.Open, h.m.ui.Focus, h.m.screen.Panel.W)
			}
			h.send(key(tea.KeyF3, 0))
			if h.m.ui.Explorer.Open {
				t.Fatalf("F3 did not hide the panel again")
			}
		}
	})
}

// TestSideHoverRelease is plan 43's review on design D4: a press on a tile
// starts no drag, so the button held off it unhovers it, and released
// elsewhere leaves no tile hovered; released on a tile, that tile is
// hovered.
func TestSideHoverRelease(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, _ := explorerRig(t, &fakeGit{})
		h.send(tea.WindowSizeMsg{Width: 200, Height: 60})
		ex, ey := itemCell(t, h, render.SidePanel, render.SideExplorer)
		mouseAt(t, h, moveTo(ex+1, ey+1))
		mouseAt(t, h, click(ex+1, ey+1))
		if !h.m.ui.Explorer.Open || h.m.ui.SideHover != render.SideExplorer {
			t.Fatalf("a press on the tile: panel %v, hover %d", h.m.ui.Explorer.Open, h.m.ui.SideHover)
		}
		mouseAt(t, h, tea.MouseMotionMsg{X: 60, Y: 20, Button: tea.MouseLeft})
		if h.m.ui.SideHover != 0 {
			t.Errorf("held off the tile: hover %d, want none", h.m.ui.SideHover)
		}
		mouseAt(t, h, tea.MouseReleaseMsg{X: 60, Y: 20, Button: tea.MouseLeft})
		if h.m.ui.SideHover != 0 {
			t.Errorf("held off the tile and released in zone B: hover %d, want none", h.m.ui.SideHover)
		}
		sx, sy := itemCell(t, h, render.SidePanel, render.SideSkills)
		mouseAt(t, h, click(60, 20))
		mouseAt(t, h, tea.MouseReleaseMsg{X: sx, Y: sy, Button: tea.MouseLeft})
		if h.m.ui.SideHover != render.SideSkills {
			t.Errorf("released on Skills' tile: hover %d, want Skills'", h.m.ui.SideHover)
		}
	})
}
