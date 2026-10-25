# Plan 48 evaluation (2026-10-25)

The check of plan 48, the audit's phase 2, robustness at the boundaries
([`audit.md`](audit.md), Part 2, C; brief amendment 49):
- finding 2: the TIFF and ICO decoders' unbounded allocations;
- finding 7a: a running command's streamed output, kept whole;
- finding 9: the Claude Code reader writing to stdin itself;
- finding 3c: git calls with no bound on a child that keeps their pipes;
- finding 6: panics outside Bubble Tea's goroutines, and no fuzzing;
- finding 8: format characters on the approval card.

It covers the 14 criteria of [`plan.md`, Plan 48](../plan.md), written
before the code. Each fix was test-first, except where said: its test was
written and seen failing on the old code, then the change made it pass.
Nothing is committed: the owner reviews first. No git command touched this
repository; every git test ran on temporary repositories.

How it was checked:
- **Test-first, per fix**, each seen failing on the old code first:
  - `TestCraftedImagesBounded` (`internal/input`): 64.0 MiB allocated for
    each crafted TIFF, 77.3 MiB for the ICO;
  - `TestLiveOutputTail` (`internal/state`): all 204,800 bytes kept;
  - `TestReaderNeverWaitsOnStdin` (`internal/provider/claude`): "stdout
    line 2 not read within 2 s";
  - `TestGitPipesBounded` (`internal/input`): 15.1 s for each of a read, a
    write and a stream;
  - `TestWritePanic` and `TestDetectPanic` (`internal/app`): the test
    binary killed by each panic;
  - `TestApprovalRevealsFormat` (`internal/render`): the raw characters
    drawn.
- **Written with the code, their mutants showing they fail without it:**
  `TestOutputCutNote`, `TestReaderPanic` in each adapter, `TestRecovered`,
  and the duplicate-tag case of `TestCraftedImagesBounded`.
- **The documented state machines:** each new rule has its row in
  `events-and-state.md` and its case in `TestStateMachines`.
- **Each fix broken on purpose** through `go test -overlay`, nothing
  written in the repository (`scratchpad\plan48\mutate48.mjs`).
- **Fuzzing** of the image readers and both mappers, 60 s each.
- **Build, vet, gofmt, the whole suite** after the last document edit, the
  race detector on the touched packages, and the timing step alone, with
  the new large-output test also run against the old append.
- **Windows Terminal in mock mode** on a temporary repository whose
  pre-commit hook leaves a background process on git's output
  (`scratchpad\plan48\mkrepo48.mjs`, `mkmock48.mjs`,
  `scratchpad\term\p46.ps1`, `p48call.ps1`).

```text
Evaluation - Plan 48, the audit's phase 2, robustness at the boundaries - 14 criteria: 13 PASS, 1 FAIL
PASS 48.1 TestCraftedImagesBounded: 1 MiB TIFFs with 16 entries of an unused tag, 16 strip offsets, or four widths after the first, 1,048,576 values each over the same bytes: each refused ("TIFF: photometric 2 with 1 samples: not read", the first width kept), 0.00 MiB allocated; the test TIFF still decodes (TestImageInspect). On the old code: 64.0 MiB allocated for each 16-entry TIFF
PASS 48.2 the same test: an ICO whose PNG declares 9000x9000 px is refused ("ICO: 9000x9000 px: over 60 megapixels"), 0.00 MiB allocated; the test ICO still decodes. On the old code: 77.3 MiB (81.0 MB) allocated, then "ICO: its PNG: png: invalid format: not enough pixel data"
PASS 48.3 FuzzInspectData, 60 s with -fuzzminimizetime 2s: 243,348 inputs, no crash, no input kept
PASS 48.4 TestLiveOutputTail: 200 deltas of 1 KiB: 65,536 bytes kept, from a line's start, ending with the last delta, 139,264 counted as dropped (204,800 in all); the completion's output replaces it and the count is 0; the cut falls on a line's start as it is, else at the next line, else at a character's start (three-byte characters). TestOutputCutNote: "→ … 139 KB of earlier output not kept" in the inspection and "… 139 KB of earlier output not kept" in the full view, before the tail; nothing once the count is 0. On the old code: all 204,800 bytes kept
FAIL 48.5 TestOutputToFrame in the timing step: 64 MiB in 8,192 deltas of 8 KiB, p95 1.04 ms (target 16 ms), max 11.8 ms, then the completion's output is the call's: what the criterion asks of the new code holds. Its last bullet does not: on today's code (the old append, by an overlay of reduce.go) the p95 is 7.42 ms (p50 4.0 ms, max 16.3 ms), not over 16 ms as predicted from the audit's 17 ms late deltas; the old code fails the test only by its output check (all 64 MiB kept)
PASS 48.6 TestReaderNeverWaitsOnStdin, Claude Code on in-memory pipes: stdin unread after initialize, the handshake's answer and three more stdout lines are each read within 2 s; once stdin is read again, get_settings then "/usage" arrive, the events initialize, ready, get_settings, usage in that order; a set_model answered while stdin is unread, two more lines are read, then the settings read arrives; 20 runs in a row; the Claude Code suite passes unchanged. On the old code: "stdout line 2 not read within 2 s"
PASS 48.7 TestGitPipesBounded: a fake git (this test binary) exits 0 while its child keeps stdout and stderr 15 s: a read 2.11 s, a write (Stage's exec) 2.09 s, a stream 2.08 s, each with git's output and no error; the input package passes whole, with real git. On the old code: 15.1 s each, the read and the stream ending in "git … took more than 15s", the read's output lost
PASS 48.8 TestWritePanic: a scripted Send panicking: Crashed, the card "The session ended unexpectedly" and "writing to the CLI failed: panic: the adapter broke (at …)", one Close, "hello" back in the editor, the program running. TestDetectPanic: Codex listed with "detection failed: panic: detection broke (at …)", the startup screen asking. On the old code each panic killed the test binary
PASS 48.9 TestReaderPanic in each adapter, a nil mapper: read returns, its fatal Error "reading Claude Code's output failed: panic: runtime error: invalid memory address or nil pointer dereference (at claude.(*Mapper)…)" (Codex's likewise), the rest of its input read, both mutexes free; a function run apart that panics gives "… failed: panic: boom (at …)". TestStateMachines: a fatal error or Panicked crashes a spawning, handshaking or ready session with its message and a close; while Closing it is kept and the quit ends; after a crash the first reason stays. Written with the code: its mutants fail it (the panic escaping, 184,501 and 124,504 bytes left unread, a deadlock ended by the timeout, the fatal mark missing)
PASS 48.10 FuzzMapperOut, 60 s each with -fuzzminimizetime 2s: 762,522 Claude Code scripts, 733,327 Codex scripts, no crash, no input kept
PASS 48.11 TestApprovalRevealsFormat: the card shows "echo safe<U+202E>txt.exe<U+200B>" and its summary "lists the files<U+200B> of the" with an emoji family joined by U+200D drawn whole; the full view the same; a question's card "Which one<U+202E>?" and "this<U+200B> — the first"; no raw U+202E or U+200B in any of them; the assistant message drawn with its raw characters, as before. On the old code: the raw characters on the card and in the full view
PASS 48.12 Build, vet, gofmt clean; go test ./... ok, 16 packages, 814 tests (799 and the 15 new, 5 of them skipped as before), run again after the last document edit; no file under testdata changed (0 golden frames; the probe recordings and mock files as they were); -race ok on input (187 s), app (289 s), state (13 s), render (79 s) and the five provider packages, 0 data races; the timing step alone, twelve tests within their targets, on mains power with the CPU 24% busy just before: TestOutputToFrame p95 1.04 ms, TestMessageToFrame p95 1.04 ms at 120x40 and 1.55 ms at 208x59, the highest p95 4.86 ms (TestLongSession's end), the 50,000-line paste 1.3 ms at worst, 800 of 800 idle motion reports dropped in each case
PASS 48.13 Mutations through go test -overlay: 44 of 44 caught, each fix broken: the old TIFF and ICO readers, the PNG size unchecked, later tag entries read; the whole output kept, the last cut only, the count kept, each of the cut's three rules, both notes; the reader's two writes inline; no WaitDelay, and ErrWaitDelay a failure in a read, a stream and a write; the write worker and the detection recovering nothing, the fatal error and Panicked ignored, a fatal error crashing while Closing, the CLI left open; in each adapter the reader recovering and draining nothing, the mapping lock not released by defer, apart recovering nothing, the fatal mark missing; the panic's place unnamed; nothing revealed, the joiner hidden, U+202E left out, the card's target and summary, the full view's target and a question's options raw, revealed everywhere. The first run caught 41; two test gaps and one mutant were corrected, then those three and the other three of that test were run again: all caught. The Go files were the same before and after the run
PASS 48.14 Windows Terminal, the final build (scratchpad\term\acc-p48.exe, SHA-256 69113fad74923990…, the tree's own), maximised in the acc tab's profile (208x59), then 120x40, mock mode with a replay ending on a pending request whose command holds U+202E and U+200B, on a temporary repository whose pre-commit hook leaves "sleep 30 &" on git's output: the card read "echo safe<U+202E>txt.exe<U+200B> && ls -p …", no raw U+202E on the screen; Commit showed "Committed ba93b6f." and "Committed 4abf490." about 3 s after Ctrl+Enter (23.0 to 26.1 s and 22.6 to 25.9 s, with the harness's 0.8 s wait), git log "hook test" on "Initial commit", no index.lock; with real git and the same hook the old Wait took 30.4 s, the new one 2.33 s (probe); no terminal-hosted agent session was open; this repository's status, diff, HEAD and origin/main the same before and after; the clipboard unchanged
```

Changed from the design:
- **48.5's prediction about today's code did not hold.** The criterion
  said the old code's p95 would be over 16 ms with 64 MiB, from the
  audit's late deltas of about 17 ms at 32 MiB. On this machine the old
  append gave a p95 of 7.42 ms: the quadratic cost is real (p50 4.0 ms
  against 0, max 16.3 ms), but not over the target at the 95th percentile.
  The timing test fails on the old code only by its output check;
  `TestLiveOutputTail` is the test that pins the fix.
- **Criterion 48.1's expected refusal** was "no strips"; the crafted TIFFs
  meet the reader's photometric check first. Corrected in the plan before
  the fix.
- **Tests grew before the mutations,** so that a mutant could be caught at
  all: the read after a settings change in `TestReaderNeverWaitsOnStdin`,
  a TIFF whose later widths must not replace the first, a question and a
  summary in `TestApprovalRevealsFormat`; after the first mutation run,
  `liveTail`'s edge cases (a second line, three-byte characters).
- **D6 reaches a tool call's full view too:** it is drawn by the same
  function as a request's (`viewerBodyFor`), the same input, so the same
  safety reading.
- **The adapters' panic tests came with the code**, not before it; their
  mutants show they fail without it.

Observed and kept:
- **Go's fuzzing minimises each new interesting input for up to 60 s by
  default**, running nothing else: the image fuzzer sat at 0 inputs a
  second. With `-fuzzminimizetime 2s` it ran thousands a second; the
  evaluation guide's Fuzzing section says so.
- **A hook's background process keeps its Windows Terminal tab open**
  after the dashboard quits: the `sleep` holds the console. Sampled, the
  dashboard left about 1 s after Ctrl+Q; plan 47's build and this one, with
  no commit, each closed their tab at once.
- **Copying from a card copies the markers**, `<U+202E>`, not the
  characters: what is drawn.
- **Once, the fuzz build failed in Go's own linker** (a panic in
  `cmd/link` while other builds ran); the same command built and ran at
  once after.

Not verified:
- **A real panic** in a mapper, a write or a detection: none is known. The
  recover paths are tested with injected panics (a nil mapper, a scripted
  `Send`, a panicking `Detect`), and the fuzzers found none.
- **The stdin deadlock with the real Claude Code**, which would need a CLI
  that stops reading its stdin while its stdout is full: tested on
  in-memory pipes.
- **A live Codex command streaming over 64 KiB**: the cap is tested through
  the reducer and the application with scripted events.
- **A crafted image through the Open dialog or a paste in the real
  terminal**: the readers are tested directly through `InspectData`.
- **Format characters from a live CLI's request**, and in a question in the
  real terminal: a mock replay carried them to the card; questions only in
  tests.

Limitations:
- An emoji sequence's width is still counted per code point (finding 5,
  phase 4): its row can be misaligned. Its characters are kept.
- An animated GIF's frames are still all decoded (optional, phase 4).
- The fuzzers ran 60 s each: a sample of their inputs, not a proof.
- Once git has ended, a call waits 2 s for its pipes; what a child of git
  writes after that is not read.
- While a command runs, only the last 64 KiB of its output can be seen;
  the whole output comes with its completion.
