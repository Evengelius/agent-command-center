package render

import (
	"strings"
	"testing"

	"github.com/Evengelius/agent-command-center/internal/state"
)

// Plan 14, item 6: "newer than tested" follows the version in the project
// bar, in the attention colour, only when the CLI is newer than tested.
func TestProjectBarNewerThanTested(t *testing.T) {
	for _, c := range []struct {
		version string
		mock    bool
		want    bool
	}{{"2.1.303", false, true}, {"2.1.302", false, false}, {"2.1.301", false, false}, {"", false, false}, {"2.1.303", true, false}} {
		s := &state.State{Context: state.Context{Provider: "claude-code", ProviderName: "Claude Code", Project: `C:\tmp\p`, Mock: c.mock}}
		s.Startup.CLIs = []state.CLI{{ID: "claude-code", Tested: "2.1.302"}}
		s.Session.Version = c.version
		l := projectBar(s, UI{Width: 160})
		got := rowsText([]line{l})
		if strings.Contains(got, "newer than tested") != c.want {
			t.Errorf("version %q, mock %v: %q", c.version, c.mock, got)
		}
		if c.want {
			// The mark ends the bar while approvals are Manual (plan 28).
			if !strings.HasSuffix(strings.TrimSuffix(got, "\n"), "Claude Code 2.1.303 newer than tested ") {
				t.Errorf("mark misplaced: %q", got)
			}
			for _, x := range l {
				if x.text == " newer than tested" && x.st != stAttention {
					t.Error("mark not in the attention colour")
				}
			}
		}
	}
}
