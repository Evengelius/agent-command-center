package confcheck

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

const testProj = "C:/work/test-proj"

func claudeJSONSetup(t *testing.T, content string) (*Checker, string) {
	c, home := setup(t)
	c.Project = testProj
	write(t, filepath.Join(home, ".claude.json"), content)
	return c, home
}

const baseJSON = `{"numStartups":1,"tipsHistory":{"a":1},"cachedGrowthBookFeatures":{"x":1},
"mcpServers":{"s":{"command":"a"}},"oauthAccount":{"v":"old"},
"projects":{"C:/work/test-proj":{"allowedTools":[],"lastCost":1},"C:/other":{"allowedTools":["x"]}}}`

func TestKeysStateOnly(t *testing.T) {
	c, home := claudeJSONSetup(t, baseJSON)
	b := snap(t, c)
	write(t, filepath.Join(home, ".claude.json"), strings.Replace(
		strings.Replace(strings.Replace(baseJSON, `"numStartups":1`, `"numStartups":2`, 1),
			`"tipsHistory":{"a":1}`, `"tipsHistory":{"a":2}`, 1),
		`"cachedGrowthBookFeatures":{"x":1}`, `"cachedGrowthBookFeatures":{"x":2}`, 1))
	if r := Compare(b, snap(t, c), true); r.Verdict != Pass || !strings.Contains(r.String(), "key numStartups  changed (state)") {
		t.Fatalf("%s", r)
	}
	if r := Compare(b, snap(t, c), false); r.Verdict != Fail {
		t.Fatalf("%s", r)
	}
}

func TestKeysConfigFail(t *testing.T) {
	c, home := claudeJSONSetup(t, baseJSON)
	b := snap(t, c)
	write(t, filepath.Join(home, ".claude.json"), strings.Replace(baseJSON, `"command":"a"`, `"command":"b"`, 1))
	r := Compare(b, snap(t, c), true)
	if r.Verdict != Fail || !strings.Contains(r.String(), "key mcpServers  changed (config)") {
		t.Fatalf("%s", r)
	}
	write(t, filepath.Join(home, ".claude.json"), strings.Replace(baseJSON, `{"numStartups"`, `{"brandNewKey":1,"numStartups"`, 1))
	r = Compare(b, snap(t, c), true)
	if r.Verdict != Fail || !strings.Contains(r.String(), "key brandNewKey  changed (config)") {
		t.Fatalf("%s", r)
	}
}

func TestKeysTestProject(t *testing.T) {
	c, home := claudeJSONSetup(t, `{"projects":{"C:/other":{"x":1}}}`)
	b := snap(t, c)
	write(t, filepath.Join(home, ".claude.json"),
		"{\"projects\":{\"C:/other\":{\"x\":1},\"c:\\\\Work\\\\Test-Proj\\\\\":{\"lastSessionId\":\"abc\",\"projectOnboardingSeenCount\":1}}}")
	if r := Compare(b, snap(t, c), true); r.Verdict != Pass {
		t.Fatalf("%s", r)
	}
	write(t, filepath.Join(home, ".claude.json"),
		`{"projects":{"C:/other":{"x":1},"C:/work/test-proj":{"allowedTools":["Bash"]}}}`)
	r := Compare(b, snap(t, c), true)
	if r.Verdict != Fail || !strings.Contains(r.String(), "].allowedTools  changed (config)") {
		t.Fatalf("%s", r)
	}
}

func TestKeysOtherProject(t *testing.T) {
	c, home := claudeJSONSetup(t, baseJSON)
	b := snap(t, c)
	write(t, filepath.Join(home, ".claude.json"), strings.Replace(baseJSON, `["x"]`, `["y"]`, 1))
	r := Compare(b, snap(t, c), true)
	if r.Verdict != NotVerified || !strings.Contains(r.String(), "projects[C:/other]  changed (another project)") {
		t.Fatalf("%s", r)
	}
}

func TestKeysSensitiveNeverLeaks(t *testing.T) {
	c, home := claudeJSONSetup(t, baseJSON)
	b := snap(t, c)
	write(t, filepath.Join(home, ".claude.json"), strings.Replace(baseJSON, `"v":"old"`, `"v":"SECRET-MARKER-456"`, 1))
	a := snap(t, c)
	r := Compare(b, a, true)
	if r.Verdict != NotVerified || !strings.Contains(r.String(), "key oauthAccount  not compared (sensitive name)") ||
		!strings.Contains(r.String(), "no compared key changed") {
		t.Fatalf("%s", r)
	}
	for _, s := range []*Snapshot{b, a} {
		data, _ := json.Marshal(s)
		if strings.Contains(string(data), "SECRET-MARKER-456") {
			t.Fatal("marker in snapshot")
		}
	}
	if strings.Contains(r.String(), "SECRET-MARKER-456") {
		t.Fatal("marker in report")
	}
}

func TestKeysInvalidJSONFallsBack(t *testing.T) {
	c, home := claudeJSONSetup(t, baseJSON)
	good := snap(t, c)
	write(t, filepath.Join(home, ".claude.json"), `{"numStartups":`)
	bad := snap(t, c)
	for _, p := range [][2]*Snapshot{{good, bad}, {bad, good}} {
		r := Compare(p[0], p[1], true)
		if r.Verdict != NotVerified || !strings.Contains(r.String(), "changed, not classified") {
			t.Fatalf("%s", r)
		}
	}
}
