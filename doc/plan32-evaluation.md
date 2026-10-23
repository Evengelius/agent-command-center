# Plan 32 evaluation (2026-10-23)

The check of plan 32, image paste and preview (brief amendment 32: "now I
also want to be allow to copy and paste image in the terminal chat just
like I did here", and "Is there a way btw to have a preview before
sending like here on chat?"). It covers the criteria written in
[`plan.md`, Plan 32](../plan.md) before the code, with the corrections
recorded there. Nothing is committed: the owner reviews the work.

How it was checked:
- The research first (doc/research/image-paste.md): probe P32a sent each
  of Ctrl+V, Alt+V, Ctrl+Shift+V and Ctrl+Alt+V to Windows Terminal
  1.23.20211.0 with an image, a copied file and text on the clipboard in
  turn (the owner's clipboard saved and restored), and recorded the answer
  to the device attributes request; probe P32b drew Sixel images over a
  numbered grid, then text over them; Bubble Tea's and ultraviolet's code
  read for the order of raw output and frames and for a line's rewrite.
- Deterministic tests: `internal/app` (TestPasteImage, TestPasteCtrlV,
  TestPastePaths, TestPastedDraft, TestPreviewDraw, and
  TestWin32PasteRelease, which drives the real program with the
  sequences Windows Terminal sent in P32a), `internal/render` (TestSixel
  with the test's own Sixel decoder, TestPreviewStrip, TestRegion),
  `internal/input` (TestThumbnail); four golden frames added.
- Windows Terminal: the executable built from this tree
  (`scratchpad\term\acc-p32.exe`) in mock mode, driven by
  `scratchpad\term\p32demo.ps1` (plan 31's harness with clipboard, chord,
  drag and wait steps; run with `-STA`; the owner's clipboard saved before
  the steps and restored after), maximised at 208x59 and at 120x40; two
  prototype runs before the final ones.
- Build, vet, gofmt, the whole suite, the race detector on the changed
  packages, the timing step alone.

```text
Evaluation - Plan 32, image paste and preview - 10 criteria: 10 PASS, 0 FAIL
PASS 32.1 TestPasteImage: Alt+V with image data attaches "pasted-image-1.png · 2 KB ✕", the draft's text and cursor unchanged, nothing sent; the same bytes again: "pasted-image-1.png: already attached"; other bytes (the same drawing at another compression): "pasted-image-2.png"; a copied PNG and JPEG: both attached by name; a copied text file: refused with its reason; text alone: "no image on the clipboard", nothing changed; Enter: the session receives the text and 4 images, the first the pasted image's exact PNG bytes as image/png; the chips gone
PASS 32.2 TestPasteCtrlV: Ctrl+V's release without its press and no paste: one wait asked, the image attached when it ends; the release then the terminal's paste: the text inserted, nothing attached; the press then the release: the clipboard read once, its text pasted, no wait; a paste equal to the clipboard's files' paths (a PNG and a JPEG): attached, the draft unchanged; a GIF and a text file: the paths inserted as text, nothing attached; TestPastePaths: plain, quoted and line-separated paths, extra text, a share's path; TestWin32PasteRelease: the real program fed Windows Terminal's sequences (Ctrl pressed, V's release only, Ctrl released) attached "pasted-image-1.png" from the fake clipboard
PASS 32.3 TestPastedDraft: the stored draft lists "pasted:pasted-image-1.png" and the JPEG's path; no stored file holds the image's bytes, raw or in base64; a new model restores the text and the JPEG and toasts "draft restored; pasted-image-1.png was pasted, not kept"
PASS 32.4 TestSixel: the 140x140 and 100x100 thumbnails of the test drawing read back at their size from the raster attributes, every transparent pixel unset, every opaque pixel within 3 levels of its colour; a 64x64 gradient of 4096 colours in at most 255, within 24 levels; ThumbBorder the frame's #3B3F51. TestThumbnail: 140 and 100 pixels a side, the four corners transparent, the edges' middles the border's colour, the centre in the drawing's green square; a 1200x900 image, its centre over four times the side, sampled first: its red middle and its blue left kept
PASS 32.5 TestPreviewStrip: Sixel on, two attachments: at 200x60 the strip of 7 rows between the rule and the chips, thumbnails 0 and 1 of 14x7 cells each at its chip's first cell, their cells blank; at 120x40 5 rows of 10x5 cells; the conversation the strip's rows shorter; at 80x24 the frame the same as without Sixel; without Sixel no strip at any size; a WebP's box "╭────────────╮", "│    WebP    │", "╰────────────╯"; the "+" menu's bottom edge on the rule, above the strip
PASS 32.6 TestPreviewDraw: no drawing before the terminal's answer; the answer with 4: Sixel on, one drawing asked after the frame; it writes ESC[?2026h ESC 7, the CUP to the thumbnail's first cell, its Sixel (140x140), ESC 8 ESC[?2026l, and asks for the second, which draws again; a key typed (the draft's row): no drawing; the focus moved (B's frame on the strip's rows): drawn again; the image removed: its place written over with the frame's cells, no Sixel, then nothing more; an answer without 4: no strip. TestWin32PasteRelease: Windows Terminal's recorded answer (?61;4;6;7;14;21;22;23;24;28;32;42;52c) through the real decoder turned the preview on, and the thumbnail was drawn, 10x5 cells at 120x40
PASS 32.7 four golden frames added, attachments-preview at 80x24, 79x24, 120x40 and 200x60 (the first two the same as the attachments' frames: no strip under 30 rows); no other frame changed
PASS 32.8 go build ./..., go vet ./... and gofmt -l . clean; go test ./... 15 packages ok; the race detector on internal/input, imagetest, render and app: no race; the timing step alone 7 of 7: TestKeystrokeToFrame p95 1.00 ms (1.11 ms in the draft window), TestDraftKeystrokes 2.16 ms, TestKeystrokeCost 224 µs and 550 µs a key, TestLongSession 10.24 ms then 5.39 ms and 6.08 ms at the end, TestPasteToFrame, TestStreamToFrame 3.58 ms, TestThinkingFrame 3.54 ms, every target p95 ≤ 16 ms; the long session timed three times in turn with the previous commit's tree: its first tenth 1.68, 2.00, 2.43 ms (previous) and 1.61, 1.77, 2.49 ms (this), its end 5.1 to 5.5 ms and 5.1 to 6.1 ms: no cost of the change; go.mod and go.sum unchanged
PASS 32.9 Windows Terminal, 208x59 (paste-max): the generated PNG copied as a bitmap, Ctrl+V: the chip "pasted-image-1.png · 2 KB ✕" and its thumbnail drawn above it, the squares and "IMAGE UPLOAD" visible; the JPEG copied as a file, Alt+V: "acc-image-upload-test.jpg · 6 KB ✕" and its thumbnail; "Describe both images" typed; a drag from zone A's "Session (5hr)" across the strip to "6 KB" and released: the thumbnails whole, held and released; F1 opened and closed: both back, the mock's thinking card animating above them; the first chip's ✕: its thumbnail gone, the second one in its place, the vacated place clear; Enter: the message sent, its entry listing "acc-image-upload-test.jpg · 6 KB", no image left on the screen. 120x40 (paste-120x40): the same at 10x5 cells; Enter sent once the mock's own turn had ended (a first run pressed it during that turn: the draft waited, as designed, "turn running: wait, or interrupt"); the owner's clipboard saved and restored in each run (Bitmap, PNG and the clipboard history's two formats)
PASS 32.10 ui-design.md (Input editor: pasted images, the preview; Paste), architecture.md (Images: the paste, the preview), reference-design.md (the image preview), brief.md (amendment 32), doc/research/image-paste.md, AGENTS.md (status), plan.md (the plan, its corrections, its progress), references/10-chat-image-preview.png; every changed text file read byte by byte: no BOM, no CR; git status lists the intended files only; nothing committed
Not verified: a live turn with a pasted image (it goes out by plan 31's path, live on both CLIs; one can run on the owner's approval); Windows Terminal's paste of several copied files at once (P32a copied one; the comparison takes plain, quoted or line-separated paths); the owner's own keys and mouse (the harness sent them)
Verdict: DONE, waiting for the owner's review
```

## What the owner asked, answered

- **Paste an image "just like I did here".** Alt+V always does it, as
  Claude Code does on Windows. Ctrl+V does it too when the clipboard holds
  an image and no text: Windows Terminal then pastes nothing, and the
  dashboard reads the image itself on the key's release. A file copied in
  Explorer, which the terminal pastes as its path, is attached when it is
  an image. Pasting never sends.
- **"Is there a way to have a preview before sending?"** Yes: Windows
  Terminal draws Sixel images, so each attached image shows as a rounded
  square thumbnail above its chip, as in the owner's image, and goes away
  with its chip or the send.

## Limitations

- **Ctrl+V for an image** rests on Windows Terminal forwarding the V key's
  release when its own paste finds no text, which no document promises
  (P32a); a later terminal could stop. Alt+V does not depend on it. A
  clipboard holding both text and an image (cells copied from a
  spreadsheet) pastes the text on Ctrl+V; Alt+V attaches the image.
- **The preview's size** rests on Windows Terminal's 10x20 image pixels a
  cell (P32b); the dashboard supports Windows Terminal only. Without Sixel
  (Windows Terminal before 1.22, or another terminal), the chips alone
  show, as in plan 31.
- **A redraw** follows any frame that changes a strip row, 50 ms after it
  and again 250 ms later: during that time a thumbnail can show a gap, for
  example while a selection is dragged across it.
- **A pasted image** is kept in memory only: a restart restores the
  draft's text and files, not a pasted image, and says so.
- **A WebP** shows a box with its format's name: nothing here decodes
  WebP (plan 31).

## Screenshots

In `scratchpad\term\out-p32\paste-max\` and `paste-120x40\`: `pasted.png`
(the first chip and thumbnail), `both.png` (two), `selected-held.png` and
`selected.png` (the drag), `help.png` and `afterhelp.png` (F1 and back),
`afterremove.png` (one left), `end.png` (sent); `log.txt` beside them. The
first 120x40 run is in `paste-120x40-first\`, the prototype runs in
`proto-max\` and `proto2-max\`, the probes in `out-p32probe-keys-z0\`,
`out-p32probe-sixel-z0\` and `out-p32probe-da1-z0\`.

**Evidence outside the repository**, in the session's scratchpad:
`term\p32demo.ps1` (from `plan32\mkdemo.mjs`), `term\p32probe.ps1`,
`plan32\probe-tree\cmd\p32probe`, `term\acc-p32.exe`, the patch files
`plan32\p32-*.patch`.

## Follow-up (2026-10-23): the thumbnails one cell apart

The owner, on the screenshots, before approving: "there is too much space
between the two images, it should be way shorter" (brief amendment 32).
Each thumbnail stood above its own chip, so the wide chips set the space
between them. They now lie side by side, one cell apart, the first above
the first chip, as many as the width holds (plan.md, design D4 and
criterion 32.5 corrected). The block above stays as it was reported.

```text
Evaluation - Plan 32 follow-up, the thumbnails one cell apart - 3 checks: 3 PASS, 0 FAIL
PASS 32.5 TestPreviewStrip corrected: at 200x60 thumbnails 0 and 1 of 14x7 cells, the first at the first chip's first cell, the second one cell after it; at 120x40 the same at 10x5 cells; twelve attachments at both sizes: more thumbnails than chips shown (the others folded into "+N more"), each one cell after the one before, the last within the screen; the strip's other checks and the "+" menu on the rule as before
PASS checks: go build ./..., go vet ./... and gofmt -l . clean; go test ./... 15 packages ok; the golden frames unchanged, the strip's cells being blank wherever the thumbnails lie
PASS 32.9, the spacing in Windows Terminal (gap-max at 208x59, gap-120x40): the PNG pasted with Ctrl+V, the JPEG and the EXIF-rotated JPEG copied as files and attached with Alt+V: three thumbnails side by side, one cell apart, above the chips; at 120x40 the chips' row shows two and "+1 more", the third thumbnail drawn; in a first pair of runs (gap-*-dup) the GIF, the same drawing as the PNG, copied as a bitmap gave the same PNG bytes and was refused "pasted-image-1.png: already attached"; the owner's clipboard saved and restored in each run
Not verified: as in the block above
Verdict: DONE, waiting for the owner's review
```

Screenshots in `scratchpad\term\out-p32\gap-max\` and `gap-120x40\`
(`three.png`, `typed2.png`); the first pair in `gap-max-dup\` and
`gap-120x40-dup\`.
