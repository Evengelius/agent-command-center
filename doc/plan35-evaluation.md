# Plan 35 evaluation (2026-10-24)

The check of plan 35, the Ultracode toggle after the owner's image (brief
amendment 36: "The pill design for ultracode is not respected", with an
image of the toggle, a lavender pill with a round white knob). It covers
the 8 criteria of [`plan.md`, Plan 35](../plan.md): 1 to 7 written before
the code, 1 to 3 amended and 8 added after the burst measurement found
the first design blinking (plan.md, What the burst found). Nothing is
committed: the owner reviews the implementation and the screenshots
first.

How it was checked:
- Deterministic tests, none starting a CLI: `internal/render`
  (TestToggleImage, TestToggleCells, TestToggleText; TestSelectorGeometry
  and TestSelectorStates updated for the text toggle), `internal/app`
  (TestToggleDraw), and the golden frames models-claude,
  models-claude-more and models-claude-effort at 80x24, 120x40 and
  200x60, updated and reviewed. Each check was run once against code
  broken on purpose, and failed: no toggle drawn (toggle_test.go:58,
  "opened: the drawing []"), the toggle not recorded as drawn
  (toggle_test.go:84, nothing asked when the selector closed), its cells
  on the band's ground under the band (TestToggleCells), and no cells
  written before the image (toggle_test.go:58).
- Windows Terminal 1.23.20211.0 in mock mode
  (`testdata/mock/claude-models.jsonl`, Ultracode on, a change answered
  at once): the executable built from the final tree
  (`scratchpad\term\acc-p35.exe`, SHA-256 `1d0d7314…`), driven by
  `scratchpad\term\p35demo.ps1` (plan 34's harness, plus a crop of the
  toggle's place at every shot, a click on the toggle's own cells, and a
  burst of captures of its cells every 16 ms around a key), with a
  temporary home and app data and no CLI on the PATH, maximised (188x53
  after two Ctrl+minus, cells of 10x19 pixels) and at 120x40 (12x23).
  The harness places the toggle from the header's "CLAUDE MODELS" and
  "Current session", whose left edges UI Automation gives exactly (its
  widths are a few pixels too wide, which misplaced a first burst). No
  terminal-hosted Claude Code or Codex session was open, so none was
  closed.
- No live run: the change is drawing only, from the state the session
  reports (plan 34's live runs); no quota was used.
- Build, vet, gofmt, the whole suite, the race detector on render and
  app, the timing step alone; no independent review, the criteria being
  settled by the tests, the four mutation checks and the measurements.

```text
Evaluation - Plan 35, the Ultracode toggle after the owner's image - 8 criteria: 8 PASS
PASS 35.1 TestToggleImage: 50x20 pixels on, off and unavailable; on the ground, every pixel within 9.3 of the track's middle line opaque and every one from 10.7 transparent, the last 10 columns transparent; on the band, none transparent, the pill's four corners and the last 10 columns #342B47; the track's middle #CA84FB, #3B3F51, #2A2D38; the knob's centre #FFFFFF at x 30 (on), #FFFFFF and #5E616D at x 10 (off, unavailable), y 10; 16 pixels of the knob along its middle row; ToggleSixel is the image encoded
PASS 35.2 TestToggleCells, 120x40 with Sixel: columns W-7 to W-3 of the Ultracode row are blank cells of panel 'S', item ItemUltracode, with no style, on, on with the cursor on the row, off and unavailable; from W-7 to the box's edge the same styles and items with the cursor on the row or not; Screen.Toggle {Shown, X: box+W-7, Y: the row, On, Usable, Band} in each state; no "●", "▐", "▌" or half circle on the row; none reported without Sixel
PASS 35.3 TestToggleDraw: after Sixel's answer and Alt+P, one drawing asked: ESC[row;colH, the frame's 5 cells, ESC[row;colH and ToggleSixel(on) at the toggle's first cell; none after ↓ to Sonnet 5.5 (its row as it was); one after the cursor reached the row (the band's image) and one after Ultracode went off (the off image on the band), each the cells then the image; Esc: one drawing, the frame's cells at the toggle's place and no image; none after, no image left
PASS 35.4 TestToggleText, without Sixel: columns W-8 to W-4 read U+E0B6, space, U+E0B6, U+E0B4, U+E0B4 on, and U+E0B6, U+E0B6, U+E0B4, space, U+E0B4 off and unavailable; the ends #CA84FB, #3B3F51, #2A2D38 on the ground, on the band's ground with the cursor on the row; the knob white on the track (#5E616D on #2A2D38 unavailable); the gap on the track's ground; the golden frames show it on the Ultracode row
PASS 35.5 TestToggleDraw: with Sixel, a click on the toggle's second blank cell asked for Ultracode off; without Sixel, the text toggle's left end one column left of the image's place, and a click on its knob asked for Ultracode on; WT, both runs: a click on the image's second cell turned it on (onclick-zoom.png)
PASS 35.6 build, vet, gofmt clean; go test ./... ok (the 15 packages with tests); -race on render and app ok; 9 golden frames changed, one line each, the Ultracode row: "▐   ●▌" became the text toggle at W-8 to W-4, its right end at the same column; TestSelectorToFrame p95 1.01 to 1.54 ms (target 16 ms), the other seven timing tests within target
PASS 35.7 WT 188x53 and 120x40: on, the lavender pill with its white knob at the right (open-zoom.png); on the band (band-zoom.png, onclick-zoom.png, hover-zoom.png), the band running through the image's last cell to W-3 as on the model rows; off on the band and off (offband-zoom.png, off-zoom.png), the grey track with the knob at the left; reopened, on (reopen-zoom.png); the ends half circles, the knob round with a margin, 2:1 in the image and 2.1:1 on screen, 48x23 and 40x19 pixels (the owner's 1.9:1), its right end at W-4; after Esc, at the toggle's 5 cells, 0 white and 0 lavender pixels of 950 (maximised) and of 1,380 (120x40), against 172 and 476, 234 and 640 while open
PASS 35.8 WT burst at 120x40, 40 captures each, the key after the third: the cursor onto the row, the pill in every capture, the band's image from 146 ms; Ultracode off, the on image then the off image from 147 ms, none between; the cursor off the row, the pill in every capture; with the first design the same bursts lost the pill for 2 to 3 captures onto and off the row (plan.md, What the burst found)
```

Not verified:
- A live session: the toggle draws the state Claude Code reports, which
  plan 34's live runs checked; no live run was made for a change of
  drawing only.
- Terminals other than Windows Terminal 1.23, the only one the owner
  uses; one that does not answer the device attributes request with
  Sixel gets the text toggle (criterion 4).
- The text toggle on 16 colours, which has no ground for its track (its
  ends alone): no test.
- The band's colour around the pill follows the rest of the row by one
  drawing, about 50 ms, when the cursor comes onto the row or leaves it
  (the bursts' captures 5 to 7): seen, part of the design, not a
  criterion.

Seen outside the plan: in the last run at 120x40, Alt+P reopened the
selector with its cursor on the Ultracode row, under the pointer the
click on the toggle had left there, not on the model in force
(reopen.txt: no model marked "▶"; the band on the Ultracode row); the
maximised run, and the run before at each size, opened on Fable 5.1.
Plan 34's hover moves the cursor to the row under the pointer, most
likely here on a motion report Windows Terminal sent for the resting
pointer once the selector turned motion tracking on (not verified).
Enter would then turn Ultracode off rather than act on the model.
Reported to the owner; nothing changed.

**The commit and the push (2026-10-24), after the owner's approval.** The
owner approved plan 35 on 2026-10-24; commit b43cf46 (24 files: 20
modified, 4 added) and the approval commit were pushed to `origin`;
`origin/main` equals `main`, the working tree is clean, and the author is
`Evengelius <molimoabel@gmail.com>`, with no attribution line: PASS. The
blocks above stay as they were reported.
