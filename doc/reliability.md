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
| 2. Event collection | a line that does not decode: becomes an `unrecognised` event with the parse error, counted in D | the state: it never applies anything itself |
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
orphaned if the dashboard is killed (ASM-19). Whether Codex leaves shell
processes behind when its app-server is killed mid-turn is CX-10 q8: Not
verified.

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
| `unrecognised` | never absent: anything unknown lands here | counted in D, listed in Raw events |

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
| 3. Subagents and tools | task messages and `agent_id`; fallback hooks through `initialize` (CC-10 q5, Not verified) or transcript files (undocumented, labelled) | `subAgentActivity` and per-thread events; fallback `thread/list` + `thread/read`, then inferred agents |
| 4. Approvals | `can_use_tool` (`stdio` is **U**); fallback way D, an MCP permission tool; an unreadable request is denied and shown | `requestApproval` requests; an unreadable request is declined, an unknown method gets a JSON-RPC error, both shown |
| 5. Persistence, cancel, resume | `interrupt`, `--resume`; fallback transcript file (labelled undocumented), then close stdin and end the process | `turn/interrupt`, `thread/resume`; fallback `thread/read`, `codex exec resume`, rollout file (labelled undocumented) |
| 6. Reusing the TUI | not reused; escape hatch `claude --resume <id>` in another terminal (outside the MVP) | not reused; escape hatch `codex resume <id>` (outside the MVP) |
| 7. Concurrent output | route by `agent_id`, `task_id`, `parent_tool_use_id`; fallback "unattributed" on the orchestrator | route by `threadId`; fallback `optOutNotificationMethods`, per-thread `thread/read` |
| 8. Quota | `rate_limit_event`; fallback last value with its age, "N/A" before the first | `account/rateLimits/read` and pushes; fallback "N/A" with the reason, last pushed value with its age |
| 9. Platform limits | Windows native, Git Bash for the Bash tool; fallback its PowerShell tool, the reported shell shown | Windows native; the effective sandbox from `thread/start` shown, so a downgrade is visible |
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
  execution). Memory over a long session is Not verified; the test plan
  replays a long fixture to measure it.
- Codex can refuse a request with `-32001` when overloaded (INT 1): the
  command fails visibly ("Codex busy: request refused, try again"), the draft
  is restored, and nothing is retried automatically.

## Telemetry sources

Every source of [TEL](research/telemetry-sources.md), with its fallback (AM2
D1-D9; B3.3). Values follow TEL's rules: N/A until a source gives the value,
every pushed value with its age, cost apart from quota, a window past its
reset shown N/A.

**Claude Code: ways** (TEL CC, first table)

| Way | Use | When unavailable |
| - | - | - |
| stream-json messages and control responses | primary | the session cannot start: StartFailed (see [When a whole integration fails](#when-a-whole-integration-fails)) |
| `/usage` sent as a prompt | not used in the MVP: unprobed over stream-json (CC-10 q9), and it could cost a model call | nothing lost: `rate_limit_event` is the quota source |
| Status line JSON | not available to a `-p` session (CC-6) | nothing lost; its field names guide D |
| OpenTelemetry | not used: no quota metric, duplicates the stream | nothing lost |
| Hooks | not used for telemetry | nothing lost |
| Transcript files | not used for telemetry; last resort only, labelled undocumented | nothing lost |

**Claude Code: values** (TEL CC, second table)

| Value | Source | Fallback, and what D shows |
| - | - | - |
| 5-hour, 7-day quota | `rate_limit_event` (`unifiedWindows` **U**, marked †) | before the first event "N/A (no report yet)"; with an API key "N/A (API key: no subscription window)"; past the reset "N/A (window reset)"; without `unifiedWindows` but with `rateLimitType` and `status`: the status word ("allowed", "warning", "rejected") with no percentage |
| Reset times | `resetsAt` | with the window |
| Overage | `overageStatus` and related | "N/A (not reported)", shown apart from quota |
| Model | `system/init`, per message | a subagent whose text is not forwarded: "model N/A" |
| Context | `get_context_usage` (undocumented at CLI level, marked †) or derived from usage | if the request fails or is refused: derived, labelled "derived"; before the first turn without a reply: "N/A (no turn yet)" |
| Tokens | latest `result.modelUsage` | "N/A" before the first `result` |
| API-equivalent cost | `result.total_cost_usd` | "N/A" before the first `result`; always labelled "API-equiv. estimate" |
| Agent count | derived from task messages | "N/A" until the stream is up, then a real count |
| Concurrency limit | not reported | "limit N/A (not reported)" |
| Duration | `result.duration_ms`; wall time measured | "N/A" before the first `result`; wall time labelled observed |
| Errors, retries | `result`, `assistant.error`, task status; `api_retry`, `subagent_retry` | the counters start at 0 once the stream is up; retries "N/A (none reported)" |

**Codex: ways** (TEL CX, first table)

| Way | Use | When unavailable |
| - | - | - |
| app-server notifications and requests | primary | StartFailed (see [When a whole integration fails](#when-a-whole-integration-fails)) |
| `codex exec --json` | fallback driver only | not built in the MVP |
| Rollout files | not used; last resort, labelled undocumented | nothing lost |
| OpenTelemetry | not used | nothing lost |
| `notify`, hooks | rejected: they need a configuration change | nothing lost |
| `account/usage/read` | not used: not a zone D value, not probed | nothing lost |

**Codex: values** (TEL CX, second table)

| Value | Source | Fallback, and what D shows |
| - | - | - |
| 5-hour, 7-day quota | `account/rateLimits/read` at startup, then pushes; windows by `windowDurationMins` | API key: "N/A (API key: ChatGPT sign-in needed)"; a plan with no 5-hour window: "N/A (plan has no 5-hour limit)"; a failed read: "N/A (read failed: <error>)" and the last pushed value with its age |
| Reset times | `resetsAt` | with the window |
| Plan and credits | `planType`, `credits` | API key: N/A. `rateLimitResetCredits` ids are never shown or logged |
| Model | `thread/start`, `model/rerouted` | children: "model N/A" (ASM-14) |
| Context | `thread/tokenUsage/updated` per thread, raw value | before a thread's first model call: "N/A (no model call yet)" (CX-10 q7) |
| Tokens | per-thread totals; session total derived | "N/A" before the first update |
| API-equivalent cost | none | always "cost N/A (Codex reports none)" |
| Agent count | derived from thread events | "N/A" before the session starts |
| Concurrency limit | `agents.max_threads` from `config/read` when set | "limit N/A (agents.max_threads not set)"; whether `config/read` reports it when unset is Not verified |
| Duration | `turn/completed.durationMs` | "N/A" before the first turn ends |
| Errors, retries | `error`, failed `turn/completed`; `willRetry` | counters from 0 once the stream is up; retries "N/A (none reported)" (CX-10 q9) |

**Refresh policy** (TEL "Open points"): quota and context are read on
events, plus one read at session start (Codex `account/rateLimits/read`) and
one after each turn (Claude Code `get_context_usage`, whose cost is CC-10
q10, Not verified; if it turns out to cost a model call, the derived value is
used instead). No timer polls a CLI ([events-and-state.md,
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
  (Codex), and shown as "denied automatically: request not readable" with
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
  `acceptForSession` lasts for the session only.
- If writing the answer fails (the pipe closed), the session is `Crashed`
  and the approval `Expired`.

## Interruption and cancellation

| Action | Claude Code | Codex | Not verified |
| - | - | - | - |
| Interrupt the turn (Ctrl+C) | `interrupt` control request: receipt `{"still_queued":[]}`, then `result` `error_during_execution` (P2) | `turn/interrupt`, then `turn/completed` `interrupted` (P6b) | interrupt during a running Bash command or a pending approval (CC-10 q4); whether the parent's interrupt stops Codex children (CX-10 q3); Codex background terminals are not stopped by `turn/interrupt` ([T] README:1014) |
| Cancel from an approval (c) | deny the request, then interrupt | answer `cancel` | the outcome of `decline` and `cancel` (CX-10 q1) |
| No end 10 s after an interrupt | "the turn did not stop · F force stop (ends the session)" (ASM-16) | same | |
| Force stop | close the session (below); the turn ends `interrupted`, open tools `Unfinished` | same | |
| Quit (Ctrl+Q) | confirmation if a turn or an agent runs ("2 agents still running will be stopped"); then close | same | |
| Close | close stdin, wait 5 s, terminate, wait 5 s, kill (INT 9: no POSIX signals on Windows) | same escalation (ASM-5) | P2 exited with code 1 after stdin closed (CC-10 q13); Codex mid-turn shutdown (CX-10 q8) |

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
  Not verified). On success the record continues below a dashboard notice
  "reconnected: same session, earlier messages from the dashboard's record";
  on failure the error is shown and the record stays.
- **What a resume cannot bring back**: the turn that was running (it ended
  `failed`, open tools `Unfinished`), pending approvals (`Expired`), and
  Claude Code background agents, which ended with the process (P3).
- **History is the dashboard's**: Claude Code's `--resume` does not replay
  earlier messages (P4), so the record is what shows them. Codex's
  `thread/resume` returns the turns ([T] README:321-377, not probed); the
  record is still what is shown, so both CLIs behave the same.
- **Across dashboard runs**: not in the MVP. The record lives in memory
  (ASM-11) and no history file is written; reopening an older session would
  be a "recent projects" style feature, which AM4 excludes. The CLIs' own
  `--resume` in another terminal remains (INT 6).
- **If the dashboard itself fails** (killed, crashed or panicking): the
  draft is on disk (ASM-23) and offered back on the next start in the same
  project; a panic restores the terminal, saves the draft and closes the CLI
  through its input before exiting with code 2; on quit, crash and panic the
  session id and the CLI's own resume command are printed. The job object
  ends the CLI when the dashboard dies (ASM-19); whether a turn killed that
  way leaves a resumable transcript is checked live in plan 4 (CC-10 q13).

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
- Sends no request that writes configuration: no Claude Code `set_model` or
  `set_permission_mode` (whether they write is CC-10 q11, Not verified), no
  `--settings` with hooks; no Codex `config/write`, `config/batchWrite`, or
  `windowsSandbox/setupStart`; no `workspace-write` thread until CX-10 q2
  shows it does not write `trust_level` (ASM-6).
- Offers no answer that persists a rule (see [Approvals](#approvals)).
- Opens no credential file, sets no `ANTHROPIC_BASE_URL`, and reads quota only
  from the CLI's stream (TEL).
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
  `jev-pilot`, D shows "JEV: active (plugin loaded)"; otherwise nothing. No
  `claude plugin list` call, no read of `installed_plugins.json` or of
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

Every limitation still open from plan 1, with the probe or test that would
settle it (the owner's instruction 6). "Probe" means a live run that uses the
owner's quota and needs approval under [evaluation.md](evaluation.md);
"manual" means a check by hand in plan 4, with no model call.

**Claude Code**

| Limitation | Settles it |
| - | - |
| The models subagents actually use; the Explore override (plan 1, not verified) | probe: one turn spawning an Explore subagent, read `message.model` of its forwarded messages |
| The concurrency limit in effect | nothing reports it; it stays "N/A" (TEL) |
| Compaction: when it triggers and its events (`system/compact_boundary`, **D**) | probe: a long session, or recorded from a real one |
| Interrupt during a running Bash command and a pending approval (CC-10 q4) | probe: interrupt in both situations |
| `AskUserQuestion` round trip (CC-10 q3) | probe: a prompt asking the model to ask a question |
| Hook callbacks from a raw client (CC-10 q5) | probe; needed only for the INT 3 fallback |
| Two parallel subagents; volume without `--forward-subagent-text` (CC-10 q6) | probe: a turn spawning two subagents |
| Workflows under `-p` (CC-10 q7) | probe; the MVP shows a `local_workflow` task as one node meanwhile |
| `/usage` over stream-json (CC-10 q9) | probe; not used in the MVP meanwhile |
| Whether `get_context_usage` costs anything (CC-10 q10) | probe: compare `rate_limit_event` and usage before and after a run of requests |
| Whether `set_model` and `set_permission_mode` write settings (CC-10 q11) | probe with the configuration check; not sent meanwhile |
| Windows process handling: exit code 1 after stdin closes, `TerminateProcess` mid-turn, CRLF and UTF-8 through Go pipes (CC-10 q13) | test with a fake CLI for the pipes; probe for the real exit behaviour |
| The status line never runs under `-p` (CC-10 q14) | probe; nothing depends on it |
| A raw client skipping `initialize`, and an allow without `updatedInput` (CC-10 q2, the untried part) | not needed: the adapter always sends `initialize` and echoes `updatedInput` (P2, P3b); a fixture test keeps both in the adapter |
| `CLAUDE_CODE_EMIT_SESSION_STATE_EVENTS` (events-and-state.md, `turn.started`) | probe; `turn.started` stays inferred meanwhile |
| The `stdio` value of `--permission-prompt-tool` (**U**) keeps working | the recorded-fixture tests on each new CLI version, plus one live smoke test |

**Codex**

| Limitation | Settles it |
| - | - |
| Command approval parameters on Windows; `decline` and `cancel` outcomes (CX-10 q1) | probe: a command needing approval under `read-only` |
| The `trust_level` write with `workspace-write`; the effective Windows sandbox (CX-10 q2) | probe with the configuration check; `read-only` meanwhile (ASM-6) |
| Parent interrupt stopping children; `spawnAgent` appearing (CX-10 q3) | probe: interrupt during P7's scenario |
| Resume across app-server processes (CX-10 q4) | probe: `thread/resume` from a new process |
| `turn/steer` and a second `turn/start` mid-turn (CX-10 q5) | probe; the MVP refuses sending while a turn runs (ASM-15) |
| `item/tool/requestUserInput` (CX-10 q6) | probe in Plan mode; questions are plain text meanwhile |
| The context window before the first turn (CX-10 q7) | probe of `config/read` and `model/list` (no model call) |
| Shutdown mid-turn on Windows: orphaned shells, rollout state (CX-10 q8) | probe |
| `error.willRetry` behind the corporate proxy (CX-10 q9) | observed when it happens; "N/A" meanwhile |
| The rate-limit error text with an API key (CX-10 q10) | needs an API-key login; likely never tested; a synthetic fixture covers the display |
| Whether `config/read` reports `agents.max_threads` when unset | probe: `config/read` (no model call) |
| Codex's `alternate_screen` option (TI open question) | not needed: the MVP never runs Codex's TUI (INT 6); manual if the escape hatch `codex resume` is ever offered inside the dashboard |

**Terminals and input** (TI, "Open questions")

| Limitation | Settles it |
| - | - |
| WT 1.25 Kitty keyboard protocol through ConPTY (Shift+Enter) | manual in WT; Ctrl+J works meanwhile |
| WT honouring a win32-input-mode (9001) request from the application | manual |
| `trimPaste` and its effect on byte-for-byte paste | manual: paste trailing spaces and tabs in WT |
| conhost: bracketed paste, OSC 52, the QuickEdit mouse bypass, by Windows build | manual in conhost |
| VS Code terminal: Kitty protocol, Ctrl+Enter, F1, large pastes | manual |
| tmux: the effective `extended-keys` default; re-wrapping outer pastes | manual under WSL |
| OSC 52 size limits beyond kitty and WT; block selection per terminal | manual where available |
| Which console reader path Bubble Tea v2 takes on Windows | read the code at the version pinned in plan 4, then manual |
| Whether Bubble Tea v2 wraps frames in synchronized output (mode 2026) | read the renderer at the pinned version |
| Lip Gloss v2 colour downsampling on 256- and 16-colour terminals | manual with `TERM`/`COLORTERM` variations |
| Notepad as `$EDITOR`: does it block | manual |
| `EDITOR`/`VISUAL` usually unset on Windows; Git for Windows' default editor | manual on this machine |
| atotto's WSL clipboard path speed; `clip.exe` code page | not used (ASM-2); manual if the WSL build is tried |

**Platforms and paths**

| Limitation | Settles it |
| - | - |
| A UNC working directory (`\\server\share`, `\\wsl$\…`) for a native CLI | probe, one per CLI; warned, not blocked, meanwhile |
| Piping through `wsl.exe` to a CLI inside WSL | probe; outside the MVP meanwhile |
| Linux and macOS builds | build and run on those systems; the MVP targets Windows 11 |
| External URLs cited in plan 1 (not re-fetched) | re-fetch when a decision depends on one |

**This design**

| Limitation | Settles it |
| - | - |
| Glamour's cost when re-rendering a long streamed message (ASM-17) | benchmark in plan 4: render a 2,000-line message, 1 delta per frame |
| Memory over a long session (unbounded queue and record) | test 11 below |
| Whether a run changed a setting inside `~/.claude.json` or `known_marketplaces.json`, when the CLI also rewrote its own state there | not settled by any permitted check (opaque hashing only); each such run lists the file as "changed, not classified (contents not inspected)" |
| The `plugins` field of `system/init` (JEV detection): observed in P1, its documentation not checked | read the SDK reference in plan 4 |

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
   subagent's approval with its `agent_id` and one `unrecognised`
   (`commands_changed`); P7 gives 11 rate-limit pushes, 10
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
7. **Separation** (B3.1): a test reads the package graph (`go list -deps`)
   and fails if rendering imports an adapter or process package, if the
   editor imports a provider package, or if the reducer imports an I/O,
   process, network or clock package.
8. **Mock isolation**: a mock run spawns no process (checked with a fake
   spawner that fails the test if called); every event in a mock run has
   origin `mock` even when the file says otherwise; every screen's golden
   frame contains "◆ MOCK — not a live session"; `--mock` combined with
   `--provider` or `--project`, and `--provider mock`, are refused at start.
9. **Rendering**: golden frames at 80x24, 120x40 and 200x60 from mock
   fixtures, for the startup screen, the dashboard and inspection; "terminal
   too small" below 80x24; every N/A shows its reason; every undocumented
   value carries †; no line exceeds the width. A frame-count test checks
   that an idle dashboard draws nothing (ASM-18).
10. **Faults, with a fake CLI** (a test binary replaying a fixture): a
    truncated line, invalid JSON, an unknown type, a 10 MiB line, a 65 MiB
    line (ASM-20), stderr flooding, no handshake (the 60 s timeout shortened
    for the test), exit in the middle of a turn, exit with a pending
    approval, a closed stdin when sending. Each ends in the documented state
    with a visible message; none hangs the loop.
11. **Long session**: a fixture replaying 2 hours of events at speed; memory
    is reported, and the test fails if the frame time exceeds 16 ms at the
    end.
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
