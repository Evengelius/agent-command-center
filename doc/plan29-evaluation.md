# Plan 29 evaluation (2026-10-22)

The check of plan 29, the input editor's row after the owner's image
(brief amendment 29, `references/08-input-editor.png`: the helper line
"replaced by the design shown in the image", design only, no function
yet). It covers the criteria written in [`plan.md`, Plan 29](../plan.md)
before the code. Nothing is committed: the owner reviews the screenshots
first.

How it was checked:
- Deterministic tests: `internal/render` (`editorrow_test.go`:
  TestEditorRow new; `layout_test.go`'s TestCompactEditor and
  `thinking_test.go`'s TestThinkingSizes adapted) and `internal/app`
  (`editing_test.go`, `draftwin_test.go`, `keysconfig_test.go` adapted).
- Every golden frame regenerated and reviewed by script
  (`scratchpad\plan24\framecheck29.cjs`): 59 changed; 42 with the rows
  outside zone B identical to HEAD's, the editor's separator and row, the
  draft rows unchanged and the conversation HEAD's less one row; 7 whose
  conversation re-flowed with its room, read by eye (the live thinking
  card's form, two rows at 80x24 and five at 120x40 with an agent in
  play; a turn rule folded); 10 with a window over the dashboard, not
  judged.
- Windows Terminal, the executable built from this tree
  (`scratchpad\term\acc-p29.exe`), the plan 24 harness as
  `scratchpad\term\p29demo.ps1`, in mock mode with a temporary home and no
  agent CLI on the PATH: the quota mock maximised at 9 pt (208x59) and at
  120x40, the waiting copy of P3b at 208x59; the screenshots in
  `scratchpad\term\out-p29\`. The owner's Windows Terminal window stayed
  as it was, as for plans 27 and 28.
- Build, vet, gofmt, the whole suite, the race detector on the render and
  app packages, then the timing step alone (`scratchpad\plan24\p29-checks.log`).

```text
Evaluation - Plan 29, the input editor's row after the owner's image - 7 criteria: 7 PASS, 0 FAIL
PASS 29.1 TestEditorRow on the quota mock with its notices cleared, at 80x24, 120x40 and 208x59: zone B's last four inner rows are a rule, the prompt row ("> ", the cursor, "Ask Mock data anything…"), a rule in stFrame, and "  +     Haiku 5.5 High · Ultracode " then spaces then "↯ Auto" at the end, nothing else; the cells of " + " and of the pill on the ground and no other; " + " and " Haiku 5.5 High" in tint(stText), " · " in tint(stMdBullet), "Ultracode " in tint(stYou), "↯ Auto " in stText; a 12-line draft puts "12 lines" in stGrey after "Ultracode    "; the turn running puts "turn running: the draft waits · Ctrl+C interrupt" there at 208x59, "2 lines · turn running…" with a two-line draft, and alone on the row under the separator at 120x40; a pasting draft "pasting…"; the keys problem "keys.json: bad" alone in stFailed; a 40-line draft at 120x40 and 200x60 leaves the conversation at least half of B's inner rows
PASS 29.2 TestCompactEditor at 80x24, 120x40 and 200x60: one input row for an empty or one-line draft, three for a 3-line draft, the separator and the row under them, no empty row between; the row with the pills and "↯ Auto", or its middle alone where that does not fit (80x24 with the count); "3 lines" on it for the 3-line draft and no "line" for the others
PASS 29.3 TestEditorDisplay ("2 lines" on the row, "Ctrl+G draft" in the footer), TestDraftWindow (the footer names the window; its own footer "Enter send · Ctrl+J newline · Ctrl+F full screen" unchanged), the quit test (the quit question on the row), TestConfiguredKeysShown (the configured keys in the footer and on the F1 page, none on the row), TestEditorMouse and the select, pane and key tests pass; TestThinkingSizes pins the two-row card at 80x24 (room 11) and the seven-row one at 120x40 without agents
PASS 29.4 59 golden frames regenerated; framecheck29.cjs: 42 ok, 0 bad, 17 other (7 re-flowed with the room, read by eye; 10 with a window over the dashboard); TestDocFrames redrew ui-design.md's three Layouts frames
PASS 29.5 go build ./..., go vet ./... and gofmt -l . clean; go test ./... 14 packages ok; the race run on internal/render and internal/app ok, no race; the timing step alone 7 of 7: TestKeystrokeToFrame p95 1.08 ms (1.50 ms in the draft window), TestDraftKeystrokes 2.19 ms, TestKeystrokeCost 221 µs and 540 µs a key, TestLongSession delta-to-frame p95 4.63 ms after 26,209 events, TestPasteToFrame p50 1.04 ms, TestStreamToFrame p95 4.00 ms, TestThinkingFrame p95 4.15 ms, every target p95 ≤ 16 ms
PASS 29.6 Windows Terminal: quota-max editor (14.9 s, "1 DONE" found), quota-120x40 editor (14.5 s), waiting-max editor (13.5 s, "WAITING" found): the rows "> ▌Ask Mock data anything…", the separator, "  +     Haiku 5.5 High · Ultracode … ↯ Auto"; the conformance table below, 10 rows PASS, 3 with the terminal's rendering noted; the waiting copy shows the idle row too, the mock's turn being idle at the approval; the owner reviews the screenshots before any commit
PASS 29.7 ui-design.md (Input editor: Size, The row, While a turn runs; the mock-up table; the states table; the keys configuration; the three Layouts frames), reference-design.md (the image's row and section), brief.md (amendment 29), AGENTS.md (status), plan.md (D1 to D4, 29.1 to 29.7, progress); every changed text file read byte by byte: no BOM, no CR; git status lists the intended files only
Not verified: the owner's review of the screenshots; a live session's row with a turn running (the tests and the golden frames cover it; the screenshots are mock runs whose turn is idle at the approval)
Verdict: DONE
```

**Conformance with the image** (`references/08-input-editor.png`), read
against `out-p29\quota-max\editor.png` and
`out-p29\quota-120x40\editor.png`:

| The image | The dashboard | Result |
| - | - | - |
| The prompt row, "> " in lavender, the cursor block, "Ask Claude Code anything..." in grey | unchanged: "> ", the cursor, the placeholder in grey ("Ask Mock data anything…" in a mock run) | PASS |
| A thin grey separator across the box under the prompt row | a rule in the frame's colour (`#3B3F51`; the image's `#3F4650`) | PASS |
| A small rounded box with a white "+" at the left, bordered, on the dark background | " + " on the tool rows' ground, the "+" in the text colour: a one-row band where the image has a bordered box, which needs three rows | PASS, the terminal's one-row box |
| A dark grey rounded pill "Fable 5.1 High · Ultracode" after a gap | " Haiku 5.5 High · Ultracode " on the tool rows' ground (`#1E2133`; the image's `#1D2129`), three cells after the "+": the session's model, then the owner's placeholders | PASS |
| "Fable 5.1 High" in white | the text colour (`#E5E7EB`; the image's `#DDEEFE`) | PASS |
| A lavender dot | the title colour (`#C4B5FD`; the image's `#BCBAF8`) | PASS |
| "Ultracode" in violet, bold | YOU's violet bold (`#A78BFA`; the image's `#9C62F6`) | PASS |
| The middle empty | empty while the editor has nothing to say; the line count past one line and the status there otherwise, in grey | PASS, the row's free middle |
| A lightning bolt and "Auto" in white at the right end | "↯ Auto" in the text colour, one cell from the edge: the arrow glyph stands for the bolt, a two-cell emoji in the terminal's font | PASS, the glyph |
| Nothing acts: design only | no key, click, event or state behind the "+", the pill or "Auto"; a click on the row focuses the editor | PASS |

**What it costs.** Zone B's conversation is one row shorter at every
size, the editor taking four rows for a one-line draft where it took
three. At 80x24 B's room is 11, where the live thinking card takes its
two-row form (five before); at 120x40 with an agent in play its room is
19, the five-row card (seven before); at 200x60 and 208x59 the seven-row
card stays. One row of the card's thresholds would give the seven-row
card back at 120x40, if the owner prefers.

**Evidence outside the repository**, in the session's scratchpad:
`term\out-p29\` (the screenshots, their UI Automation rows and `log.txt`),
`term\p29demo.ps1`, `term\acc-p29.exe`, `plan24\p29-checks.log`,
`plan24\framecheck29.cjs`, `plan24\docframes\` (the three redrawn frames),
`plan24\uidesign28.cjs` (the frames' replacement, reused), the patch files
`p29-open.patch`, `agents29.patch`, `amendment29.md`.

**The commit and the push (2026-10-22), after the owner's approval.** The
owner approved plan 29 on 2026-10-22; commit d696f22 (74 files: 71
modified, 59 of them golden frames, 3 added) and the approval commit were
pushed to `origin`; `origin/main` equals `main`, the working tree is
clean, and the author is `Evengelius <molimoabel@gmail.com>`, with no
attribution line: PASS. The block above stays as it was reported.
