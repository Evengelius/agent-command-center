package input

import (
	"context"
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
