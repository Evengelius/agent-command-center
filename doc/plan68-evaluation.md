# Plan 68 evaluation (2026-10-29)

The check of plan 68, the keyboard reader's races (brief amendment 70):
the Windows reader of the vendored ultraviolet sent a console read's bytes
and wrote the next read into the same array before the receiver had copied
them, so that a key could be lost and the next doubled (plan 67's lost
Ctrl+F), and its two goroutines shared the grapheme state. Patch 4 sends
each read as a copy, patch 5 gives the sender a grapheme state of its own.
The plan's independent design review, sent by the owner, approved it with
both fixes and three acceptance details, which the plan took in; the
owner: "I approve the implementation."

It covers the 9 criteria of [`plan.md`, Plan 68](../plan.md), written
before the code. Nothing is committed or pushed: the owner reviews the
results and the diff first.

How it was checked:
- **The console test** (`third_party/ultraviolet/patch_windows_test.go`,
  `TestConsoleReader`, seven scenarios), written first and run on the
  reader as it was, with patch 4 alone and with both: the reader's real
  console path, in a child process with a console of its own and no
  window, each scenario's events compared whole; plain and under `-race`.
- **The suite's check** that the copy's tests ran (`internal/licenses`,
  `copyTests`).
- **Each change broken on purpose** through `go test -overlay`, the
  repository never written, its code hashed before and after: the
  patches reverted in whole and in part, a hung child, the console test
  left out and skipped (`scratchpad\plan68\mkmutants68.mjs`,
  `mutrun68.sh`).
- **The module's own tests** on a scratch copy with the patched files.
- **Stability**: 50 runs of the console test in a row, and 20 under
  `-race`.
- **Windows Terminal**: plan 67's runs on the fixed build, and a burst of
  300 letters on the builds before and after, TEA_TRACE on.
- **Build, vet (Windows and Linux), gofmt, the whole suite**, and the
  timing step alone, on mains power.

## The evaluation

```text
Evaluation - Plan 68, the keyboard reader's races - 9 criteria: 9 PASS
PASS 68.1 No key lost: TestConsoleReader/loss gives press leftctrl, press ctrl+f, release ctrl+f, release leftctrl, in that order; /burst the 200 presses a to z in order, none lost or doubled. On the reader as it was (the test written first and run there): the loss test's events release ctrl+f, unknown "\x1b[17;29;0;0;32;1", release ctrl+f, release leftctrl, both presses lost; the burst's every fifth letter lost and the next doubled (d f f, i k k, n p p)
PASS 68.2 The characters and their order: each VT form (in one read, split between two reads, cut inside a sequence, as two surrogate records) gives press x, press 🙂, press y, the emoji once; VT input mode off, press a, press é, press 🙂, press b; the same lists on the reader as it was and with patch 4 alone (Details, Each version)
PASS 68.3 No race: go test -race -count=1 -v . in third_party/ultraviolet, the console test's seven scenarios and the module's other tests, 0 race reports. Each version's evidence kept: as it was, 12 reports, race 1 in all seven scenarios and race 2 in the four that reach the grapheme state, their stacks naming terminal_reader.go:201 against terminal_reader_windows.go:106 (or 99), and the sender's flush (line 203) or store (line 111) against encodeGraphemeBufs and storeGraphemeRune (terminal_reader.go:393 and 439); patch 4 alone, 5 reports, race 2 alone
PASS 68.4 The harness: no window, the child's GetConsoleWindow 0 in every scenario, its input its own; a hung child through -overlay failed its test with the child's output, at 4.6 s by the child's own timeout (H1) and, that timeout off, killed by the parent at its 5 s deadline, "the child ran past its 5s deadline and was killed" (H2); no test process or console host left after the mutants and the stability runs (tasklist); 50 runs in a row with no failure (265 s), 20 under -race (252 s, 0 reports)
PASS 68.5 The tests run: internal/licenses' TestThirdPartyTable requires TestPatchedWin32Input, TestPatchedDriftRows and TestConsoleReader to report PASS and none skipped (copyTests), ok in go test ./...; through -overlay, the console test left out by a build constraint fails it ("TestConsoleReader did not run and pass", L1), the test skipped fails it ("the patched copy skipped a test", L2), and with copyTests' check removed the exclusion passes unseen (L3, the control)
PASS 68.6 Nothing else: the copy's diff is patches 4 and 5 in terminal_reader_windows.go, each change marked PATCHED (agent-command-center, PATCHES.md, patch 4 or patch 5), the new patch_windows_test.go and PATCHES.md; outside the copy, internal/licenses/third_party_test.go's check alone, and the documents; every existing test passing unchanged (patch_test.go's TestPatchedWin32Input and TestPatchedDriftRows, TestWin32InputKeys, TestWideRowsKept, go test ./... ok); the module's own tests on a scratch copy with the patched files, plain and under -race: ok
PASS 68.7 Windows Terminal: ACC's profile, Windows Terminal 1.23.20211.0, maximised and at 120x40, a mock, TEA_TRACE on, the fixed build: plan 67's runs A, B and C, each step as plan 67 recorded it ("@\#|{}[]~êà" in the draft, Ctrl with the left Alt typing nothing, "@" in the draft window, the bar, Source Control's message box, the project field, the new branch and the branches popup's search, "@F" there, Alt+B closing and opening it); the burst, 300 letters typed 1 ms apart: the draft exactly the 300 at both sizes, all 300 presses delivered (the traces), no read left partly processed but a quit's last. The build before, the same burst: maximised, 298 letters, one wrong ("hijklq" for "hijklm"), and two records' text typed into the draft ("[88;45;120;0;32;1_", "[68;0;100;1;32;1_"); at 120x40, one letter wrong ("bcdefl" for "bcdefg"); all 300 presses delivered there too
PASS 68.8 The checks: go build ./... ok; go vet ./... ok (Windows and Linux); gofmt -l . empty; go test ./... ok (16 packages) after the last code edit; 9 mutants through -overlay, the repository never written (its hash the same before and after): 8 caught (P4a plain; P5a, P5b and P5c under -race; H1, H2, L1, L2), L3 the control, surviving as designed; the timing step alone, on mains power, at 19:56:52: its 21 tests within their targets
PASS 68.9 The documents: PATCHES.md (patches 4 and 5: why, the changes, the checks), THIRD_PARTY.md's patched copy, reliability.md (Not verified: whether the Windows console reader delivers every key), architecture.md (The keyboard reader's races), evaluation.md's race detector section (the copy's own -race run), brief amendment 70 with the review and the approval, AGENTS.md's status at the approval (TestAgentsMDWithinCodexLimit ok)
Not verified: a burst from a physical keyboard (the 300 letters were injected with keybd_event, as plan 67's presses were; the owner's keyboard check of plan 67 covered its fields, not a burst); a Codex session (the reader is the dashboard's own input, the same for both CLIs)
Verdict: DONE
Likely cause: -
Proposed next step: the owner's review of this report and of the diff, then the commits on the owner's approval - waiting for approval
```

**The review the owner shared (2026-10-29).** The owner shared, with no
words of their own, the plan's independent implementation review:
"Verdict: APPROVED — 9.8/10". It accepts the three versions' comparison,
the seven scenarios, the stability and race-detector runs, the module's
own tests, the check that the copy's tests run, the mutants with their
surviving control, the suite and standard checks, the Windows Terminal
burst at both sizes and the timing step; it finds the 20 ms hold
justified, and the missing physical burst no blocker ("Controlled input
injection is appropriate for reproducing this concurrency defect."). It
asked one clarification of wording: no statement that the characters and
their order are the same in every version, since the original
implementation loses and corrupts events; the conclusion is that patch 5
preserves correct Unicode and grapheme behaviour while it removes the
remaining race. 68.2's "the same lists on the reader as it was and with
patch 4 alone" speaks of the five character scenarios alone, which do not
reach the loss's window; Details, Each version, now says so, and the
block above stays as it was reported. The review closes, word for word:
"Plan 68 is approved for closure." and "You may proceed with the final
commits and push once those checks pass, following the established
project workflow."

## Details

### Each version, the evidence the review asked to keep

The console test's seven scenarios on the reader as it was, with patch 4
alone, and with patches 4 and 5 (`scratchpad\plan68\evidence\`: `as-is`,
`patch4`, `patch45`, each `.log` and `-race.log`). Race 1 is a read's
bytes (the receiver's copy, `terminal_reader.go:201`, against the sender's
write of the next read, `terminal_reader_windows.go:106`, or line 99 for
an emoji's two surrogate records); race 2 the grapheme state
(`encodeGraphemeBufs` and `storeGraphemeRune`, `terminal_reader.go:393`
and `:439`, against the sender's flush at line 203, or its store at line
111 with VT input mode off).

| Scenario | As it was | `-race` | Patch 4 | `-race` | Patches 4 and 5 | `-race` |
| - | - | - | - | - | - | - |
| loss | both presses lost: `release ctrl+f`, unknown `\x1b[17;29;0;0;32;1`, `release ctrl+f`, `release leftctrl` | race 1 | the four in order | none | the four in order | none |
| burst | every fifth letter lost and the next doubled (`d f f`, `i k k`, `n p p`) | race 1 | the 200 in order | none | the 200 in order | none |
| emoji in one read | `x 🙂 y` | races 1 and 2 | `x 🙂 y` | race 2 | `x 🙂 y` | none |
| emoji split between reads | `x 🙂 y` | races 1 and 2 | `x 🙂 y` | race 2 | `x 🙂 y` | none |
| emoji cut inside a sequence | `x 🙂 y` | races 1 and 2 | `x 🙂 y` | race 2 | `x 🙂 y` | none |
| emoji as surrogate records | `x 🙂 y` | race 1 | `x 🙂 y` | none | `x 🙂 y` | none |
| characters without VT input | `a é 🙂 b` | race 1, race 2 twice | `a é 🙂 b` | race 2 twice | `a é 🙂 b` | none |

12 race reports on the reader as it was, 5 with patch 4, none with both.
The reader as it was loses and corrupts events: both of Ctrl+F's presses
in the loss scenario, every held letter in the burst (68.1), and wrong
letters and records' text in the Windows Terminal burst (68.7). The five
character scenarios do not reach that window, and give the same events in
every version: patch 5 keeps the characters delivered and their order,
each emoji once and in its place, while it removes the grapheme race the
detector still reports with patch 4 alone (68.2, 68.3).

### What the build found and changed

- **The burst's hold, 20 ms.** The plan said 2 ms; `streamData` sleeps
  10 ms when it finds no input, so a 2 ms hold ends before it reads the
  next letter. At 20 ms the sender reads the next letter while the
  receiver is held, and on the reader as it was every held letter is lost
  and the next doubled. Recorded in plan.md's D4.
- **The cut sequence's escape timeout, 2 s.** An emoji cut inside a
  sequence across two reads waits for its end; with the default 50 ms a
  slow machine could end it first and report its first part as unknown.
  The test sets the reader's `EscTimeout` for that scenario alone.
- **Race 1 in every scenario.** The race detector reports it wherever a
  scenario has more than one read, not in the loss and the burst alone:
  nothing orders the receiver's copy and the sender's next write, so the
  detector needs no lost key to see it.
- **The race in Windows Terminal.** The build before lost and changed
  letters, and typed two records' text into the draft, while every read
  of its traces was consumed whole: the bytes the receiver copied in place
  of its own were still whole records. The draft is the check; a trace
  shows only that Windows Terminal delivered the 300 presses.
- **The copy's tests shown to run** (D6). `internal/licenses` ran the
  copy's tests and checked their exit code alone, so a test file left out
  by a build constraint would have passed unseen, as L3 shows; it now
  reads `-v`'s output and requires each test's PASS, none skipped.

### The mutants

Through `go test -overlay`, the repository never written (its code hashed
before and after, the same: de050ea5), the copy's console test or
`internal/licenses`' test run with each (`scratchpad\plan68\mkmutants68.mjs`,
`mutrun68.sh`; logs in `mut\logs\`). mutrun runs no `-race`, so these ran
by a script of their own.

| Mutant | Broken | Run | Caught by |
| - | - | - | - |
| P4a | patch 4 reverted: `buf.Bytes()` sent | plain | TestConsoleReader: the loss's and the burst's events |
| P5a | patch 5 reverted: the three grapheme calls on `d.eventScanner` | `-race` | TestConsoleReader: the children's race reports, the events still right |
| P5b | patch 5's end-of-batch flush on `d.eventScanner` | `-race` | the same |
| P5c | patch 5's store, VT input mode off, on `d.eventScanner` | `-race` | the same, in the VT-off scenario |
| H1 | the loss scenario's child hung, a 5 s deadline | plain | its test, failed with the child's output: "the child failed (exit status 2)", the child's own timeout |
| H2 | the same, the child's own timeout off | plain | "the child ran past its 5s deadline and was killed", with its output |
| L1 | the console test left out by `//go:build ignore`, in `internal/licenses`' run of the copy | plain | TestThirdPartyTable: "TestConsoleReader did not run and pass" |
| L2 | the console test skipped (`t.Skip`), the same way | plain | TestThirdPartyTable: "did not run and pass", "the patched copy skipped a test" |
| L3 | L1's exclusion with `copyTests`' check removed, as before plan 68 | plain | not caught, the control: the exclusion passes unseen without D6 |

No test process was left after them (tasklist).

### The checks

| Check | Result |
| - | - |
| build, vet (Windows, Linux), gofmt | ok, ok, ok, nothing printed |
| `go test ./...`, after the last code edit | 16 packages ok, 277 s; internal/licenses 14.5 s, the copy's three tests among it |
| the copy's tests, `go test -count=1 -v .` and with `-race` | TestPatchedWin32Input, TestPatchedDriftRows, TestConsoleReader: PASS; under `-race` the same, 0 reports |
| the module's own tests on a scratch copy with the patched files | ok, and under `-race` ok, 0 reports (`scratchpad\plan68\upstream-tests.mjs`) |
| the mutants | 9: 8 caught, L3 the control |
| stability | 50 runs in a row, 265 s; 20 under `-race`, 252 s, 0 reports; no failure |
| processes left | none: no `ultraviolet.test.exe`; the one console host started meanwhile belongs to the tool's own shell |
| Windows Terminal, the fixed build | the burst and runs A, B and C at both sizes, as 68.7 says |
| the files | UTF-8 without a BOM, LF only (read from the bytes) |
| the timing step, alone, on mains power | below |

### The timing step

The command of evaluation.md, the twenty-one tests, alone, at 19:56:52,
93 s: on mains power, battery 96 %; CPU load over the 2 s before, 28, 18,
18, 19 and 20 %; the windows open, Chrome, Claude, Edge, Outlook, Notepad,
Task Manager, Move Mouse, Settings, Microsoft Lightbox; Defender idle (0 s
of CPU). Plan 68 adds no timed test and changes nothing on a frame's path.

| Test | Measured | Target | Result |
| - | - | - | - |
| TestKeystrokeToFrame | 10,000-line draft p95 1.07 ms; in the draft window 1.10 ms | 16 ms | PASS |
| TestDraftKeystrokes | 50,000 lines, 8 saves, p95 2.57 ms | 16 ms | PASS |
| TestKeystrokeCost | reports only | | PASS |
| TestLongSession | a delta at the end, p95 2.58 ms | 16 ms | PASS |
| TestThinkingFrame | p95 2.29 ms | 16 ms | PASS |
| TestPasteToFrame | the worst 1.51 ms | 100 ms | PASS |
| TestStreamToFrame | p95 4.87 ms | 16 ms | PASS |
| TestSelectorToFrame | p95 1.01 to 1.67 ms | 16 ms | PASS |
| TestMotionToFrame | p95 1.01 to 1.53 ms; idle motion all dropped | 16 ms | PASS |
| TestDiffToFrame | p95 0.70 to 1.10 ms | 16 ms | PASS |
| TestMessageToFrame | p95 1.05 and 1.51 ms | 16 ms | PASS |
| TestOutputToFrame | p95 1.01 ms | 16 ms | PASS |
| TestToolsToFrame | p95 2.67 ms | 16 ms | PASS |
| TestBannerToFrame | p95 1.885 ms, max 3.12 ms, 11,355 bytes a frame | 2 ms | PASS |
| TestHomeKeysToFrame | p95 0.18 ms | 16 ms | PASS |
| TestHomePictureTime | the worst 214 ms | 260 ms | PASS |
| TestBranchesFrameCost | p95 1.70 ms | 16 ms | PASS |
| TestBlameFrameCost | p95 1.16 ms | 16 ms | PASS |
| TestGraphFrameCost | p95 1.02 ms | 16 ms | PASS |
| TestTurnReviewFrameCost | p95 1.50 ms | 16 ms | PASS |
| TestSearchToFrame | keys, two hours: p50 3.02, p95 5.00, max 6.33 ms; answers ten times longer: p50 7.37, p95 9.82, max 12.0 ms; deltas, the bar open: p50 2.51, p95 4.04, max 6.02 ms | 16 ms | PASS |

### Windows Terminal

Windows Terminal 1.23.20211.0, ACC's profile, maximised (208x59) and at
120x40 (`-mode 90x30` at the profile's 9 pt), the Belgian AZERTY layout, a
temporary home and app data, TEA_TRACE on; no Claude Code or Codex session
ran in a terminal before the runs.

- **The burst** (`scratchpad\term\p68runs.ps1`, `p68.ps1`'s burst step):
  the mock dashboard on a demo repository, 300 letters a to z typed into
  the draft with `keybd_event`, 1 ms from one press to the next (a spin
  wait; 0.62 to 0.80 s for the 300), read from the shot's editor rows and
  checked against the trace (`scratchpad\plan68\burstcheck.mjs`).

| Build | Size | The draft | The trace |
| - | - | - | - |
| before (`acc-p68-before.exe`, the code as 0316a30 has it) | 208x59 | 333 characters: 298 letters, one wrong at the 91st ("hijklq" for "hijklm"), and two records' text, `[88;45;120;0;32;1_` and `[68;0;100;1;32;1_` | 94 reads, 300 presses delivered, none left partly processed |
| before | 120x40 | 300 letters, one wrong at the 267th ("bcdefl" for "bcdefg") | 92 reads, 300 delivered, none left |
| after (`acc-p68.exe`) | 208x59 | exactly the 300 | 117 reads, 300 delivered, none left |
| after | 120x40 | exactly the 300 | 80 reads, 300 delivered, none left |

- **Plan 67's runs A, B and C on the fixed build** (`p67runs.ps1 -exeName
  acc-p68.exe -suffix 68`, `p67runc.ps1 -tag altgrC68`): at both sizes,
  each step as plan 67 recorded it, the traces with no read left partly
  processed but run B's last before its quit.

The shots are in `scratchpad\term\out-p68\` (the burst: `burst-before-max`,
`burst-before-90x30`, `burst-after-max`, `burst-after-90x30`, each `k2.png`
and its text) and `out-p67\` (`altgrA68-*`, `altgrB68-*`, `altgrC68-*`),
the traces beside them. The ones to look at first: `burst-before-max\k2`
(the record text in the draft) beside `burst-after-max\k2`.
