package app

import (
	"fmt"
	"slices"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/Evengelius/agent-command-center/internal/event"
	"github.com/Evengelius/agent-command-center/internal/render"
)

// Plan 5, item 5: zone A's tree, zone C's list and C's preview scroll.

// feed sends h's session the events build adds, after its last event.
func feed(h *harness, build func(add func(k event.Kind, agent string, p any))) {
	s := h.m.State()
	at, seq := s.Log[len(s.Log)-1].At, s.Log[len(s.Log)-1].Seq
	var evs []event.Event
	build(func(k event.Kind, agent string, p any) {
		seq++
		evs = append(evs, event.Event{Seq: seq, At: at.Add(time.Duration(len(evs)+1) * time.Second), Provider: s.CLI(), Kind: k, Origin: event.Observed, Agent: agent, Payload: p})
	})
	h.send(eventsMsg{sess: h.m.sess, evs: evs, closed: true})
}

// agentRun adds agent key's start, msgs messages of its own and its end.
func agentRun(add func(k event.Kind, agent string, p any), key, label string, msgs int) {
	add(event.AgentStarted, "", event.Agent{Key: key, Label: label, Type: "subagent", Description: "scan"})
	for j := 0; j < msgs; j++ {
		id := fmt.Sprintf("%s-m%d", key, j)
		add(event.MessageDelta, key, event.Delta{MessageID: id, Block: "text", Text: fmt.Sprintf("message %d", j)})
		add(event.MessageCompleted, key, event.Completed{MessageID: id})
	}
	add(event.AgentCompleted, "", event.Agent{Key: key, Status: "completed", Summary: "done"})
}

// busy adds n turns of the owner's to h's session, each a stage with one
// agent, p<from> to p<from+n-1>; the first of them says msgs messages of its
// own, for the preview. Each turn clears zone C of the earlier ones (plan 18).
func busy(h *harness, from, n, msgs int) {
	feed(h, func(add func(k event.Kind, agent string, p any)) {
		for i := from; i < from+n; i++ {
			said := 0
			if i == from {
				said = msgs
			}
			add(event.UserMessage, "", event.User{Text: fmt.Sprintf("turn %d", i)})
			add(event.TurnStarted, "", nil)
			agentRun(add, fmt.Sprintf("p%d", i), fmt.Sprintf("agent%d", i), said)
			add(event.TurnCompleted, "", event.TurnEnd{Outcome: "completed", DurationMS: 1000})
		}
	})
}

// crowd adds one turn of the owner's that deploys n agents, c00 to c<n-1>,
// so that zone C lists them all (plan 18); the first says msgs messages.
func crowd(h *harness, n, msgs int) {
	feed(h, func(add func(k event.Kind, agent string, p any)) {
		add(event.UserMessage, "", event.User{Text: "crowd"})
		add(event.TurnStarted, "", nil)
		for i := 0; i < n; i++ {
			said := 0
			if i == 0 {
				said = msgs
			}
			agentRun(add, fmt.Sprintf("c%02d", i), fmt.Sprintf("crowd%02d", i), said)
		}
		add(event.TurnCompleted, "", event.TurnEnd{Outcome: "completed", DurationMS: 1000})
	})
}

// cellOf is a cell of panel p in the last frame.
func cellOf(t *testing.T, h *harness, p byte) (int, int) {
	t.Helper()
	for y := range h.m.ui.Height {
		for x := range h.m.ui.Width {
			if h.m.screen.PanelAt(x, y) == p {
				return x, y
			}
		}
	}
	t.Fatalf("no panel %c in:\n%s", p, h.frame())
	return 0, 0
}

func wheel(h *harness, x, y int, up bool) {
	b := tea.MouseWheelDown
	if up {
		b = tea.MouseWheelUp
	}
	h.draw(tea.MouseWheelMsg{X: x, Y: y, Button: b})
}

// paneDash is the codex-subagents mock with 60 more turns at w x h, drawn.
func paneDash(t *testing.T, w, ht, msgs int) *harness {
	h := mockDash(t, "codex-subagents.jsonl")
	busy(h, 0, 60, msgs)
	h.draw(tea.WindowSizeMsg{Width: w, Height: ht})
	return h
}

func TestPaneScroll(t *testing.T) {
	// Criteria 1, 3, 5: focus A, the keys, the cursor, a new stage.
	t.Run("tree", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			h := paneDash(t, 120, 40, 0)
			focusZone(t, h, "A")
			v := h.m.screen.Tree
			if v.Rows <= v.Room || v.Top != v.Rows-v.Room {
				t.Fatalf("tree %+v: want more rows than room, at its tail", v)
			}
			bottom, page := v.Rows-v.Room, v.Room-1
			h.draw(key(tea.KeyPgUp, 0))
			if g := h.m.screen.Tree.Top; g != bottom-page || !h.m.ui.Tree.Hold {
				t.Fatalf("PgUp: top %d hold %v, want %d held", g, h.m.ui.Tree.Hold, bottom-page)
			}
			// "· End follows" is left out where zone A is too narrow (plan 20).
			if f := h.frame(); !strings.Contains(f, fmt.Sprintf("↑ %d ↓ %d", bottom-page, page)) {
				t.Errorf("held hint missing:\n%s", f)
			}
			h.draw(key(tea.KeyPgDown, 0))
			if h.m.ui.Tree.Hold || h.m.screen.Tree.Top != bottom {
				t.Errorf("PgDn to the tail: %+v top %d, want following at %d", h.m.ui.Tree, h.m.screen.Tree.Top, bottom)
			}
			h.draw(key(tea.KeyHome, 0))
			if h.m.screen.Tree.Top != 0 || !h.m.ui.Tree.Hold {
				t.Fatalf("Home: top %d", h.m.screen.Tree.Top)
			}
			if h.m.ui.ACursor != "" {
				t.Errorf("scrolling moved the cursor to %q", h.m.ui.ACursor)
			}
			busy(h, 60, 2, 0) // new stages while held
			h.m.View()
			if g := h.m.screen.Tree; g.Top != 0 || g.Rows <= v.Rows {
				t.Errorf("new stages moved the held tree: %+v", g)
			}
			h.draw(key(tea.KeyEnd, 0))
			if h.m.ui.Tree != (render.Pane{}) || h.m.screen.Tree.Top != h.m.screen.Tree.Rows-h.m.screen.Tree.Room {
				t.Errorf("End: %+v %+v, want following", h.m.ui.Tree, h.m.screen.Tree)
			}
			if f := h.frame(); strings.Contains(f, "End follows") {
				t.Errorf("hint still held:\n%s", f)
			}

			// The cursor: on the last node, then held at the top, ↑ shows it
			// with the least move.
			h.draw(key(tea.KeyUp, 0))
			cur := h.m.ui.ACursor
			h.draw(key(tea.KeyHome, 0))
			if h.m.ui.ACursor != cur {
				t.Fatalf("Home moved the cursor from %q to %q", cur, h.m.ui.ACursor)
			}
			h.draw(key(tea.KeyUp, 0))
			v = h.m.screen.Tree
			i := slices.Index(v.IDs, h.m.ui.ACursor)
			last := i // a folded stage's block is several rows of one node (plan 24): its last row at the bottom
			for last+1 < len(v.IDs) && v.IDs[last+1] == h.m.ui.ACursor {
				last++
			}
			if i < v.Room || v.Top != last-v.Room+1 || !h.m.ui.Tree.Hold {
				t.Errorf("↑ in a held tree: cursor rows %d-%d top %d, want the bottom of the view, top %d", i, last, v.Top, last-v.Room+1)
			}
			h.draw(key(tea.KeyDown, 0)) // the last row: the tail, following
			if h.m.ui.Tree != (render.Pane{}) {
				t.Errorf("↓ to the tail: %+v, want following", h.m.ui.Tree)
			}

			// A cursor above the tail: End shows the tail, held there.
			h.draw(key(tea.KeyHome, 0))
			for slices.Index(h.m.screen.Tree.IDs, h.m.ui.ACursor) >= h.m.screen.Tree.Rows-h.m.screen.Tree.Room {
				h.draw(key(tea.KeyUp, 0))
			}
			h.draw(key(tea.KeyEnd, 0))
			v = h.m.screen.Tree
			if v.Top != v.Rows-v.Room || slices.Index(v.IDs, h.m.ui.ACursor) >= v.Top {
				t.Errorf("End with the cursor above the tail: %+v cursor row %d", v, slices.Index(v.IDs, h.m.ui.ACursor))
			}
			busy(h, 62, 1, 0)
			h.m.View()
			if w := h.m.screen.Tree; w.Top != w.Rows-w.Room {
				t.Errorf("held at the tail, a new stage: top %d of %d rows, room %d", w.Top, w.Rows, w.Room)
			}
		})
	})

	// Criteria 2, 3, 5: focus C; End no longer resumes B's follow mode.
	t.Run("list", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			h := paneDash(t, 120, 40, 0)
			crowd(h, 30, 0)             // a list longer than C
			h.draw(key(tea.KeyPgUp, 0)) // B held
			if !h.m.ui.ConvHold {
				t.Skip("B fits: nothing to hold")
			}
			focusZone(t, h, "C")
			v := h.m.screen.List
			if v.Rows <= v.Room || v.Top != 0 {
				t.Fatalf("list %+v: want more agents than room, at its top", v)
			}
			bottom, page := v.Rows-v.Room, v.Room-1
			sel := h.m.ui.Selected
			h.draw(key(tea.KeyPgDown, 0))
			if g := h.m.screen.List.Top; g != page || !h.m.ui.List.Hold {
				t.Fatalf("PgDn: top %d, want %d held", g, page)
			}
			// In C's top border since plan 20, its hint row gone.
			if f := h.frame(); !strings.Contains(f, fmt.Sprintf("↑ %d above · ↓ %d below", page, bottom-page)) {
				t.Errorf("held hint missing:\n%s", f)
			}
			h.draw(key(tea.KeyEnd, 0))
			if g := h.m.screen.List.Top; g != bottom {
				t.Errorf("End: top %d, want %d", g, bottom)
			}
			if !h.m.ui.ConvHold {
				t.Errorf("End in C resumed B's follow mode")
			}
			if h.m.ui.Selected != sel {
				t.Errorf("scrolling changed the selection to %q", h.m.ui.Selected)
			}
			h.draw(key(tea.KeyHome, 0))
			if g := h.m.screen.List.Top; g != 0 {
				t.Errorf("Home: top %d", g)
			}
			if f := h.frame(); strings.Contains(f, "Home top") {
				t.Errorf("at the top, the hint still says held:\n%s", f)
			}
			h.draw(key(tea.KeyEnd, 0))
			h.draw(key(tea.KeyDown, 0)) // the first agent, above the view
			v = h.m.screen.List
			if i := slices.Index(v.IDs, h.m.ui.Selected); i != 0 || v.Top != 0 {
				t.Errorf("↓ to the first agent: row %d top %d, want 0 0", i, v.Top)
			}
			for range v.Room {
				h.draw(key(tea.KeyDown, 0))
			}
			v = h.m.screen.List
			if i := slices.Index(v.IDs, h.m.ui.Selected); i != v.Room || v.Top != 1 {
				t.Errorf("↓ past the view: row %d top %d, want %d 1", i, v.Top, v.Room)
			}
		})
	})

	// Criterion 4: the wheel over each pane, whatever has the focus; the
	// preview held as events arrive, following at its end or for another
	// agent.
	t.Run("wheel", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			h := paneDash(t, 170, 50, 0)
			crowd(h, 30, 30) // c00 says 30 messages, for the preview
			focusZone(t, h, "C")
			for range len(h.m.screen.List.IDs) {
				if h.m.ui.Selected == "c00" {
					break
				}
				h.draw(key(tea.KeyDown, 0))
			}
			h.draw(key(tea.KeyEscape, 0))
			if h.m.ui.Selected != "c00" || h.m.ui.Focus != "B" {
				t.Fatalf("selected %q focus %q", h.m.ui.Selected, h.m.ui.Focus)
			}
			for _, p := range []byte{'A', 'C', 'P'} {
				x, y := cellOf(t, h, p)
				ui, v := h.m.pane(p)
				if v.Rows <= v.Room {
					t.Fatalf("%c fits: %+v", p, *v)
				}
				from := v.Top
				wheel(h, x, y, p != 'C')
				_, v = h.m.pane(p)
				want := max(from-wheelRows, 0)
				if p == 'C' {
					want = min(from+wheelRows, v.Rows-v.Room)
				}
				if v.Top != want || !ui.Hold || h.m.ui.Focus != "B" {
					t.Errorf("wheel over %c: top %d hold %v focus %q, want %d held, B", p, v.Top, ui.Hold, h.m.ui.Focus, want)
				}
			}
			// A's and the preview's "End follows", C's "above" (plan 20).
			if f := h.frame(); strings.Count(f, "End follows") < 2 || !strings.Contains(f, " above") {
				t.Errorf("held hints:\n%s", f)
			}
			top := h.m.screen.Preview.Top
			// Events while held: a turn the CLI opens, with an agent of its
			// own, which leaves zone C's list in place (plan 18).
			feed(h, func(add func(k event.Kind, agent string, p any)) {
				add(event.TurnStarted, "", nil)
				agentRun(add, "c99", "crowd99", 0)
				add(event.TurnCompleted, "", event.TurnEnd{Outcome: "completed", DurationMS: 1000})
			})
			h.m.View()
			if h.m.screen.Preview.Top != top {
				t.Errorf("the held preview moved: %d, was %d", h.m.screen.Preview.Top, top)
			}
			x, y := cellOf(t, h, 'P')
			wheel(h, x, y, false)
			if h.m.ui.Preview != (render.Pane{}) {
				t.Errorf("the preview at its end: %+v, want following", h.m.ui.Preview)
			}
			wheel(h, x, y, true)
			focusZone(t, h, "C")
			h.draw(key(tea.KeyDown, 0))
			if h.m.ui.Selected == "c00" || h.m.ui.Preview != (render.Pane{}) {
				t.Errorf("another agent: selected %q preview %+v, want its tail", h.m.ui.Selected, h.m.ui.Preview)
			}
		})
	})

	// Criterion 6: zoomed A and C, and the 80x24 layout.
	t.Run("zoom and 80x24", func(t *testing.T) {
		for _, c := range []struct {
			name string
			w, h int
			zone string
			zoom bool
			p    byte
			hint string
		}{
			{"zoomed A", 120, 40, "A", true, 'A', "End follows"},
			{"zoomed C", 170, 50, "C", true, 'C', " above · "},
			{"80x24 A", 80, 24, "A", false, 'A', "End follows"},
			{"80x24 C", 80, 24, "C", false, 'C', ""},
		} {
			t.Run(c.name, func(t *testing.T) {
				synctest.Test(t, func(t *testing.T) {
					h := paneDash(t, c.w, c.h, 0)
					crowd(h, 60, 0) // a list longer than C, zoomed too
					focusZone(t, h, c.zone)
					if c.zoom {
						h.draw(typed('z'))
					}
					_, v := h.m.pane(c.p)
					if v.Rows <= v.Room {
						t.Fatalf("fits: %+v", *v)
					}
					from := v.Top
					step := tea.KeyPgUp
					if c.p == 'C' {
						step = tea.KeyPgDown
					}
					h.draw(key(step, 0))
					_, v = h.m.pane(c.p)
					want := max(from-max(v.Room-1, 1), 0) // a page, or to the end
					if c.p == 'C' {
						want = min(from+max(v.Room-1, 1), v.Rows-v.Room)
					}
					if v.Top != want || v.Top == from {
						t.Errorf("a page: top %d from %d, want %d (room %d)", v.Top, from, want, v.Room)
					}
					f := h.frame()
					if c.hint != "" && !strings.Contains(f, c.hint) {
						t.Errorf("hint %q missing:\n%s", c.hint, f)
					}
					if c.hint == "" && strings.Contains(f, "Home top") {
						t.Errorf("a three-row C has no hint row:\n%s", f)
					}
					checkFrame(t, f, c.w, c.h)
				})
			})
		}
	})
}
