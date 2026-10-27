package render

import (
	"image"
	"math"
	"strings"

	"github.com/Evengelius/agent-command-center/internal/state"
)

// The home screen (plan 51, brief amendment 52): the startup screen drawn
// after the owner's image, references/16-home-screen.webp, measured in
// reference-design.md, The home screen. Its places are the image's pixels,
// design pixels, 1347x757; in picture mode (design D1) they are scaled to
// the Sixel images' pixels, SixelCellW by SixelCellH a cell, the header
// from the top of the screen and the panels and the footer from its
// bottom, the banner between them.

const (
	designW, designH = 1347, 757
	// designHead is the header's height and designFoot the height from the
	// banner's end to the bottom, the panels' and the footer's.
	designHead = 45
	designFoot = designH - 375
	// pictureScale is the least scale of picture mode (design D1): the
	// design's smallest text is then 70% of the terminal's own.
	pictureScale = 0.875
	// bannerRows is the least banner, in rows (design D2): a band of
	// aliens, a shot's flight and the ship.
	bannerRows = 8
)

// HomeGeom is the home screen's geometry on a terminal (plan 51, designs D1
// and D2).
type HomeGeom struct {
	W, H int // the terminal, in cells
	// Picture is picture mode: the header, the panels and the footer drawn
	// as images, the design scaled by K (image pixels a design pixel) and
	// moved right by OX image pixels.
	Picture bool
	K, OX   float64
	// BannerRows are the rows between the header and the panels, the
	// banner's place; Banner says it is drawn, with bannerRows and more
	// (design D2).
	Banner     bool
	BannerRows int
}

// HomeLayout is the geometry for a terminal of w by h cells, sixel saying
// it draws Sixel images (designs D1 and D2). Without picture mode the grid
// mode lays the screen out in rows.
func HomeLayout(w, h int, sixel bool) HomeGeom {
	pw, ph := float64(w*SixelCellW), float64(h*SixelCellH)
	k := math.Min(pw/designW, ph/(designHead+designFoot))
	if !sixel || k < pictureScale {
		return gridLayout(w, h, sixel)
	}
	g := HomeGeom{W: w, H: h}
	g.Picture, g.K, g.OX = true, k, (pw-designW*k)/2
	g.BannerRows = int((ph - (designHead+designFoot)*k) / SixelCellH)
	g.Banner = g.BannerRows >= bannerRows
	return g
}

// gridLayout is grid mode's geometry: with Sixel, the banner on the rows
// above the panels' least height when there are bannerRows of them
// (design D2).
func gridLayout(w, h int, sixel bool) HomeGeom {
	g := HomeGeom{W: w, H: h}
	if sixel {
		g.BannerRows = h - gridRows
		g.Banner = g.BannerRows >= bannerRows
	}
	return g
}

// X is a design x in image pixels.
func (g HomeGeom) X(x float64) float64 { return g.OX + x*g.K }

// Y is a design y in image pixels: the header's from the top, the panels'
// and the footer's from the bottom.
func (g HomeGeom) Y(y float64) float64 {
	if y <= designHead {
		return y * g.K
	}
	return float64(g.H*SixelCellH) - (designH-y)*g.K
}

// Pixels is the picture's size in image pixels, the whole screen.
func (g HomeGeom) Pixels() image.Rectangle {
	return image.Rect(0, 0, g.W*SixelCellW, g.H*SixelCellH)
}

// The look's text (reference-design.md, The home screen, Text): each kind's
// characters' advance and capitals' height in design pixels, the advance
// fitted through the design's glyph centres and the height read on its
// stems, and whether it is semi-bold.
type textKind struct {
	adv, caps float64
	bold      bool
}

var (
	tkTitle     = textKind{8.94, 10.1, false}  // "AGENT COMMAND CENTER"
	tkHead      = textKind{7.69, 9.4, false}   // the version, the system, the status
	tkPanel     = textKind{10.63, 11.2, false} // the panels' titles
	tkButton    = textKind{9.92, 10.8, true}   // START SESSION
	tkAsk       = textKind{8.58, 10.3, false}  // the questions
	tkCLI       = textKind{8.96, 10.5, false}  // an agent CLI's name
	tkVersion   = textKind{8.77, 10.5, false}  // its version
	tkSection   = textKind{8.36, 9.3, false}   // FAVORITES, RECENT PROJECTS
	tkName      = textKind{8.19, 9.6, false}   // a project's name
	tkRow       = textKind{8.02, 9.6, false}   // its path and its age
	tkShortcuts = textKind{7.89, 9.5, true}    // "Shortcuts:"
	tkKey       = textKind{7.88, 9.5, false}   // a shortcut's key
	tkAction    = textKind{7.78, 9.5, false}   // its action
)

// HomeCLI is an agent CLI's option box (design D6).
type HomeCLI struct {
	Name, Version string // the version as shown, "(2.1.303)", or why it cannot be chosen
	Chosen        bool
	Usable        bool // installed: it can be chosen
}

// HomeRow is an entry of the Projects panel (design D7): a section's
// heading ('f' FAVORITES, 'r' RECENT PROJECTS) or a project, its index
// in the list, its middle at Y design pixels.
type HomeRow struct {
	Head            byte
	Name, Path, Age string
	Fav, Selected   bool
	Index           int
	Y               float64
}

// HomeKey is one of the footer's shortcuts (design D8).
type HomeKey struct{ Key, Action string }

// HomeNote is text the home screen adds to the design (design D6): its
// mark ("✗ ", "! ", "→ " or none), its text, wrapped under itself, and its
// kind: 'f' a failure, 'g' grey, 'a' a warning.
type HomeNote struct {
	Mark, Text string
	Kind       byte
}

// homeNotes are the notes between the project field and START SESSION
// (design D6): the agent CLI's problem, the project's (the live one grey
// until Start), the resolved path when it is not what is typed, the
// path's warning, the configuration file's problems and Codex's proxy.
func homeNotes(s *state.State, ui UI) []HomeNote {
	var out []HomeNote
	detecting := s.Startup.Phase == state.Detecting || s.Startup.Phase == ""
	projectFailed := s.Startup.Field == "project" && s.Startup.Message != ""
	switch {
	case !detecting && len(s.Startup.Installed) == 0:
		out = append(out, HomeNote{"✗ ", "No supported agent CLI found. Install Claude Code or Codex CLI, then restart.", 'f'})
	case s.Startup.Field == "provider" && s.Startup.Message != "":
		out = append(out, HomeNote{"✗ ", s.Startup.Message, 'f'})
	}
	switch {
	case projectFailed:
		out = append(out, HomeNote{"✗ ", hideUser(s.Startup.Message), 'f'})
	case ui.PathProblem != "":
		out = append(out, HomeNote{"✗ ", hideUser(ui.PathProblem), 'g'})
	case ui.Resolved != "" && ui.Resolved != strings.TrimSpace(ui.Path):
		out = append(out, HomeNote{"→ ", hideUser(ui.Resolved), 'g'})
	}
	if ui.PathProblem == "" && ui.PathWarning != "" && !projectFailed {
		out = append(out, HomeNote{"! ", ui.PathWarning, 'a'})
	}
	if ui.KeysProblem != "" {
		out = append(out, HomeNote{"✗ ", hideUser(ui.KeysProblem), 'f'}) // plan 21, item 5.1: the default keys apply
	}
	if ui.CodexProxyProblem != "" {
		out = append(out, HomeNote{"✗ ", hideUser(ui.CodexProxyProblem), 'f'}) // plan 22, amendment 19: no proxy
	}
	if ui.CodexProxy != "" {
		out = append(out, HomeNote{"", "Codex through the proxy " + ui.CodexProxy + " (config.json)", 'g'})
	}
	return out
}

// homeListNote is PROJECTS' note above its rows: the store's problem, or
// "No projects yet" when it lists none.
func homeListNote(ui UI) HomeNote {
	switch {
	case ui.ProjectsProblem != "":
		return HomeNote{"✗ ", hideUser(ui.ProjectsProblem), 'f'}
	case len(ui.Projects) == 0:
		return HomeNote{"", "No projects yet", 'g'}
	}
	return HomeNote{}
}

// homeDim says START SESSION is dimmed: while detecting, with no agent CLI,
// while a start is validated.
func homeDim(s *state.State) bool {
	return s.Startup.Phase == "" || s.Startup.Phase == state.Detecting || len(s.Startup.Installed) == 0 || s.Startup.Phase == state.Validating
}

// HomeLook is what the home screen's picture shows (plan 51), drawn by
// HomePaint: the application compares two looks to draw again what
// changed.
type HomeLook struct {
	Geom                    HomeGeom
	Version, System, Status string
	StatusKind              byte // 'g' ready, 'a' detecting or validating, 'r' no agent CLI
	CLIs                    []HomeCLI
	Rows                    []HomeRow
	Keys                    []HomeKey
	// FieldRow is the row of the project field's text, terminal text, and
	// FieldCols its columns, [first, end): the picture leaves them alone
	// (design D4).
	FieldRow  int
	FieldCols [2]int
	// Field is the focus (design D10): 0 the agent CLI, 1 the project
	// field, 2 START SESSION, 3 the project list.
	Field int
	// Dim dims START SESSION: nothing to start, or a start running.
	Dim bool
	// Notes are the lines between the field and START SESSION (design D6),
	// Grow the design pixels the panels grew into the banner to hold them.
	Notes []HomeNote
	Grow  float64
	// List is PROJECTS' note above its rows, the store's problem or "No
	// projects yet"; Scroll the rows' window when they do not all fit: the
	// first project shown, how many, of how many.
	List   HomeNote
	Scroll [3]int
}

// HomePanel is the home screen's panel for the mouse (design D11): its
// spans' items say what a click there does.
const HomePanel = 'H'

// The home screen's items: an option, a project's row and its star are
// these plus their index.
const (
	HomeItemField  = 1
	HomeItemFolder = 2
	HomeItemStart  = 3
	HomeItemOption = 1 << 20
	HomeItemRow    = 2 << 20
	HomeItemStar   = 3 << 20
)

// HomeItemOf is an item's kind and index: HomeItemOption, HomeItemRow or
// HomeItemStar and the index, else the item itself and 0.
func HomeItemOf(item int) (int, int) {
	if item >= HomeItemOption {
		return item &^ (1<<20 - 1), item & (1<<20 - 1)
	}
	return item, 0
}

// homeKeys are the footer's shortcuts (design D8): no command palette.
var homeKeys = []HomeKey{{"Ctrl+N", "New Session"}, {"Ctrl+O", "Open Project"}, {"Ctrl+Q", "Quit"}}

// homeStatus is the header's status from the startup machine (design D5).
func homeStatus(s *state.State) (string, byte) {
	switch {
	case s.Startup.Phase == "" || s.Startup.Phase == state.Detecting:
		return "Detecting…", 'a'
	case len(s.Startup.Installed) == 0:
		return "No agent CLI", 'r'
	case s.Startup.Phase == state.Validating:
		return "Validating…", 'a'
	}
	return "Ready", 'g'
}

// homeCLIs are the option boxes, in the order the screen lists them.
func homeCLIs(s *state.State, ui UI) []HomeCLI {
	detecting := s.Startup.Phase == "" || s.Startup.Phase == state.Detecting
	var out []HomeCLI
	for _, c := range supported {
		o := HomeCLI{Name: c.name, Chosen: ui.Choice == c.id && !detecting}
		switch {
		case detecting:
			o.Version = "(detecting…)"
		default:
			o.Version = "not installed"
			for _, f := range s.Startup.CLIs {
				if f.ID != c.id {
					continue
				}
				switch {
				case f.Installed:
					o.Version, o.Usable = "("+f.Version+")", true
				case f.Problem != "":
					o.Version = "not installed: " + f.Problem
				}
			}
		}
		out = append(out, o)
	}
	return out
}

// The Projects panel's rows in design pixels, their capitals' middles
// fitted through the design's (reference-design.md, The home screen): the
// first heading's, a section's first row 29.73 below its heading, the
// favourites 30.09 apart and the recent projects 28.7 as the design draws
// them, the next heading 41.92 under a section's last row. The design's
// last row lies at 686.2; one fits down to 689, a selected row's band
// still clear of the frame.
const (
	rowFirstHead = 438.5
	rowAfterHead = 29.73
	rowPitchFav  = 30.09
	rowPitch     = 28.7
	rowNextHead  = 41.92
	rowLast      = 689.0
)

// homeRows lays out the project list (design D7), its first heading's
// middle at top: FAVORITES then RECENT PROJECTS, a section with no project
// left out, windowed around the selected project when they do not all fit
// down to rowLast; and the window: the first project shown, how many, of
// how many.
func homeRows(ui UI, top float64) ([]HomeRow, [3]int) {
	type item struct {
		r HomeRow
		i int // the project's index, -1 for a heading
	}
	var items []item
	for _, fav := range []bool{true, false} {
		head := false
		for i, p := range ui.Projects {
			if p.Favourite != fav {
				continue
			}
			if !head {
				h := byte('r')
				if fav {
					h = 'f'
				}
				items = append(items, item{HomeRow{Head: h, Index: -1}, -1})
				head = true
			}
			name := base(hideUser(p.Path))
			if name == "" {
				name = p.Path // a drive's root
			}
			items = append(items, item{HomeRow{Name: name, Path: hideUser(p.Path), Age: p.Age, Fav: p.Favourite, Selected: i == ui.ProjectSel, Index: i}, i})
		}
	}
	// Each item's middle from the top heading's; the window starts at the
	// first item that keeps the selected project within the last row.
	ys := make([]float64, len(items))
	y := top
	for k, it := range items {
		if k > 0 {
			switch {
			case it.r.Head != 0:
				y += rowNextHead
			case items[k-1].r.Head != 0:
				y += rowAfterHead
			case it.r.Fav:
				y += rowPitchFav
			default:
				y += rowPitch
			}
		}
		ys[k] = y
	}
	first := 0
	for k, it := range items {
		if it.i == ui.ProjectSel {
			for ys[k]-ys[first] > rowLast-top {
				first++
			}
		}
	}
	var out []HomeRow
	var window [3]int
	for k, it := range items {
		if it.i < 0 {
			continue
		}
		if k < first {
			window[0]++
		}
		window[2]++
	}
	for k := first; k < len(items); k++ {
		r := items[k].r
		r.Y = ys[k] - ys[first] + top
		if r.Y > rowLast {
			break
		}
		if r.Head == 0 {
			window[1]++
		}
		out = append(out, r)
	}
	return out, window
}

// homeLook is the home screen's look for the frame (plan 51): the notes
// the panels grow into the banner for (design D6), the banner left with
// the rows it keeps, a reason too long for its option among the notes in
// full.
func homeLook(s *state.State, ui UI, g HomeGeom) HomeLook {
	status, kind := homeStatus(s)
	l := HomeLook{Geom: g, Version: "v" + ui.Version, System: ui.System, Status: status, StatusKind: kind,
		CLIs: homeCLIs(s, ui), Keys: homeKeys, Field: ui.Field, Dim: homeDim(s), Notes: homeNotes(s, ui), List: homeListNote(ui)}
	for _, c := range l.CLIs {
		if !c.Usable && len([]rune(c.Version)) > homeVersionChars(c.Name) {
			l.Notes = append(l.Notes, HomeNote{"", c.Name + " " + c.Version, 'g'})
		}
	}
	l.Grow = min(float64(max(len(homeNoteLines(l.Notes))-1, 0))*noteLine, maxGrow)
	if g.Picture {
		rows := int((float64(g.H*SixelCellH) - (designHead+designFoot+l.Grow)*g.K) / SixelCellH)
		l.Geom.BannerRows, l.Geom.Banner = rows, rows >= bannerRows
	}
	top := rowFirstHead - l.Grow
	if l.List.Text != "" {
		top += float64(len(homeListLines(l.List))) * noteLine // the store's problem above the rows
	}
	l.Rows, l.Scroll = homeRows(ui, top)
	l.FieldRow, l.FieldCols = homeFieldCells(l.Geom, l.Grow)
	return l
}

// maxGrow is the most the panels grow into the banner: their tops stay
// under the header.
const maxGrow = 320.0

// homeFieldCells is the row and the columns of the project field's text in
// picture mode (design D4): the row nearest the design's, its capitals'
// middle at y 605, grow higher with the panels; the columns from the
// path's start (x 47) to the cell before the divider (x 549).
func homeFieldCells(g HomeGeom, grow float64) (int, [2]int) {
	row := int(g.Y(605-grow) / SixelCellH)
	return row, [2]int{int(math.Floor(g.X(47) / SixelCellW)), int(math.Floor(g.X(549)/SixelCellW)) - 1}
}

// homePicture draws the startup screen's cells in picture mode: the page's
// ground everywhere, which the picture covers, and the project field's text
// in its cells, which the picture leaves alone (design D4); each cell a
// click acts on carries its item (design D11).
func homePicture(ui UI, l HomeLook) []line {
	ts := homeTargets(l)
	ls := make([]line, ui.Height)
	for y := range ls {
		ls[y] = homeTargetRow(ts, y, 0, ui.Width)
	}
	c0, c1 := l.FieldCols[0], l.FieldCols[1]
	if l.FieldRow < 0 || l.FieldRow >= ui.Height || c1 <= c0 {
		return ls
	}
	text := ui.Path
	cursor := ""
	if ui.Field == 1 {
		cursor = "█"
	}
	if w := c1 - c0; textWidth(text)+1 > w {
		text = "…" + tailCut(text, w-2)
	}
	field := line{sp(text, fieldGround.on(stHomePath)), sp(cursor, fieldGround.on(stTitle))}.fit(c1 - c0).pad(c1 - c0)
	for i := range field {
		if field[i].st == stNone || field[i].st == nil {
			field[i].st = fieldGround.on(stNone)
		}
		field[i].ref = ref{panel: HomePanel, kind: refDeco, item: HomeItemField}
	}
	ls[l.FieldRow] = append(append(homeTargetRow(ts, l.FieldRow, 0, c0), field...), homeTargetRow(ts, l.FieldRow, c1, ui.Width)...)
	return ls
}

// homeTarget is what a click on its cells does in picture mode.
type homeTarget struct {
	item int
	r    image.Rectangle // cells
}

// homeTargets are the cells a click acts on in picture mode (design D11):
// each option, the field, its folder button, START SESSION, each
// project's row and its star; a cell is a target's when its middle lies in
// the target's place.
func homeTargets(l HomeLook) []homeTarget {
	g := l.Geom
	var out []homeTarget
	add := func(item int, x0, y0, x1, y1 float64) { // image pixels
		cell := func(v, size float64) int { return int(math.Ceil(v/size - 0.5)) }
		r := image.Rect(cell(x0, SixelCellW), cell(y0, SixelCellH), cell(x1, SixelCellW), cell(y1, SixelCellH))
		if !r.Empty() {
			out = append(out, homeTarget{item, r})
		}
	}
	for i := range l.CLIs {
		if i < len(homeBoxes) {
			add(HomeItemOption+i, g.X(29.5), g.Y(homeBoxes[i][0]-l.Grow), g.X(594.5), g.Y(homeBoxes[i][1]-l.Grow))
		}
	}
	y0, y1, _ := homeFieldBox(l)
	add(HomeItemField, g.X(29.5), y0, g.X(549), y1)
	add(HomeItemFolder, g.X(549), y0, g.X(594), y1)
	add(HomeItemStart, g.X(29.5), g.Y(644.5), g.X(594), g.Y(688.5))
	for _, r := range l.Rows {
		if r.Head == 0 {
			add(HomeItemStar+r.Index, g.X(644), g.Y(r.Y-14.5), g.X(674), g.Y(r.Y+14.5))
			add(HomeItemRow+r.Index, g.X(674), g.Y(r.Y-14.5), g.X(1319), g.Y(r.Y+14.5))
		}
	}
	return out
}

// homeTargetRow is row y's blank cells from x0 to x1 on the page's ground,
// each run of a target's cells a span carrying its item.
func homeTargetRow(ts []homeTarget, y, x0, x1 int) line {
	var l line
	for x := x0; x < x1; {
		item, end := 0, x1
		for _, t := range ts {
			if y < t.r.Min.Y || y >= t.r.Max.Y {
				continue
			}
			if x >= t.r.Min.X && x < t.r.Max.X {
				item, end = t.item, min(t.r.Max.X, x1)
				break
			}
			if t.r.Min.X > x {
				end = min(end, t.r.Min.X)
			}
		}
		s := sp(strings.Repeat(" ", end-x), pageGround.on(stNone))
		if item != 0 {
			s.ref = ref{panel: HomePanel, kind: refDeco, item: item}
		}
		l = append(l, s)
		x = end
	}
	return l
}
