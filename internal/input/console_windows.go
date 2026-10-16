package input

import "golang.org/x/sys/windows"

// TerminalSelects lets the console select with the mouse (QuickEdit) or not.
// Bubble Tea's Windows input reader clears QuickEdit, so in conhost F12 alone
// would leave no mouse selection at all; Windows Terminal selects on its
// own and ignores the flag (ui-design.md, Selection and copy; found in
// conhost, plan 4, step 3, criterion 9).
func TerminalSelects(on bool) error {
	name, err := windows.UTF16PtrFromString("CONIN$")
	if err != nil {
		return err
	}
	h, err := windows.CreateFile(name, windows.GENERIC_READ|windows.GENERIC_WRITE,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE, nil, windows.OPEN_EXISTING, 0, 0)
	if err != nil {
		return err
	}
	defer windows.CloseHandle(h)
	var mode uint32
	if err := windows.GetConsoleMode(h, &mode); err != nil {
		return err
	}
	mode |= windows.ENABLE_EXTENDED_FLAGS
	if on {
		mode |= windows.ENABLE_QUICK_EDIT_MODE
	} else {
		mode &^= windows.ENABLE_QUICK_EDIT_MODE
	}
	return windows.SetConsoleMode(h, mode)
}
