package render

import (
	"fmt"
	"sort"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/Evengelius/agent-command-center/internal/state"
)

// The Explorer (plan 17, amendment 12): the project as a tree in the
// viewer's window, each folder read when it is opened, git's markers beside
// the rows. The application reads the folders and git in commands; this
// draws what it holds.

// ExplorerView is the Explorer as the application holds it.
type ExplorerView struct {
	Open   bool
	Name   string                  // the project's name, on the top edge
	Dirs   map[string]*ExplorerDir // the folders read, by path in the project; "" the project
	Opened map[string]bool         // the folders open
	Cursor string                  // the path of the row under the cursor
	Top    int                     // the first row shown, before ListTop keeps the cursor in view
	Git    GitView
}

// ExplorerDir is a folder as read: its entries, or why there are none.
type ExplorerDir struct {
	Entries   []ExplorerEntry
	Truncated bool   // the listing stopped at its limit
	State     string // why the folder cannot be read; "" once read
}

// ExplorerEntry is an entry of a folder.
type ExplorerEntry struct {
	Name    string
	Dir     bool
	Deleted bool // a path git versions that the disk no longer has
}

// GitView is git's markers as the Explorer shows them.
type GitView struct {
	Marks   map[string]byte            // each changed path's marker, a folder's path ending in "/"
	Ignored map[string]bool            // the paths git ignores, a folder's ending in "/"
	Dirty   map[string]bool            // the folders holding a change, without "/"
	Deleted map[string][]ExplorerEntry // each folder's deleted entries, sorted; a deleted folder rebuilt from the paths under it
	Note    string                     // "not a git repository", "git not found", git's error; "" none
	Failed  bool                       // Note is an error
}

// NewGitView is the view of git's markers: each folder above a changed path
// dirty; each deleted path, and each folder above it, an entry of its
// folder, shown where the disk has no entry of that name.
func NewGitView(marks map[string]byte, ignored map[string]bool) GitView {
	dirty := map[string]bool{}
	deleted := map[string][]ExplorerEntry{}
	seen := map[string]bool{}
	for p, c := range marks {
		if c == 'D' && !strings.HasSuffix(p, "/") {
			for q, dir := p, false; q != "" && !seen[q]; q, dir = ParentPath(q), true {
				seen[q] = true
				parent := ParentPath(q)
				deleted[parent] = append(deleted[parent], ExplorerEntry{Name: q[len(parent)+min(len(parent), 1):], Dir: dir, Deleted: true})
			}
		}
		p = strings.TrimSuffix(p, "/")
		for i := strings.LastIndex(p, "/"); i > 0; i = strings.LastIndex(p, "/") {
			p = p[:i]
			dirty[p] = true
		}
	}
	for _, es := range deleted {
		sort.Slice(es, func(i, j int) bool { return entryLess(es[i], es[j]) })
	}
	return GitView{Marks: marks, Ignored: ignored, Dirty: dirty, Deleted: deleted}
}

// entryLess is a folder's order, ListDir's: folders first, then files, each
// sorted without case.
func entryLess(a, b ExplorerEntry) bool {
	if a.Dir != b.Dir {
		return a.Dir
	}
	la, lb := strings.ToLower(a.Name), strings.ToLower(b.Name)
	if la != lb {
		return la < lb
	}
	return a.Name < b.Name
}

// withDeleted is a folder's entries on the disk, sorted, with its deleted
// ones merged in order, a name the disk also has left out. Under a cut
// listing a deleted folder is left out too: the folder may be past the cut.
func withDeleted(disk, gone []ExplorerEntry, cut bool) []ExplorerEntry {
	if len(gone) == 0 {
		return disk
	}
	on := map[string]bool{}
	for _, e := range disk {
		on[strings.ToLower(e.Name)] = true
	}
	out := make([]ExplorerEntry, 0, len(disk)+len(gone))
	i := 0
	for _, g := range gone {
		if on[strings.ToLower(g.Name)] || cut && g.Dir {
			continue
		}
		for i < len(disk) && entryLess(disk[i], g) {
			out = append(out, disk[i])
			i++
		}
		out = append(out, g)
	}
	return append(out, disk[i:]...)
}

// ExplorerRow is a row of the tree: an entry, or a note under a folder.
type ExplorerRow struct {
	Path    string // the entry's path in the project, with "/"; "" for a note
	Name    string // the entry's name, or the note
	Dir     bool
	Open    bool // a folder open, its entries under it
	Depth   int
	Failed  bool // a note saying why a folder cannot be read
	Deleted bool // a path git versions that the disk no longer has
}

// ExplorerRows is the tree as drawn: each folder's entries, the deleted
// ones git reports among them, an open folder's under it; "reading…" under
// a folder not read yet, "empty folder", a folder's reason, "first N
// entries" after a cut listing. A deleted folder is never read: its entries
// are git's.
func ExplorerRows(v ExplorerView) []ExplorerRow {
	var rows []ExplorerRow
	var walk func(rel string, depth int, deleted bool)
	walk = func(rel string, depth int, deleted bool) {
		d := v.Dirs[rel]
		if deleted {
			d = &ExplorerDir{}
		}
		switch {
		case d == nil:
			rows = append(rows, ExplorerRow{Name: "reading…", Depth: depth})
			return
		case d.State != "":
			rows = append(rows, ExplorerRow{Name: d.State, Depth: depth, Failed: true})
			return
		}
		entries := withDeleted(d.Entries, v.Git.Deleted[rel], d.Truncated)
		if len(entries) == 0 {
			rows = append(rows, ExplorerRow{Name: "empty folder", Depth: depth})
			return
		}
		for _, e := range entries {
			p := e.Name
			if rel != "" {
				p = rel + "/" + e.Name
			}
			open := e.Dir && v.Opened[p]
			rows = append(rows, ExplorerRow{Path: p, Name: e.Name, Dir: e.Dir, Open: open, Depth: depth, Deleted: e.Deleted})
			if open {
				walk(p, depth+1, e.Deleted)
			}
		}
		if d.Truncated {
			rows = append(rows, ExplorerRow{Name: "first " + grouped(len(d.Entries)) + " entries", Depth: depth})
		}
	}
	walk("", 0, false)
	return rows
}

// ExplorerCursor is the row of rows the cursor is on: the one at path, else
// the nearest folder above it shown, else the first entry; -1 with none.
func ExplorerCursor(rows []ExplorerRow, path string) int {
	for p := path; p != ""; p = ParentPath(p) {
		for i, r := range rows {
			if r.Path == p {
				return i
			}
		}
	}
	for i, r := range rows {
		if r.Path != "" {
			return i
		}
	}
	return -1
}

// ParentPath is the folder holding path, "" for the project's top level.
func ParentPath(path string) string {
	if i := strings.LastIndex(path, "/"); i >= 0 {
		return path[:i]
	}
	return ""
}

// explorerMark is the marker of the row at path: its own, 'U' under a
// folder all untracked, "●" on a folder holding a change; "" none.
func explorerMark(g GitView, path string, dir bool) string {
	key := path
	if dir {
		key += "/"
	}
	if c, ok := g.Marks[key]; ok {
		return string(c)
	}
	for p := ParentPath(path); p != ""; p = ParentPath(p) {
		if g.Marks[p+"/"] == 'U' {
			return "U"
		}
	}
	if dir && g.Dirty[path] {
		return "●"
	}
	return ""
}

// explorerIgnored is true for a path git ignores, or one under a folder it
// ignores.
func explorerIgnored(g GitView, path string) bool {
	if g.Ignored[path] || g.Ignored[path+"/"] {
		return true
	}
	for p := ParentPath(path); p != ""; p = ParentPath(p) {
		if g.Ignored[p+"/"] {
			return true
		}
	}
	return false
}

// rowMark is the marker of a row: "D" on a deleted one, else its path's.
func rowMark(g GitView, r ExplorerRow) string {
	if r.Deleted {
		return "D"
	}
	return explorerMark(g, r.Path, r.Dir)
}

// markStyle is a marker's colour, on the cursor row's background when cur:
// modified and "●" amber, untracked, added and renamed green, a conflict
// and a deletion red (plan 17, amendment 12).
func markStyle(m string, cur bool) *lipgloss.Style {
	switch m {
	case "U", "A", "R":
		return either(cur, stExpRowDone, stDone)
	case "!", "D":
		return either(cur, stExpRowFailed, stFailed)
	}
	return either(cur, stExpRowProgress, stProgress)
}

func either(cond bool, a, b *lipgloss.Style) *lipgloss.Style {
	if cond {
		return a
	}
	return b
}

// explorerKeys are the Explorer's key hints, the longest first.
var explorerKeys = []string{
	"↑↓ PgUp PgDn Home End move · → open · ← close · Enter view file · / quick open · F1 keys · Esc back · F3 close",
	"↑↓ move · → open · ← close · Enter view · / quick open · Esc back · F3 close",
	"↑↓ · → open · ← close · Enter view · / find · Esc back · F3 close",
	"Enter view · / find · Esc back · F3 close",
}

// explorerWindow is the Explorer's window, ww by hh: the cursor's path and
// git's note, the tree, the footer.
func explorerWindow(s *state.State, ui UI, ww, hh int) []line {
	v := ui.Explorer
	inner := ww - 4
	rows := ExplorerRows(v)
	n := 0
	for _, r := range rows {
		if r.Path != "" {
			n++
		}
	}
	items := grouped(n) + " items"
	if n == 1 {
		items = "1 item"
	}
	right := decoLine(items, stGrey)
	if v.Name != "" {
		right = line{deco(v.Name, stText), deco(" · "+items, stGrey)}
	}
	cur := ExplorerCursor(rows, v.Cursor)
	shown := ""
	if cur >= 0 {
		shown = rows[cur].Path
		if rows[cur].Dir {
			shown += "/"
		}
		if rows[cur].Deleted {
			shown += " · deleted, not on disk"
		}
	}
	note := decoLine(v.Git.Note, either(v.Git.Failed, stAttention, stGrey))
	body := []line{spread(line{deco(elide(shown, max(inner-note.width()-1, 8)), stText)}, note, inner)}

	room := hh - 4
	top := ListTop(v.Top, max(cur, 0), room, len(rows))
	for i := top; i < top+room && i < len(rows); i++ {
		body = append(body, explorerRow(v.Git, rows[i], i == cur, inner))
	}
	for len(body) < room+1 {
		body = append(body, nil)
	}
	var count line
	if cur >= 0 {
		count = decoLine(fmt.Sprintf("%s of %s", grouped(entryIndex(rows, cur)+1), grouped(n)), stGrey)
	}
	footer := spread(windowLeft(s, ui, explorerKeys, inner-count.width()-1), count, inner)
	body = append(body[:room+1], footer)
	return frameIn(frameBox(line{deco("Explorer", stTitle)}, right, body, ww, hh, stFileFrame))
}

// entryIndex is how many entry rows come before row i.
func entryIndex(rows []ExplorerRow, i int) int {
	k := 0
	for _, r := range rows[:i] {
		if r.Path != "" {
			k++
		}
	}
	return k
}

// explorerRow is a row of the tree, w cells: its indent, "▸" or "▾" before
// a folder, the name, git's marker on the right; dimmed when git ignores
// it, red when deleted, on the cursor row's background when cur.
func explorerRow(g GitView, r ExplorerRow, cur bool, w int) line {
	indent := strings.Repeat("  ", r.Depth)
	if r.Path == "" {
		return line{deco(indent+"  ", stNone), deco(elide(r.Name, max(w-len(indent)-2, 1)), either(r.Failed, stAttention, stGrey))}
	}
	mark := rowMark(g, r)
	ignored := !r.Deleted && explorerIgnored(g, r.Path)
	nameSt := stText
	switch {
	case r.Deleted:
		nameSt = either(cur, stExpRowFailed, stFailed)
	case ignored:
		nameSt = either(cur, stExpRowGrey, stGrey)
	case r.Dir:
		nameSt = either(cur, stExpRowDir, stTitle)
	case cur:
		nameSt = stTreeCursor
	}
	head := "  "
	switch {
	case r.Open:
		head = "▾ "
	case r.Dir:
		head = "▸ "
	}
	var right line
	if mark != "" && !ignored {
		right = line{deco(mark, markStyle(mark, cur)), deco(" ", either(cur, stTreeCursor, stNone))}
	}
	pad := either(cur, stTreeCursor, stNone)
	l := line{deco(indent, pad), deco(head, either(cur, stExpRowGrey, stGrey)),
		deco(elide(r.Name, max(w-len(indent)-2-right.width()-1, 1)), nameSt)}
	return spreadOn(l, right, w, pad)
}

// spreadOn is l on the left and right on the right, w cells, the gap in
// the style st.
func spreadOn(l, right line, w int, st *lipgloss.Style) line {
	if n := w - l.width() - right.width(); n > 0 {
		l = append(l, deco(strings.Repeat(" ", n), st))
	}
	return append(l, right...)
}
