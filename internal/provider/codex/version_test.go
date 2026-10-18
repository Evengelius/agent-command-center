package codex

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/Evengelius/agent-command-center/internal/provider"
)

// Plan 14, item 6: TestedVersion is the README's last probed Codex CLI
// version (doc/evaluation.md, After a CLI update), and Detect reports it.
func TestTestedVersion(t *testing.T) {
	b, err := os.ReadFile("../../../doc/research/probes/README.md")
	if err != nil {
		t.Fatal(err)
	}
	// The bullet, with its wrapped lines.
	var last string
	in := false
	for _, l := range strings.Split(string(b), "\n") {
		switch {
		case strings.HasPrefix(l, "- **Last probed:**"):
			in, last = true, l
		case in && strings.HasPrefix(l, "  "):
			last += " " + strings.TrimSpace(l)
		default:
			in = false
		}
	}
	if !strings.Contains(last, "Codex CLI "+TestedVersion+" ") {
		t.Errorf("TestedVersion %s not in the README's last probed line %q", TestedVersion, last)
	}
	// An empty PATH and home: nothing is found and nothing runs.
	h := provider.Host{Run: func(context.Context, string, ...string) (provider.Output, error) {
		t.Fatal("a CLI was run")
		return provider.Output{}, nil
	}}
	if d := (&Provider{Host: h}).Detect(context.Background()); d.Tested != TestedVersion {
		t.Errorf("Detect: Tested %q, want %q", d.Tested, TestedVersion)
	}
}
