//go:build !windows

package input

// NerdIcons is false outside Windows, the one platform the dashboard
// supports: the Explorer draws its own glyphs (plan 55, design D3).
func NerdIcons(func(string) string) bool { return false }
