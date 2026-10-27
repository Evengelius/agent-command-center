package render

import (
	"bytes"
	"fmt"
	"image"
	"math"
	"slices"
	"strings"
	"testing"
)

// Plan 51, criterion 51.6: the banner's scene from a fixed seed over 1,000
// frames, as wide and high as the design's banner at 208x59: aliens
// appearing across the whole width, moving both ways and descending, never
// overlapping, and replaced, as many always; the ship moving in all four
// directions and firing two-bolt shots; each hit exploding.
func TestBannerScene(t *testing.T) {
	b := NewBanner(1347, 324, 51)
	many := len(b.aliens) + len(b.waits)
	if many != 32 {
		t.Errorf("%d aliens, want 32 for the design's area", many)
	}
	// Each alien's frames on the screen, and whether it stepped left, right
	// and down.
	type life struct {
		frames            int
		last              alien
		left, right, down bool
	}
	lives := map[int]*life{}
	var eighths [8]int
	var ship [4]bool // moved left, right, up, down
	var shots []int  // the frames a bolt was fired
	blasts := map[int]bool{}
	note := func() {
		for _, a := range b.aliens {
			l := lives[a.id]
			if l == nil {
				l = &life{last: a}
				lives[a.id] = l
				eighths[int(a.x*8/b.W)]++
			}
			l.frames++
			l.left = l.left || a.x < l.last.x
			l.right = l.right || a.x > l.last.x
			l.down = l.down || a.y > l.last.y
			l.last = a
		}
	}
	note()
	for range 1000 {
		was, fired := b.ship, b.Shots
		b.Step()
		note()
		ship[0] = ship[0] || b.ship.x < was.x
		ship[1] = ship[1] || b.ship.x > was.x
		ship[2] = ship[2] || b.ship.y < was.y
		ship[3] = ship[3] || b.ship.y > was.y
		for range b.Shots - fired {
			shots = append(shots, b.Frame)
		}
		for _, e := range b.blasts {
			blasts[e.id] = true
		}
		if n := len(b.aliens) + len(b.waits); n != many {
			t.Fatalf("frame %d: %d aliens and waits, want %d", b.Frame, n, many)
		}
		for i, a := range b.aliens {
			for _, o := range b.aliens[i+1:] {
				if math.Abs(a.x-o.x) < alienW+alienGap && math.Abs(a.y-o.y) < alienH+alienGap {
					t.Fatalf("frame %d: aliens %d and %d within %d pixels", b.Frame, a.id, o.id, alienGap)
				}
			}
		}
	}
	for i, n := range eighths {
		if n == 0 {
			t.Errorf("no alien appeared in the width's eighth %d: %v", i, eighths)
		}
	}
	still := 0
	for id, l := range lives {
		if l.frames >= 50 && !(l.down && (l.left || l.right)) {
			still++ // four seconds and more on the screen, yet not moving and descending
			t.Logf("alien %d: %d frames, left %v right %v down %v", id, l.frames, l.left, l.right, l.down)
		}
	}
	if still > 0 {
		t.Errorf("%d aliens of four seconds and more on the screen not moving and descending", still)
	}
	if b.Spawned <= many {
		t.Errorf("%d aliens appeared in all, none replaced", b.Spawned)
	}
	if ship != [4]bool{true, true, true, true} {
		t.Errorf("the ship moved left, right, up, down: %v", ship)
	}
	// Two bolts a shot, a frame apart, the shots far apart.
	if len(shots) < 10 {
		t.Fatalf("%d bolts fired", len(shots))
	}
	for i := 0; i+1 < len(shots); i += 2 {
		if shots[i+1] != shots[i]+1 || i+2 < len(shots) && shots[i+2]-shots[i+1] < 5 {
			t.Fatalf("bolts at frames %v: not in pairs a frame apart", shots[i:min(i+4, len(shots))])
		}
	}
	if b.Hits == 0 || len(blasts) != b.Hits {
		t.Errorf("%d hits, %d explosions", b.Hits, len(blasts))
	}
	t.Logf("%d aliens appeared, %d bolts, %d hits", b.Spawned, b.Shots, b.Hits)
}

// Plan 51, design D12: the view over the home screen's picture at 208x59,
// and the terminal as the frames' sequences leave it. Each frame changes no
// pixel outside its tiles, and the picture after it is the scene drawn
// whole over the background, so frames leave no trail; the terminal, given
// the banner whole in its palette and then each frame's images, the
// pixels a frame did not change transparent, shows what the banner drawn
// whole again would show.
func TestBannerFrames(t *testing.T) {
	s, ui := homeTestUI()
	pic, l := pictureAt(s, ui)
	bg := clonePic(pic)
	v := NewBannerView(l, pic, 51, nil)
	z := v.Cells()
	if z != image.Rect(0, 4, 208, 29) {
		t.Fatalf("the banner's cells %v, want rows 4 to 28", z)
	}
	whole := func() *image.NRGBA {
		w := clonePic(bg)
		v.Overlay(w)
		return w
	}
	if !bytes.Equal(pic.Pix, whole().Pix) {
		t.Fatal("the first frame is not the scene over the background")
	}
	// The terminal's banner, in pixels from the banner's first cell.
	_, _, _, term := readSixel(t, v.Encode(pic, z))
	sent, bytesSent := 0, 0
	for f := range 160 {
		before := clonePic(pic)
		v.B.Step()
		tiles := v.Frame(pic)
		for y := range pic.Rect.Dy() {
			i := pic.PixOffset(0, y)
			row, was := pic.Pix[i:i+pic.Stride], before.Pix[i:i+pic.Stride]
			if bytes.Equal(row, was) {
				continue
			}
			for x := range pic.Rect.Dx() {
				p := image.Pt(x/SixelCellW, y/SixelCellH)
				if !bytes.Equal(row[4*x:4*x+4], was[4*x:4*x+4]) && !slices.ContainsFunc(tiles, func(tl BannerTile) bool { return p.In(tl.Cells) }) {
					t.Fatalf("frame %d: pixel (%d, %d) changed outside the tiles", f, x, y)
				}
			}
		}
		for _, tl := range tiles {
			sent += tl.Cells.Dx() * tl.Cells.Dy()
			if !tl.Cells.In(z) {
				t.Fatalf("frame %d: tile %v outside the banner's cells", f, tl.Cells)
			}
		}
		seq := v.BannerSequence(pic, tiles)
		bytesSent += len(seq)
		for _, im := range sixelsAt(t, seq) {
			_, _, _, img := readSixel(t, im.six)
			for y := range img.Rect.Dy() {
				for x := range img.Rect.Dx() {
					c := img.NRGBAAt(x, y)
					if c.A == 0 {
						continue
					}
					px, py := im.at.X*SixelCellW+x, im.at.Y*SixelCellH+y
					if o := pic.PixOffset(px, py); bytes.Equal(pic.Pix[o:o+4], before.Pix[o:o+4]) {
						t.Fatalf("frame %d: pixel (%d, %d) sent though it did not change", f, px, py)
					}
					term.SetNRGBA(px-z.Min.X*SixelCellW, py-z.Min.Y*SixelCellH, c)
				}
			}
		}
		if f%8 == 7 {
			if !bytes.Equal(pic.Pix, whole().Pix) {
				t.Fatalf("frame %d: the picture is not the scene over the background", f)
			}
			if _, _, _, want := readSixel(t, v.Encode(pic, z)); !bytes.Equal(term.Pix, want.Pix) {
				t.Fatalf("frame %d: the terminal is not the banner drawn whole", f)
			}
		}
	}
	if v.B.Hits == 0 {
		t.Error("no explosion in the frames checked")
	}
	t.Logf("a frame on average: %d cells of %d, %d bytes", sent/160, z.Dx()*z.Dy(), bytesSent/160)
}

// sixelAt is a Sixel image of a sequence and the cell it is written at.
type sixelAt struct {
	at  image.Point
	six string
}

// sixelsAt are a sequence's Sixel images, each after its cursor move.
func sixelsAt(t *testing.T, seq string) []sixelAt {
	t.Helper()
	var out []sixelAt
	for {
		i := strings.Index(seq, "\x1bP")
		if i < 0 {
			return out
		}
		j := strings.LastIndex(seq[:i], "\x1b[")
		var y, x int
		if _, err := fmt.Sscanf(seq[j:i], "\x1b[%d;%dH", &y, &x); err != nil {
			t.Fatalf("no cursor move before an image: %q", seq[j:i])
		}
		end := strings.Index(seq[i:], "\x1b\\")
		out = append(out, sixelAt{image.Pt(x-1, y-1), seq[i : i+end+2]})
		seq = seq[i+end+2:]
	}
}

// Plan 51, design D12: a view made again for a banner as wide at the same
// scale, its height changed by the panels' notes, keeps its scene, the ship
// back in its band; another width starts a new scene.
func TestBannerKept(t *testing.T) {
	s, ui := homeTestUI()
	pic, l := pictureAt(s, ui)
	v := NewBannerView(l, pic, 51, nil)
	for range 20 {
		v.B.Step()
	}
	grown := l
	grown.Grow = 96 // six lines of notes
	w := NewBannerView(grown, pic, 52, v)
	if w.B != v.B || w.Cells().Max.Y >= v.Cells().Max.Y || w.Fits(l) || !w.Fits(grown) {
		t.Fatalf("grown: scene kept %v, cells %v from %v", w.B == v.B, w.Cells(), v.Cells())
	}
	if hi := w.B.H - shipH - 4; w.B.ship.y > hi+1e-9 || w.B.ship.ty > hi+1e-9 {
		t.Errorf("the ship at %.1f going to %.1f, below its band's end %.1f", w.B.ship.y, w.B.ship.ty, hi)
	}
	g := HomeLayout(150, 40, true)
	l2 := homeLook(s, ui, g)
	pic2 := image.NewNRGBA(g.Pixels())
	HomePaint(pic2, l2, &boxGlyphs{}, pic2.Rect)
	if x := NewBannerView(l2, pic2, 53, w); x.B == w.B || math.Abs(x.B.W-1347) > 1 {
		t.Errorf("150x40: scene kept %v, %.1f wide", x.B == w.B, x.B.W)
	}
}

func clonePic(p *image.NRGBA) *image.NRGBA {
	c := *p
	c.Pix = slices.Clone(p.Pix)
	return &c
}
