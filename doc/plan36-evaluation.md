# Plan 36 evaluation (2026-10-24)

The check of plan 36, the selector's cursor and a pointer that has not
moved (brief amendment 37: "ok lets fix that", after plan 35's report that
Alt+P had reopened the model selector with its cursor on the Ultracode
row, under a pointer left resting there). It covers the 7 criteria
written in [`plan.md`, Plan 36](../plan.md) before the code; criterion 1
was corrected before the code too, Enter on the model in force asking for
that model again (`chooseModel`), not for nothing. Nothing is committed:
the owner reviews the result first.

How it was checked:
- The probe first, P36a (plan.md, What the probe found): Windows Terminal
  1.23.20211.0 at 120x40 on `testdata/mock/claude-models.jsonl`, plan 35's
  build (`scratchpad\term\acc-p35.exe`), with Bubble Tea's `TEA_TRACE`
  logging every sequence in and out (`scratchpad\term\p35demo.ps1
  -trace`; outputs in `scratchpad\term\out-p36`).
- Deterministic tests, none starting a CLI: `internal/app`
  (TestSelectorPointerAtRest, TestSelectorPointerLevels, and
  TestSelectorMouse unchanged). Each was run against code broken on
  purpose, and failed: every report a hover, as before
  (pointer_test.go:28, the cursor on row 7, the Ultracode row;
  pointer_test.go:76); only the first report after opening ignored
  (the same two failures: the reports wander a cell); the pill's click
  recording nothing (models_test.go:137, "hover: cursor 2, want 3").
- Windows Terminal again, the build of the final tree
  (`scratchpad\term\acc-p36.exe`, SHA-256 `792cf694…`) against plan 35's,
  on the same plans: the probe's three, then the pointer put down by
  `SetCursorPos` (as software moves it) before each Alt+P, 6 and 16
  times, and at the end moved onto Sonnet 5.5 in ten `SendInput` steps,
  as a mouse moves; every reopen's reports read from the trace.
- Build, vet, gofmt, the whole suite, the race detector on app, the
  timing step alone; no independent review, the criteria being settled
  by the tests, the three mutation checks and the traces.
- The owner's dashboard, `go run ./cmd/agent-command-center` in a
  Windows Terminal tab with Claude Code under it (started 02:11), ran
  during the WT runs; it was not closed. The standing note asks to close
  such sessions before tests, for their configuration writes and quota
  during live runs; these runs used the mock, started no CLI, and send
  input to their own window only after checking it is in front, and
  closing it would have ended the owner's session. It drew about 7% of
  one core, its Claude Code about 1%, during the timing step.

```text
Evaluation - Plan 36, the selector's cursor and a pointer that has not moved - 7 criteria: 7 PASS
PASS 36.1 TestSelectorPointerAtRest: Alt+P, then reports at the text toggle's cell, a cell to its right and back: the cursor on row 2, Fable 5.1, the model in force; Enter asks for {Model: fable}, Ultracode nil, the one change
PASS 36.2 TestSelectorPointerAtRest: then a report over Sonnet 5.5 puts the cursor on row 3, and one back over the toggle on row 7, Ultracode
PASS 36.3 TestSelectorPointerAtRest: ↑ to row 6, Effort, then a report on the toggle's row again: the cursor stays on row 6
PASS 36.4 TestSelectorMouse unchanged: opened by a click on the pill, the first report over Sonnet 5.5 puts the cursor on row 3; it fails when the click records nothing
PASS 36.5 TestSelectorPointerLevels: the effort list open, a report outside the selector then one over Low: level 0; ↓ to level 1, then a report on Low again: level 1
PASS 36.6 build, vet, gofmt clean; go test ./... ok (the 15 packages with tests); -race on app ok; no golden frame changed; TestSelectorToFrame p95 1.02 to 1.53 ms (target 16 ms), the other seven timing tests within target
PASS 36.7 WT 120x40: 16 cycles of the pointer put down on the toggle, then Alt+P: plan 35's build had 1 reopen with reports of the resting pointer (106,30 107,30 106,30, the probe's), and its cursor left the models; this build had 2 (7,4; and 106,30 107,30 106,30), both on Fable 5.1, 0 of 18 opens off the model in force; the probe's three variants, 12 of 12 reopens on Fable 5.1 (no report came in those runs); the ten-step move onto Sonnet 5.5 put the cursor there on both builds
```

Not verified:
- A physical mouse: in the probe, moves made as a mouse makes them
  (`SendInput`) brought no report at a reopen; only a pointer put down by
  software did. Which other software does so (remote desktops, mouse
  utilities) was not tried.
- A pointer that jumps onto a row in one report right after Alt+P, as
  software moves it, does not take the cursor until it moves again
  (the build's first check, a one-step `Glide` onto Sonnet 5.5, left the
  cursor on Fable 5.1); a mouse reports every cell it crosses, the ten-step
  move took the cursor. By design (D1), not a criterion.

Seen outside the plan: in one of 72 Esc presses over the WT runs (the
fixed build's 6-cycle run, its sixth close), the selector stayed open
until the next key came, about a second later, and both were handled in
one frame. The trace shows the Esc read and parsed at once ("processed 17
bytes from buffer"), its key-down and key-up in two reads, then no frame
until Alt+P. The cause was not found; plan 36 changes only the handling
of mouse motion and clicks. Reported to the owner; nothing changed.

**The commit and the push (2026-10-24), after the owner's approval.** The
owner approved plan 36 on 2026-10-24; commit c0cebc4 (10 files: 8
modified, 2 added) and the approval commit were pushed to `origin`;
`origin/main` equals `main`, the working tree is clean, and the author is
`Evengelius <molimoabel@gmail.com>`, with no attribution line: PASS. The
blocks above stay as they were reported.
