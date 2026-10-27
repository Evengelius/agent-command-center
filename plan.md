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

## Plan 51 - Home screen redesign

Brief amendment 52 (2026-10-26): the home screen (the startup screen) made
to look like the owner's image, `references/16-home-screen.webp`, the
single source of truth for its look, with an animated Space Invaders
banner; its functions kept: CLI detection and choice, the project field,
validation, the project store, the browser, Worktrees and every key. The
owner approved the revised plan with its recommended decisions and one
safeguard: the project field keeps its editing, selection and copy/paste.
A visual review with the owner comes before step 4.

### What was found (2026-10-26, on dc26ad4)

- **Today's screen.** `render/startup.go` draws one centred column; the
  keys are `app/keys.go`'s `startupKey` (↑↓, Tab, Shift+Tab, Enter, f, Del,
  Ctrl+O, Ctrl+W, Esc, Ctrl+Q). The project field is an `input.Editor`
  edited by `editKey` on one line (typing, Backspace, Delete, ← →, Home,
  End, Ctrl+V, a bracketed paste); it has no selection of its own: text is
  selected and copied by the terminal, from the screen. The mouse is off on
  this screen (`app.go` sets a mouse mode on the dashboard only); an idle
  startup screen schedules no frame; the application has no version.
- **The design** (measured in
  [reference-design.md, The home screen](doc/reference-design.md#the-home-screen)):
  1347x757 pixels; six text sizes, their characters 7.7 (the header's
  right), 8.0 (project rows, footer), 8.4 (section titles), 8.95 (the
  title), 9.1 (questions, CLI choices, the field) and 10.6 pixels (panel
  titles, the button) apart; project rows about 29.3 pixels apart, 1.6
  rows of a terminal at its scale; header 0-45, banner 45-375, panels
  380-710, footer 712-757.
- **Windows Terminal 1.23.20211.0**, probed by `scratchpad\plan51\probe`
  (`probe51.exe`, maximised at 9 pt, 208x59, and at 120x40):
  - it answers CSI 14t and 16t in virtual pixels, 10x20 a cell: the real
    cell (9.0x17.0 screen pixels at 9 pt) cannot be read, and images are
    resampled to it (90% across, 85% down at 9 pt);
  - an image drawn over text hides it, the text kept in the buffer (UI
    Automation reads it); text written over an image replaces it in those
    cells;
  - a tab's shell is a child of `WindowsTerminal.exe`, whose folder in its
    package holds `CascadiaMono.ttf`, readable by the owner's account;
    GDI loads it privately (`AddFontResourceExW`, `FR_PRIVATE`), for the
    process alone; the design's Projects panel
    drawn by GDI at the design's sizes and row spacing lies on the design,
    in place, at both sizes;
  - costs: a 600x40-pixel tile 1.2 ms to encode and 0.6 ms in the
    terminal (p50); a list's two-row move 1.6 ms in the terminal; the
    whole screen 100 ms to encode at 208x59 and 40 ms in the terminal; a
    banner frame of 40 sprite tiles (24.5 KB) 7 to 13 ms to encode, the
    terminal at 18 to 27% of a core at 12.5 frames a second.

### Design

- **D1. Two modes.** Picture mode when the terminal draws Sixel and the
  design scale k = min(cols×10/1347, rows×20/427), in image pixels per
  design pixel, is at least 0.875 (from 118 columns): the header, the
  panels and the footer drawn as images, the design scaled by k, the
  header at the top, the footer on the last rows, the panels just above
  it, horizontally centred. Grid mode otherwise: the same structure in
  terminal text, one text size on whole rows, frames in box drawing, in
  the design's colours (each with its 16-colour fallback).
- **D2. The banner** fills the rows between the header and the panels when
  the terminal draws Sixel and at least 8 rows are free (a band of aliens,
  a shot's flight and the ship); otherwise it is hidden, and a hidden
  banner schedules no frame, sends nothing and keeps no buffer. The breaks
  this gives: 208x59 picture, banner 26 rows; 150x40 picture, 16 rows;
  120x40 picture, 20 rows; 120x30 picture, 10 rows; 120x26 picture, no
  banner; 100x40 grid, 22 rows; 80x24 grid, no banner.
- **D3. Text in images** is drawn by GDI in Cascadia Mono, loaded from the
  hosting Windows Terminal's package (found from the terminal's process),
  else an installed Cascadia Mono, else Consolas (today's fallback), each
  of the design's text kinds at its size times k (thirteen, measured in
  step 2), drawn four times larger and averaged down.
- **D4. The project field** (the owner's safeguard): in picture mode its
  text is terminal text on one row, its cursor, horizontal scrolling and
  the terminal's own selection and copy as today, and editing and paste,
  which are input, unchanged; its outline, ground, rounded corners and
  folder button are images around that row, as the commit message box's
  (plan 45). The row is the one nearest the design's, the box centred on
  it. The visual impact: the path at the terminal's text size, about 30%
  smaller than the design's at 208x59, the same within 1% at 150x40,
  about 23% larger at 120x40. Every other cell under the picture is blank:
  no text lies under an image.
- **D5. The header:** the design's logo and title; "v0.1.0" from a version
  constant, 0.1.0; "Windows" from the operating system; the status from
  the startup machine: "Detecting…", "Ready", "Validating…", "No agent
  CLI".
- **D6. New Session:** the design's option boxes, the chosen one in purple,
  each CLI's version in parentheses; a CLI not installed greyed, "not
  installed: <reason>" for its version, not choosable; "(detecting…)" and
  START SESSION dimmed while detecting; validation messages and warnings
  in the row between the field and START SESSION, the panel growing into
  the banner when they wrap; the resolved path there only when it differs
  from what is typed (`~`, `/mnt/c/…`). Gone: the subtitle, the `*`, the
  CLI's path, the help line, the branch.
- **D7. Projects:** FAVORITES (★) and RECENT PROJECTS (☆) from the store,
  at the design's row spacing; the band on the list's selected row; the
  separators; ages as the design writes them ("now", "8m ago", "2h ago",
  "1d ago", "never"); more projects than fit: the list follows the
  selection, with a slim scroll mark; the store's problems and "No
  projects yet" in the panel; a section with no row hidden.
- **D8. The footer:** "Shortcuts:" with Ctrl+N New Session, Ctrl+O Open
  Project and Ctrl+Q Quit, Ctrl+Q's group in Ctrl+P's place (no command
  palette, the owner's decision). ↑↓, Tab, Enter, f, Del and Ctrl+W work
  as today, documented, not shown.
- **D9. Ctrl+N** puts the form back as the screen opened it: the default
  agent CLI, the starting project folder, the focus on Agent CLI, no
  message; not while validating.
- **D10. Focus:** the focused control outlined in the design's purple (the
  option box, the field, START SESSION; the list's band brighter); the
  field's cursor as today.
- **D11. The mouse:** the screen takes clicks, as the dashboard, F12 giving
  the mouse back to the terminal: an option chooses it; the field takes
  the focus; the folder button opens the browser (Ctrl+O); a project row
  is selected; a star toggles the favourite (f); START SESSION starts
  (Enter). No double click, wheel or footer click. The terminal's own
  selection then takes Shift and a drag, or F12.
- **D12. The banner's scene**, as agreed: the design's background (its
  ground, dot grid, halftone bands, stars and sparkles) drawn once a size;
  its sprites (two kinds of alien, two frames each, the ship, the
  two-bolt shots, the pink explosion) with their glow; aliens appearing at
  random places across the width, drifting, descending and replaced; the
  ship wandering in every direction and firing; a hit exploding; 10 to
  12.5 frames a second, only the tiles that change sent, in a fixed
  palette; paused while the terminal window lacks the focus; seeded, so
  tests are deterministic.
- **D13. Updates in picture mode:** the picture drawn whole on entry and
  after a resize (debounced), then only the tiles an update changes (an
  outline when the focus moves, list rows when the selection moves, a
  star, the status), after the frame and in synchronized output, as
  today's images.
- **D14. Unchanged:** the project browser, the Worktrees view, the
  dashboard; validation and the store; every key.

### Criteria

1. **51.1 Picture fidelity.** Windows Terminal screenshots at 208x59,
   150x40 and 120x40 against the design scaled by k: every element of the
   design present; every edge within 2 screen pixels of the design's,
   apart from the terminal's resampling, measured and reported; flat
   colours within 2 per channel; the seven project rows at the design's
   spacing; each text at the design's size times k, except the field's
   path (D4).
2. **51.2 Modes and banner** as D2 lists them, at the seven sizes, by state
   tests and screenshots.
3. **51.3 A hidden banner** schedules no frame, writes no tile and keeps no
   buffer.
4. **51.4 Behaviour unchanged.** The tests of the startup screen, the
   store, validation, the browser and Worktrees keep their expectations,
   apart from drawn text the design changes (listed); the keys as today,
   plus Ctrl+N; the field's editing, paste, and the terminal's selection
   and copy of its text, as today (D4).
5. **51.5 The mouse** as D11 maps it, by state tests.
6. **51.6 The banner's behaviour**, a fixed seed over 1,000 frames: aliens
   appearing across the whole width, moving and descending; the ship
   moving in all four directions and firing; each hit exploding; aliens
   replaced.
7. **51.7 Performance.** Typing in the field with the banner running,
   keystroke to frame p95 at most 16 ms at 208x59 (the timing step); a
   banner frame at most 2 ms (p95) in the application; the terminal at
   most 8% of a core while the banner runs; the picture on screen within
   300 ms of entering.
8. **51.8 Grid mode** right without Sixel and in 16 colours: golden frames
   at the four sizes and the message frames, each reviewed as a diff.
9. **51.9 End to end.** In Windows Terminal, maximised then at 120x40: a
   session started from the new screen by the keyboard and by the mouse;
   Ctrl+N; the field's path selected by the terminal and pasted back; the
   clipboard restored.
10. **51.10 Mutants** on the new logic, through `go test -overlay`.
11. **51.11 The checks:** build, vet, gofmt, the whole suite, `-race` on
    `internal/app` and `internal/render`, the timing step alone.

### Steps

1. Record amendment 52 and this plan; the design measured into
   reference-design.md (every element's geometry, the text sizes, the
   colours from the pixels, the icons, the banner's background and
   sprites); the breaks (D1, D2) and the font's source (D3) fixed.
2. The picture: the layout at a scale, a compositor (rounded boxes,
   lines, icons, GDI text), tiles and the updates they need, a
   fixed-palette encoding; pixel tests with a deterministic stand-in for
   GDI's glyphs.
3. Grid mode, with its golden frames. Then **the owner's visual review**:
   screenshots beside the design, picture mode at 208x59, 150x40 and
   120x40, grid mode at 100x30 and 80x24.
4. The application: the modes and breaks, images after frames, resizes,
   focus, keys, Ctrl+N, the mouse, messages.
5. The banner (D12), with its tests and timing.
6. The documents: ui-design.md, architecture.md, AGENTS.md.
7. Verification (51.1 to 51.11), then `doc/plan51-evaluation.md`.

### Progress

- 2026-10-26: step 1 done. Amendment 52 recorded, the image copied as
  `references/16-home-screen.webp` (byte for byte, SHA-256 ab1c95bc…),
  this plan written with the probe's results; the design measured into
  reference-design.md, The home screen (regions, the six text sizes and
  their colours, every box, icon and row, the banner's ground, lattice,
  bands, stars and sprites; the sprites' pixel grids left to step 5); the
  breaks fixed (D1, D2) and the font's source checked (D3). The picture is
  put on screen by step 2 already, so that the review before step 4 shows
  the real screen.
- 2026-10-26: step 2 done. The picture (`render/home.go`, `homepaint.go`,
  `hometile.go`; `app/home.go`): the layout at a scale, the painter, GDI's
  glyphs, the tiles and their updates, after the frame. Found and fixed in
  Windows Terminal:
  - a Sixel image whose last band of six pixels passes the screen's bottom
    scrolls the screen a row: the picture's last three rows (60 pixels,
    ten bands) are a tile of their own, any change reaching them written
    as that tile;
  - Sixel's colours are percentages, a colour drawn within 1.3 of a unit
    of its own; a median cut per tile drew the page 2 units apart in two
    tiles (a seam): each tile keeps its 32 commonest colours exact, the
    same in every tile (D13's "fixed palette" for the picture; the
    banner's own in step 5);
  - GDI's hinting draws text crisper and heavier than the design's
    smooth text: each glyph is drawn four times larger and averaged down;
    its ink then matches the design's within about 10% at 208x59, 150x40
    and 120x40 (Regular; SemiLight lighter, SemiBold heavier).
  The design measured again, on its glyph centres and stems in
  continuous coordinates (reference-design.md, The home screen, Text):
  thirteen text kinds, each its advance, capitals' height and place
  (the questions 8.58 apart, not 9.1; the names 8.19, the paths 8.02; the
  footer 7.78 to 7.88; START SESSION 9.92, semi-bold); the project rows'
  middles fitted within 0.3 pixels (the favourites 30.09 apart, the recent
  projects 28.7); the separators midway; the marks' colours the mean of
  their brightest pixels. Painted at the design's scale beside the
  design, every edge and text within a pixel or two (the field's path
  apart, D4). Tests: `render/home_test.go` (layout, rows, frame, painter,
  tiles, exact colours), `app/home_test.go` (the sequence), `input`'s
  TestGlyph; mutants through `-overlay` for the exact colours and the
  last rows' tile.
- 2026-10-26: step 3 done. Grid mode (`render/homegrid.go`, replacing
  `startup`): the header and a blank row; with Sixel, the banner's rows
  above the panels' least height (14 rows: D2's 22 at 100x40, none at
  80x24), an image over blank cells; the panels side by side in the
  design's proportions, in box drawing: the options and START SESSION a
  row each on their fills, the field in a box with its folder button, the
  messages between the field and START SESSION, wrapped whole; the
  projects in their sections with separators while they all fit, the
  list following the selection with a slim mark when they do not; the
  focus's outline as purple ends of the focused row (D10); a blank row
  and the footer. The panels grow into the banner and the blank rows when
  NEW SESSION needs it; when its messages need more rows than the screen
  has beside PROJECTS, NEW SESSION takes the width and PROJECTS goes under
  it. 16 colours: no ground, START SESSION's label in the button's
  colour. Golden frames rewritten (18 startup frames, each reviewed), the
  tests of the old screen's wording adapted to the panels (the message
  frames, the start gate, the Codex proxy's lines: read in NEW SESSION
  whatever the wrap; "Start session" now "START SESSION →"); the
  selection move's size bound in TestHomeSequence a third of the whole
  (the smooth glyphs' tiles: 70 KB of 266 KB). Screenshots beside the
  design at 208x59, 150x40, 120x40 (picture) and 100x30, 80x24 (grid):
  waiting for the owner's review before step 4.
- 2026-10-27: the owner approved the look; step 4 done. In picture mode
  the focused control outlined in the design's purple, START SESSION's in
  white, the list's band brighter (D10); START SESSION dimmed while
  detecting, with no agent CLI, while validating; the notes between the
  field and START SESSION, a line each beyond the first growing the panels
  into the banner (16 design pixels a line, the field with them), a
  reason too long for its option among them in full; PROJECTS' store
  problem or "No projects yet" above its rows, the slim mark when the
  rows do not all fit (D6, D7); the notes shared with grid mode. The
  mouse (D11): the home screen takes clicks (F12 gives them back, not in
  the browser or Worktrees), each cell a click acts on carrying its item
  (`render.HomePanel`): an option, the field, its folder button, START
  SESSION (as Enter), a project's row, its star. Ctrl+N (D9): a startup
  machine transition, `Detecting`/`Asking` | form reset, documented in
  events-and-state.md with its invalid clause and tested by
  TestStateMachines. The picture waits 120 ms after a size report before
  it is drawn whole, and the second drawing writes the first's sequence
  again rather than encoding it twice (D13). Checked in Windows Terminal
  by the mouse, maximised: the field, a row, a star, an option, START
  SESSION's note, Ctrl+N. The harness's SendInput Tab reaches the
  application twice (TEA_TRACE: two key-down records a press), so the Tab
  order is covered by the unit tests. Tests: TestHomePictureFocus,
  TestHomePictureNotes, TestHomeTargets, TestHomeScrollMark (render),
  TestHomeMouse, TestHomeCtrlN, TestHomeSettle (app).
- 2026-10-27: step 5 done. The banner (D12). Its background gains the
  design's halftone bands and sparkles, measured on the design: four bands
  of the lattice's dots, brighter to their middles and dithered at their
  edges; four white crosses and two dim ones; their places scaled to the
  banner's height. The scene (`render/banner.go`): aliens appearing at
  random places clear of the others, 33 to the design's area (40 at most),
  marching as the classic game's, the aliens of a band of rows stepping
  together every 0.6 s, the three bands in turn, turning at the edges and
  before another, waiting above one, leaving at 68% of the height and
  replaced; the ship wandering in its band and firing two-bolt shots,
  aimed when an alien is above it; a hit exploding into a starburst and
  pink particles in a dithered purple haze. The view
  (`render/bannerview.go`) draws it over a copy of the background, on the
  whole rows within the design's banner: the sprites at the design's 2.5
  design pixels a pixel, crisp, in three shades with a two-step glow, so
  that a moving sprite's tile holds few colours; each frame's tiles the
  cells of what appeared, moved or went, in the view's fixed palette (its
  48 commonest colours exact), the pixels a frame did not change
  transparent (Windows Terminal keeps what lies under them); the picture's
  banner rows written whole in the same palette, so that the two meet
  without a seam. The application (`app/home.go`): the view kept with the
  picture, a frame every 100 ms (BannerTick) while the window has the
  focus (focus reports, `?1004`), the scene kept across a resize at the
  same scale and when the panels grow; a hidden banner keeps no view and
  asks for no frame. The whole drawing's tiles are encoded at once, one
  goroutine each, the banner's together, and `encodeSixel` reads each band
  once (its output the old one's byte for byte, checked on 3,000 random
  images): the whole drawing at 208x59 from 393 to 55 ms. Measured in
  Windows Terminal, maximised at 208x59: with each alien on its own clock
  at 12.5 frames a second, the terminal took 10.9% of a core (14.4 of the
  banner's 25 rows touched a frame); at 10 frames a second, D12's least,
  with the march, 6.6% (WindowsTerminal 5.0%, its OpenConsole 1.6%; 10.4
  rows a frame), the application 4.6% (3.0% on the same screen without a
  banner); without the focus the frames stop (the terminal 0.7%, the
  focus reports CSI O and CSI I received). Timing, on the performance
  counter (Go's clock steps by 0.5 ms here): a banner frame p50 0.46 ms,
  p95 1.27 ms, max 2.6 ms, 8.3 KB; a key in the project field with the
  banner running p95 0.13 ms; the picture on entering 195 ms at worst,
  the 120 ms wait included. Seen in Windows Terminal maximised and at
  150x40, 120x40 (picture) and 100x40 (grid): no seam, no trail after
  25 s. Tests: TestBannerScene (51.6, 1,000 frames of a fixed seed),
  TestBannerFrames (no pixel changed outside the tiles sent; the terminal
  given the frames' images shows the banner drawn whole), TestBannerKept
  (render); TestHomeBanner (51.3: the frames, the second drawing, the
  focus, a hidden banner) and TestHomeSequence's tiles (app); the timing
  step's TestBannerToFrame, TestHomeKeysToFrame and TestHomePictureTime
  (`app/hometiming_test.go`, Windows only for the counter).
- 2026-10-27: step 6 done. ui-design.md's Startup screen rewritten for the
  home screen: its modes and breaks, the picture, the banner, the
  questions, the messages, the projects, the browser, the keys and the
  mouse, what follows and departs from the owner's image, grid mode's
  mock-ups drawn by the renderer; its Animations name the banner.
  reference-design.md: the banner's bands, sparkles, sprites and explosion
  as measured, and its departures (the march, 10 frames a second, the
  crisp sprites, the dithered haze). architecture.md: The home screen, its
  parts and files; the banner's timer and the picture's encoding
  goroutines in Threads of execution. evaluation.md: the timing step's
  sixteen tests and plan 51's three. events-and-state.md: the banner's
  frames among the timers. AGENTS.md: the home screen in its summary
  (20,706 bytes).
- 2026-10-27: step 7 done, [doc/plan51-evaluation.md](doc/plan51-evaluation.md):
  10 PASS, 1 FAIL. 51.1: the project field's box, which D4 puts on the
  terminal row nearest the design's, lies 3.6 screen pixels from the
  design's at 208x59 and 6.5 at 120x40, every other edge within 1.91.
  51.9's copy of the path and its paste back not verified: the antivirus
  quarantined the harness that saved and restored the owner's clipboard
  around them. Mutants: 22 of 23 caught, one equivalent. Waiting for the
  owner's decision on 51.1 and approval.
- 2026-10-27: the owner found the animation too slow. The livelier pace
  (`render/banner.go`): the march every 0.4 s in two bands, the aliens'
  drift and descent about twice as fast, the ship 1.4 times, more shots,
  faster bolts, snappier explosions; chosen among five paces simulated
  through overlays for the rows a frame touches (11.8, from 10.4). A
  banner frame p95 1.66 ms (target 2), 11 KB; Windows Terminal 5.7 to
  8.9% of a core over seven samples (mean 7.75%), the old pace 6.0 to
  8.3% in the same hours, the machine's load swinging from 1 to 74%: the
  8% target is at the edge, recorded as a FAIL in the evaluation's
  addendum, waiting for the owner's decision.
- 2026-10-27: the owner asked for a far faster ship dash, then withdrew it
  and asked for the ship's pace raised again, not drastically. The dash
  and its afterimages are out (the three banner files as before it,
  checked against the pace above); the ship's legs go at 100 to 220
  design pixels a second (70 to 150). 11.8 rows a frame as before, a
  banner frame p95 1.60 ms (target 2), 11 KB; checked in Windows Terminal
  at 208x59, its share of a core not sampled again. The block is in the
  evaluation's addendum.
- 2026-10-27: the owner asked for the ship a little faster still, not
  drastically: its legs at 125 to 275 design pixels a second. 11.6 rows a
  frame; checked in Windows Terminal at 208x59. The timing step missed:
  a banner frame p95 2.05 ms (target 2) with the machine busy (Task
  Manager 45% of a core, Edge 39%); timed in turn, the previous pace
  missed too (p95 up to 2.70 ms), this one no slower. A FAIL in the
  evaluation's addendum, waiting for the owner's decision. The owner
  found this pace right.
- 2026-10-27: the timing step again, the owner's applications idle (Task
  Manager closed; on mains, the CPU 5 to 28% busy just before): all
  sixteen pass. The banner frame p50 0.69 ms, p95 1.76 ms, max 2.80 ms
  (target 2); the home keys p95 0.12 ms; the picture's worst 200 ms
  (target 260); the slowest of the others TestStreamToFrame, p95 4.55 ms
  (target 16). The earlier miss was the machine's load.
- 2026-10-27: Windows Terminal at the final pace, the owner's applications
  idle: 7.54, 7.46 and 7.70% of a core over 60 s (target 8), the
  evaluation's build between them 7.16 and 7.04%. 51.7 passes both its
  parts on a quiet machine; the earlier FAIL blocks stay recorded.
- 2026-10-27: the owner accepted 51.1 as a documented exception: the
  field's box on the terminal row nearest the design's, the path centred
  in it (D4); it stays FAIL in the evaluation, and ui-design.md and
  reference-design.md list it among the departures. Left: 51.9's paste
  back, the owner's check by hand, and the plan's approval.
- 2026-10-27: the owner checked 51.9's paste back by hand: the field's
  path copied from the terminal's selection and pasted back came back
  exactly. Plan 51 waits for the owner's approval.

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

## Plan 43 - A right activity sidebar with the Explorer docked

Brief amendment 44: a narrow activity sidebar right of zone B, from 100
columns, with the Explorer's and Skills' icons after the owner's two
images. The Explorer, a floating window since plan 17, is a panel docked
between zone B and the sidebar where zone B keeps 64 columns, and in zone
B's place where it does not and under 100 columns; F3 and the Explorer's
icon show and hide that one panel, the focus with it, F3 keeping its file
meaning. The icons' four looks are Sixel images, their hover from
every motion reported, filtered before Update; Skills is a placeholder.
The review led to a third patch of the vendored ultraviolet: a row
holding a wide character is written only as far as it changed, so the
icons beside it stay. Approved on 2026-10-25: 15 of 15 PASS
([`doc/plan43-evaluation.md`](doc/plan43-evaluation.md)); commit
48bde52, then the approval commit.

## Plan 44 - Source Control and a Git diff viewer

Brief amendment 45: the sidebar's second icon, Skills' placeholder until
then, is Source Control. Its panel shares the Explorer's docked place and
lists the project's Git changes from one git status read, the Explorer's
too: MERGE CHANGES, CHANGES and STAGED CHANGES, each file with its real
status, a file staged and changed again in both. A file opens its diff in
zone B's place, the panel beside it from 118 columns: git's blobs read raw
by cat-file and the working file, compared by a Myers diff of our own,
unified or side by side, with line numbers, green and red rows and
navigation between changes; read only. The engine was built by an
implementer subagent. Approved on 2026-10-25: 14 of 14 PASS
([`doc/plan44-evaluation.md`](doc/plan44-evaluation.md)); commit
d9d7d83, then the approval commit.

## Plan 45 - Commit, Push and Commit & Push in Source Control

Brief amendment 46: under Source Control's repository row, a message
field, a purple "✓ Commit" button with a separator and a chevron, and a
dropdown with one item, Commit & Push, after the owner's image. Commit
takes the index as staged (asking before staging anything), refuses a
staged set other than the one shown and counts once HEAD is a new commit
on the old one; Push goes to the configured upstream only, never forced;
the main action is read from git's branch line. After the first report
the owner asked for two additions: the staged files' contents compared
too, and a long operation's elapsed time with Esc cancelling a push (a
Ctrl+Break to git's own process group, behind a check of its lock files;
probes P45a to P45e). The engine's first version was built by an
implementer subagent. Approved on 2026-10-25: 19 of 19 PASS
([`doc/plan45-evaluation.md`](doc/plan45-evaluation.md)); commit
f71e022, then the approval commit.

## Plan 46 - Three fixes from the audit

Brief amendment 47: the first three fixes of the read-only audit of
2026-10-25 ([`doc/audit.md`](doc/audit.md)), each test-first.
- The worktree merge reads the main worktree first, refuses a merge in
  progress or uncommitted changes, and aborts only its own conflict
  (MERGE_HEAD the branch's tip).
- A refused Codex `config/read` or `model/list` no longer ends the
  handshake (`event.Problem.Optional`).
- An opened untracked folder in Source Control is read again after each
  status read and each opening.

Approved on 2026-10-25: 10 of 10 PASS
([`doc/plan46-evaluation.md`](doc/plan46-evaluation.md)); commit 9375233,
then the approval commit.

## Plan 47 - The rest of the audit's phase 1

Brief amendment 48: the rest of phase 1 of the audit's roadmap
([`doc/audit.md`](doc/audit.md), Part 2, C), each fix test-first.
- Ctrl+Q while a commit, push or staging runs asks to confirm, naming
  it.
- An answer to a request the CLI withdrew first expires the approval and
  the session goes on (`provider.ErrNoPending`).
- The reducer's Mode case returns, so mode events leave no false note,
  and a mock replay of Codex declines in Codex's word.
- The Explorer opened over the staging question drops it.
- `realPath` strips the `\\?\` prefix; two stale comments are corrected.
- `TestStateMachines`, failing since plan 46's commit, is repaired.

Approved on 2026-10-25: 10 of 10 PASS
([`doc/plan47-evaluation.md`](doc/plan47-evaluation.md)); commit d44bc0f,
then the approval commit.

## Plan 48 - The audit's phase 2: robustness at the boundaries

Brief amendment 49: phase 2 of the audit's roadmap
([`doc/audit.md`](doc/audit.md), Part 2, C), each fix test-first.
- The TIFF and ICO readers are bounded: the tags read from the file when
  used, an icon's PNG size checked before decoding.
- A running command's streamed output keeps its last 64 KiB, the rest
  counted and said.
- The Claude Code reader never writes to stdin.
- Git calls are bounded by `WaitDelay` once git has ended.
- A panic in the dashboard's own goroutines ends its session in order
  (`Fatal`, `Panicked`); fuzz targets for the image readers and both
  mappers.
- Bidi controls and zero-width spaces are drawn as `<U+XXXX>` on the
  approval card.

Approved on 2026-10-25: 13 of 14 PASS; 48.5 FAIL, its prediction about
the old code, accepted as a documented exception
([`doc/plan48-evaluation.md`](doc/plan48-evaluation.md)); commit 067aeb4,
then the approval commit.

## Plan 49 - The rest of phase 0, and phases 3 and 4 of the audit

Brief amendment 50: the rest of phase 0 of the audit's roadmap and its
phases 3 and 4 ([`doc/audit.md`](doc/audit.md), Part 2, C).
- Phase 0: the event queue's and the write queue's tests, and a matrix of
  the ten overlays alone and in their 90 ordered pairs.
- Tool rows memoised and long sessions' scans bounded (`render.Memo`,
  `State.ToolOrder`, `State.ParentRev`, zone A's rows made when shown);
  `TestToolsToFrame` joins the timing step.
- One owner of the overlays (`render.Layers`), the matrix's five defects
  fixed.
- The adapters' shared lifecycle (`pipe.Proc`).
- The git tests in parallel, git's environment set once in `TestMain`.
- Normalised approval decisions (`event.Decision`), each adapter writing
  its CLI's words byte for byte as before.
- Widths by grapheme cluster (`render/cluster.go`).
- A GIF's first frame alone decoded, its other frames walked by their
  blocks.

Approved on 2026-10-25: 11 of 16 PASS; 49.7 and 49.10, short of their
targets, and 49.8, 49.9 and 49.11, on tests that changed how they call
the code, FAIL, accepted as documented exceptions
([`doc/plan49-evaluation.md`](doc/plan49-evaluation.md)); commit 5c06d39,
then the approval commit.

## Plan 50 - Source Control multi-selection

Brief amendment 51: Ctrl+click selects files in Source Control, a "▎" on
each selected row and "Commit N files" on the button, and Commit and
Commit & Push commit them alone, by `git commit --only`: each selected
file staged whole, a partly staged one refused, unstaged and untracked
ones staged only after the question's yes, the rest of the index left as
it is. Windows Terminal passes Ctrl with a click, never Shift. The
owner's question before the approval of the plan: when the commit fails
after the question's staging, the staging stays, as git leaves it, and
the note says so. After the first evaluation (13 of 13 PASS) the owner
asked that a plain click on a file clear the selection, Commit being the
whole index as before with none selected, and that the note say
"changes"; the review of the complete diff then found a file moved on
disk, both its rows selected, refused once staged, fixed by reading
git's status path by path. Approved on 2026-10-26: 14 of 14 PASS
([`doc/plan50-evaluation.md`](doc/plan50-evaluation.md)); commit
b9f8323, then the approval commit.
