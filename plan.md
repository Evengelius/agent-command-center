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
for plan 29, `git show 6025513:plan.md` for plan 30, `git show
8e2a721:plan.md` for plan 31), and its results in its evaluation file.

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

## Plan 32 - Image paste and preview

Brief amendment 32 (2026-10-23): paste an image into the terminal chat as
in the owner's chat, and a preview before sending like the chat's
(`references/10-chat-image-preview.png`,
[reference-design.md](doc/reference-design.md#the-image-preview)). The
criteria below were written before the code; nothing is committed until
the owner has reviewed the implementation and the screenshots.

### Research (D0, [doc/research/image-paste.md](doc/research/image-paste.md))

- **Keys (probe P32a, Windows Terminal 1.23.20211.0).** The owner's
  settings bind Ctrl+V to the terminal's own paste, Ctrl+Shift+V too.
  With an image alone on the clipboard both paste nothing: the terminal
  keeps the V key's press and forwards its release with Ctrl held
  (`ESC[86;47;22;0;40;1_`), which is undocumented. With a file copied in
  Explorer they paste its full path as a bracketed paste; with text, the
  text. Alt+V and Ctrl+Alt+V reach the application as key presses
  whatever the clipboard holds. Claude Code gives Alt+V to its image
  paste on Windows for this reason.
- **Images (probe P32b).** The terminal draws Sixel images; a cell holds
  10x20 of their pixels whatever the font (200x100 pixels covered 20
  columns by 5 rows); text or spaces written into a cell erase the image
  there, and cells left alone keep it.
- **The renderer (Bubble Tea v2.1.0, ultraviolet).** A changed line is
  rewritten from its first to its last changed cell, the cells between
  included; at each frame tick the raw sequences queued with `tea.Raw`
  go out before the frame. The terminal's answer to the device
  attributes request (`ESC[c`) reaches the model as ultraviolet's own
  event, untranslated.
- **The clipboard (golang.design/x/clipboard v0.11.0, approved).** It
  reads an image as PNG (`FmtImage`, from CF_DIBV5 or CF_DIB), the
  original bytes of the Windows "PNG" format that browsers and Snipping
  Tool publish (registered as `image/png`), and copied files
  (`ReadFiles`, CF_HDROP).

### Design

- **D1 Alt+V** attaches what the clipboard holds as images: copied files
  first, each through plan 31's checks by content (a non-image refused by
  name); else the "PNG" format's bytes; else the bitmap as PNG; else the
  toast "no image on the clipboard". It never inserts text and never
  sends.
- **D2 Ctrl+V** pastes text as before. When the terminal keeps its press
  and forwards only its release (an image alone on the clipboard), the
  dashboard, seeing the release with no press and no paste within 200
  ms, attaches the clipboard's image as Alt+V does: undocumented
  behaviour, labelled so in the code and the docs; Alt+V works without
  it. When a paste is exactly the paths of the files the clipboard holds
  and every one is a supported image, they are attached instead of
  their paths inserted; otherwise the text goes in, as today. Where the
  terminal sends Ctrl+V's press, the dashboard's own paste reads the
  clipboard: its text, else its image.
- **D3 A pasted image** is named "pasted-image-N.png" (N from 1 in a
  run), held in memory as PNG bytes and never written to a file, checked
  as an upload is (by content, the provider's limits, scaled down when
  over), shown by plan 31's chip and sent as plan 31's images are; the
  same bytes twice are "already attached"; its bytes are dropped when its
  chip is removed or its message accepted. The stored draft lists it by
  name as pasted: restored, the draft's text and files come back, and a
  toast says the pasted image was not kept.
- **D4 The preview.** When the terminal answers the device attributes
  request with Sixel graphics (attribute 4, asked once at start), a strip
  of thumbnails lies between the editor's rule and the chips' row, side by
  side, one cell apart, the first above the first chip at its left edge
  (corrected on the owner's remark, below; first each above its chip, at
  the chip's left edge): 14 columns by 7 rows
  (140x140 image pixels) when the dashboard has 50 rows or more, 10 by 5
  (100x100) from 30 to 49 rows, no strip under 30; the strip's rows come
  out of zone B's conversation. Each image is cut from its centre to a
  square, scaled by area averaging, its corners rounded (8 pixels at 140,
  6 at 100) and transparent, with a 1-pixel border in the frame's colour.
  A WebP, which nothing here decodes, gets a box of text with "WebP" in
  it. Without Sixel, plan 31's chips alone.
- **D5 Drawing.** The frame keeps the strip's cells blank, marked as the
  thumbnails' (panel 'T', the image's index); the dashboard draws the
  images over them in one raw sequence (the cursor saved, each
  thumbnail's position and its Sixel, the cursor restored, the whole
  inside synchronized output), 50 ms after the last frame that changed a
  strip row, so that frame is out first, and again 300 ms after it. Any
  later frame that changes a strip row (a selection, a resize, a
  full-screen view closed, the focus moving) is followed by the same
  drawing. Each image's Sixel is encoded once per size, when it is
  attached, away from the screen's thread.
- **D6 Keys.** F1's editor group: "Alt+V attach the clipboard's image
  (Ctrl+V too, when the clipboard holds no text)"; the footer is the
  owner's approved row, unchanged.
- **D7 Docs.** ui-design.md (Input editor: the paste, the preview),
  architecture.md (Images: the paste, the preview), reference-design.md
  (the image), brief.md (amendment 32), the research, AGENTS.md (status),
  `doc/plan32-evaluation.md`; golden frames of the strip.

### Criteria

- 32.1 TestPasteImage (`internal/app`, a fake clipboard): Alt+V with image
  data attaches "pasted-image-1.png" with its size, the draft's text and
  cursor unchanged, nothing sent; the same bytes again: "already
  attached"; other bytes: "pasted-image-2.png"; copied files, a PNG and a
  JPEG: both attached by name; a copied text file: refused with its
  reason; text alone: the toast "no image on the clipboard", nothing
  changed; Enter: the session receives the text and the pasted image's
  exact PNG bytes as image/png, the chips gone.
- 32.2 TestPasteCtrlV (`internal/app`): the release of Ctrl+V without its
  press and no paste: after 200 ms the image attached; the release then a
  paste within 200 ms: the text inserted, nothing attached; the press
  then the release: the text pasted once, nothing attached; a paste equal
  to the clipboard's file paths, all images: attached, nothing inserted;
  with one that is not an image: the text inserted, nothing attached.
- 32.3 TestPastedDraft (`internal/app`): a draft holding a pasted image
  and a file, stored and restored by a new model: the text and the file
  back, the toast "pasted-image-1.png was pasted, not kept"; the stored
  files hold no image byte.
- 32.4 TestSixel (`internal/render`): the encoder's output, read back by
  the test's own decoder, gives the image's size from its raster
  attributes, each opaque pixel within the palette's error of its colour,
  each transparent pixel unset, at most 256 colours. TestThumbnail
  (`internal/input`): the 320x160 drawing gives a 140x140 square cut from
  its centre (the green square at its middle), transparent corners, the
  border's pixels in the frame's colour; a 100x100 likewise.
- 32.5 TestPreviewStrip (`internal/render`): Sixel on, two attachments: at
  200x60 seven blank rows between the rule and the chips, panel 'T' items
  0 and 1 of 14x7 cells, the first at its chip's column, the second one
  cell after it (corrected on the owner's remark; first each at its
  chip's column); at 120x40
  five rows, 10x5 cells; at 80x24 the frame equals the one without Sixel;
  without Sixel no strip at any size; a WebP's box reads "WebP"; zone B's
  conversation has the strip's rows fewer.
- 32.6 TestPreviewDraw (`internal/app`): before the terminal's answer, no
  drawing; after an answer holding 4 and an attached image, one raw
  sequence 50 ms after the frame: ESC 7, each thumbnail's position (CUP
  to its first cell) and its Sixel, ESC 8, inside ESC[?2026h and
  ESC[?2026l; again 300 ms after; a frame changing a strip row (a
  selection over it): drawn again; a frame changing other rows only: not;
  the image removed: no more drawing; an answer without 4: no strip.
- 32.7 Golden frames added: the preview at 120x40 and 200x60; every other
  frame unchanged.
- 32.8 Checks: go build, go vet, gofmt; the suite; the race detector on
  the changed packages; the timing step alone within every target.
- 32.9 Windows Terminal, the executable built from this tree in mock mode,
  maximised at 208x59, then at 120x40, the owner's clipboard saved first
  and restored after: the generated PNG copied as a bitmap, then Ctrl+V:
  its chip "pasted-image-1.png" and its thumbnail drawn (the squares
  visible in the strip); a file copied, then Alt+V: a second chip and
  thumbnail; text typed; a selection dragged over the strip and released,
  F1 opened and closed: the thumbnails back each time; a chip's ✕: its
  thumbnail gone; Enter: the message sent, its entry listing the images;
  screenshots of each step.
- 32.10 Docs as D7; every changed text file read byte by byte (no BOM, no
  CR); git status lists the intended files only; nothing committed.

A live turn with a pasted image is not part of this plan: the image goes
out by plan 31's path, proved live on both CLIs; one can run on the
owner's approval.

### Progress

Corrections made as the work went, each with its reason:
- D5: a place an image left (its chip removed, the message sent, the
  editor grown) is written over with the frame's own cells, which erases
  the image, before the images of the frame are drawn. A first run in
  Windows Terminal left both images on the screen after the send: their
  cells were blank before and after, so the renderer never rewrote them
  (P32b: cells left alone keep an image).
- D5: the "+" menu stands on the editor's rule found by its dashes
  (`editorRule`, plan 30): the strip's blank rows were taken for the rule,
  which would have set the menu on the strip; TestPreviewStrip checks it,
  and fails with the old search.
- 32.6: the device attributes answer, recorded from Windows Terminal with
  win32-input-mode on (`\x1b[?61;4;6;7;14;21;22;23;24;28;32;42;52c`,
  probe P32a's da1 part), is also fed to the real program with Ctrl+V's
  lone release (TestWin32PasteRelease).
- 32.7: four frames added, the golden test drawing each scenario at four
  sizes; at 80x24 and 79x24 they equal the attachments' frames, no strip
  under 30 rows.
- D4 and 32.5, on the owner's remark on the screenshots ("there is too
  much space between the two images, it should be way shorter", brief
  amendment 32): the thumbnails side by side, one cell apart, the first
  above the first chip, as many as the width holds, past the chips
  folded into "+N more"; each above its own chip had put the wide chips'
  spacing between them.

- [x] D0 the research (probes P32a and P32b,
  `scratchpad\term\out-p32probe-*`).
- [x] D1 to D3 the paste (`internal/app/paste.go`, `internal/input`
  clipboard and InspectData) and its tests (TestPasteImage,
  TestPasteCtrlV, TestPastePaths, TestPastedDraft, TestWin32PasteRelease).
- [x] D4 and D5 the preview (`internal/render/sixel.go`, the strip in
  `attach.go`, `internal/input/imagethumb.go`, `internal/app/preview.go`)
  and its tests (TestSixel, TestRegion, TestPreviewStrip, TestThumbnail,
  TestPreviewDraw).
- [x] The golden frames (four added, no other changed), the checks: build,
  vet, gofmt, the suite, the race run on input, render and app, the timing
  step alone 7 of 7 (the long session timed in turn with the previous
  commit's tree: the same figures).
- [x] The screenshots (32.9): two prototype runs, then the final runs at
  208x59 and 120x40 once the workstation was unlocked
  (`scratchpad\term\out-p32\paste-max\`, `paste-120x40\`), the owner's
  clipboard saved and restored in each.
- [x] D7 and the evaluation (`doc/plan32-evaluation.md`: 10 of 10 PASS).
- [x] The owner's remark: the thumbnails one cell apart, TestPreviewStrip
  corrected, the suite, screenshots at 208x59 and 120x40 (the
  evaluation's follow-up).
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

## Plan 31 - Image upload

Brief amendment 31: the "+" menu's "Upload from computer" works. The
system's Open dialog, several images at once, each checked by its bytes
(PNG, JPEG, GIF, BMP, TIFF, ICO and WebP accepted, the orientation
applied, conversions to PNG, an image over a provider's limits scaled
down and said so; HEIC, AVIF, JPEG XL, JPEG 2000, PSD, TGA, DDS, HDR,
EXR, camera raws and SVG refused by name); chips under the editor's
rule, removable; sent only when the owner submits, as base64 image
blocks to Claude Code and `image` inputs holding data URLs to Codex, no
event keeping the bytes (Codex writes them back). Approved on
2026-10-23: 9 of 9 PASS, eight golden frames added, the real dialog in
Windows Terminal, live turns on Claude Code (Haiku 5.5) and Codex
(gpt-6-sol) naming the colours and the text, and a third on Codex
checking the echo fix
([`doc/plan31-evaluation.md`](doc/plan31-evaluation.md)); commit 8e2a721,
then the approval commit.
