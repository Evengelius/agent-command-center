package input

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"golang.design/x/clipboard"
)

// Clipboard is the system clipboard as text. Tests replace it.
type Clipboard interface {
	Read() (string, error)
	Write(text string) error
}

// System is the system clipboard (golang.design/x/clipboard; on Windows the
// Win32 clipboard, no cgo).
type System struct{}

// clipboardTimeout bounds a clipboard call, so a clipboard held by another
// program cannot freeze the screen.
const clipboardTimeout = 2 * time.Second

// Read returns the clipboard's text, byte for byte.
func (System) Read() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), clipboardTimeout)
	defer cancel()
	b, err := clipboard.Read(ctx, clipboard.FmtText)
	return string(b), err
}

// Write puts text on the clipboard.
func (System) Write(text string) error {
	ctx, cancel := context.WithTimeout(context.Background(), clipboardTimeout)
	defer cancel()
	_, err := clipboard.Write(ctx, clipboard.FmtText, []byte(text))
	return err
}

// OSC52 is the escape sequence that asks the terminal to set its clipboard.
func OSC52(text string) string {
	return "\x1b]52;c;" + base64.StdEncoding.EncodeToString([]byte(text)) + "\x07"
}

// TmuxWrap wraps seq in tmux passthrough, every ESC doubled.
func TmuxWrap(seq string) string {
	return "\x1bPtmux;" + strings.ReplaceAll(seq, "\x1b", "\x1b\x1b") + "\x1b\\"
}

// Copier copies text by the native clipboard and OSC 52 (doc/ui-design.md,
// Clipboard). Getenv is os.Getenv in the application.
type Copier struct {
	Native Clipboard
	Getenv func(string) string
}

// Copy returns the OSC 52 sequence to print (tmux-wrapped under tmux) and the
// toast naming the path used. Over SSH only OSC 52 is used.
func (c Copier) Copy(text string) (seq string, toast string, err error) {
	if text == "" {
		return "", "nothing to copy", nil
	}
	env := c.Getenv
	if env == nil {
		env = func(string) string { return "" }
	}
	seq = OSC52(text)
	if env("TMUX") != "" {
		seq = TmuxWrap(seq)
	}
	ssh := env("SSH_CONNECTION") != "" || env("SSH_TTY") != ""
	if !ssh && c.Native != nil && c.Native.Write(text) == nil {
		n := strings.Count(text, "\n") + 1
		if strings.HasSuffix(text, "\n") {
			n--
		}
		unit := "lines"
		if n == 1 {
			unit = "line"
		}
		return seq, fmt.Sprintf("copied %d %s · native", n, unit), nil
	}
	return seq, "copied · OSC 52 (terminal must allow it)", nil
}
