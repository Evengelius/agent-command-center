package render

import (
	"image"
	"image/color"
	"math"
	"strings"
	"testing"

	"github.com/Evengelius/agent-command-center/internal/state"
)

// Plan 51, designs D1 and D2: picture mode from a design scale of 0.875
// with Sixel, the banner's rows between the header and the panels, drawn
// from 8; grid mode without Sixel or below the scale, its banner above the
// panels' least height with Sixel. The seven sizes of D2 and the edges of
// picture mode.
func TestHomeLayout(t *testing.T) {
	for _, c := range []struct {
		w, h    int
		sixel   bool
		picture bool
		rows    int // the banner's rows, -1 when hidden
	}{
		{208, 59, true, true, 26}, {150, 40, true, true, 16}, {120, 40, true, true, 20}, {120, 30, true, true, 10},
		{120, 26, true, true, -1}, {100, 40, true, false, 22}, {80, 24, true, false, -1},
		{208, 59, false, false, -1}, {118, 40, true, true, 21}, {117, 40, true, false, 22}, {100, 30, true, false, 12}, {100, 25, true, false, -1},
	} {
		g := HomeLayout(c.w, c.h, c.sixel)
		if g.Picture != c.picture || g.W != c.w || g.H != c.h {
			t.Errorf("%dx%d sixel %v: picture %v, want %v", c.w, c.h, c.sixel, g.Picture, c.picture)
			continue
		}
		if !c.picture && g.K != 0 {
			t.Errorf("%dx%d grid: k %v", c.w, c.h, g.K)
		}
		if c.rows < 0 {
			if g.Banner || g.BannerRows >= bannerRows {
				t.Errorf("%dx%d: banner %v, %d rows; want none", c.w, c.h, g.Banner, g.BannerRows)
			}
		} else if !g.Banner || g.BannerRows != c.rows {
			t.Errorf("%dx%d: banner %v, %d rows; want %d", c.w, c.h, g.Banner, g.BannerRows, c.rows)
		}
		if c.picture && g.K < pictureScale {
			t.Errorf("%dx%d: picture mode at k %v", c.w, c.h, g.K)
		}
	}
	// The scale the width or the height allows, the picture centred.
	g := HomeLayout(208, 59, true)
	if math.Abs(g.K-2080.0/1347) > 1e-9 || g.OX != 0 {
		t.Errorf("208x59: k %v, x %v", g.K, g.OX)
	}
	g = HomeLayout(300, 40, true)
	if k := 800.0 / 427; math.Abs(g.K-k) > 1e-9 || math.Abs(g.OX-(3000-1347*k)/2) > 1e-9 || g.Banner {
		t.Errorf("300x40: k %v, x %v, banner %v", g.K, g.OX, g.Banner)
	}
	// The header from the top, the panels and the footer from the bottom.
	g = HomeLayout(150, 40, true)
	if g.Y(45) != 45*g.K || math.Abs(g.Y(757)-800) > 1e-9 || math.Abs(g.Y(380.5)-(800-(757-380.5)*g.K)) > 1e-9 {
		t.Errorf("150x40: y 45 at %v, 757 at %v, 380.5 at %v", g.Y(45), g.Y(757), g.Y(380.5))
	}
}

// homeTestUI is the design's form: its seven projects, the first selected.
func homeTestUI() (*state.State, UI) {
	s := &state.State{}
	s.Startup.Phase = state.Asking
	s.Startup.Installed = []string{"claude-code", "codex"}
	s.Startup.CLIs = []state.CLI{{ID: "claude-code", Name: "Claude Code", Version: "2.1.303", Installed: true}, {ID: "codex", Name: "OpenAI Codex", Version: "0.163.0", Installed: true}}
	ui := UI{Width: 208, Height: 59, Sixel: true, Version: "0.1.0", System: "Windows", Choice: "claude-code", Path: `C:\Projects\agent-command-center`}
	for i, n := range []string{"agent-command-center", "personal-assistant", "code-workflow", "marketing-site", "data-pipeline", "research-notes", "automation-lab"} {
		ui.Projects = append(ui.Projects, ProjectRow{Path: `C:\Projects\` + n, Age: "2h ago", Favourite: i < 3})
	}
	return s, ui
}

// Plan 51, design D7: FAVORITES then RECENT PROJECTS, each project once,
// at the design's places; a section with no project hidden; more projects
// than fit, the list follows the selection.
func TestHomeRows(t *testing.T) {
	_, ui := homeTestUI()
	rows, _ := homeRows(ui, rowFirstHead)
	want := []float64{438.5, 468.23, 498.32, 528.41, 570.33, 600.06, 628.76, 657.46, 686.16}
	heads := map[int]byte{0: 'f', 4: 'r'}
	if len(rows) != len(want) {
		t.Fatalf("%d rows, want %d", len(rows), len(want))
	}
	for i, r := range rows {
		if math.Abs(r.Y-want[i]) > 0.01 || r.Head != heads[i] {
			t.Errorf("row %d: y %.2f head %q, want %.2f %q", i, r.Y, r.Head, want[i], heads[i])
		}
	}
	if rows[1].Name != "agent-command-center" || rows[1].Path != `C:\Projects\agent-command-center` || !rows[1].Fav || !rows[1].Selected || rows[2].Selected {
		t.Errorf("first project: %+v", rows[1])
	}
	if rows[5].Name != "marketing-site" || rows[5].Fav {
		t.Errorf("first recent project: %+v", rows[5])
	}
	// No favourite: no FAVORITES heading.
	for i := range ui.Projects {
		ui.Projects[i].Favourite = false
	}
	rows, _ = homeRows(ui, rowFirstHead)
	if rows[0].Head != 'r' || rows[0].Y != rowFirstHead || len(rows) != 8 {
		t.Errorf("no favourite: first row %+v, %d rows", rows[0], len(rows))
	}
	// Twenty recent projects: the last selected shows, at most down to
	// rowLast, the rows above it before.
	ui.Projects = nil
	for i := range 20 {
		ui.Projects = append(ui.Projects, ProjectRow{Path: `C:\p\` + string(rune('a'+i)), Age: "now"})
	}
	ui.ProjectSel = 19
	rows, _ = homeRows(ui, rowFirstHead)
	last := rows[len(rows)-1]
	if !last.Selected || last.Name != "t" || last.Y > rowLast || rows[0].Y != rowFirstHead || rows[0].Head != 0 {
		t.Errorf("twenty, the last selected: first %+v, last %+v", rows[0], last)
	}
	ui.ProjectSel = 0
	if rows, _ = homeRows(ui, rowFirstHead); rows[0].Head != 'r' || !rows[1].Selected || rows[len(rows)-1].Y > rowLast {
		t.Errorf("twenty, the first selected: first %+v, second %+v", rows[0], rows[1])
	}
}

// Plan 51, design D4: in picture mode every cell of the frame is blank on
// the page's ground but the project field's, which hold its text, its
// cursor with the focus, a long path cut at its start.
func TestHomePictureFrame(t *testing.T) {
	s, ui := homeTestUI()
	ui.Field = 1
	g := HomeLayout(ui.Width, ui.Height, true)
	l := homeLook(s, ui, g)
	ls := homePicture(ui, l)
	if len(ls) != ui.Height {
		t.Fatalf("%d lines", len(ls))
	}
	text := strings.Split(rowsText(ls), "\n")
	c0, c1 := l.FieldCols[0], l.FieldCols[1]
	for y, row := range text[:ui.Height] {
		rs := []rune(row)
		if len(rs) != ui.Width {
			t.Fatalf("row %d: %d cells", y, len(rs))
		}
		for x, r := range rs {
			inField := y == l.FieldRow && x >= c0 && x < c1
			if !inField && r != ' ' {
				t.Fatalf("cell %d,%d holds %q outside the field", x, y, r)
			}
		}
	}
	if got := string([]rune(text[l.FieldRow])[c0:c1]); !strings.HasPrefix(got, ui.Path+"█") {
		t.Errorf("field: %q", got)
	}
	ui.Path = `C:\` + strings.Repeat("deep\\", 30) + "project"
	ls = homePicture(ui, l)
	got := strings.TrimRight(string([]rune(strings.Split(rowsText(ls), "\n")[l.FieldRow])[c0:c1]), " ")
	if !strings.HasPrefix(got, "…") || !strings.HasSuffix(got, "project█") || len([]rune(got)) != c1-c0 {
		t.Errorf("long path: %q (%d cells, field %d)", got, len([]rune(got)), c1-c0)
	}
	// The field's cells on its ground, the others on the page's.
	for _, x := range ls[l.FieldRow] {
		if strings.TrimSpace(x.text) == "" {
			continue
		}
		if x.st != fieldGround.on(stHomePath) && x.st != fieldGround.on(stTitle) {
			t.Errorf("%q not on the field's ground", x.text)
		}
	}
}

// boxGlyphs are a stand-in for GDI: each glyph a block filling its cell's
// middle, the capitals' rows from em/4 to 3em/4, its advance 0.6 em.
type boxGlyphs struct{ asked int }

func (b *boxGlyphs) Glyph(r rune, em int, bold bool) *image.Alpha {
	b.asked++
	w := max(em*3/5, 1)
	m := image.NewAlpha(image.Rect(0, 0, w, em*3/2))
	for y := em / 4; y < em*3/4; y++ {
		for x := 1; x < w-1; x++ {
			m.SetAlpha(x, y, color.Alpha{0xFF})
		}
	}
	return m
}

func nrgbaAt(m *image.NRGBA, x, y float64) color.NRGBA { return m.NRGBAAt(int(x), int(y)) }

// Plan 51, step 2: the picture's places and colours, the design's scaled
// (reference-design.md, The home screen); a clip drawn alone.
func TestHomePaint(t *testing.T) {
	s, ui := homeTestUI()
	g := HomeLayout(ui.Width, ui.Height, true)
	l := homeLook(s, ui, g)
	pic := image.NewNRGBA(g.Pixels())
	HomePaint(pic, l, &boxGlyphs{}, pic.Rect)
	at := func(x, y float64) color.NRGBA { return nrgbaAt(pic, g.X(x), g.Y(y)) }
	for _, c := range []struct {
		what string
		x, y float64
		want colour
	}{
		{"the page", 3, 3, cHomePage},
		{"the page between the panels", 623, 550, cHomePage},
		{"the chosen option's inside", 300, 476.8 + 12, cHomeChosenFill},
		{"the other option's inside", 300, 519.8 + 12, cHomePage},
		{"START SESSION", 100, 660, cHomeButton},
		{"the band", 1100, 468.23 + 10, cHomeBand},
		{"a favourite's star", 662.6, 468.23 + 0.35, cHomeStarOn},
		{"a key cap's inside", 145, 745, cHomeCapFill},
	} {
		if got, want := at(c.x, c.y), hexColour(c.want.tc); got != want {
			t.Errorf("%s at %v,%v: %v, want %v", c.what, c.x, c.y, got, want)
		}
	}
	// The banner's ground in its middle, away from the dots and stars; the
	// page's there without a banner.
	mid := int((g.Y(45) + g.Y(375)) / 2)
	ground := map[color.NRGBA]int{}
	for x := 0; x < pic.Rect.Dx(); x++ {
		ground[pic.NRGBAAt(x, mid)]++
	}
	if n := ground[hexColour(cHomeBanner.tc)]; n < pic.Rect.Dx()*9/10 {
		t.Errorf("the banner's middle row: %d of %d pixels its ground", n, pic.Rect.Dx())
	}
	nb := g
	nb.Banner = false
	lb := l
	lb.Geom = nb
	plain := image.NewNRGBA(g.Pixels())
	HomePaint(plain, lb, &boxGlyphs{}, plain.Rect)
	if got := plain.NRGBAAt(int(g.X(600)), mid); got != hexColour(cHomePage.tc) {
		t.Errorf("no banner: %v in its place", got)
	}
	// The title's first glyph in its first cell, centred; the status ends
	// where the design's does.
	if c := at(45.54+tkTitle.adv/2, 18.88); c != hexColour(cHomeTitle.tc) {
		t.Errorf("the title's first glyph: %v", c)
	}
	if c := at(1317.04-tkHead.adv/2, 18.84); c != hexColour(cHomeStatus.tc) {
		t.Errorf("the status's last glyph: %v", c)
	}
	// A clip draws nothing outside it, and inside it what the whole draws.
	clip := image.Rect(int(g.X(640)), int(g.Y(450)), int(g.X(900)), int(g.Y(520)))
	part := image.NewNRGBA(g.Pixels())
	HomePaint(part, l, &boxGlyphs{}, clip)
	for y := 0; y < part.Rect.Dy(); y += 7 {
		for x := 0; x < part.Rect.Dx(); x += 7 {
			p := image.Pt(x, y)
			if got := part.NRGBAAt(x, y); p.In(clip) && got != pic.NRGBAAt(x, y) {
				t.Fatalf("inside the clip at %v: %v, the whole %v", p, got, pic.NRGBAAt(x, y))
			} else if !p.In(clip) && got != (color.NRGBA{}) {
				t.Fatalf("outside the clip at %v: %v", p, got)
			}
		}
	}
}

// Plan 51, design D13: the picture's cells but the field's text; a
// change's cells, near runs joined; a change on the last rows grown to
// whole bands of the last three (no scroll, step 2).
func TestHomeTiles(t *testing.T) {
	l := HomeLook{Geom: HomeGeom{W: 100, H: 30, Picture: true}, FieldRow: 20, FieldCols: [2]int{4, 50}}
	got := HomeRects(l)
	want := []image.Rectangle{image.Rect(0, 0, 100, 20), image.Rect(0, 20, 4, 21), image.Rect(50, 20, 100, 21), image.Rect(0, 21, 100, 27), image.Rect(0, 27, 100, 30)}
	if len(got) != len(want) {
		t.Fatalf("rects %v", got)
	}
	cells := 0
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("rect %d: %v, want %v", i, got[i], want[i])
		}
		cells += got[i].Dx() * got[i].Dy()
	}
	if cells != 100*30-46 {
		t.Errorf("%d cells covered", cells)
	}
	old := image.NewNRGBA(image.Rect(0, 0, 100*SixelCellW, 30*SixelCellH))
	now := image.NewNRGBA(old.Rect)
	mark := func(cx, cy int) {
		now.SetNRGBA(cx*SixelCellW+3, cy*SixelCellH+5, color.NRGBA{1, 2, 3, 255})
	}
	mark(10, 5)
	mark(12, 5) // two apart: joined
	mark(16, 5) // four apart: a run of its own
	mark(30, 20)
	changes := HomeChanges(old, now, got)
	wantC := []image.Rectangle{image.Rect(10, 5, 13, 6), image.Rect(16, 5, 17, 6)}
	if len(changes) != 2 || changes[0] != wantC[0] || changes[1] != wantC[1] {
		t.Errorf("changes %v, want %v (the field's cell 30,20 left out)", changes, wantC)
	}
	for _, c := range []struct{ in, out image.Rectangle }{
		{image.Rect(5, 28, 9, 29), image.Rect(5, 27, 9, 30)},
		{image.Rect(5, 25, 9, 28), image.Rect(5, 25, 9, 30)},
		{image.Rect(5, 10, 9, 11), image.Rect(5, 10, 9, 11)},
	} {
		if got := BottomBlock(c.in, 30); got != c.out {
			t.Errorf("BottomBlock(%v): %v, want %v", c.in, got, c.out)
		}
	}
}

// Plan 51, design D13: a tile keeps its commonest colours exact, so a
// ground drawn in two tiles is the same colour in both, whatever else
// they hold; images with 255 colours or fewer as before.
func TestHomeSixelExact(t *testing.T) {
	// The ground on the first two cells; two hundred shades near it on the
	// next twenty, a column each; 2,700 bright colours on the last
	// eighteen: the median cut alone merges the ground with its neighbours
	// (to 9,11,9), which the commonest colours kept exact prevent.
	pic := image.NewNRGBA(image.Rect(0, 0, 400, 60))
	ground := color.NRGBA{9, 11, 12, 255}
	for y := range 60 {
		for x := range 400 {
			c := ground
			switch {
			case x >= 220:
				k := (x-220)/2*30 + y/2
				c = color.NRGBA{uint8(60 + k%60*3), uint8(40 + (k/60)%45*4), uint8(180 + k%7), 255}
			case x >= 20:
				k := x - 20
				c = color.NRGBA{uint8(5 + k%9), uint8(7 + (k/9)%9), uint8(8 + (k/81)%9), 255}
				if c == ground {
					c.B = 40
				}
			}
			pic.SetNRGBA(x, y, c)
		}
	}
	whole := HomeSixel(pic, image.Rect(0, 0, 40, 3))
	left := HomeSixel(pic, image.Rect(0, 0, 2, 3))
	_, _, _, wi := readSixel(t, whole)
	_, _, _, li := readSixel(t, left)
	if wi.NRGBAAt(5, 5) != li.NRGBAAt(5, 5) {
		t.Errorf("the ground: %v in the whole, %v alone", wi.NRGBAAt(5, 5), li.NRGBAAt(5, 5))
	}
	if g := wi.NRGBAAt(5, 5); absDiff(g.R, ground.R) > 2 || absDiff(g.G, ground.G) > 2 || absDiff(g.B, ground.B) > 2 {
		t.Errorf("the ground drawn %v, %v given", g, ground)
	}
}

func absDiff(a, b uint8) int { return int(math.Abs(float64(a) - float64(b))) }

// Plan 51, design D1's grid mode: the design's structure in text, every
// row the screen's width: the header, the panels side by side with their
// titles, START SESSION, the design's projects in their sections, the
// footer; with Sixel and room, the banner's rows blank above the panels'
// least height, its image over them (design D2).
func TestHomeGrid(t *testing.T) {
	s, ui := homeTestUI()
	ui.Width, ui.Height, ui.Sixel = 80, 24, false
	ls, g := homeGrid(s, ui, HomeLayout(80, 24, false))
	text := strings.Split(rowsText(ls), "\n")
	if len(ls) != 24 || g.Banner {
		t.Fatalf("%d rows, banner %v", len(ls), g.Banner)
	}
	for i, l := range ls {
		if l.width() != 80 {
			t.Errorf("row %d: %d cells", i, l.width())
		}
	}
	for _, c := range []struct {
		row  int
		want []string
	}{
		{0, []string{"AGENT COMMAND CENTER", "v0.1.0", "Windows", "● Ready"}},
		{2, []string{"╭", "╮ ╭", "╮"}},
		{3, []string{"› NEW SESSION", "› PROJECTS"}},
		{23, []string{"Shortcuts:", "Ctrl+N  New Session", "Ctrl+O  Open Project", "Ctrl+Q  Quit"}},
	} {
		for _, w := range c.want {
			if !strings.Contains(text[c.row], w) {
				t.Errorf("row %d lacks %q: %q", c.row, w, text[c.row])
			}
		}
	}
	all := strings.Join(text, "\n")
	for _, w := range []string{"1.  Agent CLI", "◉  Claude Code (2.1.303)", "○  OpenAI Codex (0.163.0)", "2.  Project directory", `\agent-command-center │ 🗀 │`, "START SESSION →", "☆ FAVORITES", "◷ RECENT PROJECTS", "★  agent-command-center", "☆  automation-lab"} {
		if !strings.Contains(all, w) {
			t.Errorf("the frame lacks %q", w)
		}
	}
	if strings.Index(all, "FAVORITES") > strings.Index(all, "RECENT PROJECTS") || strings.Index(all, "code-workflow") > strings.Index(all, "marketing-site") {
		t.Error("FAVORITES not before RECENT PROJECTS")
	}
	// With Sixel at 100x30: twelve banner rows, blank, the panels under them.
	ui.Width, ui.Height, ui.Sixel = 100, 30, true
	ls, g = homeGrid(s, ui, HomeLayout(100, 30, true))
	text = strings.Split(rowsText(ls), "\n")
	if !g.Banner || g.BannerRows != 12 {
		t.Fatalf("100x30: banner %v, %d rows", g.Banner, g.BannerRows)
	}
	for y := gridBannerTop; y < gridBannerTop+12; y++ {
		if strings.TrimSpace(text[y]) != "" {
			t.Errorf("banner row %d: %q", y, text[y])
		}
	}
	if !strings.HasPrefix(strings.TrimSpace(text[14]), "╭") || !strings.Contains(all, "START SESSION →") {
		t.Errorf("the panels under the banner: %q", text[14])
	}
	if r := HomeRects(HomeLook{Geom: g}); len(r) != 1 || r[0] != image.Rect(0, 2, 100, 14) {
		t.Errorf("the banner's image: %v", r)
	}
}

// Plan 51, design D6 in grid mode: messages wrapped whole, never cut; the
// banner gives NEW SESSION the rows they need, hidden under bannerRows;
// beyond the screen's rows beside PROJECTS, NEW SESSION across the screen.
func TestHomeGridMessages(t *testing.T) {
	s, ui := homeTestUI()
	keys := `config.json: "ctrl+s" is not a send or newline key (use Enter with Ctrl, Alt or Shift, or Ctrl+J); default keys used`
	ui.KeysProblem = keys
	ui.Width, ui.Height, ui.Sixel = 100, 40, true
	ls, g := homeGrid(s, ui, HomeLayout(100, 40, true))
	if !g.Banner || g.BannerRows >= 22 || g.BannerRows < bannerRows {
		t.Errorf("one problem at 100x40: banner %v, %d rows (22 without it)", g.Banner, g.BannerRows)
	}
	flat := gridPanel(ls)
	if !strings.Contains(flat, strings.Join(strings.Fields("✗ "+keys), "")) || !strings.Contains(flat, "STARTSESSION→") {
		t.Errorf("the problem or START SESSION missing:\n%s", rowsText(ls))
	}
	// Long problems at 80x24: NEW SESSION across the screen, all shown.
	ui.CodexProxyProblem = `config.json: codex.proxy "socks5://127.0.0.1:1080" must start with http:// or https:// (such as http://127.0.0.1:3128); no proxy for Codex`
	ui.Width, ui.Height, ui.Sixel = 80, 24, false
	s.Startup.CLIs[1] = state.CLI{ID: "codex", Name: "OpenAI Codex", Problem: "npm install found (codex.cmd), native codex.exe needed"}
	s.Startup.Installed = []string{"claude-code"}
	ls, _ = homeGrid(s, ui, HomeLayout(80, 24, false))
	text := rowsText(ls)
	flat = gridPanel(ls)
	for _, w := range []string{"✗ " + keys, "✗ " + ui.CodexProxyProblem, "not installed: npm install found (codex.cmd), native codex.exe needed", "START SESSION →"} {
		if !strings.Contains(flat, strings.Join(strings.Fields(w), "")) {
			t.Errorf("80x24 lacks %q:\n%s", w, text)
		}
	}
	if strings.Contains(text, "PROJECTS") && strings.Index(text, "PROJECTS") < strings.Index(text, "START SESSION") {
		t.Errorf("PROJECTS beside NEW SESSION, not under it:\n%s", text)
	}
	if strings.Contains(text, "…\n") || len(ls) != 24 {
		t.Errorf("a row cut, or %d rows:\n%s", len(ls), text)
	}
}

// Plan 51, design D10 in grid mode: the focused control's ends in the
// focus's purple, the field's outline with it; the list's selected row on
// its band, brighter with the focus; in 16 colours START SESSION's label
// in the button's colour, no ground to show it.
func TestHomeGridFocus(t *testing.T) {
	s, ui := homeTestUI()
	ui.Width, ui.Height = 120, 40
	for field, want := range map[int]string{0: "▏ ◉  Claude Code", 2: "▏", 3: "▏★  agent-command-center"} {
		ui.Field = field
		ls, _ := homeGrid(s, ui, HomeLayout(120, 40, false))
		if !strings.Contains(rowsText(ls), want) {
			t.Errorf("field %d: no %q", field, want)
		}
	}
	ui.Field = 1
	ls, _ := homeGrid(s, ui, HomeLayout(120, 40, false))
	text := rowsText(ls)
	if !strings.Contains(text, ui.Path+"█") || strings.Contains(text, "▏ ◉") {
		t.Errorf("field 1: the cursor or a stray outline:\n%s", text)
	}
	var field *span
	for _, l := range ls {
		for i := range l {
			if strings.HasPrefix(l[i].text, "╭") && strings.Contains(l[i].text, "┬") {
				field = &l[i]
			}
		}
	}
	if field == nil || field.st != pageGround.on(homeStyle(cHomeFocus, false)) {
		t.Error("the field's outline not in the focus's purple")
	}
	sel := homeProjectRow(ui.Projects[0], true, false, 60, 20, 27, 20, 6)
	if sel[1].st != rowBandGround.on(homeStyle(cHomeFocus, false)) {
		t.Error("the selected row not on the band")
	}
	foc := homeProjectRow(ui.Projects[0], true, true, 60, 20, 27, 20, 6)
	if foc[1].st != rowFocusGround.on(homeStyle(cHomeFocus, false)) || foc[1].text != "▏" {
		t.Errorf("the focused row: %q not on the brighter band", foc[1].text)
	}
	UsePalette(pickANSI)
	defer UsePalette(pickTrue)
	b := homeButton(s, ui, 50)
	if b[2].st != homeStyle(cHomeButton, true) {
		t.Error("16 colours: START SESSION's label not in the button's colour")
	}
}

// gridPanel is the inside of a grid mode frame's first panel, NEW SESSION:
// its rows' text between its frame's sides, joined without spaces.
func gridPanel(ls []line) string {
	rows := strings.Split(rowsText(ls), "\n")
	var a, b int
	top := -1
	for i, r := range rows {
		rs := []rune(r)
		if j := strings.IndexRune(r, '╭'); j >= 0 {
			a = len([]rune(r[:j]))
			for k := a + 1; k < len(rs); k++ {
				if rs[k] == '╮' {
					b = k
					break
				}
			}
			top = i
			break
		}
	}
	var out strings.Builder
	for _, r := range rows[top+1:] {
		rs := []rune(r)
		if len(rs) <= b || rs[a] == '╰' {
			break
		}
		out.WriteString(strings.Join(strings.Fields(string(rs[a+1:b])), ""))
	}
	return out.String()
}

// pictureAt is the home screen's picture painted for ui at 208x59, with the
// stand-in glyphs, and its look.
func pictureAt(s *state.State, ui UI) (*image.NRGBA, HomeLook) {
	g := HomeLayout(208, 59, true)
	l := homeLook(s, ui, g)
	pic := image.NewNRGBA(l.Geom.Pixels())
	HomePaint(pic, l, &boxGlyphs{}, pic.Rect)
	return pic, l
}

// Plan 51, design D10 in picture mode: the focused control outlined in the
// design's purple, START SESSION's in a lighter colour, the list's band
// brighter; and design D6's START SESSION dimmed while there is nothing
// to start.
func TestHomePictureFocus(t *testing.T) {
	s, ui := homeTestUI()
	at := func(pic *image.NRGBA, l HomeLook, x, y float64) color.NRGBA {
		return pic.NRGBAAt(int(l.Geom.X(x)), int(l.Geom.Y(y)))
	}
	for field := range 4 {
		ui.Field, ui.ProjectSel = field, 0
		pic, l := pictureAt(s, ui)
		// The chosen option's outline at its top edge's middle.
		box := at(pic, l, 300, homeBoxes[0][0]+0.6)
		if want := cHomeChosenLine; field == 0 && box != hexColour(cHomeFocus.tc) || field != 0 && box != hexColour(want.tc) {
			t.Errorf("field %d: the option's outline %v", field, box)
		}
		// The field's outline at its top.
		y0, _, _ := homeFieldBox(l)
		top := pic.NRGBAAt(int(l.Geom.X(300)), int(y0)+1)
		if (field == 1) != (top == hexColour(cHomeFocus.tc)) {
			t.Errorf("field %d: the field's outline %v", field, top)
		}
		// START SESSION's outline at its top.
		ring := at(pic, l, 300, 645.2)
		if (field == 2) != (ring == hexColour(cHomeFocusOnButton.tc)) {
			t.Errorf("field %d: START SESSION's outline %v", field, ring)
		}
		// The selected row's band.
		band := at(pic, l, 1100, l.Rows[1].Y+10)
		if want := map[bool]colour{true: cHomeBandFocus, false: cHomeBand}[field == 3]; band != hexColour(want.tc) {
			t.Errorf("field %d: the band %v", field, band)
		}
	}
	for _, phase := range []state.StartupPhase{state.Detecting, state.Validating} {
		s.Startup.Phase = phase
		pic, l := pictureAt(s, ui)
		if c := at(pic, l, 100, 660); c != hexColour(cHomeButtonDim.tc) {
			t.Errorf("%s: START SESSION's fill %v, want dimmed", phase, c)
		}
	}
}

// Plan 51, design D6 in picture mode: the notes between the field and
// START SESSION, a line each after the first growing the panels into the
// banner, the field's row with them; START SESSION in its place; a reason
// too long for its option in full among the notes; PROJECTS' note when it
// lists nothing.
func TestHomePictureNotes(t *testing.T) {
	s, ui := homeTestUI()
	_, l0 := pictureAt(s, ui)
	if len(l0.Notes) != 0 || l0.Grow != 0 {
		t.Fatalf("no note: %v, grow %v", l0.Notes, l0.Grow)
	}
	ui.KeysProblem = `config.json: "ctrl+s" is not a send or newline key (use Enter with Ctrl, Alt or Shift, or Ctrl+J); default keys used`
	s.Startup.Field, s.Startup.Message = "project", `Does not exist: C:\Projects\agent-command-centre`
	pic, l := pictureAt(s, ui)
	lines := homeNoteLines(l.Notes)
	if len(l.Notes) != 2 || len(lines) != 3 || l.Grow != 2*noteLine {
		t.Fatalf("notes %v, %d lines, grow %v", l.Notes, len(lines), l.Grow)
	}
	if l.Geom.BannerRows >= l0.Geom.BannerRows || l.FieldRow >= l0.FieldRow {
		t.Errorf("banner %d rows (%d before), field row %d (%d before)", l.Geom.BannerRows, l0.Geom.BannerRows, l.FieldRow, l0.FieldRow)
	}
	// The project's problem in red on the first note's line, under the field.
	red := hexColour(cFailed.tc)
	y := int(l.Geom.Y(634.75 - l.Grow))
	found := false
	for x := int(l.Geom.X(31)); x < int(l.Geom.X(200)); x++ {
		if pic.NRGBAAt(x, y) == red {
			found = true
		}
	}
	if !found {
		t.Error("the first note not drawn in red under the field")
	}
	if c := pic.NRGBAAt(int(l.Geom.X(100)), int(l.Geom.Y(660))); c != hexColour(cHomeButton.tc) {
		t.Errorf("START SESSION moved: %v at its place", c)
	}
	// A long reason, cut in its option, in full among the notes.
	s, ui = homeTestUI()
	s.Startup.Installed = []string{"claude-code"}
	s.Startup.CLIs[1] = state.CLI{ID: "codex", Name: "OpenAI Codex", Problem: "npm install found (codex.cmd), native codex.exe needed, version unreadable: exit 1"}
	_, l = pictureAt(s, ui)
	if len(l.Notes) != 1 || !strings.Contains(l.Notes[0].Text, "native codex.exe needed, version unreadable: exit 1") {
		t.Errorf("the reason's note: %v", l.Notes)
	}
	// PROJECTS with none.
	ui.Projects = nil
	_, l = pictureAt(s, ui)
	if l.List.Text != "No projects yet" || len(l.Rows) != 0 {
		t.Errorf("no project: %q, %d rows", l.List.Text, len(l.Rows))
	}
}

// Plan 51, design D11: the cells a click acts on, in picture and in grid
// mode: each option, the field, its folder button, START SESSION, a
// project's row and its star, and the PROJECTS panel's for the wheel (plan
// 54); nothing on the header, the banner or the footer.
func TestHomeTargets(t *testing.T) {
	s, ui := homeTestUI()
	for _, c := range []struct {
		w, h  int
		sixel bool
	}{{208, 59, true}, {150, 40, true}, {80, 24, false}, {120, 40, false}} {
		ui.Width, ui.Height, ui.Sixel = c.w, c.h, c.sixel
		sc := Draw(s, ui)
		count := map[int]int{}
		for y := range c.h {
			for x := range c.w {
				p, item := sc.ItemAt(x, y)
				if p != HomePanel {
					continue
				}
				count[item]++
				if y == 0 || y == c.h-1 {
					t.Errorf("%dx%d: item %d on row %d", c.w, c.h, item, y)
				}
			}
		}
		for _, item := range []int{HomeItemOption, HomeItemOption + 1, HomeItemField, HomeItemFolder, HomeItemStart, HomeItemRow, HomeItemStar, HomeItemRow + 6, HomeItemStar + 6, HomeItemList} {
			if count[item] == 0 {
				t.Errorf("%dx%d sixel %v: no cell for item %#x", c.w, c.h, c.sixel, item)
			}
		}
		// Plan 54, design D3: the list's cells right of NEW SESSION's, and on
		// rows with no project too, the headings and the gaps.
		session, list := -1, c.w
		listRows, projectRows := map[int]bool{}, map[int]bool{}
		for y := range c.h {
			for x := range c.w {
				p, item := sc.ItemAt(x, y)
				if p != HomePanel {
					continue
				}
				switch kind, _ := HomeItemOf(item); kind {
				case HomeItemOption, HomeItemField, HomeItemFolder, HomeItemStart:
					session = max(session, x)
				case HomeItemList:
					list, listRows[y] = min(list, x), true
				case HomeItemRow, HomeItemStar:
					projectRows[y] = true
				}
			}
		}
		heading := false
		for y := range listRows {
			heading = heading || !projectRows[y]
		}
		if list <= session || !heading {
			t.Errorf("%dx%d sixel %v: the list's cells from column %d, NEW SESSION's to %d; on a row with no project %v", c.w, c.h, c.sixel, list, session, heading)
		}
		if kind, i := HomeItemOf(HomeItemStar + 6); kind != HomeItemStar || i != 6 {
			t.Errorf("HomeItemOf: %#x %d", kind, i)
		}
	}
}

// Plan 54, design D3, in grid mode's stacked layout, NEW SESSION's notes too
// long beside PROJECTS: the PROJECTS panel, all under START SESSION,
// carries the list's item too.
func TestHomeTargetsStacked(t *testing.T) {
	s, ui := homeTestUI()
	ui.Width, ui.Height, ui.Sixel = 80, 40, false
	ui.KeysProblem = strings.Repeat("config.json cannot be read, the default keys apply. ", 9)
	ui.CodexProxyProblem = strings.Repeat("codex.proxy must start with http:// or https://. ", 9)
	sc := Draw(s, ui)
	start, rows, list, top := -1, 0, 0, ui.Height
	for y := range ui.Height {
		for x := range ui.Width {
			p, item := sc.ItemAt(x, y)
			if p != HomePanel {
				continue
			}
			switch kind, _ := HomeItemOf(item); kind {
			case HomeItemStart:
				start = max(start, y)
			case HomeItemRow:
				rows++
			case HomeItemList:
				list, top = list+1, min(top, y)
			}
		}
	}
	if start < 0 || rows == 0 || list == 0 || top <= start {
		t.Errorf("stacked: START SESSION's last row %d, %d cells of rows, %d of the list from row %d", start, rows, list, top)
	}
}

// Plan 51, design D7 in picture mode: more projects than fit, the slim mark
// at the panel's right, as long as the share shown, where they lie.
func TestHomeScrollMark(t *testing.T) {
	s, ui := homeTestUI()
	ui.Projects = nil
	for i := range 20 {
		ui.Projects = append(ui.Projects, ProjectRow{Path: `C:\p\` + string(rune('a'+i)), Age: "now"})
	}
	ui.ProjectSel = 19
	pic, l := pictureAt(s, ui)
	first, shown, total := l.Scroll[0], l.Scroll[1], l.Scroll[2]
	if total != 20 || shown >= 20 || first+shown != 20 {
		t.Fatalf("window %v", l.Scroll)
	}
	mark := hexColour(cHomeFrame.tc)
	x := int(l.Geom.X(1323.5))
	top, bottom := -1, -1
	for y := int(l.Geom.Y(400)); y < int(l.Geom.Y(705)); y++ {
		if pic.NRGBAAt(x, y) == mark {
			if top < 0 {
				top = y
			}
			bottom = y
		}
	}
	rows := l.Geom.Y(l.Rows[len(l.Rows)-1].Y+14) - l.Geom.Y(l.Rows[0].Y-14)
	if top < 0 || float64(bottom-top) > rows*float64(shown)/20+4 || bottom < int(l.Geom.Y(l.Rows[len(l.Rows)-1].Y))-2 {
		t.Errorf("the mark from %d to %d, the rows %v high, %d of 20 shown", top, bottom, rows, shown)
	}
}
