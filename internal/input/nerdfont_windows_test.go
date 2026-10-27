package input

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// Plan 55, design D3, with this machine's registry and processes: outside
// the acc tab the icons are never used, whatever the font; where the font
// is installed for the user, it is found, with the time its file was made.
func TestNerdIconsHere(t *testing.T) {
	for _, profile := range []string{"", "{61c54bbd-c2c6-5271-96e7-009a87ff44bf}"} {
		if NerdIcons(func(string) string { return profile }) {
			t.Errorf("profile %q: the icons used outside the acc tab", profile)
		}
	}
	file := filepath.Join(os.Getenv("LOCALAPPDATA"), `Microsoft\Windows\Fonts\CascadiaMonoNF.ttf`)
	if _, err := os.Stat(file); err != nil {
		t.Skipf("Cascadia Mono NF not installed for the user: %v", err)
	}
	at, found := nerdFontInstalled()
	if !found || at.IsZero() || at.After(time.Now()) {
		t.Errorf("the installed font: found %v at %v", found, at)
	}
}
