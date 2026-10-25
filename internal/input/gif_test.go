package input

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"image"
	"image/color"
	"image/gif"
	"runtime"
	"slices"
	"testing"

	"github.com/Evengelius/agent-command-center/internal/input/imagetest"
)

// gifParts cuts a GIF made here into its header, with its global colour
// table, and its top-level blocks, the trailer last.
func gifParts(t *testing.T, b []byte) (head []byte, blocks [][]byte) {
	t.Helper()
	i := 13
	if b[10]&0x80 != 0 {
		i += 3 << (1 + b[10]&7)
	}
	subs := func(j int) int {
		for b[j] != 0 {
			j += 1 + int(b[j])
		}
		return j + 1
	}
	head = b[:i]
	for i < len(b) {
		start := i
		switch b[i] {
		case 0x21: // label, then sub-blocks
			i = subs(i + 2)
		case 0x2C: // descriptor, colour table, code size, sub-blocks
			j := i + 10
			if b[i+9]&0x80 != 0 {
				j += 3 << (1 + b[i+9]&7)
			}
			i = subs(j + 1)
		case 0x3B:
			i++
		default:
			t.Fatalf("block 0x%02x at %d", b[i], i)
		}
		blocks = append(blocks, b[start:i])
	}
	return head, blocks
}

// join is a GIF of its header and blocks.
func join(head []byte, blocks ...[]byte) []byte {
	return bytes.Join(append([][]byte{head}, blocks...), nil)
}

// frameAt is the index of the n-th frame (from 0) among the blocks.
func frameAt(blocks [][]byte, n int) int {
	for i, b := range blocks {
		if b[0] == 0x2C {
			if n == 0 {
				return i
			}
			n--
		}
	}
	return -1
}

// testFrame is a frame r of palette p with a pattern, k shifting it.
func testFrame(r image.Rectangle, p color.Palette, k int) *image.Paletted {
	m := image.NewPaletted(r, p)
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			m.SetColorIndex(x, y, uint8((x/7+y/5+k)%len(p)))
		}
	}
	return m
}

func encodeGIF(t testing.TB, g *gif.GIF) []byte {
	t.Helper()
	var b bytes.Buffer
	if err := gif.EncodeAll(&b, g); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

var testPalette = color.Palette{color.NRGBA{}, color.NRGBA{255, 255, 255, 255}, color.NRGBA{200, 30, 30, 255}, color.NRGBA{30, 160, 60, 255}, color.NRGBA{40, 70, 200, 255}}

// offsetGIF has three frames of other sizes and offsets, with disposals and
// a transparent colour.
func offsetGIF(t testing.TB) []byte {
	return encodeGIF(t, &gif.GIF{
		Image:     []*image.Paletted{testFrame(image.Rect(20, 10, 120, 60), testPalette, 0), testFrame(image.Rect(0, 0, 320, 160), testPalette, 1), testFrame(image.Rect(100, 50, 200, 150), testPalette, 2)},
		Delay:     []int{10, 20, 30},
		Disposal:  []byte{gif.DisposalNone, gif.DisposalBackground, gif.DisposalPrevious},
		LoopCount: 2,
		Config:    image.Config{ColorModel: testPalette, Width: 320, Height: 160},
	})
}

// gifCases are the GIFs plan 49's D12 is checked on: imagetest's two,
// offsetGIF, frames with their own colour tables and no global one, and
// comment and plain text extensions between frames.
func gifCases(t *testing.T) map[string][]byte {
	t.Helper()
	out := map[string][]byte{"offsets": offsetGIF(t)}
	var buf bytes.Buffer
	if err := gif.Encode(&buf, imagetest.Draw(), nil); err != nil {
		t.Fatal(err)
	}
	out["single"] = bytes.Clone(buf.Bytes())
	buf.Reset()
	if err := imagetest.EncodeAnimatedGIF(&buf, imagetest.Draw()); err != nil {
		t.Fatal(err)
	}
	out["animated"] = bytes.Clone(buf.Bytes())
	other := color.Palette{color.NRGBA{10, 10, 10, 255}, color.NRGBA{250, 220, 0, 255}, color.NRGBA{0, 200, 220, 255}}
	out["local tables"] = encodeGIF(t, &gif.GIF{
		Image:  []*image.Paletted{testFrame(image.Rect(0, 0, 64, 48), other, 0), testFrame(image.Rect(8, 8, 40, 40), testPalette, 1)},
		Delay:  []int{5, 5},
		Config: image.Config{Width: 64, Height: 48},
	})
	head, blocks := gifParts(t, out["offsets"])
	k := frameAt(blocks, 0)
	comment := []byte{0x21, 0xFE, 5, 'h', 'e', 'l', 'l', 'o', 0}
	text := []byte{0x21, 0x01, 12, 0, 0, 0, 0, 10, 0, 10, 0, 8, 8, 1, 0, 2, 'h', 'i', 0}
	out["extensions"] = join(head, slices.Concat(blocks[:k+1], [][]byte{comment, text}, blocks[k+1:])...)
	return out
}

// gifSent is what each case sends, measured on a clean copy of the code
// before plan 49 (DecodeAll): the media type, the note, the size, the bytes'
// length and SHA-256.
var gifSent = map[string]string{
	"single":       "image/gif |  | 320x160 | 2232 | d52c8320b56e26ec36ef9f166d8fd99bffb9e603b81fcf3efe312f4491b1f1b5",
	"animated":     "image/png | animated: the first frame is sent | 320x160 | 1177 | 3d0cdc53882ee89b1a13e80766dc4a659eb8f4505aa1d61d2bbd7448c59213a5",
	"offsets":      "image/png | animated: the first frame is sent | 320x160 | 1521 | 640c72850d6f80be2f6a6ed02b11de6cf284afeb81a1ff6385ef47d657c44bbc",
	"local tables": "image/png | animated: the first frame is sent | 64x48 | 206 | e60a1d4451dfcf89a2aff08b9668dbdd114867d5ad92be39c2a07f84e4b9ebcd",
	"extensions":   "image/png | animated: the first frame is sent | 320x160 | 1521 | 640c72850d6f80be2f6a6ed02b11de6cf284afeb81a1ff6385ef47d657c44bbc",
}

// TestGIFSent is plan 49's criterion 49.13 (design D12): every case sends
// the bytes it sent when every frame was decoded, a single frame's file as
// it is, an animated GIF's first frame as PNG.
func TestGIFSent(t *testing.T) {
	for name, b := range gifCases(t) {
		img, data, _, err := InspectData(name+".gif", b, claudeLimits)
		if err != nil {
			t.Errorf("%s: refused: %v", name, err)
			continue
		}
		got := fmt.Sprintf("%s | %s | %dx%d | %d | %x", img.Media, img.Note, img.Width, img.Height, len(data), sha256.Sum256(data))
		if want, ok := gifSent[name]; !ok {
			t.Logf("SENT %q: %q,", name, got)
		} else if got != want {
			t.Errorf("%s: sends %s\nwant         %s", name, got, want)
		}
		if name == "single" && !bytes.Equal(data, b) {
			t.Errorf("the single frame is not sent as it is")
		}
	}
}

// gifRefused is each damaged GIF's refusal, measured as gifSent is; "" for
// a GIF taken. The second frame's damaged pixels were refused before plan 49,
// with "GIF: gif: reading image data: lzw: invalid code".
var gifRefused = map[string]string{
	"cut after the first frame":    "GIF: gif: reading frames: unexpected EOF",
	"no trailer":                   "GIF: gif: reading frames: unexpected EOF",
	"unknown block":                "GIF: gif: unknown block type: 0x99",
	"unknown extension":            "GIF: gif: unknown extension 0x42",
	"graphic control's size":       "GIF: gif: invalid graphic control extension block size: 5",
	"graphic control's terminator": "GIF: gif: invalid graphic control extension block terminator: 7",
	"second frame's bounds":        "GIF: gif: frame bounds larger than image bounds",
	"second frame's code size":     "GIF: gif: pixel size in decode out of range: 12",
	"second frame cut":             "GIF: gif: not enough image data",
	"second frame's pixels":        "",
}

// TestGIFRefused is plan 49's criterion 49.13: a GIF cut after its first
// frame, or holding an unknown block, is refused as it was, in the same
// words; so is every other fault of the file's structure. A later frame's
// damaged pixels, which are never sent, no longer refuse the file.
func TestGIFRefused(t *testing.T) {
	b := offsetGIF(t)
	head, blocks := gifParts(t, b)
	k, second := frameAt(blocks, 0), frameAt(blocks, 1)
	patched := func(i int, edit func(p []byte)) []byte {
		bs := slices.Clone(blocks)
		bs[i] = bytes.Clone(bs[i])
		edit(bs[i])
		return join(head, bs...)
	}
	after := func(i int, extra ...byte) []byte {
		return join(head, slices.Concat(blocks[:i+1], [][]byte{extra}, blocks[i+1:])...)
	}
	data := 10 + 1 // the second frame's sub-blocks start after its descriptor and code size (no local table)
	cases := map[string][]byte{
		"cut after the first frame":    join(head, blocks[:k+1]...),
		"no trailer":                   b[:len(b)-1],
		"unknown block":                after(k, 0x99),
		"unknown extension":            after(k, 0x21, 0x42, 0),
		"graphic control's size":       after(k, 0x21, 0xF9, 5, 0, 0, 0, 0, 0),
		"graphic control's terminator": after(k, 0x21, 0xF9, 4, 0, 0, 0, 0, 7),
		"second frame's bounds":        patched(second, func(p []byte) { p[5], p[6] = 0x90, 0x01 }),
		"second frame's code size":     patched(second, func(p []byte) { p[10] = 12 }),
		"second frame cut":             join(head, append(slices.Clone(blocks[:second]), blocks[second][:len(blocks[second])/2])...),
		"second frame's pixels": patched(second, func(p []byte) {
			for i := data; i < len(p)-1; {
				n := int(p[i])
				for j := i + 1; j <= i+n; j++ {
					p[j] = 0xFF
				}
				i += 1 + n
			}
		}),
	}
	for name, c := range cases {
		_, _, _, err := InspectData(name+".gif", c, claudeLimits)
		got := ""
		if err != nil {
			got = err.Error()
		}
		if want, ok := gifRefused[name]; !ok {
			t.Logf("REFUSED %q: %q,", name, got)
		} else if got != want {
			t.Errorf("%s: %q, want %q", name, got, want)
		}
	}
}

// FuzzGIFFrames is plan 49's design D12: the walk takes every GIF that
// image/gif's DecodeAll reads, with as many frames, so no file sent before is
// refused now. Small canvases only: DecodeAll keeps every frame.
func FuzzGIFFrames(f *testing.F) {
	f.Add(offsetGIF(f))
	var buf bytes.Buffer
	if err := imagetest.EncodeAnimatedGIF(&buf, imagetest.Draw()); err != nil {
		f.Fatal(err)
	}
	f.Add(buf.Bytes())
	f.Fuzz(func(t *testing.T, b []byte) {
		cfg, err := gif.DecodeConfig(bytes.NewReader(b))
		if err != nil || len(b) > 64<<10 || cfg.Width*cfg.Height > 64<<10 {
			return
		}
		g, err := gif.DecodeAll(bytes.NewReader(b))
		n, werr := gifFrames(b)
		if err == nil && (werr != nil || n != len(g.Image)) {
			t.Fatalf("DecodeAll reads %d frames, the walk %d: %v", len(g.Image), n, werr)
		}
	})
}

// TestGIFMemory is plan 49's criterion 49.13: a 200-frame 640x480 animated
// GIF is inspected with under 8 MiB allocated; with every frame decoded, it
// took 65.8 MiB.
func TestGIFMemory(t *testing.T) {
	f := testFrame(image.Rect(0, 0, 640, 480), testPalette, 0)
	g := &gif.GIF{Config: image.Config{ColorModel: testPalette, Width: 640, Height: 480}}
	for range 200 {
		g.Image, g.Delay = append(g.Image, f), append(g.Delay, 4)
	}
	b := encodeGIF(t, g)
	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	img, _, _, err := InspectData("long.gif", b, claudeLimits)
	runtime.ReadMemStats(&after)
	if err != nil || img.Note != noteFirst {
		t.Fatalf("%+v, %v", img, err)
	}
	alloc := after.TotalAlloc - before.TotalAlloc
	t.Logf("ALLOC %d bytes (%.1f MiB) for %d bytes of GIF", alloc, float64(alloc)/(1<<20), len(b))
	if alloc >= 8<<20 {
		t.Errorf("%.1f MiB allocated, the limit 8 MiB", float64(alloc)/(1<<20))
	}
}
