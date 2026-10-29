# Reliability

Plan 2, 2026-10-16; the brief's phase 3 (B3.1-B3.6). How the application keeps
working, and keeps showing the truth, when a CLI, an event source or a
telemetry source is missing or fails; how approvals, interruption,
cancellation and resumption work without losing the session; how every CLI's
configuration stays untouched; mock mode; and the test plan. The parts are
in [architecture.md](architecture.md), the events and state machines in
[events-and-state.md](events-and-state.md), the screens in
[ui-design.md](ui-design.md). Citation keys are those of
[architecture.md, Conventions](architecture.md#conventions).

Three rules apply everywhere (AGENTS.md):

- **No silent fallback.** A fallback is shown as such: its origin label, its
  reason, or "N/A". The dashboard never switches integration, restarts a CLI
  or answers an approval on its own.
- **No invented value.** Missing data is "N/A" with its reason, never zero,
  never a guess.
- **Read only towards every CLI's configuration**, in code and in tests.

## Separated parts

The five parts of [architecture.md, The five parts](architecture.md#the-five-parts)
(B3.1; AM2 E5), and what each failure can and cannot reach:

| Part | A failure here | Cannot reach |
| - | - | - |
| 1. Execution and session | a spawn error, a closed pipe, a timeout: becomes a session event (`Crashed`, a refused command) | the screen directly; it reports through events |
| 2. Event collection | a line that does not decode: becomes an `unrecognised` event with the parse error, counted and listed in the details view (plan 26: zone A draws no counts) | the state: it never applies anything itself |
| 3. Application state | a reducer bug: caught by the test plan; at runtime a panic is recovered, logged, and the terminal restored | the CLI: the reducer does no I/O |
| 4. Rendering | a layout error: only the frame is wrong; the state is intact and the next frame redraws | the state or the CLI |
| 5. Input and commands | a refused command (not ready, turn running): a visible notice, the draft kept | the CLI directly: commands go through part 1 |

**Rendering reads the state only.** Part 4 gets a snapshot and returns a
frame; it holds no data of its own apart from caches keyed by state and
width ([ui-design.md, Rendering](ui-design.md#rendering)). The separation is
enforced by a test of the package graph (see [Test plan](#test-plan), item
7): the rendering package imports no adapter or process package, the editor
imports no provider package, and the reducer imports no I/O, process,
network or clock package.

**The dashboard's own crash.** A panic in the Bubble Tea loop is recovered:
the terminal is restored (alternate screen, mouse and keyboard modes reset),
the CLI is closed with the normal escalation, and the panic is printed after
the terminal is restored. On Windows each CLI process is started in a job
object that kills it when the dashboard's process ends, so no CLI is left
orphaned if the dashboard is killed (ASM-19). Codex killed mid-turn by its
job object leaves no process behind: all 9 were gone 24 ms after the kill
(plan 4, L3; CX-10 q8).

**Panics in the dashboard's own goroutines** (plan 48, design D5; audit
finding 6). Bubble Tea recovers its loop and its commands, but not the
goroutines the dashboard starts itself, where a panic would end the program
with the terminal still in its modes. Those that handle what comes from
outside recover:
- each adapter's stdout reader: the panic becomes an error marked `Fatal`,
  and the rest of stdout is read and dropped, so the CLI never blocks on a
  full pipe;
- the goroutines an adapter starts for its writes (Codex's replies and
  handshake steps, Claude Code's handshake reads and settings read): the
  same error;
- the application's write worker: `Panicked`;
- the detection: the CLI's problem, "detection failed: …", and the startup
  screen goes on.

A fatal error or `Panicked` crashes a session that is starting or ready,
the panic's value and place as the reason on the failure card ("panic: …
(at function, file:line)"), and closes its CLI; while the owner's quit
closes it, it is only kept as an error, so the quit still ends. The
adapters map lines under their mutex in functions that release it by
`defer`, so a recovered panic leaves no lock held. The stderr readers, the
queue's forwarder, the waits and the mock replay parse nothing from
outside and do not recover.

**Bounds on what comes from outside** (plan 48):
- the TIFF reader reads only the tags it uses, the first of each, their
  values from the file when used: a crafted file allocates nothing in
  proportion to its tag counts (design D1);
- an icon's PNG has its size checked before it is decoded (design D1);
- a running command's streamed output keeps its last 64 KiB, the rest
  counted and said in the inspection and the full view; the completed item
  replaces it whole (design D2);
- the Claude Code reader never writes to stdin, as Codex's never did: a CLI
  that stops reading its stdin cannot stop the reader (design D3);
- a git call waits at most 2 s, once git has ended, for its pipes to close
  (`WaitDelay`): a hook or helper that keeps them cannot hold a read, a
  commit or a push for ever, and git's success is taken as such (design
  D4).

## Fallbacks for every source

What happens when each event source of
[events-and-state.md, Conceptual events](events-and-state.md#conceptual-events)
is absent, and what the screen shows meanwhile (B3.3; AM2 U9). "Absent" means:
not emitted by this CLI or version, not yet received, or malformed.

| Event | When its source is absent | On screen |
| - | - | - |
| `conversation.message.delta` | the complete message is rendered when it arrives (INT 2, both CLIs). Claude Code subagent text: never deltas (CC-10 q15); whole forwarded messages instead | B's badge says "◉ Running", not "Streaming"; the text appears whole |
| `conversation.message.completed` | if deltas were received, the entry is closed at the turn's end with the text received | "incomplete: no final message" under the entry |
| `tool.started` | a result without a start creates the tool line from the result | the line, with "start not reported" |
| `tool.completed` | at the turn's end the tool becomes `Unfinished` | "unfinished", inferred |
| `workflow.stage.changed` | Claude Code: always absent; Codex: when no plan is reported. The stage tree is derived from turns and task events | A's heading "STAGES inferred from activity" |
| `agent.started` | an event carrying an unknown `agent_id` or `threadId` creates the agent (INT 3 fallback) | the agent, with "start not reported", inferred |
| `agent.progress` | elapsed time from the agent's own events only | tokens "N/A (no progress report)" |
| `agent.completed`, `agent.failed` | Codex: a child's `idle` status after its turn ends it as completed, inferred; otherwise the agent stays `Running` until the session closes, then `Ended` | "ended, status not reported" |
| `approval.requested` | the CLI's own rules decide (a `-p` session denies what would prompt and reports `system/permission_denied`) | the denial as an observed error line |
| `approval.resolved` | Claude Code: the answer is observed as sent. Codex: the same when `serverRequest/resolved` does not arrive | "allowed by you" / "denied by you" |
| `question.requested` | as events-and-state.md: Claude Code shows the questions and takes the answer as the next message; Codex questions arrive as plain agent text | the card, or the agent's text |
| `diff.updated` | Claude Code: derived from `Edit` and `Write` inputs; a tool with no such input has no diff | Diff tab "derived", or "N/A (no diff reported)" |
| `telemetry.updated` | per value, in [Telemetry sources](#telemetry-sources) | per value |
| `session.started` | 60 s without the handshake ends the start (events-and-state.md, Session machine) | the StartFailed card |
| `session.ended` | always observed: the reader sees the exit or the end of stdout | the crash card ([Resumption](#resumption)) |
| `turn.started` | inferred when the message is written; observed at the first event | "Submitted" until the first event |
| `turn.completed` | if the process ends, the turn ends `failed`, "session ended". A turn that simply runs long has no timeout: long tools are legitimate | elapsed time keeps ticking; Ctrl+C stays available |
| `retry` | not counted (both are **D**, not observed) | retries "N/A (none reported)" until one arrives |
| `error` | the other error fields still count (TEL) | the counter |
| `status.changed` | no compaction or MCP status shown | nothing extra |
| `models.listed` | Claude Code: an `initialize` response without `models`; Codex: a `model/list` page refused (before any list: no list) or unreadable (the list stays unfinished); a page cursor that comes back, or a hundredth page, ends the list there, the models so far kept, the reason an error | no selector: the pill keeps the model in force, and its click or Alt+P says "the session has not listed its models yet", or "no model list: <the CLI's words>" for a refused page (plan 34) |
| `models.settings` | Claude Code: `get_settings` refused or unanswered (undocumented, so a version could drop it); Codex: nothing after `thread/start` until a turn carries a change | the pill keeps the last values read, else the session's model, no effort while none is known, no Ultracode; the Ultracode row dimmed, "Ultracode's state is not known: <reason>"; a change asked for is answered with the CLI's words in zone B, never shown as applied (plan 34) |
| `unrecognised` | never absent: anything unknown lands here | counted in the details view, where the last 20 are named with their time, type, reason and line (plan 25; the count left zone A's notes in plan 26), listed in Raw events |

### Integration sources

Every source of [INT](research/integration.md) sections 1 to 9, by CLI. The
MVP builds the primary ways only; the fallback ways are designed so that they
can be added behind the provider interface without touching other parts, and
are built only with the owner's approval ([When a whole integration
fails](#when-a-whole-integration-fails)).

| INT | Claude Code primary; fallback | Codex primary; fallback |
| - | - | - |
| 1. Controlling a session | stream-json over stdio (way A); fallback way B (`-p --resume` per turn) or C (SDK sidecar) | `codex app-server`; fallback `codex exec --json` + `exec resume` |
| 2. Streaming | `stream_event` deltas; fallback whole `assistant` messages | `item/agentMessage/delta`; fallback `item/completed` |
| 3. Subagents and tools | task messages and `agent_id`; fallback hooks through `initialize` (CC-10 q5, closed: not needed) or transcript files (undocumented, labelled) | `subAgentActivity` and per-thread events; fallback `thread/list` + `thread/read`, then inferred agents |
| 4. Approvals | `can_use_tool` (`stdio` is **U**); fallback way D, an MCP permission tool; an unreadable request is denied and shown | `requestApproval` requests; an unreadable request is declined, an unknown method gets a JSON-RPC error, both shown |
| 5. Persistence, cancel, resume | `interrupt`, `--resume`; fallback transcript file (labelled undocumented), then close stdin and end the process | `turn/interrupt`, `thread/resume`; fallback `thread/read`, `codex exec resume`, rollout file (labelled undocumented) |
| 6. Reusing the TUI | not reused; escape hatch `claude --resume <id>` in another terminal (outside the MVP) | not reused; escape hatch `codex resume <id>` (outside the MVP) |
| 7. Concurrent output | route by `agent_id`, `task_id`, `parent_tool_use_id`; fallback "unattributed" on the orchestrator | route by `threadId`; fallback `optOutNotificationMethods`, per-thread `thread/read` |
| 8. Quota | the `/usage` report after the handshake and after each turn (plan 24, amendment 23), `rate_limit_event` between; fallback last value with its age, "N/A" before the first | `account/rateLimits/read` and pushes; fallback "N/A" with the reason, last pushed value with its age |
| 9. Platform limits | Windows native, Git Bash for the Bash tool; fallback its PowerShell tool, the reported shell shown | Windows native; the effective sandbox from `thread/start` shown, so a downgrade is visible |
| Network path | the proxy from Claude Code's own settings, never set by the dashboard | the dashboard's environment. The proxy comes from `config.json`'s `codex.proxy` when set (plan 22, amendment 19), shown on the startup screen. If it does not answer, Codex's own errors and retries appear in zone B as received: nothing is hidden, and nothing switches proxy |
| Path forms (INT paths) | the validation steps and messages of [ui-design.md, Validation and messages](ui-design.md#validation-and-messages); a failed `Stat` after a WSL translation names the mount root as the likely cause | same |

### Concurrent output

One ordered stream per CLI, one reader, routing by key
([events-and-state.md, Concurrent output](events-and-state.md#concurrent-output);
INT 7):

- An event with no agent key is attached to the orchestrator and marked
  "unattributed", never guessed (INT 7).
- High-rate kinds are coalesced to one state update per frame (ASM-3), so 11
  rate-limit pushes in one turn (P7) cost one redraw each frame at most.
- The reader has no small line limit: a line is read whole, whatever its
  length (a tool result can be megabytes), up to 64 MiB; a longer line is
  recorded as an error with its size and skipped, and the stream continues
  (ASM-20). A line that is not valid JSON becomes `unrecognised` with the
  parse error.
- The queue between the reader and the forwarder is unbounded, so a paused
  loop (`$EDITOR`) never blocks the CLI's pipe (architecture.md, Threads of
  execution). Memory over a long session is measured by test 11
  (`TestLongSession`), which replays a long generated session. Since plan
  49 (design D8) one implementation, `pipe.Proc`, holds the queue, the
  readers, the wait and the close for both adapters, its messages naming
  the CLI as before; its own tests pin the exit as the last event, the
  readers given up 2 s after the exit, and the close's escalation.
- Codex can refuse a request with `-32001` when overloaded (INT 1): the
  refusal is an error with Codex's words, and nothing is retried
  automatically. A refused `turn/start` starts no turn: since plan 34 (the
  review's finding) the turn ends, "the message could not be sent:
  turn/start refused by Codex: <its words>", and the draft is restored;
  before, the turn stayed "Submitted" and only Ctrl+Q was left.

### The file viewer

Plan 17 (ui-design.md, File viewer; architecture.md, File viewer). It only
reads, in commands, so a slow disk or a large file never freezes the loop.

| Failure | On screen |
| - | - |
| the file is missing, unreadable, a directory, over 32 MiB, outside the project, or a credential file | the reason in red in the window, nothing shown of the file |
| a NUL in its first 8 KiB | "binary file, not shown" and its size |
| invalid UTF-8 | U+FFFD where the bytes were; the rest exact |
| colouring not back yet | the lines uncoloured, "colouring…" in the header |
| a file over 1 MiB | coloured 1000 lines on either side of the view, "coloured around the view": a comment or string opened above that range may be coloured wrongly |
| Chroma fails, or its runs do not join to the line byte for byte | the lines uncoloured, the text exact |
| a Chroma pattern over its 250 ms match timeout | Chroma treats it as no match (Chroma v2.27.0, `regexp.go`, `matchRules`); text no pattern matches is an error token, drawn in the text colour |
| the listing over 100,000 files | "first 100,000 files" |
| a read, colouring, search or listing for an older open | dropped (generation numbers) |
| the Explorer: a folder that cannot be read | its reason in the attention colour under it; the rest of the tree shown |
| the Explorer: a folder over 10,000 entries | "first 10,000 entries" under it |
| the Explorer: outside a repository, or git not on the PATH | no markers; "not a git repository" or "git not found" in grey on the path row |
| the Explorer: git fails otherwise | no markers; "git: <its error>" in the attention colour |
| the Explorer: a folder read or git status for an Explorer since closed | dropped (generation numbers) |
| the Explorer: copying from it | no drag selection, as in the quick open; F12 gives the terminal's own selection |

The Explorer's git status passes `--no-optional-locks`, so git does not
refresh its index while answering, and writes nothing in the repository.
The quick open lists the files git ignores, as the Explorer shows them
dimmed: ignoring is git's, not a reason to hide a file.

### The turn review

Plan 65 (brief amendment 67; ui-design.md, Turn review). The snapshots go
to a store of the dashboard's own, never to the project or its `.git`
(plan 65, What was found: every file of `.git`, path, size and time,
unchanged in a main and a linked worktree and a repository of 20,000
files), one time aside: a split index's shared file, which git refreshes
when it reads it (the table below). They run in an ordered queue of their
own, off the loop: a turn's end is always taken before the next turn's
start.

| Failure | What happens |
| - | - |
| the project outside a git repository, or git not on the PATH | no store, no snapshot, no row; Source Control says why |
| the store cannot be made | each turn's row: "No review of turn N: the review's store could not be made: …" |
| a start's snapshot past 5 s from the send | the message goes at 5 s; the snapshot, running, is stopped, or, still queued, never taken; the turn's row says so |
| an end's snapshot past 30 s | stopped; the turn's row says so; no message waits for it |
| a snapshot stopped midway | its `index.lock` in the store removed, so the next one works |
| the store would pass 256 MiB on disk (its size and the files to take in), or has | that snapshot not taken, and none after; the turns without both snapshots say so; what was taken stays readable |
| a file over 32 MiB, outside the project once its links and junctions are resolved, or a credential file | never read; listed as left out when its size or time changed |
| a file with more than one name, a hard link (the review of 2026-10-29) | left out, "a hard link": the checks see the link's name alone, and its other names, which no path shows, may be a credential file's or outside the project; opened for its attributes alone, never read; a project's own hard-linked files are left out with them |
| a file written while a snapshot is taken | caught before or after; the listing after the end marks the turn "still changing" |
| a `git gc --prune=now` removing an object only the store's index named | that side shows git's error, never another content |
| the dashboard quits | the snapshot running stopped, waited for 3 s, the store removed |
| the dashboard crashes or is killed | the store stays on this machine, readable by the user alone, until a dashboard starts again and removes every store no running dashboard holds |
| a turn that takes in many files at once | git writes about 7 ms a new file here (2,000 small files 13.9 s, plan 65's store tests): past about 700 the start's 5 s pass, the message going and the turn without a review; past about 4,000 the end's 30 s; once a snapshot has taken them, the next is quick again |
| a repository with Git LFS | the filters off, every checked-out LFS file differs from its pointer in the seeded index: the first snapshot takes each in as its bytes, 32 MiB at most each, so that with over 256 MiB of them the store is full from the first snapshot and the session has no review |
| a repository whose index is split already (`core.splitIndex`) | the store writes nothing in `.git`, but git itself refreshes the time of the shared index file each time it reads it, so that no git deletes a shared index in use (git's documentation of `core.splitIndex`), Source Control's reads as much as the store's: the store's first snapshot reads it through its copy of the index. A time alone, a documented exception (plan65-evaluation.md): the file's bytes, the index, the references and every other file of `.git` and of the project stay as they were; the store writes its own index whole (`core.splitIndex=false`), so no later call reads the shared one (TestTurnSplitIndex) |
| an untracked folder holding a repository of its own | left out, "a nested repository": git would refuse it, and the whole snapshot with it |
| a panic in a snapshot | that turn's problem, said in its row, never the session's or the program's end; its gate opened, the message going |
| more than 16 snapshots waiting | the one more refused, its turn without a review, saying so |

Every git call of the store runs with its index and its object folder, its
hooks at an empty folder, every filter driver of git's configuration given
empty commands, the file system monitor off; the repository's objects
behind the store's only to read. Git's own walk enters a junction to a
folder outside the project: each file it lists passes the file viewer's
checks before git reads it.

## Telemetry sources

Every source of [TEL](research/telemetry-sources.md), with its fallback (AM2
D1-D9; B3.3). Values follow TEL's rules: N/A until a source gives the value,
every pushed value with its age, cost apart from quota, a window past its
reset shown N/A.

**Claude Code: ways** (TEL CC, first table)

| Way | Use | When unavailable |
| - | - | - |
| stream-json messages and control responses | primary | the session cannot start: StartFailed (see [When a whole integration fails](#when-a-whole-integration-fails)) |
| `/usage` sent as a user message | the quota's primary source since plan 24 (amendment 23; P24u): written right after the handshake's answer and after each turn's `result`, answered by Claude Code itself with no model call, its documented `usage_report` holding the session, weekly and model-scoped windows ("Fable limit") with their reset times; the exchange is kept out of the conversation, the stages and zone C (events-and-state.md, Polling) | the last values stay with their age, and one error, "usage report not read: <reason>"; `rate_limit_event` keeps updating the 5-hour and 7-day windows; before the first report "N/A (no report yet)" |
| Status line JSON | not available to a `-p` session (CC-6) | nothing lost; its field names guide zone A's telemetry |
| OpenTelemetry | not used: no quota metric, duplicates the stream | nothing lost |
| Hooks | not used for telemetry | nothing lost |
| Transcript files | not used for telemetry; last resort only, labelled undocumented | nothing lost |

**Claude Code: values** (TEL CC, second table)

| Value | Source | Fallback, and what zone A shows |
| - | - | - |
| 5-hour, 7-day quota | the `/usage` report's `session` and `weekly_all` rows (documented; plan 24, amendment 23), read right after the handshake and after each turn; between reads `rate_limit_event` (`unifiedWindows` **U**, marked †) | before the first report "N/A (no report yet)"; with an API key "N/A (API key: no subscription window)"; past the reset "N/A (window reset)"; without `unifiedWindows` but with `rateLimitType` and `status`: the status word ("allowed", "warning", "rejected") with no percentage |
| Model-scoped weekly windows | the `/usage` report's `weekly_scoped` rows, `scope.model.display_name` naming the model ("Fable limit"; documented; plan 24, amendment 23), and `rate_limit_event`: a `unifiedWindows` key or a `rateLimitType` of the form `seven_day_<model>` (a key that is no model's, `seven_day_overage_included` on 2.1.303 since 2026-10-22, is skipped: plan 25; **U** for `unifiedWindows`, marked †; `seven_day_opus` and `seven_day_sonnet` are documented `rateLimitType` values: claude-code.md, section 7; TEL CC); metric `quota.7d.<model>` | shown as "<Model> limit" only once reported, never assumed from the model or the plan: no report, no row; past the reset "N/A (window reset)", as the quota row above |
| Reset times | `resetsAt` | with the window |
| Overage | `overageStatus` and related | "N/A (not reported)", shown apart from quota |
| Model | `system/init`, per message; since plan 34 the `get_settings` read below | a subagent whose text is not forwarded: "model N/A" |
| Model, effort and Ultracode in force (plan 34) | `get_settings` `applied` (undocumented, marked † in the details view), read after the handshake, each change's answer and each turn; the effort's variable `CLAUDE_CODE_EFFORT_LEVEL` from the dashboard's environment or the read's `env`, its value only | before the first read "model N/A", no effort, no Ultracode in the pill; a refused read keeps the last values, its reason in the details view and, after a change, in zone B; an `apply_flag_settings` success for a level the CLI dropped (P34c, "extreme") is caught by the read: "Claude Code kept the effort Max (asked: …)" |
| Agent reasoning effort | none: no event reports a subagent's effort | always "effort N/A (not reported)" in the inspection header (plan 6, item 6) |
| Context | `get_context_usage` (undocumented at CLI level, marked †), read after each turn end, or derived from usage | if the request fails or is refused: derived, labelled "derived"; before the first turn: "N/A (no turn yet)"; while the first turn runs, before its result: "N/A (no result yet)" (plan 20, the owner's live review) |
| Tokens | latest `result.modelUsage` | "N/A" before the first `result` |
| API-equivalent cost | `result.total_cost_usd` | "N/A" before the first `result`; always labelled "API-equiv. estimate" |
| Agent count | derived from task messages | "N/A" until the stream is up, then a real count |
| Concurrency limit | not reported | "limit N/A (not reported)" |
| Duration | `result.duration_ms`; wall time measured | "N/A" before the first `result`; wall time labelled observed |
| Errors, retries | `result`, `assistant.error`, task status; `api_retry`, `subagent_retry`; a hook that fails (`system/hook_response` with an outcome other than "success" or an exit code other than 0; plan 22, amendment 20) | the counters start at 0 once the stream is up; retries "N/A (none reported)"; a hook's failure is a notice in zone B, amber like the CLI's own notes, naming the hook, its exit code and its stderr's first line, and never an error, since a SessionStart hook's lines come during the handshake, where an error ends the session while the CLI carries on; a hook that succeeds draws nothing |
| A thinking phase's live count (zone B, plan 22) | `system/thinking_tokens`, the CLI's running estimate of the open block | before the first estimate, or none for a short block: the card's bar indeterminate and "Thinking · token usage pending", no number; shown as "≈ … · live estimate", never as a count |
| A thinking phase's count (zone B, plan 22) | the message's `message_delta` `usage.output_tokens_details.thinking_tokens` | until it comes, the summary row gives the estimate marked "(estimate)", or no count; an interrupted message's 0 is no count: the row says only that it was interrupted, its details that the CLI reports 0 for an interrupted message (plan 22, correction C1); a phase whose turn ended with no count takes none later (C2) |

**Codex: ways** (TEL CX, first table)

| Way | Use | When unavailable |
| - | - | - |
| app-server notifications and requests | primary | StartFailed (see [When a whole integration fails](#when-a-whole-integration-fails)) |
| `codex exec --json` | fallback driver only | not built in the MVP |
| Rollout files | not used; last resort, labelled undocumented | nothing lost |
| OpenTelemetry | not used | nothing lost |
| `notify`, hooks | rejected: they need a configuration change | nothing lost |
| `account/usage/read` | not used: not a value of zone A's telemetry, not probed | nothing lost |

**Codex: values** (TEL CX, second table)

| Value | Source | Fallback, and what zone A shows |
| - | - | - |
| 5-hour, 7-day quota | `account/rateLimits/read` at startup, then pushes; windows by `windowDurationMins` | API key: "N/A (API key: ChatGPT sign-in needed)"; a plan with no 5-hour window: "N/A (plan has no 5-hour limit)"; a failed read: "N/A (read failed: <error>)" and the last pushed value with its age |
| Reset times | `resetsAt` | with the window |
| Plan and credits | `planType`, `credits` | API key: N/A. `rateLimitResetCredits` ids are never shown or logged |
| Model | `thread/start`, `model/rerouted`; a child's `subAgentActivity.model` (Codex CLI 0.163.0, P7b), else the `model` of the `spawnAgent` call that created it, "Model requested for the spawned agent" (`collabAgentToolCall`, 0.163.0 generated schema; P7c, on the v2 backend, sent none and its `subAgentActivity` carried the model; P9a, on the v1 backend, recorded it; source "spawn request", plan 6, item 4; plan 9), which the agent's own report always replaces | older children: "model N/A" (ASM-14) |
| Model and effort in force (plan 34) | `thread/start`'s `model` and `reasoningEffort`; `thread/settings/updated`'s `threadSettings.model` and `.effort` (experimental, plan 38); `thread/read`'s `thread.model` and `thread.reasoningEffort` after a turn that carried a change, or after `model/rerouted` | a change asked for and not yet sent: the pill keeps the values in force, the editor row's middle "→ <model> <Effort> (next turn)"; a refused turn drops the change, said in zone B; a refused read keeps the last values, its reason in zone B |
| Agent reasoning effort | a child's `subAgentActivity.reasoningEffort` (Codex CLI 0.163.0, P7b "high", P7c "low"), else the `reasoningEffort` of its `spawnAgent` call, "Reasoning effort requested for the spawned agent, when applicable" (generated schema; recorded by P9a, "low"; source "spawn request"), by the model's rule (plan 6, item 6) | older children: "effort N/A (not reported)" in the inspection header, nothing in C |
| Context | `thread/tokenUsage/updated` per thread, raw value | before a thread's first model call: "N/A (no model call yet)" (CX-10 q7) |
| Tokens | per-thread totals; session total derived | "N/A" before the first update |
| API-equivalent cost | none | always "cost N/A (Codex reports none)" |
| Agent count | derived from thread events | "N/A" before the session starts |
| Concurrency limit | `config/read` (`{"cwd": <project>, "includeLayers": false}`), once per session after `initialized`; only `config.agents.max_concurrent_threads_per_session` (fallback name `agents.max_threads`) is decoded, and `config.features.multi_agent_v2.max_concurrent_threads_per_session`, the V2 backend's own limit, which counts the main thread (Codex 0.163.0 source, `codex-rs/core/src/config/mod.rs:1666-1680`; probe 2026-10-17). The reply is the effective configuration and can hold secrets (an MCP server's `env`, a provider's bearer token), so no event keeps any byte of it: its `Raw` is a fixed placeholder. When unset the reply has no `agents` key and the built-in default is never reported (probe 2026-10-17, Codex CLI 0.163.0; owner's approval 2026-10-17) | set: "limit 3 (config)"; replied but unset: "limit N/A (not set; default not reported)"; the V2 limit minus one differs from `agents`' (or `agents` unset): "limit N/A (set per backend; backend not reported)", since the active backend is not reported; no reply yet or refused: "limit N/A (not read)" |
| Duration | `turn/completed.durationMs` | "N/A" before the first turn ends |
| Errors, retries | `error`, failed `turn/completed`; `willRetry` | counters from 0 once the stream is up; retries "N/A (none reported)" (CX-10 q9) |
| A thinking phase (zone B, plan 22) | a `reasoning` item, which starts while the model works since every `turn/start` asks for reasoning summaries (plan 22, amendment 20; P22s), and its summary when Codex sends one, at its completion; the step's `thread/tokenUsage/updated` `last.reasoningOutputTokens` after it | while the model works on a step: the card "Model working · reasoning tokens reported when the step ends", then "Thinking · …" from the reasoning item's start, its bar indeterminate, no number; no streamed summary text, the heading kept for the details; the reasoning's own duration never reported, the item's span possibly holding the summary's making: "Thought for up to …", the step's time; until the usage comes, no count; a repeated update (Codex 0.145.0 on an interrupt) gives none; a phase whose turn or session ended with no count takes none later, so a resume's replayed usage counts nothing (plan 22, correction C2); a reasoning item with no step open starts at its turn's last usage update, "up to" like a step (C5) |

**Refresh policy** (TEL "Open points"): quota and context are read on
events, plus one read at session start (Codex `account/rateLimits/read`) and
one after each turn (Claude Code `get_context_usage`, which costs no model
call: P14a, CC-10 q10). No timer polls a CLI ([events-and-state.md,
Polling](events-and-state.md#polling)).

## When a whole integration fails

If a CLI's primary way does not work (a release changes the control wire
form, the handshake fails, `app-server` is removed), the session ends at
StartFailed or Crashed with the cause, the exit code and stderr; the
dashboard does not switch to another way on its own. The fallback ways are
designed here and built only with the owner's approval, as a mode of the same
adapter, labelled on screen.

| CLI | Fallback | Keeps | Loses |
| - | - | - | - |
| Claude Code | way B: one `claude -p --resume <id>` per turn (CC-4) | streaming per turn, tools, subagent task events, quota events, cost | in-turn approvals (actions that would prompt are denied), interrupt (only ending the process), background agents beyond the turn |
| Claude Code | way C: the Agent SDK as a sidecar (CC-4) | everything of way A, through the SDK | adds Node or Python as a runtime dependency (owner's approval) |
| Claude Code | way D: an MCP permission-prompt tool served by the dashboard (CC-4) | approvals, if only `--permission-prompt-tool stdio` (**U**) breaks | nothing else; it replaces one piece |
| Codex | `codex exec --json` + `codex exec resume` (CX-4.2; P5, P5b) | the final messages, items, per-turn usage, errors | deltas, approvals (policy `never`, so writes are refused under `read-only`), quota, context, interrupt |

Either CLI may be missing entirely: the startup screen shows it "not
installed" and it cannot be chosen; with neither, Start is disabled
([ui-design.md, Startup screen](ui-design.md#startup-screen); B3.3).

## Approvals

The CLI's native approvals, relayed (AM2 B6, I7; AM4 W3; events-and-state.md,
Approval machine):

- **Never answered by the dashboard on its own**: no auto-allow, no timeout
  answer. A pending approval waits until the owner answers, the turn ends,
  or the CLI resolves it (then `Expired`).
- **A request the dashboard cannot read** is denied (Claude Code) or declined
  (Codex): the reducer answers it with the dashboard's deny, which each
  adapter writes in its CLI's word (plan 49, design D10), and shows it as
  "denied automatically: request not readable" with
  its raw line in Raw events; an unknown Codex request method gets a JSON-RPC
  error (INT 4). Refusing is the safe side: the CLI's native rules are never
  widened.
- **One answer per request**: the `Answering` state ignores a second key
  press; the answer is written once.
- **Approvals from subagents** are routed by `agent_id` (Claude Code, P3b) or
  `threadId` (Codex, P7) and shown on that agent.
- **Nothing persists**: options that write a rule ("always allow",
  `permission_suggestions`, execpolicy or network amendments) are not
  offered ([Configuration untouched](#configuration-untouched)). Codex
  `acceptForSession` lasts for the session only; it is offered unless the
  request's `availableDecisions` omits it (plan 8).
- If writing the answer fails (the pipe closed), the session is `Crashed`
  and the approval `Expired`. An answer to a request the CLI withdrew just
  before (Claude Code's `control_cancel_request`, Codex's
  `serverRequest/resolved`, the adapter then holding no pending request:
  `provider.ErrNoPending`) only expires the approval: the editor's status
  line says "the request was withdrawn before the answer reached it" and the
  session goes on, whether the withdrawal reaches the state before or after
  the failure (plan 47, design D2).
- **The modes are the CLI's own** (plan 39, brief amendment 40): the
  dashboard never changes a mode by itself; every session starts in
  Manual (Claude Code's `default`; Codex's Ask for approval, the read-only
  sandbox); a mode the owner chooses is sent through the CLI and shown in
  force only once the CLI reports it, a refusal leaving the mode as it was.
  Codex's Full access, no sandbox and no approval, is asked for only after
  the owner confirms it in the popup's confirmation, the state refusing it
  unconfirmed; a mode the managed requirements leave out takes no choice;
  the adapter writes `dangerFullAccess` only as Full access's preset, which
  a test guard checks on every line the Codex tests write.

## Interruption and cancellation

| Action | Claude Code | Codex | Edge cases (plan 14 outcome) |
| - | - | - | - |
| Interrupt the turn (Ctrl+C) | `interrupt` control request: receipt `{"still_queued":[]}`, then `result` `error_during_execution` (P2) | `turn/interrupt`, then `turn/completed` `interrupted` (P6b) | settled: during a running Bash command and with an approval pending, `error_during_execution` 3.3 s and 66 ms after (plan 4, L1; CC-10 q4); Codex children report their own status, whatever the parent's interrupt does (CX-10 q3, not depended on); Codex background terminals are not stopped by `turn/interrupt` ([T] README:1014) |
| Cancel from an approval (c) | deny the request, then interrupt | answer `cancel` | settled: `decline` completed the turn, `cancel` interrupted it 8 ms later (plan 4, L3; CX-10 q1) |
| No end 10 s after an interrupt | "the turn did not stop · F force stop (ends the session)" (ASM-16) | same | |
| Force stop | close the session (below); the turn ends `interrupted`, open tools `Unfinished` | same | |
| Quit (Ctrl+Q) | confirmation if a turn or an agent runs ("2 agents still running will be stopped"); then close | same | |
| Close | close stdin, wait 5 s, terminate, wait 5 s, kill (INT 9: no POSIX signals on Windows) | same escalation (ASM-5) | settled: Claude Code exits 0 in 1.4 s after stdin closes (P2's code 1 followed its interrupt's error) and ends 41 ms after a job kill mid-turn (CC-10 q13); Codex killed mid-turn ends in 24 ms with every process gone (CX-10 q8) |

After an interrupt the session stays `Ready` and the next message continues
it (P2: a third turn after the interrupt). Background Claude Code agents
outlive the turn that started them (P3, P3b) and stop when the session
closes; stopping one agent (`stop_task`, **D**) is not in the MVP's screens.

## Resumption

Without losing the session (B3.4):

- **Within a running session** nothing needs resuming: an interrupted or
  failed turn leaves the session `Ready`, and the conversation record keeps
  every earlier turn (ASM-11).
- **After the CLI process crashes** (`Crashed` after `Ready`; a crash
  before it is the startup's StartFailed, with no session to resume), the record stays on screen
  and readable, the draft is kept, and the crash card offers two keys: **Ctrl+R
  resume** and **Ctrl+Q quit** (a key the editor does not take, since it
  keeps the focus and the draft). Nothing restarts without the owner's key
  (events-and-state.md, Session machine). Resume starts a new process on the
  same CLI session: Claude Code `claude -p --resume <session_id>` with the
  same flags (P4); Codex a new `app-server` and `thread/resume` (CX-10 q4,
  settled live in plan 4). On success the record continues below a dashboard
  notice "reconnected: same session, earlier messages from the dashboard's
  record"; on failure the error is shown and the record stays.
- **What a resume cannot bring back**: the turn that was running (it ended
  `failed`, open tools `Unfinished`), pending approvals (`Expired`), and
  Claude Code background agents, which ended with the process (P3).
- **History is the dashboard's**: Claude Code's `--resume` does not replay
  earlier messages (P4), so the record is what shows them. Codex's
  `thread/resume` returns the turns ([T] README:321-377, not checked: the
  dashboard does not depend on it); the record is still what is shown, so
  both CLIs behave the same.
- **Across dashboard runs**: not in the MVP. The record lives in memory
  (ASM-11) and no history file is written; reopening an older session is
  not built; plan 15's recent projects keep paths only (ASM-24). The CLIs'
  own `--resume` in another terminal remains (INT 6).
- **If the dashboard itself fails** (killed, crashed or panicking): the
  draft is on disk (ASM-23) and offered back on the next start in the same
  project, its images with it, pasted ones included; a kill loses at most
  the last 500 ms of typing, an image being written as it is attached
  (plan 33); a panic restores the terminal, saves the draft and closes the CLI
  through its input before exiting with code 2; on quit, crash and panic the
  session id and the CLI's own resume command are printed. The job object
  ends the CLI when the dashboard dies (ASM-19); a turn killed that way
  leaves a resumable transcript (plan 4, L1 (g); CC-10 q13). Its session
  registration (ASM-25) is a file only an open handle keeps live: Windows
  closes the handle with the process, so the next read finds it stale and
  deletes it. A git operation of the Worktrees view the dashboard's death
  interrupts is git's to recover; the next list shows the worktree as git
  then sees it, and a merge left half done shows main with its changes, so
  no merge or removal is offered on it.

## Configuration untouched

AGENTS.md: never change an agent CLI's configuration, never read credential
files. Two layers: what the application never does, and how tests and probes
check it. The check hashes whole files and never reads what is in them (the
owner's instruction at plan 2's approval, 2026-10-16: "Never inspect, parse,
expose, or modify credential values. Use only permitted integrity-verification
methods, such as opaque file hashing where appropriate"). That instruction
supersedes the earlier aim of separating volatile runtime state from
meaningful configuration changes inside a file
([plan1-evaluation.md](research/plan1-evaluation.md), item 2;
[plan2-evaluation.md](plan2-evaluation.md)): a file that mixes the two cannot
be split without reading it, so a change there is reported, not classified.

### What the application never does

- Writes no file under `~/.claude`, `~/.codex`, `~/.claude.json`, or the
  project's `.claude\` or `.codex\`. Its own files (configuration, ASM-7) live
  in its own folder.
- Sends no request that writes configuration: no Claude Code
  `update_settings`, no `--settings` with hooks; `set_permission_mode` only
  when the owner chooses a mode (plans 12 and 39), `set_model` and
  `apply_flag_settings` only when the owner picks a model, an effort or
  Ultracode (plan 34), and none writes a settings file (P12c-switch, CC-10
  q11; P34c). No Codex `config/write`, `config/batchWrite` or
  `windowsSandbox/setupStart`; threads start read-only, and Approve for me
  asks for `workspaceWrite`, which writes no `trust_level` (plans 12 and
  13, CX-10 q2; ASM-6), and Full access, once the owner confirms it, for
  `dangerFullAccess`, each a thread's setting written between turns with
  `thread/settings/update` and carried by every `turn/start`, neither
  writing a configuration file (P39x and plan 39's live check left the
  configuration identical); a model or effort the owner picks is a
  thread's setting "for subsequent turns", written between turns with the
  experimental `thread/settings/update` (plan 38) or carried by the next
  `turn/start` ("for this turn and subsequent turns"), and neither writes
  a configuration file (P38x and P38t left `config.toml` identical; P38t's
  turn ran at the model and effort an update set).
- Offers no answer that persists a rule (see [Approvals](#approvals)).
- Writes no proxy into a CLI's configuration. Codex's proxy, when
  `config.json` names one, goes only into the environment of the Codex
  process the dashboard starts (plan 22, amendment 19).
- Opens no credential file, sets no `ANTHROPIC_BASE_URL`, and reads quota only
  from the CLI's stream (TEL). The file viewer refuses a credential file by
  its name before opening it, lists none in the quick open or the Explorer,
  opens nothing outside the project, and writes nothing; the Explorer's git
  status takes no optional lock (plan 17).
- What the CLIs write by design for each session (Claude Code transcripts,
  Codex rollouts and logs) is theirs; Codex records app-server threads with
  `source: "vscode"`, so they appear among the owner's sessions (INT 5): said
  in the key overlay's "about" page.

### The check

A test and probe tool, never part of the running application. It runs before
and after every live test and every approved probe, and its report is part of
the evidence. Three classes of file:

| Class | Files | Compared how | A difference means |
| - | - | - | - |
| Configuration | the 20 configuration files of the plan 1 check (Claude Code settings, `CLAUDE.md`, remote settings, plugin lists, skills; Codex `config.toml`, `AGENTS.md`), and the chosen project's `.claude\` and `.codex\` files | SHA-256 of the whole file | FAIL: a configuration change |
| Mixed state | `~/.claude.json`; `~/.claude/plugins/known_marketplaces.json` (both hold settings beside state the CLI rewrites itself, plan 1 criterion 9) | SHA-256 of the whole file, as for configuration; the file is opened only by the hash function, never decoded, parsed or tokenised | "changed, not classified (contents not inspected)": the run's configuration criterion is not PASS; the file is listed under Not verified with the run |
| Session data | Claude Code transcripts, Codex sessions and logs | not compared; the number of new files is reported, by name only | nothing: written by design |

Rules of the check:

- **Credential files are never opened**: any file whose name matches
  credentials, auth, oauth, token or secret is skipped, as plan 1's
  [`hash-config.sh`](research/probes/drivers/hash-config.sh) does.
- **No value is ever printed, logged or stored.** The report gives file
  names and digests only, with the user folder elided (`C:\Users\…`). The
  before snapshot stores hashes, not values.
- **Opaque hashing only.** No configuration, mixed-state or session file is
  parsed, tokenised or read key by key, by the check or by the application;
  the hash function is the only code that sees their bytes, and only the
  digest is kept.
- **The limitation, stated.** A mixed-state file differs whenever the CLI
  rewrites its own state (as on 2026-10-15), and the check cannot tell that
  from a changed setting in the same file. Such a run reports the file as
  "changed, not classified (contents not inspected)", its configuration
  criterion is not PASS, and the file goes under the run's Not verified with
  that reason. The criterion is never weakened to make it pass: there is no
  volatile list and no exception by file.
- **What narrows it, without reading contents.** The check records whether a
  CLI process ran during the window (the test started it, or another session
  was open), from the process list, not from the file. With no CLI running
  and the hash changed, the run is FAIL, not "not classified": nothing else
  is expected to write the file. The owner may also compare the file
  themselves; the check never does.
- **Whole files are never excluded** (the owner's instruction 2 at plan 1).
- **Key-level diff of `~/.claude.json` only** (the owner's approval,
  2026-10-17; this test tool only, never the application). The snapshot also
  keeps the SHA-256 of each top-level key's compacted value, and of each
  sub-key of each `projects` entry; keys whose name matches
  `oauth|token|secret|password|credential|api_key|auth|account|email|session`
  are not hashed and are reported "not compared (sensitive name)"; no value
  and no value digest is printed or stored for them. A changed key is
  **state** (Claude Code's bookkeeping: a fixed list of names, the prefixes
  `cached`, `tip(s)`, `last`, `has`, and the suffixes `Cache`, `At`; see
  `internal\confcheck\keys.go`) or **config** (every other key, including any
  unknown one). Config change: FAIL; state change: fine when a CLI ran, FAIL
  otherwise. In `projects`, the test project's entry may change in state
  sub-keys only (`allowedTools`, the MCP keys, `hasTrustDialogAccepted` and
  `ignorePatterns` are config); another project's entry changing gives
  NOT-VERIFIED naming its path. A changed file in which no compared key
  changed (a sensitive key, or formatting only) is NOT-VERIFIED, never a
  silent pass. If the file is not valid JSON in either snapshot, the
  whole-file behaviour above applies. `known_marketplaces.json` stays
  hash-only.

The earlier evidence stays as recorded: on 2026-10-15, 20 of 22 files were
identical and `~/.claude.json` and `known_marketplaces.json` differed (plan 1
criterion 9, FAIL, accepted as a documented exception); on 2026-10-16, P7's 22
hashes were identical. The new check applies to future runs only; it does not
reinterpret those results.

## JEV

Optional (B3.5): never required, never installed, never launched; the
application starts plain `claude`, never `claude-jev`, and never sets
`ANTHROPIC_BASE_URL` ([jev-pilot.md](research/jev-pilot.md), section 14).

- **Detection without reading any file**: Claude Code's `system/init` lists
  the loaded plugins (`plugins[].name`, observed in P1). If one is named
  `jev-pilot`, zone A shows "JEV: active (plugin loaded)"; otherwise nothing.
  No `claude plugin list` call, no read of `installed_plugins.json` or of
  `settings.json`'s `pluginConfigs` (a clone install keeps an API key there).
- **What is shown when present**: the badge, and the models and efforts the
  CLI's own events report. Jev's decisions and confidences are "N/A": jev-pilot
  exposes them only in its own UI.
- **Claude Code mods** as a data source (jev-pilot.md, section 14): not used
  in the MVP; loading one runs code with the owner's full permissions, so it
  would be proposed to the owner like any integration.
- Codex: JEV does not apply.

## Mock mode

A separate provider through the same pipeline (B3.6; architecture.md, Mock
provider):

- **Start**: only with `--mock <file>`. It is not on the startup screen and no
  setting turns it on. With `--mock`, detection, the two questions and CLI
  spawning are skipped (events-and-state.md, Startup machine). `--mock` with
  `--provider` or `--project` is refused at start ("--mock runs alone"), and
  `--provider` accepts only `claude-code` or `codex`: the `mock` provider id
  is reachable through `--mock` alone.
- **Input**: a JSONL file of normalised events with their timing, either
  hand-written or produced from a probe recording by the real adapter's
  mapping, so that mock screens exercise the same parts 2 to 4.
  `claude-thinking.jsonl` (P12c-block) and `claude-thinking-subagent.jsonl`
  (P303b) replay thinking phases as recorded (plan 22). Five files are
  synthetic: a probe recording with documented replies put in, for a case
  no probe has (`TestMockFiles`, internal/provider/mock/files_test.go).
  `claude-quota-windows.jsonl` is P3b with its `rate_limit_event`'s
  `unifiedWindows` holding three windows, five_hour, seven_day and
  seven_day_fable, at the values of the owner's image (plan 24).
  `codex-plan6-limits.jsonl` is P7 with plan 6's cases: each child's
  `spawnAgent` call, the first requesting a model and an effort; a long
  orchestrator answer in Markdown; a failed orchestrator command whose
  output ends with its error.
  `claude-api-key-context-refused.jsonl` is P2 with `apiKeySource`
  `ANTHROPIC_API_KEY`, no `rate_limit_event` and `get_context_usage` refused;
  `codex-api-key.jsonl` is P7 with `account/rateLimits/read` refused as
  account_processor.rs does without a ChatGPT sign-in (codex-cli.md) and no
  pushed update; `codex-no-5h-window.jsonl` is P7 with the 7-day window as
  the only one (`primary` 10,080 minutes, `secondary` null). They show what
  the mapping does with those replies, not that a live CLI sends them.
- **Labelled everywhere**: every event gets `Origin: mock`, whatever the file
  says; the title bar of every screen shows "◆ MOCK — not a live session"
  instead of "● LIVE"; inspection labels every value mock
  ([ui-design.md, Origins on screen](ui-design.md#origins-on-screen)).
- **No path to a live session**: one provider per run, chosen at start
  (AGENTS.md: no switching); the mock provider has no process, no stdin and no
  network; its "send" and "answer" append mock notices to the record and go
  nowhere. A mock run never spawns a CLI, so a mock event cannot reach one,
  and a live run never loads a mock file. Tests check each of these (item 8
  below).

## Not verified

Every limitation left open by plan 1, and how it was resolved. Until plan 14
the right-hand column named the probe or test that would settle each one (the
owner's instruction 6). Plan 14 (2026-10-18) recorded an outcome for every
row: settled, with its evidence; closed by the owner; or closed because the
dashboard does not depend on the answer. No row is left open. CC-10 and CX-10
are the numbered questions in section 10 of
[claude-code.md](research/claude-code.md) and
[codex-cli.md](research/codex-cli.md), which give the details.

**Claude Code**

| Limitation | Outcome |
| - | - |
| The models subagents actually use; the Explore override (plan 1, not verified) | settled (plan 5, item 7): a subagent's forwarded `assistant` lines carry `message.model` with its `agent_id` (P3, P3b: "claude-haiku-5-5"); zone C shows each subagent's observed model, whichever model Claude Code picks |
| The concurrency limit in effect | closed, no source reports it: shown as "N/A" (TEL), never guessed |
| Compaction: when it triggers and its events (`system/compact_boundary`, **D**) | closed, not depended on: Claude Code decides when to compact; `compact_boundary` maps to the status "compacted" (`TestCompactBoundary`), and the threshold comes from `get_context_usage` (`context.compact_at`) |
| Interrupt during a running Bash command and a pending approval (CC-10 q4) | settled (plan 4, L1 (c) and (d)): `error_during_execution` 3.3 s and 66 ms after the interrupt, nothing left running, the pending write never made |
| `AskUserQuestion` round trip (CC-10 q3) | settled (plan 14, P14a): an allow whose `updatedInput` adds `answers` gives a non-error tool result holding them; the question card sends it |
| Hook callbacks from a raw client (CC-10 q5) | closed, not depended on: `initialize` sends no hooks; subagents come from the `task_*` events |
| Two parallel subagents; volume without `--forward-subagent-text` (CC-10 q6) | settled: four parallel subagents in plan 4, two in plan 5's L1; the flag is always passed |
| Workflows under `-p` (CC-10 q7) | closed, not depended on: the dashboard starts none; a `local_workflow` task shows in zone C as agent type "workflow" |
| `/usage` over stream-json (CC-10 q9) | reopened and settled in plan 24 (amendment 23; P24u): answered locally with no model call; sent after the handshake and after each turn, its report the quota's primary source; context still from `get_context_usage` |
| Whether `get_context_usage` costs anything (CC-10 q10) | settled (P14a): answered in 914 ms with no `assistant` message, `result` or `rate_limit_event` after it |
| Whether `set_model` and `set_permission_mode` write settings (CC-10 q11) | settled for `set_permission_mode` (P12c-switch, settings identical); `set_model` closed, never sent (model configuration is outside the MVP, amendment 4), until plan 34 (amendment 34) sent it with `apply_flag_settings`: settled, neither writes a settings file (P34c, `settings.json` identical; the configuration check PASS around plan 34's live checks) |
| Windows process handling: exit code 1 after stdin closes, `TerminateProcess` mid-turn, CRLF and UTF-8 through Go pipes (CC-10 q13) | settled: stdin closed exits 0 in 1.4 s, a job kill mid-turn ends in 41 ms with exit code 1 and resumes (plan 4, L1 (e), (g)); UTF-8 and CRLF byte for byte live (P14a) and by `TestEchoUnicode` (a fake CLI through the real pipes) |
| The status line never runs under `-p` (CC-10 q14) | closed, not depended on: zone A's telemetry reads the stream |
| A raw client skipping `initialize`, and an allow without `updatedInput` (CC-10 q2, the untried part) | closed, not needed: the adapter always sends `initialize` and echoes `updatedInput` (P2, P3b); a fixture test keeps both in the adapter |
| `CLAUDE_CODE_EMIT_SESSION_STATE_EVENTS` (events-and-state.md, `turn.started`) | settled: the adapter sets it unless the environment already does (`internal/provider/claude/session.go`), and the dashboard's live recording P13c carries `system/session_state_changed` `running` then `idle` (the probe drivers, which do not set it, recorded none) |
| The `stdio` value of `--permission-prompt-tool` (**U**) keeps working | settled on each version used live, 2.1.300 (plan 4) to 2.1.303 (plans 13 and 14, the re-check of 2026-10-19): every `can_use_tool` reached the client; the recorded-fixture tests replay it |

**Codex**

| Limitation | Outcome |
| - | - |
| Command approval parameters on Windows; `decline` and `cancel` outcomes (CX-10 q1) | settled: `availableDecisions` recorded (P8, P11n); `decline` completed with the file absent, `cancel` interrupted 8 ms later (plan 4, L3) |
| The `trust_level` write with `workspace-write`; the effective Windows sandbox (CX-10 q2) | settled: no `cwd` in `thread/start`; every dashboard run on a new, untrusted project left the configuration files identical, Auto turns with `workspaceWrite` included (plans 12, 13); the reported sandbox is shown (P12x-sandbox2: `workspaceWrite`) |
| Parent interrupt stopping children; `spawnAgent` appearing (CX-10 q3) | `spawnAgent` settled (P9a, on the v1 backend); the children's stop closed, not depended on: zone C shows each child's own events |
| Resume across app-server processes (CX-10 q4) | settled: P6b's thread resumed; live in plan 4 (L3, and L4 run 3 after a `taskkill` crash) |
| `turn/steer` and a second `turn/start` mid-turn (CX-10 q5) | closed, not depended on: never sent; the state refuses sending while a turn runs (ASM-15) |
| `item/tool/requestUserInput` (CX-10 q6) | closed, not depended on: off in Default mode; if it arrives it is answered -32601 and shown as unrecognised |
| The context window before the first turn (CX-10 q7) | settled, not available before a model call: zone A's session usage reads "N/A" with "no model call yet" until `thread/tokenUsage/updated` |
| Shutdown mid-turn on Windows: orphaned shells, rollout state (CX-10 q8) | settled (plan 4, L3): a job kill mid-turn ends in 24 ms, all 9 processes gone; resume after a crash works (L4 run 3) |
| `error.willRetry` behind the corporate proxy (CX-10 q9) | closed, not depended on: read when sent, never seen; "retries N/A (none reported)" meanwhile |
| The rate-limit error text with an API key (CX-10 q10) | closed by the owner (plan 14, item 3); the fixture `codex-api-key.jsonl` covers the display |
| Whether `config/read` reports `agents.max_threads` when unset | answered, probe 2026-10-17, Codex CLI 0.163.0: unset, `config` has no `agents` key; set, the key is `agents.max_concurrent_threads_per_session` (alias `max_threads`) |
| Codex's `alternate_screen` option (TI open question) | not needed: the MVP never runs Codex's TUI (INT 6) |

**Terminals and input** (TI, "Open questions")

| Limitation | Outcome |
| - | - |
| WT 1.25 Kitty keyboard protocol through ConPTY | not needed: Shift+Enter works through win32-input-mode |
| WT honouring a win32-input-mode (9001) request from the application | settled 2026-10-16: honoured by WT 1.23, measured ([ui-design.md, Focus and keys](ui-design.md#focus-and-keys)) |
| `trimPaste` and its effect on byte-for-byte paste | closed, not depended on: a Windows Terminal setting applied before the bytes reach the dashboard, which keeps what it receives byte for byte (the editor's paste tests); the owner closed the checks by hand (plan 14, item 3) |
| tmux: the effective `extended-keys` default; re-wrapping outer pastes | closed, not depended on: tmux is not used (AGENTS.md); OSC 52's tmux wrapping stays tested |
| OSC 52 size limits beyond kitty and WT; block selection per terminal | closed, not depended on: the native clipboard is the primary path (x/clipboard), OSC 52 a second copy; Windows Terminal is the terminal the application targets |
| Which console reader path Bubble Tea v2 takes on Windows | settled by use: every live run in Windows Terminal read the keys, Shift+Enter and Ctrl+C included, through win32-input-mode |
| Whether the Windows console reader delivers every key | settled 2026-10-29, plan 68: ultraviolet's reader wrote a console read's next batch into the bytes it had just sent, before the receiver had copied them, so a key could be lost and the next one doubled (plan 67's lost Ctrl+F), and its two goroutines shared the grapheme state; the patched copy sends a copy and keeps the sender's state its own (`third_party/ultraviolet/PATCHES.md`, patches 4 and 5), tested on the real console path in a console of the test's own |
| Whether Bubble Tea v2 wraps frames in synchronized output (mode 2026) | settled 2026-10-17: yes once the terminal reports it supported, as WT 1.23 does; measured, and tested offline with WT's recorded answer ([ui-design.md, Rendering](ui-design.md#rendering)) |
| Lip Gloss v2 colour downsampling on 256- and 16-colour terminals | settled 2026-10-17 (plan 5, item 13): measured; an explicit 16-colour palette replaces the downsampling, which merged roles ([ui-design.md, Colours and symbols](ui-design.md#colours-and-symbols)) |
| Notepad as `$EDITOR`: does it block | gone (2026-10-19): no external editor, the draft opens in a window in the terminal (brief amendment 13). Before that, closed, not depended on: whichever way Notepad behaves, an editor that returns at once is detected and the status line says "the editor returned immediately: add --wait (or your editor's blocking flag)", the draft kept (the editing tests); the owner closed the checks by hand (plan 14, item 3) |
| `EDITOR`/`VISUAL` usually unset on Windows; Git for Windows' default editor | gone (2026-10-19): no external editor (brief amendment 13). Before that, settled on this machine (2026-10-18): `VISUAL`, `EDITOR` and Git's `core.editor` are all unset; the dashboard then uses a detected editor and names it ("no $VISUAL or $EDITOR: using code --wait", `TestEditorCommand`) |
| atotto's WSL clipboard path speed; `clip.exe` code page | closed, not used (ASM-2) |

**Platforms and paths**

| Limitation | Outcome |
| - | - |
| A UNC working directory (`\\server\share`, `\\wsl$\…`) for a native CLI | closed, not depended on: the startup screen warns ("a network share can be slow or disappear", `internal/input/path.go`, tested) and leaves the choice to the owner |
| Piping through `wsl.exe` to a CLI inside WSL | closed, not depended on: outside the MVP, which runs the Windows CLIs |
| Linux and macOS builds | closed by the owner (plan 14, item 3): the application targets Windows 11 |
| External URLs cited in plan 1 (not re-fetched) | closed: no open decision depends on one; a later decision re-fetches its source |

**This design**

| Limitation | Outcome |
| - | - |
| Glamour's cost when re-rendering a long streamed message (ASM-17) | closed, not used: Glamour is not a dependency; the cost of a long streamed session is measured by `TestLongSession` in the timing step |
| Memory over a long session (unbounded queue and record) | settled by test 11 (`TestLongSession`) |
| Whether a run changed a setting inside `~/.claude.json` or `known_marketplaces.json`, when the CLI also rewrote its own state there | settled for `~/.claude.json` by the key-level diff the owner approved (plan 5, F4; see [The check](#the-check)): the runs of plans 13 and 14 changed only Claude Code's cache-state keys; `known_marketplaces.json` stays hash-only, and a change there makes a run NOT-VERIFIED, never PASS: it was identical in those runs |
| The `plugins` field of `system/init` (JEV detection): observed in P1, its documentation not checked | settled by the recordings: every `system/init` in the nine Claude Code recordings, P1 to P14a, carries it; the adapter reads its names (`Started.Plugins`) |

## Test plan

Recorded fixtures and deterministic tests first; a live run only when a
fixture cannot settle the question, and with approval (the owner's
instruction 5). Every test runs without a CLI, a network or a model, except
those marked live.

1. **Event normalisation per CLI** (B3.2; INT shape). The probe recordings P1
   to P4 and P3b (Claude Code) and P5 to P7 (Codex) are the fixtures: each
   adapter's `map(line)` is run over every line, and golden files fix the
   expected events. Checks include, from the recordings: P2 gives 25
   `content_block_delta` lines (3 of them `text_delta`) and an interrupt ending `error_during_execution`; P3b gives the
   subagent's approval with its `agent_id` and no `unrecognised`
   (`commands_changed` is classified); P7 gives 11 rate-limit pushes, 10
   `thread/status/changed`, two children routed by `threadId`, and a
   child's approval. Every line maps to at least one event: none is dropped.
   Synthetic fixtures, labelled synthetic, cover what no probe produced
   (`agent.failed`, `retry`, a Codex command approval, an API-key rate-limit
   error), shaped from the documentation each cites.
2. **Telemetry parsing per CLI**: P1 line 3 gives 22% and 45% with their
   reset times (Claude Code, marked undocumented); P1 line 12 gives
   `total_cost_usd` 0.00249374 as an estimate; P6b line 7 gives both Codex
   windows identified by `windowDurationMins`; the P7 token totals per
   thread (76,521; 30,455; 77,336) and their derived sum. Each N/A case of
   [Telemetry sources](#telemetry-sources) has a test that shows "N/A" with
   its reason, never 0; a window past its reset turns N/A.
3. **State machines** (events-and-state.md, State machines): a table-driven
   test per machine with one case per listed transition, run with each
   provider's events where the transition holds for it; one case per invalid
   transition, checking the documented effect (refused with a notice,
   logged, ignored). The startup machine is tested with a fake detector and a
   fake file system for every validation message of
   [ui-design.md](ui-design.md#validation-and-messages).
4. **The reducer replays every fixture end to end**, and the resulting state
   is compared with a golden summary (turns, agents and their statuses,
   approvals, telemetry with origins). Replaying twice gives the same state.
5. **Input, byte for byte** (AM3; TI 3): a `PasteMsg` with LF, CRLF, lone CR,
   tabs, trailing spaces, combining marks, emoji, wide characters and C0
   controls is inserted; the stored text, then the string handed to the
   provider's send, are compared byte for byte with the expected text (CR
   stored as LF, everything else unchanged). A paste containing CR, LF,
   `\r\n` or an Enter-like sequence never submits; Enter after it submits
   once. A paste with no `PasteEndMsg` leaves the editor in paste state.
6. **Input at scale** (ASM-10): a benchmark on this machine: a keystroke in a
   10,000-line buffer is applied and the frame drawn in at most 16 ms (95th
   percentile); a 50,000-line paste is inserted and the next frame drawn in
   at most 100 ms. Both figures are targets (ASM-10) until this benchmark
   measures them; a result is reported with the measured value, never as met
   before it is measured. The same tests run with the Claude Code, Codex and mock
   providers attached: the results must not differ, since the editor has no
   provider code.
7. **Separation** (B3.1): a test parses the module's sources with
   `go/parser` (`internal/arch/separation_test.go`; non-test files, whatever
   their build tags) and follows each package's imports through the
   module. It fails if rendering or the editor reaches an adapter, process
   or application package; if the reducer reaches one of those or
   rendering; if any package the reducer reaches imports an I/O, process or
   network package (`os`, `io`, `bufio`, `net`, `syscall`, `path/filepath`,
   `golang.org/x/sys` and the like); or if it calls the clock (`time.Now`,
   `time.Since`, `time.After`, timers and the like; `time`'s types are
   allowed). Planted faults show it fails (TestSeparationCatchesPlantedFaults;
   plan 21, item 5.5).
8. **Mock isolation**: a mock run spawns no process (checked with a fake
   spawner that fails the test if called); every event in a mock run has
   origin `mock` even when the file says otherwise; every golden frame of a
   mock session contains "◆ MOCK — not a live session", the file viewer and
   the draft window at full screen included, and TestGoldenFrames fails one
   that does not (plan 21, item 5.7); `--mock` combined with
   `--provider` or `--project`, and `--provider mock`, are refused at start.
9. **Rendering**: golden frames at 80x24, 120x40 and 200x60 from mock
   fixtures, for the startup screen, the dashboard and inspection; "terminal
   too small" below 80x24; every N/A shows its reason; every undocumented
   value carries †; no line exceeds the width. A frame-count test checks
   that an idle dashboard asks for no spinner and no animation frame, only
   the 1 s clock for the ages it shows (ASM-18; `TestIdleDrawsNothing`,
   plan 22).
10. **Faults, with a fake CLI** (a test binary replaying a fixture): a
    truncated line, invalid JSON, an unknown type, a 10 MiB line, a 65 MiB
    line (ASM-20), stderr flooding, no handshake (the 60 s timeout shortened
    for the test), exit in the middle of a turn, exit with a pending
    approval, a closed stdin when sending. Each ends in the documented state
    with a visible message; none hangs the loop.
11. **Long session**: a fixture replaying 2 hours of events at speed; memory
    is reported, and the test fails if the frame time exceeds 16 ms at the
    end. `TestLongSession`, in the timing step (evaluation.md).
    Generated, not replayed: no probe lasts two hours.
12. **Clipboard and selection**: OSC 52 encoding and its tmux passthrough
    wrapping; the native path behind an interface with a fake; selection
    mapped to the content model (a wrapped line copies as one line; origin
    labels, symbols and borders are not copied; a code block copies its exact
    text, tabs included).
13. **Configuration check** ([Configuration untouched](#configuration-untouched)):
    unit tests of the check itself on fake home folders: a changed
    configuration file fails; a changed mixed-state file with a CLI running
    is reported "changed, not classified (contents not inspected)" and the
    criterion is not PASS; the same change with no CLI running fails; a
    credential-named file is never opened (the test gives it no read
    permission); the check's code calls no JSON, TOML or tokenising function
    on any file (a test over its imports); no value appears in the report.
14. **Manual checks** (no model call): the "What works where" matrix of
    [ui-design.md](ui-design.md#what-works-where), the terminal rows of
    [Not verified](#not-verified).
15. **Live smoke tests** (quota; with approval; the configuration check
    before and after): one short Claude Code session and one short Codex
    session from the dashboard, each with one turn, one approval and one
    interrupt; run once per new CLI version, not on every change.

## Assumptions used here

ASM-2, ASM-3, ASM-5, ASM-6, ASM-7, ASM-10, ASM-11, ASM-14, ASM-15, ASM-16,
ASM-17, ASM-18, and three new ones: ASM-19 (a Windows job object per CLI process), ASM-20
(the 64 MiB line ceiling) and ASM-23 (the draft kept on disk, added in
plan 3); all are listed in
[architecture.md, Assumptions](architecture.md#assumptions).
