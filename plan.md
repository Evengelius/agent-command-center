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
- Step 3 (interactive conversation with streaming): **done** (started and
  finished 2026-10-16; criteria 1 to 8 PASS; criterion 9 measured, PASS
  except Shift+Enter in both terminals and three conhost input cells, FAIL
  and documented below). Criteria, each yes/no (proposal step 3):
  1. Test 5: for pastes with LF, CRLF, lone CR, tabs, trailing spaces,
     combining marks, emoji, wide characters and C0 controls, the stored text
     equals the input with each CRLF and lone CR turned into one LF and
     nothing else changed, and the `message.content` decoded from the bytes
     written to a fake Claude Code's stdin equals the stored text.
  2. A paste ending in a newline sends nothing (0 writes); Enter afterwards
     writes exactly 1 message; after `PasteStartMsg` with no end, Enter writes
     0 and the editor shows "pasting…".
  3. Test 6, measured: keystroke-to-frame 95th percentile with a 10,000-line
     buffer at most 16 ms; 50,000-line paste-to-frame at most 100 ms. Either
     figure over its target is a FAIL, reported with its value.
  4. Streaming: golden frames of P2's replay at successive deltas show the
     answer growing; a follow-up is written to the same session (same session
     id); after an interrupt the session is Ready, the turn Idle and the
     draft typed during the turn unchanged.
  5. Every conversation entry, after replaying every fixture and mock file,
     carries the `Seq` of an applied CLI event of a kind allowed for its
     role; a planted entry without one makes the test fail.
  6. Test 12: OSC 52 and its tmux wrapping equal the expected bytes; the
     native path writes through a fake; a mouse selection copies a wrapped
     line as one line, without borders, symbols or origin labels, and a code
     block exactly, tabs included.
  7. Markdown (ASM-17): a 2,000-line message streamed one delta per frame,
     measured with Glamour v2 against 16 ms per frame; if it misses, our own
     renderer for the CLIs' subset, measured against the same target, no new
     dependency.
  8. The app's input tests give identical results with the Claude Code,
     Codex and mock adapters attached.
  9. Listed for the owner, not verified by the agent: Windows Terminal, VS
     Code terminal and conhost for paste, Shift+Enter and copy. The owner
     then asked for it automated wherever possible (keyboard, multiline
     paste, clipboard, mouse selection, F12, scrolling, rendering), every
     check maximised as well as at the default size, and dropped the VS
     Code terminal (brief, amendment 6).
  Regressions: `go build`, `go vet`, `gofmt -l .` clean; `go test ./...`
  all PASS, test 7 included.
  Progress (2026-10-16):
  - Done: the input editor drawn in B (`internal/render/editor.go`: tabs at
    4 columns, controls as `^X`, C1 and invalid bytes as `<XX>`, hard wrap,
    cursor, selection, "pasting…", "N lines · Ctrl+G editor"); copy keys,
    toasts, `$EDITOR` round trip (`internal/app/editing.go`); the thinking
    flag no longer greys a real answer (stream frame 06 and the three
    dashboard frames regenerated, diffs reviewed).
  - Criterion 1 PASS: `TestInputByteForByte`, ten cases, through the real
    Claude Code adapter on an in-memory process; each stdin line equals the
    expected JSON byte for byte (a mutation, HTML escaping on, makes it fail).
  - Criterion 2 PASS: `TestPasteNeverSubmits`.
  - Criterion 3 PASS, measured by `TestKeystrokeToFrame` and
    `TestPasteToFrame` (Update plus View): keystroke p50 2.0 ms, p95 4.0 ms,
    max 10.2 ms, 800 keys, 10,000-line draft; 50,000-line paste (3.35 MB)
    p50 2.5 ms, max 3.3 ms over 10 runs.
  - Criteria 4 and 5 PASS (`TestStreamingP2`, the trace test).
  - Criterion 6 PASS: OSC 52, tmux wrap, SSH and the native path through
    a fake (`TestCopyKeys`, `TestCopyCutAndInterrupt`); mouse selection
    (`TestMouseSelection`): a wrapped line copies as one line, a code block
    exactly with its tab, a drag across borders, labels and into C copies
    exactly the answer plus "Bash", nothing else; double and triple click;
    the highlight leaves the plain frame unchanged. Also
    `TestCtrlCAfterMouseSelection` (Ctrl+C copies, then interrupts),
    `TestEditorMouse`, `TestMouseFocusAndCapture` (a plain click selects
    nothing; F12), `TestConversationScroll` (wheel, PgUp/PgDn, "↓ N new",
    long pastes folded to "… N more lines").
  - Criterion 7: Glamour v2 re-rendering the whole message missed (p95
    590 ms; 3.5 ms with a per-block cache). The owner chose our own
    renderer, no new dependency, with Markdown styling and highlighting in
    the final stage (2026-10-16; ASM-17 updated). PASS for the renderer as
    built (plain wrapped text, every byte kept), `TestStreamToFrame`, 2,000
    lines at one line per delta through Update and View: p95 5.5, 4.4 and
    5.0 ms over three runs (p50 1.0 to 2.7 ms, max 16.5 to 23.9 ms). It
    first measured p95 58.8 ms; a per-line wrap cache, rows drawn only when
    shown and rows sharing their styles (1,456 to 80 bytes a row) brought
    it under. The Markdown subset is re-measured when it is added.
  - Criterion 8 PASS with the Claude Code adapter, a scripted session under
    the `codex` id and the mock adapter; the Codex adapter's own session is
    step 5, so its stdin bytes are not verified yet.
  - Done in B: scrolling and follow mode with "↓ N new", long pastes
    folded, Ctrl+C with nothing to interrupt saying "Ctrl+Q quits".
  - Regressions PASS: `go build ./...`, `go vet ./...`, `gofmt -l .`
    empty, `go test ./...` all ok (internal/app 130 s); golden frames
    unchanged by the renderer work.
  - Criterion 9, measured 2026-10-16 in Windows Terminal 1.23.20211.0 and
    conhost (Windows 11 10.0.26200.8246), the mock replay in each, maximised
    from launch and at the default size: 4 runs, 40 to 41 checks each. A
    harness outside the repository sends real keyboard and mouse input
    (SendInput), sets and reads the system clipboard, and reads the screen
    through UI Automation; no CLI, no quota. Results, the same at both sizes:
    - Windows Terminal: 39 of 41 PASS maximised (156x44), 38 of 40 at
      120x30. Paste (Ctrl+V and right-click) never submits and is kept byte
      for byte, CRLF as LF: tabs, trailing spaces, wide characters, emoji,
      and a 2,000-line paste of 94,908 characters (on screen 2.9 to 6.1 s
      after Ctrl+V: WT's input throughput, not a frame cost); typed text
      exact; Ctrl+J newline; Tab, Backspace, Left; Enter sends; mouse
      triple-click, double-click and a drag over a wrapped message copy
      exactly; F12 hands selection to WT and back; Ctrl+O and OSC 52 reach
      the clipboard; wheel and PgUp/PgDn scroll B and return; every frame
      closed by its borders after replay, paste, resize (compact layout at
      86 columns) and maximising; Ctrl+Q quits. **FAIL: Shift+Enter
      arrives as Enter and sends the message** (K2, K5).
    - conhost: 36 of 41 PASS maximised (213x52), 35 of 40 at 120x40. As WT
      (F12 hands selection to QuickEdit), except **FAIL**: Shift+Enter
      sends (K2, K5); a typed emoji arrives twice (K1; O2, which copies
      typed text through OSC 52, fails for the same reason: the OSC 52
      path itself delivered); conhost's own right-click paste loses the
      emoji and the final newline (P9; Ctrl+V keeps them).
    - Cause of every FAIL, from `TEA_TRACE` and the console input queue:
      Bubble Tea's Windows input reader (ultraviolet
      `terminal_reader_windows.go`) turns console key records into VT
      bytes without Shift on Enter, decodes a surrogate pair on both key
      down and key up, and writes NUL for a Char-0 key down; conhost's
      paste sends non-keyboard characters as Alt+numpad records. Fixing it
      means patching a dependency: proposed to the owner, not applied.
      Ctrl+J is the newline key meanwhile, and the hint line names it.
    - Harness defects found and fixed on the way, none in the app: a
      PowerShell variable clash (`$k` and `$K` are one variable) sent keys
      as VK 0; a drag moved the cursor with SetCursorPos, which gives WT no
      pointer moves, so its own selection never started (now SendInput
      moves); checks looked for lines that B folds.
    - App defects the runs found, fixed with a deterministic test each:
      zone C broken by an agent's multi-line summary at 156 columns
      (`TestFrameInvariants`, widths 80 to 240); Ctrl+O copying a
      thinking-only entry (`TestCopyLastAnswer`); F12 in conhost leaving no
      selection, as Bubble Tea clears QuickEdit (`TestTerminalSelects`,
      `input.TerminalSelects`, and a refusal is shown); a raw Ctrl+V whose
      clipboard read fails now says "paste failed: …" instead of doing
      nothing (`TestRawPasteFailure`).
    - Not verified: Shift-drag selection in WT while the app captures the
      mouse; a NUL in a paste (the Windows clipboard ends its text at the
      first NUL, so no terminal can deliver one; the editor keeps NUL,
      `TestInputByteForByte`).
    - Later the same day the owner dropped conhost (brief, amendment 7):
      the conhost results above stay as measured, a record only. Its one
      adaptation, F12 handing selection to QuickEdit
      (`input.TerminalSelects`, `TestTerminalSelects`), was removed; F12
      still turns capture off for WT (`TestMouseFocusAndCapture`).
  - Left for later steps, not criteria of this one: Enter expanding folded
    items; scrolling in A and C; copying failure-card text; the Codex
    adapter's own stdin bytes (step 5).
- Step 4, permissions and session continuity: **deterministic checks 1
  to 5 done** (2026-10-16). Check 6 (L2): the first and second runs failed
  at turn 2. The first was a test defect; the second was an app defect,
  found and fixed since (a streaming answer marked finished). The third run
  passed turns 1 to 3, the crash and the resume, and failed two of the
  test's own checks (resume argument order, quit timing): test defects,
  fixed on fixtures. **The fourth run passed every check** (2026-10-16):
  step 4 is done. Checks, yes or no, deterministic unless marked live:
  1. Approvals (`TestApprovalCards`): P2's request is a card in B and P3b's
     on its subagent's row and in B labelled with it; `a` or `d` writes one
     `control_response` (fake CLI's stdin), a second key writes nothing; an
     unreadable request reads "denied automatically: request not readable"
     with one deny written; a pending card reads "expired, not answered"
     after the turn ends; no frame contains "always", "don't ask again" or
     an `s` key for Claude Code.
  2. `c` (`TestApprovalCancel`): stdin holds the deny line, then the
     interrupt line, in that order; the session is Ready; the next message
     is written to the same process.
  3. Crash (`TestCrashAndResume`): a fake CLI exits 3 with stderr after
     Ready; the card shows "exit code 3" and the stderr tail; the record
     and the draft are byte-identical before and after; Ctrl+R's argv is
     the first spawn's plus `--resume <id>`; a crash before Ready shows
     StartFailed and Ctrl+R spawns nothing.
  4. Drafts (`TestDraft*`): a draft saved, the model dropped without
     shutdown, a new model in the same project opens with it byte for
     byte; an accepted send deletes the file, a failed send keeps it; the
     file is in `%LOCALAPPDATA%\agent-command-center\drafts`, named by the
     SHA-256 of the project path; keystroke p95 under 16 ms with a
     50,000-line draft and saves running.
  5. Panic (`TestPanic`, a child process): a panic injected in the reducer
     ends the run with exit code 2, the draft file written, the fake CLI
     seeing end of input, and the session id and `claude --resume <id>`
     printed.
  6. L2, live (`internal/app/live_test.go`, `-tags live`): approved by the
     owner on 2026-10-16. Scope: Claude Code with `--model haiku` in an
     empty folder outside the repository, three short turns, each capped
     at 2 minutes; any failure stops the run, with no retry without the
     owner. The test drives the real dashboard model with the owner's
     keys: F2, `a`, Ctrl+C, an outside `taskkill /F`, Ctrl+R and Ctrl+Q.

  Results (2026-10-16): `go build`, `go vet`, `gofmt -l .` clean;
  `go test ./... -count=1` passes. The fake CLI is the test binary's
  `wireClaude`, which reads what the app writes to its stdin.
  1. PASS. P2's Write card in B; P3b's Bash card on agent
     a9dcbbc8d63adf34c's row ("waiting: approval") and in B. `F2` then
     `a a d` writes exactly one allow, which echoes `updatedInput`. A key
     without focus types into the editor. An `agent_id` sent as a number
     makes the request unreadable: one deny is written with
     `claude.DenyMessage`. After a result, the card reads "expired, not
     answered" and nothing is written. No frame offers "always" or `s`.
  2. PASS. The deny line comes before the interrupt line on stdin, through
     the one serial writer. The session goes from Interrupting to Ready
     with the approval Denied, and the next message reaches the same
     process (1 spawn).
  3. PASS. Exit 3 with stderr after Ready. The card shows "exit code 3",
     the stderr tail and "Ctrl+R resume"; the conversation and the draft
     are unchanged. Ctrl+R's argv is the first spawn's plus
     `--resume <id>`, and the session is Ready again. When the crash comes
     before Ready, the session is StartFailed, there is no Ctrl+R hint and
     nothing spawns.
  4. PASS. `TestDraftRestored`, `TestDraftAcceptedDeletes`,
     `TestDraftFailedSendKeeps`, `TestDraftPath` and `TestDrafts` (input).
     The draft text contains tabs, trailing spaces, CJK, an emoji, a
     combining mark, ESC and NUL, and is restored byte for byte. Between
     the send and its acceptance, the file holds the sent message; typing
     does not replace it. The file is deleted at acceptance. Files are
     named by SHA-256, case-insensitively on Windows; no temporary file
     is left. `TestDraftKeystrokes`: a 50,000-line draft, 800 keys 3 ms
     apart, real 500 ms save timer: p50 2.5 ms, p95 4.8 ms, max 8.7 ms,
     11 saves (target 16 ms).
  5. PASS. `TestPanic`: the child process exits with code 2 and prints
     "stopped on an internal error", followed by "session s-panic · resume
     it in <project> with: claude --resume s-panic". The draft file equals
     the typed draft, and the fake CLI saw end of input. The panic is
     injected at the start of `Update`, where the reducer runs, through a
     test-only hook; it takes the same recovery path. A planted fault (no
     flush in `Run`) fails the test.
  - Defects found and fixed, each with a test that fails without its fix:
    - The adapter refused to deny an unreadable request ("no pending
      request"); it now tracks the request and accepts only a deny.
    - A result arriving while Submitted, with nothing streamed before it,
      did not accept the message. The sent text stayed as the stored
      draft and would have come back as a duplicate. A turn end now
      counts as the turn's first event (machines test, `Submitted` row).
    - A usage error exited with 2, the same code as a panic; it now exits
      with 1.
  - Golden frames rewritten and reviewed. A resolved approval is now one
    line ("✓ orchestrator · Write <path> allowed by you"), and zone B's
    header shows "Interrupting…".
  6. **FAIL**, first run (2026-10-16 19:37, Claude Code 2.1.301, which
     auto-updated from 2.1.300; 25 s; about $0.01 API-equivalent). Stopped
     at turn 2 and not retried.
     - Turn 1 PASS. The Write request for `allow.txt` in the folder became
       a card. F2 then `a` allowed it, `allow.txt` holds "ok", and the
       answer was "done".
     - Turn 2 not checked, because of a test defect. The harness's `send`
       waits for 300 ms without a message, and a streamed answer never
       pauses that long, so all 300 lines streamed inside the Enter
       keypress. The test now delivers Enter and Ctrl+C without waiting.
     - Test defect, found before it could fail: Claude Code reports the
       session id with the first turn, not at Ready, so the resume check
       would have used an empty id. The test now reads the id after turn 1.
     - App defect, a real one: a resolved approval whose path is long lost
       its outcome off the end of the line. At 160 columns, "allowed by
       you" did not show. The reviewed golden frames had it too ("all…",
       "02:0…").
       - Fixed: the target now gets the room left once the outcome or
         clock fits. A path is shortened from the left and keeps its file
         name; a command is cut on the right.
       - `elide` no longer prints "……" when the cut meets the hidden user
         folder.
       - `TestApprovalLineKeepsOutcome` covers 5 statuses and widths 60
         to 200. It fails 97 times on the old code and passes now. Eight
         golden frames were rewritten and reviewed; the only change is
         this line.
     - Not run: interruption, kill and crash card, Ctrl+R resume, memory
       after resume, and quit.
     - Configuration after the run: every configuration file is identical,
       and `~\.claude.json` is "changed, not classified" (exit 2,
       NOT-VERIFIED), as in L1. The folder holds only `allow.txt`. No L2
       process is left running.
     - A second run, with the same scope and test, needs the owner's
       agreement.
  6. **FAIL**, second run (approved 2026-10-16; run at 19:50, Claude Code
     2.1.301, Haiku 5.5 `claude-haiku-5-5`; 19 s; about $0.01
     API-equivalent). Stopped at turn 2 and not retried. Same scope; no
     configuration change by the run or by me.
     - Turn 1 PASS. Session `14a9a3d2-7829-40f8-bd7d-3a2d0385ae9f`; F2
       then `a` allowed the Write in the folder; `allow.txt` holds "ok";
       the answer was "done"; the frame shows "allowed by you".
     - Turn 2 FAIL: "no streamed text before the turn ended". The turn
       completed with all 300 lines and no Ctrl+C was sent. The test
       waited for the last conversation entry to be an answer, streaming,
       with more than 20 characters, and never saw that state between two
       batches of events. The final entry also carried a thinking block
       (`∴ thinking`).
     - Cause not settled: the test logged nothing on the stream. The P3
       and P3b recordings (Claude Code 2.1.300) show text deltas about
       20 ms apart, so the stream itself paces. Two hypotheses remain: the
       deltas reached the dashboard in a few large batches, or each delta
       made an entry of its own (a message id missing). The second would be
       an app defect: the answer would show as fragments.
     - Test changes since, for a third run (none weakens a criterion):
       - Ctrl+C fires at the first streamed text of the turn's answer,
         wherever its entry stands, still only while the turn runs.
       - The test counts the turn's event batches and deltas, logs when
         the text deltas came, and with `ACC_LIVE_RECORD` keeps the CLI's
         stdout and read timings outside the project (since replaced:
         the recorder keeps the stream's shape only, see below).
       - New check: each answer of turn 2 has one entry with a message id
         (catches the fragments hypothesis).
       - `lastAssistant` reads an answer that had a thinking block: turn 3
         would otherwise miss the code word if Haiku thought first.
     - Not run: interruption, kill and crash card, Ctrl+R resume, memory
       after resume, and quit.
     - Configuration after the run: identical to before for every file,
       `~\.claude.json` "changed, not classified" (exit 2, NOT-VERIFIED)
       as in L1 and the first run. The folder holds only `allow.txt`. No
       `claude.exe` with `--model haiku` is left; the three running predate
       the run.
     - A third run needs the owner's approval.
  - Before the third L2 run (approved 2026-10-16, on condition that the
    test logic is first checked on deterministic fixtures and that any
    defect found is reported before the run):
    - Cause of the second run's failure, settled from P2 without quota:
      an **app defect**. Claude Code sends one `assistant` line per
      content block, each with the message's id. P2's last turn sends its
      thinking block's line before its text deltas, and the reducer's
      `completed` marked the entry finished at that line. The 300 lines
      then streamed into an entry the dashboard showed as finished: no
      streaming cursor, so the test's trigger never held. The same rule let
      a later block's line replace the text already streamed: a thinking
      line wiped it, and a second text block's line dropped the first.
    - Fixed in `internal/state/reduce.go`. A text or thinking delta marks
      its entry streaming again. A block's line keeps the text streamed:
      an empty line, or one the text already holds, changes nothing. A line
      that extends the text replaces it, and one that contradicts it still
      wins, with its note. Codex's whole message still ends the streaming.
      `doc/events-and-state.md`, `conversation.message.completed`, says
      so.
    - `TestAnswerStreamsAcrossBlockLines` (state) fails in 4 of 5 cases
      on the old code and passes now. Golden frame `stream-p2-05` now
      shows P2's "1" with its streaming cursor; the old frame had recorded
      the defect. It was reviewed, and it is the only frame change.
    - The live test's turn 2 is now `interruptMidStream`, in
      `internal/app/livecheck_test.go`, without the live tag. It sends the
      prompt and presses Ctrl+C at the first streamed text, and the frame
      must show the streaming cursor at that moment. It then requires:
      - the turn idle within 10 s, outcome "interrupted", session Ready;
      - **the CLI's own result** "interrupted" (`error_during_execution`
        after the interrupt). The dashboard calls any end after Ctrl+C
        interrupted (events-and-state.md, Turn), so its outcome alone
        would not prove Ctrl+C stopped the turn;
      - one entry per message id;
      - one answer with text, which begins with every byte streamed.
    - `TestInterruptMidStream` plays P2's interrupted turn through the
      real adapter. It has the thinking block, then a text delta every
      20 ms (P3, P3b), up to 300, and ends as P2 does once the interrupt is
      written. Five runs in a row:
      - paced: **PASS**; Ctrl+C after the first delta, idle 20 ms later;
      - burst, the whole answer at once: the check **fails**, as it must;
      - not interrupted, the turn ends "success": the check **fails** on
        the CLI's result.
      With the reducer fix undone, the paced case fails with the second
      run's own message: "no streamed text before the turn ended
      ("completed")", after 300 deltas.
    - `TestFragmentChecks`: deltas without an id, two entries with one
      id, two messages, and an answer that lost its text all fail the
      fragment checks; one answer passes.
    - The recorder no longer keeps the CLI's stdout. `ACC_LIVE_RECORD`
      now receives `reads-N.txt` (time and size of each read) and
      `lines-N.tsv` (per line: time, size, type, subtype, event, block,
      delta type, text size, message number). It holds no text, id or
      path. `TestShapeReader` on P2 checks every byte passes through, one
      row per line, and only numbers and lower-case protocol names.
    - `go build`, `go vet` (with and without `-tags live`) and `gofmt -l .`
      are clean; `go test ./... -count=1` passes.
  6. **FAIL**, third run (approved 2026-10-16; run at 20:21, Claude Code
     2.1.301, `claude-haiku-5-5`, 20.9 s; `ACC_LIVE_RECORD` set):
     - Turn 1: PASS. Write approval allowed with F2 and `a`; `allow.txt`
       holds "ok"; answer "done".
     - Turn 2: PASS. The thinking block's `assistant` line came before
       the first text delta, the case the fix covers. Ctrl+C at the first
       text delta (3 bytes), with the cursor on screen; the turn was idle
       21 ms later. The CLI's result was `error_during_execution` after the
       interrupt, so the outcome was interrupted and the session Ready. One
       entry per message, one coherent answer.
     - Crash: PASS. `taskkill /F`, exit code 1, the crash card with
       Ctrl+R.
     - Resume: the session resumed with the same id, but the test's check
       **failed**. It wanted `--resume <id>` after `--model haiku`;
       the adapter puts it before, as it documents (`Provider.Args` come
       after the session's own flags). Test defect.
     - Turn 3 after the resume: PASS, "LANTERN-73".
     - Quit: **FAIL**, "Ctrl+Q did not quit". The frame showed CLOSING.
       Ctrl+Q closes Claude Code's input and quits once it has ended
       ("close the CLI, restore the terminal"). Claude Code ended with code
       0 about 1.8 s after its input closed; the test looked 300 ms after
       the key. Test defect.
     - Configuration: NOT-VERIFIED, as before: `~\.claude.json` changed,
       not classified (contents not inspected); every other file
       identical; 1 new session file. The project folder holds only
       `allow.txt`; no Haiku `claude.exe` left (the three running were
       there before the run).
     - The recording: `lines-1.tsv`, `lines-2.tsv`, `reads-1.txt`,
       `reads-2.txt`, in the session's scratch folder. They hold only
       digits and lower-case protocol names; no text, id or path.
  - After the third run (2026-10-16), without quota:
    - `resumeArgv` builds the expected resume arguments: the first
      spawn's, with `--resume <id>` before the adapter's extra ones.
      `quitWithin` waits for the quit up to `CloseTimeout` (5 s), the
      close step before any kill. Check 6 never fixed a quit time; the
      test's comment "quits at once" did, against the design.
    - `TestResumeAndQuitChecks` runs the run's last steps on the fake
      CLI: `--model haiku`, a crash, Ctrl+R, and a CLI that ends 1.8 s
      after its input closes. The old checks fail on it, as they failed
      live: `--resume` is before the extra arguments, and once settled
      the session is Closing, not quit. The new ones pass: Closed, quit
      within 2 s, exit code 0, no kill.
    - The owner's added requirement: a turn the CLI completed is never
      shown interrupted. The reducer forced "interrupted" on any turn end
      after Ctrl+C. It now keeps "completed"; any other end after Ctrl+C
      stays "interrupted" (events-and-state.md, Turn: new row). The
      mappers already report the CLI's own result. Tests:
      - `TestStateMachines`, new row "turn end reported as completed":
        it fails with the old rule ("got interrupted, want completed").
      - `TestInterruptMidStream`: in "burst" and "not interrupted",
        Claude Code completes the turn after Ctrl+C, and the dashboard
        must now show "completed". Both fail with the old rule.
    - `go build`, `go vet` (with and without `-tags live`), `gofmt -l .`
      clean. `go test ./... -count=1`: 4 of 6 runs pass. In 2 runs,
      `TestDraftKeystrokes` measured p95 18.8 ms and 17.0 ms against its
      16 ms target, with every package testing in parallel. It passes
      alone, 5 of 5 (p95 8.1 to 10.3 ms), and with `-p 1`, 2 of 2. The
      test times typing in the editor and touches none of today's
      changes. The target stays; this is open.
    - Investigation of that failure (2026-10-16, no code changed). It is
      not an app regression:
      - Alone, the working tree and HEAD 078e7db time the same: p50 4.2
        to 4.7 ms, p95 7.5 to 9.9 ms (HEAD 8.4 to 8.6 ms).
      - The CPU profile puts about 91% in View and 5% in Update. View
        costs about 4 ms at 1k, 10k or 50k draft lines. GOGC=off and
        disabling the saves change nothing.
      - The suite's schedule changed. At HEAD, `internal/app` took 191 s
        and this test ran after every other package had finished. With
        the synctest bubbles it takes 23 to 34 s, and the test now
        overlaps internal/licenses (its nested `go test` on
        third_party/ultraviolet) and internal/provider/claude.
      - The machine: an i7-1265U (2 performance, 8 efficiency cores) on
        battery, near base clock. Over six back-to-back suites,
        everything slowed (`internal/app` 23.7 to 34.9 s) and this p95
        rose from 9.5 to 17.8 ms. In that state `go test ./internal/app`
        alone failed 2 of 3.
      - The earlier 4.8 ms p95 was a different machine state.
      - Afterwards, `go test ./... -count=1` passed 6 of 6, p95 7.6 to
        11.5 ms. The failure can come back under sustained load.
      - The suggested measurement change (up to three passes, every one
        logged, failing only if all miss 16 ms) is not applied: it
        changes a criterion and waits for the owner.
      - The owner's decision (2026-10-16): option 2. The test and its 16 ms
        target stay unchanged. When the full suite misses the target, the
        keystroke tests are rerun once, alone, under controlled conditions,
        and both results are kept. A miss in that run is investigated
        (evaluation.md, Keystroke timing).
  - Before the fourth run (approved 2026-10-16, with the 5 s quit check
    accepted and a clean end of the app and its CLI to verify):
    - The live quit step now checks, before the shutdown could kill
      anything: the session is Closed, its exit code is 0, and Claude
      Code's process has ended with 0 within 2 s of the quit.
      `TestResumeAndQuitChecks` checks the same exit code in the state.
      With the fake CLI exiting 1 instead, it fails ("exit code 1 (true)
      in the state; want 0").
    - `TestResumeAndQuitChecks` and `TestInterruptMidStream` pass.
      `go build`, `go vet` (with and without `-tags live`) and
      `gofmt -l .` are clean. `go test ./... -count=1` passes, exit 0
      (`TestDraftKeystrokes` p95 10.8 ms, `TestKeystrokeToFrame` 3.5 ms).
  7. **PASS**, fourth run (run at 21:28, Claude Code 2.1.301,
     `claude-haiku-5-5`, 36.6 s; `ACC_LIVE_RECORD` set):
     - Turn 1: PASS. Write approval allowed with F2 and `a`; `allow.txt`
       holds "ok"; answer "done".
     - Turn 2: PASS. "interrupted" 65 ms after Ctrl+C, the session Ready;
       23 bytes streamed as one answer (6 thinking deltas, 2 text deltas).
     - Crash: PASS. `taskkill /F`, exit code 1, the crash card.
     - Resume: PASS. Ctrl+R spawned a second process with the expected
       arguments; the same session id.
     - Turn 3 after the resume: PASS, "LANTERN-73".
     - Quit: PASS. The app quit 1.911 s after Ctrl+Q with the session
       Closed. Claude Code (pid 23924) had exited with 0 before the
       shutdown.
     - Clean end, checked in the OS afterwards: no process with pid 23924
       or 25948 (the killed one), and none whose parent is either. The only
       `claude.exe` running were the two there before the run (28268,
       24256).
     - Configuration: NOT-VERIFIED, the accepted limit: `~\.claude.json`
       changed, not classified (contents not inspected); every other file
       identical; 1 new session file. The project folder holds only
       `allow.txt`.
     - The recording (`lines-1.tsv`, `lines-2.tsv`, `reads-1.txt`,
       `reads-2.txt`) holds only digits and lower-case protocol names.
  - Deferred, not criteria of this step:
    - Enter showing an approval's full diff.
    - Codex's `s` (accept for the session), at step 5.
    - A deny message specific to a question (CC-10 q3).
- Fixes before L2 and step 5 (the owner's request, 2026-10-16), committed
  with step 4.
  1. Shift+Enter in Windows Terminal: **PASS**.
     - The app asks for win32-input-mode (`CSI ?9001h`) in Init and turns
       it off in `Run` after the program ends, a panic included. On other
       systems it does neither.
     - The enhanced protocols were checked first.
       - Without them, WT 1.23.20211 (the installed version) sends
         Shift+Enter as a plain CR (measured at step 3).
       - The Kitty keyboard protocol through ConPTY needs WT 1.25
         (reliability.md, Terminals and input), so it could not be tested
         here.
       - win32-input-mode is WT's own protocol, and Bubble Tea and
         ultraviolet already decode it.
     - Recorded from WT 1.23 with `TEA_TRACE`, ultraviolet's decoder had
       two faults in that mode: it recased letters from the modifiers, and
       it dropped emoji sent with vk 255.
     - Both are fixed in a patched copy, `third_party/ultraviolet` (MIT),
       used through a `replace` in `go.mod`. No new module was added.
       `PATCHES.md` records the source, the change and the evidence.
       Upstream's tests pass on a scratch copy, apart from the deleted
       function's own subtest. That subtest's four cases now live in the
       copy's `patch_test.go` (`TestPatchedWin32Input`), with Caps
       Lock+Shift and an emoji sent with vk 255 and with vk 0. It passes 7
       of 7 on the copy and fails exactly 3 on the unpatched files.
       `internal/licenses` runs it, because `go test ./...` skips a nested
       module; breaking patch 2 makes that check fail.
     - `internal/licenses` now fails if a replaced module has no
       `PATCHES.md` or is not named in THIRD_PARTY.md. Both planted faults
       fail it.
     - Tests:
       - `TestWin32InputMode`: one request, and one reset after a quit
         and after a panic. Planted faults fail it.
       - `TestWin32InputKeys`: the real program on the recorded WT bytes
         gives "Z🙂éZz\né", Enter sends and Ctrl+Q exits. Unpatched, it
         gives "zéZZ\né".
     - Real WT, with the harness's keyboard, mouse and clipboard input:
       maximised, 43 of 43 checks PASS; at the default size, 42 of 42.
       This includes K2 (Shift+Enter inserts a newline), K5 (no newline key
       submits), K6 (Enter sends), K1 and O2 (emoji kept), K4 (case kept),
       and Q2 (the next program in the console reads plain `x\rz\r` after
       Ctrl+Q).
     - The keys were synthesised with SendInput, which WT receives as it
       does a keyboard. Physical key presses were not tested.
     - The hint line still names Ctrl+J, which works everywhere.
  2. Ctrl+Q during a turn: **PASS**.
     - While a turn or an agent runs, Ctrl+Q asks to confirm: a second
       Ctrl+Q quits and Esc stays.
     - Confirmed, the quit cancels each pending approval and interrupts the
       turn before stdin is closed, in that order, through the serial
       writer.
     - A message sent but not yet accepted goes back to the editor and the
       stored draft.
     - Tests: `TestQuitDuringTurn`, `TestQuitBeforeAccepted`,
       `TestQuitIdle`, and the machines-test rows. The state table in
       events-and-state.md is updated.
  3. brief.md line 284: `â€”` is now `—`, and nothing else changed (the
     diff is one line). No `â€` is left.
  4. `system/commands_changed`: **PASS**. It lists the session's slash
     commands and skills (P3 and P3b). It is now classified as a
     `status.changed` with no status, because no MVP zone lists commands; it
     is no longer `unrecognised`.
     - Tests: a mapper test, and the replayed mock file. The doc tables are
       updated.
  5. Race detector: **PASS**, 0 data races.
     - The owner approved option (a), a portable MinGW-w64 under the user
       profile. It is WinLibs GCC 16.2.0 with MinGW-w64 14.0.0 UCRT r2,
       from the GitHub release `16.2.0posix-14.0.0-ucrt-r2` of
       brechtsanders/winlibs_mingw, unzipped to
       `%LOCALAPPDATA%\Programs\winlibs-gcc-16.2.0-ucrt-r2`. There is no
       installer, no administrator right and no PATH change; the race
       session alone sets `PATH` and `CGO_ENABLED=1`.
     - Checksum: SHA-256 `d5dbafc4…5276a` matches both the author's
       `.sha256` file and GitHub's own asset digest, and the SHA-512 matches
       too. The zip is not signed.
     - Compatibility: Go's race detector on Windows needs mingw-w64
       runtime 8 or later, with `libsynchronization.a` (go.dev race
       detector article; golang.org/issue/61916). `gcc
       -print-file-name=libsynchronization.a` resolves it.
     - A planted race in a scratch program is reported ("WARNING: DATA
       RACE", FAIL), so the detector works.
     - `go test -race ./... -count=1`, run twice: every package passes,
       with 0 races; `internal/app` takes 102 s. The patched ultraviolet
       copy passes under `-race` too.
     - Timing targets are not judged under `-race`, which slows code
       several times over. In a `-race` build, `raceBuild` (build-tagged
       `race_on_test.go` and `race_off_test.go`) makes `overTarget` log
       the timing instead of failing; a normal build still fails it, as a
       planted 1 µs target shows. Logged under `-race`: keystroke p95
       69 ms (continuity) and 36 ms (perf), against the 16 ms target.
  6. Slow tests: **PASS**. `internal/app` took about 190 s and now takes
     20 to 32 s, with no test removed or weakened.
     - Cause: the harness waited 300 ms of real time after every message
       (`settle`).
     - Fix: the tests run in `testing/synctest` bubbles, where that wait
       ends as soon as every goroutine is blocked.
     - The tests that time real work (perf, `TestDraftKeystrokes`) still
       run on the real clock.
  - Checks: `go build ./...`, `go vet ./...` and `gofmt -l .` are clean, and
    `go test ./... -count=1` passes.
- Step 5, Codex integration: **done** (2026-10-16): checks 1 to 7 PASS,
  check 8 PASS under the keystroke fix's procedure, check 9 (L3 and L4) PASS
  on its last runs, results below. Checks,
  yes or no, deterministic unless marked live. The fake app-server is the
  test binary replaying P6, P6b and P7 (each recorded request waits for the
  adapter's request of the same method, response ids follow the adapter's);
  no check starts Codex or uses quota.
  1. Fixtures through steps 0 to 4's checks, on the real adapter and process
     plumbing: P6b's file-change approval reaches the screen as a card and
     `a` writes exactly `{"id":0,"result":{"decision":"accept"}}` once; P7's
     approval, requested by a child thread, is labelled with that thread's
     agent and answered with id 0; P6b's second turn, interrupted, writes
     `turn/interrupt` with the thread id and the running turn's id and ends
     the turn `interrupted` with the session Ready. P5 is `codex exec`
     output, which the MVP does not read: its lines stay covered by test 1
     only.
  2. Requests: `thread/start` params are exactly
     `{"sandbox":"read-only","approvalPolicy":"on-request"}`, plus `model`
     only after a refusal, and never `cwd` (architecture.md: cwd with a
     write-capable sandbox writes `trust_level` into config.toml). A test
     drives every request the adapter can write (initialize, initialized,
     thread/start, thread/resume, model/list, turn/start, turn/interrupt,
     account/rateLimits/read, the four answers, a JSON-RPC error) and finds
     no `workspace-write`, `danger-full-access`, `cwd`, `config/`,
     `skills/config/write`, `marketplace/`, `plugin/`,
     `externalAgentConfig/`, `windowsSandbox/` or execpolicy method; a
     planted `cwd` fails it.
  3. Model refusal (P6): the error is in the errors list with the text as
     received ("The 'gpt-6-sol' model is not supported when using Codex
     with a ChatGPT account."), then `model/list` is written, then
     `thread/start` with `"model":"gpt-5.6-sol"` (the `isDefault` entry of
     `p6-codex-model-list.json`); B shows a notice naming both models; the
     next `turn/start` names the new thread. A second refusal starts no
     second fallback.
  4. Answers: `a`, `d` and `c` write `accept`, `decline` and `cancel` (then
     `turn/interrupt`), echoing the request's id as sent; `allow`, `deny`
     and an unknown id are refused without a write. An unknown server
     request gets a JSON-RPC error (-32601) with its id and is counted as
     unrecognised. Allow for the session (`s`) is not offered: no recorded
     request carries `availableDecisions`.
  5. Continuity: with a resume id the adapter writes `thread/resume
     {threadId, sandbox, approvalPolicy}` and no `thread/start`; in the app,
     Ctrl+R after a crash spawns `codex app-server` again and resumes the
     crashed thread.
  6. Process: Claude Code's process tests pass on the Codex adapter: exit
     mid-turn gives a crash with the code, an over-long line is skipped,
     the close escalation ends a CLI that ignores stdin (between CloseStep
     and 11 s) and the forced kill leaves no process.
  7. The input tests (step 3, item 8) run on the real Codex adapter instead
     of the scripted session.
  8. Checks: `go build ./...`, `go vet ./...`, `go vet -tags live ./...`,
     `gofmt -l .` clean; `go test ./... -count=1` passes; keystroke timing
     as evaluation.md says.
  9. L3 and L4 (live): **not started; each needs the owner's approval** of
     its scope, model, expected usage and stopping conditions.
  - Results, 2026-10-16 (offline; checks 1 to 7 PASS, 8 FAIL on keystroke
    timing only, 9 not started):
    1. PASS: `TestHandshakeApprovalInterrupt`, `TestChildApproval` (codex
       package); `TestCodexApprovalCards`, `TestCodexChildApproval`,
       `TestCodexInterrupt` (app), on P6b's and P7's exact lines.
    2. PASS: `TestForbiddenCheck` (a planted `cwd` and each forbidden method
       or value fails it), `TestNoForbiddenMethodInSource`, and every session
       test runs `forbidden` on each line written.
    3. PASS: `TestModelFallback` (P6 then P6b: the error as received,
       `model/list`, `thread/start` with `gpt-5.6-sol`, the notice),
       `TestNoSecondFallback`. Limitation: the message refused with the old
       model is not resent; the owner sends it again.
    4. PASS: `TestAnswersEchoTheID` (string ids, decline, cancel),
       `TestUnknownServerRequest` (-32601, one unrecognised),
       `TestUnreadableRequest`; the app's a, d, c write accept, decline,
       cancel then `turn/interrupt`; no frame offers `s`. Fixed on the way:
       the reducer auto-answered an unreadable Codex request `deny`, a word
       Codex does not accept; it now answers `decline`
       (`machines_test.go`).
    5. PASS: `TestResume` (`thread/resume` with the pinned model, no
       `thread/start`); `TestCodexCrashAndResume` (two identical
       `app-server` spawns, the third line written is `thread/resume` for
       the crashed thread, the draft kept).
    6. PASS: `TestExitMidTurn` (code 3), `TestLongLine`,
       `TestCloseEscalation`, `TestForcedKill`, `TestNoHandshake`,
       `TestRefusedThreadStart`. The line queue and reader shared with
       Claude Code moved to `internal/provider/pipe`.
    7. PASS: `TestInputByteForByte` and `TestPasteNeverSubmits` run with
       the `codex` rig (`codexRig`, every message a `turn/start` line).
    8. FAIL on keystroke timing only. `go build`, `go vet`,
       `go vet -tags live` and `gofmt -l .` are clean, and every package
       except `internal/app` passes. In `internal/app` only the keystroke
       tests fail:
       - Full suite, run 1: `TestDraftKeystrokes` p50 12.94 ms, p95
         32.55 ms, max 224.86 ms.
       - Isolated rerun (evaluation.md, Keystroke timing, step 2), on mains
         power (battery at 6%, charging); RanorexStudio, Firefox and
         VS Code were open, CPU load 44% on 12 logical CPUs:
         - `TestDraftKeystrokes`: p50 11.66 ms, p95 22.59 ms, max
           144.18 ms;
         - `TestKeystrokeToFrame`: p50 8.23 ms, p95 17.70 ms, max
           50.32 ms.
         FAIL.
       - Investigation (step 4): the same isolated run on the last commit
         6d3e548, in a temporary worktree, fails the same way:
         - `TestDraftKeystrokes`: p95 34.50 ms;
         - `TestKeystrokeToFrame`: p95 21.65 ms, max 302.67 ms.
         Step 5 changes no code on the keystroke path; only a comment in
         `keys.go` changed. The miss follows the machine's load, not this
         step.
       The criterion stays FAIL until an isolated run on a quiet machine
       meets 16 ms; it is not rerun meanwhile.
    9. Not started. The tests are written and only vetted:
       `internal/provider/codex/live_test.go` (`TestLiveGate`, L3) and
       `internal/app/live_codex_test.go` (`TestLiveDashboardCodex`, L4). L4's
       approval guard `insideCodex` is checked offline by `TestInsideCodex`.
  - Keystroke fix, approved by the owner on 2026-10-16, after check 8:
    headroom in the drawing, the timing moved to its own step, and a
    deterministic guard in `go test ./...` (evaluation.md, Keystroke timing,
    rewritten).
    - Changes:
      - The editor builds only the rows it shows and draws a run of plain
        cells as one span (`render/editor.go`).
      - Spans hold a shared `*lipgloss.Style`.
      - `Screen.StringAfter` reuses each line drawn the same as in the frame
        before; `View` uses it.
      - Rune widths past U+02FF are memoised.
      - `TestKeystrokeToFrame` and `TestDraftKeystrokes` moved to
        `keystroke_test.go` under the `timing` tag, with 50 untimed warm-up
        keys; the 10,000-line test now types real text (`typed('x')`; it
        inserted nothing before).
      - `TestPasteToFrame` and `TestStreamToFrame` stay in the suite.
    - Results, 2026-10-16:
      1. PASS: `TestGoldenFrames` passes without `-update` (8 frames,
         `testdata/frames` unchanged).
         - `TestEditorRowsMatchReference` holds the new editor to the old
           per-cell drawing, row by row, on 3,000 random cases; two planted
           faults fail it.
         - The frames with their escape codes are byte-identical before and
           after the style change (19 frames: every size, edits, wraps,
           controls, a selection).
         - `TestFrameReuse` checks reused frames equal fresh ones.
      2. PASS: per-key cost, `TestKeystrokeCost`, before and after:

         | Draft | Time | Bytes | Allocations |
         |---|---|---|---|
         | 10,000 lines | 1.57 ms → 0.22 ms | 3.78 MB → 0.40 MB | 8,462 → 822 |
         | 50,000 lines | 1.74 ms → 0.52 ms | 4.79 MB → 1.40 MB | 8,461 → 823 |

      3. PASS: the timing step under the conditions of evaluation.md
         (Keystroke timing, as the owner set them on 2026-10-16): on mains
         power, with no other heavy job running. CPU load was 24% on 12
         logical CPUs just before the run. RanorexStudio, Firefox and VS Code
         were open and idle.
         - `TestKeystrokeToFrame`: p50 0 s, p95 1.11 ms, max 2.01 ms. The p50
           of 0 s is below the clock's resolution on Windows.
         - `TestDraftKeystrokes`: p50 0.62 ms, p95 2.26 ms, max 4.88 ms, 8
           saves.
         - Target: p95 ≤ 16 ms.

         Stress check, run once at the owner's request: the same step while
         `go test ./... -count=1` ran in parallel. This is the load under which
         the old tests missed, with p95 up to 32.55 ms. Conditions: mains
         power; CPU load 43% just before the run; VS Code, Firefox and Edge
         open. The timing step took 15.6 s, and the slowest package of the
         full suite took 11.1 s, so the two overlapped for most of the run.
         The full suite also passed (exit 0).
         - `TestKeystrokeToFrame`: p50 0.53 ms, p95 1.07 ms, max 4.03 ms.
         - `TestDraftKeystrokes`: p50 1.19 ms, p95 3.92 ms, max 7.87 ms, 9
           saves.
         - `TestKeystrokeCost`: 0.26 ms, 0.40 MB and 822 allocations per key
           in the 10,000-line draft; 0.52 ms, 1.41 MB and 823 allocations in
           the 50,000-line draft.
         - PASS against the same target.
      4. PASS: `TestKeystrokeAllocs` is in `go test ./...`: at most 1,200
         allocations per key, and the editor's window at most 4 screen
         heights; measured 822 at the top of the draft and 593 at the bottom.
         Five planted faults fail it:

         | Planted fault | Result |
         |---|---|
         | No frame reuse | 2,014 allocations per key |
         | Per-cell editor spans | 2,951 |
         | No width memo | 3,376 |
         | The whole draft given to the renderer | a window of 10,002 lines |
         | That, with every row of the window built | 20,802 allocations and the 10,002-line window |

      5. PASS: `go build ./...`, `go vet ./...`, `go vet -tags live ./...`,
         `go vet -tags timing ./...` and `gofmt -l .` are clean;
         `go test ./... -count=1` passes in every package.
      6. PASS: evaluation.md, Keystroke timing, rewritten; the timing command
         added to AGENTS.md, Commands. The owner then adapted the conditions
         (2026-10-16). Usual idle applications may stay open. Mains power is
         required, with no concurrent heavy job. Power, CPU load and open
         applications are recorded with each run, not used as a gate.
    - Check 8 above stays FAIL as recorded. Under the new procedure, its
      keystroke part passes on this run (results 3 and 4).
  - Check 9, live L3 and L4, approved by the owner on 2026-10-16 ("Commit
    step 5 after L3 and L4"): L3 with `gpt-5.6-luna` (`ACC_LIVE_MODEL`), L4
    with Codex's default model. Each run in an empty folder under
    `%LOCALAPPDATA%\Temp`, with `confcheck` before and after; Codex 0.145.0.
    - L3, run 1 (`acc-l3-225635`): FAIL, a test defect. `start()` stopped at
      `session.ready` and read the thread and cwd before `session.started`,
      which the same response line gives next (`codex/mapper.go`,
      `threadStart`): "thread "", cwd """. No turn ran. Configuration PASS;
      the folder stayed empty. Fixed: `start()` waits for `session.started`.
    - L3, run 2 (`acc-l3b-225755`): **PASS** in 55.69 s. Thread
      `01a14681-e33f-7370-b4ad-8d2805b56264`, model `gpt-5.6-luna`, cwd the
      folder; handshake 481 ms.
      - Accept: the fileChange approval accepted, `accept.txt` holds "ok",
        17 text deltas. A second request in the same turn, a
        commandExecution, was declined by the test's rule ("exec command
        rejected by user").
      - Decline: completed, `decline.txt` absent, the answer 1.057 s after
        the decline.
      - Cancel: interrupted 8 ms after the answer.
      - Interrupt: interrupted 9 ms after `turn/interrupt`, 46 text deltas.
      - Close: exit 0, 3.768 s after stdin closed; 6 processes checked gone.
      - Resume: handshake 483 ms; the code word "ORCHID-41" recalled.
      - Kill: exit code 1, 24 ms after the kill; 9 processes checked gone.
      - 20 lines written, none forbidden.
      - Configuration PASS, 1 new session file; the folder holds only
        `accept.txt`.
    - L4, run 1 (`acc-l4-225916`): FAIL at turn 1 in 7.54 s. Codex's default
      model `gpt-6-sol` was refused ("The 'gpt-6-sol' model is not supported
      when using Codex with a ChatGPT account.", as in P6), and the test
      failed on the failed turn before the adapter's fallback had started
      its new thread. A test defect: the test did not follow the fallback of
      check 3. Configuration PASS; the folder stayed empty.
    - L4, run 2 (`acc-l4b-230252`): FAIL at turn 1 in 25.17 s, a test
      defect. The fallback worked live: the refusal, `model/list`, a new
      thread `01a14686-7d84-7363-8133-79a64b562ae6` on `gpt-5.6-sol`, the
      notice in B, and the message sent again. Its fileChange approval was
      accepted and `allow.txt` written; the model then asked for a
      commandExecution to remove the trailing newline, and the test's rule
      stopped the run on any second request. Fixed: as L3 does, a request
      other than the first inside the folder is declined with F2, d and the
      turn goes on; the prompt says a trailing newline is fine. Only one
      request is ever accepted. Configuration PASS; the folder holds only
      `allow.txt`.
    - L4, run 3 (`acc-l4c-230410`): **PASS** in 26.29 s.
      - Ready on `gpt-6-sol`, thread `01a14687-a46f-7cb3-a8a8-939324ecafa9`;
        turn 1 refused, the fallback thread
        `01a14687-ac6b-7192-88a9-ff013cd0f84e` on `gpt-5.6-sol`, the
        message sent again.
      - Turn 1: the fileChange accepted with F2, a; `allow.txt` holds "ok";
        answer "done"; no request declined; "allowed by you" in the frame.
      - Turn 2: interrupted 9 ms after Ctrl+C, one answer entry.
      - Crash (taskkill): exit code 1 shown; Ctrl+R resumed the fallback
        thread; turn 3 answered the code word "LANTERN-73".
      - Ctrl+Q: Closed 3.031 s after, `app-server` exited 0.
      - Configuration PASS, 2 new session files; the folder holds only
        `allow.txt`.
    - Findings:
      - Check 4's note is wrong for live Codex: L3's commandExecution request
        carried `availableDecisions`
        `["accept", {"acceptWithExecpolicyAmendment": …}, "cancel"]`. The
        adapter still offers only a, d and c. `acceptWithExecpolicyAmendment`
        must never be offered: it changes Codex's exec policy.
      - `thread/goal/cleared` is a notification the mapper does not
        recognise; it is counted as unrecognised (L3, L4).
      - Codex's websocket to `chatgpt.com/backend-api/codex/responses` got
        a proxy 407 (stderr); the turns worked regardless (L3).
      - A failed turn's error text reaches only the errors list, which D
        shows as a count ("errors 3" at L4's end); B shows the refusal only
        through the fallback notice. ui-design.md's "Enter on D lists the
        last errors" is not built yet and no step of the proposal names it.
        Owner's decision (2026-10-16): build it at step 7, below.
- Step 6, genuine agent activity: **done, 8 of 8 PASS under the amended
  criterion 1** (the owner's amendment of 2026-10-17; the FAIL against the
  original wording is kept below). Criteria, written
  2026-10-16 before the work, from proposal.md step 6, ui-design.md (Zone A,
  Zone C, Agent inspection) and events-and-state.md (Agent tree):
  1. Stages per turn: replaying P2 gives 3 stages (TURN 1 and 2 completed,
     TURN 3 interrupted) with P2's Write under the turn it ran in; P3b gives
     1 stage with the orchestrator's Agent tool and the subagent
     `general-purpose` under it; P7 gives 1 stage with both children
     `/root/list_files` and `/root/create_found` directly under it (today
     P7's children hang under the main thread's id, which no node has, so
     zone A shows none of them: a defect this step fixes in the mapper,
     Parent "" for the orchestrator's thread).
     Amended by the owner on 2026-10-17, the original wording kept above:
     "P3b gives 2 stages: turn 1 with the orchestrator's Agent tool and the
     subagent `general-purpose` under it, turn 2 opened by Claude Code
     itself, with no prompt, for the subagent's report". Reason: the
     criterion was written before P3b's second turn was looked at; the
     recording holds that turn, and the stage model shows it as the CLI's.
     No other criterion changed.
  2. Zone A renders "STAGES inferred from activity" for all three; each
     node shows its origin (stage "inferred", agent status as its machine
     says, tools "observed"); previous turns fold to one line "TURN n
     <symbol> <outcome> · <duration>"; "○ (more as events arrive)" shows
     while a turn runs and not after; ↑↓ move a cursor over the visible
     nodes, ←→ fold and unfold a turn, the cursor's node stays visible.
  3. Codex `turn/plan/updated` (schema only, not recorded: v2 schema
     `TurnPlanUpdatedNotification`): a line built from the schema maps to a
     stage event; zone A then heads "PLAN observed (turn/plan/updated)" with
     the steps and their statuses. Labelled "schema only, not recorded" in
     the code.
  4. Zone C: P3b's background subagent goes Running, WaitingApproval
     (twice), Running and Completed during the replay; P7's two children
     start and end Completed; rows ordered waiting, running, ended, most
     recent first (table test over all three groups); the counts give
     Running and WaitingApproval as active.
  5. Agent inspection: Enter on an agent in A or C opens it full screen with
     the header (back, label, model or "model N/A", status and elapsed,
     tokens, "[ ] prev/next") and tabs Messages, Tools, Diff, Raw events
     (←→ or 1 to 4). Each tab holds that agent's own events only: inspecting
     P7's `/root/list_files` shows none of `/root/create_found`'s tool ids,
     messages, approval or diff, and the reverse; P3b's subagent shows its
     two approvals and its Bash and Write; its Write shows in Diff as
     "derived"; P7's fileChange diff shows as "observed". `[` and `]` go to
     the previous and next agent; Esc returns to the dashboard with the same
     focus.
  6. Golden frames: the dashboard with each CLI's mock (Claude Code
     `claude-subagent-approval.jsonl`, Codex `codex-subagents.jsonl`) and
     the inspection view, at 80x24, 120x40 and 200x60; every frame exactly
     the terminal's height, no line wider; the diffs reviewed.
  7. The agent machine's table tests pass for both CLIs, with the rows this
     step adds (turn of an agent and a tool; parent of a Codex child).
  8. `go build ./...`, `go vet ./...` (also `-tags live` and `-tags
     timing`) exit 0; `gofmt -l .` prints nothing; `go test ./...` exit 0;
     the keystroke guard (`TestKeystrokeAllocs`) passes unchanged.
  Not verified by these checks: the look (the owner's visual review).
  - Results, 2026-10-16:
    1. FAIL against the original wording (kept); PASS under the amendment
       of 2026-10-17. P2 gives 3 stages (completed, completed,
       interrupted) with its Write under turn 1; P7 gives 1 stage with both
       children under it; both are as specified (`TestStagesP2`,
       `TestStagesP7`). P3b gives **2** stages, not 1: turn 1 holds the
       Agent tool and the subagent (stage 1, Parent ""), and Claude Code then
       opens turn 2 itself, with no prompt, for the agent's report
       (`ByCLI`). `TestStagesP3b` asserts what the recording holds. Cause:
       the criterion was written before P3b's second turn was looked at.
       Proposed amendment: "P3b gives 2 stages: turn 1 with the Agent tool
       and the subagent, turn 2 opened by the CLI for the report".
    2. PASS. `TestTreeP3b`: STAGES heading, turn 1 folded to one line by
       default, unfolded s:1, the tool, the agent, s:2; "observed" twice; no
       "more" after the turn. `TestTreeP7Running`: "○ (more as events
       arrive)" mid-turn only. `TestTreeKeys`: ↑ from no cursor goes to the
       last node, ↑↓ move, ← folds to 1 node, → unfolds to 5, Enter on a
       tool gives the zone B toast. `TestTreeWindowKeepsCursor`: the
       cursor's node stays in the window.
    3. PASS. `TestTreeP7Running` applies a `turn/plan/updated` line built
       from the schema (labelled "schema only, not recorded" in the
       mapper): heading "PLAN observed (turn/plan/updated)" with its steps.
       `TestPlanUpdated` (codex mapper): a stage event with its steps,
       observed; an unreadable plan is counted unrecognised.
    4. PASS. `TestStagesP3b`: Running, WaitingApproval, Running,
       WaitingApproval, Running, Completed, final origin observed.
       `TestStagesP7`: both children end Completed. `TestAgentOrder`: 6
       rows over waiting, running and ended, most recent first; active =
       Running + WaitingApproval.
    5. PASS. `TestInspectP7`: Tools, Diff, Raw and Messages of each child
       hold none of the other's (both ways); the fileChange diff "observed".
       `TestInspectP3b`: 2 approvals, Bash and Write once each, the Write
       "derived" in Diff, no Agent tool. `TestInspectKeysFromC`: Enter
       opens it with its header and tabs; ←→ and 1 to 4; `]` `[`; a stray
       letter dropped; Esc back to zone C. `TestTreeKeys`: Enter from A,
       Esc back to A with the cursor kept. `TestInspectScroll`: ↑ holds,
       "↓ 1 more", Home, the wheel, End follows; zone B not held.
    6. PASS. `TestGoldenFrames`: new `dashboard-codex-*`, `inspect-*` and
       `tree-*` at 80x24, 120x40, 200x60 (and 79x24 too small); `dashboard-*`
       and `stream-p2-01..06` regenerated (zone A now STAGES and TURN rows).
       Heights exact, no line wider; diffs reviewed. The stream-p2 frames
       show "running 9h 29m": the recording's times against the tests'
       fixed clock, deterministic. Mock mode labels origins "mock".
    7. PASS. `TestStateMachines` with the new rows (the turn of an agent and
       a tool; the parent of a Codex child).
    8. PASS. `go build ./...` 0; `go vet ./...` 0, also `-tags live` and
       `-tags timing`; `gofmt -l .` empty; `go test ./... -count=1` all ok;
       `TestKeystrokeAllocs` PASS unchanged (ctrl+home 820, ctrl+end 591,
       bound 1200). Timing step PASS: 10k lines p95 1.1 ms, 50k lines p95
       2.5 ms (target 16 ms).
    - Not verified: the look; live rendering in Windows Terminal; the zone A
      cursor's highlight (plain-text frames carry no background).
- Step 7, usage and quota telemetry: **done** (2026-10-17; approved by the owner after the mock scenarios, "it is good"): 8 criteria PASS; the timing step passed on battery and its gate run on mains power is outstanding.
  Criteria, written before the work, from proposal.md step 7, reliability.md
  (Telemetry sources), telemetry-sources.md, and the owner's two additions
  of 2026-10-16 (ui-design.md, States: Error; Zone D: Errors):
  1. Telemetry parsing (test 2), table tests on the probes' exact lines:
     P1 line 3 gives 5h 22% and 7d 45% with resets 1792109400 and
     1792216800, undocumented (shown "22%†"), and overage "rejected
     (org_level_disabled)" shown apart from quota; P1's `result` gives
     `total_cost_usd` 0.00249374, shown only as "API-equiv. estimate"; P6b
     line 7 gives 5h and 7d by `windowDurationMins` (300, 10080), plan
     "plus" and credits; P7 gives the thread totals 76,521, 30,455 and
     77,336 and the derived session total 184,312.
  2. Every N/A case of reliability.md's two value tables shows "N/A" with
     its reason in zone D, never 0 (one table-test row each, on zone D's
     text). Claude Code: quota before the first event "no report yet"; an
     API key (`apiKeySource` other than "none") "API key: no subscription
     window"; a window past its reset "window reset"; a `rate_limit_event`
     with `rateLimitType` and `status` only shows the status word with no
     percentage; overage "not reported"; context before the first turn
     "no turn yet", and **derived** (from the last message's usage and
     `modelUsage.contextWindow`, labelled "derived") once `get_context_usage`
     is refused, the refusal counted as an error (step 2's finding); tokens,
     cost and the last turn's duration N/A before the first `result`; agents
     N/A before the stream is up; "limit N/A (not reported)"; retries
     "N/A (none reported)" until one arrives. Codex: a rate-limits read
     refused for an API key "API key: ChatGPT sign-in needed"; windows
     without a 300-minute one "plan has no 5-hour limit"; another refused
     read "read failed: <error>" with the last pushed value and its age;
     context before a thread's first model call "no model call yet"; "cost
     N/A (Codex reports none)"; the thread limit N/A (see 3).
  3. Decided, not built: `config/read` (the source of `agents.max_threads`)
     is not called, because the effective configuration it returns can hold
     secrets (an MCP server's `env`, a provider's bearer token); the limit
     shows "limit N/A (not read)", and reliability.md records the reason.
     `rateLimitResetCredits` ids are never shown (a frame test).
  4. A failed turn shows in B "✗ turn failed: <first line of the error, as
     received>" in red, for each CLI: Codex P6's refusal replayed; Claude
     Code a `result` with `is_error` and its `errors` or `result` text (the
     documented schema; no probe failed a turn).
  5. Enter on D (focused) lists the last errors (`state.Errors`, at most
     `MaxErrors`), newest first: time, agent label or "session", the whole
     message wrapped; ↑↓ PgUp PgDn scroll; Esc returns to D.
  6. Long session (test 11, `-tags timing`): two hours of generated events
     at speed through the model; heap and retained list sizes reported; the
     frame time at the end within 16 ms (FAIL above).
  7. Golden frames: the dashboards updated for zone D's new values, and the
     errors list at 80x24, 120x40 and 200x60; exact height, no line wider;
     the diffs reviewed.
  8. `go build ./...`, `go vet ./...` (also `-tags live` and `-tags timing`)
     exit 0; `gofmt -l .` prints nothing; `go test ./...` exit 0; the timing
     step passes.
  Not verified by these checks: the look; live Windows Terminal rendering;
  a live `get_context_usage` refusal, a live API-key session and a live
  plan without a 5-hour window (no probe has them).
  - Results, 2026-10-17:
    1. PASS. `TestTelemetryParsing`, 5 cases on the probes' exact lines: P1
       line 3 (5h 22%, 7d 45%, the two resets, "22%†", overage "rejected
       (org_level_disabled)" apart); P1's `result` (0.00249374 only as
       "API-equiv. estimate"); P6b line 7 (300 and 10080 minutes, "plus",
       credits); P7 (76,521, 30,455, 77,336; session 184,312); P3b (the
       running total, subagents included).
    2. PASS. `TestTelemetryNA`, 16 rows, one per N/A case of both tables,
       on zone D's text: each shows "N/A" with its reason, never 0; the
       refused `get_context_usage` counts as an error and the context turns
       "derived" after the `result`. Fixed on the way: `missing()` gives the
       source's own reason (an API key, a failed read) before the generic
       "no report yet", with the last value and its age when there was one.
    3. PASS. `config/read` is not called; the limit shows "limit N/A (not
       read)"; reliability.md's Codex table records why (its effective
       configuration can hold secrets). `TestCreditIDsNotKept`: P6b's first
       lines plus a pushed and a read `rateLimitResetCredits` and a cut line;
       the secret id is in no event's Raw or Payload, no metric, no log
       entry (which says "not kept"), and no frame (200x60, the errors list
       at 200x60 and 80x24, ANSI stripped).
    4. PASS. `TestTurnFailedLine`: Codex P6's refusal gives two rows,
       "✗ turn failed: " bold red then the first line in red, as received;
       Claude Code `result` with `errors`, and with `result` text, give one
       row each, and `state.Errors` keeps the whole message.
    5. PASS. `TestDetailsErrors`: newest first, time and "session" or the
       agent's label (an unknown key shown as is), "retry", the whole
       message wrapped, no row wider; "no errors" when there are none.
       `TestDetailsKeys` (80x24, 23 errors): the title and "23 errors · last
       50 kept"; ↓ ↑ PgDn PgUp End Home and the wheel scroll by the expected
       rows without holding zone B; new errors show "26 errors"; Esc clears
       a selection first, then returns to D with "Enter details" and B shown.
       A long agent label now keeps its own row, whole.
    6. FAIL, then PASS (the first result kept). First runs: p95 at the end
       18 to 26 ms, p50 about 12 ms: every frame drew all 5,281 conversation
       entries' rows, all 720 agents in zone C, and the log's bounded append
       copied 2,000 events per event. Fixed: `appendBounded` slides its
       window (apply about 3x faster); zone C draws only the rows its box
       shows; tool and error rows in B are drawn only when shown; zone B
       draws only the entries in view and counts the others from a per-entry
       cache (`WrapCache.rowCount`, keyed by text, role, width and thinking;
       streaming entries and approval cards always counted).
       `TestConversationWindow` checks the cached frames equal the uncached
       ones across 5 sizes, 5 scroll positions, new entries, a changed
       entry, a thinking flag and moved indices; removing the width, text or
       thinking key each makes it fail (60, 10 and 5 frames). Now, 26,209
       events (240 turns, 720 agents, 6,960 tools): heap 6.5 MiB after (1.2
       before); kept conversation 5,281, stages 240, agents 720, tools 6,960,
       diffs 20, log 2,000, errors 48; at the end p50 2.0 to 3.7 ms, p95 4.4
       to 7.1 ms, max 5.5 to 10.1 ms (4 runs; target 16 ms). Conditions not
       met: these runs were on battery (see 8).
    7. PASS. `TestGoldenFrames`: new `details-*` at 80x24, 120x40, 200x60
       (and 79x24 too small); the dashboards and stream frames updated for
       zone D's values; heights exact, no line wider; diffs reviewed. The
       zone B window left every golden frame unchanged.
    8. PASS for the checks; the timing step's conditions not met. `go build
       ./...` 0; `go vet ./...` 0, also `-tags live` and `-tags timing`;
       `gofmt -l .` empty; `go test ./... -count=1` all ok. Timing step, run
       once on 2026-10-17: every test passes (10k lines p95 1.0 ms, 50k lines
       p95 3.0 ms, `TestLongSession` p95 4.4 ms; `TestKeystrokeCost` 0.30 and
       0.66 ms per key, 825 and 826 allocations), but on **battery** (power
       line offline, 72%), CPU load 29%, with VS Code, Firefox, Teams,
       Outlook and Notepad open. evaluation.md requires mains power: the
       gate run is outstanding. `TestLongSession` joins the timing command
       (AGENTS.md, evaluation.md).
    - Simulated on mock, at the owner's request (2026-10-17): the three
      cases no probe has, as synthetic mock files built from P2 and P7 with
      the documented reply put in (reliability.md, Mock mode, Input).
      `TestSyntheticScenarios`, mapper to frame at 120x40 and 200x60: Claude
      Code with an API key shows "5h N/A (API key: no subscription window)"
      and the same for 7d, the refusal counts as "errors 1", and the context
      turns "3% of 1M (derived)"; Codex with an API key shows 5h, 7d, plan
      and credits "N/A (API key: ChatGPT sign-in needed)"; Codex without a
      5-hour window shows "5h N/A (plan has no 5-hour limit)" with the 7d
      bar at 1%. No row wider than the screen. Run on the original mock
      files instead, it fails (12 assertions).
    - Not verified: the look; live Windows Terminal rendering; a live
      `get_context_usage` refusal, a live API-key session, a live plan
      without a 5-hour window (simulated on mock above, not live); the
      timing step on mains power.
    - Noted for step 8: zone C's hint "z zoom" names a feature not built.
- Step 8, UI refinement: **done** (2026-10-17); approved by the owner after
  a mock demo of the Markdown styling ("I like it a lot"). Each screen (startup;
  dashboard at 80x24, 120x40, 200x60; inspection; details; approval cards)
  compared with reference-design.md and ui-design.md. Departures found:
  zone C titled "C · AGENTS" at 120x40 (its own width tested, not the
  screen's); "AGENT" for the conversation's label in mock mode; zone D's
  values cut mid-way at 120x40 ("limit N/…") and "$ N/A" at 80x24; B's
  badge without a spinner, "Running" or "Idle"; no spinner on a running
  tool, no moving dot on zone A's running connector; z zoom, the F1 / `?`
  key overlay and Tab between zones advertised but not built; AI text
  drawn plain, without Markdown styling or highlighting. Criteria
  (baseline: `go test ./... -count=1 -v`, 158 top-level tests, 0 FAIL):
  1. Each departure above is removed, with its test, or listed in
     ui-design.md's departures; none left unlisted.
  2. Zone C's title is "C · LIVE AGENTS" on a screen of 100 columns or
     more and "C · AGENTS" below; the conversation's label is "CLAUDE" for
     every `claude-*` mock file and "CODEX" for every `codex-*` one.
  3. Zone D, every mock file at 80x24, 120x40 and 200x60: each item is
     drawn whole or not at all (no item cut, no row wider than the screen);
     what does not fit is counted at the row's end; "cost N/A", never
     "$ N/A".
  4. Zone B's badge, one table test: a spinner glyph then "Streaming"
     while the last entry streams; a spinner then "Running" while the turn
     runs otherwise; "! Waiting for approval"; "Interrupting…"; "Idle" once
     the session is ready with no turn running; the glyph changes with the
     spinner's frame.
  5. While a turn runs, a running tool's row in zone A shows the spinner
     and a dot moves along the running stage's connector, one row per
     frame; idle, frames 0 and 5 of the spinner draw the same screen.
  6. z with A, C or D focused draws that zone over the dashboard's middle
     and bottom; z or Esc gives back the exact frame from before; z in B
     types "z"; Enter on an agent in zoomed C inspects it and Esc returns
     to the zoom.
  7. F1, or `?` outside the editor, opens the key overlay with every row of
     ui-design.md's key tables; it scrolls at 80x24; Esc, F1 or `?` close
     it; `?` in the editor types "?". Tab and Shift+Tab outside the editor
     move to the next and previous zone; in the editor Tab inserts a tab.
  8. AI text styled as Markdown (headings, emphasis, inline code, lists,
     quotes, links, tables, fenced code with its language and highlighted
     keywords, strings, comments and numbers) draws the same characters:
     on a Markdown fixture at three widths, each frame's text equals the
     unstyled frame's; selection and copy tests unchanged;
     `TestStreamToFrame` within its 16 ms target.
  9. Golden frames updated, each diff reviewed; `go build ./...`, `go vet
     ./...` (also `-tags live`, `-tags timing`) exit 0; `gofmt -l .` empty;
     `go test ./...` exit 0 with at least 158 top-level tests.
  Not verified by these checks: the look and feel (the owner's call); the
  256- and 16-colour mapping; synchronized output; live Windows Terminal
  rendering.
  Results (2026-10-17):
  1. PASS. Every departure listed above is removed, each with its test
     (2 to 8 below). Found while building, and listed in ui-design.md's new
     Departures section, none advertised on screen: the approval card's
     Enter (full diff) and s (allow for the session, never offered); no
     status symbol on B's tool lines; the inspection's Messages tab drawn
     plain; a block comment highlighted on its first line only.
  2. PASS. `TestZoneLabels`: every mock file at 80x24, 99x30, 100x30,
     120x40, 200x60; "C · LIVE AGENTS" from 100 columns, "C · AGENTS"
     below; CLAUDE for `claude-*`, CODEX for `codex-*`.
  3. PASS. `TestZoneDWholeItems`: every mock file at the three sizes, no
     item cut, no "…", "+N more" equal to the items not drawn, no "$ N/A".
  4. PASS. `TestTurnBadge`: the eight turn states, the glyph changing
     between spinner frames 0 and 3, and the badge in B's header.
  5. PASS. `TestTreeAnimation`, Claude Code and Codex mock files: the
     spinner on the running tool, one dot moving one connector row per
     frame; frames 0 and 5 identical while waiting for approval and idle.
  6. PASS. `TestZoom`: A, C and D at 80x24, 120x40, 200x60 (title on row
     2, frame to the hints, no other zone's title); z, and z then Esc, give
     back the identical frame, focus kept; F6 to B unzooms; z in B typed;
     Enter in zoomed C inspects and Esc returns the identical zoomed frame.
  7. PASS. `TestKeyOverlay`, `TestKeyOverlayRows`, `TestTabZones`: all 24
     rows of ui-design.md's Anywhere and focused-panel tables in the
     overlay, none cut at 80 columns; it scrolls at 80x24 to "Ctrl+Q", the
     approval card and the editor tables; Esc, F1 and `?` close it with the
     view under it (a held inspection included) restored exactly; `?` in the
     editor typed; Tab and Shift+Tab cycle C, A, D, B outside the editor,
     Tab in the editor inserts a tab.
  8. PASS. `TestMarkdownSameText`, a fixture with every construct and an
     unclosed fence at 80x60, 120x60, 200x80: the styled frame's text equals
     the plain frame's, every cell of zone B hits the same place for
     selection, and 19 constructs carry their style (plain: none);
     `TestMarkdownLines` for fences and block kinds. Selection and copy
     tests unchanged and passing. `TestStreamToFrame`, three runs: p95
     5.7, 2.1 and 2.3 ms (target 16 ms). `TestKeystrokeAllocs` at most 846
     allocations per key (bound 1,200); `TestFrameReuse` passes.
  9. PASS. `TestGoldenFrames -update` changed nothing further; the 15
     frames changed in this step were reviewed: the badges ("Idle",
     "⠋ Streaming"), "C · LIVE AGENTS", CLAUDE and CODEX, the moving dot,
     zone D's whole items with "+N more" and "cost N/A"; no row cut or
     wider. `go build ./...` and `go vet ./...` exit 0, also `-tags live`
     and `-tags timing`; `gofmt -l .` empty; `go test ./... -count=1 -v`
     exit 0, 168 top-level tests (158 + 10), 0 FAIL.
  - Mutation checks, each made its test fail, then restored: zoom not
    cleared by Esc, nor by leaving the panel; the zoomed zone 2 rows short;
    the overlay not restoring the view under it; `?` always opening it; Tab
    taken from the editor; a key row removed; overlay rows wrapped at full
    width; Markdown styling off; markers dropped from styled spans; only a
    line's first row styled; a styled span's offset shifted by one; a
    fence's body plain; inline code unstyled.
  - Timing step (step 7's outstanding gate), run alone on 2026-10-17 on
    **mains power** (charge 32%), CPU load 32% just before, with VS Code,
    Firefox, Teams, Outlook and Notepad open: `TestKeystrokeToFrame` p95
    1.0 ms, max 1.9 ms; `TestDraftKeystrokes` p95 2.5 ms, max 8.0 ms;
    `TestKeystrokeCost` 0.22 and 0.56 ms per key, 845 and 846
    allocations; `TestLongSession` delta-to-frame p95 3.0 ms. PASS.
  - Not verified: the look and feel; the 256- and 16-colour mapping;
    synchronized output; live Windows Terminal rendering of the styles,
    the zoom and the overlay.
  - The owner's judgement (2026-10-17): the step 8 build in Windows
    Terminal, maximised, on a mock demo (the subagent-approval recording
    with a 50-line Markdown answer added, kept outside the repository):
    "I like it a lot". Approved. Committed as `9a27951`, not pushed.
- Done, amendment 4's acceptance criteria (proposal.md, section 4, Done):
  **done** (2026-10-17), awaiting the owner's approval. L2 and L4 ran
  at steps 4 and 5, before steps 6 to 8 changed zones A, B, C and D; they
  are re-run on the current build, within their approved scope (re-runs
  after a change, no new turn, no extra quota), with a check of what
  steps 6 to 8 draw.
  Criteria (baseline: 168 top-level tests):
  1. An acceptance matrix below: each of amendment 4's ten criteria ("any
     of the three providers" read as the two of amendment 5) for Claude
     Code and for Codex, each cell with its evidence (test, live run,
     step) or "not verified" with the reason; no cell blank.
  2. `TestActivityShown` (no tag): `activityShown` returns nil on the
     Claude Code and Codex recordings (`claude-subagent-approval`,
     `codex-subagents`) at 160x60, and an error on each of 8 mutations
     (stages open, only CLI stages, no tool, a silent N/A, a value without
     source, no token value, badge not Idle, no stage drawn).
  3. L2 re-run (Haiku, three short turns, each capped at 2 minutes, any
     failure stops): PASS as before, and after turn 1 `activityShown` nil
     (the owner's last turn a completed stage in zone A with a completed
     tool; every zone D value present with a source or N/A with a reason;
     a token or context value present; badge "Idle") and the frame within
     160x60.
  4. L4 re-run (Codex's default model and its fallback, three short
     turns): the same as 3.
  5. `confcheck` before and after each run: PASS, or NOT-VERIFIED only for
     files the CLI itself writes, as at L1 to L4; never FAIL. Each folder
     holds only `allow.txt` afterwards.
  6. `go build ./...`, `go vet ./...` (also `-tags live`, `-tags timing`)
     exit 0; `gofmt -l .` empty; `go test ./...` exit 0, at least 169
     top-level tests.
  Not verified by these checks: live subagents through the dashboard (no
  approved run spawns one; the evidence is P3b's and P7's live
  recordings, replayed); the real executable started from Windows
  Terminal with a live CLI (the live tests drive the same program in
  process; the owner's call).
  Acceptance matrix (criterion 1; "L2d" and "L4d" are this step's re-runs,
  results below; the same cell holds for both CLIs unless split):
  1. Launched from a terminal. Both: the owner started the executable from
     a terminal at step 1 (item 6) and on mock data at steps 7 and 8;
     step 3's Windows Terminal runs (criterion 9) drove the built program
     maximised and at the default size. A live session started from
     Windows Terminal by the agent: not verified (L2d and L4d run the same
     program in process).
  2. Either CLI selectable. Both: `TestDetect` (neither, either, both),
     `TestStartupGate` (the other CLI stays selectable, `--provider` only
     prefills); detection on this machine at step 1; L2d detected Claude
     Code 2.1.301, L4d Codex 0.145.0.
  3. Project path validated. Both (the check is the CLI's choice's
     companion, independent of it): `TestResolvePath` (44 cases),
     `TestCheckProject`, `TestCheckProjectNotReadable`,
     `TestStartupMessageFrames`, `TestStartupGate`.
  4. The CLI working in that directory. Claude Code: L1 (cwd checked), L2d
     "Ready, cwd" the folder and `allow.txt` written there. Codex: L3
     (cwd in `thread/start`), L4d the same; each folder holds only
     `allow.txt`.
  5. Real prompts, streamed answers. Both: L2d and L4d, three turns each;
     turn 2 streamed and interrupted mid-stream as one answer (Claude Code
     8 ms after Ctrl+C, Codex 9 ms, 8 text deltas); turn 3 answered after
     a crash and Ctrl+R from the resumed session.
  6. Files inspected and changed with approvals. Claude Code: L2d, Glob
     then Write, its approval allowed with F2, a; "allowed by you" in B.
     Codex: L4d, fileChange accepted with F2, a, then a commandExecution
     reading it back (no request: Codex's read-only sandbox runs it); L3
     covers decline and cancel. Approval cards for both CLIs:
     `TestApprovalCards` and step 4's checks.
  7. Multiline input, long pastes, copying. Both, the same code for every
     CLI (step 3, criterion 8: identical results with the Claude Code,
     Codex and mock adapters): `TestInputByteForByte`,
     `TestPasteNeverSubmits`, the keystroke timing step (on mains power at
     step 8), step 3's Windows Terminal runs for paste, Shift+Enter, copy
     and mouse selection. Not verified: Shift-drag selection in Windows
     Terminal while the app captures the mouse; a NUL in a paste (Windows'
     clipboard cannot carry one).
  8. Real agent activity. Both: L2d and L4d, zone A's stages and tools
     from the live events (`activityShown`); Codex's refused first turn
     shown as a failed stage. Subagents: P3b (Claude Code) and P7 (Codex),
     live recordings, replayed by step 6's tests. Not verified: a live
     subagent through the dashboard.
  9. Telemetry accurate or unavailable. Both: step 7's parsing tests on the
     probes' exact lines; L2d and L4d, every zone D value from a named
     source or N/A with its reason, a token value present. Claude Code:
     5h 53%, context 3% of 1M (derived), tokens 99.5k, cost an API-
     equivalent estimate. Codex: 5h 0%, context 6% of 258.4k, tokens 45.5k
     (derived), "cost N/A (Codex reports none)". Not verified: the values
     against each CLI's own usage screen at the same moment; a live
     `get_context_usage` refusal, API-key session, or plan without a
     5-hour window (step 7, simulated on mock).
  10. Both screens, sequence and layout. Both: golden frames of the startup
      screen and the dashboard for each CLI at 80x24, 120x40, 200x60;
      step 8's comparison with reference-design.md; the owner's approvals
      of the look (steps 1, 7 and 8).
  Results (2026-10-17):
  1. PASS. The matrix above: ten criteria, each with its evidence for both
     CLIs; the not-verified parts named in criteria 1, 7, 8 and 9.
  2. PASS. `TestActivityShown`: nil on both recordings at 160x60; each of
     the 8 mutations gives an error. Written first against the last stage,
     it failed on P3b, whose last turn Claude Code opened itself with no
     tool; it now takes the owner's last turn.
  3. PASS. L2d (`acc-l2d-101915`) in 19.61 s on Claude Code 2.1.301, Haiku
     (`claude-haiku-5-5`): turn 1 Write allowed, `allow.txt` "ok", answer
     "done", `activityShown` nil, frame within 160x60; turn 2 interrupted
     8 ms after Ctrl+C; taskkill shown as exit code 1, Ctrl+R resumed the
     same session; turn 3 "LANTERN-73"; Ctrl+Q closed in 1.152 s, exit 0.
  4. PASS. L4d (`acc-l4d-102105`) in 22.20 s on Codex 0.145.0: `gpt-6-sol`
     refused, fallback thread on `gpt-5.6-sol`, message sent again; the
     fileChange accepted, `allow.txt` "ok", answer "done", none declined,
     `activityShown` nil, frame within 160x60; turn 2 interrupted 9 ms
     after Ctrl+C; crash, Ctrl+R resumed the fallback thread; turn 3
     "LANTERN-73"; Ctrl+Q closed in 2.609 s, `app-server` exit 0.
  5. PASS. L2d: NOT-VERIFIED for `~\.claude.json` only ("changed, not
     classified", Claude Code writes it), every other file identical, 1
     new session file. L4d: PASS, 2 new session files. Each folder holds
     only `allow.txt`.
  6. PASS. `go build ./...`, `go vet ./...` (also `-tags live` and
     `-tags timing`), `gofmt -l .` empty, `go test ./... -count=1` ok in
     all 14 packages: 169 top-level tests (168 + `TestActivityShown`).
  - Seen in both live frames: zone D's "time -- session". A test-harness
    artefact, not a defect: `newHarness` pins the app's clock to
    `testNow` (2026-10-15) while the live CLI's events carry the real time,
    so the session seems to start in the future; the application uses
    `time.Now` for both.
- L5, live subagents in Windows Terminal: **done** (2026-10-17), one FAIL
  for the owner to judge (result 5, Claude Code) (approved by the
  owner on 2026-10-17: "Live subagent run ... I want this"; the owner reads
  the CLIs' own usage screens). Scope: one turn per CLI, the built
  executable started by `wt.exe` in a new, maximised window, in a fresh
  folder under `%LOCALAPPDATA%\Temp`; step 3's harness (SendInput, UI
  Automation, the clipboard saved and restored) types and reads. Claude
  Code: P3b's prompt, on Haiku through `ANTHROPIC_MODEL` and
  `CLAUDE_CODE_SUBAGENT_MODEL` set for that process only (no configuration
  file; the application has no model option, amendment 4). Codex: P7's
  prompt, on the application's fallback model. Criteria, for each CLI:
  1. Launch: the startup screen shows the CLI and the folder; Enter gives
     the dashboard and "Ready" within 60 s.
  2. Subagents: every approval card answered with F2, a; within 240 s
     `found.txt` exists in the folder, zone C reads "0 ACTIVE · n DONE"
     with n ≥ 1 (Claude Code) or n ≥ 2 (Codex), B's badge is Idle, every
     row of the frame closed by its border.
  3. Real clock: zone D's session time is not "--" (the L2d and L4d
     "time --" is the test harness's).
  4. Shift-drag with mouse capture on, maximised and at the default size:
     a plain drag copies through the application on release (control);
     a Shift-drag over the same cells leaves the clipboard unchanged on
     release, and Ctrl+C then copies exactly those cells' text.
  5. `confcheck`: PASS, or NOT-VERIFIED only for files the CLI writes;
     the folder holds only `found.txt`.
  6. Zone D against the CLI's own screen (Claude Code `/usage`, Codex
     `/status`): read and judged by the owner, reported as given.
  7. Ctrl+Q closes the application; no CLI process it started remains.
  Not verifiable, stays listed: a NUL in a paste (the Windows clipboard
  ends text at the first NUL); an API-key session (needs a key and API
  billing); a plan without a 5-hour window (another account); Claude Code
  refusing `get_context_usage` (cannot be caused on a real CLI).
  Runs: Claude Code `acc-l5c-103340` (10:33, the full turn) and
  `acc-l5c-105614` (10:56, reopened for criteria 6 and 7 after the owner
  closed Windows Terminal: no subagent prompt, one "Reply with the single
  word ok." so that zone D carries Claude Code's rate-limit report).
  Codex 0.145.0: `acc-l5x-103340` (the harness missed the refusal note,
  wrapped over two rows; no turn ran), `acc-l5x-104828` (the approval
  allowed and `found.txt` written, then the owner closed Windows Terminal),
  `acc-l5x-105458` (the full turn).
  Results (2026-10-17):
  1. PASS, both. Window maximised 1938x1098 px; the startup screen showed
     the folder; "Ready" 2 to 3 s after Enter; 2 CLI processes started; the
     frame 44x156, every row closed by its border.
  2. PASS, both. Claude Code: 29.8 s, 2 approvals allowed (the subagent's
     PowerShell `Get-ChildItem`, then its Write of `found.txt`, both
     "allowed by you"), zone C "0 ACTIVE · 1 DONE" with the row
     "✓ general-purpose … completed 10s 26.7k tok", zone A turn 1 with
     Agent, agent and Glob, B Idle; `found.txt` empty (the folder was empty
     when listed; the answer says so). Codex: `gpt-6-sol` refused, the
     prompt sent again on `gpt-5.6-sol`, 37.8 s, 1 approval allowed
     (`/root/create_found`'s fileChange), zone C "0 ACTIVE · 2 DONE"
     (`/root/create_found` 13s 93.5k tok, `/root/list_files` 11s 46.1k
     tok), B Idle, `found.txt` "found\n". Every row closed by its border.
  3. PASS, both. Claude Code "time 2m 41s session" (48 rows: one Windows
     Terminal Ctrl+minus, as at 44 rows step 8's zone D folds the time into
     "+2 more"); Codex "time 48s session" at 44 rows.
  4. PASS, both, maximised (44x156) and at the default size (30x120): the
     plain drag copied through the application on release ("Use th" /
     "Use t", "Spawn " / "Spawn"); the Shift-drag left the clipboard
     unchanged; Ctrl+C copied exactly "Use the Agent " / "Use the Agen" and
     "Spawn two suba" / "Spawn two su". Claude Code's first attempt
     (10:34) failed both Ctrl+C checks, "se the Agent " for "Use the
     Agent ": the harness dragged from cell centres and Windows Terminal
     snaps to the nearest cell edge (fix below); the rechecks at 10:36:07
     and 10:36:42 passed. The clipboard was saved and restored, never
     printed.
  5. Codex: PASS. `acc-l5x-104828`: every file identical; `acc-l5x-105458`:
     NOT-VERIFIED for `~\.claude.json` only, written by the Claude Code
     dashboard started inside that window (10:56:19). Each folder holds only
     `found.txt`. Claude Code: `acc-l5c-103340` NOT-VERIFIED for
     `~\.claude.json` only (Claude Code writes it), every other file
     identical; **`acc-l5c-105614` FAIL**: `~\.codex\config.toml` changed
     (written 11:16:39), and `~\.claude.json` (11:14:38) and
     `~\.claude\plugins\known_marketplaces.json` (11:00:31) changed, not
     classified; contents not inspected. Attribution from the timestamps
     only: Codex was updated to 0.163.0 at 11:15:09 (its executable
     recreated) and the owner's own Codex 0.163.0 started at 11:16:40 (its
     `app-server-daemon` folder created then), one second after the
     `config.toml` write; the owner's Claude Code `/usage` ran about 11:14.
     The application writes no file; that it caused none of these writes is
     inferred, not verified. The folder holds no file.
  6. Read by the owner, as given. Codex `/status` (0.163.0, about 11:17):
     5h 97% left, resets 15:21; weekly 99% left, resets 10:30 on 22 Oct.
     Zone D (10:56): 5h 3%, resets Sat 15:21; 7d 1%, resets Thu 10:30:
     equal. Claude Code `/usage` (about 11:14): session 64% used, resets
     1:19pm; week 11% used, resets Oct 24, 7:59am. Zone D (10:57, "quota
     updated 12m 36s ago" when the owner looked): 5h 63%, resets Sat 13:20;
     7d 10%, resets Sat 08:00. The percentages differ by one point,
     consistent with the owner's sessions between the two readings; the
     reset times by one minute: Claude Code reports whole minutes (the
     probes' `resetsAt` 1792216800 is 08:00:00) and zone D shows them as
     reported, while `/usage` shows the minute before.
  7. PASS, both. Ctrl+Q closed the window in 5.22 s (Codex) and 0.75 s
     (Claude Code); 2 seconds later none of the CLI processes the
     application started remained (2264, 12032; 17448, 16108).
  - Covered now, listed as not verified in the Done matrix above: a live
    session started from Windows Terminal (criterion 1), Shift-drag with
    mouse capture (7), a live subagent through the dashboard (8), zone D
    against each CLI's own screen (9).
  - Harness fixes during the run (scratchpad, not in the repository):
    Windows Terminal snaps a selection to the nearest cell edge, so a drag
    starts at 0.2 of its first cell and ends at 0.8 of its last; the
    session time read after Ctrl+minus with `time (\S+(?: \S+)?) session`;
    the refusal detected on "Codex refused the model", the one row of the
    note that does not wrap.
  - Seen by the owner: Codex has no slash commands through `app-server`,
    so `/status` typed in the dashboard went to the model as a message
    (its answer "Idle. No active tasks." is the model's, not Codex's
    status); Claude Code ran `/usage` and its text reached B. A file name
    in an answer is not clickable (Windows Terminal links URLs only; editor
    integration is outside amendment 4).

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
