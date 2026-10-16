package confcheck

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type fakeProcs []string

func (f fakeProcs) Names() ([]string, error) { return f, nil }

func write(t *testing.T, p, s string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(s), 0o644); err != nil {
		t.Fatal(err)
	}
}

func snap(t *testing.T, c *Checker) *Snapshot {
	t.Helper()
	s, err := c.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func setup(t *testing.T) (*Checker, string) {
	home := t.TempDir()
	write(t, filepath.Join(home, ".claude", "settings.json"), "a")
	write(t, filepath.Join(home, ".claude.json"), "m")
	write(t, filepath.Join(home, ".codex", "config.toml"), "c")
	return &Checker{Home: home, Procs: fakeProcs{"explorer.exe", "node.exe"}}, home
}

func TestIdenticalPass(t *testing.T) {
	c, _ := setup(t)
	r := Compare(snap(t, c), snap(t, c), false)
	if r.Verdict != Pass || r.Verdict.ExitCode() != 0 {
		t.Fatalf("%s\n%s", r.Verdict, r)
	}
}

func TestConfigChangedFail(t *testing.T) {
	c, home := setup(t)
	b := snap(t, c)
	write(t, filepath.Join(home, ".claude", "settings.json"), "changed")
	r := Compare(b, snap(t, c), true)
	if r.Verdict != Fail || !strings.Contains(r.String(), "settings.json  changed") {
		t.Fatalf("%s", r)
	}
}

func TestMixedWithCLINotVerified(t *testing.T) {
	c, home := setup(t)
	c.Procs = fakeProcs{"claude.exe"}
	b := snap(t, c)
	write(t, filepath.Join(home, ".claude.json"), "rewritten")
	r := Compare(b, snap(t, c), false)
	if r.Verdict != NotVerified || r.Verdict.ExitCode() != 2 ||
		!strings.Contains(r.String(), "changed, not classified (contents not inspected)") {
		t.Fatalf("%s", r)
	}
}

func TestMixedWithoutCLIFail(t *testing.T) {
	c, home := setup(t)
	b := snap(t, c)
	write(t, filepath.Join(home, ".claude.json"), "rewritten")
	r := Compare(b, snap(t, c), false)
	if r.Verdict != Fail {
		t.Fatalf("%s", r)
	}
	if Compare(b, snap(t, c), true).Verdict != NotVerified {
		t.Fatal("cli-ran should give NOT-VERIFIED")
	}
}

func TestCredentialNeverOpened(t *testing.T) {
	c, home := setup(t)
	proj := t.TempDir()
	cred := []string{
		filepath.Join(home, ".claude", "skills", "x", ".credentials.json"),
		filepath.Join(proj, ".codex", "auth.json"),
		filepath.Join(proj, ".claude", "OAuth_creds.json"),
	}
	for _, p := range cred {
		write(t, p, "SECRET")
	}
	write(t, filepath.Join(proj, ".claude", "ok.md"), "x")
	c.Project = proj
	var opened []string
	c.Hash = func(p string) (string, error) {
		opened = append(opened, p)
		return FileHash(p)
	}
	b := snap(t, c)
	if len(opened) == 0 {
		t.Fatal("hasher never called")
	}
	for _, o := range opened {
		if IsCredentialName(o) {
			t.Fatalf("credential file opened: %s", o)
		}
	}
	r := Compare(b, snap(t, c), false)
	if got := strings.Count(r.String(), "skipped (credential name)"); got != 3 {
		t.Fatalf("want 3 skipped, got %d\n%s", got, r)
	}
	if r.Verdict != Pass {
		t.Fatal(r.Verdict)
	}
}

func TestBaselineHoldsNoContent(t *testing.T) {
	c, home := setup(t)
	write(t, filepath.Join(home, ".claude", "settings.json"), "MARKER-xyzzy-12345")
	write(t, filepath.Join(home, ".claude.json"), "MARKER-xyzzy-12345")
	data, err := json.MarshalIndent(snap(t, c), "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "MARKER-xyzzy") {
		t.Fatal("content leaked into baseline")
	}
	if strings.Contains(string(data), home) {
		t.Fatal("home path leaked into baseline")
	}
}

func TestNewSessionFilesCounted(t *testing.T) {
	c, home := setup(t)
	write(t, filepath.Join(home, ".claude", "projects", "p", "old.jsonl"), "1")
	b := snap(t, c)
	write(t, filepath.Join(home, ".claude", "projects", "p", "new.jsonl"), "2")
	write(t, filepath.Join(home, ".codex", "sessions", "2026", "r.jsonl"), "3")
	r := Compare(b, snap(t, c), true)
	if r.NewSessions != 2 || r.Verdict != Pass {
		t.Fatalf("%d %s", r.NewSessions, r.Verdict)
	}
}

func TestProjectClaudeChangedFail(t *testing.T) {
	c, _ := setup(t)
	proj := t.TempDir()
	write(t, filepath.Join(proj, ".claude", "settings.local.json"), "a")
	c.Project = proj
	b := snap(t, c)
	write(t, filepath.Join(proj, ".claude", "settings.local.json"), "b")
	r := Compare(b, snap(t, c), false)
	if r.Verdict != Fail || !strings.Contains(r.String(), `<project>\.claude\settings.local.json  changed`) {
		t.Fatalf("%s", r)
	}
}

func TestAppearedAndAbsent(t *testing.T) {
	c, home := setup(t)
	b := snap(t, c)
	write(t, filepath.Join(home, ".claude", "CLAUDE.md"), "new")
	r := Compare(b, snap(t, c), false)
	if r.Verdict != Fail || !strings.Contains(r.String(), "appeared") || !strings.Contains(r.String(), "absent") {
		t.Fatalf("%s", r)
	}
}
