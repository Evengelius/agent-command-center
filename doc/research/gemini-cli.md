# Google Gemini CLI

**Out of scope since 2026-10-15 (amendment 5).** Kept only as a research
record for a possible future provider; no longer a plan 1 deliverable, and
nothing in the plans builds on it.

Plan 1, step 1. **Gemini CLI is not installed on this machine** (not on the
PATH, AGENTS.md), so everything below comes from documentation and source only:
nothing was installed, built or run, and nothing under `~/.gemini` was read.
Citations `path:line` refer to the repository at the commit in section 1.
Web pages were accessed 2026-10-15. The site geminicli.com is built from the
repository's `docs/` folder: pages marked "(fetched)" were read live; the others
were read as `docs/...` at the commit, and their site URL is given for
reference.

## 1. Sources

- Repository: https://github.com/google-gemini/gemini-cli, shallow clone at
  `a2426f78653c8a3848fb2173d76b9dbf5b8e4fe2` (committed 2026-10-15 18:52 UTC).
  The tree's version is `0.66.0-nightly.20261013.g204db94ae` (package.json).
- **Latest stable release: v0.64.0**, 13 Oct 2026
  (`docs/changelogs/latest.md:1-3`; GitHub releases page, fetched, marks
  v0.64.0 "Latest"). Latest preview: v0.65.0-preview.0, 13 Oct 2026
  (`docs/changelogs/preview.md:1-3`).
- The key interfaces were checked against the v0.64.0 tag (fetched from
  raw.githubusercontent.com): `--acp` and `--experimental-acp`, the
  `stream-json` event types, the `ask_user` exclusion and
  `AGENT_TOOL_NAME = 'invoke_agent'` are identical to the commit.
- License: `LICENSE` is the Apache License 2.0; files carry
  `SPDX-License-Identifier: Apache-2.0` (e.g. `packages/core/src/output/types.ts:4`).
  **SPDX: Apache-2.0.** Copying is allowed with the notice kept; we only drive
  the binary, so nothing needs to be copied.
- Documentation pages (geminicli.com, 2026-10-15):
  - https://geminicli.com/docs/ (fetched): banner "Unpaid tier and Google One
    users: Gemini CLI was replaced by Antigravity CLI on June 18th, 2026", linking
    https://developers.googleblog.com/an-important-update-transitioning-gemini-cli-to-antigravity-cli
    (fetched; see section 7.1).
  - https://geminicli.com/docs/cli/headless/ (fetched) = `docs/cli/headless.md`
  - https://geminicli.com/docs/cli/acp-mode/ (fetched) = `docs/cli/acp-mode.md`
  - https://geminicli.com/docs/cli/cli-reference/ = `docs/cli/cli-reference.md`
  - https://geminicli.com/docs/core/subagents/ = `docs/core/subagents.md`
  - https://geminicli.com/docs/hooks/reference/ = `docs/hooks/reference.md`
  - https://geminicli.com/docs/cli/session-management/ = `docs/cli/session-management.md`
  - https://geminicli.com/docs/cli/telemetry/ = `docs/cli/telemetry.md`
  - https://geminicli.com/docs/resources/quota-and-pricing/ = `docs/resources/quota-and-pricing.md`
  - https://geminicli.com/docs/cli/sandbox/ = `docs/cli/sandbox.md`
  - https://geminicli.com/docs/cli/trusted-folders/ = `docs/cli/trusted-folders.md`
  - https://geminicli.com/docs/reference/configuration/ = `docs/reference/configuration.md`
  - https://geminicli.com/docs/reference/policy-engine/ = `docs/reference/policy-engine.md`
  - https://geminicli.com/docs/get-started/installation/ = `docs/get-started/installation.mdx`
- Agent Client Protocol spec (fetched): https://agentclientprotocol.com/protocol/prompt-turn,
  https://agentclientprotocol.com/protocol/session-setup,
  https://agentclientprotocol.com/protocol/tool-calls,
  https://agentclientprotocol.com/libraries/community. Gemini CLI uses
  `@agentclientprotocol/sdk` 0.16.1 (`packages/cli/package.json:33`).
- npm folders (fetched): https://docs.npmjs.com/cli/v11/configuring-npm/folders.

## 2. Detection and version

| Item | Value | Source |
|---|---|---|
| Package | `@google/gemini-cli` (`npm install -g @google/gemini-cli`; also Homebrew, npx) | `docs/get-started/installation.mdx:36,45,94` (documented) |
| Binary | `gemini` (bin `dist/index.js`) | `packages/cli/package.json:12-13` |
| Windows PATH names | npm puts global executables "directly into `{prefix}`", default prefix `%AppData%\npm`: expect `gemini.cmd`, `gemini.ps1` and an extensionless shell shim there (the `.cmd`/`.ps1` names are npm's usual shims; the page does not name them) | npm folders page (documented); shim names not verified on this machine |
| Runtime | Node.js >= 20 | `package.json:4-5`; installation.mdx:17 |
| Version | `gemini --version` (`-v`), prints package.json version | `packages/cli/src/config/config.ts:500-501`; `docs/cli/sandbox.md:13-15` |
| Supported OS | macOS 15+, Windows 11 24H2+, Ubuntu 20.04+ | installation.mdx:10-13 |

Detection from Go: `exec.LookPath("gemini")` (finds `gemini.cmd` through
PATHEXT on Windows), then run `gemini --version` with a timeout. Note that on
Windows a `.cmd` shim is run by `cmd.exe`: arguments must be quoted for cmd, or
the dashboard resolves `node` + `<prefix>\node_modules\@google\gemini-cli\dist\index.js`
itself (layout inferred from npm, unverified).

`~/.gemini` existing without the binary (AGENTS.md) means it was installed once;
it does not mean it works.

## 3. Working directory, approvals, sandbox, configuration

**Working directory.** There is no `--cwd` flag (none in
`config.ts:288-498`); the CLI works in the process's current directory. Headless:
start the process with `cmd.Dir = project`. ACP: `session/new` and
`session/load` take a `cwd` parameter, and the session's settings and config are
built for it (`packages/cli/src/acp/acpSessionManager.ts:66-77,426`).
`--include-directories a,b` adds workspace roots (`config.ts:455-461`;
cli-reference.md:68). Sessions are stored per project (section 6.3).

**Approval modes** (`--approval-mode`, `config.ts:340-346`, documented in
cli-reference.md:57):

| Mode | Meaning |
|---|---|
| `default` | prompt for approval |
| `auto_edit` | auto-approve edit tools |
| `yolo` (also `-y`) | auto-approve all tools |
| `plan` | read-only mode (can be disabled by `general.plan.enabled`, `config.ts:714-720`) |

Overrides: YOLO is refused when `security.disableYoloMode` or
`admin.secureModeEnabled` (`config.ts:737-751`); **any mode is forced back to
`default` when the folder is untrusted** (`config.ts:758-763`).
Fine-grained rules come from the Policy Engine (TOML, documented in
policy-engine.md); `--policy <file>` adds a policy file for one invocation
without touching config (`config.ts:347-354`). **In non-interactive mode an
`ask_user` decision is a `deny`** (policy-engine.md:116-117;
`packages/core/src/policy/policy-engine.ts:301,1010`).

**Trusted folders** (documented, trusted-folders.md): the feature is off by
default (`:10-11`). When on and the folder is untrusted, the CLI runs in "safe
mode": no project `.gemini/settings.json`, no `.env`, no MCP, no tool
auto-acceptance (`:74-101`); headless exits with `FatalUntrustedWorkspaceError`
unless `--skip-trust` or `GEMINI_CLI_TRUST_WORKSPACE=true` is given, which
trusts the folder "for the duration of the session" (`:103-117`;
`config.ts:308-312,517`).

**Sandbox** (documented, sandbox.md): `-s/--sandbox` or
`GEMINI_SANDBOX=true|docker|podman|sandbox-exec|runsc|lxc` (`:39-77`). Methods:
macOS Seatbelt with `SEATBELT_PROFILE` profiles (`:86-101`), Docker/Podman
(`:103-180`), **Windows native sandbox** using `icacls` Low integrity level,
whose changes **persist on the files after the session** (`:181-197`), gVisor
and LXC (Linux only, `:199-261`), plus per-tool sandboxing
(`security.toolSandboxing`, `:267-294`) and "sandbox expansion" requests
(`:296-335`, surfaced as a `sandbox_expansion` confirmation,
`packages/cli/src/acp/acpUtils.ts:191`). The dashboard must not enable a sandbox
the owner did not configure; it passes the owner's own choice through.

**Configuration loading** (documented, configuration.md:7-21): defaults <
system defaults < user `~/.gemini/settings.json` < project
`.gemini/settings.json` < system settings < environment (incl. `.env`) <
command-line flags. Per-invocation knobs that avoid changing config:
`--model`, `--approval-mode`, `--policy`, `--include-directories`,
`--skip-trust`, `--output-format`, `--sandbox`, `--resume`, `--session-id`
(`config.ts:288-498`), and the `GEMINI_TELEMETRY_*` variables (section 6.4).
Proxy: `HTTPS_PROXY`/`https_proxy` are honoured (`config.ts:1029-1030`), which
matters behind this machine's corporate proxy.

**Things that write config, to avoid** (undocumented, from source):
- ACP `authenticate` writes `security.auth.selectedType` to the **user**
  settings file (`acpRpcDispatcher.ts:162-166`) and may clear cached
  credentials when switching method (`:111-114`). The dashboard must never call
  it; it relies on the owner having logged in with `gemini` itself.
- Choosing a permission option "Allow ... in all future sessions"
  (`ProceedAlwaysAndSave`, offered only if
  `security.enablePermanentToolApproval`) persists a policy rule
  (`acpUtils.ts:114-169`; `acpSession.ts:776-784`). The dashboard must not
  offer or select it.
- `-w/--worktree` creates a git worktree (`config.ts:313-326`); out of MVP scope.

## 4. Ways to drive a session from another program

### 4.1 ACP: `gemini --acp` (recommended)

- **Transport:** JSON-RPC 2.0, newline-delimited JSON over stdin/stdout
  (`packages/cli/src/acp/acpStdioTransport.ts:20-29`; acp-mode.md:45-50).
- **Documented:** yes, https://geminicli.com/docs/cli/acp-mode/ (fetched) and
  the ACP spec. The flag is `--acp`; `--experimental-acp` is kept as
  "deprecated, use --acp instead" (`config.ts:363-371`, same in v0.64.0).
  The doc page has no experimental label; only `unstable_setSessionModel` is
  marked unstable. cli-reference.md:59-60 is stale (lists
  `--experimental-acp` and `--experimental-zed-integration`, not `--acp`).
  Used by Zed and JetBrains through the ACP registry (acp-mode.md:27-38).
- **Methods implemented** (`acpRpcDispatcher.ts`): `initialize` (returns
  `agentCapabilities.loadSession: true`, image/audio/embeddedContext prompts,
  MCP http/sse, `agentInfo.version`, `:40-104`), `authenticate` (`:106-167`),
  `session/new` (`:177-181`), `session/load` (`:183-187`), `session/cancel`
  (`:189-198`), `session/prompt` (`:200-209`), `session/set_mode`
  (`:211-222`), `unstable_setSessionModel` (`:224-235`). Not implemented:
  `session/resume` (not advertised), any session list method.
- **Streaming granularity:** per model chunk. `session/update` kinds emitted:
  `agent_message_chunk` per content chunk (`acpSession.ts:446-457`),
  `agent_thought_chunk` per thought summary (`:459-466`), `tool_call`
  (`pending` when approval is needed, `:736-744`; `in_progress` otherwise,
  `:827-835`), `tool_call_update` (`completed`/`failed`, with text or `diff`
  content, `:845-853,909-917`), `usage_update` with `used` and `size` after
  each model response (`:472-491`), `available_commands_update` (`:232-242`),
  `user_message_chunk` only when replaying history (`:249-317`). Not emitted:
  `plan`, `current_mode_update`, `session_info_update`. An approval-mode change
  is sent as an `agent_message_chunk` with text `[MODE_UPDATE] <mode>`
  (`:179-189`), which a client must filter out of the conversation.
- **Turn end:** the `session/prompt` response carries `stopReason`
  (`end_turn`, `max_turn_requests`, `max_tokens`, `cancelled`), `usage`
  (`inputTokens`, `outputTokens`, `cachedReadTokens`, `thoughtTokens`,
  `totalTokens`) and `_meta.quota.model_usage[]` per model (`:381-409`). The
  `_meta.quota` name holds token counts only, not quota. A 429 becomes JSON-RPC
  error 429 "Rate limit exceeded. Try again later." (`:527-532`).
- **Multi-turn in one process:** yes; several sessions per process are kept in
  a map (`acpSessionManager.ts:35`). A new `session/prompt` aborts a prompt
  still running in the same session (`acpSession.ts:320`).
- **Approvals:** `session/request_permission` with options `allow_once`
  (`ProceedOnce`), `allow_always` (session-scoped `ProceedAlways`, or server/tool
  for MCP), `reject_once` (`Cancel`) (`acpUtils.ts:85-200`); edits carry a
  `diff` (`acpSession.ts:713-727`). The client's answer is applied, then the
  tool runs. Mode switch with `session/set_mode` (`default`, `auto_edit`,
  `yolo`, `plan` if enabled; `acpUtils.ts:224-252`). Access to files outside
  the workspace named with `@path` also asks permission (`acpSession.ts:1080-1115`).
- **Clarification questions:** the `ask_user` tool is **excluded in ACP mode**
  (`config.ts:796-804`, comment: "IDEs intercept tool calls ... breaking
  conversational flows"). The model can only ask in plain text and wait for the
  next prompt. `exit_plan_mode` confirmations do arrive as permission requests.
- **Interrupt/cancel:** `session/cancel` aborts the pending prompt's
  AbortController (`acpSession.ts:206-213`); the prompt returns
  `stopReason: "cancelled"` (`:421-423,441-443,534-539`). It throws "Not
  currently generating" if idle.
- **Resume:** `session/load` with `sessionId` and `cwd` resolves the saved
  session of that project, restores the chat and **replays history** as
  `session/update` notifications (`acpSessionManager.ts:204-294`;
  `acpSession.ts:249-317`). It requires an already selected auth type, else
  `authRequired` (`acpSessionManager.ts:322-330`). The session id of a new ACP
  session is a UUID chosen by the agent (`:70`).
- **Limitations:** (1) tool calls of one model turn run **sequentially**
  (`acpSession.ts:595-604`), unlike headless; (2) subagent internals are not
  streamed (section 5); (3) no quota; (4) no tool name or raw arguments in
  `tool_call`, only `title`, `kind`, `locations`, `content`; the tool name is
  recoverable only from the `toolCallId` prefix `<tool>__...`
  (`packages/core/src/core/turn.ts:480-482`, undocumented); (5) auth must
  pre-exist (`newSession` fails with -32000 "Authentication required." or
  "Gemini API key is missing", `acpSessionManager.ts:96-123`); (6) if the
  client advertises `fs` capabilities, file reads/writes are proxied to the
  client (`:127-136`): the dashboard should **not** advertise them, so the CLI
  keeps its own file access and permissions.
- **Go client:** no official Go SDK; the ACP site lists seven community Go
  libraries, e.g. `github.com/coder/acp-go-sdk` (libraries/community, fetched;
  licenses not checked). The subset needed (initialize, session/new, prompt,
  cancel, load, set_mode, answer request_permission, read session/update) is
  small enough to write directly on `encoding/json` over the pipes.

### 4.2 Headless: `gemini -p ... --output-format stream-json`

- **Transport:** one process per prompt, JSONL on stdout; logs on stderr.
  Headless when `-p` is given or stdin is not a TTY (headless.md:8-9). Exit
  codes 0, 1, 42 (input), 53 (turn limit) (headless.md:37-45, fetched).
- **Documented:** yes, headless.md (event list only, no field schema). The
  field schema is in source, identical at v0.64.0
  (`packages/core/src/output/types.ts:29-117`), see section 6.1.
- **Streaming:** assistant text in `message` events with `delta: true` per
  chunk (`packages/cli/src/nonInteractiveCli.ts:340-351`); `tool_use` when the
  model requests a tool (`:359-369`); `tool_result` after the batch has run
  (`:484-512`); thoughts are **not** emitted.
- **Multi-turn:** no; each turn is a new process. Continue with
  `--resume <session_id|index|latest>` (`config.ts:401-421`;
  session-management.md:30-53), or start with a chosen id via `--session-id`
  (`config.ts:426-444`, not in cli-reference.md: undocumented).
- **Approvals:** none interactive. `ask_user` decisions become `deny`
  (policy-engine.md:116-117) and `ask_user` is excluded (`config.ts:796-804`).
  So the dashboard must choose the mode up front: `default` lets only
  allow-listed tools run (writes and shell are denied), `auto_edit`/`yolo`
  approve without asking. This breaks "approve actions" in zone B.
- **Cancel:** Ctrl+C is read from stdin only when stdin is a TTY
  (`nonInteractiveCli.ts:133-137`); otherwise SIGINT/SIGTERM trigger graceful
  shutdown (`packages/cli/src/utils/cleanup.ts:147-148`). On Windows there is
  no SIGINT for a child without a console: killing the process is the realistic
  cancel (unverified).
- **Limitations:** process start-up cost per turn (MCP servers, auth) and
  loss of in-memory state; `result` is per process.

### 4.3 Other interfaces

| Interface | Transport | Status | Use for us |
|---|---|---|---|
| `--output-format json` | one JSON object at the end: `session_id`, `response`, `stats` (SessionMetrics), `error`, `warnings` (`types.ts:21-27`) | documented (headless.md:15-22) | no streaming: no |
| SDK `@google/gemini-cli-sdk` | in-process TypeScript (`GeminiCliAgent.session().sendStream()`) | SDK_DESIGN.md:3-5: "hooks, skills, subagents, and ACP are currently missing"; README only, no site page found | would need a Node sidecar: no |
| A2A server `@google/gemini-cli-a2a-server` (bin `gemini-cli-a2a-server`, HTTP on localhost, port `CODER_AGENT_PORT`) | HTTP, A2A protocol | README: "All code in this package is experimental"; separate package, not in the `gemini` install | no |
| Remote subagents (A2A client) | Gemini calling remote agents | documented, docs/core/remote-agents.md | not a driving interface |
| Interactive TUI in a PTY | ConPTY / pty, ANSI screen | the TUI is Ink (React), `packages/cli/package.json:52`; screen layout is not an API | no (see below) |

**Interactive TUI in a PTY.** Possible (Gemini already handles Windows ConPTY,
`docs/changelogs/latest.md:24-28`), and it is the only place showing quota
(`/stats model`). But it means parsing a full-screen React/Ink render with
dialogs, which changes every weekly release; no events, approvals only as
screen dialogs, and it breaks "the interface renders state". Use it only as a
last-resort "open native Gemini" escape hatch, not for zone B.

## 5. Subagents and parallel work

- **What exists** (documented, subagents.md): built-in `codebase_investigator`,
  `cli_help`, `generalist`, `browser_agent` (off by default) (`:52-116`);
  custom agents as Markdown files with `name`, `tools`, model, limits
  (`:313-395`); remote A2A agents (`:575-581`); enabled by default, disable
  with `experimental.enableAgents: false` (`:589-598`). Each has its own
  context window (`:16-17`). Policy rules can target a subagent (`:442-466`).
- **How they are invoked:** the doc says each agent is "exposed to the main
  agent as a tool of the same name" (`:19-22`), but at this commit and at
  v0.64.0 the registry registers a single tool `invoke_agent` with arguments
  `agent_name` and `prompt` (`packages/core/src/tools/tool-names.ts:191`;
  `packages/core/src/agents/agent-tool.ts:43-72`;
  `packages/core/src/config/config.ts:4127-4130`). Which name reaches the event
  streams must be checked on a run.
- **Parallelism:** the core scheduler batches contiguous "parallelizable" tool
  calls and runs them with `Promise.all` (`packages/core/src/scheduler/scheduler.ts:519-565`);
  a call is parallelizable unless it is an edit tool or sets
  `wait_for_previous: true` (`packages/core/src/utils/tool-utils.ts:29-41`). So
  in headless mode several `invoke_agent` calls can run at once. **ACP runs a
  turn's tool calls one after another** (`acpSession.ts:595-604`). No documented
  concurrency limit; the only constant found is `MAX_PARALLEL_INSTANCES = 5`
  for browser agents (`packages/core/src/agents/browser/browserManager.ts:121`).
- **Live progress exists only in-process:** the subagent reports
  `SubagentProgress { agentName, recentActivity[{type: thought|tool_call,
  content, status}], state, result, terminateReason }` through `updateOutput`
  (`packages/core/src/agents/types.ts:98-115`;
  `packages/core/src/agents/local-invocation.ts:103-123,301-323`). Neither
  driver forwards it: headless and ACP call `execute` without a live callback
  (`acpSession.ts:838-840`), and the final `returnDisplay` is that object, not a
  string, so headless `tool_result.output` is empty (`nonInteractiveCli.ts:501-504`)
  and ACP's completed `tool_call_update` has no content (`acpUtils.ts:54-82`).
- **What a program sees:**

| Source | Subagent start | Progress | End | Status |
|---|---|---|---|---|
| ACP | `tool_call` with `kind: "think"` (Kind.Agent maps to think, `acpUtils.ts:215-216`), title `Delegating to agent '<name>'` (`agent-tool.ts:152-153`), `toolCallId` `invoke_agent__...` | none | `tool_call_update` completed/failed, empty content | undocumented mapping |
| stream-json | `tool_use` `tool_name: "invoke_agent"`, `parameters.agent_name`, `parameters.prompt` | none | `tool_result` status, no output | fields documented only as type names |
| Session files | subagent chat written to `chats/<parentSessionId>/...jsonl`, `kind: "subagent"` (`packages/core/src/services/chatRecordingService.ts:780-823`; `chatRecordingTypes.ts:92-106`) | yes, by tailing the file | yes | undocumented format |
| OpenTelemetry | `gemini_cli.agent.start` (`agent_id`, `agent_name`) | none | `gemini_cli.agent.finish` (`duration_ms`, `turn_count`, `terminate_reason`) | documented (telemetry.md:788-817) |

  Zone C can therefore show real start/end, name, prompt and duration, and
  (undocumented) live steps by tailing the subagent's chat file.
- **Workflow tree (zone A):** no plan event. Real sources are the
  `write_todos` tool (`base-declarations.ts:76`) and the experimental task
  tracker tools `tracker_create_task`, `tracker_update_task`, ...
  (`tool-names.ts:184-189`, enabled by `experimental.taskTracker`,
  `config.ts:1049`), visible as tool calls with their arguments (headless
  `tool_use.parameters`; ACP only titles).

## 6. Observation points

### 6.1 stream-json events (documented names; fields from source, `packages/core/src/output/types.ts`)

Every event has `type` and `timestamp` (ISO string) (`:39-42`).

| `type` | Fields | Emitted at |
|---|---|---|
| `init` | `session_id`, `model` (`:44-48`) | start (`nonInteractiveCli.ts:253-260`) |
| `message` | `role` user/assistant, `content`, `delta?` (`:50-55`) | user prompt (`:302-309`); each assistant chunk with `delta: true` (`:344-351`) |
| `tool_use` | `tool_name`, `tool_id`, `parameters` (`:57-62`) | model requests a tool (`:360-368`) |
| `tool_result` | `tool_id`, `status` success/error, `output?` (string only), `error?{type,message}` (`:64-73`) | after the batch ran (`:494-511`) |
| `error` | `severity` warning/error, `message` (`:75-79`) | loop detected, max turns, blocked by hook, invalid stream (`:370-391,428-479`) |
| `result` | `status`, `error?`, `stats?{total_tokens, input_tokens, output_tokens, cached, input, duration_ms, tool_calls, models{<model>:{total_tokens,input_tokens,output_tokens,cached,input}}}` (`:81-109`) | end (`:592-600`) |

API errors are thrown, not emitted as `error` events (`nonInteractiveCli.ts:392-393`):
they end the process with exit code 1 (unverified how they are printed in
stream-json). Core `Retry`, `ChatCompressed`, `Thought`, `Citation` events
(`packages/core/src/core/turn.ts:55-74`) are not forwarded.

### 6.2 ACP `session/update` (documented by the ACP spec; Gemini's use from source)

See 4.1. Key fields: `agent_message_chunk.content.text`;
`tool_call{toolCallId, status, title, kind, locations, content[]}`;
`tool_call_update{toolCallId, status, content[] (content|diff)}`;
`usage_update{used, size}` (`acpSession.ts:483-489`). Prompt response
`usage` and `_meta.quota.model_usage` (`:381-409`).

### 6.3 Session files (location documented, format undocumented)

- Location: `~/.gemini/tmp/<project>/chats/` (session-management.md:18-20). The
  doc says `<project_hash>`; the source now uses a short slug from a registry
  `~/.gemini/projects.json` and migrates old hash folders
  (`packages/core/src/config/storage.ts:274-322`). `GEMINI_CLI_HOME` moves the
  whole folder (docs/cli/enterprise.md:224-240).
- Format: JSONL, `session-<timestamp>-<shortId>.jsonl` (or `<sessionId>.jsonl`),
  append-only deltas (`chatRecordingService.ts:733,812-823`); records
  `ConversationRecord{sessionId, projectHash, startTime, lastUpdated, messages[], kind}`
  with messages `user|gemini|info|error|warning`, `toolCalls[]` (`name`,
  `args`, `status`, `agentId`), `thoughts`, `tokens{input, output, cached,
  thoughts, tool, total}`, `model` (`chatRecordingTypes.ts:19-106`).
- Hooks receive `transcript_path` to this file (hooks/reference.md:46-58).
- Listing: `gemini --list-sessions` (text output), `--delete-session`
  (`config.ts:445-454`). Retention settings in session-management.md:146-208.

### 6.4 OpenTelemetry (documented, telemetry.md)

- Enable per process without changing config: `GEMINI_TELEMETRY_ENABLED=true`,
  `GEMINI_TELEMETRY_TARGET=local`, `GEMINI_TELEMETRY_OUTFILE=<path>`
  (telemetry.md:40-49; acp-mode.md:114-124, which says the file holds
  "detailed information about all the events ..., including ACP requests and
  responses"). `GEMINI_TELEMETRY_LOG_PROMPTS` defaults to `true`: set it to
  `false` (telemetry.md:47).
- Logs useful for zone D: `gemini_cli.api_response` (`model`, `status_code`,
  `duration_ms`, input/output/cached/thoughts/tool/total token counts,
  `auth_type`) (`:519-540`), `gemini_cli.api_error` (`error_type`,
  `status_code`) (`:542-558`), `gemini_cli.tool_call` (`:387`),
  `gemini_cli.chat.content_retry` / `content_retry_failure` (`:643-669`),
  `gemini_cli.flash_fallback` (`:687`), `gemini_cli.chat_compression` (`:620`),
  `gemini_cli.agent.start` / `agent.finish` (`:792-817`).
- Metrics: `gemini_cli.token.usage` (by `model`, `type`), `gemini_cli.api.request.count`,
  `gemini_cli.api.request.latency`, `gemini_cli.tool.call.count`,
  `gemini_cli.agent.run.count`, `gemini_cli.agent.duration`,
  `gemini_cli.agent.turns`, `gemini_cli.chat.content_retry.count`
  (`:958-1155`).
- The exact layout of the outfile (one JSON per line or concatenated objects)
  is not documented: to check on a run.

### 6.5 Hooks (documented, docs/hooks/reference.md)

Command hooks configured in `settings.json` `hooks` (`:21-42`), JSON on stdin
(`session_id`, `transcript_path`, `cwd`, `hook_event_name`, `timestamp`,
`:46-58`), events `BeforeTool`/`AfterTool` (`tool_name`, `tool_input`,
`tool_response`), `BeforeAgent`/`AfterAgent`, `BeforeModel`,
`BeforeToolSelection`, `AfterModel` (every chunk), `SessionStart`,
`SessionEnd`, `Notification`, `PreCompress` (`:92-300`). No subagent-specific
event found. **Hooks need a settings change** (no per-invocation flag found),
so under AGENTS.md they can only be proposed with the exact change; ACP and
stream-json already cover what they would give.

## 7. Telemetry for zone D

### 7.1 Subscription model and availability (documented)

Gemini quotas are **requests per user per day** (plus per-minute limits), not
5-hour or 7-day windows: Code Assist individual 1,000/day, Google AI Pro 1,500,
Ultra 2,000, API key free tier 250 (Flash only), Code Assist Standard 1,500,
Enterprise 2,000 (quota-and-pricing.md table and "Free usage"). This table
predates the **Antigravity transition**: per the Google blog linked from the
docs banner (fetched), Gemini CLI stopped serving free users and Google AI
Pro/Ultra (and Code Assist for individuals) on **2026-06-18**; access stays for
Gemini Code Assist Standard/Enterprise and paid Gemini API / Gemini Enterprise
Agent Platform keys. Which account the owner has decides whether Gemini works
at all.

### 7.2 Metric sources

| Metric | Source | Status |
|---|---|---|
| 5-hour quota | N/A: no such window | N/A — no supported source |
| 7-day quota | N/A: no such window | N/A — no supported source |
| Daily requests remaining (the equivalent) | Code Assist `retrieveUserQuota` (`https://cloudcode-pa.googleapis.com`, `v1internal`; buckets `remainingAmount`, `remainingFraction`, `resetTime`, `modelId`) (`packages/core/src/code_assist/server.ts:73-74,367-373`; `types.ts:250-265`; `config.ts:2315-2392`); shown only in the TUI (`/stats model`, commands.md:438-449; `AppContainer.tsx:650-663`). Not exposed by ACP or stream-json; calling it needs the OAuth token, which we must not read | N/A — no supported source (internal API, undocumented) |
| Reset times | same bucket `resetTime` | N/A — no supported source |
| Model | stream-json `init.model`, `result.stats.models` keys; ACP `session/new` `models.currentModelId`, `_meta.quota.model_usage[].model`; telemetry `api_response.model` | documented (event names) / source |
| Context window usage | ACP `usage_update{used, size}`: `used` = prompt+output tokens of the last response, `size` = `tokenLimit(model)` from a static table, default 1,048,576 for unknown models (`acpSession.ts:483-489`; `packages/core/src/core/tokenLimits.ts:20-39`). Headless: none (only cumulative tokens) | ACP spec documented; Gemini values source only, label `size` as "table value" |
| Tokens | stream-json `result.stats`; ACP prompt `usage`; telemetry `token.usage`, `api_response`; session file `tokens` | documented / source |
| API-equivalent cost | not computed anywhere in the CLI. Could be derived from tokens x Google's published API prices, shown apart and labelled "estimate" | N/A — no supported source |
| Active agent count | inferred: open `invoke_agent` calls (ACP `tool_call` without final update; stream-json `tool_use` without `tool_result`) | inferred, label as such |
| Concurrency limit | none documented (browser agents: 5, source) | N/A — no supported source |
| Duration | stream-json `result.stats.duration_ms`; telemetry `agent.duration`, `api_response.duration_ms`; ACP: measure in client | documented / measured |
| Errors | stream-json `error` events, `tool_result.status`, exit codes; ACP JSON-RPC errors (429 mapped), `tool_call_update failed`; telemetry `api_error` | documented / source |
| Retries | telemetry `gemini_cli.chat.content_retry(.count)`, `flash_fallback` only; core `Retry` event not forwarded by ACP or headless | documented via OTel only |

## 8. Platform notes

- **Windows native:** supported (Windows 11 24H2+, PowerShell,
  installation.mdx:10-20). npm shim `gemini.cmd` (quoting via cmd.exe). Windows
  native sandbox changes file integrity levels persistently
  (sandbox.md:181-197). Session files under `%USERPROFILE%\.gemini\tmp` (via
  `homedir()`, `storage.ts:92-100`). Recent ConPTY fixes
  (changelogs/latest.md:24-28) show the TUI is used on Windows.
- **WSL2:** a Linux install inside WSL is a separate install with its own
  `~/.gemini` and auth; a Windows dashboard would have to spawn `wsl.exe -e
  gemini --acp` with a `/mnt/c/...` cwd (stdio crosses fine; unverified).
- **Linux:** Docker/Podman, gVisor, LXC sandboxes (sandbox.md:103-261).
- **macOS:** Seatbelt profiles (sandbox.md:86-101), Docker/Podman.
- All: Node >= 20 and an internet connection; Code Assist location limits
  (installation.mdx:17-21); proxy through `HTTPS_PROXY`.

## 9. Answers to amendment 2's research questions

1. **Embed/control?** Yes: `gemini --acp`, documented, JSON-RPC over stdio,
   multi-session, used by Zed/JetBrains (acp-mode.md). Limitations: auth must
   already exist; never call `authenticate` (it writes user settings). Fallback:
   headless `-p --output-format stream-json` + `--resume`.
2. **Streaming?** ACP streams message and thought chunks, tool calls and usage
   (`acpSession.ts:446-491`); stream-json streams message deltas and tool events,
   no thoughts (`nonInteractiveCli.ts:340-369`). Both genuine.
3. **Subagent lifecycle and tools?** Only start and end of `invoke_agent`, no
   internal steps in ACP or stream-json (section 5). Fallbacks: OTel
   `agent.start/finish` (documented) and tailing the subagent chat JSONL under
   `chats/<parentSessionId>/` (undocumented, labelled).
4. **Approvals?** ACP `session/request_permission` with once/session/reject
   options and diffs; mode via `session/set_mode`. Headless has none (ask = deny).
   Clarification questions via `ask_user` are disabled in both; the model asks
   in text. Never pick "all future sessions" options.
5. **Persistence, cancel, resume?** Automatic JSONL recording;
   `session/cancel` -> `stopReason: cancelled`; `session/load` restores and
   replays history; headless `--resume <id>`. Cancel in headless = kill.
6. **Reuse the TUI?** Not as zone B: Ink screen in a PTY is not an API. Keep it
   only as an external "open in Gemini" fallback.
7. **Concurrent subagent output?** ACP serialises a turn's tools; headless can
   run subagents in parallel but reports results as a batch. Demultiplex by
   `toolCallId`/`tool_id`; per-agent live output only from the chat files.
8. **Quota reliably?** Not possible through supported interfaces: daily quota
   lives in an internal Code Assist endpoint shown only by `/stats model`.
   Show 5 h/7 d/reset as N/A, tokens and context from ACP, and a hint
   to `/stats model`. Account eligibility (Antigravity transition) is the
   bigger risk.
9. **OS limits?** Windows: `.cmd` shim quoting, no SIGINT to children (use ACP
   cancel), persistent icacls side effect of the native sandbox. WSL: separate
   install and auth. macOS/Linux: no specific limit for ACP over pipes.

## 10. Open questions for a run (CLI not installed)

1. Does the owner's Google account still get service after 2026-06-18 (Code
   Assist Standard/Enterprise or paid API key), or does `session/new` fail?
2. Exact `gemini.cmd`/`gemini.ps1` names and whether spawning the `.cmd` from Go
   passes multi-line prompts intact (or whether to call `node index.js`).
3. Real ACP transcript: order of `tool_call`/`request_permission`, whether
   `[MODE_UPDATE]` text appears, `usage_update` frequency and values.
4. Does `invoke_agent` (or a per-agent tool name, as the docs say) appear in
   events, and do parallel subagents ever run in headless mode in practice?
5. Live format and write timing of subagent chat files under
   `chats/<parentSessionId>/` (usable for zone C?).
6. Layout of `GEMINI_TELEMETRY_OUTFILE` (JSONL or concatenated JSON) and its
   flush latency.
7. Behaviour of `session/cancel` while a permission request is open, and of
   a new `session/prompt` during a running one.
8. Exit and output when an API error (429, auth) happens in stream-json.
9. Whether `--resume <uuid>` works with ACP-created session ids from the
   command line, and `session/load` with headless-created ids.
10. Start-up time of `gemini --acp` on this machine behind the proxy.
