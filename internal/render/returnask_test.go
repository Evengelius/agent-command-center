package render

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"

	"github.com/Evengelius/agent-command-center/internal/state"
)

// returnState is a ready Claude Code session on the dashboard.
func returnState() *state.State {
	s := state.New()
	s.Screen = state.ScreenDashboard
	s.Context = state.Context{Provider: "claude-code", ProviderName: "Claude Code", Project: `C:\work\demo-project`}
	s.Startup.Phase, s.Session.Phase = state.StartReady, state.Ready
	return s
}

// boxText is the text of the card's rows, between its side edges and their
// margins, row by row.
func boxText(sc *Screen) []string {
	b := sc.Return
	var out []string
	for y := b.Y + 1; y < b.Y+b.H-1; y++ {
		var t strings.Builder
		for _, s := range cols(sc.lines[y], b.X+3, b.X+b.W-3) {
			t.WriteString(s.text)
		}
		out = append(out, strings.TrimRight(t.String(), " "))
	}
	return out
}

// TestReturnBox is criterion 75.11 in the drawing (plan 75, design D7): over
// the dashboard at the three sizes, a card on zone B's columns, its title in
// amber (the palette's In progress, #FBBF24) and bold, the text wrapped inside its margins in order, Yes and No
// with No selected, the keys in grey; the buttons the summary card's, the
// selected one bright and bold, its label underlined, the other subdued; no
// ground or inverse; the layer alone on top.
func TestReturnBox(t *testing.T) {
	defer UsePalette(pickTrue)
	s := returnState()
	for _, pick := range []lipgloss.CompleteFunc{pickTrue, pickANSI} {
		UsePalette(pick)
		for _, size := range [][2]int{{80, 24}, {120, 40}, {200, 60}} {
			for _, yes := range []bool{false, true} {
				ui := UI{Width: size[0], Height: size[1], Focus: "B", Return: ReturnAsk{Open: true, Yes: yes}}
				sc := Draw(s, ui)
				if sc.Layers.Top != LayerReturn || sc.Return.W == 0 {
					t.Fatalf("%v: top %d, card %+v", size, sc.Layers.Top, sc.Return)
				}
				if b := sc.Return; b.X < sc.B.X || b.X+b.W > sc.B.X+sc.B.W || b.Y < 0 || b.Y+b.H > size[1] {
					t.Errorf("%v: the card %+v outside zone B's columns %+v or the screen", size, b, sc.B)
				}
				rows := boxText(sc)
				var words []string
				for _, r := range rows {
					words = append(words, strings.Fields(r)...)
				}
				joined := strings.Join(words, " ")
				want := returnTitle + " " + ReturnText("Claude Code") + " [ Yes ] [ No ] " + returnHint
				if joined != want {
					t.Errorf("%v: the card says\n%q\nwant\n%q", size, joined, want)
				}
				styles := map[string][]*lipgloss.Style{}
				for y := sc.Return.Y; y < sc.Return.Y+sc.Return.H; y++ {
					for _, sp := range cols(sc.lines[y], sc.Return.X, sc.Return.X+sc.Return.W) {
						if sp.st == nil {
							continue
						}
						if _, none := sp.st.GetBackground().(lipgloss.NoColor); !none || sp.st.GetReverse() {
							t.Errorf("%v: %q has a ground or inverse", size, sp.text)
						}
						if txt := strings.TrimSpace(sp.text); txt != "" {
							styles[txt] = append(styles[txt], sp.st)
						}
						if sp.ref.panel == ReturnPanel && sp.ref.item != ReturnYes && sp.ref.item != ReturnNo {
							t.Errorf("%v: a click target %d", size, sp.ref.item)
						}
					}
				}
				if st := styles[returnTitle]; len(st) != 1 || st[0] != stProgressBold {
					t.Errorf("%v: the title's style %v, want the amber of In progress (#FBBF24), bold", size, st)
				}
				sel, other := stSummaryFocus, stMuted
				yesLabel, noLabel := styles["Yes"], styles["No"]
				if yes {
					if len(yesLabel) != 1 || yesLabel[0] != sel || len(noLabel) != 1 || noLabel[0] != other {
						t.Errorf("%v, Yes selected: labels %v, %v", size, yesLabel, noLabel)
					}
				} else if len(yesLabel) != 1 || yesLabel[0] != other || len(noLabel) != 1 || noLabel[0] != sel {
					t.Errorf("%v, No selected: labels %v, %v", size, yesLabel, noLabel)
				}
				if st := styles[returnHint]; len(st) != 1 || st[0] != stGrey {
					t.Errorf("%v: the keys' style %v, want grey", size, st)
				}
			}
		}
	}
}

// TestReturnLayer is the question's place among the layers (plan 49's rule):
// on the dashboard alone at a size it draws, none on the startup screen or
// too small; under it no popup shows.
func TestReturnLayer(t *testing.T) {
	s := returnState()
	ui := UI{Width: 120, Height: 40, Return: ReturnAsk{Open: true}, Plus: true, Keys: true}
	if st := Layers(s, ui); st.Top != LayerReturn || st.Shown(LayerPlus) || st.Shown(LayerKeys) {
		t.Errorf("the dashboard: top %d, plus %v, keys %v", st.Top, st.Shown(LayerPlus), st.Shown(LayerKeys))
	}
	ui.Width = 70
	if st := Layers(s, ui); st.Shown(LayerReturn) {
		t.Error("shown on a screen too small")
	}
	ui.Width = 120
	s.Screen = state.ScreenStartup
	if st := Layers(s, ui); st.Shown(LayerReturn) {
		t.Error("shown over the startup screen")
	}
}
