# Integration: the amendment 2 questions and the startup paths

Plan 1, step 2. For each agent CLI in scope (Claude Code 2.1.300, Codex CLI
0.145.0; Gemini CLI is out of scope since amendment 5), the 9 research
questions of amendment 2 (`doc/brief.md`), each answered with its sources, its
limitations and a realistic fallback. Then the path forms the startup screen
accepts and validates (amendment 4, question 2).

The detail behind each answer is in the two CLI files; this file gives the
answer, the decisive evidence and what plan 2 must design around.

## Conventions

- **CC-n** is section n of [`claude-code.md`](claude-code.md); **CX-n** is
  section n of [`codex-cli.md`](codex-cli.md). Their own source keys are kept
  as they are: for Claude Code doc `page#section`
  (`https://code.claude.com/docs/en/page`, accessed 2026-10-15), **PY** (Python
  Agent SDK at `c10b52c674e5c709c7263544b38e8a5dad10e01d`), **CC** (the
  `anthropics/claude-code` repository at
  `94506ff0d3bd9185f1752da36e968775021fb48f`); for Codex **[T]**
  (`openai/codex` at `rust-v0.145.0`, commit
  `25af12f7e61572b0bc18ddb1008be543b91519b0`) and **[D:page]** (docs accessed
  2026-10-15, URLs in CX-1).
- **Pn** is probe n, recorded in [`probes/`](probes/README.md): P1 to P4 and P3b
  for Claude Code (2026-10-15), P5, P5b, P6, P6b for Codex (2026-10-15), P7 for
  Codex (2026-10-16). File names: `p<n>-claude-*.jsonl`, `p<n>-codex-*.jsonl`.
- Path sources, all accessed 2026-10-16:
  - **[MS:paths]** "File path formats on Windows systems",
    https://learn.microsoft.com/en-us/dotnet/standard/io/file-path-formats,
    source commit `8a189e07f74e50fa0e49551857c3b88d69a7630b`.
  - **[MS:wslfs]** "Working across Windows and Linux file systems",
    https://learn.microsoft.com/en-us/windows/wsl/filesystems, source commit
    `7b28cc1ee9b8ff672ada5e1c6c326d3573d703e5`.
  - **[MS:wslconf]** "Advanced settings configuration in WSL",
    https://learn.microsoft.com/en-us/windows/wsl/wsl-config, source commit
    `7ea1c6f9e25f1c89a05a0e97e5325a02a66ac6cd` (page updated 2026-09-16).
  - **[Go:filepath]** https://pkg.go.dev/path/filepath and **[Go:os]**
    https://pkg.go.dev/os, Go 1.27.2.
- **Documented**, **undocumented** and **inferred** as in CC and CX:
  inferred means drawn from documented facts, not stated anywhere and not
  observed.

## Summary

| # | Question | Claude Code | Codex CLI |
| - | - | - | - |
| 1 | Embed or control a session | Yes: `claude -p` stream-json in and out, stdio control protocol; one long-lived process | Yes: `codex app-server`, JSON-RPC over stdio; one long-lived process |
| 2 | Genuine streaming | Token deltas for the orchestrator; subagents as complete messages | Token deltas for every thread, subagents included |
| 3 | Subagent lifecycle and tool events | `system/task_*` and `background_tasks_changed`, tool events keyed by `parent_tool_use_id` / `agent_id` | Each subagent is a thread: `subAgentActivity` on the parent, then the child's own events by `threadId` |
| 4 | Approvals | `can_use_tool` control requests, answered allow or deny | Server requests `item/*/requestApproval`, answered `accept`, `decline`, `cancel` |
| 5 | Persistence, cancellation, resumption | CLI persists; `interrupt` request; `--resume` without history replay | CLI persists; `turn/interrupt`; `thread/resume` with history |
| 6 | Reuse the existing terminal UI | No; escape hatch only | No; escape hatch only |
| 7 | Concurrent subagent output | One ordered stream, routed by ids | One ordered stream, routed by `threadId` |
| 8 | Reliable quota | `rate_limit_event` (on transitions) and `/usage` on demand; N/A until one arrives | `account/rateLimits/read` any time, pushes after model calls; ChatGPT sign-in only |
| 9 | Windows, WSL2, Linux, macOS | Windows native works, no sandbox, no signals | Windows native works; its own Windows sandbox; WSL2 needs Linux paths |

Both CLIs can be driven properly through supported interfaces, from a client
written without their SDKs (P2, P6b). The two protocols differ in shape but
carry the same concepts, so one provider interface covers both (section
"Common provider shape").

## 1. Embedding or controlling a session

**Claude Code.** Run `claude -p --input-format stream-json --output-format
stream-json --verbose --include-partial-messages --permission-prompt-tool
stdio` as one long-lived child process in the project directory; send user
messages and control requests on stdin, read NDJSON on stdout. This is how
both Agent SDKs and the VS Code extension drive it.
- Sources: doc `headless`; PY `subprocess_cli.py:570-795`; CC
  `CHANGELOG.md:1650`; CC-4, way A; P2 (a raw Node client without the SDK, three
  turns: `initialize`, user messages, `interrupt`, `get_context_usage`).
- Limitations: the message stream is documented, but the wire form of the
  control requests is only in SDK code (undocumented at CLI level) and can
  change between releases. The host must keep stdin open while background
  tasks run, or they are lost (P3). On Windows, never spawn `claude.cmd` with
  arguments (CC-2; PY `subprocess_cli.py:415-448`).
- Fallback: one `claude -p --resume <id>` process per turn (CC-4, way B), losing
  in-turn approvals and interrupt; or the Agent SDK as a sidecar process
  (CC-4, way C), which adds Node or Python as a runtime dependency.

**Codex.** Run `codex app-server` as one long-lived child process; JSON-RPC 2.0
(without the `"jsonrpc"` field) as JSONL on stdin and stdout: `initialize`,
`initialized`, `thread/start`, `turn/start`, `turn/interrupt`.
- Sources: [D:app-server]; [T] `codex-rs/app-server/README.md:20-53,74-81`;
  CX-4.1; P6, P6b (a 100-line Node driver, no SDK); P7.
- Limitations: labelled experimental ("aren't supported for production
  workloads", [D:app-server]), although the TUI, `codex exec` and the VS Code
  extension are all app-server clients at 0.145.0 (CX-4.1). Requests can be
  refused with `-32001` when the server is overloaded ([T] README:49-53).
  `thread/start` with `cwd` and a write sandbox on an untrusted project writes
  `trust_level` into the user `config.toml`, which AGENTS.md forbids (CX-3.1).
- Fallback: `codex exec --json` plus `codex exec resume <id>`, one process per
  turn, with no deltas, no approvals (policy forced to `never`) and no quota
  (CX-4.2; P5, P5b).

## 2. Genuine streaming

**Claude Code.** With `--include-partial-messages`, `stream_event` carries
token-level deltas of the orchestrator's text, thinking and tool input; the
complete `assistant` message follows.
- Sources: doc `headless#stream-responses`; doc
  `agent-sdk/typescript#sdkpartialassistantmessage`; P1, P2.
- Limitations: subagents do not stream tokens: their text arrives as complete
  messages, and only with `--forward-subagent-text` (doc
  `headless#follow-subagent-messages`). In P1 to P3b, 0 of 230
  `stream_event` lines carried a non-null `parent_tool_use_id` (CC-10,
  question 15).
- Fallback: without partial messages, render each complete `assistant`
  message as it arrives (coarser, still real).

**Codex.** `item/agentMessage/delta`, reasoning deltas and command output
deltas, for the parent thread and, tagged by `threadId`, for each subagent
thread.
- Sources: [T] README:1477-1501; P6b lines 22-33; P7 (child deltas tagged with
  the child's `threadId`).
- Limitations: none for streaming itself; a noisy delta type can be switched
  off per connection with `optOutNotificationMethods` ([T] README:1389-1401).
- Fallback: render on `item/completed` only (the complete item).

## 3. Subagent lifecycle and tool events

**Claude Code.** `system/task_started`, `task_progress`, `task_updated`,
`task_notification` (with `status` and `usage`) and `background_tasks_changed`;
subagent `tool_use` / `tool_result` keyed by `parent_tool_use_id` and
`agent_id`; a subagent's approval request carries its `agent_id`.
- Sources: doc `agent-sdk/typescript`; doc `headless#follow-subagent-messages`;
  CC-5, CC-6; P3 and P3b (one backgrounded subagent: `is_backgrounded: true`,
  `spawn_depth: 1`, the full task sequence, no `tool_progress`).
- Limitations: two parallel subagents not probed; workflow phases (the
  `Workflow` tool) are not documented in the stream and were not probed under
  `-p` (CC-10, questions 6 and 7). The model a subagent actually used is only
  known from its forwarded messages' `message.model`.
- Fallback: `SubagentStart` / `SubagentStop` hooks registered through
  `initialize` (documented as SDK hooks; their raw-client behaviour is open,
  CC-10 question 5); the subagent transcript files (internal format,
  labelled so).

**Codex.** Each subagent is its own thread. P7 showed: the parent's
`item/completed` of type `subAgentActivity` (`kind: "started"`,
`agentThreadId`, `agentPath`) announces each child; the child then emits
`thread/status/changed`, `turn/started`, items (`commandExecution`,
`fileChange`, `mcpToolCall`, `webSearch`), deltas, `thread/tokenUsage/updated`
and `turn/completed` (with `durationMs`), all tagged with its `threadId`;
`collabAgentToolCall` items with `tool: "wait"` show the parent waiting.
- Sources: [T] `v2/item.rs:336-346,359-364,1153-1189`;
  `codex-rs/app-server/src/lib.rs:1083-1097`; CX-5; P7.
- Limitations: a child gets no `thread/started`, and no event marks its end
  except its `turn/completed` and `idle` status; the schema's `spawnAgent`
  call did not appear in P7 (possibly the V2 agent backend, which is not
  reported). Whether interrupting the parent stops the children is open
  (CX-10, question 3).
- Fallback: `thread/list` with `parentThreadId` and `thread/read` to rebuild
  the tree after the fact (experimental API, [T] README:144,418-435); a
  thread seen only through its events is shown with an "inferred" status.

## 4. Approvals and permissions

**Claude Code.** With `--permission-prompt-tool stdio`, every action the
native rules would prompt for arrives as a `can_use_tool` control request
(`tool_name`, `display_name`, `input`, `description`,
`permission_suggestions`, `tool_use_id`, and `agent_id` for subagents); the
dashboard answers allow (echoing `updatedInput`) or deny with a message.
`AskUserQuestion` uses the same path.
- Sources: PY `types.py:2281-2304`; doc `agent-sdk/user-input`; doc
  `permission-modes#available-modes`; P2, P3b (an allow echoing
  `updatedInput` ran the tool).
- Limitations: the `stdio` value of `--permission-prompt-tool` is
  undocumented. Native permission modes, rules and the sandbox decide first;
  only what would prompt reaches the dashboard, which never widens them.
  Accepting a `permission_suggestions` entry can persist a rule to settings:
  not offered in the MVP. The `AskUserQuestion` round trip is unprobed (CC-10,
  question 3).
- Fallback: an MCP permission-prompt tool served by the dashboard (documented,
  CC-4, way D). A request the dashboard cannot parse is denied, and shown.

**Codex.** Server-to-client requests `item/commandExecution/requestApproval`
and `item/fileChange/requestApproval` (plus permission and MCP elicitation
requests), answered `accept`, `acceptForSession`, `decline` or `cancel`; the
thread's status shows `waitingOnApproval`. A subagent's request carries the
child's `threadId`.
- Sources: [T] README:1526-1553; CX-4.1 step 4; P6b lines 35-47; P7.
- Limitations: a command that needs approval was not probed (only a file
  change; a read-only command ran without one, P6b line 56), so the exact
  command parameters on Windows and the outcome of `decline` and `cancel` are
  open (CX-10, question 1). The execpolicy and network "amendment" decisions
  persist rules to configuration: not offered in the MVP. `approvalPolicy` and
  `sandbox` are set per thread, from the owner's choice, never weaker than
  Codex's own settings.
- Fallback: a request the dashboard cannot render is answered `decline` (or,
  for an unknown method, a JSON-RPC error, as the probe driver does,
  [`drivers/codex-driver.mjs`](probes/drivers/codex-driver.mjs)) and shown;
  how Codex continues after either answer is unprobed (CX-10, question 1).

## 5. Persistence, cancellation and resumption

**Claude Code.** The CLI writes each session as JSONL itself; resume with
`--resume <id>`, `--continue`, or a fixed `--session-id`. Cancel a turn with
the `interrupt` control request (receipt `{"still_queued":[]}`, then
`result` with subtype `error_during_execution`); stop a background task with
`stop_task`.
- Sources: doc `headless#continue-conversations`; doc
  `headless#stop-a-run-with-sigterm`; doc `sessions#where-transcripts-are-stored`;
  P2 (interrupt during streaming text); P4 (resume).
- Limitations: a resumed process does not replay earlier messages on stdout
  (P4: only `system/init`, `assistant`, `rate_limit_event`, `result`), so the
  dashboard must keep its own record of the conversation to show it after a
  restart. Interrupt during a running Bash command or a pending approval is
  unprobed (CC-10, question 4). On Windows there are no POSIX signals (CC-8).
- Fallback: read the transcript file (internal format, labelled
  undocumented; the TS SDK's `getSessionMessages()` reads the same files, doc
  `agent-sdk/typescript#getsessionmessages`); last resort, close stdin and
  terminate the process, which leaves the turn unfinished.

**Codex.** The CLI persists every thread (rollout file in `thread.path`);
`turn/interrupt` ends a turn as `interrupted`; `thread/resume` reopens a
thread with its turns and replays `thread/tokenUsage/updated` (documented,
not probed).
- Sources: interrupt, P6b lines 80-91; persistence and resume, [T]
  README:77-81,321-377 and CX-4.1 step 7.
- Limitations: resume across two app-server processes not probed (CX-10,
  question 4); `turn/interrupt` does not stop background terminals ([T]
  README:1014); every app-server thread is recorded with `source: "vscode"`
  and so appears among the owner's interactive sessions (P6b line 9).
- Fallback: `thread/read` for history; `codex exec resume <id>` (CX-4.2);
  the rollout file (internal format).

## 6. Reusing an existing terminal UI

**Claude Code.** No, for the main path: hosting the TUI in a PTY yields a
repainted screen, not events; approvals and subagents would be scraped from
it; its keyboard, mouse and clipboard handling would fight the dashboard's
(amendment 3); ConPTY or tmux on Windows adds a terminal emulator to the
project.
- Sources: CC-4, way E; [`terminal-input.md`](terminal-input.md) sections 6 and 7.
- Limitations: the dashboard re-renders what the TUI would show, so features
  only the TUI has (its slash-command menus, its status line) are not
  inherited. The status line JSON exists only in the TUI (doc
  `statusline#available-data`).
- Fallback: an "open in Claude Code" action that runs `claude --resume <id>`
  in the project directory, in a separate terminal (escape hatch, outside
  the MVP's scope unless plan 3 keeps it).

**Codex.** No, for the same reasons; but the TUI is itself an app-server
client at 0.145.0, so everything it shows is reachable through the same
protocol, and its rules (window labelling by `windowDurationMins`, the
12,000-token context baseline) can be reused as reference.
- Sources: CX-4.3; [T] `codex-rs/tui/src/app_server_session.rs`;
  `codex-rs/tui/src/chatwidget/rate_limits.rs:77-104`;
  `codex-rs/tui/src/status/card.rs:326-333`.
- Limitations: the committed JSON Schema can generate Go types (Apache-2.0,
  notice kept), but TUI code is Rust: rules are reimplemented, not copied.
- Fallback: `codex resume <id>` in a separate terminal as the escape hatch.

## 7. Concurrent subagent output

**Claude Code.** All agents share one ordered NDJSON stream; each event
carries `agent_id`, `task_id` or `parent_tool_use_id`, so one reader routes
it to a per-agent buffer in the state, and each panel renders from its
buffer. Parallel subagents started by one turn share the Agent call's
`tool_use_id`, so routing keys on `agent_id` first.
- Sources: doc `agent-sdk/typescript#sdktoolprogressmessage`; CC-9 answer 7;
  P3b (subagent approval with its `agent_id`).
- Limitations: subagent text arrives whole, not as deltas (question 2); two
  parallel subagents not probed. Without `--forward-subagent-text`, zone C
  shows tools and status but not the subagent's text.
- Fallback: when an event lacks an agent key, attach it to the orchestrator
  and mark it "unattributed", never guess.

**Codex.** One stdout stream multiplexes every thread; route by `threadId`,
then `turnId` and `itemId`. P7 interleaved two children and the parent in one
stream without ambiguity.
- Sources: CX-9 answer 7; [T] README:1389-1401; P7.
- Limitations: rate-limit pushes are frequent with several threads (11
  `account/rateLimits/updated` in P7's single parent turn): coalesce before
  rendering.
- Fallback: `optOutNotificationMethods` for noisy delta types; per-thread
  `thread/read` to refill a panel.

## 8. Reliable quota

**Claude Code.** Two documented sources on the same stream:
`rate_limit_event` (`unifiedWindows.five_hour` and `.seven_day` with
`utilization` and `resetsAt`), emitted on transitions; and `/usage` sent as a
prompt, returning `usage_report` (experimental). The status line JSON has the
same values but only inside the TUI.
- Sources: doc `agent-sdk/typescript#sdkratelimitevent`; doc
  `agent-sdk/typescript#sdkusagereport`; PY `types.py:1780-1788`; CC-7;
  P1, P2, P3b (2 events for P1's one turn, 2 for P2's three, 1 for P3b's two).
- Limitations: neither arrives on a schedule; `/usage` over stream-json is
  unprobed (CC-10, question 9) and needs subscription credentials; the usage
  endpoint can itself be rate-limited (doc
  `costs#when-the-usage-request-fails`).
- Fallback: show the last value with its age, and "N/A" before the first.
  Never estimate quota from tokens.

**Codex.** `account/rateLimits/read` at startup and on demand, without a
turn, returns both windows (`primary` 300 min, `secondary` 10,080 min) with
`usedPercent` and `resetsAt`; `account/rateLimits/updated` is pushed after
model calls.
- Sources: [D:app-server]; [T] README:2206-2247, `v2/account.rs:294-300,
  518-534`; CX-7; P6 line 7, P6b lines 7 and 42, P7.
- Limitations: requires ChatGPT sign-in; with an API key it fails, and the
  value is N/A ([T] `request_processors/account_processor.rs:1016-1030`).
  Some plans have no 5-hour window ([D:pricing]), so windows are identified by
  `windowDurationMins`, never by position.
- Fallback: "N/A" with the reason (API key, or the read failed), and the last
  pushed value with its age.

## 9. Limits on Windows, WSL2, Linux and macOS

**Claude Code.**
- Windows native (this machine): works over pipes (P1 to P4). No Bash
  sandbox; the Bash tool needs Git for Windows (`CLAUDE_CODE_GIT_BASH_PATH`
  in the child's environment when `bash.exe` is not on the PATH, as here);
  no POSIX signals, so interrupt through the control request and end by
  closing stdin. P2 exited with code 1 after stdin closed (CC-10, question
  13, open).
- WSL2: a separate install and login inside the distribution; the sandbox is
  available there. A Windows dashboard would run it through `wsl.exe`, with
  Linux paths; pipe behaviour through `wsl.exe` is unverified.
- Linux, macOS: native install; sandbox (bubblewrap, Seatbelt); signals work.
- Sources: CC-8; doc `setup`; doc `sandboxing`; doc `env-vars`.
- Limitations: only Windows native was probed; no sandbox and no signals
  there; WSL2 through `wsl.exe`, Linux and macOS are from documentation.
- Fallback: on Windows without Git Bash, Claude Code uses its PowerShell tool
  (CC-8); the dashboard shows which shell it reported.

**Codex.**
- Windows native (this machine): supported for the CLI; app-server works over
  stdio without a PTY; commands run through `powershell.exe` (P6b). Its
  sandbox depends on Codex's own `[windows] sandbox` setting, and a write
  sandbox may be downgraded to read-only; `windowsSandbox/setupStart` changes
  system state and is never called.
- WSL2: install inside WSL; repositories should live in the Linux filesystem
  ([D:wsl]); a Windows dashboard can run `wsl.exe -- codex app-server` with
  Linux paths (unverified).
- Linux, macOS: bubblewrap with seccomp, Seatbelt.
- Sources: CX-8; [D:win-sandbox]; [D:wsl]; [D:commands]; [T]
  `thread_processor.rs:1130-1134`.
- Limitations: only Windows native was probed; the Windows sandbox depends on
  the owner's Codex setting and may fall back to read-only; WSL2, Linux and
  macOS are from documentation.
- Fallback: the dashboard reports the effective `sandbox` returned by
  `thread/start`, so a downgrade is visible, never hidden.

## Common provider shape

What plan 2 can build one interface around, with the field each CLI uses:

| Concept | Claude Code | Codex |
| - | - | - |
| Start | spawn with `cmd.Dir` = project, then `initialize` | spawn with `cmd.Dir` = project, `initialize`, `thread/start` without `cwd` until the trust-write probe (CX-10, question 2) |
| Send a message | user message on stdin | `turn/start` |
| Text delta | `stream_event` | `item/agentMessage/delta` |
| Agent key | `agent_id` / `parent_tool_use_id` | `threadId` |
| Agent started / ended | `task_started` / `task_notification` | `subAgentActivity started` / child `turn/completed` |
| Approval | `can_use_tool` request, allow or deny | `item/*/requestApproval`, accept or decline |
| Interrupt | `interrupt` control request | `turn/interrupt` |
| Quota | `rate_limit_event`, `/usage` | `account/rateLimits/read` and `updated` |
| Tokens, context | `assistant.message.usage`, `result.modelUsage`, `get_context_usage` | `thread/tokenUsage/updated` |
| Cost | `result.total_cost_usd` (API-equivalent estimate) | N/A |
| Resume | `--resume <id>`, no replay | `thread/resume`, with history |

Unknown event types and fields are kept, logged and shown as "unrecognised",
so a newer CLI degrades visibly instead of breaking (CX-9).

## Path forms for the startup screen

Amendment 4, question 2: the owner types a directory (examples
`C:\Users\Developer\Projects\my-app` and `~/projects/my-app`), prefilled with
the current directory; the application expands and validates it, supporting
"Windows, Linux, macOS, and WSL paths where applicable"; the directory must
exist and be accessible, and becomes the agent's working directory; no
project browser (`doc/brief.md`, amendment 4).

"Where applicable" depends on where the dashboard runs, which plan 2 decides.
The rules below cover both cases: a Windows-native dashboard (this machine,
the agent CLIs installed natively) and a Linux or macOS dashboard (including
one running inside WSL).

### Forms and rules

| Form typed | Example | Dashboard on Windows | Dashboard on Linux, macOS or in WSL | Source |
| - | - | - | - | - |
| DOS absolute | `C:\dev\app`, `C:/dev/app` | Accept. `filepath.Clean` (turns `/` into `\`, evaluates `.` and `..`) | In WSL: translate `C:\dev\app` to `/mnt/c/dev/app` (inferred, see WSL notes). Elsewhere: reject, "a Windows path" | [MS:paths] "Traditional DOS paths"; [Go:filepath] `Clean`, `FromSlash` |
| Drive-relative | `C:dev\app` | Reject: it is relative to that drive's current directory, "a common source of bugs"; ask for `C:\dev\app` | Reject | [MS:paths] |
| Root-relative | `\dev\app` | Reject: relative to the root of the current drive; ask for a full path | n/a (it is not a POSIX path) | [MS:paths] |
| Relative | `app`, `.\app`, `../app` | Resolve with `filepath.Abs` against the dashboard's current directory, and show the result before accepting | Same | [Go:filepath] `Abs` |
| Home | `~`, `~/projects/app`, `~\projects\app` | Expand `~` to `os.UserHomeDir()` (`%USERPROFILE%`), then as absolute | Expand `~` to `os.UserHomeDir()` (`$HOME`) | [Go:os] `UserHomeDir` |
| Another user's home | `~bob/app` | Reject: not expanded | Reject (the shell form is not supported) | inferred: no source names a portable way to resolve it |
| UNC | `\\server\share\app` | Accept if `Stat` shows a directory; warn that a network share can be slow or disappear | n/a | [MS:paths] "UNC paths": must be fully qualified; [Go:filepath] `VolumeName` |
| Device path | `\\?\C:\dev\app`, `\\.\C:\dev\app` | Accept `\\?\C:\...`, `\\.\C:\...` and `\\?\UNC\server\share\...` by rewriting them to `C:\...` and `\\server\share\...`, then as above; reject other device paths (`\\.\PhysicalDrive0`, volume GUIDs) | n/a | [MS:paths] "DOS device paths"; the rewrite is inferred from the page's statement that these forms name the same file, and avoids handing an agent CLI a cwd it may not handle |
| WSL drive mount | `/mnt/c/dev/app` | Translate to `C:\dev\app` (drive letter after the mount root, upper-cased, rest with `\`), then as DOS absolute; show the result | Accept as POSIX absolute | [MS:wslfs] "/mnt/c"; [MS:wslconf] `automount.root` default `/mnt/` |
| WSL Linux filesystem | `\\wsl$\Ubuntu\home\u\app`, `\\wsl.localhost\Ubuntu\home\u\app` | Accept only if `Stat` shows a directory (the distribution must be running), with a warning: the files live in Linux, cross-system access is slow, and a native Windows CLI running there is unverified | In WSL: translate to `/home/u/app` for the same distribution (inferred) | [MS:wslfs]; [MS:wslconf] (example path `\\wsl.localhost\...`) |
| POSIX absolute | `/home/u/app`, `/Users/u/app` | Reject, unless it starts with the mount root (`/mnt/<letter>/`) | Accept; `filepath.Clean` | [Go:filepath] `Clean`, `IsAbs` |
| Empty | (nothing typed) | Use the prefilled current directory | Same | amendment 4 |

Surrounding quotes (a path pasted from Explorer's "Copy as path") are removed
before parsing; inner spaces are kept. Windows trims trailing periods and
spaces from path segments during normalization ([MS:paths] "Normalize
paths"), so `C:\dev\app.` resolves to `C:\dev\app`: always show the resolved
path the agent will receive, never only what was typed.

### Validation, in order

1. Trim surrounding whitespace and quotes; expand `~`; translate the WSL forms
   as above.
2. Classify the form; reject the forms marked "reject" with a message that
   names the form and gives the corrected example.
3. Make it absolute (`filepath.Abs`) and clean it (`filepath.Clean`).
4. `os.Stat`: it must exist and be a directory (`IsDir`); otherwise "does not
   exist" or "is a file", with the resolved path ([Go:os] `Stat`,
   `FileInfo.IsDir`).
5. Accessibility: open the directory and read one entry (`os.Open`,
   `File.ReadDir(1)`); a permission error is shown as "not readable"
   ([Go:os]). Write access is not tested: the agent CLI's own sandbox and
   permissions decide what it may write (AGENTS.md, hard constraints).
6. Show the resolved absolute path; on confirmation it becomes the child's
   working directory (`cmd.Dir`), for Claude Code and for Codex alike (CC-3;
   CX-3.1).

Windows paths are case-insensitive ([MS:paths]); the dashboard keeps the case
the owner typed and does not compare paths itself in the MVP.

### WSL notes

- From WSL, Windows drives appear under `/mnt/<letter>`; from Windows, Linux
  files appear under `\\wsl$` ([MS:wslfs]). The mount root is configurable
  (`[automount] root`, default `/mnt/`, [MS:wslconf]), so a translated path
  is always checked with `Stat` before use, and a failed check says that the
  translation may not match a custom mount root.
- Microsoft advises storing files on the same operating system as the tools
  that use them, for performance ([MS:wslfs]); Codex says the same for its
  WSL install ([D:wsl]). A Linux-filesystem project used by a Windows-native
  agent CLI goes against that advice: warned, not blocked.
- Running the agent CLI inside WSL from a Windows dashboard (`wsl.exe`, which
  "uses the same working directory" and needs WSL-format paths, [MS:wslfs])
  is outside the MVP's "one agent CLI per session, installed natively" path on
  this machine, and unverified: an open question for plan 2.

## Open points for plan 2

- Where the dashboard runs (Windows native, WSL, or both), which fixes which
  column of the path table applies.
- Whether a native Windows agent CLI accepts a UNC working directory
  (`\\server\share`, `\\wsl$\...`): unprobed for both CLIs.
- The open probe questions that touch these answers: Claude Code interrupt
  during a tool and a pending approval, `AskUserQuestion`, hook callbacks,
  `/usage` over stream-json, two parallel subagents (CC-10); Codex command
  approval and `decline`, the trust write, parent interrupt with children,
  resume across processes (CX-10).
