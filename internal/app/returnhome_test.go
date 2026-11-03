package app

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/Evengelius/agent-command-center/internal/event"
	"github.com/Evengelius/agent-command-center/internal/input"
	"github.com/Evengelius/agent-command-center/internal/provider/claude"
	"github.com/Evengelius/agent-command-center/internal/render"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// Plan 75, step 2: the return to the home screen. Criteria 75.10 to 75.15 at
// the application's level, over the real Claude Code adapter and its fake
// process, the scripted sessions and the mock dashboard.

// liveDash is the Claude mock's dashboard, replayed, then named a live Claude
// Code session in the mock file's folder, so that the question can open: a
// mock run never asks it (design D9). git is the dashboard's git, nil for none.
func liveDash(t *testing.T, git input.Git) *harness {
	t.Helper()
	mockFile := filepath.Join("..", "..", "testdata", "mock", "claude-subagent-approval.jsonl")
	h := newHarness(t, Options{Config: Config{Mock: mockFile}, Git: git})
	h.settle()
	c := &h.m.s.Context
	c.Mock, c.Provider, c.ProviderName, c.Project = false, claude.ProviderID, "Claude Code", filepath.Dir(mockFile)
	h.m.s.Turn.Notice = "" // the mock's last ("turn started by the CLI")
	h.send(tea.WindowSizeMsg{Width: 120, Height: 40})
	if !h.m.s.CanReturn() || h.m.s.ReturnRefusal("") != "" {
		t.Fatalf("the dashboard cannot return: session %q, running %q", h.m.s.Session.Phase, h.m.s.Running())
	}
	return h
}

// noQuestion fails when the question to return home is open.
func noQuestion(t *testing.T, h *harness, when string) {
	t.Helper()
	if h.m.ui.Return.Open || strings.Contains(h.frame(), "RETURN TO THE HOME SCREEN?") {
		t.Errorf("%s: the question opened:\n%s", when, h.frame())
	}
}

// focusB gives zone B the focus by F6, never by Esc.
func focusB(t *testing.T, h *harness) {
	t.Helper()
	for i := 0; i < 6 && h.m.ui.Focus != "B"; i++ {
		h.send(key(tea.KeyF6, 0))
	}
	if h.m.ui.Focus != "B" {
		t.Fatalf("F6 did not reach zone B: focus %q", h.m.ui.Focus)
	}
}

// TestReturnEscLadder is criterion 75.10: every use Esc had on the dashboard
// and its windows keeps its effect, one row each, and opens no question;
// only the Esc that did nothing, the focus on zone B with nothing open or
// selected, opens it, No selected.
func TestReturnEscLadder(t *testing.T) {
	branches := func() input.Git { return &fakeBranches{fakeGit: &fakeGit{}, b: sampleBranches()} }
	graph := func() input.Git { return sampleGraph() }
	plain := func() input.Git { return &fakeGit{} }
	source := func() input.Git {
		return &fakeGit{snap: input.GitSnapshot{Top: `C:\work\demo`, Branch: input.GitBranch{Name: "main"}, Entries: []input.GitEntry{
			{X: ' ', Y: 'M', Path: "a.go", RepoPath: "a.go"}}},
			sides: map[string]input.DiffPair{"a.go false": {Old: input.DiffSide{Name: "index", ID: "1a", Data: []byte("a\n")}, New: input.DiffSide{Name: "working tree", Data: []byte("b\n")}}}}
	}
	wide := func(h *harness) { h.send(tea.WindowSizeMsg{Width: 200, Height: 60}) }
	showSrc := func(t *testing.T, h *harness) {
		wide(h)
		for _, c := range h.m.showSource() {
			h.exec(c)
		}
		h.settle()
		if !h.m.ui.Source.Open || h.m.ui.Focus != "R" {
			t.Fatalf("Source Control: shown %v, focus %q", h.m.ui.Source.Open, h.m.ui.Focus)
		}
	}
	edit := func(t *testing.T, h *harness) {
		in, _ := json.Marshal(map[string]string{"file_path": "sample/viewer.go", "old_string": "x", "new_string": "y"})
		pushEvents(h, "claude-code", evOf(event.ToolStarted, event.Tool{ToolID: "ladder-edit", Name: "Edit", Input: in}))
		h.send(key(tea.KeyUp, 0))
	}
	approval := func(t *testing.T, h *harness) {
		in, _ := json.Marshal(map[string]string{"command": "go test ./..."})
		pushEvents(h, "claude-code", evOf(event.ApprovalRequest, event.Approval{ID: "ladder-a1", Tool: "Bash", Input: in}))
		h.send(key(tea.KeyF2, 0))
		if h.m.ui.Card != "ladder-a1" {
			t.Fatalf("F2: card %q", h.m.ui.Card)
		}
	}
	type row struct {
		name  string
		rig   func(t *testing.T) *harness // nil: liveDash on git
		git   func() input.Git
		setup func(t *testing.T, h *harness)
		check func(t *testing.T, h *harness) // the old effect of one Esc
	}
	rows := []row{
		{"the + menu", nil, nil, func(t *testing.T, h *harness) {
			x, y := at(t, h, "+     ", 0)
			mouseAt(t, h, click(x+1, y))
			mouseAt(t, h, tea.MouseReleaseMsg{X: x + 1, Y: y, Button: tea.MouseLeft})
			if !h.m.ui.Plus {
				t.Fatal("the menu did not open")
			}
		}, func(t *testing.T, h *harness) {
			if h.m.ui.Plus {
				t.Error("the menu stayed open")
			}
		}},
		{"the model selector", nil, nil, func(t *testing.T, h *harness) {
			h.send(key('p', tea.ModAlt))
			if !h.m.ui.Models.Open {
				t.Fatal("Alt+P opened nothing")
			}
		}, func(t *testing.T, h *harness) {
			if h.m.ui.Models.Open {
				t.Error("the selector stayed open")
			}
		}},
		{"the modes selector", func(t *testing.T) *harness {
			h, _ := readyScript(t, "claude-code", true, event.Started{SessionID: "s-modes"})
			return h
		}, nil, func(t *testing.T, h *harness) {
			h.send(key(tea.KeyF4, 0))
			if !h.m.ui.Modes.Open {
				t.Fatal("F4 opened nothing")
			}
		}, func(t *testing.T, h *harness) {
			if h.m.ui.Modes.Open {
				t.Error("the selector stayed open")
			}
		}},
		{"the branches popup", nil, branches, func(t *testing.T, h *harness) {
			h.send(actionMsg{state.BranchRead{Branch: "main"}})
			openBranches(t, h)
		}, func(t *testing.T, h *harness) {
			if h.m.ui.Branches.Open {
				t.Error("the popup stayed open")
			}
		}},
		{"the Git Graph's scope popup", nil, graph, func(t *testing.T, h *harness) {
			wide(h)
			h.send(key('g', tea.ModAlt))
			h.send(typed('s'))
			if !h.m.ui.Scope.Open {
				t.Fatal("s opened no scope popup")
			}
		}, func(t *testing.T, h *harness) {
			if h.m.ui.Scope.Open || !h.m.ui.Graph.Open {
				t.Errorf("scope open %v, the graph shown %v", h.m.ui.Scope.Open, h.m.ui.Graph.Open)
			}
		}},
		{"the Git Graph's panel", nil, graph, func(t *testing.T, h *harness) {
			wide(h)
			h.send(key('g', tea.ModAlt))
			if !h.m.ui.Graph.Open || h.m.ui.Focus != "R" {
				t.Fatalf("Alt+G: shown %v, focus %q", h.m.ui.Graph.Open, h.m.ui.Focus)
			}
		}, func(t *testing.T, h *harness) {
			if h.m.ui.Focus != "B" || !h.m.ui.Graph.Open {
				t.Errorf("focus %q, the graph shown %v; want B, kept", h.m.ui.Focus, h.m.ui.Graph.Open)
			}
		}},
		{"the Explorer's panel", nil, plain, func(t *testing.T, h *harness) {
			wide(h)
			h.send(key(tea.KeyF3, 0))
			if !h.m.ui.Explorer.Open || h.m.ui.Focus != "R" {
				t.Fatalf("F3: shown %v, focus %q", h.m.ui.Explorer.Open, h.m.ui.Focus)
			}
		}, func(t *testing.T, h *harness) {
			if h.m.ui.Focus != "B" || !h.m.ui.Explorer.Open {
				t.Errorf("focus %q, the Explorer shown %v; want B, kept", h.m.ui.Focus, h.m.ui.Explorer.Open)
			}
		}},
		{"Source Control's panel", nil, source, showSrc, func(t *testing.T, h *harness) {
			if h.m.ui.Focus != "B" || !h.m.ui.Source.Open {
				t.Errorf("focus %q, Source Control shown %v; want B, kept", h.m.ui.Focus, h.m.ui.Source.Open)
			}
		}},
		{"the diff view", nil, source, func(t *testing.T, h *harness) {
			showSrc(t, h)
			sourceTo(t, h, "f:changes:a.go")
			h.send(key(tea.KeyEnter, 0))
			if !h.m.ui.Diff.Open || h.m.ui.Focus != "D" {
				t.Fatalf("Enter: diff %v, focus %q", h.m.ui.Diff.Open, h.m.ui.Focus)
			}
		}, func(t *testing.T, h *harness) {
			if h.m.ui.Diff.Open || h.m.ui.Focus != "R" {
				t.Errorf("diff %v, focus %q; want closed, Source Control", h.m.ui.Diff.Open, h.m.ui.Focus)
			}
		}},
		{"the commit dropdown", nil, source, func(t *testing.T, h *harness) {
			showSrc(t, h)
			for _, k := range append(append([]tea.KeyPressMsg{key(tea.KeyHome, 0), key(tea.KeyUp, 0), key(tea.KeyUp, 0)}, typedText("Message")...), key(tea.KeyDown, 0), key(tea.KeyRight, 0)) {
				h.send(k)
			}
			if !h.m.ui.Source.Menu {
				t.Fatal("the dropdown did not open")
			}
		}, func(t *testing.T, h *harness) {
			if h.m.ui.Source.Menu {
				t.Error("the dropdown stayed open")
			}
		}},
		{"the staging question, the focus on zone B", func(t *testing.T) *harness {
			h, _ := commitRigOn(t, "")
			return h
		}, nil, func(t *testing.T, h *harness) {
			stagingAsked(t, h)
			focusB(t, h)
		}, func(t *testing.T, h *harness) {
			if h.m.ui.Source.Ask == 0 || h.m.ui.Focus != "B" {
				t.Errorf("the staging question %d, focus %q; want it as it was", h.m.ui.Source.Ask, h.m.ui.Focus)
			}
		}},
		{"a pinned Git Graph card, the focus on zone B", nil, graph, func(t *testing.T, h *harness) {
			wide(h)
			h.send(key('g', tea.ModAlt))
			h.send(key(tea.KeyDown, 0))
			h.send(typed('i'))
			h.send(esc()) // the focus back to zone B, the card kept
			if !h.m.graph.pinned || h.m.ui.Graph.Card == nil || h.m.ui.Focus != "B" {
				t.Fatalf("the card: pinned %v, shown %v, focus %q", h.m.graph.pinned, h.m.ui.Graph.Card != nil, h.m.ui.Focus)
			}
		}, func(t *testing.T, h *harness) {
			if !h.m.graph.pinned || h.m.ui.Graph.Card == nil || h.m.ui.Focus != "B" {
				t.Errorf("the card: pinned %v, shown %v, focus %q; want it as it was", h.m.graph.pinned, h.m.ui.Graph.Card != nil, h.m.ui.Focus)
			}
		}},
		{"the key overlay", nil, nil, func(t *testing.T, h *harness) {
			h.send(key(tea.KeyF1, 0))
			if !h.m.ui.Keys {
				t.Fatal("F1 opened nothing")
			}
		}, func(t *testing.T, h *harness) {
			if h.m.ui.Keys {
				t.Error("the overlay stayed open")
			}
		}},
		{"the search bar", nil, nil, func(t *testing.T, h *harness) {
			h.send(key('f', tea.ModCtrl))
			if !h.m.ui.Search.Open {
				t.Fatal("Ctrl+F opened nothing")
			}
		}, func(t *testing.T, h *harness) {
			if h.m.ui.Search.Open {
				t.Error("the bar stayed open")
			}
		}},
		{"the file viewer", nil, plain, func(t *testing.T, h *harness) {
			edit(t, h)
			h.send(key(tea.KeyF3, 0))
			if !h.m.ui.File.Open {
				t.Fatal("F3 opened no file")
			}
		}, func(t *testing.T, h *harness) {
			if h.m.ui.File.Open {
				t.Error("the viewer stayed open")
			}
		}},
		{"the quick open", nil, plain, func(t *testing.T, h *harness) {
			wide(h)
			h.send(key(tea.KeyF3, 0))
			h.send(typed('/'))
			if !h.m.ui.Finder.Open {
				t.Fatal("/ opened no quick open")
			}
		}, func(t *testing.T, h *harness) {
			if h.m.ui.Finder.Open {
				t.Error("the quick open stayed open")
			}
		}},
		{"the draft window", nil, nil, func(t *testing.T, h *harness) {
			h.send(tea.PasteMsg{Content: "a draft"})
			h.send(key('g', tea.ModCtrl))
			if !h.m.ui.DraftWin.Open {
				t.Fatal("Ctrl+G opened nothing")
			}
		}, func(t *testing.T, h *harness) {
			if h.m.ui.DraftWin.Open || h.m.edit.Text() != "a draft" {
				t.Errorf("the window open %v, draft %q", h.m.ui.DraftWin.Open, h.m.edit.Text())
			}
		}},
		{"an approval card", nil, nil, approval, func(t *testing.T, h *harness) {
			if h.m.ui.Card != "" || h.m.ui.Focus != "B" || h.m.s.Approvals["ladder-a1"].Status != state.Pending {
				t.Errorf("card %q, focus %q, approval %q; want the input, still pending", h.m.ui.Card, h.m.ui.Focus, h.m.s.Approvals["ladder-a1"].Status)
			}
		}},
		{"an approval's full view", nil, nil, func(t *testing.T, h *harness) {
			approval(t, h)
			h.send(key(tea.KeyEnter, 0))
			if h.m.ui.View != "ladder-a1" {
				t.Fatalf("Enter: view %q", h.m.ui.View)
			}
		}, func(t *testing.T, h *harness) {
			if h.m.ui.View != "" {
				t.Errorf("the view %q stayed", h.m.ui.View)
			}
		}},
		{"the entry cursor", nil, nil, func(t *testing.T, h *harness) {
			h.send(key(tea.KeyUp, 0))
			if h.m.ui.Entry == 0 {
				t.Fatal("↑ put the cursor on no entry")
			}
		}, func(t *testing.T, h *harness) {
			if h.m.ui.Entry != 0 {
				t.Errorf("the cursor stayed on entry %d", h.m.ui.Entry)
			}
		}},
		{"an entry's full view", nil, nil, func(t *testing.T, h *harness) {
			h.send(key(tea.KeyUp, 0))
			h.send(key(tea.KeyEnter, 0))
			if h.m.ui.ViewEntry == 0 {
				t.Fatal("Enter opened no entry")
			}
		}, func(t *testing.T, h *harness) {
			if h.m.ui.ViewEntry != 0 {
				t.Errorf("the full view of entry %d stayed", h.m.ui.ViewEntry)
			}
		}},
		{"the summary suggestion", nil, nil, func(t *testing.T, h *harness) {
			h.m.s.Session.SummaryOffered = true
			h.frame()
			h.send(key(tea.KeyUp, 0))
			if !h.m.ui.Summary.On {
				t.Fatal("↑ gave the suggestion no focus")
			}
		}, func(t *testing.T, h *harness) {
			if h.m.ui.Summary.On || !h.m.s.Session.SummaryOffered {
				t.Errorf("focus %v, offer %v; want the input, the offer kept", h.m.ui.Summary.On, h.m.s.Session.SummaryOffered)
			}
		}},
		{"the inspection", nil, nil, func(t *testing.T, h *harness) {
			h.send(key(tea.KeyF6, 0))
			h.send(key(tea.KeyEnter, 0))
			if h.m.ui.Inspect == "" {
				t.Fatal("Enter on zone C inspected nothing")
			}
		}, func(t *testing.T, h *harness) {
			if h.m.ui.Inspect != "" {
				t.Errorf("the inspection of %q stayed", h.m.ui.Inspect)
			}
		}},
		{"the details view", nil, nil, func(t *testing.T, h *harness) { openDetailsView(t, h) }, func(t *testing.T, h *harness) {
			if h.m.ui.Details {
				t.Error("the details stayed")
			}
		}},
		{"a mouse selection", nil, nil, func(t *testing.T, h *harness) {
			x, y := at(t, h, "launched", 0)
			mouseAt(t, h, click(x, y))
			mouseAt(t, h, tea.MouseMotionMsg{X: x + 2, Y: y, Button: tea.MouseLeft})
			mouseAt(t, h, tea.MouseReleaseMsg{X: x + 2, Y: y, Button: tea.MouseLeft})
			if h.m.ui.Sel.Empty() {
				t.Fatal("the drag selected nothing")
			}
		}, func(t *testing.T, h *harness) {
			if !h.m.ui.Sel.Empty() {
				t.Error("the selection stayed")
			}
		}},
		{"the draft's selection", nil, nil, func(t *testing.T, h *harness) {
			typeText(h, "some words")
			h.send(key(tea.KeyLeft, tea.ModShift))
			h.send(key(tea.KeyLeft, tea.ModShift))
			if !h.m.edit.HasSelection() {
				t.Fatal("Shift+← selected nothing")
			}
		}, func(t *testing.T, h *harness) {
			if h.m.edit.HasSelection() || h.m.edit.Text() != "some words" {
				t.Errorf("selection %v, draft %q", h.m.edit.HasSelection(), h.m.edit.Text())
			}
		}},
		{"a zoomed zone", nil, nil, func(t *testing.T, h *harness) {
			h.send(key(tea.KeyF6, 0))
			h.send(typed('z'))
			if !h.m.ui.Zoom || h.m.ui.Focus != "C" {
				t.Fatalf("z: zoom %v, focus %q", h.m.ui.Zoom, h.m.ui.Focus)
			}
		}, func(t *testing.T, h *harness) {
			if h.m.ui.Zoom || h.m.ui.Focus != "C" {
				t.Errorf("zoom %v, focus %q; want none, C kept", h.m.ui.Zoom, h.m.ui.Focus)
			}
		}},
		{"the focus on zone C", nil, nil, func(t *testing.T, h *harness) {
			h.send(key(tea.KeyF6, 0))
			if h.m.ui.Focus != "C" {
				t.Fatalf("F6: focus %q", h.m.ui.Focus)
			}
		}, func(t *testing.T, h *harness) {
			if h.m.ui.Focus != "B" {
				t.Errorf("focus %q, want B", h.m.ui.Focus)
			}
		}},
		{"the focus on zone A", nil, nil, func(t *testing.T, h *harness) {
			h.send(key(tea.KeyF6, 0))
			h.send(key(tea.KeyF6, 0))
			if h.m.ui.Focus != "A" {
				t.Fatalf("F6 twice: focus %q", h.m.ui.Focus)
			}
		}, func(t *testing.T, h *harness) {
			if h.m.ui.Focus != "B" {
				t.Errorf("focus %q, want B", h.m.ui.Focus)
			}
		}},
		{"the quit question", nil, nil, func(t *testing.T, h *harness) {
			pushEvents(h, "claude-code", evOf(event.TurnStarted, nil))
			h.send(ctrlQ)
			if !h.m.s.QuitAsked {
				t.Fatal("Ctrl+Q during a turn asked nothing")
			}
		}, func(t *testing.T, h *harness) {
			if h.m.s.QuitAsked || h.quit || strings.HasPrefix(h.m.s.Turn.Notice, "Not now") {
				t.Errorf("asked %v, quit %v, notice %q; want the session going on", h.m.s.QuitAsked, h.quit, h.m.s.Turn.Notice)
			}
		}},
		{"the Worktrees view", nil, plain, func(t *testing.T, h *harness) {
			h.send(ctrl('w'))
			if h.m.wt == nil {
				t.Fatal("Ctrl+W opened nothing")
			}
		}, func(t *testing.T, h *harness) {
			if h.m.wt != nil {
				t.Error("the view stayed open")
			}
		}},
	}
	for _, r := range rows {
		t.Run(r.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				var git input.Git
				if r.git != nil {
					git = r.git()
				}
				var h *harness
				if r.rig != nil {
					h = r.rig(t)
				} else {
					h = liveDash(t, git)
				}
				if !h.m.s.CanReturn() {
					t.Fatalf("the session cannot return: %q", h.m.s.Session.Phase)
				}
				r.setup(t, h)
				h.send(esc())
				r.check(t, h)
				noQuestion(t, h, r.name)
			})
		})
	}

	t.Run("the failed start's back", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			w := &wireClaude{noInit: true}
			h := wireRig(t, w, Options{})
			h.send(sessionMsg{gen: h.m.sessGen, msg: actionMsg{state.HandshakeTimeout{}}})
			if s := h.m.State(); s.Startup.Phase != state.StartFailed {
				t.Fatalf("startup %q, want StartFailed", s.Startup.Phase)
			}
			h.send(esc())
			if s := h.m.State(); s.Screen != state.ScreenStartup || s.Startup.Phase != state.Asking || s.Startup.Provider != claude.ProviderID {
				t.Errorf("Esc: screen %q, phase %q, provider %q; want the home screen, the answers kept", s.Screen, s.Startup.Phase, s.Startup.Provider)
			}
			noQuestion(t, h, "the failed start's back")
		})
	})

	t.Run("the Esc that did nothing", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			h := liveDash(t, nil)
			if h.m.ui.Focus != "B" {
				t.Fatalf("focus %q", h.m.ui.Focus)
			}
			h.send(esc())
			if !h.m.ui.Return.Open || h.m.ui.Return.Yes {
				t.Fatalf("Esc: question %+v, want it open, No selected", h.m.ui.Return)
			}
			if s := h.m.State(); s.Screen != state.ScreenDashboard || s.Session.Phase != state.Ready || s.Turn.Notice != "" {
				t.Errorf("the question changed the session: screen %q, session %q, notice %q", s.Screen, s.Session.Phase, s.Turn.Notice)
			}
		})
	})
}

// returnCell is a cell of the question's button item.
func returnCell(t *testing.T, h *harness, item int) (int, int) {
	t.Helper()
	return itemCell(t, h, render.ReturnPanel, item)
}

// TestReturnQuestion is criterion 75.11: the question's text and buttons, No
// selected; ← and → move, Enter answers, Esc means No; a click on a button
// answers, one outside the card means No, one inside it on no button does
// nothing; it takes every other key, the paste and the mouse; Ctrl+Q quits.
func TestReturnQuestion(t *testing.T) {
	open := func(t *testing.T, h *harness) {
		t.Helper()
		h.send(esc())
		if !h.m.ui.Return.Open || h.m.ui.Return.Yes {
			t.Fatalf("Esc: %+v", h.m.ui.Return)
		}
	}
	synctest.Test(t, func(t *testing.T) {
		h := liveDash(t, nil)
		open(t, h)
		f := h.frame()
		joined := strings.Join(strings.Fields(f), " ")
		for _, want := range []string{"RETURN TO THE HOME SCREEN?", "[ Yes ]", "[ No ]", "←→ select · Enter answer · Esc no"} {
			if !strings.Contains(f, want) {
				t.Errorf("the question lacks %q:\n%s", want, f)
			}
		}
		// The text, wrapped in the card (its exact rows: render's TestReturnBox).
		if !strings.Contains(joined, "The session ends and Claude Code closes.") || !strings.Contains(joined, "history.") {
			t.Errorf("the text:\n%s", f)
		}
		// (The title's amber and the buttons' looks: render's TestReturnBox.)

		// ← → move; nothing else changes.
		h.send(key(tea.KeyLeft, 0))
		if !h.m.ui.Return.Yes {
			t.Error("← did not select Yes")
		}
		h.send(key(tea.KeyRight, 0))
		if h.m.ui.Return.Yes {
			t.Error("→ did not select No")
		}
		h.send(key(tea.KeyRight, 0))
		if h.m.ui.Return.Yes || !h.m.ui.Return.Open {
			t.Error("→ twice: want No, still open")
		}
		// Every other key, a paste among them, does nothing under it.
		for _, k := range []tea.KeyPressMsg{typed('a'), key(tea.KeyF1, 0), key(tea.KeyF6, 0), key(tea.KeyF3, 0), key('f', tea.ModCtrl), key(tea.KeyUp, 0), key(tea.KeyTab, 0), key('c', tea.ModCtrl)} {
			h.send(k)
		}
		h.send(tea.PasteMsg{Content: "pasted"})
		if !h.m.ui.Return.Open || h.m.ui.Keys || h.m.ui.Search.Open || h.m.ui.Focus != "B" || h.m.ui.Entry != 0 || h.m.edit.Text() != "" || h.m.ui.Explorer.Open {
			t.Errorf("a key went under the question: open %v, keys %v, search %v, focus %q, entry %d, draft %q", h.m.ui.Return.Open, h.m.ui.Keys, h.m.ui.Search.Open, h.m.ui.Focus, h.m.ui.Entry, h.m.edit.Text())
		}
		if h.m.s.Turn.Notice != "" {
			t.Errorf("Ctrl+C under the question: notice %q", h.m.s.Turn.Notice)
		}
		// Enter on No: closed, the session going on.
		h.send(key(tea.KeyEnter, 0))
		if h.m.ui.Return.Open || h.m.s.Session.Phase != state.Ready || h.m.s.Screen != state.ScreenDashboard {
			t.Errorf("Enter on No: open %v, session %q", h.m.ui.Return.Open, h.m.s.Session.Phase)
		}
		// Esc is No, even with Yes selected.
		open(t, h)
		h.send(key(tea.KeyLeft, 0))
		h.send(esc())
		if h.m.ui.Return.Open || h.m.s.Session.Phase != state.Ready {
			t.Errorf("Esc with Yes selected: open %v, session %q", h.m.ui.Return.Open, h.m.s.Session.Phase)
		}
		// Reopened, No is selected again.
		open(t, h)
		// A click on No; on the card's frame, nothing; outside, No.
		x, y := returnCell(t, h, render.ReturnNo)
		mouseClick(t, h, x, y)
		if h.m.ui.Return.Open || h.m.s.Session.Phase != state.Ready {
			t.Errorf("a click on No: open %v, session %q", h.m.ui.Return.Open, h.m.s.Session.Phase)
		}
		open(t, h)
		box := screenNow(h).Return
		mouseClick(t, h, box.X+1, box.Y+1)
		if !h.m.ui.Return.Open {
			t.Error("a click inside the card on no button closed it")
		}
		mouseAt(t, h, tea.MouseWheelMsg{X: box.X + 2, Y: box.Y + 2, Button: tea.MouseWheelUp})
		if !h.m.ui.Return.Open || h.m.ui.ConvHold {
			t.Errorf("the wheel under the question: open %v, conversation held %v", h.m.ui.Return.Open, h.m.ui.ConvHold)
		}
		mouseClick(t, h, box.X+box.W+1, box.Y)
		if h.m.ui.Return.Open || h.m.s.Session.Phase != state.Ready {
			t.Errorf("a click outside: open %v, session %q", h.m.ui.Return.Open, h.m.s.Session.Phase)
		}
		// A right click on Yes does nothing.
		open(t, h)
		x, y = returnCell(t, h, render.ReturnYes)
		mouseAt(t, h, tea.MouseClickMsg{X: x, Y: y, Button: tea.MouseRight})
		if !h.m.ui.Return.Open || h.m.s.Session.Phase != state.Ready {
			t.Errorf("a right click on Yes: open %v, session %q", h.m.ui.Return.Open, h.m.s.Session.Phase)
		}
		// Ctrl+Q quits from the question.
		h.send(ctrlQ)
		if h.m.ui.Return.Open || h.m.s.Startup.Phase != state.Exited {
			t.Errorf("Ctrl+Q: open %v, startup %q; want the quit", h.m.ui.Return.Open, h.m.s.Startup.Phase)
		}
	})
	synctest.Test(t, func(t *testing.T) {
		// A click on Yes answers Yes; Enter on Yes too (TestReturnHome).
		h := liveDash(t, nil)
		open(t, h)
		x, y := returnCell(t, h, render.ReturnYes)
		mouseClick(t, h, x, y)
		if h.m.ui.Return.Open || h.m.s.Screen != state.ScreenStartup {
			t.Errorf("a click on Yes: open %v, screen %q; want the home screen", h.m.ui.Return.Open, h.m.s.Screen)
		}
	})
	synctest.Test(t, func(t *testing.T) {
		// A terminal made too small for the dashboard closes the question, No.
		h := liveDash(t, nil)
		open(t, h)
		h.send(tea.WindowSizeMsg{Width: 70, Height: 20})
		h.frame()
		h.send(tea.WindowSizeMsg{Width: 120, Height: 40})
		if h.m.ui.Return.Open || h.m.s.Session.Phase != state.Ready {
			t.Errorf("too small: open %v, session %q", h.m.ui.Return.Open, h.m.s.Session.Phase)
		}
	})
}

// TestReturnRefused is criterion 75.12: during a turn, an agent, a waiting
// approval, a commit, a push, a staging, a branch switch and a fetch, Esc
// writes nothing to the CLI, interrupts nothing and opens no question; the
// input row says why.
func TestReturnRefused(t *testing.T) {
	const turnNote = "Not now: the turn is still running · Ctrl+C interrupts it"
	t.Run("a turn, an agent, an approval", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			recs := probe(t, "p2-claude-conversation.jsonl")
			const id = "5438db7f-8078-47e9-bfda-767bbf738c1a"
			w := &wireClaude{}
			h := wireRig(t, w, Options{})
			lines := func() int { w.mu.Lock(); defer w.mu.Unlock(); return len(w.lines) }
			refused := func(when, note string) {
				t.Helper()
				before := lines()
				phase, turn := h.m.s.Session.Phase, h.m.s.Turn.Phase
				h.send(esc())
				noQuestion(t, h, when)
				if n := lines(); n != before {
					t.Errorf("%s: %d lines written to the CLI", when, n-before)
				}
				if s := h.m.s; s.Session.Phase != phase || s.Turn.Phase != turn {
					t.Errorf("%s: session %q, turn %q; want %q, %q", when, s.Session.Phase, s.Turn.Phase, phase, turn)
				}
				if s := h.m.s; s.Turn.Notice != note || !strings.Contains(h.frame(), note) {
					t.Errorf("%s: notice %q, want %q on the input row:\n%s", when, s.Turn.Notice, note, h.frame())
				}
			}
			text, at := firstUser(t, recs)
			h.send(tea.PasteMsg{Content: text})
			h.send(key(tea.KeyEnter, 0))
			refused("a message waiting", turnNote)
			replay(t, h, w.last(), recs, at+1, 20)
			if h.m.s.Turn.Phase != state.Running {
				t.Fatalf("turn %q, want Running", h.m.s.Turn.Phase)
			}
			refused("a turn", turnNote)
			replay(t, h, w.last(), recs, 21, 36) // the Bash approval pending
			if s := h.m.s; s.Turn.Phase != state.WaitingApproval || s.Approvals[id].Status != state.Pending {
				t.Fatalf("turn %q; want a pending approval", s.Turn.Phase)
			}
			refused("a waiting approval", turnNote)
			if h.m.s.Approvals[id].Status != state.Pending {
				t.Errorf("the approval %q, want still pending", h.m.s.Approvals[id].Status)
			}
		})
		synctest.Test(t, func(t *testing.T) {
			w := &wireClaude{}
			h := wireRig(t, w, Options{})
			pushEvents(h, claude.ProviderID, evOf(event.AgentStarted, event.Agent{Key: "bg", Type: "subagent", Backgrounded: true, Description: "background agent"}))
			if h.m.s.Turn.Phase != state.Idle || h.m.s.Running() != "1 agent still running" {
				t.Fatalf("turn %q, running %q", h.m.s.Turn.Phase, h.m.s.Running())
			}
			w.mu.Lock()
			before := len(w.lines)
			w.mu.Unlock()
			h.send(esc())
			noQuestion(t, h, "an agent")
			w.mu.Lock()
			after := len(w.lines)
			w.mu.Unlock()
			if after != before || h.m.s.Agents["bg"].Status.Final() || h.m.s.Turn.Notice != turnNote || !strings.Contains(h.frame(), turnNote) {
				t.Errorf("an agent: %d lines written, agent %q, notice %q", after-before, h.m.s.Agents["bg"].Status, h.m.s.Turn.Notice)
			}
		})
	})
	// The scripted session's own record: what reached it, and its closes.
	writes := func(h *harness) string {
		ss := h.m.sess.(*scriptSession)
		sent, answers, interrupts := ss.calls()
		ss.mu.Lock()
		defer ss.mu.Unlock()
		return strings.Join(sent, ",") + "|" + strings.Join(answers, ",") + "|" + string(rune('0'+interrupts)) + "|" + string(rune('0'+ss.closes))
	}
	refusedGit := func(t *testing.T, h *harness, what string) {
		t.Helper()
		focusB(t, h)
		before := writes(h)
		h.send(esc())
		noQuestion(t, h, what)
		want := "Not now: " + what + " is still running"
		if after := writes(h); after != before {
			t.Errorf("%s: the CLI got %q, had %q", what, after, before)
		}
		if s := h.m.s; s.Turn.Notice != want || s.Session.Phase != state.Ready || !strings.Contains(h.frame(), want) {
			t.Errorf("%s: notice %q, session %q, want %q on the input row:\n%s", what, s.Turn.Notice, s.Session.Phase, want, h.frame())
		}
	}
	t.Run("a commit, a push, a staging", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			h, w := commitRig(t)
			typeMessage(t, h, "Slow")
			hold := released(t, w.holdOn())
			h.send(ctrlEnter) // the commit, held
			if h.m.op == nil {
				t.Fatal("no commit running")
			}
			refusedGit(t, h, "the commit")
			if h.m.op == nil {
				t.Fatal("the commit stopped")
			}
			w.mu.Lock()
			w.stoppable = true
			w.mu.Unlock()
			hold()
			h.settle()
			clickQueued(t, h, render.ItemCommitMain, 0, 0) // the push, running until stopped
			if h.m.op == nil {
				t.Fatal("no push running")
			}
			refusedGit(t, h, "the push")
			if h.m.op == nil {
				t.Fatal("the push stopped")
			}
			for i := 0; i < 6 && h.m.ui.Focus != "R"; i++ {
				h.send(key(tea.KeyF6, 0))
			}
			h.send(esc()) // in the panel: the push stopped, nothing left waiting
			if h.m.op != nil {
				t.Error("the push was not stopped at the end")
			}
		})
		synctest.Test(t, func(t *testing.T) {
			h, w := commitRigOn(t, "") // no upstream: after a commit the button stays on Commit
			typeMessage(t, h, "Staged only")
			h.send(ctrlEnter)
			h.settle()
			typeMessage(t, h, "All of it")
			h.send(ctrlEnter)
			if h.m.ui.Source.Ask == 0 {
				t.Fatal("no staging question")
			}
			hold := released(t, w.holdOn())
			clickQueued(t, h, render.ItemAskYes, 0, 0) // queued: the staging, held, runs off the event loop
			if h.m.op == nil || h.m.ui.GitRunning != "the staging" {
				t.Fatalf("the staging: op %v, running %q", h.m.op != nil, h.m.ui.GitRunning)
			}
			refusedGit(t, h, "the staging")
			hold()
			h.settle()
		})
	})
	t.Run("a switch, a fetch", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			r, f, _ := branchesRig(t)
			h := r.h
			f.mu.Lock()
			f.stopper = func() (bool, string) { return true, "" }
			f.mu.Unlock()
			openBranches(t, h)
			h.send(key(tea.KeyF5, 0))
			if h.m.fetch == nil {
				t.Fatal("no fetch running")
			}
			h.m.closeBranches() // the fetch outlives the popup (plan 59, design D7)
			refusedGit(t, h, "the fetch")
			if h.m.fetch == nil {
				t.Error("the fetch stopped")
			}
			h.send(altB)
			h.send(esc()) // the fetch stopped, nothing left waiting
			if h.m.fetch != nil {
				t.Error("the fetch was not stopped at the end")
			}
		})
		synctest.Test(t, func(t *testing.T) {
			r, f, _ := branchesRig(t)
			h := r.h
			hold := make(chan struct{})
			free := released(t, hold)
			f.mu.Lock()
			f.switchHold = hold
			f.mu.Unlock()
			openBranches(t, h)
			branchTo(t, h, "b:1:feat")
			h.send(key(tea.KeyEnter, 0))
			if !h.m.branchOpRunning() {
				t.Fatal("no switch running")
			}
			h.m.closeBranches()
			refusedGit(t, h, "the switch")
			free()
			h.settle()
		})
	})
}

// TestReturnHome is criteria 75.13 and 75.14 with the session idle: Yes
// closes the CLI's stdin, the close's timer asked as at quit; once the
// process has exited, the home screen, its answers kept and the session in
// the history; the state New's but for the answers, every field of the
// session a fresh model's; the draft saved and back at the next start in the
// project; and a second session in the same model drawing only its own
// entries in zones A, B and C.
func TestReturnHome(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		dir := t.TempDir()
		ps := &input.Projects{Dir: t.TempDir()}
		d := input.Drafts{Dir: t.TempDir()}
		reg := &input.Sessions{Dir: t.TempDir()}
		w := &wireClaude{}
		eof := -1
		w.onEOF = func() { w.mu.Lock(); eof = len(w.lines); w.mu.Unlock() }
		h := homeRig(t, w, &wireCodex{}, ps, dir)
		h.m.opt.Drafts, h.m.opt.Sessions = &d, reg
		turns := &fakeTurns{trees: []string{"t1", "t2", "t3", "t4"}}
		h.m.opt.TurnStore = func(string) (TurnStore, error) { return turns, nil }
		notified := 0
		var mu sync.Mutex
		h.m.opt.Notify = func(string, string) error { return nil }
		h.m.opt.Unnotify = func() error { mu.Lock(); notified++; mu.Unlock(); return nil }
		h.m.opt.Location = time.UTC

		h.send(key(tea.KeyEnter, 0))
		w.last().emit(claudeInit(dir, "s-first"))
		h.settle()
		h.send(tea.PasteMsg{Content: "FIRST-PROMPT"})
		h.send(key(tea.KeyEnter, 0))
		h.send(tea.BlurMsg{}) // the turn's end rings: the ping's notification goes up (plan 64)
		w.last().emit(claudeTurn("s-first", "FIRST-ANSWER")...)
		h.settle()
		pushEvents(h, claude.ProviderID, evOf(event.AgentStarted, event.Agent{Key: "fa", Type: "subagent", Label: "FIRST-AGENT", Description: "FIRST-AGENT"}),
			evOf(event.AgentCompleted, event.Agent{Key: "fa", Status: "completed"}))
		if f := h.frame(); !strings.Contains(f, "FIRST-ANSWER") || !strings.Contains(f, "FIRST-AGENT") {
			t.Fatalf("the first session's frame:\n%s", f)
		}
		if h.m.lock == nil || h.m.turns == nil {
			t.Fatalf("the session's lock %v, store %v", h.m.lock, h.m.turns)
		}
		if !h.m.ping.notified {
			t.Fatal("no notification up after the turn")
		}
		// The Git Graph's read under way.
		reader := &fakeReader{}
		h.m.graph.reader = reader
		h.send(tea.PasteMsg{Content: "unsent words"})
		before := gensOf(h.m)
		memo, wrap, colors := h.m.ui.Memo, h.m.ui.Wrap, h.m.ui.Colors

		h.send(esc())
		if !h.m.ui.Return.Open || h.m.ui.Return.Yes {
			t.Fatalf("Esc: %+v", h.m.ui.Return)
		}
		var timers []string
		h.m.opt.After = func(d time.Duration, msg tea.Msg) tea.Cmd {
			if sm, ok := msg.(sessionMsg); ok {
				mu.Lock()
				timers = append(timers, d.String()+" "+reflect.TypeOf(sm.msg.(actionMsg).a).Name())
				mu.Unlock()
			}
			return nil
		}
		w.mu.Lock()
		from := len(w.lines)
		w.mu.Unlock()
		h.send(key(tea.KeyLeft, 0))
		h.send(key(tea.KeyEnter, 0))
		h.settle()
		if o := stdinOrder(w, &eof, from, ""); !slices.Equal(o, []string{"EOF"}) {
			t.Errorf("stdin after Yes %q; want its end alone", o)
		}
		mu.Lock()
		if !slices.Contains(timers, "5s CloseTimeout") {
			t.Errorf("the timers asked %q; want the close's 5 s, as at quit", timers)
		}
		mu.Unlock()

		// The home screen, the answers kept, the session in the history.
		s := h.m.State()
		if s.Screen != state.ScreenStartup || s.Startup.Phase != state.Asking || s.Startup.Provider != claude.ProviderID || !state.SamePath(s.Startup.Project, dir) {
			t.Fatalf("after the exit: screen %q, phase %q, provider %q, project %q", s.Screen, s.Startup.Phase, s.Startup.Provider, s.Startup.Project)
		}
		if h.m.path.Text() != dir || h.m.ui.Choice != claude.ProviderID {
			t.Errorf("the form: field %q, choice %q", h.m.path.Text(), h.m.ui.Choice)
		}
		if f := h.frame(); !strings.Contains(f, "NEW SESSION") || strings.Contains(f, "FIRST-ANSWER") {
			t.Errorf("the home screen:\n%s", f)
		}
		if list, err := ps.Load(); err != nil || len(list) != 1 || len(list[0].History) != 1 || list[0].History[0].ID != "s-first" || list[0].History[0].Title != "FIRST-PROMPT" {
			t.Errorf("the history: %+v, %v", list, err)
		}
		openHist(t, h)
		if f := h.frame(); !strings.Contains(f, "FIRST-PROMPT") {
			t.Errorf("Alt+H: the session missing:\n%s", f)
		}
		h.send(esc())

		// Criterion 75.14: the state New's but for the answers.
		want := *state.New()
		want.Startup = s.Startup
		if !reflect.DeepEqual(*s, want) {
			t.Errorf("the state after the return:\n%+v\nwant\n%+v", *s, want)
		}
		// The session's part of the application ended.
		mu.Lock()
		if notified != 1 {
			t.Errorf("the notification taken out %d times, want once", notified)
		}
		mu.Unlock()
		if !reader.isClosed() || !turns.closed {
			t.Errorf("the Git Graph's read closed %v, the turn review's store removed %v", reader.closed, turns.closed)
		}
		if live, err := reg.Live(); err != nil || len(live) != 0 {
			t.Errorf("registrations after the return: %+v, %v", live, err)
		}
		if got := draftFile(t, d, dir); !bytes.Equal(got, []byte("unsent words")) {
			t.Errorf("the stored draft %q", got)
		}
		fresh := homeRig(t, &wireClaude{}, &wireCodex{}, ps, dir)
		sameAsFresh(t, h.m, fresh.m, before)
		if h.m.ui.Memo == memo || h.m.ui.Wrap == wrap || h.m.ui.Colors == colors {
			t.Error("the interface's caches outlived the session")
		}

		// The next start in the project: its draft back, a second session that
		// draws only its own entries.
		h.send(key(tea.KeyEnter, 0))
		if sp := w.spawns(); len(sp) != 2 || resumedArg(sp[1]) != "" {
			t.Fatalf("spawns %q; want a second, new session", sp)
		}
		w.last().emit(claudeInit(dir, "s-second"))
		h.settle()
		if h.m.edit.Text() != "unsent words" {
			t.Errorf("the draft at the next start: %q", h.m.edit.Text())
		}
		h.m.edit.Clear()
		h.send(tea.PasteMsg{Content: "SECOND-PROMPT"})
		h.send(key(tea.KeyEnter, 0))
		w.last().emit(claudeTurn("s-second", "SECOND-ANSWER")...)
		h.settle()
		for _, size := range [][2]int{{120, 40}, {200, 60}} {
			h.send(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
			f := h.frame()
			if !strings.Contains(f, "SECOND-ANSWER") || !strings.Contains(f, "SECOND-PROMPT") {
				t.Errorf("%v: the second session's entries missing:\n%s", size, f)
			}
			for _, old := range []string{"FIRST-ANSWER", "FIRST-PROMPT", "FIRST-AGENT", "TURN 2"} {
				if strings.Contains(f, old) {
					t.Errorf("%v: the second session draws %q:\n%s", size, old, f)
				}
			}
		}
		if s := h.m.State(); len(s.Stages) != 1 || len(s.Order) != 0 || s.Session.ID != "s-second" {
			t.Errorf("the second session: %d stages, agents %q, ID %q", len(s.Stages), s.Order, s.Session.ID)
		}
	})
}

// gensOf are the model's generations by name.
func gensOf(m *Model) map[string]int {
	return map[string]int{"wtGen": m.wtGen, "histGen": m.histGen, "fileGen": m.fileGen, "searchGen": m.searchGen, "finderGen": m.finderGen,
		"explorerGen": m.explorerGen, "blameSeq": m.blameSeq, "statusGen": m.statusGen, "pollGen": m.pollGen, "diffGen": m.diffGen,
		"folderSeq": m.folderSeq, "opGen": m.opGen, "opSeq": m.opSeq, "branchGen": m.branchGen, "headGen": m.headGen,
		"graph.gen": m.graph.gen, "graph.refsGen": m.graph.refsGen, "graph.watchGen": m.graph.watchGen, "draftGen": m.draftGen,
		"thumbGen": m.thumbGen, "turnsGen": m.turnsGen, "sessGen": m.sessGen}
}

// appFields are the model's fields that belong to the application, not to a
// session, which a return keeps (resetSession).
var appFields = map[string]bool{
	"opt": true, "s": true, "ui": true, "path": true, "env": true, "detected": true, "check": true,
	"projects": true, "projectList": true, "projectsGen": true, "projectsSeen": true,
	"folderOpen": true, "histRefused": true, "histRefusedFor": true, "lock": true, "locking": true,
	"spinning": true, "clocking": true, "framing": true, "bannering": true, "blurred": true,
	"pasting": true, "pasteN": true, "ctrlVDown": true, "pasteCount": true, "lastPaste": true, "pasteWait": true, "waitPastes": true,
	"drawn": true, "send": true, "kicks": true, "pillLabels": true, "liveLabel": true, "toastGen": true,
	"home": true, "glyphs": true, "homeSecond": true, "resized": true, "screen": true, "repaint": true, "motionDropped": true,
}

// appUI are the interface's fields that belong to the application, or that
// syncUI makes from the startup screen's own: a return keeps them.
var appUI = map[string]bool{
	"Width": true, "Height": true, "Now": true, "Spinner": true, "Field": true, "Choice": true, "ProjectSel": true,
	"ProjectsProblem": true, "FolderProblem": true, "Sixel": true, "PillNoFont": true, "NotifyProblem": true, "MouseOff": true, "Toast": true,
	"Wrap": true, "Memo": true, "Colors": true,
}

// sameAsFresh is criterion 75.14's model: every field of m tied to a session
// has the value it has in fresh, a model just started on the same home
// screen; the generations are past the session's (before), so that its reads'
// late answers are dropped.
func sameAsFresh(t *testing.T, m, fresh *Model, before map[string]int) {
	t.Helper()
	gens := gensOf(m)
	mv, fv := reflect.ValueOf(m).Elem(), reflect.ValueOf(fresh).Elem()
	for i := range mv.NumField() {
		name := mv.Type().Field(i).Name
		_, gen := gens[name]
		switch {
		case appFields[name], gen, name == "graph":
		case name == "ping":
			// The bell's limit is the terminal's: its last ring and its timer
			// asked for stay; what the session needed the owner for goes.
			p := m.ping
			p.last, p.due = time.Time{}, false
			if !reflect.ValueOf(p).IsZero() {
				t.Errorf("the ping's needs outlived the session: %+v", m.ping)
			}
		case name == "colours":
			// Taken anew at each update, for the state now: the commands'
			// colours asked from the first again.
			if m.colours.s != m.s || m.colours.tools != fresh.colours.tools || m.colours.asks != fresh.colours.asks {
				t.Errorf("the commands' colours: %+v, a fresh model's %+v", m.colours, fresh.colours)
			}
		case !mv.Field(i).IsZero() || !fv.Field(i).IsZero():
			t.Errorf("model field %s: after the return zero %v, in a fresh model zero %v", name, mv.Field(i).IsZero(), fv.Field(i).IsZero())
		}
	}
	g := reflect.ValueOf(m.graph)
	for i := range g.NumField() {
		if name := g.Type().Field(i).Name; name != "gen" && name != "refsGen" && name != "watchGen" && !g.Field(i).IsZero() {
			t.Errorf("the Git Graph's %s outlived the session", name)
		}
	}
	for name, n := range gens {
		if n <= before[name] {
			t.Errorf("generation %s %d, %d before the return: an answer of the session's would be taken", name, n, before[name])
		}
	}
	uv, fu := reflect.ValueOf(m.ui), reflect.ValueOf(fresh.ui)
	for i := range uv.NumField() {
		name := uv.Type().Field(i).Name
		if appUI[name] {
			continue
		}
		if !reflect.DeepEqual(uv.Field(i).Interface(), fu.Field(i).Interface()) {
			t.Errorf("the interface's %s: %+v after the return, %+v in a fresh model", name, uv.Field(i).Interface(), fu.Field(i).Interface())
		}
	}
}
