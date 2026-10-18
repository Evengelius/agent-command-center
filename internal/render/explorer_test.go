package render

import (
	"reflect"
	"strings"
	"testing"

	"github.com/Evengelius/agent-command-center/internal/state"
)

// treeText is the tree's rows, each as its indent and name.
func treeText(rows []ExplorerRow) []string {
	var out []string
	for _, r := range rows {
		out = append(out, strings.Repeat("  ", r.Depth)+r.Name)
	}
	return out
}

// TestExplorerRows is plan 17, criterion 5.3: an open folder's entries
// under it; "reading…" under a folder not read, its reason under one that
// cannot be, "empty folder", "first N entries" after a cut listing; a
// closed folder's entries hidden.
func TestExplorerRows(t *testing.T) {
	v := ExplorerView{
		Dirs: map[string]*ExplorerDir{
			"":  {Entries: []ExplorerEntry{{"a", true, false}, {"b", true, false}, {"c", true, false}, {"d", true, false}, {"e", true, false}, {"f.go", false, false}}, Truncated: true},
			"a": {Entries: []ExplorerEntry{{"x.go", false, false}}},
			"c": {State: "Access is denied."},
			"d": {},
			"e": {Entries: []ExplorerEntry{{"hidden.go", false, false}}},
		},
		Opened: map[string]bool{"a": true, "b": true, "c": true, "d": true},
	}
	rows := ExplorerRows(v)
	want := []string{"a", "  x.go", "b", "  reading…", "c", "  Access is denied.", "d", "  empty folder", "e", "f.go", "first 6 entries"}
	if got := treeText(rows); !reflect.DeepEqual(got, want) {
		t.Fatalf("rows %q, want %q", got, want)
	}
	if r := rows[5]; !r.Failed || r.Path != "" {
		t.Errorf("the reason's row: %+v", r)
	}
	if r := rows[1]; r.Path != "a/x.go" || r.Depth != 1 || r.Dir {
		t.Errorf("a/x.go's row: %+v", r)
	}
	if !rows[0].Open || rows[8].Open {
		t.Error("open marks wrong")
	}
	if got := treeText(ExplorerRows(ExplorerView{})); !reflect.DeepEqual(got, []string{"reading…"}) {
		t.Errorf("nothing read: %q", got)
	}
}

// TestExplorerCursor is plan 17, criterion 5.4: the cursor on its row,
// else the nearest folder above it shown, else the first entry.
func TestExplorerCursor(t *testing.T) {
	v := ExplorerView{
		Dirs:   map[string]*ExplorerDir{"": {Entries: []ExplorerEntry{{"a", true, false}, {"b.go", false, false}}}, "a": {Entries: []ExplorerEntry{{"c", true, false}}}},
		Opened: map[string]bool{"a": true},
	}
	rows := ExplorerRows(v)
	for _, c := range []struct {
		path string
		want int
	}{{"b.go", 2}, {"a/c", 1}, {"a/c/deep.go", 1}, {"gone.go", 0}, {"", 0}} {
		if got := ExplorerCursor(rows, c.path); got != c.want {
			t.Errorf("%q: row %d, want %d", c.path, got, c.want)
		}
	}
	if got := ExplorerCursor([]ExplorerRow{{Name: "reading…"}}, "x"); got != -1 {
		t.Errorf("no entry: %d", got)
	}
	for p, want := range map[string]string{"a/b/c.go": "a/b", "a": "", "": ""} {
		if got := ParentPath(p); got != want {
			t.Errorf("ParentPath(%q) = %q, want %q", p, got, want)
		}
	}
}

// TestExplorerMarks is plan 17, criterion 5.6: each path's own marker, U
// under an untracked folder, ● on each folder above a change, nothing
// elsewhere; an ignored path, or one under an ignored folder, dimmed with
// no marker, on the cursor row too.
func TestExplorerMarks(t *testing.T) {
	g := NewGitView(map[string]byte{"src/app/main.go": 'M', "new/": 'U', "top.go": 'A', "x/y.go": '!'},
		map[string]bool{"build/": true, "secret.log": true})
	if !reflect.DeepEqual(g.Dirty, map[string]bool{"src": true, "src/app": true, "x": true}) {
		t.Errorf("dirty folders %v", g.Dirty)
	}
	for _, c := range []struct {
		path string
		dir  bool
		want string
	}{
		{"src/app/main.go", false, "M"}, {"src", true, "●"}, {"src/app", true, "●"},
		{"new", true, "U"}, {"new/deep/a.go", false, "U"}, {"top.go", false, "A"},
		{"x/y.go", false, "!"}, {"clean.go", false, ""}, {"docs", true, ""},
	} {
		if got := explorerMark(g, c.path, c.dir); got != c.want {
			t.Errorf("%q: mark %q, want %q", c.path, got, c.want)
		}
	}
	for p, want := range map[string]bool{"build": true, "build/out/a.o": true, "secret.log": true, "src": false, "builder.go": false} {
		if got := explorerIgnored(g, p); got != want {
			t.Errorf("ignored(%q) = %v, want %v", p, got, want)
		}
	}
	for _, c := range []struct {
		mark string
		cur  bool
		st   any
	}{{"M", false, stProgress}, {"●", false, stProgress}, {"U", false, stDone}, {"A", false, stDone}, {"R", false, stDone},
		{"!", false, stFailed}, {"D", false, stFailed}, {"M", true, stExpRowProgress}, {"U", true, stExpRowDone}, {"!", true, stExpRowFailed}} {
		if got := markStyle(c.mark, c.cur); got != c.st {
			t.Errorf("%q, cursor %v: wrong style", c.mark, c.cur)
		}
	}
	g2 := NewGitView(map[string]byte{"build/a.o": 'M'}, map[string]bool{"build/": true})
	for _, cur := range []bool{false, true} {
		l := explorerRow(g2, ExplorerRow{Path: "build/a.o", Name: "a.o", Depth: 1}, cur, 30)
		if l.width() != 30 || strings.Contains(plainOf(l), "M") {
			t.Errorf("an ignored row, cursor %v: %q", cur, plainOf(l))
		}
		if st := l[2].st; st != either(cur, stExpRowGrey, stGrey) {
			t.Errorf("an ignored row, cursor %v: its name not dimmed", cur)
		}
	}
	l := explorerRow(g, ExplorerRow{Path: "src", Name: "src", Dir: true, Open: true}, false, 20)
	if got := plainOf(l); got != "▾ src             ● " {
		t.Errorf("an open dirty folder: %q", got)
	}
}

// TestExplorerDeleted is plan 17, criteria 6.1 and 6.2: each deleted path a
// row under its folder, in the folder's order, red with a red D; a deleted
// folder rebuilt from the paths under it, never read; a name the disk has
// listed once; a folder empty on the disk listing its deleted paths; the
// folders above a deletion keep their ●; the path row says it is deleted.
func TestExplorerDeleted(t *testing.T) {
	g := NewGitView(map[string]byte{"main.go": 'D', "docs/old.md": 'D', "gone/a/b.txt": 'D', "gone/c.txt": 'D',
		"src/x.go": 'M', "keep.go": 'D', "empty/z.go": 'D', "Big/sub/q.go": 'D'}, nil)
	v := ExplorerView{
		Dirs: map[string]*ExplorerDir{
			"":      {Entries: []ExplorerEntry{{"Big", true, false}, {"docs", true, false}, {"empty", true, false}, {"src", true, false}, {"keep.go", false, false}, {"zz.go", false, false}}},
			"docs":  {Entries: []ExplorerEntry{{"guide.md", false, false}, {"zeta.md", false, false}}},
			"empty": {},
			"Big":   {Entries: []ExplorerEntry{{"a.go", false, false}}, Truncated: true},
			"gone":  {State: "never read"},
		},
		Opened: map[string]bool{"Big": true, "docs": true, "gone": true, "gone/a": true, "empty": true},
		Git:    g,
	}
	rows := ExplorerRows(v)
	want := []string{"Big", "  a.go", "  first 1 entries", "docs", "  guide.md", "  old.md", "  zeta.md", "empty", "  z.go",
		"gone", "  a", "    b.txt", "  c.txt", "src", "keep.go", "main.go", "zz.go"}
	if got := treeText(rows); !reflect.DeepEqual(got, want) {
		t.Fatalf("rows %q, want %q", got, want)
	}
	deleted := map[string]bool{"docs/old.md": true, "empty/z.go": true, "gone": true, "gone/a": true, "gone/a/b.txt": true, "gone/c.txt": true, "main.go": true}
	for _, r := range rows {
		if r.Path != "" && r.Deleted != deleted[r.Path] {
			t.Errorf("%q: deleted %v", r.Path, r.Deleted)
		}
	}
	if r := rows[10]; !r.Dir || !r.Open || r.Path != "gone/a" || r.Depth != 1 {
		t.Errorf("the deleted folder gone/a: %+v", r)
	}
	for _, c := range []struct {
		row  int
		want string
	}{{5, "D"}, {9, "D"}, {10, "D"}, {11, "D"}, {15, "D"}, {3, "●"}, {7, "●"}, {14, "D"}, {16, ""}} {
		if got := rowMark(g, rows[c.row]); got != c.want {
			t.Errorf("%q: mark %q, want %q", rows[c.row].Path, got, c.want)
		}
	}
	for _, cur := range []bool{false, true} {
		l := explorerRow(g, rows[15], cur, 20)
		if got := plainOf(l); got != "  main.go         D " {
			t.Errorf("a deleted file, cursor %v: %q", cur, got)
		}
		if l[2].st != either(cur, stExpRowFailed, stFailed) || l[len(l)-2].st != either(cur, stExpRowFailed, stFailed) {
			t.Errorf("a deleted file, cursor %v: its name or its D not red", cur)
		}
	}
	v.Cursor = "gone/a"
	if body := texts(explorerWindow(state.New(), UI{Width: 80, Height: 24, Explorer: v}, 60, 22)); !strings.Contains(body, "gone/a/ · deleted, not on disk") {
		t.Errorf("the path row of a deleted folder:\n%s", body)
	}
}

// plainOf is a line's text without its styles.
func plainOf(l line) string {
	var b strings.Builder
	for _, s := range l {
		b.WriteString(s.text)
	}
	return b.String()
}
