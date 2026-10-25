package app

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/Evengelius/agent-command-center/internal/event"
	"github.com/Evengelius/agent-command-center/internal/input"
	"github.com/Evengelius/agent-command-center/internal/render"
)

// Plan 44 (brief amendment 45): Source Control and the diff view through
// the application, git's answers from the test double.

// Snapshot is the fake's status: its entries, branch and top, its markers;
// recorded as the status it is.
func (f *fakeGit) Snapshot(dir string) (input.GitSnapshot, error) {
	f.wait()
	f.record("status " + dir)
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.statusErr != nil {
		return input.GitSnapshot{}, f.statusErr
	}
	s := f.snap
	s.Marks, s.Entries = f.marks, slices.Clone(f.snap.Entries)
	return s, nil
}

// Sides is the fake's sides of a file's comparison.
func (f *fakeGit) Sides(dir string, snap input.GitSnapshot, e input.GitEntry, staged bool) (input.DiffPair, error) {
	f.wait()
	f.record(fmt.Sprintf("sides %s %v", e.Path, staged))
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.sidesErr != nil {
		return input.DiffPair{}, f.sidesErr
	}
	return f.sides[fmt.Sprintf("%s %v", e.Path, staged)], nil
}

// set changes the fake's answers under its lock.
func (f *fakeGit) set(change func(f *fakeGit)) {
	f.mu.Lock()
	defer f.mu.Unlock()
	change(f)
}

// side is a side named n holding text, its stamp id.
func side(n, id, text string) input.DiffSide {
	return input.DiffSide{Name: n, ID: id, Data: []byte(text), Size: int64(len(text))}
}

// numbered is lines 1 to n, "line i", each with its line end.
func numbered(n int, change map[int]string) string {
	var b strings.Builder
	for i := 1; i <= n; i++ {
		if s, ok := change[i]; ok {
			b.WriteString(s)
		} else {
			fmt.Fprintf(&b, "line %d\n", i)
		}
	}
	return b.String()
}

// sourceRig is the file rig's project as a repository on main: main.go
// changed in the working tree, src/Main.java staged and changed again,
// docs/readme.md added, gone.txt deleted, web/ untracked; each file's
// sides.
func sourceRig(t *testing.T) (inputRig, *fakeGit, string) {
	r, _, root := fileRig(t)
	g := &fakeGit{marks: input.GitMarks{Files: map[string]byte{"main.go": 'M'}},
		snap: input.GitSnapshot{Top: root, Branch: input.GitBranch{Name: "main"}, Entries: []input.GitEntry{
			{X: ' ', Y: 'M', Path: "main.go", RepoPath: "main.go"},
			{X: 'M', Y: 'M', Path: "src/Main.java", RepoPath: "src/Main.java"},
			{X: 'A', Y: ' ', Path: "docs/readme.md", RepoPath: "docs/readme.md"},
			{X: ' ', Y: 'D', Path: "gone.txt", RepoPath: "gone.txt"},
			{X: '?', Y: '?', Path: "web/", RepoPath: "web/"},
		}},
		sides: map[string]input.DiffPair{
			"main.go false": {Old: side("index", "i1", numbered(100, nil)),
				New: side("working tree", "", numbered(100, map[int]string{10: "line ten " + strings.Repeat("x", 150) + "\n", 50: "", 90: "line 90\nline 90b\n"}))},
			"src/Main.java true":  {Old: side("HEAD", "h2", "class Main {}\n"), New: side("index", "i2", "class Main { int x; }\n")},
			"src/Main.java false": {Old: side("index", "i2", "class Main { int x; }\n"), New: side("working tree", "", "class Main { int x, y; }\n")},
			"docs/readme.md true": {Old: input.DiffSide{Name: "HEAD", Missing: true}, New: side("index", "i3", "# Readme\n")},
			"gone.txt false":      {Old: side("index", "i4", "bye\n"), New: input.DiffSide{Name: "working tree", Missing: true}},
			"web/app.ts false":    {Old: input.DiffSide{Name: "index", Missing: true}, New: side("working tree", "", "export const a: number = 1;\n")},
		}}
	r.h.m.opt.Git = g
	r.h.send(tea.WindowSizeMsg{Width: 200, Height: 60})
	return r, g, root
}

// showSource clicks Source Control's tile.
func showSource(t *testing.T, h *harness) {
	t.Helper()
	x, y := itemCell(t, h, render.SidePanel, render.SideSource)
	mouseClick(t, h, x, y)
	if !h.m.ui.Source.Open || h.m.ui.Focus != "R" {
		t.Fatalf("the tile: Source Control shown %v, focus %q", h.m.ui.Source.Open, h.m.ui.Focus)
	}
}

// sourceTo puts the panel's cursor on the row with key, by the keys: Home,
// then ↓.
func sourceTo(t *testing.T, h *harness, key string) {
	t.Helper()
	h.send(keyOf(tea.KeyHome))
	for range 40 {
		rows := render.SourceRows(h.m.ui.Source)
		if i := render.SourceCursor(rows, h.m.ui.Source.Cursor); i >= 0 && rows[i].Key == key {
			return
		}
		h.send(keyOf(tea.KeyDown))
	}
	t.Fatalf("no row %q: %q", key, h.m.ui.Source.Cursor)
}

func keyOf(code rune) tea.KeyPressMsg { return key(code, 0) }

// screenNow is the screen of a frame drawn now: the harness draws one only when
// a test asks.
func screenNow(h *harness) *render.Screen {
	h.m.View()
	return h.m.screen
}

// sideLookOf is a tile's look on the frame drawn now, with Sixel on.
func sideLookOf(h *harness, tile int) render.SideLook {
	h.m.View()
	for _, tl := range h.m.screen.Tiles {
		if tl.Shown && tl.Look.Tile == tile {
			return tl.Look.Look
		}
	}
	return -1
}

// TestSourceTile is plan 44, criterion 44.1 (design D1): a click on Source
// Control's tile shows its panel docked, the tile selected and the focus on
// it, a second hides it; the Explorer's tile and Source Control's switch
// the docked place, each keeping its state; F3 with Source Control shown
// shows the Explorer, F3 again hides it; the tooltip names a hovered tile
// beside it and goes with the pointer.
func TestSourceTile(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r, g, root := sourceRig(t)
		h := r.h
		h.send(PrimaryDeviceAttributesEvent{61, 4, 6, 7, 14}) // Sixel: the tiles' looks on the screen
		showSource(t, h)
		f := h.frame()
		if !strings.Contains(f, "SOURCE CONTROL") || !strings.Contains(f, filepath.Base(root)+" · main") || h.m.screen.B.W != 106 ||
			sideLookOf(h, render.SideSource) != render.SideSelectedHover || h.m.ui.Explorer.Open {
			t.Fatalf("shown: zone B %d, look %v, Explorer %v:\n%s", h.m.screen.B.W, sideLookOf(h, render.SideSource), h.m.ui.Explorer.Open, f)
		}
		if got := g.called("status"); len(got) != 1 || got[0] != "status "+root {
			t.Errorf("git calls %q", got)
		}
		sourceTo(t, h, "f:changes:gone.txt")
		h.send(keyOf(tea.KeyF3)) // the Explorer in the docked place
		if !h.m.ui.Explorer.Open || h.m.ui.Source.Open || h.m.ui.Focus != "R" || !strings.Contains(h.frame(), "EXPLORER") {
			t.Fatalf("F3: Explorer %v, Source Control %v, focus %q", h.m.ui.Explorer.Open, h.m.ui.Source.Open, h.m.ui.Focus)
		}
		h.send(keyOf(tea.KeyRight)) // docs opened: the Explorer's state to find again
		explorer := fmt.Sprint(treeNames(h.m.ui.Explorer), h.m.ui.Explorer.Cursor)
		showSource(t, h)
		if h.m.ui.Explorer.Open || h.m.ui.Source.Cursor != "f:changes:gone.txt" {
			t.Errorf("back to Source Control: Explorer %v, cursor %q", h.m.ui.Explorer.Open, h.m.ui.Source.Cursor)
		}
		x, y := itemCell(t, h, render.SidePanel, render.SideExplorer)
		mouseClick(t, h, x, y)
		if got := fmt.Sprint(treeNames(h.m.ui.Explorer), h.m.ui.Explorer.Cursor); !h.m.ui.Explorer.Open || got != explorer {
			t.Errorf("the Explorer's tile: %v %s, want %s", h.m.ui.Explorer.Open, got, explorer)
		}
		h.send(keyOf(tea.KeyF3)) // hidden, as in plan 43
		if h.m.ui.Explorer.Open || h.m.ui.Source.Open || h.m.ui.Focus != "B" || screenNow(h).B.W != 136 {
			t.Errorf("F3 again: Explorer %v, Source Control %v, focus %q", h.m.ui.Explorer.Open, h.m.ui.Source.Open, h.m.ui.Focus)
		}
		showSource(t, h)
		sx, sy := itemCell(t, h, render.SidePanel, render.SideSource)
		mouseClick(t, h, sx, sy)
		if h.m.ui.Source.Open || h.m.ui.Focus != "B" || screenNow(h).B.W != 136 || sideLookOf(h, render.SideSource) != render.SideHover {
			t.Errorf("a second click: shown %v, focus %q, zone B %d", h.m.ui.Source.Open, h.m.ui.Focus, h.m.screen.B.W)
		}

		// The tooltip, left of the sidebar on the tile's middle row; none after
		// the click, the pointer still on the tile, until it leaves.
		if strings.Contains(h.frame(), " Source Control │") {
			t.Error("the tooltip stays after a click on the tile")
		}
		for _, c := range []struct {
			tile  int
			label string
		}{{render.SideSource, " Source Control "}, {render.SideExplorer, " Explorer "}} {
			x, y := itemCell(t, h, render.SidePanel, c.tile)
			mouseAt(t, h, moveTo(5, 5))
			mouseAt(t, h, moveTo(x+1, y))
			if row := strings.Split(h.frame(), "\n")[y+1]; !strings.Contains(row, c.label+"│") {
				t.Errorf("hovering %d: no %q beside the sidebar: %q", c.tile, c.label, row)
			}
			mouseAt(t, h, moveTo(5, 5))
			if strings.Contains(h.frame(), c.label+"│") {
				t.Errorf("the tooltip %q stays off the tile", c.label)
			}
		}
	})
}

// TestSourceList is criteria 44.4 and 44.11 through the application (designs
// D4 and D11): the panel lists the sections of git's entries, an untracked
// folder opening on its files read from the disk; not a repository, git
// missing and git failing are said in the list's place, nothing else run.
func TestSourceList(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r, g, root := sourceRig(t)
		h := r.h
		showSource(t, h)
		f := h.frame()
		for _, s := range []string{"▾ CHANGES", "▾ STAGED CHANGES", "M  main.go", "M  src/Main.java", "D  gone.txt", "U  › web/", "A  docs/readme.md"} {
			if !strings.Contains(f, s) {
				t.Errorf("%q not drawn:\n%s", s, f)
			}
		}
		sourceTo(t, h, "f:changes:web/")
		g.set(func(f *fakeGit) { f.marks.Ignored = map[string]bool{"web/build.log": true} }) // an ignored file in the folder
		if err := os.WriteFile(filepath.Join(root, "web", "build.log"), []byte("log\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		h.send(typed('r'))
		h.send(keyOf(tea.KeyRight))
		if f := h.frame(); !strings.Contains(f, "U  ⌄ web/") || !strings.Contains(f, "U  web/app.ts") || strings.Contains(f, "build.log") {
			t.Errorf("web/ opened, its ignored file left out:\n%s", f)
		}
		h.send(keyOf(tea.KeyLeft))
		if strings.Contains(h.frame(), "web/app.ts") {
			t.Error("web/ folded, its file still shown")
		}
		sourceTo(t, h, "h:changes")
		h.send(keyOf(tea.KeyEnter))
		if f := h.frame(); !strings.Contains(f, "▸ CHANGES") || strings.Contains(f, "M  main.go") {
			t.Errorf("CHANGES folded by Enter:\n%s", f)
		}
	})
	for _, c := range []struct {
		err       error
		untracked bool
		note      string
	}{
		{input.NotRepoError{Dir: "x"}, false, "No Git repository found for this project."},
		{input.ErrNoGit, false, "git not found"},
		{errors.New("index locked"), false, "git: index locked"},
		{nil, true, "The project's folder is untracked in this repository: git lists none of its files apart."},
	} {
		t.Run(c.note, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				r, g, root := sourceRig(t)
				g.set(func(f *fakeGit) {
					f.statusErr = c.err
					if c.untracked {
						f.snap.Untracked, f.snap.Entries = true, nil
					}
				})
				h := r.h
				showSource(t, h)
				var words []string
				x0, x1 := panelExtent(h, render.SourcePanel)
				for y := range h.m.ui.Height {
					words = append(words, strings.Fields(strings.Trim(plainText(h.m.screen.Region(x0, y, x1-x0+1)), "│"))...)
				}
				if got := strings.Join(words, " "); !strings.Contains(got, c.note) {
					t.Errorf("%q not drawn whole: %q", c.note, got)
				}
				h.send(keyOf(tea.KeyEnter))
				if got := g.calls; len(got) != 1 || got[0] != "status "+root || h.m.ui.Diff.Open {
					t.Errorf("git calls %q, diff %v", got, h.m.ui.Diff.Open)
				}
			})
		})
	}
}

// TestSourceFoldersFollow is plan 46's criterion 46.6 (design D3): an opened
// untracked folder is read again after each status read and each time it is
// opened, so a file written in it appears; the listing never shows less than
// what "Stage and commit" would stage.
func TestSourceFoldersFollow(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r, _, root := sourceRig(t)
		h := r.h
		showSource(t, h)
		sourceTo(t, h, "f:changes:web/")
		h.send(keyOf(tea.KeyRight))
		if f := h.frame(); !strings.Contains(f, "U  web/app.ts") || strings.Contains(f, "secret.env") {
			t.Fatalf("web/ opened:\n%s", f)
		}
		if err := os.WriteFile(filepath.Join(root, "web", "secret.env"), []byte("KEY=1\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		h.send(typed('r')) // the next status read, as the poll's
		if f := h.frame(); !strings.Contains(f, "U  web/secret.env") || !strings.Contains(f, "U  web/app.ts") {
			t.Errorf("after the next status read, the new file not listed:\n%s", f)
		}
		h.send(keyOf(tea.KeyLeft))
		if err := os.WriteFile(filepath.Join(root, "web", "two.txt"), []byte("2\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		h.send(keyOf(tea.KeyRight)) // opened again: read again
		if f := h.frame(); !strings.Contains(f, "U  web/two.txt") {
			t.Errorf("opened again, the folder not read again:\n%s", f)
		}
	})
}

// TestSourceFolderReadsOrdered is plan 46's criterion 46.7 (design D3): of
// two reads of one folder answered in reverse order, the newer listing is
// kept; a read for a folder folded, or no longer untracked, is dropped.
func TestSourceFolderReadsOrdered(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r, g, _ := sourceRig(t)
		h := r.h
		showSource(t, h)
		sourceTo(t, h, "f:changes:web/")
		h.send(keyOf(tea.KeyRight))
		files := func() []string {
			if d := h.m.ui.Source.Folders["web/"]; d != nil {
				return d.Files
			}
			return nil
		}
		seq := h.m.folderSeq
		h.send(folderMsg{seq: seq + 2, rel: "web/", files: []string{"web/new.ts"}})
		h.send(folderMsg{seq: seq + 1, rel: "web/", files: []string{"web/old.ts"}})
		if got := files(); !slices.Equal(got, []string{"web/new.ts"}) {
			t.Errorf("after the newer then the older read: %q", got)
		}
		h.send(keyOf(tea.KeyLeft)) // folded
		h.send(folderMsg{seq: seq + 3, rel: "web/", files: []string{"web/late.ts"}})
		if got := files(); !slices.Equal(got, []string{"web/new.ts"}) {
			t.Errorf("a read for the folded folder applied: %q", got)
		}
		g.set(func(f *fakeGit) { f.snap.Entries = f.snap.Entries[:4] }) // web/ no longer untracked
		h.send(typed('r'))
		h.send(folderMsg{seq: seq + 4, rel: "web/", files: []string{"web/gone.ts"}})
		if got := files(); got != nil {
			t.Errorf("a read for a folder no longer untracked applied: %q", got)
		}
	})
}

// TestSourceRefresh is criterion 44.5 (design D3): git's status is read
// when the panel shows, on "r", after a tool call and a turn end; a change
// appears, a revert removes it, the cursor kept on its file; a read equal
// to the last changes nothing; never two reads at once, one more asked
// while one runs; none while no panel or diff shows.
func TestSourceRefresh(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r, g, _ := sourceRig(t)
		h := r.h
		reads := func() int { return len(g.called("status")) }
		showSource(t, h)
		sourceTo(t, h, "f:changes:gone.txt")
		before := h.frame()
		h.send(typed('r'))
		if reads() != 2 || h.frame() != before {
			t.Fatalf("r: %d reads; an equal read changed the frame", reads())
		}
		g.set(func(f *fakeGit) {
			f.snap.Entries = append([]input.GitEntry{{X: ' ', Y: 'M', Path: "docs/new.md", RepoPath: "docs/new.md"}}, f.snap.Entries...)
		})
		h.send(typed('r'))
		if f := h.frame(); !strings.Contains(f, "M  docs/new.md") || h.m.ui.Source.Cursor != "f:changes:gone.txt" {
			t.Errorf("a new change: cursor %q:\n%s", h.m.ui.Source.Cursor, f)
		}
		g.set(func(f *fakeGit) { f.snap.Entries = f.snap.Entries[1:] })
		// gone.txt's deletion staged: the cursor follows it into STAGED CHANGES.
		g.set(func(f *fakeGit) {
			for i := range f.snap.Entries {
				if f.snap.Entries[i].Path == "gone.txt" {
					f.snap.Entries[i].X, f.snap.Entries[i].Y = 'D', ' '
				}
			}
		})
		h.send(typed('r'))
		if h.m.ui.Source.Cursor != "f:staged:gone.txt" {
			t.Errorf("gone.txt staged: cursor %q", h.m.ui.Source.Cursor)
		}
		r.emit(event.ToolStarted, event.Tool{ToolID: "t1", Name: "Bash"})
		h.settle()
		n := reads()
		r.emit(event.ToolCompleted, event.Tool{ToolID: "t1", Name: "Bash"})
		h.settle()
		if reads() != n+1 || strings.Contains(h.frame(), "docs/new.md") {
			t.Errorf("a tool's end: %d reads, want %d; the revert still shown", reads(), n+1)
		}
		r.emit(event.TurnCompleted, event.TurnEnd{Outcome: "completed"})
		h.settle()
		if reads() != n+2 {
			t.Errorf("a turn's end: %d reads, want %d", reads(), n+2)
		}

		// One read at a time: two more asked while one runs make one more.
		block := make(chan struct{})
		g.set(func(f *fakeGit) { f.block = block })
		n = reads()
		for range 3 {
			h.send(typed('r'))
		}
		if !h.m.statusBusy || !h.m.statusAgain {
			t.Fatalf("busy %v, again %v", h.m.statusBusy, h.m.statusAgain)
		}
		g.set(func(f *fakeGit) { f.block = nil })
		close(block)
		h.settle()
		if reads() != n+2 || h.m.statusBusy {
			t.Errorf("three r while one ran: %d reads, want %d; busy %v", reads(), n+2, h.m.statusBusy)
		}

		h.send(keyOf(tea.KeyEscape)) // zone B, the panel kept: still read
		r.emit(event.ToolCompleted, event.Tool{ToolID: "t2", Name: "Bash"})
		h.settle()
		if reads() != n+3 {
			t.Errorf("the panel beside zone B: %d reads, want %d", reads(), n+3)
		}
		sx, sy := itemCell(t, h, render.SidePanel, render.SideSource)
		mouseClick(t, h, sx, sy) // hidden
		r.emit(event.ToolCompleted, event.Tool{ToolID: "t3", Name: "Bash"})
		r.emit(event.TurnCompleted, event.TurnEnd{Outcome: "completed"})
		h.settle()
		if reads() != n+3 {
			t.Errorf("no panel: %d reads, want %d", reads(), n+3)
		}
	})
}

// TestSourcePoll is criterion 44.5's pace (design D3): every 3 s while a
// panel or a diff shows, ten times a slow read's duration, at most 30 s; a
// stale timer or one with nothing shown reads nothing.
func TestSourcePoll(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r, g, _ := sourceRig(t)
		h := r.h
		var mu sync.Mutex
		var timed []time.Duration
		var polls []pollMsg
		h.m.opt.After = func(d time.Duration, msg tea.Msg) tea.Cmd {
			if p, ok := msg.(pollMsg); ok {
				mu.Lock()
				timed, polls = append(timed, d), append(polls, p)
				mu.Unlock()
			}
			return nil
		}
		var clockMu sync.Mutex
		now, step := testNow, time.Duration(0)
		h.m.opt.Now = func() time.Time {
			clockMu.Lock()
			defer clockMu.Unlock()
			now = now.Add(step)
			return now
		}
		setStep := func(d time.Duration) { clockMu.Lock(); step = d; clockMu.Unlock() }
		showSource(t, h)
		if len(timed) != 1 || timed[0] != 3*time.Second {
			t.Fatalf("after the first read: %v", timed)
		}
		n := len(g.called("status"))
		h.send(polls[0])
		if len(g.called("status")) != n+1 || len(timed) != 2 || timed[1] != 3*time.Second {
			t.Errorf("the timer: %d reads, timed %v", len(g.called("status"))-n, timed)
		}
		h.send(polls[0]) // stale
		if len(g.called("status")) != n+1 {
			t.Error("a stale timer read git's status")
		}
		setStep(time.Second) // a read takes a second of the clock (one call at its start, one at its end)
		h.send(typed('r'))
		setStep(0)
		if d := timed[len(timed)-1]; d != 10*time.Second {
			t.Errorf("a read of 1 s: next in %v, want 10s", d)
		}
		setStep(5 * time.Second)
		h.send(typed('r'))
		setStep(0)
		if d := timed[len(timed)-1]; d != 30*time.Second {
			t.Errorf("a read of 5 s: next in %v, want 30s", d)
		}
		last := polls[len(polls)-1]
		sx, sy := itemCell(t, h, render.SidePanel, render.SideSource)
		mouseClick(t, h, sx, sy) // hidden
		n = len(g.called("status"))
		h.send(last)
		if len(g.called("status")) != n {
			t.Error("a timer with nothing shown read git's status")
		}
	})
}

// openDiffOf opens the diff of the row with key by Enter, and checks it.
func openDiffOf(t *testing.T, h *harness, key string) {
	t.Helper()
	sourceTo(t, h, key)
	h.send(keyOf(tea.KeyEnter))
	if !h.m.ui.Diff.Open || h.m.ui.Focus != "D" || h.m.ui.Diff.Note == "comparing…" {
		t.Fatalf("%s: diff %v, focus %q, note %q", key, h.m.ui.Diff.Open, h.m.ui.Focus, h.m.ui.Diff.Note)
	}
}

// TestDiffOpen is criteria 44.8 and 44.9 through the application (design
// D8): Enter on a file shows its diff in zone B's place, the panel, zone A,
// the sidebar and zone C kept; its lines numbered, green and red; Esc gives
// back zone B as it was, the draft and the conversation's scroll, the focus
// on the panel's file; a click opens it too.
func TestDiffOpen(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r, _, _ := sourceRig(t)
		h := r.h
		for _, k := range typedText("a draft kept") {
			h.send(k)
		}
		h.m.ui.ConvHold, h.m.ui.ConvTop = true, 0
		showSource(t, h)
		sourceTo(t, h, "f:changes:main.go")
		before, conv := h.frame(), fmt.Sprint(h.m.ui.ConvHold, h.m.ui.ConvTop, h.m.ui.ConvSeen)
		ax0, ax1 := panelExtent(h, 'A')
		h.send(keyOf(tea.KeyEnter))
		f := h.frame()
		if !h.m.ui.Diff.Open || h.m.ui.Focus != "D" || h.m.screen.Diff != (render.Cols{X: 56, W: 106}) || h.m.screen.B.W != 0 {
			t.Fatalf("Enter: diff %v, focus %q, box %+v, zone B %d", h.m.ui.Diff.Open, h.m.ui.Focus, h.m.screen.Diff, h.m.screen.B.W)
		}
		for _, s := range []string{"DIFF · main.go", "main.go · Working tree: index → working tree", "+2 −2", "@@ -7,7 +7,7 @@",
			" 10     − line 10", "     10 + line ten xxx", "SOURCE CONTROL"} {
			if !strings.Contains(f, s) {
				t.Errorf("%q not drawn:\n%s", s, f)
			}
		}
		if x0, x1 := panelExtent(h, 'A'); x0 != ax0 || x1 != ax1 {
			t.Errorf("zone A moved: %d..%d, was %d..%d", x0, x1, ax0, ax1)
		}
		// The panel docked beside the diff: F6 gives it the focus, a paste then
		// goes nowhere (no draft on screen), Esc gives zone B's focus, the
		// diff's in its place.
		h.send(keyOf(tea.KeyF6))
		h.send(tea.PasteMsg{Content: "not for the draft"})
		if h.m.ui.Focus != "R" || h.m.edit.Text() != "a draft kept" {
			t.Errorf("a paste beside the diff: focus %q, draft %q", h.m.ui.Focus, h.m.edit.Text())
		}
		h.send(keyOf(tea.KeyEscape))
		if h.m.ui.Focus != "D" || !h.m.ui.Diff.Open {
			t.Errorf("Esc from the panel: focus %q, diff %v", h.m.ui.Focus, h.m.ui.Diff.Open)
		}
		h.send(keyOf(tea.KeyEscape))
		if got := h.frame(); got != before || h.m.ui.Focus != "R" || h.m.ui.Diff.Open {
			t.Errorf("Esc: focus %q, the frame changed:\n%s\nwas\n%s", h.m.ui.Focus, got, before)
		}
		if h.m.edit.Text() != "a draft kept" || fmt.Sprint(h.m.ui.ConvHold, h.m.ui.ConvTop, h.m.ui.ConvSeen) != conv {
			t.Errorf("the draft %q, the scroll %s, was %s", h.m.edit.Text(), fmt.Sprint(h.m.ui.ConvHold, h.m.ui.ConvTop, h.m.ui.ConvSeen), conv)
		}
		// A click on a file opens its diff.
		x, y := at(t, h, "readme.md", 0)
		mouseClick(t, h, x, y)
		if f := h.frame(); !h.m.ui.Diff.Open || !strings.Contains(f, "Staged: nothing → index") || !strings.Contains(f, "+ # Readme") {
			t.Errorf("a click on docs/readme.md:\n%s", f)
		}
	})
}

// TestDiffKeys is criterion 44.9 (design D9): ↑↓ PgUp PgDn Home End and the
// wheel scroll, n and N go from change to change, ←→ shift within the
// longest line, Tab shows the other comparison, f widens over zone A and
// back, r compares again.
func TestDiffKeys(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r, g, _ := sourceRig(t)
		h := r.h
		h.send(tea.WindowSizeMsg{Width: 200, Height: 24})
		showSource(t, h)
		openDiffOf(t, h, "f:changes:main.go")
		v := func() render.DiffView { return h.m.ui.Diff }
		rows, changes := render.DiffRows(v(), h.m.diffInner())
		room := render.DiffRoom(h.m.State(), h.m.ui)
		bottom := len(rows) - room
		if len(changes) != 3 || changes[1]-changeLead <= 0 || changes[1]-changeLead >= bottom || changes[2]-changeLead <= bottom {
			t.Fatalf("%d changes %v, %d rows in %d", len(changes), changes, len(rows), room)
		}
		for _, c := range []struct {
			k    tea.Msg
			want int
		}{
			{keyOf(tea.KeyDown), 1}, {keyOf(tea.KeyPgDown), min(room, bottom)}, {keyOf(tea.KeyUp), min(room, bottom) - 1}, {keyOf(tea.KeyEnd), bottom},
			{keyOf(tea.KeyHome), 0}, {typed('n'), changes[1] - changeLead}, {typed('n'), bottom},
			{typed('N'), changes[1] - changeLead}, {typed('N'), 0},
		} {
			h.send(c.k)
			if v().Top != c.want {
				t.Errorf("%v: top %d, want %d", c.k, v().Top, c.want)
			}
		}
		x, y := at(t, h, "line ten", 0)
		mouseAt(t, h, tea.MouseWheelMsg{X: x, Y: y, Button: tea.MouseWheelDown})
		if v().Top != wheelRows {
			t.Errorf("the wheel: top %d", v().Top)
		}
		for range 20 {
			h.send(keyOf(tea.KeyRight))
		}
		if most := render.DiffMaxLeft(v(), h.m.diffInner()); v().Left != most {
			t.Errorf("→ past the end: left %d, want %d", v().Left, most)
		}
		h.send(keyOf(tea.KeyLeft))
		if v().Left != max(render.DiffMaxLeft(v(), h.m.diffInner())-8, 0) {
			t.Errorf("←: left %d", v().Left)
		}
		h.send(typed('f'))
		if screenNow(h).Diff != (render.Cols{X: 0, W: 162}) || !strings.Contains(h.frame(), "ORIGINAL  index") {
			t.Errorf("f: box %+v, side by side %v", h.m.screen.Diff, render.DiffSideBySide(h.m.diffInner()))
		}
		h.send(typed('f'))
		if screenNow(h).Diff != (render.Cols{X: 56, W: 106}) {
			t.Errorf("f again: box %+v", h.m.screen.Diff)
		}
		n := len(g.called("sides"))
		h.send(typed('r'))
		if len(g.called("sides")) < n+1 {
			t.Error("r compared nothing")
		}
		h.send(keyOf(tea.KeyTab)) // one comparison only: Tab is the next zone
		if h.m.ui.Focus != "R" || !h.m.ui.Diff.Open {
			t.Errorf("Tab on main.go: focus %q, diff %v", h.m.ui.Focus, h.m.ui.Diff.Open)
		}
		h.send(key(tea.KeyTab, tea.ModShift))
		if h.m.ui.Focus != "D" {
			t.Errorf("Shift+Tab back: focus %q", h.m.ui.Focus)
		}

		// Tab between a file's two comparisons.
		h.send(keyOf(tea.KeyEscape))
		openDiffOf(t, h, "f:changes:src/Main.java")
		if f := h.frame(); !strings.Contains(f, "Working tree: index → working tree · Tab staged") || !strings.Contains(f, "+ class Main { int x, y; }") {
			t.Errorf("Main.java's working tree:\n%s", f)
		}
		h.send(keyOf(tea.KeyTab))
		if f := h.frame(); !v().Staged || !strings.Contains(f, "Staged: HEAD → index · Tab working tree") || !strings.Contains(f, "+ class Main { int x; }") ||
			h.m.ui.Source.Cursor != "f:staged:src/Main.java" {
			t.Errorf("Tab: cursor %q\n%s", h.m.ui.Source.Cursor, f)
		}
	})
}

// TestDiffRefresh is criterion 44.5's open diff (design D3): a read whose
// sides kept their stamps leaves the view as it was, its scroll too; a
// working file changed is compared again, the scroll kept.
func TestDiffRefresh(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r, g, _ := sourceRig(t)
		h := r.h
		h.send(tea.WindowSizeMsg{Width: 200, Height: 24}) // room to scroll
		showSource(t, h)
		openDiffOf(t, h, "f:changes:main.go")
		h.send(keyOf(tea.KeyDown))
		lay := h.m.ui.Diff.Layout
		r.emit(event.ToolCompleted, event.Tool{ToolID: "t1", Name: "Bash"})
		h.settle()
		if h.m.ui.Diff.Layout != lay || h.m.ui.Diff.Top != 1 {
			t.Errorf("an equal read: the diff laid out again (%v), top %d", h.m.ui.Diff.Layout != lay, h.m.ui.Diff.Top)
		}
		g.set(func(f *fakeGit) {
			p := f.sides["main.go false"]
			p.New = side("working tree", "", numbered(100, map[int]string{10: "line ten\n", 50: "", 90: "line 90\nline 90b\n", 95: "line ninety-five\n"}))
			p.New.ModTime = testNow.Add(time.Minute)
			f.sides["main.go false"] = p
		})
		r.emit(event.ToolCompleted, event.Tool{ToolID: "t2", Name: "Bash"})
		h.settle()
		if h.m.ui.Diff.Layout == lay || h.m.ui.Diff.Top != 1 || !strings.Contains(fmt.Sprint(h.m.ui.Diff.Hunks), "line ninety-five") {
			t.Errorf("a changed file: laid out again %v, top %d", h.m.ui.Diff.Layout != lay, h.m.ui.Diff.Top)
		}
	})
}

// TestDiffNotes is criterion 44.6's cases without lines, through the
// application (design D6): binary, an LFS pointer against content, too
// large, line endings only, no textual change, git's error; an added file
// against nothing, a deleted one to nothing.
func TestDiffNotes(t *testing.T) {
	for _, c := range []struct {
		name string
		pair input.DiffPair
		err  error
		want string
	}{
		{"binary", input.DiffPair{Old: input.DiffSide{Name: "index", Binary: true, Size: 1200}, New: input.DiffSide{Name: "working tree", Binary: true, Size: 1300}},
			nil, "Binary file: no textual diff (1.2 KiB → 1.3 KiB)"},
		{"lfs", input.DiffPair{Old: input.DiffSide{Name: "index", LFS: true, Data: []byte("version https://git-lfs.github.com/spec/v1\n")}, New: side("working tree", "", "content")},
			nil, "Git LFS file: its content is kept outside git, no textual diff"},
		{"large", input.DiffPair{Old: side("index", "i", "a\n"), New: input.DiffSide{Name: "working tree", TooLarge: true, Size: 40 << 20}},
			nil, "Too large to compare"},
		{"line endings", input.DiffPair{Old: side("index", "i", "a\r\nb\r\n"), New: side("working tree", "", "a\nb\n")}, nil, "Only the line endings differ"},
		{"equal", input.DiffPair{Old: side("index", "i", "a\n"), New: side("working tree", "", "a\n")}, nil, "No textual changes"},
		{"error", input.DiffPair{}, errors.New("bad object"), "git: bad object"},
		{"credential", input.DiffPair{}, input.ErrCredential, "a credential file: not opened"},
	} {
		t.Run(c.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				r, g, _ := sourceRig(t)
				g.set(func(f *fakeGit) { f.sides["main.go false"], f.sidesErr = c.pair, c.err })
				h := r.h
				showSource(t, h)
				openDiffOf(t, h, "f:changes:main.go")
				if f := h.frame(); !strings.Contains(f, c.want) || h.m.ui.Diff.Layout != nil || h.m.ui.Diff.Failed != (c.err != nil) ||
					strings.Contains(f, "git: a credential") {
					t.Errorf("%q not drawn, or lines laid out:\n%s", c.want, f)
				}
			})
		})
	}
	synctest.Test(t, func(t *testing.T) {
		r, _, _ := sourceRig(t)
		h := r.h
		showSource(t, h)
		openDiffOf(t, h, "f:changes:gone.txt")
		if f := h.frame(); !strings.Contains(f, "index → nothing") || !strings.Contains(f, "− bye") || !strings.Contains(f, "+0 −1") {
			t.Errorf("a deleted file:\n%s", f)
		}
	})
	// A side's missing line end is said where its last line is shown, and
	// only there.
	for _, c := range []struct {
		name     string
		old, cur string
		want     string
	}{
		{"both last lines", "a\nb", "a\nc", "no newline at the end of either side"},
		{"the new one", "a\nb\n", "a\nc", "no newline at the end of the modified"},
		{"not shown", numbered(20, nil) + "end", numbered(20, map[int]string{2: "two\n"}) + "end", ""},
	} {
		t.Run(c.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				r, g, _ := sourceRig(t)
				g.set(func(f *fakeGit) {
					f.sides["main.go false"] = input.DiffPair{Old: side("index", "i", c.old), New: side("working tree", "", c.cur)}
				})
				h := r.h
				showSource(t, h)
				openDiffOf(t, h, "f:changes:main.go")
				f := h.frame()
				if c.want != "" && !strings.Contains(f, c.want) || c.want == "" && strings.Contains(f, "no newline") {
					t.Errorf("want %q:\n%s", c.want, f)
				}
			})
		})
	}
}

// TestDiffCompact is criterion 44.9 where the panel takes zone B's place
// (design D8): at 120x40 the diff takes it with the panel docked beside it,
// F6 reaching both, F3 putting the Explorer there, the diff kept; at 110
// columns the diff would keep under 48 columns: it takes the panel's place,
// Esc comes back to the panel, F3 there shows the Explorer and closes the
// diff; under 100 columns the diff takes the screen's width.
func TestDiffCompact(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r, _, _ := sourceRig(t)
		h := r.h
		h.send(tea.WindowSizeMsg{Width: 120, Height: 40})
		showSource(t, h)
		if !render.PanelCompact(h.m.State(), h.m.ui) {
			t.Fatal("not compact at 120x40")
		}
		openDiffOf(t, h, "f:changes:main.go")
		if sc := screenNow(h); sc.Diff != (render.Cols{X: 36, W: 50}) || sc.Panel != (render.Cols{X: 86, W: 26}) ||
			!slices.Equal(h.m.zonesWith(), []string{"D", "R", "C", "A"}) || !strings.Contains(h.frame(), "SOURCE CONTROL") {
			t.Errorf("the diff at 120x40: %+v, panel %+v, zones %q", sc.Diff, sc.Panel, h.m.zonesWith())
		}
		h.send(keyOf(tea.KeyF6))
		if h.m.ui.Focus != "R" || !h.m.ui.Diff.Open {
			t.Errorf("F6 from the diff: focus %q, diff %v", h.m.ui.Focus, h.m.ui.Diff.Open)
		}
		h.send(keyOf(tea.KeyF3))
		if !h.m.ui.Diff.Open || !h.m.ui.Explorer.Open || h.m.ui.Focus != "R" || !strings.Contains(h.frame(), "EXPLORER") {
			t.Errorf("F3 beside the diff: diff %v, Explorer %v, focus %q", h.m.ui.Diff.Open, h.m.ui.Explorer.Open, h.m.ui.Focus)
		}
		showSource(t, h)
		h.send(keyOf(tea.KeyEnter)) // main.go's diff again, its focus

		h.send(tea.WindowSizeMsg{Width: 110, Height: 40})
		if sc := screenNow(h); h.m.ui.Focus != "D" || sc.Diff != (render.Cols{X: 36, W: 66}) || sc.Panel.W != 0 ||
			!slices.Equal(h.m.zonesWith(), []string{"D", "C", "A"}) {
			t.Errorf("the diff at 110x40: focus %q, %+v, panel %+v, zones %q", h.m.ui.Focus, sc.Diff, sc.Panel, h.m.zonesWith())
		}
		h.send(keyOf(tea.KeyEscape))
		if h.m.ui.Diff.Open || h.m.ui.Focus != "R" || !strings.Contains(h.frame(), "SOURCE CONTROL") {
			t.Errorf("Esc: diff %v, focus %q", h.m.ui.Diff.Open, h.m.ui.Focus)
		}
		h.send(keyOf(tea.KeyEnter))
		h.send(keyOf(tea.KeyF3))
		if h.m.ui.Diff.Open || !h.m.ui.Explorer.Open || h.m.ui.Focus != "R" || !strings.Contains(h.frame(), "EXPLORER") {
			t.Errorf("F3 over the diff: diff %v, Explorer %v, focus %q", h.m.ui.Diff.Open, h.m.ui.Explorer.Open, h.m.ui.Focus)
		}
		// Under 100 columns no sidebar: the panel shown wider, then narrowed,
		// takes the screen, the diff too.
		showSource(t, h)
		h.send(tea.WindowSizeMsg{Width: 80, Height: 24})
		openDiffOf(t, h, "f:changes:main.go")
		if screenNow(h).Diff != (render.Cols{X: 0, W: 80}) {
			t.Errorf("the diff at 80x24: %+v", h.m.screen.Diff)
		}
	})
}

// TestDiffApproval is criterion 44.12's approvals beside the diff (design
// D12): a waiting approval is said first in the footer while the diff has
// the focus; F2, from the diff or from the panel beside it, closes the diff
// and gives the approval's card the focus in zone B, back in its place.
func TestDiffApproval(t *testing.T) {
	for _, from := range []string{"D", "R"} {
		t.Run(from, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				r, _, _ := sourceRig(t)
				h := r.h
				showSource(t, h)
				openDiffOf(t, h, "f:changes:main.go")
				r.emit(event.ApprovalRequest, event.Approval{ID: "ap1", Tool: "Bash", ToolID: "t1",
					Input: toolInput(t, map[string]any{"command": "go test ./..."})})
				h.settle()
				if f := lastRow(h); !strings.Contains(f, "approval waiting · F2 to answer") {
					t.Errorf("the footer while the diff has the focus: %q", f)
				}
				if from == "R" {
					h.send(keyOf(tea.KeyF6))
				}
				h.send(keyOf(tea.KeyF2))
				if h.m.ui.Diff.Open || h.m.ui.Card != "ap1" || h.m.ui.Focus != "B" || screenNow(h).B.W == 0 {
					t.Errorf("F2 from %s: diff %v, card %q, focus %q, zone B %d", from, h.m.ui.Diff.Open, h.m.ui.Card, h.m.ui.Focus, h.m.screen.B.W)
				}
			})
		})
	}
}

// TestSourceReadOnly is criterion 44.10 through the application (design
// D10): showing, refreshing, opening, Tab, n, f, r and Esc ask git only its
// status and the sides of a file, never anything that writes.
func TestSourceReadOnly(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r, g, _ := sourceRig(t)
		h := r.h
		showSource(t, h)
		h.send(typed('r'))
		openDiffOf(t, h, "f:changes:src/Main.java")
		for _, k := range []tea.Msg{keyOf(tea.KeyTab), typed('n'), typed('f'), typed('r'), keyOf(tea.KeyEscape), keyOf(tea.KeyDown), keyOf(tea.KeyEnter)} {
			h.send(k)
		}
		for _, c := range g.calls {
			if !strings.HasPrefix(c, "status ") && !strings.HasPrefix(c, "sides ") {
				t.Errorf("git asked %q", c)
			}
		}
	})
}

// TestSourceHover is criterion 44.4's hover (design D5): a motion onto a row
// hovers it, the filter passing only a motion that changes the hovered
// row; the cursor stays.
func TestSourceHover(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r, _, _ := sourceRig(t)
		h := r.h
		showSource(t, h)
		cur := h.m.ui.Source.Cursor
		x, y := at(t, h, "gone.txt", 0)
		mouseAt(t, h, moveTo(x, y))
		rows := render.SourceRows(h.m.ui.Source)
		if hv := h.m.ui.Source.Hover; hv < 1 || rows[hv-1].Path != "gone.txt" || h.m.ui.Source.Cursor != cur {
			t.Fatalf("hover %d, cursor %q", hv, h.m.ui.Source.Cursor)
		}
		dropped := h.m.motionDropped
		mouseAt(t, h, moveTo(x+2, y))
		if h.m.motionDropped != dropped+1 {
			t.Error("a motion within the row was not dropped")
		}
		mouseAt(t, h, moveTo(5, 5))
		if h.m.ui.Source.Hover != 0 {
			t.Errorf("off the panel: hover %d", h.m.ui.Source.Hover)
		}
	})
}
