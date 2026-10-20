# Events and state

Plan 2, 2026-10-16. How the events of each agent CLI become the normalised
events of the application, the state they update, the state machines that
govern it, and where polling is unavoidable. Citation keys are those of
[architecture.md, Conventions](architecture.md#conventions). Evidence marks:
**P** observed in a probe recording (the probe and line or count named);
**D** documented but not observed in the probes; **U** undocumented, labelled
so wherever it is used.

## Pipeline

```text
CLI stdout line ──> frame (JSONL) ──> decode (typed, unknown fields kept)
   ──> map (adapter-specific) ──> Event{...} ──> queue ──> coalesce per frame
   ──> reducer: apply(state, event) ──> state ──> view(state, ui) ──> frame
```

Each normalised event carries one envelope:

| Field | Meaning |
| - | - |
| `Seq` | a counter the dashboard assigns in stream order; the reducer applies events in `Seq` order only |
| `At` | the dashboard's clock when the line was read (observed time, not the CLI's) |
| `Provider`, `Session` | the adapter id; the CLI's session or thread id once known |
| `Kind` | one of the conceptual events below |
| `Origin` | `observed`, `inferred`, `ai`, `derived` or `mock` (see [Origins](#origins)) |
| `Agent` | the agent key: `""` for the orchestrator; Claude Code `agent_id`; Codex `threadId` of a child |
| `Turn`, `Item` | the turn and item ids where the CLI gives them (Codex `turnId`, `itemId`; Claude Code `message.id`, `tool_use_id`) |
| `Payload` | the kind-specific fields |
| `Raw` | the original line, kept for the session log and the "raw event" view |

The mapping is a pure function per adapter (`map(line) -> []Event`), so it is
tested from the recorded probes alone
([reliability.md, Test plan](reliability.md#test-plan)).

## Origins

Every event and every value on screen has one origin, rendered distinctly
([ui-design.md, Origins on screen](ui-design.md#origins-on-screen); AM2 U9;
TEL rules; AGENTS.md, "The conversation is real").

| Origin | Meaning | Example |
| - | - | - |
| `ai` | text the model wrote, as the CLI delivered it | an `assistant` text block; `item/agentMessage/delta` |
| `observed` | a fact the CLI reported in a structured field | `task_started`; `turn/completed status:"failed"`; a quota percentage |
| `derived` | computed by the dashboard from observed values, with the rule named | the active agent count; the Codex session token total |
| `inferred` | a status the dashboard assumes without a direct report | "turn submitted" before the first event; an agent seen only through its events |
| `mock` | produced by the mock provider | every event in mock mode |

Nothing is ever shown as `ai` that the CLI did not deliver as model text, and
no `inferred` status is ever displayed as `observed`.

## Conceptual events

The eleven events of AM2 E3, then the ones the state also needs. For each
CLI: the source, its evidence, or "unavailable" with the fallback. Unknown
types are mapped to `unrecognised` and never dropped (INT shape).

| Event | Claude Code | Codex |
| - | - | - |
| `conversation.message.delta` | `stream_event` with `content_block_delta` (`text_delta`, `thinking_delta`) and `parent_tool_use_id: null`. **P**: P1; P2 (25 `content_block_delta` lines: 3 `text_delta`, 1 `thinking_delta`, 1 `signature_delta`, 20 `input_json_delta` for tool input). Subagent text: **unavailable as deltas** (0 of 230 `stream_event` lines had a parent id, CC-10 q15); fallback: the whole forwarded message as one `completed` event | `item/agentMessage/delta` (`itemId`, `delta`), `item/reasoning/*Delta`, per `threadId`. **P**: P6b lines 22-33; P7 (51 deltas, parent and children) |
| `conversation.message.completed` | `assistant` message (`message.id`, `content`, `model`, `usage`), main thread when `parent_tool_use_id` is null, else attributed to the subagent's `agent_id` (with `--forward-subagent-text`). One line **per content block**, all with the message's id, while the message goes on streaming: P2's last turn sends its thinking block's line before its text deltas. The entry keeps streaming until `message_stop`, and no block's line takes away text another block streamed (the second L2 run, 2026-10-16). **P**: P1, P2, P3b | `item/completed` with `type:"agentMessage"` (`text`). **P**: P6b, P7 |
| `tool.started` | an `assistant` content block `tool_use` (`id`, `name`, `input`), carrying `agent_id` for subagents. **P**: P2 (Write), P3b (subagent Bash) | `item/started` of type `commandExecution`, `fileChange`, `mcpToolCall`, `webSearch`, `dynamicToolCall`. **P**: P6b, P7 (21 `item/started`) |
| `tool.completed` | a `user` message `tool_result` block (`tool_use_id`, `is_error`, content). **P**: P2, P3b. A tool still open when the turn's `result` arrives ends `interrupted` (inferred) | `item/completed` of the same types (`status`, `exitCode`, `aggregatedOutput`, `durationMs`). **P**: P6b line 56 (`durationMs`), P7 (23 `item/completed`) |
| `workflow.stage.changed` | **unavailable**: the `Workflow` tool's phases have no documented stream form and were not probed under `-p` (CC-5; CC-10 q7, closed in plan 14: the dashboard starts no workflow, so it does not depend on one). Fallback: the stage tree is **derived** from the turn and its task events ([Agent tree](#agent-tree)), labelled inferred; a `local_workflow` task appears as one node with its status (**D**: `task_started.task_type`, CC-5) | `turn/plan/updated` (`plan[{step, status}]`) and `item/started` of type `plan`. **D** only: no plan update occurred in P6b or P7 (codex-cli.md, section 6). Fallback: the same derived tree |
| `agent.started` | `system/task_started` with `task_type` `local_agent` (or `local_workflow`) and `ambient` false: `task_id`, `description`, `is_backgrounded`, `spawn_depth`, `parent_task_id`. **P**: P3, P3b | the parent's `item/completed` of type `subAgentActivity`, `kind:"started"`, `agentThreadId`, `agentPath`. **P**: P7 (two children) |
| `agent.progress` | `system/task_progress` (`usage.total_tokens`, `tool_uses`, `last_tool_name`); the subagent's `tool_use` and forwarded messages. **P**: P3b (2 `task_progress`) | any event tagged with the child's `threadId`: `thread/status/changed`, items, deltas, `thread/tokenUsage/updated`. **P**: P7 |
| `agent.completed` | `system/task_notification` `status:"completed"` or `task_updated.patch.status:"completed"`. **P**: P3b | the child's `turn/completed` `status:"completed"` then `thread/status/changed` `idle`. **P**: P7 |
| `agent.failed` | `task_notification` `status` `failed` or `stopped`; `task_updated.patch.status` `failed` or `killed`. **D**: not observed (P3b's task completed) | the child's `turn/completed` `status` `failed` or `interrupted`. **D** for a child; the failed shape is **P** on the parent (P6 lines 23-24) |
| `approval.requested` | `control_request` `can_use_tool` (`tool_name`, `display_name`, `input`, `description`, `tool_use_id`, `agent_id`). **P**: P2, P3b (two, one from the subagent). `--permission-prompt-tool stdio` is **U** (INT 4) | `item/commandExecution/requestApproval`, `item/fileChange/requestApproval` (`threadId`, `itemId`, the item's `changes`), plus `thread/status/changed` `waitingOnApproval`. **P**: fileChange in P6b lines 35-47 and P7; command approval P8, P11n, plan 4 L3 (CX-10 q1) |
| `telemetry.updated` | `rate_limit_event` (quota, **P** P1, P2, P3b; `unifiedWindows` **U**); `assistant.message.usage`; `result` (`usage`, `modelUsage`, `total_cost_usd`, `duration_ms`, **P** P1 line 12); the `get_context_usage` response (**P** P2 lines 4-6); `system/thinking_tokens` (**P** P3b) | `thread/tokenUsage/updated` (**P** P6b line 41; P7, 11); `account/rateLimits/updated` (**P** P7, 11) and the `account/rateLimits/read` response (**P** P6 line 7); the `thread/start` response model (**P** P6b line 9); `model/rerouted` (**D**) |
| `session.started` | `system/init` (`session_id`, `model`, `cwd`, `permissionMode`, `claude_code_version`) after the `initialize` response. **P**: P1 to P4 | the `thread/start` response (`thread.id`, `model`, `sandbox`, `approvalPolicy`) and `thread/started`. **P**: P6b lines 8-11 |
| `session.ended` | the process exits (code, stderr tail); observed by the reader. **P**: P2 (exit code 1 after stdin closed); the dashboard's close exits 0 (plan 4, L1 (e) and L2d; CC-10 q13) | same; the probe drivers recorded each exit (P6b, P7) |
| `turn.started` | no explicit event: `inferred` when the dashboard writes the user message, `observed` at the first `stream_event` or `system/session_state_changed` `running` (needs `CLAUDE_CODE_EMIT_SESSION_STATE_EVENTS=1`, which the adapter sets; **P**: P13c, `running` then `idle`) | `turn/started`. **P**: P6b, P7 (3) |
| `turn.completed` | `result` (`subtype`, `is_error`, `terminal_reason`); `success`, or `error_during_execution` after an interrupt. **P**: P1, P2 | `turn/completed` (`status` `completed`, `failed`, `interrupted`, `durationMs`). **P**: P6b lines 68 and 80-91, P7 |
| `approval.resolved` | the dashboard's own `control_response` (observed as sent); no confirmation from the CLI | `serverRequest/resolved` (`threadId`, `requestId`). **P**: P6b line 39, P7 line 91 |
| `question.requested` | `AskUserQuestion` arrives through `can_use_tool` with `tool_name:"AskUserQuestion"` (INT 4). **P**: P14a (CC-10 q3). Shown as an approval card with its questions and options; the owner picks an option of each and the answer is an allow whose `updatedInput` is the request's input plus `answers` (`{"<question>": "<label>"}`, a multiple choice's labels joined by ", "), which Claude Code returns to the model as the tool result (plan 14, item 1). A deny instead says the user will answer in their next message (`QuestionDenyMessage`, plan 5, item 10; a cancel keeps the usual message); an allow without answers is never sent | `item/tool/requestUserInput` exists only behind `experimentalApi` and is disabled in Default mode (CX-10 q6): **unavailable** in the MVP. Fallback: the model asks in plain agent text, answered with the next message |
| `diff.updated` | **derived** from `tool_use.input` of `Edit` (`old_string`, `new_string`) and `Write` (`content`), labelled derived. **P**: P2 (Write input) | `fileChange.changes[{path, kind, diff}]` and `turn/diff/updated` (the turn's unified diff). **P**: P6b, P7 (5 `turn/diff/updated`) |
| `retry` | `system/api_retry` (main loop), `tool_progress.subagent_retry` (subagents). **D**, not observed | `error` with `willRetry:true`. **D**, not observed (CX-10 q9, closed in plan 14: read when sent, not depended on) |
| `error` | `result.is_error`, `assistant.error`, `system/permission_denied`, a failed `tool_result`. **P**: P2 (`error_during_execution` after interrupt, not counted as an error) | `error` with `willRetry:false`, `turn/completed failed` with `codexErrorInfo`. **P**: P5, P6 lines 23-24 |
| `status.changed` | `system/status` (`compacting`, `permissionMode`) **P** P1; `system/compact_boundary` **D**; `system/commands_changed` **P** P3b, logged with no status (no MVP zone lists commands) | `thread/status/changed` **P** P7 (10); `mcpServer/startupStatus/updated` **P** P7 (12) |
| `unrecognised` | any other `type`/`subtype`, kept with its raw line | any other method or item type |

Every row's fallback, and what the screen shows while the fallback applies,
is in [reliability.md, Fallbacks for every source](reliability.md#fallbacks-for-every-source).

## State model

One immutable value, replaced by each reducer step. Field names are
indicative; plan 4 fixes the code.

```text
State
  Screen           startup | dashboard
  Startup          StartupState        (startup machine below)
  Context          SessionContext      (architecture.md, Session context)
  Session          SessionState        (session machine)
  Conversation     []Entry             the record of zone B
  Turn             TurnState           (turn machine)
  Agents           map[AgentKey]Agent  + Order []AgentKey
  Stages           []Stage             the derived or reported stage tree
  Approvals        map[ApprovalID]Approval, Queue []ApprovalID
  Tools            map[ToolID]ToolCall
  Telemetry        Telemetry
  Errors           []ErrorEntry        bounded list shown in zone D
  Log              ring of Event       raw events for the inspection view
  Unrecognised     count, last N raw lines

Entry        {Seq, Role: user|assistant|tool|notice|approval, Agent, Text,
              Origin, Streaming bool, At}
Agent        {Key, Parent, Label, Kind (subagent|workflow|bash|thread),
              Model (value or N/A), Status, StatusOrigin, Task, Files,
              Tools []ToolID, Messages []Entry, Tokens, StartedAt, EndedAt,
              Retries, Error}
Approval     {ID, Agent, Tool, Summary, Input, Diff, Options, Status,
              RequestedAt, AnsweredAt, Answer}
Telemetry    {Quota5h, Quota7d, Context, Tokens, Cost, Model, Agents,
              Concurrency, Duration, Errors, Retries} each a Value
Value        {Present bool, V, Unit, Origin, Source, ReceivedAt, ResetsAt,
              Reason (why N/A)}
```

`Value.Present == false` renders "N/A" with its `Reason`; a `Value` whose
`ResetsAt` has passed is cleared to N/A by the clock tick (TEL rules). Ages
are computed at render time from `ReceivedAt`, so no event is needed to
age a value.

### Conversation record

- The record is built only from CLI events: the user's own messages as sent
  (byte for byte, `Origin: observed` once written to the CLI), `ai` text,
  tool calls and results, approvals and their answers, and notices
  (interrupted, error, reconnected) marked as dashboard notices.
- Deltas append to the open `Streaming` entry of the same message id; the
  `completed` event replaces its text with the complete content (the
  complete message wins if they differ, and the difference is logged).
- Each entry records the turn it belongs to, `Entry.Stage` (the stage of
  zone A's tree, 1-based): the one open when it came, for the owner's
  prompt the one it opens, after a turn ended that turn, 0 before any
  (plan 19). Zone B reads it for RESULT: an exchange's last message is a
  result only once the turn that ended the exchange completed and every
  agent whose `Stage` falls in the exchange's stages is `Completed` (plan
  20, the owner's section 6.1: one running, failed or stopped leaves the
  message neutral, so "I've launched the subagent…" never reads as the
  result), and only after other activity; never from the text
  ([ui-design.md, Zone B](ui-design.md#zone-b)). The labels, the cards,
  RESULT and the folds are drawn from the record, the turn machine and the
  agent machines each frame; nothing of them is stored, except the runs of
  calls the owner expanded, a choice of the interface.
- A thinking block is kept as its entry (`Entry.Thinking`, no text) and
  drawn only while it streams with no text yet; once it ends, zone B does
  not draw it (plan 20, the owner's section 11), so a finished one never
  looks active. The record keeps it for the thinking display the owner
  plans separately.
- The record lives in memory for the session (ASM-11). Claude Code's
  `--resume` does not replay history (P4), so the dashboard's record is the
  only source of earlier turns in one dashboard session
  ([reliability.md, Resumption](reliability.md#resumption)).

### Agent tree

- The root is the orchestrator (the main session or thread). Each agent
  node hangs under its parent: Claude Code `parent_task_id`, else the
  orchestrator (P3b: `spawn_depth: 1`); Codex the thread that emitted the
  `subAgentActivity` (P7), with `agentPath` as its label. An activity toward
  the orchestrator's own thread (`/root`, P7b) is not an agent.
- **Stages** (zone A's workflow tree): with a reported plan (Codex
  `turn/plan/updated`), the stages are its steps with their status
  (`observed`). Otherwise each user turn is a stage, and its agents and
  tool groups are its children, with statuses `derived` from them; the
  stage tree is labelled "inferred from activity" (AM2 A2, A3; CC-10 q7).
- Background task sets: Claude Code's `background_tasks_changed` replaces
  the set of live background tasks (replace, not merge, CC-5); a task in the
  set with no node is created `inferred`; a node missing from the set and
  without a terminal event is marked "ended, status not reported".
- `ambient: true` tasks are ignored in counts and the tree (CC-5).
- An event carrying an unknown agent key creates a node with status
  `inferred` and label "unannounced agent"; an event lacking any key is
  attached to the orchestrator and marked "unattributed" (INT 7). A Codex
  `spawnAgent` call with a `prompt` labels its child "task: " and the
  prompt's first line, cut at 24 characters, and gives it the whole prompt
  as its description, with the spawn request as origin, as the requested
  model: the label fills only an agent with none, "unannounced agent" or a
  previous spawn label, and the agent's own label replaces it, in either
  order (plan 10, item 1).

### Zone C's height

Zone C lists the agents in play and is sized by them (`State.InPlay`, their
keys, and `State.AgentsInPlay`, their count; plan 18, brief amendment 14),
an empty strip with none ([ui-design.md](ui-design.md#zone-c)); so each
prompt the owner submits clears it of the earlier agents, which zone A's
tree keeps. Each agent keeps the stage it
appeared in and, once it ends, the stage it ended in (`Stage`, `EndStage`;
0 while it runs). An agent is in play when it runs, or appeared or ended in
the stage of the owner's last prompt or a later one. A stage the CLI opens
by itself, such as Claude Code's for a background agent's notification
(`ByCLI`), is not a prompt, so it changes nothing either way; before any
prompt, as a replay can start, every agent counts. An agent still running
when the owner sends a prompt stays in play through that turn, also once it
ends; the next prompt resets the count.

## State machines

For each machine: its states, every transition with its trigger and which
CLI supports it, the invalid transitions and what they do. **P**, **D** and
**U** as above; "dash" is a transition the dashboard drives itself; "n/a"
means the CLI has no such transition.

### Startup machine

States: `Detecting`, `Asking`, `Validating`, `Starting`, `Ready`,
`StartFailed`.

| From | Trigger | To | Claude Code | Codex |
| - | - | - | - | - |
| (launch) | application start | `Detecting` | dash | dash |
| (launch) | application start with `--mock <file>` | `Starting` with the mock provider: detection and both questions skipped; the session context names provider "Mock data" and the file as project ([reliability.md, Mock mode](reliability.md#mock-mode)) | n/a | n/a |
| `Detecting` | both detections returned or timed out | `Asking` | dash (CC-2) | dash (CX-2) |
| `Asking` | Confirm pressed | `Validating` | dash | dash |
| `Validating` | path or provider invalid | `Asking`, with the message on the field | dash (INT paths) | dash |
| `Validating` | both valid | `Starting` (dashboard shown, session `Spawning`) | dash | dash |
| `Starting` | handshake done (`session.ready`) | `Ready` | **P** P2 | **P** P6b |
| `Starting` | quit | session `Closing`, then (exit) when the process has exited | dash | dash |
| `Starting` | spawn error, handshake timeout, process exit | `StartFailed` | dash | dash |
| `StartFailed` | owner chooses "back" | `Asking`, answers kept | dash | dash |
| `Ready` | quit | (exit) | dash | dash |
| `Detecting`, `Asking`, `Validating`, `StartFailed` | quit (Ctrl+Q; Esc on `Asking`) | (exit), nothing started | dash | dash |

Invalid: Confirm while `Detecting` is ignored (the button is disabled until
detection returns); a second Confirm while `Validating` or `Starting` is
ignored; choosing a provider shown "not installed" is impossible (not
selectable). No transition leads from `Ready` back to `Asking`: one provider
and one project per session (AGENTS.md; AM4 X9).

### Session machine (the CLI process)

States: `Spawning`, `Handshaking`, `Ready`, `Closing`, `Closed`, `Crashed`.

| From | Trigger | To | Claude Code | Codex |
| - | - | - | - | - |
| `Spawning` | process started | `Handshaking` | dash | dash |
| `Spawning` | spawn error (not found, access denied) | `Crashed` | dash | dash |
| `Handshaking` | `initialize` response (Claude Code: `system/init` comes only after the first user message, P2 lines 5 and 7); `initialize`, `thread/start` responses (Codex) | `Ready` | **P** P2 | **P** P6, P6b |
| `Handshaking` | 60 s timeout, error response, `cwd` mismatch | `Crashed` | dash (CC-4 timeout) | dash (ASM-21) |
| `Ready` | `session.started` with a `cwd` other than the project (Claude Code reports it with the first turn) | `Crashed`, the process closed | dash | dash |
| `Ready` | owner quits | `Closing`: pending approvals answered `cancel`, the turn interrupted, then stdin closed, in that order | dash | dash |
| `Ready` | owner quits while a turn or an agent runs | stays `Ready`, asks to confirm (ui-design.md, Keys): a second Ctrl+Q quits, Esc stays | dash | dash |
| `Closing` | process exited | `Closed` | **P** P2 (exit after stdin closed) | **D** |
| `Closing` | process exited with a message still `Submitted` | `Closed`; the message goes back to the editor and the stored draft (ASM-23) | dash | dash |
| `Closing` | 5 s, then 5 s more, elapsed | forced kill, then `Closed` | **P** job kill mid-turn, 41 ms, resumable (plan 4, L1 (g); CC-10 q13) | **P** job kill mid-turn, 24 ms, no process left (plan 4, L3; CX-10 q8) |
| `Ready`, `Handshaking` | process exited unexpectedly or stdout EOF | `Crashed` | dash | dash |

Invalid: a user message, approval answer or interrupt while not `Ready` is
refused with a visible notice, and the draft is kept; events read after
`Closed` are logged, not applied. `Crashed` is final for the process. A crash
before `Ready` is the startup's `StartFailed` (no session id exists yet).
After `Ready`, the dashboard shows the exit code and stderr tail and offers
two keys, quit or resume the same CLI session (its id is known) in a new
process, which starts a new session machine at `Spawning` (no silent restart; AGENTS.md "Keep failures visible";
[reliability.md, Resumption](reliability.md#resumption)).

### Turn machine

States: `Idle`, `Submitted`, `Running`, `WaitingApproval`, `Interrupting`,
then `Idle` with the last outcome `completed`, `failed` or `interrupted`.

| From | Trigger | To | Claude Code | Codex |
| - | - | - | - | - |
| `Idle` | owner sends a message | `Submitted` (inferred) | dash | dash |
| `Submitted` | first event of the turn | `Running` | **P** first `stream_event` (P1) | **P** `turn/started` (P6b) |
| `Submitted` | write to stdin fails | `Idle`, outcome `failed`, draft restored | dash | dash |
| `Running` | `approval.requested` for the orchestrator or any agent | `WaitingApproval` | **P** P2, P3b | **P** P6b, P7 |
| `WaitingApproval` | every pending approval answered | `Running` | **P** P2, P3b | **P** P6b |
| `Running`, `WaitingApproval` | owner interrupts | `Interrupting` | **P** `interrupt` during text (P2); during a tool or a pending approval **P** (plan 4, L1 (c) and (d); CC-10 q4) | **P** `turn/interrupt` during text (P6b); with children **D**, not depended on: each child's own events update it (CX-10 q3) |
| `Interrupting` | turn end reported | `Idle`, outcome `interrupted` | **P** `result error_during_execution` (P2) | **P** `turn/completed interrupted` (P6b) |
| `Interrupting` | turn end reported as completed | `Idle`, outcome `completed`: the turn ended before the interrupt reached it, and is never shown as interrupted (owner, 2026-10-16) | **D** `result success` | **D** `turn/completed completed` |
| `Interrupting` | no turn end within 10 s | stays, offers "force stop" (closes the session) | dash | dash |
| `Interrupting` | owner forces stop | `Idle`, outcome `interrupted`; session `Closing` | dash | dash |
| `Submitted`, `Running`, `WaitingApproval` | owner quits, confirmed | `Running` and `WaitingApproval` go `Interrupting`; `Submitted` stays until its first event; the session `Closing` | dash | dash |
| `WaitingApproval` | turn end reported | `Idle` with that outcome; its pending approvals `Expired` | **D** `result` | **D** `turn/completed`, then `serverRequest/resolved` |
| `Idle` | approval from a background agent | stays `Idle`; the agent goes `WaitingApproval` and the approval is queued | **D** (background agents outlive the turn, P3) | **D** (a child's own turn) |
| `Idle` | the CLI starts a turn itself: `message_start` or `turn/started` with no message sent, session `Ready` | `Running`, marked "started by the CLI"; no draft is accepted | **P** after a background agent's `task_notification` (P3 lines 80-98, P3b; found in plan 4) | **D** not observed |
| `Idle` | a prompt the dashboard did not send: a mock file's `conversation.user` | `Submitted`, the owner's turn, as sending opens it: the last turn's notice ("started by the CLI") cleared (plan 20) | dash (mock mode) | dash (mock mode) |
| `Running` | turn end, success | `Idle`, outcome `completed` | **P** `result success` | **P** `turn/completed completed` |
| `Running` | turn end, error | `Idle`, outcome `failed` | **D** `result is_error` | **P** `turn/completed failed` (P6) |
| any | session `Crashed` | `Idle`, outcome `failed` | dash | dash |

Invalid: sending while a turn is not `Idle` is refused in the MVP, with the
draft kept and the hint "turn running: wait, or interrupt" (ASM-15; Codex
`turn/steer` and Claude Code's message queue are never used, CX-10 q5);
an interrupt while `Idle` or `Submitted`-without-events is a no-op with a
notice; a turn-end event while `Idle` is applied to telemetry and logged as
"unexpected turn end". Background agents may keep running after the turn is
`Idle` (P3, P3b): the agent machine, not the turn machine, tracks them.

### Agent machine

States: `Announced`, `Running`, `WaitingApproval`, `Completed`, `Failed`,
`Stopped`, `Ended` (status not reported). Each state carries its origin.

| From | Trigger | To | Claude Code | Codex |
| - | - | - | - | - |
| (none) | `agent.started` | `Running` (Claude Code) / `Announced` (Codex) | **P** `task_started` (P3b) | **P** `subAgentActivity started` (P7) |
| (none) | event with an unknown key | `Running`, origin inferred | dash (INT 7) | dash |
| `Announced` | child `thread/status/changed active` or `turn/started` | `Running` | n/a | **P** P7 |
| `Running` | `agent.progress` | `Running` (task, tokens, last tool updated) | **P** `task_progress` (P3b) | **P** P7 |
| `Running` | approval request carrying this key | `WaitingApproval` | **P** P3b (`agent_id`) | **P** P7 (child `threadId`) |
| `WaitingApproval` | its approvals answered | `Running` | **P** P3b | **P** P7 |
| `Running`, `WaitingApproval` | completed | `Completed` | **P** `task_notification completed` (P3b) | **P** child `turn/completed completed` (P7); `subAgentActivity completed` on the parent, then the child's `turn/completed` (P7b) |
| `Running`, `WaitingApproval` | failed | `Failed` | **D** `task_notification failed` | **D** child `turn/completed failed` |
| `Running`, `WaitingApproval` | stopped or killed | `Stopped` | **D** `task_notification stopped`, `task_updated killed` | **D** child `turn/completed interrupted` |
| `Running` | absent from `background_tasks_changed` with no terminal event | `Ended`, inferred | **D** (replace semantics, CC-5) | n/a |
| `Completed` | the same key starts again | `Running`, a new run appended | **D** (`agent_id` stable across resumes, CC-5) | **D** (a new `turn/started` on the child) |
| any non-final | session `Crashed` or `Closed` | `Ended`, inferred | dash | dash |

Invalid: a progress or tool event for an agent in a final state is logged in
the agent's history, flagged "after end", and does not change the status; a
terminal event for an unknown key creates the node directly in that state,
inferred; a completed turn end for an agent already `Completed` only confirms
it, nothing logged; an event carrying the session's own key is the
orchestrator's, never a node (L5: Codex 0.163.0's activity toward `/root`).

### Approval machine

States: `Pending`, `Answering`, `Allowed`, `Denied`, `Expired`.

| From | Trigger | To | Claude Code | Codex |
| - | - | - | - | - |
| (none) | `approval.requested` | `Pending`, queued | **P** P2, P3b | **P** P6b, P7 |
| `Pending` | owner allows (once) | `Answering` → `Allowed` | **P** allow with `updatedInput` (P2, P3b) | **P** `accept` (P6b) |
| `Pending` | owner allows for the session | `Answering` → `Allowed` | n/a: Claude Code's session rule goes through `updatedPermissions`, not used (CC-4) | **D** `acceptForSession`, unless the request's `availableDecisions` omits it (Codex 0.163.0: a file change has no list, offered; a command lists accept, the amendment and cancel, not offered) or the request could not be read; otherwise refused with a note (plan 5, item 8; plan 8) |
| `Pending` | owner denies | `Answering` → `Denied` | **P** deny with a message (plan 5, L9; P13c) | **P** `decline` (plan 4, L3; CX-10 q1) |
| `Pending` | owner cancels the turn | `Answering` → `Denied`, then the turn interrupts | dash: deny, then `interrupt` | **P** `cancel`, the turn interrupted 8 ms later (plan 4, L3; CX-10 q1) |
| `Pending` | the request cannot be parsed | `Denied` automatically, shown with its raw line | dash (INT 4 fallback) | dash: `decline`, or a JSON-RPC error for an unknown method (P6b driver) |
| `Answering` | writing the answer fails (pipe closed) | `Expired`; session `Crashed` | dash | dash |
| `Pending` | turn ends, CLI resolves it, or session ends | `Expired` | dash | dash: `serverRequest/resolved` for a request not answered (**P** for answered ones, P6b line 39) |

Invalid: a second answer to the same id is ignored and logged; an answer
for an unknown id shows an error; nothing is ever answered without the
owner's action except unparseable requests, which are denied (never
allowed). "Allow and remember" options that persist a rule
(`permission_suggestions`, execpolicy or network amendments) are not
offered (INT 4; [reliability.md, Configuration untouched](reliability.md#configuration-untouched)).

### Tool call machine

States: `Started`, `Running`, `Completed`, `Failed`, `Denied`,
`Unfinished`.

| From | Trigger | To | Claude Code | Codex |
| - | - | - | - | - |
| (none) | `tool.started` | `Started` | **P** `tool_use` (P2) | **P** `item/started` (P6b) |
| `Started` | output or progress | `Running` | **D** `tool_progress` heartbeats | **P** `item/commandExecution/outputDelta` (P7b; none in P6b or P7) |
| `Started`, `Running` | `tool.completed` without error | `Completed` | **P** `tool_result` (P2) | **P** `item/completed` (P6b) |
| `Started`, `Running` | `tool.completed` with an error | `Failed` | **D** `is_error` | **D** `status:"failed"` |
| `Started` | its approval denied | `Denied` | **D** | **D** |
| `Started`, `Running` | turn or agent ends first | `Unfinished`, inferred | dash | dash |

Invalid: a completion for an unknown tool id creates the call in its final
state, inferred; a second completion is logged and ignored.

### Stage machine

States: `Open`, `Closed` (with the turn's outcome). Zone A's stages are
inferred from the turn machine, one per turn, a turn the CLI started
included; a plan, when reported, is observed (plan 4, step 6).

| From | Trigger | To | Claude Code | Codex |
| - | - | - | - | - |
| (none), `Closed` | the turn leaves `Idle` | a new stage, `Open` | **P** P2, P3b | **P** P6b, P7 |
| `Open` | the turn returns to `Idle` | `Closed`, with the turn's outcome | **P** P2 | **P** P6b |
| `Open` | `turn/plan/updated` | `Open`, its plan replaced | n/a | **D** v2 schema `TurnPlanUpdatedNotification`, not recorded |
| `Open`, `Closed` | a tool or agent appears | it joins the last stage | **P** P3b | **P** P7 |

Invalid: a plan outside any turn is logged and ignored; a child thread's
plan is noted in its agent's history, not drawn.

## Concurrent output

One stdout stream per CLI carries every agent; the reducer routes by agent
key: Claude Code `agent_id` first (parallel subagents share a
`parent_tool_use_id`, INT 7), Codex `threadId`, then `turnId` and `itemId`
(P7: two children and the parent interleaved without ambiguity). Each agent
has its own message and tool buffers, so zone C and the inspection view
update independently of the conversation (AM2 C10, C11). The reducer never
blocks on rendering; high-rate events are coalesced to one state update per
frame (architecture.md, Threads of execution).

## Polling

The CLIs push events; the dashboard does not poll them on a timer (AM2 E6).
The few reads it makes, each once and on a trigger:

| Read | When | Why it cannot be an event | Source |
| - | - | - | - |
| Codex `account/rateLimits/read` | after the handshake, and after each `turn/completed` | quota before the first model call has no push; pushes follow model calls only | TEL CX; P6 line 7 |
| Claude Code `get_context_usage` | after each `result` | context is not pushed; the derived value from `usage` lacks the auto-compact threshold | TEL CC; P2 lines 4-6; no model call: answered in 914 ms with no `assistant` message, `result` or `rate_limit_event` after it (P14a, CC-10 q10) |
| Claude Code `/usage` | **not used**: never sent (CC-10 q9, closed in plan 14) | it is a prompt in the conversation, may cost, and the endpoint can rate-limit | TEL CC |
| Git branch | once at startup, from `.git/HEAD` | display only; no git events exist | architecture.md, Session context |
| Detection (`-v`, `--version`, auth status) | once at startup | no other source | CC-2; CX-2 |

The only timer is the local clock tick, which advances animations and value
ages and expires values past their reset time; it reads no external source.
