package app

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/Evengelius/agent-command-center/internal/render"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// p2Pending is P2 replayed to its pending Write of hello.txt.
func p2Pending(t *testing.T) (*harness, *wireClaude) {
	t.Helper()
	recs := probe(t, "p2-claude-conversation.jsonl")
	w := &wireClaude{}
	h := wireRig(t, w, Options{})
	text, at := firstUser(t, recs)
	h.send(tea.PasteMsg{Content: text})
	h.send(key(tea.KeyEnter, 0))
	replay(t, h, w.last(), recs, at+1, 36)
	if ap := h.m.State().Approvals[p2Approval]; ap == nil || ap.Status != state.Pending {
		t.Fatalf("approval %+v; want pending", ap)
	}
	return h, w
}

const p2Approval = "5438db7f-8078-47e9-bfda-767bbf738c1a"

// TestApprovalViewer is plan 5, item 3: Enter on a focused card shows the
// whole request full screen; Esc returns to the card, a d c answer there.
func TestApprovalViewer(t *testing.T) {
	t.Run("open, back, answer", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			h, w := p2Pending(t)
			h.check = func(f string) { noPersistentAnswer(t, f) }
			h.send(key(tea.KeyF2, 0))
			card := h.frame()
			h.send(key(tea.KeyEnter, 0))
			f := h.frame()
			for _, want := range []string{"AGENT COMMAND CENTER", "◀ back (Esc)", "! APPROVAL", "orchestrator · Write", "hello.txt", "+hello", "a allow once · d deny · c cancel the turn", "Esc back to the card"} {
				if !strings.Contains(f, want) {
					t.Errorf("viewer lacks %q:\n%s", want, f)
				}
			}
			for _, zone := range []string{"A · ORCHESTRATOR", "B · CONVERSATION", "C · LIVE AGENTS", "D · QUOTA"} {
				if strings.Contains(f, zone) {
					t.Errorf("viewer draws zone %q:\n%s", zone, f)
				}
			}

			h.send(typed('x')) // answers nothing: nothing happens
			if h.m.ui.View != p2Approval || h.m.edit.Text() != "" || len(answersFor(w, p2Approval)) != 0 {
				t.Fatalf("x in the viewer: view %q, editor %q, %d answers", h.m.ui.View, h.m.edit.Text(), len(answersFor(w, p2Approval)))
			}
			h.send(key(tea.KeyF1, 0))
			if f := h.frame(); !h.m.ui.Keys || !strings.Contains(f, "Enter") || !strings.Contains(f, "the whole command, diff or question, full screen") {
				t.Errorf("key overlay from the viewer:\n%s", f)
			}
			h.send(key(tea.KeyEscape, 0))
			if h.m.ui.Keys || h.m.ui.View != p2Approval {
				t.Fatalf("Esc on the overlay: keys %v, view %q; want the viewer back", h.m.ui.Keys, h.m.ui.View)
			}

			h.send(key(tea.KeyEscape, 0))
			f = h.frame()
			if h.m.ui.View != "" || h.m.ui.Card != p2Approval || h.m.State().Approvals[p2Approval].Status != state.Pending {
				t.Fatalf("Esc: view %q, card %q, status %q; want the card focused, pending", h.m.ui.View, h.m.ui.Card, h.m.State().Approvals[p2Approval].Status)
			}
			if !strings.Contains(f, "B · CONVERSATION") || !strings.Contains(f, "▸ a allow once · d deny · c cancel the turn · Enter full view · Esc later") {
				t.Errorf("after Esc, the focused card is not drawn:\n%s", f)
			}
			if f != card {
				t.Errorf("after Esc, the dashboard differs from before Enter:\n%s\nbefore:\n%s", f, card)
			}
			h.send(key(tea.KeyEscape, 0))
			if h.m.ui.Card != "" || h.m.ui.Focus != "B" {
				t.Fatalf("second Esc: card %q, focus %q; want the input", h.m.ui.Card, h.m.ui.Focus)
			}

			h.send(key(tea.KeyF2, 0))
			h.send(key(tea.KeyEnter, 0))
			h.send(typed('a'))
			h.send(typed('a'))
			got := answersFor(w, p2Approval)
			if len(got) != 1 || got[0]["behavior"] != "allow" {
				t.Fatalf("answers %v; want one allow", got)
			}
			if ap := h.m.State().Approvals[p2Approval]; ap.Status != state.Allowed || h.m.ui.View != "" {
				t.Errorf("after a: status %q, view %q; want allowed, closed", ap.Status, h.m.ui.View)
			}
			if h.m.edit.Text() != "a" { // the second a reaches the input, as on the card
				t.Errorf("editor %q after the answer", h.m.edit.Text())
			}
		})
	})

	t.Run("scroll", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			h, _ := p2Pending(t)
			var cmd []string
			for i := 1; i <= 200; i++ {
				cmd = append(cmd, fmt.Sprintf("echo line %d", i))
			}
			ap := h.m.State().Approvals[p2Approval]
			ap.Tool, ap.Summary, ap.Input = "Bash", "", json.RawMessage(`{"command":`+quote(strings.Join(cmd, "\n"))+`}`)
			h.send(key(tea.KeyF2, 0))
			h.draw(key(tea.KeyEnter, 0))
			has := func(n int) bool { return strings.Contains(h.frame(), fmt.Sprintf("$ echo line %d ", n)) }
			if !has(1) || has(200) {
				t.Fatalf("opens not at its top:\n%s", h.frame())
			}
			h.draw(key(tea.KeyEnd, 0))
			if has(1) || !has(200) {
				t.Errorf("End:\n%s", h.frame())
			}
			h.draw(key(tea.KeyHome, 0))
			if !has(1) {
				t.Errorf("Home:\n%s", h.frame())
			}
			h.draw(key(tea.KeyDown, 0))
			if has(1) || !has(2) {
				t.Errorf("Down:\n%s", h.frame())
			}
			h.draw(key(tea.KeyUp, 0))
			h.draw(key(tea.KeyPgDown, 0))
			if has(1) || has(200) {
				t.Errorf("PgDn:\n%s", h.frame())
			}
			h.draw(key(tea.KeyPgUp, 0))
			if !has(1) {
				t.Errorf("PgUp:\n%s", h.frame())
			}
			h.draw(tea.MouseWheelMsg{X: 10, Y: 8, Button: tea.MouseWheelDown})
			if has(1) {
				t.Errorf("wheel down:\n%s", h.frame())
			}
			if strings.Contains(h.frame(), "more lines") {
				t.Errorf("the viewer cuts:\n%s", h.frame())
			}
			h.send(tea.WindowSizeMsg{Width: 120, Height: 220}) // tall enough: every line shows
			h.draw(key(tea.KeyHome, 0))
			for i := 1; i <= 200; i++ {
				if !has(i) {
					t.Fatalf("line %d missing at 220 rows:\n%s", i, h.frame())
				}
			}
		})
	})

	t.Run("resolved while open", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			h, w := p2Pending(t)
			h.send(key(tea.KeyF2, 0))
			h.send(key(tea.KeyEnter, 0))
			h.m.State().Approvals[p2Approval].Status = state.Expired
			h.draw(tea.WindowSizeMsg{Width: 120, Height: 40}) // an update, as the event that expires it
			h.draw(typed('a'))
			f := h.frame()
			if !strings.Contains(f, "· EXPIRED") || !strings.Contains(f, "expired, not answered") || h.m.ui.View != p2Approval {
				t.Errorf("view %q, frame:\n%s", h.m.ui.View, f)
			}
			if n := len(answersFor(w, p2Approval)); n != 0 {
				t.Errorf("%d answers to an expired approval", n)
			}
			h.send(key(tea.KeyEscape, 0))
			if h.m.ui.View != "" || h.m.ui.Card != "" || h.m.ui.Focus != "B" {
				t.Errorf("Esc: view %q, card %q, focus %q; want the input", h.m.ui.View, h.m.ui.Card, h.m.ui.Focus)
			}
		})
	})

	t.Run("Codex fileChange", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			recs := probe(t, "p6b-codex-app-server.jsonl")
			text, at := firstTurn(t, recs, 1)
			w := &wireCodex{thread: p6bThread}
			h := codexWire(t, w, Options{})
			h.send(tea.PasteMsg{Content: text})
			h.send(key(tea.KeyEnter, 0))
			codexPlay(t, h, w, recs, at+1, 37)
			h.send(key(tea.KeyF2, 0))
			h.send(key(tea.KeyEnter, 0))
			f := h.frame()
			for _, want := range []string{"! APPROVAL", "orchestrator · fileChange", `\p6b\hello.txt  add`, "+hello"} {
				if !strings.Contains(f, want) {
					t.Errorf("viewer lacks %q:\n%s", want, f)
				}
			}
			before := len(w.written(0))
			h.send(typed('d'))
			h.send(typed('d'))
			h.settle()
			if got, want := w.written(before), []string{`{"id":0,"result":{"decision":"decline"}}`}; !reflect.DeepEqual(got, want) {
				t.Errorf("written\n got %q\nwant %q", got, want)
			}
			if h.m.ui.View != "" || h.m.State().Approvals["0"].Status != state.Denied {
				t.Errorf("view %q, status %q", h.m.ui.View, h.m.State().Approvals["0"].Status)
			}
		})
	})
}

// p2Done is P2 replayed whole: the Write allowed, the turn finished.
func p2Done(t *testing.T) *harness {
	t.Helper()
	recs := probe(t, "p2-claude-conversation.jsonl")
	w := &wireClaude{}
	h := wireRig(t, w, Options{})
	text, at := firstUser(t, recs)
	h.send(tea.PasteMsg{Content: text})
	h.send(key(tea.KeyEnter, 0))
	replay(t, h, w.last(), recs, at+1, len(recs))
	return h
}

// entryAt is the 1-based index of the last entry ok accepts; 0 none.
func entryAt(s *state.State, ok func(e state.Entry) bool) int {
	for i := len(s.Conversation) - 1; i >= 0; i-- {
		if ok(s.Conversation[i]) {
			return i + 1
		}
	}
	return 0
}

// TestEntryViewer is plan 5, item 4: ↑ from an empty input puts a cursor on
// B's entries shown in part, Enter shows one whole, full screen; → opens
// nothing (the owner's live review, 2026-10-20).
func TestEntryViewer(t *testing.T) {
	t.Run("cursor, view, back", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			h := p2Done(t)
			s := h.m.State()
			var paste []string
			for i := 1; i <= 200; i++ {
				paste = append(paste, fmt.Sprintf("pasted line %d", i))
			}
			h.send(tea.PasteMsg{Content: strings.Join(paste, "\n")})
			h.draw(key(tea.KeyEnter, 0))
			p := entryAt(s, func(e state.Entry) bool { return e.Role == "user" && strings.Count(e.Text, "\n") == 199 })
			tool := entryAt(s, func(e state.Entry) bool { return e.Role == "tool" && e.Text == "Write" })
			fail := entryAt(s, func(e state.Entry) bool { return e.Role == "error" })
			// P2's third turn thought before the interrupt: its summary row is a
			// stop of the cursor too (plan 22, design 8).
			thought := entryAt(s, func(e state.Entry) bool { return e.Role == "thinking" })
			var exp []int // P2's expandable entries: the Write, the thought, its third turn's failure, the paste
			for i, e := range s.Conversation {
				if render.Expandable(e) {
					exp = append(exp, i+1)
				}
			}
			if !reflect.DeepEqual(exp, []int{tool, thought, fail, p}) || tool == 0 || thought == 0 {
				t.Fatalf("expandable %v, want the Write, the thought, the failure and the paste", exp)
			}
			rows := h.m.screen.Conv.Rows

			h.draw(key(tea.KeyUp, 0))
			f := h.frame()
			if h.m.ui.Entry != p || !rowWith(f, "▎▸", "… 195 more lines", "Enter full view · ↑↓ entries · Esc back") { // the cursor in the card's margin (plan 20)
				t.Fatalf("↑: entry %d, want %d:\n%s", h.m.ui.Entry, p, f)
			}
			if h.m.screen.Conv.Rows != rows {
				t.Errorf("the cursor changed B's rows: %d, want %d", h.m.screen.Conv.Rows, rows)
			}
			h.draw(key(tea.KeyUp, 0))
			if f := h.frame(); h.m.ui.Entry != fail || !rowWith(f, "✗ turn failed", "Enter full view") {
				t.Fatalf("↑↑: entry %d, want the failure at %d:\n%s", h.m.ui.Entry, fail, f)
			}
			h.draw(key(tea.KeyUp, 0))
			if f := h.frame(); h.m.ui.Entry != thought || !rowWith(f, "✗ Thinking interrupted", "Enter full view · → details") {
				t.Fatalf("↑↑↑: entry %d, want the thought at %d:\n%s", h.m.ui.Entry, thought, f)
			}
			h.draw(key(tea.KeyUp, 0))
			if f := h.frame(); h.m.ui.Entry != tool || !rowWith(f, "▸ ✓ Write", "Enter full view") {
				t.Fatalf("↑↑↑↑: entry %d, want the Write at %d:\n%s", h.m.ui.Entry, tool, f)
			}
			h.draw(key(tea.KeyUp, 0)) // nothing older to expand: it stays
			if h.m.ui.Entry != tool {
				t.Errorf("↑ at the oldest: entry %d, want %d", h.m.ui.Entry, tool)
			}
			h.draw(key(tea.KeyDown, 0))
			if h.m.ui.Entry != thought {
				t.Errorf("↓: entry %d, want %d", h.m.ui.Entry, thought)
			}
			h.draw(key(tea.KeyDown, 0))
			if h.m.ui.Entry != fail {
				t.Errorf("↓↓: entry %d, want %d", h.m.ui.Entry, fail)
			}
			h.draw(key(tea.KeyEnter, 0))
			want := strings.Split(s.Conversation[fail-1].Text, "\n")
			if f := h.frame(); !strings.Contains(f, "✗ TURN FAILED") || !strings.Contains(f, want[len(want)-1]) || !strings.Contains(f, want[0]) {
				t.Errorf("the failure's view lacks a line of %q:\n%s", want, f)
			}
			h.draw(key(tea.KeyEscape, 0))
			h.draw(key(tea.KeyDown, 0))
			h.draw(key(tea.KeyDown, 0))
			if h.m.ui.Entry != 0 || h.m.ui.Focus != "B" || strings.Contains(h.frame(), "Enter full view") {
				t.Fatalf("↓ past the newest: entry %d, focus %q; want the input", h.m.ui.Entry, h.m.ui.Focus)
			}

			h.draw(key(tea.KeyUp, 0))
			h.draw(typed('x'))
			if h.m.ui.Entry != 0 || h.m.edit.Text() != "x" {
				t.Fatalf("typing on the cursor: entry %d, editor %q; want the input with x", h.m.ui.Entry, h.m.edit.Text())
			}
			h.draw(key(tea.KeyUp, 0))
			if h.m.ui.Entry != 0 {
				t.Errorf("↑ with a draft moved the cursor to %d", h.m.ui.Entry)
			}
			h.draw(key(tea.KeyEnd, 0)) // ↑ took the caret to the start
			h.draw(key(tea.KeyBackspace, 0))

			h.draw(key(tea.KeyUp, 0))
			before := h.frame()
			h.draw(key(tea.KeyEnter, 0))
			f = h.frame()
			for _, want := range []string{"◀ back (Esc)", "YOU", "200 lines", "pasted line 1 ", "↑↓ PgUp PgDn scroll"} {
				if !strings.Contains(f, want) {
					t.Errorf("view lacks %q:\n%s", want, f)
				}
			}
			if h.m.ui.ViewEntry != p || strings.Contains(f, "B · CONVERSATION") || strings.Contains(f, "more lines") || len(s.Conversation) != p {
				t.Errorf("Enter: view %d, want %d (nothing sent):\n%s", h.m.ui.ViewEntry, p, f)
			}
			h.draw(key(tea.KeyEnd, 0))
			if f := h.frame(); !strings.Contains(f, "pasted line 200 ") || strings.Contains(f, "pasted line 1 ") {
				t.Errorf("End:\n%s", f)
			}
			h.draw(tea.MouseWheelMsg{X: 10, Y: 8, Button: tea.MouseWheelUp})
			h.draw(key(tea.KeyHome, 0))
			if !strings.Contains(h.frame(), "pasted line 1 ") {
				t.Errorf("Home:\n%s", h.frame())
			}
			h.draw(key(tea.KeyEscape, 0))
			if f := h.frame(); h.m.ui.ViewEntry != 0 || h.m.ui.Entry != p || f != before {
				t.Errorf("Esc: view %d, entry %d; the frame differs from before Enter:\n%s\nbefore:\n%s", h.m.ui.ViewEntry, h.m.ui.Entry, f, before)
			}
			h.draw(key(tea.KeyEscape, 0))
			if h.m.ui.Entry != 0 || h.m.ui.Focus != "B" {
				t.Errorf("second Esc: entry %d, focus %q", h.m.ui.Entry, h.m.ui.Focus)
			}

			h.draw(key(tea.KeyUp, 0))
			h.draw(key(tea.KeyRight, 0)) // → unfolds, it opens nothing (the owner's live review)
			if h.m.ui.ViewEntry != 0 || h.m.ui.Entry != p {
				t.Errorf("→: view %d, entry %d; want no view, the cursor kept on %d", h.m.ui.ViewEntry, h.m.ui.Entry, p)
			}
			h.draw(key(tea.KeyUp, 0)) // the failure, the thought, then the Write
			h.draw(key(tea.KeyUp, 0))
			h.draw(key(tea.KeyUp, 0))
			h.draw(key(tea.KeyEnter, 0))
			if tc := s.Tools[s.Conversation[tool-1].MessageID]; !strings.HasPrefix(tc.Output, "File created successfully at: ") {
				t.Fatalf("P2's Write output %q", tc.Output)
			}
			f = h.frame()
			for _, want := range []string{"✓ Write", "completed", "hello.txt", "+hello", "OUTPUT", "File created successfully at: "} {
				if !strings.Contains(f, want) {
					t.Errorf("the Write's view lacks %q:\n%s", want, f)
				}
			}
			h.draw(key(tea.KeyEscape, 0))
			h.draw(key(tea.KeyF2, 0)) // no card waits: the cursor leaves all the same
			if h.m.ui.Entry != 0 || h.m.ui.ViewEntry != 0 {
				t.Errorf("F2: entry %d, view %d", h.m.ui.Entry, h.m.ui.ViewEntry)
			}
		})
	})

	t.Run("B scrolls to it", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			h := p2Done(t)
			s := h.m.State()
			const n = 12
			for k := 1; k <= n; k++ { // message k folds with 16+k more lines
				s.Conversation = append(s.Conversation, state.Entry{Seq: uint64(len(s.Conversation) + 1), Role: "user", Text: strings.Repeat("line\n", 20+k) + "end", At: testNow})
			}
			h.draw(tea.WindowSizeMsg{Width: 160, Height: 60})
			if c := h.m.screen.Conv; c.Rows < 2*c.Room {
				t.Fatalf("%d rows in %d: nothing to scroll", c.Rows, c.Room)
			}
			for k := n; k >= 1; k-- {
				h.draw(key(tea.KeyUp, 0))
				if f := h.frame(); !rowWith(f, "▎▸", fmt.Sprintf("… %d more lines", 16+k), "Enter full view") {
					t.Fatalf("↑ to message %d: not in view:\n%s", k, f)
				}
			}
			h.draw(key(tea.KeyUp, 0))
			if f := h.frame(); !rowWith(f, "✗ turn failed", "Enter full view") {
				t.Fatalf("↑ to the failure: not in view:\n%s", f)
			}
			h.draw(key(tea.KeyUp, 0))
			if f := h.frame(); !rowWith(f, "✗ Thinking interrupted", "Enter full view") {
				t.Fatalf("↑ to the thought: not in view:\n%s", f)
			}
			h.draw(key(tea.KeyUp, 0))
			if f := h.frame(); !rowWith(f, "▸ ✓ Write", "Enter full view") {
				t.Fatalf("↑ to the Write: not in view:\n%s", f)
			}
			h.draw(key(tea.KeyDown, 0)) // back over the thought and the failure
			h.draw(key(tea.KeyDown, 0))
			for k := 1; k <= n; k++ {
				h.draw(key(tea.KeyDown, 0))
				if f := h.frame(); !rowWith(f, "▎▸", fmt.Sprintf("… %d more lines", 16+k), "Enter full view") {
					t.Fatalf("↓ to message %d: not in view:\n%s", k, f)
				}
			}
		})
	})

	t.Run("live", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			h, _ := p2Pending(t)
			s := h.m.State()
			tool := entryAt(s, func(e state.Entry) bool { return e.Role == "tool" && e.Text == "Write" })
			h.draw(key(tea.KeyUp, 0))
			h.draw(key(tea.KeyEnter, 0))
			if f := h.frame(); h.m.ui.ViewEntry != tool || tool == 0 || !strings.Contains(f, "no output") {
				t.Fatalf("view %d, want the Write at %d:\n%s", h.m.ui.ViewEntry, tool, f)
			}
			tc := s.Tools[s.Conversation[tool-1].MessageID]
			tc.Output, tc.Status, tc.EndedAt = "File created successfully\n", state.ToolCompleted, tc.StartedAt.Add(time.Second)
			h.draw(tea.WindowSizeMsg{Width: 160, Height: 60}) // an update, as the event that ends it
			if f := h.frame(); !strings.Contains(f, "File created successfully") || !strings.Contains(f, "completed · 1s") || strings.Contains(f, "no output") {
				t.Errorf("the output that arrived does not show:\n%s", f)
			}
		})
	})
}

// rowWith reports whether a row of frame f holds every one of parts.
func rowWith(f string, parts ...string) bool {
	for _, r := range strings.Split(f, "\n") {
		all := true
		for _, p := range parts {
			all = all && strings.Contains(r, p)
		}
		if all {
			return true
		}
	}
	return false
}
