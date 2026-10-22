# Plan 28 evaluation (2026-10-22)

The check of plan 28, the top right after the owner's image (brief
amendment 28, `references/07-top-right.webp`: "Now a small fix,  on the
top right, I want that design on my image"). It covers the criteria
written in [`plan.md`, Plan 28](../plan.md) before the code. Nothing is
committed: the owner reviews the screenshot first.

How it was checked:
- Deterministic tests: `internal/render` (`topright_test.go`: TestTopRight
  new; `version_test.go` adapted) and `internal/app` (`approvals_test.go`:
  the bar helper and the Manual expectations adapted).
- Every golden frame regenerated and reviewed by script
  (`scratchpad\plan24\framecheck28.cjs`): 27 changed, each differing from
  HEAD on its project bar row only; no bar reads "approvals Manual" in
  force.
- Windows Terminal, the executable built from this tree
  (`scratchpad\term\acc-p28.exe`), the plan 24 harness as
  `scratchpad\term\p28demo.ps1`: the quota mock maximised at 9 pt
  (208x59), a temporary home and no agent CLI on the PATH, in mock mode;
  the screenshot in `scratchpad\term\out-p28\quota-max\top-right.png`
  with its UI Automation rows. The owner's Windows Terminal window stayed
  as it was, as for plan 27's second pass.
- Build, vet, gofmt, the whole suite, the race detector on the render and
  app packages, then the timing step alone (`scratchpad\plan24\p28-checks.log`).

```text
Evaluation - Plan 28, the top right after the owner's image - 6 criteria: 6 PASS, 0 FAIL
PASS 28.1 TestTopRight on P3b at its Write as a Claude Code 2.1.303 session with a 200-cell project path, at 120x40 and 208x59: the project bar ends with "Claude Code 2.1.303 " and holds no "approvals", 120 and 208 cells wide, starting " ▣ acc-demo"; with Auto in force it ends with "Claude Code 2.1.303 · approvals Auto ", "Auto" in stAttentionBold; with Auto asked for "Claude Code 2.1.303 · approvals Manual → Auto … "; with Manual asked for from Auto "Claude Code 2.1.303 · approvals Auto → Manual … "; the middle's path shows 17 cells more of itself without the mode than with it, at both sizes; TestProjectBarNewerThanTested: "Claude Code 2.1.303 newer than tested " ends the bar, the mark in stAttention
PASS 28.2 TestF4ClaudeSwitch: the bar ends with "2.1.300" and holds no "approvals" in Manual, "approvals Manual → Auto …" asked, "approvals Auto" in force, "approvals Auto → Manual …" asked back, "approvals Auto" after the refusal; TestF4CodexNextTurn at 100, 120 and 156 columns: no "approvals" in Manual, the bar ending with "policy on-request" from 120 columns, "approvals Manual → Auto (next turn)" asked, "approvals Auto" in force; TestKeysPage: "F4 Manual/Auto" in the footer, "approvals Manual / Auto" on the key page
PASS 28.3 27 golden frames regenerated (the 25 whose bar read "approvals Manual" and 2 narrow ones whose window hides all but the bar's end), each differing from HEAD on row 2 only, no bar holding "approvals"; the frames of a change asked for and of Auto in force unchanged; 0 bars read "approvals Manual" in force; TestDocFrames redrew ui-design.md's three Layouts frames (80x24 added)
PASS 28.4 go build ./..., go vet ./... and gofmt -l . clean; go test ./... 14 packages ok; the race run on internal/render and internal/app ok, no race; the timing step alone 7 of 7: TestKeystrokeToFrame p95 1.03 ms (1.51 ms in the draft window), TestDraftKeystrokes 1.96 ms, TestKeystrokeCost 219 µs and 553 µs a key, TestLongSession delta-to-frame p95 4.54 ms after 26,209 events, TestPasteToFrame p50 1.06 ms, TestStreamToFrame p95 3.68 ms, TestThinkingFrame p95 3.52 ms, every target p95 ≤ 16 ms
PASS 28.5 Windows Terminal, quota-max top-right (14.9 s, "1 DONE" found): the title row ends "◆ MOCK — not a live session", the project bar "Mock data 2.1.300" with nothing after it, zone B's edge "─ Idle ─╮"; the conformance table below, 4 rows PASS, the mock's words standing for the live ones; the owner reviews it before any commit
PASS 28.6 ui-design.md (Project bar; the three Layouts frames), reference-design.md (the image's row and section), brief.md (amendment 28), AGENTS.md (status), plan.md (D1 to D3, 28.1 to 28.6 with 28.3 corrected as the frames were reviewed, progress); every changed text file read byte by byte: no BOM, no CR; git status lists the intended files only
Not verified: the owner's review of the screenshot; the live words "● LIVE" and "Claude Code 2.1.303" (unchanged code, seen in the owner's own run of 19:06), the screenshot being a mock run
Verdict: DONE
```

**Conformance with the image** (`references/07-top-right.webp`), read
against `out-p28\quota-max\top-right.png`:

| The image | The dashboard | Result |
| - | - | - |
| "● LIVE", a green dot and the word in green, at the title row's right end | the live session's title row, unchanged; the mock run reads "◆ MOCK — not a live session" there by design | PASS, the mock's words |
| "Claude Code 2.1.303" in white under it, nothing after it | the provider's name and version alone, one cell from the edge: "Mock data 2.1.300" in the mock run, "Claude Code 2.1.303" live | PASS |
| No approvals mode on the row | none while approvals are Manual with no change asked; "approvals Auto" in the attention colour while Auto is in force, a change asked for after an arrow until it is in force | PASS |
| Zone B's top edge in lavender with "Idle" in grey let into it | unchanged: the session's state, dynamic ("Idle", "Running", "Interrupted") | PASS |
| The row's left part and middle | not in the image; the middle's path shows up to 17 cells more of itself | not in the image |

**One choice to state.** The image shows a Manual session, where the mode
is gone from the row. Auto approvals stay visible there in the attention
colour (brief amendment 8 made them visible on purpose), and a change
asked for shows with its arrow until it is in force; the owner can ask for
them to go elsewhere.

**Evidence outside the repository**, in the session's scratchpad:
`term\out-p28\quota-max\` (the screenshot, its rows and `log.txt`),
`term\p28demo.ps1`, `term\acc-p28.exe`, `plan24\p28-checks.log`,
`plan24\framecheck28.cjs`, `plan24\docframes\` (the three redrawn frames),
`plan24\uidesign28.cjs`, the patch files `p28-open.patch`,
`p28-crit3.patch`, `agents28.patch`.
