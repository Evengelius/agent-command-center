# Plan 37 evaluation (2026-10-24)

The check of plan 37, the effort slider after the owner's images (brief
amendment 38: the selector's effort dropdown replaced by one pill-shaped
slider of discrete levels after the owner's five images, "the
authoritative visual references", and the Effort and Ultracode rows laid
on shared columns). It covers the 9 criteria written in
[`plan.md`, Plan 37](../plan.md) before the code, the slider's image
excepted, written for the probe; design D5 was amended after a burst in
Windows Terminal (plan.md, D5). Nothing is committed: the owner reviews
the result first.

How it was checked:
- The probe first (plan.md, What the probe found): `SliderImage` drawn in
  Windows Terminal 1.23.20211.0 at one row beside each of the owner's
  images scaled to that row (`scratchpad\plan37\probe-wt.png`).
- Deterministic tests, none starting a CLI: `internal/render`
  (TestSliderImage, TestSliderCells, TestSliderText, TestSettingsColumns,
  TestSliderLevels; TestSelectorGeometry and TestSelectorStates moved to
  the shared columns and the slider; TestKeyOverlayRows at 55 rows with
  ← →; TestEffortList gone with the list), `internal/app` (TestSliderKeys,
  TestSliderDraw; TestSelectorKeys, TestSelectorMouse and
  TestSelectorKeepsDraft without the list; plan 36's
  TestSelectorPointerLevels on the slider's cells, the list gone;
  TestSelectorPointerAtRest and TestToggleDraw finding the toggle on its
  own row, the slider's half circles above it), and 15 golden frames. Each
  new check was run once against code broken on purpose, and failed: no
  glide; the waiting level ignored by ← →; the slider's cells on the
  band's ground; the lowest level's sliver of fill; no cells written
  before the image; a hover asking for a level; the glide's clock from the
  first drawing (plan.md, D5).
- Windows Terminal 1.23.20211.0 in mock mode, the build of the final tree
  (`scratchpad\term\acc-p37.exe`, SHA-256 `74134d6b…`), driven by
  `scratchpad\term\p35demo.ps1` (a crop of the two settings rows at every
  shot, a `type` step, a burst of the slider's cells): Claude Code's mock
  (`claude-models.jsonl`, Fable 5.1 at Max, Ultracode on) at 120x40 and
  maximised (188x53 after two Ctrl+minus), Fable 5.1's five levels on and
  off the band and Opus 4.6's four; Codex's mock (`codex-models.jsonl`,
  GPT-6-Sol at High) at 120x40, its six levels brought into force by a
  message each; the glide caught by a burst. No CLI ran: no quota was
  used. The owner's own dashboard, open during the first runs, had been
  closed by the time of the runs reported here.
- Build, vet, gofmt, the whole suite, the race detector on render and
  app, the timing step alone, all on the final tree; no independent
  review, the criteria being settled by the tests, the seven mutation
  checks and the screenshots.

```text
Evaluation - Plan 37, the effort slider after the owner's images - 9 criteria: 9 PASS
PASS 37.1 TestSliderImage: 170x20 pixels; on the ground transparent outside the pill and opaque inside, the last 10 columns transparent; with 5 levels the thumb #FFFFFF at x 10, 45, 80, 115 and 150; at x 25 the track #272B2D on the lowest level and the fill #6784C1 on the second to fourth; no blue at the lowest level's left edge; the lowest level's dot on the fill nearer #95A9D4, the fourth's on the track nearer #697276; at the highest, the gradient at x 5, 72 and 128 within 14 of the fifth image's, and specks; 6 levels: the sixth lit, the fifth blue; 4 levels: the fourth lit, the third blue; held: fill #4B505A, thumb #9AA0AA; a level asked for: a light ring; none known: no white pixel; on the band: #342B47 at the corners and in the last 10 columns
PASS 37.2 TestSliderCells, 120x40 with Sixel: columns W-19 to W-3 of the Effort row blank on no style, the same styles and items with the cursor on the row or not from W-19 to the box's edge; the pill's cells carrying levels 0, 1, 2, 4 at cells 0, 4, 8, 15, the 17th Effort's; Screen.Slider {Shown, X: box+W-19, Y: the row, Look: 5 levels at 2, none asked}, Band with the cursor on the row; none reported without Sixel
PASS 37.3 TestSliderText: the 16 cells at each of 5 levels read as D2 (the thumb over the left end at the lowest, over the right at the highest); at the middle level the ends #6784C1 and #272B2D, the fill's and the track's grounds, their dots, the thumb's halves on the fill and the track; at the highest the gradient's grounds and dots; held: the grey ground, dots and thumb; a level asked for "○"; on the band the ends on the band's ground
PASS 37.4 TestSettingsColumns at 80x24, 120x40 and 200x60, each of Fable 5.1's five levels: "Effort (<level>)" and "Ultracode" at column 3, "Controls reasoning depth" and "Enables dynamic workflows" at column 20, the text slider at W-19 (ending at W-4) and the toggle at W-8 (ending at W-4), the box 66 wide, "Fable 5.1" at column 5; GPT-6-Sol's Effort row the same columns; golden frames: Claude Code's two settings rows changed in place (the list's box already wider), Codex's box one cell wider (65)
PASS 37.5 TestSliderLevels: Fable 5.1 at Max five levels, lit; Opus 4.6 four, lit at Max and not at High; Haiku 4.5 "Effort (N/A)", no slider; GPT-6-Sol six, lit at Ultra and not at Max
PASS 37.6 TestSliderKeys: → from High asks XHigh alone; ← while it waits asks nothing (one Claude Code change at a time); ← from XHigh asks High; → at Max and ← at Low ask nothing; ← on More models closes the selector, asking nothing; a held effort asks nothing and zone B says "the effort is set by CLAUDE_CODE_EFFORT_LEVEL"; with changes waiting for the next turn, → twice asks Max; TestSelectorKeys: Enter on Effort asks nothing; TestSelectorMouse: a click on Effort's label moves the cursor there, one on XHigh's cell asks XHigh
PASS 37.7 TestSliderDraw: the image drawn after the frame, the frame's 17 cells written first; after Max came in force, the first drawing 50 ms after the frame already past 2.5, then 4 to 6 drawings rising to 4, none after; Esc: the place written over with no image, the glide forgotten
PASS 37.8 build, vet, gofmt clean; go test ./... ok (the 15 packages with tests); -race on render and app ok; 15 golden frames changed, only in the selector's box (the settings rows; Codex's box one cell wider), the effort case now ← on Effort (XHigh); the F1 overlay's 55 key rows as ui-design.md's; TestSelectorToFrame p95 1.01 to 1.55 ms (target 16 ms), the other seven timing tests within target
PASS 37.9 WT: Fable 5.1's five levels on the band (Max to Low by ←) and off it (Low to Max), at 120x40 and maximised, each label the level shown (p37-claude-fable-levels.png, p37-claude-maximised.png); Opus 4.6's four (p37-claude-opus46-levels.png); GPT-6-Sol's six in force, XHigh ringed while it waited (p37-codex-gpt6sol-levels.png); both rows' labels, descriptions and controls' right ends on the same columns in every shot; the glide: the thumb at x 180, 153, 143, 139, 138 from Max to XHigh, the fill turning from the gradient to blue with it (p37-glide-frames.png), and 138, 164, 174, 178, 180 back up
```

Not verified:
- A live session: the slider asks for a level through plan 34's
  `ChooseEffort`, which plan 34's live runs checked on Claude Code
  2.1.303 and Codex 0.163.0; no live run was made for this change.
- A Claude Code level ringed while its change waits: the mock answers at
  once, so only the test shows it; Codex's ring is in the screenshots.
- The text slider on a 16-colour terminal, which has no ground for its
  track and fill: no test.
- The label leads the thumb by the image's 50 ms after the frame (and,
  live, by the CLI's answer before the frame): the glide's clock runs from
  the frame, so the thumb's first drawing is already more than half way.
- Other terminals than Windows Terminal 1.23.

The owner's text asks for colour that grows with the level ("High effort:
stronger blue/lavender fill"); the owner's images 2 to 4 have one fill
colour, `#6784C1`, and grow its length, as their captions say
("Approximately half the track is filled", "Most of the track becomes
filled"). The slider follows the images, the authoritative
references, and keeps the distinctive gradient for the highest level.
