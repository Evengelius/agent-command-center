# Current Plan

The active plan with its progress, the owner's standing decisions, and one
entry per closed plan. Each plan comes from `doc/brief.md` and its
amendments, ends with the evaluation summary of `doc/evaluation.md`, and
waits for the owner's approval before the next one starts. This file keeps
the current state only (the owner, 2026-10-18): a closed plan's full text,
criteria and progress log are in git (`git show 8888f59:plan.md` for plans 1
to 13, `git show 84d2ad8:plan.md` for plan 14, `git show b047434:plan.md`
for plan 15, `git show becfb34:plan.md` for plan 16, `git show f00c2d8:plan.md`
for plan 17, `git show 8b3b16b:plan.md` for plan 18, `git show
d84e5d4:plan.md` for plan 20, `git show f939d96:plan.md` for plan 21,
`git show d13ef71:plan.md` for plan 22, `git show e8d3e3b:plan.md` for
plan 23, `git show a3bfe5d:plan.md` for plan 24, `git show cb8f76d:plan.md`
for plan 25, `git show a9a5145:plan.md` for plan 26, `git show 041e7b8:plan.md`
for plan 27, `git show 8fe0077:plan.md` for plan 28, `git show d696f22:plan.md`
for plan 29, `git show 6025513:plan.md` for plan 30, `git show
8e2a721:plan.md` for plan 31, `git show b56d23e:plan.md` for plan 32, `git show
bc5d92b:plan.md` for plan 33, `git show 47b50ce:plan.md` for plan 34, `git show
b43cf46:plan.md` for plan 35, `git show c0cebc4:plan.md` for plan 36, `git show
31c22ef:plan.md` for plan 37, `git show dc9c6d9:plan.md` for plan 38, `git show
55e2bed:plan.md` for plan 39), and its results in its evaluation file.

## The owner's standing decisions

Decisions of closed plans that still apply; `AGENTS.md` and
`doc/evaluation.md` hold the others.

- **Live runs.** Each one is proposed first with its scope, model, expected
  usage and stopping conditions, and runs only once approved (plan 3,
  2026-10-16); Claude runs use Haiku as a per-process override.
- **The configuration check is hash-only**: credential contents are never
  read, and a file that cannot be classified safely is "changed, not
  classified", never reported unchanged (2026-10-16). No parser runs over a
  file that may hold credentials (the mixed-state tokenizer was withdrawn,
  2026-10-16); the one exception is confcheck's key-level comparison of
  `~/.claude.json`, sensitive keys skipped (plan 5).
- **Measured, not assumed.** Performance figures are targets until a test
  measures them; a documented design is never presented as verified runtime
  behaviour (2026-10-16).
- **The integration.** If driving a CLI without a terminal falls short, the
  limit is documented and the smallest adjustment proposed; the architecture
  never changes silently (2026-10-16).
- **Drafts and session data** survive cancellation, crashes and recovery; no
  conversation data is stored beyond the draft, and nothing sensitive goes
  into a log (2026-10-16).
- **Zone A shows values, not notes** (amendment 25, 2026-10-22, plan 26):
  its telemetry pane carries no notes row, legend or counter, now or in
  any later plan; errors, retries, unrecognised lines, freshness and the
  legends live in the details view (Enter on TELEMETRY), where each value
  has its source and age. An undocumented value keeps its "†" mark where
  it is shown.

## Recurring

Tasks tagged **[Recurring]** repeat on an outside event: they are never
closed, never listed as remaining, and each occurrence is reported in the
summary format (the owner, 2026-10-18).

- **[Recurring] After a CLI update.** At session start, `claude --version`
  and `codex --version` are compared with the "Last probed" line of
  `doc/research/probes/README.md`; a newer one is told to the owner with the
  re-check proposed (`doc/evaluation.md`, After a CLI update), which runs
  only once the owner approves it. The dashboard's project bar shows
  "newer than tested" meanwhile. Last occurrence: Claude Code 2.1.303 on
  2026-10-19, approved, every check PASS (P303a to P303d in
  `doc/research/probes/README.md`); it found that plan 14's shape check
  counted an interrupted tool use as unrecognised, fixed the same day at the
  owner's request (P303e).

## Nice to have

Items tagged **[Nice to have]** are future improvements: none is built, and
none is listed as remaining, until the owner asks for one in a new plan
(the owner, 2026-10-18). Amendment 4 names ten "future improvements"
(`doc/brief.md`, Amendment 4, "For this phase, do NOT implement"). Five
remain below. Recent projects, favourites and a project browser were plan 15
(amendment 9); git worktree management was plan 16 (amendment 10). The
automatic VS Code integration was replaced by the owner with a file viewer in
the terminal, plan 17 (amendment 11). The session's model, effort and
Ultracode selection was plan 34 (amendment 34); configuring models beyond the
session stays below.

- **[Nice to have]** Advanced model configuration.
- **[Nice to have]** Cross-provider agent collaboration.
- **[Nice to have]** Provider switching during an active session.
- **[Nice to have]** Complex session profiles.
- **[Nice to have]** Advanced orchestration configuration.
- **[Nice to have]** Google Antigravity as an agent CLI, removed from the
  MVP by amendment 5; the provider interface stays open to it.
- **[Nice to have]** Starting it like an app (the owner, 2026-10-19): a built
  `agent-command-center.exe` started from a Start-menu shortcut, pinnable to
  the taskbar, whose target is `wt.exe -d <projects folder> <path to the
  .exe>`, so that it always opens in Windows Terminal (the only supported
  terminal, amendments 6 and 7) with the project field at that folder; the
  `.exe` is rebuilt after each code change.

## Plan 48 - The audit's phase 2: robustness at the boundaries

Brief amendment 49 (2026-10-25): phase 2 of the audit's roadmap
([`doc/audit.md`](doc/audit.md), Part 2, C), its six findings:
- 2: the TIFF and ICO decoders' unbounded allocations;
- 7a: a running command's streamed output, kept whole;
- 9: the Claude Code reader writing to stdin itself;
- 3c: git calls with no bound on a child that keeps their pipes;
- 6: panics outside Bubble Tea's goroutines, and no fuzzing;
- 8: format characters on the approval card.

Each fix is test-first: its test is written and seen failing on today's
code, then the change follows. The criteria below were written before the
code. Nothing is committed until the owner has reviewed the result. The
phase 0 safeguards that phase 2 relies on come with their fix: the fuzz
targets (findings 2 and 6) and the large-output timing test (7a).
First-frame GIF decoding stays optional (phase 4).

### What was found

- **2, images** (`internal/input/imagetiff.go:46-65`, `imageico.go:47`).
  - The TIFF reader copies every directory entry's values into a new
    slice, up to 1,048,576 values each, for up to 65,535 entries whose
    values may overlap, though it uses ten tags. The audit measured 64 MiB
    allocated for 16 entries of a 1 MiB file.
  - The ICO reader decodes an embedded PNG without reading its size
    first: a PNG that declares 9000x9000 px makes `png.Decode` allocate
    its 81 MB image before it fails.
- **7a, streamed output** (`internal/state/reduce.go:499`).
  - Each Codex `outputDelta` is appended to the tool's output
    (`t.Output += p.Output`), which copies the whole output each time, with
    no bound. The audit measured 18.3 s for 32 MiB in 4 KiB deltas, late
    deltas about 17 ms each, on the event loop that draws the screen.
  - The inspection and the full view draw the whole output on each frame.
  - The completed item replaces it (`t.Output = p.Output`).
- **9, the Claude Code reader** (`internal/provider/claude/session.go:183-205`).
  - On the handshake's answer it writes `get_settings` and "/usage"
    itself, and after a settings change the read that follows. Codex's
    reader never writes (`codex/session.go:211`).
  - If Claude Code stops reading stdin while its stdout fills, the reader
    blocks on the write and the pipes deadlock.
- **3c, git** (`internal/input/git.go:116`).
  - Every git call starts through `command`, with no `WaitDelay`: a hook
    or helper that keeps git's output open after git ends keeps `Wait`
    waiting. The writes (stage, commit, push) have no time limit, so a
    commit would show "Committing…" for as long as that child lives.
  - `stream` drains its `StdoutPipe` before `Wait`, which no `WaitDelay`
    can bound.
  - Probe of 2026-10-25 (`scratchpad\plan48\probe\waitdelay`, Go 1.27.2
    on this machine), a grandchild holding stdout 12 s after its parent
    exits 0, with a `WaitDelay` of 1 s:
    - with output buffers, and with an `io.Pipe` read beside `Wait`, `Wait`
      ended after 1.06 s, returning `exec.ErrWaitDelay`, the output intact;
    - the `StdoutPipe` drained before `Wait` was still draining after 6 s.
- **6, panics and fuzzing.**
  - No `recover` in the adapters or in the application's own goroutines
    (`internal/app/writer.go:26`, the detection at `app.go:835`). A panic
    there ends the program with the terminal still in alternate screen,
    mouse and win32-input modes.
  - The adapters hold their mutex while their mapper maps a line
    (`claude/session.go:176` and `:320`, `codex/session.go:222` and
    `:598`): a recovered panic there would leave it locked.
  - No fuzz target exists.
- **8, the approval card** (`internal/render/text.go:59-87`, `approval.go`).
  - `visible` neutralises C0, DEL and C1 only. Bidi controls and
    zero-width spaces reach the card and its full view unchanged.
  - Windows Terminal reorders nothing but draws them as nothing (audit,
    Part 2, A).

### Design

- **D1. Bounded image decoders** (2).
  - TIFF: only the tags the reader uses are kept, the first of each, and
    their values are read from the file when used, never copied. Other
    tags are skipped.
  - ICO: the embedded PNG's size is read (`png.DecodeConfig`) and checked
    (`tooBig`) before it is decoded.
  - `FuzzInspectData`, seeded with every format `imagetest` writes and the
    checked-in WebP.
- **D2. A running command's output keeps its tail** (7a).
  - `state.MaxLiveOutput` (64 KiB): a streamed output over it keeps its
    last bytes, from a line's start when one begins in them, else from a
    character's. The bytes dropped are counted (`ToolCall.OutputCut`).
  - The completed item's output replaces it whole, the count cleared.
  - While the count is not zero, the inspection and the full view open the
    output with how much earlier output is not kept.
  - `TestOutputToFrame`, a timing test: 8,192 deltas of 8 KiB.
- **D3. The Claude Code reader never writes** (9).
  - The handshake's two reads and the read after a change run in a
    goroutine of their own, as Codex's replies do.
  - The handshake's events stay held until both reads are written, so
    what the dashboard sees is in the same order.
- **D4. Git's pipes bounded** (3c).
  - `command` sets `WaitDelay` (2 s): once git has ended, its pipes are
    closed 2 s later at the latest.
  - `call` and `exec` take `exec.ErrWaitDelay` as git's success, which it
    is: git exited 0 and its output was read.
  - `stream` reads git's stdout through an `io.Pipe` while `Wait` runs
    beside it, which `WaitDelay` bounds.
- **D5. Panics contained** (6).
  - `provider.Recovered`: a panic's value and the frame it came from, as
    text.
  - Each adapter's stdout reader and the goroutines it starts for writes
    recover. The panic becomes an Error marked `Fatal`, and the reader
    reads its input to the end, so the CLI never blocks. The mapping runs
    in functions that release the mutex by `defer`.
  - The reducer turns a fatal error into `Crashed`, its message the
    reason, and closes the CLI. While `Closing`, it is only kept as an
    error, so the quit still ends.
  - The application's write worker recovers into `state.Panicked`, which
    crashes and closes the same way. The detection recovers into the CLI's
    problem ("detection failed: …").
  - The stderr readers, the queue's forwarder, the waits and the mock
    replay parse nothing from outside, and stay as they are.
  - `FuzzMapperOut` in both adapters, seeded with every line the CLI wrote
    in the recordings.
- **D6. Format characters revealed on the approval card** (8).
  - On the approval card, its answered row and its full view, the bidi
    controls (U+061C, U+200E, U+200F, U+202A to U+202E, U+2066 to U+2069)
    and the zero-width spaces (U+200B, U+2060, U+FEFF) are drawn as
    `<U+XXXX>`.
  - The zero-width joiner stays, for emoji sequences and the scripts that
    need it.
  - Elsewhere nothing changes. The questions' texts, which key the
    answers, are revealed only when drawn.

### Criteria

1. **48.1 A crafted TIFF stays bounded.**
   - A 1 MiB TIFF with 16 entries of an unused tag, 1,048,576 values each
     over the same bytes, is refused with under 8 MiB allocated; so is one
     whose 16 entries are strip offsets. (Its refusal is "photometric 2
     with 1 samples: not read", the reader's first check that fails; the
     plan first said "no strips".)
   - The test TIFF still decodes (`TestImageInspect`).
   - On today's code: 64 MiB or more allocated.
2. **48.2 A crafted ICO stays bounded.**
   - An ICO whose PNG declares 9000x9000 px is refused ("ICO: 9000x9000
     px: over 60 megapixels") with under 8 MiB allocated.
   - The test ICO still decodes.
   - On today's code: about 81 MB allocated, and an error from the PNG's
     data.
3. **48.3 The image fuzzer.** `FuzzInspectData` runs 60 s without a
   crash. A crash it finds is fixed in this plan, its input kept as a
   seed.
4. **48.4 A running command's output keeps its tail.**
   - After 200 deltas of 1 KiB, the output is at most 64 KiB, ends with
     the last delta, begins at a line's start, and `OutputCut` plus its
     length is 200 KiB.
   - The completion's output replaces it, and `OutputCut` is 0.
   - The inspection and the full view say how much earlier output is not
     kept.
   - On today's code all 200 KiB are kept.
5. **48.5 Large output stays fluid** (the timing step).
   - 8,192 deltas of 8 KiB (64 MiB), each timed from Update to its frame:
     p95 at most 16 ms.
   - After the completion, the output is the completion's.
   - On today's code the p95 is over 16 ms.
6. **48.6 The Claude Code reader never waits on stdin.**
   - A fake Claude Code on in-memory pipes stops reading stdin after
     `initialize`: three more stdout lines are each read within 2 s.
   - Once it reads stdin again, `get_settings` then "/usage" arrive, and
     `session.ready` comes before both writes' events.
   - The existing handshake, usage and settings tests pass unchanged.
   - On today's code the next stdout line is never read.
7. **48.7 Git's pipes are bounded.**
   - A fake git (this test binary) exits 0 while a child it started keeps
     its stdout and stderr 15 s. A read (`run`), a write (`Stage`,
     through `exec`) and a stream each return within 8 s, with git's
     output and no error.
   - On today's code each takes about 15 s.
8. **48.8 A panic in the application's goroutines is contained.**
   - A scripted session whose `Send` panics: the session is `Crashed`,
     its card says "The session ended unexpectedly" and "writing to the
     CLI failed: panic: …"; the CLI is closed (one `Close`); the message
     is back in the editor; the program runs on.
   - A provider whose `Detect` panics is listed with "detection failed:
     panic: …", and the startup screen goes on.
   - On today's code the test binary dies.
9. **48.9 A panic in an adapter is contained.**
   - Each adapter's reader, its mapping panicking (a nil mapper), returns.
     It has pushed a fatal Error naming the panic, read its input to the
     end, and left both mutexes free.
   - A function each adapter runs apart that panics gives the same fatal
     Error.
   - `TestStateMachines`: a fatal error crashes the session with its
     message and closes the CLI; while `Closing`, it is only kept;
     `Panicked` likewise.
   - On today's code the panic escapes.
10. **48.10 The mapper fuzzers.** `FuzzMapperOut` runs 60 s in each
    adapter without a crash. A crash found is fixed here.
11. **48.11 Format characters on the approval card.**
    - An approval whose command holds U+202E and U+200B shows `<U+202E>`
      and `<U+200B>` on its card and in its full view, and neither raw
      character.
    - An emoji sequence joined by U+200D in its summary is drawn whole.
    - The same characters in an assistant message are drawn as before.
    - On today's code the raw characters are drawn.
12. **48.12 No regression.**
    - Build, vet and gofmt clean; `go test ./...` passes after the last
      document edit.
    - No golden frame changes; the probe replays and mock files
      unchanged.
    - `-race` passes on input, app, state, render and the providers.
    - The timing step, with `TestOutputToFrame`, within its targets.
13. **48.13 Mutations.** Each fix is broken on purpose through
    `go test -overlay`, and each mutant is caught by its test.
14. **48.14 Windows Terminal.** At 208x59 and 120x40:
    - on a temporary repository whose pre-commit hook leaves a background
      process holding git's output for 30 s, Commit shows "Committed …"
      within 5 s of Ctrl+Enter;
    - in a mock replay whose approval holds U+202E and U+200B, the card
      shows `<U+202E>` and `<U+200B>`.

### Steps

1. The plan and brief amendment 49.
2. Each fix test-first: D1 (48.1 to 48.3), D2 (48.4, 48.5), D3 (48.6),
   D4 (48.7), D5 (48.8 to 48.10), D6 (48.11).
3. The checks (48.12, 48.13), Windows Terminal (48.14), the documents, the
   evaluation (`doc/plan48-evaluation.md`), the report.

### Progress

- 2026-10-25: step 1 done; step 2 in progress.
- 2026-10-25: D1 done.
  - `TestCraftedImagesBounded` was written first and failed on the old
    code: 64.0 MiB allocated for each crafted TIFF, 77.3 MiB for the ICO.
  - Then the TIFF reader's entries left in the file (`tiffEntry`) and the
    ICO's PNG size read first: 0.00 MiB for all three.
  - `FuzzInspectData`: a first 60 s run stopped at 0 executions a second,
    because Go minimises each new interesting input for up to 60 s by
    default. No input took over a second (a scratch copy of the target,
    added by overlay, wrote out any that did: none). With
    `-fuzzminimizetime 2s` it ran 102,832 inputs in 21 s, no crash.
  - Once, the fuzz build itself failed in Go's linker (a panic in
    `cmd/link`, while other builds ran); the same command built and ran
    at once after.
- 2026-10-25: D2 done.
  - `TestLiveOutputTail` was written first and failed on the old code:
    all 204,800 bytes kept. Then `liveTail` and `OutputCut`: 65,536 kept,
    139,264 counted as dropped; the cut's edge cases pass.
  - The inspection and the full view say "… 139 KB of earlier output not
    kept" (`TestOutputCutNote`).
  - `TestOutputToFrame` passes in the ordinary suite (512 deltas, p95
    1.0 ms); its timing-step run and its run on the old reducer come with
    the checks.
- 2026-10-25: D3 done.
  - `TestReaderNeverWaitsOnStdin` was written first and failed on the old
    code: "stdout line 2 not read within 2 s".
  - Then the handshake's reads and the read after a change in goroutines
    of their own. The events seen are in the same order: initialize,
    session.ready, get_settings, usage. The Claude suite passes unchanged;
    the new test passed 20 runs in a row.
- 2026-10-25: D4 done.
  - `TestGitPipesBounded` was written first and failed on the old code:
    each case took 15.1 s; the read ended in "git status took more than
    15s" with its output lost, the stream likewise.
  - Then `WaitDelay`, `exec.ErrWaitDelay` taken as success, and `stream`
    through an `io.Pipe`: each case ends after about 2.1 s, with git's
    output. The input package passes whole.
- 2026-10-25: D5 done.
  - `TestWritePanic` and `TestDetectPanic` were written first; on the old
    code each killed the test binary ("panic: the adapter broke",
    "panic: detection broke").
  - Then `provider.Recovered`, `event.Problem.Fatal`, `state.Panicked`,
    the reducer's `fatal`, the app's `runWrite` and detection recover, and
    in both adapters the reader's recover and drain, `mapOut` and `holdIn`
    (the mutex released by `defer`), `apart` and `fatal`.
  - The adapters' `TestReaderPanic` and `TestRecovered` were written with
    the code; their mutants must show that they fail without it. The two
    documented rules have their cases (`TestStateMachines`).
  - `FuzzMapperOut` in both adapters: 30 s runs, 298,799 inputs (Claude
    Code) and 196,662 (Codex), no crash.
  - A slip, caught at once and undone: writing the app's test, the Write
    tool overwrote the existing `internal/app/panic_test.go` (plan 33's
    `TestPanic`). It was restored from git (`git checkout`, the diff
    empty) and the new tests went into `recover_test.go`.
- 2026-10-25: D6 done.
  - `TestApprovalRevealsFormat` was written first and failed on the old
    code: the raw characters drawn on the card and in the full view.
  - Then `revealed` on the card, its answered row, the questions and the
    full view. `viewerBodyFor` also draws a tool call's full view, which
    therefore reveals them too: the same input, the same safety reading.
    The render suite and the golden frames pass unchanged.
- Step 2 done; step 3 in progress.
- 2026-10-25: before the mutations, two cases were added so that a mutant
  could be caught at all: the read after a settings change in
  `TestReaderNeverWaitsOnStdin`, and a TIFF whose later widths must not
  replace the first in `TestCraftedImagesBounded`. `TestApprovalRevealsFormat`
  gained a question and a summary with a zero-width space.
- 2026-10-25: the mutations, 44 of 44 caught
  (`scratchpad\plan48\mutate48.mjs`; Go files unchanged by the run).
  - The first run caught 41. "A line's start not seen" and "a cut inside
    a character" were not caught: `liveTail`'s edge cases had no later
    line after a cut on a line's start, and their two-byte characters made
    every cut fall on a character's start. The cases now have a second
    line and three-byte characters. The first of the two mutants had also
    left `utf8` unused, a build failure, and was rewritten.
  - "Revealed everywhere" changed `visible`, which an assistant message
    never goes through; it now changes `entryLines`, and the message's
    check catches it.
  - The six mutants of that test were run again: all caught.
  - Two mutants deadlock, the adapters' mapping lock not released by
    `defer`: each run ends by its 120 s timeout, a failure.
- 2026-10-25: the fuzzers, 60 s each with `-fuzzminimizetime 2s`, no
  crash: 243,348 image inputs, 762,522 Claude Code scripts, 733,327 Codex
  scripts.
- 2026-10-25: the timing step alone, twelve tests within their targets
  (on mains power, the CPU 24% busy just before); `TestOutputToFrame`,
  64 MiB in 8,192 deltas: p95 1.04 ms, max 11.8 ms.
  - On the old append (an overlay of `reduce.go`), the same test: p50
    4.0 ms, p95 7.42 ms, max 16.3 ms, then its output check fails (all
    64 MiB kept). Criterion 48.5 predicted a p95 over 16 ms on today's
    code, from the audit's 17 ms late deltas at 32 MiB: on this machine
    that does not hold, and the criterion is reported as such.
- 2026-10-25: `-race` ok on input, app, state, render and the five provider
  packages.
- 2026-10-25: Windows Terminal at 208x59 and 120x40
  (`scratchpad\term\p48call.ps1`): the pending card read
  "echo safe<U+202E>txt.exe<U+200B> && …", no raw U+202E on the screen;
  Commit, its hook leaving `sleep 30 &` on git's output, showed
  "Committed …" about 3 s after Ctrl+Enter. A probe of the same hook with
  real git: the old Wait took 30.4 s, the new one 2.33 s.
  - Both runs ended with the harness closing the window: the hook's
    `sleep` holds the tab's console. Sampled at 120x40, the dashboard left
    1 s after the last Ctrl+Q (30.0 s), the `sleep` lived on until the
    window was closed (39.9 s). With the same mock and no commit, plan 47's
    build and this one each quit and closed their tab at once.
  - The clipboard and this repository unchanged.
- 2026-10-25: the documents updated (ui-design.md: the card's format
  characters and the output not kept; events-and-state.md: the Session
  rules and the Tool call row; reliability.md: panics and bounds;
  architecture.md: plan 48's table; evaluation.md: `TestOutputToFrame` and
  Fuzzing; AGENTS.md: the timing command). `doc/plan48-evaluation.md`
  written: 13 PASS, 1 FAIL (48.5's prediction about today's code). Step 3
  done; the plan waits for the owner's review. Nothing is committed.

## Closed plans

One entry each: what the plan did, its approval, its evaluation file and
its commits. Their criteria and progress logs: `git show 8888f59:plan.md`
(plans 1 to 13), `git show 84d2ad8:plan.md` (plan 14), `git show
b047434:plan.md` (plan 15), `git show becfb34:plan.md` (plan 16), `git show
f00c2d8:plan.md` (plan 17), `git show 8b3b16b:plan.md` (plan 18), `git show
d84e5d4:plan.md` (plan 20), `git show f939d96:plan.md` (plan 21), `git show
d13ef71:plan.md` (plan 22), `git show e8d3e3b:plan.md` (plan 23), `git show
a3bfe5d:plan.md` (plan 24), `git show cb8f76d:plan.md` (plan 25), `git show
a9a5145:plan.md` (plan 26).

## Plan 1 - Research

The agent CLIs and how to drive them, telemetry, terminal input, the
reference projects, and the decision to build rather than fork
(`doc/research/`). Accepted on 2026-10-16, 9 PASS and criteria 8 and 9 kept as
FAIL, documented exceptions
([`doc/research/plan1-evaluation.md`](doc/research/plan1-evaluation.md));
commit 7e4d67c.

## Plan 2 - Architecture and reliability

`doc/architecture.md`, `doc/events-and-state.md`, `doc/ui-design.md` and
`doc/reliability.md`. Approved on 2026-10-16, 15 of 15 PASS
([`doc/plan2-evaluation.md`](doc/plan2-evaluation.md)); commit d5b910e.

## Plan 3 - Proposal

The MVP roadmap and the decisions it asked for (stack, dependencies, live
runs, configuration check). Approved on 2026-10-16, 12 of 12 PASS; commit
d5b910e, whose `doc/proposal.md` holds the roadmap. The design documents
and the code have replaced it since.

## Plan 4 - MVP (from amendment 4; steps detailed by plan 3)

The MVP, steps 0 to 8: startup, Claude Code, the streaming conversation,
approvals and resumption, Codex, agent activity, telemetry, the UI.
Approved on 2026-10-17 after a live demo on both CLIs, 14 of 14 PASS
([`doc/plan4-evaluation.md`](doc/plan4-evaluation.md)); commits ded53d8 to
cf58739.

## Plan 5 - Post-MVP improvements

The list plan 4 left open, Windows only. Approved on 2026-10-17, 17 of 17
PASS, then 15 of 15 live and 6 of 6 for the follow-up
([`doc/plan5-evaluation.md`](doc/plan5-evaluation.md)); commits d42531f,
897f640, 6813c0e.

## Plan 6 - Plan 5's known limits

Four known limits fixed, then items 5 and 6 added by the owner. Approved on
2026-10-17, 6 of 6 and 8 of 8 PASS
([`doc/plan6-evaluation.md`](doc/plan6-evaluation.md)); commits df273fe,
0814d67.

## Plan 7 - The open list after plan 6

The open list cleaned. Approved on 2026-10-17, 7 of 7 PASS
([`doc/plan7-evaluation.md`](doc/plan7-evaluation.md)); commit bffe9a5.

## Plan 8 - Codex's allow for the session

`acceptForSession` on Codex 0.163.0. Approved on 2026-10-17, 6 of 6 PASS
and its follow-up 3 of 3 ([`doc/plan8-evaluation.md`](doc/plan8-evaluation.md));
commits e75d591, 1daeec1.

## Plan 9 - Plan 8's items left not verified

Two items verified, the third recorded as not verifiable. Approved on
2026-10-18, 6 of 6 PASS
([`doc/plan9-evaluation.md`](doc/plan9-evaluation.md)); commits
c9576d9, 8dea129.

## Plan 10 - A spawned Codex agent's label and the timing step

Both items verified. Approved on 2026-10-18, 7 of 7 PASS
([`doc/plan10-evaluation.md`](doc/plan10-evaluation.md)); commits b71c114,
621609e.

## Plan 11 - A live approval offering the allow for the session

The live approval closed as not verifiable, by the owner's choice.
Approved on 2026-10-18, 3 of 3 PASS
([`doc/plan11-evaluation.md`](doc/plan11-evaluation.md)); commits 81f3bf8,
cb206b6.

## Plan 12 - Manual or Auto approvals

Brief amendment 8: a per-session switch between Manual and each CLI's own
Auto. Approved on 2026-10-18, 18 of 19 PASS and 1 FAIL accepted
([`doc/plan12-evaluation.md`](doc/plan12-evaluation.md)); commits cb31557,
83fa023.

## Plan 13 - A reviewer's refusal live

Claude Code's classifier denial seen live and a render defect it exposed
fixed; Codex's reviewer not reached, closed by the owner. Approved on
2026-10-18, 14 PASS and 3 not observed
([`doc/plan13-evaluation.md`](doc/plan13-evaluation.md));
commits d68bb55, 8888f59.

## Plan 14 - Nothing left open

Every open question of the research files settled or closed by the owner,
Claude Code's question card built and seen live, the documents reduced to
the latest state and a CLI update made visible. Approved on 2026-10-18, 18
PASS and 15 PASS ([`doc/plan14-evaluation.md`](doc/plan14-evaluation.md));
commits 0206ac1, ce06b97.

## Plan 15 - Recent projects, favourites and a project browser

Brief amendment 9: the startup screen's project list (`projects.json`,
favourites first) and its Ctrl+O browser. Approved on 2026-10-18, 21 PASS
([`doc/plan15-evaluation.md`](doc/plan15-evaluation.md)); commits bd9ffa3,
b047434.

## Plan 16 - Git worktree management

Brief amendment 10: the Worktrees view (Ctrl+W, at startup and in session)
with create, list, diff, merge and remove, git through `os/exec`. Approved
on 2026-10-18, 26 PASS
([`doc/plan16-evaluation.md`](doc/plan16-evaluation.md)); commits ce95ba2,
fad6d20, 642c5b9, becfb34.

## Plan 17 - A file viewer in the terminal

Brief amendments 11, 12 and 13: a file viewer in the terminal with Chroma's
highlighting, an Explorer of the project (F3) and the draft in a window
(Ctrl+G), with no external editor. Approved on 2026-10-19, 91 PASS
([`doc/plan17-evaluation.md`](doc/plan17-evaluation.md)); commits 1748808,
10c3e74, 904f9a6, 749cef4.

## Plan 18 - An adaptive layout

Brief amendment 14: zone C sized by the agents in play, a strip of 4 rows (3
under 100 columns) when there are none, and cleared at each new prompt.
Approved on 2026-10-19, 22 PASS
([`doc/plan18-evaluation.md`](doc/plan18-evaluation.md)); commit 8b3b16b.

## Plan 19 - Zone B as the centerpiece

Brief amendment 15: one speaker label per exchange, RESULT over an
exchange's outcome, compact tool rows and folds of long runs of calls, each
entry's turn (`Entry.Stage`). Done on 2026-10-20, 19 PASS
([`doc/plan19-evaluation.md`](doc/plan19-evaluation.md)). The owner
reviewed it in a demo the same day and asked for plan 20 (amendment 16),
which keeps its logic and replaces its look; not committed on its own, it
was committed with plan 20 once the owner approved that, on 2026-10-20;
commit d84e5d4.

## Plan 20 - Strict conformance to the approved templates

Brief amendments 16 and 17: the owner's two templates as one design
system, A and B in one upper row and C full width under them, zone B's
messages as blocks with a rule between them, the tool rows, the folds, the
pending approval's pink box and RESULT. Approved on 2026-10-20 after the
owner's live review, 32 of 32 PASS
([`doc/plan20-evaluation.md`](doc/plan20-evaluation.md)); commit d84e5d4,
with plan 19.

## Plan 21 - What the review of d84e5d4 found

A read-only review of commit d84e5d4: zone D's session time past 2,000
events, two timed tests outside the timing step, Claude Code's
undocumented `api_message_id` labelled with a fallback, zone C's floor at
160x24 to 160x26, and documents promising what the code did not do (the
send and newline keys built in `config.json`, the others corrected).
Approved on 2026-10-20 with the recommended choices, 38 PASS and 2 FAIL,
then the long session's frame time fixed at the owner's request, 6 PASS
([`doc/plan21-evaluation.md`](doc/plan21-evaluation.md)); commits bdc4250,
f939d96.

## Plan 22 - A live thinking indicator

Brief amendments 18 to 20: zone B's live thinking card for Claude Code and
Codex, with its animations, a summary row for each thinking phase that
opens on its details, and an activity line while a turn works; corrections
C1 to C12 from a read-only investigation
([`doc/plan22-investigation.md`](doc/plan22-investigation.md)); Codex
started through the local proxy that `config.json` names (amendment 19);
Codex's reasoning summaries asked for and Claude Code's hook events known
(amendment 20), checked live with one Codex turn (L20). Approved on
2026-10-21: 31 of 31 PASS, then the corrections 17 of 18 with 5.14 fixed,
amendment 20 13 of 13, and L20 8 of 8 once two criteria were corrected
([`doc/plan22-evaluation.md`](doc/plan22-evaluation.md)); commits
e6895c6, 6306bfb and d13ef71, then the approval commit.

## Plan 23 - The acc launcher in the project

Brief amendment 21: `acc`, typed in Windows Terminal, opens the dashboard
in a tab of its own at the owner's size, a 9 pt font. `acc.cmd` at the root
runs `scripts\acc.ps1`, which writes a Terminal fragment for the profile
and makes Terminal reload. The day's machine-local changes were undone.
Approved on 2026-10-22, 6 of 6 PASS before the commit, then committed and
pushed ([`doc/plan23-evaluation.md`](doc/plan23-evaluation.md)); commit
e8d3e3b, then the approval commit.

## Plan 24 - Zone A with the telemetry, zone D removed

Brief amendment 22: zone A redrawn after the owner's image, holding the
orchestrator's dynamic status, the stages as blocks and zone D's telemetry
(session usage, quota windows with lavender bars and countdowns, the notes,
the details view); zone D removed; Fable and Mythos as model families.
Amendment 23, after the owner's live session: the Claude adapter reads the
`/usage` report after the handshake and after each turn, so the
model-scoped "Fable limit" shows live, the exchange kept out of the
conversation. Approved on 2026-10-22: 35 of 35 PASS with the conformance
table 22 rows PASS, amendment 23 6 of 6 and its live check 8 of 8
([`doc/plan24-evaluation.md`](doc/plan24-evaluation.md)); commits 6209267
and a3bfe5d, then the approval commit.

## Plan 25 - The unrecognised lines named

Brief amendment 24: the owner's scenario 1 run on Fable 5.1 ended with
"unrecognised 2" and no way to see which lines. The details view now lists
the last 20 unrecognised lines with their time, type, reason and text; a
replay tool names them from a logging wrapper's record; the owner's two,
named by a logged rerun, were `system/task_summary`, known with no event
since. Approved on 2026-10-22: 7 of 7 PASS, two live runs
([`doc/plan25-evaluation.md`](doc/plan25-evaluation.md)); commit cb8f76d,
then the approval commit.

## Plan 26 - Zone A without notes

Brief amendment 25: the owner asked to get rid of zone A's notes
("unrecognised 1", "quota updated", "† undocumented field", "cost at API
rates") completely and forever. The telemetry pane now ends with the last
quota window's rows; the counts, ages and legends live in the details view,
where an undocumented value keeps its "†" and the heading explains it; a
standing decision. Approved on 2026-10-22: 6 of 6 PASS, 51 golden frames
reviewed ([`doc/plan26-evaluation.md`](doc/plan26-evaluation.md)); commit
a9a5145, then the approval commit.

## Plan 27 - Zone C with the execution timeline

Brief amendments 26 and 27: the owner's design of zone C and its
close-ups, to respect strictly. Zone C from 100 columns: the header
inside the frame, a rule and the panels' line in the title colour, the
agents' panel as wide as its two-column cards (the label over the clock
and the tokens, a grey bar, the model over the current tool; the selected
agent's rows on a band) and the selected agent's execution timeline with
its status at the heading's right end; the narrow layout keeps plan 18's
row. Approved on 2026-10-22: 15 of 16 PASS, 27.6 corrected with a note
at the owner's decision, 46 golden frames, Windows Terminal screenshots
at 208x59 and 120x40 against the images
([`doc/plan27-evaluation.md`](doc/plan27-evaluation.md)); commit 041e7b8,
then the approval commit.

## Plan 28 - The top right after the owner's image

Brief amendment 28: the owner's close-up of the top right, "● LIVE" over
"Claude Code 2.1.303" over zone B's edge with "Idle". The project bar's
right end is the provider's name and version alone while approvals are
Manual with no change asked; the mode shows there only as Auto or while
changing. Approved on 2026-10-22: 6 of 6 PASS, 27 golden frames, a
Windows Terminal screenshot against the image
([`doc/plan28-evaluation.md`](doc/plan28-evaluation.md)); commit 8fe0077,
then the approval commit.

## Plan 29 - The input editor's row after the owner's image

Brief amendment 29: the owner's image of the editor's row, design only.
Under the draft a separator, then " + " and the pill " <model> High ·
Ultracode " on the tool rows' ground, the editor's messages and the line
count in the free middle, "↯ Auto" at the right end; the "+", "High ·
Ultracode" and "Auto" are placeholders with no function. Zone B is one
row shorter. Approved on 2026-10-22: 7 of 7 PASS, 59 golden frames,
Windows Terminal screenshots at 208x59 and 120x40 against the image
([`doc/plan29-evaluation.md`](doc/plan29-evaluation.md)); commit d696f22,
then the approval commit.

## Plan 30 - The "+" menu after the owner's image

Brief amendment 30: the owner's image of the "+" clicked, design first. A
click on the editor row's band opens a three-row box above the editor's
rule, "🗀  Upload from computer", the band and the item under the
pointer on the active ground (`#272745`), the terminal asked for every
mouse motion only while it is open; Esc, any key or a click elsewhere
closes it; the item does nothing yet, the upload being a later plan's.
Approved on 2026-10-22: 6 of 6 PASS, four golden frames added, Windows
Terminal screenshots with a real click and pointer moves at 208x59 and
120x40 against the image
([`doc/plan30-evaluation.md`](doc/plan30-evaluation.md)); commit 6025513,
then the approval commit.

## Plan 31 - Image upload

Brief amendment 31: the "+" menu's "Upload from computer" works. The
system's Open dialog, several images at once, each checked by its bytes
(PNG, JPEG, GIF, BMP, TIFF, ICO and WebP accepted, the orientation
applied, conversions to PNG, an image over a provider's limits scaled
down and said so; HEIC, AVIF, JPEG XL, JPEG 2000, PSD, TGA, DDS, HDR,
EXR, camera raws and SVG refused by name); chips under the editor's
rule, removable; sent only when the owner submits, as base64 image
blocks to Claude Code and `image` inputs holding data URLs to Codex, no
event keeping the bytes (Codex writes them back). Approved on
2026-10-23: 9 of 9 PASS, eight golden frames added, the real dialog in
Windows Terminal, live turns on Claude Code (Haiku 5.5) and Codex
(gpt-6-sol) naming the colours and the text, and a third on Codex
checking the echo fix
([`doc/plan31-evaluation.md`](doc/plan31-evaluation.md)); commit 8e2a721,
then the approval commit.

## Plan 32 - Image paste and preview

Brief amendment 32: paste an image as in the owner's chat, and a preview
before sending. Alt+V attaches the clipboard's images (copied files, the
"PNG" format's bytes, else the bitmap as PNG); Ctrl+V pastes text as
before and, when Windows Terminal keeps its press for want of text,
attaches the image on the key's lone release (undocumented, probe P32a);
a paste of the copied files' paths attaches them when all are images;
a pasted image is held in memory only. With Sixel graphics, the images
show as thumbnails between the rule and the chips, rounded squares cut
from their centre, 14x7 cells from 50 rows, 10x5 from 30, side by side
one cell apart (the owner's remark on the screenshots), drawn after the
frame and again after any frame that changes their rows, a vacated
place written over with the frame's cells. Approved on 2026-10-23: 10 of
10 PASS, four golden frames added, Windows Terminal at 208x59 and
120x40 with the owner's clipboard saved and restored, the remark in a
follow-up ([`doc/plan32-evaluation.md`](doc/plan32-evaluation.md));
commit b56d23e, then the approval commit.

## Plan 33 - Unsent drafts and images through crashes and restarts

Brief amendment 33: whether unsent image attachments survive a crash, a
forced termination, a restart and a resumed session; an investigation
first, then its nine corrections. Each dashboard's draft has a folder of
its own, its lock held while the dashboard owns it, taken back by the
project's next start (its own CLI's first, then the newest); images are
saved at once, the text every 500 ms; pasted images are kept as
`<sha256>.png` until their message is accepted; a message waiting for
its acceptance is stored with what was typed since; Back forgets the
attachments; a damaged list is set aside; a changed file is attached
again as it is now; refused renames and failed saves are tried again;
unused files are swept at each start. An independent review found five
defects, fixed; on the owner's questions, a Codex session resumed with
images, a slow and an offline share were simulated, fixed where they
lost data, and a logoff was not simulated. Approved on 2026-10-23: 14 of
14 PASS and the follow-up
([`doc/plan33-evaluation.md`](doc/plan33-evaluation.md)); commit
bc5d92b, then the approval commit.

## Plan 34 - The model selector, its effort and Ultracode

Brief amendment 34: a click on the editor row's model pill, or Alt+P,
opens the active CLI's own models with the effort and, for Claude Code,
Ultracode, after the owner's image. Claude Code's list comes from
`initialize`, its changes go by `set_model` and `apply_flag_settings`
and are read back by the undocumented `get_settings`; Codex's list from
every page of `model/list`, its changes ride the next `turn/start`,
read back by `thread/read`. The screen shows a value only once the CLI
reports it in force; an effort held by `CLAUDE_CODE_EFFORT_LEVEL` is
shown and never changed; Claude Code's changes wait for a turn's end. An
independent review found four defects and six doubts, fixed, and three
small points on its second pass. At the owner's request
`CLAUDE_CODE_EFFORT_LEVEL` was removed from their Claude Code settings,
and the dashboard then set the effort live; amendment 35, an opt-in line
telling the agent its settings, was built and withdrawn. Approved on
2026-10-24: 20 of 21 PASS, 34.18 (golden frames changed beyond the
pill's values) accepted
([`doc/plan34-evaluation.md`](doc/plan34-evaluation.md)); commit
47b50ce, then the approval commit.

## Plan 35 - The Ultracode toggle after the owner's image

Brief amendment 36: the owner's image of the Ultracode toggle, a lavender
pill with fully rounded ends and a round white knob, against the
dashboard's `▐   ●▌`. Where the terminal draws Sixel, the toggle is an
image over blank cells of its row, drawn as the preview's thumbnails are:
a pill of 40x20 pixels in an image of five cells whose last carries the
band, since the renderer writes every cell between a row's first and last
changes and a band cell after the image made it blink when the band came
or went (bursts of captures in Windows Terminal); without Sixel, a text
pill of Powerline's half circles. Approved on 2026-10-24: 8 of 8 PASS
([`doc/plan35-evaluation.md`](doc/plan35-evaluation.md)); commit
b43cf46, then the approval commit.

## Plan 36 - The selector's cursor and a pointer that has not moved

Brief amendment 37: Alt+P could reopen the model selector with its
cursor on the row under a resting pointer, so that Enter acted on that
row. The probe (Bubble Tea's `TEA_TRACE`) showed Windows Terminal
reporting a pointer put down by software, without its moving, once the
selector asked for every motion, its reports wandering a cell. The
selector now moves its cursor only when the pointer comes onto another
row than the last report's, the first report after Alt+P only recording
where it rests. Approved on 2026-10-24: 7 of 7 PASS
([`doc/plan36-evaluation.md`](doc/plan36-evaluation.md)); commit
c0cebc4, then the approval commit.

## Plan 37 - The effort slider after the owner's images

Brief amendment 38: the owner's five images of one pill-shaped effort
slider at five levels replaced the selector's effort dropdown. Effort
reads "Effort (<level>)", "Controls reasoning depth" and the slider: the
model's own levels as dots, a fill of `#6784C1` growing to the thumb,
the whole pill lit by a purple gradient with specks at the highest
level, a level asked for ringed, a held effort greyed; with Sixel an
image of 170x20 pixels over 17 blank cells, its thumb gliding in 180 ms;
without, 16 text cells. Effort and Ultracode share label, description
and control columns; ← → and a click on the slider ask for a level.
Approved on 2026-10-24: 9 of 9 PASS
([`doc/plan37-evaluation.md`](doc/plan37-evaluation.md)); commit
31c22ef, then the approval commit.

## Plan 38 - Instant Codex model and effort changes, as in VS Code

Brief amendment 39: a Codex model or effort change took effect only with
the next message, where VS Code's Codex extension changes it at once. The
dashboard now asks for Codex's experimental API at initialize, as that
extension does, and writes a change between turns at once with
`thread/settings/update`, the `thread/settings/updated` that follows
putting it in force (P38x: 1 to 3 ms); a change asked during a turn is
written at its end; a Codex that refuses the update takes every change
with the next message, as before. Approved on 2026-10-24: 12 of 12 PASS
([`doc/plan38-evaluation.md`](doc/plan38-evaluation.md)); commit
dc9c6d9, then the approval commit.

## Plan 39 - The permission modes selector after the owner's images

Brief amendment 40: the editor row's right-hand "Auto" became the mode
pill of the owner's third image, and opens a "MODES" popup after the
first two: the active CLI's own modes (Claude Code's Manual, Edit
automatically, Plan and Auto; Codex's Ask for approval, Approve for me and
Full access), each applied through the CLI (`set_permission_mode`; the
experimental `thread/settings/update` with the presets of VS Code's Codex
extension) and shown in force only once the CLI reports it; Codex's Full
access only after a confirmation. The icons, the check and the band's
rounded ends are Sixel strips beside the text cells, the pill a Sixel
image whose title Windows' GDI draws. Approved on 2026-10-24: 15 of 15
PASS ([`doc/plan39-evaluation.md`](doc/plan39-evaluation.md)); commit
55e2bed, then the approval commit.

## Plan 40 - No empty rows in zone C's execution timeline

Brief amendment 41: on Codex, zone C's execution timeline and the
inspection's Messages tab showed a subagent's rows "assistant │" with no
text, one for each of its reasoning steps: since plan 22 a subagent's
reasoning summaries leave assistant entries holding only a thinking mark,
which zone B never drew. Neither draws an entry with no text now (a
thinking mark, a message that ended empty, a notice with none), a rule
only between two drawn entries. Approved on 2026-10-24: 6 of 6 PASS
([`doc/plan40-evaluation.md`](doc/plan40-evaluation.md)); commit
4158751, then the approval commit.

## Plan 41 - No authorization on low-risk review rows

Brief amendment 42: a Codex auto-review row rated low risk read
"authorization unknown": Codex's reviewer answers a low-risk action with
a bare allow, and Codex then reports the authorization as unknown with a
stock rationale. After a preview, the owner chose to drop the
authorization from review rows rated low risk, in zone B, zone C's
timeline and the inspection; at medium, high and critical it stays.
Approved on 2026-10-24: 7 of 7 PASS
([`doc/plan41-evaluation.md`](doc/plan41-evaluation.md)); commit
3904f13, then the approval commit.

## Plan 42 - The mouse in zone B: a click to fold, a double click to open

Brief amendment 43: beside the keyboard's path, which the owner approved
and which stays as it is, a click anywhere on a foldable item's row in
zone B (a run of calls, a thinking phase's summary row) unfolds it or
folds it again, and a double click on a row of an entry Enter shows whole
opens its full view as Enter does, an approval's rows the approval whole
as from its card. The mouse puts the entry cursor nowhere and shows none
of the keyboard's hints. On those rows a double click no longer selects a
word; a drag still does. Approved on 2026-10-24: 7 of 7 PASS
([`doc/plan42-evaluation.md`](doc/plan42-evaluation.md)); commit
11f794d, then the approval commit.

## Plan 43 - A right activity sidebar with the Explorer docked

Brief amendment 44: a narrow activity sidebar right of zone B, from 100
columns, with the Explorer's and Skills' icons after the owner's two
images. The Explorer, a floating window since plan 17, is a panel docked
between zone B and the sidebar where zone B keeps 64 columns, and in zone
B's place where it does not and under 100 columns; F3 and the Explorer's
icon show and hide that one panel, the focus with it, F3 keeping its file
meaning. The icons' four looks are Sixel images, their hover from
every motion reported, filtered before Update; Skills is a placeholder.
The review led to a third patch of the vendored ultraviolet: a row
holding a wide character is written only as far as it changed, so the
icons beside it stay. Approved on 2026-10-25: 15 of 15 PASS
([`doc/plan43-evaluation.md`](doc/plan43-evaluation.md)); commit
48bde52, then the approval commit.

## Plan 44 - Source Control and a Git diff viewer

Brief amendment 45: the sidebar's second icon, Skills' placeholder until
then, is Source Control. Its panel shares the Explorer's docked place and
lists the project's Git changes from one git status read, the Explorer's
too: MERGE CHANGES, CHANGES and STAGED CHANGES, each file with its real
status, a file staged and changed again in both. A file opens its diff in
zone B's place, the panel beside it from 118 columns: git's blobs read raw
by cat-file and the working file, compared by a Myers diff of our own,
unified or side by side, with line numbers, green and red rows and
navigation between changes; read only. The engine was built by an
implementer subagent. Approved on 2026-10-25: 14 of 14 PASS
([`doc/plan44-evaluation.md`](doc/plan44-evaluation.md)); commit
d9d7d83, then the approval commit.

## Plan 45 - Commit, Push and Commit & Push in Source Control

Brief amendment 46: under Source Control's repository row, a message
field, a purple "✓ Commit" button with a separator and a chevron, and a
dropdown with one item, Commit & Push, after the owner's image. Commit
takes the index as staged (asking before staging anything), refuses a
staged set other than the one shown and counts once HEAD is a new commit
on the old one; Push goes to the configured upstream only, never forced;
the main action is read from git's branch line. After the first report
the owner asked for two additions: the staged files' contents compared
too, and a long operation's elapsed time with Esc cancelling a push (a
Ctrl+Break to git's own process group, behind a check of its lock files;
probes P45a to P45e). The engine's first version was built by an
implementer subagent. Approved on 2026-10-25: 19 of 19 PASS
([`doc/plan45-evaluation.md`](doc/plan45-evaluation.md)); commit
f71e022, then the approval commit.

## Plan 46 - Three fixes from the audit

Brief amendment 47: the first three fixes of the read-only audit of
2026-10-25 ([`doc/audit.md`](doc/audit.md)), each test-first.
- The worktree merge reads the main worktree first, refuses a merge in
  progress or uncommitted changes, and aborts only its own conflict
  (MERGE_HEAD the branch's tip).
- A refused Codex `config/read` or `model/list` no longer ends the
  handshake (`event.Problem.Optional`).
- An opened untracked folder in Source Control is read again after each
  status read and each opening.

Approved on 2026-10-25: 10 of 10 PASS
([`doc/plan46-evaluation.md`](doc/plan46-evaluation.md)); commit 9375233,
then the approval commit.

## Plan 47 - The rest of the audit's phase 1

Brief amendment 48: the rest of phase 1 of the audit's roadmap
([`doc/audit.md`](doc/audit.md), Part 2, C), each fix test-first.
- Ctrl+Q while a commit, push or staging runs asks to confirm, naming
  it.
- An answer to a request the CLI withdrew first expires the approval and
  the session goes on (`provider.ErrNoPending`).
- The reducer's Mode case returns, so mode events leave no false note,
  and a mock replay of Codex declines in Codex's word.
- The Explorer opened over the staging question drops it.
- `realPath` strips the `\\?\` prefix; two stale comments are corrected.
- `TestStateMachines`, failing since plan 46's commit, is repaired.

Approved on 2026-10-25: 10 of 10 PASS
([`doc/plan47-evaluation.md`](doc/plan47-evaluation.md)); commit d44bc0f,
then the approval commit.
