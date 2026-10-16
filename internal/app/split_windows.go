package app

import "golang.org/x/sys/windows"

// splitCommand splits an editor command with CommandLineToArgvW's rules, so
// `code --wait` and quoted paths work (ui-design.md, $EDITOR, item 1).
func splitCommand(s string) ([]string, error) { return windows.DecomposeCommandLine(s) }
