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
d84e5d4:plan.md` for plan 20), and its results in its evaluation file.

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

## Plan 21 - What the review of d84e5d4 found

**Done on 2026-10-20 and committed at the owner's request** ("Apply the
speed fix, then commit"). Approved with the recommended choices ("I
approve plan 21 with the recommended choices"): 2 (a), 4 (a), 5.1 build,
5.2 to 5.5 correct, 5.6a build, 5.6b to 5.6d correct, 5.7 build, 5.8
correct. Its evaluation, [`doc/plan21-evaluation.md`](doc/plan21-evaluation.md):
38 PASS and 2 FAIL, the long session's frame time, a cost plan 20's
RESULT rule added; then the proposed fix, approved by the owner: 6 PASS,
the timing step 6 of 6.

The source: a read-only review of commit d84e5d4 on 2026-10-20, plan 20
closed and approved, no plan active. At d84e5d4, `go build ./...` and `go
vet ./...` exit 0, `gofmt -l .` prints nothing and `go test ./... -count=1`
exits 0 (14 packages ok); Claude Code 2.1.303 and Codex 0.163.0 are the last
probed versions. Each finding below was checked in the code at its file and
line; items 1 and 4 were reproduced with a throwaway test in a scratch copy
outside the repository. No live run is needed: deterministic tests and a
mock run in Windows Terminal settle every criterion, so the plan uses no
quota.

### The owner's decisions

Each choice is the owner's; the recommended one is in the middle column.

| Item | Recommended | The other choice |
| - | - | - |
| 2 Two timed tests | (a) their time judged in the timing step only | (b) evaluation.md corrected: judged in every run |
| 4 Zone C at 160x24 to 160x26 | (a) the floor raised: C always shows one agent | (b) a minimum of 27 rows documented |
| 5.1 Configuration file | build the send and newline keys only, in JSON | correct the documents |
| 5.2 Sign-in kind | correct the documents | build |
| 5.3 Capabilities flags | correct the documents | build |
| 5.4 Frame rate | correct the documents (60 fps) | build (30 fps) |
| 5.5 Separation test | correct the document | none |
| 5.6a "↻ n" retries in zone C | build | correct the document |
| 5.6b "ai" origin label | correct the document | build |
| 5.6c B's dashboard notices | correct the document | build |
| 5.6d Markdown cache | correct the document | build |
| 5.7 MOCK badge at full screen | build | correct the documents |
| 5.8 ASM-23's path | correct the document | none |

### Item 1 - Zone D's session time, wrong after 2,000 events

`sessionTime` (`internal/render/telemetry.go:402-407`) counts from
`s.Log[0].At`, but the state keeps only the last `MaxLog` = 2,000 events
(`internal/state/state.go:324`, `reduce.go:873`), and each streamed text
delta is one event. Past 2,000 events, zone D's "time … session" counts from
the oldest event kept: 6,000 events over 1 h 40 min read "33m 20s session".
A wrong value with no sign of it, which AGENTS.md says to fix. The fix: the
state keeps the time of the first event it applied (where `log` appends to
`Log`), through the log's trimming and a resume, and zone D counts from it.
The value is today's while fewer than 2,000 events were applied, so no
golden frame changes. (`provider.SessionContext.StartedAt` exists, but
never reaches the state.)

### Item 2 - Two timed tests outside the timing step

`internal/app/perf_test.go` has no build tag: `TestPasteToFrame` (a
50,000-line paste, max ≤ 100 ms) and `TestStreamToFrame` (p95 ≤ 16 ms) judge
their targets through `overTarget` in every `go test ./...`, while
`doc/evaluation.md:86-87` says "The timed tests carry the `timing` build
tag, so `go test ./...` never runs them": the owner's rule of 2026-10-16,
after the keystroke tests missed their target from load alone (plan 4).
Besides the time, they check what no other test of the ordinary run does: a
50,000-line paste kept byte for byte, and a 2,000-line streamed answer kept
as one entry and drawn to its end.

- **(a) Recommended.** Their time judged only in the timing step, as
  `raceBuild` does under `-race`: a `timingBuild` constant (two files, the
  `timing` tag and its negation) makes `overTarget` log the time outside
  the timing step; both tests keep running in `go test ./...` for their
  other checks. The timing step runs six tests, and the command in AGENTS.md
  and evaluation.md names them.
- **(b)** evaluation.md corrected instead: these two are judged in every
  `go test ./...`; the timing step keeps its four tests.

### Item 3 - An undocumented field, unlabelled and without a fallback

`stream` (`internal/provider/claude/mapper.go:328`) merges Claude Code's
deltas into one message by the line's top-level `api_message_id`
(`mapper.go:48`). The documented `SDKPartialAssistantMessage` has `type`,
`event`, `parent_tool_use_id`, `uuid`, `session_id`, `ttft_ms`,
`user_message_uuid`, `user_message_uuids` and `resume_reason`, and no
`api_message_id` (code.claude.com/docs/en/agent-sdk/typescript, read on
2026-10-20); `doc/research/claude-code.md:423` does not list it, and the
code does not label it. Without it each delta becomes its own entry
(`internal/state/reduce.go:376-398`; `entry` returns nil for an empty id),
against AGENTS.md: "Anything undocumented is labelled so wherever it is
used, and the dashboard keeps working without it." In the 13 Claude Code
recordings (P1 to P303e), all 1,237 `stream_event` lines carry it, and each
equals the id of the same agent's last `message_start` (checked by script
on 2026-10-20). The fix, documented source first: a delta's message id is
the id of its agent's last `message_start` (the `event` field is the
Anthropic raw stream event, documented); `api_message_id` only when no
`message_start` came first, labelled undocumented in the mapper and in
claude-code.md's `stream_event` row. The live check `oneEntryPerMessage`
(`internal/app/livecheck_test.go:175-189`) stays.

### Item 4 - Zone C empty at 160 columns and 24 to 26 rows

From 160 columns with an agent in play, `cHeight`
(`internal/render/dashboard.go:41-53`) caps C at `max(min(region/2,
region-14), 3)`. At 160x24 to 160x26 the rows A, B and C share are 15 to 17,
so C gets 3 rows, `room` = (3-2)/2 = 0 agents (`dashboard.go:852`), and C
draws no agent: a blank list row, the preview's heading beside it, "↓ 1
below" in its border. Reproduced with P3b's agent at 160x24 and 160x26; from
27 rows C has 4 rows and the agent's two rows show. No size under 160
columns is affected.

- **(a) Recommended.** The floor raised: with an agent in play, C keeps one
  agent's rows (4 from 100 columns), A and B giving the rows: at 160x24,
  160x25 and 160x26, C 4 rows, A and B 11, 12 and 13 rows; from 27 rows, as
  today.
- **(b)** A minimum documented instead: ui-design.md (Zone C) and ASM-12 say
  that from 160 columns an agent's rows need 27 rows, and a test holds
  today's 3 rows at 160x24 to 160x26.

### Item 5 - Documents that promise what the code does not do

For each, the owner chooses: **build** the feature, or **correct** the
document to what is built.

- **5.1 The application's own configuration file.** architecture.md, How
  it starts, step 1 (`architecture.md:312-314`), with ASM-7 (its path and
  TOML format) and ASM-9; ui-design.md: "Colours are configurable in the
  application's own configuration file" (`ui-design.md:1497-1498`) and
  "Bindings are lists, configurable" (`:1597-1598`); and brief amendment 3,
  "Implement configurable Enter-to-send and newline shortcuts"
  (`doc/brief.md:296`, AM3 M7). Nothing reads such a file: key bindings and
  colours are fixed in the code, and no evaluation records the gap.
  Recommended, **build the two shortcuts only**, since amendment 3 asks for
  them and AGENTS.md makes its input MVP-essential: the send keys and the
  newline keys, each a list, read at start from
  `%APPDATA%\agent-command-center\config.json`. JSON, not ASM-7's TOML: Go's
  standard library reads JSON, and a TOML parser would be a new dependency,
  which needs the owner's approval. A file that cannot be read, an unknown
  key name, an empty list or a key in both lists: the first screen names
  the file and the problem, and the default keys apply. The other bindings
  and the colours stay fixed, and the documents say so. Or correct: the
  documents say keys and colours are fixed in the code, and amendment 3's
  shortcuts go under Nice to have by the owner's decision.
- **5.2 The sign-in kind.** architecture.md's Detect rows read it with
  `claude auth status` (`authMethod`) and `codex login status`
  (`architecture.md:169`, `:203`); both adapters say "The sign-in kind is
  not read yet" (`internal/provider/claude/provider.go:46`,
  `internal/provider/codex/provider.go:50`), and `Detection.SignIn`
  (`internal/provider/provider.go:21`) is never set or shown. Recommended,
  **correct**: the dashboard tells a subscription from an API key by the
  session's own reports, Claude Code's `system/init` `apiKeySource`
  (`claude/mapper.go:152-169`, the quota's N/A reason) and Codex's
  `account/rateLimits/read`, refused without a ChatGPT sign-in; the unused
  field and the two comments go. Building it would start two more processes
  at detection to show nothing the screen lacks.
- **5.3 The Capabilities flags.** architecture.md (`:157-163`): each adapter
  fills them from detection and the handshake, and "The UI shows "N/A" or
  hides an action when a flag is off". They are constants
  (`claude/session.go:352-353`, `codex/session.go:522-523`,
  `mock/session.go:153`), and only `AutoApprovals` is read
  (`internal/app/keys.go:165`, F4); the other per-CLI differences come from
  the provider id (`keys.go:422` `decision`, `state/reduce.go:665`,
  `state/actions.go:293`, `render/dashboard.go:214`) and from the events
  each CLI sends. Recommended, **correct**: rewiring working code to the
  flags would change nothing on screen (AGENTS.md: change what fails, not
  what works).
- **5.4 The frame rate.** ASM-3, "Rendering at most 30 frames per second,
  coalescing events per frame" (`architecture.md:425`), with
  architecture.md's Threads (`:118-122`), ui-design.md's Rendering
  (`:1570-1573`) and Assumptions (`:1929`), and events-and-state.md's
  Pipeline (`:15`). No `tea.WithFPS` is passed
  (`cmd/agent-command-center/main.go:37`, `internal/app/run.go:34`), so
  Bubble Tea v2.1.0's default of 60 applies (its `renderer.go:13`,
  `defaultFPS = 60`), and events are not coalesced by kind: `listen`
  (`internal/app/app.go:657-683`) hands over what is waiting, at most
  `maxDrain` = 512 events per update (`app.go:111`). Recommended,
  **correct**: every live review ran at 60; at 30, a key could wait up to
  33 ms for its frame, against up to 17 ms at 60.
- **5.5 The separation test.** reliability.md's test 7 "reads the package
  graph (`go list -deps`)" (`reliability.md:617-620`); the test parses the
  sources with `go/parser` (`internal/arch/separation_test.go:7`, `:53`).
  **Correct**: the test works, and its planted faults fail it.
- **5.6 ui-design.md's promises.**
  - a. A retry count "↻ n" on an agent's row (`ui-design.md:1040-1041`, the
    symbols' table `:1486`): amendment 2's "Retry information" in zone C
    (`doc/brief.md:151`, AM2 C7; `architecture.md:517`). The state counts
    each agent's retries (`internal/state/reduce.go:195-200`,
    `Agent.Retries`), and zone C draws none. Recommended, **build**: "↻ n"
    in amber after the agent's time when n ≥ 1, at every width. Codex gives
    an agent's retries (a thread's `error` with `willRetry`,
    `codex/mapper.go:459-465`); Claude Code's `system/api_retry` is the
    orchestrator's (`claude/mapper.go:200`), so a Claude Code agent shows
    none, and ui-design.md says so. Mapping Claude Code's documented
    `tool_progress` `subagent_retry` (claude-code.md:405) is left out: no
    recording holds one, so it could not be checked without a live retry.
  - b. The "ai" label, right-aligned "in lists and previews"
    (`ui-design.md:1462`); `originLabel` returns "" for AI
    (`internal/render/dashboard.go:690-695`). Recommended, **correct**: AI
    text is the text without a label, under the model's header in B; every
    other origin keeps its label, so AI messages stay distinct, as the
    owner reviewed them in plans 19 and 20.
  - c. The notices "interrupted by you", "turn failed: …" and "send refused:
    session not ready", grey, marked "dashboard" (`ui-design.md:872-873`;
    the States table, `:1447`). The code writes no such entry: a refused
    send says "not sent: the session is <phase>; the draft is kept" on the
    editor's status row (`internal/state/actions.go:286`), and an
    interrupted turn's stage in zone A reads "○ interrupted"
    (`internal/render/tree.go:130-131`). Recommended, **correct**:
    ui-design.md names what the code shows, and where.
  - d. "wrapping, Markdown and highlighting are cached per entry and width"
    (`ui-design.md:1591-1592`). The wrap is cached per source line and
    widths (`WrapCache.ranges`, `internal/render/content.go:235`), a
    finished entry's row count per entry, width and shape (`rowCount`,
    `content.go:210`); Markdown is classified again for each entry drawn
    (`mdLines`, `render/cards.go:135`, `content.go:391`). Recommended,
    **correct**: the stream's frame time is within its target
    (`TestStreamToFrame`) without that cache.
- **5.7 The MOCK badge on two full screens.** In a mock run, the file
  viewer full screen (f) and the draft window full screen (Ctrl+F) fill the
  terminal (`FileRect`, `internal/render/file.go:65-71`), so their frames
  have no title bar and no "◆ MOCK — not a live session"
  (`testdata/frames/file-full-*.txt` and `draft-full-*.txt` at 80x24,
  120x40 and 200x60), against reliability.md (`:472-475`, the title bar of
  every screen; test 8, `:621-625`) and evaluation.md's standing check
  (`:47`, "in mock mode every screen shows that it is mock data"). The
  inspection and zone D's details keep the title bar at full screen.
  Recommended, **build**: these two full screens start under the title
  bar, as the inspection does, live and mock; the golden test fails a
  mock-session frame without the badge.
- **5.8 ASM-23's path** (`architecture.md:445`) reads
  `%LOCALAPPDATA%agent-command-centerdrafts`, its backslashes lost.
  **Correct** to `%LOCALAPPDATA%\agent-command-center\drafts`
  (`internal/input/draft.go:23-25`, `os.UserCacheDir`).

### Criteria

Each criterion of items 2, 4 and 5 applies to the choice the owner makes:
"(a)", "(b)", "Build" or "Correct" marks it.

Item 1, the session time (a test in `internal/render`).

- 1.1 6,000 events one second apart from t0, the frame at t0 + 6,000 s:
  zone D reads "time 1h 40m session"; today's code reads "33m 20s session"
  (the check run once on it, and failing).
- 1.2 1,999 events: the value today's code gives; no event: "N/A (…)", as
  TestTelemetryNA checks today.
- 1.3 A resume after a crash, past 2,000 events: the time still counts from
  the first event.
- 1.4 No golden frame changes; events-and-state.md's State names what is
  kept.

Item 2, the timed tests.

- 2.1 (a) With both targets set to 1 ns, a mutation reverted after: `go
  test ./internal/app -run '^(TestPasteToFrame|TestStreamToFrame)$'`
  passes, each test logging its time as not judged; with `-tags timing`
  both fail.
- 2.2 (a) Both tests still run in `go test ./... -v` and pass, their checks
  besides time unchanged.
- 2.3 (a) The timing step runs the six tests, each within its target, its
  p50, p95 and maximum recorded in plan.md with the conditions
  (evaluation.md, Keystroke timing).
- 2.4 (a) AGENTS.md (Commands) and evaluation.md (Keystroke timing) give the
  six-test command, and evaluation.md says what the code does.
- 2.1 (b) evaluation.md says these two are judged in `go test ./...`, the
  four others in the timing step; the timing step 4 of 4 within its
  targets.

Item 3, a delta's message id (`internal/provider/claude`).

- 3.1 In each of the 13 Claude Code recordings, every `stream_event` line
  maps to the id of its agent's last `message_start`, equal to its
  `api_message_id`: 1,237 of 1,237 lines (a test over
  `testdata/probes`).
- 3.2 Each recording replayed with `api_message_id` removed from every line
  gives the same conversation as replayed as is (roles, message ids,
  texts), one entry per message (`oneEntryPerMessage` on each replay).
- 3.3 A delta with no `message_start` before it takes its `api_message_id`.
- 3.4 The code that reads `api_message_id` says it is undocumented; so does
  claude-code.md's `stream_event` row, with the recordings that show it.
- 3.5 Checked by mutation: a mapper that reads neither id fails 3.2.

Item 4, zone C (TestAdaptiveLayout, `internal/render`).

- 4.1 (a) P3b's agent at 160x24, 160x25 and 160x26: C 4 rows, the agent's
  label "general-purpose" on C's first row inside its border, left of the
  rule, the preview's heading right of it; A's and B's bottom borders on
  one row, A and B 11, 12 and 13 rows; no row wider than the screen.
- 4.2 (a) At 160x27, and at every size TestAdaptiveLayout and
  TestDashboardGeometry check today, the heights of today.
- 4.3 (a) ASM-12 and ui-design.md (Zone C) state the floor.
- 4.1 (b) ASM-12 and ui-design.md (Zone C) state the 27-row minimum; a
  TestAdaptiveLayout case holds C at 3 rows with "↓ 1 below" in its border
  at 160x24 to 160x26.

Item 5, per the owner's choice for each.

- 5.1 Build: (i) no file: Enter sends, Ctrl+J and Shift+Enter add a newline,
  and the existing input tests pass unchanged; (ii) `{"keys": {"send":
  ["ctrl+enter"], "newline": ["enter"]}}`: Ctrl+Enter sends, Enter adds a
  newline, Ctrl+J and Shift+Enter neither send nor add a newline; (iii) the
  footer and the F1 page show the keys in force; (iv) a bracketed paste
  ending with a newline never sends, with either file
  (TestPasteNeverSubmits); (v) invalid JSON, an unknown key name, an empty
  list, a key in both lists: a message naming `config.json` and the
  problem on the first screen (the startup screen; the editor's status row
  in a mock run), and the default keys apply; (vi) the same input tests
  pass with the Claude Code, Codex and mock adapters; (vii) the file is
  only read, nothing else in it is used, `go.mod` unchanged; (viii)
  architecture.md (How it starts, ASM-7, ASM-9) and ui-design.md (Focus
  and keys, Colours) say what is built. Correct: those passages say keys
  and colours are fixed in the code, and plan.md's Nice to have lists
  amendment 3's configurable shortcuts with the owner's decision.
- 5.2 Correct: architecture.md's Detect rows name no status command and say
  which reports tell a subscription from an API key; `SignIn` and the two
  "not read yet" comments gone; `go build ./...` exit 0.
- 5.3 Correct: architecture.md's Capabilities paragraph names the one flag
  read and the provider-id checks, each with its file.
- 5.4 Correct: architecture.md (Threads, ASM-3), ui-design.md (Rendering,
  Assumptions) and events-and-state.md (Pipeline) say 60 frames per second,
  Bubble Tea's default, and a burst of at most 512 events applied as one
  update; none of them says 30 frames per second.
- 5.5 Correct: reliability.md's test 7 says the test parses the sources
  with `go/parser`.
- 5.6a Build: an agent with 2 retries shows "↻ 2" in amber after its time
  at 120x40, 200x60 and 80x24, none with 0 (a test from a Codex thread's
  `error` with `willRetry`); ui-design.md says why a Claude Code agent
  shows none. Correct: ui-design.md has no "↻ n", and says retries are
  counted in zone D only.
- 5.6b Correct: ui-design.md's Origins table says AI text carries no label.
  Build: "ai" right-aligned on the AI rows of C's preview and the
  inspection, their golden frames rewritten.
- 5.6c Correct: ui-design.md names the texts shown for a refused send, an
  interrupted turn and a failed one, each found in the code, and no longer
  promises "interrupted by you" or "send refused". Build: the three notices
  as entries in B, marked "dashboard", a test for each.
- 5.6d Correct: ui-design.md's Rendering says what is cached and what is
  not.
- 5.7 Build: the six frames `file-full-*` and `draft-full-*` at 80x24,
  120x40 and 200x60 begin with the title bar and "◆ MOCK — not a live
  session", each window one row shorter; TestGoldenFrames fails any frame
  of a mock session without the badge (run once on today's frames: those
  six fail); the viewer's and the draft window's tests pass. Correct:
  reliability.md and evaluation.md name the two exceptions.
- 5.8 ASM-23 reads `%LOCALAPPDATA%\agent-command-center\drafts`, checked
  from the bytes.

Item 6, regressions and checks.

- 6.1 `go build ./...` and `go vet ./...` exit 0; `gofmt -l .` empty; `go
  test ./...` 0 FAIL, only TestDiskListLink skipped; `go.mod` and `go.sum`
  unchanged.
- 6.2 The timing step within its targets: 6 of 6 under 2 (a), 4 of 4 under
  2 (b).
- 6.3 Only the golden frames the chosen items change, each diff reviewed:
  the six full-screen frames under 5.7's build; none for items 1 to 4.
- 6.4 A mock run in Windows Terminal, maximised: the file viewer and the
  draft window at full screen with the badge (5.7); then a 160x24 grid,
  P3b's agent's rows in C (item 4); screenshots for the owner, the look
  listed as not verified until the owner has seen them.
- 6.5 AGENTS.md's status line names plan 21; plan.md holds the progress;
  `doc/plan21-evaluation.md` the summary; every changed file UTF-8 without
  BOM, LF only, checked from the bytes.
- 6.6 `git status` lists only the intended files; no commit before the
  owner's approval.

### Progress

- 2026-10-20: proposed from the review of d84e5d4. Each finding checked
  again at its file and line; the Agent SDK's `SDKPartialAssistantMessage`
  read; the 1,237 `stream_event` lines of the recordings checked by
  script; `go test ./... -count=1` exit 0 at d84e5d4. Waiting for the
  owner's approval and decisions.
- 2026-10-20: approved with the recommended choices; built the same day.
  Item 3 and item 5.1 by an implementer agent each (Sonnet), reviewed and
  rerun here; the rest here.
  - Item 1: `State.Applied` and `State.FirstAt` (`internal/state`), set
    where `log` appends; `sessionTime` counts from `FirstAt`.
    TestSessionTimeKeepsFirstEvent (`internal/render`): run once on the old
    code, it failed with "time 33m 20s session" for 6,000 events over
    1 h 40 min and "34m 55s session" after a resume; now 1.1, 1.2
    ("33m 19s session" for 1,999 events, as before; N/A with no event) and
    1.3 ("1h 00m session") pass.
  - Item 2 (a): `timingBuild` (`timing_on_test.go`, `timing_off_test.go`);
    `overTarget` logs outside the timing step. Mutation, both targets
    1 ns: `go test ./internal/app -run '^(TestPasteToFrame|TestStreamToFrame)$'`
    ok, each logging "not judged outside the timing step"; with `-tags
    timing` both FAIL ("max 2.9571ms over the 1ns target", "p95 5.986ms
    over the 1ns target"); targets restored.
  - Item 3: the mapper takes a delta's id from its agent's last
    `message_start`, `api_message_id` (labelled undocumented) only before
    one. TestStreamEventsTakeTheMessageStartID: 1,237 lines in 13 of 14
    recordings (p4 has none); TestConversationDoesNotNeedAPIMessageID: the
    14 recordings replayed with and without the field, equal, one entry per
    message; TestStreamEventWithoutMessageStartFallsBack; mutation (the
    mapper reading neither id): both tests fail ("assistant entry with
    message id \"\" (empty or repeated)").
  - Item 4 (a): `agentC` = 4, the floor in `cHeight`. TestZoneCFloor: on
    the old code C 3 and no agent at 160x24 to 160x26; now C 4, A and B 11,
    12, 13, the label left of the preview's rule; 160x27 as before.
  - Item 5.1: `internal/input/config.go` (`LoadKeys`, `NormaliseKey`),
    read in `main.go`; `editorKey` and `draftWinKey` use the lists; the
    footer, status row, draft window and F1 page show the keys in force
    (`internal/render/keyset.go`). Tests: TestLoadKeys* (`internal/input`),
    TestConfiguredKeys, TestPasteNeverSubmitsConfigured,
    TestConfiguredKeysShown, TestConfiguredDraftWindow,
    TestKeysProblemOnStartup, TestKeysProblemInMockRun (`internal/app`).
  - Item 5.6a: "↻ n" in amber after the agent's time (`agentRetries`).
    TestAgentRetries, from two `error` notifications with `willRetry` on a
    child thread of P7 through Codex's mapper, at 120x40, 200x60, 80x24;
    mutation (no count drawn): it fails at the three sizes.
  - Item 5.7: `FileRect` full screen under the title bar. The new check in
    TestGoldenFrames, run once with the old layout: exactly the six
    `file-full-*` and `draft-full-*` frames failed it; frames rewritten,
    those six only, each the title bar added and the window one row
    shorter. TestFileRect, TestDraftWindow and TestFileKeys updated to the
    new full screen.
  - Documents: architecture.md (Provider interface, Capabilities, the two
    Detect rows, Threads, How it starts, ASM-3, ASM-7, ASM-9, ASM-12,
    ASM-13, ASM-23), ui-design.md (Zone B's notices, Zone C, States,
    Origins, Colours, Rendering, Focus and keys, File viewer, Draft
    window, Assumptions), events-and-state.md (Pipeline, State, Polling),
    reliability.md (test 7, test 8), evaluation.md (Keystroke timing),
    claude-code.md (`stream_event`), AGENTS.md (status, timing command).
  - Checks: `go build ./...`, `go vet ./...` exit 0; `gofmt -l .` empty;
    `go test ./... -count=1 -v` exit 0, 0 FAIL, only TestDiskListLink
    skipped, TestPasteToFrame and TestStreamToFrame run and pass; `go.mod`
    and `go.sum` unchanged; 48 changed or new files UTF-8, no BOM, no CR.
  - Windows Terminal, the P3b mock (`scratchpad\term\p21shot.ps1`,
    plan 20's harness): maximised 156x44, the file viewer and the draft
    window full screen, each under the title bar with "◆ MOCK — not a live
    session"; a 160x24 grid (147x22 and one Ctrl+minus): C 4 rows, the
    agent's two rows left of the preview, A and B 11 rows. Screenshots for
    the owner.
  - The timing step (2.3, 6.2), not run on battery (status 1, 37% then
    34%), evaluation.md requiring mains power; run alone at 19:40 once on
    mains (status 2, 30%), the CPU at 37%, the usual applications open:
    TestKeystrokeToFrame p50 0 s, p95 1.0062 ms, max 1.9993 ms (draft
    window 503.4 µs, 1.3271 ms, 9.5393 ms); TestDraftKeystrokes 1.034 ms,
    3.0566 ms, 41.7277 ms; TestKeystrokeCost 524 µs and 1.27 ms a key;
    TestPasteToFrame p50 3.5169 ms, max 4.2555 ms; TestStreamToFrame
    3.0111 ms, 8.9818 ms, 21.9225 ms; **TestLongSession FAIL**, p50
    10.6808 ms, p95 16.6898 ms over 16 ms, max 22.169 ms. 5 of 6.
  - The miss investigated, not rerun: d84e5d4 (plan 20, committed) p95
    16.10 ms, FAIL; 8b3b16b (plan 18) p95 2.65 ms, PASS; a CPU profile:
    `agentsDone` (plan 20's RESULT rule, `internal/render/look.go`) 32%,
    walking every agent for every exchange each frame. Proposed, tried in
    a scratch copy only: prefix sums of each stage's agents not completed,
    once a frame; TestLongSession p95 4.19 ms, render tests and golden
    frames unchanged. Waiting for the owner's approval
    (`doc/plan21-evaluation.md`: 38 PASS, 2 FAIL, the same miss).
- 2026-10-20: the owner approved the fix and the commit ("Apply the speed
  fix, then commit"). `undoneByStage` and the new `agentsDone`
  (`internal/render/look.go`); TestAgentsDoneCounts: the rule's answer for
  every pair of turns in P3b, P7 and a synthetic session, two mutations
  each failing it. `go test ./... -count=1` 0 FAIL, TestDiskListLink
  skipped, no golden frame changed by the fix. The timing step, alone, at
  20:00 on mains (status 2, 53%), the CPU at 29%: 6 of 6;
  TestKeystrokeToFrame p50 0 s, p95 1.0415 ms, max 2.0003 ms (draft window
  0 s, 1.1376 ms, 2.1069 ms); TestDraftKeystrokes 1.0121 ms, 3.0418 ms,
  5.391 ms; TestKeystrokeCost 220 µs and 556 µs a key; TestLongSession
  2.0002 ms, 3.9032 ms, 5.1539 ms (last tenth 3.54 ms); TestPasteToFrame
  p50 1.0041 ms, max 1.1631 ms; TestStreamToFrame 1.5154 ms, 3.515 ms,
  15.5502 ms. Committed; not pushed.

## Closed plans

One entry each: what the plan did, its approval, its evaluation file and
its commits. Their criteria and progress logs: `git show 8888f59:plan.md`
(plans 1 to 13), `git show 84d2ad8:plan.md` (plan 14), `git show
b047434:plan.md` (plan 15), `git show becfb34:plan.md` (plan 16), `git show
f00c2d8:plan.md` (plan 17), `git show 8b3b16b:plan.md` (plan 18), `git show
d84e5d4:plan.md` (plan 20).

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
