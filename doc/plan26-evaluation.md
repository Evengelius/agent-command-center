# Plan 26 evaluation (2026-10-22)

The check of plan 26, zone A without notes (brief amendment 25: "Can we
just get rid of this part/section completely and forever?"). It covers the
criteria written in [`plan.md`, Plan 26](../plan.md) before the code.
Nothing is committed until the owner approves.

How it was checked:
- Deterministic tests in `internal/render` (TestNoNotes, TestZoneAStructure,
  TestDetailsUnrecognised, TestDetailsErrors, TestTelemetryParsing,
  TestTelemetryNA, TestTelemetryWholeItems) and `internal/app`
  (TestSyntheticScenarios, the frame tests), on the mock files and the
  probe recordings.
- Every golden frame regenerated (`-update`) and reviewed by script
  (`scratchpad\plan24\framecheck.cjs`): for each of the 51 changed frames,
  the new frame is the old one with zone A's rule and notes rows removed
  and the same number of blank rows put before the pane's rule, every B and
  C row unchanged; no "ⓘ" left in `testdata/frames`.
- Build, vet, gofmt, the whole suite, the race detector on `internal/render`,
  the timing step alone.

```text
Evaluation - Plan 26, zone A without notes - 6 criteria: 6 PASS, 0 FAIL
PASS 26.1 TestNoNotes: the P3b mock with two errors, retries 3, three unrecognised lines, a derived context and undocumented quota values, and the Codex mock, at 120x40 and 208x59: zone A's rows hold none of "ⓘ", "errors ", "retries ", "unrecognised", "quota updated", "derived", "undocumented field", "cost at API rates", "overage", "plan plus", "credits", "values from the CLI's events"; the full form's last row is "  Resets in …"; the TELEMETRY heading row ends with "Enter details"; an empty state's full form draws no "ⓘ"; the details view holds "value · source · received · † undocumented field", "unrecognised events 3", "retries 3", a "%†" value, "(derived)" and the line "system/x"
PASS 26.2 TestZoneAStructure: at 120x40 and 208x59 two "Resets in" rows, no "ⓘ", the pane's last row the last window's countdown, the tree 6 rows at least and 3 more than plan 24's (9 at 120x40, 24 at 208x59); at 100x24 the tight form on one row a window and 3 tree rows; SESSION USAGE folded moves TELEMETRY 3 rows down
PASS 26.3 TestDetailsUnrecognised and TestNoNotes: the TELEMETRY heading reads "value · source · received · † undocumented field"; TestDetailsErrors, the "†" and "(derived)" marks of the values unchanged (TestTelemetryParsing on P1, P3b, P6b, P7)
PASS 26.4 51 golden frames regenerated: framecheck.cjs finds 35 exactly as expected (the full form's 3 rows gone under the windows, 3 blank rows before the pane, every B and C row unchanged); the 16 others read in their diffs: 10 with a window over the dashboard (draft, explorer, file, file-finder, file-search at 120x40 and 200x60) show the pane 3 rows lower and nothing else, quota-120x40's pane goes from the compact to the full form (the blank rows between its sections and three windows) with 8 tree rows, quota-80x24 and tree-80x24 lose their notes row and gain a tree row, the three details frames gain the heading's legend; grep "ⓘ" over testdata/frames: 0 files
PASS 26.5 go build ./... ok; go vet ./... clean; gofmt -l . empty; go test ./... -count=1 14 packages ok (internal/app 39.1 s); go test -race -count=1 on internal/render with CGO_ENABLED=1 and the winlibs gcc: ok, no race (103.3 s); the timing step alone 7 of 7 within every target (16 ms, 100 ms for the paste), run at 16:38 right after the race run and again alone at 16:39 (plan24\p26-checks.log, p26-timing2.log): TestKeystrokeToFrame p95 1.00 and 1.07 ms (2.01 and 2.08 ms in the window), TestDraftKeystrokes p95 3.14 and 3.64 ms, TestLongSession delta-to-frame p95 14.06 and 13.08 ms, TestPasteToFrame max 3.94 and 2.90 ms, TestStreamToFrame p95 8.92 and 9.01 ms, TestThinkingFrame p95 9.17 and 11.31 ms, about twice the figures of 16:12 (plan 25's run); so plan 25's tree (HEAD, by git archive) and this tree were timed in turn at 16:40 and 16:41, alone: HEAD gave TestLongSession p95 10.59 ms, TestStreamToFrame 7.73 ms, TestThinkingFrame 7.55 ms, TestKeystrokeCost 519 µs a key, this tree 11.17 ms, 8.74 ms, 9.01 ms and 544 µs: the same slowdown on both trees, the machine's at that hour (CPU 22% busy with Chrome, the VPN agent and Task Manager; the paste test, which the change does not touch, doubled too), not the change's, which draws fewer rows
PASS 26.6 ui-design.md (the two Layouts frames redrawn by script and their text, the pane's forms, "No notes", the telemetry table, Freshness, the details view, the states and symbols tables, Animations), events-and-state.md and reliability.md (where the counts show), AGENTS.md (status), brief.md (amendment 25), plan.md (D1 to D5, 26.1 to 26.6, the standing decision, progress); grep "ⓘ" over doc finds brief.md, plan24-evaluation.md and reference-design.md only (the amendments' words, plan 24's record, the owner's image); every changed text file read byte by byte: no BOM, no CR; git status lists the intended files only; nothing committed
```

**What the owner sees now.** Zone A ends with the last quota window's
rows: the tree above it has three rows more at the full form (one at the
compact and tight forms). The counts of errors, retries and unrecognised
lines, the quota's age, the derived and undocumented legends and the cost
wording are in the details view only (Enter on TELEMETRY), which keeps
every value with its source and age and lists the unrecognised lines
(plan 25). The "†" after an undocumented value stays, in zone A and in the
details view, where the TELEMETRY heading now reads "value · source ·
received · † undocumented field".

**Evidence outside the repository**, in the session's scratchpad
(`C:\Users\amolimom\AppData\Local\Temp\claude\C--javadev-projets-agent-command-center\dcc3e64d-0d35-4470-ab9a-d258f8675bae\scratchpad\plan24\`):
`p26-checks.log` (the suite, the race run, the timing step),
`framecheck.cjs` (the frames' review), `docframes2.cjs` (the design
document's frames redrawn), the patch files `p26-*.patch`.

**The commit and the push (2026-10-22), after the owner's approval.** The
owner approved plan 26 on 2026-10-22; commit a9a5145 (61 files: 60
modified, 51 of them golden frames, 1 added) and the approval commit were
pushed to `origin`; `origin/main` equals `main`, the working tree is
clean, and the author is `Evengelius <molimoabel@gmail.com>`, with no
attribution line: PASS. The block above stays as it was reported.
