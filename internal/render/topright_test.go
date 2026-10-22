package render

import (
	"strings"
	"testing"

	"github.com/Evengelius/agent-command-center/internal/state"
)

// TestTopRight is plan 28's criterion 28.1 (brief amendment 28, the owner's
// image of the top right): the project bar ends with the provider's name
// and version alone while approvals are Manual with no change asked; the
// mode shows there only while Auto or changing; the freed cells go to the
// path in the middle.
func TestTopRight(t *testing.T) {
	s := p3bAtWrite(t)
	s.Context.Provider, s.Context.ProviderName = "claude-code", "Claude Code"
	project := `C:\Users\owner\projects\` + strings.Repeat("folder-", 24) + `\acc-demo` // longer than the room at 208 columns
	s.Context.Project = project
	s.Session.Version = "2.1.303"
	bar := func(ui UI) (string, line) {
		l := Draw(s, ui).lines[1]
		return strings.TrimSuffix(texts([]line{l}), "\n"), l
	}
	// shown is the longest tail of the project path on the bar: the
	// middle's elided path.
	shown := func(b string) int {
		for n := len(project); n > 0; n-- {
			if strings.Contains(b, project[len(project)-n:]) {
				return n
			}
		}
		return 0
	}
	for _, size := range [][2]int{{120, 40}, {208, 59}} {
		ui := UI{Width: size[0], Height: size[1], Focus: "B"}
		s.Session.ApprovalMode, s.Session.ApprovalRequested = state.ModeManual, ""
		manual, _ := bar(ui)
		if !strings.HasSuffix(manual, "Claude Code 2.1.303 ") || strings.Contains(manual, "approvals") || textWidth(manual) != size[0] {
			t.Errorf("%v: Manual: %q", size, manual)
		}
		if !strings.HasPrefix(manual, " ▣ acc-demo") {
			t.Errorf("%v: the bar's left part %q", size, manual)
		}
		s.Session.ApprovalMode = state.ModeAuto
		auto, l := bar(ui)
		if !strings.HasSuffix(auto, "Claude Code 2.1.303 · approvals Auto ") {
			t.Errorf("%v: Auto: %q", size, auto)
		}
		if x, ok := spanOf(l, "Auto"); !ok || x.st != stAttentionBold {
			t.Errorf("%v: Auto styled %v, want stAttentionBold", size, x.st)
		}
		s.Session.ApprovalMode, s.Session.ApprovalRequested = state.ModeManual, state.ModeAuto
		if b, _ := bar(ui); !strings.HasSuffix(b, "Claude Code 2.1.303 · approvals Manual → Auto … ") {
			t.Errorf("%v: Auto asked for: %q", size, b)
		}
		s.Session.ApprovalMode, s.Session.ApprovalRequested = state.ModeAuto, state.ModeManual
		if b, _ := bar(ui); !strings.HasSuffix(b, "Claude Code 2.1.303 · approvals Auto → Manual … ") {
			t.Errorf("%v: Manual asked for: %q", size, b)
		}
		// Without the mode, the path shows at least 10 cells more of itself.
		if d := shown(manual) - shown(auto); d < 10 {
			t.Errorf("%v: the path shows %d cells without the mode and %d with it, want 10 more at least:\n%s\n%s", size, shown(manual), shown(auto), manual, auto)
		}
	}
}
