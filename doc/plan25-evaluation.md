# Plan 25 evaluation (2026-10-22)

The check of plan 25, the unrecognised lines named (brief amendment 24: the
owner's scenario 1 run on Fable 5.1 ended with "unrecognised 2" in zone A's
notes and no way to see which lines). It covers the criteria written in
[`plan.md`, Plan 25](../plan.md) before the code. Nothing is committed until
the owner approves.

How it was checked:
- Deterministic tests: `internal/state` (TestUnrecognisedKept, the replay
  tests), `internal/render` (TestDetailsUnrecognised, TestOverageWindowSkipped,
  TestDetailsErrors), `internal/provider/claude`
  (TestHookStartAndSummaryGiveNoEvent on the recorded lines), the golden
  frames `details-120x40`, `details-200x60` and `details-80x24` regenerated
  and read.
- Two live runs the owner approved, each the dashboard built from the tree
  on Claude Code 2.1.303 (Fable 5.1, the CLI's default) through the test-only
  logging wrapper, in a new maximised Windows Terminal window at 208x59, the
  real home, Px, a fresh project outside the repository, `confcheck` before
  and after, the screen read through UI Automation
  (`scratchpad\term\p25live.ps1`, `p25scen1.ps1`, their criteria written
  before each run): one short turn at 15:51, then scenario 1 at 16:05 (the
  project filled like plan 20's demo folder, the Write allowed with F2 and
  a). No terminal-hosted CLI session was open before either; none was
  closed.
- `TestWireReplay` (`ACC_WIRE_LOG`) on the three wrapper logs of the day:
  the startup-only run of plan 24 (`live\log-p24`), the short turn
  (`live\log-p25`) and scenario 1 (`live\log-p25s1`), before and after the
  mapper change.
- Build, vet, gofmt, the whole suite, the race detector on the three
  packages that changed, the timing step alone.

```text
Evaluation - Plan 25, the unrecognised lines named - 7 criteria: 7 PASS, 0 FAIL
PASS 25.1 TestUnrecognisedKept: 22 unrecognised events leave Unrecognised 22 and UnrecognisedLast 20 entries, the oldest kept "system/s3" with its time, reason "r" and line, the newest "system/s22"
PASS 25.2 TestDetailsUnrecognised at 98 and 198 cells: the UNRECOGNISED section between ERRORS and TELEMETRY, newest first ("stream_event/content_block_x" before "user/tool_result_meta" before "system/tool_use_summary"), each head row "<time>  <type> · <reason>", the 300-character line cut to the width ending in "…", no row wider than the width, "unrecognised events 3" in COUNTERS, "none" on an empty state, cutRight("héllo wörld", 6) = "héllo…"; the golden frames details-120x40, details-200x60 and details-80x24 regenerated: the section with "none" (3 frames changed, nothing else)
PASS 25.3 scenario 1 through the wrapper at 16:05 (S1.1 to S1.8 below: the turn 89 s, one Write card allowed, zone A's note "unrecognised 2" as in the owner's run, found.txt the only new file, the configuration check PASS; S1.5 FAIL is the script's: End then Enter did not open the details view, the count was read from the note); TestWireReplay on its log before the change: 2 unrecognised, both "system/task_summary", at +68.8 s right after system/task_started with the orchestrator's narration as detail, at +102.0 s right after the turn's result with detail null; after the change (task_summary known with no event, as post_turn_summary): 0 unrecognised, 87 lines, the same 20 event kinds otherwise; the two recorded lines in TestHookStartAndSummaryGiveNoEvent give no event
PASS 25.4 TestWireReplay: log-p24 11 lines 0 unrecognised, log-p25 58 lines 0 unrecognised, log-p25s1 87 lines 0 unrecognised, every event kind counted; without ACC_WIRE_LOG the test is skipped
PASS 25.5 TestOverageWindowSkipped on the recorded rate_limit_event of 15:51 (unifiedWindows five_hour, seven_day, seven_day_overage_included): quota.5h 17, quota.7d 26, no value for the overage key, two windows in zone A, 0 unrecognised; research/claude-code.md and reliability.md name the key
PASS 25.6 go build ./... ok; go vet ./... clean; gofmt -l . empty; go test ./... -count=1 14 packages ok (internal/app 38.4 s); go test -race -count=1 with CGO_ENABLED=1 and the winlibs gcc on internal/state (8.6 s), internal/render (52.0 s) and internal/provider/claude (23.8 s): ok, no race; the timing step alone 7 of 7 (plan24\p25-checks.log, 16:12): TestKeystrokeToFrame p95 1.01 ms and 1.11 ms in the window, TestDraftKeystrokes p95 2.04 ms, TestLongSession delta-to-frame p95 5.04 ms, TestPasteToFrame max 1.24 ms, TestStreamToFrame p95 4.00 ms, TestThinkingFrame p95 3.56 ms, every target 16 ms (100 ms for the paste)
PASS 25.7 ui-design.md (the telemetry table's errors row, the details view, the states table), events-and-state.md (the State model, the known lines), reliability.md (the unrecognised row), research/claude-code.md (task_summary's row), evaluation.md (the replay tool), AGENTS.md (status), brief.md (amendment 24), plan.md (D1 to D6, 25.1 to 25.7, progress); every changed text file read byte by byte (crcheck.mjs): no BOM, no CR; git status lists the intended files only; nothing committed
```

**The two live runs**, as their scripts recorded them (the lines cut at
400 characters here keep their meaning; the files hold them whole):

```text
p25live claude-code  2026-10-22T15:51:32  project C:\Users\amolimom\AppData\Local\Temp\acc-p25-claude-155131  prompt 'Reply with the word ok.'
15:51:32.249 [L25.1 before] PASS  terminal CLI sessions: none; Px listening: True; claude 2.1.303 (Claude Code); snapshot exit 0
15:51:40.744 [L25.2 Ready on Claude Code through the wrapper, the quota windows before any prompt] PASS  ready True; windows True; argv {"argv":["<HOME>\\.local\\bin\\claude.exe","-p","--input-format","stream-json","--output-format","stream-json","--verbose","--include-partial-messages","--forward-subagent-text","--permission-prompt-tool","stdio","--permission-mode","default"],"cli_pid":34652,"sta…
15:51:59.028 [L25.3 the turn completed with an answer] FAIL  10 s; model row '│ ✿ Claude Fable 5.1                         ✓ COMPLETED ││ ▎ YOU  15:51                                                                                                                                       │'; RESULT on screen False
15:51:59.036 [L25.4 the unrecognised note read] PASS  before the prompt 0, after the turn 0; errors note ''; quota note 'quota updated 3s ago'
15:52:01.519 [L25.5 the details view: the count equals the note] FAIL  2 F6, A focused True; unrecognised events '', errors ''
15:52:05.090 [L25.6 Ctrl+Q closes the window, no process left] PASS  2,73 s; left: wrapper or CLI , new claude.exe , acc-p24 0; of the 1 processes seen under the wrapper, left 
15:52:05.094 [L25.7 the project is empty] PASS  no file
15:52:06.322 [L25.8 configuration check] PASS  exit 0; mixed  C:\Users\…\.claude.json  changed (key-level diff follows)  sha256=bcabd4bcdfd29d230b43d3bb816c399267c830d9669f35c2cdaabea30ea263da | mixed  C:\Users\…\.claude.json  key cachedExperimentData  changed (state) | mixed  C:\Users\…\.claude.json  key cachedExperimentFeatures  changed (state) | mixed  C:\Users\…\.claude.json  key cachedGrowthB…
15:52:06.324 [END] FAIL  2 FAIL
```

```text
p25scen1 claude-code  2026-10-22T16:05:18  project C:\Users\amolimom\AppData\Local\Temp\acc-p25-scen1-160516
16:05:18.454 [S1.1 before] PASS  terminal CLI sessions: none; Px listening: True; claude 2.1.303 (Claude Code); snapshot exit 0; 14 files in the project
16:05:27.441 [S1.2 Ready on Claude Code through the wrapper] PASS  ready True; windows True; argv {"argv":["<HOME>\\.local\\bin\\claude.exe","-p","--input-format","stream-json","--output-format","stream-json","--verbose","--include-partial-messages","--forward-subagent-text","--permission-prompt-tool","stdio","--permission-mode","default"],"cli_pid":31396,"start_ms":1792677923455,"wrapper_pid":117…
16:07:13.097 [S1.3 the turn completed, the approval allowed] PASS  89 s; model row '│ ✿ Claude Fable 5.1                         ✓ COMPLETED ││ ▎ YOU  16:05                                                                                                                                       │'; 1 card(s) allowed; 'DONE' row: True
16:07:13.115 [S1.4 the unrecognised note read] PASS  after the turn 2; errors note ''; quota note 'quota updated 1s ago'
16:07:15.498 [S1.5 the details view] FAIL  2 F6, A focused True; title False; unrecognised events ''
16:07:19.127 [S1.6 Ctrl+Q closes the window, no process left] PASS  2,79 s; left: wrapper or CLI , new claude.exe , acc-p24 0; of the 1 processes seen under the wrapper, left 
16:07:19.134 [S1.7 the project gained found.txt only] PASS  new files: found.txt
16:07:20.426 [S1.8 configuration check] PASS  exit 0; mixed  C:\Users\…\.claude.json  changed (key-level diff follows)  sha256=a9d4432a96ea35a88f0881cca053d6d6c4179af0a39e3c50e6bc007da0a946e9 | mixed  C:\Users\…\.claude.json  key cachedGrowthBookFeaturesAt  changed (state) | mixed  C:\Users\…\.claude.json  key cachedUsageUtilization  changed (state) | mixed  C:\Users\…\.claude.json  key claudeCode…
16:07:20.428 [END] FAIL  1 FAIL
```

**What the lines are.** Both are `system/task_summary`, a system subtype the
mapper did not know and no probe had recorded (P303b, the subagent probe of
2026-10-19 on the same 2.1.303, has `task_started`, `task_progress`,
`task_updated` and `task_notification` only), so a server-side change, not
the binary's. The first comes right after `task_started`, its `detail` the
orchestrator's narration before the Agent call (the Px reminder from the
owner's global notes, then "I'm launching one general-purpose subagent…"),
which zone B already shows as the assistant's message; the second comes
right after the turn's `result`, its `detail` null. The mapper now knows
them as it knows `post_turn_summary`: no event, nothing drawn, nothing
counted. The short turn of 15:51 (no subagent) had none, which placed them
in the subagent turn before the rerun named them.

**What the owner sees now.** The note "unrecognised N" stays in zone A when
a line the mapper does not know arrives; Enter on TELEMETRY lists the last
20 such lines with their time, the mapper's type and reason and the line
itself, so the next one is read there. For this day's lines the note no
longer appears.

**Evidence outside the repository**, in the session's scratchpad
(`C:\Users\amolimom\AppData\Local\Temp\claude\C--javadev-projets-agent-command-center\dcc3e64d-0d35-4470-ab9a-d258f8675bae\scratchpad\`):
`term\out-p25-live\` and `term\out-p25-scen1\` (`results.txt`, the
screenshots with their UI Automation rows, `confcheck.txt`, `before.json`),
`live\log-p25\` and `live\log-p25s1\` (the wrapper logs: `in.jsonl`,
`out.jsonl`, the times, `argv.json`, `end.json`), `term\p25live.ps1`,
`term\p25scen1.ps1`, `livewrap\claudebin\` (the wrapper), `plan24\p25-checks.log`
(vet, the suite, the race run, the timing step).
