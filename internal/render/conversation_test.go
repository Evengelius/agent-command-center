package render

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/Evengelius/agent-command-center/internal/fixture"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// Zone B draws only the rows in view and counts the others from the cache
// (test 11). The cache is a memo: across widths, scroll positions, new
// entries and an entry that changed, every frame and row count matches the
// frame drawn without it.
func TestConversationWindow(t *testing.T) {
	s := state.New()
	s.Screen = state.ScreenDashboard
	s.Context.Provider = "claude-code"
	at := fixture.Epoch
	add := func(role, text string, thinking bool) {
		s.Conversation = append(s.Conversation, state.Entry{Seq: uint64(len(s.Conversation) + 1), Role: role, Text: text, Thinking: thinking, At: at})
	}
	for i := 0; i < 40; i++ {
		add("user", fmt.Sprintf("question %d: %s", i, strings.Repeat("word ", 5*(i%7))), false)
		add("tool", fmt.Sprintf("Read file%d.go\nsecond line", i), false)
		add("assistant", fmt.Sprintf("answer %d\n%s\nend %d", i, strings.Repeat("long ", 12*(i%5)), i), i%4 == 0)
		if i%9 == 0 {
			add("error", fmt.Sprintf("turn failed: error %d\ndetails", i), false)
			add("notice", strings.Repeat("a progress note ", 1+i%6), false)
			add("user", strings.Repeat("pasted line\n", foldOver+3), false)
		}
	}
	wc := NewWrapCache()
	check := func(name string, ui UI) {
		t.Helper()
		cached := ui
		cached.Wrap = wc
		got, want := Draw(s, cached), Draw(s, ui)
		if want.Conv.Rows <= want.Conv.Room {
			t.Fatalf("%s: %d rows in %d: nothing to window", name, want.Conv.Rows, want.Conv.Room)
		}
		if got.String() != want.String() || got.Conv != want.Conv {
			t.Errorf("%s: the cached frame differs (rows %d, want %d):\n%s\nwant:\n%s", name, got.Conv.Rows, want.Conv.Rows, got, want)
		}
	}
	// Each round starts at the width the last ended on, so the counts it
	// finds were cached at that width.
	steps := func(label string) {
		for _, size := range [][2]int{{120, 40}, {200, 60}, {80, 24}, {100, 30}, {120, 40}} {
			ui := UI{Width: size[0], Height: size[1], Now: at.Add(time.Minute)}
			check(fmt.Sprintf("%s %dx%d, at the end", label, size[0], size[1]), ui)
			for _, top := range []int{0, 7, 150, 1 << 20} {
				ui.ConvHold, ui.ConvTop = true, top
				check(fmt.Sprintf("%s %dx%d, held at %d", label, size[0], size[1], top), ui)
			}
		}
	}
	steps("first")
	add("assistant", "a new answer\nstill streaming", false)
	s.Conversation[len(s.Conversation)-1].Streaming = true
	steps("new entries")
	s.Conversation[2].Text = "an answer that changed\n" + strings.Repeat("more ", 40)
	s.Conversation[8].Thinking = !s.Conversation[8].Thinking // the same text, one row more
	s.Conversation[len(s.Conversation)-1].Streaming = false
	steps("changed")
	s.Conversation = s.Conversation[10:] // the oldest dropped: every index moves
	steps("moved")
}
