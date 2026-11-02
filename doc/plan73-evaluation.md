# Plan 73 evaluation (2026-11-01)

The check of plan 73, virtual attachments for long user messages (brief amendments 76 and 77): a
user message that draws more than 3 rows is one row in zone B, `▣ Long message · 404 lines · 18 KB`;
a paste whose own content draws more than 3 rows is such a row at once in the composer, its bytes
kept in the one text, with typing, growth, scrolling and Draft as they were; the existing full view
reads the message and `y` copies it. The owner authorized the implementation on 2026-11-01.

**This report is written for independent review and the owner's decision. Plan 73 is not closed:
nothing is committed or pushed, and 73.30, the owner's Human Evaluation, has not been done.** Every
result below comes from automated tests, or from Windows Terminal runs on mock sessions, and says
which. No real Claude Code or Codex turn ran before the final end-to-end verification, below.

**The third review, 2026-11-01: APPROVED WITH RESERVATIONS, for the owner's Human Evaluation only.**
It reviewed `acc-plan73-review-3.zip`, this report as it stood then. What it corrected, and one thing
found while correcting it (R-H), are in "The independent review", below; each later sentence elsewhere
in this report is marked "Reconciled" or dated, and every other result stands as handed off.

**Then R-H was corrected, on the owner's authorization of the same day:** the application code changed. "The
correction of R-H", below, gives the results on the corrected build, which replace the reviewed build's where they
differ. The reviewer approved it.

**Then the final end-to-end verification ran, on the owner's authorization, through real paste, copy, mouse and one turn of each provider:** 33 of 34 scenario runs PASS, and
one open defect, F1, older than this plan. "The final end-to-end verification", below.

**Then F1 was corrected and zone B's selection made to light a chip, after the reviewer approved the core, with no further real CLI turn:** the application code changed again. "The correction of F1
and the chip's selection", below, gives the results on the corrected build, 14 criteria PASS; F1's FAIL stays recorded, on the previous build.

**Approved and closed on 2026-11-02.** The independent review of `acc-plan73-f1-review.zip` is APPROVED, and the owner explicitly approved the Human Evaluation on the evidence already provided, with no manual session: 73.30 is
`PASS — Owner approved`. The blocks below stay as they were reported; "Approval and closure", below, gives the final result. What the sections before that one say about the plan being open, about 73.30 not being done or about nothing
being committed was true when they were written. The sections of `plan.md` they name are now in "The plan as it ran", at the end of this file.

How it was checked:
- **Tests**: the rule, the words and the composer's cells (`internal/render/longmessage_test.go`); the
  ranges, with a model and a fuzz target (`internal/input/ranges_test.go`); paste, keys, Draft, restored
  text, delivery on both adapters, `y`, the mouse on drawn frames, with the existing fake sessions and
  clipboard (`internal/app/longmessage_test.go`), all driven by `tea.PasteMsg` and keys through `Update`.
- **The baseline first**: the whole timing step and the guards on the unchanged code, before any test or
  application code changed; then the guards' fills, compared on the unchanged application code.
- **Each rule broken on purpose** through `go test -overlay`, the repository never written: 75 mutants.
- **The golden frames**: 18 new, 17 changed, each changed one compared with HEAD by a script.
- **Build, vet, gofmt, the whole suite** after the last edit; `-race` on the changed packages; the timing
  step alone, on mains power; C2 against the build from before the feature; 7 fuzz targets, 60 s each.
- **Windows Terminal**: W01 to W08 on the final build, on a mock session, maximised first.

## The evaluation

```text
Evaluation - Plan 73, virtual attachments for long user messages - 30 criteria: 29 PASS, 0 FAIL, 1 not verified yet (73.30, the owner's Human Evaluation)
PASS 73.1 The definition: TestLongMessageDefinition at 49, 69, 73 and 129 cells, 25 cases, each against the sum of D1 and against its expected answer: a\nb\nc false; a\nb\nc\n, \n\n\nabc, a\n\n\n\n true; a line filling exactly 3 rows false, one cell more true; tabs, control characters, bytes that are not UTF-8, C1 controls, wide clusters at a row's edge, emoji, a long word, spaces dropped at three rows' edges. FuzzLongMessage, at any width from 1 to 140: 1,294,906 runs in 60 s, no difference
PASS 73.2 The bound: TestLongMessageBound, through the test hook messageScan: on 50,000 lines exactly 4 lines' cells; on one 3 MB line 3 x (w + 1) cells and the one cell that passes them, which it must read to know the line passes; 0 allocations on both, at each width. M05, M06, M07 killed; TestChipScale: LongMessage 1 to 3 microseconds at 5 KB, 500 KB and 5 MB; C06 and C07, reading the whole text, 59 times slower at 5 MB
PASS 73.3 The chip's words: TestChipWords: exactly "▣ Long message · 404 lines · 18 KB" for 404 lines and 18,000 bytes, "1 line · 15 KB" for one 15,000-byte line, "1,204 lines", "4 lines · 6 B" for a\nb\nc\n, "4 lines · 39 B" for four lines of 日本語, "1.2 MB", the same at 49, 69, 73 and 129 cells; the full view's head gives the same count, "1 line" for one
PASS 73.4 Zone B's two forms: TestZoneBForms at 80x24, 100x30, 120x40, 200x60 and 208x59: 1, 2 and 3 rows are the card's rows, each span's text and style compared with cardText's; four lines, 3 lines and a break, one cell over 3 rows, and 404 lines are the YOU row and one chip row, no byte of the text in zone B, no box, no "more lines". TestZoneBSameFrame: from the composer, from Draft and from a replay the same rows
PASS 73.5 With images: TestChipWithImages: the chip's row, then one row an image with its size; images and short text: the text's row, then the images'
PASS 73.6 Resizing: TestChipResize: one message normal at 120x40, a chip at 100x30, normal at 200x60 and 120x40, the state unchanged; the Explorer docked the same at its width; four logical lines a chip at each size; the cursor on it leaves when it stops being a chip; a composer chip unchanged at 200x60. TestMessageWidthIsDrawn: the layout's width is what zone B wraps to, at five sizes, docked and not
PASS 73.7 The cursor and the full view: TestChipFullView: ↑ stops on the chip with ▸ and the hint, passes over the short message, B's rows unchanged; Enter opens the full view from its top, every line reached with End; Esc gives back the same frame; a double click on the chip's row and on the YOU row opens it, one click nothing. TestEntryViewer, TestEntryClicks, TestEntryCursor. W03, W04 in Windows Terminal
PASS 73.8 y (on a test clipboard; the real one is the owner's step): TestChipFullView: the clipboard holds exactly the text, tabs, trailing spaces, indentation, blank lines, Unicode and a wrapped 15,000-cell line with no line break added; the toast "copied 404 lines · native"; the hint "y copy the message" (TestCommandCopyKey); a command's y unchanged; the entry unchanged
PASS 73.9 Search and selection: TestChipMatchRow: a match behind the chip on the chip's own row, the chip lit as current or as a match; TestSearchLit, TestSearchFolds; TestSearchWhole: Tab opens the full view with the match's row in view; TestChipSearchSelection: a selection across a chip copies the whole message byte for byte
PASS 73.10 Delivery, both providers: TestDelivery on the real Claude Code and Codex adapters: 15 kinds of text (Markdown, fenced code, table, JSON, YAML, PowerShell, Unicode, indentation, tabs, empty lines, a final \, no final break, CRLF, controls, 50,000 lines), each pasted in the composer, in Draft, edited in Draft and mixed with typed text: what the session received equals Editor.Text(), the input under the line-break rule and Entry.Text, with no NUL; Claude Code's stdin line compared byte for byte. TestLiteralNUL: a NUL the owner pasted is kept and sent, and taken for no chip
PASS 73.11 Typing unchanged: TestTypingUnchanged: 60 typed lines, no range, ordinary rows, "60 lines"; no golden frame of a typed draft changed; the existing size and scrolling tests pass unchanged. The key tables: y's row, and the ↑ ↓ row's one word "folded", now "long", the fold being gone (a reservation, below)
PASS 73.12 A long paste: TestLongPaste: 3 lines and 3 rows stay text, 4 rows and 3 rows and a break are a chip at once, nothing sent; 2 typed lines then a long paste; 10 typed lines then a 2-line paste, no chip; 8 short pastes, no chip; a paste with Draft open, no range. TestLongPasteOtherWays: the raw Ctrl+V and the paths' fallback
PASS 73.13 A chip's edges: TestChipEdges: a\nb\nc\n shows "3 lines · 5 B", the cursor on the row below, then "4 lines · 6 B" once sent; typed text on the chip's row with no byte added; Ctrl+J; pasted between typed text, both sides intact; "201 lines" with a typed line, none with a chip alone
PASS 73.14 Two chips: TestTwoChips on both adapters: typed text before, after and between them, visible and editable; each chip's words its own; one deleted and undone, the other and the text byte-identical; sent, those blocks and that text in that order
PASS 73.15 The cursor: TestChipCursor: 1,500 random motions, Shift forms and clicks on three chips, two adjacent: never inside, the text unchanged; TestChipMotions; TestChipClicks; the model test checks it after every operation
PASS 73.16 Deletion, undo, the clipboard: TestChipDeleteKeys, TestChipDeletion: each of the four keys from its edge; a word deletion stopping at each edge; Ctrl+Z and Ctrl+Y restoring bytes, chip and cursor; a selection holding a chip deleted, typed over and cut; adjacent chips; Ctrl+C, Ctrl+X and a drag's copy on release give exactly the selected bytes
PASS 73.17 Enter and \: TestChipEnter: a paste ending in \ sends, unchanged; a typed \ after a chip is a newline; in Draft as today; during open markers no key sends; a turn running: the chip and text kept, "turn running: wait, or interrupt"
PASS 73.18 Draft: no draft-* golden frame changed; TestChipDraft: the frames, cursor and text the same with and without ranges through ten keys; no change then Esc, the chip back, the cursor where it was on reopening; an edit, ranges gone, nothing lost; undo to the bytes, the chip back; an overlapping list dropped, the text intact. TestInvalidRanges: eight kinds of bad list against ten uses
PASS 73.19 Restored text: TestChipRestored: a refused write's text comes back ordinary, byte-identical to what was sent, CRLF stored as LF being the only difference from the paste; with text typed since, joined as today; TestChipStoredDraft: the store holds the text, and it comes back ordinary
PASS 73.20 Model test: TestEditorRangesModel, 60 seeds of 1,500 operations of every kind in both modes; TestEditorPlainModel: with no range the editor is today's, anchor included. FuzzEditorRanges: its first run found the model's own check too strict (below); after that correction, 652,279 runs in 60 s, no failure
PASS 73.21 Golden frames: 18 new (a chip, its cursor, a composer chip with typed text, at 80x24, 100x30, 120x40, 200x60, 208x59 and 79x24); at 100x30 "▣ Long message · 404 lines · 10 KB" and "▸▣ Long m… Enter full view · ↑↓ entries · Esc back", each one row; 17 changed, reviewed by script: 113 rows, none outside zone B
PASS 73.22 The scope: git diff shows nothing under internal/state and internal/provider, nor go.mod, go.sum or draft.go; the changed code adds no file write; architecture test 7 passes
PASS 73.23 The guards: TestKeystrokeToFrame, TestDraftKeystrokes, TestKeystrokeCost and TestKeystrokeAllocs fill with fillOrdinary and call ordinaryDraft before and after their keys; TestGuardFailsOnAChip: on the original composer-paste fill under the feature the check fails, naming the chip; over Draft with ranges it passes
PASS 73.24 Baseline and comparisons: the baseline, C1 and C2 below, each run's conditions with it; C1's functional equivalence holds; no reproducible difference in either, the small ones named under Reservations
PASS 73.25 Budgets, final build: a chip in view p95 1.11 ms; keys around it p95 1.07, 1.26 and 1.51 ms; the full view p95 0 and 1.00 ms, after P4's remedy (first measured at 52.7 and 44.3 ms, a miss); a paste in the composer max 1.18 ms, with Draft open 4.06 ms; a chip's frame 507 to 519 allocations against 523 for a 3-row message; y's key in Update max 0.75 ms; all 21 existing timed tests and the guards within their unchanged thresholds
PASS 73.26 Hidden text: TestComposerHiddenText: no hidden byte in the view's lines, its line, column, selection and total the compact form's; M54 to M58 killed by value; C01 to C05, adding the whole-text scans with no value changed, 97 to 699 times slower at 5 MB, seen by TestChipScale; the unmutated build's 36 rows between 0.34 and 1.20 from 5 KB to 5 MB, in the 70 ns to 3 microseconds range. m.edit is the only authoritative text: Compact builds a string for one call; by reading the diff
PASS 73.27 The checks: go build ./..., go vet ./... (and with the timing tag), gofmt -l . printing nothing; the whole suite after the last edit, exit 0: 1,235 tests pass, 3,384 with subtests, 0 fail, 5 skip on conditions of their own (TestRecoveryChild, TestWriteTestImages, TestTurnLinks, TestWireReplay, TestDocFrames); the 7 fuzz targets, 60 s each; 75 mutants, one per rule of D1 to D5 and D7 to D11 at least: 68 by value killed, 7 by cost seen; -race on render, input and app: no race; the timing step alone: all 25 timed tests pass
PASS 73.28 The documents: ui-design.md (YOU's entry, Long messages, Expanding, Search, Scrolling, Paste, Long pastes, Budget, Draft window, the key table, Selection and copy), architecture.md (ASM-22, the matcher's row, plan 73's table), evaluation.md (the timing list and count, the fuzz list), keys.go's rows, plan.md, this file, brief amendment 77; UTF-8, LF, no BOM, checked by script
PASS 73.29 The Windows Terminal scenes: W01 to W08, 9 runs on one build, each exit 0, 39 assertions passed, none failed; below
Not verified 73.30: the owner's Human Evaluation. A real paste, double clicks, y and what a paste of it gives back, Shift+click in a real terminal, the chip's glyph and colours by eye, whether a very large y's OSC 52 stalls Windows Terminal (P5, V7, V8). Any real CLI turn, which needs its own authorization
Verdict: NOT DONE: implementation and automated verification complete, waiting for the owner's Human Evaluation and review
```

The block above is the report as handed off, unchanged, for the build reviewed. Read 73.1, 73.2 and 73.11 with
"The independent review", below, which corrects 73.2's and 73.11's wording, and with "The correction of R-H", whose own
block gives the results on the corrected build and restores the rule that 73.1 checks.

## Failures, limitations and reservations

Nothing below is hidden in a PASS; each is for the reviewer's and the owner's reading.

**Failures met and resolved during the work:**
- **The full view missed its budget** on its first measure (p95 52.7 and 44.3 ms): remedied as P4 foresaw;
  see The full view, below. The baseline had the same cost.
- **`FuzzEditorRanges` failed at 8 s** on its first 60 s run. Its input leaves the cursor inside a block
  through Draft and then, skipping `Settle`, deletes a word: the deletion runs from inside the block, and
  the range ends. The text still equalled the model's and nothing was lost; what failed was the rig's rule
  that a word deletion never crosses into a chip. The application cannot reach this: every composer key,
  paste and copy settles first, `Undo` too. The rule now applies to settled states only, no application code
  changed, and the input stays as a seed. A rerun of 60 s found nothing.
- **Three mutants survived the first run**: M15 and M65 for want of a test (`TestChipMatchRow`, and a long
  line in `TestMessageViewMemo`, added), M69 being a cost alone (C07). Four mutants did not compile and M13 did
  not apply, each my own mistake in writing it; corrected and run again, all killed. The first run's log is
  kept.
- **The first run of the Windows Terminal scenes failed in 8 of 9**, by my plans: the recording replays in real
  time for 26.3 s, and they waited 25 s for `RESULT`, or sent W08's keys during the replay. The application was
  not at fault and did not change: the plans now wait for `COMPLETED`. Those runs are under `-tag claude`.
- **The race run's first attempt never compiled**, my script naming the wrong folder for gcc; run again.
- **`TestEntryViewer` failed once** among the runs of the whole suite ("↑ to the thought: not in view"), an
  existing test's own flake and no application defect. Whether a frame sees P2's thinking phase live depends on how
  the replay's events were batched, and once seen, on the test's clock that stands still, the phase keeps its card, no
  stop for the cursor. On the baseline's untouched code, the state forced, it gives the same failure
  (`flake/baseline-proof.log`). 150 repeats under full CPU load, on both trees, did not bring it back. `p2Done` now moves
  the clock a second past the phase's end, assertions unchanged; the same state then passes
  (`flake/final-proof.log`), and the suite ran again.

**Limitations and reservations, none of them failing a stated budget:**
- **R-A, the search's bar over a very long full view.** With the bar open, over the full view of a 50,000-line
  message, a frame takes p50 32 ms, p95 34 ms, over the 16 ms every other frame meets. It was 74 ms (p95 76 to
  124 ms) on the baseline's code, so this plan halves it through P4's memo. `lightView` scans the message each
  frame, as it does any entry's. No criterion names this path. I left it: it predates the plan and belongs to
  the optimization campaign the owner set apart. Your decision if it matters sooner.
- **R-B, the full view's first frame.** The one wrap falls on the frame that opens the view, or a resize while
  it is open: 40 to 55 ms once, on 50,000 lines. On the baseline every frame cost that.
- **R-C, P8's small differences, inconclusive.** In C1 and C2 the cost of a key in `TestKeystrokeCost` reads +1%
  to +3% on the pair's second binary, less than either side's own range, allocations identical. With the pairs
  reversed it reads -0.3%, and a binary against a copy of itself -0.2% and -0.4%. So: not reproducible, and no
  budget near (0.25 and 0.5 ms a key against 16). A real difference of under about 1% cannot be excluded at this
  noise; the new branches in a frame's path are a few comparisons (`drawRow`'s per cell, the empty chip loops).
- **R-D, D10's wording.** It says the mode is set "at the top of `Update`"; the code sets it in `syncUI`, at the
  end of every `Update`, so it holds at the top of the next. Set only at the top, the frame Ctrl+G opens would
  draw the compact form. No behaviour differs from D10's intent (V2, below). **Reconciled:** D10 and the plan's
  Components now say `syncUI` (the review's R2).
- **R-E, 73.2's wording.** "At most `3 × (w + 1)` cells": the scan also reads the one cell that passes them, as
  it must. 73.11's wording, "but for `y`'s row": the `↑ ↓` row changed its word "folded" to "long" too, in the
  overlay and in `ui-design.md`, the fold being gone. **Reconciled:** both criteria now say what is true (the
  review's R1 and R3), and R1 went further than this entry did: below `w` = 4 the bound was not `3 × (w + 1)` at all.
  (Since "The correction of R-H" it is again, a cell wider than `w` counting as `w`.)
- **R-F, the composer's cost with many chips** is measured below (`TestChipCount`), not budgeted by the plan.
- **R-G, what no test here reaches**: a real paste through Windows Terminal, the real clipboard, and each CLI's
  own limit on a very large message, which is not probed; a refusal before acceptance returns the text to the
  editor as ordinary text (73.19).
- **D12 as it stands**: an instruction added to a pasted block is inside zone B's chip once sent, and zone A shows
  the prompt's first line, as the plan says.

**Found on 2026-11-01, on checking the third review's R1 against the code, after the review:**
- **R-H, a false chip: a defect, now fixed.** Characters joined by U+200D are one cell as wide as they all are
  (`cluster`, `clusterWidth`), and a wrap puts that cell on one row. So a line holding a cell wider than a row could
  pass the bound and still draw 3 rows or fewer, and `LongMessage` said long where D1's sum says it is not: at 120x40,
  106 wide characters on one line, each joined to the next. `FuzzLongMessage` had not met it in its 1,294,906 runs, so
  73.1's line above is true of what ran, and the rule was not exact for every text, as D2 claimed. **History:** I first
  judged it not material, recorded it here as an exception, left the code as it was and narrowed `FuzzLongMessage`'s
  property to fit it, with a test pinning the difference. The owner and the reviewer judged it a functional defect,
  since it breaks the plan's rule, and the narrowing unacceptable, and authorized the fix. It is no longer an accepted
  behaviour: the code is corrected and the full property restored ("The correction of R-H", below).
- **R-I, the bound counts cells, not bytes: open, a measured limitation.** What takes no cell adds nothing to the sum
  and one cluster of any length is one cell, so a line made only of those is read to its end. On the reviewed code,
  first measured: `LongMessage` on one line of 1,000,000 U+200B, 3 MB, takes 8 to 10 ms, and on `e` and 1,000,000 U+0301,
  2 MB, 4 to 5 ms (best of 5, at three widths); on 3,000,000 `x`, the clock reads 0. `entryBody` scans a message in view
  on every frame, so with that 3 MB message on screen a frame's median is 119 ms, against 108 ms on the baseline's code;
  with the 2 MB one 88 ms against 84 ms (15 frames, one run each, `Frame` alone at 120x40); a short message's reads 0 on
  both. Such a frame was already far over 16 ms, and no ordinary text gets here. Measured again after the correction,
  with the investigation the owner asked for: "The correction of R-H", below.

## The independent review (2026-11-01)

The owner's independent reviewer examined `acc-plan73-review-3.zip` (SHA-256 `2A3E8781…A11E`) and returned **APPROVED WITH
RESERVATIONS**: "approved to proceed to the owner's Human Evaluation (73.30)", no blocking functional defect identified, and
"not authorization to close Plan 73, commit, push, or start real Claude Code/Codex sessions". The reviewer applied the patches
and read the code and the evidence; the tests and measurements were reviewed "through the provided evidence", not run again.
The review asked for three corrections of wording and one check, with no application code changed.

**Superseded in part.** This section records that reconciliation as it stood, for build `8E4E97BB…16A0`. "The correction of R-H",
below, has since changed the code: the bound is no longer `3 × (max(w, 4) + 1)`, `TestLongMessageWideCell` no longer exists, and
`FuzzLongMessage`'s property is the full one again. R2, R3 and the golden frames' explanation stand as they are.

| Item | What was written | What was true then | Corrected in |
|---|---|---|---|
| **R1**, the bound | 73.2: "at most `3 × (w + 1)` cells"; D2: the same | At most 4 lines and, in each, widths summing to at most `3 × (max(w, 4) + 1)` and the one cell that passes that sum | D2, 73.2, `ui-design.md` (Long messages), `TestLongMessageBound`, `FuzzLongMessage` |
| **R2**, the mode | D10: "at the top of `Update`"; Components: "the mode at `Update`'s top" | Set in `syncUI`, at the end of every `Update` but the quit's, so it holds at the top of the next and for the frame Ctrl+G opens | D10, Components, V2's note |
| **R3**, the key tables | 73.11: "but for `y`'s row" | The `↑ ↓` row's word `folded` is now `long` too, the message's fold being gone | 73.11, 73.28 |
| Golden frames | Three warnings, unexplained | Intentional blank rows; `git diff --check` does not see untracked files | "Golden frames", below |

**R1, as checked against `messageScan`,** with `most` = `3 × (max(w, 4) + 1)`:
- **Lines.** Each line adds at least one row, so the fourth is the last it can read. Each line's cells show which were
  read: a fifth line is not; a fourth is read up to the bound (3 + `most` + 1 cells).
- **Cells in a line.** It reads until the sum passes `most`: the last cell starts at or under `most` and adds its width, at
  most 4 for every code point and every byte that is not UTF-8, all of them tested. So at most `most + 4`, reached at `w` 7 by
  tabs: 24, then 28. In all, at most `4 × most + 4`, which the fuzz target checks at any width, a cell of R-H adding its own width instead.
- **Why that formula.** A row holds at most `max(w, 4)` cells, a tab or a byte's `<XX>` taking its 4 even on a narrower
  row, and one space dropped at its edge. It can be no lower: at each width, exactly `most` cells draw exactly 3 rows
  and are not long, and one cell more is. `3 × (w + 1)` is 6 at `w` 1, where three tabs and their spaces take 15.
- **A wrap.** A line wider than `w` and within `most` is then wrapped once, over the same cells; `looked` counts the sum, not the wrap.
- **What R1 did not name and this check found:** R-H and R-I, above. The criterion now says both.

`TestLongMessageBound` now checks that at 12 widths (1 to 8, 49, 69, 73, 129), on seven kinds of cell, each line of 3 MB.
`TestLongMessageWideCell` is new. The exact old wording stands in the plan's 73.2 and 73.11, and in R-D and R-E above.

**The performance reservations.** The reviewer accepted R-A, R-B and R-C as documented, non-blocking reservations for this scope and
asked for no further optimization under this plan. They stay open, as written above: nothing here claims 16 ms for the
search's bar over a 50,000-line full view, for that view's first frame, or a proof of no difference under about 1%. A broader
optimization stays apart from Plan 73.

**The focused checks,** the review having authorized no others. No application code changed, so the whole suite, the timing step, C1 and
C2, `-race` and the Windows Terminal scenes did not run again, and their results above stand for that code:

```text
Reconciliation, 2026-11-01
PASS The application code: the non-test diff's fingerprint a8e6e10736aa6016 and the build's SHA-256 8E4E97BB...16A0, both as reviewed
PASS What changed since package 3: of its 82 inventoried files, 4: plan.md, doc/ui-design.md, doc/plan73-evaluation.md, internal/render/longmessage_test.go; git status --short the same as the package's, nothing under internal/state, internal/provider or go.mod
PASS gofmt -l . prints nothing; go vet ./internal/render, with and without the timing tag, exit 0
PASS go test ./internal/render, the whole package, TestKeyOverlayRows included: 334 pass, 1 skips on its own condition, 0 fail
PASS TestLongMessageBound, TestLongMessageWideCell, TestLongMessageDefinition: pass; FuzzLongMessage: 218 seeds, then 60 s, 1,971,300 runs, no failure
PASS 13 mutants of the bound (B01 to B13: the formula up and down, the passing cell, the fourth line, what takes no cell, widestCell at 3 and 5, the sum), through overlays, the tree never written: 13 killed by those two tests alone, none a build failure
PASS TestGoldenFrames: exit 0, its 299 subtests with the 18 chip frames; no frame changed
PASS go test ./internal/arch: pass
PASS The four files: UTF-8, LF, no BOM, by script
Not rerun: everything else; 73.30 is still the owner's. Verdict: reconciled; the plan stays open at the owner's Human Evaluation
```

Unchanged: package 3 and its evidence, as the owner asked. `doc/brief.md` and `AGENTS.md` are untouched too, the review's scope naming neither; this section
is where the review's decision stands. No new archive, these changes being within that scope.

## The correction of R-H (2026-11-01)

After the reconciliation above, the owner and the independent reviewer reconsidered R-H: "R-H must be corrected before Human
Evaluation 73.30", a functional correctness defect and no acceptable exception, and the narrowed fuzz property "not an acceptable
final resolution". The owner authorized the smallest application correction, its regression tests, the fuzz property restored, a
focused look at R-I and the documents; no commit, push, closure, real CLI session or wider optimization. This section is what came
of it, for the independent review that comes next. The owner's Human Evaluation waits for that review.

**Root cause.** `messageScan` added up each cell's full width and took a sum over `3 × (max(w, 4) + 1)` for more than 3 rows. That
rests on a row holding at most `max(w, 4)` cells, which is false for one kind of cell: `cluster` keeps characters joined by U+200D
as one cell, `clusterWidth` measures it as wide as all its characters, and `wrapCells` gives a cell wider than a row one row. The
constant `widestCell`, "the most cells one drawn unit takes", was wrong for it. So the sum counted what takes one row as many.

**The change.** In `internal/render/content.go` alone; no other application file. `widestCell` is gone, and in `messageScan`:

```go
// reviewed
most := messageRows * (max(w, widestCell) + 1)
width, i := 0, at
for i < len(text) && text[i] != '\n' && width <= most {
	n, cw := cellSpan(text, i)
	width, i = width+cw, i+n
}
looked += width
switch {
case width > most:

// corrected
most := messageRows * (w + 1)
width, sum, i := 0, 0, at
for i < len(text) && text[i] != '\n' && sum <= most {
	n, cw := cellSpan(text, i)
	width, sum, i = width+cw, sum+min(cw, w), i+n
}
looked += sum
switch {
case sum > most:
```

The renderer stays the source of truth: `case width > w` still takes the line's rows from `wrapRanges`, through the cache the card
then draws from. No buffer, file, dependency or architecture changed, nor the composer, Draft, the clipboard or the send path.

**Why it is exact for every text,** from `wrapCells`:
- A cell wider than a row is placed only on an empty row, and the next cell of any width, even none, ends that row. So such a cell
  has its row alone, and every other row's real widths sum to at most `w`. With a cell wider than `w` counted as `w`, a row counts at
  most `w`.
- Only a space is dropped, and dropping it ends the row. So `R` rows count at most `R × (w + 1)`, and a line over `3 × (w + 1)` is over
  3 rows, whatever its cells.
- A line within that is read to its end. With a real width of at most `w` nothing ever overflows: one row. Otherwise `wrapRanges` gives
  the rows themselves. `rows` sums them and stops as soon as it passes 3.

So the bound is again the `3 × (w + 1)` that 73.2 first named, the count being what changed, and no character's width enters it. A
further read of the reference found no error in it: `entryBody` draws an ordinary message with `cardText` over `entryLines`, whose lines
are `strings.Split(e.Text, "\n")`, each wrapped by `wrapRanges(l, tw, tw)`, which is `rowsDrawn`.

**The full property is back.** `FuzzLongMessage`'s check is again, word for word, what the review package held:

```go
want := rowsDrawn(text, w) > messageRows
if got := LongMessage(text, w); got != want {
```

Both directions, no exception. It gains two checks: `rowsDrawn` equals the rows of `cardText` itself, and the scan stays within its
bound. Nine sets of R-H's inputs stay among its seeds for good, the fewest characters that showed it at each of zone B's widths and at
narrow ones, so `go test` runs them every time. The narrowed property, `widestOf` and `TestLongMessageWideCell` are gone.

**The new tests catch the defect.** Run against the reviewed `content.go` through an overlay, the tree untouched: `TestLongMessageBound`,
`TestLongMessageRenderer`, `TestLongMessageEquivalence`, `TestJoinedCells` and 25 of `FuzzLongMessage`'s seeds fail, for instance
"[80 24], one extremely wide grapheme: LongMessage true, but 1 rows are drawn". On the correction all of them pass. 60 s of fuzzing on
the final code: 848,369 runs, no failure.

| The owner's case | Expected | Result, at 80x24, 100x30, 120x40, 200x60 and 208x59, against `cardText` and zone B's own rows |
|---|---|---|
| Exactly 3 visual rows | Ordinary | PASS, 3 rows, no chip |
| Exactly 4 visual rows | Chip | PASS |
| One extremely wide grapheme on one row | Ordinary | PASS, 1 row, `…` on it, with 2,000 joined characters and with the fewest past the old bound |
| Multiple joined graphemes spanning 3 rows | Ordinary | PASS, with and without spaces, a final space, on three lines; also joined characters a row can hold |
| The same content spanning 4 rows | Chip | PASS, with and without spaces, three and one cell, three lines and a final break |
| A long ASCII line over 3 rows | Chip | PASS, 100 rows |
| Three logical lines and a final newline | Chip, the renderer drawing 4 rows | PASS |
| A large zero-width sequence | Correct, cost measured | PASS, 1 row, no chip; with three more lines, 4 rows, a chip; cost below |
| Mixed Unicode, spaces, tabs and controls | Exact equivalence | PASS, on one row and wrapping; `TestLongMessageEquivalence`, 20,000 random mixes of some 30 kinds of cell at any width, in every run; the fuzz target |

`TestJoinedCells` takes the same cells through the composer and both real adapters: one and three ordinary, four a chip, and what the
session receives and `Entry.Text` equal to the bytes pasted. **What such a row looks like:** a cell wider than its row shows `…`, as it
does on the baseline (`line.fit`), with no full view, as for any ordinary message. W09 below shows it.

**Before and after.** Three builds, the same probes through overlays, none of it in the tree (`zz_rh_frames_test.go`, `zz_rh_scan_test.go`):
the baseline `1c36ceb`, the reviewed code and the corrected code, in turn with the order rotating, nothing else running, on mains power,
the owner's usual applications open and idle, each build's conditions logged. Four rounds; the reviewed build's first (`compare-b/rev-1.log`) is left out of the tables, being the
first run after the builds and about three times slower on every line, ordinary ones included, and its log is kept. So each cell is three
or four rounds: the range of the p50, then the worst p95. `text` or `chip` is the form each build gave.

LongMessage alone, per call, at 69 cells, the width at 120x40 (the baseline has no such function):

| Message | Reviewed | Corrected |
|---|---|---|
| a short message | text: 90 to 95 ns | text: 90 to 95 ns |
| 3 rows of ASCII | text: 5.1 to 5.4 µs | text: 4.5 to 5.4 µs |
| 200 lines of ASCII, 12 KB | chip: 890 to 950 ns | chip: 895 to 965 ns |
| a 15 KB ASCII line | chip: 770 to 825 ns | chip: 770 to 825 ns |
| one 3 MB line | chip: 770 to 825 ns | chip: 770 to 825 ns |
| one 5 MB line | chip: 770 to 825 ns | chip: 825 ns |
| 50,000 lines, 2.8 MB | chip: 815 to 875 ns | chip: 825 to 880 ns |
| R-H: 300 joined characters, 1.8 KB | chip: 3.3 to 3.6 µs | text: 14.3 to 15.3 µs |
| R-H: 500,000 joined characters, 3 MB | chip: 6.2 to 6.6 ms | text: 22.7 to 23.1 ms |
| R-I: 1,000,000 U+200B, 3 MB | text: 9.3 to 9.5 ms | text: 9.5 to 9.8 ms |
| R-I: `e` and 1,000,000 U+0301, 2 MB | text: 5.4 to 5.5 ms | text: 5.4 to 5.5 ms |

A frame with the message in zone B, a key typed in the composer below it, at 120x40:

| Message | Baseline | Reviewed | Corrected |
|---|---|---|---|
| a short message | text: 88 to 110 µs, p95 274 µs | text: 80 to 82 µs, p95 191 µs | text: 80 to 118 µs, p95 237 µs |
| 3 rows of ASCII | text: 88 to 91 µs, p95 234 µs | text: 85 to 90 µs, p95 200 µs | text: 86 to 135 µs, p95 243 µs |
| 200 lines of ASCII, 12 KB | text: 94 to 118 µs, p95 523 µs | chip: 83 to 88 µs, p95 228 µs | chip: 86 to 124 µs, p95 300 µs |
| a 15 KB ASCII line | text: 160 to 265 µs, p95 1.1 ms | chip: 100 to 147 µs, p95 413 µs | chip: 103 to 153 µs, p95 313 µs |
| one 3 MB line | text: 11.3 to 12.1 ms, p95 14.1 ms | chip: 4.4 ms, p95 5.3 ms | chip: 3.9 to 4.4 ms, p95 5.4 ms |
| one 5 MB line | text: 18.3 to 19.8 ms, p95 23.8 ms | chip: 7.3 to 7.5 ms, p95 9.0 ms | chip: 6.8 to 7.8 ms, p95 8.7 ms |
| 50,000 lines, 2.8 MB | text: 693 to 771 µs, p95 1.2 ms | chip: 144 to 189 µs, p95 400 µs | chip: 137 to 142 µs, p95 388 µs |
| R-H: 300 joined, 1.8 KB | text: 117 to 123 µs, p95 458 µs | chip: 96 to 133 µs, p95 288 µs | text: 112 to 121 µs, p95 308 µs |
| R-H: 500,000 joined, 3 MB | text: 47.1 to 48.0 ms, p95 50.5 ms | chip: 22.1 to 22.5 ms, p95 25.2 ms | text: 50.5 to 52.5 ms, p95 55.5 ms |
| R-I: 1,000,000 U+200B, 3 MB | text: 106.0 to 108.1 ms, p95 109.8 ms | text: 115.0 to 123.2 ms, p95 130.9 ms | text: 116.3 to 117.3 ms, p95 122.1 ms |
| R-I: `e` and 1,000,000 U+0301, 2 MB | text: 81.1 to 82.0 ms, p95 84.6 ms | text: 86.9 to 90.3 ms, p95 94.1 ms | text: 86.9 to 87.4 ms, p95 99.6 ms |

The same text pasted in the composer, to its frame, at 120x40:

| Message | Baseline | Reviewed | Corrected |
|---|---|---|---|
| a short message | text: 81 to 131 µs, p95 247 µs | text: 161 to 250 µs, p95 426 µs | text: 82 to 198 µs, p95 258 µs |
| 200 lines of ASCII, 12 KB | text: 119 to 213 µs, p95 1.6 ms | chip: 94 to 202 µs, p95 327 µs | chip: 133 to 232 µs, p95 366 µs |
| a 15 KB ASCII line | text: 682 to 757 µs, p95 3.1 ms | chip: 129 to 219 µs, p95 1.1 ms | chip: 123 to 153 µs, p95 504 µs |
| one 3 MB line | text: 121.7 to 124.8 ms, p95 131.8 ms | chip: 279 to 410 µs, p95 1.1 ms | chip: 227 to 302 µs, p95 1.2 ms |
| one 5 MB line | text: 199.7 to 209.6 ms, p95 213.6 ms | chip: 447 to 623 µs, p95 2.2 ms | chip: 400 to 471 µs, p95 1.5 ms |
| 50,000 lines, 2.8 MB | text: 752 µs to 1.1 ms, p95 2.5 ms | chip: 448 to 488 µs, p95 769 µs | chip: 204 to 413 µs, p95 812 µs |
| R-H: 300 joined, 1.8 KB | text: 110 to 268 µs, p95 365 µs | chip: 84 to 209 µs, p95 351 µs | text: 171 to 311 µs, p95 2.7 ms |
| R-H: 500,000 joined, 3 MB | text: 49.1 to 50.8 ms, p95 54.3 ms | chip: 6.3 to 6.7 ms, p95 7.7 ms | text: 69.5 to 70.9 ms, p95 72.5 ms |
| R-I: 1,000,000 U+200B, 3 MB | text: 104.2 to 105.8 ms, p95 120.1 ms | text: 113.3 to 114.7 ms, p95 119.9 ms | text: 112.7 to 115.2 ms, p95 119.0 ms |
| R-I: `e` and 1,000,000 U+0301, 2 MB | text: 58.0 to 60.8 ms, p95 68.6 ms | text: 64.8 to 65.3 ms, p95 68.2 ms | text: 64.9 to 65.8 ms, p95 67.1 ms |

The 208x59 rows are in `before-after.md`, and they say the same. `TestKeystrokeCost` on the guards' replacement fill: 598 allocations a key on both builds in every run;
a 10,000-line draft 243 to 271 µs a key reviewed, 249 to 260 µs corrected; a 50,000-line draft 504 to 513 µs reviewed, 506 to 508 µs corrected.

**How I read them (P8).** Ordinary text, chips, the 50,000-line and the 3 to 5 MB messages: no reproducible difference between the reviewed and corrected builds, the
ranges overlapping and the direction changing, allocations identical. The named budgets all hold on the final build (below). What the correction does change is reproducible and only on
its own input:
- **A message with megabytes in one joined cell** is now the ordinary row the baseline drew, so it costs what that costs and no longer a chip's: a frame 50.5 to 52.5 ms (47.1 to 48.0 ms on the
  baseline, 22.1 to 22.5 ms as the false chip), the paste 69.5 to 70.9 ms (49.1 to 50.8 ms; 6.3 to 6.7 ms), `LongMessage` 22.7 to 23.1 ms (6.2 to 6.6 ms), the whole line's cells
  being built for its wrap. On the frame path that wrap is cached, so about 4 ms of each frame is the scan. Outside it, `LongMessage` runs on no cache: on a paste, and on ↑ and ↓ for each
  message. Those frames are over 16 ms, as they were on the baseline, and no budget names this input. At 1.8 KB, the size such a text has if anyone writes it: 14 µs, and a frame unchanged.
- **Also seen, not caused by this:** a chip of one line costs its frame 3.9 to 4.4 ms at 3 MB and 6.8 to 7.8 ms at 5 MB, on both builds, where a chip of 50,000 lines costs under 0.2 ms. `messageScan` is 0.8 µs of
  it, so it is elsewhere on the frame's path. It is within the budget and it improves on the baseline's 11 to 20 ms; I did not look further, that being outside this correction.

**R-I, as the owner asked.**
1. **No common cause.** R-H is the sum overstating rows, a cell counted for more than its one row: an error in the answer. R-I is the sum not bounding bytes, what takes no cell rightly counting for nothing:
   a cost, the answer being right. They meet only in that both come from measuring a line by its cells. Clamping a cell's top does nothing for its bottom.
2. **Measured again:** the tables above. `LongMessage` 9.5 to 9.8 ms and 5.4 to 5.5 ms; a frame 116.3 to 117.3 ms and 86.9 to 87.4 ms; a paste 112.7 to 115.2 ms and 64.9 to 65.8 ms.
3. **No regression from this fix:** the reviewed build's ranges hold those values. The classification is right, and `TestLongMessageRenderer` checks it.
4. **No efficient shared correction.** To say a text is not long, every byte of it must be read, and such a text is megabytes that are not long, so no bound can stop early and stay exact. What could go is the repeat: a memo of the answer per text and width in `WrapCache`,
   some ten lines and a test. It would save the scan's 9.5 of the frame's 117 ms and leave 106 ms that the baseline already spends drawing that row, still over six times the budget. So I am not proposing it. The paste, over the 100 ms named for 50,000 lines, is on
   this input too, and was 104 ms on the baseline.

**Recommendation: leave R-I documented, as a measured limitation of a text no one writes.** Making such a frame fast is the baseline's drawing, which belongs with the optimization campaign the owner set apart, by a separate decision.

**What ran on the corrected code.** Fingerprint of the non-test diff `aa7bcf0262892aa4`, build SHA-256 `9EDA74D96DAAB6CEFA3CB5729E647C783D61256820F8DCE4196EA1D84ED6AEE6`:

```text
Evaluation - Plan 73 on the corrected build, 2026-11-01 - 30 criteria: 29 PASS, 0 FAIL, 1 not verified yet (73.30, the owner's Human Evaluation, after the independent review)
PASS 73.1 The definition: restored without exception. TestLongMessageDefinition at 49, 69, 73 and 129 cells, its 25 cases; TestLongMessageRenderer, 27 cases at 5 sizes, against cardText and zone B's rows; TestLongMessageEquivalence, 20,000 random mixes; TestJoinedCells on both adapters; FuzzLongMessage, full equivalence at any width from 1 to 140, its seeds holding R-H's inputs: 225 seeds, then 60 s, 848,369 runs (-parallel 4), no failure
PASS 73.2 The bound: TestLongMessageBound at 12 widths from 1 to 129, on eight kinds of cell of 3 MB each: with most = 3 x (w + 1) and a cell wider than w counted as w, exactly the first multiple of the cell past most, at most most + w; 4 lines at most, a fifth not read, a fourth read up to the bound; exactly most cells on exactly 3 rows and one more over them, at every width, for plain cells, joined cells and tabs; 0 allocations for a text its breaks or a line's cells decide; a zero-width line read to its end, 0 cells counted (R-I). B01 to B17 killed; TestChipScale: LongMessage 1 to 2 microseconds at 5 KB, 500 KB and 5 MB; C06 and C07, reading the whole text, 42 to 66 times slower at 5 MB
PASS 73.3 to 73.20: their tests in the whole suite on this code, unchanged in what they check
PASS 73.21 Golden frames: TestGoldenFrames exit 0; no frame file changed since the review package, by its inventory's SHA-256
PASS 73.22 The scope: nothing under internal/state or internal/provider, nor go.mod, go.sum or draft.go; architecture test 7 passes; one application file changed since the review package
PASS 73.23 The guards: as before, in the suite
PASS 73.24 Baseline and comparisons: the baseline and C1 reused, on code this does not touch; C2 reused for the feature against the baseline; the comparison above for the correction: no reproducible difference on ordinary text
PASS 73.25 Budgets, final build, the whole timing step alone: a chip in view p95 1.09 ms; keys around it p95 1.06, 1.52 and 1.26 ms; the full view p95 0 and 1.00 ms, its one wrap on opening 39.9 and 38.4 ms (R-B); a paste in the composer max 1.01 ms, with Draft open 2.65 ms; a chip's frame 507 to 519 allocations against 523 for a 3-row message; y's key max 0.64 and 0.92 ms; all 25 timed tests and the guards within their unchanged thresholds. The search's bar over that view p50 30.1 and 32.0 ms, p95 34.3 and 34.5 ms (R-A, no budget)
PASS 73.26 Hidden text: TestComposerHiddenText in the suite; TestChipScale: 36 rows between 0.33 and 1.40 from 5 KB to 5 MB; C01 to C05 reused, the composer's code untouched
PASS 73.27 The checks: go build ./..., go vet ./... (and with the timing tag), gofmt -l . printing nothing; the whole suite after the last edit to the code, the tests and the documents tests read, exit 0: 1,238 tests pass, 3,416 with subtests, 0 fail, 5 skip on conditions of their own (TestRecoveryChild, TestWriteTestImages, TestTurnLinks, TestWireReplay, TestDocFrames); -race on render, input and app: no race; FuzzLongMessage 60 s; 17 + 19 mutants of the rule killed, M69 a cost alone as before, seen by C07; the timing step alone
PASS 73.28 The documents: plan.md (D2, 73.1, 73.2), ui-design.md (Long messages), this file; UTF-8, LF, no BOM, by script
PASS 73.29 The Windows Terminal scenes: W01 to W08 again, 9 runs on this build, each exit 0, 39 assertions passed; W09, R-H's input, maximised and at 120x40, 8 assertions passed
Not verified 73.30: the owner's Human Evaluation, which waits for the independent review. A real paste, double clicks, y and what it pastes back, Shift+click, the glyph and colours by eye, a very large y through OSC 52. Any real CLI turn, which needs its own authorization
Verdict: NOT DONE: corrected and verified, waiting at the independent-review gate. Nothing committed or pushed; the plan is not closed
```

**Executed on the corrected code:** `go build`, `go vet`, `gofmt`; the whole suite, again after the last edit; `-race` on `internal/render`, `internal/input` and `internal/app`, exit 0, no race (259.6, 225.1 and 565.5 s); `FuzzLongMessage`, 60 s; the whole timing step alone
and `TestKeystrokeAllocs`, `TestFrameReuse`, `TestChipFrameAllocs`, on mains power, load 20% before it, no other `go` process; the golden frames; the mutants of the rule and C06, C07; W01 to W09; the comparison above.

**Mutants of the corrected rule.** B01 to B17 through overlays, the tests of the rule alone: the first run killed 16; **B17 survived**, the guard `w = max(w, 1)` removed, which only a width of 0 reaches, as no caller gives: the answer stays right and only the read goes unbounded. I added a check to `TestLongMessageBound`, and the second run killed all 17. Of the plan's mutants, those that touch this rule, M01 to M16, M22, M24, M25 and M69, adapted where the lines changed (M04, M06): 19 killed, and M69, removing nothing and only adding a whole-text count, survives as it did before, a cost that C07 sees: 57 and 42 times slower at 5 MB; C06, 66 and 59 times.

**W09, the one scene whose rendering changed.** A recording outside the repository, three messages of joined characters: `rh-ordinary` and one wide cell, `rh-three` and two, `rh-chip` and three. On the corrected build, maximised and at 120x40: `rh-ordinary` and `…`, `rh-three` and `…`, `…`, then `▣ Long message · 2 lines · 7 KB`; 8 assertions passed. On the reviewed build, for comparison: three chips, the false ones among them. I looked at those three pictures. **W01 to W08,** whose rendering did not change: 104 screen texts of this build compared by script with the same step on the reviewed build, the same but for the clock and the run's folder (`scenes-compared.txt`). I did not look at each of their pictures again.

**Reused, still valid, the code they cover being untouched:** the baseline (P7) and C1; C2 with its reversal and A/A, for the feature against the baseline; the other six fuzz targets; the rest of the mutants, M17 to M21, M23, M26 to M68, and C01 to C05; the flake's proof; everything about Draft, the ranges and the paste's composer. **Kept as they were, failures included:** the block at the head of this report, the first fuzz failure, the first mutant run, the reconciliation's block above and its record of the exception, the logs of the new tests failing on the reviewed code, and the disturbed `rev-1`.

**Deviations to weigh.** D2 and 73.2 now say what the code does; the bound is simpler, and reads a little differently at narrow widths: a tab at 1 cell counts as 1, so up to 7 cells where it read 4. The criterion's wording changed twice in one day, each time openly: the plan says so. Nothing else moved.

## The final end-to-end verification (2026-11-01)

After the independent reviewer approved R-H's correction, the owner authorized the remaining tests through the complete user-facing path: Windows Terminal,
ACC, virtual attachment handling, the actual provider and back to ACC's conversation. It ran on the build of package 4, unchanged: SHA-256 `9EDA74D9…EE6`,
fingerprint `aa7bcf0262892aa4`. No application code changed. All of it is synthetic, marked `P73F-`, with the clipboard overwritten after an announcement,
never saved, restored or read for what it held before.

| Evidence class | What it is here |
|---|---|
| Automated Go tests | `go test ./...` again on the final tree: exit 0. Race, timing, fuzz and mutants earlier, on this build ("The correction of R-H", above), not repeated |
| Mock-session Windows Terminal scenes | W01 to W09, earlier, on this build. Not repeated |
| **Native Windows clipboard and mouse** | T1 to T4, on a mock session: real `Ctrl+V` through Windows Terminal, real `y`, real mouse events by `SendInput`, the real clipboard read back by a separate process |
| **Real provider integration** | One turn each of Claude Code and Codex, through ACC and Windows Terminal, with protocol evidence |

```text
Final verification - 34 scenario runs on build 9EDA74D9, both sizes where it applies: 33 PASS, 1 FAIL
PASS T1 Real paste: 3 lines stay text; 4 lines "▣ Long message · 4 lines · 70 B"; one word of exactly 3 rows stays text in the composer and draws exactly 3 ordinary rows once sent; one cell more "▣ Long message · 1 line · 208 B" (406 B maximised), in the composer and in zone B; 3 lines and a final break "3 lines · 52 B"; 200 lines "200 lines · 5 KB"; TEA_TRACE shows the bracketed paste reaching ACC; nothing submitted by any paste. Maximised and 120x40: 14 runs
PASS T2 Real y, Windows clipboard read back: Unicode, indentation, tabs, blank lines, trailing spaces, 275 bytes, byte-identical. LF fixture; CRLF fixture, 285 B before and 275 B LF after (design D7, no other conversion); and after a sentinel put on the clipboard mid-run, so that only y can have written it. Toast "copied 11 lines · native". TEA_TRACE shows OSC 52 too. Both sizes: 6 runs
PASS T3 Real mouse: one click on zone B's chip opens nothing; a double click opens the full view; Esc returns; neither writes the clipboard. A drag across zone B's chip copies "P73F-ANSWER-BEFORE\n\n<the message byte for byte>\n\nP73F-ANSWER-AFTER", no decoration. A drag across a composer chip copies typed text, the hidden bytes and typed text, byte for byte, nothing submitted. A double click on a composer chip selects and copies its bytes, after a sentinel. Both sizes: 8 runs
PASS T4 A large clipboard, 50,000 lines, 3,649,999 bytes: real paste; Windows Terminal's own warning answered by its button; the chip "▣ Long message · 50,000 lines · 3.6 MB", on the first poll after the answer; a key beside it at once; send; the full view, first and last line reached; y after a sentinel: the clipboard read back byte-identical (CR 0, LF 49,999); the terminal's window answered within 5 ms at every poll, none hung, in the final runs. Both sizes: 2 runs
PASS Rehearsal: the live keys on a mock session, the assembled message read back through y, 3,251 bytes, byte-identical, before a turn was spent
PASS Live Claude Code: model claude-haiku-5-5 by a per-process ANTHROPIC_MODEL, one attempt, one message; the bytes on its stdin equal the expected 3,251, and its own session record holds the exact message; the reply "LINES=60" and "LAST=P73F-LIVE 60 of 60 final marker KESTREL-7319 café 日本語", exact; ACC showed the composer chip, zone B's "▣ Long message · 61 lines · 3 KB" and the reply; no tool, no subagent; the temporary project unchanged; confcheck PASS
PASS Live Codex: the configured model gpt-6-sol, effort xhigh, sandbox readOnly, policy on-request, one attempt, one message; the bytes in turn/start equal the expected 3,251, and its own rollout holds the exact message; the same exact reply; ACC showed the same chips and the reply; no tool, no subagent; the temporary project unchanged; confcheck PASS
FAIL F1 No false message: during a paste over 5 KiB, at the terminal's warning, ACC shows "nothing to paste: no text and no image on the clipboard": on all 7 runs with such a paste, counted or not, on none of the other 34. Older than this plan; below
Not verified natively: right-click paste, Shift+click, draft recovery of a real paste, deleting and undoing several chips, live resizing (these last three by the Go tests and W01 to W09 only); the owner's eye on the glyph and colours; 73.30 stays the owner's to decide on
Verdict: NOT DONE: verified end to end but for F1; waiting at the independent-review gate. Nothing committed or pushed; the plan is not closed
```

**F1, an open defect, not fixed: an application change needs the owner's approval.** Windows Terminal keeps Ctrl+V's press and forwards its release; `ctrlVReleased` (`internal/app/paste.go`, plan 32) then
waits `pasteWindow`, 200 ms, for the paste, and if none has come it reads the clipboard's image; finding none, `clipped` says "nothing to paste: no text and no image on the clipboard". Above 5 KiB Windows
Terminal holds the paste until its warning is answered, so that wait expires before anyone can answer: the message shows behind the warning, is false, as the clipboard holds text, and can still be there beside the chip, for its 3 s. It alters
no byte and blocks nothing. It is not plan 73's: it reproduces on the baseline's build (`x5-toast-baseline`), and the commit that added the message is plan 32's `b56d23e`. But it falls on exactly the long pastes this plan
is about. A clipboard holding an image as well as text would, by the same path, attach the image and then receive the text; that is from reading the code, not tested. **Proposed correction, not implemented:** in the command `pasteImages(true)` runs, read whether the clipboard holds
text; if it does, do nothing, since the terminal has that paste in hand. A few lines and a test in `paste_test.go`, on a code path outside the chip's.

**Windows Terminal's warning is not ACC's.** Every paste over 5 KiB raises "Avertissement, vous êtes sur le point de coller du texte de plus de 5 KiB" (`largePasteWarning`, on by default), so such a paste is not one key: the chip follows at once from "Coller tout de même". T1's largest
paste, 200 lines and 4,799 bytes, raised none. I changed no setting of the terminal's: `settings.json` was last written on 2026-10-28.

**Attempts that do not count, kept with their reason.** They are recorded in each run's `NOTE` and in `results.json`:
- `t4-large`, maximised. Its 70,737 ms to the chip is not ACC, the console or the clipboard: **the owner pressed Enter by hand on the terminal's warning**, the harness's step having looked for the English word "Paste" on a French interface. It failed an assertion of my plan's too, "3.6 MB x" for "3.6 MBx". I first read that delay as a possible slowness of the input path; the owner's report and the screenshot corrected it.
- `t4b-large`, both sizes: all worked, byte-identical; my plan looked for "copied 50,000 lines" where ACC prints "copied 50000 lines · native".
- `t1c-3rows`, both sizes: my fixture held a space, which moved its long word to the next row: 4 rows by `cardText`, so the chip was right (`fixture-rows.txt`). Replaced by `t1g` and `t1h`.
- The first reading of the Claude Code summary called the reply wrong: my reader joined the CLI's local `/usage` answers to the model's. The raw log was never in doubt; `summary-first-reader.json` is kept.

**Provider usage, as the providers report it.**

| | Claude Code 2.1.303 | Codex 0.163.0 |
|---|---|---|
| Model | `claude-haiku-5-5`, per-process override | `gpt-6-sol`, as configured, effort xhigh |
| Turns | 1, 7.5 s | 1 |
| Tokens | 2 input, 1,258 output, 23,042 cache creation, 18,138 cache read | 17,398 input, of which 7,168 cached, 30 output; 17,428 in all |
| Cost | $0.00542, API-equivalent | N/A: Codex reports none, and ACC says so |
| What ACC showed | 42.4k tokens, $0.01, 4% of 1M | 17.4k tokens, N/A, 7% of 258.4k |

**Also to weigh.**
- Keys and mouse events are real to Windows, through `SendInput`, but not from hardware.
- The final clipboard is what the test reads, not each of the two writes: ACC writes the native clipboard and then sends OSC 52, and Windows Terminal may write it again.
- ACC's fresh app data gave Claude Code's Manual mode (`--permission-mode default`), and ACC starts Codex with `sandbox read-only, policy on-request` (`internal/provider/codex/session.go:31`, thread/start); Codex's model and effort are its own configuration's, turn/start naming neither. So a tool call would have stopped for approval; none was made, the logs holding no tool call.
- Claude Code kept the real home, for its own login, so it started the owner's configured MCP servers as on any start: `system/init` lists 5 (1 connected, 1 pending, 3 needing sign-in; their names left out of this record), and 8 of its 39 tools were theirs when the turn ran. The pending one connected after the turn, the last `system/init` listing 246 tools. None was called. One server starts through `npx`, which, given the run's own `LOCALAPPDATA`, downloaded a fresh npm cache of 382 MB into the run's temporary folder: inventoried.
- The Claude Code pill read "Haiku 5.5 Medium · Ultracode", the owner's own setting as the CLI reports it, which I did not touch. No subagent ran.
- Two cosmetic points for the owner's eye: the toast prints "50000" where the chip prints "50,000"; and a drag across zone B's chip highlights the rows around it but not the chip's own row, though its bytes are copied.
- `-race`, the timing step, the fuzz run and the mutants were not repeated: no code changed. The scripts `t4x.sh`, `cpu-watch.ps1` and `mk-t4-sizes.mjs` were written to investigate the delay and never ran, the owner's report having explained it.

## The correction of F1 and the chip's selection (2026-11-02)

The owner's independent review of `acc-plan73-final-verification.zip` approved the core: the native clipboard tests, the mouse, the large paste and both real-provider integrations
show that the feature works and keeps the exact bytes. It asked, before closure, for F1's correction (required), for zone B's selection to light a chip if a small, safe change does it
(recommended), and for a reverification, with no further real CLI turn while the provider send path stays untouched. The criteria went into `plan.md`, The final review's corrections, before the
evidence. F1's FAIL in the section above stays as it was recorded: it describes the previous build.

**What changed: four application files, 58 lines added and 7 removed.**
- `internal/app/paste.go`, +43 −5. When Ctrl+V's wait ends with no paste, `pasteWaited` asks the clipboard for its text, in a command, and `pasteRead` acts on the answer, `pasteTextMsg`. Text other
  than blanks is the terminal's to paste, later than the wait behind its warning, or never if the warning is refused: nothing is read, attached or said. Otherwise `pasteImages(true)`, as before. An
  answer is dropped if a paste or a newer release came since the wait began (`pasteWaiting`).
- `internal/app/app.go`, +2: that message's case.
- `internal/render/select.go`, +11 −1: `highlight` lights a chip's span, the only `refFold`, when the selection copies any of its message.
- `internal/render/text.go`, +2 −1: the comment on `refFold`.

`pasteImages`, `clipped`, Alt+V and the raw Ctrl+V are as they were, and nothing changed under `internal/provider` or `internal/state`. New fingerprint `d7aa8b45d8276c2a` (was `aa7bcf0262892aa4`); new build
SHA-256 `358d81d4feefbcde2561c7c09c5e676f808262b89411abb9428972fc4e31da3b` (was `9eda74d9…6ee6`).

**Why text decides.** It is the terminal's own order (microsoft/terminal, `TerminalPage.cpp`, `doc/research/image-paste.md`): text first, a copied file's path only without it. So the wait now ends as a paste in time
always did: an image or a file beside the text is not touched. **Blanks count as no text** because Windows Terminal 1.23 pastes nothing for them, even bracketed (probe `p1-blank-old`, below), so
today's message stays true there and an image beside them still attaches. The set is the terminal's, not Go's: a no-break space is pasted, so it counts as text.

**The chip's selection needed no redesign, so no trade-off to bring back.** `highlight` runs only while a selection exists; the check is two comparisons of the span's entry with the selection's ends, reading no text, adding
no row. A chip stays no selection's end (`units` is untouched), so a drag ending on its row clamps to its neighbours as before, unlit and not copied. Lit means the selection copies some of the message, which
`TestChipSelectionLit` checks against `Text` itself. The one corner, a selection starting at the very end of the message's last line, would light it with no byte copied; no mouse position gives that place, the message having no row.

```text
Evaluation - Plan 73, the final review's corrections, on build 358d81d4 - 14 criteria: 14 PASS, 0 FAIL. Native runs on mock sessions, no real CLI: 62 judged runs: 45 PASS on the corrected build, 0 FAIL, and 17 BEFORE runs on the previous build by the same plans
PASS 73.F1 No false message. A real paste of 6,569 bytes behind the terminal's warning, by Ctrl+V and by Ctrl+Shift+V, and with the draft window open, maximised and 120x40: "nothing to paste" 0 times in TEA_TRACE's whole run and on no saved screen, its chip or its text there once the warning is answered. The previous build, by the same plans: 1 time in TEA_TRACE and on 2 or 3 of the saved screens. 50,000 lines, 3,649,999 bytes: on no saved screen at either size, and 0 times in a third run's whole TEA_TRACE, 5.9 MB of it
PASS 73.F2 The warning refused. By its button "Annuler": "nothing to paste" 0 times, no paste reached ACC (no bracketed paste in TEA_TRACE), no chip, no image, the draft empty, both sizes. The previous build: the false toast on 3 of 4 saved screens
PASS 73.F3 Normal paste. t1-3lines.txt, its three lines as text; t1-4lines.txt, "▣ Long message · 4 lines · 70 B"; t1-200lines.txt, 4,799 bytes, "▣ Long message · 200 lines · 5 KB", no warning; each a bracketed paste in TEA_TRACE, nothing submitted, no false toast, the clipboard equal, both sizes
PASS 73.F4 Image alone, Ctrl+V. The clipboard holding no text: "pasted-image-1.png · 779 B", no paste in TEA_TRACE, no toast, the draft empty; a second Ctrl+V "pasted-image-1.png: already attached", one chip; both builds, both sizes
PASS 73.F5 Alt+V. An image alone: its chip, the draft empty. Text alone: "no image on the clipboard", no paste, the draft empty. Text and an image: the image's chip, the text in the draft as Ctrl+V had left it. TestAltVReadsNoText: no text read, on any clipboard
PASS 73.F6 A copied image file. Ctrl+V: the terminal pastes its path (TEA_TRACE), and the chip "p73f-image.png · 51 KB", the path not in the draft. Alt+V: the same chip, no paste. Both sizes
PASS 73.F7 Text and an image. Within 5 KiB, Ctrl+V: the text in the draft, no image's chip, one paste; Alt+V then adds the image, the text as it was. Over it: no toast and no image's chip with the warning up or answered, "▣ Long message · 90 lines · 7 KB". The previous build attached the image as well as the chip, at both sizes: what the section above only read from the code, now seen, and gone
PASS 73.F8 The true message stays. An empty clipboard (formats: (empty)) and a clipboard of blanks (formats: UnicodeText, no paste in TEA_TRACE): the toast "nothing to paste: no text and no image on the clipboard", the draft empty, both sizes, both builds
PASS 73.F9 Byte-identical. 44 clipboard read-backs, every one equal to its expected file; y after a sentinel: 275 bytes, 6,569 through the warning, and 3,649,999, SHA-256 9db0d888…51ea, CR 0, LF 49,999. The window answered within 4 ms at every poll, none hung; the chip 165 ms after the warning's answer, maximised
PASS 73.F10 Regression tests. New: TestPasteWaitEnds (10 clipboards), TestPasteHeldBehindWarning (4), TestPasteWaitRaces (3), TestPasteWaitOtherClipboards (7), TestAltVReadsNoText, TestWin32HeldPaste (the real program). On the previous code, recorded: 17 of their 24 subtests and TestWin32HeldPaste fail, the 7 others and TestAltVReadsNoText being what must not change. All pass now. F01 to F15, through go test -overlay: 15 of 15 killed, F12 by the nil dereference it causes
PASS 73.S1 The chip lit. TestChipSelectionLit: 7 selections, lit exactly when the copy holds some of the message, the copy exact; under the cursor the hint unlit; over the search's light; not by another panel's selection. TestChipSearchSelection: a real drag restyles the chip's row, the same words, and Esc restores it. Natively, the selection's ground #4C4F6B: previous build 0 pixels on the chip's row, corrected 3,804 pixels, over its 32 cells, at both sizes; the copy equal to t3-selection-expected.txt, 179 bytes; a click and a double click as before; a composer chip's selection unchanged. S01 to S08 killed. The cost: below
PASS 73.G1 The checks. gofmt -l . empty; go build ./..., go vet ./... and with the timing tag, exit 0; go test ./... exit 0, 1,245 tests pass, 3,447 with subtests, 0 fail, 5 skip on conditions of their own (TestRecoveryChild, TestWriteTestImages, TestTurnLinks, TestWireReplay, TestDocFrames); no golden frame changed; -race on internal/render, internal/input and internal/app: ok, no race; the timing step alone, on mains power: 25 timed tests pass, each within its unchanged threshold, and the 3 guards
PASS 73.G2 The scope. Application code changed only in the four files above, by content against the tree of the final verification; nothing under internal/provider, internal/state, cmd or scripts, nor go.mod, go.sum, AGENTS.md; Windows Terminal's settings.json last written on 2026-10-28, untouched. No real CLI ran. Fingerprint and build above; the tested binary equals a fresh go build of the tree
PASS 73.G3 The record. This section, plan.md, ui-design.md (Pasted images, Selection and copy), doc/research/image-paste.md; the failed attempts below, each kept. brief.md not touched: this review's scope did not name it
Not verified natively: right-click paste; Shift+click; a clipboard held by another program, its failed read by a fake only (TestPasteWaitOtherClipboards); a later Windows Terminal, whose main branch reads differently (doc/research/image-paste.md). The owner's eye on the look, and 73.30, stay the owner's to decide on
Verdict: NOT DONE: corrected and verified, waiting at the independent-review gate. Nothing committed or pushed; the plan is not closed
```

**A frame's cost, with a selection across a chip and without any**, by the same frame at 120x40 through `go test -overlay` (`scripts/zz_chipsel_test.go` in the package), the previous `highlight` and the corrected one alternating, each after a first run left out:

| Round | Selected, previous | Selected, corrected | No selection, previous | No selection, corrected |
|---|---|---|---|---|
| 1 | 316,204 ns | 310,754 ns | 291,474 ns | 265,179 ns |
| 2 | 309,467 ns | 297,327 ns | 273,610 ns | 272,760 ns |
| 3 | 321,402 ns | 294,494 ns | 262,009 ns | 272,943 ns |
| 4 | 309,871 ns | 322,237 ns | 281,550 ns | 269,405 ns |

Allocations 985 and 802 a frame on both (one reading of 984), about 322,080 and 247,375 bytes. The differences go both ways between the pairs and stay inside each side's spread: no reproducible difference (P8).

**The timing step on this build**, alone, on mains power (`doc/evaluation.md`, Keystroke timing), each line as its test printed it; the 25 timed tests pass, each within its unchanged threshold, the search's bar over a very long full view staying R-A's 33 ms:

```text
blame_test.go:485: a key to its frame, 10,000 lines blamed: p50 522.9µs, p95 1.5037ms (target p95 ≤ 16ms)
branches_test.go:888: a key to its frame, 3,000 refs: p50 998.5µs, p95 2.3242ms (target p95 ≤ 16ms)
chipscale_test.go:91: many lines LongMessage       5 KB    2.695µs  500 KB      925ns  5 MB      955ns  5 MB / 5 KB 0.35
chipscale_test.go:91: many lines view              5 KB     1.08µs  500 KB      385ns  5 MB      380ns  5 MB / 5 KB 0.35
chipscale_test.go:91: many lines Left              5 KB      385ns  500 KB       80ns  5 MB      115ns  5 MB / 5 KB 0.30
chipscale_test.go:91: many lines Shift+Left        5 KB      365ns  500 KB      100ns  5 MB      100ns  5 MB / 5 KB 0.27
chipscale_test.go:91: many lines Right             5 KB      315ns  500 KB       80ns  5 MB       80ns  5 MB / 5 KB 0.25
chipscale_test.go:91: many lines Shift+Right       5 KB      335ns  500 KB       75ns  5 MB       80ns  5 MB / 5 KB 0.24
chipscale_test.go:91: many lines Up                5 KB      195ns  500 KB      100ns  5 MB      100ns  5 MB / 5 KB 0.51
chipscale_test.go:91: many lines Shift+Up          5 KB      205ns  500 KB      105ns  5 MB      100ns  5 MB / 5 KB 0.49
chipscale_test.go:91: many lines Down              5 KB      260ns  500 KB      190ns  5 MB       90ns  5 MB / 5 KB 0.35
chipscale_test.go:91: many lines Shift+Down        5 KB      285ns  500 KB      170ns  5 MB       90ns  5 MB / 5 KB 0.32
chipscale_test.go:91: many lines Home              5 KB      100ns  500 KB      130ns  5 MB       80ns  5 MB / 5 KB 0.80
chipscale_test.go:91: many lines Shift+Home        5 KB      310ns  500 KB       85ns  5 MB       80ns  5 MB / 5 KB 0.26
chipscale_test.go:91: many lines End               5 KB      305ns  500 KB       95ns  5 MB       85ns  5 MB / 5 KB 0.28
chipscale_test.go:91: many lines Shift+End         5 KB      325ns  500 KB      125ns  5 MB       80ns  5 MB / 5 KB 0.25
chipscale_test.go:91: many lines WordLeft          5 KB      445ns  500 KB      190ns  5 MB      110ns  5 MB / 5 KB 0.25
chipscale_test.go:91: many lines Shift+WordLeft    5 KB      540ns  500 KB      155ns  5 MB      110ns  5 MB / 5 KB 0.20
chipscale_test.go:91: many lines WordRight         5 KB      335ns  500 KB      110ns  5 MB      105ns  5 MB / 5 KB 0.31
chipscale_test.go:91: many lines Shift+WordRight   5 KB      330ns  500 KB      105ns  5 MB      100ns  5 MB / 5 KB 0.30
chipscale_test.go:91: one line LongMessage         5 KB     1.51µs  500 KB    1.525µs  5 MB    1.415µs  5 MB / 5 KB 0.94
chipscale_test.go:91: one line view                5 KB      955ns  500 KB      440ns  5 MB      295ns  5 MB / 5 KB 0.31
chipscale_test.go:91: one line Left                5 KB       75ns  500 KB       75ns  5 MB       75ns  5 MB / 5 KB 1.00
chipscale_test.go:91: one line Shift+Left          5 KB       85ns  500 KB       85ns  5 MB       75ns  5 MB / 5 KB 0.88
chipscale_test.go:91: one line Right               5 KB      100ns  500 KB       75ns  5 MB       70ns  5 MB / 5 KB 0.70
chipscale_test.go:91: one line Shift+Right         5 KB       75ns  500 KB       90ns  5 MB       75ns  5 MB / 5 KB 1.00
chipscale_test.go:91: one line Up                  5 KB      100ns  500 KB      140ns  5 MB       95ns  5 MB / 5 KB 0.95
chipscale_test.go:91: one line Shift+Up            5 KB      115ns  500 KB      100ns  5 MB       95ns  5 MB / 5 KB 0.83
chipscale_test.go:91: one line Down                5 KB      125ns  500 KB       90ns  5 MB       85ns  5 MB / 5 KB 0.68
chipscale_test.go:91: one line Shift+Down          5 KB      125ns  500 KB       95ns  5 MB       85ns  5 MB / 5 KB 0.68
chipscale_test.go:91: one line Home                5 KB      110ns  500 KB       90ns  5 MB       75ns  5 MB / 5 KB 0.68
chipscale_test.go:91: one line Shift+Home          5 KB      100ns  500 KB       80ns  5 MB       75ns  5 MB / 5 KB 0.75
chipscale_test.go:91: one line End                 5 KB       75ns  500 KB       75ns  5 MB      165ns  5 MB / 5 KB 2.20
chipscale_test.go:91: one line Shift+End           5 KB       75ns  500 KB       75ns  5 MB      120ns  5 MB / 5 KB 1.60
chipscale_test.go:91: one line WordLeft            5 KB      105ns  500 KB      105ns  5 MB      105ns  5 MB / 5 KB 1.00
chipscale_test.go:91: one line Shift+WordLeft      5 KB      130ns  500 KB      145ns  5 MB      105ns  5 MB / 5 KB 0.81
chipscale_test.go:91: one line WordRight           5 KB      100ns  500 KB      145ns  5 MB      100ns  5 MB / 5 KB 1.00
chipscale_test.go:91: one line Shift+WordRight     5 KB       95ns  500 KB      130ns  5 MB       95ns  5 MB / 5 KB 1.00
committiming_test.go:59: message field at 120x40, 400 keys after 50 warm-up, 18 lines typed: p50 0s, p95 1.1132ms, max 9.3388ms (target p95 ≤ 16ms)
committiming_test.go:59: message field at 208x59, 400 keys after 50 warm-up, 18 lines typed: p50 587.6µs, p95 1.9994ms, max 8.804ms (target p95 ≤ 16ms)
difftiming_test.go:61: 120x40: big.go opened, compared and laid out in 319.1813ms (in a command, the frames going on)
difftiming_test.go:76: diff view, wide false (side by side false), at 120x40, 400 keys after 50 warm-up: p50 0s, p95 1.0145ms, max 1.6299ms (target p95 ≤ 16ms)
difftiming_test.go:76: diff view, wide true (side by side false), at 120x40, 400 keys after 50 warm-up: p50 0s, p95 1.0128ms, max 9.3639ms (target p95 ≤ 16ms)
difftiming_test.go:61: 208x59: big.go opened, compared and laid out in 321.4138ms (in a command, the frames going on)
difftiming_test.go:76: diff view, wide false (side by side false), at 208x59, 400 keys after 50 warm-up: p50 0s, p95 1.0756ms, max 6.5062ms (target p95 ≤ 16ms)
difftiming_test.go:76: diff view, wide true (side by side true), at 208x59, 400 keys after 50 warm-up: p50 0s, p95 1.0864ms, max 2.3605ms (target p95 ≤ 16ms)
graph_test.go:755: the history kept: 209 bytes a commit: 91 its entry (88 bytes, the slice's room included), 8 its row, 49 its subject's text, 61 the index by hash and the lanes
graph_test.go:767: a key to its frame, 50,000 commits: p50 0s, p95 1.006ms (target p95 ≤ 16ms)
hometiming_test.go:90: banner frame to frame, 208x59, 1000 frames after 50 warm-up: p50 716.6µs, p95 1.7275ms, max 3.763ms (target p95 ≤ 2ms); 11355 bytes a frame
hometiming_test.go:124: key to frame in the project field, banner running, 208x59, 400 keys: p50 77.9µs, p95 174.3µs, max 695µs (target p95 ≤ 16ms)
hometiming_test.go:149: the picture on entering, 208x59, the size's 120ms wait included: p50 182.8402ms, max 189.4299ms (target ≤ 260ms, the terminal's 40 ms apart)
keystroke_test.go:51: keystroke-to-frame, 10,000-line draft, 800 keys after 50 warm-up: p50 0s, p95 1.0221ms, max 1.9993ms (target p95 ≤ 16ms)
keystroke_test.go:70: keystroke-to-frame in the draft window, 10,000-line draft, 800 keys: p50 0s, p95 1.0477ms, max 2.5102ms (target p95 ≤ 16ms)
keystroke_test.go:119: model selector, Alt+P at 120x40, 400 keys after 50 warm-up: p50 0s, p95 999.5µs, max 1.1132ms (target p95 ≤ 16ms)
keystroke_test.go:119: model selector, ↑↓ in the expanded list at 120x40, 400 keys after 50 warm-up: p50 0s, p95 1.1025ms, max 1.6649ms (target p95 ≤ 16ms)
keystroke_test.go:119: model selector, Alt+P at 208x59, 400 keys after 50 warm-up: p50 0s, p95 1.1855ms, max 6.0311ms (target p95 ≤ 16ms)
keystroke_test.go:119: model selector, ↑↓ in the expanded list at 208x59, 400 keys after 50 warm-up: p50 0s, p95 1.0437ms, max 2.2179ms (target p95 ≤ 16ms)
keystroke_test.go:183: sidebar hover, the panel shown false, icons false, at 120x40, 400 reports after 50 warm-up: p50 505.7µs, p95 1.1173ms, max 4.0394ms (target p95 ≤ 16ms)
keystroke_test.go:202: idle motion, the panel shown false, icons false, at 120x40: 800 of 800 reports dropped by the filter before Update and View; p50 0s, p95 0s, max 600.4µs
keystroke_test.go:183: sidebar hover, the panel shown true, icons false, at 120x40, 400 reports after 50 warm-up: p50 0s, p95 680.2µs, max 7.3981ms (target p95 ≤ 16ms)
keystroke_test.go:202: idle motion, the panel shown true, icons false, at 120x40: 800 of 800 reports dropped by the filter before Update and View; p50 0s, p95 0s, max 526.3µs
keystroke_test.go:183: sidebar hover, the panel shown false, icons false, at 208x59, 400 reports after 50 warm-up: p50 0s, p95 1.5268ms, max 8.6679ms (target p95 ≤ 16ms)
keystroke_test.go:202: idle motion, the panel shown false, icons false, at 208x59: 800 of 800 reports dropped by the filter before Update and View; p50 0s, p95 0s, max 717.1µs
keystroke_test.go:183: sidebar hover, the panel shown true, icons false, at 208x59, 400 reports after 50 warm-up: p50 587.1µs, p95 1.5119ms, max 2.1725ms (target p95 ≤ 16ms)
keystroke_test.go:202: idle motion, the panel shown true, icons false, at 208x59: 800 of 800 reports dropped by the filter before Update and View; p50 0s, p95 0s, max 1.0158ms
keystroke_test.go:183: sidebar hover, the panel shown false, icons true, at 120x40, 400 reports after 50 warm-up: p50 0s, p95 1.2622ms, max 2.398ms (target p95 ≤ 16ms)
keystroke_test.go:202: idle motion, the panel shown false, icons true, at 120x40: 800 of 800 reports dropped by the filter before Update and View; p50 0s, p95 0s, max 0s
keystroke_test.go:183: sidebar hover, the panel shown true, icons true, at 120x40, 400 reports after 50 warm-up: p50 0s, p95 1.0544ms, max 2.6751ms (target p95 ≤ 16ms)
keystroke_test.go:202: idle motion, the panel shown true, icons true, at 120x40: 800 of 800 reports dropped by the filter before Update and View; p50 0s, p95 0s, max 0s
keystroke_test.go:183: sidebar hover, the panel shown false, icons true, at 208x59, 400 reports after 50 warm-up: p50 504.6µs, p95 1.4701ms, max 2.0088ms (target p95 ≤ 16ms)
keystroke_test.go:202: idle motion, the panel shown false, icons true, at 208x59: 800 of 800 reports dropped by the filter before Update and View; p50 0s, p95 0s, max 0s
keystroke_test.go:183: sidebar hover, the panel shown true, icons true, at 208x59, 400 reports after 50 warm-up: p50 0s, p95 1.5358ms, max 3.2369ms (target p95 ≤ 16ms)
keystroke_test.go:202: idle motion, the panel shown true, icons true, at 208x59: 800 of 800 reports dropped by the filter before Update and View; p50 0s, p95 0s, max 1.0105ms
keystroke_test.go:256: keystroke-to-frame, 50,000-line draft, 800 keys after 50 warm-up, 9 saves: p50 1.0045ms, p95 3.1153ms, max 5.958ms (target p95 ≤ 16ms)
keystroke_test.go:289: 10000-line draft: 6057 keys, 218.221µs per key, 449137 B and 598 allocations per key
keystroke_test.go:289: 50000-line draft: 2250 keys, 677.754µs per key, 1453933 B and 598 allocations per key
longmessage_timing_test.go:52: a 50,000-line chip in view (3349999 bytes), 400 deltas to their frames at 200x60: p50 950µs, p95 1.3437ms, max 2.1676ms (target p95 ≤ 16ms)
longmessage_timing_test.go:93: around a 50,000-line chip, the motions, 480 keys at 200x60: p50 0s, p95 1.0459ms, max 1.9866ms (target p95 ≤ 16ms)
longmessage_timing_test.go:99: around a 50,000-line chip, typing beside it, 400 keys at 200x60: p50 999.9µs, p95 2.0213ms, max 3.52ms (target p95 ≤ 16ms)
longmessage_timing_test.go:101: around a 50,000-line chip, its whole deletion and its undo, 200 keys at 200x60: p50 0s, p95 1.2804ms, max 2.1189ms (target p95 ≤ 16ms)
longmessage_timing_test.go:120: the full view of 50,000 lines at 120x40, its one wrap, on opening: 54.7182ms
longmessage_timing_test.go:129: the full view of 50,000 lines at 120x40, 200 keys: p50 0s, p95 0s, max 1.4232ms (target p95 ≤ 16ms)
longmessage_timing_test.go:140: y's key in Update at 120x40, 100 presses: p50 0s, p95 0s, max 0s (target ≤ 16ms)
longmessage_timing_test.go:150: y's sequence, 4466676 bytes: its copy on the event loop 2.1625ms
longmessage_timing_test.go:171: that view with the search's bar open at 120x40, 40 keys: p50 33.6644ms, p95 39.9829ms, max 46.8159ms
longmessage_timing_test.go:120: the full view of 50,000 lines at 208x59, its one wrap, on opening: 78.7055ms
longmessage_timing_test.go:129: the full view of 50,000 lines at 208x59, 200 keys: p50 0s, p95 999.4µs, max 1.6616ms (target p95 ≤ 16ms)
longmessage_timing_test.go:140: y's key in Update at 208x59, 100 presses: p50 0s, p95 0s, max 999.9µs (target ≤ 16ms)
longmessage_timing_test.go:150: y's sequence, 4466676 bytes: its copy on the event loop 3.042ms
longmessage_timing_test.go:171: that view with the search's bar open at 208x59, 40 keys: p50 33.0734ms, p95 36.21ms, max 37.781ms
longsession_test.go:111: 26209 events over 2h0m0s of session time, applied in 654ms (40045 events/s), a frame every 50
longsession_test.go:112: heap in use 10.7 MiB after (2.5 MiB before), 117 GC cycles
longsession_test.go:113: kept: conversation 5281, stages 240, agents 720, tools 6960, diffs 20, log 2000, errors 48, notes 1
longsession_test.go:115: frames during: first tenth p95 3.5814ms, last tenth p95 2.0079ms
longsession_test.go:117: delta-to-frame at the end, 200x60, 400 deltas: p50 1.0048ms, p95 2.2603ms, max 4.0144ms (target p95 ≤ 16ms)
outputtiming_test.go:45: delta-to-frame, a command's output in 8192 deltas of 8192 bytes (64 MiB): p50 0s, p95 1.0173ms, max 2.9791ms (target p95 ≤ 16ms)
perf_test.go:137: paste-to-frame in the composer, 50,000 lines (3350000 bytes), 10 runs: p50 541.3µs, max 664.9µs (target ≤ 100ms)
perf_test.go:137: paste-to-frame with the draft window open, 50,000 lines (3350000 bytes), 10 runs: p50 1.4959ms, max 6.2527ms (target ≤ 100ms)
perf_test.go:201: delta-to-frame, 2000-line answer, one line per delta: p50 552.7µs, p95 1.669ms, max 3.4042ms (target p95 ≤ 16ms)
review_test.go:548: a key to its frame, a turn of 1,000 files unfolded: p50 952.9µs, p95 1.5321ms (target p95 ≤ 16ms)
searchtiming_test.go:110: search, two hours (5282 entries), a key in the bar to its frame at 200x60, 407 keys after 50 warm-up: p50 3.5088ms, p95 5.9983ms, max 7.2678ms (target p95 ≤ 16ms)
searchtiming_test.go:138: search, two hours, the bar open, a delta to its frame at 200x60, 400 deltas: p50 2.9996ms, p95 5.5287ms, max 9.5245ms (target p95 ≤ 16ms)
searchtiming_test.go:110: search, answers ten times longer (5282 entries), a key in the bar to its frame at 200x60, 407 keys after 50 warm-up: p50 7.7238ms, p95 13.8881ms, max 19.9486ms (target p95 ≤ 16ms)
thinkframe_test.go:113: conversation 5522 entries, 721 of them thinking; card frames at 200x60, 80 over 4 s: p50 1.0049ms, p95 3.004ms, max 14.0749ms (target p95 ≤ 16ms)
toolstiming_test.go:85: event-to-frame, 2400 calls (30 Write calls of 98 KB) in one open turn, 8 agents running, 200x60, the last 481 events: p50 1.0165ms, p95 2.5476ms, max 4.5322ms (target p95 ≤ 16ms)
```

The guards: TestKeystrokeAllocs, TestFrameReuse and TestChipFrameAllocs pass:

```text
framecost_test.go:56: ctrl+home: 599 allocations per keystroke (bound 1200)
framecost_test.go:56: ctrl+end: 565 allocations per keystroke (bound 1200)
framecost_test.go:56: ctrl+home in the draft window: 1002 allocations per keystroke (bound 1200)
framecost_test.go:56: ctrl+end in the draft window: 872 allocations per keystroke (bound 1200)
longmessage_test.go:1437: a frame with a chip of four lines: 507 allocations; with a three-row message: 523
longmessage_test.go:1437: a frame with a chip of 404 lines: 509 allocations; with a three-row message: 523
longmessage_test.go:1437: a frame with a chip of 50,000 lines: 519 allocations; with a three-row message: 523
longmessage_test.go:1437: a frame with a chip of one 3 MB line: 508 allocations; with a three-row message: 523
```

**The mutants**, each one replacement through `go test -overlay`, the tree never written:

| Mutant | The rule broken | Result | By |
|---|---|---|---|
| F01 | the wait goes straight to the images | KILLED | `TestPasteWaitEnds`, `TestPasteHeldBehindWarning`, `TestPasteWaitOtherClipboards`, `TestWin32HeldPaste` |
| F02 | text on the clipboard ignored | KILLED | `TestPasteWaitEnds`, `TestPasteHeldBehindWarning`, `TestPasteWaitOtherClipboards`, `TestWin32HeldPaste` |
| F03 | the answer's wait not checked | KILLED | `TestPasteWaitRaces` |
| F04 | blanks count as text | KILLED | `TestPasteWaitEnds` |
| F05 | Go's blanks, not the terminal's | KILLED | `TestPasteWaitEnds` |
| F06 | a failed read counts as text | KILLED | `TestPasteCtrlV`, `TestWin32PasteRelease`, `TestPasteWaitEnds`, `TestPasteWaitOtherClipboards` |
| F07 | the blanks lack \v and \f | KILLED | `TestPasteWaitEnds` |
| F08 | the images asked for as Alt+V does | KILLED | `TestPasteWaitEnds`, `TestPasteWaitOtherClipboards` |
| F09 | an older wait still counts | KILLED | `TestPasteWaitRaces` |
| F10 | a paste in the wait does not end it | KILLED | `TestPasteWaitRaces` |
| F11 | a draft no longer taking pastes | KILLED | `TestBranchesKeys` |
| F12 | no clipboard, no check | KILLED | the nil dereference it causes, in `TestPasteWaitOtherClipboards` |
| F13 | Alt+V reads the text too | KILLED | `TestPasteWaitEnds`, `TestAltVReadsNoText` |
| F14 | the answer not handled | KILLED | `TestPasteCtrlV`, `TestWin32PasteRelease`, `TestPasteWaitEnds`, `TestPasteWaitRaces`, `TestPasteWaitOtherClipboards` |
| F15 | the text attached as the image | KILLED | `TestPasteCtrlV`, `TestPasteWaitEnds` |
| S01 | the chip never lit | KILLED | `TestChipSelectionLit`, `TestChipSearchSelection` |
| S02 | lit whatever the selection's start | KILLED | `TestChipSelectionLit` |
| S03 | lit whatever the selection's end | KILLED | `TestChipSelectionLit` |
| S04 | not lit from inside the message | KILLED | `TestChipSelectionLit` |
| S05 | not lit to inside the message | KILLED | `TestChipSelectionLit` |
| S06 | lit by a selection up to its start | KILLED | `TestChipSelectionLit` |
| S07 | lit by another panel's selection | KILLED | `TestChipSelectionLit` |
| S08 | lit in another style | KILLED | `TestChipSelectionLit` |

**Every judged native run**, 62: 45 PASS on the corrected build and 17 BEFORE runs on the previous one, by the same plans. BEFORE means the run showed what its checks expected of that build, the false toast or the accidental image among them:

| Scenario | Build | Size | Verdict |
|---|---|---|---|
| F1 a paste over 5 KiB, the warning answered | previous | 208x59 | BEFORE |
| F1 a paste over 5 KiB, the warning answered | corrected | 208x59 | PASS |
| F2 the same paste, the warning refused | previous | 208x59 | BEFORE |
| F2 the same paste, the warning refused | corrected | 208x59 | PASS |
| F7 text over 5 KiB and an image | previous | 208x59 | BEFORE |
| F7 text over 5 KiB and an image | corrected | 208x59 | PASS |
| F1, F9 50,000 lines | corrected | 208x59 | PASS |
| F3 normal paste, 3 lines | corrected | 208x59 | PASS |
| F3 normal paste, 4 lines | corrected | 208x59 | PASS |
| F3 normal paste, 200 lines, 4,799 bytes | corrected | 208x59 | PASS |
| F9 y after a sentinel, 275 bytes | corrected | 208x59 | PASS |
| F9 y after a sentinel, 6,569 bytes through the warning | corrected | 208x59 | PASS |
| F4 an image alone, Ctrl+V | previous | 208x59 | BEFORE |
| F4 an image alone, Ctrl+V | corrected | 208x59 | PASS |
| F5 an image alone, Alt+V | corrected | 208x59 | PASS |
| F6 a copied image file, Ctrl+V | corrected | 208x59 | PASS |
| F6 a copied image file, Alt+V | corrected | 208x59 | PASS |
| F7 text within 5 KiB and an image, Ctrl+V then Alt+V | corrected | 208x59 | PASS |
| F5 text alone, Alt+V | corrected | 208x59 | PASS |
| F8 an empty clipboard, Ctrl+V | corrected | 208x59 | PASS |
| F8 a clipboard of blanks, Ctrl+V | corrected | 208x59 | PASS |
| S1 a click and a double click on zone B's chip | corrected | 208x59 | PASS |
| S1 a selection across a composer chip, unchanged | corrected | 208x59 | PASS |
| F1 a paste over 5 KiB, the warning answered | previous | 120x40 | BEFORE |
| F1 a paste over 5 KiB, the warning answered | corrected | 120x40 | PASS |
| F2 the same paste, the warning refused | previous | 120x40 | BEFORE |
| F2 the same paste, the warning refused | corrected | 120x40 | PASS |
| F7 text over 5 KiB and an image | previous | 120x40 | BEFORE |
| F7 text over 5 KiB and an image | corrected | 120x40 | PASS |
| F1, F9 50,000 lines | corrected | 120x40 | PASS |
| F3 normal paste, 3 lines | corrected | 120x40 | PASS |
| F3 normal paste, 4 lines | corrected | 120x40 | PASS |
| F3 normal paste, 200 lines, 4,799 bytes | corrected | 120x40 | PASS |
| F9 y after a sentinel, 275 bytes | corrected | 120x40 | PASS |
| F9 y after a sentinel, 6,569 bytes through the warning | corrected | 120x40 | PASS |
| F4 an image alone, Ctrl+V | previous | 120x40 | BEFORE |
| F4 an image alone, Ctrl+V | corrected | 120x40 | PASS |
| F5 an image alone, Alt+V | corrected | 120x40 | PASS |
| F6 a copied image file, Ctrl+V | corrected | 120x40 | PASS |
| F6 a copied image file, Alt+V | corrected | 120x40 | PASS |
| F7 text within 5 KiB and an image, Ctrl+V then Alt+V | corrected | 120x40 | PASS |
| F5 text alone, Alt+V | corrected | 120x40 | PASS |
| F8 an empty clipboard, Ctrl+V | corrected | 120x40 | PASS |
| F8 a clipboard of blanks, Ctrl+V | corrected | 120x40 | PASS |
| S1 a click and a double click on zone B's chip | corrected | 120x40 | PASS |
| S1 a selection across a composer chip, unchanged | corrected | 120x40 | PASS |
| F1 50,000 lines, the whole stream | corrected | 120x40 | PASS |
| F1 Ctrl+Shift+V, the warning answered | corrected | 208x59 | PASS |
| F1 the draft window open, the warning answered | corrected | 208x59 | PASS |
| F1 Ctrl+Shift+V, the warning answered | corrected | 120x40 | PASS |
| F1 the draft window open, the warning answered | corrected | 120x40 | PASS |
| F1 Ctrl+Shift+V, the warning answered | previous | 120x40 | BEFORE |
| F1 the draft window open, the warning answered | previous | 120x40 | BEFORE |
| F5 an image alone, Alt+V | previous | 120x40 | BEFORE |
| F6 a copied image file, Ctrl+V | previous | 120x40 | BEFORE |
| F7 text within 5 KiB and an image, Ctrl+V then Alt+V | previous | 120x40 | BEFORE |
| F8 an empty clipboard, Ctrl+V | previous | 120x40 | BEFORE |
| F8 a clipboard of blanks, Ctrl+V | previous | 120x40 | BEFORE |
| S1 a selection across zone B's chip | previous | 208x59 | BEFORE |
| S1 a selection across zone B's chip | corrected | 208x59 | PASS |
| S1 a selection across zone B's chip | previous | 120x40 | BEFORE |
| S1 a selection across zone B's chip | corrected | 120x40 | PASS |

**Attempts kept, none counted, each with its log in the package.**
- **The first run of the new tests on the previous code** (`new-tests-before-the-fix.log`) ended in a panic, "deadlock: main bubble goroutine has exited": `TestPasteWaitOtherClipboards` set the rig's session to nil, leaving its goroutine stranded at the rig's end. The test's bug, not the application's: it now restores the session, and `new-tests-before-the-fix-2.log` is the complete record.
- **Five native runs, `c7-empty-*` and `c8-blank-*`**, exited 1: my plan's expected text held a colon, the harness's step separator, and it refused the plan before opening a window. Run again as `c7b-empty` and `c8b-blank`, the exact toast judged from the saved screen.
- **`crop8.ps1` stopped once** on one of those runs' old names, its partial folder `shots` kept; the screenshots come from `shots-2` and `shots-3`.
- **Two probes on the previous build before the fix was designed**: `p1-blank-old` settled that blanks paste nothing, and `p2-image-old` that the image path works, a 779-byte PNG of a smooth gradient, its thumbnail right.
- **The package's first stage stopped at its own private scan**, which allowed a short label of the proxy's host only inside one ordinary English word: the suite's log holds it in a second one. Checked with the label masked, the host itself nowhere; the rule now names both words, and stays as strict otherwise. A patch script's line broken by the Bash tool's handling of backslashes had stopped it once before that, with nothing staged.
- **A count in a progress message, not in a record.** One message to the owner said 61 judged runs, before the last run on the previous build was judged. The count is 62, as the owner's correction says: 45 PASS on the corrected build and 17 BEFORE runs.

**Also to weigh.**
- **A later Windows Terminal may need one more look.** Its `main` branch reads as sending an empty bracketed paste when there is no text, so that an application can tell the user pasted. If a release does that, the dashboard counts it as the terminal's paste and Ctrl+V stops attaching an image alone; Alt+V still does. Not measured and not changed: 1.23 does not do it, and it is for the check after a Terminal update.
- `TestDelivery` on both real adapters is in the suite and passes; no real turn was spent on this, as the owner's review allowed, the send path being untouched.
- The toast's "50000" against the chip's "50,000" stays: the review did not ask for it.
- The timing step's conditions record no CPU load: the system's reading came back empty that time. Power and the applications open are recorded.

## Approval and closure (2026-11-02)

**Approved.** After the independent review, the owner wrote: "The independent review of `acc-plan73-f1-review.zip` is APPROVED.", and "I also explicitly approve the Human Evaluation (73.30), based on the screenshots, native Windows
Terminal tests, automated tests, and real-provider verification already provided. No additional manual testing or Human Evaluation session is required. Record 73.30 as `PASS — Owner approved`." **Plan 73: CLOSED.**

So 73.30 rests on that evidence and on the owner's approval. The seven scenarios of the Manual Human Evaluation, kept in The plan as it ran, were not run by hand as a session, and this file does not say they were.

**The one correction the owner asked for.** The count of native runs is 62 judged runs: 45 PASS on the corrected build and 17 BEFORE runs. The correction's block now says so. The 61 stood in one progress message and in no record:
no tracked file and nothing in `acc-plan73-f1-review.zip` said it, and that package's own numbers, 62, 45 and 17, stand. As the owner asked, it was not built again.

```text
Evaluation - Plan 73, virtual attachments for long user messages, final - 30 criteria and the final review's 14: 44 PASS, 0 FAIL
PASS 73.1 to 73.29 As reported in The evaluation, read with The independent review and with The correction of R-H, whose block gives the results on the corrected rule
PASS — Owner approved 73.30 The owner's Human Evaluation, 2026-11-02, on the evidence in this file, no manual session
PASS 73.F1 to 73.F10, 73.S1, 73.G1 to 73.G3 As reported in The correction of F1 and the chip's selection, on the final build, fingerprint d7aa8b45d8276c2a, SHA-256 358d81d4feefbcde2561c7c09c5e676f808262b89411abb9428972fc4e31da3b
F1's FAIL stays recorded in The final end-to-end verification, on the build before its correction; 73.F1 is its reverification. That section's two real turns, one per CLI, were not repeated: the provider send path did not change
Verdict: DONE: approved by the owner on 2026-11-02
```

### The reviews

The reviewers' own reports are not in this repository: each verdict is recorded as the owner reported it. They worked from these packages, built outside the repository and kept in the owner's Downloads folder, through inspection of their code, tests, documents and evidence.

| Package | SHA-256 | Verdict |
|---|---|---|
| `acc-plan73-review-1.zip` | `D11CB4A6CF1EAA112850945BBCF03E5A1ADE08A33C5A242600303DE1A99DF7BE` | CHANGES REQUIRED, in documentation and criteria only |
| `acc-plan73-review-2.zip` | `46106C03DFC62B13D37CE05E69C1A826A1A991837E26735D19CC0FF29B685B4A` | APPROVED WITH RESERVATIONS, on how performance regressions are read |
| `acc-plan73-review-3.zip` | `2A3E8781B283C095D18425B89407E1F2BCB96F76541643F8F3E494033F93A11E` | APPROVED WITH RESERVATIONS, for the owner's Human Evaluation only; then R-H, to be corrected first |
| `acc-plan73-review-4.zip` | `F726694AB113A686594CD61F2CC98F224BDAC0597233CC985B27A03203A89CD5` | R-H's correction approved |
| `acc-plan73-final-verification.zip` | `3552951DE664A6B074E818E759012A159DB6DC7EFCA0E112B4E0EAA4B8D8BDB7` | The core approved; F1's correction required, the chip's selection recommended |
| `acc-plan73-f1-review.zip` | `6B2E07D8F83AE705FD1DB00E7FFA2187FD3F1E61BE58818EC3D07AB8351D42F6` | APPROVED, with one documentation correction |

### Still documented, none a task unless the owner asks

The owner's approval names none of these and reopens no decision; each stands as this file writes it. R-A, the search's bar over a very long full view, about 33 ms a frame; R-B, the full view's first frame; R-C, P8's small differences, inconclusive;
R-F, the composer's cost with many chips, measured and not budgeted; R-I, the bound counts cells, not bytes. Not driven natively: right-click paste, Shift+click, a clipboard held by another program, and draft recovery of a real paste, deleting and undoing several chips, and
live resizing, which rest on the Go tests and W01 to W09. To check after a Windows Terminal update: an empty bracketed paste, which its `main` branch reads as sending (The correction of F1, Also to weigh; `doc/research/image-paste.md`). The toast's "50000" against the chip's "50,000".

### The closure

As the owner asked, no implementation changed for the closure, and no provider test, Human Evaluation or review package was repeated. The application code, its tests, the golden frames and the other documents are what the reviewer received, byte for byte,
compared file by file with the tree `acc-plan73-f1-review.zip`'s patches give; only this file, `plan.md` and `AGENTS.md`'s status differ. On the closure's tree: `gofmt -l .` empty; `go build ./...`, `go vet ./...` and `go test ./...` exit 0, 1,245 tests pass, 3,447 with subtests, 0 fail, 5 skip on conditions of their own; a fresh `go build` gives the tested binary, SHA-256 `358d81d4feefbcde2561c7c09c5e676f808262b89411abb9428972fc4e31da3b`; the fingerprint `d7aa8b45d8276c2a`; `git diff --check` exit 0; the private scan of the 86 changed files, by counts: no user name, computer or domain name, proxy host or git email outside the repository's own identity, a label of the proxy's host only inside ordinary words.

`plan.md`'s section of plan 73, its design, criteria, steps and progress, is kept word for word under The plan as it ran, but for the count above, as the plan's one commit holds it, `plan.md` keeping the current state only. `doc/brief.md` holds
amendments 76 and 77; the later reviews are recorded here, not there, as they were in their turn. The commit, "Plan 73: Virtual attachments for long user messages", holds the whole plan, its code, tests, golden frames and documents; it is made on the owner's approval
(`AGENTS.md`, Working rules), under the repository's own identity and with no AI attribution line. Its hash is git's: a commit cannot hold its own.

The temporary testing infrastructure stays where it is, as the owner asked, inventoried in the last two packages (`inventory/temporary-infrastructure.md`). Its cleanup is for the owner to authorize separately (`doc/evaluation.md`, Evidence and temporary files): what matters of it is in this file.

## The baseline (step 2, P7, A1)

Run on 2026-11-01 before any test or application code changed: `HEAD` `1c36ceb`,
no change under `cmd`, `internal`, `scripts`, `testdata`, `third_party`, `go.mod` or
`go.sum`; only the plan's documents differed. Go 1.27.2, the whole timing step of
`doc/evaluation.md` alone, then `TestKeystrokeAllocs` and `TestFrameReuse`.

Conditions: on mains power, CPU load 15% of 12 logical processors just before the
run, no other `go` process; open and idle: the browser, Teams, Outlook, Notepad,
Calculator, Settings, Windows Terminal and Claude. All 21 timed tests and both guards
passed (81.7 s, then 3.6 s).

**The clock.** Go's clock steps by about 0.5 ms on this machine (`doc/evaluation.md`,
plan 51), so a p50 of `0s` means under one step, and times near a step are coarse.
`TestKeystrokeCost`'s means, over thousands of keys, and the allocation counts are the
sharper signals for a small change.

| Test | What it measures | p50 | p95 | Max | Target |
|---|---|---:|---:|---:|---:|
| `TestKeystrokeToFrame` | 10,000-line draft | 0 | 1.045 ms | 4.03 ms | p95 ≤ 16 ms |
| `TestKeystrokeToFrame` | in the draft window | 0 | 1.116 ms | 2.16 ms | p95 ≤ 16 ms |
| `TestDraftKeystrokes` | 50,000-line draft, 8 saves | 1.015 ms | 3.011 ms | 5.62 ms | p95 ≤ 16 ms |
| `TestPasteToFrame` | 50,000-line paste, 3.35 MB | 1.145 ms | | 1.94 ms | max ≤ 100 ms |
| `TestStreamToFrame` | 2,000-line answer | 0.600 ms | 1.991 ms | 14.13 ms | p95 ≤ 16 ms |
| `TestLongSession` | two hours, 400 deltas at 200x60 | 1.506 ms | 3.023 ms | 4.78 ms | p95 ≤ 16 ms |
| `TestThinkingFrame` | 5,522 entries, card frames | 1.023 ms | 3.060 ms | 11.42 ms | p95 ≤ 16 ms |
| `TestSearchToFrame` | two hours, a key in the bar | 3.030 ms | 4.998 ms | 5.66 ms | p95 ≤ 16 ms |
| `TestSearchToFrame` | the bar open, a delta | 2.097 ms | 3.996 ms | 5.51 ms | p95 ≤ 16 ms |
| `TestSearchToFrame` | answers ten times longer | 7.083 ms | 9.526 ms | 12.04 ms | p95 ≤ 16 ms |
| `TestToolsToFrame` | 2,400 calls in one turn | 1.006 ms | 2.548 ms | 4.05 ms | p95 ≤ 16 ms |
| `TestOutputToFrame` | 64 MiB of output | 0 | 1.024 ms | 10.26 ms | p95 ≤ 16 ms |
| `TestSelectorToFrame` | ↑↓ in the list at 208x59 | 0 | 1.583 ms | 14.55 ms | p95 ≤ 16 ms |
| `TestMessageToFrame` | message field at 208x59 | 0.555 ms | 1.568 ms | 6.75 ms | p95 ≤ 16 ms |
| `TestDiffToFrame` | wide, at 208x59 | 0 | 1.068 ms | 1.90 ms | p95 ≤ 16 ms |
| `TestMotionToFrame` | hover, panel shown, icons, 208x59 | 0.545 ms | 1.645 ms | 3.35 ms | p95 ≤ 16 ms |
| `TestBranchesFrameCost` | 3,000 refs | 0.988 ms | 1.950 ms | | p95 ≤ 16 ms |
| `TestBlameFrameCost` | 10,000 lines blamed | 0.503 ms | 1.142 ms | | p95 ≤ 16 ms |
| `TestGraphFrameCost` | 50,000 commits | 0 | 1.024 ms | | p95 ≤ 16 ms |
| `TestTurnReviewFrameCost` | a turn of 1,000 files | 0.506 ms | 1.508 ms | | p95 ≤ 16 ms |
| `TestBannerToFrame` | 1,000 banner frames | 0.676 ms | 1.606 ms | 2.91 ms | p95 ≤ 2 ms |
| `TestHomeKeysToFrame` | the project field | 0.076 ms | 0.179 ms | 0.84 ms | p95 ≤ 16 ms |
| `TestHomePictureTime` | the picture on entering | 181.5 ms | | 196.4 ms | max ≤ 260 ms |

`TestKeystrokeCost`, which judges nothing: 10,000 lines, 222.0 µs and 598 allocations a
key, 449,083 B; 50,000 lines, 542.7 µs and 598 allocations a key, 1,455,067 B.

`TestKeystrokeAllocs`, its bound 1,200 allocations a keystroke: 599 at the top and 565
at the bottom of a 10,000-line draft; 1,002 and 872 in the draft window.

## C1: the original fill against the replacement fill (step 3, P7, A2, A3)

Both test binaries were built before any application code changed, on `1c36ceb`'s
application code: `orig.test.exe` with the original tests (SHA-256 `a565485c…`),
`repl.test.exe` with the replacement fill and the workload's check (`b54c8d5c…`).
Conditions: mains power; CPU load 20% and 46% of 12 logical processors before the two
series; no other `go` process; the same document sizes, cursor positions, keys, sizes
and focus. Fill: a paste in the composer against Ctrl+G, the same paste, Esc.

**Functional equivalence: PASS.** `TestC1FillsEquivalent`, a scratch test run through
`-overlay` on the unchanged application code, compares the two fills at 80x24, 120x40,
200x60 and 208x59: the same text byte for byte, the same cursor, line count and one undo
step, and the same rows drawn, the temporary folder's name on the project bar aside (its
number differs between any two rigs).

**The workload's check** (`ordinaryDraft`, 73.23) holds before and after the measured keys
of `TestKeystrokeToFrame`, `TestDraftKeystrokes`, `TestKeystrokeCost` and
`TestKeystrokeAllocs`. `TestFrameReuse` fills the same way, and `TestPasteToFrame` now
measures both a paste in the composer and one with Draft open. The check and the fill sit
outside the timed regions; they lengthen those tests' total time (`TestKeystrokeToFrame`
3.6 s to 6.0 s, `TestKeystrokeAllocs` 3.3 s to 8.7 s), not what they measure.

**Performance**, first series, three alternating pairs, p95 and p50 in µs, median of each
side, each side's range beside it; the clock's step is about 0.5 ms (above):

| Measure | Original | Replacement | Change |
|---|---:|---:|---:|
| 10,000 lines, composer, p95 | 1,016 (1,008 to 1,055) | 1,013 (1,006 to 1,050) | -3 (-0.3%) |
| 10,000 lines, composer, p50 | 0 | 0 | 0 |
| 10,000 lines, Draft, p95 | 1,093 (1,056 to 1,212) | 1,057 (1,026 to 1,111) | -36 (-3.3%) |
| 10,000 lines, Draft, p50 | 0 | 0 | 0 |
| 50,000 lines, p95 | 2,601 (2,584 to 2,663) | 2,558 (2,536 to 2,915) | -43 (-1.7%) |
| 50,000 lines, p50 | 744 (642 to 1,008) | 1,007 (779 to 1,019) | +263 (+35%) |
| `TestKeystrokeCost`, 10,000 lines, a key | 220.8 (206.4 to 222.9) | 224.8 (224.6 to 225.4) | +4.0 (+1.8%) |
| `TestKeystrokeCost`, 50,000 lines, a key | 511.4 (505.8 to 575.2) | 527.4 (504.4 to 843.8) | +16.0 (+3.1%) |

Allocations, in all six runs: `TestKeystrokeCost` 598 a key on either side, its bytes
within 0.04% (449,027 and 449,075; 1,454,046 and 1,454,336); `TestKeystrokeAllocs` 599,
565, 1,002 and 872 on either side, identical to the baseline.

Those p50 and cost results being noisy, six more alternating pairs of `TestKeystrokeCost`
alone (P8), CPU load 46% before them:

| Measure | Original | Replacement | Change |
|---|---:|---:|---:|
| 10,000 lines, a key | 256.5 (252.9 to 285.0) | 254.6 (245.9 to 326.9) | -1.9 (-0.8%) |
| 50,000 lines, a key | 500.3 (489.9 to 552.6) | 516.2 (508.1 to 610.7) | +15.9 (+3.2%) |

**Reading, by P8.** Nothing reproducible at 10,000 lines. At 50,000 lines the medians differ
by about +16 µs in both series, below each side's own range (63 and 103 µs in the second
series), with two pairs of nine going the other way: consistent in sign but not above the
observed variability, so **inconclusive** at that size, and not evidence of a regression. The
50,000-line p50 sits on one or two steps of the clock, whose readings are 0.5 ms apart, so
its +263 µs says nothing either; its p95 goes the other way. The two fills run the same
`Editor.Paste` and the same keys, with the same allocations: if the 16 µs is real, its likely
cause is the heap the fill's Draft frames and the check's frames leave behind for the garbage
collector, outside the editor's work, and it would be in both of C2's sides, which both use the
replacement fill. The budget, a p95 of 16 ms, is not near. C1 does not invalidate the fill
(73.24: functional equivalence decides). Logs: `evidence/c1`, `evidence/c1cost`.

## The full view: a miss, its cause, its remedy (P4, P6)

**First measure, on the implementation as first written** (`TestLongMessageView`, a
50,000-line message, 3.35 MB, its keys PgDn, ↓, ↑, PgUp, End and Home timed to their
frames):

| Size | p50 | p95 | Max | Target |
|---|---:|---:|---:|---:|
| 120x40 | 41.38 ms | 52.71 ms | 128.12 ms | p95 ≤ 16 ms |
| 208x59 | 40.17 ms | 44.28 ms | 50.33 ms | p95 ≤ 16 ms |

Over budget. **Cause:** `entryViewer` split the message and wrapped all its lines again on
every frame, 50,000 `wrapped` calls, though `fullScreen` then draws one screen of them. Its
cost follows the message's size.

**Remedy (P4, as the plan foresaw):** `Memo.messageView` keeps the rows and the line count of
the one message in view, by its text and the width: a string the message never changes,
compared at no cost while it is the same string. The memo is a memo: a nil `Memo` draws the same
frames (`TestMessageViewMemo`), and it is dropped at the second frame after the view closes, as
the file viewer's is, so it holds one message's rows while that message is open and nothing
otherwise. No second copy of the text: the rows' spans are its own wrapped text, for the screen,
and nothing reads them but the frame.

**Measured again:** the table under "The final timing step" below. The one wrap now falls on the
frame that opens the view, or on a resize while it is open, about 40 to 55 ms once.

## The final timing step (P7, 73.25)

Run on 2026-11-01 on the final code, alone, under `doc/evaluation.md`'s conditions: mains power, CPU load 29% of 12
logical processors before it, no other `go` process; open and idle: the browser, Teams, Outlook, Notepad,
Calculator, Settings, Windows Terminal and Claude. All 25 timed tests pass, in 106 s, and the guards. In
milliseconds; a p50 of 0 means under the clock's step of about 0.5 ms. The search's bar over the full view has no
target of its own (R-A, above).

| What is measured | Baseline p50 | p95 | max | Final p50 | p95 | max | Target |
|---|---:|---:|---:|---:|---:|---:|---:|
| a key to its frame, 10,000 lines blamed | 0.503 | 1.142 |  | 0 | 1.509 |  | p95 ≤ 16ms |
| a key to its frame, 3,000 refs | 0.988 | 1.950 |  | 0.995 | 1.729 |  | p95 ≤ 16ms |
| message field at 120x40, 18 lines typed | 0 | 0.671 | 7.823 | 0 | 1.000 | 1.544 | p95 ≤ 16ms |
| message field at 208x59, 18 lines typed | 0.555 | 1.568 | 6.748 | 0.549 | 1.504 | 15.24 | p95 ≤ 16ms |
| diff view, wide false (side by side false), at 120x40, 400 keys after 50 warm-up | 0 | 1.019 | 2.005 | 0 | 1.003 | 2.010 | p95 ≤ 16ms |
| diff view, wide true (side by side false), at 120x40, 400 keys after 50 warm-up | 0 | 0.655 | 2.314 | 0 | 0.675 | 1.504 | p95 ≤ 16ms |
| diff view, wide false (side by side false), at 208x59, 400 keys after 50 warm-up | 0 | 1.057 | 2.558 | 0 | 1.026 | 2.509 | p95 ≤ 16ms |
| diff view, wide true (side by side true), at 208x59, 400 keys after 50 warm-up | 0 | 1.068 | 1.904 | 0 | 1.108 | 7.267 | p95 ≤ 16ms |
| a key to its frame, 50,000 commits | 0 | 1.024 |  | 0 | 1.005 |  | p95 ≤ 16ms |
| banner frame to frame, 208x59, 1000 frames after 50 warm-up | 0.676 | 1.605 | 2.907 | 0.693 | 1.558 | 2.958 | p95 ≤ 2ms |
| key to frame in the project field, banner running, 208x59, 400 keys | 0.076 | 0.179 | 0.841 | 0.074 | 0.135 | 0.733 | p95 ≤ 16ms |
| the picture on entering, 208x59, the size's 120ms wait included | 181.53 |  | 196.36 | 180.56 |  | 195.73 | ≤ 260ms, the terminal's 40 ms apart |
| keystroke-to-frame, 10,000-line draft, 800 keys after 50 warm-up | 0 | 1.045 | 4.031 | 0 | 1.091 | 2.265 | p95 ≤ 16ms |
| keystroke-to-frame in the draft window, 10,000-line draft, 800 keys | 0 | 1.116 | 2.156 | 0 | 1.144 | 2.994 | p95 ≤ 16ms |
| model selector, Alt+P at 120x40, 400 keys after 50 warm-up | 0 | 1.001 | 1.605 | 0 | 1.002 | 1.517 | p95 ≤ 16ms |
| model selector, ↑↓ in the expanded list at 120x40, 400 keys after 50 warm-up | 0 | 1.058 | 1.622 | 0 | 1.070 | 1.914 | p95 ≤ 16ms |
| model selector, Alt+P at 208x59, 400 keys after 50 warm-up | 0 | 1.112 | 6.514 | 0 | 1.119 | 13.47 | p95 ≤ 16ms |
| model selector, ↑↓ in the expanded list at 208x59, 400 keys after 50 warm-up | 0 | 1.583 | 14.55 | 0 | 1.507 | 8.954 | p95 ≤ 16ms |
| sidebar hover, the panel shown false, icons false, at 120x40, 400 reports after 50 warm-up | 0.526 | 1.114 | 2.092 | 0 | 1.026 | 6.130 | p95 ≤ 16ms |
| sidebar hover, the panel shown true, icons false, at 120x40, 400 reports after 50 warm-up | 0 | 1.016 | 12.15 | 0 | 1.021 | 2.554 | p95 ≤ 16ms |
| sidebar hover, the panel shown false, icons false, at 208x59, 400 reports after 50 warm-up | 0.504 | 1.520 | 10.56 | 0 | 1.522 | 8.145 | p95 ≤ 16ms |
| sidebar hover, the panel shown true, icons false, at 208x59, 400 reports after 50 warm-up | 0.651 | 1.552 | 10.18 | 0 | 1.523 | 4.978 | p95 ≤ 16ms |
| sidebar hover, the panel shown false, icons true, at 120x40, 400 reports after 50 warm-up | 0 | 1.567 | 12.61 | 0 | 1.101 | 1.833 | p95 ≤ 16ms |
| sidebar hover, the panel shown true, icons true, at 120x40, 400 reports after 50 warm-up | 0 | 1.000 | 1.837 | 0 | 1.010 | 1.551 | p95 ≤ 16ms |
| sidebar hover, the panel shown false, icons true, at 208x59, 400 reports after 50 warm-up | 0.511 | 1.509 | 2.635 | 0 | 1.509 | 2.523 | p95 ≤ 16ms |
| sidebar hover, the panel shown true, icons true, at 208x59, 400 reports after 50 warm-up | 0.545 | 1.645 | 3.354 | 0 | 1.361 | 2.222 | p95 ≤ 16ms |
| keystroke-to-frame, 50,000-line draft | 1.015 | 3.011 | 5.624 | 0.682 | 2.159 | 3.709 | p95 ≤ 16ms |
| delta-to-frame at the end, 200x60, 400 deltas | 1.506 | 3.022 | 4.776 | 1.002 | 2.515 | 3.618 | p95 ≤ 16ms |
| delta-to-frame, a command's output in 8192 deltas of 8192 bytes (64 MiB) | 0 | 1.024 | 10.26 | 0 | 1.015 | 13.52 | p95 ≤ 16ms |
| delta-to-frame, 2000-line answer, one line per delta | 0.600 | 1.991 | 14.13 | 0.543 | 1.632 | 6.535 | p95 ≤ 16ms |
| a key to its frame, a turn of 1,000 files unfolded | 0.506 | 1.508 |  | 0.531 | 1.134 |  | p95 ≤ 16ms |
| search, two hours (5282 entries), a key in the bar to its frame at 200x60, 407 keys after 50 warm-up | 3.030 | 4.998 | 5.656 | 3.015 | 4.640 | 9.892 | p95 ≤ 16ms |
| search, two hours, the bar open, a delta to its frame at 200x60, 400 deltas | 2.097 | 3.996 | 5.511 | 2.061 | 3.715 | 5.070 | p95 ≤ 16ms |
| search, answers ten times longer (5282 entries), a key in the bar to its frame at 200x60, 407 keys after 50 warm-up | 7.082 | 9.525 | 12.04 | 7.054 | 9.269 | 10.83 | p95 ≤ 16ms |
| conversation 5522 entries, 721 of them thinking; card frames at 200x60, 80 over 4 s | 1.023 | 3.060 | 11.42 | 1.002 | 2.094 | 9.041 | p95 ≤ 16ms |
| event-to-frame, 2400 calls (30 Write calls of 98 KB) in one open turn, 8 agents running, 200x60, the last 481 events | 1.006 | 2.548 | 4.055 | 1.013 | 2.541 | 7.392 | p95 ≤ 16ms |
| a 50,000-line chip in view (3349999 bytes), 400 deltas to their frames at 200x60 (new) | | | | 0 | 1.113 | 2.074 | p95 ≤ 16ms |
| around a 50,000-line chip, the motions, 480 keys at 200x60 (new) | | | | 0 | 1.066 | 1.736 | p95 ≤ 16ms |
| around a 50,000-line chip, typing beside it, 400 keys at 200x60 (new) | | | | 0 | 1.263 | 2.593 | p95 ≤ 16ms |
| around a 50,000-line chip, its whole deletion and its undo, 200 keys at 200x60 (new) | | | | 0 | 1.511 | 2.071 | p95 ≤ 16ms |
| the full view of 50,000 lines at 120x40, 200 keys (new) | | | | 0 | 0 | 1.036 | p95 ≤ 16ms |
| y's key in Update at 120x40, 100 presses (new) | | | | 0 | 0 | 0.502 | ≤ 16ms |
| that view with the search's bar open at 120x40, 40 keys (new) | | | | 31.63 | 33.95 | 34.33 |  |
| the full view of 50,000 lines at 208x59, 200 keys (new) | | | | 0 | 1.000 | 1.507 | p95 ≤ 16ms |
| y's key in Update at 208x59, 100 presses (new) | | | | 0 | 0 | 0.748 | ≤ 16ms |
| that view with the search's bar open at 208x59, 40 keys (new) | | | | 31.93 | 34.22 | 34.38 |  |
| paste-to-frame in the composer, 50,000 lines (3350000 bytes), 10 runs (new) | | | | 0 |  | 1.182 | ≤ 100ms |
| paste-to-frame with the draft window open, 50,000 lines (3350000 bytes), 10 runs (new) | | | | 1.385 |  | 4.060 | ≤ 100ms |

`TestPasteToFrame` on the baseline, one path, the whole text drawn in the composer: p50 1.145 ms, max 1.942 ms.

`TestKeystrokeAllocs`, its bound 1,200: 599, 565, 1,002 and 872, identical to the baseline. `TestFrameReuse` passes.
`TestChipFrameAllocs`: a frame with a chip of four lines 507 allocations, of 404 lines 509, of 50,000 lines 519, of one
3 MB line 508; with a three-row message 523. `y`'s sequence for 3.35 MB is 4,466,676 bytes, and Bubble Tea's copy of it
on the event loop took 2 and 8.7 ms.

## C2: the replacement fill before against after the feature (P7, P8)

`repl.test.exe`, built before any application code changed (SHA-256 `b54c8d5c…`), against `final.test.exe`, the same
tests' fill on the final code (`3532d53a…`): the same document sizes, cursor positions, keys, sizes and focus, in
alternating pairs. Mains power; CPU load 18% and 13% before the two series; no other `go` process. Times in µs, median of
each side, its range beside it.

| Measure | Before | After | Change |
|---|---:|---:|---:|
| 10,000 lines, composer, p95 | 1,010 (1,003 to 1,069) | 1,006 (1,005 to 1,029) | -4 (-0.4%) |
| 10,000 lines, composer, p50 | 0 | 0 | 0 |
| 10,000 lines, Draft, p95 | 1,109 (1,106 to 1,200) | 1,111 (1,062 to 1,116) | +2 (+0.2%) |
| 10,000 lines, Draft, p50 | 0 | 0 | 0 |
| 50,000 lines, p95 | 2,371 (2,017 to 2,732) | 2,186 (2,033 to 2,783) | -185 (-7.8%) |
| 50,000 lines, p50 | 1,011 (560 to 1,015) | 867 (606 to 1,017) | -143 (-14%) |
| `TestKeystrokeCost`, 10,000 lines, a key | 211.6 (203.5 to 217.1) | 215.9 (207.3 to 223.7) | +4.3 (+2.0%) |
| `TestKeystrokeCost`, 50,000 lines, a key | 482.6 (480.8 to 542.7) | 492.5 (483.7 to 510.3) | +10.0 (+2.1%) |

Allocations, in all ten runs: `TestKeystrokeCost` 598 a key on either side, its bytes within 0.04%; `TestKeystrokeAllocs`
599, 565, 1,002 and 872 on either side.

The cost being the sharpest signal, and reading higher on the second binary, three more series of `TestKeystrokeCost`
alone, six alternating pairs each, on mains power, CPU load 13%, 21% and 17% before them:

| Series | 10,000 lines, a key | 50,000 lines, a key |
|---|---:|---:|
| Before, then after: before / after | 246.1 (240.1 to 248.8) / 249.6 (238.5 to 261.4): +3.4 (+1.4%) | 497.9 (486.9 to 519.3) / 513.3 (494.2 to 539.6): +15.3 (+3.1%) |
| After, then before: after / before | 248.6 (247.1 to 293.5) / 249.4 (248.1 to 254.1): -0.8 (-0.3%) | 506.3 (502.4 to 538.2) / 507.9 (499.3 to 550.8): -1.6 (-0.3%) |
| Before, then a copy of the same binary | 248.3 (245.0 to 254.9) / 247.8 (245.2 to 262.7): -0.5 (-0.2%) | 506.4 (498.3 to 691.2) / 504.5 (490.5 to 513.4): -1.9 (-0.4%) |

**Reading, by P8.** Nothing above each side's own range, and the sign does not survive reversing the pairs: no
reproducible regression, and inconclusive for a difference under about 1%, which this noise cannot resolve. Likely
source of any real part: the few comparisons added per frame with no range (`drawRow`'s per cell, the empty chip loops,
`bound()`); P3's no-range path visits no text. It would not grow with the document or the session, and the budget left
is 15.5 ms of 16 at 50,000 lines. Named under Reservations (R-C) for the owner's reading.

## Hidden text: how the cost follows the hidden size (73.26)

`TestChipScale`, Windows only, on the performance counter: the median of 200 samples, each of 20 calls, of the view and
each motion from its edge, the same text around a chip of 5 KB, 500 KB and 5 MB, a block of many lines and of one line.
The final build:

| Measure | 5 KB | 500 KB | 5 MB | 5 MB / 5 KB |
|---|---:|---:|---:|---:|
| many lines LongMessage | 2.785µs | 950ns | 955ns | 0.34 |
| many lines view | 525ns | 430ns | 350ns | 0.67 |
| many lines Left | 135ns | 100ns | 80ns | 0.59 |
| many lines Shift+Left | 90ns | 80ns | 75ns | 0.83 |
| many lines Right | 85ns | 75ns | 80ns | 0.94 |
| many lines Shift+Right | 115ns | 75ns | 75ns | 0.65 |
| many lines Up | 150ns | 100ns | 100ns | 0.67 |
| many lines Shift+Up | 100ns | 120ns | 100ns | 1.00 |
| many lines Down | 110ns | 130ns | 85ns | 0.77 |
| many lines Shift+Down | 90ns | 125ns | 90ns | 1.00 |
| many lines Home | 75ns | 105ns | 75ns | 1.00 |
| many lines Shift+Home | 95ns | 105ns | 80ns | 0.84 |
| many lines End | 80ns | 90ns | 80ns | 1.00 |
| many lines Shift+End | 100ns | 85ns | 80ns | 0.80 |
| many lines WordLeft | 140ns | 105ns | 105ns | 0.75 |
| many lines Shift+WordLeft | 125ns | 105ns | 110ns | 0.88 |
| many lines WordRight | 100ns | 100ns | 100ns | 1.00 |
| many lines Shift+WordRight | 100ns | 120ns | 100ns | 1.00 |
| one line LongMessage | 1.51µs | 1.51µs | 1.51µs | 1.00 |
| one line view | 985ns | 330ns | 340ns | 0.35 |
| one line Left | 75ns | 85ns | 75ns | 1.00 |
| one line Shift+Left | 85ns | 95ns | 80ns | 0.94 |
| one line Right | 100ns | 75ns | 75ns | 0.75 |
| one line Shift+Right | 75ns | 75ns | 70ns | 0.93 |
| one line Up | 95ns | 100ns | 90ns | 0.95 |
| one line Shift+Up | 95ns | 110ns | 90ns | 0.95 |
| one line Down | 105ns | 90ns | 80ns | 0.76 |
| one line Shift+Down | 90ns | 90ns | 85ns | 0.94 |
| one line Home | 75ns | 75ns | 75ns | 1.00 |
| one line Shift+Home | 95ns | 85ns | 80ns | 0.84 |
| one line End | 75ns | 95ns | 90ns | 1.20 |
| one line Shift+End | 75ns | 80ns | 75ns | 1.00 |
| one line WordLeft | 115ns | 110ns | 105ns | 0.91 |
| one line Shift+WordLeft | 105ns | 110ns | 105ns | 1.00 |
| one line WordRight | 95ns | 100ns | 100ns | 1.05 |
| one line Shift+WordRight | 120ns | 135ns | 100ns | 0.83 |

No row grows with the hidden size; the ratio of 5 is not approached. Those rows would show a traversal, as these mutants
prove, each adding a scan of the whole text with no value changed (M69 is C07):

| Mutant | What it adds, its values unchanged | Its worst row | 5 KB | 500 KB | 5 MB | 5 MB / 5 KB |
|---|---|---|---:|---:|---:|---:|
| C01 | Lines on the whole text in the view | many lines view | 585ns | 6.29µs | 84.57µs | 144.56 |
| C02 | CursorLine on the whole text in the view | many lines view | 480ns | 6.17µs | 90.94µs | 189.46 |
| C03 | LineRange on the whole text in the view | many lines view | 1.43µs | 76.93µs | 845.27µs | 591.10 |
| C04 | linePos on the whole text in the view | one line view | 1.95µs | 128.71µs | 1.36382ms | 699.39 |
| C05 | a motion reading the whole text | one line Left | 130ns | 5.73µs | 90.255µs | 694.27 |
| C07 | LongMessage counting the whole text (M69) | one line LongMessage | 1.535µs | 7.4µs | 90.8µs | 59.15 |
| C06 | LongMessage looking for a line's end | one line LongMessage | 1.585µs | 8.915µs | 93.615µs | 59.06 |

**`TestChipCount`**, the same measures around 1, 10 and 100 chips of 50 KB each: with 1 chip, the view 405 ns, a motion over it 75 ns, a key to its frame p50 150 µs; with 10 chips, 500 KB, 1.41 µs, 175 ns, 154 µs; with 100 chips, 5 MB, 26.1 µs, 1.45 µs, 433 µs (max 753 µs). The view grows with the number of chips, each chip's place being found in the compact text, and stays far inside a frame; no hidden byte is visited. Added after the timing step, judging nothing, and run alone once.

## Golden frames (V5, 73.21)

343 frames; 18 new, 17 changed, none of them `draft-*`. The review by script (`goldens-review.txt`, its script in the
package): each changed frame has the same rows as HEAD's and its chip's row; 113 rows changed, none outside zone B's
columns where zone B stands beside other zones. What changes is a prompt of more than 3 rows becoming its chip, in
the recordings that have one, and zone B's rows moving up, the conversation being a few rows shorter.

**The three whitespace warnings (added after the third review, which met them).** `git apply --whitespace=warn` on
the new-files patch says "3 lines add whitespace errors", all "new blank line at EOF", on `chip-79x24.txt`,
`chip-composer-79x24.txt` and `chip-cursor-79x24.txt`. They are expected, and the frames stay as they are. At 79x24
the screen is the too-small message and blank rows, and `TestGoldenFrames` requires each frame to have exactly the
terminal's height (`len(lines) != hgt`, `golden_test.go`): 24 rows, 21 of them blank. Those three frames are byte for
byte the same as each other and as 60 of the 75 tracked 79x24 frames, `dashboard-79x24.txt` and `draft-79x24.txt`
among them. Every tracked 79x24 frame gives the same report when compared with the empty tree (75 of 75), no other
tracked frame does, and neither do the 15 new frames at the other sizes. The `git diff --check` recorded in
`suite-final/static.log` says exit 0 for the tracked files only: it does not see the 18 untracked frames.
`git diff --no-index --check /dev/null <frame>` reports the three today, so `git diff --cached --check` will once they
are staged; that is expected, for the commit's record to say, and no reason to trim them.

## Mutants (73.27)

Through `go test -overlay`, one at a time, the repository never written. A build failure is no kill; a mutant the named
tests leave alive runs its whole package. All 68 by value killed, and 7 by cost seen above. Rule by rule:

| Mutant | Rule | Result | By |
|---|---|---|---|
| M01 | D1 the threshold | KILLED | `TestLongMessageDefinition`, `TestLongMessageBound`, `TestZoneBForms` |
| M02 | D1 the width | KILLED | `TestZoneBForms`, `TestMessageWidthIsDrawn` |
| M03 | D1 an empty line a row | KILLED | `TestLongMessageDefinition`, `TestZoneBForms`, `FuzzLongMessage` |
| M04 | D2 the bound holds a dropped space | KILLED | `TestLongMessageDefinition`, `TestLongMessageBound`, `FuzzLongMessage` |
| M05 | D2 stops at the fourth line | KILLED | `TestLongMessageBound` |
| M06 | D2 stops at the bound | KILLED | `TestLongMessageBound` |
| M07 | D2 no allocation | KILLED | `TestLongMessageBound` |
| M08 | D3 N is logical lines | KILLED | `TestChipWords`, `TestZoneBForms`, `TestChipWithImages` |
| M09 | D3 S is UTF-8 bytes | KILLED | `TestChipWords` |
| M10 | D3 "1 line" | KILLED | `TestChipWords`, `TestZoneBForms` |
| M11 | D3 the muted colour | KILLED | `TestZoneBForms` |
| M12 | D4 a long message is its chip | KILLED | `TestZoneBForms`, `TestMessageWidthIsDrawn`, `TestChipWithImages` |
| M13 | D4 its images under it | KILLED (first run: NOT APPLIED) | `TestChipWithImages` |
| M14 | D4 the cursor on its row | KILLED | `TestChipFullView` |
| M15 | D5 the row stands for its lines | KILLED (first run: SURVIVED) | `TestChipMatchRow` |
| M16 | D4 Expandable asks the rule | KILLED | `TestChipFullView` |
| M17 | D5 y copies its bytes | KILLED | `TestChipFullView` |
| M18 | D5 y in the full view | KILLED | `TestChipFullView` |
| M19 | D5 y's hint | KILLED | `TestCommandCopyKey`, `TestChipFullView` |
| M20 | D5 the search lights the chip | KILLED | `TestSearchLit` |
| M21 | D5 the full view's head | KILLED | `TestChipWords` |
| M22 | D6 the cursor leaves | KILLED | `TestChipResize` |
| M23 | D7 the final break stays out | KILLED | `TestMark`, `TestEditorRangesModel`, `FuzzEditorRanges` |
| M24 | D7 judged alone | KILLED (first run: BUILD FAILED) | `TestLongPaste`, `TestChipDraft` |
| M25 | D7 not with Draft on top | KILLED | `TestLongPaste`, `TestChipDraft` |
| M26 | D7 in the paste's undo step | KILLED | `TestMark` |
| M27 | D8 an edit before shifts | KILLED (first run: BUILD FAILED) | `TestRangesFollowEdits`, `TestChipDeletion`, `TestEditorRangesModel` |
| M28 | D8 touching drops | KILLED | `TestRangesFollowEdits`, `TestChipDeletion`, `TestChipMotions` |
| M29 | D8 undo puts them back | KILLED | `TestRangesFollowEdits`, `TestChipDeletion`, `TestDraftWindowRanges` |
| M30 | D8 redo puts them back | KILLED | `TestMark`, `TestRangesFollowEdits`, `TestDraftWindowRanges` |
| M31 | D8 SetText ends them | KILLED | `TestRangesFollowEdits`, `TestEditorRangesModel` |
| M32 | D8 a bad list is dropped | KILLED | `TestInvalidRanges` |
| M33 | D8 merged steps keep theirs | KILLED | `TestEditorRangesModel` |
| M34 | D8 the check takes rune boundaries | KILLED (first run: BUILD FAILED) | `TestInvalidRanges` |
| M35 | D9 Backspace takes the block | KILLED | `TestChipDeletion`, `TestEditorRangesModel`, `FuzzEditorRanges` |
| M36 | D9 Delete takes the block | KILLED | `TestChipDeletion`, `TestEditorRangesModel` |
| M37 | D9 Ctrl+Backspace takes the block | KILLED | `TestChipDeletion`, `TestEditorRangesModel`, `FuzzEditorRanges` |
| M38 | D9 Ctrl+Delete takes the block | KILLED | `TestChipDeletion`, `TestEditorRangesModel`, `FuzzEditorRanges` |
| M39 | D9 a word deletion stops, left | KILLED | `TestChipDeletion`, `TestEditorRangesModel` |
| M40 | D9 a word deletion stops, right | KILLED | `TestChipDeletion`, `TestEditorRangesModel` |
| M41 | D9 motions over a chip | KILLED | `TestChipMotions`, `TestEditorRangesModel`, `FuzzEditorRanges` |
| M42 | D9 the mapping back | KILLED | `TestChipDeletion`, `TestChipMotions`, `TestCompactMapping` |
| M43 | D9 Enter after a chip sends | KILLED | `TestChipEnter` |
| M44 | D10 a change in Draft ends them | KILLED | `TestDraftWindowRanges`, `TestEditorRangesModel`, `FuzzEditorRanges` |
| M45 | D10 ranges do not bind Draft | KILLED | `TestChipDeletion`, `TestChipMotions`, `TestCompactMapping` |
| M46 | D10 the mode is set | KILLED | `TestChipEnter`, `TestChipDraft`, `TestComposerHiddenText` |
| M47 | D10 Settle moves the cursor | KILLED (first run: BUILD FAILED) | `TestSettle`, `TestEditorRangesModel`, `FuzzEditorRanges` |
| M48 | D10 Settle ends the selection | KILLED | `TestSettle` |
| M49 | D10 undo settles | KILLED | `TestEditorRangesModel`, `FuzzEditorRanges` |
| M50 | D10 a composer key settles | KILLED | `TestChipDraft` |
| M51 | D10 Ctrl+G keeps the cursor | KILLED | `TestChipDraft` |
| M52 | D10 a paste settles | KILLED | `TestChipDraft` |
| M53 | D10 a copy settles | KILLED | `TestChipDraft` |
| M54 | D11 the view on the compact form | KILLED | `TestLongPaste`, `TestChipEdges`, `TestChipCursor` |
| M55 | D11 R1 Lines on the whole text | KILLED | `TestChipEdges`, `TestComposerHiddenText` |
| M56 | D11 R1 CursorLine on the whole text | KILLED | `TestChipEdges`, `TestComposerHiddenText` |
| M57 | D11 R1 LineRange on the whole text | KILLED | `TestLongPaste`, `TestChipEdges`, `TestTwoChips` |
| M58 | D11 R1 linePos on the whole text | KILLED | `TestTwoChips`, `TestComposerHiddenText` |
| M59 | D11 the chip's cell | KILLED | `TestEditorChipCell` |
| M60 | D11 the row's count | KILLED | `TestEditorLineCount` |
| M61 | D11 a chip's rows counted with it | KILLED | `TestGoldenFrames` |
| M62 | D11 the mouse maps back | KILLED | `TestChipCursor`, `TestChipClicks`, `TestChipDeleteKeys` |
| M63 | D11 the mouse's word | KILLED | `TestChipClicks` |
| M64 | D11 the compact form holds no hidden byte | KILLED | `TestChipDeletion` |
| M65 | P4 the memo's width | KILLED (first run: SURVIVED) | `TestMessageViewMemo` |
| M66 | P4 the memo's text | KILLED | `TestMessageViewMemo` |
| M67 | P4 wrapped once | KILLED | `TestMessageViewMemo` |
| M68 | P4 dropped at the second frame | KILLED | `TestMessageViewMemo` |

M69, `strings.Count` over the whole text in `LongMessage`, changes no value and survived by value: it is C07 above.

## Fuzzing

Each alone, 60 s, `-fuzzminimizetime 2s -parallel 4`:

| Target | Runs | New interesting | Result |
|---|---:|---:|---|
| `FuzzLongMessage` (new) | 1,294,906 | 137 | no failure |
| `FuzzEditorRanges` (new), first run | stopped at 8 s | | the model's rule, above; no application code changed |
| `FuzzEditorRanges`, after it | 652,279 | 362 | no failure |
| `FuzzTextWidth` | 1,434,525 | 20 | no failure |
| `FuzzInspectData` | 223,830 | 110 | no failure |
| `FuzzGIFFrames` | 2,110,805 | 23 | no failure |
| `FuzzMapperOut`, Claude Code | 992,658 | 123 | no failure |
| `FuzzMapperOut`, Codex | 984,157 | 173 | no failure |

## Race

`go test -race ./internal/render ./internal/input ./internal/app -count=1`, WinLibs gcc 16.2.0, `CGO_ENABLED=1`: exit 0, no data race: 276.9 s, 248.6 s and 589.6 s. The one test changed after that run, `TestEntryViewer`, five times under `-race`: exit 0.

## The Windows Terminal scenes (A9, 73.29)

**Actual Windows Terminal observations**, by `scripts/harness/wt-check.ps1`, on `testdata/mock/claude-thinking.jsonl`, with no
CLI, no quota and no clipboard: no double click, drag, `y` or real paste. The build's SHA-256:
`8E4E97BB9A5A4D369EF68DC18AB4A6FB32AE9647D3C0E5EEE58DBE13360016A0`; rebuilt from the final tree it gives the same hash. Each run's folder is
`%TEMP%\acc-wt\<outName>-claude2-<mode>`, the harness refusing a used one (the first attempts are `-claude-`). A composer chip
needs a real paste, so the `tea.PasteMsg` tests and the owner's second step stand in for it.

What they show, in words; I viewed the screenshots:
- **W01**: after the replay, PgUp brings the second prompt's row, `▣ Long message · 1 line · 282 B`, under `YOU`, with the agent's
  work below it, `✓ Thought for 14s · 2,852 thinking tokens` first. The prompt has 282 characters and draws 5 rows here.
- **W02**, maximised, 208x59: the same prompt drawn whole on 3 rows, ending "happened to each attempt."; none of the run's screens
  holds `Long message`, read from their saved text.
- **W03**: six ↑ reach the chip: `▎▸▣ Long message · 1 line       Enter full view · ↑↓ entries · Esc back`, its size dropped for the
  hint, the footer "Enter full view · ↑↓ entries · Esc back to the input · F1 keys".
- **W04**: Enter shows `◀ back (Esc)  YOU  17:36 … 1 line`, the whole text, "y copy the message · ↑↓ PgUp PgDn scroll · Home top · End
  bottom · Esc back"; PgDn keeps it; Esc gives the chip again.
- **W05**: six lines typed with Ctrl+J, each on its own row, `6 lines` in the row under them; none of its 19 screens holds `Long message`.
- **W06**: Enter sends them: `▣ Long message · 6 lines · 38 B`, and the mock's own note that nothing reached a CLI.
- **W07**, maximised and at 120x40: Ctrl+G, Ctrl+F, four typed lines in the full-screen `Draft`, `4 lines`, `line 4/4`; Esc: the same lines in
  the composer; Ctrl+G, Enter: `▣ Long message · 4 lines · 25 B`.
- **W08**, 100x30, zone B's narrowest form: `▎▸▣ Long m… Enter full view · ↑↓ entries · Esc back`, the hint whole, on one row.

Every screenshot, with what the run's log says of it:

**W01 zone-b-chip, 120x40** (`-outName p73-w01-zone-b-chip`, folder `p73-w01-zone-b-chip-claude2-90x30`; the log's last line: "assertions: 3 passed, 0 failed"; exit 0).

| Screenshot | What it records |
|---|---|
| `w01-01-end-of-session.png` | asserted at 28191 ms: 'COMPLETED' found |
| `w01-02-page-up.png` | keys at 30368 ms: 21 |
| `w01-03-chip.png` | asserted at 31591 ms: 'Long message' found |
| `w01-04-agent-work.png` | asserted at 32823 ms: '2,852 thinking tokens' found |
| `w01-05-settled.png` | screen at 33189 ms |

**W02 zone-b-normal, maximised, 208x59** (`-outName p73-w02-zone-b-normal`, folder `p73-w02-zone-b-normal-claude2-max`; the log's last line: "assertions: 2 passed, 0 failed"; exit 0).

| Screenshot | What it records |
|---|---|
| `w02-01-end-of-session.png` | asserted at 28785 ms: 'COMPLETED' found |
| `w02-02-prompt-last-words.png` | asserted at 30611 ms: 'each attempt.' found |
| `w02-03-settled.png` | screen at 31416 ms |

**W03 chip-cursor, 120x40** (`-outName p73-w03-chip-cursor`, folder `p73-w03-chip-cursor-claude2-90x30`; the log's last line: "assertions: 3 passed, 0 failed"; exit 0).

| Screenshot | What it records |
|---|---|
| `w03-01-end-of-session.png` | asserted at 28127 ms: 'COMPLETED' found |
| `w03-02-up-to-chip.png` | keys at 31508 ms: 26.26.26.26.26.26 |
| `w03-03-chip.png` | asserted at 32724 ms: 'Long message' found |
| `w03-04-hint.png` | asserted at 33954 ms: 'Enter full view' found |

**W04 full-view, 120x40** (`-outName p73-w04-full-view`, folder `p73-w04-full-view-claude2-90x30`; the log's last line: "assertions: 4 passed, 0 failed"; exit 0).

| Screenshot | What it records |
|---|---|
| `w04-01-end-of-session.png` | asserted at 28173 ms: 'COMPLETED' found |
| `w04-02-up-to-chip.png` | keys at 31600 ms: 26.26.26.26.26.26 |
| `w04-03-enter.png` | keys at 32748 ms: 0D |
| `w04-04-full-view-last-words.png` | asserted at 33849 ms: 'each attempt.' found |
| `w04-05-page-down.png` | keys at 35146 ms: 22 |
| `w04-06-still-there.png` | asserted at 36478 ms: 'each attempt.' found |
| `w04-07-esc.png` | keys at 37896 ms: 1B |
| `w04-08-chip-again.png` | asserted at 39211 ms: 'Long message' found |

**W05 typed-composer, 120x40** (`-outName p73-w05-typed-composer`, folder `p73-w05-typed-composer-claude2-90x30`; the log's last line: "assertions: 8 passed, 0 failed"; exit 0).

| Screenshot | What it records |
|---|---|
| `w05-01-end-of-session.png` | asserted at 28185 ms: 'COMPLETED' found |
| `w05-02-type-alpha.png` | keys at 31319 ms: 41.4C.50.48.41 |
| `w05-03-ctrl-j-1.png` | chord at 32461 ms: 11.4A |
| `w05-04-type-bravo.png` | keys at 34789 ms: 42.52.41.56.4F |
| `w05-05-ctrl-j-2.png` | chord at 35889 ms: 11.4A |
| `w05-06-type-charlie.png` | keys at 38871 ms: 43.48.41.52.4C.49.45 |
| `w05-07-ctrl-j-3.png` | chord at 39971 ms: 11.4A |
| `w05-08-type-delta.png` | keys at 42481 ms: 44.45.4C.54.41 |
| `w05-09-ctrl-j-4.png` | chord at 43602 ms: 11.4A |
| `w05-10-type-echo.png` | keys at 45770 ms: 45.43.48.4F |
| `w05-11-ctrl-j-5.png` | chord at 46868 ms: 11.4A |
| `w05-12-type-foxtrot.png` | keys at 49921 ms: 46.4F.58.54.52.4F.54 |
| `w05-13-line-alpha.png` | asserted at 51156 ms: 'alpha' found |
| `w05-14-line-bravo.png` | asserted at 52394 ms: 'bravo' found |
| `w05-15-line-charlie.png` | asserted at 53572 ms: 'charlie' found |
| `w05-16-line-delta.png` | asserted at 54805 ms: 'delta' found |
| `w05-17-line-echo.png` | asserted at 55973 ms: 'echo' found |
| `w05-18-line-foxtrot.png` | asserted at 57174 ms: 'foxtrot' found |
| `w05-19-line-count.png` | asserted at 58353 ms: '6 lines' found |

**W06 typed-send, 120x40** (`-outName p73-w06-typed-send`, folder `p73-w06-typed-send-claude2-90x30`; the log's last line: "assertions: 4 passed, 0 failed"; exit 0).

| Screenshot | What it records |
|---|---|
| `w06-01-end-of-session.png` | asserted at 28308 ms: 'COMPLETED' found |
| `w06-02-type-alpha.png` | keys at 31208 ms: 41.4C.50.48.41 |
| `w06-03-ctrl-j-1.png` | chord at 32334 ms: 11.4A |
| `w06-04-type-bravo.png` | keys at 34817 ms: 42.52.41.56.4F |
| `w06-05-ctrl-j-2.png` | chord at 35932 ms: 11.4A |
| `w06-06-type-charlie.png` | keys at 38914 ms: 43.48.41.52.4C.49.45 |
| `w06-07-ctrl-j-3.png` | chord at 40031 ms: 11.4A |
| `w06-08-type-delta.png` | keys at 42398 ms: 44.45.4C.54.41 |
| `w06-09-ctrl-j-4.png` | chord at 43537 ms: 11.4A |
| `w06-10-type-echo.png` | keys at 45838 ms: 45.43.48.4F |
| `w06-11-ctrl-j-5.png` | chord at 46944 ms: 11.4A |
| `w06-12-type-foxtrot.png` | keys at 49995 ms: 46.4F.58.54.52.4F.54 |
| `w06-13-line-count.png` | asserted at 51194 ms: '6 lines' found |
| `w06-14-enter.png` | keys at 52543 ms: 0D |
| `w06-15-chip.png` | asserted at 53779 ms: 'Long message' found |
| `w06-16-chip-lines.png` | asserted at 55028 ms: '6 lines' found |

**W07 draft, maximised, 208x59** (`-outName p73-w07-draft`, folder `p73-w07-draft-claude2-max`; the log's last line: "assertions: 6 passed, 0 failed"; exit 0).

| Screenshot | What it records |
|---|---|
| `w07-01-end-of-session.png` | asserted at 28733 ms: 'COMPLETED' found |
| `w07-02-ctrl-g.png` | chord at 30593 ms: 11.47 |
| `w07-03-draft.png` | asserted at 32161 ms: 'Draft' found |
| `w07-04-ctrl-f.png` | chord at 33353 ms: 11.46 |
| `w07-05-type-alpha.png` | keys at 35558 ms: 41.4C.50.48.41 |
| `w07-06-ctrl-j-1.png` | chord at 36694 ms: 11.4A |
| `w07-07-type-bravo.png` | keys at 38891 ms: 42.52.41.56.4F |
| `w07-08-ctrl-j-2.png` | chord at 40028 ms: 11.4A |
| `w07-09-type-charlie.png` | keys at 42889 ms: 43.48.41.52.4C.49.45 |
| `w07-10-ctrl-j-3.png` | chord at 44067 ms: 11.4A |
| `w07-11-type-delta.png` | keys at 46512 ms: 44.45.4C.54.41 |
| `w07-12-typed-in-draft.png` | asserted at 47727 ms: 'delta' found |
| `w07-13-esc.png` | keys at 49474 ms: 1B |
| `w07-14-back-in-composer.png` | asserted at 51196 ms: 'delta' found |
| `w07-15-ctrl-g-again.png` | chord at 52687 ms: 11.47 |
| `w07-16-enter.png` | keys at 54437 ms: 0D |
| `w07-17-chip.png` | asserted at 56059 ms: 'Long message' found |
| `w07-18-chip-lines.png` | asserted at 57722 ms: '4 lines' found |

**W07 draft, 120x40** (`-outName p73-w07-draft`, folder `p73-w07-draft-claude2-90x30`; the log's last line: "assertions: 6 passed, 0 failed"; exit 0).

| Screenshot | What it records |
|---|---|
| `w07-01-end-of-session.png` | asserted at 28298 ms: 'COMPLETED' found |
| `w07-02-ctrl-g.png` | chord at 30163 ms: 11.47 |
| `w07-03-draft.png` | asserted at 31578 ms: 'Draft' found |
| `w07-04-ctrl-f.png` | chord at 32572 ms: 11.46 |
| `w07-05-type-alpha.png` | keys at 35073 ms: 41.4C.50.48.41 |
| `w07-06-ctrl-j-1.png` | chord at 36211 ms: 11.4A |
| `w07-07-type-bravo.png` | keys at 38706 ms: 42.52.41.56.4F |
| `w07-08-ctrl-j-2.png` | chord at 39837 ms: 11.4A |
| `w07-09-type-charlie.png` | keys at 42773 ms: 43.48.41.52.4C.49.45 |
| `w07-10-ctrl-j-3.png` | chord at 43802 ms: 11.4A |
| `w07-11-type-delta.png` | keys at 46304 ms: 44.45.4C.54.41 |
| `w07-12-typed-in-draft.png` | asserted at 47423 ms: 'delta' found |
| `w07-13-esc.png` | keys at 48920 ms: 1B |
| `w07-14-back-in-composer.png` | asserted at 50289 ms: 'delta' found |
| `w07-15-ctrl-g-again.png` | chord at 51495 ms: 11.47 |
| `w07-16-enter.png` | keys at 52972 ms: 0D |
| `w07-17-chip.png` | asserted at 54185 ms: 'Long message' found |
| `w07-18-chip-lines.png` | asserted at 55436 ms: '4 lines' found |

**W08 narrow, 100x30** (`-outName p73-w08-narrow`, folder `p73-w08-narrow-claude2-100x30`; the log's last line: "assertions: 3 passed, 0 failed"; exit 0).

| Screenshot | What it records |
|---|---|
| `w08-01-end-of-session.png` | asserted at 28231 ms: 'COMPLETED' found |
| `w08-02-up-to-chip.png` | keys at 31489 ms: 26.26.26.26.26.26 |
| `w08-03-chip.png` | asserted at 32662 ms: 'Long' found |
| `w08-04-hint.png` | asserted at 33816 ms: 'Enter full view' found |

## What only the owner can do

Plan 73's **Manual Human Evaluation** (`plan.md`), none of it done or claimed here, on a mock session, in Windows Terminal, maximised and at
120x40: type 5 and 50 lines; paste 200 lines, Enter, the chip with the session's work beside it; paste, Ctrl+G, edit, Enter; paste, Ctrl+G, Esc,
the chip back; edit then Esc, ordinary text; instructions after and before a chip, with and without Ctrl+J; several chips; Backspace, Delete,
Ctrl+Z, Ctrl+Y; a selection across a chip; a paste ending in `\`; a long Draft sent; ↑, Enter and a double click on the chip; the search reaching
into it; `y`, pasted elsewhere and compared with the text, and whether a very large one stalls the terminal (P5); resize, dock the Explorer,
80x24 and 200x60; the glyph and colours. Then, only if separately authorized, one short live turn per CLI. Committing and pushing wait for
the owner's approval.

**Reconciled:** the third review agreed seven scenarios, the six above with undo, and a large clipboard as a seventh; `plan.md`, Manual Human
Evaluation, now lists them.

**Since then,** the final end-to-end verification, above, drove the paste, `y` with its read-back, the mouse and the large clipboard natively, and one live turn per CLI. It does not replace this
evaluation: the judgement by eye and whatever the owner wants to try by hand stay the owner's, with the decision on F1, then committing and pushing.

**Done, 2026-11-02:** the owner approved 73.30 on the evidence, with no manual session ("Approval and closure", above).

## What the plan asked to verify (V1 to V8)

- **V1, every way a text change reaches `m.edit`.** The text is assigned in four places only:
  `apply`, `Undo`, `Redo`, `SetText` (`git grep -n "e.text =" internal/input/editor.go`).
  `Paste`, `Insert`, `Newline`, `Backspace`, `Delete`, both word deletions and `Cut` go through
  `apply`, which runs `edited`; `Undo` and `Redo` set the ranges of their entry; `SetText` and
  `Clear` end them. The summary's and the branch switch's `Insert` run only on an empty draft
  (`summaryoffer.go:77`, `branches.go:607`). `TestEditorRangesModel` and `FuzzEditorRanges` drive
  every one of these operations against a plain model.
- **V2, the mode at Draft's opening and closing.** `ui.DraftWin` opens at `editing.go`'s Ctrl+G and
  closes at its Esc, F2, F3 and F6 and after a send, and at `closeSession`; every one runs inside
  `Update`, which ends in `syncUI`, on every path but the quit (`leaveMsg`). The mode is set there,
  just before the view is built. **A difference from D10's wording:** D10 says "at the top of
  `Update`". At the top of an `Update` the mode already holds, from the end of the one before;
  set only at the top, the frame Ctrl+G opens would draw the compact form. No path changes
  `DraftWin` and then edits in the same `Update`: F2, F3 and F6 go on to their own keys. The wheel's
  `Move` in Draft (`mouse.go:256`) runs with the mode on: `TestDraftWindowMouse`. **Reconciled:** D10 now
  says this, and no code changed (the review's R2).
- **V3, the mouse on and beside a chip.** `TestChipClicks`: a click on it puts the cursor at one of
  its edges, a click beside it where it falls, a double click selects the chip whole, a triple
  click its compact line, a drag and a Shift+click hold it whole; `TestChipCursor`: 1,500 random
  motions and clicks, with and without Shift, on three chips, two of them adjacent. Windows
  Terminal keeps Shift+click for itself: the owner's step.
- **V5, the golden frames.** Below, "Golden frames".
- **V6, `Expandable`'s and `Targets`' callers.** `Targets`, `entryKey` and `searchWhole` take the
  width from `MessageWidth`, from the layout. Where zone B is not drawn (the panel in its place) the
  width is 1 cell, so a message counts as long; no cursor moves there, the focus being elsewhere,
  no paste reaches the composer (`draftBeside`), and the moment zone B is drawn again its own width
  decides. `TestMessageWidthIsDrawn` checks the layout's width against what zone B wraps to at five
  sizes, docked and not.
- **V7, what `y` puts on the native clipboard and what a paste of it gives back.** The fake
  clipboard receives the exact text (`TestChipFullView`). The real clipboard is the owner's step.
- **V8, how `tea.Raw` writes a long OSC 52 sequence.** Read in Bubble Tea v2.1.0, `tea.go`: the
  copy, native and base64, runs in a command, off the event loop. The `copiedMsg` then returns
  `tea.Raw(seq)`, whose `RawMsg` the event loop handles with `p.execute`: it copies the sequence
  into `outputBuf`, under `p.mu`. The write to the terminal is `p.flush`, on the ticker's goroutine,
  which holds `p.mu` for the whole `Write`. So a very long sequence does not block the event loop's
  own work unless it needs `p.mu` during that write, and the frames wait for it. For a 3.35 MB
  message the sequence is 4.47 MB. Whether Windows Terminal takes that without a visible stall is
  the owner's check (P5), `y` being the owner's step.

## Tests that fill the composer by paste (V4)

An overlay of `Editor.Mark` logged the test behind each chip made during a whole run of
`internal/app`, so that this list comes from what ran and not from a search of the sources. Besides
this plan's own tests, these existing ones make a chip, or would have:

| Test | What it asserts | How it keeps asserting it |
|---|---|---|
| `TestKeystrokeToFrame`, `TestDraftKeystrokes`, `TestKeystrokeCost`, `TestKeystrokeAllocs` | a key's cost over a large ordinary editor | `fillOrdinary`, then `ordinaryDraft` before and after the measured keys (C1; 73.23); no chip logged |
| `TestFrameReuse` | a reused frame equals one drawn afresh, over ordinary rows of every kind | its text pasted with Draft open, then Esc; no chip logged |
| `TestPasteToFrame` | a 50,000-line paste kept byte for byte, within 100 ms | measures both paths, in the composer (10 chips) and with Draft open |
| `TestEditorDisplay` | tabs, controls, wraps and the line count in the composer | `fillOrdinary`: the rows it reads are an ordinary draft's |
| `TestSelectorNoRoom` | Alt+P with no room above a ten-line draft | `fillOrdinary`: ten lines drawn |
| `TestDraftWindowMouse` | a drag and the wheel in Draft | pastes in the composer, then opens Draft, which draws and moves over the whole text: unchanged, and it now covers Draft over a range |
| `TestDraftWindowByteForByte` | a draft byte for byte through Draft | pastes with Draft open: no chip logged |
| `TestGoldenFrames`, the `draft` scenes | Draft's frames | a long paste then Ctrl+G: 26 chips logged, and no `draft-*` frame changed (73.18) |
| `TestInputByteForByte` | stored and sent bytes on each adapter | text alone; 9 of its cases are now chips, the same bytes demanded of them |
| `TestPasteCtrlV` | the paths' fallback goes in as text | text alone |
| `TestEntryViewer`, `TestConversationScroll` | a sent long message's form | changed to the chip's words, as the owner asked |
| `TestQuitDuringTurn`, `TestQuitBeforeAccepted`, `TestDraftRestored`, `TestDraftFailedSendKeeps`, `TestDraftAcceptedDeletes`, `TestCrashAndResume`, `TestCodexCrashAndResume` | a draft's persistence and delivery | text alone, which a chip does not touch |

Every other `tea.PasteMsg` of the suite is short enough to stay text: no chip logged. The tests whose
assertions changed because the owner asked for the behaviour: `conversation_test.go`,
`viewer_test.go`, `search_test.go` (render), `view_test.go` (its chips being shorter than the fold,
"B scrolls to it" needs 40 messages instead of 12 to scroll), `select_test.go`, `search_test.go`
(app), `searchtiming_test.go` (a comment), `convclick_test.go` (the click's target),
`command_test.go` (`y` in a message's view now copies it), `mdspans_test.go` (the message drawn raw
is now short enough to draw), and `thinking_test.go` for `Expandable`'s width.

## The plan as it ran

`plan.md`'s section of plan 73 as it stood when the owner approved it on 2026-11-02, word for word but for the count the owner corrected, its headings one level down: `plan.md` keeps the current state only (`AGENTS.md`, Working rules), and this file the plan's whole record, as no earlier commit holds it.

### Plan 73 - Virtual attachments for long user messages

Brief amendment 76 (2026-11-01): a long user message takes one row, so that
zone B keeps showing what the agents do. The design went through three rounds
with the owner and the owner's independent reviewer, who read the source at
`1c36ceb`; the owner approved paste ranges and asked for this plan on
2026-11-01, and **authorized its implementation the same day** (Steps, below).
Nothing is committed before the owner's approval. Below, **approved** marks the
owner's decisions and **to
verify** what the implementation must still check. Sizes and costs are
estimates until measured. The owner's independent review of this plan's first
draft returned CHANGES REQUIRED, in documentation and criteria only: its
amendments A1 to A9 and its findings R1 and R2 are marked below where they are
met.

**The invariant (approved).** Presentation metadata is never a source of
truth for the message, and invalid metadata never causes content loss or
corruption.

#### Objectives and scope

- **Zone B.** A user message that draws more than 3 rows shows as one row,
  `▣ Long message · 404 lines · 18 KB`, whatever its origin: no preview, no
  box, no expansion in place, no file.
- **The composer.** A paste whose own content draws more than 3 rows becomes
  such a row at once. Typing, growth, scrolling and keys stay as they are.
- **Draft.** Unchanged in its drawing, keys and editing, with no conversion.
- **Reading and copying.** The existing full view reads the message; `y`
  copies its exact text.
- **In scope**: `internal/render` (the rule, the chip's row, the search's fold
  row, the composer's chip cells, the full view's hint), `internal/input` (the
  ranges in `Editor`), `internal/app` (the paste site, the send key's `\` rule,
  the composer's view and mouse mapping, `y`), the documents and the tests.
- **Out of scope**: `internal/state` and `internal/provider`; the send path;
  the stored draft's format; Draft's drawing and keys; answers, notices and
  tool rows; zone A; session resumption, conversation history, model memory,
  compaction; any file for an attachment; editing a sent message; any new
  dependency.

#### What was found (2026-11-01, on 1c36ceb)

- **The fold counts logical lines.** `entryBody` (`cards.go`) shows a message
  over 20 lines as its first 5 and "… N more lines" (`foldOver`, `foldShow`,
  `content.go`) and ignores wrapping: up to 20 lines, or one huge line, is
  drawn whole. That is the owner's problem.
- **What a message draws.** `entryLines` is `strings.Split(e.Text, "\n")`,
  `wrapRanges` gives an empty line one row, `cardText` draws every range: a
  final line break draws a blank row. User text is never parsed as Markdown.
- **What can be reused for a chip.** The fold row's `crow.more` and `refFold`,
  `rowEnd`, `ConvMatchRow`, `lightSearch`; the entry cursor (`Expandable`,
  `Targets`, `entryKey`); the full view (`entryViewer`, `openView`, from its
  top); double clicks (`convClick`, `opens`); the search's Tab (`searchWhole`,
  `EntryViewRow`); the image rows' look (`imageRow`, `byteSize`).
- **One buffer, one send path.** Composer and Draft are `m.edit`, drawn from
  one `ui.Editor`. `draftWinKey` goes through `editorKey`, Esc keeps the text;
  a send is `state.Send{Text: e.Text()}` for both. `CmdWrite` clears the
  editor; `CmdRestoreDraft` puts text back with `SetText`.
- **One paste site.** A bracketed paste is one `tea.PasteMsg`; it, the raw
  Ctrl+V and the paths' fallback all reach the one `m.edit.Paste` in `m.paste`
  (`app.go`). Typed keys go through `Insert`.
- **One width.** The composer's text and a card's content are both `inner - 3`
  (`draftRows`, `cardContent`).
- **Costs today.** A visible long message is split whole every frame
  (`entryRows`, `entryLines`), and `Expandable` splits every user entry on each
  cursor step; the full view re-wraps the whole message every frame
  (`entryViewer`); its copy takes rows as drawn, tabs as four spaces.
- **Scans of the whole text (R1).** `editorView` reads `CursorLine`,
  `LineRange`, `Lines` and `linePos`, and `Move` reads `lineStart` and
  `lineEnd`: each walks `m.edit`'s text. On a 50,000-line chip they would walk
  hidden bytes on every key or frame, and undo the compact form's gain, unless
  they run on the compact form (D11).
- **Guards that fill the composer by paste.** `TestKeystrokeToFrame`,
  `TestDraftKeystrokes`, `TestKeystrokeCost`, `TestKeystrokeAllocs` and
  `TestFrameReuse` paste to get a visible draft, and `TestPasteToFrame` measures
  a paste itself. Under this plan such a paste is a chip, so they would keep
  passing while no longer measuring what they were written to measure.
- **The clipboard.** `x/clipboard` v0.11.0 converts UTF-8 to UTF-16 and leaves
  line breaks alone (`clipboard_windows.go:241`); a NUL fails its conversion,
  and `Copier` then falls back to OSC 52. Every copy also prints an OSC 52
  sequence (`app.go`, `copiedMsg`).
- **The harness.** `scripts/harness/wt-check.ps1` holds no clipboard code and
  never will (plan 51), so a real paste and `y` are the owner's steps.

#### Design (approved by the owner, 2026-11-01)

- **D1 One definition of a visual row.** A text's visual rows at width `w` are
  the sum, over the lines of `strings.Split(text, "\n")`, of
  `len(wrapRanges(line, w, w))`: what `entryLines` and `cardText` draw, an empty
  line a row. `w` is the message width, `cardContent(cardYou, inner)`, read from
  `layout`, not from the last frame. A text is **long** when its rows exceed 3
  (`messageRows`). The `YOU` row, image rows and separators do not count, and no
  line is left out: `a\nb\nc\n` draws 4 rows, the last blank, and is long; so is
  `\n\n\nabc`; `a\nb\nc` is not. The composer's own character wrap plays no part,
  so a paste alone in the composer is a chip exactly when it would be one in
  zone B at that width.
- **D2 Bounded.** `LongMessage(text, w)` gives the same answer as that sum, for
  every text, without reading the whole text: it stops at the fourth line; with
  `most` = `3 × (w + 1)` and a cell wider than `w` counted as `w`, a line over `most`
  cells is over 3 rows unwrapped, a row holding at most `w` cells or one cell wider
  than itself (`wrapCells`) and a wrap dropping at most one space at its edge; a
  line within `most` goes through `wrapRanges`, or is one row when its real width
  fits. That counts cells, not bytes, so a line of what takes no cell is read to
  its end (`doc/plan73-evaluation.md`, R-I). Corrected after the third review: a
  cell was counted at its full width, which made a chip of a line drawing 3 rows
  or fewer (R-H; the same file, The correction of R-H).
- **D3 The chip's words.** `▣ Long message · N lines · S`, one span in the
  muted colour. `N` is logical lines, `strings.Count(text, "\n") + 1` through
  `grouped`, "1 line" for one, the count the full view's head shows, so the words
  do not change on a resize. `S` is the text's UTF-8 bytes through `byteSize`.
  Zone B's chip describes the whole message, a composer chip the bytes it hides.
- **D4 Zone B.** In `entryBody`, a long message is its `YOU` row, the chip's row
  and its image rows as today; any other is today's rows. This replaces the
  20/5 fold: `foldOver`, `foldShow` and a message's "… N more lines" go; command
  blocks keep theirs. The chip's row is the fold row's `crow` (`more`, `refFold`,
  its entry), and no line of the text is split or drawn. Under the entry cursor
  it has `▸` and `entryHint` on its right, its size dropped first when the row
  has no room, then `spread`'s cut. `Expandable` takes a width and asks D2, and
  `Targets` with it.
- **D5 Reading, copying, searching.** ↑ from an empty input then Enter, or a
  double click, opens the existing full view from its top, every line, and Esc
  returns; nothing expands in place. `y` there copies `Entry.Text` byte for byte
  through the usual copy and toast, beside the command's case in `viewKey`, with
  the hint `y copy the message · `. Its head says "1 line" for one. A selection
  across a chip copies the whole message, as one across a fold does today. The
  search finds its text and lights the chip's row, and Tab opens the full view at
  the match; `ConvMatchRow` and `lightSearch` lose their `foldShow` conditions.
- **D6 Resizing.** Zone B decides at the current width in every frame, docked
  panels included, with no stored state; a message of 4 or more logical lines is
  a chip at every width, so only one of 1 to 3 can change form. The row cache
  already keys on the width. An entry cursor on a message that stops being a chip
  leaves it. A composer chip stays a chip.
- **D7 A long paste in the composer.** At the one `m.edit.Paste`, with Draft not
  on top: when the pasted block, as stored after the line-break rule, is long
  (D1), its bytes become a range, in the paste's undo step. It is judged on the
  block alone, once, never on the buffer's total; typed text never makes one. One
  final `\n` of the block stays outside the range as ordinary text, so that the
  cursor's row says where typed text goes: `a\nb\nc\n` shows `3 lines · 5 B` with
  the cursor on the row below, then `4 lines · 6 B` in zone B once sent.
- **D8 Ranges.** `input.Editor` gains a sorted list of ranges: start, end, line
  count; offsets only, no bytes. `Text()`, the send, the stored draft and `state`
  never read them. An edit in the composer before a range shifts it, after it
  leaves it, touching it drops it. Each undo entry keeps the ranges from before
  and after its edit, so undo and redo put back only ranges that match their
  exact bytes. `SetText` and `Clear` end them all: a send, a failed send's
  restored text, a session's close; such text comes back ordinary. A range
  failing a bounds or order check when read is dropped, its text shown as
  ordinary.
- **D9 The composer with a chip.** Only while a range exists:
  - the cursor moves over a chip as one unit and never stops inside it, by any
    motion or click: the motions run on the compact form, where a chip is one
    byte (D11);
  - Backspace or Ctrl+Backspace right after a chip, Delete or Ctrl+Delete right
    before it, removes the whole block in one undo step, through `apply`: the one
    text change ranges decide, asked for by the owner and undoable;
  - a word deletion that would cross into a chip stops at its edge, as
    `wordLeft` and `wordRight` otherwise reach hidden bytes;
  - a selection holding a chip holds it whole; delete, typing, cut and copy act
    on all its bytes;
  - Enter right after a chip sends: the `\` newline rule applies only to a `\`
    the owner can see;
  - text typed at either edge stays outside it, with no character added. Typed
    right after a block with no final line break it stays on the chip's row, as
    it joins that line in the bytes; Ctrl+J starts a new line;
  - growth and scrolling are today's, over the rows drawn.
- **D10 Draft.** Draft shows `m.edit` whole with today's view, keys and cursor;
  ranges neither draw nor bind the cursor there, and nothing converts on a paste
  or typing. **Any text change while Draft is open ends every range**; edits
  inside a range are not tracked. Esc after no change: the chips return. After a
  change: ordinary text under today's rules. Undo back to the original bytes: the
  chips return (D8). A cursor left inside a range's bytes is drawn at the chip's
  end; the first composer key or paste moves it there and clears a selection with
  an end inside a chip, and no text changes. The application sets the mode from
  `ui.DraftWin.Open` in `syncUI`, just before it builds the composer's view, at the
  end of every `Update` but the quit's: so the mode holds at the top of the next
  `Update` and for the frame Ctrl+G opens. While Draft is open the whole
  dashboard draws as today, the composer behind it included.
- **D11 The compact form (R1).** While a range exists and Draft is closed, every
  composer calculation of visible lines, cursor position and viewport runs on a
  display string: `m.edit`'s text with each range as one placeholder byte. That
  covers `editorView`'s `CursorLine`, `LineRange`, `Lines` and `linePos`; `Move`'s
  `lineStart`, `lineEnd`, goal column and word search; the mouse's `Offset`,
  `LineRange` and `WordBounds`. They are the existing functions, given that
  string, and offsets map back to `m.edit` through the ranges, never inside one.
  No key or frame visits a hidden byte for these, so that their cost does not depend
  on a block's size, as 73.26 measures. The initial strategy builds the string from
  the visible segments for one call and drops it. It is derived, never edited, stored
  or sent, so it is no second message buffer, and `m.edit` stays the only source of
  truth. No cache is built now, and none is ruled out: if measurements show a derived
  one necessary and beneficial, a bounded and correctly invalidated one may be
  considered under P6, the text's status unchanged. `EditorView` says where
  chips go: `rowsOf` and `rowCount` put a chip's cell there (the existing `cell`,
  `sub`), and `drawRow` draws it as an atom. The row's line count is the message's,
  from the display string's lines and the ranges' recorded counts, shown when the
  composer draws more than one line, so a chip alone shows only its own. What still
  walks the whole text is its storage, as for any draft today: an edit's splice in
  `apply`, and the stored draft's save, at most every 500 ms and off the event loop.
- **D12 As they are.** Zone A keeps the prompt's first line. Under D4 an
  instruction added to a block is inside zone B's chip once sent; typed first, it
  is what zone A shows. Session resumption stays each CLI's own.

#### Performance and cache

- **P1** No line of a chip's text split or wrapped on a frame's path in zone B.
  `LongMessage` allocates nothing for a text decided by its line breaks or a
  line's width. A chip's `N` is one `strings.Count` when drawn, measured by 73.25,
  with a memo only under P6.
- **P2 Hidden text (R1).** With a range and Draft closed, no key or frame visits a
  hidden byte for the view or the cursor (D11): ranges adjust in their number, a
  chip's line count is recorded with its range at the paste, and the row's count
  needs no scan. No second message buffer (D11). A text change still copies the
  string in `apply`, as every draft's does today: that cost is measured, not
  hidden (73.25, `TestLongPasteKeys`).
- **P3** No range, no cost: every new branch of the editor and the view is
  skipped, so typing and Draft run today's code.
- **P4 The full view, a remediation (R2).** Its cost on a 50,000-line message is
  measured first. If it misses its budget and the cause justifies it, its wrapped
  rows get a memo by entry and width, bounded and swept as the others are, and it
  is measured again (P6). Nothing is built before that measure says so.
- **P5 A very large `y` (R2).** The key's cost in `Update` is measured, the copy
  running off the event loop as today. Whether the OSC 52 sequence stalls the real
  terminal is the owner's check, `y` being the owner's step. A stall goes through
  P6.
- **P6 The sequence, for every budget (R2).** (1) Measure the implementation. (2)
  If it exceeds its budget, investigate, and apply a remediation if justified. (3)
  Measure again. (4) The final implementation must meet the stated budget. (5)
  Otherwise record a FAIL and ask for the owner's explicit decision. A remediation
  is never an alternative success condition: no criterion is PASS because an
  optimization was identified or attempted.
- **P7 Baseline and comparisons (A1, A3).** Before any test or application code
  changes, the whole timing step of `doc/evaluation.md`, alone and under its
  conditions, and the deterministic guards `TestKeystrokeAllocs` and `TestFrameReuse`
  run on the unchanged application code; their numbers and conditions go in
  `doc/plan73-evaluation.md`. This is a step of the implementation, not run before
  it is authorized. Thresholds stay as they are. Two comparisons follow, each over
  the same document size, cursor positions, operations and rendering conditions
  (terminal size, focus, session), each reporting p50, p95 and allocations as its
  test gives them:
  - **C1**, the original fill against the replacement fill, on the unchanged
    application code;
  - **C2**, the replacement fill before against after the feature.

  The replacement fill pastes the same text with Draft open, then Esc: the same
  `Editor.Paste`, one undo step, the cursor at the text's end, and no range. The
  long-paste-to-chip path keeps separate coverage with its own budgets
  (`TestPasteToFrame` in the composer, `TestLongPasteKeys`). It never stands in for
  the ordinary editor's workload, nor the reverse.
- **P8 Regressions (A4).** Every measurement of P7's comparisons is reported with
  its absolute value and its change from its counterpart, absolute and relative. No
  percentage decides whether a change matters. Comparisons alternate before and after
  under the same conditions, at least three pairs, and report the spread of each
  side's own runs beside the change. Repeated, consistent changes exceeding observed
  measurement variability provide evidence of a regression, and such a change is
  **reproducible**. Inconsistent or noisy results are classified as **inconclusive**,
  not automatically as measurement variation or absence of regression: additional
  controlled measurements are performed when necessary, all results stay documented,
  and unresolved regressions are escalated to the owner. The absolute budgets stay as
  they are. Then:
  - a missed absolute budget is always a FAIL, with its evidence and likely cause, and
    P6 applies;
  - a reproducible degradation within its budget is never dismissed for being small:
    `doc/plan73-evaluation.md` gives its size, its likely cause, the budget left,
    whether it grows with the document or the session, and whether it contradicts P3
    where no range exists; materiality and acceptance come from that evidence;
  - a significant or unresolved regression, an inconclusive result that stays so among
    them, is recorded as a FAIL and goes to the owner for a decision; one found
    immaterial is named in the evaluation's summary with its evidence, so that none is
    accepted silently;
  - tests and thresholds are never weakened to obtain a PASS, and workloads not
    redefined.

#### To verify during the implementation

Implementation details, not design decisions, each reported with its evidence
in `doc/plan73-evaluation.md`:
- **V1** every way a text change reaches `m.edit`: `Paste`, `Insert`, `Newline`,
  `Backspace`, `Delete`, the word deletions, `Cut`, `Undo`, `Redo`, `SetText`,
  `Clear`, the summary's and the branch switch's `Insert`;
- **V2** the mode at each of Draft's opening and closing paths (`editorKey`,
  `draftWinKey`, `closeSession`), and the wheel's `Move` in Draft;
- **V3** a click, double click, triple click, drag and Shift+click on and beside
  a chip;
- **V4** every test filling the composer by paste, which behaviour it asserts,
  and how it keeps asserting it, beyond those named under What was found
  (`TestDraftWindowByteForByte` and `TestDraftWindowMouse` among them);
- **V5** the golden frames that change, reviewed by script;
- **V6** `Expandable`'s and `Targets`' callers with the width from `layout`,
  where zone B is not drawn;
- **V7** what `y` puts on the native clipboard and what a paste of it gives back,
  in the owner's step;
- **V8** how `tea.Raw` writes a long OSC 52 sequence, on the event loop or off it.

#### Risks and fallbacks

- A range wrong through a bug: dropped when it fails its check, its text shown
  ordinary; a whole-block deletion goes through `apply`, so Ctrl+Z restores it.
- Many ordinary prompts become chips, about 207 to 219 characters being 3 rows at
  80x24 and 120x40: accepted by the owner, subject to the visual evaluation, and
  one constant.
- A 3-line block with a final line break is a chip: accepted, by D1.
- Guards silently measuring a chip: their fills change first, on the unchanged
  code (Steps), each shows its workload (73.23), and new tests measure the chip's
  path.
- A full view or a `y` over budget on very large text: P6's sequence, ending in the
  budget met, or a FAIL and the owner's decision.
- Each CLI's own limit on a very large message: not probed; a refusal before
  acceptance returns the text to the editor, as today.

#### Criteria

- **73.1 The definition.** `LongMessage(text, w)` equals `rows > 3` for `rows` the
  sum of D1, itself what `cardText` draws, at widths 49, 69, 73 and 129: `a\nb\nc`
  false; `a\nb\nc\n`, `\n\n\nabc`, `a\n\n\n\n` true; a line filling exactly 3 rows
  false, one cell more true; tabs, control characters, wide clusters at a row's
  edge, a long word, spaces at a row's edge; a cell wider than a row, as characters
  joined by U+200D make: one or three false, whatever their width, four true; a
  large line of zero-width characters false; `FuzzLongMessage`, with such cells among
  its seeds, finds no difference in 60 s, in either direction, at any width from 1
  to 140, and neither do `TestLongMessageRenderer` and `TestLongMessageEquivalence`
  against `cardText` in every run. Restored after R-H (`doc/plan73-evaluation.md`,
  The correction of R-H).
- **73.2 The bound.** With `most` = `3 × (w + 1)` and a cell wider than `w` counted
  as `w`, `LongMessage` reads, by its test hook `messageScan`, at most 4 lines and,
  in each, cells summing to at most `most` and the one cell that passes that sum: at
  most `most + w` in a line and `4 × most + w` in all. A line wider than `w` and
  within `most` is then wrapped once, over the same cells. On a 50,000-line text and
  on one 3 MB line, at widths from 1 to 129, it reads exactly that, with 0
  allocations for a text decided by its line breaks or a line's cells. The bound
  counts cells, not bytes: what takes no cell is read to the line's end, and a
  cluster of any length is one cell (R-I). Corrected twice after the third review:
  this said "at most `3 × (w + 1)` cells", counting a cell at its full width, then
  "`3 × (max(w, 4) + 1)`" with the same count (`doc/plan73-evaluation.md`, The
  independent review; The correction of R-H).
- **73.3 The chip's words.** Exactly `▣ Long message · 404 lines · 18 KB` for 404
  lines and 18,000 bytes; `1 line · 15 KB` for one 15,000-byte line; `1,204 lines`;
  `4 lines · 6 B` for `a\nb\nc\n`; `4 lines · 39 B` for four lines of `日本語`,
  UTF-8 bytes and not characters (A7); `1.2 MB`; the same at every width; the full
  view's head gives the same count, "1 line" for one.
- **73.4 Zone B's two forms.** At 80x24, 120x40, 200x60 and 208x59: 1, 2 and 3
  rows are today's rows, each span byte-exact; 4 rows and 404 lines are the `YOU`
  row and one chip row, no byte of the text in the frame, no box and no "more
  lines"; a message from the composer, from Draft and from a replay gives the
  same frame.
- **73.5 With images.** Long text and images: the chip's row, then one row an
  image as today; images and short text: today's rows.
- **73.6 Resizing.** One message at widths where it draws 3 and 4 rows: normal,
  then chip, then normal; docking the Explorer the same; 4 logical lines a chip at
  every width; the state unchanged by all of it; an entry cursor on it leaves when
  it stops being a chip; a composer chip unchanged.
- **73.7 The cursor and the full view.** ↑ from an empty input stops on a chip
  with `▸` and the hint, on no normal message; B's height unchanged by the
  cursor; Enter opens the full view from its top with every line, Esc returns; a
  double click on the chip's row or the `YOU` row opens it.
- **73.8 `y`.** In a long message's full view `y` gives the clipboard exactly
  `Entry.Text`: tabs, trailing spaces, indentation, blank lines, Unicode, a wrapped
  15,000-cell line with no line break added; the toast `copied 404 lines · native`;
  its hint shown; a command's `y` and other views unchanged; the submitted text
  unchanged.
- **73.9 Search and selection.** Text inside a chip is found and the chip lit,
  the current match in its own style; Tab opens the full view with the match's row
  in view; a selection across a chip copies the whole message byte for byte.
- **73.10 Delivery, both providers.** For Markdown, fenced code, tables, JSON,
  YAML, PowerShell, Unicode, indentation, tabs, empty lines, a final `\`, no final
  line break and 50,000 lines, pasted in the composer, in Draft, edited in Draft,
  and mixed with typed text: the line each adapter writes decodes to exactly
  `Editor.Text()`, equal to the input under the paste's line-break rule, and to
  `Entry.Text`; no placeholder byte in any of them.
- **73.11 Typing unchanged.** Typing any number of lines makes no range; every
  existing golden frame of a typed draft, the cap `(h-2)/2-3`, its scrolling and
  the key tables unchanged, but for `y`'s row and the `↑ ↓` row's word `folded`, now
  `long`, the message's fold being gone (D4). Corrected after the third review (its
  R3): this named `y`'s row alone.
- **73.12 A long paste.** 3 rows stay text, 4 rows are a chip at once, nothing
  sent; 2 typed lines then a long paste: the lines visible and editable, the chip
  after them; 10 typed lines then a 2-line paste: no chip; several short pastes:
  no chip; the raw Ctrl+V and the paths' fallback the same; a paste with Draft
  open: no range.
- **73.13 A chip's edges.** `a\nb\nc\n` shows `3 lines · 5 B` and the cursor on
  the row below; a block with no final break takes typed text on its row with no
  byte added; pasted between typed text, the text on both sides intact; the row's
  count `201 lines` with a typed line, none with a chip alone.
- **73.14 Two chips (A5).** Two long pastes make two independent chips, with
  ordinary text typed before, after and between them: that text visible and
  editable, each chip's words its own; deleting and undoing one leaves the other
  and the text byte-identical; sent, the message is exactly those blocks and that
  text in that order, on both adapters.
- **73.15 The cursor.** Every motion, its Shift form and a click: never inside a
  range, by a property test; a selection holding a chip holds it whole.
- **73.16 Deletion, undo and the clipboard.** Whole-block deletion from each edge
  by each of its four keys; a word deletion stopping at an edge; Ctrl+Z and Ctrl+Y
  restoring bytes, chip and cursor; a selection holding a chip deleted, typed over
  and cut; several and adjacent chips; unrelated text byte-identical in all of them.
  Copying or cutting a selection holding a chip, with typed text on either side,
  gives a test clipboard exactly the selected original bytes, by Ctrl+C, Ctrl+X and
  a drag's copy on release (A6).
- **73.17 Enter and `\`.** A paste ending in `\`: Enter sends, bytes unchanged; a
  typed `\` after a chip then Enter: a newline, as today; in Draft as today; during
  a paste's open markers no key sends; a turn running: the chip and text kept, the
  notice as today.
- **73.18 Draft.** Every `draft-*` golden frame unchanged; its frames and keys the
  same with and without ranges; no change then Esc: chips back, the cursor where
  it was on reopening; an edit: ranges gone, text ordinary, nothing lost; undo to
  the original bytes: chips back; an invalid range injected: dropped, the text
  intact.
- **73.19 Restored text (A7).** A failed send's text comes back as ordinary text,
  byte-identical to what was sent, which differs from the pasted input only by the
  paste's existing line-break rule; with text typed since, joined as today; a
  restored stored draft: ordinary text, byte-identical.
- **73.20 Model test.** Random sequences of every editing operation, in both
  modes, against a plain model: the text equals the model's after every operation,
  whole-block deletions applied to both; the ranges sorted, in bounds and matching
  their recorded line counts; `FuzzEditorRanges` finds no failure in 60 s.
- **73.21 Golden frames (A8).** New scenes for a chip, its cursor, a composer chip
  and typed text beside it, at the golden sizes; at 100x30, zone B's narrowest form
  (content width 49), a chip's row unfocused and focused, each one row, the focused
  one with its hint whole; every changed frame reviewed as a diff by script, nothing
  changed but what this plan changes.
- **73.22 The scope.** `git diff` shows nothing under `internal/state` and
  `internal/provider`, and no change to `go.mod` or the stored draft's format; no
  file written for an attachment at run time; architecture test 7 passes.
- **73.23 The guards measure what they were written for (A2).** Each test meant for
  a large ordinary editor (What was found, V4) shows, before and after its measured
  keys, that it measures that content: the editor holds its full size; the view draws
  ordinary rows of that text, `EditorView` naming no chip and the frame holding no
  `▣ Long message`; the row under the cursor shows the text at the cursor; a typed
  key changes both the text and that row. The check is of the view and the workload,
  not of the metadata: Draft may keep ranges while showing ordinary text, and passes.
  Run on a damaged input, the original composer-paste fill under the feature, it
  must fail.
- **73.24 Baseline and comparisons (A1, A3).** `doc/plan73-evaluation.md` holds P7's
  baseline from before any test or application code changed, then C1's and C2's
  tables, by test: document size, cursor positions, operations, rendering conditions,
  p50, p95 and allocations, absolute and relative, each run's conditions as
  `doc/evaluation.md` requires. **Functional equivalence** decides a replacement fill:
  in C1 it gives the original's text byte for byte, the same cursor, line count and
  undo steps, and the same rows drawn. **Performance** is compared and reported as P8
  says: a reproducible difference in C1 is explained, as the two fills should cost the
  same, but no tolerance by itself invalidates a functionally equivalent fill.
  **Acceptance** rests on the existing absolute guards and budgets (73.25) and on P8's
  evidence-backed assessment of any change in C1 or C2.
- **73.25 Budgets (R2, A4).** The final implementation, under P6, the budgets as stated:
  `TestLongMessageToFrame`, a 50,000-line chip in view and 400 answer deltas, p95
  within 16 ms; `TestLongPasteKeys`, keys around a 50,000-line chip, p95 within 16
  ms; `TestLongMessageView`, the full view of 50,000 lines, p95 within 16 ms;
  `TestPasteToFrame`, in the composer, the long-paste-to-chip path, and with Draft
  open: the worst within 100 ms; a frame with a chip in view allocating no more than
  with a 3-row message; `y`'s key within 16 ms in `Update`; every existing timed test
  and guard at its unchanged threshold. A miss is a FAIL with its evidence and likely
  cause (P8), and goes to the owner for a decision; a remediation identified or
  attempted is not a PASS. A reproducible degradation inside a budget goes through
  P8's assessment.
- **73.26 Hidden text (R1).** With a range and Draft closed, cursor and viewport
  calculations run on the compact form, and no hidden byte is traversed on the chip's
  drawing or a motion's path: `EditorView`'s lines hold no hidden byte, and its line,
  column, selection and total are the display string's. Each mutant putting `Lines`,
  `CursorLine`, `LineRange` or `linePos` back on the whole text on that path is
  detected, by a value it changes or else by the cost it adds against the unmutated
  build. `TestChipScale`, in the timing step, records the median of 200 calls of the
  view and of each motion (← → ↑ ↓ Home End, the word moves, their Shift forms) around
  chips of 5 KB, 500 KB and 5 MB, for a block of many lines and for one line, and
  `doc/plan73-evaluation.md` documents how the cost changes with hidden size. A ratio
  of 5 between 5 KB and 5 MB is a provisional diagnostic unless the measurements
  justify it: staying under it proves nothing about traversal by itself, and a cost
  that grows with hidden size on these paths is a finding against R1, reported under
  P8. `m.edit` stays the only authoritative and mutable text, by reading the diff.
- **73.27 The checks.** Build, vet, gofmt, the whole suite after the last edit,
  the fuzz targets, at least one mutant through `-overlay` per rule of D1 to D5 and
  D7 to D11, each detected or named equivalent with its reason, `-race` on the
  changed packages, the timing step alone.
- **73.28 The documents.** `ui-design.md` (Zone B's entries, Long pasted messages,
  Expanding; Search; Scrolling; the Input editor's Paste and a section for long
  pastes; Draft window; the key tables; Selection and copy), `architecture.md`
  (ASM-22, the matcher's row, the Input part), `evaluation.md` (the timing list and
  count, the fuzz list), the overlay's `y` and `↑ ↓` rows in `keys.go`, this plan,
  `doc/plan73-evaluation.md`; UTF-8, LF, no BOM.
- **73.29 The Windows Terminal scenes (A9).** W01 to W08, below, on the final build:
  each assertion passes, each run exits 0, each scene's evidence is in
  `doc/plan73-evaluation.md`.
- **73.30 The owner's Human Evaluation**, below.

#### Tests

The project's own means, no new framework: unit and model tests in
`internal/input` (ranges, modes, undo); table and fuzz tests in `internal/render`
(the definition, the chip, the composer's cells, the search); app tests driven by
`tea.PasteMsg` and keys through `Update` (paste, Draft, send, `y`, mouse on a drawn
screen), with the existing fake sessions and clipboard; golden frames, new scenes
built from synthetic events and labelled so; both adapters' `Send` for delivery;
the timing step, scale tests and allocation guards; mutants through `-overlay`;
`-race`; architecture test 7. Tests asserting the old fold change because the owner
asked for the behaviour, each reviewed. No threshold is lowered, and no test is
rewritten to make a run pass. No automated test waits for a person.

#### Windows Terminal scenes (A9)

Automated, apart from the owner's clipboard steps below. Each runs
`scripts/harness/wt-check.ps1` on a recording or a mock session, with no CLI, no
quota and no clipboard: no double click, drag, `y` or real paste. Each run's
`-outName` is `p73-wNN-<scene>`, and its step labels `wNN-MM-<what>`, so that its
screenshots are `results\wNN-MM-<what>.png`, sequential and descriptive.
`doc/plan73-evaluation.md` lists, by scene: the run's name, the build's SHA-256, each
screenshot's name and what it shows, the assertions passed and failed from the log's
last line, and the exit code; what matters is written there in words. A composer chip
needs a real paste, so the automated `tea.PasteMsg` tests and the Human Evaluation's
second step cover it in a real terminal's place.

| Scene | Size | What it runs | What it asserts or records |
|---|---|---|---|
| W01 zone-b-chip | 120x40 | `claude-thinking.jsonl`, whose prompt draws 5 rows | `Long message`, and the agent's work beside it |
| W02 zone-b-normal | maximised | The same recording, the prompt drawing 3 rows | The prompt's last words; no chip, read from the saved screen text |
| W03 chip-cursor | 120x40 | ↑ to the chip | `Enter full view` on its row |
| W04 full-view | 120x40 | Enter, PgDn, Esc | The prompt's last words, then `Long message` again |
| W05 typed-composer | 120x40 | Six short lines typed with Ctrl+J | Each line and `6 lines`; no chip, read from the saved screen text |
| W06 typed-send | 120x40 | Enter on W05's text | `Long message` and `6 lines` |
| W07 draft | maximised and 120x40 | Ctrl+G, Ctrl+F, typed lines, Esc, Ctrl+G, Enter | `Draft`; the text back in the composer; `Long message` once sent |
| W08 narrow | 100x30 | W01's recording, the chip focused | `Long`, and `Enter full view` whole |

#### Manual Human Evaluation

Separate from the automated scenes above. In Windows Terminal, maximised and at
120x40, on a mock session, which uses no quota and sends nothing to any CLI; the
owner's own steps, since a real paste, a double click and `y` use the clipboard. The
seven scenarios the third review agreed, from the repository's root, with
`.\acc --mock testdata\mock\claude-thinking.jsonl`, after `COMPLETED`, about 26 s:

1. **Typing.** Type 5 and then 50 lines, Ctrl+J between them: ordinary rows, growth,
   scrolling and the line count as today, no chip. Enter: zone B's chip.
2. **A long paste.** 3 rows stay text. 4 rows are the chip at once, nothing sent;
   `a`, `b`, `c` and a final break shows `3 lines · 5 B` with the cursor below it,
   `4 lines · 6 B` once sent. 200 lines: the chip at once; Enter; zone B's chip with
   the session's work beside it.
3. **Draft.** Paste, Ctrl+G, edit, Enter; paste, Ctrl+G, Esc with no change, the chip
   back; edit then Esc, ordinary text; Ctrl+Z back to the original bytes, the chip
   back.
4. **Several chips.** Instructions after and before a chip, with and without Ctrl+J;
   two chips, also adjacent; Backspace, Delete and the word deletions at a chip's
   edge; Ctrl+Z, Ctrl+Y; a selection across a chip, its copy compared; a paste ending
   in `\`, Enter sending it.
5. **Reading it.** A long message sent; ↑ to it, Enter, and a double click on the
   chip; the search reaching into it; `y`, the clipboard compared with the text.
6. **Resizing and the look.** A one-line message near 3 rows changing form with the
   width; a composer chip staying one; the Explorer docked; 80x24 and 200x60; the glyph
   and colours.
7. **A large clipboard.** 50,000 lines pasted, sent, opened, searched and copied with
   `y`, with any stall of Windows Terminal noted, the clipboard compared with the text.

The texts, set from a second PowerShell tab, and the comparison:

```powershell
Set-Clipboard -Value "a`nb`nc"       # 3 rows: stays text
Set-Clipboard -Value "a`nb`nc`nd"    # 4 rows: ▣ Long message · 4 lines · 7 B
Set-Clipboard -Value "a`nb`nc`n"     # ▣ Long message · 3 lines · 5 B; once sent, 4 lines · 6 B
Set-Clipboard -Value "a`nb`nc`nd\"   # ends in \: Enter sends it
Set-Clipboard -Value ('x' * 200)     # one line: 3 rows at 120x40 and 80x24, 5 at 100x30, 2 at 200x60
$t = (1..200 | ForEach-Object { "pasted line $_" }) -join "`n"; Set-Clipboard -Value $t   # 200 lines · 3 KB
$t = "`ttab first`n  indented, spaces after  `n`nUnicode 日本語 é`n" + ('x' * 15000) + "`nlast line"; Set-Clipboard -Value $t   # 6 lines · 15 KB
$t = (1..50000 | ForEach-Object { "line $_ of a long paste, for the large clipboard check" }) -join "`n"; Set-Clipboard -Value $t   # 50,000 lines · 2.9 MB
((Get-Clipboard -Raw) -replace "`r`n", "`n") -ceq $t   # after y or a copy: True when the clipboard holds exactly $t
```

Then, only if the owner separately authorizes its scope and quota (`doc/evaluation.md`,
live runs), one short live turn per CLI in a temporary project, with the configuration
check before and after: Claude Code on Haiku as a per-process override, Codex on the
model the owner names; a pasted 60-line fixture, asking the agent to report its line
count and last line; one turn each, then stop. The expected usage is proposed with that
request, and no real turn runs before it is authorized. The owner records the result;
the plan closes only on the owner's approval.

#### Components

- `internal/render/content.go`: `messageRows`, `LongMessage`, `MessageWidth`, the
  chip's words; `foldOver` and `foldShow` gone (D1 to D3).
- `internal/render/cards.go`: the chip's row in `entryBody` (D4);
  `viewer.go`: `Expandable`, the head, `y`'s hint; `look.go`: `Targets`;
  `search.go`: the fold row's conditions (D5).
- `internal/render/editor.go`: `EditorView`'s chips, `rowsOf`, `rowCount`,
  `drawRow`, the row's count (D11).
- `internal/input/editor.go`: the ranges, their adjustment and undo, the mode,
  cursor rules, whole-block deletion, the display string and its mapping (D8 to
  D11).
- `internal/app/app.go`: the paste site, the mode in `syncUI` (D7, D10);
  `editing.go`: the `\` rule, the composer's view (D9, D11); `mouse.go`: offsets
  through the display string (D11); `inspect.go`: `y` (D5).
- `internal/app`'s tests: the replacement fill and the workload's check, shared by
  the guards (73.23); the new timed tests (73.25, 73.26).
- `doc/plan73-evaluation.md`, new, from the baseline on (73.24).
- Unchanged: `internal/state`, `internal/provider`, `internal/input/draft.go`.

#### Steps

1. This plan, amendment 76, AGENTS.md's status. **Gate**: the owner's review of
   this plan and authorization to implement, given on 2026-11-01.
2. The baseline (P7, A1): the whole timing step alone and the deterministic guards
   on the unchanged application code; `doc/plan73-evaluation.md` opened with their
   numbers and conditions. Nothing changes before it.
3. The guards (A2, A3): the replacement fill and the workload's check for each test
   of V4, their thresholds unchanged; C1, on the unchanged application code.
4. The rule and the words (D1 to D3), with their tests and fuzz target.
5. Zone B (D4), reading, copying and searching (D5), resizing (D6), their tests;
   the golden frames by script.
6. The ranges in the editor (D8, D10), with the model test and fuzz target: inert,
   nothing makes one yet.
7. The composer (D7, D9, D11), its tests, the hidden-text checks (73.26, R1), the
   long-paste path's coverage, and 73.23's damaged input.
8. The documents (73.28).
9. The checks (73.21 to 73.27): the suite, the golden frames, fuzz, mutants,
   `-race`, the timing step alone, C2 against step 2's baseline (A3); any miss or
   reproducible degradation through P6 and P8.
10. The Windows Terminal scenes (W01 to W08, A9); `doc/plan73-evaluation.md`.
11. The owner's Human Evaluation and review; the commits only on approval.

#### The final review's corrections

The owner's independent review of `acc-plan73-final-verification.zip` (2026-11-01)
approved the core and asked, before closure, for F1's correction (required) and for
zone B's selection to light a chip if a small, safe change does it (recommended). The
provider send path does not change, so no real CLI turn runs. No commit, push, closure,
other plan or removal of temporary infrastructure.

- **F1.** When Ctrl+V's wait ends with no paste, `pasteWaited` asks the clipboard for its
  text, in a command, and `pasteRead` acts on the answer. Text other than blanks is the
  terminal's to paste: nothing is read, attached or said. Otherwise `pasteImages(true)`,
  as before. `pasteImages`, `clipped`, Alt+V and the raw Ctrl+V do not change.
- **The chip's selection.** `highlight` lights a chip's span, the one `refFold`, when the
  selection copies any of its message, from the positions alone: no text read, no row
  added, nothing when no selection exists. A chip stays no selection's end.

Each checked on the corrected build, against the previous one where it says so, on mock
sessions, maximised and at 120x40:

- **73.F1 No false message.** A real paste of 6,569 bytes behind the terminal's warning:
  "nothing to paste" 0 times in TEA_TRACE's whole run and on no saved screen; at least
  once on the previous build, by the same plan. 50,000 lines: on no saved screen.
- **73.F2 The warning refused.** The same paste, its warning refused by its button: 0
  times; no chip and no image's chip; the draft empty.
- **73.F3 Normal paste.** `t1-3lines.txt`, its lines as text; `t1-4lines.txt`,
  `▣ Long message · 4 lines · 70 B`; `t1-200lines.txt`, 4,799 bytes,
  `▣ Long message · 200 lines · 5 KB`; each a bracketed paste in TEA_TRACE, no warning,
  nothing submitted, no false toast.
- **73.F4 Image alone, Ctrl+V.** `pasted-image-1.png`, no toast, the draft empty, on both
  builds.
- **73.F5 Alt+V.** An image alone: its chip. Text alone: `no image on the clipboard`, the
  draft empty. Text and an image: the image's chip, no text in the draft.
- **73.F6 A copied image file.** Ctrl+V and Alt+V: `p73f-image.png`'s chip, its path not
  in the draft.
- **73.F7 Text and an image.** Within 5 KiB, Ctrl+V: the text in the draft, no image's
  chip. Over it: no toast and no image's chip, the warning up or answered,
  `▣ Long message · 90 lines · 7 KB`. The previous build's result recorded beside it.
- **73.F8 The true message stays.** An empty clipboard, and a clipboard of blanks: the
  toast `nothing to paste: no text and no image on the clipboard`.
- **73.F9 Byte-identical.** After each paste the clipboard still holds the fixture; `y`
  after a sentinel reads back 275, 6,569 and 3,649,999 bytes, each equal.
- **73.F10 Regression tests.** The new tests in `paste_test.go`, recorded failing on the
  previous code where they should, pass; their rules' mutants are killed by value, through
  `go test -overlay`.
- **73.S1 The chip lit.** `TestChipSelectionLit`: lit exactly when the copy holds some of
  the message, whatever the selection's ends. Natively: pixels of the selection's ground
  on the chip's row, none on the previous build, some on the corrected one; the copy
  equal to `t3-selection-expected.txt`, 179 bytes; a click and a double click on the chip
  as before. Its cost with a selection reported against the previous code; none without.
- **73.G1 The checks.** `gofmt -l .` empty; `go build ./...`, `go vet ./...` and
  `go test ./...` exit 0, no golden frame changed; `-race` on `internal/app`,
  `internal/render` and `internal/input`; the timing step alone, every timed test within
  its unchanged threshold.
- **73.G2 The scope.** Application code changed only in `internal/app/paste.go`,
  `internal/app/app.go`, `internal/render/select.go` and `internal/render/text.go`;
  nothing under `internal/provider` or `internal/state`, nor `scripts`, nor the terminal's
  settings. The new fingerprint and build SHA-256 stated.
- **73.G3 The record.** `doc/plan73-evaluation.md`, this file, `ui-design.md` and
  `doc/research/image-paste.md` updated, the failed attempts kept.

#### Progress

- 2026-11-01: the design: the owner's discussion, the proposal, the owner's
  feedback and the independent reviewer's corrections; the owner chose paste
  ranges and asked for this plan (amendment 76).
- 2026-11-01: step 1: this plan, amendment 76 and AGENTS.md's status, nothing
  else; nothing committed. Waiting for the owner's review and authorization.
- 2026-11-01: the owner's independent review of `acc-plan73-review-1.zip`: CHANGES
  REQUIRED, in documentation and criteria only, the architecture approved in principle
  and no settled decision reopened. Taken into this plan: A1 to A4 (P7, P8, 73.23 to
  73.25, steps 2, 3 and 9), A5 to A8 (73.14, 73.16, 73.3 and 73.19, 73.21), A9 (the
  scenes, 73.29, step 10), R1 (D11, P2, 73.26), R2 (P4 to P6, 73.25). D11's display
  string is no longer kept between calls, and the motions run on it. Only this file
  changed; nothing implemented or committed. Implementation still waits for the
  owner's explicit authorization.
- 2026-11-01: the owner's independent review of `acc-plan73-review-2.zip`: APPROVED
  WITH RESERVATIONS, on how performance regressions are read. P8, 73.24 to 73.26, D11,
  P2 and step 9 corrected: no universal percentage decides a regression; every
  measurement is reported, absolute and relative, by alternating repeats; a missed
  absolute budget is always a FAIL; a reproducible degradation inside a budget is
  assessed from its evidence, and a significant or unresolved one goes to the owner. The
  fills' equivalence is functional. The scale ratio is a provisional diagnostic. A
  derived cache is not ruled out, only a second authoritative or mutable buffer. Only
  this file changed; nothing implemented or committed. Implementation still waits for
  the owner's explicit authorization.
- 2026-11-01: **the owner authorized the implementation** of steps 2 to 10, the
  owner's Human Evaluation and any real CLI turn staying the owner's, after one last
  correction: P8's reproducible and inconclusive results. No commit, push or closure
  before the owner's review of the evidence.
- 2026-11-01: step 2 done: the baseline on `1c36ceb`, all 21 timed tests and both guards
  passing (`doc/plan73-evaluation.md`).
- 2026-11-01: step 3 done: the replacement fill and the workload's check in the guards,
  built and compared before any application code changed; C1's functional equivalence
  passes, its performance reported under P8 (the same evaluation file).
- 2026-11-01: steps 4 to 7 done: the rule and the words, zone B, the ranges, the
  composer, with their tests, model and fuzz targets. The full view of 50,000 lines
  missed its 16 ms (p95 about 50 ms): P4's memo applied, measured again at about 1 ms.
  The mode is set where the view is built, at the end of every Update, not at its top,
  which holds it at the top of the next one and for the frame Ctrl+G opens.
- 2026-11-01: step 8 done: `ui-design.md`, `architecture.md`, `evaluation.md`, the
  overlay's rows; brief amendment 77 records the owner's reviews and authorization.
- 2026-11-01: steps 9 and 10 done: the suite, 75 mutants, 7 fuzz targets, `-race`, the
  timing step alone, C2, and W01 to W08 on a mock session; 29 of 30 criteria PASS, in
  `doc/plan73-evaluation.md`, with its reservations. Outstanding: step 11, the owner's
  Human Evaluation (73.30) and review. Nothing committed or pushed; the plan is not
  closed.
- 2026-11-01: the owner's independent review of `acc-plan73-review-3.zip`: APPROVED
  WITH RESERVATIONS, for the owner's Human Evaluation only. Its three corrections of
  wording are in: 73.2 and D2 (the bound), D10 and Components (the mode in `syncUI`),
  73.11 and 73.28 (the `↑ ↓` row); the Manual Human Evaluation lists its seven
  scenarios. Checking the bound against the code found two things it had not named,
  both recorded and neither a blocker: R-H, a cell wider than a row can make a chip
  of a line under 3 rows, and R-I, the bound counts cells, not bytes. Only documents and
  `longmessage_test.go` changed; no application code; focused checks only
  (`doc/plan73-evaluation.md`, The independent review). Outstanding: step 11, the
  owner's Human Evaluation (73.30) and approval. Nothing committed or pushed; the plan
  is not closed.
- 2026-11-01: **R-H corrected, on the owner's authorization**: the owner and the
  reviewer judged the false chip a functional defect and the fuzz property narrowed
  around it unacceptable. `messageScan` now counts a cell wider than `w` as `w`, with
  `most` = `3 × (w + 1)`, `widestCell` gone: one application file. `FuzzLongMessage`'s
  full equivalence is restored, R-H's inputs among its seeds; `TestLongMessageRenderer`,
  `TestLongMessageEquivalence` and `TestJoinedCells` are new; D2, 73.1, 73.2 and
  `ui-design.md` corrected. On the new build, the suite, `-race`, the whole timing step,
  fuzz, the rule's mutants, golden frames and W01 to W09; R-I measured again, unchanged by
  this and left documented (`doc/plan73-evaluation.md`, The correction of R-H).
  Outstanding: the independent review of `acc-plan73-review-4.zip`, then the owner's
  decision on the Human Evaluation (73.30). Nothing committed or pushed; the plan is not
  closed.
- 2026-11-01: the reviewer approved the correction, and the owner authorized the final
  end-to-end verification: T1 to T4 through the real clipboard, paste and mouse on mock
  sessions, and one real turn each of Claude Code (Haiku) and Codex through ACC. 33 of 34
  scenario runs PASS, the 60-line message reaching both CLIs byte for byte; no application
  code changed. One open defect, F1, from plan 32: a false "nothing to paste" toast during
  every paste over 5 KiB, a fix proposed and not made (`doc/plan73-evaluation.md`, The
  final end-to-end verification). Outstanding: the independent review of
  `acc-plan73-final-verification.zip`, the owner's decision on F1 and on 73.30, then
  closure. Nothing committed or pushed; the plan is not closed.
- 2026-11-01: the owner's independent review of that package approved the core and asked
  for F1's correction, zone B's chip lit by a selection if small and safe, and a
  reverification (The final review's corrections, above). The tests first, recorded
  failing on the previous code; then the fix, four application files, no real CLI turn.
  Its 14 criteria PASS: on mock sessions in Windows Terminal, 62 judged runs: 45 PASS on the
  corrected build and 17 BEFORE runs on the previous one by the same plans; 1,245 tests,
  `-race`, 23 mutants, the timing step alone (`doc/plan73-evaluation.md`, The correction of
  F1). New fingerprint `d7aa8b45d8276c2a`. Outstanding: the independent review of
  `acc-plan73-f1-review.zip`, the owner's decision on 73.30, then closure. Nothing committed
  or pushed; the plan is not closed.
- 2026-11-02: the independent review of `acc-plan73-f1-review.zip` is APPROVED, and the owner
  approved the Human Evaluation on the evidence already provided, with no manual session:
  73.30 `PASS — Owner approved`. The plan is closed (`doc/plan73-evaluation.md`, Approval and
  closure).
