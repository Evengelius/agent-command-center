# Plan 31 evaluation (2026-10-23)

The check of plan 31, image upload (brief amendment 31: "Implement genuine
image attachment support, validate it end to end, and report any
technical limitations accurately"). It covers the criteria written in
[`plan.md`, Plan 31](../plan.md) before the code, with the corrections
recorded there. Nothing is committed: the owner reviews the work, and the
live tests of criterion 31.8 wait for the owner's approval, since they use
quota.

How it was checked:
- The research first (doc/research/image-input.md): Claude Code's
  stream-json takes the Agent SDK's user message, whose content may hold
  base64 image blocks (code.claude.com, Streaming input); Codex's
  `turn/start` takes `image` and `localImage` inputs (the app-server v2
  schema probed in plan 1); both models take PNG, JPEG, WebP and GIF, with
  the limits the research quotes.
- Deterministic tests on images the tests draw themselves (a red, a green
  and a blue square and "ACC IMAGE UPLOAD TEST", 320x160): `internal/input`
  (TestImageInspect, TestImageFit, TestSniffRefusals, TestDraftImages,
  TestPickedPaths), `internal/input/imagetest` (TestDrawPixels),
  `internal/provider/claude` (TestClaudeImageMessage) and
  `internal/provider/codex` (TestCodexImageTurn), both through the real
  process plumbing to a fake CLI that logs its stdin, `internal/state`
  (TestSentImages), `internal/render` (TestAttachRow), `internal/app`
  (TestAttachments, TestAttachmentsDraft); eight golden frames added.
- Windows Terminal, the executable built from this tree
  (`scratchpad\term\acc-p31.exe`), the harness as
  `scratchpad\term\p31demo.ps1` with three new steps (`dialog`, `type`,
  `enter`), in mock mode with a temporary home and no agent CLI on the
  PATH, maximised at 9 pt (208x59): a click on the "+", a click on
  "Upload from computer", the real Open dialog, the generated images'
  folder typed, two of them chosen, a chip removed by its ✕, the message
  sent; the screenshots in `scratchpad\term\out-p31\quota-max\`.
- Build, vet, gofmt, the whole suite, the race detector on the changed
  packages, the timing step alone (`scratchpad\plan24\p31-checks.log`).

```text
Evaluation - Plan 31, image upload - 9 criteria: 8 PASS, 0 FAIL, 1 waiting for approval
PASS 31.1 TestImageInspect: the drawn image written as PNG, JPEG, GIF, BMP, TIFF, ICO, APNG, an EXIF-rotated JPEG (orientation 6), an animated GIF, a PNG named .jpg, a Unicode name and a name with spaces, and the checked-in WebP, all accepted at 320x160 with the media type of the matrix; each as sent decodes to the drawn squares (red, green, blue at their centres, JPEG within 24 to 32 levels); the conversions noted ("converted from BMP", "orientation applied", "animated: the first frame is sent"); the original files unchanged; refused with their reasons: an empty file, a text file, a truncated and a corrupt PNG, an SVG with a script, an HEIC, a TGA, a PSD, an LZW TIFF, an animated WebP, a 9000x9000 PNG header ("over 60 megapixels", never decoded), a missing file ("the file is gone"), a folder; TestSniffRefusals: AVIF, HEIF, both JPEG XL forms, both JPEG 2000 forms, DDS, Radiance HDR, OpenEXR, CR2, DNG, NEF, ARW, SVGZ, an SVG with a BOM, a broken BMP, a zip named .png, each refused with its reason; TestImageFit: over 100 px a side the PNG is resized to 100x50 and said so, its squares kept; over 10 patches of 32 px the JPEG is scaled to fit and stays JPEG; over 600 bytes the PNG is scaled until it fits; a WebP over the side is refused by name
PASS 31.2 the dialog behind Options.PickImages, which the app's tests replace by a fake; TestPickedPaths: one full path and a folder with two names read back, Unicode and spaces kept, the filter's pairs, OPENFILENAMEW 152 bytes on 64-bit Windows; the real dialog opened by the harness (31.7)
PASS 31.3 TestAttachments: two images chosen at once give two chips ("acc-image-upload-test.png · 1 KB ✕", "acc-image-upload-test.jpg · 6 KB ✕"), the draft's text and cursor unchanged, nothing sent; the PNG again is skipped ("already attached"), the SVG refused with its reason, a cancel changes nothing; a click on the first ✕ removes it, Ctrl+Shift+X the last; three images (a Unicode name, a name with spaces) sent with the text: the session receives the text and three image/png images holding the PNG file's exact bytes, the chips gone; the entry the adapter's line makes lists the three images by name; a file deleted between attaching and sending: nothing sent, the text and the chip back, the notice "the message could not be sent: gone.png: the file is gone"; TestAttachmentsDraft: the stored draft lists its two images, a new model restores the text and the image still there and names the one deleted; choosing never sends
PASS 31.4 TestClaudeImageMessage: the line on the fake CLI's stdin is a stream-json user message whose content is an image block (base64, image/png, the PNG's exact bytes) then the text; the mapped event carries image/png and the size, its raw line "<1177 bytes not kept>" instead of the data; the state's entry names the file; TestCodexImageTurn: the turn/start on the fake CLI's stdin has an `image` input whose data URL holds the PNG's exact bytes, then the text; the mapped event the same, its raw request without the data; each adapter's capabilities state its limits
PASS 31.5 eight golden frames added (attachments and attachments-sent at 79x24, 80x24, 120x40, 200x60), no other frame changed; TestZoneCDesign, TestEditorRow, TestPlusMenu, TestPlusMenuMouse and the select, pane, key and draft tests pass unchanged
PASS 31.6 go build ./..., go vet ./... and gofmt -l . clean; go test ./... 15 packages ok; the race run on internal/input, imagetest, state, render, app and the four provider packages ok, no race; the timing step alone 7 of 7: TestKeystrokeToFrame p95 1.38 ms (1.51 ms in the draft window), TestDraftKeystrokes 2.52 ms, TestKeystrokeCost 229 µs and 550 µs a key, TestLongSession 7.08 ms then 5.46 ms, TestPasteToFrame p50 1.03 ms, TestStreamToFrame 5.18 ms then 7.07 ms, TestThinkingFrame 4.59 ms, every target p95 ≤ 16 ms; timed in turn with the previous commit's tree, the long session and the streamed answer gave 7.35 and 5.48 ms (previous), 5.12 and 5.01 ms (this), 5.11 and 6.04 ms (previous), 5.51 and 4.72 ms (this): no cost of the change
PASS 31.7 Windows Terminal, quota-max: the click on "+" (15.5 s), on "Upload from computer" (16.3 s), the Open dialog "Upload from computer" up at 18.9 s in front of the terminal (its foreground window, owned by it, pid of the dashboard), its folder typed and opened, '"acc-image-upload-test.png" "acc-image-upload-test.jpg"' typed, Enter: the dialog closed and the two chips on their row (28.9 s); the first chip's ✕ clicked: one chip left (29.6 s); Enter: the message sent, its entry listing "🗀  acc-image-upload-test.jpg · 6 KB" (30.4 s); the "+", the menu, the pill, "Auto", zone A, C and the footer as approved
PASS 31.9 ui-design.md (Input editor: the menu's item, the attached images; Zone B: the images under a sent message), architecture.md (the provider interface's Send, the capabilities, a section on attached images), doc/research/image-input.md, reference-design.md unchanged (no new image), brief.md (amendment 31), AGENTS.md (status), plan.md (the plan, its corrections, its progress), testdata/images/README.md; every changed text file read byte by byte: no BOM, no CR; git status lists the intended files only; nothing committed
Not verified: 31.8, the live tests on Claude Code and Codex (the providers' own reading of the image), waiting for the owner's approval; the Open dialog with the owner's own mouse (the harness typed into it)
Verdict: NOT DONE until 31.8 has run: the image reaches each CLI's input as image input, proved at the wire, and no model has yet been seen reading it
```

## The format matrix

| Format | Extensions | How it is found | What is sent | Verified by |
| - | - | - | - | - |
| PNG | .png | its signature | as is, image/png | TestImageInspect, both adapters' wire tests, the harness |
| APNG | .apng | a PNG with an acTL chunk | as is, image/png, noted "animated: the first frame is what the model sees" | TestImageInspect |
| JPEG | .jpg .jpeg .jpe .jfif | its SOI marker | as is, image/jpeg; with an EXIF orientation other than 1, re-encoded upright at quality 95, noted | TestImageInspect, the harness |
| GIF | .gif | its signature | a still one as is, image/gif; an animated one's first frame as PNG, noted (OpenAI refuses animated GIFs, Claude reads the first frame) | TestImageInspect |
| WebP | .webp | RIFF and WEBP | as is, image/webp, its VP8, VP8L or VP8X header read for the size; an animated one refused (no frame can be taken without a decoder) | TestImageInspect (the checked-in lossless WebP), TestSniffRefusals |
| BMP | .bmp .dib | its signature | converted to PNG: 24 or 32 bits, BI_RGB or BI_BITFIELDS, 8 bits with a palette, top-down or bottom-up; RLE and other depths refused by name | TestImageInspect |
| TIFF | .tif .tiff | its signature | converted to PNG: baseline, uncompressed, 8 bits a sample, RGB, RGBA or grey, in strips; LZW, Deflate, JPEG, PackBits and the others refused by name | TestImageInspect |
| ICO | .ico | its signature | its largest image converted to PNG (a PNG inside or a bitmap with its mask) | TestImageInspect |
| HEIC, HEIF, AVIF | .heic .heif .avif | the ftyp brand | refused: "no decoder here; convert it to PNG or JPEG first" | TestImageInspect, TestSniffRefusals |
| JPEG XL, JPEG 2000 | .jxl .jp2 .j2k | their signatures | refused, the same reason | TestSniffRefusals |
| PSD, DDS, HDR, EXR | .psd .dds .hdr .exr | their signatures | refused, the same reason | TestImageInspect, TestSniffRefusals |
| TGA | .tga | its extension (TGA has no signature) | refused, the same reason | TestImageInspect |
| Camera raws | .dng .cr2 .nef .arw | the TIFF signature with CR2's mark or the extension | refused, the same reason | TestSniffRefusals |
| SVG | .svg .svgz | its XML or gzip start | refused: "SVG is not an image the providers take, and is never rendered or executed here" | TestImageInspect, TestSniffRefusals |
| Anything else | | | refused: "not an image" | TestImageInspect, TestSniffRefusals |

Every format is known by its bytes; the extension only names TGA and the
TIFF-based raws, which have no signature of their own. A file over 64 MB,
an image over 60 megapixels by its header, and a file gone are refused
before decoding. An image over its provider's side, patches or bytes is
scaled down to fit when it can be decoded, its note saying so; a WebP over
them is refused by name. The originals are never written; the bytes
sent live in the message only and no event, log or screen holds them.

## Claude Code

- **The interface**: `--input-format stream-json`, the user message's
  `content` as content blocks, each image a `{"type":"image","source":
  {"type":"base64","media_type":…,"data":…}}` block before the text block,
  as the Agent SDK's streaming-input page documents and the vision guide
  advises. No new flag, method or file: the session started as before.
- **Proved at the wire** (TestClaudeImageMessage): the line Claude Code's
  process reads on its stdin holds the image's exact bytes in base64 with
  its media type, then the text; the event the adapter keeps of it holds
  "<1177 bytes not kept>" in place of the data.
- **Limits applied**: 100 images, 7.4 MB an image as sent (the API's 10 MB
  once base64-encoded), 8000 px a side, 7.4 MB a message (the dashboard's
  own, so a written line stays under the 10 MB of piped input the
  headless page names).
- **Not yet seen**: the model reading the image (31.8).

## Codex

- **The interface**: `turn/start`'s `input`, each image an `image` input
  whose `url` is a data URL, before the `text` input (UserInput, the
  app-server v2 schema). `localImage`, which Codex reads from a path, is
  not used: what was checked is what is sent.
- **Proved at the wire** (TestCodexImageTurn): the `turn/start` Codex's
  process reads on its stdin holds the image's exact bytes in its data
  URL, then the text; the event the adapter keeps holds the media type
  and the size only.
- **Limits applied**: 100 images and 50 MB a message (the dashboard's own,
  under the API's 1,500 and 512 MB), 20 MB an image, 30,000 patches of 32
  px an image (rejected by the API beyond).
- **Not yet seen**: Codex accepting a data URL in `image` (the schema's
  string, the API's data URLs; no recording holds one), and the model
  reading the image (31.8). Should the live test show Codex refusing it,
  `localImage` with a converted copy in a temporary file is the fallback.

## Screenshots

In `scratchpad\term\out-p31\quota-max\`: `menu-click.png` (the menu over
the terminal), `dialog.png` and `dialog-dialog.png` (the Open dialog as it
opens, in front of the terminal), `files-typed-dialog.png` (the generated
images' folder, its thumbnails showing the three squares and the text, the
two names typed), `attached.png` (the two chips), `removed.png` (one chip
left after its ✕), `sent-later.png` (the sent message listing its image).
The dialog's first view shows the owner's own Desktop and the folders the
system lists beside it: the screenshots stay in the session's scratchpad.

## Limitations

- **Formats**: HEIC, HEIF, AVIF, JPEG XL, JPEG 2000, PSD, TGA, DDS, HDR,
  EXR and camera raws are refused: no decoder for them exists in the
  standard library or the approved dependencies. golang.org/x/image would
  add WebP decoding (so an animated WebP's first frame and a WebP's
  resizing), compressed TIFF, and others; it is a new direct dependency,
  for the owner to approve.
- **Animation**: the models read one frame; an animated GIF's first frame
  is sent as PNG and said so; an APNG is sent as a PNG, which every
  decoder reads as its first frame.
- **Claude Code's history**: each request resends the conversation, its
  earlier images included, under the API's 32 MB a request; a long
  session with several large images can reach it, which Claude Code would
  report as an error.
- **The dialog** opens over the window in front at the click and keeps it
  until it closes, as a modal dialog does.

## The live tests proposed (31.8), for the owner's approval

One turn on each CLI, in a temporary project outside the repository,
through the logging wrapper, with the configuration check before and
after, the dashboard driven by the harness in Windows Terminal: the
generated PNG (1,177 bytes, 320x160, 120 visual tokens on Claude) chosen
through the real Open dialog, sent with "What colours and what text are in
this image? Answer in one line."
- **Claude Code**: Haiku as the per-process override (the standing
  decision), one turn, no tool expected; expected usage a few thousand
  tokens.
- **Codex**: its default model, one turn, no tool expected; expected usage
  a few thousand tokens.
- **Passes when**: the answer names red, green and blue and "ACC IMAGE
  UPLOAD TEST"; the wrapper's log shows the image block or the image input
  as sent; the configuration check finds every configuration file
  unchanged.
- **Stops**: after the one turn, or at the first error, an approval
  request or two minutes without an answer; the wrapper's logs are deleted
  once the evidence is read, since they hold the image's bytes.

**Evidence outside the repository**, in the session's scratchpad:
`term\out-p31\`, `term\p31demo.ps1`, `term\acc-p31.exe`,
`plan24\p31-checks.log`, `plan31\images\` (the generated test files),
`plan24\harness31.cjs`, the patch files `p31-*.patch`.

## Follow-up (2026-10-23): the live tests

The owner approved the live tests ("run the live tests of criterion 31.8
as proposed: one turn on Claude Code (Haiku) and one on Codex, each in a
temporary project outside the repository, through the logging wrapper,
with the configuration check before and after; then report"). The block
below is the summary of that run; the one above stays as it was reported.

How it ran: the executable built from this tree, in a new maximised
Windows Terminal window for each CLI, driven by
`scratchpad\term\p31live.ps1`; the CLI through the test-only logging
wrapper, first on that window's PATH; Claude Code on Haiku through
`ANTHROPIC_MODEL=haiku`, set in that window only (the dashboard passes no
`--model`); a new empty project under `%LOCALAPPDATA%\Temp`; the
configuration check before and after. No Claude Code or Codex session ran
in a terminal. The wrapper's logs were read by
`scratchpad\live\analyze-p31.mjs`, which had passed its self-test on
synthetic logs (16 of 16), then deleted, since they held the image's
bytes; what the analysis kept holds none.

```text
Evaluation - Plan 31 follow-up, the live tests - criterion 31.8 on 2 CLIs: 2 PASS, 0 FAIL; one fix found by the run
PASS 31.8 Claude Code 2.1.303 on claude-haiku-5-5 (zone A "✿ Claude Haiku 5.5"), project acc-p31-claude-012406, 01:24: the PNG chosen through the real Open dialog, in front of the terminal and closed by Enter; its chip "acc-image-upload-test.png · 1 KB ✕"; the line Claude Code read on its stdin, 1,844 bytes: an image block (base64, image/png, 1,177 bytes, SHA-256 3d0cdc53882ee89b1a13e80766dc4a659eb8f4505aa1d61d2bbd7448c59213a5, the generated PNG's) then the text as typed; the turn ended in 13.6 s, no tool, no approval; the answer: Red, green and blue squares, left to right, with the text "ACC IMAGE UPLOAD TEST" below them. (then the Px reminder the owner's global CLAUDE.md asks of every session); zone A "✓ COMPLETED", zone B the message with its image row and the answer; Ctrl+Q 3.3 s, no process left, the project empty; configuration check PASS, .claude.json's own state keys only
PASS 31.8 Codex 0.163.0 on its default gpt-6-sol, project acc-p31-codex-012526, 01:25: the same steps; the turn/start Codex read, 1,892 bytes: an image input whose data URL holds the same 1,177 bytes (the same SHA-256), then the text; Codex took the data URL; the turn ended in 2.2 s, no tool, no approval; the answer: The image has red, green, and blue squares on a white background, with the text “ACC IMAGE UPLOAD TEST”.; zone A "✓ COMPLETED"; Ctrl+Q 2.7 s, no process left, the project empty; configuration check PASS, nothing changed
Found by the run and fixed: Codex wrote the image's base64 back on its stdout (Claude Code did not). The Codex mapper kept the lines it reads whole as the events' raw text, so the state's event log (the last 2,000 events, for the inspection view) held the image until it rotated out. Every event mapped from Codex's output now loses its image data URLs once mapped, the conversation unchanged (keepNoImages; withoutImageURLs now takes any string that is an image's data URL). TestCodexImageEcho: the item/started and item/completed userMessage lines, whose content is the UserInput sent, and a legacy user_message line keep "<1177 bytes not kept>" and none of the bytes; a data URL in the model's text reaches the conversation whole; without the fix the test fails on all three lines.
After the fix: go build and go vet clean, gofmt -l . empty; go test ./... 15 packages ok; the race run on internal/provider/codex ok; the timing step alone 7 of 7 (TestKeystrokeToFrame p95 1.01 ms, TestDraftKeystrokes 2.52 ms, TestLongSession 4.52 ms, TestStreamToFrame 3.88 ms, TestThinkingFrame 4.03 ms, every target p95 ≤ 16 ms).
Quota: Claude Code 56,620 input tokens written to the cache (its own prompt and tools), 2 uncached, 1,866 output of which 1,762 thinking, $0.01 API-equivalent; Codex 16,431 tokens, 16,402 input of which 8,064 cached, 29 output. The proposal's "a few thousand tokens" was low for Claude Code.
With the block above: 9 criteria, 9 PASS.
Not verified: the fix in a live turn (it changes what the dashboard keeps of Codex's lines, not what it sends or shows; a third turn was outside the approved scope); the Open dialog with the owner's own mouse.
Verdict: DONE: each CLI received the image as image input and its model read it
```

Screenshots, in `scratchpad\term\out-p31-live-claude\` and
`out-p31-live-codex\`: `01-ready.png` (the model in zone A),
`02-dialog.png` (the Open dialog, the PNG's path typed),
`03-attached.png` (the chip), `04-answer.png` (the message with its
image row, the answer, zone A's "✓ COMPLETED"); `results.txt`,
`analysis.json` and `confcheck.txt` beside them.

**Seen after the run, read only**: each CLI keeps the message, image
included, in its own session history, as it does for every message:
Claude Code in the project's transcript under `~/.claude/projects`, Codex
in the day's rollout under `~/.codex/sessions` (each file holds the PNG's
base64 twice). The dashboard keeps no copy, and it does not touch those
folders (AGENTS.md: the CLIs' configuration is read only); the two files
of this run hold the generated test image only, and stay until the owner
removes them.

## Follow-up (2026-10-23): the echo fix live

The owner asked for it ("run one more Codex image turn, same harness and
scope as 31.8, to check live that the dashboard's events no longer keep
the echoed image; then report"). The block below is that run's summary;
the blocks above stay as they were reported.

How it was checked: no view shows the orchestrator's raw events (the
inspection view is for zone C's agents), so the screen cannot show what
the dashboard's events keep. The turn ran as in 31.8, the dashboard
rebuilt from this tree after the fix (`acc-p31.exe`, its SHA-256 in
`results.txt`). Then, before the wrapper's log was deleted, every line of
the session, both ways and in the order they arrived, went through the
dashboard's own Codex mapper and state, compiled from this tree
(`scratchpad\plan31\echo-after`, the program in
`scratchpad\plan31\echocheck\`), each event stamped as the session's
queue stamps it; a walk over everything the mapper and the state then
held looked for the image's base64. The same replay through a copy whose
only difference is the fix's call removed (`echo-before`) is the control,
which must find the image. Both were first proved on a synthetic log
holding an echo.

```text
Evaluation - Plan 31 second follow-up, the echo fix live - 2 checks: 2 PASS, 0 FAIL
PASS 31.8 Codex 0.163.0 on gpt-6-sol, project acc-p31-codex-echo-014735, 01:47: the PNG through the real Open dialog, its chip, the message sent; the turn/start Codex read holds the image's exact 1,177 bytes (SHA-256 3d0cdc53882ee89b1a13e80766dc4a659eb8f4505aa1d61d2bbd7448c59213a5), then the text; the turn ended in 2.1 s, no tool, no approval; the answer: The image shows red, green, and blue squares on a white background, with the text “ACC IMAGE UPLOAD TEST”.; Ctrl+Q 2.8 s, no process left, the project empty; configuration check PASS, nothing changed
PASS echo Codex wrote the image back in 2 of the 55 lines the session logged both ways: the userMessage item's item/started and item/completed, at params.item.content[0].url, a data URL; replayed through this tree's mapper and state, 60 events: no string the state or the mapper keeps holds the image, and the 3 events of those 2 lines carry "<1177 bytes not kept>"; the control without the fix: 3 strings holding the image, each a raw line in the state's event log (state.Log[*].Raw)
Quota: Codex 16,435 tokens, 16,406 input of which 15,360 cached, 29 output.
Not verified: the running dashboard's own state, which no view shows; the replay ran the same code on the same lines.
Verdict: DONE: the dashboard keeps no echoed image; with the blocks above, 9 criteria, 9 PASS
```

Evidence in `scratchpad\term\out-p31-live-codex-echo\`: `results.txt`,
`echo-after.txt` and `.json`, `echo-before.txt` and `.json`,
`analysis.json`, `confcheck.txt` and the screenshots; none holds an image
byte (checked). Codex's history keeps this run's message too, image
included, in a third file under `~/.codex/sessions`
(`rollout-2026-10-23T01-47-43-….jsonl`), left as the first two.

**The commit and the push (2026-10-23), after the owner's approval.** The
owner approved plan 31 on 2026-10-23; commit 8e2a721 (69 files: 34
modified, 35 added, 8 of them golden frames) and the approval commit were
pushed to `origin`; `origin/main` equals `main`, the working tree is
clean, and the author is `Evengelius <molimoabel@gmail.com>`, with no
attribution line: PASS. The blocks above stay as they were reported.
