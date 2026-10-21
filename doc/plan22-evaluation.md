# Plan 22 evaluation (2026-10-21)

The final check of plan 22, a live thinking indicator in zone B (brief
amendment 18). It checks the criteria of
[`plan.md`, Plan 22](../plan.md#plan-22---a-live-thinking-indicator), written
before the code. The evidence is the tests (each new check also run under a
mutation that makes it fail), the golden frames, an independent
verification and two reviews, the timing step, and mock replays of real
recordings in Windows Terminal. The block below is the summary reported to
the owner. The results are not to be rewritten.

## The independent verification, before the last fixes

Three agents read the change on 2026-10-21, read only (`scratchpad\p22\plan22-verify.js`):
a verifier checking every criterion against its value (Sonnet), a review
of the logic and one of what the screen claims (Opus; the second stopped
on `ERR_PROXY_TUNNEL`, the corporate proxy refusing a tunnel, and was not
run again). The verifier: 31 criteria, 28 PASS, 1 FAIL (4.5: this file
not yet written, ui-design.md's exit stages and the failed summary row out
of step with the code, criterion 3.2 naming test 11 for test 9), 2 not
verified (4.2 and 4.4, this step's), with 50 mutations of its own, 47
caught. The logic review: 9 findings, each reproduced by a throwaway test:

| Finding | Fixed by |
| - | - |
| 1 A Codex usage update repeated on an interrupt (0.145.0, P6b line 87) taken as a step's count, which could count a step twice | the adapter reads a total equal to the thread's last as no model response: no count, no step (TestRepeatedUsageCountsNothing; TestThinkingCodexP6b's last thought now has none) |
| 2 A Claude Code block whose stop never comes stays live, beside the next one | it ends with its message's `message_delta` or `message_stop`, or when the next message starts, its end inferred (TestThinkingBlockWithoutStop) |
| 3 A phase opened after a crash never ends: frames every 50 ms for good | no phase opens once the session crashed (TestThinkingAfterCrash) |
| 4 Zone B's keys for a phase were its Seq, which starts again after a resume | `Thought.ID`, numbered by `State.Thoughts` across a resume (TestThinkingIDs, TestThinkingReviewFixes) |
| 5 A phase ended by its turn, a crash or a force stop described as ended by itself; a Codex one losing "up to"; a count that can no longer come "not reported yet" | `Thought.EndInferred`, the details say so; "after up to"; "no exact count reported" (TestThinkingReviewFixes) |
| 6 "· N waiting for you" on the activity line could never show | removed: one waiting makes the turn wait, and the line is off |
| 7 The summary row moved up a row when the exit ended | from End+200 ms the summary row in its own place, settling until End+300 (`thinkSettle`; TestThinkingExit, TestThinkingReviewFixes) |
| 8 The placeholder read "Ask … anything…" while a request waited; "↓ N new" counted steps that left no row | the placeholder for any turn under way; `newShown` leaves those steps out |

Also from the verifier: the 80x24 Codex card cut its count text (the 2-row
card now takes a short form when the long one would not fit); a Codex step
that reasoned still read "Model working" as it closed (now "Thinking");
no test asked for the 100 ms spinner of a turn with no card (added to
TestThinkingTicks, the verifier's mutation now caught). Earlier, here: a
card in its exit folded away in a run of calls, an activity line under a
closing card, a child's end on its parent counted as a call, a force stop
and the owner's close leaving a phase live (each with its test), and mock
replays at speed 0 whose batches, and so which phase a frame drew live,
changed from run to run (`mock.Provider.Start` now queues every event
first).

## The summary

```text
Evaluation - Plan 22, a live thinking indicator in zone B - 31 criteria: 31 PASS
PASS 1.1 Claude Code's markers: each content_block_start and _stop carries its index; each message_delta its thinking_tokens when present (P303b: 719, 281, 268); a line without the field none (TestP303bThinkingCounts, TestBlockMarkersCarryTheirIndex, TestMessageDeltaThinkingMatchesEveryLine over 18 recordings)
PASS 1.2 Codex's step markers: P7 5 at 3,380, 7,120, 9,632, 13,471 and 24,076 ms, none after a step that made no call; reasoning per update 83, 0, 8, 7, 0; a child's end on its parent no call; a repeated update no count and no step (TestP7StepStarted, TestP7ReasoningLast, TestSubAgentActivityIsACall, TestRepeatedUsageCountsNothing and 9 synthetic sequences)
PASS 1.3 P12c-block: 4 phases of 3,476, 13,902, 56 and 2,161 ms; the second's estimates 50 to 2,700 in order; exact counts 717, 2,852, 17, 354; P303b: 3 phases, 719, 281, 268, none for its subagent's 4 blocks (TestThinkingClaudeP12cBlock, TestThinkingClaudeP303b)
PASS 1.4 P303b's phases sum to 1,268, the result's total, which no phase takes; a lower, equal or homeless estimate changes nothing; two blocks in a message: the count on the second, shared 2, drawn "over 2 phases" (TestThinkingEstimateRules, TestThinkingMessageWithTwoBlocks, TestThinkingSummary)
PASS 1.5 P7: 3 thoughts "up to" 3,140, 3,743 and 1,440 ms, 83, 8 and 7 reasoning tokens; 1 step dropped (the last, a message); the same without the children's events (TestThinkingCodexP7)
PASS 1.6 P303a's cut message: interrupted, 0 as reported; a turn interrupted, a failed one, a crash, the owner's close, a force stop, an open step dropped; a block with no stop ended by its message; none opened after a crash (TestThinkingClaudeP303a, TestThinkingEnds, TestThinkingBlockWithoutStop, TestThinkingAfterCrash)
PASS 1.7 P12c-block applied as one update: at +0, +100 and +300 ms its 4 summary rows, no card text or expression, none marked live, Animating false (TestThinkingReplayInOneBatch; a markLive marking every phase fails it)
PASS 2.1 thinking-live-120x40: 7 rows, "00:03", "≈ 650 thinking tokens · live estimate", " 2k"; the fill 26 of 80 cells = 650/2,000 (the verifier's replay at 11,000 ms; TestThinkingCardModeB: 52 of 160 half cells from 10,910 ms)
PASS 2.2 The bar to half a cell; 1,950 reads 2k, 2,200 4k; a phase's label never lower; never full because it ended (2,700 of 4k: 108 of 160 half cells at its exit) (TestThinkingBarMath)
PASS 2.3 Budget 4,096 and 1,284: "Thinking · 1,284 / 4,096 tokens", the fill proportional; no code sets a budget (grep)
PASS 2.4 Mode C: "Thinking · token usage pending", "Model working · reasoning tokens reported when the step ends"; the segment's head at 0, 20, 41, 62 for 0, 0.4, 0.8, 1.2 s; no digit (TestThinkingModeC)
PASS 2.5 Growth 300 to 650 at T: 24 half cells at T, 48 at T+150, 52 from T+300; the edge lit to T+299, mid to T+699; the same fill between two values (TestThinkingGrowth)
PASS 2.6 Frames every 10 ms from T-200 to T+1000: only "≈ 300" or "≈ 650", lit for 400 ms
PASS 2.7 ◜◠◝◞◡◟, 100 ms each, repeating; a still ◌ while a request waits (TestThinkingSpinner)
PASS 2.8 The owner's ten word for word, a new one every 4 s, never twice in a row for 10 seeds over 200 s, dim 200 ms (TestThinkingExpressions)
PASS 2.9 Entrance 0 to 280 ms at 7, 5 and 2 rows, the rows reserved from the first frame (TestThinkingEntrance)
PASS 2.10 Exit: ✓ and the duration at End, 7 then 3 rows to End+200, the summary row in its place from End+200, settled at End+300, never moving; a dropped step 7, 2, 1, 0; none for a phase not drawn live (TestThinkingExit, TestThinkingReviewFixes)
PASS 2.11 Word for word: "Thought for 2.8s · 719 thinking tokens", "… · ≈ 650 thinking tokens (estimate)", "Thought for 0.1s", "Thought for up to 3.1s · 83 reasoning tokens", "Thinking interrupted after 0.2s · 0 thinking tokens", and the failed and the Codex "after up to" forms (TestThinkingSummary, TestThinkingReviewFixes)
PASS 2.12 → 4 detail rows, ← folds, Enter shows it whole; the cursor never on a card; plan 19's fold tests unchanged (TestThinkingDetails, TestThinkingKeys, TestToolFold, TestToolFoldKeys)
PASS 2.13 The activity line with "◇ 1 subagent working" in P303b from 8,000 to 19,000 ms (the verifier's replay); none while the approval waits, while text streams, under a closing card, after the turn (TestThinkingActivity)
PASS 2.14 7, 5 and 2 rows by room, no row wider than its zone at 80, 120 and 200 columns; resizes through the application equal the goldens byte for byte (TestThinkingSizes; the verifier)
PASS 2.15 50 frames 50 ms apart with B held: Focus, ConvTop, ConvHold and Entry unchanged, the card live (TestThinkingHoldsView)
PASS 3.1 One 50 ms frame asked for at a time while a card lives (11 after 10 frames); animMsg leaves the spinner; a running turn with no card asks for the 100 ms spinner alone; none once idle (TestThinkingTicks; each clause fails under its mutation)
PASS 3.2 An idle dashboard asks for the 1 s clock alone (TestIdleDrawsNothing, reliability.md's test 9, added)
PASS 3.3 claude-thinking.jsonl (P12c-block) and claude-thinking-subagent.jsonl (P303b) by TestMockFiles; codex-subagents.jsonl replays its 3 thoughts
PASS 4.1 go build ./... and go vet ./... (and -tags timing) exit 0; gofmt -l . empty; go test ./... 14 packages ok, run twice, only TestDiskListLink skipped; go test -race ./... 14 ok, no race; go.mod and go.sum unchanged
PASS 4.2 The timing step 7 of 7 on mains power, CPU 6% before: TestThinkingFrame p50 1.13 ms, p95 3.33 ms, max 8.20 ms (5,522 entries, 721 phases); TestLongSession p95 4.52 ms; TestStreamToFrame p95 3.19 ms; the others within their targets
PASS 4.3 32 existing frames changed, each diff read: thought rows (16), the processing placeholder (2), thinking.estimate's † gone (2), a listing with the two new mock files or B behind a window a row on (12); 32 new frames, 8 scenarios at 4 sizes
PASS 4.4 Windows Terminal, the three mock replays maximised (1938x1098 px) then at 120x40: 26 screenshots, and 6 bursts of 12 to 30 frames 39 to 50 ms apart made into GIFs (entrance, pending, growth, exit, Codex's step and its end)
PASS 4.5 ui-design.md (Thinking, Animations, Rendering, Departures), events-and-state.md, architecture.md (ASM-18, ASM-22), reliability.md (the fallbacks, test 9, the mock files), claude-code.md and codex-cli.md (section 12); plan.md; this file; UTF-8 without BOM, LF only
PASS 4.6 git status lists only the intended files; no commit
Not verified: the look against the owner's three images, until the owner has seen the demonstrations; Claude Code with Opus or Sonnet (every recording runs Haiku); Codex with reasoning summaries asked for; a resume replaying Codex's usage (none recorded)
Verdict: DONE
```

## Limitations, as built

- **Codex reports no live reasoning** (codex-cli.md, section 12): its card
  is the model step's, "Model working", its bar indeterminate, and a
  thought's time is the step's, "up to". Asking for reasoning summaries at
  `turn/start` might make reasoning observable as it happens: proposed in
  plan.md, not tried.
- **Claude Code's live count is the CLI's estimate**, from 14% under to
  nearly four times the exact count; the exact count comes with its
  message's end, 0 to 2.4 s after the phase.
- **No thinking budget** from either CLI: mode A is tested, never shown.
- **Mock replays** wait 2 s at most between two events, so a replayed step
  can read shorter than recorded ("up to 2.0s" for P7's 3.1 s); at speed 0,
  in the golden frames, every event has one time ("Thought for 0.0s").
- The `testdata/probes/p12x-sandbox-codex-turn-workspace-write.jsonl`
  recording, from plan 12, holds the owner's Windows user name 23 times; no
  mock file comes from it; left for the owner to decide.

## The corrections C1 to C12 (2026-10-21)

The owner's decision on the investigation (`doc/plan22-investigation.md`):
"Apply the recommended corrections C1 to C12, with C1's wording (a) and
without the optional C13. Then re-verify, including the demonstrations the
corrections change." Its criteria, plan.md's 5.1 to 5.18, were written
before the code; four additions to C4, C7, C8 and C9 came from the review of
the first build, before their code.

How it was checked:
- An implementer agent (Sonnet, max effort) built it in two rounds, from
  the investigation's verified copies for C2 to C5. Each round was reviewed
  here, diff by diff, against a copy of the tree taken before the
  corrections.
- The tests: a `corrections_test.go` in `internal/state`, `internal/render`
  and `internal/app`. Six existing tests changed, each made stricter or
  given the new state, none loosened.
- A mutation run here, in a scratch copy: each fix undone alone, its tests
  run, the file restored byte for byte.
- The checks run here: the suite in the owner's environment and with the
  proxy variables unset, `-race`, the timing step alone, and the golden
  frames compared file by file with the copy.
- Windows Terminal: criterion 4.4's six runs again with the corrected build,
  moment for moment, plus C6's moments. P303a's and P303e's phases came from
  mocks made in a scratch copy, never added to the repository.

```text
Evaluation - Plan 22's corrections C1 to C12 - 18 criteria: 17 PASS, 1 FAIL
PASS 5.1 C1: P303a's cut phase interrupted, no exact count, the CLI's 0 kept apart (Cut, CutTokens 0); its row "✗ Thinking interrupted after 0.2s" with nothing after; its details "no exact count: the CLI reports 0 for an interrupted message" beside "live estimate 50"; its exit never "0 thinking tokens"; P2 the same, golden stream-p2-06 line 23 (TestCorrectionC1CutMessage, TestCorrectionC1Rows, TestThinkingClaudeP303a; Windows Terminal at 3 sizes)
PASS 5.2 C2: P12x-sandbox cut at line 61 by a crash, resumed, line 57 replayed: the crashed step closed, no count (was "10 reasoning tokens"), the next turn's 33 on its own phase; P9n cut at 25, line 21 replayed: none (was 0); a turn with no prompt shares no earlier phase; P6b: "no exact count reported" once its turn ended (TestCorrectionC2ResumeReplay, TurnWithoutPrompt, RecordedPhasesClosed, LateCount, Details)
PASS 5.3 C3: a phase live while the turn is Idle ends when a prompt is sent, at the last event's time, its end inferred; no 50 ms frame an hour later (TestCorrectionC3PromptEndsStrayPhase, TestCorrectionC3NoFramesAfterStrayPhase)
PASS 5.4 C4: a live confirmed reasoning item ends at the next step's start, its end inferred, never two live phases; its details "when the next step began (inferred)" (TestCorrectionC4NextStepEndsItem, TestCorrectionC4Details)
PASS 5.5 C5: P9a's second phase starts at 4,252 ms, its turn's last usage update: "Thought for up to 1.4s · 118 reasoning tokens", its exit timed "00:01"; P6b's second at 19,363 ms, "up to 1.6s · 7"; of the 60 recorded phases only these two and C1's two change (TestCorrectionC5StartAtLastUsage, TestCorrectionC5Rows, TestThinkingCodexP6b; the agent's dump of every phase before and after)
PASS 5.6 C6: the 26 recordings of both CLIs at 120x40 and 80x24, a frame every 50 ms: 26,372 frames, 9,838 drawing the line, never before 200 ms and always from 200 ms; 78 gaps under 200 ms crossed with no line; after a prompt none before +200 ms; no new tick, TestThinkingTicks and TestIdleDrawsNothing unchanged (TestCorrectionsEveryRecording, TestCorrectionC6Boundary, TestCorrectionC6AfterPrompt, TestCorrectionC6ActivityDelay; Windows Terminal: none at +131 ms, drawn at +349 ms)
PASS 5.7 C7: P303a at 80x24, its exit row whole, "✗ Thinking interrupted after 0.2s  ≈ 50 thinking tokens · estimate  00:00"; a Codex phase interrupted or failed at 80x24 leaves its count out, its head and time whole (TestCorrectionC7ExitRowNotCut, TestCorrectionC7CountLeftOut; Windows Terminal 80x24)
PASS 5.8 C8: in the exit, P303e's "46 thinking tokens" 11 ms after its stop, P12c-block's "≈ 2,700 thinking tokens · estimate", P2's cut phase "Thinking · no count" at 80x24, a shared count "700 thinking tokens over 2 phases"; never "live estimate", nor "token usage pending" once a count came or none can come; Codex's unchanged (TestCorrectionC8ExitCountRow, RecordedExits, NoExactCount, SharedCount; the frame scan; Windows Terminal)
PASS 5.9 C9: no summary row cut at 80 columns in any frame of any recording, the origin dropped whole and the end time kept ("▸ ✓ Thought for 14s · ≈ 2,700 thinking tokens (estimate)  18:08:12"); the summary row equals toolLine's own row at every width from 30 to 140, both sharing toolLeft, toolRight and toolRoom; tool rows unchanged (TestCorrectionC9SummaryOrigin, Recordings, FollowsToolLine; Windows Terminal 80x24)
PASS 5.10 C10: zone B held with only a dropped step new, reveal scrolls nothing and no badge shows; with a real new answer the badge's row is still kept free (TestCorrectionC10RevealCountsWhatTheBadgeCounts)
PASS 5.11 C11: TestThinkingReplayInOneBatch reads ui.Live by Thought.ID, and fails when markLive marks every phase
PASS 5.12 C12: codex-cli.md section 10 item 4 (the resume's usage replay documented for 0.145.0, sent by 0.163.0's source, a fork included, never recorded) and section 12 ("4 to 126 ms"); plan.md's X2; claude-code.md section 12 (the API's per-delta estimate relayed, the last value's fit, an interrupted message's usage its message_start's)
PASS 5.13 Each fix undone alone in a scratch copy fails its tests: 11 of 11, C1 to C11, every file restored byte for byte; the agent's own mutation runs agree
FAIL 5.14 expected go test ./... 0 FAIL twice, found 1 FAIL in the owner's environment: TestClaudeEnvironmentIgnoresCodexProxy (internal/app/codexproxy_test.go:413, amendment 19's), failing the same on the tree before the corrections; with the proxy variables unset 14 of 14 packages ok, twice. The rest PASS: gofmt empty; go build and go vet, plain and with -tags timing and live, exit 0; -race 14 ok, 0 races, 483 top-level tests, only TestDiskListLink skipped; go.mod and go.sum unchanged; the timing step 7 of 7 on mains, CPU 21% before: TestThinkingFrame p95 3.02 ms, TestLongSession p95 4.00 ms, TestStreamToFrame p95 3.53 ms, TestKeystrokeToFrame p95 1.00 ms, TestDraftKeystrokes p95 2.14 ms, TestPasteToFrame worst 1.21 ms
PASS 5.15 Golden frames: stream-p2-06.txt line 23 alone ("· 0 thinking tokens" gone, C1); the other 151 byte for byte as before the corrections
PASS 5.16 ui-design.md (Thinking: the interrupted row, the details, the exit's count row, the count left out, the activity line's 200 ms, the summary row's origin; Animations), events-and-state.md (the cut and the closed phase, C3 to C5's ends, UI.ActivitySince), reliability.md (both thinking rows); criteria 1.6 and 2.11 corrected with a note; UTF-8 without BOM, LF only
PASS 5.17 Windows Terminal, maximised then 120x40, and 80x24 for C1, C7 and C9: the six earlier runs again, P303a's and P303e's phases, P12c-block at 80x24; 39 screenshots, 354 burst frames, 14 GIFs; the earlier demonstrations kept unchanged
PASS 5.18 git status: the 81 files modified and 46 untracked before, plus the three corrections_test.go; nothing staged; no commit
Not verified: the look, until the owner has seen the demonstrations; on screen, the cases no mock replay reaches (C2 to C4, C10, a Codex exit's count left out, "no exact count reported" and "over 2 phases" in an exit), checked by tests only; the limits plan 22's summary lists (Opus or Sonnet, Codex's reasoning summaries, a real Codex resume)
Verdict: NOT DONE
Likely cause: amendment 19's test pins HTTPS_PROXY only and reads the ambient HTTP_PROXY, which since 2026-10-21 (the Windows user variables pointing at Px) equals its Codex proxy, http://127.0.0.1:3128
Proposed next step: the test pins HTTP_PROXY too (t.Setenv), so the environment cannot decide it - waiting for approval
```

**5.14's failure, fixed at the owner's request (2026-10-21).** The owner
approved the proposed step: "make TestClaudeEnvironmentIgnoresCodexProxy
pin HTTP_PROXY as well as HTTPS_PROXY, so the ambient variables cannot
decide it, then rerun the suite in my environment." The block above stays
as it was reported.
- The test now pins `HTTP_PROXY` too, with its own value, and checks that
  Claude Code keeps it, as it already did for `HTTPS_PROXY`. It is
  amendment 19's own file, so the fix goes with amendment 19's commit.
- In the owner's environment, both variables at Px, it passes.
- In a scratch copy, it fails twice when the application leaks the Codex
  proxy into its own environment (a planted `os.Setenv`): "Claude Code's
  own HTTP_PROXY is not kept" and "the Codex proxy is in Claude Code's
  environment".
- In the owner's environment, nothing unset: gofmt empty; go vet, plain
  and with -tags timing and live, exit 0; go test ./... 14 of 14 packages
  ok, twice, then with -v 483 top-level tests, 0 FAIL, only TestDiskListLink
  skipped. Every criterion of the corrections now holds.

Where the evidence is:
- The demonstrations: in this session's scratch folder, a temporary one,
  `%TEMP%\claude\C--javadev-projets-agent-command-center\dcc3e64d-0d35-4470-ab9a-d258f8675bae\scratchpad\term\out-p22c`.
  Each run and size has its own folder, and `gifs\` holds the 14 GIFs. The
  earlier demonstrations are unchanged, in session a5552254's
  `scratchpad\term\out-p22`.
- The last Limitations item above counts lines, not occurrences: the user
  name appears 40 times on those 23 lines, in each of the recording's two
  copies, and in four more committed files (plan.md, Plan 22's progress).
  The owner's decision (2026-10-21): left as it is, the repository being
  private.
