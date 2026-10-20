# Plan 21 evaluation (2026-10-20)

The final check of plan 21, what the review of d84e5d4 found, approved by
the owner with the recommended choices ("I approve plan 21 with the
recommended choices"): 2 (a), 4 (a), 5.1 build, 5.2 to 5.5 correct, 5.6a
build, 5.6b to 5.6d correct, 5.7 build, 5.8 correct. It checks the criteria
of [`plan.md`, Plan 21](../plan.md#plan-21---what-the-review-of-d84e5d4-found),
written before the code, each for the choice the owner made. The evidence
is the tests (each new check also run once on the old code or under a
mutation, so it can fail), the golden frames, a mock run in a real Windows
Terminal, and the timing step. The block below is the summary reported to
the owner. The results are not to be rewritten.

```text
Evaluation - Plan 21, what the review of d84e5d4 found - 40 criteria: 38 PASS, 2 FAIL
PASS 1.1 6,000 events one second apart: zone D reads "time 1h 40m session"; the old code read "33m 20s session" (TestSessionTimeKeepsFirstEvent, run once on it and failing)
PASS 1.2 1,999 events: "33m 19s session", the value the old code gives; no event: "N/A (…)"; TestTelemetryNA passes
PASS 1.3 A resume after a crash, past 2,000 events: "1h 00m session" from the first event (the old code: "34m 55s session")
PASS 1.4 No golden frame changed by item 1 (TestGoldenFrames without -update); events-and-state.md's State names Applied and FirstAt
PASS 2.1 Both targets at 1 ns, reverted after: the ordinary run ok, each test logging "not judged outside the timing step"; with -tags timing both FAIL ("max 2.9571ms over the 1ns target", "p95 5.986ms over the 1ns target")
PASS 2.2 go test ./... -count=1 -v: TestPasteToFrame and TestStreamToFrame run and PASS, their other checks unchanged
FAIL 2.3 The timing step runs the six tests, each within its target: expected 6 of 6 within target, found 5; TestLongSession p95 16.6898 ms over its 16 ms target (p50, p95 and maximum of each recorded in plan.md with the conditions)
PASS 2.4 AGENTS.md (Commands) and evaluation.md (Keystroke timing) give the six-test command; evaluation.md says what the code does (timingBuild, overTarget)
PASS 3.1 Every stream_event line of the 13 Claude Code recordings maps to its agent's last message_start id, equal to its api_message_id: 1,237 of 1,237 (TestStreamEventsTakeTheMessageStartID; p4 has none)
PASS 3.2 The 14 recordings replayed with api_message_id removed from every line give the same conversation as replayed as is, one entry per message, oneEntryPerMessage on each (TestConversationDoesNotNeedAPIMessageID)
PASS 3.3 A delta with no message_start before it takes its api_message_id (TestStreamEventWithoutMessageStartFallsBack)
PASS 3.4 The mapper's comment and claude-code.md's stream_event row say api_message_id is undocumented, with the recordings that show it
PASS 3.5 Mutation, a mapper reading neither id: 3.2's test fails ("assistant entry with message id \"\" (empty or repeated)"), and 3.1's
PASS 4.1 P3b's agent at 160x24, 160x25, 160x26: C 4 rows, "general-purpose" on C's first row left of the rule, the preview's heading right of it; A's and B's bottoms on one row, A and B 11, 12, 13 rows; no row wider than 160 (TestZoneCFloor; on the old code C 3 and no agent)
PASS 4.2 160x27: C 4, A and B 14, as before; TestAdaptiveLayout and TestDashboardGeometry pass unchanged
PASS 4.3 ASM-12 and ui-design.md (Zone C) state the floor
PASS 5.1 (i) No file: the default keys, no problem (TestLoadKeysNoFile); the existing input tests pass unchanged
PASS 5.1 (ii) {"keys": {"send": ["ctrl+enter"], "newline": ["enter"]}}: Ctrl+Enter sends, Enter adds a newline, Ctrl+J and Shift+Enter do neither (TestConfiguredKeys)
PASS 5.1 (iii) The footer, the editor's status row, the draft window and the F1 page show the keys in force (TestConfiguredKeysShown, TestConfiguredDraftWindow)
PASS 5.1 (iv) A paste ending in a newline never sends, with either file; a send key in an open paste adds a newline (TestPasteNeverSubmits, TestPasteNeverSubmitsConfigured)
PASS 5.1 (v) Invalid JSON, an unknown setting, an unknown key, an empty list, a key in both lists: config.json and the problem on the startup screen, and in a mock run's status row; the default keys apply (TestKeysProblemOnStartup, TestKeysProblemInMockRun, TestLoadKeysProblems)
PASS 5.1 (vi) (ii) and (iv) pass with the Claude Code, Codex and mock adapters
PASS 5.1 (vii) The file is only read (TestLoadKeysOnlyReads); nothing else in it is used; go.mod and go.sum unchanged
PASS 5.1 (viii) architecture.md (How it starts, ASM-7, ASM-9, ASM-13) and ui-design.md (Focus and keys, Colours) say what is built
PASS 5.2 architecture.md's Detect rows name no status command and the reports that tell a subscription from an API key; SignIn and the two "not read yet" comments gone; go build ./... exit 0
PASS 5.3 architecture.md's Capabilities paragraph names the one flag read (AutoApprovals, keys.go) and the provider-id checks, each with its file
PASS 5.4 architecture.md (Threads, ASM-3), ui-design.md (Rendering, Assumptions) and events-and-state.md (Pipeline) say 60 frames per second, Bubble Tea's default, and at most 512 events as one update; none says 30
PASS 5.5 reliability.md's test 7 says the test parses the sources with go/parser
PASS 5.6a "↻ 2" in amber after the agent's time at 120x40, 200x60 and 80x24, none with no retry, from a Codex thread's error with willRetry (TestAgentRetries; mutation, no count drawn: it fails at the three sizes); ui-design.md says why a Claude Code agent shows none
PASS 5.6b ui-design.md's Origins table says AI text carries no label
PASS 5.6c ui-design.md names the texts shown for a failed turn, an interrupted one and a refused send, each found in the code; "interrupted by you" and "send refused" no longer promised
PASS 5.6d ui-design.md's Rendering says what is cached across frames (each line's wrap, a finished entry's row count) and what is not (Markdown)
PASS 5.7 The six file-full-* and draft-full-* frames at 80x24, 120x40 and 200x60 begin with the title bar and "◆ MOCK — not a live session", each window one row shorter; TestGoldenFrames' new check, run once with the old layout, failed those six; the viewer's and the draft window's tests pass (TestFileRect, TestDraftWindow, TestFileKeys updated to the new full screen)
PASS 5.8 ASM-23 reads %LOCALAPPDATA%\agent-command-center\drafts, checked from the bytes
PASS 6.1 go build ./... and go vet ./... exit 0; gofmt -l . empty; go test ./... 0 FAIL in 14 packages, only TestDiskListLink skipped; go.mod and go.sum unchanged
FAIL 6.2 The timing step within its targets: expected 6 of 6, found 5 (TestLongSession, as 2.3)
PASS 6.3 Only the six frames of 5.7 changed, each diff reviewed (the title bar added, the window one row shorter); none for items 1 to 4
PASS 6.4 Windows Terminal, the P3b mock, maximised 156x44: the file viewer and the draft window full screen, each under the title bar with the badge; a 160x24 grid: C 4 rows, the agent's two rows left of the preview, A and B 11 rows; three screenshots sent to the owner
PASS 6.5 AGENTS.md's status line names plan 21; plan.md holds the progress; this file the summary; every changed file UTF-8 without BOM, LF only, checked from the bytes
PASS 6.6 git status lists only the intended files; no commit
Not verified: the look of the three screenshots, until the owner has seen them; Claude Code's subagent_retry (tool_progress), not mapped since no recording holds one
Verdict: NOT DONE
Likely cause: for 2.3 and 6.2, plan 20's RESULT rule, not plan 21: agentsDone (internal/render/look.go) walks every agent for every exchange on every frame, so a frame's cost grows with exchanges times agents. In the long session (5,281 entries, 240 turns, 720 agents) it takes 32% of the CPU (profile: 2.09 s of 6.57 s), and the frames grow from p95 2.6 ms in the first tenth to 10.0 ms in the last. The same test on d84e5d4, plan 20 as committed, under the same conditions: p95 16.10 ms, FAIL; on 8b3b16b, plan 18, before that rule: p95 2.65 ms, PASS
Proposed next step: count each stage's agents not completed once a frame (prefix sums, undoneByStage) and answer agentsDone with two lookups, the rule unchanged. Tried in a scratch copy of this tree, not applied: TestLongSession p95 4.19 ms (last tenth 2.87 ms), the render tests and the golden frames unchanged. Then the timing step again, 6 of 6 - waiting for approval
```

## The timing step

Run alone on 2026-10-20 at 19:40, on mains power (status 2; on battery at
30% charge until just before), the CPU at 37% just before, the
applications open: Chrome, Edge, Teams, VS Code, the Claude app, Windows
Terminal, Settings. The command:

```text
go test -tags timing ./internal/app -run '^(TestKeystrokeToFrame|TestDraftKeystrokes|TestKeystrokeCost|TestLongSession|TestPasteToFrame|TestStreamToFrame)$' -count=1 -v
```

| Test | p50 | p95 | Maximum | Target | Result |
| - | - | - | - | - | - |
| TestKeystrokeToFrame, 10,000 lines | 0 s | 1.0062 ms | 1.9993 ms | p95 ≤ 16 ms | PASS |
| TestKeystrokeToFrame, draft window | 503.4 µs | 1.3271 ms | 9.5393 ms | p95 ≤ 16 ms | PASS |
| TestDraftKeystrokes, 50,000 lines, 9 saves | 1.034 ms | 3.0566 ms | 41.7277 ms | p95 ≤ 16 ms | PASS |
| TestKeystrokeCost | 524 µs a key at 10,000 lines, 1.27 ms at 50,000; 616 allocations a key | | | judges nothing | PASS |
| TestLongSession, the end, 200x60, 400 deltas | 10.6808 ms | 16.6898 ms | 22.169 ms | p95 ≤ 16 ms | FAIL |
| TestPasteToFrame, 50,000 lines, 10 runs | 3.5169 ms | | 4.2555 ms | max ≤ 100 ms | PASS |
| TestStreamToFrame, 2,000 lines | 3.0111 ms | 8.9818 ms | 21.9225 ms | p95 ≤ 16 ms | PASS |

TestLongSession's frames during the session: first tenth p95 3.35 ms, last
tenth p95 16.55 ms. Plan 20 recorded 5.52 ms and then 5.49 ms for its
end, before its last correction; that correction and the commit were not
timed again.

## The investigation

Not reruns of the failed criterion, which stays FAIL; each was run once,
under the same conditions, for the cause (doc/evaluation.md, Keystroke
timing):

- d84e5d4, the committed plan 20, in a scratch copy: TestLongSession p95
  16.0973 ms, FAIL; first tenth 2.29 ms, last tenth 12.51 ms. Plan 21 did
  not cause the miss.
- 8b3b16b, plan 18, in a scratch copy: p95 2.646 ms, PASS; first tenth
  2.47 ms, last tenth 2.00 ms. The machine is not slower than when the
  timings were first recorded (plan 17: 3.59 ms).
- A CPU profile of this tree's TestLongSession (6.57 s of samples):
  `render.Draw` 75%, `render.looks` 37%, its `exchange` 33%, and
  `agentsDone` 32% with the map lookups under it.
- The proposed change in a scratch copy of this tree: p95 4.185 ms,
  maximum 9.6263 ms, last tenth 2.87 ms; `go test ./internal/render` and
  TestGoldenFrames ok. Not applied: the owner decides.

## The proposed next step, approved

The owner approved it on 2026-10-20 ("Apply the speed fix, then commit").
`undoneByStage` counts each stage's agents not completed once a frame, and
`agentsDone` answers from it with two lookups (`internal/render/look.go`);
the rule is unchanged. The block below re-checks the two failed criteria
and the regressions; the block above stays as it was.

```text
Evaluation - Plan 21, the approved fix of the long session's frame time - 6 criteria: 6 PASS
PASS F.1 agentsDone from undoneByStage gives the rule's answer for every pair of turns from 0 to n+1 and from -1 to n+1: P3b 20 pairs, P7 12, a synthetic session with an agent of each status in each turn and turns with none and one not completed 72, 38 of them done (TestAgentsDoneCounts); two mutations (`<= 1` for `== 0`; the range's first turn left out) each fail it
PASS 2.3 The timing step, the six tests, each within its target: TestKeystrokeToFrame p95 1.0415 ms (draft window 1.1376 ms), TestDraftKeystrokes 3.0418 ms, TestKeystrokeCost 220 µs and 556 µs a key, TestLongSession p95 3.9032 ms (it was 16.6898 ms), TestPasteToFrame max 1.1631 ms, TestStreamToFrame p95 3.515 ms
PASS 6.2 The timing step within its targets: 6 of 6
PASS 6.1 go build ./... and go vet ./... exit 0; gofmt -l . empty; go test ./... -count=1 0 FAIL in 14 packages, only TestDiskListLink skipped; go.mod and go.sum unchanged
PASS 6.3 No golden frame changed by the fix: the six of 5.7 only (TestGoldenFrames without -update)
PASS 6.5 Every changed file UTF-8 without BOM, LF only, checked from the bytes
Not verified: the look of the three screenshots, until the owner has seen them
Verdict: DONE
```

The timing step, run alone on 2026-10-20 at 20:00, on mains power (status
2, 53%), the CPU at 29% just before, the applications open: Chrome, Edge,
Teams, Windows Terminal, Settings.

| Test | p50 | p95 | Maximum | Target | Result |
| - | - | - | - | - | - |
| TestKeystrokeToFrame, 10,000 lines | 0 s | 1.0415 ms | 2.0003 ms | p95 ≤ 16 ms | PASS |
| TestKeystrokeToFrame, draft window | 0 s | 1.1376 ms | 2.1069 ms | p95 ≤ 16 ms | PASS |
| TestDraftKeystrokes, 50,000 lines, 8 saves | 1.0121 ms | 3.0418 ms | 5.391 ms | p95 ≤ 16 ms | PASS |
| TestKeystrokeCost | 220 µs a key at 10,000 lines, 556 µs at 50,000; 616 allocations a key | | | judges nothing | PASS |
| TestLongSession, the end, 200x60, 400 deltas | 2.0002 ms | 3.9032 ms | 5.1539 ms | p95 ≤ 16 ms | PASS |
| TestPasteToFrame, 50,000 lines, 10 runs | 1.0041 ms | | 1.1631 ms | max ≤ 100 ms | PASS |
| TestStreamToFrame, 2,000 lines | 1.5154 ms | 3.515 ms | 15.5502 ms | p95 ≤ 16 ms | PASS |

TestLongSession's frames during the session: first tenth p95 1.79 ms, last
tenth p95 3.54 ms (16.55 ms before the fix): a frame no longer grows with
the session's turns times its agents.
