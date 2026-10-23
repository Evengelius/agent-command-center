# Current Plan

The active plan with its progress, the owner's standing decisions, and one
entry per closed plan. Each plan comes from `doc/brief.md` and its
amendments, ends with the evaluation summary of `doc/evaluation.md`, and
waits for the owner's approval before the next one starts. This file keeps
the current state only (the owner, 2026-10-18): a closed plan's full text,
criteria and progress log are in git (`git show 8888f59:plan.md` for plans 1
to 13, `git show 84d2ad8:plan.md` for plan 14, `git show b047434:plan.md`
for plan 15, `git show becfb34:plan.md` for plan 16, `git show f00c2d8:plan.md`
for plan 17, `git show 8b3b16b:plan.md` for plan 18, `git show
d84e5d4:plan.md` for plan 20, `git show f939d96:plan.md` for plan 21,
`git show d13ef71:plan.md` for plan 22, `git show e8d3e3b:plan.md` for
plan 23, `git show a3bfe5d:plan.md` for plan 24, `git show cb8f76d:plan.md`
for plan 25, `git show a9a5145:plan.md` for plan 26, `git show 041e7b8:plan.md`
for plan 27, `git show 8fe0077:plan.md` for plan 28, `git show d696f22:plan.md`
for plan 29, `git show 6025513:plan.md` for plan 30), and its results in
its evaluation file.

## The owner's standing decisions

Decisions of closed plans that still apply; `AGENTS.md` and
`doc/evaluation.md` hold the others.

- **Live runs.** Each one is proposed first with its scope, model, expected
  usage and stopping conditions, and runs only once approved (plan 3,
  2026-10-16); Claude runs use Haiku as a per-process override.
- **The configuration check is hash-only**: credential contents are never
  read, and a file that cannot be classified safely is "changed, not
  classified", never reported unchanged (2026-10-16). No parser runs over a
  file that may hold credentials (the mixed-state tokenizer was withdrawn,
  2026-10-16); the one exception is confcheck's key-level comparison of
  `~/.claude.json`, sensitive keys skipped (plan 5).
- **Measured, not assumed.** Performance figures are targets until a test
  measures them; a documented design is never presented as verified runtime
  behaviour (2026-10-16).
- **The integration.** If driving a CLI without a terminal falls short, the
  limit is documented and the smallest adjustment proposed; the architecture
  never changes silently (2026-10-16).
- **Drafts and session data** survive cancellation, crashes and recovery; no
  conversation data is stored beyond the draft, and nothing sensitive goes
  into a log (2026-10-16).
- **Zone A shows values, not notes** (amendment 25, 2026-10-22, plan 26):
  its telemetry pane carries no notes row, legend or counter, now or in
  any later plan; errors, retries, unrecognised lines, freshness and the
  legends live in the details view (Enter on TELEMETRY), where each value
  has its source and age. An undocumented value keeps its "†" mark where
  it is shown.

## Recurring

Tasks tagged **[Recurring]** repeat on an outside event: they are never
closed, never listed as remaining, and each occurrence is reported in the
summary format (the owner, 2026-10-18).

- **[Recurring] After a CLI update.** At session start, `claude --version`
  and `codex --version` are compared with the "Last probed" line of
  `doc/research/probes/README.md`; a newer one is told to the owner with the
  re-check proposed (`doc/evaluation.md`, After a CLI update), which runs
  only once the owner approves it. The dashboard's project bar shows
  "newer than tested" meanwhile. Last occurrence: Claude Code 2.1.303 on
  2026-10-19, approved, every check PASS (P303a to P303d in
  `doc/research/probes/README.md`); it found that plan 14's shape check
  counted an interrupted tool use as unrecognised, fixed the same day at the
  owner's request (P303e).

## Nice to have

Items tagged **[Nice to have]** are future improvements: none is built, and
none is listed as remaining, until the owner asks for one in a new plan
(the owner, 2026-10-18). Amendment 4 names ten "future improvements"
(`doc/brief.md`, Amendment 4, "For this phase, do NOT implement"). Five
remain below. Recent projects, favourites and a project browser were plan 15
(amendment 9); git worktree management was plan 16 (amendment 10). The
automatic VS Code integration was replaced by the owner with a file viewer in
the terminal, plan 17 (amendment 11).

- **[Nice to have]** Advanced model configuration.
- **[Nice to have]** Cross-provider agent collaboration.
- **[Nice to have]** Provider switching during an active session.
- **[Nice to have]** Complex session profiles.
- **[Nice to have]** Advanced orchestration configuration.
- **[Nice to have]** Google Antigravity as an agent CLI, removed from the
  MVP by amendment 5; the provider interface stays open to it.
- **[Nice to have]** Starting it like an app (the owner, 2026-10-19): a built
  `agent-command-center.exe` started from a Start-menu shortcut, pinnable to
  the taskbar, whose target is `wt.exe -d <projects folder> <path to the
  .exe>`, so that it always opens in Windows Terminal (the only supported
  terminal, amendments 6 and 7) with the project field at that folder; the
  `.exe` is rebuilt after each code change.

## Plan 31 - Image upload

Brief amendment 31 (2026-10-23): the "+" menu's "Upload from computer"
works: the native Windows file dialog, the images validated by their
content, attached to the draft as compact removable attachments, and sent
as real image input to Claude Code and Codex when the owner submits;
nothing is committed until the owner has reviewed the implementation and
the results. The criteria below were written before the code.

### Research (D0, [doc/research/image-input.md](doc/research/image-input.md))

- Claude Code: `--input-format stream-json` takes the Agent SDK's user
  message, whose `message.content` is a string or the Messages API's
  content blocks; the streaming-input page shows an image block, `{"type":
  "image", "source": {"type": "base64", "media_type": "image/png", "data":
  …}}`, beside a text block, and says an image block without a source is
  replaced by a text note (code.claude.com, Agent SDK, Streaming input,
  read 2026-10-23). The Claude API takes image/jpeg, image/png, image/gif
  and image/webp, 10 MB base64-encoded an image, up to 100 images a request
  on 200k-context models, 8000x8000 px, a 32 MB request; animations are
  unsupported, the first frame is used; images are best placed before the
  text (platform.claude.com, Vision, read 2026-10-23).
- Codex: `turn/start`'s `input` is a list of `UserInput`: `text`, `image`
  (`url`, a data URL or a web URL, with an optional `detail`) or
  `localImage` (`path`) (the app-server v2 schema probed in plan 1,
  doc/research/probes/p0-codex-app-server-schema-v2.json). The OpenAI API
  takes PNG, JPEG, WebP and non-animated GIF, up to 1,500 images and 512
  MB a request, each image at most 30,000 patches of 32 px, rejected
  rather than resized (developers.openai.com, Images and vision, read
  2026-10-23).
- Both take the image as content, never as a path in the text. Neither
  takes BMP, TIFF, ICO, HEIC, AVIF, JPEG XL, SVG, PSD, TGA, DDS, HDR, EXR,
  JPEG 2000 or camera RAW: those need a conversion to PNG here, which
  needs a decoder the dashboard has. The dependencies stay the approved
  ones: the decoders of D1 are Go's standard library and the dashboard's
  own; golang.org/x/image (WebP, TIFF with compression) would widen the
  matrix and needs the owner's approval.

### Design

- **D1 The format matrix** (`internal/input/image.go`): by the file's
  bytes, never its extension; a file read in a command, never in the
  update loop; the original never written. The matrix below says what is
  found, validated and sent; a refusal names the format and the reason.
  The limits are the provider's (`provider.Capabilities.Images`): Claude
  Code 100 images a message, 7.5 MB a file as sent (10 MB base64), 8000
  px a side; Codex 100 images a message (the dashboard's own cap), 20 MB
  a file, 30,000 patches of 32 px an image; an image over a limit is
  refused when attached, naming the limit; an image over 100 megapixels
  by its header is refused before decoding.
- **D2 The dialog** (`internal/input/dialog_windows.go`): the Common Item
  Dialog (IFileOpenDialog through COM, with x/sys/windows and syscall, no
  new module): multiple selection, existing file-system files only, the
  filter "Images" with every extension of D1 and "All files", the title
  "Upload from computer", owned by the foreground window at the click so
  it is modal to the terminal; run in a command on its own OS thread with
  an apartment-threaded COM; cancel gives no files and no message. Other
  platforms: "no file dialog on this platform". The app takes the dialog
  through its options, so tests give it a fake.
- **D3 Attaching** (app): a click on "Upload from computer" opens the
  dialog; each chosen file is inspected (D1) in the same command; the
  result adds the accepted images to the draft's attachments (path, name,
  size, media type, width, height, note), refuses the others with a toast
  naming the file and the reason, skips a file already attached ("already
  attached"), and never touches the text or the cursor. A chosen image is
  never sent by itself: only the send key sends.
- **D4 The attachments row** (render): while the draft has attachments,
  one row between the editor's rule and the prompt row, the chips on the
  tool rows' ground: "🗀 name.png · 12 KB ✕", the note after the size when
  there is one, two cells between chips, the chips past the width folded
  into "+N"; the "✕" is panel 'X' with the chip's index and a click on it
  removes the attachment; Ctrl+Shift+X removes the last one; the editor is
  one row taller with attachments, within its half of B. The sent
  message's entry in zone B lists its images under the text, "🗀 name.png
  · 12 KB", one a row.
- **D5 Sending**: the session's Send takes a `provider.Message`, the text
  and the images (name, media type, bytes, width, height); the app reads
  and converts each attachment in the write command and refuses the send
  with "not sent: <name>: <reason>", the draft and its attachments kept,
  when a file is gone or over a limit; Claude Code's adapter writes the
  user message with the content blocks, the images first then the text;
  Codex's adapter starts the turn with `image` inputs holding data URLs,
  then the text; the mock records the images' names and media types and
  echoes them in its user message. A failed write restores the text and
  the attachments; an accepted one clears both. The bytes live in the
  message only; nothing of them is logged.
- **D6 Drafts**: the stored draft gains a sidecar, `<draft>.attachments`
  (JSON, the paths); a restored draft re-inspects its paths, keeps the
  images still valid and says which are gone.
- **D7 Test images** (`internal/input/imagetest`): drawn at run time in
  the test's temporary directory, 320x160: a red, a green and a blue
  square and the text "ACC IMAGE UPLOAD TEST" in a 5x7 pixel font; written
  as PNG, JPEG, GIF, BMP, TIFF and ICO by the standard library and the
  dashboard's own writers; the WebP generated once by a browser canvas
  (its script kept with the fixture) and checked in as
  testdata/images/acc-image-upload-test.webp, since Go's standard library
  has no WebP encoder.
- **D8 Docs**: ui-design.md (Input editor: the attachments row, the key;
  Zone B: the entry's images), architecture.md (the input part, the
  provider interface), AGENTS.md (status), brief.md (amendment 31),
  doc/research/image-input.md, `doc/plan31-evaluation.md`.

The format matrix (D1):

| Format | Found by | Validated by | Sent as | Note |
| - | - | - | - | - |
| PNG (.png, .apng) | the signature | image/png, a full decode | image/png as is | an APNG (acTL chunk): "animated: the first frame is what the model sees" |
| JPEG (.jpg, .jpeg, .jpe, .jfif) | the SOI marker | image/jpeg, a full decode | image/jpeg as is; re-encoded at quality 95 with the rotation or flip applied when its EXIF orientation is not 1, "orientation applied" | the providers read no metadata |
| GIF (.gif) | the signature | image/gif, every frame | image/gif as is with one frame; the first frame as PNG when animated, "animated: the first frame is sent" | OpenAI refuses animated GIFs |
| WebP (.webp) | the RIFF/WEBP signature | the VP8, VP8L or VP8X header: its dimensions and flags | image/webp as is | an animated WebP is refused: no frame can be taken without a decoder |
| BMP (.bmp, .dib) | the signature | the dashboard's decoder: BITMAPINFOHEADER, 24 or 32 bits BI_RGB or BI_BITFIELDS, 8 bits with a palette, top-down or bottom-up | PNG | other depths and compressions refused by name |
| TIFF (.tif, .tiff) | the signature | the dashboard's decoder: baseline uncompressed 8-bit RGB, RGBA or grayscale in strips | PNG | compressed or other TIFFs refused by name |
| ICO (.ico) | the signature | its largest image: PNG by image/png, DIB by the BMP decoder | PNG | |
| HEIC, HEIF, AVIF (ftyp brands), JPEG XL, JPEG 2000, PSD, DDS, HDR, EXR, camera RAW (DNG, CR2, NEF, ARW), TGA (by its extension: it has no signature) | their signatures | refused: "<format>: no decoder here; convert it to PNG or JPEG first" | | not supported, never claimed |
| SVG, SVGZ | the XML or gzip start | refused: "SVG is not an image the providers take, and is never rendered or executed here" | | |
| anything else | | refused: "not an image" | | |

### Criteria

- 31.1 TestImageInspect (`internal/input`): the generated images (D7) in
  every written format and the checked-in WebP are accepted with the
  right media type at 320x160; the conversions (BMP, TIFF and ICO to PNG,
  an animated GIF to its first frame, a JPEG with EXIF orientation 6
  rotated back) decode to the drawn pixels (the red square at the top
  left); a PNG renamed .jpg is accepted as PNG; an empty file, a text
  file, a truncated PNG, a PNG with a corrupt chunk, an SVG, an HEIC
  brand, a TGA, a PSD, a compressed TIFF and an animated WebP are refused
  with the designed reasons; a 9000x9000 PNG header is refused for its
  size before decoding; a Unicode name and a path with spaces are
  accepted; the capabilities' limits refuse a file over them by name.
- 31.2 The dialog's COM calls are behind an interface the app's tests
  fake; on Windows the real dialog is exercised by the harness (31.7).
- 31.3 TestAttachments (`internal/app`): with a fake picker, choosing two
  images shows two chips with their names and sizes and leaves the text
  and the cursor as they were; choosing one of them again is skipped
  with a toast; a refused file toasts its reason; cancel changes nothing;
  a click on a chip's "✕" removes it; Ctrl+Shift+X removes the last; the
  send key sends the text and the images, the fake session receiving the
  generated PNG's bytes, and the chips go; a failed write restores the
  text and the chips; a missing file at send time keeps the draft with
  "not sent: …"; the stored draft's sidecar holds the paths, a restart
  restores the chips and drops a deleted file with a toast; choosing an
  image never submits.
- 31.4 TestClaudeImageMessage and TestCodexImageTurn: the line written to
  Claude Code holds the content blocks, the images first with their media
  types and base64 data, then the text; the `turn/start` sent to Codex
  holds `image` inputs with data URLs, then the text; each provider's
  capabilities state its limits.
- 31.5 Golden frames: the editor with two attachments at 80x24, 120x40
  and 200x60 and the sent message's entry; every other frame unchanged;
  TestZoneCDesign, TestEditorRow, TestPlusMenu and the select, pane, key
  and draft tests pass unchanged.
- 31.6 `go build ./...`, `go vet ./...`, `gofmt -l .` empty, `go test
  ./...` all ok, the race run on `internal/input`, `internal/render`,
  `internal/app` and `internal/provider/...` clean, the timing step alone
  7 of 7.
- 31.7 Windows Terminal at 208x59 on the quota mock: the dialog open over
  the terminal (a screenshot), the generated PNG chosen through it (its
  path typed into the dialog's name field), the chip on the attachments
  row, the chip removed with a click on its "✕"; nothing else of the
  approved design changed.
- 31.8 Live, once the owner approves: Claude Code and Codex each sent the
  generated PNG with "What colours and what text are in this image?
  Answer in one line." in a temporary project; the answer names red,
  green and blue and "ACC IMAGE UPLOAD TEST"; the wrapper's log shows the
  image block or the image input as sent; the configuration check before
  and after.
- 31.9 The documents of D8; every changed text file UTF-8 without BOM,
  LF; `git status` lists the intended files only; nothing committed.

### Progress

Corrections made as the work went, each with its reason:
- D1: an image over a side, its patches or its bytes is scaled down to the
  provider's limits by area averaging when it can be decoded, and said so
  ("resized from 9000x6000 to 8000x5333 for the provider's limits"), not
  refused: a 4K screenshot over Claude Code's 7.4 MB is common; a WebP,
  which nothing here decodes, stays refused. The decode guard is 60
  megapixels, not 100, for memory: a 50 MP camera's image passes. The
  limits gained a message total (Claude Code 7.4 MB, Codex 50 MB).
- D2: GetOpenFileNameW with the Explorer flag and no hook, not
  IFileOpenDialog through COM's vtables: Windows shows it as the same
  common item dialog, with far less code to get wrong.
- D4: the chips carry no icon, so two fit on a 120-column row; the sent
  message's rows in zone B have it.
- 31.1: "the capabilities' limits refuse a file over them by name" reads
  "scale a decodable file down to them, said so, and refuse a WebP over
  them by name" (D1's correction).
- 31.8's run: Codex writes the owner's message back with its images, and
  the mapper kept the lines it reads whole in the events, so the event log
  held the image's bytes; every event mapped from Codex's output now loses
  its image data URLs (keepNoImages, TestCodexImageEcho).

- [x] D0 the research written (doc/research/image-input.md).
- [x] D1 the image package (`internal/input/image*.go`) and its tests
  (TestImageInspect, TestImageFit, TestSniffRefusals, TestDraftImages);
  D7 the test images (`internal/input/imagetest`, TestDrawPixels;
  testdata/images/acc-image-upload-test.webp from a browser canvas,
  checked against the drawing's pixel hash).
- [x] D2 the dialog (`internal/input/dialog*.go`, TestPickedPaths).
- [x] D5 the message, the adapters and their tests (TestClaudeImageMessage
  and TestCodexImageTurn through the real process plumbing; TestSentImages).
- [x] D3, D4 and D6: the app and the render (TestAttachments,
  TestAttachmentsDraft, TestAttachRow), eight golden frames added.
- [x] The checks (31.6): build, vet, gofmt, the suite, the race run on the
  changed packages, the timing step alone 7 of 7 (the streamed answer's
  p95 moving between 4 and 7 ms, the previous commit's tree timed in turn
  alike); the screenshots (31.7, `scratchpad\term\out-p31\`).
- [x] D8 (ui-design.md, architecture.md, brief.md's amendment 31,
  AGENTS.md) and the evaluation (`doc/plan31-evaluation.md`).
- [x] The live tests (31.8), approved by the owner and run on 2026-10-23:
  Claude Code on Haiku and Codex each read the image (2 PASS,
  `doc/plan31-evaluation.md`, Follow-up); the fix the run found, tested;
  the checks again, the timing step 7 of 7.
- [x] The echo fix checked live at the owner's request (2026-10-23): a
  third Codex turn, its real lines replayed through this tree's mapper and
  state: nothing kept holds the image, while the control without the fix
  keeps it three times (`doc/plan31-evaluation.md`, second follow-up).
- [ ] The owner's review; then the commit.

## Closed plans

One entry each: what the plan did, its approval, its evaluation file and
its commits. Their criteria and progress logs: `git show 8888f59:plan.md`
(plans 1 to 13), `git show 84d2ad8:plan.md` (plan 14), `git show
b047434:plan.md` (plan 15), `git show becfb34:plan.md` (plan 16), `git show
f00c2d8:plan.md` (plan 17), `git show 8b3b16b:plan.md` (plan 18), `git show
d84e5d4:plan.md` (plan 20), `git show f939d96:plan.md` (plan 21), `git show
d13ef71:plan.md` (plan 22), `git show e8d3e3b:plan.md` (plan 23), `git show
a3bfe5d:plan.md` (plan 24), `git show cb8f76d:plan.md` (plan 25), `git show
a9a5145:plan.md` (plan 26).

## Plan 1 - Research

The agent CLIs and how to drive them, telemetry, terminal input, the
reference projects, and the decision to build rather than fork
(`doc/research/`). Accepted on 2026-10-16, 9 PASS and criteria 8 and 9 kept as
FAIL, documented exceptions
([`doc/research/plan1-evaluation.md`](doc/research/plan1-evaluation.md));
commit 7e4d67c.

## Plan 2 - Architecture and reliability

`doc/architecture.md`, `doc/events-and-state.md`, `doc/ui-design.md` and
`doc/reliability.md`. Approved on 2026-10-16, 15 of 15 PASS
([`doc/plan2-evaluation.md`](doc/plan2-evaluation.md)); commit d5b910e.

## Plan 3 - Proposal

The MVP roadmap and the decisions it asked for (stack, dependencies, live
runs, configuration check). Approved on 2026-10-16, 12 of 12 PASS; commit
d5b910e, whose `doc/proposal.md` holds the roadmap. The design documents
and the code have replaced it since.

## Plan 4 - MVP (from amendment 4; steps detailed by plan 3)

The MVP, steps 0 to 8: startup, Claude Code, the streaming conversation,
approvals and resumption, Codex, agent activity, telemetry, the UI.
Approved on 2026-10-17 after a live demo on both CLIs, 14 of 14 PASS
([`doc/plan4-evaluation.md`](doc/plan4-evaluation.md)); commits ded53d8 to
cf58739.

## Plan 5 - Post-MVP improvements

The list plan 4 left open, Windows only. Approved on 2026-10-17, 17 of 17
PASS, then 15 of 15 live and 6 of 6 for the follow-up
([`doc/plan5-evaluation.md`](doc/plan5-evaluation.md)); commits d42531f,
897f640, 6813c0e.

## Plan 6 - Plan 5's known limits

Four known limits fixed, then items 5 and 6 added by the owner. Approved on
2026-10-17, 6 of 6 and 8 of 8 PASS
([`doc/plan6-evaluation.md`](doc/plan6-evaluation.md)); commits df273fe,
0814d67.

## Plan 7 - The open list after plan 6

The open list cleaned. Approved on 2026-10-17, 7 of 7 PASS
([`doc/plan7-evaluation.md`](doc/plan7-evaluation.md)); commit bffe9a5.

## Plan 8 - Codex's allow for the session

`acceptForSession` on Codex 0.163.0. Approved on 2026-10-17, 6 of 6 PASS
and its follow-up 3 of 3 ([`doc/plan8-evaluation.md`](doc/plan8-evaluation.md));
commits e75d591, 1daeec1.

## Plan 9 - Plan 8's items left not verified

Two items verified, the third recorded as not verifiable. Approved on
2026-10-18, 6 of 6 PASS
([`doc/plan9-evaluation.md`](doc/plan9-evaluation.md)); commits
c9576d9, 8dea129.

## Plan 10 - A spawned Codex agent's label and the timing step

Both items verified. Approved on 2026-10-18, 7 of 7 PASS
([`doc/plan10-evaluation.md`](doc/plan10-evaluation.md)); commits b71c114,
621609e.

## Plan 11 - A live approval offering the allow for the session

The live approval closed as not verifiable, by the owner's choice.
Approved on 2026-10-18, 3 of 3 PASS
([`doc/plan11-evaluation.md`](doc/plan11-evaluation.md)); commits 81f3bf8,
cb206b6.

## Plan 12 - Manual or Auto approvals

Brief amendment 8: a per-session switch between Manual and each CLI's own
Auto. Approved on 2026-10-18, 18 of 19 PASS and 1 FAIL accepted
([`doc/plan12-evaluation.md`](doc/plan12-evaluation.md)); commits cb31557,
83fa023.

## Plan 13 - A reviewer's refusal live

Claude Code's classifier denial seen live and a render defect it exposed
fixed; Codex's reviewer not reached, closed by the owner. Approved on
2026-10-18, 14 PASS and 3 not observed
([`doc/plan13-evaluation.md`](doc/plan13-evaluation.md));
commits d68bb55, 8888f59.

## Plan 14 - Nothing left open

Every open question of the research files settled or closed by the owner,
Claude Code's question card built and seen live, the documents reduced to
the latest state and a CLI update made visible. Approved on 2026-10-18, 18
PASS and 15 PASS ([`doc/plan14-evaluation.md`](doc/plan14-evaluation.md));
commits 0206ac1, ce06b97.

## Plan 15 - Recent projects, favourites and a project browser

Brief amendment 9: the startup screen's project list (`projects.json`,
favourites first) and its Ctrl+O browser. Approved on 2026-10-18, 21 PASS
([`doc/plan15-evaluation.md`](doc/plan15-evaluation.md)); commits bd9ffa3,
b047434.

## Plan 16 - Git worktree management

Brief amendment 10: the Worktrees view (Ctrl+W, at startup and in session)
with create, list, diff, merge and remove, git through `os/exec`. Approved
on 2026-10-18, 26 PASS
([`doc/plan16-evaluation.md`](doc/plan16-evaluation.md)); commits ce95ba2,
fad6d20, 642c5b9, becfb34.

## Plan 17 - A file viewer in the terminal

Brief amendments 11, 12 and 13: a file viewer in the terminal with Chroma's
highlighting, an Explorer of the project (F3) and the draft in a window
(Ctrl+G), with no external editor. Approved on 2026-10-19, 91 PASS
([`doc/plan17-evaluation.md`](doc/plan17-evaluation.md)); commits 1748808,
10c3e74, 904f9a6, 749cef4.

## Plan 18 - An adaptive layout

Brief amendment 14: zone C sized by the agents in play, a strip of 4 rows (3
under 100 columns) when there are none, and cleared at each new prompt.
Approved on 2026-10-19, 22 PASS
([`doc/plan18-evaluation.md`](doc/plan18-evaluation.md)); commit 8b3b16b.

## Plan 19 - Zone B as the centerpiece

Brief amendment 15: one speaker label per exchange, RESULT over an
exchange's outcome, compact tool rows and folds of long runs of calls, each
entry's turn (`Entry.Stage`). Done on 2026-10-20, 19 PASS
([`doc/plan19-evaluation.md`](doc/plan19-evaluation.md)). The owner
reviewed it in a demo the same day and asked for plan 20 (amendment 16),
which keeps its logic and replaces its look; not committed on its own, it
was committed with plan 20 once the owner approved that, on 2026-10-20;
commit d84e5d4.

## Plan 20 - Strict conformance to the approved templates

Brief amendments 16 and 17: the owner's two templates as one design
system, A and B in one upper row and C full width under them, zone B's
messages as blocks with a rule between them, the tool rows, the folds, the
pending approval's pink box and RESULT. Approved on 2026-10-20 after the
owner's live review, 32 of 32 PASS
([`doc/plan20-evaluation.md`](doc/plan20-evaluation.md)); commit d84e5d4,
with plan 19.

## Plan 21 - What the review of d84e5d4 found

A read-only review of commit d84e5d4: zone D's session time past 2,000
events, two timed tests outside the timing step, Claude Code's
undocumented `api_message_id` labelled with a fallback, zone C's floor at
160x24 to 160x26, and documents promising what the code did not do (the
send and newline keys built in `config.json`, the others corrected).
Approved on 2026-10-20 with the recommended choices, 38 PASS and 2 FAIL,
then the long session's frame time fixed at the owner's request, 6 PASS
([`doc/plan21-evaluation.md`](doc/plan21-evaluation.md)); commits bdc4250,
f939d96.

## Plan 22 - A live thinking indicator

Brief amendments 18 to 20: zone B's live thinking card for Claude Code and
Codex, with its animations, a summary row for each thinking phase that
opens on its details, and an activity line while a turn works; corrections
C1 to C12 from a read-only investigation
([`doc/plan22-investigation.md`](doc/plan22-investigation.md)); Codex
started through the local proxy that `config.json` names (amendment 19);
Codex's reasoning summaries asked for and Claude Code's hook events known
(amendment 20), checked live with one Codex turn (L20). Approved on
2026-10-21: 31 of 31 PASS, then the corrections 17 of 18 with 5.14 fixed,
amendment 20 13 of 13, and L20 8 of 8 once two criteria were corrected
([`doc/plan22-evaluation.md`](doc/plan22-evaluation.md)); commits
e6895c6, 6306bfb and d13ef71, then the approval commit.

## Plan 23 - The acc launcher in the project

Brief amendment 21: `acc`, typed in Windows Terminal, opens the dashboard
in a tab of its own at the owner's size, a 9 pt font. `acc.cmd` at the root
runs `scripts\acc.ps1`, which writes a Terminal fragment for the profile
and makes Terminal reload. The day's machine-local changes were undone.
Approved on 2026-10-22, 6 of 6 PASS before the commit, then committed and
pushed ([`doc/plan23-evaluation.md`](doc/plan23-evaluation.md)); commit
e8d3e3b, then the approval commit.

## Plan 24 - Zone A with the telemetry, zone D removed

Brief amendment 22: zone A redrawn after the owner's image, holding the
orchestrator's dynamic status, the stages as blocks and zone D's telemetry
(session usage, quota windows with lavender bars and countdowns, the notes,
the details view); zone D removed; Fable and Mythos as model families.
Amendment 23, after the owner's live session: the Claude adapter reads the
`/usage` report after the handshake and after each turn, so the
model-scoped "Fable limit" shows live, the exchange kept out of the
conversation. Approved on 2026-10-22: 35 of 35 PASS with the conformance
table 22 rows PASS, amendment 23 6 of 6 and its live check 8 of 8
([`doc/plan24-evaluation.md`](doc/plan24-evaluation.md)); commits 6209267
and a3bfe5d, then the approval commit.

## Plan 25 - The unrecognised lines named

Brief amendment 24: the owner's scenario 1 run on Fable 5.1 ended with
"unrecognised 2" and no way to see which lines. The details view now lists
the last 20 unrecognised lines with their time, type, reason and text; a
replay tool names them from a logging wrapper's record; the owner's two,
named by a logged rerun, were `system/task_summary`, known with no event
since. Approved on 2026-10-22: 7 of 7 PASS, two live runs
([`doc/plan25-evaluation.md`](doc/plan25-evaluation.md)); commit cb8f76d,
then the approval commit.

## Plan 26 - Zone A without notes

Brief amendment 25: the owner asked to get rid of zone A's notes
("unrecognised 1", "quota updated", "† undocumented field", "cost at API
rates") completely and forever. The telemetry pane now ends with the last
quota window's rows; the counts, ages and legends live in the details view,
where an undocumented value keeps its "†" and the heading explains it; a
standing decision. Approved on 2026-10-22: 6 of 6 PASS, 51 golden frames
reviewed ([`doc/plan26-evaluation.md`](doc/plan26-evaluation.md)); commit
a9a5145, then the approval commit.

## Plan 27 - Zone C with the execution timeline

Brief amendments 26 and 27: the owner's design of zone C and its
close-ups, to respect strictly. Zone C from 100 columns: the header
inside the frame, a rule and the panels' line in the title colour, the
agents' panel as wide as its two-column cards (the label over the clock
and the tokens, a grey bar, the model over the current tool; the selected
agent's rows on a band) and the selected agent's execution timeline with
its status at the heading's right end; the narrow layout keeps plan 18's
row. Approved on 2026-10-22: 15 of 16 PASS, 27.6 corrected with a note
at the owner's decision, 46 golden frames, Windows Terminal screenshots
at 208x59 and 120x40 against the images
([`doc/plan27-evaluation.md`](doc/plan27-evaluation.md)); commit 041e7b8,
then the approval commit.

## Plan 28 - The top right after the owner's image

Brief amendment 28: the owner's close-up of the top right, "● LIVE" over
"Claude Code 2.1.303" over zone B's edge with "Idle". The project bar's
right end is the provider's name and version alone while approvals are
Manual with no change asked; the mode shows there only as Auto or while
changing. Approved on 2026-10-22: 6 of 6 PASS, 27 golden frames, a
Windows Terminal screenshot against the image
([`doc/plan28-evaluation.md`](doc/plan28-evaluation.md)); commit 8fe0077,
then the approval commit.

## Plan 29 - The input editor's row after the owner's image

Brief amendment 29: the owner's image of the editor's row, design only.
Under the draft a separator, then " + " and the pill " <model> High ·
Ultracode " on the tool rows' ground, the editor's messages and the line
count in the free middle, "↯ Auto" at the right end; the "+", "High ·
Ultracode" and "Auto" are placeholders with no function. Zone B is one
row shorter. Approved on 2026-10-22: 7 of 7 PASS, 59 golden frames,
Windows Terminal screenshots at 208x59 and 120x40 against the image
([`doc/plan29-evaluation.md`](doc/plan29-evaluation.md)); commit d696f22,
then the approval commit.

## Plan 30 - The "+" menu after the owner's image

Brief amendment 30: the owner's image of the "+" clicked, design first. A
click on the editor row's band opens a three-row box above the editor's
rule, "🗀  Upload from computer", the band and the item under the
pointer on the active ground (`#272745`), the terminal asked for every
mouse motion only while it is open; Esc, any key or a click elsewhere
closes it; the item does nothing yet, the upload being a later plan's.
Approved on 2026-10-22: 6 of 6 PASS, four golden frames added, Windows
Terminal screenshots with a real click and pointer moves at 208x59 and
120x40 against the image
([`doc/plan30-evaluation.md`](doc/plan30-evaluation.md)); commit 6025513,
then the approval commit.
