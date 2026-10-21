# Plan 22 investigation (2026-10-21)

The owner asked for a read-only technical investigation of the live
thinking indicator, for Claude Code and Codex, before plan 22 is
finalised. It covers:

1. how Claude Code's live estimate is computed and how accurate it is;
2. how the Codex adapter processes `thread/tokenUsage/updated`;
3. defects;
4. regressions;
5. verification.

Investigation first, corrections second. Nothing in the application was
changed: every check ran in scratch copies, and no CLI was started.

**Method.**
- Read: the recordings (`doc/research/probes`, 18 Claude Code and
  23 Codex recordings), the documentation, and Codex's source at tag
  `rust-v0.163.0`, studied only.
- Two adversarial reviews: one of the adapters and the reducer, one of
  the rendering and the application. Each was re-checked independently,
  the first by an agent, the second here.
- A regression analysis against the baseline, commit f939d96 (plan 21).
- The test suites, the timing step and the race detector, run here.
- The corporate proxy cut the agents about every 15 minutes
  (`ERR_PROXY_TUNNEL`). Each one resumed from its own notes, and no result
  was lost.

## At a glance

- **Claude Code's "≈ 1,100" is the CLI's own running estimate.**
  - What it is: `system/thinking_tokens`, the sum of increments the API
    attaches to each thinking delta: steps of 50 tokens, about once a
    second.
  - The replay drew exactly the recorded value at that moment.
  - That phase's exact count, 2,852, came from the API's final
    `message_delta`, 0.4 s after the phase ended. Its last estimate, 2,700,
    was 5.3% under.
  - Over 27 recorded phases, the last estimate is 14% under to 281% over
    the exact count: short phases run over, long ones a little under.
  - The dashboard always labels it as an estimate, except an interrupted
    message's 0 (D1).
- **Codex: the adapter and the reducer are correct on every recorded
  update.**
  - 76 updates, 0 mismatches.
  - Each count goes to the right phase. The phases' counts add up to
    Codex's own total in 14 of 14 recordings.
  - Reasoning usage only arrives after a model step has finished. Codex's
    source and 75 of 75 recorded updates agree.
  - That is a provider limitation: the indeterminate bar stays.
- **Defects.**
  - 13 worth correcting, 1 optional, 4 not worth it.
  - One is medium (D2): after a crash and Ctrl+R, a resumed Codex
    thread's replayed usage lands on the crashed step. The rest are low.
  - The most visible is D3: the activity line flickers at the end of
    every Codex turn.
- **Regressions: none.** Checked by:
  - every baseline test;
  - the golden frames, zone by zone;
  - every line of the 23 recordings, drawn by both trees at three sizes;
  - the timing step;
  - the race detector.
- **Corrections C1 to C13** wait for the owner's approval. C1's wording
  is the owner's choice.

## 1. Confirmed findings

### 1.1 Claude Code's live estimate

**Where it comes from.** Confirmed on 33 thinking blocks in
14 recordings, Claude Code 2.1.300 to 2.1.303, all Haiku 5.5.

1. Claude Code asks the API for `display: "updates"`: each `stream_event`
   line carries `"thinking_display":"updates"`. Reasoning text is omitted,
   so a thinking block streams empty `thinking_delta` events.
   - Each delta carries an `estimated_tokens` increment, always a multiple
     of 50.
   - The first is 50, arriving 146 to 392 ms (median 206 ms) after the
     block starts.
   - The next are 100 to 350 tokens (median 250), 1.0 to 1.9 s apart
     (median 1.25 s).
   - This field is in no API documentation. The streaming and thinking
     pages describe an omitted block as one empty `thinking_delta`, then
     its `signature_delta`
     (https://platform.claude.com/docs/en/build-with-claude/streaming,
     read 2026-10-21).
2. Claude Code adds the increments up. 0 to 4 ms before it forwards each
   delta, it emits `system/thinking_tokens`:
   - `estimated_tokens` is the running sum;
   - `estimated_tokens_delta` is the increment.

   All 69 such lines follow this pattern. The value always equals the sum
   of the deltas and always grows. This message is documented as
   SDKThinkingTokensMessage: "a running estimate of the thinking tokens
   generated so far in the current block", for "progress display", Claude
   Code 2.1.153 or later
   (https://code.claude.com/docs/en/agent-sdk/typescript, read 2026-10-21).
3. At the block's end, a closing `thinking_delta` carries
   `estimated_tokens: null`.
   - In 21 of the 32 blocks with a closing delta, Claude Code then emits
     one more `system/thinking_tokens`, 0 to 2 ms before the
     `signature_delta`.
   - Its increment is uneven: 20 to 328 tokens, never a multiple of 50.
   - Its value F follows the block's exact count E closely:
     F = 153 + 0.78 × E (R² 0.993, standard error 17 tokens).
   - Inferred, not confirmed: it is computed from something proportional
     to the whole block and known at its end. The likely basis is the
     signature, which carries the encrypted full thinking (API
     documentation). The recordings scrub the signatures, so this cannot
     be checked.
   - The recordings also cannot show when this last value is sent. It is
     missing from:
     - 6 blocks of 719 tokens or more;
     - 4 blocks under 50 tokens, which have no estimate at all;
     - the interrupted one.
   - A live probe keeping the signatures' lengths would settle both
     questions. The dashboard does not need the answer.
4. **The exact count is the provider's.** It is the API's final
   `message_delta` `usage.output_tokens_details.thinking_tokens`.
   - The documentation says "this breakdown appears only on the final
     `message_delta` event"
     (https://platform.claude.com/docs/en/build-with-claude/extended-thinking).
   - The API computes it by re-tokenizing the raw reasoning, delimiters
     included (Agent SDK reference, `Usage`).
   - It arrives 0 to 2,378 ms (median 636 ms) after the block's stop.

**How accurate it is.** On the 27 completed blocks that had an estimate,
the last estimate shown during a phase, against that phase's exact count:

| Blocks | Error |
| - | - |
| All (27) | −168 to +155 tokens; −13.8% to +280.8%; median +26.3%; mean absolute error 96 tokens |
| With a last value (21) | over in 19, by up to 155 tokens |
| Without one (6, all ≥ 719 tokens) | under by 34 to 168 tokens (3.8% to 13.8%): the last step trails the block's end |
| Under 100 tokens (3) | +137% to +281% |
| 100 to 300 tokens (10) | +33% to +121% |
| 300 to 1,000 tokens (11) | −9.6% to +26.3% (mean absolute error 11.9%) |
| 1,000 tokens and more (3) | −5.3% to −13.8% |
| By CLI version | no effect beyond block size (2.1.301: 15 blocks, median +18.5%; 2.1.303: 8 blocks, median +43.9%) |

- The values during a phase have no exact counterpart to compare with.
- They follow 200 tokens a second of elapsed time (R² 0.994, error
  48 tokens). The exact counts follow 206 tokens a second of thinking
  time (R² 0.995).
- So the live value tracks generation in 50-token steps refreshed about
  once a second. At most it is 1.9 s behind. It runs high only at the end
  of short blocks.

**The "≈ 1,100" in the replay is exact.**
- It comes from P12c-block's second phase.
- Replayed with its recorded times, the dashboard draws "≈ 1,100" from
  13,182 to 14,813 ms, the times of the recorded line and of the next one.
- The phase ended at 21,351 ms on a last estimate of 2,700. Its exact
  count, 2,852, came 382 ms later: the estimate was 5.3% under.

**It is never shown as the provider's count.** Checked by replaying 10
recordings at 4 sizes, a frame every 25 ms and around every event:
33,944 frames.
- Every live number reads "≈ N thinking tokens · live estimate" ("≈ N
  tokens · estimate" in the 2-row card). It equals the recording's value
  at that instant.
- From the phase's end until its exact count arrives, the summary reads
  "· ≈ N thinking tokens (estimate)".
- The exact count, with no "≈", appears only after its `message_delta`,
  and equals it.
- The details give both, each with its source: "N thinking tokens ·
  exact, for this phase · message_delta …" and "live estimate N at its
  last update · system/thinking_tokens, estimated by the CLI".
- Zone D's details list it as `thinking.estimate`.
- The result's turn total and `modelUsage.thinkingTokens` never stand
  for a phase's count.
- The one exception is an interrupted message's 0 (D1).

### 1.2 Codex's `thread/tokenUsage/updated`

**When Codex reports usage.** Confirmed in Codex's source at tag
`rust-v0.163.0` (commit 83086188ae06; Apache-2.0, studied only).

- **The notification.** The app-server sends `thread/tokenUsage/updated`
  from the core's `TokenCount` event: app-server/src/bespoke_event_handling.rs,
  built at line 1595, sent at 1601.
- **When the core sends it.** The core records a response's usage when
  the response completes (core/src/session/turn.rs, the
  `ResponseEvent::Completed` arm).
  - It adds the usage to a running `total` and sets `last` to it
    (protocol.rs, `append_last_usage`).
  - It sends the event only after the response's stream and its tool
    calls are done, never while a response streams.
- **The client side.** Codex's Responses client reads usage only from
  `response.completed` or `response.incomplete`
  (codex-api/src/sse/responses.rs).
- **The documentation.** OpenAI's documentation names the reasoning-token
  field but not when usage arrives (developers.openai.com, guides
  "reasoning" and "streaming-responses", read 2026-10-21).
- **Reasoning items.** A hidden reasoning item reaches the client only
  once it has finished. Its `item/started` and `item/completed` come back
  to back (core/src/stream_events_utils.rs, `handle_output_item_done`),
  hence the 0 to 10 ms recorded between them.
- **Usage is replayed.** Codex sends the thread's last usage again, once
  and outside any turn, on:
  - a resume from stored history: a new app-server process, which is the
    dashboard's Ctrl+R after a crash;
  - a resume of a running thread;
  - a fork.

  The code is request_processors/token_usage_replay.rs, called from
  thread_processor.rs:4433, :4731 and :5699. It is tested in
  app-server/tests/suite/v2/thread_fork.rs. 0.145.0's README documents
  it (line 321); 0.163.0's says nothing.
- **Updates with an unchanged total** also exist: rate limits, a
  recompute, a full context window. The adapter's repeat check covers
  them.

**The adapter is correct on every update.** All 76 updates of the
14 app-server recordings were checked against the raw lines by an
independent parser: 57 on the main thread and 19 on child threads, 60
from 0.163.0 and 16 from 0.145.0. There were 0 mismatches.
- Each update gives one telemetry event. Its agent is empty for the
  main thread, or the child's thread id.
- `tokens.total` and `tokens.last` equal the line's totals. `context.used`
  and `context.window` are present only when the line has the window.
- `reasoning.last` is present exactly when `last.reasoningOutputTokens`
  is a number and the thread's total changed, and it equals that field.
- No step marker comes from a child's update or from a repeat. The one
  repeat is P6b's line 87, after an interrupt.
- On every update after a thread's first, `last` equals the total minus
  the thread's previous total, for all six fields.
- A thread's first update has `last` equal to its total, except a forked
  child's: P7 line 95, total 108 reasoning tokens against last 25, its
  inherited history.
- No update lacks `reasoningOutputTokens`.

**What a count covers.**
- 47 model steps had exactly one reasoning item and a count above 0.
- 28 had no reasoning item and a count of 0.
- None had a count without a reasoning item, and none had two items.

So a step's `last.reasoningOutputTokens` is its reasoning item's own
count, and the dashboard can show it as that phase's.

**The reducer gives each count to the right phase, once.** All 14
recordings were replayed through the adapter and the reducer, against
counts read from the raw lines. There were 0 problems.
- Every main-thread update whose step had a reasoning item counts it on
  exactly one phase, that item's.
- The 28 steps without a reasoning item count nothing (their count is 0
  anyway).
- Children's updates and the repeat never reach a phase. No phase is
  counted twice.
- The phases' counts add up to the main thread's final
  `total.reasoningOutputTokens` in all 14 recordings: for example P7,
  98 = 83 + 8 + 7; P12x-switch, 242; P12x-sandbox, 382.
- P6b's interrupted step has no count, and its tokens are missing from
  Codex's total as well (16 = 7 + 9): its response never completed.
- Step markers against an independent oracle built from the lines: the
  same in 12 of 14 recordings. In P9a and P6b, three markers come 12 to
  85 ms early (D14).

**Mis-attribution cases**, built from synthetic lines and run through the
real adapter and reducer.
- **Handled right:**
  - an interrupted turn, then a repeat, then a new turn;
  - an interrupted step's usage arriving after its turn ended;
  - a child's usage arriving between the orchestrator's reasoning item
    and its update;
  - a forked child whose first total holds history;
  - an update before any turn.
- **Read "over 2 phases", which is honest:**
  - two reasoning items in one step;
  - a step abandoned without usage.
- **Not reachable:**
  - Usage arriving after the next step's reasoning item would lose a
    count. But Codex 0.163.0 sends usage before its next request, and
    0 of 75 recorded updates arrive late.
  - A missing `reasoningOutputTokens` gives no metric, never a 0. The
    schema requires the field.
- **Two source paths that break the rule that `last` is the step's
  usage (D18).** No recording takes either: in all 60 0.163.0 updates,
  `last` equals the difference of the totals.
  - An incomplete response's spending is added to `total` only, so the
    next `last` is stale and would count again.
  - A full context window resets both totals with the reasoning count
    zeroed.
- **Out of plan 22's scope,** and there before it: a forked child's
  first total includes its parent's history (P7 line 95, 0.145.0 only),
  so zone D's derived `tokens.session` counts that history twice. No
  0.163.0 child inherits history.

**Can reasoning usage update during execution?** No. Codex reports it
only after a model step ends. This is a provider limitation, not a defect.
- Not one of the 75 updates arrives while a reasoning item is open, or
  before its own step's reasoning item.
- Each comes 165 ms to 9.2 s (median 1.5 s) after that item completed,
  and 4 to 126 ms after the step's last tool completed (0.163.0).
- The source shows why: usage exists only once a response has completed.

The card's indeterminate bar and "reasoning tokens reported when the
step ends" are the truthful choice, and they stay, as the owner asked.

## 2. Defects and their root causes

Each defect was reproduced by a throwaway test and re-checked
independently. "Recorded" means a real recording, or documented
behaviour, triggers it.

### Worth correcting

**D1. An interrupted Claude Code message's 0 is drawn as its exact
count.**
- Severity: low, but it breaks the hard rule "missing telemetry is shown
  as unavailable, never as zero". Recorded: P303a and P2.
- What the owner sees: "▸ ✗ Thinking interrupted after 0.2s · 0 thinking
  tokens". The details say "0 thinking tokens · exact, for this phase ·
  message_delta …", next to "live estimate 50".
- Cause: on an interrupt, Claude Code closes the stream itself (P303a,
  lines 75 to 78). Its `message_delta` has `stop_reason` null and the
  message_start's usage (output_tokens 7), with a default
  thinking_tokens of 0. The result's usage is zero.
- `messageCount` (internal/state/reduce.go) records that 0 as the phase's
  count.

**D2. After a crash and Ctrl+R, a resumed Codex thread's replayed usage
lands on the crashed step.**
- Severity: medium, a wrong count with no sign of it. Recorded: no
  recording, but documented behaviour and Codex 0.163.0's code trigger it
  (section 1.2).
- The new process's mapper has no previous total for the thread, so the
  replay is not taken as a repeat.
- `reasoningTokens` then puts the previous step's count on the crashed
  step's uncounted phase.
- Example, P12x-sandbox cut at line 61 with line 57 replayed: "Thought for
  up to 4.1s · 10 reasoning tokens", where that step's own count is 16.
  P9n gives "0 reasoning tokens" for 25.

**D3. The activity line flickers at the end of nearly every turn.**
- Severity: low to medium; visible at every Codex turn's end. Recorded.
- Between the final answer's end and the turn's end, the turn still runs
  with no card and nothing streaming. So `activityOn`
  (internal/render/thinking.go:867) adds a blank row and "◡ Connecting
  dots… 00:01" under the finished answer, gone at the turn's end.
- In follow mode, zone B scrolls up two rows and back.
- Codex's last message completes 27 to 145 ms (median 62 ms) before
  `turn/completed` in all 17 recorded turns: one to three frames. Claude
  Code's gap is 0 to 27 ms.
- Shorter flickers come:
  - before a new Codex step card (33 to 39 ms);
  - between two step cards (53 ms);
  - after an answered approval (104 ms);
  - before an approval box (47 ms).

**D4. A thinking phase opened while the turn is Idle never ends.**
- Severity: low; no known trigger and no wrong number. Not recorded: no
  main-thread block or reasoning item arrives outside a turn. The three
  in P13c are artefacts of replaying its raw recording.
- `blockStarted` and `reasoningStarted` do not check the turn, and
  `endThoughts`, `closeBlocks` and `messageCount` stop at the last prompt.
  After the next prompt the phase is out of their reach.
- It then stays live for good, with a stuck card. The application asks
  for a frame every 50 ms for the rest of the session, against
  reliability.md's test 9.

**D5. "exact count not reported yet" stays after the turn has ended.**
- Severity: low, a details line. Recorded: P6b (1 of 66 rows) and after
  any crash.
- `thoughtDetails` (thinking.go:236) says "no exact count reported" only
  on an outcome or an inferred end. A Codex step that ended on its own in
  a turn interrupted later keeps "not reported yet" for good.
- This completes the review's finding 5, whose fix was incomplete.

**D6. A Codex step that follows a response with no call has no start.**
- Severity: low. Recorded: P9a, and P6b's second phase.
- The adapter assumes that a step which made no call ends the turn
  (`steps.usage()`, internal/provider/codex/mapper.go:754, design 4).
  That is wrong for a response with only reasoning: Codex asks the model
  again.
- After P9a's line 22, such a response, the next reasoning item (line 24)
  finds no step open.
- Its row reads "Thought · 118 reasoning tokens" with no duration, and
  its exit reads "✓ Thought --:--".

**D7. The 2-row card's exit row is cut at 80 columns.**
- Severity: low, it lasts 100 ms. Recorded: P303a at 80x24.
- `countRow` (thinking.go:437) chooses the long or short count against
  the spinner's head. The exit draws the summary's longer head, so it
  shows "✗ Thinking interrupted after 0.2s  ≈ 50 thinking tokens · live
  esti…00:00".
- It affects every interrupted or failed phase at a height that uses the
  2-row card, Codex's too.

**D8. A summary row at 80 columns cuts its origin.**
- Severity: low. Recorded.
- `toolLine` (internal/render/cards.go:285) keeps the name whole and cuts
  the right part, so a long summary ends in "02:00:09  obse…".
- In P303a at 80x24 this lasts for good. The estimate form is cut the same
  way for 0 to 2.4 s.

**D9. The exit's count row says "live estimate" or "token usage pending"
after the exact count came.**
- Severity: low, at most 100 ms. Recorded: 3 of 22 phases.
- `countText` (thinking.go:463) ignores `HasTokens`. When the
  `message_delta` comes within 100 ms of the block's stop (P303a 0 ms,
  P303e 11 ms, P2 24 ms), the exit reads "✓ Thought for 0.3s" over
  "Thinking · token usage pending", with 46 tokens already reported.

**D10. `reveal` keeps a row free for a "↓ N new" badge that is not
shown.**
- Severity: low, and rare.
- internal/app/inspect.go:461 tests `len(Conversation) > ConvSeen`. The
  badge counts `newShown` (internal/render/dashboard.go:462), which leaves
  dropped steps out.
- With zone B held and only a dropped step new, revealing an entry
  already in view scrolls one row (ConvTop 217 to 218).

**D11. One test assertion can never fail.**
- Severity: low, test only.
- internal/app/thinking_test.go:347 looks `ui.Live` up by `e.Seq`. But
  `ui.Live` has been keyed by `Thought.ID` since the review's fix 4 (in
  P12c-block, Seq 8, 73, 127 and 160 against IDs 1 to 4).
- Criterion 1.7 stays covered by the frame checks and by `Animating`
  being false.

**D12. A Codex reasoning item that never completes stays live beside
later steps until the turn ends.**
- Severity: low. Not recorded: all 48 recorded items complete 0 to 10 ms
  after they start.
- If it happened, its time would read wrong with no sign of it.

**D13. A Codex turn with no prompt entry shares its count with an earlier
turn's uncounted phase ("over 2 phases").**
- This is a turn the CLI starts itself.
- Severity: low. Not recorded.

### Optional

**D18. The adapter takes `last.reasoningOutputTokens` as the step's
count.**
- Codex 0.163.0's source has two paths where `last` is not the latest
  response's usage (section 1.2), so a count could repeat or read 0.
- No recording takes either path.

### Not worth correcting

**D14. The steps tracker's `complete` flag outlives a usage that started
no step** (mapper.go:758).
- It is the same wrong assumption as D6.
- In the recordings it only places three markers 12 to 85 ms early, in
  P9a and P6b, which no one can see.
- A phantom card needs a synthetic two-call step.
- Its fix is small, but it changes a test expectation and a mock file.

**D15. A reasoning item reported only by its `item/completed` is
dropped, with its count.**
- Not recorded.
- The proposed fix breaks two deliberate tests: without the turn id, it
  cannot tell such an item from a late completion of an earlier turn's.

**D16. `barScale` (thinking.go:636) never returns for an estimate above
9×10^18 tokens (int64 overflow).** Theoretical.

**D17. The 2-row card's count form can alternate with the expression**,
for a phase over an hour long with 12,345 tokens or more. Theoretical.

## 3. Regression-test results

None found. The baseline is commit f939d96 (plan 21), taken with
`git archive`.

**Tests**
- Baseline: 893 results in 14 packages, 892 pass and 1 skip
  (`TestDiskListLink`).
- Working tree: 1,062 results, 1,061 pass and the same skip; 14 packages
  ok.
- Every baseline test and subtest still exists and passes. 169 results
  are new.
- Run again here on the working tree:
  - `go build ./...` and `go vet ./...` exit 0;
  - `gofmt -l .` prints nothing;
  - `go test ./... -count=1` passes 14 of 14 packages;
  - no file changed.

**Baseline tests changed** (10 files):
- **Extended, intent kept:**
  - the Codex mock's event count, 177 to 182: five inferred steps;
  - TestEntryViewer: a thought is now a cursor stop;
  - the fixture, `msgid` and replay tests: their record counts gain
    P12c-block;
  - the mock files test: two more files and a stronger marker check;
  - the trace test, for thinking entries;
  - the timing command's comment.
- **Inverted by design:** TestConversationLook and
  TestAnswerStreamsAcrossBlockLines.
  - The orchestrator's "∴ thinking" mark gave way to the thinking entry
    (plan 22, item 9).
  - A subagent keeps its mark, which gained a test.

**Golden frames** (32 changed, 32 new). Zones A, C and D are unchanged
in all 32 changed frames.
- 26 change only in zone B: thought rows, the "… is processing your
  request…" placeholder, and zone B behind a window shifted a row.
- 2 details windows lost the † after `thinking.estimate`, which is now
  documented.
- 4 Explorer and Quick open windows list the two new mock files.

**Every recording, line by line.** After each line of the 23 recordings,
both trees drew the screen at three sizes: 2,953 snapshots, 53,154 zone
hashes.
- Zones A, C and D, the header and the footer are identical everywhere.
  That covers the workflow tree, the subagents, the quota and context
  bars, and the telemetry.
- With the thinking entries removed and the turn idle, zone B is
  identical in 2,841 snapshots. The other 112 show the baseline's
  "∴ thinking" rows, and speaker heads that only those rows made.
- A running tool's mark (◉ idle, ⠋ running) behaves as in the baseline.

**Conversations with no thinking.** Both trees drew 13 synthetic scenes
at 80x24, 120x40 and 200x60:
- prompts and answers;
- tool calls running, done and failed;
- an approval and a subagent;
- folded and unfolded runs;
- the entry cursor;
- a held view with its badge;
- failed and interrupted turns.

The idle scenes are identical byte for byte, colours included. The
running ones differ only in zone B, by design: the placeholder and the
activity line.

**The mock provider.** Its change applies at speed 0 only, which only
the tests use: every event is queued before `Start` returns. The
application replays at speed 1 (cmd/agent-command-center/main.go:42),
so nothing changes for it.

**The owner's areas.**
- **Streaming:** TestStreamingP2 passes; its frames change only by the
  placeholder and a thought row.
- **Animations:** the spinner's and zone A's tests are unchanged and
  pass.
- **Progress bars:** zone D is identical everywhere.
- **Tool execution and subagent monitoring:** their rows and zone C are
  unchanged.
- **Approvals:** their tests pass, and their frames gain only thought
  rows.
- **Scrolling:** the pane tests pass, but see D3 and D10.
- **Existing conversation rendering:** see the two checks above.
- **Timing:** the timing step passes 7 of 7.
  - TestThinkingFrame: 95th percentile 3.53 ms, on 5,522 entries with
    721 phases.
  - Three alternating runs of the two slowest tests, against the
    baseline (95th percentile):

    | Measure | Baseline | Working tree |
    | - | - | - |
    | TestLongSession, last tenth of frames | 2.7 to 3.0 ms | 3.0 to 3.5 ms |
    | TestLongSession, end of session | 3.5 to 3.6 ms | 3.5 to 3.7 ms |
    | TestStreamToFrame | 3.0 to 4.0 ms | 3.0 to 4.0 ms |

  - The other four tests differ by 0.2 ms at most.
  - The target is 16 ms, and the clock rounds to about 1 ms.
  - A first run's 4.7 ms for the long session's first tenth did not
    repeat: 1.3 to 2.3 ms afterwards.
- **Race detector:** `go test -race ./... -count=1`, with the portable
  compiler doc/evaluation.md names, passes 14 of 14 packages with
  0 data races. No file changed.

## 4. Genuine provider limitations

What each CLI sends, not defects. The dashboard shows each as such.

**Claude Code**
- The only live count is an estimate. The API attaches it to its thinking
  deltas (undocumented) and the CLI sums it: 50-token steps, about once a
  second, up to 1.9 s behind.
- The stream does not show how its last value is computed.
- The exact count comes per message, with its `message_delta`, 0 to 2.4 s
  after the phase.
- No thinking text: `display: "updates"` leaves reasoning blocks empty.
- An interrupted message has no count: the CLI sends 0 (D1).
- `system/thinking_tokens` has no agent field. Subagents send no stream
  events, so this never made a value ambiguous.
- No thinking budget is reported, so mode A of the bar is never shown.

**Codex**
- No live reasoning count: usage comes only after a model step completed
  and its tools finished.
- A hidden reasoning item reaches the client only once it has finished.
  So a phase's start is its step's, inferred, and its time is an upper
  bound ("up to").
- One count per model response. Every recorded step had one reasoning
  item at most.
- The last usage is replayed on a resume or a fork.

## 5. Recommended corrections

None is applied: each waits for the owner's approval.
- Every correction keeps Codex's indeterminate bar, fabricates no count
  and adds no dependency.
- Unless noted, each keeps the approved look and animations.
- "Verified in a copy" means the change was applied to a scratch copy
  and `go test ./...` passed there.

| # | For | The correction | Where | Checked |
| - | - | - | - | - |
| C1 | D1 | An interrupted message reports no exact count; the owner chooses the wording (below) | reduce.go `messageCount`; thinking.go details | (a) and (b) change criterion 2.11's interrupted example; (a) also TestThinkingSummary and golden stream-p2-06.txt line 23; (c) keeps every test |
| C2 | D2, D5, D13 | A `Thought.Closed` flag: a phase whose turn or session ended with no count takes none later, and its details say "no exact count reported" | state.go; reduce.go `endThoughts`, `reasoningTokens`; thinking.go | verified in a copy, every test passing |
| C3 | D4 | Sending a prompt ends any phase still live, at the last event's time | actions.go, one line | verified in a copy |
| C4 | D12 | A Codex step's start ends a reasoning item still live, its end inferred | reduce.go `stepStarted` | verified in a copy |
| C5 | D6 | A reasoning item that finds no step open starts at its turn's last usage update, so its time is a bound like every other Codex phase's | reduce.go | in a copy; one expectation changes (TestThinkingCodexP6b's second entry gains a start) |
| C6 | D3 | The activity line appears once its condition has held 200 ms, longer than every recorded gap | app.go, look.go | to build |
| C7 | D7 | The 2-row card's exit chooses the count's form against the head it draws | thinking.go `countRow`, one line | to build |
| C8 | D9 | In the exit, Claude Code's count row shows the exact count once it came, else the estimate without "live"; Codex's text is unchanged | thinking.go `countText` | to build |
| C9 | D8 | A thought's summary row drops its origin whole when the right part does not fit, never "obse…"; tool rows unchanged | thinking.go `summaryRow` | to build |
| C10 | D10 | `reveal` counts new entries as the badge does | inspect.go; dashboard.go exports the count | to build |
| C11 | D11 | The assertion looks phases up by `Thought.ID` | thinking_test.go:347 | to build |
| C12 | | Documentation corrections (below) | doc/research, plan.md | |
| C13 | D18 | Optional: when the difference of a thread's reasoning totals disagrees with `last`, use the difference; no count when it is negative | codex/mapper.go `tokenUsage` | to build; no recorded update changes |

**C1's three wordings:**
- (a) Recommended: the row reads "✗ Thinking interrupted after 0.2s".
  The details say "no exact count: the CLI reports 0 for an interrupted
  message" and keep "live estimate 50".
- (b) The row shows the estimate: "· ≈ 50 thinking tokens (estimate)".
- (c) The row keeps "0 thinking tokens"; only the details relabel it as
  reported for an interrupted message, not exact.

**C2's alternative for D2 alone** (tested in a copy): one line, with the
reducer ignoring reasoning counts while the turn is idle.

**C6's mechanism and checks.** The application notes when the line's
condition began, and the 100 ms spinner tick draws the line once due.
TestThinkingActivity and the activity frames need checking. The line
comes 200 ms after a prompt rather than at once.

**C12's documentation corrections:**
- codex-cli.md, section 10 item 4, states the resume's usage replay as if
  observed.
  - Plan 1's verification had already marked that claim "not probed"
    elsewhere (doc/research/plan1-evaluation.md, line 14).
  - It becomes: documented for 0.145.0 and present in 0.163.0's source,
    fork included; never recorded.
- The gap from a step's last tool to its usage becomes 4 to 126 ms, not
  4 to 87 (codex-cli.md section 12, plan.md).
- claude-code.md, section 12, gains:
  - the relay of the API's per-delta estimate;
  - the last value's fit;
  - an interrupted message's usage being its message_start's.

**Not recommended:**
- D14: its fix changes an expectation and a mock file for markers no one
  can see.
- D15: it cannot be fixed without breaking deliberate tests.
- D16 and D17: theoretical.

## 6. Risks to existing functionality

- **C1** changes an approved criterion's text (2.11), one golden frame
  and two tests. Nothing else reads the interrupted count.
- **C2** only withholds a count that arrives after its turn ended.
  - No recording has one: 0 of 56 Codex reasoning counts come after their
    turn, and every Claude Code `message_delta` precedes its result.
  - A CLI that sent one later would show "no exact count reported":
    visible and honest.
- **C3 and C4** only reach phases left live by paths no recording takes.
  Every recorded replay ends with no phase live.
- **C5** turns one Codex phase's unknown start into a bound, as every
  other Codex phase is timed.
- **C6** is the largest.
  - It adds state to the application, and the activity line comes 200 ms
    later at a turn's start.
  - It must keep the 100 ms spinner and the "an idle dashboard draws
    nothing" test.
  - Its golden frames and TestThinkingActivity may need new moments.
- **C7 to C9** change text at 80 columns only. The 80x24 golden frames
  with an interrupted phase, an exit or a long summary may change. Tool
  rows are untouched.
- **C10** changes scrolling only in the rare case it fixes.
- **C11** changes a test only.
- **C13** changes nothing on recorded data. It reaches only the two source
  paths of D18.
- **Every correction** goes through the full checks: build, vet, gofmt,
  the tests twice, `-race`, the timing step, and every golden frame's diff
  read. A correction that changes a frame is shown to the owner before
  any commit.

## Not verified

- Opus 5.5 and Sonnet: every Claude Code recording runs Haiku 5.5.
  Their estimate's cadence and accuracy may differ.
- How Claude Code computes the estimate's last value: the recordings
  scrub the signatures.
- Codex 0.163.0's usage replay on a real resume: it is in the source, but
  no recording has it.
- Whether asking Codex for reasoning summaries would show reasoning as it
  happens: the server's behaviour.
- How visible D3's flicker is in a real terminal: inferred from the
  recorded gaps and the frame schedule.
- C6 to C11 and C13: described, not built.

## Evidence

Kept outside the repository, in this session's scratch folder
`p22inv\`:
- `claude-estimate\`: the block scripts and the accuracy tables.
- `codex-usage\`: notes.md, results 1 to 15, and the replay tests.
- `review-state\`, `verify-state\`, `review-render\`: the reviews, the
  verdicts, and each fix applied to its own copy.
- `regression\`: both trees' test runs, the zone hashes and the synthetic
  scenes.
- `me\`: the zone comparisons, the timing runs, the race run, and a fresh
  copy for the reviewers' reproductions.
