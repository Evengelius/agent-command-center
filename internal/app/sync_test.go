package app

import (
	"bytes"
	"io"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/Evengelius/agent-command-center/internal/state"
)

// wtModeReport is Windows Terminal 1.23's answer to Bubble Tea's query for
// modes 2026 and 2027: synchronized output supported and off, Unicode core
// permanently set (plan 5, item 14, recorded with TEA_TRACE on 2026-10-17).
const wtModeReport = "\x1b[?2026;2$y\x1b[?2027;3$y"

const (
	syncQuery = "\x1b[?2026$p"
	syncBegin = "\x1b[?2026h"
	syncEnd   = "\x1b[?2026l"
)

// escapeSeq is a CSI, an OSC or another two-byte escape sequence.
var escapeSeq = regexp.MustCompile("\x1b\\[[0-?]*[ -/]*[@-~]|\x1b\\][^\x07\x1b]*(\x07|\x1b\\\\)|\x1b[ -~]")

// drawnOutside is what out draws outside synchronized output once the first
// frame is wrapped, escape sequences and cursor moves aside, and the number
// of wrapped writes; a "?2026h" left open or nested is an error.
func drawnOutside(out string) (drawn string, pairs int, bad string) {
	i := strings.Index(out, syncBegin)
	if i < 0 {
		return "", 0, ""
	}
	var outside strings.Builder
	for rest := out[i:]; rest != ""; {
		b := strings.Index(rest, syncBegin)
		if b < 0 {
			outside.WriteString(rest)
			break
		}
		outside.WriteString(rest[:b])
		rest = rest[b+len(syncBegin):]
		e := strings.Index(rest, syncEnd)
		if e < 0 {
			return "", pairs, "a frame left open"
		}
		if strings.Contains(rest[:e], syncBegin) {
			return "", pairs, "a frame opened twice"
		}
		pairs++
		rest = rest[e+len(syncEnd):]
	}
	for _, r := range escapeSeq.ReplaceAllString(outside.String(), "") {
		if r >= 0x20 {
			drawn += string(r)
		}
	}
	return drawn, pairs, ""
}

// runSynced runs the application as Windows Terminal starts it, the
// terminal answering report, then types three characters and quits; it
// returns everything written and the model.
func runSynced(t *testing.T, report string) (string, *Model) {
	in, keys := io.Pipe()
	typed := false
	updateHook = func(m *Model, _ tea.Msg) []tea.Cmd {
		if !typed && m.s.Screen == state.ScreenDashboard && m.s.Session.Phase == state.Ready {
			typed = true
			go func() {
				for _, k := range []string{"a", "b", "c", wtCtrlQ} {
					time.Sleep(50 * time.Millisecond) // a frame each
					_, _ = io.WriteString(keys, k)
				}
			}()
		}
		return nil
	}
	resetTerminal = func(string) {}
	t.Cleanup(func() { resetTerminal, updateHook = consoleReset, nil })
	go func() { _, _ = io.WriteString(keys, report) }()

	m := New(Options{
		Config: Config{Mock: filepath.Join("..", "..", "testdata", "mock", "claude-subagent-approval.jsonl")},
		Cwd:    t.TempDir(),
		Now:    time.Now,
	})
	var out bytes.Buffer
	done := make(chan int, 1)
	go func() {
		code, _ := Run(m, tea.WithInput(in), tea.WithOutput(&out), tea.WithWindowSize(120, 40), tea.WithoutSignals(),
			tea.WithEnvironment([]string{"TTY_FORCE=1", "WT_SESSION=1"}))
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
	return out.String(), m
}

// TestSynchronizedOutput is plan 5, item 14, criteria 2 and 3: under Windows
// Terminal, Bubble Tea asks for mode 2026; once the terminal reports it
// supported, as Windows Terminal 1.23 does, every frame is drawn inside
// "?2026h … ?2026l", the report decoded by the same input reader as keys;
// with no report or another answer, no frame is wrapped.
func TestSynchronizedOutput(t *testing.T) {
	defer trueColour()
	for _, c := range []struct {
		name, report string
		wrapped      bool
	}{
		{"Windows Terminal", wtModeReport, true},
		{"no answer", "", false},
		{"not recognised", "\x1b[?2026;0$y", false},
		{"permanently reset", "\x1b[?2026;4$y", false},
	} {
		t.Run(c.name, func(t *testing.T) {
			out, m := runSynced(t, c.report)
			if !strings.Contains(out, syncQuery) {
				t.Fatal("Bubble Tea did not ask the terminal for mode 2026")
			}
			if m.edit.Text() != "abc" || !strings.Contains(out, "A · ORCHESTRATOR") {
				t.Fatalf("typed %q; the dashboard and every key must be handled", m.edit.Text())
			}
			drawn, pairs, bad := drawnOutside(out)
			switch {
			case bad != "":
				t.Errorf("%s:\n%q", bad, out)
			case !c.wrapped && (pairs != 0 || strings.Contains(out, syncEnd)):
				t.Errorf("%d frames wrapped with no report that 2026 is supported", pairs)
			case c.wrapped && pairs < 4:
				t.Errorf("%d frames wrapped, want the redraw and one per key", pairs)
			case c.wrapped && drawn != "":
				t.Errorf("drawn outside synchronized output: %q", drawn)
			}
		})
	}
}
