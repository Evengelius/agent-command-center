//go:build !windows

package input

import "errors"

// PickFolder has no dialog outside Windows, the one platform the dashboard
// supports (plan 52, design D1).
func PickFolder(string) (string, error) {
	return "", errors.New("no folder dialog on this platform")
}
