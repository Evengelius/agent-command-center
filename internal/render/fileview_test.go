package render

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/Evengelius/agent-command-center/internal/state"
)

// numberedLines is n lines "// line 1" to "// line n".
func numberedLines(n int) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = fmt.Sprintf("// line %d", i+1)
	}
	return out
}

// TestFileRect is plan 17, criterion 3.1: about 80% of the screen, centred,
// or all of it full screen; the dashboard's rows and columns outside the
// window are drawn as without it.
func TestFileRect(t *testing.T) {
	for _, c := range []struct{ w, h, x, y, ww, hh int }{
		{80, 24, 2, 1, 76, 22},
		{120, 40, 12, 4, 96, 32},
		{200, 60, 20, 6, 160, 48},
	} {
		if x, y, ww, hh := FileRect(c.w, c.h, false); x != c.x || y != c.y || ww != c.ww || hh != c.hh {
			t.Errorf("%dx%d: window %dx%d at %d,%d, want %dx%d at %d,%d", c.w, c.h, ww, hh, x, y, c.ww, c.hh, c.x, c.y)
		}
		if x, y, ww, hh := FileRect(c.w, c.h, true); x != 0 || y != 0 || ww != c.w || hh != c.h {
			t.Errorf("%dx%d full screen: %dx%d at %d,%d", c.w, c.h, ww, hh, x, y)
		}

		s := state.New()
		s.Screen = state.ScreenDashboard
		ui := UI{Width: c.w, Height: c.h, Focus: "B"}
		base := strings.Split(texts(Draw(s, ui).lines), "\n")
		ui.File = FileView{Open: true, Path: "main.go", Rel: "main.go", Opened: "read by Read", Lines: numberedLines(100), Match: -1}
		sc := Draw(s, ui)
		got := strings.Split(texts(sc.lines), "\n")
		for i := range c.h {
			in := i >= c.y && i < c.y+c.hh
			if !in && got[i] != base[i] {
				t.Errorf("%dx%d row %d changed outside the window:\n%q\n%q", c.w, c.h, i, got[i], base[i])
			}
			if in {
				l, b := sc.lines[i], Draw(s, UI{Width: c.w, Height: c.h, Focus: "B"}).lines[i]
				if texts([]line{cols(l, 0, c.x)}) != texts([]line{cols(b, 0, c.x)}) || texts([]line{cols(l, c.x+c.ww, c.w)}) != texts([]line{cols(b, c.x+c.ww, c.w)}) {
					t.Errorf("%dx%d row %d: the dashboard beside the window changed", c.w, c.h, i)
				}
				if w := textWidth(texts([]line{l})[:len(texts([]line{l}))-1]); w != c.w {
					t.Errorf("%dx%d row %d is %d cells", c.w, c.h, i, w)
				}
			}
		}
		if top := got[c.y]; !strings.Contains(top, "╭") || !strings.Contains(top, "main.go") || !strings.Contains(top, "100 lines") {
			t.Errorf("%dx%d: the top edge %q", c.w, c.h, top)
		}
		if row := got[c.y+2]; !strings.Contains(row, "  1  // line 1") {
			t.Errorf("%dx%d: the first code row %q", c.w, c.h, row)
		}
		ui.File.Full = true
		full := strings.Split(texts(Draw(s, ui).lines), "\n")
		if !strings.HasPrefix(full[0], "╭") || !strings.HasPrefix(full[c.h-1], "╰") {
			t.Errorf("%dx%d full screen: rows %q and %q", c.w, c.h, full[0], full[c.h-1])
		}
	}
}

// TestFileStates is plan 17, criterion 3.5, as drawn: each state and refusal
// shows in the window.
func TestFileStates(t *testing.T) {
	s := state.New()
	s.Screen = state.ScreenDashboard
	for _, f := range []FileView{
		{State: "reading…"},
		{State: "outside the project: C:\\elsewhere\\x.go", Failed: true},
		{State: "a credential file: not opened", Failed: true},
		{State: "binary file, not shown · 12 bytes"},
		{State: "too large to show: 40.0 MiB (32.0 MiB at most)", Failed: true},
		{State: "not found: missing.go", Failed: true},
		{Lines: numberedLines(3), Colouring: true},
		{Lines: numberedLines(3), Windowed: true},
		{Lines: []string{}},
	} {
		f.Open, f.Path, f.Match = true, "x.go", -1
		text := texts(Draw(s, UI{Width: 120, Height: 40, Focus: "B", File: f}).lines)
		want := f.State
		switch {
		case f.Colouring:
			want = "colouring…"
		case f.Windowed:
			want = "coloured around the view"
		case want == "":
			want = "empty file"
		}
		if !strings.Contains(text, want) {
			t.Errorf("%q not drawn:\n%s", want, text)
		}
	}
}

// TestToolFile is plan 17, criterion 3.2, for the targets: each tool's file,
// cursor line and marked lines.
func TestToolFile(t *testing.T) {
	js := func(v any) json.RawMessage { b, _ := json.Marshal(v); return b }
	lines := numberedLines(100)
	for _, c := range []struct {
		name   string
		tool   string
		input  json.RawMessage
		opened string
		cursor int
		mark   [2]int
	}{
		{"Read at a number", "Read", js(map[string]any{"file_path": "a.go", "offset": 40}), "read by Read", 39, [2]int{}},
		{"Read at a string", "Read", js(map[string]any{"file_path": "a.go", "offset": "12"}), "read by Read", 11, [2]int{}},
		{"Read whole", "Read", js(map[string]any{"file_path": "a.go"}), "read by Read", 0, [2]int{}},
		{"Edit", "Edit", js(map[string]any{"file_path": "a.go", "old_string": "x", "new_string": "// line 50\n// line 51\n"}), "edited by Edit", 49, [2]int{49, 51}},
		{"Edit not applied yet", "Edit", js(map[string]any{"file_path": "a.go", "old_string": "// line 7", "new_string": "seven"}), "edited by Edit", 6, [2]int{6, 7}},
		{"MultiEdit", "MultiEdit", js(map[string]any{"file_path": "a.go", "edits": []map[string]string{{"old_string": "q", "new_string": "absent"}, {"old_string": "w", "new_string": "// line 90"}}}), "edited by MultiEdit", 89, [2]int{89, 90}},
		{"Write", "Write", js(map[string]any{"file_path": "a.go", "content": "x"}), "written by Write", 0, [2]int{}},
		{"fileChange", "fileChange", js(map[string]any{"changes": []map[string]string{{"path": "a.go", "diff": "@@ -10,2 +10,3 @@\n // line 10\n+// line 11\n+// line 12\n // line 13\n"}}}), "changed by fileChange", 10, [2]int{10, 12}},
		{"fileChange added file", "fileChange", js(map[string]any{"changes": []map[string]string{{"path": "a.go", "diff": "one\ntwo\n"}}}), "changed by fileChange", 0, [2]int{0, 2}},
		{"another tool", "NotebookEdit", js(map[string]any{"notebook_path": "a.go"}), "named by NotebookEdit", 0, [2]int{}},
	} {
		ft, ok := ToolFile(c.tool, c.input)
		if !ok || ft.Path != "a.go" || ft.Opened != c.opened {
			t.Errorf("%s: %+v %v", c.name, ft, ok)
			continue
		}
		if cur, mark := ft.Resolve(lines); cur != c.cursor || mark != c.mark {
			t.Errorf("%s: cursor %d mark %v, want %d %v", c.name, cur, mark, c.cursor, c.mark)
		}
	}
	for _, c := range []struct {
		tool  string
		input json.RawMessage
	}{
		{"Bash", js(map[string]any{"command": "ls"})},
		{"fileChange", js(map[string]any{"changes": []any{}})},
		{"Read", json.RawMessage(`not json`)},
	} {
		if ft, ok := ToolFile(c.tool, c.input); ok {
			t.Errorf("%s %s names %+v", c.tool, c.input, ft)
		}
	}
}

// TestFileFooters is plan 17, criterion 3.9, for the windows: every footer
// fits its window whole, its keys ending with "Esc back · F3 close" ("Esc cancel" in a field); the
// Explorer's too (amendment 12).
func TestFileFooters(t *testing.T) {
	s := state.New()
	for _, w := range []int{80, 99, 100, 120, 200} {
		_, _, ww, _ := FileRect(w, 24, false)
		inner := ww - 4
		for _, f := range []FileView{
			{Lines: numberedLines(100_000), Cursor: 99_999},
			{Lines: numberedLines(10), Field: '/', Typed: "a long search text"},
			{Lines: numberedLines(10), Field: 'g', Typed: "12345"},
			{Lines: numberedLines(10), Query: "x", Matches: make([]FileMatch, MaxMatches), Match: 9_999, Capped: true},
			{Lines: numberedLines(10), Query: "x"},
			{Lines: numberedLines(10), Searching: true},
		} {
			l := fileFooter(s, UI{Width: w, Height: 24, File: f}, inner)
			text := texts([]line{l})
			end := "Esc back · F3 close"
			if f.Field != 0 {
				end = "Esc cancel" // a field open: Esc cancels it first
			}
			if n := l.width(); n != inner || strings.Contains(strings.ReplaceAll(text, "searching…", ""), "…") || !strings.Contains(text, end) {
				t.Errorf("file footer at %d columns: %d cells of %d: %q", w, n, inner, text)
			}
		}
		fw := finderWindow(UI{Width: w, Height: 24, Finder: FinderView{Open: true, Files: []string{"a.go"}, Total: 100_000, Truncated: true}}, ww, 22)
		if text := texts(fw[len(fw)-2 : len(fw)-1]); strings.Contains(text, "…") || !strings.Contains(text, "Esc back · F3 close") {
			t.Errorf("quick open footer at %d columns: %q", w, text)
		}
		deep := strings.Repeat("folder/", 30) + "file.go"
		ev := ExplorerView{Open: true, Name: "a-long-project-name", Dirs: map[string]*ExplorerDir{"": {Entries: []ExplorerEntry{{Name: deep}}}}, Cursor: deep}
		ew := explorerWindow(s, UI{Width: w, Height: 24, Explorer: ev}, ww, 22)
		if text := texts(ew[len(ew)-2 : len(ew)-1]); strings.Contains(text, "…") || !strings.Contains(text, "Esc back · F3 close") || !strings.Contains(text, "1 of 1") {
			t.Errorf("Explorer footer at %d columns: %q", w, text)
		}
	}
	found := map[string]bool{}
	for _, g := range KeyTable {
		for _, k := range g.keys {
			if k[0] == "F3" {
				found[g.title] = true
			}
		}
	}
	if len(found) < 3 {
		t.Errorf("F3 in the key overlay's groups %v, want ANYWHERE, B's entries and the file viewer", found)
	}
}

// TestDashboardF3 is plan 17's F3 in the dashboard footer: "F3 Explorer" at
// every layout; "Ctrl+J newline", which B's input shows too, left out only
// when the whole line does not fit, so that Ctrl+Q quit shows from 109
// columns in the standard layout and from 174 in the wide one.
func TestDashboardF3(t *testing.T) {
	for _, c := range []struct {
		w           int
		ctrlJ, quit bool
	}{
		{80, true, false}, {99, true, false}, {100, false, false}, {108, false, false}, {109, false, true},
		{120, false, true}, {125, false, true}, {126, true, true}, {159, true, true},
		{160, false, false}, {173, false, false}, {174, false, true}, {190, false, true}, {191, true, true}, {200, true, true},
	} {
		ls := dashboard(state.New(), UI{Width: c.w, Height: 40}, new(Screen))
		foot := texts(ls[len(ls)-1:])
		if c.w < 100 { // the compact layout's own line, whole at 80 columns
			c.quit = true
			if !strings.Contains(foot, "F2 approve · F3 Explorer · F1 keys") {
				t.Errorf("%d columns: %q", c.w, foot)
			}
			continue
		}
		if !strings.Contains(foot, "F2 approvals · F3 Explorer ·") ||
			strings.Contains(foot, "Ctrl+J newline") != c.ctrlJ || strings.Contains(foot, "Ctrl+Q quit") != c.quit {
			t.Errorf("%d columns: %q", c.w, foot)
		}
	}
	// On an entry that names a file, the entry footer: "F3 file".
	s := entryState()
	in, _ := json.Marshal(map[string]string{"file_path": "main.go"})
	s.Tools["t2"] = &state.ToolCall{ID: "t2", Name: "Read", Input: in, Status: state.ToolCompleted}
	s.Conversation = append(s.Conversation, state.Entry{Seq: uint64(len(s.Conversation) + 1), Role: "tool", MessageID: "t2", Text: "Read"})
	for _, w := range []int{80, 120, 200} {
		for n, want := range map[int]string{len(s.Conversation): "F3 file", 4: "Esc back to the input · F1 keys"} {
			ls := dashboard(s, UI{Width: w, Height: 40, Focus: "B", Entry: n}, new(Screen))
			if foot := texts(ls[len(ls)-1:]); !strings.Contains(foot, want) || strings.Contains(foot, "F3 Explorer") {
				t.Errorf("%d columns, entry %d: %q", w, n, foot)
			}
		}
	}
}
