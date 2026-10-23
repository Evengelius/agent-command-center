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

Every dependency above needs the owner's approval in plan 3 (AGENTS.md, "Add
nothing casually"). No dependency is copied as source; any idea taken from the
Bubbles textarea or tview (MIT) is recorded with its source, commit and
license where it is used (AGENTS.md; TI 11 item 2).

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
  the last 50 lines for error display (ASM-22).
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
  handshake and per shutdown step (INT 1; CC-4). No timer polls a CLI (see
  [events-and-state.md, Polling](events-and-state.md#polling)).

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
  Answer(ApprovalID, Decision) error  allow / deny / allow-for-session where offered
  AnswerQuestion(QuestionID, Answers) error   where the CLI supports it
  Interrupt() error                 cancel the running turn
  SetApprovalMode(auto bool) error  Manual or Auto approvals (plan 12): Claude Code at once,
                                    Codex from the next turn; refused where not supported
  Refresh(TelemetryKind) error      on-demand reads (quota, context); never a timer
  Capabilities() Capabilities       what this CLI and version support (below)
  Close(ctx) error                  graceful, then forced (reliability.md)
```

`Capabilities` is a set of flags each adapter returns, so the interface never
pretends both CLIs are the same (AM4 D6 "Do not force all providers to
expose identical telemetry"): streaming of subagent text, per-subagent model,
quota without a turn, API-equivalent cost, clarification questions, plan
steps, resume with history, allow-for-session, Manual or Auto approvals, and
the limits on a message's images (`Images`: count, bytes an image and a
message, pixels a side, patches; plan 31, doc/research/image-input.md). They
are fixed per adapter (`internal/provider/claude/session.go`,
`internal/provider/codex/session.go`, `internal/provider/mock/session.go`),
not read from detection or the handshake, and one is read:
`AutoApprovals`, which F4 needs to ask for Manual or Auto approvals
(`internal/app/keys.go`). A value no event gives stays "N/A" with its reason,
whatever the flags. The other differences come from the provider id
(plan 21, item 5.3; rewiring them to the flags would change nothing on
screen):

| What differs | Where |
| - | - |
| the answer's words: accept and decline for Codex, allow and deny for Claude Code (`decision`) | `internal/app/keys.go` |
| the word an unreadable request is refused with (`requested`) | `internal/state/reduce.go` |
| Codex's slash commands held back once (`send`) | `internal/state/actions.go` |
| the CLI's own resume command printed at the end (`ResumeCommand`) | `internal/app/run.go` |
| the two supported CLIs, their flags and their startup rows | `internal/app/app.go`, `internal/render/startup.go` |
| the project bar: Codex's model, sandbox and policy; "(next turn)" for Codex's mode change | `internal/render/dashboard.go` |
| zone B's speaker label, CLAUDE or CODEX (`providerLabel`) | `internal/render/dashboard.go` |
| the Write and Edit sentence on a pending Claude Code request (`effect`) | `internal/render/approval.go` |
| zone A's telemetry (zone D until plan 23): Codex's credits or Claude Code's overage, the concurrency limit's reason, the N/A reasons each CLI never reports | `internal/render/telemetry.go` |

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
| Close | close stdin, wait 5 s, terminate, wait 5 s, kill | CC-4; INT 9 (no POSIX signals on Windows) |

Permission mode: the startup screen asks nothing beyond the two questions
(AM4 S1 to S13), so the adapter passes `--permission-mode default`, the
CLI's own default, which prompts through the dashboard (CC-3 "Permission
modes"; ASM-4). It never passes a mode weaker than the owner's settings would
apply, and shows the `permissionMode` that `system/init` reports.

Manual or Auto (plan 12, brief amendment 8): every session starts Manual
(`default`); F4 sends the `set_permission_mode` control request, `auto` or
`default`, and the project bar shows the mode Claude Code confirms (the
control response, then `system/status` `permissionMode`), "→ Auto …" while
it is asked, and a notice if it is refused. In Auto, an action the
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
| Start | `codex app-server` with `cmd.Dir` = project path, in the dashboard's environment. With `config.json`'s `codex.proxy` set, `HTTPS_PROXY` and `HTTP_PROXY` carry that address instead of any inherited value (plan 22, amendment 19). Then `initialize` (`clientInfo`) and `initialized`; `thread/start` with `sandbox`, `approvalPolicy` and `model`, **without `cwd`**: every dashboard run on a new, untrusted project left Codex's configuration identical, so no `trust_level` is written (CX-10 q2, settled in plan 14) | CX-4.1; INT 1; INT shape |
| Model | the per-session model `gpt-5.6-sol` (the default `model/list` reports) is passed in `thread/start` only when the configured model is refused; the refusal is shown as received | P5, P6, P6b; CX-3.3 |
| Converse | `turn/start {threadId, input:[{type:"text", text}]}`, with the reviewer and sandbox below and `summary: "auto"`: the reasoning summary asked for, so a step's reasoning item starts while the model works (plan 22, amendment 20) | CX-4.1; P6b; P22s |
| Approvals | `item/commandExecution/requestApproval` and `item/fileChange/requestApproval` answered `accept`, `acceptForSession`, `decline` or `cancel`; an unknown request method gets a JSON-RPC error | INT 4; P6b, P7 |
| Interrupt | `turn/interrupt {threadId, turnId}` | INT 5; P6b |
| Telemetry | notifications; `account/rateLimits/read` at start and after each turn | TEL CX; P6 |
| Never called | `config/value/write`, `config/batchWrite`, `skills/config/write`, `marketplace/*`, `plugin/*`, `externalAgentConfig/import`, `windowsSandbox/setupStart`, execpolicy and network amendments | CX-3.3; INT 4; INT 9; AGENTS.md |
| Close | close stdin, then the same timed escalation as Claude Code; mid-turn, stdin closed exits 0 and a job kill leaves no process (plan 4, L3; CX-10 q8) | ASM-5 |

Sandbox and approval policy: `sandbox: "read-only"` and
`approvalPolicy: "on-request"`, the combination P6b and P7 ran with the
configuration hashes unchanged; every file change then asks through the
dashboard (`item/fileChange/requestApproval`, P6b), which is what AM4 W3 asks
for. `workspace-write` is **not** used in Manual (CX-3, "Configuration
write to avoid"; ASM-6); Auto's turns use it, below. The sandbox the
`thread/start` response reports is shown in the project bar; a downgrade on
Windows is visible, never hidden (INT 9).

Manual or Auto (plan 12, brief amendment 8): every session starts Manual,
`approvalsReviewer: "user"` on `thread/start`; F4 sets the reviewer the
next `turn/start` carries, `auto_review` or `user`; Codex echoes the
reviewer only on `thread/start`, so a turn's change is never confirmed. The
project bar shows "→ Auto (next turn)" until that message is sent, then the
new mode. In Auto, the escalations Codex's reviewer handles never reach the
dashboard; its `item/autoApprovalReview/started` and `.../completed`
notifications ([UNSTABLE] in the schema) become one zone B entry per
review, "auto-review approved · risk low · …"; `guardianWarning` repeats it
and stays in the event log. A request that still reaches the dashboard is a
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
[reliability.md, Mock mode](reliability.md#mock-mode).

## Session context

Built once by the startup screen, immutable for the session, read by every
part (AM4 "Project awareness"; AGENTS.md "One agent CLI per session").

| Field | Value | Used by |
| - | - | - |
| `Provider` | the adapter id and display name; the detected version and executable path | part 1 spawns it; the project bar shows it |
| `ProjectName` | the last element of the resolved path | the project bar and window title |
| `ProjectPath` | the resolved absolute path (INT paths, "Validation, in order", step 6) | `cmd.Dir` of the CLI; the project bar |
| `Branch` | the current git branch when the directory is a git work tree, read once at startup from `.git/HEAD`; otherwise absent | display only (RD "Where the images and amendment 4's text differ") |
| `Mock` | true only for the mock provider | every screen's mock label |
| `StartedAt` | the dashboard's clock at spawn | the session wall time (TEL CC "Duration") |

The branch is read from the file, not by running git, and never changed.
Git runs only in the Worktrees view (plan 16, amendment 10; ui-design.md,
Worktrees): `internal/input/git.go` runs the `git` on the PATH inside a
Bubble Tea command, never in the reducer or the view, and never with
`--force`, `-f` or `-D`. A worktree chosen there is validated as a typed
path is, and the session context stays immutable (ASM-25).

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
  experimental (INT 1).
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
ui-design.md, Startup screen).
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
| 5. Input | reads a file and lists the project's files, each in a Bubble Tea command, never in the update loop; refuses a path outside the project (after both paths' links are resolved), a credential file by its name before it is opened, a directory and a file over 32 MiB; a NUL in the first 8 KiB is binary; the listing leaves out `.git`, credential files and anything behind a link, at most 100,000 files. The Explorer's folders are read one at a time, when opened, with the same refusals, at most 10,000 entries; git's markers come from one `git --no-optional-locks status --porcelain=v1 -z --untracked-files=normal --ignored -- .` each time it opens (`GitCLI.Status`, the `Git` interface, so tests use a double) | `internal/input/file.go`, `explorer.go`, `git.go` |
| 4. Rendering | colours lines with Chroma (pure computation, so the rendering part may use it), the language from the name, else from Chroma's analysis, else "Plain text"; draws the floating window, the quick open and the Explorer over the dashboard, which is drawn behind them as without them; the Explorer's rows and git's markers are computed from what the application holds, a path git reports deleted merged into its folder's listing as a row, a folder deleted whole rebuilt from the paths under it and never read | `internal/render/syntax.go`, `file.go`, `explorer.go` |
| The application | F3, the windows' keys and mouse (in front: the quick open, then the viewer, then the Explorer; Esc goes back one level, F3 closes them all), the generation numbers that drop a stale read, colouring, search, listing, folder read or git status | `internal/app/file.go`, `explorer.go` |

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

## Images attached to a message

Plan 31 (brief amendment 31; ui-design.md, Input editor; the research in
doc/research/image-input.md).

| Part | What it does | Where |
| - | - | - |
| 5. Input | the system's Open dialog (GetOpenFileNameW, Explorer style, owned by the window in front, on a locked thread with COM), in a command; each file read and checked by its bytes, never its extension: PNG, JPEG and GIF by the standard library, BMP, baseline TIFF, ICO and the WebP header by the dashboard's own readers; an EXIF orientation applied, an animated GIF's first frame and the converted formats sent as PNG; an image over the provider's limits scaled down by area averaging; HEIC, AVIF, JPEG XL, JPEG 2000, PSD, TGA, DDS, HDR, EXR, camera raws and SVG refused by name; the original never written; the stored draft's images listed beside it | `internal/input/image*.go`, `dialog*.go`, `draft.go` |
| The application | the "+" menu's item opens the dialog; the accepted images are the draft's attachments, removable by a chip's ✕ or Ctrl+Shift+X; at send each is read and checked again in the write command, a file gone or changed failing the send with the draft and its images kept | `internal/app/attach.go`, `app.go`, `draft.go` |
| 2 and 3. State | the message's images ride on the write command and stay with the turn until it is accepted, a failed write giving them back; the owner's entry is named from them | `internal/state/actions.go`, `reduce.go` |
| 1. Providers | `Send(provider.Message)`: Claude Code's base64 image blocks before the text, Codex's `image` inputs with data URLs; each adapter's mapper keeps the line it wrote without the image's bytes, and Codex's every line it reads too, since the app-server writes the message back with its images (the event, the log and the screen hold the media type and size) | `internal/provider/claude/images.go`, `codex/images.go` |
| 4. Rendering | the chips' row under the editor's rule, the images under a sent message's text | `internal/render/attach.go`, `editor.go`, `cards.go` |

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
| ASM-23 | The editor's draft is saved byte for byte at most 1 s after the last change, by atomic rename, to `%LOCALAPPDATA%\agent-command-center\drafts`, one file per project named by the SHA-256 of its path; deleted once the CLI accepts the send, kept if the send fails, offered back on the next start in that project. On quit, crash or panic the dashboard prints the CLI session id and the CLI's own resume command; a panic restores the terminal, saves the draft, closes the CLI and exits with code 2. Not a history file (ASM-11) | reliability.md, Resumption; plan 3 (d5b910e) |
| ASM-24 | The startup screen's project store is `projects.json` in `%LOCALAPPDATA%\agent-command-center\`, written by atomic rename: per project its validated path, its last start and whether it is a favourite; at most 10 recent projects, favourites never dropped; paths compared without case on Windows. No conversation data (ASM-11). A store that cannot be read or written is said so on screen; a mock run uses none | ui-design.md, Projects; plan 15 |
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
| AM4 S13 | No advanced project browser; amendment 9 adds one | UI Startup screen, Project browser |
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
| AM4 X1-X3 | Recent projects, favourites, a project browser: built by amendment 9 | UI Startup screen, Projects and Project browser |
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
