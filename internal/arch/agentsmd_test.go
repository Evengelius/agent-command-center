package arch

import (
	"os"
	"path/filepath"
	"testing"
)

// codexDocLimit is Codex CLI's default project_doc_max_bytes: the most it
// reads of a project's AGENTS.md files, together. A file past what is left is
// cut to fit, with nothing but a log line to say so (openai/codex
// rust-v0.163.0: codex-rs/config/src/config_toml.rs:75,
// codex-rs/core/src/agents_md.rs:156-166). Instructions from outside the
// project are added apart from it (agents_md.rs:63-68).
const codexDocLimit = 32 * 1024

// agentsMDCeiling keeps an eighth of that limit, 4 KiB, unused: a warning
// before the end of AGENTS.md, its Working rules, would be cut, and room for
// another AGENTS.md on the path to Codex's working directory, which would
// share the limit (the repository has none).
const agentsMDCeiling = codexDocLimit * 7 / 8

// TestAgentsMDWithinCodexLimit holds AGENTS.md under that ceiling, counting
// its bytes on disk, as Codex reads them (AGENTS.md, Working rules: keep this
// file short).
func TestAgentsMDWithinCodexLimit(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("..", "..", "AGENTS.md"))
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("AGENTS.md: %d bytes; ceiling %d; Codex's limit %d", len(b), agentsMDCeiling, codexDocLimit)
	if len(b) > agentsMDCeiling {
		t.Errorf("AGENTS.md is %d bytes, over its ceiling of %d (7/8 of the %d bytes Codex reads): move detail into doc/ and point to it",
			len(b), agentsMDCeiling, codexDocLimit)
	}
}
