# Test images (plan 31)

`acc-image-upload-test.webp` is the dashboard's own test image (a red, a
green and a blue square and the text "ACC IMAGE UPLOAD TEST", 320x160) as
WebP, which Go's standard library cannot write. Every other format of the
same image is drawn and written at test time by
`internal/input/imagetest`; this file was made once, on 2026-10-23, by a
browser canvas drawing the same pixels with the same rectangles and 5x7
font (`imagetest.Draw`), in a page's console:

```js
const W = 320, H = 160, c = document.createElement("canvas");
c.width = W; c.height = H;
const ctx = c.getContext("2d"), id = ctx.createImageData(W, H), p = id.data;
const set = (x, y, r, g, b) => { const i = (y * W + x) * 4; p[i] = r; p[i+1] = g; p[i+2] = b; p[i+3] = 255; };
// white, the three squares, then the text at (34, 112), each font pixel 2 px:
// imagetest.Squares and imagetest's font, row by row
ctx.putImageData(id, 0, 0);
c.toDataURL("image/webp", 1.0); // lossless: VP8X with an ICC profile, VP8L
```

The checks made then: the SHA-256 of the canvas's pixels (RGBA, row by row)
equals `imagetest.PixelsSHA256`
(`1e89b6e2a41bfb8f95cdb875268b17206b79502c0bba1e8ba36bd3df897c1f5c`, which
`TestDrawPixels` pins); the WebP decoded again in the browser gives the same
pixels; the file's SHA-256 is
`79cf28692ec70c8a775efe5995022df3ebac05e3aaf320f5956c727859757659` (1,420
bytes).

No personal file is used anywhere in the tests.
