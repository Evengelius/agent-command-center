package render

import (
	"image"
	"image/color"
	"math"
)

// The banner's sprites (plan 51, design D12), read from the owner's image,
// references/16-home-screen.webp, enlarged on a grid of its pixel art (each
// of its pixels about 2.5 design pixels): reference-design.md, The home
// screen, The banner. A grid's letters are its colours; '.' is clear.

// spritePx is a sprite's pixel in design pixels.
const spritePx = 2.5

// spriteArt is a sprite's pixel grid, one frame or two, and its colours by
// letter.
type spriteArt struct {
	frames  [][]string
	palette map[byte]color.NRGBA
}

var (
	// The lavender alien: antennae, two dark eyes, arms down at its sides;
	// its second frame its arms up, its feet apart.
	artLavender = spriteArt{
		frames: [][]string{{
			"...L...L...",
			"...LLLLL...",
			"..LLLLLLL..",
			".LLeLLLeLL.",
			"LLLLLLLLLLL",
			"MMMMMMMMMMM",
			"M.MMMMMMM.M",
			"..M.....M..",
			"...M...M...",
		}, {
			"...L...L...",
			"M..LLLLL..M",
			"M.LLLLLLL.M",
			"MLLeLLLeLLM",
			"LLLLLLLLLLL",
			"MMMMMMMMMMM",
			"..MMMMMMM..",
			"..M.....M..",
			".M.......M.",
		}},
		palette: map[byte]color.NRGBA{'L': {0xDB, 0xC0, 0xFC, 0xFF}, 'M': {0xD0, 0xB2, 0xF8, 0xFF}, 'e': {0x2A, 0x1A, 0x48, 0xFF}},
	}
	// The purple alien: antennae, a wide body with two dark eyes, four legs;
	// its second frame its legs the other way.
	artPurple = spriteArt{
		frames: [][]string{{
			"...P...P...",
			"...PPPPP...",
			"..PPPPPPP..",
			".PPPPPPPPP.",
			"PPPePPPePPP",
			"QQQQQQQQQQQ",
			"Q.QQQ.QQQ.Q",
			"Q.R.....R.Q",
			"..R.....R..",
		}, {
			"...P...P...",
			"...PPPPP...",
			"..PPPPPPP..",
			".PPPPPPPPP.",
			"PPPePPPePPP",
			"QQQQQQQQQQQ",
			".QQ.QQQ.QQ.",
			".R..R.R..R.",
			"R..R...R..R",
		}},
		palette: map[byte]color.NRGBA{'P': {0x9E, 0x7C, 0xFC, 0xFF}, 'Q': {0x93, 0x73, 0xF0, 0xFF}, 'R': {0x84, 0x69, 0xD6, 0xFF}, 'e': {0x1E, 0x12, 0x46, 0xFF}},
	}
	// The ship: a pointed nose, a dark cockpit, swept wings with fins, two
	// exhaust blocks.
	artShip = spriteArt{
		frames: [][]string{{
			"........N........",
			".......NNN.......",
			".......NNN.......",
			"......SSSSS......",
			".....SScccSS.....",
			"...F..ScbcS..F...",
			"...F.SScbcSS.F...",
			"..FSSSScccSSSSF..",
			"WWSSSSSSSSSSSSSWW",
			"WWVVVVVVVVVVVVVWW",
			"WW.............WW",
			".....XX...XX.....",
			".....XX...XX.....",
		}},
		palette: map[byte]color.NRGBA{'N': {0xDA, 0xC2, 0xFA, 0xFF}, 'S': {0xD4, 0xC0, 0xFD, 0xFF}, 'F': {0xEB, 0xE3, 0xFB, 0xFF},
			'W': {0xDF, 0xC7, 0xFF, 0xFF}, 'V': {0xA8, 0x8C, 0xF4, 0xFF}, 'c': {0x22, 0x16, 0x58, 0xFF}, 'b': {0x5A, 0x4A, 0xC8, 0xFF},
			'X': {0xC2, 0xA8, 0xF0, 0xFF}},
	}
	// A shot's bolt: one pixel across, seven high, white.
	artBolt = spriteArt{
		frames:  [][]string{{"B", "B", "B", "B", "B", "B", "B"}},
		palette: map[byte]color.NRGBA{'B': {0xFF, 0xF6, 0xFF, 0xFF}},
	}
)

// The glows round the sprites, their colour and strength: purple round the
// aliens and the ship, brighter round a bolt.
var (
	glowSprite = color.NRGBA{0x7A, 0x5C, 0xE8, 0xFF}
	glowBolt   = color.NRGBA{0xB0, 0x78, 0xF8, 0xFF}
)

// The explosion's colours, read from the design's: its starburst's middle
// and rays, the purple haze round it, its particles' pinks.
var (
	blastCore  = color.NRGBA{0xFF, 0xC8, 0xFF, 0xFF}
	blastRay   = color.NRGBA{0xFF, 0xA8, 0xFC, 0xFF}
	blastHazeC = color.NRGBA{0x5A, 0x0C, 0x7A, 0xFF}
	blastParts = []color.NRGBA{{0xFF, 0xB6, 0xFF, 0xFF}, {0xFF, 0xC6, 0xFF, 0xFF}, {0xF9, 0xA6, 0xFC, 0xFF}, {0xE8, 0x90, 0xF0, 0xFF}}
)

// spriteImage is a sprite's frame drawn f image pixels a pixel, as the
// design's 2.5 design pixels a pixel are, with its glow g pixels wide round
// it in colour gc. Each image pixel takes the sprite pixel under its middle,
// crisp as pixel art, its colour shaded a little lighter at the top than at
// the bottom, as the design's are, in three bands; the glow under the
// sprite is in two steps of alpha, the nearer half stronger. Few colours,
// so that a tile of the moving sprite stays small (plan 51, design D12).
// The image's origin is the sprite's first pixel less the glow.
func spriteImage(a spriteArt, frame int, f float64, g int, gc color.NRGBA, strength float64) *image.NRGBA {
	rows := a.frames[frame%len(a.frames)]
	h, w := len(rows), 0
	for _, r := range rows {
		w = max(w, len(r))
	}
	cell := func(r, c int) (color.NRGBA, bool) {
		if r < 0 || r >= h || c < 0 || c >= len(rows[r]) {
			return color.NRGBA{}, false
		}
		col, ok := a.palette[rows[r][c]]
		shade := 1.03 - 0.04*float64(3*r/h) // lighter at the top, in thirds
		return color.NRGBA{scale8(col.R, shade), scale8(col.G, shade), scale8(col.B, shade), 0xFF}, ok
	}
	gf := float64(g)
	out := image.NewNRGBA(image.Rect(0, 0, int(math.Ceil(float64(w)*f))+2*g, int(math.Ceil(float64(h)*f))+2*g))
	for y := range out.Rect.Dy() {
		for x := range out.Rect.Dx() {
			px, py := float64(x)+0.5, float64(y)+0.5
			r0, c0 := int(math.Floor((py-gf)/f)), int(math.Floor((px-gf)/f))
			if col, ok := cell(r0, c0); ok {
				out.SetNRGBA(x, y, col)
				continue
			}
			// The glow: by the distance from the pixel's middle to the
			// nearest sprite pixel's square, among those near enough to
			// glow there.
			best := math.Inf(1)
			reach := int(math.Ceil(gf/f)) + 1
			for r := max(r0-reach, 0); r <= min(r0+reach, h-1); r++ {
				for c := max(c0-reach, 0); c <= min(c0+reach, len(rows[r])-1); c++ {
					if _, ok := cell(r, c); !ok {
						continue
					}
					sx, sy := gf+float64(c)*f, gf+float64(r)*f
					dx := math.Max(math.Max(sx-px, px-sx-f), 0)
					dy := math.Max(math.Max(sy-py, py-sy-f), 0)
					best = math.Min(best, math.Hypot(dx, dy))
				}
			}
			switch {
			case best < gf/2:
				out.SetNRGBA(x, y, color.NRGBA{gc.R, gc.G, gc.B, uint8(255 * strength * 0.6)})
			case best < gf:
				out.SetNRGBA(x, y, color.NRGBA{gc.R, gc.G, gc.B, uint8(255 * strength * 0.25)})
			}
		}
	}
	return out
}

// scale8 is v times f, within a byte.
func scale8(v uint8, f float64) uint8 { return uint8(math.Min(math.Round(float64(v)*f), 255)) }

// over lays src over dst at (x, y), src's alpha as its coverage, within
// clip.
func over(dst, src *image.NRGBA, x, y int, clip image.Rectangle) {
	r := image.Rect(x, y, x+src.Rect.Dx(), y+src.Rect.Dy()).Intersect(clip).Intersect(dst.Rect)
	for py := r.Min.Y; py < r.Max.Y; py++ {
		for px := r.Min.X; px < r.Max.X; px++ {
			s := src.Pix[src.PixOffset(px-x, py-y):]
			a := int(s[3])
			if a == 0 {
				continue
			}
			d := dst.Pix[dst.PixOffset(px, py):]
			if a == 255 {
				d[0], d[1], d[2] = s[0], s[1], s[2]
				continue
			}
			d[0] = uint8((int(s[0])*a + int(d[0])*(255-a) + 127) / 255)
			d[1] = uint8((int(s[1])*a + int(d[1])*(255-a) + 127) / 255)
			d[2] = uint8((int(s[2])*a + int(d[2])*(255-a) + 127) / 255)
		}
	}
}
