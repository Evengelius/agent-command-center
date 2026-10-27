package input

import (
	"testing"
	"time"
)

// Plan 55, design D3 and criterion 55.4 (R2): the Explorer's Nerd Font
// icons only in the acc tab of a Windows Terminal started after the font
// was installed; no terminal, another profile, no font, or a terminal
// started before the font each give the panel's own glyphs.
func TestNerdIconsUsable(t *testing.T) {
	installed := time.Date(2026, 10, 27, 15, 0, 0, 0, time.UTC)
	after, before := installed.Add(time.Hour), installed.Add(-time.Hour)
	for _, c := range []struct {
		name     string
		profile  string
		terminal time.Time
		found    bool
		want     bool
	}{
		{"the acc tab, the terminal started after the font", AccProfile, after, true, true},
		{"the profile's id in other letters' case", "{27195DE0-7DAF-4340-8EF6-2C640982BD39}", after, true, true},
		{"no Windows Terminal", AccProfile, time.Time{}, true, false},
		{"another profile", "{61c54bbd-c2c6-5271-96e7-009a87ff44bf}", after, true, false},
		{"no profile: another terminal", "", after, true, false},
		{"no font", AccProfile, after, false, false},
		{"a terminal started before the font", AccProfile, before, true, false},
		{"a terminal started as the font was installed", AccProfile, installed, true, false},
	} {
		if got := nerdIconsUsable(c.profile, c.terminal, c.found, installed); got != c.want {
			t.Errorf("%s: %v, want %v", c.name, got, c.want)
		}
	}
}
