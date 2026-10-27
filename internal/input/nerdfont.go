package input

import (
	"strings"
	"time"
)

// AccProfile is the Windows Terminal profile scripts/acc.ps1 writes for the
// dashboard's tab (plan 23): its face is Cascadia Mono NF when the font is
// there for it (plan 55, design D2). Windows Terminal gives each tab its
// profile's id in WT_PROFILE_ID.
const AccProfile = "{27195de0-7daf-4340-8ef6-2c640982bd39}"

// NerdFontFace is the font the acc tab names, Microsoft's Cascadia Mono NF.
const NerdFontFace = "Cascadia Mono NF"

// nerdIconsUsable is plan 55's rule for the Explorer's Nerd Font icons
// (design D3): in the acc tab, under a Windows Terminal that started after
// the font was installed, as it reads the fonts when it starts. Anything
// missing, or a terminal started first, gives the panel's own glyphs.
func nerdIconsUsable(profile string, terminalStarted time.Time, fontFound bool, fontInstalled time.Time) bool {
	return strings.EqualFold(profile, AccProfile) && fontFound && !terminalStarted.IsZero() && fontInstalled.Before(terminalStarted)
}
