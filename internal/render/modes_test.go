package render

import (
	"image"
	"image/color"
	"slices"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"

	"github.com/Evengelius/agent-command-center/internal/state"
)

// The modes selector (plan 39, brief amendment 40; reference-design.md, The
// modes selector).

// modesState is plan 34's mock dashboard of the CLI, the mode in force as
// given.
func modesState(t *testing.T, cli, mode string) *state.State {
	t.Helper()
	file := "claude-models.jsonl"
	if cli == "codex" {
		file = "codex-models.jsonl"
	}
	s, _ := stampedMock(t, file, appClock)
	s.Session.ApprovalMode = mode
	return s
}

// findModes finds the selector on the screen: its rows' text by cells, from
// its top edge to its bottom edge, its left column, its top row and its
// width.
func findModes(t *testing.T, sc *Screen) (rows []string, x, y, w int) {
	t.Helper()
	frame := strings.Split(strings.TrimSuffix(texts(sc.lines), "\n"), "\n")
	y = -1
	for yy, l := range sc.lines {
		c := 0
		for _, s := range l {
			if s.ref.panel == ModesPanel && y < 0 {
				x, y = c, yy
			}
			if s.ref.panel == ModesPanel && yy == y {
				w += textWidth(s.text)
			}
			c += textWidth(s.text)
		}
		if y >= 0 {
			break
		}
	}
	if y < 0 {
		t.Fatalf("no modes selector on the screen:\n%s", texts(sc.lines))
	}
	for yy := y; yy < len(frame); yy++ {
		rows = append(rows, cellSlice(frame[yy], x, x+w))
		if strings.HasPrefix(rows[len(rows)-1], "╰") {
			return rows, x, y, w
		}
	}
	t.Fatalf("the selector has no bottom edge:\n%s", strings.Join(rows, "\n"))
	return nil, 0, 0, 0
}

// halfRow says r, a row between the box's edges, holds only half-row glyphs
// and blanks.
func halfRow(r string) bool {
	return strings.Trim(r, "│ ▀▄█") == ""
}

// TestModesCells is criterion 39.6: on both CLIs, at 120x40 and 200x60,
// with Sixel and without, "MODES" alone as the heading at column 3,
// "Current session" ending at W-3, W 51 for Claude Code and 54 for Codex,
// the titles and descriptions at column 11, a half row before each option
// and after the last, the bottom edge on the editor's rule with "▾" above
// the pill's middle, the right edge on the cell after the pill; no other
// text: with Sixel nothing in the strips, without, the icons and the check.
func TestModesCells(t *testing.T) {
	defer UsePalette(pickTrue)
	for _, cli := range []string{"claude-code", "codex"} {
		for _, size := range [][2]int{{120, 40}, {200, 60}} {
			for _, sixel := range []bool{false, true} {
				s := modesState(t, cli, state.ModeAuto)
				ui := UI{Width: size[0], Height: size[1], Now: appClock, Focus: "B", Sixel: sixel}
				ui.Modes = OpenModes(s)
				sc := Draw(s, ui)
				rows, x, y, w := findModes(t, sc)
				name := func() string {
					return strings.Join([]string{cli, map[bool]string{true: "sixel", false: "text"}[sixel]}, " ")
				}
				if want := map[string]int{"claude-code": 51, "codex": 54}[cli]; w != want {
					t.Errorf("%s %v: W %d, want %d", name(), size, w, want)
				}
				head := rows[1]
				if col(head, "MODES") != 3 || col(head, "Current session")+len("Current session") != w-3 ||
					!slices.Equal(strings.Fields(strings.Trim(head, "│")), []string{"MODES", "Current", "session"}) {
					t.Errorf("%s %v: the heading row %q", name(), size, head)
				}
				// Every option: a half row, its title, its description lines;
				// a half row after the last.
				r := 2
				var words []string
				for _, o := range s.ModeOptions() {
					if !halfRow(rows[r]) {
						t.Errorf("%s %v: row %d %q, want a half row before %s", name(), size, r, rows[r], o.Title)
					}
					r++
					if col(rows[r], o.Title) != 11 {
						t.Errorf("%s %v: %q at column %d, want 11: %q", name(), size, o.Title, col(rows[r], o.Title), rows[r])
					}
					words = append(words, strings.Fields(o.Title)...)
					if !sixel {
						words = append(words, modesTextIcon[o.Icon])
						if o.Mode == state.ModeAuto {
							words = append(words, "✓")
						}
					}
					r++
					for _, d := range o.Desc {
						if col(rows[r], d) != 11 {
							t.Errorf("%s %v: %q at column %d, want 11", name(), size, d, col(rows[r], d))
						}
						words = append(words, strings.Fields(d)...)
						r++
					}
				}
				if !halfRow(rows[r]) || r+2 != len(rows) {
					t.Errorf("%s %v: rows after the last option %q", name(), size, rows[r:])
				}
				var got []string
				for _, row := range rows[2 : len(rows)-1] {
					got = append(got, strings.Fields(strings.Trim(row, "│ ▀▄█"))...)
				}
				slices.Sort(got)
				slices.Sort(words)
				if !slices.Equal(got, words) {
					t.Errorf("%s %v: the words\n got %q\nwant %q", name(), size, got, words)
				}
				// Every cell of the box on its ground or on the band (the
				// heading's gap once fell on the terminal's own).
				for by := range rows {
					c := 0
					for _, sp := range sc.lines[y+by] {
						cw := textWidth(sp.text)
						if c >= x && c+cw <= x+w {
							if bg := sp.st.GetBackground(); bg != lipgloss.Color(cModesGround.tc) && bg != lipgloss.Color(cModesBand.tc) {
								t.Errorf("%s %v: row %d's %q at column %d on %v", name(), size, by, sp.text, c-x, bg)
							}
						}
						c += cw
					}
				}
				// The bottom edge on the editor's rule, "▾" above the pill's
				// middle, the right edge on the cell after the pill.
				px, py, ok := sc.find(ModePillPanel)
				pw := sc.spanWidth(ModePillPanel, py)
				bottom := rows[len(rows)-1]
				if !ok || y+len(rows)-1 != sc.editorRule(py) || !strings.HasPrefix(bottom, "╰") || !strings.HasSuffix(bottom, "╯") ||
					strings.Trim(bottom, "╰─▾╯") != "" || strings.Count(bottom, "▾") != 1 || x+col(bottom, "▾") != px+pw/2 || x+w-1 != px+pw {
					t.Errorf("%s %v: the bottom edge %q at row %d, column %d, the pill at %d,%d (%d cells), the rule on row %d",
						name(), size, bottom, y+len(rows)-1, x, px, py, pw, sc.editorRule(py))
				}
				if top := rows[0]; !strings.HasPrefix(top, "╭") || !strings.HasSuffix(top, "╮") || strings.Trim(top, "╭─╮") != "" || y < sc.selectorTop() {
					t.Errorf("%s %v: the top edge %q at row %d", name(), size, top, y)
				}
			}
		}
	}
}

// modesSpan is the span at the box's cell (bx, by) on the screen.
func modesSpan(sc *Screen, x, y, bx, by int) span {
	sp, _ := spanAt(sc.lines[y+by], x+bx)
	return sp
}

// TestModesHighlight is criterion 39.7: the selected option's cells on the
// band (#2E2842) from its upper half row to its lower one, its title and
// description in their colours, its check; the cursor's option on the same
// ground with its own colours and no check, one beside the other sharing
// the half row between them; the cursor back on the selected one leaves it
// alone lit; Full access amber in every state; without Sixel the strips on
// the band too, with Sixel blank, their looks given to the images.
func TestModesHighlight(t *testing.T) {
	defer UsePalette(pickTrue)
	band, plain := modesBandGround, modesGround
	for _, sixel := range []bool{false, true} {
		s := modesState(t, "claude-code", state.ModeAuto)
		ui := UI{Width: 120, Height: 40, Now: appClock, Focus: "B", Sixel: sixel}
		// The rows: 0 top, 1 heading, 2 half, 3-5 Manual, 6 half, 7-9 Edit
		// automatically, 10 half, 11-13 Plan, 14 half, 15-18 Auto, 19 half.
		for _, c := range []struct {
			cursor     int
			lit        []int // the options on the band
			halves     map[int]string
			titleStyle map[int]*lipgloss.Style
		}{
			{3, []int{3}, map[int]string{2: " ", 6: " ", 10: " ", 14: "▄", 19: "▀"},
				map[int]*lipgloss.Style{0: plain.on(stModesTitle), 2: plain.on(stModesTitle), 3: band.on(stModesSelTitle)}},
			{2, []int{2, 3}, map[int]string{10: "▄", 14: "█", 19: "▀"},
				map[int]*lipgloss.Style{2: band.on(stModesTitle), 3: band.on(stModesSelTitle)}},
			{0, []int{0, 3}, map[int]string{2: "▄", 6: "▀", 10: " ", 14: "▄"},
				map[int]*lipgloss.Style{0: band.on(stModesTitle), 1: plain.on(stModesTitle)}},
		} {
			ui.Modes = ModesView{Open: true, Cursor: c.cursor}
			sc := Draw(s, ui)
			rows, x, y, _ := findModes(t, sc)
			titleRow := map[int]int{0: 3, 1: 7, 2: 11, 3: 15}
			for opt, by := range titleRow {
				lit := slices.Contains(c.lit, opt)
				for k := range map[int]int{0: 3, 1: 3, 2: 3, 3: 4}[opt] { // its title and description rows
					sp := modesSpan(sc, x, y, 11, by+k)
					ground := plain
					if lit {
						ground = band
					}
					want := ground.on(stModesDesc)
					switch {
					case k == 0 && c.titleStyle[opt] != nil:
						want = c.titleStyle[opt]
					case k == 0 && opt == 3:
						want = ground.on(stModesSelTitle)
					case k == 0:
						want = ground.on(stModesTitle)
					case opt == 3:
						want = ground.on(stModesSelDesc)
					}
					if sp.st != want {
						t.Errorf("sixel %v, cursor %d: option %d row %d styled %v, want %v", sixel, c.cursor, opt, k, sp.st, want)
					}
					// The strips: blank on the box's ground with Sixel, on the
					// option's ground without.
					left := modesSpan(sc, x, y, 1, by+k)
					if sixel && (left.st != plain.on(stNone) || strings.TrimSpace(left.text) != "") || !sixel && left.st != ground.on(stModesIcon) {
						t.Errorf("sixel %v, cursor %d: option %d row %d's left strip %q styled %v", sixel, c.cursor, opt, k, left.text, left.st)
					}
				}
				check := strings.Contains(rows[by], "✓")
				if check != (!sixel && opt == 3) {
					t.Errorf("sixel %v, cursor %d: option %d's check %v", sixel, c.cursor, opt, check)
				}
			}
			for by, glyph := range c.halves {
				mid := modesSpan(sc, x, y, 11, by)
				if strings.Trim(mid.text, glyph) != "" || mid.st != plain.on(stModesHalf) {
					t.Errorf("sixel %v, cursor %d: half row %d %q styled %v, want %q", sixel, c.cursor, by, mid.text, mid.st, glyph)
				}
			}
			if sixel {
				var sel, cur []int
				for i, o := range sc.Modes.Left.Options {
					if o.Selected {
						sel = append(sel, i)
					}
					if o.Cursor {
						cur = append(cur, i)
					}
					if o.Row != titleRow[i]-2 || o.Rows != map[int]int{0: 3, 1: 3, 2: 3, 3: 4}[i] {
						t.Errorf("cursor %d: strip option %d at %d, %d rows", c.cursor, i, o.Row, o.Rows)
					}
				}
				wantCur := slices.DeleteFunc(slices.Clone(c.lit), func(i int) bool { return i == 3 })
				if !slices.Equal(sel, []int{3}) || !slices.Equal(cur, wantCur) || !slices.EqualFunc(sc.Modes.Left.Options, sc.Modes.Right.Options, func(a, b StripOption) bool { return a == b }) {
					t.Errorf("cursor %d: strips selected %v, cursor %v", c.cursor, sel, cur)
				}
			}
		}
	}
	// A greyed option is never lit, even under the cursor, its title in the
	// off colour.
	{
		s := modesState(t, "codex", state.ModeManual)
		s.Session.ModesUnavailable = map[string]string{state.ModeFull: "not allowed"}
		sc := Draw(s, UI{Width: 120, Height: 40, Now: appClock, Focus: "B", Modes: ModesView{Open: true, Cursor: 2}})
		_, x, y, _ := findModes(t, sc)
		if sp := modesSpan(sc, x, y, 11, 11); sp.st != plain.on(stModesOff) || !strings.HasPrefix(sp.text, "Full access") {
			t.Errorf("the greyed Full access under the cursor: %q styled %v", sp.text, sp.st)
		}
		if sp := modesSpan(sc, x, y, 11, 10); strings.Trim(sp.text, " ") != "" {
			t.Errorf("the half row above the greyed option %q", sp.text)
		}
	}
	// Full access amber, unlit, under the cursor and selected.
	for _, c := range []struct {
		mode   string
		cursor int
	}{{state.ModeAuto, 1}, {state.ModeAuto, 2}, {state.ModeFull, 2}} {
		s := modesState(t, "codex", c.mode)
		sc := Draw(s, UI{Width: 120, Height: 40, Now: appClock, Focus: "B", Modes: ModesView{Open: true, Cursor: c.cursor}})
		_, x, y, _ := findModes(t, sc)
		ground := plain
		if c.cursor == 2 || c.mode == state.ModeFull {
			ground = band
		}
		if sp := modesSpan(sc, x, y, 11, 11); sp.st != ground.on(stModesWarn) || !strings.HasPrefix(sp.text, "Full access") {
			t.Errorf("%s, cursor %d: Full access %q styled %v", c.mode, c.cursor, sp.text, sp.st)
		}
		if sp := modesSpan(sc, x, y, 4, 11); sp.st != ground.on(stModesWarn) {
			t.Errorf("%s, cursor %d: Full access's icon styled %v", c.mode, c.cursor, sp.st)
		}
	}
}

// TestModesConfirm is design D4's box: "TURN ON FULL ACCESS?" in the warning
// colour in the heading's place, Turn on Full access with what it allows,
// then Cancel keeping the mode in force, the cursor on Cancel, its rows
// lit; no check; the box as wide, its bottom on the rule.
func TestModesConfirm(t *testing.T) {
	defer UsePalette(pickTrue)
	s := modesState(t, "codex", state.ModeManual)
	v := OpenConfirm(s)
	if v.Cursor != 1 || ModesMode(s, v, 0) != state.ModeFull || ModesMode(s, v, 1) != "" || ModesCount(s, v) != 2 {
		t.Fatalf("the confirmation %+v: %q, %q", v, ModesMode(s, v, 0), ModesMode(s, v, 1))
	}
	sc := Draw(s, UI{Width: 120, Height: 40, Now: appClock, Focus: "B", Modes: v})
	rows, x, y, w := findModes(t, sc)
	want := []string{"TURN ON FULL ACCESS? Current session", "", "⚠ Turn on Full access", "Codex will run commands and edit", "any file on this computer, with",
		"internet access, without asking.", "", "✋ Cancel", "Keep Ask for approval", ""}
	var got []string
	for _, r := range rows[1 : len(rows)-1] {
		got = append(got, strings.Join(strings.Fields(strings.Trim(r, "│ ▀▄█")), " "))
	}
	if w != 54 || !slices.Equal(got, want) {
		t.Errorf("W %d, rows\n got %q\nwant %q", w, got, want)
	}
	if sp := modesSpan(sc, x, y, 3, 1); sp.st != modesGround.on(stModesWarnHead) {
		t.Errorf("the heading styled %v", sp.st)
	}
	if sp := modesSpan(sc, x, y, 11, 3); sp.st != modesGround.on(stModesWarn) {
		t.Errorf("Turn on Full access styled %v", sp.st)
	}
	if sp := modesSpan(sc, x, y, 11, 8); sp.st != modesBandGround.on(stModesTitle) {
		t.Errorf("Cancel styled %v", sp.st)
	}
	_, py, _ := sc.find(ModePillPanel)
	if y+len(rows)-1 != sc.editorRule(py) {
		t.Errorf("the bottom edge on row %d, the rule on %d", y+len(rows)-1, sc.editorRule(py))
	}
}

// TestModePillRow is criterion 39.9: the editor row's right end is the mode
// pill of the mode in force and one cell: without Sixel its icon and title
// on the pill's ground, with Sixel as many blank cells as the image takes,
// its look given to the application; Full access in the warning colour;
// no chevron; the icon alone where a middle text leaves no room for the
// title, and the row alone where it leaves none for the icon.
func TestModePillRow(t *testing.T) {
	defer UsePalette(pickTrue)
	// cells: the image's, 24 pixels before the title, 8 a character, 10
	// after, rounded up to the cell.
	for _, c := range []struct {
		cli, mode, text string
		look            ModePillLook
		cells           int
	}{
		{"claude-code", state.ModeAuto, " ↯ Auto ", ModePillLook{Icon: state.IconBolt, Title: "Auto"}, 7},
		{"claude-code", state.ModeManual, " ✋ Manual ", ModePillLook{Icon: state.IconHand, Title: "Manual"}, 9},
		{"claude-code", state.ModeAcceptEdits, " </> Edit automatically ", ModePillLook{Icon: state.IconCode, Title: "Edit automatically"}, 18},
		{"claude-code", state.ModePlan, " ▤ Plan ", ModePillLook{Icon: state.IconPlan, Title: "Plan"}, 7},
		{"claude-code", "bypassPermissions", " bypassPermissions ", ModePillLook{Title: "bypassPermissions"}, 17},
		{"codex", state.ModeManual, " ✋ Ask for approval ", ModePillLook{Icon: state.IconHand, Title: "Ask for approval"}, 17},
		{"codex", state.ModeAuto, " ⛨ Approve for me ", ModePillLook{Icon: state.IconShield, Title: "Approve for me"}, 15},
		{"codex", state.ModeFull, " ⚠ Full access ", ModePillLook{Icon: state.IconHexagon, Title: "Full access", Warning: true}, 13},
		{"codex", state.ModeCustom, " Custom ", ModePillLook{Title: "Custom"}, 9},
	} {
		if n := ModePillCells(c.look); n != c.cells {
			t.Errorf("%s %s: the image pill %d cells, want %d", c.cli, c.mode, n, c.cells)
		}
		s := modesState(t, c.cli, c.mode)
		if got := modePillLook(s); got != c.look {
			t.Errorf("%s %s: look %+v, want %+v", c.cli, c.mode, got, c.look)
		}
		row := modePill(s, false, false)
		if got := texts([]line{row}); strings.TrimSuffix(got, "\n") != c.text+" " || strings.ContainsAny(got, "▾⌄˅") {
			t.Errorf("%s %s: the text pill %q, want %q and a cell", c.cli, c.mode, got, c.text)
		}
		if row[0].ref.panel != ModePillPanel || row[len(row)-1].ref.panel == ModePillPanel {
			t.Errorf("%s %s: the pill's cells %+v", c.cli, c.mode, row)
		}
		six := modePill(s, true, false)
		if got := strings.TrimSuffix(texts([]line{six}), "\n"); got != strings.Repeat(" ", ModePillCells(c.look)+1) || six[0].ref.panel != ModePillPanel {
			t.Errorf("%s %s: the image pill's cells %q", c.cli, c.mode, got)
		}
		ui := UI{Width: 200, Height: 60, Now: appClock, Focus: "B", Sixel: true}
		if sc := Draw(s, ui); !sc.ModePill.Shown || sc.ModePill.Look != c.look || sc.spanWidth(ModePillPanel, sc.ModePill.Y) != ModePillCells(c.look) {
			t.Errorf("%s %s: the frame's pill %+v", c.cli, c.mode, sc.ModePill)
		}
	}
	// The icon alone, then the row alone, as the middle grows.
	s := modesState(t, "codex", state.ModeManual)
	left := 1 + 3 + 3 + modelPill(s).width()
	full, icon := modePill(s, false, false).width(), modePill(s, false, true).width()
	for _, c := range []struct {
		n    int
		want string
	}{{80 - left - 6 - full, "Ask for approval"}, {80 - left - 6 - full + 1, "✋"}, {80 - left - 6 - icon, "✋"}, {80 - left - 6 - icon + 1, ""}} {
		middle := strings.Repeat("x", c.n)
		got := strings.TrimRight(texts([]line{inputRow(s, middle, 80, false, false)}), " \n")
		switch {
		case c.want == "" && got != "  "+middle:
			t.Errorf("a middle of %d: %q, want the row alone", c.n, got)
		case c.want != "" && !strings.HasSuffix(got, c.want):
			t.Errorf("a middle of %d: %q, want it ending with %q", c.n, got, c.want)
		}
	}
	compact := ModePillLook{Icon: state.IconHand}
	if ModePillCells(compact) != 3 {
		t.Errorf("the compact pill's cells %d", ModePillCells(compact))
	}
}

// TestModesRoom is criterion 39.11's frame: the box only where its rows fit
// between zone B's title and the editor's rule, and nothing of it drawn
// where they do not; none at 80x24, where neither CLI's fits; Claude Code's
// 21 rows not at 120x40 beside a tall draft, Codex's 16 there.
func TestModesRoom(t *testing.T) {
	for _, cli := range []string{"claude-code", "codex"} {
		s := modesState(t, cli, state.ModeManual)
		for _, c := range []struct {
			w, h  int
			lines int
			fits  bool
		}{{80, 24, 1, false}, {120, 40, 1, true}, {120, 40, 30, cli == "codex"}, {120, 30, 1, cli == "codex"}, {200, 60, 1, true}, {200, 60, 40, true}} {
			var draft []string
			for i := range c.lines {
				draft = append(draft, strings.Repeat("d", i+1))
			}
			ui := UI{Width: c.w, Height: c.h, Now: appClock, Focus: "B", Editor: EditorView{Lines: draft, Total: len(draft)}}
			plain := Draw(s, ui)
			ui.Modes = OpenModes(s)
			sc := Draw(s, ui)
			if plain.ModesFit(s, ui.Modes, c.w) != c.fits || sc.Modes.Shown != c.fits {
				t.Errorf("%s %dx%d, %d lines: fits %v, shown %v, want %v", cli, c.w, c.h, c.lines, plain.ModesFit(s, ui.Modes, c.w), sc.Modes.Shown, c.fits)
			}
			if !c.fits && texts(sc.lines) != texts(plain.lines) {
				t.Errorf("%s %dx%d, %d lines: something of the box drawn", cli, c.w, c.h, c.lines)
			}
		}
	}
}

// near says the pixel p is within d of colour c on each channel, opaque.
func nearColour(p, c color.NRGBA, d int) bool {
	diff := func(a, b uint8) int { return max(int(a)-int(b), int(b)-int(a)) }
	return p.A == 255 && diff(p.R, c.R) <= d && diff(p.G, c.G) <= d && diff(p.B, c.B) <= d
}

// countNear counts the pixels of img within r near colour c.
func countNear(img *image.NRGBA, r image.Rectangle, c color.NRGBA, d int) int {
	n := 0
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			if nearColour(img.NRGBAAt(x, y), c, d) {
				n++
			}
		}
	}
	return n
}

// TestModesStripImages is criterion 39.8's strips: opaque, the box's ground
// outside the lit options; each lit option's band from the middle of the
// half row above its title to the middle of the one below its last line,
// its outer corners rounded, its inner edge the band's colour so it meets
// the text cells' ground; the icons in their colours at their places, the
// selected one's in its own, Full access's amber; the white check on the
// selected option's title row alone.
func TestModesStripImages(t *testing.T) {
	opts := []StripOption{
		{Row: 1, Rows: 3, Icon: state.IconHand},
		{Row: 5, Rows: 3, Icon: state.IconShield, Selected: true},
		{Row: 9, Rows: 3, Icon: state.IconHexagon, Warning: true, Cursor: true},
	}
	left, right := StripImage(StripLook{Rows: 13, Options: opts}), StripImage(StripLook{Rows: 13, Right: true, Options: opts})
	if left.Bounds() != image.Rect(0, 0, 100, 260) || right.Bounds() != image.Rect(0, 0, 50, 260) {
		t.Fatalf("bounds %v %v", left.Bounds(), right.Bounds())
	}
	for _, img := range []*image.NRGBA{left, right} {
		for i := 3; i < len(img.Pix); i += 4 {
			if img.Pix[i] != 255 {
				t.Fatalf("a transparent pixel at %d", i/4)
			}
		}
	}
	at := func(img *image.NRGBA, x, y int) color.NRGBA { return img.NRGBAAt(x, y) }
	// Option 0, unlit: the ground through its rows, but for its icon.
	for _, p := range [][2]int{{2, 30}, {90, 30}, {99, 50}} {
		if c := at(left, p[0], p[1]); c != stripGround {
			t.Errorf("the unlit option at %v: %v", p, c)
		}
	}
	// Option 1, selected: the band from y 90 (the middle of row 4) to y 170
	// (the middle of row 8), option 2's, the cursor's, from there to y 250;
	// the inner edges the band's colour; rounded.
	for _, c := range []struct {
		img  *image.NRGBA
		x, y int
		want color.NRGBA
	}{
		{left, 99, 91, stripBand}, {left, 99, 169, stripBand}, {left, 60, 130, stripBand}, {left, 99, 88, stripGround}, {left, 99, 172, stripBand}, {left, 99, 251, stripGround},
		{left, 5, 90, stripGround}, {left, 4, 130, stripGround}, {left, 7, 130, stripBand}, // the rounded corner, the left edge at x 5
		{right, 0, 91, stripBand}, {right, 30, 130, stripBand}, {right, 44, 130, stripBand}, {right, 46, 130, stripGround}, {right, 44, 90, stripGround},
	} {
		if got := at(c.img, c.x, c.y); got != c.want {
			t.Errorf("(%d, %d) of the %s strip: %v, want %v", c.x, c.y, map[bool]string{true: "left", false: "right"}[c.img == left], got, c.want)
		}
	}
	// The icons: each in its box around (49, row*20+20), its colour.
	for i, c := range []struct{ col color.NRGBA }{{stripIcon}, {stripIconSel}, {stripWarn}} {
		cy := opts[i].Row*20 + 20
		box := image.Rect(31, cy-18, 67, cy+18)
		if n := countNear(left, box, c.col, 12); n < 25 {
			t.Errorf("option %d's icon: %d pixels near %v", i, n, c.col)
		}
	}
	if n := countNear(left, image.Rect(31, 82, 67, 118), stripWarn, 12); n != 0 {
		t.Errorf("the shield has %d amber pixels", n)
	}
	// The check: on the selected option's title row alone.
	if n := countNear(right, image.Rect(0, 100, 50, 120), stripCheck, 12); n < 15 {
		t.Errorf("the selected option's check: %d pixels", n)
	}
	for _, r := range []image.Rectangle{image.Rect(0, 0, 50, 100), image.Rect(0, 121, 50, 260)} {
		if n := countNear(right, r, stripCheck, 40); n != 0 {
			t.Errorf("check pixels in %v: %d", r, n)
		}
	}
	// A greyed option's icon in the off colour, no band.
	off := StripImage(StripLook{Rows: 5, Options: []StripOption{{Row: 1, Rows: 3, Icon: state.IconHexagon, Off: true, Cursor: false}}})
	if n := countNear(off, image.Rect(31, 22, 67, 58), stripOff, 12); n < 25 || countNear(off, off.Bounds(), stripBand, 0) != 0 {
		t.Errorf("the greyed option: %d pixels near the off colour", n)
	}
}

// TestModePillImage is criterion 39.8's pill: transparent around its
// rounded corners, the outline's colour on its edge, the fill inside, the
// icon in its colour at the left (the lightning's purple, Full access's
// amber), the title's pixels where the label puts them and the fill where
// it has none; the compact pill's icon in its middle.
func TestModePillImage(t *testing.T) {
	label := image.NewAlpha(image.Rect(0, 0, 36, 20))
	for y := 6; y < 15; y++ {
		for x := 2; x < 30; x++ {
			label.SetAlpha(x, y, color.Alpha{A: 255})
		}
	}
	look := ModePillLook{Icon: state.IconBolt, Title: "Auto"}
	img := PillImage(look, label)
	w := ModePillCells(look) * SixelCellW
	if img.Bounds() != image.Rect(0, 0, w, SixelCellH) || w != 70 {
		t.Fatalf("bounds %v", img.Bounds())
	}
	for _, p := range [][2]int{{0, 0}, {w - 1, 0}, {0, 19}, {w - 1, 19}, {1, 1}, {30, 0}} {
		if a := img.NRGBAAt(p[0], p[1]).A; a != 0 {
			t.Errorf("(%d, %d) not transparent: %v", p[0], p[1], img.NRGBAAt(p[0], p[1]))
		}
	}
	// The outline: the edges, and at (1, 4) the corner's arc, radius 6.5 (a
	// half circle's would leave it out).
	for _, p := range [][2]int{{1, 10}, {w - 2, 10}, {35, 1}, {1, 4}} {
		if c := img.NRGBAAt(p[0], p[1]); !nearColour(c, pillOutline, 40) {
			t.Errorf("(%d, %d) %v, want near the outline %v", p[0], p[1], c, pillOutline)
		}
	}
	if c := img.NRGBAAt(w-5, 10); c != pillFill {
		t.Errorf("the fill %v", c)
	}
	if n := countNear(img, image.Rect(5, 2, 21, 18), pillBolt, 12); n < 15 {
		t.Errorf("the lightning: %d pixels", n)
	}
	if c := img.NRGBAAt(pillTextX+10, 10); !nearColour(c, pillLabel, 2) {
		t.Errorf("the label's pixel %v, want %v", c, pillLabel)
	}
	if c := PillImage(look, nil).NRGBAAt(pillTextX+10, 10); c != pillFill {
		t.Errorf("no label: %v", c)
	}
	full := PillImage(ModePillLook{Icon: state.IconHexagon, Title: "Full access", Warning: true}, label)
	if n := countNear(full, image.Rect(5, 2, 21, 18), stripWarn, 12); n < 15 || !nearColour(full.NRGBAAt(pillTextX+10, 10), stripWarn, 2) {
		t.Errorf("Full access: %d amber icon pixels, the label %v", n, full.NRGBAAt(pillTextX+10, 10))
	}
	compact := PillImage(ModePillLook{Icon: state.IconBolt}, nil)
	if compact.Bounds().Dx() != 30 || countNear(compact, image.Rect(7, 2, 23, 18), pillBolt, 12) < 15 || countNear(compact, image.Rect(0, 0, 6, 20), pillBolt, 40) != 0 {
		t.Errorf("the compact pill: %v", compact.Bounds())
	}
}
