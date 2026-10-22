# Plan 24 evaluation (2026-10-22)

The check of plan 24, zone A redesigned with the telemetry and zone D removed
(brief amendment 22, the owner's image `references/03-zone-a-telemetry.webp`).
It covers the criteria written in [`plan.md`, Plan 24](../plan.md) before the
code. Nothing is committed: the owner reviews the Windows Terminal screenshots
first, as the amendment asks.

How it was checked:
- Deterministic tests in `internal/render` (`zonea_test.go`, `telemetry_test.go`,
  `layout_test.go`, `refine_test.go`, `inspect_test.go`) and `internal/app`
  (`zonea_test.go`, `details_test.go`, `inspect_test.go`, `zoom_test.go`,
  `scenarios_test.go`, `golden_test.go`), on the probe recordings, the mock
  files and the new synthetic mock `testdata/mock/claude-quota-windows.jsonl`
  (P3b with three quota windows at the image's values, criterion 24.17).
- The golden frames regenerated (`-update`) and read: 66 changed, 4 added
  (`quota-*`); zone C's rows compared with the frames before.
- Windows Terminal, the executable built from this tree, two sessions on a
  copy of the synthetic mock whose reset times were moved to 4h 12m and 5d 3h
  after the run (`scratchpad\plan24\livemock.mjs`), each in a window of its
  own with a temporary home and no agent CLI on the PATH
  (`scratchpad\term\p24demo.ps1`, the plan 22 harness): maximised at 9 pt
  (208x59) and at 120x40. The rows were read through UI Automation at each
  screenshot; the screenshots are in `scratchpad\term\out-p24\claude-max`
  and `claude-120x40` (A-completed, B-running, B-thinking, C-details,
  A-focused, D-120x40).
- The timing step, the race detector, build, vet, gofmt and the whole suite.

```text
Evaluation - Plan 24, zone A with the telemetry, zone D removed - 35 criteria: 35 PASS, 0 FAIL
PASS 24.1 TestDashboardGeometry: at 120x40, 156x44, 160x45, 200x60 and 208x59 on P2 and P3b, A and B from row 2 to one bottom row, A 36, 44, 45, 56 and 58 columns, B the rest, C full width from the row under them to the footer, three boxes only, no row wider than the width
PASS 24.2 TestDashboardGeometry and TestZoneAStructure find no "D · " and no "QUOTA & TELEMETRY" row; grep over the 156 golden frames: 0 files with "D · QUOTA" or "D · ERRORS"
PASS 24.3 TestDashboardGeometry at 80x24: A's row on row 2, B rows 3 to 19 (17 rows), C rows 20 to 22, both full width, two boxes; with A focused one box from row 3 to 22 holding "STAGES · inferred" and "✦ TELEMETRY" (golden tree-80x24.txt)
PASS 24.4 TestZoneAStructure at 120x40 and 208x59 on the P3b mock: the top edge "╭──" without a title; rows "A · ORCHESTRATOR", "✿ …", a rule, "STAGES · inferred from activity"; then "✦ TELEMETRY", "▾ SESSION USAGE" under it, "▾ QUOTA WINDOWS" later, the notes "ⓘ …" after a rule as the last rows (golden quota-120x40.txt, quota-200x60.txt)
PASS 24.5 TestZoneAStructure: the tree 6 rows or more at both sizes with the full form (two "Resets in" rows); at 100x24 the tight form, "  Session (5hr)  35%† · 14h 39m" on one row, no "Resets in", the tree 5 rows; SESSION USAGE folded reads "▸ SESSION USAGE", its three rows gone to the tree
PASS 24.6 TestOrchestratorRow: "✿ Claude Haiku 5.5" on the P3b mock, "◆ GPT-5.6-sol" on the Codex mock, "model N/A (session starting)" before the session starts, the icon in the model's colour; modelTitle gives "Claude Sonnet 4.5" for a dated id, "Claude Fable 5.1" and "Claude Mythos 5.1" (the families added after the owner's live session, below), an unknown id as it is
PASS 24.7 TestOrchestratorRow: the row ends with "✓ COMPLETED" at 32, 40, 52 and 54 cells, each row exactly that wide; P3b at its Write request at 32 cells reads "✿ Haiku 5.5 … ! AGENT WAITING", at 54 "✿ Claude Haiku 5.5 …"; the Windows Terminal rows: the status ends the second inner row of zone A at both sizes, the heading row holds nothing else
PASS 24.8 TestOrchestratorRow: 15 states, each the text and the style of the plan (starting, closing, crashed, closed, an approval pending, interrupting, "⠋ THINKING" in the title colour on a live thinking phase, "⠋ RUNNING", an agent waiting, one and two agents running, completed, failed, interrupted, idle)
PASS 24.9 TestOrchestratorRow: P3b at its Write request "! AGENT WAITING" (the turn completed, the agent waiting); the mock whole "✓ COMPLETED"
PASS 24.10 TestStageBlocks at 208x59: "✓ TURN 1" with its time at the row's end, "  Use the Agent tool to launch one gene…", "  Agent · agent general-purpose" (the Agent call came before the agent's start); "✓ TURN 2" then "  started by the CLI"; unfolded with ui.Fold, "├─ ✓ Agent  " and "└─ ✓ agent  general-purpose" under the prompt's line and no items row; TestTreeP3b: no "completed" word on a head row
PASS 24.11 TestTreeKeys (app): ↓ past the tree's last node reaches TELEMETRY, then SESSION USAGE and QUOTA WINDOWS where it stops; ← on QUOTA WINDOWS folds it ("▸ QUOTA WINDOWS", no "Weekly"), → unfolds; Enter on an agent's node inspects, Esc returns to A; TestStageBlocks: the cursor's background on the folded block's three rows and not on the next, on the head row alone once unfolded
PASS 24.12 TestTreeP7Running: an open stage unfolded with "○ (more as events arrive)"; TestTreeKeys: the closed stage folded by default, → unfolds it and the fold state stays; the Windows Terminal B-thinking screenshot shows "◉ TURN 2 · running" unfolded with the moving dot, A-completed the same turn folded to its block
PASS 24.13 TestUsageSection: at 208x59 "CONTEXT │ TOKENS │ EST. COST", "4% of 1M │ 159.4k │ $0.01", the bar │ "session total" │ "API-equivalent"; at 120x40 (32 cells) "Context 4% of 1M" and its bar, "Tokens 159.4k session total", "Est. cost $0.01 API-equiv."; Codex P7 "6% of 258.4k", "184.3k" "session total", "N/A" "Codex reports none"
PASS 24.14 TestUsageSection: P2 up to its prompt, the three columns "N/A" with "no result yet", no "0" and no "$" in the rows; TestTelemetryNA: Codex before a model call "context N/A (no model call yet)"
PASS 24.15 TestUsageSection: fillBar(3.5, 50) is 2 "█" and 48 "░", the fill in the title colour
PASS 24.16 TestQuotaWindows: P1 line 3 "  Session (5hr)" ending "22%†" (52 cells), 11 of 50 cells filled, "  Resets in …"; "Weekly (7 day)" 45%† 23 cells; two windows. TestTelemetryParsing: P6b line 7 "Session (5hr) 0%", "Weekly (7 day) 0%", no "†"
PASS 24.17 TestQuotaWindows on claude-quota-windows.jsonl: "Session (5hr) 0%† (Resets in 4h 12m)", "Weekly (7 day) 17%† (Resets in 5d 3h)", "Fable limit 0%† (Resets in 5d 3h)" at the test clock, 9 of 50 cells filled for 17%, an empty track for 0%; the Codex mock and the P3b mock two windows each, no "Fable"; golden quota-*.txt at the four sizes
PASS 24.18 TestQuotaWindows: 22%, 45% and 89% in the title colour, 90% and 100% red; -5% fills 0 cells, 150% all 50; TestTelemetryWholeItems: no "▰" or "▱" on any mock file's frame at 100x30, 120x40, 156x44 and 200x60
PASS 24.19 TestQuotaWindows: a second report with quota.5h 60 reads 60%, 30 of 50 cells, "22%" gone
PASS 24.20 TestQuotaWindows: countdown 5d 3h 20m → "5d 3h", 5d 0h 20m → "5d", 4h 12m 30s → "4h 12m", 4h → "4h", 12m 30s → "12m", 45s and -1m → "<1m"; an hour later "Resets in 3h 12m"; no reset time "Reset time unavailable"; the clock tick past the reset makes the window N/A "window reset; waiting for the next update", nothing negative
PASS 24.21 TestTelemetryNA: "Session (5hr) N/A (API key: ChatGPT sign-in needed)" and the weekly window alike, "Session (5hr) N/A (plan has no 5-hour limit)" with "Weekly (7 day) 12%", "N/A (read failed: upstream timeout; last 0% 1m 00s ago)", "N/A (API key: no subscription window)", "Weekly (7 day) allowed_warning (no percentage reported)"; no errors note at 0 errors
PASS 24.22 TestNotes: P3b "ⓘ quota updated Ns ago · context derived from usage" then "† undocumented field · cost at API rates" on two rows; with 2 errors the first part "errors 2" in red; Codex P7 "plan plus", "credits none", "tokens summed over threads", no cost; an empty state "ⓘ values from the CLI's events"; the heading row "✦ TELEMETRY … Enter details" 54 cells wide
PASS 24.23 TestDetailsKeys: F6 to A, ↓ to TELEMETRY, Enter opens "◀ back (Esc)   ERRORS AND TELEMETRY" with "23 errors · last 50 kept", newest first, scrolling as before; Esc returns to A, the cursor on TELEMETRY, the footer "Enter inspect or details"; TestTelemetryMouse: a click on TELEMETRY opens the details, on SESSION USAGE folds then unfolds it, on turn 1 moves the cursor; Windows Terminal C-details (F6 F6 ↓ ↓ Enter) shows the view with quota.7d.fable 0%† among the values and the COUNTERS
PASS 24.24 TestTabZones: F6 and Tab cycle B, C, A, B; TestZoneAKeyWords: the overlay says "next / previous zone (B, C, A)", "in A on TELEMETRY, the errors and every value", "A FOCUSED PANEL (A, C, INSPECTION)", nothing of zone D; the footer with A focused is D11's line (Windows Terminal A-focused, both sizes)
PASS 24.25 TestZoom: A and C zoom and restore the exact frame at the three sizes; A zoomed shows "✦ TELEMETRY" and "Weekly (7 day)" everywhere, "Resets in" at 120x40 and 200x60
PASS 24.26 go build ./... and go vet ./... exit 0; gofmt -l . empty; go test ./... 14 of 14 packages ok, 521 top-level tests, 1 skipped (TestDiskListLink, as before); the golden frames: 66 changed, 4 added, every dashboard frame three boxes; zone C's rows identical before and after at dashboard-120x40, dashboard-200x60 and dashboard-codex-200x60; zone B's rows the same texts re-wrapped to its width
PASS 24.27 The timing step 7 of 7, run alone on mains (BatteryStatus 2), CPU 45% just before, VS Code and the browsers open: TestKeystrokeToFrame p95 1.02 ms (the draft window 1.42 ms), TestDraftKeystrokes p95 2.04 ms, TestLongSession p50 3.52 ms, p95 6.00 ms, max 10.52 ms (first tenth p95 2.08 ms, last tenth 4.27 ms, heap 7.8 MiB), TestPasteToFrame worst 1.65 ms, TestStreamToFrame p95 4.77 ms, TestThinkingFrame p50 2.11 ms, p95 3.51 ms, max 8.15 ms. A first run failed 2 of 7 (TestLongSession p95 96.10 ms, TestThinkingFrame p95 16.59 ms): a Windows Terminal replay with UI Automation polling ran beside it, against the step's conditions, and the tree scanned every tool once per folded turn each frame; the scan is now one pass per frame (render.indexItems), then the run above
PASS 24.28 go test -race ./internal/app ./internal/render ./internal/state: 3 ok, 0 races (243.9, 93.4 and 9.4 s); render again after the tree's change, ok
PASS 24.29 TestCountdownRedraw (render) and TestClockRedraw (app): a second later only rows of zone A differ, a minute later likewise with "Resets in 4h 11m"; no row of B or C differs either time
PASS 24.30 TestScopedWindows: unifiedWindows.seven_day_fable gives quota.7d.fable 30%, undocumented, its reset time, shown third as "Fable limit"; rateLimitType seven_day_opus with a utilization gives quota.7d.opus 50% documented; rateLimitType seven_day_fable beside unifiedWindows' five_hour and seven_day gives quota.7d.fable 40% documented, the three windows shown (the mapper's gap found after the owner's live session, below); no window the event did not carry; the labels "Opus limit", "90 min window", "1-day window", "2h window". TestMockFiles with -update rewrote nothing but the new file (git status: the eight mock files unchanged, claude-quota-windows.jsonl added)
PASS 24.31 Windows Terminal, maximised at 9 pt, 208x59: A-completed (24.1 s) with "A · ORCHESTRATOR", "✿ Claude Haiku 5.5 … ✓ COMPLETED", the two turns' blocks, SESSION USAGE's three columns, the three windows with "Resets in 4h 10m", "5d 3h", "5d 3h", C full width, no "D ·" row; B-running (3.6 s) "model N/A (not reported yet) … ⠸ RUNNING" before the model was reported, B-thinking (16.3 s) "✿ Claude Haiku 5.5 … ⠇ THINKING" with "◉ TURN 2 · running" and the moving dot; C-details after F6 F6 ↓ ↓ Enter; A-focused after Esc; at 120x40 the same five, D-120x40 in the compact form (C holds an agent), every section readable, nothing overlapping
PASS 24.32 In the UI Automation rows the model row's right edge "│" is on the same column as every other row's at both sizes, so "✿" took one cell (a wider glyph would have pushed the status into the edge); "◆" the same on the Codex golden frames' rows; the screenshots' pixels show the status flush with the frame's inner edge, as "Enter details" and the percentages
PASS 24.33 The conformance table below, 22 rows PASS
PASS 24.34 ui-design.md (Layouts with the three frames redrawn, Dashboard, Zone A rewritten, "Zone A: telemetry" in place of Zone D, Colours and symbols, Animations, Focus and keys, Scrolling), architecture.md (the coverage tables, ASM-12), reliability.md (the model-scoped windows row, "what zone A shows"), events-and-state.md (the errors list and the session time in the details view), reference-design.md (the image) and AGENTS.md (three zones; the status) updated; grep -i "zone d" over AGENTS.md and the four design documents: 14 mentions, each historical ("until plan 23", "since plan 24", "plan 24 moved", "No zone D since plan 24"); after the documentation's review, the Codex /status note names zone A's quota windows (state/actions.go, its test), the pane's text for a window past its reset is the clock tick's ("waiting for the next update"), and reliability.md counts the five synthetic mocks
PASS 24.35 Every changed or added text file read byte by byte (scratchpad\plan24\crcheck.mjs): 121 files, 0 with a BOM, 0 with a carriage return; the image binary; git status lists the 44 changed and 6 added files of the plan and the 70 golden frames, go.mod, go.sum and THIRD_PARTY.md untouched; nothing committed
Not verified: a live session with either CLI (mock replays only, no live run approved: the real key of a model-scoped window, "seven_day_fable" or another, was never observed, and the mapping follows the documented seven_day_<model> form); a quota window reset passing live (the clock tick's path, with a fake clock); a 16-colour terminal; Windows Terminal Preview; the mouse on the real terminal (the application harness only; the terminal runs used keys); a session of hundreds of turns in the terminal (TestLongSession's generated one only)
Verdict: DONE
```

**The conformance table of the brief's section 14.**

| Requirement | Verdict | Evidence |
| - | - | - |
| Zone D removed from dashboard | PASS | 24.1 to 24.3: three boxes at every size, no "D ·" row in 156 golden frames or the terminal rows |
| Telemetry integrated into Zone A | PASS | 24.4, 24.5: the pane at A's bottom, its three forms, the sections folding |
| Orchestrator header alignment | PASS | 24.4, 24.7: the heading alone on the first row, the status ending the model row |
| Dynamic orchestrator status | PASS | 24.8, 24.9: 15 states from the state machines, "! AGENT WAITING" after a completed turn with an agent waiting; the terminal's RUNNING and THINKING |
| Stage hierarchy | PASS | 24.10 to 24.12: head rows with the time at the right, blocks for closed turns, the tree with its connectors unfolded, the cursor and the folds |
| Session Usage layout | PASS | 24.13: three columns with thin rules from 41 cells, three rows under it |
| Context progress bar | PASS | 24.15: round(used / window × width) cells in the title colour |
| Token and cost presentation | PASS | 24.13, 24.22: "session total", "API-equivalent", "cost at API rates" in the notes; the details view's "API-equiv. estimate" |
| Three quota categories when available | PASS | 24.17: the synthetic mock's three, the other mocks' two, no window the CLI did not report |
| Subscription-aware telemetry | PASS | 24.17, 24.21, 24.30: a model-scoped window only once reported; an API key or a plan without a window read N/A with the reason |
| Purple/lavender quota bars | PASS | 24.18: every window's fill in the title colour, red from 90% only; the terminal screenshots |
| Dynamic reset countdowns | PASS | 24.20, 24.29: from the reported reset time and the frame's time, "4h 12m" to "4h 11m" a minute later, N/A at the reset, "Reset time unavailable" without one |
| Accurate data/unavailable-state handling | PASS | 24.14, 24.21: N/A with its reason, never 0; no "$" before a result |
| Notes and caveats | PASS | 24.22: each part from the state, errors first, nothing hardcoded |
| Existing telemetry details preserved | PASS | 24.23: the same view, its title without "D ·", the counters added under the values, Enter or a click on TELEMETRY, Esc back |
| Keyboard/mouse navigation | PASS | 24.11, 24.23, 24.24: ↑ ↓ ← → Enter Esc, F6 and Tab, the clicks |
| Responsive layout | PASS | 24.1 to 24.5, 24.25, 24.31: 80x24 to 208x59, the pane's forms, zoom, the 120x40 screenshot |
| Zone B unchanged | PASS | 24.26: its rows the same texts re-wrapped to its width; plan 20 to 22's tests pass |
| Zone C unchanged | PASS | 24.26: its rows identical before and after; its height rule's tests adapted to the rows it gained |
| Functional regression tests | PASS | 24.26, 24.28, 24.30: the whole suite, the race detector, the mock files |
| Rendering performance | PASS | 24.27, 24.29: the timing step within its targets, the clock's redraws confined to zone A |
| Visual conformity to screenshot | PASS | 24.31, 24.32: the hierarchy, the rows and the colours of the image in the terminal; what differs is listed below |

**After the owner's live session (2026-10-22).** The owner ran the tree on
Claude Code 2.1.303 with claude-fable-5-1 (two turns, "Hello !" and "What
model are you using") and sent the screenshot: zone A as designed, the 5-hour
window 0%†, the weekly 22%†, "Resets in 4h 58m" and "4d 22h" counting, and
no "Fable limit". That is the CLI's doing, not the subscription's: the
dashboard draws the windows the stream reports, and Claude Code's
`rate_limit_event` carried `unifiedWindows` with five_hour and seven_day
only, as in every probe recorded since plan 1 (15 events of rateLimitType
five_hour, 5 of seven_day, never another key). Where Claude Code reports a
model-scoped weekly limit (claude-code.md, section 7): the documented
`rateLimitType` values `seven_day_opus` and `seven_day_sonnet`, sent only
when that limit's status changes; and the `/usage` report's `weekly_scoped`
rows with `scope.model`, which the dashboard never asks for (CC-10 q9,
closed in plan 14: a prompt in the conversation, and the endpoint can
rate-limit). Two corrections made on the spot, the suite rerun:
- the mapper dropped the event's own model-scoped window when
  `unifiedWindows` was present; it keeps it now (24.30);
- `claude-fable-5-1` read as its id in zone A's model row and zone B's
  label, the family table ending at Haiku; Fable and Mythos are added
  ("Claude Fable 5.1", 24.6).

Showing the Fable limit live needs a source the dashboard does not read:
the `/usage` report. The owner approved one probe, P24u
(`doc/research/probes/README.md`; the recording
`doc/research/probes/p24u-claude-usage-report.jsonl`; 2026-10-22, 12:39,
Claude Code 2.1.303, Haiku, the dashboard's arguments, on a temporary
project, `hash-config.sh` before and after: 24 of 25 files identical,
`~/.claude.json` rewritten by its Claude Code as always, no process left,
nothing written). What it holds:
- "/usage" sent as a user message is answered by the CLI itself, 2.2 s
  later, with no model call: `result` `num_turns` 0, `total_cost_usd` 0,
  `duration_api_ms` 0, `modelUsage` empty.
- The answer is an `assistant` message whose text is the TUI's usage page
  ("Current session: 6% used · resets Oct 22, 5:20pm", "Current week (all
  models): 24%", "Current week (Fable): 11%", then a usage breakdown) and
  whose documented `usage_report.rate_limits.limits[]` holds one row a
  window: `{kind: "session", percent: 6, resets_at, is_active: false}`,
  `{kind: "weekly_all", percent: 24, …, is_active: true}` and `{kind:
  "weekly_scoped", percent: 11, scope: {model: {display_name: "Fable"}}}`,
  each with `group`, `severity` and an ISO 8601 `resets_at`; `extra_usage`
  with `is_enabled` false.
- No `rate_limit_event` came: a session with no model call gets none, so
  the report is also the only quota a session has before its first turn.
- The exchange is in the stream as an assistant message and a result: a
  dashboard reading it must keep it out of zone B, out of zone A's turns
  and out of zone C's clearing.

Proposed as a plan of its own, then asked for at once by the owner ("VS
Code manage to show it directly and you telling me you can't do it ?"):
amendment 23, done in this plan (it reverses plan 14's decision, CC-10
q9). The Claude adapter sends "/usage" right after the handshake and after
each turn end, where `get_context_usage` is read; its `limits[]` rows
become `quota.5h` (session), `quota.7d` (weekly_all) and `quota.7d.<model>`
(weekly_scoped, "Fable limit" from `scope.model.display_name`), documented,
source "usage_report (<kind>)", with their reset times; the exchange is
never a conversation entry, a stage or a prompt; a failed read keeps the
last values and reports one error; `rate_limit_event` keeps updating the
windows between reads. Its criteria and results are under "Amendment 23"
below.

**What differs from the image, and why.**
- The image's "▾" before "A · ORCHESTRATOR" and the "›" before STAGES are not
  drawn: neither heading folds, and "STAGES · inferred from activity" with a
  mark would not fit zone A's least width.
- Each closed turn's block names the real prompt and the real tools and
  agents ("Agent · agent general-purpose"), not the image's "Agent launched"
  and "Results": the events decide the content (AGENTS.md, the conversation
  is real).
- The percentages carry "†" where the value comes from Claude Code's
  undocumented `unifiedWindows` field, as every undocumented value does.
- At 120x40 the pane takes its compact form when zone C holds an agent (one
  row of notes, no blank rows); the full form of the image needs the rows
  the owner's 208x59 gives it.
- "Enter details" at the heading's right end replaces the image's generic
  note "Values are from the provider when available": the notes say what
  applies to the values shown.

**Evidence outside the repository**, in the session's scratchpad
(`C:\Users\amolimom\AppData\Local\Temp\claude\C--javadev-projets-agent-command-center\dcc3e64d-0d35-4470-ab9a-d258f8675bae\scratchpad\`):
`term\out-p24\claude-max\` and `claude-120x40\` (the screenshots, their UI
Automation rows and `log.txt`), `term\p24demo.ps1`, `plan24\p24-live.jsonl`
(the mock with live reset times), `plan24\timing.log` and `timing2.log` (the
two runs of the timing step), `plan24\race.log` and `race2.log`,
`plan24\full.log` (the whole suite, verbose).

## Amendment 23: the usage report read

The owner's third message after the screenshots, with VS Code's "ACCOUNT &
USAGE" panel showing "Fable limit 12%" beside the session and weekly
windows: "VS Code manage to show it directly and you telling me you can't
do it ?". The read proposed above was done at once, inside plan 24 (brief
amendment 23; plan.md D14, criteria 24.36 to 24.41), on the recording of
P24u and the fake CLIs: no live session was started for it.

How it was checked:
- `internal/provider/claude/usage_test.go`: P24u replayed through the
  mapper twice, once with "/usage" as the dashboard's own write
  (`InInternal`), once as the owner's (`In`); the failure, the running
  state, the streamed part and the report's shapes on synthetic lines.
- `internal/provider/claude/usage_session_test.go`: the fake CLI (this
  package's test binary) in its new mode, through the real process plumbing:
  the read after the handshake, a turn, `Refresh`.
- `internal/app/usage_test.go`: the dashboard on the wire fake, the lines
  written in order, zone A's rows before any turn.
- Every fake CLI of the suite answers "/usage" as Claude Code does, with a
  report and a `num_turns` 0 result (`session_test.go`'s `next`,
  `approval_test.go`'s `wireClaude`, `input_test.go`'s `fakeClaude`): the
  existing tests run unchanged in what they check, their lists of written
  lines now showing the read where it goes (`TestHandshakeAndTurn`,
  `TestSetApprovalMode`, `TestAnswerQuestion`).
- Build, vet, gofmt, the whole suite, the race run on the two packages that
  changed, the timing step alone.

```text
Evaluation - Plan 24, amendment 23, the usage report read - 6 criteria: 6 PASS, 0 FAIL
PASS 24.36 TestUsageReadInternal: the recording through the mapper with the dashboard's own "/usage" gives one Sent "usage", session.ready, session.started and one telemetry.updated holding quota.5h 6%, quota.7d 24% and quota.7d.fable 11%, each documented with source "usage_report (session|weekly_all|weekly_scoped)" and its reset time (15:20:00.473808 UTC on the 22nd, 09:00:00.473852 and 09:00:00.474012 UTC on the 27th); no turn, message, unrecognised or error event; the reducer: 0 entries, 0 stages, turn Idle, session Ready with the init's model, quota.7d.fable 11 present and documented; the next running state maps to turn.started
PASS 24.37 TestUsageTypedByOwner: the same recording with "/usage" by In gives conversation.user, telemetry.updated, conversation.message.completed (its text naming Fable) and turn.completed
PASS 24.38 TestUsageReadFailed: a result with no report before it maps to one error "usage report not read: usage endpoint unavailable" and leaves no read pending, the owner's next result to turn.completed; TestUsageReadRunningState: a running state, a streamed part and an answer without a report during the read map to status.changed "usage report: reading", "streaming", "the answer holds none", the result to "read", the next running state to turn.started; TestUsageReportShapes: resets_at "soon" leaves quota.5h 50% without a reset time, "Opus 5.5" gives quota.7d.opus55, no scope gives quota.7d.scoped, kind monthly_new is dropped, scopeSlug("Fable") is "fable"
PASS 24.39 TestUsageReadAroundTurns: after session.ready the first read ends ("usage report: read") with quota.5h 6 and quota.7d.fable 11 documented, 0 entries, 0 stages, turn Idle; Send("hello") ends in turn.completed with 1 closed stage; Refresh writes get_context_usage then "/usage": context.used 15, quota.5h 7, still 1 stage and the same entries, 0 errors; TestHandshakeAndTurn's six lines byte for byte: initialize, "/usage", get_context_usage, "/usage", the prompt, the allow; TestSetApprovalMode's four: initialize, "/usage", set_permission_mode auto, default
PASS 24.40 TestUsageReads: on the wire fake, before any turn: session Ready, turn Idle, quota.5h 6, quota.7d 24, quota.7d.fable 11 documented with reset times, 0 entries, 0 stages, 0 errors, 0 unrecognised; zone A's text holds "Session (5hr)", "Weekly (7 day)", "Fable limit" and "Resets in"; written so far: initialize, "/usage"; after a turn and a prompt typed as soon as it ends: initialize, "/usage", user, get_context_usage, "/usage", user; 2 stages, 0 errors; written("user") is the two prompts
PASS 24.41 go build ./... ok; go vet ./... clean; gofmt -l . empty; go test ./... 14 packages ok (internal/app 31.5 s, internal/provider/claude 9.1 s), 1 skipped as before (TestDiskListLink); go test -race -count=1 on internal/provider/claude (24.1 s) and internal/app (132.5 s), CGO_ENABLED=1 with the winlibs gcc: ok, no race (plan24\race5.log); the timing step alone 7 of 7 (plan24\race3.log): TestKeystrokeToFrame p95 1.01 ms and 1.11 ms in the window, TestDraftKeystrokes p95 2.13 ms, TestLongSession delta-to-frame p95 4.69 ms, TestPasteToFrame max 1.20 ms, TestStreamToFrame p95 3.98 ms, TestThinkingFrame p95 3.54 ms, every target 16 ms (100 ms for the paste)
```

What the owner should know:
- The dashboard now writes "/usage" to Claude Code twice around each turn:
  once right after the handshake, before anything the owner can write, and
  once after each `result`, with `get_context_usage`. P24u showed the
  answer costs no model call (`num_turns` 0, `total_cost_usd` 0,
  `duration_api_ms` 0), so the quota is untouched; the documentation says
  the usage endpoint can itself be rate-limited, in which case the
  dashboard shows one error "usage report not read: <reason>" and keeps the
  last windows with their age.
- The exchange is kept out of zone B, out of zone A's stages and out of
  zone C; its running state and its result are statuses. A "/usage" the
  owner types is still shown as Claude Code answers it, its report read
  too.
- The windows from the report are documented, so they carry no "†"; the
  "†" stays on values `rate_limit_event`'s undocumented `unifiedWindows`
  updates between reads.

Not verified, to check in the owner's next live session (it needs no plan:
starting the dashboard on Claude Code and reading zone A before any prompt;
a live run, so with the owner's approval):
- The real CLI's answer inside the dashboard: whether a
  `session_state_changed` running line precedes it (P24u had none; the
  mapper handles both), and the windows on screen before the first turn,
  "Fable limit" among them, at the values VS Code shows.
- The read after a real turn, where `rate_limit_event` and the report both
  update the windows.

**Live check (2026-10-22, 13:51), approved by the owner.** The dashboard
built from this tree (`acc-p24.exe`) on Claude Code 2.1.303, in a new
Windows Terminal window, maximised and zoomed out three times (208x59, the
acc launcher's size); Claude Code through a test-only logging wrapper first
on that window's PATH (plan 20's, rebuilt as `claude.exe`: it runs the real
`<HOME>\.local\bin\claude.exe` and copies stdin and stdout with each line's
time); the real home, so that Claude Code is signed in; Px through the
window's environment; a fresh project `%LOCALAPPDATA%\Temp\acc-p24-claude-135136`;
`confcheck snapshot` before and `confcheck compare -cli-ran` after. No
prompt was typed: the script (`scratchpad\term\p24live.ps1`, its criteria
written before the run) read zone A through UI Automation as soon as the
session was ready, opened the details view from A, then pressed Ctrl+Q.
No terminal-hosted Claude Code or Codex session was open; none was closed.

```text
p24live claude-code  2026-10-22T13:51:38  project C:\Users\amolimom\AppData\Local\Temp\acc-p24-claude-135136
13:51:38.055 [L24.1 before] PASS  terminal CLI sessions: none; Px listening: True; claude 2.1.303 (Claude Code); snapshot exit 0
13:51:45.124 [L24.2 Ready on Claude Code, maximised, through the wrapper] PASS  ready True in 1632 ms; 59 rows x 208 columns; argv {"argv":["<HOME>\\.local\\bin\\claude.exe","-p","--input-format","stream-json","--output-format","stream-json","--verbose","--include-partial-messages","--forward-subagent-text","--permission-prompt-tool","stdio","--permission-mode","default"],"cli_pid":19484,"start_ms":1792669903365,"wrapper_pid":14456}
13:51:52.114 [L24.3 the three windows with percentages, reset times and no dagger, before any prompt] PASS  windows after Ready in 762 ms: Session (5hr) 13% '', Weekly (7 day) 25% '', 'Fable limit' 15% ''; 'Resets in' x3; note 'quota updated 5s ago'
13:51:52.154 [L24.4 no entry, no stage, no error, the status IDLE] PASS  model row '│ ✿ Claude Fable 5.1                              ○ IDLE ││ Ready to work on this project.                                                                                                                     │'; texts found that should not be: none
13:51:54.722 [L24.5 the details view lists the three windows from usage_report, 0 errors] PASS  2 F6, A focused True; title True; sources found: usage_report (session), usage_report (weekly_all), usage_report (weekly_scoped); errors '0'
13:51:58.470 [L24.6 Ctrl+Q closes the window, no process left] PASS  2,79 s; left: wrapper or CLI , new claude.exe , acc-p24 0; of the 1 processes seen under the wrapper, left 
13:51:58.482 [L24.7 the project is empty] PASS  no file
13:51:59.538 [L24.8 configuration check] PASS  exit 0; configuration  C:\Users\…\.claude\CLAUDE.md  identical | configuration  C:\Users\…\.claude\plugins\plugin-directory-cache-v2.json  identical | configuration  C:\Users\…\.claude\remote-settings.json  identical | configuration  C:\Users\…\.claude\settings.json  identical | configuration  C:\Users\…\.claude\skills\code-review-and-quality\SKILL.md  identical | configuration  C:\Users\…\.claude\skills\debug\SKILL.md  identical | configuration  C:\Users\…\.claude\skills\find-skills\SKILL.md  identical | configuration  C:\Users\…\.claude\skills\reverse-engineer-anything\SKILL.md  identical | configuration  C:\Users\…\.claude\skills\reverse-engineer-anything\references\android-applications.md  identical | configuration  C:\Users\…\.claude\skills\reverse-engineer-anything\references\connection-and-recovery.md  identical | configuration  C:\Users\…\.claude\skills\reverse-engineer-anything\references\evidence-workflows.md  identical | configuration  C:\Users\…\.claude\skills\reverse-engineer-anything\references\javascript-applications.md  identical | configuration  C:\Users\…\.claude\skills\reverse-engineer-anything\references\native-and-artifacts.md  identical | configuration  C:\Users\…\.claude\skills\reverse-engineer-anything\references\runtime-observation.md  identical | configuration  C:\Users\…\.claude\skills\security-review\SKILL.md  identical | configuration  C:\Users\…\.claude\skills\security-review\cloud-infrastructure-security.md  identical | configuration  C:\Users\…\.claude\skills\testing-strategy\SKILL.md  identical | configuration  C:\Users\…\.codex\.codex-global-state.json  identical | configuration  C:\Users\…\.codex\AGENTS.md  identical | configuration  C:\Users\…\.codex\config.toml  identical | configuration  C:\Users\…\.codex\plugins\.remote-plugin-install-staging\remote-plugin-bundle-GSsSCp\README.md  identical | configuration  C:\Users\…\.codex\plugins\.remote-plugin-install-staging\remote-plugin-bundle-OTxF4w\README.md  identical | configuration  C:\Users\…\.codex\plugins\.remote-plugin-install-staging\remote-plugin-bundle-Qu0fUA\README.md  identical | configuration  C:\Users\…\.codex\plugins\.remote-plugin-install-staging\remote-plugin-bundle-lprmnz\README.md  identical | mixed  C:\Users\…\.claude.json  changed (key-level diff follows)  sha256=a18f1d0e33cd7ed9ea4e5fe8b0011086fa25978053862f8bbbc86aa9c2928a57 | mixed  C:\Users\…\.claude.json  key cachedExperimentData  changed (state) | mixed  C:\Users\…\.claude.json  key cachedExperimentFeatures  changed (state) | mixed  C:\Users\…\.claude.json  key cachedGrowthBookFeatures  changed (state) | mixed  C:\Users\…\.claude.json  key cachedGrowthBookFeaturesAt  changed (state) | mixed  C:\Users\…\.claude.json  key cachedUsageUtilization  changed (state) | mixed  C:\Users\…\.claude.json  key claudeCodeFirstTokenDate  not compared (sensitive name) | mixed  C:\Users\…\.claude.json  key oauthAccount  not compared (sensitive name) | mixed  C:\Users\…\.claude\plugins\known_marketplaces.json  identical | session data: 1 new file(s) (count only) | configuration: PASS
13:51:59.554 [END] PASS  0 FAIL
```

What the wrapper logged (ms after the CLI's start; the usage page's text
is not reproduced):

```text
argv: ["C:\\Users\\amolimom\\.local\\bin\\claude.exe","-p","--input-format","stream-json","--output-format","stream-json","--verbose","--include-partial-messages","--forward-subagent-text","--permission-prompt-tool","stdio","--permission-mode","default"] start 1792669903365
end: {"end_ms":1792669916294,"exit":0}

== the dashboard wrote (ms after start) ==
19	control_request/initialize req_1
1458	user content="/usage"

== Claude Code answered (ms after start) ==
1382	system/hook_started
1425	system/hook_response
1458	control_response req_1 success
1463	system/session_state_changed state=running
3377	system/init model=claude-fable-5-1 permissionMode=default apiKeySource=none cwd=<HOME>\AppData\Local\Temp\acc-p24-claude-135136
3378	assistant model=<synthetic> usage_report=yes
	  limit kind=session group=session percent=13 resets_at=2026-10-22T15:20:00.161756+00:00 scope=null severity=normal is_active=false
	  limit kind=weekly_all group=weekly percent=25 resets_at=2026-10-27T09:00:00.161780+00:00 scope=null severity=normal is_active=true
	  limit kind=weekly_scoped group=weekly percent=15 resets_at=2026-10-27T09:00:00.161969+00:00 scope=Fable severity=normal is_active=false
	  extra_usage={"is_enabled":false,"monthly_limit":null,"used_credits":null,"utilization":null,"currency":null}
3379	result/success is_error=false num_turns=0 total_cost_usd=0 duration_ms=1910 duration_api_ms=0 modelUsage={}
3381	system/session_state_changed state=idle
3757	system/commands_changed

== stderr ==
<empty>
```

What it settles:
- The read works live: "/usage" went out the same millisecond the
  handshake's answer arrived (1458 ms), and Claude Code answered it with no
  model call: `system/init` (model `claude-fable-5-1`), the `assistant`
  message with the report and the `result` (`num_turns` 0, cost 0,
  `duration_api_ms` 0) within 2 ms of each other, 1.9 s after the write.
- The CLI wraps that answer in `session_state_changed` running (5 ms after
  the write) and idle (2 ms after the result), which P24u's recording did
  not show; the mapper took them as the read's statuses: no stage, no
  entry, "unrecognised events 0", "no errors" in the details view.
- Zone A before any prompt, 762 ms after "Ready to work on this project.":
  "Session (5hr) 13%", "Weekly (7 day) 25%", "Fable limit 15%", each with
  its lavender bar and its countdown ("Resets in 3h 28m", "4d 21h", "4d
  21h", right for 15:20 UTC and the 27th at 09:00 UTC), no "†", the note
  "quota updated 5s ago"; the model row "✿ Claude Fable 5.1 … ○ IDLE" from
  the read's init. The details view: `quota.5h 13% · usage_report (session)
  8s ago`, `quota.7d 25% · usage_report (weekly_all) 8s ago`,
  `quota.7d.fable 15% · usage_report (weekly_scoped) 8s ago`.
- Nothing of the exchange reached zone B ("Waiting for instructions",
  "Ready to work on this project." only), zone C ("no agents yet") or the
  stages ("stages appear as turns run").
- Ctrl+Q closed the window in 2.8 s with no process left; the project stayed
  empty; the configuration check passed: every configuration file identical,
  `~/.claude.json` changed in state keys only (`cachedUsageUtilization`,
  the experiment caches), one new session file.
- The values are the owner's at 13:51 (VS Code showed 12% on the Fable
  limit at about 12:30; the owner's sessions ran in between): the same
  windows, read from the same report.

Evidence: `scratchpad\term\out-p24-live\` (`results.txt`, the screenshots
`00-startup` to `06-back` with their UI Automation rows, `confcheck.txt`,
`before.json`, `processes-ready.txt`), `scratchpad\live\log-p24\`
(`in.jsonl`, `out.jsonl`, `in-times.txt`, `out-times.txt`, `argv.json`,
`end.json`, `err.txt`), `scratchpad\term\p24live.ps1`,
`scratchpad\live\analyze-p24.mjs`, `scratchpad\livewrap\claude\` (the
wrapper's source) and `claudebin\` (its executable and `wrap.json`).

**Evidence outside the repository**, in the session's scratchpad
(`C:\Users\amolimom\AppData\Local\Temp\claude\C--javadev-projets-agent-command-center\dcc3e64d-0d35-4470-ab9a-d258f8675bae\scratchpad\plan24\`):
`race3.log` (the timing step's third run), `race5.log` (the race run), the
patch files `usage-*.patch` and `doc-*.patch` (what was changed, block by
block).
