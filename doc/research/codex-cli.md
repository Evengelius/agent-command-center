# OpenAI Codex CLI

Plan 1, step 1 research note on driving OpenAI Codex CLI from the dashboard.
Installed version: **codex-cli 0.145.0** (probe P0, `p0-detect.txt`).
Probe evidence (P0, P5, P5b, P6, P6b, P6m) is in [`probes/`](probes/README.md).

## 1. Sources

Abbreviations used in every citation below:

| Tag | Source |
| --- | --- |
| **[T]** | `github.com/openai/codex` at tag `rust-v0.145.0`, commit `25af12f7e61572b0bc18ddb1008be543b91519b0` (2026-07-21): the installed version. Paths are repo-relative. |
| **[H]** | Same repo at `HEAD` `1ef30bcbe784f1006548adc1542985be32b05109` (2026-10-15): used only to show what changed after 0.145.0. |
| **[D:page]** | Official docs, accessed 2026-10-15. `https://developers.openai.com/codex/...` answers `308 Permanent Redirect` to `https://learn.chatgpt.com/docs/...`; the pages below are the redirect targets. **The site documents the current release, which is later than 0.145.0**; where it disagrees with [T], [T] wins for the installed binary. |
| **[P:file]** | Probe recordings in `doc/research/probes/` (2026-10-15, owner-approved; P7 on 2026-10-16 under the re-run rule; Codex 0.145.0 on Windows 11). |

Docs pages (all accessed 2026-10-15):
[D:cli] https://learn.chatgpt.com/docs/codex/cli ·
[D:noninteractive] https://learn.chatgpt.com/docs/non-interactive-mode ·
[D:app-server] https://learn.chatgpt.com/docs/app-server ·
[D:commands] https://learn.chatgpt.com/docs/developer-commands ·
[D:sdk] https://learn.chatgpt.com/docs/codex-sdk ·
[D:config-basic] https://learn.chatgpt.com/docs/config-file/config-basic ·
[D:config-advanced] https://learn.chatgpt.com/docs/config-file/config-advanced ·
[D:config-ref] https://learn.chatgpt.com/docs/config-file/config-reference ·
[D:hooks] https://learn.chatgpt.com/docs/hooks ·
[D:subagents] https://learn.chatgpt.com/docs/agent-configuration/subagents ·
[D:win-sandbox] https://learn.chatgpt.com/docs/windows/windows-sandbox ·
[D:wsl] https://learn.chatgpt.com/docs/windows/wsl ·
[D:sandboxing] https://learn.chatgpt.com/docs/sandboxing ·
[D:security] https://learn.chatgpt.com/docs/agent-approvals-security ·
[D:pricing] https://learn.chatgpt.com/docs/pricing ·
[D:changelog] https://learn.chatgpt.com/docs/changelog ·
[D:maturity] https://learn.chatgpt.com/docs/feature-maturity ·
[D:open-source] https://learn.chatgpt.com/docs/open-source.

In-repo docs at [T]: `codex-rs/app-server/README.md` (2398 lines, the
app-server protocol reference), `sdk/typescript/README.md`,
`sdk/python/README.md`, `codex-rs/docs/codex_mcp_interface.md`,
`docs/install.md`. The repo's `docs/exec.md`, `docs/sandbox.md` and
`docs/config.md` are only links to the site ([T] `docs/exec.md:3`,
`docs/sandbox.md:3`, `docs/config.md:3-7`).

**License: Apache-2.0** ([T] `LICENSE:1-3`, "Apache License Version 2.0,
January 2004"; a `NOTICE` file is present; `@openai/codex-sdk` declares
`"license": "Apache-2.0"`, [T] `sdk/typescript/package.json`). Permissive:
code may be copied with the notice kept (AGENTS.md rule). [D:open-source]
states the repository does not accept external contributions.

**Schema check.** The v2 JSON Schema that the installed binary generated
(`codex app-server generate-json-schema`, [P:p0-codex-app-server-schema-v2.json])
has the same 532 definitions as the fixture committed at [T]
(`codex-rs/app-server-protocol/schema/json/codex_app_server_protocol.v2.schemas.json`),
and they are identical once key order and `$ref` prefixes are normalised
(comparison made for this note). So the [T] source is the exact protocol of
the installed binary, and Go types can be generated from the committed
fixture without running `codex`.

## 2. Detection and version

| Item | Fact | Source |
| --- | --- | --- |
| On this machine | `codex.exe` in `%LOCALAPPDATA%\Programs\OpenAI\Codex\bin` (standalone installer), on the PATH | [P:p0-detect.txt], [P:README.md] "Versions" |
| Install channels | Windows standalone: `powershell -ExecutionPolicy ByPass -c "irm https://chatgpt.com/codex/install.ps1 \| iex"`; npm `npm install -g @openai/codex`; Homebrew `brew install --cask codex`; macOS/Linux `curl -fsSL https://chatgpt.com/codex/install.sh \| sh`; GitHub release binaries | [D:cli] "Quickstart"; [T] `README.md:14-40` |
| npm layout | package `@openai/codex`, `bin.codex = bin/codex.js` (a Node wrapper); it runs the native `vendor/<triple>/bin/codex.exe` from `@openai/codex-win32-x64` / `-win32-arm64` and forwards SIGINT/SIGTERM/SIGHUP | [T] `codex-cli/package.json:2-7`, `codex-cli/bin/codex.js:21-22,92,113-225` |
| PATH names to look for on Windows | `codex.exe` (standalone, or inside the npm package); `codex.cmd` / `codex.ps1` (shims npm creates for `bin/codex.js`; npm behaviour, not Codex docs); `codex` (DotSlash file in releases, [T] `docs/install.md`) | as cited |
| Version | `codex --version` / `-V` prints `codex-cli 0.145.0` | [P:p0-detect.txt]; [T] `codex-rs/cli/src/main.rs:95-97` (`#[clap(author, version)]`) |
| Version without a subprocess per call | `initialize` response `userAgent` = `"<clientInfo.name>/0.145.0 (Windows 10.0.26200; x86_64) ..."`; every `Thread` has `cliVersion: "0.145.0"` | [P:p6b-codex-app-server.jsonl] lines 3 and 9 |
| Other health checks | `codex login status` exits 0 when logged in; `codex doctor` (stable at [H] docs) | [D:commands] "login", "doctor"; [T] `codex-rs/cli/src/main.rs:163-164` |

Recommendation: resolve `codex.exe` itself (not the `.cmd` shim) so the
dashboard owns the real process and its signals; when only the npm shim is
found, resolve `vendor\x86_64-pc-windows-msvc\bin\codex.exe` the way
`codex.js` does ([T] `codex-cli/bin/codex.js:85-95`).

## 3. Working directory, permissions, configuration

### 3.1 Working directory

| Interface | How | Source |
| --- | --- | --- |
| Any command | `-C, --cd <DIR>`: "Tell the agent to use the specified directory as its working root"; `--add-dir <DIR>` adds writable roots | [T] `codex-rs/utils/cli/src/shared_options.rs:56-62` |
| `codex exec` | refuses to run outside a Git repository unless `--skip-git-repo-check` (exit 1, "Not inside a trusted directory and --skip-git-repo-check was not specified.") | [T] `codex-rs/exec/src/lib.rs:785-792`; [D:noninteractive] |
| app-server | `thread/start` / `turn/start` param `cwd`; otherwise the server process's own cwd | [T] `codex-rs/app-server/README.md:79,784`; [P:p6b] line 8 (`"cwd"` accepted, echoed in `thread.cwd`) |

**Configuration write to avoid.** `thread/start` with a `cwd`, on a project
with no trust level, and a write-capable sandbox (`workspace-write` or full
access) **writes `trust_level = "trusted"` for that project into the user
`config.toml`** ([T] `codex-rs/app-server/README.md:140`;
`codex-rs/app-server/src/request_processors/thread_processor.rs:1142-1170,5133-5160`).
That breaks the rule "never change an agent CLI's configuration". P6/P6b used
`sandbox: "read-only"`, which does not trigger it, and the config hash was
unchanged ([P:README.md] "Configuration check"). Ways out, to be confirmed by a
probe (section 10): start `codex app-server` with the project as the
**process** working directory and omit `cwd` from `thread/start`
(`requested_cwd.is_some()` is then false, `thread_processor.rs:1142`), or use
`read-only` until the owner trusts the project in Codex itself.

### 3.2 Sandbox modes and approval policies (0.145.0)

| Setting | Values at [T] | Notes |
| --- | --- | --- |
| `sandbox_mode` / `-s, --sandbox` | `read-only` (enum default), `workspace-write`, `danger-full-access` | [T] `codex-rs/protocol/src/config_types.rs:86-98`; docs: workspace-write is "the default low-friction mode for local work" [D:sandboxing] |
| `approval_policy` / `-a, --ask-for-approval` | `untrusted` (only known-safe read commands auto-run), `on-request` (default; `on-failure` is an accepted alias), `granular` (`sandbox_approval`, `rules`, `skill_approval`, `request_permissions`, `mcp_elicitations` booleans), `never` | [T] `codex-rs/protocol/src/protocol.rs:914-960`; `-a` accepts `untrusted`, `on-request`, `never`: [T] `codex-rs/utils/cli/src/approval_mode_cli_arg.rs:10-28`, `codex-rs/tui/src/cli.rs:60-62` |
| Docs drift | Current docs: "`untrusted` is unsupported, and `on-failure` is deprecated"; `-a on-request \| never` | [D:config-ref] `approval_policy`; [D:security]; [D:commands] "Global flags" |
| `approvalsReviewer` | `user` (default) or `auto_review` (a reviewer subagent decides) | [T] `codex-rs/app-server/README.md:773-776`; [D:sandboxing] |
| Bypass | `--dangerously-bypass-approvals-and-sandbox` (alias `--yolo`) | [T] `shared_options.rs:42-49` |
| `--full-auto` | removed; `codex exec --full-auto` only prints "use `--sandbox workspace-write` instead" | [T] `codex-rs/exec/src/cli.rs:42-50,103-106` |
| Exec mode | `codex exec` forces `approval_policy = never` (unless the reviewer is `auto_review`); every approval request is rejected | [T] `codex-rs/exec/src/lib.rs:422-428,1679-1789` |
| Network | off by default in the sandbox; `[sandbox_workspace_write] network_access = true` | [D:security] |
| Project trust | untrusted projects skip project `.codex/config.toml`, hooks and rules; trust is `projects.<path>.trust_level` in the user config | [D:config-basic]; [D:config-ref] |

Windows sandbox at [T]: `[windows] sandbox = "elevated" | "unelevated"`
(plus `sandbox_private_desktop`), else disabled
([T] `codex-rs/config/src/types.rs:160-172`;
`codex-rs/core/src/config/mod.rs:3174-3194,3253-3257`). A comment states that a
requested write sandbox can be "downgraded to ReadOnly (perhaps there is no
sandbox available on Windows ...)" ([T] `thread_processor.rs:1130-1134`). The
newer `mxc` mode and `features.prefer_mxc` exist only after 0.145.0 (absent at
[T], present at [H] `codex-rs/features/src/lib.rs`; [D:win-sandbox]).

### 3.3 Configuration loading (read only; per-invocation overrides)

- Files: `~/.codex/config.toml` (`CODEX_HOME`, default `~/.codex`), project
  `.codex/config.toml` when trusted, profile files
  `$CODEX_HOME/<name>.config.toml` via `-p, --profile` (0.145.0 help:
  "Layer $CODEX_HOME/<name>.config.toml on top of the base user config",
  [T] `shared_options.rs:33-35`), `requirements.toml` constraints. Precedence:
  CLI flags and `-c` > project > profile > user > cloud-managed > system >
  defaults ([D:config-basic] "precedence").
- **Per-invocation overrides, no file written:** `-c key=value` (dotted path,
  value parsed as TOML, else a string; [T]
  `codex-rs/utils/cli/src/config_override.rs:15-36`), `-m`, `-s`, `-a`, `-C`,
  `--add-dir`, `--enable/--disable <feature>`; `codex exec --ignore-user-config`
  and `--ignore-rules` ([T] `codex-rs/exec/src/cli.rs:34-40`); app-server
  `thread/start`/`turn/start` fields `model`, `cwd`, `approvalPolicy`,
  `sandbox`/`sandboxPolicy`, `effort`, `config` overrides
  ([T] `codex-rs/app-server/README.md:253-307,771-818`).
- **App-server methods that DO write config**, never to be called:
  `config/value/write`, `config/batchWrite`, `skills/config/write`,
  `marketplace/*`, `plugin/install|uninstall`, `externalAgentConfig/import`
  ([T] `codex-rs/app-server/README.md:216-249`), and the implicit trust write
  above. Read-only alternatives: `config/read`, `configRequirements/read`,
  `model/list`, `experimentalFeature/list` ([T] same lines).
- The owner's `config.toml` sets a model the ChatGPT account refuses, so every
  turn fails until a per-session model is passed (`-m gpt-5.6-sol`,
  `thread/start.model`) [P:p5-codex-exec.jsonl], [P:p6-codex-app-server.jsonl]
  lines 23-24, [P:p5b-codex-exec.jsonl], [P:p6b] line 8. The dashboard must
  show such failures in the conversation, never hide them.

## 4. Ways to drive a session from another program

| Interface | Transport | Documented | Stability (0.145.0) | Streaming | Multi-turn in one process | Approvals | Clarification questions | Interrupt | Resume |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| **`codex app-server`** | JSON-RPC 2.0 without the `"jsonrpc"` field, JSONL on stdio (default); `ws://` experimental/unsupported; `unix://` | yes: [D:app-server]; [T] `codex-rs/app-server/README.md` | `[experimental]` in `codex --help` ([P:p0-codex-help.txt] line 16; [T] `codex-rs/cli/src/main.rs:147-148`); "experimental and aren't supported for production workloads" [D:app-server]; stable vs `experimentalApi` surfaces | token deltas (`item/agentMessage/delta`), reasoning deltas, command output deltas, per-item lifecycle | yes, many threads and turns | yes: server-to-client requests for commands, file changes, permissions, MCP elicitations | `item/tool/requestUserInput` (experimental; Plan mode only by default); otherwise plain text in an agent message | `turn/interrupt`; `turn/steer` adds input mid-turn | `thread/resume`, `thread/fork`, `thread/list`, `thread/read` |
| `codex exec --json` (alias `--experimental-json`) | one process per turn, JSONL on stdout | yes: [D:noninteractive] | exec is Stable [D:commands] | items only on start/completion; **no text deltas** | no (one prompt per process) | none: policy forced to `never`, requests rejected | none (rejected) | kill the process | `codex exec resume <id>` / `--last` |
| TypeScript SDK `@openai/codex-sdk` | spawns `codex exec --experimental-json` per `run()` | yes: [D:sdk]; [T] `sdk/typescript/README.md:5` | no statement | as `exec --json` | per call (new process, `resume <threadId>`) | none | none | `AbortSignal` kills the child | `resumeThread(id)` |
| Python SDK `openai-codex` | spawns `codex app-server --listen stdio://` | yes: [D:sdk] ("controls the local Codex app-server over JSON-RPC") | "available as a stable release" [D:sdk] | as app-server | yes | as app-server | as app-server | as app-server | as app-server |
| `codex mcp-server` | MCP over stdio; tools `codex` (prompt, model, cwd, approval-policy, sandbox, config, ...) and `codex-reply`; events as `codex/event` notifications; approvals as MCP `elicitation/create` | in repo only: `codex-rs/docs/codex_mcp_interface.md:1-7` ("experimental and subject to change"); not on the site | **deprecated 2026-08-24, removed 2026-09-05** [D:changelog]; still present in 0.145.0 | coarse | yes (`codex-reply`) | via elicitation | no | MCP cancel | by thread id |
| TUI in a PTY | ConPTY / pty, VT screen | n/a | n/a | screen repaint only | yes | only by sending keystrokes | screen text | Esc/Ctrl-C keys | `codex resume` |
| `codex proto` | - | - | **absent** from 0.145.0's subcommand list | - | - | - | - | - | - |

Sources for the table: [T] `codex-rs/app-server/README.md:20-53,74-81,138-251,1379-1690`;
`codex-rs/exec/src/cli.rs:63-85,166-222`; `codex-rs/exec/src/exec_events.rs:8-37`;
`codex-rs/exec/src/event_processor_with_jsonl_output.rs:465-583`;
`sdk/typescript/src/exec.ts:87-181`; `sdk/python/src/openai_codex/client.py:213,252`;
`codex-rs/mcp-server/src/codex_tool_config.rs:27-62,237`,
`codex-rs/mcp-server/src/outgoing_message.rs:121`,
`codex-rs/mcp-server/src/exec_approval.rs:98`;
`codex-rs/cli/src/main.rs:125-210` (subcommand list, no `proto`).

### 4.1 `codex app-server` (recommended)

What it is: "the interface Codex uses to power rich interfaces such as the
Codex VS Code extension" ([T] `codex-rs/app-server/README.md:3`). In 0.145.0
the TUI itself and `codex exec` are clients of an in-process app-server
([T] `codex-rs/tui/src/app_server_session.rs`; `codex-rs/exec/src/lib.rs:16-21`),
and `codex mcp-server` was later removed in its favour [D:changelog]. Despite
the "experimental" label, it is the interface every first-party surface uses.

Handshake and flow, **settled by P6/P6b from a 100-line Node driver without
any SDK** ([P:p6b-codex-app-server.jsonl], [`drivers/codex-driver.mjs`](probes/drivers/codex-driver.mjs)):

1. `initialize` `{clientInfo:{name,title,version}}` → `{userAgent, codexHome, platformFamily:"windows", platformOs:"windows"}` (line 3), then the `initialized` notification (line 4). Other requests before it get "Not initialized" ([T] README:85).
2. `thread/start` `{cwd, sandbox:"read-only", approvalPolicy:"on-request", model:"gpt-5.6-sol"}` → `thread` (`id`, `sessionId`, `path` = the rollout file, `cliVersion`, `source:"vscode"`, `status`), plus effective `model`, `approvalPolicy`, `sandbox`, `reasoningEffort:"high"`, `multiAgentMode:"explicitRequestOnly"`, `instructionSources` (lines 8-9). `thread/started` follows (line 11).
3. `turn/start` `{threadId, input:[{type:"text",text}]}` → `turn` `inProgress` (lines 10, 14); then `thread/status/changed` `active`, `turn/started`, `item/started|completed` `userMessage`, `item/started` `agentMessage` (`phase:"commentary"`), `item/agentMessage/delta`×N, `item/completed` (lines 15-34).
4. File change: `item/started` `fileChange`, `thread/status/changed` `activeFlags:["waitingOnApproval"]`, **server request** `item/fileChange/requestApproval` `{threadId, turnId, itemId, startedAtMs, reason, grantRoot}`; client answers `{"decision":"accept"}`; `serverRequest/resolved`; `item/completed` `fileChange` with `changes[{path, kind:{type:"add"}, diff}]`, `turn/diff/updated` (lines 35-47).
5. Usage after each model call: `thread/tokenUsage/updated` then `account/rateLimits/updated` (lines 41-42, 48-49, 57-58, 64-65).
6. End: `thread/status/changed` `idle`, `turn/completed` `{status:"completed", startedAt, completedAt, durationMs:30628}` (lines 67-68).
7. Interrupt: `turn/interrupt {threadId, turnId}` → `{}`, then `turn/completed` `status:"interrupted"` (lines 80-91).

Limitations and caveats:
- A read-only PowerShell command (`Get-Content`) ran without an approval
  request under `read-only` + `on-request` (line 56); a command that needs
  approval was not probed (section 10).
- `turn/completed` and `turn/started` carry an empty `items` array; the item
  notifications are the canonical list ([T] README:1446; [P:p6b] line 68).
- `turn/interrupt` does not stop background terminals
  (`thread/backgroundTerminals/clean`, experimental) ([T] README:1014).
- Threads started by any app-server client are recorded with
  `source: "vscode"` ([P:p6b] line 9), so they appear among interactive
  sessions (`thread/list` default `sourceKinds` = `cli`, `vscode`,
  [T] README:389).
- Backpressure: requests can be rejected with `-32001` "Server overloaded;
  retry later." ([T] README:49-53).
- stderr carries tracing logs; in the probes an error about the models cache
  ("missing field `base_instructions`") that did not affect the turns
  ([P:p6b] lines 51, 60, 72). `LOG_FORMAT=json` makes stderr JSON
  ([T] README:44-47).
- Without `capabilities.experimentalApi:true`, experimental methods and
  fields are rejected with "`<descriptor>` requires experimentalApi capability"
  ([T] README:2286-2345). The dashboard can stay on the stable surface for the
  MVP; subagent tree filters (`thread/list.parentThreadId`/`ancestorThreadId`)
  are experimental ([T] README:144,418-420).
- Schema per version: `codex app-server generate-ts|generate-json-schema --out DIR [--experimental]`
  ([T] README:55-62,2293-2305; `codex-rs/cli/src/main.rs:599-603`).

### 4.2 `codex exec --json`

Sequence settled by probes: success = `thread.started`, `turn.started`,
`item.completed` (`agent_message` "ok"), `turn.completed` with
`usage {input_tokens, cached_input_tokens, cache_write_input_tokens, output_tokens, reasoning_output_tokens}`
[P:p5b-codex-exec.jsonl]; failure = `thread.started`, `item.completed`
(`type:"error"`, a non-fatal warning), `turn.started`, `error`, `turn.failed`,
exit 1 [P:p5-codex-exec.jsonl]. stderr says "Reading additional input from
stdin..." when stdin is a pipe ([P:p5-codex-stderr.txt]; [T]
`codex-rs/exec/src/cli.rs:81-85`): close or null stdin.
Item types: `agent_message`, `reasoning`, `command_execution`, `file_change`,
`mcp_tool_call`, `collab_tool_call`, `web_search`, `todo_list`, `error`
([T] `codex-rs/exec/src/exec_events.rs:107-133`); `item.updated` is only
emitted for the todo list ([T] `event_processor_with_jsonl_output.rs:556-573`).
No rate limits, no context window, no deltas, no approvals. Useful as a
fallback or for one-shot tasks, not for zone B.

### 4.3 Interactive TUI in a PTY

Possible (ConPTY on Windows) but not a good primary path: the output is a
full-screen ratatui repaint, not events; approvals, agent messages and
subagent activity would have to be scraped from the screen, which breaks the
"observed events vs AI messages" distinction; keystroke protocols change
between versions; and Windows ConPTY plus Bubble Tea rendering a nested
terminal adds a VT emulator to the project. Its only advantage, the exact
native UI, is already reproducible from app-server events because the TUI
is itself an app-server client in 0.145.0. Reserve it for a "raw native
terminal" escape hatch, not the MVP.

## 5. Subagents and parallel work

- **Feature:** `multi_agent` is Stable and on by default; tools
  `spawn_agent`, `send_input`, `resume_agent`, `wait`, `close_agent`
  ([T] `codex-rs/features/src/lib.rs:1049-1054`;
  `codex-rs/app-server-protocol/src/protocol/v2/item.rs:1028-1034`;
  [D:config-ref] `features.multi_agent`; [D:subagents] "Current Codex releases
  enable subagent workflows by default"). Built-in roles at [T]:
  `explorer`, `awaiter` (`codex-rs/core/src/agent/builtins/`); docs list
  `default`, `worker`, `explorer` [D:subagents]. Nicknames from a fixed list
  (`codex-rs/core/src/agent/agent_names.txt`).
- **When:** the model spawns agents on explicit request; reasoning effort
  `ultra` ("Maximum reasoning with automatic task delegation") makes it
  proactive ([P:p6-codex-model-list.json]; [T] README:140,143).
- **Limits:** `[agents] max_threads` (alias of
  `max_concurrent_threads_per_session`) default **6** for the V1 backend,
  `max_depth` default **1**; V2 backend default 4 concurrent threads per
  session ([T] `codex-rs/config/src/config_toml.rs:679-700`;
  `codex-rs/core/src/config/mod.rs:206-207,268,1463-1477`). Exceeding gives
  `AgentLimitReached` ([T] `codex-rs/core/src/agent/control/execution.rs:67-71`).
- **Each subagent is its own thread** with `parentThreadId`, `agentNickname`,
  `agentRole` ([T] README:144,395,495; `v2/thread_data.rs:172-224`).

How a program sees them through app-server:

| Signal | Content | Source |
| --- | --- | --- |
| `collabAgentToolCall` item on the parent thread (`item/started`/`completed`) | `tool` (`spawnAgent`, `sendInput`, `resumeAgent`, `wait`, `closeAgent`), `status`, `senderThreadId`, `receiverThreadIds[]`, `prompt`, `model`, `reasoningEffort`, `agentsStates{threadId:{status, message}}` with status `pendingInit`, `running`, `interrupted`, `completed`, `errored`, `shutdown`, `notFound` | [T] `v2/item.rs:336-346,1028-1034,1172-1189` (the README's `collabToolCall` field list at README:1459 and [D:app-server] is older; trust the schema) |
| `subAgentActivity` item | `kind` (`started`, `interacted`), `agentThreadId`, `agentPath` | [T] `v2/item.rs:359-364,1153-1163` |
| Child thread events | every initialized connection is auto-attached to each newly created thread, so child `turn/*`, `item/*`, `thread/status/changed` arrive tagged with the child `threadId` | [T] `codex-rs/app-server/src/lib.rs:1083-1097`; `request_processors/thread_processor.rs:2981-3011`; `thread_status.rs:309-326` |
| Tree after the fact | `thread/list` with `parentThreadId` / `ancestorThreadId` (experimental), `thread/read` | [T] README:144,418-435 |
| Exec mode | `collab_tool_call` items only | [T] `exec_events.rs:250-259` |

Observed in P7 (two subagents in parallel, [P:p7-codex-subagents.jsonl]):
- **Spawning shows as `subAgentActivity`, not as a `spawnAgent` call.** The
  parent's `item/completed` of type `subAgentActivity` with `kind: "started"`,
  `agentThreadId` and `agentPath` (`/root/list_files`, `/root/create_found`)
  announced each child; no `collabAgentToolCall` with `tool: "spawnAgent"`
  was emitted. The two `collabAgentToolCall` items were `wait` calls, with
  empty `receiverThreadIds` and `agentsStates`. The path-style names suggest
  the V2 agent backend; which backend is active is not reported.
- **A child gets no `thread/started`.** Its first notification is
  `thread/status/changed` (`idle`, then `active`); then `turn/started`,
  items, deltas, `thread/tokenUsage/updated`, and `turn/completed` with
  `durationMs`, all tagged with the child's `threadId`. No
  `subAgentActivity` marks the end; the child's `turn/completed` and its
  status `idle` do.
- **A child's approval request carries the child's `threadId`**
  (`item/fileChange/requestApproval`), with `waitingOnApproval` in the
  child's status; one host answer serves parent and children alike.
- Not observed: `turn/interrupt` on the parent while children run (section
  10, question 3).

Observed in P7b (P7 again on Codex CLI 0.163.0, [P:p7b-codex-subagents.jsonl]):
- **A child reports to the orchestrator with `subAgentActivity`
  `kind:"interacted"`** in its own thread, whose `agentThreadId` is the
  orchestrator's thread and `agentPath` is `/root`. Read as an agent, it made
  a "/root" node that never ended (L5, 2026-10-17); it is now a status of the
  child.
- **A child's end is also marked on the parent**: `subAgentActivity`
  `kind:"completed"` (`agentPath` the child's), after the child's
  `turn/completed`.
- **`item/commandExecution/outputDelta`** streams command output (28 chunks),
  and `account/updated` carries `authMode` and `planType`.
- The `started` activity now carries the child's `model` and
  `reasoningEffort`; the dashboard does not read them yet.

## 6. Observation points

| Point | Format / location | Documented | Notes and source |
| --- | --- | --- | --- |
| app-server notifications | JSON-RPC notifications on stdout; full list in [T] `codex-rs/app-server-protocol/src/protocol/common.rs:1639-1730` | yes [D:app-server] | key ones below |
| app-server server requests | `item/commandExecution/requestApproval`, `item/fileChange/requestApproval`, `item/tool/requestUserInput` (experimental), `mcpServer/elicitation/request`, `item/permissions/requestApproval`, `item/tool/call` (dynamic tools, experimental) | yes | [T] `common.rs:1486-1530`; README:1526-1690 |
| `codex exec --json` | `thread.started`, `turn.started`, `turn.completed{usage}`, `turn.failed{error}`, `item.started|updated|completed`, `error` | yes [D:noninteractive] | [T] `exec_events.rs:11-37,50-73` |
| Session rollout files | `$CODEX_HOME/sessions/YYYY/MM/DD/rollout-YYYY-MM-DDThh-mm-ss-<threadId>.jsonl`; line = `{timestamp, ordinal?, type, payload}`, `type` ∈ `session_meta`, `response_item`, `inter_agent_communication`, `compacted`, `turn_context`, `world_state`, `event_msg` | **undocumented** (no site page; [D:config-ref] has no key for it) | [T] `codex-rs/rollout/src/recorder.rs:1528-1546`; `codex-rs/protocol/src/protocol.rs:3209-3221,3404-3411`; path confirmed by `thread.path` [P:p6b] line 9. Persisted `event_msg`s: `token_count`, `task_started`, `task_complete`, `turn_aborted`, ... (`codex-rs/rollout/src/policy.rs:87-104`); deltas and approvals are not persisted (`policy.rs:120-182`). Files older than 7 days may be compressed to `.zst` (`codex-rs/rollout/src/compression.rs:4,253`). |
| `token_count` event (rollout, legacy protocol) | `{info:{total_token_usage, last_token_usage, model_context_window}, rate_limits:{limit_id, primary:{used_percent, window_minutes, resets_at}, secondary, credits, plan_type, ...}}` | undocumented | [T] `codex-rs/protocol/src/protocol.rs:2059-2081,2143-2199` |
| `notify` | `notify = ["prog", ...]` gets one JSON argument `{"type":"agent-turn-complete","thread-id","turn-id","cwd","client","input-messages","last-assistant-message"}` | yes [D:config-advanced] | requires a config change; [T] `codex-rs/config/src/config_toml.rs:214-216`; `codex-rs/hooks/src/legacy_notify.rs:87-95` |
| Hooks | `SessionStart`, `SessionEnd`, `UserPromptSubmit`, `PreToolUse`, `PermissionRequest`, `PostToolUse`, `PreCompact`, `PostCompact`, `SubagentStart`, `SubagentStop`, `Stop`; JSON on stdin | yes [D:hooks] | requires `hooks.json`/config and hook trust; app-server forwards `hook/started`, `hook/completed`; [T] `codex-rs/protocol/src/protocol.rs:1496-1508`; `common.rs` (HookStarted/HookCompleted) |
| OpenTelemetry | `[otel]` `exporter`/`trace_exporter`/`metrics_exporter` = `none`, `otlp-http`, `otlp-grpc` (metrics default `statsig`); events `codex.conversation_starts`, `codex.api_request`, `codex.sse_event`, `codex.websocket_request`, `codex.user_prompt`, `codex.tool_decision`, `codex.sandbox_outcome`, `codex.tool_result`, `codex.turn_ttft`, ... | yes [D:config-advanced] "otel" | off by default; could be enabled per run with `-c otel.exporter=...` (no file write) but needs an OTLP receiver in the dashboard; [T] `codex-rs/config/src/types.rs:529-580`; `codex-rs/otel/src/events/session_telemetry.rs:239-1136` |
| Logs | `RUST_LOG`, `LOG_FORMAT=json` (app-server stderr); `-c log_dir=...` for the TUI | yes | [T] README:44-47; `docs/install.md` "Tracing" |
| History | `~/.codex/history.jsonl` (prompts) | yes [D:config-advanced] | not needed |

Key app-server notifications ([T] `common.rs:1639-1700`, README:1432-1524):
`turn/started`, `turn/completed` (`status`, `error{message, codexErrorInfo, additionalDetails}`, `startedAt`, `completedAt`, `durationMs`),
`item/started`, `item/completed`, `item/agentMessage/delta` (`itemId`, `delta`),
`item/reasoning/summaryTextDelta`, `item/reasoning/textDelta`,
`item/commandExecution/outputDelta`, `item/mcpToolCall/progress`,
`turn/diff/updated`, `turn/plan/updated` (`plan[{step, status}]`),
`thread/tokenUsage/updated`, `account/rateLimits/updated`,
`thread/status/changed` (`notLoaded | idle | systemError | active{activeFlags: waitingOnApproval | waitingOnUserInput}`, [T] `v2/thread.rs:1345-1348`),
`error` (`error`, `willRetry`, `threadId`, `turnId`; [T] `v2/notification.rs:41-47`),
`warning`, `model/rerouted`, `mcpServer/startupStatus/updated`, `serverRequest/resolved`.
Item types (`ThreadItem`, [T] `v2/item.rs:227-392`): `userMessage`,
`hookPrompt`, `agentMessage` (`text`, `phase`), `plan`, `reasoning`,
`commandExecution` (`command`, `cwd`, `status`, `commandActions`,
`aggregatedOutput`, `exitCode`, `durationMs`), `fileChange` (`changes`,
`status`), `mcpToolCall`, `dynamicToolCall`, `collabAgentToolCall`,
`subAgentActivity`, `webSearch`, `imageView`, `sleep`, `imageGeneration`,
`enteredReviewMode`, `exitedReviewMode`, `contextCompaction`.

## 7. Telemetry for zone D

All from app-server unless stated. "N/A" means no supported source.

| Metric | Source | Notes |
| --- | --- | --- |
| 5-hour quota | `account/rateLimits/read` → `rateLimits.primary {usedPercent, windowDurationMins:300, resetsAt}`; pushed as `account/rateLimits/updated` after each model call | documented [D:app-server]; [T] README:2206-2247, `v2/account.rs:294-300,518-534,610-617`; **works without a turn** [P:p6] line 7, [P:p6b] lines 7, 42. Requires ChatGPT sign-in ("chatgpt authentication required to read rate limits", [T] `request_processors/account_processor.rs:1016-1030`): with an API key, N/A. Identify windows by `windowDurationMins`, not by primary/secondary position, as the TUI does ([T] `codex-rs/tui/src/chatwidget/rate_limits.rs:77-104`: 300 → "5h", 10080 → "weekly"). Docs: Pro plans "currently have no five-hour limit" [D:pricing], so `primary` may be absent or another window. |
| 7-day quota | same, `secondary {windowDurationMins:10080}` | [P:p6b] line 7 |
| Reset times | `resetsAt` (Unix seconds) per window | as above |
| Plan, credits | `planType` ("plus" here), `credits {hasCredits, unlimited, balance}`, `rateLimitReachedType`, `rateLimitResetCredits` | [P:p6b] line 7; [T] `v2/account.rs:525-534,631-636` |
| Model | `thread/start` response `model`, `reasoningEffort`; `model/rerouted` when the backend switches | [P:p6b] line 9; [T] README:1442 |
| Context window usage | `thread/tokenUsage/updated.tokenUsage.last.totalTokens` / `modelContextWindow` (258400 in P6b) | [P:p6b] line 41; [T] `v2/thread.rs:1459-1522`. The TUI uses `last` and subtracts a 12 000-token baseline ([T] `codex-rs/protocol/src/protocol.rs:2215-2262`; `codex-rs/tui/src/status/card.rs:326-333`). Before the first turn: N/A (or `config/read` `model_context_window` if configured; unverified). |
| Tokens | `thread/tokenUsage/updated` `total` and `last`: `totalTokens`, `inputTokens`, `cachedInputTokens`, `cacheWriteInputTokens`, `outputTokens`, `reasoningOutputTokens`; replayed on `thread/resume` | [P:p6b] line 41; [T] README:321 |
| Account token activity | `account/usage/read` → `summary`, `dailyUsageBuckets` | [T] `common.rs:1056-1060`, `v2/account.rs:392-395`; not probed |
| API-equivalent cost | **N/A — no supported source.** Codex reports no cost field. [D:pricing] gives *credits* per 1M tokens per model, not USD, and "API key" use is billed "based on API pricing" without prices on the page. Any USD estimate would need an external price table, kept separate from subscription usage and labelled as an estimate. | — |
| Active agent count | derived: subagent threads whose last `agentsStates` status is `pendingInit`/`running`, or loaded threads with `thread/status/changed` `active` | [T] `v2/item.rs:1172-1189`; inferred, label as such |
| Concurrency limit | `agents.max_threads` if the user set it (`config/read`); otherwise the built-in default (6, V1) is not reported by any API → show "6 (default)" only if labelled as a source-code default, else N/A | [T] `core/src/config/mod.rs:206` |
| Duration | `turn/completed.turn.durationMs`, `startedAt`, `completedAt`; per command `durationMs` | [P:p6b] lines 56, 68 |
| Errors | `error` notification, `turn/completed status:"failed"` with `codexErrorInfo` (`ContextWindowExceeded`, `UsageLimitExceeded`, `HttpConnectionFailed`, ..., `other`) | [P:p6] lines 23-24 (`codexErrorInfo:"other"` for the refused model); [T] README:1503-1524 |
| Retries | `error.willRetry: true` notifications (stream retries) | [T] `v2/notification.rs:41-47`; count them; not observed in probes |

## 8. Platform notes

| Platform | Notes |
| --- | --- |
| Windows native | Supported for the CLI ([D:win-sandbox] "Native Windows is supported for the CLI, the desktop app, and the IDE extension"); the repo's own `docs/install.md` still lists "Windows 11 via WSL2" ([T] `docs/install.md:5-9`). Probes ran natively: app-server reports `platformFamily:"windows"`, commands run through `powershell.exe -Command` ([P:p6b] lines 3, 56). Sandbox at 0.145.0: `windows.sandbox = elevated | unelevated` (config), else none, and a write sandbox can be downgraded to read-only ([T] see 3.2); `windowsSandbox/setupStart` exists on app-server but changes system state, so the dashboard must not call it. stdio JSONL works without a PTY. Unix-socket transport and the `codex app-server proxy` are not needed. |
| WSL2 | Install inside WSL with `install.sh`; WSL1 unsupported since 0.115 (bubblewrap sandbox); keep repos in the Linux filesystem [D:wsl]. A Windows dashboard can run `wsl.exe -- codex app-server` (stdio passes through) but paths become Linux paths; not needed on this machine. |
| Linux | bubblewrap + seccomp sandbox [D:commands] "sandbox". |
| macOS | Seatbelt sandbox [D:commands] "sandbox". |

## 9. Answers to the research questions of amendment 2

1. **Embed or control the session?** Yes: `codex app-server` over stdio
   JSON-RPC drives full multi-turn sessions from any language; proven from a
   non-SDK driver ([P:p6b]). Documented, labelled experimental
   ([D:app-server]). Limitation: protocol may change between releases.
   Fallback: `codex exec --json` + `exec resume` per turn, without approvals
   or deltas.
2. **Genuine streaming?** app-server: `item/agentMessage/delta`, reasoning
   and command-output deltas ([P:p6b] lines 22-33; [T] README:1477-1501).
   `exec --json` and the TS SDK: item granularity only.
3. **Subagent lifecycle and tool events?** `collabAgentToolCall` and
   `subAgentActivity` items on the parent, child threads' own `turn/*` and
   `item/*` notifications by `threadId`, `parentThreadId` on threads
   ([T] section 5). Tool events: `commandExecution`, `fileChange`,
   `mcpToolCall`, `webSearch` items. Observed in P7: spawning appears as
   `subAgentActivity` (`started`, `agentThreadId`, `agentPath`), waiting as
   `collabAgentToolCall` `wait`, and each child's own events and approval
   requests by its `threadId`; the end of a child is its `turn/completed`
   (section 5).
4. **Approvals?** Server-to-client requests answered by the dashboard:
   `accept`, `acceptForSession`, `decline`, `cancel` (+ execpolicy and
   network amendments for commands; prefer `availableDecisions`)
   ([P:p6b] lines 36-40; [T] README:1526-1553). Do not offer the
   "amendment" decisions in the MVP: they persist rules to configuration.
   Native sandbox stays in charge; the dashboard never weakens it.
5. **Persistence, cancellation, resume?** Codex persists every thread
   (rollout path in `thread.path`); `turn/interrupt` ends a turn as
   `interrupted` ([P:p6b] lines 80-91); `thread/resume` reopens a thread
   with history and token usage; `thread/fork`, `ephemeral:true` available
   ([T] README:77-81,321-377). Resume across a dashboard restart not yet
   probed.
6. **Reuse the existing terminal UI?** Not the TUI (section 4.3). Reuse the
   protocol instead: the committed JSON Schema (Apache-2.0) to generate Go
   types, and the TUI's rules (window labelling, context baseline) as
   reference.
7. **Concurrent subagent output?** One stdout stream multiplexes all threads;
   demultiplex by `threadId`/`turnId`/`itemId` into per-agent buffers in the
   state store; render from state. Deltas can be opted out per connection
   (`optOutNotificationMethods`) if a stream is too noisy ([T] README:1389-1401).
8. **Reliable quota?** `account/rateLimits/read` at startup and on demand,
   plus `account/rateLimits/updated` pushes; works without a turn, gives both
   windows with reset times ([P:p6] line 7). ChatGPT sign-in only; API-key
   sessions show N/A.
9. **OS limits?** Windows native works over stdio without a PTY; Windows
   sandbox strength depends on Codex's own Windows sandbox config (elevated /
   unelevated, MXC only after 0.145.0); WSL2 needs Linux paths; Linux and
   macOS use bubblewrap / Seatbelt.

**Recommended Go integration:** spawn the resolved `codex.exe app-server`
with `cmd.Dir` = the project (no `cwd` in `thread/start` until the trust-write
probe is done), one long-lived process per dashboard session; a
JSON-RPC reader goroutine on stdout (JSONL, large line buffer), a writer for
requests and approval responses, stderr to a log panel; `initialize` with
`clientInfo.name = "agent_command_center"`, stable API only; `thread/start`
with explicit `model` when configured models fail; per-session
`approvalPolicy: "on-request"` and the sandbox the owner chooses;
`account/rateLimits/read` at startup. Go structs generated from the 0.145.0
JSON Schema; unknown methods and fields tolerated and logged, so a newer
Codex degrades visibly instead of breaking.

## 10. Open questions for further probes

Settled by the probes (no longer open): app-server handshake from a non-SDK
client (P6, P6b); file-change approval flow (P6b); `turn/interrupt` (P6b);
`account/rateLimits/read` without a turn (P6); `thread/tokenUsage/updated`
shape and timing (P6b); `codex exec --json` sequence, success and failure
(P5, P5b); a configured model refused by the account surfaces as `error` +
`turn/completed failed` and is fixed by a per-session model (P5, P6, P6b);
the installed binary's schema equals the [T] fixture (section 1).

Still open:

1. **Command approval:** a command that needs approval (e.g. a write under
   `read-only`): the exact `item/commandExecution/requestApproval` params on
   Windows (`command`, `cwd`, `commandActions`, `availableDecisions`), and the
   item and turn outcome after `decline` and `cancel`.
2. **Trust write:** with `workspace-write` on an untrusted project, does
   omitting `cwd` from `thread/start` (process cwd = project) avoid the
   `config.toml` write, and what effective `sandbox` does the response report
   on Windows with no `windows.sandbox` set (downgrade to read-only?). Needs a
   config hash before/after.
3. **Subagents:** settled by P7 (section 5), except whether
   `turn/interrupt` on the parent stops the children, and whether
   `collabAgentToolCall` `spawnAgent` ever appears (another backend or
   version). P7c (0.163.0, the orchestrator asked to pass a model and an
   effort): no `spawnAgent` item; the child's `subAgentActivity`
   `kind:"started"` carries the model and effort asked for (`gpt-6-luna`,
   `low`) [P:p7c-codex-spawn-model.jsonl].
4. **Resume across processes:** a new app-server process, `thread/resume`
   on P6b's thread: returned `turns`, replayed `thread/tokenUsage/updated`.
5. **`turn/steer`** mid-turn and a second `turn/start` while one is running
   (expected `ActiveTurnNotSteerable`/error).
6. **Clarification questions:** whether `item/tool/requestUserInput` appears
   in Plan collaboration mode with `experimentalApi`, or the MVP treats
   questions as plain agent text (Default mode has it disabled:
   `default_mode_request_user_input` is under development, [T]
   `codex-rs/features/src/lib.rs:1240-1245`).
7. **Context window before the first turn:** any stable source (`config/read`,
   `model/list`) for `modelContextWindow`.
8. **Shutdown:** behaviour when the dashboard closes stdin or kills the
   process mid-turn on Windows (orphaned shells, rollout state).
9. **Retries:** whether `error.willRetry:true` appears on transient network
   errors behind the corporate proxy, to drive the retry counter.
10. **API-key sessions:** `account/rateLimits/read` error text, to show N/A
    cleanly (needs an API-key login, so likely untested).
