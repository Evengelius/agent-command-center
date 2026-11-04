package app

import (
	"slices"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/Evengelius/agent-command-center/internal/input"
	"github.com/Evengelius/agent-command-center/internal/provider"
	"github.com/Evengelius/agent-command-center/internal/provider/claude"
	"github.com/Evengelius/agent-command-center/internal/render"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// Plan 76, step 2 (design D5, brief amendment 96): Codex is paused, so the
// build holds Claude Code's adapter alone. A supported CLI without an adapter
// keeps its row, greyed and not choosable, reading "paused"; its history
// entries stay listed and start nothing; a request for it is said, never
// followed silently. Criteria 76.8 to 76.10.

// pausedRig is the home screen over Claude Code's adapter alone, at w x h,
// with the store list and dir in the field.
func pausedRig(t *testing.T, opt Options, list []input.Project, dir string, w, h int) (*harness, *wireClaude, *input.Projects) {
	t.Helper()
	wc := &wireClaude{}
	opt.Providers = []provider.Provider{installedClaude{&claude.Provider{Spawner: wc, Now: func() time.Time { return testNow }}}}
	opt.Cwd = dir
	if list != nil {
		opt.Projects = storeWith(t, list)
	}
	hr := newHarness(t, opt)
	hr.m.opt.Location = time.UTC
	hr.settle()
	hr.send(tea.WindowSizeMsg{Width: w, Height: h})
	if s := hr.m.State(); s.Screen != state.ScreenStartup || s.Startup.Phase != state.Asking {
		t.Fatalf("screen %q, phase %q", s.Screen, s.Startup.Phase)
	}
	return hr, wc, opt.Projects
}

// codexRow is the frame's row naming OpenAI Codex among the options: the
// home screen's, in a row of the left panel.
func codexRow(frame string) string {
	for l := range strings.SplitSeq(frame, "\n") {
		if strings.Contains(l, "OpenAI Codex") && strings.Contains(l, "○") {
			return l
		}
	}
	return ""
}

// TestPausedRow is criterion 76.8: the row of a CLI without an adapter keeps
// its place in both modes, greyed, reading "paused" where a version stands,
// not "not installed"; a click on it, the arrows and Enter choose nothing.
func TestPausedRow(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, wc, _ := pausedRig(t, Options{}, nil, t.TempDir(), 120, 40)
		row := codexRow(h.frame())
		if !strings.Contains(row, "OpenAI Codex paused") || strings.Contains(row, "not installed") || strings.Contains(h.frame(), "adapter not built") {
			t.Errorf("the text mode's row is %q:\n%s", row, h.frame())
		}
		if strings.Index(h.frame(), "Claude Code") > strings.Index(h.frame(), "OpenAI Codex") {
			t.Error("the paused row left its place")
		}

		// Not choosable: a click on its option, ↓ and ↑ on the question, Enter.
		sc := screenNow(h)
		clicked := false
		for y := range h.m.ui.Height {
			for x := range h.m.ui.Width {
				if p, it := sc.ItemAt(x, y); p == render.HomePanel && it == render.HomeItemOption+1 && !clicked {
					clicked = true
					mouseClick(t, h, x, y)
				}
			}
		}
		if !clicked {
			t.Fatal("no cell for the paused option")
		}
		if h.m.ui.Choice != claude.ProviderID {
			t.Errorf("a click chose the paused CLI: %q", h.m.ui.Choice)
		}
		for _, k := range []tea.KeyPressMsg{key(tea.KeyDown, 0), key(tea.KeyDown, 0), key(tea.KeyUp, 0)} {
			h.m.ui.Field = 0
			h.send(k)
		}
		if h.m.ui.Choice != claude.ProviderID {
			t.Errorf("the paused CLI was chosen: %q", h.m.ui.Choice)
		}
		h.m.ui.Field = 2
		h.send(key(tea.KeyEnter, 0))
		if sp := wc.spawns(); len(sp) != 1 || h.m.State().Context.Provider != claude.ProviderID {
			t.Errorf("Enter started %q on %q", sp, h.m.State().Context.Provider)
		}
	})
}

// TestPausedRowPicture is the same row in the picture mode, Sixel at 208x59.
func TestPausedRowPicture(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		// The picture mode's look, with Sixel at 208x59.
		if _, err := input.Glyph('H', 20, false); err != nil {
			t.Skipf("GDI: %v", err) // no font: the text screen, the golden frames'
		}
		h, _, _ := pausedRig(t, Options{}, nil, t.TempDir(), 208, 59)
		h.send(PrimaryDeviceAttributesEvent{61, 4, 6, 7, 14})
		sc := screenNow(h)
		if !sc.Home.Shown || !sc.Home.Look.Geom.Picture {
			t.Fatal("no picture")
		}
		if cl := sc.Home.Look.CLIs; len(cl) != 2 || cl[1].Version != render.PausedText || cl[1].Usable || cl[0].Version != "(2.1.300)" {
			t.Errorf("the picture's rows: %+v", cl)
		}
		if row := codexRow(h.frame()); row != "" && !strings.Contains(row, "paused") {
			t.Errorf("the grid row %q", row)
		}
	})
}

// TestPausedHistory is criterion 76.9: a paused CLI's entries stay listed,
// greyed; Enter and a double click on one start nothing, say "OpenAI Codex is
// paused" and remove nothing; Delete and Delete All work as before.
func TestPausedHistory(t *testing.T) {
	list := func(dir string) []input.Project {
		return []input.Project{{Path: dir, Used: testNow.Add(-time.Hour), History: histEntries(3)}} // the third is Codex's
	}
	said := "✗ OpenAI Codex is paused"
	synctest.Test(t, func(t *testing.T) {
		dir := t.TempDir()
		h, wc, ps := pausedRig(t, Options{}, list(dir), dir, 120, 40)
		start := histRead(t, ps)
		openHist(t, h)
		if rows := h.m.ui.History.Rows; len(rows) != 3 || rows[0].Dim || rows[1].Dim || !rows[2].Dim || rows[2].CLI != "OpenAI Codex" {
			t.Fatalf("the rows: %+v", rows)
		}
		if f := h.frame(); !strings.Contains(f, "OpenAI Codex") {
			t.Errorf("the Codex entry is not listed:\n%s", f)
		}
		h.send(key(tea.KeyEnd, 0))
		h.send(key(tea.KeyEnter, 0))
		if h.m.hist == nil || len(wc.spawns()) != 0 || h.m.State().Startup.Phase != state.Asking || !strings.Contains(h.frame(), said) || strings.Contains(h.frame(), "not installed") {
			t.Errorf("Enter: view %v, spawns %q, phase %q:\n%s", h.m.hist != nil, wc.spawns(), h.m.State().Startup.Phase, h.frame())
		}
		h.send(key(tea.KeyUp, 0))
		if strings.Contains(h.frame(), said) {
			t.Error("the note stays after a move")
		}
		x, y := histCell(t, h, render.HistoryItemRow+2)
		mouseClick(t, h, x, y)
		mouseClick(t, h, x, y)
		if h.m.hist == nil || len(wc.spawns()) != 0 || h.m.State().Startup.Phase != state.Asking || !strings.Contains(h.frame(), said) {
			t.Errorf("a double click: view %v, spawns %q:\n%s", h.m.hist != nil, wc.spawns(), h.frame())
		}
		if histRead(t, ps) != start {
			t.Error("the store changed")
		}
	})
	synctest.Test(t, func(t *testing.T) {
		// Delete removes the paused CLI's entry alone; Delete All the rest.
		dir := t.TempDir()
		h, _, ps := pausedRig(t, Options{}, list(dir), dir, 120, 40)
		openHist(t, h)
		h.send(key(tea.KeyEnd, 0))
		h.send(key(tea.KeyDelete, 0))
		h.send(key(tea.KeyLeft, 0))
		h.send(key(tea.KeyEnter, 0))
		want := list(dir)
		want[0].History = slices.Delete(slices.Clone(want[0].History), 2, 3)
		if got := histRead(t, ps); got != histFile(t, want) {
			t.Errorf("after Delete:\n%s\nwant\n%s", got, histFile(t, want))
		}
	})
	synctest.Test(t, func(t *testing.T) {
		dir := t.TempDir()
		h, _, ps := pausedRig(t, Options{}, list(dir), dir, 120, 40)
		openHist(t, h)
		h.send(key(tea.KeyDelete, tea.ModShift))
		if !strings.Contains(h.frame(), "DELETE ALL 3 SESSIONS OF ") || !strings.Contains(h.frame(), "Claude Code and OpenAI Codex keep their own records") {
			t.Fatalf("Delete All's question:\n%s", h.frame())
		}
		h.send(key(tea.KeyLeft, 0))
		h.send(key(tea.KeyEnter, 0))
		want := list(dir)
		want[0].History = nil
		if got := histRead(t, ps); got != histFile(t, want) {
			t.Errorf("after Delete All:\n%s\nwant\n%s", got, histFile(t, want))
		}
	})
}

// TestPausedNoFallback is criterion 76.10: --provider codex chooses Claude
// Code and the CLI question's note says "OpenAI Codex is paused"; a saved
// Codex choice gives the first CLI offered, the paused row in view.
func TestPausedNoFallback(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, _, _ := pausedRig(t, Options{Config: Config{Provider: "codex"}}, nil, t.TempDir(), 120, 40)
		s := h.m.State()
		if h.m.ui.Choice != claude.ProviderID || s.Startup.Field != "provider" || s.Startup.Message != "OpenAI Codex is paused" {
			t.Errorf("choice %q, field %q, message %q", h.m.ui.Choice, s.Startup.Field, s.Startup.Message)
		}
		if f := h.frame(); !strings.Contains(f, "✗ OpenAI Codex is paused") {
			t.Errorf("the note is not shown:\n%s", f)
		}
	})
	synctest.Test(t, func(t *testing.T) {
		wc := &wireClaude{}
		h := newHarness(t, Options{Providers: []provider.Provider{installedClaude{&claude.Provider{Spawner: wc, Now: func() time.Time { return testNow }}}}, Cwd: t.TempDir()})
		h.m.ui.Choice = "codex" // a choice saved before detection
		h.settle()
		h.send(tea.WindowSizeMsg{Width: 120, Height: 40})
		if h.m.ui.Choice != claude.ProviderID || !strings.Contains(codexRow(h.frame()), "OpenAI Codex paused") {
			t.Errorf("choice %q, the row %q", h.m.ui.Choice, codexRow(h.frame()))
		}
	})
}

// pausedNoCLI is the note when Claude Code is not installed and Codex is
// paused: it names the paused CLI and asks for Claude Code alone.
const pausedNoCLI = "✗ No usable agent CLI: OpenAI Codex is paused. Install Claude Code, then restart."

// TestPausedNoCLINote is the "no CLI" message with a CLI paused: built from
// the CLIs list, and with --provider codex shown alone, as the note already
// says Codex is paused. The old text, with both adapters and neither CLI
// installed, stays byte for byte: TestStartupGate's "neither CLI: Start
// disabled" and TestStartupMessageFrames's "no-cli" case check it.
func TestPausedNoCLINote(t *testing.T) {
	for _, c := range []struct {
		name string
		cfg  Config
	}{{"no provider asked", Config{}}, {"--provider codex", Config{Provider: "codex"}}} {
		t.Run(c.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				h := newHarness(t, Options{Config: c.cfg, Cwd: t.TempDir(), Providers: []provider.Provider{
					fakeCLI{provider.Detection{ID: "claude-code", Name: "Claude Code", Problem: "claude not found on PATH"}},
				}})
				h.settle()
				h.send(tea.WindowSizeMsg{Width: 120, Height: 40})
				panel := homePanel(h.m.View().Content, 0)
				if !homeSays(panel, pausedNoCLI) {
					t.Errorf("the panel lacks %q:\n%s", pausedNoCLI, h.frame())
				}
				if n := strings.Count(strings.Join(strings.Fields(strings.Join(panel, " ")), ""), "OpenAICodexispaused"); n != 1 {
					t.Errorf("\"OpenAI Codex is paused\" is said %d times, want 1:\n%s", n, h.frame())
				}
				if homeSays(panel, "No supported agent CLI found") || homeSays(panel, "Codex CLI") {
					t.Errorf("the old text is shown:\n%s", h.frame())
				}
				h.send(key(tea.KeyEnter, 0))
				if s := h.m.State(); s.Startup.Phase != state.Asking || s.Screen != state.ScreenStartup {
					t.Errorf("Start went on: phase %q screen %q", s.Startup.Phase, s.Screen)
				}
			})
		})
	}
}

// TestPausedProxyLines: while Codex is paused (its adapter not in the build)
// neither Codex's proxy line nor its section's problem shows, from the first
// frame, detection included; with its adapter in the build both show.
func TestPausedProxyLines(t *testing.T) {
	const problem = "config.json: codex.proxy must start with http:// or https://; no proxy for Codex"
	opt := func(provs ...provider.Provider) Options {
		return Options{CodexProxy: proxyAddr, CodexProxyProblem: problem, Cwd: t.TempDir(), Providers: provs}
	}
	says := func(h *harness, s string) bool { return homeSays(homePanel(h.m.View().Content, 0), s) }
	synctest.Test(t, func(t *testing.T) {
		gate := make(chan struct{})
		wc := &wireClaude{}
		h := newHarness(t, opt(gatedClaude{installedClaude{&claude.Provider{Spawner: wc, Now: func() time.Time { return testNow }}}, gate}))
		h.settle()
		h.send(tea.WindowSizeMsg{Width: 120, Height: 40})
		if p := h.m.State().Startup.Phase; p != state.Detecting {
			t.Fatalf("phase %q, want Detecting", p)
		}
		if says(h, "Codex through the proxy") || says(h, "no proxy for Codex") {
			t.Errorf("during detection, with Codex paused:\n%s", h.frame())
		}
		close(gate)
		h.settle()
		if h.m.State().Startup.Phase != state.Asking {
			t.Fatalf("phase %q", h.m.State().Startup.Phase)
		}
		if says(h, "Codex through the proxy") || says(h, "no proxy for Codex") {
			t.Errorf("after detection, with Codex paused:\n%s", h.frame())
		}
	})
	synctest.Test(t, func(t *testing.T) {
		// Both adapters: both lines show, while detecting and after (the
		// proxy tests of plan 22 check the lines in full).
		gate := make(chan struct{})
		wc := &wireClaude{}
		h := newHarness(t, opt(gatedClaude{installedClaude{&claude.Provider{Spawner: wc, Now: func() time.Time { return testNow }}}, gate}, bothCLIs[1]))
		h.settle()
		h.send(tea.WindowSizeMsg{Width: 120, Height: 40})
		if !says(h, "Codex through the proxy "+proxyAddr+" (config.json)") || !says(h, "✗ "+problem) {
			t.Errorf("during detection, with Codex in the build:\n%s", h.frame())
		}
		close(gate)
		h.settle()
		if !says(h, "Codex through the proxy "+proxyAddr+" (config.json)") || !says(h, "✗ "+problem) {
			t.Errorf("after detection, with Codex in the build:\n%s", h.frame())
		}
	})
}
