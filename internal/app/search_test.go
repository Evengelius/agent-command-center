package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
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

// searchTurn is one turn pushed to the scripted CLI: the owner's prompt,
// calls each reading a file, and an answer.
func searchTurn(h *harness, id, prompt string, files []string, answer string) {
	evs := []event.Event{evOf(event.UserMessage, event.User{Text: prompt}), evOf(event.TurnStarted, nil)}
	for k, f := range files {
		tid := fmt.Sprintf("%s-t%d", id, k)
		in, _ := json.Marshal(map[string]string{"file_path": f})
		evs = append(evs, evOf(event.ToolStarted, event.Tool{ToolID: tid, Name: "Read", Input: in}),
			evOf(event.ToolCompleted, event.Tool{ToolID: tid, Name: "Read", Output: "ok"}))
	}
	evs = append(evs, evOf(event.MessageCompleted, event.Completed{MessageID: id, Text: answer}),
		evOf(event.TurnCompleted, event.TurnEnd{Outcome: "completed", Subtype: "success"}))
	pushEvents(h, claude.ProviderID, evs...)
}

// searchRig is a ready Claude Code dashboard at 120x40 holding three turns
// in which "needle" appears in a prompt, in a call's target and in answers,
// the conversation longer than zone B, a frame drawn.
func searchRig(t *testing.T) (*harness, *scriptCLI) {
	t.Helper()
	h, cli := readyScript(t, claude.ProviderID, false, event.Started{})
	var long strings.Builder
	for i := range 40 {
		fmt.Fprintf(&long, "line %d of a long answer\n\n", i+1)
	}
	searchTurn(h, "m1", "find the needle in the code", []string{"a.go", "needle.go"}, "The needle is in needle.go.\n\n"+long.String())
	searchTurn(h, "m2", "and now the hay", []string{"hay.go"}, "Only hay here.\n\n"+long.String())
	searchTurn(h, "m3", "last one", nil, "One more needle at the end.")
	h.m.View()
	return h, cli
}

// open sends Ctrl+F and draws.
func openBar(h *harness) { h.draw(key('f', tea.ModCtrl)) }

// typeQuery types q into the bar, a frame drawn after each key.
func typeQuery(h *harness, q string) {
	for _, k := range typedText(q) {
		h.draw(k)
	}
}

// current is the bar's current match.
func current(t *testing.T, h *harness) render.ConvMatch {
	t.Helper()
	v := h.m.ui.Search
	if v.Cur < 0 || v.Cur >= len(v.Matches) {
		t.Fatalf("no current match: %+v", v)
	}
	return v.Matches[v.Cur]
}

// matchText is the text of the current match's part.
func matchText(h *harness, m render.ConvMatch) string {
	e := h.m.State().Conversation[m.Item]
	if m.Target {
		return "the target of " + e.Text
	}
	return strings.Split(e.Text, "\n")[m.Line]
}

// TestSearchOpen is criterion 66.3's openings (design D2): Ctrl+F from the
// input, zone A, zone C, and from the diff view in zone B's place, which
// closes; no bar inside the draft window, whose Ctrl+F stays its full
// screen, nor over the key overlay or a full view.
func TestSearchOpen(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, _ := searchRig(t)
		for _, focus := range []string{"B", "A", "C"} {
			h.m.ui.Focus = focus
			openBar(h)
			if !h.m.ui.Search.Open || h.m.ui.Focus != "B" || lineWith(h, "type to search the conversation") == "" {
				t.Fatalf("Ctrl+F from %s: open %v, focus %q\n%s", focus, h.m.ui.Search.Open, h.m.ui.Focus, h.frame())
			}
			h.draw(key(tea.KeyEscape, 0))
			if h.m.ui.Search.Open || h.m.ui.Focus != focus {
				t.Fatalf("Esc from the bar opened in %s: open %v, focus %q", focus, h.m.ui.Search.Open, h.m.ui.Focus)
			}
		}
		h.m.ui.Diff, h.m.ui.Focus = render.DiffView{Open: true, Path: "a.go"}, "D"
		openBar(h)
		if !h.m.ui.Search.Open || h.m.ui.Diff.Open {
			t.Fatalf("Ctrl+F from the diff: bar %v, diff %v", h.m.ui.Search.Open, h.m.ui.Diff.Open)
		}
		h.draw(key(tea.KeyEscape, 0))
		if h.m.ui.Focus != "B" {
			t.Fatalf("Esc after the diff closed: focus %q, want the input's", h.m.ui.Focus)
		}
		h.draw(key('g', tea.ModCtrl)) // the draft window
		full := h.m.ui.DraftWin.Full
		openBar(h)
		if h.m.ui.Search.Open || h.m.ui.DraftWin.Full == full {
			t.Fatalf("Ctrl+F in the draft window: bar %v, full screen %v then %v", h.m.ui.Search.Open, full, h.m.ui.DraftWin.Full)
		}
		h.draw(key(tea.KeyEscape, 0))
		h.draw(key(tea.KeyF1, 0)) // the key overlay
		openBar(h)
		if h.m.ui.Search.Open {
			t.Fatal("Ctrl+F over the key overlay opened the bar")
		}
		h.draw(key(tea.KeyEscape, 0))
		h.m.openView("", 1) // a full view
		openBar(h)
		if h.m.ui.Search.Open {
			t.Fatal("Ctrl+F over a full view opened the bar")
		}
	})
}

// TestSearchField is criterion 66.3's field and 66.8's draft: typing,
// Backspace, ← →, Home, End and a paste edit the query; the draft, its
// cursor and its selection are byte for byte the same from Ctrl+F to Esc,
// and nothing reaches the CLI.
func TestSearchField(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, cli := searchRig(t)
		h.m.edit.SetText("my draft, unsent")
		h.m.edit.Select(3, 8)
		draft, cur := h.m.edit.Text(), h.m.edit.Cursor()
		from, to, _ := h.m.edit.Selection()
		sent, _, _ := cli.sess.calls()
		openBar(h)
		typeQuery(h, "needlx")
		h.draw(key(tea.KeyBackspace, 0))
		typeQuery(h, "e")
		h.draw(key(tea.KeyLeft, 0))
		h.draw(key(tea.KeyLeft, 0))
		typeQuery(h, "-")
		h.draw(key(tea.KeyHome, 0))
		typeQuery(h, ">")
		h.draw(key(tea.KeyEnd, 0))
		h.send(tea.PasteMsg{Content: "!\nsecond line"})
		h.m.View()
		// AltGr's characters as Windows Terminal sends them, with left Ctrl
		// and right Alt (AltGr+2 on the owner's keyboard); Ctrl or Alt alone
		// types nothing, whatever text the key carries.
		h.draw(tea.KeyPressMsg{Code: '@', Text: "@", Mod: tea.ModCtrl | tea.ModAlt})
		h.draw(tea.KeyPressMsg{Code: 'k', Text: "k", Mod: tea.ModAlt})
		h.draw(tea.KeyPressMsg{Code: 'k', Text: "k", Mod: tea.ModCtrl})
		if q := h.m.ui.Search.Query; q != ">need-le!@" {
			t.Fatalf("the query %q, want %q", q, ">need-le!@")
		}
		h.draw(key(tea.KeyEscape, 0))
		f2, t2, _ := h.m.edit.Selection()
		if h.m.edit.Text() != draft || h.m.edit.Cursor() != cur || f2 != from || t2 != to {
			t.Errorf("the draft %q at %d, %d..%d; want %q at %d, %d..%d", h.m.edit.Text(), h.m.edit.Cursor(), f2, t2, draft, cur, from, to)
		}
		if now, _, _ := cli.sess.calls(); len(now) != len(sent) {
			t.Errorf("the CLI was sent %q during the search", now[len(sent):])
		}
	})
}

// TestSearchSteps is criterion 66.4: the first match the newest at or above
// zone B's last row shown, ↑ and Enter older, ↓ and Shift+Enter newer, the
// number following, round at either end; zone B scrolled to each and held.
func TestSearchSteps(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, _ := searchRig(t)
		openBar(h)
		typeQuery(h, "needle")
		v := h.m.ui.Search
		n := len(v.Matches)
		if n < 4 || v.Cur != 0 || !strings.Contains(lineWith(h, "match 1 of"), fmt.Sprintf("match 1 of %d", n)) {
			t.Fatalf("%d matches, current %d, the bar %q", n, v.Cur, lineWith(h, "match 1 of"))
		}
		if !strings.Contains(matchText(h, current(t, h)), "One more needle") {
			t.Errorf("the first match is %q, want the newest", matchText(h, current(t, h)))
		}
		for i, k := range []tea.KeyPressMsg{key(tea.KeyUp, 0), key(tea.KeyEnter, 0)} {
			h.draw(k)
			if h.m.ui.Search.Cur != i+1 || !strings.Contains(lineWith(h, "match"), fmt.Sprintf("match %d of %d", i+2, n)) {
				t.Fatalf("after %d older: current %d, the bar %q", i+1, h.m.ui.Search.Cur, lineWith(h, "match"))
			}
			if !h.m.ui.ConvHold {
				t.Fatal("zone B not held at a match")
			}
		}
		h.draw(key(tea.KeyDown, 0))
		h.draw(key(tea.KeyEnter, tea.ModShift))
		if h.m.ui.Search.Cur != 0 {
			t.Fatalf("↓ then Shift+Enter: current %d, want 0", h.m.ui.Search.Cur)
		}
		h.draw(key(tea.KeyDown, 0))
		if h.m.ui.Search.Cur != n-1 {
			t.Fatalf("↓ from the newest: current %d, want the oldest, %d", h.m.ui.Search.Cur, n-1)
		}
		// The oldest, the prompt of turn 1, in view.
		if !strings.Contains(matchText(h, current(t, h)), "find the needle") || lineWith(h, "find the needle in the code") == "" {
			t.Errorf("the oldest match %q not in view:\n%s", matchText(h, current(t, h)), h.frame())
		}
		h.draw(key(tea.KeyUp, 0))
		if h.m.ui.Search.Cur != 0 {
			t.Errorf("↑ from the oldest: current %d, want the newest", h.m.ui.Search.Cur)
		}
	})
}

// TestSearchStartsInView is design D5's start: with zone B scrolled up, the
// first current match is the newest at or above its last row shown, not
// the conversation's newest.
func TestSearchStartsInView(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, _ := searchRig(t)
		for range 5 {
			h.draw(key(tea.KeyPgUp, tea.ModShift))
		}
		openBar(h)
		typeQuery(h, "needle")
		bottom := h.m.screen.Conv.Bottom
		m := current(t, h)
		if !m.AtOrBefore(bottom) {
			t.Fatalf("the first match %+v is past B's last row shown %+v", m, bottom)
		}
		for _, older := range h.m.ui.Search.Matches[:h.m.ui.Search.Cur] {
			if older.AtOrBefore(bottom) {
				t.Fatalf("a newer match %+v at or above the last row %+v was passed over", older, bottom)
			}
		}
	})
}

// TestSearchFolds is criterion 66.4's folds: a match in a run's folded call
// unfolds the run; one in a phase's folded details unfolds them; one in a
// long paste's folded lines shows its "… N more lines" row.
func TestSearchFolds(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, _ := readyScript(t, claude.ProviderID, false, event.Started{})
		files := []string{"pin.go", "b.go", "c.go", "d.go", "e.go", "f.go"}
		searchTurn(h, "m1", "read them", files, "Done.")
		var paste []string
		for i := range 30 {
			paste = append(paste, fmt.Sprintf("log %d", i))
		}
		paste[12] = "connection refused"
		searchTurn(h, "m2", strings.Join(paste, "\n"), nil, "Seen.")
		h.m.View()
		openBar(h)
		typeQuery(h, "pin.go")
		m := current(t, h)
		e := h.m.State().Conversation[m.Item]
		if !m.Target || !h.m.ui.Unfolded[e.MessageID] || lineWith(h, "pin.go") == "" {
			t.Fatalf("a folded call's match %+v: run unfolded %v\n%s", m, h.m.ui.Unfolded, h.frame())
		}
		for range 6 {
			h.draw(key(tea.KeyBackspace, 0))
		}
		typeQuery(h, "refused")
		m = current(t, h)
		if m.Line != 12 || lineWith(h, "more lines") == "" {
			t.Fatalf("a long paste's folded match %+v, its fold row not in view:\n%s", m, h.frame())
		}
	})
}

// TestSearchThinkingDetails is criterion 66.4's phase: a match in a
// thinking phase's folded details unfolds them.
func TestSearchThinkingDetails(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, _ := readyScript(t, claude.ProviderID, false, event.Started{})
		t0 := time.Date(2026, 10, 18, 9, 0, 0, 0, time.UTC)
		s := h.m.State()
		s.Conversation = append(s.Conversation, state.Entry{Role: "thinking", Thought: &state.Thought{ID: 7, Start: t0, End: t0.Add(2 * time.Second)}})
		h.m.View()
		openBar(h)
		typeQuery(h, "count")
		m := current(t, h)
		e := h.m.State().Conversation[m.Item]
		if m.Line == 0 || e.Role != "thinking" || !h.m.ui.Unfolded[render.ThoughtKey(e)] {
			t.Fatalf("a phase's details match %+v: unfolded %v", m, h.m.ui.Unfolded)
		}
	})
}

// TestSearchStreaming is criterion 66.5: with an older match current, an
// answer streamed with the bar open adds its matches, the count and the
// current match's number growing, the current match and zone B's view not
// moving.
func TestSearchStreaming(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, _ := searchRig(t)
		openBar(h)
		typeQuery(h, "needle")
		h.draw(key(tea.KeyUp, 0))
		h.draw(key(tea.KeyUp, 0))
		was, n, top := current(t, h), len(h.m.ui.Search.Matches), h.m.ui.ConvTop
		cur := h.m.ui.Search.Cur
		pushEvents(h, claude.ProviderID, evOf(event.UserMessage, event.User{Text: "a needle more"}), evOf(event.TurnStarted, nil),
			evOf(event.MessageDelta, event.Delta{MessageID: "m4", Block: "text", Text: "needle and needle\n"}))
		h.m.View()
		v := h.m.ui.Search
		if len(v.Matches) != n+3 || v.Cur != cur+3 || current(t, h) != was || h.m.ui.ConvTop != top {
			t.Fatalf("after the stream: %d matches (were %d), current %d (was %d) %+v (was %+v), top %d (was %d)",
				len(v.Matches), n, v.Cur, cur, current(t, h), was, h.m.ui.ConvTop, top)
		}
		if !strings.Contains(lineWith(h, "match"), fmt.Sprintf("match %d of %d", cur+4, n+3)) {
			t.Errorf("the bar %q", lineWith(h, "match"))
		}
	})
}

// TestSearchClose is criterion 66.8: the keys that leave the conversation
// close the bar and do what they do; F1 opens the key overlay over it, the
// bar there once the overlay closes; a click on zone C closes it, the focus
// C's; zone B stays where the search took it.
func TestSearchClose(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, _ := searchRig(t)
		openBar(h)
		typeQuery(h, "needle")
		h.draw(key(tea.KeyUp, 0))
		top := h.m.ui.ConvTop
		h.draw(key(tea.KeyEscape, 0))
		if h.m.ui.Search.Open || !h.m.ui.ConvHold || h.m.ui.ConvTop != top {
			t.Fatalf("Esc: open %v, held %v at %d (the search left it at %d)", h.m.ui.Search.Open, h.m.ui.ConvHold, h.m.ui.ConvTop, top)
		}
		openBar(h)
		h.draw(key(tea.KeyF1, 0))
		if !h.m.ui.Keys || !h.m.ui.Search.Open {
			t.Fatalf("F1: overlay %v, bar %v", h.m.ui.Keys, h.m.ui.Search.Open)
		}
		h.draw(key(tea.KeyEscape, 0))
		if h.m.ui.Keys || !h.m.ui.Search.Open || lineWith(h, "type to search the conversation") == "" {
			t.Fatalf("Esc from the overlay: overlay %v, bar %v", h.m.ui.Keys, h.m.ui.Search.Open)
		}
		h.draw(key(tea.KeyF6, 0)) // the next zone: the bar closed first, the key on
		if h.m.ui.Search.Open || h.m.ui.Focus == "B" {
			t.Fatalf("F6: bar %v, focus %q", h.m.ui.Search.Open, h.m.ui.Focus)
		}
		h.m.ui.Focus = "B"
		openBar(h)
		x, y := cellOf(t, h, 'C')
		mouseAt(t, h, tea.MouseClickMsg{X: x, Y: y, Button: tea.MouseLeft})
		if h.m.ui.Search.Open || h.m.ui.Focus != "C" {
			t.Fatalf("a click on zone C: bar %v, focus %q", h.m.ui.Search.Open, h.m.ui.Focus)
		}
		// A click that opens a popup, the "+" menu, closes the bar too; the
		// menu then on top takes the keys.
		h.m.ui.Focus = "B"
		openBar(h)
		x, y = cellOf(t, h, '+')
		mouseAt(t, h, tea.MouseClickMsg{X: x, Y: y, Button: tea.MouseLeft})
		if top := render.Layers(h.m.State(), h.m.ui).Top; h.m.ui.Search.Open || !h.m.ui.Plus || top != render.LayerPlus {
			t.Fatalf("a click on +: bar %v, the menu %v, the top layer %v", h.m.ui.Search.Open, h.m.ui.Plus, top)
		}
	})
}

// TestSearchClosesPopups is design D2's popups: Ctrl+F over Source
// Control's commit dropdown, which takes no key of its own, closes it, as
// every popup is closed before the bar opens; the keys then reach the
// bar, never the draft.
func TestSearchClosesPopups(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, _ := commitRig(t)
		clickItem(t, h, render.ItemCommitMore, 0, 0)
		if !h.m.ui.Source.Menu {
			t.Fatal("the chevron opened no dropdown")
		}
		draft := h.m.edit.Text()
		openBar(h)
		typeQuery(h, "abc")
		if top := render.Layers(h.m.State(), h.m.ui).Top; h.m.ui.Source.Menu || !h.m.ui.Search.Open || top != render.LayerSearch {
			t.Fatalf("Ctrl+F over the dropdown: dropdown %v, bar %v, top layer %v", h.m.ui.Source.Menu, h.m.ui.Search.Open, top)
		}
		if h.m.ui.Search.Query != "abc" || h.m.edit.Text() != draft {
			t.Errorf("typed: the query %q, the draft %q (was %q)", h.m.ui.Search.Query, h.m.edit.Text(), draft)
		}
	})
}

// TestSearchWhole is criterion 66.7: Tab on a match in a long paste's
// folded lines opens the full view with the match's row in view, its own
// keys working there, the bar kept; Esc back to zone B and the bar, the
// current match the same; Tab on a call's target opens the call's full
// view at its top; Tab on an answer B shows whole opens nothing.
func TestSearchWhole(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, _ := readyScript(t, claude.ProviderID, false, event.Started{})
		searchTurn(h, "m0", "read the log", []string{"refused.log"}, "Read.")
		var paste []string
		for i := range 300 {
			paste = append(paste, fmt.Sprintf("log %d", i))
		}
		paste[250] = "connection refused"
		searchTurn(h, "m1", strings.Join(paste, "\n"), nil, "Seen: connection refused.")
		h.m.View()
		openBar(h)
		typeQuery(h, "refused")
		if m := current(t, h); h.m.State().Conversation[m.Item].Role != "assistant" {
			t.Fatalf("the first match %+v, want the answer's", m)
		}
		h.draw(key(tea.KeyTab, 0))
		if h.m.ui.ViewEntry != 0 {
			t.Fatalf("Tab on an answer opened entry %d", h.m.ui.ViewEntry)
		}
		h.draw(key(tea.KeyUp, 0))
		m := current(t, h)
		h.draw(key(tea.KeyTab, 0))
		if h.m.ui.ViewEntry != m.Item+1 || lineWith(h, "connection refused") == "" {
			t.Fatalf("Tab on the paste's match: entry %d, want %d\n%s", h.m.ui.ViewEntry, m.Item+1, h.frame())
		}
		if h.m.ui.InspectTop == 0 {
			t.Fatal("the full view at its top, not at the match's row")
		}
		h.draw(key(tea.KeyHome, 0)) // the view's own key
		if h.m.ui.ViewEntry != m.Item+1 || h.m.ui.InspectTop != 0 || !h.m.ui.Search.Open {
			t.Fatalf("Home in the full view: entry %d, top %d, bar %v", h.m.ui.ViewEntry, h.m.ui.InspectTop, h.m.ui.Search.Open)
		}
		h.draw(key(tea.KeyEscape, 0))
		if h.m.ui.ViewEntry != 0 || !h.m.ui.Search.Open || current(t, h) != m {
			t.Fatalf("Esc from the full view: entry %d, bar %v, current %+v (was %+v)", h.m.ui.ViewEntry, h.m.ui.Search.Open, current(t, h), m)
		}
		h.draw(key(tea.KeyUp, 0))
		call := current(t, h)
		if !call.Target || h.m.State().Conversation[call.Item].Role != "tool" {
			t.Fatalf("the oldest match %+v, want the Read's target", call)
		}
		h.draw(key(tea.KeyTab, 0))
		if h.m.ui.ViewEntry != call.Item+1 || h.m.ui.InspectTop != 0 {
			t.Fatalf("Tab on a call's target: entry %d (want %d), top %d", h.m.ui.ViewEntry, call.Item+1, h.m.ui.InspectTop)
		}
		h.draw(key(tea.KeyEscape, 0))
		if h.m.ui.ViewEntry != 0 || !h.m.ui.Search.Open {
			t.Fatalf("Esc from the call's view: entry %d, bar %v", h.m.ui.ViewEntry, h.m.ui.Search.Open)
		}
	})
}

// TestSearchEntryCursor is design D9's B's entries: Ctrl+F from them takes
// their cursor off while the bar is open, and Esc gives it back on its
// entry.
func TestSearchEntryCursor(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, _ := searchRig(t)
		h.draw(key(tea.KeyUp, 0)) // from the empty input, B's entries
		entry := h.m.ui.Entry
		if entry == 0 {
			t.Fatal("↑ from the empty input put no entry cursor")
		}
		openBar(h)
		if h.m.ui.Entry != 0 || !h.m.ui.Search.Open {
			t.Fatalf("Ctrl+F from B's entries: entry %d, bar %v", h.m.ui.Entry, h.m.ui.Search.Open)
		}
		typeQuery(h, "needle")
		h.draw(key(tea.KeyUp, 0))
		h.draw(key(tea.KeyEscape, 0))
		if h.m.ui.Search.Open || h.m.ui.Focus != "B" || h.m.ui.Entry != entry {
			t.Fatalf("Esc: bar %v, focus %q, entry %d; want the entry cursor back on %d", h.m.ui.Search.Open, h.m.ui.Focus, h.m.ui.Entry, entry)
		}
	})
}

// TestSearchDockedPanel is criteria 66.3 and 66.8 for a docked panel: at
// 200x60, the Explorer beside zone B, Ctrl+F from it opens the bar, the
// panel kept, and Esc gives the panel its focus back; over the quick open
// and the file viewer Ctrl+F opens no bar; at 120x40, where the panel takes
// zone B's place, Ctrl+F hides it as zone B's focus does, and Esc gives the
// focus to the input.
func TestSearchDockedPanel(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, _ := explorerRig(t, &fakeGit{})
		h.send(tea.WindowSizeMsg{Width: 200, Height: 60})
		h.send(key(tea.KeyF3, 0))
		if !h.m.ui.Explorer.Open || h.m.ui.Focus != "R" || render.PanelCompact(h.m.State(), h.m.ui) {
			t.Fatalf("F3 at 200x60: Explorer %v, focus %q, in B's place %v", h.m.ui.Explorer.Open, h.m.ui.Focus, render.PanelCompact(h.m.State(), h.m.ui))
		}
		openBar(h)
		if !h.m.ui.Search.Open || h.m.ui.Focus != "B" || !h.m.ui.Explorer.Open {
			t.Fatalf("Ctrl+F from the panel: bar %v, focus %q, Explorer %v", h.m.ui.Search.Open, h.m.ui.Focus, h.m.ui.Explorer.Open)
		}
		h.draw(key(tea.KeyEscape, 0))
		if h.m.ui.Search.Open || h.m.ui.Focus != "R" {
			t.Fatalf("Esc: bar %v, focus %q, want the panel's", h.m.ui.Search.Open, h.m.ui.Focus)
		}
		h.send(typed('/')) // the quick open
		openBar(h)
		if h.m.ui.Search.Open || !h.m.ui.Finder.Open {
			t.Fatalf("Ctrl+F in the quick open: bar %v, quick open %v", h.m.ui.Search.Open, h.m.ui.Finder.Open)
		}
		h.draw(key(tea.KeyEscape, 0))
		openPath(h, "main.go") // the file viewer
		openBar(h)
		if h.m.ui.Search.Open || !h.m.ui.File.Open {
			t.Fatalf("Ctrl+F in the file viewer: bar %v, viewer %v", h.m.ui.Search.Open, h.m.ui.File.Open)
		}
		h.draw(key(tea.KeyEscape, 0)) // back to the Explorer, on the file
		h.send(tea.WindowSizeMsg{Width: 120, Height: 40})
		if !h.m.ui.Explorer.Open || h.m.ui.Focus != "R" || !render.PanelCompact(h.m.State(), h.m.ui) {
			t.Fatalf("at 120x40: Explorer %v, focus %q, in B's place %v", h.m.ui.Explorer.Open, h.m.ui.Focus, render.PanelCompact(h.m.State(), h.m.ui))
		}
		openBar(h)
		if !h.m.ui.Search.Open || h.m.ui.Explorer.Open || h.m.ui.Focus != "B" {
			t.Fatalf("Ctrl+F from the panel in B's place: bar %v, Explorer %v, focus %q", h.m.ui.Search.Open, h.m.ui.Explorer.Open, h.m.ui.Focus)
		}
		h.draw(key(tea.KeyEscape, 0))
		if h.m.ui.Focus != "B" {
			t.Fatalf("Esc after the panel gave way: focus %q, want the input's", h.m.ui.Focus)
		}
	})
}

// TestSearchReveal is criterion 66.4's scroll: B scrolls the least that
// shows the current match's row, a match on a wrapped line's later row
// above the view brought to its top, the newest, below it again, to its
// last row.
func TestSearchReveal(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, _ := readyScript(t, claude.ProviderID, false, event.Started{})
		var words []string
		for i := range 40 {
			words = append(words, fmt.Sprintf("word%02d", i))
		}
		var after strings.Builder
		for i := range 40 {
			fmt.Fprintf(&after, "line %d after it\n\n", i+1)
		}
		searchTurn(h, "m1", "first", nil, strings.Join(words, " ")+" needle\n\n"+after.String())
		searchTurn(h, "m2", "second", nil, "The newest needle.")
		h.m.View()
		openBar(h)
		typeQuery(h, "needle")
		h.draw(key(tea.KeyUp, 0)) // the long line's, far above the view
		m, c := current(t, h), h.m.screen.Conv
		row, _, ok := render.ConvMatchRow(h.m.State(), h.m.ui, c.W, c.Room, m)
		first, _, _ := render.ConvMatchRow(h.m.State(), h.m.ui, c.W, c.Room, render.ConvMatch{Item: m.Item, Line: m.Line, N: 1})
		if !ok || row-first < 2 {
			t.Fatalf("the match's row %d, its line's first %d (ok %v): want its third row or later", row, first, ok)
		}
		if h.m.ui.ConvTop != row || !h.m.ui.ConvHold {
			t.Fatalf("B's top %d (held %v), want the match's row %d: the least scroll up", h.m.ui.ConvTop, h.m.ui.ConvHold, row)
		}
		h.draw(key(tea.KeyDown, 0)) // the newest, below the view
		m, c = current(t, h), h.m.screen.Conv
		row, _, _ = render.ConvMatchRow(h.m.State(), h.m.ui, c.W, c.Room, m)
		if want := row - c.Room + 1; h.m.ui.ConvTop != want {
			t.Fatalf("B's top %d, want %d: the newest match on B's last row", h.m.ui.ConvTop, want)
		}
	})
}

// TestSearchKeepsPart is design D6's part: the matches found again after
// events keep the current match by its part too, a call's name where its
// target, the newer, matches on the same row before it.
func TestSearchKeepsPart(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, _ := readyScript(t, claude.ProviderID, false, event.Started{})
		in, _ := json.Marshal(map[string]string{"file_path": "read.go"})
		pushEvents(h, claude.ProviderID, evOf(event.UserMessage, event.User{Text: "go"}), evOf(event.TurnStarted, nil),
			evOf(event.ToolStarted, event.Tool{ToolID: "x1", Name: "my-read", Input: in}),
			evOf(event.ToolCompleted, event.Tool{ToolID: "x1", Name: "my-read", Output: "ok"}),
			evOf(event.MessageCompleted, event.Completed{MessageID: "m1", Text: "done."}),
			evOf(event.TurnCompleted, event.TurnEnd{Outcome: "completed", Subtype: "success"}))
		h.m.View()
		openBar(h)
		typeQuery(h, "read")
		if v := h.m.ui.Search; len(v.Matches) != 2 || !current(t, h).Target {
			t.Fatalf("matches %+v, current %d: want the call's target, then its name", v.Matches, v.Cur)
		}
		h.draw(key(tea.KeyUp, 0))
		name := current(t, h)
		if name.Target || name.Off != len("my-") {
			t.Fatalf("↑: current %+v, want the call's name", name)
		}
		pushEvents(h, claude.ProviderID, evOf(event.UserMessage, event.User{Text: "next"}))
		h.m.View()
		if m := current(t, h); m != name {
			t.Fatalf("after an event: current %+v, want the name %+v kept", m, name)
		}
	})
}

// TestSearchGone is criterion 66.5's last case (design D6): when the
// current match's text is gone, its message ending with other words, the
// nearest match at or above its place becomes current, and zone B stays.
func TestSearchGone(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, _ := searchRig(t)
		pushEvents(h, claude.ProviderID, evOf(event.UserMessage, event.User{Text: "go on"}), evOf(event.TurnStarted, nil),
			evOf(event.MessageDelta, event.Delta{MessageID: "m4", Block: "text", Text: "a needle for now\n"}))
		h.m.View()
		openBar(h)
		typeQuery(h, "needle")
		was := current(t, h)
		if e := h.m.State().Conversation[was.Item]; e.MessageID != "m4" {
			t.Fatalf("the first match %+v in %q, want the streamed answer's", was, e.Text)
		}
		n, top := len(h.m.ui.Search.Matches), h.m.ui.ConvTop
		pushEvents(h, claude.ProviderID, evOf(event.MessageCompleted, event.Completed{MessageID: "m4", Text: "only hay in the end"}))
		h.m.View()
		v, m := h.m.ui.Search, current(t, h)
		if len(v.Matches) != n-1 || v.Cur != 0 || !strings.Contains(matchText(h, m), "One more needle") || !m.AtOrBefore(render.SearchPos{Item: was.Item, Line: was.Line, Off: was.Off}) {
			t.Fatalf("after the rewrite: %d matches (were %d), current %d %+v %q", len(v.Matches), n, v.Cur, m, matchText(h, m))
		}
		if h.m.ui.ConvTop != top {
			t.Errorf("zone B moved from %d to %d when the matches were found again", top, h.m.ui.ConvTop)
		}
	})
}

// TestSearchCloseKeys is criterion 66.8's keys (design D9): each key that
// leaves the conversation closes the bar, then acts; the dashboard's keys
// that leave it as it is act with the bar kept; any other key is the
// bar's, none reaching the draft.
func TestSearchCloseKeys(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, _ := searchRig(t)
		for _, tc := range []struct {
			k     tea.KeyPressMsg
			acted func() bool
		}{
			{key(tea.KeyF2, 0), nil},
			{key(tea.KeyF3, 0), nil},
			{key(tea.KeyF4, 0), func() bool { return h.m.ui.Toast == "no modes with this session" }}, // this rig's session has none
			{key(tea.KeyF6, 0), func() bool { return h.m.ui.Focus != "B" }},
			{key(tea.KeyF6, tea.ModShift), func() bool { return h.m.ui.Focus != "B" }},
			{key('p', tea.ModAlt), nil},
			{key('b', tea.ModAlt), nil},
			{key('g', tea.ModAlt), nil},
			{key('w', tea.ModCtrl), func() bool { return h.m.wt != nil }}, // last: the Worktrees view takes the keys
		} {
			h.m.ui.Focus = "B"
			openBar(h)
			h.draw(tc.k)
			if h.m.ui.Search.Open {
				t.Errorf("%s kept the bar", tc.k)
			}
			if tc.acted != nil && !tc.acted() {
				t.Errorf("%s closed the bar but did not act", tc.k)
			}
		}
		h.m.wt = nil
		h.m.ui.Focus = "B"
		h.m.edit.SetText("my draft")
		openBar(h)
		mouse := h.m.ui.MouseOff
		h.draw(key(tea.KeyF12, 0))
		h.draw(key(tea.KeyPgUp, tea.ModShift))
		h.draw(key('o', tea.ModCtrl))
		if !h.m.ui.Search.Open || h.m.ui.MouseOff == mouse || !h.m.ui.ConvHold {
			t.Errorf("F12, Shift+PgUp, Ctrl+O: bar %v, mouse capture toggled %v, B scrolled %v", h.m.ui.Search.Open, h.m.ui.MouseOff != mouse, h.m.ui.ConvHold)
		}
		h.draw(key('g', tea.ModCtrl))
		h.draw(key('k', tea.ModCtrl))
		if !h.m.ui.Search.Open || h.m.ui.DraftWin.Open || h.m.ui.Search.Query != "" || h.m.edit.Text() != "my draft" {
			t.Errorf("Ctrl+G, Ctrl+K: bar %v, draft window %v, query %q, draft %q", h.m.ui.Search.Open, h.m.ui.DraftWin.Open, h.m.ui.Search.Query, h.m.edit.Text())
		}
	})
}

// TestSearchNotWritten is criterion 66.8's last part (design D2): the query
// in no file the dashboard writes, the draft's recovery among them, and in
// nothing written to the CLI, the draft stored as it was.
func TestSearchNotWritten(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		d := input.Drafts{Dir: t.TempDir()}
		w := &wireClaude{}
		h := wireRig(t, w, Options{Drafts: &d})
		h.send(tea.PasteMsg{Content: "a draft kept"})
		h.settle()
		openBar(h)
		typeQuery(h, "qvx-only-in-the-bar")
		h.send(tea.PasteMsg{Content: " and qvx pasted"})
		h.settle()
		h.draw(key(tea.KeyEscape, 0))
		h.settle()
		if err := filepath.WalkDir(d.Dir, func(p string, e fs.DirEntry, err error) error {
			if err != nil || e.IsDir() {
				return err
			}
			if b, err := os.ReadFile(p); err == nil && bytes.Contains(b, []byte("qvx")) {
				t.Errorf("the query in %s: %q", p, b)
			}
			return nil
		}); err != nil {
			t.Fatal(err)
		}
		w.mu.Lock()
		lines := append([]string(nil), w.lines...)
		w.mu.Unlock()
		for _, l := range lines {
			if strings.Contains(l, "qvx") {
				t.Errorf("the query written to the CLI: %s", l)
			}
		}
		if got := draftFile(t, d, h.m.State().Context.Project); string(got) != "a draft kept" {
			t.Errorf("the stored draft %q, want %q", got, "a draft kept")
		}
	})
}

// TestSearchCounts is criterion 66.3's words: no match, its note of what is
// not searched; past the cap, the newest kept.
func TestSearchCounts(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, _ := searchRig(t)
		openBar(h)
		typeQuery(h, "zzqx")
		if l := lineWith(h, "no match"); !strings.Contains(l, "no match · calls' inputs and outputs not searched") {
			t.Errorf("no match: %q", l)
		}
		h.m.ui.Search.Matches = make([]render.ConvMatch, render.MaxMatches)
		h.m.ui.Search.Capped, h.m.ui.Search.Cur = true, 0
		if l := lineWith(h, "match 1 of"); !strings.Contains(l, "match 1 of the newest 10,000") {
			t.Errorf("past the cap: %q", l)
		}
	})
}

// TestSearchCapStart is criterion 66.2's cap in the application: past
// 10,000 matches the newest are kept and the count says so, and with zone
// B scrolled up the first current match is still the newest at or above
// its last row shown.
func TestSearchCapStart(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, _ := readyScript(t, claude.ProviderID, false, event.Started{})
		var ans strings.Builder
		for i := range 120 {
			fmt.Fprintf(&ans, "%s%d\n\n", strings.Repeat("x ", 100), i)
		}
		searchTurn(h, "m1", "many", nil, ans.String())
		h.m.View()
		for range 3 {
			h.draw(key(tea.KeyPgUp, tea.ModShift))
		}
		openBar(h)
		typeQuery(h, "x")
		v, bottom := h.m.ui.Search, h.m.screen.Conv.Bottom
		if !v.Capped || len(v.Matches) != render.MaxMatches || !strings.Contains(lineWith(h, "match"), "of the newest 10,000") {
			t.Fatalf("capped %v, %d matches, the bar %q", v.Capped, len(v.Matches), lineWith(h, "match"))
		}
		m := current(t, h)
		if v.Cur == 0 || !m.AtOrBefore(bottom) {
			t.Fatalf("the first match, %d, %+v, B's last row %+v: want the newest at or above it, not the newest of all", v.Cur, m, bottom)
		}
		for _, newer := range v.Matches[:v.Cur] {
			if newer.AtOrBefore(bottom) {
				t.Fatalf("a newer match %+v at or above B's last row %+v passed over", newer, bottom)
			}
		}
	})
}
