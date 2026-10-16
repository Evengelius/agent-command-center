//go:build !windows

package app

// win32Input is false: win32-input-mode is a Windows console mode, and
// Windows Terminal is the only terminal the application targets.
const win32Input = false

func consoleReset(string) {}
