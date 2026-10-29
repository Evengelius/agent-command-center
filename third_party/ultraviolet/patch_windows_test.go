//go:build windows

// This file is not upstream: it is agent-command-center's test of patches 4
// and 5 in PATCHES.md, the Windows reader's batches and grapheme state (plan
// 68). It drives the reader's real console path: each scenario runs in a
// child process the test starts with CREATE_NO_WINDOW, which has a console
// of its own and no window. The child writes key records into that
// console's input with WriteConsoleInputW, reads them back through
// conInputReader, streamData and StreamEvents, and prints the events it got
// for the parent to compare whole. It uses the standard library and the
// module's own Windows dependencies.

package uv

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
	"unicode/utf16"
	"unsafe"

	xwindows "github.com/charmbracelet/x/windows"
	"golang.org/x/sys/windows"
)

// consoleScenarioEnv names the scenario a child runs; the parent leaves it
// unset.
const consoleScenarioEnv = "UV_PATCH_CONSOLE_SCENARIO"

// consoleChildDeadline bounds a child's run: past it the child is killed and
// its test fails with what it wrote.
var consoleChildDeadline = 60 * time.Second

var (
	kernel32               = windows.NewLazySystemDLL("kernel32.dll")
	procWriteConsoleInputW = kernel32.NewProc("WriteConsoleInputW")
	procGetConsoleWindow   = kernel32.NewProc("GetConsoleWindow")
)

// The records the scenarios write: an emoji's two halves as Windows Terminal
// 1.23 sends them in win32-input-mode, virtual key 255 (PATCHES.md, Why), and
// Ctrl+F's four records in the two reads plan 67's trace recorded.
const (
	emojiHigh = "\x1b[255;0;55357;1;32;1_"
	emojiLow  = "\x1b[255;0;56898;1;32;1_"
	ctrlFDown = "\x1b[17;29;0;1;40;1_\x1b[70;0;6;1;40;1_"
	ctrlFUp   = "\x1b[70;33;6;0;40;1_\x1b[17;29;0;0;32;1_"
	emoji     = "\xf0\x9f\x99\x82" // U+1F642
	eAcute    = "\xc3\xa9"         // U+00E9
)

// TestConsoleReader is patches 4 and 5's test: each scenario's events, in
// order, none lost, doubled or moved. Run with -race, it is also their race
// check: the child's own race report fails it.
func TestConsoleReader(t *testing.T) {
	if s := os.Getenv(consoleScenarioEnv); s != "" {
		runConsoleScenario(t, s)
		return
	}
	burst := make([]string, 200)
	for i := range burst {
		burst[i] = "press " + string(rune('a'+i%26))
	}
	x, y, e := "press x", "press y", "press "+emoji
	for _, tc := range []struct {
		scenario string
		want     []string
	}{
		{"loss", []string{"press leftctrl", "press ctrl+f", "release ctrl+f", "release leftctrl"}},
		{"burst", burst},
		{"emoji in one read", []string{x, e, y}},
		{"emoji split between reads", []string{x, e, y}},
		{"emoji cut inside a sequence", []string{x, e, y}},
		{"emoji as surrogate records", []string{x, e, y}},
		{"characters without VT input", []string{"press a", "press " + eAcute, e, "press b"}},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			got := runConsoleChild(t, tc.scenario)
			if strings.Join(got, " | ") != strings.Join(tc.want, " | ") {
				t.Errorf("events:\n got %q\nwant %q", got, tc.want)
			}
		})
	}
}

// runConsoleChild runs scenario in a child process with a console of its own
// and no window, and returns the events it printed. A child that fails, runs
// past consoleChildDeadline or prints no events fails the test with all it
// wrote.
func runConsoleChild(t *testing.T, scenario string) []string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), consoleChildDeadline)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestConsoleReader$", "-test.count=1",
		"-test.timeout="+(consoleChildDeadline*9/10).String())
	cmd.Env = append(os.Environ(), consoleScenarioEnv+"="+scenario)
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: windows.CREATE_NO_WINDOW}
	cmd.WaitDelay = 5 * time.Second
	var out bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &out
	err := cmd.Run()
	switch {
	case ctx.Err() != nil:
		t.Fatalf("the child ran past its %v deadline and was killed; it wrote:\n%s", consoleChildDeadline, out.Bytes())
	case err != nil:
		t.Fatalf("the child failed (%v); it wrote:\n%s", err, out.Bytes())
	}
	for _, line := range strings.Split(out.String(), "\n") {
		if rest, ok := strings.CutPrefix(strings.TrimRight(line, "\r"), "events: "); ok {
			if rest == "" {
				return nil
			}
			return strings.Split(rest, " | ")
		}
	}
	t.Fatalf("the child printed no events; it wrote:\n%s", out.Bytes())
	return nil
}

// runConsoleScenario is the child's side: scenario's records written into
// its own console and read back through the real path. It prints "events: "
// and the events, separated by " | ".
func runConsoleScenario(t *testing.T, scenario string) {
	if hwnd, _, _ := procGetConsoleWindow.Call(); hwnd != 0 {
		t.Fatalf("the child's console has a window (0x%x)", hwnd)
	}
	var events []string
	switch scenario {
	case "loss":
		// The receiver held in its log of the first read, before its copy,
		// as TEA_TRACE's logging held it in plan 67's run, while the sender
		// reads the second read into its buffer.
		hold := &holdLogger{first: make(chan struct{}), release: make(chan struct{})}
		c := openTestConsole(t, true, hold, 0)
		t.Cleanup(hold.free) // the receiver let go before the reader is stopped
		c.write(vtRecords(ctrlFDown)...)
		hold.held(t)
		c.write(vtRecords(ctrlFUp)...)
		time.Sleep(50 * time.Millisecond) // the sender writes it into its buffer, then waits to send it
		hold.free()
		events = c.settle(4)
	case "burst":
		// One letter a read, the receiver held 20 ms in every fifth log:
		// longer than the 10 ms the sender sleeps when it finds no input, so
		// the sender reads the next letter while the receiver is held.
		c := openTestConsole(t, true, &holdLogger{every: 5, pause: 20 * time.Millisecond}, 0)
		for i := 0; i < 200; i++ {
			c.write(vtRecords(string(rune('a' + i%26)))...)
		}
		events = c.settle(200)
	case "emoji in one read":
		c := openTestConsole(t, true, nil, 0)
		c.write(vtRecords("x")...)
		c.write(vtRecords(emojiHigh + emojiLow)...)
		c.write(vtRecords("y")...)
		events = c.settle(3)
	case "emoji split between reads":
		c := openTestConsole(t, true, nil, 0)
		c.write(vtRecords("x")...)
		c.write(vtRecords(emojiHigh)...)
		c.write(vtRecords(emojiLow)...)
		c.write(vtRecords("y")...)
		events = c.settle(3)
	case "emoji cut inside a sequence":
		// The cut sequence waits for its end: a 2 s escape timeout in place
		// of 50 ms, so that a slow machine does not end it first.
		c := openTestConsole(t, true, nil, 2*time.Second)
		c.write(vtRecords("x")...)
		c.write(vtRecords(emojiHigh + emojiLow[:9])...)
		c.write(vtRecords(emojiLow[9:])...)
		c.write(vtRecords("y")...)
		events = c.settle(3)
	case "emoji as surrogate records":
		// The emoji's halves as two key records without a virtual key, in
		// two reads: the sender joins them (its UTF-16 halves).
		c := openTestConsole(t, true, nil, 0)
		c.write(vtRecords("x")...)
		c.write(keyRecord(0, 0, 0xD83D, 0))
		c.write(keyRecord(0, 0, 0xDE42, 0))
		c.write(vtRecords("y")...)
		events = c.settle(3)
	case "characters without VT input":
		// VT input mode off, the sender's other branch: a and b with their
		// virtual keys, é and the emoji without one, the emoji's halves in
		// two reads (the sender's grapheme state, patch 5).
		c := openTestConsole(t, false, nil, 0)
		c.write(keyRecord(0x41, 0x1E, 'a', 0))
		c.write(keyRecord(0, 0, 0xE9, 0), keyRecord(0, 0, 0xD83D, 0))
		c.write(keyRecord(0, 0, 0xDE42, 0), keyRecord(0x42, 0x30, 'b', 0))
		events = c.settle(4)
	default:
		t.Fatalf("no scenario %q", scenario)
	}
	fmt.Printf("events: %s\n", strings.Join(events, " | "))
}

// holdLogger holds the receiver in its log of a read, which comes before its
// copy of the read: the first read until free, or every every-th read for
// pause.
type holdLogger struct {
	mu      sync.Mutex
	reads   int
	first   chan struct{} // closed once the first read is held
	release chan struct{}
	once    sync.Once
	every   int
	pause   time.Duration
}

func (l *holdLogger) Printf(format string, _ ...interface{}) {
	if format != "input: %q" {
		return
	}
	l.mu.Lock()
	l.reads++
	n := l.reads
	l.mu.Unlock()
	switch {
	case l.release != nil && n == 1:
		close(l.first)
		<-l.release
	case l.every > 0 && n%l.every == 0:
		time.Sleep(l.pause)
	}
}

// held waits until the first read is held.
func (l *holdLogger) held(t *testing.T) {
	t.Helper()
	select {
	case <-l.first:
	case <-time.After(5 * time.Second):
		t.Fatal("the receiver logged no read within 5 s")
	}
}

// free lets the held receiver go on; again, it does nothing.
func (l *holdLogger) free() {
	if l.release != nil {
		l.once.Do(func() { close(l.release) })
	}
}

// testConsole is the child's console and the reader on it.
type testConsole struct {
	t      *testing.T
	h      windows.Handle
	mu     sync.Mutex
	events []string
}

// openTestConsole opens the child's console input in the modes
// NewCancelReader sets, VT input mode on or off, and starts a reader on it
// with lg as its logger and escTimeout as its escape timeout, 0 for the
// default. The test's cleanup stops the reader and waits for it.
func openTestConsole(t *testing.T, vt bool, lg Logger, escTimeout time.Duration) *testConsole {
	t.Helper()
	name, err := windows.UTF16PtrFromString("CONIN$")
	if err != nil {
		t.Fatal(err)
	}
	h, err := windows.CreateFile(name, windows.GENERIC_READ|windows.GENERIC_WRITE,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE, nil, windows.OPEN_EXISTING, 0, 0)
	if err != nil {
		t.Fatalf("CONIN$: %v", err)
	}
	t.Cleanup(func() { _ = windows.CloseHandle(h) })
	mode := uint32(windows.ENABLE_WINDOW_INPUT | windows.ENABLE_EXTENDED_FLAGS)
	if vt {
		mode |= windows.ENABLE_VIRTUAL_TERMINAL_INPUT
	}
	if err := windows.SetConsoleMode(h, mode); err != nil {
		t.Fatalf("SetConsoleMode: %v", err)
	}
	if err := xwindows.FlushConsoleInputBuffer(h); err != nil {
		t.Fatalf("FlushConsoleInputBuffer: %v", err)
	}
	cr := &conInputReader{conin: h, newMode: mode}
	r := NewTerminalReader(cr, "xterm-256color")
	if escTimeout > 0 {
		r.EscTimeout = escTimeout
	}
	if lg != nil {
		r.SetLogger(lg)
	}
	c := &testConsole{t: t, h: h}
	ctx, cancel := context.WithCancel(context.Background())
	evc := make(chan Event, 1024)
	drained := make(chan struct{})
	go func() {
		for e := range evc {
			if s, ok := consoleEvent(e); ok {
				c.mu.Lock()
				c.events = append(c.events, s)
				c.mu.Unlock()
			}
		}
		close(drained)
	}()
	errc := make(chan error, 1)
	go func() { errc <- r.StreamEvents(ctx, evc) }()
	t.Cleanup(func() {
		cancel()
		cr.Cancel()
		select {
		case err := <-errc:
			if err != nil {
				t.Errorf("the reader stopped with %v", err)
			}
		case <-time.After(5 * time.Second):
			t.Errorf("the reader did not stop within 5 s")
			return
		}
		close(evc)
		<-drained
	})
	return c
}

// consoleEvent is e as the parent compares it: a key's press or release by
// its name, an unknown sequence quoted. The reader's other events (a window
// size, a focus) are not this file's subject and are left out.
func consoleEvent(e Event) (string, bool) {
	switch e := e.(type) {
	case KeyPressEvent:
		return "press " + e.String(), true
	case KeyReleaseEvent:
		return "release " + e.String(), true
	case UnknownEvent:
		return fmt.Sprintf("unknown %q", string(e)), true
	}
	return "", false
}

// keyRecord is a key's press as an INPUT_RECORD, laid out as
// xwindows.InputRecord.KeyEvent reads it.
func keyRecord(vk, sc, char uint16, cks uint32) xwindows.InputRecord {
	var r xwindows.InputRecord
	r.EventType = xwindows.KEY_EVENT
	binary.LittleEndian.PutUint32(r.Event[0:4], 1) // key down
	binary.LittleEndian.PutUint16(r.Event[4:6], 1) // repeat count
	binary.LittleEndian.PutUint16(r.Event[6:8], vk)
	binary.LittleEndian.PutUint16(r.Event[8:10], sc)
	binary.LittleEndian.PutUint16(r.Event[10:12], char)
	binary.LittleEndian.PutUint32(r.Event[12:16], cks)
	return r
}

// vtRecords is s as a console in VT input mode gives it: a key press for
// each UTF-16 unit, with no virtual key.
func vtRecords(s string) []xwindows.InputRecord {
	var recs []xwindows.InputRecord
	for _, u := range utf16.Encode([]rune(s)) {
		recs = append(recs, keyRecord(0, 0, u, 0))
	}
	return recs
}

// write puts recs into the console's input, then waits until the reader has
// read them, so that the next write is a read of its own.
func (c *testConsole) write(recs ...xwindows.InputRecord) {
	c.t.Helper()
	var n uint32
	r1, _, err := procWriteConsoleInputW.Call(uintptr(c.h), uintptr(unsafe.Pointer(&recs[0])),
		uintptr(len(recs)), uintptr(unsafe.Pointer(&n)))
	if r1 == 0 {
		c.t.Fatalf("WriteConsoleInputW: %v", err)
	}
	if int(n) != len(recs) {
		c.t.Fatalf("WriteConsoleInputW wrote %d of %d records", n, len(recs))
	}
	deadline := time.Now().Add(5 * time.Second)
	for {
		var left uint32
		if err := xwindows.GetNumberOfConsoleInputEvents(c.h, &left); err != nil {
			c.t.Fatalf("GetNumberOfConsoleInputEvents: %v", err)
		}
		if left == 0 {
			return
		}
		if time.Now().After(deadline) {
			c.t.Fatalf("the reader left %d records unread for 5 s", left)
		}
		time.Sleep(time.Millisecond)
	}
}

// settle waits until n events have come, or 10 s, then 300 ms more, six
// escape timeouts, for any further one (a key doubled), and returns them all.
func (c *testConsole) settle(n int) []string {
	deadline := time.Now().Add(10 * time.Second)
	for {
		c.mu.Lock()
		got := len(c.events)
		c.mu.Unlock()
		if got >= n || time.Now().After(deadline) {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	time.Sleep(300 * time.Millisecond)
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]string(nil), c.events...)
}
