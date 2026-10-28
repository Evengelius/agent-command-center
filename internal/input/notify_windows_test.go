//go:build windows

package input

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestNotifyShortcut is plan 64's criterion 64.5, in a folder of the test's
// and never the Start menu: the shortcut made when missing, to Windows
// Terminal's alias with ACC's profile and ACC's app ID; not rewritten once
// there; one of that name that is not ACC's left as it is, with no
// notifier; no shortcut when the terminal is missing. No notification is
// shown: Show and Clear are checked in Windows Terminal (64.6).
func TestNotifyShortcut(t *testing.T) {
	dir := t.TempDir()
	terminal := filepath.Join(t.TempDir(), "wt.exe")
	if err := os.WriteFile(terminal, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, NotifyShortcut+".lnk")
	n, problem := SetUpNotifications(dir, terminal)
	if n == nil || problem != "" {
		t.Fatalf("missing: notifier %v, problem %q", n, problem)
	}
	var target, args, id string
	if err := withRuntime(func() (err error) { target, args, id, err = readShortcut(path); return }); err != nil {
		t.Fatal(err)
	}
	if !strings.EqualFold(target, terminal) || args != `-p "Agent Command Center"` || id != NotifyAppID {
		t.Errorf("made: target %q, arguments %q, app ID %q", target, args, id)
	}
	old := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	if err := os.Chtimes(path, old, old); err != nil {
		t.Fatal(err)
	}
	if n, problem := SetUpNotifications(dir, terminal); n == nil || problem != "" {
		t.Errorf("ACC's there: notifier %v, problem %q", n, problem)
	}
	if fi, err := os.Stat(path); err != nil || !fi.ModTime().Equal(old) {
		t.Errorf("ACC's there: rewritten (%v, %v)", fi.ModTime(), err)
	}

	// Another program's shortcut of that name: left as it is, no notifier.
	other := t.TempDir()
	foreign := filepath.Join(other, NotifyShortcut+".lnk")
	if err := withRuntime(func() error { return makeShortcut(foreign, terminal, "", "", "Someone.Else") }); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(foreign)
	if err != nil {
		t.Fatal(err)
	}
	n, problem = SetUpNotifications(other, terminal)
	after, _ := os.ReadFile(foreign)
	if n != nil || !strings.Contains(problem, "not ACC's") || !strings.Contains(problem, `"Someone.Else"`) || string(after) != string(before) {
		t.Errorf("not ACC's: notifier %v, problem %q, changed %v", n, problem, string(after) != string(before))
	}

	// No Windows Terminal: no shortcut, no notifier.
	none := t.TempDir()
	n, problem = SetUpNotifications(none, filepath.Join(none, "missing", "wt.exe"))
	if _, err := os.Stat(filepath.Join(none, NotifyShortcut+".lnk")); n != nil || !strings.Contains(problem, "wt.exe was not found") || err == nil {
		t.Errorf("no terminal: notifier %v, problem %q, shortcut made %v", n, problem, err == nil)
	}
}
