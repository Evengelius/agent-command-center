# Plan 30 evaluation (2026-10-22)

The check of plan 30, the "+" menu after the owner's image (brief
amendment 30, `references/09-plus-menu.png`: "clicking on "+" must trigger
exactly what you see on image", "Implement that design first"). It covers
the criteria written in [`plan.md`, Plan 30](../plan.md) before the code,
30.1 and 30.3 corrected with a note as the work went. Nothing is
committed: the owner reviews the screenshots first.

How it was checked:
- Deterministic tests: `internal/render` (`plusmenu_test.go`: TestPlusMenu
  new) and `internal/app` (`plusmenu_test.go`: TestPlusMenuMouse new;
  `golden_test.go`: the plus-menu scenario, a click on the band and the
  pointer over the item).
- The golden frames: four added for the scenario, every other frame
  unchanged (`git status` under `testdata/frames`).
- Windows Terminal, the executable built from this tree
  (`scratchpad\term\acc-p30.exe`), the plan 24 harness as
  `scratchpad\term\p30demo.ps1` with two new steps, `click` (a left click
  one cell into a text found on screen through UI Automation) and `hover`
  (real pointer moves onto a text, by SendInput: the harness's
  SetCursorPos sends no motion to the terminal, so the first run showed
  no hover); the quota mock maximised at 9 pt (208x59) and at 120x40, a
  temporary home, no agent CLI on the PATH, in mock mode; the screenshots
  in `scratchpad\term\out-p30\`, their pixels read for the grounds. The
  owner's Windows Terminal window stayed as it was.
- Build, vet, gofmt, the whole suite, the race detector on the render and
  app packages, then the timing step alone (`scratchpad\plan24\p30-checks.log`).

```text
Evaluation - Plan 30, the "+" menu after the owner's image - 6 criteria: 6 PASS, 0 FAIL
PASS 30.1 TestPlusMenu on the quota mock at 80x24, 120x40 and 208x59, the menu open: the three rows above the editor's rule read "╭" + 26 "─" + "╮", "│  🗀  Upload from computer │" and "╰" + 26 "─" + "╯" from the band's cell, the box in stFrame, the item in stThinkEdge; the band " + " on the active ground (active(stThinkEdge)); hovered, the item on the active ground, not hovered on none; the band's cells answer '+' and the box's 'U' to PanelAt; closed, no box and the band on the tool rows' ground; every other cell equal between the open and the closed frames; the folder icon one cell by the width table; on the 16-colour palette the band in stThinkEdge alone
PASS 30.2 TestPlusMenuMouse: a click on the band opens the menu ("Upload from computer" on screen, the view's mouse mode all motion); a motion over the item sets the hover, off it clears it; a click on the item closes the menu with the draft "draft", the focus B and the sent messages unchanged, the mouse mode cell motion again; a second click opens, Esc closes with the draft unchanged; open, a click on zone C closes it and focuses C; open, typing "a" closes it and the draft reads "drafta"
PASS 30.3 Four golden frames added, plus-menu at 79x24 (too small), 80x24, 120x40 and 200x60, the menu open and hovered; git status lists no other frame
PASS 30.4 go build ./..., go vet ./... and gofmt -l . clean; go test ./... 14 packages ok; the race run on internal/render and internal/app ok, no race; the timing step alone 7 of 7: TestKeystrokeToFrame p95 1.11 ms (1.50 ms in the draft window), TestDraftKeystrokes 2.12 ms, TestKeystrokeCost 238 µs and 500 µs a key, TestLongSession delta-to-frame p95 5.15 ms after 26,209 events, TestPasteToFrame p50 1.05 ms, TestStreamToFrame p95 4.05 ms, TestThinkingFrame p95 4.26 ms, every target p95 ≤ 16 ms
PASS 30.5 Windows Terminal: quota-max open (15.7 s, the click on the band) and hover (16.3 s, the pointer on the item), quota-120x40 open (15.3 s) and hover (15.9 s): the box with "🗀  Upload from computer" above the editor's rule, the band's ground #272745 in both screenshots, the item row's ground #272745 in hover and the background #0C0C0C in open, the item's text #EDE9FE; the conformance table below, 6 rows PASS and 1 documented deviation (the tail); the owner reviews the screenshots before any commit
PASS 30.6 ui-design.md (Input editor: the row's "+", the "+" menu bullet; Focus and keys: Esc), reference-design.md (the image's row and section), brief.md (amendment 30), AGENTS.md (status), plan.md (D1 to D5, 30.1 to 30.6, progress); every changed text file read byte by byte: no BOM, no CR; git status lists the intended files only
Not verified: the owner's review of the screenshots; the hover with the owner's own mouse (the harness moves the pointer by SendInput, which the terminal reports as the owner's moves)
Verdict: DONE
```

**Conformance with the image** (`references/09-plus-menu.png`), read
against `out-p30\quota-max\hover.png`, `open.png` and
`out-p30\quota-120x40\hover.png`:

| The image | The dashboard | Result |
| - | - | - |
| The "+" box on an indigo background (`#242D52`), its "+" in lavender-white, while the menu is open | the band " + " on the active ground, `#272745` by the screenshot's pixels, the "+" in the thinking edge's lavender-white (`#EDE9FE`) | PASS |
| A rounded box above the "+", a grey border (`#333447`), a dark fill | a three-row box above the editor's rule, its left edge on the band's cell, in the frame's colour (`#3B3F51`); the fill is the background | PASS, the fill |
| A small tail pointing down from the box at the "+" | none: a one-cell grid has no such shape; the box stands two rows above the band, its left edge on the band's | deviation, documented |
| A folder outline icon, then "Upload from computer" in lavender-white (`#D9D5FC`) | "🗀  Upload from computer" in `#EDE9FE`, the icon drawn over two cells by Windows Terminal with one visible space after it | PASS |
| The item on an indigo highlight (`#272745`) under the mouse | the item's inner row on the active ground while the pointer is over it (`#272745` in hover.png, the background in open.png) | PASS |
| The menu opens on a click on the "+" | a click on the band opens it; a second click, Esc, any key or a click elsewhere closes it; a click on the item closes it and does nothing else | PASS |
| Nothing uploads yet: design first | no key, click, event or state behind the item; the upload is a later plan's | PASS |

**Evidence outside the repository**, in the session's scratchpad:
`term\out-p30\` (the screenshots open and hover at both sizes, their UI
Automation rows and `log.txt`), `term\p30demo.ps1` and `term\Term.cs` (its
Glide made public for the hover step), `term\acc-p30.exe`,
`plan24\p30-checks.log`, `plan24\harness30.cjs`, the patch files
`p30-open.patch`, `p30-crit1.patch`, `p30-crit3.patch`, `agents30.patch`,
`amendment30.md`.
