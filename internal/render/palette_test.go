package render

import (
	"bytes"
	"context"
	"fmt"
	"image/color"
	"slices"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// Lip Gloss's choice among an ANSI, a 256 and a true colour, as
// lipgloss.Complete makes it for each profile.
var (
	pickANSI  lipgloss.CompleteFunc = func(a, _, _ color.Color) color.Color { return a }
	pick256   lipgloss.CompleteFunc = func(_, b, _ color.Color) color.Color { return b }
	pickTrue  lipgloss.CompleteFunc = func(_, _, c color.Color) color.Color { return c }
	pickNoTTY lipgloss.CompleteFunc = func(_, _, _ color.Color) color.Color { return lipgloss.NoColor{} }
)

// paletteStyles is every shared style the palette paints, by name.
func paletteStyles() map[string]*lipgloss.Style {
	m := map[string]*lipgloss.Style{
		"frame": stFrame, "title": stTitle, "text": stText, "done": stDone, "progress": stProgress,
		"attention": stAttention, "failed": stFailed, "grey": stGrey, "bold": stBold, "button": stButton,
		"attentionBold": stAttentionBold, "failedBold": stFailedBold, "cursor": stCursor, "selected": stSelected,
		"control": stControl, "inferred": stInferred, "treeCursor": stTreeCursor, "mdBullet": stMdBullet,
		"mdEm": stMdEm, "mdCode": stMdCode, "mdLink": stMdLink, "mdLang": stMdLang, "mdKeyword": stMdKeyword,
		"mdString": stMdString, "mdNumber": stMdNumber, "mdComment": stMdComment, "synBuiltin": stSynBuiltin,
		"synTag": stSynTag, "bright": stBright, "you": stYou, "speaker": stSpeaker, "result": stResult,
		"ground": tint(stText),
		// Plan 20: a target's muted text, the tool's name, the fold's mark,
		// the pill (its ends are its ground's colour).
		"muted": stMuted, "toolName": stToolName, "foldMark": stFoldMark, "pill": stPill,
	}
	for i, mc := range modelColours {
		m[mc.family] = modelStyles[i]
	}
	return m
}

// snapshot is how each shared style draws.
func snapshot() map[string]string {
	out := map[string]string{}
	for k, st := range paletteStyles() {
		out[k] = st.Render("x")
	}
	return out
}

// TestPalette16 is plan 5, item 13, criterion 3: on a 16-colour terminal
// the explicit palette applies, its groups stay apart, and a later true
// colour or 256 profile brings back every style as it was.
func TestPalette16(t *testing.T) {
	defer UsePalette(pickTrue)
	before := snapshot()
	for _, pick := range []lipgloss.CompleteFunc{pick256, pickTrue, pickNoTTY} {
		if UsePalette(pick) {
			t.Fatal("a 256, true colour or no-colour profile changed the palette")
		}
	}
	if !UsePalette(pickANSI) || UsePalette(pickANSI) {
		t.Fatal("the 16-colour profile changes the palette once")
	}
	basic := map[color.Color]bool{}
	for i := range 16 {
		basic[lipgloss.Color(fmt.Sprint(i))] = true
	}
	sixteenSnap := snapshot()
	for k, st := range paletteStyles() {
		if f := st.GetForeground(); !basic[f] {
			t.Errorf("%s: foreground %v is not one of the 16", k, f)
		}
		if b, ok := st.GetBackground().(lipgloss.NoColor); !ok && !basic[st.GetBackground()] {
			t.Errorf("%s: background %v is not one of the 16", k, b)
		}
		if sixteenSnap[k] == before[k] {
			t.Errorf("%s draws as in true colour", k)
		}
	}
	fgs := func(sts ...*lipgloss.Style) []color.Color {
		var out []color.Color
		for _, st := range sts {
			out = append(out, st.GetForeground())
		}
		return out
	}
	for name, group := range map[string][]color.Color{
		"statuses": fgs(stDone, stProgress, stAttention, stFailed, stGrey),
		"models":   fgs(append(modelStyles, modelStyle("other"))...),
		"code":     fgs(stMdCode, stMdKeyword, stMdString, stMdNumber, stMdComment),
	} {
		for i := range group {
			for j := range i {
				if group[i] == group[j] {
					t.Errorf("%s %d and %d share %v", name, i, j, group[i])
				}
			}
		}
	}
	// Neutral text, frame and hints: white and bright black.
	for k, want := range map[*lipgloss.Style]string{stText: "7", stBold: "7", stFrame: "8", stGrey: "8", stTitle: "15"} {
		if k.GetForeground() != lipgloss.Color(want) {
			t.Errorf("%s is %v, want %s", k.Render("x"), k.GetForeground(), want)
		}
	}
	for _, st := range []*lipgloss.Style{stButton, stCursor, stSelected, stTreeCursor} {
		if st.GetForeground() == st.GetBackground() {
			t.Errorf("%q: text on its own colour", st.Render("x"))
		}
	}
	if !UsePalette(pickTrue) {
		t.Fatal("the true colour profile does not bring the palette back")
	}
	for k, v := range snapshot() {
		if v != before[k] {
			t.Errorf("%s after the round trip draws %q, want %q", k, v, before[k])
		}
	}
	UsePalette(pickANSI)
	if !UsePalette(pick256) {
		t.Fatal("the 256 profile does not bring the palette back")
	}
}

// TestPalette256 is plan 5, item 13, criterion 2: through Bubble Tea's
// renderer on a 256-colour terminal, the palette's distinct foregrounds stay
// distinct, and so do its backgrounds; on a 16-colour one each colour is
// the ANSI colour the palette names.
func TestPalette256(t *testing.T) {
	defer UsePalette(pickTrue)
	names := []string{}
	for k := range paletteStyles() {
		names = append(names, k)
	}
	slices.Sort(names)
	measure := func(env string) (fgs, bgs map[string]string) {
		var rows []string
		for i, k := range names {
			rows = append(rows, fmt.Sprintf("%02d", i)+paletteStyles()[k].Render("X"))
		}
		text, sts := sgrCells(teaOut(t, env, strings.Join(rows, "\n")))
		fgs, bgs = map[string]string{}, map[string]string{}
		s := string(text)
		for i, k := range names {
			at := strings.Index(s, fmt.Sprintf("%02dX", i))
			if at < 0 {
				t.Fatalf("%s: row %d not drawn", env, i)
			}
			c := sts[len([]rune(s[:at]))+2]
			fgs[k], bgs[k] = c.fg, c.bg
		}
		return fgs, bgs
	}
	tcF, tcB := measure("WT_SESSION=1")
	f256, b256 := measure("TERM=xterm-256color")
	n := 0
	for _, k := range names {
		if tcF[k] == "" {
			t.Errorf("%s: no foreground drawn", k)
		}
		if tcB[k] != "" {
			n++
		}
	}
	if n != 6 { // the button, the editor's cursor and selection, the tree's cursor, zone B's tool rows, the pill (plan 20)
		t.Errorf("%d backgrounds drawn, want 6", n)
	}
	for _, pair := range [][2]map[string]string{{tcF, f256}, {tcB, b256}} {
		to := map[string]string{}
		for _, k := range names {
			a, b := pair[0][k], pair[1][k]
			if a != "" && !strings.HasPrefix(a, "38;2;") && !strings.HasPrefix(a, "48;2;") {
				t.Errorf("%s: true colour drawn as %q", k, a)
			}
			if a != "" && !strings.HasPrefix(b, "38;5;") && !strings.HasPrefix(b, "48;5;") {
				t.Errorf("%s: 256 colours drawn as %q", k, b)
			}
			if prev, ok := to[a]; ok && prev != b {
				t.Errorf("%s: %s drawn as %s and %s", k, a, prev, b)
			}
			to[a] = b
		}
		seen := map[string]string{}
		for a, b := range to {
			if other, ok := seen[b]; ok {
				t.Errorf("%s and %s both drawn as %s", a, other, b)
			}
			seen[b] = a
		}
	}
	UsePalette(pickANSI)
	f16, b16 := measure("TERM=xterm")
	sgr := func(c string, bg bool) string { // the SGR of ANSI colour c
		n := 0
		fmt.Sscan(c, &n)
		base := 30
		if bg {
			base = 40
		}
		if n >= 8 {
			base, n = base+60, n-8
		}
		return fmt.Sprint(base + n)
	}
	for _, k := range names {
		st := paletteStyles()[k]
		if want := sgr(colourIndex(st.GetForeground()), false); f16[k] != want {
			t.Errorf("%s: 16 colours drawn as %q, want %q", k, f16[k], want)
		}
		if _, none := st.GetBackground().(lipgloss.NoColor); none {
			continue
		}
		if want := sgr(colourIndex(st.GetBackground()), true); b16[k] != want {
			t.Errorf("%s: background drawn as %q, want %q", k, b16[k], want)
		}
	}
}

// colourIndex is the ANSI index of one of the 16 colours.
func colourIndex(c color.Color) string {
	for i := range 16 {
		if lipgloss.Color(fmt.Sprint(i)) == c {
			return fmt.Sprint(i)
		}
	}
	return "?"
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

// teaOut is what Bubble Tea writes to a terminal of environment env, forced
// to be one, to show frame.
func teaOut(t *testing.T, env, frame string) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var buf bytes.Buffer
	_, err := tea.NewProgram(frameModel{frame}, tea.WithContext(ctx), tea.WithInput(nil), tea.WithOutput(&buf),
		tea.WithEnvironment([]string{"TTY_FORCE=1", env}), tea.WithWindowSize(60, 80), tea.WithoutSignals()).Run()
	if err != nil {
		t.Fatalf("%s: %v", env, err)
	}
	return buf.String()
}

type cellSt struct{ fg, bg string }

// sgrCells reads what a terminal receives: each printable rune with the
// foreground and background SGR parameters in force.
func sgrCells(out string) (text []rune, sts []cellSt) {
	var cur cellSt
	for i := 0; i < len(out); {
		if out[i] == 0x1b && i+1 < len(out) && out[i+1] == '[' {
			j := i + 2
			for j < len(out) && (out[j] < 0x40 || out[j] > 0x7e) {
				j++
			}
			if j < len(out) && out[j] == 'm' {
				ps := strings.Split(out[i+2:j], ";")
				for k := 0; k < len(ps); k++ {
					switch p := ps[k]; {
					case p == "" || p == "0":
						cur = cellSt{}
					case (p == "38" || p == "48") && k+1 < len(ps):
						n := 3
						if ps[k+1] == "5" {
							n = 1
						}
						v := strings.Join(ps[k:min(k+2+n, len(ps))], ";")
						if p == "38" {
							cur.fg = v
						} else {
							cur.bg = v
						}
						k += 1 + n
					case p == "39":
						cur.fg = ""
					case p == "49":
						cur.bg = ""
					case len(p) == 2 && (p[0] == '3' || p[0] == '9') && p[1] <= '7':
						cur.fg = p
					case len(p) == 2 && p[0] == '4' && p[1] <= '7', len(p) == 3 && p[:2] == "10" && p[2] <= '7':
						cur.bg = p
					}
				}
			}
			i = j + 1
			continue
		}
		if out[i] == 0x1b {
			i += 2
			continue
		}
		r := []rune(out[i:min(i+4, len(out))])[0]
		if r >= 0x20 {
			text = append(text, r)
			sts = append(sts, cur)
		}
		i += len(string(r))
	}
	return text, sts
}

// TestTrueColours is plan 5, item 13, criterion 4's guard: the true colours
// are ui-design.md's, Colours and symbols, whatever the 16-colour palette.
func TestTrueColours(t *testing.T) {
	for c, want := range map[*colour]string{
		&cFrame: "#3B3F51", &cTitle: "#C4B5FD", &cText: "#E5E7EB", &cDone: "#4ADE80", &cProgress: "#FBBF24",
		&cAttention: "#F472B6", &cFailed: "#F87171", &cGrey: "#6B7280", &cInk: "#111827", &cSelection: "#4C4F6B",
		&cTreeRow: "#2E3247", &cTreeText: "#E5E7EB", &cKeyword: "#A78BFA", &cString: "#86EFAC", &cNumber: "#FB923C",
		&cCode: "#93C5FD", &cLink: "#60A5FA", &modelColours[0].c: "#A78BFA", &modelColours[1].c: "#FB923C",
		&modelColours[2].c: "#60A5FA", &modelColours[3].c: "#2DD4BF",
	} {
		if c.tc != want {
			t.Errorf("%s is %s, want %s", c.ansi, c.tc, want)
		}
	}
}

// TestStyleWraps checks the wrap line.String writes around a span against
// Lip Gloss's Render, for every shared style in each palette, so that the
// cheaper path draws the same bytes (ui-design.md, Budget).
func TestStyleWraps(t *testing.T) {
	defer UsePalette(pickTrue)
	texts := []string{"x", " ", "   ", "a b", "─╭╮│", "· é", "日本語", "👍🏽 end", "␉ ␊", "\t", "a\tb", "a\nb", "a\r\nb", "\x1b"}
	for _, pick := range []lipgloss.CompleteFunc{pickTrue, pick256, pickANSI} {
		UsePalette(pick)
		if len(wraps) < 30 {
			t.Errorf("%d styles with a wrap, want most of them", len(wraps))
		}
		for _, p := range []*lipgloss.Style{stMdLink, stMatchCur} {
			if _, ok := wraps[p]; ok {
				t.Errorf("an underlined style has a wrap: Lip Gloss styles its spaces apart")
			}
		}
		for p := range wraps {
			for _, s := range texts {
				if got, want := (line{{text: s, st: p}}).String(), p.Render(s); got != want {
					t.Errorf("%q: line.String %q, Render %q", s, got, want)
				}
			}
		}
	}
}
