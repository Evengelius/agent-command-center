package app

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"testing/synctest"

	tea "charm.land/bubbletea/v2"

	"github.com/Evengelius/agent-command-center/internal/event"
	"github.com/Evengelius/agent-command-center/internal/render"
)

// Plan 49's criterion 49.3 (design D3; audit finding 4): the overlay
// matrix. Ten layers over the dashboard, each opened as the owner opens it,
// alone and over each other: which are open, which the frame draws, which
// takes the next key (Esc) and the next click, and where the focus goes.

// layer is one of the interaction layers, as the owner opens it.
type layer struct {
	name   string
	open   func(t *testing.T, h *harness) // its opener, whatever is open
	isOpen func(m *Model) bool
	drawn  func(h *harness) bool // its marks in the frame drawn last
}

// anyItem asks layerCell for any cell of the panel: the panels number some
// of their items below zero.
const anyItem = -1 << 30

// layerCell is the first cell of panel p, with item unless it is anyItem,
// on the frame drawn last; ok false when there is none.
func layerCell(h *harness, p byte, item int) (x, y int, ok bool) {
	for y := range h.m.ui.Height {
		for x := range h.m.ui.Width {
			if q, it := h.m.screen.ItemAt(x, y); q == p && (item == anyItem || it == item) {
				return x, y, true
			}
		}
	}
	return 0, 0, false
}

// clickOn clicks the first cell of panel p (and item), if the frame shows
// one, as the program would: its commands run in their own goroutines.
func clickOn(h *harness, p byte, item int) {
	h.m.View()
	if x, y, ok := layerCell(h, p, item); ok {
		h.send(click(x, y))
		h.send(tea.MouseReleaseMsg{X: x, Y: y, Button: tea.MouseLeft})
	}
}

// windowTitled says the frame draws a window whose top row, at the
// window's place, starts with title.
func windowTitled(h *harness, x, y int, title string) bool {
	lines := strings.Split(h.frame(), "\n")
	if y >= len(lines) {
		return false
	}
	r := []rune(lines[y])
	return x < len(r) && strings.HasPrefix(string(r[x:]), "╭─ "+title)
}

var layers = []layer{
	{"plus", func(t *testing.T, h *harness) { clickOn(h, '+', anyItem) },
		func(m *Model) bool { return m.ui.Plus },
		func(h *harness) bool { return strings.Contains(h.frame(), "Upload from computer") }},
	{"models", func(t *testing.T, h *harness) { h.send(key('p', tea.ModAlt)) },
		func(m *Model) bool { return m.ui.Models.Open },
		func(h *harness) bool { h.m.View(); return h.m.screen.Selector.Shown }},
	{"modes", func(t *testing.T, h *harness) { h.send(key(tea.KeyF4, 0)) },
		func(m *Model) bool { return m.ui.Modes.Open },
		func(h *harness) bool { h.m.View(); return h.m.screen.Modes.Shown }},
	{"dropdown", func(t *testing.T, h *harness) { clickOn(h, render.SourcePanel, render.ItemCommitMore) },
		func(m *Model) bool { return m.ui.Source.Menu },
		func(h *harness) bool {
			h.m.View()
			_, _, ok := layerCell(h, render.SourcePanel, render.ItemCommitMenu)
			return ok
		}},
	{"question", func(t *testing.T, h *harness) {
		h.m.View()
		if _, _, ok := layerCell(h, render.SourcePanel, render.ItemMsg); !ok {
			return
		}
		clickOn(h, render.SourcePanel, render.ItemMsg)
		h.send(typed('m'))
		h.send(ctrlEnter)
	},
		func(m *Model) bool { return m.ui.Source.Ask > 0 },
		func(h *harness) bool {
			h.m.View()
			_, _, ok := layerCell(h, render.SourcePanel, render.ItemAskYes)
			return ok
		}},
	{"viewer", func(t *testing.T, h *harness) {
		h.send(key(tea.KeyUp, 0))
		h.send(key(tea.KeyF3, 0))
	},
		func(m *Model) bool { return m.ui.File.Open },
		func(h *harness) bool {
			x, y, _, _ := render.FileRect(h.m.ui.Width, h.m.ui.Height, false)
			return windowTitled(h, x, y, "main.go")
		}},
	{"finder", func(t *testing.T, h *harness) {
		clickOn(h, render.SidePanel, render.SideExplorer)
		h.send(typed('/'))
	},
		func(m *Model) bool { return m.ui.Finder.Open },
		func(h *harness) bool {
			x, y, _, _ := render.FileRect(h.m.ui.Width, h.m.ui.Height, false)
			return windowTitled(h, x, y, "Quick open")
		}},
	{"draft", func(t *testing.T, h *harness) { h.send(key('g', tea.ModCtrl)) },
		func(m *Model) bool { return m.ui.DraftWin.Open },
		func(h *harness) bool {
			x, y, _, _ := render.DraftRect(h.m.ui)
			return windowTitled(h, x, y, "Draft")
		}},
	{"keys", func(t *testing.T, h *harness) { h.send(key(tea.KeyF1, 0)) },
		func(m *Model) bool { return m.ui.Keys },
		func(h *harness) bool { return strings.Contains(h.frame(), "F1 or Esc closes") }},
	{"worktrees", func(t *testing.T, h *harness) { h.send(key('w', tea.ModCtrl)) },
		func(m *Model) bool { return m.wt != nil },
		func(h *harness) bool {
			return strings.Contains(h.frame(), "Worktrees ") && !strings.Contains(h.frame(), "▣")
		}},
}

// layerRig is the dashboard every layer opens over, at 200x60: a Codex
// session with the model and modes selectors, a turn that read main.go,
// Source Control docked with changes and nothing staged, the focus in zone
// B's editor. Each message is followed by View, as in the program, which
// closes a selector the frame cannot draw.
func layerRig(t *testing.T) *harness {
	t.Helper()
	r, g, root := sourceRig(t)
	h := r.h
	h.check = func(string) {}
	s := r.script
	s.mu.Lock()
	s.models, s.auto = true, true
	s.mu.Unlock()
	r.emit(event.ModelsListed, event.Models{Entries: selectorModels, Source: "model/list"})
	r.emit(event.ModelSettings, event.Settings{Model: "claude-fable-5-1", Choice: "fable", Effort: "high", EffortKnown: true, Source: "thread/settings"})
	r.emit(event.ApprovalMode, event.Mode{Mode: event.ModeManual})
	r.emit(event.TurnStarted, nil)
	r.emit(event.ToolStarted, event.Tool{ToolID: "t1", Name: "Read", Input: toolInput(t, map[string]any{"file_path": filepath.Join(root, "main.go")})})
	r.emit(event.ToolCompleted, event.Tool{ToolID: "t1", Name: "Read", Output: "100 lines"})
	r.emit(event.TurnCompleted, event.TurnEnd{Outcome: "completed", Subtype: "success"})
	h.settle()
	for i := range g.snap.Entries {
		if e := &g.snap.Entries[i]; e.X != '?' {
			e.X = ' '
			if e.Y == ' ' {
				e.Y = 'M'
			}
		}
	}
	h.m.opt.Git = &fakeWrite{fakeGit: g}
	showSource(t, h)
	refresh(h)
	clickOn(h, 'E', anyItem) // the editor: zone B has the focus
	if h.m.ui.Focus != "B" || !h.m.ui.Source.Open {
		t.Fatalf("the rig: focus %q, Source Control %v\n%s", h.m.ui.Focus, h.m.ui.Source.Open, h.frame())
	}
	return h
}

// openLayers names the layers open, in the list's order.
func openLayers(m *Model) string {
	var out []string
	for _, l := range layers {
		if l.isOpen(m) {
			out = append(out, l.name)
		}
	}
	return strings.Join(out, ",")
}

// drawnLayers names the layers the frame draws.
func drawnLayers(h *harness) string {
	var out []string
	for _, l := range layers {
		if l.drawn(h) {
			out = append(out, l.name)
		}
	}
	return strings.Join(out, ",")
}

// observePair opens a, then b's opener (none when b is nil), and says what is
// open and drawn; then what Esc leaves open and where the focus is; then,
// from the same two openings, what a click at the top-left cell leaves open.
func observePair(t *testing.T, a, b *layer) string {
	var got string
	synctest.Test(t, func(t *testing.T) {
		h := layerRig(t)
		a.open(t, h)
		if !a.isOpen(h.m) {
			t.Fatalf("%s did not open from the rig:\n%s", a.name, h.frame())
		}
		if b != nil {
			b.open(t, h)
		}
		open, drawn := openLayers(h.m), drawnLayers(h)
		h.send(key(tea.KeyEscape, 0))
		esc, focus := openLayers(h.m), h.m.ui.Focus
		got = fmt.Sprintf("open=%s drawn=%s esc=%s focus=%s", open, drawn, esc, focus)
	})
	synctest.Test(t, func(t *testing.T) {
		h := layerRig(t)
		a.open(t, h)
		if b != nil {
			b.open(t, h)
		}
		h.send(click(0, 0))
		h.send(tea.MouseReleaseMsg{X: 0, Y: 0, Button: tea.MouseLeft})
		got += " click=" + openLayers(h.m)
	})
	return got
}

// TestLayerMatrix runs the matrix. With ACC_LAYERS_RECORD set it prints
// every cell instead, for the expectations to be reviewed.
func TestLayerMatrix(t *testing.T) {
	record := os.Getenv("ACC_LAYERS_RECORD") != ""
	var cells []string
	t.Cleanup(func() {
		if record {
			slices.Sort(cells)
			_ = os.WriteFile(os.Getenv("ACC_LAYERS_RECORD"), []byte(strings.Join(cells, "\n")+"\n"), 0o644)
		}
	})
	for i := range layers {
		a := &layers[i]
		for j := -1; j < len(layers); j++ {
			if j == i {
				continue
			}
			var b *layer
			name := a.name
			if j >= 0 {
				b = &layers[j]
				name += ">" + b.name
			}
			// Each cell in bubbles of its own, one after the other: render draws
			// on one goroutine, making its tinted styles on first use.
			t.Run(name, func(t *testing.T) {
				got := observePair(t, a, b)
				if record {
					cells = append(cells, fmt.Sprintf("%-22s %s", name, got))
					return
				}
				if want, ok := layerWant[name]; !ok || got != want {
					t.Errorf("\n got %s\nwant %s", got, want)
				}
			})
		}
	}
}

// layerWant is each cell as it must be: the layer opened on top takes the
// keys and the clicks; a popup closes when another layer opens over it; a
// popup closes on a key it does not take or a click outside it (the commit
// dropdown and the staging question doing nothing else with that click); a
// full view and the draft window keep their keys and clicks; the file
// window, the viewer or the quick open, keeps its keys and closes on a left
// click outside it as Esc does, doing nothing else with that click (plan
// 58); Esc closes the top layer. Five cells failed before design D7 (plan
// 49, progress), a popup left open under the layer opened over it and
// taking the next click there: dropdown>worktrees, question>worktrees,
// question>keys, question>models and question>modes. Plan 58 moved 19 cells:
// the 16 whose click at the top-left cell met the window, closed now, and
// three whose second layer's opener clicks outside the viewer, closing it
// instead (viewer>dropdown, viewer>question; viewer>finder's '/' then
// typed in the Explorer, which opens the quick open).
var layerWant = map[string]string{
	"draft":              "open=draft drawn=draft esc= focus=B click=draft",
	"draft>dropdown":     "open=draft drawn=draft esc= focus=B click=draft",
	"draft>finder":       "open=draft drawn=draft esc= focus=B click=draft",
	"draft>keys":         "open=draft,keys drawn=keys esc=draft focus=B click=draft,keys",
	"draft>models":       "open=draft drawn=draft esc= focus=B click=draft",
	"draft>modes":        "open=draft drawn=draft esc= focus=B click=draft",
	"draft>plus":         "open=draft drawn=draft esc= focus=B click=draft",
	"draft>question":     "open=draft drawn=draft esc= focus=B click=draft",
	"draft>viewer":       "open= drawn= esc= focus=B click=",
	"draft>worktrees":    "open=draft,worktrees drawn=worktrees esc=draft focus=B click=draft,worktrees",
	"dropdown":           "open=dropdown drawn=dropdown esc= focus=R click=",
	"dropdown>draft":     "open= drawn= esc= focus=B click=",
	"dropdown>finder":    "open= drawn= esc= focus=B click=",
	"dropdown>keys":      "open=keys drawn=keys esc= focus=R click=keys",
	"dropdown>models":    "open=models drawn=models esc= focus=R click=",
	"dropdown>modes":     "open=modes drawn=modes esc= focus=R click=",
	"dropdown>plus":      "open= drawn= esc= focus=B click=",
	"dropdown>question":  "open= drawn= esc= focus=B click=",
	"dropdown>viewer":    "open= drawn= esc= focus=B click=",
	"dropdown>worktrees": "open=worktrees drawn=worktrees esc= focus=R click=worktrees",
	"finder":             "open=finder drawn=finder esc= focus=R click=",
	"finder>draft":       "open=finder drawn=finder esc= focus=R click=",
	"finder>dropdown":    "open=finder drawn=finder esc= focus=R click=",
	"finder>keys":        "open=finder,keys drawn=keys esc=finder focus=R click=finder,keys",
	"finder>models":      "open=finder drawn=finder esc= focus=R click=",
	"finder>modes":       "open=finder drawn=finder esc= focus=R click=",
	"finder>plus":        "open=finder drawn=finder esc= focus=R click=",
	"finder>question":    "open=finder drawn=finder esc= focus=R click=",
	"finder>viewer":      "open= drawn= esc= focus=B click=",
	"finder>worktrees":   "open=finder,worktrees drawn=worktrees esc=finder focus=R click=finder,worktrees",
	"keys":               "open=keys drawn=keys esc= focus=B click=keys",
	"keys>draft":         "open=keys drawn=keys esc= focus=B click=keys",
	"keys>dropdown":      "open=keys drawn=keys esc= focus=B click=keys",
	"keys>finder":        "open=keys drawn=keys esc= focus=B click=keys",
	"keys>models":        "open=keys drawn=keys esc= focus=B click=keys",
	"keys>modes":         "open=keys drawn=keys esc= focus=B click=keys",
	"keys>plus":          "open=keys drawn=keys esc= focus=B click=keys",
	"keys>question":      "open=keys drawn=keys esc= focus=B click=keys",
	"keys>viewer":        "open=keys drawn=keys esc= focus=B click=keys",
	"keys>worktrees":     "open=keys,worktrees drawn=worktrees esc=keys focus=B click=keys,worktrees",
	"models":             "open=models drawn=models esc= focus=B click=",
	"models>draft":       "open=draft drawn=draft esc= focus=B click=draft",
	"models>dropdown":    "open=dropdown drawn=dropdown esc= focus=R click=",
	"models>finder":      "open=finder drawn=finder esc= focus=R click=",
	"models>keys":        "open=keys drawn=keys esc= focus=B click=keys",
	"models>modes":       "open=modes drawn=modes esc= focus=B click=",
	"models>plus":        "open=plus drawn=plus esc= focus=B click=",
	"models>question":    "open=question drawn=question esc= focus=R click=",
	"models>viewer":      "open= drawn= esc= focus=B click=",
	"models>worktrees":   "open=worktrees drawn=worktrees esc= focus=B click=worktrees",
	"modes":              "open=modes drawn=modes esc= focus=B click=",
	"modes>draft":        "open=draft drawn=draft esc= focus=B click=draft",
	"modes>dropdown":     "open=dropdown drawn=dropdown esc= focus=R click=",
	"modes>finder":       "open=finder drawn=finder esc= focus=R click=",
	"modes>keys":         "open=keys drawn=keys esc= focus=B click=keys",
	"modes>models":       "open=models drawn=models esc= focus=B click=",
	"modes>plus":         "open=plus drawn=plus esc= focus=B click=",
	"modes>question":     "open=question drawn=question esc= focus=R click=",
	"modes>viewer":       "open= drawn= esc= focus=B click=",
	"modes>worktrees":    "open=worktrees drawn=worktrees esc= focus=B click=worktrees",
	"plus":               "open=plus drawn=plus esc= focus=B click=",
	"plus>draft":         "open=draft drawn=draft esc= focus=B click=draft",
	"plus>dropdown":      "open=dropdown drawn=dropdown esc= focus=R click=",
	"plus>finder":        "open=finder drawn=finder esc= focus=R click=",
	"plus>keys":          "open=keys drawn=keys esc= focus=B click=keys",
	"plus>models":        "open=models drawn=models esc= focus=B click=",
	"plus>modes":         "open=modes drawn=modes esc= focus=B click=",
	"plus>question":      "open=question drawn=question esc= focus=R click=",
	"plus>viewer":        "open=viewer drawn=viewer esc= focus=R click=",
	"plus>worktrees":     "open=worktrees drawn=worktrees esc= focus=B click=worktrees",
	"question":           "open=question drawn=question esc= focus=R click=",
	"question>draft":     "open=question drawn=question esc= focus=R click=",
	"question>dropdown":  "open= drawn= esc= focus=B click=",
	"question>finder":    "open= drawn= esc= focus=B click=",
	"question>keys":      "open=keys drawn=keys esc= focus=R click=keys",
	"question>models":    "open=models drawn=models esc= focus=R click=",
	"question>modes":     "open=modes drawn=modes esc= focus=R click=",
	"question>plus":      "open= drawn= esc= focus=B click=",
	"question>viewer":    "open= drawn= esc= focus=B click=",
	"question>worktrees": "open=worktrees drawn=worktrees esc= focus=R click=worktrees",
	"viewer":             "open=viewer drawn=viewer esc= focus=R click=",
	"viewer>draft":       "open=viewer drawn=viewer esc= focus=R click=",
	"viewer>dropdown":    "open= drawn= esc= focus=B click=",
	"viewer>finder":      "open=finder drawn=finder esc= focus=R click=",
	"viewer>keys":        "open=viewer,keys drawn=keys esc=viewer focus=B click=viewer,keys",
	"viewer>models":      "open=viewer drawn=viewer esc= focus=R click=",
	"viewer>modes":       "open=viewer drawn=viewer esc= focus=R click=",
	"viewer>plus":        "open=viewer drawn=viewer esc= focus=R click=",
	"viewer>question":    "open= drawn= esc= focus=B click=",
	"viewer>worktrees":   "open=viewer,worktrees drawn=worktrees esc=viewer focus=B click=viewer,worktrees",
	"worktrees":          "open=worktrees drawn=worktrees esc= focus=B click=worktrees",
	"worktrees>draft":    "open=worktrees drawn=worktrees esc= focus=B click=worktrees",
	"worktrees>dropdown": "open=worktrees drawn=worktrees esc= focus=B click=worktrees",
	"worktrees>finder":   "open=worktrees drawn=worktrees esc= focus=B click=worktrees",
	"worktrees>keys":     "open=worktrees drawn=worktrees esc= focus=B click=worktrees",
	"worktrees>models":   "open=worktrees drawn=worktrees esc= focus=B click=worktrees",
	"worktrees>modes":    "open=worktrees drawn=worktrees esc= focus=B click=worktrees",
	"worktrees>plus":     "open=worktrees drawn=worktrees esc= focus=B click=worktrees",
	"worktrees>question": "open=worktrees drawn=worktrees esc= focus=B click=worktrees",
	"worktrees>viewer":   "open=worktrees drawn=worktrees esc= focus=B click=worktrees",
}

// TestPassedKeyGoesToTheDashboard is criterion 49.8's other side (design
// D7): a key the layer on top passes on goes to the dashboard, never to a
// window under it. The key overlay over the draft window passes Ctrl+R on,
// which keeps its dashboard meaning, the resume: the hidden window took it
// before.
func TestPassedKeyGoesToTheDashboard(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := layerRig(t)
		h.send(key('g', tea.ModCtrl))
		h.send(key(tea.KeyF1, 0))
		if got := openLayers(h.m); got != "draft,keys" {
			t.Fatalf("open %q, want the key overlay over the draft window", got)
		}
		h.send(key('r', tea.ModCtrl))
		if s := h.m.State(); !slices.Contains(s.Notes, "action: nothing to resume") {
			t.Errorf("Ctrl+R under the key overlay did not reach the dashboard: notes %q", s.Notes)
		}
	})
}
