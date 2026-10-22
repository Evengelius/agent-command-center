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
plan 23), and its results in its evaluation file.

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
the terminal, plan 17 (amendment 11).

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

## Plan 24 - Zone A with the telemetry, zone D removed

Brief amendment 22 (2026-10-22), the owner's image
`references/03-zone-a-telemetry.webp`
([reference-design.md](doc/reference-design.md#zone-a-with-the-telemetry)):
zone A redrawn after the image and given zone D's telemetry; zone D gone;
zones B and C as plans 20 to 22 left them. Amendment 23 (2026-10-22), after
the owner's review of the screenshots: the usage report read, so the
model-scoped "Fable limit" shows live (D14, criteria 24.36 to 24.41). The
criteria below were written before the code; the owner reviews the Windows
Terminal screenshots before any commit.

### Design

- **D1 Layout.** Three zones. From 100 columns A takes 28% of the width,
  rounded, at least 36 columns (36 at 120, 44 at 156, 45 at 160, 56 at
  200, 58 at 208; the image's A is about 59 of 208, and the usage section's
  rows need the least width), B the rest, both from
  the row under the project bar to one bottom row; C the whole width under
  them, its height rule unchanged (plan 18, plan 20); the footer last.
  The rows A and B share are the height less the two bars and the footer:
  zone D's rows go to them. Under 100 columns A is one row and F6 opens it
  over B and C, telemetry included.
- **D2 Zone A's rows**, top to bottom, inside a frame whose top edge
  carries no title (the image): "A · ORCHESTRATOR" in the title colour;
  the model row; a rule; "STAGES · inferred from activity" ("PLAN ·
  observed (turn/plan/updated)" with a Codex plan); the tree, a pane that
  scrolls as before (plan 5, item 5), resting at its tail; a rule; "✦
  TELEMETRY"; "▾ SESSION USAGE" and its rows; a blank row; "▾ QUOTA
  WINDOWS" and its rows; a rule; the notes. No hint row: A's keys are in
  the footer while A has the focus, as C's since plan 20, and a held
  tree's rows out of view are in A's top edge, in the tree's own words
  ("↑ 12 ↓ 25 · End follows", plan 5), as C's are in its.
- **D3 The model row.** An icon in the model's colour, "✿" for a Claude
  model, "◆" for any other, then the model's title: a Claude id of the
  families Opus, Sonnet, Haiku, Fable and Mythos as "Claude <Family>
  <major>.<minor>" (claude-haiku-5-5 → Claude Haiku 5.5, claude-fable-5-1
  → Claude Fable 5.1; the owner's live session of 2026-10-22 read the id
  until Fable was added), an id starting with "gpt-" as "GPT-" and the
  rest (gpt-5.6-sol →
  GPT-5.6-sol), any other as the CLI gives it; "model N/A (<reason>)"
  before `session.started`. The status at the row's right end. When both
  do not fit the row, the short name of zone B's labels ("Haiku 5.5"),
  then the title cut with "…".
- **D4 The status**, from the state, never from text: the session's phase
  first: "◉ STARTING" and "◉ CLOSING" in amber, "✗ ENDED" red, "○
  CLOSED" grey; then the turn's: "! WAITING" pink while an approval or
  question waits, "◉ INTERRUPTING" amber, the spinner and "THINKING" in
  the title colour while a thinking phase is live (plan 22), the spinner
  and "RUNNING" in amber while the turn works otherwise; then, the turn
  idle, the agents: "! AGENT WAITING" pink while one waits for the owner,
  "◉ AGENT RUNNING" ("◉ 2 AGENTS RUNNING") amber while one still runs,
  so a completed turn with a background agent never reads COMPLETED; then
  the turn's outcome: "✓ COMPLETED" green, "✗ FAILED" red, "○
  INTERRUPTED" amber, "○ IDLE" grey before any turn.
- **D5 The stages.** A stage's head row: its symbol, "TURN n" in white,
  "· failed", "· interrupted", "· running" or "· waiting for approval"
  when that is its word (completed says nothing more), its elapsed time at
  the right end. Unfolded, under it as before: the prompt's first line (or
  "started by the CLI") in grey, then its tools and agents with their
  connectors, the moving dot and "○ (more as events arrive)" while it
  runs. Folded, the image's block: the same first line, then one grey row
  of the stage's items in order, a tool by its name, an agent as "agent
  <label>", each name once, joined by " · " ("Agent · Bash · Write"), cut
  at the width. A closed stage folds by default, an open one is unfolded;
  → and ← change it, and the owner's choice stays (plan 4). The cursor
  row's background covers every row of a folded stage's block.
- **D6 The telemetry pane**, fixed at A's bottom, never scrolled: it takes
  the rows its form needs, and the tree the rest, at least 6. Three forms,
  the largest that leaves the tree its 6 rows: *full*, the image's (a
  rule, the heading, the usage section's 4 rows, a blank, the quota
  section's heading, 3 rows a window with a blank between windows, a rule,
  2 rows of notes); *compact* (no blank rows, one row of notes); *tight*
  (the usage on two rows, a window on one row with its percentage and its
  countdown and no bar, one row of notes). A terminal too short for the
  tight form beside 6 tree rows leaves the tree 3.
  Each section folds with ← → on its heading ("▸ SESSION USAGE"), and the
  pane shrinks. Its three headings are nodes of zone A's cursor after the
  tree's: TELEMETRY, SESSION USAGE, QUOTA WINDOWS; Enter on any of them
  opens the details view; ← → fold the two sections. A click on a heading
  row does the same: TELEMETRY opens the details, a section toggles.
- **D7 Session usage.** Three columns from 41 inner cells, "CONTEXT",
  "TOKENS", "EST. COST" in grey capitals, the values in white, the
  support in grey: the context's bar (lavender fill on the frame's track,
  the used share of the window), "session total" under the tokens,
  "API-equivalent" under the cost ("API-equiv." under 45 cells); thin
  rules between the columns. Under 41 cells, three rows: "Context 4% of
  1M" and its bar, "Tokens 61.9k session total", "Est. cost $0.00
  API-equivalent". A value no source gave reads "N/A" with its reason as
  its support ("no result yet", "Codex reports none"), never 0; "†" after
  an undocumented value; "(derived)" goes to the notes.
- **D8 Quota windows.** One block a window, in this order: "Session
  (5hr)" (`quota.5h`), "Weekly (7 day)" (`quota.7d`), each model-scoped
  weekly window the CLI reports ("Fable limit": `quota.7d.<model>`, from
  Claude Code's `unifiedWindows` key or `rateLimitType` of the form
  `seven_day_<model>`, as `seven_day_opus` is documented; never shown
  unless reported), then any other Codex window by its duration ("90 min
  window"). A block: the label with the percentage ("17%", "†" when
  undocumented) at the right end; a bar of "█" on "░", the fill
  round(p × width), clamped to the width, lavender, red from 90%; "Resets
  in <countdown>" in grey from the reported reset time, "Reset time
  unavailable" without one. The countdown: "5d 3h" ("5d" on the hour),
  "4h 12m" ("4h"), "12m", "<1m"; it is drawn from the frame's time, so it
  changes as the clock ticks and never goes negative: at the reset the
  value becomes "N/A" (the clock tick, plan 4) and the block reads "N/A"
  with "window reset; waiting for the next update". A window the CLI
  reported N/A (an API key, a plan with no 5-hour limit, a read that
  failed) reads "N/A" with its reason on the second row and no bar; a
  status word with no percentage reads the word and "no percentage
  reported". A new report replaces the value and the bar's fill.
- **D9 The notes**: grey, "ⓘ" first, each part only when it applies,
  joined by " · ", what went wrong first: "errors N" (red), "retries N"
  and "unrecognised N" (amber) when above 0; then "quota updated <age>
  ago" (the last report's age, so a stale value shows), "context derived
  from usage", "tokens summed over threads" (Codex's derived total), "†
  undocumented field", "cost at API rates" (the column says
  API-equivalent; never the subscription's charge), "overage in use",
  "plan <plan>" and "credits <credits>" (Codex); wrapped to the form's
  rows and cut with "…"; with nothing to say, "values from the CLI's
  events". The way to the rest, "Enter details", is at the right end of
  the TELEMETRY heading, as zone D's edge said it.
- **D10 The details view** stays (ui-design.md, Zone D: Errors): opened
  from A by Enter on a telemetry heading or a click on TELEMETRY, Esc back
  to A with its cursor as it was; its title "ERRORS AND TELEMETRY", the
  "D ·" gone; its body, keys and scrolling unchanged.
- **D11 Keys.** F6 and Tab cycle B, C, A; the overlay and the footer say
  so; the footer with A focused: "↑↓ move · ←→ fold · Enter inspect or
  details · z zoom · PgUp PgDn Home End scroll · F6 zone · Esc back to the
  input · F1 keys".
- **D12 Drawing.** No new timer: the countdowns and ages follow the
  one-second clock, and a row is written to the terminal only when it
  changed (Screen.StringAfter), so a countdown that changed rewrites its
  row and no row of B or C.
- **D13 Docs.** ui-design.md (Layouts, Dashboard, Zone A, Zone D folded
  into it, Focus and keys, Colours: the 7-day bar lavender),
  architecture.md's coverage table, reliability.md (the model-scoped
  windows), events-and-state.md (the errors list's place), AGENTS.md (the
  zones, the status), reference-design.md (the image).
- **D14 The usage report (amendment 23).** The Claude adapter writes
  "/usage" as a user message right after the handshake's answer, before
  the owner can write (the answer's events are held until the line is
  written), and after each turn's `result` beside `get_context_usage`
  (`Refresh`, through the application's write queue, so a prompt typed at
  once follows it). Claude Code answers it itself, with no model call
  (P24u): the mapper takes `usage_report.rate_limits.limits[]`, `session`
  to `quota.5h`, `weekly_all` to `quota.7d`, `weekly_scoped` to
  `quota.7d.<model>` from `scope.model.display_name` ("Fable limit"),
  documented, source "usage_report (<kind>)", with their `resets_at`, and
  keeps the exchange out of the conversation, the stages and zone C: its
  running state, its streamed parts, its assistant message and its result
  are statuses ("usage report: reading", "streaming", "read"), never a
  turn or an entry; a result with no report before it is one error,
  "usage report not read: <reason>", and the last values stay with their
  age. A "/usage" the owner types is a message like any other, its report
  read too. `rate_limit_event` keeps updating the windows between reads.
  The tests' fake CLIs answer "/usage" as Claude Code does.

### Criteria

Layout (brief sections 1, 9, 10, 12):

- 24.1 At 120x40, 156x44, 160x45, 200x60 and 208x59, with the states of
  P2 (no agent) and P3b (one agent): boxes A and B both from row 2 to
  one bottom row, A 36, 44, 45, 56 and 58 columns wide, B the rest; C
  full width from the row under them; the footer on the last row; no
  fourth box; no row wider than the width (TestDashboardGeometry).
- 24.2 No dashboard frame carries "D ·", "QUOTA & TELEMETRY" or a fourth
  box: checked over every frame of TestGoldenFrames and the three mock
  files at 80x24, 120x40 and 200x60.
- 24.3 At 80x24 with no agent: A's row on row 2, B 17 rows, C 3 rows,
  both full width, the footer last; F6 on A draws the tree with its
  telemetry over B and C.
- 24.4 Zone A's rows in order, at 120x40 and 208x59 on the P3b mock's
  state: the frame's top edge without a title; "A · ORCHESTRATOR" the
  first inner row; the model row; a rule; "STAGES · inferred from
  activity"; the tree; a rule; "✦ TELEMETRY"; "▾ SESSION USAGE"; "▾
  QUOTA WINDOWS"; a rule; the notes starting with "ⓘ", the last inner
  rows.
- 24.5 Room: at 120x40 and 208x59 the tree keeps at least 6 rows, the
  telemetry its full form (the "Resets in" rows present); at 100x24 the
  tight form (one row a window, "Session (5hr)" and its percentage on one
  row) and the tree at least 3 rows (5 with two windows); with SESSION USAGE folded ("▸ SESSION USAGE")
  its three rows are gone and the tree has them.

The orchestrator (brief section 2):

- 24.6 The model row reads "✿ Claude Haiku 5.5" on the P3b mock, "◆
  GPT-5.6-sol" on the Codex mock, "model N/A (session starting)" before
  the session starts; the model's title changes with `session.started`.
- 24.7 The status ends on the model row's last cell, in zone A's second
  inner row, never on the heading row, at every width of 24.1; at 120x40
  on P3b at its Write request the row reads "✿ Haiku 5.5" and "! AGENT
  WAITING" (the short name, since the title does not fit).
- 24.8 The status for each state, text and style (TestOrchestratorStatus
  on synthetic states): starting "◉ STARTING" amber; closing "◉ CLOSING"
  amber; crashed "✗ ENDED" red; closed "○ CLOSED" grey; an approval
  pending "! WAITING" pink; interrupting "◉ INTERRUPTING" amber; a live
  thinking phase "⠋ THINKING" (the spinner's frame) in the title colour;
  running otherwise "⠋ RUNNING" amber; the turn idle with a waiting agent
  "! AGENT WAITING" pink; with one running agent "◉ AGENT RUNNING", with
  two "◉ 2 AGENTS RUNNING", amber; completed "✓ COMPLETED" green; failed
  "✗ FAILED" red; interrupted "○ INTERRUPTED" amber; no turn yet "○ IDLE"
  grey.
- 24.9 P3b at its Write request (the turn completed, the agent waiting):
  the status "! AGENT WAITING", not "✓ COMPLETED"; the P3b mock whole
  (the agent done): "✓ COMPLETED".

The stages (brief section 3):

- 24.10 On the P3b mock at 208x59, A focused: "✓ TURN 1" with its elapsed
  time at the row's end, then "  Use the Agent tool to launch one gene…"
  (the prompt's first line, cut), then the items row "  Agent · agent
  general-purpose" (the stage's items in order, each name once);
  "✓ TURN 2" then "  started by the CLI"; both folded, being closed. → on
  TURN 1 unfolds it: its rows with connectors ("├─", "└─"), the "agent
  general-purpose" row and the "Agent" tool row; ← folds it again.
- 24.11 With A focused, ↑ ↓ move the cursor through the turns, then
  TELEMETRY, SESSION USAGE and QUOTA WINDOWS, and back; the cursor row's
  background (stTreeCursor) is on the head row and on both rows under it
  while the stage is folded, on the head row alone once unfolded; Enter
  on an agent's node opens its inspection; Esc returns to A.
- 24.12 An open stage (a running turn) is unfolded with "○ (more as
  events arrive)" and the moving dot, as plan 4 drew it; once its turn
  ends it folds; an owner's → during the run keeps it unfolded after.

Session usage (brief section 5):

- 24.13 On P3b's state at 208x59 (54 inner cells): the columns "CONTEXT",
  "TOKENS", "EST. COST" on one row with two "│"; "4% of 1M†", "61.9k",
  "$0.00" under them; the context's bar, "session total",
  "API-equivalent" under those; at 120x40 (30 cells) the three rows
  "Context", "Tokens", "Est. cost" with the same values. Codex P7:
  "6% of 258.4k", "184.3k", "N/A" with "Codex reports none".
- 24.14 P2 up to its prompt: "N/A" with "no result yet" for the context,
  the tokens and the cost; no "0" and no "$0" in the section; the Codex
  state before a model call: "N/A" with "no model call yet".
- 24.15 The context bar's fill is round(used / window × width) cells of
  "█" in the title colour: 2 of 50 for P3b's 3.5%.

Quota windows (brief section 6):

- 24.16 P1 line 3 (Claude): "Session (5hr)" with "22%†" at the row's end,
  a bar 11 of 50 cells filled, "Resets in …" from 1792109400; "Weekly (7
  day)" with "45%†", 23 of 50 cells (22.5 rounded away from zero); no
  third window. P6b line 7
  (Codex): "Session (5hr)" "0%" and "Weekly (7 day)" "0%", no "†", no
  third window.
- 24.17 The synthetic mock `claude-quota-windows.jsonl` (P3b with its
  rate_limit_event carrying `unifiedWindows` five_hour 0 resetting at
  1792071720, seven_day 0.17 and seven_day_fable 0 resetting at
  1792499400, the image's values): three windows, "Session (5hr)" "0%†",
  "Weekly (7 day)" "17%†" with 9 of 50 cells filled, "Fable limit" "0%†"
  with an empty track ("†": unifiedWindows is undocumented); at the test clock (2026-10-15 09:30 UTC) "Resets
  in 4h 12m", "Resets in 5d 3h", "Resets in 5d 3h". The Codex mock shows
  no "Fable" row; the P3b mock, whose event carries no seven_day_fable,
  shows none either.
- 24.18 The bars are lavender (stTitle) at 22%, 45% and 89%, red
  (stFailed) at 90% and 100%; the fill is 0 cells at -5% and the whole
  width at 150%; no "▰" glyph remains on the dashboard.
- 24.19 A second telemetry event with quota.5h at 60% changes the row to
  "60%" and its fill to 30 of 50 cells; the earlier 22% is gone
  (replaced, not added).
- 24.20 The countdown: 5d 3h 20m → "5d 3h"; 5d 0h 20m → "5d"; 4h 12m 30s
  → "4h 12m"; 4h 0m → "4h"; 12m 30s → "12m"; 45s → "<1m"; a frame an
  hour later reads "3h 12m" for the "4h 12m" window; a window with no
  reset time: "Reset time unavailable"; the clock past the reset: "N/A"
  and "window reset; waiting for the next update", nothing negative.
- 24.21 N/A states: Codex with an API key: both windows "N/A" with "API
  key: ChatGPT sign-in needed", no bar; a plan with no 5-hour limit:
  "Session (5hr)" "N/A" with "plan has no 5-hour limit" and the weekly
  window "12%"; a failed read: "N/A" with "read failed: upstream timeout;
  last 0% 1m 00s ago"; Claude with an API key: "API key: no subscription
  window"; a status word alone: "allowed_warning" and "no percentage
  reported".

The notes (brief section 7):

- 24.22 P3b: the notes read "ⓘ quota updated 0s ago · context derived from
  usage · † undocumented field · cost at API rates" over two rows at
  208x59 (a row cut with "…" where it does not fit); with 2 errors they
  start "ⓘ errors 2" in red; P6b (Codex): "plan plus" present, no "cost"
  part; nothing hardcoded: a state with no value gives "ⓘ values from the
  CLI's events" alone; the TELEMETRY heading ends with "Enter details".

The details view and the keys (brief sections 8, 12):

- 24.23 F6 to A, ↓ to TELEMETRY, Enter: the details view ("◀ back (Esc)   ERRORS AND
  TELEMETRY", "N errors · last 50 kept"), its errors newest first and its
  values with their sources as before (TestDetailsKeys adapted); Esc
  returns to the dashboard with A focused and the cursor on TELEMETRY; a
  click on the "✦ TELEMETRY" row opens it as well; a click on "▾ SESSION
  USAGE" folds the section, a second click unfolds it.
- 24.24 F6 cycles B → C → A → B and Shift+F6 the reverse; Tab likewise
  outside the editor; the overlay reads "next / previous zone (B, C, A)"
  and "in A on TELEMETRY, the errors and every value"; the footer with A
  focused reads D11's line.
- 24.25 z on A draws A alone under the bars with its telemetry, in the full
  form from 120x40 (at 80x24 its 19 inner rows take the tight form), z or
  Esc restores the exact frame (TestZoom adapted to A and C).

Regression and performance (brief sections 10, 11, 12):

- 24.26 `go build ./...`, `go vet ./...` exit 0, `gofmt -l .` prints
  nothing, `go test ./...` exit 0 with its count; the golden frames
  regenerated and reviewed: every dashboard frame has three boxes, and
  zone C's rows at 120x40 and 200x60 read as before except where B's
  width moved a cut; zone B's rows keep their texts, re-wrapped to its new
  width.
- 24.27 The keystroke timing step passes its targets
  (TestKeystrokeToFrame, TestDraftKeystrokes, TestKeystrokeCost,
  TestLongSession, TestThinkingFrame, TestPasteToFrame, TestStreamToFrame).
- 24.28 `go test -race ./internal/app ./internal/render ./internal/state`
  exit 0.
- 24.29 Two frames of the three-window mock a second apart differ only in
  rows of zone A (the quota's age), a minute later likewise, the countdown
  changed ("Resets in 4h 11m"); no row of B or C differs either time.
- 24.30 The provider's rate-limit mapping: a `rate_limit_event` whose
  `unifiedWindows` holds `seven_day_fable` yields `quota.7d.fable`
  (undocumented, †) with its reset time; `rateLimitType` "seven_day_opus"
  with a utilization yields `quota.7d.opus` (documented); TestMockFiles
  regenerates the mocks unchanged but for the new synthetic one.

Visual evidence (brief section 13):

- 24.31 Windows Terminal, the application built from this tree, a mock
  replay in a temporary project, no agent CLI started: screenshot A, the
  completed session on the three-window mock, maximised at 9 pt (208x59):
  "A · ORCHESTRATOR", the model row ending with "✓ COMPLETED", the stages,
  SESSION USAGE, the three windows with "Resets in", C full width, no "D
  ·" row (the rows read by UI Automation); screenshot B, the replay
  caught while its turn runs: the status "RUNNING" or "THINKING" on the
  model row; screenshot C, the details view reached by F6, End, Enter;
  screenshot D, the same at 120x40, no overlap, every section readable.
- 24.32 In the UI Automation rows of 24.31 the status's last character is
  on the column before zone A's right edge, so "✿" and "◆" each took one
  cell; otherwise the glyph is replaced and the run repeated.
- 24.33 The conformance table of brief section 14, every row PASS or FAIL
  with its evidence, in `doc/plan24-evaluation.md`.

Documents and hygiene:

- 24.34 ui-design.md, architecture.md, reliability.md, events-and-state.md,
  reference-design.md, AGENTS.md updated as D13 says; `grep -n "zone D"`
  over doc and AGENTS.md finds only historical mentions naming the plan
  that removed it.
- 24.35 Every changed text file UTF-8 without BOM with LF line endings;
  the image binary; `git status` lists only the intended files, and
  nothing is committed until the owner approves the screenshots.

The usage report (amendment 23, D14):

- 24.36 `TestUsageReadInternal` (`internal/provider/claude/usage_test.go`):
  P24u through the mapper with the dashboard's own "/usage" gives
  `quota.5h` 6%, `quota.7d` 24% and `quota.7d.fable` 11%, documented,
  source `usage_report (<kind>)`, with their reset times, and no turn,
  message, unrecognised or error event; the reducer keeps no entry and no
  stage, the turn idle, the session Ready with the init's model, and the
  next running state is a turn's again.
- 24.37 `TestUsageTypedByOwner`: the same recording with "/usage" as the
  owner's message gives the user message, the completed message naming
  Fable, the turn's end, and the windows.
- 24.38 `TestUsageReadFailed`, `TestUsageReadRunningState`,
  `TestUsageReportShapes`: a result with no report before it is one error
  naming the reason and leaves no read pending; a running state and a
  streamed part during the read are statuses, an answer without a report
  too; an unparsable `resets_at` leaves the window without a reset time,
  an unnamed scope reads `quota.7d.scoped`, an unknown kind is dropped.
- 24.39 `TestUsageReadAroundTurns` (the fake CLI): the first report is in
  the state before any turn (6%, Fable 11% documented), a turn adds one
  stage and no error, and `Refresh` writes `get_context_usage` then
  "/usage", whose report (7%) replaces the first with no new stage or
  entry; `TestHandshakeAndTurn` and `TestSetApprovalMode` list the lines
  written, byte for byte, "/usage" right after `initialize`.
- 24.40 `TestUsageReads` (`internal/app/usage_test.go`): over the wire
  fake, zone A shows "Session (5hr)", "Weekly (7 day)", "Fable limit" and
  "Resets in" before any turn, with no entry, stage, error or unrecognised
  event; the lines written are `initialize`, "/usage", the prompt,
  `get_context_usage`, "/usage", the next prompt; two stages after two
  prompts.
- 24.41 `go build ./...`, `go vet ./...`, `gofmt -l .` empty, `go test
  ./...` all packages ok with every existing test (the fakes answer
  "/usage"), `go test -race` on `internal/provider/claude` and
  `internal/app` clean, the timing step alone 7 of 7.
- 24.42 The live check the owner approved (2026-10-22, 13:48): the
  dashboard built from the tree on Claude Code 2.1.303, in a temporary
  project outside the repository, no prompt typed, the configuration check
  before and after; its eight criteria (L24.1 to L24.8: the preconditions,
  Ready through the logging wrapper, the three windows with percentages,
  reset times and no "†" before any prompt, no entry, stage or error and
  the status IDLE, the details view naming `usage_report` for the three
  windows with 0 errors, Ctrl+Q leaving no process, the project empty, the
  configuration check PASS) written in `scratchpad\term\p24live.ps1` before
  the run, every one PASS, in `doc/plan24-evaluation.md`, "Live check".

### Progress

- [x] Amendment 22 recorded, the image in references, this plan written.
- [x] Zone A: the header, the stages, the telemetry pane; zone D removed;
  the layout widths; the adapter's model-scoped windows
  (`internal/render/zonea.go`, `telemetry.go`, `tree.go`, `dashboard.go`,
  `text.go`, `select.go`, `keys.go`; `internal/provider/claude/mapper.go`).
- [x] The keys, the mouse, the details view, the footer and the overlay
  (`internal/app/keys.go`, `inspect.go`, `mouse.go`, `pane.go`, `app.go`).
- [x] Tests: the new ones (`internal/render/zonea_test.go`,
  `internal/app/zonea_test.go`, the synthetic mock
  `testdata/mock/claude-quota-windows.jsonl` and its golden frames), the
  adapted ones, the golden frames regenerated and read.
- [x] Build, vet, gofmt, tests, the timing step, the race run
  ([`doc/plan24-evaluation.md`](doc/plan24-evaluation.md), 24.26 to 24.28:
  the step's first run failed beside a terminal replay and found a per-frame
  scan of the tools, fixed; the second run alone 7 of 7).
- [x] Windows Terminal screenshots and their comparison with the image
  (24.31, 24.32; the differences listed in the evaluation).
- [x] Documents; the evaluation file with the conformance table: 35 of 35
  PASS, waiting for the owner's review of the screenshots before any
  commit.
- [x] The owner's live session (2026-10-22, Fable 5.1) showed no "Fable
  limit": the stream's `rate_limit_event` carries five_hour and seven_day
  only. The mapper keeps a model-scoped window beside them now, and Fable
  and Mythos are family names. P24u, the `/usage` probe the owner approved
  (12:39, `doc/research/probes/README.md`): answered locally with no model
  call, its `usage_report` holding the session, weekly and "Fable" windows
  ([`doc/plan24-evaluation.md`](doc/plan24-evaluation.md), "After the
  owner's live session"); the read followed as amendment 23.
- [x] The usage report read (amendment 23, D14): the Claude adapter and
  its mapper, `Refresh` through the application's write queue, the tests
  (24.36 to 24.41; every fake CLI answers "/usage"), the documents; the
  whole suite, the race run and the timing step rerun
  ([`doc/plan24-evaluation.md`](doc/plan24-evaluation.md), "Amendment
  23").
- [x] The live check the owner approved (13:48; 24.42): the dashboard on
  Claude Code 2.1.303, zone A read before any prompt, 8 of 8 PASS, "Fable
  limit 15%" on screen 762 ms after Ready, the report answered with no
  model call, the configuration check PASS
  ([`doc/plan24-evaluation.md`](doc/plan24-evaluation.md), "Live check").
  Waiting for the owner's approval before any commit.

## Closed plans

One entry each: what the plan did, its approval, its evaluation file and
its commits. Their criteria and progress logs: `git show 8888f59:plan.md`
(plans 1 to 13), `git show 84d2ad8:plan.md` (plan 14), `git show
b047434:plan.md` (plan 15), `git show becfb34:plan.md` (plan 16), `git show
f00c2d8:plan.md` (plan 17), `git show 8b3b16b:plan.md` (plan 18), `git show
d84e5d4:plan.md` (plan 20), `git show f939d96:plan.md` (plan 21), `git show
d13ef71:plan.md` (plan 22), `git show e8d3e3b:plan.md` (plan 23).

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
