package render

import (
	"encoding/hex"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/Evengelius/agent-command-center/internal/state"
)

// hashBytes is commit n's made-up hash.
func hashBytes(n int) [20]byte {
	var h [20]byte
	b, _ := hex.DecodeString(fmt.Sprintf("%040x", n))
	copy(h[:], b)
	return h
}

// graphSample is a history of four commits, the newest first: main's tip,
// a merge, a commit on main with origin/main and a tag, the root; the
// merge unfolded into five files and the guard's note; the cursor on the
// first commit.
func graphSample() GraphView {
	v := GraphView{Open: true, Scope: "Auto", Follows: "main → origin/main",
		Commits: []GraphEntry{
			{Hash: hashBytes(4), Parent: hashBytes(3), Parents: 1, Author: "Ana Hubaut", Subject: "Name the footer's parts"},
			{Hash: hashBytes(3), Parent: hashBytes(2), Parents: 2, Author: "Bo Li", Subject: "Merge branch 'rows'"},
			{Hash: hashBytes(2), Parent: hashBytes(1), Parents: 1, Author: "Ana Hubaut", Subject: "Rework the rows: no CR, no last empty line"},
			{Hash: hashBytes(1), Author: "Ana Hubaut", Subject: "Add the viewer"},
		},
		Labels: map[int32][]GraphLabel{
			0: {{Name: "main", Kind: 'h', Head: true}},
			2: {{Name: "origin/main", Kind: 'r'}, {Name: "v1.0", Kind: 't'}},
		},
		Files: map[int32]*GraphFiles{1: {Total: 7, Files: []GraphFile{
			{Status: 'M', Path: "internal/render/file.go", Added: 12, Removed: 3},
			{Status: 'A', Path: "doc/graph.md", Added: 40},
			{Status: 'D', Path: "old.txt", Removed: 2},
			{Status: 'R', Path: "web/app.ts", Orig: "web/old.ts", Added: 1, Removed: 1},
			{Status: 'T', Path: "link"},
			{Status: 'A', Path: "logo.png", Binary: true},
		}}},
	}
	v.Rows = []GraphRowRef{{0, -1}, {1, -1}, {1, 0}, {1, 1}, {1, 2}, {1, 3}, {1, 4}, {1, 5}, {1, 6}, {2, -1}, {3, -1}}
	return v
}

// graphFrame draws the dashboard at w x h with the Git Graph's panel shown,
// and gives the screen and its rows' text.
func graphFrame(v GraphView, sv ScopeView, w, h int) (*Screen, []string) {
	s := state.New()
	s.Screen = state.ScreenDashboard
	sc := Draw(s, UI{Width: w, Height: h, Focus: "R", Graph: v, Scope: sv})
	return sc, strings.Split(texts(sc.lines), "\n")
}

// TestGraphPanel is plan 62's panel as drawn (designs D2, D5 and D11): the
// header and the scope row; a commit's dot, a merge's ring, its labels and
// its subject, cut with "…" only where the row ends, the labels giving way
// first, the author where room remains; an unfolded commit's files, their
// letters in Source Control's colours, their counts, a rename's source, a
// binary file and the guard's note; the cursor's row on the band; each
// row's cells its row plus one, the scope row's ItemGraphScope; the states
// in the history's place.
func TestGraphPanel(t *testing.T) {
	v := graphSample()
	sc, rows := graphFrame(v, ScopeView{}, 120, 40) // the panel in zone B's place, 76 columns
	for _, want := range []string{
		"GIT GRAPH",
		"◎ Auto ⌄ · main → origin/main",
		"● ◎ main Name the footer's parts  Ana Hubaut",
		"◉ Merge branch 'rows'  Bo Li",
		"    M  file.go internal/render/",
		"+12 −3",
		"    R  app.ts ← web/old.ts web/",
		"    T  link",
		"    A  logo.png",
		"binary",
		"    first 6 of 7 files",
		"● origin/main v1.0 Rework the rows",
		"● Add the viewer  Ana Hubaut",
	} {
		if rowWith(rows, want) < 0 {
			t.Errorf("120x40: no row %q:\n%s", want, strings.Join(rows, "\n"))
		}
	}
	y := rowWith(rows, "Name the footer's parts")
	if st := blameStyle(sc, y, "Name the footer's parts"); st != bandGround.on(stText) {
		t.Errorf("the cursor's row not on the band")
	}
	fy, ry := rowWith(rows, "file.go"), rowWith(rows, "app.ts")
	if blameStyle(sc, fy, "M") != letterStyle('M') || blameStyle(sc, ry, "R") != letterStyle('R') ||
		blameStyle(sc, fy, "+12") != stDiffAdd || blameStyle(sc, fy, " −3") != stDiffDel {
		t.Errorf("a file's letter or counts not in their colours")
	}
	if x, y, ok := sc.findItem(GraphPanel, ItemGraphScope); !ok || !strings.Contains(rows[y][strings.Index(rows[y], "◎"):], "Auto") || x < 0 {
		t.Errorf("the scope row's item not on it")
	}
	if _, item := sc.ItemAt(strings.Index(rows[rowWith(rows, "Add the viewer")], "Add the viewer"), rowWith(rows, "Add the viewer")); item != 11 {
		t.Errorf("the last row's item %d, want 11", item)
	}

	// Docked at 208x59, 31 columns: the long subject cut with "…", its labels
	// given way first, no author.
	_, rows = graphFrame(v, ScopeView{}, 208, 59)
	if y := rowWith(rows, "Rework the"); y < 0 || !strings.Contains(rows[y], "…") || strings.Contains(rows[y], "origin/main") || strings.Contains(rows[y], "Ana Hubaut") {
		t.Errorf("208x59: the long subject's row %q", rows[max(y, 0)])
	}
	if rowWith(rows, "● Add the viewer") < 0 {
		t.Errorf("208x59: a short subject cut:\n%s", strings.Join(rows, "\n"))
	}

	// The states in the history's place.
	for _, c := range []struct {
		state  string
		failed bool
	}{{"reading…", false}, {"no commits yet", false}, {"git: fatal: bad object", true}} {
		w := GraphView{Open: true, State: c.state, Failed: c.failed}
		sc, rows := graphFrame(w, ScopeView{}, 120, 40)
		y := rowWith(rows, c.state)
		if y < 0 || blameStyle(sc, y, c.state) != either(c.failed, stAttention, stGrey) {
			t.Errorf("the state %q not shown in its colour", c.state)
		}
	}
	w := graphSample()
	w.More = "reading…"
	if _, rows := graphFrame(w, ScopeView{}, 120, 40); rowWith(rows, "  reading…") < 0 {
		t.Errorf("the note under the last commit missing")
	}
}

// scopeSample is the popup over graphSample's panel: two local branches,
// one remote, one tag, origin/main ticked.
func scopeSample() ScopeView {
	return ScopeView{Open: true, Checked: map[string]bool{"refs/remotes/origin/main": true}, Cursor: "u",
		Groups: [ScopeGroups][]ScopeRef{
			{{Name: "refs/heads/main", Short: "main", Hash: "1a2b3c4"}, {Name: "refs/heads/feat", Short: "feat", Hash: "5d6e7f8"}},
			{{Name: "refs/remotes/origin/main", Short: "origin/main", Hash: "9a1b2c3"}},
			{{Name: "refs/tags/v1.0", Short: "v1.0", Hash: "1a2b3c4"}},
		}}
}

// TestScopePopup is design D3 as drawn: under the scope row, "History
// scope" in its top edge, the search row; All and Auto, the mode's mark;
// the groups' headings with their counts, folded or open; each reference's
// check box, name and commit, "Remote branch at" a remote one's; the
// cursor's row on the band; the search keeping its matches' groups open, a
// message when none; the shown list recorded for the wheel and clicks.
func TestScopePopup(t *testing.T) {
	sv := scopeSample()
	sc, rows := graphFrame(graphSample(), sv, 120, 40)
	for _, want := range []string{"History scope", "⌕ type to search the references", "○ All · every branch, remote branch and tag",
		"○ Auto · the current branch and its upstream", "▾ Local branches 2", "☐ main  1a2b3c4", "☐ feat  5d6e7f8", "▾ Remote branches 1",
		"☑ origin/main  Remote branch at 9a1b2c3", "▾ Tags 1", "☐ v1.0  1a2b3c4", "Space tick", "Enter apply"} {
		if rowWith(rows, want) < 0 {
			t.Errorf("no row %q:\n%s", want, strings.Join(rows, "\n"))
		}
	}
	if !sc.Scope.Shown || sc.Scope.Rows != 9 {
		t.Errorf("the list recorded %+v, want 9 rows", sc.Scope)
	}
	_, scopeY, _ := sc.findItem(GraphPanel, ItemGraphScope)
	if y := rowWith(rows, "History scope"); y != scopeY+1 {
		t.Errorf("the popup's top on row %d, want under the scope row %d", y, scopeY)
	}
	if y := rowWith(rows, "○ Auto"); y < 0 || blameStyle(sc, y, "Auto · the current") != bandGround.on(stText) {
		t.Errorf("the cursor's row, Auto, not on the band")
	}
	// No reference ticked: Auto the mode; All ticked: All.
	sv.Checked = map[string]bool{}
	if _, rows := graphFrame(graphSample(), sv, 120, 40); rowWith(rows, "◉ Auto") < 0 || rowWith(rows, "○ All") < 0 {
		t.Errorf("Auto the mode: its mark")
	}
	sv.All = true
	if _, rows := graphFrame(graphSample(), sv, 120, 40); rowWith(rows, "◉ All") < 0 || rowWith(rows, "○ Auto") < 0 {
		t.Errorf("All the mode: its mark")
	}
	sv = scopeSample()
	sv.Folded = [ScopeGroups]bool{false, true, true}
	if _, rows := graphFrame(graphSample(), sv, 120, 40); rowWith(rows, "▸ Remote branches 1") < 0 || rowWith(rows, "☑ origin/main") >= 0 {
		t.Errorf("Remote folded: its rows shown")
	}
	sv.Query = "ma"
	_, rows = graphFrame(graphSample(), sv, 120, 40)
	if rowWith(rows, "☐ main") < 0 || rowWith(rows, "☑ origin/main") < 0 || rowWith(rows, "feat") >= 0 || rowWith(rows, "Tags") >= 0 || rowWith(rows, "⌕ ma▌") < 0 {
		t.Errorf("the search \"ma\":\n%s", strings.Join(rows, "\n"))
	}
	sv.Query = "zz"
	if _, rows := graphFrame(graphSample(), sv, 120, 40); rowWith(rows, "No reference matches “zz”") < 0 || rowWith(rows, "○ Auto") < 0 {
		t.Errorf("nothing matching: no message, or All and Auto gone")
	}
}

// TestScopeRows is the popup's list (design D3): All and Auto first, the
// groups after, folded ones without their rows, an empty group's message;
// keys naming each row once.
func TestScopeRows(t *testing.T) {
	sv := scopeSample()
	sv.Groups[ScopeTags] = nil
	sv.Folded[ScopeLocal] = true
	var kinds, keys []string
	for _, r := range ScopeRows(sv) {
		kinds = append(kinds, string(r.Kind))
		keys = append(keys, r.Key())
	}
	if got := strings.Join(kinds, ""); got != "auggrgm" {
		t.Errorf("rows %q, want All, Auto, Local folded, Remote and its branch, Tags and its message", got)
	}
	if ScopeCursor(ScopeRows(sv), "r:refs/remotes/origin/main") < 0 || ScopeCursor(ScopeRows(sv), "r:refs/heads/main") >= 0 {
		t.Errorf("the cursor found a folded row, or missed an open one: %v", keys)
	}
}

// TestGraphPanelLanes is criteria 62.6 and 62.7 in the panel (designs D9 to
// D11): each row's lanes before its labels and subject, in their lanes'
// colours, on the band on the cursor's row; the rows under an unfolded
// commit under the lanes going on; lanes beyond the room hidden behind the
// overflow column, the footer saying so; the wide view over zone A, the
// author and the date in columns of their own, in the zone given.
func TestGraphPanelLanes(t *testing.T) {
	v := graphSample()
	v.Lanes = layOut(dag{{1}, {2, 3}, {3}, {}})
	sc, rows := graphFrame(v, ScopeView{}, 120, 40)
	for _, want := range []string{"●   ◎ main Name the footer's parts", "◉─╮ Merge branch 'rows'", "│ │   M  file.go", "● │ origin/main v1.0 Rework the rows", "●─╯ Add the viewer"} {
		if rowWith(rows, want) < 0 {
			t.Errorf("no row %q:\n%s", want, strings.Join(rows, "\n"))
		}
	}
	if y := rowWith(rows, "◉─╮"); y < 0 || blameStyle(sc, y, "◉") != stLane[0] || blameStyle(sc, y, "╮") != stLane[1] {
		t.Errorf("the merge's ring and curve not in their lanes' colours")
	}
	if y := rowWith(rows, "Name the footer's parts"); y < 0 || blameStyle(sc, y, "●") != bandGround.on(stLane[0]) {
		t.Errorf("the cursor's lanes not on the band")
	}

	// Ten lanes in the docked panel's 27 cells: six shown, the others behind
	// the overflow column, no curve drawn to them.
	var d dag
	d = append(d, []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10})
	for range 10 {
		d = append(d, []int{11})
	}
	d = append(d, nil)
	w := GraphView{Open: true, Lanes: layOut(d)}
	for i := range d {
		w.Commits = append(w.Commits, GraphEntry{Hash: hashBytes(i + 1), Subject: fmt.Sprintf("commit %d", len(d)-1-i)})
		w.Rows = append(w.Rows, GraphRowRef{Commit: int32(i), File: -1})
	}
	_, rows = graphFrame(w, ScopeView{}, 208, 59)
	for _, want := range []string{"◉─┬─┬─┬─┬─╮ ┆ commit 11", "│ │ │ │ │ │ ◆ commit 4", "●─┴─┴─┴─┴─╯ ┆ commit 0"} {
		if rowWith(rows, want) < 0 {
			t.Errorf("208x59: no row %q:\n%s", want, strings.Join(rows, "\n"))
		}
	}
	if foot := lastRow(rows); !strings.Contains(foot, "lanes hidden: f widens the graph") {
		t.Errorf("the footer %q", foot)
	}
	w.Wide = true
	if _, rows = graphFrame(w, ScopeView{}, 208, 59); rowWith(rows, "◉─┬─┬─┬─┬─┬─┬─┬─┬─╮ commit 11") < 0 || strings.Contains(lastRow(rows), "lanes hidden") {
		t.Errorf("wide: every lane, no note:\n%s", strings.Join(rows, "\n"))
	}

	// Wide at 120x40: from zone A's edge, the author's and the date's
	// columns, the dates in the zone given.
	v.Wide, v.Loc = true, time.FixedZone("+0100", 3600)
	for i := range v.Commits {
		v.Commits[i].Time = 1_700_000_000
	}
	_, rows = graphFrame(v, ScopeView{}, 120, 40)
	if y := rowWith(rows, "Add the viewer"); y < 0 || !strings.HasPrefix(rows[y], "│ ●─╯ Add the viewer") || !strings.Contains(rows[y], "Ana Hubaut") || !strings.Contains(rows[y], "14/11/2023 23:13") {
		t.Errorf("wide: the row %q", rows[max(y, 0)])
	}
}

// lastRow is the frame's last row with text: the footer.
func lastRow(rows []string) string {
	for i := len(rows) - 1; i >= 0; i-- {
		if strings.TrimSpace(rows[i]) != "" {
			return rows[i]
		}
	}
	return ""
}

// graphFrameAt is graphFrame at time now.
func graphFrameAt(v GraphView, w, h int, now time.Time) (*Screen, []string) {
	s := state.New()
	s.Screen = state.ScreenDashboard
	sc := Draw(s, UI{Width: w, Height: h, Focus: "R", Graph: v, Now: now})
	return sc, strings.Split(texts(sc.lines), "\n")
}

// TestGraphFooter is criterion 62.8's footer (design D12): on a commit its
// short hash, author and when; on a file its full path and status in words,
// a rename's source; the keys after them; the exact date without the time
// of the frame.
func TestGraphFooter(t *testing.T) {
	v := graphSample()
	v.Loc = time.UTC
	for i := range v.Commits {
		v.Commits[i].Time = 1_700_000_000
	}
	now := time.Unix(1_700_000_000, 0).Add(50 * time.Hour)
	for _, c := range []struct {
		cursor int
		want   string
	}{{0, " 0000000 · Ana Hubaut · 2 days ago · ↑↓"}, {2, " internal/render/file.go · Modified · ↑↓"}, {5, " web/app.ts · Renamed from web/old.ts · ↑↓"}} {
		v.Cursor = c.cursor
		if _, rows := graphFrameAt(v, 208, 59, now); !strings.HasPrefix(lastRow(rows), c.want) {
			t.Errorf("cursor %d: the footer %q, want it to start %q", c.cursor, lastRow(rows), c.want)
		}
	}
	v.Cursor = 0
	if _, rows := graphFrameAt(v, 208, 59, time.Time{}); !strings.HasPrefix(lastRow(rows), " 0000000 · Ana Hubaut · 14/11/2023 22:13 · ") {
		t.Errorf("no time: the footer %q", lastRow(rows))
	}
	for _, c := range []struct {
		d    time.Duration
		want string
	}{{30 * time.Second, "just now"}, {5 * time.Minute, "5 minutes ago"}, {time.Hour, "1 hour ago"}, {20 * 24 * time.Hour, "2 weeks ago"},
		{90 * 24 * time.Hour, "3 months ago"}, {800 * 24 * time.Hour, "2 years ago"}, {-time.Hour, "14/11/2023 22:13"}} {
		if got := commitAgo(1_700_000_000, time.Unix(1_700_000_000, 0).Add(c.d), time.UTC); got != c.want {
			t.Errorf("%v after: %q, want %q", c.d, got, c.want)
		}
	}
}

// TestGraphCardDrawn is criterion 62.8's card as drawn (design D13): a
// commit's author, when, relative and exact, its message, cut after 20
// lines, its files, its labels, its short hash in the top edge; left of the
// docked panel beside its row, in the panel under the row where no room is
// left of it, never over the row; a file's path and status; none for a row
// out of view, nor under a popup.
func TestGraphCardDrawn(t *testing.T) {
	v := graphSample()
	v.Loc = time.UTC
	for i := range v.Commits {
		v.Commits[i].Time = 1_700_000_000
	}
	now := time.Unix(1_700_000_000, 0).Add(50 * time.Hour)
	v.Card = &GraphCard{Row: 9, Message: "Rework the rows\n\nNo CR, no last empty line.", Files: "2 files changed, +3 −1"}
	sc, rows := graphFrameAt(v, 208, 59, now)
	for _, want := range []string{"╭─ 0000000 ", "│ Ana Hubaut", "│ 2 days ago (14/11/2023 22:13)", "│ Rework the rows", "│ No CR, no last empty line.", "│ 2 files changed, +3 −1", "│ origin/main  v1.0"} {
		if rowWith(rows, want) < 0 {
			t.Errorf("208x59: no %q:\n%s", want, strings.Join(rows, "\n"))
		}
	}
	_, ry, _ := sc.findItem(GraphPanel, 10)
	if top := rowWith(rows, "╭─ 0000000 "); top != ry-1 || !strings.Contains(rows[ry], "● Rework the rows") {
		t.Errorf("the card's top on row %d, its row %d: %q", top, ry, rows[ry])
	}
	if x := strings.Index(rows[ry], "│ Ana Hubaut"); x < 0 || textWidth(rows[ry][:x])+56 > sc.Panel.X {
		t.Errorf("the card not left of the panel at %d: %q", sc.Panel.X, rows[ry])
	}

	// Compact at 120x40: in the panel, under the row.
	sc, rows = graphFrameAt(v, 120, 40, now)
	_, ry, _ = sc.findItem(GraphPanel, 10)
	if top := rowWith(rows, "╭─ 0000000 "); top != ry+1 || !strings.Contains(rows[ry], "Rework the rows") {
		t.Errorf("120x40: the card's top on row %d, its row %d", top, ry)
	}

	// A message of 25 lines: 19, then "…".
	var long []string
	for i := range 25 {
		long = append(long, fmt.Sprintf("line %d of the message", i+1))
	}
	v.Card.Message = strings.Join(long, "\n")
	if _, rows = graphFrameAt(v, 208, 59, now); rowWith(rows, "│ line 19 of the message") < 0 || rowWith(rows, "│ line 20 of") >= 0 || rowWith(rows, "│ …") < 0 {
		t.Errorf("a long message not cut after 19 lines and …:\n%s", strings.Join(rows, "\n"))
	}

	// A file's card; none out of view; none under the scope popup.
	v.Card = &GraphCard{Row: 5}
	if _, rows = graphFrameAt(v, 208, 59, now); rowWith(rows, "╭─ R ") < 0 || rowWith(rows, "│ web/app.ts") < 0 || rowWith(rows, "│ Renamed from web/old.ts") < 0 {
		t.Errorf("the file's card:\n%s", strings.Join(rows, "\n"))
	}
	v.Card, v.Top, v.Cursor = &GraphCard{Row: 0}, 9, 10
	if _, rows = graphFrameAt(v, 120, 20, now); rowWith(rows, "╭─ 0000000 ") >= 0 {
		t.Errorf("a card for a row out of view")
	}
	v.Top, v.Cursor = 0, 0
	if st := Layers(state.New(), UI{Width: 208, Height: 59, Graph: v, Scope: ScopeView{Open: true}}); st.Card {
		t.Error("the card under the scope popup")
	}
}

// TestGraphLanesRoom is design D10's room: ten lanes are shown whole where
// they leave the subject its 12 cells, hidden behind the overflow column a
// cell narrower, two at least shown however narrow.
func TestGraphLanesRoom(t *testing.T) {
	d := dag{{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}}
	for range 10 {
		d = append(d, []int{11})
	}
	d = append(d, nil)
	v := GraphView{Lanes: layOut(d)}
	for i := range d {
		v.Rows = append(v.Rows, GraphRowRef{Commit: int32(i), File: -1})
	}
	for _, c := range []struct {
		inner, cells int
		hidden       bool
	}{{32, 20, false}, {31, 16, true}, {10, 4, true}} {
		lv := viewLanes(v, 0, len(v.Rows), c.inner)
		if lv.hidden != c.hidden || len(lv.cells[0]) != c.cells {
			t.Errorf("%d cells: hidden %v, %d cells of lanes; want %v, %d", c.inner, lv.hidden, len(lv.cells[0]), c.hidden, c.cells)
		}
	}
}
