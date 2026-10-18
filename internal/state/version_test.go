package state

import "testing"

// Plan 14, item 6: the project bar marks a CLI newer than the version its
// adapter was last probed with (doc/evaluation.md, After a CLI update); an
// equal, older, unparsable or unknown version, or the mock, gives no mark.
func TestNewerThanTested(t *testing.T) {
	for _, c := range []struct {
		version, tested string
		mock            bool
		want            bool
	}{
		{"2.1.303 (Claude Code)", "2.1.302", false, true},
		{"2.2.0", "2.1.302", false, true},
		{"3.0.0", "2.1.302", false, true},
		{"2.1.302 (Claude Code)", "2.1.302", false, false},
		{"2.1.301", "2.1.302", false, false},
		{"1.9.999", "2.1.302", false, false},
		{"v2.1.310", "2.1.302", false, true},
		{"2.1.310", "2.1.302", true, false},
		{"unknown", "2.1.302", false, false},
		{"", "2.1.302", false, false},
		{"2.1.310", "", false, false},
	} {
		s := &State{Context: Context{Provider: "claude-code", Mock: c.mock}}
		s.Startup.CLIs = []CLI{{ID: "codex", Tested: "0.0.1"}, {ID: "claude-code", Tested: c.tested}}
		s.Session.Version = c.version
		if got := s.NewerThanTested(); got != c.want {
			t.Errorf("version %q, tested %q, mock %v: %v, want %v", c.version, c.tested, c.mock, got, c.want)
		}
	}
	// No detection entry for the session's provider: no mark.
	s := &State{Context: Context{Provider: "claude-code"}}
	s.Session.Version = "9.9.9"
	if s.NewerThanTested() {
		t.Error("no CLI entry: marked newer")
	}
}
