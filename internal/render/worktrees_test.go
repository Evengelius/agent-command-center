package render

import (
	"strings"
	"testing"

	"github.com/Evengelius/agent-command-center/internal/state"
)

// TestFootersFit is criteria 2.4 and 3.7 of plan 16: every footer the
// startup screen and the Worktrees view can show fits within 79 cells below
// 100 columns, and within the terminal from 100.
func TestFootersFit(t *testing.T) {
	fits := func(what, footer string, w int) {
		t.Helper()
		limit := 79
		if w >= 100 {
			limit = w
		}
		if n := textWidth(footer); n > limit {
			t.Errorf("%s at %d columns: %d cells: %q", what, w, n, footer)
		}
	}
	for _, w := range []int{80, 99, 100, 200} {
		// The home screen's footer, the same in every field (plan 51, design D8).
		fits("home screen", strings.TrimRight(rowsText([]line{homeGridFooter(UI{}, w)}), " \n"), w)
	}
	rows := []WorktreeRow{{Branch: "main", Main: true}}
	for _, v := range []WorktreesView{
		{Startup: true, Rows: rows},
		{Rows: rows},
		{Startup: true},
		{Mode: 'n'},
		{Mode: 'm', Rows: rows},
		{Mode: 'x', Rows: rows},
	} {
		if n := textWidth(worktreesKeys(v)); n > 79 {
			t.Errorf("worktrees footer, mode %q startup %v: %d cells: %q", v.Mode, v.Startup, n, worktreesKeys(v))
		}
	}
	ls := worktreeDiff(state.New(), UI{Width: 80, Height: 24, Worktrees: WorktreesView{Mode: 'd', Diff: []string{"+a"}}})
	if n := ls[len(ls)-1].width(); n > 79 {
		t.Errorf("diff footer: %d cells", n)
	}
	// A tab in a diff is four spaces, as wrap draws it.
	var got string
	for _, s := range diffRow("+\tx", 80) {
		got += s.text
	}
	if got = strings.TrimRight(got, " "); got != " +    x" {
		t.Errorf("tab drawn as %q", got)
	}
}
