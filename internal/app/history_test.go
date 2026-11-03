package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/Evengelius/agent-command-center/internal/input"
	"github.com/Evengelius/agent-command-center/internal/provider"
	"github.com/Evengelius/agent-command-center/internal/provider/claude"
	"github.com/Evengelius/agent-command-center/internal/provider/codex"
	"github.com/Evengelius/agent-command-center/internal/render"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// Plan 75, step 1: the session history on the home screen. Criteria 75.1 to
// 75.9 at the application's level, over the real adapters and the fake CLIs
// of plan 69's tests.

// histEntries are n entries of project history, the newest ended 3 minutes
// before testNow, one a minute older each, Claude Code's, the third one
// Codex's, the second without a title.
func histEntries(n int) []input.Entry {
	var out []input.Entry
	for i := range n {
		e := input.Entry{CLI: claude.ProviderID, ID: fmt.Sprintf("c-%02d-%08d", i, 21000000+i), Ended: testNow.Add(-time.Duration(3+i) * time.Minute),
			Model: "Claude Haiku 4.5", Title: fmt.Sprintf("Title of session %d", i)}
		if i%3 == 2 {
			e.CLI, e.ID, e.Model = codex.ProviderID, fmt.Sprintf("x-%02d-%08d", i, 21000000+i), "GPT-5.5"
		}
		if i%3 == 1 {
			e.Title = ""
		}
		out = append(out, e)
	}
	return out
}

// histFile is the file a store holds for list, as written.
func histFile(t *testing.T, list []input.Project) string {
	t.Helper()
	ps := &input.Projects{Dir: t.TempDir()}
	if err := ps.Save(list); err != nil {
		t.Fatal(err)
	}
	return histRead(t, ps)
}

func histRead(t *testing.T, ps *input.Projects) string {
	t.Helper()
	f, err := ps.File()
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(f)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// histRig is the home screen at 120x40 on a store holding list, dir in the
// field, UTC for the times; both fake CLIs installed.
func histRig(t *testing.T, list []input.Project, dir string) (*harness, *wireClaude, *wireCodex, *input.Projects) {
	t.Helper()
	ps := storeWith(t, list)
	wc, wx := &wireClaude{}, &wireCodex{thread: "th-new"}
	h := homeRig(t, wc, wx, ps, dir)
	h.m.opt.Location = time.UTC
	return h, wc, wx, ps
}

func openHist(t *testing.T, h *harness) {
	t.Helper()
	h.send(key('h', tea.ModAlt))
	if h.m.hist == nil || !h.m.ui.History.Open {
		t.Fatalf("Alt+H did not open the view:\n%s", h.frame())
	}
}

// histCell is a cell of the view's panel carrying item.
func histCell(t *testing.T, h *harness, item int) (int, int) {
	t.Helper()
	sc := screenNow(h)
	for y := range h.m.ui.Height {
		for x := range h.m.ui.Width {
			if p, it := sc.ItemAt(x, y); p == render.HistoryPanel && it == item {
				return x, y
			}
		}
	}
	t.Fatalf("no cell for history item %#x:\n%s", item, h.frame())
	return 0, 0
}

func histProjects(dir, other string) []input.Project {
	return []input.Project{
		{Path: dir, Used: testNow.Add(-time.Hour), History: histEntries(6)},
		{Path: other, Used: testNow.Add(-2 * time.Hour), History: []input.Entry{{CLI: claude.ProviderID, ID: "other-0001", Ended: testNow.Add(-time.Hour), Title: "ANOTHER PROJECT"}}},
	}
}

// TestHistoryOpen is criterion 75.3: Alt+H opens the view for the project
// field's path as Start resolves it, full screen, showing that project's
// sessions only; it opens nothing while a start is validated, from the
// Worktrees view, or with a path Start refuses, whose refusal the field
// shows.
func TestHistoryOpen(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		dir, other := t.TempDir(), t.TempDir()
		h, _, _, _ := histRig(t, histProjects(dir, other), dir)
		openHist(t, h)
		f := h.frame()
		for _, want := range []string{"SESSION HISTORY", "Title of session 0", "Title of session 5"} {
			if !strings.Contains(f, want) {
				t.Errorf("the view lacks %q:\n%s", want, f)
			}
		}
		for _, no := range []string{"ANOTHER PROJECT", "NEW SESSION", "PROJECTS", "Shortcuts"} {
			if strings.Contains(f, no) {
				t.Errorf("the view shows %q:\n%s", no, f)
			}
		}
		if h.m.hist.project != dir {
			t.Errorf("the view's project %q, want %q", h.m.hist.project, dir)
		}
		h.send(key(tea.KeyEscape, 0))
		if h.m.hist != nil || !strings.Contains(h.frame(), "NEW SESSION") {
			t.Errorf("Esc did not close the view:\n%s", h.frame())
		}
		if h.m.State().Screen != state.ScreenStartup {
			t.Error("Esc left the startup screen")
		}

		// The path as Start resolves it: the other project's, in capitals and
		// with slashes on Windows.
		h.m.path.SetText(other)
		openHist(t, h)
		if f := h.frame(); !strings.Contains(f, "ANOTHER PROJECT") || strings.Contains(f, "Title of session 0") {
			t.Errorf("the other project:\n%s", f)
		}
		h.send(key(tea.KeyEscape, 0))
		h.m.path.SetText(dir + string(os.PathSeparator) + ".")
		openHist(t, h)
		if !strings.Contains(h.frame(), "Title of session 0") {
			t.Errorf("a path with a dot:\n%s", h.frame())
		}
		h.send(key(tea.KeyEscape, 0))

		// A path Start refuses: nothing opens and the field shows why.
		h.m.path.SetText("~nobody/x")
		h.send(key('h', tea.ModAlt))
		if h.m.hist != nil || h.m.ui.PathProblem == "" || !strings.Contains(h.frame(), "✗ Another user's home is not expanded") {
			t.Errorf("a refused path: view %v, problem %q:\n%s", h.m.hist, h.m.ui.PathProblem, h.frame())
		}
		h.m.path.SetText(dir)

		// While a start is validated.
		h.m.do(state.Confirm{Provider: claude.ProviderID, Project: dir})
		if h.m.State().Startup.Phase != state.Validating {
			t.Fatalf("phase %q", h.m.State().Startup.Phase)
		}
		h.send(key('h', tea.ModAlt))
		if h.m.hist != nil {
			t.Error("the view opened while a start is validated")
		}
	})
	synctest.Test(t, func(t *testing.T) {
		// From the Worktrees view.
		dir := t.TempDir()
		h, _, _, _ := histRig(t, histProjects(dir, t.TempDir()), dir)
		h.m.wt = &worktrees{dir: dir}
		h.send(key('h', tea.ModAlt))
		if h.m.hist != nil || h.m.wt == nil {
			t.Errorf("from the Worktrees view: history %v, worktrees %v", h.m.hist, h.m.wt)
		}
	})
}

// TestHistoryPicture is criterion 75.3's "nothing is drawn over the Sixel
// picture": with Sixel the view replaces the home screen's picture, which
// is drawn again, whole, when the view closes.
func TestHistoryPicture(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		if _, err := input.Glyph('H', 20, false); err != nil {
			t.Skipf("GDI: %v", err) // no font: the text screen, the golden frames'
		}
		dir := t.TempDir()
		h, _, _, _ := histRig(t, histProjects(dir, t.TempDir()), dir)
		h.send(tea.WindowSizeMsg{Width: 208, Height: 59})
		h.send(PrimaryDeviceAttributesEvent{61, 4, 6, 7, 14})
		if sc := screenNow(h); !sc.Home.Shown || !sc.Home.Look.Geom.Picture || !strings.Contains(h.m.thumbSequence(), "\x1bP") {
			t.Fatal("no picture on the home screen")
		}
		openHist(t, h)
		sc := screenNow(h)
		if seq := h.m.thumbSequence(); sc.Home.Shown || strings.Contains(seq, "\x1bP") || h.m.home.pixels != nil || h.m.home.banner != nil {
			t.Errorf("the picture is drawn under the view: shown %v, an image written %v", sc.Home.Shown, strings.Contains(seq, "\x1bP"))
		}
		h.send(key(tea.KeyEscape, 0))
		if sc := screenNow(h); !sc.Home.Shown || !strings.Contains(h.m.thumbSequence(), "\x1bP") {
			t.Error("the picture is not drawn again once the view is closed")
		}
	})
}

// TestHistoryRows is criterion 75.4: newest first, each entry's title and
// under it its date and time, age, CLI, model ("N/A" when not recorded) and
// the ID's first 8 characters; an entry without a title the grey line alone;
// a selection beyond the visible rows scrolls the list, with the scroll mark.
func TestHistoryRows(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		dir := t.TempDir()
		es := histEntries(4)
		es[0].ID = "21f3b775-162f-47e1-bb16-aa91f8a808b9"
		es[3].Model = "" // not recorded
		h, _, _, _ := histRig(t, []input.Project{{Path: dir, Used: testNow.Add(-time.Hour), History: es}}, dir)
		openHist(t, h)
		f := h.frame()
		rows := strings.Split(f, "\n")
		at := func(s string) int {
			for y, r := range rows {
				if strings.Contains(r, s) {
					return y
				}
			}
			t.Fatalf("no row with %q:\n%s", s, f)
			return -1
		}
		meta := func(row string) string { return strings.TrimSpace(strings.Trim(strings.TrimSpace(row), "│▏▕ ")) }
		y0 := at("Title of session 0")
		if m := meta(rows[y0+1]); m != "2026-10-15 09:27 · 3m ago · Claude Code · Claude Haiku 4.5 · 21f3b775" {
			t.Errorf("the first entry's grey line %q", m)
		}
		if y1 := at("2026-10-15 09:26"); y1 != y0+2 || strings.Contains(rows[y1-1], "Title") {
			t.Errorf("the entry without a title: its line at %d, the one above %q", y1, rows[y1-1])
		}
		if m := meta(rows[at("Title of session 2")+1]); !strings.Contains(m, "OpenAI Codex · GPT-5.5 · x-02-210") {
			t.Errorf("the Codex entry %q", m)
		}
		if m := meta(rows[at("Title of session 3")+1]); !strings.Contains(m, "Claude Code · N/A · c-03-210") {
			t.Errorf("no model recorded %q", m)
		}
		if y := at("Title of session 3"); y < y0 || at("Title of session 2") > y {
			t.Error("not newest first")
		}
		if strings.Contains(f, "▐") {
			t.Error("a scroll mark with everything in view")
		}
	})
	synctest.Test(t, func(t *testing.T) {
		// Twenty entries at 80x24: the selection past the rows scrolls the list.
		dir := t.TempDir()
		h, _, _, _ := histRig(t, []input.Project{{Path: dir, Used: testNow.Add(-time.Hour), History: histEntries(20)}}, dir)
		h.send(tea.WindowSizeMsg{Width: 80, Height: 24})
		openHist(t, h)
		if f := h.frame(); strings.Contains(f, "Title of session 19") || !strings.Contains(f, "▐") {
			t.Errorf("the top of the list:\n%s", f)
		}
		h.send(key(tea.KeyEnd, 0))
		f := h.frame()
		if !strings.Contains(f, "x-17-210") || strings.Contains(f, "Title of session 0") || !strings.Contains(f, "▐") {
			t.Errorf("the end of the list:\n%s", f)
		}
		// The selected entry is always drawn.
		for i := range 20 {
			h.send(key(tea.KeyHome, 0))
			for range i {
				h.send(key(tea.KeyDown, 0))
			}
			id := fmt.Sprintf("c-%02d-210", i)
			if i%3 == 2 {
				id = fmt.Sprintf("x-%02d-210", i)
			}
			if !strings.Contains(h.frame(), id) {
				t.Fatalf("entry %d not drawn:\n%s", i, h.frame())
			}
		}
	})
	synctest.Test(t, func(t *testing.T) {
		// No entry: the message, and nothing to delete.
		dir := t.TempDir()
		h, _, _, _ := histRig(t, []input.Project{{Path: dir, Used: testNow.Add(-time.Hour)}}, dir)
		openHist(t, h)
		if f := h.frame(); !strings.Contains(f, "No sessions recorded for this project yet") {
			t.Errorf("none:\n%s", f)
		}
		h.send(key(tea.KeyDelete, 0))
		h.send(key(tea.KeyDelete, tea.ModShift))
		h.send(key(tea.KeyEnter, 0))
		if h.m.hist == nil || h.m.hist.ask != 0 || h.m.State().Startup.Phase != state.Asking {
			t.Error("a key acted on an empty history")
		}
	})
}

// TestHistoryKeysAndMouse is criterion 75.5: each key and click of design D3
// acts; a right click changes nothing; Ctrl+N, Ctrl+O, Ctrl+R and Ctrl+W do
// nothing while the view is open and Ctrl+Q quits.
func TestHistoryKeysAndMouse(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		dir := t.TempDir()
		h, wc, wx, _ := histRig(t, []input.Project{{Path: dir, Used: testNow.Add(-time.Hour), History: histEntries(20)}}, dir)
		picks := 0
		h.m.opt.PickFolder = func(string) (string, error) { picks++; return "", nil }
		openHist(t, h)
		sel := func(want int, what string) {
			t.Helper()
			if h.m.hist == nil || h.m.hist.sel != want || h.m.ui.History.Sel != want {
				t.Fatalf("%s: selection %d, want %d", what, h.m.hist.sel, want)
			}
		}
		sel(0, "open")
		for _, c := range []struct {
			k    tea.KeyPressMsg
			want int
			name string
		}{
			{key(tea.KeyDown, 0), 1, "↓"}, {key(tea.KeyDown, 0), 2, "↓"}, {key(tea.KeyUp, 0), 1, "↑"}, {key(tea.KeyUp, 0), 0, "↑"}, {key(tea.KeyUp, 0), 0, "↑ at the top"},
			{key(tea.KeyPgDown, 0), 14, "PgDn"}, {key(tea.KeyPgDown, 0), 19, "PgDn at the end"}, {key(tea.KeyPgUp, 0), 5, "PgUp"}, {key(tea.KeyHome, 0), 0, "Home"},
			{key(tea.KeyEnd, 0), 19, "End"}, {key(tea.KeyDown, 0), 19, "↓ at the end"},
		} {
			h.send(c.k)
			sel(c.want, c.name)
		}
		h.send(key(tea.KeyHome, 0))

		// Ctrl+N, Ctrl+O, Ctrl+R and Ctrl+W do nothing.
		// The field away from the starting folder, question 2 focused, Codex
		// chosen: Ctrl+N would put all three back.
		away := t.TempDir()
		h.m.path.SetText(away)
		h.m.ui.Choice, h.m.ui.Field = codex.ProviderID, 1
		for _, r := range "norw" {
			h.send(ctrl(r))
			if h.m.hist == nil || h.m.wt != nil || picks != 0 || len(wc.spawns())+len(wx.spawns()) != 0 || h.m.path.Text() != away || h.m.ui.Choice != codex.ProviderID || h.m.ui.Field != 1 || h.m.State().Startup.Phase != state.Asking {
				t.Fatalf("Ctrl+%c: view %v, worktrees %v, %d dialogs, %d spawns, field %q, choice %q, focus %d", r, h.m.hist != nil, h.m.wt != nil, picks, len(wc.spawns())+len(wx.spawns()), h.m.path.Text(), h.m.ui.Choice, h.m.ui.Field)
			}
		}
		h.send(tea.PasteMsg{Content: "pasted text"})
		if h.m.path.Text() != away {
			t.Errorf("a paste reached the field: %q", h.m.path.Text())
		}
		h.m.path.SetText(dir) // the project the view is about, for the resume below

		// A click selects, a right click changes nothing.
		x, y := histCell(t, h, render.HistoryItemRow+3)
		mouseClick(t, h, x, y)
		sel(3, "a click")
		x5, y5 := histCell(t, h, render.HistoryItemRow+5)
		mouseAt(t, h, tea.MouseClickMsg{X: x5, Y: y5, Button: tea.MouseRight})
		mouseAt(t, h, tea.MouseClickMsg{X: 0, Y: 0, Button: tea.MouseRight})
		sel(3, "a right click")
		if h.m.hist == nil || h.m.hist.ask != 0 {
			t.Error("a right click did something")
		}
		// The wheel: one entry a notch, over the box only.
		wheel := func(x, y int, b tea.MouseButton) { mouseAt(t, h, tea.MouseWheelMsg{X: x, Y: y, Button: b}) }
		wheel(x, y, tea.MouseWheelDown)
		sel(4, "wheel down")
		wheel(x, y, tea.MouseWheelDown)
		wheel(x, y, tea.MouseWheelUp)
		sel(4, "down, down, up")
		wheel(0, 0, tea.MouseWheelDown)
		sel(4, "wheel outside the box")

		// A double click resumes.
		x, y = histCell(t, h, render.HistoryItemRow+0)
		mouseClick(t, h, x, y)
		mouseClick(t, h, x, y)
		if h.m.hist != nil || len(wc.spawns()) != 1 || resumedArg(wc.spawns()[0]) != "c-00-21000000" {
			t.Errorf("a double click: view %v, spawns %q", h.m.hist != nil, wc.spawns())
		}
	})
	synctest.Test(t, func(t *testing.T) {
		// A click outside the box closes the view; Esc too; Ctrl+Q quits.
		dir := t.TempDir()
		h, _, _, _ := histRig(t, histProjects(dir, t.TempDir()), dir)
		openHist(t, h)
		mouseClick(t, h, 0, h.m.ui.Height-1)
		if h.m.hist != nil || h.quit {
			t.Errorf("a click outside: view %v, quit %v", h.m.hist != nil, h.quit)
		}
		openHist(t, h)
		x, y := histCell(t, h, render.HistoryItemRow+1)
		mouseClick(t, h, x, y)
		if h.m.hist == nil {
			t.Error("a click inside the box closed the view")
		}
		h.send(ctrl('q'))
		if !h.quit {
			t.Error("Ctrl+Q did not quit with the view open")
		}
	})
}

// TestHistoryResume is criterion 75.6: Enter starts the entry's CLI on that
// session, question 1 then showing that CLI; Claude Code with --resume,
// Codex with thread/resume without its turns; a CLI not installed starts
// nothing and says so; a refused resume removes the entry and Ctrl+R then
// offers the CLI's previous one.
func TestHistoryResume(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		dir := t.TempDir()
		es := histEntries(3) // Claude Code, Claude Code (no title), Codex
		h, wc, wx, _ := histRig(t, []input.Project{{Path: dir, Used: testNow.Add(-time.Hour), History: es}}, dir)
		h.m.ui.Choice = codex.ProviderID
		openHist(t, h)
		h.send(key(tea.KeyDown, 0)) // the second, Claude Code's
		h.send(key(tea.KeyEnter, 0))
		s := h.m.State()
		if sp := wc.spawns(); len(sp) != 1 || resumedArg(sp[0]) != es[1].ID || len(wx.spawns()) != 0 {
			t.Fatalf("Enter spawned claude %q, codex %q", wc.spawns(), wx.spawns())
		}
		if h.m.hist != nil || h.m.ui.Choice != claude.ProviderID || s.Startup.Provider != claude.ProviderID || s.Session.Resumed != es[1].ID || s.Screen != state.ScreenDashboard {
			t.Errorf("view %v, question 1 %q, provider %q, resumed %q, screen %q", h.m.hist != nil, h.m.ui.Choice, s.Startup.Provider, s.Session.Resumed, s.Screen)
		}
	})
	synctest.Test(t, func(t *testing.T) {
		dir := t.TempDir()
		es := histEntries(3)
		h, wc, wx, _ := histRig(t, []input.Project{{Path: dir, Used: testNow.Add(-time.Hour), History: es}}, dir)
		openHist(t, h)
		h.send(key(tea.KeyEnd, 0)) // Codex's
		h.send(key(tea.KeyEnter, 0))
		ls := wx.method("thread/resume")
		if len(ls) != 1 || !strings.Contains(ls[0], `"threadId":"`+es[2].ID+`"`) || !strings.Contains(ls[0], `"excludeTurns":true`) || len(wx.method("thread/start")) != 0 || len(wc.spawns()) != 0 {
			t.Fatalf("thread/resume %q, thread/start %q, claude %q", ls, wx.method("thread/start"), wc.spawns())
		}
		if h.m.ui.Choice != codex.ProviderID {
			t.Errorf("question 1 shows %q", h.m.ui.Choice)
		}
	})
	synctest.Test(t, func(t *testing.T) {
		// Codex not installed: the row is dimmed and Enter says so.
		dir := t.TempDir()
		ps := storeWith(t, []input.Project{{Path: dir, Used: testNow.Add(-time.Hour), History: histEntries(3)}})
		wc := &wireClaude{}
		h := newHarness(t, Options{Providers: []provider.Provider{
			installedClaude{&claude.Provider{Spawner: wc, Now: func() time.Time { return testNow }}},
			fakeCLI{provider.Detection{ID: codex.ProviderID, Name: codex.Name, Problem: "not found on the PATH"}},
		}, Cwd: dir, Projects: ps})
		h.settle()
		h.send(tea.WindowSizeMsg{Width: 120, Height: 40})
		openHist(t, h)
		if rows := h.m.ui.History.Rows; rows[0].Dim || rows[1].Dim || !rows[2].Dim {
			t.Errorf("dimmed rows: %v %v %v", rows[0].Dim, rows[1].Dim, rows[2].Dim)
		}
		h.send(key(tea.KeyEnd, 0))
		h.send(key(tea.KeyEnter, 0))
		if h.m.hist == nil || len(wc.spawns()) != 0 || h.m.State().Startup.Phase != state.Asking || !strings.Contains(h.frame(), "✗ OpenAI Codex is not installed") {
			t.Errorf("Enter on a Codex entry: view %v, spawns %q, phase %q:\n%s", h.m.hist != nil, wc.spawns(), h.m.State().Startup.Phase, h.frame())
		}
		h.send(key(tea.KeyUp, 0))
		if h.m.ui.History.Note != "" {
			t.Errorf("the note stays after a move: %q", h.m.ui.History.Note)
		}
	})
	synctest.Test(t, func(t *testing.T) {
		// A refused resume: the entry goes, and Ctrl+R offers the one before.
		dir := t.TempDir()
		es := []input.Entry{
			{CLI: claude.ProviderID, ID: "s-gone", Ended: testNow.Add(-time.Hour), Title: "refused"},
			{CLI: claude.ProviderID, ID: "s-older", Ended: testNow.Add(-2 * time.Hour), Title: "kept"},
		}
		ps := storeWith(t, []input.Project{{Path: dir, Used: testNow.Add(-time.Hour), History: es}})
		w := &wireClaude{noInit: true}
		h := homeRig(t, w, &wireCodex{}, ps, dir)
		h.m.opt.Location = time.UTC
		openHist(t, h)
		h.send(key(tea.KeyEnter, 0))
		p := w.last()
		p.emit(`{"type":"result","subtype":"error_during_execution","is_error":true,"num_turns":0,"session_id":"s-gone","errors":["No conversation found with session ID: s-gone"]}`)
		p.exit(1, "No conversation found with session ID: s-gone\n")
		h.settle()
		if s := h.m.State(); s.Startup.Phase != state.StartFailed || !s.Startup.ResumeGone {
			t.Fatalf("startup %q, gone %t", s.Startup.Phase, s.Startup.ResumeGone)
		}
		h.send(key(tea.KeyEscape, 0))
		list, err := ps.Load()
		if err != nil || len(list) != 1 || len(list[0].History) != 1 || list[0].History[0].ID != "s-older" {
			t.Fatalf("the store after the refusal: %+v, %v", list, err)
		}
		if want := "Claude Code's last session here: 2h ago · Ctrl+R continues it"; h.m.ui.LastSession != want {
			t.Errorf("the offer %q, want %q", h.m.ui.LastSession, want)
		}
		h.send(ctrl('r'))
		if sp := w.spawns(); len(sp) != 2 || resumedArg(sp[1]) != "s-older" {
			t.Errorf("Ctrl+R after the refusal spawned %q", sp)
		}
	})
}

// TestHistoryCtrlR is criteria 75.2 and 75.1's use: Ctrl+R resumes the
// chosen CLI's newest entry, never the other CLI's; after the newest is
// deleted, the previous one.
func TestHistoryCtrlR(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		dir := t.TempDir()
		es := []input.Entry{
			{CLI: codex.ProviderID, ID: "x-newest", Ended: testNow.Add(-time.Minute)},
			{CLI: claude.ProviderID, ID: "c-new", Ended: testNow.Add(-5 * time.Minute)},
			{CLI: claude.ProviderID, ID: "c-old", Ended: testNow.Add(-time.Hour)},
		}
		h, wc, _, _ := histRig(t, []input.Project{{Path: dir, Used: testNow.Add(-time.Hour), History: es}}, dir)
		if want := "Claude Code's last session here: 5m ago · Ctrl+R continues it"; h.m.ui.LastSession != want {
			t.Fatalf("the offer %q, want %q", h.m.ui.LastSession, want)
		}
		// Delete the newest Claude Code entry from the view.
		openHist(t, h)
		h.send(key(tea.KeyDown, 0))
		h.send(key(tea.KeyDelete, 0))
		h.send(key(tea.KeyLeft, 0))
		h.send(key(tea.KeyEnter, 0))
		h.send(key(tea.KeyEscape, 0))
		if want := "Claude Code's last session here: 1h ago · Ctrl+R continues it"; h.m.ui.LastSession != want {
			t.Fatalf("after the deletion the offer is %q, want %q", h.m.ui.LastSession, want)
		}
		h.send(ctrl('r'))
		if sp := wc.spawns(); len(sp) != 1 || resumedArg(sp[0]) != "c-old" {
			t.Errorf("Ctrl+R spawned %q", sp)
		}
	})
}

// TestHistoryDelete is criterion 75.7: Del then Delete removes that entry
// only; Shift+Del then Delete empties that project's history; every other
// project, the favourites and their order stay byte for byte, the file being
// compared; Cancel, Esc or a click elsewhere removes nothing, and Cancel is
// selected when the question opens.
func TestHistoryDelete(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		dir, other, fav := t.TempDir(), t.TempDir(), t.TempDir()
		es := histEntries(6)
		list := []input.Project{
			{Path: fav, Used: testNow.Add(-5 * time.Hour), Favourite: true, History: []input.Entry{{CLI: es[0].CLI, ID: es[0].ID, Ended: es[0].Ended, Title: "the favourite's"}}},
			{Path: dir, Used: testNow.Add(-time.Hour), History: es},
			{Path: other, Used: testNow.Add(-2 * time.Hour), History: slices.Clone(es[:2])},
		}
		h, _, _, ps := histRig(t, list, dir)
		start := histRead(t, ps)
		if start != histFile(t, list) {
			t.Fatal("the store is not what the test saved")
		}
		without := func(i int) []input.Project { // list without entry i of dir's history
			out := slices.Clone(list)
			out[1].History = slices.Delete(slices.Clone(es), i, i+1)
			return out
		}
		openHist(t, h)

		// Del: the question, Cancel selected, nothing removed by Cancel, Esc, a
		// click elsewhere, or Enter on the default.
		h.send(key(tea.KeyDown, 0)) // the second entry
		h.send(key(tea.KeyDelete, 0))
		if h.m.hist.ask != render.HistoryAskOne || h.m.hist.askSel != 0 {
			t.Fatalf("the question: ask %q, selected %d", h.m.hist.ask, h.m.hist.askSel)
		}
		f := h.frame()
		for _, want := range []string{"DELETE THIS SESSION FROM THE HISTORY?", "ACC's list only: Claude Code keeps its own record", "Delete", "Cancel"} {
			if !strings.Contains(f, want) {
				t.Errorf("the question lacks %q:\n%s", want, f)
			}
		}
		h.send(key(tea.KeyEnter, 0)) // Cancel
		if h.m.hist == nil || h.m.hist.ask != 0 || histRead(t, ps) != start || h.m.hist.sel != 1 {
			t.Errorf("Enter on Cancel: ask %q, the file changed %t", h.m.hist.ask, histRead(t, ps) != start)
		}
		h.send(key(tea.KeyDelete, 0))
		h.send(key(tea.KeyLeft, 0)) // Delete selected: Esc cancels all the same
		h.send(key(tea.KeyEscape, 0))
		if h.m.hist == nil || h.m.hist.ask != 0 || histRead(t, ps) != start {
			t.Errorf("Esc: view %v, ask %q", h.m.hist != nil, h.m.hist.ask)
		}
		h.send(key(tea.KeyDelete, 0))
		x, y := histCell(t, h, render.HistoryItemCancel)
		mouseClick(t, h, x, y)
		if h.m.hist.ask != 0 || histRead(t, ps) != start {
			t.Error("a click on Cancel removed something or kept the question")
		}
		h.send(key(tea.KeyDelete, 0))
		mouseClick(t, h, 0, h.m.ui.Height-1)
		if h.m.hist == nil || h.m.hist.ask != 0 || histRead(t, ps) != start {
			t.Errorf("a click elsewhere: view %v, ask %q", h.m.hist != nil, h.m.hist.ask)
		}

		// Delete: that entry only, the file as it would be written without it.
		// The arrows go the way they point, Delete drawn on the left and
		// Cancel on the right: → keeps Cancel, ← selects Delete and keeps it.
		h.send(key(tea.KeyDelete, 0))
		h.send(key(tea.KeyRight, 0))
		if h.m.hist.askSel != 0 {
			t.Fatal("→ left Cancel")
		}
		h.send(key(tea.KeyLeft, 0))
		if h.m.hist.askSel != 1 {
			t.Fatal("← did not select Delete")
		}
		h.send(key(tea.KeyLeft, 0))
		if h.m.hist.askSel != 1 {
			t.Fatal("a second ← left Delete")
		}
		h.send(key(tea.KeyRight, 0))
		if h.m.hist.askSel != 0 {
			t.Fatal("→ did not select Cancel")
		}
		h.send(key(tea.KeyLeft, 0))
		h.send(key(tea.KeyEnter, 0))
		if got, want := histRead(t, ps), histFile(t, without(1)); got != want {
			t.Errorf("after Delete:\n%s\nwant\n%s", got, want)
		}
		if h.m.hist.ask != 0 || h.m.hist.sel != 1 || strings.Contains(h.frame(), "c-01-210") {
			t.Errorf("after Delete: ask %q, selection %d", h.m.hist.ask, h.m.hist.sel)
		}

		// A click on Delete.
		h.send(key(tea.KeyDelete, 0))
		x, y = histCell(t, h, render.HistoryItemDelete)
		mouseClick(t, h, x, y)
		want := slices.Clone(list)
		want[1].History = slices.Delete(slices.Clone(es), 1, 3) // the entries at 1 and 2 went; the third is now selected
		if got := histRead(t, ps); got != histFile(t, want) {
			t.Errorf("after a click on Delete:\n%s\nwant\n%s", got, histFile(t, want))
		}

		// Shift+Del: the question names the project and the CLIs; Delete empties.
		h.send(key(tea.KeyDelete, tea.ModShift))
		f = h.frame()
		if h.m.hist.ask != render.HistoryAskAll || h.m.hist.askSel != 0 || !strings.Contains(f, "DELETE ALL 4 SESSIONS OF ") || !strings.Contains(f, "ACC's list only: Claude Code and OpenAI Codex keep their own records") {
			t.Fatalf("Delete All's question: ask %q, selected %d:\n%s", h.m.hist.ask, h.m.hist.askSel, f)
		}
		h.send(key(tea.KeyEnter, 0))
		if histRead(t, ps) != histFile(t, want) {
			t.Error("Enter on Cancel removed something")
		}
		h.send(key(tea.KeyDelete, tea.ModShift))
		h.send(key(tea.KeyLeft, 0))
		h.send(key(tea.KeyEnter, 0))
		want[1].History = nil
		if got := histRead(t, ps); got != histFile(t, want) {
			t.Errorf("after Delete All:\n%s\nwant\n%s", got, histFile(t, want))
		}
		if !strings.Contains(h.frame(), "No sessions recorded for this project yet") {
			t.Errorf("after Delete All:\n%s", h.frame())
		}
		got, err := ps.Load()
		if err != nil || len(got) != 3 || got[0].Path != fav || !got[0].Favourite || len(got[0].History) != 1 || len(got[2].History) != 2 {
			t.Errorf("the other projects: %+v, %v", got, err)
		}
	})
}

// TestHistoryRecord is criterion 75.1 at the application's level: a session
// started new is recorded when the owner's turn ends with the model zone A
// names and the title of its first message; the same ID again moves first
// with its title kept; a resumed session records itself with no title of
// its own and keeps the one it had.
func TestHistoryRecord(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		dir := t.TempDir()
		ps := &input.Projects{Dir: t.TempDir()}
		w := &wireClaude{}
		h := homeRig(t, w, &wireCodex{}, ps, dir)
		if _, err := ps.Update(func(l []input.Project) []input.Project { return ps.Record(l, dir, testNow.Add(-time.Hour)) }); err != nil {
			t.Fatal(err)
		}
		h.send(key(tea.KeyEnter, 0))
		w.last().emit(claudeInit(dir, "s-first"))
		h.settle()
		h.send(tea.PasteMsg{Content: "  remember\n\tthe code   word \x1b[31m now  "})
		h.send(key(tea.KeyEnter, 0))
		w.last().emit(claudeTurn("s-first", "OK")...)
		h.settle()
		list, err := ps.Load()
		if err != nil || len(list) != 1 || len(list[0].History) != 1 {
			t.Fatalf("the store: %+v, %v", list, err)
		}
		if e := list[0].History[0]; e.CLI != claude.ProviderID || e.ID != "s-first" || !e.Ended.Equal(testNow) || e.Model != "Claude Haiku 4.5" || e.Title != "remember the code word [31m now" {
			t.Errorf("the entry: %+v", e)
		}
		// A second message: the title stays.
		h.send(tea.PasteMsg{Content: "and the other thing"})
		h.send(key(tea.KeyEnter, 0))
		w.last().emit(claudeTurn("s-first", "TWO")...)
		h.settle()
		list, _ = ps.Load()
		if e := list[0].History[0]; len(list[0].History) != 1 || e.Title != "remember the code word [31m now" {
			t.Errorf("after a second message: %+v", list[0].History)
		}
		h.crash()

		// Another session, then the first one resumed: it moves first, its title
		// kept, and a session resumed from the view gets none of its own.
		w2 := &wireClaude{}
		h2 := homeRig(t, w2, &wireCodex{}, ps, dir)
		h2.send(key(tea.KeyEnter, 0))
		w2.last().emit(claudeInit(dir, "s-second"))
		h2.settle()
		h2.send(tea.PasteMsg{Content: "second session"})
		h2.send(key(tea.KeyEnter, 0))
		w2.last().emit(claudeTurn("s-second", "OK")...)
		h2.settle()
		h2.crash()
		if list, _ = ps.Load(); len(list[0].History) != 2 || list[0].History[0].ID != "s-second" || list[0].History[1].ID != "s-first" {
			t.Fatalf("two sessions: %+v", list[0].History)
		}

		w3 := &wireClaude{}
		h3 := homeRig(t, w3, &wireCodex{}, ps, dir)
		h3.m.opt.Location = time.UTC
		openHist(t, h3)
		h3.send(key(tea.KeyDown, 0)) // s-first
		h3.send(key(tea.KeyEnter, 0))
		if sp := w3.spawns(); len(sp) != 1 || resumedArg(sp[0]) != "s-first" {
			t.Fatalf("spawned %q", sp)
		}
		w3.last().emit(claudeInit(dir, "s-first"))
		h3.settle()
		h3.send(tea.PasteMsg{Content: "a different first message"})
		h3.send(key(tea.KeyEnter, 0))
		w3.last().emit(claudeTurn("s-first", "THREE")...)
		h3.settle()
		list, _ = ps.Load()
		if len(list[0].History) != 2 || list[0].History[0].ID != "s-first" || list[0].History[0].Title != "remember the code word [31m now" || list[0].History[1].Title != "second session" {
			t.Errorf("after a resume: %+v", list[0].History)
		}
	})
	synctest.Test(t, func(t *testing.T) {
		// A migrated entry has no title and a resume gives it none.
		dir := t.TempDir()
		ps := &input.Projects{Dir: t.TempDir()}
		f, _ := ps.File()
		if err := os.MkdirAll(strings.TrimSuffix(f, "projects.json"), 0o700); err != nil {
			t.Fatal(err)
		}
		v1 := `{"version":1,"projects":[{"path":` + quote(dir) + `,"used":"2026-10-15T08:30:00Z","sessions":{"claude-code":{"id":"s-v1","ended":"2026-10-15T08:30:00Z"}}}]}`
		if err := os.WriteFile(f, []byte(v1), 0o600); err != nil {
			t.Fatal(err)
		}
		w := &wireClaude{}
		h := homeRig(t, w, &wireCodex{}, ps, dir)
		h.m.opt.Location = time.UTC
		openHist(t, h)
		if rows := h.m.ui.History.Rows; len(rows) != 1 || rows[0].Title != "" || !strings.Contains(rows[0].Meta, "Claude Code · N/A · s-v1") {
			t.Fatalf("the migrated entry: %+v", rows)
		}
		if b := histRead(t, ps); b != v1 {
			t.Error("reading the file changed it")
		}
		h.send(key(tea.KeyEnter, 0))
		w.last().emit(claudeInit(dir, "s-v1"))
		h.settle()
		h.send(tea.PasteMsg{Content: "first words here"})
		h.send(key(tea.KeyEnter, 0))
		w.last().emit(claudeTurn("s-v1", "OK")...)
		h.settle()
		list, _ := ps.Load()
		if e := list[0].History[0]; e.ID != "s-v1" || e.Title != "" || e.Model != "Claude Haiku 4.5" || !strings.Contains(histRead(t, ps), `"version": 2`) {
			t.Errorf("the resumed migrated entry: %+v", e)
		}
	})
}

// TestHistoryStartRefuses is criterion 75.3 read literally: a path Start
// would refuse (a folder that is not there, a file) opens nothing and the
// field shows Start's own refusal; a valid folder opens the view; the
// refusal goes with the path it was for.
func TestHistoryStartRefuses(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		dir := t.TempDir()
		h, _, _, _ := histRig(t, histProjects(dir, t.TempDir()), dir)
		missing := filepath.Join(dir, "gone")
		file := filepath.Join(dir, "a-file.txt")
		if err := os.WriteFile(file, []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
		for _, c := range []struct{ name, path, want string }{
			{"a missing folder", missing, "✗ Does not exist:"},
			{"a file", file, "✗ Is a file, not a directory:"},
		} {
			h.m.path.SetText(c.path)
			h.m.ui.Field = 1
			h.send(key('h', tea.ModAlt))
			f := h.frame()
			if h.m.hist != nil || !strings.HasPrefix(h.m.ui.OpenProblem, strings.TrimPrefix(c.want, "✗ ")) || !strings.Contains(f, c.want) {
				t.Errorf("%s: view %v, refusal %q:\n%s", c.name, h.m.hist != nil, h.m.ui.OpenProblem, f)
			}
			// Typing in the field takes Start's refusal of the old path away.
			h.send(typed('x'))
			if h.m.ui.OpenProblem != "" {
				t.Errorf("%s: the refusal stays after the path changed", c.name)
			}
		}
		h.m.path.SetText(dir)
		openHist(t, h)
		if h.m.ui.OpenProblem != "" {
			t.Errorf("a refusal kept for a valid folder: %q", h.m.ui.OpenProblem)
		}
	})
}

// gatedClaude is Claude Code, installed, whose detection waits for gate.
type gatedClaude struct {
	installedClaude
	gate chan struct{}
}

func (g gatedClaude) Detect(ctx context.Context) provider.Detection {
	<-g.gate
	return g.installedClaude.Detect(ctx)
}

// TestHistoryDetecting is design D2's one exception, a start being validated:
// Alt+H opens during detection too; the rows are not dimmed then (which CLI is
// installed is not known), and Enter and a double click start nothing and say
// the CLIs are still being detected; once detection is done, Enter resumes.
func TestHistoryDetecting(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		dir := t.TempDir()
		wc := &wireClaude{}
		gate := make(chan struct{})
		es := histEntries(2)
		h := newHarness(t, Options{Providers: []provider.Provider{
			gatedClaude{installedClaude{&claude.Provider{Spawner: wc, Now: func() time.Time { return testNow }}}, gate},
		}, Cwd: dir, Projects: storeWith(t, []input.Project{{Path: dir, Used: testNow.Add(-time.Hour), History: es}})})
		h.m.opt.Location = time.UTC
		h.settle()
		if p := h.m.State().Startup.Phase; p != state.Detecting {
			t.Fatalf("phase %q, want Detecting", p)
		}
		openHist(t, h)
		for _, r := range h.m.ui.History.Rows {
			if r.Dim {
				t.Error("a row is dimmed while detection runs")
			}
		}
		h.send(key(tea.KeyEnter, 0))
		if h.m.hist == nil || len(wc.spawns()) != 0 || !strings.Contains(h.frame(), "✗ The agent CLIs are still being detected") {
			t.Errorf("Enter during detection: view %v, spawns %q: %s", h.m.hist != nil, wc.spawns(), h.frame())
		}
		x, y := histCell(t, h, render.HistoryItemRow+0)
		mouseClick(t, h, x, y)
		mouseClick(t, h, x, y)
		if h.m.hist == nil || len(wc.spawns()) != 0 {
			t.Errorf("a double click during detection: view %v, spawns %q", h.m.hist != nil, wc.spawns())
		}
		close(gate)
		h.settle()
		if h.m.State().Startup.Phase != state.Asking || h.m.hist == nil {
			t.Fatalf("after detection: phase %q, view %v", h.m.State().Startup.Phase, h.m.hist != nil)
		}
		if h.m.ui.History.Rows[0].Dim {
			t.Error("an installed CLI's row dimmed after detection")
		}
		h.send(key(tea.KeyEnter, 0))
		if sp := wc.spawns(); len(sp) != 1 || resumedArg(sp[0]) != es[0].ID {
			t.Errorf("Enter after detection spawned %q", sp)
		}
	})
}

// TestHistoryF12: F12 works in the view as on the home screen, the mouse
// given to the terminal and taken again; a view opened with it off has none.
func TestHistoryF12(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		dir := t.TempDir()
		h, _, _, _ := histRig(t, histProjects(dir, t.TempDir()), dir)
		openHist(t, h)
		if h.m.View().MouseMode != tea.MouseModeCellMotion {
			t.Fatal("the view has no mouse")
		}
		h.send(key(tea.KeyF12, 0))
		if !h.m.ui.MouseOff || h.m.View().MouseMode != tea.MouseModeNone || h.m.hist == nil {
			t.Errorf("F12: mouse off %v, mode %v, view %v", h.m.ui.MouseOff, h.m.View().MouseMode, h.m.hist != nil)
		}
		h.send(key(tea.KeyF12, 0))
		if h.m.ui.MouseOff || h.m.View().MouseMode != tea.MouseModeCellMotion {
			t.Error("F12 did not give the mouse back")
		}
		h.send(key(tea.KeyF12, 0))
		h.send(key(tea.KeyEscape, 0))
		openHist(t, h)
		if !h.m.ui.MouseOff || h.m.View().MouseMode != tea.MouseModeNone {
			t.Error("the view opened with the mouse off turned it on")
		}
	})
}

// TestHistoryCtrlRDoesNothing: with the field on a project that has a session
// of the chosen CLI, a Ctrl+R passed through would resume it; the view takes
// it.
func TestHistoryCtrlRDoesNothing(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		dir := t.TempDir()
		h, wc, _, _ := histRig(t, histProjects(dir, t.TempDir()), dir)
		if h.m.ui.Choice != claude.ProviderID || h.m.ui.LastSession == "" {
			t.Fatalf("the rig: choice %q, offer %q", h.m.ui.Choice, h.m.ui.LastSession)
		}
		openHist(t, h)
		h.send(ctrl('r'))
		if len(wc.spawns()) != 0 || h.m.hist == nil || h.m.State().Startup.Phase != state.Asking {
			t.Errorf("Ctrl+R: spawns %q, view %v, phase %q", wc.spawns(), h.m.hist != nil, h.m.State().Startup.Phase)
		}
		h.send(key(tea.KeyEscape, 0))
		h.send(ctrl('r'))
		if len(wc.spawns()) != 1 {
			t.Error("Ctrl+R on the home screen does not resume: the test proves nothing")
		}
	})
}

// TestHistoryRecordTitleBoundary: a first message whose 60th character ends a
// word, white space and more text after it, is stored as its first 60
// clusters.
func TestHistoryRecordTitleBoundary(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		dir := t.TempDir()
		ps := &input.Projects{Dir: t.TempDir()}
		w := &wireClaude{}
		h := homeRig(t, w, &wireCodex{}, ps, dir)
		if _, err := ps.Update(func(l []input.Project) []input.Project { return ps.Record(l, dir, testNow.Add(-time.Hour)) }); err != nil {
			t.Fatal(err)
		}
		h.send(key(tea.KeyEnter, 0))
		w.last().emit(claudeInit(dir, "s-long"))
		h.settle()
		h.send(tea.PasteMsg{Content: "Add a history view to the home screen with resume and delete entries please"})
		h.send(key(tea.KeyEnter, 0))
		w.last().emit(claudeTurn("s-long", "OK")...)
		h.settle()
		list, err := ps.Load()
		if err != nil || len(list) != 1 || len(list[0].History) != 1 {
			t.Fatalf("the store: %+v, %v", list, err)
		}
		want := "Add a history view to the home screen with resume and delete"
		if got := list[0].History[0].Title; got != want || len(got) != 60 {
			t.Errorf("the title %q, want %q", got, want)
		}
	})
}

// TestHistoryEntryWithoutID: a hand-edited file's entry with no ID is not
// listed, and Enter on the first listed entry resumes it.
func TestHistoryEntryWithoutID(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		dir := t.TempDir()
		es := histEntries(2)
		list := []input.Project{{Path: dir, Used: testNow.Add(-time.Hour), History: []input.Entry{{CLI: claude.ProviderID, Ended: testNow, Title: "NO ID"}, es[0], es[1]}}}
		h, wc, _, _ := histRig(t, list, dir)
		openHist(t, h)
		if f := h.frame(); strings.Contains(f, "NO ID") || len(h.m.ui.History.Rows) != 2 {
			t.Errorf("an entry without an ID is listed:\n%s", f)
		}
		h.send(key(tea.KeyEnter, 0))
		if sp := wc.spawns(); len(sp) != 1 || resumedArg(sp[0]) != es[0].ID {
			t.Errorf("Enter spawned %q", sp)
		}
	})
}

// TestHistoryDeleteAllExact: Delete All removes the entries its question
// counted, by CLI and ID, from the file as it is then: an entry another
// dashboard added meanwhile stays; and a question whose entries are no longer
// the list's is withdrawn.
func TestHistoryDeleteAllExact(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		dir := t.TempDir()
		es := histEntries(3)
		h, _, _, ps := histRig(t, []input.Project{{Path: dir, Used: testNow.Add(-time.Hour), History: es}}, dir)
		openHist(t, h)
		h.send(key(tea.KeyDelete, tea.ModShift))
		if f := h.frame(); !strings.Contains(f, "DELETE ALL 3 SESSIONS OF ") {
			t.Fatalf("the question:\n%s", f)
		}
		late := input.Entry{CLI: claude.ProviderID, ID: "late-0001", Ended: testNow, Title: "added meanwhile"}
		if _, err := ps.Update(func(l []input.Project) []input.Project { return ps.RecordSession(l, dir, late) }); err != nil {
			t.Fatal(err)
		}
		h.send(key(tea.KeyLeft, 0))
		h.send(key(tea.KeyEnter, 0))
		list, err := ps.Load()
		if err != nil || len(list) != 1 || len(list[0].History) != 1 || list[0].History[0].ID != "late-0001" {
			t.Errorf("after Delete All: %+v, %v", list, err)
		}
	})
	synctest.Test(t, func(t *testing.T) {
		// The list changes under an open question: it stays, Enter on Cancel
		// answers it and starts nothing; asked again and answered Delete, only
		// the counted entries that remain go.
		dir := t.TempDir()
		es := histEntries(3)
		h, wc, wx, ps := histRig(t, []input.Project{{Path: dir, Used: testNow.Add(-time.Hour), History: es}}, dir)
		openHist(t, h)
		h.send(key(tea.KeyDelete, tea.ModShift))
		h.m.projectList = ps.ForgetSession(h.m.projectList, dir, es[0].CLI, es[0].ID) // the list changes
		h.send(key(tea.KeyDown, 0))
		if h.m.hist.ask != render.HistoryAskAll || !strings.Contains(h.frame(), "DELETE ALL 3 SESSIONS OF ") {
			t.Fatalf("the question did not stay: ask %q", h.m.hist.ask)
		}
		h.send(key(tea.KeyEnter, 0)) // Cancel is selected
		if h.m.hist == nil || h.m.hist.ask != 0 || len(wc.spawns())+len(wx.spawns()) != 0 {
			t.Errorf("Enter on Cancel: view %v, spawns %q %q", h.m.hist != nil, wc.spawns(), wx.spawns())
		}
		if list, _ := ps.Load(); len(list[0].History) != 3 {
			t.Errorf("Cancel deleted: %+v", list[0].History)
		}
		h.send(key(tea.KeyDelete, tea.ModShift)) // counts the 2 listed now
		late := input.Entry{CLI: claude.ProviderID, ID: "late-0002", Ended: testNow}
		if _, err := ps.Update(func(l []input.Project) []input.Project {
			return ps.RecordSession(ps.ForgetSession(l, dir, es[1].CLI, es[1].ID), dir, late)
		}); err != nil { // another dashboard: one counted entry gone, one added
			t.Fatal(err)
		}
		h.send(key(tea.KeyLeft, 0))
		h.send(key(tea.KeyEnter, 0))
		list, err := ps.Load()
		// es[0] was not counted (the list had lost it): it stays, as the entry added.
		if err != nil || len(list[0].History) != 2 || list[0].History[0].ID != "late-0002" || list[0].History[1].ID != es[0].ID {
			t.Errorf("after Delete: %+v, %v", list[0].History, err)
		}
	})
}

// TestHistoryDoubleClickScrolls is criterion 75.6 in a list that scrolls: a
// click moves the list under no pointer, so a double click resumes the entry
// it landed on, and one whose presses land on two entries resumes none.
func TestHistoryDoubleClickScrolls(t *testing.T) {
	for _, c := range []struct {
		name         string
		w, h, n, sel int
		row          int
	}{{"120x40, 20 entries", 120, 40, 20, 0, 10}, {"80x24, 12 entries", 80, 24, 12, 0, 6}, {"80x24, 20 entries, selection far down", 80, 24, 20, 13, 9}} {
		t.Run(c.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				dir := t.TempDir()
				h, wc, _, _ := histRig(t, []input.Project{{Path: dir, Used: testNow.Add(-time.Hour), History: histEntries(c.n)}}, dir)
				h.send(tea.WindowSizeMsg{Width: c.w, Height: c.h})
				openHist(t, h)
				for range c.sel {
					h.send(key(tea.KeyDown, 0))
				}
				top := h.m.hist.top
				x, y := histCell(t, h, render.HistoryItemRow+c.row)
				mouseClick(t, h, x, y)
				if h.m.hist.sel != c.row || h.m.hist.top != top {
					t.Fatalf("a click: selection %d, top %d to %d", h.m.hist.sel, top, h.m.hist.top)
				}
				if x2, y2 := histCell(t, h, render.HistoryItemRow+c.row); x2 != x || y2 != y {
					t.Fatalf("the entry moved under the pointer: %d,%d to %d,%d", x, y, x2, y2)
				}
				mouseClick(t, h, x, y)
				want := fmt.Sprintf("c-%02d-%08d", c.row, 21000000+c.row)
				if sp := wc.spawns(); len(sp) != 1 || resumedArg(sp[0]) != want {
					t.Errorf("a double click spawned %q, want --resume %s", sp, want)
				}
			})
		})
	}
	synctest.Test(t, func(t *testing.T) {
		// Presses on two entries within 400 ms are no double click.
		dir := t.TempDir()
		h, wc, _, _ := histRig(t, []input.Project{{Path: dir, Used: testNow.Add(-time.Hour), History: histEntries(6)}}, dir)
		openHist(t, h)
		x, y := histCell(t, h, render.HistoryItemRow+3)
		mouseClick(t, h, x, y)
		// Another entry takes the place under the pointer (one is added at the
		// head of the list), and the second press lands on it.
		list := h.m.projects.p.RecordSession(h.m.projectList, dir, input.Entry{CLI: claude.ProviderID, ID: "new-0001", Ended: testNow})
		h.m.projectList = list
		h.m.View()
		mouseClick(t, h, x, y)
		if len(wc.spawns()) != 0 || h.m.hist == nil || h.m.hist.sel != 3 {
			t.Errorf("two entries: spawns %q", wc.spawns())
		}
	})
}

// TestHistoryOlderRefusal: a refusal of a session that was not its CLI's
// newest says "that session", not "its last session" (plan 69's text stays
// for the newest, Ctrl+R's case).
func TestHistoryOlderRefusal(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		dir := t.TempDir()
		es := []input.Entry{
			{CLI: claude.ProviderID, ID: "s-newest", Ended: testNow.Add(-time.Hour)},
			{CLI: claude.ProviderID, ID: "s-gone", Ended: testNow.Add(-2 * time.Hour)},
		}
		w := &wireClaude{noInit: true}
		h := homeRig(t, w, &wireCodex{}, storeWith(t, []input.Project{{Path: dir, Used: testNow.Add(-time.Hour), History: es}}), dir)
		openHist(t, h)
		h.send(key(tea.KeyDown, 0))
		h.send(key(tea.KeyEnter, 0))
		p := w.last()
		p.emit(`{"type":"result","subtype":"error_during_execution","is_error":true,"num_turns":0,"session_id":"s-gone","errors":["No conversation found with session ID: s-gone"]}`)
		p.exit(1, "No conversation found with session ID: s-gone\n")
		h.settle()
		h.send(key(tea.KeyEscape, 0))
		if want := "Not resumed: Claude Code no longer has that session"; h.m.ui.NotResumed != want {
			t.Errorf("the note %q, want %q", h.m.ui.NotResumed, want)
		}
		if want := "Claude Code's last session here: 1h ago · Ctrl+R continues it"; h.m.ui.LastSession != want {
			t.Errorf("the offer %q, want %q", h.m.ui.LastSession, want)
		}
	})
}

// TestHistoryNoCLIData is criterion 75.8: nothing the plan adds opens,
// writes or removes anything under ~/.claude or ~/.codex.
func TestHistoryNoCLIData(t *testing.T) {
	for _, f := range []string{"history.go", "lastsession.go", "../input/projects.go", "../render/history.go"} {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		for _, bad := range []string{`".claude`, `".codex`, "~/.claude", "~/.codex", "UserHomeDir", "os.Remove", "os.Open("} {
			// projects.go removes its own temporary file; nothing else may.
			if strings.Contains(string(b), bad) && !(bad == "os.Remove" && strings.HasSuffix(f, "projects.go")) {
				t.Errorf("%s mentions %s", f, bad)
			}
		}
	}
}
