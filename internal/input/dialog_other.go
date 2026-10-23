//go:build !windows

package input

import "errors"

// PickImages has no dialog outside Windows, the one platform the dashboard
// supports (plan 31, design D2).
func PickImages() ([]string, error) {
	_ = pickFilter()
	return nil, errors.New("no file dialog on this platform")
}
