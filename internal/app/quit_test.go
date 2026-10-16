package app

import (
	"bytes"
	"reflect"
	"strings"
	"testing"
	"testing/synctest"

	tea "charm.land/bubbletea/v2"

	"github.com/Evengelius/agent-command-center/internal/input"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// ctrlQ is Ctrl+Q as the terminal reports it.
var ctrlQ = key('q', tea.ModCtrl)

// stdinOrder is what reached the fake CLI's stdin from line from on: deny
// for a deny of id, interrupt, the end of input as EOF.
func stdinOrder(w *wireClaude, eof *int, from int, id string) []string {
	w.mu.Lock()
	defer w.mu.Unlock()
	var order []string
	for i := from; i < len(w.lines); i++ {
		l := w.lines[i]
		switch {
		case strings.Contains(l, `"control_response"`) && strings.Contains(l, id) && strings.Contains(l, `"deny"`):
			order = append(order, "deny")
		case strings.Contains(l, `"subtype":"interrupt"`):
			order = append(order, "interrupt")
		default:
			order = append(order, l)
		}
		if i+1 == *eof {
			order = append(order, "EOF")
		}
	}
	if *eof == len(w.lines) && *eof == from {
		order = append(order, "EOF")
	}
	return order
}

// leave is what Run does once the program ends: the draft written, the
// session closed, the resume line.
func leave(t *testing.T, h *harness) string {
	t.Helper()
	if !h.quit {
		t.Fatal("the program did not quit")
	}
	if err := h.m.flushDraft(); err != nil {
		t.Fatal(err)
	}
	h.m.Shutdown()
	return h.m.resumeLine()
}

// TestQuitDuringTurn: Ctrl+Q while a turn runs asks first; confirmed, the
// pending approval is denied and the turn interrupted before the CLI's
// input closes; the draft and the resume line survive (ui-design.md, Keys;
// reliability.md, Interruption and cancellation).
func TestQuitDuringTurn(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		recs := probe(t, "p2-claude-conversation.jsonl")
		const id = "5438db7f-8078-47e9-bfda-767bbf738c1a"
		d := input.Drafts{Dir: t.TempDir()}
		eof := -1
		w := &wireClaude{}
		w.onEOF = func() { w.mu.Lock(); eof = len(w.lines); w.mu.Unlock() }
		h := wireRig(t, w, Options{Drafts: &d})
		text, at := firstUser(t, recs)
		h.send(tea.PasteMsg{Content: text})
		h.send(key(tea.KeyEnter, 0))
		replay(t, h, w.last(), recs, at+1, 36) // the Bash approval pending
		h.send(tea.PasteMsg{Content: draftText})
		s := h.m.State()
		if s.Turn.Phase != state.WaitingApproval || s.Approvals[id].Status != state.Pending {
			t.Fatalf("turn %q, approval %v; want a pending approval", s.Turn.Phase, s.Approvals[id])
		}
		w.mu.Lock()
		before := len(w.lines)
		w.mu.Unlock()

		h.send(ctrlQ)
		if !s.QuitAsked || s.Session.Phase != state.Ready || h.quit {
			t.Fatalf("asked %v, session %q, quit %v after one Ctrl+Q; want a question", s.QuitAsked, s.Session.Phase, h.quit)
		}
		if f := frameText(h); !strings.Contains(f, "quit? the running turn will be stopped · Ctrl+Q again to quit · Esc to stay") {
			t.Errorf("no confirmation on screen:\n%s", f)
		}
		h.send(key(tea.KeyEscape, 0))
		if s.QuitAsked || strings.Contains(frameText(h), "quit?") || s.Turn.Phase != state.WaitingApproval {
			t.Fatalf("Esc: asked %v, turn %q; want the session going on", s.QuitAsked, s.Turn.Phase)
		}
		if h.m.edit.Text() != draftText {
			t.Fatalf("editor %q after Esc; want the draft untouched", h.m.edit.Text())
		}
		if o := stdinOrder(w, &eof, before, id); len(o) != 0 {
			t.Fatalf("stdin %q before the quit was confirmed; want nothing", o)
		}

		h.send(ctrlQ)
		h.send(ctrlQ)
		if o := stdinOrder(w, &eof, before, id); !reflect.DeepEqual(o, []string{"deny", "interrupt", "EOF"}) {
			t.Fatalf("stdin %q; want the deny, the interrupt, then the end of input", o)
		}
		if s.Session.Phase != state.Closed || s.Turn.Phase != state.Idle {
			t.Errorf("session %q, turn %q; want Closed, Idle", s.Session.Phase, s.Turn.Phase)
		}
		line := leave(t, h)
		if got := draftFile(t, d, s.Context.Project); !bytes.Equal(got, []byte(draftText)) {
			t.Errorf("stored draft %q after the quit", got)
		}
		if !strings.Contains(line, "claude --resume "+s.Session.ID) || s.Session.ID == "" {
			t.Errorf("resume line %q", line)
		}
	})
}

// TestQuitBeforeAccepted: quitting while a sent message waits for Claude
// Code keeps it: the CLI ends without taking it, so it goes back to the
// editor and the stored draft (ASM-23).
func TestQuitBeforeAccepted(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		d := input.Drafts{Dir: t.TempDir()}
		eof := -1
		w := &wireClaude{}
		w.onEOF = func() { w.mu.Lock(); eof = len(w.lines); w.mu.Unlock() }
		h := wireRig(t, w, Options{Drafts: &d})
		s := h.m.State()
		h.send(tea.PasteMsg{Content: draftText})
		h.send(key(tea.KeyEnter, 0))
		if s.Turn.Phase != state.Submitted {
			t.Fatalf("turn %q after Enter", s.Turn.Phase)
		}
		w.mu.Lock()
		before := len(w.lines)
		w.mu.Unlock()
		h.send(ctrlQ)
		if !s.QuitAsked {
			t.Fatal("no question while the message waits")
		}
		h.send(ctrlQ)
		if o := stdinOrder(w, &eof, before, ""); !reflect.DeepEqual(o, []string{"interrupt", "EOF"}) {
			t.Fatalf("stdin %q; want the interrupt, then the end of input", o)
		}
		if h.m.edit.Text() != draftText {
			t.Errorf("editor %q; want the message back", h.m.edit.Text())
		}
		leave(t, h)
		if got := draftFile(t, d, s.Context.Project); !bytes.Equal(got, []byte(draftText)) {
			t.Errorf("stored draft %q after the quit; want the message", got)
		}
	})
}

// TestQuitIdle: with nothing running, one Ctrl+Q quits.
func TestQuitIdle(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := &wireClaude{}
		h := wireRig(t, w, Options{})
		h.send(ctrlQ)
		if s := h.m.State(); s.QuitAsked || s.Session.Phase != state.Closed || !h.quit {
			t.Fatalf("asked %v, session %q, quit %v; want closed and quit", s.QuitAsked, s.Session.Phase, h.quit)
		}
	})
}
