package render

import (
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Evengelius/agent-command-center/internal/state"
)

// sessionStates are zone B's session states (ui-design.md, Zone B, States),
// each with a long, wrapped reason, a cut stderr line and a tab.
func sessionStates() []struct {
	name string
	s    *state.State
} {
	at := time.Date(2026, 10, 17, 9, 30, 5, 0, time.Local)
	mk := func(f func(s *state.State)) *state.State {
		s := state.New()
		s.Screen = state.ScreenDashboard
		s.Context.Provider, s.Context.ProviderName = "codex", "Codex CLI"
		s.Context.Project = `C:\work\projects\a-rather-long-project-name\with\a\deep\path`
		s.Session.Version = "0.163.0"
		f(s)
		return s
	}
	failed := func(s *state.State) {
		s.Session.Reason = "the app-server closed its output before the handshake ended, after the initialize request\nsecond line of the reason"
		s.Session.ExitCode, s.Session.HasExitCode = 3, true
		s.Session.StderrTail = []string{"one", "two", "three", "four", "five", "six: a very long stderr line that no zone B is wide enough to show on one row, cut at its edge", "seven\twith a tab"}
	}
	return []struct {
		name string
		s    *state.State
	}{
		{"mock start failed", mk(func(s *state.State) { s.Context.Mock = true; s.Startup.Phase = state.StartFailed; failed(s) })},
		{"start failed", mk(func(s *state.State) { s.Startup.Phase = state.StartFailed; failed(s) })},
		{"crashed, resumable", mk(func(s *state.State) { s.Session.Phase, s.Session.ID = state.Crashed, "t1"; failed(s) })},
		{"crashed", mk(func(s *state.State) { s.Session.Phase = state.Crashed; s.Session.Reason = "killed" })},
		{"spawning", mk(func(s *state.State) { s.Session.Phase = state.Spawning })},
		{"closed", mk(func(s *state.State) {
			s.Session.Phase = state.Closed
			s.Conversation = []state.Entry{{Seq: 1, At: at, Role: "user", Text: "a question\nover two lines"}}
		})},
		{"ready", mk(func(s *state.State) { s.Session.Phase = state.Ready })},
	}
}

// sessionRows is what sessionState draws: each row's text, then its style
// runs, adjacent spans of one style merged.
func sessionRows(s *state.State, w int) string {
	var rows []string
	for _, l := range sessionState(s, UI{Spinner: 1}, w) {
		var text, runs strings.Builder
		last := ""
		for _, x := range l.fit(w) {
			st := "-"
			if x.st != nil {
				st = x.st.Render("x")
			}
			if x.text != "" && st != last {
				fmt.Fprintf(&runs, " %d:%q", textWidth(text.String()), st)
				last = st
			}
			text.WriteString(x.text)
		}
		rows = append(rows, text.String()+"|"+runs.String())
	}
	return strings.Join(rows, "\n")
}

// TestSessionStateCopy is plan 5, item 9, criteria 1 to 4: every session
// state's content, after the last entry, is copied from its logical lines
// (a wrapped reason as one line, a cut stderr line whole, a tab kept), never
// its symbol, indentation, spinner or key hint; a selection there is
// highlighted; the rows drawn are pinned.
func TestSessionStateCopy(t *testing.T) {
	failed := []string{"", "the app-server closed its output before the handshake ended, after the initialize request", "second line of the reason", "exit code 3", "stderr:", "three", "four", "five", "six: a very long stderr line that no zone B is wide enough to show on one row, cut at its edge", "seven\twith a tab"}
	card := func(title string) []string { return append([]string{title}, failed[1:]...) }
	want := map[string][]string{
		"mock start failed":  card("The mock replay could not start"),
		"start failed":       card("The session could not start"),
		"crashed, resumable": card("The session ended unexpectedly"),
		"crashed":            {"The session ended unexpectedly", "killed"},
		"spawning":           {`Starting Codex CLI 0.163.0 in C:\work\projects\a-rather-long-project-name\with\a\deep\path…`},
		"closed":             {"Session closed."},
		"ready":              {"Waiting for instructions", "Ready to work on this project."},
	}
	for _, c := range sessionStates() {
		n, ls := len(c.s.Conversation), want[c.name]
		if got := sessionLines(c.s); !slices.Equal(got, ls) {
			t.Errorf("%s: lines %q, want %q", c.name, got, ls)
		}
		for j, l := range ls {
			if got := (&Screen{}).LineText(c.s, Pos{'B', n, j, 0}); got != l {
				t.Errorf("%s: line %d is %q, want %q", c.name, j, got, l)
			}
		}
		for _, size := range [][2]int{{80, 24}, {120, 40}, {200, 60}} {
			sc := Draw(c.s, UI{Width: size[0], Height: size[1], Spinner: 1})
			// The selection a drag over every shown row of the state makes.
			var from, to Pos
			for y := range sc.lines {
				for _, u := range sc.units('B', y) {
					if u.start.Item != n {
						continue
					}
					if from.Panel == 0 {
						from = u.start
					}
					to = u.end
				}
			}
			if from.Panel == 0 {
				t.Errorf("%s %v: nothing of the state is selectable", c.name, size)
				continue
			}
			got := sc.Text(c.s, Selection{from, to})
			if exp := between(ls, from.Line, from.Off, len(ls)-1, len(ls[len(ls)-1])); got != exp {
				t.Errorf("%s %v: copied %q, want %q", c.name, size, got, exp)
			}
			if size[1] >= 40 && got != strings.Join(ls, "\n") {
				t.Errorf("%s %v: copied %q, want all of it", c.name, size, got)
			}
			for _, d := range []string{"✗", "Ctrl+", spinner[1]} {
				if strings.Contains(got, d) {
					t.Errorf("%s %v: copied %q with %q", c.name, size, got, d)
				}
			}
			hl := Draw(c.s, UI{Width: size[0], Height: size[1], Spinner: 1, Sel: Selection{from, to}})
			last := ls[len(ls)-1]
			if l, _ := rowOf(t, hl, firstWord(last)); styleOf(l, firstWord(last)) != stSelected {
				t.Errorf("%s %v: %q not highlighted", c.name, size, last)
			}
			if strings.Contains(texts(hl.lines), "✗") {
				if l, _ := rowOf(t, hl, "✗"); styleOf(l, "✗") == stSelected {
					t.Errorf("%s %v: the ✗ is highlighted", c.name, size)
				}
			}
		}
	}
	// A selection kept after the state it ended in is gone (Back, a resume)
	// copies what is left.
	ready := sessionStates()[6].s
	ready.Conversation = []state.Entry{{Seq: 1, Role: "user", Text: "kept"}}
	if got := (&Screen{}).Text(ready, Selection{Pos{'B', 0, 0, 0}, Pos{'B', 1, 0, 5}}); got != "kept" {
		t.Errorf("a stale selection copied %q", got)
	}
	// Criterion 4: the rows, as before the change but for the tab, now
	// drawn to its stop as in every content line of B.
	for _, c := range []struct {
		state string
		w     int
		rows  string
	}{
		{"crashed, resumable", 30, "✗ The session ended unexpecte…\n  the app-server closed its   \n  output before the handshake \n  ended, after the initialize \n  request                     \n  second line of the reason   \n  exit code 3                 \n  stderr:                     \n    three                     \n    four                      \n    five                      \n    six: a very long stderr l…\n    seven    with a tab       \nCtrl+R resume · Ctrl+Q quit   "},
		{"spawning", 56, "  ⠙ Starting Codex CLI 0.163.0 in                       \n" + `  C:\work\projects\a-rather-long-project-name\with\a\dee` + "\n" + `  p\path…                                               `},
		{"ready", 30, "Waiting for instructions      \nReady to work on this project."},
	} {
		for _, st := range sessionStates() {
			if st.name != c.state {
				continue
			}
			var rows []string
			for _, r := range strings.Split(sessionRows(st.s, c.w), "\n") {
				rows = append(rows, r[:strings.LastIndex(r, "|")])
			}
			if got := strings.Join(rows, "\n"); got != c.rows {
				t.Errorf("%s at %d:\n%s\nwant\n%s", c.state, c.w, got, c.rows)
			}
		}
	}
}

func firstWord(s string) string { return strings.Fields(s)[0] }

// TestSessionCutRow is plan 5, item 9, criterion 1, at every width: a cut
// row's shown part, then its "…", copy its source line exactly, the cut
// falling in a word or just after a tab.
func TestSessionCutRow(t *testing.T) {
	s := sessionStates()[3].s // crashed
	s.Session.StderrTail = []string{"\ta\tbc\tdef\tghij\tklmno\tpqrstu\tEND"}
	src := sessionLines(s)[3]
	cuts := 0
	for w := 8; w <= 60; w++ {
		sc := &Screen{lines: sessionState(s, UI{}, w)}
		for y := range sc.lines {
			us := sc.units('B', y)
			if len(us) < 2 || us[0].start.Line != 3 {
				continue
			}
			last := us[len(us)-1]
			if last.end.Off == last.start.Off+1 || last.end.Off != len(src) {
				continue // not cut
			}
			cuts++
			shown := sc.Text(s, Selection{us[0].start, us[len(us)-2].end})
			rest := sc.Text(s, Selection{last.start, last.end})
			if shown+rest != src {
				t.Errorf("width %d: shown %q, then %q; want %q", w, shown, rest, src)
			}
		}
	}
	if cuts < 40 {
		t.Errorf("%d widths cut the line", cuts)
	}
}
