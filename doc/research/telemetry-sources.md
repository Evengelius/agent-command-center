# Telemetry sources for zone D

Plan 1, step 4. For each agent CLI in scope (Claude Code 2.1.300, Codex CLI
0.145.0), every way found to get zone D's values, documented or not, with its
source, and what is simply unavailable, shown as "N/A". Gemini CLI is out of
scope since amendment 5. What plan 1 left unprobed has its outcome in
[reliability.md, Not verified](../reliability.md#not-verified) (plan 14).

Conventions are those of [`integration.md`](integration.md): **CC-n** and
**CX-n** are sections of [`claude-code.md`](claude-code.md) and
[`codex-cli.md`](codex-cli.md), whose source keys (doc `page#section`, PY, CC,
[T], [D:page]) are kept; **Pn line k** is line k of a probe recording in
[`probes/`](probes/README.md). "Documented", "undocumented" and "inferred" as
in those files.

## Rules for every value

These follow from AGENTS.md ("Real values or N/A", "Keep failures visible")
and apply to both CLIs:

- **N/A until a source gives the value.** Never zero, never a guess; quota is
  never estimated from tokens.
- **Every pushed value carries its age.** Quota and context arrive on events,
  not on a schedule; the panel shows when the value was last received.
- **Subscription usage and API-equivalent cost are separate rows.** Quota
  percentages come from the provider's servers; Claude Code's cost is a
  client-side estimate at list price, labelled "API-equivalent estimate",
  never "spent" (doc `costs#using-the-usage-command`; doc
  `agent-sdk/cost-tracking`).
- **Observed, inferred and AI-reported values are labelled distinctly**
  (AGENTS.md, "The conversation is real"). A value derived by the dashboard
  (an agent count, a session total) is labelled "derived".
- **A window passes its reset time:** drop the value and show N/A until a new
  one arrives, as Claude Code does for its status line (doc
  `statusline#available-data`).

## Claude Code

Primary feed: the stream-json stdout of the long-lived `claude -p` process
(integration.md, question 1). Other ways, kept as fallbacks or rejected:

| Way | What it gives | Status | Use |
| - | - | - | - |
| stream-json messages and control responses | everything in the table below | documented messages; control requests documented as SDK methods, wire form in SDK code (CC-4, way A) | primary |
| `/usage` sent as a prompt | `usage_report` with quota rows | documented, experimental (doc `agent-sdk/typescript#sdkusagereport`); unprobed over stream-json (CC-10, question 9) | on-demand quota refresh, after a probe |
| Status line JSON | model, context, cost, both quota windows, per-subagent rows | documented (doc `statusline#available-data`), but only in the interactive TUI and only with a `statusLine` setting (CC-6) | not available to a `-p` session; field names are the model for zone D |
| OpenTelemetry | `claude_code.token.usage`, `claude_code.cost.usage`, `api_error`, `api_retries_exhausted`, `subagent_completed`, ... | documented (doc `monitoring-usage`), enabled per process by environment variables, no configuration written | not needed for the MVP: no quota metric, and it duplicates the stream (CC-6) |
| Hooks | per-tool and subagent start and stop events | documented (doc `hooks#hook-events`); raw-client behaviour unprobed (CC-10, question 5) | not needed for telemetry |
| Transcript files `~/.claude/projects/...` | per-message usage | path documented, format "internal" (doc `sessions#where-transcripts-are-stored`) | last resort only, labelled undocumented |

| Value | Source | Status and evidence | N/A when |
| - | - | - | - |
| 5-hour quota | `rate_limit_event.rate_limit_info.unifiedWindows.five_hour.utilization` (0-1); also `rateLimitType: "five_hour"` with `status` (`allowed`, `allowed_warning`, `rejected`); `usage_report` row `kind: "session"` | `rateLimitType`, `status`, `resetsAt`: documented (doc `agent-sdk/python`, `RateLimitInfo`; doc `agent-sdk/typescript#sdkratelimitevent`). `unifiedWindows`: undocumented, observed in P1 lines 3 and 11 (`five_hour.utilization` 0.22 then 0.23). Emitted on transitions, not every turn: 2 events for P1's one turn, 2 for P2's three, 1 for P3b's two (CC-10, question 8) | before the first event; with an API key (no subscription window); after the reset time passes |
| 7-day quota | `unifiedWindows.seven_day.utilization`; `rateLimitType` `seven_day`, `seven_day_opus`, `seven_day_sonnet`; `usage_report` rows `weekly_all`, `weekly_scoped` | as above; P1 line 3 (`seven_day.utilization` 0.45). Classify `usage_report` rows on `kind`, never on labels (doc `#sdkusagereport`) | as above |
| Reset times | `resetsAt` (Unix seconds) per window; `usage_report ... resets_at` (ISO 8601) | documented; P1 line 3 | with the window |
| Overage / extra usage | `overageStatus`, `overageDisabledReason`, `isUsingOverage`; `usage_report.rate_limits.extra_usage` | documented (Python reference); P1 line 3 (`overageStatus: "rejected"`, `org_level_disabled`) | not reported; shown apart from plan quota |
| Model | `system/init.model`; `assistant.message.model` per message, subagents' forwarded messages included; `result.modelUsage` keys | documented; P1 (`claude-haiku-5-5`) | a subagent whose text is not forwarded: its model is N/A |
| Context | `get_context_usage` control request → `totalTokens`, `maxTokens`, `percentage`, `model`, `categories`, `autoCompactThreshold`; or derived from the last main-thread `assistant.message.usage` over `modelUsage[model].contextWindow` | request: undocumented at CLI level, documented as SDK `getContextUsage()`; observed P2 lines 4-6. `contextWindow` 1,000,000 for the model in P1 line 12. Whether the request costs anything: not measured (CC-10, question 10) | before the first turn, unless `get_context_usage` is called |
| Tokens | per step `assistant.message.usage` (deduplicated by `message.id`); per turn `result.usage`; running total including subagents `result.modelUsage[*]` (`inputTokens`, `outputTokens`, `cacheReadInputTokens`, `cacheCreationInputTokens`, `thinkingTokens`); per subagent `task_progress.usage.total_tokens` | documented (doc `agent-sdk/cost-tracking`, doc `agent-sdk/typescript#modelusage`); P1 line 12. In streaming input mode read the latest `result`, do not sum (doc `agent-sdk/cost-tracking`) | before the first `result` |
| API-equivalent cost | `result.total_cost_usd` (running total of the process), `modelUsage[*].costUSD`, `costBasis` | documented; P1 line 12 (`total_cost_usd` 0.00249374, `costBasis: "list"`). An estimate at list price: never mixed with quota | before the first `result` |
| Agent count | derived: tasks in the last `background_tasks_changed.tasks` (non-ambient, `task_type` `local_agent` or `local_workflow`) plus foreground `task_started` without a terminal `task_updated` or `task_notification` | documented messages (doc `#sdkbackgroundtaskschangedmessage`); P3, P3b. Reset when the CLI process restarts | never: 0 is a real value once the stream is up; N/A before |
| Concurrency limit | not reported by the CLI. Documented defaults: 20 subagents (`CLAUDE_CODE_MAX_CONCURRENT_SUBAGENTS`), 16 workflow agents or fewer on fewer CPUs | doc `sub-agents#concurrent-subagent-limit`, doc `workflows#behavior-and-limits` | always N/A in the MVP: a managed setting, the owner's environment or the CPU count can change the effective value, and the dashboard sets no orchestration settings (amendment 4). A documented default may be shown only labelled "default, not reported" |
| Duration | `result.duration_ms`, `duration_api_ms`; `task_*.usage.duration_ms`; `tool_progress.elapsed_time_seconds`; session wall time measured by the dashboard from process start | documented; P1 line 12 (`duration_api_ms` 2837). Wall time: observed, labelled so | before the first `result` |
| Errors | `result.is_error`, `subtype`, `errors`, `api_error_status`; `assistant.error` (`rate_limit`, `overloaded`, `authentication_failed`, ...); `system/permission_denied`; `tool_result.is_error`; `task_notification.status: "failed"` | documented; P2 (`result` subtype `error_during_execution` after an interrupt, which is not an error to count) | never: a count starts at 0 once the stream is up |
| Retries | `system/api_retry` (main loop); `tool_progress.subagent_retry` (subagents) | documented (doc `headless#handle-api-retries`, doc `#sdktoolprogressmessage`); not observed in the probes | as errors |

Sources for the table: CC-6 and CC-7, with the documentation pages and
probe lines they cite.

## Codex CLI

Primary feed: the JSON-RPC stream of the long-lived `codex app-server`
process (integration.md, question 1). Other ways:

| Way | What it gives | Status | Use |
| - | - | - | - |
| app-server notifications and requests | everything in the table below | documented [D:app-server], labelled experimental | primary |
| `codex exec --json` | per-turn `usage` on `turn.completed`; errors | documented [D:noninteractive]; P5, P5b | fallback driver only: no quota, no context, no deltas (CX-4.2) |
| Rollout files `$CODEX_HOME/sessions/.../rollout-*.jsonl` | `token_count` events with token totals, `model_context_window` and `rate_limits` | undocumented ([T] `codex-rs/rollout/src/recorder.rs:1528-1546`; `codex-rs/protocol/src/protocol.rs:2059-2081`) | last resort, labelled undocumented |
| OpenTelemetry `[otel]` | `codex.api_request`, `codex.sse_event`, `codex.tool_result`, `codex.turn_ttft`, ... | documented [D:config-advanced]; off by default; enabling per run with `-c otel.exporter=...` writes no file but needs an OTLP receiver | not needed for the MVP |
| `notify`, hooks | turn-complete notifications; tool and subagent events | documented [D:config-advanced], [D:hooks]; both need a configuration change | rejected: AGENTS.md forbids changing the CLI's configuration |
| `account/usage/read` | account token activity, daily buckets | in the protocol ([T] `common.rs:1056-1060`, `v2/account.rs:392-395`); not probed | possible later; not a zone D value |

| Value | Source | Status and evidence | N/A when |
| - | - | - | - |
| 5-hour quota | `account/rateLimits/read` → `rateLimits` window with `windowDurationMins: 300`, `usedPercent`; pushed as `account/rateLimits/updated` after model calls | documented [D:app-server]; [T] README:2206-2247, `v2/account.rs:294-300,518-534`; works without a turn (P6 line 7, P6b line 7); 11 pushes during P7's single parent turn. Identify windows by `windowDurationMins`, not by `primary` or `secondary` ([T] `codex-rs/tui/src/chatwidget/rate_limits.rs:77-104`) | signed in with an API key ("chatgpt authentication required to read rate limits", [T] `request_processors/account_processor.rs:1016-1030`); a plan with no 5-hour limit ([D:pricing]: Pro plans "currently have no five-hour limit"); the read failed (shown with its error) |
| 7-day quota | the window with `windowDurationMins: 10080` | as above; P6b line 7 | as above |
| Reset times | `resetsAt` (Unix seconds) per window | as above | with the window |
| Plan and credits | `planType`, `credits {hasCredits, unlimited, balance}`, `rateLimitReachedType`, `rateLimitResetCredits` | P6b line 7; [T] `v2/account.rs:525-534,631-636`. `rateLimitResetCredits` ids are account-linked: never logged or shown | API key |
| Model | `thread/start` response `model`, `reasoningEffort`; `model/rerouted` when the backend switches | P6b line 9; P7 (`gpt-5.6-sol`, the per-session model, since the configured one is refused, CX-3.3); [T] README:1442 | a subagent's model: no event of P7 reported it (the schema's `collabAgentToolCall.model` belongs to the `spawnAgent` call P7 did not emit, CX-5): N/A |
| Context | `thread/tokenUsage/updated.tokenUsage.last.totalTokens` over `modelContextWindow`, per thread | P6b line 41; P7 (258,400 for every thread); [T] `v2/thread.rs:1459-1522`. The TUI subtracts a 12,000-token baseline ([T] `codex-rs/protocol/src/protocol.rs:2215-2262`; `codex-rs/tui/src/status/card.rs:326-333`): show the raw value, or the TUI's, labelled | before a thread's first model call |
| Tokens | `thread/tokenUsage/updated` `total` and `last`: `totalTokens`, `inputTokens`, `cachedInputTokens`, `cacheWriteInputTokens`, `outputTokens`, `reasoningOutputTokens`, per thread; replayed on `thread/resume` ([T] README:321, not probed) | P6b line 41; P7: one series per thread (parent 76,521, children 30,455 and 77,336 total tokens at the end). Session total: derived, the sum over threads | before the first update |
| API-equivalent cost | **N/A: no supported source.** Codex reports no cost field; [D:pricing] gives credits per million tokens, not USD | CX-7 | always. A USD estimate would need an external price table, outside the MVP |
| Agent count | derived: subagent threads announced by `subAgentActivity` (`kind: "started"`) whose last `thread/status/changed` is `active` | P7; [T] `v2/item.rs:1153-1189`; labelled derived | before the session starts |
| Concurrency limit | `agents.max_threads` from `config/read` (read-only) when the owner set it | [T] `codex-rs/config/src/config_toml.rs:679-700`. Built-in defaults (6 for the V1 agent backend, 4 for V2, [T] `codex-rs/core/src/config/mod.rs:206-207,268`) are not reported by any API, and which backend is active is not reported either (P7) | when not set: N/A, or the default labelled "source-code default, not reported" |
| Duration | `turn/completed.turn.durationMs`, `startedAt`, `completedAt`, per thread; `commandExecution.durationMs` | P6b lines 56 and 68 (`durationMs` 30628); P7 (children's `turn/completed`) | before the first turn ends |
| Errors | `error` notification; `turn/completed` `status: "failed"` with `codexErrorInfo` (`ContextWindowExceeded`, `UsageLimitExceeded`, `HttpConnectionFailed`, ..., `other`) | P6 lines 23-24 (`codexErrorInfo: "other"` for the refused model); [T] README:1503-1524 | never: a count starts at 0 once the stream is up |
| Retries | `error` notifications with `willRetry: true` | [T] `v2/notification.rs:41-47`; not observed in the probes | as errors |

Sources for the table: CX-6 and CX-7, with the code, documentation and
probe lines they cite; the P7 figures were recounted from
`p7-codex-subagents.jsonl` for this file.

## Side by side

| Value | Claude Code | Codex CLI |
| - | - | - |
| 5-hour quota | `rate_limit_event` (on transitions) | `account/rateLimits/read`, any time; pushes |
| 7-day quota | `rate_limit_event` | `account/rateLimits/read` |
| Reset times | `resetsAt` | `resetsAt` |
| Model | `system/init`, per message | `thread/start` response; subagents N/A |
| Context | `get_context_usage`, or derived | `thread/tokenUsage/updated`, per thread |
| Tokens | `result.modelUsage` running total | per-thread totals; session total derived |
| API-equivalent cost | `result.total_cost_usd` (estimate) | N/A |
| Agent count | derived from task events | derived from thread events |
| Concurrency limit | N/A (default documented, not reported) | `config/read` if set, else N/A |
| Duration | `result.duration_ms` | `turn/completed.durationMs` |
| Errors | `result`, `assistant.error`, task status | `error`, failed `turn/completed` |
| Retries | `system/api_retry`, `subagent_retry` | `error.willRetry` |

## Open points for plan 2

- Claude Code: `/usage` over stream-json, its cost and its rate limit (CC-10,
  question 9); whether `get_context_usage` costs anything (CC-10, question 10).
- Codex: a stable source for the context window before the first turn (CX-10,
  question 7); whether `config/read` reports `agents.max_threads` when unset.
- Both: the refresh policy for on-demand reads (startup, after each turn, on
  a key), set so that it never adds model calls.
