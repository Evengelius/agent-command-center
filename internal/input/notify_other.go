//go:build !windows

package input

// Outside Windows the dashboard shows no notification (plan 64, design D7).

// Notifier shows nothing outside Windows.
type Notifier struct{}

// Show does nothing outside Windows.
func (*Notifier) Show(title, text string) error { return nil }

// Clear does nothing outside Windows.
func (*Notifier) Clear() error { return nil }

// NotifyPaths gives no folder outside Windows.
func NotifyPaths() (dir, terminal string, err error) { return "", "", nil }

// SetUpNotifications gives no notifier outside Windows, and no problem.
func SetUpNotifications(dir, terminal string) (*Notifier, string) { return nil, "" }

// Notifications gives no calls outside Windows, and no problem.
func Notifications() (show func(title, text string) error, clear func() error, problem string) {
	return nil, nil, ""
}
