# Evaluation

How an agent checks its own work after each instruction, before saying it is
done.

Every check is a yes/no check with an exact expected value: a count, a text, an
exit code, a source. There is no score: a criterion passes or fails, and the
work is done only when every criterion passes. What only the owner can judge
(how the screen looks and feels) is listed as not verified, never assumed.

## The loop

1. **Before working**, write down how the result will be checked: one line per
   criterion, each with its exact expected value, taken from the instruction,
   `doc/brief.md` and `AGENTS.md`. Put the criteria of a plan in `plan.md`.
2. **Work.**
3. **Check** every criterion and keep its evidence: a link with a commit, a
   file and line, a command output, an exit code, a diff. What must last goes
   in the plan's evaluation file (Evidence and temporary files, below).
4. **Summarise** in the format below.
5. **If something failed**, give the likely cause and a proposed next step. Do
   not apply it without the owner's approval; a plan's Review cycle
   (`AGENTS.md`) applies its corrections itself, within its limits.

## Evidence and temporary files

A completed plan never depends on temporary files for its documentation, its
evaluation history, its reproducibility or its maintenance (the owner,
2026-11-01). The committed code, the plan history and the evaluation files
are the project's durable record.

- **Before a plan closes**, its evaluation file holds what matters: the
  criteria and their outcomes, the measurements and benchmark results relied
  on, failed checks and accepted exceptions, independent reviews' decisions,
  the owner's approvals, known limitations and reservations. A result found
  only in a log or a screenshot is written there first, in words or numbers.
- **What is needed later goes in the repository**, or stays only with the
  owner's decision: a tool kept for reuse, a design reference, a probe's
  recording (sanitized, as the probes README says). A scratch folder or
  `%TEMP%` is not storage: Windows' scheduled cleanup can remove files there
  without asking, as it did on this machine on 2026-11-01.
- **After closure** the plan's working files (builds, tree copies, fixtures,
  mutants, logs, screenshots, review packages) may be removed. An agent moves
  them to the Recycle Bin, never deletes them for good, and leaves alone
  anything unique it cannot place: the owner decides. What is in the bin can
  be recovered only until the bin is emptied.
- **No document or workflow relies on them.** A scratch or `%TEMP%` path in an
  earlier document says where a check or a measurement was made; the file may
  be gone, and the result stands as reported, never rewritten.

## Kinds of exact checks

| Check | Expected value, for example |
|---|---|
| A research claim | a source: repository URL, the commit inspected, file and line; or a documentation URL and the section |
| A license | the SPDX identifier read from the repository's license file, at that commit |
| A dependency | its name and version, read from `go.mod`, `package.json` or the lock file |
| A documented interface | the documentation page and section that defines it; "undocumented" when there is none |
| A probe (plan 1) | approved by the owner first; run in a temporary folder outside the repository; its command, CLI version and raw output recorded in `doc/research/probes/`, with credentials and personal data removed; no CLI configuration changed (hashes before and after) |
| A document | the sections the plan requires, each present; every brief item covered, with where |
| Working copy | `git status` lists only the intended files |
| Build | `go build ./...` exit 0, no warning |
| Tests | `go test ./...` exit 0, with the count of tests run |
| Vet and format | `go vet ./...` exit 0; `gofmt -l .` prints nothing |
| Telemetry parsing | each fixture parses to the expected values; malformed input gives an error, not a crash |
| Agent state transitions | each transition in a table test, including the invalid ones |
| Rendering | a golden snapshot of a view at a fixed size, from mock data, matches |
| Agent CLI configuration untouched | every file under `~/.claude`, `~/.codex` and `~/.gemini` that the work could touch, and the project settings, have the same hash before and after |
| No credential read | no credential file (`auth.json`, `oauth_creds.json`, `.credentials.json`, any token or key) opened, in research notes or in code |
| Each agent CLI optional | with one CLI's data absent, the views show it as not installed and the others keep working |
| Conversation is real | every message in the conversation state traces to an event from the CLI's session (a recorded stream replayed in a test); none is built from status messages |
| Event mapping | each conceptual event of amendment 2 maps to a documented source, with the docs URL, or is marked unavailable with its fallback |
| Real values or "N/A" | each telemetry field with its source absent renders "N/A", never 0 or a guess; API-equivalent cost never shown as subscription usage |
| Rendering without models | the rendering and state packages make no model or network call (dependency check, or a test with the network unavailable) |
| Mock mode labelled | in mock mode every screen shows that it is mock data; mock events go through the same pipeline as real ones |
| Approval and cancellation | an approval request reaches the screen and the answer reaches the CLI; a cancellation leaves the session resumable (a recorded session in a test) |
| Paste preserved | a bracketed paste of fixtures (code with tabs and spaces, CRLF and LF text, Unicode, trailing whitespace) reaches the CLI byte for byte |
| Paste never submits | a multiline paste ending with a newline leaves the message unsent; only the configured send key sends |
| Large input | a 10,000-line paste stays editable, each keystroke handled within a time budget set in plan 2 |
| Clipboard and selection | copy and paste per terminal and OS listed in a matrix (Windows Terminal, PowerShell, WSL2...): each cell tested, or marked not verified |
| Provider-independent input | the same input tests pass with each agent CLI's adapter |
| Provider detection | for each agent CLI, installed gives its version, missing gives the startup error message (a test with the CLI absent from the PATH) |
| Path validation | table tests: Windows path, `~`, WSL path, relative path, trailing separator, non-existent, a file instead of a directory, no access; each accepted or refused with its message |
| Working directory | the started CLI's working directory equals the chosen project path, read from the process or the CLI's own output |
| Startup gate | the dashboard cannot open before both answers are validated |
| MVP scope | none of the features amendment 4 leaves out exists in the code or the screens, except recent projects and favourites (amendment 9), the system's folder dialog in the project browser's place (amendment 53), git worktree management (amendment 10) and the file viewer and Explorer that replace the VS Code integration (amendments 11 and 12) |
| File viewer | a file outside the project, a credential name, a directory, a file over the limit and a binary file are refused before any content is shown, each with its reason; the viewer writes nothing; every key it takes leaves the draft unchanged; the Explorer lists no credential file and nothing outside the project, and its git status leaves the repository's status and `.git/index` unchanged (plan 17) |
| MVP acceptance | plan 4 ends with amendment 4's acceptance criteria, each checked with each agent CLI, or listed under not verified |

A check that cannot fail proves nothing. When you write a new check, run it once
on a damaged input and make sure it fails.

## Keystroke timing

A keystroke is checked twice: a deterministic guard in every test run, and a
timing step on the real clock run on its own (owner, 2026-10-16). Timed under
`go test ./...`, the keystroke tests shared the CPU with every other package
and missed the target from load alone (plan 4, step 4: 2 of 6 suites; step 5,
check 8), with no change in the app's per-key cost. This replaces the earlier
procedure of the same day, which reran them alone after a full-suite miss; the
results it produced stay recorded in plan 4's log (`git show 8888f59:plan.md`).

**The guard, in `go test ./...`.** `TestKeystrokeAllocs` (`internal/app`)
types, moves, deletes and selects in a 10,000-line draft, at its top and at its
bottom. It fails when a keystroke, from `Update` to the frame's string, takes
more than `keystrokeAllocs` (1,200) allocations on average, or when the editor
draws from more than four screen heights of the draft. Allocations do not
depend on the machine's load, so the guard gives the same answer every run;
drawing the whole draft, the whole screen afresh, or the editor one span per
cell breaks it (each checked by mutation). `TestFrameReuse` checks that a
frame built from the one before it equals the frame drawn afresh, escape codes
included.

**The timing step, on its own.** A time is judged against its target only in
the timing step, built with the `timing` tag. Nineteen timed tests carry the
tag, so `go test ./...` never runs them; plan 51's three
(`hometiming_test.go`), `TestChipScale` and `TestChipCount` are built on Windows
only, for its performance counter. Seven more, `TestPasteToFrame`,
`TestStreamToFrame` (`internal/app/perf_test.go`), `TestOutputToFrame`
(`outputtiming_test.go`), `TestBranchesFrameCost` (`branches_test.go`,
plan 59), `TestBlameFrameCost` (`blame_test.go`, plan 60),
`TestGraphFrameCost` (`graph_test.go`, plan 62) and `TestTurnReviewFrameCost`
(`review_test.go`, plan 65), run in
`go test ./...` too, for their other checks: a 50,000-line paste kept byte
for byte, a 2,000-line streamed answer kept as one entry and drawn to its
end, a command's streamed output kept as the call's, the branches popup's
3,000 refs listed with their cap's note, a 10,000-line file's blame kept
whole, its 1,000 commits with it, a history of 50,000 commits read to its
first commit a page at a time, none missing or twice, a turn of 1,000 files
unfolded under the cursor. There
`overTarget` logs their time as "not judged outside the timing step", as it
does under `-race` (`timingBuild`, `timing_on_test.go` and
`timing_off_test.go`; plan 21, item 2). The step runs the twenty-six,
under the owner's normal working conditions (owner, 2026-10-16), since the
target is about real use:

```text
go test -tags timing ./internal/app -run '^(TestKeystrokeToFrame|TestDraftKeystrokes|TestKeystrokeCost|TestLongSession|TestThinkingFrame|TestPasteToFrame|TestStreamToFrame|TestSelectorToFrame|TestMotionToFrame|TestDiffToFrame|TestMessageToFrame|TestOutputToFrame|TestToolsToFrame|TestBannerToFrame|TestHomeKeysToFrame|TestHomePictureTime|TestBranchesFrameCost|TestBlameFrameCost|TestGraphFrameCost|TestTurnReviewFrameCost|TestSearchToFrame|TestLongMessageToFrame|TestLongPasteKeys|TestLongMessageView|TestChipScale|TestChipCount)$' -count=1 -v
```

- Allowed: the owner's usual applications open and idle (VS Code, browsers,
  RanorexStudio).
- Required: the laptop on mains power, and no other heavy job at the same time:
  no other `go` build or test, no Ranorex test execution, no install, virus
  scan or large copy.
- Recorded with each run, not a gate: the power state, the CPU load just
  before the run, and the applications open.

- `TestKeystrokeToFrame` (10,000-line draft) and `TestDraftKeystrokes`
  (50,000 lines, saved while typing) each send 50 untimed warm-up keys, then
  time every key to its frame on the wall clock, the garbage collector on.
- The target is p95 ≤ 16 ms (`keystrokeTarget`); it never changes to make a
  run pass. p50, p95 and the maximum are recorded in `plan.md` with the
  conditions.
- `TestKeystrokeCost` judges nothing: it reports a key's mean time, bytes and
  allocations, the before and after of a change to the drawing.
- `TestLongSession` is test 11 (reliability.md, Test plan): two hours of
  generated events applied at speed, then 400 answer deltas each timed to
  its frame at 200x60; the same 16 ms target at p95 (`streamTarget`). It
  reports the heap and the sizes of the lists the state keeps.
- `TestThinkingFrame` (plan 22, criterion 4.2): a long session of turns
  with three thinking phases each, then zone B's thinking card live at its
  end, a frame every 50 ms of its clock over 4 s as the animation tick draws
  them (ASM-18), its estimate rising every second: each frame timed, the
  same 16 ms target at p95 (`streamTarget`).
- `TestPasteToFrame` pastes 50,000 lines into the editor, ten times, each to
  its frame: the worst within 100 ms (`pasteTarget`, ASM-10).
  `TestStreamToFrame` streams a 2,000-line answer, each delta timed to its
  frame: p95 within 16 ms (`streamTarget`).
- `TestSelectorToFrame` (plan 34, criterion 19): the model selector of
  P34c's mock opened and closed with Alt+P, then its cursor moved through the
  expanded list, each key timed to its frame at 120x40 and 208x59: p95
  within 16 ms (`keystrokeTarget`).
- `TestMotionToFrame` (plan 43, criterion 43.5): with Sixel, at 120x40
  and 208x59, the Explorer's panel hidden and shown, without and with the
  Nerd Font icons (plan 55), the pointer moving from tile to tile over the
  sidebar, each report that changes the hover timed from the program's
  filter through `Update` to its frame: p95 within 16 ms
  (`keystrokeTarget`); then idle motion, within a tile and left of the
  sidebar, every report dropped by the filter before `Update` and `View`,
  counted, or the test fails.
- `TestDiffToFrame` (plan 44, criterion 44.13): a 20,000-line file with one
  line in ten changed (2,000 hunks) open in the diff view at 120x40 and
  208x59, unified and widened side by side; ↓ ↑ PgDn PgUp, n N and the wheel,
  each timed to its frame: p95 within 16 ms (`keystrokeTarget`).
- `TestMessageToFrame` (plan 45, criterion 45.16): Source Control's
  message field at 120x40 and 208x59, with Sixel, so the commit controls'
  images are in each frame's signature; characters, Backspace, a new line
  and the arrows, the field grown to three rows and scrolling, each key
  timed to its frame: p95 within 16 ms (`keystrokeTarget`).
- `TestOutputToFrame` (plan 48, criterion 48.5): a running Codex command's
  output streamed in 8,192 deltas of 8 KiB (64 MiB), each delta timed from
  Update to its frame: p95 within 16 ms (`streamTarget`); then the
  completion's output is the call's. Outside the timing step it streams 512
  deltas and only logs the time.
- `TestToolsToFrame` (plan 49, criterion 49.7): one open turn of 2,000 tool
  calls, 30 of them `Write` calls of 100 KB, while 8 agents run 50 calls
  each; the last tenth of the events each timed from Update to its frame at
  200x60: p95 within 16 ms (`streamTarget`).
- Plan 51, criterion 51.7, the home screen at 208x59 with Sixel, its
  banner running, timed on the performance counter (Go's clock steps by
  0.5 ms on this machine): `TestBannerToFrame`, 1,000 banner frames, each
  from its message through Update to its frame: p95 within 2 ms
  (`bannerTarget`), the bytes a frame sends reported;
  `TestHomeKeysToFrame`, typing in the project field, a banner frame
  between two keys: p95 within 16 ms (`keystrokeTarget`);
  `TestHomePictureTime`, the picture drawn whole on entering, ten times,
  the 120 ms wait for the size to settle counted: the worst within 260 ms
  (`pictureTarget`), the terminal's 40 ms to show it (probe P51) making
  the criterion's 300.
- `TestSearchToFrame` (plan 66, criterion 66.9): the conversation's search
  on the probe's two sessions at 200x60, 240 turns each of a prompt, 20
  calls, a thinking phase and an answer of 30 lines, then of 300, and a
  2,000-line paste: "telemetry" and "refused" typed and taken back in the
  bar, each key timed whole from Update to its frame (the query read, the
  matches found and placed, the rows lit, the frame drawn), then "refused",
  found behind the paste's chip alone; with the bar open on the first
  session, 400 answer deltas, each finding the matches again, timed to
  their frames: p95 within 16 ms (`keystrokeTarget`).
- `TestPasteToFrame` measures both paths since plan 73: in the composer, where a
  50,000-line paste becomes a chip, and with the draft window open, where the whole
  text is drawn. The tests of a large ordinary editor fill it as an ordinary draft
  (`fillOrdinary`: pasted with the draft window open, then Esc) and show it, before and
  after their measured keys (`ordinaryDraft`, criterion 73.23).
- `TestLongMessageToFrame`, `TestLongPasteKeys` and `TestLongMessageView` (plan 73,
  criterion 73.25; `longmessage_timing_test.go`): a 50,000-line chip in zone B's view
  and 400 answer deltas; the motions, typing and a whole-block deletion and its undo
  around one in the composer; the full view of 50,000 lines, then `y`'s key in
  `Update`: p95 within 16 ms (`keystrokeTarget`), `y` at its worst. `TestChipScale`
  (criterion 73.26; `chipscale_test.go`, Windows only, for its performance counter)
  judges nothing: it records the median of 200 samples of the view and of each motion
  around chips of 5 KB, 500 KB and 5 MB, a sample being 20 calls, for the evaluation to
  say how the cost follows the hidden size; `TestChipCount` does the same around 1, 10
  and 100 chips.
- A miss is a FAIL: do not rerun until it passes. Investigate the cause (a CPU
  profile, the same test on the last commit) and report it as the loop above
  says.

## Fuzzing

Plan 48 (audit finding 6) added three fuzz targets for what comes from
outside: `FuzzInspectData` (`internal/input`: every image reader, seeded
with every file `imagetest` writes and the checked-in WebP) and
`FuzzMapperOut` (in each adapter: scripts of recorded lines, the CLI's to
`Out` and the dashboard's to `In`). Plan 49 added two: `FuzzGIFFrames`
(`internal/input`: the GIF block walk takes every GIF `image/gif`'s
`DecodeAll` reads, with as many frames) and `FuzzTextWidth`
(`internal/render`: on text without controls, `textWidth` is
`lipgloss.Width`, and `cut` and `tailCut` keep the most that fits without
splitting a cluster). Plan 73 added two: `FuzzLongMessage` (`internal/render`: the
bounded answer is the sum of a text's rows, at any width) and `FuzzEditorRanges`
(`internal/input`: sequences of every editing operation, in both modes, against a
plain model). `go test ./...` runs their seeds as ordinary tests. A
plan that changes a reader, a mapper or the widths fuzzes it for a bounded
time, each target alone:

```text
go test ./internal/input -run '^$' -fuzz '^FuzzInspectData$' -fuzztime 60s -fuzzminimizetime 2s -parallel 4
go test ./internal/input -run '^$' -fuzz '^FuzzGIFFrames$' -fuzztime 60s -fuzzminimizetime 2s -parallel 4
go test ./internal/provider/claude -run '^$' -fuzz '^FuzzMapperOut$' -fuzztime 60s -fuzzminimizetime 2s -parallel 4
go test ./internal/provider/codex -run '^$' -fuzz '^FuzzMapperOut$' -fuzztime 60s -fuzzminimizetime 2s -parallel 4
go test ./internal/render -run '^$' -fuzz '^FuzzTextWidth$' -fuzztime 60s -fuzzminimizetime 2s -parallel 4
go test ./internal/render -run '^$' -fuzz '^FuzzLongMessage$' -fuzztime 60s -fuzzminimizetime 2s -parallel 4
go test ./internal/input -run '^$' -fuzz '^FuzzEditorRanges$' -fuzztime 60s -fuzzminimizetime 2s -parallel 4
```

- `-fuzzminimizetime 2s`: by default Go minimises each new interesting input
  for up to 60 s and runs nothing else meanwhile, which stopped the image
  fuzzer at 0 inputs a second (plan 48, D1).
- A crash is a FAIL: Go keeps its input under the package's
  `testdata/fuzz/`, where it stays as a seed once the reader is fixed.

## The race detector

`go test -race ./... -count=1` needs cgo, so a C compiler; the owner approved
a portable one under the user profile (plan 4, the fixes before L2 and
step 5, 2026-10-16): WinLibs GCC 16.2.0
with MinGW-w64 14.0.0 UCRT r2, the GitHub release
`16.2.0posix-14.0.0-ucrt-r2` of brechtsanders/winlibs_mingw, unzipped to
`%LOCALAPPDATA%\Programs\winlibs-gcc-16.2.0-ucrt-r2`. Its SHA-256
(`d5dbafc4…5276a`) matched the author's `.sha256` file and GitHub's asset
digest; the zip is not signed.

- No installer, no administrator right, no PATH change: only the race run's
  own shell puts the compiler's `bin` first on `PATH` and sets
  `CGO_ENABLED=1`.
- Go's race detector on Windows needs the MinGW-w64 runtime 8 or later, with
  `libsynchronization.a` (go.dev, the race detector article;
  golang.org/issue/61916); `gcc -print-file-name=libsynchronization.a` must
  resolve it.
- The expected result is 0 data races in every package.
- Timing targets are not judged under `-race`, which slows code several times
  over: in a `-race` build, `raceBuild` (`internal/app/race_on_test.go`,
  `race_off_test.go`) makes `overTarget` log the timing instead of failing.
- `go test -race ./...` skips the patched copy, a module of its own; a plan
  that changes it also runs `go test -race -count=1 -v .` in
  `third_party/ultraviolet`, whose console test checks the Windows reader's
  two goroutines (plan 68).

## Windows Terminal checks

What the terminal draws is also checked in a real Windows Terminal through
`scripts/harness/wt-check.ps1`: on a recording (`-mock`), on the home screen
(`-homeScreen`), or on a disposable repository (`-mock` with `-fixture` and
`-git`). Its head lists its parameters and steps, and what it does and does not
enforce. From the repository root, in PowerShell:

```text
New-Item -ItemType Directory -Force "$env:TEMP\acc-check" | Out-Null
go build -o "$env:TEMP\acc-check\acc.exe" ./cmd/agent-command-center
powershell -NoProfile -ExecutionPolicy Bypass -File scripts\harness\wt-check.ps1 -exe "$env:TEMP\acc-check\acc.exe" -mock testdata\mock\codex-commands.jsonl -wtProfile 'Agent Command Center' -mode max -outName check1 -plan 'expect:end:0:to answer;press:up:3000:26'
```

- Maximised first, then at 120x40 (`-mode 90x30` in ACC's 9 pt profile).
- **It checks first.** A wrong parameter or step stops it before it writes
  anything. Names are letters, digits, `_` and `-`.
- **Its exit code speaks for its assertions.** Only `expect` and `expectfield`
  assert: a failed one keeps its screen and `log.txt`, logs `ASSERTION FAILED`,
  stops the plan and exits 1, and the log's last line counts them. Every other
  step observes: a run of observations exits 0 whatever the screen shows, so
  what it shows is read from its results. `poll` and `fieldcheck`, which logged
  a miss and exited 0, are gone under those names, and a plan naming them is
  refused.
- **It deletes nothing.** Each run makes a new folder,
  `%TEMP%\acc-wt\<outName>-<tag>-<mode>`, and refuses one that exists, or a
  link on the way to it, so each run needs a new `-outName`. No step writes or
  removes a file a plan chooses. What matters of a run goes in the plan's
  evaluation file (Evidence and temporary files, above); the folder stays until
  it goes with the plan's temporary files.
- **No CLI and no quota, by its launch, not by a sandbox.** The launch is fixed: a
  generated `launch.cmd`, holding only checked paths and run by `cmd.exe /c`,
  starts a build that holds the dashboard's main package, with `--mock` or with
  no argument, under a temporary home and app data, a PATH of System32 and every
  inherited `GIT_` setting cleared. No parameter adds an argument or a command.
  The old `cmd /c rmdir` is gone. The build still runs with the owner's rights,
  so a changed build can do anything.
- **A run on a repository** needs a fixture, a folder you made as
  `%TEMP%\acc-wt-fixtures\<name>` and can lose, with no link inside it and no
  `mock.jsonl`. With `-git`, a plan's keys and clicks drive the dashboard's real
  git there, contained so that it touches only the fixtures; the harness's head
  has the rules. Before the launch it refuses a `.git` file or another
  repository inside, a config key off its allow-list, an active hook, alternates,
  and a remote that is not a plain local path to a bare repository among the
  fixtures. In the launch it forces an empty hooks folder and allows only the
  file transport, so a push reaches a local remote and never a network one. The
  checks run once: another process changing a fixture during a run is not seen.
- Its keys and clicks are real. `Term.cs` checks before each batch that its
  window has the focus and stops otherwise, but it cannot see a focus change
  inside a batch: nothing else uses the machine during a run, and the owner is
  told first.
- It holds no clipboard code, and none is to be added: the antivirus
  quarantined a harness that saved and restored the clipboard beside its keys
  (plan 51). A double click and `y` make the dashboard write the owner's
  clipboard, so those steps are the owner's: a rule, which the code does not
  enforce.
- After a change to it, `powershell -NoProfile -ExecutionPolicy Bypass -File
  scripts\harness\wt-check-tests.ps1` runs its tests with no terminal and no key
  sent: refused parameters and steps, and git's containment on disposable
  repositories under a hostile environment. `wt-check-window-tests.ps1`, with
  `-exe` and the same build, runs those that need a screen, on a recording:
  assertions that pass and that fail, a commit and a push on a fixture. It opens
  terminals, so the owner is told first.
- `scripts/harness/screen-colours.ps1` reads colours back from a shot;
  `image-colours.ps1` and `glyph-colour.ps1` read them from a design image.

### The migration's approval

These tools reached the repository through a maintenance migration after plan
72, which this record closes; plan 72's results are unchanged. An independent
reviewer inspected the code, tests, documentation and supplied evidence of the
archive `acc-migration-review-3.zip` (SHA-256
`770CE36A90C386334503B06C3DB286CA5E18E6265E87BB33D28F7BA0B5F21C10`, baseline
`890536844bffd65bf61639804c47382733b4b091`). The reviewer did not run the Windows
tests: they ran on the owner's machine, and their results were part of that
evidence. The verdict, word for word:

> APPROVED WITH RESERVATIONS — no blocking findings.

The owner approved the commit on 2026-11-01 and accepted these four reservations.
They are limitations that stand, not findings that were resolved:

- **R1, no sandbox:** the executable and Git retain the Windows user's
  permissions.
- **R2, symbolic links:** junctions were tested, but symbolic links could not be
  exercised on the available Windows account.
- **R3, Git compatibility:** the relevant protections were verified with Git 2.56,
  not every supported version.
- **R4, real input events:** keyboard and mouse automation can affect the active
  window or clipboard under the documented circumstances.

## After a CLI update

**[Recurring]**: on every Claude Code or Codex CLI update, never closed
(`plan.md`, Recurring).

The dashboard depends on each CLI's documented interfaces and on a few
undocumented fields, each labelled where it is used. A new CLI version can
change either without notice, so an update is never silent.

**At session start**, the agent compares `claude --version` and
`codex --version` with the last probed versions, the "Last probed" line of
[the probes README](research/probes/README.md). If one is newer, it tells the
owner and proposes the re-check below; nothing runs before the owner
approves. The dashboard says the same: its project bar adds "newer than
tested" when the session's CLI is newer than the version its adapter names
(`TestedVersion` in `internal/provider/claude` and `internal/provider/codex`).

**The re-check**, for the updated CLI only, on the owner's quota, each run in
a temporary project outside the repository with the configuration check
before and after:

1. The live gate: `go test -tags live -count=1 -v -run TestLiveGate` in
   `./internal/provider/claude` or `./internal/provider/codex`, with
   `ACC_LIVE_PROJECT` set as the test's comment says.
2. The probe drivers of `doc/research/probes/drivers` the dashboard's
   features rest on (a conversation, a subagent, an approval, a question for
   Claude Code; a conversation, a subagent, an approval for Codex),
   recorded and sanitized as the probes README says.
3. Every new recording added to the replay tests (`internal/state`
   `TestReplayDeterministic`; Codex's `TestEveryLineMaps`), which must show
   no unrecognised event; a difference is investigated, never accepted by
   changing a test.
4. Once every check passes: the "Last probed" line and the adapter's
   `TestedVersion` updated together (`TestTestedVersion` compares them), and
   the result reported in the summary format.

A live run of the dashboard through the test-only logging wrapper
(`doc/research/probes/drivers/wirelog`, built as `codex.exe` or `claude.exe`
and put first on the window's PATH, a `wrap.json` beside it naming the real
CLI and the log folder: stdin and stdout copied with each line's time) can be
replayed through the Claude Code mapper afterwards: `go test ./internal/provider/claude -run
TestWireReplay -count=1 -v` with `ACC_WIRE_LOG` set to the log folder prints
every event kind and names each line the mapper could not map, with the
mapper's type and reason and the line itself (plan 25). The dashboard shows
the same lines live in its details view.

**The wrapper's logs are raw.** They hold everything the session sent and
received: prompts, answers, file contents, paths, whatever a tool printed. They
may hold sensitive information: inspect and sanitize them, as the probes README
says, before they are shared or committed.

## The Reviewer

How a plan's steps are reviewed (`AGENTS.md`, Review cycle). The CLI offers
the Reviewer `Read`, `Grep` and `Glob`, its own report tool, and `advisor`, a
model it may consult, which touches nothing on the machine. Its definition,
`.claude/agents/reviewer.md`, has exactly these fields, which `internal/arch`
guards: name `reviewer`, description, model `claude-opus-5-5`, effort `high`,
tools `Read, Grep, Glob`, omitClaudeMd `true`, maxTurns `100`, and color. A
`go test -overlay` mutant repoints the guard's path constant at a changed copy,
to show that the guard fails.

**Inputs**, in `%TEMP%\acc-review-inputs\`, rewritten for each review so that
nothing of an earlier one stays there:

- `plan-files.txt`: each file of the plan that exists, one path per line from
  the repository root, records included (`plan.md`, `doc/brief.md`, and the
  plan's evaluation file once it exists);
- `changes.diff`: `git --no-optional-locks diff <baseline> -- <the plan's
  tracked files>`;
- `new-files.txt`: the plan's untracked files, which the Reviewer reads whole in
  the repository. No other untracked file is read or hashed, and nothing goes
  in the index, `git add -N` included;
- `evidence.txt`: the Implementer's evidence file;
- `criteria.txt` and `baseline.txt`: the approved criteria, word for word, and
  the baseline's hash;
- the orchestrator's own evidence, where a criterion needs what only it holds
  (the full test run, transcripts, settings, the index): each command printed
  before its raw output, and nothing that argues for a verdict. A reading of a
  criterion goes in the report to the owner, never to the Reviewer.

**The fingerprint** ties the evidence to the code: every byte of each file in
`plan-files.txt`, with `git status`, which names any other change. In Git Bash,
from the repository root:

```text
F="$TEMP/acc-review-inputs"; { git --no-optional-locks status --porcelain --untracked-files=all; while IFS= read -r f || [ -n "$f" ]; do sha256sum -b -- "$f"; done < "$F/plan-files.txt"; } | sha256sum
```

**The evidence command** is the Implementer's last, from the same place; a
step's own checks go before the redirection, each printing its command and
inputs before its output:

```text
F="$TEMP/acc-review-inputs"; { { git --no-optional-locks status --porcelain --untracked-files=all; while IFS= read -r f || [ -n "$f" ]; do sha256sum -b -- "$f"; done < "$F/plan-files.txt"; } | sha256sum; go build ./...; echo "build exit $?"; go vet ./...; echo "vet exit $?"; gofmt -l .; echo "gofmt done"; go test ./...; echo "test exit $?"; } > "$F/evidence.txt" 2>&1
```

**Around each review.** Before it: `git status` against the plan's files, the
fingerprint against the evidence file's first line, the inputs rewritten; the
orchestrator's full test run starts with the fingerprint too. After it: the
fingerprint again; in the transcript (`agent-*.jsonl`, `agent-*.meta.json`),
the tools offered, the model and the effort, and every path its `Read`, `Grep`
and `Glob` reached. A tool beyond the definition's and the CLI's two, a path
outside the repository other than the inputs, or an earlier verdict on this
plan's work makes the review count for nothing. A search of the repository can
meet earlier plans' evaluation files, which judge other work.

**Tiers** (`CLAUDE.md` gives their efforts), set in the plan before any code;
size never decides, and when two tiers apply, the higher one does:

1. Can runtime behaviour change? If not, Light is possible.
2. Does it touch event order, concurrency, approvals, process lifetime, Git,
   credentials, security, saved drafts or the agent rules (`.claude/**`,
   `AGENTS.md`, `CLAUDE.md`)? Deep.
3. If it were wrong, would it fail silently or lose data? Deep.
4. Could no deterministic test catch it being wrong? One tier up: reading is
   then the only check.
5. Did it need the Opus Implementer? At least Standard.

**The record.** Each Agent call's description names the tier and its reason.
At the plan's close, its evaluation file gets the reports and one line per
review: the Reviewer's agent ID, model, effort, tier, reason, minutes and
verdict, each checked against the transcript's metadata.

**Known limits.**

- The tool list is the CLI's lock: no shell, edit, test, clipboard or desktop.
  What `Read`, `Grep` and `Glob` open is a rule: they can reach any path, and
  only the definition keeps them to the repository and the inputs; the
  transcript shows afterwards what they opened.
- A read-only Reviewer judges, it does not verify by running: a `reported`
  criterion rests on evidence. The orchestrator's full run repeats the build,
  vet, gofmt and tests once per plan; a step's own checks, a mutant's for one,
  rest on the Implementer's evidence alone.
- A changed definition can reach new Reviewers late, for a reason not known:
  in plan 74, two Reviewers started two and four minutes after a rewrite still
  got the old one, body and tools, and one started nine minutes after an edit
  got the new body with no message from the owner in between. The guard test
  checks the file; the transcript shows what each Reviewer got.
- The fingerprint covers the plan's files and `git status`, not ignored files
  (`.claude/settings.local.json`) or the Go environment (`GOFLAGS`, `GOWORK`).
- The CLI also gives a Reviewer the session's git status, the user's email and
  the instructions of connected servers, which no marker check covers.
- Nothing mechanical detects a criterion edited during a plan: the report's
  criteria are compared with those the owner approved.
- The cycle is the orchestrator following these rules, not a lock. Each
  Reviewer's own report stays in its transcript, where a claimed verdict can be
  checked.
- Codex has no Reviewer (`AGENTS.md`, Review cycle).

## Summary format

```text
Evaluation - <the instruction in a few words> - <n> criteria: <p> PASS, <f> FAIL
PASS 1  <criterion>: <evidence>
FAIL 2  <criterion>: expected <...>, found <...>
Not verified: <what could not be checked, and why>
Verdict: DONE | NOT DONE
Likely cause: <for each FAIL>
Proposed next step: <for each FAIL> - waiting for approval
```

## Examples

### Research one reference project (plan 1)

Instruction: "Inspect jedarden/ccdash."

```text
1. doc/research/ccdash.md exists, with the sections of plan 1
2. Commit inspected recorded (full SHA) and its date
3. License: SPDX id from its LICENSE file, and what it allows (fork, copy, link)
4. Language, UI library and every direct dependency with its version, from go.mod
5. Each data source it reads (files, endpoints, hooks), with file:line where it reads it
6. Each source marked documented (with the docs URL) or undocumented
7. Agent CLIs it supports (Claude Code, Codex CLI, others), and how
8. Reusable parts listed, each with its license consequence
9. No claim without a source
Not verified: whether it runs on this machine (nothing is installed in plan 1)
```

### Parse one agent CLI's session telemetry (an implementation plan)

Instruction: "Parse token usage from Codex CLI session files."

```text
1. The Codex collector has a parser with table tests over fixtures in testdata/,
   and fills the common model only
2. Fixtures cover: a normal session, a session with parallel work, an empty
   file, a truncated last line, an unknown event type
3. go test ./... exit 0, n tests; the truncated-line test expects an error, not a panic
4. The parser never writes: no file under ~/.codex changes (hashes before/after);
   auth.json never opened
5. With no Codex data at all, the views show Codex as not installed; the Claude
   Code views unchanged
6. go vet ./... exit 0; gofmt -l . empty; go build ./... exit 0
Not verified: real values against the vendor's usage page (needs the owner's account)
```

## Known limits

- **The look is the owner's call.** A golden snapshot proves a view did not
  change, not that it looks right. Show the owner a screenshot or a mock-data
  run, and list "visual review" under not verified.
- **Quota figures are only as good as their source.** Each vendor counts and
  resets usage its own way (Claude's 5-hour and 7-day windows have no exact
  equivalent elsewhere). If a figure comes from an undocumented source, it can
  change without notice; say so wherever it is shown.
- **Windows is the host.** The owner works on Windows 11; tmux only runs under
  WSL there. Anything that needs tmux, a Unix shell or a POSIX terminal must say
  how it runs on this machine.
- **The network goes through a proxy.** `curl` fails the TLS handshake here;
  PowerShell's web client with the system proxy works (see AGENTS.md).
