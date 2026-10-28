package input

import (
	"strings"
	"testing"
)

// TestToastXML is plan 64's criterion 64.3, the input part: the title over
// the text, escaped, the sound muted, nothing else in the content.
func TestToastXML(t *testing.T) {
	got := toastXML("Approval required", "Claude Code is waiting for your approval.")
	want := `<toast><visual><binding template="ToastGeneric"><text>Approval required</text><text>Claude Code is waiting for your approval.</text></binding></visual><audio silent="true"/></toast>`
	if got != want {
		t.Errorf("content\n got %s\nwant %s", got, want)
	}
	got = toastXML(`a & b < c`, `"d" > 'e'`)
	if !strings.Contains(got, "<text>a &amp; b &lt; c</text><text>&quot;d&quot; &gt; &apos;e&apos;</text>") {
		t.Errorf("not escaped: %s", got)
	}
}
