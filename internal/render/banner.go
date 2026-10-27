package render

import (
	"math"
	"math/rand"
	"time"
)

// The banner's scene (plan 51, design D12): aliens appearing at random
// places across its width, drifting and descending a pixel at a time, and
// replaced; the ship wandering in every direction and firing two-bolt
// shots; a hit alien exploding into pink particles. A decorative scene,
// not a game: nothing in it answers a key. Its places are design pixels,
// its frames BannerTick apart; a seed makes it the same each run.

// BannerTick is the time between two of the banner's frames: 10 a second,
// the least of design D12's, each a terminal's work on every row it
// touches (plan 51, step 5: Windows Terminal at 10.9% of a core at 12.5).
const BannerTick = 100 * time.Millisecond

const bannerDT = 0.1 // seconds a frame, BannerTick's

// The aliens march as the classic game's: those of a band of their area
// step together, every stepFrames frames (0.4 s, the owner's livelier pace
// after step 7), the bands in turn, so that a frame touches the rows of
// one band (plan 51, step 5).
const (
	stepFrames = 4
	alienBands = 2
)

// The scene's bands, as fractions of its height: the aliens leave once
// their bottom passes alienLow, as low as the design's come; the ship
// wanders below shipHigh.
const (
	alienLow = 0.68
	shipHigh = 0.72
)

// The sprites' sizes in design pixels.
const (
	alienW, alienH = 11 * spritePx, 9 * spritePx
	shipW, shipH   = 17 * spritePx, 13 * spritePx
	boltW, boltH   = spritePx, 7 * spritePx
)

// alien is an alien of the scene: its kind ('l' lavender, 'p' purple), its
// top left, its drift and descent a second, its frame.
type alien struct {
	id     int
	kind   byte
	x, y   float64
	vx, vy float64
	frame  int
}

// shipState is the ship: its top left, where it goes, how fast, the time
// before it chooses elsewhere, before its next shot, and a second bolt to
// fire.
type shipState struct {
	x, y, tx, ty float64
	speed        float64
	wander       float64
	reload       float64
	second       bool
}

// bolt is a shot's bolt, its top left.
type bolt struct {
	id   int
	x, y float64
}

// particle is an explosion's square: its middle, its speed, its side in
// design pixels, its colour, its life and its age in seconds.
type particle struct {
	x, y, vx, vy float64
	size         float64
	colour       int
	life, age    float64
}

// blast is an explosion: its middle, its age, its particles.
type blast struct {
	id    int
	x, y  float64
	age   float64
	parts []particle
}

// The explosion as the design's (reference-design.md, The home screen, The
// banner): how long it lasts, its starburst's flash the start of it, and
// its purple haze's radius in design pixels.
const (
	blastLife  = 0.55
	blastFlash = 0.35
	blastHaze  = 26
)

// newBlast is an explosion at (x, y): its particles flying out in every
// direction from about its middle, small and bright.
func newBlast(rng *rand.Rand, id int, x, y float64) blast {
	e := blast{id: id, x: x, y: y}
	for range 36 {
		ang := rng.Float64() * 2 * math.Pi
		v := 30 + rng.Float64()*70
		size := 1.5 + rng.Float64()*1.5
		if rng.Intn(6) == 0 {
			size = 3.5
		}
		r := rng.Float64() * 3
		e.parts = append(e.parts, particle{x: x + math.Cos(ang)*r, y: y + math.Sin(ang)*r, vx: math.Cos(ang) * v, vy: math.Sin(ang) * v,
			size: size, colour: rng.Intn(len(blastParts)), life: 0.3 + rng.Float64()*0.25})
	}
	return e
}

// step ages the explosion dt seconds, its particles slowing as they fly.
func (e *blast) step(dt float64) {
	e.age += dt
	for i := range e.parts {
		p := &e.parts[i]
		p.age += dt
		p.x += p.vx * dt
		p.y += p.vy * dt
		p.vx *= 0.94
		p.vy *= 0.94
	}
}

// Banner is the banner's scene, w by h design pixels.
type Banner struct {
	W, H   float64
	Frame  int
	rng    *rand.Rand
	nextID int
	aliens []alien
	ship   shipState
	bolts  []bolt
	blasts []blast
	waits  []float64 // the seconds before each replaced alien appears
	// What the scene did, for its tests (plan 51, criterion 51.6).
	Spawned, Shots, Hits int
}

// NewBanner is a scene w by h design pixels, its aliens as many to its area
// as the design's (33 in its 1347 by 330), and no more than 40 in a taller
// one (grid mode's, design D2); from seed.
func NewBanner(w, h float64, seed int64) *Banner {
	b := &Banner{W: w, H: h, rng: rand.New(rand.NewSource(seed))}
	n := int(math.Round(w * h / 13500))
	n = min(max(n, 4), 40)
	for range n {
		b.spawn(true)
	}
	b.ship = shipState{x: (w - shipW) / 2, y: h - shipH - 4, speed: 80}
	b.ship.tx, b.ship.ty = b.ship.x, b.ship.y
	b.ship.reload = 0.5 + b.rng.Float64()
	return b
}

// resize makes the scene h high, as wide: the ship and where it goes back
// in its band; an alien now too low leaves at the next frame.
func (b *Banner) resize(h float64) {
	b.H = h
	lo, hi := h*shipHigh, math.Max(h-shipH-4, h*shipHigh)
	s := &b.ship
	s.y, s.ty = math.Min(math.Max(s.y, lo), hi), math.Min(math.Max(s.ty, lo), hi)
}

// spawn puts an alien at a random place in the upper part of the scene, at
// the start anywhere above the ship's band, clear of the others; when a few
// tries find no room, it comes a moment later.
func (b *Banner) spawn(start bool) {
	top := b.H * 0.5
	if start {
		top = b.H*alienLow - 4
	}
	var a alien
	for try := 0; ; try++ {
		if try == 10 {
			b.waits = append(b.waits, 0.3)
			return
		}
		a = alien{x: b.rng.Float64() * math.Max(b.W-alienW, 1), y: 2 + b.rng.Float64()*math.Max(top-alienH, 1)}
		if !b.crowded(-1, a.x, a.y) {
			break
		}
	}
	b.nextID++
	a.id = b.nextID
	a.kind = 'l'
	if b.rng.Intn(2) == 1 {
		a.kind = 'p'
	}
	a.vx = (10 + b.rng.Float64()*16) * float64(1-2*b.rng.Intn(2))
	a.vy = 4 + b.rng.Float64()*6
	a.frame = b.rng.Intn(2)
	b.aliens = append(b.aliens, a)
	b.Spawned++
}

// Step moves the scene on a frame.
func (b *Banner) Step() {
	b.Frame++
	dt := bannerDT
	// The aliens step a sprite's pixel or so at a time, their band's turn,
	// turn at the edges and before another, wait above one, and leave once
	// they come down to the ship's band.
	step := float64(stepFrames) * dt
	for i := range b.aliens {
		a := &b.aliens[i]
		band := min(int(a.y/(b.H*alienLow/alienBands)), alienBands-1)
		if (b.Frame+band*stepFrames/alienBands)%stepFrames != 0 {
			continue
		}
		a.frame ^= 1
		x := a.x + a.vx*step
		if x < 0 || x > b.W-alienW || b.crowded(i, x, a.y) {
			a.vx, x = -a.vx, a.x
		}
		y := a.y + a.vy*step
		if b.crowded(i, x, y) {
			y = a.y
		}
		a.x, a.y = x, y
	}
	kept := b.aliens[:0]
	for _, a := range b.aliens {
		if a.y > b.H*alienLow-alienH {
			b.waits = append(b.waits, 0.4+b.rng.Float64()*0.8)
			continue
		}
		kept = append(kept, a)
	}
	b.aliens = kept
	// Those replaced appear when their wait is over.
	waits := b.waits[:0]
	for _, w := range b.waits {
		if w -= dt; w <= 0 {
			b.spawn(false)
			continue
		}
		waits = append(waits, w)
	}
	b.waits = waits
	b.moveShip(dt)
	b.moveBolts(dt)
	// The explosions grow old and go.
	blasts := b.blasts[:0]
	for _, e := range b.blasts {
		if e.step(dt); e.age < blastLife {
			blasts = append(blasts, e)
		}
	}
	b.blasts = blasts
}

// alienGap is the room an alien keeps round it, in design pixels.
const alienGap = 4

// crowded says alien i at (x, y) would come within alienGap of another.
func (b *Banner) crowded(i int, x, y float64) bool {
	for j, o := range b.aliens {
		if j != i && math.Abs(o.x-x) < alienW+alienGap && math.Abs(o.y-y) < alienH+alienGap {
			return true
		}
	}
	return false
}

// moveShip moves the ship toward where it goes, chooses elsewhere when it
// is there or after a while, anywhere in the lower part of the scene; and
// fires when its gun is ready and an alien is above it, or now and then
// when none is.
func (b *Banner) moveShip(dt float64) {
	s := &b.ship
	dx, dy := s.tx-s.x, s.ty-s.y
	if d := math.Hypot(dx, dy); d > 1 {
		step := math.Min(s.speed*dt, d)
		s.x += dx / d * step
		s.y += dy / d * step
	}
	s.wander -= dt
	if math.Hypot(dx, dy) <= 1 || s.wander <= 0 {
		s.tx = b.W*0.06 + b.rng.Float64()*math.Max(b.W*0.88-shipW, 1)
		s.ty = b.H*shipHigh + b.rng.Float64()*math.Max(b.H*(1-shipHigh)-shipH-4, 1)
		// A leg's pace, design pixels a second, slow to fast, as the owner
		// raised it after step 7 (plan 51).
		s.speed = 125 + b.rng.Float64()*150
		s.wander = 1.5 + b.rng.Float64()*2.5
	}
	if s.second {
		s.second = false
		b.fire()
	}
	s.reload -= dt
	if s.reload > 0 {
		return
	}
	above := false
	nose := s.x + shipW/2
	for _, a := range b.aliens {
		if math.Abs(a.x+alienW/2-nose) < 26 {
			above = true
			break
		}
	}
	if above || b.rng.Intn(5) == 0 {
		b.fire()
		s.second = true
		s.reload = 1.1 + b.rng.Float64()*0.8
	} else {
		s.reload = 0.2
	}
}

// fire puts a bolt at the ship's nose.
func (b *Banner) fire() {
	b.nextID++
	b.bolts = append(b.bolts, bolt{id: b.nextID, x: b.ship.x + shipW/2 - boltW/2, y: b.ship.y - boltH})
	b.Shots++
}

// moveBolts moves the bolts up; one that meets an alien on its way
// explodes it.
func (b *Banner) moveBolts(dt float64) {
	kept := b.bolts[:0]
	for _, bl := range b.bolts {
		from := bl.y + boltH // the bolt's bottom before the move: it sweeps from there
		bl.y -= 420 * dt
		hit := -1
		for i, a := range b.aliens {
			if bl.x+boltW > a.x && bl.x < a.x+alienW && bl.y < a.y+alienH && from > a.y {
				hit = i
				break
			}
		}
		if hit >= 0 {
			a := b.aliens[hit]
			b.explode(a.x+alienW/2, a.y+alienH/2)
			b.aliens = append(b.aliens[:hit], b.aliens[hit+1:]...)
			b.waits = append(b.waits, 1.0+b.rng.Float64()*1.2)
			b.Hits++
			continue
		}
		if bl.y+boltH > 0 {
			kept = append(kept, bl)
		}
	}
	b.bolts = kept
}

// explode starts an explosion at (x, y).
func (b *Banner) explode(x, y float64) {
	b.nextID++
	b.blasts = append(b.blasts, newBlast(b.rng, b.nextID, x, y))
}
