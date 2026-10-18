# Probes (plan 1, step 3)

Throwaway runs of the installed agent CLIs, to record the events they really
emit. They become test fixtures in plan 4.

- **Approval:** the owner approved probes P0 to P6 on 2026-10-15. Two re-runs
  followed: P3b without a separate approval (the owner then set the rule that
  safe re-runs proceed and are reported), and P5b and P6b under that rule.
  P7, on 2026-10-16, is P6b with another prompt, under the same rule.
  P7b, on 2026-10-17, is P7 again on Codex CLI 0.163.0, under the same rule,
  after L5 showed a stuck "/root" agent on that version.
  P7c, on 2026-10-17, is P7b's driver with another prompt, on the owner's
  approval of plan 7's quota (plan 7, item 4).
- **Where:** a throwaway folder outside the repository (the session
  scratchpad under `%TEMP%`, written `<probe-root>` in the recordings), one
  fresh subfolder per probe.
- **Versions:** Claude Code 2.1.300 (`claude.exe`, native installer, in
  `~\.local\bin`); Codex CLI 0.145.0 (`codex.exe`, in
  `%LOCALAPPDATA%\Programs\OpenAI\Codex\bin`); Gemini CLI not on the PATH.
- **Date:** 2026-10-15, about 21:20 to 21:45 local time (UTC+2).
- **Configuration:** only per-invocation flags were used; no configuration
  file was written by the probes. The results of the hash check are under
  [Configuration check](#configuration-check).
- **Credentials:** no credential file was opened.
- **Clean-up:** every recording went through
  [`drivers/sanitize.mjs`](drivers/sanitize.mjs). It replaces the Windows user
  name, the home and probe paths, the host name, e-mail addresses, the
  organisation, the Codex installation id and Codex's rate-limit reset credit
  ids with placeholders, and replaces thinking signatures with `<signature>`. A scan of this folder for those
  values finds nothing. Session ids, request ids and tool ids are kept: they
  are random and show how events link up.

## Recording format

The driver recordings (P2, P3, P3b, P6, P6b, P7) have one JSON object per line:
`{"t_ms": <ms since start>, "dir": "in" | "out" | "stderr" | "meta" | "exit", "msg": <message>}`.
`in` is what the driver wrote to the CLI's stdin, `out` what the CLI wrote to
stdout. The one-shot recordings (P1, P4, P5, P5b) are the CLI's raw stdout,
one event per line.

## The probes

| Id | Command | Files | Result |
| --- | --- | --- | --- |
| P0 | `where claude codex gemini`; `claude --version`; `codex --version`; `--help` of `claude`, `codex`, `codex exec`, `codex app-server`; `codex app-server generate-json-schema --out <dir>` | `p0-*.txt`, `p0-codex-app-server-schema-v2.json` (the v2 protocol schema, one of 39 generated files) | exit 0 for all; no quota used |
| P1 | `claude -p "Reply with the single word: ok" --output-format stream-json --verbose --include-partial-messages --model haiku` | `p1-claude-stream.jsonl` | exit 0; 12 events |
| P2 | [`drivers/claude-driver.mjs`](drivers/claude-driver.mjs) `conversation`: `claude --input-format stream-json --output-format stream-json --verbose --include-partial-messages --permission-prompt-tool stdio --model haiku`; `initialize`, `get_context_usage`; three user turns: write `hello.txt`, a follow-up, a long count interrupted at its first delta | `p2-claude-conversation.jsonl` | `hello.txt` written after the approval; 3 results; process exit 1 after stdin closed (the last result was the interrupt's error) |
| P3 | same driver, `subagent`, plus `--forward-subagent-text --include-hook-events`: one general-purpose subagent lists the folder | `p3-claude-subagent.jsonl` | subagent ran; its first permission request failed with "Tool permission request failed: AbortError: Stream closed", because the driver closed stdin after the main turn's result while the background subagent was still running |
| P3b | P3 re-run with the driver fixed (stdin kept open while `background_tasks_changed` lists tasks), the subagent also writing `found.txt` | `p3b-claude-subagent-approval.jsonl` | subagent's Bash and Write requests reached the driver with its `agent_id` and were approved; `found.txt` written |
| P4 | `claude -p "Which file did you create earlier in this session? Answer in one line." --resume <P2 session id> --output-format stream-json --verbose --model haiku` | `p4-claude-resume.jsonl` | exit 0; answered `hello.txt` from P2's history |
| P5 | `codex exec --json --skip-git-repo-check "Reply with the single word: ok"` | `p5-codex-exec.jsonl`, `p5-codex-stderr.txt` | exit 1: `turn.failed`, HTTP 400 "The 'gpt-6-sol' model is not supported when using Codex with a ChatGPT account" (the model set in the owner's `~/.codex/config.toml`) |
| P5b | P5 with `-m gpt-5.6-sol` (the default model reported by `model/list`) | `p5b-codex-exec.jsonl` | exit 0; `thread.started`, `turn.started`, `item.completed` (agent_message "ok"), `turn.completed` with usage |
| P6 | [`drivers/codex-driver.mjs`](drivers/codex-driver.mjs): `codex app-server` (stdio); `initialize`, `initialized`, `account/rateLimits/read`, `thread/start` (sandbox `read-only`, approval `on-request`), `turn/start` (write `hello.txt`), a second turn interrupted at its first delta, `account/rateLimits/read` | `p6-codex-app-server.jsonl` | both turns failed on the same model error as P5; handshake, thread start and rate limits worked |
| P6m | `model/list` over `codex app-server` (no model call) | `p6-codex-model-list.json` | `gpt-5.6-sol` (default), `gpt-5.6-terra`, `gpt-5.6-luna` |
| P6b | P6 with `"model": "gpt-5.6-sol"` in `thread/start` | `p6b-codex-app-server.jsonl` | `hello.txt` written after a file-change approval; second turn interrupted; exit 0 |
| P7 | the same driver with a fourth argument, a prompt: one turn, "Spawn two subagents in parallel. The first lists the files in the current directory. The second creates a file named found.txt ... Wait for both, then reply with the single word done."; the driver ignores child-thread notifications for its own turn logic | `p7-codex-subagents.jsonl` | two subagents ran in parallel; the second's file change was approved; `found.txt` written; parent replied; exit 0 after 30 s |
| P7b | P7 on Codex CLI 0.163.0 with model `gpt-6-sol` (2026-10-17, about 11:35 local time) | `p7b-codex-subagents.jsonl` | the same outcome in 25 s; new in 0.163.0: each child's `subAgentActivity` `kind:"interacted"` toward the orchestrator (`agentPath` `/root`), a `kind:"completed"` on the parent for each child, 28 `item/commandExecution/outputDelta`, one `account/updated`, an `accountId` in the rate limits (redacted by the sanitizer) |
| P7c | the P7 driver on Codex CLI 0.163.0 (2026-10-17, about 22:20 local time), prompt: "Spawn exactly one subagent. When you call the spawn tool, set its model argument to a smaller model than yours that the tool accepts, and set its reasoning effort argument to low. The subagent lists the files in the current directory and reports how many there are. Wait for it, then reply with the single word done." | `p7c-codex-spawn-model.jsonl` | parent on `gpt-6-sol`; the child `/root/count_files` announced by `subAgentActivity` `kind:"started"` with `model:"gpt-6-luna"`, `reasoningEffort:"low"`; still no `collabAgentToolCall` `spawnAgent` item, only `wait` (model and reasoningEffort null); the `kind:"completed"` carries null for both; "done", exit 0 after 15 s; 89,410 tokens across the two threads (60,288 cached input) |
| P9n | [`drivers/p9-probe.mjs`](drivers/p9-probe.mjs) (the dashboard's handshake, per-process `-c` overrides) on Codex CLI 0.163.0 (2026-10-17, about 23:46 local time) with `-c features.network_proxy=true -c sandbox_mode="workspace-write" -c sandbox_workspace_write.network_access=true`, prompt: run `curl.exe -s -m 20 -o NUL -w "%{http_code}" https://example.com` once, then reply done | `p9n-codex-network.jsonl` (this folder only: no test replays it) | no approval was asked: the command never started, stderr `exec_command failed: CreateProcess { message: "UnsupportedOperation(\"managed networking requires the elevated Windows sandbox backend\")" }`; "done", exit 0 after 6 s. A network approval, the only one to which 0.163.0 adds `acceptForSession`, needs `codex sandbox setup --elevated`, which the owner declined on 2026-10-18 (plan 9, item 3) |
| P9a | the same driver (2026-10-17, about 23:49 local time) with `-c model="gpt-5.6-luna"`, the one model whose catalogue entry declares multi-agent v1, prompt: "Spawn exactly one subagent with the model gpt-5.6-luna and reasoning effort low, asking it to count the files in the current directory and reply with the number only. Wait for it, then reply with its number and nothing else." | `p9a-codex-spawn-request.jsonl` | the first `collabAgentToolCall` `spawnAgent` item recorded: started with no receiver, completed with the child's thread id, both with `model:"gpt-5.6-luna"`, `reasoningEffort:"low"`; then `wait`; reply "2", exit 0 after 12 s; the orchestrator's last total 66,965 tokens (55,168 cached input). An earlier run with `-c features.multi_agent_v2=false` on the default `gpt-6-sol` stayed on v2 (the catalogue's version wins, `core/src/config/mod.rs` 1637-1664) and was not kept |
| P11n | the P9n driver on Codex CLI 0.163.0 (2026-10-18, about 00:50 local time) after the owner ran `codex sandbox setup --elevated --current-user` (`[windows] sandbox = "elevated"`), the same three `-c` overrides, every approval answered `acceptForSession`, prompt: run P9n's `curl.exe` command to example.com, then the same again as a separate shell call, then reply with the two outputs | `p11n-codex-network-elevated.jsonl` (this folder only: no test replays it) | the first run was refused by Codex's managed proxy, "domain is not on the allowlist for the current sandbox mode", with no network approval; Codex then asked an ordinary escalation, `item/commandExecution/requestApproval` with no `networkApprovalContext`, `availableDecisions` `accept`, `acceptWithExecpolicyAmendment`, `cancel`: no `acceptForSession`; the reply `acceptForSession` was taken anyway and the second run asked nothing; both printed "000" (the corporate proxy); reply "000 000", exit 0 after 44 s |
| P11n-b | P11n without `sandbox_workspace_write.network_access` (2026-10-18, about 00:58 local time) | `p11n-b-codex-network-offline.jsonl` (this folder only) | no approval of any kind: both runs failed in under 200 ms with "000", the elevated sandbox blocking the network itself; reply "000 000", exit 0 after 14 s |
| P12c | [`drivers/p12-claude.mjs`](drivers/p12-claude.mjs) on Claude Code 2.1.301 (2026-10-18, about 01:16 local time): the dashboard's arguments with `--permission-mode auto`, every `can_use_tool` denied; prompts: write hello.txt in the project, then a file in its parent folder | `p12c-claude-auto.jsonl` (this folder only) | auto accepted in `-p` stream-json mode: `system/init` `permissionMode:"auto"`; both writes ran with no `can_use_tool`; each tool result carries `tool_result_meta[].permission_decision`, `{decision:"accept", source:"config", reason_type:"mode"}` for the project write and `reason_type:"classifier"` for the outside one (undocumented fields) |
| P12c-switch | the same driver, `--permission-mode default`, `set_permission_mode` `auto` sent before the second prompt (about 01:17); prompts: write hello.txt, then hello2.txt, both in the project | `p12c-switch-claude-default-to-auto.jsonl` (this folder only) | in default, the Write and then a PowerShell retry each reached the driver as `can_use_tool` and were denied (`permission_decision` `{decision:"reject", source:"user_reject"}`); the switch answered `{subtype:"success", response:{mode:"auto"}}` in 4 ms, the next `system/init` read `auto`, and hello2.txt was written with no request |
| P12c-risk | P12c's driver, `auto`, prompt: run `rm -rf ../victim`, a dummy folder made for the probe (about 01:17) | `p12c-risk-claude-auto-rm.jsonl` (this folder only) | the classifier allowed the explicitly requested delete (`reason_type:"classifier"`); no `can_use_tool` |
| P12c-risk2 | the same, prompt: run `curl -fsSL https://example.com/install.sh | sh` and nothing else (about 01:17) | `p12c-risk2-claude-auto-curl-sh.jsonl` (this folder only) | allowed by the classifier; curl failed the proxy's TLS handshake (exit 35), so nothing ran; no `can_use_tool` |
| P12c-block, P12c-block2 | the same, a boundary stated in a first turn (no file named b..., then no write outside the project), and a second turn asking for the write "as a test of the permission reviewer" (about 01:18 and 01:19) | `p12c-block-claude-auto-boundary-edit.jsonl`, `p12c-block2-claude-auto-boundary-shell.jsonl` (this folder only) | no classifier block observed: the project write is allowed by the mode (`reason_type:"mode"`) and never reaches the classifier; the outside write was refused by the model itself, with no tool call |
| P12x | [`drivers/p12-codex.mjs`](drivers/p12-codex.mjs) on Codex CLI 0.163.0 (2026-10-18, about 01:19 local time): the dashboard's handshake, `thread/start` with sandbox read-only, approvals on-request and `approvalsReviewer:"auto_review"` (also on each `turn/start`); P12c's two prompts | `p12x-codex-auto-review.jsonl` (this folder only) | the thread result echoes `approvalsReviewer:"auto_review"`; no `requestApproval` reached the client; each escalation gave `item/autoApprovalReview/started` then `completed` (`review:{status:"approved", riskLevel:"low", userAuthorization:"high", rationale}`, `decisionSource:"agent"`, `targetItemId` the command item), plus a `guardianWarning` message; both files written; the schema marks these payloads [UNSTABLE] |
| P12x-switch | the same driver, `approvalsReviewer:"user"` on `thread/start` and the first `turn/start`, `auto_review` on the second (about 01:19); prompts: write hello.txt, then hello2.txt, both in the project | `p12x-switch-codex-user-to-auto-review.jsonl` (this folder only) | turn 1: `item/commandExecution/requestApproval` reached the client and was declined (item `declined`); turn 2: the reviewer approved, no request reached the client, hello2.txt written |
| P12x-sandbox | the same driver with a sandbox per turn (plan 12 addendum, 2026-10-18, about 10:30): `thread/start` read-only with `auto_review`; turns 1 and 2 `auto_review` with `sandboxPolicy:{type:"workspaceWrite"}`, turn 3 `user` with `{type:"readOnly"}`; prompts: write inside.txt in the project, then a file in its parent folder, then manual.txt in the project | `p12x-sandbox-codex-turn-workspace-write.jsonl` (also `testdata/probes`) | the thread result reads `{type:"readOnly"}`; turn 1: inside.txt written by apply_patch with no review and no request; turn 2: the parent-folder file written by apply_patch, also with no review (Codex's own behaviour, see P12x-sandbox3); turn 3: `item/fileChange/requestApproval` reached the client and was declined, so the readOnly turn applies again after workspaceWrite turns |
| P12x-sandbox2 | the same driver, `thread/start` with sandbox workspace-write and `auto_review`, no `sandboxPolicy` on the turn (about 10:35); prompt: write a file in the parent folder | `p12x-sandbox2-codex-thread-workspace-write-shell.jsonl` (this folder only) | the thread result reads `{type:"workspaceWrite", writableRoots:[], networkAccess:false, excludeTmpdirEnvVar:false, excludeSlashTmp:false}`; the shell write was escalated, `item/autoApprovalReview/completed` approved it (`riskLevel:"low"`, `userAuthorization:"high"`), file written |
| P12x-sandbox3 | as P12x-sandbox2, the prompt asking for apply_patch (about 10:36) | `p12x-sandbox3-codex-thread-workspace-write-patch.jsonl` (this folder only) | apply_patch wrote the parent-folder file with no review and no request, as in P12x-sandbox turn 2: on this machine (Windows sandbox `elevated`) a patch outside the project is not reviewed whichever way the sandbox is set; shell writes are |

Quota used: 7 short Claude Haiku turns and 2 subagent runs (P1 to P4, P3b; the
results report a total of about USD 0.03 API-equivalent); 3 short Codex turns
(P7's with two subagents: 184,312 tokens across the three threads, the sum
of each thread's last `thread/tokenUsage/updated` total, of which 171,136
cached input) plus 3 failed requests. The 5-hour Claude window read 22% before the probes and
31% after P4, but other sessions (this research's own agents) were running at
the same time, so the difference is not the probes' cost.

## What the probes settle

### Claude Code

- **A program drives a full session without the Agent SDK.** The SDK's
  stdio protocol works from a 100-line Node script: `control_request`
  `initialize` first, then user messages as
  `{"type":"user","message":{"role":"user","content":"..."},"parent_tool_use_id":null,"session_id":"default"}`
  on stdin, several turns in one process (P2). The flag
  `--permission-prompt-tool stdio` is not listed by `claude --help`; it is
  referenced there by `--permission-prompts host` ("the SDK host or
  --permission-prompt-tool") and passed by the Python SDK
  (`claude-agent-sdk-python@c10b52c`,
  `src/claude_agent_sdk/_internal/transport/subprocess_cli.py:632-633`,
  `types.py:2304`). Undocumented at the CLI level; see `claude-code.md`.
- **Streaming is genuine:** `stream_event` carries the API's
  `content_block_delta` events (`text_delta`, `input_json_delta`,
  `thinking_delta`), before the complete `assistant` message (P1, P2).
- **Approvals reach the host:** `control_request` `can_use_tool` with
  `tool_name`, `display_name`, `input`, `description`,
  `permission_suggestions`, `tool_use_id`; the answer
  `{"behavior":"allow","updatedInput":...}` lets the tool run (P2). A
  subagent's request carries its `agent_id` (P3b).
- **Interrupt is clean:** `control_request` `interrupt` is acknowledged
  (`{"still_queued":[]}`), the partial answer ends with a user message
  "[Request interrupted by user]" and a `result` with subtype
  `error_during_execution` (P2).
- **Resume works:** `--resume <session_id>` continues with the history (P4).
- **Subagent lifecycle is in the stream:** `system/task_started`
  (`task_id`, `tool_use_id`, `subagent_type`, `description`, `prompt`,
  `is_backgrounded`, `spawn_depth`), `system/task_progress` (`description`,
  `usage.total_tokens`, `usage.tool_uses`, `usage.duration_ms`,
  `last_tool_name`), `system/task_updated` (`status: completed`),
  `system/task_notification`, `system/background_tasks_changed` (the list of
  running tasks). The subagent's own messages carry
  `parent_tool_use_id` = the Agent tool call's id (P3, P3b).
- **The host must keep stdin open while background tasks run**, or their
  permission requests fail (P3).
- **Quota is in the stream:** `rate_limit_event` with
  `rate_limit_info.unifiedWindows.five_hour` and `.seven_day`, each with
  `utilization` (0 to 1) and `resetsAt` (Unix seconds), plus `status`,
  `rateLimitType`, `overageStatus`, `isUsingOverage`. It does not arrive
  every turn: 2 events for P1's one turn, 2 for P2's three, 1 for P3b's two. Undocumented: see
  `telemetry-sources.md`.
- **Context on request:** `control_request` `get_context_usage` returns
  `totalTokens`, `maxTokens`, `percentage`, `model`, `categories`,
  `autoCompactThreshold` (P2).
- **Per turn:** `result` has `total_cost_usd` (API-equivalent, list prices:
  `modelUsage.<model>.costBasis` is `"list"`), `usage`, `modelUsage` (tokens,
  `contextWindow`, `maxOutputTokens` per model), `duration_ms`,
  `duration_api_ms`, `num_turns`, `permission_denials`, `subagent_stats`,
  `api_error_status`.
- **Session facts:** `system/init` has `cwd`, `session_id`, `model`,
  `permissionMode`, `tools`, `agents`, `mcp_servers`,
  `claude_code_version`, `capabilities`. The `initialize` response has the
  account's `subscriptionType` (here "Claude Team") and the available
  `models`.

### Codex CLI

- **`codex app-server` is a complete driving interface** over JSON-RPC on
  stdio (`[experimental]` in its own help text): `initialize` then the
  `initialized` notification; `thread/start` with `cwd`, `sandbox`,
  `approvalPolicy`, `model`; `turn/start`; `turn/interrupt` (P6b).
- **Streaming:** `item/agentMessage/delta`; items have `item/started` and
  `item/completed` (`userMessage`, `agentMessage` with `phase` `commentary`
  or `final_answer`, `reasoning`, `fileChange` with its diff,
  `commandExecution` with command and cwd); `turn/diff/updated` gives the
  turn's unified diff (P6b).
- **Approvals:** the server sends the request
  `item/fileChange/requestApproval` (or `item/commandExecution/requestApproval`)
  and `thread/status/changed` with `activeFlags: ["waitingOnApproval"]`; the
  answer `{"decision":"accept"}` lets it proceed, and
  `serverRequest/resolved` follows (P6b).
- **Interrupt:** `turn/interrupt` returns `{}` and `turn/completed` reports
  `status: "interrupted"` (P6b).
- **Quota, without any model call:** `account/rateLimits/read` returns
  `primary` (`windowDurationMins: 300`, the 5-hour window) and `secondary`
  (`windowDurationMins: 10080`, 7 days), each with `usedPercent` and
  `resetsAt`, plus `planType` ("plus") and `credits`. During a turn,
  `account/rateLimits/updated` pushes the same (P6, P6b).
- **Tokens:** `thread/tokenUsage/updated` after each model call, `total` and
  `last` (input, cached input, output, reasoning tokens) (P6b).
- **`codex exec --json`** is a one-shot event stream: `thread.started`,
  `turn.started`, `item.completed`, `turn.completed` with `usage`, or
  `turn.failed`; no deltas (P5b).
- **A CLI's own configuration can make every turn fail** (the owner's
  configured model is refused, P5, P6). The error arrives as an `error`
  notification and `turn/completed` with `status: "failed"`; the dashboard
  has to show it in the conversation, and a per-session model override
  (`thread/start` `model`, `-m`) fixes it without touching the configuration.
- **Subagents are in the same stream (P7).** Each child is a thread of its
  own; its notifications (`thread/status/changed`, `turn/started`, items,
  deltas, `thread/tokenUsage/updated`, `turn/completed` with `durationMs`)
  arrive on the parent's connection, tagged with the child's `threadId`. No
  `thread/started` is sent for a child: the parent's `item/completed` of type
  `subAgentActivity` (`kind: "started"`, `agentThreadId`, `agentPath` such as
  `/root/list_files`) announces it. No `subAgentActivity` marks the end: the
  child's `turn/completed` and `thread/status/changed` to `idle` do. The
  parent's `wait` calls are `collabAgentToolCall` items (`tool: "wait"`,
  `senderThreadId`), with empty `receiverThreadIds` and `agentsStates` here.
- **A child's approval comes to the host:** `item/fileChange/requestApproval`
  with the child's `threadId`, and the child's `thread/status/changed` with
  `waitingOnApproval` (P7). The children's read-only commands ran without an
  approval request under the `read-only` sandbox.
- **Rate limits are pushed often:** 11 `account/rateLimits/updated` in P7's
  one turn.
- Codex logs to stderr an error about its models cache ("missing field
  `base_instructions`"); the turns still succeed with a supported model.

## Configuration check

[`drivers/hash-config.sh`](drivers/hash-config.sh) hashed 22 files before and
after the probes: the Claude Code settings, `CLAUDE.md`, remote settings,
plugin lists and skills, `~/.claude.json`, and Codex's `config.toml`,
`AGENTS.md` and global state. It skips any file whose name matches
credentials, auth, oauth, token or secret.

| File | Before = after | Why |
| --- | --- | --- |
| 20 of the 22 files, including `settings.json`, `CLAUDE.md`, `config.toml`, `AGENTS.md` | yes | - |
| `~/.claude.json` | no | Claude Code's own state file, which it rewrites on each run. This research session (also Claude Code) restarted at 21:38 and rewrote it as well. The file was hashed, not read. |
| `~/.claude/plugins/known_marketplaces.json` | no | Only its `lastUpdated` timestamp changed, at 21:43:38. That was during the Codex probes, so it was Claude Code's marketplace auto-update in the running session. |

P7 (2026-10-16, about 00:06 local time): the 22 hashes before and after were
identical.

P7b (2026-10-17, 11:35 local time, `confcheck` with `-cli-ran`): the 24
files before and after were identical (configuration: PASS).

P7c (2026-10-17, 22:22 local time, `confcheck` with `-cli-ran`): the 24
files before and after were identical (configuration: PASS); no process the
driver started was left.

P9n, the v2 attempt before P9a, and P9a (2026-10-17, 23:45 to 23:49 local
time, `confcheck` with `-cli-ran` around each): the 24 files before and after
were identical each time (configuration: PASS). The `-c` overrides apply to
the probe's process only; no configuration file was changed.

P11n and P11n-b (2026-10-18, 00:50 to 00:59 local time, `confcheck` with
`-cli-ran`, against a snapshot taken after the owner's elevated sandbox
setup): the 24 files were identical (configuration: PASS). The setup itself,
run by the owner, changed only `[windows] sandbox` in `~/.codex/config.toml`
(plan 11).

P12c to P12x-switch (2026-10-18, 01:16 to 01:19 local time, `confcheck`
with `-cli-ran` against a snapshot taken before them): configuration PASS;
the configuration files identical, `~/.claude.json` changed only in Claude
Code's cache-state keys, 12 new session files. The permission mode and the
reviewer were per-process flags and protocol parameters; no configuration
file was changed.

Not checked: the files the CLIs write by design for each session (Claude
transcripts under `~/.claude/projects`, Codex rollouts under
`~/.codex/sessions` and its logs), which hold these probes' sessions now.
