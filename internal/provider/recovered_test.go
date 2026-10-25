package provider

import (
	"regexp"
	"testing"
)

// TestRecovered is plan 48's design D5: a recovered panic's text names its
// value and the function, file and line it came from, a runtime error's too.
func TestRecovered(t *testing.T) {
	recovered := func(f func()) (s string) {
		defer func() { s = Recovered(recover()) }()
		f()
		return ""
	}
	var m map[string]int
	for _, c := range []struct {
		f    func()
		want string
	}{
		{func() { panic("boom") }, `^panic: boom \(at provider\.TestRecovered\.func\d+, recovered_test\.go:\d+\)$`},
		{func() { m["x"] = 1 }, `^panic: assignment to entry in nil map \(at provider\.TestRecovered\.func\d+, recovered_test\.go:\d+\)$`},
	} {
		if got := recovered(c.f); !regexp.MustCompile(c.want).MatchString(got) {
			t.Errorf("%q, want %s", got, c.want)
		}
	}
}
