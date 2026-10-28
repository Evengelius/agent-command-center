package app

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"testing"
	"testing/synctest"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/Evengelius/agent-command-center/internal/event"
	"github.com/Evengelius/agent-command-center/internal/input"
	"github.com/Evengelius/agent-command-center/internal/input/imagetest"
	"github.com/Evengelius/agent-command-center/internal/provider"
	"github.com/Evengelius/agent-command-center/internal/render"
	"github.com/Evengelius/agent-command-center/internal/state"
)

var update = flag.Bool("update", false, "rewrite the golden frames in testdata/frames")

// fakeCLI is a detected agent CLI that is never started.
type fakeCLI struct{ d provider.Detection }

func (p fakeCLI) ID() string                                { return p.d.ID }
func (p fakeCLI) DisplayName() string                       { return p.d.Name }
func (p fakeCLI) Detect(context.Context) provider.Detection { return p.d }
func (p fakeCLI) Start(context.Context, provider.SessionContext) (provider.Session, error) {
	return nil, fmt.Errorf("fake CLI")
}

var sizes = [][2]int{{80, 24}, {120, 40}, {200, 60}, {79, 24}}

// TestGoldenFrames is test 9 (proposal step 0, item 9): both screens at the
// three reference sizes and too small at 79x24, each frame exactly the
// terminal's height and no line wider than it.
func TestGoldenFrames(t *testing.T) {
	startup := func(t *testing.T) *harness {
		h := newHarness(t, Options{
			Providers: []provider.Provider{
				fakeCLI{provider.Detection{ID: "claude-code", Name: "Claude Code", Installed: true, Version: "2.1.300", Path: `C:\Tools\claude.exe`}},
				fakeCLI{provider.Detection{ID: "codex", Name: "OpenAI Codex", Problem: "not found on the PATH"}},
			},
			Cwd: `C:\work\demo-project`,
		})
		h.settle()
		if s := h.m.State(); s.Screen != state.ScreenStartup || s.Startup.Phase != state.Asking {
			t.Fatalf("startup: screen %q, phase %q", s.Screen, s.Startup.Phase)
		}
		return h
	}
	dashGit := func(t *testing.T, git input.Git) *harness {
		h := newHarness(t, Options{Config: Config{Mock: filepath.Join("..", "..", "testdata", "mock", "claude-subagent-approval.jsonl")}, Git: git})
		h.settle()
		// 154, the subagent's model (plan 5, item 7) and the session's models
		// (plan 34).
		if s := h.m.State(); s.Screen != state.ScreenDashboard || len(s.Log) != 156 {
			t.Fatalf("dashboard: screen %q, %d events, want 156", s.Screen, len(s.Log))
		}
		return h
	}
	dash := func(t *testing.T) *harness { return dashGit(t, nil) }
	codexDash := func(t *testing.T) *harness {
		h := newHarness(t, Options{Config: Config{Mock: filepath.Join("..", "..", "testdata", "mock", "codex-subagents.jsonl")}})
		h.settle()
		// 177, the 5 steps Codex's adapter infers (plan 22) and the thread's
		// model settings (plan 34).
		if s := h.m.State(); s.Screen != state.ScreenDashboard || len(s.Log) != 183 {
			t.Fatalf("codex dashboard: screen %q, %d events, want 183", s.Screen, len(s.Log))
		}
		return h
	}
	// The inspection: the Claude mock's subagent, on its Tools tab.
	inspected := func(t *testing.T) *harness {
		h := dash(t)
		h.send(key(tea.KeyF6, 0)) // zone C
		h.send(key(tea.KeyEnter, 0))
		h.send(typed('2'))
		if h.m.ui.Inspect == "" || h.m.ui.Tab != 1 {
			t.Fatalf("inspect %q, tab %d", h.m.ui.Inspect, h.m.ui.Tab)
		}
		return h
	}

	// Zone A focused, its closed stage unfolded with →, its cursor on the
	// first child: at 80 columns the tree takes the place of zones B and C.
	tree := func(t *testing.T) *harness {
		h := codexDash(t)
		for _, k := range []tea.KeyPressMsg{key(tea.KeyF6, 0), key(tea.KeyF6, 0), key(tea.KeyRight, 0), key(tea.KeyDown, 0)} {
			h.send(k)
		}
		if h.m.ui.Focus != "A" || !strings.HasPrefix(h.m.ui.ACursor, "a:") {
			t.Fatalf("focus %q cursor %q", h.m.ui.Focus, h.m.ui.ACursor)
		}
		return h
	}

	// The telemetry details, from zone A since plan 24, with errors of the
	// session, of an agent and a retry, one long and on two lines (plan 4,
	// step 7).
	details := func(t *testing.T) *harness {
		h := dash(t)
		addErrors(h, 2)
		openDetailsView(t, h)
		return h
	}

	// Plan 24, criterion 24.17: the three quota windows of the synthetic
	// mock, the image's values, with zone A focused and its cursor on QUOTA
	// WINDOWS.
	quota := func(t *testing.T) *harness {
		h := newHarness(t, Options{Config: Config{Mock: filepath.Join("..", "..", "testdata", "mock", "claude-quota-windows.jsonl")}})
		h.settle()
		if s := h.m.State(); s.Screen != state.ScreenDashboard || !s.Metrics["quota.7d.fable"].Present {
			t.Fatalf("quota dashboard: screen %q, fable window %+v", s.Screen, s.Metrics["quota.7d.fable"])
		}
		h.send(key(tea.KeyF6, 0))
		h.send(key(tea.KeyF6, 0))
		for i := 0; i < 40 && h.m.ui.ACursor != render.NodeQuota; i++ {
			h.send(key(tea.KeyDown, 0))
		}
		if h.m.ui.Focus != "A" || h.m.ui.ACursor != render.NodeQuota {
			t.Fatalf("focus %q cursor %q", h.m.ui.Focus, h.m.ui.ACursor)
		}
		return h
	}

	// Plan 12, item 2: Claude Code in Auto, its reviewer's mark on a tool;
	// Codex with a change requested for its next turn; Codex in Auto with an
	// auto-review entry in zone B. Manual is the bar of every other frame.
	claudeAuto := func(t *testing.T) *harness {
		h := dash(t)
		var ids []string
		for id, tc := range h.m.State().Tools {
			if tc.Agent == "" {
				ids = append(ids, id)
			}
		}
		sort.Strings(ids)
		if len(ids) == 0 {
			t.Fatal("no orchestrator tool in the mock")
		}
		pushEvents(h, "claude-code", evOf(event.ToolDecision, event.Verdict{ToolID: ids[0], Decision: event.Allow, ReasonType: "classifier"}))
		pushEvents(h, "claude-code", evOf(event.ApprovalMode, event.Mode{Mode: event.ModeAuto, Answers: true}))
		if h.m.State().Mode() != state.ModeAuto {
			t.Fatalf("mode %q", h.m.State().Mode())
		}
		return h
	}
	codexRequested := func(t *testing.T) *harness {
		h := codexDash(t)
		h.m.State().Do(state.ChooseMode{Mode: state.ModeAuto, Capable: true})
		if h.m.State().Session.ApprovalRequested != state.ModeAuto {
			t.Fatalf("requested %q", h.m.State().Session.ApprovalRequested)
		}
		return h
	}
	codexReview := func(t *testing.T) *harness {
		h := codexDash(t)
		pushEvents(h, "codex",
			evOf(event.ApprovalMode, event.Mode{Mode: event.ModeAuto}),
			evOf(event.ReviewUpdated, event.Review{ReviewID: "rv1", Command: "go vet ./...", Status: "approved", Risk: "low", Authorization: "high", Rationale: "a read-only check of the project"}))
		return h
	}
	// Plan 17: the file an Edit changed, from its entry in zone B, the lines
	// it wrote marked; then full screen, a search; the Explorer (amendment
	// 12), git's markers from a test double, and the quick open from it.
	file := func(keys ...tea.KeyPressMsg) func(*testing.T) *harness {
		return func(t *testing.T) *harness {
			h := dash(t)
			in, _ := json.Marshal(map[string]string{"file_path": "sample/viewer.go", "old_string": "x",
				"new_string": "func (c Colour) Hex() string {\n\treturn fmt.Sprintf(\"#%06x\", c.RGB)\n}"})
			pushEvents(h, "claude-code", evOf(event.ToolStarted, event.Tool{ToolID: "golden-edit", Name: "Edit", Input: in}))
			h.send(key(tea.KeyUp, 0))
			h.send(key(tea.KeyF3, 0))
			if f := h.m.ui.File; !f.Open || f.Rel != "sample/viewer.go" || f.Mark != [2]int{34, 37} || f.Hl.Lang != "Go" {
				t.Fatalf("the viewer: open %v, %q, mark %v, %q, state %q", f.Open, f.Rel, f.Mark, f.Hl.Lang, f.State)
			}
			for _, k := range keys {
				h.send(k)
			}
			return h
		}
	}
	explorerGit := &fakeGit{marks: input.GitMarks{
		Files: map[string]byte{"sample/viewer.go": 'M', "codex-api-key.jsonl": 'U', "codex-spawn-request.jsonl": 'A', "codex-plan6-limits.jsonl": '!',
			"sample/viewer_test.go": 'D', "tools/old.jsonl": 'D'},
		Ignored: map[string]bool{"codex-no-5h-window.jsonl": true},
	}}
	// The Explorer with its own glyphs, and with Cascadia Mono NF's icons
	// (plan 55, design D4), the flag set before the panel is first drawn.
	explorerRig := func(nerd bool) func(*testing.T) *harness {
		return func(t *testing.T) *harness {
			h := dashGit(t, explorerGit)
			h.m.ui.NerdIcons = nerd
			h.send(key(tea.KeyF3, 0))
			h.send(key(tea.KeyRight, 0))
			h.send(key(tea.KeyDown, 0))
			if v := h.m.ui.Explorer; !v.Open || v.Cursor != "sample/viewer.go" || h.m.ui.File.Open || v.Git.Note != "" {
				t.Fatalf("the Explorer: open %v, cursor %q, note %q", v.Open, v.Cursor, v.Git.Note)
			}
			return h
		}
	}
	explorer := explorerRig(false)
	finder := func(t *testing.T) *harness {
		h := dashGit(t, explorerGit)
		h.send(key(tea.KeyF3, 0))
		h.send(typed('/'))
		for _, k := range typedText("vi") {
			h.send(k)
		}
		if v := h.m.ui.Finder; !v.Open || len(v.Files) == 0 || v.Files[0] != "sample/viewer.go" {
			t.Fatalf("the quick open: %+v", v)
		}
		return h
	}
	// Plan 44 (amendment 45): Source Control shown, a change of each kind,
	// its cursor on a file; then that file's diff in zone B's place, unified,
	// and widened over zone A, side by side from 150 columns; not a
	// repository.
	sourceGit := func() *fakeGit {
		old := "package sample\n\nimport \"fmt\"\n\n// Colour is an RGB value.\ntype Colour struct{ RGB int }\n\n" +
			"// Hex is the colour as #rrggbb.\nfunc (c Colour) Hex() string {\n\treturn fmt.Sprintf(\"#%06X\", c.RGB)\n}\n\n" +
			"// Dark says the colour is dark.\nfunc (c Colour) Dark() bool { return c.RGB < 0x808080 }\n"
		cur := strings.Replace(strings.Replace(old, "#%06X", "#%06x", 1), "func (c Colour) Dark() bool { return c.RGB < 0x808080 }\n",
			"func (c Colour) Dark() bool {\n\tr, g, b := c.RGB>>16, c.RGB>>8&0xFF, c.RGB&0xFF\n\treturn r*299+g*587+b*114 < 128000\n}\n", 1)
		return &fakeGit{snap: input.GitSnapshot{Top: `C:\work\demo-project`, Branch: input.GitBranch{Name: "main"}, Entries: []input.GitEntry{
			{X: 'U', Y: 'U', Path: "sample/merge.go", RepoPath: "sample/merge.go"},
			{X: ' ', Y: 'M', Path: "sample/viewer.go", RepoPath: "sample/viewer.go"},
			{X: 'M', Y: 'M', Path: "README.md", RepoPath: "README.md"},
			{X: 'R', Y: ' ', Path: "tools/replay.jsonl", Orig: "tools/old.jsonl", RepoPath: "tools/replay.jsonl", RepoOrig: "tools/old.jsonl"},
			{X: 'A', Y: ' ', Path: "codex-spawn-request.jsonl", RepoPath: "codex-spawn-request.jsonl"},
			{X: ' ', Y: 'D', Path: "sample/viewer_test.go", RepoPath: "sample/viewer_test.go"},
			{X: '?', Y: '?', Path: "codex-api-key.jsonl", RepoPath: "codex-api-key.jsonl"},
			{X: '?', Y: '?', Path: "notes/", RepoPath: "notes/"},
		}}, sides: map[string]input.DiffPair{
			"sample/viewer.go false": {Old: input.DiffSide{Name: "index", ID: "1a", Data: []byte(old)}, New: input.DiffSide{Name: "working tree", Data: []byte(cur)}},
		}}
	}
	source := func(keys ...tea.KeyPressMsg) func(*testing.T) *harness {
		return func(t *testing.T) *harness {
			h := dashGit(t, sourceGit())
			h.send(tea.WindowSizeMsg{Width: 200, Height: 60})
			for _, c := range h.m.showSource() {
				h.exec(c)
			}
			h.settle()
			for _, k := range append([]tea.KeyPressMsg{key(tea.KeyDown, 0), key(tea.KeyDown, 0), key(tea.KeyDown, 0)}, keys...) {
				h.send(k)
			}
			if v := h.m.ui.Source; !v.Open || v.Cursor != "f:changes:sample/viewer.go" || v.Note != "" {
				t.Fatalf("Source Control: shown %v, cursor %q, note %q", v.Open, v.Cursor, v.Note)
			}
			return h
		}
	}
	// Plan 50 (amendment 51): three files selected by Ctrl+click, the first
	// one the cursor's, its diff open, the third hovered; the button
	// "Commit 3 files". Also drawn at 208x59, the acc tab maximised.
	sourceSelection := func(t *testing.T) *harness {
		h := source()(t)
		for _, k := range []string{"f:changes:sample/viewer.go", "f:staged:codex-spawn-request.jsonl", "f:changes:codex-api-key.jsonl"} {
			ctrlClick(t, h, rowItem(t, h, k))
		}
		hover := rowItem(t, h, "f:changes:codex-api-key.jsonl")
		x, y := itemCell(t, h, render.SourcePanel, hover)
		mouseAt(t, h, tea.MouseMotionMsg{X: x, Y: y})
		if v := h.m.ui.Source; len(v.Selected) != 3 || v.Hover != hover || v.Cursor != "f:changes:sample/viewer.go" || !h.m.ui.Diff.Open {
			t.Fatalf("the selection %q, hover %d, cursor %q, diff open %v", selectedPaths(h), v.Hover, v.Cursor, h.m.ui.Diff.Open)
		}
		return h
	}
	// The commit controls (plan 45, brief amendment 46): a message typed in
	// the field, reached by ↑ from the list's first row and ↑ again, then
	// the button given the keys by ↓ and its dropdown opened by →, its item
	// on the band; with Sixel, the controls' cells blank for their images;
	// the button on Push after a commit (an upstream, no conflict), its note
	// under it.
	commitRig := func(t *testing.T, git input.Git, keys ...tea.KeyPressMsg) *harness {
		h := dashGit(t, git)
		h.send(tea.WindowSizeMsg{Width: 200, Height: 60})
		for _, c := range h.m.showSource() {
			h.exec(c)
		}
		h.settle()
		for _, k := range append(append([]tea.KeyPressMsg{key(tea.KeyHome, 0), key(tea.KeyUp, 0), key(tea.KeyUp, 0)},
			typedText("Add Source Control diff viewer")...), keys...) {
			h.send(k)
		}
		return h
	}
	commitMenu := func(t *testing.T) *harness {
		h := commitRig(t, sourceGit(), key(tea.KeyDown, 0), key(tea.KeyRight, 0))
		if v := h.m.ui.Source; !v.Menu || !v.MenuHot || h.m.msg.Text() != "Add Source Control diff viewer" {
			t.Fatalf("the dropdown: open %v, hot %v, message %q", v.Menu, v.MenuHot, h.m.msg.Text())
		}
		return h
	}
	commitSixel := func(t *testing.T) *harness {
		h := commitMenu(t)
		h.send(PrimaryDeviceAttributesEvent{61, 4, 6, 7, 14})
		return h
	}
	commitPush := func(t *testing.T) *harness {
		g := sourceGit()
		g.snap.Entries, g.snap.Branch.Upstream = g.snap.Entries[1:], "origin/main"
		h := commitRig(t, &fakeWrite{fakeGit: g}, key(tea.KeyEnter, tea.ModCtrl))
		if v := h.m.ui.Source; v.Action != render.ActionPush || v.Notice != "Committed abc0001." {
			t.Fatalf("after the commit: action %q, note %q", v.Action, v.Notice)
		}
		return h
	}
	sourceNone := func(t *testing.T) *harness {
		h := dashGit(t, &fakeGit{statusErr: input.NotRepoError{Dir: `C:\work\demo-project`}})
		h.send(tea.WindowSizeMsg{Width: 200, Height: 60})
		for _, c := range h.m.showSource() {
			h.exec(c)
		}
		h.settle()
		if v := h.m.ui.Source; !v.Open || v.Note != "No Git repository found for this project." {
			t.Fatalf("Source Control: shown %v, note %q", v.Open, v.Note)
		}
		return h
	}
	// draft is the draft window, Ctrl+G, over the dashboard (amendment 13).
	draft := func(keys ...tea.KeyPressMsg) func(*testing.T) *harness {
		return func(t *testing.T) *harness {
			h := dash(t)
			h.send(tea.PasteMsg{Content: "Review the colour palette in sample/viewer.go:\n\n" +
				"1. keep the five accents of the reference screen;\n" +
				"2. check each one against the dark background;\n" +
				"3. report any pair under 4.5:1, with its hex values.\n\n" +
				"Do not change the file yet."})
			h.send(key('g', tea.ModCtrl))
			if !h.m.ui.DraftWin.Open {
				t.Fatal("Ctrl+G did not open the draft window")
			}
			for _, k := range keys {
				h.send(k)
			}
			return h
		}
	}
	// attached is the dashboard with two of the generated test images
	// attached to a draft through the "+" menu and a fake Open dialog (plan
	// 31); sent is the same message sent, the mock session recording it.
	attached := func(t *testing.T) *harness {
		files, err := imagetest.WriteAll(t.TempDir())
		if err != nil {
			t.Fatal(err)
		}
		h := dash(t)
		p := &picker{answers: [][]string{{files["png"], files["jpg"]}}}
		h.m.opt.PickImages = p.pick
		h.send(tea.PasteMsg{Content: "What colours and what text are in these images?"})
		upload(t, h)
		if len(h.m.attach) != 2 {
			t.Fatalf("%d attachments", len(h.m.attach))
		}
		return h
	}
	sent := func(t *testing.T) *harness {
		h := attached(t)
		h.send(key(tea.KeyEnter, 0))
		if len(h.m.attach) != 0 {
			t.Fatalf("%d attachments after the send", len(h.m.attach))
		}
		return h
	}
	// preview is the attachments once the terminal says it draws Sixel
	// images: their thumbnails' strip above the chips, its cells blank in
	// the frame (plan 32, design D4); none under 30 rows.
	preview := func(t *testing.T) *harness {
		h := attached(t)
		h.send(PrimaryDeviceAttributesEvent{61, 4, 6, 7, 14})
		if !h.m.ui.Sixel {
			t.Fatal("the answer did not turn the preview on")
		}
		return h
	}
	// plusMenu is the "+" menu open over the dashboard, its item under the
	// pointer (plan 30, amendment 30).
	plusMenu := func(t *testing.T) *harness {
		h := dash(t)
		x, y := at(t, h, "+     ", 0)
		mouseAt(t, h, click(x+1, y))
		mouseAt(t, h, tea.MouseReleaseMsg{X: x + 1, Y: y, Button: tea.MouseLeft})
		ix, iy := at(t, h, "Upload from computer", 0)
		mouseAt(t, h, tea.MouseMotionMsg{X: ix + 3, Y: iy})
		if !h.m.ui.Plus || !h.m.ui.PlusHover {
			t.Fatalf("the menu: open %v, hover %v", h.m.ui.Plus, h.m.ui.PlusHover)
		}
		return h
	}
	// The model selector (plan 34): P34c's Claude Code session opened by a
	// click on its pill, P34x's Codex session by Alt+P; then keys move the
	// cursor to More models and Enter expands it, or to Effort and ← asks for
	// the level below, which the mock applies (plan 37).
	selector := func(mock string, keys ...tea.KeyPressMsg) func(*testing.T) *harness {
		return func(t *testing.T) *harness {
			h := newHarness(t, Options{Config: Config{Mock: filepath.Join("..", "..", "testdata", "mock", mock)}})
			h.settle()
			if mock == "claude-models.jsonl" {
				x, y := at(t, h, "Fable 5.1 Max", 0)
				mouseAt(t, h, click(x, y))
				mouseAt(t, h, tea.MouseReleaseMsg{X: x, Y: y, Button: tea.MouseLeft})
			} else {
				h.send(key('p', tea.ModAlt))
			}
			if !h.m.ui.Models.Open {
				t.Fatalf("%s: the selector did not open", mock)
			}
			for _, k := range keys {
				h.send(k)
			}
			return h
		}
	}
	// The branches popup (plan 59, amendment 61): opened by a click on the
	// project bar's branch name, the lists read, Recent and Local open,
	// Remote and Tags folded, the cursor on the current branch; a search
	// typed; the confirmation of a switch with uncommitted changes.
	branches := func(keys ...tea.KeyPressMsg) func(*testing.T) *harness {
		return func(t *testing.T) *harness {
			h := dashGit(t, &fakeBranches{fakeGit: &fakeGit{}, b: sampleBranches(), changes: 3})
			h.send(actionMsg{state.BranchRead{Branch: "main"}})
			x, y := at(t, h, "branch main", 0)
			mouseClick(t, h, x+len("branch "), y)
			for _, k := range keys {
				h.send(k)
			}
			if v := h.m.ui.Branches; !v.Open || v.Reading {
				t.Fatalf("the popup: open %v, reading %v", v.Open, v.Reading)
			}
			return h
		}
	}
	down := func(n int, last tea.KeyPressMsg) []tea.KeyPressMsg {
		var ks []tea.KeyPressMsg
		for range n {
			ks = append(ks, key(tea.KeyDown, 0))
		}
		return append(ks, last)
	}
	for _, sc := range []struct {
		name string
		rig  func(*testing.T) *harness
	}{{"models-claude", selector("claude-models.jsonl")}, {"models-claude-more", selector("claude-models.jsonl", down(3, key(tea.KeyEnter, 0))...)},
		{"models-claude-effort", selector("claude-models.jsonl", down(4, key(tea.KeyLeft, 0))...)}, {"models-codex", selector("codex-models.jsonl")},
		{"models-codex-more", selector("codex-models.jsonl", down(3, key(tea.KeyEnter, 0))...)},
		{"startup", startup}, {"dashboard", dash}, {"plus-menu", plusMenu}, {"attachments", attached}, {"attachments-sent", sent}, {"attachments-preview", preview}, {"dashboard-codex", codexDash}, {"inspect", inspected}, {"tree", tree}, {"details", details}, {"quota", quota}, {"approvals-auto", claudeAuto}, {"approvals-requested", codexRequested}, {"approvals-review", codexReview}, {"startup-projects", projectsRig(demoProjects(10), -1)}, {"startup-projects-focused", projectsRig(demoProjects(10), 6)}, {"startup-favourites", projectsRig(favouriteProjects(), 1)}, {"worktrees", worktreesRig(true)}, {"worktrees-new", worktreesRig(true, append([]tea.KeyPressMsg{typed('n')}, typedText("release-2.1")...)...)}, {"worktrees-session", worktreesRig(false, key(tea.KeyDown, 0))}, {"worktrees-diff", worktreesRig(false, key(tea.KeyDown, 0), typed('d'))}, {"worktrees-confirm", worktreesRig(false, key(tea.KeyDown, 0), typed('m'))}, {"file", file()}, {"file-full", file(typed('f'))}, {"file-search", file(append([]tea.KeyPressMsg{typed('/')}, append(typedText("colour"), key(tea.KeyEnter, 0))...)...)}, {"file-finder", finder}, {"explorer", explorer}, {"explorer-nerd", explorerRig(true)}, {"draft", draft()}, {"draft-full", draft(key('f', tea.ModCtrl))},
		{"source", source()}, {"source-none", sourceNone}, {"source-diff", source(key(tea.KeyEnter, 0))}, {"source-diff-wide", source(key(tea.KeyEnter, 0), typed('f'))},
		{"commit-menu", commitMenu}, {"commit-sixel", commitSixel}, {"commit-push", commitPush}, {"source-selection", sourceSelection},
		{"branches", branches()}, {"branches-search", branches(typedText("fe")...)}, {"branches-confirm", branches(key(tea.KeyUp, 0), key(tea.KeyEnter, 0))}} {
		szs := sizes
		if sc.name == "source-selection" {
			szs = append(slices.Clone(sizes), [2]int{208, 59}) // the acc tab maximised (plan 50, criterion 50.10)
		}
		for _, size := range szs {
			w, hgt := size[0], size[1]
			name := fmt.Sprintf("%s-%dx%d", sc.name, w, hgt)
			t.Run(name, func(t *testing.T) {
				synctest.Test(t, func(t *testing.T) {
					h := sc.rig(t)
					h.send(tea.WindowSizeMsg{Width: w, Height: hgt})
					frame := h.m.View().Content
					lines := strings.Split(frame, "\n")
					if len(lines) != hgt {
						t.Errorf("%d lines, want %d", len(lines), hgt)
					}
					for i, l := range lines {
						if n := lipgloss.Width(l); n > w {
							t.Errorf("line %d is %d columns wide, terminal %d: %q", i+1, n, w, plainText(l))
						}
					}
					text := plainText(frame)
					small := strings.Contains(text, "Terminal too small")
					if small != (w < render.MinWidth || hgt < render.MinHeight) {
						t.Errorf("too-small message shown %v at %dx%d", small, w, hgt)
					}
					if strings.HasPrefix(sc.name, "startup") && !small {
						// START SESSION shows whatever the project list holds (plan 15,
						// 1.4), above the footer (plan 51, design D8).
						tl := strings.Split(text, "\n")
						i := slices.IndexFunc(tl, func(l string) bool { return strings.Contains(l, "START SESSION →") })
						if i < 1 || i+1 >= len(tl) || !strings.Contains(tl[len(tl)-1], "Shortcuts:") {
							t.Errorf("START SESSION is not shown above the footer")
						}
					}
					// A mock session's every frame says it is mock data, full
					// screens included (reliability.md, Mock mode; plan 21,
					// item 5.7).
					if h.m.State().Context.Mock && !strings.Contains(text, render.MockBadge) {
						t.Errorf("a mock session's frame without %q", render.MockBadge)
					}
					golden(t, name, text)
				})
			})
		}
	}
}

// golden compares a frame with testdata/frames/<name>.txt; -update rewrites it.
func golden(t *testing.T, name, text string) {
	t.Helper()
	path := filepath.Join("..", "..", "testdata", "frames", name+".txt")
	text = strings.Join(trimRight(strings.Split(text, "\n")), "\n") + "\n"
	if *update {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(want); got != text {
		gl, wl := strings.Split(text, "\n"), strings.Split(got, "\n")
		for i := 0; i < len(gl) && i < len(wl); i++ {
			if gl[i] != wl[i] {
				t.Fatalf("%s differs at line %d:\n got %q\nwant %q\n(rerun with -update and review the diff)", path, i+1, gl[i], wl[i])
			}
		}
		t.Fatalf("%s: %d lines, golden has %d", path, len(gl), len(wl))
	}
}

// trimRight drops trailing spaces, so the golden files diff cleanly; the
// width check runs on the frame itself.
func trimRight(ls []string) []string {
	for i, l := range ls {
		ls[i] = strings.TrimRight(l, " ")
	}
	return ls
}

// TestStartupMessageFrames is proposal step 1, item 5: the startup screen
// with each kind of message, at 80x24 where they are tightest. Each message
// is wrapped whole, never cut.
func TestStartupMessageFrames(t *testing.T) {
	claudeOK := fakeCLI{provider.Detection{ID: "claude-code", Name: "Claude Code", Installed: true, Version: "2.1.300", Path: `C:\Users\dev\.local\bin\claude.exe`}}
	codexOK := fakeCLI{provider.Detection{ID: "codex", Name: "OpenAI Codex", Installed: true, Version: "0.145.0", Path: `C:\Users\dev\AppData\Local\Programs\OpenAI\Codex\bin\codex.exe`}}
	codexMissing := fakeCLI{provider.Detection{ID: "codex", Name: "OpenAI Codex", Problem: "codex not found on PATH"}}
	cases := []struct {
		name  string
		provs []provider.Provider
		typed string // "" keeps the default
		enter bool
		want  string // in the frame, whitespace collapsed
	}{
		{"no-cli", []provider.Provider{
			fakeCLI{provider.Detection{ID: "claude-code", Name: "Claude Code", Path: `C:\Users\dev\.local\bin\claude.exe`, Problem: "version unreadable: exit 1: error: the installation is damaged, reinstall Claude Code"}},
			fakeCLI{provider.Detection{ID: "codex", Name: "OpenAI Codex", Problem: "npm install found (codex.cmd), native codex.exe needed"}},
		}, "", true, "✗ No supported agent CLI found. Install Claude Code or Codex CLI, then restart."},
		{"does-not-exist", []provider.Provider{claudeOK, codexMissing}, `C:\Users\dev\projets\agent-comand-center`, true,
			`✗ Does not exist: C:\Users\…\projets\agent-comand-center`},
		{"wsl-not-found", []provider.Provider{claudeOK, codexOK}, "/mnt/q/acc-test-missing/app", true,
			`✗ Does not exist: Q:\acc-test-missing\app (the WSL mount root may not be /mnt/)`},
		{"live-linux-path", []provider.Provider{claudeOK, codexOK}, "/home/u/app", false,
			`✗ A Linux path. On Windows use C:\…, or /mnt/c/… for a Windows drive`},
		{"live-drive-relative", []provider.Provider{claudeOK, codexOK}, `C:dev\app`, false,
			`✗ Drive-relative path. Type a full path, for example C:\dev\app`},
		{"warning-wsl", []provider.Provider{claudeOK, codexOK}, `\\wsl.localhost\Ubuntu\home\u\app`, false,
			"! files in Linux, used by a Windows agent CLI: slow, and unverified"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				h := newHarness(t, Options{Providers: c.provs, Cwd: `C:\work\demo-project`, Home: `C:\Users\dev`})
				h.settle()
				if c.typed != "" {
					h.m.path.SetText(c.typed)
				}
				h.send(tea.WindowSizeMsg{Width: 80, Height: 24})
				if c.enter {
					h.send(key(tea.KeyEnter, 0))
				}
				frame := h.m.View().Content
				lines := strings.Split(frame, "\n")
				if len(lines) != 24 {
					t.Errorf("%d lines, want 24", len(lines))
				}
				for i, l := range lines {
					if n := lipgloss.Width(l); n > 80 {
						t.Errorf("line %d is %d columns wide", i+1, n)
					}
				}
				text := plainText(frame)
				// In NEW SESSION, whole wherever it wraps (plan 51, grid mode).
				if !homeSays(homePanel(frame, 0), c.want) {
					t.Errorf("NEW SESSION lacks %q", c.want)
				}
				if strings.Contains(text, "…\n") || strings.Contains(text, "… │") {
					t.Error("a line is cut")
				}
				golden(t, "startup-msg-"+c.name, text)
			})
		})
	}
}
