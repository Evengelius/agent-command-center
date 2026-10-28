package render

import (
	"reflect"
	"strings"
	"testing"

	"github.com/Evengelius/agent-command-center/internal/provider/claude"
)

// branchesSample is the popup over a repository like the owner's image's:
// two recent branches, four local ones (one in another worktree), two
// remote ones and two tags, Remote and Tags folded, the cursor on main.
func branchesSample() BranchesView {
	v := BranchesView{Open: true, Head: "main", Upstream: "origin/main", Ahead: 3, Behind: 1, Cursor: "b:1:main"}
	v.Groups[GroupRecent] = []BranchRow{{Name: "feat", Upstream: "origin/feat", Behind: 2}, {Name: "chore/ranorex"}}
	v.Groups[GroupLocal] = []BranchRow{{Name: "chore/ranorex"}, {Name: "feat", Upstream: "origin/feat", Behind: 2},
		{Name: "main", Upstream: "origin/main", Ahead: 3, Behind: 1, Current: true}, {Name: "wtb", Elsewhere: `C:\work\wt`}}
	v.Groups[GroupRemote] = []BranchRow{{Name: "origin/feat"}, {Name: "origin/main"}}
	v.Groups[GroupTags] = []BranchRow{{Name: "v2"}, {Name: "v1"}}
	for g := range BranchGroups {
		v.Totals[g] = len(v.Groups[g])
	}
	v.Folded[GroupRemote], v.Folded[GroupTags] = true, true
	return v
}

func branchKeys(rows []BranchItem) []string {
	var k []string
	for _, r := range rows {
		if r.Kind == 'm' {
			k = append(k, "m:"+r.Text)
			continue
		}
		k = append(k, r.Key())
	}
	return k
}

// TestBranchRows is plan 59, criteria 59.2 to 59.4, in the list: the groups'
// first folding, a search showing every group it matches open and hiding the
// others, nothing matching, an empty group and the cap's note.
func TestBranchRows(t *testing.T) {
	v := branchesSample()
	want := []string{"g:0", "b:0:feat", "b:0:chore/ranorex", "g:1", "b:1:chore/ranorex", "b:1:feat", "b:1:main", "b:1:wtb", "g:2", "g:3"}
	if got := branchKeys(BranchRows(v)); !reflect.DeepEqual(got, want) {
		t.Errorf("rows %q, want %q", got, want)
	}
	v.Query = "FEA"
	want = []string{"g:0", "b:0:feat", "g:1", "b:1:feat", "g:2", "b:2:origin/feat"}
	if got := branchKeys(BranchRows(v)); !reflect.DeepEqual(got, want) {
		t.Errorf("searching FEA: %q, want %q", got, want)
	}
	v.Query = "zzz"
	if got := branchKeys(BranchRows(v)); !reflect.DeepEqual(got, []string{"m:No branch or tag matches “zzz”"}) {
		t.Errorf("searching zzz: %q", got)
	}
	v.Groups[GroupLocal] = append(v.Groups[GroupLocal], BranchRow{Name: "Feature/Login"})
	v.Totals[GroupLocal]++
	v.Query = "login"
	if got := branchKeys(BranchRows(v)); !reflect.DeepEqual(got, []string{"g:1", "b:1:Feature/Login"}) {
		t.Errorf("searching login: %q", got)
	}
	v = branchesSample()
	v.Groups[GroupRecent], v.Folded[GroupTags], v.Totals[GroupTags] = nil, false, 1500
	want = []string{"g:0", "m:No recent branches", "g:1", "b:1:chore/ranorex", "b:1:feat", "b:1:main", "b:1:wtb", "g:2",
		"g:3", "t:3:v2", "t:3:v1", "m:first 2 shown"}
	if got := branchKeys(BranchRows(v)); !reflect.DeepEqual(got, want) {
		t.Errorf("an empty group and a cut one: %q, want %q", got, want)
	}
	if i := BranchCursor(BranchRows(v), "b:1:main"); i != 5 {
		t.Errorf("the cursor's row %d, want 5", i)
	}
}

// branchesBox draws the dashboard at w x h with the branch name "main" in
// the project bar and the popup v, and gives the frame and the box's rows'
// text from its top edge, the cells between its edges.
func branchesBox(t *testing.T, v BranchesView, w, h int, nerd bool) (*Screen, []string, int, int) {
	t.Helper()
	s := probeState(t, "p2-claude-conversation.jsonl", claude.ProviderID, claude.NewMapper())
	sc := Draw(s, UI{Width: w, Height: h, Focus: "B", Branch: "main", Branches: v, NerdIcons: nerd})
	_, by, ok := sc.find(BranchPanel)
	if !ok {
		t.Fatal("no branch name in the project bar")
	}
	var rows []string
	x := -1
	for y := by + 1; y < len(sc.lines); y++ {
		l := plainOf(sc.lines[y])
		if x < 0 {
			if x = strings.Index(l, "╭─ Branches "); x < 0 {
				return sc, nil, -1, by + 1
			}
			x = len([]rune(l[:x]))
		}
		r := []rune(l)
		end := x
		for end < len(r) && (end == x || r[end] != '│' && r[end] != '╮' && r[end] != '╯' && r[end] != '┃') {
			end++
		}
		rows = append(rows, string(r[x:min(end+1, len(r))]))
		if strings.HasPrefix(rows[len(rows)-1], "╰") {
			break
		}
	}
	return sc, rows, x, by + 1
}

// TestBranchesPopup is plan 59, design D2: the box under the branch name, its
// title, search row, current state, groups, rows (the current branch's ✓, an
// upstream with ↑ and ↓, a branch in another worktree), the cursor's band,
// the keys, and its cells' items.
func TestBranchesPopup(t *testing.T) {
	defer UsePalette(pickTrue)
	UsePalette(pickTrue)
	sc, rows, x, y := branchesBox(t, branchesSample(), 200, 60, false)
	if x < 0 {
		t.Fatal("no box under the branch name")
	}
	bx, _, _ := sc.find(BranchPanel)
	if x != bx-1 {
		t.Errorf("the box at column %d, want %d (one left of the branch name)", x, bx-1)
	}
	for i, want := range []string{
		"╭─ Branches ─", "│ › ▌Search branches and tags", "│ main → origin/main ↑3 ↓1", "│ ⌄ Recent", "│     feat",
		"│     chore/ranorex", "│ ⌄ Local", "│     chore/ranorex", "│     feat", "│   ✓ main", "│     wtb", "│ › Remote", "│ › Tags",
		"│ ↑↓ move · Enter switch · Ctrl+C copy · F5 fetch · Esc close", "╰─",
	} {
		if i >= len(rows) || !strings.HasPrefix(rows[i], want) {
			t.Fatalf("row %d %q, want %q…\n%s", i, at(rows, i), want, strings.Join(rows, "\n"))
		}
	}
	for i, end := range map[int]string{1: "F5 fetch │", 4: "origin/feat ↓2 │", 9: "origin/main ↑3 ↓1 │", 10: "in another worktree │"} {
		if !strings.HasSuffix(rows[i], end) {
			t.Errorf("row %d %q, want …%q", i, rows[i], end)
		}
	}
	// The box's cells of its row i, not the dashboard's around it.
	line := func(i int) line { return cols(sc.lines[y+i], x, x+len([]rune(rows[i]))) }
	if s, _ := spanOf(line(9), "✓"); s.st != bandGround.on(stDone) {
		t.Error("the current branch's ✓ is not the done colour on the cursor's band")
	}
	if s, _ := spanOf(line(8), "↓2"); s.st != stAttention {
		t.Error("↓2 is not the attention colour")
	}
	if s, _ := spanOf(line(5), "chore/"); s.st != stGrey {
		t.Error("the part before the slash is not grey")
	}
	if s, _ := spanOf(line(10), "wtb"); s.st != stGrey {
		t.Error("the branch in another worktree is not grey")
	}
	if p, item := sc.ItemAt(x+6, y+8); p != BranchesPanel || item != ItemBranchRow+5 {
		t.Errorf("feat's row: panel %c item %d, want %c %d", p, item, BranchesPanel, ItemBranchRow+5)
	}
	if p, item := sc.ItemAt(x+len([]rune(rows[1]))-4, y+1); p != BranchesPanel || item != ItemBranchFetch {
		t.Errorf("fetch: panel %c item %d", p, item)
	}
	if p, item := sc.ItemAt(x, y+2); p != BranchesPanel || item != ItemBranchFrame {
		t.Errorf("the frame: panel %c item %d", p, item)
	}
	if p, item := sc.ItemAt(bx+len("branch "), y-1); p != BranchPanel || item != 1 {
		t.Errorf("the branch name: panel %c item %d", p, item)
	}
	if b := sc.Branches; !b.Shown || b.Top != 0 || b.Height != 10 || b.Rows != 10 {
		t.Errorf("shown %+v", b)
	}
}

func at(rows []string, i int) string {
	if i < len(rows) {
		return rows[i]
	}
	return ""
}

// TestBranchesPopupStates is plan 59, criteria 59.4 and 59.5, drawn: the
// switch's confirmation in its exact words, a fetch running, the reading, a
// failure, a detached HEAD and a branch with no commits; the Nerd Font
// icons in the acc tab; a small screen's scrolling list; and no box without
// the branch name.
func TestBranchesPopupStates(t *testing.T) {
	v := branchesSample()
	v.Confirm, v.Changes = "feat", 3
	_, rows, _, _ := branchesBox(t, v, 200, 60, false)
	// The box's rows inside its edges, joined: the wrapped sentence whole.
	var inside []string
	for _, r := range rows {
		if s := strings.TrimSpace(strings.Trim(r, "│┃")); s != "" {
			inside = append(inside, s)
		}
	}
	want := "3 uncommitted changes. Git keeps them on feat unless they touch files that differ there; then it refuses and nothing changes. Enter switch · Esc cancel"
	if !strings.Contains(strings.Join(inside, " "), want) {
		t.Errorf("the confirmation is not %q:\n%s", want, strings.Join(rows, "\n"))
	}
	// One change, in the singular.
	v.Changes = 1
	_, rows, _, _ = branchesBox(t, v, 200, 60, false)
	inside = inside[:0]
	for _, r := range rows {
		if s := strings.TrimSpace(strings.Trim(r, "│┃")); s != "" {
			inside = append(inside, s)
		}
	}
	want = "1 uncommitted change. Git keeps it on feat unless it touches a file that differs there; then Git refuses and nothing changes. Enter switch · Esc cancel"
	if !strings.Contains(strings.Join(inside, " "), want) {
		t.Errorf("the confirmation is not %q:\n%s", want, strings.Join(rows, "\n"))
	}
	v = branchesSample()
	v.Busy = "fetching… Esc stops"
	if _, rows, _, _ = branchesBox(t, v, 200, 60, false); !strings.HasSuffix(rows[1], "fetching… │") || !strings.Contains(rows[len(rows)-2], "fetching… Esc stops") {
		t.Errorf("fetching: %q, %q", rows[1], rows[len(rows)-2])
	}
	// A note of git's that runs long takes three rows, the last cut.
	v = branchesSample()
	v.Note, v.NoteBad = "error: Your local changes to the following files would be overwritten by checkout: "+strings.Repeat("src/a/long/path/file.go ", 20), true
	_, rows, _, _ = branchesBox(t, v, 200, 60, false)
	n := len(rows)
	note := rows[n-4 : n-1] // the three rows above the bottom edge
	last := strings.TrimRight(strings.TrimSuffix(note[2], "│"), " ")
	if !strings.HasPrefix(note[0], "│ error: Your local changes") || !strings.HasSuffix(last, "…") || strings.Contains(rows[n-5], "src/a/long") {
		t.Errorf("a long note:\n%s", strings.Join(rows, "\n"))
	}
	for _, c := range []struct {
		name string
		edit func(*BranchesView)
		row  int
		want string
	}{
		{"reading", func(v *BranchesView) { v.Reading = true }, 3, "│  reading…"},
		{"a failure", func(v *BranchesView) { v.Problem = "git not found" }, 3, "│  git not found"},
		{"detached", func(v *BranchesView) { v.Detached, v.ID, v.Head = true, "a1b2c3d", "" }, 2, "│ detached at a1b2c3d"},
		{"no commits", func(v *BranchesView) { v.NoCommits, v.Upstream = true, "" }, 2, "│ main, no commits yet"},
	} {
		v := branchesSample()
		c.edit(&v)
		if _, rows, _, _ := branchesBox(t, v, 200, 60, false); !strings.HasPrefix(at(rows, c.row), c.want) {
			t.Errorf("%s: row %d %q, want %q…", c.name, c.row, at(rows, c.row), c.want)
		}
	}

	_, rows, _, _ = branchesBox(t, branchesSample(), 200, 60, true)
	for i, want := range map[int]string{1: " fetch │", 4: "│     " + string(rune(0xF418)) + " feat", 3: "│ " + string(rune(0xEAB4)) + " Recent"} {
		if !strings.HasPrefix(rows[i], want) && !strings.HasSuffix(rows[i], want) {
			t.Errorf("Nerd Font icons: row %d %q, want %q", i, rows[i], want)
		}
	}
	if !strings.Contains(rows[1], string(rune(0xEB37))) {
		t.Errorf("no refresh icon on the search row: %q", rows[1])
	}

	v = branchesSample()
	v.Folded[GroupRemote], v.Folded[GroupTags] = false, false
	for i := range 30 {
		v.Groups[GroupTags] = append(v.Groups[GroupTags], BranchRow{Name: "t" + string(rune('a'+i%26)) + string(rune('0'+i/26))})
	}
	v.Totals[GroupTags] = len(v.Groups[GroupTags])
	v.Cursor = "t:3:" + v.Groups[GroupTags][len(v.Groups[GroupTags])-1].Name
	sc, rows, _, _ := branchesBox(t, v, 80, 24, false)
	if len(rows) == 0 || len([]rune(rows[0])) > 80 || len(rows) > 24-2 || !strings.Contains(strings.Join(rows, ""), "┃") {
		t.Errorf("80x24: %d rows, the first %d wide, a scroll bar %v", len(rows), len([]rune(at(rows, 0))), strings.Contains(strings.Join(rows, ""), "┃"))
	}
	if b := sc.Branches; b.Top == 0 || b.Top+b.Height != b.Rows {
		t.Errorf("80x24: the last tag's cursor at the end of the list shown: %+v", b)
	}
	// The rows drawn are the list's from its first shown: under the top
	// edge, the search and the state, the row at Top first, the cursor's
	// tag last.
	all, b := BranchRows(v), sc.Branches
	name := func(it BranchItem) string {
		if it.Kind == 'g' {
			return branchGroupNames[it.Group]
		}
		return it.Row.Name
	}
	if first, last := at(rows, 3), at(rows, 3+b.Height-1); b.Height < 2 || !strings.Contains(first, " "+name(all[b.Top])) ||
		!strings.Contains(last, " "+name(all[b.Top+b.Height-1])) || !strings.Contains(last, " td1") {
		t.Errorf("80x24, from row %d: drawn %q … %q, want %q … %q", b.Top, first, last, name(all[b.Top]), name(all[b.Top+b.Height-1]))
	}

	s := probeState(t, "p2-claude-conversation.jsonl", claude.ProviderID, claude.NewMapper())
	if sc := Draw(s, UI{Width: 200, Height: 60, Focus: "B", Branches: branchesSample()}); sc.Branches.Shown {
		t.Error("a box with no branch name in the project bar")
	}
}

// TestBranchAsideWidth checks the width the box is measured with is the
// width of the aside drawn, for every kind of local branch.
func TestBranchAsideWidth(t *testing.T) {
	for _, r := range []BranchRow{
		{Name: "a"},
		{Name: "a", Elsewhere: `C:\work\a`},
		{Name: "a", Upstream: "origin/a"},
		{Name: "a", Upstream: "origin/a", Gone: true},
		{Name: "a", Upstream: "origin/a", Ahead: 7},
		{Name: "a", Upstream: "origin/a", Behind: 12},
		{Name: "a", Upstream: "origin/a", Ahead: 1234, Behind: 99, Gone: true},
		{Name: "a", Upstream: "origine/été"},
	} {
		if got, want := branchAsideWidth(r), branchAside(r).width(); got != want {
			t.Errorf("%+v: counted %d, drawn %d", r, got, want)
		}
	}
}
