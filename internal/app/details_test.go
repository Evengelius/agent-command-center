package app

import (
	"fmt"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/Evengelius/agent-command-center/internal/event"
	"github.com/Evengelius/agent-command-center/internal/render"
)

// Plan 4, step 7, criterion 5: Enter on zone D lists the last errors, newest
// first, and scrolls; Esc returns to D.

// addErrors sends the errors the details tests show: one of the session, one
// retry of the first agent, one long and on two lines, then n numbered ones.
func addErrors(h *harness, n int) {
	s := h.m.State()
	at, seq := s.Log[len(s.Log)-1].At, s.Log[len(s.Log)-1].Seq
	agent := render.AgentOrder(s)[0]
	var evs []event.Event
	add := func(k event.Kind, agent, msg string) {
		seq++
		evs = append(evs, event.Event{Seq: seq, At: at.Add(time.Duration(len(evs)+1) * time.Second), Provider: s.CLI(), Kind: k, Origin: event.Observed, Agent: agent, Payload: event.Problem{Message: msg}})
	}
	add(event.Error, "", "get_context_usage refused by Claude Code: Unsupported control request subtype: get_context_usage")
	add(event.Retry, agent, "API Error: 529 Overloaded; retrying in 2s (attempt 1 of 10)")
	add(event.Error, "", "turn failed: API Error: 400 the request was rejected because the prompt, together with the files it names and the tool results already in the context, is longer than the model accepts\nrequest id req_011CgExample")
	for i := 1; i <= n; i++ {
		add(event.Error, "", fmt.Sprintf("numbered error %d", i))
	}
	h.send(eventsMsg{sess: h.m.sess, evs: evs, closed: true})
}

// openDetailsView focuses zone D and presses Enter.
func openDetailsView(t *testing.T, h *harness) {
	t.Helper()
	for i := 0; i < len(zones) && h.m.ui.Focus != "D"; i++ {
		h.draw(key(tea.KeyF6, 0))
	}
	if h.m.ui.Focus != "D" {
		t.Fatalf("focus %q", h.m.ui.Focus)
	}
	if f := h.frame(); !strings.Contains(f, "Enter details") {
		t.Fatalf("D focused without its hint:\n%s", f)
	}
	h.draw(key(tea.KeyEnter, 0))
	if !h.m.ui.Details {
		t.Fatal("Enter on D did not open the details")
	}
}

func TestDetailsKeys(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := mockDash(t, "claude-subagent-approval.jsonl")
		h.draw(tea.WindowSizeMsg{Width: 80, Height: 24})
		addErrors(h, 20)
		openDetailsView(t, h)

		f := h.frame()
		if !strings.Contains(f, "D · ERRORS AND TELEMETRY") || !strings.Contains(f, "23 errors · last 50 kept") || strings.Contains(f, "B · CONVERSATION") {
			t.Fatalf("details frame:\n%s", f)
		}
		if i, j := strings.Index(f, "numbered error 20"), strings.Index(f, "numbered error 19"); i < 0 || j < i {
			t.Errorf("newest not first:\n%s", f)
		}
		v := h.m.screen.Inspect
		if v.Top != 0 || !h.m.ui.InspectHold || v.Rows <= v.Room {
			t.Fatalf("opened at top %d hold %v, %d rows in %d", v.Top, h.m.ui.InspectHold, v.Rows, v.Room)
		}
		bottom := v.Rows - v.Room

		for _, c := range []struct {
			name string
			msg  tea.Msg
			top  int
		}{
			{"↓", key(tea.KeyDown, 0), 1},
			{"↑", key(tea.KeyUp, 0), 0},
			{"PgDn", key(tea.KeyPgDown, 0), v.Room - 1},
			{"PgUp", key(tea.KeyPgUp, 0), 0},
			{"End", key(tea.KeyEnd, 0), bottom},
			{"Home", key(tea.KeyHome, 0), 0},
			{"wheel", tea.MouseWheelMsg{X: 10, Y: v.Y + 2, Button: tea.MouseWheelDown}, wheelRows},
		} {
			h.draw(c.msg)
			if got := h.m.screen.Inspect.Top; got != c.top {
				t.Errorf("%s: top %d, want %d", c.name, got, c.top)
			}
		}
		if h.m.ui.ConvHold {
			t.Error("the details' scrolling held the conversation")
		}

		// It is the state rendered: a new error shows while it is open.
		h.draw(key(tea.KeyHome, 0))
		addErrors(h, 0)
		if f := h.frame(); !strings.Contains(f, "26 errors") || !strings.Contains(f, "numbered error") && !strings.Contains(f, "turn failed: API Error: 400") {
			t.Errorf("new errors not shown:\n%s", f)
		}

		// Esc clears a selection first, then returns to D, still focused.
		h.m.ui.Sel = render.Selection{From: render.Pos{Panel: 'I', Item: 1}, To: render.Pos{Panel: 'I', Item: 1, Off: 4}}
		h.draw(key(tea.KeyEscape, 0))
		if !h.m.ui.Sel.Empty() || !h.m.ui.Details {
			t.Errorf("first Esc: selection %+v, details %v", h.m.ui.Sel, h.m.ui.Details)
		}
		h.draw(key(tea.KeyEscape, 0))
		if h.m.ui.Details || h.m.ui.Focus != "D" || h.m.ui.InspectHold {
			t.Errorf("second Esc: details %v, focus %q, hold %v", h.m.ui.Details, h.m.ui.Focus, h.m.ui.InspectHold)
		}
		if f := h.frame(); !strings.Contains(f, "Enter details") || !strings.Contains(f, "B · CONVERSATION") {
			t.Errorf("not back on the dashboard:\n%s", f)
		}
	})
}
