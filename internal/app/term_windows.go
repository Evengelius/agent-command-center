package app

import (
	"os"

	"golang.org/x/sys/windows"
)

// win32Input asks the console for win32-input-mode (ui-design.md,
// Shift+Enter): in VT input mode the console reports Shift+Enter as a plain
// carriage return, while a win32-input-mode key carries its modifiers.
const win32Input = true

// consoleReset writes seq to the console once the program has given the
// terminal back, so a mode the application asked for does not outlive it.
// The console's output mode is put back as it was; nothing is written when
// the output is not a console.
func consoleReset(seq string) {
	h := windows.Handle(os.Stdout.Fd())
	var mode uint32
	if windows.GetConsoleMode(h, &mode) != nil {
		return
	}
	if mode&windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING == 0 {
		if windows.SetConsoleMode(h, mode|windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING) != nil {
			return
		}
		defer windows.SetConsoleMode(h, mode)
	}
	_, _ = os.Stdout.WriteString(seq)
}
