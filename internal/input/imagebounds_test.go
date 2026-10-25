package input

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"hash/crc32"
	"runtime"
	"strings"
	"testing"
)

// allocated is the bytes f allocates.
func allocated(f func()) uint64 {
	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	f()
	runtime.ReadMemStats(&after)
	return after.TotalAlloc - before.TotalAlloc
}

// craftedTIFF is a little-endian TIFF whose directory gives a width and a
// height of 4 px, then n entries of tag, each 1,048,576 BYTE values over the
// same 1 MiB after the directory (audit finding 2).
func craftedTIFF(tag uint16, n int) []byte {
	le := binary.LittleEndian
	entries := 2 + n
	values := 8 + 2 + entries*12 + 4
	b := make([]byte, values+1<<20)
	copy(b, "II*\x00")
	le.PutUint32(b[4:], 8)
	le.PutUint16(b[8:], uint16(entries))
	put := func(i int, tag, typ uint16, count, value uint32) {
		e := b[10+i*12:]
		le.PutUint16(e[0:], tag)
		le.PutUint16(e[2:], typ)
		le.PutUint32(e[4:], count)
		le.PutUint32(e[8:], value)
	}
	put(0, 256, 4, 1, 4)
	put(1, 257, 4, 1, 4)
	for i := range n {
		put(2+i, tag, 1, 1<<20, uint32(values))
	}
	return b
}

// craftedICO is an icon whose one image is a PNG declaring w by h px, 8-bit
// grey, with a little image data: enough for png.Decode to make its image
// before it runs out of data.
func craftedICO(w, h int) []byte {
	chunk := func(typ string, data []byte) []byte {
		out := binary.BigEndian.AppendUint32(nil, uint32(len(data)))
		out = append(append(out, typ...), data...)
		return binary.BigEndian.AppendUint32(out, crc32.ChecksumIEEE(append([]byte(typ), data...)))
	}
	ihdr := make([]byte, 13)
	binary.BigEndian.PutUint32(ihdr[0:], uint32(w))
	binary.BigEndian.PutUint32(ihdr[4:], uint32(h))
	ihdr[8] = 8 // 8 bits a sample, colour type 0: grey
	var z bytes.Buffer
	zw := zlib.NewWriter(&z)
	_, _ = zw.Write(make([]byte, 64))
	_ = zw.Close()
	png := append([]byte("\x89PNG\r\n\x1a\n"), chunk("IHDR", ihdr)...)
	png = append(append(png, chunk("IDAT", z.Bytes())...), chunk("IEND", nil)...)
	ico := make([]byte, 22)
	binary.LittleEndian.PutUint16(ico[2:], 1) // an icon
	binary.LittleEndian.PutUint16(ico[4:], 1) // one image, 256x256 by its directory
	binary.LittleEndian.PutUint32(ico[6+8:], uint32(len(png)))
	binary.LittleEndian.PutUint32(ico[6+12:], 22)
	return append(ico, png...)
}

// TestCraftedImagesBounded is plan 48's criteria 48.1 and 48.2 (design D1;
// audit finding 2): crafted TIFF and ICO files are refused with under 8 MiB
// allocated, where the TIFF reader copied every entry's values and the ICO
// reader decoded a PNG before reading its size.
func TestCraftedImagesBounded(t *testing.T) {
	const bound = 8 << 20
	for _, c := range []struct {
		name string
		b    []byte
		want string
	}{
		{"unused-tags.tif", craftedTIFF(700, 16), "TIFF: photometric 2 with 1 samples: not read"},
		{"strip-offsets.tif", craftedTIFF(273, 16), "TIFF: photometric 2 with 1 samples: not read"},
		// Four more widths after the first, whose value is 0: the first wins.
		{"widths.tif", craftedTIFF(256, 4), "TIFF: photometric 2 with 1 samples: not read"},
		{"huge-png.ico", craftedICO(9000, 9000), "ICO: 9000x9000 px: over 60 megapixels"},
	} {
		var err error
		n := allocated(func() { _, _, _, err = InspectData(c.name, c.b, claudeLimits) })
		if err == nil || !strings.Contains(err.Error(), c.want) || n > bound {
			t.Errorf("%s: %v, %.1f MiB allocated; want %q with under %d MiB", c.name, err, float64(n)/(1<<20), c.want, bound>>20)
		}
		t.Logf("%s: %v, %.2f MiB allocated", c.name, err, float64(n)/(1<<20))
	}
}
