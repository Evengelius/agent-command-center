package render

import (
	"image"
	"image/color"
	"math"
	"math/rand"
	"strings"
)

// Glyphs draws characters for the home screen's picture (plan 51, design
// D3): a coverage mask of r at em pixels, bold or not, its character cell
// from the mask's top left. The application gives GDI's, in Cascadia Mono;
// tests give a stand-in.
type Glyphs interface {
	Glyph(r rune, em int, bold bool) *image.Alpha
}

// HomePaint draws the look's picture into dst, the whole screen in image
// pixels, within clip: the page, the header, the banner's ground, the
// panels and the footer (reference-design.md, The home screen). Only what
// meets clip is drawn, so that a change draws only its place again.
func HomePaint(dst *image.NRGBA, l HomeLook, gl Glyphs, clip image.Rectangle) {
	p := &painter{dst: dst, clip: clip.Intersect(dst.Rect), g: l.Geom, gl: gl}
	if p.clip.Empty() {
		return
	}
	g := l.Geom
	if !g.Picture {
		// Grid mode: the banner's rows alone, the rest terminal text.
		if g.Banner {
			z := gridBanner(g)
			p.fill(image.Rect(0, int(z.top), g.W*SixelCellW, int(z.end)).Intersect(p.clip), hexColour(cHomePage.tc))
			p.banner(z)
		}
		return
	}
	p.fill(p.clip, hexColour(cHomePage.tc))
	if g.Banner {
		// Its place from the header's end (design y 45) to the panels'
		// (375); its ground from y 32 to 381, fading from the page's up to 56
		// and from 367 down, as the design's.
		up := l.Grow // the panels grown into the banner for the notes (design D6)
		p.banner(bannerZone{top: g.Y(designHead), end: g.Y(375 - up), fadeIn: 32 * g.K, full: 56 * g.K, fadeOut: g.Y(367 - up), out: g.Y(381 - up), k: g.K})
	}
	p.header(l)
	p.panels(l)
	p.footer(l)
}

// bannerZone is where the banner lies, in image pixels: its place from
// top to end, its ground fading in from fadeIn to full and out from
// fadeOut to out; k image pixels a design pixel.
type bannerZone struct{ top, end, fadeIn, full, fadeOut, out, k float64 }

// gridBanner is grid mode's banner: its rows, the design across the
// screen's width, its fades inside them.
func gridBanner(g HomeGeom) bannerZone {
	k := float64(g.W*SixelCellW) / designW
	top, end := float64(gridBannerTop*SixelCellH), float64((gridBannerTop+g.BannerRows)*SixelCellH)
	return bannerZone{top: top, end: end, fadeIn: top, full: top + 24*k, fadeOut: end - 14*k, out: end, k: k}
}

// painter draws in image pixels within its clip; a design place goes
// through g.
type painter struct {
	dst   *image.NRGBA
	clip  image.Rectangle
	g     HomeGeom
	gl    Glyphs
	ratio float64 // the font's capitals' height for its em, once read
}

// blend lays c over the pixel at (x, y) with coverage a.
func (p *painter) blend(x, y int, c color.NRGBA, a float64) {
	if a <= 0 || x < p.clip.Min.X || y < p.clip.Min.Y || x >= p.clip.Max.X || y >= p.clip.Max.Y {
		return
	}
	a = math.Min(a, 1)
	o := p.dst.PixOffset(x, y)
	px := p.dst.Pix[o : o+4 : o+4]
	px[0] = uint8(float64(px[0]) + (float64(c.R)-float64(px[0]))*a + 0.5)
	px[1] = uint8(float64(px[1]) + (float64(c.G)-float64(px[1]))*a + 0.5)
	px[2] = uint8(float64(px[2]) + (float64(c.B)-float64(px[2]))*a + 0.5)
	px[3] = 0xFF
}

// fill sets every pixel of r within the clip to c.
func (p *painter) fill(r image.Rectangle, c color.NRGBA) {
	r = r.Intersect(p.clip)
	for y := r.Min.Y; y < r.Max.Y; y++ {
		o := p.dst.PixOffset(r.Min.X, y)
		for x := r.Min.X; x < r.Max.X; x++ {
			copy(p.dst.Pix[o:o+4], []byte{c.R, c.G, c.B, 0xFF})
			o += 4
		}
	}
}

// meets says the box from (x0, y0) to (x1, y1), in image pixels, meets the
// clip.
func (p *painter) meets(x0, y0, x1, y1 float64) bool {
	return image.Rect(int(math.Floor(x0)), int(math.Floor(y0)), int(math.Ceil(x1))+1, int(math.Ceil(y1))+1).Overlaps(p.clip)
}

// overlap is how much of [a0, a1] lies in [b0, b1].
func overlap(a0, a1, b0, b1 float64) float64 { return math.Max(0, math.Min(a1, b1)-math.Max(a0, b0)) }

// rect paints the rectangle from (x0, y0) to (x1, y1), in image pixels, its
// edge pixels in part.
func (p *painter) rect(x0, y0, x1, y1 float64, c color.NRGBA) {
	r := image.Rect(int(math.Floor(x0)), int(math.Floor(y0)), int(math.Ceil(x1)), int(math.Ceil(y1))).Intersect(p.clip)
	for y := r.Min.Y; y < r.Max.Y; y++ {
		cy := overlap(float64(y), float64(y+1), y0, y1)
		for x := r.Min.X; x < r.Max.X; x++ {
			p.blend(x, y, c, cy*overlap(float64(x), float64(x+1), x0, x1))
		}
	}
}

// drect is rect in design pixels.
func (p *painter) drect(x0, y0, x1, y1 float64, c color.NRGBA) {
	p.rect(p.g.X(x0), p.g.Y(y0), p.g.X(x1), p.g.Y(y1), c)
}

// box paints a rounded rectangle, in image pixels: inside filled with fill
// when it is opaque, its outline lw wide in the colour line gives at a
// height, nil for none. A pixel's coverage comes from its distance to the
// edges, the corners included.
func (p *painter) box(x0, y0, x1, y1, r, lw float64, fill color.NRGBA, line func(y float64) color.NRGBA) {
	if !p.meets(x0, y0, x1, y1) {
		return
	}
	cx, cy, hx, hy := (x0+x1)/2, (y0+y1)/2, (x1-x0)/2, (y1-y0)/2
	sd := func(x, y float64) float64 { // the signed distance to the outer edge
		qx, qy := math.Abs(x-cx)-hx+r, math.Abs(y-cy)-hy+r
		return math.Hypot(math.Max(qx, 0), math.Max(qy, 0)) + math.Min(math.Max(qx, qy), 0) - r
	}
	b := image.Rect(int(math.Floor(x0)), int(math.Floor(y0)), int(math.Ceil(x1)), int(math.Ceil(y1))).Intersect(p.clip)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		py := float64(y) + 0.5
		for x := b.Min.X; x < b.Max.X; x++ {
			px := float64(x) + 0.5
			// Inside, far from the edges: the fill alone, without a distance.
			if px > x0+r+lw+1 && px < x1-r-lw-1 && py > y0+lw+1 && py < y1-lw-1 {
				if fill.A > 0 {
					p.blend(x, y, fill, 1)
				}
				continue
			}
			d := sd(px, py)
			out := math.Min(math.Max(0.5-d, 0), 1) // covered by the shape
			in := out
			if line != nil {
				in = math.Min(math.Max(0.5-(d+lw), 0), 1) // within the outline
			}
			if fill.A > 0 && in > 0 {
				p.blend(x, y, fill, in)
			}
			if line != nil && out-in > 0 {
				p.blend(x, y, line(py), out-in)
			}
		}
	}
}

// dbox is box in design pixels, its radius and line too, dy image pixels
// lower.
func (p *painter) dbox(x0, y0, x1, y1, r, lw, dy float64, fill color.NRGBA, line func(y float64) color.NRGBA) {
	k := p.g.K
	p.box(p.g.X(x0), p.g.Y(y0)+dy, p.g.X(x1), p.g.Y(y1)+dy, r*k, lw*k, fill, line)
}

// flat is a line colour the same at every height.
func flat(c colour) func(float64) color.NRGBA {
	v := hexColour(c.tc)
	return func(float64) color.NRGBA { return v }
}

// lineDist is the distance from (x, y) to the segment from a to b.
func lineDist(x, y float64, a, b [2]float64) float64 {
	dx, dy := b[0]-a[0], b[1]-a[1]
	t := 0.0
	if l := dx*dx + dy*dy; l > 0 {
		t = math.Min(math.Max(((x-a[0])*dx+(y-a[1])*dy)/l, 0), 1)
	}
	return math.Hypot(x-a[0]-t*dx, y-a[1]-t*dy)
}

// stroke paints the polyline pts, in design pixels, lw design pixels wide,
// closed when closed, dy image pixels lower: a pixel covered by its
// distance to the line.
func (p *painter) stroke(pts [][2]float64, closed bool, lw, dy float64, c color.NRGBA) {
	k := p.g.K
	q := make([][2]float64, len(pts))
	x0, y0, x1, y1 := math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)
	for i, v := range pts {
		q[i] = [2]float64{p.g.X(v[0]), p.g.Y(v[1]) + dy}
		x0, y0, x1, y1 = math.Min(x0, q[i][0]), math.Min(y0, q[i][1]), math.Max(x1, q[i][0]), math.Max(y1, q[i][1])
	}
	half := lw * k / 2
	if !p.meets(x0-half-1, y0-half-1, x1+half+1, y1+half+1) {
		return
	}
	n := len(q) - 1
	if closed {
		n = len(q)
	}
	for y := int(math.Floor(y0 - half - 1)); y <= int(math.Ceil(y1+half+1)); y++ {
		for x := int(math.Floor(x0 - half - 1)); x <= int(math.Ceil(x1+half+1)); x++ {
			d := math.Inf(1)
			for i := range n {
				d = math.Min(d, lineDist(float64(x)+0.5, float64(y)+0.5, q[i], q[(i+1)%len(q)]))
			}
			p.blend(x, y, c, half+0.5-d)
		}
	}
}

// ring paints a circle of radius r (to its line's middle) lw wide, centred
// at (x, y), all in design pixels, dy image pixels lower; disc paints a
// filled one.
func (p *painter) ring(x, y, r, lw, dy float64, c color.NRGBA) { p.circle(x, y, r, lw, dy, c, false) }
func (p *painter) disc(x, y, r, dy float64, c color.NRGBA)     { p.circle(x, y, r, 0, dy, c, true) }

func (p *painter) circle(x, y, r, lw, dy float64, c color.NRGBA, filled bool) {
	k := p.g.K
	cx, cy, R, half := p.g.X(x), p.g.Y(y)+dy, r*k, lw*k/2
	e := R + half + 1
	if !p.meets(cx-e, cy-e, cx+e, cy+e) {
		return
	}
	for py := int(math.Floor(cy - e)); py <= int(math.Ceil(cy+e)); py++ {
		for px := int(math.Floor(cx - e)); px <= int(math.Ceil(cx+e)); px++ {
			d := math.Hypot(float64(px)+0.5-cx, float64(py)+0.5-cy) - R
			if filled {
				p.blend(px, py, c, 0.5-d)
			} else {
				p.blend(px, py, c, half+0.5-math.Abs(d))
			}
		}
	}
}

// starPoints are a five-pointed star's corners, its outer points r from
// its middle (x, y), its inner ones ri, in design pixels.
func starPoints(x, y, r, ri float64) [][2]float64 {
	var pts [][2]float64
	for i := range 10 {
		rr := r
		if i%2 == 1 {
			rr = ri
		}
		a := -math.Pi/2 + float64(i)*math.Pi/5
		pts = append(pts, [2]float64{x + rr*math.Cos(a), y + rr*math.Sin(a)})
	}
	return pts
}

// polygon paints the polygon pts, in design pixels, dy image pixels lower,
// its pixels sampled four by four.
func (p *painter) polygon(pts [][2]float64, dy float64, c color.NRGBA) {
	q := make([][2]float64, len(pts))
	x0, y0, x1, y1 := math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)
	for i, v := range pts {
		q[i] = [2]float64{p.g.X(v[0]), p.g.Y(v[1]) + dy}
		x0, y0, x1, y1 = math.Min(x0, q[i][0]), math.Min(y0, q[i][1]), math.Max(x1, q[i][0]), math.Max(y1, q[i][1])
	}
	if !p.meets(x0, y0, x1, y1) {
		return
	}
	for y := int(math.Floor(y0)); y <= int(math.Ceil(y1)); y++ {
		for x := int(math.Floor(x0)); x <= int(math.Ceil(x1)); x++ {
			n := 0
			for sy := range 4 {
				for sx := range 4 {
					if inside(q, float64(x)+(float64(sx)+0.5)/4, float64(y)+(float64(sy)+0.5)/4) {
						n++
					}
				}
			}
			p.blend(x, y, c, float64(n)/16)
		}
	}
}

// inside says (x, y) lies inside the polygon q, by the even-odd rule.
func inside(q [][2]float64, x, y float64) bool {
	in := false
	for i, j := 0, len(q)-1; i < len(q); j, i = i, i+1 {
		if (q[i][1] > y) != (q[j][1] > y) && x < (q[j][0]-q[i][0])*(y-q[i][1])/(q[j][1]-q[i][1])+q[i][0] {
			in = !in
		}
	}
	return in
}

// capRow is the row of an em-pixel font's mask where its capitals' middle
// lies, read from "H".
func (p *painter) capRow(em int, bold bool) float64 {
	top, bot := capRows(p.gl.Glyph('H', em, bold))
	if top < 0 {
		return float64(em) / 2
	}
	return float64(top+bot+1) / 2
}

// text writes s from design x, where its first character's cell starts,
// its capitals' middle at design y, dy image pixels lower: in the text
// kind's size and weight, each character centred in its cell, the kind's
// advance wide, as the design's glyphs are.
func (p *painter) text(x, y float64, s string, tk textKind, c colour, dy float64) {
	if s == "" || p.gl == nil {
		return
	}
	k := p.g.K
	em := max(int(math.Round(tk.caps*k/p.capRatio())), 4)
	rs := []rune(s)
	x0, mid := p.g.X(x), p.g.Y(y)+dy
	if !p.meets(x0-float64(em), mid-float64(em), x0+float64(len(rs))*tk.adv*k+float64(em), mid+float64(em)) {
		return
	}
	top := int(math.Round(mid - p.capRow(em, tk.bold)))
	v := hexColour(c.tc)
	for i, r := range rs {
		if r == ' ' {
			continue
		}
		m := p.gl.Glyph(r, em, tk.bold)
		b := m.Rect
		left := int(math.Round(x0 + (float64(i)+0.5)*tk.adv*k - float64(b.Dx())/2))
		for gy := b.Min.Y; gy < b.Max.Y; gy++ {
			for gx := b.Min.X; gx < b.Max.X; gx++ {
				if a := m.AlphaAt(gx, gy).A; a > 0 {
					p.blend(left+gx-b.Min.X, top+gy-b.Min.Y, v, float64(a)/255)
				}
			}
		}
	}
}

// capRatio is the font's capitals' height for its em, read once from "H" at
// 100 pixels: a text kind's em is its capitals' height over it, whatever
// the face GDI found (design D3).
func (p *painter) capRatio() float64 {
	if p.ratio == 0 {
		p.ratio = 0.7
		if top, bot := capRows(p.gl.Glyph('H', 100, false)); top >= 0 {
			p.ratio = float64(bot-top+1) / 100
		}
	}
	return p.ratio
}

// capRows are the first and last rows of m holding a capital's ink, -1
// when it has none.
func capRows(m *image.Alpha) (top, bot int) {
	top, bot = -1, -1
	for y := m.Rect.Min.Y; y < m.Rect.Max.Y; y++ {
		for x := m.Rect.Min.X; x < m.Rect.Max.X; x++ {
			if m.AlphaAt(x, y).A > 127 {
				if top < 0 {
					top = y
				}
				bot = y
			}
		}
	}
	return top, bot
}

// fitChars is s cut with "…" to at most n characters.
func fitChars(s string, n int) string {
	rs := []rune(s)
	if len(rs) <= n || n < 1 {
		return s
	}
	return string(rs[:n-1]) + "…"
}

// fitCharsLeft is s cut from its left with "…" to at most n characters.
func fitCharsLeft(s string, n int) string {
	rs := []rune(s)
	if len(rs) <= n || n < 1 {
		return s
	}
	return "…" + string(rs[len(rs)-n+1:])
}

// header draws the header (reference-design.md, The home screen, The
// header): the logo, the title, and from the right the status, its dot, the
// system and the version, bars between them, each the design's distance
// from the next, the status ending where the design's does.
func (p *painter) header(l HomeLook) {
	logo := flat(cHomeLogo)
	for _, sq := range [][4]float64{{8, 11, 15.2, 18.2}, {16.8, 11, 24, 18.2}, {8, 18.8, 15.2, 26}, {16.8, 18.8, 24, 26}} {
		p.dbox(sq[0], sq[1], sq[2], sq[3], 2.2, 1.3, 0, color.NRGBA{}, logo)
	}
	p.text(45.54, 18.88, "AGENT COMMAND CENTER", tkTitle, cHomeTitle, 0)
	width := func(s string) float64 { return float64(len([]rune(s))) * tkHead.adv }
	bar := func(x float64) { p.drect(x-0.6, 9.6, x+0.6, 28.4, hexColour(cHomeBar.tc)) }
	xs := 1317.04 - width(l.Status)
	p.text(xs, 18.84, l.Status, tkHead, cHomeStatus, 0)
	dot := cHomeReady
	switch l.StatusKind {
	case 'a':
		dot = cProgress
	case 'r':
		dot = cFailed
	}
	p.disc(xs-14.88, 18.64, 4.15, 0, hexColour(dot.tc))
	b := xs - 38.52
	bar(b)
	xo := b - 20.52 - width(l.System)
	p.text(xo, 18.84, l.System, tkHead, cHomeSystem, 0)
	b = xo - 20.72
	bar(b)
	xv := b - 20.42 - width(l.Version)
	p.text(xv, 18.84, l.Version, tkHead, cHomeVersion, 0)
	bar(xv - 19.83)
}

// homeBand is a halftone band of the banner: from (ax, ay) to (bx, by) in
// design pixels, its half width w0 there and w1 at its end, fading in over
// the first in of its length (none from the screen's edge) and out over its
// last 35%.
type homeBand struct{ ax, ay, bx, by, w0, w1, in float64 }

// homeBands are the design's four (references/16-home-screen.webp: the
// lattice's dots brighter, measured on 10-pixel cells): two from the left
// edge down to the right, one from the top right corner and one from inside
// the right side down to the left.
var homeBands = []homeBand{
	{-10, 60.8, 200, 205, 19, 11, 0},
	{-10, 254.9, 215, 362, 15, 8, 0},
	{1357, 30, 1158, 205, 22, 20, 0},
	{1310, 252, 1040, 365, 18, 10, 0.15},
}

// bandStrength is how strongly the bands light the dot at design (x, y):
// across a band falling from its middle to its edge, along it with its
// fades; the strongest band's.
func bandStrength(x, y float64) float64 {
	best := 0.0
	for _, b := range homeBands {
		vx, vy := b.bx-b.ax, b.by-b.ay
		t := ((x-b.ax)*vx + (y-b.ay)*vy) / (vx*vx + vy*vy)
		if t < 0 || t > 1 {
			continue
		}
		d := math.Abs((x-b.ax)*vy-(y-b.ay)*vx) / math.Hypot(vx, vy)
		hw := b.w0 + (b.w1-b.w0)*t
		if d >= hw {
			continue
		}
		along := math.Min(1, (1-t)/0.35)
		if b.in > 0 {
			along = math.Min(along, t/b.in)
		}
		best = math.Max(best, (1-(d/hw)*(d/hw))*along)
	}
	return best
}

// bandDither is the lattice dot (i, j)'s threshold, 0 to 0.9, the same
// each time: a band lights it when its strength there reaches it, so the
// band thins out to its edges, scattered as the design's.
func bandDither(i, j int) float64 {
	h := uint32(i)*73856093 ^ uint32(j)*19349663
	h ^= h >> 13
	h *= 0x5bd1e995
	h ^= h >> 15
	return float64(h&0xFFFF) / 0x10000 * 0.9
}

// homeSparkles are the design's sparkles, in design pixels: its four white
// crosses and its two dim ones, their middles dark.
var homeSparkles = []struct {
	x, y float64
	dim  bool
}{{33, 224.5, false}, {461.5, 335, false}, {1240.5, 259.5, false}, {1313, 192.5, false}, {1305.5, 57.5, true}, {1278, 327.5, true}}

// sparkle draws a sparkle at (cx, cy), in image pixels, k a design pixel:
// a violet glow, then its cross, white with arms 4.5 design pixels long, or
// dim, shorter, its middle left out.
func (p *painter) sparkle(cx, cy, k float64, dim bool) {
	c, strength, arm, half := hexColour(cHomeSparkle.tc), 0.64, 4.5*k, 0.8*k
	if dim {
		c, strength, arm, half = hexColour(cHomeSparkleDim.tc), 0.35, 4*k, 0.65*k
	}
	glow, r := hexColour(cHomeSparkleGlow.tc), 8*k
	for y := int(cy - r); y <= int(cy+r); y++ {
		for x := int(cx - r); x <= int(cx+r); x++ {
			if d := math.Hypot(float64(x)+0.5-cx, float64(y)+0.5-cy) / r; d < 1 {
				p.blend(x, y, glow, strength*(1-d))
			}
		}
	}
	if dim {
		gap := 1.5 * k
		p.rect(cx-arm, cy-half, cx-gap, cy+half, c)
		p.rect(cx+gap, cy-half, cx+arm, cy+half, c)
		p.rect(cx-half, cy-arm, cx+half, cy-gap, c)
		p.rect(cx-half, cy+gap, cx+half, cy+arm, c)
		return
	}
	p.rect(cx-arm, cy-half, cx+arm, cy+half, c)
	p.rect(cx-half, cy-arm, cx+half, cy-half, c)
	p.rect(cx-half, cy+half, cx+half, cy+arm, c)
}

// banner draws the banner's ground, which fades from the page's at its top
// and bottom, its lattice of faint dots, its halftone bands, its stars and
// its sparkles (reference-design.md, The home screen, The banner); the
// sprites are the banner's scene's (BannerView). The bands and sparkles lie
// at the design's places, its banner's height scaled to this one's.
func (p *painter) banner(z bannerZone) {
	top, end := z.top, z.end
	fadeIn, full, fadeOut, out := z.fadeIn, z.full, z.fadeOut, z.out
	page, ground := hexColour(cHomePage.tc), hexColour(cHomeBanner.tc)
	for y := max(int(fadeIn), p.clip.Min.Y); y < min(int(math.Ceil(out)), p.clip.Max.Y); y++ {
		py := float64(y) + 0.5
		t := math.Min(math.Min((py-fadeIn)/(full-fadeIn), (out-py)/(out-fadeOut)), 1)
		if t <= 0 {
			continue
		}
		c := color.NRGBA{lerp8(page.R, ground.R, t), lerp8(page.G, ground.G, t), lerp8(page.B, ground.B, t), 0xFF}
		p.fill(image.Rect(p.clip.Min.X, y, p.clip.Max.X, y+1), c)
	}
	dot, band, rim := hexColour(cHomeDot.tc), hexColour(cHomeHalftone.tc), hexColour(cHomeHalftoneRim.tc)
	mix := func(a, b color.NRGBA, t float64) color.NRGBA {
		return color.NRGBA{lerp8(a.R, b.R, t), lerp8(a.G, b.G, t), lerp8(a.B, b.B, t), 0xFF}
	}
	dx, dy := 7.9*z.k, 8.2*z.k
	sy := (end - top) / (375 - designHead) // image pixels a design pixel down the banner
	q, e := 1.4*z.k, 0.8*z.k               // a band's dot's side, and its soft rim's width
	for y, j := top+dy/2, 0; y < end; y, j = y+dy, j+1 {
		for x, i := dx/2, 0; x < float64(p.g.W*SixelCellW); x, i = x+dx, i+1 {
			p.blend(int(x), int(y), dot, 1)
			if s := bandStrength(x/z.k, designHead+(y-top)/sy); s > 0 && s >= bandDither(i, j) {
				// The design's dot: a bright pixel, its edge soft and purple.
				x0, y0 := float64(int(x)), float64(int(y))
				p.rect(x0-e, y0-e, x0+q+e, y0+q+e, mix(ground, rim, 0.3+0.5*s))
				p.rect(x0, y0, x0+q, y0+q, mix(ground, band, (0.4+0.6*s)*(0.75+0.25*bandDither(j, i)/0.9)))
			}
		}
	}
	// Stars at places of a fixed seed, as many to the design's area as the
	// design's: 70 in its banner.
	rng := rand.New(rand.NewSource(52))
	star := hexColour(cHomeStar.tc)
	w, h := float64(p.g.W*SixelCellW), end-top
	for range int(w * h / (z.k * z.k) / (designW * 330) * 70) {
		x, y := rng.Float64()*w, top+rng.Float64()*h
		size := 1 + rng.Intn(3)
		a := 0.55 + rng.Float64()*0.45
		for oy := range size {
			for ox := range size {
				p.blend(int(x)+ox, int(y)+oy, star, a)
			}
		}
	}
	for _, s := range homeSparkles {
		if y := top + (s.y-designHead)*sy; y < end {
			p.sparkle(s.x*z.k, y, z.k, s.dim)
		}
	}
}

func lerp8(a, b uint8, t float64) uint8 { return uint8(float64(a) + (float64(b)-float64(a))*t + 0.5) }

// panelTitle draws a panel's title from design x, up design pixels higher:
// the chevron, the title, the dim line under them to under, then the
// purple rule from rule to end.
func (p *painter) panelTitle(x, up float64, title string, under, rule, end float64) {
	p.stroke([][2]float64{{x + 1.5, 397.4 - up}, {x + 5.1, 402.05 - up}, {x + 1.5, 406.7 - up}}, false, 1.5, 0, hexColour(cHomeChevron.tc))
	p.text(x+24.8, 402.15-up, title, tkPanel, cHomePanelTitle, 0)
	p.drect(x+2, 414-up, under, 416-up, hexColour(cHomeUnderline.tc))
	p.drect(rule, 414-up, end, 415-up, hexColour(cHomeRule.tc))
	p.drect(rule, 415-up, end, 416-up, hexColour(cHomeRuleLow.tc))
}

// panels draws the two panels (reference-design.md, The home screen, The
// panels), their tops l.Grow higher for NEW SESSION's notes (design D6).
func (p *painter) panels(l HomeLook) {
	frame := flat(cHomeFrame)
	up := l.Grow
	p.dbox(7.5, 380.5-up, 616, 710, 7, 1.2, 0, color.NRGBA{}, frame)
	p.dbox(631, 380.5-up, 1331, 710, 7, 1.2, 0, color.NRGBA{}, frame)
	p.panelTitle(26, up, "NEW SESSION", 166, 184, 594)
	p.panelTitle(655, up, "PROJECTS", 763, 786, 1310)
	p.newSession(l)
	p.projects(l)
}

// The option boxes' tops and bottoms, and the middles of their radios and
// texts, in design pixels.
var homeBoxes = [][3]float64{{457.5, 495.5, 476.8}, {498.5, 539, 519.8}}

// noteLine is the design pixels between two notes' lines, and noteChars
// the characters a line holds, from x 31 to the field's right edge.
const (
	noteLine  = 16.0
	noteChars = 70
)

// homeNoteLine is a line of the notes as drawn: its text, its kind,
// whether it goes on a note under its mark, and its note's item.
type homeNoteLine struct {
	text string
	kind byte
	more bool
	item int
}

// homeNoteLines are the notes wrapped in noteChars, each one's lines after
// its first under its text.
func homeNoteLines(notes []HomeNote) []homeNoteLine {
	var out []homeNoteLine
	for _, n := range notes {
		for i, t := range wrap(n.Text, noteChars-textWidth(n.Mark)) {
			if i == 0 {
				out = append(out, homeNoteLine{n.Mark + t, n.Kind, false, n.Item})
			} else {
				out = append(out, homeNoteLine{strings.Repeat(" ", textWidth(n.Mark)) + t, n.Kind, true, n.Item})
			}
		}
	}
	return out
}

// homeNoteColour is a note's colour by its kind.
func homeNoteColour(kind byte) colour {
	switch kind {
	case 'f':
		return cFailed
	case 'a':
		return cAttention
	}
	return cGrey
}

// homeFieldBox is the project field's box, its top and bottom in image
// pixels: the design's, l.Grow higher, centred on its text's row, the
// terminal's (design D4).
func homeFieldBox(l HomeLook) (y0, y1, dy float64) {
	g := l.Geom
	dy = float64(l.FieldRow*SixelCellH+SixelCellH/2) - g.Y(605.5-l.Grow)
	return g.Y(586-l.Grow) + dy, g.Y(625-l.Grow) + dy, dy
}

// newSession draws NEW SESSION's questions, its option boxes, the project
// field around its text's cells, the notes and START SESSION; what lies
// above the notes l.Grow higher, the focused control outlined in the
// design's purple (designs D6 and D10).
func (p *painter) newSession(l HomeLook) {
	up := l.Grow
	focus := hexColour(cHomeFocus.tc)
	p.text(30.85, 441.8-up, "1.", tkAsk, cHomeNumber, 0)
	p.text(60.9, 441.8-up, "Agent CLI", tkAsk, cHomeAsk, 0)
	for i, c := range l.CLIs {
		if i >= len(homeBoxes) {
			break
		}
		y0, y1, mid := homeBoxes[i][0]-up, homeBoxes[i][1]-up, homeBoxes[i][2]-up
		line, fill, name, ver := cHomeOtherLine, color.NRGBA{}, cHomeOtherName, cHomeOtherVer
		if c.Chosen {
			line, fill, name, ver = cHomeChosenLine, hexColour(cHomeChosenFill.tc), cHomeChosenName, cHomeChosenVer
		}
		if !c.Usable {
			name = cHomeOtherVer
		}
		outline := flat(line)
		if c.Chosen && l.Field == 0 {
			outline = flat(cHomeFocus)
		}
		p.dbox(29.5, y0, 594.5, y1, 6, 1.2, 0, fill, outline)
		if c.Chosen {
			p.ring(54.4, mid, 7.85, 1.3, 0, hexColour(cHomeRadioOn.tc))
			p.disc(54.4, mid, 4, 0, hexColour(cHomeRadioDot.tc))
		} else {
			p.ring(54.4, mid, 7.85, 1.3, 0, hexColour(cHomeRadioOff.tc))
		}
		p.text(78.4, mid, c.Name, tkCLI, name, 0)
		// The version a space after the name, a pixel further as the
		// design's, cut to end inside the box.
		vx := 78.4 + float64(len([]rune(c.Name))+1)*tkCLI.adv + 1.1
		p.text(vx, mid, fitChars(c.Version, homeVersionChars(c.Name)), tkVersion, ver, 0)
	}
	p.text(30.85, 569.88-up, "2.", tkAsk, cHomeNumber, 0)
	p.text(60.9, 569.88-up, "Project directory", tkAsk, cHomeAsk, 0)
	// The field, its outline lighter at its top than at its bottom, the
	// focus's purple with the focus.
	y0, y1, dy := homeFieldBox(l)
	top, low := hexColour(cHomeFieldTop.tc), hexColour(cHomeFieldLow.tc)
	if l.Field == 1 {
		top, low = focus, focus
	}
	grad := func(y float64) color.NRGBA {
		t := math.Min(math.Max((y-y0)/(y1-y0), 0), 1)
		return color.NRGBA{lerp8(top.R, low.R, t), lerp8(top.G, low.G, t), lerp8(top.B, low.B, t), 0xFF}
	}
	p.dbox(29.5, 586-up, 594, 625-up, 6, 1.2, dy, hexColour(cHomeField.tc), grad)
	p.rect(p.g.X(549), p.g.Y(588-up)+dy, p.g.X(550.2), p.g.Y(623-up)+dy, hexColour(cHomeDivider.tc))
	folder := [][2]float64{{562.5, 613.5}, {562.5, 599.5}, {563.5, 598.5}, {568.5, 598.5}, {570.5, 600.5}, {578.5, 600.5}, {579.5, 601.5}, {579.5, 613.5}}
	for i := range folder {
		folder[i][1] -= up
	}
	p.stroke(folder, true, 1.5, dy, hexColour(cHomeFolder.tc))
	// The notes from under the field, the last where the design leaves a
	// line's room above START SESSION.
	for i, n := range homeNoteLines(l.Notes) {
		p.text(31, 634.75-up+float64(i)*noteLine, n.text, tkRow, homeNoteColour(n.kind), 0)
	}
	// START SESSION, its label and arrow where the design has them; dimmed
	// halfway to the page; with the focus, outlined in a lighter colour.
	fill, ink := cHomeButton, cHomeButtonText
	if l.Dim {
		fill, ink = cHomeButtonDim, cHomeButtonTextDim
	}
	var ring func(float64) color.NRGBA
	if l.Field == 2 {
		ring = flat(cHomeFocusOnButton)
	}
	p.dbox(29.5, 644.5, 594, 688.5, 6, 1.2, 0, hexColour(fill.tc), ring)
	p.text(235, 667.37, "START SESSION", tkButton, ink, 0)
	arrow := hexColour(ink.tc)
	p.stroke([][2]float64{{374.5, 666.8}, {385.5, 666.8}}, false, 1.6, 0, arrow)
	p.stroke([][2]float64{{381.3, 662.5}, {385.5, 666.8}, {381.3, 671.1}}, false, 1.6, 0, arrow)
}

// homeVersionChars are the characters an option's version shows after the
// CLI's name, to end inside its box.
func homeVersionChars(name string) int {
	vx := 78.4 + float64(len([]rune(name))+1)*tkCLI.adv + 1.1
	return int((582 - vx) / tkVersion.adv)
}

// projects draws the Projects panel's rows: the headings with their icons,
// the projects, the selected one's band, brighter with the focus (design
// D10), and the separators; the store's problem or "No projects yet"
// above them; a slim mark at the right when they do not all fit (design
// D7).
func (p *painter) projects(l HomeLook) {
	sep := hexColour(cHomeSeparator.tc)
	if l.List.Text != "" {
		y := rowFirstHead - l.Grow
		for _, n := range homeListLines(l.List) {
			p.text(684, y, n.text, tkRow, homeNoteColour(n.kind), 0)
			y += noteLine
		}
	}
	for i, r := range l.Rows {
		y := r.Y
		switch r.Head {
		case 'f':
			pts := starPoints(662.8, y+0.15, 8, 3.4)
			p.stroke(pts, true, 1.3, 0, hexColour(cHomeSectionIcon.tc))
			p.text(682.64, y, "FAVORITES", tkSection, cHomeSection, 0)
		case 'r':
			ic := hexColour(cHomeClock.tc)
			p.ring(662.3, y, 7.5, 1.3, 0, ic)
			p.stroke([][2]float64{{662.3, y}, {662.3, y - 4.8}}, false, 1.3, 0, ic)
			p.stroke([][2]float64{{662.3, y}, {665.9, y + 2.1}}, false, 1.3, 0, ic)
			p.text(682.64, y, "RECENT PROJECTS", tkSection, cHomeSection, 0)
		default:
			name, path, age := cHomeName, cHomeRowPath, cHomeAge
			if r.Selected {
				// The band, a lighter line along its top and its sides;
				// brighter with the focus.
				band, line := cHomeBand, cHomeBandTop
				if l.Field == 3 {
					band, line = cHomeBandFocus, cHomeBandFocusTop
				}
				p.drect(644, y-15.8, 1319, y+15.8, hexColour(band.tc))
				p.drect(644, y-15.2, 1319, y-14.2, hexColour(line.tc))
				p.drect(644, y-14.2, 645, y+15.8, hexColour(cHomeBandSide.tc))
				p.drect(1318, y-14.2, 1319, y+15.8, hexColour(cHomeBandSide.tc))
				name, path, age = cHomeNameSel, cHomeRowPathSel, cHomeAgeSel
			}
			if r.Fav {
				p.polygon(starPoints(662.6, y+0.35, 8.3, 3.6), 0, hexColour(cHomeStarOn.tc))
			} else {
				p.stroke(starPoints(662.8, y+0.3, 7.6, 3.2), true, 1.2, 0, hexColour(cHomeStarOff.tc))
			}
			p.text(698.1, y, fitChars(r.Name, 25), tkName, name, 0)
			p.text(917.9, y, fitCharsLeft(r.Path, 38), tkRow, path, 0)
			p.text(1238.3, y, r.Age, tkRow, age, 0)
		}
		if i+1 < len(l.Rows) && !r.Selected && !l.Rows[i+1].Selected {
			// Midway to the next row; 15.5 under a heading, 16 under a
			// section's last row, as the design's.
			s := (y + l.Rows[i+1].Y) / 2
			switch {
			case r.Head != 0:
				s = y + 15.5
			case l.Rows[i+1].Head != 0:
				s = y + 16
			}
			p.drect(644, s-0.5, 1319, s+0.5, sep)
		}
	}
	// The slim mark: along the rows' height, its length and place those of
	// the projects shown among all.
	if first, shown, total := l.Scroll[0], l.Scroll[1], l.Scroll[2]; total > shown && len(l.Rows) > 0 {
		y0, y1 := l.Rows[0].Y-14, l.Rows[len(l.Rows)-1].Y+14
		h := y1 - y0
		t0 := y0 + h*float64(first)/float64(total)
		t1 := y0 + h*float64(first+shown)/float64(total)
		p.dbox(1322, t0, 1325, t1, 1.5, 0, 0, hexColour(cHomeFrame.tc), nil)
	}
}

// homeListLines are PROJECTS' note's lines, wrapped in the panel's width.
func homeListLines(n HomeNote) []homeNoteLine {
	var out []homeNoteLine
	for i, t := range wrap(n.Text, 76-textWidth(n.Mark)) {
		if i == 0 {
			out = append(out, homeNoteLine{n.Mark + t, n.Kind, false, 0})
		} else {
			out = append(out, homeNoteLine{strings.Repeat(" ", textWidth(n.Mark)) + t, n.Kind, true, 0})
		}
	}
	return out
}

// The footer's groups at the design's first three places (design D8,
// Ctrl+Q's in Ctrl+P's): each key cap's left edge, its key's and its
// action's first cells; and the bars between them.
var (
	footCaps    = [3]float64{141, 357.3, 582.5}
	footKeys    = [3]float64{150.61, 366.75, 591.83}
	footActions = [3]float64{219.2, 435.79, 659.36}
	footBars    = [2]float64{334.3, 559.5}
)

// footer draws the footer (reference-design.md, The home screen, The
// footer): its line, "Shortcuts:", and each shortcut's key cap and action,
// bars between them.
func (p *painter) footer(l HomeLook) {
	p.drect(0, 711, designW, 713, hexColour(cHomeFootLine.tc))
	p.stroke([][2]float64{{10.5, 734.9}, {15, 738.9}, {10.5, 742.9}}, false, 1.5, 0, hexColour(cHomeFootChevron.tc))
	p.text(32.86, 738.75, "Shortcuts:", tkShortcuts, cHomeShortcuts, 0)
	for i, k := range l.Keys {
		if i >= len(footCaps) {
			break
		}
		x := footCaps[i]
		p.dbox(x, 727, x+67, 750, 4, 1, 0, hexColour(cHomeCapFill.tc), flat(cHomeCapLine))
		p.text(footKeys[i], 738.75, k.Key, tkKey, cHomeCapText, 0)
		p.text(footActions[i], 738.75, k.Action, tkAction, cHomeAction, 0)
		if i < len(footBars) && i < len(l.Keys)-1 {
			p.drect(footBars[i]-0.55, 729.4, footBars[i]+0.55, 748.75, hexColour(cHomeFootBar.tc))
		}
	}
}
