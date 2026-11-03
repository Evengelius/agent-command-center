package arch

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// reviewerPath is the Reviewer's definition, relative to this package. It is
// one constant so that an overlay can repoint the test (doc/evaluation.md, The
// Reviewer).
const reviewerPath = "../../.claude/agents/reviewer.md"

// reviewerFields is the definition's frontmatter as doc/evaluation.md, The
// Reviewer, lists it. Claude Code ignores an unknown frontmatter field without
// a word (doc/plan74-evaluation.md), so a misspelt omitClaudeMd would change
// the Reviewer with no sign: the keys are exactly these, and the values are
// exact, except the description's and the colour's, which only must not be
// empty.
var reviewerFields = map[string]string{
	"name":         "reviewer",
	"description":  "",
	"model":        "claude-opus-5-5",
	"effort":       "high",
	"tools":        "Read, Grep, Glob",
	"omitClaudeMd": "true",
	"maxTurns":     "100",
	"color":        "",
}

// TestReviewerDefinition holds the Reviewer's frontmatter to reviewerFields
// and the retired verifier absent (doc/evaluation.md, The Reviewer).
func TestReviewerDefinition(t *testing.T) {
	b, err := os.ReadFile(reviewerPath)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.ReplaceAll(string(b), "\r\n", "\n"), "\n")
	if lines[0] != "---" {
		t.Fatal("reviewer.md: no frontmatter")
	}
	got := map[string]string{}
	for _, l := range lines[1:] {
		if l == "---" {
			break
		}
		key, value, _ := strings.Cut(l, ":")
		if _, dup := got[key]; dup {
			t.Errorf("repeated field %q", key)
		}
		if _, ok := reviewerFields[key]; !ok {
			t.Errorf("unknown field %q", key)
		}
		got[key] = strings.TrimSpace(value)
	}
	for key, want := range reviewerFields {
		v, ok := got[key]
		if !ok {
			t.Errorf("missing field %q", key)
		} else if want == "" && v == "" {
			t.Errorf("field %q is empty", key)
		} else if want != "" && v != want {
			t.Errorf("field %q = %q, want %q", key, v, want)
		}
	}
	if _, err := os.Stat(filepath.Join("..", "..", ".claude", "agents", "verifier.md")); err == nil {
		t.Error("verifier.md exists: the Reviewer replaced it")
	}
}
