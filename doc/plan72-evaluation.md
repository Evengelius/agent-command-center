# Plan 72 evaluation (2026-10-31)

The check of plan 72, tool commands, approval cards and the activity
timeline (brief amendment 75): zone B names a command by its shell, shows
Codex's command out of its launcher when that is exact, lays long
commands out over lines under their row, folded from four lines; draws an
answered request and Codex's review on the call they belong to; shows a
command's card laid out and coloured with the context its CLI sent; and
gives a command's full view its sections, copied byte for byte, and `y`.
The owner authorized the implementation on 2026-10-31 with C1 kept and C2
and C3 required.

It covers the 22 criteria of [the plan](#the-plan-as-it-ran), written
before the code. When this report was first written nothing was committed
or pushed: the owner reviewed the implementation and did the Human
Evaluation (below) first; the plan closed only on the owner's approval.

**Approved and closed on 2026-10-31.** The owner's Human Evaluation
approved the visual design, usability and interactions, in Windows Terminal
([The owner's Human
Evaluation](#the-owners-human-evaluation-2026-10-31)); 72.22 is PASS on
that approval, and the blocks below stay as they were reported. The
independent review's verdict, APPROVED WITH RESERVATIONS with no blocking
issue, its three accepted reservations and the closure are in [Approval and
closure](#approval-and-closure-2026-10-31). What the sections before that
one say about the plan being open, or nothing being committed, was true
when they were written.

How it was checked:
- **Tests**: the command model, its reading of every recorded Codex
  invocation and its layout (`internal/render/command_test.go`); zone B's
  blocks, decisions, cards, full views, copy, search and colours on the
  approved frames' scenes, built from synthetic events labelled so; the
  row counts through the cache (`command_bench_test.go`); the state's
  associations (`internal/state/associations_test.go`); the keys, `y`, the
  clicks, F12 and the two mock sessions in the application
  (`internal/app/command_test.go`).
- **Each change broken on purpose** through `go test -overlay`, the
  repository never written, its code hashed before and after: 64 mutants
  (`scratchpad\plan72\mutants.mjs`), one at a time.
- **The golden frames**: 25 new, 15 changed, each changed one compared
  with HEAD by a script.
- **Both CLIs' recordings** replayed: P12x, P2, P3b, P6b, P7; two mock
  sessions made from P12x and P2 for the Human Evaluation.
- **Build, vet, gofmt, the whole suite** after the last edit, `-race` on
  the changed packages and **the timing step** alone, on mains power, with
  a comparison against HEAD's tree. No Go build or test ran beside another.
- **Windows Terminal**: the final build on both mock sessions, maximised (208x59) and at 120x40, in ACC's profile: 72 steps of keys and single clicks, each screen read back as text; `y` and double clicks, which write the clipboard, left to the owner (Details).

## The evaluation

```text
Evaluation - Plan 72, tool commands, approval cards and the activity timeline - 22 criteria: 21 PASS, 0 FAIL, 1 not verified yet (72.22, the owner's Human Evaluation)
PASS 72.1 The shell and the command, both CLIs: TestUnwrap, Claude Code's Bash and PowerShell named by their tool and shown as sent; Codex's PowerShell, pwsh, bash, sh, zsh and cmd named by their executable, any other "command", the command out of its launcher; TestCommandRecordings on every recorded Codex invocation (23): each "PowerShell" and its script; TestCommandP8Card: the card "! APPROVAL · orchestrator · PowerShell" over "Add-Content -Path note.txt -Value one", the launcher's path nowhere (plan 8's request was a live run, not recorded: its command on P12x's recorded request's shape); TestCommandMocks: Codex's mock on P12x, every command "PowerShell", no `commandExecution` in zone B
PASS 72.2 Exact or whole: TestShellWords, 23 cases: POSIX's quotes and escapes, Codex's mixed chunks, a line continued; refused, the words read before it kept: a quote not closed, $ or a backtick in double quotes, an unquoted ; | && * ~ # $, a trailing backslash; TestUnwrap, 18 invocations: each launcher's shape, flags before -Command, -c, /c and /C, and whole for -File, a flag's value, words after the script, bash -x, a parse refused; TestCommandRecordings: every recorded invocation read back exactly, all 23 equal byte for byte to Codex's own reading in commandActions (D2's wording: see Details, The design as built)
PASS 72.3 The layout: TestCmdLayout on 69 commands (every recorded Codex script and invocation, every recorded Claude Code command, and commands made to break it: cmd's &, nested and quoted operators, PowerShell's backtick, a 300-character token, tabs and runs of spaces, a here-string, wide and joined emoji, format characters, twelve pipes, the empty command) at 16 widths from 8 to 140, 74, 76, 78, 134, 136 and 140 among them: the rows, in order, give the command back with only blanks and its own line breaks between them; none wider than its width but a lone unit; a break only after a top-level ; (cmd's &), before a top-level && || or |, at a blank, or at the width; the frames' command laid out row for row as the reference frames draw it, and the denied && chain
PASS 72.4 Short and long: TestCommandBlocks, the frames' rows at 74 cells: "✓ PowerShell  git status --short" on its row; the reviewed command under its row, 1 line, "▸" for its cut rationale; the long one's first 2 lines and "… 7 more lines" (N = 9 − 2); at 134 cells the short ones on their rows, the rationale whole, no mark; Claude Code's error row under its row; "▸" only on a block with something hidden; the golden frames commands-*
PASS 72.5 Folded and unfolded: TestCommandBlocks: unfolded, all 9 lines as the reference frame draws them, the rationale whole on two rows, "▾"; TestCommandCounts: each entry's rows counted from the cache equal the rows a fresh frame draws, through its request answered, unfolded, failed, folded again, output after the failure, a review, a longer name in its group, at 74, 76, 134 and 140; TestCommandCacheFrames: 24 commands coming, asked, answered, reviewed, failed, unfolded, scrolled, at 80, 120, 200 and 208 columns: the cached frame the fresh one; TestCommandKeys: the fold kept through the full view and back
PASS 72.6 The keyboard: TestCommandKeys: ↑ from the input to the block, its row's hint "→ unfold · Enter full view · ↑↓ entries · Esc back", the footer's "… Esc back to the input · F1 keys"; → unfolds ("← fold" then), Enter opens the full view, Esc returns, the fold kept; ← folds; ← again leaves the cursor; ↑ stops once per operation (c3, c2, c1: no stop for a request or a review drawn on them); TestCommandCursor (V6)
PASS 72.7 Clicks and double clicks: TestCommandClicks: with the focus in zone A, a click on the folded block's row unfolds it and gives zone B the focus, no cursor, no hint; a click on the unfolded row folds it; a click on "… 7 more lines" unfolds it; a press on a command's line toggles nothing; a double click on a command's line, then on the row, opens the full view, the fold as it was; TestCommandCardClicks: a click on a card's command leaves it pending, folds nothing, answers nothing
PASS 72.8 F12: TestCommandClicks: F12, capture off, the view's mouse mode none; ← folds and → unfolds the block by the keys
PASS 72.9 The card: TestCommandCard at 74 and 134 cells: 7 lines whole; 8 lines, 6 and "… 2 more lines", no seventh; Codex's reason, its folder outside the project as "in", neither when not sent or the project's; Claude Code's description; the grey box from 90 cells; no target row, the command once; the pill; TestApprovalLineKeepsOutcome (its heading's time kept); TestConversationLook 3.6; TestCommandBlocks: no row of the call while its card shows; TestApprovalCards and TestApprovalViewer: F2, a/d/c/s, Enter and Esc as before
PASS 72.10 Decisions and outcomes: TestCommandBlocks: "allowed by you · 11:30:12" completed, "allowed for the session by you · running", "allowed by you · failed", "expired, not answered" not run and "expired, not answered · …" run, the classifier's "approved by auto-review · running" and "denied by auto-review" with no ✓, failure or error, Codex's denied, in progress and timed out; "denied by you" never a failure, whatever the CLI reported after (V5); TestCommandRequestsKept: a subagent's request, one naming no call drawn, an unreadable one keep their rows; TestCommandQuestionKept: a question's row and answers kept; TestToolFailureRow, TestConversationLook 3.7
PASS 72.11 The approval's history: TestCommandView: Codex's long command: "✓ PowerShell  completed · 3s · exit 0", COMMAND, AS SENT, IN, the request's time and who, its reason, "allowed by you, once" and its time, "ran 11:30:11 to 11:30:12", OUTPUT; the reviewed call: its verdict, risk and rationale, and the [UNSTABLE] note; Claude Code's denied call: "✗ Bash  denied", no exit code, its description, "denied by you", "ran  not run"; the classifier with its undocumented field's note; TestCommandRanTimes: the classifier's call ran from its own start to its end (its verdict comes with the result), no run while Codex's review is in progress, the run from the review's approval once approved; the inspection's tests unchanged and passing
PASS 72.12 Codex's review: TestCommandBlocks: approved, denied, in progress, timed out, each on its call's row, the rationale under the command with its authorization from medium up, no ◈ entry; one naming no call drawn keeps its entry; TestCommandReviewOfFile: a file change's review keeps its entry; TestReviewRowsZoneB on P12x's two reviews at 120x40 and 200x60: "PowerShell", "approved by auto-review · ", "◈ risk low: Creates the", no authorization, no entry of their own
PASS 72.13 Copy: TestCommandZoneBCopy: a selection over all of zone B, blocks folded, unfolded and a card in it, copies no command line, the calls by their names, and no command span is selectable (C1); TestCommandCopy at 80, 120 and 200 columns: COMMAND's rows selected copy the script, tabs, double spaces and its own line break kept, no break where it wraps and no indent; AS SENT's the invocation byte for byte; LineText on a row the section's text; a message's view copying its rows as before (C2); TestCommandCopyKey: y copies Codex's invocation byte for byte, "copied 1 line · native"; the native clipboard failing, OSC 52's toast; y in a message's view copies nothing (C3)
PASS 72.14 The search: TestCommandSearch: "Set-Content" found once, the call's target, out of the launcher; "powershell.exe" not found; a match in the folded block's hidden lines unfolds it, one in its first lines does not; its row found once unfolded; the review's rationale searched on its call, its cut part unfolding the block; a card's hidden line lights its "… 3 more lines" row
PASS 72.15 Highlighting: TestCommandColours: every recorded command's tokens rejoin, PowerShell's, bash's and batch's, 120 coloured spans; an unknown lexer plain; TestCommandColourRejoin: a stream with a stray token plain; a frame drawn plain asks for its commands, drawn again with their colours the same rows, each span of a block in its own token's class (the script's colours, not the invocation's), nothing asked again; on 16 colours none; TestCmdColorsStore: asked once, wanted when drawn plain, kept, bounded at 2,048
PASS 72.16 Widths and golden frames: 25 new frames (commands, commands-open, commands-card, commands-view, commands-claude at 80x24, 120x40, 200x60, 208x59 and 79x24, too small), every row within its width, the decision words whole at 74 and 76 cells; 15 frames changed, each against HEAD by scratchpad\plan72\frames.mjs: 2 the classifier's words, 4 a Write's request on its call's row, 9 the Explorer's and the quick open's listing of testdata\mock with the two new mock files; nothing else
PASS 72.17 Both CLIs, the scope kept: P12x (TestReviewRowsZoneB; the Codex mock), P2 (stream-p2 frames; the Claude Code mock), P3b and P7 (their mocks' golden frames, unchanged but for the classifier's words), P6b (TestApprovalCards, TestApprovalViewer, TestEntryViewer); TestCommandMocks: both mocks replayed to the frames' rows, Codex's waiting request answered onto its call's row; internal/provider's code unchanged (its mock's test only, which writes the two mock files); zones A and C, telemetry, the header and the sidebar unchanged in every changed frame, the footer in its hints only
PASS 72.18 Performance: TestCommandMemo: frames drawn again read no input again (the memo's misses unchanged), each command's facts kept, one layout a width, two after a resize; TestCmdColorsStore and TestCommandColoursKept: a command's colours asked for once, made off the draw path; TestCommandRowsShown: 200 commands more add 3.52 allocations a call to a frame drawn again (1,750 then 2,454), as on HEAD's tree (1,839 then 2,542), the command rows' drawing alike at 200 and 400 commands in a memory profile, mutant 64 (every entry built to count its rows) at 31.48 a call failing it; BenchmarkCommandFrames, 200 commands, three rounds in turn with HEAD's tree: medians at 120x40 653 µs on HEAD, 809 µs folded and 693 µs unfolded here, at 200x60 1,184, 1,231 and 1,347 µs, the slowest frame 1.42 ms within the 16 ms budget; TestCommandFrameAllocs outside -race: 1,937 allocations a frame folded, 1,750 unfolded, its bound 2,500; the timing step: 21 of 21 within their targets (72.20)
PASS 72.19 Plan 71 kept: markdown.go, content.go's Markdown, the WrapCache's Markdown memos, their tests and benchmarks unchanged; TestMarkdown* passing; BenchmarkMarkdown* run: the 18 run once each (-benchtime=1x), ok
PASS 72.20 The checks: go build ./... ok; go vet ./... ok (Windows and Linux); gofmt -l . empty; go test ./... ok after the last code edit (19 packages, 16 with tests, 1,196 tests listed, 5 min 14 s); 64 mutants through -overlay, one at a time, the repository never written (the non-test code's hash 3d9e7c58… before the second pass and after it): 62 caught by a failing test, 30 and 31 equivalent (Details), none uncompiled or void at the end; -race (CGO_ENABLED=1, WinLibs gcc) on internal/state (10 s), internal/render (160 s) and internal/app (364 s): ok, no race reported; the timing step alone at 19:13:50 (84 s), on mains power, the CPU 0 to 31 % just before, no other Go process: 21 of 21 within their targets, the search's two hours p95 4.98 ms, its answers ten times longer p95 10.12 ms, the banner p95 1.65 ms (2 ms)
PASS 72.21 The documents: ui-design.md (Zone B's tool rows, Commands, approval cards, runs; Expanding and the full view; Approvals: a command's card, Manual or Auto, after the answer; Format characters; Search; Selection and copy; the key tables: → ← y, TestKeyOverlayRows 106 rows); architecture.md (Commands, approval cards and the activity timeline); events-and-state.md (the state model, Conversation record: a call, its requests and its reviews; TestStateMachines passing); plan.md's progress; this file
NOT VERIFIED 72.22 Windows Terminal and the owner's Human Evaluation: Windows Terminal on the final build (acc-p72.exe, sha256 3ee2da80…, a rebuild of the final tree byte-identical), ACC's profile, both mock sessions, maximised (208x59) and at 120x40, no CLI and no quota: 72 steps, each done and its screen read back, every observed row as the design draws it (Details, Windows Terminal); the owner's Human Evaluation (below) not done yet: it is the owner's, and the criterion waits for it
Not verified: 72.22's Human Evaluation, the owner's (below); in Windows Terminal, `y` with its toast, the copy compared with the mock's command field, a selection's copy and a double click, every one writing the system clipboard (Details, Windows Terminal: the tests cover them, the owner's steps 8 and 9 check them); 80x24 and 200x60 in Windows Terminal (the golden frames cover both); a live session per CLI (none in the criteria without a scope the owner approves); Codex's bash, sh, zsh and cmd launchers in a recording (none recorded: made invocations only, V1); the colours' look by eye, and 16 colours on a screen (tests only)
Verdict: NOT DONE
Likely cause: no FAIL; 72.22 waits for the owner's Human Evaluation
Proposed next step: the owner's Human Evaluation (below) and review of this report and the diff; then the commits, on the owner's approval - waiting for approval
```

The recommendation: **PENDING THE OWNER'S HUMAN EVALUATION**. Every
automated check passes, and the Windows Terminal runs drew every scene as
the design does; what is left is the owner's: the Human Evaluation
below, the review of this report and of the diff, and the commits.

## Manual Human Evaluation

For the owner, in Windows Terminal, no CLI and no quota: two mock sessions
made for it, real recordings with only plan 72's lines put in, labelled
synthetic (`internal/provider/mock/files_test.go`, `codexCommands` and
`claudeCommands`). From the repository, in Windows Terminal maximised
(ACC's tab, about 208x59), then at 120x40:

```text
.\acc --mock testdata\mock\codex-commands.jsonl
.\acc --mock testdata\mock\claude-commands.jsonl
```

Codex's session ends with its third turn waiting for your answer; Claude
Code's has ended. Shift+PgUp scrolls zone B to the earlier turns.

1. **A short command.** In Codex's first turn: `✓ PowerShell  git status
   --short`, one row, the shell's name, no `commandExecution` and no
   launcher path in zone B (zone A's tree, outside this plan, still names
   the CLI's tool, as before).
2. **A long command, collapsed.** Below it, `▸ ✓ PowerShell … allowed by
   you · <time>`, the command's first 2 lines under the row and `… N more
   lines` (2 maximised, 7 at 120x40). The reviewed command above it reads
   `approved by auto-review · <time>` with `◈ risk low: …` under it, cut
   with "…" where it is longer than its row.
3. **Mouse.** A click on the `▸ ✓ PowerShell` row, or on `… N more lines`:
   the block unfolds, `▾`, every line shown; a click on the `▾` row folds
   it. A click on a command's own line changes nothing. A double click on
   any of the block's rows opens its full view; Esc returns, the block as it
   was. With the focus in another zone first (F6), the click works the same and
   puts no cursor in zone B. F12 off: the terminal selects; F12 on again.
4. **Keyboard.** From the empty input, ↑ until the cursor is on the long
   command's row: its hint reads `→ unfold · Enter full view · ↑↓ entries ·
   Esc back`, the footer the same ending `Esc back to the input · F1 keys`.
   → unfolds (`← fold · …`), ← folds, ← again returns to the input. F1's
   overlay lists → ← and `y` under B's entries.
5. **An approval request.** Codex's last turn: the card `! APPROVAL ·
   orchestrator · PowerShell`, its command laid out (from 90 cells in its
   grey box), `reason  Write TODO-COUNT.md…`, `in <HOME>\…` (the request's
   folder is not this project's), `▐ F2 to answer ▌`; no row of its call
   above it. F2, Enter: the request's full view (COMMAND, AS SENT, REASON,
   IN); Esc, then a: the card goes, and the call's row reads `allowed by
   you · running …`.
6. **A completed command and RESULT.** Codex's first turn ends in the
   RESULT box; nothing of the commands is in it. In Claude Code's session:
   `✗ Bash  go test ./...  failed <time>` with `--- FAIL:
   TestMarkdownCacheBytes (0.65s)` under it; the denied update,
   `✗ Bash  git stash && git pull --rebase origin main && go test ./...
   denied by you <time>` on one row maximised, and at 120x40
   `✗ Bash  denied by you <time>` with `git stash` /
   `&& git pull --rebase origin main` / `&& go test ./...` under it; no
   failure and no error either way; the Write's `allowed by you · <time>`;
   RESULT `done`.
7. **The full view.** Enter on the long Codex command (or a double click):
   its head `✓ PowerShell  completed · … · exit 0`, then COMMAND laid out
   for reading, AS SENT verbatim (the launcher and its quoting), IN, the
   APPROVAL rows (requested, reason, decision "allowed by you, once", ran),
   OUTPUT `14 items in 9 files`; the hints start with `y copy the command`.
   Claude Code's denied update: `✗ Bash  denied · …`, no exit code, its
   description, its decision "denied by you", `ran  not run`.
8. **Copy with the new shortcut.** In that view, y: the toast `copied 1
   line · native`.
9. **The copy matches the source.** Paste it into a text editor: it is
   AS SENT's text, Codex's command field byte for byte; compare it with the
   `"command"` of `exec-p72-todo` in `testdata\mock\codex-commands.jsonl`
   (its JSON string, unescaped). Then select COMMAND's rows with the mouse
   and paste: the script itself, on one line, no line break where the view
   wraps it and no indent space; select AS SENT's rows: the invocation
   again.

Then, should you approve one, a short live session per CLI within a scope
you set (doc/evaluation.md, the live runs).

## The owner's Human Evaluation (2026-10-31)

The owner ran the Human Evaluation in Windows Terminal and approved it:
"I have completed my manual evaluation of Plan 72 in Windows Terminal. I
APPROVE the visual design, usability, and interactions." The
implementation meets the owner's expectations, the owner wrote,
including:

- Compact command presentation in Zone B.
- Syntax highlighting.
- Mouse-based expansion and collapse of long commands.
- Full command inspection.
- Copying the original command from the detailed view.
- Approval cards and keyboard controls.
- Consolidated tool activity and approval decisions.
- Correct presentation of Claude Code and Codex mock sessions.

The owner asked that 72.22 be recorded PASS on this explicit approval,
kept apart from the automated coverage. The owner's report gives its
result as a whole: it names no step, terminal size or byte comparison,
and this file claims nothing beyond its words. The nine steps above were
the instructions; which of them were run, and at which sizes, is the
owner's account, not recorded here. No check was run again for it: 72.1
to 72.21 and their evidence stand as reported in the block above.

```text
Evaluation - Plan 72, the owner's Human Evaluation (2026-10-31) - 1 criterion: 1 PASS
PASS 72.22 Windows Terminal and the owner's Human Evaluation: Windows Terminal as reported above (the final build, both mock sessions, 208x59 and 120x40, 72 steps of keys and single clicks, no clipboard written); the owner's Human Evaluation in Windows Terminal, approved by the owner on 2026-10-31: "I APPROVE the visual design, usability, and interactions.", naming the compact commands in zone B, the syntax highlighting, the mouse's unfolding and folding of long commands, the full view, the copy of the original command from it, the approval cards and the keyboard, the tool activity and approval decisions consolidated, and both mock sessions drawn correctly
Not verified: in Windows Terminal, what the owner's report does not name: whether the copy was made with `y` or a selection, `y`'s toast, a selection of COMMAND's rows, the copy compared byte for byte with the mock's command field (step 9), and a double click (the tests cover each: TestCommandCopyKey, TestCommandCopy, TestCommandClicks); 80x24 and 200x60 in Windows Terminal (the golden frames cover both); a live session per CLI (none in the criteria without a scope the owner approves); Codex's bash, sh, zsh and cmd launchers in a recording (none recorded: made invocations only, V1); 16 colours on a screen (tests only)
Verdict: DONE
Likely cause: -
Proposed next step: the owner's independent code review of the implementation; then the owner's decision to close plan 72 and commit - waiting for approval
```

With 72.1 to 72.21 as reported, plan 72 stands at **22 criteria: 22
PASS, 0 FAIL**. The status: **HUMAN EVALUATION APPROVED, INDEPENDENT CODE
REVIEW PENDING**. The approval is of the user experience; the plan stays
open until the owner's independent review of the code and the owner's
decision to close it. Nothing is committed or pushed.

### The owner's clarification (2026-10-31)

With the final approval (Approval and closure, below) the owner clarified
the manual observations. 72.22 stays PASS on the owner's explicit approval,
and the block above stays as it was reported. As the owner wrote:

- Mouse-based expansion and collapse were demonstrated.
- Full command inspection was demonstrated.
- Double-click navigation was demonstrated.
- The `y` copy shortcut and its success toast were demonstrated.
- The copied Codex invocation was pasted and visually inspected.
- The Codex and Claude Code mock sessions were reviewed.

This settles, on the owner's word, these items of the block's Not verified
line: `y`, its toast and a double click. The pasted copy was inspected by
eye: **no automated byte-for-byte clipboard comparison was performed**, and
this file claims none. The byte-for-byte checks are the tests'
(`TestCommandCopyKey`, `TestCommandCopy`): they compare the text the
application gives its clipboard, or a selection would copy, with the
source, on a test's clipboard, never the system clipboard's content.

### Verification boundaries

| Category | What |
| - | - |
| Approved by the owner in Windows Terminal (the owner's report and clarification) | the visual design, usability and interactions; named: the compact commands in zone B, the syntax highlighting, the mouse's unfolding and folding of long commands, the full view, the copy of the original command from it, the approval cards and the keyboard, the tool activity and approval decisions consolidated, both mock sessions drawn correctly; demonstrated: the mouse's expansion and collapse, the full view, a double click's navigation, `y` and its success toast, the copied Codex invocation pasted and inspected by eye, both mock sessions reviewed |
| Verified by the automated tests (72.1 to 72.21) | the shells and the exact reading of the 23 recorded Codex invocations; the layout of 69 commands at 16 widths; the blocks, folds, keys, clicks, double clicks and F12; the cards; the decisions and the approval's history; Codex's reviews; the copy byte for byte and `y` with its toasts; the search; the colours; 25 new and 15 changed golden frames, 79x24 to 208x59; both CLIs' recordings; the memos and allocations; 64 mutants; `-race`; the timing step |
| Verified by the Windows Terminal harness (Details, Windows Terminal) | the final build on both mock sessions, at 208x59 and 120x40: 72 steps of keys and single clicks, each screen read back as text; no clipboard written |
| Reviewed independently (as the owner reported it) | the implementation, by ChatGPT acting as the owner's independent reviewer: APPROVED WITH RESERVATIONS, no blocking issue (Approval and closure) |
| Not verified | in Windows Terminal, a selection of COMMAND's rows and its paste (`TestCommandCopy` covers it) and the system clipboard's content compared byte for byte with the mock's command field; 80x24 and 200x60 in Windows Terminal (the golden frames cover both; the owner's report states no size); a live session per CLI (none in the criteria without a scope the owner approves); Codex's bash, sh, zsh and cmd launchers in a recording (none recorded: made invocations only, V1); 16 colours on a screen (tests only) |

## Approval and closure (2026-10-31)

**Approved.** After the final review of plan 72 with ChatGPT acting as the
owner's independent reviewer, the owner wrote: "I APPROVE Plan 72 and
explicitly authorize its closure, commit, and push to `origin/main`.", and
"I accept the implementation as it stands. No additional code changes are
required." **Plan 72: CLOSED.**

**The independent review's verdict: APPROVED WITH RESERVATIONS — No
blocking issues identified.** The reviewer's own report is not in this
repository: the verdict, the reservations and what the reviewer examined
are recorded as the owner reported them. The review worked from a package
of this plan's working tree against its baseline 4bd297b, kept outside the
repository.

The results stand as they were reported: **22 criteria, 22 PASS, 0 FAIL**:
72.1 to 72.21 on the automated checks (The evaluation), 72.22 on the
owner's explicit approval (The owner's Human Evaluation), each block word
for word, the first still showing 72.22 NOT VERIFIED as it was first
reported.

### The three reservations, accepted and not blocking

**R1 — Rendering overhead.** The new command presentation introduces a
measurable but acceptable rendering and allocation overhead compared with
HEAD. The measurements stand (Details, Performance):
`BenchmarkCommandFrames`, 200 commands, the medians of three rounds:

| Size | HEAD | Plan 72 folded | Plan 72 unfolded |
| - | - | - | - |
| 120x40 | 653 µs, 1,883 allocations | 809 µs, 1,984 | 693 µs, 1,793 |
| 200x60 | 1,184 µs, 2,698 | 1,231 µs, 2,971 | 1,347 µs, 2,735 |

The slowest frame read 1.42 ms: the measured frame times remain comfortably
below the applicable 16 ms performance target. **Decision: ACCEPTED.** No
optimization requested.

**R2 — Out-of-order review notifications.** The independent review
identified a possible edge case: if a Codex review notification arrives
before its corresponding tool call, the notification remains visible but is
not automatically associated with the subsequently created call. The
available recordings demonstrate the normal ordering, where the tool call
arrives first (P12x; Details, V4). This is a potential robustness
limitation, not a reproduced defect. **Decision: ACCEPTED FOR PLAN 72.** The
implementation stays as it is, and no plan follows from it.

What the code does in that order, read and run on this tree when this was
recorded, on synthetic events, as no recording holds this order (one scratch
test through `go test -overlay`, in no file of the repository):

- The review comes first, its call not known: its `◈` entry is drawn, as
  the reservation says.
- Then the call starts. The state does not look back for a review (a
  request is looked for at `ToolStarted`, S1; a review sets
  `ToolCall.Review` only when its event finds the call, `review` in
  `internal/state/reduce.go`), and zone B stops drawing the entry, which now
  names a command call drawn there (`attached`,
  `internal/render/block.go`). So "remains visible" holds until the call
  starts: from then on zone B shows no verdict, neither as an entry nor on
  the call's row, and the call's full view has none; the entry stays in the
  state's record.
- A later update of the same review, once the call is known (started before
  the call, completed after it), is joined to the call as usual.

Should a recording ever show this order, this is where to start;
`doc/events-and-state.md`, A call, its requests and its reviews, points
here.

**R3 — Evidence traceability.** Some reported measurements lack retained
detailed logs, and the final race-detector run preceded two test-only
modifications (The checks: one `t.Logf` in `TestCmdLayout`, and
`TestCommandRowsShown`, which skips under `-race`). The evidence is tied to
the final tree by file times, not by a hash of the tree: the last edit of
code or tests at 18:52:02, the whole suite from 19:08 to 19:13:20, the
timing step from 19:13:50 to 19:15:21, the binary built at 18:52:58 and a
rebuild at 19:15:41 identical. The logs stay outside the repository, in the
session's scratch folder. These limitations stand as they are. **Decision:
ACCEPTED.** For future development, retaining stronger evidence-to-code
traceability is recommended; no new testing campaign is requested for plan
72.

### Mutation testing

As reported (Details, The mutants, The checks): **64 mutants evaluated, 62
detected, 2 surviving (30 and 31)**, assessed as equivalent under the current
reducer invariants. The independent reviewer examined the relevant code and
found the equivalence explanation coherent. The two stay in the results as
survivors, neither counted as detected nor left out.

### The closure

As the owner asked, no implementation changed for the closure, and no full
suite, race run, mutant pass, benchmark or manual test was repeated: the
evidence above was reviewed and accepted as it is. The checks are Git's and
the documents': `gofmt`, `go build` and `go vet`, and, as AGENTS.md and
events-and-state.md changed, the two tests that read them, alone
(`TestAgentsMDWithinCodexLimit`, `TestStateMachines`). The commit holds what
the reviewer received, byte for byte, but for the closure's documents: this
file, plan.md, AGENTS.md's status and events-and-state.md's note of R2.
`git diff --cached --check` reports only the 5 new 79x24 frames, "new blank
line at EOF": the too-small screen padded to 24 rows, as in the 70 frames of
that size before them; `TestGoldenFrames` needs the full height.

plan.md's section of plan 72, its design, criteria, steps and progress, is
kept word for word under The plan as it ran, as the plan's one commit holds
it, plan.md keeping the current state only. The commit, "Plan 72: Readable
tool commands and approvals in Zone B", holds the whole plan, its code,
tests, golden frames, mock recordings and documents; it is pushed to
`origin/main` after the final Git checks. Its hash is git's: a commit cannot
hold its own.

## Details

### The design as built, and what changed on the way

The design is the approved one (The plan as it ran: Design, Keys and the
mouse, Copy and search); where its words and the code differ, here is how
and why.

- **D2's exactness.** The design says an unwrap is kept "only when quoting
  it again gives the invocation back byte for byte". Codex quotes its argv
  its own way, single and double quotes in turn within one word
  (`'$p = … '"'found.txt'; "'$exists …'`, P7), which no quoting of mine
  gives back. What is kept instead is the same guarantee read the other way:
  the command field split into its words by POSIX's rules, with nothing
  left over and nothing a shell would expand (`$` or a backtick in double
  quotes, an unquoted `;`, `|`, `&`, `*`, `~`, `#` and the like refuse it);
  those words are exactly the argv the shell receives. Every recorded
  invocation reads so, and where Codex's own `commandActions` read the
  script, it is the same script byte for byte (V1). Any other invocation is
  shown whole, as the design says.
- **Runs of calls.** A request drawn on its call's row, and a review drawn
  under its call, still end a run of calls, as their own rows did: runs
  fold exactly as before plan 72. Letting them join runs folded calls that
  had always shown (a test of plan 5's caught it).
- **Questions** keep their request's row, their answers under it, as
  before: merging them would hide the answers. A question's call row is
  left out while its card shows, as any request's.
- **Codex's reviews** are joined to a command's call only, the calls that
  draw a block; a review of another item keeps its `◈` entry.
- **A call waiting on a card**: any pending request naming it hides its
  row, read from the request queue once a frame (not a lookup a call), so
  that a long session's frames cost what they did (The timing step).
- **The history's label column** is 15 cells, for "description"; the
  reference frame's 13 had no description row.
- **The full view of a request** (Enter on a card) shows the same COMMAND
  and AS SENT sections and the same `y`, then what the CLI sent with the
  request: C2 and C3 apply to every command's full view.
- **"ran"** reads from the call's start, or the owner's answer or Codex's
  approval when later, to its end. Claude Code's classifier gives its
  verdict on the result's line (`tool_result_meta`), after the run, so its
  time is never the start. The first build took it as one, and drew a
  run while Codex's review was still in progress; both found on review
  and fixed, with `TestCommandRanTimes` and mutants 60 to 63. "not run"
  once denied; nothing while a request waits, while Codex's review is in
  progress (the command not known to run yet), or for an expired request
  whose call never ran.
- **The colours** are asked for as each call or request comes, in Update,
  and for any command a frame drew plain; the app's tests make them on the
  goroutine outside their bubbles, as plan 17's highlighting is, since
  Chroma's regexp2 starts a clock goroutine on first use (a bubble left
  with it panics; `internal/app/main_test.go`).

### What was verified on the way (V1 to V8)

- **V1** The recordings hold 23 distinct Codex invocations
  (`testdata/probes`, `doc/research/probes`): all PowerShell's `-Command`
  forms, the script single-quoted, double-quoted or in mixed chunks; each
  one read back exactly, its last word the script; all 23 of them carry
  Codex's own reading of the script in `commandActions`, every one equal
  byte for byte (`TestCommandRecordings`). No recording holds bash, sh,
  zsh or cmd: those forms are checked on made invocations (`TestUnwrap`).
- **V2** Chroma's PowerShell lexer on every recorded Codex script, its
  bash lexer on every recorded Claude Code command, batch on a cmd line:
  every token stream rejoins its text byte for byte; 120 spans in
  a class other than the text's (`TestCommandColours`).
- **V3** The tokens are made in a `tea.Cmd` (`colourCommands`), never in
  View; the first tokenisation of each lexer: PowerShell's 1.0 to 3.0 ms, bash's 1.0 to 1.5 ms, batch's 5.0 to 8.5 ms, in three runs of `TestCommandColours`.
- **V4** Codex's review: `item/autoApprovalReview/started`, then
  `completed`, both naming `targetItemId`, after the item's start (P12x);
  the in-progress status is shown "auto-review in progress"; its risk and
  rationale are searched as the call's (`TestCommandSearch`).
- **V5** After a denial Claude Code reports the call failed ("The user
  doesn't want to proceed…"): the state keeps the denial (the tool call
  machine), and zone B shows `✗ … denied by you <time>`, no failure, no
  error (`TestCommandBlocks`, `TestCommandMocks`).
- **V6** At 74, 76, 134 and 140 cells, folded and unfolded, the cursor on
  a block shows its hint on the block's row and leaves zone B's rows as
  many (`TestCommandCursor`).
- **V7** The golden frames (below).
- **V8** Zone B draws the orchestrator's calls only: a subagent's call has
  no entry, its request keeps its row (`TestCommandRequestsKept`).

### The golden frames (V7)

`TestGoldenFrames` and the plan's other frame tests rewrote 15 frames and
wrote 25 new ones. `scratchpad\plan72\frames.mjs` compared each changed
frame with HEAD's and put it in a class by what changed:

| Class | Frames |
| - | - |
| Claude Code's classifier: "approved by auto-review · <end>" (D5), its target one cell shorter | approvals-auto at 120x40 and 200x60 |
| a Write's request on its call's row (D5): one row, "allowed by you · 02:00:09", the path cut from its left, the rows under it one up | stream-p2-01 to 04 |
| the Explorer's and the quick open's listing of `testdata\mock`, gaining the two mock files ("16 files", the git markers one row down) | explorer, explorer-nerd and file-finder at 80x24, 120x40 and 200x60 |

Nothing else changed. The new frames are the reference frames' scenes on
the dashboard, from synthetic events labelled so (`commandEvents`, in
`internal/app/command_test.go`): `commands` (folded), `commands-open`,
`commands-card`, `commands-view` (the long command's full view) and
`commands-claude` (the failure and the denial), each at 80x24, 120x40,
200x60, 208x59 and 79x24 (too small); read by eye at each size against
the reference frames.

### The mutants

Each mutant through `go test -overlay`, the package's tests named by its row's part (`scratchpad\plan72\mutants.mjs`). The first run caught 42 of the first 59; of the others, four did not compile or matched no code and were rewritten, one broke dead code, which was removed, its mutant replaced; each survivor led to a new or stronger test, except 30 and 31, equivalent (The checks). Mutants 60 to 64 came with the "ran" fix and `TestCommandRowsShown`. A pass number says the pass that caught a mutant first run in an earlier one.

| Mutant | What it breaks | Verdict |
| - | - | - |
| 1 | shellWords: a tab no blank | caught by TestShellWords |
| 2 | shellWords: $ in double quotes accepted | caught by TestShellWords |
| 3 | shellWords: ; \| & taken as a word | caught by TestShellWords |
| 4 | unwrap: any last word a script | caught by TestUnwrap |
| 5 | unwrap: a flag's value taken | caught by TestUnwrap |
| 6 | unwrap: bash with more words | caught by TestUnwrap |
| 7 | exeShell: pwsh unknown | caught by TestUnwrap |
| 8 | commandOf: the invocation shown though unwrapped | caught by TestUnwrap |
| 9 | lexerFor: cmd as bash | caught by TestUnwrap |
| 10 | layout: no break after ; | caught by TestCmdLayout |
| 11 | layout: && rows not indented | caught by TestCmdLayout |
| 12 | layout: no break before \| | caught by TestCmdLayout |
| 13 | layout: brackets not nested | caught by TestCmdLayout |
| 14 | layout: a wrap not indented | caught by TestCmdLayout |
| 15 | layout: a top-level blank not first | caught by TestCmdLayout (pass 2) |
| 16 | layout: indents past the width | caught by TestCmdLayout |
| 17 | colours: a token not rejoined kept | caught by TestCommandColourRejoin (pass 3) |
| 18 | colours: ok only with spans | caught by TestCommandColours |
| 19 | store: asked twice | caught by TestCmdColorsStore |
| 20 | store: unbounded | caught by TestCmdColorsStore |
| 21 | decision: the session's words | caught by TestCommandBlocks |
| 22 | decision: a denial shown as run | caught by TestCommandBlocks |
| 23 | merges: a question merged | caught by TestCommandQuestionKept (pass 2) |
| 24 | block: a denial's error shown | caught by TestCommandBlocks |
| 25 | block: the outcome's room forgotten | caught by TestCommandStaysPut (pass 2) |
| 26 | block: 4 lines not folded | caught by TestCommandCursor |
| 27 | counter: every line counted | caught by TestCommandBlocks (pass 2) |
| 28 | review: the authorization at low risk | caught by TestCommandBlocks |
| 29 | attached: any call's review | caught by TestCommandReviewOfFile (pass 2) |
| 30 | callEntry: a subagent's call | survived: equivalent (Details) |
| 31 | elsewhere: a subagent's request merged | survived: equivalent (Details) |
| 32 | waiting: answering not waiting | caught by TestCommandAnswering (pass 2) |
| 33 | count key: the fold forgotten | caught by TestCommandCounts (pass 2) |
| 34 | count key: the failure's output forgotten | caught by TestCommandCounts (pass 3) |
| 35 | cwd: the project's shown | caught by TestCommandCard (pass 2) |
| 36 | want: the invocation coloured | caught by TestCommandColours (pass 3) |
| 37 | callRow: a denial's ✓ | caught by TestCommandDeniedSymbol (pass 2) |
| 38 | callOutcome: no decision | caught by TestCommandBlocks |
| 39 | exchange: a card with no label | caught by TestCommandCard |
| 40 | looks: the first entry's fold | caught by TestCommandFirstEntry (pass 2) |
| 41 | card: 7 lines folded | caught by TestCommandCard |
| 42 | card: the command as its description | caught by TestCommandCardNoRepeat (pass 2) |
| 43 | view: a denial's "not run" | caught by TestCommandView |
| 44 | view: AS SENT for the wrong commands | caught by TestCommandView |
| 45 | view: AS SENT's blank kept on its row | caught by TestCommandAsSentRows (pass 2) |
| 46 | copy: a byte past the end | caught by TestCommandCopyPart (pass 2) |
| 47 | LineText: a row's text | caught by TestCommandCopy |
| 48 | search: the launcher searched | caught by TestCommandSearch |
| 49 | search: merged entries twice | caught by TestCommandSearch |
| 50 | search: a hidden line not unfolded | caught by TestCommandSearch |
| 51 | lighting: the review's matches on the command | caught by TestCommandSearchLit (pass 2) |
| 52 | S1: a request before its call | caught by TestCallRequests |
| 53 | S1: a request after its call | caught by TestCallRequests |
| 54 | S2: the authorization lost | caught by TestCallReview |
| 55 | S3: the denial's time lost | caught by TestCallReview |
| 56 | → unfolds no block | caught by TestCommandKeys |
| 57 | ← folds no block | caught by TestCommandKeys |
| 58 | y copies a byte more | caught by TestCommandCopyKey (pass 2) |
| 59 | colours never kept | caught by TestCommandColoursKept (pass 2) |
| 60 | ran: from the classifier's verdict | caught by TestCommandRanTimes |
| 61 | ran: while the review is in progress | caught by TestCommandRanTimes |
| 62 | ran: not from the owner's answer | caught by TestCommandView |
| 63 | ran: not from the review's approval | caught by TestCommandRanTimes (pass 4) |
| 64 | rows: every entry built to count its rows | caught by TestCommandFrameAllocs |

### Performance (P1 to P5, 72.18)

- **The memos**: `TestCommandMemo`: frames drawn again read no input
  again (`Memo.misses` unchanged), keep each command's facts, one layout a
  width (two after a resize); the colours asked for once a command
  (`TestCmdColorsStore`, `TestCommandColoursKept`), made in a `tea.Cmd`,
  never while drawing; the first tokenisation of each lexer, in three
  runs of `TestCommandColours`: PowerShell's 1.0 to 3.0 ms, bash's 1.0 to
  1.5 ms, batch's 5.0 to 8.5 ms.
- **Rows drawn only when shown**: `TestCommandRowsShown`, outside `-race`:
  200 commands more add 3.52 allocations a call to a frame drawn again
  (1,750 then 2,454, unfolded); the same session on HEAD's tree adds 3.52
  too (1,839 then 2,542, a scratch copy of the test there). A memory
  profile of each says where: zone A's workflow tree builds a row for
  every call each frame (`itemRows`, `treeRow.lead`: 2 a call), and zone
  B's runs grow a slice a run (`lookAll`, look.go:126), both as before
  plan 72; the drawing of command rows (`cmdRow`, `cmdSpans`, `callRow`,
  `toolRow`) allocates alike at 200 and 400 commands. Mutant 64, every
  entry built each frame to count its rows, read 31.48 a call and failed
  the test. The growth with the session's calls is HEAD's, outside this
  plan's parts; it is noted here, not changed.
- **The frame**: `BenchmarkCommandFrames`, 200 Codex commands in one turn,
  half of them long, half the long ones reviewed, one in ten failed, drawn
  again to its string through one WrapCache, one Memo and the colours
  made, 200 times each; three rounds, this tree and HEAD's in turn, one at
  a time (HEAD's session a scratch benchmark in `scratchpad\plan72\oldtree`);
  the medians, the range in brackets:

  | Size | HEAD | Plan 72 folded | Plan 72 unfolded |
  | - | - | - | - |
  | 120x40 | 653 µs (527 to 682), 1,883 allocations | 809 µs (605 to 892), 1,984 | 693 µs (674 to 732), 1,793 |
  | 200x60 | 1,184 µs (1,032 to 1,209), 2,698 | 1,231 µs (1,160 to 1,274), 2,971 | 1,347 µs (1,129 to 1,417), 2,735 |

  Zone B draws the same calls as blocks, their colours as more spans; the
  slowest frame 1.42 ms, eleven times within the 16 ms budget. The
  allocations do not depend on the machine's load; the times swing
  between rounds by a fifth on either tree.
- **The guard**: `TestCommandFrameAllocs`, outside `-race`: the
  200-command frame drawn again allocates 1,937 times folded, 1,750
  unfolded, its bound 2,500.
- **The cache**: `TestCommandCacheFrames`: the cached frame the fresh one
  through 24 commands coming, asked, answered, reviewed, failed, unfolded,
  scrolled and resized; `TestCommandCounts`: each entry's rows from the
  cache the rows a fresh frame draws, each step from the width the step
  before ended at, output after a failure among them.
- **The search**: the first timing step on the way read the search's
  two-hours case slower than plan 71's evidence. Timed on HEAD's tree and
  this one in turn, three rounds, the key in the bar read p50 3.0 to 4.6
  ms on HEAD and 3.7 to 5.7 ms here: zone B's passes looked each call up
  two to four times more a frame (its card, its name column, its row
  count's key) and the search built each call's lines. Those runs, before
  the fix, also missed targets: the first timing step 20 of 21,
  `TestBannerToFrame` p95 2.14 ms over its 2 ms; in the comparison,
  `TestBannerToFrame` once on each tree (p95 2.98 ms on HEAD's, 4.00 ms
  here) and the search's answers ten times longer once here (p95 18.51 ms
  over 16 ms) (`scratchpad\plan72\timing1.txt`, `alternate1.txt`).
  Fixed: a call's name and kind read from its entry before any lookup,
  the calls a card hides read from the request queue once a pass, the
  search's parts in place; then p50 3.05, 3.07, 3.06 ms on HEAD and 5.18,
  4.51, 3.15 ms here, and a CPU profile of each tree's run found the same
  work (7.88 s and 7.77 s of samples, the matcher 3.43 s and 3.30 s). The
  final timing step: p50 3.02 ms, p95 4.98 ms (The checks).

### The checks (72.20)

All on this machine, one Go command at a time, none beside another:

- `go build ./...` ok; `go vet ./...` ok on Windows and with `GOOS=linux`;
  `gofmt -l .` empty.
- `go test ./...` after the last edit of code or of a document a test
  reads (ui-design.md, events-and-state.md): ok, 19 packages, 16 with tests, 1,196 tests listed by `go test -list`, 5 min 14 s.
- `-race`, `CGO_ENABLED=1` with WinLibs' gcc: `internal/state` ok
  (10 s), `internal/render` ok (160 s), `internal/app` ok (364 s); no race
  reported. The tree as tested, but for one `t.Logf` in `TestCmdLayout`
  and `TestCommandRowsShown`, both added after (a log line; a test that
  skips under `-race`).
- **Mutants**: 64, through `go test -overlay`, one at a time
  (`scratchpad\plan72\mutants.mjs`, its `check` mode: each mutant's code
  in the tree once, before and after). Five passes: the first 59 mutants;
  then those that did not compile or matched no code rewritten, and the
  tests a survivor showed weak made stronger; then the "ran" row's four
  after its fix, and mutant 64 with `TestCommandRowsShown`. 62 caught by a
  failing test, 2 survived as equivalent (30, 31: below), none left
  uncompiled or void. The non-test code's hash
  (its diff against HEAD and the new files, sha256) 3d9e7c5881d20c9b
  before the second pass and the same after it, the "ran" fix taken out
  in a scratch copy for the comparison: the repository never written by
  a mutant.
- **Mutants 30 and 31 are equivalent**: `callEntry`'s `tc.Agent != ""` and
  `elsewhere`'s `ap.Agent == ""` guard a case the reducer never makes. A
  "tool" entry is added only for the orchestrator's call
  (`internal/state/reduce.go`, ToolStarted: `ev.Agent == ""`), and
  `callEntry` finds a call by its own entry, so a subagent's call has
  none with or without the guard, and its request then never merges.
  They stay as early exits; no test can tell them apart.
- **The timing step**, alone, the exact command of `doc/evaluation.md`,
  Keystroke timing: at 19:13:50, 84 s, on mains power (battery 98 %), the CPU at 0, 0, 0, 0 and 31 % in five samples just before, no other Go process; the applications with a window: Chrome, Edge, Outlook, Notepad, Paint, Settings, Windows Terminal, the VPN client, Move Mouse and the Claude desktop app, idle; 21 of 21 within their targets: the search's two hours p50 3.02 ms, p95 4.98 ms, its answers ten times longer p95 10.12 ms, the banner p95 1.65 ms (2 ms), a key in the 10,000-line draft p95 1.10 ms, 2,400 calls' events p95 2.30 ms, the long session p95 2.63 ms.

### Windows Terminal (72.22)

The final build (`acc-p72.exe`, sha256 3ee2da80…, a rebuild of the final
tree byte-identical) on both mock sessions, in ACC's profile, maximised
(208x59) and at 120x40 (`wt --size 90,30` at the profile's 9 pt), through
plan 51's harness (`scratchpad\term\p70b.ps1`): a temporary home and app
data, a PATH of System32 alone, no CLI started, no quota; each step's
screen read back as text through UI Automation, with a screenshot
(`scratchpad\term\out-p72`, the steps in `scratchpad\plan72\wtplans`).
72 steps, every one done. Two exploratory runs found the entries' stops
first; a first full run stopped at its twelfth step, on a click target of
mine that named a time (the mock replays on today's clock, and the
harness splits a step at ":"), and was run again whole.

**The antivirus.** My first script for this check saved the owner's
clipboard, sent keys and captured the screen; the antivirus blocked it as
"malicious content" when it started and removed it, before it did
anything. Plan 51 met the same and rebuilt its harness without clipboard
code; the runs used that harness, and no step writes the clipboard: no
`y` and no double click, which copies on release. The owner's clipboard
was never touched; `y`, the copy's comparison and the double click are
the owner's steps 8 and 9 (and `TestCommandCopyKey`, `TestCommandCopy`,
`TestCommandClicks` in the tests).

**Codex's mock**, 28 steps at each size:
- The replay's end: turn 3's card, "! APPROVAL · orchestrator ·
  PowerShell", the command laid out (in its grey box maximised; seven
  lines whole and no box at 120x40, zone B under 90 cells), "reason",
  "in <HOME>\AppData\Local\Temp\acc-p12\x" (the mock's folder, not the
  project), "▐ F2 to answer ▌", no row of its call above it.
- ↑ to the first entry, then ↓ three times: the thought, `✓ PowerShell
  git status --short` on its row, the reviewed command (two lines and its
  risk row cut with "…"), the long command: `▸ ✓ PowerShell`, its first two
  lines and "… 2 more lines" maximised, "… 7 more lines" at 120x40; the
  row's hint "→ unfold · Enter full view · ↑↓ entries · Esc back", the
  footer "… Esc back to the input · F1 keys".
- → unfolds (`▾`, four lines maximised, the reference frame's nine at
  120x40, "← fold" in the hint); ← folds; Enter opens the full view:
  "✓ PowerShell  completed · 1s · exit 0", COMMAND laid out, AS SENT
  verbatim over three rows (four at 120x40), IN, APPROVAL (requested by
  the orchestrator, the reason, "allowed by you, once", "ran" from the
  answer to the end), OUTPUT "14 items in 9 files", the hints starting
  "y copy the command"; Esc back, the block as it was, the cursor on it.
- Clicks: on "… N more lines", the block unfolds, no cursor, the input's
  footer; on its `▾` row, it folds. F6 to the next zone, then the same
  click: it unfolds and the focus is zone B's input again; a click on the
  `▾` row folds it back.
- F1: the rows "→ on a folded command, unfold it", "← on an unfolded
  command, fold it", "y in a command's full view, copy the command as sent,
  byte for byte".
- Ctrl+F, "Unique Path" (the long command's last line, hidden while
  folded): "match 1 of 1", the block unfolded on the match; Tab, the full
  view; Esc, Esc: the bar closed.
- F2: the card's keys row "▸ a allow once · d deny · c cancel the turn ·
  Enter full view · Esc later · ↑ ↓ other approvals"; Enter: the
  request's full view, "! APPROVAL  orchestrator · PowerShell", COMMAND,
  AS SENT, REASON, IN, its keys "a allow once · d deny · c cancel the turn
  · y copy the command · …"; Esc; a: the card gone, the call's row
  "allowed by you · running 44s", its command under it (two lines and
  "… 2 more lines" maximised, "… 4 more lines" at 120x40).
- F12: "F12 back on" in the footer, then "F12 mouse capture" again.

**Claude Code's mock**, 8 steps at each size: `✗ Bash   go test ./...
failed <time>` with "--- FAIL: TestMarkdownCacheBytes (0.65s)" under it;
the denied update `✗ Bash  git stash && git pull --rebase origin main &&
go test ./...  denied by you <time>` on its row maximised, and at 120x40
`✗ Bash  denied by you <time>` with its three lines under it; no failure
row, no error; `✓ Write  …  allowed by you · <time>`; RESULT "done". ↑
three times to the denied update, Enter: "✗ Bash  denied · 0s", COMMAND
on three rows, APPROVAL (its description, "denied by you", "ran  not
run"), OUTPUT "no output": the CLI's rejection text is not shown as a
result. Esc, ↑, Enter on the failed test: "✗ Bash  failed · 0s", COMMAND,
OUTPUT its three lines, no APPROVAL.

The runs corrected two lines of this file's Human Evaluation draft (the
long command's counter maximised; the denied update on one row
maximised). Zone A, out of the plan's scope, still names Codex's calls
`commandExecution` with their invocation in its workflow tree, as before.

### The files

Against the baseline 4bd297b, the parent of the plan's one commit:

- **New code**: `internal/render/command.go` (the command model: the
  shell, the exact reading of Codex's invocation, the layout, the
  colours and their store), `internal/render/block.go` (zone B's command
  blocks, the decisions on a call's row, the counts' key, the full view's
  command, `y`'s text).
- **Changed code**: `internal/render/` `approval.go` (the card's command
  and context), `cards.go` (a call's row as a block, the decision words),
  `content.go` (the count cache's key, a row's search target),
  `dashboard.go`, `keys.go` (→ ← y), `look.go` (requests and reviews on
  their calls, a call left out while its card shows), `memo.go` (the
  command's facts), `render.go` (`UI.Colors`), `search.go` (the commands'
  targets, `MatchFolds`), `select.go` (a full view's sections copied from
  their source), `viewer.go` (the full view's sections and history);
  `internal/state/` `reduce.go`, `state.go` (S1 to S3); `internal/app/`
  `app.go` (the colours made in a command), `inspect.go` (→ ← y),
  `search.go` (the unfolding).
- **New tests**: `internal/render/command_test.go`,
  `internal/render/command_bench_test.go`,
  `internal/state/associations_test.go`, `internal/app/command_test.go`.
- **Changed tests**: `internal/render/` `approval_test.go`,
  `formatchars_test.go`, `look_test.go`, `outputcut_test.go`,
  `refine_test.go`, `review_test.go`, `search_test.go`,
  `toolfail_test.go`, `viewer_test.go`; `internal/app/` `approval_test.go`,
  `golden_test.go` (the commands scenes), `main_test.go` (the colours made
  outside the tests' bubbles), `mock_test.go`, `scenarios_test.go`,
  `view_test.go`; `internal/provider/mock/files_test.go` (the two mock
  sessions; no provider code).
- **Test data**: 25 new golden frames (`testdata/frames/commands*.txt`),
  15 changed (Details, The golden frames); two new mock sessions,
  `testdata/mock/codex-commands.jsonl` and `claude-commands.jsonl`.
- **Documents**: `AGENTS.md` (the status line), `doc/brief.md`
  (amendment 75, step 1), `doc/ui-design.md`, `doc/architecture.md`,
  `doc/events-and-state.md`, `plan.md` (the plan's closed entry, its section
  moved here, The plan as it ran), this file (new).

## The plan as it ran

plan.md's section of plan 72 as it stood when the owner approved it on
2026-10-31, word for word, its headings one level down: plan.md keeps the
current state only (AGENTS.md, Working rules), and this file the plan's
whole record, as no earlier commit holds it.

### Plan 72 - Tool commands, approval cards and the activity timeline

Brief amendment 75 (2026-10-31): how zone B presents the commands Claude
Code and Codex run, their approval requests and the activity around them.
Phase 1 was a design proposal; after the owner's design review the owner
approved on 2026-10-31 Alternative B with the compact default and direct
mouse expansion, and the six decisions of the revised proposal (Design,
below). The owner authorized this plan for planning and documentation
first, then, after reviewing it, **authorized its implementation on
2026-10-31**, with C1, C2 and C3 decided (Copy and search); the owner's
Human Evaluation follows it, and nothing is committed before the owner's
approval. Below, **approved** marks the owner's decisions, and **to
verify** the implementation details still to be checked.

#### Objectives and scope

- **Readable commands** in zone B: the shell's name in place of
  `commandExecution`; Codex's command taken out of its launcher when that
  is exact; long commands and commands of several operations laid out
  over lines, short ones on one row as today; the original command whole
  in the full view.
- **Approval cards** showing the command laid out and highlighted, up to
  seven lines, and only the context the CLI sends.
- **One entry per operation**: the decision on its call's row, no row of
  the request beside it, no row of the call under its card, Codex's review
  under the call it judged; the whole history in the full view.
- **In scope**: zone B's rows, cards and full views, their contextual key
  hints (the row's and the footer's), plan 42's fold rows for the mouse,
  and the state's associations of requests and reviews with their calls.
  **Out of scope**: file paths (no blocks for them), zones A and C,
  telemetry, the header, the sidebar, the footer's design, the providers,
  their protocols and execution, the model and effort shown elsewhere,
  and plan 71's Markdown rendering, which stays as it is.

#### What was found (2026-10-31, on 4bd297b)

- **Rows.** `callRow` and `toolLine` (`internal/render/cards.go`) draw
  the status, the name, the target in muted text, cut at its right end to
  the room the outcome leaves (`fitDetail`), the outcome and the origin;
  the name column is the run's widest name (`nameWidth`, look.go).
- **Codex's commands** show the item type `commandExecution` and the whole
  invocation, `"C:\WINDOWS\…\powershell.exe" -Command '…'`: at 120x40 no
  part of the command itself shows. Both CLIs report a call before its
  request (P12x and P6b: `item/started` before `requestApproval`; P3b:
  `tool_use` before `can_use_tool`), so a pending call's row repeats its
  card, and an answered request draws a row of its own beside its call's:
  zone B never joins them (the inspection, the tree and the viewer do,
  through `Approval.ToolID`).
- **Cards** (`pendingBox`, approval.go): a one-line command is a target
  row, cut at its end; only a command of several lines gets a preview
  ("▸ Command (N lines)", `$ ` before each line, 6 lines then "… N more
  lines", a grey box from 90 cells); the CLI's summary unlabelled; no
  highlighting.
- **Reviews.** Claude's classifier verdict is on the call's row
  (`ToolCall.Reviewed`, from `tool_result_meta.permission_decision`, an
  undocumented field, said so in the full view); Codex's auto-review is
  an entry of its own (`review`, reduce.go), its event's `targetItemId`
  not kept.
- **Failures**: the error line (`toolError`: Claude's first non-blank
  output line, Codex's `exit N` and last line) after the target, both cut.
- **What can be reused**: plan 42's fold rows (`crow.fold`,
  `ConvView.Folds`, `convClick`, `unfold`) and the folds of plans 19 and
  22 in `ui.Unfolded`; `entryKey`'s → and ←; the search's `MatchFolds`;
  `Highlight` and `SynClass` (syntax.go, palette.go); the Memo and its
  sweep. No mouse path answers a request: the only `answer(...)` in
  mouse.go is Source Control's staging question.
- **Copy today.** A selection in zone B copies the conversation between
  its two ends (`Screen.Text`, `bLines`): messages whole, a long paste's
  folded lines among them, a tool entry as its name and its error line; a
  tool row's target belongs to the search alone, "a kind no selection
  takes and no copy copies" (text.go, plan 66). A full view copies its
  rows as drawn (`fullScreen`, one item a row): a long command's wrapped
  rows copy joined by line breaks, the spaces at the breaks dropped
  (`wrap`), tabs as four spaces: not byte for byte.

#### Design (approved by the owner, 2026-10-31)

- **D1 The shell's name.** Claude: its tool's, `Bash` or `PowerShell`.
  Codex: its invocation's executable when identified: powershell or pwsh
  `PowerShell`, bash, sh and zsh by their names, cmd `cmd`; any other
  `command`. Codex's other item types keep their names.
- **D2 Codex's command, exact or not at all.** A known launcher form
  (PowerShell's `-Command '…'`, `bash -lc '…'`, `cmd /c …`) is undone by
  its shell's own quoting rules, and the result kept only when quoting it
  again gives the invocation back byte for byte; otherwise the invocation
  is shown whole, never an approximation. Claude's command is shown as
  sent. The invocation stays verbatim in the full view.
- **D3 The layout, for display only.** One command a row: a break after
  each top-level `;` (and cmd's `&`) and before each top-level `&&` and
  `||`, that row 2 cells in; a row still too wide breaks before top-level
  `|`, as few times as needed, filled from the left, 2 cells further in; a
  row still too wide wraps at a space outside quotes, a top-level one
  first, 4 cells further in, else at the width. Top level: outside quotes,
  `()`, `{}` and `[]`; the command's own line breaks are kept. The text
  never changes: joined, the rows give the command back, the spaces at
  the breaks aside.
- **D4 The compact default** (decisions 1, 2 and 5). A command whose whole
  text fits the room its row leaves stays on its row, in today's columns.
  Any other goes under its row, at the indent of an unfolded thought's
  details: whole from 1 to 3 lines; from 4 lines, folded by default to its
  first 2 and a row of its own, `… N more lines` (N at least 2: a single
  hidden line is shown instead). Under the command, one row of a
  failure's error or of Codex's review rationale, cut with "…" when
  longer. `▸` before the status marks a block with something hidden, `▾`
  once unfolded, in the run rows' fold-mark blue; the counter grey.
  Unfolded, every line shows, the error and the rationale whole. The
  fold state lasts the session, as a run's.
- **D5 One entry per operation.** A decided request whose call is drawn in
  zone B draws no row of its own: its decision is on the call's row, in
  today's words joined to the call's own outcome by ` · `:
  `allowed by you · 14:06:30` (the call's end),
  `allowed by you · running 4s`, `allowed by you · failed 14:06:30`,
  `allowed for the session by you · …`; with nothing run,
  `denied by you 14:21:05`, and `expired, not answered` in grey. Claude's
  classifier: `approved by auto-review · <end>`,
  `denied by auto-review <time>`. A denied call shows no ✓, no "failed",
  no exit code and no error line, whatever the CLI reports after the
  denial. A pending call draws no row while its card shows. A subagent's
  request (its call is zone C's), a request naming no call drawn in B and
  an unreadable request keep today's rows.
- **D6 Codex's auto-review on its call** (decision 4). A review whose
  `targetItemId` names a call drawn in zone B puts its verdict on that row
  (`approved by auto-review · <end>`, `denied by auto-review <time>`) and
  `◈ risk <level>: <rationale>` under it, its authorization with it when
  the risk is not low (plan 41): one row folded, whole unfolded; no entry
  of its own. Any other review keeps today's `◈` entry.
- **D7 The card**, Template 2 kept. Its heading
  `! APPROVAL · <who> · <shell>`, its time and origin; the command laid
  out (D3) and highlighted (D8), in a grey box of its own from 90 cells as
  today's preview, straight in the card below; whole up to 7 lines, else
  its first 6 and `… N more lines`; no target row and no
  "Command (N lines)" title for a command. Then the CLI's own context,
  each only when sent, under the CLI's own word: Codex's `reason`,
  Claude's `description`, Codex's `cwd` as `in …` only when it is not the
  project folder; Claude's effect sentence for Write and Edit as today.
  The pill, F2, the keys row and a/s/d/c unchanged; a click never folds or
  answers a card.
- **D8 Highlighting.** Chroma's bash lexer for Claude's `Bash`, its
  PowerShell lexer for Claude's `PowerShell` and Codex's powershell and
  pwsh, batch for cmd, none for `command`; the token classes through
  `classOf` to the file viewer's `SynClass` colours, no new colour; in
  cards, blocks and full views. Plain text when the shell is unknown, the
  lexer fails, the tokens do not rejoin into the source byte for byte, or
  on 16 colours as today. Colour never changes a layout.
- **D9 The full view** (Enter, or a double click; decision 6). Its head:
  the symbol, the shell, status · elapsed, the exit code when the CLI
  reports one, the origin. Then COMMAND, laid out for reading; AS SENT,
  verbatim, for a Codex command taken out of its launcher; IN, the cwd
  when sent; APPROVAL when a request or a review came: requested (its
  time, who asks), the reason or the description, the decision (its time
  and source: you once, you for the session, auto-review with its risk and
  authorization, Claude's classifier with today's note of an undocumented
  field, expired), Codex's rationale, ran (from, to; "not run" for a
  denied call); then OUTPUT as today.
- **D10 RESULT unchanged**: the agent's final response, never one per
  command; the activity above it; the outputs in the full view and the
  inspection.

**Reference frames** from the approved mock-ups (2026-10-31), at 120x40,
zone B's rows only; the content is illustrative, every field one the CLIs
send; the notes right of a frame are annotations, not drawn.

```text
╭─ B · CONVERSATION ─────────────────────────────────────────── ⠋ Running ─╮
│ ▎  ✓ PowerShell  git status --short                 14:05:04  observed   │
│ ▎  ✓ PowerShell                                     14:05:09  observed   │
│ ▎     Get-ChildItem -Recurse -Filter *.go -File | Measure-Object         │
│ ▎  ▸ ✓ PowerShell                  allowed by you · 14:06:30  observed   │  ◀ click: unfold
│ ▎     $files = Get-ChildItem -Recurse -Filter *.go -File;                │  ◀ not a click target: a double click opens the full view
│ ▎     $todos = $files | Select-String -Pattern 'TODO|FIXME'              │
│ ▎     … 7 more lines                                                     │  ◀ click: unfold
│ ▎                                                                        │
│ ▎ ◠ Cooking...                                                    01:25  │
╰──────────────────────────────────────────────────────────────────────────╯
```

```text
╭─ B · CONVERSATION ─────────────────────────────────────────── ⠋ Running ─╮
│ ▎  ✓ PowerShell  git status --short                 14:05:04  observed   │
│ ▎  ✓ PowerShell                                     14:05:09  observed   │
│ ▎     Get-ChildItem -Recurse -Filter *.go -File | Measure-Object         │
│ ▎  ▾ ✓ PowerShell                  allowed by you · 14:06:30  observed   │  ◀ click: fold
│ ▎     $files = Get-ChildItem -Recurse -Filter *.go -File;                │  ◀ not a click target: a double click opens the full view
│ ▎     $todos = $files | Select-String -Pattern 'TODO|FIXME'              │
│ ▎       | Sort-Object Path, LineNumber;                                  │
│ ▎     $todos                                                             │
│ ▎       | ForEach-Object                                                 │
│ ▎           { "- $($_.Path):$($_.LineNumber) $($_.Line.Trim())" }        │
│ ▎       | Set-Content -LiteralPath 'TODO.md' -Encoding utf8;             │
│ ▎     '{0} items in {1} files' -f $todos.Count,                          │
│ ▎         ($todos | Select-Object -Unique Path).Count                    │
│ ▎                                                                        │
│ ▎ ◠ Cooking...                                                    01:26  │
╰──────────────────────────────────────────────────────────────────────────╯
```

```text
╭─ B · CONVERSATION ────────────────────────────── ! Waiting for approval ─╮
│ ▎ YOU  14:05                                                             │
│ ▎ Collect the TODO and FIXME comments of the Go sources into TODO.md.    │
│                                                                          │
│ ──────────────────────────────────────────────────────────────────────── │
│                                                                          │
│ ▎ CODEX  gpt-5.6-sol  14:05                                              │
│ ▎ I'll list the Go files, then write their TODO and FIXME comments to    │
│ ▎ TODO.md.                                                               │
│ ▎                                                                        │
│ ▎  ✓ PowerShell  git status --short                 14:05:04  observed   │
│ ▎  ✓ PowerShell                                     14:05:09  observed   │
│ ▎     Get-ChildItem -Recurse -Filter *.go -File | Measure-Object         │
│ ▎                                                                        │
│ ▎ ╭───────────────────────────────────────────────────────────────────╮  │
│ ▎ │ ! APPROVAL · orchestrator · PowerShell            14:05  observed │  │  ◀ no '! … waiting for you' row above it
│ ▎ │   $files = Get-ChildItem -Recurse -Filter *.go -File;             │  │
│ ▎ │   $todos = $files | Select-String -Pattern 'TODO|FIXME'           │  │
│ ▎ │     | Sort-Object Path, LineNumber;                               │  │
│ ▎ │   $todos                                                          │  │
│ ▎ │     | ForEach-Object                                              │  │
│ ▎ │         { "- $($_.Path):$($_.LineNumber) $($_.Line.Trim())" }     │  │
│ ▎ │   … 3 more lines                                                  │  │
│ ▎ │ reason  Create TODO.md from the TODO and FIXME comments of the Go │  │
│ ▎ │         files                                                     │  │
│ ▎ │ ▐ F2 to answer ▌                                                  │  │
│ ▎ ╰───────────────────────────────────────────────────────────────────╯  │
╰──────────────────────────────────────────────────────────────────────────╯
```

```text
╭─ B · CONVERSATION ──────────────────────────────────────────────── Idle ─╮
│ ▎ YOU  14:20                                                             │
│ ▎ Run the tests, and push to origin if they pass.                        │
│                                                                          │
│ ──────────────────────────────────────────────────────────────────────── │
│                                                                          │
│ ▎ CLAUDE  Opus 5.5  14:20                                                │
│ ▎                                                                        │
│ ▎  ✗ Bash  go test ./...                     failed 14:20:41  observed   │
│ ▎     --- FAIL: TestMarkdownCacheBytes (0.65s)                           │
│ ▎                                                                        │
│ ▎ TestMarkdownCacheBytes fails. I will update from origin and run the    │
│ ▎ tests again.                                                           │
│ ▎                                                                        │
│ ▎  ✗ Bash                             denied by you 14:21:05  observed   │
│ ▎     git stash                                                          │
│ ▎       && git pull --rebase origin main                                 │
│ ▎       && go test ./...                                                 │
│                                                                          │
│ ╭──────────────────────────────────────────────────────────────────────╮ │
│ │ RESULT  14:21                                                        │ │
│ │ The tests fail on TestMarkdownCacheBytes, and you denied the update: │ │
│ │ nothing was pushed.                                                  │ │
│ ╰──────────────────────────────────────────────────────────────────────╯ │
╰──────────────────────────────────────────────────────────────────────────╯
```

```text
╭─ B · CONVERSATION ─────────────────────────────────────────── ⠋ Running ─╮
│ ▎ CODEX  gpt-5.6-sol  14:31                                              │
│ ▎                                                                        │
│ ▎  ▸ ✓ PowerShell         approved by auto-review · 14:31:10  observed   │  ◀ click: unfold
│ ▎     Get-ChildItem -Recurse -Filter *.go -File | Measure-Object         │  ◀ not a click target: a double click opens the full view
│ ▎     ◈ risk low: Counts the Go files of the project and reads nothing…  │
╰──────────────────────────────────────────────────────────────────────────╯
```

```text
 ← Esc  ✓ PowerShell  completed · 1.2s · exit 0                                                                observed

 COMMAND, laid out for reading
   $files = Get-ChildItem -Recurse -Filter *.go -File;
   $todos = $files | Select-String -Pattern 'TODO|FIXME' | Sort-Object Path, LineNumber;
   $todos | ForEach-Object { "- $($_.Path):$($_.LineNumber) $($_.Line.Trim())" }
     | Set-Content -LiteralPath 'TODO.md' -Encoding utf8;
   '{0} items in {1} files' -f $todos.Count, ($todos | Select-Object -Unique Path).Count

 AS SENT, verbatim (Codex's command field)
   "C:\WINDOWS\System32\WindowsPowerShell\v1.0\powershell.exe" -Command '$files = Get-ChildItem -Recurse -Filter *.go
   -File; $todos = $files | Select-String -Pattern ''TODO|FIXME'' | Sort-Object Path, LineNumber; $todos |
   ForEach-Object { "- $($_.Path):$($_.LineNumber) $($_.Line.Trim())" } | Set-Content -LiteralPath ''TODO.md''
   -Encoding utf8; ''{0} items in {1} files'' -f $todos.Count, ($todos | Select-Object -Unique Path).Count'

 IN         C:\work\acc-demo

 APPROVAL
   requested  14:05:58  by the orchestrator
   reason     Create TODO.md from the TODO and FIXME comments of the Go files
   decision   14:06:29  allowed by you, once
   ran        14:06:29 to 14:06:30

 OUTPUT
   14 items in 9 files
```

#### Keys and the mouse (approved)

- **K1** ↑ from the empty input brings the entry cursor, one stop per
  operation. On a block, its row's hint, in the outcome's place as on any
  call row, reads `→ unfold · Enter full view · ↑↓ entries · Esc back`
  folded and `← fold · …` unfolded; the footer's contextual hint says the
  same, ending `Esc back to the input · F1 keys`; the footer's design
  unchanged (decision 3).
- **K2** → unfolds; ← folds the block, then, as today, the expanded run it
  is in, else leaves the cursor; Enter opens the full view; Esc returns,
  the fold kept. F2 and the cards' keys unchanged.
- **M1** With capture on, a click on a folded block's header row or its
  `… N more lines` row unfolds it, a click on an unfolded block's header
  folds it: plan 42's fold rows (`crow.fold` on those two rows,
  `ConvView.Folds`, `convClick`, `unfold`), no new mouse code.
- **M2** A double click on any of a block's rows opens its full view, the
  fold its first click toggled put back as it was (plan 42).
- **M3** No navigation first: whatever has the focus, a click in zone B
  gives B the focus, as today, and moves no cursor and shows no hint.
- **M4** No click answers a request or runs anything: a card has no fold
  row; a double click on it opens its full view, as today, where only
  keys answer.
- **M5** With F12's capture off, the frame asks for no mouse reporting:
  the terminal selects, and the keyboard does all of the above.
- **M6** A press on a fold row toggles at once and starts no selection, as
  on every fold row today.

#### Commands: extraction, layout and highlighting

D1, D2, D3 and D8 above, approved. To verify: the launcher forms and
quoting in the recordings, which forms D2 undoes (V1); the tokens of each
lexer through `classOf`, and the rejoin check on every recorded command
(V2); the path that makes the tokens off the draw path, and the lexers'
cost at first use (V3).

#### State and event associations

- **S1 A request and its call.** `Approval.ToolID` (Claude's
  `tool_use_id`, Codex's `itemId`) names `ToolCall.ID`, a call drawn in
  zone B, the orchestrator's; matched whichever arrives first. A request
  naming no call, or a subagent's call, matches nothing (D5).
- **S2 A review and its call.** The review entry keeps its event's target
  item (`targetItemId`, today `ev.Item`, not kept): one field more on the
  entry; attached when that call is drawn in zone B (D6).
- **S3 The classifier's time.** The verdict's event time kept on the call
  with `ToolCall.Reviewed` (decision 6).
- **S4 Nothing else.** The folds in `ui.Unfolded`, under render's
  `CommandKey` as phases are under `ThoughtKey`; the adapters, their
  events and the commands to the CLI unchanged. `doc/events-and-state.md`
  gains S2 and S3 with the code (`TestStateMachines` reads it).

#### Copy and search

- **C1 Zone B's selection keeps its contract** (approved, no change). A
  selection copies the conversation between its two ends: messages whole,
  a long paste's folded lines among them, a tool entry as its name and its
  error line, never the decoration. A command shown in zone B, on its row,
  in a block folded or unfolded, in a card, stays what plan 66 made a tool
  row's target: searched and lit by the search, never taken by a
  selection nor copied; no partial selection of a command in zone B. An
  ordinary selection thus never copies a command line, hidden or shown,
  and what it lights is what it copies. The full view is where a command
  is copied on purpose; messages and the rest copy as before.
- **C2 Exact copy from the full view** (approved and required). Its
  COMMAND section copies the command's own text, its separators and
  whitespace as they are; its AS SENT section copies the invocation byte
  for byte. The layout drawn and the text copied stay apart: a wrap at the
  terminal's width puts no line break in the copy, the layout's indent no
  space, and the command's own spaces and tabs are kept. Only these two
  sections change; every other full view's content copies as before, its
  rows as drawn.
- **C3 A copy by keyboard** (approved and required). `y` in a command's
  full view copies the command as sent, Claude's command or Codex's
  invocation, byte for byte: the text C2's sources hold, through the
  clipboard's usual copy (native and OSC 52), its toast saying it was
  copied or why it failed. Chosen after the existing keys: Ctrl+C copies
  a selection or interrupts the turn, and `c` cancels the turn in an
  approval's full view, so neither can take it; `y` is used nowhere else
  but the Worktrees view's own yes, and it follows the vim-like keys of
  the file viewer and the search (`/`, `n`, `N`, `g`). The full view's
  key hints and the F1 overlay show it; no other view's keys change.
- **C4 The search**, as today (ui-design.md, Search): a call's target as
  zone B writes it before any cut, now the command, taken out of its
  launcher when D2 did so (a Codex launcher's path is then no longer
  matched), every line of a block. A match in a folded block's lines
  unfolds it, as a click does, and it stays so, as a run's or a phase's
  details; a match in a card's lines past its 6 is lit on its
  `… N more lines` row, as a long paste's; a review's rationale is
  searched as its entry is today (to verify, V4), a match in its folded
  part unfolding its block.

#### Performance and cache

- **P1** Rows still drawn only when shown (`crow.fn`), a block's height
  known without drawing it, from the layout's memo.
- **P2** The layout once per command and width, highlighting once per
  lexer and command, the shell and the unwrap once per call: in the Memo,
  bounded, swept with the others (plan 71's generational sweep); a resize
  lays out again at the new width, the old widths swept.
- **P3** No Chroma work on a frame's path: the tokens made off it, as the
  file viewer's are, the command drawn plain until they come, in the same
  layout (V3).
- **P4** Plan 71's Markdown rendering, `WrapCache`, their tests and their
  benchmarks untouched: commands are not Markdown.
- **P5** Budgets: the keystroke timing step (doc/evaluation.md) within
  its budgets; a frame benchmark of 200 commands, half of them long,
  folded and unfolded, at 120x40 and 200x60; plan 71's allocation guard,
  outside `-race`; the search's cost per target as today.

#### To verify during the implementation

Implementation details, not design decisions, each reported with its
evidence in `doc/plan72-evaluation.md`:
- **V1** the launcher forms and their quoting in the recordings (P6b, P7,
  P8's request, P12x), and which forms D2 undoes;
- **V2** Chroma's PowerShell, bash and batch tokens through `classOf`,
  and the rejoin check on every recorded command;
- **V3** the highlighting path off the draw path, and each lexer's cost
  at first use;
- **V4** the order of Codex's review events, `targetItemId` always sent,
  the in-progress words, the review's text in the search;
- **V5** what Claude reports after a denial, for D5's denial to win;
- **V6** each width's rows with the entry cursor's hint, B's height
  unchanged by the cursor;
- **V7** the golden frames that change, reviewed by script as plan 71's;
- **V8** that zone B draws the orchestrator's calls only.

#### Risks and fallbacks

- An unwrap that is not exact: the invocation whole (D2).
- Tokens that do not rejoin, an unknown shell, a lexer failure: plain
  text (D8).
- A request or a review not matched with certainty: today's rows and
  entries (D5, D6); a pending request's card always shows.
- An unusual command (a very long token, a here-string, a script of many
  lines): wrapped at the width, its text unchanged (D3).
- Many golden frames changed by the new heights: each reviewed as a
  diff, by script.
- The mouse out of reach (F12 off, a layer on top, a docked panel in B's
  place): the keyboard (K1, K2).
- A long session: memos and lazy rows (P1, P2), measured (P5).
- A Codex launcher's path no longer found by the search (C4): said so in
  ui-design.md.
- Claude's undocumented `permission_decision`: labelled so, as today (D9).

#### Criteria

- **72.1 The shell and the command, both CLIs.** Claude's `Bash` and
  `PowerShell` calls show their tool's name and the command as sent;
  Codex's show PowerShell, bash, sh, zsh or cmd, else `command`, and the
  command out of a known launcher form; P8's card reads
  `! APPROVAL · orchestrator · PowerShell` and
  `Add-Content -Path note.txt -Value one`.
- **72.2 Exact or whole.** An unwrap is kept only when quoting it again
  gives the invocation byte for byte; any other invocation shows whole;
  tested on every recorded invocation and on malformed ones.
- **72.3 The layout.** A block's lines, joined, give the command back, the
  spaces at the breaks aside; breaks only at top-level `;`, `&&`, `||`,
  `|`, cmd's `&` and spaces outside quotes; no line wider than its zone at
  74, 78, 134 and 140 cells.
- **72.4 Short and long.** A command that fits its row takes one row; any
  other goes under it: 1 to 3 lines whole, 4 or more its first 2 and
  `… N more lines` (N = its lines − 2); one row of error or rationale cut
  with "…"; `▸` on a folded block, `▾` on an unfolded one, none on the
  others.
- **72.5 Folded and unfolded.** Unfolded: every line, the error and the
  rationale whole; folded again as before; the state kept for the session,
  through scrolling, resizing and the full view; only the block's own
  rows change B's height.
- **72.6 The keyboard.** On a block → unfolds, ← folds it (then its
  expanded run), Enter opens the full view, Esc returns; the row's hint
  and the footer's name → and ←; ↑↓ stop once per operation.
- **72.7 Clicks and double clicks.** A click on a folded block's header or
  counter row unfolds it, on an unfolded block's header folds it, with the
  focus anywhere before; a double click on any of its rows opens its full
  view, the fold as it was; a press on a command line toggles nothing; no
  click moves the entry cursor or answers a request.
- **72.8 F12.** With capture off the frame asks for no mouse reporting,
  and the keyboard does all of 72.6.
- **72.9 The card.** The command laid out and highlighted, whole up to 7
  lines, else 6 and `… N more lines`; no target row; reason, description
  and cwd only when sent, cwd only outside the project; the pill, F2,
  a/s/d/c, Enter and Esc unchanged; no row of its call while it shows.
- **72.10 Decisions and outcomes.** Completed, running, failed, denied,
  expired, allowed for the session, Claude's classifier and Codex's
  auto-review, each on its call's row in D5's words, a denial never shown
  as a failure; a request whose call is drawn in B has no row of its own;
  a subagent's request, one naming no call and an unreadable one keep
  today's rows.
- **72.11 The approval's history.** The full view shows D9's fields for
  each case of 72.10, a denied call "not run"; the inspection unchanged.
- **72.12 Codex's review.** A review naming a call drawn in B: its verdict
  on that row and `◈ risk …: <rationale>` under it, one row folded, whole
  unfolded, no entry of its own; any other review: today's entry.
- **72.13 Copy.** A selection across blocks, folded or not, and across
  cards copies what it copies today, no command line in it (C1); the full
  view's COMMAND copies the command's text and AS SENT the invocation byte
  for byte, a wrapped line with no line break added, the indent with no
  space, tabs and spaces kept, compared with the recording; the other
  full views copy as before (C2); `y` copies the command as sent, its
  toast on success and on failure (C3).
- **72.14 The search.** C4: the commands found, a folded block unfolded
  by its match, a card's hidden lines lit on its counter row.
- **72.15 Highlighting.** Bash, PowerShell and cmd commands coloured in
  cards, blocks and full views with `SynClass`'s colours; an unknown
  shell, a lexer failure, a failed rejoin check, 16 colours: plain; the
  text drawn the same with and without colour, and the layout.
- **72.16 Widths and golden frames.** At 80x24, 120x40, 200x60 and
  208x59: nothing cut but by the rules shown, the decision words whole at
  74 cells for the orchestrator's operations; each changed golden frame
  reviewed as a diff, nothing changed but what this plan changes.
- **72.17 Both CLIs, the scope kept.** P3b, P6b, P7, P8's request and
  P12x replayed to the expected frames; `internal/provider` unchanged;
  zones A and C, telemetry, the header and the sidebar unchanged in every
  golden frame, the footer in its contextual hints only.
- **72.18 Performance.** The layout and the highlighting once per command
  (the memos' hits); rows drawn only when shown; the 200-command benchmark
  and the timing step within their budgets; the allocation guard holding
  outside `-race`.
- **72.19 Plan 71 kept.** Its Markdown tests and benchmarks unchanged and
  passing.
- **72.20 The checks.** Build, vet, gofmt, the whole suite after the last
  edit, mutants through `-overlay`, `-race` on the changed packages, the
  timing step alone.
- **72.21 The documents.** ui-design.md (Zone B's tool rows and blocks,
  Expanding, the mouse; Approvals; Search; Selection and copy; the key
  tables), architecture.md, events-and-state.md, this plan,
  `doc/plan72-evaluation.md`.
- **72.22 Windows Terminal and the owner's Human Evaluation**, below.

#### Tests

The project's own means, no new framework: unit tests in render (the
command model, layouts, blocks, cards, full views, highlighting), state
(the associations) and app (the keys, clicks through `convClick` on a
drawn screen, the search, the copy); golden frames (`TestGoldenFrames`),
new scenes built from synthetic events in `golden_test.go` and labelled
so, as the review's scene is; the recordings of both CLIs replayed; the
timing step and the benchmarks; mutants through `-overlay`; `-race` on
the changed packages; architecture test 7.

#### Manual Human Evaluation

In Windows Terminal, maximised and at 120x40 (acc's tab, about 208x59),
on mock sessions with the design's scenes, which use no quota; then a
short live session per CLI, only within a scope the owner approves
(doc/evaluation.md, the live runs): the six scenarios; a click on a
header and on a counter row, a double click, a click with the focus in
zone A, F12 off and on; the keyboard; F2 and a card's keys; a search
reaching into a folded block; the full view's AS SENT copied, pasted and
compared byte for byte with the recording; 80x24 and 200x60. The owner
records the result; the plan closes only on the owner's approval.

#### Components

- `internal/render/command.go`, new: `shellOf`, `unwrap`, the layout and
  `CommandKey` (D1 to D3).
- `internal/render/cards.go`: `callRow` as a block, its counter row and
  its error or rationale row, `crow.fold` on its header and counter rows
  (D4 to D6, M1).
- `internal/render/look.go`: requests and reviews joined to their calls, a
  pending call's row left out while its card shows (D5, D6, S1, S2).
- `internal/render/approval.go`: the card's command and context (D7).
- `internal/render/viewer.go`: the full view's sections and their copy
  (D9, C2), a block's hints (K1).
- `internal/render/syntax.go`: `Highlight` by lexer name (D8);
  `internal/render/search.go`: `MatchFolds` for blocks (C4).
- `internal/app/inspect.go`: `entryKey`'s → and ← on a block (K2);
  `internal/app/search.go`: the unfolding (C4); `internal/app/mouse.go`
  unchanged (M1 to M6).
- `internal/state`: the review's target and the classifier's time (S2,
  S3).
- Unchanged: `internal/provider`, the events the adapters send, the
  commands written to the CLIs.

#### Steps

1. This plan, amendment 75, AGENTS.md's status. **Gate**: the owner's
   review of this plan and authorization to implement, given on
   2026-10-31.
2. The command model (D1 to D3) and its tests on the recordings.
3. The state's associations (S1 to S3), `events-and-state.md` with them,
   their tests.
4. Zone B: the blocks (D4), one entry per operation (D5), the reviews
   (D6), the card (D7), the highlighting (D8), their tests.
5. The full view (D9) and its copy (C2, C3 as decided), the keys and the
   mouse (K1 to M6), the search (C4), their tests.
6. The documents (72.21).
7. The checks (72.16 to 72.20): the suite, the golden frames by script,
   the mutants, `-race`, the timing step, the benchmarks.
8. Windows Terminal; `doc/plan72-evaluation.md`.
9. The owner's Human Evaluation and review; the commits only on approval.

#### Progress

- 2026-10-31: the design: phase 1's proposal, the owner's review, the
  revision; the owner approved Alternative B with the compact default and
  direct mouse expansion, and the six decisions (amendment 75).
- 2026-10-31: step 1: this plan, amendment 75 and AGENTS.md's status,
  nothing else.
- 2026-10-31: the owner authorized the implementation ("I explicitly
  authorize you to implement Plan 72."), C1 kept, C2 and C3 approved and
  required; `y` chosen for C3 after the existing keys; no commit before
  the owner's review.
- 2026-10-31: steps 2 to 7 built and checked, nothing committed: the
  command model (`internal/render/command.go`), zone B's blocks and
  decisions (`block.go`, `cards.go`, `look.go`), the card
  (`approval.go`), the full view's sections and their copy (`viewer.go`,
  `select.go`), the search (`search.go`), the keys and the colours
  (`internal/app`), the state's associations (`internal/state`); the
  documents (ui-design.md, architecture.md, events-and-state.md); two mock
  sessions for the Human Evaluation (`testdata/mock/codex-commands.jsonl`,
  `claude-commands.jsonl`); results and evidence in
  `doc/plan72-evaluation.md`. As built, beside the design's words:
  D2's exactness is the invocation read back word for word by POSIX's
  rules, nothing left over and nothing a shell would expand, not a
  quoting again, which cannot give Codex's own chunks back (V1); a
  request or a review drawn on its call's row still ends a run of calls,
  as its row did, so that runs fold as before; questions keep their
  request rows, their answers under them; a Codex review is joined to a
  command's call only; the history's label column is 15 cells, for
  "description".
- 2026-10-31: steps 7 and 8 finished, nothing committed or pushed: the
  mutants' last passes (64 in all: 62 caught, 30 and 31 equivalent),
  which made tests stronger and led to one fix, the full view's "ran"
  started neither at Claude Code's classifier verdict, which comes with
  the result, nor while Codex's review is in progress
  (`TestCommandRanTimes`); `TestCommandRowsShown` (a frame draws the rows
  it shows; its growth with a session's calls is HEAD's, zone A's tree and
  the runs); `-race`, the whole suite after the last edit and the timing
  step alone (21 of 21); Windows Terminal on both mocks, maximised and at
  120x40 (72 steps; `y` and double clicks left to the owner, the
  antivirus having blocked a harness that saved the clipboard);
  `doc/plan72-evaluation.md`: 21 PASS, 72.22 waiting for the owner's
  Human Evaluation. Step 9 is the owner's.
- 2026-10-31: the owner's Human Evaluation in Windows Terminal, approved
  ("I APPROVE the visual design, usability, and interactions."): 72.22
  PASS on that approval, plan 72 at 22 PASS, 0 FAIL
  (`doc/plan72-evaluation.md`, The owner's Human Evaluation); no check
  run again. The owner's independent code review comes next, on a review
  package prepared outside the repository; nothing committed or pushed,
  the plan open.
