package fixture

import (
	"crypto/sha256"
	"os"
	"path/filepath"
	"testing"
)

// The fixtures in testdata/probes are byte-identical copies of the plan 1
// recordings in doc/research/probes (proposal step 0, item 5).
func TestFixturesMatchRecordings(t *testing.T) {
	want := map[string]int{
		"p1-claude-stream.jsonl":                        12,
		"p2-claude-conversation.jsonl":                  79,
		"p3-claude-subagent.jsonl":                      99,
		"p3b-claude-subagent-approval.jsonl":            153,
		"p4-claude-resume.jsonl":                        4,
		"p5-codex-exec.jsonl":                           5,
		"p5b-codex-exec.jsonl":                          4,
		"p6-codex-app-server.jsonl":                     37,
		"p6b-codex-app-server.jsonl":                    94,
		"p7-codex-subagents.jsonl":                      173,
		"p7b-codex-subagents.jsonl":                     215,
		"p7c-codex-spawn-model.jsonl":                   108,
		"p9a-codex-spawn-request.jsonl":                 63,
		"p12c-block-claude-auto-boundary-edit.jsonl":    208,
		"p12c-claude-auto.jsonl":                        99,
		"p12c-switch-claude-default-to-auto.jsonl":      211,
		"p12x-codex-auto-review.jsonl":                  167,
		"p12x-sandbox-codex-turn-workspace-write.jsonl": 245,
		"p12x-switch-codex-user-to-auto-review.jsonl":   180,
		"p13c-claude-auto-classifier-deny.jsonl":        260,
		"p14a-claude-question-answer.jsonl":             68,
		"p22o-claude-opus-thinking.jsonl":               57,  // plan 22, amendment 20: Opus thinks; the owner's SessionStart hook
		"p22s-codex-reasoning-summary.jsonl":            124, // plan 22, amendment 20: Codex's reasoning summaries asked for
		"p24u-claude-usage-report.jsonl":                11,  // plan 24, amendment 23: "/usage" over stream-json, the usage report
		"p303a-claude-conversation.jsonl":               79,
		"p303b-claude-subagent-approval.jsonl":          198,
		"p303c-claude-default-to-auto.jsonl":            259,
		"p303d-claude-question-answer.jsonl":            62,
		"p303e-claude-interrupt-tool.jsonl":             88,
		"p34c-claude-model-settings.jsonl":              57,  // plan 34: Claude Code's models and settings, no message
		"p34x-codex-model-list-settings.jsonl":          75,  // plan 34: Codex's model/list pages and thread settings, no turn
		"p38x-codex-settings.jsonl":                     40,  // plan 38: thread/settings/update with the experimental API, no turn
		"p38t-codex-turn-settings.jsonl":                73,  // plan 38: the same, then one turn and a command approval
		"p39c-claude-permission-modes.jsonl":            23,  // plan 39: Claude Code's permission modes, no message
		"p39x-codex-permission-modes.jsonl":             48,  // plan 39: Codex's three presets through thread/settings/update, no turn
		"p69a-claude-resume.jsonl":                      77,  // plan 69: Claude Code resumed in a new process, and an unknown session
		"p69b-codex-resume.jsonl":                       100, // plan 69: Codex resumed in a new app-server, with and without its turns
	}
	copies, _ := filepath.Glob("../../testdata/probes/*.jsonl")
	if len(copies) != len(want) {
		t.Fatalf("testdata/probes has %d recordings, want %d", len(copies), len(want))
	}
	for name, lines := range want {
		copyPath := filepath.Join("../../testdata/probes", name)
		a, err := os.ReadFile(copyPath)
		if err != nil {
			t.Fatal(err)
		}
		b, err := os.ReadFile(filepath.Join("../../doc/research/probes", name))
		if err != nil {
			t.Fatal(err)
		}
		if sha256.Sum256(a) != sha256.Sum256(b) {
			t.Errorf("%s: SHA-256 differs from the original recording", name)
		}
		recs, err := Read(copyPath)
		if err != nil {
			t.Fatal(err)
		}
		if len(recs) != lines {
			t.Errorf("%s: %d records, want %d", name, len(recs), lines)
		}
	}
}

// The model list Codex returned in P6 answers model/list in the step 5 fake
// app-server; it too is a byte-identical copy.
func TestModelListMatchesRecording(t *testing.T) {
	const name = "p6-codex-model-list.json"
	a, err := os.ReadFile(filepath.Join("../../testdata/probes", name))
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join("../../doc/research/probes", name))
	if err != nil {
		t.Fatal(err)
	}
	if sha256.Sum256(a) != sha256.Sum256(b) {
		t.Errorf("%s: SHA-256 differs from the original recording", name)
	}
}
