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
8e2a721:plan.md` for plan 31, `git show b56d23e:plan.md` for plan 32, `git show
bc5d92b:plan.md` for plan 33, `git show 47b50ce:plan.md` for plan 34, `git show
b43cf46:plan.md` for plan 35, `git show c0cebc4:plan.md` for plan 36, `git show
31c22ef:plan.md` for plan 37, `git show dc9c6d9:plan.md` for plan 38, `git show
55e2bed:plan.md` for plan 39), and its results in its evaluation file.

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
the terminal, plan 17 (amendment 11). The session's model, effort and
Ultracode selection was plan 34 (amendment 34); configuring models beyond the
session stays below.

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

## Plan 43 - A right activity sidebar with the Explorer docked

Brief amendment 44 (2026-10-24): a narrow activity sidebar right of zone B
with two icons, Explorer and Skills; the existing Explorer docked between
zone B and the sidebar, shown and hidden by F3 and by its icon; the icons'
inactive, hover and selected looks after the owner's two images
(`references/12-sidebar-collapsed.png`, `references/13-sidebar-explorer.webp`);
Skills visual only. The criteria below were written before the code;
nothing is committed until the owner has reviewed the result.

### What was found

A read-only mapping (three scans, 2026-10-24; scratchpad `plan43/map.md`):

- **Layout** (`internal/render/dashboard.go`, `dashboard`): below 100
  columns a narrow stack (zone A one row, zone B, zone C); from 100 zone A
  takes `max(round(0.28 w), 36)` columns and zone B the rest, joined by
  `beside`, zone C full width under them, the footer last. Zone B is 84
  wide at 120x40, 112 at 156x44, 144 at 200x60, 150 at 208x59. Zone B's
  editor row needs about 51 columns for its two pills. Popups (model
  selector, modes box, "+" menu) are clamped to the screen's width, not to
  zone B. 86 golden frames show zone B; 64 of them are 100 columns or more.
- **Explorer** (plan 17; `internal/render/explorer.go`,
  `internal/app/explorer.go`): F3 opens it as a window centred over the
  dashboard (`FileRect`, `floatWindow`); while open it takes every key and
  mouse event and counts as a window (`WindowOpen`), which turns off F4,
  Alt+P, the "+" menu, the selectors and paste to zone B. Its rows, cursor,
  listing (`input.ListDir`), git marks, keys (↑ ↓ PgUp PgDn Home End → ←
  Enter Esc /) and the viewer it opens are independent of the window's
  geometry; `explorerWindow` adds the frame, a path row and a footer. F3
  first opens the file the entry in front names (`f3`), else the Explorer.
- **Images and pointer**: images are drawn at 10x20 pixels a cell over
  constant blank cells and redrawn when their look changes
  (`previewSignature`, `thumbSequence`); helpers `fillRounded`,
  `strokePaths`, `iconShape`, `blend`. The mouse reports motion without a
  button only while a popup is open (all-motion, 1003); each motion report
  costs an `Update` and a full `View`; `tea.WithFilter` can drop a message
  before both. A pointer at rest may report a wandering cell (plan 36): only
  a change of item may count as a hover.
- **The references** (measured, `scratchpad/plan43/fg.ps1`): the sidebar a
  box of its own, about 8 cells wide; each icon a square tile about 6
  cells by 3 rows; the selected Explorer tile a dark fill `#181C30`, a thin
  purple outline `#7A4EBA`, a lavender accent bar `#A272F1` on its left
  edge, its icon (a tree: a trunk and two folders) light grey-blue
  `#A6B3C6`; the Skills icon (three linked nodes) muted grey `#6F7C88`, no
  tile; the Explorer panel about 30 cells at the reference's scale, a header
  "EXPLORER" `#A785DD` with a chevron at the right, a rule, the root row
  (chevron, open-folder icon, the project's name), folders with a chevron,
  a blue folder icon `#6CA1DB` and a light name, files a grey document
  icon `#A3B1C9`.

### Design

- **D1. Layout.** From 100 columns the upper row becomes zone A | zone B |
  the Explorer panel when shown | the sidebar, zone A's width and place
  unchanged, the sidebar `8` columns wide on the far right, zone C full
  width under all of them, the footer last. The panel is
  `clamp(round(0.15 w), 26, 36)` columns; it docks when zone B keeps at
  least 64 columns; otherwise (compact) it takes zone B's place while it is
  shown. Below 100 columns (the narrow stack) there is no sidebar and the
  shown Explorer takes zone B's place. No zone is narrower than 5 columns.
- **D2. The sidebar.** A box with the zones' thin grey border and no
  title; two square tiles, 6 cells by 3 rows, the Explorer's on the box's
  rows 1-3 and Skills' on rows 5-7, centred. With Sixel each tile is an
  image (60x60 pixels) over constant blank cells, its look one of
  inactive, hover, selected, selected and hovered; without Sixel a text
  form (a glyph, a band ground, a "▎" accent). No text label, no tooltip.
- **D3. The looks**, from the references and the theme: inactive, no
  tile, the icon `#6F7C88`; hover, a rounded tile of the selectors' hover
  ground, the icon brighter; selected (Explorer while the panel is shown),
  the rounded tile `#181C30` with its `#7A4EBA` outline, the `#A272F1`
  accent bar on its left edge, the icon `#A6B3C6`; selected and hovered,
  the selected tile a shade lighter. The Explorer icon a tree (a trunk, two
  folders), Skills three linked nodes, both drawn with the existing helpers
  on the same grid and weight.
- **D4. Hover.** While the sidebar shows, the mouse reports every motion
  (1003); only a change of tile under the pointer changes the hover (plan
  36's rule); a filter before `Update` drops a motion report that changes
  no hover while no popup is open and no drag runs, so idle motion costs no
  frame. Hover never changes what is selected.
- **D5. Showing and hiding.** `ui.Explorer.Open` is the one state. F3 on
  the dashboard, a click on the Explorer tile and a click on the panel's
  chevron toggle it; opening gives the panel the keyboard focus, closing
  gives it back to zone B. F3 keeps its file meaning where it had one (a
  focused approval card, an entry shown whole, or the entry cursor on a
  call naming a file opens that file in the viewer, plan 17); from the
  viewer or the quick open, F3 closes them and toggles the panel; from the
  inspection or the details view, F3 closes it and shows the panel; from a
  zoomed zone, which covers the panel as they do, F3 ends the zoom and
  shows it (step 4, the review). A click on the Skills tile does nothing:
  no state, focus or selection changes.
- **D6. The panel**, docked, reusing the Explorer's rows, listing, git
  marks, cursor and keys: a header row "EXPLORER" in the title colour,
  bold, with "⌄" at its right end; a rule; the root row "⌄ 🗀 <project>"
  (the project folder's name); the tree under it, one step deeper,
  folders "›" or "⌄", a folder icon in blue and the name, files a document
  icon in grey and the name, git marks at the right as before. The path
  row and footer of the old window go: its keys go to the global footer
  while the panel has the focus, git's failure note to the panel's last
  row. Rows scroll inside the panel, the header, rule and root row stay.
  Its border is the title colour while it has the focus, grey otherwise.
- **D7. Keys and focus.** The panel has its own focus; F6 and Tab go B →
  the panel (when shown) → C → A. With the focus on it the Explorer's keys
  work as before, except that Esc on a top-level row gives the focus back
  to zone B and leaves the panel shown. A click on a row moves its cursor
  or opens it as before, the wheel scrolls it only over the panel, a click
  in zone B gives B the focus. Enter on a file opens the floating viewer;
  Esc from it returns to the panel. Where the panel takes zone B's place,
  zone B taking the focus hides it (the compact rule, step 3), so the focus
  is never on a zone not drawn. While the panel has the focus or takes zone
  B's place, the footer shows a waiting approval first, and in zone B's
  place a quit question before it (step 4, the review).
- **D8. Not a window.** The docked panel is no longer a floating window
  (`WindowOpen` leaves it out): F4, Alt+P, the "+" menu, the selectors,
  paste and the draft keep working while it is shown; opening and closing
  it keep the draft, the attachments and B's scroll. The viewer and the
  quick open stay floating windows. With the focus on the docked panel the
  draft beside it keeps the clipboard: a paste, Alt+V and Ctrl+C on the
  draft's selection act on the draft, as when the old window left B the
  focus; in zone B's place a paste goes nowhere and Ctrl+C interrupts, as
  from zones A and C. F2 from the panel, the viewer or the quick open
  leaves the panel as it is (step 4, the review).
- **D9. Popups** (the model selector, the modes box, the "+" menu) stay
  inside zone B: clamped to its right edge, never over the panel or the
  sidebar. One wider than zone B keeps its width and extends left over
  zone A, as the model selector did at 100 and 101 columns before: from
  100 to 109 columns, and beside the panel docked at 136 to 138 (step 4,
  the review: narrowing it would cut its Effort row).
- **D10. Unchanged**: zone A, zone C, the footer's place, zone B's
  internal design (only narrower), every shortcut but F3's toggle; the
  footer and F1 say "F3 Explorer".
- **D11. Images beside wide characters** (step 4, the review's
  verification): Bubble Tea clears its buffer before each frame, and
  ultraviolet repaints every touched row holding a wide character from
  column 0, changed or not, which erases the tiles beside a ✅ in zone A or
  B until a frame changes their rows. A third patch of the vendored copy
  (`third_party/ultraviolet/PATCHES.md`) leaves such a row alone when
  unchanged and otherwise writes it only up to its last changed cell. The
  images are also drawn again after every size report, the same size
  included: Bubble Tea erases the screen for each.

### Criteria

1. **43.1 Collapsed layout** at 120x40, 156x44, 200x60 and 208x59: the
   sidebar 8 columns wide right of zone B on the upper rows only, zone A
   as before (same width, same rows), zone B `w - A - 8`, zone C full
   width and as before, the footer last.
2. **43.2 Expanded layout**: the panel `clamp(round(0.15 w), 26, 36)`
   wide between zone B and the sidebar at 156x44, 200x60 and 208x59
   (zone B 78, 106 and 111); at 120x40 (zone B would be 50) it takes zone
   B's place, the sidebar kept; at 80x24 no sidebar, the panel in zone B's
   place; zone A and C as before; TestFrameInvariants from 80 to 240
   columns: no width under 5, every row exactly the screen's width.
3. **43.3 The sidebar's tiles**: exactly two, Explorer then Skills, 6x3
   cells on the box's rows 1-3 and 5-7, centred; with Sixel blank cells
   that never change under the images, without Sixel the text form in
   each look.
4. **43.4 The images**: from their pixels, selected: fill within 6
   levels of `#181C30`, outline of `#7A4EBA`, an accent bar of `#A272F1` on
   the left edge, the icon `#A6B3C6`; hover: the hover ground, the icon
   brighter than inactive; inactive: transparent ground, icon `#6F7C88`;
   the Explorer icon a trunk and two folders, Skills three nodes and their
   links; rounded corners.
5. **43.5 Hover**: all-motion while the sidebar shows; a motion onto a
   tile hovers it, off it unhovers it, a report wandering within one tile
   changes nothing; hovering Skills while Explorer is selected keeps
   Explorer selected; a motion that changes no hover reaches neither
   `Update` nor `View` (counted); a timing test of motion within the 16 ms
   target.
6. **43.6 Toggle**: F3 and a click on the Explorer tile open the panel
   (focus on it, the tile selected) and close it (focus to B, zone B's
   width back); the chevron's click closes it; the same state either way;
   a click on Skills changes nothing; F3's file meaning kept in its three
   cases; F3 from the viewer, the quick open, the inspection and the
   details view as D5 says, and from a zoomed zone (added in step 4).
7. **43.7 The panel's content**: header "EXPLORER" (title colour, bold)
   and "⌄" at its right; the rule; the root row with the project's own
   name; the tree of the real folder (a test project with folders and
   files), folders and files with their chevrons, icons and styles; git
   marks; scrolling keeps the header, rule and root row.
8. **43.8 The Explorer's behaviour kept**: its existing tests' keys,
   listing, marks, deleted entries, stale listing, quick open, the viewer
   and back, against the docked panel; Esc on a top-level row gives B the
   focus with the panel shown; the wheel over zone B does not move the
   panel; the state (folders, cursor, marks) survives hiding and showing;
   the compact rule (added in step 3); F2 from the panel, the viewer or the
   quick open keeps the panel (added in step 4).
9. **43.9 Not a window**: with the panel shown, F4 opens the modes box,
   Alt+P the model selector, a click the "+" menu, a paste reaches the
   draft; the draft, the attachments and B's scroll are the same before
   opening, while shown and after closing. With the focus on the docked
   panel, a paste and Alt+V reach the draft and Ctrl+C copies its selection
   without interrupting the turn; in zone B's place, neither (added in
   step 4).
10. **43.10 Focus and hints**: F6 and Tab through B, the panel, C and A;
    the panel's border in the title colour while focused, grey otherwise;
    the footer shows the panel's keys while it has the focus; the footer
    and F1 say "F3 Explorer"; a waiting approval and, in zone B's place, a
    quit question come first in the footer (added in step 4).
11. **43.11 Popups inside zone B**: the model selector, the modes box and
    the "+" menu end at zone B's right edge or before it, with the panel
    shown and hidden, at 120x40 to 208x59; one no wider than zone B starts
    at its left edge or after it, a wider one ends at its right edge, from
    100x40 (amended in step 4).
12. **43.12 Unchanged elsewhere**: every golden frame change explained by
    a script: zone A's rows and zone C's rows identical to before, zone B
    narrower with the same content reflowed, the new columns; the frames
    under 100 columns change only where the Explorer was shown.
13. **43.13 The checks**: build, vet and gofmt clean; `go test ./...` ok;
    `-race` on render, app and input ok; the timing step alone within its
    targets, with the motion test; breakages of the new code caught by
    tests.
14. **43.14 Windows Terminal**, the acc tab's profile (208x59) and
    120x40, real keys and clicks: the collapsed sidebar, the panel shown,
    each tile hovered, Explorer selected, F3 showing and hiding; the
    colours read from the screenshots against the references'.
15. **43.15 Images beside wide characters** (added in step 4, D11): the
    real program at 208x59 with Sixel, a list of ✅ beside both tiles: a key
    typed and a running turn's spinner frames write none of those rows;
    the patched renderer's own test; the module's upstream tests pass;
    Windows Terminal: the tiles stay after typing, where step 3's build
    erased them.

### Steps

1. The finding, brief amendment 44, the criteria.
2. The layout, the docked panel, its keys, focus and mouse, F3 (text
   tiles).
3. The tiles' images, hover and the motion filter.
4. An independent review of the result; its findings fixed.
5. The checks; Windows Terminal; the documents; the evaluation
   (`doc/plan43-evaluation.md`); the report.

### Progress

- 2026-10-24: step 1 done. Step 2 done: the layout, the docked panel, its
  keys, focus and mouse, F3, the text tiles; 66 golden frames rewritten,
  each change explained by a script. Step 3 done: the tiles' images, the
  hover, the motion filter, the compact rule. Step 4 done: a review in
  three lenses (layout, interaction, pointer) and a verification of its one
  medium finding. That finding (the tiles' images redrawn ten times a
  second in a running turn) was refuted as deliberate and cheap, and the
  verification found D11's erasure instead; the low findings fixed or
  decided (D5, D7, D8, D9, the hover after a press held off a tile, the
  timing step's motion test), D11 patched, each fix tested and checked by
  a mutation run (13 breakages, all caught).
- 2026-10-25: step 5 done: the checks on the final tree, Windows Terminal
  at 208x59 and 120x40, the documents (ui-design.md's Right sidebar and
  Explorer, its layouts redrawn; reference-design.md; architecture.md;
  PATCHES.md; the timing step with TestMotionToFrame), and
  `doc/plan43-evaluation.md`: 15 criteria, 15 PASS. A mutation run that
  wrote into the tree was cut short by a stop of the session and left nine
  files zero-filled; they were rebuilt and proved byte for byte by the
  `git diff` hash, and the mutations now run through `go test -overlay`.
  Waiting for the owner's review; nothing committed.

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

## Plan 32 - Image paste and preview

Brief amendment 32: paste an image as in the owner's chat, and a preview
before sending. Alt+V attaches the clipboard's images (copied files, the
"PNG" format's bytes, else the bitmap as PNG); Ctrl+V pastes text as
before and, when Windows Terminal keeps its press for want of text,
attaches the image on the key's lone release (undocumented, probe P32a);
a paste of the copied files' paths attaches them when all are images;
a pasted image is held in memory only. With Sixel graphics, the images
show as thumbnails between the rule and the chips, rounded squares cut
from their centre, 14x7 cells from 50 rows, 10x5 from 30, side by side
one cell apart (the owner's remark on the screenshots), drawn after the
frame and again after any frame that changes their rows, a vacated
place written over with the frame's cells. Approved on 2026-10-23: 10 of
10 PASS, four golden frames added, Windows Terminal at 208x59 and
120x40 with the owner's clipboard saved and restored, the remark in a
follow-up ([`doc/plan32-evaluation.md`](doc/plan32-evaluation.md));
commit b56d23e, then the approval commit.

## Plan 33 - Unsent drafts and images through crashes and restarts

Brief amendment 33: whether unsent image attachments survive a crash, a
forced termination, a restart and a resumed session; an investigation
first, then its nine corrections. Each dashboard's draft has a folder of
its own, its lock held while the dashboard owns it, taken back by the
project's next start (its own CLI's first, then the newest); images are
saved at once, the text every 500 ms; pasted images are kept as
`<sha256>.png` until their message is accepted; a message waiting for
its acceptance is stored with what was typed since; Back forgets the
attachments; a damaged list is set aside; a changed file is attached
again as it is now; refused renames and failed saves are tried again;
unused files are swept at each start. An independent review found five
defects, fixed; on the owner's questions, a Codex session resumed with
images, a slow and an offline share were simulated, fixed where they
lost data, and a logoff was not simulated. Approved on 2026-10-23: 14 of
14 PASS and the follow-up
([`doc/plan33-evaluation.md`](doc/plan33-evaluation.md)); commit
bc5d92b, then the approval commit.

## Plan 34 - The model selector, its effort and Ultracode

Brief amendment 34: a click on the editor row's model pill, or Alt+P,
opens the active CLI's own models with the effort and, for Claude Code,
Ultracode, after the owner's image. Claude Code's list comes from
`initialize`, its changes go by `set_model` and `apply_flag_settings`
and are read back by the undocumented `get_settings`; Codex's list from
every page of `model/list`, its changes ride the next `turn/start`,
read back by `thread/read`. The screen shows a value only once the CLI
reports it in force; an effort held by `CLAUDE_CODE_EFFORT_LEVEL` is
shown and never changed; Claude Code's changes wait for a turn's end. An
independent review found four defects and six doubts, fixed, and three
small points on its second pass. At the owner's request
`CLAUDE_CODE_EFFORT_LEVEL` was removed from their Claude Code settings,
and the dashboard then set the effort live; amendment 35, an opt-in line
telling the agent its settings, was built and withdrawn. Approved on
2026-10-24: 20 of 21 PASS, 34.18 (golden frames changed beyond the
pill's values) accepted
([`doc/plan34-evaluation.md`](doc/plan34-evaluation.md)); commit
47b50ce, then the approval commit.

## Plan 35 - The Ultracode toggle after the owner's image

Brief amendment 36: the owner's image of the Ultracode toggle, a lavender
pill with fully rounded ends and a round white knob, against the
dashboard's `▐   ●▌`. Where the terminal draws Sixel, the toggle is an
image over blank cells of its row, drawn as the preview's thumbnails are:
a pill of 40x20 pixels in an image of five cells whose last carries the
band, since the renderer writes every cell between a row's first and last
changes and a band cell after the image made it blink when the band came
or went (bursts of captures in Windows Terminal); without Sixel, a text
pill of Powerline's half circles. Approved on 2026-10-24: 8 of 8 PASS
([`doc/plan35-evaluation.md`](doc/plan35-evaluation.md)); commit
b43cf46, then the approval commit.

## Plan 36 - The selector's cursor and a pointer that has not moved

Brief amendment 37: Alt+P could reopen the model selector with its
cursor on the row under a resting pointer, so that Enter acted on that
row. The probe (Bubble Tea's `TEA_TRACE`) showed Windows Terminal
reporting a pointer put down by software, without its moving, once the
selector asked for every motion, its reports wandering a cell. The
selector now moves its cursor only when the pointer comes onto another
row than the last report's, the first report after Alt+P only recording
where it rests. Approved on 2026-10-24: 7 of 7 PASS
([`doc/plan36-evaluation.md`](doc/plan36-evaluation.md)); commit
c0cebc4, then the approval commit.

## Plan 37 - The effort slider after the owner's images

Brief amendment 38: the owner's five images of one pill-shaped effort
slider at five levels replaced the selector's effort dropdown. Effort
reads "Effort (<level>)", "Controls reasoning depth" and the slider: the
model's own levels as dots, a fill of `#6784C1` growing to the thumb,
the whole pill lit by a purple gradient with specks at the highest
level, a level asked for ringed, a held effort greyed; with Sixel an
image of 170x20 pixels over 17 blank cells, its thumb gliding in 180 ms;
without, 16 text cells. Effort and Ultracode share label, description
and control columns; ← → and a click on the slider ask for a level.
Approved on 2026-10-24: 9 of 9 PASS
([`doc/plan37-evaluation.md`](doc/plan37-evaluation.md)); commit
31c22ef, then the approval commit.

## Plan 38 - Instant Codex model and effort changes, as in VS Code

Brief amendment 39: a Codex model or effort change took effect only with
the next message, where VS Code's Codex extension changes it at once. The
dashboard now asks for Codex's experimental API at initialize, as that
extension does, and writes a change between turns at once with
`thread/settings/update`, the `thread/settings/updated` that follows
putting it in force (P38x: 1 to 3 ms); a change asked during a turn is
written at its end; a Codex that refuses the update takes every change
with the next message, as before. Approved on 2026-10-24: 12 of 12 PASS
([`doc/plan38-evaluation.md`](doc/plan38-evaluation.md)); commit
dc9c6d9, then the approval commit.

## Plan 39 - The permission modes selector after the owner's images

Brief amendment 40: the editor row's right-hand "Auto" became the mode
pill of the owner's third image, and opens a "MODES" popup after the
first two: the active CLI's own modes (Claude Code's Manual, Edit
automatically, Plan and Auto; Codex's Ask for approval, Approve for me and
Full access), each applied through the CLI (`set_permission_mode`; the
experimental `thread/settings/update` with the presets of VS Code's Codex
extension) and shown in force only once the CLI reports it; Codex's Full
access only after a confirmation. The icons, the check and the band's
rounded ends are Sixel strips beside the text cells, the pill a Sixel
image whose title Windows' GDI draws. Approved on 2026-10-24: 15 of 15
PASS ([`doc/plan39-evaluation.md`](doc/plan39-evaluation.md)); commit
55e2bed, then the approval commit.

## Plan 40 - No empty rows in zone C's execution timeline

Brief amendment 41: on Codex, zone C's execution timeline and the
inspection's Messages tab showed a subagent's rows "assistant │" with no
text, one for each of its reasoning steps: since plan 22 a subagent's
reasoning summaries leave assistant entries holding only a thinking mark,
which zone B never drew. Neither draws an entry with no text now (a
thinking mark, a message that ended empty, a notice with none), a rule
only between two drawn entries. Approved on 2026-10-24: 6 of 6 PASS
([`doc/plan40-evaluation.md`](doc/plan40-evaluation.md)); commit
4158751, then the approval commit.

## Plan 41 - No authorization on low-risk review rows

Brief amendment 42: a Codex auto-review row rated low risk read
"authorization unknown": Codex's reviewer answers a low-risk action with
a bare allow, and Codex then reports the authorization as unknown with a
stock rationale. After a preview, the owner chose to drop the
authorization from review rows rated low risk, in zone B, zone C's
timeline and the inspection; at medium, high and critical it stays.
Approved on 2026-10-24: 7 of 7 PASS
([`doc/plan41-evaluation.md`](doc/plan41-evaluation.md)); commit
3904f13, then the approval commit.

## Plan 42 - The mouse in zone B: a click to fold, a double click to open

Brief amendment 43: beside the keyboard's path, which the owner approved
and which stays as it is, a click anywhere on a foldable item's row in
zone B (a run of calls, a thinking phase's summary row) unfolds it or
folds it again, and a double click on a row of an entry Enter shows whole
opens its full view as Enter does, an approval's rows the approval whole
as from its card. The mouse puts the entry cursor nowhere and shows none
of the keyboard's hints. On those rows a double click no longer selects a
word; a drag still does. Approved on 2026-10-24: 7 of 7 PASS
([`doc/plan42-evaluation.md`](doc/plan42-evaluation.md)); commit
11f794d, then the approval commit.
