package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/Evengelius/agent-command-center/internal/render"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// lockedBuffer is a program's output that the test reads while it runs.
type lockedBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *lockedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *lockedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// wideMock writes a mock into dir: the thinking mock's start, its answer 70
// lines that each begin with a check mark, a cell two columns wide, and the
// turn's end when done is set.
func wideMock(t *testing.T, dir string, done bool) string {
	t.Helper()
	src, err := os.ReadFile(filepath.Join("..", "..", "testdata", "mock", "claude-thinking.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	var recs []map[string]any
	for _, l := range strings.Split(strings.TrimSpace(string(src)), "\n") {
		var r map[string]any
		if err := json.Unmarshal([]byte(l), &r); err != nil {
			t.Fatal(err)
		}
		recs = append(recs, r)
	}
	payload := func(r map[string]any) map[string]any { p, _ := r["payload"].(map[string]any); return p }
	start := -1
	for i, r := range recs {
		if p := payload(r); r["kind"] == "conversation.message.marker" && p["Name"] == "content_block_start" && p["Block"] == "text" {
			start = i
			break
		}
	}
	if start < 0 {
		t.Fatal("no text block in the thinking mock")
	}
	msg, session, at := payload(recs[start])["MessageID"], recs[start]["session"], recs[start]["t_ms"].(float64)
	out := append([]map[string]any{}, recs[:start+1]...)
	var text strings.Builder
	for i := 1; i <= 70; i++ {
		d := fmt.Sprintf("- ✅ item %d is done and checked\n", i)
		text.WriteString(d)
		out = append(out, map[string]any{"t_ms": at + float64(i), "kind": "conversation.message.delta", "provider": "claude-code",
			"session": session, "type": "Delta", "payload": map[string]any{"MessageID": msg, "Index": 1, "Block": "text", "Text": d}})
	}
	if done {
		for _, r := range recs[start+1:] {
			if r["kind"] == "turn.completed" {
				out = append(out, r)
				break
			}
			if p := payload(r); p["MessageID"] == msg && r["kind"] != "conversation.message.delta" {
				if r["kind"] == "conversation.message.completed" {
					p["Text"] = text.String()
				}
				out = append(out, r)
			}
		}
	}
	var b strings.Builder
	for _, r := range out {
		l, err := json.Marshal(r)
		if err != nil {
			t.Fatal(err)
		}
		b.Write(l)
		b.WriteByte('\n')
	}
	file := filepath.Join(dir, "wide.jsonl")
	if err := os.WriteFile(file, []byte(b.String()), 0o600); err != nil {
		t.Fatal(err)
	}
	return file
}

// TestWideRowsKept is plan 43's review (third_party/ultraviolet/PATCHES.md,
// patch 3): the rows beside the sidebar's tiles hold wide cells, which the
// renderer repaints rather than diffs, and a repaint over the tiles' blank
// cells would erase their images until a frame changed those rows again. The
// real program, as Windows Terminal starts it with Sixel, at 208x59, on a
// mock whose answer is a list of check marks in zone B beside both tiles: a
// key typed once the turn is over, and the spinner's frames while the turn
// runs, write none of those rows' wide cells.
func TestWideRowsKept(t *testing.T) {
	defer trueColour()
	for _, c := range []struct {
		name string
		done bool
	}{{"a key, the turn over", true}, {"the spinner, the turn running", false}} {
		t.Run(c.name, func(t *testing.T) {
			dir := t.TempDir()
			in, keys := io.Pipe()
			ready := make(chan []string, 1)
			updateHook = func(m *Model, _ tea.Msg) []tea.Cmd {
				conv := m.s.Conversation
				if m.screen == nil || len(conv) == 0 || !strings.Contains(conv[len(conv)-1].Text, "item 70 is done") ||
					c.done != (m.s.Turn.Phase == state.Idle) || !m.screen.Tiles[1].Shown {
					return nil
				}
				var rows []string
				for _, t := range m.screen.Tiles {
					for y := t.Y; y < t.Y+render.SideTileRows; y++ {
						rows = append(rows, m.screen.Row(y))
					}
				}
				select {
				case ready <- rows:
				default:
				}
				return nil
			}
			resetTerminal = func(string) {}
			t.Cleanup(func() { resetTerminal, updateHook = consoleReset, nil })
			go func() { _, _ = io.WriteString(keys, wtModeReport+"\x1b[?61;4c") }() // and Sixel (plan 32)

			m := New(Options{Config: Config{Mock: wideMock(t, dir, c.done)}, Cwd: dir, Now: time.Now})
			var out lockedBuffer
			done := make(chan int, 1)
			go func() {
				code, _ := Run(m, tea.WithInput(in), tea.WithOutput(&out), tea.WithWindowSize(208, 59), tea.WithoutSignals(),
					tea.WithEnvironment([]string{"TTY_FORCE=1", "WT_SESSION=1"}))
				done <- code
			}()
			var rows []string
			select {
			case rows = <-ready:
			case <-time.After(20 * time.Second):
				t.Fatal("the mock's answer was not shown")
			}
			for i, r := range rows {
				if !strings.Contains(r, "✅") {
					t.Fatalf("tile row %d holds no wide cell, the test proves nothing: %q", i, r)
				}
			}
			time.Sleep(600 * time.Millisecond) // the tiles drawn, twice (plan 32, design D5)
			from := len(out.String())
			if c.done {
				_, _ = io.WriteString(keys, "a")
			}
			time.Sleep(1200 * time.Millisecond)
			seg := out.String()[from:]
			_, _ = io.WriteString(keys, wtCtrlQ)
			time.Sleep(300 * time.Millisecond)
			_, _ = io.WriteString(keys, wtCtrlQ) // a running turn asks first
			select {
			case code := <-done:
				if code != ExitOK {
					t.Fatalf("exit code %d", code)
				}
			case <-time.After(20 * time.Second):
				t.Fatal("the program did not end on Ctrl+Q")
			}
			out0 := out.String()[:from]
			if n := strings.Count(out0, "\x1bP"); n < 2 {
				t.Fatalf("%d Sixel images drawn before the frames checked; want the two tiles", n)
			}
			frames := strings.Count(seg, syncBegin)
			switch {
			case c.done && m.edit.Text() != "a":
				t.Errorf("draft %q, want the key typed", m.edit.Text())
			case !c.done && frames < 5:
				t.Errorf("%d frames in 1.2 s of a running turn, want the spinner's", frames)
			case strings.Contains(seg, "✅"):
				t.Errorf("%d frames wrote the rows beside the tiles again, erasing them:\n%q", frames, seg)
			}
		})
	}
}
