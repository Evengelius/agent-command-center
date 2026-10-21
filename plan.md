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
d84e5d4:plan.md` for plan 20, `git show f939d96:plan.md` for plan 21), and
its results in its evaluation file.

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

## Plan 22 - A live thinking indicator

**Started on 2026-10-21** at the owner's request ("Ok onto the next one."),
recorded word for word as brief amendment 18, its three images described.
Zone B gets a live thinking card with its animations, a summary row for
each thinking phase that opens on its details, and an activity line while
the turn works, for Claude Code and Codex, from what each CLI reports.
Zones A, C and D do not change. No commit before the owner has reviewed
the Windows Terminal demonstrations (amendment 18, section 13). No live
run: the recordings settle what each CLI reports and the demonstrations
replay them; the live checks this leaves open are proposed at the end, not
run.

### What the CLIs report

Read from every recording in `doc/research/probes` with two scripts (each
thinking block or reasoning item, its times, the estimates and counts
around it); then an independent agent tried to refute each claim over all
recordings (7 held as written, 7 reworded below, 1 not recorded), and the
documentation was checked (sources in `doc/research/claude-code.md` and
`doc/research/codex-cli.md`). `testdata/probes` holds byte-identical
copies.

Claude Code: 2.1.300 to 2.1.303, the four 2.1.303 recordings P303a to
P303d among them; every one runs Haiku 5.5 (the standing decision: live
runs use Haiku).

- **C1 Start and end, observed.** A `stream_event` `content_block_start`
  of type `thinking` and the `content_block_stop` of its index: 33 blocks
  in 14 recordings, main thread only, one per API message, always at index
  0 and first; no `redacted_thinking`. The text is always empty (101
  `thinking_delta` lines) and the lines carry `"thinking_display":
  "updates"`: the CLI sends no thinking text. 22 of the 55 messages have no
  thinking block.
- **C2 A live estimate, not a count.** `system` `thinking_tokens`, 90
  lines: `estimated_tokens` cumulative, the previous plus
  `estimated_tokens_delta`, strictly increasing, starting at 50 in each
  block 0.15 to 0.4 s after its start, then a step about every 1 to 1.9 s;
  a last, uneven value at the block's stop when there is a remainder (21
  of 27 blocks); none for 4 blocks under 0.3 s; no agent field, and never
  outside an open main-thread block. P12c-block's second phase: 50, 350,
  650, 850, 1,100, 1,450, 1,700, 1,950, 2,200, 2,450, 2,700 over 13.9 s.
  Documented: the Agent SDK's `SDKThinkingTokensMessage`, "a running
  estimate of the thinking tokens generated so far in the current block"
  (zone D's † on it is out of date). The undocumented `thinking_display`
  and a `thinking_delta`'s `estimated_tokens` are not used.
- **C3 The exact count, per API message.** The `message_delta` event's
  `usage.output_tokens_details.thinking_tokens`, in 55 of 55 messages (0
  with no thinking), after the message's last block, 0 to 2.4 s (median
  0.6 s) after its thinking ended. One thinking block per message: the
  exact count of that phase (P303b: 719, 281, 268). Documented: "When
  streaming, this breakdown appears only on the final `message_delta`
  event" (the Messages API's extended-thinking page).
- **C4 The turn's total.** The `result` line's
  `usage.output_tokens_details.thinking_tokens`, the sum of the turn's
  main-thread messages' in 31 of 31 results (P303b: 1,268 = 719 + 281 +
  268; its subagent's 180 not included). `modelUsage`'s `thinkingTokens`
  is the session's, subagents included: never a phase's.
- **C5 The estimate against the count.** A block's last estimate is from
  14% under (1,050 for 1,218) to nearly four times (198 for 52) the exact
  count: short blocks over, long ones a little under.
- **C6 Subagents.** With `--forward-subagent-text` (the adapter's), their
  thinking comes only in `assistant` lines with `parent_tool_use_id`, no
  stream event and no estimate: not zone B's.
- **C7 An interrupted message.** `message_delta` with `stop_reason` null
  and `thinking_tokens` 0 (P303a line 76; P2 line 76); the turn's result
  zero.
- **C8 No thinking budget** is reported. A resume replays no history (P4).

Codex CLI 0.163.0 (11 recordings) and 0.145.0 (P6, P6b, P7); gpt-6-sol,
reasoning effort high, reasoning summaries not asked for.

- **X1 No live thinking.** 48 reasoning items: each `item/started` and
  `item/completed` 0 to 10 ms apart, `startedAtMs` and `completedAtMs`
  within 3 ms, `summary` and `content` empty; no `item/reasoning/*`
  notification. The item says the model reasoned in its step; it does not
  say when the reasoning began. The schema has the summary notifications
  and `turn/start`'s `summary` (auto, concise, detailed, none). Asked for in
  P22s (Proposed, below), a reasoning item starts 2.5 to 3.4 s before it
  completes, while the model works; its summary's text does not stream.
- **X2 The exact count, per model step.** `thread/tokenUsage/updated`
  `last.reasoningOutputTokens`, once per model response and thread; in
  0.163.0 just after the step's tool items completed (4 to 126 ms), so
  0.2 to 6.9 s after its reasoning item; `last` is the difference of two
  consecutive `total`s (60 of 60 updates in 0.163.0), so `last` never
  counts twice. Reasoning tokens with exactly one reasoning item since the
  thread's previous update in 47 of 47 (P12x-sandbox: 10, 16, 47, 249, 14,
  46; P7's orchestrator: 83, 8, 7). 28 of 76 steps had no reasoning item
  and 0 reasoning tokens. No update ever comes after the next step began.
- **X3 No live count, no budget.**
- **X4 Child threads** report their own usage under their thread id: not
  zone B's.
- **X5 A step's start** is not reported: after the previous step's usage
  update, 0.8 to 4.1 s of silence until the next reasoning item (P7: the
  usage at 13,471 ms, after its `wait` call completed at 13,448; the
  reasoning item at 14,910). The README says a resume replays the usage; no
  recording has one.

### Design

The choices amendment 18 leaves open, as built; the owner may change any at
the review.

1. **One conversation entry per thinking phase**, role `thinking`, in its
   place among the turn's entries, with its times, the CLI's estimate, the
   exact count and its scope (`state.Thought`). Every animation is a
   function of those times and the frame's time: the frame never writes
   the chronology.
2. **The live card**, in the CLI's block of the exchange, in the violet of
   thinking: the circular spinner, the playful expression, the elapsed time
   at the right; the count line; the bar. Seven rows (bordered, blank rows
   between, the mock-up of section 2) when zone B's conversation has 20
   rows or more, five from 12, two without a border below (80x24); the
   size does not change while a phase lasts unless zone B does.
3. **Claude Code: the card is the observed phase.** The count line shows
   the CLI's estimate, "≈ 650 thinking tokens · live estimate" (mode B,
   the scale 0–2k, doubling as the estimate passes it: 2k, 4k, 8k…), and
   "Thinking · token usage pending" before the first estimate (mode C).
   The summary row shows the estimate, marked as one, until the message's
   exact count comes, then the count.
4. **Codex: the card is the model step.** No thinking is reported live, so
   the card shows while the model works on a step, "Model working ·
   reasoning tokens reported when the step ends", the bar indeterminate
   (mode C), from the step's start: the turn's message recorded, or the
   previous step's response complete and its tools finished (X5). A
   reasoning item turns it into "✓ Thought for up to 4.1s" (the step's
   time up to the reasoning's end: an upper bound), and the step's usage
   adds "· 16 reasoning tokens"; a step whose first output is not
   reasoning collapses away and leaves no row. The Codex adapter reports
   each step's start, inferred; the reducer stays provider-independent.
5. **Mode A**, a known budget: "Thinking · 1,284 / 4,096 tokens", the fill
   proportional. No supported CLI reports a budget: tested, never shown.
6. **Animations.** Entrance 280 ms: the frame's top edge swept by a light,
   then the spinner's row, the count's and the track, each from dim to
   full. The bar: after each new value the fill grows over 300 ms (ease
   out) with its leading edge lit, settling over 400 ms more; between
   values a shimmer crosses the filled part, the fill unchanged; mode C's
   segment sweeps the track every 1.6 s. The spinner: six frames, 100 ms
   each. The expression: every 4 s from the owner's ten, never twice in a
   row, its first 200 ms dimmed. Exit 300 ms: ✓ in place of the spinner and
   the duration in place of the expression, lit, then the bar's row and the
   count's close, to the summary row in muted colours. Frames every 50 ms
   while one of these runs, the 100 ms spinner otherwise.
7. **Only what was drawn live animates.** A phase whose start and end come
   in one batch (a replay, a resume) is drawn as its summary row at once.
8. **The summary row**, among the tool rows: "▸ ✓ Thought for 2.8s · 719
   thinking tokens", its end time and origin at the right. → opens its
   details (the times, each value's source and scope), ← folds it, Enter
   shows it whole; the entry cursor stops on it. "✗ Thinking interrupted
   after 0.2s" when the turn was interrupted.
9. **The activity line**, while the turn runs with no live card, nothing
   streaming and no request waiting: the spinner, an expression, the
   turn's time at the right ("00:24"), and "◇ 1 subagent working" while
   agents run. A request waiting stops every animation of the turn.
10. **Image 1's line**: the editor's placeholder reads "Claude Code is
    processing your request…" while a turn runs and the draft is empty.

### Criteria

Recordings (`internal/provider`, `internal/state`; the recordings replayed
through the adapters, times from `fixture.Epoch`).

- 1.1 Claude Code's mapper: each `content_block_start` and
  `content_block_stop` marker carries its block index; each
  `message_delta` marker its `thinking_tokens` when the line has it
  (P303b: 719, 281, 268, in order).
- 1.2 Codex's mapper: a `step_started` marker, origin inferred, at the
  orchestrator's `userMessage` item, and once a step's response is
  complete (its usage) and its tool items have finished, when the step
  made a call (a tool item or a spawn); none after a step that made none.
  P7: 5 markers, at 3,380, 7,120, 9,632, 13,471 and 24,076 ms; the
  reasoning tokens of each usage update as a metric, by thread (P7's
  orchestrator: 83, 0, 8, 7, 0).
- 1.3 Claude Code's phases: one `thinking` entry per orchestrator thinking
  block, none for a subagent's. P12c-block: 4 entries; their durations
  3,476, 13,902, 56 and 2,161 ms; their estimates as C2 lists for the
  second; exact counts 717, 2,852, 17, 354. P303b: 3 entries, 719, 281,
  268, and none for the subagent's 4 thinking blocks.
- 1.4 Never counted twice, never mixed: the sum of P303b's phases is 1,268,
  the `result`'s total; the turn total never adds to a phase; an estimate
  lower than the last leaves the phase's unchanged (a synthetic event); an
  estimate with no open phase changes none; a message with two thinking
  blocks (synthetic) puts its count on the second, with "over 2 phases".
- 1.5 Codex's phases: P7's orchestrator: 3 thoughts, "up to" 3,140,
  3,743 and 1,440 ms (from the step's start to the reasoning item's
  completion; a step that made a spawn and no reasoning stays open into
  the next), counts 83, 8, 7; 1 step dropped (the last, a message); no
  count from a child thread on any of them; a usage update goes only to
  the thoughts since the previous one of its thread.
- 1.6 Ends: a message ended with `stop_reason` null marks its phases
  interrupted (P303a: "interrupted", no exact count, the CLI's 0 kept
  apart as reported; corrected on 2026-10-21 by correction C1, the owner's
  choice, from "0 thinking tokens, as reported"); a
  turn interrupted ends an open phase with the event's time and
  "interrupted", a failed turn or a crash with "failed"; an open Codex
  step ends dropped.
- 1.7 A replay in one batch: P12c-block applied as one batch, then frames
  at +0, +100 and +300 ms: every phase drawn as its summary row, no card,
  no entrance or exit.

Rendering (`internal/render`, at chosen times; golden frames at 80x24,
79x24, 120x40 and 200x60).

- 2.1 The live card, mode B: P12c-block's second phase at 11,000 ms (it
  began at 7,449; the estimate 650 came at 10,610, its growth over by
  10,910), 120x40, seven rows as design 2 and 3 say: "00:03" at the right
  of the expression's row, "≈ 650 thinking tokens · live estimate", the bar
  filled on 650/2,000 of its width, its right label "2k" (a golden frame).
  Corrected on 2026-10-21 from 10,700 ms, where the fill still grows from
  350 (design 6), as the rendering's agent found.
- 2.2 The bar: fill = width × count / scale, to half a cell ("╸"); the
  scale 2,000 doubled until the count fits (2,200 reads 4k, 1,950 reads
  2k); the label of a phase never goes down; at the end of a phase the
  fill is the last value's, never full because it ended (a table test).
- 2.3 Mode A with a budget of 4,096 and 1,284: "Thinking · 1,284 / 4,096
  tokens", the fill 1,284/4,096 of the width; no adapter sets a budget.
- 2.4 Mode C: before C2's first estimate, "Thinking · token usage
  pending"; a Codex step, "Model working · reasoning tokens reported when
  the step ends"; the segment at its positions for 0, 0.4, 0.8 and 1.2 s;
  no token count drawn.
- 2.5 Growth: an estimate from 300 to 650 at T: the fill f(300) at T,
  between at T+150 ms, f(650) from T+300 ms; the edge lit until T+300 ms,
  normal from T+700 ms; frames between two estimates show the same fill.
- 2.6 The count only shows reported values: frames every 10 ms across a
  change show 300 or 650, nothing between; the number lit for 400 ms after
  a change.
- 2.7 The spinner: six frames, the frame at Now−Start of 0, 100, …, 600 ms
  cycles through all six and repeats.
- 2.8 The expressions: the owner's ten, word for word; a new one every 4 s;
  over 200 s none twice in a row; dimmed for its first 200 ms.
- 2.9 Entrance: for a phase drawn live, frames at 0, 70, 140, 210 and
  280 ms reveal the edge, the spinner's row, the count's, the track; the
  card's row count is final from the first frame.
- 2.10 Exit: at End the spinner is ✓ and the expression the duration; at
  End+100 and End+200 ms fewer rows; from End+300 ms the summary row only;
  no animation for a phase not drawn live (1.7's replay).
- 2.11 The summary rows, word for word: "✓ Thought for 2.8s · 719 thinking
  tokens"; "✓ Thought for 2.8s · ≈ 650 thinking tokens (estimate)" before
  the exact count; "✓ Thought for 0.1s" with none; "✓ Thought for up to
  3.1s · 83 reasoning tokens"; "✗ Thinking interrupted after 0.2s"
  (corrected on 2026-10-21 by correction C1 (a), the owner's choice, from
  "… · 0 thinking tokens").
- 2.12 The details: → opens them under the row (the times, each value's
  source and scope), ← folds; Enter shows the phase whole; the entry
  cursor stops on a summary row, never on a live card; the folds of tool
  runs count only calls (plan 19's tests pass unchanged).
- 2.13 The activity line: P303b while its subagent works: the spinner, an
  expression, the turn's time, "◇ 1 subagent working"; none while a
  request waits (P303b's approval), while text streams, or once the turn
  ended.
- 2.14 Sizes and resize: seven, five and two rows by zone B's room; no row
  wider than its zone; a card drawn at 120x40, then the same instant at
  80x24, equals the 80x24 golden.
- 2.15 Focus and scrolling: 50 frames of animation with the view held
  (scrolled up) change none of `Focus`, `ConvTop`, `ConvHold`, `Entry`.

Application (`internal/app`).

- 3.1 Ticks: 50 ms frames while a card is live or a transition runs, the
  100 ms spinner otherwise; none once the turn is idle and every exit has
  ended (`Options.After` recorded); `ui.Spinner` counts only the 100 ms
  ticks.
- 3.2 An idle dashboard draws nothing: no frame scheduled with the turn
  idle and no transition left (the test reliability.md's test 9 names,
  not found by plan 22's survey, added).
- 3.3 The mock replays: `claude-thinking.jsonl` (from P12c-block) and
  `claude-thinking-subagent.jsonl` (from P303b), by TestMockFiles;
  `codex-subagents.jsonl` (P7) shows its three thoughts.

Checks and regressions.

- 4.1 `go build ./...`, `go vet ./...` exit 0; `gofmt -l .` empty; `go
  test ./...` 0 FAIL, only TestDiskListLink skipped; `go.mod` unchanged.
- 4.2 The timing step, 6 of 6 within targets, plus TestThinkingFrame: a
  live card animated over the long session, frame p95 ≤ 16 ms.
- 4.3 Golden frames: only those with a thinking block or a running turn
  change (each diff read); the new scenarios' frames at the four sizes.
- 4.4 Windows Terminal, maximised then 120x40: screenshots of A to F
  (amendment 18, section 11), and for A to C and F a sequence of frames
  50 ms or less apart made into an animated GIF; the look listed as not
  verified until the owner has seen it.
- 4.5 Documents: ui-design.md (zone B's thinking), events-and-state.md,
  architecture.md (ASM-18's frame rate, ASM-22, the provider-id table),
  reliability.md (the fallbacks), claude-code.md and codex-cli.md (the
  findings above, each with its source and documented or not); plan.md;
  `doc/plan22-evaluation.md`; UTF-8 without BOM, LF only.
- 4.6 `git status` lists only the intended files; no commit before the
  owner's review.

### Proposed, not run

- A live run with Opus 5.5 (the owner's model) on Claude Code: one prompt
  that thinks, to see whether Opus also sends `thinking_display`
  "updates", the estimates and the exact counts that Haiku sends.
- Codex with reasoning summaries asked for at `turn/start`, to see whether
  a reasoning item then starts when the reasoning does: the smallest
  change that would make Codex's thinking observable live. A change to how
  the dashboard drives Codex: the owner's decision.

**The two as probes**, proposed on 2026-10-21 at the owner's request ("Let's
do: The two proposed live checks"). Neither runs before the owner approves
this proposal (standing decision: live runs). What both share:
- An empty temporary folder outside the repository.
- The configuration check (`drivers/hash-config.sh`) before and after.
- No CLI session open in a terminal.
- The CLI through Px, by the environment it inherits.
- Each recording sanitized by `drivers/sanitize.mjs`, the organisation's
  name taken from the recording itself and never typed, then kept in
  `doc/research/probes`.

P22o, Opus's thinking, on Claude Code 2.1.303:
- How: a new driver `drivers/p22-claude.mjs` with `p303-claude.mjs`'s
  arguments (the dashboard's), and `ANTHROPIC_MODEL=claude-opus-5-5` for
  this process only. That is Opus where the standing decision says Haiku,
  which is the check's purpose. One prompt that needs reasoning and no tool:
  "Without using any tool, find the smallest positive integer that leaves
  the remainders 1, 2, 3, 4 and 5 when divided by 2, 3, 4, 5 and 6. Think it
  through, then reply with the number and one line of justification." Any
  tool request is denied. Only if that turn has no thinking block, a second
  prompt asks it to think hard about another question.
- Usage: one Opus turn, two at most, over a cached context of about 25k
  tokens, with 1 to 5k output tokens, thinking included. The 5-hour quota is
  read before and after from the CLI's own rate-limit events.
- Stops: after the turn's result; at 180 s; at any request other than a
  tool's, denied.
- Recorded, not judged against Haiku's numbers:
  - O1 the session's model (`system/init`) and the CLI's version;
  - O2 a main-thread thinking block, its start and stop;
  - O3 `"thinking_display":"updates"` on its stream events, or not;
  - O4 `system/thinking_tokens`: how many, the first one's delay after the
    block's start, the steps (multiples of 50 or not), the interval, and the
    last value against the exact count;
  - O5 the exact count in the final `message_delta`
    (`usage.output_tokens_details.thinking_tokens`), or none;
  - O6 replayed through the dashboard's adapter and reducer in a scratch
    copy: no unrecognised event, the phase's estimates and count as
    recorded;
  - O7 the configuration hashes before and after (`~/.claude.json`
    excepted, Claude Code's own state, as documented); the quota used.

P22s, Codex's reasoning summaries, on Codex CLI 0.163.0:
- How: a new driver `drivers/p22-codex.mjs` with `p12-codex.mjs`'s
  handshake and thread (the dashboard's: read-only, approvals on request),
  `turn/start` with `summary`: `auto` on turn 1, `detailed` on turn 2
  (`TurnStartParams.summary`, schema line 3772). The prompts:
  - turn 1: "Without running any command or editing any file: how many
    trailing zeros does 100! have? Reason it through, then reply with the
    number and one line.";
  - turn 2: "Without running any command or editing any file: in how many
    ways can 10 be written as an ordered sum of 1s and 2s? Reason it
    through, then reply with the number and one line.".

  Every approval is declined. `account/rateLimits/read` runs at the start
  and at the end.
- Usage: two Codex turns at its configured model and effort (gpt-6-sol,
  high), a few thousand tokens each; the 5-hour window read before and
  after.
- Stops: after turn 2 completes; at 300 s; on a refused `turn/start`, which
  is recorded and not retried.
- Recorded:
  - S1 `turn/start` with `summary` accepted, or its error;
  - S2 for each reasoning item, its `item/started` to `item/completed`, set
    against 0 to 10 ms without summaries; the
    `item/reasoning/summaryTextDelta`, `summaryPartAdded` and `textDelta`
    notifications, how many and when, against the step's start and the
    item's end;
  - S3 the question itself: does the reasoning item now start when the
    reasoning does, its `item/started` before its summary deltas and well
    before its completion? Yes or no;
  - S4 each step's `last.reasoningOutputTokens`, as before;
  - S5 replayed through the dashboard's adapter in a scratch copy: what it
    does with the summary notifications today;
  - S6 the configuration hashes; the quota used.

After both runs:
- The findings go to claude-code.md and codex-cli.md, section 12, and here.
- Nothing in the dashboard changes in this step. If S3 is yes, sending
  `summary` and drawing Codex's reasoning live is proposed to the owner
  with its design. If it is no, the indeterminate bar stays.

**Results, 2026-10-21**, both run as approved, in 17 s each
(`doc/research/probes`, `p22o-claude-opus-thinking.jsonl` and
`p22s-codex-reasoning-summary.jsonl`; claude-code.md and codex-cli.md,
section 12).

P22o (18:51):
- O1: `claude-opus-5-5`, CLI 2.1.303, one turn, thinking at the first
  prompt.
- O2: one main-thread thinking block, 2,796 to 10,807 ms (8.0 s).
- O3: `"thinking_display":"updates"` on 8 of its 10 lines, as on Haiku's
  blocks.
- O4: six estimates, 50 to 850, the first 508 ms into the block (Haiku: 146
  to 392 ms), then every 1.0 to 1.6 s, each step a multiple of 50. No last,
  uneven value. The last estimate is 8.4% under the exact count.
- O5: the exact count, 928, in the final `message_delta`, 967 ms after the
  block's stop.
- O6: replayed, one observed phase, estimate 850, count 928, "Thought for
  8.0s · 928 thinking tokens". Three unrecognised events, none about
  thinking: `system/hook_started` and `hook_response` (the owner's
  `SessionStart` hook) and `system/post_turn_summary`. Zone D counts them
  ("unrecognised events").
- O7: 21 of 22 configuration files identical; `~/.claude.json` rewritten by
  the probe's Claude Code (hash only). After the turn, 5-hour 9% and 7-day
  14%: the CLI sends no reading before its first request. 1,050 output
  tokens, 928 of them thinking, $0.42 API-equivalent.

P22s (18:52):
- S1: `summary` accepted, `auto` and `detailed`.
- S2: the reasoning items span 2,506 and 3,436 ms (0 to 10 ms without
  summaries), each starting about 1.6 s after its turn's `userMessage` item.
  `auto` gave no summary. `detailed` gave one heading, "**Computing
  f(10)**", 9 ms before its item completed.
- S3: yes, with a limit. The item now starts while the model works, so the
  reasoning becomes observable live. Its text does not stream, and its span
  may include making the summary (28 and 36 reasoning tokens over 2.5 and
  3.4 s).
- S4: one usage update per step, 28 and 36 reasoning tokens, as before.
- S5: replayed as today, each step is a phase "up to" 4.1 s and 5.0 s. The
  item's start confirms the step, so the card already reads "Thinking"
  from it. `summaryTextDelta` maps to a hidden thinking delta;
  `summaryPartAdded` is unrecognised.
- S6: Codex's configuration files identical; the 5-hour and weekly windows
  0% before and after; nothing left running, nothing written in either work
  folder.

**Proposed from these results, for the owner's decision:**
- Codex: send `summary: "auto"` with every `turn/start`. With no other change
  to the drawing, the card then reads "Thinking" from the reasoning item's
  start, about 1.6 s into the step, for the 2.5 to 3.4 s it lasts, instead
  of at its very end. With it:
  - the summary row keeps its "up to" time;
  - the details give the item's own start and end, and the summary's heading
    when Codex sends one, marked as Codex's text;
  - the mapper recognises `summaryPartAdded`.
  
  It is a change to how the dashboard drives Codex (plan 22, X1).
- Claude Code: map `system/hook_started`, `hook_response` and
  `post_turn_summary` as known events with nothing drawn. Today each session
  adds 2 to zone D's "unrecognised events", and each turn 1.

### Amendment 19: Codex through the local proxy

The owner's request during this plan (doc/brief.md, amendment 19). It is
built, not yet committed, and is to be committed apart from the thinking
indicator:
- its own files: `internal/input/config.go` and `config_test.go`;
  `internal/provider/codex/provider.go`, `session.go` and `proxy_test.go`;
  `cmd/agent-command-center/main.go`; `internal/render/startup.go`;
  `internal/app/codexproxy_test.go`;
- its documentation: `doc/brief.md` amendment 19, and parts of
  `ui-design.md`, `architecture.md`, `reliability.md`, `AGENTS.md` and
  this file;
- `internal/app/app.go` and `internal/render/render.go` hold hunks of both:
  stage those by hunk with a filtered patch (`git apply --cached`), or
  commit plan 22 first.

**Design.**
- `config.json` (ASM-7) gains one setting, `{"codex": {"proxy": "<URL>"}}`.
  The URL is `http://` or `https://` with a host, for example
  `http://127.0.0.1:3128` for Px.
- When it is set, the dashboard starts Codex with `HTTPS_PROXY` and
  `HTTP_PROXY` set to it:
  - every inherited value of either name, in any case, is replaced;
  - every other variable is inherited as today;
  - the commands Codex runs inherit them too.
- Claude Code is unchanged: the dashboard gives it no proxy. Its
  environment and `~/.claude/settings.json` both carry Px since 2026-10-21
  (AGENTS.md, Environment).
- Absent, it changes nothing: Codex inherits the dashboard's environment.
- `input.LoadCodexProxy` reads the same file, only reads it, and is
  independent of the keys:
  - A problem in `codex` (a wrong type, an unknown setting in it, a value
    that is not such a URL, or a URL with credentials) sets no proxy and
    is named on the startup screen, under the keys' problem: `config.json:
    …; no proxy for Codex`.
  - A problem with the file as a whole (unreadable, not JSON, an unknown
    setting at its top level) is already named by the keys' problem, and
    it sets no proxy.
  - The startup screen shows the proxy in force on a grey line: "Codex
    through the proxy http://127.0.0.1:3128 (config.json)".
- A resume after a crash starts Codex the same way.

**Criteria.**
- 19.1 With `{"codex":{"proxy":"http://127.0.0.1:3128"}}`, Codex's spawn
  environment equals the dashboard's environment:
  - every `HTTPS_PROXY` and `HTTP_PROXY` entry is removed, in any case;
  - `HTTPS_PROXY=http://127.0.0.1:3128` and
    `HTTP_PROXY=http://127.0.0.1:3128` are appended;
  - every other entry is unchanged and in the same order.
- 19.2 No file, no `codex`, or no `proxy`: no environment is given
  (inherited), as today. Existing tests and golden frames are unchanged.
- 19.3 Claude Code's spawn environment is unchanged with the setting.
- 19.4 These values are refused, each with no proxy and one problem naming
  `codex.proxy`, or `codex` for the last two:
  - `""`, `"127.0.0.1:3128"`, `"socks5://127.0.0.1:1080"`, `"http://"`;
  - `"http://user:secret@127.0.0.1:3128"`, where the message never shows
    `secret`;
  - a number or a list as the value;
  - `"codex": "x"`, and an unknown setting inside `codex`.

  The keys still load from that file.
- 19.5 A problem with the whole file gives no proxy and no second problem
  line; the keys' problem stays as in plan 21.
- 19.6 The startup screen shows the `codex` problem as a ✗ line under the
  keys' problem, and the proxy in force as a grey line.
- 19.7 A resume after a crash spawns Codex with the same environment.
- 19.8 The file is only read: none is created or changed.
- 19.9 The documentation states all this, kept current:
  - ui-design.md, the `config.json` paragraph;
  - architecture.md, the start's step 1, ASM-7 and the Codex adapter;
  - reliability.md, a proxy that does not answer (Codex's own errors in
    zone B, nothing hidden).
- 19.10 Checks:
  - `go build ./...`, `go vet ./...` and `gofmt -l .` pass;
  - `go test ./...` passes twice, and `-race` passes;
  - the golden frames are unchanged.
- 19.11 Live check, with the owner's approval: the owner's `config.json`
  set, Px running, the dashboard started with Codex on a temporary
  project, and no turn. Codex's connections go only to 127.0.0.1:3128.

### Corrections C1 to C12

The owner's decision of 2026-10-21 on the investigation
(`doc/plan22-investigation.md`, sections 2 and 5): "Apply the recommended
corrections C1 to C12, with C1's wording (a) and without the optional C13.
Then re-verify, including the demonstrations the corrections change."

**Design**, where section 5 leaves a choice:
- C1 is made in the state, not only in the drawing. A message cut short
  (`stop_reason` null) gives its phase no exact count. The 0 the CLI sends
  with it is kept apart, as reported, so nothing that reads a count shows
  it as one, the exit's count row included.
- C2, C3 and C4 as the investigation's verified copy
  (`verify-state\bundle.diff`), C5 as its copy `fix-f4r`.
- C6: the application notes when the activity line's condition began, by
  the function the drawing uses. The line shows from 200 ms after, drawn
  by the 100 ms spinner, with no new tick.
- C8: the exit's count row reads the exact count once it came ("2,852
  thinking tokens", short "2,852 tokens"), else the live text without
  "live" ("≈ 2,700 thinking tokens · estimate", short "≈ 2,700 tokens ·
  estimate").
- Added at the review of the first build, before the code that follows
  them (the implementer's open points):
  - C7 completed: when neither form fits beside the exit's head (a Codex
    phase interrupted or failed at 80x24), the 2-row card leaves the count
    out of that row, never cut, as D7 says "Codex's too".
  - C8 completed: in its exit, a phase whose count can no longer come (cut
    or closed) with no estimate reads "Thinking · no exact count reported"
    (short "Thinking · no count"), not "token usage pending" (P2); a shared
    count "1,268 thinking tokens over 2 phases", as the summary row says it.
  - C4's details: a phase it ended says "when the next step began
    (inferred)", not "when its turn ended".
  - C9: toolLine's left and right parts become two helpers that the summary
    row's check uses too, so the rule cannot drift.

**Criteria.**
- 5.1 C1, P303a's cut message: its phase interrupted, with no exact count,
  the CLI's 0 kept as reported; its summary row "✗ Thinking interrupted
  after 0.2s" with nothing after it; its details "no exact count: the CLI
  reports 0 for an interrupted message" and "live estimate 50"; no frame of
  its exit shows "0 thinking tokens". P2 the same (golden `stream-p2-06`).
- 5.2 C2, a phase whose turn or session ended with no exact count takes
  none later:
  - P12x-sandbox cut at line 61 by a crash, line 57 replayed after the
    resume: its phase has no count and reads "no exact count reported"
    (was "up to 4.1s · 10 reasoning tokens"); P9n the same (was "0
    reasoning tokens");
  - a turn with no prompt (synthetic) never shares an earlier turn's
    phase: no "over 2 phases";
  - in P6b no row reads "exact count not reported yet" once its turn ended
    (was 1 of 66).
- 5.3 C3: a phase live while the turn is Idle (synthetic) ends when a
  prompt is sent, at the last event's time, its end inferred; no 50 ms
  frame once its exit is over.
- 5.4 C4: a confirmed reasoning item still live when the next Codex step
  starts (synthetic) ends at that start, its end inferred: never two live
  phases; its details say it ended when the next step began.
- 5.5 C5: a reasoning item that finds no step open starts at its turn's
  last usage update. P9a line 24's phase reads "Thought for up to …s · 118
  reasoning tokens" and its exit is timed (no "--:--"); P6b's second entry
  gains a start; no other recorded phase changes.
- 5.6 C6: the activity line shows once its condition has held 200 ms.
  Every recording replayed with a frame every 50 ms: no frame draws it in
  the gaps D3 lists (the 17 Codex turn ends, 27 to 145 ms; Claude Code's 0
  to 27 ms; before and between step cards; around approvals); after a
  prompt, not before +200 ms; P303b's subagent at work still shows it
  (2.13); no new tick (TestThinkingTicks and TestIdleDrawsNothing
  unchanged).
- 5.7 C7: P303a at 80x24, the 2-row card's exit row is never cut ("live
  esti…" gone): the count's form is chosen against the exit's head; a
  Codex phase interrupted or failed (synthetic) leaves its count out
  rather than cut.
- 5.8 C8: the exit's count row as the design says; never "live estimate"
  or "token usage pending" once a count is reported (P303a, P303e, P2), nor
  "token usage pending" when none can come (P2's cut phase); a shared
  count "over 2 phases" (synthetic); Codex's unchanged.
- 5.9 C9: at 80 columns a thought's summary row drops its origin whole when
  its right part does not fit: no cut origin ("obse…") in any frame of any
  recording at 80x24; tool rows unchanged.
- 5.10 C10: zone B held, only a dropped step new: revealing an entry in
  view scrolls nothing (the reproduction's ConvTop 217 stays 217); reveal
  and the "↓ N new" badge count alike.
- 5.11 C11: TestThinkingReplayInOneBatch reads `ui.Live` by `Thought.ID`,
  and fails when markLive marks every phase.
- 5.12 C12, as section 5 lists: codex-cli.md section 10 item 4 (the
  resume's usage replay documented for 0.145.0, present in 0.163.0's
  source, fork included, never recorded); "4 to 126 ms" where "4 to 87"
  was (codex-cli.md section 12, this file); claude-code.md section 12 (the
  relay of the API's per-delta estimate, the last value's fit, an
  interrupted message's usage its message_start's).
- 5.13 Each correction's test fails with its fix undone, one at a time.
- 5.14 `go build ./...` and `go vet ./...` (and `-tags timing`) exit 0;
  `gofmt -l .` empty; `go test ./...` twice, 0 FAIL, only TestDiskListLink
  skipped; `-race` 0 races; `go.mod` and `go.sum` unchanged; the timing
  step 7 of 7 within targets, on mains.
- 5.15 Golden frames: only those the corrections change, each diff read:
  C1's (`stream-p2-06`), C6's (an activity line drawn under 200 ms), C7 to
  C9's at 80x24.
- 5.16 Documents: ui-design.md (Thinking: the interrupted row, the details,
  the exit's count row, the activity line's 200 ms, the summary row at 80
  columns); events-and-state.md (the closed and the cut phase, the ends of
  C3 to C5); criteria 1.6 and 2.11 corrected here with a note; UTF-8
  without BOM, LF only.
- 5.17 Windows Terminal, maximised then at 120x40: the demonstrations whose
  frames the corrections change, run again, the earlier ones kept; P303a's
  interrupted phase at 80x24 too, for C1, C7 and C9, from a mock made in a
  scratch copy (not added to the repository). The look stays not verified
  until the owner has seen it.
- 5.18 `git status` lists only the intended files; no commit.

### Amendment 20: Codex's reasoning summaries, Claude Code's known events

The owner's request after the two live checks (doc/brief.md, amendment 20),
the two proposals of their results, both taken.

**Design.**
- **Codex's turns.** Every `turn/start` the dashboard sends carries
  `"summary": "auto"` (`TurnStartParams.summary`; P22s: accepted).
  `thread/start` and `thread/resume` are unchanged, and nothing else is
  sent.
- **Codex's adapter.**
  - `item/reasoning/summaryPartAdded` is known and gives no event: the text
    comes in its delta and in the item's completion.
  - `summaryTextDelta` stays a thinking delta, never drawn as text.
  - A reasoning item's `item/completed` marker carries the item's summary
    as Codex sent it, its parts joined by a blank line, in a new
    `event.Marker` field.
- **The state.** A Codex phase keeps when its reasoning item started
  (`item/started`, observed) and the item's summary. Its start stays the
  step's, inferred, and its time stays "up to".
- **The card** does not change. The item's start confirms the step, so the
  card reads "Thinking · reasoning tokens reported when the step ends" from
  then until the item completes: 2.5 and 3.4 s in P22s, instead of 0 to
  10 ms.
- **The details of a Codex phase:**
  - the step's line, as today;
  - a new line, "the reasoning item from <start> to <end> · observed", when
    its start is known;
  - the count, as today;
  - "no live estimate: Codex reports reasoning tokens when the step ends",
    in place of "… Codex reports reasoning once it ended", no longer true;
  - "Codex's summary, its own text: **Computing f(10)**" when Codex sent
    one, else "no reasoning summary: Codex sent none".
- **Claude Code's adapter.**
  - `system/hook_started` and `system/post_turn_summary` are known and give
    no event.
  - `system/hook_response` gives none when its `outcome` is "success" and
    its `exit_code` 0. Otherwise it gives an observed notice, drawn in zone B
    in amber like the CLI's own notes, naming the hook, its exit code and the
    first line of its stderr. A failing hook stays visible (AGENTS.md, keep
    failures visible), and the session goes on, as the CLI does.
  - Changed at the review of the first build, before its code: the design
    first said "an observed error". A SessionStart hook's lines come before
    the initialize response, and any error during the handshake ends the
    session (`reduce.go`, the handshake rule). A failing `pxw` hook would
    then have stopped a session that Claude Code itself carries on.

**Criteria.**
- 20.1 Every `turn/start` the Codex session sends carries `"summary":
  "auto"`, its other parameters unchanged; `thread/start` and
  `thread/resume` unchanged.
- 20.2 P22s through the Codex adapter: no unrecognised event (was 1,
  `summaryPartAdded`). The second reasoning item's completion marker carries
  "**Computing f(10)**", the first's nothing.
- 20.3 P22s through the reducer:
  - two phases, their reasoning items started at 6,882 and 11,397 ms and
    completed at 9,388 and 14,833 ms;
  - their starts the steps', inferred, 5,252 and 9,849 ms;
  - their summaries none and "**Computing f(10)**", their counts 28 and 36;
  - the summary rows unchanged, "Thought for up to 4.1s · 28 reasoning
    tokens" and "… up to 5.0s · 36 reasoning tokens".
- 20.4 P22s replayed in the application: at 6,000 ms the card reads "Model
  working · …"; at 8,000 ms, between the first item's start and its end,
  "Thinking · reasoning tokens reported when the step ends".
- 20.5 The details, word for word: the item's line and the new "no live
  estimate" line for both phases; the summary line for the second; "no
  reasoning summary: Codex sent none" for the first.
- 20.6 Recordings without summaries are drawn as before: every golden frame
  unchanged, unless a Codex phase's details are open in it (each such diff
  read).
- 20.7 P22o through the Claude Code adapter: no unrecognised event (was 3);
  its conversation unchanged. A hook response with outcome "error", exit
  code 1 and a stderr (synthetic) gives one observed notice naming the hook,
  the code and the stderr's first line; one with "success" and 0 gives
  nothing. P22o with its hook made to fail: the session reaches Ready,
  never crashed, the notice drawn in zone B before the prompt (changed with
  the design at the review, above).
- 20.8 The two recordings are copied to `testdata/probes`, and the replay
  tests take them with no unrecognised event.
- 20.9 Each change's test fails with the change undone, one at a time.
- 20.10 Checks: build, vet (also `timing`, `live`), gofmt; go test ./...
  twice in the owner's environment; `-race` 0 races; the timing step 7 of
  7; go.mod and go.sum unchanged.
- 20.11 Documents:
  - ui-design.md: Codex's card and details;
  - events-and-state.md: the phase's reasoning item and summary, and the
    marker;
  - architecture.md: the Codex adapter's `turn/start`;
  - reliability.md: Codex's thinking, a failing hook;
  - codex-cli.md and claude-code.md: what the dashboard now sends and maps;
  - plan.md.

  All UTF-8 without BOM, LF only.
- 20.12 Windows Terminal, maximised then 120x40: P22s replayed from a mock
  made in a scratch copy, not added to the repository. The card reads
  "Thinking" from the item's start, and the details show the summary. The
  look stays not verified until the owner has seen it.
- 20.13 `git status` lists only the intended files; no commit.
- Proposed, not run, with the owner's approval only: a live run of the
  dashboard with Codex, one turn on a temporary project, to see the card read
  "Thinking" from the reasoning item's start.

**The live check L20**, approved on 2026-10-21: "run the proposed live
dashboard check with Codex, one turn on a temporary project, as plan.md's
amendment 20 criteria describe, then record the findings".
- How:
  - The dashboard built from this tree, in a new Windows Terminal window,
    maximised.
  - Codex 0.163.0 through a test-only wrapper first on that window's PATH.
    It passes every argument through, adds no override, and logs the
    app-server's stdin and stdout byte for byte, each line with the time it
    arrived.
  - The real home, so that Codex is signed in; Px through the window's
    environment and the owner's `config.json`.
  - An empty temporary project outside the repository; the configuration
    check (confcheck) before and after; no CLI session open in a terminal.
  - One prompt, which asks Codex to reason without running a command or
    editing a file. A card, should one come, is denied.
  - UI Automation reads the screen from the prompt to the turn's end, each
    read timed, with a screenshot at each change of the card's line. Once
    the turn has ended, the window is also shown at its default size,
    without another turn.
- Criteria:
  - L20.1 Before: no Claude Code or Codex session in a terminal; Px
    listening on 127.0.0.1:3128; `codex --version` 0.163.0; the snapshot
    taken.
  - L20.2 The dashboard reaches Ready on Codex in the temporary project,
    maximised. The wrapper logs the app-server session, with the
    dashboard's own arguments.
  - L20.3 The `turn/start` the dashboard sends carries `"summary":"auto"`
    (the wrapper's stdin log).
  - L20.4 The turn ends (`turn/completed`, completed) with no approval
    card. Zone D reads "errors 0" on screen. The session's recording,
    replayed through the Codex adapter and the reducer, gives 0
    unrecognised events.

    Corrected on 2026-10-21, after the run, at the owner's request, from
    "Zone D reads "unrecognised events 0" and "errors 0"". Zone D's two-row
    form, maximised here and at 120x30, draws no such counter.
  - L20.5 The card, against the wrapper's times:
    - every read from the card's count row's appearance, 140 ms after the
      step's start, to the reasoning item's `item/started` reads "Model
      working · reasoning tokens reported when the step ends", and none
      reads "Thinking" before it.

      Corrected on 2026-10-21, after the run, at the owner's request, from
      "every read from the step's start". The card's 280 ms entrance draws
      its count row from 140 ms, the third of its four 70 ms stages
      (`render/thinking.go`, `entrance`);
    - the first read after `item/started` reads "Thinking · reasoning
      tokens reported when the step ends", within 250 ms of it;
    - every read from then to the item's `item/completed` reads
      "Thinking".

    A turn without a reasoning item leaves L20.5 not verified rather than
    failed: it is Codex's choice, and is recorded as such.
  - L20.6 After the turn:
    - the phase's row reads "Thought for up to <t>s · <n> reasoning tokens",
      where n is the turn's `last.reasoningOutputTokens`;
    - its details (→) give "the reasoning item from <start> to <end> ·
      observed", each time within 50 ms of the wrapper's `item/started` and
      `item/completed`;
    - they also give "Codex's summary, its own text: …" with the summary
      `item/completed` carries, or "no reasoning summary: Codex sent none"
      when it carries none.
  - L20.7 At the default size, after the turn: every row is closed by its
    border, and the phase's row and details are drawn.
  - L20.8 Ctrl+Q closes the window, and no Codex or dashboard process is
    left. The project is empty. The configuration check passes:
    configuration files identical, Codex's new session files allowed.
- Recorded, not judged: where the item starts in the step and how long it
  lasts; whether `auto` gave a summary; the turn's length; the quota
  windows before and after.

### Progress

- 2026-10-21: the request recorded as amendment 18; the recordings read by
  script, an agent checking the claims, the documentation checked; this
  plan written.
- 2026-10-21: the contract written here (`event.Marker`'s `Index`,
  `Thinking`, `HasThinking`; `event.StepStarted`, `ThinkingEstimate`,
  `ReasoningLast`; `state.Thought` and `Entry.Thought`; `render.UI.Live`),
  `go build ./...` exit 0. Building: the adapters and the mock files, then
  the reducer (an implementer agent each, Sonnet); zone B's rendering in
  parallel (an implementer agent, Opus). At 01:26 the adapters' and the
  reducer's agents stopped on `ERR_PROXY_TUNNEL` (the corporate proxy
  refused a tunnel), the adapters' having written nothing; both restarted
  at once, the rendering's agent unaffected. The documentation check
  added to `doc/research/claude-code.md` and `codex-cli.md`, section 12.
- 2026-10-21: built and reviewed here. The rendering (`internal/render/thinking.go`,
  look, cards, content, dashboard, viewer, palette; 18 tests): two fixes
  after review, a card in its exit never folded away in a run of calls and
  no activity line under a closing card (TestThinkingFoldKeepsCard, a case
  of TestThinkingActivity; each fails without its fix); criterion 2.1's
  moment corrected to 11,000 ms. The adapters (`steps` in the Codex
  mapper, the block indices and `message_delta` counts in Claude Code's;
  P7: 5 markers, reasoning 83, 0, 8, 7, 0; two mock files added, seven
  regenerated): one fix after review, only a spawn counts as a call, not a
  child's end told on its parent (TestSubAgentActivityIsACall). The reducer
  (`marker` and the thinking section of `reduce.go`; 13 tests): two fixes,
  a force stop ends a live phase (interrupted) and the owner's close too,
  a crash failed (TestThinkingEnds). Then the integration (the application,
  golden frames, the frame-time test), an implementer agent (Sonnet).
- 2026-10-21: the integration's agent stopped on `ERR_PROXY_TUNNEL` after
  `markLive`, the 50 ms `animMsg` and the → / ← keys (kept, reviewed); the
  rest done here. `internal/app/thinking_test.go`: TestThinkingFrames (8
  scenarios at the 4 sizes, 32 frames, recordings replayed with their own
  times), TestThinkingTicks, TestIdleDrawsNothing (reliability.md's test 9,
  missing until now), TestThinkingHoldsView, TestThinkingReplayInOneBatch,
  TestThinkingKeys, each but the frames checked by a mutation that makes it
  fail; TestEntryViewer updated for the thought's stop of the cursor; 32
  frames rewritten, each diff read (thought rows, the placeholder, the †
  of `thinking.estimate`, two mock files in the listings). Found and fixed:
  a mock replay at speed 0 emitted from a goroutine, so where the batches
  fell, and which phase a frame drew live, changed from run to run
  (TestColourProfiles and TestGoldenFrames failed 2 runs in 6); now every
  event is queued before the session starts (`mock.Provider.Start`), 7 runs
  of 7 pass. TestThinkingFrame (timing tag): p95 3.42 ms in a 5,522-entry
  session (a first run, not the timing step). `go test -race ./...`: 14
  packages ok, no race.
- 2026-10-21: verified by three read-only agents (a verifier, a review of
  the logic, a review of what the screen claims, the last stopped by the
  proxy): 28 PASS, 1 FAIL, 2 not verified, and 9 findings; each fixed here
  with its test (`doc/plan22-evaluation.md`, The independent
  verification). Then: `go test ./...` 14 ok twice, `-race` 14 ok; the
  timing step 7 of 7 on mains (TestThinkingFrame p95 3.33 ms,
  TestLongSession 4.52 ms); Windows Terminal, the workstation unlocked by
  the owner: the three mock replays maximised and at 120x40, 26
  screenshots and 6 GIFs, a first run before the fixes and a final one
  after; 31 of 31 PASS. Waiting for the owner's review of the
  demonstrations; no commit.
- 2026-10-21: the owner asked for a read-only investigation before
  finalising: the Claude Code estimate, the Codex usage updates, defects,
  regressions, verification. It is reported in `doc/plan22-investigation.md`.
  - The estimate is the CLI's sum of the API's per-delta increments, and
    it is always labelled as one, except an interrupted message's 0 (D1).
  - The Codex adapter and reducer are correct on all 76 recorded updates.
    Reasoning usage comes only after a step ends: the indeterminate bar
    stays.
  - Defects: 13 worth correcting, one medium (D2, a resumed thread's
    replayed usage), 1 optional, 4 not.
  - No regression: every baseline test, the golden frames zone by zone,
    2,953 snapshots of 23 recordings, the timing step 7 of 7, `-race`
    with 0 races.
  - Corrections C1 to C13 wait for the owner's approval. Nothing in the
    application changed; no commit.
- 2026-10-21: amendment 19, Codex through the local proxy.
  - Built by an implementer agent (Sonnet) and reviewed here.
    - `input.LoadCodexProxy` reads `config.json`'s `codex.proxy`.
    - `codex.Provider.Proxy` gives the Codex it starts `HTTPS_PROXY` and
      `HTTP_PROXY`; with no setting there is no environment, as before.
    - The startup screen shows a red problem line and a grey proxy line.
  - Checks: gofmt clean; build and vet exit 0; `go test ./...` 14 ok,
    twice by the agent and once here; `-race` 14 ok with 0 races; the timing
    step 7 of 7; the golden frames unchanged.
  - Criteria 19.1 to 19.8 and 19.10 PASS. 15 faults put in by the agent
    and 2 put in here each failed a test.
  - 19.9: the documentation (ui-design.md, architecture.md, reliability.md,
    AGENTS.md's environment).
  - The owner's own setup, outside the repository:
    - Px's upstream in `%APPDATA%\px\px.ini`;
    - `~/.claude/settings.json` with the four proxy keys and a
      `SessionStart` hook that starts `pxw`;
    - the global CLAUDE.md's Px section rewritten without the company's
      name;
    - `config.json` created with `codex.proxy`, at the owner's request.
  - 19.11 PASS, live, with the owner's approval. A throwaway test with the
    `live` tag, in a scratch copy, started Codex 0.163.0 the dashboard's
    way: the proxy read from the owner's real `config.json` by
    `input.LoadCodexProxy`, in an empty temporary project outside the
    repository, with no turn.
    - The spawn carried `HTTPS_PROXY` and `HTTP_PROXY` set to
      `http://127.0.0.1:3128`.
    - The handshake took 2.6 s. The rate limits came back through Px 4.4 s
      after the start.
    - netstat saw `codex.exe` connect only to 127.0.0.1:3128, and nowhere
      else.
    - Codex closed with exit 0, and its 6 processes were gone.
    - confcheck PASS: every configuration file identical, 0 new session
      files.
  - No commit.
- 2026-10-21: the check after Claude Code's restart, from a new Claude
  Desktop session, read-only.
  - This session's Claude Code (Claude Desktop's own, 2.1.304) connected to
    the corporate proxy directly during 150 s of sampling, and never to Px.
  - Its tool shells carry `HTTPS_PROXY` and `HTTP_PROXY` equal to the
    Windows user variables, not the settings' four keys; Internet Options
    sets no proxy. In a Claude Desktop session the keys do not take effect.
  - Px runs (`pxw`, started at 14:17, before this session) and listens on
    127.0.0.1:3128 only.
  - The owner approved the global setup, applied the same day:
    - `setx` pointed the Windows user variables `HTTPS_PROXY` and
      `HTTP_PROXY` at `http://127.0.0.1:3128`; `NO_PROXY` is unchanged
      (the same hash before and after);
    - `px --install` added Px's logon start, `pxw.exe` with `--config` set
      to `%APPDATA%\px\px.ini`, naming no proxy;
    - the `SessionStart` hook is kept as a safety net;
    - the global CLAUDE.md's Px section and AGENTS.md's Environment are
      rewritten to match, with no company name.
  - Left to check once Claude Desktop is quit and reopened: from a new
    session, its Claude Code connects to 127.0.0.1:3128 only. Done: the
    next entry.
- 2026-10-21: that check, from a new Claude Desktop session after the
  restart, read-only: PASS.
  - Claude Desktop, now the Microsoft Store app 3.1623.0.0, restarted at
    15:38. This session's Claude Code is the app's own CLI 2.1.304 (PID
    29068, started at 15:39), found as the parent of the tool shells. It
    runs from `%LOCALAPPDATA%\Claude-Data\claude_pzs8sxrjxfjjc\claude-code\2.1.304\`;
    `%APPDATA%\Claude\claude-code` does not exist.
  - The shells carry `HTTPS_PROXY` and `HTTP_PROXY` equal to
    `http://127.0.0.1:3128`, as do the Windows user variables.
  - Sampled for 251 s, 15:51:43 to 15:55:54, while it worked: turns, a
    WebFetch, a call to a remote MCP connector, 21 s idle. Two samplers
    ran side by side:
    - the TCP and UDP tables, IPv4 and IPv6, every 65 ms: 3,855 samples;
    - the documented `Get-NetTCPConnection -OwningProcess 29068`: 152
      snapshots.
  - Every connection went to 127.0.0.1:3128, and none anywhere else:
    - 18 connections, one open since 15:39:06 and 17 of 0.3 to 0.7 s;
    - connected to Px in 3,855 samples of 3,855; no UDP socket;
    - the cmdlet's 159 `Bound` rows have no remote end: each is the local
      port of one of these connections.
  - Px held the other end of each, in every sample: its server
    `pythonw.exe` (PID 21352, a child of `pxw.exe`). Px opened one
    connection to the corporate proxy for each, 18, and none elsewhere.
    The script matched the address against `px.ini`'s `server` line and
    never printed it.
  - The session's children opened no connection: the tool shells, git, a
    child `claude.exe`, and the `rea` MCP server (`npx`, then `node`). A
    corporate alert client was left out: it predates the session, and its
    stale parent PID had been reused by that `npx`.
  - Not verified: a connection shorter than the poll (no ETW trace without
    elevation); paths the window did not use, such as a sign-in refresh.
  - The global CLAUDE.md's Check named `%APPDATA%\Claude\claude-code` for a
    Desktop session's `claude.exe`. At the owner's request it now names the
    Store app's path.
- 2026-10-21: the session-start checks after the restart, read-only.
  - The handoff written before the restart was acted on and deleted at
    15:11 by the session that ran the first Px check. Its open points are
    all recorded here or in `doc/plan22-evaluation.md`.
  - `claude` 2.1.303 and `codex` 0.163.0 are the last probed; Claude
    Desktop's own CLI is 2.1.304. HEAD is still f939d96: nothing committed.
  - The demonstrations for the owner's review are in a temporary folder,
    `%TEMP%\claude\C--javadev-projets-agent-command-center\a5552254-6236-4074-b92d-9ffa1ad3272e\scratchpad\term\out-p22`
    (`out-p22-first` beside it: the run before the fixes).
  - The owner's Windows user name, left for the owner to decide
    (`doc/plan22-evaluation.md`, Limitations), is in six committed files,
    all pushed:
    - the plan 12 recording `p12x-sandbox-codex-turn-workspace-write.jsonl`,
      40 times on 23 lines (the "23 times" counts lines), in both copies,
      `testdata/probes` and `doc/research/probes`;
    - `p12x-sandbox2` (16 times) and `p12x-sandbox3` (35 times), in
      `doc/research/probes`;
    - one temporary path each in `doc/plan16-evaluation.md` (line 123) and
      `doc/plan19-evaluation.md` (line 98).

    None of the 127 files waiting to be committed holds it.

    The owner's decision, the same day: "Leave it: the repository is
    private." The six files stay as they are; GitHub answers an anonymous
    request for the repository with "not found". This settles only these
    files: new recordings still have the user name replaced
    (`doc/research/probes/README.md`).
- 2026-10-21: corrections C1 to C12, the owner's choice (C1's wording (a),
  no C13): built and verified, 17 of 18 criteria PASS
  (`doc/plan22-evaluation.md`, The corrections C1 to C12). No commit.
  - The design and criteria 5.1 to 5.18 above, written before the code;
    four additions came from the review of the first build, before their
    code.
  - Built by an implementer agent (Sonnet, max) in two rounds, from the
    investigation's verified copies for C2 to C5. Each round was reviewed
    here against a copy of the tree taken before the corrections.
  - The tests: a `corrections_test.go` in `internal/state`,
    `internal/render` and `internal/app`. A mutation run here caught 11 of
    11 fixes undone. One golden frame changed, `stream-p2-06.txt` line 23
    (C1).
  - C12 and the documents: codex-cli.md, claude-code.md, ui-design.md,
    events-and-state.md, reliability.md; criteria 1.6 and 2.11 corrected
    with a note.
  - Checks: build, vet (also `timing` and `live`) and gofmt clean; `-race`
    0 races; the timing step 7 of 7; with the proxy variables unset, the
    suite 14 of 14 packages, twice.
  - 5.14 FAIL, not caused by the corrections: amendment 19's
    TestClaudeEnvironmentIgnoresCodexProxy
    (`internal/app/codexproxy_test.go:413`) fails in the owner's environment
    since the Windows user variables point at Px. It pins `HTTPS_PROXY` only
    and reads the ambient `HTTP_PROXY`, which equals its Codex proxy. It
    fails the same on the tree before the corrections.
  - Then fixed at the owner's request: the test pins `HTTP_PROXY` too and
    checks it is kept. It passes with the owner's environment as it is, and
    fails on a planted leak of the Codex proxy. In the owner's environment,
    go test ./... gives 14 of 14 packages, twice, 0 FAIL. The fix is in
    amendment 19's own file, so it goes with amendment 19's commit.
  - The demonstrations (5.17): 13 runs, 39 screenshots, 354 burst frames,
    14 GIFs, in a temporary folder:
    `%TEMP%\claude\C--javadev-projets-agent-command-center\dcc3e64d-0d35-4470-ab9a-d258f8675bae\scratchpad\term\out-p22c`.
    Beside them are the harness copy (`p22cdemo.ps1`, `p22call.ps1`) and
    the two scratch mocks. The earlier demonstrations are unchanged.
  - Waiting for the owner: the review of the demonstrations, then the
    commits (plan 22 with these corrections, and amendment 19 apart).
- 2026-10-21: the two live checks, P22o and P22s, proposed and approved
  ("run P22o (Opus 5.5 instead of Haiku for this one process) and P22s
  (Codex with summary auto then detailed) exactly as proposed in plan.md"),
  then run at 18:51 and 18:52.
  - The drivers are `doc/research/probes/drivers/p22-claude.mjs` and
    `p22-codex.mjs`.
  - The recordings were sanitized, the organisation's candidates taken from
    the Windows domain and Px's upstream and never typed: 0 user name, 0 host
    name, 0 candidate left.
  - Results under Proposed, not run, above. Opus thinks as Haiku does, and
    the dashboard draws it right. Asking Codex for summaries makes its
    reasoning observable as it starts, not its text.
  - The research documents and the probes README are updated. No dashboard
    code changed.
  - Waiting for the owner: the two proposals made from the results (Codex's
    `summary`, Claude Code's three unrecognised events).
- 2026-10-21: amendment 20, the owner's request on those two proposals
  (doc/brief.md): built and verified, 13 of 13 criteria PASS
  (`doc/plan22-evaluation.md`, Amendment 20). No commit.
  - The design and criteria 20.1 to 20.13 above, written before the code.
    One change came at the review of the first build, before its code: a
    failing hook is a notice, not an error.
  - Built by an implementer agent in two rounds, each reviewed here against
    a copy of the tree taken before the amendment. Its requests went out as
    Sonnet 5.5 at effort max, the session's: no effort was passed, and the
    agent's frontmatter says high (Claude Desktop's CLI 2.1.304).
  - The tests: eight new files, and nine existing ones updated to the new
    behaviour, none loosened. The new files are `summary_test.go` in the Codex adapter, the
    mock files, the state, render and the application, and `hook_test.go`
    in Claude Code's adapter, the state and the application.
  - A mutation run here caught 10 of 10 changes undone.
  - Checks: build, vet (also `timing` and `live`) and gofmt clean; go test
    ./... 14 of 14 packages, twice in the owner's environment; `-race` 0
    races; the timing step 7 of 7. The 152 golden frames are byte for byte
    as before; go.mod and go.sum are unchanged.
  - The timing step's first two runs gave p95s about twice the
    corrections'. Timed in turn with the copy taken before the amendment,
    the two trees gave the same values: the machine at that time, not the
    change (`doc/plan22-evaluation.md`, Amendment 20).
  - Windows Terminal (20.12): P22s from a scratch mock, maximised then
    120x40. The card turns from "Model working" to "Thinking" within one
    frame of each reasoning item's start, and the details read word for
    word.
  - At 120x40, the second phase's details, opened at zone B's bottom, show
    their last two lines, the summary among them, only after Shift+PgDn.
    Zone B holds its view while an entry is selected, as before.
  - The summary drawn comes from P22s's `detailed` turn. Its `auto` turn,
    the value the dashboard now sends, gave none.
  - Proposed, not run, with the owner's approval only: the live run of the
    dashboard with Codex (criteria, above).
  - Waiting for the owner: the review of the demonstrations, then the
    commits (plan 22 with the corrections, amendment 19 apart, and
    amendment 20).
- 2026-10-21: the live check L20, approved the same day ("run the proposed
  live dashboard check with Codex, one turn on a temporary project, as
  plan.md's amendment 20 criteria describe, then record the findings"), run
  at 21:38: 6 of 8 criteria PASS, 2 FAIL (`doc/plan22-evaluation.md`, The
  live check L20). No commit.
  - The criteria L20.1 to L20.8 above were written before the run.
  - The card, read 227 times against the wrapper's times, behaved as
    designed. It read "Model working" from its entrance's count row, 142 ms
    into the step, up to the reasoning item's start 1,701 ms in. It read
    "Thinking" from 24 to 44 ms after that start, and on every read until
    the item completed 2,211 ms later.
  - Under `auto`, Codex sent a summary this time, "**Calculating 12!
    divisors**". The details show it, and the item's start and end to the
    wrapper's millisecond.
  - L20.4 FAIL, the criterion's: zone D's two-row form, maximised here and
    at 120x30, draws no unrecognised-events counter. The recording replayed
    through the adapter gives 0.
  - L20.5 FAIL, the criterion's: the first 3 reads, in the card's first
    111 ms, came before the entrance draws its count row at 140 ms.
  - Configuration check PASS, nothing left running; the temporary project
    deleted.
  - Then, at the owner's request ("Yes: correct L20.4 and L20.5 with a
    note, as the evaluation proposes … No code change, no new run."), both
    were corrected above, each with a note. Judged again by script on the
    run's records, both PASS, so all 8 of L20's criteria hold as corrected.
    The block as reported stays FAIL for both (`doc/plan22-evaluation.md`,
    The live check L20).
  - Waiting for the owner: the review of the demonstrations and of L20's
    screenshots, then the commits (plan 22 with the corrections, amendment
    19 apart, and amendment 20 with L20).

## Closed plans

One entry each: what the plan did, its approval, its evaluation file and
its commits. Their criteria and progress logs: `git show 8888f59:plan.md`
(plans 1 to 13), `git show 84d2ad8:plan.md` (plan 14), `git show
b047434:plan.md` (plan 15), `git show becfb34:plan.md` (plan 16), `git show
f00c2d8:plan.md` (plan 17), `git show 8b3b16b:plan.md` (plan 18), `git show
d84e5d4:plan.md` (plan 20), `git show f939d96:plan.md` (plan 21).

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
