# Plan 43 evaluation (2026-10-25)

The check of plan 43, a right activity sidebar with the Explorer docked
(brief amendment 44: a narrow sidebar right of zone B with the Explorer's and
Skills' icons, the existing Explorer docked between zone B and the sidebar,
F3 and the Explorer's icon showing and hiding the same panel, the icons'
inactive, hover and selected looks after the owner's two images, Skills a
placeholder). It covers the 15 criteria of [`plan.md`, Plan 43](../plan.md):
14 written before the code, five of them amended in step 4 after the review
and said where (43.6, 43.8, 43.9, 43.10, 43.11), and 43.15 added in step 4.
Nothing is committed: the owner reviews the screenshots and the results first.

How it was checked:
- The finding from the code and the two images, measured from their pixels
  (plan.md, What was found; reference-design.md, The right sidebar).
- Deterministic tests, none starting a CLI. The render's layout, panel and
  tiles: TestSidebarLayout, TestLayoutWidths (every width from 80 to 240
  columns), TestSidebarTiles, TestSideTileCells, TestSideImages,
  TestSideSixel, TestExplorerPanel, TestExplorerNames. The application,
  with keys and mouse messages through the program's update and its filter:
  TestExplorerToggle, TestSkillsTile, TestExplorerF3, TestExplorerDocked,
  TestExplorerContent, TestExplorerNotWindow, TestExplorerFocus,
  TestPopupsInsideB, TestSideHover, TestMotionFilter,
  TestMotionFilterProgram, TestSideMotionMode, TestSideDraw,
  TestCompactFocus; after the review, TestExplorerDraftKeys,
  TestExplorerF2, TestExplorerHiddenB, TestExplorerZoomF3 and
  TestSideHoverRelease. Plan 17's Explorer tests (TestExplorerOpen,
  TestExplorerKeys, TestExplorerBack, TestExplorerMarks, TestExplorerStale,
  TestExplorerDeleted, TestExplorerMouse) were adapted to the docked panel.
- The 66 golden frames that changed, each checked against HEAD by a script
  (`scratchpad\plan43\check-goldens-sixel.mjs`). Zone A's columns and zone
  C's rows must be identical, and so must the bars and, unless the panel
  has the focus, the footer; only zone B's columns, the panel's and the
  sidebar's may differ. Under 100 columns a frame may change only where the
  Explorer shows.
- An independent review in three lenses (layout, interaction, pointer), each
  probing a scratch copy of the tree, with its one medium finding
  verified by a fourth agent (step 4). The verification refuted that
  finding, the tiles' images redrawn ten times a second in a running turn,
  as deliberate and cheap. It found instead the erasure D11 fixes: the rows
  beside the tiles holding a wide character were rewritten by every frame.
  The renderer's patch (`third_party/ultraviolet/PATCHES.md`, patch 3) has
  its own test, TestPatchedDriftRows; TestWideRowsKept runs the real
  program on a list of ✅ beside the tiles; the module's upstream tests ran
  on a scratch copy with the three patched files.
- Each new behaviour broken on purpose, through `go test -overlay`, nothing
  written in the repository (`scratchpad\plan43\mutate3.mjs` and
  `mutate-overlay.mjs`).
- Build, vet, gofmt, the whole suite, the race detector on render, app and
  input, the timing step alone, all on the final tree.
- Windows Terminal 1.23 in mock mode, real keys and clicks (`Term.cs`,
  SendInput), on `testdata\mock\claude-thinking-subagent.jsonl` with the
  repository's own tree (a `git archive` of HEAD) as the project, so the
  Explorer lists it. The final build (`scratchpad\term\acc-p43s3.exe`,
  SHA-256 aa5b697f85f30cc5…, built from the final sources) maximised in the
  acc tab's profile (208x59), and at 120x40 in the default profile (the acc
  profile's 9 pt font gives 160x54 there). Before and after D11: step 3's
  build (`acc-p43s2.exe`, 2b7f4036f8065a0d…) and the final one on a mock
  whose answer is 70 lines of ✅. The owner's clipboard was read before the
  runs and found unchanged after. The owner's terminal-hosted Codex session
  (the dashboard started with `go run` at 18:22, idle) stayed open: the
  auto mode classifier refused to stop it, and the runs replay mocks only,
  touching no quota and no CLI's configuration.

```text
Evaluation - Plan 43, a right activity sidebar with the Explorer docked - 15 criteria: 15 PASS
PASS 43.1 TestSidebarLayout at 120x40, 156x44, 200x60 and 208x59: the sidebar 8 columns on the upper rows only, right of zone B, zone A as before (36, 44, 56, 58 columns, the same rows), zone B w - A - 8 (76, 104, 136, 142), zone C full width and as before, the footer last; the 66 golden frames' check: zone A's and zone C's cells identical to HEAD's; Windows Terminal at 208x59 and 120x40: the sidebar's box right of zone B, its two icons, zone C under all of it
PASS 43.2 TestSidebarLayout: the panel 26, 30 and 31 columns at 156x44, 200x60 and 208x59, zone B 78, 106 and 111; at 120x40 in zone B's place (zone B would keep 50), the sidebar kept; at 80x24 no sidebar, the panel in zone B's place; TestLayoutWidths from 80 to 240 columns, the panel shown and hidden: no zone under 5 columns, every row exactly the screen's width; golden frames explorer at 80x24, 120x40 and 200x60; Windows Terminal: the panel docked at 208x59, in zone B's place at 120x40
PASS 43.3 TestSidebarTiles: exactly two tiles, the Explorer's then Skills', 6x3 cells on the box's rows 1-3 and 5-7, centred, the text form in each of the four looks ("≡" and "⑂", "▎" when selected, the grounds); TestSideTileCells: with Sixel, blank cells that stay the same in every look, the tiles' places recorded only when whole on screen; TestSideDraw: each image drawn over its cells written first
PASS 43.4 TestSideImages, from the images' pixels: 60x60; selected, the fill #181C30 inside the #7A4EBA outline, the #A272F1 bar on the left edge, the icon #A6B3C6; hover, #342B47 with no outline, the icon #B9C3D3, brighter than the inactive #6F7C88; selected and hovered, #20253D inside #8C5DD0; inactive, transparent but for the icon; rounded corners transparent; the Explorer's trunk, crossbar, foot and two folders, Skills' three nodes and their links, each within 6 levels; read back from Windows Terminal's screenshots: the fills, the bar and the icons within 1 level (#171C30 x1301, #A373F2, #A6B3C7, #332B47, #BAC2D4, #707D87), the outline within 9 (#7B57BA, the terminal scaling the 1.4-pixel line)
PASS 43.5 TestSideHover at 120x40 and 200x60: a motion onto a tile hovers it, off it unhovers it, the 18 reports within one tile all dropped before Update; Skills hovered with the Explorer selected keeps it selected, the panel shown, the focus on it; TestMotionFilter and TestMotionFilterProgram (the real program): a motion that changes no hover reaches neither Update nor View, counted, every other message passes; TestSideMotionMode: every motion reported while the sidebar shows; cell motion under 100 columns, with the viewer or the draft window open, zone C zoomed or the keys' overlay shown; none with F12; TestSideHoverRelease: a press on a tile, the button held off it unhovers it, a release on Skills hovers Skills; TestMotionToFrame, the timing step: hover p95 1.50 to 1.80 ms (target 16 ms) at 120x40 and 208x59, the panel hidden and shown, and 800 of 800 idle reports dropped in each case
PASS 43.6 TestExplorerToggle at 200x60: F3 and a click on the Explorer's tile show the panel (focus R, the tile selected, zone B 106) and hide it (focus B, zone B 136), the panel's ⌄ hides it, F3 and the tile leave the same state; TestSkillsTile: a click on Skills changes nothing (frame, focus, selection, draft, popups, no command), the panel shown or not, a popup open or not; TestExplorerF3: F3's file meaning kept on a focused approval, an entry shown whole and the entry cursor on a call naming a file; from the viewer and the quick open, F3 closes them and shows or hides the panel; from the inspection and the details view it closes them and shows it; TestExplorerZoomF3 (added in step 4): from a zoomed zone, the zoom ended and the panel shown with the focus, shown before or not; Windows Terminal: the tile's click hides and shows the panel, F3 hides it and shows it again
PASS 43.7 TestExplorerPanel: the header "EXPLORER" in the title colour, bold, and "⌄" at its right end; the rule; the root row with the project folder's name; the tree one step under it, "›" and "⌄", the folder and file icons and names in their colours, git's marks; scrolling keeps the header, rule and root row; TestExplorerContent on a project read from the disk; TestExplorerNames: the names' colours; Windows Terminal: the repository's own tree under "proj", the names #A0ABD8 and #9EAAD5 read back exactly
PASS 43.8 Plan 17's tests against the docked panel: TestExplorerOpen, TestExplorerKeys, TestExplorerBack, TestExplorerMarks, TestExplorerStale, TestExplorerDeleted, TestExplorerMouse; TestExplorerDocked: Esc on a top-level row gives zone B the focus with the panel shown, the wheel over zone B leaves the panel, the folders, cursor and marks the same after hiding and showing; TestCompactFocus (added in step 3): where the panel takes zone B's place, zone B taking the focus hides it; TestExplorerF2 and TestExplorerKeys (added in step 4): F2 from the panel, the viewer and the quick open leaves the panel shown
PASS 43.9 TestExplorerNotWindow at 200x60: with the panel shown, F4 opens the modes selector and Alt+P the model selector from the panel's focus, a click the "+" menu, a paste reaches the draft; the draft, its image and zone B's held scroll the same before, while shown and after; TestExplorerDraftKeys (added in step 4): from the docked panel's focus a paste reaches the draft, Ctrl+C copies the draft's selection ("again") with the turn still running, Alt+V asks for the clipboard's images; at 120x40, the panel in zone B's place, a paste is dropped and Ctrl+C interrupts
PASS 43.10 TestExplorerFocus: F6 C, A, B with the panel hidden; with it shown F6 C, A, B, R, Shift+F6 B, A, C, R, Tab C, A, Shift+Tab C, R; the panel's keys in the footer while it has the focus; "F3 Explorer" in the footer and F1; TestExplorerF2 and TestExplorerHiddenB (added in step 4): "◆ approval waiting · F2 to answer" first in the footer while the panel has the focus at 200x60 and 120x40, and in zone B's place whatever the focus; there the quit question too, before it, and Esc to stay keeps the panel; TestKeyOverlayRows: ui-design.md's key tables and the F1 page agree
PASS 43.11 TestPopupsInsideB at 100x40, 105x40, 112x40, 120x40, 136x40, 156x44, 200x60 and 208x59, the panel shown and hidden (13 cases): the model selector, the modes selector and the "+" menu end at zone B's right edge or before it, one no wider than zone B starts at its left edge or after it, a wider one ends at its right edge (amended in step 4: the model selector at 100x40 and beside the panel docked at 136x40, a case the test keeps); at 120x40 with the panel in zone B's place none opens
PASS 43.12 66 golden frames changed, 0 new: 66 PASS by the script, zone A's and zone C's cells identical to HEAD's, zone B narrower with its content reflowed, the panel's and the sidebar's columns new; under 100 columns only explorer-80x24 and file-finder-80x24 changed, both showing the Explorer (in the second, the quick open over it)
PASS 43.13 Build, vet, gofmt clean; go test ./... ok, 15 packages with tests, 719 tests (28 new), the vendored module's two (TestPatchedDriftRows new) run by internal/licenses; -race on render, app and input ok (95 s, 311 s, 42 s), 0 data races; the timing step alone, all nine within target: the 10,000-line draft's keys p95 1.11 ms, the draft window's 1.52 ms, the 50,000-line draft's 3.08 ms, the selector's 1.01 to 1.59 ms, the hover's 1.50 to 1.80 ms, the long session's delta 4.81 ms, the stream's 4.41 ms, the thinking card's 3.53 ms (target 16 ms), the 50,000-line paste 1.80 ms at most (target 100 ms); 51 of 52 breakages caught by a test: step 4's 13 (the review's fixes and D11's patch), stage 1's 13 and stage 2's 25; one survives as equivalent (the size dropped from the tiles' redraw signature: since step 4 every size report draws them again), and one stage 1 breakage no longer applies (the sidebar's cache key, rewritten in stage 2)
PASS 43.14 Windows Terminal, the final build, 208x59 in the acc tab's profile and 120x40, real keys and clicks: the collapsed sidebar, both icons inactive; each tile hovered; F3 showing the panel, the Explorer selected; the Explorer hovered while selected, Skills hovered while the Explorer is selected; the tile's click hiding and showing the panel; F3 hiding and showing it again; at 120x40 the panel in zone B's place; the colours read back as 43.4 and 43.7 say, against the references' (reference-design.md, The right sidebar); the clipboard unchanged
PASS 43.15 (added in step 4) TestWideRowsKept, the real program at 208x59 with Sixel, a list of 70 ✅ lines, the six rows beside the tiles each holding one: a key typed once the turn is over, and 1.2 s of a running turn's spinner frames, write none of those rows (with the unpatched renderer through -overlay: the key's frame and 24 spinner frames rewrote them); TestPatchedDriftRows, with and without grapheme widths: an unchanged row written not at all, a change written from column 0 to its last cell with no erase, the model following, a wide cell at the change whole, a resize repainting whole (on the unpatched file it fails four checks on each path); the module's upstream tests pass with the three patches; TestSideDraw: a size report of the same size draws the tiles again; Windows Terminal: after a typed key beside the ✅ list, step 3's build lost both tiles and the final build kept them
```

Observed and kept:
- **The collapsed sidebar shows the Explorer's tile inactive.** Reference 1
  shows it selected; the brief's text ties the selected look to the Explorer
  being active ("F3 when collapsed … Mark Explorer as active"; "Explorer
  hovered" and "Explorer selected" listed as separate states), so the tile
  is selected only while the panel shows.
- **The tiles are smaller in the sidebar than the references'.** Measured
  at the same box width, the reference's selected tile nearly fills its
  sidebar (49 of about 51 pixels) where the terminal's fills about two
  thirds of it. The icon keeps the reference's ratio to its tile (0.38
  against 0.39). The rest is the cell. The reference's cells are about 6.6
  by 15.7 pixels, the terminal's 9 by 18.4, so the reference's sidebar is
  about 6.6 of the terminal's columns. Its border hugs the tile, where a
  box drawn in cells keeps a whole cell each side. A closer match would take
  a 7-column sidebar whose images cover the border cells: a change of
  design D1 and of every sidebar's golden frame, left to the owner.
- The header "EXPLORER" is in the dashboard's title colour, #C4B5FD, as zone
  B's title; the images' titles are #A785DD. The panel's icons are Segoe UI
  Symbol's folder and page, thinner than the images' at 9 points; their
  brightest pixels read #50759E and #6E7786 for the #6CA1DB and #A3B1C9 set.
- A popup wider than zone B, the model selector at 100 to 109 columns and
  beside a panel docked at 136 to 138, extends over zone A, as it did at
  100 and 101 columns before: narrowing it would cut its Effort row.
- While the sidebar shows, every motion is reported, and the frames that
  change a tile's rows draw its image again, even when the renderer did not
  touch it: about 3 KB a second in a running turn, measured by the
  verification (0.3% of a core). It is the rule every image follows since
  plan 32, and narrowing it is safe only with D11's patch; not changed.
- `ui-design.md`'s layout frames, written by TestDocFrames, were drawn
  again; the old ones showed a model pill "Haiku 5.5 High · Ultracode" from
  a recording that names no effort, and rows of plan 26's heights (33 and
  49 where plan 27 made them 25 and 41).
- During the final checks, a mutation run that wrote its mutants into the
  tree was cut short when the session was stopped. Nine source files came
  back as NUL bytes. They were rebuilt from the review's snapshot copies and
  from the overlay mutants' copies with their mutations reversed, every
  source of a file agreeing. The tree's `git diff` hash then equalled the
  one recorded before that run (5933249ea88eab6c), and the build and the
  suite passed. Go's build cache held a zero-filled entry from the same stop
  ("can't find export data (bufio: buffer full)" on one overlay build); it
  was cleaned and the build and the suite run again from scratch. The stage
  1 and stage 2 mutation lists were then run through overlays.

Not verified:
- A live session: the sidebar and the panel were checked on recordings, the
  layout and the mouse being the same whatever the CLI; a live turn would
  need a model call.
- A terminal without Sixel: the text form is checked by the golden frames
  and TestSidebarTiles, not on screen.
- Erasure beside a wide character in a running turn, on screen: the
  Windows Terminal check typed a key on an idle session; the running
  turn's case is checked by TestWideRowsKept on the real program's output.

**The commit and the push (2026-10-25), after the owner's approval.** The
owner approved plan 43 on 2026-10-25; commit 48bde52 (132 files: 122
modified, 10 added) and the approval commit were pushed to `origin`;
`origin/main` equals `main`, the working tree is clean, and the author is
`Evengelius <molimoabel@gmail.com>`, with no attribution line: PASS. The
blocks above stay as they were reported.
