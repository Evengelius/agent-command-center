package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/synctest"

	tea "charm.land/bubbletea/v2"

	"github.com/Evengelius/agent-command-center/internal/render"
)

// TestMarkdownFileKey is plan 71, design D5, criterion 71.4 in the
// application: a Markdown file opens as its preview; m shows its source and
// m again its preview, the shift back to the lines' start each time; a match
// is brought into view by the column the view draws it at; m in any other
// file does nothing and types nothing.
func TestMarkdownFileKey(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r, _, root := fileRig(t)
		h := r.h
		long := strings.Repeat("**bold** ", 30) + "needle"
		if err := os.WriteFile(filepath.Join(root, "docs", "long.md"), []byte("# Long\n"+long+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		openPath(h, filepath.Join(root, "docs", "readme.md"))
		f := &h.m.ui.File
		frame := plainText(h.m.View().Content)
		if !f.Open || f.Source || !strings.Contains(frame, "preview · ") || strings.Contains(frame, "# Readme") || !strings.Contains(frame, "Readme") || !strings.Contains(frame, "m source") {
			t.Fatalf("readme.md not opened as its preview:\n%s", frame)
		}
		f.Left = 8
		h.send(typed('m'))
		frame = plainText(h.m.View().Content)
		if !f.Source || f.Left != 0 || !strings.Contains(frame, "source · ") || !strings.Contains(frame, "# Readme") || !strings.Contains(frame, "m preview") {
			t.Errorf("m: source %v, left %d:\n%s", f.Source, f.Left, frame)
		}
		h.send(typed('m'))
		if f.Source || !strings.Contains(plainText(h.m.View().Content), "preview · ") {
			t.Errorf("m again: source %v", f.Source)
		}

		// The search's match in view, by the cells each view draws before it.
		openPath(h, filepath.Join(root, "docs", "long.md"))
		cw := render.FileCodeWidth(h.m.ui)
		for _, src := range []bool{false, true} {
			if !src {
				h.send(typed('/'))
				for _, k := range typedText("needle") {
					h.send(k)
				}
				h.send(key(tea.KeyEnter, 0))
			} else {
				h.send(typed('m')) // the source, from its start: n finds the match again
				if f.Left != 0 {
					t.Errorf("m: left %d", f.Left)
				}
				h.send(typed('n'))
			}
			col := 30 * len("bold ")
			if src {
				col = 30 * len("**bold** ")
			}
			if len(f.Matches) != 1 || f.Cursor != 1 || f.Left != max(col-cw/3, 0) {
				t.Errorf("source %v: %d matches, cursor %d, left %d; want left %d", src, len(f.Matches), f.Cursor, f.Left, max(col-cw/3, 0))
			}
			if !strings.Contains(plainText(h.m.View().Content), "needle") {
				t.Errorf("source %v: the match not in view", src)
			}
		}
		// Go to a line, in the preview as in the source.
		h.send(typed('m'))
		for _, k := range []tea.KeyPressMsg{typed('g'), typed('1'), key(tea.KeyEnter, 0)} {
			h.send(k)
		}
		if f.Source || f.Cursor != 0 || f.Field != 0 {
			t.Errorf("g 1 in the preview: source %v, cursor %d, field %q", f.Source, f.Cursor, f.Field)
		}

		// Any other file: m does nothing, and types nothing in the draft.
		openPath(h, filepath.Join(root, "main.go"))
		h.send(typed('m'))
		if f.Source || f.Rel != "main.go" || h.m.edit.Text() != "" {
			t.Errorf("m in main.go: source %v, %q, draft %q", f.Source, f.Rel, h.m.edit.Text())
		}
	})
}
