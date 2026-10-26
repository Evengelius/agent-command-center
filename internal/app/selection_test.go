package app

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"testing/synctest"

	tea "charm.land/bubbletea/v2"

	"github.com/Evengelius/agent-command-center/internal/input"
	"github.com/Evengelius/agent-command-center/internal/render"
)

// Plan 50 (brief amendment 51): Source Control's selection, Ctrl+click, and
// the commit of the selected files alone; git's answers from the test
// double, and in TestSelectionGit from git on a temporary repository.

// rowItem is the panel's item of the row with key: its index plus one.
func rowItem(t *testing.T, h *harness, key string) int {
	t.Helper()
	rows := render.SourceRows(h.m.ui.Source)
	for i, r := range rows {
		if r.Key == key {
			return i + 1
		}
	}
	t.Fatalf("no row %q among %+v", key, rows)
	return 0
}

// ctrlClick is a left click with Ctrl on the panel's item.
func ctrlClick(t *testing.T, h *harness, item int) {
	t.Helper()
	x, y := itemCell(t, h, render.SourcePanel, item)
	mouseAt(t, h, tea.MouseClickMsg{X: x, Y: y, Button: tea.MouseLeft, Mod: tea.ModCtrl})
	mouseAt(t, h, tea.MouseReleaseMsg{X: x, Y: y, Button: tea.MouseLeft, Mod: tea.ModCtrl})
}

// plainClick is a left click on the panel's item.
func plainClick(t *testing.T, h *harness, item int) {
	t.Helper()
	x, y := itemCell(t, h, render.SourcePanel, item)
	mouseClick(t, h, x, y)
}

// selectedPaths is the selection, sorted.
func selectedPaths(h *harness) []string {
	var ps []string
	for p := range h.m.ui.Source.Selected {
		ps = append(ps, p)
	}
	slices.Sort(ps)
	return ps
}

// TestSelectionClicks is criterion 50.3 (designs D1 to D4): Ctrl+click
// toggles a file, the first one selected opening as a plain click does,
// the others leaving the cursor and the diff; deselecting the active file
// changes nothing else; a plain click on a file clears the selection and
// shows its diff, one on a header, a folder or the field keeps it, and so
// does Enter (D4, as the owner changed it); elsewhere Ctrl+click does
// nothing; Esc clears; a status read drops what is no longer a change; a
// Ctrl+click is no click of a double click.
func TestSelectionClicks(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, w := commitRig(t)
		state := func() string {
			v, d := h.m.ui.Source, h.m.ui.Diff
			return strings.Join(selectedPaths(h), ",") + " | " + v.Cursor + " | " + d.Path + " " + map[bool]string{true: "staged", false: "unstaged"}[d.Staged] +
				" | " + h.m.ui.Focus + " | " + v.Place
		}
		main := rowItem(t, h, "f:changes:main.go")
		ctrlClick(t, h, main)
		if got, want := state(), "main.go | f:changes:main.go | main.go unstaged | D | "; got != want || !h.m.ui.Diff.Open {
			t.Fatalf("the first Ctrl+click: %q, want %q (diff open %v)", got, want, h.m.ui.Diff.Open)
		}
		ctrlClick(t, h, rowItem(t, h, "f:staged:docs/readme.md"))
		if got, want := state(), "docs/readme.md,main.go | f:changes:main.go | main.go unstaged | R | "; got != want {
			t.Fatalf("a second: %q, want %q", got, want)
		}
		// A partly staged file is one path on two rows (D1).
		ctrlClick(t, h, rowItem(t, h, "f:changes:src/Main.java"))
		if got := selectedPaths(h); !slices.Equal(got, []string{"docs/readme.md", "main.go", "src/Main.java"}) {
			t.Fatalf("after src/Main.java: %q", got)
		}
		ctrlClick(t, h, rowItem(t, h, "f:staged:src/Main.java"))
		if got := selectedPaths(h); !slices.Equal(got, []string{"docs/readme.md", "main.go"}) {
			t.Fatalf("its other row deselects it: %q", got)
		}
		// The active file deselected: its diff and the cursor stay (D3).
		ctrlClick(t, h, main)
		if got, want := state(), "docs/readme.md | f:changes:main.go | main.go unstaged | R | "; got != want || !h.m.ui.Diff.Open {
			t.Fatalf("the active file deselected: %q, want %q", got, want)
		}
		// A plain click on a file clears the selection and shows that file's
		// diff (D4, as the owner changed it).
		plainClick(t, h, rowItem(t, h, "f:changes:gone.txt"))
		if got, want := state(), " | f:changes:gone.txt | gone.txt unstaged | D | "; got != want || !h.m.ui.Diff.Open {
			t.Fatalf("a plain click on a file: %q, want %q", got, want)
		}
		ctrlClick(t, h, rowItem(t, h, "f:staged:docs/readme.md")) // a selection again, its diff open
		if got, want := state(), "docs/readme.md | f:staged:docs/readme.md | docs/readme.md staged | D | "; got != want {
			t.Fatalf("a new selection: %q, want %q", got, want)
		}
		// Elsewhere a Ctrl+click does nothing (D2): a header, an untracked
		// folder, a blank row, the message field, the button.
		blank := 0
		for i, r := range render.SourceRows(h.m.ui.Source) {
			if r.Kind == 'b' {
				blank = i + 1
				break
			}
		}
		folded := len(h.m.ui.Source.Folded)
		for _, item := range []int{rowItem(t, h, "h:changes"), rowItem(t, h, "f:changes:web/"), blank, render.ItemMsg, render.ItemCommitMain} {
			before := state()
			ctrlClick(t, h, item)
			if got := state(); got != before || len(h.m.ui.Source.Folded) != folded || h.m.ui.Source.Opened["web/"] || h.m.op != nil || len(w.logged("")) != 0 {
				t.Errorf("a Ctrl+click on item %d: %q, want %q; folded %v, opened %v, op %v, writes %q", item, got, before,
					h.m.ui.Source.Folded, h.m.ui.Source.Opened, h.m.op != nil, w.logged(""))
			}
		}
		// A plain click on a header (folding it, then unfolding it), on a
		// folder (opening it) or on the field keeps the selection (D4).
		for _, key := range []string{"h:staged", "h:staged", "f:changes:web/"} {
			plainClick(t, h, rowItem(t, h, key))
			h.settle()
			if got := selectedPaths(h); !slices.Equal(got, []string{"docs/readme.md"}) {
				t.Fatalf("a plain click on %s: selection %q", key, got)
			}
		}
		clickItem(t, h, render.ItemMsg, 0, 0)
		if got := selectedPaths(h); !slices.Equal(got, []string{"docs/readme.md"}) || h.m.ui.Source.Place != render.PlaceMsg {
			t.Fatalf("a click on the field: selection %q, place %q", got, h.m.ui.Source.Place)
		}
		// A folder's file (D1): the folder opened above, its file selected.
		ctrlClick(t, h, rowItem(t, h, "u:web/app.ts"))
		if got := selectedPaths(h); !slices.Equal(got, []string{"docs/readme.md", "web/app.ts"}) {
			t.Fatalf("a folder's file: %q", got)
		}
		// Enter keeps the selection (D4); the diff's own Esc puts the keys back
		// on the list; there Esc clears the selection and nothing else; then Esc
		// is today's.
		h.send(keyOf(tea.KeyDown)) // the field to the button
		h.send(keyOf(tea.KeyDown)) // the button to the list
		sourceTo(t, h, "f:changes:main.go")
		h.send(keyOf(tea.KeyEnter))
		if got := selectedPaths(h); !slices.Equal(got, []string{"docs/readme.md", "web/app.ts"}) || h.m.ui.Focus != "D" {
			t.Fatalf("Enter on the list: selection %q, focus %q", got, h.m.ui.Focus)
		}
		h.send(keyOf(tea.KeyEscape))
		if h.m.ui.Focus != "R" || h.m.ui.Diff.Open || len(h.m.ui.Source.Selected) != 2 {
			t.Fatalf("the diff's Esc: focus %q, diff open %v, selection %q", h.m.ui.Focus, h.m.ui.Diff.Open, selectedPaths(h))
		}
		cursor := h.m.ui.Source.Cursor
		h.send(keyOf(tea.KeyEscape))
		if v := h.m.ui.Source; len(v.Selected) != 0 || h.m.ui.Focus != "R" || v.Cursor != cursor || h.m.ui.Diff.Open {
			t.Fatalf("Esc on the list: selection %q, focus %q, cursor %q, diff open %v", selectedPaths(h), h.m.ui.Focus, v.Cursor, h.m.ui.Diff.Open)
		}
		h.send(keyOf(tea.KeyEscape))
		if h.m.ui.Focus != "B" {
			t.Fatalf("Esc with no selection: focus %q, want B", h.m.ui.Focus)
		}
		// A status read drops a file no longer a change (D1).
		ctrlClick(t, h, rowItem(t, h, "f:staged:docs/readme.md"))
		ctrlClick(t, h, rowItem(t, h, "f:changes:main.go"))
		w.set(func(f *fakeGit) {
			f.snap.Entries = slices.DeleteFunc(f.snap.Entries, func(e input.GitEntry) bool { return e.Path == "docs/readme.md" })
		})
		refresh(h)
		if got := selectedPaths(h); !slices.Equal(got, []string{"main.go"}) {
			t.Fatalf("after a status read without docs/readme.md: %q", got)
		}
		// A plain click, a Ctrl+click and a plain click on one cell: no double
		// click (D2), the clock standing still in the bubble.
		gone := rowItem(t, h, "f:changes:gone.txt")
		plainClick(t, h, gone)
		ctrlClick(t, h, gone)
		plainClick(t, h, gone)
		if h.m.click.n != 1 {
			t.Errorf("clicks counted %d after the Ctrl+click, want 1", h.m.click.n)
		}
	})
}

// TestSelectionCommit is criteria 50.6 and 50.9, and D6, D8 and D9, over
// the double: the button's label and action; a selection of staged files
// committed alone, its field and selection cleared; files not staged, the
// question, Cancel doing nothing, Stage and commit checking, staging and
// committing; refusals before any write; Commit & Push.
func TestSelectionCommit(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, w := commitRig(t)
		readme := rowItem(t, h, "f:staged:docs/readme.md")
		ctrlClick(t, h, readme)
		w.set(func(f *fakeGit) { f.snap.Branch.Ahead = 2 })
		refresh(h)
		if label, check := render.ButtonLabel(h.m.ui.Source); label != "Commit 1 file" || !check || h.m.ui.Source.Action != render.ActionCommit || h.m.ui.Source.Dim {
			t.Fatalf("the button with a selection, the branch ahead: %q %v, action %q, dim %v", label, check, h.m.ui.Source.Action, h.m.ui.Source.Dim)
		}
		typeMessage(t, h, "Readme")
		clickItem(t, h, render.ItemCommitMain, 0, 0)
		if got := w.logged(""); !slices.Equal(got, []string{"commit-sel Readme docs/readme.md staged "}) {
			t.Fatalf("writes %q", got)
		}
		if v := h.m.ui.Source; v.Notice != "Committed abc0001." || len(v.Selected) != 0 || h.m.msg.Text() != "" {
			t.Fatalf("after the commit: note %q, selection %q, field %q", v.Notice, selectedPaths(h), h.m.msg.Text())
		}

		// Two files not staged: the question, with their number; Cancel does
		// nothing; Stage and commit checks, stages them, then commits.
		ctrlClick(t, h, rowItem(t, h, "f:changes:main.go"))
		ctrlClick(t, h, rowItem(t, h, "f:changes:gone.txt"))
		typeMessage(t, h, "Two")
		clickItem(t, h, render.ItemCommitMain, 0, 0)
		if v := h.m.ui.Source; v.Ask != 2 || v.AskSel != 2 || render.AskSelText(v.Ask, v.AskSel, v.AskPush) != "Stage the 2 selected files and commit them?" ||
			!strings.Contains(h.frame(), "Stage the 2 selected") || !strings.Contains(h.frame(), "Stage and commit") {
			t.Fatalf("the question: ask %d, sel %d\n%s", v.Ask, v.AskSel, h.frame())
		}
		clickItem(t, h, render.ItemAskNo, 0, 0)
		if got := w.logged(""); len(got) != 1 || len(h.m.ui.Source.Selected) != 2 {
			t.Fatalf("Cancel: writes %q, selection %q", got, selectedPaths(h))
		}
		clickItem(t, h, render.ItemCommitMain, 0, 0)
		clickItem(t, h, render.ItemAskYes, 0, 0)
		if got := w.logged("")[1:]; !slices.Equal(got, []string{"check gone.txt,main.go", "stage gone.txt,main.go", "commit-sel Two gone.txt,main.go staged gone.txt,main.go"}) {
			t.Fatalf("writes %q", got)
		}
		if v := h.m.ui.Source; v.Notice != "Committed abc0002." || len(v.Selected) != 0 {
			t.Fatalf("after the commit: note %q, selection %q", v.Notice, selectedPaths(h))
		}

		// Refusals before any write: a partly staged file at once, from the
		// panel's status; an empty field; the check's refusal before the
		// staging (D7).
		n := len(w.logged(""))
		ctrlClick(t, h, rowItem(t, h, "f:changes:src/Main.java"))
		typeMessage(t, h, "Part")
		clickItem(t, h, render.ItemCommitMain, 0, 0)
		want := "src/Main.java has staged and unstaged changes, and a selection commits whole files: leave it out of the selection, or stage the rest in a terminal."
		if v := h.m.ui.Source; v.Notice != want || v.NoticeKind != render.NoticeCheck || v.Ask != 0 || len(w.logged("")) != n {
			t.Fatalf("a partly staged file: note %q, ask %d, writes %q", v.Notice, v.Ask, w.logged("")[n:])
		}
		ctrlClick(t, h, rowItem(t, h, "f:changes:src/Main.java")) // deselected
		w.set(func(f *fakeGit) {
			f.snap.Entries = append(f.snap.Entries, input.GitEntry{X: ' ', Y: 'M', Path: "c.go", RepoPath: "c.go"})
		})
		refresh(h)
		ctrlClick(t, h, rowItem(t, h, "f:changes:c.go"))
		clickItem(t, h, render.ItemMsg, 0, 0) // the field emptied as the owner would
		h.send(key('a', tea.ModCtrl))
		h.send(keyOf(tea.KeyBackspace))
		if h.m.msg.Text() != "" {
			t.Fatalf("the field still holds %q", h.m.msg.Text())
		}
		clickItem(t, h, render.ItemCommitMain, 0, 0)
		if v := h.m.ui.Source; v.Notice != "Enter a commit message." || len(w.logged("")) != n {
			t.Fatalf("an empty field: note %q, writes %q", v.Notice, w.logged("")[n:])
		}
		w.checkErr = input.Refusal{Kind: "merge", Msg: "A merge is in progress: commit it whole, without a selection."}
		typeMessage(t, h, "C")
		clickItem(t, h, render.ItemCommitMain, 0, 0)
		clickItem(t, h, render.ItemAskYes, 0, 0)
		if got := w.logged("")[n:]; !slices.Equal(got, []string{"check c.go"}) {
			t.Fatalf("a refused check: writes %q", got)
		}
		if v := h.m.ui.Source; v.Notice != w.checkErr.Error() || !slices.Equal(selectedPaths(h), []string{"c.go"}) || h.m.msg.Text() != "C" {
			t.Fatalf("a refused check: note %q, selection %q, field %q", v.Notice, selectedPaths(h), h.m.msg.Text())
		}
		w.checkErr = nil

		// Commit & Push: the destination, the selection, the push (50.9).
		ctrlClick(t, h, rowItem(t, h, "f:changes:c.go")) // deselected
		w.set(func(f *fakeGit) {
			f.snap.Entries = append(f.snap.Entries, input.GitEntry{X: 'A', Y: ' ', Path: "d.go", RepoPath: "d.go"})
		})
		refresh(h)
		ctrlClick(t, h, rowItem(t, h, "f:staged:d.go"))
		clickItem(t, h, render.ItemCommitMore, 0, 0)
		clickItem(t, h, render.ItemCommitMenu, 3, 1)
		if got := w.logged("")[n+1:]; !slices.Equal(got, []string{"target", "commit-sel C d.go staged ", "push"}) {
			t.Fatalf("Commit & Push: writes %q", got)
		}
		if v := h.m.ui.Source; v.Notice != "Committed abc0003 and pushed to origin/main." || len(v.Selected) != 0 {
			t.Fatalf("Commit & Push: note %q, selection %q", v.Notice, selectedPaths(h))
		}
	})
}

// TestSelectionClearedByCommit is D9: a commit made from the selection
// clears it, though a file of it is a change again once committed (edited
// on disk meanwhile), which a status read alone would keep selected.
func TestSelectionClearedByCommit(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, w := commitRig(t)
		ctrlClick(t, h, rowItem(t, h, "f:staged:docs/readme.md"))
		w.relisted = []input.GitEntry{{X: ' ', Y: 'M', Path: "docs/readme.md", RepoPath: "docs/readme.md"}}
		typeMessage(t, h, "Readme")
		clickItem(t, h, render.ItemCommitMain, 0, 0)
		if got := w.logged(""); !slices.Equal(got, []string{"commit-sel Readme docs/readme.md staged "}) {
			t.Fatalf("writes %q", got)
		}
		rowItem(t, h, "f:changes:docs/readme.md") // listed again, unstaged
		if v := h.m.ui.Source; v.Notice != "Committed abc0001." || len(v.Selected) != 0 {
			t.Fatalf("after the commit: note %q, selection %q", v.Notice, selectedPaths(h))
		}
	})
}

// TestSelectionWholeIndex is criterion 50.14 (D4 as the owner changed it,
// amendment 51): a selection cleared by a plain click on a file, that
// file's diff shown, Commit and Commit & Push run plan 45's whole-index
// commit, every staged entry expected, never the selection's.
func TestSelectionWholeIndex(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, w := commitRig(t)
		ctrlClick(t, h, rowItem(t, h, "f:staged:docs/readme.md"))
		ctrlClick(t, h, rowItem(t, h, "f:changes:main.go"))
		plainClick(t, h, rowItem(t, h, "f:changes:gone.txt"))
		if v := h.m.ui.Source; len(v.Selected) != 0 || !h.m.ui.Diff.Open || h.m.ui.Diff.Path != "gone.txt" {
			t.Fatalf("a plain click on a file: selection %q, diff %q open %v", selectedPaths(h), h.m.ui.Diff.Path, h.m.ui.Diff.Open)
		}
		if label, _ := render.ButtonLabel(h.m.ui.Source); label != "Commit" {
			t.Fatalf("the button %q, want Commit", label)
		}
		typeMessage(t, h, "Whole")
		clickItem(t, h, render.ItemCommitMain, 0, 0)
		if got := w.logged(""); !slices.Equal(got, []string{"commit Whole"}) {
			t.Fatalf("writes %q, want plan 45's commit", got)
		}
		var expected []string
		for _, e := range w.expects[0] {
			expected = append(expected, e.Path)
		}
		if !slices.Equal(expected, []string{"src/Main.java", "docs/readme.md"}) {
			t.Errorf("the commit expects %q, want every staged entry", expected)
		}
		// Commit & Push, none selected, another file's diff shown: the
		// destination, the whole index, the push.
		w.set(func(f *fakeGit) {
			f.snap.Entries = append(f.snap.Entries, input.GitEntry{X: 'A', Y: ' ', Path: "e.go", RepoPath: "e.go"})
		})
		refresh(h)
		plainClick(t, h, rowItem(t, h, "f:changes:main.go"))
		typeMessage(t, h, "Whole and push")
		clickItem(t, h, render.ItemCommitMore, 0, 0)
		clickItem(t, h, render.ItemCommitMenu, 3, 1)
		if got := w.logged("")[1:]; !slices.Equal(got, []string{"target", "commit Whole and push", "push"}) {
			t.Fatalf("Commit & Push writes %q", got)
		}
	})
}

// TestSelectionFailures is D7's commit other than the selection and D8's
// failure after the question's staging, over the double: the Unexpected
// told, its commit kept and not pushed, the field and the selection
// cleared; the failure's note git's words, then that the files stay
// staged, for the selection's question and for plan 45's, the field and
// the selection kept, a retry asking nothing.
func TestSelectionFailures(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, w := commitRig(t)
		w.unexpected = "it also holds x.go, which was not selected"
		ctrlClick(t, h, rowItem(t, h, "f:staged:docs/readme.md"))
		typeMessage(t, h, "Odd")
		clickItem(t, h, render.ItemCommitMore, 0, 0)
		clickItem(t, h, render.ItemCommitMenu, 3, 1)
		if got := w.logged(""); !slices.Equal(got, []string{"target", "commit-sel Odd docs/readme.md staged "}) {
			t.Fatalf("an Unexpected: writes %q, want no push", got)
		}
		want := "Committed abc0001, but it also holds x.go, which was not selected: check it in a terminal."
		if v := h.m.ui.Source; v.Notice != want || v.NoticeKind != render.NoticeFailed || len(v.Selected) != 0 || h.m.msg.Text() != "" {
			t.Fatalf("an Unexpected: note %q (%c), selection %q, field %q", v.Notice, v.NoticeKind, selectedPaths(h), h.m.msg.Text())
		}
		w.unexpected = ""

		// The selection's question, then a refusing hook (D8).
		w.commitErr = errors.New("hook says no")
		ctrlClick(t, h, rowItem(t, h, "f:changes:main.go"))
		typeMessage(t, h, "Fails")
		clickItem(t, h, render.ItemCommitMain, 0, 0)
		clickItem(t, h, render.ItemAskYes, 0, 0)
		if got := w.logged("")[2:]; !slices.Equal(got, []string{"check main.go", "stage main.go", "commit-sel Fails main.go staged main.go"}) {
			t.Fatalf("writes %q", got)
		}
		if v := h.m.ui.Source; v.Notice != "hook says no. The change staged for it stays staged." || v.NoticeKind != render.NoticeFailed ||
			!slices.Equal(selectedPaths(h), []string{"main.go"}) || h.m.msg.Text() != "Fails" {
			t.Fatalf("a failed commit after the staging: note %q, selection %q, field %q", v.Notice, selectedPaths(h), h.m.msg.Text())
		}
		w.commitErr = nil
		clickItem(t, h, render.ItemCommitMain, 0, 0)
		if got := w.logged("")[5:]; !slices.Equal(got, []string{"commit-sel Fails main.go staged "}) || h.m.ui.Source.Ask != 0 {
			t.Fatalf("the retry asks %d, writes %q", h.m.ui.Source.Ask, got)
		}

		// Plan 45's question: nothing staged, no selection, the branch not
		// ahead (else the button is Push, design D7).
		w.set(func(f *fakeGit) {
			f.snap.Entries = []input.GitEntry{{X: ' ', Y: 'M', Path: "a.go", RepoPath: "a.go"}, {X: '?', Y: '?', Path: "b.go", RepoPath: "b.go"}}
			f.snap.Branch.Ahead = 0
		})
		refresh(h)
		w.commitErr = errors.New("hook says no.")
		typeMessage(t, h, "Whole")
		clickItem(t, h, render.ItemCommitMain, 0, 0)
		if v := h.m.ui.Source; v.Ask != 2 || v.AskSel != 0 || !strings.Contains(h.frame(), "Nothing is staged.") {
			t.Fatalf("plan 45's question: ask %d, sel %d, note %q, selection %q, entries %+v, field %q\n%s", v.Ask, v.AskSel, v.Notice,
				selectedPaths(h), v.Entries, h.m.msg.Text(), h.frame())
		}
		clickItem(t, h, render.ItemAskYes, 0, 0)
		if got := w.logged("")[6:]; !slices.Equal(got, []string{"stage a.go,b.go", "commit Whole"}) {
			t.Fatalf("writes %q", got)
		}
		if v := h.m.ui.Source; v.Notice != "hook says no. The 2 changes staged for it stay staged." || h.m.msg.Text() != "Whole" {
			t.Fatalf("plan 45's question, a failed commit: note %q, field %q", v.Notice, h.m.msg.Text())
		}
	})
}

// TestSelectionGit is criterion 50.8 on real git (amendment 51): "Stage and
// commit", then a pre-commit hook that exits 1, for the selection's
// question and for plan 45's. HEAD stays; exactly the question's files are
// staged, their blobs their working tree's; the other files and the working
// tree as they were; the note git's words, then that the files stay
// staged; the field and the selection kept. The hook removed, Commit
// commits exactly them, asking nothing. Then a file moved on disk, both its
// rows selected: Commit commits the rename.
func TestSelectionGit(t *testing.T) {
	isolateGit(t)
	r := claudeRig(t)
	h := r.h
	root := h.m.State().Context.Project
	gitRun(t, root, "init", "-q", "-b", "main")
	put := func(name, text string) {
		if err := os.WriteFile(filepath.Join(root, name), []byte(text), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	put("a.txt", "a\n")
	put("b.txt", "b\n")
	put("c.txt", "c\n")
	gitRun(t, root, "add", "a.txt", "b.txt", "c.txt")
	gitRun(t, root, "commit", "-q", "-m", "init")
	hook := filepath.Join(root, ".git", "hooks", "pre-commit")
	refuse := func() {
		if err := os.MkdirAll(filepath.Dir(hook), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(hook, []byte("#!/bin/sh\necho 'not today' >&2\nexit 1\n"), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	put("b.txt", "B\n") // to select
	put("u.txt", "u\n") // to select
	put("c.txt", "C\n") // staged, not selected
	gitRun(t, root, "add", "c.txt")
	refuse()

	h.send(tea.WindowSizeMsg{Width: 200, Height: 60})
	for _, c := range h.m.showSource() {
		h.exec(c)
	}
	// until settles until ready holds, at most a minute: real git.
	until := func(ready func() bool) {
		for i := 0; i < 200 && !ready(); i++ {
			h.settle()
		}
	}
	listed := func(key string) func() bool {
		return func() bool {
			for _, r := range render.SourceRows(h.m.ui.Source) {
				if r.Key == key {
					return true
				}
			}
			return false
		}
	}
	until(listed("f:changes:u.txt"))
	settle := func() {
		until(func() bool { return h.m.op == nil })
		if h.m.op != nil {
			t.Fatal("the operation never ended")
		}
	}
	old := gitRun(t, root, "rev-parse", "HEAD")
	blobs := func() string { return gitRun(t, root, "ls-files", "-s", "a.txt", "b.txt", "c.txt", "u.txt") }
	files := func() string {
		var b strings.Builder
		for _, n := range []string{"a.txt", "b.txt", "c.txt", "u.txt"} {
			data, _ := os.ReadFile(filepath.Join(root, n))
			b.WriteString(n + "=" + string(data) + ";")
		}
		return b.String()
	}
	tree := files()

	// The selection's question.
	ctrlClick(t, h, rowItem(t, h, "f:changes:b.txt"))
	ctrlClick(t, h, rowItem(t, h, "f:changes:u.txt"))
	typeMessage(t, h, "Two files")
	h.send(ctrlEnter)
	if v := h.m.ui.Source; v.Ask != 2 || v.AskSel != 2 {
		t.Fatalf("the question: ask %d, sel %d, note %q", v.Ask, v.AskSel, v.Notice)
	}
	h.send(keyOf(tea.KeyEnter)) // Stage and commit
	settle()
	if head := gitRun(t, root, "rev-parse", "HEAD"); head != old {
		t.Fatalf("HEAD moved to %s", head)
	}
	if st := gitRun(t, root, "status", "--porcelain"); st != "M  b.txt\nM  c.txt\nA  u.txt" {
		t.Fatalf("status after the failure:\n%s", st)
	}
	for _, n := range []string{"b.txt", "u.txt"} {
		if blob := gitRun(t, root, "hash-object", n); !strings.Contains(blobs(), blob+" 0\t"+n) {
			t.Errorf("%s not staged with its working tree's content: %s\n%s", n, blob, blobs())
		}
	}
	if got := files(); got != tree {
		t.Errorf("the working tree changed: %q, want %q", got, tree)
	}
	v := h.m.ui.Source
	if !strings.Contains(v.Notice, "not today") || !strings.HasSuffix(v.Notice, " The 2 changes staged for it stay staged.") || v.NoticeKind != render.NoticeFailed {
		t.Fatalf("the note %q (%c)", v.Notice, v.NoticeKind)
	}
	if !slices.Equal(selectedPaths(h), []string{"b.txt", "u.txt"}) || h.m.msg.Text() != "Two files" {
		t.Fatalf("the selection %q, the field %q", selectedPaths(h), h.m.msg.Text())
	}
	if err := os.Remove(hook); err != nil {
		t.Fatal(err)
	}
	refresh(h)
	until(listed("f:staged:u.txt"))
	h.send(ctrlEnter)
	if h.m.ui.Source.Ask != 0 {
		t.Fatalf("the retry asks %d", h.m.ui.Source.Ask)
	}
	settle()
	if got := gitRun(t, root, "diff-tree", "--no-commit-id", "--name-only", "-r", "HEAD"); got != "b.txt\nu.txt" {
		t.Fatalf("the commit changes %q, note %q", got, h.m.ui.Source.Notice)
	}
	if st := gitRun(t, root, "status", "--porcelain"); st != "M  c.txt" || len(h.m.ui.Source.Selected) != 0 {
		t.Fatalf("after the commit: status %q, selection %q", st, selectedPaths(h))
	}

	// Plan 45's question: nothing staged, no selection.
	gitRun(t, root, "commit", "-q", "-m", "c")
	put("a.txt", "A\n")
	put("v.txt", "v\n")
	refuse()
	old = gitRun(t, root, "rev-parse", "HEAD")
	refresh(h)
	until(listed("f:changes:v.txt"))
	typeMessage(t, h, "Whole")
	h.send(ctrlEnter)
	if v := h.m.ui.Source; v.Ask != 2 || v.AskSel != 0 {
		t.Fatalf("plan 45's question: ask %d, sel %d, note %q", v.Ask, v.AskSel, v.Notice)
	}
	h.send(keyOf(tea.KeyEnter))
	settle()
	if head := gitRun(t, root, "rev-parse", "HEAD"); head != old {
		t.Fatalf("HEAD moved to %s", head)
	}
	if st := gitRun(t, root, "status", "--porcelain"); st != "M  a.txt\nA  v.txt" {
		t.Fatalf("status after the failure:\n%s", st)
	}
	if v := h.m.ui.Source; !strings.Contains(v.Notice, "not today") || !strings.HasSuffix(v.Notice, " The 2 changes staged for it stay staged.") || h.m.msg.Text() != "Whole" {
		t.Fatalf("plan 45's note %q, field %q", v.Notice, h.m.msg.Text())
	}
	if err := os.Remove(hook); err != nil {
		t.Fatal(err)
	}
	refresh(h)
	until(listed("f:staged:v.txt"))
	h.send(ctrlEnter)
	settle()
	if got := gitRun(t, root, "diff-tree", "--no-commit-id", "--name-only", "-r", "HEAD"); got != "a.txt\nv.txt" {
		t.Fatalf("the commit changes %q, note %q", got, h.m.ui.Source.Notice)
	}

	// A file moved on disk (the review of plan 50): its source deleted and
	// the file untracked, both Ctrl+clicked; the question stages them, git
	// pairs them as a rename, and the commit is exactly that rename.
	if err := os.Rename(filepath.Join(root, "b.txt"), filepath.Join(root, "moved.txt")); err != nil {
		t.Fatal(err)
	}
	refresh(h)
	until(listed("f:changes:moved.txt"))
	ctrlClick(t, h, rowItem(t, h, "f:changes:b.txt"))
	ctrlClick(t, h, rowItem(t, h, "f:changes:moved.txt"))
	typeMessage(t, h, "Moved")
	h.send(ctrlEnter)
	if v := h.m.ui.Source; v.Ask != 2 || v.AskSel != 2 {
		t.Fatalf("the moved file's question: ask %d, sel %d, note %q", v.Ask, v.AskSel, v.Notice)
	}
	h.send(keyOf(tea.KeyEnter)) // Stage and commit
	settle()
	if got := gitRun(t, root, "diff-tree", "--no-commit-id", "--name-status", "-r", "-M", "HEAD"); got != "R100\tb.txt\tmoved.txt" {
		t.Fatalf("the commit changes %q, note %q", got, h.m.ui.Source.Notice)
	}
	if st := gitRun(t, root, "status", "--porcelain"); st != "" || len(h.m.ui.Source.Selected) != 0 || h.m.ui.Source.NoticeKind != render.NoticeDone {
		t.Fatalf("after the commit: status %q, selection %q, note %q", st, selectedPaths(h), h.m.ui.Source.Notice)
	}
}
