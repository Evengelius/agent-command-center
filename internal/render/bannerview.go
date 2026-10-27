package render

import (
	"image"
	"image/color"
	"math"
	"math/rand"
	"strconv"
	"strings"
)

// BannerView draws the banner's scene into the home screen's picture (plan
// 51, design D12): over the background its place held when the view was
// made, the scene's sprites; each frame, the cells of what moved, which
// the application writes as tiles in the view's fixed palette.
type BannerView struct {
	B     *Banner
	zone  image.Rectangle // the scene's place, image pixels
	k     float64         // image pixels a design pixel
	f     float64         // image pixels a sprite's pixel, the design's 2.5 design pixels
	s, g  int             // that rounded, and the sprites' glow's width
	bg    *image.NRGBA    // the place's background
	art   map[*spriteArt][]*image.NRGBA
	bolt  *image.NRGBA
	haze  *image.NRGBA // an explosion's haze, made once (hazeImage)
	drawn map[int]drawnItem
	pal   *sixelPalette
}

// drawnItem is a sprite as a frame drew it: its place and its look.
type drawnItem struct {
	r    image.Rectangle
	look int
}

// bannerItem is a sprite of a frame: its key (an alien's, a bolt's or an
// explosion's id, the ship -1), its look, its place, its image or its
// explosion.
type bannerItem struct {
	key, look int
	r         image.Rectangle
	img       *image.NRGBA
	blast     *blast
}

// bannerPlace is where the scene lies in the picture and its scale: in
// picture mode the whole cells within the design's banner, y 45 to 375 less
// the panels' growth, across the screen; in grid mode its rows. Whole
// cells, so that a tile of the scene holds the banner's pixels alone, all
// of them in its palette.
func bannerPlace(l HomeLook) (image.Rectangle, float64) {
	g := l.Geom
	if g.Picture {
		top := int(math.Ceil(g.Y(designHead) / SixelCellH))
		end := max(int(math.Floor(g.Y(375-l.Grow)/SixelCellH)), top)
		return image.Rect(0, top*SixelCellH, g.W*SixelCellW, end*SixelCellH).Intersect(g.Pixels()), g.K
	}
	z := image.Rect(0, gridBannerTop*SixelCellH, g.W*SixelCellW, (gridBannerTop+g.BannerRows)*SixelCellH)
	return z.Intersect(g.Pixels()), float64(g.W*SixelCellW) / designW
}

// NewBannerView is the view of the look's banner, its background pic's
// there: old's scene and sprites when old was as wide at the same scale
// (the panels grew or shrank), else a new scene from seed. It draws the
// scene into pic.
func NewBannerView(l HomeLook, pic *image.NRGBA, seed int64, old *BannerView) *BannerView {
	zone, k := bannerPlace(l)
	s := max(1, int(math.Round(spritePx*k)))
	v := &BannerView{zone: zone, k: k, f: spritePx * k, s: s, g: s, art: map[*spriteArt][]*image.NRGBA{}, drawn: map[int]drawnItem{}}
	v.bg = image.NewNRGBA(zone)
	for y := zone.Min.Y; y < zone.Max.Y; y++ {
		copy(v.bg.Pix[v.bg.PixOffset(zone.Min.X, y):v.bg.PixOffset(zone.Max.X, y)], pic.Pix[pic.PixOffset(zone.Min.X, y):pic.PixOffset(zone.Max.X, y)])
	}
	w, h := float64(zone.Dx())/k, float64(zone.Dy())/k
	if old != nil && old.k == k && old.zone.Dx() == zone.Dx() {
		v.B, v.art, v.bolt = old.B, old.art, old.bolt
		v.B.resize(h)
	} else {
		v.bolt = spriteImage(artBolt, 0, v.f, 2*s, glowBolt, 0.5)
		v.B = NewBanner(w, h, seed)
	}
	v.pal = v.palette()
	v.Frame(pic)
	return v
}

// Cells are the view's place in cells.
func (v *BannerView) Cells() image.Rectangle {
	return image.Rect(v.zone.Min.X/SixelCellW, v.zone.Min.Y/SixelCellH, v.zone.Max.X/SixelCellW, v.zone.Max.Y/SixelCellH)
}

// Fits says the view is the look's banner: shown, at the same place and
// scale.
func (v *BannerView) Fits(l HomeLook) bool {
	z, k := bannerPlace(l)
	return l.Geom.Banner && z == v.zone && k == v.k
}

// sprite is an art's frame drawn at the view's scale, with its glow.
func (v *BannerView) sprite(a *spriteArt, frame int) *image.NRGBA {
	imgs := v.art[a]
	if imgs == nil {
		for fr := range a.frames {
			imgs = append(imgs, spriteImage(*a, fr, v.f, v.g, glowSprite, 0.35))
		}
		v.art[a] = imgs
	}
	return imgs[frame%len(imgs)]
}

// palette is the view's fixed palette, its 48 commonest colours exact: from
// its background with every sprite drawn on it, and, on a strip of ground
// below, explosions at four ages as frames draw them.
func (v *BannerView) palette() *sixelPalette {
	z := v.zone
	strip := int(math.Ceil(2*blastHaze*v.k)) + 2
	sample := image.NewNRGBA(image.Rect(z.Min.X, z.Min.Y, z.Max.X, z.Max.Y+strip))
	for y := z.Min.Y; y < z.Max.Y; y++ {
		copy(sample.Pix[sample.PixOffset(z.Min.X, y):sample.PixOffset(z.Max.X, y)], v.bg.Pix[v.bg.PixOffset(z.Min.X, y):v.bg.PixOffset(z.Max.X, y)])
	}
	ground := hexColour(cHomeBanner.tc)
	for y := z.Max.Y; y < sample.Rect.Max.Y; y++ {
		for x := z.Min.X; x < z.Max.X; x++ {
			sample.SetNRGBA(x, y, ground)
		}
	}
	x, y := z.Min.X, z.Min.Y+z.Dy()/3
	for _, a := range []*spriteArt{&artLavender, &artPurple, &artShip} {
		for fr := range a.frames {
			img := v.sprite(a, fr)
			over(sample, img, x, y, sample.Rect)
			x += img.Rect.Dx()
		}
	}
	over(sample, v.bolt, x, y, sample.Rect)
	rng := rand.New(rand.NewSource(1))
	for i, age := range []float64{0.04, 0.16, 0.4, 0.72} {
		e := newBlast(rng, 0, float64(2*i+1)*blastHaze, (float64(z.Dy())+float64(strip)/2)/v.k)
		for e.age < age {
			e.step(bannerDT)
		}
		v.drawBlast(sample, &e, sample.Rect)
	}
	return newSixelPalette(sample, 48)
}

// Encode is the picture's cells c as a Sixel image in the view's palette.
func (v *BannerView) Encode(pic *image.NRGBA, c image.Rectangle) string {
	px := image.Rect(c.Min.X*SixelCellW, c.Min.Y*SixelCellH, c.Max.X*SixelCellW, c.Max.Y*SixelCellH)
	return v.pal.encode(pic.SubImage(px).(*image.NRGBA))
}

// items are the scene's sprites as they are now, in the order they are
// drawn: the aliens, the explosions, the bolts, the ship.
func (v *BannerView) items() []bannerItem {
	b := v.B
	var out []bannerItem
	for _, a := range b.aliens {
		art, look := &artLavender, a.frame
		if a.kind == 'p' {
			art, look = &artPurple, a.frame+2
		}
		out = append(out, v.place(a.id, look, v.sprite(art, a.frame), v.g, a.x, a.y))
	}
	for i := range b.blasts {
		e := &b.blasts[i]
		out = append(out, bannerItem{key: e.id, look: int(e.age * 1000), r: v.blastRect(e), blast: e})
	}
	for _, bl := range b.bolts {
		out = append(out, v.place(bl.id, 0, v.bolt, 2*v.s, bl.x, bl.y))
	}
	return append(out, v.place(-1, 0, v.sprite(&artShip, 0), v.g, b.ship.x, b.ship.y))
}

// place is a sprite's item at scene place (x, y), its image g pixels of
// glow round it.
func (v *BannerView) place(key, look int, img *image.NRGBA, g int, x, y float64) bannerItem {
	px := v.zone.Min.X + int(math.Round(x*v.k)) - g
	py := v.zone.Min.Y + int(math.Round(y*v.k)) - g
	return bannerItem{key: key, look: look, r: image.Rect(px, py, px+img.Rect.Dx(), py+img.Rect.Dy()), img: img}
}

// at is a scene place in image pixels.
func (v *BannerView) at(x, y float64) (float64, float64) {
	return float64(v.zone.Min.X) + x*v.k, float64(v.zone.Min.Y) + y*v.k
}

// blastPart is an explosion's particle's square in image pixels and how
// far it has faded, 0 to 1; not ok once it is gone.
func (v *BannerView) blastPart(p particle) (sq [4]float64, fade float64, ok bool) {
	if p.age >= p.life {
		return sq, 0, false
	}
	cx, cy := v.at(p.x, p.y)
	h := p.size * v.k / 2
	return [4]float64{cx - h, cy - h, cx + h, cy + h}, p.age / p.life, true
}

// blastRect is everything an explosion draws now, in whole pixels: its
// haze, which holds its starburst, and its particles.
func (v *BannerView) blastRect(e *blast) image.Rectangle {
	cx, cy := v.at(e.x, e.y)
	r := blastHaze * v.k
	out := image.Rect(int(math.Floor(cx-r)), int(math.Floor(cy-r)), int(math.Ceil(cx+r)), int(math.Ceil(cy+r)))
	for _, p := range e.parts {
		if sq, _, ok := v.blastPart(p); ok {
			out = out.Union(image.Rect(int(math.Floor(sq[0])), int(math.Floor(sq[1])), int(math.Ceil(sq[2])), int(math.Ceil(sq[3]))))
		}
	}
	return out
}

// drawBlast draws an explosion within clip as the design's: its purple
// haze, stronger to the middle, in five strengths dithered between, the
// same all its life; its starburst while it flashes; its particles, bright
// pink squares, half faded for the last third of their lives. Few colours,
// and none that change from frame to frame but where something moves
// (plan 51, design D12).
func (v *BannerView) drawBlast(pic *image.NRGBA, e *blast, clip image.Rectangle) {
	p := &painter{dst: pic, clip: clip.Intersect(pic.Rect)}
	cx, cy := v.at(e.x, e.y)
	h := v.hazeImage()
	r := h.Rect.Dx() / 2
	over(pic, h, int(math.Round(cx))-r, int(math.Round(cy))-r, p.clip)
	if t := 1 - e.age/blastFlash; t > 0 {
		v.starburst(p, cx, cy, t)
	}
	for _, part := range e.parts {
		sq, fade, ok := v.blastPart(part)
		if !ok {
			continue
		}
		c := blastParts[part.colour]
		if fade > 2.0/3 {
			c = color.NRGBA{c.R / 2, c.G / 2, c.B / 2, 0xFF}
		}
		p.fill(image.Rect(int(math.Round(sq[0])), int(math.Round(sq[1])), int(math.Round(sq[2])), int(math.Round(sq[3]))), c)
	}
}

// hazeImage is an explosion's haze at the view's scale, made once: its
// colour, its alpha stronger to its middle in five steps of 0.14, a 2x2
// ordered dither between them.
func (v *BannerView) hazeImage() *image.NRGBA {
	if v.haze != nil {
		return v.haze
	}
	rf := blastHaze * v.k
	r := int(math.Ceil(rf))
	v.haze = image.NewNRGBA(image.Rect(0, 0, 2*r, 2*r))
	bayer := [4]float64{0.125, 0.625, 0.875, 0.375}
	for y := range 2 * r {
		for x := range 2 * r {
			d := math.Hypot(float64(x-r)+0.5, float64(y-r)+0.5) / rf
			if d >= 1 {
				continue
			}
			level := 4 * math.Pow(1-d, 1.5)
			if level-math.Floor(level) > bayer[(y&1)*2+x&1] {
				level++
			}
			c := blastHazeC
			c.A = uint8(255*0.14*math.Floor(level) + 0.5)
			v.haze.SetNRGBA(x, y, c)
		}
	}
	return v.haze
}

// starburst draws an explosion's flash at (cx, cy), t of it left: a bright
// disc, then eight rays from it, the upright and level ones longer, each
// thinning to its tip, all shrinking with t.
func (v *BannerView) starburst(p *painter, cx, cy, t float64) {
	k := v.k
	rc, long, short, w := 4*k*math.Sqrt(t), 14*k*t, 9*k*t, 2.2*k
	reach := math.Max(long, rc) + 1
	for y := int(cy - reach); y <= int(cy+reach); y++ {
		for x := int(cx - reach); x <= int(cx+reach); x++ {
			px, py := float64(x)+0.5-cx, float64(y)+0.5-cy
			d := math.Hypot(px, py)
			if a := rc + 0.5 - d; a > 0 {
				f := d / (rc + 1)
				c := color.NRGBA{lerp8(blastCore.R, blastRay.R, f), lerp8(blastCore.G, blastRay.G, f), lerp8(blastCore.B, blastRay.B, f), 0xFF}
				p.blend(x, y, c, math.Min(a, 1))
			}
			for i := range 8 {
				l := long
				if i%2 == 1 {
					l = short
				}
				ux, uy := math.Cos(float64(i)*math.Pi/4), math.Sin(float64(i)*math.Pi/4)
				along := px*ux + py*uy
				if along < 0 || along > l {
					continue
				}
				if a := w/2*(1-along/l) + 0.5 - math.Abs(px*uy-py*ux); a > 0 {
					p.blend(x, y, blastRay, math.Min(a, 1)*t)
				}
			}
		}
	}
}

// Frame draws the scene as it is now into pic and gives the cells it drew:
// those of what appeared, moved, changed or went since the last frame,
// their background back and every sprite there over it, a rectangle a
// run of a row's cells, runs of the same columns in rows below joined;
// each with its pixels as they were before.
func (v *BannerView) Frame(pic *image.NRGBA) []BannerTile {
	items := v.items()
	now := make(map[int]drawnItem, len(items))
	var dirty []image.Rectangle
	for _, it := range items {
		d := drawnItem{it.r, it.look}
		now[it.key] = d
		if old, ok := v.drawn[it.key]; !ok || old != d {
			dirty = append(dirty, it.r)
			if ok {
				dirty = append(dirty, old.r)
			}
		}
	}
	for key, old := range v.drawn {
		if _, ok := now[key]; !ok {
			dirty = append(dirty, old.r)
		}
	}
	v.drawn = now
	cells := v.cellRuns(dirty)
	tiles := make([]BannerTile, len(cells))
	for i, c := range cells {
		px := image.Rect(c.Min.X*SixelCellW, c.Min.Y*SixelCellH, c.Max.X*SixelCellW, c.Max.Y*SixelCellH)
		was := make([]byte, 0, px.Dx()*px.Dy()*4)
		for y := px.Min.Y; y < px.Max.Y; y++ {
			was = append(was, pic.Pix[pic.PixOffset(px.Min.X, y):pic.PixOffset(px.Max.X, y)]...)
		}
		tiles[i] = BannerTile{Cells: c, was: was}
		v.paint(pic, items, px)
	}
	return tiles
}

// BannerTile is a frame's change: its cells, and their pixels before it,
// row by row.
type BannerTile struct {
	Cells image.Rectangle
	was   []byte
}

// paint draws clip again: the background, then every sprite over it.
func (v *BannerView) paint(pic *image.NRGBA, items []bannerItem, clip image.Rectangle) {
	for y := clip.Min.Y; y < clip.Max.Y; y++ {
		copy(pic.Pix[pic.PixOffset(clip.Min.X, y):pic.PixOffset(clip.Max.X, y)], v.bg.Pix[v.bg.PixOffset(clip.Min.X, y):v.bg.PixOffset(clip.Max.X, y)])
	}
	for _, it := range items {
		if !it.r.Overlaps(clip) {
			continue
		}
		if it.blast != nil {
			v.drawBlast(pic, it.blast, clip)
			continue
		}
		over(pic, it.img, it.r.Min.X, it.r.Min.Y, clip)
	}
}

// Overlay draws every sprite of the scene as it is into pic: a picture
// painted again, the scene on it as the last frame drew it.
func (v *BannerView) Overlay(pic *image.NRGBA) {
	v.paint(pic, v.items(), v.zone)
}

// cellRuns are the cells the rectangles of image pixels meet within the
// view's place: on each row, the runs of those cells, less than two cells
// apart joined; then a run and the same run on the rows below, one
// rectangle.
func (v *BannerView) cellRuns(rs []image.Rectangle) []image.Rectangle {
	cz := v.Cells()
	if cz.Empty() {
		return nil
	}
	w := cz.Dx()
	marks := make([]bool, w*cz.Dy())
	for _, r := range rs {
		r = r.Intersect(v.zone)
		if r.Empty() {
			continue
		}
		c := image.Rect(r.Min.X/SixelCellW, r.Min.Y/SixelCellH, (r.Max.X+SixelCellW-1)/SixelCellW, (r.Max.Y+SixelCellH-1)/SixelCellH).Intersect(cz)
		for y := c.Min.Y; y < c.Max.Y; y++ {
			for x := c.Min.X; x < c.Max.X; x++ {
				marks[(y-cz.Min.Y)*w+x-cz.Min.X] = true
			}
		}
	}
	var out []image.Rectangle
	open := map[[2]int]int{} // a run's columns to its rectangle in out, from the row above
	for y := cz.Min.Y; y < cz.Max.Y; y++ {
		next := map[[2]int]int{}
		for x := cz.Min.X; x < cz.Max.X; {
			if !marks[(y-cz.Min.Y)*w+x-cz.Min.X] {
				x++
				continue
			}
			x0, x1 := x, x+1
			for x1 < cz.Max.X {
				if marks[(y-cz.Min.Y)*w+x1-cz.Min.X] {
					x1++
				} else if x1+1 < cz.Max.X && marks[(y-cz.Min.Y)*w+x1+1-cz.Min.X] {
					x1 += 2
				} else {
					break
				}
			}
			key := [2]int{x0, x1}
			if i, ok := open[key]; ok {
				out[i].Max.Y = y + 1
				next[key] = i
			} else {
				next[key] = len(out)
				out = append(out, image.Rect(x0, y, x1, y+1))
			}
			x = x1
		}
		open = next
	}
	return out
}

// BannerSequence is the sequence writing a frame's tiles, each at its
// first cell, inside synchronized output, the cursor saved and restored:
// the pixels the frame changed in the view's palette, the others
// transparent, which keeps what the terminal shows there (the toggle's
// pill relies on it, plan 35); nothing when no pixel changed.
func (v *BannerView) BannerSequence(pic *image.NRGBA, tiles []BannerTile) string {
	var b strings.Builder
	for _, t := range tiles {
		px := image.Rect(t.Cells.Min.X*SixelCellW, t.Cells.Min.Y*SixelCellH, t.Cells.Max.X*SixelCellW, t.Cells.Max.Y*SixelCellH)
		if six := v.pal.encodeChanged(pic.SubImage(px).(*image.NRGBA), t.was); six != "" {
			b.WriteString(cup(t.Cells.Min.Y, t.Cells.Min.X))
			b.WriteString(six)
		}
	}
	if b.Len() == 0 {
		return ""
	}
	return "\x1b[?2026h\x1b7" + b.String() + "\x1b8\x1b[?2026l"
}

// cup is the cursor's move to row y, column x, counted from 0.
func cup(y, x int) string {
	return "\x1b[" + strconv.Itoa(y+1) + ";" + strconv.Itoa(x+1) + "H"
}
