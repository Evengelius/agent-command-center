package input

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"golang.org/x/sys/windows/registry"
)

// NerdIcons says whether the Explorer draws Cascadia Mono NF's icons
// (plan 55, design D3): in the acc tab of a Windows Terminal that started
// after the font was installed. Read once, at start.
func NerdIcons(getenv func(string) string) bool {
	_, started := hostTerminal()
	installed, found := nerdFontInstalled()
	return nerdIconsUsable(getenv("WT_PROFILE_ID"), started, found, installed)
}

// nerdFontInstalled is when Cascadia Mono NF was installed, for the user or
// the machine: the latest creation time of the files its registered names
// lead to (the regular and the italic); not found when none is there.
func nerdFontInstalled() (time.Time, bool) {
	var latest time.Time
	found := false
	for _, root := range []registry.Key{registry.CURRENT_USER, registry.LOCAL_MACHINE} {
		k, err := registry.OpenKey(root, `Software\Microsoft\Windows NT\CurrentVersion\Fonts`, registry.QUERY_VALUE)
		if err != nil {
			continue
		}
		names, _ := k.ReadValueNames(0)
		for _, n := range names {
			if !strings.HasPrefix(n, NerdFontFace+" ") {
				continue // "Cascadia Mono NF (TrueType)", "Cascadia Mono NF Italic (TrueType)"
			}
			file, _, err := k.GetStringValue(n)
			if err != nil || file == "" {
				continue
			}
			if !filepath.IsAbs(file) {
				file = filepath.Join(os.Getenv("WINDIR"), "Fonts", file) // the machine's, by name
			}
			fi, err := os.Stat(file)
			if err != nil {
				continue
			}
			at := time.Now()
			if a, ok := fi.Sys().(*syscall.Win32FileAttributeData); ok {
				at = time.Unix(0, a.CreationTime.Nanoseconds())
			}
			if !found || at.After(latest) {
				latest = at
			}
			found = true
		}
		k.Close()
	}
	return latest, found
}
