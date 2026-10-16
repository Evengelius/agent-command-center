# Proposal

Plan 3. The proposal the brief asks for before any code (phase 4), updated by
amendments 2 to 5, for the owner's approval. It summarises plans 1 and 2 and
links to their documents, which hold the detail and the evidence; it adds the
MVP roadmap (plan 4) and the approvals needed to start it.

How to read it:
- Sections 1 to 6 are the six items of the brief's phase 4; sections 7 to 13
  are the seven deliverables of amendment 2. Where two overlap, the brief's
  item gives the design and amendment 2's gives what changed or what limits it.
- Citation keys are those of [architecture.md, Conventions](architecture.md#conventions):
  INT, TEL, TI, CC-n, CX-n, P0 to P7 (the probes), AM2 to AM5 (amendments),
  ASM-n (assumptions). **U** marks an undocumented interface.
- Verified means a probe recording, a document or a check shows it; an
  assumption (ASM-n) or a target is never presented as verified.

## The decision in brief

- **Build from scratch, in Go**, with Bubble Tea v2 and Lip Gloss v2; no
  reference project is forked ([research/summary.md](research/summary.md)).
- **Drive each CLI through its own machine interface over pipes**: Claude
  Code `claude -p` with stream-json and the stdio control protocol; Codex
  `codex app-server` (JSON-RPC). No PTY, no tmux, no screen scraping
  ([architecture.md, Integration strategy](architecture.md#integration-strategy)).
- **Events update a state; the state is rendered.** No model call draws the
  screen; every value is real or "N/A" with its reason
  ([events-and-state.md](events-and-state.md)).
- **Claude Code end to end first, then Codex**, in the owner's order of
  2026-10-16 (section 4).

## 1. Comparison of the reference projects

Brief phase 4, item 1. Detail: [research/summary.md, Why build from scratch](research/summary.md#why-build-from-scratch)
and the four files under [research/](research/).

| Project (commit) | License | What it is | Why not a fork | What is worth studying |
| - | - | - | - | - |
| [agent-dashboard](research/agent-dashboard.md) (`b3c04cf`) | MIT | observes agent TUIs in tmux, controls them by `send-keys` | no streaming, no hosted conversation, no native approvals; needs tmux and Unix; reads `~/.claude/.credentials.json`; installs hooks | transcript and subagent parsing; quota bar and subagent tree presentation |
| [claude-dashboard](research/claude-dashboard.md) (`711650f`) | MIT | tmux session switcher with screen scraping | no conversation, quota, tokens, cost or subagents; no native Windows; rewrites `~/.tmux.conf` | confirms Go with Bubble Tea and Lip Gloss fits; a counter-example for guessed statuses |
| [ccdash](research/ccdash.md) (`5263e72`) | MIT | observe-only usage dashboard, polling transcripts | no conversation, subagent tree or quota; Linux and tmux; its default launch rewrites `~/.claude/settings*.json` | usage parsers and price tables (an undocumented fallback) |
| [jev-pilot](research/jev-pilot.md) (`5d27135`) | MIT | a Claude Code plugin that steers the agent | not a dashboard; no Go code; optional parts change configuration, proxy traffic and read the credential file | read-only detection of JEV through `system/init` plugins |

All four observe agent CLIs; none converses with one. Three break hard
constraints as shipped (credential reads, configuration rewrites). The
conversation, approvals and live subagents the brief asks for come from the
CLIs' own protocols, which none of them uses (INT questions 1 to 4).

## 2. Recommended architecture and stack

Brief phase 4, item 2. Detail: [architecture.md](architecture.md) (stack,
five parts, threads, provider interface, both adapters, startup).

**Stack** ([architecture.md, Stack](architecture.md#stack)):

| Part | Choice | License |
| - | - | - |
| Language | Go 1.27.2, the stable release on 2026-10-16 (ASM-1) | BSD-3-Clause |
| TUI runtime | Bubble Tea v2 (`charm.land/bubbletea/v2`): the only Go option with one-message bracketed paste on Windows (TI 2.2, 2.3) | MIT |
| Layout and style | Lip Gloss v2, with its compositor for mouse hit testing (TI 2.5) | MIT |
| Input editor | our own component: the Bubbles textarea caps at 10,000 lines and has no undo (TI 2.4) | ours |
| Clipboard | `golang.design/x/clipboard` on Windows (no cgo); OSC 52 through `x/ansi` (TI 5.2, 5.3; ASM-2) | MIT |
| Windows process control | `golang.org/x/sys/windows` for the job object (ASM-19) | BSD-3-Clause |
| Markdown | undecided (ASM-17): Glamour v2 with Chroma v2, or our own, settled by a benchmark in roadmap step 3 | MIT |
| tmux | not used, not a dependency; the application runs inside tmux like any program | n/a |

**Five parts**, each testable alone ([architecture.md, The five parts](architecture.md#the-five-parts)):

```text
 execution and session ──► event collection ──► state ──► rendering
 (adapters, processes)     (decode, map)        (reducer,  (reads the
        ▲                                        no I/O)    state only)
        └──────────────── input and commands ◄──────────────────┘
                          (editor, keys, clipboard)
```

- **Provider interface**: `ID`, `DisplayName`, `Detect`, `Start`, and
  `Capabilities`; adapters for Claude Code, Codex and the mock provider
  ([architecture.md, Provider interface](architecture.md#provider-interface)).
  Adding a CLI later is an adapter, not a rework; no third adapter is built.
- **Native Windows program** in Windows Terminal; the VS Code terminal and
  conhost are out of scope (brief, amendments 6 and 7); the code stays portable, Linux and macOS not verified
  ([architecture.md, Where it runs](architecture.md#where-it-runs)).

## 3. Detailed terminal UI design

Brief phase 4, item 3. Detail: [ui-design.md](ui-design.md), with the
references in [reference-design.md](reference-design.md).

- **Startup screen**: exactly two questions, the agent CLI (detected ones
  selectable, the other "not installed") and the project directory
  (prefilled with the current directory), validated before the dashboard
  opens; every failure stays on the screen with its message
  ([ui-design.md, Startup screen](ui-design.md#startup-screen)).
- **Dashboard, four zones**: A the orchestrator and its stage tree; B the
  conversation with the editor and approval cards; C the live agents, each
  inspectable; D quota, tokens, context, cost and errors; a project bar above
  ([ui-design.md, Dashboard](ui-design.md#dashboard)).
- **Three sizes**: narrow from 80x24, standard from 100 columns, wide from 160
  (ASM-12), with text mock-ups of each
  ([ui-design.md, Layouts at three sizes](ui-design.md#layouts-at-three-sizes)).
- **Input editor**: multiline, bracketed paste that never submits, content
  byte for byte, undo, Enter sends and Ctrl+J adds a line (Shift+Enter where
  the terminal reports it), `$EDITOR` for long texts
  ([ui-design.md, Input editor](ui-design.md#input-editor)).
- **Clipboard and selection**: in-app mouse selection on any panel, copied
  through the native clipboard and OSC 52; what works in which terminal is a
  matrix ([ui-design.md, Clipboard](ui-design.md#clipboard),
  [What works where](ui-design.md#what-works-where)).
- **Origins on screen**: AI text, observed events, derived and inferred
  values look different; undocumented values carry †; "N/A" always gives its
  reason ([ui-design.md, Origins on screen](ui-design.md#origins-on-screen)).
- **Dark technical style** from the references: palette, symbols, animations
  only while something runs (ASM-13, ASM-18;
  [ui-design.md, Colours and symbols](ui-design.md#colours-and-symbols)).

## 4. Prioritised MVP roadmap

Brief phase 4, item 4; it fills plan 4. Order: the owner's eight priorities
of 2026-10-16, after a foundation step. Each step ends with the evaluation
summary of [evaluation.md](evaluation.md), recorded in `plan.md`; the next
step starts only when every check of the current one passes (a FAIL stops and
is reported with its cause and a proposed fix). Checks are written as yes/no
with their expected value; "live" marks a run that uses quota, done only
under the approvals of section [Approvals requested](#approvals-requested),
with the configuration check before and after. Test numbers refer to
[reliability.md, Test plan](reliability.md#test-plan).

| Step | Owner priority | Result |
| - | - | - |
| 0 | (foundation) | toolchain, skeleton, event and state core, mock mode, both screens on mock data, configuration check |
| 1 | 1 startup and project selection | the startup screen validates the CLI and the directory |
| 2 | 2 real Claude Code integration | Claude Code starts in the project; the integration gate passes |
| 3 | 3 interactive conversation, streaming | editor, clipboard, streamed turns, follow-ups, interrupt |
| 4 | 4 permissions and session continuity | approvals; cancel, crash, resume without losing drafts or sessions |
| 5 | 5 Codex integration | Codex through steps 2 to 4 |
| 6 | 6 genuine agent activity | zones A and C from real events, both CLIs |
| 7 | 7 usage and quota telemetry | zone D from real sources, "N/A" otherwise |
| 8 | 8 UI refinement | the look of the two reference images, no function lost |

### Step 0: foundation

1. Go installed as approved: `go version` prints `go version go1.27.2
   windows/amd64`; the downloaded archive's SHA-256 equals the value in
   [Approvals requested](#approvals-requested).
2. Skeleton: one module, one package per part; `go build ./...` exit 0,
   `go vet ./...` exit 0, `gofmt -l .` prints nothing, `go test ./...` exit 0
   with its count; the Commands section of `AGENTS.md` filled in.
3. Dependencies: `go.mod` lists exactly the approved direct modules at the
   approved versions; `THIRD_PARTY.md` lists every module of `go list -m
   all` with its version and license, and a test fails on any module missing
   from it; every license is MIT, BSD-2/3-Clause or Apache-2.0 (any other
   stops for approval).
4. Separation (test 7): fails if rendering imports an adapter or process
   package, the editor a provider package, or the reducer an I/O, process,
   network or clock package; run once with a planted bad import, which
   fails.
5. Fixtures: the probe recordings P1 to P7 copied to `testdata/` unchanged
   (SHA-256 equal to the files in `doc/research/probes/`).
6. Event mapping (test 1): every line of every fixture maps to at least one
   event, 0 dropped; P2 gives 25 `content_block_delta` lines, 3 of them
   `text_delta`; P7 gives 11 rate-limit pushes and 10
   `thread/status/changed`.
7. State machines (test 3): one case per transition and per invalid
   transition listed in [events-and-state.md](events-and-state.md#state-machines),
   the case count equal to the row count; replaying each fixture twice gives
   the same state (test 4).
8. Mock mode (test 8): `--mock <file>` spawns no process (a fake spawner
   fails if called); every event has origin `mock`; every frame shows
   "◆ MOCK — not a live session"; `--mock` with `--provider` or `--project`
   is refused.
9. Both screens on mock data (test 9): golden frames of the startup screen
   and the dashboard at 80x24, 120x40 and 200x60; "terminal too small" at
   79x24; no line wider than the terminal.
10. Configuration check tool (test 13): its unit tests pass; one run on this
    machine writes a baseline of digests only; it opens no credential file
    and parses no file.

### Step 1: startup and project selection (owner priority 1)

1. Path validation table test ([ui-design.md, Validation and messages](ui-design.md#validation-and-messages);
   INT paths): a Windows path, `~`, `/mnt/c/...`, `\\wsl.localhost\...`, a
   relative path, a trailing separator, a missing path, a file, a folder
   without access: each accepted, or refused with its exact message.
2. Detection with a fake `PATH`: neither CLI, either one, both; a missing CLI
   shows "not installed" and cannot be chosen; with neither, Start is
   disabled; a detection over 10 s shows its timeout message (ASM-21).
3. Detection on this machine, `--version` only (no model call): Claude Code
   2.1.300 and Codex 0.145.0 shown, or the versions installed that day.
4. Startup gate: no event sequence reaches the dashboard before both answers
   are validated (state-machine test); `--provider` and `--project` prefill
   only, validation still runs (ASM-8).
5. Golden frames of the startup screen with each message.
6. Not verified by the agent: the owner launching it from Windows Terminal
   (listed for the owner's check).

### Step 2: real Claude Code integration and the integration gate (owner priority 2)

The early validation the owner asked for: the no-PTY approach is proven over
real pipes before any conversation screen depends on it.

1. Adapter tests with a fake CLI (a test binary replaying fixtures, test
   10): handshake success; no handshake (timeout shortened in the test);
   exit mid-turn; closed stdin on send; a 65 MiB line skipped with an error
   (ASM-20); each ends in its documented state with a visible message.
2. Close escalation: close stdin, 5 s, terminate, 5 s, kill; with a fake CLI
   that ignores stdin, the process is gone within 11 s.
3. Job object (ASM-19): when the test's dashboard process is killed, the
   fake CLI's PID is gone within 1 s.
4. Working directory: the `cwd` in `system/init` equals the chosen project
   path (fixture; then live in 5).
5. **Integration gate, live (L1)**, a Go test behind a `live` build tag, in a
   temporary project outside the repository, `--model haiku` as in the
   probes: (a) at least 2 `text_delta` events before `result`; (b) a
   `can_use_tool` request answered allow writes the file, answered deny
   leaves it absent and the turn ends normally; (c) an interrupt ends the
   turn with `error_during_execution` within 10 s, and the next turn runs;
   (d) an interrupt while an approval is pending ends the turn (CC-10 q4);
   (e) after stdin closes the process exits within 10 s; (f) a new process
   with `--resume <id>` answers from the first process's history; (g) the
   same after the first process was killed mid-turn by its job object; (h)
   the configuration check shows every configuration file identical.
6. **If the gate fails**: work stops on what depends on it; the failure is
   recorded under "Not verified" or as a limitation, and the smallest
   adjustment is proposed from [reliability.md, When a whole integration fails](reliability.md#when-a-whole-integration-fails)
   (for example the MCP permission tool, way D, if only `--permission-prompt-tool
   stdio` breaks), for the owner's approval. The architecture is not changed
   silently.

### Step 3: interactive conversation with streaming (owner priority 3)

The input editor and the clipboard are essential here, not polish (AM3).

1. Byte for byte (test 5): pastes with LF, CRLF, lone CR, tabs, trailing
   spaces, combining marks, emoji, wide characters and C0 controls; the
   stored text and the bytes written to the CLI's stdin equal the expected
   text (CR stored as LF, nothing else changed).
2. A paste never submits: a paste ending in a newline leaves the message
   unsent; Enter afterwards sends exactly once; a paste with no end marker
   leaves the editor in paste state.
3. Benchmarks (test 6), reported as measured values: keystroke-to-frame 95th
   percentile with a 10,000-line buffer, target 16 ms; 50,000-line
   paste-to-frame, target 100 ms. PASS only if measured within the target;
   until then the figures are targets (ASM-10), never reported as met.
4. Streaming: replaying P2 draws the answer delta by delta (golden frames at
   successive events); a follow-up continues the same session; an interrupt
   leaves the session ready and the draft intact.
5. Conversation is real: every message in the state traces to the `Seq` of
   a CLI event; a test fails on a message without one.
6. Clipboard and selection (test 12): OSC 52 encoding and tmux wrapping; the
   native path behind a fake; a wrapped line copies as one line; borders,
   symbols and origin labels are not copied; a code block copies exactly,
   tabs included.
7. Markdown (ASM-17): a 2,000-line message streamed one delta per frame,
   target 16 ms per frame with Glamour v2; if it misses, our own renderer
   for the subset the CLIs emit, with no new dependency.
8. Same input tests with the Claude Code, Codex and mock adapters attached:
   identical results.
9. Not verified by the agent: the terminal matrix cells (Windows Terminal)
   for paste, Shift+Enter and copy, listed for the owner's manual check. The
   VS Code terminal was dropped by amendment 6 and conhost by amendment 7
   (both 2026-10-16).

### Step 4: permissions and session continuity (owner priority 4)

1. Approvals: the P2 and P3b requests appear as cards, P3b's on the
   subagent (by `agent_id`); each answer writes exactly one response; a
   second key press writes nothing; an unreadable request is denied and shown
   as such; a pending approval turns `Expired` when the turn ends; no frame
   offers an option that persists a rule.
2. Cancellation (from an approval, `c`): deny then interrupt; the session
   stays ready and the next message continues it.
3. Crash with a fake CLI after `Ready`: the crash card shows the exit code
   and the stderr tail; the record and the draft are unchanged (byte
   compare); Ctrl+R spawns `claude -p --resume <id>` with the same flags (the
   fake spawner's argv checked); a crash before `Ready` shows StartFailed.
4. Draft kept across the dashboard's own failure (ASM-23): after the
   dashboard process is killed, a restart in the same project opens with the
   draft byte for byte; a successful send deletes it; a failed send keeps it;
   saving a 50,000-line draft keeps step 3's keystroke target.
5. Panic: a panic injected in the reducer restores the terminal, saves the
   draft, closes the CLI (the fake CLI sees end of input), prints the session
   id and the CLI's own resume command, and exits with code 2.
6. **Smoke test, live (L2)**, through the dashboard on a temporary project:
   one turn, one approval, one interrupt, one Ctrl+R resume after the CLI was
   killed; configuration check before and after.

### Step 5: Codex integration (owner priority 5)

1. Codex fixtures P5 to P7 through steps 0 to 4's checks: mapping, state
   machines, approvals routed by `threadId`, interrupt `interrupted`.
2. `thread/start` with `read-only` and `on-request` (ASM-6); no
   `workspace-write`, `config/write` or `config/batchWrite` is ever sent (a
   test over every request the adapter can write).
3. Model refusal (P5, P6): the error is shown as received, then the thread
   starts with the model `model/list` reports as default (fixture).
4. **Integration gate, live (L3)**, a `live` Go test like L1 in a temporary
   project: handshake within 60 s (ASM-21); deltas before `turn/completed`;
   a file-change approval accepted writes the file; `decline` and `cancel`
   each end as documented (CX-10 q1); `turn/interrupt` ends the turn
   `interrupted`; `thread/resume` from a new process continues the thread
   (CX-10 q4); an app-server killed mid-turn by its job object leaves no
   process behind (CX-10 q8); `~/.codex/config.toml` has the same digest
   before and after.
   A failure follows step 2's rule: recorded, the smallest adjustment
   proposed (for resume: Ctrl+R hidden for Codex and the limit shown).
5. **Smoke test, live (L4)**: L2's scenario with Codex.

### Step 6: genuine agent activity (owner priority 6)

1. Zone A: the stage tree from P2, P3b and P7, each node with its origin;
   inferred stages labelled "inferred".
2. Zone C: P3b's background subagent and P7's two children appear, run and
   end with their observed statuses; ordering per ASM-22.
3. Agent inspection: Enter on an agent shows its messages, tools and
   approvals from its own events only.
4. Golden frames at the three sizes; the agent machine's table tests pass for
   both CLIs.

### Step 7: usage and quota telemetry (owner priority 7)

1. Telemetry parsing (test 2): P1 line 3 gives 22% and 45% with their reset
   times, marked †; P1 line 12 gives `total_cost_usd` 0.00249374 shown as an
   API-equivalent estimate, never as subscription usage; P6b line 7 gives
   both Codex windows by `windowDurationMins`; P7 gives the thread totals
   76,521, 30,455 and 77,336 and their derived sum 184,312.
2. Every N/A case of [reliability.md, Telemetry sources](reliability.md#telemetry-sources)
   shows "N/A" with its reason, never 0; a window past its reset turns N/A.
3. Long session (test 11): two hours of events at speed; memory reported;
   frame time at the end within the 16 ms target, reported as measured.

### Step 8: UI refinement (owner priority 8)

1. Each screen compared with [reference-design.md](reference-design.md); each
   departure is listed or removed.
2. Golden frames updated deliberately, each change reviewed; every test of
   steps 0 to 7 still passes, with a test count no lower than before.
3. Not verified by the agent: how it looks and feels, judged by the owner.

### Done

Amendment 4's acceptance criteria, each checked with Claude Code and with
Codex (two providers since amendment 5), or listed as not verified: launched
from a terminal; either CLI selectable; project path validated; the CLI
working in that directory; real prompts with streamed answers; files
inspected and changed with approvals; multiline input, long pastes and
copying reliable; real agent activity; telemetry accurate or marked
unavailable; both screens in the sequence and layout of the images.

## 5. Reuse versus develop

Brief phase 4, item 5. Detail: [architecture.md, Reuse and build](architecture.md#reuse-and-build).

| Item | Decision |
| - | - |
| The four reference projects | studied, not forked or copied |
| Bubble Tea v2, Lip Gloss v2, the clipboard library, `x/sys` | used as dependencies, pinned (section [Approvals requested](#approvals-requested)) |
| Codex protocol types | written by hand for the messages the MVP uses, checked against Codex's Apache-2.0 JSON Schema (P0) by fixture tests; no generator dependency (a change from plan 2, section [Changes to plan 2's design](#changes-to-plan-2s-design)) |
| Claude Code wire form | written from the documented messages; the MIT Python Agent SDK is the reference for the control requests; nothing copied |
| Codex display rules (window labels, 12,000-token baseline) | reimplemented from their description in the Rust TUI, not copied |
| Input editor | built; ideas from the Bubbles textarea and tview (MIT), recorded with source, commit and license where used |
| Markdown | Glamour v2 if the step 3 benchmark passes, else ours |
| Workflow engine, PTY, tmux, scheduler | none: the CLIs run their own orchestration |

## 6. Technical risks and compatibility constraints

Brief phase 4, item 6. Detail: [reliability.md](reliability.md) and its
[Not verified](reliability.md#not-verified) list.

| Risk | Effect | What contains it |
| - | - | - |
| Claude Code's control wire form and `--permission-prompt-tool stdio` are undocumented (**U**) | a release could break approvals or the session | fixture tests on each CLI version; the step 2 gate; fallback ways B, C, D ([reliability.md](reliability.md#when-a-whole-integration-fails)) |
| Codex's app-server is labelled experimental | a release could change methods | fixture tests; `codex exec --json` fallback with stated losses |
| `workspace-write` can write `trust_level` into `~/.codex/config.toml` | a configuration change | `read-only` with every write approved (ASM-6) until CX-10 q2 is probed |
| The editor misses its targets at 10,000 or 50,000 lines | typing lags | step 3 benchmark; the own editor exists for this (TI 2.4) |
| Streamed Markdown re-rendering is slow (ASM-17) | B lags on long answers | step 3 benchmark; own renderer as fallback |
| A crash of the CLI or of the dashboard | lost draft or session | the CLI keeps its own session files; the dashboard keeps the draft (ASM-23); Ctrl+R resume; job object, so no orphan |
| Mixed-state files change by themselves | the configuration check cannot classify the change | reported "changed, not classified", never PASS (section [Changes to plan 2's design](#changes-to-plan-2s-design)) |
| Terminal differences (Shift+Enter, OSC 52) | a key or copy path missing | Ctrl+J and native clipboard work everywhere; the matrix lists the rest |
| Quota sources are partial (Claude Code on transitions only, **U** windows) | stale or missing values | shown with age and †, "N/A" past reset (TEL) |

Compatibility: native Windows 11 in Windows Terminal (amendments 6 and 7),
Claude Code 2.1.300 and Codex 0.145.0 as probed; other versions run the
fixture tests first. Linux, macOS and CLIs inside WSL driven from Windows are
not verified ([architecture.md, Where it runs](architecture.md#where-it-runs)).

## 7. Revised terminal UI layout

Amendment 2, deliverable 1. Detail: [ui-design.md, Dashboard](ui-design.md#dashboard)
and [Layouts at three sizes](ui-design.md#layouts-at-three-sizes).

What changed from the brief's first, observe-only dashboard: zone B became a
conversation with an editor and approval cards (AM2); the editor and
selection became essential (AM3); a startup screen comes first and a project
bar names the CLI and directory (AM4). The narrow layout keeps every function
at reduced size:

```text
 >_ AGENT COMMAND CENTER                                                 ● LIVE 
 ▣ agent-command-center  branch main                                Claude Code 
 A ✓ Turn 1 › ◉ Turn 2: scan-routes ◉ write-tests !   (inferred)   F6: tree     
╭─ B · CONVERSATION ───────────────────────────────────────────── ◉ Streaming ─╮
│ YOU  Refactor the auth module and add tests for the token refresh.           │
│ CLAUDE  I'll read the current module first, then split the refresh           │
│   ▸ Read src/auth/middleware.go                                     observed │
│ ! APPROVAL write-tests: Edit refresh_test.go +12  a allow  d deny            │
│ > Also check the logout path█                                                │
╰──────────────────────────────────────────────────────────────────────────────╯
╭─ C · AGENTS ────────────────────────────────────────────────────── 2 ACTIVE ─╮
│ ◉ scan-routes  sonnet-5-5  Bash go test ./auth           41s   12.1k         │
│ ! write-tests  sonnet-5-5  waiting: approval             18s   6.4k          │
╰──────────────────────────────────────────────────────────────────────────────╯
╭─ D ──────────────────────────────────────────────────────────────────────────╮
│ 5h ▰▰▰▱▱▱▱▱▱▱ 23%† 7d ▰▰▰▰▰▱▱▱▱▱ 45%† ctx 12%  tok 41.2k  $0.42 est.  err 0  │
╰──────────────────────────────────────────────────────────────────────────────╯
```

(Shortened from the 80x24 mock-up; the standard and wide layouts put A on the
left and C beside B.)

## 8. Interaction-flow diagram

Amendment 2, deliverable 2. Detail: [ui-design.md, Interaction flow](ui-design.md#interaction-flow)
and the state machines in [events-and-state.md](events-and-state.md#state-machines).

```text
 launch ─► detect CLIs ─► STARTUP: choose CLI, enter directory ─► validate
                                     ▲                         ✗ │ ✓
                                     └── message under field ◄───┘ │
                                                                   ▼
                    DASHBOARD starting ─► handshake ✗ ─► StartFailed card
                                              │ ✓
              ┌────────────────────────► READY: draft in B
              │                               │ Enter
              │                               ▼
              │          turn running: B streams · A stages · C agents · D telemetry
              │             ├─► approval card ─► allow / session / deny / cancel
              │             ├─► Ctrl+C ─► interrupting ─► interrupted
              │             └─► CLI exits ─► Crashed card ─► Ctrl+R resume
              │                               │
              └──── follow-up, same session ◄─┘ turn completed / failed
 Ctrl+Q anywhere: confirm if work runs, close the CLI, restore the terminal
```

## 9. Event and state architecture

Amendment 2, deliverable 3. Detail: [events-and-state.md](events-and-state.md).

- **Pipeline**: CLI line ─► decode ─► adapter `map(line)` (pure) ─► event
  with an envelope (`Seq`, `At`, `Provider`, `Session`, `Kind`, `Origin`,
  `Agent`, `Turn`, `Item`, `Payload`, `Raw`) ─► queue ─► reducer ─► state
  ─► view ([Pipeline](events-and-state.md#pipeline)).
- **Origins**: `ai`, `observed`, `derived`, `inferred`, `mock`, each shown
  differently ([Origins](events-and-state.md#origins)).
- **Conceptual events** of amendment 2, each with a source per CLI and its
  evidence, or "unavailable" with a fallback
  ([Conceptual events](events-and-state.md#conceptual-events)).
- **State**: the conversation record, the agent tree, telemetry with origins
  ([State model](events-and-state.md#state-model)).
- **Six state machines**: startup, session, turn, agent, approval, tool call,
  each with its invalid transitions and a column per CLI
  ([State machines](events-and-state.md#state-machines)).
- **Concurrency**: one reader per stream, one queue, one reducer; subagent
  output routed by agent key, never interleaved in one message
  ([Concurrent output](events-and-state.md#concurrent-output)).

## 10. Integration strategy with Claude Code

Amendment 2, deliverable 4. Detail: [architecture.md, Claude Code adapter](architecture.md#claude-code-adapter)
and [Integration strategy](architecture.md#integration-strategy);
[research/claude-code.md](research/claude-code.md), section 4.

- One long-lived `claude -p --input-format stream-json --output-format
  stream-json --verbose --include-partial-messages --permission-prompt-tool
  stdio` in the project directory (way A, CC-4), as the Agent SDKs and the
  VS Code extension drive it. Proven from a client without the SDK: three
  turns, deltas, an approval, an interrupt, a context request (P2), a
  subagent's approval (P3b), resume (P4).
- Control requests: `initialize`, `interrupt`, `get_context_usage`;
  `can_use_tool` answered from the approval cards. Not sent: `set_model`,
  `set_permission_mode` (they might write settings, CC-10 q11).
- `--permission-mode default` (ASM-4): the CLI's own rules decide what asks;
  the dashboard never widens them.
- Codex follows the same shape through `codex app-server`
  ([architecture.md, Codex adapter](architecture.md#codex-adapter)).

## 11. Reusable components

Amendment 2, deliverable 5. Detail: the "worth studying" sections of
[agent-dashboard.md](research/agent-dashboard.md) (section 13),
[claude-dashboard.md](research/claude-dashboard.md) (section 13),
[ccdash.md](research/ccdash.md) (section 13) and
[jev-pilot.md](research/jev-pilot.md) (section 14).

From the four reference repositories, **no code is copied**: each would bring
tmux, scraping or configuration writes, and what is useful is small enough to
write from its idea. Ideas taken, each recorded where used:
- the subagent tree and quota bar presentation (agent-dashboard);
- the Bubble Tea and Lip Gloss structure (claude-dashboard);
- usage parsing from transcripts, kept as an undocumented fallback only
  (ccdash);
- JEV detection through the loaded-plugin list (jev-pilot).

Components reused from elsewhere are in section 5.

## 12. Technical limitations and risks

Amendment 2, deliverable 6, and amendment 4 section 8 (the limits each CLI
puts on the interaction). Detail: [research/integration.md](research/integration.md)
(the "Limitations" of each question), [reliability.md, Not verified](reliability.md#not-verified).

**Claude Code**
- The control protocol's wire form and `--permission-prompt-tool stdio` are
  undocumented (**U**; INT 1, INT 4).
- `--resume` does not replay earlier messages (P4): history after a resume is
  the dashboard's own record.
- Background subagents end with the process (P3); a resume cannot bring them
  back, nor a running turn or pending approvals.
- Subagents do not stream tokens by default; `--forward-subagent-text`'s
  volume with two parallel subagents is not verified (CC-10 q6).
- Quota arrives in `rate_limit_event` only when a window changes state, and
  its `unifiedWindows` is undocumented (TEL CC).
- Sending while a turn runs is not supported in the MVP (ASM-15); interrupt
  during a running Bash command is probed in the step 2 gate (CC-10 q4).
- The concurrency limit and the models subagents use are reported by
  nothing: "N/A" (TEL).

**Codex CLI**
- The app-server is documented but labelled experimental (INT 1).
- Sandbox `read-only` with approvals `on-request` (ASM-6): every write asks,
  because `workspace-write` may write `trust_level` into `config.toml`
  (CX-10 q2).
- The model in the owner's `config.toml` was refused for a ChatGPT account
  (P5, P6); the session then uses the default from `model/list`.
- App-server threads are recorded with `source: "vscode"` among the owner's
  sessions (INT 5).
- No cost is reported ("N/A"); a child thread's model is "N/A" (ASM-14).
- Not verified until the step 5 gate: resume across processes (CX-10 q4),
  `decline` and `cancel` outcomes (CX-10 q1); background terminals are not
  stopped by `turn/interrupt` ([T] README).

**Both, on this machine**
- No POSIX signals on Windows: closing is stdin, then terminate, then kill
  (INT 9); the job object prevents orphans.
- Shift+Enter needs the terminal to report it; Ctrl+J always works (TI 4).
- CLIs inside WSL driven from the Windows program are outside the MVP.

## 13. Updated MVP implementation plan

Amendment 2, deliverable 7; plan 4 in [plan.md](../plan.md). The steps are in
section 4; what changed from the plan 4 outline written before plan 2:

- **The owner's order of 2026-10-16**: Codex (priority 5) now comes right
  after Claude Code's permissions and continuity, before agent activity (6)
  and telemetry (7); the earlier outline finished zones A to D with Claude
  Code first. Zones A, C and D are therefore built once, for both CLIs.
- **An integration gate per CLI** (steps 2 and 5), live and approved, before
  the screens that depend on it.
- **Crash, cancel and recovery** checks with byte compares of the draft and
  the record (step 4), and the draft kept across the dashboard's own failure
  (ASM-23).
- **Targets stay targets**: 16 ms and 100 ms are measured in step 3 and
  reported as measured.
- **Milestones**: each step ends with an evaluation summary in `plan.md`;
  routine fixes go ahead; a FAIL that needs a decision, a new live probe or
  a new dependency waits for the owner.

## What amendments 2 to 4 changed in the architecture

- **Amendment 2, interactive conversation.** The dashboard went from
  observing to conversing: one long-lived CLI session per dashboard over a
  machine interface, an approval relay, a conversation record, origins on
  every value, and event-driven state with state machines
  ([architecture.md](architecture.md); [events-and-state.md](events-and-state.md)).
- **Amendment 3, input and clipboard.** The editor became our own component
  (the Bubbles textarea's limits), Bubble Tea v2 became required (paste on
  Windows), mouse capture with in-app selection and two copy paths were
  added, and input became provider-independent by construction
  ([research/terminal-input.md](research/terminal-input.md), section 11;
  [ui-design.md, Input editor](ui-design.md#input-editor)).
- **Amendment 4, project-aware startup.** A startup machine before the
  session, `Detect` in the provider interface, path validation, the project
  directory as the CLI's only working directory, a fixed MVP scope, and the
  two images as the visual specification
  ([ui-design.md, Startup screen](ui-design.md#startup-screen)).
- **Amendment 5** (for completeness): Gemini CLI and Google Antigravity left
  the scope; no adapter, option or test; the interface stays open.

## Changes to plan 2's design

Made in this plan, at the owner's instructions of 2026-10-16, or proposed
here for approval:

1. **Configuration check, opaque hashing only** (made; the owner's
   instruction). The streaming tokenizer over `~/.claude.json` and
   `known_marketplaces.json` is withdrawn: no file is parsed. A changed
   mixed-state file is reported "changed, not classified (contents not
   inspected)", the run's configuration criterion is not PASS, and the file
   is listed under Not verified; with no CLI running, a change is a FAIL.
   Limitation: a setting changed inside those two files while the CLI also
   rewrote its state cannot be told apart by any permitted check
   ([reliability.md, Configuration untouched](reliability.md#configuration-untouched)).
2. **Draft kept across the dashboard's own failure** (proposed, ASM-23):
   the editor's text saved, at most 1 s after the last change and by atomic
   rename, to `%LOCALAPPDATA%\agent-command-center\drafts\`, one file per
   project named by a hash of its path; deleted once the CLI accepted the
   send; offered back in the editor on the next start in that project. It is
   the owner's own text, in plain text in their profile; it is not a history
   file (ASM-11 unchanged).
3. **Exit and panic** (proposed): on quit, crash and panic, the dashboard
   prints the CLI session id and the CLI's own resume command, so the
   session stays reachable from a terminal.
4. **Codex types by hand** (proposed): the subset the MVP uses, checked
   against the schema by fixture tests, instead of generating them (one tool
   fewer).
5. **Performance figures are targets** (made): ASM-10's 16 ms and 100 ms are
   reported only as measured.

## Approvals requested

Nothing below is done before the owner approves it. After approval, re-runs
of an approved live test after a fix go ahead and are reported afterwards;
any other new probe is asked for first.

1. **This proposal**, including the roadmap order and the changes above
   (items 2 to 4 of [Changes to plan 2's design](#changes-to-plan-2s-design)).
2. **Go 1.27.2**, `go1.27.2.windows-amd64.zip` from `https://go.dev/dl/`,
   SHA-256 `1314008898bd40df77af4b014f777f08873dbdfbcd3d92308728ee03304fe04f`
   (read from `https://go.dev/dl/?mode=json` on 2026-10-16), extracted to
   `%LOCALAPPDATA%\Programs\go` and added to the user `PATH`: no
   administrator rights, removed by deleting the folder. License
   BSD-3-Clause.
3. **Dependencies**, pinned in `go.mod` (versions as published on
   2026-10-16; plan 2 studied Bubble Tea at `f9df43c` and Lip Gloss at
   `19f011f`, and the paste and console paths are re-read at the pinned
   version, as [reliability.md, Not verified](reliability.md#not-verified)
   requires):

   | Module | Version | License |
   | - | - | - |
   | `charm.land/bubbletea/v2` | v2.1.0 | MIT |
   | `charm.land/lipgloss/v2` | v2.0.6 | MIT |
   | `golang.design/x/clipboard` | v0.11.0 | MIT |
   | `golang.org/x/sys` | v0.49.0 | BSD-3-Clause |

   Their transitive modules are listed in `THIRD_PARTY.md` at step 0; any
   license other than MIT, BSD or Apache-2.0 stops for approval. Glamour
   v2.0.1 and Chroma v2.27.0 (MIT) are asked for at step 3 only if the
   benchmark keeps them.
4. **Live runs**, each in a temporary project outside the repository, with
   the configuration check before and after:

   | Run | Step | Quota, estimated from the probes |
   | - | - | - |
   | L1 Claude Code integration gate | 2 | about 6 short Haiku turns, P2 and P4's scale (P1 to P4 and P3b together: about USD 0.03 API-equivalent) |
   | L2 Claude Code smoke test through the dashboard | 4 | 2 short turns on the configured default model |
   | L3 Codex integration gate | 5 | about 4 short turns, P6b's scale |
   | L4 Codex smoke test through the dashboard | 5 | 2 short turns |

5. **The configuration-check decision**: opaque hashing, with its stated
   limitation, as the standing check for plan 4.

## Assumptions to confirm

The assumptions of [architecture.md, Assumptions](architecture.md#assumptions)
(ASM-1 to ASM-23) stand unless the owner changes one; those the roadmap
measures are ASM-10 (step 3), ASM-17 (step 3), ASM-19 (step 2), ASM-20
(step 2), ASM-21 (steps 1, 2, 5) and ASM-23 (step 4).
