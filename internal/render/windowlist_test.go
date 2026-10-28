package render

import (
	"fmt"
	"strings"
	"testing"

	"github.com/Evengelius/agent-command-center/internal/state"
)

// TestWindowList is plan 61's design D5: the frame records what the
// window's rows showed, the viewer's lines or the quick open's files, from
// the row under the path or field row and between the side borders, and
// those rows show them; nothing while the file is read or the project
// listed. At maps a cell to its line or file, -1 off the list.
func TestWindowList(t *testing.T) {
	lines := make([]string, 100)
	for i := range lines {
		lines[i] = fmt.Sprintf("line %d", i+1)
	}
	files := make([]string, 60)
	for i := range files {
		files[i] = fmt.Sprintf("dir/file%02d.go", i)
	}
	s := state.New()
	s.Screen = state.ScreenDashboard
	viewer := func(f FileView) UI {
		f.Open, f.Path, f.Match = true, "f.txt", -1
		return UI{Width: 120, Height: 40, Focus: "B", File: f}
	}
	finder := func(v FinderView) UI {
		v.Open = true
		return UI{Width: 120, Height: 40, Focus: "B", Finder: v}
	}
	for _, c := range []struct {
		name        string
		ui          UI
		first, n    int
		top, bottom string // the texts of the first and the last row shown
	}{
		{"the viewer, the cursor on line 51", viewer(FileView{Lines: lines, Cursor: 50}), 23, 28, "line 24", "line 51"},
		{"a short file", viewer(FileView{Lines: lines[:3]}), 0, 3, "line 1", "line 3"},
		{"reading", viewer(FileView{State: "reading…"}), 0, 0, "", ""},
		{"the quick open, file 41 selected", finder(FinderView{Files: files, Sel: 40, Total: 60}), 13, 28, "dir/file13.go", "dir/file40.go"},
		{"listing", finder(FinderView{State: "listing…"}), 0, 0, "", ""},
	} {
		sc := Draw(s, c.ui)
		x, y, ww, _ := FileRect(c.ui.Width, c.ui.Height, false)
		want := ListShown{X: x + 1, Y: y + 2, W: ww - 2, First: c.first, N: c.n}
		if sc.WindowList != want {
			t.Errorf("%s: recorded %+v, want %+v", c.name, sc.WindowList, want)
			continue
		}
		rows := strings.Split(texts(sc.lines), "\n")
		if c.n > 0 && (!strings.Contains(rows[want.Y], c.top) || !strings.Contains(rows[want.Y+c.n-1], c.bottom) ||
			strings.Contains(rows[want.Y-1], c.top)) {
			t.Errorf("%s: rows %d to %d are %q and %q", c.name, want.Y, want.Y+c.n-1, rows[want.Y], rows[want.Y+c.n-1])
		}
	}

	l := ListShown{X: 11, Y: 5, W: 20, First: 40, N: 3}
	for _, c := range []struct{ x, y, want int }{
		{11, 5, 40}, {30, 7, 42}, {20, 6, 41},
		{10, 5, -1}, {31, 5, -1}, {11, 4, -1}, {11, 8, -1},
	} {
		if got := l.At(c.x, c.y); got != c.want {
			t.Errorf("At(%d, %d) = %d, want %d", c.x, c.y, got, c.want)
		}
	}
}
