package render

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/Evengelius/agent-command-center/internal/event"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// Plan 72's commands measured a frame at a time through one WrapCache, one
// Memo and the colours made, as the program draws (P5):
//
//	go test ./internal/render -run '^$' -bench Command -benchmem
//
// 200 commands in one turn, half of them long enough to fold, Codex's out of
// their launcher, half the long ones reviewed, a few failed; folded and
// unfolded, at 120x40 and 200x60.

// commandSession is a Codex session of n commands, every other one long.
func commandSession(tb testing.TB, n int) *convBench {
	b := newConvBench("codex", "gpt-5.6-sol")
	b.prompt("Collect the TODO and FIXME comments.")
	for i := range n {
		id := fmt.Sprintf("c%d", i)
		script := fmt.Sprintf("Get-ChildItem -LiteralPath dir%d -File | Measure-Object", i)
		if i%2 == 1 {
			script = fmt.Sprintf("%s; Write-Output %d; Get-Content -Raw -LiteralPath 'file%d.txt' | Select-String -Pattern 'TODO|FIXME' | Sort-Object LineNumber", todoScript, i, i)
		}
		b.ev(event.ToolStarted, event.Tool{ToolID: id, Name: "commandExecution", Input: codexItem(tb, id, script)})
		if i%4 == 1 {
			b.itemEv(event.ReviewUpdated, id, event.Review{ReviewID: "rv" + id, Status: "approved", Risk: "low", Rationale: "Reads the files of the project and writes TODO.md, a routine and reversible change."})
		}
		fail := i%10 == 7
		b.ev(event.ToolCompleted, event.Tool{ToolID: id, Name: "commandExecution", IsError: fail, Output: "done", ExitCode: map[bool]int{true: 1}[fail], HasExitCode: true})
	}
	return b
}

// colouredUI is a UI with caches, the session's colours made.
func colouredUI(b *convBench, w, h int) UI {
	ui := UI{Width: w, Height: h, Focus: "B", Wrap: NewWrapCache(), Memo: NewMemo(), Colors: NewCmdColors()}
	for _, x := range ui.Colors.Ask(CommandWants(b.s, 0, 0)) {
		ui.Colors.Put(ColourCommand(x))
	}
	return ui
}

// unfoldAll is every command's block unfolded.
func unfoldAll(s *state.State) map[string]bool {
	m := map[string]bool{}
	for id := range s.Tools {
		m[CommandKey(id)] = true
	}
	return m
}

func BenchmarkCommandFrames(b *testing.B) {
	s := commandSession(b, 200)
	for _, size := range [][2]int{{120, 40}, {200, 60}} {
		for _, open := range []bool{false, true} {
			name := fmt.Sprintf("%dx%d-folded", size[0], size[1])
			if open {
				name = fmt.Sprintf("%dx%d-unfolded", size[0], size[1])
			}
			b.Run(name, func(b *testing.B) {
				ui := colouredUI(s, size[0], size[1])
				if open {
					ui.Unfolded = unfoldAll(s.s)
				}
				Draw(s.s, ui)
				b.ReportAllocs()
				for b.Loop() {
					_ = Draw(s.s, ui).String()
				}
			})
		}
	}
}

// TestCommandCacheFrames is P1 and P2: through a session's commands coming,
// answered, reviewed, folded and unfolded, scrolled and resized, the frame
// drawn through the caches is the frame drawn afresh.
func TestCommandCacheFrames(t *testing.T) {
	wc, memo := NewWrapCache(), NewMemo()
	b := newConvBench("codex", "gpt-5.6-sol")
	b.prompt("go")
	widths := []int{120, 80, 200, 208}
	n := 0
	check := func(step string, ui UI) {
		t.Helper()
		n++
		ui.Width, ui.Height, ui.Focus = widths[n%len(widths)], 40, "B"
		if n%5 == 3 {
			ui.ConvHold, ui.ConvTop = true, n%30
		}
		cached := ui
		cached.Wrap, cached.Memo = wc, memo
		if got, want := Draw(b.s, cached).String(), Draw(b.s, ui).String(); got != want {
			t.Fatalf("%s: the cached frame differs:\n%s\nwant\n%s", step, got, want)
		}
	}
	for i := range 24 {
		id := fmt.Sprintf("c%d", i)
		script := strings.Repeat(fmt.Sprintf("Write-Output %d; ", i), 1+i%9)
		b.ev(event.ToolStarted, event.Tool{ToolID: id, Name: "commandExecution", Input: codexItem(t, id, script)})
		check("started "+id, UI{})
		if i%3 == 0 {
			req, _ := json.Marshal(map[string]any{"itemId": id, "command": psExe + " -Command " + posixQuote(script), "reason": "prints"})
			b.ev(event.ApprovalRequest, event.Approval{ID: "r" + id, Tool: "commandExecution", ToolID: id, Input: req})
			check("asked "+id, UI{})
			b.ev(event.ApprovalResolved, event.Approval{ID: "r" + id, Decision: []event.Decision{event.Allow, event.Deny, event.AllowSession}[i%3]})
			check("answered "+id, UI{})
		}
		if i%4 == 2 {
			b.itemEv(event.ReviewUpdated, id, event.Review{ReviewID: "rv" + id, Status: "approved", Risk: "medium", Authorization: "high", Rationale: strings.Repeat("a long rationale ", 1+i%5)})
			check("reviewed "+id, UI{})
		}
		b.ev(event.ToolCompleted, event.Tool{ToolID: id, Name: "commandExecution", IsError: i%5 == 4, Output: strings.Repeat("failure output ", 1+i%7), ExitCode: i % 2, HasExitCode: true})
		check("completed "+id, UI{})
		open := map[string]bool{}
		for k := 0; k <= i; k += 2 {
			open[CommandKey(fmt.Sprintf("c%d", k))] = true
		}
		check("unfolded "+id, UI{Unfolded: open})
	}
}

// TestCommandFrameAllocs is P5's guard, outside -race: a frame of the
// 200-command session drawn again, nothing changed, allocates within its
// bound, folded and unfolded.
func TestCommandFrameAllocs(t *testing.T) {
	if raceBuild {
		t.Skip("-race allocates on its own")
	}
	s := commandSession(t, 200)
	for _, open := range []bool{false, true} {
		ui := colouredUI(s, 120, 40)
		if open {
			ui.Unfolded = unfoldAll(s.s)
		}
		Draw(s.s, ui)
		n := testing.AllocsPerRun(20, func() { Draw(s.s, ui) })
		t.Logf("unfolded %v: %.0f allocations a frame", open, n)
		if n > commandFrameAllocs {
			t.Errorf("unfolded %v: %.0f allocations a frame, over %d", open, n, commandFrameAllocs)
		}
	}
}

// commandFrameAllocs is the bound of TestCommandFrameAllocs, set from its
// measure on 2026-10-31 with room for the frame's other parts to vary.
const commandFrameAllocs = 2500

// TestCommandRowsShown is P2 and criterion 72.18, outside -race: a frame
// draws the rows it shows, not the session's. 200 commands more add to a
// frame drawn again what each call costs zone A's workflow tree and zone
// B's runs, about 3.5 allocations a call, as before plan 72; never their
// blocks' rows, which would cost ten or more a call.
func TestCommandRowsShown(t *testing.T) {
	if raceBuild {
		t.Skip("-race allocates on its own")
	}
	frame := func(n int) float64 {
		s := commandSession(t, n)
		ui := colouredUI(s, 120, 40)
		ui.Unfolded = unfoldAll(s.s)
		Draw(s.s, ui)
		return testing.AllocsPerRun(20, func() { Draw(s.s, ui) })
	}
	a, b := frame(200), frame(400)
	t.Logf("200 commands unfolded: %.0f allocations a frame; 400: %.0f, %.2f a call more", a, b, (b-a)/200)
	if (b-a)/200 > 5 {
		t.Errorf("400 commands: %.0f allocations a frame, 200: %.0f; %.2f a call, over 5: rows not shown drawn", b, a, (b-a)/200)
	}
}
