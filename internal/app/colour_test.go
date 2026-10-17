package app

import (
	"bytes"
	"context"
	"fmt"
	"image/color"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"testing/synctest"
	"time"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/Evengelius/agent-command-center/internal/provider"
	"github.com/Evengelius/agent-command-center/internal/render"
)

// The terminals of plan 5, item 13, as Bubble Tea detects them: Windows
// Terminal, a 256-colour and a 16-colour one, and output that is no terminal.
var colourEnvs = []struct{ name, env string }{
	{"true colour", "TTY_FORCE=1 WT_SESSION=1"},
	{"256", "TTY_FORCE=1 TERM=xterm-256color"},
	{"16", "TTY_FORCE=1 TERM=xterm"},
	{"none", "TERM=xterm-256color"},
}

// profileModel waits for Bubble Tea's colour profile message.
type profileModel struct{ got *tea.ColorProfileMsg }

func (p profileModel) Init() tea.Cmd  { return nil }
func (p profileModel) View() tea.View { return tea.NewView("") }
func (p profileModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m, ok := msg.(tea.ColorProfileMsg); ok {
		*p.got = m
		return p, tea.Quit
	}
	return p, nil
}

// frameModel shows one frame and quits.
type frameModel struct{ frame string }

func (f frameModel) Init() tea.Cmd                       { return tea.Quit }
func (f frameModel) Update(tea.Msg) (tea.Model, tea.Cmd) { return f, nil }
func (f frameModel) View() tea.View {
	v := tea.NewView(f.frame)
	v.AltScreen = true
	return v
}

// runTea runs m as on a terminal of environment env and returns what it
// wrote.
func runTea(t *testing.T, env string, m tea.Model) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var buf bytes.Buffer
	_, err := tea.NewProgram(m, tea.WithContext(ctx), tea.WithInput(nil), tea.WithOutput(&buf),
		tea.WithEnvironment(strings.Fields(env)), tea.WithWindowSize(120, 40), tea.WithoutSignals()).Run()
	if err != nil {
		t.Fatalf("%s: %v", env, err)
	}
	return buf.String()
}

// sgrColours counts the colours of each kind in the SGR sequences of out:
// 24-bit, 256 and the 16.
func sgrColours(out string) (tc, c256, c16 int) {
	for _, m := range sgrSeq.FindAllStringSubmatch(out, -1) {
		ps := strings.Split(m[1], ";")
		for k := 0; k < len(ps); k++ {
			switch p := ps[k]; {
			case (p == "38" || p == "48") && k+1 < len(ps) && ps[k+1] == "2":
				tc++
				k += 4
			case (p == "38" || p == "48") && k+1 < len(ps) && ps[k+1] == "5":
				c256++
				k += 2
			case len(p) == 2 && strings.IndexByte("3949", p[0]) >= 0 && p[1] <= '7',
				len(p) == 3 && p[:2] == "10" && p[2] <= '7':
				c16++
			}
		}
	}
	return tc, c256, c16
}

var (
	sgrSeq = regexp.MustCompile(`\x1b\[([0-9;:]*)m`)
	csiSeq = regexp.MustCompile(`^\x1b\[([0-9;?]*)([@-~])`)
)

// screenOf is the w x h screen a terminal shows after out: the characters
// at their cells, through the cursor moves and erasures Bubble Tea's
// renderer uses, which differ by profile.
func screenOf(out string, w, h int) string {
	g := make([][]rune, h)
	for y := range g {
		g[y] = []rune(strings.Repeat(" ", w))
	}
	x, y := 0, 0
	erase := func(from, to int) {
		for i := max(from, 0); i < min(to, w) && y < h; i++ {
			g[y][i] = ' '
		}
	}
	for i := 0; i < len(out); {
		if m := csiSeq.FindStringSubmatch(out[i:]); m != nil {
			n := 1
			if m[1] != "" && !strings.ContainsAny(m[1], ";?") {
				fmt.Sscan(m[1], &n)
			}
			switch m[2] {
			case "H":
				x, y = 0, 0
				if m[1] != "" {
					fmt.Sscanf(m[1], "%d;%d", &y, &x)
					x, y = x-1, y-1
				}
			case "C":
				x += n
			case "X":
				erase(x, x+n)
			case "K":
				erase(x, w)
			}
			i += len(m[0])
			continue
		}
		r, size := utf8.DecodeRuneInString(out[i:])
		i += size
		switch {
		case r == '\r':
			x = 0
		case r == '\n':
			y++
		case r >= 0x20 && x < w && y < h:
			g[y][x] = r
			x += max(lipgloss.Width(string(r)), 1)
		}
	}
	var b strings.Builder
	for _, row := range g {
		b.WriteString(string(row) + "\n")
	}
	return b.String()
}

// colourFrames are the frames of plan 5, item 13, criterion 1: the startup
// screen, the Claude and Codex dashboards and an agent's inspection.
var colourFrames = []struct {
	name, mock string
	keys       []tea.Msg
}{
	{"startup", "", nil},
	{"claude", "claude-subagent-approval.jsonl", nil},
	{"codex", "codex-subagents.jsonl", nil},
	{"inspect", "claude-subagent-approval.jsonl", []tea.Msg{key(tea.KeyF6, 0), key(tea.KeyEnter, 0), typed('2')}},
}

// drawWith is the frame the application draws at 120x40 once Bubble Tea
// gave it profile.
func drawWith(t *testing.T, mock string, keys []tea.Msg, profile tea.ColorProfileMsg) (frame string) {
	synctest.Test(t, func(t *testing.T) {
		opt := Options{Providers: []provider.Provider{fakeCLI{provider.Detection{ID: "claude-code", Name: "Claude Code", Installed: true, Version: "2.1.300", Path: `C:\Tools\claude.exe`}}}, Cwd: `C:\work\demo-project`}
		if mock != "" {
			opt = Options{Config: Config{Mock: filepath.Join("..", "..", "testdata", "mock", mock)}}
		}
		h := newHarness(t, opt)
		h.settle()
		h.send(profile)
		for _, k := range keys {
			h.send(k)
		}
		h.send(tea.WindowSizeMsg{Width: 120, Height: 40})
		frame = h.m.View().Content
	})
	return frame
}

// trueColour is the palette as it is before any profile, restored after a
// test.
func trueColour() {
	render.UsePalette(func(_, _, c color.Color) color.Color { return c })
}

// TestColourProfiles is plan 5, item 13, criteria 1, 3 and 4 through
// Bubble Tea itself: its detection gives each terminal's profile, the
// application draws with it, and its renderer writes the frame as that
// terminal receives it.
func TestColourProfiles(t *testing.T) {
	defer trueColour()
	profiles := map[string]tea.ColorProfileMsg{}
	for _, e := range colourEnvs {
		var got tea.ColorProfileMsg
		runTea(t, e.env, profileModel{&got})
		profiles[e.name] = got
	}
	for _, f := range colourFrames {
		t.Run(f.name, func(t *testing.T) {
			var text string
			content := map[string]string{}
			for _, e := range colourEnvs {
				content[e.name] = drawWith(t, f.mock, f.keys, profiles[e.name])
				out := runTea(t, e.env, frameModel{content[e.name]})
				// Criterion 1: the characters drawn are the same under all four.
				if s := screenOf(out, 120, 40); text == "" {
					text = s
					if !strings.Contains(s, "─") {
						t.Fatalf("%s: no frame drawn:\n%s", e.name, s)
					}
				} else if s != text {
					t.Errorf("%s draws other characters:\n%s\n--- true colour ---\n%s", e.name, s, text)
				}
				tc, c256, c16 := sgrColours(out)
				switch e.name {
				case "true colour":
					if tc == 0 || c256 != 0 || c16 != 0 {
						t.Errorf("true colour: %d 24-bit, %d 256, %d 16", tc, c256, c16)
					}
				case "256":
					if tc != 0 || c256 == 0 || c16 != 0 {
						t.Errorf("256: %d 24-bit, %d 256, %d 16", tc, c256, c16)
					}
				case "16":
					if tc != 0 || c256 != 0 || c16 == 0 {
						t.Errorf("16: %d 24-bit, %d 256, %d 16", tc, c256, c16)
					}
				case "none":
					if tc != 0 || c256 != 0 || c16 != 0 {
						t.Errorf("no terminal: %d 24-bit, %d 256, %d 16", tc, c256, c16)
					}
				}
			}
			// Criterion 4: in 256 colours and with none the application draws
			// what it draws in true colour, which Bubble Tea maps or strips.
			for _, n := range []string{"256", "none"} {
				if content[n] != content["true colour"] {
					t.Errorf("%s: the application draws another frame", n)
				}
			}
			// Criterion 3: the 16-colour palette, drawn with no 24-bit colour.
			if tc, _, c16 := sgrColours(content["16"]); tc != 0 || c16 == 0 {
				t.Errorf("16: the application draws %d 24-bit colours, %d of the 16", tc, c16)
			}
		})
	}
}

// TestPaletteSwitch is plan 5, item 13, criterion 3: the palette changes on
// a later profile message, and the next frame draws every row again in it,
// none kept from the frame before; back in true colour, the frame is the
// first one byte for byte.
func TestPaletteSwitch(t *testing.T) {
	defer trueColour()
	var sixteen, tc tea.ColorProfileMsg
	runTea(t, "TTY_FORCE=1 TERM=xterm", profileModel{&sixteen})
	runTea(t, "TTY_FORCE=1 WT_SESSION=1", profileModel{&tc})
	synctest.Test(t, func(t *testing.T) {
		h := newHarness(t, Options{Config: Config{Mock: filepath.Join("..", "..", "testdata", "mock", "claude-subagent-approval.jsonl")}})
		h.settle()
		h.send(tc)
		h.send(tea.WindowSizeMsg{Width: 120, Height: 40})
		first := h.m.View().Content
		h.m.View() // a second frame, every row reused
		h.send(sixteen)
		if n, _, _ := sgrColours(h.m.View().Content); n != 0 {
			t.Errorf("after the 16-colour profile, %d 24-bit colours remain", n)
		}
		h.send(tc)
		if again := h.m.View().Content; again != first {
			t.Error("back in true colour, the frame differs from the first")
		}
		h.send(tc) // the same profile again changes nothing
		if !strings.Contains(h.m.View().Content, "38;2;") {
			t.Error("no true colour drawn")
		}
	})
}
