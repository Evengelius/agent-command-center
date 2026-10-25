package render

import (
	"reflect"
	"strings"
	"testing"

	"github.com/Evengelius/agent-command-center/internal/state"
)

// diffDemo is a diff of two hunks: the first replaces two lines by three
// between context lines, the second adds a line at 1,203, past four digits'
// numbers' width.
func diffDemo() DiffView {
	v := DiffView{Open: true, Path: "src/app/main.go", OldName: "index", NewName: "working tree", Added: 4, Removed: 2,
		Hunks: []DiffHunk{
			{OldStart: 1, OldLines: 4, NewStart: 1, NewLines: 5, Lines: []DiffLine{
				{' ', 1, 1, "package main"},
				{'-', 2, 0, "old one"},
				{'-', 3, 0, "old two"},
				{'+', 0, 2, "new one"},
				{'+', 0, 3, "new two"},
				{'+', 0, 4, "\tnew three"},
				{' ', 4, 5, "}"},
			}},
			{OldStart: 1200, OldLines: 3, NewStart: 1201, NewLines: 4, Lines: []DiffLine{
				{' ', 1200, 1201, "a"},
				{' ', 1201, 1202, "b"},
				{'+', 0, 1203, "added"},
				{' ', 1202, 1204, "c"},
			}},
		}}
	v.Layout = LayoutDiff(v)
	return v
}

// TestLayoutDiff is plan 44, criterion 44.8's rows (design D8): unified,
// each hunk's header then its lines, a change's first row noted; side by
// side, a removed run facing the added run after it, blank where a side
// has none; the numbers' width, the longest line's cells with its tab.
func TestLayoutDiff(t *testing.T) {
	lay := diffDemo().Layout
	uni := []DiffRow{{0, -1, -1}, {0, 0, -1}, {0, 1, -1}, {0, 2, -1}, {0, 3, -1}, {0, 4, -1}, {0, 5, -1}, {0, 6, -1},
		{1, -1, -1}, {1, 0, -1}, {1, 1, -1}, {1, 2, -1}, {1, 3, -1}}
	side := []DiffRow{{0, -1, -1}, {0, 0, 0}, {0, 1, 3}, {0, 2, 4}, {0, -1, 5}, {0, 6, 6},
		{1, -1, -1}, {1, 0, 0}, {1, 1, 1}, {1, -1, 2}, {1, 3, 3}}
	if !reflect.DeepEqual(lay.Unified, uni) || !reflect.DeepEqual(lay.UniChanges, []int{2, 11}) {
		t.Errorf("unified %v, changes %v", lay.Unified, lay.UniChanges)
	}
	if !reflect.DeepEqual(lay.Side, side) || !reflect.DeepEqual(lay.SideChanges, []int{2, 9}) {
		t.Errorf("side by side %v, changes %v", lay.Side, lay.SideChanges)
	}
	if lay.NumWidth != 4 || lay.TextWidth != 13 {
		t.Errorf("numbers %d wide, lines %d", lay.NumWidth, lay.TextWidth)
	}
	for w, want := range map[int]bool{149: false, 150: true, 220: true} {
		if DiffSideBySide(w) != want {
			t.Errorf("side by side at %d: %v", w, !want)
		}
		rows, ch := DiffRows(diffDemo(), w)
		if want && (len(rows) != len(side) || ch[0] != 2) || !want && (len(rows) != len(uni) || ch[1] != 11) {
			t.Errorf("rows at %d: %d", w, len(rows))
		}
	}
}

// TestDiffLines is criterion 44.8's lines (design D8): unified, the old
// and new numbers grey, "−" or "+", the text; a removed line on the red
// ground and an added one on the green, the whole row; context muted; a
// hunk's header in lavender; side by side, each side numbered with "│"
// between; a tab to the next multiple of 4; a long line cut with "…"; the
// shift; every row its width.
func TestDiffLines(t *testing.T) {
	v := diffDemo()
	hk := v.Hunks[0]
	for _, c := range []struct {
		r    DiffRow
		want string
	}{
		{DiffRow{0, -1, -1}, "@@ -1,4 +1,5 @@             "},
		{DiffRow{0, 0, -1}, "   1    1   package main    "},
		{DiffRow{0, 1, -1}, "   2      − old one         "},
		{DiffRow{0, 5, -1}, "        4 +     new three   "},
	} {
		l := diffUniRow(v, c.r, 28)
		if got := plainOf(l); got != c.want || l.width() != 28 {
			t.Errorf("%v: %q (%d cells), want %q", c.r, got, l.width(), c.want)
		}
	}
	if x, _ := spanOf(diffUniRow(v, DiffRow{0, -1, -1}, 28), "@@"); x.st != stTitle {
		t.Error("the hunk's header is not lavender")
	}
	del := diffUniRow(v, DiffRow{0, 1, -1}, 28)
	if x, _ := spanOf(del, "old one"); x.st != diffDelGround.on(stDiffDel) {
		t.Error("a removed line is not light red on the red ground")
	}
	if x, _ := spanOf(del, "   2"); x.st != stGrey {
		t.Error("the numbers are not grey")
	}
	add := diffText(hk.Lines[3], 0, 20)
	for _, x := range add {
		if x.st != diffAddGround.on(stDiffAdd) && x.st != diffAddGround.on(stDone) && x.st != diffAddGround.on(stNone) {
			t.Errorf("%q of an added line is off the green ground", x.text)
		}
	}
	if x, _ := spanOf(diffText(hk.Lines[0], 0, 20), "package"); x.st != stMuted {
		t.Error("context is not muted")
	}
	long := DiffLine{'+', 0, 9, "0123456789abcdefghij"}
	for _, c := range []struct {
		left, w int
		want    string
	}{{0, 12, "+ 012345678…"}, {0, 22, "+ 0123456789abcdefghij"}, {4, 12, "+ 456789abc…"}, {16, 12, "+ ghij      "}, {30, 8, "+       "}} {
		if got := plainOf(diffText(long, c.left, c.w)); got != c.want {
			t.Errorf("from %d, %d cells: %q, want %q", c.left, c.w, got, c.want)
		}
	}
	if got := plainOf(diffText(DiffLine{' ', 1, 1, "a\tb"}, 0, 10)); got != "  a   b   " {
		t.Errorf("a tab: %q", got)
	}
	s := diffSideRow(v, DiffRow{0, 2, 4}, 41)
	if got := plainOf(s); got != "   3 − old two      │   3 + new two      " || s.width() != 41 {
		t.Errorf("side by side: %q (%d cells)", got, s.width())
	}
	if got := plainOf(diffSideRow(v, DiffRow{0, -1, 5}, 41)); !strings.HasPrefix(got, strings.Repeat(" ", 20)+"│   4 +     new three") {
		t.Errorf("an added line alone: %q", got)
	}
	for w, want := range map[int]int{28: 13 - (28 - 8 - 2 - 2) + 1, 60: 0, 200: 0} {
		if got := DiffMaxLeft(v, w); got != max(want, 0) {
			t.Errorf("the most shift at %d: %d, want %d", w, got, max(want, 0))
		}
	}
}

// TestDiffView is criterion 44.8's box (design D8): "DIFF · name" with
// "+N −M"; the path and the comparison, a rename's "old → new", the other
// comparison's Tab, a conflict said unresolved; the sides' heads side by
// side; the lines in view from Top, each carrying its row; a note in their
// place; the box's columns on each screen.
func TestDiffView(t *testing.T) {
	v := diffDemo()
	v.Both = true
	text := texts(diffView(UI{Focus: "D", Diff: v}, 100, 16))
	for _, s := range []string{"DIFF · main.go", "+4 −2", "src/app/main.go · Working tree: index → working tree · Tab staged", "@@ -1,4 +1,5 @@", "+     new three"} {
		if !strings.Contains(text, s) {
			t.Errorf("%q not drawn:\n%s", s, text)
		}
	}
	v.Staged, v.Orig, v.OldName, v.NewName = true, "src/app/old.go", "HEAD", "index"
	v.Top = 9
	ls := diffView(UI{Focus: "B", Diff: v}, 160, 10)
	text = texts(ls)
	for _, s := range []string{"src/app/old.go → src/app/main.go · Staged: HEAD → index · Tab working tree", "ORIGINAL  HEAD", "MODIFIED  index", "1203 + added"} {
		if !strings.Contains(text, s) {
			t.Errorf("%q not drawn:\n%s", s, text)
		}
	}
	if strings.Contains(text, "package main") {
		t.Errorf("the rows above Top drawn:\n%s", text)
	}
	if ls[0][0].st != stFrame {
		t.Error("the frame is not grey without the focus")
	}
	for _, l := range ls {
		for _, x := range l {
			if x.ref.panel == DiffPanel && x.ref.item > 0 && strings.Contains(x.text, "added") && x.ref.item != 10 {
				t.Errorf("the added line's cells carry row %d, want 10", x.ref.item)
			}
		}
	}
	for _, n := range []DiffView{
		{Open: true, Path: "a.png", Note: "Binary file: no textual diff (1.2 KiB → 1.3 KiB)"},
		{Open: true, Path: "u.go", Conflict: "both modified", OldName: "ours", NewName: "working tree", Note: "comparing…"},
	} {
		text := texts(diffView(UI{Focus: "D", Diff: n}, 120, 8))
		if !strings.Contains(text, n.Note) || n.Conflict != "" && !strings.Contains(text, "Conflict, both modified, unresolved: ours → working tree") {
			t.Errorf("the note:\n%s", text)
		}
	}

	s := state.New()
	s.Screen = state.ScreenDashboard
	src := SourceView{Open: true}
	for _, c := range []struct {
		w, h int
		wide bool
		want Cols
	}{
		{200, 60, false, Cols{56, 106}}, {200, 60, true, Cols{0, 162}},
		{120, 40, false, Cols{36, 50}}, {120, 40, true, Cols{0, 86}}, // the panel beside it (26)
		{110, 40, false, Cols{36, 66}}, {110, 40, true, Cols{0, 102}}, // 40 columns beside it: the panel's place
		{80, 24, false, Cols{0, 80}},
	} {
		ui := UI{Width: c.w, Height: c.h, Source: src, Diff: DiffView{Open: true, Wide: c.wide}}
		if got := DiffBox(s, ui); got != c.want {
			t.Errorf("%dx%d, wide %v: %+v, want %+v", c.w, c.h, c.wide, got, c.want)
		}
		if under := PanelUnderDiff(s, ui); under != (c.w < 118) {
			t.Errorf("%dx%d, wide %v: the panel under the diff %v", c.w, c.h, c.wide, under)
		}
	}
	for _, c := range []struct {
		both bool
		w    int
		want string
	}{
		{true, 200, "↑↓ PgUp PgDn Home End scroll · ←→ shift · n N change · Tab other comparison · f wide · r compare again · Esc back · F1 keys"},
		{false, 200, "↑↓ PgUp PgDn Home End scroll · ←→ shift · n N change · f wide · r compare again · Esc back · F1 keys"},
		{true, 80, "↑↓ PgUp PgDn scroll · ←→ shift · n N change · Tab other · Esc back · F1 keys"},
		{false, 80, "↑↓ PgUp PgDn scroll · ←→ shift · n N change · Esc back · F1 keys"},
	} {
		if got := diffHints(DiffView{Both: c.both}, c.w); got != c.want {
			t.Errorf("footer, both %v, at %d: %q", c.both, c.w, got)
		}
	}
	// The side heads fill their halves exactly, no "…" at the end.
	for _, w := range []int{150, 151, 165} {
		h := diffSideHeads(DiffView{OldName: "index", NewName: "working tree"}, w)
		if got := plainOf(h); h.width() != w || strings.Contains(got, "…") {
			t.Errorf("the heads at %d: %q (%d cells)", w, got, h.width())
		}
	}
}
