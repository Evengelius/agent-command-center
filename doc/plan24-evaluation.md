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

Proposed, as a plan of its own (it reverses plan 14's decision, CC-10 q9):
the Claude adapter sends "/usage" after the handshake and after each turn
end, where `get_context_usage` is read; its `limits[]` rows become
`quota.5h` (session), `quota.7d` (weekly_all) and `quota.7d.<model>`
(weekly_scoped, "Fable limit" from `scope.model.display_name`), documented,
source "usage_report", with their reset times; the exchange is never a
conversation entry, a stage or a prompt; a failed read keeps the last
values with the reason, as Codex's does; `rate_limit_event` keeps updating
the windows between reads. Until then a model-scoped window appears only
when a `rate_limit_event` carries it, which none has.

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
