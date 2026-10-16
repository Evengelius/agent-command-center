//go:build !windows

package input

// TerminalSelects does nothing outside Windows: a terminal selects on its own
// once the application stops capturing the mouse.
func TerminalSelects(on bool) error { return nil }
