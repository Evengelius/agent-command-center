package render

import (
	"fmt"
	"image"
	"image/color"
	"strings"
	"testing"

	"github.com/Evengelius/agent-command-center/internal/provider/claude"
)

// Plan 43 (brief amendment 44), stage 2: the sidebar's tiles as images.

// TestSideImages is criterion 43.4 (design D3), from the images' pixels:
// 60x60 each; inactive, transparent but for the icon in #6F7C88; hovered,
// a rounded tile of the hover ground #342B47, no outline, the icon
// brighter; selected, the fill #181C30 inside the #7A4EBA outline, the
// #A272F1 accent bar on the left edge, the icon #A6B3C6; selected and
// hovered, #20253D inside #8C5DD0, the bar, the icon #BAC6D8; each within 6
// levels. Around the tile, at its rounded corners and outside, the pixels
// are transparent. The Explorer's icon is a vertical trunk with a crossbar
// and a foot and two folder blobs, the upper with its tab; Source Control's three
// node discs and their links.
func TestSideImages(t *testing.T) {
	type look struct {
		fill, outline, icon color.NRGBA
		tile, bar           bool
	}
	looks := map[SideLook]look{
		SideInactive:      {icon: hexColour("#6F7C88")},
		SideHover:         {fill: hexColour("#342B47"), outline: hexColour("#342B47"), icon: hexColour("#B9C3D3"), tile: true},
		SideSelected:      {fill: hexColour("#181C30"), outline: hexColour("#7A4EBA"), icon: hexColour("#A6B3C6"), tile: true, bar: true},
		SideSelectedHover: {fill: hexColour("#20253D"), outline: hexColour("#8C5DD0"), icon: hexColour("#BAC6D8"), tile: true, bar: true},
	}
	if lum := func(c color.NRGBA) int { return int(c.R) + int(c.G) + int(c.B) }; lum(looks[SideHover].icon) <= lum(looks[SideInactive].icon) {
		t.Error("the hovered icon is not brighter than the inactive one")
	}
	bar := hexColour("#A272F1")
	for _, tile := range []int{SideExplorer, SideSource, SideGraph} {
		for lk, want := range looks {
			name := fmt.Sprintf("tile %d, look %d", tile, lk)
			img := SideImage(SideTileLook{Tile: tile, Look: lk})
			if img.Bounds() != image.Rect(0, 0, 60, 60) {
				t.Fatalf("%s: bounds %v", name, img.Bounds())
			}
			at := func(x, y int) color.NRGBA { return img.NRGBAAt(x, y) }
			clear := func(x, y int) bool { return at(x, y).A == 0 }
			// Outside: the image's corners, the tile's rounded corners, around
			// the tile; the bar's ends, rounded.
			for _, p := range [][2]int{{0, 0}, {59, 0}, {0, 59}, {59, 59}, {6, 5}, {56, 5}, {6, 55}, {56, 55},
				{58, 30}, {30, 2}, {30, 57}, {1, 30}, {3, 7}, {3, 52}, {2, 8}} {
				if !clear(p[0], p[1]) {
					t.Errorf("%s: (%d, %d) %v, not transparent", name, p[0], p[1], at(p[0], p[1]))
				}
			}
			// The tile's ground: the fill away from the icon, the outline on
			// the edges (the fill's colour when hovered: no outline).
			ground := func(x, y int, c color.NRGBA, what string) {
				t.Helper()
				switch {
				case !want.tile && !clear(x, y):
					t.Errorf("%s: %s (%d, %d) %v, want transparent", name, what, x, y, at(x, y))
				case want.tile && !nearColour(at(x, y), c, 6):
					t.Errorf("%s: %s (%d, %d) %s, want %s", name, what, x, y, hexOf(at(x, y)), hexOf(c))
				}
			}
			for _, p := range [][2]int{{48, 48}, {12, 46}, {48, 10}, {30, 50}} {
				ground(p[0], p[1], want.fill, "the fill")
			}
			for _, p := range [][2]int{{30, 4}, {56, 30}, {30, 55}} {
				ground(p[0], p[1], want.outline, "the outline")
			}
			// The accent bar on the left edge, over the outline there.
			for _, p := range [][2]int{{3, 9}, {3, 30}, {4, 30}, {5, 30}, {3, 50}} {
				switch c := at(p[0], p[1]); {
				case want.bar && !nearColour(c, bar, 6):
					t.Errorf("%s: the bar (%d, %d) %s, want %s", name, p[0], p[1], hexOf(c), hexOf(bar))
				case !want.bar && p[0] < 5 && !clear(p[0], p[1]):
					t.Errorf("%s: (%d, %d) %v, want no bar", name, p[0], p[1], c)
				}
			}
			icon := func(x, y int) bool { return nearColour(at(x, y), want.icon, 6) }
			off := func(x, y int) bool { // the ground there, no icon
				return want.tile && nearColour(at(x, y), want.fill, 6) || !want.tile && clear(x, y)
			}
			switch tile {
			case SideExplorer:
				// The trunk, a column from the crossbar's row to the foot's.
				for y := 19; y <= 36; y++ {
					if !icon(22, y) || !icon(23, y) {
						t.Errorf("%s: the trunk at (22..23, %d): %s %s", name, y, hexOf(at(22, y)), hexOf(at(23, y)))
						break
					}
				}
				for _, p := range [][2]int{{26, 23}, {26, 36}, {33, 19}, {36, 24}, {40, 26}, {36, 37}, {40, 40}} { // crossbar, foot, tab, folders
					if !icon(p[0], p[1]) {
						t.Errorf("%s: the tree's icon at (%d, %d): %s", name, p[0], p[1], hexOf(at(p[0], p[1])))
					}
				}
				// Two folder blobs: the column through them broken between them,
				// and the upper one's tab a bump at its left.
				for _, p := range [][2]int{{36, 30}, {40, 30}, {40, 19}, {26, 29}, {28, 30}} {
					if !off(p[0], p[1]) {
						t.Errorf("%s: (%d, %d) %s, want the ground between the tree's parts", name, p[0], p[1], hexOf(at(p[0], p[1])))
					}
				}
			case SideSource:
				// Three node discs, each wider than a link.
				for _, c := range [][2]int{{23, 21}, {37, 21}, {23, 38}} {
					if n := countNear(img, image.Rect(c[0]-3, c[1]-3, c[0]+4, c[1]+4), want.icon, 6); n < 25 {
						t.Errorf("%s: the node at (%d, %d): %d pixels of the icon", name, c[0], c[1], n)
					}
					run := 0
					for x := c[0] - 4; x <= c[0]+4; x++ {
						if icon(x, c[1]) {
							run++
						}
					}
					if run < 5 {
						t.Errorf("%s: the node at (%d, %d) %d pixels across", name, c[0], c[1], run)
					}
				}
				for _, p := range [][2]int{{23, 30}, {29, 31}, {37, 25}} { // the left line, the branch, up to the right node
					if !icon(p[0], p[1]) {
						t.Errorf("%s: a link at (%d, %d): %s", name, p[0], p[1], hexOf(at(p[0], p[1])))
					}
				}
				for _, p := range [][2]int{{30, 21}, {30, 38}, {37, 35}} { // no link between the upper nodes, none beside the lower
					if !off(p[0], p[1]) {
						t.Errorf("%s: (%d, %d) %s, want the ground between the nodes", name, p[0], p[1], hexOf(at(p[0], p[1])))
					}
				}
			case SideGraph:
				// The merge, a commit on each line, the fork, each wider than a
				// link (plan 62, design D1).
				for _, c := range [][2]int{{31, 21}, {24, 30}, {38, 30}, {31, 39}} {
					if n := countNear(img, image.Rect(c[0]-3, c[1]-3, c[0]+4, c[1]+4), want.icon, 6); n < 20 {
						t.Errorf("%s: the commit at (%d, %d): %d pixels of the icon", name, c[0], c[1], n)
					}
				}
				for _, p := range [][2]int{{27, 25}, {34, 25}, {27, 34}, {34, 34}} { // the two lines, each from the fork to the merge
					if !icon(p[0], p[1]) {
						t.Errorf("%s: a link at (%d, %d): %s", name, p[0], p[1], hexOf(at(p[0], p[1])))
					}
				}
				for _, p := range [][2]int{{31, 30}, {19, 30}, {43, 30}} { // the ground between the lines and beside them
					if !off(p[0], p[1]) {
						t.Errorf("%s: (%d, %d) %s, want the ground", name, p[0], p[1], hexOf(at(p[0], p[1])))
					}
				}
			}
			// Every pixel opaque or unset, as Sixel draws them; with no tile,
			// each pixel drawn the icon's colour itself, none blended into a
			// ground the image cannot know.
			for i := 3; i < len(img.Pix); i += 4 {
				if a := img.Pix[i]; a != 0 && a != 255 {
					t.Fatalf("%s: a pixel of alpha %d", name, a)
				}
			}
			for y := range 60 {
				for x := range 60 {
					if c := at(x, y); !want.tile && c.A != 0 && c != want.icon {
						t.Fatalf("%s: (%d, %d) %s on no tile, not the icon's %s", name, x, y, hexOf(c), hexOf(want.icon))
					}
				}
			}
		}
	}
}

// TestSideSixel: each tile's look encoded once, its unset pixels left
// unset (P2 = 1) and its raster 60x60; the twelve images differ.
func TestSideSixel(t *testing.T) {
	seen := map[string]SideTileLook{}
	for _, tile := range []int{SideExplorer, SideSource, SideGraph} {
		for _, lk := range []SideLook{SideInactive, SideHover, SideSelected, SideSelectedHover} {
			k := SideTileLook{Tile: tile, Look: lk}
			s := SideSixel(k)
			if !strings.HasPrefix(s, "\x1bP0;1;0q\"1;1;60;60") || s != Sixel(SideImage(k)) || SideSixel(k) != s {
				t.Errorf("%+v: %.40q", k, s)
			}
			if o, ok := seen[s]; ok {
				t.Errorf("%+v draws as %+v", k, o)
			}
			seen[s] = k
		}
	}
}

// TestSideTileCells is criterion 43.3 with Sixel (design D2): each tile's
// 6x3 cells blank, carrying their tile, the frame the same in every look,
// so a hover or the selection never has the renderer write them; the
// screen records each tile's first cell and look for its image. Without
// Sixel, under 100 columns or under a full-screen window, no image. Since
// plan 44 (design D1) a hover draws its tooltip left of the sidebar on the
// tile's middle row: every other cell is the unhovered frame's, the
// sidebar's on that row too.
func TestSideTileCells(t *testing.T) {
	s := probeState(t, "p2-claude-conversation.jsonl", claude.ProviderID, claude.NewMapper())
	for _, size := range [][2]int{{120, 40}, {156, 44}, {200, 60}, {208, 59}} {
		w, h := size[0], size[1]
		for _, open := range []bool{false, true} {
			var frame string
			var base *Screen
			for _, hover := range []int{0, SideExplorer, SideSource, SideGraph} {
				ui := UI{Width: w, Height: h, Focus: "B", Sixel: true, SideHover: hover}
				if open {
					ui.Explorer = shownExplorer()
				}
				name := fmt.Sprintf("%dx%d, shown %v, hover %d", w, h, open, hover)
				sc := Draw(s, ui)
				f := sc.String()
				if frame == "" {
					frame, base = f, sc
				} else {
					ty := map[int]int{SideExplorer: 3, SideSource: 7, SideGraph: 11}[hover] + 1
					label := " " + sideLabels[hover] + " "
					lx := w - 8 - textWidth(label)
					fl, hl := strings.Split(frame, "\n"), strings.Split(f, "\n")
					for i := range fl {
						if i != ty && fl[i] != hl[i] {
							t.Errorf("%s: row %d differs from the same frame unhovered", name, i)
						}
					}
					if got := cellRange(plainOf(sc.lines[ty]), lx, lx+textWidth(label)); got != label {
						t.Errorf("%s: the tooltip %q, want %q", name, got, label)
					}
					if sc.Region(0, ty, lx) != base.Region(0, ty, lx) || sc.Region(w-8, ty, 8) != base.Region(w-8, ty, 8) {
						t.Errorf("%s: the tooltip's row changed outside the tooltip", name)
					}
				}
				for i, tile := range []int{SideExplorer, SideSource, SideGraph} {
					y := map[int]int{SideExplorer: 3, SideSource: 7, SideGraph: 11}[tile]
					want := SideTileShown{Shown: true, X: w - 7, Y: y, Look: SideTileLook{Tile: tile, Look: sideLook(ui, tile)}}
					if sc.Tiles[i] != want {
						t.Errorf("%s: tile %d shown %+v, want %+v", name, tile, sc.Tiles[i], want)
					}
					for r := range 3 {
						if got := sc.Region(w-7, y+r, 6); got != "      " {
							t.Errorf("%s: tile %d's row %d %q, not six blank cells", name, tile, r, got)
						}
						for x := w - 7; x < w-1; x++ {
							if sc.SideTile(x, y+r) != tile {
								t.Errorf("%s: (%d, %d) is not tile %d's", name, x, y+r, tile)
							}
						}
					}
				}
			}
		}
	}
	// No image without Sixel, under 100 columns, or under the draft window
	// full screen; the window as a window leaves them.
	for _, c := range []struct {
		name  string
		ui    UI
		shown bool
	}{
		{"no Sixel", UI{Width: 200, Height: 60, Focus: "B"}, false},
		{"99 columns", UI{Width: 99, Height: 40, Focus: "B", Sixel: true}, false},
		{"80x24", UI{Width: 80, Height: 24, Focus: "B", Sixel: true}, false},
		{"the draft full screen", UI{Width: 200, Height: 60, Focus: "B", Sixel: true, DraftWin: DraftWindow{Open: true, Full: true}}, false},
		{"the draft window", UI{Width: 200, Height: 60, Focus: "B", Sixel: true, DraftWin: DraftWindow{Open: true}}, true},
	} {
		sc := Draw(s, c.ui)
		if sc.Tiles[0].Shown != c.shown || sc.Tiles[1].Shown != c.shown || sc.Tiles[2].Shown != c.shown {
			t.Errorf("%s: tiles %+v, want shown %v", c.name, sc.Tiles, c.shown)
		}
	}
}
