package render

import (
	"fmt"
	"image"
	"image/color"
	"strconv"
	"strings"
	"testing"

	"github.com/Evengelius/agent-command-center/internal/input"
	"github.com/Evengelius/agent-command-center/internal/input/imagetest"
)

// readSixel is the test's own Sixel decoder: the raster attributes' size,
// the colours defined, and the image, its pixels set where the data sets
// them, transparent elsewhere.
func readSixel(t *testing.T, s string) (w, h, colours int, img *image.NRGBA) {
	t.Helper()
	if !strings.HasPrefix(s, "\x1bP0;1;0q") || !strings.HasSuffix(s, "\x1b\\") {
		t.Fatalf("not a Sixel image: %.40q", s)
	}
	body := strings.TrimSuffix(strings.TrimPrefix(s, "\x1bP0;1;0q"), "\x1b\\")
	num := func() int {
		i := 0
		for i < len(body) && body[i] >= '0' && body[i] <= '9' {
			i++
		}
		n, _ := strconv.Atoi(body[:i])
		body = body[i:]
		return n
	}
	if body == "" || body[0] != '"' {
		t.Fatal("no raster attributes")
	}
	body = body[1:]
	var ra [4]int
	for i := range ra {
		ra[i] = num()
		if i < 3 && body != "" && body[0] == ';' {
			body = body[1:]
		}
	}
	w, h = ra[2], ra[3]
	img = image.NewNRGBA(image.Rect(0, 0, w, h))
	pal := map[int]color.NRGBA{}
	cur, x, y0 := 0, 0, 0
	put := func(ch byte, n int) {
		bits := ch - 63
		for range n {
			for dy := range 6 {
				if bits&(1<<dy) != 0 && x < w && y0+dy < h {
					img.SetNRGBA(x, y0+dy, pal[cur])
				}
			}
			x++
		}
	}
	for body != "" {
		c := body[0]
		body = body[1:]
		switch {
		case c == '#':
			n := num()
			if body != "" && body[0] == ';' {
				var v [4]int
				for i := range v {
					body = body[1:]
					v[i] = num()
				}
				scale := func(p int) uint8 { return uint8((p*255 + 50) / 100) }
				pal[n] = color.NRGBA{scale(v[1]), scale(v[2]), scale(v[3]), 255}
				continue
			}
			cur = n
		case c == '!':
			n := num()
			put(body[0], n)
			body = body[1:]
		case c == '$':
			x = 0
		case c == '-':
			x, y0 = 0, y0+6
		case c >= 63 && c <= 126:
			put(c, 1)
		default:
			t.Fatalf("unexpected %q in the data", c)
		}
	}
	return w, h, len(pal), img
}

// TestSixel is plan 32's criterion 32.4: the encoder's output, read back,
// gives the image's size, each opaque pixel within the palette's error of
// its colour, each transparent pixel unset, at most 256 colours.
func TestSixel(t *testing.T) {
	for _, side := range []int{140, 100} {
		src := input.Thumbnail(imagetest.Draw(), side, 8, ThumbBorder)
		w, h, colours, got := readSixel(t, Sixel(src))
		if w != side || h != side || colours < 2 || colours > 256 {
			t.Fatalf("%d: %dx%d, %d colours", side, w, h, colours)
		}
		worst := 0
		for y := range side {
			for x := range side {
				s, g := src.NRGBAAt(x, y), got.NRGBAAt(x, y)
				if s.A < 128 {
					if g.A != 0 {
						t.Fatalf("%d: (%d,%d) transparent in the image, set in the Sixel", side, x, y)
					}
					continue
				}
				if g.A == 0 {
					t.Fatalf("%d: (%d,%d) opaque in the image, unset in the Sixel", side, x, y)
				}
				for _, d := range []int{int(s.R) - int(g.R), int(s.G) - int(g.G), int(s.B) - int(g.B)} {
					worst = max(worst, max(d, -d))
				}
			}
		}
		// The drawing's colours are few: the palette holds them all, each
		// as near as Sixel's percents give.
		if worst > 3 {
			t.Errorf("%d: a channel off by %d", side, worst)
		}
	}
	// Many colours: a gradient of 4096 in a 64x64 image, reduced to 255.
	grad := image.NewNRGBA(image.Rect(0, 0, 64, 64))
	for y := range 64 {
		for x := range 64 {
			grad.SetNRGBA(x, y, color.NRGBA{uint8(x * 4), uint8(y * 4), uint8((x + y) * 2), 255})
		}
	}
	_, _, colours, got := readSixel(t, Sixel(grad))
	if colours > 255 {
		t.Errorf("the gradient in %d colours", colours)
	}
	worst := 0
	for y := range 64 {
		for x := range 64 {
			s, g := grad.NRGBAAt(x, y), got.NRGBAAt(x, y)
			for _, d := range []int{int(s.R) - int(g.R), int(s.G) - int(g.G), int(s.B) - int(g.B)} {
				worst = max(worst, max(d, -d))
			}
		}
	}
	if worst > 24 {
		t.Errorf("the gradient: a channel off by %d", worst)
	}
	if ThumbBorder != (color.NRGBA{0x3B, 0x3F, 0x51, 0xFF}) {
		t.Errorf("the border %v, want the frame's #3B3F51", ThumbBorder)
	}
}

// TestPreviewStrip is criterion 32.5 (design D4): with Sixel, the strip of
// blank cells between the rule and the chips, each thumbnail above its
// chip at its first cell, 14x7 cells from 50 rows, 10x5 from 30, none
// under 30; no strip without Sixel; a WebP's box of text; the strip's rows
// taken from the conversation.
func TestPreviewStrip(t *testing.T) {
	defer UsePalette(pickTrue)
	s := quiet(t)
	chips := []Attachment{
		{Name: "acc-image-upload-test.png", Size: "1 KB", Pixels: true, Format: "PNG"},
		{Name: "acc-image-upload-test.jpg", Size: "6 KB", Pixels: true, Format: "JPEG"},
	}
	for _, tc := range []struct{ w, h, cols, rows int }{{200, 60, 14, 7}, {120, 40, 10, 5}, {80, 24, 0, 0}} {
		ui := UI{Width: tc.w, Height: tc.h, Now: appClock, Focus: "B", Sixel: true}
		ui.Editor.Attach = chips
		off := ui
		off.Sixel = false
		on, plain := Draw(s, ui), Draw(s, off)
		if len(plain.Thumbs()) != 0 {
			t.Errorf("%dx%d: a strip without Sixel", tc.w, tc.h)
		}
		thumbs := on.Thumbs()
		if tc.rows == 0 {
			// The frame the same as without Sixel but for the mode pill, an
			// image over blank cells (plan 39): every other row, and its row
			// up to the pill.
			onRows, plainRows := strings.Split(texts(on.lines), "\n"), strings.Split(texts(plain.lines), "\n")
			px, py, ok := on.find(ModePillPanel)
			qx, qy, _ := plain.find(ModePillPanel) // the text pill, wider
			px = min(px, qx)
			same := ok && qy == py && len(onRows) == len(plainRows) && cellSlice(onRows[py], 0, px) == cellSlice(plainRows[py], 0, px)
			for y := range onRows {
				same = same && (y == py || onRows[y] == plainRows[y])
			}
			if len(thumbs) != 0 || !same {
				t.Errorf("%dx%d: a strip under 30 rows", tc.w, tc.h)
			}
			continue
		}
		if len(thumbs) != 2 {
			t.Fatalf("%dx%d: %d thumbnails", tc.w, tc.h, len(thumbs))
		}
		_, cy, ok := on.find(attachPanel)
		if !ok {
			t.Fatalf("%dx%d: no chips", tc.w, tc.h)
		}
		rows := strings.Split(strings.TrimSuffix(texts(on.lines), "\n"), "\n")
		if ry := on.editorRule(cy); ry != cy-tc.rows-1 {
			t.Errorf("%dx%d: the rule on row %d, the chips on %d, want the strip's %d rows between", tc.w, tc.h, ry, cy, tc.rows)
		}
		at := strings.Index(rows[cy], chips[0].Name)
		x := textWidth(rows[cy][:max(at, 0)]) - 1 // the first chip's first cell, its margin
		for i, th := range thumbs {
			// Side by side, one cell apart, the first above the first chip
			// (the owner: the space between the images "way shorter").
			if th.Item != i || th.W != tc.cols || th.H != tc.rows || th.Y != cy-tc.rows || th.X != x || th.Text {
				t.Errorf("%dx%d: thumbnail %+v, want %dx%d cells at (%d, %d)", tc.w, tc.h, th, tc.cols, tc.rows, x, cy-tc.rows)
			}
			x += tc.cols + 1
			for y := th.Y; y < th.Y+th.H; y++ {
				if r := []rune(rows[y]); strings.TrimSpace(string(r[th.X:th.X+th.W])) != "" {
					t.Errorf("%dx%d: the strip's row %d is not blank under thumbnail %d", tc.w, tc.h, y, i)
				}
			}
		}
		if on.Conv.Room != plain.Conv.Room-tc.rows {
			t.Errorf("%dx%d: the conversation has %d rows, %d without the strip", tc.w, tc.h, on.Conv.Room, plain.Conv.Room)
		}
		// The "+" menu's bottom edge stands on the rule, above the strip, not
		// on the strip's blank rows (editorRule wants the rule's dashes).
		menu := ui
		menu.Plus = true
		if _, my, ok := Draw(s, menu).find(menuPanel); !ok || my != cy-tc.rows-1 {
			t.Errorf("%dx%d: the menu's bottom edge on row %d, want the rule's %d", tc.w, tc.h, my, cy-tc.rows-1)
		}
	}
	// More images than the chips' row shows: the thumbnails go on, one
	// cell apart, as many as the width holds, past the chips folded into
	// "+N more".
	many := make([]Attachment, 12)
	for i := range many {
		many[i] = Attachment{Name: fmt.Sprintf("pasted-image-%d.png", i+1), Size: "2 KB", Pixels: true, Format: "PNG"}
	}
	for _, tc := range []struct{ w, h, cols int }{{200, 60, 14}, {120, 40, 10}} {
		ui := UI{Width: tc.w, Height: tc.h, Now: appClock, Focus: "B", Sixel: true}
		ui.Editor.Attach = many
		sc := Draw(s, ui)
		th := sc.Thumbs()
		chipsShown := strings.Count(texts(sc.lines), "✕")
		if len(th) <= chipsShown || len(th) >= len(many) {
			t.Errorf("%dx%d: %d thumbnails for %d chips shown of %d", tc.w, tc.h, len(th), chipsShown, len(many))
		}
		for i := 1; i < len(th); i++ {
			if th[i].Item != i || th[i].X != th[i-1].X+tc.cols+1 {
				t.Errorf("%dx%d: thumbnail %d at %d, the one before at %d", tc.w, tc.h, i, th[i].X, th[i-1].X)
			}
		}
		if last := th[len(th)-1]; last.X+last.W > tc.w {
			t.Errorf("%dx%d: the last thumbnail past the screen: %+v", tc.w, tc.h, last)
		}
	}

	// A WebP: a box of text with its format's name.
	ui := UI{Width: 200, Height: 60, Now: appClock, Focus: "B", Sixel: true}
	ui.Editor.Attach = []Attachment{{Name: "animated.webp", Size: "2 KB", Format: "WebP"}}
	sc := Draw(s, ui)
	th := sc.Thumbs()
	if len(th) != 1 || !th[0].Text {
		t.Fatalf("a WebP's thumbnail %+v", th)
	}
	rows := strings.Split(strings.TrimSuffix(texts(sc.lines), "\n"), "\n")
	box := func(y int) string { return string([]rune(rows[y])[th[0].X : th[0].X+th[0].W]) }
	if top, mid, bottom := box(th[0].Y), box(th[0].Y+3), box(th[0].Y+6); top != "╭"+strings.Repeat("─", 12)+"╮" || mid != "│    WebP    │" || bottom != "╰"+strings.Repeat("─", 12)+"╯" {
		t.Errorf("the WebP's box %q, %q, %q", top, mid, bottom)
	}
}

// TestRegion is design D5's repaint: a row's cells [x, x+w) as drawn,
// styles kept, a wide character cut by an edge as spaces.
func TestRegion(t *testing.T) {
	sc := &Screen{lines: []line{{deco("ab", stText), deco("界c", stGrey), deco("de", stNone)}}}
	for _, tc := range []struct {
		x, w int
		want string
	}{{0, 2, "ab"}, {1, 3, "b界"}, {3, 3, " cd"}, {2, 1, " "}, {5, 4, "de  "}} {
		if got := plainSGR(sc.Region(tc.x, 0, tc.w)); got != tc.want {
			t.Errorf("Region(%d, %d) = %q, want %q", tc.x, tc.w, got, tc.want)
		}
	}
	if got := sc.Region(0, 3, 4); got != "    " {
		t.Errorf("past the frame %q", got)
	}
}

// plainSGR strips the style sequences.
func plainSGR(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == 0x1b && i+1 < len(s) && s[i+1] == '[' {
			j := i + 2
			for j < len(s) && (s[j] < 0x40 || s[j] > 0x7e) {
				j++
			}
			i = j
			continue
		}
		b.WriteByte(s[i])
	}
	return b.String()
}
