# Plan 7 evaluation (2026-10-17)

The final check of plan 7, the open list after plan 6: each item of
[`plan.md`, Plan 7](../plan.md#plan-7---the-open-list-after-plan-6) against
the yes/no criteria written in the progress list before its work started,
and the plan's rules (Windows 11 and Windows Terminal only, the owner's quota
for items 4 and 5 only, no new dependency). The evidence is `plan.md`'s
record of each item; the block below is the summary reported to the owner.
The results are not to be rewritten.

```text
Evaluation - Plan 7, the open list after plan 6 - 7 criteria: 7 PASS, 0 FAIL
PASS 1  Long agent labels in zone C (item 1): 4 of 4; TestAgentLabelColumn (labels of 1 to 40 cells at 120, 160 and 200 columns, one model cell per frame); 5 Codex golden frames changed in C's rows only, reviewed; the Claude frames unchanged
PASS 2  The preview's header from 160 to 220 columns (item 2): 3 of 3; TestPreviewHeaderFallback at every width, for a requested, a reported and no model: the first form that fits, the model never cut; golden frames unchanged
PASS 3  The error line of a failed Codex command (item 3): closed as a limit of Codex's interface (0.163.0 generated schema, commandExecution.aggregatedOutput), recorded in doc/ui-design.md; no code change
PASS 4  A spawn request carrying a model or an effort (item 4): 2 of 2; probe P7c (Codex 0.163.0, 15 s, configuration check PASS): no spawnAgent item at all, the model and effort on the child's subAgentActivity; recorded, sanitized, in 12 fixture recordings; TestSubagentModel "gpt-6-luna", TestAgentEffort "low"
PASS 5  Plan 6's items live, both CLIs (item 5): 5 of 5, one amended; 21 of 21 harness checks per CLI, maximised (156 columns) and 120 by 30; Codex "✗ commandExecution · exit 1 ·     git push <groupname>" (exit 1 by hand), C aligned with "gpt-6-sol high", sources in preview and inspection; Claude "✗ Bash · Exit code 128" (128 by hand), "effort N/A (not reported)", no effort column; "↓ 1 new" on both; Ctrl+Q leaves no process; configuration check PASS twice
PASS 6  go build ./... and go vet ./... (also -tags live, -tags timing) exit 0; gofmt -l . prints nothing; go test ./... 231 top-level tests PASS in 14 packages (229 at 0814d67, +2), 0 FAIL; the changed files have no CR and no BOM
PASS 7  Rules: quota spent on P7c and item 5's two sessions only, each in a temporary project outside the repository with the configuration check before and after; go.mod and go.sum unchanged since 0814d67, 4 direct modules; Windows 11 only; none of amendment 4's excluded features
Mutation checks: 4 mutants killed (item 1: the column held at 14, no 12-cell floor for the activity; item 2: the old all-or-nothing header, the shortest form first), 0 survived; none for items 3 to 5.
Not verified: a spawn request carrying a model or an effort (Codex 0.163.0 sent none); the exit code against Codex's own event (compared with the same command run by hand); the timing step (no input or stream path changed). Each with its reason below.
Verdict: DONE
```

## Not verified

What no check of this plan settles, and why. None of it is reported as PASS.

- **A spawn request carrying a model or an effort** (item 4). Asked for a
  smaller model and the effort "low", Codex 0.163.0 sent no `spawnAgent`
  item at all, only `wait` items with both fields null; the values arrived
  on the child's own `subAgentActivity`, which the mapper reads as reported.
  The fixture built from the generated schema stays the only spawn-request
  test.
- **The exit code against Codex's own event** (item 5, criterion 1). The
  application keeps no event log, so the harness cannot read the event.
  The same command run by hand the way Codex ran it, in the same folder,
  exits 1 and ends with the same line B shows; git alone exits 128, so 1 is
  PowerShell's code, which Codex reports.
- **The keystroke timing step.** Plan 7 changed how zone C and the preview
  are laid out, not the input, the stream or the frame loop; the step was
  not run.

## Failures on record

None of the plan's criteria failed. Item 5's harness, a scratchpad script
written for this plan, had six errors of its own before the passing runs,
none in the application; each is recorded in `plan.md`, item 5:

| Run | Failure | What followed |
| --- | --- | --- |
| Codex, 22:29 | The agent-row pattern missed C's padding cell: 0 rows read, 7 FAIL (rows, efforts, preview, inspection) | Pattern fixed; the rows read at both sizes |
| Codex, 22:29 | B followed the 40-line answer, so the failed row was off screen: 2 FAIL | `FindInB`: scroll B up until the row shows, then follow again |
| Codex, 22:29 | The whole answer arrived at once before B overflowed: no badge in 15 holds, NOT VERIFIED | A follow-up turn while B is held (criterion 3, amended below) |
| Codex, 22:34 | An F6 after Esc moved the focus to A, so Enter at 120 columns opened no inspection: 1 FAIL | The F6 removed; Esc already returns the focus to B (`internal/app/keys.go`) |
| Claude, 22:36 | The model pattern did not know the short "haiku-5-5" form C draws: 7 FAIL | Pattern widened to Claude's model families |
| Claude, 22:37 | The preview check required the model's source, which the criterion does not ask for and item 2 drops first for a long label; then one edit left a non-ASCII character that Windows PowerShell 5.1 misread: 1 FAIL twice | The check reads the model, its source optional; the script ASCII again |

## Criteria amended

- Item 5, criterion 3: "↓ N new" was checked on a second turn in the same
  session, a 40-line follow-up sent while B was held, not on the first turn.
  In the first Codex run the answer arrived at once before B overflowed, so
  the harness could not hold B while it streamed. Criteria 1 and 4 ("one
  turn") were checked on the first turn as written.

## The list this plan closed

Four entries of the list the owner was given before this plan had already
been closed by plan 5's follow-up ([`plan5-evaluation.md`](plan5-evaluation.md),
After the commit) and were wrongly listed as open: a selection from a
conversation entry into a failure card (F1), a NUL in a paste (F2), 256 and
16 colours (L10, L11), and Codex's built-in thread limit. They were not
redone. Excluded by the owner and left open: Codex's `acceptForSession`
offer; the live API-key, no-five-hour-window and refusal cases; Linux and
macOS; amendment 4's excluded features.
