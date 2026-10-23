package app

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"testing/synctest"

	tea "charm.land/bubbletea/v2"

	"github.com/Evengelius/agent-command-center/internal/event"
	"github.com/Evengelius/agent-command-center/internal/input"
	"github.com/Evengelius/agent-command-center/internal/input/imagetest"
	"github.com/Evengelius/agent-command-center/internal/provider"
	"github.com/Evengelius/agent-command-center/internal/render"
)

// Plan 34 (brief amendment 34): the model selector worked with the keys
// and the mouse over a scripted session, which records each change the
// application hands it.

var selectorModels = []event.ModelEntry{
	{ID: "default", Model: "claude-opus-5-5", Name: "Default (recommended)", Description: "Opus 5.5 · Best for everyday, complex tasks", Efforts: []string{"low", "medium", "high", "xhigh", "max"}},
	{ID: "opus", Model: "claude-opus-5-5", Name: "Opus 5.5", Description: "For complex work and everyday tasks", Efforts: []string{"low", "medium", "high", "xhigh", "max"}},
	{ID: "fable", Model: "claude-fable-5-1", Name: "Fable 5.1", Description: "For your toughest challenges", Efforts: []string{"low", "medium", "high", "xhigh", "max"}},
	{ID: "sonnet", Model: "claude-sonnet-5-5", Name: "Sonnet 5.5", Description: "Most efficient for simpler tasks", Efforts: []string{"low", "medium", "high", "xhigh", "max"}},
	{ID: "haiku", Model: "claude-haiku-5-5", Name: "Haiku 5.5", Description: "Fastest for quick answers", Efforts: []string{"low", "medium", "high", "xhigh", "max"}},
	{ID: "claude-haiku-4-5-20251001", Model: "claude-haiku-4-5-20251001", Name: "Haiku 4.5", Description: "Fastest for quick answers"},
	{ID: "claude-opus-4-6", Model: "claude-opus-4-6", Name: "Opus 4.6", Description: "Best for everyday, complex tasks", Efforts: []string{"low", "medium", "high", "max"}},
}

// selectorRig is a ready scripted Claude Code session with the models
// listed and Fable 5.1 at High with Ultracode in force.
func selectorRig(t *testing.T) (*harness, *scriptSession) {
	t.Helper()
	h, cli := startScript(t, "claude-code")
	s := cli.sess
	s.models, s.ultracode = true, true
	emit := func(k event.Kind, p any) {
		s.events <- event.Event{Provider: "claude-code", Kind: k, Origin: event.Observed, Payload: p}
	}
	emit(event.SessionSpawned, event.Spawned{Argv: []string{`C:\Tools\claude.exe`}})
	emit(event.SessionReady, event.Ready{Detail: "scripted"})
	emit(event.ModelsListed, event.Models{Entries: selectorModels, Source: "initialize"})
	emit(event.ModelSettings, event.Settings{Model: "claude-fable-5-1", Choice: "fable", Effort: "high", EffortKnown: true,
		Ultracode: true, UltracodeAvailable: true, UltracodeKnown: true, Source: "get_settings", Undocumented: true})
	h.settle()
	if !strings.Contains(frameText(h), "Fable 5.1 High · Ultracode") {
		t.Fatalf("no pill:\n%s", frameText(h))
	}
	return h, s
}

func (s *scriptSession) modelChanges() []provider.ModelRequest {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]provider.ModelRequest(nil), s.changes...)
}

// clickText clicks the first cell of the text's nth occurrence.
func clickText(t *testing.T, h *harness, text string, nth int) {
	t.Helper()
	x, y := at(t, h, text, nth)
	mouseAt(t, h, click(x, y))
	mouseAt(t, h, tea.MouseReleaseMsg{X: x, Y: y, Button: tea.MouseLeft})
}

// TestSelectorKeys is criteria 14 and 16: Alt+P opens the selector on the
// model in force; ↑↓ move the cursor; Enter acts on its row and never
// sends the draft, which stays with its text; Esc closes the effort list,
// then the selector; Alt+P closes it too.
func TestSelectorKeys(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, s := selectorRig(t)
		h.send(tea.PasteMsg{Content: "a draft kept"})
		h.send(key('p', tea.ModAlt))
		v := h.m.ui.Models
		if !v.Open || v.Cursor != 2 || v.More || !v.Ultracode {
			t.Fatalf("opened %+v, want the cursor on Fable 5.1", v)
		}
		if f := frameText(h); !strings.Contains(f, "CLAUDE MODELS") || !strings.Contains(f, "Current session") || !strings.Contains(f, render.UltracodeDescription) {
			t.Errorf("the selector:\n%s", f)
		}
		h.send(key(tea.KeyDown, 0)) // Sonnet 5.5
		h.send(key(tea.KeyEnter, 0))
		if got := s.modelChanges(); len(got) != 1 || got[0].Model != "sonnet" || got[0].Effort != "" || got[0].Ultracode != nil {
			t.Errorf("changes %+v", got)
		}
		if sent, _, _ := s.calls(); len(sent) != 0 {
			t.Errorf("Enter sent %q", sent)
		}
		if h.m.edit.Text() != "a draft kept" || h.m.ui.Focus != "B" {
			t.Errorf("draft %q, focus %q", h.m.edit.Text(), h.m.ui.Focus)
		}
		for range 3 {
			h.send(key(tea.KeyDown, 0)) // Haiku 5.5, More models, Effort
		}
		h.send(key(tea.KeyEnter, 0))
		if !h.m.ui.Models.Effort {
			t.Fatalf("Enter on Effort: %+v", h.m.ui.Models)
		}
		h.send(key(tea.KeyEsc, 0))
		if v := h.m.ui.Models; !v.Open || v.Effort {
			t.Errorf("Esc in the effort list: %+v", v)
		}
		h.send(key(tea.KeyEsc, 0))
		if h.m.ui.Models.Open {
			t.Error("Esc did not close the selector")
		}
		h.send(key('p', tea.ModAlt))
		h.send(key('p', tea.ModAlt))
		if h.m.ui.Models.Open {
			t.Error("Alt+P did not close the selector")
		}
		if sent, _, _ := s.calls(); len(sent) != 0 || h.m.edit.Text() != "a draft kept" {
			t.Errorf("sent %q, draft %q", sent, h.m.edit.Text())
		}
	})
}

// TestSelectorMouse is criteria 13, 15 and 16: a click on the pill opens
// the selector; the pointer moves the cursor; a click chooses a model, but
// the pill keeps what is in force until the session answers; More models
// expands; the effort box opens its list and a level is chosen; a click on
// Ultracode turns it off; a click outside closes the selector and goes on.
func TestSelectorMouse(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, s := selectorRig(t)
		clickText(t, h, "Fable 5.1 High", 0)
		if !h.m.ui.Models.Open {
			t.Fatal("a click on the pill did not open the selector")
		}
		x, y := at(t, h, "Sonnet 5.5", 0)
		mouseAt(t, h, tea.MouseMotionMsg{X: x + 2, Y: y})
		if h.m.ui.Models.Cursor != 3 {
			t.Errorf("hover: cursor %d, want 3", h.m.ui.Models.Cursor)
		}
		clickText(t, h, "Sonnet 5.5", 0)
		if got := s.modelChanges(); len(got) != 1 || got[0].Model != "sonnet" {
			t.Fatalf("changes %+v", got)
		}
		if !strings.Contains(frameText(h), "Fable 5.1 High · Ultracode") {
			t.Error("the pill changed before the answer")
		}
		s.events <- event.Event{Provider: "claude-code", Kind: event.ModelSettings, Origin: event.Observed, Payload: event.Settings{Model: "claude-sonnet-5-5",
			Choice: "sonnet", Effort: "high", EffortKnown: true, Ultracode: true, UltracodeAvailable: true, UltracodeKnown: true, Answers: true}}
		h.settle()
		if f := frameText(h); !strings.Contains(f, "Sonnet 5.5 High · Ultracode") {
			t.Errorf("the pill after the answer:\n%s", f)
		}
		clickText(t, h, "More models", 0)
		if v := h.m.ui.Models; !v.More || !strings.Contains(frameText(h), "Opus 4.6") {
			t.Errorf("More models: %+v\n%s", v, frameText(h))
		}
		x, y = at(t, h, "Effort:", 0)
		mouseAt(t, h, click(x+12, y)) // the box: a second click would close its list
		mouseAt(t, h, tea.MouseReleaseMsg{X: x + 12, Y: y, Button: tea.MouseLeft})
		if !h.m.ui.Models.Effort {
			t.Fatalf("a click on the effort box did not open its list:\n%s", frameText(h))
		}
		mouseAt(t, h, click(x+12, y))
		mouseAt(t, h, tea.MouseReleaseMsg{X: x + 12, Y: y, Button: tea.MouseLeft})
		if h.m.ui.Models.Effort {
			t.Fatal("a second click on the box left its list open")
		}
		mouseAt(t, h, click(x+12, y))
		mouseAt(t, h, tea.MouseReleaseMsg{X: x + 12, Y: y, Button: tea.MouseLeft})
		if !h.m.ui.Models.Effort {
			t.Fatalf("a click on the effort box did not open its list:\n%s", frameText(h))
		}
		clickText(t, h, "XHigh", 0)
		if got := s.modelChanges(); len(got) != 2 || got[1].Effort != "xhigh" || got[1].Model != "" {
			t.Errorf("changes %+v", got)
		}
		s.events <- event.Event{Provider: "claude-code", Kind: event.ModelSettings, Origin: event.Observed, Payload: event.Settings{Model: "claude-sonnet-5-5",
			Choice: "sonnet", Effort: "xhigh", EffortKnown: true, Ultracode: true, UltracodeAvailable: true, UltracodeKnown: true, Answers: true}}
		h.settle()
		clickText(t, h, render.UltracodeDescription, 0)
		if got := s.modelChanges(); len(got) != 3 || got[2].Ultracode == nil || *got[2].Ultracode || got[2].Effort != "" {
			t.Errorf("changes %+v", got)
		}
		clickText(t, h, "Ask ", 0) // the draft: outside the selector
		if h.m.ui.Models.Open || h.m.ui.Focus != "B" {
			t.Errorf("a click outside: open %v, focus %q", h.m.ui.Models.Open, h.m.ui.Focus)
		}
	})
}

// TestSelectorWithoutModels: a session that lists no models, or has not
// yet, opens no selector; a toast says why.
func TestSelectorWithoutModels(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, cli := startScript(t, "claude-code")
		cli.sess.events <- event.Event{Provider: "claude-code", Kind: event.SessionSpawned, Origin: event.Observed, Payload: event.Spawned{}}
		cli.sess.events <- event.Event{Provider: "claude-code", Kind: event.SessionReady, Origin: event.Observed, Payload: event.Ready{}}
		h.settle()
		h.send(key('p', tea.ModAlt))
		if h.m.ui.Models.Open || h.m.ui.Toast != "no model selection with this session" {
			t.Errorf("open %v, toast %q", h.m.ui.Models.Open, h.m.ui.Toast)
		}
		cli.sess.models = true
		h.send(key('p', tea.ModAlt))
		if h.m.ui.Models.Open || h.m.ui.Toast != "the session has not listed its models yet" {
			t.Errorf("open %v, toast %q", h.m.ui.Models.Open, h.m.ui.Toast)
		}
		// A list the CLI refused: the toast gives its words (plan 34's review).
		cli.sess.events <- event.Event{Provider: "claude-code", Kind: event.ModelsListed, Origin: event.Observed,
			Payload: event.Models{Source: "model/list", Problem: "model/list refused by Codex: boom"}}
		h.settle()
		h.send(key('p', tea.ModAlt))
		if h.m.ui.Models.Open || h.m.ui.Toast != "no model list: model/list refused by Codex: boom" {
			t.Errorf("open %v, toast %q", h.m.ui.Models.Open, h.m.ui.Toast)
		}
	})
}

// TestSelectorNoRoom is plan 34's review: Alt+P on a screen without room
// for the selector opens nothing and says so; a selector a later frame has
// no room for closes, so no key acts on it unseen; a paste closes it and
// goes on to the draft.
func TestSelectorNoRoom(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, s := selectorRig(t)
		h.send(tea.WindowSizeMsg{Width: 80, Height: 24})
		h.send(tea.PasteMsg{Content: strings.Repeat("line\n", 9) + "line"}) // ten lines
		_ = h.frame()
		h.send(key('p', tea.ModAlt))
		if h.m.ui.Models.Open || h.m.ui.Toast != "no room for the model selector above the editor" {
			t.Fatalf("Alt+P with a ten-line draft at 80x24: open %v, toast %q\n%s", h.m.ui.Models.Open, h.m.ui.Toast, h.frame())
		}
		h.send(tea.WindowSizeMsg{Width: 120, Height: 40})
		_ = h.frame()
		h.send(key('p', tea.ModAlt))
		if !h.m.ui.Models.Open {
			t.Fatalf("Alt+P at 120x40: toast %q\n%s", h.m.ui.Toast, h.frame())
		}
		h.send(tea.WindowSizeMsg{Width: 80, Height: 24}) // the frame now has no room for it
		if f := h.frame(); h.m.ui.Models.Open || strings.Contains(f, "CLAUDE MODELS") {
			t.Errorf("the selector stayed open, or drawn, with no room:\n%s", f)
		}
		h.send(key(tea.KeyUp, 0))
		h.send(key(tea.KeyEnter, tea.ModAlt)) // a newline in the draft, the selector closed
		if got := s.modelChanges(); len(got) != 0 {
			t.Errorf("changes %+v", got)
		}
		h.send(tea.WindowSizeMsg{Width: 120, Height: 40})
		_ = h.frame()
		h.send(key('p', tea.ModAlt))
		h.send(tea.PasteMsg{Content: "pasted"})
		if h.m.ui.Models.Open || !strings.Contains(h.m.edit.Text(), "pasted") {
			t.Errorf("a paste: open %v, draft %q", h.m.ui.Models.Open, h.m.edit.Text())
		}
		if sent, _, _ := s.calls(); len(sent) != 0 {
			t.Errorf("sent %q", sent)
		}
	})
}

// TestSelectorSendKey is plan 34's review: a send key the configuration
// names acts in the selector as Enter does, and sends nothing.
func TestSelectorSendKey(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, s := selectorRig(t)
		h.m.opt.Keys = input.Keys{Send: []string{"ctrl+enter"}, Newline: []string{"enter"}}.WithDefaults()
		h.send(tea.PasteMsg{Content: "a draft"})
		h.send(key('p', tea.ModAlt))
		h.send(key(tea.KeyDown, 0)) // Sonnet 5.5
		h.send(key(tea.KeyEnter, tea.ModCtrl))
		if got := s.modelChanges(); len(got) != 1 || got[0].Model != "sonnet" {
			t.Errorf("changes %+v", got)
		}
		if sent, _, _ := s.calls(); len(sent) != 0 || h.m.edit.Text() != "a draft" {
			t.Errorf("sent %q, draft %q", sent, h.m.edit.Text())
		}
	})
}

// selectorBox is the open selector's size in cells, from its top edge to
// its bottom edge, and the row of its header.
func selectorBox(t *testing.T, h *harness) (w, rows, header int) {
	t.Helper()
	x, y := at(t, h, "CLAUDE MODELS", 0)
	xr, _ := at(t, h, "Current session", 0)
	_, yb := at(t, h, "╰▾", 0)
	return xr + len("Current session") + 2 - (x - 2), yb - y + 2, y
}

// TestSelectorScroll is criteria 4, 5 and the rest of 14 on P34c's 13
// models: Space acts as Enter; More models expanded leaves the selector's
// size as it was and puts its row at the top of the list; PgUp, PgDn, the
// wheel and the cursor scroll the list; a model in force under More models
// opens the selector expanded on it.
func TestSelectorScroll(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newHarness(t, Options{Config: Config{Mock: filepath.Join("..", "..", "testdata", "mock", "claude-models.jsonl")}})
		h.settle()
		h.send(tea.WindowSizeMsg{Width: 120, Height: 40})
		h.send(key('p', tea.ModAlt))
		w0, r0, y0 := selectorBox(t, h)
		if r0 != 25 {
			t.Fatalf("the selector is %d rows, want 25:\n%s", r0, frameText(h))
		}
		if strings.Contains(frameText(h), "GPT") {
			t.Errorf("a Codex model in Claude Code's selector:\n%s", frameText(h))
		}
		top := func() int {
			t.Helper()
			_ = h.m.View()
			if !h.m.screen.Selector.Shown {
				t.Fatal("the selector is not on screen")
			}
			return h.m.screen.Selector.Top
		}
		for range 3 {
			h.send(key(tea.KeyDown, 0)) // Sonnet 5.5, Haiku 5.5, More models
		}
		h.send(key(tea.KeySpace, 0))
		if !h.m.ui.Models.More {
			t.Fatal("Space on More models did not expand it")
		}
		if w, r, _ := selectorBox(t, h); w != w0 || r != r0 {
			t.Errorf("expanded: %dx%d, want %dx%d as collapsed", w, r, w0, r0)
		}
		if _, y := at(t, h, "▼ More models", 0); y != y0+2 || top() != 15 {
			t.Errorf("More models on row %d (the list's first is %d), the list from row %d", y, y0+2, top())
		}
		// The list holds 41 rows (five models, More models and eight models
		// between its two separators), 17 on screen.
		for _, c := range []struct {
			what string
			msg  tea.Msg
			want int
		}{{"PgUp", key(tea.KeyPgUp, 0), 0}, {"PgDn", key(tea.KeyPgDown, 0), 17}, {"PgDn again", key(tea.KeyPgDown, 0), 24},
			{"PgDn at the end", key(tea.KeyPgDown, 0), 24}} {
			h.send(c.msg)
			if got := top(); got != c.want {
				t.Errorf("%s: the list from row %d, want %d", c.what, got, c.want)
			}
		}
		if !strings.Contains(frameText(h), "Sonnet 4.6") {
			t.Errorf("the list's end not shown:\n%s", frameText(h))
		}
		x, y := at(t, h, "Opus 4.6", 0)
		mouseAt(t, h, tea.MouseWheelMsg{X: x, Y: y, Button: tea.MouseWheelUp})
		if got := top(); got != 24-wheelRows {
			t.Errorf("the wheel: the list from row %d, want %d", got, 24-wheelRows)
		}
		h.send(key(tea.KeyHome, 0))
		if got := top(); got != 0 || h.m.ui.Models.Cursor != 0 {
			t.Errorf("Home: cursor %d, the list from row %d", h.m.ui.Models.Cursor, got)
		}
		for range 13 {
			h.send(key(tea.KeyDown, 0)) // down to Sonnet 4.6, the last model
		}
		if got := top(); got != 23 || !strings.Contains(frameText(h), "▶ Sonnet 4.6") {
			t.Errorf("the cursor on Sonnet 4.6: the list from row %d, want 23:\n%s", got, frameText(h))
		}

		// Opus 4.6 in force, under More models: the selector opens expanded,
		// its cursor and check on it.
		r, s := selectorRig(t)
		s.events <- event.Event{Provider: "claude-code", Kind: event.ModelSettings, Origin: event.Observed, Payload: event.Settings{Model: "claude-opus-4-6",
			Choice: "claude-opus-4-6", Effort: "high", EffortKnown: true, UltracodeKnown: true}}
		r.settle()
		r.send(key('p', tea.ModAlt))
		v := r.m.ui.Models
		if !v.Open || !v.More {
			t.Fatalf("Opus 4.6 in force: open %v, More models expanded %v", v.Open, v.More)
		}
		line := ""
		for _, l := range strings.Split(frameText(r), "\n") {
			if strings.Contains(l, "▶ Opus 4.6") {
				line = l
			}
		}
		if line == "" || !strings.Contains(line, "✓") {
			t.Errorf("Opus 4.6's row %q:\n%s", line, frameText(r))
		}
	})
}

// TestSelectorKeepsDraft is criterion 16 with images: a draft of text and
// two attached images, the cursor inside it; the selector opened by Alt+P
// and by the pill, a model chosen, the effort list opened and closed, More
// models expanded, the selector closed by Esc and by a click outside: the
// text, the cursor, the images and the focus as they were, nothing sent.
func TestSelectorKeepsDraft(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		files, err := imagetest.WriteAll(t.TempDir())
		if err != nil {
			t.Fatal(err)
		}
		h, s := selectorRig(t)
		p := &picker{answers: [][]string{{files["png"], files["jpg"]}}}
		h.m.opt.PickImages = p.pick
		h.send(tea.PasteMsg{Content: "a draft\nkept"})
		h.send(key(tea.KeyHome, 0))
		upload(t, h)
		text, cursor := h.m.edit.Text(), h.m.edit.Cursor()
		names := func() []string {
			var out []string
			for _, a := range h.m.attach {
				out = append(out, a.Name)
			}
			return out
		}
		images := names()
		if len(images) != 2 {
			t.Fatalf("attached %q", images)
		}
		h.send(key(tea.KeyF6, tea.ModShift)) // the focus on zone A: the selector keeps it (design D10)
		if h.m.ui.Focus != "A" {
			t.Fatalf("focus %q after Shift+F6", h.m.ui.Focus)
		}
		h.send(key('p', tea.ModAlt))
		h.send(key(tea.KeyDown, 0))
		h.send(key(tea.KeyEnter, 0)) // Sonnet 5.5
		for range 3 {
			h.send(key(tea.KeyDown, 0)) // Haiku 5.5, More models, Effort
		}
		h.send(key(tea.KeyEnter, 0))
		h.send(key(tea.KeyEsc, 0)) // the effort list
		h.send(key(tea.KeyUp, 0))
		h.send(key(tea.KeySpace, 0)) // More models expanded
		h.send(key(tea.KeyEsc, 0))
		if h.m.ui.Models.Open || h.m.ui.Focus != "A" {
			t.Errorf("after Esc: open %v, focus %q, want closed and A", h.m.ui.Models.Open, h.m.ui.Focus)
		}
		clickText(t, h, "Fable 5.1 High", 0)
		if !h.m.ui.Models.Open || h.m.ui.Focus != "A" {
			t.Fatalf("the pill's click: open %v, focus %q, want open and A", h.m.ui.Models.Open, h.m.ui.Focus)
		}
		x, y := at(t, h, "Opus 5.5", 1) // the second: the entry's name, not Default's description
		mouseAt(t, h, tea.MouseMotionMsg{X: x, Y: y})
		clickText(t, h, "CONVERSATION", 0) // outside, on zone B's title: closes it and focuses B, as a click there does
		if h.m.ui.Models.Open {
			t.Error("a click outside left the selector open")
		}
		if got := s.modelChanges(); len(got) != 1 || got[0].Model != "sonnet" {
			t.Errorf("changes %+v", got)
		}
		if sent, _, _ := s.calls(); len(sent) != 0 || len(s.messages()) != 0 {
			t.Errorf("sent %q", sent)
		}
		if h.m.edit.Text() != text || h.m.edit.Cursor() != cursor || !reflect.DeepEqual(names(), images) || h.m.ui.Focus != "B" {
			t.Errorf("draft %q at %d, images %q, focus %q; want %q at %d, %q, B", h.m.edit.Text(), h.m.edit.Cursor(), names(), h.m.ui.Focus, text, cursor, images)
		}
	})
}

// TestSelectorPillHidden is plan 34's second review: while a turn's status
// too long to sit beside the pill has the editor row alone (plan 29, design
// D1), the pill is not on screen and the selector says it needs it.
func TestSelectorPillHidden(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, s := selectorRig(t)
		h.send(tea.PasteMsg{Content: "hello"})
		h.send(key(tea.KeyEnter, 0))
		s.events <- event.Event{Provider: "claude-code", Kind: event.TurnStarted, Origin: event.Observed}
		h.settle()
		if f := h.frame(); strings.Contains(f, "Fable 5.1 High") || !strings.Contains(f, "turn running") {
			t.Fatalf("the editor row during the turn:\n%s", f)
		}
		h.send(key('p', tea.ModAlt))
		if h.m.ui.Models.Open || h.m.ui.Toast != "the model selector needs the model pill on screen" {
			t.Errorf("open %v, toast %q", h.m.ui.Models.Open, h.m.ui.Toast)
		}
	})
}
