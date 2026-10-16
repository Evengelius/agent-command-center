package app

import (
	"io"
	"path/filepath"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/Evengelius/agent-command-center/internal/event"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// TestWin32InputMode: on Windows, Init asks the console for win32-input-mode
// (ui-design.md, Shift+Enter) and Run resets it once the program has ended,
// whether it quit or panicked; elsewhere nothing is asked or reset.
func TestWin32InputMode(t *testing.T) {
	m := New(Options{Cwd: t.TempDir()})
	asked := 0
	for _, c := range batch(m.Init()) {
		if raw, ok := c().(tea.RawMsg); ok && raw.Msg == win32InputOn {
			asked++
		}
	}
	want := 0
	if win32Input {
		want = 1
	}
	if asked != want {
		t.Errorf("Init asked for win32-input-mode %d times, want %d", asked, want)
	}

	for _, tc := range []struct {
		name string
		hook func() []tea.Cmd
		code int
	}{
		{"quit", func() []tea.Cmd { return []tea.Cmd{tea.Quit} }, ExitOK},
		{"panic", func() []tea.Cmd { panic("injected panic") }, ExitPanic},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var resets []string
			resetTerminal = func(seq string) { resets = append(resets, seq) }
			updateHook = func(*Model, tea.Msg) []tea.Cmd { return tc.hook() }
			t.Cleanup(func() { resetTerminal, updateHook = consoleReset, nil })

			m := New(Options{Cwd: t.TempDir()})
			code, _ := Run(m, tea.WithInput(nil), tea.WithOutput(io.Discard), tea.WithWindowSize(120, 40), tea.WithoutSignals())
			if code != tc.code {
				t.Errorf("exit code %d, want %d", code, tc.code)
			}
			if win32Input && (len(resets) != 1 || resets[0] != win32InputOff) {
				t.Errorf("resets %q, want one %q", resets, win32InputOff)
			}
			if !win32Input && len(resets) != 0 {
				t.Errorf("resets %q, want none", resets)
			}
		})
	}
}

// batch flattens a command into the commands a tea.Batch holds.
func batch(c tea.Cmd) []tea.Cmd {
	if c == nil {
		return nil
	}
	msg := c()
	if b, ok := msg.(tea.BatchMsg); ok {
		var out []tea.Cmd
		for _, c := range b {
			out = append(out, batch(c)...)
		}
		return out
	}
	return []tea.Cmd{func() tea.Msg { return msg }}
}

// Keys as Windows Terminal 1.23 sends them in win32-input-mode
// (ESC [ Vk ; Sc ; Uc ; Kd ; Cs ; Rc _), recorded from the application's
// input trace (TEA_TRACE) on 2026-10-16 while SendInput typed them; Cs 32 is
// Num Lock. capsShiftZ is built from the same format, not recorded.
const (
	wtZUnicode  = "\x1b[90;0;90;1;32;1_\x1b[90;17;122;0;32;1_"                                        // Z typed as a character, Shift up
	wtEmoji     = "\x1b[255;0;55357;1;32;1_\x1b[255;0;56898;1;32;1_"                                  // U+1F642 as two surrogates
	wtEAcute    = "\x1b[50;0;233;1;32;1_\x1b[50;3;233;0;32;1_"                                        // é
	wtShiftZ    = "\x1b[16;42;0;1;48;1_\x1b[90;0;90;1;48;1_\x1b[90;17;90;0;48;1_\x1b[16;42;0;0;32;1_" // Shift+Z
	wtShiftDown = "\x1b[16;42;0;1;48;1_"                                                              // Shift pressed
	wtEnterUp   = "\x1b[13;0;13;1;48;1_\x1b[13;28;13;0;48;1_\x1b[16;42;0;0;32;1_"                     // Enter, then Shift released
	wtEnter     = "\x1b[13;28;13;1;32;1_\x1b[13;28;13;0;32;1_"                                        // Enter alone
	wtCtrlQ     = "\x1b[17;29;0;1;40;1_\x1b[81;0;17;1;40;1_\x1b[81;30;17;0;40;1_\x1b[17;29;0;0;32;1_" // Ctrl+Q
	capsShiftZ  = "\x1b[16;42;0;1;176;1_\x1b[90;44;122;1;176;1_\x1b[90;44;122;0;176;1_\x1b[16;42;0;0;160;1_"
)

// TestWin32InputKeys drives the real program with win32-input-mode keys: the
// character the console reports is the one inserted (third_party/ultraviolet,
// PATCHES.md), Shift+Enter inserts a newline and Enter sends.
func TestWin32InputKeys(t *testing.T) {
	in, keys := io.Pipe()
	typed := false
	updateHook = func(m *Model, _ tea.Msg) []tea.Cmd {
		if !typed && m.s.Screen == state.ScreenDashboard && m.s.Session.Phase == state.Ready {
			typed = true
			go func() {
				_, _ = io.WriteString(keys, wtZUnicode+wtEmoji+wtEAcute+wtShiftZ+capsShiftZ+wtShiftDown+wtEnterUp+wtEAcute)
				time.Sleep(200 * time.Millisecond) // the newline is drawn before Enter
				_, _ = io.WriteString(keys, wtEnter)
				time.Sleep(200 * time.Millisecond)
				_, _ = io.WriteString(keys, wtCtrlQ)
			}()
		}
		return nil
	}
	resetTerminal = func(string) {}
	t.Cleanup(func() { resetTerminal, updateHook = consoleReset, nil })

	m := New(Options{
		Config:    Config{Mock: filepath.Join("..", "..", "testdata", "mock", "claude-subagent-approval.jsonl")},
		Cwd:       t.TempDir(),
		Now:       time.Now,
		MockSpeed: 0,
	})
	done := make(chan int, 1)
	go func() {
		code, _ := Run(m, tea.WithInput(in), tea.WithOutput(io.Discard), tea.WithWindowSize(120, 40), tea.WithoutSignals())
		done <- code
	}()
	select {
	case code := <-done:
		if code != ExitOK {
			t.Fatalf("exit code %d", code)
		}
	case <-time.After(20 * time.Second):
		t.Fatal("the program did not end on Ctrl+Q")
	}
	var sent []string
	for _, ev := range m.s.Log {
		if u, ok := ev.Payload.(event.User); ok {
			sent = append(sent, u.Text)
		}
	}
	want := "Z\U0001F642éZz\né"
	// The replay holds one user message of its own; the keys make the second.
	if len(sent) != 2 || sent[1] != want {
		t.Errorf("sent %q, want the replay's message and %q", sent, want)
	}
}
