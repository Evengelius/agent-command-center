# Claude Code: integration research

Plan 1, step 1. Subject: how `agent-command-center` (Go, Bubble Tea) can drive
and observe Claude Code, installed here as version 2.1.300. The research
itself never ran `claude`. The owner-approved probes of plan 1, step 3
(`doc/research/probes/`, recorded 2026-10-15 against 2.1.300) are cited as
evidence where they settle a point; section 10 gives the outcome of every
question (plan 14: settled, or closed with the reason).

Conventions used below:

- **doc `page#section`** means `https://code.claude.com/docs/en/page`,
  section named after `#`, accessed 2026-10-15.
- **PY** is `anthropics/claude-agent-sdk-python` at commit
  `c10b52c674e5c709c7263544b38e8a5dad10e01d`, paths relative to
  `src/claude_agent_sdk/`. **CC** is `anthropics/claude-code` at
  `94506ff0d3bd9185f1752da36e968775021fb48f`. **TS** is
  `anthropics/claude-agent-sdk-typescript` at
  `fbc529490ca0fb07b486c6f9c8e553d3fb7c690d`.
- **Documented** means described on the docs site (URL given).
  **Undocumented** means found only in SDK code, release notes or probe
  recordings, never on the docs site. **Inferred** means a conclusion drawn
  from documented facts, not stated anywhere and not yet observed.
- **Pn line k** means line k of the probe recording
  `doc/research/probes/p<n>-claude-*.jsonl` (format and sanitising rules in
  `doc/research/probes/README.md`). Driver recordings wrap each message as
  `{"t_ms", "dir", "msg"}`; P1 and P4 are raw stdout.

## 1. Sources

### Documentation pages (all accessed 2026-10-15)

Index: https://code.claude.com/docs/llms.txt (222 pages listed).

| Page | URL |
| - | - |
| Run Claude Code programmatically (headless) | https://code.claude.com/docs/en/headless |
| CLI reference | https://code.claude.com/docs/en/cli-reference |
| Agent SDK overview | https://code.claude.com/docs/en/agent-sdk/overview |
| Agent SDK reference, TypeScript | https://code.claude.com/docs/en/agent-sdk/typescript |
| Agent SDK reference, Python | https://code.claude.com/docs/en/agent-sdk/python |
| Streaming input | https://code.claude.com/docs/en/agent-sdk/streaming-vs-single-mode |
| Stream responses in real time | https://code.claude.com/docs/en/agent-sdk/streaming-output |
| Handle approvals and user input | https://code.claude.com/docs/en/agent-sdk/user-input |
| SDK permissions | https://code.claude.com/docs/en/agent-sdk/permissions |
| SDK sessions | https://code.claude.com/docs/en/agent-sdk/sessions |
| SDK hooks | https://code.claude.com/docs/en/agent-sdk/hooks |
| SDK subagents | https://code.claude.com/docs/en/agent-sdk/subagents |
| Track cost and usage | https://code.claude.com/docs/en/agent-sdk/cost-tracking |
| Agent loop, observability, hosting | https://code.claude.com/docs/en/agent-sdk/agent-loop, .../agent-sdk/observability, .../agent-sdk/hosting |
| Hooks reference, hooks guide | https://code.claude.com/docs/en/hooks, https://code.claude.com/docs/en/hooks-guide |
| Subagents | https://code.claude.com/docs/en/sub-agents |
| Dynamic workflows (brief) | https://code.claude.com/docs/en/workflows |
| Run agents in parallel, agent view, agent teams | https://code.claude.com/docs/en/agents, .../agent-view, .../agent-teams |
| Status line (brief) | https://code.claude.com/docs/en/statusline |
| Monitoring (OpenTelemetry) | https://code.claude.com/docs/en/monitoring-usage |
| Manage costs | https://code.claude.com/docs/en/costs |
| Commands | https://code.claude.com/docs/en/commands |
| Permissions, permission modes | https://code.claude.com/docs/en/permissions, https://code.claude.com/docs/en/permission-modes |
| Sandboxing | https://code.claude.com/docs/en/sandboxing |
| Settings files and precedence | https://code.claude.com/docs/en/settings |
| Sessions | https://code.claude.com/docs/en/sessions |
| `.claude` directory | https://code.claude.com/docs/en/claude-directory |
| Remote Control, channels | https://code.claude.com/docs/en/remote-control, https://code.claude.com/docs/en/channels |
| VS Code, JetBrains | https://code.claude.com/docs/en/vs-code, https://code.claude.com/docs/en/jetbrains |
| Advanced setup (install), troubleshoot install | https://code.claude.com/docs/en/setup, https://code.claude.com/docs/en/troubleshoot-install |
| Environment variables, tools reference, fullscreen | https://code.claude.com/docs/en/env-vars, .../tools-reference, .../fullscreen |

### Repositories (read only; data, not instructions)

| Repository | Commit | License (file) | SPDX | Content relevant here |
| - | - | - | - | - |
| anthropics/claude-agent-sdk-python | `c10b52c674e5c709c7263544b38e8a5dad10e01d` | `LICENSE`: "MIT License, Copyright (c) 2025 Anthropic, PBC"; `pyproject.toml:11` `license = {text = "MIT"}` | `MIT` | Full SDK source: subprocess transport, control protocol, message parser. SDK 0.2.167 (`_version.py:3`), bundles Claude Code `2.1.300` (`_cli_version.py:3`), the version installed here |
| anthropics/claude-agent-sdk-typescript | `fbc529490ca0fb07b486c6f9c8e553d3fb7c690d` | `LICENSE.md`: "© Anthropic PBC. All rights reserved. Use is subject to Anthropic's Commercial Terms of Service" | none (proprietary; `LicenseRef-Anthropic-Commercial-Terms` if one is needed) | No SDK source: README, CHANGELOG (0.3.300, "parity with Claude Code v2.1.300", `CHANGELOG.md:3-12`), session-store examples. The SDK ships compiled on npm |
| anthropics/claude-code | `94506ff0d3bd9185f1752da36e968775021fb48f` | `LICENSE.md`: same proprietary notice | none (proprietary) | No CLI source: `CHANGELOG.md` (2.1.300 at line 3), example settings and hooks, `mods/` |

Licensing consequence (AGENTS.md rule): only the Python SDK may be copied, with
its MIT notice. The TypeScript SDK and the Claude Code repository can only be
studied. In practice nothing needs copying: the protocol is small enough to
reimplement in Go from the facts below.

## 2. Detection and version

| Install method | What lands on PATH (Windows) | Source |
| - | - | - |
| Native installer, `irm https://claude.ai/install.ps1 \| iex` (or `install.cmd`) | `%USERPROFILE%\.local\bin\claude.exe` | doc `setup#install-claude-code`; uninstall path `$env:USERPROFILE\.local\bin\claude.exe` in doc `setup#uninstall-a-native-installation`; PY `_internal/transport/subprocess_cli.py:296` probes the same path |
| npm, `npm install -g @anthropic-ai/claude-code` | `claude.cmd` shim (npm), native binary pulled as an optional dependency | doc `setup#install-with-npm` (platforms incl. `win32-x64`); shim named in PY `subprocess_cli.py:329-330` |
| WinGet, Homebrew, apt/dnf/apk | package-manager paths | doc `setup#uninstall-with-winget` and following sections |
| macOS/Linux native | `~/.local/bin/claude`, a symlink into `~/.local/share/claude/versions/` | doc `setup#auto-updates` |

On this machine, a PATH lookup (no execution) finds only
`%USERPROFILE%\.local\bin\claude.exe`: the native installer. Git for
Windows is at `C:\Program Files\Git\bin\bash.exe` but not on PATH; the `bash`
on PATH is WSL's (`C:\windows\system32\bash.exe`).

Detection rules for Go, taken from what the Python SDK does (PY
`subprocess_cli.py:255-339`):

1. Look up `claude` on PATH; on Windows accept only a `.exe`/`.com` result,
   else look up `claude.exe` explicitly, else try
   `%USERPROFILE%\.local\bin\claude.exe`.
2. Never spawn a `.cmd`/`.bat` shim with runtime arguments: Windows runs it
   through `cmd.exe`, which re-parses arguments (CVE-2024-27980, "BatBadBut").
   The SDK refuses (PY `subprocess_cli.py:415-448`) and also rejects cmd.exe
   metacharacters in `--resume`/`--session-id` values (`:451-471`). If only the
   shim exists, report "npm install found, native claude.exe needed" rather
   than guess.
3. Version: `claude --version` or `claude -v` prints e.g.
   `2.1.211 (Claude Code)` (doc `setup#verify-your-installation`); the SDK runs
   `claude -v` with a 2 s timeout and matches `^([0-9]+\.[0-9]+\.[0-9]+)`
   (PY `subprocess_cli.py:1164-1189`, minimum `2.0.0` at `:37`). The version
   also arrives at runtime in `system/init.claude_code_version` and in the
   `initialize` response `claude_code_version` (doc
   `agent-sdk/typescript#sdksystemmessage`,
   `#sdkcontrolinitializeresponse`). Prefer the `capabilities` array of
   `system/init` over version comparisons (doc
   `headless#read-session-metadata`).
4. Optional, read-only diagnostics: `claude doctor` and `claude auth status`
   (JSON, `authMethod` one of `none`, `claude.ai`, `oauth_token`, `api_key`,
   `api_key_helper`, `third_party`; exit 1 when logged out) (doc
   `cli-reference#cli-commands`). `authMethod` tells the dashboard whether
   subscription quota can exist at all.

## 3. Working directory, permissions, sandbox, configuration

### Working directory

The SDK starts the CLI with the process working directory set to the chosen
folder and `PWD` set to the same path (PY `subprocess_cli.py:867-878`); there is
no `--cwd` flag for a session. Extra folders: `--add-dir` (grants file access,
does not load their `.claude/` configuration; doc
`cli-reference#cli-flags`). Sessions are stored per working directory
(doc `sessions#where-transcripts-are-stored`).

### Permission modes (documented, doc `permission-modes#available-modes`)

| Mode (`--permission-mode`) | Runs without asking |
| - | - |
| `default` (alias `manual`, labelled Manual) | Reads only |
| `acceptEdits` | Reads, file edits, common filesystem commands (`mkdir`, `touch`, `mv`, `cp`) |
| `plan` | Reads, plus classifier-approved commands when auto mode is available |
| `auto` | Everything, with a background classifier reviewing actions |
| `dontAsk` | Reads in working directories and pre-approved tools; anything that would prompt is denied |
| `bypassPermissions` | Everything (containers/VMs only) |

- Deny rules apply in every mode; some actions are never auto-approved (ask
  rules, protected paths, critical-path removals) (doc
  `permission-modes#actions-no-mode-auto-approves`).
- Starting mode of a `claude -p` / SDK session when nothing sets one: `default`
  in sessions that fetch feature flags, otherwise `auto` on 2.1.285+ (doc
  `permission-modes#which-mode-a-session-starts-in`). Interactive terminal
  sessions start in `auto` on 2.1.283+. The dashboard should always pass
  `--permission-mode` so the mode is known, not inferred.
- The mode can change mid-session: control request `set_permission_mode` (PY
  `_internal/query.py:796-803`; documented as SDK method `setPermissionMode()`,
  doc `agent-sdk/typescript#query-object`). The current mode is reported in
  `system/init.permissionMode` and `system/status.permissionMode`.

### Sandbox (documented, doc `sandboxing`)

- An OS-enforced boundary around shell commands only (Bash, PowerShell,
  Monitor); file tools, MCP servers and hooks run outside it.
- macOS (Seatbelt), Linux and WSL2 (`bubblewrap` + `socat`). **On native
  Windows commands run unsandboxed**; use WSL2 for the sandbox (doc
  `sandboxing`, intro; doc `setup#set-up-on-windows`).
- Two modes: auto-allow (sandboxed commands run without a prompt) and regular
  permissions (sandboxed commands still prompt) (doc `sandboxing#sandbox-modes`).
- Configured through the `sandbox` settings key; the Python SDK passes it in
  the `--settings` JSON (PY `subprocess_cli.py:473-525`). The dashboard never
  weakens it (AGENTS.md): it passes no sandbox setting at all and inherits the
  owner's.

### How configuration loads, and how to stay read-only

Precedence, highest first (doc `settings#settings-precedence`): managed
settings > `--settings <file-or-json>` (this session only) >
`.claude/settings.local.json` > `.claude/settings.json` >
`~/.claude/settings.json`. Lists such as `permissions.allow` merge across
levels. `--setting-sources user,project,local` limits which files load.

Per-invocation levers that write nothing to disk:

| Lever | Effect | Source |
| - | - | - |
| CLI flags (`--permission-mode`, `--model`, `--effort`, `--allowedTools`, `--add-dir`, ...) | One session | doc `cli-reference#cli-flags` |
| `--settings '<json>'` | Any key the user settings file can set, this session only; files untouched | doc `settings#change-a-setting-for-one-session` |
| Environment variables on the child process | One process | doc `env-vars` |

Ways the dashboard could change configuration by accident, all to be avoided:

- Answering a permission request with `updatedPermissions` whose destination
  is `userSettings`, `projectSettings` or `localSettings`. Bash prompts arrive
  with a `localSettings` suggestion that writes `.claude/settings.local.json`
  if returned (doc `agent-sdk/typescript#canusetool`, `#permissionupdatedestination`).
  "Always allow" must use destination `session` only.
- The `updateSettings` control request (writes a settings file; doc
  `agent-sdk/typescript#query-object`).
- Slash commands that save: `/config` writes settings files, `/model` saves a
  default (doc `settings#change-a-setting-for-one-session`). Whether the
  `set_model` control request persists anything is not documented (probe).
- `--bare` avoids loading the owner's hooks, skills and CLAUDE.md but never
  uses the subscription login (needs `ANTHROPIC_API_KEY`) (doc
  `headless#start-faster-with-bare-mode`), so it would remove the subscription
  quota the dashboard must show. Not usable for the MVP.

Two behaviours to surface to the owner: a `-p` session runs the project's
`.claude/settings.json` hooks and `.mcp.json` servers without a trust dialog
(doc `headless#start-faster-with-bare-mode`); and Claude Code itself writes its
session transcripts under `~/.claude/projects/` as part of normal operation
(doc `sessions#where-transcripts-are-stored`). Neither is a change made by the
dashboard.

## 4. Ways to drive a session from another program

| # | Interface | Transport | Status | Streaming | Multi-turn in one process | Approvals | Interrupt | Resume | Verdict |
| - | - | - | - | - | - | - | - | - | - |
| A | `claude -p` with `--input-format stream-json --output-format stream-json` and the control protocol | NDJSON on child stdin/stdout | Flags documented; control protocol partly documented, mostly SDK-code only | Token deltas (`stream_event`) for the main thread; complete content blocks for subagents | Yes | Yes, `can_use_tool` control requests (`--permission-prompt-tool stdio`) | Yes, `interrupt` control request | Yes, `--resume`, `--continue`, `--session-id` | **Recommended** |
| B | `claude -p "<prompt>"` per turn with `--output-format stream-json` and `--resume <id>` | argv + stdout | Documented | Same as A for one turn | No, one process per turn | Only pre-approved tools, or an MCP prompt tool (D) | SIGINT/SIGTERM only | Yes | Fallback if A fails a probe |
| C | Agent SDK (Python/TS) as a sidecar process the Go app talks to | SDK API over our own IPC | Documented | Same as A | Yes | `canUseTool` | `interrupt()` | Yes | Adds Python or Node at runtime; TS SDK proprietary. Fallback only |
| D | `--permission-prompt-tool <mcp tool>` served by an MCP server we provide | MCP | Documented | n/a (approvals only) | n/a | Yes, via MCP tool calls | n/a | n/a | Approval fallback for B |
| E | Interactive TUI in a PTY or tmux | Terminal bytes | TUI documented, no machine interface | Screen only | Yes | On screen | Esc keystroke | Yes | Rejected (below) |
| F | Remote Control (`claude remote-control`, `--rc`) | Outbound HTTPS to Anthropic; claude.ai / app clients | Documented | n/a locally | n/a | n/a | n/a | n/a | No local API; not usable |
| G | Background sessions: `claude --bg`, `claude agents --json`, `claude logs`, `claude attach` | Supervisor daemon with sockets | CLI commands documented; daemon protocol undocumented | Logs only | n/a | n/a | `claude stop` | `claude respawn` | Out of MVP scope |
| H | Channels (MCP server pushing events into a session) | MCP | Documented, research preview | Input only | n/a | n/a | n/a | n/a | Not needed |
| I | VS Code / JetBrains integration | The VS Code extension drives the CLI with `--input-format stream-json` (CC `CHANGELOG.md:1650`) | Extension documented; transport only in release notes | Same as A | Yes | Yes | Yes | Yes | Confirms A is Anthropic's own GUI path |

### A. stream-json with the control protocol (recommended)

What is documented:

- `--input-format stream-json` and `--output-format stream-json` in print
  mode; `--verbose` and `--include-partial-messages` for token deltas; the
  last line of a turn is a `result` message (doc `headless#stream-responses`,
  doc `cli-reference#cli-flags`).
- Every message type on stdout, with fields (doc `agent-sdk/typescript#message-types`;
  list in section 6).
- User messages on stdin and their fields: `type: "user"`, `message`
  (Anthropic `MessageParam`), `parent_tool_use_id: null`, and optional
  `uuid`, `priority` (`now`/`next`/`later`), `origin: {kind: "human"}`,
  `client_composed`, `shouldQuery`, `pasted_content`, `inline_pastes` (doc
  `agent-sdk/typescript#sdkusermessage`). `origin` should be set to
  `{kind: "human"}` for typed input (doc `agent-sdk/typescript#sdkmessageorigin`).
- The envelope of control traffic: `{"type": "control_request",
  "request_id": ..., "request": {...}}`, answered by a `control_response`
  that echoes `request_id`; permission prompts do not time out (doc
  `agent-sdk/typescript#canusetool`, `#sdkcontrolinitializeresponse`). The
  docs address "a client that drives the CLI's control protocol directly"
  (doc `agent-sdk/typescript#sdkcontrolinterruptresponse`) and "if you parse
  the wire protocol yourself" (`#sdkcontrolinitializeresponse`), so direct use
  is anticipated, but the request catalogue below is not on the docs site.
- Interrupt receipt `{still_queued, cancelled?}` and `cancel_queued: true`
  (capability `interrupt_cancel_queued_v1`) (doc
  `agent-sdk/typescript#sdkcontrolinterruptresponse`).
- `--permission-prompts none` for unattended runs (doc
  `headless#turn-off-permission-prompts-in-unattended-runs`); not for the
  dashboard, which has a person at the keyboard.

What exists only in SDK code (undocumented at the CLI level), from PY at the
commit above, which bundles the installed CLI version 2.1.300:

| Item | Wire form | Source |
| - | - | - |
| Command line the SDK builds | `claude --output-format stream-json --verbose [--system-prompt ""] [--tools ...] [--allowedTools ...] [--disallowedTools ...] [--max-turns N] [--model M] [--permission-prompt-tool stdio] [--permission-mode M] [--continue] [--resume=ID] [--session-id=UUID] [--settings JSON] [--add-dir D]... [--mcp-config JSON] [--include-partial-messages] [--include-hook-events] [--fork-session] [--setting-sources=...] [--effort E] --input-format stream-json` | PY `_internal/transport/subprocess_cli.py:570-795` |
| No `-p` / `--print` on that command line | The docs say `--input-format` and `--include-partial-messages` belong to print mode; the SDK relies on non-TTY stdio instead. Inferred: passing `-p` explicitly is equivalent; probe | PY `subprocess_cli.py:574`, `:793` |
| `--system-prompt ""` when the caller sets no system prompt | Replaces Claude Code's default system prompt with an empty one. The dashboard must **not** copy this, or the conversation stops behaving like Claude Code | PY `subprocess_cli.py:576-577` |
| `--permission-prompt-tool stdio` | Special value that routes permission prompts to stdout as `can_use_tool` control requests; the docs only describe MCP tool names for this flag | PY `types.py:2281-2304` |
| Child environment | `CLAUDE_CODE_ENTRYPOINT=sdk-py`, `CLAUDE_AGENT_SDK_VERSION`, `CLAUDE_CODE_SDK_READS_SESSION_STATE=1`; removes `CLAUDECODE` | PY `subprocess_cli.py:44`, `:819-861` |
| Handshake | First write: `control_request` `{"subtype": "initialize", "hooks": {Event: [{"matcher": ..., "hookCallbackIds": ["hook_0"], "timeout"?}]} \| null, "agents"?, "skills"?, "forwardSubagentText"?, ...}`; 60 s timeout | PY `_internal/query.py:309-363`; `client.py:244` |
| Request IDs | `req_<counter>_<8 hex>` (any unique string) | PY `_internal/query.py:720` |
| Host to CLI requests | `initialize`, `interrupt`, `set_permission_mode {mode}`, `set_model {model}`, `get_context_usage`, `mcp_status`, `rewind_files {user_message_id}`, `mcp_reconnect {serverName}`, `mcp_toggle {serverName, enabled}`, `stop_task {task_id}` | PY `_internal/query.py:784-868`; `types.py:2796-2896` |
| CLI to host requests | `can_use_tool {tool_name, input, permission_suggestions, blocked_path, decision_reason?, title?, display_name?, description?, tool_use_id, agent_id?}`; `hook_callback {callback_id, input, tool_use_id}`; `mcp_message {server_name, message}` | PY `types.py:2800-2840`; `_internal/query.py:577-677` |
| Responses | `{"type": "control_response", "response": {"subtype": "success", "request_id": ..., "response": {...}}}` or `{"subtype": "error", "request_id": ..., "error": "..."}` | PY `_internal/query.py:680-704`; `types.py:2885-2900` |
| Permission answer | allow: `{"behavior": "allow", "updatedInput": <input or edited input>, "updatedPermissions"?: [...]}`; deny: `{"behavior": "deny", "message": "...", "interrupt"?: true}` | PY `_internal/query.py:617-634`; same shape documented as `PermissionResult`, doc `agent-sdk/typescript#permissionresult` |
| Cancellation from the CLI | `{"type": "control_cancel_request", "request_id": ...}` withdraws a pending request (e.g. a permission prompt) | PY `_internal/query.py:420-426` |
| Run-state frames | `system/session_state_changed` with `sdk_host_only: true` when `CLAUDE_CODE_SDK_READS_SESSION_STATE=1`; the public form needs `CLAUDE_CODE_EMIT_SESSION_STATE_EVENTS=1` (documented, doc `env-vars`) | PY `_internal/query.py:446-452` |
| Framing | One JSON object per line; skip lines not starting with `{` (some builds print `[SandboxDebug]` lines to stdout); 1 MB per-line default cap in the SDK | PY `subprocess_cli.py:199-221`, `:36` |
| Shutdown | Close stdin, wait 5 s, terminate, wait 5 s, kill | PY `subprocess_cli.py:962-1061` |

Conclusion: a Go program can speak this protocol directly: it is NDJSON with
roughly ten request types, and the Python SDK (MIT) shows every field. The
risk is that the catalogue is undocumented and can change; mitigations are the
documented `capabilities` array, the version pin of the bundled SDK, and a
probe suite run on each Claude Code update.

Per-requirement summary for A:

- **Streaming granularity**: `stream_event` carries raw Anthropic stream events
  (`message_start`, `content_block_delta` with `text_delta` /
  `input_json_delta` / thinking, `message_stop`) for the **main thread only**;
  `parent_tool_use_id` is always null on them (doc
  `agent-sdk/typescript#sdkpartialassistantmessage`). Complete `assistant`
  messages follow each finished content block, sharing `message.id` (doc
  `agent-sdk/streaming-output#message-flow`). A cut stream still gets
  `message_stop` on 2.1.290+ (doc `agent-sdk/streaming-output#handle-a-stream-thats-cut-off`).
- **Multi-turn**: one process carries many turns; each turn ends with its own
  `result`. A message sent mid-turn is read after the current tool calls
  (`priority: next`), held to the next turn (`later`), or interrupts or
  backgrounds work (`now`) (doc `agent-sdk/typescript#sdkusermessage`).
- **Approvals**: `can_use_tool` reaches the host only when rules and mode
  resolve to a prompt; requests from background subagents carry `agent_id`
  (doc `agent-sdk/typescript#canusetool`; CC `CHANGELOG.md:1212` for
  background subagent prompts). A `system/permission_denied` message reports
  denials the CLI decided itself (doc `#sdkpermissiondeniedmessage`).
- **Clarification questions**: `AskUserQuestion` arrives as a `can_use_tool`
  request with `tool_name: "AskUserQuestion"`, input `{questions: [{question,
  header, options: [{label, description}], multiSelect}]}` (1-4 questions,
  2-4 options); answer with allow and `updatedInput: {questions, answers:
  {"<question>": "<label>"}, response?}`. Not available inside subagents (doc
  `agent-sdk/user-input#handle-clarifying-questions`, `#limitations`).
- **Interrupt/cancel**: control request `interrupt` (PY `_internal/query.py:792-794`);
  receipt documented. SIGINT also ends the turn; SIGTERM exits with 143 and
  leaves the turn unfinished; closing stdin cancels a pending permission
  prompt (doc `headless#stop-a-run-with-sigterm`). Background tasks: control
  request `stop_task {task_id}`.
- **Resume**: `--resume <session-id | name | absolute .jsonl path>`,
  `--continue` (includes `-p` and SDK sessions), `--session-id <uuid>` to fix
  the ID up front, `--fork-session` (doc `headless#continue-conversations`,
  doc `cli-reference#cli-flags`). `-p`/SDK sessions do not appear in the
  interactive picker but resume by ID (doc
  `sessions#loop-p-agent-sdk-and-background-sessions`). Resuming does not
  replay earlier messages on stdout (P4, section 10).
- **Limitations**: agent teams never spawn under `-p` (doc
  `agent-teams#enable-agent-teams`); fork mode is off under `-p`, so Claude
  may run subagents in the foreground (doc
  `sub-agents#run-subagents-in-foreground-or-background`); terminal-only
  commands such as `/login` are unavailable (doc `headless#create-a-commit`,
  note); the status line is a TUI element and, inferred, does not run under
  `-p`.

### B. One `claude -p` per turn

Documented and simplest: `claude -p "<text>" --output-format stream-json
--verbose --include-partial-messages --resume <id>`. Each turn pays process
start-up and MCP connection time; a prompt that needs approval is denied
unless an allow rule, the mode, a `PermissionRequest` hook or an MCP prompt
tool (D) approves it (doc `agent-sdk/typescript#sdkpermissiondeniedmessage`).
Interrupt is a signal only, and on native Windows a console Ctrl+C event to a
child process group is the only SIGINT equivalent (inferred).

### C. Agent SDK as a sidecar

Documented and complete, but makes Python (PY bundles `claude.exe` in its
Windows wheel) or Node a runtime dependency, needs an IPC protocol of our own
between Go and the sidecar, and the TS SDK is proprietary. It gives no data
the CLI does not already emit, because the SDK is a thin client of A. Keep as
the fallback if raw control traffic misbehaves.

### E. Interactive TUI in a PTY or tmux: why not

- No structured data: the dashboard would scrape a screen that Claude Code
  redraws itself (classic renderer, or the fullscreen alternate-screen
  renderer, doc `fullscreen`). The conversation would not be "real" in the
  AGENTS.md sense of coming from the session's messages; it would be pixels
  re-parsed.
- Two TUIs fight over one terminal: key bindings, mouse capture, clipboard,
  bracketed paste (amendment 3) would all have to be forwarded byte for byte.
- On Windows this needs ConPTY; tmux exists only under WSL on this machine
  (AGENTS.md, Environment).
- Acceptable only as an "open in native Claude Code" escape hatch: suspend the
  dashboard and run `claude --resume <id>` in the same terminal.

## 5. Subagents and parallel work

How Claude Code runs them (documented):

- **Agent tool** (tool name `Agent`, input/output typed in doc
  `agent-sdk/typescript#agent`): spawns a subagent. Foreground subagents block
  the parent turn; background ones run concurrently. Under `-p` fork mode is
  off: Claude backgrounds by default and foregrounds when it needs the result;
  `CLAUDE_CODE_DISABLE_BACKGROUND_TASKS=1` forces foreground (doc
  `sub-agents#run-subagents-in-foreground-or-background`).
- **Nesting**: up to 3 layers by default (`CLAUDE_CODE_MAX_SUBAGENT_SPAWN_DEPTH`).
  Under `-p` a launcher does not wait for its nested background subagents,
  which then report to the main conversation (doc
  `sub-agents#let-subagents-spawn-their-own-subagents`).
- **Concurrency**: 20 running subagents per session by default
  (`CLAUDE_CODE_MAX_CONCURRENT_SUBAGENTS`, 2.1.217+); workflow agents and team
  members have their own limits (doc `sub-agents#concurrent-subagent-limit`).
- **Dynamic workflows** (brief): a JavaScript script Claude writes, run by the
  `Workflow` tool in the background: up to 16 concurrent agents by default
  (`CLAUDE_CODE_WORKFLOW_MAX_CONCURRENT_AGENTS`, 1-256), 1,000 agents per run,
  no mid-run user input except agent permission prompts and usage-limit waits;
  phases via `phase()` and `meta.phases`; resumable within the session (doc
  `workflows#behavior-and-limits`). The tool returns at once with
  `{status: "async_launched", taskId, taskType: "local_workflow", runId,
  scriptPath, transcriptDir, error?}`; the result arrives later as a task
  completion (doc `agent-sdk/typescript#workflow`, output types). The
  interactive `/workflows` progress view (phases, agent counts, per-agent
  tool calls) has no documented stream equivalent (probe).
- **Agent teams**: experimental, `CLAUDE_CODE_EXPERIMENTAL_AGENT_TEAMS=1`,
  interactive sessions only; under `-p` named subagents run as ordinary
  subagents (doc `agent-teams#enable-agent-teams`). Not available to the
  dashboard's session type.

How a program sees each subagent in the stream (documented unless marked):

| Signal | Fields | Source |
| - | - | - |
| `assistant` / `user` messages of a subagent | `parent_tool_use_id` = the spawning Agent `tool_use` id; `agent_id` (= task id, stable across resumes). Default: only `tool_use` / `tool_result` blocks; with `--forward-subagent-text` (2.1.211+) also text and thinking, at every depth (2.1.219+) | doc `headless#follow-subagent-messages`; `agent-sdk/typescript#sdkassistantmessage` |
| `system/task_started` | `task_id`, `run_id`, `tool_use_id`, `description`, `task_type` (`local_agent`, `local_bash`, `remote_agent`; workflows `local_workflow`), `is_backgrounded`, `spawn_depth`, `parent_task_id`, `ambient` | doc `agent-sdk/typescript#sdktaskstartedmessage`; `local_workflow` in PY `_internal/query.py:52` and doc `#workflow` output |
| `system/task_progress` | `usage {total_tokens, tool_uses, duration_ms}`, `last_tool_name`, `subagent_type`, `summary` (only with the TS option `agentProgressSummaries`; its CLI form is undocumented) | doc `#sdktaskprogressmessage` |
| `system/task_updated` | `patch {status: pending/running/completed/failed/killed, end_time, error, is_backgrounded, ...}` | doc `#sdktaskupdatedmessage` |
| `system/task_notification` | `status` completed/failed/stopped, `summary`, `output_file`, `usage` | doc `#sdktasknotificationmessage` |
| `system/background_tasks_changed` | full live set `tasks[] {task_id, run_id, task_type, subagent_type, description, parent_task_id, ambient}`; replace, do not merge; snapshot after a repeated `initialize` (2.1.203+) | doc `#sdkbackgroundtaskschangedmessage` |
| `tool_progress` | per Agent call: `subagent_type`, `subagent_retry {attempt, max_retries, retry_delay_ms, error_category}`; 30 s heartbeats for main-thread tools | doc `#sdktoolprogressmessage` |
| `can_use_tool` | `agent_id` when the request comes from a subagent | PY `types.py:2800-2812`; doc `#canusetool` (`agentID`) |
| Hooks `SubagentStart` / `SubagentStop` | `agent_id`, `agent_type`; Stop adds `agent_transcript_path`, `last_assistant_message`, `background_tasks` | doc `hooks#subagentstart`, `hooks#subagentstop` |
| Hooks `PreToolUse` / `PostToolUse` / `PostToolUseFailure` / `PermissionRequest` inside a subagent | carry `agent_id`, `agent_type`; the only reliable attribution when parallel subagents interleave | doc `hooks#common-input-fields`; PY `types.py:329-346` |
| Subagent transcripts | `~/.claude/projects/<project>/<session>/subagents/agent-<agentId>.jsonl`, format internal | doc `sub-agents#resume-subagents`, doc `claude-directory` |

Attribution rule: match messages to tasks on `agent_id`, not on
`parent_tool_use_id` vs `tool_use_id`, which diverge after a resume (doc
`agent-sdk/typescript#sdkassistantmessage`). Ignore `ambient: true` tasks in
activity counts (doc `#sdktaskstartedmessage`).

## 6. Observation points

### Stream messages (stdout of A/B; documented in doc `agent-sdk/typescript#sdkmessage`)

| `type` / `subtype` | Key fields | Use |
| - | - | - |
| `system/init` | `session_id`, `model`, `cwd`, `tools`, `mcp_servers`, `permissionMode`, `slash_commands`, `agents`, `skills`, `plugins`, `claude_code_version`, `capabilities` | Session header, zone A orchestrator model |
| `stream_event` | `event` (Anthropic raw stream event), `ttft_ms`; also `api_message_id`, **undocumented** (not in `SDKPartialAssistantMessage`, code.claude.com/docs/en/agent-sdk/typescript, read 2026-10-20), seen on every `stream_event` line of P1 to P303e (1,237 lines in 13 recordings) and equal to the id of the same agent's last `message_start` | Token deltas, zone B: the dashboard takes a delta's message id from its agent's last `message_start` (`event.message.id`) first, and from `api_message_id` only when no `message_start` came before (plan 21, item 3) |
| `assistant` | `message` (id, model, content, usage, stop_reason), `parent_tool_use_id`, `agent_id`, `error`, `aborted`, `usage_report`, `context_usage` | Completed blocks, tool calls, tokens |
| `user` | tool results (`tool_use_result` structured output), replays with `--replay-user-messages` | Tool completion |
| `result` | `subtype` (`success`, `error_max_turns`, `error_during_execution`, `error_max_budget_usd`, ...), `is_error`, `duration_ms`, `duration_api_ms`, `num_turns`, `total_cost_usd`, `usage`, `modelUsage`, `permission_denials`, `terminal_reason` | Turn end, cost, errors |
| `system/status` | `status: "compacting"`, `permissionMode` | Status chip |
| `system/compact_boundary` | `trigger`, `pre_tokens` | Context reset marker |
| `system/api_retry` | `attempt`, `max_retries`, `retry_delay_ms`, `error_status`, `error` | Retry counter (doc `headless#handle-api-retries`) |
| `system/informational` | `content`, `level` | Notices |
| `system/permission_denied` | `tool_name`, `tool_use_id`, `agent_id`, `decision_reason`, `decision_reason_type` (e.g. `"classifier"`, P13c), `message` | Denials |
| `system/permission_check_status` | `tool_use_id`, `status` (`"checking"`, `"done"`); undocumented, P13c | A tool's permission check, before its verdict |
| `system/task_*`, `system/background_tasks_changed`, `tool_progress` | section 5 | Zone C |
| `system/hook_started`, `hook_progress`, `hook_response` | hook name, output (with `--include-hook-events`) | Hook activity |
| `system/session_state_changed` | `state`: `idle` / `running` / `requires_action` (needs `CLAUDE_CODE_EMIT_SESSION_STATE_EVENTS=1`) | Orchestrator state |
| `system/thinking_tokens` | `estimated_tokens`, `estimated_tokens_delta`; documented as `SDKThinkingTokensMessage`, Claude Code 2.1.153 or later (read 2026-10-21) | Thinking progress: a running estimate (section 12) |
| `rate_limit_event` | `rate_limit_info {status, resetsAt, utilization, rateLimitType, overageStatus, ...}` | Zone D quota |
| `tool_use_summary`, `prompt_suggestion`, `conversation_reset`, `files_persisted`, `local_command_output`, `commands_changed`, `auth_status`, `worker_shutting_down`, `plugin_install` | as named | Secondary |

Forward compatibility: the Python SDK skips unknown `type` values instead of
failing (PY `_internal/message_parser.py:391-394`); the Go parser should too.

### Hooks (documented, doc `hooks#hook-events`)

Events: `SessionStart`, `Setup`, `InstructionsLoaded`, `UserPromptSubmit`,
`UserPromptExpansion`, `MessageDisplay`, `PreToolUse`, `PermissionRequest`,
`PostToolUse`, `PostToolUseFailure`, `PostToolBatch`, `PermissionDenied`,
`Notification`, `SubagentStart`, `SubagentStop`, `TaskCreated`,
`TaskCompleted`, `Stop`, `StopFailure`, `TeammateIdle`, `ConfigChange`,
`CwdChanged`, `DirectoryAdded`, `FileChanged`, `WorktreeCreate`,
`WorktreeRemove`, `PreCompact`, `PostCompact`, `PreModelSwitch`,
`PostModelSwitch`, `SessionEnd`, `Elicitation`, `ElicitationResult`.
Handler types: `command`, `http`, `mcp_tool`, `prompt`, `agent` (doc
`hooks#hook-handler-fields`).

Common input fields: `session_id`, `prompt_id`, `transcript_path`, `cwd`,
`scratchpad_dir`, `permission_mode`, `effort.level`, `hook_event_name`, plus
`agent_id` / `agent_type` inside subagents (doc `hooks#common-input-fields`).
Event-specific: `PreToolUse` `tool_name`, `tool_input`, `tool_use_id`;
`PostToolUse` adds `tool_response`, `duration_ms`; `PostToolUseFailure`
`error`, `is_interrupt`; `Notification` `message`, `title`,
`notification_type` (`permission_prompt`, `idle_prompt`, `auth_success`,
`elicitation_*`, `agent_needs_input`, `agent_completed`,
`quota_auto_resume_*`); `Stop` `stop_hook_active`, `last_assistant_message`,
`background_tasks`, `session_crons`; `SubagentStart`/`SubagentStop` as in
section 5 (doc `hooks`, each event's section; PY `types.py:305-497`).

Three ways for the dashboard to receive hooks without editing any settings
file:

1. **Control protocol** (undocumented at CLI level): declare hook matchers in
   `initialize.hooks` with `hookCallbackIds`; the CLI then sends
   `hook_callback` control requests and waits for the answer (`{}` to continue)
   (PY `_internal/query.py:319-338`, `:640-654`). Documented only as the SDK
   `hooks` option (doc `agent-sdk/hooks`).
2. `--settings '{"hooks": {...}}'` with an `http` hook to a local port:
   documented that `--settings` accepts any user-settings key for one session
   (doc `settings#change-a-setting-for-one-session`); that hooks work from this
   layer is inferred (probe). Hooks merge across levels, so the owner's own
   hooks still run (doc `hooks#hook-locations`).
3. `--include-hook-events`: observe the owner's configured hooks running (doc
   `cli-reference#cli-flags`).

For the MVP the stream messages of section 5 are enough; hooks add only
per-tool attribution inside subagents (`agent_id` on Pre/PostToolUse). Option 1
is preferred if that is needed.

### Session transcript files

`~/.claude/projects/<project>/<session-id>.jsonl` (project = working directory
with non-alphanumerics replaced by `-`, truncated to 200 characters plus a
hash), subagents under `<session-id>/subagents/`. Path documented; **entry
format explicitly internal and unstable** (doc
`sessions#where-transcripts-are-stored`). Path logic also in PY
`_internal/sessions.py:43`, `:104-141` (honours `CLAUDE_CONFIG_DIR`). Use only
as a last-resort fallback, never as the primary feed.

### Status line JSON (documented, doc `statusline#available-data`)

Fields on stdin of the status line command: `model.id`,
`model.display_name`, `cwd`, `workspace.{current_dir, project_dir,
added_dirs, git_worktree, repo.{host, owner, name}}`, `cost.{total_cost_usd,
total_duration_ms, total_api_duration_ms, total_lines_added,
total_lines_removed}`, `context_window.{total_input_tokens,
total_output_tokens, context_window_size, used_percentage,
remaining_percentage, current_usage.{input_tokens, output_tokens,
cache_creation_input_tokens, cache_read_input_tokens}}`,
`exceeds_200k_tokens`, `fast_mode`, `effort.level`, `thinking.enabled`,
`rate_limits.five_hour.{used_percentage, resets_at}`,
`rate_limits.seven_day.{used_percentage, resets_at}`,
`rate_limits.spend_limit.{used_percentage, resets_at, used_usd, limit_usd,
period}`, `prompt_cache.{warm, caching_observed, ttl, expires_at, requests,
misses, expected_rebuilds, hit_ratio, cache_write_tokens,
miss_recache_tokens, last_miss_at, last_miss_cause, miss_causes,
recache_tokens_if_cold}`, `session_id`, `session_name`, `prompt_id`,
`transcript_path`, `version`, `output_style.name`, `vim.mode`, `agent.name`,
`pr.{number, url, review_state, kind}`, `worktree.{name, path, branch,
original_cwd, original_branch}`.

- `rate_limits` appears only for claude.ai Pro/Max subscribers (or behind a
  gateway with a spend limit), only after the first API response, each window
  independently absent, dropped once `resets_at` passes; `used_percentage` is
  0-100, `resets_at` Unix seconds (doc `statusline#available-data`, "Fields
  that may be absent").
- Updates: session start, each new assistant message, `/compact`, mode change,
  `refreshInterval`, window reset; debounced 300 ms (doc
  `statusline#when-the-status-line-updates`).
- `subagentStatusLine` gives one row per subagent: `id`, `name`, `type`,
  `agentType`, `status`, `description`, `label`, `startTime`, `model`,
  `effort`, `contextWindowSize`, `tokenCount`, `tokenSamples`, `cwd` (doc
  `statusline#task-fields`).
- On Windows the command runs through Git Bash if installed, else PowerShell
  (doc `statusline#windows-configuration`).
- Limitation for the dashboard: it is a feature of the interactive TUI and
  needs a `statusLine` setting. The dashboard runs `-p`, so (inferred) no
  status line runs; adding one through `--settings` would be per-session and
  write nothing, but is useful only if a TUI session runs. It remains the
  documented source the brief names, and its field names are the model for
  zone D.

### OpenTelemetry (documented, doc `monitoring-usage`)

Enabled per process by environment: `CLAUDE_CODE_ENABLE_TELEMETRY=1`,
`OTEL_METRICS_EXPORTER`, `OTEL_LOGS_EXPORTER` (`otlp`, `console`,
`prometheus` for metrics), `OTEL_EXPORTER_OTLP_PROTOCOL`/`_ENDPOINT`. Nothing
is written to configuration.

- Metrics: `claude_code.session.count`, `claude_code.lines_of_code.count`,
  `claude_code.pull_request.count`, `claude_code.commit.count`,
  `claude_code.cost.usage`, `claude_code.token.usage`,
  `claude_code.code_edit_tool.decision`, `claude_code.active_time.total`.
- Events: `claude_code.user_prompt`, `assistant_response`, `tool_result`,
  `api_request`, `api_error`, `api_refusal`, `api_request_body`,
  `api_response_body`, `tool_decision`, `permission_mode_changed`, `auth`,
  `mcp_server_connection`, `internal_error`, `plugin_installed`,
  `plugin_loaded`, `skill_activated`, `at_mention`, `api_retries_exhausted`,
  `hook_registered`, `hook_execution_start`, `hook_execution_complete`,
  `hook_plugin_metrics`, `compaction`, `subagent_completed`,
  `feedback_survey`, `retention_sweep`, `managed_settings_resolved` (each
  prefixed `claude_code.`).
- Traces (beta): spans `claude_code.interaction`, `claude_code.llm_request`,
  `claude_code.tool`, `claude_code.tool.execution`,
  `claude_code.tool.blocked_on_user`, `claude_code.hook`.
- No quota or rate-limit metric. Receiving it would need a local OTLP
  receiver in the Go app; it duplicates the stream. Not needed for the MVP.

## 7. Telemetry for zone D

Subscription usage and API-equivalent cost are separate rows: the first comes
from Anthropic's servers, the second is a client-side estimate at list price
(doc `costs#using-the-usage-command`; doc `agent-sdk/cost-tracking`).

| Value | Source | Status | Notes |
| - | - | - | - |
| 5-hour quota | (1) `rate_limit_event.rate_limit_info` with `rateLimitType: "five_hour"`, `utilization` 0-1, `status` allowed/allowed_warning/rejected; (2) `/usage` sent as a prompt: `assistant.usage_report.rate_limits.limits[]` rows `{kind: "session", percent, resets_at, severity, is_active}`; (3) status line `rate_limits.five_hour.used_percentage` (TUI only) | (1) documented: doc `agent-sdk/python` (`RateLimitInfo`, `rate_limit_type`), doc `agent-sdk/typescript#sdkratelimitevent` (without the type field); (2) documented, experimental: doc `agent-sdk/typescript#sdkusagereport`; (3) doc `statusline#available-data` | (1) emitted "whenever the rate limit status transitions" (PY `types.py:1780-1788` docstring), so it may not arrive every turn; (2) needs a claude.ai credential with a known plan or `user:profile` scope and no usage-based billing; the usage endpoint can itself be rate-limited (doc `costs#when-the-usage-request-fails`). Show "N/A" until one arrives |
| 7-day quota | Same three sources: `rateLimitType` `seven_day`, `seven_day_opus`, `seven_day_sonnet`; `usage_report` rows `weekly_all`, `weekly_scoped` (with `scope.model`); status line `rate_limits.seven_day` | As above | Classify rows on `kind`, never on labels (doc `#sdkusagereport`) |
| Reset times | `rate_limit_info.resetsAt` (Unix s); `usage_report ... resets_at` (ISO 8601); status line `resets_at` (Unix s) | Documented | Drop a window after its reset passes, as Claude Code does |
| Overage / usage credits | `rate_limit_info.overageStatus`, `overageResetsAt`, `overageDisabledReason`; `usage_report.rate_limits.extra_usage` | Documented (Python ref; TS `#sdkusagereport`) | Keep separate from plan quota |
| Model | `system/init.model`; `assistant.message.model` (also per forwarded subagent message); `result.modelUsage` keys; `set_model` changes it | Documented | Subagent model: `message.model` of its forwarded messages, else "N/A" |
| Context window usage | `get_context_usage` control request -> `{totalTokens, maxTokens, rawMaxTokens, percentage, model, categories, ...}`; or compute from the last main-thread `assistant.message.usage` (input + cache creation + cache read) over `modelUsage[model].contextWindow` | Request: undocumented at CLI level (PY `_internal/query.py:788-790`, `types.py:1024-1060`), documented as SDK `getContextUsage()` (doc `#sdkcontrolgetcontextusageresponse`). Computation: inferred from documented fields | `get_context_usage` issues token-counting API requests (not billed on the Anthropic API) (doc `#sdkcontrolgetcontextusageresponse`); call it on demand, not per frame |
| Tokens | Per step: `assistant.message.usage`, deduplicated by `message.id`; per turn: `result.usage` (main loop only); running total incl. subagents: `result.modelUsage[*]` (`inputTokens`, `outputTokens`, `cacheReadInputTokens`, `cacheCreationInputTokens`, `thinkingTokens`); per subagent: `task_progress.usage.total_tokens` | Documented: doc `agent-sdk/cost-tracking`, doc `agent-sdk/typescript#modelusage` | In streaming input mode read the latest `result` for running totals, do not sum (doc `agent-sdk/cost-tracking`) |
| API-equivalent cost | `result.total_cost_usd` (running total for the process), `modelUsage[*].costUSD`, `costBasis` | Documented | Client-side estimate at list price unless `modelPricing` is managed; label "API-equivalent estimate", never "spent" |
| Active agent count | `background_tasks_changed.tasks` (non-ambient, `task_type` `local_agent`/`local_workflow`) plus foreground `task_started` without a terminal `task_updated`/`task_notification` | Documented | Reset the set when the CLI process (re)starts (doc `#sdkbackgroundtaskschangedmessage`) |
| Concurrency limit | Not reported by the CLI. Documented defaults: 20 subagents (`CLAUDE_CODE_MAX_CONCURRENT_SUBAGENTS`), 16 workflow agents or fewer on fewer CPUs (`CLAUDE_CODE_WORKFLOW_MAX_CONCURRENT_AGENTS`) | Documented defaults: doc `sub-agents#concurrent-subagent-limit`, doc `workflows#behavior-and-limits` | Show a value only if the dashboard sets the variable on the child; otherwise "N/A" (a managed setting or the CPU count can change the effective value) |
| Duration | `result.duration_ms`, `duration_api_ms`; `task_*.usage.duration_ms`; `tool_progress.elapsed_time_seconds` | Documented | Session wall time: measured by the dashboard from process start (observed, labelled so) |
| Errors | `result.is_error`, `subtype`, `errors`, `api_error_status`; `assistant.error` (`rate_limit`, `overloaded`, `authentication_failed`, ...); `system/permission_denied`; `tool_result.is_error`; `task_notification.status: failed` | Documented | Count per category |
| Retries | `system/api_retry` (main loop); `tool_progress.subagent_retry` (subagents) | Documented: doc `headless#handle-api-retries`, doc `#sdktoolprogressmessage` | Clear subagent retry indicator per `parent_tool_use_id` |

## 8. Platform notes

| Platform | Notes | Source |
| - | - | - |
| Windows native | `claude.exe` (native installer). Bash tool needs Git for Windows; otherwise the PowerShell tool is used; `CLAUDE_CODE_GIT_BASH_PATH` points to `bash.exe` when it is not on PATH (the case here) and can be passed in the child's environment | doc `setup#install-on-native-windows`, doc `env-vars` |
| | No Bash sandbox | doc `sandboxing` |
| | No POSIX signals to a child: interrupt through the `interrupt` control request; end through closing stdin, then `TerminateProcess` (the SDK maps SIGTERM to terminate, PY `subprocess_cli.py:61-68`, `:1029-1046`). Hard termination leaves the turn unfinished (doc `headless#stop-a-run-with-sigterm`) | inferred from those sources |
| | Never spawn `claude.cmd` with arguments (section 2) | PY `subprocess_cli.py:415-448` |
| | Fixed in recent releases: blank CRLF lines on stream-json input killed the session; unreadable stdin crashed print mode; multibyte UTF-8 split across chunks became U+FFFD | CC `CHANGELOG.md:4169`, `:4091`, `:6324` |
| WSL2 | Install inside the distribution (`install.sh`); sandbox available (`bubblewrap`, `socat`); separate `~/.claude` and login from Windows. A Windows-native dashboard would spawn `wsl.exe` around `claude` and translate paths (`C:\` to `/mnt/c/`); pipe behaviour through `wsl.exe` is unverified | doc `setup#install-in-wsl`, doc `sandboxing#set-up-linux-and-wsl2`; `wsl.exe` path inferred |
| Linux | `~/.local/bin/claude`; sandbox with `bubblewrap` + `socat`; SIGINT ends the turn, SIGTERM exits 143 | doc `setup`, doc `sandboxing`, doc `headless#stop-a-run-with-sigterm` |
| macOS | `~/.local/bin/claude` or Homebrew; sandbox with Seatbelt; signals as Linux | doc `setup`, doc `sandboxing#get-started` |
| All | Transcripts and configuration under `~/.claude` (or `CLAUDE_CONFIG_DIR`); `--tools` default set omits `Glob`/`Grep` on macOS, Linux and WSL | doc `env-vars`, doc `cli-reference#cli-flags` (`--tools`) |

## 9. Answers to the amendment 2 research questions

1. **Embed or control through supported interfaces?** Yes. The documented
   path is `claude -p` with stream-json input and output (doc `headless`),
   which is what both Agent SDKs and the VS Code extension use (PY
   `subprocess_cli.py:570-795`; CC `CHANGELOG.md:1650`). The message stream
   is documented; the control requests (approvals, interrupt, initialize) are
   documented as SDK methods, and their wire form only in SDK code.
   Limitation: undocumented wire details can change. Fallback: B (one process
   per turn) or C (SDK sidecar).
2. **Genuine streaming?** A and B with `--include-partial-messages`:
   token-level `stream_event` deltas for the orchestrator (doc
   `headless#stream-responses`). Subagents stream only as complete messages
   (doc `#sdkpartialassistantmessage`). The interactive TUI streams only to a
   screen.
3. **Subagent lifecycle and tool activity?** `task_started` / `task_progress`
   / `task_updated` / `task_notification` / `background_tasks_changed`,
   subagent `tool_use`/`tool_result` keyed by `parent_tool_use_id` and
   `agent_id`, `--forward-subagent-text` for their text (doc
   `headless#follow-subagent-messages`, doc `agent-sdk/typescript`). Hooks
   add per-tool `agent_id`. Limitation: workflow phase detail is not
   documented in the stream. Fallback: SubagentStart/Stop hooks via the
   control protocol; subagent transcript files (internal format).
4. **Approval prompts?** `--permission-prompt-tool stdio` plus `can_use_tool`
   control requests, answered allow (optionally edited input) or deny;
   `AskUserQuestion` through the same path (PY `types.py:2281-2304`, doc
   `agent-sdk/user-input`). Native rules, modes and sandbox still decide
   first; only what would prompt reaches the dashboard. Limitation: the
   `stdio` value is undocumented. Fallback: an MCP prompt tool (D,
   documented).
5. **Persistence, cancellation, resumption?** Claude Code persists each
   session as JSONL itself; `--resume <id>` / `--continue` /
   `--session-id` (doc `headless#continue-conversations`). Cancel: `interrupt`
   control request with receipt, `stop_task` for background tasks; SIGTERM
   leaves the turn unfinished (doc `headless#stop-a-run-with-sigterm`).
   Limitation: a resumed process does not replay history on stdout (P4,
   section 10). Fallback: TS SDK `getSessionMessages()` reads transcripts (doc
   `agent-sdk/typescript#getsessionmessages`), i.e. the internal format.
6. **Reuse the existing TUI?** No for the main path: no structured output,
   input and clipboard conflicts, ConPTY/tmux constraints (section 4E).
   Keep "open in Claude Code" (`claude --resume <id>`) as an escape hatch.
7. **Concurrent subagent output?** Every event carries `agent_id` /
   `task_id` / `parent_tool_use_id`; route each to a per-agent buffer and
   render panels independently; the stream is one ordered NDJSON pipe, so no
   locking across agents is needed beyond the reader goroutine. Parallel
   subagents from one turn share the Agent call's `tool_use_id`, so key on
   `parent_tool_use_id`/`agent_id` (doc `#sdktoolprogressmessage`).
8. **Reliable quota?** Two documented sources in the stream:
   `rate_limit_event` (passive, on transitions) and `/usage` ->
   `usage_report` (on demand, experimental, subscription credentials only).
   The status line JSON has the same data but only in the TUI. Neither is
   guaranteed every turn: show the age of the last value and "N/A" before the
   first. Never estimate quota from tokens.
9. **OS limits?** Native Windows: no sandbox, no POSIX signals (use control
   interrupt), no `.cmd` spawning, Git Bash for the Bash tool. WSL2: separate
   install and login, path translation. Linux/macOS: sandbox available,
   signals work. Details in section 8.

## 10. Open questions after the probes

The probes of 2026-10-15 (`doc/research/probes/README.md`, P1 to P4 and P3b)
settled some of the questions. Plan 14 (2026-10-18) gave each remaining one an
outcome: settled with evidence, or closed because the dashboard does not depend
on the answer. No question is left open. Each probe runs in a temporary folder
outside the repository, changes no configuration, and uses the owner's quota
(AGENTS.md).

Settled:

1. **Baseline sequence.** `system/init`, `system/status`,
   `rate_limit_event`, `stream_event` deltas, the complete `assistant`
   message, `result` (P1 with `-p`; P2 without `-p`, as the SDK runs it,
   same event types plus `control_response`).
2. **Control protocol from a non-SDK client.** `initialize` first,
   then user messages, works; `can_use_tool` arrives with `tool_name`,
   `display_name`, `input`, `description`, `permission_suggestions`,
   `tool_use_id`, and an allow that echoes `updatedInput` runs the tool (P2,
   P3b). The two variants not tried (no `initialize`, an allow without
   `updatedInput`) are closed as not depended on: the adapter always sends
   `initialize` first and always echoes `updatedInput`
   (`internal/provider/claude/session.go`).
3. **AskUserQuestion round trip** (plan 14, P14a, Claude Code 2.1.302, Haiku).
   The request is a `can_use_tool` with `tool_name:"AskUserQuestion"`,
   `input.questions[]` (each `question`, `header`, `options[]` of `label` and
   `description`, `multiSelect`), a UUID `request_id`, and a new field
   `requires_user_interaction:true`. The answer is an allow whose
   `updatedInput` keeps `questions` unchanged and adds `answers`, keyed by the
   question text; a multiSelect answer is its labels joined by ", ". The tool
   result is not an error and reads `Your questions have been answered:
   "Which colour?"="Rouge café ✓", "Which days?"="Monday, Tuesday". You can now
   continue with these answers in mind.`; the `user` line's `tool_use_result`
   holds `{questions, answers}`, and the model's next text used both answers
   [P:p14a-claude-question-answer.jsonl]. A deny still works as plan 5 found
   (L9): the model asks in text instead.
4. **Interrupt.** During streaming text: receipt `{"still_queued":[]}`, a user
   message "[Request interrupted by user]", `result` subtype
   `error_during_execution` (P2). During a running Bash command: the turn
   ended `error_during_execution` 3.3 s after the interrupt and no `PING.EXE`
   was left running (plan 4, L1 (c), `TestLiveBashInterrupt`). With a
   permission prompt pending: `error_during_execution` 66 ms later, the
   pending write never made (plan 4, L1 (d)).
6. **Subagent events.** With one backgrounded subagent
   (`is_backgrounded: true`, `spawn_depth: 1`): `task_started`,
   `task_progress`, `task_updated`, `task_notification`,
   `background_tasks_changed`; no `tool_progress`; the subagent's
   `can_use_tool` carries its `agent_id` (P3, P3b). The host must keep stdin
   open while background tasks run (P3). Parallel subagents: four parallel
   Explore subagents in plan 4 and two in plan 5's L1, each its own row in
   zone C. The volume without `--forward-subagent-text` is closed as not
   depended on: the adapter always passes that flag.
8. **Rate-limit events.** Not every turn: 2 events for 3 turns in P2, 1 for
   2 turns in P3b, 2 for P1's single turn. `unifiedWindows.five_hour` and
   `.seven_day` both appear, with `utilization` and `resetsAt`.
10. **Context usage.** `get_context_usage` from a raw client returns
    `totalTokens`, `maxTokens`, `percentage`, `model`, `categories`,
    `autoCompactThreshold` (P2). It costs no model call: in P14a it was
    answered 914 ms after it was sent, with no `assistant` message, `result`
    or `rate_limit_event` after it, only `system/commands_changed`
    [P:p14a-claude-question-answer.jsonl].
11. **Configuration safety.** `set_permission_mode` writes no settings file
    (P12c-switch, settings files identical, section 11). `set_model` is closed
    as not depended on: the dashboard never sends it (model configuration is
    outside the MVP, amendment 4).
12. **Resume.** History is not re-emitted: a `--resume` run outputs only
    `system/init`, `assistant`, `rate_limit_event`, `result` (P4). The
    dashboard keeps its own record of the conversation, or reads the
    transcript (internal format) to show an earlier session.
13. **Windows process handling.** Stdin closed: exit code 0 in 1.4 s (plan 4,
    L1 (e)). A process killed by its job object mid-turn ended 41 ms after,
    exit code 1, and a `--resume` in a new process answered (L1 (g)). Ctrl+Q
    quits with exit code 0 in 1.152 s (plan 4, L2d). P3b's subagent ran Bash
    without `CLAUDE_CODE_GIT_BASH_PATH`. UTF-8 and CRLF through Go pipes:
    P14a carried "Rouge café ✓" both ways live, and `TestEchoUnicode`
    sends and reads back "café ✓ — 日本\r\nline 2\n" byte for byte.
15. **Partial messages and subagents.** `stream_event` never carried a
    non-null `parent_tool_use_id`: 0 of 230 across P1 to P3b.

Closed, the dashboard does not depend on the answer:

5. **Hook callbacks.** The adapter's `initialize` sends `"hooks":null`;
   subagents come from the stream's `task_*` events (item 6), so no hook is
   needed, and adding one to the owner's settings is forbidden without their
   request (AGENTS.md).
7. **Workflows under `-p`.** The dashboard starts none; if Claude Code runs
   one, its `local_workflow` task maps to the agent type "workflow" in zone C
   (`internal/provider/claude/mapper.go`), like any other task.
9. **`/usage` over stream-json.** The dashboard never sends it; zone D's quota
   comes from `rate_limit_event` (item 8) and its context from
   `get_context_usage` (item 10). A `/usage` the owner types is an ordinary
   message, shown as Claude Code answers it (plan 4, the owner's live run).
14. **Status line under `-p`.** Zone D reads the stream (items 8 and 10), not
    the status line, whether or not it runs.

## 11. Auto mode through stream-json (plan 12, 2026-10-18)

Probes P12c to P12c-block2 on Claude Code 2.1.301, the dashboard's arguments
with only `--permission-mode` changed (`probes/README.md`):

- **`auto` is accepted in `-p` stream-json mode:** `system/init` reads
  `permissionMode:"auto"` [P:p12c-claude-auto.jsonl]. No `can_use_tool`
  reached the client for a write in the project, a write in its parent
  folder, `rm -rf ../victim` (a dummy folder) or `curl ... | sh`
  [P:p12c-risk-claude-auto-rm.jsonl, P:p12c-risk2-claude-auto-curl-sh.jsonl].
- **Who decided** is on the `user` line that carries the tool result:
  `tool_result_meta[].permission_decision` (undocumented), one per
  `tool_use_id`. Seen: `{decision:"accept", source:"config",
  reason_type:"mode"}` (allowed by the mode, no classifier),
  `reason_type:"classifier"` (allowed by the classifier), and in default
  mode `{decision:"reject", source:"user_reject"}` for a denied
  `can_use_tool`. Earlier recordings also show `source:"user_temporary"`,
  `reason_type:"subcommandResults"` and `{decision:"reject",
  source:"config", reason_type:"other"}`. A tool use an interrupt rejected,
  while its approval was pending or while it ran, has no decision: its entry
  is `{id, non_execution_kind:"user-rejected"}`, and a pending `can_use_tool`
  is first withdrawn by `control_cancel_request`
  [P:p303e-claude-interrupt-tool.jsonl] (2.1.303; the same on 2.1.300 in the
  live gate).
- **Switching mid-session:** the control request `{subtype:
  "set_permission_mode", mode:"auto"}` is answered `{subtype:"success",
  response:{mode:"auto"}}` in 4 ms, followed by `system/status` with
  `permissionMode:"auto"`; the next `system/init` reads `auto` and the next
  write asked nothing [P:p12c-switch-claude-default-to-auto.jsonl]. The
  configuration check after the run found the settings files identical, so
  `set_permission_mode` writes no settings file (open question 11, for that
  request).
- **A classifier block, observed live** (plan 13, Claude Code 2.1.302, Haiku,
  the dashboard in Auto, 2026-10-18): asked to follow a `SETUP.md` whose
  steps the owner never authorised, the classifier denied a PowerShell call
  that scouted the exfil [P:p13c-claude-auto-classifier-deny.jsonl]. The
  reject is the mirror of the accepts above: the `user` line's tool result
  carries `tool_result_meta[].non_execution_kind:"automode-blocked"` with
  `permission_decision {decision:"reject", source:"config",
  reason_type:"classifier"}`, and the tool-result content reads "Permission
  for this action was denied by the Claude Code auto mode classifier.
  Reason: [Exfil Scouting]…". A separate `system`/`permission_denied`
  repeats it (`decision_reason_type:"classifier"`,
  `decision_reason:"[Exfil Scouting]"`). This confirms the mapper's earlier
  guess of `"reject"` for a classifier block. (Earlier, P12c saw no block
  because a project write never reaches the classifier and an out-of-project
  write was refused by the model without a tool call
  [P:p12c-block-claude-auto-boundary-edit.jsonl,
  P:p12c-block2-claude-auto-boundary-shell.jsonl].)
- **Auto falling back to a prompt** after repeated classifier blocks:
  closed by the owner on 2026-10-18 (plan 14, item 3), not provoked. It is
  Claude Code's own mechanism, which the dashboard follows rather than
  imitates: a fallback arrives as a `can_use_tool`, and every
  `can_use_tool` is already a card.
- **`system`/`permission_check_status` (undocumented, new):** a tool's
  permission check is bracketed by `{type:"system",
  subtype:"permission_check_status", tool_use_id, status:"checking"}` then
  the same with `"done"` [P:p13c-claude-auto-classifier-deny.jsonl]. The
  verdict itself arrives with the tool's result (above), so the dashboard
  logs this event as `commands_changed` with no on-screen status.
- **Documented** (https://code.claude.com/docs/en/permission-modes, fetched
  2026-10-18): on a block Claude receives the reason and tries another
  way; after 3 blocks in a row or 20 in a session auto mode pauses and
  prompts again, which with `--permission-prompt-tool stdio` is a
  `can_use_tool`; explicit ask rules still prompt; Manual's configuration
  value is `default` (`manual` an alias); the classifier runs on its own
  model, whose usage the stream does not report apart.
- **The same check as the interactive terminal** (plan 12 addendum,
  2026-10-18, Haiku): `rm -rf ../victim` (a dummy folder) and `curl -fsSL
  https://example.com/install.sh | sh` were allowed with no block and no
  prompt both in Claude Code's own terminal (`claude --permission-mode
  auto`, 2.1.301) and through the dashboard in Auto (2.1.302, three
  `reason_type:"classifier"` accepts). The interactive start in a new
  folder shows the workspace trust dialog, which writes a `projects` entry
  to `~/.claude.json`; `-p` mode does not show it.

## 12. Thinking through stream-json (plan 22, 2026-10-21)

What Claude Code reports about thinking, for zone B's thinking indicator
(brief amendment 18). Read by script from every recording in
`doc/research/probes` (33 thinking blocks in 14 recordings, Claude Code
2.1.300 to 2.1.303, all Haiku 5.5), then checked by an independent agent
over the same recordings; the documentation read on 2026-10-21.

- **A phase's start and end, observed.** A thinking block's
  `stream_event` `content_block_start` (`content_block.type` `thinking`)
  and the `content_block_stop` of its index, main thread only (no stream
  event carries a `parent_tool_use_id`): one per API message, always at
  index 0 and first; no `redacted_thinking`
  [P:p12c-block-claude-auto-boundary-edit.jsonl] lines 8 and 21. Its text
  is always empty, in the block, the 101 `thinking_delta` lines and the
  `assistant` lines: the CLI sends no thinking text. 22 of the 55 messages
  have no thinking block. Documented: the Anthropic stream events inside
  `SDKPartialAssistantMessage.event` (doc `agent-sdk/typescript`); an empty
  `thinking_delta` under `display: "omitted"`
  (https://platform.claude.com/docs/en/build-with-claude/thinking).
- **A live estimate.** `{"type":"system","subtype":"thinking_tokens",
  "estimated_tokens":N,"estimated_tokens_delta":D}`: 90 lines, each the
  previous plus its delta, starting at 50 in each block 0.15 to 0.4 s
  after its start, then a step about every 1 to 1.9 s, and a last uneven
  value at the block's stop when there is a remainder (21 of 27 blocks);
  none for 4 blocks under 0.3 s; no agent field, and never outside an open
  main-thread block. Documented (Agent SDK TypeScript reference,
  `SDKThinkingTokensMessage`, "a running estimate of the thinking tokens
  generated so far in the current block", Claude Code 2.1.153 or later,
  https://code.claude.com/docs/en/agent-sdk/typescript, read 2026-10-21).
  Against the exact count below, a block's last estimate runs from 14%
  under (1,050 for 1,218, [P:p12c-block2-claude-auto-boundary-shell.jsonl]
  lines 120 and 168) to nearly four times over (198 for 52,
  [P:p12c-risk-claude-auto-rm.jsonl] lines 93 and 128): short blocks over,
  long ones a little under. The dashboard shows it as an estimate.
- **Where the estimate comes from** (plan 22's investigation, section
  1.1). The CLI asks the API for `display: "updates"`, so each
  `thinking_delta` of a block carries an `estimated_tokens` increment, a
  multiple of 50: the first, 50, 146 to 392 ms after the block's start,
  then 100 to 350 tokens every 1.0 to 1.9 s. 0 to 4 ms before it forwards
  each delta, the CLI sends `system/thinking_tokens` with their running sum
  (69 of 69 lines: the sum of the increments, never lower). The
  increments are undocumented: the API's streaming page describes an
  omitted block as one empty `thinking_delta`, then its `signature_delta`
  (https://platform.claude.com/docs/en/build-with-claude/streaming, read
  2026-10-21).
- **The last value.** A closing `thinking_delta` carries
  `estimated_tokens` null. In 21 of the 32 blocks with one, a last
  `system/thinking_tokens` follows 0 to 2 ms before the `signature_delta`,
  its increment uneven (20 to 328 tokens), and it fits the block's exact
  count E closely: 153 + 0.78 × E (R² 0.993, standard error 17 tokens), as
  if computed from the whole block, likely its signature (inferred: the
  recordings scrub the signatures). It is missing from 6 blocks of 719
  tokens or more, from 4 under 50 tokens (which have no estimate at all),
  and from the interrupted one.
- **The exact count, per API message.** The `message_delta` event's
  `usage.output_tokens_details.thinking_tokens`, in 55 of 55 messages (0
  with no thinking), after the message's last block, 0 to 2.4 s (median
  0.6 s) after its thinking ended [P:p303b-claude-subagent-approval.jsonl]
  lines 82, 146 and 195: 719, 281, 268. One thinking block per message, so
  it is that phase's count. Documented ("When streaming, this breakdown
  appears only on the final `message_delta` event",
  https://platform.claude.com/docs/en/build-with-claude/extended-thinking).
- **The turn's total.** The `result` line's
  `usage.output_tokens_details.thinking_tokens`, the sum of the turn's
  main-thread messages' in 31 of 31 results ([P:p303b-claude-subagent-approval.jsonl]
  line 197: 1,268; its subagent's 180, line 104, not included).
  `modelUsage.<model>.thinkingTokens` is the session's, subagents and side
  calls included (line 197: 2,305): never a phase's.
- **Subagents.** With `--forward-subagent-text`, a subagent's thinking
  comes only in `assistant` lines with `parent_tool_use_id` set, empty, no
  stream event and no estimate ([P:p303b-claude-subagent-approval.jsonl]
  lines 84, 88, 93, 100); a foreground Agent call's result carries the
  subagent's usage with its thinking count (line 104).
- **An interrupted message.** On an interrupt the CLI closes the stream
  itself. Its `message_delta` has `stop_reason` null and the usage of its
  `message_start` (`output_tokens` 7, `thinking_tokens` 0 by default), not
  a count of the thinking, and the result's usage is zeroed
  ([P:p303a-claude-conversation.jsonl] lines 70 to 78; P2 line 76). The
  dashboard shows no exact count for such a phase (plan 22, correction
  C1).
- **Opus 5.5 (P22o, 2026-10-21) thinks as Haiku does.** One turn,
  `ANTHROPIC_MODEL=claude-opus-5-5`, with the dashboard's arguments, on
  Claude Code 2.1.303 ([P:p22o-claude-opus-thinking.jsonl]).
  - One main-thread thinking block of 8.0 s (2,796 to 10,807 ms).
    `"thinking_display":"updates"` is on 8 of its 10 lines, as on Haiku's
    (13 of 19 in P303b, 24 of 32 in P12c-block).
  - The increments of its `thinking_delta` lines are 50, 150, 150, 150, 200
    and 150, then null at its close. Six `system/thinking_tokens` give 50 to
    850: the first 508 ms into the block (Haiku: 146 to 392 ms), then every
    1.0 to 1.6 s, each step a multiple of 50.
  - No last, uneven value follows the close, as in Haiku's long blocks. The
    last estimate, 850, is 8.4% under the exact count.
  - The exact count, 928, is in the final `message_delta`, 967 ms after the
    block's stop, and in the result's usage.
  - Replayed through the adapter and the reducer, it is one observed phase
    with the estimate 850 and the count 928: "Thought for 8.0s · 928 thinking
    tokens".
- **Not thinking, from the same run:** the owner's `SessionStart` hook makes
  the CLI send `system/hook_started` and `system/hook_response`, and the turn
  ends with a `system/post_turn_summary`. No earlier recording has them, and
  the adapter counted all three as unrecognised. Since plan 22's amendment
  20 it knows them and draws nothing, except for a hook response that is not
  a success: a notice in zone B naming the hook, its exit code and the first
  line of its stderr. It is not an error, since the hook's lines come before
  the initialize response, and the CLI carries on.
- **Not used, undocumented:** the `thinking_display` field (`"updates"`) on
  `stream_event` lines and the `estimated_tokens` field of a
  `thinking_delta` (the Messages API's `ThinkingDelta` has `thinking` and
  `type` only). No thinking budget is reported. A resume replays no
  history (P4).
- **Not verified:** Sonnet (every recording but P22o runs Haiku, and P22o
  runs Opus); a resume of a session that thought, with partial messages; a
  subagent thinking while the orchestrator thinks.
