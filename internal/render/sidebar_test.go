package render

import (
	"fmt"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"

	"github.com/Evengelius/agent-command-center/internal/provider/claude"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// Plan 43 (brief amendment 44): the activity sidebar and the Explorer
// docked between it and zone B, as drawn.

// frameRows is a frame's rows as text.
func frameRows(sc *Screen) []string {
	return strings.Split(strings.TrimSuffix(texts(sc.lines), "\n"), "\n")
}

// shownExplorer is an Explorer's panel shown, its project's top level read.
func shownExplorer() ExplorerView {
	return ExplorerView{Open: true, Name: "proj", Dirs: map[string]*ExplorerDir{"": {Entries: []ExplorerEntry{{"src", true, false}, {"a.go", false, false}}}},
		Opened: map[string]bool{}}
}

// TestSidebarLayout is criteria 43.1 and 43.2 (design D1): from 100
// columns the sidebar's box, 8 columns, ends the upper row on its rows
// only, zone B the rest, w - A - 8; the Explorer's panel, clamp(round(0.15
// w), 26, 36) columns, between zone B and the sidebar while zone B keeps 64
// columns, else in zone B's place; under 100 columns no sidebar, the panel
// in zone B's place. Zones A and C are the same with the panel shown or
// hidden, the footer last.
func TestSidebarLayout(t *testing.T) {
	quiet := probeState(t, "p2-claude-conversation.jsonl", claude.ProviderID, claude.NewMapper())
	one := probeState(t, "p3b-claude-subagent-approval.jsonl", claude.ProviderID, claude.NewMapper())
	for _, s := range []*state.State{quiet, one} {
		for _, c := range []struct{ w, h, aw, pw, bw int }{
			{120, 40, 36, 76, 0}, // zone B would be 50: the panel in its place
			{156, 44, 44, 26, 78},
			{200, 60, 56, 30, 106},
			{208, 59, 58, 31, 111},
		} {
			name := fmt.Sprintf("%dx%d, %d agents", c.w, c.h, s.AgentsInPlay())
			hidden := UI{Width: c.w, Height: c.h, Focus: "B"}
			shown := hidden
			shown.Focus, shown.Explorer = "R", shownExplorer()
			rh, rs := frameRows(Draw(s, hidden)), frameRows(Draw(s, shown))
			z := zoneBoxes(rh)
			a, b, cz := z["A"], z["B"], z["C"]
			// 43.1 Hidden: zone B w - A - 8 wide, the sidebar's box in the last 8
			// columns of A's and B's rows, zone C full width under them.
			if a.left != 0 || a.right != c.aw-1 || b.left != c.aw || b.right != c.w-9 || a.top != 2 || b.bottom != a.bottom {
				t.Errorf("%s hidden: A %+v, B %+v", name, a, b)
			}
			for y := a.top; y <= a.bottom; y++ {
				got, l, r := cellSlice(rh[y], c.w-8, c.w), "│", "│"
				switch y {
				case a.top:
					l, r = "╭──────", "╮"
				case a.bottom:
					l, r = "╰──────", "╯"
				}
				if !strings.HasPrefix(got, l) || !strings.HasSuffix(got, r) || textWidth(got) != 8 {
					t.Errorf("%s hidden: row %d's last 8 columns %q, not the sidebar's", name, y, got)
				}
			}
			if cz.top != a.bottom+1 || cz.left != 0 || cz.right != c.w-1 || cz.bottom != c.h-2 || !strings.HasPrefix(rh[c.h-1], " Enter send") {
				t.Errorf("%s hidden: C %+v, footer %q", name, cz, rh[c.h-1])
			}
			// 43.2 Shown: the panel's box between zone B and the sidebar, its
			// header inside; or in zone B's place, no zone B drawn.
			g := layout(s, shown)
			if g.aw != c.aw || g.bw != c.bw || g.pw != c.pw || g.sw != 8 {
				t.Errorf("%s shown: geometry %+v, want A %d, B %d, the panel %d, the sidebar 8", name, g, c.aw, c.bw, c.pw)
			}
			zs := zoneBoxes(rs)
			if bs, ok := zs["B"]; c.bw > 0 && (!ok || bs.left != c.aw || bs.right != c.aw+c.bw-1) || c.bw == 0 && ok {
				t.Errorf("%s shown: zone B %+v (drawn %v), want %d columns", name, bs, ok, c.bw)
			}
			x0 := c.aw + c.bw
			if top := cellSlice(rs[2], x0, x0+c.pw); !strings.HasPrefix(top, "╭─") || !strings.HasSuffix(top, "╮") || textWidth(top) != c.pw ||
				!strings.Contains(cellSlice(rs[3], x0, x0+c.pw), "EXPLORER") {
				t.Errorf("%s shown: the panel's top edge %q and header %q at columns %d to %d", name, top, cellSlice(rs[3], x0, x0+c.pw), x0, x0+c.pw-1)
			}
			if got := cellSlice(rs[2], c.w-8, c.w); got != "╭──────╮" {
				t.Errorf("%s shown: the sidebar's top edge %q", name, got)
			}
			// Zones A and C as with the panel hidden; the footer last.
			for y := 2; y <= a.bottom; y++ {
				if cellSlice(rs[y], 0, c.aw) != cellSlice(rh[y], 0, c.aw) {
					t.Errorf("%s: zone A's row %d changed with the panel shown", name, y)
				}
			}
			for y := cz.top; y < c.h-1; y++ {
				if rs[y] != rh[y] {
					t.Errorf("%s: zone C's row %d changed with the panel shown", name, y)
				}
			}
			if !strings.Contains(rs[c.h-1], "F3 Explorer") {
				t.Errorf("%s shown: the footer %q", name, rs[c.h-1])
			}
		}
	}
	// 80x24: no sidebar; shown, the panel takes zone B's rows, zone A's line
	// and zone C as before.
	hidden := UI{Width: 80, Height: 24, Focus: "B"}
	shown := hidden
	shown.Focus, shown.Explorer = "R", shownExplorer()
	rh, rs := frameRows(Draw(quiet, hidden)), frameRows(Draw(quiet, shown))
	zh, zs := zoneBoxes(rh), zoneBoxes(rs)
	if _, ok := zs["B"]; ok || zh["B"].top != 3 || zh["B"].right != 79 || strings.Contains(strings.Join(rh, "\n"), "≡") {
		t.Errorf("80x24: zones hidden %+v, shown %+v", zh, zs)
	}
	// The panel on zone B's rows, 3 to 19, the whole width.
	if !strings.HasPrefix(rs[3], "╭─") || !strings.HasSuffix(rs[3], "╮") || !strings.Contains(rs[4], "EXPLORER") || !strings.HasPrefix(rs[19], "╰") || !strings.HasSuffix(rs[19], "╯") {
		t.Errorf("80x24 shown: rows 3, 4 and 19: %q %q %q", rs[3], rs[4], rs[19])
	}
	for _, y := range []int{0, 1, 2, 20, 21, 22} {
		if rs[y] != rh[y] {
			t.Errorf("80x24: row %d changed with the panel shown: %q, was %q", y, rs[y], rh[y])
		}
	}
}

// TestLayoutWidths is criterion 43.2's invariants at every width from 80 to
// 240 columns and three heights, the panel hidden and shown: no zone under
// 5 columns, the upper row's columns summing to the screen's width, the
// panel docked exactly when zone B keeps 64 columns beside it, and every
// row of the dashboard exactly the screen's width before Draw fits it.
func TestLayoutWidths(t *testing.T) {
	one := probeState(t, "p3b-claude-subagent-approval.jsonl", claude.ProviderID, claude.NewMapper())
	for w := 80; w <= 240; w++ {
		for _, h := range []int{24, 40, 60} {
			for _, open := range []bool{false, true} {
				ui := UI{Width: w, Height: h, Focus: "B"}
				if open {
					ui.Focus, ui.Explorer = "R", shownExplorer()
				}
				name := fmt.Sprintf("%dx%d, shown %v", w, h, open)
				g := layout(one, ui)
				for _, x := range []int{g.aw, g.bw, g.pw, g.sw} {
					if x != 0 && x < 5 {
						t.Errorf("%s: a zone %d columns wide: %+v", name, x, g)
					}
				}
				switch {
				case w < standardWidth && (g.sw != 0 || g.aw != 0 || g.bw+g.pw != w || open != (g.pw == w)):
					t.Errorf("%s: under 100 columns %+v", name, g)
				case w >= standardWidth && (g.aw+g.bw+g.pw+g.sw != w || g.sw != sideWidth || g.aw != zoneAWidth(w)):
					t.Errorf("%s: %+v, not summing to the width", name, g)
				case w >= standardWidth && open && g.bw > 0 && (g.bw < dockMinB || g.pw != explorerWidth(w)):
					t.Errorf("%s: docked with zone B %d, the panel %d: %+v", name, g.bw, g.pw, g)
				case w >= standardWidth && open && g.bw == 0 && w-g.aw-sideWidth-explorerWidth(w) >= dockMinB:
					t.Errorf("%s: in zone B's place where it docks: %+v", name, g)
				}
				ls := dashboard(one, ui, new(Screen))
				if len(ls) != h {
					t.Errorf("%s: %d rows", name, len(ls))
				}
				for i, l := range ls {
					if l.width() != w {
						t.Errorf("%s: row %d is %d cells: %q", name, i, l.width(), plainOf(l))
						break
					}
				}
			}
		}
	}
}

// TestSidebarTiles is criterion 43.3 (designs D2 and D3), the text form:
// exactly two tiles, the Explorer's then Source Control's, 6 cells by 3 rows on the
// sidebar's rows 1 to 3 and 5 to 7, centred; inactive, the icon muted on no
// ground; selected (the Explorer's while its panel is shown), its ground,
// the lavender "▎" in its left cell and the icon lit; hovered, the hover
// band; a hovered Source Control keeps the Explorer selected.
func TestSidebarTiles(t *testing.T) {
	defer UsePalette(pickTrue)
	UsePalette(pickTrue)
	s := probeState(t, "p2-claude-conversation.jsonl", claude.ProviderID, claude.NewMapper())
	for _, g := range sideGlyph {
		if textWidth(g) != 1 {
			t.Errorf("the icon %q is %d cells", g, textWidth(g))
		}
	}
	type look struct {
		lead, icon, blank *lipgloss.Style
		bar               string
	}
	looks := map[SideLook]look{
		SideInactive:      {stNone, stSideIcon, stNone, " "},
		SideHover:         {bandGround.on(stNone), bandGround.on(stSideIconLit), bandGround.on(stNone), " "},
		SideSelected:      {sideSelGround.on(stSideAccent), sideSelGround.on(stSideIconLit), sideSelGround.on(stNone), "▎"},
		SideSelectedHover: {sideSelHoverGround.on(stSideAccent), sideSelHoverGround.on(stSideIconLit), sideSelHoverGround.on(stNone), "▎"},
	}
	for _, size := range [][2]int{{120, 40}, {200, 60}} {
		w, h := size[0], size[1]
		for _, c := range []struct {
			open                    bool
			hover                   int
			explorer, skills, graph SideLook
		}{
			{false, 0, SideInactive, SideInactive, SideInactive},
			{true, 0, SideSelected, SideInactive, SideInactive},
			{true, SideSource, SideSelected, SideHover, SideInactive},
			{true, SideExplorer, SideSelectedHover, SideInactive, SideInactive},
			{false, SideExplorer, SideHover, SideInactive, SideInactive},
			{false, SideGraph, SideInactive, SideInactive, SideHover}, // the Git Graph's (plan 62, design D1)
		} {
			ui := UI{Width: w, Height: h, Focus: "B", SideHover: c.hover}
			if c.open {
				ui.Explorer = shownExplorer()
			}
			name := fmt.Sprintf("%dx%d, shown %v, hover %d", w, h, c.open, c.hover)
			sc := Draw(s, ui)
			cells := map[int]int{}
			tops := map[int]int{SideExplorer: 3, SideSource: 7, SideGraph: 11} // under the box's top edge on row 2
			for y := range sc.lines {
				for x := range w {
					if p, it := sc.ItemAt(x, y); p == SidePanel && it != 0 {
						cells[it]++
						top := tops[it]
						if x < w-7 || x > w-2 || y < top || y > top+2 {
							t.Errorf("%s: tile %d's cell at %d,%d, outside columns %d to %d and rows %d to %d", name, it, x, y, w-7, w-2, top, top+2)
						}
					}
				}
			}
			if len(cells) != 3 || cells[SideExplorer] != 18 || cells[SideSource] != 18 || cells[SideGraph] != 18 {
				t.Errorf("%s: tiles' cells %v, want three of 18", name, cells)
			}
			for tile, want := range map[int]SideLook{SideExplorer: c.explorer, SideSource: c.skills, SideGraph: c.graph} {
				top := tops[tile]
				lk := looks[want]
				for r := range 3 {
					l := sc.lines[top+r]
					lead, _ := spanAt(l, w-7)
					if lead.text != lk.bar || lead.st != lk.lead {
						t.Errorf("%s: tile %d row %d's left cell %q, want %q in look %d", name, tile, r, lead.text, lk.bar, want)
					}
					for x := w - 6; x < w-1; x++ {
						sp, _ := spanAt(l, x)
						icon := r == 1 && x == w-5
						switch {
						case icon && (sp.text != sideGlyph[tile] || sp.st != lk.icon):
							t.Errorf("%s: tile %d's icon %q, styled for look %d %v", name, tile, sp.text, want, sp.st == lk.icon)
						case !icon && (strings.TrimSpace(sp.text) != "" || sp.st != lk.blank):
							t.Errorf("%s: tile %d's cell %d,%d %q not blank on its look %d's ground", name, tile, x, top+r, sp.text, want)
						}
					}
				}
			}
		}
	}
}

// TestExplorerPanel is criterion 43.7 (design D6), as drawn: the header
// "EXPLORER" in the title colour, bold, with "⌄" at its right end; the
// rule; the root row with the project folder's name; the tree one step
// under it, folders with "›" or "⌄" and a blue folder icon, files with a
// grey document icon, git's markers at the right, the cursor's row on its
// background; scrolling keeps the header, the rule and the root row; git's
// note on the last row; the frame in the title colour while the panel has
// the focus (criterion 43.10).
func TestExplorerPanel(t *testing.T) {
	defer UsePalette(pickTrue)
	UsePalette(pickTrue)
	s := probeState(t, "p2-claude-conversation.jsonl", claude.ProviderID, claude.NewMapper())
	v := ExplorerView{Open: true, Name: "my-project", Cursor: "docs/guide.md",
		Dirs: map[string]*ExplorerDir{
			"":     {Entries: []ExplorerEntry{{"docs", true, false}, {"src", true, false}, {"README.md", false, false}, {"main.go", false, false}}},
			"docs": {Entries: []ExplorerEntry{{"guide.md", false, false}}},
		},
		Opened: map[string]bool{"docs": true},
		Git:    NewGitView(map[string]byte{"src/x.go": 'M', "main.go": 'U'}, nil),
	}
	ui := UI{Width: 200, Height: 60, Focus: "R", Explorer: v}
	sc := Draw(s, ui)
	g := layout(s, ui)
	x0, x1 := g.aw+g.bw, g.aw+g.bw+g.pw
	panel := func(y int) line { return cols(sc.lines[y], x0, x1) }
	text := func(y int) string { return plainOf(panel(y)) }
	style := func(y int, s string) *lipgloss.Style { x, _ := spanOf(panel(y), s); return x.st }
	for _, c := range []struct {
		y    int
		want string
	}{
		{3, "│ EXPLORER"}, {5, "│ ⌄ 🗀  my-project"}, {6, "│   ⌄ 🗀  docs"}, {7, "│       🗎  guide.md"},
		{8, "│   › 🗀  src"}, {9, "│     🗎  README.md"}, {10, "│     🗎  main.go"},
	} {
		if got := text(c.y); !strings.HasPrefix(got, c.want) {
			t.Errorf("row %d %q, want %q…", c.y, got, c.want)
		}
	}
	if got := text(3); !strings.HasSuffix(got, "⌄ │") || style(3, "EXPLORER") != stTitle || !stTitle.GetBold() || style(3, "⌄") != stTitle {
		t.Errorf("the header %q: not the title colour, bold, with its ⌄ at the right", got)
	}
	if _, item := sc.ItemAt(x1-3, 3); item != ItemPanelHide {
		t.Errorf("the header's ⌄ carries item %d, want %d", item, ItemPanelHide)
	}
	if got := strings.Trim(text(4), "│ "); got != strings.Repeat("─", g.pw-4) || style(4, "─") != stFrame {
		t.Errorf("the rule %q", got)
	}
	for _, c := range []struct {
		y      int
		s      string
		st     *lipgloss.Style
		reason string
	}{
		{5, "🗀", stExpFolder, "the root's icon"}, {5, "my-project", stExpDirName, "the root's name"},
		{6, "🗀", stExpFolder, "a folder's icon"}, {6, "⌄", stGrey, "an open folder's chevron"}, {6, "docs", stExpDirName, "a folder's name"},
		{7, "🗎", stExpRowFile, "the cursor row's file icon"}, {7, "guide.md", stExpRowFileName, "the cursor row's name"},
		{8, "›", stGrey, "a closed folder's chevron"}, {8, "●", stProgress, "a folder holding a change"},
		{9, "🗎", stExpFile, "a file's icon"}, {9, "README.md", stExpFileName, "a file's name"},
		{10, "U", stDone, "an untracked file's marker"},
	} {
		if got := style(c.y, c.s); got != c.st {
			t.Errorf("%s %q on row %d: wrong style", c.reason, c.s, c.y)
		}
	}
	if !strings.HasSuffix(strings.TrimRight(text(10), " │"), "U") {
		t.Errorf("main.go's marker not at the right: %q", text(10))
	}
	// The frame's colour follows the focus.
	if edge, _ := spanAt(sc.lines[2], x0); edge.st != stTitle {
		t.Error("the focused panel's frame is not in the title colour")
	}
	ui.Focus = "B"
	if edge, _ := spanAt(Draw(s, ui).lines[2], x0); edge.st != stFrame {
		t.Error("the unfocused panel's frame is not grey")
	}

	// Scrolling: a hundred files, the cursor on the last; the header, the
	// rule and the root row stay, the cursor's row the tree's last shown.
	var many []ExplorerEntry
	for i := range 100 {
		many = append(many, ExplorerEntry{Name: fmt.Sprintf("f%03d.go", i)})
	}
	ui.Focus, ui.Explorer = "R", ExplorerView{Open: true, Name: "my-project", Cursor: "f099.go", Dirs: map[string]*ExplorerDir{"": {Entries: many}}, Opened: map[string]bool{}}
	sc = Draw(s, ui)
	last := 2 + g.upper - 2 // the panel's last inner row
	if !strings.Contains(text(3), "EXPLORER") || !strings.Contains(text(5), "my-project") || !strings.Contains(text(last), "f099.go") || strings.Contains(text(6), "f000.go") {
		t.Errorf("scrolled to the end: rows %q, %q, %q, %q", text(3), text(5), text(6), text(last))
	}
	if room := ExplorerRoom(s, ui); room != g.upper-panelFixed || !strings.Contains(text(6), fmt.Sprintf("f%03d.go", 100-room)) {
		t.Errorf("the tree's room %d, want %d; its first row %q", room, g.upper-panelFixed, text(6))
	}
	// Git's note on the last row, in grey or, a failure, in the attention
	// colour; while git's status is asked for, the tree keeps the row.
	for _, c := range []struct {
		note   string
		failed bool
		shown  bool
	}{{"not a git repository", false, true}, {"git: index locked", true, true}, {GitPending, false, false}} {
		ui.Explorer.Git = GitView{Note: c.note, Failed: c.failed}
		sc = Draw(s, ui)
		got := strings.Contains(text(last), c.note)
		if got != c.shown || c.shown && style(last, c.note) != either(c.failed, stAttention, stGrey) {
			t.Errorf("git's note %q on the last row: %v, want %v: %q", c.note, got, c.shown, text(last))
		}
		if want := g.upper - panelFixed - boolInt(c.shown); ExplorerRoom(s, ui) != want {
			t.Errorf("git's note %q: room %d, want %d", c.note, ExplorerRoom(s, ui), want)
		}
	}
}

// TestExplorerNames is design D6's names after the reference (plan 43,
// stage 2): the project root's and the folders' names #A0ABD8, the files'
// #9EAAD5, light lavender greys, not bold; on the cursor row the same on its
// background; a deleted name red and an ignored one grey as before.
func TestExplorerNames(t *testing.T) {
	defer UsePalette(pickTrue)
	UsePalette(pickTrue)
	for _, c := range []struct {
		name   string
		st     *lipgloss.Style
		fg, bg string
	}{
		{"a folder's", stExpDirName, "#A0ABD8", ""},
		{"a file's", stExpFileName, "#9EAAD5", ""},
		{"a folder's on the cursor row", stExpRowDirName, "#A0ABD8", "#2E3247"},
		{"a file's on the cursor row", stExpRowFileName, "#9EAAD5", "#2E3247"},
	} {
		_, none := c.st.GetBackground().(lipgloss.NoColor)
		if c.st.GetForeground() != lipgloss.Color(c.fg) || c.st.GetBold() || c.bg == "" && !none || c.bg != "" && c.st.GetBackground() != lipgloss.Color(c.bg) {
			t.Errorf("%s name: %v on %v, bold %v; want %s on %q, not bold", c.name, c.st.GetForeground(), c.st.GetBackground(), c.st.GetBold(), c.fg, c.bg)
		}
	}
	g := GitView{Ignored: map[string]bool{"tmp.log": true}}
	for _, c := range []struct {
		row ExplorerRow
		cur bool
		st  *lipgloss.Style
	}{
		{ExplorerRow{Path: "docs", Name: "docs", Dir: true}, false, stExpDirName},
		{ExplorerRow{Path: "docs", Name: "docs", Dir: true, Open: true}, true, stExpRowDirName},
		{ExplorerRow{Path: "a.go", Name: "a.go"}, false, stExpFileName},
		{ExplorerRow{Path: "a.go", Name: "a.go"}, true, stExpRowFileName},
		{ExplorerRow{Path: "gone.go", Name: "gone.go", Deleted: true}, false, stFailed},
		{ExplorerRow{Path: "gone.go", Name: "gone.go", Deleted: true}, true, stExpRowFailed},
		{ExplorerRow{Path: "tmp.log", Name: "tmp.log"}, false, stGrey},
		{ExplorerRow{Path: "tmp.log", Name: "tmp.log"}, true, stExpRowGrey},
	} {
		if sp, _ := spanOf(explorerRow(g, c.row, c.cur, 30, false), c.row.Name); sp.st != c.st {
			t.Errorf("%q, cursor %v: the name's style is not the one wanted", c.row.Name, c.cur)
		}
	}
	if sp, _ := spanOf(rootRow("proj", 30, false), "proj"); sp.st != stExpDirName {
		t.Error("the root row's name is not in a folder's colour")
	}
}
