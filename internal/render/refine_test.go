package render

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Evengelius/agent-command-center/internal/provider/mock"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// Plan 4, step 8, criterion 3, as plan 24 keeps it: on every mock file at
// four layouts, zone A's telemetry pane draws each quota window's label,
// its value and its bar whole, never cut; a missing cost reads "N/A",
// never "$ N/A"; no "▰" of zone D's bars remains.
func TestTelemetryWholeItems(t *testing.T) {
	files, err := filepath.Glob(filepath.Join("..", "..", "testdata", "mock", "*.jsonl"))
	if err != nil || len(files) == 0 {
		t.Fatalf("mock files: %v %v", files, err)
	}
	for _, f := range files {
		evs, err := mock.Load(f)
		if err != nil {
			t.Fatal(err)
		}
		s, now := mockState(f, evs)
		for _, size := range [][2]int{{100, 30}, {120, 40}, {156, 44}, {200, 60}} {
			w, h := size[0], size[1]
			t.Run(fmt.Sprintf("%s/%dx%d", filepath.Base(f), w, h), func(t *testing.T) {
				ui := UI{Width: w, Height: h, Now: now, Focus: "B"}
				frame := strings.Split(texts(Draw(s, ui).lines), "\n")
				aw := zoneAWidth(w)
				var body []string
				for _, l := range frame[3:] {
					r := []rune(l)
					if len(r) == 0 || r[0] == '╰' {
						break // zone A's bottom edge
					}
					if len(r) >= aw && r[0] == '│' {
						body = append(body, strings.TrimRight(string(r[2:aw-2]), " "))
					}
				}
				all := strings.Join(body, "\n")
				// The label and the value on one row, whole: at the row's end,
				// or before the tight form's countdown.
				for _, q := range windows(s, ui) {
					value := strings.TrimSuffix(texts([]line{q.right()}), "\n")
					found := false
					for _, l := range body {
						found = found || strings.Contains(l, q.label) && (strings.HasSuffix(l, value) || strings.Contains(l, value+" · "))
					}
					if !found {
						t.Errorf("no row with %q and %q:\n%s", q.label, value, all)
					}
				}
				for _, l := range body {
					if (strings.Contains(l, "█") || strings.Contains(l, "░")) && strings.Contains(l, "…") {
						t.Errorf("a bar's row cut: %q", l)
					}
				}
				if strings.Contains(all, "$ N/A") || strings.Contains(all, "▰") || strings.Contains(all, "▱") {
					t.Errorf("$ N/A or zone D's bar glyphs shown:\n%s", all)
				}
				if os.Getenv("ZONEA_SHOW") != "" {
					t.Logf("\n%s", all)
				}
			})
		}
	}
}

// mockState is the dashboard's state after a mock file's events, as mock mode
// starts it; now is a second after the last event.
func mockState(file string, evs []mock.Timed) (*state.State, time.Time) {
	s := state.New()
	s.Context = state.Context{Provider: state.MockProvider, ProviderName: "Mock data", Project: file, Mock: true}
	for _, e := range evs {
		s.Apply(e.Event)
	}
	s.Screen = state.ScreenDashboard
	now := time.Unix(0, 0)
	if len(s.Log) > 0 {
		now = s.Log[len(s.Log)-1].At.Add(time.Second)
	}
	return s, now
}

// Plan 4, step 8, criterion 2: zone C's title follows the screen's layout,
// and the conversation names the CLI a mock file recorded.
func TestZoneLabels(t *testing.T) {
	files, _ := filepath.Glob(filepath.Join("..", "..", "testdata", "mock", "*.jsonl"))
	for _, f := range files {
		evs, err := mock.Load(f)
		if err != nil {
			t.Fatal(err)
		}
		s, now := mockState(f, evs)
		want := map[bool]string{true: "CLAUDE", false: "CODEX"}[strings.HasPrefix(filepath.Base(f), "claude-")]
		if got := providerLabel(s); got != want {
			t.Errorf("%s: label %q, want %q", filepath.Base(f), got, want)
		}
		for _, size := range [][2]int{{80, 24}, {99, 30}, {100, 30}, {120, 40}, {200, 60}} {
			frame := texts(Draw(s, UI{Width: size[0], Height: size[1], Now: now, Focus: "B"}).lines)
			live := strings.Contains(frame, "C · LIVE AGENTS")
			if live != (size[0] >= standardWidth) || !strings.Contains(frame, "C · ") {
				t.Errorf("%s at %dx%d: zone C titled live=%v", filepath.Base(f), size[0], size[1], live)
			}
		}
	}
}

// Plan 4, step 8, criterion 4: zone B's badge for each turn state, its
// spinner turning with the frame.
func TestTurnBadge(t *testing.T) {
	for _, c := range []struct {
		name      string
		session   state.SessionPhase
		turn      state.TurnPhase
		streaming bool
		want      string // with "%s" for the spinner's glyph
	}{
		{"answer streaming", state.Ready, state.Running, true, "%s Streaming"},
		{"tool running", state.Ready, state.Running, false, "%s Running"},
		{"submitted", state.Ready, state.Submitted, false, "%s Running"},
		{"approval", state.Ready, state.WaitingApproval, false, "! Waiting for approval"},
		{"interrupting", state.Ready, state.Interrupting, false, "Interrupting…"},
		{"idle", state.Ready, state.Idle, false, "Idle"},
		{"starting", state.Handshaking, state.Idle, false, ""},
		{"ended", state.Crashed, state.Idle, false, ""},
	} {
		t.Run(c.name, func(t *testing.T) {
			s := state.New()
			s.Session.Phase, s.Turn.Phase = c.session, c.turn
			s.Conversation = []state.Entry{{Role: "user", Text: "hi"}, {Role: "assistant", Text: "Hel", Streaming: c.streaming}}
			var seen []string
			for f := 0; f < 2; f++ {
				ui := UI{Width: 120, Height: 40, Spinner: f * 3}
				got := strings.TrimSuffix(texts([]line{turnBadge(s, ui)}), "\n")
				want := c.want
				if strings.Contains(want, "%s") {
					want = fmt.Sprintf(want, spinner[f*3])
				}
				if got != want {
					t.Errorf("frame %d: %q, want %q", f*3, got, want)
				}
				// and in B's header, on the frame
				s.Screen = state.ScreenDashboard
				if frame := texts(Draw(s, ui).lines); want != "" && !strings.Contains(frame, want+" ─╮") {
					t.Errorf("frame %d: %q not in B's header", f*3, want)
				}
				seen = append(seen, got)
			}
			if strings.Contains(c.want, "%s") && seen[0] == seen[1] {
				t.Errorf("the spinner does not turn: %q", seen)
			}
		})
	}
}

// Plan 4, step 8, criterion 5: while the turn runs, a running tool in zone A
// shows the spinner and a dot moves down the running stage's connector, one
// row per frame; with the turn idle, the frames do not change.
func TestTreeAnimation(t *testing.T) {
	for _, file := range []string{"claude-subagent-approval.jsonl", "codex-subagents.jsonl"} {
		t.Run(file, func(t *testing.T) {
			evs, err := mock.Load(filepath.Join("..", "..", "testdata", "mock", file))
			if err != nil {
				t.Fatal(err)
			}
			// the first moment the turn runs with a tool under way
			s, _ := mockState(file, nil)
			var at int
			for at = range evs {
				s.Apply(evs[at].Event)
				if s.Turn.Phase == state.Running && runningTool(s) != nil {
					break
				}
			}
			if runningTool(s) == nil {
				t.Fatal("no running tool in the file")
			}
			now := s.Log[len(s.Log)-1].At
			tree := func(f int) []string {
				ui := UI{Width: 120, Height: 40, Now: now, Spinner: f}
				var out []string
				for _, r := range treeRows(s, ui, 44) { // wide enough for the names (A is 26 at 120 since plan 20)
					out = append(out, strings.TrimSuffix(texts([]line{r.line(s, ui, 44)}), "\n"))
				}
				return out
			}
			name := runningTool(s).Name
			var dots []int
			for f := 0; f < 4; f++ {
				rows := tree(f)
				spun := false
				dot := -1
				for i, r := range rows {
					if strings.Contains(r, spinner[f]+" "+name) {
						spun = true
					}
					if strings.Contains(r, "•") {
						if dot >= 0 {
							t.Errorf("frame %d: two dots", f)
						}
						dot = i
					}
				}
				if !spun {
					t.Errorf("frame %d: no %q spinner on %s:\n%s", f, spinner[f], name, strings.Join(rows, "\n"))
				}
				if dot < 0 {
					t.Fatalf("frame %d: no dot:\n%s", f, strings.Join(rows, "\n"))
				}
				dots = append(dots, dot)
			}
			// the connector's rows: those with ├ │ └ or the dot below the indent
			var trunk []int
			for i, r := range tree(0) {
				if p := []rune(r); len(p) > 2 && strings.ContainsRune("├│└•", p[2]) {
					trunk = append(trunk, i)
				}
			}
			for f, d := range dots {
				if len(trunk) < 2 || d != trunk[f%len(trunk)] {
					t.Errorf("frame %d: the dot on row %d, want connector row %d of %v", f, d, f%len(trunk), trunk)
				}
			}
			// No spinner turns while the turn waits for an approval, nor once
			// it is over: frames 0 and 5 draw the same screen.
			s.Screen = state.ScreenDashboard
			same := func(when string) {
				for _, size := range [][2]int{{80, 24}, {120, 40}, {200, 60}} {
					a := Draw(s, UI{Width: size[0], Height: size[1], Now: now, Spinner: 0}).String()
					b := Draw(s, UI{Width: size[0], Height: size[1], Now: now, Spinner: 5}).String()
					if a != b {
						t.Errorf("%s, %dx%d: frames 0 and 5 differ", when, size[0], size[1])
					}
				}
			}
			waited := false
			for _, e := range evs[at+1:] {
				s.Apply(e.Event)
				if s.Turn.Phase == state.WaitingApproval && !waited {
					waited = true
					same("waiting for approval")
				}
			}
			if turnRuns(s) {
				t.Fatalf("turn still %s at the end of the file", s.Turn.Phase)
			}
			same("idle")
		})
	}
}

// runningTool is a running call of the orchestrator's, if any.
func runningTool(s *state.State) *state.ToolCall {
	for _, tc := range s.Tools {
		if tc.Agent == "" && (tc.Status == state.ToolStarted || tc.Status == state.ToolRunning) {
			return tc
		}
	}
	return nil
}

// Plan 4, step 8, check 7: the overlay has every row of ui-design.md's
// "Anywhere" and "A focused panel" tables, and plan 5, item 4, of "B's
// entries"; plan 12 adds F4, plan 16 Ctrl+W, plan 17 F3, the file viewer and the Explorer
// (45 rows), plan 34 Alt+P and the model selector; at 80 columns nothing is
// cut.
func TestKeyOverlayRows(t *testing.T) {
	doc, err := os.ReadFile(filepath.Join("..", "..", "doc", "ui-design.md"))
	if err != nil {
		t.Fatal(err)
	}
	sec := string(doc)
	sec = sec[strings.Index(sec, "\n## Focus and keys"):]
	sec = sec[:strings.Index(sec, "\n**The editor**")]
	have := map[string]bool{}
	for _, g := range KeyTable {
		for _, k := range g.keys {
			have[k[0]] = true
		}
	}
	n := 0
	for _, l := range strings.Split(sec, "\n") {
		cells := strings.Split(l, "|")
		if len(cells) < 3 || strings.TrimSpace(cells[1]) == "Key" || strings.TrimSpace(cells[1]) == "-" {
			continue
		}
		k := strings.ReplaceAll(strings.TrimSpace(cells[1]), "`", "")
		n++
		if !have[k] {
			t.Errorf("ui-design.md key %q missing from the overlay", k)
		}
	}
	if n != 67 { // B's entries: Enter, → and ← apart since the owner's live review; plan 34's seven; plan 37's ← →; plan 44's Source Control (5) and diff (7)
		t.Errorf("%d rows read from ui-design.md, want 67", n)
	}
	for _, l := range strings.Split(texts(keysBody(78)), "\n") {
		if strings.Contains(l, "…") || textWidth(l) > 78 {
			t.Errorf("overlay row cut at 80 columns: %q", l)
		}
	}
}
