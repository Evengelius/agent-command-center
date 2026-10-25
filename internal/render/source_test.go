package render

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

// sourceText is the list's rows as "kind letter path", a header as "h
// section count", a note as "n note".
func sourceText(rows []SourceRow) []string {
	var out []string
	for _, r := range rows {
		switch r.Kind {
		case 'h':
			out = append(out, fmt.Sprintf("h %s %d", r.Section, r.Count))
		case 'n':
			out = append(out, "n "+r.Note)
		case 'b':
			out = append(out, "b")
		default:
			s := fmt.Sprintf("%c %c %s", r.Kind, r.Letter, r.Path)
			if r.Orig != "" {
				s += " ← " + r.Orig
			}
			out = append(out, s)
		}
	}
	return out
}

// sourceDemo is a status of every kind: unstaged, staged, both, added,
// deleted on either side, a rename changed again, a conflict, an untracked
// file and folder, an ignored folder, a type change.
func sourceDemo() SourceView {
	return SourceView{
		Entries: []SourceEntry{
			{X: ' ', Y: 'M', Path: "src/main.go"},
			{X: 'M', Y: ' ', Path: "README.md"},
			{X: 'M', Y: 'M', Path: "both.go"},
			{X: 'A', Y: ' ', Path: "new.go"},
			{X: 'D', Y: ' ', Path: "gone.go"},
			{X: ' ', Y: 'D', Path: "lost.go"},
			{X: 'R', Y: 'M', Path: "lib/new name.go", Orig: "lib/old.go"},
			{X: 'U', Y: 'U', Path: "conflict.go"},
			{X: '?', Y: '?', Path: "notes.txt"},
			{X: '?', Y: '?', Path: "web/"},
			{X: '!', Y: '!', Path: "build/"},
			{X: 'T', Y: ' ', Path: "link"},
		},
		Opened:  map[string]bool{"web/": true},
		Folders: map[string]*SourceFolder{"web/": {Files: []string{"web/a.js", "web/b.js"}, Truncated: true}},
	}
}

// TestSourceRows is plan 44, criteria 44.2 and 44.4 (design D4): MERGE
// CHANGES, CHANGES and STAGED CHANGES in that order, each with its count; a
// file staged and changed again in both; each row's letter, its section's
// (the index's staged, the working tree's in CHANGES, U untracked, ! a
// conflict); a rename's source on its staged row only; an untracked folder
// opened on its files, a cut list said; an ignored path never a row; a
// folded section's files hidden; the seven conflicts named.
func TestSourceRows(t *testing.T) {
	v := sourceDemo()
	want := []string{
		"b", "h merge 1", "b", "f ! conflict.go",
		"b", "h changes 6", "b", "f M src/main.go", "f M both.go", "f D lost.go", "f M lib/new name.go", "f U notes.txt", "d U web/", "u U web/a.js", "u U web/b.js", "n first 2 files",
		"b", "h staged 6", "b", "f M README.md", "f M both.go", "f A new.go", "f D gone.go", "f R lib/new name.go ← lib/old.go", "f T link",
	}
	rows := SourceRows(v)
	if got := sourceText(rows); !reflect.DeepEqual(got, want) {
		t.Fatalf("rows\n%q\nwant\n%q", got, want)
	}
	keys := map[string]bool{}
	for _, r := range rows {
		if keys[r.Key] {
			t.Errorf("key %q twice", r.Key)
		}
		keys[r.Key] = true
		if r.Selectable() == (r.Kind == 'n' || r.Kind == 'b') {
			t.Errorf("%q: selectable %v", r.Key, r.Selectable())
		}
	}
	if r := rows[8]; r.Key != "f:changes:both.go" || r.Entry != 2 || rows[20].Key != "f:staged:both.go" || rows[20].Entry != 2 {
		t.Errorf("both.go's rows: %+v, %+v", r, rows[20])
	}

	v.Folded = map[string]bool{SectionChanges: true}
	if got := sourceText(SourceRows(v)); !reflect.DeepEqual(got, append([]string{"b", "h merge 1", "b", "f ! conflict.go", "b", "h changes 6"}, want[16:]...)) {
		t.Errorf("CHANGES folded: %q", got)
	}
	v.Folded = nil
	for _, c := range []struct {
		d    *SourceFolder
		want []string
	}{
		{nil, []string{"n reading…"}},
		{&SourceFolder{Note: "Access is denied.", Failed: true}, []string{"n Access is denied."}},
		{&SourceFolder{Files: []string{"web/a.js"}}, []string{"u U web/a.js"}},
	} {
		v.Folders = map[string]*SourceFolder{}
		if c.d != nil {
			v.Folders["web/"] = c.d
		}
		got := sourceText(SourceRows(v))
		if i := slicesIndex(got, "d U web/"); i < 0 || !reflect.DeepEqual(got[i+1:i+1+len(c.want)], c.want) {
			t.Errorf("the folder %+v: %q", c.d, got)
		}
	}
	v.Opened = nil
	if got := sourceText(SourceRows(v)); slicesIndex(got, "n reading…") >= 0 || slicesIndex(got, "d U web/") < 0 {
		t.Errorf("the folder closed: %q", got)
	}

	for xy, name := range map[string]string{"UU": "both modified", "AA": "both added", "DD": "both deleted", "AU": "added by us",
		"UA": "added by them", "DU": "deleted by us", "UD": "deleted by them", "MM": "", "A ": "", "??": ""} {
		e := SourceEntry{X: xy[0], Y: xy[1], Path: "f"}
		if got := Conflict(e); got != name {
			t.Errorf("%q: %q, want %q", xy, got, name)
		}
		if name != "" && !reflect.DeepEqual(Sections(e), []string{SectionMerge}) {
			t.Errorf("%q: sections %q", xy, Sections(e))
		}
	}
	if got := Sections(SourceEntry{X: '!', Y: '!', Path: "x/"}); got != nil {
		t.Errorf("an ignored path: %q", got)
	}
}

// slicesIndex is the index of s in l, -1 when absent.
func slicesIndex(l []string, s string) int {
	for i, x := range l {
		if x == s {
			return i
		}
	}
	return -1
}

// TestSourceCursor is criterion 44.4's cursor: on its row by key, else the
// first row it stops on; none in a list of notes.
func TestSourceCursor(t *testing.T) {
	rows := SourceRows(sourceDemo())
	for key, want := range map[string]int{"f:staged:new.go": 21, "h:changes": 5, "gone": 1, "": 1, "n:web/": 1, "b:0": 1} {
		if got := SourceCursor(rows, key); got != want {
			t.Errorf("%q: row %d, want %d", key, got, want)
		}
	}
	if got := SourceCursor([]SourceRow{{Kind: 'n', Key: "n:x"}}, "n:x"); got != -1 {
		t.Errorf("notes only: %d", got)
	}
}

// TestSourceRowLooks is criterion 44.4's rows (design D4): a header, "▾"
// or "▸", its name bold in the title colour, its count at the right; a file
// two cells in, its letter in its colour, its directory grey, its name; a
// rename's source grey after "←"; a folder's "›" or "⌄"; an opened folder's
// file one step deeper; every row its width; the cursor's on the band, the
// hovered one on its ground.
func TestSourceRowLooks(t *testing.T) {
	for _, c := range []struct {
		r    SourceRow
		want string
	}{
		{SourceRow{Kind: 'h', Section: SectionStaged, Open: true, Count: 12}, "▾ STAGED CHANGES     12"},
		{SourceRow{Kind: 'h', Section: SectionMerge, Count: 1}, "▸ MERGE CHANGES       1"},
		{SourceRow{Kind: 'f', Letter: 'M', Path: "src/app/main.go"}, "   M  src/app/main.go  "},
		{SourceRow{Kind: 'f', Letter: 'R', Path: "b.go", Orig: "a.go"}, "   R  b.go ← a.go      "},
		{SourceRow{Kind: 'd', Letter: 'U', Path: "web/"}, "   U  › web/           "},
		{SourceRow{Kind: 'd', Letter: 'U', Path: "web/", Open: true}, "   U  ⌄ web/           "},
		{SourceRow{Kind: 'u', Letter: 'U', Path: "web/a.js"}, "     U  web/a.js       "},
		{SourceRow{Kind: 'n', Note: "first 2 files"}, "     first 2 files     "},
		{SourceRow{Kind: 'b'}, "                       "},
	} {
		l := sourceRow(c.r, false, false, 23)
		if got := plainOf(l); got != c.want || l.width() != 23 {
			t.Errorf("%+v: %q (%d cells), want %q", c.r, got, l.width(), c.want)
		}
	}
	// A path wider than the row: its leading folders give way to "…/", the
	// name kept in view, cut only when it alone is too wide.
	for w, want := range map[int]string{
		30: "   M  internal/state/reduce.go",
		23: "   M  …/state/reduce.go",
		20: "   M  …/reduce.go   ",
		12: "   M  reduc…",
	} {
		l := sourceRow(SourceRow{Kind: 'f', Letter: 'M', Path: "internal/state/reduce.go"}, false, false, w)
		if got := plainOf(l); got != want || l.width() != w {
			t.Errorf("a long path in %d cells: %q, want %q", w, got, want)
		}
	}
	// A rename too wide: its source is cut, never the name.
	for w, want := range map[int]string{
		50: "   R  internal/util/text.go ← internal/util/strin…",
		27: "   R  …/util/text.go ← int…",
	} {
		l := sourceRow(SourceRow{Kind: 'f', Letter: 'R', Path: "internal/util/text.go", Orig: "internal/util/strings.go"}, false, false, w)
		if got := plainOf(l); got != want || l.width() != w {
			t.Errorf("a rename in %d cells: %q, want %q", w, got, want)
		}
	}
	l := sourceRow(SourceRow{Kind: 'f', Letter: 'D', Path: "src/gone.go"}, false, false, 30)
	if x, _ := spanOf(l, "D"); x.st != stFailed {
		t.Error("D is not red")
	}
	if x, _ := spanOf(l, "src/"); x.st != stGrey {
		t.Error("the directory is not grey")
	}
	if x, _ := spanOf(l, "gone.go"); x.st != stExpFileName {
		t.Error("the name is not in the file colour")
	}
	for c, st := range map[byte]any{'M': stProgress, 'A': stDone, 'U': stDone, 'R': stDone, 'C': stDone, 'D': stFailed, '!': stFailed, 'T': stInfo} {
		if letterStyle(c) != st {
			t.Errorf("%c: wrong colour", c)
		}
	}
	h := sourceRow(SourceRow{Kind: 'h', Section: SectionChanges, Open: true, Count: 3}, false, false, 30)
	if x, _ := spanOf(h, "CHANGES"); x.st != stTitleBold {
		t.Error("the header's name is not bold in the title colour")
	}
	for _, c := range []struct {
		cur, hover bool
		g          *ground
	}{{true, false, bandGround}, {false, true, sourceHoverGround}, {true, true, bandGround}} {
		l := sourceRow(SourceRow{Kind: 'f', Letter: 'M', Path: "a.go"}, c.cur, c.hover, 20)
		for _, x := range l {
			if x.st != c.g.on(stNone) && x.st != c.g.on(stProgress) && x.st != c.g.on(stExpFileName) {
				t.Errorf("cursor %v, hover %v: %q not on its ground", c.cur, c.hover, x.text)
			}
		}
	}
}

// TestSourcePanel is criterion 44.4's panel and 44.11 (designs D4 and
// D11): "SOURCE CONTROL" with its "⌄", a rule, the repository's row, the
// list scrolled to keep the cursor in view, each list row carrying its index
// plus one; a note in the list's place; "No changes" for none; its frame in
// the title colour with the focus.
func TestSourcePanel(t *testing.T) {
	v := sourceDemo()
	v.Open, v.Repo, v.Branch, v.Cursor = true, "agent-command-center", "main", "f:staged:gone.go"
	ui := UI{Width: 200, Height: 60, Focus: "R", Source: v}
	ls := sourcePanel(ui, 36, 12)
	text := texts(ls)
	for _, s := range []string{"SOURCE CONTROL", "⌄", "agent-command-center · main", "STAGED CHANGES", "D  gone.go"} {
		if !strings.Contains(text, s) {
			t.Errorf("%q not drawn:\n%s", s, text)
		}
	}
	if len(ls) != 12 || strings.Contains(text, "MERGE CHANGES") {
		t.Errorf("%d rows, the top not scrolled off:\n%s", len(ls), text)
	}
	if ls[0][0].st != stTitle {
		t.Error("the frame is not in the title colour with the focus")
	}
	rows := SourceRows(v)
	for _, l := range ls {
		for _, x := range l {
			if x.ref.panel == SourcePanel && x.ref.item > 0 && strings.Contains(x.text, "gone.go") && rows[x.ref.item-1].Path != "gone.go" {
				t.Errorf("gone.go's cells carry row %d", x.ref.item)
			}
		}
	}
	for _, c := range []struct {
		v    SourceView
		want string
	}{
		{SourceView{Open: true, Note: "No Git repository found for this project."}, "No Git repository found for this project."},
		{SourceView{Open: true, Repo: "demo", Branch: "main"}, "No changes"},
		{SourceView{Open: true, Repo: "demo", Note: "git: index locked", Failed: true}, "git: index locked"},
	} {
		ls := sourcePanel(UI{Width: 200, Height: 60, Focus: "B", Source: c.v}, 31, 12) // 208 columns' panel
		var words []string
		for _, l := range ls[4:] {
			words = append(words, strings.Fields(strings.Trim(plainOf(l), "│"))...)
		}
		if got := strings.Join(words, " "); !strings.HasPrefix(got, c.want) {
			t.Errorf("%q not drawn whole:\n%s", c.want, texts(ls))
		}
		if ls[0][0].st != stFrame {
			t.Error("the frame is not grey without the focus")
		}
	}
	if got := sourceKeys(200); !strings.HasPrefix(got, "↑↓ PgUp PgDn Home End move") {
		t.Errorf("footer at 200: %q", got)
	}
	if got := sourceKeys(60); textWidth(got) >= 60 {
		t.Errorf("footer at 60: %q", got)
	}
}
