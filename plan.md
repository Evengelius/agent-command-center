# Current Plan

Four plans from `doc/brief.md` and its four amendments, run one at a time and in
order. Each ends with the evaluation summary of `doc/evaluation.md` and waits
for the owner's approval before the next starts. Until plan 3 (the proposal) is
approved: no project code, no dependency installed, no file written outside
`doc/` and this plan; the only exceptions are plan 1's probes, below, and the
development setup the owner asked for (`AGENTS.md`, `CLAUDE.md`,
`.claude/agents/`).

The application is an interactive workspace (amendment 2): a real conversation
with the orchestrator, beside its workflow tree, its live agents and their
telemetry. It drives Claude Code first, then Codex CLI, each optional; Gemini
CLI and Google Antigravity are out of scope (amendment 5), and the provider
interface stays open to another agent CLI later. It opens on a startup
screen (agent CLI and project directory) before the dashboard, and its MVP is
fixed by amendment 4: plan 4 below, whose steps plan 3 details.

The order puts the biggest risk first: whether each agent CLI can be driven
from another program decides the architecture, and whether a reference project
is worth forking; reliability is designed with the architecture, not after it.

Active: **Plan 4, the MVP, in progress (started 2026-10-16).** Plan 3
approved by the owner on 2026-10-16 ([`doc/proposal.md`](doc/proposal.md);
12 of 12 PASS on the main session's checks, no verifier run). Plan 2 approved
on 2026-10-16 (15 of 15 PASS; the independent verifier did not re-run after
the final fixes, [`doc/plan2-evaluation.md`](doc/plan2-evaluation.md)).
Plan 1 accepted on 2026-10-16, with criteria 8 and 9 kept as FAIL and accepted
as documented exceptions ([`doc/research/plan1-evaluation.md`](doc/research/plan1-evaluation.md)).

Plan 4 progress (update as each step finishes; read this first after a
context compaction). Steps and their checks: [proposal.md, section 4](doc/proposal.md#4-prioritised-mvp-roadmap).

- Step 0 (foundation): **done** (2026-10-16), 10 of 10 items PASS, checked
  by the main session (deterministic commands; no verifier run, no CLI
  started, no quota used):
  1. Go: `go version` gives go1.27.2 windows/amd64; zip SHA-256 matched go.dev.
  2. Skeleton: `go build ./...` and `go vet ./...` exit 0, `gofmt -l .` prints
     nothing; `go test ./...` 40 top-level tests (135 with subtests) PASS,
     0 FAIL, 0 SKIP. AGENTS.md Commands filled in.
  3. Dependencies: the four approved direct modules only; 34 modules in the
     graph, 19 built into the binary, each licence recorded in
     `THIRD_PARTY.md` and checked by a test (fallback to the module cache when
     `go list` gives no directory).
  4. Test 7 (`internal/arch`): the import graph and clock rules pass on the
     real tree; a planted-fault test asserts 5 of 5 faults found, and a manual
     run with two files planted in the real tree failed as expected, then
     passed once they were removed (no leftovers in `git status`).
  5. Fixtures: the 10 `testdata/probes` files have the same SHA-256 as
     `doc/research/probes`. They were re-sanitized during step 0 (a user path
     had leaked; it was never committed); the mock replay exempts the probe's
     cwd from path validation.
  6. Test 1 (`TestEveryLineMaps`, Claude and Codex): every fixture line maps
     to at least one event, 0 dropped; P2 gives 25 deltas, 3 `text_delta`;
     P7 gives 11 rate-limit pushes and 10 `thread/status/changed`.
  7. Tests 3 and 4: `TestStateMachines` has one case per transition and per
     invalid transition and fails if its case count differs from the rows of
     `doc/events-and-state.md`; `TestReplayDeterministic` replays each
     fixture twice to the same state. Found on the way: a CLI-initiated turn
     after a background agent's notification, now a row of that document
     ("turn started by the CLI").
  8. Test 8 (mock mode): both mock files replay (154 and 176 events), paste
     and send, four resizes and Ctrl+Q; every frame carries the mock badge and
     every event the mock origin. Fixes: the first frame lacked the badge
     (launch now applied before the first frame), mock Close made atomic, Back
     and Resume refused in mock mode.
  9. Test 9 (golden frames): startup and dashboard at 80x24, 120x40, 200x60
     and 79x24; line count = height, every line within the width, the
     too-small notice only below 80x24. Found and fixed a box top edge one
     cell too wide (corner replaced by "…"), with `TestBoxEdges` (w 5 to 40).
  10. `confcheck`: hash-only snapshot of the CLI configuration and session
     files (24 entries, 158 session files; digests only, never contents;
     credential files hashed opaquely). Exit codes: 0 PASS, 1 FAIL,
     2 NOT-VERIFIED, 3 usage or error. The step 0 baseline is kept outside
     the repository, as L1's "before" reference.
  Not verified at step 0: anything needing a live CLI (L1 to L4), clipboard
  and terminal behaviour outside tests, performance (targets only).
- Step 1 (startup and project selection): **done** (2026-10-16), 6 of 6
  items PASS (item 6 by the owner). `go build`, `go vet`, `gofmt -l .` clean;
  `go test ./... -count=1`: 160 PASS, 0 FAIL, 0 SKIP; test 7 passes with the
  new packages.
  1. Paths: `TestResolvePath`, 44 cases (Windows and Linux columns: DOS, UNC,
     WSL `/mnt/<x>` and `\\wsl.localhost`, device paths, `~`, relative,
     drive- and root-relative, trailing periods), each with its resolved path
     or ui-design.md's exact message; no I/O. `TestCheckProject` on disk:
     missing, file, `~`, relative, refused form, WSL translation not found
     (with the mount-root hint), and the branch from `.git/HEAD` (branch,
     inside a work tree, linked worktree with a detached head);
     `TestParseHead`. `TestCheckProjectNotReadable`: a folder with an icacls
     deny on "list folder" for the current user gives "Not readable
     (permission denied): <path>"; a planted fault skipping the read check
     makes it fail (`os.Stat` succeeds on such a folder; `os.Open` refuses it).
  2. Detection with a fake PATH (`TestDetect`, 7 cases): neither, either,
     both, the installers' folders off the PATH, npm shims reported and never
     run, version check failing (exit 1 with stderr; unparseable output);
     `TestDetectVersionTimeout`: a hanging CLI reported after 2 s. Through the
     application (`TestStartupGate`): with neither CLI Start does nothing and
     the screen shows "No supported agent CLI found…"; a hanging detection
     shows "detection timed out after 0.2 s" with a 200 ms bound (10 s by
     default) and the other CLI stays selectable.
  3. Detection on this machine, version commands only (`claude -v`,
     `codex --version`): Claude Code 2.1.300 at
     `C:\Users\…\.local\bin\claude.exe`, OpenAI Codex 0.145.0 at
     `C:\Users\…\AppData\Local\Programs\OpenAI\Codex\bin\codex.exe`, about
     0.1 s each. The sign-in kind is not read yet.
  4. Gate: the invalid "not installed" case of `TestStateMachines` now also
     covers neither CLI installed; `TestStartupGate` checks a missing project
     refused with its message, then a valid one accepted with its branch,
     and that `--provider` and `--project` only prefill (a `--provider` not
     installed falls back to the first installed CLI).
  5. Golden frames (`TestStartupMessageFrames`, 80x24): no CLI, does not
     exist (user folder elided), WSL path not found, live Linux path, live
     drive-relative path, `\\wsl.localhost` warning; each message whole
     (wrapped, never cut), 24 lines, none wider than 80. Found and fixed:
     long messages and CLI reasons were cut at 80 columns (now wrapped), and
     the no-CLI screen at 80x24 cut the Start button (blank spacing lines now
     go first).
  6. The owner launched `go run ./cmd/agent-command-center` and the mock
     replay from the repository root on 2026-10-16 and approved both screens
     ("This is really good, I like it, you can continue"). Start then reports
     that the session is not built yet (step 2).
- Step 2 (real Claude Code integration and the gate): **done** (2026-10-16),
  items 1 to 4 PASS, item 5 (L1) (a) to (g) PASS, (h) NOT-VERIFIED (the
  approved hash-only limit). `go build`, `go vet` (also `-tags live`),
  `gofmt -l .` clean; `go test ./... -count=1`: 170 PASS, 0 FAIL, 0 SKIP;
  test 7 passes. No race detector: cgo is off, so `-race` cannot run.
  1. Adapter (`internal/provider/claude/session.go`): way A over plain pipes,
     the project as working directory, `initialize` on start; events mapped
     in order, with no gap in Seq: a write's own events are held until it
     succeeds and dropped if it fails. Fake-CLI tests (the test binary
     replaying P2, test 10): `TestHandshakeAndTurn` (argv, cwd of
     `system/init` = project, approval allowed, every line written compared
     byte for byte), `TestNoHandshake` (crashed after the shortened
     timeout, "no handshake within…"), `TestExitMidTurn` ("the CLI exited
     unexpectedly"), `TestSendClosedStdin` ("the message could not be sent:
     …", draft restored), `TestLongLine` (65 MiB line skipped with its
     error, the next line read).
  2. `TestCloseEscalation`: a fake CLI that ignores stdin and starts a
     grandchild is gone, with it, in 5.0 s (bound 11 s); session Closed, no
     error. `TestForcedKill`: the context's cancel kills the tree within
     1 s.
  3. `TestJobKillsTreeWithDashboard` (`internal/proc`): the "dashboard"
     test process killed, its CLI gone 11 ms later. Planted faults: without
     KILL_ON_JOB_CLOSE it fails; without the job closed in Wait it fails
     ("alive: CLI false, grandchild true"). The second fault revealed a
     hang, fixed: on Windows closing a pipe does not wake a blocked read,
     so after the exit the session waits 2 s for its readers, then ends
     with the error "Claude Code's output stayed open after its exit…".
  4. Working directory: fixture in 1; live in 5 (cwd = the temporary
     project).
  5. L1, `go test -tags live -run TestLiveGate ./internal/provider/claude`
     (`live_test.go`), Claude Code 2.1.300, `--model haiku`
     (claude-haiku-5-5), a new empty folder under the session scratchpad,
     outside the repository; event log kept there, not in the repository.
     (a) 9 text deltas before `result`. (b) allow wrote `allow.txt` ("ok");
     deny left `deny.txt` absent, turn completed. (d) interrupt with a Write
     approval pending: `error_during_execution` 66 ms later, `pending.txt`
     absent. (c) first run **FAIL**: the prompt's standalone `sleep 30` was
     refused by Claude Code before running ("Blocked: standalone sleep 30"),
     so no interrupt was sent; the test's fault, not the adapter's. After the
     fix (a `ping -n 30 127.0.0.1`), re-run alone (`TestLiveBashInterrupt`,
     new empty folder, configuration check around it): the Bash approval
     allowed, the command running, the turn ended `error_during_execution`
     3.3 s after the interrupt, the next turn answered; no PING.EXE left
     running. **PASS** on the re-run. (e) stdin closed: exit code 0 in
     1.4 s, before the 11 s step. (f) `--resume` in a new process: same
     session id, answered the code word. (g) a second process killed by its
     job mid-turn (ended 41 ms after, exit code 1), a third with `--resume`
     answered the code word. (h) `confcheck compare -cli-ran`, both runs:
     the 22 configuration files identical, `~\.claude.json` "changed, not
     classified" (Claude Code writes it; this session's Claude Code and
     Codex were also running), 1 new session file each: **NOT-VERIFIED**
     (exit 2), the limit approved with hash-only checking.
     Usage: 10 short turns (8 in the gate, 2 in the re-run; 4 ended early),
     against "about six"; `total_cost_usd` is a running total per process:
     USD 0.0096 + 0.0101 + 0.0102 + 0.0029 = about 0.033 API-equivalent,
     plus the killed turn, which reports none.
  Findings for later steps: Claude Code emits `system/commands_changed`
  (listing the owner's skills and commands), now shown as unrecognised;
  a `get_context_usage` failure is not reported (step 7); Ctrl+Q during a
  turn (step 3); `CmdDraftAccepted` deletes the draft at step 4.
- Steps 3 to 8: outstanding.

The owner's decisions at plan 3's approval (2026-10-16):
- Proposal and roadmap approved; follow its order and the MVP scope.
- Go 1.27.2 approved: the official Windows zip under the user profile, no
  administrator rights, no system-wide change; SHA-256 checked against go.dev
  before use.
- The four pinned dependencies approved (Bubble Tea v2.1.0, Lip Gloss v2.0.6,
  x/clipboard v0.11.0, x/sys v0.49.0); integrity, licenses and compatibility
  verified; no other direct dependency without a technical justification.
- Live run L1 approved (about six short Haiku turns, controlled temporary
  project). **L2, L3 and L4 need approval first**, each proposed with its
  exact scope, model, expected usage and stopping conditions; independent
  work goes on meanwhile.
- Hash-only configuration check approved; credential contents never read; a
  file that cannot be classified safely is "changed, not classified", never
  reported unchanged.
- Drafts: a user-restricted local directory, atomic writes, kept across
  failures, deleted only once the message was reliably accepted; no
  unnecessary conversation data stored, no sensitive content in logs.
- Verify crash recovery, session resumption and process termination;
  performance figures stay targets until measured; documented design is never
  presented as verified runtime behaviour.

Plan 3 progress (closed):

- Done: plan 2's evaluation recorded; the configuration check revised to
  opaque whole-file hashing; ASM-23 added; `doc/proposal.md` written; final
  check 12 of 12 PASS. Approved 2026-10-16.

The owner's instructions for plans 3 and 4 (2026-10-16), kept in the proposal:
- MVP order: 1 startup and project selection; 2 real Claude Code
  integration; 3 streamed interactive conversation; 4 permissions and session
  continuity; 5 Codex integration; 6 real agent activity; 7 usage and quota
  telemetry; 8 UI refinement to the two reference images. Each milestone
  verified before the next; no scope beyond amendment 4.
- Never inspect, parse, expose or modify credential values; opaque file
  hashing only, its limits documented.
- Performance figures are targets until benchmarks measure them.
- Validate the no-PTY integration early; if it falls short, document the
  limit and propose the smallest adjustment, never change the architecture
  silently.
- Drafts and session data survive cancellation, crashes and recovery.
- Deterministic tests, recorded fixtures and mock data first; live runs only
  when needed, approved where evaluation.md requires it; tests never weakened.
- The compaction settings stay as they are; the blocked
  `.claude/settings.json` write is not retried.
- Claude Code and Codex only; the provider interface stays open.
- Routine work goes ahead without approval; stop only for destructive or
  risky operations, required approvals, unresolved architecture or a
  blocker.

Plan 2 progress (closed):

- Done: criteria written; plan 1's evaluation recorded word for word;
  `doc/architecture.md`, `doc/events-and-state.md`, `doc/ui-design.md`,
  `doc/reliability.md`; deterministic checks; an independent verifier's
  pass (criterion 14 failing, gaps under 4, 7, 8, 13), all fixed and
  re-checked by the main session only. Result: 15 of 15 PASS.
- Approved 2026-10-16. The same day the owner withdrew the mixed-state
  tokenizer of criterion 13 (no parser over files that may hold credentials).

Plan 1 progress (closed):

- Done: step 1, `claude-code.md`, `codex-cli.md` (10 sections each, updated
  from the probe results); step 3, probes P0 to P7 recorded in
  `doc/research/probes/` (README lists them; P7, Codex subagents, on
  2026-10-16 under the re-run rule); step 6, the four reference project files
  (all: not a fork candidate); step 5, `terminal-input.md`; step 2,
  `integration.md` (18 answers and the path forms); step 4,
  `telemetry-sources.md`; step 7, `summary.md` (build from scratch; Claude
  Code first).
- Final check, 2026-10-16: 9 PASS, 2 FAIL on the strict reading, both
  past events that no further work changes: criterion 8 (P3b ran before the
  owner's re-run rule) and criterion 9 (2 of 22 hashes differed on
  2026-10-15, from Claude Code's own state writes; P7's were identical).
  Fixed during the check: a user name in `claude-code.md`, two resume
  claims cited to P6b, two missing "Limitations" lines in `integration.md`.
- Accepted 2026-10-16, with the 2 FAILs as documented exceptions.
- Decided on 2026-10-15, at the owner's request: development setup in
  `CLAUDE.md` and `.claude/agents/` (delegation rule in `AGENTS.md`); Gemini
  CLI and Google Antigravity removed from scope (amendment 5): `GEMINI.md`
  deleted, `gemini-cli.md` kept only as an out-of-scope research record, no
  longer a plan 1 deliverable.

## Plan 1 - Research

Biggest unknowns first: the agent CLIs and how to drive them, then the
telemetry and terminal input, then the reference projects, then the decision.
Do not assume the reference video, the layout mock-up or the screen images show
a working application. Never read credential files.

1. **The agent CLIs**, one file each: `claude-code.md` (including the workflows
   and statusline pages of the brief), `codex-cli.md`, each with: its official documentation (pages and date); every supported way to
   drive a session from another program (SDK, headless or streaming modes,
   servers, protocols); how it runs subagents or parallel work; where a program
   can observe it (stream events, hooks, session files, status line, telemetry
   export); how to detect it is installed (and its version), start it in a
   given working directory, and its native permission and sandbox modes; each
   marked documented or undocumented.
2. **`integration.md`**: the 9 research questions of amendment 2, answered for
   each agent CLI with sources: embedding or controlling a session; genuine
   streaming; subagent lifecycle and tool events; approvals and permissions;
   persistence, cancellation and resumption; reusing an existing terminal UI;
   concurrent subagent output; reliable quota; limits on Windows, WSL2, Linux
   and macOS. Each answer with its limitations and a realistic fallback. Also
   the path forms the startup screen accepts and validates (Windows, `~`, WSL
   `/mnt/c/...` and `\\wsl$\...`, Linux, macOS).
3. **Probes**, to settle what documentation alone cannot: throwaway runs of the
   installed CLIs (Claude Code, Codex CLI) in streaming mode with a one-line
   prompt, to record the real events they emit, in a temporary folder outside
   the repository, without changing any configuration. They use some of the
   owner's quota: list the probes and ask the owner's approval before running
   them. Recorded outputs go to `doc/research/probes/`, credentials and
   personal data removed; they become test fixtures later.
4. **`telemetry-sources.md`**: for each agent CLI, every way to get zone D's
   values (5-hour and 7-day quota or the CLI's equivalent, reset times, model,
   context, tokens, API-equivalent cost, agent count and concurrency limit,
   duration, errors and retries), documented or not, with its source; what is
   simply unavailable (shown as "N/A").
5. **`terminal-input.md`** (amendment 3): what each candidate UI library offers
   for a multiline editor (selection, cursor movement, very long input,
   bracketed paste); clipboard access per terminal and OS (Windows Terminal,
   PowerShell, WSL2, Linux, macOS; OSC 52 and native clipboards); mouse capture
   versus the terminal's own text selection; tmux copy mode; `$EDITOR`; the
   shortcuts each OS and terminal already takes. Each with its source and its
   limits.
6. **The reference projects**, one file each: `agent-dashboard.md`,
   `claude-dashboard.md`, `ccdash.md`, `jev-pilot.md`, each with: source and
   commit, license, stack and dependencies, architecture, data sources (and
   whether each is documented), which agent CLIs it supports, whether it hosts
   a conversation or only observes, whether its integration matches what steps
   1-3 found to work, what is reusable and under which license terms (the
   project is proprietary: permissive licenses only).
7. **`summary.md`**: fork versus build from scratch, with the reasons; which
   agent CLI to build first (Claude Code unless steps 1-3 show it cannot be
   driven properly).

Checks: the 10 files exist with their sections; every claim has a source; every
license is an SPDX id read at a recorded commit; every reference, every agent
CLI, all 9 questions and every item of amendment 3 are covered; probes run only
after the owner's approval, outside the repository, with their outputs
recorded and cleaned; no credential file read; no CLI configuration changed;
nothing outside `doc/` changed (`git status`).

Exact criteria, checked at the end of plan 1:

1. 10 files in `doc/research/`: `claude-code.md`, `codex-cli.md`,
   `integration.md`, `telemetry-sources.md`,
   `terminal-input.md`, `agent-dashboard.md`, `claude-dashboard.md`,
   `ccdash.md`, `jev-pilot.md`, `summary.md`; each with the sections its step
   lists.
2. Each agent CLI file covers: documentation pages with access date; every
   way to drive a session; subagents and parallel work; observation points;
   detection and version; start in a working directory; permission and
   sandbox modes; each item marked documented or undocumented.
3. `integration.md` answers the 9 questions of amendment 2 for each of the 2
   CLIs (18 answers), each with sources, limitations and a fallback, plus the
   path forms (Windows, `~`, `/mnt/c/...`, `\\wsl$\...`, Linux, macOS).
4. `telemetry-sources.md` gives, for each CLI, a source or "N/A" for every
   zone D value: 5-hour and 7-day quota, reset times, model, context, tokens,
   API-equivalent cost, agent count, concurrency limit, duration, errors,
   retries.
5. `terminal-input.md` covers every item of amendment 3: multiline editor,
   selection, cursor movement, long input, bracketed paste, Enter and newline
   keys, `$EDITOR`, clipboard per terminal and OS (Windows Terminal,
   PowerShell, WSL2, Linux, macOS; OSC 52), mouse capture versus terminal
   selection, tmux copy mode, OS and terminal shortcuts.
6. Each reference project file records the full commit SHA inspected and its
   date, and the SPDX license id read from its license file at that commit
   (or "no license", meaning not reusable).
7. Every claim names its source: a URL with access date, or repository,
   commit, file and line.
8. Probes run only after the owner's approval, in a folder outside the
   repository; each recorded in `doc/research/probes/` with its command, CLI
   version and raw output, credentials and personal data removed.
9. No credential file opened; the hashes of the CLI configuration files
   (credential files excluded) identical before and after the probes.
10. `git status` lists only files under `doc/` and `plan.md`, plus the
    development setup the owner asked for on 2026-10-15: `AGENTS.md`,
    `CLAUDE.md`, `.claude/agents/`.
11. `summary.md` gives fork or build with the reasons, and the first agent
    CLI.

## Plan 2 - Architecture and reliability

Design the application from the research, with its reliability in the same
pass: fallbacks, cancellation, mock mode and tests shape the event flow and the
state, so they are decided together. Zones A to D of amendment 2 (A:
orchestrator and workflow tree; B: interactive conversation, multiline input,
streaming, Markdown and code blocks, tool activity, clarification and approval
requests, history, follow-ups, interruption; C: live subagents, selectable and
expandable, independent from the conversation; D: usage and telemetry), the
interaction flow of amendment 2 (launch, ask, orchestrate, watch, inspect,
approve, report, continue), the two screens of amendment 4, keyboard navigation
and focus, responsive layouts, and the style of `doc/reference-design.md`.
Bubble Tea, Lip Gloss and tmux unless plan 1 found better; each CLI's own
orchestration reused, not rebuilt.

- `doc/architecture.md`: stack with versions; the five separated parts of
  amendment 2 (CLI execution and session management, event collection and
  normalisation, application state, terminal rendering, input and commands);
  one adapter per agent CLI behind one provider interface (detect, start in a
  directory, converse, observe), with nothing hardcoded around Claude Code; the
  session context (provider, project name, path) shared by every part; the
  integration strategy with Claude Code; how it runs on Windows (tmux needs
  WSL); how it starts.
- `doc/events-and-state.md`: the conceptual events of amendment 2
  (`conversation.message.delta`, `tool.started`, `agent.failed`,
  `approval.requested`, `telemetry.updated`...), each mapped to a real
  documented source per CLI (plan 1's probes as evidence), or marked
  unavailable; the state model; where polling cannot be avoided, and why.
- `doc/ui-design.md`: the two screens of amendment 4 and the transition between
  them: the startup screen (the two questions, validation messages, keys) and
  the dashboard with its project bar (provider, project, path); the revised
  layout at 3 terminal sizes (text mock-ups); the interaction-flow diagram;
  focus and every key binding; scrolling, collapsing, full-screen inspection;
  animations; colors and symbols mapped to the references; every state
  (loading, streaming, waiting for approval, cancelled, no data, error, CLI not
  installed); how AI messages, observed events and inferred labels look
  different; the input editor of amendment 3 (multiline editing and selection,
  paste, clipboard and mouse copy from every panel, configurable Enter-to-send
  and newline keys, `$EDITOR`), with what works where across terminals and OSes.
- `doc/reliability.md` (the brief's phase 3): the parts separated as above
  (rendering reads the state only); what happens when each agent CLI, each
  event source or each telemetry source is missing; approvals, interruption,
  cancellation and resumption without losing the session; concurrent output;
  how every CLI's configuration stays untouched; JEV optional; the mock-data
  mode (mock events through the same pipeline, never shown as real); the test
  plan (event normalisation and telemetry parsing per CLI, from plan 1's
  recorded probes; conversation and agent state transitions: each state and
  transition listed; input: pastes preserved byte for byte, a multiline paste
  never submitting, thousands of lines without freezing, the same behaviour for
  every agent CLI).

Checks: every item of the brief (phases 2 and 3) and of amendments 2 to 4
placed in a section; both screens of amendment 4 designed, and where they
follow or depart from the reference images; every choice traced to a plan 1
finding; every conceptual event mapped or marked unavailable; each source from
plan 1 has a fallback; the state machines list every state and transition, and
fit both CLIs; mock mode can never be mistaken for a live session;
mock-ups for 3 sizes; no rebuilt workflow engine or terminal protocol.

Exact criteria, checked at the end of plan 2:

1. 4 files in `doc/`: `architecture.md`, `events-and-state.md`,
   `ui-design.md`, `reliability.md`; each with the sections its bullet above
   lists.
2. A coverage table lists every item of the brief's phases 2 and 3 and of
   amendments 2 to 4, each with the section that places it: 0 items without
   a section.
3. Both screens of amendment 4 designed, each with where it follows and where
   it departs from the reference images.
4. Every design choice names its plan 1 finding (research file and section)
   or brief item; a choice with neither is listed as an assumption.
5. Every conceptual event of amendment 2 has, for each of the 2 CLIs, a
   source with its documentation or probe evidence, or "unavailable" with a
   fallback.
6. Every source in `telemetry-sources.md` and `integration.md` has a fallback
   in `reliability.md`.
7. Every state machine lists all its states and transitions, the invalid
   transitions and what they do, and states for each transition whether it
   holds for Claude Code and for Codex.
8. Mock mode: a separate provider through the same pipeline, labelled on
   every screen, with no path by which a mock event reaches a live session.
9. Text mock-ups of the dashboard at 3 terminal sizes, and of the startup
   screen.
10. The CLIs drive their own sessions and subagents: no workflow engine, no
    terminal protocol (PTY or screen scraping) in the design.
11. Gemini CLI and Google Antigravity appear only as out of scope; none of
    amendment 4's excluded features appears in the design.
12. The large-input keystroke time budget is set (a number in milliseconds,
    for a 10,000-line buffer).
13. The configuration-untouched check separates volatile runtime state from
    meaningful configuration changes, without reading credential contents
    (the owner's instruction of 2026-10-16).
14. Every limitation still open from plan 1 is listed as "Not verified" with
    the probe or test that would settle it.
15. `git status` shows changes only under `doc/` and to `plan.md`, beyond the
    files plan 1 left; the new files are UTF-8 without BOM, LF only.

## Plan 3 - Proposal

- `doc/proposal.md`: the 6 items of the brief and the 7 deliverables of
  amendment 2 (revised layout, interaction-flow diagram, event and state
  architecture, integration strategy, reusable components, limitations and
  risks, updated MVP plan), summarised from plans 1 and 2 with links,
  explaining what amendments 2 to 4 changed in the architecture, and the
  technical limits each CLI puts on the interaction (amendment 4). The MVP
  roadmap fills plan 4 below: step 0 and each priority as ordered steps with
  their own checks, the input editor and clipboard as essential (amendment 3),
  the first agent CLI end to end before the others.

Checks: every item present; each links to its detailed document; the roadmap
in ordered steps, each with its own checks. Then present it and wait for the
owner's approval.

Exact criteria, checked at the end of plan 3:

1. `doc/proposal.md` has the 6 items of the brief's phase 4 and the 7
   deliverables of amendment 2, each under its own heading: 13 of 13.
2. Each of the 13 links to the detailed document (research file or plan 2
   document) it summarises: 0 items without a link.
3. A section states what amendments 2, 3 and 4 changed in the architecture,
   one entry per amendment.
4. A section states the technical limits each of the 2 CLIs puts on the
   interaction (amendment 4 section 8), with the source of each.
5. The roadmap: step 0 and the owner's 8 priorities as ordered steps, each
   with yes/no checks that have exact expected values; Claude Code end to end
   (through permissions and continuity) before Codex; the input editor and
   clipboard inside the conversation step as essential.
6. The roadmap has an early integration-validation step (streaming,
   approvals, interrupt, resume over pipes, no PTY) before the UI work that
   depends on it, with the adjustment rule if it falls short.
7. The 16 ms and 100 ms figures appear only as targets, with the benchmark
   that measures them.
8. Crash, cancellation and recovery tests show drafts and session data
   survive (job object, Ctrl+R resume).
9. The configuration check uses opaque whole-file hashes only: no parser,
   tokenizer or key-level read of any configuration or state file, in
   `proposal.md` or `reliability.md`; its limitation is stated.
10. An "Approvals requested" section lists the proposal, the Go toolchain
    (version, source), each dependency (version, license) and each live
    smoke test (what it costs in quota).
11. Gemini CLI and Google Antigravity appear only as out of scope; none of
    amendment 4's excluded features is in the roadmap.
12. `git status` shows changes only under `doc/` and to `plan.md`
    beyond the files plans 1 and 2 left; new and changed files UTF-8 without
    BOM, LF only; links 0 bad; no personal names.

## Plan 4 - MVP (from amendment 4; steps detailed by plan 3)

The fundamental plan: a working application, before a beautiful one and long
before a bigger one. Starts only after plan 3 is approved; plan 3 turns step 0
and each priority into steps with their own checks. The first agent CLI
(Claude Code, unless plan 1 decided otherwise) end to end first (step 0 and
priorities 1 to 5), then the two others, validated the same way; never a
function traded for a visual detail.

0. Foundation: the toolchain installed with the owner's approval (Go is not on
   this machine); the project skeleton (module, folders, build, test commands
   in AGENTS.md); the event and state core with the mock-data mode; the test
   harness (table tests, golden screen snapshots, plan 1's recorded probes as
   fixtures); both screens and the four zones laid out on mock data, so the
   layout exists from the start.
1. Startup configuration: the two questions; provider detection with a clear
   error when the CLI is missing; path entry, expansion and validation (exists,
   is a directory, accessible), current directory prefilled.
2. The chosen provider starts in the chosen directory.
3. The conversation end to end: streaming, multiline input and long pastes,
   clipboard, permission handling.
4. Zones A and C show the real execution information available.
5. Zone D shows the real metrics available, "N/A" otherwise.
6. The visual presentation polished to match the reference images (the layout
   itself exists since step 0).

Out of scope (amendment 4): recent projects, favourites, project browser, git
worktrees, editor integration, model configuration, cross-provider
collaboration, switching provider during a session, session profiles,
orchestration settings; and (amendment 5) Gemini CLI and Google Antigravity.

Done when every acceptance criterion of amendment 4 passes, with each of the
two agent CLIs: launched from a terminal; either selectable;
project path validated; the CLI working in that directory; real prompts with
streamed answers; files inspected and changed with approvals; multiline input,
long pastes and copying reliable; real agent activity shown; telemetry accurate
or marked unavailable; both screens in the sequence and layout of the images.
