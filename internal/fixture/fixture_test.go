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
		"p1-claude-stream.jsonl":             12,
		"p2-claude-conversation.jsonl":       79,
		"p3-claude-subagent.jsonl":           99,
		"p3b-claude-subagent-approval.jsonl": 153,
		"p4-claude-resume.jsonl":             4,
		"p5-codex-exec.jsonl":                5,
		"p5b-codex-exec.jsonl":               4,
		"p6-codex-app-server.jsonl":          37,
		"p6b-codex-app-server.jsonl":         94,
		"p7-codex-subagents.jsonl":           173,
		"p7b-codex-subagents.jsonl":          215,
		"p7c-codex-spawn-model.jsonl":        108,
		"p9a-codex-spawn-request.jsonl":      63,
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
