# Architecture

Plan 2, 2026-10-16. How Agent Command Center is built: the stack, the five
separated parts, the provider interface with one adapter per agent CLI, the
session context, the integration strategy, where it runs and how it starts.
The events and the state are in [events-and-state.md](events-and-state.md),
the screens in [ui-design.md](ui-design.md), and failures, fallbacks and tests
in [reliability.md](reliability.md). The owner approved it with the proposal of
plan 3 on 2026-10-16 (commit d5b910e); later plans keep it current.

## Conventions

Every choice names where it comes from. The keys used in all four plan 2
documents:

| Key | Meaning |
| - | - |
| `INT n` | [research/integration.md](research/integration.md), section n (1 to 9); `INT shape`, its "Common provider shape"; `INT paths`, its "Path forms for the startup screen" |
| `TEL CC`, `TEL CX`, `TEL rules` | [research/telemetry-sources.md](research/telemetry-sources.md), its "Claude Code", "Codex CLI" and "Rules for every value" sections |
| `TI n` | [research/terminal-input.md](research/terminal-input.md), section n |
| `CC-n`, `CX-n` | [research/claude-code.md](research/claude-code.md) and [research/codex-cli.md](research/codex-cli.md), section n (`CC-10 q3` is open question 3 of section 10) |
| `SUM` | [research/summary.md](research/summary.md) |
| `RD` | [reference-design.md](reference-design.md) |
| `P0` to `P7` | the probe recordings, [research/probes/README.md](research/probes/README.md) |
| `B2`, `B3`, `AM2` to `AM5` | [brief.md](brief.md): phases 2 and 3, amendments 2 to 5; item ids are those of [Coverage of the brief](#coverage-of-the-brief) |
| `ASM-n` | an assumption: a choice with no plan 1 finding or brief item behind it, listed under [Assumptions](#assumptions) |

## Stack

| Part | Choice | Version | Source |
| - | - | - | - |
| Language | Go | the current stable release when plan 4 starts (the reference projects use 1.24 to 1.26) | brief B2.5; SUM "Decision"; ASM-1 for the version |
| TUI runtime | Bubble Tea v2, `charm.land/bubbletea/v2` (MIT) | the v2 line studied at commit `f9df43c`; pinned in plan 4 | TI 2.2, TI 11 item 1: the only Go option with one-message bracketed paste on Windows, Kitty and modifyOtherKeys requests, OSC 52, SGR mouse; v1 fails on Windows paste (TI 2.3) |
| Layout and style | Lip Gloss v2 (MIT), with its compositor for hit testing | the v2 line studied at commit `19f011f` | TI 2.5 (`Compositor.Hit`) |
| Input editor | our own component, inside the application | n/a | TI 11 item 2: the Bubbles textarea caps at 10,000 lines, renders in O(n) and has no undo (TI 2.4) |
| Native clipboard | `golang.design/x/clipboard` (MIT, no cgo on Windows) on Windows and macOS; on Linux and WSL, the external tools of TI 5.3 (`wl-copy`, `xclip`, `clip.exe`) | v0.11.0 (commit `d95c794`) | TI 5.2, TI 11 item 4; choosing the MIT library avoids a second notice (ASM-2) |
| OSC 52 | `x/ansi` (already a Bubble Tea dependency), with its tmux passthrough | as pinned by Bubble Tea | TI 5.3 "Tools for wrapping OSC 52" |
| File viewer highlighting | Chroma, `github.com/alecthomas/chroma/v2` (MIT, pure Go), with `github.com/dlclark/regexp2/v2` (MIT); its token types coloured from the palette, never a Chroma theme. Zone B's Markdown keeps its own highlighter (ASM-17) | v2.27.0, regexp2 v2.2.1 | plan 17; brief amendment 11, approved by the owner on 2026-10-18 |
| Codex protocol types | written by hand for the messages the MVP uses, checked against Codex's committed JSON Schema (Apache-2.0) by fixture tests; changed from "generated" at plan 3 | the schema of Codex CLI 0.145.0 (P0 `p0-codex-app-server-schema-v2.json`) | SUM "What plan 2 inherits"; INT 6 |
| Claude Code protocol | written from the documented messages, with the Python Agent SDK (MIT) as the reference for the control-request wire form | Claude Code 2.1.300 | INT 1; CC-4 way A |
| tmux | **not used and not a dependency** | n/a | SUM: both CLIs run over pipes on native Windows (P1 to P4, P6b, P7); INT 6: a PTY or tmux yields a screen, not events. The application runs inside tmux like any terminal program (clipboard passthrough, TI 7) |

Every dependency above needs the owner's approval in plan 3 (AGENTS.md, Hard
constraints, "Dependencies are fixed"). No dependency is copied as source; any
idea taken from the Bubbles textarea or tview (MIT) is recorded with its
source, commit and license where it is used (AGENTS.md; TI 11 item 2).

## Where it runs

- **The MVP runs as a native Windows program** on the owner's Windows 11
  machine, in Windows Terminal, with both agent CLIs installed natively;
  the VS Code terminal and conhost are out of scope (brief, amendments 6
  and 7). Both CLIs were driven this way (INT 9; P1 to P7). This
  settles plan 1's open point "where the dashboard runs" (INT "Open points
  for plan 2") and selects the Windows column of the path table (INT paths).
- The code stays portable (Go, no Windows-only API outside the clipboard and
  process helpers), so a Linux or macOS build is possible later; it is **not
  verified** in the MVP (INT 9: only Windows native was probed; see
  [reliability.md, Not verified](reliability.md#not-verified)).
- **Running inside WSL** (a Linux build, with the CLIs installed inside the
  distribution) follows the Linux column of INT paths; **driving a CLI inside
  WSL from the Windows program** through `wsl.exe` is outside the MVP and
  unverified (INT "WSL notes").
- **tmux** needs WSL on this machine (AGENTS.md, Environment). The application
  does not start, require or script tmux; it works inside a tmux pane in WSL,
  where copy uses OSC 52 with passthrough and `tmux load-buffer -w` (TI 7,
  TI 11 item 4).

## The five parts

Amendment 2 asks for five separated parts (AM2 E5). Each is one Go package
group; arrows are the only paths data takes.

```text
 keyboard, mouse, paste                     agent CLI child process
        |                                    (stdin/stdout pipes)
        v                                            ^      |
 +--------------+   commands   +----------------+    |      | raw lines
 | 5. Input and |------------->| 1. Execution   |----+      v
 |    commands  |              |  and session   |   +---------------+
 +--------------+              |  (provider     |-->| 2. Collection |
        ^                      |   adapters)    |   | and normalis- |
        |                      +----------------+   | ation         |
        | key and mouse msgs           ^            +---------------+
        |                              | approvals,         | normalised
 +--------------+   reads only  +-------------+  answers    | events
 | 4. Terminal  |<--------------| 3. Applica- |<------------+
 |  rendering   |   (snapshot)  |  tion state |
 +--------------+               +-------------+
```

| Part | Owns | Never does | Source |
| - | - | - | - |
| 1. **Execution and session management** | detection, spawning the CLI in the project directory, the handshake, writing user messages and control answers, interrupt, shutdown; one adapter per CLI | parse for display, touch the screen | AM2 E5; INT 1; INT shape |
| 2. **Event collection and normalisation** | reading each stdout line, framing, decoding, mapping to the normalised events of [events-and-state.md](events-and-state.md), keeping unknown events as "unrecognised", writing the session log | decide anything about the UI | AM2 E5; INT shape ("unknown event types ... kept, logged and shown"); CX-9 |
| 3. **Application state** | one reducer: `apply(state, event) -> state`; the conversation record, the agent tree, approvals, telemetry with ages, errors; the state machines | do I/O, call a model, read the clock except through events | AM2 E1, E2; AGENTS.md "The interface renders state" |
| 4. **Terminal rendering** | `view(state, ui) -> frame` with Lip Gloss; layout per terminal size; animation frames from a local tick | change the state, read the CLI | B3.2; AM2 E2 |
| 5. **Input and commands** | the editor, key bindings, focus, selection and clipboard, the draft window; turning keys into commands (send, approve, interrupt, select agent) | send to the CLI directly: commands go through part 1 | AM2 E5; AM3; TI 11 |

Rules that keep them apart:

- Only part 1 writes to the CLI; only part 2 reads from it; only part 3 holds
  state; part 4 reads a state snapshot; part 5 emits commands. The
  conversation shown is always the reducer's record of real CLI events
  (AGENTS.md, "The conversation is real").
- The reducer is pure and synchronous, so every state transition is testable
  from recorded events without a terminal or a CLI
  ([reliability.md, Test plan](reliability.md#test-plan)).
- No part calls a model to draw the screen (AM2 E1). The only model calls are
  the ones the owner's prompts cause in the CLI.

### Threads of execution

- **One reader goroutine per CLI process** reads stdout line by line into an
  unbounded in-memory queue (part 2 decodes there), so the CLI's pipe never
  blocks, however slowly the event loop drains it. A stderr reader keeps
  the last 50 lines for error display (ASM-22). Both adapters' sessions run
  on `pipe.Proc` (`internal/provider/pipe/proc.go`, plan 49, design D8): the
  queue and its lock, the spawn's first event, both readers with their limit
  and their recovery, the wait that reports the exit last, the line write
  with its hold, and the close with its escalation; each adapter keeps its
  protocol (its arguments, its mapper, what a line calls for, its answers).
- **One forwarder** takes normalised events from the queue, in order, and
  hands them to the Bubble Tea program: each message carries the first event
  waiting and whatever else is already waiting, at most 512 (`maxDrain`,
  `internal/app/app.go`), applied as one update, so a burst (INT 7: 11
  rate-limit pushes in one P7 turn) is one screen update. Events are not
  coalesced by kind; every one is applied. Bubble Tea draws at most 60 frames
  per second, its default (ASM-3; [ui-design.md](ui-design.md#rendering)).
- **The Bubble Tea loop** runs the reducer and the view. A command (send,
  approve) is handed to part 1, which writes to stdin from its own goroutine,
  so a slow pipe never freezes input.
- **Timers**: one local tick for animation and value ages; one timeout per
  handshake and per shutdown step (INT 1; CC-4); the home screen's banner,
  a frame every 100 ms while it is drawn and the window has the focus
  ([The home screen](#the-home-screen)). No timer polls a CLI (see
  [events-and-state.md, Polling](events-and-state.md#polling)).
- **The home screen's picture**, when drawn whole, encodes its images on a
  goroutine each, joined before the loop goes on (plan 51).

## Provider interface

One interface; each agent CLI is one adapter behind it; nothing outside an
adapter speaks a CLI's protocol (AM4 S4, S5), and the few places the rest of
the application tells one CLI from the other by its provider id are listed
under `Capabilities` below. It follows the concepts both CLIs share (INT
shape).

```text
Provider
  ID() string                       "claude-code" | "codex" | "mock"
  DisplayName() string              "Claude Code" | "OpenAI Codex" | "Mock data"
  Detect(ctx) Detection             installed?, executable path, version, problems
  Start(ctx, SessionContext) (Session, error)
                                    spawn in SessionContext.Path, handshake, first state

Session
  Events() <-chan Event             normalised events, in stream order
  Send(Message) error               a user turn: the text byte for byte and its images
                                    as image input (plan 31; Claude Code: base64 image
                                    blocks before the text; Codex: `image` inputs with
                                    data URLs, then the text)
  Answer(ApprovalID, Decision) error  the owner's decision, the dashboard's own word
                                    (event.Decision: allow, allow for the session where
                                    offered, deny, cancel), written in the CLI's (plan 49)
  AnswerQuestion(QuestionID, Answers) error   where the CLI supports it
  Interrupt() error                 cancel the running turn
  SetApprovalMode(mode string) error  one of the CLI's modes (plan 12; plan 39): Claude
                                    Code at once (set_permission_mode); Codex at once
                                    between turns (thread/settings/update), at a turn's
                                    end during one, with the next turn where Codex
                                    refuses it; another CLI's mode refused
  SetModelSettings(ModelRequest) error  a model, an effort or Ultracode (plan 34): Claude
                                    Code at once, then read back; Codex at once between
                                    turns (the experimental thread/settings/update,
                                    plan 38), at a turn's end during one, with the next
                                    turn where Codex refuses it; refused where not
                                    supported (Model settings)
  Refresh(TelemetryKind) error      on-demand reads (quota, context); never a timer
  Capabilities() Capabilities       what this CLI and version support (below)
  Close(ctx) error                  graceful, then forced (reliability.md)
```

`Capabilities` is a set of flags each adapter returns, so the interface never
pretends both CLIs are the same (AM4 D6 "Do not force all providers to
expose identical telemetry"): streaming of subagent text, per-subagent model,
quota without a turn, API-equivalent cost, clarification questions, plan
steps, resume with history, allow-for-session, the CLI's own modes (`Modes`,
plan 39),
the limits on a message's images (`Images`: count, bytes an image and a
message, pixels a side, patches; plan 31, doc/research/image-input.md), and
the model selection (`Models`, `ModelsNextTurn`, `Ultracode`; plan 34). They
are fixed per adapter (`internal/provider/claude/session.go`,
`internal/provider/codex/session.go`, `internal/provider/mock/session.go`),
not read from detection or the handshake, but for `ModelsNextTurn`, which a
Codex session reports while its main turn runs and for good once Codex
refused `thread/settings/update` (plan 38), and which a mode asked for
follows too (plan 39), read at each choice; these are read:
`Modes`, which F4 and the mode pill need to open the modes selector
(`internal/app/modes.go`), and the three of the model selection, which open
it, mark a change as waiting for the next message and show its Ultracode
row (`internal/app/models.go`). A value no event gives stays "N/A" with its reason,
whatever the flags. The other differences come from the provider id
(plan 21, item 5.3; rewiring them to the flags would change nothing on
screen):

| What differs | Where |
| - | - |
| Codex's slash commands held back once (`send`) | `internal/state/actions.go` |
| the CLI's own resume command printed at the end (`ResumeCommand`) | `internal/app/run.go` |
| the two supported CLIs, their flags and their rows on the home screen | `internal/app/app.go`, `internal/render/startup.go`, `home.go`, `homegrid.go` |
| the project bar: Codex's model, sandbox and policy; "(next turn)" for Codex's mode change | `internal/render/dashboard.go` |
| zone B's speaker label, CLAUDE or CODEX (`providerLabel`) | `internal/render/dashboard.go` |
| the Write and Edit sentence on a pending Claude Code request (`effect`) | `internal/render/approval.go` |
| zone A's telemetry (zone D until plan 23): Codex's credits or Claude Code's overage, the concurrency limit's reason, the N/A reasons each CLI never reports | `internal/render/telemetry.go` |
| the model selector's heading, CLAUDE MODELS or CODEX MODELS (`SelectorHeading`), and the CLI its notices name (`cliName`) | `internal/render/models.go`, `internal/state/models.go` |

The owner's answers are no longer among them (plan 49, design D10): the
keys, the reducer and render know only `event.Decision` (allow, allow for
the session, deny, cancel, and withdrawn for a request the CLI ended
itself). Each adapter writes a decision in its CLI's words, byte for byte as
before, and reads its CLI's words back: Claude Code's allow and deny (a
cancel is a deny), Codex's accept, acceptForSession, decline and cancel. A
word an adapter does not know stays as it is, which the reducer keeps as
an expiry.

### Claude Code adapter

| Step | Design | Source |
| - | - | - |
| Detect | `claude` on the PATH, accepting only `claude.exe`, else `%USERPROFILE%\.local\bin\claude.exe`; never a `.cmd` shim. Version from `claude -v` with a 2 s timeout; nothing else runs and no status command is used. A subscription is told from an API key by the session's own report, `system/init` `apiKeySource`, which makes the quota "N/A (API key: no subscription window)"; never from credential files (plan 21, item 5.2) | CC-2; INT 1 limitations; AGENTS.md (credentials) |
| Start | `claude -p --input-format stream-json --output-format stream-json --verbose --include-partial-messages --forward-subagent-text --permission-prompt-tool stdio --permission-mode <mode>`; `cmd.Dir` = project path; never `--bare`, never `--system-prompt ""`; `CLAUDE_CODE_GIT_BASH_PATH` set in the child's environment only when Git Bash is found at its default path and not on the PATH; `CLAUDE_CODE_EMIT_SESSION_STATE_EVENTS=1` in the child's environment | CC-4 way A; CC-3; CC-8; INT 9 |
| Handshake | `initialize` control request first, 60 s timeout; its response completes the handshake. `system/init` comes only after the first user message (P2 lines 5 and 7, found in plan 4) and gives `session_id`, `model`, `cwd`, `permissionMode`, `capabilities`; the reducer then checks `cwd` equals the project path and closes the session on a mismatch | CC-4; P2 |
| Converse | user messages as stream-json lines on stdin; stdin stays open for the life of the session (background tasks die if it closes) | INT 1; P3, P3b |
| Approvals | `can_use_tool` requests answered allow (echoing `updatedInput`) or deny with a message; `permission_suggestions` and `updatedPermissions` other than `session` never sent | INT 4; CC-4; P2, P3b |
| Interrupt | `interrupt` control request | INT 5; P2 |
| Telemetry | stream messages; "/usage" right after the handshake and, with `get_context_usage`, after each turn (plan 24, amendment 23) | TEL CC |
| Models | the `initialize` response's `models` (documented `ModelInfo`) are the list; `get_settings` (undocumented, its ids `read_N`) after the handshake, after each change's answer and after each turn, reduced to `applied` and, from `effective`, the model and `env.CLAUDE_CODE_EFFORT_LEVEL`; a model by `set_model`, an effort by `apply_flag_settings {effortLevel, ultracode}` with the Ultracode in force, Ultracode by `apply_flag_settings {ultracode}`, all for the session only; `update_settings`, which writes settings files, is never sent ([Model settings](#model-settings)) | CC-13; P34c, P34c2, P34c3 |
| Close | close stdin, wait 5 s, terminate, wait 5 s, kill | CC-4; INT 9 (no POSIX signals on Windows) |

Permission mode: the startup screen asks nothing beyond the two questions
(AM4 S1 to S13), so the adapter passes `--permission-mode default`, the
CLI's own default, which prompts through the dashboard (CC-3 "Permission
modes"; ASM-4). It never passes a mode weaker than the owner's settings would
apply, and shows the `permissionMode` that `system/init` reports.

The modes (plan 12, brief amendment 8; plan 39, brief amendment 40):
every session starts Manual (`default`); the modes selector sends the
`set_permission_mode` control request, `default`, `acceptEdits`, `plan` or
`auto` (P39c: each answered within 3 ms, then reported by `system/status`;
another word refused with the CLI's list), and the mode pill and the
project bar show the mode Claude Code confirms (the control response, then
`system/status` `permissionMode`), "→ Auto …" while it is asked, and a
notice if it is refused. In Auto, an action the
classifier allows runs without a card; one it would pause still arrives as
`can_use_tool` and is a card as in Manual. A tool result's
`tool_result_meta[].permission_decision` with `reason_type:"classifier"`
(undocumented) marks the call "approved by auto-review" or "denied by
auto-review"; other reasons (`mode`, a rule, the owner) leave no mark. The
reviewer's verdict is placed on the tool row before the error text, so a
refusal's long error does not push the verdict off the row (plan 13, item
1; P13c). The mode is never saved: it is not a setting, and the CLI's
configuration is not touched (CC-11; P12c, P12c-switch).

### Codex adapter

| Step | Design | Source |
| - | - | - |
| Detect | `codex.exe` on the PATH or in `%LOCALAPPDATA%\Programs\OpenAI\Codex\bin`, the `.exe` resolved, not a `.cmd`; `codex --version` (`codex-cli 0.145.0`), with a 5 s timeout; no status command is used. A ChatGPT sign-in is told from an API key by the session's `account/rateLimits/read`, refused without a ChatGPT sign-in, which makes the quota "N/A (API key: ChatGPT sign-in needed)" (plan 21, item 5.2) | CX-2; P0 |
| Start | `codex app-server` with `cmd.Dir` = project path, in the dashboard's environment. With `config.json`'s `codex.proxy` set, `HTTPS_PROXY` and `HTTP_PROXY` carry that address instead of any inherited value (plan 22, amendment 19). Then `initialize` (`clientInfo`, and `capabilities.experimentalApi`, Codex's experimental API, for `thread/settings/update`: plan 38) and `initialized`; `thread/start` with `sandbox`, `approvalPolicy` and `model`, **without `cwd`**: every dashboard run on a new, untrusted project left Codex's configuration identical, so no `trust_level` is written (CX-10 q2, settled in plan 14) | CX-4.1; INT 1; INT shape |
| Model | the model `model/list` marks as default and does not hide (`gpt-6.1-sol` on Codex 0.163.0) is passed in `thread/start` only when the configured model is refused; the refusal is shown as received | P5, P6, P6b; CX-3.3; P34x |
| Models | `model/list` right after `initialized`, every page (`cursor` from `nextCursor`; a cursor that comes back or a hundredth page ends the list there), hidden models left out; the settings in force from `thread/start`'s `model` and `reasoningEffort`, then from `thread/read` after a turn that carried a change or after `model/rerouted`; a change between turns written at once with `thread/settings/update` (experimental: the thread and what the change asks for), answered by the `thread/settings/updated` that follows, whose `threadSettings.model` and `.effort` alone are read; one asked during the main turn written when it ends; until its notification, a change also rides on the next `turn/start` as `model` and `effort` ("for this turn and subsequent turns"), as every change does once Codex refuses the update as unknown or as needing the opt-in; kept for the next message when a write fails; no Ultracode ([Model settings](#model-settings)) | CX-13; P34x; P38x, P38t |
| Converse | `turn/start {threadId, input:[{type:"text", text}]}`, with the reviewer and sandbox below and `summary: "auto"`: the reasoning summary asked for, so a step's reasoning item starts while the model works (plan 22, amendment 20) | CX-4.1; P6b; P22s |
| Approvals | `item/commandExecution/requestApproval` and `item/fileChange/requestApproval` answered `accept`, `acceptForSession`, `decline` or `cancel`; an unknown request method gets a JSON-RPC error | INT 4; P6b, P7 |
| Experimental requests | `currentTime/read`, which the experimental API adds, answered `{currentTimeAt}` with the time in whole Unix seconds, as OpenAI's Codex extension for VS Code answers it; `turn/moderationMetadata` and `thread/readState/changed`, first seen with the opt-in and in the stable schema too, logged (plan 38) | P38t; the 0.163.0 schemas |
| Interrupt | `turn/interrupt {threadId, turnId}` | INT 5; P6b |
| Telemetry | notifications; `account/rateLimits/read` at start and after each turn | TEL CX; P6 |
| Never called | `config/value/write`, `config/batchWrite`, `skills/config/write`, `marketplace/*`, `plugin/*`, `externalAgentConfig/import`, `windowsSandbox/setupStart`, execpolicy and network amendments | CX-3.3; INT 4; INT 9; AGENTS.md |
| Close | close stdin, then the same timed escalation as Claude Code; mid-turn, stdin closed exits 0 and a job kill leaves no process (plan 4, L3; CX-10 q8) | ASM-5 |

Sandbox and approval policy: `sandbox: "read-only"` and
`approvalPolicy: "on-request"`, the combination P6b and P7 ran with the
configuration hashes unchanged; every file change then asks through the
dashboard (`item/fileChange/requestApproval`, P6b), which is what AM4 W3 asks
for. `workspace-write` is **not** used in Manual (CX-3, "Configuration
write to avoid"; ASM-6); Approve for me uses it, and Full access
`dangerFullAccess` once the owner confirms it (plan 39), below. The sandbox the
`thread/start` response reports is shown in the project bar; a downgrade on
Windows is visible, never hidden (INT 9).

The modes (plan 12, brief amendment 8; plan 39, brief amendment 40):
every session starts Ask for approval (plan 12's Manual), the read-only
sandbox with `on-request` and `approvalsReviewer: "user"` on
`thread/start`. The modes selector's three are the presets OpenAI's Codex
extension for VS Code sets, a sandbox, an approval policy and a reviewer:
Ask for approval (`readOnly`, `on-request`, `user`), Approve for me
(`workspaceWrite`, `on-request`, `auto_review`) and Full access
(`dangerFullAccess`, `never`, `user`), the last asked for only once the
owner confirms it. Between turns a choice writes the experimental
`thread/settings/update` with the three, answered by the
`thread/settings/updated` that follows (P39x); during the main turn it
waits for the turn's end; with a Codex that refuses the update as one it
does not take, it goes with the next `turn/start`, as plan 38's model
changes. Every `turn/start` carries the mode in force. `configRequirements/read`,
once after `config/read`, gives the managed requirements: a mode whose
sandbox or policy they leave out is greyed (none here: `requirements:
null`). Settings outside the three read as Custom; a `turn/start` naming no
sandbox (P12x's) is read by its reviewer, as plan 12 read it. In Auto, the escalations Codex's reviewer handles never reach the
dashboard; its `item/autoApprovalReview/started` and `.../completed`
notifications ([UNSTABLE] in the schema) become one zone B entry per
review (a subagent's in zone C and its inspection), "auto-review approved ·
risk low: …", the reviewer's authorization named from medium risk up (plan
41); `guardianWarning` repeats it and stays in the event log. A request that still reaches the dashboard is a
card as in Manual. Auto is Codex's own "Approve for me" (`codex
--approve-for-me`: the `auto_review` reviewer, the `on-request` policy, the
`workspace-write` sandbox), so each `turn/start` also carries
`sandboxPolicy`, `{type: "workspaceWrite"}` in Auto and `{type: "readOnly"}`
in Manual, sent every turn because Codex keeps a turn's sandbox for the turns
after it; the threads still start read-only, and the project bar shows the
sandbox the last turn sent. Nothing is written to `config.toml`, unlike the
TUI's permissions picker (codex-cli.md 11.1; P12x, P12x-switch,
P12x-sandbox to P12x-sandbox3).

### Mock provider

A third adapter, `mock`, for visual development and tests (B3.6). It replays
recorded or scripted event files through the same parts 2 to 4. It is not
offered on the startup screen; it starts only with an explicit command-line
flag, and every screen shows that it is mock data. The design and its
isolation from live sessions are in
[reliability.md, Mock mode](reliability.md#mock-mode). A model change it
answers itself, from the models and settings its file recorded: at once,
as Claude Code and Codex with `thread/settings/update` do, or with the next
message for a Codex file that shows the update refused (plans 34 and 38;
`testdata/mock/claude-models.jsonl`, `codex-models.jsonl` and
`codex-models-next-turn.jsonl`, made from probes P34c and P34x).

## Session context

Built once by the startup screen, immutable for the session, read by every
part (AM4 "Project awareness"; AGENTS.md "One agent CLI per session").

| Field | Value | Used by |
| - | - | - |
| `Provider` | the adapter id and display name; the detected version and executable path | part 1 spawns it; the project bar shows it |
| `ProjectName` | the last element of the resolved path | the project bar |
| `ProjectPath` | the resolved absolute path (INT paths, "Validation, in order", step 6) | `cmd.Dir` of the CLI; the project bar |
| `Branch` | the current git branch when the directory is a git work tree, read at validation from `.git/HEAD`, and again when that file changes or the branches popup switches (plan 59); otherwise absent | display only (RD "Where the images and amendment 4's text differ"); the branches popup's anchor |
| `Mock` | true only for the mock provider | every screen's mock label |
| `StartedAt` | the dashboard's clock at spawn | the session wall time (TEL CC "Duration") |

The branch is read from the file, not by running git. Since plan 59 it is
the one field that changes during a session: a switch made elsewhere,
seen by the HEAD file's watch (`state.BranchRead`), or by the branches
popup (`state.BranchSwitched`); the project does not change. Git runs only
in the Worktrees view (plan 16, amendment 10; ui-design.md, Worktrees),
Source Control (plan 44) and the branches popup (plan 59):
`internal/input/git.go` runs the `git` on the PATH inside a Bubble Tea
command, never in the reducer or the view, and never with `--force`, `-f`
or `-D`. A worktree chosen there is validated as a typed path is, and the
session context's project stays immutable (ASM-25).

## Integration strategy

**Drive each CLI through its own supported, long-lived machine interface,
and render what it reports.** Claude Code: `claude -p` with stream-json in and
out and the stdio control protocol (CC-4 way A, the interface both Agent SDKs
and the VS Code extension use; INT 1). Codex: `codex app-server` over stdio
(CX-4.1; the TUI, `codex exec` and the VS Code extension are its clients).

- **Each CLI runs its own orchestration.** Subagents, workflows, plans,
  retries, compaction and tools run inside the CLI; the dashboard observes
  their events and relays the owner's prompts, approvals and interrupts. It
  has no workflow engine, no scheduler and no agent runner of its own (B2.6;
  AM2 "Do not rebuild").
- **No terminal protocol.** No PTY, no ConPTY, no tmux pane, no screen
  scraping: the CLIs' terminal UIs are not hosted (INT 6; CC-4 way E; CX-4.3).
- **Documented first.** The documented messages are the base. The two
  undocumented pieces the MVP relies on are labelled where they are used:
  Claude Code's control-request wire form and `--permission-prompt-tool
  stdio` (INT 1, INT 4), and Claude Code's `rate_limit_event.unifiedWindows`
  (TEL CC), which updates the quota between the documented `/usage` reports
  (plan 24, amendment 23). Codex's app-server is documented but labelled
  experimental (INT 1), and the dashboard asks for its experimental API
  too, for `thread/settings/update` (plan 38), as OpenAI's Codex extension
  for VS Code does; without it, every model change goes with the next
  message.
  Each has a fallback in [reliability.md](reliability.md#fallbacks-for-every-source).
- **Read only towards configuration.** No CLI configuration is written; no
  request that writes it is sent; configuration comes from the CLI's own
  loading (CC-3; CX-3.3; AGENTS.md). See
  [reliability.md, Configuration untouched](reliability.md#configuration-untouched).
- **Claude Code first, end to end, then Codex** (SUM "Why Claude Code first";
  AM4 P7). The interface is written for both from the start, so Codex is an
  adapter, not a rework.
- **Fallback ways**, used only if the primary breaks: Claude Code one
  `claude -p --resume` per turn (way B), the Agent SDK as a sidecar (way C, a
  new runtime dependency), an MCP permission tool (way D); Codex `codex exec
  --json` (CX-4.2). What each loses is in
  [reliability.md](reliability.md#when-a-whole-integration-fails).

## How it starts

```text
agent-command-center [--mock <file>] [--provider <id>] [--project <path>]
```

1. Read the application's own configuration, if present:
   `%APPDATA%\agent-command-center\config.json`. It holds the editor's send
   keys and newline keys (ASM-7; plan 21, item 5.1), and Codex's proxy
   (plan 22, amendment 19). Every other key and every colour is fixed in
   the code (ASM-9, ASM-13).
   - A file that cannot be used is named on the first screen with its
     problem; the default keys apply, and Codex gets no proxy.
   - A problem in the `codex` setting alone is named on its own line, and
     sets no proxy.
   - The file is only read, and it is never a CLI's configuration.
2. Run `Detect` for Claude Code and Codex in parallel, each with a 10 s timeout (ASM-21)
   (AM4 S2).
3. Show the startup screen: the agent CLI question (detected ones selectable,
   the other shown as not installed) and the project directory, prefilled
   with the current directory (AM4 S12). `--provider` and `--project` only
   prefill the answers; validation still runs and the owner still confirms
   (AM4 S14; ASM-8).
4. On confirmation, validate the path (INT paths, steps 1 to 6) and the
   provider (detected, version readable). Any failure stays on the startup
   screen with its message ([ui-design.md](ui-design.md#startup-screen)).
5. Build the session context; `Start` the provider in the project directory;
   switch to the dashboard in its "starting" state while the handshake runs.
6. On the handshake's success, the dashboard is ready for the first prompt;
   on failure, it shows the error and offers to return to the startup screen
   (AM4 S6 "clear error").
7. Quit: close the session (graceful, then forced), restore the terminal.

`--mock <file>` skips detection and starts the mock provider, labelled
([reliability.md, Mock mode](reliability.md#mock-mode)).

## Reuse and build

| Item | Reuse or build | Source |
| - | - | - |
| The four reference projects | studied, not forked or copied | SUM "Why build from scratch" |
| Codex protocol types | written by hand for the messages the MVP uses, checked against the Apache-2.0 JSON Schema by fixture tests (changed from "generated" at plan 3, approved 2026-10-16) | INT 6; SUM; plan 3 (d5b910e) |
| Claude Code wire form | written from the docs; the MIT Python SDK as reference | INT 1 |
| Codex rules (window labelling by `windowDurationMins`, 12,000-token baseline) | reimplemented from the TUI source (Rust), not copied | INT 6; TEL CX |
| TUI runtime, layout | Bubble Tea v2, Lip Gloss v2 | TI 11 |
| Editor | built; ideas from Bubbles textarea and tview, notices recorded | TI 11 item 2 |
| Clipboard | library on Windows and macOS; external tools on Linux and WSL; OSC 52 through x/ansi | TI 5.2, 5.3 |
| Workflow engine, PTY, tmux | none | B2.6; INT 6 |

## Out of scope

Not designed, not built, and no option for them (AM4 "MVP Scope"; AM5).
Google Antigravity and five of amendment 4's ten exclusions are **[Nice to have]**
([plan.md](../plan.md#nice-to-have)): built only if the owner asks for one
in a new plan.

- Gemini CLI and Google Antigravity: no adapter, option or test (AM5). The
  research record [research/gemini-cli.md](research/gemini-cli.md) is kept,
  marked out of scope.
- Advanced model configuration; cross-provider agent collaboration;
  provider switching during a session; complex session profiles; advanced
  orchestration configuration (AM4 X6 to X10).

Recent project history, project favourites and a project browser (AM4 X1
to X3) are built on the startup screen since plan 15 (amendment 9;
ui-design.md, Startup screen); since plan 52 the system's folder dialog
takes the browser's place (amendment 53; [The folder
dialog](#the-folder-dialog) below).
Git worktree management (AM4 X4) is built since plan 16 (amendment 10;
ui-design.md, Worktrees).
The automatic VS Code integration (AM4 X5) is not built: the owner replaced
it with a file viewer in the terminal and an Explorer of the project,
built since plan 17 (amendments 11 and 12; ui-design.md, File viewer and
Explorer; [File viewer](#file-viewer) below).

The provider interface stays open to another agent CLI later (AGENTS.md).

## File viewer

Plan 17 (amendments 11 and 12; ui-design.md, File viewer and Explorer).
It only reads; nothing is written anywhere.

| Part | What it does | Where |
| - | - | - |
| 5. Input | reads a file and lists the project's files, each in a Bubble Tea command, never in the update loop; refuses a path outside the project (after both paths' links are resolved), a credential file by its name before it is opened, a directory and a file over 32 MiB; a NUL in the first 8 KiB is binary; the listing leaves out `.git`, credential files and anything behind a link, at most 100,000 files. The Explorer's folders are read one at a time, when opened, with the same refusals, at most 10,000 entries; git's markers come from git's status read when it shows and kept current while it shows, the snapshot Source Control reads too (plan 44, [Source Control](#source-control); `GitCLI.Status` the snapshot's markers, the `Git` interface, so tests use a double) | `internal/input/file.go`, `explorer.go`, `git.go` |
| 4. Rendering | colours lines with Chroma (pure computation, so the rendering part may use it), the language from the name, else from Chroma's analysis, else "Plain text"; draws the floating window and the quick open over the dashboard, which is drawn behind them as without them, and the Explorer as a panel docked in the dashboard's layout (plan 43, [The right sidebar](#the-right-sidebar)); the Explorer's rows and git's markers are computed from what the application holds, a path git reports deleted merged into its folder's listing as a row, a folder deleted whole rebuilt from the paths under it and never read; where the frame drew the window (`Screen.Window`, a `Box`), for the clicks (plan 58) | `internal/render/syntax.go`, `file.go`, `explorer.go`, `select.go` |
| The application | F3, the windows' and the Explorer panel's keys and mouse (in front: the quick open, then the viewer; Esc goes back one level, to the panel; F3 closes them and shows or hides the panel, plan 43; a left click outside the window closes it as Esc's last step does and does nothing else, plan 58), the generation numbers that drop a stale read, colouring, search, listing, folder read or git status | `internal/app/file.go`, `explorer.go` |

- **Colouring runs in a command** (`Options.Highlight`, by default
  `render.Highlight`), never while drawing: a file up to 1 MiB whole, a
  larger one 1000 lines on either side of the view, again when the view
  leaves that range ("coloured around the view").
- **regexp2's clock.** Chroma gives each regexp2 match a 250 ms timeout, and
  regexp2 checks it against a package-wide clock goroutine, started on first
  use and stopped only once the last match's deadline has passed (regexp2
  v2.2.1, `fastclock.go`, `runClock`; Chroma v2.27.0, `regexp.go`). In the application that goroutine is harmless. In a
  `synctest` bubble it outlives the test and the bubble panics, so the
  application tests inject a colourer that runs `render.Highlight` on one
  goroutine started in `TestMain`, outside every bubble
  (`internal/app/main_test.go`).
- **The search** runs in a command too, at most 10,000 matches.

Plan 60 (brief amendment 62; ui-design.md, Git blame) adds git blame to
the viewer, read only, nothing written:

| Part | What it does | Where |
| - | - | - |
| 5. Input | `BlameGit`, implemented by `GitCLI.Blame`: `git blame --porcelain --contents - -- <path>` through `stream`, the file's bytes on its standard input, the file system monitor off, no optional lock, within `blameLimit` (15 s); `parseBlame` reads a commit's fields once and its later lines by hash, a line not committed by git's all-zero hash; an untracked file, no commits, a folder outside git and the time limit named (`Refusal`, `NotRepoError`; `timeoutError`, `call`'s and `stream`'s, `context.DeadlineExceeded` to `errors.Is`); `FileText.Raw`, the bytes `ReadFile` read, CRs kept | `internal/input/blame.go`, `git.go`, `file.go` |
| 4. Rendering | `FileView.Blame` (`FileBlame`, `BlameCommit` with its texts made once); the column left of the numbers, 25 cells or 14 where the code would keep fewer than 60 (`blameCols`, `blameCell`), `FileCodeWidth` without it; the footer's commit or note after a toast, the mouse capture's note and a waiting approval (`blameFooter`, `windowNotice`); the key hints and the key overlay's `b` | `internal/render/blame.go`, `file.go`, `keys.go` |
| The application | `b` in the viewer, on for the session (`toggleBlame`: on a file shown, a second read compared with the lines shown before git is asked); the blame of a file just shown from its read's bytes (`blameFile`); an answer kept only for the same file and the latest read, by the file's generation and the read's number (`blameMsg`, `blameRead`); the commits' texts in `Options.Location`, nil the machine's zone (`blameCommits`, `initials`) | `internal/app/blame.go`, `file.go`, `app.go` |

Plan 61 (brief amendment 63; ui-design.md, File viewer) adds the clicks:

| Part | What it does | Where |
| - | - | - |
| 4. Rendering | `fileWindow` and `finderWindow` give the first line or file their rows show and how many; `floatWindow` records them as `Screen.WindowList` (`ListShown`: from the row under the path or field row, between the side borders), beside plan 58's `Screen.Window`; `ListShown.At` maps a cell to its line or file, -1 off the list | `internal/render/file.go`, `select.go` |
| The application | in `fileMouse`, a left click on a line puts the viewer's cursor on it, the view anchored on the rows drawn, before `press` starts the selection as before; in the quick open, `finderClick` selects the file clicked or opens the selected one as Enter does; both judge the click by the frame drawn last | `internal/app/file.go` |

## Images attached to a message

Plan 31 (brief amendment 31; ui-design.md, Input editor; the research in
doc/research/image-input.md); the paste and the preview, plan 32 (brief
amendment 32; doc/research/image-paste.md).

| Part | What it does | Where |
| - | - | - |
| 5. Input | the system's Open dialog (GetOpenFileNameW, Explorer style, owned by the window in front, on a locked thread with COM), in a command; each file read and checked by its bytes, never its extension: PNG, JPEG and GIF by the standard library, BMP, baseline TIFF, ICO and the WebP header by the dashboard's own readers; an EXIF orientation applied, an animated GIF's first frame and the converted formats sent as PNG, a GIF's first frame decoded alone and its other frames only walked by their blocks, refused for what `DecodeAll` refused of the structure (plan 49); an image over the provider's limits scaled down by area averaging; HEIC, AVIF, JPEG XL, JPEG 2000, PSD, TGA, DDS, HDR, EXR, camera raws and SVG refused by name; the original never written; each file's SHA-256 kept, to tell it changed since (plan 33); each dashboard's stored draft in a folder of its own, its lock held open with no sharing while the draft is its own, its images listed beside its text, its pasted images kept as their SHA-256, the project's stored drafts taken back at a start (its own CLI's first, then the newest), unused files swept, a rename refused by another program tried again (plan 33); the clipboard's image (the "PNG" format's bytes, else the bitmap as PNG) and copied files, through golang.design/x/clipboard; a pasted image's bytes read and checked as a file's are, written only into the stored draft's folder; a thumbnail cut from the centre to a square, rounded and bordered | `internal/input/image*.go`, `dialog*.go`, `draft.go`, `clipboard.go`, `imagethumb.go` |
| The application | the "+" menu's item opens the dialog; the accepted images are the draft's attachments, removable by a chip's ✕ or Ctrl+Shift+X; at send each is read and checked again in the write command, a file gone or changed failing the send with the draft and its images kept, a file whose bytes changed since it was attached read again in its place (plan 33); Alt+V, and Ctrl+V's release when Windows Terminal kept its press (undocumented, probe P32a), attach the clipboard's images, and a paste of copied files' paths attaches them; a pasted image's bytes in memory, by key, and in the stored draft's folder, both dropped once removed or accepted (plan 33); the stored draft written at once when its images change, with a message waiting for its acceptance and what was typed since, let go at quit and Back, a failed save tried again every second (plan 33); the device attributes request at start turns the preview on with Sixel; the thumbnails drawn with Sixel 50 ms after the frame that shows them and after any frame that changes their rows, a vacated place written over with the frame's own cells | `internal/app/attach.go`, `app.go`, `draft.go`, `paste.go`, `preview.go` |
| 2 and 3. State | the message's images ride on the write command and stay with the turn until it is accepted, a failed write giving them back; the owner's entry is named from them | `internal/state/actions.go`, `reduce.go` |
| 1. Providers | `Send(provider.Message)`: Claude Code's base64 image blocks before the text, Codex's `image` inputs with data URLs; each adapter's mapper keeps the line it wrote without the image's bytes, and Codex's every line it reads too, since the app-server writes the message back with its images (the event, the log and the screen hold the media type and size) | `internal/provider/claude/images.go`, `codex/images.go` |
| 4. Rendering | the chips' row under the editor's rule, the images under a sent message's text; with Sixel, the strip of the thumbnails' blank cells above the chips (panel 'T', ThumbCells by the terminal's rows) and the Sixel encoder (median cut to 255 colours, transparent pixels unset) | `internal/render/attach.go`, `editor.go`, `cards.go`, `sixel.go` |

## Model settings

Plan 34 (brief amendment 34; ui-design.md, Input editor, The model
selector; the research in claude-code.md and codex-cli.md, section 13);
the Ultracode toggle as an image, plan 35 (brief amendment 36); the
cursor and a pointer that has not moved, plan 36 (brief amendment 37);
the effort slider, plan 37 (brief amendment 38); Codex's changes at once,
plan 38 (brief amendment 39).
The models come from the CLI, never from the code; the screen shows a
model, an effort or Ultracode only once the CLI reports it in force.

| Part | What it does | Where |
| - | - | - |
| 1. Providers | two events: `models.listed`, the CLI's own list (each entry's id, model, name, description, effort levels, default level), and `models.settings`, the settings in force (model, effort, Ultracode and its availability, the variable that holds the effort) or a change's answer (`Answers`, `Refused`, the CLI's words). Claude Code: the list from `initialize`; a change written at once and its answer followed by a `get_settings` read, whose event alone says what applied, the read reduced to the values above before it becomes an event so no other setting is logged or shown; `CLAUDE_CODE_EFFORT_LEVEL` read from the dashboard's environment and from the read's `env` (its value only). Codex: the list from every `model/list` page; between turns a change written at once with the experimental `thread/settings/update`, its answer the `thread/settings/updated` that follows, `NextTurn` when Codex refuses the update as one it does not take (plan 38); during the main turn kept, and written at its end; until answered, and for good after `NextTurn`, written with the next `turn/start`, its answer followed by `thread/read`; a refused turn answers the change as refused; `currentTime/read` answered with the time | `internal/event/event.go`, `internal/provider/provider.go`, `claude/models.go`, `claude/mapper.go`, `claude/session.go`, `codex/models.go`, `codex/mapper.go`, `codex/session.go`, `mock/session.go` |
| 2 and 3. State | `Models`: the list, the settings in force merged from the reports, the change asked for until its answer; the choices as actions (`ChooseModel`, `ChooseEffort`, `ChooseUltracode`), each checked against the list (a level the model lists, Ultracode only while available, no effort a variable holds) and handed to the session as a `CmdModels` command; a change taking effect at once, Claude Code's and Codex's between turns (plan 38), one at a time and none while a turn runs, a Codex one of what is in force asking nothing; a change waiting for the next message, Codex's during a turn or after the next-turn event, which turns the change waiting for its answer into one (plan 38), a new choice adding to it and a choice of what is in force cancelling it; a refusal, or what the CLI kept, a notice in zone B; a change still waiting when the session ends dropped, said; a refused `turn/start` ending the submitted turn, the message given back; zone B's speaker and zone A's model follow the model in force; a new CLI process (a restart, a resume) starts with no list and no settings | `internal/state/models.go`, `reduce.go`, `actions.go` |
| The application | the pill's click and Alt+P open the selector, on a screen with room for it, the focus left where it is; its keys (a configured send key as Enter; on Effort, ← and → asking for the level before or after the one the slider shows, plan 37), hover, clicks (on a slider's cell, its level) and wheel, the hover only onto another row than the pointer's last reported one, the first report after Alt+P only recording it (`ModelsView.Pointer`, plan 36); a paste closes it; a frame that could not draw it closes it (`View`), so no key acts unseen; a change written through the write queue, in order with the owner's messages, a failed hand-over a notice; with Sixel, the Ultracode toggle's image drawn over its cells with the preview's thumbnails, after the frame and after any frame that changes its row or its state, its cells written first in the same synchronized output, its place written over once it goes (plan 35), and the effort slider's the same way, its thumb gliding to a new level in force in 180 ms, drawn every 30 ms (`slideMsg`), its clock from the frame that changed the label (plan 37) | `internal/app/models.go`, `app.go`, `keys.go`, `mouse.go`, `preview.go` |
| 4. Rendering | the pill from the settings in force; the popup over the dashboard, its list scrolled inside, the Effort and Ultracode rows on shared columns, and what the frame drew of them (`Screen.Selector`); the effort slider (plan 37): with Sixel, 17 blank cells on the ground whatever the band, each pill cell carrying its nearest level, their place and the slider's look (`Screen.Slider`), its image of 170x20 pixels (`SliderImage`), and its Sixel, made once a look on a level (`SliderSixel`); without, 16 text cells (`sliderText`); the rows' hit areas for the mouse (panels 'M' and 'S'); "→ … (next turn)" in the editor row's middle; the Ultracode toggle: with Sixel, five blank cells on the ground whatever the band and their place and the toggle's state (`Screen.Toggle`), its image of 50x20 pixels, the pill in the first 40 and the band's last cell after it (`ToggleImage`), and its Sixel, made once a state (`ToggleSixel`); without, five cells of Powerline's half circles (plan 35) | `internal/render/models.go`, `slider.go`, `toggle.go`, `editor.go`, `select.go` |

## Permission modes

Plan 39 (brief amendment 40; ui-design.md, Input editor, The modes
selector; reference-design.md, The modes selector; P39c and P39x). The
modes are each CLI's own, applied through the CLI; the screen shows a mode
in force only once the CLI reports it, and Codex's Full access is asked
for only after the owner confirms it.

| Part | What it does | Where |
| - | - | - |
| 1. Providers | `SetApprovalMode(mode)`, the dashboard's word (`event.ModeManual`, `ModeAcceptEdits`, `ModePlan`, `ModeAuto`, `ModeFull`); the events `approvals.mode` (`event.Mode`: the mode in force, an answer, a refusal or Codex taking it with the next message, in the CLI's words) and `approvals.allowed` (`event.ModeLimits`: the modes the requirements leave out, with why). Claude Code: `set_permission_mode` with its word, another refused unwritten. Codex: the three presets (`modePresets`), `thread/settings/update` between turns, at a turn's end during one, with the next `turn/start` after a refusal it does not take; `codexMode` naming the settings Codex reports; `configRequirements/read` once. The mock: its file's CLI's modes, each answered at once | `internal/event/event.go`, `internal/provider/provider.go`, `claude/session.go`, `claude/mapper.go`, `codex/session.go`, `codex/mapper.go`, `codex/models.go`, `mock/session.go` |
| 2 and 3. State | the catalogs (`ModeOptions`: titles, descriptions, icons, the warning colour); `ChooseMode` checked (the session ready and capable, the CLI's own mode, not left out, Full access confirmed) and handed over as `CmdSetApprovals`; the mode asked for until the CLI reports it (`ApprovalRequested`, `ModeNextTurn`), a refusal or the next-turn fallback a notice in zone B; `ModesUnavailable` from the requirements | `internal/state/modes.go`, `reduce.go`, `actions.go`, `state.go` |
| The application | F4 and the mode pill's click open the selector, on a screen with room for it, one popup at a time; its keys, the hover (plan 36's rule; back to the mode in force off the options), clicks and the Full access confirmation (`modesAct`, `OpenConfirm`, the pointer's next report a baseline); a paste and a frame without room close it; the change written through the write queue; with Sixel, the strips' and the pill's images drawn after the frame with the preview's, the pill's title raster cached a title (`pillLabel`), a pill without a font drawn as text (`UI.PillNoFont`) | `internal/app/modes.go`, `app.go`, `keys.go`, `mouse.go`, `preview.go` |
| 4. Rendering | the mode pill (`modePill`: with Sixel blank cells for `PillImage`, compact where the middle leaves no room for its title; without, text) and its place (`Screen.ModePill`); the box (`modesBox`): its rows, half rows and colours, the band, the confirmation, its place above the pill (`modesPlace`), its items for the mouse (panels 'O' and 'Q'); with Sixel, the strips' blank cells and their looks (`Screen.Modes`), the strips' images (`StripImage`: the band's rounded ends, the icons, the check) and the Sixel of each, made once a look (`StripSixel`, `PillSixel`) | `internal/render/modes.go`, `modesimage.go`, `editor.go`, `select.go`, `palette.go` |
| Input | the pill's title drawn by Windows' GDI in Cascadia Mono, Consolas where it is missing, as a coverage mask (`TextMask`; an error elsewhere, the pill then text) | `internal/input/textmask_windows.go`, `textmask_other.go` |

## The right sidebar

Plan 43 (brief amendment 44; ui-design.md, Right sidebar and Explorer;
reference-design.md, The right sidebar). The Explorer is the panel docked
between zone B and the sidebar; the sidebar's second icon, Skills' then, a
placeholder, is Source Control's since plan 44 ([Source
Control](#source-control)), its panel in the same place, one at a time. No
provider or state change: the panel's and the sidebar's state is the
interface's (`render.UI`).

| Part | What it does | Where |
| - | - | - |
| 4. Rendering | the upper row: zone A, zone B, the docked panel while shown (`PanelShown`), docked where zone B keeps 64 columns, else in zone B's place, and the sidebar, zone C under them (`layout`, `PanelCompact`); the panel (panel 'R': the header and its "⌄", the rule, the root row, the tree, git's note) from the Explorer's view as before; the sidebar (panel 'K') and its two tiles: with Sixel 18 blank cells each, their places and looks (`Screen.Tiles`), the images of 60x60 pixels (`SideImage`) and their Sixel, made once a look (`SideSixel`); without, the text form; the popups clamped to zone B's right edge; the footer's cues where the panel has the focus or takes zone B's place (`keyFooter`) | `internal/render/dashboard.go`, `explorer.go`, `sidebar.go`, `sidebarimage.go`, `select.go` |
| The application | F3 and the tile's click toggle `ui.Explorer.Open`, the focus with it; the panel's keys (focus 'R'), clicks and wheel; the compact rule in `syncUI` (zone B taking the focus where the panel takes its place hides it); the draft's clipboard keys and pastes beside the docked panel (`draftBeside`); the hover from motion reports (`SideHover`), all-motion reporting while the sidebar shows, and the program's filter (`motionFilter`, `tea.WithFilter` in `run.go`) dropping, before `Update` and `View`, a motion report that changes nothing, counted; the tiles' images drawn with the preview's (`previewSignature`), after the frame, again after any frame that changes their rows and after every size report | `internal/app/explorer.go`, `file.go`, `keys.go`, `mouse.go`, `app.go`, `run.go`, `preview.go` |
| Third party | ultraviolet's renderer, patched (`third_party/ultraviolet/PATCHES.md`, patch 3): a row holding a wide character, which it repaints rather than diffs, is written only up to its last changed cell, and not at all when unchanged, so an image beside the change stays | `third_party/ultraviolet/terminal_renderer.go` |

## Source Control

Plan 44 (brief amendment 45; ui-design.md, Source Control and Diff view;
reference-design.md, Source Control). The project's Git changes in the
docked place and a file's diff in zone B's place, read only. No provider or
state change: git is the input part's, the views are the interface's
(`render.UI`).

| Part | What it does | Where |
| - | - | - |
| 5. Input | `GitCLI.Snapshot`: one `rev-parse --show-prefix --show-toplevel` and one `status --porcelain=v1 -z --branch --untracked-files=normal --ignored -- .`, through `git -c core.fsmonitor=false`, no optional lock, no prompt, English; `ParseSnapshot` gives each entry's X and Y, its project and repository paths, a rename's or copy's source, the branch line (a detached HEAD's short id by a third `rev-parse`), and the Explorer's markers as `ParseStatus` gives them (`Status` is the snapshot's markers); a project folder untracked as a whole told (`Untracked`). `GitCLI.Sides`: a file's two sides for one comparison, HEAD's and the index's blobs (stage 0, a conflict's stage 2) by one `cat-file --batch`, no filter or text conversion, the working file from the disk with the file viewer's refusals; binary, Git LFS pointer and too large told apart, no data kept for them. `UntrackedFiles`: an untracked folder's files, with `ListDir`'s rules, those git ignores left out by the caller (`GitMarks.IsIgnored`). The `SourceGit` interface, so tests use a double; every git call is `rev-parse`, `status` or `cat-file` | `internal/input/source.go`, `git.go`, `explorer.go` |
| A pure package | `diff.Compare`: two texts' lines (a CR before an LF dropped), a minimal Myers difference in linear space, hunks with their context; bounded (`MaxLines` lines a side of the middle between the common head and tail, `MaxEdits` edits): past them the head and tail kept and the middle removed then added (`Whole`); line endings alone told (`EOLOnly`), a last line with no newline (`NoNewlineOld`, `NoNewlineNew`); no file, no process, no clock (test 7 allows it beside the five parts) | `internal/diff/diff.go` |
| 4. Rendering | the panel (panel 'G': the header and its "⌄", the rule, the repository's row, the sections and their rows from `SourceRows`, each row's cells its index, the cursor's band and the hover's ground) in the docked place (`dockPanel`, `PanelShown`, `PanelCompact`); the diff (panel 'D': its title and "+N −M", the path and comparison row, the unified or side by side rows from `LayoutDiff`, made once a diff, only the rows in view drawn) in zone B's place (`DiffBox`, `PanelUnderDiff`), zone B not drawn while it shows; the sidebar's tooltip (`sideTooltip`); the footer's keys while either has the focus | `internal/render/source.go`, `diffview.go`, `dashboard.go`, `sidebar.go`, `select.go`, `palette.go` |
| The application | the tile's click and F3 switch the docked panel (`toggleSource`, `showSource`, `openExplorer`); git's status read in a command, one at a time, when a panel shows, on "r", after a tool call or a turn ends and every 3 s while a panel or a diff shows, the interval ten times a slow read, at most 30 s (`readSnapshot`, `snapshotRead`, `armPoll`); one read updating both views (`applySnapshot`); the panel's keys, clicks, wheel and hover (focus 'R'); a file's sides read and compared in a command with a generation, compared again only when a side's stamp changed (`compareOpen`, `diffRead`); the diff's keys and wheel (focus 'D'); zone B's state untouched while the diff shows | `internal/app/source.go`, `explorer.go`, `keys.go`, `mouse.go`, `app.go` |

Plan 45 (brief amendment 46; ui-design.md, Commit and push;
reference-design.md, Commit and push) adds the panel's commit controls,
the application's only writes to a repository. Still no provider or state
change.

| Part | What it does | Where |
| - | - | - |
| 5. Input | the branch line's upstream, ahead, behind and gone (`parseBranch`); the `SourceWrite` interface, implemented by `GitCLI`: `Stage` (`git add -A`, literal pathspecs on standard input), `Commit` (a rebase, am, cherry-pick or revert refused; the whole repository's staged entries read, a conflict, staged changes outside the project or a staged set other than the expected refused; `git -c user.useConfigOnly=true commit --file=-`; success only once HEAD is a new commit on the old one), `PushTarget` (`symbolic-ref`, `for-each-ref`, `remote`: a detached HEAD, no remote, no upstream, a local or gone upstream, push settings sending the branch elsewhere refused) and `Push` (to the upstream only, `--porcelain`, tags and submodules off, ssh never prompting on the console, rejections read from the porcelain lines, URLs' user info removed); each refusal a `Refusal` with its message, a remote's or git's rejection a `PushRejected`; the writes with no time limit, never forcing. The owner's two additions: `Snapshot` keeps each staged entry's blob in the index (`GitEntry.Index`, `stagedBlobs`: one `cat-file --batch-check` of `:0:<path>`; `Status`, the Explorer's, reads none) and `Commit` compares them with the index's (design D14); `Push(dir, stop)` takes a `GitStop`: git started in a process group of its own, its pid held by a handle while the stop is armed (`arm`, `holdPid`), `Stop` refused while one of the push's lock files exists, else a console Ctrl+Break to that group (`breakGroup`), the check and the break under the stop's lock, which git's end takes before letting the pid go; a stopped push the `cancelled` refusal, a lock file left named (`cancelledPush`) (design D15) | `internal/input/sourcewrite.go`, `gitstop.go`, `gitstop_windows.go`, `gitstop_other.go`, `explorer.go`, `source.go` |
| 4. Rendering | the message box, the button and the note or question under the repository row (`commitControls`, items `ItemMsg` to `ItemNotice` of panel 'G'), the dropdown over the panel's rows under the button (`commitMenu`), in text or as blank cells whose images the frame records (`commitShown`: `Screen.Commit`, `Menu`, `MsgBox`); the images (`CommitImage`, `MenuImage`, `MsgBoxImage` and `MsgStrips`, encoded once a look; a running operation's button, `CommitLook.Live`, encoded each time and not kept); the note while an operation runs (`BusyHint`, in the attention colour with `BusyWarn`); the footer's keys for each place, "Esc cancel the push" while one runs; the list's room less the controls' rows (`SourceRoom`) | `internal/render/commit.go`, `commitimage.go`, `source.go`, `select.go`, `palette.go` |
| The application | the message field, a second `input.Editor` kept as the focus and the panels change; the panel's keys, clicks and the dropdown's hover by place (`commitKey`, `commitClick`, `menuHotAt`); the main action from the snapshot (`commitState`); an operation's steps (the destination, the staging, the commit, the push) run one command at a time, one operation at a time, the last or failing step reading git's status in the same command (`startOp`, `runStep`, `stepDone`); the elapsed time, a tick a second while an operation runs, a tick of an ended or earlier one dropped (`opTickMsg`, `opTicked`, `elapsed`); Esc cancelling a running push through its `GitStop` (`cancelPush`, `pushing`); the notes; the images drawn after the frame with GDI's labels, a running operation's kept only while it shows (`commitSequence`, `textLabel`, `liveLabel`) | `internal/app/commit.go`, `source.go`, `mouse.go`, `paste.go`, `preview.go`, `app.go` |

Plan 46 (brief amendment 47; the first three fixes of
[audit.md](audit.md), Part 2) changes no part's role.

| Part | What changes | Where |
| - | - | - |
| 5. Input | `GitCLI.Merge` reads the main worktree first (`mergeable`): a merge in progress or uncommitted changes refuse it before any `git merge`; a conflict is aborted only when MERGE_HEAD is the merged branch's tip, so a merge someone else started is never aborted | `internal/input/git.go` |
| 2. Collection, 3. State | `event.Problem.Optional`: the Codex mapper marks a refused `config/read` or `model/list` page; the reducer's handshake rule ends the session only on an Error that is not Optional | `internal/event/event.go`, `internal/provider/codex/mapper.go`, `internal/state/reduce.go` |
| The application | opened untracked folders read again after each status read and each opening, the reads numbered per folder so that an older one never replaces a newer one (`readFolders`, `folderSeq`, `folderSeen`) | `internal/app/source.go`, `commit.go`, `app.go` |

Plan 47 (brief amendment 48; the rest of [audit.md](audit.md)'s phase 1)
changes no part's role either.

| Part | What changes | Where |
| - | - | - |
| 1. Execution | `provider.ErrNoPending`, wrapped by both adapters' "no pending request" errors, their text unchanged | `internal/provider/provider.go`, `claude/session.go`, `codex/session.go` |
| 3. State | `Quit.GitRunning`: Ctrl+Q asks while a git operation runs, whatever the session's phase; `AnswerFailed.Gone`: an answer that a withdrawal beat expires its approval, the session going on, and a late withdrawal is no second answer; the Mode case returns; an unreadable request's word from `CLI()` | `internal/state/actions.go`, `reduce.go` |
| 4. Rendering | the quit question names the git operation (`UI.GitRunning`) | `internal/render/dashboard.go`, `render.go` |
| 5. Input | `realPath` gives a drive letter path, or `\\server\share\…`, without the final path's `\\?\` (`dosPath`) | `internal/input/realpath_windows.go` (`browse_windows.go` until plan 52) |
| The application | the quit key passes whether a git operation runs; `syncCommit` names its step; an answer error matching `ErrNoPending` is `Gone`; `dropSourcePopups`, called by `closeSource` and `openExplorer` | `internal/app/keys.go`, `commit.go`, `app.go`, `source.go`, `explorer.go` |

Plan 48 (brief amendment 49; [audit.md](audit.md)'s phase 2, robustness
at the boundaries) changes no part's role either.

| Part | What changes | Where |
| - | - | - |
| 1. Execution | the Claude Code reader writes nothing itself: its handshake reads and the read after a change run apart (`apart`); in both adapters the reader recovers a panic into an error marked `Fatal` and drains stdout, the goroutines started for writes recover, and the mapping holds the mutex in functions that release it by `defer` (`mapOut`, `holdIn`); `provider.Recovered`, a panic as text | `internal/provider/claude/session.go`, `codex/session.go`, `provider/recovered.go` |
| 2. Collection | `event.Problem.Fatal` | `internal/event/event.go` |
| 3. State | a running call's streamed output keeps its tail (`MaxLiveOutput`, `liveTail`, `ToolCall.OutputCut`); a fatal error or `Panicked` crashes a starting or ready session and closes its CLI (`fatal`) | `internal/state/reduce.go`, `actions.go`, `state.go` |
| 4. Rendering | the note of output not kept (`cutNote`); a request's bidi controls and zero-width spaces drawn as `<U+XXXX>` on its card, answered row, questions and full view (`revealed`) | `internal/render/inspect.go`, `viewer.go`, `text.go`, `approval.go`, `question.go` |
| 5. Input | the TIFF reader's entries left in the file (`tiffEntry`); an icon's PNG size checked first; git's `WaitDelay`, `exec.ErrWaitDelay` taken as git's success, `stream` read beside `Wait` | `internal/input/imagetiff.go`, `imageico.go`, `git.go`, `sourcewrite.go` |
| The application | the write worker recovers into `Panicked` (`runWrite`); the detection recovers into the CLI's problem | `internal/app/writer.go`, `app.go` |

Plan 50 (brief amendment 51; ui-design.md, Source Control, The selection,
and Commit and push) adds a selection of files that Commit commits alone.
No provider or state change.

| Part | What changes | Where |
| - | - | - |
| 5. Input | `SourceWrite.CheckSelection` and `CommitSelection`: a rebase, am, cherry-pick, revert or merge in progress (`merging`), a conflict, a selected file changed since it was shown, partly staged or deleted in the index beside its file on disk (`WholeRefusal`, which the panel asks too), a rename's source outside the project or back on disk, each a `Refusal` before any write, git's status read path by path (`--no-renames`) so that a file moved on disk, both its rows selected, stays two paths once staged; then `git -c user.useConfigOnly=true commit --only --file=-`, the files and a rename's source as literal pathspecs from a file, git building the commit from HEAD in its own transaction and leaving the rest of the index; after it, HEAD a new commit on the old one (`madeOn`, `Commit`'s too), holding exactly the selection, each blob the index's (`holds`, `blobsAt`), else an `Unexpected`, the commit made | `internal/input/sourcewrite.go`, `source.go` |
| 4. Rendering | `SourceView.Selected` and `AskSel`; a selected file's row with "▎" on the sidebar's selected ground (`sourceRow`), "Commit N files" (`ButtonLabel`), the selection's question (`AskSelText`), the hints with a selection (`sourceSelHints`), the key list's Ctrl+click | `internal/render/source.go`, `commit.go`, `dashboard.go`, `keys.go` |
| The application | Ctrl+click on a file row (`sourceCtrlClick`), a plain click on a file row clearing the selection (`sourceClick`), Esc's clearing, the selection kept to the changes at each status read (`stillChanged`); Commit with a selection (`commitSelected`): a file not whole refused at once, files not staged the question, then the steps check, stage and commit (`selectionSteps`, `stepCheck`); after a failed commit the question's staging said to stay (`stayStaged`), for plan 45's question too; the selection cleared by a commit made from it | `internal/app/source.go`, `commit.go`, `mouse.go` |

## The branches popup

Plan 59 (brief amendment 61; ui-design.md, Branches). Under the project
bar's branch name, the repository's branches and tags, searched and
folded; a switch, a fetch, a name copied; the header's branch kept
current. The view is the interface's (`render.UI.Branches`); the state
holds only the branch and the switch's notice; git is the input part's.

| Part | What it does | Where |
| - | - | - |
| 3. State | `BranchRead`, the project's branch read again (`Context.Branch`); `BranchSwitched`, the branch and the dashboard's notice in zone B (`Role: "dashboard"`, never sent to the CLI): "Switched from A to B", the changes that came along, "the agent's conversation still reflects A" while a session runs | `internal/state/branches.go`, `actions.go` |
| 4. Rendering | the project bar's "branch `<name>`" as panel L; the popup, panel J, a popup layer (`LayerBranches`, closed by `Stack.only` when not shown): `BranchesView`, the list (`BranchRows`: the groups, their folds, the search, the messages, the cap's note; `BranchItem.Key` made when asked, `BranchCursor` comparing fields), the box under the name (`branchesPopup`: the rows in view only, `BranchesShown` for the clicks and the wheel), its width counted without drawing (`branchAsideWidth`), its footer (`branchesFoot`, a text three rows at most), the Nerd Font icons where plan 55's rule draws them (`nfBranch`, `nfTag`, `nfRefresh`, registered as the Explorer's); `Screen.BranchShown`; the key overlay's rows | `internal/render/branches.go`, `dashboard.go`, `layers.go`, `select.go`, `render.go`, `keys.go` |
| 5. Input | `BranchGit`, implemented by `GitCLI`: `Branches` (`rev-parse --show-toplevel`; `for-each-ref` over `refs/heads`, `refs/remotes` and `refs/tags`, each ref's name, HEAD mark, worktree path, upstream and tracking, date, a remote's `HEAD` left out, tags newest first; the head by `symbolic-ref` and `rev-parse`; Recent from `reflog show`'s "checkout: moving from A to B", local branches only, `RecentMax` 5; `BranchCap` 1,000 rows a group), `Changes` (`status --porcelain`'s lines), `Switch` (`git switch`, or `git switch --track` for a remote branch, refused when its local name exists; the branch checked out given back), `Fetch` (`fetch --all --no-prune` through `exec`: no prompt, ssh's askpass refused, a `GitStop` waiting out the lock files of `packed-refs`, `shallow` and each remote branch, URLs' user info removed; `FetchLimit` 2 minutes, kept by the application), `HeadFile` (`rev-parse --git-path HEAD`); the reads through `runRead`; never `--force`, `-f`, `--discard-changes`, `--merge`, a stash, a reset or a prune | `internal/input/branches.go`, `git.go`, `gitstop.go`, `sourcewrite.go` |
| The application | Alt+B and the name's click (`toggleBranches`: a toast outside a repository and where `render.Layers` would not show it), Alt+B passed on by the docked panels, their commit controls and the diff as Alt+P is (`explorerKey`, `sourceKey`, `globalKey`, `diffViewKey`), the keys (`branchesKey`: any other key closes it and goes on, a send key is Enter), the clicks, the wheel and a paste (`branchesClick`, `branchesWheel`, `branchPaste`; `editorTakesPaste` leaves the popup out); the switch (`askSwitch`, `branchChanges`, `runSwitch`, `branchSwitched`): refused while the agent works (`State.Running`) or Source Control's git runs, the changes counted and confirmed, then the state's notice, the empty draft's note, the Explorer or Source Control read again; the fetch (`fetchBranches`, `stopFetch`, `fetchLimit`, `fetched`), outliving the popup; the quit question's "the switch" and "the fetch" (`branchOpRunning`, `syncBranches`); the HEAD watch from the validation (`watchHead`, `headFound`, `headTick`, `headChecked`: one `os.Stat` every `Options.BranchWatch`, 2 s in the real run, and `readBranch` when the time changed); the header showing the session's branch on the dashboard | `internal/app/branches.go`, `app.go`, `keys.go`, `mouse.go`, `layers.go`, `paste.go`, `explorer.go`, `source.go`, `commit.go`, `cmd/agent-command-center/main.go` |

## The Git Graph

Plan 62 (brief amendment 64; ui-design.md, Git Graph). The sidebar's third
tile shows the repository's history in the docked place: lanes, labels, a
commit's files and their historical diffs, a scope of references, the
cursor's row in the footer and a card beside a row. The view is the
interface's (`render.UI.Graph`, `render.UI.Scope`); the state holds none of
it; git is the input part's.

| Part | What it does | Where |
| - | - | - |
| 4. Rendering | the panel Y (`graphPanel`: the header, the scope row, the rows in view only; `GraphRoom`), the history's compact rows (`GraphEntry`, about 90 bytes, its first parent's hash, its parents' count), the files and labels by commit; the lanes (`GraphLanes`: laid out commit by commit from a running state, each row kept as its events, the lanes whole every 64 rows, a row drawn from the nearest; `viewLanes`, the lanes shown, the overflow column, `Screen.GraphHidden` for the footer's note); the wide view (`GraphWide`); the scope popup, panel V, a popup layer (`LayerScope`; `ScopeRows`, `scopePopup`, `ScopeShown`); the footer (`graphFooter`) and the card (`graphCard`, `Stack.Card`, under every popup and window); the third tile, its icon and its tooltip; the key overlay's rows | `internal/render/graph.go`, `graphlanes.go`, `graphscope.go`, `graphcard.go`, `palette.go`, `dashboard.go`, `layers.go`, `select.go`, `sidebar.go`, `sidebarimage.go`, `keys.go` |
| 5. Input | `GraphGit`, implemented by `GitCLI`: `GraphRefs` (`for-each-ref` over `refs/heads`, `refs/remotes` and `refs/tags`, a tag peeled to its commit, a symbolic reference left out; `rev-parse` of HEAD off a branch), `GraphScope.Tips`, `GraphLog` (one `git log --topo-order -z` process read page by page through a pipe, `graphPageLimit` a page, ended by `Close` or its last commit, `ErrGraphClosed` after), `CommitFiles` (`diff-tree -r -z --raw --numstat -M`, `--root` for a root, `GraphFileCap` 100,000 files listed), `CommitSides` (one `cat-file --batch`), `CommitMessage` (`log -1 --format=%B`); every read through `runRead` or `stream`, `GIT_NO_LAZY_FETCH=1` on the graph's calls alone, a name that is no hash refused before git | `internal/input/graph.go`, `git.go` |
| The application | Alt+G and the tile (`toggleGraph`, `showGraph`: the other panels hidden, Source Control's diff kept), the reads (`graphRefsRead`, `readGraph`, `nextGraphPage`, `graphPaged`: the tips compared before each page, the history read again when they moved, a commit never twice, the guard; `graphLabels`; the lanes laid out afresh for each read), the watch (`graphTick`, `Options.GraphWatch`, 2 s in the real run, while the panel shows), the place kept by hash (`keepGraphPlace`, `putGraphPlace`), the keys, the clicks and the wheel (`graphKey`, `graphClick`, `graphWheel`), a commit's files (`unfoldCommit`, `readCommitFiles`, `graphFilesRead`) and a file's diff (`openGraphDiff` through `diff.Compare`); the scope popup (`openScope`, `scopeKey`, `applyScope`, `scopeClick`, `scopeWheel`, `scopePaste`); the card (`graphHover`, `graphCardPin`, `graphCardShow`, `readCommitMessage`, its messages and files kept up to `graphCardKept`), the motion filter's rule for the graph's rows | `internal/app/graph.go`, `graphscope.go`, `graphcard.go`, `app.go`, `keys.go`, `mouse.go`, `layers.go`, `paste.go`, `source.go`, `explorer.go`, `branches.go`, `commit.go`, `cmd/agent-command-center/main.go` |

## The home screen

Plan 51 (brief amendment 52; ui-design.md, Startup screen; reference-design.md,
The home screen). The startup screen redrawn as the owner's image: in picture
mode its header, panels and footer are Sixel images of the design at a scale,
its banner a moving scene; in grid mode the same structure in text. The
startup machine gains the form's reset; the rest is the interface's.

| Part | What it does | Where |
| - | - | - |
| 3. State | Ctrl+N's form reset, while `Detecting` or `Asking` (`ResetForm`; events-and-state.md, Startup machine) | `internal/state/actions.go` |
| 4. Rendering | the mode and the banner's rows at a size (`HomeLayout`); the look (`homeLook`: the rows, the focus, the notes, the panels' growth); the picture's painter (`HomePaint`: rounded boxes, lines, icons, the glyphs a `Glyphs` gives), its cells a click acts on (panel 'H', `HomeItemOf`); its tiles, their changes and their Sixel, each tile's 32 commonest colours exact (`HomeRects`, `HomeChanges`, `BottomBlock`, `HomeSixel`); grid mode (`homeGrid`); the banner's scene, seeded (`Banner`), and its view over a copy of the background (`BannerView`): a frame's tiles of what moved, written in the view's fixed palette, the pixels it did not change transparent (`Frame`, `BannerSequence`), the sprites' images (`spriteImage`); the encoder reading each band once (`encodeSixel`) | `internal/render/home.go`, `homepaint.go`, `hometile.go`, `homegrid.go`, `banner.go`, `bannerview.go`, `bannerart.go`, `sixel.go`, `select.go` |
| 5. Input | a glyph drawn by GDI in Cascadia Mono, from the hosting Windows Terminal's package, loaded privately, else installed, else Consolas; four times larger, averaged down (`Glyph`) | `internal/input/textmask_windows.go` |
| The application | the picture drawn whole once the size has settled (`homeSettle`, 120 ms), then the cells a new look changes and the rows the frame wrote again, after the frame with the preview's images (`homeSequence`), its images encoded at once (`homePic.encode`); the banner's view kept with the picture (`homeBanner`), a frame every `render.BannerTick` while the window has the focus (`bannerMsg`, `tea.FocusMsg` and `tea.BlurMsg`, `View.ReportFocus`), none and no view when the banner is hidden; the clicks (`homeMouse`), F12, Ctrl+N (`resetForm`); the header's version constant and system; since plan 53 (amendment 54), a project chosen in the list, by ↑/↓ or a click on its row or star, put into the field (`chooseProject`); since plan 54 (amendment 55), the wheel over the PROJECTS panel moving the selection one project a notch (`homeWheel`, `moveProject`), the panel's other cells carrying the list's item (`HomeItemList`: `homeTargets`, its runs under the rows' in `homeTargetRow`; `homeListArea` in grid mode) | `internal/app/home.go`, `app.go`, `keys.go`, `mouse.go`, `preview.go`, `projects.go`; `internal/render/home.go`, `homegrid.go` |

## The folder dialog

Plan 52 (brief amendment 53; ui-design.md, The folder dialog). The home
screen's folder button and Ctrl+O open the system's dialog for choosing a
folder, and its answer goes into the project field; plan 15's project
browser is removed. No state changes: the dialog is input, its answer the
field's text, its failure a note.

| Part | What it does | Where |
| - | - | - |
| 4. Rendering | the dialog's failure as a "✗" note in NEW SESSION (`UI.FolderProblem`, `homeNotes`) | `internal/render/home.go`, `render.go` |
| 5. Input | the common item dialog for folders (IFileOpenDialog with FOS_PICKFOLDERS, through COM by its method table, x/sys's LazyDLL), owned by the window in front, on its own locked OS thread with COM, opening at a start folder (`PickFolder`); none outside Windows; `realPath` and `dosPath` moved out of the browser's files | `internal/input/folder_windows.go`, `folder_other.go`, `realpath_windows.go`, `realpath_other.go` |
| The application | the button's click and Ctrl+O open it in a command, one at a time, none while validating, at the field's directory, else its nearest existing parent, else the starting folder (`openFolder`, `folderStart`); the folder chosen in the field with the focus, a failure kept until the next dialog or Ctrl+N (`folderPicked`, `resetForm`); `Options.PickFolder`, `input.PickFolder` in the real run | `internal/app/folder.go`, `home.go`, `keys.go`, `app.go`, `cmd/agent-command-center/main.go` |

## The Explorer's icons

Plan 55 (brief amendment 56; ui-design.md, Explorer). In the acc tab, with
Cascadia Mono NF installed for the user (`%LOCALAPPDATA%\Microsoft\Windows\Fonts`,
`HKCU\...\Fonts`), the Explorer's rows draw Nerd Font icons and chevrons;
anywhere else, today's glyphs. Since plan 56 (amendment 57) the font is
bundled in the repository (`third_party/fonts`, `THIRD_PARTY.md`) and the
launcher installs it for the user when Windows has no font of that name,
under Windows Terminal only, so that a clone shows the icons from
Terminal's next start. Whether the terminal has the font is decided
twice, by the same rule: by the launcher, which names the face in the acc
tab's fragment before the tab opens, and by the dashboard, which draws the
icons. The rule: the acc profile (`WT_PROFILE_ID`), the font registered for
the user or the machine, and its files made before the Windows Terminal
among the process's ancestors started, as Terminal reads fonts when it
starts. The files' creation time stands for the install's, so a font's
files and its registration go in together, as Windows' installer puts them;
a registration made from a shell of the Claude desktop app stays in that
app's private registry (MSIX), where Terminal never reads it (plan 55,
found at step 5). No state changes: the flag is read once at start and
rendered.

| Part | What it does | Where |
| - | - | - |
| 4. Rendering | the icons by name, kind of name and extension, their colours with 16-colour fallbacks painted with the palette (`expIconFor`, `expIcons`, `paintExpIcons`); the rows' chevrons and icons when `UI.NerdIcons`, an ignored or deleted row's icon in its name's style, widths unchanged (`explorerRow`, `rootRow`) | `internal/render/expicons.go`, `explorer.go`, `palette.go`, `render.go` |
| 5. Input | the rule (`nerdIconsUsable`); the font's registry values and its files' creation time (`nerdFontInstalled`), the hosting Windows Terminal and its start (`hostTerminal`, the walk `terminalDir` uses since plan 51); false outside Windows (`NerdIcons`) | `internal/input/nerdfont.go`, `nerdfont_windows.go`, `nerdfont_other.go`, `textmask_windows.go` |
| The application | `Options.NerdIcons` into `UI.NerdIcons`; `input.NerdIcons(os.Getenv)` in the real run | `internal/app/app.go`, `cmd/agent-command-center/main.go` |
| The launcher | the same rule in PowerShell (`NerdFontReady`, the hosting Terminal's start from `TerminalStart`), the fragment's `"face": "Cascadia Mono NF"` when it holds, none otherwise; before it, since plan 56, the bundled font installed when missing: its files checked against `SHA256SUMS`, copied (the same bytes already there reused, other bytes left and the install stopped), registered under HKCU, their creation time the registration's, loaded and announced, one line printed and left 3 s; a failure one line, the launch going on | `scripts/acc.ps1`, `install-font.ps1`; `third_party/fonts/cascadia-code-2407.24` |

## The ping

Plan 63 (brief amendment 65; ui-design.md, The ping). When the agent needs
the owner while the window lacks the focus, a BEL in ACC's own Windows
Terminal profile, whose fragment's `bellStyle` makes it a flash of the
taskbar button, and the reason in the window's title; since plan 64
(amendment 66), a Windows notification at each ring, under the app ID of a
Start menu shortcut the dashboard makes. No state changes: the
application reads the needs from the state and keeps what it rang;
nothing is drawn in the frame.

| Part | What it does | Where |
| - | - | - |
| 4. Rendering | the startup screen's red note when there are no notifications, and why (`UI.NotifyProblem`, `homeNotes`) | `internal/render/home.go`, `render.go` |
| 5. Input | the user's Start menu and Windows Terminal's alias from Windows' known folders (`NotifyPaths`); ACC's shortcut made when missing, with its app ID, `NotifyAppID`, and never rewritten, one of that name with another app ID left as it is (`SetUpNotifications`, `makeShortcut`, `readShortcut`: IShellLinkW, its property store, IPersistFile); `Notifier.Show` and `Clear` through the Windows Runtime's ToastNotificationManager (combase.dll, method tables, each call on its locked thread with the runtime initialised: `withRuntime`), one tag and group, the content silent (`toastXML`); `Notifications` gives the calls or why there are none; nothing outside Windows | `internal/input/notify.go`, `notify_windows.go`, `notify_other.go` |
| The application | focus reports on every screen (`View.ReportFocus`; `tea.FocusMsg`, `tea.BlurMsg`, `blurred`); after each message, the needs that came while the window lacked the focus (`pings`: an approval or a question added to `State.Queue` and pending, a turn closed in `State.Stages` completed or failed, the session `Crashed`, each looked at once), those that no longer wait dropped (`waits`), the focus back (`pingFocused`); a ring at most every `pingEvery`, 5 s, a need within it rung at its end (`pingMsg` through `Options.After`): one BEL through `tea.Raw` where `WT_PROFILE_ID` is `input.AccProfile`, read through `Options.Getenv` (`ring`, `bellHere`), and, where `Options.Notify` is set, the notification of the most urgent need rung, its generic text (`notify`, `notification`), the calls one at a time in Update's order (`notifyCall`, as the write queue), taken out once nothing waits (`unnotify`) and at the end (`stopNotifying`, in `Shutdown`); `Options.NotifySetUp`, `input.Notifications` in the real run, made in a command at the start, off the first frame's path (`notifySetUpMsg`); the window's title from the most urgent need (`title`, `View.WindowTitle`), cleared by Bubble Tea when the program ends | `internal/app/ping.go`, `notify.go`, `app.go`, `cmd/agent-command-center/main.go` |
| The launcher | the fragment's `"bellStyle": "taskbar"` | `scripts/acc.ps1` |

## The turn review

Plan 65 (brief amendment 67; ui-design.md, Turn review; events-and-state.md,
Turn review; reliability.md, The turn review). Under each turn in zone B,
the files it changed in the project and each one's change in the diff
view, from git snapshots of the project at the turn's start and at its end,
taken in a store of the dashboard's own. The state asks for the snapshots
and keeps what they gave; the store is the input part's; the application
runs them off the loop, in order.

| Part | What it does | Where |
| - | - | - |
| 3. State | each stage's review (`Stage.Review`, `TurnReview`: its trees, the files each snapshot left out, the files between, "still changing", its problem), whether the turns are reviewed (`State.Reviewing`); the commands (`CmdTurnSnapshot` before the send's `CmdWrite`, at a stage the send did not open, at the turn's end or a crash during it; `CmdTurnFiles` once both trees are known and differ), the actions (`ReviewStore`, `TurnSnapshot`, `TurnFiles`), a CLI-opened stage starting at the stage before's end (`reviewOpen`), the editor's notice while the start is taken | `internal/state/review.go`, `actions.go`, `reduce.go`, `state.go` |
| 4. Rendering | the review's rows under a turn's last entry, counted in its span (`reviewAnchors`, `reviewCount`, `withReview`): the summary as a run's fold row, the files as the Git Graph's (`graphFileRow`), the files left out, the cap's note, the failure's row (`reviewRows`, `reviewSummary`, `reviewFileRow`, `reviewLeftRow`, `reviewProblem`); the entry cursor's stops and keys on them (`UI.EntryReview`, `ReviewStops`, `ReviewStopsAll`, `ReviewFile`, `ReviewRowsAfter`, `reviewFooter`), the file rows for a double click (`ConvView.Files`); the diff's "Turn 2: start → end" (`DiffView.Commit`, `DiffView.Turn`); the key overlay's rows | `internal/render/review.go`, `dashboard.go`, `cards.go`, `content.go`, `select.go`, `diffview.go`, `viewer.go`, `keys.go`, `render.go` |
| 5. Input | the session's store (`OpenTurnStore`: `sweepTurns`, the stores no dashboard holds removed, folders of 16 hex digits only; `readTurnRepo`, the repository read once with git's own environment, its filter drivers among it; `create`, the folder through `makeStoreDir` with the user's access list alone, SDDL `D:P(A;OICI;FA;;;<user SID>)`, its lock held as a draft's, the repository's index copied while sharing its deletion, `copyIndex`); a snapshot (`TurnStore.Take`: the listing, `git status --porcelain=v2` against the store's index, `list` and `parseTurnStatus`, the credential names excluded by pathspecs, `credentialExcludes`; the file viewer's checks before git reads a file, `check` and `checkFile`, which `ReadFile` shares, a nested repository left out, a file with more than one name too, `linkCount` opening it for its attributes alone (the review of 2026-10-29); the projection and the measure, `turnCost`, `measure`, `TurnStoreLimit`; `git add -A` of exactly those, then `write-tree --missing-ok`, `ahead` when an add ran and its tree did not; a stopped git's `index.lock` removed, `unlock`); the listing once more (`Changing`); the files and the sides through the Git Graph's reads, tree names only (`Files`, `Sides`: `CommitFiles`, `CommitSides`, `snapshotTree`); `Size`; `Close`, once the calls under way have ended; every call through `GitCLI`'s `env` and `opts` (`reads` with the repository's objects as alternates, `writes` with none, whatever the environment says) | `internal/input/turns.go`, `turns_windows.go`, `turns_other.go`, `git.go`, `file.go` |
| The application | the store made when the session starts, in a command (`openTurns`, `turnStoreRead`, `Options.TurnStore`, `input.OpenTurnStore` in the real run); the snapshots in their own ordered queue (`turnCall`, `runTurn`), the start's the gate the next write waits behind, from the send, 5 s at most (`turnSnapshot`, `turnGate`), the end's 30 s at most with the listing once more; the files read in a command (`turnFiles`); a file's diff (`openTurnDiff` through `diff.Compare`, "r" again, Esc back to zone B); the cursor's walk over the review's stops and its keys (`stepCursor`, `lastStop`, `putCursor`, `revealRow`, `reviewKey`), a double click on a file (`press`); the store removed at the end (`closeTurns`, in `Shutdown`) | `internal/app/review.go`, `app.go`, `inspect.go`, `keys.go`, `mouse.go`, `source.go`, `cmd/agent-command-center/main.go` |

## The conversation's search

Plan 66 (brief amendment 68; ui-design.md, Zone B, Search). Ctrl+F's bar at
the foot of zone B finds a query in what B holds as text, the newest match
first, lights the matches in B's rows and in an entry's full view, and
steps through them, B scrolled to each. No state changes: the reducer and
its machines are as they were, no event, nothing to the CLI; the view is
the interface's (`render.UI.Search`), found again in Update from the state
as it is, with no command that could race the entries.

| Part | What it does | Where |
| - | - | - |
| 4. Rendering | the matcher (`ConvMatches`, `ConvMatch`): the entries B draws (`searchable`), their parts in the rows' order (`entryParts`: a message's lines read in place; a tool row's name, target and error; a request's first line and target), the scan (`findIn`: the query's first byte in either case, then `strings.EqualFold`, each case's next place kept), `MaxMatches` from the newest; the targets read once while the bar is open (`SearchMemo`); the bar's row and the footer (`SearchView`, `searchBar`, `tailFit`, `searchFooter`), a layer over the dashboard and under the windows (`LayerSearch`, `FullView`), B's conversation a row shorter (`zoneB`), the input drawn without its caret; a match's place (`SearchPos`, `AtOrBefore`, `rowEnd`; B's last row shown, `ConvView.Bottom`), the folds it is behind (`MatchFolds`), its row in B (`ConvMatchRow`, over `convStarts` and `entryRows`) and in the full view (`EntryViewRow`, `viewOffset`); the lighting (`lightSearch`, before the selection's, `stMatch` and `stMatchCur`): the targets' references (`markTarget`, `retarget` through `fitDetail`; `refTarget`, `refTargetCut`), a long paste's fold row (`refFold`, `crow.more`), none a selection takes; in the full view, `lightView`; the key overlay's rows | `internal/render/search.go`, `dashboard.go`, `cards.go`, `approval.go`, `content.go`, `text.go`, `select.go`, `layers.go`, `editor.go`, `render.go`, `keys.go` |
| The application | Ctrl+F from any focus of the dashboard, a window or full view keeping its own (`openSearch`: the popups and the diff closed, the focus in B); the field and its keys (`searchState`, `searchKey`, `keyText`: AltGr's characters typed, `searchPaste`, `syncSearch`); the matches found again after a keystroke and after applied events (`searchAgain`; `State.Applied` compared in `syncUI`), the current one kept by its place; the steps and the reveal (`stepMatch`, `revealMatch`: the folds opened, B held, from the last frame's `ConvView`); Tab's full view (`searchWhole`); Esc and what else closes it, the focus given back, B's entry cursor with it (`closeSearch`, `searchKey`'s Esc, `searchAfterMouse`, Ctrl+W in `key`) | `internal/app/search.go`, `app.go`, `keys.go` |

## The characters typed with AltGr

Plan 67 (brief amendment 69; ui-design.md, Focus and keys, Typed
characters). Every field types the characters AltGr types. The
application alone: no state, rendering or input change, nothing sent to a
CLI, the decoder as it is.

| Part | What it does | Where |
| - | - | - |
| The application | the one rule of the fields (`keyText`, beside `editKey`: a key's text with no Ctrl and no Alt, or with both, the decoder giving a text to AltGr's state alone among the Ctrl and Alt chords); the fields typing through it (`editKey`: the draft, the draft window, the startup screen's project field, Source Control's message box, the Worktrees view's new branch; `branchesKey`, its printable test kept; `searchKey`; `fieldKey`, the go-to field keeping digits; `finderKey`; `scopeKey`); the approval card swallowing a key's text (`cardKey`); a modifier's own press, which Windows Terminal sends before the key it comes with, kept by the branches popup (`modifierKey`) | `internal/app/keys.go`, `branches.go`, `search.go`, `file.go`, `graphscope.go` |

## The keyboard reader's races

Plan 68 (brief amendment 70; reliability.md, Not verified). On Windows,
ultraviolet's reader runs two goroutines, one reading the console's input
records and one scanning them into events; the patched copy keeps what
each one owns apart. The application is unchanged.

| Part | What it does | Where |
| - | - | - |
| Third party | ultraviolet's Windows reader, patched (`third_party/ultraviolet/PATCHES.md`, patches 4 and 5): each console read's batch sent as a copy, the receiver's to keep; the sender's grapheme state a scanner of its own; tested on the real console path, in a child process with a console of its own and no window (`patch_windows_test.go`, `TestConsoleReader`) | `third_party/ultraviolet/terminal_reader_windows.go`, `patch_windows_test.go` |
| Checks | the copy's tests shown to run in `go test ./...`, each reporting PASS and none skipped (`copyTests`, `TestThirdPartyTable`) | `internal/licenses/third_party_test.go` |

## The session's resumption from the home screen

Plan 69 (brief amendment 71; reliability.md, Resumption; ui-design.md,
Startup screen, Last session). The home screen offers the chosen agent
CLI's last session in the chosen project and resumes it through that
CLI's own resume; the crash resume's path is as it was.

| Part | What it does | Where |
| - | - | - |
| 1. Input | the store's field (`Project.Sessions`, `Session`: an ID and when the owner's last turn ended, per provider id, an optional field of version 1); `Record` keeping it with the star, `RecordSession`, `ForgetSession` (only while the record names the refused session), `LastSession` (the entry `Same` matches) | `internal/input/projects.go` |
| 2. Normalisation and adapters | Claude Code: `--resume <id>` as for the crash resume; the refusal before the handshake's answer recognised for the home screen's resume only (`Mapper.ResumeAsked`, `noConversation`, `event.Problem.NoSession`). Codex: `thread/resume` with `excludeTurns: true` for the home screen's resume only (`threadParams.ExcludeTurns`, `Session.last`); its refusal for want of the thread (`noThread`); `thread/goal/cleared` logged. The resume told apart from the crash resume by `provider.SessionContext.ResumeLast` | `internal/provider/provider.go`, `claude/mapper.go`, `claude/session.go`, `codex/mapper.go`, `codex/session.go`, `internal/event/event.go` |
| 3. State | `Confirm`'s session to resume and its project through validation (`StartupState.Resume`, `ResumeProject`; a validated path not the record's refuses the start), the start naming it (`SessionState.Resumed`, `CmdStartSession` with `Last`), zone B's note once the CLI names the session (`resumedNote`), a refusal for good (`CmdForgetSession`, `StartupState.ResumeGone`) or any other failure (`ResumeFailure`), the record at the owner's turn end (`CmdRecordSession`) | `internal/state/actions.go`, `reduce.go`, `state.go` |
| 4. Rendering | the home screen's two notes, the offer a click target (`HomeNote.Item`, `HomeItemResume`; `homeTargets`, `homeNoteRows`), zone B's starting line and failure card (`sessionParts`, `failureCard`), a resumed session's N/A reasons (`missing`) | `internal/render/home.go`, `homepaint.go`, `homegrid.go`, `dashboard.go`, `telemetry.go`, `render.go` |
| The application | Ctrl+R and the click on the home screen (`resumeLast`), the notes' text (`resumeNotes`, `lastSession`), the start's flag (`start`), the record's writes and its clearing through the store (`recordSession`, `forgetSession`) | `internal/app/lastsession.go`, `app.go`, `keys.go`, `home.go` |

## Build order

Input to plan 3's roadmap, from AM4's priorities (P1 to P6) and AGENTS.md
("First make it work"):

0. The pure reducer and event types, tested from P1 to P7 recordings, and the
   mock provider (so the screens can be developed without quota).
1. Startup screen: detection and path validation (AM4 P1).
2. Claude Code adapter: start in the project, handshake, close (AM4 P2).
3. Conversation end to end: editor, streaming, approvals, interrupt,
   clipboard (AM4 P3; AM3 M13).
4. Zones A and C from real events (AM4 P4).
5. Zone D from real telemetry (AM4 P5); since plan 24 the telemetry is zone
   A's pane.
6. Codex adapter through the same steps 2 to 5 (AM4 P7).
7. Visual refinement against the references (AM4 P6), never at the cost of a
   function (AM4 P8).

## Assumptions

Choices with no plan 1 finding or brief item behind them; each is to be
confirmed in plan 3 or measured in plan 4.

| Id | Assumption | Where |
| - | - | - |
| ASM-1 | Go's current stable release when plan 4 starts | Stack |
| ASM-2 | `golang.design/x/clipboard` rather than atotto (both fit TI 5.2; MIT needs no extra notice handling) | Stack |
| ASM-3 | Rendering at most 60 frames per second, Bubble Tea v2's default (`defaultFPS`, its `renderer.go`; no `tea.WithFPS` is passed); the events waiting, at most 512, applied as one update, none coalesced by kind. Every live review ran at this rate, where a key waits at most 17 ms for its frame (plan 21, item 5.4) | Threads; ui-design.md |
| ASM-4 | Claude Code runs with `--permission-mode default` | Claude Code adapter |
| ASM-5 | Codex shutdown uses the same timed escalation as Claude Code | Codex adapter |
| ASM-6 | Codex runs with `read-only` and `on-request`, every write approved in the dashboard (CX-10 q2, settled in plan 14: no `trust_level` written); Auto's turns run `workspaceWrite`, the owner's choice (plan 12 addendum, brief amendment 8) | Codex adapter |
| ASM-7 | The application's own configuration file: `%APPDATA%\agent-command-center\config.json` (`os.UserConfigDir`), holding the send keys and the newline keys (plan 21, item 5.1) and Codex's proxy (plan 22, amendment 19) only; JSON, which Go's standard library reads, where TOML would need a new dependency | How it starts |
| ASM-8 | `--provider` and `--project` prefill the startup answers, never skip validation | How it starts |
| ASM-9 | Default key bindings beyond those TI fixes; only the send and newline keys are configurable (ASM-7), the others fixed in the code | ui-design.md |
| ASM-10 | The editor budget: a keystroke drawn within 16 ms with a 10,000-line buffer; a 50,000-line paste inserted and drawn within 100 ms | ui-design.md, reliability.md |
| ASM-11 | The dashboard's own session record, kept in memory for the session; no history file written in the MVP | reliability.md |
| ASM-12 | The layout geometry: breakpoints at 100 and 160 columns; the 80x24 minimum; from 100 columns A and B side by side in the upper row, their bottoms on one row, A 28% of the width, at least 36 columns (36 at 120, 56 at 200), C the whole width under them (plan 20, the owner's two templates; plan 24, the owner's image: A took 28% for its telemetry and zone D, once under C, was removed); C 3 rows until agents are in play, then two rows an agent (one under 100 columns), up to half the rows A, B and C share (5 rows under 100 columns; at least 8 from 160 columns, for the preview inside C, right of a 70-cell list), B keeping at least 12 rows (14 from 160 columns), except that with an agent in play C keeps one agent's two rows, 4 rows from 100 columns, A and B giving the rows (at 160x24 to 160x26, A and B 11 to 13 rows; plan 21, item 4); zone B's labels beside their text from 90 cells inside B (plans 18 and 20) | ui-design.md, Layouts |
| ASM-13 | The colours: the palette hex values, each model family's colour, a quota bar turning red from 90%; fixed in the code, not configurable (plan 21, item 5.1) | ui-design.md |
| ASM-14 | Codex threads given model "N/A" until a per-thread source appears; from Codex CLI 0.163.0, `subAgentActivity.model` is that source (P7b; plan 5, item 7) | events-and-state.md |
| ASM-15 | No message is sent while a turn runs; the draft waits (Codex `turn/steer` and Claude Code's queue are never used; CX-10 q5, closed in plan 14) | events-and-state.md, Turn machine |
| ASM-16 | The 10 s an interrupt has to end the turn before "force stop" is offered (the 5 s shutdown steps are the Agent SDK's, CC-4; ASM-5 extends them to Codex) | events-and-state.md, Turn machine |
| ASM-17 | The Markdown renderer: our own, no new dependency (the owner's decision of 2026-10-16, plan 4 step 3, after Glamour v2 measured p95 590 ms re-rendering a streamed 2,000-line message); Markdown styling and syntax highlighting built in plan 4, step 8, and extended in plans 5 and 6 | ui-design.md, Markdown |
| ASM-18 | Animation rates: spinners and the moving connector dot at 10 frames per second, only while something runs; zone B's thinking card and its transitions at 20 frames per second while one is live or in its entrance or exit (plan 22), the 10 a second otherwise; nothing scheduled with nothing running | ui-design.md, Animations; Zone B, Thinking |
| ASM-19 | Each CLI process runs in a Windows job object that ends it when the dashboard's process ends, so no CLI is orphaned | reliability.md, Separated parts |
| ASM-20 | The event reader accepts lines up to 64 MiB; a longer line is recorded as an error with its size and skipped | reliability.md, Concurrent output |
| ASM-21 | Timeouts no source gives: Codex's handshake 60 s, as Claude Code's (CC-4); each CLI's detection 10 s | events-and-state.md, Startup and Session machines; How it starts |
| ASM-22 | Display defaults: a thinking phase's details folded under its summary row (neither CLI sends thinking text: plan 22); a sent message over 20 lines folded to its first 5; zone C ordered waiting, running, ended, most recent first; copy on mouse release; the last 50 stderr lines kept for error display | ui-design.md; Threads of execution |
| ASM-23 | The editor's draft is saved byte for byte, its text at most every 500 ms and its images at once, by atomic rename, to `%LOCALAPPDATA%\agent-command-center\drafts`, one folder per dashboard's draft named by the SHA-256 of the project's path and a random key, its lock held open while the dashboard runs, its pasted images in it (plan 33); a message waiting for the CLI's acceptance is stored with what was typed and attached since; deleted once the CLI accepts the send, kept if the send fails, taken back by the project's next start, its own CLI's first, then the newest. On quit, crash or panic the dashboard prints the CLI session id and the CLI's own resume command; a panic restores the terminal, saves the draft, closes the CLI and exits with code 2. Not a history file (ASM-11) | reliability.md, Resumption; plan 3 (d5b910e) |
| ASM-24 | The startup screen's project store is `projects.json` in `%LOCALAPPDATA%\agent-command-center\`, written by atomic rename: per project its validated path, its last start, whether it is a favourite and, per agent CLI, its last session's ID and when the owner's last turn there ended (plan 69, an optional field of version 1); at most 10 recent projects, favourites never dropped; paths compared without case on Windows. No conversation data (ASM-11). A store that cannot be read or written is said so on screen; a mock run uses none | ui-design.md, Projects; plan 15 |
| ASM-25 | The Worktrees view: git calls time out after 15 s for a read and 2 min for an operation (a merge may run hooks); a diff is capped at 5000 lines, the rest counted. Sessions are registered in `%LOCALAPPDATA%\agent-command-center\sessions\`, one file per dashboard session holding its project path, kept open until the session ends; a file nobody holds is a session that ended and is deleted at the next read. It knows the dashboard's own sessions only, not an agent CLI started in another terminal, and is "N/A" outside Windows, where an open file cannot be told from a closed one; a mock run registers none | ui-design.md, Worktrees; plan 16 |

## Coverage of the brief

Every item of the brief's phases 2 and 3 and of amendments 2 to 4, with the
section that places it. AM5 removes Gemini CLI from every item that names it.
Section links: `ARCH` this file, `EVT` events-and-state.md, `UI`
ui-design.md, `REL` reliability.md.

### Phase 2, architecture

| Id | Item | Placed in |
| - | - | - |
| B2.1 | A native terminal application | ARCH Stack; ARCH Where it runs |
| B2.2 | Left: orchestrator, workflow tree, phase status, agent hierarchy | UI Zone A; EVT Agent tree |
| B2.3 | Main: live subagent activity, execution logs, progress, agent selection, diff inspection | UI Zone C; UI Agent inspection; EVT `tool.completed`, `diff.updated` |
| B2.4 | Bottom: real 5-hour and 7-day quota, reset timers, token usage, context, model, agent statistics | UI Zone A, telemetry (zone D until plan 23); EVT `telemetry.updated`; REL Telemetry sources |
| B2.5 | Bubble Tea, Lip Gloss and tmux unless better | ARCH Stack (tmux not needed, with the reason) |
| B2.6 | Reuse the CLI's native orchestration; no rebuilt workflow engine | ARCH Integration strategy |
| B2.7 | Keyboard navigation, live updates, responsive layouts, polished style | UI Focus and keys; UI Layouts at three sizes; UI Colours and symbols |

### Phase 3, reliability

| Id | Item | Placed in |
| - | - | - |
| B3.1 | Officially documented interfaces where possible | ARCH Integration strategy; REL Fallbacks for every source |
| B3.2 | Data collection separate from the interface | ARCH The five parts; REL Separated parts |
| B3.3 | Unavailable telemetry handled gracefully | REL Telemetry sources; UI States |
| B3.4 | CLI configurations intact | REL Configuration untouched |
| B3.5 | JEV optional | REL JEV |
| B3.6 | Mock-data mode | REL Mock mode; ARCH Mock provider |
| B3.7 | Tests for telemetry parsing and agent state transitions | REL Test plan |

### Amendment 2, zone A (orchestrator and workflow tree)

| Id | Item | Placed in |
| - | - | - |
| AM2 A1 | The orchestrator and its model | UI Zone A; EVT `session.started` |
| AM2 A2 | The hierarchical execution plan | UI Zone A; EVT `workflow.stage.changed` |
| AM2 A3 | Stages, dependencies, active tasks, completion status | UI Zone A; EVT Agent tree |
| AM2 A4 | Delegated subagents with parent-child links | EVT Agent tree; UI Zone A |
| AM2 A5 | Updates on workflow events | EVT Pipeline; UI Zone A |
| AM2 A6 | Keyboard navigation through nodes | UI Focus and keys |

### Amendment 2, zone B (conversation)

| Id | Item | Placed in |
| - | - | - |
| AM2 B1 | A persistent conversation panel | UI Zone B |
| AM2 B2 | A multiline input area | UI Input editor |
| AM2 B3 | Streaming responses, incremental | EVT `conversation.message.delta`; UI Zone B |
| AM2 B4 | Markdown, syntax highlighting, code blocks | UI Zone B, "Markdown" |
| AM2 B5 | Progress messages and tool activity | EVT `tool.started`; UI Zone B |
| AM2 B6 | Clarification questions and approval requests | EVT `approval.requested`, `question.requested`; UI Approvals |
| AM2 B7 | History and session continuity | REL Resumption; EVT Conversation record |
| AM2 B8 | Follow-ups without restarting | EVT Turn machine |
| AM2 B9 | Interruptions and cancellation, safely | REL Interruption and cancellation; EVT Turn machine |
| AM2 B10 | The conversation is real, not simulated | ARCH The five parts; EVT Origins |
| AM2 B11 | Preserve native interaction through documented interfaces, without recreating the terminal protocol | ARCH Integration strategy |

### Amendment 2, zone C (live subagents)

| Id | Item | Placed in |
| - | - | - |
| AM2 C1 | Active, waiting, completed, failed agents | EVT Agent machine; UI Zone C |
| AM2 C2 | Model per agent | EVT `agent.started`; UI Zone C |
| AM2 C3 | Current task and file paths | EVT `agent.progress`, `tool.started`; UI Zone C |
| AM2 C4 | Real tool executions and progress | EVT `tool.started`, `tool.completed` |
| AM2 C5 | Agent-specific logs and outputs | UI Agent inspection |
| AM2 C6 | Verification and test results | UI Agent inspection ("tool results") |
| AM2 C7 | Retry information | EVT `retry`; UI Zone C; Zone A, telemetry (notes and details view) |
| AM2 C8 | Select an agent and expand its activity | UI Focus and keys; UI Agent inspection |
| AM2 C9 | Return to the orchestrator conversation | UI Focus and keys |
| AM2 C10 | Agent activity updates independently of the conversation | ARCH Threads of execution; EVT Concurrent output |
| AM2 C11 | Watch several agents while the orchestrator talks | UI Layouts at three sizes; EVT Concurrent output |

### Amendment 2, zone D (telemetry; zone A's pane since plan 24)

| Id | Item | Placed in |
| - | - | - |
| AM2 D1 | 5-hour quota | EVT `telemetry.updated`; UI Zone A, telemetry |
| AM2 D2 | 7-day quota | same |
| AM2 D3 | Reset times when available | same |
| AM2 D4 | Model and context-window usage | same |
| AM2 D5 | Tokens and API-equivalent cost, apart from subscription usage | UI Zone A, telemetry; REL Telemetry sources |
| AM2 D6 | Active agents and concurrency limit | UI Zone A, telemetry: the details view (Codex: the limit from `config/read`, only it kept; else N/A, with the reason) |
| AM2 D7 | Execution duration | UI Zone A, telemetry: the details view |
| AM2 D8 | Errors and retry counters | UI Zone A, telemetry: the details view for the errors, the notes for the counts; EVT `error`, `retry` |
| AM2 D9 | Real telemetry; "N/A" when unavailable | REL Telemetry sources; UI States |

### Amendment 2, interaction model, events, UX

| Id | Item | Placed in |
| - | - | - |
| AM2 I1 | Launch the application | ARCH How it starts |
| AM2 I2 | Type a request | UI Input editor |
| AM2 I3 | The orchestrator responds and orchestrates | EVT Turn machine |
| AM2 I4 | The workflow tree updates | EVT Agent tree |
| AM2 I5 | Workers appear and run | EVT Agent machine |
| AM2 I6 | Live, without blocking the conversation | ARCH Threads of execution |
| AM2 I7 | Inspect, review, approve | UI Agent inspection; UI Approvals |
| AM2 I8 | Results reported in the conversation | UI Zone B |
| AM2 I9 | Continue without leaving | EVT Turn machine |
| AM2 I10 | An IDE-style command center in the terminal | UI Layouts at three sizes |
| AM2 E1 | No redraw through model calls | ARCH The five parts |
| AM2 E2 | A persistent TUI rendering state | ARCH The five parts |
| AM2 E3 | The 11 conceptual events | EVT Conceptual events |
| AM2 E4 | Mapped to real documented sources | EVT Conceptual events |
| AM2 E5 | The five separated parts | ARCH The five parts |
| AM2 E6 | No unnecessary polling | EVT Polling |
| AM2 U1 | Smooth, restrained animations | UI Animations |
| AM2 U2 | Colour-coded agent statuses | UI Colours and symbols |
| AM2 U3 | Shortcuts and panel focus | UI Focus and keys |
| AM2 U4 | Scrollable conversation and logs | UI Scrolling, collapsing, full screen |
| AM2 U5 | Expandable and collapsible panels | same |
| AM2 U6 | Responsive sizing | UI Layouts at three sizes |
| AM2 U7 | Full-screen agent inspection | UI Agent inspection |
| AM2 U8 | No flicker | UI Animations ("Rendering") |
| AM2 U9 | AI messages, observed events and inferred labels distinct | UI Origins on screen; EVT Origins |
| AM2 U10 | Input easy to reach while agents work | UI Focus and keys |
| AM2 R1-R9 | The 9 research questions | answered by plan 1, INT 1 to 9; used in ARCH Integration strategy and REL |
| AM2 V1 | Revised layout | UI Layouts at three sizes |
| AM2 V2 | Interaction-flow diagram | UI Interaction flow |
| AM2 V3 | Event and state architecture | EVT |
| AM2 V4 | Integration strategy | ARCH Integration strategy |
| AM2 V5 | Reusable components | ARCH Reuse and build |
| AM2 V6 | Limitations and risks | REL Not verified; REL When a whole integration fails |
| AM2 V7 | Updated MVP plan | ARCH Build order (plan 3 completes it) |

### Amendment 3, input and clipboard

| Id | Item | Placed in |
| - | - | - |
| AM3 M1 | Multiline typing and pasting of long text | UI Input editor |
| AM3 M2 | System clipboard copy and paste | UI Clipboard |
| AM3 M3 | Mouse selection and copy from every panel | UI Selection and copy |
| AM3 M4 | Keyboard selection, cursor, editing, deletion | UI Input editor |
| AM3 M5 | Bracketed paste never submits | UI Input editor, "Paste"; REL Test plan |
| AM3 M6 | Scrollable input with thousands of lines | UI Input editor, "Budget" |
| AM3 M7 | Configurable Enter-to-send and newline keys | UI Focus and keys |
| AM3 M8 | `$EDITOR`, replaced by the draft window (brief amendment 13) | UI Input editor, "Draft window" |
| AM3 M9 | Content, whitespace, indentation, line breaks preserved | UI Input editor; REL Test plan |
| AM3 M10 | Works across WT, PowerShell, WSL2, Linux, macOS where supported | UI What works where |
| AM3 M11 | Terminal limits investigated | TI (plan 1); UI What works where |
| AM3 M12 | Provider-independent | ARCH The five parts (part 5 has no provider code); REL Test plan |
| AM3 M13 | Essential MVP features | ARCH Build order step 3 |

### Amendment 4, startup and dashboard

| Id | Item | Placed in |
| - | - | - |
| AM4 S0 | Exactly two mandatory questions | UI Startup screen, Questions |
| AM4 S1 | Question 1: the agent CLI (Claude Code, OpenAI Codex; Gemini removed by AM5) | UI Startup screen |
| AM4 S2 | Detect whether it is installed | ARCH How it starts; ARCH adapters, Detect |
| AM4 S3 | Clear error when unavailable | UI Startup screen, messages |
| AM4 S4 | Modular provider architecture | ARCH Provider interface |
| AM4 S5 | Not hardcoded around Claude Code | ARCH Provider interface |
| AM4 S6 | Native provider functions preserved | ARCH Integration strategy |
| AM4 S7 | Question 2: manual path entry | UI Startup screen |
| AM4 S8 | Windows, Linux, macOS, WSL paths where applicable | ARCH Where it runs; INT paths; UI Startup screen |
| AM4 S9 | Expand and validate | UI Startup screen, validation |
| AM4 S10 | Exists and accessible | same |
| AM4 S11 | It becomes the working directory | ARCH Session context |
| AM4 S12 | Current directory prefilled | ARCH How it starts |
| AM4 S13 | No advanced project browser; amendment 9 added one, amendment 53 replaced it with the system's folder dialog | UI Startup screen, The folder dialog |
| AM4 S14 | Dashboard only after both are validated | EVT Startup machine |
| AM4 S15 | Startup follows image 1 | UI Startup screen, "Follows and departs" |
| AM4 D1 | Transition to the dashboard | UI Transition |
| AM4 D2 | Dashboard follows image 2 | UI Dashboard, "Follows and departs" |
| AM4 D3 | Zones A to D kept (A: stages, progress, tree, task status, observed activity; B: real conversation, input, streaming, multiline, clipboard, history, native approvals; C: active and completed agents, tasks, activity, logs, status; D: quota, context, tokens, agent count, model) | UI Zones A to C; Zone A, telemetry (zone D until plan 23) |
| AM4 D4 | The conversation is the primary surface | UI Focus and keys (default focus) |
| AM4 D5 | Real provider data; "N/A" otherwise | REL Telemetry sources |
| AM4 D6 | Providers need not expose identical telemetry | ARCH Provider interface, `Capabilities` |
| AM4 W1 | Project bar: provider, project name, path | UI Project bar; ARCH Session context |
| AM4 W2 | File operations in the project, under native permissions and sandbox | ARCH adapters; REL Configuration untouched |
| AM4 W3 | The agent inspects, modifies with approval, runs tools, reports | EVT Approval machine; UI Approvals |
| AM4 F1-F10 | The 10-step expected experience | UI Interaction flow |
| AM4 F11 | One coherent application | UI Transition |
| AM4 X1-X3 | Recent projects, favourites, a project browser: built by amendment 9; the browser replaced by the system's folder dialog, amendment 53 | UI Startup screen, Projects and The folder dialog |
| AM4 X4 | Worktree management: built by amendment 10 | UI Worktrees; ARCH Session context |
| AM4 X5 | VS Code integration: replaced by a file viewer in the terminal, amendment 11 | UI File viewer; ARCH File viewer |
| AM4 X6-X10 | The 5 excluded features left | ARCH Out of scope |
| AM4 P1-P6 | The six priorities | ARCH Build order |
| AM4 P7 | One provider end to end first | ARCH Integration strategy; ARCH Build order |
| AM4 P8 | No function traded for a visual detail | ARCH Build order step 7 |
| AM4 K1 | Launch from a terminal | ARCH How it starts |
| AM4 K2 | Select any supported provider | UI Startup screen |
| AM4 K3 | Select and validate the path | UI Startup screen |
| AM4 K4 | The provider runs in the right directory | ARCH adapters, Handshake (`cwd` check) |
| AM4 K5 | Real prompts, streamed responses | EVT `conversation.message.delta` |
| AM4 K6 | Inspect and modify files with approvals | EVT Approval machine |
| AM4 K7 | Multiline input, long pastes, copy | UI Input editor; REL Test plan |
| AM4 K8 | Dashboard reflects real activity | EVT Conceptual events |
| AM4 K9 | Telemetry accurate or marked unavailable | REL Telemetry sources |
| AM4 K10 | Both screens follow the images' layout and order | UI Startup screen; UI Dashboard |
| AM4 N1 | Update the architecture and the MVP plan | these four documents; plan 3 |
| AM4 N2 | The two images as the visual specification of the startup-to-dashboard flow | RD; UI Transition |
| AM4 N3 | Technical limitations affecting interaction with the CLIs | REL Not verified; REL When a whole integration fails; INT 9 |
| AM4 N4 | No implementation before approval | plan.md (plans 3 and 4) |

Items without a section: 0.
