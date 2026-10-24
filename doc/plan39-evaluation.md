# Plan 39 evaluation (2026-10-24)

The check of plan 39, the permission modes selector after the owner's
images (brief amendment 40: the editor row's "Auto" becomes the mode pill
of the third image and opens a "MODES" popup after the first two, the
active CLI's own modes, applied through the CLI and shown in force only
once the CLI reports them; Codex's Full access only after an explicit
confirmation). It covers the 15 criteria written in
[`plan.md`, Plan 39](../plan.md) before the code; one was amended in step
4, said where it applies (39.13). Nothing is committed: the owner reviews
the screenshots and the results first.

How it was checked:
- The probe first (plan.md, What the probe found): P39c on Claude Code
  2.1.303 and P39x on Codex CLI 0.163.0, no model call, each in a new
  temporary project with the configuration check before and after;
  OpenAI's Codex extension for VS Code (`openai.chatgpt` 26.1021.22027)
  read for its modes.
- Deterministic tests, none starting a CLI: the state (TestModeCatalogs,
  TestModesP39c and TestModesP39x through the real mappers,
  TestChooseMode, TestModeRequested, TestChooseModeRefused), the adapters
  (Claude Code's TestSetApprovalMode; Codex's TestSetApprovalMode,
  TestModesBetweenTurns on P39x, TestModeDuringTurn,
  TestModeNextTurnAfterRefusal, TestModeUpdateRefusals, TestModeLimits,
  TestCodexModeOf, TestForbiddenCheck with the presets,
  TestNoForbiddenMethodInSource), the mock (TestMockModes), the rendering
  (TestModesCells, TestModesHighlight, TestModesConfirm, TestModePillRow,
  TestModesRoom, TestModesStripImages, TestModePillImage), the GDI label
  (TestTextMask), the application (TestModesKeys, TestModesMouse,
  TestModesPointerAtRest, TestModesFullAccess, TestModesConfirmPointer,
  TestModesRoomToast, TestModesBesideOthers, TestModesDraw, TestModesMock,
  and plan 12's F4 tests, now through the popup). Each new check was run
  against code broken on purpose (`scratchpad\plan39\mut39.mjs`), 48
  breakages on the final tree: the state (Full access unconfirmed,
  another CLI's mode, a mode left out, the limits not kept, the next turn
  not marked), the adapters (Claude Code's words, an unknown mode written
  by either, Full access's policy, no hold during a turn, no write at its
  end, VS Code's workspace-write Ask or a reviewer read as Manual, no
  requirements read, the policies ignored, a refusal kept asked, the mock
  without Full access), the cells (one cell wider, the heading's gap off
  the ground, the right edge on B's margin, a greyed option lit, the
  selected title's and Full access's colours, the confirmation's order),
  the pill (no compact form, its cells one short), the images (square
  corners, the check on the cursor's option, the selected icon's colour, a
  half-circle pill, no label) and the application (Full access without
  the confirmation, the confirmation keeping the pointer, the keys onto a
  greyed option, the hover never back, a half row sending the cursor back,
  a resting pointer hovering, F4 opening the model selector, a paste or a
  click outside leaving it open, a frame keeping a box it lost, no room
  unchecked, the strips or the pill not drawn, the label one row low,
  "(next turn)" for every Codex change, the mock's answer). 47 failed as
  they must. One is equivalent: the next-message fallback moving the mode
  asked for into the one `turn/start` carries, which `Send` takes either
  way, so no test can tell them apart. One, no hold during a turn, slipped
  through once: TestModeDuringTurn checked the lines' order before the
  fake had logged the interrupt; it now waits for the turn's end and both
  lines, and the breakage failed three runs out of three.
- Windows Terminal 1.23 in mock mode, the build of the final tree
  (`scratchpad\term\acc-p39.exe`, SHA-256 `87b5578d9a206665…`), driven by
  `scratchpad\term\p39demo.ps1`, maximised (156x44) and at 120x40:
  `claude-models.jsonl` (the popup on Manual, Auto chosen, the popup with
  Auto selected, the hover on Plan and on Edit automatically, the pointer
  leaving) and `codex-models.jsonl` (the popup, Approve for me chosen, the
  hover on Full access, its confirmation by a click, the pointer leaving,
  Esc back, Full access turned on, the popup with it selected); the
  colours read back from the crops (`scratchpad\plan39\colours.ps1`,
  `pillcolours.ps1`).
- Live in Windows Terminal, no message sent so no model call: the same
  build on Claude Code 2.1.303 and on Codex CLI 0.163.0 (through the
  test-only logging wrapper), `scratchpad\term\p39live.ps1`, each in a new
  temporary project with the configuration check before and after.
- Build, vet, gofmt, the whole suite, the race detector on the changed
  packages, the timing step alone, all on the final tree; no independent
  review, the criteria being settled by the tests, the mutation checks, the
  screenshots and the live runs.

```text
Evaluation - Plan 39, the permission modes selector after the owner's images - 15 criteria: 15 PASS
PASS 39.1 P39c (23 lines) and P39x (48 lines) recorded and sanitized, in doc/research/probes and testdata/probes: Claude Code's current_permission_mode "default", each of default, acceptEdits, plan and auto answered {"mode": ...} within 3 ms then system/status with its permissionMode (none for the mode in force), an unknown mode refused with the CLI's list; Codex's requirements null, each of VS Code's presets answered {} and notified with the three settings asked for, a change of nothing not notified; the configuration check PASS before and after each
PASS 39.2 TestModeCatalogs: Claude Code's Manual, Edit automatically, Plan, Auto and Codex's Ask for approval, Approve for me, Full access, with the owner's titles and descriptions word for word, broken as the images break them ("Claude will ask for approval|before making each edit", ..., "Unrestricted access to the internet|and any file on your computer"), Full access alone in the warning colour; none for a session of neither CLI; TestModesCells: every title and description line at column 11 and the box's words exactly these
PASS 39.3 TestModesP39c: the mode after each of P39c's events manual, acceptEdits x2, plan x2, auto x3 (the refused word leaving Auto), manual x2, the refusal in zone B in the CLI's words; TestModesP39x: manual (the thread's start), custom (VS Code's workspace-write "Ask for approval"), auto, full, manual, the change of nothing reporting nothing, no mode left out; TestCodexModeOf: 17 combinations (the CLI's spellings, guardian_subagent, P12x's reviewer-only turn/start, Custom otherwise); TestModesBetweenTurns: the mode and the request unchanged until thread/settings/updated, then in force; TestModesKeys: the pill "↯ Auto" kept after the choice until the answer, then "</> Edit automatically"
PASS 39.4 Claude Code's TestSetApprovalMode: after the handshake's three lines exactly {"request":{"subtype":"set_permission_mode","mode":"auto"},"request_id":"req_2",...}, acceptEdits req_3, plan req_4, default req_5, nothing for full, custom, "" or bypassPermissions; Codex's TestModesBetweenTurns: one thread/settings/update each with P39x's params (workspaceWrite/on-request/auto_review; dangerFullAccess/never/user; readOnly/on-request/user), each answered; TestSetApprovalMode (P6b): the update written at once, then the turn/start carrying "approvalPolicy":"on-request","approvalsReviewer":"auto_review","sandboxPolicy":{"type":"workspaceWrite"}; TestModeDuringTurn: nothing during the turn, one update after the interrupt, answered, Auto in force; TestModeNextTurnAfterRefusal: "Codex takes mode changes with the next message: thread/settings/update refused by Codex: Method not found", the request marked (next turn), the next turn/start carrying Auto; TestModeUpdateRefusals: -32601 and -32600 the next-message fallback, -32602 the error and "the mode was not changed: ..."; TestChooseMode and TestF4ClaudeSwitch: a refusal leaves the mode, the pill and the bar, zone B saying why
PASS 39.5 TestChooseMode: Full access unconfirmed asks nothing and requests nothing, confirmed asks it, Claude Code never takes it, left out by the requirements "Full access is not available: Codex's requirements do not allow the approval policy never"; TestModesFullAccess: Enter on Full access opens the confirmation on Cancel, nothing written; Cancel and Esc back to the modes, Esc again closed; a double click on its title lands on a half row, on its description on Cancel; a click outside closes; only Turn on Full access asks [full]; greyed, End skips it, the hover keeps the cursor, a click asks nothing and says why; TestModesConfirmPointer: a pointer at rest where Turn on Full access comes to lie moves nothing, Enter cancels; TestForbiddenCheck: dangerFullAccess caught with on-request, with auto_review, with networkAccess and on thread/fork, allowed only with never and the owner reviewing, and every line the Codex tests write passes it; TestNoForbiddenMethodInSource: "dangerFullAccess" once in session.go, in Full access's preset; TestModeLimits: requirements without danger-full-access or never leave Full access out
PASS 39.6 TestModesCells, both CLIs, 120x40 and 200x60, with Sixel and without: W 51 and 54; "MODES" at column 3 and "Current session" ending at W-3, the heading's words exactly these; a half row before each option and after the last; the bottom edge on the editor's rule, "▾" above the pill's middle, the right edge on the cell after the pill, the top edge below zone B's title; every cell on the box's ground or the band; no other text
PASS 39.7 TestModesHighlight: the selected Auto's rows on #2E2842 from its upper half row ("▄") to its lower one ("▀"), its title #F2EFF6 and description #B09FD5, its check; the cursor's option on the band with its own colours and no check, two lit options sharing "█"; the strips' looks Selected and Cursor; Full access amber unlit, under the cursor and selected; a greyed option never lit; TestModesMouse: the pointer leaving onto the heading, the frame or zone A takes the cursor's highlight away, the selected one's stays
PASS 39.8 TestModesStripImages: opaque strips, the ground around unlit options, the selected band from y 90 to 170 with rounded outer corners and band-coloured inner edges, the icons in their colours at their places (no amber in the shield), the check on the selected title row alone, a greyed icon in the off colour; TestModePillImage: transparent outside the rounded corners, the outline on the edges and on the corner's arc, the fill, the lightning, the label's pixels where the raster puts them, Full access's amber, the compact pill's icon in its middle; TestTextMask: "Auto" 32x20 pixels, its letters on rows 6 to 14, grey anti-aliased, "Approve for me" 112 pixels, its descenders down to row 17; square corners, the check on the cursor's option, the selected icon's colour, a half-circle pill and no label each failed them
PASS 39.9 TestModePillRow: the mode in force's icon and title for nine modes, panel Q, one cell after it, no chevron; with Sixel 7, 9, 18, 7, 17, 17, 15, 13 and 9 blank cells and the frame's ModePill look; the compact pill, then the row alone, as the middle grows; the golden frames: the editor row's right end the pill, the model pill and the rest of the row unchanged (39.13)
PASS 39.10 TestModesKeys: F4 opens on the mode in force; ↓ ↑ Home End within the ends; Enter and Space choose and close; Esc and F4 close; another key and a paste close and go on to the editor; nothing sent, the focus kept; TestModesMouse: a click on the pill opens, a click on an option chooses, on the frame nothing, the wheel nothing, a click outside closes and goes on (zone A focused), the pill twice opens and closes; TestModesPointerAtRest: reports of a pointer at rest move nothing; TestModesBesideOthers: one popup at a time; TestF4WithCardOpen: F4 from two zones, the card still pending
PASS 39.11 TestModesRoom: Claude Code's 21 rows and Codex's 16 at 120x40 and 200x60, Claude Code's not beside a 30-line draft at 120x40 nor at 120x30, neither at 80x24, nothing of the box drawn where it does not fit; TestModesRoomToast: "no room for the modes above the editor" at 80x24, a frame that loses the room closes it, a full view toasts
PASS 39.12 TestMockModes: claude-models.jsonl takes acceptEdits, plan, auto and manual, codex-models.jsonl auto, full and manual, each answered at once, the other CLI's refused, none with no CLI known; TestModesMock: Plan and Approve for me in force at once, the pill showing them
PASS 39.13 (amended in step 4) build, vet, gofmt clean; go test ./... ok, 15 packages with tests, 676 tests; -race on state, codex, claude, mock, render, input and app ok, 0 data races; the golden frames: 81 files, 112 lines, each where the pill, the popup or their consequences are: 69 the mode pill at the editor row's end, 9 the pill as its icon alone beside a long status, 3 its blank cells under the image in the Sixel frames, 25 the footer's "F4 modes", 6 the project bar's mode names (Codex's "approvals Approve for me", no "(next turn)" between turns); the model pill and the rest of the row unchanged; the timing step alone: all eight within target, TestSelectorToFrame p95 1.06 to 1.61 ms, the long session's delta p95 5.46 ms, the stream's p95 4.34 ms (targets 16 ms), the 50,000-line paste 3.00 ms at most (100 ms); 48 mutation checks, 47 caught, one equivalent
PASS 39.14 Windows Terminal on the mocks, maximised (156x44) and 120x40, the final build: Claude Code's popup on Manual and with Auto selected (its check), the hover on Plan and on Edit automatically, the pointer out leaving Auto alone lit; Codex's popup with Approve for me selected, the hover on Full access, the confirmation opened by a click ("TURN ON FULL ACCESS?", Turn on Full access, Cancel on the band), Esc back, Full access turned on and selected, its amber pill; the pills "Auto", "Approve for me", "Full access"; the colours read from the crops against the references': the ground #111518, the frame #B064F4, MODES #B86AF6, Current session #A0AECC, titles #F0F0F2, descriptions #95A3BF, the band #2E2842, the selected title #F2EFF6 and description #B09FD5, Full access's title #F5C673, exact; the icons #ADBDDB, #E873F2, #EDE0FA, #F5C46E, the check #F5F2F7, the pill's fill #21213B, outline #7D4AE0, lightning #E882F5, amber #F3C36E, within 1 to 3 levels of #AEBCDC, #E874F2, #EEE0FA, #F4C46E, #F6F2F8, #21223A, #7E4AE1, #E982F6, #F4C46E (Sixel's palette); the label's brightest pixel #D7D6E3 against #DBD9E6 on "Auto" (10, anti-aliased thin strokes), #DAD8E5 on "Approve for me"
PASS 39.15 Live, the final build, no message sent: Claude Code 2.1.303 (L39.1 no terminal CLI session, Px listening, the version, the snapshot; L39.2 Ready in Manual; L39.3 Edit automatically by ↓ Enter, Plan by a click, Auto by End Enter, Manual by Home Enter, each named on the project bar 264 to 521 ms after the choice, the popup closed, no change left waiting; L39.6 the configuration check PASS); Codex 0.163.0 through the wrapper (L39.2 Ready in Ask for approval, the opt-in, configRequirements/read written once and answered requirements null; L39.3 Approve for me: one update workspaceWrite/on-request/auto_review and its notification, 506 ms; L39.4 the confirmation shown, Esc back, Esc closed, nothing written; Full access by End Enter ↑ Enter: one update dangerFullAccess/never/user and its notification, 933 ms; Ask for approval: readOnly/on-request/user, 487 ms; L39.5 no turn/start, dangerFullAccess written once; L39.6 the configuration check PASS)
```

What the CLIs do against the owner's descriptions, reported and not
changed (amendment 40: "report the discrepancy before changing the
approved wording"; plan.md, What the probe found):
- Claude Code's Edit automatically (`acceptEdits`) approves file edits
  without asking, wherever the edit is; the CLI has no "selected text".
- Claude Code's Manual (`default`) asks before commands too, not only
  before edits.
- Claude Code's Plan presents its plan as an `ExitPlanMode` request the
  owner approves.
- Codex's Ask for approval is the read-only sandbox the dashboard starts
  with (plan 12): Codex asks before any edit, inside the project too, more
  than "edit external files"; VS Code's default for the same label is the
  workspace-write sandbox.

Observed and kept, each said in the documents:
- The box's ground also fills the outer half of its frame's cells (a box
  line sits in the middle of its cell): a halo half a cell wide, `#111518`
  against the terminal's `#0C0C0C`; the references' own outside is
  `#0E1216`. The model selector's frame has no ground; giving none here
  would put the terminal's ground inside the frame line instead.
- Where an image meets the text cells, Windows Terminal's scaling leaves
  a one-pixel column up to 3 levels darker (`#2B283F` beside `#2E2842` on
  the live confirmation).
- The pill's title is larger against its height than the image's label:
  at one row, the image's proportions would be unreadable (D7).
- The pill is wider than plan 29's "↯ Auto": a long status (a turn
  running with a draft) now keeps the pills with the mode pill as its icon
  alone at 120x40, and at 80x24 too, where the status had the row alone.
- A click on a greyed option says why in the editor row's middle; that
  reason is too long to sit beside the pills, so the pill and the popup go
  (plan 29's rule).
- Neither popup fits at 80x24 (Claude Code's 21 rows, Codex's 16): F4
  says so in a toast.

Not verified:
- What Claude Code's Edit automatically, Plan and Auto, and Codex's three
  modes, do in a turn: no message was sent (no model call).
- Claude Code's `bypassPermissions` and `dontAsk` reported in force: a
  session reporting one shows its word in the pill, no option checked
  (TestModeCatalogs, TestModePillRow); none was seen.
- Codex managed requirements on an account that has them
  (`requirements: null` here): the greyed option is checked by tests
  alone.
- A live Codex without `thread/settings/update`: the fallback is checked
  with scripted refusals and plan 38's P34x.
- The pill without Cascadia Mono or Consolas (`PillNoFont`, the text
  pill), and outside Windows: checked by the code paths' tests only.
- Other terminals than Windows Terminal 1.23.

Quota used: none; no message was sent, the CLIs only started.

The owner's terminal-hosted sessions: before the Windows Terminal runs,
the `claude.exe` (PID 2876) of the owner's dashboard started at 11:03 with
`go run` was stopped, under the standing instruction to close
terminal-hosted CLI sessions before a test; that dashboard can resume it
with Ctrl+R. No other was open.

Evidence outside the repository, in the session's scratchpad
(`C:\Users\amolimom\AppData\Local\Temp\claude\C--javadev-projets-agent-command-center\dcc3e64d-0d35-4470-ab9a-d258f8675bae\scratchpad\`):
`plan39\` (the references, the measures, the probes' configuration
snapshots and hashes, `mut39.mjs` and its results `mut39-run1.txt`,
`mut39.txt` and `mut39-final.txt`, the colour scripts and
`colours-final.txt`, the doc scripts), `term\out-p39\` (the final build's
mock runs, with their step plans in `plan39\plan-*.txt`),
`term\out-p39-live-claude\` and `term\out-p39-live-codex\` (the final
build's live runs: results, screenshots, configuration checks),
`live\log-p39\` (the Codex wrapper's log); the earlier build's runs under
`-pre` names, and `term\out-p39-live-codex-run1\` (a harness bug: its one
FAIL, L39.2, was the script indexing a one-line result as an array; the
answer was there).

**The commit and the push (2026-10-24), after the owner's approval.** The
owner approved plan 39 on 2026-10-24; commit 55e2bed (149 files: 131
modified, 18 added) and the approval commit were pushed to `origin`;
`origin/main` equals `main`, the working tree is clean, and the author is
`Evengelius <molimoabel@gmail.com>`, with no attribution line: PASS. The
blocks above stay as they were reported.
