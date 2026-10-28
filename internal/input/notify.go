package input

import "strings"

// Windows' notifications for the ping (plan 64, brief amendment 66;
// ui-design.md, The ping): ACC's app ID, the Start menu shortcut that names
// it for Windows, and a notification's content. The calls to Windows are
// notify_windows.go's; elsewhere there are none.

// NotifyAppID is ACC's app ID for Windows' notifications, the shortcut's and
// the notifier's alike (design D5).
const NotifyAppID = "Evengelius.AgentCommandCenter"

// NotifyShortcut is the shortcut's name in the user's Start menu; it opens
// a tab in ACC's own Windows Terminal profile, whose name it shares
// (scripts/acc.ps1).
const NotifyShortcut = "Agent Command Center"

// notifyTag and notifyGroup label every notification ACC shows, so that a
// newer one replaces the older and one call takes it out (design D4).
const (
	notifyTag   = "ping"
	notifyGroup = "acc"
)

// toastEscape escapes a text for the notification's XML.
var toastEscape = strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;", "'", "&apos;")

// toastXML is a notification's content: its title over its text, the sound
// muted (Microsoft Learn, toast schema, audio: silent).
func toastXML(title, text string) string {
	return `<toast><visual><binding template="ToastGeneric"><text>` + toastEscape.Replace(title) +
		`</text><text>` + toastEscape.Replace(text) + `</text></binding></visual><audio silent="true"/></toast>`
}
